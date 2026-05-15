# __TITLE__

- Status: Draft
- Owner: Claude Code
- Date: __DATE__
- Related request: __REQUEST__
- Related issue: __ISSUE__
- Type: __TYPE__
- Branch: TBD
- Integration branch: TBD
- Execution: Ralph Loop (parallel slices)
- PR strategy: grouped

## Objective

## Scope

## Non-goals

## Assumptions

## Affected areas

## Design decisions

<!-- Critical forks resolved with the user. Each entry: decision, chosen option, rationale. -->
<!-- No critical forks? Write: "Critical forks: None" -->

## PR grouping

Ralph Loop defaults to grouped PRs. Keep related slices together so each PR is reviewable.
Use `unified` only for small or atomic changes, and `stacked` only when group order is a real dependency chain.
The AI proposes this strategy during planning; human approval at plan approval time makes it final.

```toml
pr_strategy = "grouped" # grouped | stacked | unified

[pr_strategy_decision]
selected = "grouped" # grouped | stacked | unified
recommended_by = "ai"
human_approved = false
approval_note = ""
rationale = "Generated default: grouped PRs keep Ralph Loop work reviewable while allowing parallel CI and review."

[[pr_groups]]
name = "implementation"
slices = [__PR_GROUP_SLICES__]

[[pr_strategy_decision.group_rationale]]
name = "implementation"
independent = true
depends_on = []
reason = "Generated default group; split this if slices can be reviewed independently."
```

## Shared-file locklist

Files that must not be modified by parallel slices simultaneously.
The orchestrator auto-detects overlapping affected files and adds them here.
Manually list any additional shared files.

- `CLAUDE.md`
- `AGENTS.md`

## Dependency graph

```
slice-1 ──→ slice-3
slice-2 ──→ slice-3
```

Independent slices run in parallel. A slice starts only after all its
dependencies complete.

## Integration-level verify plan

- Static analysis checks:
- Spec compliance criteria to confirm:
- Documentation drift to check:
- Evidence to capture:

## Integration-level test plan

- Unit tests:
- Integration tests:
- Regression tests:
- Edge cases:
- Evidence to capture:

## Risks and mitigations

## Rollout or rollback notes

## Open questions

## Progress checklist

- [ ] Plan reviewed
- [ ] Slices defined and dependencies mapped
- [ ] Shared-file locklist finalized
- [ ] Integration branch created
- [ ] Pipeline execution started
- [ ] All slices complete
- [ ] Sequential merge to typed integration branch passed
- [ ] Integration-level verification passed
- [ ] Grouped PRs created, or unified PR created when explicitly selected
- [ ] Temporary integration branch cleanup completed, or diagnostics retained with cleanup instructions
