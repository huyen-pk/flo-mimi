from datetime import datetime, timezone
from uuid import uuid4

from fastapi import FastAPI


app = FastAPI(title="Mock Third-Party API")


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.get("/partners/snapshot")
def partner_snapshot() -> list[dict]:
    collected_at = datetime.now(timezone.utc).isoformat()
    return [
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