#!/usr/bin/env sh
set -eu

if [ ! -f pyproject.toml ] && [ ! -f requirements.txt ] && [ ! -f setup.py ] && [ ! -f tox.ini ]; then
  echo "Skipping Python verifier: no Python project markers found."
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for Python verification."
  exit 1
fi

status=0

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

if command -v pytest >/dev/null 2>&1; then
  pytest -q || status=1
elif python3 -c "import pytest" >/dev/null 2>&1; then
  python3 -m pytest -q || status=1
else
  echo "Skipping pytest: command not found."
fi

exit "$status"
