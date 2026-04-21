# Test report: upgrade-gha-actions-node24

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-upgrade-gha-actions-node24.md`
- Branch: `ci/upgrade-gha-actions-node24`
- Tester: tester subagent (Claude Code)
- Scope: Regression + integration gates for a workflow-only diff (4 files under `.github/workflows/` and `templates/base/.github/workflows/`). No Go source modified.
- Evidence:
  - `docs/evidence/test-2026-04-22-upgrade-gha-actions-node24.log` (index)
  - `docs/evidence/test-2026-04-22-upgrade-gha-actions-node24-gotest.log` (raw `go test -v`)
  - `docs/evidence/test-2026-04-22-upgrade-gha-actions-node24-runverify.log` (raw `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`)
  - `docs/evidence/test-2026-04-22-upgrade-gha-actions-node24-coverage.log` (raw `./scripts/check-coverage.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test ./... -count=1 -v` (9 packages with tests) | 272 RUN / 169 top-level | 169 | 0 | 2 | ~22s wall |
| `tests/test-check-mojibake.sh` (via `run-verify.sh` local verifier) | 11 | 11 | 0 | 0 | <1s |
| `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh` (full gate incl. gofmt + staticcheck + go test) | — | PASS | 0 | 0 | ~10s (cached go) |
| `./scripts/check-coverage.sh` | 0 | — | — | — (skip) | <1s |

All per-package Go suites reported `ok`:

| Package | Duration | Coverage |
| --- | --- | --- |
| `internal/action` | 4.273s | 95.9% |
| `internal/cli` | 1.731s | 32.5% |
| `internal/config` | 2.163s | 62.5% |
| `internal/scaffold` | 1.088s | 66.7% |
| `internal/state` | 0.744s | 87.9% |
| `internal/ui` | 1.826s | 84.0% |
| `internal/ui/panes` | 3.620s | 88.9% |
| `internal/upgrade` | 3.184s | 82.0% |
| `internal/watcher` | 4.128s | 78.2% |
| `cmd/ralph`, `cmd/ralph-tui`, repo root | — | no test files |

## Coverage

- Statement (go tool cover -func total): **57.6%** — includes `cmd/ralph` and `cmd/ralph-tui` entrypoints at 0.0% which dilute the aggregate.
- Per-package statement coverage (Go): see table above. Core refactor-sensitive packages (`internal/action`, `internal/state`, `internal/upgrade`, `internal/ui/panes`, `internal/ui`, `internal/watcher`) all sit **≥ 78%**, with several ≥ 85%.
- `check-coverage.sh` gate: **not enforced** — `COVERAGE_THRESHOLD=80` is respected only when `packs/languages/<lang>/coverage.sh` exists; the Go pack currently ships no `coverage.sh`. This is pre-existing scaffolding state, not a regression introduced by this PR. Recorded as a standing coverage blind spot (see "Test gaps" below).
- Branch / function coverage: not collected (the project does not instrument either; `go test -cover` reports statements only).

### Scope relevance note

This PR changes only `.github/workflows/*.yml` + `templates/base/.github/workflows/verify.yml`. These files are not compiled into Go binaries and have no Go code paths, so coverage numbers are **baseline-equivalent** to `main` and are reported here only to confirm no regression.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Go build / unit tests unaffected by workflow edits | PASS (all 9 packages `ok`, 0 FAIL) | `test-...-gotest.log`, `run-verify.sh` golang verifier output |
| Static analysis gate still green (gofmt + staticcheck) | PASS | `run-verify.sh` shows `gofmt: ok`, `0 issues.` |
| Hook unit suite (`check_mojibake.sh`) still green | PASS (11/11) | local verifier block in `run-verify.sh` output |
| Pre-existing SKIPs still skipping (not newly failing) | Confirmed: `TestBaseFS_WithMockFS`, `TestAvailablePacks_WithMockFS` remain SKIP | `test-...-gotest.log` |

## Test gaps

### Intentionally out of scope (recorded in plan)

- **goreleaser-action v6 → v7 real-tag behavior**: The `release.yml` path only fires on tag push. `workflow_dispatch` / snapshot dry-run are explicitly deferred to a follow-up PR (plan "Open questions / Follow-up"). This cannot be exercised by `/test` — the new action stack will run for the first time on the next `v*` tag.
- **Unit tests for workflow YAML**: Not applicable — the repository has no YAML-contract test harness, and the plan's Test plan declares unit tests out of scope.

### Standing infra blind spots (pre-existing, not introduced by this PR)

- `scripts/check-coverage.sh` is a no-op because no `packs/languages/*/coverage.sh` ships. The 80% threshold is declared but not enforced. Tracked in tester memory (`coverage_gaps.md`).
- `cmd/ralph` and `cmd/ralph-tui` have no tests (entrypoints only).

### Unverified items for this PR

- CI-side execution of the updated workflows on GitHub Actions runners (verify.yml, check-template.yml). The new SHAs resolve locally via `grep` but cannot be run outside GitHub's runner environment.
- `goreleaser --clean` invocation under `goreleaser-action@v7.1.0` with `version: "~> v2"` — deferred to post-merge tag push.

## Verdict

- Pass: **YES**
- Fail: no
- Blocked: no

All in-scope suites (`go test ./...`, `./scripts/run-verify.sh`, mojibake hook tests) pass with zero failures and zero new skips. Coverage is unchanged from baseline because no Go source was touched. `/test` gate is satisfied; ready to proceed to `/sync-docs` and `/pr`.

Note: the two unverified items above are CI-side behaviors that cannot be exercised locally. They are documented in the plan's Open questions / Follow-up and must be watched on the PR's CI run (verify + check-template) and on the next release tag push (goreleaser v7 real execution).
