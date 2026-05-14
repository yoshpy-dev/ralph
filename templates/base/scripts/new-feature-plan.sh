#!/usr/bin/env sh
set -eu

type="feat"

if [ "${1:-}" = "--type" ]; then
  if [ "${2:-}" = "" ]; then
    echo "Usage: ./scripts/new-feature-plan.sh [--type <type>] <short-slug> [issue-number]"
    echo "Allowed types: $(./scripts/branch-name.sh allowed-types)"
    exit 1
  fi
  type="${2:-}"
  shift 2
fi

if [ "${1:-}" = "" ] || [ -z "$type" ]; then
  echo "Usage: ./scripts/new-feature-plan.sh [--type <type>] <short-slug> [issue-number]"
  echo "Allowed types: $(./scripts/branch-name.sh allowed-types)"
  exit 1
fi

slug="$1"
issue="${2:-N/A}"
date_str="$(date '+%Y-%m-%d')"
target="docs/plans/active/${date_str}-${slug}.md"

if ! ./scripts/branch-name.sh validate "${type}/placeholder" >/dev/null 2>&1; then
  echo "Error: invalid branch type: ${type}"
  echo "Allowed types: $(./scripts/branch-name.sh allowed-types)"
  exit 1
fi

if [ -e "$target" ]; then
  echo "Plan already exists: $target"
  exit 1
fi

mkdir -p docs/plans/active
sed \
  -e "s#__TITLE__#${slug}#g" \
  -e "s#__DATE__#${date_str}#g" \
  -e "s#__REQUEST__#${slug}#g" \
  -e "s#__ISSUE__#${issue}#g" \
  -e "s#__TYPE__#${type}#g" \
  docs/plans/templates/feature-plan.md > "$target"

echo "Created $target"
