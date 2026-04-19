# Local Data Engineering Stack

This workspace recreates the architecture from `architecture.md` with Docker Compose using local equivalents for cloud-managed services.

## Components

- `event-gateway`: receives streaming email-marketing and analytics events and publishes them to Redpanda.
- `platform`: embedded Go + Svelte operator console that mirrors the design system in `platform/design`, loads relational seed data from appdb tables and views, and routes click interactions into `event-gateway`.
- `mock-third-party-api`: simulates a scheduled third-party data source.
- `dagster-webserver` and `dagster-daemon`: orchestrate scheduled collection from third-party APIs, run dbt transformations, and publish serving tables into ClickHouse.
- `redpanda`: event bus for both real-time sources. (~AWS DMS, CDC, Flink)
- `stream-loader`: sinks raw stream topics directly into MinIO bronze storage and ClickHouse raw landing tables.
- `minio`: object storage for raw bronze files and Trino-managed Iceberg tables. (~S3)
- `appdb`: PostgreSQL app database for platform state, consolidated analytics, and dbt transformations. (~Aurora)
- `clickhouse`: low-latency serving store.
- `trino`: query layer plus batch publication path across PostgreSQL, ClickHouse, and an Iceberg catalog backed by MinIO. (~Athena)
- `lakehouse-init`: one-shot bootstrap that creates ClickHouse and Iceberg objects required by the local stack.
- `catalog-db`: PostgreSQL metadata store for the Trino Iceberg catalog.

## Networks

- `ingress_net`: inbound event traffic into `event-gateway`
- `stream_net`: Redpanda and raw stream landing into MinIO
- `batch_net`: Dagster and third-party data collection
- `lakehouse_net`: MinIO, Trino, and Iceberg catalog services
- `serving_net`: appdb, ClickHouse, and data-serving components

## Storage Wiring

The Compose stack uses three storage patterns:

1. Dedicated named volumes for mutable service state.
2. Read-only bind mounts for config and bootstrap inputs.
3. A small number of host-level read-only mounts for log discovery.

No mutable local storage is shared across services except `dagster_home`, which is intentionally mounted into both Dagster processes so they operate on the same Dagster instance state.

### Named Volumes

| Volume | Service(s) | Mount Path | Purpose |
| --- | --- | --- | --- |
| `redpanda_data` | `redpanda` | `/var/lib/redpanda/data` | Broker log segments and Redpanda local state |
| `minio_data` | `minio` | `/data` | Object storage backing the local bronze and lakehouse buckets |
| `appdb_data` | `appdb` | `/var/lib/postgresql/data` | PostgreSQL data for platform, analytics, dbt, and Grafana database tables |
| `catalog_data` | `catalog-db` | `/var/lib/postgresql/data` | PostgreSQL data for the Iceberg JDBC catalog |
| `clickhouse_data` | `clickhouse` | `/var/lib/clickhouse` | ClickHouse tables, metadata, and MergeTree state |
| `trino_metastore_data` | `trino` | `/var/lib/trino` | Local Hive file-metastore state used by the `hive` catalog |
| `trino_node_data` | `trino` | `/var/trino` | Trino node-local data directory |
| `dagster_home` | `dagster-webserver`, `dagster-daemon` | `/opt/dagster/dagster_home` | Shared Dagster instance state and local artifact storage |
| `prometheus_data` | `prometheus` | `/prometheus` | Prometheus TSDB blocks and WAL |
| `grafana_data` | `grafana` | `/var/lib/grafana` | Grafana local data such as plugins and runtime state |
| `loki_data` | `loki` | `/loki` | Loki chunk, index, and rules filesystem state |
| `tempo_data` | `tempo` | `/var/tempo` | Tempo trace WAL and local trace blocks |

### Read-Only Bind Mounts

These mounts are shared as inputs, not as mutable storage:

- `./appdb/init` is mounted into `appdb` and `appdb-init` at `/docker-entrypoint-initdb.d` so schema and seed SQL can be applied consistently.
- `./catalog-db/init` is mounted into `catalog-db-init` at `/docker-entrypoint-initdb.d` for Iceberg catalog bootstrap SQL.
- `./trino/init` is mounted into `lakehouse-init`, `dagster-webserver`, and `dagster-daemon` so bootstrap SQL and helper scripts are visible in each container.
- Service config files are mounted read-only into their native paths, including Redpanda Connect, OpenTelemetry Collector, Prometheus, Loki, Tempo, Promtail, and Grafana provisioning.
- `./dagster/dagster.yaml` is mounted into `/opt/dagster/dagster_home/dagster.yaml` for both Dagster services so the Dagster instance config always lives under `DAGSTER_HOME`, even when the named volume already exists.

