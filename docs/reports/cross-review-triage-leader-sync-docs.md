# Cross-review triage report: leader-sync-docs

- Date: 2026-10-30
- Plan: docs/plans/active/2026-10-30-leader-sync-docs.md
- Base branch: origin/main (0351800d, merged into the branch in 0dfc49f0; `RALPH_XREVIEW_BASE=origin/main`; the local `main` was at the same commit)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-30-leader-sync-docs.md (issue #268). A feature org's leader hands sync-docs to the doc-maintainer at the end of step 9, keeps only changes inside the reservation (anything outside, a frozen file, or the plan changed: undo all but the report), commits once, and writes drift outside as follow-ups (`DOC_FOLLOWUP:` and the PR body); the director gathers them into a docs-only feature.
- Reviewed head: e5f90219 (20 files, +2956/-94 against origin/main).
- Self-review report: docs/reports/self-review-2026-10-30-leader-sync-docs.md (run 1: MEDIUM 2, LOW 6, fixed in F1, F2, 8556265e; run 2: LOW 3, R2-1 and R2-3 fixed in F3, R2-2 in the ledger; run 3: LOW 1, fixed in feeb1e80).
- Verify report: docs/reports/verify-2026-10-30-leader-sync-docs.md (first run fail on template purity, fixed in F4 d8e73710; re-run pass at feeb1e80).
- Test report: docs/reports/test-2026-10-30-leader-sync-docs.md (pass at 91e82463; `run-test.sh` rc 0, 12 of 12 mutants failed).
- Sync-docs report: docs/reports/sync-docs-2026-10-30-leader-sync-docs.md (e5f90219).
- Switch: `[standard].cross_review` read with a `ralph` built from this branch: `true`, exit 0.
- Reviewer run: `codex exec review --base origin/main` with `-c sandbox_mode=read-only` and `--ignore-rules` exited 0 and wrote a non-empty `-o` file (986 bytes). The reviewer ran no Go tests (the read-only sandbox blocked them); `/test` ran them. The git state written before and after the run is the same (`cmp`).
- Checked before classifying: the leader template reads the changed paths with `git status --porcelain` (`internal/org/prompts/leader.md:401-404`), and so does the `/org` skill's item 1. With git's default `status.showUntrackedFiles=normal`, a new directory with new files shows as one `?? <dir>/` entry, so the per-file reservation check cannot see the files under it.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Enumerate untracked files before checking reserved paths: the default `git status --porcelain` collapses a new untracked directory into one `?? docs/new/` entry, so a reservation that names `docs/new/guide.md` does not match it (a valid change gets undone) and unreserved siblings under it are hidden (an out-of-scope file gets committed). Use `--untracked-files=all` for the new check and undo, in the leader template and the mirrored `/org` instructions. | Real and worth fixing now: the doc-maintainer may well add a new doc in a new directory, and the check is the guard D2 and AC2/AC8 rely on. The fix is a flag on the commands that list paths (and the undo that deletes untracked files reads the same list), in the template and the skill with their pins. | `internal/org/prompts/leader.md`, `.claude/skills/org/SKILL.md` and its 3 copies, `internal/org/prompts_leader_syncdocs_test.go`, `internal/org/skill_doc_followup_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
