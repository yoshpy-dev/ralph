# Sync-docs report: doc drift sweep

- Date: 2026-05-18
- Branch: `docs/sync-docs-drift-audit`
- Scope: current product docs, harness instructions, mirrored skills, templates, quality docs, recipes, and active plan inventory

## Drift found and fixed

| Area | Finding | Fix |
|------|---------|-----|
| Ralph Loop PR strategy docs | `README.md`, root `CLAUDE.md`, and the `/loop` skill still described unified PR creation as the default even though Ralph Loop now defaults to grouped PRs from manifest `pr_groups`, with unified/stacked as explicit strategies. | Updated user-facing examples and workflow text to describe manifest-selected PR strategy, grouped default, and `--pr-strategy unified` as the fallback single-PR command. Mirrored the skill changes across Claude/Codex and root/template copies. |
| Active plan inventory | `docs/plans/active/` still contained two merged/completed upgrade plans from PR #98 and PR #100. | Moved both completed plans to `docs/plans/archive/`; marked `upgrade-partial-hunks` implemented and linked PR #98 before archival. |

## Checks run

| Check | Result | Notes |
|-------|--------|-------|
| `./scripts/check-sync.sh` | Pass | Root/template mirrors in sync. |
| `./scripts/check-skill-sync.sh` | Pass | Claude/Codex skill bodies and metadata in lock-step. |
| `./scripts/check-pipeline-sync.sh` | Pass | Canonical pipeline order references remain aligned. |
| `./scripts/run-static-verify.sh` | Pass | Static mode, changed scope; evidence saved to `docs/evidence/verify-2026-05-18-054558.log`. |
| `./scripts/run-test.sh` | Pass | Test mode, changed scope; evidence saved to `docs/evidence/verify-2026-05-18-054707.log`. |
| Markdown local link audit | Pass | Checked current non-archival markdown files; no missing local links found. |
| `CI=true ./scripts/check-template.sh` | Pass | Template structure check passes when the local-only Git hook installation gate is disabled as it is in CI. |
| `./scripts/check-template.sh` | Local setup gap | Failed only because Git hooks are not installed in this checkout; not a documentation drift finding. |

## Remaining gaps

- No remaining current-doc drift found in this sweep.
- Historical reports and archived plans still preserve the plan paths and command text that were true when they were written; they were treated as evidence records, not mutable current guidance.
