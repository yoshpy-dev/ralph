# Sync-docs report: upgrade-hunk-apply

- Date: 2026-05-18
- Branch: `feat/99/upgrade-hunk-apply`
- Plan: `docs/plans/active/2026-05-18-upgrade-hunk-apply.md`
- Verdict: PASS

## Updated

- `README.md`: replaced file-level prompt docs with hunk-level prompt, summary timing, keep vs skip semantics, and baseline-missing fallback.
- `docs/specs/2026-04-16-ralph-cli-tool.md`: updated local-edit and manifest v2 contracts for hunk-level partial apply, `disk_hash`, editor behavior, and summary confirmation.
- `docs/tech-debt/README.md`: removed the completed debt item for baseline-backed review being file-level until the hunk editor lands.
- `docs/plans/active/2026-05-18-upgrade-hunk-apply.md`: refreshed progress and evidence references.

## Checks

- `rg` for stale `apply template file` / `keep local file` prompt docs found only archived/old active-plan context outside the current behavior docs.
- `./scripts/run-static-verify.sh` includes `scripts/check-sync.sh`, `scripts/check-pipeline-sync.sh`, and `scripts/check-skill-sync.sh`; all passed in the final verify run.

## Remaining Notes

- The earlier active plan `docs/plans/active/2026-05-18-upgrade-partial-hunks.md` is historical context from the merged first slice and still describes that slice's scope.

