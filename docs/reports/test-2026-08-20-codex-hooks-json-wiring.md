# Test report — codex-hooks-json-wiring (cycle 1)

- Plan: `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
- Branch: `fix/codex-hooks-json-wiring`
- Scope: `./scripts/run-test.sh` (changed scope; detected `.codex/config.toml` as unclassified → full fallback → golang pack selected)
- Verdict: **PASS**

## Summary

`./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=changed`, `HARNESS_VERIFY_MODE=test`) ran the full shell suite (27 files) plus the golang test pack. All 621 shell assertions passed (0 failures) and all 8 Go packages with tests passed. A fresh, uncached `go test ./... -count=1 -cover` run confirms the same result with per-package coverage.

Evidence log: `docs/evidence/verify-2026-08-20-063848.log`

## Counts

### Shell (27 files, 621 assertions, 0 failures)

| File | Pass/Total |
|---|---|
| test-agent-phase-boundaries.sh | 44/44 |
| test-branch-name.sh | 26/26 |
| test-check-mojibake.sh | 11/11 |
| test-check-skill-sync.sh | 13/13 |
| test-detect-changed-languages.sh | 23/23 |
| test-detect-languages-terraform.sh | 8/8 |
| test-ensure-pr-ready.sh | 7/7 |
| test-ensure-pr-title-prefix.sh | 13/13 |
| test-gc-artifacts.sh | 11/11 |
| **test-hook-wiring.sh** | **26/26** |
| test-insights-append.sh | 39/39 |
| test-language-pack-monorepo-roots.sh | 29/29 |
| test-no-loop-references.sh | 1/1 |
| test-pre-bash-guard.sh | 4/4 |
| test-ralph-config.sh | 15/15 |
| test-ralph-dispatch.sh | 26/26 |
| test-ralph-worktree.sh | 29/29 |
| test-run-verify-scope.sh | 12/12 |
| test-secret-scan.sh | 6/6 |
| test-self-review-scope.sh | 64/64 |
| test-sync-skills.sh | 22/22 |
| test-template-purity.sh | 10/10 |
| test-terraform-gitignore.sh | 47/47 |
| test-terraform-pack-verify.sh | 36/36 |
| test-terraform-rule-frontmatter.sh | 11/11 |
| test-verify-mode-split.sh | 59/59 |
| test-xreview-helpers.sh | 29/29 |

`test-hook-wiring.sh` grew from the plan's 26-case rewrite (`.codex/hooks.json` validity, PostToolUse dispatcher routing, `config.toml` no-longer-has `[hooks]`/`[[hooks.*]]`, no direct-call hook commands, byte-identity root ↔ `templates/base`, plus the whitespace-tolerant TOML-table-detector self-test), run for both `root` and `templates/base`. Re-run isolated 3x: 26/26 every time.

### Go (fresh `go test ./... -count=1 -cover`, uncached)

```
?    github.com/yoshpy-dev/ralph                          [no test files]
     github.com/yoshpy-dev/ralph/cmd/ralph                coverage: 0.0% of statements
