#!/usr/bin/env sh
set -eu
: "${RALPH_VERIFY_SCOPE:=changed}"
export RALPH_VERIFY_SCOPE
HARNESS_VERIFY_MODE=test exec ./scripts/run-verify.sh "$@"
