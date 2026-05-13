# codex-hook-duplicate-doctor

- Status: Complete
- Owner: Codex
- Date: 2026-05-13
- Related request: Codex startup warning for duplicate hook representations
- Branch: fix-codex-hook-duplicate-doctor

## Objective

Make `ralph doctor` catch the same duplicate Codex hook representation that
Codex reports at startup when `.codex/config.toml` inline hooks and
`.codex/hooks.json` coexist in the same project layer.

## Scope

- `internal/cli/doctor.go`
- `internal/cli/cli_test.go`
- Root and template `.codex/README.md`
- Root and template `.codex/config.toml`
- Review, verify, and test artifacts

## Non-goals

- Changing the supported Codex hook representation away from inline
  `.codex/config.toml` entries.
- Adding `.codex/hooks.json` to the scaffold.
- Changing Claude Code hook behavior.

## Cause

The repository already standardizes on `.codex/config.toml` inline
`[[hooks.*]]` entries. Codex emits the startup warning when a stray
`.codex/hooks.json` file is present beside that inline `[hooks]` table.
The shipped tree no longer contains `.codex/hooks.json`, but `ralph doctor`
did not previously fail if a local stray file reappeared.

## Acceptance Criteria

- [x] No tracked `.codex/hooks.json` is introduced.
- [x] `ralph doctor` fails when `.codex/config.toml` has inline hooks and
  `.codex/hooks.json` also exists.
- [x] The failure detail tells the operator to remove `hooks.json` and keep
  `.codex/config.toml` as source of truth.
- [x] A unit test covers the duplicate representation case.
- [x] Root/template `.codex` docs stay synchronized.

## Verify Plan

- `go test ./internal/cli -run TestCheckCodexEffectiveConfig`
- `git diff --check`
- `./scripts/check-sync.sh`
- `./scripts/run-static-verify.sh`

## Test Plan

- `./scripts/run-test.sh`

## Risks

- False positive if a project intentionally uses `hooks.json` only. Mitigated
  by failing only when the inline `[hooks]` representation is also present.

## Rollout

Merge normally. Existing projects with only `.codex/config.toml` are
unchanged. Projects with a stray `.codex/hooks.json` will get a clear doctor
failure before the next Codex startup warning.

## Progress

- [x] Cause identified.
- [x] Implementation completed.
- [x] Self-review completed.
- [x] Verification completed.
- [x] Tests completed.
