# Verify: worktree-first standard flow

- Date: 2026-05-14
- Branch: `feat/worktree-first-standard-flow`
- Related issue: #77
- Verdict: Pass

## Acceptance Criteria

| Criterion | Status | Evidence |
| --- | --- | --- |
| `/work` no longer creates a normal branch via `git checkout -b` | Pass | `.agents/skills/work/SKILL.md` and `.claude/skills/work/SKILL.md` now resolve/resume task worktrees. |
| `/plan` ensures a task worktree before writing plan artifacts | Pass | Plan skill now runs `ralph-worktree.sh current`/`ensure` before plan creation. |
| `/spec` separates issue-only, save-spec-file, and save-and-handoff modes | Pass | Spec skill output selection now defines all three modes and cleanup behavior. |
| Issue-only spec cleanup is contracted | Pass | Spec skill writes issue body inside spec worktree and cleans up after successful issue creation. |
| Saved spec file creates docs/spec PR and cleans up | Pass | Spec skill requires commit + docs/spec PR + cleanup for saved spec files. |
| `/pr` cleans up task worktree and local branch | Pass | PR skill runs `ralph-worktree.sh cleanup --id <state> --force-branch` after successful hand-off. |
| Cleanup failure leaves recoverable state | Pass | PR skill and helper leave git-common-dir state on cleanup failure. |
| Worktree state is under `git-common-dir` | Pass | `scripts/ralph-worktree.sh state-root` and docs use `$(git rev-parse --git-common-dir)/ralph/worktrees`. |
| Branch/path collision and resume are explicit | Pass | Helper tests cover matching resume, branch collision refusal, and dirty-base refusal. |
| Invalid absolute plan path recovery is documented | Pass | Cross-review skill attempts resume via `worktree_state_id` before fallback. |
| Ralph Loop compatibility preserved | Pass | Loop skill uses task worktree as control worktree and leaves per-slice worktrees to orchestrator. |
| Skill/template drift avoided | Pass | `check-skill-sync.sh` and `check-sync.sh` passed. |

## Static Verification

- `bash -n scripts/ralph-worktree.sh` passed.
- `bash -n templates/base/scripts/ralph-worktree.sh` passed.
- `sh -n tests/test-ralph-worktree.sh` passed.
- `./scripts/check-skill-sync.sh` passed.
- `./scripts/check-sync.sh` passed.
- `./scripts/check-pipeline-sync.sh` passed.
- `./scripts/run-static-verify.sh` passed.

Evidence:

- `docs/evidence/verify-2026-05-14-102417.log`

## Notes

An earlier sandboxed static verify failed because Go could not write to the user build cache. The same command passed when rerun with escalated permissions.
