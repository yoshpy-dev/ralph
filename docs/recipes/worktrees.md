# Worktrees

Ralph uses task worktrees by default for any flow that writes repo artifacts.
Read-only exploration may happen in the current checkout, but the first write to
specs, plans, code, tests, docs, reports, or temporary issue bodies must happen
inside a worktree created from a clean default branch.

## Contract

- Create task worktrees from the resolved default branch (`origin/HEAD`, then
  `main`/`master` fallback).
- Refuse to start if the default branch checkout is dirty.
- Store local orchestration state under
  `$(git rev-parse --git-common-dir)/ralph/worktrees/`, not in tracked files.
- Resume only when the existing branch, path, kind, and state id match.
- Treat branch/path mismatches as collisions; do not overwrite automatically.
- After successful PR hand-off, remove the task worktree and local branch.
- Do not delete the remote PR branch while the PR is open.
- Leave state behind when cleanup fails so `ralph-worktree.sh gc` can report
  and recover stale entries.

## Helper

Use `scripts/ralph-worktree.sh` instead of hand-written `git worktree` calls in
skills and scripts.

Common commands:

```sh
./scripts/ralph-worktree.sh default-branch
./scripts/ralph-worktree.sh validate-clean-base
./scripts/ralph-worktree.sh ensure --id plan-my-task --kind standard \
  --branch feat/my-task --path .claude/worktrees/my-task
./scripts/ralph-worktree.sh resume --id plan-my-task
./scripts/ralph-worktree.sh cleanup --id plan-my-task --force-branch
./scripts/ralph-worktree.sh gc
```

`--force-branch` is reserved for successful hand-off paths where the branch has
already been pushed and a PR exists. It is not a general-purpose cleanup escape
hatch.

## Spec Modes

- Issue-only specs use a temporary spec worktree, write the issue body inside
  that worktree, create the GitHub issue, then clean up the worktree and local
  branch.
- Saved specs write `docs/specs/<date>-<slug>.md`, commit it, create a docs/spec
  PR, then clean up.
- Saved specs that hand off to `/plan` keep using the same task worktree until
  the implementation PR succeeds.

## Org runtime

`ralph org spawn` records each seat's worktree path in the org manifest
(`.harness/state/org/manifest.jsonl`, `worktree` field). The pulse-layer
watchdog compares a seat's live `git status --porcelain` against its
recorded worktree to detect out-of-scope changes (see
`.claude/rules/ralph/agent-messaging.md` and `docs/specs/2026-08-01-org-runtime.md`).
