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

## AC-2(b) fresh-scaffold fixture evidence (2026-08-20, post-Slice-2)

- Fixture: `ralph init --yes <tmpdir>` using the Slice-2 build (ships .codex/hooks.json,
  manifest-tracked). hooks.json validated as JSON; manifest carries the entry.
- Invocation: `codex exec --skip-git-repo-check --dangerously-bypass-hook-trust
  '<file edit prompt>'` run inside the fresh, UNTRUSTED fixture.
- Result: the SHIPPED hooks.json fired — a probe drop-in at the fixture's
  .claude/hooks/local/PostToolUse.d/ was executed (marker file created), i.e.
  project-layer hooks.json is discovered and loaded even in an untrusted project,
  and the dispatcher fan-out reaches the third layer. "hook: PostToolUse" x16 on stderr.
- Contrast with the P5-era scratch-repo non-fire: that attempt used the inline TOML
  representation; with hooks.json the fresh-scaffold path works under bypass.
- Remaining constraint (unchanged): WITHOUT --dangerously-bypass-hook-trust, untrusted
  hook commands are silently skipped by codex exec; persisting trust requires one
  interactive session approval (per-command-hash, ~/.codex/config.toml hooks.state).
  Downstream users therefore run `codex` interactively once after `ralph init` and
  approve the hook — documented in .codex/README.md (Slice 3).
- AC-2(a) note: the earlier non-bypass fire on this machine (2026-08-20) used the
  then-trusted ABSOLUTE-path command string; the final shipped string (git-root form)
  has a different hash and would need re-approval, so per the plan the shipped-form
  proof uses bypass, with this reason recorded.
