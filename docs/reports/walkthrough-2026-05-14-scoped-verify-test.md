# Walkthrough: scoped-verify-test

- Date: 2026-05-14
- Related issue: #71
- Branch: `codex-scoped-verify-test`

## What Changed

- Added `scripts/detect-changed-languages.sh` and the matching template copy.
- Added `RALPH_VERIFY_SCOPE=full|changed` handling in `scripts/run-verify.sh`.
- Changed `run-static-verify.sh` and `run-test.sh` wrappers to default to changed-language scope.
- Kept `run-verify.sh` and CI full-scope by default.
- Made Ralph Loop integration (`--skip-pr --fix-all`) export full scope, while per-slice runs default to changed scope.
- Added detector and runner integration tests.
- Updated quality docs, pipeline rules, skills, prompts, and language-pack recipe docs.

## Safety Properties

- Project-local gates still run before language-pack selection.
- Shared, ambiguous, language-pack, CI, and detector/script changes fall back to full scope.
- `.harness/*` runtime files are ignored by changed-language detection so one verify run does not poison the next run.
- Machine-readable scope state is written to `.harness/state/verify-scope`.

## Validation Summary

- `tests/test-detect-changed-languages.sh`: pass
- `tests/test-run-verify-scope.sh`: pass
- `tests/test-verify-mode-split.sh`: pass
- `scripts/check-sync.sh`: pass
- `scripts/check-skill-sync.sh`: pass
- `go test ./...`: pass
- `./scripts/run-static-verify.sh`: pass
- `./scripts/run-test.sh`: pass

## Follow-up

- CI remains the full-scope merge gate after PR creation.
