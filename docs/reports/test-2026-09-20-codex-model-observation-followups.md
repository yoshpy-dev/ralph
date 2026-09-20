# Test report: codex-model-observation-followups

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-model-observation-followups.md` (issue #173)
- Tester: `tester` subagent (Claude Code), pipeline cycle 1/2
- Scope: behavioral tests only for `git diff main...HEAD` at `638db39` (branch `fix/codex-model-observation-followups`). Not in scope: static analysis (`/verify`, already run — `docs/reports/verify-2026-09-20-codex-model-observation-followups.md`, pass) or diff quality (`/self-review`, already run — `docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`, 2 MEDIUM + 5 LOW, all fixed in `6564680`).
- Evidence: `docs/evidence/test-2026-09-20-codex-model-observation-followups.log`

Precondition confirmed before starting: HEAD was `638db3926f41b68ffdcda21ad50d72f195e2678a`, `git status --porcelain` was empty.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (shell, changed-language scope) | multiple suites | all | 0 | 0 | ~1 min |
| `./scripts/run-test.sh` → golang verifier (`go test ./...`) | 8 test-bearing packages | 8 | 0 | 0 | ~8 s (org) |
| `go test ./internal/org/... ./internal/cli/... ./internal/insights/... -count=1` | 799 (top-level + subtests, `-v`) | 799 | 0 | 0 | ~55 s |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 | ~9 s |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | ~52 s |
| Home independence (`org.test`/`cli.test`, `HOME=/nonexistent-home-for-test CODEX_HOME=`, run from each package's own directory) | 2 binaries | 2 | 0 | 0 | — |
| Flakiness: broad codex set `-count=50` | 1 package | ok | 0 | 0 | ~28 s |
| Flakiness: `-race` codex set `-count=20` | 1 package | ok | 0 | 0 | ~6 s |
| Flakiness: two timing-shaped Spawn tests `-count=200` | 400 subtests | 400 | 0 | 0 | ~2 s |
| Flakiness: same two tests `-count=50` under load (full `internal/cli` suite running concurrently) | 100 subtests | 100 | 0 | 0 | ~1 s foreground, ~43 s background |
| Time zones (`UTC`, `Asia/Tokyo`, `America/Los_Angeles`, `Pacific/Kiritimati`, `Pacific/Pago_Pago`) × `Codex\|Rollout\|Stop_Codex` | 5 runs | 5 | 0 | 0 | <1 s each |
| Red/green mutation checks 9a–9h + item-10 gap checks | 11 checks | see below | — | — | — |

Full command tails: `docs/evidence/test-2026-09-20-codex-model-observation-followups.log`.

### Command 5 detail (home independence)

Built `org.test`/`cli.test` via `go test -c`. Running `cli.test` from the worktree root failed two unrelated tests (`TestBackfillCmd_DryRun`, `TestBackfillCmd_Apply`) with exit 1. Root-caused: those two tests depend on the process's current working directory for a relative fixture path, unrelated to `$HOME`/`$CODEX_HOME` — not a regression from this branch (`internal/cli` is untouched by this diff, confirmed by `git diff main...HEAD --stat -- internal/cli` being empty) and not new: re-running the same binary from inside `internal/cli/` (matching the convention `docs/reports/test-2026-09-20-codex-effective-model-receipt.md` line 26 documents — "ran each binary directly... from inside its own package directory") passed cleanly, `PASS`, exit 0, for both `org.test` and `cli.test`. Every place this package reads `$HOME`/`$CODEX_HOME` for codex session observation goes through the package-level seam (`o.CodexSessionsDir`), pinned to a `t.TempDir()` value by every test — none falls through to the real environment lookup.

## Coverage

- Statement: `internal/org` 90.5% (unchanged from before this cycle's test additions — the new tests reinforce discriminating power on already-covered lines, they don't newly execute previously-dead code), `internal/org/driver` 92.0%, `internal/org/protocol` 97.9% (`go test -coverprofile`, `-covermode=atomic`).
- Function (the plan's named sites): `ObserveCodexEffectiveModel` 100.0%, `isCtxDoneErr` 100.0%, `observeCodexSpawnReceipt` 100.0%, `seatFromEvents` 100.0%, `codexRolloutCandidates` 88.9%, `scanRolloutRecord` 97.3%, `Stop` 94.3%, `codexSpawnCorrelation` 94.7%, `observeStopModelReceipt` 94.7%.
- Notes on the sub-100% functions (read from source, not assumed — see `docs/evidence/test-2026-09-20-codex-model-observation-followups.log` for the raw `cover -func` lines):
  - `codexRolloutCandidates` (88.9%): three untested statement blocks — the `!isCodexRolloutFileName(name)` → `continue` branch (no test places a non-`rollout-*.jsonl`-named file inside a walked directory), the per-file `info.ModTime().Before(minModTime)` → `continue` branch (distinct from a whole date-directory being skipped by the existing "N days old" tests), and the `len(candidates) > codexObserveMaxFiles` (200) cap-trim branch. Confirmed via `git show 3d355bd:internal/org/codex_session.go` that these three lines existed **before** this plan's diff — pre-existing gaps, not introduced by the ctx-threading work. The lines this plan actually added or touched inside this function (the two new `ctx.Err()` checks) are separately confirmed 100%-exercised via the red/green mutation runs below.
  - `scanRolloutRecord` (97.3%): one untested block, `errors.Is(openErr, fs.ErrNotExist)` (the file vanishing in the Lstat→Open race) — also pre-existing (confirmed same way), a genuine race condition not practically fault-injectable without a symlink-swap harness this plan's scope does not call for.
  - `Stop`/`codexSpawnCorrelation`/`observeStopModelReceipt`'s remaining gaps are all `o.Manifest.Read()` / `o.Receipts.Read()` I/O-failure branches and the `time.Parse` malformed-TS branch — the same "untested filesystem-fault-injection branch" pattern repeated at every read call in this file (`findSeat`, `Status`, `Disband...`), not specific to this plan's changes. `Stop`'s own `o.Herdr.PaneSendKeys` failure branch is the same pre-existing pattern.
  - None of these gaps are newly introduced by this plan's diff; all were traced to source lines that existed at `3d355bd` (pre-diff HEAD) and follow the same shape as other untested I/O-fault branches already present throughout `internal/org`.

## Red/green discrimination results (issue #173 handoff item 9)

All mutations were applied in-place to the worktree, tested, then reverted with `git checkout -- <file>`; `git status --porcelain` was confirmed empty after every revert (only the three test files carry a diff at the end of this cycle — see Test gaps below).

| # | Mutation | Expected | Actual | Verdict |
| --- | --- | --- | --- | --- |
| 9a | Stop: rebuild the receipt from the roster seat (`seat.Model` etc.) instead of the correlated spawn | AC-1 test fails (false `honored=false`) | `TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected` **and** `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver` both failed | Matches (stronger: 2 tests, not just 1 — expected, since the mutation also corrupts Driver/Role) |
| 9b | Stop: decide codex-ness from `seat.Driver` instead of the correlated spawn's driver | Both AC-2 driver-mismatch tests fail | `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver` and `TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry` both failed | Matches exactly |
| 9c | `codexSpawnCorrelation`: let a `rejected` event update Model/Driver/Role | A test fails | 4 tests failed (`TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected`, `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver`, `TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry`, `TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins`) | Matches (no missing unit case needed) |
| 9d | Observer: delete each of the four `ctx.Err()` checkpoints one at a time (per-date-dir, per-entry, before-open, per-line) | Exactly the pinned test for each fails | Each deletion failed exactly one test: `TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx`, `TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory`, `TestScanRolloutRecord_BeforeOpenCheckIsPinned_AlreadyDoneCtx`, `TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch` | Matches exactly, all 4 |
| 9e | Observer: return `found` from a pass cut short after a match was already seen on an earlier candidate | Second-candidate test fails | `TestObserveCodexEffectiveModel_CtxDoneBeforeSecondCandidate_SecondFileNeverOpened` failed | Matches exactly |
| 9f | Observer: add a `ctx.Err()` check **after** the candidate loop (before the match-count switch) | Completed-pass test (`..._CtxExhaustedExactlyAsPassCompletes_FoundStillReturned`) fails | **Full suite passed (exit 0), no test failed** | **Gap found and closed** — see below |
| 9g | Spawn poll: report the read-error reason whenever ANY pass had a read error, never clear `lastErr` | A test may fail; add one if deterministic, else name the gap | **Full suite passed (exit 0), no test failed** | **Gap found and closed** — see below |
| 9h | Spawn poll: swap priority so the budget reason wins over a cancelled parent ctx | Parent-cancelled test fails | **Full suite passed (exit 0), no test failed** (`TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason` writes no fixture, so its `lastErr` stays `nil` throughout — the swap only changes behavior when both `lastErr != nil` and `ctx.Err() != nil` at once, a case that test never reaches) | **Gap found and closed** — see below |

### Gaps found and closed (9f, 9g, 9h, plus one item-10 gap)

Four genuine test-coverage gaps were found — in each case the *production* code and its doc-comment claims were already correct (confirmed by the red/green cycle below); only the *test suite* lacked a case that discriminated the specific behavior. Per the handoff's own guidance for 9g ("add one only if it can be deterministic... otherwise name it as a gap") and this task's general framing, a deterministic test was added for each, verified red against the mutation and green at HEAD, then the mutation was reverted (production code is byte-identical to `638db39` at the end of this cycle — confirmed via `git diff --stat` showing only the three test files).

1. **9f — no test proved "no `ctx.Err()` check after the candidate loop completes naturally".** `TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned` self-calibrates its budget by running `ObserveCodexEffectiveModel` itself once to learn its own call count — so an extra real `ctx.Err()` call the mutated function makes gets silently absorbed into that calibration rather than exceeding it (the mutation always lands as the new call `#budget`, still within the recalibrated budget). Added `TestObserveCodexEffectiveModel_NoCtxCheckAfterCandidateLoopCompletes` (`internal/org/codex_session_test.go`), which instead derives its budget by calling `codexRolloutCandidates` and `scanRolloutRecord` **directly** and summing their own call counts — never running `ObserveCodexEffectiveModel` to calibrate, so a stray top-level call is not part of the budget and flips the result to not-found. Confirmed red against the mutation, green at HEAD, stable.
2. **9g — no test proved a later clean pass clears an earlier pass's `lastErr`.** This is exactly the behavior commit `6b63b69` introduced; no dedicated test pinned it. Added `TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass` (`internal/org/spawn_test.go`): a fixture that is permission-denied on the first poll (genuine read error, same technique as `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason`), fixed to readable — but content that never matches `promptPath` — by a background goroutine after 15 ms (interval 5 ms, timeout 300 ms; direct unit call to `o.observeCodexSpawnReceipt`, same pattern as `TestObserveCodexSpawnReceipt_CtxDone_DistinctReason`). Asserts `codexNotFoundReason`, never `codexReadErrorReason`. Confirmed red against the mutation, green at HEAD, stable across 20 repeats and under concurrent `internal/cli` load.
3. **9h — no test proved the parent-ctx-done check outranks a genuine read error in a tie.** Added `TestOrgSpawn_Codex_ParentCtxCancelledAfterReadError_CutShortStillWins` (`internal/org/spawn_test.go`): a permission-denied fixture (genuine read error on the first poll) combined with `p.TimeoutMS = 50` (long enough to survive the saga and that first poll, short enough to expire during the 500 ms observe interval that follows) — both `lastErr != nil` and `ctx.Err() != nil` hold when the wait fails. Asserts `codexCutShortReason` wins, matching the doc comment's documented priority. Confirmed red against the mutation, green at HEAD, stable across 30 repeats and 30 repeats under concurrent `internal/cli` load.
4. **Item 10 — "Stop for a seat with no `spawn_started` at all" was exercised but not asserted on the specific claim.** `TestOrgStop_ExistingSeat_NoPaneOrAgmsgTeam_RecordsSkippedNotes` already drives `codexSpawnCorrelation`'s `ok=false` path (via bisection: it is the *only* existing test that reaches `verbs.go:785.22,787.3`, confirmed via targeted `-coverprofile` runs), but never asserts "no `model_observed=` token at all" or `StopResult.ModelReceipt == Receipt{}`. Added `TestOrgStop_LegacySpawnedEvent_NoSpawnStarted_NoModelObservedToken` (`internal/org/verbs_test.go`), naming that exact claim. Confirmed red against a targeted mutation (forcing `isCodexSpawn=true` for the zero-value correlation result, which would wrongly emit `model_observed=none`), green at HEAD.

