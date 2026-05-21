# Verify report: language-pack-monorepo-roots

- Date: 2026-05-21
- Plan: `docs/plans/archive/2026-05-21-language-pack-monorepo-roots.md`
- Verifier: Codex
- Scope: Acceptance criteria, static analysis, template sync, and documentation drift.
- Evidence: `docs/evidence/verify-2026-05-21-language-pack-monorepo-roots.log`

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n` on edited verifier, detection, and test scripts | PASS | Shell syntax passed before aggregate verification. |
| `shellcheck --severity=warning` on edited shell scripts | PASS | No warning-level findings after root-scope updates. |
| `scripts/check-sync.sh` | PASS | `templates/base/scripts/*` and `templates/packs/*` mirrors are in sync. |
| `scripts/check-template.sh` | PASS | Template structure and executable checks passed. |
| `./scripts/run-static-verify.sh` | PASS | Local static gates, sync gates, Go verifier, `gofmt`, and `staticcheck` passed. |

## Acceptance criteria

| Criterion | Status | Evidence |
| --- | --- | --- |
| Shipped packs execute from nested project roots | Verified | `tests/test-language-pack-monorepo-roots.sh` covers TypeScript, Python, Rust, Go, Dart, and Terraform nested roots. |
| Single-root behavior is preserved | Verified | `tests/test-verify-mode-split.sh` still passes for existing per-pack mode dispatch. |
| `jvm` is no longer emitted without a shipped pack | Verified | `tests/test-detect-changed-languages.sh` and `tests/test-language-pack-monorepo-roots.sh` cover JVM marker fallback/no emit. |
| Template mirrors are updated | Verified | `scripts/check-sync.sh` passed with 0 drifted files. |
| Changed-scope can narrow to project roots | Verified | `scripts/detect-changed-languages.sh` emits `<lang>_roots`, `run-verify.sh` passes `RALPH_VERIFY_PROJECT_ROOTS`, and `tests/test-run-verify-scope.sh` covers the handoff. |
| Format checks are non-mutating in CI paths | Verified | Dart now uses `--output=none --set-exit-if-changed`; Terraform uses root-local `fmt -check`. |

## Observational checks

- The aggregate static verifier selected full language scope because pack files changed, then ran the Go verifier for the repo root successfully.

## Coverage gaps

- Root lists remain space-separated to match the existing language scope contract; paths containing spaces are not newly supported.

## Verdict

- Verified: All acceptance criteria.
- Partially verified: None.
- Not verified: None.
