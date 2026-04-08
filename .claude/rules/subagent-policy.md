# Subagent Delegation Policy

Single source of truth for when to delegate work to subagents.

## Post-implementation pipeline — always delegate

After implementation completes (`/work` or `/loop`), run the post-implementation pipeline via subagents:

| Step | Subagent | Skill | Purpose |
|------|----------|-------|---------|
| 1 | `reviewer` | `/self-review` | Diff quality |
| 2 | `verifier` | `/verify` | Spec compliance + static analysis |
| 3 | `tester` | `/test` | Behavioral tests |

Each step runs sequentially (output of one informs the next). Use the Task tool with `subagent_type` matching the agent name.

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

## Planning — always inline

`/plan` runs in the main context because it relies heavily on `AskUserQuestion` for user interaction (task type selection, objective confirmation, flow selection, Codex advisory response). Subagent execution would add indirection without benefit. No agent definition exists for this skill.

## Codex triage — always inline

`/codex-review` triage runs in the main context (not delegated to a subagent) because triage accuracy depends on implementation context — knowing *why* the code was written that way, what design decisions were made, what the plan's non-goals are, and what the self-review already addressed. A subagent would lack this context and produce unreliable classifications (more false negatives in DISMISSED, more false positives in ACTION_REQUIRED).

The triage step reads existing artifacts (plan, self-review report, verify report) and produces `docs/reports/codex-triage-<slug>.md`. No new subagent definition is needed.

## Documentation sync — always delegate

After implementation and before PR creation, run `/sync-docs` via the `doc-maintainer` subagent:

```
Task(subagent_type="doc-maintainer", prompt="Run /sync-docs after <slug> implementation")
  → doc-maintainer updates docs, rules, and reports as needed
```

This runs after the test step and before `/pr`, producing documentation updates as a separate concern from implementation.

## Rationale

- Post-implementation steps produce independent artifacts with clear boundaries — ideal for subagent isolation.
- Subagent execution preserves main context tokens for implementation work.
- Sequential execution ensures each step can react to prior findings.
