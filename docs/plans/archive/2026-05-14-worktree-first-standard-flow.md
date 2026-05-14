# worktree-first-standard-flow

- Status: Draft
- Owner: Claude Code
- Date: 2026-05-14
- Related request: https://github.com/yoshpy-dev/ralph/issues/77
- Related issue: 77
- Type: feat
- Branch: feat/worktree-first-standard-flow

## Objective

Move the standard development flow to a worktree-first contract so spec, plan,
implementation, pipeline, and PR artifacts are produced outside the default
checkout, while preserving Ralph Loop's existing multi-worktree behavior.

## Scope

- Define a reusable task worktree lifecycle for standard flow and spec flow.
- Add deterministic worktree helper tooling for state path, default-branch,
  clean-base validation, ensure, cleanup, and gc operations.
- Update `/spec`, `/plan`, `/work`, `/loop`, `/cross-review`, and `/pr` skill
  bodies for the new contract.
- Sync `.claude/skills/` and `.agents/skills/`.
- Update docs and templates that describe branch/worktree behavior.
- Add focused shell tests for the new helper and run existing sync gates.

## Non-goals

- Do not delete remote PR branches immediately after PR creation.
- Do not force-delete local branches or worktrees automatically on cleanup
  failure.
- Do not make local worktree state portable across clones.
- Do not rewrite Ralph Loop's per-slice orchestrator architecture.

## Assumptions

- `git rev-parse --git-common-dir` is available in supported Git versions.
- Local worktree state belongs outside tracked files because it stores absolute
  paths and cleanup status.
- Issue-only specs still require an isolated worktree because they create
  temporary issue body artifacts.
- A docs/spec PR is acceptable for saved specs that do not hand off to
  implementation.

## Affected areas

- `scripts/ralph-worktree.sh`
- `.claude/skills/{spec,plan,work,loop,cross-review,pr}/SKILL.md`
- `.agents/skills/{spec,plan,work,loop,cross-review,pr}/SKILL.md`
- `CLAUDE.md`, `AGENTS.md`, `README.md`
- `docs/recipes/worktrees.md`, `docs/recipes/ralph-loop.md`
- `.claude/rules/post-implementation-pipeline.md`
- `templates/base/` mirrors for affected docs, skills, and scripts
- `scripts/check-sync.sh`, `scripts/check-skill-sync.sh`, related tests

## Design decisions

- Worktree state lives under `$(git rev-parse --git-common-dir)/ralph/worktrees`
  so all linked worktrees can read it without tracking local absolute paths.
- PR cleanup removes the task worktree first, then attempts safe local branch
  deletion with `git branch -d`. It never force-deletes by default.
- `/spec` has three concrete output modes: issue-only with immediate cleanup,
  save-spec-file with docs/spec PR, and save-spec-file plus `/plan` handoff.
- Downstream standard-flow skills consume pinned state instead of rediscovering
  plans by scanning active plan directories.

## Acceptance criteria

- [x] Standard `/work` no longer says it creates a normal branch via
      `git checkout -b`; it requires/resumes a task worktree.
- [x] `/plan` ensures a task worktree before writing plan artifacts and pins
      absolute plan path plus worktree metadata.
- [x] `/spec` documents issue-only, save-spec-file, and save-and-handoff
      worktree lifecycles.
- [x] Issue-only `/spec` creates and then cleans up a spec worktree/local
      branch after successful issue creation.
- [x] Save-spec-file `/spec` creates a docs/spec PR and cleans up the
      worktree/local branch after PR creation.
- [x] `/pr` cleans up the task worktree and local branch after successful PR
      creation, leaving state behind on cleanup failure.
- [x] Worktree state is stored under `git-common-dir`, not in tracked files.
- [x] Branch/path collision and existing worktree resume behavior is explicit.
- [x] Invalid pinned absolute plan paths have a documented recovery path.
- [x] Ralph Loop's multi-worktree slice execution remains compatible.
- [x] Skill mirrors and template mirrors stay in sync.

## Implementation outline

1. Add `scripts/ralph-worktree.sh` with state, validation, ensure, cleanup,
   and gc subcommands.
2. Update standard-flow and spec skill contracts to call the helper at the
   first write boundary.
3. Update PR and cross-review pinned-state behavior.
4. Sync mirrored skills and template files.
5. Add focused shell tests and run sync/verification gates.

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` or
  `./scripts/run-verify.sh`; shell syntax checks for new scripts.
- Spec compliance criteria to confirm: every Issue #77 acceptance criterion has
  a corresponding skill/doc/script change.
- Documentation drift to check: `.claude` vs `.agents`, root vs
  `templates/base`, README/AGENTS/CLAUDE consistency.
- Evidence to capture: command outputs for helper tests, sync checks, verify,
  and test.

## Test plan

- Unit tests: shell tests for `ralph-worktree.sh` path/state/default branch and
  collision behavior.
- Integration tests: run sync checks and broad repository verification.
- Regression tests: existing `scripts/check-skill-sync.sh` and
  `scripts/check-sync.sh`.
- Edge cases: dirty base refusal, stale state cleanup, branch/path collisions,
  missing absolute plan path recovery docs.
- Evidence to capture: test logs under `docs/evidence/`.

## Risks and mitigations

- Risk: cleanup could remove useful local work. Mitigation: safe deletion only;
  no automatic `-D` or forced worktree removal on unexpected state.
- Risk: docs-only contract without tooling drifts. Mitigation: add helper script
  and shell tests, then wire skills to the helper.
- Risk: worktree paths inside `.claude/worktrees` confuse parent status.
  Mitigation: existing `.gitignore` already excludes `.claude/worktrees/`.

## Rollout or rollback notes

Roll out as a harness contract change with helper tooling. Roll back by
reverting the helper and restoring `/work` branch-first language.

## Open questions

- Whether a future `ralph spec` command should wrap the `/spec` issue-only and
  docs/spec PR modes deterministically.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
