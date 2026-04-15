import json
import os
import subprocess
from io import BytesIO
from pathlib import Path
from datetime import datetime, timezone

import clickhouse_connect
import psycopg
import requests
from dagster import Definitions, In, Nothing, OpExecutionContext, ScheduleDefinition, job, op
from minio import Minio
from psycopg.types.json import Jsonb

from telemetry import instrumented_op, log_event, record_dbt_run, record_rows, setup_telemetry


setup_telemetry()


def minio_client() -> Minio:
    return Minio(
        endpoint=os.environ["MINIO_ENDPOINT"],
        access_key=os.environ["MINIO_ACCESS_KEY"],
        secret_key=os.environ["MINIO_SECRET_KEY"],
        secure=False,
    )


def appdb_connection():
    return psycopg.connect(os.environ["APPDB_DSN"])


@op(out=Nothing)
def fetch_third_party_data(context: OpExecutionContext) -> None:
    with instrumented_op(context, "fetch_third_party_data"):
        response = requests.get(
            f"{os.environ['THIRD_PARTY_API_BASE_URL']}/partners/snapshot",
            headers={"X-Correlation-ID": context.run_id},
            timeout=30,
        )
        response.raise_for_status()
        records = response.json()

        bucket = os.environ["MINIO_BUCKET"]
        captured_at = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        object_key = f"bronze/third_party/captured_at={captured_at}/snapshot.json"
        client = minio_client()
        payload = json.dumps(records).encode("utf-8")
        client.put_object(
            bucket,
            object_key,
            data=BytesIO(payload),
            length=len(payload),
            content_type="application/json",
            metadata={"correlation-id": context.run_id},
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
        clickhouse = clickhouse_connect.get_client(
            host=os.environ["CLICKHOUSE_HOST"],
            port=int(os.environ["CLICKHOUSE_PORT"]),
        )
        clickhouse.command("create database if not exists serving")
        clickhouse.command(
            """
            create table if not exists serving.campaign_performance (
                campaign_id String,
                delivered_events UInt64,
                open_events UInt64,
                click_events UInt64,
                first_seen_at DateTime,
                last_seen_at DateTime
            ) engine = ReplacingMergeTree()
            order by campaign_id
            """
        )
        clickhouse.command(
            """
            create table if not exists serving.product_engagement (
                page_url String,
                event_name String,
                event_count UInt64,
                unique_users UInt64,
                first_seen_at DateTime,
                last_seen_at DateTime
            ) engine = ReplacingMergeTree()
            order by (page_url, event_name)
            """
        )

        with appdb_connection() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    """
                    select campaign_id, delivered_events, open_events, click_events, first_seen_at, last_seen_at
                    from analytics.mart_campaign_performance
                    order by campaign_id
                    """
                )
                campaign_rows = cursor.fetchall()

                cursor.execute(
                    """
                    select page_url, event_name, event_count, unique_users, first_seen_at, last_seen_at
                    from analytics.mart_product_engagement
                    order by page_url, event_name
                    """
                )
                engagement_rows = cursor.fetchall()

        clickhouse.command("truncate table serving.campaign_performance")
        clickhouse.command("truncate table serving.product_engagement")

        if campaign_rows:
            clickhouse.insert(
                "serving.campaign_performance",
                campaign_rows,
                column_names=[
                    "campaign_id",
                    "delivered_events",
                    "open_events",
                    "click_events",
                    "first_seen_at",
                    "last_seen_at",
                ],
            )
        if engagement_rows:
            clickhouse.insert(
                "serving.product_engagement",
                engagement_rows,
                column_names=[
                    "page_url",
                    "event_name",
                    "event_count",
                    "unique_users",
                    "first_seen_at",
                    "last_seen_at",
                ],
            )

        record_rows("campaign_performance", len(campaign_rows))
        record_rows("product_engagement", len(engagement_rows))
        log_event(
            context,
            "serving_tables_published",
            campaign_rows=len(campaign_rows),
            engagement_rows=len(engagement_rows),
        )


@job
def refresh_batch_and_serving() -> None:
    fetched = fetch_third_party_data()
    transformed = run_dbt_models(fetched)
    publish_serving_tables(transformed)


refresh_batch_and_serving_schedule = ScheduleDefinition(
    job=refresh_batch_and_serving,
    cron_schedule="0 2 * * *",
)


@op(out=Nothing)
def init_clickhouse_streaming_schema(context: OpExecutionContext) -> None:
    with instrumented_op(context, "init_clickhouse_streaming_schema"):
        clickhouse = clickhouse_connect.get_client(
            host=os.environ["CLICKHOUSE_HOST"],
            port=int(os.environ["CLICKHOUSE_PORT"]),
        )
        clickhouse.command("create database if not exists serving")
        clickhouse.command("create database if not exists streaming")
        clickhouse.command(
            """
            create table if not exists serving.raw_events (
                event_name String,
                campaign_id String,
                user_id String,
                occurred_at DateTime,
                payload String
            ) engine = MergeTree()
            order by (occurred_at)
            """
        )

        clickhouse.command(
            "create table if not exists streaming.email_events_kafka (payload String) engine = Kafka('redpanda:9092', 'email_events_raw', 'ch_email_group', 'JSONEachRow')"
        )
        clickhouse.command(
            "create table if not exists streaming.analytics_events_kafka (payload String) engine = Kafka('redpanda:9092', 'analytics_events_raw', 'ch_analytics_group', 'JSONEachRow')"
        )

        clickhouse.command(
            "create materialized view if not exists mv_email_events to serving.raw_events as select JSONExtractString(payload, 'event') as event_name, JSONExtractString(payload, 'campaign_id') as campaign_id, JSONExtractString(payload, 'user_id') as user_id, parseDateTimeBestEffort(JSONExtractString(payload, 'timestamp')) as occurred_at, payload from streaming.email_events_kafka"
        )
        clickhouse.command(
            "create materialized view if not exists mv_analytics_events to serving.raw_events as select JSONExtractString(payload, 'event') as event_name, JSONExtractString(payload, 'page_url') as campaign_id, JSONExtractString(payload, 'user_id') as user_id, parseDateTimeBestEffort(JSONExtractString(payload, 'timestamp')) as occurred_at, payload from streaming.analytics_events_kafka"
        )
        log_event(context, "clickhouse_streaming_schema_initialized")


@job
def init_clickhouse_schema_job() -> None:
    init_clickhouse_streaming_schema()


defs = Definitions(
    jobs=[refresh_batch_and_serving, init_clickhouse_schema_job],
    schedules=[refresh_batch_and_serving_schedule],
)