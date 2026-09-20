# Test report: codex-effective-model-receipt

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-effective-model-receipt.md` (issue #165)
- Tester: `tester` subagent (Claude Code), standard flow, pipeline cycle 1 of cap 2
- Scope: behavioral tests only for `git diff main...HEAD` on `feat/codex-effective-model-receipt`, base `642214d`, plan/self-review/verify HEAD `3031afc`. No static analysis, diff quality, or spec-compliance review — those are `/self-review`'s and `/verify`'s jobs (self-review: MERGE after cycle-1 fixes; verify: PASS, both already on record).
- Evidence: `docs/evidence/test-2026-09-20-codex-effective-model-receipt.log` (gitignored per `docs/evidence/*.log`, written locally; commands and results also reproduced below)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (28 shell test files + golang verifier) | 636+ shell assertions across 25 suites with numeric roll-ups (3 more suites report a bare OK) + 8 Go packages | all | 0 | 0 | 1m46.8s |
| `go test ./internal/org/... ./internal/cli/... ./internal/insights/... -count=1 -v` | 5 packages | 5 | 0 | 0 | 44.9s |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 | 8.9s |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | 41.7s |
| Home independence: built `org.test`/`cli.test`, ran with `HOME=/nonexistent-home-for-test CODEX_HOME=` | 2 binaries | 2 | 0 | 0 | — |
| `go test ./internal/org/ -run 'Spawn\|Stop\|Codex' -count=50` | 1 package × 50 iterations | 50/50 | 0 | 0 | 25.4s |
| `go test -race ./internal/org/ -run 'Codex' -count=20` | 1 package × 20 iterations | 20/20 | 0 | 0 | 5.0s |
| `go test ./internal/cli/ -run 'Codex\|ModelWarning\|ModelObserv' -count=10` (foreground) **while** `go test ./internal/cli/... -count=1` (background, full suite) | 1 narrow test × 10 iterations + 1 full-package run, concurrent | 10/10 + full suite | 0 | 0 | 29.9s (foreground) / 42.5s (background) |

All eight commands from the hand-off ran clean, both before and after the one test I added (`TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins`, `internal/org/verbs_test.go`). No test failed in any run at any point in this cycle.

### Home independence detail

Built with `go test -c -o <bin> ./internal/org/` and `./internal/cli/`, then ran each binary directly (bypassing `go test`'s own build-cache HOME dependency) from inside its own package directory with `HOME=/nonexistent-home-for-test CODEX_HOME=`. Both printed `PASS` and exited 0. `internal/cli`'s suite does not need a real `HOME` either — every place that would otherwise read `$HOME`/`CODEX_HOME` for codex session observation is already routed through the package-level seam (`orgCodexSessionsDirOverride` etc., pinned by `TestMain`), and the pre-existing (unrelated) tests that scaffold projects or write config write entirely under `t.TempDir()`.

## Coverage

- Statement: `internal/org` 90.0% (org/driver 92.0%, org/protocol 97.9%), `internal/cli` 84.3%. Both packages' totals rose from this plan's own diff (new, well-tested code) — I did not have a pre-PR baseline profile to diff against in this session, but the plan's own self-review/verify reports record `internal/cli` at 82.8% and `internal/org` at 89.8%/89.6%/89.1% in prior, unrelated cycles (see the tester agent-memory `go_test_packages.md` snapshot dated 2026-09-19), consistent with a rise here.
- Branch/Function: per-function `go tool cover -func` for every function the hand-off named — see table below. Two profiles generated outside the repo (`/private/tmp/.../scratchpad/cov/{org,cli}.out`), re-generated once more after adding the new test (same numbers below, since the addition closes a *behavioral* gap that statement coverage can't see — explained under `codexSpawnCorrelation`).
- Notes: see "Named uncovered branches" below for every function under 100%.

### Per-function coverage (as requested)

| Function | File | Coverage |
| --- | --- | --- |
| `CodexSessionsDir` | `internal/org/codex_session.go` | 100.0% |
| `ObserveCodexEffectiveModel` | `internal/org/codex_session.go` | 100.0% |
| `codexRolloutCandidates` | `internal/org/codex_session.go` | 87.0% |
| `isCodexRolloutFileName` | `internal/org/codex_session.go` | 100.0% |
| `codexSessionDateDirs` | `internal/org/codex_session.go` | 100.0% |
| `truncateToLocalDay` | `internal/org/codex_session.go` | 100.0% |
| `scanRolloutRecord` | `internal/org/codex_session.go` | 97.0% |
| `codexSessionMetaQualifies` | `internal/org/codex_session.go` | 71.4% |
| `codexTurnContextModel` | `internal/org/codex_session.go` | 75.0% |
| `codexResponseItemMentionsPrompt` | `internal/org/codex_session.go` | 90.0% |
| `(o *Org) codexSessionsDir` | `internal/org/spawn.go` | 22.2% |
| `observeCodexSpawnReceipt` | `internal/org/spawn.go` | 100.0% |
| `codexFoundReceipt` | `internal/org/spawn.go` | 100.0% |
| `codexSpawnCorrelation` | `internal/org/verbs.go` | 88.9% |
| `hasObservedCodexReceiptSince` | `internal/org/verbs.go` | 75.0% |
| `observeStopModelReceipt` | `internal/org/verbs.go` | 88.9% |
| `parseCodexModelUpgrade` | `internal/cli/doctor_codex_models.go` | 100.0% |
| `codexRetirementClause` | `internal/cli/doctor_codex_models.go` | 100.0% |
| `checkCodexModelSlugs` | `internal/cli/doctor_codex_models.go` | 97.9% |
| `readCodexModelsCache` | `internal/cli/doctor_codex_models.go` | 84.2% (pre-existing, unrelated to this plan) |
| `printCodexModelMismatchWarning` | `internal/cli/org.go` | 100.0% |

### Named uncovered branches (read by hand, not just the tool's %)

- `codexRolloutCandidates` (87.0%): two untested, defensive branches — the `codexObserveMaxFiles` (200-file) truncation cap, and a file in a date directory that does not match `isCodexRolloutFileName` being skipped. Both are backstops for conditions the fixtures never need to construct; low risk.
- `scanRolloutRecord` (97.0%): the genuine (non-EOF) `readErr != nil` branch is the only line I could not place a fixture-based test on; it degrades the same way the far-more-common EOF case does, so this is cosmetic.
- `codexSessionMetaQualifies` (71.4%) / `codexTurnContextModel` (75.0%) / `codexResponseItemMentionsPrompt` (90.0%): each has an untested `json.Unmarshal(payload, ...) != nil` defensive branch, reachable only by a payload that is valid top-level JSON but the wrong *shape* for that line type (e.g. `"payload": "a bare string"` for a `turn_context` line). Every existing "malformed line" fixture (`TestObserveCodexEffectiveModel_MalformedLineSkipped`) is invalid at the outer envelope level instead, so it never reaches these three functions at all — they degrade the same way either way. `codexSessionMetaQualifies` additionally has an untested `time.Parse` failure (a `timestamp` string that isn't RFC3339), and `codexResponseItemMentionsPrompt` has an untested `p.Type != "message"` sub-condition (only `p.Role != "user"` is exercised, via the assistant-message fixture). All four are same-shape defensive branches; not a behavioral hole named in the plan's edge-case list, so I named them here rather than writing four more malformed-payload fixtures.
- `(o *Org) codexSessionsDir` (22.2%, **not** the exported `CodexSessionsDir` func, which is 100%): by design, `testOrg`'s own doc comment states every test in the package pins `Org.CodexSessionsDir` to a seam value specifically so no test ever falls through to the real `os.Getenv("CODEX_HOME")`/`os.UserHomeDir()` resolution — that is AC-8 working as intended, not a gap. The uncovered lines are exactly that fallback path.
- `codexSpawnCorrelation` (88.9%, **unchanged by the test I added** — see below): `startedIdx == -1` (no `spawn_started` at all for this seat) is unreachable via `Stop`, since `Stop` only calls this for a seat already in the roster, which implies at least one `spawn_started` exists. The `time.Parse` failure on the `spawn_started`'s own TS is likewise unreachable in practice — every TS is machine-written via `o.now()`, always valid RFC3339. Both are defensive-only.
- `hasObservedCodexReceiptSince` (75.0%): two likely-uncovered branches — a receipt belonging to a *different* org_id/seat_id (filtered out; only ever exercised with single-seat fixtures today), and a receipt whose TS is *before* `spawnStartedTS` (a stale receipt left by a prior spawn attempt of the same seat, which must not suppress the current spawn's Stop-time observation — the AC-2b analog for this suppression check). The second one is the most substantive gap I found in this whole pass; I did not add a test for it because it wasn't in the plan's or hand-off's named edge-case list, but it is a real, not-fully-defensive branch and worth a follow-up.
- `observeStopModelReceipt` (88.9%): the `Manifest.Read()`/`Receipts.Read()` error-return branches are untested — both require the store to have already succeeded once earlier in the same `Stop` call and then fail on a second read, a narrow race/corruption window. Same spirit as the existing `TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful` (a write failure), but for a read; low risk.
- `checkCodexModelSlugs` (97.9%): the one gap is `codexModelsCachePath()`'s own error branch (`CODEX_HOME` empty and home directory unresolvable) — a pre-existing environment-resolution edge case, same shape as `codexSessionsDir`'s gap above, not introduced by this plan.
- `readCodexModelsCache` (84.2%): pre-existing function, outside this plan's changed-files scope for AC-7; not investigated further here.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

No test failed in any command, run, or repetition in this cycle.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC-4: claude-seat and dry-run receipts keep their exact pre-PR text | Still fixed | `TestOrgSpawn_Claude_ModelReceiptTextUnchanged`, `TestOrgSpawn_Codex_DryRun_ModelReceiptUnchanged` pass (part of the full `internal/org` run above) |
| Pre-existing watcher/reject/dry-run receipts behavior | Unchanged | `internal/org` full suite green (`TestRunWatcher_*`, `TestOrgSpawn_*Rejected*`, `TestOrgSpawn_*DryRun*` all pass) |
| `ralph insights` receipts aggregation (tri-state honored, multi-org/seat ordering) | Unchanged | `internal/insights` full suite green, no changes in this plan's diff touch that package |
| Self-review cycle-1 M2 fix (dry-run respawn must not displace real Stop correlation) | Still fixed | `TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation` passes; red/green 9e (below) confirms it fails without the fix |

## Test gaps

- `hasObservedCodexReceiptSince`'s "stale receipt from a prior spawn attempt must not suppress the current spawn's Stop-time observation" branch (see Coverage section) — the most substantive gap found, not closed in this cycle since it wasn't in the plan's/hand-off's named edge-case list.
- The four sibling `json.Unmarshal(payload, ...)`/`time.Parse` defensive branches in `codex_session.go` (see Coverage section) — same shape, low risk, not closed here.
- `observeStopModelReceipt`'s two store-read-failure branches (narrow race window) — same shape as an existing write-failure test but for a read; not closed here.

## Red/green spot checks

All done on a throwaway basis: edit → run the named test(s) expecting a failure → `git checkout -- <file>` → confirm `git status --porcelain` is empty. Every check below left the tree clean (confirmed after each individual check, not just at the end).

| # | Change | Test(s) run | Result |
| --- | --- | --- | --- |
| 9a | `scanRolloutRecord`: `ageQualifies` forced to always `true` (drop the session_meta age condition) | `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked`, `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` | Both FAILED as expected (old/stale record picked up) |
| 9b | `codexResponseItemMentionsPrompt`: matched on the bare `promptPath` instead of `promptFilePointer(promptPath)` | `TestObserveCodexEffectiveModel_PathOnlyQuoteInUserMessageDoesNotMatch` | FAILED as expected (a bare path quote now matched) |
| 9c | `ObserveCodexEffectiveModel`: the `default:` (ambiguous, 2+ matches) case changed to return the first match instead | `TestObserveCodexEffectiveModel_TwoQualifyingRecordsAmbiguous` | FAILED as expected (returned found/first-model instead of ambiguous) |
| 9d | `hasObservedCodexReceiptSince`: return `true` for any receipt since the spawn, dropping the `ReportedEffectiveModel != ""` condition | `TestOrgStop_Codex_AppendsWhenOnlyRejectionReceiptExistsSince` | FAILED as expected (a rejection receipt with no model now wrongly suppressed observation) |
| 9e | `codexSpawnCorrelation`: removed the `ev.DryRun` filter in the `spawn_started` scan | `TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation` | FAILED as expected (this is the self-review cycle-1 M2 regression guard — confirmed it actually guards the fix) |
| 9f | `observeCodexSpawnReceipt`: removed the `promptPath == ""` early return, letting a no-role-prompt-file seat poll anyway | `TestOrgSpawn_Codex_InlinePromptSkipsObservation_NoWaiting` | FAILED as expected — the seat polled for the full 2s timeout (`spawn took 2.01s`) before returning a generic "no session record" unknown, instead of returning immediately with the "no role-prompt file" reason |
| 9g | `printCodexModelMismatchWarning`: dropped the `r.ReportedEffectiveModel == ""` condition, warning on every `honored=false` | `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning` | FAILED as expected (an envelope rejection, which has no reported model, now wrongly printed the model-mismatch warning) |
| 9h | (read-only, no edit) | `TestObserveCodexEffectiveModel_PermissionDeniedDoesNotLeakContent` | See below |

### 9h detail (privacy test, read not edited)

The test writes a rollout fixture whose user-message body contains a distinctive sentinel string, `chmod 0o000`s the file so `os.Open` fails, then asserts `err != nil`, `!strings.Contains(err.Error(), sentinel)`, and `obs.Status == CodexObservationNotFound`. This is the *only* place in the whole file where `ObserveCodexEffectiveModel` can return a non-nil error at all — every other failure mode (missing dir, malformed JSON, oversized line, a vanished file) degrades silently to `not-found, nil` by the function's own documented contract (`codex_session.go`'s doc comments on `ObserveCodexEffectiveModel` and `scanRolloutRecord`). Since the Open-failure error is a plain wrapped `os.PathError` (operation + path only, constructed *before* any read happens), and the test asserts against the exact sentinel string embedded in the fixture body, it would catch a regression that leaked that body's content into the error text through this one path. It would not catch a leak that used different wording than the sentinel, but by the function's own design this is the only code path capable of surfacing file content in an error at all, so there is no other leak surface this test needs to cover.

## Edge cases named in the hand-off (item 10)

| Edge case | Status |
| --- | --- |
| a. Record with prompt message present but `turn_context` missing yet | Covered at the observer unit level (`TestObserveCodexEffectiveModel_SessionMetaWithoutTurnContext`); the Spawn-level timeout-then-unknown behavior is identical regardless of *why* nothing was found, already covered by `TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout` — an integration-level duplicate would test nothing new |
| b. Record appearing on a later poll | Covered: `TestOrgSpawn_Codex_ModelObservation_FoundOnLaterPoll` |
| c. Sessions dir missing | Covered: `TestObserveCodexEffectiveModel_SessionsDirMissing` |
| d. Unreadable record | Covered at both levels: `TestObserveCodexEffectiveModel_PermissionDeniedDoesNotLeakContent` (observer), `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` (Spawn) |
| e. ctx cancelled during the poll | Covered: `TestObserveCodexSpawnReceipt_CtxDone_DistinctReason` |
| f. Stop when the manifest has two **real** spawns of the same seat | **Gap found and closed** — added `TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins` (see below) |
| g. Stop when the receipts file is unwritable | Covered: `TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful` (Stop still succeeds, `model_observed=none`) |
| h. doctor with `upgrade` of wrong JSON type, empty model, unparsable date, retired (past) date | Fully covered: `TestCodexRetirementClause`'s table already has cases for every one of these (string/number/array wrong-type, empty model, unparsable `retirement_at`, past-date "retired", plus the instant-equals-now edge and multi-slug join) |

### Test added: `TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins`

`internal/org/verbs_test.go` (after `TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation`). The existing dry-run test only proves a *dry-run* respawn can't displace the correlation; it does not prove `codexSpawnCorrelation` correctly picks the **later** of two **real** `spawn_started` events for the same `org_id`/`seat_id` (the actual "stop, then respawn for real" case). I unit-tested `codexSpawnCorrelation` directly (hand-built `[]ManifestEvent` with two real spawn/spawn_step pairs) rather than driving it through a full `Spawn`→`Stop`→`Spawn`→`Stop` round trip, since that's the smallest setup that reaches the exact behavior. Confirmed the test actually discriminates: temporarily changed the `startedIdx` tracking from "always overwrite on match" (last wins) to "only set once" (first wins) — the test failed (`expected the LATER real spawn_started's TS, got "2026-09-18T07:00:00Z"`), then reverted (`git checkout -- internal/org/verbs.go`, confirmed `git status --porcelain` empty). Note this addition does **not** move `codexSpawnCorrelation`'s statement-coverage % (88.9% before and after): the discriminating line (`startedIdx = i`) already executes in every existing single-spawn test, so Go's line coverage can't see "which of *multiple* matches wins" — this was a real behavioral gap invisible to the coverage number, only found by reading the code against the plan's own edge-case list.

## Verdict

- Pass: yes — all listed commands, all flakiness repetitions (50×, 20×, 10× under concurrent load), home-independence check, and all seven red/green discrimination checks (9a-9g) behaved exactly as expected; the one test I added is itself proven to discriminate.
- Fail: none.
- Blocked: none.

Proceeding to `/pr` is appropriate on this report alone (tests pass); `/sync-docs` and `/cross-review` remain the pipeline's next steps per the canonical order.

## Cycle 2 (2026-09-20)

- Tester: `tester` subagent (Claude Code), standard flow, pipeline cycle 2 of cap 2 (the final cycle).
- Scope: behavioral tests only for `git diff main...HEAD` on `feat/codex-effective-model-receipt`, re-derived against HEAD `c47370d` (working tree clean at start and end). The cycle-1 section above describes HEAD `a7180ca` and names `hasObservedCodexReceiptSince`/its stale-receipt gap — both are gone at this HEAD (renamed to `hasObservedCodexReceiptAfter`, and the stale-receipt gap was closed by `771e446`'s table test the same day, per self-review cycle-2 finding C2-6). Nothing in this section is carried forward from cycle 1; every row is re-derived from scratch against `c47370d`.
- What changed since the cycle-1 tester commit (`a7180ca`): `771e446` (table test for the receipt-age rule); `d24a030`, cross-review cycle 1 AR-1/AR-2 fixes (`promptPathFromAgentStartedDetails` strips a trailing ` agent_start_retries=<digits>` suffix; `hasObservedCodexReceiptAfter` requires strictly-after, not at-or-after); `53f6b16`, self-review cycle-2 fixes (`ObserveCodexEffectiveModel` gains an `until` parameter and `codexObserveMaxDateDirs = 32` cap; `reject()` only sets `ModelReceipt` when the append succeeded; `PromptFilePointer` exported). Confirmed via `git diff a4f6db2 c47370d --stat` that everything after the verify cycle-2 commit is docs/report/insights only — no further code drift.
- Evidence: `docs/evidence/test-2026-09-20-codex-effective-model-receipt-cycle2.log` (gitignored, written locally; commands and results also reproduced below)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (28 shell test files + golang verifier) | Same shape as cycle 1 (636+ shell assertions across 25 numeric-rollup suites, 3 more report a bare OK) + 8 Go packages | all | 0 | 0 | 1m39.3s |
| `go test ./internal/org/... ./internal/cli/... ./internal/insights/... -count=1` | 5 packages | 5 | 0 | 0 | 45.6s |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 | 8.5s |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | 42.3s |
| Home independence: built fresh `org.test`/`cli.test` (production code changed since cycle 1), ran with `HOME=/nonexistent-home-for-test CODEX_HOME=` | 2 binaries | 2 | 0 | 0 | — |
| `go test ./internal/org/ -run 'Spawn\|Stop\|Codex\|HasObserved\|PromptPath' -count=50` | 1 package × 50 iterations | 50/50 | 0 | 0 | 28.0s |
| `go test -race ./internal/org/ -run 'Codex\|HasObserved\|PromptPath' -count=20` | 1 package × 20 iterations | 20/20 | 0 | 0 | 6.1s |
| `go test ./internal/cli/ -run 'Codex\|ModelWarning\|ModelObserv' -count=10` (foreground) **while** `go test ./internal/cli/... -count=1` (background, full suite) | 1 narrow test × 10 iterations + full package, concurrent | 10/10 + full suite | 0 | 0 | 31.6s (foreground) / 45.3s (background) |
| Timezone robustness (5 zones × `-run 'Codex\|DateDir\|Stop_Codex' -count=1`) | 5 zones × 81 subtests (57 top-level) | all 5 zones identical: 57/57 | 0 | 0 | ~0.5s each |

Zero failures in any command, run, or repetition this cycle. The two test files I added to (`codex_session_test.go`, `verbs_test.go`) are the only changes in every run above — production code (`codex_session.go`, `spawn.go`, `verbs.go`) is byte-identical to `c47370d` at every point tests actually ran (confirmed via `git status --porcelain` after every red/green revert, and again at the end of this cycle).

### Timezone robustness detail

Ran `TZ=UTC`, `TZ=Asia/Tokyo`, `TZ=America/Los_Angeles`, `TZ=Pacific/Kiritimati` (UTC+14), `TZ=Pacific/Pago_Pago` (UTC-11), each `go test ./internal/org/ -run 'Codex|DateDir|Stop_Codex' -count=1 -v`. All five zones produced byte-identical results: 81 subtests run, 57 top-level `--- PASS`, 0 `--- FAIL`, exit 0. The date-window logic (`codexSessionDateDirs`, `truncateToLocalDay`) converts every timestamp to `.Local()` before truncating to a calendar day, so the UTC+14/UTC-11 extremes are the most likely zones to expose a day-boundary mismatch between a record's UTC timestamp and its local-date directory name — none did, across every fixture in the package's existing suite plus the two new date-window tests added this cycle.

### Coverage

- Statement: `internal/org` 90.3% (org/driver 92.0%, org/protocol 97.9%, up from cycle 1's 90.0%/92.0%/97.9% — the rise is `internal/org` only, from this cycle's cross-review/self-review fixes' own test coverage), `internal/cli` 84.3% (flat vs. cycle 1 — `internal/cli/org.go` and `doctor_codex_models.go` are untouched this cycle, confirmed via `git diff ccd3d2f..HEAD` being empty for both files per the verify cycle-2 report).
- Per-function, re-derived at `c47370d` (profiles outside the repo, regenerated once more after adding this cycle's own tests — identical numbers before and after, see the note on `codexSpawnCorrelation`-style coverage-blindness below):

| Function | File | Coverage | vs. cycle 1 |
| --- | --- | --- | --- |
| `CodexSessionsDir` | `codex_session.go` | 100.0% | unchanged |
| `ObserveCodexEffectiveModel` | `codex_session.go` | 100.0% | unchanged (signature grew an `until` param) |
| `codexRolloutCandidates` | `codex_session.go` | 87.0% | unchanged |
| `isCodexRolloutFileName` | `codex_session.go` | 100.0% | unchanged |
| `codexSessionDateDirs` | `codex_session.go` | 100.0% | unchanged (function itself rewritten this cycle: `until` param, cap loop) |
| `truncateToLocalDay` | `codex_session.go` | 100.0% | unchanged |
| `scanRolloutRecord` | `codex_session.go` | 97.0% | unchanged |
| `codexSessionMetaQualifies` | `codex_session.go` | 71.4% | unchanged (untouched this cycle) |
| `codexTurnContextModel` | `codex_session.go` | 75.0% | unchanged (untouched this cycle) |
| `codexResponseItemMentionsPrompt` | `codex_session.go` | 90.0% | unchanged (untouched this cycle) |
| `(o *Org) codexSessionsDir` | `spawn.go` | 22.2% | unchanged (untouched this cycle) |
| `observeCodexSpawnReceipt` | `spawn.go` | 100.0% | unchanged |
| `codexFoundReceipt` | `spawn.go` | 100.0% | unchanged (untouched this cycle) |
| `reject` | `spawn.go` | 100.0% | new to this hand-off's list; C2-3's own fix, fully covered by `TestOrgSpawn_Reject_ReceiptsAppendFailureLeavesModelReceiptZero` |
| `codexSpawnCorrelation` | `verbs.go` | 88.9% | unchanged |
| `promptPathFromAgentStartedDetails` | `verbs.go` | 100.0% | new function this cycle, fully covered |
| `isAllDigits` | `verbs.go` | 100.0% | new function this cycle, fully covered |
| `hasObservedCodexReceiptAfter` | `verbs.go` | **100.0%** | **was 75.0% as `hasObservedCodexReceiptSince` in cycle 1 — closed by `771e446`'s table test (`TestHasObservedCodexReceiptAfter_OnlyThisSpawnsObservedReceiptCounts`), which now explicitly covers the other-org/other-seat filtering and the same-second-receipt exclusion. Not re-filed as a gap per this hand-off's instruction.** |
| `observeStopModelReceipt` | `verbs.go` | 88.9% | unchanged |

`internal/cli`'s codex-related functions (`doctor_codex_models.go`, `org.go`'s `printCodexModelMismatchWarning`) are byte-identical to cycle 1's table — confirmed via a fresh `go tool cover -func` pass, not re-listed here since nothing changed.

#### Named uncovered branches, re-derived at `c47370d`

- `codexRolloutCandidates` (87.0%), `scanRolloutRecord` (97.0%), `codexSessionMetaQualifies` (71.4%), `codexTurnContextModel` (75.0%), `codexResponseItemMentionsPrompt` (90.0%), `(o *Org) codexSessionsDir` (22.2%), `codexSpawnCorrelation` (88.9%), `observeStopModelReceipt` (88.9%): all unchanged from cycle 1, all in files/functions untouched by this cycle's diff (or, for `codexSpawnCorrelation`/`observeStopModelReceipt`, touched only by the strictly-after comparison and the `until` threading, neither of which opens a new branch in the specific lines still uncovered). Same reasoning as cycle 1's report stands for each — not re-derived line-by-line a second time since nothing about them changed; see that section above.
- **Closed since cycle 1**: `hasObservedCodexReceiptAfter`'s org/seat-filtering and same-second-exclusion branches (see table above) — this was cycle 1's own flagged "most substantive gap"; per the hand-off, not re-filed.
- **New, still open**: none of this cycle's new code (`reject`'s success/failure branches, `promptPathFromAgentStartedDetails`'s stripping logic, `isAllDigits`) has an uncovered branch — all three are 100%. The `codexSessionDateDirs`/`ObserveCodexEffectiveModel` `until`-widening logic is also 100% at the function level; item 10's edge-case review below is what found the two real gaps in that area (both now closed by tests added this cycle), not the coverage tool.

### Red/green spot checks

All done on a throwaway basis: edit production code → run the named test(s) expecting a failure → `git checkout -- <file>` → confirm `git status --porcelain` after each individual check. Every check left the tree clean (only the two test files' additions ever remained).

| # | Change | Test(s) run | Result |
| --- | --- | --- | --- |
| 9a | `promptPathFromAgentStartedDetails`: `strings.LastIndex` → `strings.Index` (first occurrence, not last) | `TestPromptPathFromAgentStartedDetails` (existing table) | **All 9 pre-existing cases still PASSED — the change did not discriminate.** See finding below; a new 10th case was added and does discriminate. |
| 9b | `promptPathFromAgentStartedDetails`: stop stripping altogether (return `rest, true` unconditionally) | `TestOrgStop_Codex_RecoversPromptPathAfterAgentStartRetry` | FAILED as expected |
| 9c | `hasObservedCodexReceiptAfter`: `r.TS <= spawnStartedTS` → `r.TS < spawnStartedTS` (back to at-or-after) | `TestHasObservedCodexReceiptAfter_OnlyThisSpawnsObservedReceiptCounts` (the "same-second" case) + `TestOrgStop_Codex_SameSecondRespawn_PreviousReceiptDoesNotSuppressObservation` | Both FAILED as expected |
| 9d | `observeStopModelReceipt`: pass `spawnStartedAt` as `until` instead of `o.nowTime()` | `TestOrgStop_Codex_ObservesSessionThatStartedDaysAfterSpawn` | FAILED as expected |
| 9e | `codexSessionDateDirs`: removed the `codexObserveMaxDateDirs` cap-break | `TestCodexSessionDateDirs_CapKeepsEarliestDirectories` | FAILED as expected (`len(got) = 63, want 32`) |
| 9f | `codexSessionDateDirs`: keep the LATEST 32 directories instead of the earliest | `TestCodexSessionDateDirs_CapKeepsEarliestDirectories` (same existing test) | FAILED as expected (`got[0] = "2026/01/31", want "2025/12/31"`) — the existing test's per-element identity check catches this without needing a new one |
| 9g | `reject()`: set `ModelReceipt` unconditionally, ignoring the append error | `TestOrgSpawn_Reject_ReceiptsAppendFailureLeavesModelReceiptZero` | FAILED as expected |
| 9h-1 | (cycle-1 re-check) `scanRolloutRecord`: `ageQualifies` forced `true` | `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked`, `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` | Both still FAILED as expected — the fix still bites after the `until`/cap refactor |
| 9h-2 | (cycle-1 re-check) `codexResponseItemMentionsPrompt`: matched on the bare path instead of `PromptFilePointer(promptPath)` | `TestObserveCodexEffectiveModel_PathOnlyQuoteInUserMessageDoesNotMatch` | Still FAILED as expected |

#### 9a finding: the existing table test does not actually discriminate LastIndex vs. Index

Neither of the table's two "suffix key in the middle" cases (`"path contains the suffix key in the middle, plus a real trailing suffix"` / `"...no real trailing suffix"`) has a second *space-prefixed* `" agent_start_retries="` match — in both fixtures the embedded key text is preceded by `/`, not a space, so it never matches `marker` at all, regardless of whether `strings.LastIndex` or `strings.Index` is used. Both find the exact same single (real, trailing) occurrence, so switching them produces byte-identical output for every existing case — the table's stated intent ("distinguish first from last occurrence") was not actually being exercised. I added a 10th case, `"two real retries-suffix matches: only the last is stripped"` (`details = prefix + "/state/x agent_start_retries=42/y.md agent_start_retries=7"`), constructed so a first-occurrence implementation reads `"42/y.md agent_start_retries=7"` as its "digits" (fails `isAllDigits`, so it gives up stripping anything, leaving the whole trailing suffix in the "path"), while the correct last-occurrence implementation strips only the real trailing suffix. Confirmed it passes against the real code and fails against the `Index` variant (see the run above) before reverting the production edit.

### Edge cases from the hand-off (item 10)

| Edge case | Status |
| --- | --- |
| `until` equal to spawnStarted | Already covered: `TestObserveCodexEffectiveModel_UntilEqualsSpawnStarted_LateSessionNotFound` documents this as Spawn's own default, and every `observeCodexSpawnReceipt`-level Spawn test exercises it implicitly (Spawn always passes `spawnStartedAt` for both parameters) |
| `until` zero | **Gap found and closed** — the doc comment calls out "including the zero value" by name, but every existing test used "10 hours before", never the literal zero `time.Time{}`. Added `TestCodexSessionDateDirs_UntilZeroValue_TreatedAsSpawnStarted` |
| `until` years later (cap holds, no long loop) | **Gap found and closed** — the existing cap test only used a 60-day span. Added `TestCodexSessionDateDirs_UntilYearsLater_CapHoldsAndReturnsQuickly` (3-year span, asserts both `len(got) == codexObserveMaxDateDirs` and wall time under 100ms) |
| A retry suffix with a huge digit string | **Gap found and closed** — added a 34-digit case to `TestPromptPathFromAgentStartedDetails`'s table. No overflow risk by construction (`isAllDigits` never parses the digits into a number), but the case pins that fact as a regression guard rather than leaving it as an inference |
| Details equal to exactly the prefix (empty path): Stop must not observe | Effectively covered, not independently addable: `promptPathFromAgentStartedDetails`'s own "prefix with an empty path" table case already proves `(path="", ok=true)`; `observeStopModelReceipt`'s `promptPath == ""` early-return is already exercised (via a different input shape — an inline prompt with no prefix at all) by `TestOrgStop_Codex_NoPromptFileSeat_NothingAppended`, which hits the exact same guard clause. The one narrower thing this input shape would additionally exercise — `codexSpawnCorrelation`'s loop assigning `promptPath = ""` explicitly (`found=true, path=""`) rather than leaving it at its zero-value default (`found=false`) — is unobservable from any caller and behaviorally identical either way, so I judged a dedicated test not worth adding |
| A respawn whose previous spawn's record sits in a directory the wider stop window now also walks; AC-2b must still exclude it | **Investigated and closed with a defense-in-depth test**, see below |

#### The C2-1/AC-2b interaction, worked through

`codexSessionDateDirs`'s window is `[spawnStarted's local day − 1, until's local day + 1]` — the **lower** bound is fixed at `spawnStarted`'s own day and is entirely independent of `until`; only the upper bound widens. Codex's own convention (confirmed in this plan's earlier evidence gathering) is that a record lives in the directory of the day its *own* session started. So a genuinely stale record — one whose `session_meta` predates the *current* spawn's `spawnStarted` — necessarily started on or before `spawnStarted`'s own day, meaning its directory was always inside the original (pre-C2-1) three-directory window; C2-1's widening, which only extends forward, cannot make such a record newly reachable. In the concrete "respawn" framing (an earlier, already-stopped spawn attempt of the same seat, whose real session record predates the current spawn by construction), this means the literal risk as phrased does not arise from the code's actual bounds.

That argument rests on codex's own directory convention holding, which is exactly the kind of external, undocumented contract this whole plan already treats as capable of drifting (`docs/tech-debt/README.md`'s own noted risk). So rather than resting on the argument alone, I added `TestObserveCodexEffectiveModel_WidenedWindowStillExcludesStaleRecord`: a record whose `session_meta` predates `spawnStarted` (AC-2b's disqualifying condition) but is deliberately filed under a directory 5 days *after* `spawnStarted`'s own day — a shape real codex never produces, but the only way to make "reachable only via the widened window" and "stale" simultaneously true in a fixture — alongside a real, correctly-dated, in-window record. With `until` reaching 10 days past `spawnStarted` (so the widened window does walk the misplaced directory), the result is still `Found`/the valid record's model, not `Ambiguous` and not the stale record's model: `codexRolloutCandidates`'s directory walk is a reachability pre-filter only, never a qualification decision — `scanRolloutRecord`'s own `session_meta` check is what actually decides, and it is untouched by which directory a candidate came from. This is a genuine regression guard for the underlying safety property (not just for the specific "respawn" framing), and it would fail if a future change ever let directory membership substitute for the `session_meta` check.

### Tests added this cycle

All in `internal/org`, all confirmed to pass against `c47370d` and (where the hand-off asked for a red/green proof) confirmed to fail against a deliberately broken production edit before that edit was reverted:

- `internal/org/verbs_test.go`: one new table case in `TestPromptPathFromAgentStartedDetails` ("two real retries-suffix matches: only the last is stripped") closing the 9a finding above; one new table case ("retries suffix with an implausibly large digit string") closing the huge-digit-string edge case.
- `internal/org/codex_session_test.go`: `TestCodexSessionDateDirs_UntilZeroValue_TreatedAsSpawnStarted`, `TestCodexSessionDateDirs_UntilYearsLater_CapHoldsAndReturnsQuickly`, `TestObserveCodexEffectiveModel_WidenedWindowStillExcludesStaleRecord`.

`gofmt -l` and `go vet ./internal/org/...` on both changed files: clean. Re-ran the full `./scripts/run-test.sh` and the targeted `go test ./internal/org/... ./internal/cli/... ./internal/insights/... -count=1` once more with these tests in place: both green (see Test execution table above).

### Verdict (Cycle 2)

- Pass: yes — every command from the hand-off ran clean at `c47370d`; all flakiness repetitions (50×, 20×, 10× under concurrent load) and all five timezone runs were failure-free; the `hasObservedCodexReceiptSince` gap named in cycle 1 is confirmed closed and not re-filed; all nine red/green checks (9a-9g, 9h-1/9h-2) behaved as expected, including one genuine finding (9a) that led to closing a real test-design gap; all six item-10 edge cases were reviewed, three real gaps found and closed with new tests, one investigated and closed with a defense-in-depth regression test, two judged already/effectively covered with reasoning given.
- Fail: none.
- Blocked: none.

This is pipeline cycle 2 of cap 2 (the final cycle) — proceeding to `/pr` is appropriate on this report.
