package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	analyticsadapter "platform/internal/adapters/analytics"
	contentadapter "platform/internal/adapters/content"
	gatewayadapter "platform/internal/adapters/gateway"
	bootstrapapp "platform/internal/application/bootstrap"
	interactionapp "platform/internal/application/interactions"
	httpapp "platform/internal/http"
	"platform/internal/telemetry"
	webassets "platform/web"
)

func main() {
	logger := telemetry.NewLogger(envOrDefault("OTEL_SERVICE_NAME", "platform"))
	slog.SetDefault(logger)
	log.SetFlags(0)

	shutdownTelemetry, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "platform"), envOrDefault("PLATFORM_ENVIRONMENT", "local"))
	if err != nil {
		logger.Error("configure telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			logger.Error("shutdown telemetry", "error", err)
		}
	}()

	bootstrapRepository, err := contentadapter.NewAppDBRepository(
		envOrDefault("APPDB_DSN", "postgresql://analytics:analytics@localhost:5432/analytics?sslmode=disable"),
		envOrDefault("PLATFORM_ENVIRONMENT", "local"),
	)
	if err != nil {
		logger.Error("create bootstrap repository", "error", err)
		os.Exit(1)
	}
	defer bootstrapRepository.Close()

	httpClient := &http.Client{
		Timeout: 8 * time.Second,
	}
	eventGateway := gatewayadapter.NewHTTPEventGateway(envOrDefault("EVENT_GATEWAY_BASE_URL", "http://localhost:8000"), httpClient)

	analyticsRepo, err := analyticsadapter.NewClickHouseRepository(envOrDefault("CLICKHOUSE_HOST", "clickhouse"), envOrDefault("CLICKHOUSE_PORT", "8123"))
	if err != nil {
		logger.Error("create analytics repository", "error", err)
		os.Exit(1)
	}

	bootstrapService := bootstrapapp.NewService(bootstrapRepository, analyticsRepo)
	interactionService := interactionapp.NewService(eventGateway, bootstrapRepository)
	applicationHandler := httpapp.NewHandler(bootstrapService, interactionService, webassets.Dist())
	rootHandler := http.NewServeMux()
	rootHandler.Handle("/metrics", telemetry.MetricsHandler())
	rootHandler.Handle("/", telemetry.TraceHTTP("platform", telemetry.WrapHTTP("platform", logger, applicationHandler)))

	server := &http.Server{
		Addr:              ":" + envOrDefault("PORT", "8081"),
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	logger.Info("platform app listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("serve http", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
