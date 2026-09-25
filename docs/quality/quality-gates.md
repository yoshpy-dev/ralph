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

- `./scripts/secret-scan.sh --range <merge-base>..HEAD` — pull request secret leak scan (`.github/workflows/verify.yml`); the range scan pins the local git settings that change which added lines `git log -p` prints, so a local run scans the same added lines as CI except for a few local-only attribute sources it cannot pin (listed in the `scripts/secret-scan.sh` header and `docs/tech-debt/README.md`), and exits 3 (not clean) when git cannot read the range
  - Local mirror of this same range scan: `./scripts/secret-scan-branch.sh` runs automatically from `./scripts/run-verify.sh` (static and all modes, default mode — an unscannable state is a one-line notice, not a failure) and again with `--strict` from `/pr` right before push
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (`.github/workflows/verify.yml`)
- `./scripts/check-template.sh` (`.github/workflows/check-template.yml`)
- `./scripts/check-sync.sh` — templates/root parity (`.github/workflows/check-template.yml`)
- `./scripts/check-coverage.sh` — language-pack coverage gate (graceful skip if no packs) (`.github/workflows/verify.yml`)
- `./scripts/check-pipeline-sync.sh` — pipeline order consistency across 6 reference files (`.github/workflows/verify.yml`)
- `./scripts/check-skill-sync.sh` — `.claude/skills/` ↔ `.agents/skills/` mirror parity (`.github/workflows/verify.yml`)

### Not yet implemented in CI

The following are aspirational gates listed for future adoption:
- broader test coverage (unit, integration, e2e)
- dependency vulnerability scans and broader security scans beyond secret leak detection
- org or repo-specific policy checks

## Org runtime gates

Autonomous multi-seat execution (org runtime, `ralph org` verbs) enforces its
own gates deterministically, independent of any LLM judgment:

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Envelope validation | model pool / role pool / `max_seats` checked by `ralph org spawn` | Spawn rejected, recorded in manifest |
| Watchdog pulse layer | stall / liveness / scope-change ALERT and deadman human escalation (`ralph org watch`) | ALERT to lead; unanswered alerts escalate to a human |
| Quality pipeline gate | impl exit checks → QA (`run-static-verify.sh` / `run-test.sh`) → reviewer → lead arbitration | QA fail routes back to impl before reviewer sees it |

See `.claude/rules/ralph/agent-messaging.md` for the org runtime protocol and
`.harness/state/org/manifest.jsonl` for the append-only audit trail.

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
