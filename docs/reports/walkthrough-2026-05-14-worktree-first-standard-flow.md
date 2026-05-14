# Walkthrough: Worktree-first standard flow

- Date: 2026-05-14
- Branch: `feat/worktree-first-standard-flow`
- Issue: #77
- Verdict: ready for review

## Scope

This change moves the standard flow to a worktree-first contract and keeps
Ralph Loop aligned with the same lifecycle language. It also adds a
deterministic helper for task worktree creation, resume, cleanup, and stale
state collection.

## Main areas

1. Worktree helper
   - Added `scripts/ralph-worktree.sh`.
   - Added the same helper to `templates/base/scripts/ralph-worktree.sh`.
   - Added `tests/test-ralph-worktree.sh`.
   - Registered the helper in scaffold embed coverage and local verify.

2. Skill contracts
   - Updated spec, plan, work, loop, cross-review, and PR skills under both
     `.agents/skills/` and `.claude/skills/`.
   - Mirrored the same updates into `templates/base/`.
   - Standard flow now creates or resumes a clean-base task worktree before
     writing spec or plan artifacts.
   - PR flow now requires task worktree and local branch cleanup after the
     remote PR branch is pushed and the PR passes ready/title checks.

3. Spec modes
   - Issue-only mode creates temporary issue body artifacts inside a task
     worktree, creates the issue, and immediately cleans up.
   - Save spec file mode writes `docs/specs/*.md`, creates a docs PR, and
     cleans up.
   - Save-and-handoff mode keeps the same task worktree available for plan and
     implementation.

4. Documentation
   - Updated root and template `AGENTS.md`, `CLAUDE.md`, README files,
     recipes, quality gates, and subagent/pipeline rules.
   - Clarified clean base, branch/path collision, resume, absolute plan path,
     PR cleanup, and local branch deletion expectations.

## Verification

- `./scripts/check-skill-sync.sh`
- `./scripts/check-sync.sh`
- `./scripts/check-pipeline-sync.sh`
- `./scripts/run-static-verify.sh`
- `./scripts/run-test.sh`
- `tests/test-ralph-worktree.sh`
- `git diff --cached --check`

The first sandboxed static verify attempt could not write to the Go cache
outside the sandbox. The same verify command passed when rerun with the
required permissions.

## Review focus

- `scripts/ralph-worktree.sh` cleanup behavior, especially dirty worktree
  refusal and local branch deletion.
- Standard-flow skill sequencing around spec issue-only cleanup and
  save-spec-file docs PR cleanup.
- Template parity for every changed skill and workflow doc.
