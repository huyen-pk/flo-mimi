import json
import os
import subprocess
import hashlib
from pathlib import Path
from datetime import datetime, timezone

import boto3
import psycopg
import requests
from dagster import Definitions, In, Nothing, OpExecutionContext, Out, ScheduleDefinition, job, op
from psycopg.types.json import Jsonb

from telemetry import instrumented_op, log_event, record_dbt_run, record_rows


class TrinoError(RuntimeError):
    pass


def trino_base_url() -> str:
    return f"http://{os.environ['TRINO_HOST']}:{os.environ['TRINO_PORT']}"


def trino_headers() -> dict[str, str]:
    return {
        "X-Trino-User": os.environ.get("TRINO_USER", "dagster"),
        "X-Trino-Source": "dagster",
    }


def trino_execute(sql: str) -> list[list]:
    response = requests.post(
        f"{trino_base_url()}/v1/statement",
        data=sql.encode("utf-8"),
        headers=trino_headers(),
        timeout=30,
    )
    response.raise_for_status()
    payload = response.json()
    rows: list[list] = []
    while True:
        if payload.get("error"):
            raise TrinoError(payload["error"].get("message", "unknown Trino error"))
        if payload.get("data"):
            rows.extend(payload["data"])
        next_uri = payload.get("nextUri")
        if not next_uri:
            return rows
        response = requests.get(next_uri, headers=trino_headers(), timeout=30)
        response.raise_for_status()
        payload = response.json()


def trino_execute_file(path: Path) -> None:
    statements = [statement.strip() for statement in path.read_text().split(";") if statement.strip()]
    for statement in statements:
        trino_execute(statement)


def sql_string(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def sql_nullable_string(value: str | None) -> str:
    if value is None or value == "":
        return "NULL"
    return sql_string(value)


def sql_timestamp(value: str | datetime) -> str:
    if isinstance(value, str):
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    else:
        parsed = value
    utc_value = parsed.astimezone(timezone.utc).replace(tzinfo=None)
    return f"TIMESTAMP '{utc_value.strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]}'"


def sql_date(value: datetime) -> str:
    return f"DATE '{value.astimezone(timezone.utc).strftime('%Y-%m-%d')}'"


def sql_nullable_int(value: int | None) -> str:
    if value is None:
        return "NULL"
    return str(int(value))


def appdb_connection():
    return psycopg.connect(os.environ["APPDB_DSN"])


def minio_client():
    return boto3.client(
        "s3",
        endpoint_url=os.environ["MINIO_ENDPOINT_URL"],
        aws_access_key_id=os.environ["MINIO_ROOT_USER"],
        aws_secret_access_key=os.environ["MINIO_ROOT_PASSWORD"],
        region_name=os.environ.get("MINIO_REGION", "us-east-1"),
    )


def bronze_object_topic(object_key: str) -> str:
    parts = object_key.split("/")
    if len(parts) > 1 and parts[0] == "bronze":
        return parts[1]
    return "unknown"


def chunked(values: list[dict], size: int) -> list[list[dict]]:
    return [values[index : index + size] for index in range(0, len(values), size)]


def parse_event_timestamp(value: str | datetime | None) -> datetime:
    if isinstance(value, datetime):
        return value
    if isinstance(value, str) and value:
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00"))
        except ValueError:
            pass
    return datetime.now(timezone.utc)


def canonical_payload(payload: dict) -> str:
    return json.dumps(payload, separators=(",", ":"), sort_keys=True)


def raw_event_record_value(record: dict) -> tuple[str, str]:
    payload_text = canonical_payload(record)
    event_id = record.get("event_id") or hashlib.md5(payload_text.encode("utf-8")).hexdigest()
    event_name = record.get("event_name") or record.get("event_type")
    occurred_at = parse_event_timestamp(record.get("occurred_at"))
    value = (
        "("
        f"{sql_string(event_id)}, {sql_nullable_string(event_name)}, {sql_nullable_string(record.get('campaign_id'))}, "
        f"{sql_nullable_string(record.get('user_id'))}, {sql_nullable_string(record.get('page_url'))}, "
        f"{sql_timestamp(occurred_at)}, {sql_date(occurred_at)}, {sql_string(payload_text)}"
        ")"
    )
    return event_id, value


def processed_object_value(metadata: dict, processed_at: datetime) -> str:
    # Deprecated: replaced by `object_index_value` for the metadata-only index.
    return (
        "("
        f"{sql_string(metadata['object_key'])}, {sql_nullable_string(metadata.get('etag'))}, "
        f"{sql_string(metadata['source_topic'])}, {sql_timestamp(metadata['last_modified'])}, {sql_timestamp(processed_at)}"
        ")"
    )


def processed_object_keys() -> set[str]:
    rows = trino_execute("SELECT object_key FROM iceberg.ingress.raw_object_index")
    return {str(row[0]) for row in rows if row and row[0] is not None}


def list_unprocessed_bronze_objects() -> list[dict]:
    client = minio_client()
    seen_keys = processed_object_keys()
    discovered: list[dict] = []
    paginator = client.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=os.environ["MINIO_BUCKET"], Prefix="bronze/"):
        for item in page.get("Contents", []):
            object_key = item["Key"]
            if object_key.endswith("/") or object_key in seen_keys:
                continue
            discovered.append(
                {
                    "object_key": object_key,
                    "etag": item.get("ETag", "").strip('"') or None,
                    "source_topic": bronze_object_topic(object_key),
                    "last_modified": item["LastModified"],
                    "size": item.get("Size"),
                }
            )
    discovered.sort(key=lambda item: item["object_key"])
    return discovered


