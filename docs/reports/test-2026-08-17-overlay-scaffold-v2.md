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
