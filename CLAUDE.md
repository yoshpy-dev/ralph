@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- Manual-trigger skills (`disable-model-invocation: true`): `/release` only (cut a Homebrew release tag for the `ralph` CLI — repo-maintainer only, not included in `ralph init`). All others (spec, plan, work, self-review, verify, test, cross-review, pr, sync-docs, audit-harness, org) are auto-invoked. `anti-bottleneck` is a model-internal support skill (`user-invocable: false`) and belongs to neither list.
- Use `/spec` when the request is too vague for `/plan`. `/spec` refines abstract ideas through decision-tree questioning with recommended answers, codebase exploration, web research, and interactive clarification. Issue-only specs use a temporary clean-base worktree and cleanup; saved specs either create a docs/spec PR or hand off to `/plan` in the same task worktree.
- Use `/plan` before risky, ambiguous, or multi-file work. `/plan` ensures a clean-base task worktree before writing plan artifacts.
- `/plan` resolves critical forks (two+ approaches with materially different risk/cost that cannot be resolved from repo context) with targeted AskUserQuestion follow-ups before finalizing.
- `/work` resumes the task worktree and starts interactive implementation. Post-impl pipeline runs via subagents.
- Autonomous multi-seat execution outside this interactive flow is the org runtime's job (`ralph org spawn/send/wait/...`); see `docs/specs/2026-08-01-org-runtime.md` and `.claude/rules/agent-messaging.md`.
- After /work, the post-implementation pipeline runs via subagents (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/cross-review` (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If the `codex` binary is available, `/plan` and `/cross-review` invoke Codex for second-opinion feedback. If unavailable, the step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, cleans up the task worktree/local branch, and completes the hand-off. A task is "done" when the PR is created and cleanup has either succeeded or reported recoverable state.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- In `/work`, the post-implementation pipeline (`/self-review` → `/verify` → `/test` → `/sync-docs`) runs via subagents (`reviewer`, `verifier`, `tester`, `doc-maintainer`). See `.claude/rules/subagent-policy.md` for details.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules (also read by Codex)
- `.claude/skills/` for on-demand workflows (mirrored to `.agents/skills/` for Codex; regenerate the mirror with `scripts/sync-skills.sh`; drift-checked by `scripts/check-skill-sync.sh`)
- `.claude/agents/` for Claude Code subagent definitions (Codex custom agents live under `.codex/agents/`)
- `.claude/hooks/` for deterministic runtime controls (Codex equivalents live under `.codex/config.toml` `[hooks]` for the meta-repo and `templates/base/.codex/config.toml` for scaffolded projects; the two are kept byte-identical)
