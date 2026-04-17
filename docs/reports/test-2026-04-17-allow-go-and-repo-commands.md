# Test report: allow-go-and-repo-commands

- Date: 2026-04-17
- Plan: `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`
- Tester: `tester` subagent
- Scope: Behavioral verification of the settings-only diff on branch `chore/allow-go-and-repo-commands` (commit `7295c69`). This PR modifies `.claude/settings.json` and `templates/base/.claude/settings.json` only, plus the active plan. Behavioral tests must confirm no regression in the Go test suite, shell tests, syntax of shell scripts, and template-sync gates.
- Evidence: `docs/evidence/test-2026-04-17-allow-go-and-repo-commands.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (`HARNESS_VERIFY_MODE=test`) | n/a (aggregate) | all (delegates to golang verifier) | 0 | 0 | ~6s |
| `go test ./... -count=1` (10 packages) | 160 | 158 | 0 | 2 | ~22s (wall) |
| `gofmt -l .` | 1 | 1 (no diff) | 0 | 0 | <1s |
| `./tests/test-ralph-config.sh` | 23 | 23 | 0 | 0 | <1s |
| `./tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | ~9s |
| `./tests/test-ralph-status.sh` | 40 | 40 | 0 | 0 | ~2s |
| `bash -n scripts/ralph-pipeline.sh` | 1 | 1 | 0 | 0 | <1s |
| `shellcheck -s sh scripts/ralph` | 1 | 0 | 0 | 1 (shellcheck not installed locally) | n/a |
| `./scripts/check-sync.sh` | 1 | 0 (see note) | 1 (expected ROOT_ONLY) | 0 | <1s |
| `./scripts/check-template.sh` | 1 | 1 | 0 | 0 | <1s |
| `./scripts/check-pipeline-sync.sh` | 1 | 1 | 0 | 0 | <1s |
| **Totals** | **231** | **228** | **0 true failures** | **3** | |

Notes on the counts:
- Go package breakdown (all PASS): `internal/action` (coverage 95.9%), `internal/cli` (30.0%), `internal/config` (62.5%), `internal/scaffold` (66.7%), `internal/state` (87.9%), `internal/ui` (84.0%), `internal/ui/panes` (88.9%), `internal/upgrade` (84.2%), `internal/watcher` (79.6%). `cmd/ralph` and `cmd/ralph-tui` have no tests (0.0% coverage, reported as `[no test files]` — unchanged pre-branch baseline).
- Go SKIPs (2): `TestBaseFS_WithMockFS`, `TestAvailablePacks_WithMockFS` in `internal/scaffold`. These are intentional skips gated on mock FS setup, unchanged from main.
- `check-sync.sh` returns non-zero because of a single `ROOT_ONLY` hit on the active plan file (`docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`). The plan's Test plan section explicitly states: "DRIFTED must be 0 (ROOT_ONLY for the active plan is expected)". DRIFTED=0 is satisfied. Treated as **expected and not a real failure**.
- `shellcheck` is an optional tool — `packs/languages/golang/verify.sh` also gates linters via `command -v`. Not installed on this host, so the canary item is recorded as skipped.

## Coverage

- Statement (Go total): **57.3%** (raw `go tool cover -func` total, weighted across all packages with code including the 0%-covered `cmd/*` entrypoints). Per-package coverage is all ≥62.5% for every instrumented package except `internal/cli` (30.0%, pre-existing) and the two `cmd/*` CLI entrypoints.
- Branch: not computed (Go test does not emit branch coverage by default; no tooling configured).
- Function: not computed separately; `go tool cover -func` output is included in the raw evidence log.
- Shell test coverage: 66 assertions across three shell test files (23 + 3 + 40). No shell coverage tooling is configured; coverage is measured by scenario scope (documented in agent memory `coverage_gaps.md`).
- Notes: This PR does not change any Go code or shell scripts, so coverage is unchanged from main. The canary is explicitly about confirming that no existing test regresses.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `./scripts/check-sync.sh` | `FAIL: 1 sync issue(s) found` due to `ROOT_ONLY: docs/plans/active/2026-04-17-allow-go-and-repo-commands.md` | Active plan file exists at repo root under `docs/plans/active/` but not under `templates/base/docs/plans/active/` — this is the expected behavior for in-flight plans (plans are not copied into the template). Matches the plan's own Test-plan note that "DRIFTED must be 0 (ROOT_ONLY for the active plan is expected)." | None required. On plan archival via `/pr`, the file moves to `docs/plans/archive/` and the `ROOT_ONLY` hit disappears. DRIFTED is the gate that matters and it is 0. |

No true test failures were observed. All behavioral tests named in the plan's Test plan section passed.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Go test suite (10 packages, 158 + 2 skip tests) | PASS | `docs/evidence/test-2026-04-17-allow-go-and-repo-commands.log` Step 2 |
| Shell tests (`tests/test-ralph-config.sh`, `tests/test-ralph-signals.sh`, `tests/test-ralph-status.sh`) — 66 assertions | PASS | Log Steps 4–6 |
| Shell script syntax (`bash -n scripts/ralph-pipeline.sh`) | PASS | Log Step 7 |
| Repo formatter (`gofmt -l .`) — zero diff | PASS | Log Step 3 |
| Template parity (`check-template.sh`) | PASS | Log Step 10 |
| Pipeline-order sync (`check-pipeline-sync.sh`) | PASS | Log Step 11 |
| Root↔templates drift (`check-sync.sh` — DRIFTED slot) | PASS (DRIFTED=0) | Log Step 9; ROOT_ONLY=1 is the active plan only, as the plan declared. |

## Test gaps

- **Tool-availability canary**: `shellcheck` was not installed on this host, so the plan's `shellcheck -s sh scripts/ralph` canary could not be executed locally. The plan marked this canary as "if shellcheck available" and the production verifier also gates with `command -v`, so this is a conscious gap rather than a regression. If this canary is needed with proof, it should be run in CI or on a host with `shellcheck` installed.
- **Settings-allow resolution is not testable in isolation**: There is no automated harness that asserts `Bash(<prefix>:*)` tokens are the ones Claude Code actually uses at runtime. The plan accepts this (see Verify plan "保守的代替") and relies on entry-string presence plus JSON validity (covered by `/verify`) and on post-merge runtime canary by the operator.
- **Template-settings parity**: The PR updates both `.claude/settings.json` and `templates/base/.claude/settings.json`. `check-sync.sh` already gates DRIFTED=0, which in turn means those two files match byte-for-byte (they are in the IDENTICAL=105 bucket).

## Verdict

- Pass: YES — all behavioral tests in the plan's Test plan passed. Go test suite (158 PASS, 2 SKIP, 0 FAIL), three shell test files (66/66 assertions), shell syntax check, gofmt, template-parity check, and pipeline-sync check all pass. The only non-zero exit (`check-sync.sh`) is the documented, expected ROOT_ONLY hit on the active plan file, with DRIFTED=0 as required.
- Fail: NO
- Blocked: NO

Recommendation: proceed to `/sync-docs` → `/codex-review` → `/pr`.
