#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
bash "$script_dir/reset-postgres.sh"
bash "$script_dir/reset-redis.sh"
