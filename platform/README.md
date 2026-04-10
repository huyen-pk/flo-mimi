# Embedded Platform App

This folder contains an embedded Go + Svelte operator console for the local data platform.

## Architecture

- `cmd/server`: application entrypoint.
- `internal/domain`: typed UI models and interaction command contracts.
- `internal/application`: use cases for bootstrap queries and interaction recording.
- `internal/ports`: abstractions for content loading and event publishing.
- `internal/adapters/content`: PostgreSQL repository that loads the bootstrap payload from the `analytics.platform_bootstrap` SQL view in appdb.
- `internal/adapters/gateway`: HTTP adapter that forwards tracked UI interactions to `event-gateway`.
- `internal/http`: REST API and SPA handler.
- `web`: Svelte SPA bundled with Vite and embedded into the Go binary.

## Local Build

From this folder:

```bash
cd /home/huyenpk/data_engineering/platform
npm --prefix web install --include=dev
npm --prefix web run build
go test ./...
go build ./cmd/server
```

Or build through Docker Compose from the workspace root:

```bash
docker compose build platform
docker compose up -d platform
```

The dummy UI data is seeded through normalized platform tables in [../appdb/init/02_platform_bootstrap.sql](../appdb/init/02_platform_bootstrap.sql), and the bootstrap payload is assembled by live SQL views in [../appdb/init/03_platform_bootstrap_views.sql](../appdb/init/03_platform_bootstrap_views.sql). If your appdb volume already exists, apply both scripts manually:

```bash
docker compose exec -T appdb psql -U analytics -d analytics -f /docker-entrypoint-initdb.d/02_platform_bootstrap.sql
docker compose exec -T appdb psql -U analytics -d analytics -f /docker-entrypoint-initdb.d/03_platform_bootstrap_views.sql
```

To export the current normalized seed data back out as `INSERT` statements, run the query in [../appdb/queries/export_platform_seed.sql](../appdb/queries/export_platform_seed.sql):

```bash
docker compose exec -T appdb psql -U analytics -d analytics -f - < appdb/queries/export_platform_seed.sql
```

## Event Wiring

- Navigation, search, filters, dashboard cards, and telemetry demo actions emit `analytics` events.
- Campaign-oriented actions emit both `analytics` and `email` events through `event-gateway`.
- The backend never talks to Redpanda directly; it stays behind the existing ingress contract and lets the rest of the platform process those events.
- Platform CRUD actions also update appdb-backed campaign and subscriber tables so the bootstrap payload can refresh immediately.
- Dashboard metrics and campaign performance figures are derived from consolidated appdb analytics tables instead of raw operational event rows.