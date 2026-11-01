#!/usr/bin/env bash
set -euo pipefail

# Use a fresh schema and remove only that schema on exit.
: "${TEST_DATABASE_URL:?Set TEST_DATABASE_URL to a PostgreSQL test database}"
: "${REDIS_URL:?Set REDIS_URL to a reachable Redis instance}"
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
run_dir=$(mktemp -d)
schema="hurl_$(od -An -N12 -tx1 /dev/urandom | tr -d ' \n')"
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
psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -q -c "CREATE SCHEMA \"$schema\""
created=1
isolated_url=$(TEST_DATABASE_URL="$TEST_DATABASE_URL" python3 - "$schema" <<'PY'
import os
import sys
from urllib.parse import urlsplit, urlunsplit, parse_qsl, urlencode
parts = urlsplit(os.environ["TEST_DATABASE_URL"])
params = [(k, v) for k, v in parse_qsl(parts.query, keep_blank_values=True) if k != 'search_path']
params.append(('search_path', sys.argv[1]))
print(urlunsplit((parts.scheme, parts.netloc, parts.path, urlencode(params), parts.fragment)))
PY
)
port=$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(('127.0.0.1', 0))
print(s.getsockname()[1])
s.close()
PY
)
go -C "$root_dir/apps/server" build -o "$run_dir/migrate" ./cmd/migrate
go -C "$root_dir/apps/server" build -o "$run_dir/server" ./cmd/server
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
hurl --test --variable "base_url=http://127.0.0.1:$port" "$root_dir/apps/server/tests/hurl/accounts.hurl"
