#!/bin/sh
set -eu

./scripts/wait-for-db.sh "${DB_HOST:-db}" "${DB_PORT:-5432}"

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE:-disable}"

migrate -path ./migrations -database "$DATABASE_URL" up

./server
