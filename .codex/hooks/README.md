# `.codex/hooks/` is intentionally empty

ralph does not ship Codex-specific hook scripts here. The default-on Codex
hooks (`PostToolUse` mojibake guard on `Edit` / `Write` / `MultiEdit` /
`apply_patch`; `PreToolUse` bash guard; `SessionStart` and `UserPromptSubmit`
additionalContext hooks) are wired in **`.codex/hooks.json`**, not in this
directory and not in `.codex/config.toml`. `hooks.json` routes all four
events through `./.claude/hooks/ralph-dispatch.sh`, the same layered `.d`
dispatcher Claude Code uses, so a single drop-in script under the **shared
`.claude/hooks/` tree** covers both agents.

This directory is reserved for the case where a Codex-shaped wrapper is
required — e.g. a hook whose CLI contract differs enough from Claude's
PreToolUse / PostToolUse JSON-stdin convention that sharing the script is
not viable. When that day comes, put the adapter where the dispatcher
already looks — `.ralph/local/hooks/<event>.d/` (committed, project-wide)
or `.claude/hooks/local/<event>.d/` (machine-local, gitignored) — so it
runs through `ralph-dispatch.sh` like every other hook; referencing a
script here directly from `.codex/hooks.json` is flagged by `ralph
doctor` as a bypass of the dispatcher.

Until then, the directory exists only to make the convention visible
alongside the populated `.claude/hooks/` tree.

`.codex/hooks.json` is the source of truth for Codex project hooks: on the
codex-cli release this was verified against, project-scoped inline
`[[hooks.*]]` entries in `.codex/config.toml` never fired while the
equivalent `hooks.json` entry did. Do not add a `[hooks]` / `[[hooks.*]]`
table to `.codex/config.toml` — `ralph doctor` flags a surviving table as a
stale duplicate representation.

See also:
- `.codex/hooks.json` — the wired `PostToolUse` / `PreToolUse` /
  `SessionStart` / `UserPromptSubmit` groups.
- `.codex/README.md` — operator-facing hooks guidance, including the
  interactive hook-trust UX.
- `.claude/hooks/` — the shared script bodies.
