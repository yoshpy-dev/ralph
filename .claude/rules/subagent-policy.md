# Subagent Delegation Policy

When and how to delegate work to subagents. Pipeline order is defined in `post-implementation-pipeline.md`.

## Post-implementation pipeline for /work — delegate via subagents

After `/work` completes, run the post-implementation pipeline via subagents:

| Step | Subagent | Skill | Purpose |
|------|----------|-------|---------|
| 1 | `reviewer` | `/self-review` | Diff quality |
| 2 | `verifier` | `/verify` | Spec compliance + static analysis |
| 3 | `tester` | `/test` | Behavioral tests |
| 4 | `doc-maintainer` | `/sync-docs` | Documentation sync |

Steps 1–3 run sequentially (output of one informs the next). Step 4 runs after tests pass. After step 4, `/cross-review` runs inline (optional), then `/pr`. Use the Task tool with `subagent_type` matching the agent name.

### Execution

```
Task(subagent_type="reviewer", prompt="Run /self-review for the current diff against plan <slug>")
  → reviewer produces docs/reports/self-review-*.md
  → if CRITICAL findings: stop and fix before continuing

Task(subagent_type="verifier", prompt="Run /verify against plan <slug>")
  → verifier produces docs/reports/verify-*.md
  → if fail verdict: stop and fix before continuing

Task(subagent_type="tester", prompt="Run /test against plan <slug>")
  → tester produces docs/reports/test-*.md
  → if fail verdict: do NOT proceed to /pr
```

### Fallback

If a subagent fails to execute (tool error, not a review finding), run the corresponding skill inline and note the fallback in the report.

## Spec — always inline

`/spec` runs in the main context because it relies heavily on `AskUserQuestion` for requirement clarification (active back-and-forth with the user) and on `AskUserQuestion` for output selection (save file / create issue / both / transition to `/plan`). Subagent execution would cut off the interactive clarification loop. No agent definition exists for this skill.

## Planning — always inline

`/plan` runs in the main context because it relies heavily on `AskUserQuestion` for user interaction (task type selection, objective confirmation, flow selection, critical-fork resolution during drafting, Codex advisory response). Subagent execution would add indirection without benefit. No agent definition exists for this skill.

## Codex triage — always inline

`/cross-review` triage runs in the main context (not delegated to a subagent) because triage accuracy depends on implementation context — knowing *why* the code was written that way, what design decisions were made, what the plan's non-goals are, and what the self-review already addressed. A subagent would lack this context and produce unreliable classifications (more false negatives in DISMISSED, more false positives in ACTION_REQUIRED).

The triage step reads existing artifacts (plan, self-review report, verify report) and produces `docs/reports/cross-review-triage-<slug>.md`. No new subagent definition is needed.

## Post-implementation pipeline under Codex — sequential inline

Codex CLI does not have a subagent (`Task`) mechanism. When ralph runs under
Codex (`RALPH_PRIMARY_CLI=codex` or detected at runtime), the four
post-implementation skills run **sequentially inline in the single agent**:
the agent walks the canonical order step-by-step, writing the same reports to
`docs/reports/` that the Claude subagent path produces. This keeps artifact
parity for `/cross-review` triage, the cycle cap, and `/pr`.

Cap protection: the same `RALPH_STANDARD_MAX_PIPELINE_CYCLES` ceiling applies,
so a runaway inline pipeline cannot loop more than `cap` total runs.

## Post-implementation pipeline for /loop — orchestrator-internal

Ralph Loop uses `ralph-pipeline.sh` per slice (not subagents). Same pipeline order as `/work` (see `post-implementation-pipeline.md`), but executed via `claude -p` calls with dedicated prompts.

After all slices are merged into the integration branch, `ralph-orchestrator.sh` runs `ralph-pipeline.sh --skip-pr --fix-all` on the integration branch as a unified quality gate. This catches cross-module issues and fixes ALL findings (including MEDIUM/LOW and WORTH_CONSIDERING) before unified PR creation.

Execution model difference:
- `/work`: subagent Task calls in Claude Code session
- `/loop`: `claude -p` invocations orchestrated by `ralph-pipeline.sh`

When a user returns after a Ralph Loop run, check `./scripts/ralph status` for the final outcome rather than running the subagent chain.
