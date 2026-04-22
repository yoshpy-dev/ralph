# __TITLE__

- Status: Draft
- Owner: Claude Code
- Date: __DATE__
- Related request: __REQUEST__
- Related issue: __ISSUE__
- Branch: TBD
- Integration branch: integration/__SLUG__
- Execution: Ralph Loop (parallel slices)

## Objective

## Scope

## Non-goals

## Assumptions

## Affected areas

## Design decisions

<!-- Critical forks resolved with the user. Each entry: 判断・採用した選択肢・理由（rationale）。 -->
<!-- No critical forks? Write: "Critical forks: なし" -->

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
- [ ] Sequential merge to integration branch passed
- [ ] Integration-level verification passed
- [ ] Unified PR created
