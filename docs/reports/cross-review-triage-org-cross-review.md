# Cross-review triage report: org-cross-review

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-org-cross-review.md
- Base branch: main (4fef6b8c, the same as origin/main)
- Driver: claude  Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-org-cross-review.md (digest 6e44700b98f0)
- Self-review report: docs/reports/self-review-2026-11-01-org-cross-review.md (run 2: LOW 4, all fixed in F4; Merge: yes)
- Verify report: docs/reports/verify-2026-11-01-org-cross-review.md (pass; V1 and V2 fixed in F5)
- Test report: docs/reports/test-2026-11-01-org-cross-review.md (runs 1 and 2: pass)
- Implementation context summary: the reviewer was `codex -c sandbox_mode=read-only exec review --ignore-rules --base main` (gpt-6-astra, xhigh, from `ralph_setting` with the worktree's binary on PATH) at HEAD 2e75c82a. rc=0, the `-o` file is not empty, and the git state before and after the run is the same. The reviewer said its Go tests could not start in the read-only sandbox; the `/test` runs cover them. Two live probes of the reviewer's commands (docs/evidence/xreview-live-probe-2026-11-01.md) already fed findings into S1c and F6; F6 widened `XREVIEW_BASE` to `+` and `@`, and this cycle's second finding goes further.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] At the cycle cap, `since:` compares the tree outside `docs/reports/`, so a gate fix or a rebase that changes only `docs/reports/templates/` gives `since: same` and the leader goes on without asking about a change no other model has seen | Real: `docs/reports/templates/` holds inputs the skills read, not this org's generated records (`.claude/rules/ralph/post-implementation-pipeline.md` makes the same distinction). Cheap to fix: exclude only generated report files directly under `docs/reports/` (for example the pathspec `:(exclude,glob)docs/reports/*.md`, where `*` does not cross `/`), and test that a change under `docs/reports/templates/` gives `since: changed` | `internal/org/prompts/leader.md` (around 810), its test |
| 2 | [P2] The `XREVIEW_BASE` character whitelist refuses default-branch names git accepts (`release=stable`, `release(v2)`, `dévelop`); the reviewer then asks a QUESTION, and after the bounded resends the org blocks | Real: an org on such a repository could never run its cross-review. Validate the branch part with `git check-ref-format --branch` and keep the value one quoted word after `--end-of-options` (the guard against options and revision expressions that F5 and F6 added stays: the `origin/` prefix, no `@{`, no `..`, no leading `-`, no whitespace or control characters) | `internal/org/prompts/reviewer.md` (around 167–171), its test |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

ACTION_REQUIRED is fixed without asking, by the user's standing decision. The cap is not reached (cycle 1 of 2), so `cycle-count.json` goes to 2 and the full pipeline re-runs from `/self-review`.
