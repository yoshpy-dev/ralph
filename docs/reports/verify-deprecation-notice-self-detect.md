# Verify report: deprecation-notice-self-detect

- Date: 2026-07-24
- Plan: docs/plans/active/2026-07-24-deprecation-notice-self-detect.md
- Verifier: verifier subagent
- Scope: spec compliance + static analysis (changed-language scope)
- Commit under review: 46d8ff0 (fix: exclude self from Go CLI detection in shell deprecation notice)
- Base: ef12eaae (main)
- Diff: `git diff ef12eaae...HEAD`

## Evidence reviewed

- `scripts/ralph` lines 647–652 (deprecation notice block, post-fix)
- `templates/base/scripts/ralph` (byte-identical mirror)
- `tests/test-ralph-deprecation-notice.sh` (cases 1–5, new case 5 added)
- `./scripts/run-static-verify.sh` output (exit 0)
- `sh -n scripts/ralph` / `sh -n templates/base/scripts/ralph` (POSIX syntax check)
- `shellcheck -S warning scripts/ralph` (no warnings)
- `cmp scripts/ralph templates/base/scripts/ralph` (exit 0)
- `AGENTS.md` line 82 (doc drift check)

## Spec compliance: per-AC verdict

### AC1: PATH="$PWD/scripts:$PATH" ralph <cmd> → notice does not appear (test case 5)

**PASS — structurally satisfied.**

Implementation (scripts/ralph:647–652):

```sh
if [ -z "${RALPH_NO_DEPRECATION:-}" ]; then
  _ralph_resolved="$(command -v ralph 2>/dev/null || true)"
  if [ -n "$_ralph_resolved" ] && ! [ "$_ralph_resolved" -ef "$0" ]; then
    printf '...legacy...' >&2
  fi
fi
```

When `PATH="$PROJECT_ROOT/scripts:/usr/bin:/bin"` and the script is invoked as
`ralph`, `command -v ralph` resolves to the same inode as `$0`. The `-ef` test
(`! [ "$_ralph_resolved" -ef "$0" ]`) evaluates to false, so the inner `printf`
is never reached and no notice is emitted.

Test case 5 in `tests/test-ralph-deprecation-notice.sh` (lines 115–133) asserts
both notice-absence ("this shell entrypoint is legacy" NOT in combined_out) and
normal-startup ("=== Ralph Pipeline Status ===" IS in combined_out). The test also
guards against sibling-source failures that could produce false greens.

Behavioral runtime pass/fail is deferred to /test.

### AC2: Fake Go ralph binary on PATH → notice appears (existing test case 1)

**PASS — structurally satisfied.**

When `_ralph_resolved` resolves to a different inode from `$0` (the fake binary
at `fake-bin/ralph`), `! [ "$_ralph_resolved" -ef "$0" ]` is true. The `printf`
fires and outputs to stderr.

Test case 1 (lines 78–87) covers this case with `check_contains`. Existing
coverage, confirmed not regressed by the diff (diff adds 3 lines to the notice
block without touching the positive-notice path).

### AC3: RALPH_NO_DEPRECATION=1 → no notice (existing test case 3)

**PASS — structurally satisfied.**

The outer `if [ -z "${RALPH_NO_DEPRECATION:-}" ]` short-circuits the entire block,
including the `command -v ralph` call and `-ef` check. Test case 3 (lines 99–105)
covers this with `check_not_contains`. Not touched by the diff; confirmed intact.

### AC4: scripts/ralph and templates/base/scripts/ralph are byte-identical

**PASS — deterministically verified.**

```
cmp scripts/ralph templates/base/scripts/ralph
```

Exit 0. No output. The diff confirms both files received the identical change
(diff hunks are line-by-line identical for both paths).

### AC5: test case 5 for self-detection regression present (notice-absence + normal-startup assertions)

**PASS — structurally satisfied.**

`tests/test-ralph-deprecation-notice.sh` lines 115–133 contain:

- `check_not_contains` for "this shell entrypoint is legacy" — absence of notice
- `check_not_contains` for "No such file or directory" — absence of sibling-source failure
- `check_contains` for "=== Ralph Pipeline Status ===" — positive startup confirmation

