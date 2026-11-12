# Cross-review triage report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md (D1–D6, AC1–AC15; the Progress checklist records the cycle 1 fixes S8, the cycle 2 fixes S9 and 3bb1828b)
- Self-review report: docs/reports/self-review-2026-11-12-watch-alert-dedupe.md (cycle 2, second run, Merge: yes)
- Verify report: docs/reports/verify-2026-11-12-watch-alert-dedupe.md (cycle 2, pass)
- Implementation context summary: the reserved org's scope_change compares repository-root-relative paths with the reservation. `git status --porcelain=v1` ignores `status.relativePaths` and `ls-tree` runs with `--full-tree`, but the commit-path listing (`git log` and the fallback `git diff`) honours the user's `diff.relative`, which the cycle 1 self-review checked only for a seat whose cwd is the repository root. A seat may be spawned with a subdirectory as its cwd, so the listing can come back relative to that subdirectory.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] With `diff.relative=true` and a seat whose cwd is a subdirectory, `git log` and the fallback `git diff` return cwd-relative names and leave out changes outside that directory, so an inside path reads as outside (a false alert) and a committed change in a sibling directory disappears (a missed alert). | Real: it breaks the root-relative comparison the reservation check relies on, in both directions, and the user's git config is outside ralph's control. The fix is one flag on each of the two commands (`--no-relative`), with a real-git test that sets `diff.relative=true` and runs from a subdirectory. | internal/org/watch_scope.go, internal/org/watch_scope_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Reviewer note

codex could not run the Go tests in its read-only sandbox; its read-only git probes reproduced the relative output from `internal/org`.

## Earlier cycles

- Cycle 1/2 (18bb7424): 4 findings, all ACTION_REQUIRED (P2): a re-firing folded into a pending alert the leader had already answered; merge-authored edits missed by `git log`; a dangling `origin/HEAD` stopping commit-path detection; an escalation replay counted twice. All four were fixed in S8 (907afde3) and confirmed by the cycle 2 self-review, verify, and test.

## Decision

The cap (2) is reached with an ACTION_REQUIRED finding left. Following the user's standing instruction of 2026-10-11 (fix ACTION_REQUIRED findings without asking; when the cap is reached, raise it and re-run), the cap was raised to 3 for this task (`cap` in `cycle-count.json`), the finding is fixed, and the full pipeline re-runs from /self-review as cycle 3.
