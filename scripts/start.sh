#!/bin/sh
set -eu

./scripts/wait-for-db.sh "${DB_HOST:-db}" "${DB_PORT:-5432}"

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE:-disable}"

output=""
status=0
if output=$(migrate -path ./migrations -database "$DATABASE_URL" up 2>&1); then
  status=0
else
  status=$?
fi

if [ "$status" -ne 0 ] && [ "$output" != "no change" ]; then
  printf '%s\n' "$output"
  exit "$status"
fi

./server
