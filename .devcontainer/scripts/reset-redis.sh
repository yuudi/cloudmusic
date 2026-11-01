#!/usr/bin/env bash
set -euo pipefail

: "${REDIS_URL:?Run inside the devcontainer or set a local REDIS_URL}"
case "$REDIS_URL" in
  redis://redis:6379/0|redis://localhost:6379/0|redis://127.0.0.1:6379/0) ;;
  *) echo 'Reset is only allowed for local development Redis database 0.' >&2; exit 1 ;;
esac

echo 'Clearing local Redis database 0...'
timeout 5s redis-cli -e -u "$REDIS_URL" --no-auth-warning FLUSHDB
