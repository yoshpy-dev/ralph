# Plan: Core runtime hardening — strict mode, shellcheck, tests, parser robustness

- Status: approved
- Flow: standard (/work)
- Branch: refactor/core-hardening
- Base: develop
- Date: 2026-07-13

## Objective

The core runtime (`ralph-orchestrator.sh` 1706 lines, `ralph-pipeline.sh`
1240 lines) is the least-verified part of the repo: no pipefail, excluded
from the shellcheck CI scope, `ralph-pipeline.sh` has zero direct tests, and
plan parsing uses fragile pipe-delimited records. Harden all four fronts.
(The full Go-native pipeline migration remains Phase 6b in tech-debt; this PR
is the bridge that makes the shell core safe until then.)

## Scope

1. **Strict mode**: `set -eu` → `set -euo pipefail` in
   `ralph-orchestrator.sh`, `ralph-pipeline.sh`, `scripts/ralph`, with an
   audit of every pipeline for grep/no-match exit-status leaks and equivalent
   pipefail hazards.
2. **Testability + direct tests**: add source guards
   (`if [[ "${BASH_SOURCE[0]}" == "$0" ]]`) so test files can source the
   scripts and unit-test functions. New `tests/test-ralph-pipeline-functions.sh`
   covering at least `ckpt_update` (filter application, temp-swap atomicity,
   quoting) and preflight probe JSON assembly; new
   `tests/test-ralph-orchestrator-parsers.sh` covering `parse_slices` (both
   format variants) and `parse_pr_groups` edge cases.
3. **Parser robustness**: `parse_slices` pipe-delimited records break when an
   objective/filename contains `|`. Switch the record delimiter to the ASCII
   unit separator (0x1F) or equivalently safe encoding; add a regression test
   with `|` in the objective.
4. **shellcheck CI scope**: add `ralph-orchestrator.sh`, `ralph-pipeline.sh`,
   `ralph-status-helpers.sh`, `ralph-cli-driver.sh`, and `scripts/ralph` to
   the shellcheck list in `verify.local.sh`; fix all `--severity=warning`
   findings (behavior-preserving; targeted disables with justification
   comments are acceptable where a fix would change behavior).
5. Mirror script changes to `templates/base/scripts/`.
6. Tech-debt: update the Phase 6b entry to note the hardening bridge landed.

## Non-goals

- No Go-native pipeline migration (Phase 6b, separate effort).
- No behavior changes to orchestration/pipeline logic.
- No restructuring of main() bodies beyond what strict mode requires.

## Acceptance criteria

- AC1: all three entrypoints run under `set -euo pipefail`; existing
  dry-run/signal/option tests pass unmodified.
- AC2: `tests/test-ralph-pipeline-functions.sh` and
  `tests/test-ralph-orchestrator-parsers.sh` pass and run in run-verify.
- AC3: a slice file with `|` in its Objective parses correctly end-to-end
  (regression test).
- AC4: `shellcheck --severity=warning` clean for the five scripts, and the
  scope expansion is active in `verify.local.sh`.
- AC5: `./scripts/check-sync.sh` and `./scripts/run-verify.sh` pass.

## Verify plan

- All existing tests/test-ralph-*.sh (behavior lock)
- New function-level tests above
- `shellcheck --severity=warning` on the five scripts
- `./scripts/check-sync.sh && ./scripts/run-verify.sh`

## Risks

- R1: pipefail exposes latent failures in long pipelines (grep no-match,
  head early-close SIGPIPE) — mitigated by an explicit pipeline audit and the
  existing test suite; every `|| true` added must carry a why-comment.
- R2: shellcheck fixes can subtly change quoting/expansion behavior —
  mitigated by fixing mechanically, running the full test battery, and
  preferring targeted `# shellcheck disable=` with justification over risky
  rewrites in hot paths.
- R3: source guards change top-level execution flow — guard only wraps the
  existing main invocation; all CLI tests re-run.

## Rollout

Single PR to `develop`. Split into two implementation waves:
wave A (strict mode + guards + parsers + tests), wave B (shellcheck scope +
fixes), each verified independently.
