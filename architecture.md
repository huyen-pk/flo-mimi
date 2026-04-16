**Architecture Overview**

This document describes the major components in this workspace and how they are wired together to enable event-driven ingestion, transformation with dbt, orchestration with Dagster, and analytics via Trino and appdb.

**Components**
- **Orchestrator**: Dagster — code and container configuration live under [dagster/repository.py](dagster/repository.py) and [dagster/Dockerfile](dagster/Dockerfile). Dagster coordinates pipelines that run ingestion, dbt, and downstream jobs.
- **Transformation (dbt)**: dbt project is at [dagster/dbt/dbt_project.yml](dagster/dbt/dbt_project.yml). Models are in `models/` with staging and marts (e.g., [dagster/dbt/models/marts](dagster/dbt/models/marts)). dbt models build tables/views in appdb.
- **Event Gateway / Producers**: The event gateway service is under [event-gateway/app/main.py](event-gateway/app/main.py). Producers publish events to the streaming layer.
- **Platform App**: The embedded Go + Svelte operator console lives under [platform](platform). It serves the UI from an embedded bundle, loads its display model from normalized SQL tables and the `analytics.platform_bootstrap` view in appdb, and forwards click interactions to `event-gateway` so they land in the same analytics and campaign event path as other producers.
- **Mock Third-Party API**: A lightweight mock service for upstream dependencies at [mock-third-party-api/app/main.py](mock-third-party-api/app/main.py).
- **Event Bus**: Redpanda cluster + connectors; config in [redpanda-connect/connect.yaml](redpanda-connect/connect.yaml). Redpanda is the central event backbone for pub/sub and durable streaming.
- **Connectors**: Redpanda Connect lands raw topic payloads directly into MinIO bronze storage and ClickHouse raw landing tables.
- **Query Engine**: Trino — server config and catalogs under [trino/catalog](trino/catalog). Trino provides ad-hoc SQL access across catalogs (ClickHouse, Iceberg, appdb, etc.).
- **AppDB / Init Scripts**: SQL initialization and schema setup are in [appdb/init/01_init.sql](appdb/init/01_init.sql).
- **Platform / UI Assets**: Design and secure UI prototypes in [platform/design](platform/design) that reference dashboard and secure components.
- **Docker Compose**: Top-level orchestration for local dev in [docker-compose.yml](docker-compose.yml). Services are wired here for local runs.

**Wiring and Dataflows**
- Events produced by services (e.g., `event-gateway`) are published to Redpanda topics.
- Operator interactions from the embedded `platform` UI are posted to the Go backend, which proxies them to `event-gateway` as analytics events and, for campaign actions, email events.
- Dashboard cards and campaign performance read consolidated analytics values from appdb tables, while platform reference content is stored in normalized relational tables and assembled through SQL views.
- Redpanda topics are consumed by:
  - Stream processors or microservices that perform enrichment in-flight.
  - Redpanda Connect tasks that land raw payloads directly into MinIO and ClickHouse.
  - Dagster jobs that periodically synchronize newly landed MinIO bronze objects into Iceberg tables on MinIO through Trino, while ClickHouse remains a serving projection.
- Dagster orchestrates scheduled and ad-hoc workflows:
  - Ingest jobs that pull from appdb or call external APIs (mock-third-party-api).
  - Triggers dbt runs to materialize models in appdb (uses [dagster/dbt/dbt_project.yml](dagster/dbt/dbt_project.yml)).
  - Runs tests and downstream tasks (e.g., publishing metrics, refreshing dashboards).
- dbt transforms staged data into marts (business-level models) in appdb, which are then queried by Trino for analytics and dashboards.
- Trino reads from configured catalogs (see [trino/catalog](trino/catalog)) and is used for batch publication and Iceberg-aware synchronization, rather than per-event streaming ingress.

**Local development flow**
- Use `docker-compose up` to bring up local dev services defined in [docker-compose.yml](docker-compose.yml).
- Run Dagster pipelines locally or in the Dagster container to exercise end-to-end flows.
- Use Redpanda topics and the provided Connect config to test sink behavior; validate that dbt runs produce expected tables.

**References**
- Docker compose and service definitions: [docker-compose.yml](docker-compose.yml)
- Dagster repository and workspace: [dagster/repository.py](dagster/repository.py)
- dbt: [dagster/dbt/dbt_project.yml](dagster/dbt/dbt_project.yml)
- Redpanda connect config: [redpanda-connect/connect.yaml](redpanda-connect/connect.yaml)
- Trino catalogs: [trino/catalog](trino/catalog)
- Appdb init: [appdb/init/01_init.sql](appdb/init/01_init.sql)
---

