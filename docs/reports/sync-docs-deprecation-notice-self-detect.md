# Sync-docs report: deprecation-notice-self-detect

- Date: 2026-07-24
- Plan: docs/plans/active/2026-07-24-deprecation-notice-self-detect.md
- Agent: doc-maintainer subagent
- Commit under review: 46d8ff0 (fix: exclude self from Go CLI detection in shell deprecation notice)
- Base: ef12eaae (main)

## Summary

Commit 46d8ff0 changes the deprecation notice logic in `scripts/ralph` (and its byte-identical mirror `templates/base/scripts/ralph`) to suppress the notice when `command -v ralph` resolves to the script itself via a `-ef` inode comparison. A self-detection regression test (case 5, 3 assertions) was added to `tests/test-ralph-deprecation-notice.sh`.

## Drift check results

### AGENTS.md — scripts/ description (line 82)

Current text: "legacy `ralph` shell CLI (prints deprecation notice when Go `ralph` binary is on PATH; retirement tracked in docs/tech-debt/README.md)"

**No drift. No update required.**

The /verify report (V2) already assessed this as accurate at the map level. The behavior described — "prints deprecation notice when Go `ralph` binary is on PATH" — remains true in the normal case (a foreign ralph binary is on PATH). The fix adds a self-exclusion edge case that is intentionally omitted from the brief map entry. Adding it would add noise without benefit to AGENTS.md's role as a short, stable map.

### docs/tech-debt/README.md — Legacy shell CLI retirement entry

Previous Impact text: "Every shell-CLI session prints a deprecation notice; ..."

**Drift detected. Updated.**

After fix #134, the notice is suppressed when `scripts/` is on PATH and `ralph` resolves to the script itself. "Every shell-CLI session" was no longer accurate. The Impact text was narrowed to: "Prints a deprecation notice when a different `ralph` binary is on PATH (suppressed when `scripts/` itself is on PATH via inode self-check, fix #134)". The Related plan/report column was also updated to include the active plan for traceability.

### docs/plans/active/2026-07-24-deprecation-notice-self-detect.md — Progress checklist

**Two checklist items were not yet marked done.**

Updated:
- `[ ] Review artifact created` → `[x] Review artifact created (docs/reports/self-review-deprecation-notice-self-detect.md)`
- `[ ] Verification artifact created` → `[x] Verification artifact created (docs/reports/verify-deprecation-notice-self-detect.md)`

(Test artifact was already marked done.)

### docs/plans/archive/2026-07-13-dual-cli-consolidation.md

AC4 states "deprecation notice appears on stderr when a `ralph` binary is on PATH, is absent otherwise". This is a completed and archived plan; its acceptance criteria describe the behavior at the time of that plan. No update needed — archived plans are historical records.

### .claude/rules/ and docs/recipes/

No references to `RALPH_NO_DEPRECATION` or the deprecation notice behavior found. No updates needed.

### README.md and CLAUDE.md

No references to the deprecation notice behavior. No updates needed.

## Files changed

| File | Change |
|------|--------|
| `docs/plans/active/2026-07-24-deprecation-notice-self-detect.md` | Marked "Review artifact created" and "Verification artifact created" checklist items as done |
| `docs/tech-debt/README.md` | Narrowed Impact text for legacy shell CLI retirement entry to reflect self-exclusion behavior; added active plan to Related column |
| `docs/reports/sync-docs-deprecation-notice-self-detect.md` | This report |

## Files with no change needed

| File | Reason |
|------|--------|
| `AGENTS.md` | Map-level description accurate; self-exclusion detail intentionally omitted per /verify V2 |
| `docs/plans/archive/2026-07-13-dual-cli-consolidation.md` | Archived plan; historical record, not updated |
| `.claude/rules/*.md` | No deprecation notice references |
| `docs/recipes/*.md` | No deprecation notice references |
| `README.md` | No deprecation notice references |
| `CLAUDE.md` | No deprecation notice references |

## Verdict

Documentation is in sync. No blocking drift. Proceed to `/pr`.