# Event-level parsing and materialization into Iceberg has been removed.
# We keep only object-level metadata in `iceberg.ingress.raw_object_index` and
# rely on the Hive external table for on-demand object reads.


def object_index_value(metadata: dict, processed_at: datetime) -> str:
    return (
        "("
        f"{sql_string(metadata['object_key'])}, {sql_nullable_string(metadata.get('etag'))}, "
        f"{sql_string(metadata['source_topic'])}, {sql_timestamp(metadata['last_modified'])}, "
        f"{sql_timestamp(processed_at)}, {sql_date(processed_at)}, {sql_nullable_int(metadata.get('size'))}"
        ")"
    )


def record_object_index_entries(objects: list[dict]) -> None:
    processed_at = datetime.now(timezone.utc)
    for batch in chunked(objects, 200):
        values = ", ".join(object_index_value(item, processed_at) for item in batch)
        trino_execute(
            "INSERT INTO iceberg.ingress.raw_object_index "
            "(object_key, etag, source_topic, last_modified, processed_at, processed_date, size) VALUES "
            + values
        )


@op(out=Out(Nothing))
def fetch_third_party_data(context: OpExecutionContext) -> None:
    with instrumented_op(context, "fetch_third_party_data"):
        trino_execute_file(Path(os.environ["TRINO_INIT_DIR"]) / "01_ingress_catalogs.sql")
        response = requests.get(
            f"{os.environ['THIRD_PARTY_API_BASE_URL']}/partners/snapshot",
            headers={"X-Correlation-ID": context.run_id},
            timeout=30,
        )
        response.raise_for_status()
        records = response.json()

        captured_at = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        object_key = f"lakehouse/ingress/third_party/captured_at={captured_at}/snapshot.json"
        snapshot_captured_at = datetime.now(timezone.utc)
        values = []
        for record in records:
            values.append(
                "("
                f"{sql_string(record['record_id'])}, {sql_string(record['provider'])}, {sql_nullable_string(record.get('account_id'))}, "
                f"{sql_timestamp(record['collected_at'])}, {sql_timestamp(snapshot_captured_at)}, {sql_date(snapshot_captured_at)}, {sql_string(object_key)}, "
                f"{sql_string(json.dumps(record.get('payload', {}), separators=(',', ':'), sort_keys=True))}"
                ")"
            )
        trino_execute(
            "INSERT INTO iceberg.ingress.third_party_snapshots "
            "(record_id, provider, account_id, collected_at, captured_at, captured_date, object_key, payload) VALUES "
            + ", ".join(values)
        )

        with appdb_connection() as connection:
            with connection.cursor() as cursor:
                cursor.executemany(
                    """
                    insert into raw.raw_third_party_records (
                        record_id,
                        provider,
                        account_id,
                        collected_at,
                        payload
                    ) values (%s, %s, %s, %s, %s)
                    on conflict (record_id) do nothing
                    """,
                    [
                        (
                            record["record_id"],
                            record["provider"],
                            record.get("account_id"),
                            record["collected_at"],
                            Jsonb(record.get("payload", {})),
                        )
                        for record in records
                    ],
                )

        record_rows("raw_third_party_records", len(records))
        log_event(context, "third_party_records_loaded", record_count=len(records), object_key=object_key)


@op(ins={"start": In(Nothing)})
def run_dbt_models(context: OpExecutionContext) -> None:
    with instrumented_op(context, "run_dbt_models"):
        completed = subprocess.run(
            [
                "dbt",
                "run",
                "--project-dir",
                os.environ["DBT_PROJECT_DIR"],
                "--profiles-dir",
                os.environ["DBT_PROFILES_DIR"],
            ],
            capture_output=True,
            text=True,
            check=False,
        )

        run_results_path = Path(os.environ["DBT_PROJECT_DIR"]) / "target" / "run_results.json"
        if run_results_path.exists():
            run_results = json.loads(run_results_path.read_text())
            log_event(
                context,
                "dbt_run_results",
                result_count=len(run_results.get("results", [])),
                elapsed_time=run_results.get("elapsed_time", 0),
            )

        if completed.returncode != 0:
            record_dbt_run("error")
            log_event(
                context,
                "dbt_run_failed",
                return_code=completed.returncode,
                stderr_tail=completed.stderr[-2000:],
            )
            raise RuntimeError("dbt run failed")

        record_dbt_run("success")
        log_event(context, "dbt_run_completed", stdout_tail=completed.stdout[-2000:])


