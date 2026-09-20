# Cross-review triage report: codex-effective-model-receipt

- Date: 2026-09-20 (cycle 1)
- Plan: docs/plans/active/2026-09-20-codex-effective-model-receipt.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0
- User decision (2026-09-20, AskUserQuestion): fix both findings (AR-1, AR-2) and re-run the full pipeline as cycle 2/2

## Triage context

- Active plan: docs/plans/active/2026-09-20-codex-effective-model-receipt.md
- Self-review report: docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md (cycle 1: 4 MEDIUM + 7 LOW, all fixed in 26b03ce and 9961b8d)
- Verify report: docs/reports/verify-2026-09-20-codex-effective-model-receipt.md (pass)
- Test report: docs/reports/test-2026-09-20-codex-effective-model-receipt.md (pass)
- Reviewed HEAD: 27b6759. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the org, CLI and insights test suites (pass) and wrote two regression tests of its own that reproduce the findings; it did not start a live seat.
- Implementation context summary: a codex seat's effective model is read from its codex session record and written to the org model receipts. Spawn observes for up to 8 s; Stop observes once more when the seat's current spawn has no observed receipt yet (user decision: "spawn + stop"). Stop finds the seat's current spawn and its role-prompt path from the manifest (`codexSpawnCorrelation`) and decides whether the spawn was already observed from the receipts file (`hasObservedCodexReceiptSince`). Both findings are in that stop-time path, which exists precisely to catch seats whose spawn-time observation came back unknown, so a silent miss there defeats the feature for the seats that need it.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Strip retry metadata from the recovered prompt path. When AgentStart retried, Spawn appends ` agent_start_retries=N` after the prompt path in the `agent_started` step Details, and Stop's parser keeps that suffix as part of the path, so the pointer sentence never matches the session record. | Real: `internal/org/spawn.go:744` rewrites the Details to `agent_started prompt_file=<path> agent_start_retries=N` whenever `retries > 0`, and `codexSpawnCorrelation` (`internal/org/verbs.go:740-742`) takes everything after the prefix as the path. Retries happen on a busy pane, which is the same slow-start situation in which the spawn-time observation is most likely to come back unknown, so the stop-time recovery fails exactly where it is needed, and it fails silently (`model_observed=none`). Worth fixing: small, local, and the self-review and tests missed it because no fixture combined a retry with a stop-time observation. Direction: strip the known ` agent_start_retries=<digits>` suffix (anchored at the end, so a path containing spaces survives), and add a test that goes through Spawn with a retry and then Stop. | `internal/org/verbs.go`, `internal/org/verbs_test.go` (possibly `internal/org/spawn.go` if the suffix is given a shared constant) |
| AR-2 | [P2] Distinguish receipts from same-second respawns. A stop-time observed receipt and the next spawn's `spawn_started` can carry the same second-resolution timestamp; `r.TS < spawnStartedTS` then treats the previous spawn's receipt as having observed the new spawn and suppresses the stop-time observation. | Real: every timestamp is RFC3339 at whole seconds (`o.now()`), and `hasObservedCodexReceiptSince` (`internal/org/verbs.go:760-764`) only skips receipts strictly older than the spawn. `stop` followed by `spawn` of the same seat within one second is what a lead does when it replaces a seat, and scripted flows do it in milliseconds. The window is narrow (the new spawn's own observation must also come back unknown), but the failure is a silent miss of the mismatch this feature exists to report. Worth fixing because the fix is one comparison and errs in the safe direction: count only receipts strictly newer than the spawn start. A receipt of the current spawn cannot land in the same second as its own `spawn_started` on a real seat (the session record appears seconds after the spawn), and if it ever did, the cost is one duplicate observed receipt at stop, not a missed one. The self-review judged the string comparison safe for ordering, which it is; the tie at equal seconds was not considered. Tests with a fixed clock that rely on equal timestamps counting as "already observed" need an advancing clock. | `internal/org/verbs.go`, `internal/org/verbs_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
