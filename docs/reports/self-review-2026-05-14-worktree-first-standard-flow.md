# Self-review: worktree-first standard flow

- Date: 2026-05-14
- Branch: `feat/worktree-first-standard-flow`
- Related issue: #77
- Scope: diff quality review only

## Verdict

Pass. No blocking findings.

## Findings

| Severity | Area | Finding | Resolution |
| --- | --- | --- | --- |
| LOW | lifecycle clarity | Initial draft used separate `plan-<slug>` and `spec-<slug>` state paths, which could create a second worktree during spec-to-plan handoff. | Added `ralph-worktree.sh current` and updated `/plan`, `/work`, and `/loop` contracts to adopt an existing current task worktree before ensuring a new one. |

## Review Notes

- `scripts/ralph-worktree.sh` keeps local absolute paths under `git-common-dir`, outside tracked files.
- Cleanup removes the worktree and local branch only through explicit state; mismatched state refuses to overwrite.
- `/spec` now separates issue-only, docs/spec PR, and implementation handoff lifecycles.
- `/plan`, `/work`, `/loop`, `/cross-review`, and `/pr` now consume or preserve pinned worktree state rather than rediscovering plans casually.
- Root/template and Claude/Codex skill mirrors were kept in sync.

## Residual Risk

- The helper is shell-level orchestration, not yet wired into a Go CLI command. The contract is enforceable by skills and shell tests, and a future `ralph worktree` Go command could wrap the script if needed.
