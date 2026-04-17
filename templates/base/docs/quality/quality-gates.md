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
- project-specific CI checks as needed

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
| Self-review | `claude -p` with `pipeline-self-review.md` (agent-driven, 10-item checklist) | CRITICAL findings logged |
| Verify | `claude -p` with `pipeline-verify.md` (agent-driven, runs `run-static-verify.sh` internally) | Verdict logged |
| Test | `claude -p` with `pipeline-test.md` (agent-driven, runs `run-test.sh` internally + root cause analysis) | Retry Inner Loop |
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
