# Reports

Use this directory for:
- self-review reports (diff quality)
- verify reports (spec compliance + static analysis)
- test reports (behavioral tests)
- sync-docs reports (documentation drift)
- cross-review triage reports
- walkthrough notes

Suggested naming:
- `self-review-YYYY-MM-DD-short-slug.md`
- `verify-YYYY-MM-DD-short-slug.md`
- `test-YYYY-MM-DD-short-slug.md`
- `sync-docs-YYYY-MM-DD-short-slug.md`
- `cross-review-triage-YYYY-MM-DD-short-slug.md`
- `walkthrough-YYYY-MM-DD-short-slug.md`

Reports are intended to reduce handoff ambiguity for both humans and agents.

## Retention

Reports older than 30 days are removed by `scripts/gc-artifacts.sh` (run manually;
dry-run by default — pass `--apply` to delete). Full history is preserved in git
and in merged pull requests.

Evidence logs (`docs/evidence/verify-*.log`) are local-only and gitignored.
`scripts/run-verify.sh` keeps the newest 20 automatically after each run.
Older logs can also be pruned on demand via `scripts/gc-artifacts.sh --apply`.
