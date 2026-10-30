# Cross-review triage report: leader-sync-docs

- Date: 2026-10-30
- Plan: docs/plans/active/2026-10-30-leader-sync-docs.md
- Base branch: origin/main (0351800d, merged into the branch in 0dfc49f0; `RALPH_XREVIEW_BASE=origin/main`; the local `main` was at the same commit)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached; the cap is raised to 3 for the re-run, and `cycle-count.json` stays at 2)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- This run: cycle 2, reviewed head 822acf62 (after F5, F5b, F6, and the cycle 2 self-review, verify, test, and sync-docs). Cycle 1's triage (da707a0d: one ACTION_REQUIRED, fixed in F5 and F5b) is in git history and in the plan's Progress checklist; this run replaces it.
- Reviewer run (cycle 2): `codex exec review --base origin/main` with `-c sandbox_mode=read-only` and `--ignore-rules` exited 0 and wrote a non-empty `-o` file (1,094 bytes). It ran no Go tests (the read-only sandbox blocked them); `/test` ran them. The git state before and after is the same (`cmp`).
- Checked before classifying (cycle 2): the `/director` skill's "### 予約外の文書の follow-up" puts the follow-up files and, for a set of copies, every copy in the docs feature's `- Reserve:` (`.claude/skills/director/SKILL.md:1128-1133`), and nothing else. In this repository the skills' text is pinned by Go tests that read the SKILL.md files (`internal/org/skill_doc_followup_test.go`, `org_skill_send_sync_test.go`, `director_skill_sync_test.go`), so a docs feature that rewords a pinned sentence fails its gate, and its implementer cannot edit those tests outside the reservation.

## Cycle 1 (kept for the record)

- Active plan: docs/plans/active/2026-10-30-leader-sync-docs.md (issue #268). A feature org's leader hands sync-docs to the doc-maintainer at the end of step 9, keeps only changes inside the reservation (anything outside, a frozen file, or the plan changed: undo all but the report), commits once, and writes drift outside as follow-ups (`DOC_FOLLOWUP:` and the PR body); the director gathers them into a docs-only feature.
- Reviewed head: e5f90219 (20 files, +2956/-94 against origin/main).
- Self-review report: docs/reports/self-review-2026-10-30-leader-sync-docs.md (run 1: MEDIUM 2, LOW 6, fixed in F1, F2, 8556265e; run 2: LOW 3, R2-1 and R2-3 fixed in F3, R2-2 in the ledger; run 3: LOW 1, fixed in feeb1e80).
- Verify report: docs/reports/verify-2026-10-30-leader-sync-docs.md (first run fail on template purity, fixed in F4 d8e73710; re-run pass at feeb1e80).
- Test report: docs/reports/test-2026-10-30-leader-sync-docs.md (pass at 91e82463; `run-test.sh` rc 0, 12 of 12 mutants failed).
- Sync-docs report: docs/reports/sync-docs-2026-10-30-leader-sync-docs.md (e5f90219).
- Switch: `[standard].cross_review` read with a `ralph` built from this branch: `true`, exit 0.
- Reviewer run: `codex exec review --base origin/main` with `-c sandbox_mode=read-only` and `--ignore-rules` exited 0 and wrote a non-empty `-o` file (986 bytes). The reviewer ran no Go tests (the read-only sandbox blocked them); `/test` ran them. The git state written before and after the run is the same (`cmp`).
- Checked before classifying: the leader template reads the changed paths with `git status --porcelain` (`internal/org/prompts/leader.md:401-404`), and so does the `/org` skill's item 1. With git's default `status.showUntrackedFiles=normal`, a new directory with new files shows as one `?? <dir>/` entry, so the per-file reservation check cannot see the files under it.
- Cycle 1's finding (ACTION_REQUIRED, fixed in F5 4d0f70e4 and F5b aec1381f): [P2] Enumerate untracked files before checking reserved paths. The default `git status --porcelain` collapses a new untracked directory into one `?? docs/new/` entry, so a reservation that names `docs/new/guide.md` does not match it and unreserved siblings under it are hidden; use `--untracked-files=all` for the check and the undo, in the leader template and the `/org` skill.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Reserve documentation-contract tests alongside follow-up files (cycle 2): when a follow-up changes wording that tests pin, reserving only the documents and their mirrors leaves the docs org unable to pass its gate; include the affected contract tests and fixtures when drafting the reservation, before the human's approval. | Real in this repository (skills are pinned by tests that read them) and worth fixing now: the fix is one sentence in the director's procedure plus a pin. Before showing the draft, find the tests and fixtures that read or pin the follow-up files (for example by searching the test files for those paths and the sentences to change) and put them in `- Reserve:` too. | `.claude/skills/director/SKILL.md` and its 3 copies, `internal/org/skill_doc_followup_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
