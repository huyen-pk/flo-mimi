import json
import os
import uuid
import time
from datetime import datetime, timezone

from confluent_kafka import Producer
from fastapi import FastAPI, Request
from pydantic import BaseModel, Field

from app.observability import configure_app, current_correlation_id, observe_kafka_publish


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


class EmailEvent(BaseModel):
    event_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    campaign_id: str
    recipient_id: str | None = None
    event_type: str
    occurred_at: datetime = Field(default_factory=utc_now)
    payload: dict = Field(default_factory=dict)


class AnalyticsEvent(BaseModel):
    event_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    session_id: str | None = None
    user_id: str | None = None
    event_name: str
    page_url: str | None = None
    occurred_at: datetime = Field(default_factory=utc_now)
    payload: dict = Field(default_factory=dict)


app = FastAPI(title="Event Gateway")
logger = configure_app(app, os.getenv("OTEL_SERVICE_NAME", "event-gateway"))
producer = Producer({"bootstrap.servers": os.environ["REDPANDA_BROKERS"]})


def publish(topic: str, payload: dict, key: str | None = None, correlation_id: str | None = None) -> None:
    started_at = time.perf_counter()
    status = "success"
    try:
        producer.produce(
            topic=topic,
            key=key,
            value=json.dumps(payload).encode("utf-8"),
            headers=[("x-correlation-id", correlation_id or current_correlation_id())],
        )
        producer.flush(5)
    except Exception:
        status = "error"
        raise
    finally:
        observe_kafka_publish(os.getenv("OTEL_SERVICE_NAME", "event-gateway"), topic, status, time.perf_counter() - started_at)


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/events/email")
def ingest_email_event(event: EmailEvent, request: Request) -> dict:
    payload = event.model_dump(mode="json")
    payload["correlation_id"] = request.state.correlation_id
    publish("email_events_raw", payload, key=event.campaign_id, correlation_id=request.state.correlation_id)
    logger.info(json.dumps({"event": "email_event_accepted", "campaign_id": event.campaign_id, "event_id": event.event_id, "correlation_id": request.state.correlation_id}))
    return {"status": "accepted", "event_id": event.event_id}


@app.post("/events/analytics")
def ingest_analytics_event(event: AnalyticsEvent, request: Request) -> dict:
    payload = event.model_dump(mode="json")
    payload["correlation_id"] = request.state.correlation_id
    publish("analytics_events_raw", payload, key=event.user_id, correlation_id=request.state.correlation_id)
    logger.info(json.dumps({"event": "analytics_event_accepted", "event_name": event.event_name, "event_id": event.event_id, "correlation_id": request.state.correlation_id}))
    return {"status": "accepted", "event_id": event.event_id}