-- Materialized view: parse JSON `payload` strings from `serving.raw_payload`
-- into typed columns and insert into `serving.raw_events`.
-- Apply with:
-- curl -u ${CLICKHOUSE_USER:-default}:${CLICKHOUSE_PASSWORD:-clickhouse} \
--   -X POST --data-binary @clickhouse/init/01_mv_parse_raw_payload.sql 'http://localhost:8123/'

CREATE MATERIALIZED VIEW IF NOT EXISTS serving.mv_raw_payload_to_events
TO serving.raw_events
AS
SELECT
  if(length(JSONExtractString(payload,'event'))>0,
     JSONExtractString(payload,'event'),
     JSONExtractString(payload,'event_name')) AS event_name,
  JSONExtractString(payload,'campaign_id') AS campaign_id,
  JSONExtractString(payload,'user_id') AS user_id,
  parseDateTimeBestEffort(JSONExtractString(payload,'occurred_at')) AS occurred_at,
  payload
FROM serving.raw_payload;

-- Backfill existing rows from `serving.raw_payload` into `serving.raw_events`.
-- NOTE: This is a one-time backfill; running it multiple times may insert duplicates.
INSERT INTO serving.raw_events
SELECT
  if(length(JSONExtractString(payload,'event'))>0,
    JSONExtractString(payload,'event'),
    JSONExtractString(payload,'event_name')) AS event_name,
  JSONExtractString(payload,'campaign_id') AS campaign_id,
  JSONExtractString(payload,'user_id') AS user_id,
  if(match(JSONExtractString(payload,'occurred_at'), '^[0-9]{4}-[0-9]{2}-[0-9]{2}'),
    parseDateTimeBestEffort(JSONExtractString(payload,'occurred_at')),
    toDateTime('1970-01-01 00:00:00')) AS occurred_at,
  payload
FROM serving.raw_payload;
