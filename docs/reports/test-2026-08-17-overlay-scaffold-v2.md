# Test report: overlay-scaffold-v2 (Phase 1 — エンジン基盤)

- Date: 2026-08-17
- Plan: `docs/plans/active/2026-08-17-overlay-scaffold-v2.md`
- Tester: `tester` subagent (`/test`)
- Scope: `./scripts/run-test.sh` (changed-language scope; resolved to `golang` after a full shell-suite fallback triggered by an unclassified test file — see Notes). Behavioral tests only; static analysis is `/verify`'s responsibility (already PASS, see `docs/reports/verify-2026-08-17-overlay-scaffold-v2.md`).
- Evidence: `docs/evidence/test-2026-08-17-overlay-scaffold-v2.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | — |
| `tests/test-branch-name.sh` | 26 | 26 | 0 | 0 | — |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 | — |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | — |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | — |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 | — |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 | — |
| `tests/test-gc-artifacts.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-insights-append.sh` | 39 | 39 | 0 | 0 | — |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-no-loop-references.sh` | 1 | 1 | 0 | 0 | — |
| `tests/test-ralph-config.sh` | 15 | 15 | 0 | 0 | — |
| `tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | — |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | — |
| `tests/test-self-review-scope.sh` | 64 | 64 | 0 | 0 | — |
| `tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 | — |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 | — |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 | — |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | — |
| `tests/test-xreview-helpers.sh` | 29 | 29 | 0 | 0 | — |
| `go test ./...` (golang pack, all 8 packages) | 8 packages | 8 | 0 | 0 | ~24s (dominated by `internal/cli`'s 21.5s subprocess-heavy suite) |

Shell total: 555/555 assertions across 23 files. Go total: 8/8 packages `ok`. Combined verdict: **0 failures**.

## Coverage

Package-level (`go test ./... -cover`), full detail via `go tool cover -func` on per-package profiles:

- `internal/scaffold`: 77.7% statements
- `internal/upgrade`: 89.7% statements
- Function-level: every Phase-1-new symbol in `internal/scaffold/manifest.go` is at 100% — `SetLayoutV2`, `SetFileOwned`, `SetFileFork`, `validateManagedOwner`, `IsLegacyOwner`.
- Notes: the two packages' overall percentages are pulled down entirely by **pre-existing, untouched code**, not by Phase 1's new primitives:
  - `internal/scaffold`: `embed.go`'s `BaseFS`/`PackFS`/`AvailablePacks` (0%, go:embed accessors, no behavior to unit-test), and `render.go`/`baseline.go` partial coverage — none of these files were modified by this PR except `baseline.go`, whose only change was a mechanical rename (`cleanTemplateRelPath` → `CleanLocalRelPath`, AC-9's shared-validator consolidation) that deleted the old private duplicate and reused the shared one; no new uncovered code was introduced.
  - `internal/upgrade`: `diff.go`'s `ComputeDiffsNoRemovals`/`ComputeFileDiff` (0%) and `merge.go`'s `JoinLines` (0%) are the legacy interactive-conflict-resolution machinery the plan's Assumptions/Risks sections name as Phase 3's replacement target — confirmed untouched by `git diff main...HEAD --stat -- internal/upgrade/diff.go internal/upgrade/merge.go` (empty diff). All five new Phase-1 primitive files (`replaceplan.go` 91.2%/88.2%/94.1%/90.0%/90.9%/72.7%/76.5%/85.7% across functions, `block.go` 83–100%, `settingsmerge.go` 44–100%, `advisory.go` 83–100%, `report.go` 66–100%) sit in the 44–100% band typical of table-driven error-path testing; the plan's own edge-case list (AC-2/AC-3/AC-4/AC-9) is covered per the verify report's AC evidence table, and no primitive function is at 0%.
