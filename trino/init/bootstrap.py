import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


def execute_trino(sql: str) -> None:
    request = urllib.request.Request(
        url=f"http://{os.environ.get('TRINO_HOST', 'trino')}:{os.environ.get('TRINO_PORT', '8080')}/v1/statement",
        data=sql.encode("utf-8"),
        headers={
            "X-Trino-User": os.environ.get("TRINO_USER", "lakehouse-init"),
            "X-Trino-Source": "lakehouse-init",
        },
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        payload = json.load(response)

    while True:
        if payload.get("error"):
            raise RuntimeError(payload["error"].get("message", "unknown Trino error"))
        next_uri = payload.get("nextUri")
        if not next_uri:
            return
        next_request = urllib.request.Request(
            url=next_uri,
            headers={
                "X-Trino-User": os.environ.get("TRINO_USER", "lakehouse-init"),
                "X-Trino-Source": "lakehouse-init",
            },
            method="GET",
        )
        with urllib.request.urlopen(next_request, timeout=30) as response:
            payload = json.load(response)


def read_sql_statements(sql_path: Path) -> list[str]:
    lines = []
    for line in sql_path.read_text().splitlines():
        if line.strip().startswith("--"):
            continue
        lines.append(line)
    return [statement.strip() for statement in "\n".join(lines).split(";") if statement.strip()]


def execute_clickhouse(sql: str) -> None:
    query = urllib.parse.urlencode(
        {
            "user": os.environ.get("CLICKHOUSE_USER", "default"),
            "password": os.environ.get("CLICKHOUSE_PASSWORD", "clickhouse"),
        }
    )
    request = urllib.request.Request(
        url=f"http://{os.environ.get('CLICKHOUSE_HOST', 'clickhouse')}:{os.environ.get('CLICKHOUSE_PORT', '8123')}/?{query}",
        data=sql.encode("utf-8"),
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        response.read()


def retry(name: str, action, attempts: int = 30, delay: int = 2) -> None:
    last_error = None
    for attempt in range(1, attempts + 1):
        try:
            action()
            return
        except Exception as exc:  # noqa: BLE001
            last_error = exc
            print(f"{name} attempt {attempt}/{attempts} failed: {exc}")
            time.sleep(delay)
    raise RuntimeError(f"{name} failed after {attempts} attempts") from last_error


def main() -> None:
    sql_file = Path(os.environ.get("TRINO_INIT_SQL", "/app/trino-init/01_ingress_catalogs.sql"))
    statements = read_sql_statements(sql_file)
    print(f"Executing {len(statements)} Trino bootstrap statements")
    # Allow configuring longer retry behavior via environment for slow startups
    attempts = int(os.environ.get("TRINO_BOOTSTRAP_ATTEMPTS", "120"))
    delay = int(os.environ.get("TRINO_BOOTSTRAP_DELAY", "3"))
    retry(
        "trino bootstrap",
        lambda: [execute_trino(statement) for statement in statements],
        attempts=attempts,
        delay=delay,
    )

    mv_file = Path(os.environ.get("CLICKHOUSE_INIT_SQL", "/app/clickhouse-init/01_mv_parse_raw_payload.sql"))
    mv_statements = read_sql_statements(mv_file)
    print(f"Executing {len(mv_statements)} ClickHouse bootstrap statements")
    for index, statement in enumerate(mv_statements, start=1):
        retry(
            f"clickhouse materialized view bootstrap statement {index}/{len(mv_statements)}",
            lambda statement=statement: execute_clickhouse(statement),
            attempts=attempts,
            delay=delay,
        )


if __name__ == "__main__":
    main()