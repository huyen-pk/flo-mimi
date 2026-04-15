# Local Data Engineering Stack

This workspace recreates the architecture from `architecture.md` with Docker Compose using local equivalents for cloud-managed services.

## Components

- `event-gateway`: receives streaming email-marketing and analytics events and publishes them to Redpanda.
- `platform`: embedded Go + Svelte operator console that mirrors the design system in `platform/design`, loads relational seed data from appdb tables and views, and routes click interactions into `event-gateway`.
- `mock-third-party-api`: simulates a scheduled third-party data source.
- `dagster-webserver` and `dagster-daemon`: orchestrate scheduled collection from third-party APIs, run dbt transformations, and publish serving tables into ClickHouse.
- `redpanda`: event bus for both real-time sources. (~AWS DMS, CDC, Flink)
- `stream-loader`: sinks raw stream topics into MinIO bronze storage.
- `minio`: object storage for bronze files and future lakehouse tables. (~S3)
- `appdb`: PostgreSQL app database for platform state, consolidated analytics, and dbt transformations. (~Aurora)
- `clickhouse`: low-latency serving store.
- `trino`: query layer across PostgreSQL, ClickHouse, and an Iceberg catalog backed by MinIO. (~Athena)
- `catalog-db`: PostgreSQL metadata store for the Trino Iceberg catalog.

## Networks

- `ingress_net`: inbound event traffic into `event-gateway`
- `stream_net`: Redpanda and raw stream landing into MinIO
- `batch_net`: Dagster and third-party data collection
- `lakehouse_net`: MinIO, Trino, and Iceberg catalog services
- `serving_net`: appdb, ClickHouse, and data-serving components

## Start The Stack

1. Optionally create a `.env` file if you want to override the default credentials, ports, or image tags.
2. Start the platform:

```bash
docker compose up --build -d
```

If Docker Hub pulls are flaky or BuildKit cancels concurrent fetches on your machine, build the custom images sequentially first:

```bash
COMPOSE_PARALLEL_LIMIT=1 docker compose build event-gateway mock-third-party-api dagster-webserver
docker compose up -d
```

If the Dagster image fails during `pip install` with a transient network error such as `BrokenPipeError`, rerun the same sequential build command. The Dockerfiles now use longer pip timeouts and retries to reduce this failure mode.

The `stream-loader` service defaults to `docker.redpanda.com/redpandadata/connect:latest` because the previously pinned `4.44.1` tag is not available in the registry. If you want a stricter pin, set `STREAM_LOADER_IMAGE` in `.env` to a tag or digest you have verified.

If `docker compose up` reports that `redpanda` is unhealthy, recreate it after pulling the latest Compose changes. The original healthcheck used an unsupported `rpk cluster health --brokers=...` flag for the current Redpanda image. The fixed probe is just `rpk cluster health`.

If `minio` is marked unhealthy even though its logs show the server started, recreate it after pulling the latest Compose changes. The original healthcheck used `wget`, which is not present in the selected MinIO image. The fixed probe uses `curl` against `/minio/health/live`.

3. Open the main operator endpoints:

- Dagster UI: http://localhost:3000
- Grafana: http://localhost:3001
- Event gateway: http://localhost:8000
- Prometheus: http://localhost:9090
- Platform UI: http://localhost:8081
- Trino: http://localhost:8080
- Loki: http://localhost:3100
- MinIO console: http://localhost:9001
- Tempo: http://localhost:3200
- ClickHouse HTTP: http://localhost:8123

## Smoke Test

Send sample streaming data:

```bash
curl -X POST http://localhost:8000/events/email \
  -H 'Content-Type: application/json' \
  -d '{
    "campaign_id": "spring-launch",
    "recipient_id": "user-001",
    "event_type": "open",
    "payload": {"subject": "Spring Launch"}
  }'

curl -X POST http://localhost:8000/events/analytics \
  -H 'Content-Type: application/json' \
  -d '{
    "session_id": "session-001",
    "user_id": "user-001",
    "event_name": "page_view",
    "page_url": "/pricing",
    "payload": {"utm_source": "email"}
  }'
```

