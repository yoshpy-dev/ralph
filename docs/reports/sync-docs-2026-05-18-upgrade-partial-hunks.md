# Sync-docs report: upgrade-partial-hunks

- Plan: `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`
- Related issue: #97
- Verdict: PASS

## Docs Updated

- `docs/specs/2026-04-16-ralph-cli-tool.md`: documented v2 baseline metadata, dry-run diff, pager mode, compact diff UI without hunk/hash metadata, baseline-backed file-level prompt gating, and v1 fallback.
- `docs/tech-debt/README.md`: recorded remaining #97 work for full 3-way merge planner, manual edit, and final pre-apply summary.
- `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`: scoped this PR as the first implementation slice and updated acceptance criteria to match shipped behavior.

## Drift Check

`./scripts/run-verify.sh` passed and included `scripts/check-sync.sh`, `scripts/check-pipeline-sync.sh`, and `scripts/check-skill-sync.sh`.

## Remaining Docs Work

When the follow-up full hunk editor lands, update the same spec section to remove the "follow-up" limitation and add concrete pre-apply summary examples.
