# Post-implementation pipeline order

Single source of truth for the post-implementation pipeline. All flows (standard /work, Ralph Loop) must follow this order.

## Canonical order

```
/self-review → /verify → /test → /sync-docs → /codex-review → /pr
```

No step may be skipped. If any step triggers a fix-and-revalidate cycle (e.g., Codex ACTION_REQUIRED), the **full pipeline** re-runs from `/self-review` onwards.

**Pipeline parity:** In Ralph Loop (`ralph-pipeline.sh`), each post-implementation step runs as a dedicated `claude -p` agent with a single-responsibility prompt (not shell-direct execution). This ensures the same depth of analysis as standard-flow subagents: structured reports with findings tables, root cause analysis, spec compliance checks, and documentation drift detection. Reports are dual-written to both `.harness/state/pipeline/` and `docs/reports/`.

## Step responsibilities

| Step | Agent | Purpose | Stop condition |
|------|-------|---------|----------------|
| `/self-review` | `reviewer` | Diff quality | CRITICAL findings |
| `/verify` | `verifier` | Spec compliance + static analysis | Fail verdict |
| `/test` | `tester` | Behavioral tests | Fail verdict |
| `/sync-docs` | `doc-maintainer` | Documentation sync | — |
| `/codex-review` | inline | Cross-model second opinion | ACTION_REQUIRED triggers re-run |
| `/pr` | inline | PR creation + plan archival | — |

## Re-run after Codex ACTION_REQUIRED fix

When fixing Codex findings, the re-run includes **all** steps:

```
fix → /self-review → /verify → /test → /sync-docs → /codex-review
```

Not just `/self-review → /verify → /test → /codex-review`. The `/sync-docs` step must be included because fixes may change behavior that requires documentation updates.

## Integration pipeline (Ralph Loop only)

After all slices are merged into the integration branch, `ralph-orchestrator.sh` runs `ralph-pipeline.sh --skip-pr --fix-all` as a unified quality gate. This follows the same canonical order above but with stricter thresholds:

- `--skip-pr`: PR creation is handled by the orchestrator, not the pipeline
- `--fix-all`: ALL self-review findings (CRITICAL+HIGH+MEDIUM+LOW > 0) override COMPLETE; WORTH_CONSIDERING codex findings trigger Inner Loop regression (same as ACTION_REQUIRED)

**Intentional deviation in Ralph Loop:** Per-slice pipelines (`ralph-pipeline.sh`) do NOT stop on CRITICAL self-review findings — they log them and let verify/test catch real issues. This differs from the standard `/work` flow where CRITICAL findings block the pipeline. The rationale is that autonomous pipelines benefit from letting downstream gates (verify, test) confirm whether the finding is a true positive before halting. This deviation is tracked in `docs/tech-debt/README.md`.

See `.claude/rules/subagent-policy.md` for execution model details.

## Where this order is referenced

If you update this order, update all of these locations:
- `.claude/skills/work/SKILL.md` (Step 9)
- `.claude/skills/loop/SKILL.md` (After the loop section)
- `.claude/skills/codex-review/SKILL.md` (Case A and Case B re-run)
- `.claude/rules/subagent-policy.md` (Post-implementation pipeline table)
- `CLAUDE.md` (Default behavior)
- `docs/quality/definition-of-done.md` (Pipeline order)
- `README.md` (Quick start and Operating loop sections)
- `AGENTS.md` (Primary loop section)
