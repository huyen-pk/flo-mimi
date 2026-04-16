CREATE SCHEMA IF NOT EXISTS iceberg.ingress
WITH (location = 's3://data-lake/lakehouse/ingress');

CREATE TABLE IF NOT EXISTS iceberg.ingress.raw_events (
    event_id VARCHAR,
    event_name VARCHAR,
    campaign_id VARCHAR,
    user_id VARCHAR,
    page_url VARCHAR,
    occurred_at TIMESTAMP(3),
    event_date DATE,
    payload VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['event_date']
);

CREATE TABLE IF NOT EXISTS iceberg.ingress.processed_raw_event_objects (
    object_key VARCHAR,
    etag VARCHAR,
    source_topic VARCHAR,
    last_modified TIMESTAMP(3),
    processed_at TIMESTAMP(3)
)
WITH (
    format = 'PARQUET'
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
    occurred_at TIMESTAMP(0),
    payload VARCHAR
)
WITH (
    engine = 'MergeTree',
    order_by = ARRAY['occurred_at']
);

CREATE TABLE IF NOT EXISTS clickhouse.serving.campaign_performance (
    campaign_id VARCHAR,
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