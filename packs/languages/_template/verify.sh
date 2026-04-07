#!/usr/bin/env sh
set -eu

# HARNESS_VERIFY_MODE is set by the caller (run-verify.sh).
# Supported values: static, test, all (default).
mode="${HARNESS_VERIFY_MODE:-all}"

run_static() {
  echo "TODO: Add linters, type checks, and static analysis for __LANGUAGE__."
  return 2
}

run_tests() {
  echo "TODO: Add test runner for __LANGUAGE__."
  return 2
}

case "$mode" in
  static) run_static ;;
  test)   run_tests ;;
  all)    run_static && run_tests ;;
  *)
    echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
    exit 2
    ;;
esac
