# Post-implementation pipeline order

Single source of truth for the post-implementation pipeline. All flows (standard /work, Ralph Loop) must follow this order.

## Canonical order

```
/self-review → /verify → /test → /sync-docs → /cross-review → /pr
```

No step may be skipped. If any step triggers a fix-and-revalidate cycle (e.g., Codex ACTION_REQUIRED), the **full pipeline** re-runs from `/self-review` onwards.

**Pipeline parity:** In Ralph Loop (`ralph-pipeline.sh`), each post-implementation step runs as a dedicated `claude -p` agent with a single-responsibility prompt (not shell-direct execution). This ensures the same depth of analysis as standard-flow subagents: structured reports with findings tables, root cause analysis, spec compliance checks, and documentation drift detection. Reports are dual-written to both `.harness/state/pipeline/` and `docs/reports/`.

### CLI execution mode

Both Claude Code and Codex run the same canonical order, but the execution
model differs:

| Step | Claude Code (`/work`) | Codex |
|------|------------------------|-------|
| `/self-review` | `Task(subagent_type="reviewer")` parallel-capable | sequential inline in the single agent |
| `/verify` | `Task(subagent_type="verifier")` | sequential inline |
| `/test` | `Task(subagent_type="tester")` | sequential inline |
| `/sync-docs` | `Task(subagent_type="doc-maintainer")` | sequential inline |
| `/cross-review` | inline; calls `codex exec review` | inline; calls `claude -p` reviewer prompt |
| `/pr` | inline | inline |

Reports go to `docs/reports/` (CLI-neutral path) so the pipeline cycle counter
and PR pre-checks behave identically. The driver detection used by
`/cross-review` is documented in `.claude/skills/cross-review/SKILL.md`.

## Step responsibilities

| Step | Agent | Purpose | Stop condition |
|------|-------|---------|----------------|
| `/self-review` | `reviewer` | Diff quality | CRITICAL findings |
| `/verify` | `verifier` | Spec compliance + static analysis | Fail verdict |
| `/test` | `tester` | Behavioral tests | Fail verdict |
| `/sync-docs` | `doc-maintainer` | Documentation sync | — |
| `/cross-review` | inline | Cross-model second opinion | ACTION_REQUIRED triggers re-run |
| `/pr` | inline | PR creation + plan archival | — |

## Re-run after Codex ACTION_REQUIRED fix

When fixing Codex findings, the re-run includes **all** steps:

```
fix → /self-review → /verify → /test → /sync-docs → /cross-review
```

Not just `/self-review → /verify → /test → /cross-review`. The `/sync-docs` step must be included because fixes may change behavior that requires documentation updates.

### Pipeline cycle cap (default 2 total runs)

The post-implementation pipeline is capped at **2 total runs by default**: the initial run plus at most one fix-and-revalidate re-run. After the second run, the pipeline does not automatically regress even if Codex still reports ACTION_REQUIRED.

- **Standard flow (`/work`)**: controlled by `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default `2`). The counter is persisted to `.harness/state/standard-pipeline/cycle-count.json`, keyed by the pinned plan path in `.harness/state/standard-pipeline/active-plan.json`. When the cap is reached, `/cross-review` drops the "fix" option from Case A/B and offers: (1) raise the cap and re-run, (2) proceed to `/pr` and record remaining findings as known gaps, (3) abort.
- **Ralph Loop (`/loop`)**: controlled by `RALPH_MAX_OUTER_CYCLES` (default `2`). When exceeded, `ralph-pipeline.sh` calls `_finalize "max_outer_cycles"` and stops autonomously.

Both variables accept environment-variable overrides. Raise them only when you consciously accept additional churn; the default is a deliberate "fail fast, hand back to the operator" stance.

## Integration pipeline (Ralph Loop only)

After all slices are merged into the integration branch, `ralph-orchestrator.sh` runs `ralph-pipeline.sh --skip-pr --fix-all` as a unified quality gate. This follows the same canonical order above but with stricter thresholds:

- `--skip-pr`: PR creation is handled by the orchestrator, not the pipeline
- `--fix-all`: ALL self-review findings (CRITICAL+HIGH+MEDIUM+LOW > 0) override COMPLETE; WORTH_CONSIDERING codex findings trigger Inner Loop regression (same as ACTION_REQUIRED)

**Intentional deviation in Ralph Loop:** Per-slice pipelines (`ralph-pipeline.sh`) do NOT stop on CRITICAL self-review findings — they log them and let verify/test catch real issues. This differs from the standard `/work` flow where CRITICAL findings block the pipeline. The rationale is that autonomous pipelines benefit from letting downstream gates (verify, test) confirm whether the finding is a true positive before halting. This deviation is tracked in `docs/tech-debt/README.md`.

See `.claude/rules/subagent-policy.md` for execution model details.

## Where this order is referenced

If you update this order, update all of these locations:
- `.claude/skills/work/SKILL.md` (Step 13)
- `.claude/skills/loop/SKILL.md` (After the loop section)
- `.claude/skills/cross-review/SKILL.md` (Case A and Case B re-run)
- `.claude/rules/subagent-policy.md` (Post-implementation pipeline table)
- `CLAUDE.md` (Default behavior)
- `docs/quality/definition-of-done.md` (Pipeline order)
- `README.md` (Quick start and Operating loop sections)
- `AGENTS.md` (Primary loop section)
