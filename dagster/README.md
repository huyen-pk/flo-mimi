# Dagster Orchestration and Batch Workflows

This document outlines the Dagster jobs in the data engineering platform, explaining their intended purposes, implementation choices, and the underlying cost and operational tradeoff analysis.

## `refresh_batch_and_serving`
**Purpose**: Refreshes canonical analytics data models and publishes a reliable snapshot out to the low-latency serving layer (ClickHouse).
**Schedule**: Daily at 02:00.

1. `fetch_third_party_data`: Acquires external datasets and snapshots them to PostgreSQL and MinIO.
2. `run_dbt_models`: Runs robust `dbt` transformations against the PostgreSQL application database.
3. `publish_serving_tables`: Pushes transformed analytics tables into ClickHouse via Trino.

### Tradeoff Analysis: Serving Layer Ingestion

The `publish_serving_tables` step executes a cross-system data movement: `TRUNCATE` ClickHouse tables, followed by a Trino-coordinated `INSERT ... SELECT` from PostgreSQL.

**Why Truncate & Snapshot vs. Incremental or CDC:**
- **Truncate & Snapshot (Current)**:
  - *Pros*: Inherently idempotent, prevents split-brain anomalies, guarantees atomic consistency matching the latest transformation run. Avoids tight schema-coupling. Lowest development cost.
  - *Cons*: Expensive for high-volume datasets (full scan and write), temporary latency/availability gaps during swap.
- **CDC / Streaming Ingest (Redpanda -> ClickHouse)**:
  - *Pros*: Near zero-latency serving updates.
  - *Cons*: **Highest Operational Cost**. Operating stateful CDC agents (e.g. Debezium, Kafka Connect) means managing complex error handling, deduplication logic (using specialized `ReplacingMergeTree` views), schema evolution issues, offset management, and re-processing streams during incidents.
- **Direct dbt to ClickHouse Integration**:
  - *Pros*: One less data-movement step.
  - *Cons*: Couples transformations tightly to the serving DB, risking mixed read/write workloads impacting low-latency product dashboards.
- **Incremental Inserts with Watermarking**:
  - *Pros*: High performance batch sizes.
  - *Cons*: Medium engineering cost to manage delayed arrivals, exact-once timestamps, and resolving tricky late-update edge cases.

**Conclusion**: For the serving layer, we strongly prioritize consistency, pipeline observability, and minimizing operational complexity over real-time analytics parity. The pure-batch approach over Trino effectively isolates transformation targets while delivering a deterministic table state.

---

## `sync_object_metadata_job`
**Purpose**: Continuously catalogs new raw JSON event objects arriving into MinIO from event streams without expanding storage footprint or duplicating payloads.
**Schedule**: Hourly.

1. Lists unprocessed blobs within `<bucket>/bronze/` using lightweight S3 paginators.
2. Inserts file-level telemetry into Trino Iceberg index `iceberg.ingress.raw_object_index`.

### Tradeoff Analysis: Data Lake Ingestion Strategy

The system indexes only minimal object telemetry (timestamp, source topic, etag, payload byte size) within an Iceberg catalog, rather than opening and extracting standard table rows. Full file data is fetched on-the-fly dynamically via Trino's `hive.ingress.bronze_objects` external table connector.

**Why Metadata-Only vs. Parsing and Materializing Parquet:**
- **Metadata Index + External S3 Tables (Current)**:
  - *Pros*: Vastly reduces ingest compute cost and duplicates zero data. The index tracks ingested state immediately and scales independently without needing to parse and rewrite petabytes of unstructured JSON objects into new Parquet files on disk. 
  - *Cons*: Ad-hoc queries via Hive scanning raw JSON are noticeably slower and lack performance benefits drawn from structural predicate pushdowns and columnar optimizations.
- **Full Row-Level Materialization into Iceberg (e.g., old standard `MERGE INTO`)**:
  - *Pros*: Sub-second interactive read speeds across thousands of events.
  - *Cons*: Very costly ingestion loops. Expanding events individually requires compute-intense JSON unwrapping, UUID-based deduplication logic, memory staging, and expensive continuous Parquet compactions (small file problems) on MinIO.

**Conclusion**: Since raw events represent a cold-retention tier strictly meant for audits, replays, or infrequent ad-hoc discovery, allocating heavy, daily write resources to structure them is deemed poor ROI. The minimal metadata catalog balances keeping an accurate auditable view of ingestion while offloading parse costs purely to read-time when deliberately interrogated by Trino.
