# Cross-review triage report: org-send-enter-timing

- Date: 2026-09-19 (cycle 1)
- Plan: docs/plans/active/2026-09-19-org-send-enter-timing.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-09-19-org-send-enter-timing.md
- Self-review report: docs/reports/self-review-2026-09-19-org-send-enter-timing.md (cycle 1 + Revalidation; all findings fixed in 7813c2a, bcc1eb2, f7ea44d)
- Verify report: docs/reports/verify-2026-09-19-org-send-enter-timing.md (pass)
- Test report: docs/reports/test-2026-09-19-org-send-enter-timing.md (pass)
- Reviewed HEAD: 6a3a9d5. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the targeted org and CLI tests (pass) and reproduced both findings with isolated driver stubs; it did not re-test a live TUI.
- Implementation context summary: `ralph org send` types the message, pauses 750 ms, presses Enter exactly once, and confirms the submit through herdr's agent state. The user decided (plan, Design decisions) that ralph never resends Enter, because a codex approval dialog preselects "Yes, proceed" and a blind Enter would confirm it. The same reasoning governs what the CLI tells a human to do after a failure: the self-review revalidation (NEW-1) already split "typed, Enter not pressed" from "Enter pressed, `sent` event not recorded" so that no note tells a human to press Enter on a seat that may have submitted.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Preserve uncertainty when the Enter call times out. If the deadline expires after herdr delivers Enter but before its CLI returns, the runner reports an error although the submit may have happened. `Send` still reports "text typed but not submitted", and the CLI recommends clearing or submitting again. | Real: `driver.ExecRunner.Run` uses `exec.CommandContext(ctx, …)` (`internal/org/driver/driver.go:36-37`), so a deadline that fires while `herdr pane send-keys` is in flight kills the CLI and returns an error whether or not herdr already delivered the key. The `PaneSendKeys` error return (`internal/org/verbs.go:296-300`) sets `EnterPressed=false` and words the error as "not submitted"; the CLI note then says "typed … but not submitted … clear or submit it there". That is the same hazard class as revalidation NEW-1: a note that asserts more than ralph observed can lead a human to press Enter on a seat that did submit and has since reached an approval dialog. Worth fixing in this PR because the note is new output of this PR and its whole purpose is to keep humans from pressing a blind Enter. Direction: report three states, not two (Enter not attempted / Enter attempted with unknown outcome / Enter acknowledged); for the unknown case say so, and tell the operator to read the pane and act only on what they see. | `internal/org/verbs.go`, `internal/cli/org.go`, tests, skill + recipe wording |
| AR-2 | [P2] Preserve the state directory in recovery commands. When `send` runs with an explicit `--state-dir`, the suggested `ralph org read` command omits it and resolves the default manifest. | Real: `--state-dir` is a persistent flag resolved with flag > env > git toplevel > cwd precedence (`internal/cli/org.go:41`, `:101-102`). All three new messages print `ralph org read --org-id <org> --seat <to>` without it (`internal/cli/org.go`, the unconfirmed-submit warning and both post-error notes), so following the printed command fails with "seat not found" or, worse, reads a different seat with the same ids from the default manifest. The live verification for this very plan used `--state-dir` throughout, so the case is not hypothetical. Worth fixing: the messages are new in this PR and the fix is small (append the flag when it was explicitly passed; `--config` does not affect `read`'s seat lookup). | `internal/cli/org.go`, `internal/cli/org_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
