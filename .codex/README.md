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
   trusted, that `[features] hooks` is not explicitly disabled or malformed
   (`hooks = false`, or a non-boolean value such as a quoted "false"; an
   absent key is left to Codex's own undocumented default), and that
   `.codex/hooks.json` exists, parses, matches the expected schema, and routes
   at least one event through the dispatcher. Its "Codex hook trust" check
   looks up the approval of each hook in codex's user config (see "Trust
   UX" below).

## Daily usage

Start Codex from the project root and invoke a ralph skill by mention:

```
codex
> $spec describe the change you want to scope
> $plan
> $implement
```

Spec, plan, and implement flows create or resume clean-base task worktrees before
writing repo artifacts; PR hand-off cleans up the task worktree and local
branch.

Use the `/skills` menu to browse skills if you forget a name. Do **not** type
`/spec`, `/plan`, etc. — `/plan` (and several others) are Codex built-ins and
will not run the ralph skill.

## How Codex differs from Claude Code in this harness

| Concern | Claude Code | Codex |
|---------|-------------|-------|
| Skill invocation | `/skill-name` slash | `$skill-name` mention or `/skills` menu |
| Subagents in `/implement` post-impl | `Task(subagent_type=...)` calls | `.codex/agents/` custom agents with the same phase roles |
| Interactive choices | `AskUserQuestion` | Numbered prompt + single-digit reply |
| Cross-model second opinion | `/cross-review` calls `codex exec review` | `/cross-review` calls `claude -p` |
| Org seat permission policy | `ralph.toml` `[org.permissions] default = "autonomous"` (or `"edits"` / `"guarded"`; per-role overrides supported) | same enum, mapped to Codex's own `sandbox_mode` + `approval_policy` flags |

`scripts/check-skill-sync.sh` keeps the `.claude/skills/` and `.agents/skills/`
trees in lock-step. CI fails on drift, so fix both sides whenever you change
either.

## Agent roles

Codex role definitions live in `.codex/agents/`:

- `implementer` — scoped implementation worker; receives structured handoff from the orchestrator during `/implement` step 6; stages only handoff-listed paths, runs the slice's verification command from the plan, does a mechanical self-check on the staged diff (unnecessary changes, typos and copy-paste errors, debug leftovers, secrets, and files outside the slice), and returns a report with the self-check result and commit-boundary evidence. Like the other Codex custom agents, no per-agent model is pinned here — the Claude Code counterpart (`.claude/agents/implementer.md`) takes its model from its frontmatter, with the tier table in `.claude/rules/ralph/model-routing.md`; Codex runs follow the session/config model
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
`.codex/config.toml`. `.codex/hooks.json` routes four events —
`PostToolUse`, `PreToolUse`, `SessionStart`, and `UserPromptSubmit` — through
`./.claude/hooks/ralph-dispatch.sh`, the same layered `.d` dispatcher Claude
Code uses (`.claude/hooks/<event>.d/` core, then `.ralph/local/hooks/<event>.d/`
and `.claude/hooks/local/<event>.d/` for downstream drop-ins), so a single
drop-in script runs under both agents. Every shipped hook group's command
first `cd`s to the git root and then invokes the dispatcher by a
repo-relative path (`cd "$(git rev-parse --show-toplevel)" &&
./.claude/hooks/ralph-dispatch.sh <event>`). This matters because
ralph-dispatch.sh resolves its `.d` layers relative to its own cwd: a bare
absolute path to the dispatcher (no `cd`) would run the dispatcher fine but
leave it looking for `.claude/hooks/<event>.d/` etc. relative to whatever
directory the Codex session was launched from, so a session started from a
subdirectory would fire the hook against an empty layer set and silently run
zero scripts. The `cd`-first form keeps the dispatcher's cwd-relative
contract intact regardless of where the session starts.

- `PostToolUse` matcher is `Edit|Write|MultiEdit|apply_patch` — the payload
  reports `tool_name=apply_patch`, so the literal name is included as the
  conservative default; `Edit`/`Write`/`MultiEdit` are kept for readability
  and Claude-side parity and are also accepted.
- `PreToolUse` matcher is `Bash` — live-fire confirmed the real Codex tool
  name for shell execution is `Bash` and that a `deny` decision from
  `pre_bash_guard.sh` actually blocks the command (not just a warning), so
  this is a real enforcement hook, not a cosmetic one. `deny` is the only
  decision the guard returns: it never asks for confirmation and does not
  read `permission_mode`, so it behaves the same in every Codex approval mode.
- `SessionStart` and `UserPromptSubmit` omit a matcher (all sources /
  prompts match); both drive additionalContext-emitting hooks
  (`session_start_context.sh`, `prompt_gate.sh`). `prompt_gate.sh` is
  output-only. `session_start_context.sh` also performs idempotent
  harness-state housekeeping on every run: it creates the `.harness/` and
  `docs/plans/`/`docs/reports/` scaffold directories and resets
  `.harness/state/tool_failures.count` to `0`. Codex has no
  `PostToolUseFailure` event, so under Codex that counter is only ever reset,
  never incremented — the reset is inert for a pure-Codex session but can
  clear a concurrent Claude Code session's failure counter in the same repo.
- Codex has **no `PostToolUseFailure` event** — that hook stage is Claude
  Code-specific. The corresponding `.claude/hooks/PostToolUseFailure.d/`
  layer simply never runs under Codex; there is nothing to wire here.
- `SessionEnd` and `PreCompact` are **deliberately not wired**: their
  Claude-side hooks (`session_end_summary.sh`, `precompact_checkpoint.sh`)
  auto-commit a dirty working tree as a `wip:` checkpoint, and that side
  effect needs its own safety acceptance criteria (dirty-tree behavior,
  rollback) before it ships to Codex sessions.

`.codex/config.toml` keeps a reference comment pointing at `hooks.json` but
no `[[hooks.*]]` table; do not reintroduce one there. Codex merges both
representations when present and emits a startup warning about duplicate
hook loading, and `ralph doctor` flags the stale dual representation if it
comes back.

### Trust UX

Codex hooks only run after a **one-time interactive trust approval**, on top
of the project config trust from step 2 above:

1. `codex trust .` makes `.codex/config.toml` and `[features] hooks = true`
   load at all — but that alone does not approve any hook.
2. Run `codex` **interactively** at least once in the project. On first use,
   Codex asks you to review and approve each hook command (use the `/hooks`
   command inside the session if you need to trigger that review manually).
   Codex records each approval in `[hooks.state]` of its user config
   (`$CODEX_HOME/config.toml`, else `~/.codex/config.toml`) under the key
   `<absolute path of hooks.json>:<event in snake_case>:<group index>:<handler index>`,
   such as `/path/to/project/.codex/hooks.json:pre_tool_use:0:0`. The
   entry's `trusted_hash` is a hash of the hook's normalized declaration
   (its event, its matcher, and its handler, fields such as `timeout`
   included), not of the `command` string alone. An entry with
   `enabled = false` turns its hook off: the hook does not run although it
   is approved.
3. **Without that approval, non-interactive `codex exec` silently skips
   untrusted hooks** — no warning, no error, the hook just does not run. If a
   hook you expect to fire under `codex exec` (for example in CI) does not,
   check trust state first.
4. Because the key names the hook's place in `hooks.json` and the hash
   covers its whole declaration, editing any part of a hook's declaration
   (its `command`, its matcher, a field such as `timeout`), moving it to
   another group or handler index, or adding a new hook leaves that hook
   without a valid approval: it needs a fresh interactive `codex` session
   (or `--dangerously-bypass-hook-trust` for non-interactive `codex exec`
   runs, e.g. CI) before it actually fires. Approvals of hooks whose
   declaration and index did not change survive.
5. In a linked worktree (a task worktree, a feature org's worktree), codex
   reads the hooks of the main checkout's `.codex/hooks.json` and looks
   their approvals up under the main checkout's path, so approving the
   hooks once in the main checkout covers its worktrees (checked with codex
   0.166.0). Opening the main checkout through a symlinked path makes codex
   use that path in the keys, so an approval given under the real path does
   not count there, and the other way round. codex also reads the
   `.codex/hooks.json` of each subdirectory between the checkout's root and
   the directory it runs in (that directory included), each keyed under its
   own path (from a linked worktree, the same subdirectory's path in the
   main checkout), and ralph checks those files too.

`ralph doctor` validates `hooks.json` itself (present, valid JSON,
schema-conformant, each shipped event routed through the dispatcher with
that event's own name as the dispatcher argument — not just referenced by
basename). Its "Codex hook trust" check, which runs only when `codex` is in
`[org].driver_pool`, reads the `hooks.json` codex reads (the main
checkout's, from a linked worktree) and looks up the key of every handler
in `[hooks.state]` of codex's user config, under both paths when the main
checkout is reached through a symlink. It warns when a hook is not approved
or is turned off, or when that config does not exist, and says how to
approve the hooks: open `codex` in the directory it names and choose "Trust
all and continue" in the "Hooks need review" dialog. It passes when every
key is there with a `trusted_hash` and none is turned off. It does not
compare the hash, so a hook changed since its approval passes the check;
codex then shows its "Hooks need review" dialog in the seat's pane, and
`ralph org resume` treats that dialog as a choice dialog and types nothing
into it. `ralph org spawn`, `ralph org start`, and `ralph org director
start` print one note on stderr when they launch codex and a hook is not
approved or is turned off; they go on either way.

To extend the hook surface, add a new event group to `.codex/hooks.json`
(top-level `hooks` → event name → matcher-group array → `{type: "command",
command: <string>}` handlers), keep the command `cd`-to-git-root-first like
the shipped entry (see above — a bare absolute path to the target script
runs fine but breaks any cwd-relative lookups inside it), and add the
matching Claude-side hook in
`.claude/settings.json` when behaviour parity matters. The secret-guard
scripts are intentionally **not** wired as Codex
`PostToolUse` hooks. They are real git hooks:
`scripts/pre-commit-secret-guard.sh`, `scripts/commit-msg-guard.sh`, and
`scripts/prepare-commit-msg-secret-guard.sh`, plus
`scripts/pre-merge-commit-secret-guard.sh` for automatic merge commits.
`ralph init` and `ralph upgrade` install them into Git's hook directory. If a
local hook already exists, Ralph keeps it as `<hook>.ralph-original` and runs it
after the guard.
