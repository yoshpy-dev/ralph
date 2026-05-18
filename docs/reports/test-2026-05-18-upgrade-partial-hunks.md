# Test report: upgrade-partial-hunks

- Plan: `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`
- Related issue: #97
- Verdict: PASS
- Evidence:
  - `docs/evidence/verify-2026-05-18-021543.log` (`./scripts/run-test.sh`)
  - `docs/evidence/verify-2026-05-18-021712.log` (`./scripts/run-verify.sh`)

## Test Runs

| Command | Result | Notes |
| --- | --- | --- |
| `go test ./internal/scaffold ./internal/upgrade ./internal/cli` | PASS | Targeted package loop after implementation changes |
| `go test ./...` | PASS | Full Go package suite |
| `./scripts/run-test.sh` | PASS | Behavioral test mode, changed scope |
| `./scripts/run-verify.sh` | PASS | Full verification including Go tests |

## Coverage Added

- `internal/scaffold/baseline_test.go`: baseline write/read, escaping-path rejection, read-path prefix guard.
- `internal/scaffold/manifest_test.go`: v2 baseline field round trip and v1 manifest compatibility.
- `internal/cli/cli_test.go`: init baseline metadata, dry-run diff no-mutation behavior, v1 prompt fallback, hunk prompt option shape, baseline metadata after apply.

## Residual Risk

The full #97 merge editor is not implemented in this slice. Tests intentionally cover the new baseline-gated prompt and safety fallback, not final 3-way partial adoption semantics.
