#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
source "$repo_root/scripts/test-server.sh"
command -v hurl >/dev/null

run_tests() {
    hurl --test --variable "base_url=$CLOUDMUSIC_API_URL" \
        "$repo_root/apps/server/tests/hurl/accounts.hurl"
}

with_test_server hurl run_tests
