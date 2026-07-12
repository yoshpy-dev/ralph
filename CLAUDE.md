@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- Manual-trigger skills (`disable-model-invocation: true`): `/spec` (refine vague ideas) and `/release` (cut a Homebrew release tag for the `ralph` CLI — repo-maintainer only, not included in `ralph init`). All others (plan, work, loop, self-review, verify, test, cross-review, pr, sync-docs, audit-harness) are auto-invoked. `anti-bottleneck` is a model-internal support skill (`user-invocable: false`) and belongs to neither list.
- Use `/spec` when the request is too vague for `/plan`. `/spec` refines abstract ideas through decision-tree questioning with recommended answers, codebase exploration, web research, and interactive clarification. Issue-only specs use a temporary clean-base worktree and cleanup; saved specs either create a docs/spec PR or hand off to `/plan` in the same task worktree.
- Use `/plan` before risky, ambiguous, or multi-file work. `/plan` ensures a clean-base task worktree before writing plan artifacts.
- `/plan` asks at minimum one decision — 標準フロー (/work) or Ralph Loop (/loop) — and, when critical forks are detected during drafting (two+ approaches with materially different risk/cost that cannot be resolved from repo context), asks targeted AskUserQuestion follow-ups before finalizing.
- `/work` resumes the task worktree and starts interactive implementation. Post-impl pipeline runs via subagents.
- `/loop` uses a directory-based plan and runs `ralph-orchestrator.sh` for autonomous parallel-slice execution: multi-worktree (`git worktree add` × N) → `ralph-pipeline.sh` per slice → integration branch → sequential merge → integration pipeline (`--skip-pr --fix-all`) → grouped PRs by default, with unified or stacked PRs only when explicitly selected.
- In Ralph Loop, the scripts handle the full lifecycle autonomously — no manual subagent chain needed. Use `./scripts/ralph run` or `./scripts/ralph status` to operate.
- After /work, the post-implementation pipeline runs via subagents (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/cross-review` (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If the `codex` binary is available, `/plan` and `/cross-review` invoke Codex for second-opinion feedback. If unavailable, the step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, cleans up the task worktree/local branch, and completes the hand-off. A task is "done" when the PR is created and cleanup has either succeeded or reported recoverable state.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- In `/work`, the post-implementation pipeline (`/self-review` → `/verify` → `/test` → `/sync-docs`) runs via subagents (`reviewer`, `verifier`, `tester`, `doc-maintainer`). In Ralph Loop, the same pipeline runs internally via dedicated `claude -p` prompts per slice. See `.claude/rules/subagent-policy.md` for details.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules (also read by Codex)
- `.claude/skills/` for on-demand workflows (mirrored to `.agents/skills/` for Codex; drift-checked by `scripts/check-skill-sync.sh`)
- `.claude/agents/` for Claude Code subagent definitions (Codex custom agents live under `.codex/agents/`)
- `.claude/hooks/` for deterministic runtime controls (Codex equivalents live under `.codex/config.toml` `[hooks]` for the meta-repo and `templates/base/.codex/config.toml` for scaffolded projects; the two are kept byte-identical)
