#!/usr/bin/env sh
set -eu
HARNESS_VERIFY_MODE=static exec ./scripts/run-verify.sh "$@"
