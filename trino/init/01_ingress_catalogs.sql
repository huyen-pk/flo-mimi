-- Drop demo tables and replace with a clean, metadata-only object index.
DROP TABLE IF EXISTS iceberg.ingress.raw_events;
DROP TABLE IF EXISTS iceberg.ingress.processed_raw_event_objects;

CREATE SCHEMA IF NOT EXISTS iceberg.ingress
WITH (location = 's3://data-lake/lakehouse/ingress');

-- Metadata-only index of MinIO bronze objects. Stores one row per object.
CREATE TABLE IF NOT EXISTS iceberg.ingress.raw_object_index (
    object_key VARCHAR,
    etag VARCHAR,
    source_topic VARCHAR,
    last_modified TIMESTAMP(3),
    processed_at TIMESTAMP(3),
    processed_date DATE,
    size BIGINT
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['processed_date','source_topic']
);

-- Expose MinIO bronze files through the Hive connector so Trino can read objects on-demand.
CREATE SCHEMA IF NOT EXISTS hive.ingress;

-- This is a lightweight external table that maps the bronze prefix. Query the file contents
-- via the Hive connector (one or more rows per file depending on file format). Use the
-- file path or connector-provided metadata to join against `iceberg.ingress.raw_object_index`.
CREATE TABLE IF NOT EXISTS hive.ingress.bronze_objects (
    content VARCHAR
)
WITH (
    format = 'TEXTFILE',
    external_location = 's3://data-lake/bronze/'
);

CREATE TABLE IF NOT EXISTS iceberg.ingress.third_party_snapshots (
    record_id VARCHAR,
    provider VARCHAR,
    account_id VARCHAR,
    collected_at TIMESTAMP(3),
    captured_at TIMESTAMP(3),
    captured_date DATE,
    object_key VARCHAR,
    payload VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['captured_date', 'provider']
);

CREATE SCHEMA IF NOT EXISTS clickhouse.serving;

CREATE TABLE IF NOT EXISTS clickhouse.serving.raw_payload (
    payload VARCHAR
)
WITH (
    engine = 'Log'
);

CREATE TABLE IF NOT EXISTS clickhouse.serving.raw_events (
    event_name VARCHAR,
    campaign_id VARCHAR,
    user_id VARCHAR,
    occurred_at TIMESTAMP(0) NOT NULL,
    payload VARCHAR
)
WITH (
    engine = 'MergeTree',
    order_by = ARRAY['occurred_at']
);

CREATE TABLE IF NOT EXISTS clickhouse.serving.campaign_performance (
    campaign_id VARCHAR NOT NULL,
    delivered_events BIGINT,
    open_events BIGINT,
    click_events BIGINT,
    first_seen_at TIMESTAMP(0),
    last_seen_at TIMESTAMP(0)
)
WITH (
    engine = 'MergeTree',
    order_by = ARRAY['campaign_id']
);

CREATE TABLE IF NOT EXISTS clickhouse.serving.product_engagement (
    page_url VARCHAR NOT NULL,
    event_name VARCHAR NOT NULL,
    event_count BIGINT,
    unique_users BIGINT,
    first_seen_at TIMESTAMP(0),
    last_seen_at TIMESTAMP(0)
)
WITH (
    engine = 'MergeTree',
    order_by = ARRAY['page_url', 'event_name']
);