#!/usr/bin/env bash
# BDD: jenkins deploy requires RABBIT_URL and passes it into pm2
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
JENKINS="$ROOT/scripts/jenkins.sh"

fail=0
assert_contains() {
  local file="$1" needle="$2"
  if ! grep -qF -- "$needle" "$file"; then
    echo "FAIL: $file missing: $needle"
    fail=1
  fi
}

assert_contains "$JENKINS" ': "${RABBIT_URL:?RABBIT_URL must be set in the Jenkins job environment}"'
assert_contains "$JENKINS" 'RABBIT_URL="$RABBIT_URL"'
assert_contains "$JENKINS" 'pm2 start ./mad-news-bot-server'

if [[ "$fail" -ne 0 ]]; then
  echo "jenkins_rabbit_env: FAILED"
  exit 1
fi
echo "jenkins_rabbit_env: PASS"