### Architectural Analysis: Event Storage & Ingestion Strategy

#### 1. Current State: Parallel Dual-Ingest (Micro-Lambda Architecture)
Currently, event data is ingested in parallel into both **ClickHouse** and **MinIO**. This dual-path approach implements a "Tiered Storage" pattern that functions as a Micro-Lambda architecture:
*   **Hot / Speed Layer (ClickHouse):** Provides sub-second query performance necessary for real-time dashboards, operational alerts, and high-concurrency APIs.
*   **Cold / Batch Layer (MinIO):** Provides durable, low-cost long-term object storage. This acts as the historical record for disaster recovery and heavy data science workloads (e.g., Spark, DuckDB).

**Challenges of the Current State:**
While ingestion is extremely fast (pushed to both destinations simultaneously), maintaining two parallel configuration pipelines creates an operational burden. It introduces the risk of **logic divergence**—if one ingestion pipe fails, lags, or applies transformations differently, the "Hot" and "Cold" layers will serve conflicting data, breaking the single source of truth.

---

#### 2. Architectural Evolution: Transitioning to a Unified "Kappa" Architecture
To resolve the duplication limitations of the Lambda architecture, the system should evolve toward a Kappa-style architecture. This ensures transformation logic is written exactly once, and serving layers read from a unified source of truth. 

Three primary unification strategies exist:

**Option A: The "Lakehouse" Approach (Iceberg-First) — Recommended**
*   **Architecture:** Redpanda Connect streams data exclusively to MinIO in Iceberg format. ClickHouse uses the `Iceberg` table engine to query these remote files.
*   **Latency Impact:** Higher ingestion latency (batched every ~30–60 seconds to prevent file bloat) and slower query latency (bottlenecked by network reads over MinIO).
*   **Pros:** Fully eliminates ingestion duplication. Provides a highly "open" architecture—data serves as an immutable source of truth accessible synchronously by ClickHouse, Trino, and Spark.

**Option B: ClickHouse Native S3 Tiering**
*   **Architecture:** All data is streamed exclusively into ClickHouse. A storage policy automatically offloads aged data parts to MinIO (S3-backed MergeTree).
*   **Latency Impact:** Ultra-low ingestion latency (writes hit local disk instantly). Query latency is sub-second for recent data, and medium for historical data retrieved from MinIO.
*   **Pros:** Requires no manual data movement and keeps recent streaming data blazing fast, optimizing storage costs under the hood. 

**Option C: The Materialized View Pattern (Traffic Controller)**
*   **Architecture:** Ingest raw events into a ClickHouse staging table. A Materialized View is then used to transform, clean, and export a copy of the data to an S3/MinIO table engine.
*   **Latency Impact:** Fast ingestion and ultra-fast hot queries, but pushes the export latency penalty to the background.
*   **Pros:** Consolidates ingestion into a single Redpanda consumer, using ClickHouse to enforce data validation before it lands in the broader data lake. However, it increases ClickHouse CPU overhead.

---

#### 3. Performance & Latency Summary

| Strategy | Ingestion Latency | Query Speed (Hot Data) | Query Speed (Cold Data) | Maintainability |
| :--- | :--- | :--- | :--- | :--- |
| **Current (Parallel)** | ⚡ Fast | ⚡ Fast | 🐢 Slow | Low (Dual-pipeline drift risk) |
| **Option A (Iceberg)** | 🐢 Slow (Batching) | 📉 Medium (Network) | 🐢 Slow | High (Single immutable truth) |
| **Option B (Tiering)** | ⚡ Fast | ⚡ Fast | 📉 Medium | Medium (ClickHouse-centric) |
| **Option C (MV)** | ⚡ Fast | ⚡ Fast | 🐢 Slow | Medium (Heavy ClickHouse compute) |

---

#### 4. Recommendation
The choice of architecture depends on the core platform priority:
*   **If Real-Time Visibility is Critical:** **Option B (S3 Tiering)** is best. It retains the lowest-possible ingestion latency and serves fresh data immediately off NVMe, while still cutting long-term storage costs.
*   **If System-Wide Consistency is Critical:** **Option A (Lakehouse/Iceberg)** is best. Because the current environment is already equipped with Trino and Iceberg catalogs, treating MinIO/Iceberg as the definitive "Source of Truth" natively aligns with the stack. It unifies operations at the cost of slight (seconds-to-minutes) ingestion lag, providing guaranteed data consistency across all query engines.
