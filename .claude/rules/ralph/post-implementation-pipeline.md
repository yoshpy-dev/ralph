# Post-implementation pipeline order

Single source of truth for the post-implementation pipeline (standard flow, `/work`).

## Canonical order

```
/self-review → /verify → /test → /sync-docs → /cross-review → /pr
```

No step may be skipped. If any step triggers a fix-and-revalidate cycle (e.g., cross-review ACTION_REQUIRED), the **full pipeline** re-runs from `/self-review` onwards.

### CLI execution mode

Both Claude Code and Codex run the same canonical order, but the execution
model differs:

| Step | Claude Code (`/work`) | Codex |
|------|------------------------|-------|
| `/self-review` | `Task(subagent_type="reviewer")` | `.codex/agents/reviewer.toml` custom agent |
| `/verify` | `Task(subagent_type="verifier")` | `.codex/agents/verifier.toml` custom agent |
| `/test` | `Task(subagent_type="tester")` | `.codex/agents/tester.toml` custom agent |
| `/sync-docs` | `Task(subagent_type="doc-maintainer")` | `.codex/agents/doc-maintainer.toml` custom agent |
| `/cross-review` | inline; calls `codex exec review` | inline; calls `claude -p` reviewer prompt |
| `/pr` | inline | inline |

Reports go to `docs/reports/` (CLI-neutral path) so the pipeline cycle counter
and PR pre-checks behave identically. The driver detection used by
`/cross-review` is documented in `.claude/skills/cross-review/SKILL.md`.

## Step responsibilities

| Step | Agent | Purpose | Stop condition |
|------|-------|---------|----------------|
| `/self-review` | `reviewer` | Diff quality only; no tests/static/spec/doc-drift/broad audit | CRITICAL findings |
| `/verify` | `verifier` | Spec compliance + static analysis via `./scripts/run-static-verify.sh`; changed-language scope by default; no tests | Fail verdict |
| `/test` | `tester` | Behavioral tests via `./scripts/run-test.sh`; changed-language scope by default; no static analysis | Fail verdict |
| `/sync-docs` | `doc-maintainer` | Documentation sync | — |
| `/cross-review` | inline | Cross-model second opinion using pinned plan/worktree state | ACTION_REQUIRED triggers re-run |
| `/pr` | inline | PR creation + plan archival + task worktree/local branch cleanup | — |

## Re-run after cross-review ACTION_REQUIRED fix

When fixing cross-review findings, the re-run includes **all** steps:

```
fix → /self-review → /verify → /test → /sync-docs → /cross-review
```

Not just `/self-review → /verify → /test → /cross-review`. The `/sync-docs` step must be included because fixes may change behavior that requires documentation updates.

### Pipeline cycle cap (default 2 total runs)

The post-implementation pipeline is capped at **2 total runs by default**: the initial run plus at most one fix-and-revalidate re-run. After the second run, the pipeline does not automatically regress even if cross-review still reports ACTION_REQUIRED.

Controlled by `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default `2`). The counter is persisted to `.harness/state/standard-pipeline/cycle-count.json`, keyed by the pinned plan path and task worktree state in `.harness/state/standard-pipeline/active-plan.json`. When the cap is reached, `/cross-review` drops the "fix" option from Case A/B and offers: (1) raise the cap and re-run, (2) proceed to `/pr` and record remaining findings as known gaps, (3) abort.

Raise the cap only when you consciously accept additional churn; the default is a deliberate "fail fast, hand back to the operator" stance.

See `.claude/rules/ralph/subagent-policy.md` for execution model details.

## Where this order is referenced

If you update this order, update all of these locations:
- `.claude/skills/work/SKILL.md` (Step 13)
- `.claude/skills/cross-review/SKILL.md` (Case A and Case B re-run)
- `.claude/rules/ralph/subagent-policy.md` (Post-implementation pipeline table)
- `docs/quality/definition-of-done.md` (Pipeline order)
- `README.md` (Quick start and Operating loop sections)
- `AGENTS.md` (Primary loop section)
