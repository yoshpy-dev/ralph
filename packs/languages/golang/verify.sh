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

repo_root="$PWD"
roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-go-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name .harness -o \
      -name vendor -o \
      -name node_modules -o \
      -name dist -o \
      -name build \
    \) -prune \) -o \
    -type f -name go.mod -print 2>/dev/null |
    while IFS= read -r marker; do
      dirname "$marker"
    done |
    sort -u
}

root_selected() {
  [ -z "${RALPH_VERIFY_PROJECT_ROOTS:-}" ] && return 0
  root="${1#./}"
  [ -n "$root" ] || root="."

  for selected in $RALPH_VERIFY_PROJECT_ROOTS; do
    selected="${selected#./}"
    [ -n "$selected" ] || selected="."
    [ "$root" = "$selected" ] && return 0
  done
  return 1
}

find_project_roots |
  while IFS= read -r project_root; do
    if root_selected "$project_root"; then
      printf '%s\n' "$project_root"
    fi
  done > "$roots_file"

if [ ! -s "$roots_file" ]; then
  echo "Skipping Go verifier: no Go project roots found."
  exit 0
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required for Go verification."
  exit 1
fi

if [ -z "${GOCACHE:-}" ]; then
  GOCACHE="$repo_root/.harness/cache/go-build"
  export GOCACHE
fi
mkdir -p "$GOCACHE"

if [ -z "${STATICCHECK_CACHE:-}" ]; then
  STATICCHECK_CACHE="$repo_root/.harness/cache/staticcheck"
  export STATICCHECK_CACHE
fi
mkdir -p "$STATICCHECK_CACHE"

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

verify_root() {
  project_root="$1"
  echo "==> Go project root: $project_root"

  status=0
  case "$mode" in
    static) run_static ;;
    test)   run_tests ;;
    all)    run_static; run_tests ;;
  esac

  return "$status"
}

overall_status=0
while IFS= read -r project_root; do
  [ -n "$project_root" ] || continue
  if ! (cd "$project_root" && verify_root "$project_root"); then
    overall_status=1
  fi
done < "$roots_file"

exit "$overall_status"
