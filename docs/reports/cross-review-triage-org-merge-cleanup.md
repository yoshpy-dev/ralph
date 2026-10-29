# Cross-review triage report: org-merge-cleanup

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-28-org-merge-cleanup.md
- Base branch: main (origin/main at 5eba4013, merged into the branch in 65a7b067)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-28-org-merge-cleanup.md (issue #263). `ralph org merge` removes the feature worktree, the local branch, and the `ralph-worktree.sh` record after it records `pr_merged`, and `ralph org cleanup [--branch]` does the same for a PR merged on GitHub. D2's checks 0–6 decide whether anything is removed; the removal is `ralph-worktree.sh cleanup --id org-<org_id> --force-branch`.
- This file held cycle 1's triage until this run (committed in a7801152: two ACTION_REQUIRED, fixed as S-X1 8e46db0f and S-X2 d3452a9d). Cycle 2 replaces it; cycle 1's findings are in git history and in the plan's Progress checklist.
- Self-review report: docs/reports/self-review-2026-10-28-org-merge-cleanup.md (cycle 2: LOW 6, fixed in C2-F 54fa6351 and 060102bb; cycle 2 run 2: LOW 3, recorded in the tech-debt ledger in 09e1e93b; no CRITICAL, HIGH, or MEDIUM).
- Verify report: docs/reports/verify-2026-10-28-org-merge-cleanup.md (cycle 2: pass; drift C2-D1 to C2-D4 fixed in the sync-docs commit 68636a9c).
- Test report: docs/reports/test-2026-10-28-org-merge-cleanup.md (cycle 2: pass, `run-test.sh` rc 0, 13 of 13 mutants failed).
- Before this run, origin/main (#286) was merged in with `git merge` (65a7b067, no conflicts). After the merge, `go build ./...`, `go vet` of `internal/org`, `internal/cli`, and `internal/config`, `check-skill-sync.sh`, `check-sync.sh`, `git diff --check`, `tests/test-xreview-helpers.sh`, and the narrowed tests of `internal/org` (`Skill|Prompt|Cleanup`), `internal/org/driver`, `internal/config`, and `internal/cli` (`Help|OrgCleanup|OrgMerge`) passed.
- Switch: the PATH `ralph` is v6.0.0 without `config get`, so step 0 read `[standard].cross_review` with a `ralph` built from this branch (`go build ./cmd/ralph`): `true`, exit 0.
- Reviewer run: `codex exec review --base main` with `sandbox_mode=read-only` exited 0 and wrote a non-empty `-o` file (1,644 bytes). The git state before and after the run is the same (`cmp` of the two snapshots). The reviewer could not run Go tests in its sandbox (it says the read-only sandbox blocked the build directory).
- Checked in the code before classifying: the default ledger is the main worktree's `.harness/state/org` (`ResolveOrgStateDir`, `internal/org/statedir.go`), so finding 1 needs `--state-dir` or `RALPH_ORG_STATE_DIR` naming a directory inside the feature worktree; nothing in `CleanupFeatureWorktree` compares the ledger's directory (`filepath.Dir(o.Manifest.Path())`, `internal/org/cleanup.go:181`) with the worktree before the script runs. For finding 2, `runCleanupSteps` (`internal/cli/org_cleanup.go:327-347`) checks `ctx.Err()` only before each cleanup, while `mergeLocked` (`internal/org/merge_run.go:267`) checks it again after its cleanup.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] If the ledger chosen with `--state-dir` or `RALPH_ORG_STATE_DIR` is inside the target feature worktree, the cleanup's script removes it with the worktree (ignored `.harness/state/` files go with the worktree), and `record()` then recreates the directory and writes a manifest with only `worktree_cleaned`, losing the history, the inbox, and other orgs' state while reporting success. Refuse the cleanup when the resolved state directory is inside the worktree. | Real: the default ledger is never there, but an explicit `--state-dir` or `RALPH_ORG_STATE_DIR` can name one (a pane or a shell started in the feature worktree with a relative path), and the loss is the whole ledger of every org, not this feature's worktree. The check is cheap (compare the ledger's directory with the worktree path, symlinks resolved, before anything is removed) and belongs in `CleanupFeatureWorktree`, which both `ralph org merge` and `ralph org cleanup` call. | `internal/org/cleanup.go` (`CleanupFeatureWorktree` / the checks before `remove`) |
| 2 | [P2] When the interrupt arrives while the last (or only) branch is being removed or recorded, `CleanupFeatureWorktree` can still return success (the script gets no ctx, and the record uses `context.WithoutCancel`), and `runCleanupSteps` checks ctx only before each cleanup, so `ralph org cleanup` exits 0 although its help documents exit 1 for an interrupt. Recheck ctx after the result, as `mergeLocked` does, keeping the completed cleanup's line. | Real: a documented exit code is not kept, and a caller (the director's skill reads the exit code) would take an interrupted run as complete. The cleanup that finished is still reported by its `CLEANED:` line; only the exit code and a closing line change. The fix is a ctx check after the loop in `runCleanupSteps`, with a test. | `internal/cli/org_cleanup.go` (`runCleanupSteps`), `internal/cli/org_cleanup_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
