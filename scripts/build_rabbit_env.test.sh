#!/usr/bin/env bash
# BDD: deploy build.sh requires RABBIT_URL and passes it into pm2
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD="$ROOT/scripts/build.sh"

fail=0
assert_contains() {
  local file="$1" needle="$2"
  if ! grep -qF -- "$needle" "$file"; then
    echo "FAIL: $file missing: $needle"
    fail=1
  fi
}

assert_contains "$BUILD" ': "${RABBIT_URL:?RABBIT_URL must be set in the dplo project environment}"'
assert_contains "$BUILD" 'RABBIT_URL="$RABBIT_URL"'
assert_contains "$BUILD" 'pm2 start ./mad-news-bot-server'

if [[ "$fail" -ne 0 ]]; then
  echo "build_rabbit_env: FAILED"
  exit 1
fi
echo "build_rabbit_env: PASS"
