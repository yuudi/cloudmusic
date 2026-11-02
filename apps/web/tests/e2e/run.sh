#!/usr/bin/env bash
set -euo pipefail

: "${TEST_DATABASE_URL:?Set TEST_DATABASE_URL to a PostgreSQL test database}"
: "${REDIS_URL:?Set REDIS_URL to a reachable Redis instance}"
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
run_dir=$(mktemp -d)
schema="web_e2e_$(od -An -N12 -tx1 /dev/urandom | tr -d ' \n')"
created=0
server_pid=''
cleanup() {
    if [[ -n "$server_pid" ]]; then
        kill "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    if [[ "$created" == 1 ]]; then
        psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -q -c "DROP SCHEMA \"$schema\" CASCADE" >/dev/null
    fi
    rm -rf "$run_dir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -q -c "CREATE SCHEMA \"$schema\""
created=1
isolated_url=$(python3 - "$schema" <<'PY'
import os
import sys
from urllib.parse import urlsplit, urlunsplit, parse_qsl, urlencode
parts = urlsplit(os.environ['TEST_DATABASE_URL'])
params = [(k, v) for k, v in parse_qsl(parts.query, keep_blank_values=True) if k != 'search_path']
params.append(('search_path', sys.argv[1]))
print(urlunsplit((parts.scheme, parts.netloc, parts.path, urlencode(params), parts.fragment)))
PY
)
port=$(python3 - <<'PY'
import socket
with socket.socket() as s:
    s.bind(('127.0.0.1', 0))
    print(s.getsockname()[1])
PY
)
go -C "$repo_root/apps/server" build -o "$run_dir/migrate" ./cmd/migrate
go -C "$repo_root/apps/server" build -o "$run_dir/server" ./cmd/server
DATABASE_URL="$isolated_url" "$run_dir/migrate" up >"$run_dir/migrate.log" 2>&1
DATABASE_URL="$isolated_url" HTTP_ADDR="127.0.0.1:$port" "$run_dir/server" >"$run_dir/server.log" 2>&1 &
server_pid=$!
for ((i=0; i<100; i++)); do
    if curl --silent --fail "http://127.0.0.1:$port/api/v1/health/ready" >/dev/null; then break; fi
    if ! kill -0 "$server_pid" 2>/dev/null; then cat "$run_dir/server.log" >&2; exit 1; fi
    sleep 0.1
done
if ! curl --silent --fail "http://127.0.0.1:$port/api/v1/health/ready" >/dev/null; then
    cat "$run_dir/server.log" >&2
    exit 1
fi
cd "$repo_root/apps/web"
# One worker and one browser keep real account creation below the anonymous rate limit.
CLOUDMUSIC_ACCOUNT_E2E=1 CLOUDMUSIC_API_URL="http://127.0.0.1:$port" \
    npm run test:e2e -- e2e/accounts.spec.ts --project=chromium --workers=1
