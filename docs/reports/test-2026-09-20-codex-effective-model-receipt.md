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
