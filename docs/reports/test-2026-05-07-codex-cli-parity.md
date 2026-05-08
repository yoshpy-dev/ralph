# Test report: Codex CLI 標準フローパリティ

- Date: 2026-05-07
- Plan: `docs/plans/active/2026-05-07-codex-cli-parity.md`
- Branch: `feat/codex-cli-parity`
- Tester: tester subagent (Claude Opus 4.7, 1M context)
- Scope: behavioural test gates required by `/test` for the Codex CLI parity PR
  (Slices 1–7 already merged into the working branch). Static analysis is out
  of scope (covered by `/verify`, see
  `docs/reports/verify-2026-05-07-codex-cli-parity.md`).
- Evidence:
  - `docs/evidence/test-2026-05-07-codex-cli-parity.log` (`./scripts/run-test.sh`)
  - `docs/evidence/test-2026-05-07-codex-cli-parity-go-verbose.log` (`go test ./... -count=1 -v`)
  - `docs/evidence/verify-2026-05-07-093216.log` (auto-emitted by run-verify.sh during run-test.sh)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (aggregate) | — | OK | 0 | 2 (Go mock-FS) | ~25 s wall |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | < 1 s |
| `tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | < 1 s |
| `tests/test-ralph-config.sh` | 27 | 27 | 0 | 0 | < 1 s |
| `tests/test-ralph-status.sh` | 40 | 40 | 0 | 0 | < 1 s |
| `tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | ~2 s |
| `go test ./... -count=1 -v` (top-level cases) | 218 | 216 | 0 | 2 | 23.4 s sum |
| `go test ./... -count=1 -v` (incl. subtests, `=== RUN`) | 321 | 319 | 0 | 2 | — |

Per-package Go totals (all `ok`, no cache because `-count=1`):

| Package | Wall | Result |
| --- | --- | --- |
| `internal/action` | 4.24 s | ok |
| `internal/cli` | 1.24 s | ok |
| `internal/config` | 0.98 s | ok |
| `internal/scaffold` | 1.33 s | ok |
| `internal/state` | 1.69 s | ok |
| `internal/ui` | 3.61 s | ok |
| `internal/ui/panes` | 3.53 s | ok |
| `internal/upgrade` | 2.46 s | ok |
| `internal/watcher` | 4.33 s | ok |
| `cmd/ralph`, `cmd/ralph-tui`, `github.com/yoshpy-dev/ralph` (root) | — | no test files |

### New tests added by this PR (all green)

| Test | Package / file | Plan AC mapping |
| --- | --- | --- |
| `tests/test-check-skill-sync.sh` (6 fixtures: parity, inventory drift, body drift, description drift, policy drift claude-only-forbid, policy parity both-forbid) | `tests/` | AC-3, R-11 |
| `TestExecuteInit_RendersCodexSurfaces` | `internal/cli/cli_test.go` | AC-1 (three-tier layout: `.claude/`, `.codex/`, `.agents/skills/`) |
| `TestCheckCodexEffectiveConfig_MissingFile` | `internal/cli/cli_test.go` | AC-1b, AC-6 (warn, not fail) |
| `TestCheckCodexEffectiveConfig_MissingFeatureFlag_Warns` | `internal/cli/cli_test.go` | AC-1b (R-13) |
| `TestCheckCodexEffectiveConfig_NoHooks_Warns` | `internal/cli/cli_test.go` | AC-1b (R-13) |
| `TestCheckCodexEffectiveConfig_FullyWired` | `internal/cli/cli_test.go` | AC-1b (positive path) |
| `TestCheckCodexEffectiveConfig_InvalidTOML_Fails` | `internal/cli/cli_test.go` | AC-1b (error path) |
| `TestLoad_RequireCodexCLI` | `internal/config/config_test.go` | AC-6 (`[doctor] require_codex_cli` parsing) |
| `TestLoad_DefaultConfig` extension | `internal/config/config_test.go` | AC-6 (default `false`) |
| `TestTemplateBaseCodexAssetsExist` | `internal/scaffold/embed_test.go` | AC-1, AC-8 (go:embed coverage of `.codex/` + `.agents/skills/`) |

All ten new tests recorded in the evidence log emitted `--- PASS` on this run.

## Coverage

- Instrumented coverage was not collected on this run (the project does not
  require `-coverprofile` in `./scripts/run-test.sh`; it has not been part of
  the standard gate). Recommendation captured under "Test gaps".
