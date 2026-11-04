#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
source "$repo_root/scripts/test-server.sh"
command -v hurl >/dev/null

run_tests() {
    hurl --test --variable "base_url=$CLOUDMUSIC_API_URL" \
        "$1"
}

shopt -s nullglob
scenarios=("$repo_root/apps/server/tests/hurl/"*.hurl)
if (( ${#scenarios[@]} == 0 )); then
    echo 'No API E2E scenarios found.' >&2
    exit 1
fi

for scenario in "${scenarios[@]}"; do
    echo "Running API E2E: ${scenario##*/}"
    # Each scenario gets fresh data and a fresh process-local rate limiter.
    with_test_server hurl run_tests "$scenario"
done
