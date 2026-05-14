# Sync-docs: worktree-first standard flow

- Date: 2026-05-14
- Branch: `feat/worktree-first-standard-flow`
- Related issue: #77
- Verdict: Pass

## Updated Surfaces

| Surface | Update |
| --- | --- |
| Skills | `/spec`, `/plan`, `/work`, `/loop`, `/cross-review`, and `/pr` now reference task worktree lifecycle and cleanup contracts. |
| Root docs | `AGENTS.md`, `CLAUDE.md`, `README.md`, `docs/quality/definition-of-done.md`, and recipes now describe worktree-first behavior. |
| Templates | `templates/base` mirrors updated for skills, docs, `.codex/README.md`, and `ralph-worktree.sh`. |
| Tests | `tests/test-ralph-worktree.sh` added and wired into `scripts/verify.local.sh`. |

## Drift Checks

- `./scripts/check-skill-sync.sh` passed.
- `./scripts/check-sync.sh` passed.
- `./scripts/check-pipeline-sync.sh` passed.

## Known Follow-up

- A future Go-native `ralph worktree` command could wrap the shell helper, but the current contract is implemented through the distributed script and skill workflow.
