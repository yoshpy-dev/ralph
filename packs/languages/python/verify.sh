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

if [ ! -f pyproject.toml ] && [ ! -f requirements.txt ] && [ ! -f setup.py ] && [ ! -f tox.ini ]; then
  echo "Skipping Python verifier: no Python project markers found."
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for Python verification."
  exit 1
fi

status=0

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

case "$mode" in
  static) run_static ;;
  test)   run_tests ;;
  all)    run_static; run_tests ;;
esac

exit "$status"
