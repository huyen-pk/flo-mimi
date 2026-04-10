**Architecture Overview**

Date: 2026-04-09

This document describes the major components in this workspace and how they are wired together to enable event-driven ingestion, transformation with dbt, orchestration with Dagster, and analytics via Trino and appdb.

**Components**
- **Orchestrator**: Dagster — code and container configuration live under [dagster/repository.py](dagster/repository.py) and [dagster/Dockerfile](dagster/Dockerfile). Dagster coordinates pipelines that run ingestion, dbt, and downstream jobs.
- **Transformation (dbt)**: dbt project is at [dbt/dbt_project.yml](dbt/dbt_project.yml). Models are in `models/` with staging and marts (e.g., [dbt/models/marts](dbt/models/marts)). dbt models build tables/views in appdb.
- **Event Gateway / Producers**: The event gateway service is under [event-gateway/app/main.py](event-gateway/app/main.py). Producers publish events to the streaming layer.
- **Platform App**: The embedded Go + Svelte operator console lives under [platform](platform). It serves the UI from an embedded bundle, loads its display model from normalized SQL tables and the `analytics.platform_bootstrap` view in appdb, and forwards click interactions to `event-gateway` so they land in the same analytics and campaign event path as other producers.
- **Mock Third-Party API**: A lightweight mock service for upstream dependencies at [mock-third-party-api/app/main.py](mock-third-party-api/app/main.py).
- **Event Bus**: Redpanda cluster + connectors; config in [redpanda-connect/connect.yaml](redpanda-connect/connect.yaml). Redpanda is the central event backbone for pub/sub and durable streaming.
- **Connectors**: Redpanda Connect (connect.yaml) moves data from topics to sink systems such as MinIO bronze storage and downstream consumers.
 - **Connectors**: Redpanda Connect (connect.yaml) moves data from topics to sink systems such as MinIO bronze storage and downstream consumers. It also writes wrapped raw events into ClickHouse (table `serving.raw_payload`) for low-latency serving; a Materialized View (`serving.mv_raw_payload_to_events`) parses the JSON `payload` string into typed columns and populates `serving.raw_events`.
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
  - Redpanda Connect tasks that sink topic data into bronze object storage for downstream processing.
  - Redpanda Connect tasks that also sink wrapped raw payloads into ClickHouse (`serving.raw_payload`) where a Materialized View parses them into `serving.raw_events` for real-time analytics.
- Dagster orchestrates scheduled and ad-hoc workflows:
  - Ingest jobs that pull from appdb or call external APIs (mock-third-party-api).
  - Triggers dbt runs to materialize models in appdb (uses [dbt_project.yml](dbt/dbt_project.yml)).
  - Runs tests and downstream tasks (e.g., publishing metrics, refreshing dashboards).
- dbt transforms staged data into marts (business-level models) in appdb, which are then queried by Trino for analytics and dashboards.
- Trino reads from configured catalogs (see [trino/catalog](trino/catalog)) to provide unified SQL access across systems.

**Local development flow**
- Use `docker-compose up` to bring up local dev services defined in [docker-compose.yml](docker-compose.yml).
- Run Dagster pipelines locally or in the Dagster container to exercise end-to-end flows.
- Use Redpanda topics and the provided Connect config to test sink behavior; validate that dbt runs produce expected tables.

**References**
- Docker compose and service definitions: [docker-compose.yml](docker-compose.yml)
- Dagster repository and workspace: [dagster/repository.py](dagster/repository.py)
- dbt: [dbt/dbt_project.yml](dbt/dbt_project.yml)
- Redpanda connect config: [redpanda-connect/connect.yaml](redpanda-connect/connect.yaml)
- Trino catalogs: [trino/catalog](trino/catalog)
- Appdb init: [appdb/init/01_init.sql](appdb/init/01_init.sql)

**Next steps / Suggestions**
- Add a small architecture diagram (PNG/SVG) under `platform/design` and reference it here.
- Add README snippets for running local end-to-end tests (Dagster -> dbt -> Trino).
