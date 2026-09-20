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
