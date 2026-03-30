#!/usr/bin/env sh
set -eu

echo "# Harness status"
echo

if command -v git >/dev/null 2>&1; then
  echo "## Branch"
  git rev-parse --abbrev-ref HEAD 2>/dev/null || true
  echo
  echo "## Git status"
  git status --short 2>/dev/null || true
  echo
fi

echo "## Active plans"
if [ -d docs/plans/active ]; then
  find docs/plans/active -maxdepth 1 -type f -name '*.md' | sort
fi
echo

echo "## Recent edited files"
if [ -f .harness/state/edited-files.log ]; then
  tail -n 20 .harness/state/edited-files.log
else
  echo "No tracked edits yet."
fi
echo

echo "## Detected languages"
./scripts/detect-languages.sh || true
