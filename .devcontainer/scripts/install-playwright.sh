#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
web_dir="$repo_root/apps/web"

if ! command -v node >/dev/null 2>&1 || [[ ! -f "$web_dir/node_modules/playwright/cli.js" ]]; then
  echo 'Install Node.js and run just init (or npm ci in apps/web) first.' >&2
  exit 1
fi

# Chromium is enough for component tests; pass all three browsers for the full suite.
if [[ $# -eq 0 ]]; then
  set -- chromium
fi

cd -- "$web_dir"
exec node node_modules/playwright/cli.js install --with-deps "$@"