ok   github.com/yoshpy-dev/ralph/internal/cli       35.982s coverage: 80.5% of statements
ok   github.com/yoshpy-dev/ralph/internal/config     1.229s coverage: 94.2% of statements
ok   github.com/yoshpy-dev/ralph/internal/insights   1.442s coverage: 86.1% of statements
ok   github.com/yoshpy-dev/ralph/internal/org       11.316s coverage: 89.1% of statements
ok   github.com/yoshpy-dev/ralph/internal/org/driver 1.283s coverage: 92.0% of statements
ok   github.com/yoshpy-dev/ralph/internal/org/protocol 3.203s coverage: 97.9% of statements
ok   github.com/yoshpy-dev/ralph/internal/scaffold   2.201s coverage: 75.7% of statements
ok   github.com/yoshpy-dev/ralph/internal/upgrade    2.951s coverage: 91.2% of statements
```

8/8 packages with tests pass, 0 failures, 0 skips.

**Coverage vs v5-series baseline:** `internal/cli` is 80.5% (baseline at the P5/overlay-scaffold-v2 merge was 80.6%; -0.1pp). This is a small, expected drift, not a regression: the branch added 12 new Go tests (`checkCodexEffectiveConfig` schema-validation coverage, listed below) that exercise well-covered code paths in `doctor.go`, so the denominator (total statements) grew slightly faster than the newly-added test lines in a few branches. No existing test was weakened or removed to produce this delta — confirmed via `git diff main...HEAD -- internal/cli/doctor.go internal/cli/cli_test.go`, which shows only additions/refactors to `checkCodexEffectiveConfig` and its test table, no deletions of prior assertions. All other package coverages match the last recorded baseline exactly (config 94.2%, insights 86.1%, org 89.1%, org/driver 92.0%, org/protocol 97.9%, upgrade 91.2%; scaffold 75.7% vs P5's 75.7%).

## Named checks (per task request)

All pass, isolated `go test -run <name> -count=1 -v`:

**AC-3b negative schema-validation tests (4/4):**
- `TestCheckCodexEffectiveConfig_HooksJSONTopLevelEventKey_Warns` — `PostToolUse` at top level instead of nested under `hooks`
- `TestCheckCodexEffectiveConfig_HooksJSONHooksKeyMissing_Warns` — `hooks` key absent entirely
- `TestCheckCodexEffectiveConfig_HooksJSONHandlerMissingType_Warns` — handler object missing `type: "command"`
- `TestCheckCodexEffectiveConfig_HooksJSONCommandAsArray_Warns` — `command` is an array instead of a string

**Other AC-3/AC-4 doctor tests:**
- `TestCheckCodexEffectiveConfig_HooksFeatureExplicitFalse_Warns` — explicit `[features].hooks = false` warns (absence stays lenient, per `TestCheckCodexEffectiveConfig_HooksFeatureAbsent_Lenient`)
- `TestCheckCodexEffectiveConfig_DualRepresentation_Warns` — both `config.toml` `[[hooks.*]]` and `hooks.json` present simultaneously
- `TestCheckCodexEffectiveConfig_DispatcherRoutingMissing_Warns` — `hooks.json` present/valid/schema-correct but no entry routes through `ralph-dispatch.sh`
- `TestCheckCodexEffectiveConfig_HooksJSONMissing_Warns`, `TestCheckCodexEffectiveConfig_FullyWired_Pass`, `TestCheckCodexEffectiveConfig_InvalidTOML_Fails`, `TestCheckCodexEffectiveConfig_MissingConfigToml_Warns`, `TestCheckCodexEffectiveConfig_DeprecatedFeatureFlagKey_TreatedAsAbsent` — all pass (13 `TestCheckCodexEffectiveConfig_*` tests total in `internal/cli/cli_test.go`, 12 net-new per the handoff plus 1 pre-existing renamed/extended for the inversion)

**Init `hooks.json` owner=core test (`internal/cli/init_v2_test.go`):**
- `TestExecuteInit_V2_FreshInit_LayoutAndOwners` — asserts `.codex/hooks.json` maps to `scaffold.OwnerCore` in the manifest ownership table
- `TestExecuteInit_RendersCodexSurfaces` — asserts `.codex/hooks.json` is rendered on fresh init
- `TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses` — asserts a fresh-init scaffold passes `checkCodexEffectiveConfig` end-to-end

All three pass in isolation.

**`tests/test-hook-wiring.sh` (26 cases, 3x isolated):** 26/26 pass on every run (see Shell section above).

## Flake watchlist (3x isolated each)

- `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` (`internal/cli`) — PASS, PASS, PASS (0.57s–1.00s each)
- `TestRunWatcher_TimeoutIndependentOfSmallInterval` (`internal/org`) — PASS, PASS, PASS (~1.18s each, run against the exact package, not `/...`, per the memory note that `/...` pattern-matches into sibling packages with `[no tests to run]`)

No flakes observed in this cycle.

## Coverage gaps (unchanged from prior cycles, noted for completeness)

- `doctor.go`'s `checkScaffoldCoreHashes` untracked-drift branch remains untested (pre-existing gap, not touched by this branch).
- Shell suites have no instrumented coverage tool; scope is measured by test-case enumeration only, as in prior cycles.

## Verdict

**PASS.** All shell assertions (621/621) and all Go packages (8/8) pass. No regressions relative to the prior recorded baseline; the 12 new `checkCodexEffectiveConfig` tests (including all 4 AC-3b negative schema tests) and the init owner=core test all pass in isolation. Tests are green — proceeding to `/pr` is safe from the test gate's perspective.

---

# Test report — codex-hooks-json-wiring (cycle 2)

- Pipeline cycle: 2/2 (cap `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`)
- HEAD at test time: `67f56d5`
- Delta since cycle 1 (`4cff65f`): `c72e644`/`7af720a` (cycle-1 self-review fixes, landed before cycle-1 test), `8aaa27b`/`cb54856` (tech-debt row), `cdb0aad` (cross-review triage), `d1df46f` (AR#1 fix: non-boolean `features.hooks` now warns, + `TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns`), `00ee645` (self-review cycle-2 report), `bced11a` (self-review cycle-2 fixes C2-M1/M2/L1-L5: doc/comment + doctor message reshape), `67f56d5` (fix orphaned sentence fragment), `1b5cfa2` (verify cycle-2 report)
- Scope: `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=changed`; same full-fallback path as cycle 1 — `.codex/config.toml` is unclassified → golang pack selected)
- Verdict: **PASS**

## Summary

Re-ran the full shell suite (27 files) and the golang test pack after the cycle-1→cycle-2 delta (AR#1 fix, self-review cycle-2 fixes, doc-only commits). All 621 shell assertions passed (0 failures) — byte-identical count to cycle 1. All 8 Go packages passed with a fresh, uncached `go test ./... -count=1 -cover`. `internal/cli` coverage is unchanged at 80.5%.

Evidence log: `docs/evidence/verify-2026-08-20-112310.log`

## Counts

### Shell (27 files, 621/621, 0 failures — identical to cycle 1)

Same 27-file inventory as the cycle-1 table above; every file's pass/total is unchanged (`test-hook-wiring.sh` still 26/26). No new or removed shell tests in this delta — the AR#1 fix (`d1df46f`) and self-review cycle-2 fixes (`bced11a`) touched only `internal/cli/doctor.go` and `internal/cli/cli_test.go` (Go), plus doc/comment-only changes in `.codex/config.toml` and a report file.

### Go (fresh `go test ./... -count=1 -cover`, uncached)

```
?    github.com/yoshpy-dev/ralph                          [no test files]
     github.com/yoshpy-dev/ralph/cmd/ralph                coverage: 0.0% of statements
