#!/usr/bin/env sh
# pre-merge-commit-secret-guard.sh - block staged secrets in automatic merges.
set -eu

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
exec "$REPO_ROOT/scripts/secret-scan.sh" --staged
