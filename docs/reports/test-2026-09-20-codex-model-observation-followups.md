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

## Cycle 2 (2026-09-20)

- Scope: same behavioral-tests-only scope, re-derived against HEAD `bca703c` (branch `fix/codex-model-observation-followups`). Precondition confirmed before starting: HEAD was `bca703cdc1e47e3c6d7f534fdaf682c558d4d865`, `git status --porcelain` was empty. This section does not carry the cycle-1 section's numbers forward as current — every number below was independently re-measured at `bca703c`.
- Delta since cycle 1 (`4300307`): test budgets only, plus two doc-comment enumerations in `internal/org/verbs.go` (comment text, no logic change — confirmed via `git diff 4300307..bca703c -- internal/org/verbs.go`, both hunks add "a receipts-file read failure" as a listed branch). `5e239b0` gave every test that must actually FIND a record a 30 s `testCodexObserveGenerousBudget` (org) / `setCodexModelObserveTimeoutOverride` (cli), replacing `testOrg`'s 1 ms default that this branch's own ctx-threading turned into a real scan-time bound, not merely a between-polls bound (cross-review AR-1). `711cd9d`/`a1afc0c` gave `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` its own 200 ms budget (a third category: the budget must run out, but only after one pass completes) — this was self-review cycle-2's C2-H1, the one flake the reviewer could reproduce (7/300 under `-race`) that the implementer and team lead could not reproduce locally. Verified per-AC status: `docs/reports/verify-2026-09-20-codex-model-observation-followups.md`'s "Cycle 2" section, verdict Pass.
- Production code since `638db39`/cycle 1: unchanged (comment-only, confirmed above). `git diff bca703c --stat` at the end of this cycle (before my own commit) is empty.

