package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const CorrelationHeader = "X-Correlation-ID"

type correlationIDKey struct{}

var (
	registerMetrics sync.Once

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_http_requests_total",
			Help: "Total number of HTTP requests handled by the application.",
		},
		[]string{"service", "method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_http_request_duration_seconds",
			Help:    "Latency of HTTP requests handled by the application.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route"},
	)
	httpInflightRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_http_inflight_requests",
			Help: "Number of in-flight HTTP requests handled by the application.",
		},
		[]string{"service"},
	)
	outboundHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_outbound_http_requests_total",
			Help: "Total number of outbound HTTP requests sent by the application.",
		},
		[]string{"service", "target", "method", "route", "status"},
	)
	outboundHTTPDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_outbound_http_request_duration_seconds",
			Help:    "Latency of outbound HTTP requests sent by the application.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "target", "method", "route"},
	)
)

func init() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(
			httpRequestsTotal,
			httpRequestDuration,
			httpInflightRequests,
			outboundHTTPRequestsTotal,
			outboundHTTPDuration,
		)
	})
}

func NewLogger(service string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("service", service)
}

func Setup(ctx context.Context, serviceName string, environment string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	options := make([]otlptracehttp.Option, 0, 4)
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		options = append(options, otlptracehttp.WithEndpoint(parsed.Host))
		tracePath := strings.TrimSpace(parsed.Path)
		if tracePath == "" || tracePath == "/" {
			tracePath = "/v1/traces"
		} else if !strings.HasSuffix(tracePath, "/v1/traces") {
			tracePath = path.Join(tracePath, "/v1/traces")
		}
		options = append(options, otlptracehttp.WithURLPath(tracePath))
		if parsed.Scheme != "https" {
			options = append(options, otlptracehttp.WithInsecure())
		}
	} else {
		options = append(options, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		attribute.String("deployment.environment", environment),
	))
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func TraceHTTP(service string, next http.Handler) http.Handler {
	tracer := otel.Tracer(service)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(request.Context(), propagation.HeaderCarrier(request.Header))
		route := normalizeRoute(request.URL.Path)
		ctx, span := tracer.Start(ctx, request.Method+" "+route)
		defer span.End()

		statusWriter := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(statusWriter, request.WithContext(ctx))

		span.SetAttributes(
			attribute.String("http.method", request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", statusWriter.status),
		)
		if statusWriter.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(statusWriter.status))
		}
	})
}

func WrapHTTP(service string, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		correlationID := firstNonEmpty(strings.TrimSpace(request.Header.Get(CorrelationHeader)), generateID("corr"))
		ctx := context.WithValue(request.Context(), correlationIDKey{}, correlationID)
		request = request.WithContext(ctx)

		statusWriter := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		statusWriter.Header().Set(CorrelationHeader, correlationID)

		httpInflightRequests.WithLabelValues(service).Inc()
		defer httpInflightRequests.WithLabelValues(service).Dec()

		next.ServeHTTP(statusWriter, request)

		route := normalizeRoute(request.URL.Path)
		status := strconv.Itoa(statusWriter.status)
		duration := time.Since(startedAt)

		httpRequestsTotal.WithLabelValues(service, request.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(service, request.Method, route).Observe(duration.Seconds())

		fields := []any{
			"method", request.Method,
			"path", request.URL.Path,
			"status", statusWriter.status,
			"duration_ms", duration.Milliseconds(),
			"correlation_id", correlationID,
		}
		if spanContext := trace.SpanContextFromContext(request.Context()); spanContext.IsValid() {
			fields = append(fields, "trace_id", spanContext.TraceID().String())
		}
		logger.Info("http request", fields...)
	})
}

func ObserveOutboundHTTP(service string, target string, method string, route string, status string, duration time.Duration) {
	route = normalizeRoute(route)
	status = firstNonEmpty(strings.TrimSpace(status), "error")
	outboundHTTPRequestsTotal.WithLabelValues(service, target, method, route, status).Inc()
	outboundHTTPDuration.WithLabelValues(service, target, method, route).Observe(duration.Seconds())
}

func InjectHeaders(ctx context.Context, header http.Header) {
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		header.Set(CorrelationHeader, correlationID)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

func CorrelationIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey{}).(string)
	return value
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func normalizeRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "/"
	}
	return route
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func generateID(prefix string) string {
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		return prefix + "-fallback"
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}
