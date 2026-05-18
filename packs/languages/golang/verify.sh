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

if [ ! -f go.mod ]; then
  echo "Skipping Go verifier: go.mod not found."
  exit 0
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required for Go verification."
  exit 1
fi

if [ -z "${GOCACHE:-}" ]; then
  GOCACHE="$PWD/.harness/cache/go-build"
  export GOCACHE
fi
mkdir -p "$GOCACHE"

if [ -z "${STATICCHECK_CACHE:-}" ]; then
  STATICCHECK_CACHE="$PWD/.harness/cache/staticcheck"
  export STATICCHECK_CACHE
fi
mkdir -p "$STATICCHECK_CACHE"

status=0

run_static() {
  # Format check
  unformatted=$(gofmt -l .)
  if [ -n "$unformatted" ]; then
    echo "gofmt: the following files are not formatted:"
    echo "$unformatted"
    status=1
  else
    echo "gofmt: ok"
  fi

  # Vet
  go vet ./... || status=1

  # golangci-lint (optional)
  if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run ./... || status=1
  else
    echo "Skipping golangci-lint: command not found."
  fi

  # staticcheck (optional)
  if command -v staticcheck >/dev/null 2>&1; then
    staticcheck ./... || status=1
  else
    echo "Skipping staticcheck: command not found."
  fi
}

run_tests() {
  go test ./... || status=1
}

case "$mode" in
  static) run_static ;;
  test)   run_tests ;;
  all)    run_static; run_tests ;;
esac

exit "$status"
