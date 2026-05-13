#!/usr/bin/env sh
set -eu

mode="${HARNESS_VERIFY_MODE:-all}"
case "$mode" in
  static|test|all) ;;
  *)
    echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
    exit 2
    ;;
esac

if [ ! -f Cargo.toml ]; then
  echo "Skipping Rust verifier: Cargo.toml not found."
  exit 0
fi

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo is required for Rust verification."
  exit 1
fi

status=0

run_static() {
  cargo fmt --all --check || status=1
  cargo clippy --all-targets --all-features -- -D warnings || status=1
}

run_tests() {
  cargo test --all-features || status=1
}

case "$mode" in
  static) run_static ;;
  test)   run_tests ;;
  all)    run_static; run_tests ;;
esac

exit "$status"
