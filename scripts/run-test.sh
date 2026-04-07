#!/usr/bin/env sh
set -eu
HARNESS_VERIFY_MODE=test exec ./scripts/run-verify.sh "$@"
