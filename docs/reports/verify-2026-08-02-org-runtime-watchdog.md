# Verify report: org-runtime-watchdog

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-02-org-runtime-watchdog.md`
- Verifier: `verifier` subagent (spec compliance + static analysis, no tests)
- Scope: `git diff e7a32b9...HEAD` on `feat/org-runtime-watchdog` (HEAD `0ac03b6`), 35 files. Includes the two post-self-review fix commits `79d9f40` (address H-1/H-2/M-1..M-6 + L-7) and `0ac03b6` (Go 1.25 `sync.WaitGroup.Go` / switch modernization).

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS | Settings/hooks/mirror gates OK; `scripts/check-sync.sh` PASS (0 DRIFTED); `scripts/check-skill-sync.sh` PASS (14 skills in lock-step); Go verifier: `gofmt: ok`, `go vet` silent (pass), `golangci-lint run` → `0 issues.`, `staticcheck` silent (pass, binary present at `~/go/bin/staticcheck`). Scope fell back to `full` because `scripts/ralph-config.sh` is an "unclassified" changed file, not because of an override. Raw output saved to `docs/evidence/verify-2026-08-02-org-runtime-watchdog.log` (gitignored per `docs/evidence/*.log`, not committed). |
| `git diff e7a32b9...HEAD -- internal/org/protocol/protocol.go` | Reviewed | `ALERT` added to `validTypes` and excluded from `taskIDRequiredTypes`. |
| `cmp` root vs `templates/base/` for `.claude/rules/agent-messaging.md`, `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md` | Identical (exit 0) | Re-confirms self-review's mirror-parity claim independently. |
| `git diff e7a32b9...HEAD -- internal/config/config.go / internal/config/defaults_sync_test.go` | Reviewed | `[org.watchdog]` struct + `Load()` validation + lockstep `check()` calls added for all 4 keys plus `codex_verified`. |
| `grep -n watchdog templates/base/ralph.toml scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` | Reviewed | 3-face mirror present (this repo has no root `ralph.toml`; only `templates/base/ralph.toml` is the declarative source, consistent with repo layout). |

Tests are out of scope for `/verify` (delegated to `/test`); `go test ./...` / `go vet` beyond the static gate above were not executed here.

## Spec compliance (acceptance criteria)

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 `[org.watchdog]` 3-face lockstep | **Met** | `internal/config/config.go` (`OrgWatchdogConfig`, `Default()`, `Load()` validation for `interval_seconds≥1`/`stall_minutes≥1`/`watcher_model` non-empty when enabled), `templates/base/ralph.toml:122-126`, `scripts/ralph-config.sh:101-110` + template mirror, `internal/config/defaults_sync_test.go` new `check()` calls for all 4 keys. |
| AC-2 `ALERT` protocol type | **Met** | `internal/org/protocol/protocol.go` (`TypeAlert`, in `validTypes`, absent from `taskIDRequiredTypes`); `.claude/rules/agent-messaging.md` TYPE table row + `templates/base/` and `.agents/` mirrors byte-identical (`cmp` confirmed). |
| AC-3 seat budget cutoff + Reason + ALERT | **Met** | `evaluateSeatBudget` (`watch.go`) calls `Stop{..., Reason: details}` with condition/threshold/observed in `details`; `Cutoff` only set `true` when `Stop` returns no error (H-2 fix, `79d9f40`); ALERT sent once per key via `raiseOrClear`-style `alreadyAlerted` guard. Covered by `TestWatch_SeatBudgetCutoff_AtBoundary_NotBeforeThenCutoffThenDeduped` and `TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds` (behavioral proof delegated to `/test`). |
| AC-3b total budget | **Met** | `evaluateTotalBudget` mirrors the same Stop-error-aware ratchet across all active seats, org-level `conditionKey(orgID, "", condTotalBudget)`. Test: `TestWatch_TotalBudgetCutoff_CutsAllActiveSeats_OneOrgLevelAlert`. |
| AC-3c idempotency (dedupe) | **Met** | `watchConditionRecord.Cutoff`/`Active` persisted in `watch-status-<org_id>.json`; `conditionFirstTS` preserves `FirstTS` across a retried (failed→retried) cutoff. Test: boundary test above plus `TestWatch_Stall_AlertsNoCutoff_RecoversAndRefires` for the non-cutoff dedupe/recover/refire path. |
| AC-4 heartbeat stall / liveness / scope change ALERTs, no cutoff | **Met** | `evaluateSeat` (c)/(d)/(e) branches only call `raiseOrClear`/`sendAlert`, never `Stop`. Stall term additionally fixed (M-6/self-review-labeled "M-8" in code comments) to use `latestSeatEventTS` (any event type) instead of `Roster`'s state-only `SeatStatus.TS`, closing the "inert stall timer" finding. Tests: `TestWatch_Stall_*`, `TestWatch_Liveness_AlertOnAgentGetError`, `TestWatch_ScopeChange_AlertCarriesScopeText_NoCutoff`, `TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents`. |
| AC-5 deadman (3-source OR, escalations.jsonl + stderr + best-effort darwin notify) | **Met, with one caveat** | `checkDeadman` OR's `leadActivityEventCount(rr.Events) > pending.ManifestLen`, `leadProbeSnapshot`, `historySnapshot`. `realEscalate` uses `osascript` via `exec.CommandContext` with `%q`-quoted message (no shell). Tests: `TestWatch_Deadman_NoActivity_EscalatesOnceAfterTimeout`, `TestWatch_Deadman_LeadActivity_PreventsEscalation`, `TestWatch_Deadman_LeadIsAnomalySubject_EscalatesImmediately`. **Caveat (M-4, partial fix):** `leadActivityEventCount` (the `79d9f40` fix) only excludes the watchdog's *own* cutoff-reason events (`reason=watchdog_` substring match) from the manifest-growth activity source; it does not scope the count to `orgID` or to lead-attributable seats. The self-review's own recommended fix was "filter to events attributable to lead (`ev.OrgID == status.OrgID && SeatID == lead \|\| \"\"`), or drop source #1". The narrower fix closes the exact repro the self-review demonstrated (regression-tested by `TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert`) but a genuinely unrelated seat's non-watchdog manifest event (e.g. a third seat's own `spawned`/`sent` event, in this org or — since `w.org.Manifest.Read()` is not org-scoped — a different org sharing the same manifest file) still counts as "lead activity" and can prematurely clear a pending alert. Not a regression from pre-PR behavior (the shared-manifest-across-orgs read path predates this PR), but the self-review's root-cause fix is not fully closed. |
| AC-6 on-demand watcher, non-blocking, hang→`watcher_error`, budget cutoff unaffected by hang | **Met** | `newWatchdogHooks` dispatches `RunWatcher` in a goroutine (single-flight via `atomic.Int32`); `watcherInvokeTimeout` (fixed 60s, interval-independent) bounds it; malformed/empty/timeout paths all record `watcherErrorReason` via `recordWatcherReceipt`. `--once`'s WaitGroup fix (M-5, code-commented "M-7") makes the CLI actually wait for the in-flight goroutine instead of exiting mid-flight — `sync.WaitGroup.Go` (Go 1.25, matches `go.mod`'s `go 1.25.8`). Tests: `TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt`, `TestRunWatcher_TimeoutIndependentOfSmallInterval`, `TestNewWatchdogHooks_Dispatch_NeverBlocksCaller`, `TestNewWatchdogHooks_SingleFlight_SecondTriggerSkippedWhileBusy`. Honored tri-state fixed to compare `envelope.Model == cfg.WatcherModel` (M-3/"M-5" in code comments) — `HonoredFalse` is now reachable, tested by `TestRunWatcher_ReportedModel_MismatchHonoredFalse`. |
| AC-7 state-dir resolution (flag > env > git toplevel > cwd, resolver returns source, tech-debt row closed) | **Met** | `internal/org/statedir.go` `ResolveOrgStateDir(explicit, explicitSet)`; `explicitSet` sourced from `cmd.Flags().Changed("state-dir")` (flag default now `""`) at all 9 verb call sites via `newOrgRuntime`/`newOrgRuntimeAt`. 5-case unit coverage: `TestResolveOrgStateDir_ExplicitFlagWins`, `_EnvWinsOverGitAndCwd`, `_GitSubdirResolvesToToplevel`, `_GitRoot`, `_NonGitCwdFallsBackToCwd`. Tech-debt row struck through with a `RESOLVED 2026-08-02` HTML-comment annotation naming the commit and mechanism (`docs/tech-debt/README.md`). **Gap:** the plan's own AC-10 smoke checklist item 4 ("state-dir 解決により worktree/リポジトリルートの分裂が起きない" — a live confirmation) has no corresponding line in `docs/evidence/org-watchdog-smoke-2026-08-02.txt` (grepped for `state-dir`/`state_dir`/`toplevel`/`cwd`, no hits); only the 5 unit tests exercise this AC. Not blocking (AC-7's own text only requires the 5-case test + tech-debt closure, which are both done), but AC-10's smoke checklist promises a 4th real-machine confirmation that the committed evidence file does not show. |
| AC-8 codex permission fail-closed real probe | **Met** | `docs/evidence/...txt` records `-s/--sandbox <SANDBOX_MODE>` and `-a/--ask-for-approval <APPROVAL_POLICY>` confirmed present; `codex_verified` stays `false` by default (`config.go` `Default()`); `internal/org/permissions.go` `permissionArgsForDriver` only unlocks `codexAutonomousArgs`/`codexEditsArgs` when `cfg.Permissions.CodexVerified`; unknown-mode LOW fix (self-review L-7) adds a `default:` arm distinguishing "unknown mode" from "not yet live-verified" (`79d9f40`), tested by the new `TestPermissionArgsForDriver_Codex_UnknownMode_DistinctError` subtest. |
| AC-9 evidence redaction convention + LOW batch closure | **Met** | `docs/quality/definition-of-done.md` gained the `$HOME → ~` redaction line; `docs/evidence/org-watchdog-smoke-2026-08-02.txt` header states "redacted: $HOME -> ~" and contains no literal home path. `checkCapacityAndStart` parameterized (`spawn.go`), `newOrgRuntime` reverted to `(*org.Org, error)` + `newOrgRuntimeAt` split out, `/org` skill `description:` translated to English and mirrored (4-way, `check-skill-sync.sh` green). Permission-enum placement kept as-is per plan design decision, tech-debt row closed "final, not moved". |
| AC-10 live smoke | **Partially met** | Evidence covers: exact-boundary budget cutoff with full Reason audit (`reason=watchdog_budget_cutoff seat_wall_clock=1m observed=1m0s`), ALERT delivery watchdog→lead (2 ALERTs, attempt 2 after fix `7101351`), on-demand watcher verdict (`verdict=normal`, haiku, 60s fixed timeout), deadman escalation (`escalations.jsonl` entries + implied stderr banner), codex flag probe (AC-8), watch-status heartbeat (`cycles: 8`). **Not covered:** the state-dir cross-cwd real-machine confirmation named in the plan's Scope bullet 5 / AC-10 text (see AC-7 gap above) — no evidence line demonstrates two different cwds inside the repo converging on the same manifest in a live run. |
| AC-11 `go test ./...` / `run-verify.sh` green | **Out of scope for `/verify`** | Static half (`run-static-verify.sh`) is green (see Deterministic checks). Full `go test ./...` execution is `/test`'s responsibility per the pipeline contract — not run here. |

## Self-review fix-commit cross-check (H-1/H-2/M-1..M-6, LOW batch)

Walked each self-review finding against `git show 79d9f40`/`0ac03b6`:

| Finding | Status | Where |
| --- | --- | --- |
| H-1 (shared `watch-status.json` across orgs) | Fixed | `WatchStatusFileName(orgID)` → `watch-status-<org_id>.json`; regression test `TestRunWatch_MultipleOrgs_SeparateStatusFiles`. |
| H-2 (cutoff ratchet set on failed `Stop`) | Fixed | Both `evaluateTotalBudget` and `evaluateSeatBudget` only set `Cutoff: true` when the `Stop` call's `.Err` is nil; failure logged to `w.stderr`; `conditionFirstTS` preserves original `FirstTS` across a retry. Regression: `TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds`. |
| M-1 (stale `RunWatcher` doc mentioning nonexistent `watcherTimeout`) | Fixed | Comment now names `watcherInvokeTimeout, a fixed 60s bound`. |
| M-2 (orphaned `Spawn` doc comment) | Fixed | `checkCapacityAndStart` (with its own doc) moved below `Spawn`; `Spawn`'s saga-ordering doc reattaches to `Spawn` itself. |
| M-3 (Honored semantics ignore mismatch) | Fixed | `switch envelope.Model { case "": Unknown; case cfg.WatcherModel: True; default: False }`; `HonoredFalse` now reachable and tested. |
| M-4 (deadman activity source over-counts) | **Partially fixed** — see AC-5 caveat above. |
| M-5 (`--once` never joins the watcher goroutine) | Fixed | `newWatchdogHooks` returns `*sync.WaitGroup`; `RunE` calls `watcherWG.Wait()` after `RunWatch` returns. Tests updated to assert on `wg.Wait()` instead of polling `receiptCount`. |
| M-6 (inert stall time term) | Fixed | `latestSeatEventTS` (any manifest event type) replaces `Roster`'s state-only `SeatStatus.TS` in the stall condition's time term. |
| LOW batch (L-1..L-7) | 3 of 7 fixed in this cycle (L-1 double-resolve, L-3 banner cadence, L-4 `%w` nil typo, L-7 codex unknown-mode) — the remainder (L-2 discarded `source`, L-5 oversized scope-change ALERT truncation, L-6 unreachable `Escalated` guard/unbounded map) were not required by the self-review's own recommendation ("Follow-ups (batchable, no urgency)") and remain open, consistent with that deferral. |

Note on numbering: the self-review report's own table numbers these H-1/H-2/M-1..M-6/L-1..L-7 in row order; the fix-commit's inline comments instead say "self-review M-7"/"M-8" for the WaitGroup and stall-time fixes respectively. This is an internal labeling drift inside the fix commit's own comments (not a doc-vs-code drift against the self-review report) — cosmetic, not a compliance gap, but worth aligning the next time either file is touched.

## Documentation drift

| Area | Status | Notes |
| --- | --- | --- |
| `/org` skill (`.claude/skills/org/SKILL.md` + 3 mirrors) | **In sync** | `watch` verb row added, cwd convention relaxed to "same repository, not same cwd" text matching `ResolveOrgStateDir`'s actual precedence; `cmp` confirms all 4 copies byte-identical. |
| `.claude/rules/agent-messaging.md` + mirror | **In sync** | `ALERT` row added, mirrored, `cmp`-confirmed. |
| `docs/quality/definition-of-done.md` | **In sync** | Redaction convention line added per AC-9. |
| `docs/tech-debt/README.md` | **In sync** | Both PR④-targeted rows (state-dir cwd-split, `/org` skill description language) plus the batchable-3 row are struck through with `RESOLVED 2026-08-02 in feat/org-runtime-watchdog` HTML-comment annotations naming commit/mechanism/tests, matching this repo's established resolution-annotation convention. |
| `AGENTS.md` | **In sync** | `internal/org/` repo-map line minimally extended with "two-layer watchdog (pulse watch + on-demand watcher)" — matches plan's "AGENTS.md 最小追従" scope. |
| `docs/specs/2026-08-01-org-runtime.md` FR-8 | **Pending, as planned — plus one additional unflagged drift** | Plan explicitly defers the "常駐座席" (resident LLM seat) → "オンデマンド `claude -p`" spec revision to `/sync-docs` (design decision + Scope bullet both say this). That part is correctly flagged as pending, not a gap in this PR. **Additional finding not called out in the plan's deviation notes:** FR-8's spec text also states the pulse layer's default interval is "既定 60 秒", but the shipped default (`config.go` `Default()`, `templates/base/ralph.toml`, plan's own Scope bullet) is 30 seconds. This is a second FR-8/implementation mismatch that `/sync-docs` should reconcile alongside the resident/on-demand wording — flagging it now so it isn't missed since the plan's own deviation list only names the seat-form change. |

## Coverage gaps

- `go test ./...` and `-race` execution: not run here (belongs to `/test`).
- AC-5's `leadActivityEventCount` narrow scoping (M-4 partial fix) — the cross-org / cross-seat manifest-growth false-positive is not eliminated, only the exact self-review repro. No regression test exists for the broader case; recommend either a follow-up tech-debt row or a fix-and-revalidate pass before merge, given the self-review explicitly called this a HIGH-adjacent MEDIUM ("no urgency" was never claimed for M-4, unlike the LOW batch).
- AC-10 state-dir cross-cwd live confirmation: no line in the committed evidence transcript; only unit-tested.
- Fix-commit finding-number labels ("M-7"/"M-8" in code comments) do not match the self-review report's own M-5/M-6 row order — cosmetic drift, worth a one-line comment fix next touch.

## Verdict

**PASS with two follow-up items**, no fail-blocking findings:

- Verified: AC-1, AC-2, AC-3, AC-3b, AC-3c, AC-4, AC-6, AC-7 (implementation + unit tests), AC-8, AC-9; all static analysis (`run-static-verify.sh` full pass, mirrors green, config lockstep green); H-1/H-2/M-1/M-2/M-3/M-5/M-6 self-review fixes confirmed in code; doc drift closed for `/org` skill, `agent-messaging.md`, definition-of-done, tech-debt, `AGENTS.md`.
- Partially verified: AC-5 (deadman mechanism correct, but M-4's fix is narrower than the self-review's own recommendation — see caveat), AC-10 (3 of 4 named smoke confirmations present in evidence; state-dir cross-cwd confirmation is unit-tested only, not live-evidenced).
- Not verified (out of scope for `/verify`): AC-11 (`go test ./...`), behavioral correctness of any of the above beyond what static reading + the self-review's own test additions show — hand off to `/test` with the test names listed per AC above as the expected covering set.
- New drift flagged (not previously called out in plan deviations): FR-8 spec's "既定 60 秒" vs. shipped 30s default — add to the same `/sync-docs` pass already planned for the resident/on-demand FR-8 wording fix.

Recommend: proceed to `/test`; carry the AC-5/M-4 partial-fix and AC-10 state-dir-smoke gap forward as named follow-ups (tech-debt row or fix-and-revalidate, per `/test`'s and the human operator's judgment) rather than blocking on them here, since neither is a regression from pre-PR behavior and both are narrowly scoped.

## Cycle 2 (fix-and-revalidate)

- Date: 2026-08-03
- Verifier: `verifier` subagent (spec compliance + static analysis, no tests)
- Scope: delta since cycle-1 baseline `af7c5f6` (this report's own initial commit) through HEAD `ccf506e` on `feat/org-runtime-watchdog` (merge-base `e7a32b9`): `6a09e64` (state-dir cross-cwd smoke addendum), `255e79e`/`52ecacb` (test report + sync-docs, including the FR-8 spec revision and two new tech-debt rows), `674581e` (cross-review triage, 2 ACTION_REQUIRED), `19c7630` (fix: cross-review AR-1 org-scope + AR-2 inactive-org guard), `6836efe` (cycle-2 self-review report), `ccf506e` (fix: M2-1 seat-attribution completion + M2-2 SIGINT context threading + M2-3 comment rewrite).

### Deterministic checks run (re-confirmed at HEAD)

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS (exit 0) | Settings/hooks/mirror gates OK; Codex hook/PR-provenance guards OK; `scripts/check-sync.sh` → 179 IDENTICAL / 0 DRIFTED / 3 KNOWN_DIFF / 10 TEMPLATE_ONLY, "PASS: all files in sync."; `scripts/check-pipeline-sync.sh` → all 8 canonical-order references OK; `scripts/check-skill-sync.sh` → "14 skill(s) in lock-step"; Go verifier (full scope, `golang` pack): `gofmt: ok`, `go vet ./...` silent (pass), `golangci-lint run ./...` → `0 issues.`, `staticcheck ./...` silent (pass; binary present at `~/go/bin/staticcheck`). Re-run independently a second time for this cycle-2 pass with the same result. Evidence: `docs/evidence/verify-2026-08-02-170429.log` (gitignored per `docs/evidence/*.log`, not committed). |
| `go build ./...` | Clean (exit 0) | Direct re-confirmation beyond the pack script. |
| `go vet ./internal/org/... ./internal/cli/...` | Silent (exit 0) | Scoped re-confirmation on the two packages the delta touches. |
| `git status --porcelain` | Empty | Working tree clean at HEAD `ccf506e`; no uncommitted drift. |

### Cross-review AR-1/AR-2 and cycle-2 self-review M2-1/M2-2/M2-3

| Item | Status | Evidence |
| --- | --- | --- |
| AR-1 (deadman "lead activity" not org-scoped) | Fixed in `19c7630`, completed in `ccf506e` | `19c7630` added the `orgID` parameter to `leadActivityEventCount` and filtered `ev.OrgID != orgID`; `ccf506e` added the seat-attribution half `ev.SeatID == LeadIdentity \|\| (Event == Stopped\|\|Disbanded && !watchdog-cutoff)`, closing the self-review M-4/M2-1 gap the cycle-1 verify report flagged as a caveat. Regression tests: `TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert` (`19c7630`), `TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert`, `TestWatch_Deadman_LeadSpawnedEvent_ClearsPendingAlert`, `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` (`ccf506e`), all present in `internal/org/watch_test.go`. |
| AR-2 (inactive-org total-budget guard) | Fixed in `19c7630` | `evaluateTotalBudget` returns early when `len(activeSeats) == 0`, before the `allStopped`/ratchet logic can vacuously set `Cutoff: true` off an empty seat list. Regression test: `TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat` (`internal/org/watch_test.go:432`), whose third phase specifically asserts enforcement resumes once a new seat spawns into the previously all-stopped org — the exact "silently disabled forever" failure mode the guard prevents. |
| M2-1 (tech-debt row overclaimed "fully closes this row" while the seat half was still open) | Fixed | `docs/tech-debt/README.md`'s `leadActivityEventCount` row now reads "Both halves of the original M-4 recommendation are now implemented; this row is fully closed", citing both `19c7630` (org-scope) and the `ccf506e` fix commit (seat-scope) by name — this claim is now true against the shipped code (see AR-1 row above), closing the accuracy gap the cycle-2 self-review caught. |
| M2-2 (`--once`'s `WaitGroup.Wait()` fix regressed long-running SIGINT responsiveness by threading `context.Background()` into the tracked goroutine instead of `cmd.Context()`) | Fixed | `internal/cli/org.go`'s `newWatchdogHooks` now takes `ctx context.Context` as its first parameter (was implicit `context.Background()` internally); `newOrgWatchCmd`'s `RunE` passes `cmd.Context()` at the call site; both the `RunWatcher` and `SendWatchdogAlert` calls inside the tracked goroutine use `ctx`, not `context.Background()`. Regression test: `TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly` (`internal/cli/org_test.go`). The stale doc comment claiming "Bounded by RunWatcher's own watcherInvokeTimeout, so this can never hang the command" was also corrected to describe the SIGINT-cancellation path instead. |
| M2-3 (AR-2 guard's comment asserted a mechanism the code doesn't have — "spurious ALERT every cycle" instead of "one spurious ALERT, then permanently disabled") | Fixed | The guard's comment in `evaluateTotalBudget` (`internal/org/watch.go`) was rewritten to describe the actual ratchet consequence — the vacuous `Cutoff: true` write permanently disabling enforcement for any seat spawned into the org afterward — and points at `TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat`'s third phase as the regression this prevents, dropping the incorrect "every cycle" claim. |

Cycle-1's other follow-up item (AC-10/AC-7 state-dir cross-cwd live-smoke line) was already closed within the cycle-1 baseline window by `6a09e64` (2 minutes after the cycle-1 verify report commit `af7c5f6`), which added a `--- state-dir cross-cwd confirmation (AC-7/AC-10) ---` block to `docs/evidence/org-watchdog-smoke-2026-08-02.txt` (`grep -n -i cwd` confirms the block exists). Re-confirmed present at HEAD; no further action needed for this item in cycle 2.

### Spec FR-8 vs. shipped behavior

**In sync.** `docs/specs/2026-08-01-org-runtime.md` FR-8 (line 36) now reads "パルス層(決定論タイマー、既定 30 秒: …)" and "ウォッチャー層(パルス層トリガーのオンデマンド LLM 判定: … `claude -p` を非同期 single-flight で 1 回起動 …)", matching the shipped `internal/config/config.go` `Default()` (`IntervalSeconds: 30`) and the on-demand `newWatchdogHooks`/`RunWatcher` design. Both mismatches the cycle-1 verify report flagged (resident-seat wording, 60s-vs-30s default) were corrected in `52ecacb` and are confirmed resolved by direct read of the current spec file — no residual "常駐座席" or "60 秒" language remains in the FR-8 paragraph.

### check-sync / check-skill-sync

Both green at HEAD, re-confirmed above under Deterministic checks: `check-sync.sh` reports 0 DRIFTED (179 IDENTICAL, 3 KNOWN_DIFF, 10 TEMPLATE_ONLY — all pre-existing, unrelated to this branch), `check-skill-sync.sh` reports 14 skills in lock-step (this branch touches no `.claude/skills/` or `.agents/skills/` content in the cycle-2 delta).

### Documentation drift (cycle 2)

No new drift found. The two tech-debt rows `52ecacb` added for the cycle-1 verify/test follow-ups (deadman activity scoping, 3 remaining deferred LOWs) are current: the deadman-scoping row is now fully resolved per M2-1 above (text accurately says so); the 3-remaining-LOW row is unchanged and still accurately describes open, non-blocking items (`source`-return discard, oversized scope-change ALERT truncation, unreachable `Escalated` guard) that neither cross-review nor cycle-2 self-review flagged as requiring a fix this cycle.

### Coverage gaps (cycle 2)

- `go test ./...` / `-race` execution: not run here (belongs to `/test`, which has not yet produced a cycle-2 report as of this verify pass — the committed `test-2026-08-02-org-runtime-watchdog.md` is the cycle-1 report and predates `19c7630`/`ccf506e`). Recommend `/test` re-run to add coverage evidence for the 7 new/changed test names cited above (`TestWatch_TotalBudgetCutoff_NoActiveSeats_...`, `TestWatch_Deadman_CrossOrgActivity_...`, `TestWatch_Deadman_UnrelatedSeatEvent_...`, `TestWatch_Deadman_LeadSpawnedEvent_...`, `TestWatch_Deadman_ManualStopOfOtherSeat_...`, `TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly`) before `/pr`.
- The 3 remaining deferred LOW findings (tech-debt row, unchanged) remain unfixed by design — batchable, non-blocking, next-touch trigger already recorded.

### Verdict (cycle 2)

**PASS**, no fail-blocking findings, no open follow-up items from cycle 1 remain:

- Verified: full-scope `run-static-verify.sh` (gofmt/go vet/golangci-lint/staticcheck all green), `check-sync.sh` (0 DRIFTED), `check-skill-sync.sh` (14 in lock-step), `go build`/`go vet` direct re-confirmation, clean working tree.
- Verified: cross-review AR-1 (org-scope) and AR-2 (inactive-org guard) both fixed with regression tests; cycle-2 self-review M2-1 (seat-attribution completion), M2-2 (SIGINT context threading), M2-3 (comment accuracy) all fixed with regression tests or direct code confirmation; tech-debt row's "fully closed" claim is now accurate.
- Verified: spec FR-8 matches shipped behavior (on-demand watcher, 30s default interval), both cycle-1-flagged mismatches corrected.
- Not verified (out of scope for `/verify`): behavioral correctness of the 6 new/changed test names above — hand off to `/test` for a cycle-2 test pass before `/pr`.
- Carried forward, non-blocking: the 3 deferred LOW findings tech-debt row (unchanged, next-touch trigger already recorded).

Recommend: proceed to `/test` (cycle-2 re-run) to close the coverage-evidence gap noted above, then continue the pipeline toward `/pr`.

## Cycle 3 (fix-and-revalidate, cap raised to 3)

- Date: 2026-08-03
- Verifier: `verifier` subagent (spec compliance + static analysis, no tests)
- Scope: delta since cycle-2 verify baseline `b7110c6`, through HEAD `8bbd55e` on `feat/org-runtime-watchdog` (merge-base `e7a32b9`): `cca5201`/`d6ddf61` (cycle-2 test + sync-docs reports), `764b01f` (fix: cross-review cycle-2 #3/#4 — lead-only history filter + same-cycle cut-seat skip), `d6e557a` (refactor: `strings.Cut`/`SplitSeq` lint cleanup), `ae6d852` (docs: cycle-3 self-review section, findings H3-1/M3-1/M3-2/L3-1..3), `8bbd55e` (fix: corrected lead-activity model — `sent` events are lead-authored by construction; widened lead-driven lifecycle set for M3-1; count-based history comparison for M3-2; tech-debt row rewrite).
- Cap note: pipeline cycle cap was raised to 3 for this fix per `docs/reports/cross-review-triage-org-runtime-watchdog.md` ("Decision (cap-reached Option 1)").

### Deterministic checks run (re-confirmed at HEAD)

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS (exit 0) | Settings/hooks/mirror gates OK; Codex hook/PR-provenance guards OK; `scripts/check-sync.sh` → 179 IDENTICAL / 0 DRIFTED / 3 KNOWN_DIFF / 10 TEMPLATE_ONLY (unchanged from cycle 2, all pre-existing), "PASS: all files in sync."; `scripts/check-pipeline-sync.sh` → all 8 canonical-order references OK; `scripts/check-skill-sync.sh` → "14 skill(s) in lock-step" (this delta touches no `.claude/skills/`/`.agents/skills/` content). Go verifier (full scope): `gofmt: ok`, `golangci-lint run` → `0 issues.`; `go vet`/`staticcheck` silent (pass; `staticcheck` binary present at `~/go/bin/staticcheck`). Evidence: `docs/evidence/verify-2026-08-02-175415.log` (gitignored per `docs/evidence/*.log`, not committed). |
| `go build ./...` | Clean (exit 0) | Direct re-confirmation. |
| `go vet ./...` | Silent (exit 0) | Full-package re-confirmation (also compiles all `_test.go` files, catching any compile-time regression from the `watch_test.go` rewrite without running behavioral tests). |
| `git status --porcelain` | Empty | Working tree clean at HEAD `8bbd55e`. |

### Cross-review cycle-2 #3/#4 (764b01f)

| Item | Status | Evidence |
| --- | --- | --- |
| #3 (P1: deadman's 3rd activity source — agmsg history — cleared pending alerts on ANY team traffic, not just lead's) | Fixed | `filterLeadHistoryLines`/`leadHistoryFromField` (introduced in `764b01f`, present in current `watch.go:930-982`) parse each history line's `from` field and keep only `LeadIdentity`-authored lines before comparison. Confirmed present and unregressed at HEAD (refined further by `8bbd55e`, see M3-2 below). |
| #4 (P2: total-budget cutoff re-evaluated a just-cut seat later in the same cycle against a stale pre-cutoff snapshot, producing a spurious ALERT/deadman record) | Fixed, not regressed | `evaluateTotalBudget` (`watch.go:489-601`) returns a `cutSeats map[string]bool`; the per-seat loop's caller at `watch.go:489-491` skips any seat present in `cutSeats` for the rest of the cycle. Confirmed present at HEAD, untouched by `8bbd55e` (which only edited the deadman/history functions). |

### Self-review cycle-3 findings (H3-1, M3-1, M3-2, L3-1..3) — fix-commit cross-check (`8bbd55e`)

| Finding | Status | Where |
| --- | --- | --- |
| H3-1 (HIGH: `leadActivityEventCount`'s `ev.SeatID == LeadIdentity` branch was inverted — `Send` writes `SeatID: p.To`, the *recipient*, not the author, so seat→lead `sent` traffic still cleared pending alerts while lead→seat sends did not) | Fixed | `leadActivityEventCount` (`watch.go:876-895`) now counts every `EventSent` unconditionally: `if ev.Event == EventSent { n++; continue }`. Verified independently against the codebase (not just the fix commit's own comment): `internal/org/verbs.go:83-136` `Send` records `SeatID: p.To` on both the dry-run and real paths, and `internal/cli/org.go:340` is the only call site for `Org.Send`, wired to the CLI verb `ralph org send --to <seat>`. Grepping `internal/org/prompts/{reviewer,qa}.md` (the seat role-prompt templates) for `"org send"`/`"agmsg"` returns no hits — seats are never instructed to invoke this verb; only `internal/org/prompts/lead.md` documents it ("`ralph org send --to <seat_id>` で個別に typed message を送ってください"). This independently confirms the doc comment's "only lead/the operator drives that verb" claim rather than just trusting it. Regression tests: `TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogCutoffDoesNot` (replaces the inverted `TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert`, which is fully removed — no duplicate/dead test left behind), `TestWatch_Deadman_LeadSpawnsReplacementSeat_ClearsPendingAlert`. |
| M3-1 (MEDIUM: lead spawning a *replacement* seat in response to a stall ALERT produced a `spawned` event that counted as nothing, since only `stopped`/`disbanded` were treated as lead-driven-lifecycle evidence) | Fixed | The non-`EventSent` branch (`watch.go:896-903`) widened from `{EventStopped, EventDisbanded}` to `{EventSpawned, EventSpawnStarted, EventStopped, EventDisbanded, EventRejected}` (all 5 confirmed to exist as named constants in `internal/org/seat.go:32-38`), still gated on `!strings.Contains(ev.Details, "reason=watchdog_")` to exclude the watchdog's own cutoffs. Regression test: `TestWatch_Deadman_LeadSpawnsReplacementSeat_ClearsPendingAlert` spawns a second seat between cycles and asserts the pending alert clears with no escalation. |
| M3-2 (MEDIUM: the history source's exact-string comparison could read pure window-eviction — as other seats chat, older lead lines fall out of the underlying last-20-of-everyone window — as "activity" and wrongly clear a pending alert) | Fixed | `historySnapshot`/`filterLeadHistoryLines` (text) replaced as the deadman comparison basis by `historyLeadLineCount` (`watch.go:918-949`, returns `int`, `-1` sentinel for probe-unavailable) and `leadHistoryLines` (returns `[]string`, shared by both the new count path and the old text path, kept as `filterLeadHistoryLines`'s thin wrapper so `TestFilterLeadHistoryLines` still pins the parsing contract as a string). `checkDeadman` (`watch.go:1030-1041`) now compares `cur > pending.HistoryLeadLines` (strict growth) instead of `cur != "" && cur != pending.History` (any change) — eviction can only decrease a count, never manufacture an increase, closing the false-activity path. `watchPendingAlert.History string` → `HistoryLeadLines int` in the persisted JSON schema (`json:"history_lead_lines"`), a breaking field rename with no migration path for any in-flight `watch-status-<org_id>.json` from a prior binary version — acceptable per the plan's Non-goals/Rollout notes (watch is not yet in production use; state resets cleanly on next `ralph org watch` invocation) but worth a one-line callout since it is a schema change, not purely additive. Regression test: `TestWatch_Deadman_HistoryWindowEviction_DoesNotFalselyClearPendingAlert` (3 lead lines shrink to 1 via eviction; pending alert must remain). |
| L3-1 (LOW: `leadHistoryFromField`'s doc cited `.agents/skills/agmsg/scripts/history.sh` as if repo-relative, but this repo's `.agents/skills/` tree has no `agmsg` entry) | Fixed | Doc comment (`watch.go:952-956`) now reads "the agmsg skill's `scripts/history.sh`, a user-global install under `~/.agents/skills/agmsg/`, not vendored in this repo". Independently confirmed: `ls .agents/skills/` at HEAD lists 14 skills (matches `check-skill-sync.sh`'s count), no `agmsg` directory. |
| L3-2 (LOW: the first-`"] "`-occurrence parse anchor had no documented justification for why it's safe against a body containing a literal `"] "`) | Fixed | Doc comment (`watch.go:958-967`) now explains `history.sh`'s own `replace(replace(body, char(10), '\n'), char(9), '\t')` escaping guarantees each history record is exactly one physical line, so the first `"] "` is always the timestamp's, not a false split inside an escaped body. |
| L3-3 (LOW: `docs/tech-debt/README.md:70`'s "fully closed" claim overclaimed correctness for the second consecutive cycle, since H3-1's predicate was backwards) | Fixed | Tech-debt row rewritten (see below) to describe the corrected model and cite the H3-1/M3-1 fix commit and both new regression tests by name, rather than re-asserting "fully closed" without the underlying fix. |

### Regression check against cycle-1/cycle-2 fixes (independent re-confirmation, not just trusting the cycle-3 self-review's own table)

| Prior finding | Status at HEAD | Evidence |
| --- | --- | --- |
| H-1 (per-org `watch-status-<org_id>.json`) | Not regressed | `WatchStatusFileName(orgID)` unchanged; `8bbd55e` only touched `leadActivityEventCount`/`historySnapshot`→`historyLeadLineCount`/`checkDeadman`. |
| H-2 (cutoff ratchet only on successful `Stop`) | Not regressed | `watch.go:595` `Cutoff: allStopped`, `watch.go:737` `Cutoff: stopErr == nil` — both grep-confirmed present, untouched by the cycle-3 delta. |
| AR-1 (deadman activity `orgID`-scoped) | Not regressed | `watch.go:872` `if ev.OrgID != orgID { continue }` still the first gate inside `leadActivityEventCount`. |
| AR-2 / zero-active-seats guard | Not regressed | `watch.go:553` `if len(activeSeats) == 0 {` early-return still precedes the ratchet loop. |
| M-5 (`--once` joins the watcher goroutine) | Not regressed | `internal/cli/org.go:710` `watcherWG.Wait()` still present. |
| M2-2 (SIGINT-cancellable context threaded into the tracked goroutine) | Not regressed | `internal/cli/org.go:764` `newWatchdogHooks(ctx context.Context, ...)` signature unchanged by this delta. |

No cycle-1 or cycle-2 fix regressed by the cycle-3 delta.

### Spec / documentation drift (cycle 3)

- **Tech-debt row (`docs/tech-debt/README.md:70`)**: rewritten in `8bbd55e` to accurately describe the corrected `sent`-is-lead-by-construction model, the widened lifecycle set, and cites both new regression tests by name. Independently re-read against current `watch.go` — the row's claim now matches the code. The deferred-LOW row directly below it was also expanded from 3 to 4 items, folding in self-review cycle-2's L2-6 (zero-active-seats guard leaves `Active` uncleared) per the cycle-3 self-review's own recommendation — this is documentation catching up with a previously-untracked deferral, not new drift.
- **Plan (`docs/plans/active/2026-08-02-org-runtime-watchdog.md`) AC-5 evidence text (line 73) and "Implementation notes" deviations section**: **not updated in this delta** (`git show --stat` on all 4 delta commits shows no touch to the plan file). AC-5's "既知の残課題" (known residual) prose still describes the pre-cycle-3 state (`leadActivityEventCount` only excludes watchdog's own events, not scoped to seat/org) even though cross-review AR-1 and self-review cycle-2/3 have since layered org-scope, seat-scope (later corrected), and the H3-1 model-correction + M3-1 lifecycle widening on top of it; the "Implementation notes" section's last dated entry is still "Cycle 2". This is doc drift consistent with the known pattern of AC-checkbox/deviation-note text lagging fix commits — non-blocking (the plan's `[x]` checkboxes and evidence pointers to the verify/test/self-review reports remain directionally correct, just stale in the caveat wording), but should be swept in the same `/sync-docs` pass that will also need to add a cycle-3 test report section.
- No other drift found: `.claude/rules/agent-messaging.md`, `/org` skill mirrors, `definition-of-done.md`, spec FR-8 — none reference the deadman activity-counting internals this delta touches, so none are affected.

### Coverage gaps (cycle 3)

- `go test ./...` / `-race` execution: not run here (belongs to `/test`). The committed `docs/reports/test-2026-08-02-org-runtime-watchdog.md` has only Cycle 1 and Cycle 2 sections — no cycle-3 pass exists yet for the 5 new/changed test names introduced by `764b01f` and `8bbd55e` (`TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogCutoffDoesNot`, `TestWatch_Deadman_LeadSpawnsReplacementSeat_ClearsPendingAlert`, `TestWatch_Deadman_HistoryWindowEviction_DoesNotFalselyClearPendingAlert`, plus the `764b01f`-added history-filter and same-cycle-skip tests). `go vet ./...` confirms these compile; behavioral correctness is unverified here by design.
- Plan's AC-5 evidence text and Implementation notes section lag the cycle-3 fixes (see drift note above) — recommend `/sync-docs` sweep alongside the cycle-3 test report addition.
- `watchPendingAlert`'s `History string` → `HistoryLeadLines int` JSON field rename (M3-2) has no migration path for pre-existing `watch-status-<org_id>.json` files from an older binary; acceptable given watch has not shipped to production use yet, but not called out anywhere in the plan's Rollout notes.

### Verdict (cycle 3)

**PASS**, no fail-blocking findings:

- Verified: full-scope `run-static-verify.sh` (gofmt/golangci-lint/go vet/staticcheck all green), `check-sync.sh` (0 DRIFTED, 3 KNOWN_DIFF unchanged, 179 IDENTICAL), `check-skill-sync.sh` (14 in lock-step), `check-pipeline-sync.sh` (all 8 references OK), `go build`/`go vet ./...` direct re-confirmation, clean working tree at HEAD `8bbd55e`.
- Verified: cross-review cycle-2 #3 (lead-only history filter) and #4 (same-cycle cut-seat skip) both fixed and unregressed; self-review cycle-3 H3-1 (corrected lead-activity model, independently cross-checked against `verbs.go`/role-prompt templates, not just the fix commit's own comment), M3-1 (widened lead-driven lifecycle set), M3-2 (count-based history comparison), L3-1/L3-2/L3-3 (doc-comment accuracy + tech-debt row rewrite) all fixed with regression tests or direct code confirmation.
- Verified: no regression to the H-1/H-2/AR-1/AR-2/M-5/M2-2 fix stack from cycles 1–2.
- Not verified (out of scope for `/verify`): behavioral correctness of the cycle-3 test names — hand off to `/test` for a cycle-3 pass.
- New drift flagged (non-blocking): plan's AC-5 evidence prose and Implementation notes section have not been updated to reflect the cycle-3 fixes; the `HistoryLeadLines` JSON field rename is an undocumented (if low-risk) schema change.

Recommend: proceed to `/test` (cycle-3 re-run) to add coverage evidence for the new test names above, then `/sync-docs` to sweep the plan's stale AC-5/Implementation-notes text and add a cycle-3 test-report section, then continue toward `/pr`.
