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

roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-python-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name .venv -o \
      -name venv -o \
      -name env -o \
      -name __pycache__ -o \
      -name .mypy_cache -o \
      -name .pytest_cache -o \
      -name .ruff_cache -o \
      -name build -o \
      -name dist \
    \) -prune \) -o \
    -type f \( \
      -name pyproject.toml -o \
      -name requirements.txt -o \
      -name 'requirements-*.txt' -o \
      -name setup.py -o \
      -name tox.ini \
    \) -print 2>/dev/null |
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
  echo "Skipping Python verifier: no Python project roots found."
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for Python verification."
  exit 1
fi

run_static() {
  if command -v ruff >/dev/null 2>&1; then
    ruff check . || status=1
  else
    echo "Skipping ruff: command not found."
  fi

  if command -v mypy >/dev/null 2>&1; then
    mypy . || status=1
  else
    echo "Skipping mypy: command not found."
  fi
}

run_tests() {
  if command -v pytest >/dev/null 2>&1; then
    pytest -q || status=1
  elif python3 -c "import pytest" >/dev/null 2>&1; then
    python3 -m pytest -q || status=1
  else
    echo "Skipping pytest: command not found."
  fi
}

verify_root() {
  project_root="$1"
  echo "==> Python project root: $project_root"

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
