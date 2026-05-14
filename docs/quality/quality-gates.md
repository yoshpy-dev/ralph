# Quality gates

## Inner loop: fast and local

Use these for rapid feedback:
- hook guardrails
- targeted linting
- targeted type checks
- targeted tests
- plan and report updates

## Outer loop: stricter and broader

Use these in CI or later-stage review:
- wider test suites
- integration and e2e checks (when implemented)
- architecture or structure checks
- secret scanning, dependency checks, and broader security scans (when implemented)
- deployment validation (when implemented)

## Suggested gate policy

### Must pass locally before "done"

- `./scripts/run-verify.sh` (all checks, backward-compatible)
- `./scripts/run-static-verify.sh` (static analysis only — wrapper for `HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh`, changed-language scope by default, used by `/verify`)
- `./scripts/run-test.sh` (tests only — wrapper for `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`, changed-language scope by default, used by `/test`)
- project-specific local checks
- plan and docs sync if behavior changed

`run-static-verify.sh` and `run-test.sh` must stay non-overlapping:
- static mode runs format checks, linters, static analyzers, type checks,
  syntax checks, and drift checks; it must not run behavioral tests
- test mode runs behavioral unit, integration, and regression tests; it must not
  run format checks, linters, static analyzers, type checks, syntax-only gates,
  or drift checks
- `run-verify.sh` without `HARNESS_VERIFY_MODE` remains the backward-compatible
  aggregate and may run both static verification and tests
- `run-verify.sh` defaults to `RALPH_VERIFY_SCOPE=full`; the `/verify` and
  `/test` wrappers default to `RALPH_VERIFY_SCOPE=changed`. Changed scope runs
  only language packs affected by the current git diff while always running
  project-local gates. Shared or ambiguous changes fall back to full scope.

### Must pass in CI before merge

- `./scripts/secret-scan.sh --range <merge-base>..HEAD` — pull request secret leak scan (`.github/workflows/verify.yml`)
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (`.github/workflows/verify.yml`)
- `./scripts/check-template.sh` (`.github/workflows/check-template.yml`)
- `./scripts/check-sync.sh` — templates/root parity (`.github/workflows/check-template.yml`)
- `./scripts/check-coverage.sh` — language-pack coverage gate (graceful skip if no packs)
- `./scripts/check-pipeline-sync.sh` — pipeline order consistency across 8 reference files

### Not yet implemented in CI

The following are aspirational gates listed for future adoption:
- broader test coverage (unit, integration, e2e)
- dependency vulnerability scans and broader security scans beyond secret leak detection
- org or repo-specific policy checks

## Pipeline mode gates (`ralph-pipeline.sh`)

When running in pipeline mode, the orchestrator enforces its own gates autonomously:

### Inner Loop gates (per cycle)

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Preflight probe | `--preflight` checks claude CLI, jq, CLAUDE.md, git | Pipeline blocked |
| Hook parity check | `run_hook_parity()` emulates hook safety checks | Warning logged |
| Stuck detection | HEAD commit hash comparison (3 consecutive no-change) | Pipeline aborted |
| Self-review | `claude -p` with `pipeline-self-review.md` (diff quality only; no tests, static analysis, spec verification, doc drift, or broad repo audit) | CRITICAL findings logged |
| Verify | `claude -p` with `pipeline-verify.md` (spec compliance + static analysis + documentation drift; runs `run-static-verify.sh` internally with changed-language scope unless the pipeline exports full scope) | Verdict logged |
| Test | `claude -p` with `pipeline-test.md` (behavioral tests only; runs `run-test.sh` internally with changed-language scope unless the pipeline exports full scope + root cause analysis) | Retry Inner Loop |
| COMPLETE gating | Tests pass + COMPLETE signal required to advance; tests pass without COMPLETE → continue Inner Loop (return 6) | Inner Loop continues |
| Repair attempt limit | `MAX_REPAIR_ATTEMPTS` (default 5) | Escalate to human |

Each agent writes reports to both `.harness/state/pipeline/` (orchestrator) and `docs/reports/` (PR pre-checks), plus a sidecar signal file for machine-readable pass/fail detection.

### Outer Loop gates

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Codex ACTION_REQUIRED | Codex triage finds actionable issues | Regress to Inner Loop |
| Iteration limit | `MAX_ITERATIONS` (default 20) | Pipeline stopped |
| Inner cycle limit | `MAX_INNER_CYCLES` (default 10) | Move to Outer Loop |

### Pipeline state

- Checkpoint: `.harness/state/pipeline/checkpoint.json`
- Reports: `.harness/state/pipeline/inner-*-*.log`
- Use `./scripts/ralph status` to inspect

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
