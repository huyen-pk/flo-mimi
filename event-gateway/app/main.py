import json
import os
import uuid
from datetime import datetime, timezone

from confluent_kafka import Producer
from fastapi import FastAPI
from pydantic import BaseModel, Field


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
producer = Producer({"bootstrap.servers": os.environ["REDPANDA_BROKERS"]})


def publish(topic: str, payload: dict, key: str | None = None) -> None:
    producer.produce(topic=topic, key=key, value=json.dumps(payload).encode("utf-8"))
    producer.flush(5)


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/events/email")
def ingest_email_event(event: EmailEvent) -> dict:
    payload = event.model_dump(mode="json")
    publish("email_events_raw", payload, key=event.campaign_id)
    return {"status": "accepted", "event_id": event.event_id}


@app.post("/events/analytics")
def ingest_analytics_event(event: AnalyticsEvent) -> dict:
    payload = event.model_dump(mode="json")
    publish("analytics_events_raw", payload, key=event.user_id)
    return {"status": "accepted", "event_id": event.event_id}