### Command results

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | multiple suites | all | 0 | 0 |
| `go test ./internal/org/... ./internal/cli/... ./internal/insights/... -count=1` | 5 packages | 5 | 0 | 0 |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 |
| Home independence (`org2.test`/`cli2.test`, freshly built, run from each package's own directory) | 2 binaries | 2 | 0 | 0 |

### Flakiness (item 6)

| Run | Result |
| --- | --- |
| `go test -race ./internal/org -run 'TestOrgStop_Codex\|TestOrgSpawn_Codex\|TestObserveCodexSpawnReceipt' -count=100` | ok, 86.2 s |
| Same, `GOMAXPROCS=1` | ok, 86.3 s |
| Same, while `go test ./internal/cli/... -count=1` runs concurrently | ok, 84.7 s foreground; background `internal/cli` suite also ok, 41.7 s |
| The four CLI codex-warning tests (`TestOrgSpawn_CLI_CodexModelMismatch_PrintsWarning`, `TestOrgSpawn_CLI_CodexModelMatch_NoWarning`, `TestOrgStop_CLI_CodexModelMismatch_PrintsWarning`, `TestOrgStop_CLI_CodexModelMatch_NoWarning`) `-count=20` while `go test ./internal/org/... -count=1` runs concurrently | 80/80 pass; background `internal/org` suite also ok, 9.7 s |

Zero failures across all four flakiness runs (280 `-race` iterations of the core codex Spawn/Stop test set, 80 iterations of the CLI warning tests, under three different contention shapes).

### Slow-machine simulation (item 7)

Throwaway `time.Sleep(d)` injected at the top of `scanRolloutRecord` (and, separately, `codexRolloutCandidates`), one `d` at a time; each run reverted with `git checkout -- internal/org/codex_session.go` before the next. `grep -c 'time.Sleep' internal/org/codex_session.go` was `0` and `git status --porcelain` was empty after every single revert, confirmed individually, not just at the end.

| Injection site | `d` | `go test ./internal/org/ -count=1` | `go test ./internal/cli/ -run 'Codex\|ModelWarning\|ModelObserv' -count=1` |
| --- | --- | --- | --- |
| `scanRolloutRecord` (top) | 3 ms | ok | ok |
| `scanRolloutRecord` (top) | 10 ms | ok | ok |
| `scanRolloutRecord` (top) | 25 ms | ok | ok |
| `scanRolloutRecord` (top) | 60 ms | ok | ok |
| `codexRolloutCandidates` (top) | 10 ms | ok | ok |

Matches the handoff's own expectation exactly: nothing fails at 3/10/25 ms, and nothing fails at 60 ms either now that the read-error test has 200 ms. Spot-checked the two tests the expectation is actually about, at 60 ms (`scanRolloutRecord`) and 10 ms (`codexRolloutCandidates`): `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` passed at 0.27 s / 0.22 s respectively (its own 200 ms budget with wide margin over one permission-denied pass), and `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` passed at 0.07 s / 0.06 s — this second one is self-review's already-accepted, already-documented C2-L2 redundancy: at 50 ms budget with a 60 ms per-scan delay, the pass is genuinely cut short, so the test passes because the outcome (`Honored != HonoredUnknown` false) is satisfied by BOTH "record correctly excluded by age" and "cut short before reading the age at all" — not a new finding, just reconfirming the self-review's own characterization holds at HEAD; the deterministic pin for the age-exclusion rule itself remains `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked` (`context.Background()`, cannot be cut short).

### Suite time (item 8)

| Package | Wall time (`-count=1`) |
| --- | --- |
| `internal/org` | 7.68 s |
| `internal/cli` | 42.08 s |

5 slowest top-level tests (`-v`, sorted by their own reported duration):

`internal/org`: `TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt` (2.01s), `TestRunWatcher_TimeoutIndependentOfSmallInterval` (1.23s), `TestWithManifestLock_SerializesConcurrentCallers` (0.49s), `TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass` (0.30s), `TestRunWatcher_UnknownSeat_StillProceeds` (0.26s). None of these are this branch's own — the top two are pre-existing watchdog tests untouched by this PR; the tester-added `..._ReadErrorClearedByLaterCleanPass` (cycle 1) is the only one from this PR's own work, at 0.30s, matching its designed ~300ms budget.

`internal/cli`: `TestNewWatchdogHooks_Dispatch_NeverBlocksCaller` (3.37s), `TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr` (2.79s), `TestRunAdoptIO_SingleFork_ResetToTemplate` (1.56s), `TestRunAdoptIO_DriftedCore_ResetToTemplate` (0.95s), `TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites` (0.91s) — none of these are this branch's own either.

Every test this branch added or re-budgeted, checked individually against 0.5 s: `TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass` 0.30s, `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` 0.20s, `TestOrgStop_CLI_CodexModelMatch_NoWarning` 0.38s, `TestOrgStop_CLI_CodexModelMismatch_PrintsWarning` 0.37s, `TestOrgSpawn_CLI_CodexModelMismatch_PrintsWarning`/`..._NoWarning` 0.28s each, `TestOrgSpawn_Codex_ParentCtxCancelledAfterReadError_CutShortStillWins` 0.06s, `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` 0.06s, `TestOrgSpawn_Codex_ModelObservation_FoundOnLaterPoll` 0.04s, and every `raiseCodexObserveBudgetForStop`/`testCodexObserveGenerousBudget`-using Stop/Spawn test at 0.00–0.01s (the observation returns at once on a match, so the 30 s constant is genuinely never waited out — confirmed, not just asserted, since every one of these times is well under 0.5 s). **No test added or re-budgeted on this branch is a mis-categorisation that waits out the 30 s constant.**

### Red/green spot checks (item 9)

All mutations applied in-place, tested, reverted with `git checkout -- <file>`; `git status --porcelain` confirmed empty after every revert.

| # | Mutation | Expected | Actual | Verdict |
| --- | --- | --- | --- | --- |
| 9a | Set `testCodexObserveGenerousBudget` to 1 ms + add a 3 ms sleep at the top of `scanRolloutRecord` | Found-tests go red | 16 tests failed: `TestOrgSpawn_Codex_ModelObservation_Found` (both subtests), `..._Ambiguous`, `..._FoundOnLaterPoll`, and 12 `TestOrgStop_Codex_*` tests that use `raiseCodexObserveBudgetForStop`/the constant directly | Matches (constant is genuinely load-bearing for every test that classifies itself as "must find") |
| 9b | Remove `raiseCodexObserveBudgetForStop(o)` from `TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown`, with the 3 ms `scanRolloutRecord` sleep in place | That one test goes red | `--- FAIL: TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown` — `expected an honored=true receipt appended at stop, got {zero Receipt}` | Matches exactly |
| 9c-i | Re-run cycle 1's 9a: build the Stop receipt from the roster seat again | `TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected` goes red | Failed: `expected the receipt's commanded model to be the ORIGINAL spawn's model, not the rejected retry's, got CommandedModel:gpt-9-nonexistent` | Matches exactly, still bites |
| 9c-ii | Re-run cycle 1's 9d (per-line check): delete the per-line `ctx.Err()` check in `scanRolloutRecord` | `TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch` goes red | Failed exactly that one test: `expected ctx's own error, got <nil>` | Matches exactly, still bites |

### Coverage (item 10)

- `internal/org` **90.5%**, `internal/org/driver` 92.0%, `internal/org/protocol` 97.9% — byte-identical to cycle 1's numbers. Nothing moved: this cycle's delta is test budgets/comments only, no new or removed production statement.
- Per-function, the plan's named sites — identical to cycle 1: `ObserveCodexEffectiveModel`/`isCtxDoneErr`/`observeCodexSpawnReceipt`/`seatFromEvents` all 100.0%; `codexRolloutCandidates` 88.9%, `scanRolloutRecord` 97.3%, `Stop` 94.3%, `codexSpawnCorrelation` 94.7%, `observeStopModelReceipt` 94.7%. The self-review cycle-2 Coverage-gaps item (`o.Receipts.Read()` failure branch, `verbs.go:936.16,938.3`, count 0) is confirmed still uncovered at HEAD (checked directly against the raw profile) — not asked for in this cycle's command list, not closed here, carried forward as an open gap (see Test gaps below).

### Test gaps (re-derived at `bca703c`)

- No production or test-suite gap specific to this plan's diff remains unaddressed by either cycle. The 4 pre-existing (predate `3d355bd`) statement-coverage gaps named in cycle 1 (`codexRolloutCandidates`'s filename-filter/ModTime/cap-trim branches, `scanRolloutRecord`'s `fs.ErrNotExist` race branch) are unchanged.
- One item carried from self-review cycle 2, not closed in this `/test` cycle (not in this cycle's command list): `observeStopModelReceipt`'s `o.Receipts.Read()` failure branch (`verbs.go:935-938`) has no dedicated test — same shape as the append-failure case, which does have one (`TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful`). Low risk (same silent-degrade contract, narrow race/corruption window), flagged for whoever next touches this file, not blocking.
- The flake class this whole cycle exists to close (self-review C2-H1 / cross-review AR-1) is confirmed closed: 280 `-race` iterations across three contention shapes, plus 5 slow-machine injection points up to 60 ms, all clean.

### Verdict (cycle 2)

- **Pass.** All 8 requested commands green, re-derived independently at `bca703c` (not carried forward from cycle 1). Flakiness clean across 280 `-race` iterations under three different contention shapes (default, `GOMAXPROCS=1`, concurrent `internal/cli` load) plus 80 iterations of the four CLI warning tests under concurrent `internal/org` load. Slow-machine simulation clean at every injection point/delay combination the handoff named (3/10/25/60 ms at `scanRolloutRecord`, 10 ms at `codexRolloutCandidates`) — matches the stated expectation exactly, and the specific test the cycle exists to fix (`TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason`) stayed green at 60 ms with wide margin (0.22–0.27s against its 200ms budget). Suite timing: `internal/org` 7.68s, `internal/cli` 42.08s, no test from this branch is a mis-categorisation waiting out the 30s constant. All four red/green spot checks (9a, 9b, 9c-i, 9c-ii) still bite exactly as designed. Coverage flat at 90.5%/92.0%/97.9%, no regression. One pre-existing (self-review-flagged, not this cycle's scope) coverage gap remains open, named above.
- Fail: none.
- Blocked: none.
