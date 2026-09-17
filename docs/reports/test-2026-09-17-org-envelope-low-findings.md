# Test report: org-envelope-low-findings

- Date: 2026-09-17
- Plan: `docs/plans/active/2026-09-17-org-envelope-low-findings.md` (issue #154)
- Tester: `tester` subagent (Claude Code, standard flow)
- Diff scope: `git diff 4905c34...HEAD` on `chore/org-envelope-low-findings` (HEAD `2add1cd`), 19 files, +517/-54.
- Scope note: behavioral tests only (`./scripts/run-test.sh`, changed-language scope = golang, plus the full shell suite it always runs). No static analysis, gofmt, vet, or lint here — that is `/verify`'s job and already reported Pass. `git status --porcelain` confirmed a clean working tree before and after this pass; no source files were edited.

## Overall verdict: Pass

All targeted unit/regression tests pass. The full changed-scope suite (`./scripts/run-test.sh`) exits 0. A fresh, uncached `go test ./internal/... -count=1` (the AC-11 test clause) is green across all 8 test-bearing packages. Coverage on the three changed packages meets or exceeds the pre-change baseline. Reviewed the new/changed test bodies against the fix each protects and confirmed each one would fail if the corresponding fix were reverted (see "Discrimination check" below) — reasoned from source, not by reverting anything.

## Suite run: `./scripts/run-test.sh`

```
$ ./scripts/run-test.sh
```

- Exit code: 0
- Shell suite (always full scope, not language-scoped): **708/708 assertions pass, 0 fail, across 28 `tests/*.sh` files** (`test-agent-phase-boundaries.sh` … `test-xreview-helpers.sh`). No `FAIL:` line with a nonzero count anywhere in the log.
- Go verifier (`HARNESS_VERIFY_MODE=test`, `RALPH_VERIFY_SCOPE=changed` — the wrapper default): language scope resolved to `golang` only (matches the diff: no other language changed). Reported `ok` (cached) for all 8 Go packages; `==> All verifiers passed.`
- Evidence: `docs/evidence/verify-2026-09-17-102036.log` (gitignored, local to this worktree)

Confirmed `go build ./...` is clean before testing (exit 0, no output).

## Fresh (uncached) Go test run — three changed packages, with coverage

```
$ go test ./internal/cli/... ./internal/org/... ./internal/config/... -count=1 -cover
```

| Package | Result | Coverage | Prior baseline (2026-09-17, org-implementer-seat-envelope cycle 2) |
|---|---|---|---|
| `internal/cli` | ok (37.4s) | 81.0% | 80.9% (+0.1pp — 3 new doctor tests exercising already-covered branches) |
| `internal/org` | ok (10.4s) | 89.1% | 89.1% (unchanged) |
| `internal/org/driver` | ok (2.4s) | 92.0% | 92.0% (unchanged) |
| `internal/org/protocol` | ok (1.2s) | 97.9% | 97.9% (unchanged) |
| `internal/config` | ok (1.9s) | 92.3% | 92.3% (unchanged) |

No coverage regression in any of the three packages the plan names as changed (`internal/cli`, `internal/org`, `internal/config`); the two `internal/org/*` sub-packages are unaffected by this diff and are reported for completeness.

## AC-11 test clause: `go test ./internal/...`

```
$ go test ./internal/... -count=1
```

All 8 test-bearing packages `ok` (fresh, no cache): `internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade`. No failures, no skips beyond the platform-conditional skips noted below (this run was on macOS/darwin, so none of those skips triggered).

## Targeted tests (as named in the task handoff)

### `internal/cli/doctor_org_test.go` — `TestCheckCodexModelSlugs_*` (`-run 'TestCheckCodexModelSlugs_' -v -count=1`)

| Test | Result |
|---|---|
| `TestCheckCodexModelSlugs_AllPresent_Pass` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_NoCodexEntries_Info` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace` (new) | PASS |
| `TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally` (new) | PASS |
| `TestCheckCodexModelSlugs_HomeUnresolvable_Info` (new) | PASS |

All 8/8 pass, `ok internal/cli 0.459s`.

### `internal/cli/org_test.go` — `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry`

PASS, `ok internal/cli 0.297s`.

### `internal/org/prompts_test.go` and `internal/org/watch_test.go`

| Test | Subtests | Result |
|---|---|---|
| `TestMarkdownSection_AnchorsHeaderAndBoundsBody` (new) | normal section stops at next `##`; empty section returns empty body; section at EOF; level-3 header does not satisfy level-2 lookup; mid-line mention does not satisfy lookup | PASS, 5/5 subtests |
| `TestRolePrompts_SeatTemplatesContainFanOutSection` (section-scoped) | implementer, reviewer, qa | PASS, 3/3 subtests |
| `TestRenderRolePrompt_Lead_AllKnownVarsSubstituted` (fixture from `config.Default()`) | — | PASS |
| `TestRenderRolePrompt_Lead_DelegatesToImplementer` | — | PASS |
| `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` | — | PASS |

All 5 (9 including subtests) pass, `ok internal/org 0.403s`.

### `internal/config/config_test.go` and `defaults_sync_test.go`

| Test | Result |
|---|---|
| `TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder` (renamed) | PASS |
| `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors` (tightened) | PASS |
| `TestDefaultsLockStep` (`defaults_sync_test.go`) | PASS |

All 3/3 pass.

## Discrimination check (per `.claude/rules/ralph/testing.md`)

Instructed not to revert source to prove this; reasoned from the test and production code instead, quoting the exact lines that make each new/changed assertion fail against the pre-fix behavior it replaced.

- **`TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace`** (`internal/cli/doctor_org_test.go:521-547`) creates two directories, `<tmp>/codex ` (matching slug) and `<tmp>/codex` (no matching slug), sets `CODEX_HOME=<tmp>/codex ` and asserts `Status == "pass"`. `codexModelsCachePath` (`internal/cli/doctor_codex_models.go:24-32`) has no `TrimSpace` call — confirmed by `grep -n 'TrimSpace' internal/cli/doctor_codex_models.go` returning empty (exit 1). If a `strings.TrimSpace(home)` were reintroduced before the `filepath.Join`, the check would read the trimmed, slug-less directory and report `warn`, failing this assertion. Discriminates correctly.
- **`TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally`** (`:549-575`) sets `CODEX_HOME=" "` and asserts the `info` detail names the literal path `filepath.Join(" ", "models_cache.json")`. The pre-fix behavior (per the plan's item 1) treated a whitespace-only `CODEX_HOME` as unset via `TrimSpace(home) == ""` and fell back to `$HOME/.codex/...` instead — a different path string that would not contain `" /models_cache.json"`. Discriminates correctly.
- **`TestCheckCodexModelSlugs_HomeUnresolvable_Info`** (`:577-600`) sets both `CODEX_HOME=""` and `HOME=""` and asserts `Status == "info"` with `Detail` containing `"could not resolve codex models cache path"`. That literal string is only produced by the new `(string, error)` return path (`doctor_codex_models.go:34-38`, wrapped again at the `checkCodexModelSlugs` call site). The pre-fix version silently discarded the `os.UserHomeDir` error (per the plan's item 1 framing) and would have produced a different detail message (naming a relative `.codex/models_cache.json` path, not a resolution failure). Discriminates correctly.
- **`TestMarkdownSection_AnchorsHeaderAndBoundsBody`** (`internal/org/prompts_test.go:44-63`) uses a fixture containing the deliberate mid-line mention `"see ## D for details"` and asserts `markdownSection(doc, "## D")` returns `found=false`. The helper (`:23-38`) anchors on `"\n"+header+"\n"` specifically to reject that mid-line match; an unanchored `strings.Index(text, header)` implementation (the kind `TestRolePrompts_SeatTemplatesContainFanOutSection` used before this fix, per the plan's item 5b and Deviation notes N1/N3) would report `found=true` for the mid-line case and `found=true` returning the wrong slice boundary for the empty-section case. Discriminates correctly on both the anchoring and the empty-section cases.
- **`org_test.go:673`** (`if n := strings.Count(out, wantWarn); n != 1`) fails whenever the fallback warning is emitted zero or more-than-once, whereas the prior `strings.Contains` assertion it replaced (plan item 5a) would pass on a duplicate-emission regression. Discriminates correctly against a duplication bug that `Contains` cannot see.
- **`TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors`** (`config_test.go:812-835`) asserts the single string `` `[org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool"` `` — both the exact filtered-model phrase and the exact model literal in the same check, matching `internal/config/config.go`'s error format byte-for-byte. A weaker assertion checking only for the role name (the pre-fix version, per plan item 10) would not catch a regression that names the wrong model or drops the "not present in" reason. Discriminates correctly.
- **`pruneRetiredConditions`** (plan item 8, `internal/org/watch.go:562-578`) is a genuine no-behavior-change cleanup (removing a redundant delete from the `PendingAlerts` loop that the subsequent `Escalated` loop already performs under the identical predicate) — per the plan's own risk note, the existing regression test is expected to pass identically before and after, since it checks the end state of all three maps, not which loop performed which delete. `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` passing is consistent with this being a true refactor, not a behavior test that should discriminate.

## Edge cases from the plan's Test plan section

- `CODEX_HOME="<dir> "` (trailing space) → read literally, `pass` when the literal dir has the slug. Covered, `runtime.GOOS == "windows"` skip present (`doctor_org_test.go:522-524`) — this run was on darwin, so the skip did not trigger and the case executed.
- `CODEX_HOME="  "` (whitespace-only) → literal join, `info` with "not found at …" (not treated as unset). Covered, Windows skip present (`:558-560`).
- `HOME=""` and `CODEX_HOME=""` → `info`, no panic. Covered, Windows skip present (`:582-584`).
- All three edge-case tests ran (not skipped) on this darwin host and passed.

## Coverage gaps relative to the plan

- No gap identified against the plan's stated Unit/Integration/Regression/Edge-case list — every named test in the Test plan section exists and passes, and the plan explicitly scopes Integration tests to "none" (existing `TestOrgSpawn_ModelFlagOmitted_*` dry-run suffices), which this run confirms still passes.
- Pre-existing, out-of-plan-scope note: the Windows-only skip branches (3 tests) are untested by this run since the host is darwin — this is a portability gap inherent to the test design (documented in the plan's Edge cases and Risks sections as unavoidable, not something this PR introduces) rather than a gap this test pass can close.
- The `internal/org/prompts_test.go` `markdownSection` helper is test-only (not shipped in `internal/org/prompts.go`); its 5-subtest table is the only test surface for it, which is appropriate for a test-fixture helper.

## Flaky test watch

The two tests flagged in prior cycles as transient subprocess-timeout flakes under parallel contention (`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` in `internal/cli`, `TestRunWatcher_TimeoutIndependentOfSmallInterval` in `internal/org`) both passed in every run this cycle (the full `./scripts/run-test.sh` pass, the three-package coverage run, and the full `go test ./internal/...` run). No flake reproduced.

## Known gaps / out of scope for this pass

- Static analysis (gofmt/vet/lint/staticcheck), `check-skill-sync.sh`, `check-sync.sh` — `/verify`'s job, already reported Pass in `docs/reports/verify-2026-09-17-org-envelope-low-findings.md`.
- Diff quality — `/self-review`'s job, already reported Merge with no open findings.
- Documentation sync — `/sync-docs`'s job, not yet run at this pipeline stage.

## Evidence

- `docs/evidence/verify-2026-09-17-102036.log` (`./scripts/run-test.sh` full output, gitignored)
- Fresh coverage run: `go test ./internal/cli/... ./internal/org/... ./internal/config/... -count=1 -cover` (output captured above)
- Fresh full-package run: `go test ./internal/... -count=1` (all 8 packages `ok`)
- Targeted `-v -count=1` runs for every test named in the task handoff (output captured above)
