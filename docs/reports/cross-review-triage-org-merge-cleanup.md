# Cross-review triage report: org-merge-cleanup

- Date: 2026-10-30
- Plan: docs/plans/active/2026-10-28-org-merge-cleanup.md
- Base branch: origin/main (e3a3fbf7, merged into the branch in 7b3efd9f; `RALPH_XREVIEW_BASE=origin/main`, since the local `main` was still at 5eba4013)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/3 (the cap was raised from 2 to 3 for the re-run after cross-review cycle 2; `cycle-count.json` stays at 2)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-28-org-merge-cleanup.md (issue #263). `ralph org merge` removes the feature worktree, the local branch, and the `ralph-worktree.sh` record after it records `pr_merged`, and `ralph org cleanup [--branch]` does the same for a PR merged on GitHub. D2's checks 0–6 decide whether anything is removed.
- History of this file: cycle 1's triage (a7801152, two ACTION_REQUIRED, fixed as S-X1 and S-X2) and cycle 2's (eca287bf, two ACTION_REQUIRED, fixed as S-X3 and S-X4) are in git history and in the plan's Progress checklist. This run replaces it.
- Since cycle 2's triage: S-X3 4d7cbfd8 (`checkLedger`; the interrupt after the loop), S-X4 9dd79b94 (`pathWithin` by file identity; `ledgerMoveStep`; no Manual when the record's worktree holds the ledger), 7f91e08f (move-step wording), sync-docs 502f661c, and the merge of origin/main (#287) in 7b3efd9f, with `samePath` calls moved to `gitroot.SamePath`.
- Self-review report: docs/reports/self-review-2026-10-28-org-merge-cleanup.md (the re-run: MEDIUM 1 and LOW 4, fixed in S-X4; its run 2: LOW 3, C3R2-L1 and C3R2-L3 fixed in 7f91e08f, C3R2-L2 in the ledger).
- Verify report: docs/reports/verify-2026-10-28-org-merge-cleanup.md (the re-run: pass, HEAD 995f8b51; drift C3-D1 to C3-D4 fixed in 502f661c).
- Test report: docs/reports/test-2026-10-28-org-merge-cleanup.md (the re-run: pass, HEAD be6415c2, `run-test.sh` rc 0, 10 of 10 mutants failed).
- After the merge of #287: `go build ./...`, `go vet` of `internal/org`, `internal/cli`, `internal/config`, and `internal/gitroot`, `check-skill-sync.sh`, `check-sync.sh`, and the narrowed tests of `internal/org` (`Cleanup|Merge|Skill|Prompt|StateDir|Feature`), `internal/config`, `internal/gitroot`, and `internal/cli` (`OrgCleanup|OrgMerge|Help|Config`) passed.
- Switch: `[standard].cross_review` read with a `ralph` built from this branch: `true`, exit 0.
- Reviewer run: a first run with `--base main` was stopped about a minute in, before it wrote anything, because the local `main` (5eba4013) lacked #287 and the diff would have held #287's changes. The run with `--base origin/main` exited 0 and wrote a non-empty `-o` file (1,578 bytes). In the git state before and after it, only another lane's branch (`refs/heads/feat/org-state-dir-local`) moved; this worktree's HEAD and status and the main checkout's were the same. The reviewer could not run Go tests in its sandbox.
- Checked in the code before classifying: the first check refuses only when the binding's own org runs (`cleanupFirstCheck`, `internal/org/cleanup.go:309-318`). A `spawned` event records the seat's `--cwd` in its `worktree` field (`internal/org/spawn.go:1204-1208`, `internal/org/manifest.go:28`), so a seat of another org started with `--cwd` inside the worktree can be found from the ledger (with `pathWithin`); a seat spawned with no `--cwd` has an empty one (`spawn.go:1333-1335`) and cannot be found that way. Finding 2's window is the known gap L6(b) in `docs/tech-debt/README.md`; that row says the start's last spawn is still refused by Spawn's own check, which holds only while the cleanup holds `regate.lock`, not after it has released it, as the reviewer says.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] A live seat of another org started with `--cwd` inside the feature worktree is not seen: the first check looks only at the binding's org, so once that org is disbanded the cleanup can remove the directory under the other org's running seat. Check active seats by worktree path and keep new spawns out of the worktree while the cleanup holds it. | Real, but narrow: a seat of another org working in a disbanded feature org's worktree is not a flow ralph sets up (each feature org gets its own worktree), and check 4 still refuses while anything is uncommitted or untracked there, so no unrecorded work is removed by the cleanup itself; what is lost is the other seat's working directory. A seat started with `--cwd` could be found from the `worktree` of its `spawned` event (the first check would read every running org's active seats and compare with `pathWithin`); one spawned with no `--cwd` records none and stays unseen. Both are outside D2's checks 0–6 of this plan. Recorded as a known gap in the PR, with that fix named. | `internal/org/cleanup.go` (`cleanupFirstCheck`), `internal/org/spawn.go` |
| 2 | [P2] A concurrent `StartFeature` can pass `Ensure` and its worktree check before the cleanup starts and reach `Spawn` after the cleanup finished; with no running seat recorded yet the cleanup goes on, and once the lock is released Spawn hands herdr the deleted working directory. Cover the whole ensure-to-spawn interval or revalidate the worktree when the start is recorded. | Real, and already a known gap of this plan (Progress line of S1, ledger row L6(b), the same window Regate has), left for a later change. The row's last clause is too optimistic: Spawn's check (`regateSpawnErr`) refuses only while the cleanup holds the lock and returns nil once it is released (`internal/org/spawn.go:1490-1491`). The outcome is a start whose seat gets a missing working directory and fails, with no commit lost. Recorded as a known gap; the PR's known gaps give the corrected form, and the row is to be corrected by the next change to `StartFeature`'s checks. | `internal/org/feature.go` (`StartFeature`), `internal/org/spawn.go`, `docs/tech-debt/README.md` (L6(b)) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
