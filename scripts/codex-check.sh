#!/usr/bin/env sh
set -eu
# Exit 0 = available, Exit 1 = not available
if ! command -v codex >/dev/null 2>&1; then
  echo "codex CLI not found"
  exit 1
fi
if ! codex --version >/dev/null 2>&1; then
  echo "codex CLI not functional"
  exit 1
fi
codex --version
exit 0
