#!/usr/bin/env sh
set -eu

mkdir -p .harness/state .harness/logs docs/plans/active docs/plans/archive docs/reports docs/tech-debt

echo "Scaffold bootstrap complete."
echo
echo "Hooks are active via .claude/settings.json (committed to git)."
echo "To add personal permissions or overrides, create .claude/settings.local.json (gitignored)."
echo
echo "Next steps:"
echo "  1. Edit AGENTS.md and CLAUDE.md"
echo "  2. Customize .claude/rules and language packs"
echo "  3. Create a plan with ./scripts/new-feature-plan.sh <slug>"
echo "  4. Run ./scripts/run-verify.sh before claiming a task is done"
