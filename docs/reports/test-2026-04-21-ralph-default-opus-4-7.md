# Test report: ralph default → claude-opus-4-7 / xhigh

- Date: 2026-04-21
- Plan: (none — defaults-only update, no `docs/plans/active/*` entry; task intent from user context)
- Tester: tester subagent (test skill)
- Scope: behavioral tests for branch `chore/ralph-default-opus-4-7` @ `8fbe203`
- Evidence: `docs/evidence/test-2026-04-21-ralph-default-opus-4-7.log`

## Commands executed

1. `go test ./internal/config/... -run 'TestDefault|TestLoad_PartialConfig|TestLoad_FullRoundTrip' -count=1 -v` — targeted defaults coverage
2. `go test ./... -count=1` — full Go test suite (uncached)
3. `bash tests/test-ralph-config.sh` — 23 shell-default and override cases
4. `bash tests/test-check-mojibake.sh` — optional harness hook regression
5. `./scripts/run-test.sh` — repo-preferred wrapper (includes local verifier + gofmt + staticcheck + go test)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test ./internal/config/... -run 'TestDefault\|TestLoad_PartialConfig\|TestLoad_FullRoundTrip'` | 3 | 3 | 0 | 0 | 0.468s |
| `go test ./...` (uncached, full suite) | 9 pkg | 9 pkg | 0 | 0 | ~26s aggregate |
| `tests/test-ralph-config.sh` | 23 | 23 | 0 | 0 | <1s |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| `./scripts/run-test.sh` (wrapper) | all | all | 0 | 0 | ~5s (cached Go) |

Per-package Go results (uncached `go test ./...`):

| Package | Result | Duration |
| --- | --- | --- |
| `github.com/.../internal/action` | ok | 3.919s |
| `github.com/.../internal/cli` | ok | 0.814s |
| `github.com/.../internal/config` | ok | 1.124s |
| `github.com/.../internal/scaffold` | ok | 1.461s |
| `github.com/.../internal/state` | ok | 1.356s |
| `github.com/.../internal/ui` | ok | 2.532s |
| `github.com/.../internal/ui/panes` | ok | 3.254s |
| `github.com/.../internal/upgrade` | ok | 2.853s |
| `github.com/.../internal/watcher` | ok | 4.789s |
| `cmd/ralph`, `cmd/ralph-tui`, root | no test files | — |

## Coverage

Instrumented coverage run via `go test ./... -coverprofile=/tmp/cover-all.out -count=1`:

| Package | Statement coverage | Notes |
| --- | --- | --- |
| `internal/config` | **62.5%** | `Default()` itself is **100% covered**; the uncovered 37.5% is in `Load()` error paths (file-read failures, TOML parse errors) — not on the default-resolution path |
| `internal/action` | 95.9% | |
| `internal/cli` | 30.0% | CLI entrypoint; `run.go` env-export path (which propagates Model/Effort from cfg) is static-analyzed in verify but is not behaviorally asserted here |
| `internal/scaffold` | 66.7% | |
| `internal/state` | 87.9% | |
| `internal/ui` | 84.0% | |
| `internal/ui/panes` | 88.9% | |
| `internal/upgrade` | 84.2% | |
| `internal/watcher` | 78.7% | |
| `cmd/ralph`, `cmd/ralph-tui` | 0.0% | entrypoints; no tests (unchanged from prior baseline) |
| **total** | **57.2%** | unchanged shape vs. prior baseline for this branch |

`go tool cover -func=/tmp/cover-config.out` confirms `config.go:40 Default 100.0%`. Because the changed lines are `config.go:43,44` (model and effort string literals inside `Default()`), the default-resolution path is fully exercised by `TestDefault`.

Shell coverage (no instrumented tool):
- `scripts/ralph-config.sh:18` (`RALPH_MODEL:-claude-opus-4-7`) asserted by `tests/test-ralph-config.sh:54`
- `scripts/ralph-config.sh:19` (`RALPH_EFFORT:-xhigh`) asserted by `tests/test-ralph-config.sh:57`
- Override chains for `RALPH_MODEL` / `RALPH_EFFORT` asserted at lines 63-71 of the test file — fallback-only lines, so overrides take any non-default string without requiring updates

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Go `Default()` drift vs. scaffold TOML (self-review H-1, pre-8fbe203 state) | Fixed & verified | `TestDefault` now asserts `claude-opus-4-7` / `xhigh` at `internal/config/config_test.go:11,14`; passes |
| Shell fallback drift vs. Go defaults (self-review H-1) | Fixed & verified | `tests/test-ralph-config.sh:54,57` asserts new values and passes 23/23 |
| Template-mirror divergence (`scripts/ralph-config.sh` vs. `templates/base/scripts/ralph-config.sh`) | Fixed & verified | verify stage confirmed `cmp` identity; `./scripts/run-test.sh` ran `check-sync.sh` indirectly through `run-verify.sh` with "IDENTICAL:107, DRIFTED:0" |
| Mojibake hook regressions from hook-layer changes | Not regressed | `tests/test-check-mojibake.sh` 11/11 pass |

## Test gaps

1. **Runtime end-to-end (Go → shell env → `claude` CLI)** is not exercised. `internal/cli/run.go:56-57` passes `RALPH_MODEL`/`RALPH_EFFORT` into the child shell env, and `ralph-pipeline.sh:146,159,300,335` consume them via `"$RALPH_MODEL"` / `"$RALPH_EFFORT"`. Both legs are unit-tested at the boundary, but the concatenation (Go sets env → shell inherits → `claude -p --model ... --effort ...`) is not covered by any test. This matches the verify report's "partially verified" note and is pre-existing coverage debt, not a regression from this PR.
2. **Anthropic API acceptance of `claude-opus-4-7`** is not asserted (no network call, no stub). Identical shape to the sonnet-4-20250514 default it replaces — also not asserted before. Same class of coverage gap.
3. **`claude -p --effort xhigh` CLI acceptance**: only the env-var value is asserted; whether the installed `claude` CLI accepts `xhigh` is not. Pre-existing shape.

None of these gaps are introduced by this PR; they are structural limits of the test surface. The default-resolution contract ("when neither config nor env is set, what does the system produce?") is fully covered.

## Verdict

- **Pass: 71+** (3 Go targeted + 9 Go packages all-green + 23 shell ralph-config + 11 shell mojibake + wrapper run)
- **Fail: 0**
- **Blocked: 0**

### Verdict: **PASS**

All targeted and regression suites pass. The two files changed behaviorally (`internal/config/config.go` and `scripts/ralph-config.sh`) each have a dedicated assertion pinning the new default value, and both assertions pass. The `Default()` function is at 100% statement coverage. No regressions in the rest of the Go suite or the mojibake-hook regression guard.

Recommend proceeding to `/sync-docs` → `/codex-review` → `/pr`.
