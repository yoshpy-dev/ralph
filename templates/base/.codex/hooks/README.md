# `.codex/hooks/` is intentionally empty

ralph does not ship Codex-specific hook scripts here. The default-on Codex
hooks (`PostToolUse` mojibake guard on `Edit` / `Write`) are wired in
`.codex/config.toml` inline `[[hooks.*]]` entries and call into the
**shared scripts under `.claude/hooks/`** so a single edit covers both agents:

```toml
[[hooks.PostToolUse]]
match = { tool = "Edit" }
command = ["./.claude/hooks/check_mojibake.sh"]
```

This directory is reserved for the case where a Codex-shaped wrapper is
required — e.g. a hook whose CLI contract differs enough from Claude's
PreToolUse / PostToolUse JSON-stdin convention that sharing the script is
not viable. When that day comes, drop the wrapper here and reference it
from `.codex/config.toml`.

Until then, the directory exists only to make the convention visible
alongside the populated `.claude/hooks/` tree.

Do not add `.codex/hooks.json` next to the existing inline hook entries in
`.codex/config.toml`. This repository keeps one project-layer hook
representation to avoid Codex startup warnings about loading hooks from both
files.

See also:
- `.codex/config.toml` — `[features] hooks = true` and the
  `[[hooks.*]]` blocks.
- `.codex/README.md` — operator-facing hooks guidance.
- `.claude/hooks/` — the shared script bodies.
