# Codex triage report: mojibake-postedit-guard

- Date: 2026-04-17
- Plan: docs/plans/active/2026-04-17-mojibake-postedit-guard.md
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Total Codex findings: 3
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=1

## Triage context

- Active plan: docs/plans/active/2026-04-17-mojibake-postedit-guard.md
- Self-review report: docs/reports/self-review-mojibake-postedit-guard.md (no CRITICAL/HIGH/MEDIUM)
- Verify report: docs/reports/verify-mojibake-postedit-guard.md (PASS, 13/13 AC)
- Test report: docs/reports/test-mojibake-postedit-guard.md (PASS, 88 assertions)
- Implementation context summary: The branch adds a PostToolUse U+FFFD detection hook, registers it in settings.json (matcher extended `Edit|Write` → `Edit|Write|MultiEdit`), and mirrors both to templates/base/. Non-goal: fixing pre-existing harness-wide design issues (HARNESS_VERIFY_MODE adoption in real verifiers).

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | [P3] `PostToolUseFailure` matcher is still `Bash\|Edit\|Write`; MultiEdit failures never increment `.harness/state/tool_failures.count`. | Real inconsistency introduced by this PR. We added MultiEdit to `PostToolUse` but forgot the symmetric change on the failure matcher. Trivial fix (1 line × 2 files). Blocks merge because we created the asymmetry. | `.claude/settings.json:116`, `templates/base/.claude/settings.json:116` |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 2 | [P2] `scripts/verify.local.sh` does not honor `HARNESS_VERIFY_MODE` (`static`/`test`/`all`). Running `run-static-verify.sh` still executes the hook smoke tests, and `run-test.sh` still runs shellcheck/jq/sync. | Real behavior drift from the documented mode split (`quality-gates.md:26-27`). However, the existing `packs/languages/golang/verify.sh` also ignores the mode — the split is currently aspirational except in `packs/languages/_template/verify.sh`. Adding mode support to `verify.local.sh` is a small improvement that moves us toward the documented contract without breaking callers. Worth fixing in this PR (low cost). | `scripts/verify.local.sh:28-65` |

## DISMISSED

| # | Codex finding | Dismissal reason | Category |
|---|---------------|------------------|----------|
| 3 | [P1] `tests/test-check-mojibake.sh` Case E restricts PATH but does not link `dirname`; the hook's `HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"` supposedly exits 127 so the test should fail. | Empirically verified: `bash tests/test-check-mojibake.sh` PASSes 11/11 on the dev machine (macOS 15.3, bash 3.2). The test passes because (a) `HOOK_REPO_ROOT` override bypasses the `dirname`-dependent `REPO_ROOT` derivation in the hook, and (b) POSIX `set -e` does not propagate command-substitution failures to the outer shell, so an empty `HOOK_DIR` does not abort the script. The factual claim "fails on a normal writable machine" is not reproducible. For defense-in-depth we will still add `dirname` to the linked tool set, but this is hardening, not a fix. | false-positive |

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
