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
- dependency and security scans (when implemented)
- deployment validation (when implemented)

## Suggested gate policy

### Must pass locally before "done"

- `./scripts/run-verify.sh` (all checks, backward-compatible)
- `./scripts/run-static-verify.sh` (static analysis only — wrapper for `HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh`, used by `/verify`)
- `./scripts/run-test.sh` (tests only — wrapper for `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`, used by `/test`)
- project-specific local checks
- plan and docs sync if behavior changed

### Must pass in CI before merge

- `./scripts/run-verify.sh` (`.github/workflows/verify.yml`)
- `./scripts/check-template.sh` (`.github/workflows/check-template.yml`)

### Not yet implemented in CI

The following are aspirational gates listed for future adoption:
- broader test coverage (unit, integration, e2e)
- dependency and security scans
- org or repo-specific policy checks

## Pipeline mode gates (`ralph-pipeline.sh`)

When running in pipeline mode, the orchestrator enforces its own gates autonomously:

### Inner Loop gates (per cycle)

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Preflight probe | `--preflight` checks claude CLI, jq, CLAUDE.md, git | Pipeline blocked |
| Hook parity check | `run_hook_parity()` emulates hook safety checks | Warning logged |
| Stuck detection | HEAD commit hash comparison (3 consecutive no-change) | Pipeline aborted |
| Test pass | `run-test.sh` or `run-verify.sh HARNESS_VERIFY_MODE=test` | Retry Inner Loop |
| Repair attempt limit | `MAX_REPAIR_ATTEMPTS` (default 5) | Escalate to human |

### Outer Loop gates

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Codex ACTION_REQUIRED | Codex triage finds actionable issues | Regress to Inner Loop |
| Iteration limit | `MAX_ITERATIONS` (default 20) | Pipeline stopped |
| Inner cycle limit | `MAX_INNER_CYCLES` (default 5) | Move to Outer Loop |

### Pipeline state

- Checkpoint: `.harness/state/pipeline/checkpoint.json`
- Reports: `.harness/state/pipeline/inner-*-*.log`
- Use `./scripts/ralph status` to inspect

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