- Test-case coverage of the plan's Test plan section:
  - Unit tests (plan §Test plan / Unit tests): 5/5 areas covered
    (`embed_test.go`, `cli_test.go::TestExecuteInit_*`, `cli_test.go::TestCheckCodexEffectiveConfig_*`,
    `config_test.go`, upgrade rename behaviour).
  - Integration tests: 4/4 covered (skill-sync standalone, three drift
    fixtures, body-vs-frontmatter normalisation, policy-parity check). Note
    that `tests/upgrade_downgrade_test.go` proposed in the plan is still not
    landed — see Test gaps.
  - Regression tests: 3/3 passing (existing
    `internal/scaffold/embed_test.go` `.claude/` cases, existing
    `internal/cli/cli_test.go` cases, `./scripts/run-verify.sh` aggregate).
  - Edge cases: 5/9 explicitly covered by automated tests
    (missing-file/missing-flag/no-hooks/invalid-TOML for effective config; body
    + name + description + policy drift fixtures; default config). 4/9 are
    covered structurally (e.g. AGENTS.md size enforced by `/verify`, codex CLI
    absent path probed via warning-only design) or are deferred to the manual
    smoke test (project-trust UX).

## Failure analysis

No failures.

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Mojibake guard against U+FFFD in Edit/Write/MultiEdit payloads | PASS (11/11) | mojibake suite in run-test.log |
| Ralph signal cleanup (no orphan processes after SIGINT, orchestrator status flips to `interrupted`) | PASS (3/3) | signals suite output |
| Ralph status table + JSON rendering with no-color and missing-state branches | PASS (40/40) | status suite output |
| Numeric env-var validation for `RALPH_*` knobs (incl. `RALPH_STANDARD_MAX_PIPELINE_CYCLES`) | PASS (27/27) | config suite output |
| Existing `internal/scaffold/embed_test.go` `.claude/` + `CLAUDE.md` assertions still green after `.codex/` + `.agents/skills/` additions | PASS | `TestEmbedFS_*` cases in go verbose log |
| `internal/upgrade/` hash-based diff engine (rename surfaces as add+remove) | PASS (all 35+ TestComputeDiffs_/TestUnifiedDiff_ cases) | upgrade package output |

## Skipped / deferred tests

| Test | Reason | Where it is tracked |
| --- | --- | --- |
| `internal/scaffold::TestBaseFS_WithMockFS` | Pre-existing skip — exercises a mock FS path that is not part of the production embed flow. Not introduced by this PR. | Skip reason in source; carried over from prior runs. |
| `internal/scaffold::TestAvailablePacks_WithMockFS` | Same as above. | Skip reason in source. |
| Live Codex CLI smoke test (`$spec` → `$plan` → `$work` → post-impl pipeline → `$pr` end-to-end on a Codex agent) | Plan calls this out as a manual sign-off (Slice 7, AC-2). Not automatable on CI because `codex` CLI is not present in the verification environment, project trust is interactive, and the run mutates a real PR. | Recorded as a known gap in `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md`. R-8 in the plan. |
| `tests/upgrade_downgrade_test.go` (new→old→new manifest round-trip) | Proposed in the plan (Slice 7 / Test plan §Integration tests) but not landed in this branch. The hash-engine round-trip is partially exercised by existing `TestComputeDiffs_*` cases (rename-as-add+remove, unmanaged carry-through), but a full downgrade fixture is still missing. | Test gap below. R-7 in the plan. |
| Codex Loop driver work | Out of scope for this PR — tracked in yoshpy-dev/ralph#44. The string-only rename ripple in `scripts/ralph-pipeline.sh` / `scripts/ralph-orchestrator.sh` / `scripts/check-pipeline-sync.sh` was verified by `./scripts/check-pipeline-sync.sh` during `/verify`, not here. | Issue #44; plan Non-goals. |

## Test gaps

1. **Live Codex smoke test deferred** — only the static surfaces (`.codex/*`,
   `.agents/skills/*`, doctor effective-config probe) and the drift gate are
   automated. End-to-end Codex agent runs depend on operator sign-off recorded
   in `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md`. This is
   acceptable per AC-2 / R-8 but should be promoted to an opt-in CI job once
   `codex exec` becomes available in the runner image.
