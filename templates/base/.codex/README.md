# Codex setup for ralph projects

This directory carries Codex configuration. ralph treats Claude Code and
Codex as **first-class peers**: every skill that exists in `.claude/skills/`
also lives in `.agents/skills/`, and the post-implementation pipeline produces
the same artifacts no matter which agent drove the work.

## One-time setup

1. **Install Codex** (>= 0.128.0).
   See [https://developers.openai.com/codex/cli](https://developers.openai.com/codex/cli).
2. **Trust this project** so Codex loads `.codex/config.toml` (and, in turn,
   `.codex/hooks.json`):

   ```sh
   codex trust .
   ```

   Without trust, `model`, `sandbox_mode`, `approval_policy`, and project
   hooks are silently ignored. Project trust is necessary but not
   sufficient for hooks — see "Hooks" below for the separate, per-hook
   interactive approval Codex also requires.
3. **Verify the setup**:

   ```sh
   ralph doctor
   ```

   `ralph doctor` checks that the `codex` binary is on `$PATH`, that the project is
   trusted, that `[features] hooks = true` is set, and that `.codex/hooks.json`
   exists, parses, matches the expected schema, and routes at least one event
   through the dispatcher. It cannot check interactive hook-trust state (see
   "Hooks" below).

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

Project-level Codex hooks live in `.codex/hooks.json`, not in
`.codex/config.toml`. `.codex/hooks.json` routes `PostToolUse` through
`./.claude/hooks/ralph-dispatch.sh`, the same layered `.d` dispatcher Claude
Code uses (`.claude/hooks/<event>.d/` core, then `.ralph/local/hooks/<event>.d/`
and `.claude/hooks/local/<event>.d/` for downstream drop-ins), so a single
drop-in script runs under both agents. The shipped hook group resolves the
dispatcher path from the git root
(`"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh" PostToolUse`)
rather than a bare relative path, and matches `Edit|Write|MultiEdit|apply_patch`
— Codex reports its file-edit tool as `apply_patch`, so the literal tool name
has to be in the matcher for the hook to fire. `.codex/config.toml` keeps a
reference comment pointing at `hooks.json` but no `[[hooks.*]]` table; do not
reintroduce one there. Codex merges both representations when present and
emits a startup warning about duplicate hook loading, and `ralph doctor` /
`tests/test-hook-wiring.sh` both flag the stale dual representation if it
comes back.

### Trust UX

Codex hooks only run after a **one-time interactive trust approval**, on top
of the project config trust from step 2 above:

1. `codex trust .` makes `.codex/config.toml` and `[features] hooks = true`
   load at all — but that alone does not approve any hook.
2. Run `codex` **interactively** at least once in the project. On first use,
   Codex asks you to review and approve each hook command (use the `/hooks`
   command inside the session if you need to trigger that review manually).
   Approval is keyed to a hash of the exact `command` string in
   `hooks.json`.
3. **Without that approval, non-interactive `codex exec` silently skips
   untrusted hooks** — no warning, no error, the hook just does not run. If a
   hook you expect to fire under `codex exec` (for example in CI) does not,
   check trust state first.
4. Because trust is keyed by the command string's hash, editing a hook's
   `command` in `hooks.json` invalidates the prior approval — run another
   interactive session and re-approve after any such change.

`ralph doctor` validates `hooks.json` itself (present, valid JSON,
schema-conformant, routed through the dispatcher) but cannot probe
interactive trust state, so a clean `ralph doctor` run does not guarantee a
hook will actually fire under `codex exec` until you have approved it once.

To extend the hook surface, add a new event group to `.codex/hooks.json`
(top-level `hooks` → event name → matcher-group array → `{type: "command",
command: <string>}` handlers), keep the command git-root-resolved like the
shipped entry, and add the matching Claude-side hook in
`.claude/settings.json` when behaviour parity matters. The secret-guard
scripts are intentionally **not** wired as Codex
`PostToolUse` hooks. They are real git hooks:
`scripts/pre-commit-secret-guard.sh`, `scripts/commit-msg-guard.sh`, and
`scripts/prepare-commit-msg-secret-guard.sh`, plus
`scripts/pre-merge-commit-secret-guard.sh` for automatic merge commits.
`ralph init` and `ralph upgrade` install them into Git's hook directory. If a
local hook already exists, Ralph keeps it as `<hook>.ralph-original` and runs it
after the guard.
