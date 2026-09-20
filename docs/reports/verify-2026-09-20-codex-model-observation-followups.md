# Verify report: codex-model-observation-followups

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-model-observation-followups.md` (issue #173)
- Verifier: `verifier` subagent (Claude Code), pipeline cycle 1/2
- Scope: spec compliance + static analysis for `git diff main...HEAD` at `3320ca6` (branch `fix/codex-model-observation-followups`). Not in scope: diff quality (`/self-review`, already run — `docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`, 2 MEDIUM + 5 LOW, all fixed in `6564680`) or a test verdict (`/test`).

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | OK (exit 0) | changed-language scope; hook `sh -n`, `jq -e` on both settings.json, Codex hook guards, `check-sync.sh`, `check-pipeline-sync.sh`, `check-skill-sync.sh`, `check-template-purity.sh`, then golang verifier (`gofmt: ok`, `0 issues.`). Evidence: `docs/evidence/verify-2026-09-20-060733.log` |
| `gofmt -l internal/org internal/cli` | OK (no output) | |
| `go vet ./internal/org/... ./internal/cli/...` | OK (no output) | also proves the package compiles |
| `./scripts/check-skill-sync.sh` | OK — `13 skill(s) in lock-step` | run standalone, same result as inside the wrapper |
| `./scripts/check-sync.sh` | OK — `IDENTICAL: 158, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5` | run standalone; this branch touches no synced surface |
| `./scripts/check-template-purity.sh` | OK — no meta-repo-specific references in templates | |

No test suite was run (`go test`/`run-test.sh`), per this task's scope — that is `/test`'s responsibility.

## Spec compliance (acceptance criteria)

| AC | Status | Evidence |
| --- | --- | --- |
| AC-1 | Met | `internal/org/verbs.go:969` (`observeStopModelReceipt`) and `codexSpawnCorrelation` (`internal/org/verbs.go` ~740-800) build the receipt's `CommandedModel`/`Driver`/`Role` only from the latest **non-dry-run** `spawn_started` event, never `seat.Model`. Test `TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected` (`internal/org/verbs_test.go:1632`) reproduces the exact scenario (interrupted spawn → different-model retry rejected → stop) and asserts `CommandedModel == "gpt-5-codex"` (the original spawn), `Honored == HonoredTrue`, with a setup invariant assertion that the roster itself shows the rejected retry's model (`gpt-9-nonexistent`) — the test would fail against the pre-fix code, matching the plan's own claim. |
| AC-2 | Met | "Is this a codex seat" is decided by `corr.Driver` from `codexSpawnCorrelation`, not `seat.Driver` (`internal/org/verbs.go:906-908`). Forward case: `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver` (`verbs_test.go:1693`) — launched spawn codex, rejected retry claude, roster shows `Driver="claude"`, Stop still observes and appends `model_observed=true`. Reverse case: `TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry` (`verbs_test.go:1746`) — launched spawn claude, rejected retry codex, roster shows `Driver="codex"`, Stop appends no `model_observed=` token at all. Both assertions read directly off the appended manifest event's `Details`. |
| AC-3 | Met | `ObserveCodexEffectiveModel` checks `ctx.Err()` before doing any work inside `codexRolloutCandidates` (`codex_session.go:274`). Test `TestObserveCodexEffectiveModel_AlreadyDoneCtx_NotFoundDespiteMatchingRecord` (`codex_session_test.go:143`) passes an already-done `newCountingCtx(0)` against a fixture that would otherwise match, and asserts `Status == CodexObservationNotFound` and `errors.Is(err, context.DeadlineExceeded)`. |
| AC-4 | Met | `TestObserveCodexEffectiveModel_CtxDoneBeforeSecondCandidate_SecondFileNeverOpened` (`codex_session_test.go:178`) calibrates the exact `ctx.Err()` call count by first running `codexRolloutCandidates` and `scanRolloutRecord` directly (not a hardcoded magic number), then confirms the run cuts short exactly before the older (second) candidate would be opened — the older file's content, if read, would make the result ambiguous, so the final `CodexObservationNotFound` (never `CodexObservationAmbiguous`) proves it was never opened. |
| AC-4b | Met | Candidate-collection cut-short: `TestObserveCodexEffectiveModel_CtxDoneDuringCandidateCollection_NotFound` (`codex_session_test.go:258`), backed by `TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx` (`:300`) and `TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory` (`:331`), each of which fails if its specific checkpoint is removed (per the self-review's mutation table). Mid-line cut-short: `TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch` (`:386`). Completed-pass-survives-expired-ctx: `TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned` (`:428`) asserts `Status == CodexObservationFound` even though the budget is calibrated to be exhausted exactly as the last candidate finishes. Together these satisfy all three AC-4b clauses. |
| AC-5 | Met | `TestOrgSpawn_Codex_ObservationBudgetExhaustedMidPass_UnknownNotFoundReason` (`spawn_test.go:2351`) — `CodexModelObserveTimeout = time.Nanosecond` against a genuinely qualifying fixture still yields `codexNotFoundReason` text ("this function's own budget", not the parent). `TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason` (`spawn_test.go:2383`) — `p.TimeoutMS = 1` against a generous observe timeout yields exactly `codexCutShortReason`. The existing not-found/read-error/no-role-prompt-file reason paths are unchanged code (only `ctx`/`obsCtx` threading was added around them — `internal/org/spawn.go:1091-1144`) and are covered by pre-existing #165 regression tests (e.g. `TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout`, `spawn_test.go:2252`) that this diff does not touch. |
| AC-6 | Met | `observeStopModelReceipt` now derives `obsCtx` from `context.WithTimeout(context.Background(), o.codexModelObserveTimeout())` (`verbs.go:960-961`) and treats a cut-short observation (`obs.Status != CodexObservationFound`) as "nothing to append" (`verbs.go:964-966`), same as a genuine not-found. `TestOrgStop_Codex_ObservationCutShort_NothingAppendedStopStillSucceeds` (`verbs_test.go:2387`) sets `o.CodexModelObserveTimeout = time.Nanosecond` against a qualifying fixture, and asserts `result.Err == nil`, `result.ModelReceipt == (Receipt{})`, and the appended event's `Details` contains `model_observed=none`. |
| AC-7 | Partially met (static half only) | Static half: code compiles (`go vet` passed), `gofmt` clean, no schema/CLI/insights/config surface touched (see Non-goals below) — consistent with "existing #165 wiring unchanged". The PR-body `Closes #173` half and the actual test-execution half ("existing tests still pass unchanged") are explicitly out of this skill's scope: test execution belongs to `/test`, and the `Closes #173` text belongs to `/pr` time. Not verified here. |

