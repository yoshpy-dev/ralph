# Test report: pr-ready-branch-type-enforcement

- Date: 2026-05-14
- Plan: `docs/plans/archive/2026-05-14-pr-ready-branch-type-enforcement.md`
- Tester: Codex
- Evidence: `docs/evidence/test-2026-05-14-pr-ready-branch-type-enforcement.log` (ignored raw log, local evidence)
- Verdict: **PASS**

## Test execution

| Command | Result | Notes |
| --- | --- | --- |
| `tests/test-branch-name.sh` | PASS | 26/26 assertions. Covers plan-file and manifest branch generation, allowed prefixes, rejected old branch shapes, `--type` metadata generation, and missing `--type` values. |
| `tests/test-ensure-pr-ready.sh` | PASS | 7/7 assertions. Covers Draft PR conversion, already-ready PR no-op, and fail-closed behavior when a PR remains Draft. |
| `tests/test-ralph-signals.sh` | PASS | 3/3 assertions after updating mock plan metadata. |
| `./scripts/run-test.sh > docs/evidence/test-2026-05-14-pr-ready-branch-type-enforcement.log 2>&1` | PASS | Runs test-mode hook checks, new shell regressions, and Go tests. |

## Regression coverage

- Branch generator rejects the previous uncontrolled shapes: `issue-*`, `codex-*`, `integration/*`, and `slice/*`.
- Plan creation scripts write `Type` metadata and produce branch names through the same generator.
- `ensure-pr-ready.sh` calls `gh pr ready` only when `isDraft=true`, then re-reads `isDraft` and fails if the PR is still Draft.
- `scripts/verify.local.sh` now includes the new tests in test mode, so future aggregate test runs cover this behavior.

## Gaps

- No live GitHub network call is made in tests. This is intentional; `gh` is stubbed to keep the regression deterministic. The live check is performed during PR creation by running `scripts/ensure-pr-ready.sh` against the created PR.

## Verdict

All behavioral tests passed.
