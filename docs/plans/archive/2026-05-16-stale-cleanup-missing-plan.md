# stale-cleanup-missing-plan

- Status: Complete
- Owner: Codex
- Date: 2026-05-16
- Related request: stale-cleanup-missing-plan
- Related issue: 94
- Type: fix
- Branch: fix/94/stale-cleanup-missing-plan

## Objective

Make `ralph cleanup --stale` recover from Ralph Loop orchestrator state whose
recorded plan directory has already disappeared, so cleanup can remove the stale
state and `ralph status` stops reporting an unactionable run.

## Scope

- Add a stale-cleanup path for missing Ralph Loop plan metadata.
- Preserve strict validation for explicit `ralph cleanup --plan <plan-dir>`.
- Keep template `scripts/ralph` in sync with the repo root script.
- Add regression coverage for valid-plan cleanup and missing-plan stale cleanup.

## Non-goals

- Changing normal Ralph Loop branch naming, PR strategy, or orchestration.
- Deleting remote branches.
- Reconstructing full branch cleanup when the plan metadata is unavailable.

## Assumptions

- A missing recorded plan path is itself stale state when `--stale` selected the
  current orchestrator record and it is older than the threshold.
- Slice worktrees can be removed only when they are clearly derived from existing
  orchestrator state files.

## Affected areas

- `scripts/ralph`
- `templates/base/scripts/ralph`
- `tests/test-ralph-orchestrator-pr-strategy.sh`
- `docs/plans/active/2026-05-16-stale-cleanup-missing-plan.md`

## Design decisions

<!-- Critical forks resolved with the user. Each entry: decision, chosen option, rationale. -->
Critical forks: None.

- Missing-plan recovery is limited to `cleanup --stale`; explicit `--plan`
  retains the existing validation error.
- When the plan is unavailable, branch cleanup is skipped with an explicit log
  because branch names are derived from the plan manifest.

## Acceptance criteria

- [x] `ralph cleanup --stale --older-than 0d --dry-run` succeeds when the recorded plan path is missing.
- [x] `ralph cleanup --stale --older-than 0d` archives and removes stale orchestrator state when the plan path is missing.
- [x] `ralph status` no longer reports the stale run after cleanup.
- [x] Explicit `ralph cleanup --plan <missing-dir>` still fails clearly.
- [x] Existing `cleanup --plan <valid-plan>` behavior is unchanged.
- [x] Tests cover stale cleanup with a missing plan path and preserve the valid-plan path behavior.

## Implementation outline

1. Add a helper that handles stale orchestrator state without plan metadata.
2. In `cmd_cleanup --stale`, call that helper only after the stale threshold is met and the plan is missing or invalid.
3. Mirror the change into `templates/base/scripts/ralph`.
4. Extend the PR strategy regression test with dry-run, non-dry cleanup, status, and explicit missing-plan assertions.

## Verify plan

- Static analysis checks: `sh -n`, `shellcheck`, `git diff --check`, `scripts/check-sync.sh`, `scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm: all issue #94 acceptance criteria.
- Documentation drift to check: template sync and plan/report artifacts.
- Evidence to capture: focused regression output and full verify output.

## Test plan

- Unit tests: none; shell helper behavior is covered through script regression tests.
- Integration tests: `tests/test-ralph-orchestrator-pr-strategy.sh`.
- Regression tests: full local verify suite.
- Edge cases: missing recorded plan path, explicit missing `--plan`, valid plan dry-run.
- Evidence to capture: command output in docs reports.

## Risks and mitigations

- Risk: cleanup removes too much when metadata is incomplete. Mitigation: the
  missing-plan path removes only state selected by `--stale`, clearly derived
  slice worktrees, and skips branch cleanup.
- Risk: template drift. Mitigation: update root and template scripts together
  and run sync checks.

## Rollout or rollback notes

- Rollout: merge the PR; no migration needed.
- Rollback: revert the commit to restore strict stale cleanup behavior.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