## Non-goals held (spot-checked)

- `receipts.go` / `manifest.go` / `seat.go`: `git diff main...HEAD -- internal/org/receipts.go internal/org/manifest.go internal/org/seat.go` is empty — no schema change.
- `internal/cli/`: `git diff main...HEAD --stat -- internal/cli` is empty.
- `internal/insights/`: `git diff main...HEAD --stat -- internal/insights` is empty.
- `internal/config/` (ralph.toml keys): `git diff main...HEAD --stat -- internal/config` is empty — no new config key.
- The `stopped` event still records the roster's `Role`/`Driver`/`Model` (`verbs.go:687-696`), now with an explicit comment stating this is deliberate (plan non-goal) and can legally disagree with the `model_observed=` token on the same event when the roster's latest state is a rejected retry.
- The observer reads no wall clock: `grep -n 'time.Now()' internal/org/codex_session.go` returns no match; both time endpoints (`spawnStarted`, `until`) remain caller-supplied.

## Privacy (static reading)

Only a model string and a status (`found`/`not-found`/`ambiguous`) ever leave `ObserveCodexEffectiveModel`. Traced every place a `ctx` error or a genuine read error could reach something user-visible:
- `observeCodexSpawnReceipt` filters with `isCtxDoneErr` before ever assigning to `lastErr` (`spawn.go:1124-1126`), and the three terminal reason strings (`codexNotFoundReason`, `codexReadErrorReason`, `codexCutShortReason`, `spawn.go:1156-1158`) are static category text, not raw error content or paths.
- `observeStopModelReceipt` discards the observer's own error outright (`_ := ObserveCodexEffectiveModel(...)`, `verbs.go:962`) — no path for it to reach the receipt or the manifest event.
- `codexUnknownReceipt`/`codexFoundReceipt` (`spawn.go:1170-1186`) only ever receive the static reason strings or the observed model string, never an `error` value.