Three assertions covering notice-absence and normal-startup, satisfying the plan
requirement for both checks. This directly addresses the Codex plan advisory HIGH
finding (single-file-copy false greens avoided by using real `scripts/` in PATH
and checking the status header is reachable).

Total test count expected: 4 existing + 3 new assertions = 7 assertions total.
Pass/fail count confirmed as behavioral; deferred to /test.

## Static analysis results

### run-static-verify.sh (changed-language scope)

Exit code: **0 (all verifiers passed)**

Checks run:
- shellcheck hook + verify scripts: OK
- sh -n for all .claude/hooks/*.sh and templates/base/.claude/hooks/*.sh: OK
- jq -e . .claude/settings.json: OK
- jq -e . templates/base/.claude/settings.json: OK
- Codex hook single-source guard: OK
- Codex inline hook detector smoke test: OK
- Codex PR provenance policy guard: OK
- scripts/check-sync.sh: PASS (DRIFTED: 0, ROOT_ONLY: 0)
- scripts/check-pipeline-sync.sh: all pipeline steps referenced in all 9 files checked
- scripts/check-skill-sync.sh: 13 skills in lock-step
- Go verifier (gofmt + go vet): 0 issues

Evidence: docs/evidence/verify-2026-07-24-093000.log

### Supplementary checks

- `sh -n scripts/ralph`: OK (POSIX syntax valid)
- `sh -n templates/base/scripts/ralph`: OK (POSIX syntax valid)
- `shellcheck -S warning scripts/ralph`: no warnings
- `shellcheck -S warning tests/test-ralph-deprecation-notice.sh`: no warnings

## Documentation drift check

### AGENTS.md line 82

Current text:

> `scripts/` — ... legacy `ralph` shell CLI (prints deprecation notice when Go `ralph` binary is on PATH; retirement tracked in docs/tech-debt/README.md) ...

This description is semantically accurate after the fix. The behavior it describes
("prints deprecation notice when Go `ralph` binary is on PATH") remains true in the
normal case. The fix only adds precision: the notice is now suppressed when the
binary on PATH is the script itself. The map-level description does not need to
enumerate this edge-case condition — it would add noise without benefit given
AGENTS.md is kept intentionally brief.

**Verdict: no update required.** The plan's verify plan (line 80) also concluded
"挙動の本質は変わらないため更新不要見込みだが /sync-docs で確認" — aligns with
this assessment. Flag for /sync-docs to confirm.

### Plan AC checklist

AC1–AC5 checkboxes are marked [x] in the plan (commit f5a1a36). Per memory note
"Plan AC checkboxes lag behind implementation", stale or pre-marked checkboxes are
not a /verify failure; they are flagged as doc drift only. In this case the
checkboxes accurately reflect the implementation, so no drift to report.

## Findings summary

| ID | Area | Severity | Finding | Status |
|----|------|----------|---------|--------|
| V1 | AC4 byte-identical | — | cmp exit 0 confirmed deterministically | VERIFIED |
| V2 | AGENTS.md doc drift | INFO | "prints deprecation notice when Go `ralph` binary is on PATH" — accurate at the map level; self-exclusion detail is intentionally omitted from the brief map entry | NO UPDATE NEEDED |
| V3 | static analysis | — | run-static-verify.sh exit 0, all checks passed | VERIFIED |
| V4 | shell syntax | — | sh -n and shellcheck -S warning both clean for changed files | VERIFIED |

No FAIL-level findings.

## What remains unverified (deferred to /test)

| Item | Reason |
|------|--------|
| AC1 runtime: notice truly absent when scripts/ on PATH | Requires shell execution; deferred to /test |
| AC2 runtime: notice fires for a foreign ralph binary | Requires shell execution; deferred to /test |
| AC3 runtime: RALPH_NO_DEPRECATION=1 suppresses notice | Requires shell execution; deferred to /test |
| AC5 runtime: 7 passed, 0 failed from full test run | Requires shell execution; deferred to /test |
| `-ef` behavior under bash 3.2 (macOS) vs GNU bash | Both are bash; confirmed in plan Assumptions; runtime validates |

## Overall verdict

**PASS**

All 5 acceptance criteria are structurally satisfied. Static analysis is clean
(run-static-verify.sh exit 0, shellcheck clean, POSIX syntax valid, byte-identical
confirmed). No documentation update is required. No blocking findings.

Proceed to /test.
