#!/usr/bin/env sh
# Regression guard for org-runtime-retire-loop plan AC-5: zero live
# references to the retired Ralph Loop autonomous execution system
# (scripts/ralph-orchestrator.sh, scripts/ralph-pipeline.sh, the
# RALPH_LOOP_DRIVER env knob, scripts/ralph-cli-driver.sh, ralph-loop.sh,
# scripts/ralph-status-helpers.sh, scripts/new-ralph-plan.sh,
# scripts/build-tui.sh, cmd/ralph-tui, the per-phase RALPH_*_MODEL knobs, and
# the dead `ralph doctor` orchestrator-state check) anywhere in shell, Go,
# TOML, or Markdown sources.
#
# Historical documents (archived plans, spec narrative describing
# pre-retirement state, review/report artifacts, insight-event docs, the
# dated research note in docs/research/) and the insights package's
# historical-schema-vocabulary compat surface (AC-6b: `flow=loop` /
# `source=pipeline` are historical schema values `ralph insights` must keep
# reading, not an active code path) are excluded below. `scripts/xreview-
# helpers.sh` (+ template mirror) and its behavioural test are NOT excluded:
# their provenance comments are worded to name no guarded token, so they
# pass the live pattern on their own merits rather than through an
# exclusion.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

PATTERN='ralph-orchestrator|ralph-pipeline|RALPH_LOOP_DRIVER|ralph-cli-driver|ralph-loop\.sh|ralph-status-helpers|new-ralph-plan|build-tui|ralph-tui|RALPH_(FORCE|IMPLEMENT|SELF_REVIEW|VERIFY|TEST|SYNC_DOCS|PR|PROBE|ESCALATION)_MODEL|checkStaleOrchestratorState'

# Directories/files excluded as historical documentation, not live code:
#   - docs/plans/archive/    : completed plans (historical record)
#   - docs/plans/active/     : plans in flight necessarily describe the
#     retired surface they are removing while their own AC text is live
#   - docs/specs/            : spec narrative sections describing pre-PR
#     state (Background/Current state); FR text itself must stay live-clean
#   - docs/reports/          : self-review/verify/test/sync-docs/triage/
#     walkthrough artifacts (point-in-time record)
#   - docs/insights/         : insights schema docs that document the
#     historical `source:pipeline` / `flow:loop` vocabulary (AC-6b)
#   - docs/tech-debt/README.md : historical RESOLVED annotations
#   - docs/research/approach-comparison.md : dated historical research note,
#     explicitly banner-marked as describing pre-retirement state
#   - internal/insights/     : Go code + test fixtures implementing/testing
#     read-compat for the same historical schema vocabulary (AC-6b)
#   - internal/cli/insights_test.go : regression test pinning that
#     historical-vocab read-compat
#   - internal/cli/upgrade_retired_loop_artifacts_test.go : regression test
#     proving `ralph upgrade`'s remove-path retires these exact historical
#     filenames from a pre-retirement project's manifest (org-debt-batch
#     Slice 5, AC-5) -- it names them as proof of retirement, not a live
#     reference, same rationale as insights_test.go above. Kept in its own
#     single-purpose file so this exclusion stays narrow.
#   - templates/base/docs/insights/ : template mirror of the above
#   - this script itself           : it necessarily names the pattern it checks
#   - .git/
EXCLUDE_REGEX='^(\./)?(templates/base/)?(docs/plans/archive/|docs/plans/active/|docs/specs/|docs/reports/|docs/insights/|docs/tech-debt/README\.md$|docs/research/approach-comparison\.md$|internal/insights/|internal/cli/insights_test\.go$|internal/cli/upgrade_retired_loop_artifacts_test\.go$)|^(\./)?\.git/|^(\./)?tests/test-no-loop-references\.sh$'

# git grep scans tracked files only, so gitignored local state (e.g.
# .claude/agent-memory/, scratch files) cannot produce false FAILs that CI
# would never see.
matches="$(git grep -IlE "$PATTERN" -- '*.sh' '*.go' '*.toml' '*.md' 2>/dev/null | grep -vE "$EXCLUDE_REGEX" || true)"

if [ -n "$matches" ]; then
  echo "FAIL: live references to retired Ralph Loop execution system found:" >&2
  echo "$matches" | sed 's/^/  /' >&2
  exit 1
fi

echo "PASS: no live references to the retired Ralph Loop execution system outside historical documents"
