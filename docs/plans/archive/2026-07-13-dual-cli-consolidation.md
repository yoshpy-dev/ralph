# Plan: Dual-CLI consolidation — shared shell lib + deprecation path

- Status: approved
- Flow: standard (/work)
- Branch: refactor/dual-cli
- Base: develop
- Date: 2026-07-13

## Objective

Remove the double-maintenance between the legacy shell CLI (`scripts/ralph`)
and the Go CLI, and deduplicate utility functions copy-pasted across the core
scripts:

- `default_branch()` implemented identically in `scripts/ralph:489` and
  `scripts/ralph-worktree.sh:83`.
- `ts()` / `log()` / `log_error()` re-declared in `scripts/ralph`,
  `scripts/ralph-orchestrator.sh`, and `scripts/ralph-pipeline.sh`.
- Active-plan-dir detection copy-pasted in `scripts/ralph` (2 sites) and
  mirrored in Go (`internal/cli/run.go detectLatestPlanDir` — Go side stays).

## Scope

- New `scripts/ralph-common.sh`: `ts`, `log`, `log_error`, `default_branch`,
  `detect_active_plan_dir` (single implementation). Sourced by
  `scripts/ralph`, `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`,
  `scripts/ralph-worktree.sh`. Duplicated definitions deleted.
- Behavior-preserving: identical log format, identical default-branch logic.
  `detect_base_branch` in `ralph-cli-driver.sh` intentionally differs
  (merge-base heuristic) and is left alone with a pointer comment.
- Deprecation path for the shell CLI: `scripts/ralph` prints a one-line
  stderr notice when the Go `ralph` binary is available on PATH, recommending
  it. Behavior otherwise unchanged. Suppressible via `RALPH_NO_DEPRECATION=1`.
- Docs: AGENTS.md repo map line for `scripts/` mentions `ralph-common.sh` and
  the shell CLI's legacy status.
- Mirror to `templates/base/scripts/` (check-sync gate).

## Non-goals

- No removal of the shell CLI (that is the Phase-9-adjacent follow-up).
- No behavior change in orchestrator/pipeline logic.
- No change to Go code.

## Acceptance criteria

- AC1: exactly one definition each of `ts`, `log`, `log_error`,
  `default_branch`, `detect_active_plan_dir` across the five scripts
  (grep-verifiable), all sourced from `ralph-common.sh`.
- AC2: log output format is byte-identical before/after (spot-check via
  existing tests, which assert on log lines).
- AC3: `./scripts/ralph status`, `plan`, dry-run `run` behave as before —
  existing tests (test-ralph-*.sh) all pass unmodified.
- AC4: deprecation notice appears on stderr when a `ralph` binary is on PATH,
  is absent otherwise, and is suppressed by RALPH_NO_DEPRECATION=1; covered
  by a new test.
- AC5: `./scripts/check-sync.sh` and `./scripts/run-verify.sh` pass.

## Verify plan

- `bash tests/test-ralph-dry-run-side-effects.sh` and the other test-ralph-*.sh
- new `tests/test-ralph-deprecation-notice.sh`
- `shellcheck --severity=warning scripts/ralph-common.sh`
- `./scripts/check-sync.sh && ./scripts/run-verify.sh`

## Risks

- R1: sourcing order/`set -u` interactions when a script sources the lib
  before defining its own strict mode — the lib must not set shell options
  itself and must be safe under `set -eu` and `set -euo pipefail`.
- R2: `scripts/ralph` execs the orchestrator; PATH-based binary detection must
  not slow down or break non-interactive use — detection is one `command -v`.

## Rollout

Single PR to `develop`. Shell CLI removal tracked separately in tech-debt.
