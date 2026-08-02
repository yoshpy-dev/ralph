# Codex setup for ralph projects

This directory carries Codex configuration. ralph treats Claude Code and
Codex as **first-class peers**: every skill that exists in `.claude/skills/`
also lives in `.agents/skills/`, and the post-implementation pipeline produces
the same artifacts no matter which agent drove the work.

## One-time setup

1. **Install Codex** (>= 0.128.0).
   See [https://developers.openai.com/codex/cli](https://developers.openai.com/codex/cli).
2. **Trust this project** so Codex loads `.codex/config.toml` and the inline
   `[[hooks.*]]` entries:

   ```sh
   codex trust .
   ```

   Without trust, `model`, `sandbox_mode`, `approval_policy`, and project
   hooks are silently ignored.
3. **Verify the setup**:

   ```sh
   ralph doctor
   ```

   `ralph doctor` checks that the `codex` binary is on `$PATH`, that the project is
   trusted, that `[features] hooks = true` is set, and that at least one
   `[hooks]` entry is visible to Codex.

## Daily usage

Start Codex from the project root and invoke a ralph skill by mention:

```
codex
> $spec describe the change you want to scope
> $plan
> $work
```

Spec, plan, and work flows create or resume clean-base task worktrees before
writing repo artifacts; PR hand-off cleans up the task worktree and local
branch.

Use the `/skills` menu to browse skills if you forget a name. Do **not** type
`/spec`, `/plan`, etc. — `/plan` (and several others) are Codex built-ins and
will not run the ralph skill.

## How Codex differs from Claude Code in this harness

| Concern | Claude Code | Codex |
|---------|-------------|-------|
| Skill invocation | `/skill-name` slash | `$skill-name` mention or `/skills` menu |
| Subagents in `/work` post-impl | `Task(subagent_type=...)` calls | `.codex/agents/` custom agents with the same phase roles |
| Interactive choices | `AskUserQuestion` | Numbered prompt + single-digit reply |
| Cross-model second opinion | `/cross-review` calls `codex exec review` | `/cross-review` calls `claude -p` |
| Org seat permission policy | `ralph.toml` `[org.permissions] default = "autonomous"` (or `"edits"` / `"guarded"`; per-role overrides supported) | same enum, mapped to Codex's own `sandbox_mode` + `approval_policy` flags |

`scripts/check-skill-sync.sh` keeps the `.claude/skills/` and `.agents/skills/`
trees in lock-step. CI fails on drift, so fix both sides whenever you change
either.

## Agent roles

Codex role definitions live in `.codex/agents/`:

- `implementer` — scoped implementation worker; receives structured handoff from the orchestrator during `/work` step 6; stages only handoff-listed paths, runs verification, and returns a report with commit-boundary evidence. Like the other Codex custom agents, no per-agent model is pinned here — the `sonnet` tier applies to the Claude Code counterpart (`.claude/agents/implementer.md`); Codex runs follow the session/config model
- `reviewer` — diff quality only (post-implementation)
- `verifier` — acceptance criteria, docs drift, and static checks (post-implementation)
- `tester` — behavioral tests and failure analysis (post-implementation)
- `doc-maintainer` — plans, docs, templates, and reports (post-implementation)

The standard Codex flow uses `reviewer`, `verifier`, `tester`, and
`doc-maintainer` in the canonical post-implementation order. If Codex cannot
dispatch a subagent, run that step inline and record the fallback in the report.

## Upgrading

Before running `ralph upgrade`, commit local changes (or take a backup) so the
hash-based diff engine can be replayed cleanly. Skill renames are surfaced as
`add` + `remove` pairs and may need a manual review on the first upgrade.

## Hooks

Project-level Codex hooks live in `.codex/config.toml` as inline
`[[hooks.*]]` entries. They shell out to the same scripts under
`.claude/hooks/`, so behaviour stays identical across the two agents.

Do not add `.codex/hooks.json` beside inline hooks in `.codex/config.toml`.
Codex loads hooks per configuration layer, and two hook representations in the
same `.codex/` layer trigger a startup warning about duplicate hook loading.
`ralph doctor` and the local verifier check this so the duplicate
representation does not come back silently.

The template ships **default-on** with two `PostToolUse` hooks that point at
`./.claude/hooks/check_mojibake.sh` (one for `Edit`, one for `Write`). These
satisfy `ralph doctor`'s "at least one `[hooks]` entry visible" check on a
fresh `ralph init` and reuse the same script the Claude side calls, so a
single edit to `check_mojibake.sh` covers both agents.

To extend the hook surface, add new `[[hooks.<event>]]` entries that point at
real scripts, keep commands relative to the repo, and add the matching
Claude-side hook in `.claude/settings.json` when behaviour parity matters.
The secret-guard scripts are intentionally **not** wired as Codex
`PostToolUse` hooks. They are real git hooks:
`scripts/pre-commit-secret-guard.sh`, `scripts/commit-msg-guard.sh`, and
`scripts/prepare-commit-msg-secret-guard.sh`, plus
`scripts/pre-merge-commit-secret-guard.sh` for automatic merge commits.
`ralph init` and `ralph upgrade` install them into Git's hook directory. If a
local hook already exists, Ralph keeps it as `<hook>.ralph-original` and runs it
after the guard.
