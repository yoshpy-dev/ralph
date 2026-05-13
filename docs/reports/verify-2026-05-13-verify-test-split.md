# Verify report: verify-test-split

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-verify-test-split.md`
- Verifier: Codex
- Scope: spec compliance + static analysis + documentation drift
- Evidence: `docs/evidence/verify-2026-05-13-141403.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| Every language verifier supports `HARNESS_VERIFY_MODE=static|test|all`. | met | Go, TypeScript, Python, Rust, and Dart verifiers now branch on `static`, `test`, and `all`; Terraform and template verifiers already did. |
| Static mode does not execute language test commands or shell regression test suites. | met | `tests/test-verify-mode-split.sh` asserts static mode skips `go test`, `npm test`, `pytest`, `cargo test`, and `dart test`. |
| Test mode does not execute format, lint, static analysis, type checks, drift checks, or syntax-only gates. | met | `tests/test-verify-mode-split.sh` asserts test mode skips Go format/vet/staticcheck, TypeScript lint/typecheck, Python ruff/mypy, Rust fmt/clippy, and Dart format/analyze. |
| `run-static-verify.sh` and `run-test.sh` remain strict wrappers. | met | Wrappers still set `HARNESS_VERIFY_MODE=static` and `HARNESS_VERIFY_MODE=test`; `verify.local.sh` dispatches mode-specific functions. |
| `run-verify.sh` default mode remains backward-compatible. | met | Full `./scripts/run-verify.sh` passed in `all` mode; evidence `docs/evidence/verify-2026-05-13-141419.log`. |
| Pipeline verify prompts call only static verification. | met | Existing pipeline verify prompts call `./scripts/run-static-verify.sh`; `check-pipeline-sync.sh` passed. |
| Pipeline test prompts call only behavioral test execution. | met | Existing pipeline test prompts call `./scripts/run-test.sh`; `check-pipeline-sync.sh` passed. |
| Verifier subagents call only static verification. | met | Claude/Codex verifier agent definitions and template mirrors now name `./scripts/run-static-verify.sh`, forbid `./scripts/run-test.sh` and behavioral tests, and are covered by `tests/test-agent-phase-boundaries.sh`. |
| Tester subagents call only behavioral tests. | met | Claude/Codex tester agent definitions and template mirrors now name `./scripts/run-test.sh`, forbid `./scripts/run-static-verify.sh`, static analyzers, type checks, and drift checks, and are covered by `tests/test-agent-phase-boundaries.sh`. |
| Self-review is defined and guarded as diff-quality only. | met | Self-review skill bodies, reviewer agent, and pipeline self-review prompts now forbid tests/static/spec/doc-drift/broad audits; `tests/test-self-review-scope.sh` passed. |
| Codex reviewer agents follow the self-review definition. | met | `.codex/agents/reviewer.toml` and template mirror now forbid tests/static/spec/doc-drift/broad audits and are included in `tests/test-self-review-scope.sh`. |
| Root/template and Claude/Codex mirrors remain synchronized. | met | `scripts/check-sync.sh` and `scripts/check-skill-sync.sh` passed inside `run-static-verify.sh`. |
| Regression tests cover mode separation, self-review scope, and subagent phase boundaries. | met | Added `tests/test-verify-mode-split.sh`, `tests/test-self-review-scope.sh`, and `tests/test-agent-phase-boundaries.sh`; all are wired into `verify.local.sh` test mode. |
| Documentation matches the implementation contract. | met | `docs/quality/quality-gates.md` and `docs/quality/definition-of-done.md` plus template mirrors updated; sync checks passed. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass | Evidence `docs/evidence/verify-2026-05-13-141403.log`; shellcheck, `sh -n`, `jq`, sync checks, skill sync, `gofmt`, `go vet`, and `golangci-lint` passed. |
| `sh -n tests/test-agent-phase-boundaries.sh tests/test-self-review-scope.sh scripts/verify.local.sh` | pass | Syntax check for the new/changed shell regression guards. |
| `shellcheck --severity=warning tests/test-agent-phase-boundaries.sh tests/test-self-review-scope.sh scripts/verify.local.sh` | pass | Focused shellcheck for the new/changed shell scripts. |
| `tests/test-agent-phase-boundaries.sh` | pass | 44/44 assertions passed for Claude/Codex verifier and tester agent boundaries. |
| `tests/test-self-review-scope.sh` | pass | 96/96 assertions passed after adding Codex reviewer agent definitions to scope. |
| `./scripts/run-verify.sh` | pass | Final all-mode gate passed after subagent boundary reinforcement; evidence `docs/evidence/verify-2026-05-13-170321.log`. |
| `git diff --check` | pass | No whitespace errors. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/quality/quality-gates.md` | yes | Defines non-overlap for static/test wrappers and pipeline phase scopes. |
| `docs/quality/definition-of-done.md` | yes | Treats phase boundary violations as not done. |
| `.claude/skills/` and `.agents/skills/` | yes | Self-review scope tightened and skill sync passed. |
| `.claude/agents/` and `.codex/agents/` | yes | Reviewer/verifier/tester role boundaries are explicit and covered by regression tests. |
| `templates/base/` and `templates/packs/` | yes | Mirrors updated; `check-sync.sh` passed. |

## Observational checks

- `./scripts/run-static-verify.sh` was run with escalated permissions because
  Go static tools write to the user Go cache outside the sandbox.

## Coverage gaps

- No coverage percentage was collected; this change is primarily shell routing
  and prompt contract behavior.

## Verdict

- Verified: all acceptance criteria
- Partially verified: none
- Not verified: none
