# pr-ready-branch-type-enforcement

- Status: Ready for PR
- Owner: Claude Code
- Date: 2026-05-14
- Related request: pr-ready-branch-type-enforcement
- Related issue: N/A
- Type: fix
- Branch: fix/pr-ready-branch-type-enforcement

## Objective

Make PR readiness and branch type prefixes deterministic across both Claude Code
and Codex flows. PR creation must not remain Draft unless explicitly requested,
and new task branches must use a controlled type prefix such as `feat/`,
`fix/`, `docs/`, or `chore/`.

## Scope

- Add shared scripts for branch-name generation/validation and PR ready-state
  enforcement.
- Wire the scripts into the standard PR skill and Ralph Loop pipeline paths.
- Add branch `Type` metadata to plan templates and plan creation scripts.
- Mirror all shipped scaffold changes under `templates/base/`.
- Add regression tests for branch naming and PR ready-state enforcement.

## Non-goals

- Changing GitHub repository settings or default branch protections.
- Rewriting archived historical plan branch names.
- Changing per-slice internal branch naming unless it affects the final PR head
  branch.

## Assumptions

- `gh pr create` without `--draft` should create ready PRs, but connector or
  agent-specific paths can still produce drafts, so post-create verification is
  required.
- Ralph Loop slice branches can remain internal `slice/...` branches; the final
  PR head branch should use the controlled type prefix.

## Affected areas

- `.agents/skills/pr/SKILL.md`
- `.agents/skills/work/SKILL.md`
- `.agents/skills/loop/SKILL.md`
- `.claude/skills/pr/SKILL.md`
- `.claude/skills/work/SKILL.md`
- `.claude/skills/loop/SKILL.md`
- `docs/plans/templates/`
- `scripts/`
- `templates/base/` mirrors
- `tests/`

## Design decisions

Critical forks: None. The requested behavior needs deterministic local
verification rather than another natural-language instruction.

## Acceptance criteria

- [x] AC-1: Standard PR instructions for both Claude Code and Codex require a
  ready-for-review PR by default and forbid completing the flow while `isDraft`
  remains true.
- [x] AC-2: Ralph Loop pipeline/orchestrator PR creation paths verify draft
  state after PR creation and mark a draft PR ready before reporting success.
- [x] AC-3: Plan templates include branch type metadata, and plan creation
  scripts can set it without breaking default usage.
- [x] AC-4: Standard `/work` and `/loop` branch creation instructions use a
  shared branch-name generator instead of free-form agent inference.
- [x] AC-5: Branch validation accepts only controlled type prefixes for
  user-facing PR branches.
- [x] AC-6: Regression tests cover branch generation/validation and PR
  ready-state enforcement without making network calls.
- [x] AC-7: Root and `templates/base/` scaffold copies remain in sync.

## Implementation outline

1. Add `scripts/branch-name.sh` and `scripts/ensure-pr-ready.sh`.
2. Update plan templates and creation scripts for `Type`.
3. Update skills and pipeline scripts to use the shared scripts.
4. Add shell regression tests.
5. Run sync, static verification, and tests.

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`,
  `./scripts/check-skill-sync.sh`, `./scripts/check-sync.sh`.
- Spec compliance criteria to confirm: each AC above with file references.
- Documentation drift to check: skill instructions, Codex override, plan
  templates, and Ralph Loop recipes if branch shape changes.
- Evidence to capture: raw command output in `docs/evidence/`.

## Test plan

- Unit tests: none for Go unless script wiring affects Go code.
- Integration tests: focused shell tests for branch naming and PR ready
  enforcement.
- Regression tests: `check-skill-sync`, `check-sync`, static verify, changed
  test runner.
- Edge cases: invalid type, issue/no-issue branch names, draft PR converted to
  ready, ready PR left unchanged, `gh` failure propagates.
- Evidence to capture: `docs/evidence/test-2026-05-14-pr-ready-branch-type-enforcement.log`.

## Risks and mitigations

- Risk: older active plans without `Type` cannot generate branch names.
  Mitigation: the generator fails with a clear message; new templates/scripts
  add `Type` going forward.
- Risk: `gh pr ready` is unavailable or fails. Mitigation: the script fails
  closed and prevents the PR from being reported as complete.

## Rollout or rollback notes

Rollback by reverting the scripts, skill instructions, template metadata, and
pipeline calls from this PR.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (`fix/pr-ready-branch-type-enforcement`)
- [x] Implementation started
- [x] Review artifact created (`docs/reports/self-review-2026-05-14-pr-ready-branch-type-enforcement.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-05-14-pr-ready-branch-type-enforcement.md`)
- [x] Test artifact created (`docs/reports/test-2026-05-14-pr-ready-branch-type-enforcement.md`)
- [ ] PR created