### Host-Level Mounts

- `promtail` mounts `/var/run/docker.sock` read-only for Docker service discovery.
- `promtail` mounts `/var/lib/docker/containers` read-only to tail container JSON logs.

These host mounts are intentionally outside Compose-managed volume lifecycle and are not deleted by cleanup commands.

### Reset Semantics

Use the clean-start script when you want isolated storage and zero residue from prior runs:

```bash
bash scripts/up-clean.sh
```

This script removes Compose containers and named volumes, then recreates the stack:

```bash
docker compose down --volumes --remove-orphans
docker compose up --build --force-recreate -d
```

What gets reset:

- All named volumes listed above.
- All containers in the Compose project.

What does not get reset:

- Files in the repo that are mounted read-only.
- Host Docker logs and Docker socket access used by `promtail`.
- Any external resources outside this Compose project.

## Start The Stack

1. Optionally create a `.env` file if you want to override the default credentials, ports, or image tags.
2. Start the platform from a clean state:

```bash
bash scripts/up-clean.sh
```

This command removes the previous Compose containers and named volumes before recreating the stack, so every stateful service starts with isolated, empty storage.

If you want to reuse the current named volumes instead of resetting them, use the normal startup path:

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

The stack now includes an `appdb-init` one-shot bootstrap that reapplies the SQL bootstrap against existing Postgres volumes. Pulling newer schema changes for the platform and dbt models should no longer require deleting `appdb_data` just to pick up table or view definitions.

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

## Full Test

The platform UI emits tracked interactions through the existing data-platform ingress. Campaign-oriented actions such as `Create New Brief` and campaign-row action buttons emit both analytics and campaign events; navigation, search, and filter actions emit analytics events.

Use a correlation ID that you choose up front, then trace that same value through platform, event-gateway, Redpanda, and ClickHouse. In this repo, that works because platform forwards X-Correlation-ID, event-gateway writes it into logs, Kafka headers, and the JSON payload, and stream-loader lands the raw payload in ClickHouse.

1. Publish one uniquely identifiable event through platform.

TRACE_ID="trace-$(date +%s)"

```bash
curl -si -X POST http://localhost:8081/api/interactions \
  -H 'Content-Type: application/json' \
  -H "X-Correlation-ID: $TRACE_ID" \
  -d "{\"surface\":\"debug\",\"action\":\"trace\",\"subjectType\":\"campaign\",\"subjectId\":\"$TRACE_ID\"}"
```

What to expect:

- HTTP 202
- Response body with ```accepted: true```
- ```correlationId``` equal to your ```TRACE_ID```
- ```published``` containing analytics and email

2. Confirm platform accepted the request and kept the correlation ID.

```bash
docker compose logs --tail=100 platform | grep "$TRACE_ID"
```

You should see the request log for POST /api/interactions with ```correlation_id=$TRACE_ID```.

3. Confirm event-gateway received it and published both events. 

What to look for:

- The http_request log for the inbound request
```analytics_event_accepted```
```email_event_accepted```

4. If you want to inspect the event bus directly, consume the raw topics.

```bash
docker compose exec -T redpanda rpk topic consume analytics_events_raw -n 20
docker compose exec -T redpanda rpk topic consume email_events_raw -n 20
```

Search those outputs for your ```TRACE_ID```. If your local rpk output is noisy, pipe through grep "$TRACE_ID".

5. Check whether ```stream-loader``` is erroring on the sink path.

```bash
docker compose logs --tail=200 stream-loader | grep -Ei 'error|clickhouse|raw_payload'
```

This service is often quiet when healthy, so treat this mainly as an error check, not the primary proof that the message moved.

6. Verify raw landing in ClickHouse first.

```bash
curl -sS -u ${CLICKHOUSE_USER:-default}:${CLICKHOUSE_PASSWORD:-clickhouse} \
  --data-binary "SELECT payload FROM serving.raw_payload WHERE payload LIKE '%$TRACE_ID%' FORMAT TSVRaw" \
  http://localhost:8123/
```

If this returns rows, the event made it through Redpanda Connect into ClickHouse.

7. Verify the parsed table next.

```bash
curl -sS -u ${CLICKHOUSE_USER:-default}:${CLICKHOUSE_PASSWORD:-clickhouse} \
  --data-binary "SELECT event_name, campaign_id, user_id, occurred_at, payload FROM serving.raw_events WHERE campaign_id = '$TRACE_ID' OR payload LIKE '%$TRACE_ID%' ORDER BY occurred_at DESC FORMAT PrettyCompactMonoBlock" \
  http://localhost:8123/
```


