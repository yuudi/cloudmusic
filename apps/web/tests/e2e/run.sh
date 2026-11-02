#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
source "$repo_root/scripts/test-server.sh"
command -v npm >/dev/null

run_tests() {
    cd "$repo_root/apps/web"
    # One worker and one browser keep account creation below the anonymous rate limit.
    CLOUDMUSIC_ACCOUNT_E2E=1 \
        npm run test:e2e -- e2e/accounts.spec.ts --project=chromium --workers=1
}

with_test_server web_e2e run_tests
