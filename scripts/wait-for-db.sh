#!/bin/sh
set -eu

host="$1"
port="$2"

until pg_isready -h "$host" -p "$port" -U "${DB_USER:-postgres}" >/dev/null 2>&1; do
  sleep 1
done
