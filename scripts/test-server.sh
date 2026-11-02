#!/usr/bin/env bash
# Source this file, then call with_test_server PREFIX COMMAND [ARG...].
# See scripts/README.md for the lifecycle, requirements and limitations.

_with_test_server() {
    set -euo pipefail

    if [[ $# -lt 2 || ! "$1" =~ ^[a-z][a-z0-9_]{0,30}$ ]]; then
        echo 'Usage: with_test_server SCHEMA_PREFIX COMMAND [ARG...]' >&2
        exit 2
    fi
    : "${TEST_DATABASE_URL:?Set TEST_DATABASE_URL to a PostgreSQL test database}"
    : "${REDIS_URL:?Set REDIS_URL to a reachable Redis instance}"
    local dependency
    for dependency in go psql curl python3; do
        if ! command -v "$dependency" >/dev/null 2>&1; then
            echo "Required command not found: $dependency" >&2
            exit 1
        fi
    done

    # The worker already runs in its own shell. Keep cleanup state alive after
    # the function returns, including when errexit triggers its EXIT trap.
    created=0 server_pid='' test_pid=''
    repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
    run_dir=$(mktemp -d)

    cleanup() {
        local status=$? cleanup_failed=0
        trap - EXIT INT TERM
        if [[ -n "$test_pid" ]]; then
            kill -TERM -- "-$test_pid" 2>/dev/null || true
            wait "$test_pid" 2>/dev/null || true
        fi
        if [[ -n "$server_pid" ]]; then
            kill -TERM -- "-$server_pid" 2>/dev/null || true
            wait "$server_pid" 2>/dev/null || true
        fi
        if [[ "$created" == 1 ]]; then
            if ! psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 -q \
                -c "DROP SCHEMA \"$schema\" CASCADE" >/dev/null; then
                echo "Failed to remove test schema: $schema" >&2
                cleanup_failed=1
            fi
        fi
        if ! rm -rf -- "$run_dir"; then
            cleanup_failed=1
        fi
        if [[ "$status" == 0 && "$cleanup_failed" == 1 ]]; then
            status=1
        fi
        exit "$status"
    }
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM

    schema="${1}_$(python3 -c 'import secrets; print(secrets.token_hex(12))')"
    shift
    export PGCONNECT_TIMEOUT=5
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

with socket.socket() as sock:
    sock.bind(('127.0.0.1', 0))
    print(sock.getsockname()[1])
PY
)
    go -C "$repo_root/apps/server" build -o "$run_dir/migrate" ./cmd/migrate
    go -C "$repo_root/apps/server" build -o "$run_dir/server" ./cmd/server
    DATABASE_URL="$isolated_url" "$run_dir/migrate" up >"$run_dir/migrate.log" 2>&1 || {
        local status=$?
        cat "$run_dir/migrate.log" >&2
        exit "$status"
    }

    # Give the server and callback separate process groups, including their children.
    set -m
    DATABASE_URL="$isolated_url" HTTP_ADDR="127.0.0.1:$port" \
        "$run_dir/server" >"$run_dir/server.log" 2>&1 &
    server_pid=$!
    export CLOUDMUSIC_API_URL="http://127.0.0.1:$port"
    deadline=$((SECONDS + 10))
    until curl --silent --fail --max-time 1 "$CLOUDMUSIC_API_URL/api/v1/health/ready" >/dev/null; do
        if ! kill -0 "$server_pid" 2>/dev/null || ((SECONDS >= deadline)); then
            echo 'Test server did not become ready.' >&2
            cat "$run_dir/server.log" >&2
            exit 1
        fi
        sleep 0.1
    done
    "$@" &
    test_pid=$!
    wait "$test_pid"
}

with_test_server() {
    # Scope job-control options and restore the caller's signal handlers on return.
    local -
    local saved_traps runner_pid='' status=0
    saved_traps=$(trap -p INT TERM)
    trap '[[ -z "$runner_pid" ]] || kill -INT "$runner_pid" 2>/dev/null || true' INT
    trap '[[ -z "$runner_pid" ]] || kill -TERM "$runner_pid" 2>/dev/null || true' TERM
    set -m
    _with_test_server "$@" &
    runner_pid=$!
    while :; do
        wait "$runner_pid" && status=0 || status=$?
        # A signal interrupts wait before the worker has finished cleaning up.
        kill -0 "$runner_pid" 2>/dev/null || break
    done
    trap - INT TERM
    # Only restore trusted shell-generated trap declarations; callback arguments
    # are executed directly by the worker and never evaluated as shell text.
    eval "$saved_traps"
    return "$status"
}
