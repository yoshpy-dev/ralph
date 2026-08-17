# Model routing

How to assign Claude models when delegating work to subagents or background
tasks. Delegation timing/order lives in `subagent-policy.md`.

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

## Standard flow delegation (/work)

Implementation slices in `/work` are delegated to the `implementer` subagent
(`model: sonnet` pinned in frontmatter; Codex: `.codex/agents/implementer.toml`).
The orchestrator (session model) retains planning, decomposition, handoff
authoring, report adjudication, and final review — it does not write slice code
itself.

**Structured handoff must carry:**

| Field | Required content |
|-------|-----------------|
| Plan path | worktree-relative path to the active plan |
| Slice objective | single-sentence goal |
| Acceptance criteria | the ACs this slice addresses |
| Files in scope | exact list; implementer stages only these paths |
| Exact verification commands | copy-paste ready |
| Commit message format | conventional format string |

**Report contract (implementer → orchestrator):** changed files, decisions/deviations,
verification evidence, commit-boundary evidence (`git status --porcelain` +
`git show --stat HEAD` output), commit SHA.

**Inline exceptions** (no subagent dispatch needed):

- Trivial single-file edits where the handoff overhead exceeds the change cost.
- Dispatch failure → inline fallback, noted in the report (same convention as
  the post-implementation pipeline fallback).

**Escalating a judgment-heavy slice:** pass an explicit `model` on the Task
call (e.g. `opus` for security-sensitive changes) — no new env knob.

**Cross-review sync note:** `.claude/skills/cross-review/SKILL.md` reads
`RALPH_CLAUDE_REVIEWER_MODEL` (with an `opus` fallback) for the claude reviewer
path. Keep reviewer-model defaults in sync when changing `RALPH_CLAUDE_REVIEWER_MODEL`
in `scripts/ralph-config.sh`.

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

## Org runtime model receipts

`ralph org spawn` seats are commanded with an explicit model per seat, not
via the tier table above. Each spawn appends a JSON line to
`.harness/state/org/model-receipts.jsonl` with `ts / org_id / seat_id / role /
driver / commanded_model / reported_effective_model / honored / reason`. This
is a separate mechanism from the `/work` subagent tiers documented above —
see the org runtime spec shipped with your project for its own model
selection rules.

## Where the values live

- `.claude/agents/*.md` — pipeline subagent tiers (frontmatter `model:`)
- `scripts/ralph-config.sh` — effective Ralph defaults (`RALPH_CLAUDE_REVIEWER_MODEL`,
  `RALPH_STANDARD_MAX_PIPELINE_CYCLES`)
