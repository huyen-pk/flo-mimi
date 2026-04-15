import contextvars
import json
import logging
import os
import time
from uuid import uuid4

from fastapi import FastAPI, Request
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from prometheus_client import Counter, Gauge, Histogram, make_asgi_app


CORRELATION_HEADER = "X-Correlation-ID"

_correlation_id_ctx: contextvars.ContextVar[str] = contextvars.ContextVar("correlation_id", default="")

HTTP_REQUESTS_TOTAL = Counter(
    "app_http_requests_total",
    "Total number of HTTP requests handled by the application.",
    ["service", "method", "route", "status"],
)
HTTP_REQUEST_DURATION = Histogram(
    "app_http_request_duration_seconds",
    "Latency of HTTP requests handled by the application.",
    ["service", "method", "route"],
)
HTTP_INFLIGHT = Gauge(
    "app_http_inflight_requests",
    "Number of in-flight HTTP requests handled by the application.",
    ["service"],
)
KAFKA_PUBLISH_TOTAL = Counter(
    "app_kafka_publish_total",
    "Total number of Kafka publish attempts.",
    ["service", "topic", "status"],
)
KAFKA_PUBLISH_DURATION = Histogram(
    "app_kafka_publish_duration_seconds",
    "Latency of Kafka publish attempts.",
    ["service", "topic"],
)


def configure_app(app: FastAPI, service_name: str) -> logging.Logger:
    logger = _configure_logging(service_name)
    _configure_tracing(service_name)
    FastAPIInstrumentor.instrument_app(app, excluded_urls="/metrics")
    app.mount("/metrics", make_asgi_app())

    @app.middleware("http")
    async def observability_middleware(request: Request, call_next):
        started_at = time.perf_counter()
        correlation_id = request.headers.get(CORRELATION_HEADER) or f"corr-{uuid4().hex[:12]}"
        token = _correlation_id_ctx.set(correlation_id)
        request.state.correlation_id = correlation_id
        HTTP_INFLIGHT.labels(service=service_name).inc()

        route = request.url.path
        status_code = 500
        try:
            response = await call_next(request)
            route = getattr(request.scope.get("route"), "path", request.url.path)
            status_code = response.status_code
            response.headers[CORRELATION_HEADER] = correlation_id
            return response
        finally:
            duration = time.perf_counter() - started_at
            HTTP_REQUESTS_TOTAL.labels(service_name, request.method, route, str(status_code)).inc()
            HTTP_REQUEST_DURATION.labels(service_name, request.method, route).observe(duration)
            HTTP_INFLIGHT.labels(service=service_name).dec()
            logger.info(
                json.dumps(
                    {
                        "event": "http_request",
                        "service": service_name,
                        "method": request.method,
                        "path": request.url.path,
                        "route": route,
                        "status": status_code,
                        "duration_ms": round(duration * 1000, 2),
                        "correlation_id": correlation_id,
                        "trace_id": current_trace_id(),
                    }
                )
            )
            _correlation_id_ctx.reset(token)

    return logger


def observe_kafka_publish(service_name: str, topic: str, status: str, duration_seconds: float) -> None:
    KAFKA_PUBLISH_TOTAL.labels(service_name, topic, status).inc()
    KAFKA_PUBLISH_DURATION.labels(service_name, topic).observe(duration_seconds)


def current_correlation_id() -> str:
    return _correlation_id_ctx.get()


def current_trace_id() -> str:
    span_context = trace.get_current_span().get_span_context()
    if not span_context or not span_context.trace_id:
        return ""
    return format(span_context.trace_id, "032x")


def _configure_logging(service_name: str) -> logging.Logger:
    logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), format="%(message)s")
    logging.getLogger("uvicorn.access").disabled = True
    logger = logging.getLogger(service_name)
    logger.setLevel(os.getenv("LOG_LEVEL", "INFO"))
    return logger


def _configure_tracing(service_name: str) -> None:
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "").rstrip("/")
    if not endpoint:
        return

    provider = TracerProvider(
        resource=Resource.create(
            {
                "service.name": service_name,
                "deployment.environment": os.getenv("PLATFORM_ENVIRONMENT", "local"),
            }
        )
    )
    provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
    trace.set_tracer_provider(provider)