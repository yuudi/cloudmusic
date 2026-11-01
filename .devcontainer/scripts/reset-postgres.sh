#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
: "${DATABASE_URL:?Run inside the devcontainer or set DATABASE_URL for cloudmusic_dev}"
export PGCONNECT_TIMEOUT=5

echo 'Resetting the cloudmusic_dev public schema and applying migrations...'
psql "$DATABASE_URL" -X --set=ON_ERROR_STOP=1 --single-transaction \
  --file="$repo_dir/.devcontainer/sql/reset-postgres.sql"
go -C "$repo_dir/apps/server" run ./cmd/migrate up
