# __TITLE__

- Status: Draft
- Owner: Claude Code
- Date: __DATE__
- Related request: __REQUEST__
- Related issue: __ISSUE__
- Branch: TBD
- Execution: Ralph Loop (autonomous pipeline)

## Objective

## Scope

## Non-goals

## Assumptions

## Affected areas

## Acceptance criteria

- [ ]

## Implementation outline

1.

## Vertical slices

Define each slice as an independent unit of work that can run through the full
pipeline (implement -> self-review -> verify -> test -> sync-docs -> codex-review -> PR)
in its own Git worktree.

### Shared-file locklist

Files that must not be modified by parallel slices simultaneously.
If a slice needs to modify a locked file, it must run sequentially after
all other slices touching that file have completed.

The orchestrator auto-detects overlapping affected files across slices and
adds them here. Manually list any additional shared files.

- `CLAUDE.md`
- `AGENTS.md`

### Slice 1: __SLICE_NAME__

- Objective:
- Acceptance criteria: []
- Affected files: []
- Dependencies: none

### Slice 2: __SLICE_NAME__

- Objective:
- Acceptance criteria: []
- Affected files: []
- Dependencies: none | [slice 1]

## Slice dependency graph

```
slice-1 ──→ slice-3
slice-2 ──→ slice-3
```

Independent slices run in parallel. A slice starts only after all its
dependencies complete.

## Verify plan

- Static analysis checks:
- Spec compliance criteria to confirm:
- Documentation drift to check:
- Evidence to capture:

## Test plan

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
- [ ] Branch created
- [ ] Pipeline execution started
- [ ] All slices complete
- [ ] Integration merge check passed
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR(s) created
