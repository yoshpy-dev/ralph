# Cross-review triage report: org-followups

- Date: 2026-11-01
- Plan: docs/plans/active/2026-10-30-org-followups.md
- Base branch: origin/main (`RALPH_XREVIEW_BASE=origin/main`)
- Driver: claude  Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-30-org-followups.md (digest 6ef9fe7cb76c)
- Self-review report: docs/reports/self-review-2026-10-30-org-followups.md (run 2, LOW 4 → F3; Merge: yes)
- Verify report: docs/reports/verify-2026-10-30-org-followups.md (pass)
- Implementation context summary: the reviewer was `codex -c sandbox_mode=read-only exec review --ignore-rules --base origin/main` (model and effort from `ralph_setting`: gpt-6-astra, xhigh), with the worktree's binary on PATH. rc=0, the `-o` file is not empty, and the git state before and after the run is the same. The reviewer says its focused tests could not run in the read-only sandbox. Self-review and verify did not look at `gh_available` on the path with follow-ups and no feature branch: S3 added the follow-up lookup to that early return of `MergeOrder` without setting the field.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] With open follow-ups and no feature binding or gate record, `MergeOrder` returns early: it runs the follow-up lookup with gh but leaves `GHAvailable` false, so `merge-order --json` says `gh_available: false` while it shows a follow-up ready from gh's answer | Real: the field's doc says it is false when gh was not found or there was no branch to ask about, and here gh was found and asked about the after-branch. A reader of the JSON (#271, a director) would take a ready from gh with gh "unavailable". The fix is small: set the field from the follow-up lookup when gh was found, say in the field's doc that a follow-up's after-branch counts as a branch to ask about, and add a test for the follow-up-only path (gh found, gh missing) | `internal/org/mergeorder.go` (around 192-194, 321-327), `internal/org/mergeorder_followup.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

ACTION_REQUIRED is fixed without asking, by the user's standing decision (fix the cross-review's ACTION_REQUIRED and re-run). The cap is not reached (cycle 1 of 2), so `cycle-count.json` goes to 2 and the full pipeline re-runs from `/self-review`.