All four new tests are additive-only; no production file differs from `638db39`.

## Additional edge-case checks (item 10)

- **A nil-equivalent ctx is never passed.** Confirmed by reading, not a runtime test: `observeCodexSpawnReceipt`'s `obsCtx := context.WithTimeout(ctx, ...)` always derives from a real caller-supplied `ctx` (Spawn's own saga ctx), and `observeStopModelReceipt`'s `obsCtx := context.WithTimeout(context.Background(), ...)` always derives from `context.Background()`. Both are real, non-nil `context.Context` values at every call site; no code path in this package constructs or passes a nil `context.Context`.
- **Observer with an already-past-deadline ctx and zero candidates.** Not previously covered as a distinct case (the existing AC-3 test, `TestObserveCodexEffectiveModel_AlreadyDoneCtx_NotFoundDespiteMatchingRecord`, uses a fixture with a matching record present). Added `TestObserveCodexEffectiveModel_AlreadyDoneCtx_ZeroCandidates_ReturnsCtxError` (`internal/org/codex_session_test.go`): `sessionsDir` exists but holds no date directories or files at all. Confirms `ObserveCodexEffectiveModel` returns ctx's own error, never a plain `nil` "clean not-found" — this matters because `codexRolloutCandidates`' per-date-directory `ctx.Err()` check runs on `codexSessionDateDirs`' own always-non-empty date list, before it ever tries (and silently fails/skips) an `os.ReadDir` on a directory that does not exist on disk. Confirmed red against the per-date-directory-check-removed mutation (same one used for 9d-1), green at HEAD. (Separately confirmed by reading: when `sessionsDir` itself does not exist, `os.Stat` fails and the function returns `(NotFound, nil)` **before ever reaching any ctx check** — this is intentional, documented behavior ("a missing sessionsDir... degrades silently to CodexObservationNotFound with a nil error"), not a gap.)
- **A correlated `spawn_started` with an empty Model → token `none`.** Already covered: `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved` asserts `assertDetailsContains(t, last.Details, "model_observed=none")` exactly. No gap.
- **`seatFromEvents` agrees with `findSeat` for dry-run-only seats.** Trivially guaranteed by construction, not by a test: `findSeat` (`verbs.go:56`) is a one-line wrapper that calls `seatFromEvents` directly on the events it just read — there is no second implementation that could disagree. No test needed.

