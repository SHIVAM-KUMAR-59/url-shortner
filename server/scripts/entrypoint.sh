#!/usr/bin/env bash
set -euo pipefail

echo "Running database migrations..."
./goose -dir ./migrations postgres "$DATABASE_URL" up

echo "Migrations complete. Starting server..."
exec ./server