2. **Upgrade round-trip fixture missing** — the plan calls for
   `tests/upgrade_downgrade_test.go` to confirm new→old→new manifest stability
   when `codex-review/` ⇄ `cross-review/` rename participates. Existing
   `internal/upgrade/TestComputeDiffs_*` cases approximate this but do not
   exercise a real downgrade. Recommend a follow-up to land the round-trip
   fixture before the next ralph release tag.
3. **No instrumented coverage numbers** — `./scripts/run-test.sh` does not
   collect `-coverprofile`. For a PR that adds a new CLI surface it would be
   worth a one-shot
   `go test ./internal/cli/... ./internal/config/... ./internal/scaffold/... -coverprofile=coverage.out`
   to confirm the new lines are exercised. Not a release blocker.
4. **Codex CLI absence path** — `TestCheckCodexEffectiveConfig_MissingFile`
   covers the file-not-found branch, but there is no test that simulates
   `codex` binary missing while a `.codex/config.toml` is present. The current
   doctor implementation is warning-only by design (R-8), so this is a
   coverage gap rather than a behaviour gap.

## Verdict

- **Pass: yes.** All gated suites are green: 11 mojibake + 6 skill-sync + 27
  ralph-config + 40 ralph-status + 3 ralph-signals + 218 Go top-level (319
  including subtests). 0 failures, 2 pre-existing Go skips, 1 deferred
  manual smoke test (out-of-scope per plan).
- **Fail: no.**
- **Blocked: no.**
- **Recommendation:** proceed to `/sync-docs` → `/cross-review` → `/pr`. The
  three test gaps above are tracked here and in the plan's Risks; none of
  them block the merge per the acceptance criteria.

---

## Cycle 2 (cap-reached) re-test

- Date: 2026-05-07
- Cycle: 2/2 — `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`. Cap is reached after this run.
- Tester: `tester` subagent (Claude Opus 4.7, 1M context)
- Scope: cycle-2 delta only — commits `3abd1d7` (cross-review ACTION_REQUIRED fix — `probeBinary`, default Codex hooks, bidirectional triage template) and `79d7a73` (self-review CRITICAL fix — repaired hook paths in `templates/base/.codex/config.toml`, removed misapplied `^git commit` PostToolUse wiring, added `probeBinary` 5s timeout / first-non-empty-line return / symmetric LookPath+run error handling).
- Evidence:
  - `docs/evidence/test-2026-05-07-codex-cli-parity-pass2.log` (`./scripts/run-test.sh`)
  - `docs/evidence/test-2026-05-07-codex-cli-parity-pass2-go-verbose.log` (`go test ./... -count=1 -v`)
  - `docs/evidence/verify-2026-05-07-100810.log` (auto-emitted by `run-verify.sh` during `run-test.sh` rerun)
- Companion artifact: cycle-2 verify section in `docs/reports/verify-2026-05-07-codex-cli-parity.md` (Pass).

