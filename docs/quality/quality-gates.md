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
- integration and e2e checks
- architecture or structure checks
- dependency and security scans
- deployment validation

## Suggested gate policy

### Must pass locally before "done"

- `./scripts/run-verify.sh`
- project-specific local checks
- plan and docs sync if behavior changed

### Must pass in CI before merge

- broader test coverage
- static analysis beyond the inner loop
- any org or repo-specific policy checks

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
