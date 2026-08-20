# Slice 1 live-fire evidence: codex-hooks-json-wiring (2026-08-20, codex-cli 0.147.0)

## Experiment: git-root-resolving command form + apply_patch-inclusive matcher

- Host: trusted checkout /Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph (main)
- hooks.json PostToolUse group under test:
  - matcher: "Edit|Write|MultiEdit|apply_patch"
  - command: "$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh" PostToolUse
- Invocation: codex exec --skip-git-repo-check --dangerously-bypass-hook-trust '<file edit prompt>'
- Result: probe drop-in at .claude/hooks/local/PostToolUse.d/ FIRED (payload captured);
  "hook: PostToolUse Completed" x13 on stderr; edited file content confirmed.
- Payload: tool_name=apply_patch, hook_event_name=PostToolUse, cwd=<project root>.
- Conclusion 1: the hooks.json `command` string IS shell-evaluated — `$(git rev-parse
  --show-toplevel)` resolves, satisfying the official doc's recommendation to resolve
  from the git root instead of a bare relative path (stable across subdirectory cwd).
- Conclusion 2: the alternation matcher including the literal tool name `apply_patch`
  fires on file edits. Official doc (learn.chatgpt.com/docs/hooks): matcher is a regex
  over the tool name; `apply_patch`, `Edit`, `Write` are all accepted for file edits
  while the payload reports tool_name=apply_patch.
- Note: one earlier identical run did not perform the file edit at all (model variance);
  the non-fire in that run was caused by no PostToolUse event, not by the command form.

## Primary-source facts (learn.chatgpt.com/docs/hooks, fetched 2026-08-20)

- Schema: top-level `hooks` -> event-name keys -> matcher-group arrays ->
  handlers {type:"command", command:<string>, timeout?, async?, ...}.
- Matcher: regex over tool name; omit / "" / "*" matches all.
- Hook cwd is the session cwd; docs recommend git-root resolution over relative paths.
- [features].hooks default when absent: NOT specified by the doc (doctor stays lenient
  on absent, warns only on explicit false).
- Trust: per-hook-hash review via interactive /hooks; changed hooks are skipped until
  re-trusted. TOML [[hooks]] and hooks.json are documented as equivalent and merged
  with a warning when both exist — however on codex-cli 0.147.0 project inline TOML
  hooks empirically never execute (2026-08-19 evidence); shipping hooks.json aligns
  with both the docs and observed behavior.