No ctx error or read-error text reaches a receipt `Reason`, a manifest event `Details`, or stderr.

## Code-versus-plan specifics (read closely, per request)

- **`codexSpawnCorrelation`/`codexSpawnInfo`/`observeStopModelReceipt`/`Stop`** (`verbs.go`): `codexSpawnCorrelation` only considers `ev.Event == EventSpawnStarted` and skips `ev.DryRun` while scanning for the latest matching index (`verbs.go` ~745-751), so a `rejected` state event can never become `startedIdx`. `Stop` now reads the manifest exactly once (`rr, err := o.Manifest.Read()`, `verbs.go:637`) via the new pure helper `seatFromEvents(rr.Events, ...)`, and passes the same `rr.Events` into `observeStopModelReceipt` — confirmed no second `o.Manifest.Read()` call remains in `verbs.go` (the diff replaces the prior second read with the shared `events` parameter). `model_observed=` is appended exactly when `!p.DryRun && isCodexSpawn` (`verbs.go:686-703`).
- **`ObserveCodexEffectiveModel`/`codexRolloutCandidates`/`scanRolloutRecord`** (`codex_session.go`): the three cooperative checkpoints are present exactly where the plan and self-review describe — per-date-directory (`:274`), per-entry (`:283`), before-open (`:425`), per-line (`:450`). A pass cut short by any of the four returns `CodexObservationNotFound` and never `CodexObservationFound` (`isCtxDoneErr(scanErr)` branch, `:193-199`, and the propagated `codexRolloutCandidates` error, `:180-184`). A pass that completes naturally returns its real result with no trailing ctx check. The doc comment (`:149-167`) states the cooperative-cancellation caveat (can overrun by one read or one syscall already in flight) plainly.
- **`observeCodexSpawnReceipt`** (`spawn.go`): `obsCtx, cancel := context.WithTimeout(ctx, o.codexModelObserveTimeout())` (`:1099`) derives the observation ctx from the spawn's own ctx (itself `context.WithTimeout(context.Background(), p.TimeoutMS)`, `spawn.go:396`). The reason-selection order matches the doc comment exactly: parent-ctx-done wins first (`ctx.Err() != nil` → `codexCutShortReason`, `:1135-1136`), then the last **completed** pass's `lastErr` (→ `codexReadErrorReason`, `:1138-1139`), else `codexNotFoundReason` (`:1141`). The doc comment's self-review-flagged M1 wording issue (bullet numbering, "first" vs. "at all") is fixed at HEAD: the `lastErr` block comment now cites "second bullet" (`:1107`, matching the bullet that actually carries the `isCtxDoneErr` clause), and bullet 3 reads "is done at all" (`:1089`, matching the code's `ctx.Err() != nil` test, not an ordering claim). Ambiguous (`:1119-1120`) and no-prompt-file (`:1092-1093`) paths are byte-for-byte unchanged from #165.

## Documentation drift

No rule/skill/recipe/spec file was touched by this branch (`git diff main...HEAD --stat -- docs .claude/rules .claude/skills` shows only the plan and self-review report under `docs/reports/` and one insight event). The plan's own claim — "behavior promises did not change, so rule/recipe/skill prose stays as-is" — holds for every promise actually stated in prose:

- `.claude/rules/ralph/model-routing.md` ("最大 8 秒", "stop 時にもう一度だけ探す") — unchanged text remains accurate: Spawn's poll is still bounded at up to 8 s (`o.codexModelObserveTimeout()`, unchanged default), and Stop still looks exactly once.
- `docs/recipes/codex-seat-permissions.md` (root and `templates/base/` copy, byte-identical per `check-sync.sh`) — the `stop` looks-once-more sentence (line 122 in both) is unchanged and still accurate.
- `.claude/skills/org/SKILL.md` and its 3 mirrors (`.agents/skills/org/`, `templates/base/.claude/skills/org/`, `templates/base/.agents/skills/org/`) — all four carry identical unchanged text about spawn's 8 s search and stop's one more look.
- `docs/specs/2026-08-01-org-runtime.md` FR-9 — describes the general architecture (manifest event fields, receipt tri-state), not per-call timing; nothing there needed a change.
- `docs/evidence/codex-effective-model-receipt-2026-09-20.md` — no mention of AR-3/AR-4/#173/rejected/deadline/cancel; it documents the #165 measurement method only and was correctly left untouched.
- `docs/tech-debt/README.md` — no row mentions AR-3, AR-4, or #173 (confirmed via grep); the one existing codex-observer row (line 134, about the parser's dependence on codex's undocumented JSONL shape) is untouched and still accurate, matching the self-review's own confirmation.

**One drift item found, as flagged for confirmation by the task:** nothing operator-facing states that Stop's own single look can itself take up to `codexModelObserveTimeout` (default 8 s) on a pathological (very large or slow) sessions directory — this is new in this branch (Stop's observation was previously unbounded but implicitly fast; it is now explicitly bounded, but that bound is itself a new fact worth stating). This was also raised by `/self-review` (item 6 under "Answers to the requested judgment points") and explicitly deferred to `/sync-docs` rather than filed as a diff-quality finding. The natural insertion point is `.claude/rules/ralph/model-routing.md`'s existing sentence "`ralph org stop` looks once more for a seat that has no observed receipt since its spawn and, when it finds the record, appends one" — a clause such as "(bounded by the same up-to-8s budget as spawn's own poll)" would close the gap, mirrored into the four `org/SKILL.md` copies and the two `codex-seat-permissions.md` copies that carry the same claim. Not edited here (out of this task's scope; `/sync-docs` owns it).

**Dated records, not drift** (confirmed not to need updating): `docs/reports/cross-review-triage-codex-effective-model-receipt.md` still lists AR-3 and AR-4 as open findings from PR #174's cycle-2 cross-review — this is a point-in-time triage artifact for that earlier PR, correctly left as a historical record. `docs/reports/walkthrough-2026-09-20-codex-effective-model-receipt.md` makes no AR-3/AR-4 claims either way.

## Coverage gaps / what remains unverified

