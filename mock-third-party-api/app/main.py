from datetime import datetime, timezone
from uuid import uuid4

from fastapi import FastAPI

from app.observability import configure_app


app = FastAPI(title="Mock Third-Party API")
logger = configure_app(app, "mock-third-party-api")


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.get("/partners/snapshot")
def partner_snapshot() -> list[dict]:
    collected_at = datetime.now(timezone.utc).isoformat()
    payload = [
        {
            "record_id": str(uuid4()),
            "provider": "ads_partner",
            "account_id": "acct-001",
            "collected_at": collected_at,
            "payload": {"spend": 1200.45, "impressions": 88000, "clicks": 1290},
        },
        {
            "record_id": str(uuid4()),
            "provider": "crm_partner",
            "account_id": "acct-002",
            "collected_at": collected_at,
            "payload": {"qualified_leads": 28, "pipeline_value": 93450},
        },
    ]
    logger.info('{"event": "partner_snapshot_generated", "records": 2}')
    return payload