- No coverage regression: `internal/cli` diff is empty (verify report's AC-6 finding), so its coverage is unaffected by this PR.

## Failure analysis

None. No test failures in shell suites or Go packages.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `tests/test-gc-artifacts.sh` "recent" fixture used a fixed date (`2026-07-12`) that fell outside the `--days 30` window once real time passed 2026-08-11, causing spurious failures unrelated to any diff (found and fixed in this task's Deviations, commit `7dca14e`) | Fixed and stays fixed | Suite ran and passed 11/11 in this run (`docs/evidence/test-2026-08-17-overlay-scaffold-v2.log:199-238`), confirming the date-relative fixture no longer drifts out of the retention window |
| Existing `internal/upgrade`/`internal/scaffold`/`internal/cli` behavior unchanged by Phase 1's new, unwired primitives (AC-6) | Confirmed | `go test ./...` all 8 packages `ok`; `internal/cli` diff empty (verify report) |

## Test gaps

- Full behavioral coverage of the wired `ralph upgrade` flow (FR-1's 8-step non-interactive pipeline), `ralph eject`/`adopt`, and `ralph doctor --strict` is out of scope for Phase 1 by design — these primitives are deliberately unwired until Phases 2–5. No gap relative to this plan's own Test plan section.
- `internal/upgrade/diff.go` (`ComputeDiffsNoRemovals`, `ComputeFileDiff`) and `merge.go` (`JoinLines`) remain at 0% coverage; both are pre-existing legacy interactive-conflict machinery, untouched by this PR, and named in the plan as Phase 3's replacement target — not a Phase-1 regression, but worth tracking so Phase 3 doesn't inherit blind spots when it deletes/replaces this code.
- `internal/scaffold/embed.go`'s three go:embed accessors (0%) are trivial pass-throughs over compile-time embedded FS and have no branch logic to test; low-value gap.

## Verdict

- Pass: Yes — all 555 shell assertions across 23 files and all 8 Go packages (`go test ./...`) pass with zero failures. Proceed to `/sync-docs`.
- Fail: None.
- Blocked: None.

## Cycle 2 (re-run after cross-review fixes + cycle-2 self-review cleanup)

- Date: 2026-08-17
- Pipeline cycle: 2/2 (`RALPH_STANDARD_MAX_PIPELINE_CYCLES` default cap; no automatic third cycle)
- Tester: `tester` subagent (`/test`)
- Scope: `./scripts/run-test.sh` (changed-language scope; again resolved to `golang` after a full shell-suite fallback triggered by the unclassified `scripts/check-sync.sh` edit in `3b32c50` — same fallback trigger as cycle 1, now via a different file). Delta since the cycle-1 test report (`6564230..HEAD`): `4c38cf1` (AGENTS.md repo-map sync + `check-sync.sh` `KNOWN_DIFFS`), `ba2384f` (cross-review triage report), `1ef5be7` (cross-review fixes — `ManifestRemove` signal, `ApplyOps` validate-all-upfront, report-path prefix hardening), `c81eee2` (cycle-2 self-review report), `3b32c50` (cycle-2 self-review fixes).
- Evidence: `docs/evidence/verify-2026-08-17-074947.log`

### Test execution (full re-run)

Same 23 shell suites and same 8 Go packages as cycle 1 ran again in full (fallback scope, not delta-only). Every suite's pass/fail count is unchanged from the cycle-1 table above — `tests/test-branch-name.sh` is unchanged at 26 (the cycle-1 report row and this cycle's evidence log both show 26/26; no shell test file was touched by the cycle-2 delta) — with one exception:

| Suite / Command | Cycle 1 | Cycle 2 | Delta |
| --- | --- | --- | --- |
| `go test ./internal/upgrade/...` | passing, no `ManifestRemove`/`ApplyOps`-upfront/report-path-prefix tests | passing, **+5 new tests** | `TestPlanCoreReplace_CoreManifestRemoveWhenDiskAlreadyAbsent`, `TestApplyOps_RejectsInvalidOpPathBeforeWritingAnything`, `TestUpgradeReportRelPath_SanitizesDateParentEscape`, `TestWriteUpgradeReport_RejectsPathOutsideReportsDir`, `TestWriteUpgradeReport_RejectsReportsDirItself` |
| All other 22 shell files + 7 other Go packages | pass | pass | No change (no source touched by the cycle-2 delta) |

All 5 new tests independently re-run in isolation (`go test ./internal/upgrade/... -run '<names>' -v`): 5/5 `PASS`. Full suite: shell 555/555 assertions across 23 files, Go 8/8 packages `ok`. Combined verdict: **0 failures**.

Note on `settingsmerge_test.go`: `3b32c50` also touched this file, but only a doc-comment rewrite on `TestOwnedSettingsPaths_AnchorsMergeBehavior` (C2-2 self-review fix) — no assertion logic changed, confirmed via `git diff 6564230..HEAD -- internal/upgrade/settingsmerge_test.go`. Not counted as a new test.

### Coverage (cycle 2)

- `internal/scaffold`: 77.7% statements — **unchanged** from cycle 1 (no `internal/scaffold` file touched by the cycle-2 delta; confirmed via `git diff 6564230..HEAD --stat -- internal/scaffold` = empty).
- `internal/upgrade`: 89.9% statements — **+0.2pp** from cycle 1's 89.7%. The 5 new tests each cover a previously-untested branch: `ManifestRemove` on the already-absent-on-disk path (`replaceplan.go`), the upfront-validation loop in `ApplyOps` before any op executes, `date`-string sanitization in `UpgradeReportRelPath`, and the two `WriteUpgradeReport` rejection paths (outside-`docs/reports`, and `docs/reports` itself).
- No new 0%-coverage functions introduced. The cycle-1 report's named gaps (`diff.go`'s `ComputeDiffsNoRemovals`/`ComputeFileDiff`, `merge.go`'s `JoinLines`, `internal/scaffold/embed.go`'s go:embed accessors) are unchanged — none of those files were touched by the cycle-2 delta (`git diff 6564230..HEAD --stat` confirms).

### Failure analysis (cycle 2)

None. No test failures in shell suites or Go packages, including the 5 new tests and the reworded (not logic-changed) `settingsmerge_test.go` case.

### Regression checks (cycle 2)

| Behavior at risk | Status | Evidence |
| --- | --- | --- |
| `ApplyOps`'s pre-existing stop-at-first-failure / no-manifest-advance contract, now composed with the new upfront path-validation loop | Unchanged | `TestApplyOps_RejectsInvalidOpPathBeforeWritingAnything` asserts the valid op preceding the invalid one is never written — the new validation runs *before*, not instead of, the existing op loop |
| `ManifestRemove`'s non-destructive default (drifted/modified-core-file case must NOT emit a removal signal) | Unchanged | Existing drifted-case test (extended in `1ef5be7`) explicitly asserts `len(plan.ManifestRemove) == 0`; not weakened by the new already-absent-case test |
| AC-8 (v3 opt-in isolation) — `manifest.go` v3 write paths | Unchanged | No `manifest.go` change in the cycle-2 delta; `TestExistingConstructorsWriteNoV3Fields` untouched and still passing |
| AC-6 (no `internal/cli` behavior change) | Unchanged | `git diff main...HEAD -- internal/cli/` empty at `HEAD = 3b32c50` (cross-checked against the cycle-2 verify report's own AC-6 re-check) |

### Test gaps (cycle 2)

- Unchanged from cycle 1: full behavioral coverage of the wired `ralph upgrade`/`eject`/`adopt`/`doctor --strict` flow remains out of scope by design (Phases 2–5); `diff.go`/`merge.go` legacy machinery and `embed.go`'s go:embed accessors remain at 0%, both pre-existing and untouched by this cycle's delta.
- The 5 cycle-1 LOW findings the verify report batched into the tech-debt register (`docs/tech-debt/README.md`) as unfixed-by-cap remain untested by definition — they describe code paths the fixes did not touch. Not a cycle-2 regression; tracked for a future cycle.

### Verdict (cycle 2)

- Pass: Yes — all 555 shell assertions across 23 files and all 8 Go packages (`go test ./...`) pass with zero failures, including the 5 new `internal/upgrade` tests added in this cycle's fix commits. `internal/upgrade` coverage rose 89.7% → 89.9%; `internal/scaffold` unchanged at 77.7%. No regression in AC-2, AC-6, AC-8, or AC-9 behavior. Proceed to `/sync-docs` → `/cross-review` → `/pr`.
- Fail: None.
- Blocked: None.
