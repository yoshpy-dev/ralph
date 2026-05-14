#!/usr/bin/env sh
# pre-commit-secret-guard.sh - block staged secrets before they enter a commit.
set -eu

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
exec "$REPO_ROOT/scripts/secret-scan.sh" --staged