ok   github.com/yoshpy-dev/ralph/internal/cli       39.669s coverage: 80.5% of statements
ok   github.com/yoshpy-dev/ralph/internal/config     1.343s coverage: 94.2% of statements
ok   github.com/yoshpy-dev/ralph/internal/insights   2.283s coverage: 86.1% of statements
ok   github.com/yoshpy-dev/ralph/internal/org       11.661s coverage: 89.1% of statements
ok   github.com/yoshpy-dev/ralph/internal/org/driver 0.884s coverage: 92.0% of statements
ok   github.com/yoshpy-dev/ralph/internal/org/protocol 2.743s coverage: 97.9% of statements
ok   github.com/yoshpy-dev/ralph/internal/scaffold   1.985s coverage: 75.7% of statements
ok   github.com/yoshpy-dev/ralph/internal/upgrade    3.705s coverage: 91.2% of statements
```

8/8 packages with tests pass, 0 failures, 0 skips. `internal/cli` at 80.5% — identical to the cycle-1 figure (cycle 1's report showed the same 80.5%, itself -0.1pp vs the pre-branch P5 baseline of 80.6% due to the 12 net-new `checkCodexEffectiveConfig` tests). The AR#1 fix added one more test (`TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns`) exercising an already-covered branch of `doctor.go`, so the coverage percentage did not move further. All other package coverages are byte-identical to cycle 1 (config 94.2%, insights 86.1%, org 89.1%, org/driver 92.0%, org/protocol 97.9%, scaffold 75.7%, upgrade 91.2%).

## Named checks (per task request)

**New AR#1 test, 3x isolated (`go test -run <name> -count=3 -v`):**
- `TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns` — PASS, PASS, PASS
- `TestCheckCodexEffectiveConfig_HooksFeatureExplicitFalse_Warns` (sibling test, unaffected by the AR#1 fix) — PASS, PASS, PASS

**`tests/test-hook-wiring.sh` full suite, 3x isolated (`bash tests/test-hook-wiring.sh`):** 26/26 pass on every run, no flakes.

## Flake watchlist (3x isolated each)

- `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` (`internal/cli`) — PASS, PASS, PASS (~0.58s each)
- `TestRunWatcher_TimeoutIndependentOfSmallInterval` (`internal/org`, targeted exact package per the memory note that `/...` can mis-resolve to a sibling package) — PASS, PASS, PASS (~1.20s each)

No flakes observed in this cycle.

## Coverage gaps (unchanged from cycle 1)

- `doctor.go`'s `checkScaffoldCoreHashes` untracked-drift branch remains untested (pre-existing gap, not touched by this delta).
- Shell suites have no instrumented coverage tool; scope is measured by test-case enumeration only.

## Verdict

**PASS.** All shell assertions (621/621, byte-identical to cycle 1) and all Go packages (8/8) pass. The AR#1 fix and self-review cycle-2 fixes introduced no regressions: the new `TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns` test and its sibling pass 3x isolated, `test-hook-wiring.sh` is stable 3x isolated, and both flake-watchlist tests are stable 3x isolated. Tests are green — proceeding past the cycle-2 gate (to `/pr`, per the pipeline cap) is safe from the test gate's perspective.
