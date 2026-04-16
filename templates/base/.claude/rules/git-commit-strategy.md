# Git Commit Strategy

Single source of truth for when and how to commit in all flows.

## Validation Gate (Standard Flow — /work)

Commit after each slice passes verification, not at the end:

1. Implement a slice (one or more acceptance criteria)
2. Run `./scripts/run-verify.sh` (or equivalent)
3. If verification passes → stage and commit with conventional format
4. If verification fails → fix before committing
5. Repeat for the next slice

This produces a clean history of individually verified changes.

## Ralph Loop Commits

Each iteration must commit its changes before finishing:

1. Implement the iteration's step
2. Verify the change (static analysis, tests)
3. Commit with conventional format: `<type>: <description>`
4. Append summary to `progress.log`
5. Do NOT leave uncommitted changes between iterations

In Ralph Loop (`ralph-pipeline.sh` per slice), the implementation agent (Inner Loop) is responsible for committing each step. The pipeline detects uncommitted changes via `git diff-index --quiet HEAD` after each iteration and warns if any remain. Documentation changes from the Outer Loop (sync-docs) are committed by the pipeline-outer agent before PR creation.

## End-of-Session / Pre-Compaction WIP Commits

When a session ends or context compaction occurs on a feature branch:

- Automatically commit uncommitted changes with `wip:` prefix
- Only on feature branches (never on main/master)
- Message format: `wip: checkpoint before <reason>`
- These are safe to squash later

## Safety Bracket (Guidance)

Before risky or hard-to-reverse operations:

- Large refactors spanning many files
- Dependency upgrades
- Schema migrations
- Configuration changes affecting multiple systems

Create a checkpoint commit first: `chore: checkpoint before <operation>`.
This is guidance, not enforced by tooling.

## Safe Quoting for Commit Messages

Shell command substitution inside double-quoted `git commit -m "..."` can leak secrets. For example, backticks or `$(...)` inside double quotes are interpreted by the shell, potentially expanding environment variables (API keys, tokens) into the commit message.

**Background:** In the Apache httpd incident, a commit message containing `` `set` `` inside double quotes caused all environment variables to be expanded and published in the commit history.

**Rules:**

- **Multiline messages:** Always use HEREDOC with single-quoted delimiter:
  ```sh
  git commit -m "$(cat <<'EOF'
  feat: add feature

  Details here with `backticks` safely preserved.
  EOF
  )"
  ```
- **Single-line messages:** Prefer single quotes: `git commit -m 'fix: description'`
- **Never:** Place backticks or `$(...)` inside double-quoted `-m "..."` arguments
- **Enforcement:** `pre_bash_guard.sh` blocks dangerous patterns at command time; `commit-msg-guard.sh` scans for leaked secrets at commit time

## Commit Format

Follow Conventional Commits (see `git-workflow.md`):

```
<type>: <description>

<optional body>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`, `wip`

The `wip` type is reserved for automated checkpoints only.
