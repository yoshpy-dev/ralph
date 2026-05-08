# Codex setup for ralph projects

This directory carries Codex CLI configuration. ralph treats Claude Code and
Codex as **first-class peers**: every skill that exists in `.claude/skills/`
also lives in `.agents/skills/`, and the post-implementation pipeline produces
the same artifacts no matter which CLI drove the work.

## One-time setup

1. **Install the Codex CLI** (>= 0.128.0).
   See [https://developers.openai.com/codex/cli](https://developers.openai.com/codex/cli).
2. **Trust this project** so Codex loads `.codex/config.toml` and the
   `[hooks]` block:

   ```sh
   codex trust .
   ```

   Without trust, `model`, `sandbox_mode`, `approval_policy`, and `[hooks]`
   are silently ignored.
3. **Verify the setup**:

   ```sh
   ralph doctor
   ```

   `ralph doctor` checks that the Codex CLI is on `$PATH`, that the project is
   trusted, that `[features] codex_hooks = true` is set, and that at least one
   `[hooks]` entry is visible to Codex.

## Daily usage

Start Codex from the project root and invoke a ralph skill by mention:

```
codex
> $spec describe the change you want to scope
> $plan
> $work
```

Use the `/skills` menu to browse skills if you forget a name. Do **not** type
`/spec`, `/plan`, etc. — `/plan` (and several others) are Codex built-ins and
will not run the ralph skill.

## How Codex differs from Claude Code in this harness

| Concern | Claude Code | Codex |
|---------|-------------|-------|
| Skill invocation | `/skill-name` slash | `$skill-name` mention or `/skills` menu |
| Subagents in `/work` post-impl | `Task(subagent_type=...)` parallel | Sequential, in this single agent |
| Interactive choices | `AskUserQuestion` | Numbered prompt + single-digit reply |
| Cross-model second opinion | `/cross-review` calls `codex exec review` | `/cross-review` calls `claude -p` |
| Permission policy | `permission_mode = auto` | `sandbox_mode = workspace-write` + `approval_policy = on-request` |

`scripts/check-skill-sync.sh` keeps the `.claude/skills/` and `.agents/skills/`
trees in lock-step. CI fails on drift, so fix both sides whenever you change
either.

## Upgrading

Before running `ralph upgrade`, commit local changes (or take a backup) so the
hash-based diff engine can be replayed cleanly. Skill renames are surfaced as
`add` + `remove` pairs and may need a manual review on the first upgrade.

## Hooks

Project-level Codex hooks live in `.codex/config.toml` under `[hooks]`. They
shell out to the same scripts under `.claude/hooks/`, so behaviour stays
identical across the two CLIs.

The template ships **default-on** with two `PostToolUse` hooks that point at
`./.claude/hooks/check_mojibake.sh` (one for `Edit`, one for `Write`). These
satisfy `ralph doctor`'s "at least one `[hooks]` entry visible" check on a
fresh `ralph init` and reuse the same script the Claude side calls, so a
single edit to `check_mojibake.sh` covers both CLIs.

To extend the hook surface, add new `[[hooks.<event>]]` entries that point at
real scripts — and add the matching Claude-side hook in `.claude/settings.json`
when behaviour parity matters. `scripts/commit-msg-guard.sh` is intentionally
**not** wired as a Codex `PostToolUse` hook: it is a git `commit-msg` hook
(consumes `$1` = path to `COMMIT_EDITMSG`) and would exit 1 on every commit if
attached to `^git commit`. Install it as `.git/hooks/commit-msg` instead, or
write a Codex-shaped wrapper before adding a `PostToolUse` entry.
