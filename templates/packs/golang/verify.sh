#!/usr/bin/env sh
set -eu

if [ ! -f go.mod ]; then
  echo "Skipping Go verifier: go.mod not found."
  exit 0
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required for Go verification."
  exit 1
fi

status=0

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

# Tests
go test ./... || status=1

exit "$status"
