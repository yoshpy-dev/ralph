# Model routing

How to assign Claude models when delegating work to subagents, background
tasks, or Ralph Loop. Delegation timing/order lives in `subagent-policy.md`.

## Tier table

| Seat | Model | Examples |
|------|-------|----------|
| Orchestrator (main session) | session model (user's choice) | planning, decomposition, arbitration, final review |
| Judgment seats | `opus` | security-sensitive review (`reviewer`), design trade-offs, ambiguous root-cause debugging |
| Procedural seats | `sonnet` | scoped implementation, spec/static verification (`verifier`), test execution (`tester`), doc sync (`doc-maintainer`) |
| Bulk mechanical work | `haiku` | grep/file inventory, log scanning, large read-only sweeps |

Quality is preserved by plan artifacts, not by model tier: when delegating,
the prompt must carry acceptance criteria, exact verification commands, and a
report contract (changed files, key decisions, verification evidence). A
cheaper model with a precise plan beats an expensive model with a vague one.

## Rules

- **Always pin `model:` in agent frontmatter.** Omitted `model:` means
  `inherit` — a subagent spawned from an expensive main session silently runs
  on that expensive model, multiplied by parallel fan-out.
- **Use stable aliases (`opus`, `sonnet`, `haiku`), not full model IDs**, in
  agent frontmatter, `ralph.toml`, `scripts/ralph-config.sh`, and skill docs.
  Full IDs go stale and can break `claude -p` at runtime. Pin a full ID only
  via environment variable when a specific run must be reproducible.
- **Do not export `CLAUDE_CODE_SUBAGENT_MODEL`.** It silently overrides every
  frontmatter `model:` and per-call `model` parameter. Treat it as an
  emergency-only blunt instrument.
- **Ad-hoc Task/Agent calls should pass `model` explicitly** when the work is
  bulk search or mechanical (e.g. Explore fan-outs → `haiku`); otherwise they
  inherit the session model.
- **Scheduled, cron, and background executions must state their model
  explicitly.** They otherwise run on whatever model is currently saved as
  the session default.
- **Switch models at task boundaries, not mid-conversation.** A mid-session
  model or effort switch invalidates the prompt cache and re-reads the full
  history uncached on the next response.
- Keep `effort` at the default (`high`) unless measured evidence justifies a
  change; `xhigh`/`max` have documented diminishing returns.

## Where the values live

- `.claude/agents/*.md` — pipeline subagent tiers (frontmatter `model:`)
- `scripts/ralph-config.sh` — effective Ralph defaults (`RALPH_MODEL`,
  `RALPH_EFFORT`, `RALPH_CLAUDE_REVIEWER_MODEL`); shell wrappers do not read
  `ralph.toml`, so keep both in sync when changing defaults
- `ralph.toml` — declarative mirror of the same values for the ralph CLI
  (present in project instances generated from `templates/base/`)
