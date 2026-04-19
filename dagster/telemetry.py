import json
import os
import threading
import time
from errno import EADDRINUSE
from contextlib import contextmanager

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from prometheus_client import Counter, Histogram, start_http_server


SERVICE_NAME = os.getenv("OTEL_SERVICE_NAME", "dagster")

_setup_lock = threading.Lock()
_telemetry_ready = False

OP_RUNS_TOTAL = Counter(
    "dagster_op_runs_total",
    "Total number of Dagster op executions.",
    ["service", "op_name", "status"],
)
OP_DURATION = Histogram(
    "dagster_op_duration_seconds",
    "Latency of Dagster op executions.",
    ["service", "op_name"],
)
OP_ROWS_TOTAL = Counter(
    "dagster_op_rows_total",
    "Rows processed by Dagster ops.",
    ["service", "dataset"],
)
DBT_RUNS_TOTAL = Counter(
    "dagster_dbt_runs_total",
    "Total number of dbt executions triggered by Dagster.",
    ["service", "status"],
)


def setup_telemetry() -> None:
    global _telemetry_ready

    with _setup_lock:
        if _telemetry_ready:
            return

        metrics_port = os.getenv("DAGSTER_METRICS_PORT", "").strip()
        if metrics_port:
            try:
                start_http_server(int(metrics_port))
            except OSError as exc:
                if exc.errno != EADDRINUSE:
                    raise

        endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "").rstrip("/")
        if endpoint:
            provider = TracerProvider(
                resource=Resource.create(
                    {
                        "service.name": SERVICE_NAME,
                        "deployment.environment": os.getenv("PLATFORM_ENVIRONMENT", "local"),
                    }
                )
            )
            provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
            trace.set_tracer_provider(provider)

        _telemetry_ready = True


@contextmanager
def instrumented_op(context, op_name: str):
    setup_telemetry()
    tracer = trace.get_tracer(SERVICE_NAME)
    started_at = time.perf_counter()
    status = "success"

    with tracer.start_as_current_span(op_name) as span:
        span.set_attribute("dagster.run_id", context.run_id)
        span.set_attribute("dagster.op_name", op_name)
        job_name = getattr(context, "job_name", "")
        if job_name:
            span.set_attribute("dagster.job_name", job_name)

        try:
            yield span
        except Exception as exc:
            status = "error"
            span.record_exception(exc)
            span.set_attribute("dagster.status", status)
            raise
        finally:
            duration = time.perf_counter() - started_at
            OP_RUNS_TOTAL.labels(SERVICE_NAME, op_name, status).inc()
            OP_DURATION.labels(SERVICE_NAME, op_name).observe(duration)
            log_event(
                context,
                "dagster_op_completed",
                op_name=op_name,
                status=status,
                duration_ms=round(duration * 1000, 2),
            )


def record_rows(dataset: str, row_count: int) -> None:
    if row_count > 0:
        OP_ROWS_TOTAL.labels(SERVICE_NAME, dataset).inc(row_count)


def record_dbt_run(status: str) -> None:
    DBT_RUNS_TOTAL.labels(SERVICE_NAME, status).inc()


def log_event(context, event: str, **fields) -> None:
    payload = {
        "event": event,
        "service": SERVICE_NAME,
        "run_id": getattr(context, "run_id", ""),
    }
    payload.update(fields)
    context.log.info(json.dumps(payload))