# Cross-review triage report: org-merge-cleanup

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-28-org-merge-cleanup.md
- Base branch: main (origin/main at 7522835c, merged into the branch in b164744f)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-28-org-merge-cleanup.md (issue #263). `ralph org merge` removes the feature worktree, the local branch, and the `ralph-worktree.sh` record after it records `pr_merged`, and `ralph org cleanup [--branch]` does the same for a PR merged on GitHub, from merge-order's `--all` reading. D2's checks 0–6 decide whether anything is removed; the removal is `ralph-worktree.sh cleanup --id org-<org_id> --force-branch`.
- Self-review report: docs/reports/self-review-2026-10-28-org-merge-cleanup.md (cycle 1, three runs; no CRITICAL or HIGH; R3-L1 fixed in F4 3dfd434e, the other run-3 LOWs are tech-debt rows).
- Verify report: docs/reports/verify-2026-10-28-org-merge-cleanup.md (pass; LOW-1, the cleanup verb missing from two verb lists, fixed in the sync-docs commit 7e72d11e).
- Test report: docs/reports/test-2026-10-28-org-merge-cleanup.md (pass; `run-test.sh` rc 0; 25 of 26 mutants failed, the survivor is equivalent).
- Sync-docs report: docs/reports/sync-docs-2026-10-28-org-merge-cleanup.md (46ddd636).
- Before this run, origin/main (#280–#285) was merged in with `git merge` (b164744f). The conflicts were lines both sides changed (main made `auto_merge` default to true); the CLI merge tests this branch added now expect `AUTO MERGE: true`. After the merge, `go build ./...`, `go vet`, `check-skill-sync.sh`, `check-sync.sh`, and the narrowed tests of `internal/cli` (OrgCleanup, OrgMerge, OrgRegate, LegacyLedger, rc 0) and `internal/org` (Cleanup, Merge, Regate, StartFeature, SkillSync, Spawn) ran. In `internal/org`, `TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge/while_a_script_runs` failed once (a 10 s wait for a file under load average 10–12) and passed 5 of 5 alone; the branch does not change that test or `regate_merge.go`.
- Switch: the PATH `ralph` is v6.0.0 without `config get`, so step 0 read `[standard].cross_review` with a `ralph` built from this branch (`go build ./cmd/ralph`): `true`, exit 0.
- Reviewer run: `codex exec review --base main` with `sandbox_mode=read-only` exited 0 and wrote a non-empty `-o` file (1,722 bytes). The git state before and after the run is the same (`cmp` of the two snapshots).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] `ralph org cleanup` takes merge-order's PR lookup, which runs `gh pr list` without `-R` and checks the PR's repository against origin's by owner/name only, skipping the check when origin's URL names no repository (a local path). gh could read another host's repository of the same owner/name, or another remote's, and a merged PR there whose head equals the local branch's tip would let the cleanup remove the worktree and the branch while origin's PR is not merged. Pin the lookup to origin with the full identity, as `mergeCheck.findPR` does. | Real: merge-order's weaker check was chosen for a listing, and the cleanup turns that listing into a removal with `--force-branch`. `ralph org merge` already pins its gh calls with `-R <host>/<owner>/<repo>` and refuses an origin with no GitHub repository; the cleanup should read the PR the same way before it removes anything. The commits would still be on the other repository, so the loss is the local worktree and branch, not the commits, but the decision rests on the wrong PR. | `internal/cli/org_cleanup.go:129-134`, `internal/org/mergeorder.go` (`lookupMergePRs`, `mergeOriginRepo.check`) |
| 2 | [P1] The resume path with the worktree's directory gone runs `git worktree remove` on git's registration (`internal/org/cleanup.go:725-726`) after checking only the branch's tip. If the worktree was on a detached HEAD (or another branch) with a commit the PR did not merge and the directory was then deleted, the registration's HEAD and its reflog are the commit's last references, and removing the registration drops them. Read the registered HEAD and branch and refuse unless they match the merged feature. | Real: check 6 (`checkCheckout`) guards exactly this for a worktree whose directory exists, and D2's resume leaves it out when the directory is gone. `git worktree list --porcelain` still prints the registration's `HEAD` and `branch`/`detached`, so the same check can run on it. The case needs a detached HEAD and a deleted directory together, but it loses a commit with no other reference. | `internal/org/cleanup.go` (`readState`, the resume switch, `remove`) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