### Cycle-2 test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (aggregate, exit 0) | — | OK | 0 | 2 (Go mock-FS) | ~25 s wall |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | < 1 s |
| `tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | < 1 s |
| `go test ./... -count=1 -v` (top-level cases) | 222 | 220 | 0 | 2 | 24.5 s sum |
| `go test ./... -count=1 -v` (incl. subtests, `=== RUN`) | 323 | 321 | 0 | 2 | — |

Per-package Go totals (all `ok`, with `-count=1` so cycle-2 source compiled fresh):

| Package | Top-level tests | Wall | Result |
| --- | --- | --- | --- |
| `internal/action` | 9 | 4.64 s | ok |
| `internal/cli` | **31** (was 29 in cycle 1; +2 from `TestProbeBinary_*`) | 3.91 s | ok |
| `internal/config` | 5 | 1.20 s | ok |
| `internal/scaffold` | 14 | 2.27 s | ok |
| `internal/state` | 12 | 1.55 s | ok |
| `internal/ui` | 41 | 2.95 s | ok |
| `internal/ui/panes` | 57 | 4.04 s | ok |
| `internal/upgrade` | 39 | 3.04 s | ok |
| `internal/watcher` | 14 | 4.89 s | ok |

### Targeted verification of cycle-2 surfaces (per request)

All requested cases ran in a single `go test ./internal/cli -run "TestProbeBinary|TestCheckCodexEffectiveConfig|TestExecuteInit_RendersCodexSurfaces" -v -count=1` invocation (exit 0, package `ok` 0.403 s):

| Test | Cycle-1 status | Cycle-2 status | Notes |
| --- | --- | --- | --- |
| `TestProbeBinary_BrokenShimFails` | n/a (added cycle 2) | **PASS** (0.13 s) | New cycle-2 test. Confirms a binary that exists on PATH but fails `--version` is reported as fail (symmetric error handling — same shape as missing binary). |
| `TestProbeBinary_MissingBinary` | n/a (added cycle 2) | **PASS** (0.00 s) | New cycle-2 test. Confirms `LookPath` failure surfaces as fail with consistent error message. |
| `TestCheckCodexEffectiveConfig_MissingFile` | PASS | **PASS** | Unchanged behaviour. Warn-only when `.codex/config.toml` absent. |
| `TestCheckCodexEffectiveConfig_MissingFeatureFlag_Warns` | PASS | **PASS** | Unchanged. `[features] codex_hooks = true` absent → warn. |
| `TestCheckCodexEffectiveConfig_NoHooks_Warns` | PASS | **PASS** | Unchanged. Empty `[hooks.*]` → warn. |
| `TestCheckCodexEffectiveConfig_FullyWired` | PASS | **PASS** | Unchanged. Cycle-2 default-on hooks (`Edit` + `Write` PreToolUse referencing `.claude/hooks/check_mojibake.sh`) still produce `pass — codex_hooks=true, 2 hook entry(ies). Confirm 'codex trust .' ran for this project`. |
| `TestCheckCodexEffectiveConfig_InvalidTOML_Fails` | PASS | **PASS** | Unchanged. Malformed TOML still surfaces as fail. |
| `TestExecuteInit_RendersCodexSurfaces` | PASS | **PASS** (0.03 s) | Re-run to confirm `79d7a73`'s `templates/base/.codex/config.toml` edit (hook content change, layout unchanged) does not break the three-tier scaffold (`base 9 files / pack/golang 2 files / .ralph/manifest.toml`). |

All eight cases PASS under cycle-2 source. No regressions.

### New tests since cycle 1 (delta)

| Test | Package / file | Plan AC mapping | Cycle-2 status |
| --- | --- | --- | --- |
| `TestProbeBinary_BrokenShimFails` | `internal/cli/cli_test.go` | AC-6 (CLI detection now exercises actual `--version` invocation, not just `LookPath`) | PASS |
| `TestProbeBinary_MissingBinary` | `internal/cli/cli_test.go` | AC-6 (warn-only path when binary absent) | PASS |

These two cases close the cycle-1 test gap #4 ("Codex CLI absence path … there is no test that simulates `codex` binary missing while a `.codex/config.toml` is present"). The `MissingBinary` case directly exercises the absence path; the `BrokenShimFails` case extends it to a hung-or-broken binary — addressing R-13's adjacent risk surface.

### Cycle-2 failure analysis

No failures.

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

### Cycle-2 regression checks

| Cycle-1 invariant | Cycle-2 status | Evidence |
| --- | --- | --- |
| All 7 cycle-1 added Go tests (`TestExecuteInit_RendersCodexSurfaces`, 5 × `TestCheckCodexEffectiveConfig_*`, `TestTemplateBaseCodexAssetsExist`) | PASS | go-verbose log + targeted run above |
| `tests/test-check-skill-sync.sh` 6/6 (parity, inventory drift, body drift, description drift, policy drift claude-only-forbid, policy parity both-forbid) | PASS | run-test log |
| `tests/test-check-mojibake.sh` 11/11 (Edit/Write/MultiEdit clean+dirty payloads, allowlist, jq missing fallback) | PASS | run-test log. Particularly relevant for cycle 2 because `templates/base/.codex/config.toml` now wires the same `.claude/hooks/check_mojibake.sh` script that this suite exercises. |
| Existing `internal/scaffold::TestTemplateBaseCodexAssetsExist` (`.codex/` + `.agents/skills/` go:embed coverage) | PASS | go-verbose log |
| Existing `internal/upgrade/` hash-based diff engine (rename surfaces as add+remove) | PASS — 35+ `TestComputeDiffs_*` / `TestUnifiedDiff_*` cases | go-verbose log, package `ok 3.038s` |
| `./scripts/run-verify.sh` (auto-invoked by `run-test.sh`) | exit 0 — "All verifiers passed." | run-test log; `docs/evidence/verify-2026-05-07-100810.log` |

### Cycle-2 skipped / deferred tests

| Test | Reason | Where it is tracked |
| --- | --- | --- |
| `internal/scaffold::TestBaseFS_WithMockFS` | Pre-existing skip — exercises a mock FS path not part of production embed flow. Not introduced by cycle 1 or 2. | Skip reason in `internal/scaffold/embed_test.go:12-27`. |
| `internal/scaffold::TestAvailablePacks_WithMockFS` | Same as above. | Skip reason in `internal/scaffold/embed_test.go:29-49`. |
| Live Codex CLI smoke test (`$spec` → `$plan` → `$work` → post-impl pipeline → `$pr`) | Plan calls this out as manual sign-off (Slice 7, AC-2). Cycle-2 changes do not affect Codex flow surface — the cycle-2 fixes are doctor / hook-config robustness, not driver-side. | `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md`; R-8 in plan. Carried over from cycle 1. |
| `tests/upgrade_downgrade_test.go` | Proposed in plan (Slice 7), still not landed. Cycle-2 did not address this. | Test gap #2 below; R-7 in plan. Carried over from cycle 1. |

### Cycle-2 test gaps

1. **Cycle-1 gaps #1, #2, #3 unchanged.** Live Codex smoke deferred (#1), upgrade round-trip fixture missing (#2), no instrumented coverage numbers (#3) — none of these were in cycle-2 scope.
2. **Cycle-1 gap #4 (Codex CLI absence path) — closed.** The two new `TestProbeBinary_*` cases now cover both the `LookPath`-fails branch and the binary-exists-but-broken branch. The cycle-2 `probeBinary` timeout (5s `context.WithTimeout`) is exercised structurally by `TestProbeBinary_BrokenShimFails` (the broken shim returns immediately, but the timeout path is now in the code under test).
3. **New cycle-2 gap (low priority)**: there is no Go test that asserts `probeBinary` cancels when its timeout fires (i.e., a binary that hangs > 5 s). The current tests cover the success and immediate-failure shapes but do not exercise the timeout boundary itself. This is acceptable for cycle-2 — the timeout is a defensive measure against environmental hangs and writing a 5+ s test for it is poor cost/value at this stage. Recommend a `t.Skip` placeholder or follow-up if a CI runner ever encounters a slow-CLI hang in practice.
4. **Hook-path-existence regression coverage gap**. Cycle-2 fixed two CRITICAL hook-path bugs in `templates/base/.codex/config.toml` (the Edit/Write hooks pointed at non-existent `./scripts/check_mojibake.sh`). There is no automated test that asserts every `command = ["..."]` path in `templates/base/.codex/config.toml` actually exists on disk in the shipped scaffold. The same gap exists for `templates/base/.claude/settings.json`'s `hooks[*].command` paths under Codex (`internal/cli/doctor.go::checkHooks` only walks the Claude side). This is recorded in `docs/tech-debt/README.md` as the "checkHooks only walks .claude/settings.json paths" row and is the right next deterministic check to add — it would have caught both cycle-2 CRITICALs at PR time. Not a cycle-2 fail condition because the underlying bugs are fixed.

### Cycle-2 verdict

- **Pass: yes.** All gated suites green: 11 mojibake + 6 skill-sync + 222 Go top-level (323 incl. subtests). 0 failures, 2 pre-existing Go skips. The 8 specifically requested cases (`TestProbeBinary_BrokenShimFails`, `TestProbeBinary_MissingBinary`, 5 × `TestCheckCodexEffectiveConfig_*`, `TestExecuteInit_RendersCodexSurfaces`) all PASS under cycle-2 source.
- **Fail: no.**
- **Blocked: no.**
- **Cap status: reached.** This is cycle 2/2; no further automatic re-run.
- **Recommendation:** proceed to `/sync-docs` (the verify report flagged one stale `docs/tech-debt/README.md` row about `probeBinary` timeout that was already struck in the latest diff — confirm it lands) → `/pr`. Cycle-2 closed cycle-1 test gap #4 and added two specific regression guards for the doctor probe path. Gaps #1–#3 from cycle 1 remain known and are out-of-scope per plan; the new gap #4 (hook-path on-disk existence check) is the right next thing to add but does not block this merge.
