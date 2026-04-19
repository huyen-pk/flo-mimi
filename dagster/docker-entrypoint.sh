#!/usr/bin/env bash
set -euo pipefail

# Ensure DAGSTER_HOME exists
mkdir -p /opt/dagster/dagster_home

# Start the dagster daemon in background (schedules, sensors, run coordinator tasks)
dagster-daemon run &

# Start dagit with the workspace to expose UI and allow in-process run launching
exec dagit -w /opt/dagster/app/workspace.yaml -h 0.0.0.0 -p 3000
