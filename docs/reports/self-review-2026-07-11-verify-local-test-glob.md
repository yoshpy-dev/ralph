# Self-Review: verify-local-test-glob

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-verify-local-test-glob.md
- Branch: fix/verify-local-test-glob (base main, merge-base 57757a0)
- Commit: e89acfe
- Scope: diff quality only (no test execution, no spec compliance, no doc-drift)

## Verdict

MERGE. No CRITICAL or HIGH findings. The glob refactor preserves the
run-wrapper / status semantics of the old chain, the shellcheck directives are
exactly scoped, the deleted `RALPH=` var is genuinely unused, and comments are
accurate.

## Finding counts by severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 0 |
| LOW      | 2 |

## What was reviewed

Files in scope (`git diff main...HEAD`):

- `scripts/verify.local.sh` — if-chain → glob in `run_hook_tests()`; explicit
  test enumeration → `tests/test-*.sh` in the `run_static_checks()` shellcheck
  target loop.
- `tests/test-ralph-config.sh` — file-level `# shellcheck disable=SC1090`.
- `tests/test-ralph-status.sh` — file-level `# shellcheck disable=SC1090,SC2034`;
  removed unused `RALPH=` assignment.
- `tests/test-secret-scan.sh` — `CDPATH=` → `CDPATH=''` (2 lines).
- `tests/test-ralph-cli-driver.sh`, `tests/test-xreview-gate-regression.sh`,
  `tests/test-xreview-prompt-render.sh` — `cd "$REPO_ROOT"` → `cd "$REPO_ROOT" || exit 1`.
- `docs/plans/active/2026-07-11-verify-local-test-glob.md` — plan artifact.

## Scope checklist (from task brief)

### 1. Glob loop preserves run-wrapper / status semantics — CONFIRMED

The new loop calls `run "$f" "$f"`, identical to the old chain. The `run()`
helper (verify.local.sh:131-141) wraps the invocation in
`if "$@"; then ... else status=1; fi`. Under `set -eu` (line 15), `set -e` is
suspended inside an `if` condition, so a failing test sets `status=1` and the
loop continues to the next file rather than aborting. Semantics are preserved
exactly. Evidence: the `run()` body is unchanged by the diff, and both the old
per-test `if [ -x ... ]; then run ...` and the new
`for f ...; do [ -x "$f" ] || continue; run "$f" "$f"; done` route through the
same wrapper. The `[ -x "$f" ]` guard is retained (old chain used `-x`), so
non-executable files are still skipped — behavior-preserving.

POSIX no-match safety: with no `nullglob` in POSIX sh, an unmatched
`tests/test-*.sh` yields the literal pattern, but `[ -x "$f" ]` rejects the
literal string, so no spurious `run` occurs. Verified empirically. (28 files
exist today, so this is only a robustness note.)

### 2. shellcheck directives are narrow, not masking real issues — CONFIRMED

Ran shellcheck `--severity=warning` on the directive-bearing files with the
`disable` line stripped, to enumerate exactly what each directive suppresses:

- `test-ralph-config.sh`: only `SC1090` (×19), all from
  `. "$CONFIG"` dynamic sourcing (the deliberate test mechanism). Directive
  comment "the whole point is sourcing $CONFIG dynamically" is accurate.
- `test-ralph-status.sh`: only `SC1090` (×6, from `. "$HELPERS"`) and
  `SC2034` (×2, `STATUS_NO_COLOR`). `STATUS_NO_COLOR` is genuinely consumed by
  the sourced helper (`scripts/ralph-status-helpers.sh:13`,
  `[ "${STATUS_NO_COLOR:-0}" -eq 1 ]`) — a true false-positive. Directive
  comment "vars are consumed by sourced functions" is accurate.

No other warning codes appear, so the file-level directives are not
over-broad. Notably, the genuinely-unused `RALPH=` var was **deleted** rather
than suppressed under SC2034 — the correct choice, and evidence that the author
distinguished dead code from false positives.

Full-suite static check safety: the widened `run_static_checks()` glob now
shellchecks ALL 28 test files (not the former subset). Ran
`shellcheck --severity=warning tests/test-*.sh` — zero warnings. The widened
static gate will pass; the glob-widening did not expose an un-remediated
warning.

### 3. Deleting the `RALPH=` line is safe — CONFIRMED

`grep RALPH tests/test-ralph-status.sh` returns no matches after the deletion.
The variable was assigned but never read anywhere in the file. Safe removal.

### 4. Comment accuracy — CONFIRMED

The `run_hook_tests()` comment claims 5 of 28 suites were silently unexecuted:
ralph-config, ralph-signals, ralph-status, xreview-gate-regression,
xreview-prompt-render.

- All 5 files exist on disk.
- `git show main:scripts/verify.local.sh` confirms none of the 5 appeared in the
  old `run_hook_tests()` chain — they were indeed silently skipped.
- Total test file count is exactly 28, matching the comment.

Comment is accurate and grep-friendly.

## Findings

### LOW-1: `run_static_checks` static-target loop and test loop use different existence tests

`run_static_checks()` builds its shellcheck arg list with `[ -f "$f" ] || continue`,
while `run_hook_tests()` uses `[ -x "$f" ] || continue`. Both are correct for
their purpose (shellcheck lints any readable file; the runner only invokes
executables), but the asymmetry is a minor readability snag for a future
reader diffing the two loops. No action required — documenting for awareness.

Evidence: verify.local.sh:150 (`[ -f "$f" ]`) vs verify.local.sh:207 (`[ -x "$f" ]`).

### LOW-2: cd-guard fix is scoped to glob-exposed files only (intentional, noted for completeness)

`tests/test-ralph-slice-skip-pr.sh` was NOT given a `cd || exit 1` guard. This
is correct — that file contains no `cd` statement, so there is nothing to
guard (verified: no `cd ` lines, shellcheck reports 0 SC2164). The three files
that received the guard (`test-ralph-cli-driver`, `test-xreview-gate-regression`,
`test-xreview-prompt-render`) all use `set -uo pipefail` (not `set -eu`), so an
unguarded `cd` failure would not abort — the `|| exit 1` addition is a genuine
robustness improvement for those files, not cosmetic. No inconsistency
introduced by the diff.

## Tech debt

None identified. This diff removes enumeration-drift debt rather than adding
any.

## Evidence summary

- `run()` semantics: verify.local.sh:131-141 unchanged; `set -e` suspended in
  `if` condition.
- 28 test files, all executable, all shellcheck-clean at `--severity=warning`.
- SC1090/SC2034 directives suppress only those exact codes (verified by
  stripping the directive and re-running shellcheck).
- `STATUS_NO_COLOR` consumed at ralph-status-helpers.sh:13.
- `RALPH=` has zero remaining references.
- Comment's "5 of 28 silently unexecuted" cross-checked against
  `git show main:` old chain and disk file list.