What to expect:

- one analytics row with event_name=debug.trace
- one email row may also be present for the same campaign_id

The important caveat: the email publish uses event_type, but the ClickHouse materialized view only extracts event or event_name. So the email-side row can land in serving.raw_events with an empty event_name. For campaign actions, search by campaign_id or payload, not only by event_name.

8. If the trace breaks, interpret it like this:

- Present in ```platform``` logs but missing in event-gateway logs: ```platform``` to ```event-gateway``` forwarding issue.

- Present in ```event-gateway``` logs but missing in serving.raw_payload: ```Redpanda``` or s```tream-loader``` issue.

- Present in ```serving.raw_payload``` but missing or incomplete in serving.```raw_events```: ClickHouse materialized view parsing issue.

## Observability

The stack now includes a local observability baseline in `observability/`:

- `Prometheus` scrapes custom service metrics, Dagster metrics, Redpanda, Redpanda Connect, Trino, ClickHouse, and PostgreSQL exporters.
- `Grafana` provisions Prometheus, Loki, and Tempo datasources automatically, plus starter dashboards for services and pipeline health.
- `Loki` stores container and application logs collected by `promtail`.
- `Tempo` stores traces emitted by the instrumented Go and Python services through the OpenTelemetry Collector.
- `OpenTelemetry Collector` receives OTLP traces from `platform`, `event-gateway`, `mock-third-party-api`, and the Dagster processes.

- `Logging recommendations`: [observability/LOGGING_RECOMMENDATION.md](observability/RECOMMENDATION.md) — guidance for tiered logging, tracing, sampling, and retention at scale.
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

- Raw streaming payloads land directly in MinIO under `bronze/email_events_raw` and `bronze/analytics_events_raw`.
- The same payloads also land directly in ClickHouse table `serving.raw_payload`, and the materialized view in `clickhouse/init/01_mv_parse_raw_payload.sql` keeps `serving.raw_events` current for low-latency reads.
- Dagster indexes MinIO bronze objects in `iceberg.ingress.raw_object_index`; Trino can read raw objects on-demand via the Hive table `hive.ingress.bronze_objects`.
- dbt models live in `dagster/dbt` and run against PostgreSQL appdb as the transformation target.
- Trino is configured with PostgreSQL, ClickHouse, and Iceberg catalogs. It stays on the read path for the platform and on the batch publication path for ClickHouse and Iceberg synchronization.
- The platform app is built from `platform/` as a multi-stage container that embeds the Svelte bundle into the Go binary.
- The platform bootstrap model now comes from the `analytics.platform_bootstrap` SQL view, backed by normalized tables seeded in [appdb/init/02_platform_bootstrap.sql](appdb/init/02_platform_bootstrap.sql) and live aggregate views in [appdb/init/03_platform_bootstrap_views.sql](appdb/init/03_platform_bootstrap_views.sql).
- The current appdb seed can be exported as SQL `INSERT` statements with [appdb/queries/export_platform_seed.sql](appdb/queries/export_platform_seed.sql).

### Hive catalog note

Hive in this stack is the catalog layer for raw bronze files, not the main serving or transformation engine. In practice, the repo creates a Hive schema and an external table named hive.ingress.bronze_objects that points at the MinIO bronze prefix, so Trino can read those raw files on demand without first loading all of them into Iceberg. That wiring lives in trino/init/01_ingress_catalogs.sql and trino/catalog/hive.properties.

The split in this project is: Iceberg stores the metadata-only object index, ClickHouse holds the low-latency serving tables, and Hive is the lightweight external-table mapping over the raw objects in storage. 

This repository configures the `hive` catalog on Trino 468 to use a Hadoop-backed file system (`fs.hadoop.enabled=true`) together
with legacy `hive.s3.*` MinIO settings. Trino 468 does not reliably support mixing the native S3 file system and a local file-based
metastore in the same catalog, so Hadoop mode lets the file metastore read the local catalog directory while the legacy
`hive.s3.*` settings provide access to `s3://` external locations on MinIO. When upgrading Trino, revisit this configuration and
prefer the native S3 file system options (`fs.native-s3.enabled` / `s3.*`) available in newer releases.


## Applying ClickHouse materialized view (local)

To create the Materialized View locally (it will POPULATE existing `serving.raw_payload` rows), run:

```bash
curl -u ${CLICKHOUSE_USER:-default}:${CLICKHOUSE_PASSWORD:-clickhouse} \
  -X POST --data-binary @clickhouse/init/01_mv_parse_raw_payload.sql 'http://localhost:8123/'
```

This will create `serving.mv_raw_payload_to_events` which writes parsed rows into `serving.raw_events`.