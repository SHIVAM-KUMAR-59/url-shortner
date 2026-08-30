#!/usr/bin/env bash

set -euo pipefail

echo "Running ClickHouse migrations..."

./goose \
  -dir ./migrations/clickhouse_migrations \
  clickhouse \
  "clickhouse://${CLICKHOUSE_USER}:${CLICKHOUSE_PASSWORD}@${CLICKHOUSE_ADDR}/default" \
  up

echo "ClickHouse migrations complete."
echo "Starting consumer..."

exec ./consumer