Run the Dagster job `refresh_batch_and_serving` from the UI, or materialize it on schedule.

The platform UI emits tracked interactions through the existing data-platform ingress. Campaign-oriented actions such as `Create New Brief` and campaign-row action buttons emit both analytics and campaign events; navigation, search, and filter actions emit analytics events.

## Observability

The stack now includes a local observability baseline in `observability/`:

- `Prometheus` scrapes custom service metrics, Dagster metrics, Redpanda, Redpanda Connect, Trino, ClickHouse, and PostgreSQL exporters.
- `Grafana` provisions Prometheus, Loki, and Tempo datasources automatically, plus starter dashboards for services and pipeline health.
- `Loki` stores container and application logs collected by `promtail`.
- `Tempo` stores traces emitted by the instrumented Go and Python services through the OpenTelemetry Collector.
- `OpenTelemetry Collector` receives OTLP traces from `platform`, `event-gateway`, `mock-third-party-api`, and the Dagster processes.

Custom service telemetry behavior:

- `platform`, `event-gateway`, and `mock-third-party-api` expose `/metrics`.
- These services emit structured request logs and accept or generate `X-Correlation-ID` headers.
- `event-gateway` adds `correlation_id` to published event payloads and Kafka headers.
- `dagster-daemon` and `dagster-webserver` expose Prometheus metrics on internal ports `9108` and `9109`.

Starter dashboards are provisioned from:

- `observability/grafana/dashboards/services-overview.json`
- `observability/grafana/dashboards/data-pipeline-overview.json`

If you want to inspect the telemetry plumbing directly after startup:

```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
curl http://localhost:8081/metrics | head
curl -H 'X-Correlation-ID: demo-trace-001' http://localhost:8000/health
docker compose logs --tail=50 otel-collector promtail loki tempo
```

The Go `platform` module now resolves observability dependencies that require Go `1.25.x` for local module operations such as `go test` and `go mod tidy`.

## Notes

- Raw streaming payloads land in MinIO under `bronze/email_events_raw` and `bronze/analytics_events_raw`.
 - Raw streaming payloads land in MinIO under `bronze/email_events_raw` and `bronze/analytics_events_raw`.
 - Redpanda Connect now also writes wrapped raw event payloads into ClickHouse table `serving.raw_payload`. A Materialized View `serving.mv_raw_payload_to_events` parses the JSON `payload` into typed columns and populates `serving.raw_events` for low-latency serving. See [clickhouse/init/01_mv_parse_raw_payload.sql](clickhouse/init/01_mv_parse_raw_payload.sql).
- dbt models live in `dagster/dbt` and run against PostgreSQL appdb as the transformation target.
- Trino is configured with PostgreSQL, ClickHouse, and Iceberg catalogs so you can add direct MinIO-backed lakehouse tables later without changing the network topology.
- The platform app is built from `platform/` as a multi-stage container that embeds the Svelte bundle into the Go binary.
- The platform bootstrap model now comes from the `analytics.platform_bootstrap` SQL view, backed by normalized tables seeded in [appdb/init/02_platform_bootstrap.sql](appdb/init/02_platform_bootstrap.sql) and live aggregate views in [appdb/init/03_platform_bootstrap_views.sql](appdb/init/03_platform_bootstrap_views.sql).
- The current appdb seed can be exported as SQL `INSERT` statements with [appdb/queries/export_platform_seed.sql](appdb/queries/export_platform_seed.sql).

## Applying ClickHouse materialized view (local)

To create the Materialized View locally (it will POPULATE existing `serving.raw_payload` rows), run:

```bash
curl -u ${CLICKHOUSE_USER:-default}:${CLICKHOUSE_PASSWORD:-clickhouse} \
  -X POST --data-binary @clickhouse/init/01_mv_parse_raw_payload.sql 'http://localhost:8123/'
```

This will create `serving.mv_raw_payload_to_events` which writes parsed rows into `serving.raw_events`.