## Failure analysis

No test failures against HEAD's actual production code in any of commands 1–8. The only failures observed during this cycle were (a) the two `TestBackfillCmd_*` tests when the `cli.test` binary was run from the wrong working directory (root-caused above, not a real failure — resolved by re-running from the correct directory) and (b) the expected red-phase failures of the 11 deliberate mutations and their corresponding new tests, all reverted.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #165's own regression suite (retry-timestamp suffix, same-second re-spawn, days-later session, dry-run exclusion) | Still passing, unchanged | `go test ./internal/org/...`, all green; no #165 test file was touched by this diff |
| AR-3 (stop compares against the roster instead of the correlated spawn) | Fixed and pinned | `TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected`, `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver`, `TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry` — all pass at HEAD, all fail under the corresponding 9a/9b/9c mutations |
| AR-4 (the observer ignores its own deadline mid-pass) | Fixed and pinned | AC-3/AC-4/AC-4b tests plus the four 9d checkpoint tests — all pass at HEAD, each fails under its own targeted mutation |

## Test gaps

- Four pre-existing (not introduced by this plan) statement-coverage gaps in `codexRolloutCandidates` and `scanRolloutRecord` — see Coverage section above. Not fixed here (out of this plan's scope; the plan's own Non-goals do not name them, and they predate this diff at `3d355bd`).
- The plan's own self-review-flagged coverage gaps (`docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`, "Coverage gaps" section) — M2's placement concern was already closed in-cycle (the three `...CheckIsPinned...` tests, confirmed still discriminating by 9d above); the "rejected-only seat stop" gap is now closed by this cycle's item-10 addition; the "M1 read-error-then-cut-short scenario" gap is now closed by this cycle's 9h addition.
- No remaining gaps specific to this plan's diff. The four generic I/O-fault-injection branches noted in Coverage are the same shape as similar untested branches throughout `internal/org` (`findSeat`, `Status`, `Disband...`) and are not this plan's concern.

## Verdict

- **Pass.** All commands 1–8 green (shell suites, `go test` across the three named packages, `-race`, `TMPDIR=/tmp`, home independence from the correct working directory, coverage). All flakiness checks (broad `-count=50`, `-race -count=20`, the two timing-shaped tests at `-count=200` and again at `-count=50` under concurrent load) reported zero failures. All five time zones produced identical results. Of the 8 requested red/green mutation checks (9a–9h), 5 (9a, 9b, 9c, 9d ×4, 9e) matched the handoff's stated expectation exactly; 3 (9f, 9g, 9h) revealed genuine test-coverage gaps — in every case the *production* code was already correct, only the test suite lacked a discriminating case — and all three were closed with new, red/green-verified, deterministic tests, plus one further item-10 gap (the legacy-no-`spawn_started`-seat "no token" claim). No production file differs from `638db39`; the diff at the end of this cycle is additive-only across three test files (`internal/org/codex_session_test.go`, `internal/org/spawn_test.go`, `internal/org/verbs_test.go`).
- Fail: none.
- Blocked: none.
