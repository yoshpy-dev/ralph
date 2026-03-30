#!/usr/bin/env sh
set -eu

mkdir -p .harness/state .harness/logs docs/plans/active docs/plans/archive docs/reports docs/tech-debt

if [ ! -f .claude/settings.json ]; then
  cp .claude/settings.minimal.example.json .claude/settings.json
  echo "Created .claude/settings.json from the minimal example."
else
  echo ".claude/settings.json already exists; leaving it unchanged."
fi

echo "Scaffold bootstrap complete."
echo
echo "Next steps:"
echo "  1. Edit AGENTS.md and CLAUDE.md"
echo "  2. Customize .claude/rules and language packs"
echo "  3. Create a plan with ./scripts/new-feature-plan.sh <slug>"
echo "  4. Run ./scripts/run-verify.sh before claiming a task is done"
