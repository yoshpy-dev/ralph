# ralph-loop-grouped-pr

- Status: Complete
- Owner: Codex
- Date: 2026-05-15
- Related request: Implement issue #90: make grouped PRs the Ralph Loop default and clean up temporary integration branches.
- Related issue: 90
- Type: feat
- Branch: feat/90-ralph-loop-grouped-pr

## Objective

Implement the first production-ready grouped PR workflow for Ralph Loop while preserving the existing unified PR path. The orchestrator should read a PR strategy/group contract from the Ralph Loop manifest, create reviewable grouped PRs by default, keep full integration verification, and clean up temporary integration artifacts on success while retaining diagnostics on failure.

## Scope

- Add Ralph Loop manifest metadata for `pr_strategy` and `pr_groups`.
- Default Ralph Loop PR strategy to `grouped`; keep `unified` as an explicit fallback and add `stacked` as a parsed/documented opt-in placeholder.
- Teach `scripts/ralph-orchestrator.sh` to:
  - parse PR strategy/group metadata from `_manifest.md`,
  - create group branches from completed slice branches,
  - run full integration verification on a temporary integration branch,
  - create grouped PRs,
  - preserve unified behavior when requested,
  - clean temporary integration/slice artifacts after successful grouped PR creation.
- Update status output, CLI help, templates, docs, and base-template mirrors.
- Add deterministic shell tests for parsing, grouped branch behavior, cleanup, and docs/script drift.

## Non-goals

- Remove unified PR support.
- Implement full automated stacked PR retarget/rebase handling in this PR; `stacked` may be parsed and documented as opt-in/future behavior if full implementation exceeds the safe scope.
- Delete diagnostic branches after failure.
- Depend on GitHub UI-only behavior that cannot be verified in scripts.
- Rewrite Ralph Loop from shell to Go.

## Assumptions

- The issue discussion has already resolved the core product decision: `grouped` should be the standard Ralph Loop behavior, with `unified` retained as fallback.
- A deterministic manifest contract is preferable to auto-inferred grouping.
- Full integration verification remains required before grouped PRs are considered ready.
- GitHub CLI remains the available implementation path for PR creation in the shell orchestrator.

## Affected areas

- `scripts/ralph-orchestrator.sh`
- `scripts/ralph`
- `scripts/ralph-status-helpers.sh`
- `scripts/branch-name.sh` if temporary/group branch validation needs adjustment
- `docs/plans/templates/ralph-loop-manifest.md`
- `docs/plans/templates/ralph-loop-slice.md`
- `docs/recipes/ralph-loop.md`
- `docs/quality/definition-of-done.md`
- `AGENTS.md`
- `templates/base/` mirrors
- `tests/test-ralph-orchestrator-*.sh`, `tests/test-ralph-status.sh`, and new focused tests as needed

## Design decisions

- PR strategy default: `grouped`. Rationale: Ralph Loop is used for larger slice-based work; grouped PRs preserve reviewability better than a single final PR.
- Integration branch role: temporary verification artifact in grouped/stacked mode. Rationale: the submitted artifacts are group PR branches; cleanup can be automatic after successful PR creation.
- Failure cleanup: retain diagnostic branches and print deterministic cleanup instructions. Rationale: deleting investigation material on failure would make recovery harder.
- Critical forks: None remaining; the high-leverage product choices were resolved in issue #90.

## Acceptance criteria

- [x] Ralph Loop manifest supports `pr_strategy = grouped | stacked | unified` and `grouped` is the default when omitted.
- [x] Manifest supports explicit `[[pr_groups]]` with group names and slice lists.
- [x] Orchestrator can create grouped PR branches from completed slice branches.
- [x] Orchestrator still runs full merged integration verification.
- [x] Grouped PR creation refuses to proceed if integration verification required fixes that are only present on the temporary integration branch.
- [x] Successful grouped PR creation records PR URLs and cleanup status in orchestrator state.
- [x] Temporary integration local branch is cleaned up after successful grouped PR creation.
- [x] Diagnostic integration branch is retained and cleanup instructions are printed on integration merge, verification, or PR creation failure.
- [x] Existing unified PR behavior remains available and covered by tests.
- [x] `ralph status` shows PR strategy, groups, integration branch status, cleanup status, and PR URLs where available.
- [x] Docs/templates describe grouped default, unified fallback, stacked opt-in, and cleanup lifecycle.
- [x] Root/template mirror checks pass.

## Implementation outline

1. Inspect current manifest parsing, orchestrator state, PR creation, and branch validation.
2. Add manifest parsing helpers for PR strategy and PR groups.
3. Add grouped branch creation/merge/PR creation flow while preserving the current unified path.
4. Add cleanup helpers for temporary integration and slice branches, with failure retention.
5. Extend status JSON/table rendering for strategy/groups/cleanup.
6. Update docs/templates and root/template mirrors.
7. Add deterministic tests with fake `git`/`gh` where practical.
8. Run focused tests, sync checks, and full verification where feasible.

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`; focused `shellcheck` output if available; `./scripts/check-sync.sh`; `./scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm: all acceptance criteria above, especially grouped default, unified fallback, integration verification preservation, and cleanup semantics.
- Documentation drift to check: root docs vs `templates/base/` mirrors; AGENTS/CLAUDE parity where applicable; recipe/DoD/help output consistency.
- Evidence to capture: verification logs under `docs/evidence/`; review/verify/test reports under `docs/reports/`.

## Test plan

- Unit tests: targeted shell tests for manifest parsing and branch naming helpers.
- Integration tests: orchestrator dry-run/preflight tests for grouped strategy and cleanup-state rendering; fake `gh` tests for PR creation where possible.
- Regression tests: existing branch-name, dry-run side-effect, slice skip-PR, status, sync, and verify-mode tests.
- Edge cases: omitted `pr_strategy`, invalid strategy, empty group, unknown slice in group, PR creation failure retaining branches, cleanup failure reporting.
- Evidence to capture: focused test logs plus full `run-verify.sh` output.

## Risks and mitigations

- Risk: shell implementation grows too complex. Mitigation: keep grouped helpers small and deterministic; defer advanced stacked behavior if needed.
- Risk: integration pipeline auto-fix semantics are hard to safely backport to groups. Mitigation: fail closed when integration branch differs from submitted group branches after verification.
- Risk: cleanup accidentally deletes useful diagnostics. Mitigation: cleanup only on fully successful grouped PR creation; failure paths retain branches.
- Risk: branch naming changes break PR title-prefix checks. Mitigation: keep submitted group branches in validated `<type>/<slug>` shape and use a clearly temporary namespace only for non-PR integration branches if validation allows.

## Rollout or rollback notes

- Rollout: grouped becomes default for new Ralph Loop manifests/docs; existing explicit `--unified-pr` workflows continue to work.
- Rollback: pass/use `unified` strategy to restore previous single-PR behavior; revert orchestrator grouped path without removing unified helpers.

## Open questions

- Resolved: stacked uses manifest group order, basing each group branch on the previous group branch. Advanced automatic retarget/rebase handling remains out of scope.
- Resolved: grouped/stacked integration branches are local verification artifacts in this PR. Remote deletion only applies if a future workflow pushes temporary integration branches for remote CI.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
