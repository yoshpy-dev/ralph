# Cross-review triage report: watch-escalation-inbox

- Date: 2026-10-23
- Plan: docs/plans/active/2026-10-23-watch-escalation-inbox.md
- Base branch: main (local main and origin/main both at e8e8d652 before the run)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-23-watch-escalation-inbox.md (the sixth PR of the trial-fix series: F6, the cross-org watch's escalation that went only to escalations.jsonl and the desktop notification, and F13, the notification result that no record kept, plus the records that turn `fix/split-preamble` into #251; approved digest 0bf5660a7229 after the D5/AC5 deviation in 8d1c53ce, which added the status-file value check to the fallback)
- Self-review report: docs/reports/self-review-2026-10-23-watch-escalation-inbox.md (Merge OK; LOW F1 to F9; fixed in d9b8cd9a except F7(a), the argument regrouping; F1 on the inbox path and F9, the file size, recorded as tech-debt rows)
- Verify report: docs/reports/verify-2026-10-23-watch-escalation-inbox.md (Verdict: pass; LOW 3 and INFO 1 and the help test gap fixed in 000a2352)
- Test report: docs/reports/test-2026-10-23-watch-escalation-inbox.md (Verdict: pass on go1.26 and go1.25.8; 51 of 54 mutants red, the survivors one equivalent mutant and two in gap 2; gaps 1 to 3 filled in 08083792)
- Sync-docs report: docs/reports/sync-docs-2026-10-23-watch-escalation-inbox.md (03beb611; four drifts fixed: the AGENTS.md repo map, `docs/recipes/director-wake.md` on the next-cycle wake of a watch-written ALERT item, the tech-debt file-size row, and the plan's artifact boxes)
- Implementation context summary: `escalateAlert` (internal/org/watch.go) records an escalation of reason `deadman_timeout` or `leader_is_anomaly_subject` as an inbox item of TYPE ALERT, whose body ralph builds with the headers ORG_ID, SEAT, CONDITION, REASON, ALERT_ID, and ALERT_TS; the escalated event carries `alert_id`, and an open, unnotified item with the same alert_id is resent only when the current watch's decision is now. The delivery comes from `inboxDelivery(enabled, acceptsWork)`, shared with `escalateDelivery`; acceptsWork means the watch is not `--once` and no stop is requested. `deadline` records only and prints `WATCHDOG ESCALATION` on stderr, leaving the ack and resolve deadlines to deliver it; now sends it with `sendInboxItemToHuman` (reason inbox_director_disabled or inbox_no_watch). When the inbox is nil, unreadable, or unwritable, or the status-file values fail `checkWatchEscalationValues`, the watch falls back to the banner, the notification, and one escalations.jsonl line whose field `osascript` holds ok, failed: <err>, or skipped. `notifiableType` accepts ALERT from a list kept apart from `escalateTypes`, so `ralph org escalate` still refuses ALERT. The director skill (4 copies), `internal/org/prompts/director.md`, agent-messaging.md, the /org skill, spec FR-6 notes, the evidence table, and tech-debt were updated. The reviewer ran `codex exec review --base main` with `sandbox_mode=read-only` from 07:05 to 07:08 JST, exited 0, and wrote a non-empty `-o` file: no actionable regressions; its whitespace, Go formatting, and template-mirror checks passed, and its targeted Go tests could not run because the read-only sandbox blocked the temporary build directory. The tester ran `run-test.sh` (rc 0) and the org, cli, and config packages on go1.25.8 (rc 0) at 47c213f6, and the commits after it change tests (08083792), docs, reports, the plan's checklist, and insight events only; `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` runs at 95374404 alongside this review.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