- Test execution (all of AC-7's "existing tests pass unchanged" claim, plus whether the new AC-1..AC-6 tests actually pass, race-clean, and are deterministic under repetition) — explicitly out of scope for `/verify`; hand to `/test`.
- The `Closes #173` PR-body requirement (AC-7's second half) — verified at `/pr` time, not here.
- The self-review's own "Coverage gaps" section (a `rejected`-only seat stop with no `model_observed=` token test; the M1 read-error-then-cut-short scenario has no dedicated test) — those are `/test`-scope coverage items, restated here only for visibility, not re-adjudicated.
- I did not independently re-run the self-review's mutation-testing battery (deleting each ctx check and confirming which tests fail) — I relied on reading the test bodies and confirming each asserts the specific checkpoint it claims to pin, which is consistent with the self-review's own mutation table. Re-running mutations would require `go test`, out of scope here.

## Verdict

**Pass.** AC-1 through AC-6 are fully met with evidence-backed, non-hardcoded tests (calibrated against real call counts, not magic numbers) at the exact deciding sites named in the plan. AC-7's static half (compiles, `gofmt`/`go vet` clean, non-goals held) is met; its test-execution and PR-body halves are correctly deferred to `/test` and `/pr`. Static analysis is clean across the shell/JSON/skill-sync/template-purity/Go gates. One documentation drift item was found (Stop's new 8 s bound is undocumented) and is already correctly queued for `/sync-docs`, not blocking this verdict.

## Cycle 2 (2026-09-20)

- Scope: spec compliance + static analysis for `git diff main...HEAD` at `b1777d0` (branch `fix/codex-model-observation-followups`). This section supersedes the cycle-1 section above for "current AC status" purposes; the cycle-1 section is left unedited as a point-in-time artifact, per this repo's convention for prior-cycle reports.
- Delta since the cycle-1 report (`638db39`): `4300307` (tester, five new tests, no production change), `7c95d3f` (doc-maintainer, one clause added identically to 8 doc copies), `e5c0aed`/`0f379da` (cycle-1 cross-review: 1 ACTION_REQUIRED, test-only), `5e239b0` (AR-1 fix: generous scan budgets for tests that must find a record), `bb1e79b`/`84b3387` (cycle-2 self-review: 1 HIGH + 3 LOW), `711cd9d`/`a1afc0c` (cycle-2 self-review fixes).
- Production code since `638db39`: `git diff 638db39..HEAD --stat -- internal/org/*.go ':!internal/org/*_test.go'` shows only `internal/org/verbs.go`, +13/-13, comment text only (confirmed by reading the full diff — both doc-comment enumerations gained "a receipts-file read failure" as a listed branch; no logic line changed). No other production file differs from `638db39`.

### Per-AC status at HEAD

| AC | Status | Evidence |
| --- | --- | --- |
| AC-1 | Met (unchanged from cycle 1, line-shifted) | `TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected` (`internal/org/verbs_test.go:1693`). The only change since `638db39` is one added line setting `o.CodexModelObserveTimeout = testCodexObserveGenerousBudget` (the AR-1 fix, since this test's only `Spawn` call is the rejected retry, which fails envelope validation before reaching the observer — only Stop's own observation is at stake); the assertions are byte-identical. `codexSpawnCorrelation` (`verbs.go` ~745-800, unmoved) still builds `CommandedModel`/`Driver`/`Role` only from the latest non-dry-run `spawn_started`. |
| AC-2 | Met (unchanged, line-shifted) | Forward case `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver` (`verbs_test.go:1759`), reverse case `TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry` (`verbs_test.go:1817`) — same one-line generous-budget addition, same assertions. `isCodexSpawn` still comes from `corr.Driver`, not `seat.Driver`. |
| AC-3 | Met (unchanged, line-shifted) | `TestObserveCodexEffectiveModel_AlreadyDoneCtx_NotFoundDespiteMatchingRecord` (`internal/org/codex_session_test.go:143`, same line as cycle 1 — `codex_session_test.go` was not touched by any commit since `638db39`). |
| AC-4 | Met (unchanged, line-shifted) | `TestObserveCodexEffectiveModel_CtxDoneBeforeSecondCandidate_SecondFileNeverOpened` (`codex_session_test.go:211`, shifted +33 lines by the tester's earlier insertions, content unchanged). |
| AC-4b | Met (unchanged, line-shifted; one new test strengthens it further) | Collection cut-short: `TestObserveCodexEffectiveModel_CtxDoneDuringCandidateCollection_NotFound` (`:291`), pinned by `TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx` (`:333`) and `TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory` (`:364`). Before-open: `TestScanRolloutRecord_BeforeOpenCheckIsPinned_AlreadyDoneCtx` (`:401`). Mid-line cut-short: `TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch` (`:419`). Completed-pass-survives-expired-ctx: `TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned` (`:461`). New in this delta (tester, `4300307`): `TestObserveCodexEffectiveModel_NoCtxCheckAfterCandidateLoopCompletes` (`:511`) — asserts there is no stray `ctx.Err()` check between the candidate loop and the final match, by summing `codexRolloutCandidates`'s and `scanRolloutRecord`'s own call counts directly rather than calibrating through `ObserveCodexEffectiveModel` itself (avoids the self-calibration blind spot the self-review's Positive notes call out). |
| AC-5 | Met (unchanged, line-shifted) | `TestObserveCodexSpawnReceipt_CtxDone_DistinctReason` (`spawn_test.go:2369`), `TestOrgSpawn_Codex_ObservationBudgetExhaustedMidPass_UnknownNotFoundReason` (`:2470`), `TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason` (`:2502`). The read-error reason path — previously only covered by reading, per cycle-1's report — now has its own dedicated regression test, `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` (`:2311`), fixed in this delta from a flaky 1ms budget (cross-review AR-1's sibling gap, self-review C2-H1) to an explicit 200ms budget (`:2336`, widened from an initial 50ms by `a1afc0c` per the commit's own reasoning) — confirmed via the classification-rule comment at `:220-241`, which now names this as a third category ("a pass must complete, but the budget must still run out quickly"). Two more regression tests added in this delta cover adjacent read-error/ctx interactions: `TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass` (`:2415`) and `TestOrgSpawn_Codex_ParentCtxCancelledAfterReadError_CutShortStillWins` (`:2541`). |
| AC-6 | Met (unchanged, line-shifted) | `TestOrgStop_Codex_ObservationCutShort_NothingAppendedStopStillSucceeds` (`verbs_test.go:2467`). `observeStopModelReceipt` still derives `obsCtx` from `context.WithTimeout(context.Background(), o.codexModelObserveTimeout())` — unchanged in this delta (only its doc-comment enumeration gained the receipts-read-failure item, `verbs.go:915-921`). New in this delta: `TestOrgStop_LegacySpawnedEvent_NoSpawnStarted_NoModelObservedToken` (`verbs_test.go:1110`) closes a cycle-1 coverage gap (a seat with a legacy `spawned` event and no `spawn_started` at all gets no `model_observed=` token). |
| AC-7 | Partially met (static half only, unchanged verdict) | Compiles (`go vet` clean), `gofmt` clean, non-goals held (below). Per the task assignment, the plan's own AC-1..AC-7 checkboxes are still `[ ]` at HEAD (self-review's C2-L3) — this table is the verdict the team lead asked me to hand back so the checkboxes can be ticked from it; I have not edited the plan. Test-execution and `Closes #173` halves remain out of `/verify` scope. |

### Static analysis (re-run at b1777d0)

| Command | Result |
| --- | --- |
| `./scripts/run-static-verify.sh` | OK (exit 0). Evidence: `docs/evidence/verify-2026-09-20-114027.log` |
| `gofmt -l internal/org internal/cli` | OK (no output) |
| `go vet ./internal/org/... ./internal/cli/...` | OK (no output) |
| `./scripts/check-skill-sync.sh` | OK — `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | OK — `IDENTICAL: 158, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5` (same 5 pre-existing known diffs as cycle 1, including `model-routing.md`) |
| `./scripts/check-template-purity.sh` | OK — no meta-repo-specific references in templates |

### Documentation drift (the new clause, re-checked against code)

The clause added by `7c95d3f` — "(that single look is itself bounded by the same up-to-8s budget as spawn's poll)" / recipe's "(bounded by the same [budget]...)" / skill's "spawn と同じ最大 8 秒の上限でもう一度" — is accurate at HEAD:

- `observeStopModelReceipt` derives `obsCtx, cancel := context.WithTimeout(context.Background(), o.codexModelObserveTimeout())` (`verbs.go:960-961`), the same `o.codexModelObserveTimeout()` helper Spawn's own poll uses (`spawn.go:1099`), both falling back to `defaultCodexModelObserveTimeout = 8 * time.Second` (`spawn.go:33`) when unset — "the same … budget" is literally the same constant/config field, not merely the same default value.
- "bounded" matches the code's cooperative-cancellation contract exactly (checked before/during candidate collection, before opening, per line — `codex_session.go:274,283,425,450`), the same caveat the pre-existing "looks for up to 8 s" sentence about Spawn already approximates.
- Confirmed present and textually consistent (not merely both non-empty) across all 8 copies: `.claude/rules/ralph/model-routing.md:112` and `templates/base/.claude/rules/ralph/model-routing.md:107-108` (the only diff between the two is pre-existing meta-repo-path stripping two sentences earlier, unrelated to this clause — confirmed via `diff`); `docs/recipes/codex-seat-permissions.md:122` and its `templates/base/` copy — byte-identical (`cmp`, exit 0); all 4 `org/SKILL.md` mirrors (`.claude/skills/`, `.agents/skills/`, and both `templates/base/` copies) — byte-identical (`cmp`, exit 0).
- Stop has no outer `--timeout-ms`-equivalent bound of its own (`context.Background()` is the parent, `verbs.go:961`) — the clause correctly does not claim otherwise; it names only the observation's own budget, matching the sentence's careful "that single look is itself bounded", not "Stop is bounded".

**Nothing else went stale.** The test-only and comment-only changes since the cycle-1 report touch no operator-facing behavior, so no other doc passage needed re-checking; I re-read the same 8 files' surrounding paragraphs anyway (unchanged except the one clause) and found no other drift. `docs/tech-debt/README.md`: still no row mentions AR-3/AR-4/#173/C2-H1/C2-L1/C2-L2 (grep empty); the pre-existing codex-observer row (line 134) is untouched.

### Non-goals (re-checked at b1777d0)

- Schemas: `git diff main...HEAD -- internal/org/receipts.go internal/org/manifest.go internal/org/seat.go` is empty.
- `internal/cli/`: `git diff main...HEAD --stat -- internal/cli` shows only `internal/cli/org_test.go` (+46 lines, test-only — the per-test `setCodexModelObserveTimeoutOverride` calls the AR-1 fix added to the CLI's own codex-warning tests). No production `internal/cli` file differs from `main`. The seam it uses (`orgCodexModelObserveTimeoutOverride`, `internal/cli/org.go:158`) predates this branch entirely — introduced in `d2bf6e4`, already on `main` (confirmed via `git merge-base --is-ancestor d2bf6e4 main`).
- `internal/insights/`: diff stat empty.
- `internal/config/` (no new `ralph.toml` key): diff stat empty.
- The observer still reads no wall clock and has no leftover test-injection artifact: `grep -n 'time.Now()' internal/org/codex_session.go` and `grep -n 'time.Sleep' internal/org/codex_session.go` both return no match (the self-review's cycle-2 `time.Sleep(N)`-at-top-of-`scanRolloutRecord` injection technique, used to demonstrate C2-H1/C2-L2, was a throwaway `git archive` copy under the scratchpad, not applied to the real worktree — confirmed clean here).

### Privacy (re-checked at b1777d0)

Unchanged since cycle 1. The only production diff (`verbs.go`, comment-only) adds no new field or code path — `receipt.Reason`/`Details` assignment sites are identical to cycle 1's read. Re-confirmed: `codexNotFoundReason`/`codexReadErrorReason`/`codexCutShortReason` (`spawn.go:1156-1158`) are unchanged static strings; `observeStopModelReceipt` still discards the observer's own error outright (`verbs.go:962`, unmoved in substance).

### What remains unverified

- Same as cycle 1: test execution (including the five new tester-added tests and the two cycle-2 self-review fix commits' own tests) and the `Closes #173` PR-body text are out of `/verify` scope — hand to `/test` and `/pr`.
- I did not independently re-run the self-review's cycle-2 scan-latency injection (`time.Sleep(N)` at the top of `scanRolloutRecord`) that demonstrated C2-H1/C2-L2 — I relied on reading the fixed test bodies (the explicit 200ms budget, the deterministic-pin comment) rather than re-injecting latency myself, consistent with this skill's no-`go test` scope.
- C2-L3 (plan AC checkboxes) is intentionally left for the team lead to apply from this report's per-AC table, per the task assignment — not edited by me.

### Verdict (cycle 2)

**Pass.** All of AC-1 through AC-6 remain fully met at `b1777d0`, with the same tests (mostly line-shifted, one HIGH-severity flake closed with a dedicated new test) proving each AC's exact deciding site. AC-7's static half remains met. The one production change since cycle 1 (`verbs.go`, `711cd9d`) is comment-only and improves accuracy (adds the missing receipts-read-failure branch to both doc-comment enumerations, closing self-review C2-L1). The new documentation clause across all 8 copies is accurate, consistent, and closes the drift item raised in the cycle-1 section. No new drift found. This is the last pipeline cycle under the cap; nothing here blocks proceeding.
