# Subagent Delegation Policy

When and how to delegate work to subagents. Pipeline order is defined in `post-implementation-pipeline.md`. Model tier assignment per seat is defined in `model-routing.md`.

## Post-implementation pipeline for /work — phase roles

After `/work` completes, run the post-implementation pipeline through the
phase-specific subagents below:

| Step | Subagent | Skill | Purpose |
|------|----------|-------|---------|
| 1 | `reviewer` | `/self-review` | Diff quality |
| 2 | `verifier` | `/verify` | Spec compliance + static analysis |
| 3 | `tester` | `/test` | Behavioral tests |
| 4 | `doc-maintainer` | `/sync-docs` | Documentation sync |

Steps 1–3 run sequentially (output of one informs the next). Step 4 runs after
tests pass. After step 4, `/cross-review` runs inline (optional), then `/pr`.
Claude Code uses the Task tool with `subagent_type` matching the agent name.
Codex uses the matching `.codex/agents/` custom agents. Do not fan out steps
1–3 in parallel; verifier output may rely on self-review context, and tester
output may rely on verifier scope.

### Execution

```
reviewer: run /self-review for the current diff against plan <slug>
  → reviewer produces docs/reports/self-review-*.md
  → if CRITICAL findings: stop and fix before continuing

verifier: run /verify against plan <slug>
  → verifier produces docs/reports/verify-*.md
  → if fail verdict: stop and fix before continuing

tester: run /test against plan <slug>
  → tester produces docs/reports/test-*.md
  → if fail verdict: do NOT proceed to /pr
```

### Fallback

If a subagent fails to execute (tool error, not a review finding), run the corresponding skill inline and note the fallback in the report.

## Implementation slices (/work) — delegate to implementer

During `/work` step 6 (implementation), each implementation slice is dispatched
to the `implementer` subagent:

- **Claude Code:** `Task(subagent_type="implementer")`
- **Codex:** `.codex/agents/implementer.toml` custom agent

The orchestrator authors the structured handoff (defined in
`model-routing.md` — "Standard flow delegation (/work)") and does not write
slice code itself. The handoff must carry: plan path, slice objective,
acceptance criteria, files in scope, exact verification commands, and commit
message format. The implementer returns: changed files, decisions/deviations,
verification evidence, commit-boundary evidence, and commit SHA.

**Inline exceptions** (dispatch not required):

- Trivial single-file edits where the handoff cost exceeds the change cost.
- Dispatch failure → inline fallback, noted in the report.

The post-implementation pipeline (`reviewer` → `verifier` → `tester` →
`doc-maintainer`) is unchanged and remains the quality floor regardless of
whether slices were delegated or implemented inline.

## Spec — always inline

`/spec` runs in the main context because it relies heavily on `AskUserQuestion` for requirement clarification (active back-and-forth with the user) and on `AskUserQuestion` for output selection (issue-only / save spec file as docs PR / save spec file and transition to `/plan`). Subagent execution would cut off the interactive clarification loop. No agent definition exists for this skill.

## Planning — always inline

`/plan` runs in the main context because it relies heavily on `AskUserQuestion` for user interaction (task type selection, objective confirmation, critical-fork resolution during drafting, Codex advisory response). Subagent execution would add indirection without benefit. No agent definition exists for this skill.

## Cross-review triage — always inline

`/cross-review` triage runs in the main context (not delegated to a subagent) because triage accuracy depends on implementation context — knowing *why* the code was written that way, what design decisions were made, what the plan's non-goals are, and what the self-review already addressed. A subagent would lack this context and produce unreliable classifications (more false negatives in DISMISSED, more false positives in ACTION_REQUIRED).

The triage step reads existing artifacts (plan, self-review report, verify report) and produces `docs/reports/cross-review-triage-<slug>.md`. No new subagent definition is needed.

## Post-implementation pipeline under Codex

Codex supports subagents and project-scoped custom agents under
`.codex/agents/`. When ralph runs under Codex in the standard flow
(`RALPH_PRIMARY_CLI=codex` or detected at runtime), use the matching custom
agents (`reviewer`, `verifier`, `tester`, `doc-maintainer`) in the canonical
order. Each agent writes the same reports to `docs/reports/` that the Claude
Code subagent path produces. This keeps artifact parity for `/cross-review`
triage, the cycle cap, and `/pr`.

If Codex cannot dispatch a subagent (tool error, missing agent definition, or
environment limitation), run that step inline and note the fallback in the
report. Do not silently skip the phase.

Cap protection: the same `RALPH_STANDARD_MAX_PIPELINE_CYCLES` ceiling applies,
so a runaway inline pipeline cannot loop more than `cap` total runs.
