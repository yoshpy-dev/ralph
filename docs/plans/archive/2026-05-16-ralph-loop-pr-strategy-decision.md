# ralph-loop-pr-strategy-decision

- Status: Complete
- Owner: Claude Code
- Date: 2026-05-16
- Related request: ralph-loop-pr-strategy-decision
- Related issue: 92
- Type: feat
- Branch: feat/92-ralph-loop-pr-strategy-decision

## Objective

Close the policy gap left after grouped Ralph Loop PR support by making the PR strategy decision explicit in plans/manifests: AI recommends the strategy, humans approve it at plan approval time, stacked requires a dependency rationale, and runtime overrides are visible rather than silent.

## Scope

- Add a Ralph Loop manifest decision section for PR strategy rationale and human approval metadata.
- Parse decision metadata in `scripts/ralph-orchestrator.sh`.
- Warn when `--pr-strategy` overrides the manifest/decision strategy.
- Warn when `stacked` lacks a recorded dependency rationale.
- Persist decision metadata into orchestrator state/report for `ralph status`.
- Update docs/templates and base-template mirrors.
- Add focused shell tests for decision parsing, stacked rationale warnings, override mismatch warnings, and status rendering.

## Non-goals

- Block all unapproved Ralph Loop runs. Approval metadata should be surfaced first; hard enforcement can be a later policy decision.
- Redesign grouped/stacked PR creation from PR #91.
- Add remote GitHub approval checks.

## Assumptions

- The product decision from issues #90/#92 is settled: grouped remains default, stacked is opt-in for real dependency chains, unified is a fallback.
- Manifest metadata is the source of truth once a Ralph Loop run starts.
- Runtime override should remain available as an escape hatch, but with explicit operator feedback.

## Affected areas

- `scripts/ralph-orchestrator.sh`
- `scripts/ralph-status-helpers.sh`
- `docs/plans/templates/ralph-loop-manifest.md`
- `docs/recipes/ralph-loop.md`
- `docs/quality/definition-of-done.md`
- `.claude/rules/post-implementation-pipeline.md`
- `.claude/rules/subagent-policy.md`
- `templates/base/` mirrors for changed shipped files
- `tests/test-ralph-orchestrator-pr-strategy.sh`
- `tests/test-ralph-status.sh`
- `scripts/verify.local.sh` if the focused test list needs updating

## Design decisions

- Decision contract: AI recommends, human approves. Rationale: this keeps automation useful while preserving review/process ownership at plan approval.
- Enforcement level: warn for missing approval and missing stacked rationale in this PR. Rationale: existing Ralph Loop manifests should not be broken immediately; warnings make the policy visible and testable.
- Runtime override: allow but warn when it differs from manifest/decision metadata. Rationale: escape hatches are still useful during recovery, but silent drift undermines the manifest as source of truth.
- Critical forks: None. The high-level policy was decided in issue #92.

## Acceptance criteria

- [x] Ralph Loop manifest template includes a PR strategy decision section.
- [x] Plan/rules docs state that AI recommends the strategy and humans approve it at plan approval time.
- [x] Stacked strategy emits a clear validation warning when dependency rationale is missing.
- [x] Runtime `--pr-strategy` override that differs from manifest strategy emits a clear warning with both values.
- [x] Status output surfaces selected strategy and recorded human approval state when available.
- [x] Docs explain grouped default, stacked dependency-chain opt-in, and unified fallback.
- [x] Tests cover decision parsing, stacked-rationale warning behavior, override mismatch warning, and status output.

## Implementation outline

1. Extend manifest template with `[pr_strategy_decision]` and group rationale examples.
2. Add shell helpers to parse decision fields from `_manifest.md`.
3. Integrate warnings and state/report fields into the orchestrator startup path.
4. Extend table/JSON status output for `pr_strategy_decision`.
5. Update docs/rules/templates and mirror them under `templates/base/`.
6. Add/update focused shell tests.
7. Run focused tests, sync checks, shell syntax/static checks, and full verify where feasible.

## Verify plan

- Static analysis checks: `sh -n` on changed shell scripts; `shellcheck` where available; `git diff --check`; `./scripts/check-sync.sh`; `./scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm: all acceptance criteria above, especially warning text and status JSON/table fields.
- Documentation drift to check: root docs vs `templates/base/` mirrors; manifest template and generated plan behavior.
- Evidence to capture: verification/test reports in `docs/reports/`; full verify log if run.

## Test plan

- Unit tests: focused shell assertions for manifest decision parsing via dry-run output.
- Integration tests: orchestrator dry-run with `stacked`, missing rationale, and CLI override mismatch.
- Regression tests: existing Ralph Loop strategy tests, status tests, run-option tests, sync checks.
- Edge cases: omitted decision section, `human_approved = false`, `human_approved = true`, override mismatch, stacked without group rationale.
- Evidence to capture: focused test output and full `run-verify.sh` result.

## Risks and mitigations

- Risk: warnings are too weak to enforce the intended process. Mitigation: make approval/rationale visible in status now; hard fail can be introduced once existing manifests are migrated.
- Risk: manual shell parsing of TOML-like snippets is brittle. Mitigation: keep schema flat and deterministic; add tests for the supported patterns.
- Risk: docs and templates drift. Mitigation: update base mirrors and run sync checks.

## Rollout or rollback notes

- Rollout: new Ralph Loop manifests include decision metadata; existing manifests continue to run with warnings/defaults.
- Rollback: remove decision parsing/status fields while keeping PR strategy execution from PR #91 intact.

## Open questions

- None for this PR. Enforcement can be tightened in a follow-up once warning-only behavior has been dogfooded.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
