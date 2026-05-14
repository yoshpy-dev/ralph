# Walkthrough: pr-ready-branch-type-enforcement

- Date: 2026-05-14
- Plan: `docs/plans/archive/2026-05-14-pr-ready-branch-type-enforcement.md`
- Reason: Large workflow/scaffold diff touching scripts, skills, templates, tests, and documentation.

## What changed

### Branch naming

- Added `scripts/branch-name.sh` as the single branch naming policy:
  - `allowed-types`
  - `from-plan <plan-path>`
  - `validate <branch-name>`
  - `type <branch-name>`
  - `title-prefix <branch-name>`
- Allowed user-facing PR branch shapes are:
  - `<type>/<slug>`
  - `<type>/<issue>/<slug>`
- Controlled types are `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`, `perf`, `release`, and `security`.
- Old uncontrolled shapes like `issue-*`, `codex-*`, `integration/*`, and `slice/*` are rejected.

### Plan metadata

- Added `Type` to standard and Ralph Loop plan templates.
- Updated `new-feature-plan.sh` and `new-ralph-plan.sh` to accept `--type <type>` with `feat` as the default.
- Added tests proving generated plans produce the expected typed branches.

### PR readiness

- Added `scripts/ensure-pr-ready.sh`.
- It reads `gh pr view --json isDraft`; when the PR is Draft, it runs `gh pr ready`, then verifies `isDraft=false`.
- If the PR remains Draft or GitHub state cannot be read, the script fails closed.

### PR title prefix

- Added `scripts/ensure-pr-title-prefix.sh`.
- It reads the PR `headRefName`, derives the branch type, and enforces the matching `<type>:` title prefix.
- Missing prefixes are added, wrong allowed prefixes are replaced, and the script re-reads the PR title before reporting success.

### Claude Code and Codex parity

- Updated both `.claude/skills/*` and `.agents/skills/*` so:
  - plan creation chooses a branch type,
  - work/loop branch creation calls `branch-name.sh`,
  - PR titles use the branch type prefix,
  - PR creation does not pass `--draft` unless explicitly requested,
  - PR completion requires `ensure-pr-title-prefix.sh`,
  - PR completion requires `ensure-pr-ready.sh`.

### Pipeline and Ralph Loop

- `scripts/ralph-pipeline.sh` validates the current PR branch and runs `ensure-pr-title-prefix.sh` and `ensure-pr-ready.sh` after PR URL discovery.
- `scripts/ralph-orchestrator.sh` uses `branch-name.sh from-plan` for the final PR branch and validates generated slice branches.
- Ralph Loop PR titles derive the conventional type from the branch prefix and are post-verified.

### Templates and docs

- Mirrored all scaffold changes under `templates/base/`.
- Updated README, recipes, definition of done, repo map, Codex override notes, and path-scoped rules to describe typed PR branches and ready-state guards.

## Verification summary

- Static verification: PASS.
- Behavioral tests: PASS.
- Sync checks: PASS.
- `git diff --check`: PASS.

## Residual risk

- Tests stub `gh` rather than making live GitHub calls. The live guards are exercised by the actual PR creation flow through `scripts/ensure-pr-title-prefix.sh` and `scripts/ensure-pr-ready.sh`.
