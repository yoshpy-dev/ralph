# verify-test-split

- Status: Draft
- Owner: Codex
- Date: 2026-05-13
- Related request: Strictly split verify, test, and self-review pipeline responsibilities
- Related issue: 69
- Branch: issue-69-verify-test-split

## Objective

Make the post-implementation pipeline strict and non-overlapping: static
verification belongs to `run-static-verify.sh` / `/verify`, behavioral tests
belong to `run-test.sh` / `/test`, and self-review remains diff-quality review
only. Eliminate duplicate static/test execution in language verifiers and add
regression coverage so the split does not drift again.

## Scope

- Add `HARNESS_VERIFY_MODE=static|test|all` handling to every language verifier
  and its `templates/packs/` mirror.
- Keep `run-verify.sh` default `all` behavior backward-compatible.
- Strengthen docs, prompts, and self-review definitions so phase boundaries are
  explicit.
- Add deterministic regression tests for verifier mode separation and
  self-review scope.
- Keep root, template, Claude, and Codex mirrors synchronized.

## Non-goals

- Changing the public names of `run-verify.sh`, `run-static-verify.sh`, or
  `run-test.sh`.
- Removing the backward-compatible `all` mode.
- Replacing the Ralph Loop pipeline architecture.
- Optimizing individual test suites beyond removing duplicate phase work.

## Assumptions

- Commands that are intrinsically part of a language test runner may remain in
  the test phase, but format/lint/type/static commands must not run in `test`
  mode.
- Optional tools should continue to be skipped when not installed, matching the
  current verifier style.
- Standard flow is appropriate because the change is broad but tightly scoped
  and not independently sliceable enough to justify Ralph Loop.

## Affected areas

- `packs/languages/*/verify.sh`
- `templates/packs/*/verify.sh`
- `scripts/run-verify.sh`, `scripts/run-static-verify.sh`,
  `scripts/run-test.sh`, `scripts/verify.local.sh`
- `tests/`
- `.claude/skills/`, `.agents/skills/`, `.claude/agents/`
- `templates/base/` mirrors
- `docs/quality/`

## Design decisions

Critical forks: None.

- Use the existing `HARNESS_VERIFY_MODE` contract instead of adding new script
  entry points. Rationale: existing wrappers and Terraform/template verifiers
  already establish the pattern.
- Add lightweight shell regression tests with fake command stubs instead of
  invoking real external language toolchains. Rationale: deterministic,
  portable, and focused on command routing.

## Acceptance criteria

- [ ] Every language verifier supports `HARNESS_VERIFY_MODE=static|test|all`.
- [ ] Static mode does not execute language test commands or shell regression
  suites.
- [ ] Test mode does not execute format, lint, static analysis, type checks,
  drift checks, or syntax-only gates.
- [ ] `run-static-verify.sh` and `run-test.sh` remain strict wrappers around
  those contracts.
- [ ] `run-verify.sh` default mode remains backward-compatible.
- [ ] Pipeline verify prompts call only static verification.
- [ ] Pipeline test prompts call only behavioral test execution.
- [ ] Self-review is defined and guarded as diff-quality only.
- [ ] Root/template and Claude/Codex mirrors remain synchronized.
- [ ] Regression tests cover mode separation and self-review scope.
- [ ] Documentation matches the implementation contract.

## Implementation outline

1. Refactor language verifiers into `run_static` and `run_tests` functions.
2. Mirror language verifier changes into `templates/packs/`.
3. Add a shell regression test that exercises each verifier with fake tools and
   asserts static/test/all command routing.
4. Add a prompt/scope regression test for self-review boundaries.
5. Wire new regression tests into `scripts/verify.local.sh` test mode.
6. Update docs and prompt wording where needed.
7. Run sync, static verify, test, and full verify gates.

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`,
  `./scripts/check-sync.sh`, `./scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm: all acceptance criteria above.
- Documentation drift to check: `docs/quality/`, `.claude/skills/`,
  `.agents/skills/`, pipeline prompts, `templates/base/`.
- Evidence to capture: verify report and raw static verifier output.

## Test plan

- Unit tests: existing Go package tests via `./scripts/run-test.sh`.
- Integration tests: `./scripts/run-verify.sh` default `all` mode.
- Regression tests: new verifier mode split test; new self-review scope test.
- Edge cases: optional tool absence; unknown `HARNESS_VERIFY_MODE`; language
  verifier marker absence.
- Evidence to capture: test report and raw test output.

## Risks and mitigations

- Risk: moving a command to the wrong mode silently drops coverage.
  Mitigation: fake-tool routing tests assert both positive and negative command
  execution per language verifier.
- Risk: template drift.
  Mitigation: run `check-sync.sh` and keep `templates/packs/` in lock-step.
- Risk: Claude/Codex skill drift.
  Mitigation: run `check-skill-sync.sh`.

## Rollout or rollback notes

- Rollout is a normal PR. On regression, revert the verifier refactor and tests;
  default `run-verify.sh` behavior is preserved to reduce rollback risk.

## Open questions

- None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
