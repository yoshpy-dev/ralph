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

## Ralph Loop per-phase routing

Per-phase model variables for `ralph-pipeline.sh`. Resolved by `resolve_phase_model <phase> <cycle>` in `scripts/ralph-cli-driver.sh`.

| Phase | Variable | Default | Notes |
|-------|----------|---------|-------|
| `implement` | `RALPH_IMPLEMENT_MODEL` | `sonnet` | Inner Loop implement/fix seat |
| `self_review` | `RALPH_SELF_REVIEW_MODEL` | `opus` | Judgment seat — always quality tier |
| `verify` | `RALPH_VERIFY_MODEL` | `sonnet` | Interprets deterministic script output |
| `test` | `RALPH_TEST_MODEL` | `sonnet` | Interprets deterministic script output |
| `sync_docs` | `RALPH_SYNC_DOCS_MODEL` | `sonnet` | Doc maintenance |
| `pr` | `RALPH_PR_MODEL` | `sonnet` | PR-creation agent turn |
| `probe` | `RALPH_PROBE_MODEL` | `haiku` | CLI capability probes — cheap; no intelligence needed |
| `escalation` | `RALPH_ESCALATION_MODEL` | `opus` | Replaces implement seat on outer cycle ≥ 2 |

**Precedence:** `RALPH_FORCE_MODEL` > `RALPH_<PHASE>_MODEL` (env) > `[pipeline.phases]` (ralph.toml) > built-in default. `RALPH_MODEL` remains the global fallback for unrouted turns.

**Escalation:** When the Outer Loop enters fix-and-revalidate (outer cycle ≥ 2), the implement seat runs on `RALPH_ESCALATION_MODEL` instead of `RALPH_IMPLEMENT_MODEL`. Deterministic trigger — the existing verify/test/cross-review gates act as quality floor; no LLM router is involved.

**Single-knob rollback:** `RALPH_FORCE_MODEL=opus` overrides every phase at once, restoring pre-routing behavior. Finer-grained: `RALPH_IMPLEMENT_MODEL=opus` for the implement seat only.

**Receipts:** Each routed `run_agent` call appends one JSON line to `.harness/state/pipeline/model-receipts.jsonl` with fields `ts / phase / cycle / driver / requested_model / effective_model / honored / effort / reason`. The Codex driver ignores per-phase model args — its receipts record `effective_model="codex-default"` and `honored=false` (known gap, non-fixable without Codex API support). Receipt writes also occur at cross-review call sites and in `DRY_RUN=1` mode.

## Where the values live

- `.claude/agents/*.md` — pipeline subagent tiers (frontmatter `model:`)
- `scripts/ralph-config.sh` — effective Ralph defaults (`RALPH_MODEL`,
  `RALPH_EFFORT`, `RALPH_CLAUDE_REVIEWER_MODEL`; plus all 8 per-phase vars above); shell wrappers do not read
  `ralph.toml`, so keep both in sync when changing defaults
- `ralph.toml` — declarative mirror of the same values for the ralph CLI;
  `[pipeline.phases]` section holds per-phase keys
