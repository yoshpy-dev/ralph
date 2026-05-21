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

roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-rust-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name target \
    \) -prune \) -o \
    -type f -name Cargo.toml -print 2>/dev/null |
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
  echo "Skipping Rust verifier: no Rust project roots found."
  exit 0
fi

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo is required for Rust verification."
  exit 1
fi

run_static() {
  cargo fmt --all --check || status=1
  cargo clippy --all-targets --all-features -- -D warnings || status=1
}

run_tests() {
  cargo test --all-features || status=1
}

verify_root() {
  project_root="$1"
  echo "==> Rust project root: $project_root"

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
