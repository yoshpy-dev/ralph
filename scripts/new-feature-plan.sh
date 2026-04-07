#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "Usage: ./scripts/new-feature-plan.sh <short-slug> [issue-number]"
  exit 1
fi

slug="$1"
issue="${2:-N/A}"
date_str="$(date '+%Y-%m-%d')"
target="docs/plans/active/${date_str}-${slug}.md"

if [ -e "$target" ]; then
  echo "Plan already exists: $target"
  exit 1
fi

# Determine branch placeholder
if [ "$issue" = "N/A" ]; then
  branch="TBD"
else
  branch="TBD (issue #${issue})"
fi

mkdir -p docs/plans/active
sed \
  -e "s/__TITLE__/${slug}/g" \
  -e "s/__DATE__/${date_str}/g" \
  -e "s/__REQUEST__/${slug}/g" \
  -e "s/__ISSUE__/${issue}/g" \
  -e "s/__BRANCH__/${branch}/g" \
  docs/plans/templates/feature-plan.md > "$target"

echo "Created $target"