@op(ins={"start": In(Nothing)})
def publish_serving_tables(context: OpExecutionContext) -> None:
    with instrumented_op(context, "publish_serving_tables"):
        trino_execute_file(Path(os.environ["TRINO_INIT_DIR"]) / "01_ingress_catalogs.sql")
        trino_execute("TRUNCATE TABLE clickhouse.serving.campaign_performance")
        trino_execute(
            """
            INSERT INTO clickhouse.serving.campaign_performance
            SELECT
                campaign_id,
                CAST(delivered_events AS BIGINT),
                CAST(open_events AS BIGINT),
                CAST(click_events AS BIGINT),
                CAST(date_trunc('second', first_seen_at) AS TIMESTAMP(0)),
                CAST(date_trunc('second', last_seen_at) AS TIMESTAMP(0))
            FROM postgresql.analytics.mart_campaign_performance
            """
        )
        trino_execute("TRUNCATE TABLE clickhouse.serving.product_engagement")
        trino_execute(
            """
            INSERT INTO clickhouse.serving.product_engagement
            SELECT
                COALESCE(page_url, ''),
                COALESCE(event_name, ''),
                CAST(event_count AS BIGINT),
                CAST(unique_users AS BIGINT),
                CAST(date_trunc('second', first_seen_at) AS TIMESTAMP(0)),
                CAST(date_trunc('second', last_seen_at) AS TIMESTAMP(0))
            FROM postgresql.analytics.mart_product_engagement
            """
        )

        campaign_rows = trino_execute("SELECT count(*) FROM clickhouse.serving.campaign_performance")
        engagement_rows = trino_execute("SELECT count(*) FROM clickhouse.serving.product_engagement")
        campaign_count = int(campaign_rows[0][0]) if campaign_rows else 0
        engagement_count = int(engagement_rows[0][0]) if engagement_rows else 0

        record_rows("campaign_performance", campaign_count)
        record_rows("product_engagement", engagement_count)
        log_event(
            context,
            "serving_tables_published",
            campaign_rows=campaign_count,
            engagement_rows=engagement_count,
        )


@op(ins={"start": In(Nothing)})
def sync_object_metadata(context: OpExecutionContext) -> None:
    with instrumented_op(context, "sync_object_metadata"):
        trino_execute_file(Path(os.environ["TRINO_INIT_DIR"]) / "01_ingress_catalogs.sql")
        bronze_objects = list_unprocessed_bronze_objects()
        if not bronze_objects:
            index_rows = trino_execute("SELECT count(*) FROM iceberg.ingress.raw_object_index")
            index_count = int(index_rows[0][0]) if index_rows else 0
            record_rows("iceberg_ingress_object_index", index_count)
            log_event(context, "object_index_synced", object_rows=index_count, processed_objects=0)
            return

        record_object_index_entries(bronze_objects)

        index_rows = trino_execute("SELECT count(*) FROM iceberg.ingress.raw_object_index")
        index_count = int(index_rows[0][0]) if index_rows else 0
        record_rows("iceberg_ingress_object_index", index_count)
        log_event(
            context,
            "object_index_synced",
            object_rows=index_count,
            processed_objects=len(bronze_objects),
        )


@job
def refresh_batch_and_serving() -> None:
    fetched = fetch_third_party_data()
    transformed = run_dbt_models(fetched)
    published = publish_serving_tables(transformed)
    # Object materialization is no longer automatic; object metadata is indexed separately.


refresh_batch_and_serving_schedule = ScheduleDefinition(
    job=refresh_batch_and_serving,
    cron_schedule="0 2 * * *",
)


@op(out=Out(Nothing))
def init_trino_ingress_catalogs(context: OpExecutionContext) -> None:
    with instrumented_op(context, "init_trino_ingress_catalogs"):
        trino_execute_file(Path(os.environ["TRINO_INIT_DIR"]) / "01_ingress_catalogs.sql")
        log_event(context, "trino_ingress_catalogs_initialized")


@job
def init_trino_catalogs_job() -> None:
    init_trino_ingress_catalogs()


@job
def sync_object_metadata_job() -> None:
    sync_object_metadata()


sync_object_metadata_schedule = ScheduleDefinition(
    job=sync_object_metadata_job,
    cron_schedule="0 * * * *",
)


defs = Definitions(
    jobs=[refresh_batch_and_serving, init_trino_catalogs_job, sync_object_metadata_job],
    schedules=[refresh_batch_and_serving_schedule, sync_object_metadata_schedule],
)