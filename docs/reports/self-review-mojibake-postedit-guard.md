# Self-review report: mojibake-postedit-guard

- Date: 2026-04-17
- Plan: docs/plans/active/2026-04-17-mojibake-postedit-guard.md
- Branch: chore/mojibake-postedit-guard
- Reviewer: reviewer subagent (Claude)
- Scope: Diff quality only (naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability). Spec compliance and test coverage are explicitly out of scope (handled by /verify and /test).

## Evidence reviewed

- `git log main..HEAD` — 4 commits (22642c9, 3311dc6, 911c5ac, 7c4cc9e)
- `git diff main...HEAD --stat` — 14 files, +646/-2
- `.claude/hooks/check_mojibake.sh` (root + templates/base mirror, git mode 100755, byte-for-byte identical: `cmp` → exit 0; SHA `2b2626c...`)
- `.claude/hooks/mojibake-allowlist` (root + mirror, 100644, byte-for-byte identical: `cmp` → exit 0; SHA `d162329...`)
- `.claude/settings.json` + `templates/base/.claude/settings.json` (both show matcher `Edit|Write` → `Edit|Write|MultiEdit` and added hook entry)
- `scripts/verify.local.sh` (100755, repo-only)
- `tests/test-check-mojibake.sh` (100755, bash-based)
- `tests/fixtures/payloads/{edit,write,multiedit}.json`
- `scripts/check-sync.sh` diff — 4 new lines in `ROOT_ONLY_EXCLUSIONS` (`scripts/verify.local.sh`, `tests/`, `docs/plans/active/`)
- `AGENTS.md` diff — 1 line under `.claude/hooks/`
- `grep` for literal U+FFFD bytes (`EF BF BD`) across all new files — **no matches** (`grep` exit 1)
- `sh -n` + `bash -n` on all new shell scripts — clean
- `bash tests/test-check-mojibake.sh` — **11/11 PASS**
- `bash scripts/check-sync.sh` — PASS, 0 DRIFTED / 0 ROOT_ONLY, 107 IDENTICAL
- Cross-comparison with existing hooks (`post_edit_verify.sh`, `pre_bash_guard.sh`, `prompt_gate.sh`) — shebang, `set -eu`, and HOOK_DIR pattern match existing house style
- Manual probes: POSIX `case`-glob behavior with quoted `$REPO_ROOT`, unquoted `$normalised`, and `$(cmd)` inside pattern values
- tech-debt register (`docs/tech-debt/README.md`) — entry for "Per-slice pipeline CRITICAL behavior" already exists; no stale mojibake entry to reconcile

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `tests/test-check-mojibake.sh` cleanup trap unconditionally removes `$REPO_ROOT/.harness/state/mojibake-jq-missing`. If a real Claude Code session on the dev machine legitimately created that marker (jq genuinely missing), running the test erases the signal. The test should scope cleanup to files it created under `$workdir` (Case E already uses `$alt_root/.harness/state/...` for its own marker). | `tests/test-check-mojibake.sh` L51-55; marker write location in hook is `$REPO_ROOT/.harness/state/mojibake-jq-missing` (line 39). | Drop the `rm -f "$REPO_ROOT/.harness/state/mojibake-jq-missing"` line from `cleanup()`. Case E writes to `$alt_root/...`, so no repo-root file is ever created by the test itself. (Non-blocking — the marker is advisory, and a developer can re-trigger it by editing any file without jq.) |
| LOW | maintainability | Hook silently no-ops on malformed JSON payloads. `jq -r '...' 2>/dev/null \|\| true` swallows jq errors, producing an empty `file_path` which routes to `exit 0`. A payload with corrupted JSON (e.g. a future Claude Code bug) would bypass the scan without any log line. | `.claude/hooks/check_mojibake.sh` L44-48. Contract comment L19-21 says "If file does not exist, is empty, or has no U+FFFD, exit 0" but does not mention malformed JSON. | Consider emitting a one-line stderr warning when the payload is non-empty but `file_path` extraction fails, e.g. `printf 'check_mojibake.sh: could not extract file_path from payload; skipping.\n' >&2`. Or tighten the contract comment to say "malformed payload → silent exit 0 (fail-open)". Not blocking — fail-open on malformed payload is a defensible choice. |
| LOW | security / robustness | `[ ! -f "$file_path" ]` follows symlinks (POSIX `-f` returns true for symlink-to-regular-file). A malicious or accidental payload with `file_path` pointing to `/etc/passwd` would cause the hook to `grep -q` over it. Impact is minimal (read-only grep, no output on miss, no information leak because hook only exits 0/2), but the hook will do work on paths outside the repo root. | `.claude/hooks/check_mojibake.sh` L50. Probe: `ln -sf /etc/passwd /tmp/x; [ -f /tmp/x ]` → true. | Optional hardening: constrain scan to files under `$REPO_ROOT` by rejecting `$file_path` that does not start with `$REPO_ROOT` (or a user-specified allowlist of prefixes). Not a vulnerability today; log as a tech-debt note only if the hook is ever reused in multi-tenant contexts. |
| LOW | readability | `HOOK_REPO_ROOT` is documented in the header comment as "used by tests", which is accurate, but the name does not follow any existing hook convention (other hooks use HOOK_DIR only and derive REPO_ROOT relative to the script). The override is legitimate because the test creates synthetic repo trees, but the name is the only public contract for that indirection. | `.claude/hooks/check_mojibake.sh` L27-28, L33; `tests/test-check-mojibake.sh` uses it 7 times. | No change required. The `HOOK_REPO_ROOT` name is grep-able and the comment is adequate. Mention in PR description that this env var is test-only and should not be set in normal Claude Code sessions. |
| LOW | readability | `verify.local.sh` builds `$hook_scripts` as an unquoted space-separated string and intentionally disables `SC2086`. This works (all paths are internal globs, no shell metacharacters), but using a POSIX array-equivalent (positional parameters with `set --`) would be clearer and shellcheck-clean without a disable comment. | `scripts/verify.local.sh` L30-36. | Optional refactor: replace the loop + unquoted expansion with `set -- .claude/hooks/*.sh templates/base/.claude/hooks/*.sh scripts/verify.local.sh tests/test-check-mojibake.sh; for f do [ -f "$f" ] || continue; ...; done` + `run "shellcheck" shellcheck "$@"`. Not blocking. |
| LOW | maintainability | `scripts/check-sync.sh` adds `"tests/"` as a ROOT_ONLY prefix exclusion. This silently excludes any future test file from the sync check. That's correct for today (tests/ is repo-only), but a future contributor adding a test that should be distributed (e.g. a user-facing smoke test) would see it invisibly excluded. | `scripts/check-sync.sh` L38-39. | Not a change request. Consider leaving a comment above the entry pointing at `scripts/verify.local.sh`'s "not shipped to scaffolded projects" note so the rationale is discoverable when editing check-sync.sh. |
| LOW | maintainability | Plan mentions the AGENTS.md addition as "2 行注記" (two-line annotation) in the plan's acceptance criteria and implementation outline, but the actual addition is a single nested bullet. Not a bug — the plan drifted from implementation on wording only, and the one-line form fits AGENTS.md's "keep short" rule. | Plan L27, L56, L101 vs. AGENTS.md diff (1 line added under `.claude/hooks/`). | No change needed; flagged only to note that the progress checklist item "AGENTS.md repo map に 2 行注記" (line 72 of plan) is now satisfied by one line. |

No CRITICAL, HIGH, or MEDIUM findings. All diffs reviewed are internally consistent; the plan's acceptance criteria are reflected in the code; tests pass 11/11; sync-check passes; byte-for-byte mirror verified.

## Positive notes

- **Byte-for-byte mirror discipline**: `cmp` confirms both the hook script and the allowlist are identical between root and `templates/base/`, and git modes (100755 for scripts, 100644 for data file) match. `chmod +x` was not forgotten.
- **No U+FFFD literal in sources**: grep across all new files returned exit 1. The runtime-construction of `FFFD="$(printf '\357\277\275')"` (hook line 83) is correctly paired with the allowlist self-entry (`.claude/hooks/check_mojibake.sh`), giving belt-and-suspenders protection against self-detection.
- **Defense-in-depth for `$REPO_ROOT` globbing**: `case "$file_path" in "$REPO_ROOT"/*)` correctly quotes `$REPO_ROOT`, so even a REPO_ROOT containing glob metacharacters (e.g. `/tmp/[weird]/root`) is treated as literal (verified via probe). Unquoted `$normalised` in the allowlist loop is glob-expanded as intended, but `$(...)` inside an allowlist line is NOT command-substituted (POSIX case-pattern semantics), so malicious allowlist entries cannot execute code.
- **fail-open-with-warning is deliberate and documented**: The jq-missing path writes a marker to `.harness/state/mojibake-jq-missing` so later tooling can detect the degraded state. The rationale is in the hook's header comment (L18-21), matching Codex finding #2 from the planning advisory.
- **Commit slicing is coherent**: 4 commits, each mapping to a single concern (hook + tests / settings registration / AGENTS.md note / plan checkbox flip). This is exactly the pattern called for by `.claude/rules/git-commit-strategy.md` (commit after each passing slice).
- **Existing hook style preserved**: same shebang (`#!/usr/bin/env sh`), same `set -eu`, same `HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"` idiom as `post_edit_verify.sh` and `pre_bash_guard.sh`. Reviewer can scan hook sources without context-switch.
- **Fixture design is robust**: the `__FILE_PATH__` placeholder in `tests/fixtures/payloads/*.json` means schema changes in Claude Code (e.g. extra fields in `tool_input`) only need one fixture update, not a test rewrite. The Edit fixture deliberately includes an escaped `\"quotes\"` field — this exercises the sed-fallback edge case flagged in the planning risk register.
- **No debug code / no hardcoded secrets / no swallowed security-relevant errors**: searched for `eval`, `sh -c`, `bash -c`, `exec `, and dynamic execution — none. The only `2>/dev/null` suppressions are on non-security-relevant ops (directory creation in `.harness/state`, marker file touch, jq extraction).
- **Rollback plan is concrete**: plan section "Rollout or rollback notes" (L156-158) lists all 8 file-level actions needed to retire the hook. Low future cost.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `check_mojibake.sh` is a temporary mitigation for upstream Claude Code Issue #43746. Hook, allowlist, tests, fixtures, settings.json entry, AGENTS.md line, and check-sync.sh exclusions must all be removed when upstream ships a fix. | Carrying permanent hooks for fixed upstream bugs becomes dead weight; the hook scans every `Edit|Write|MultiEdit` on every session. | Upstream fix not released yet; interim detection is needed to keep the workflow correct. | Claude Code release that closes Issue #43746 and one week of observed non-recurrence locally. | `docs/plans/active/2026-04-17-mojibake-postedit-guard.md` |
| Hook silently passes on malformed JSON payloads (jq extract fails → empty file_path → exit 0). No log line, no marker file. If Claude Code ever emits a broken payload, the mojibake scan degrades without visibility. | Low (detection gap narrows but not eliminated; malformed payload is not the bug this hook targets). | Keeping the hook minimal and fail-open on parsing errors; adding a warning path adds conditional complexity. | A reported case where U+FFFD made it through because the payload was malformed. | `docs/reports/self-review-mojibake-postedit-guard.md` (this file) |

_The first entry should be appended to `docs/tech-debt/README.md`. The second is lower priority and can be tracked only inline in this report._

## Recommendation

- **Merge**: YES.
- **Blockers**: none (no CRITICAL or HIGH findings).
- **Follow-ups** (all LOW, none gate merge):
  1. Narrow `tests/test-check-mojibake.sh` cleanup to `$workdir`-only (remove the `rm -f "$REPO_ROOT/.harness/state/mojibake-jq-missing"` line). Small follow-up PR or amend in `/verify` cycle.
  2. Optionally add a one-line stderr warning when jq extraction fails on a non-empty payload (covers the silent-malformed-JSON path).
  3. Optionally add a comment in `scripts/check-sync.sh` cross-referencing `scripts/verify.local.sh`'s "not shipped to scaffolded projects" contract so the `"tests/"` exclusion rationale is self-evident.
  4. Append the "temporary mitigation" entry to `docs/tech-debt/README.md` so the retirement trigger is tracked alongside other debt items.

Hand off to `/verify` for spec-compliance + static-analysis (`./scripts/run-verify.sh` is reported green by the user; verifier should re-confirm all 13 acceptance criteria map to diff evidence).

## Re-review after Codex fixes (commit 306b23a)

- Date: 2026-04-17
- Reviewer: reviewer subagent (Claude) — 2nd pass
- Scope: commit 306b23a only (the Codex triage fix slice). Diff quality only.
- Commit contents (per `git show 306b23a`): 5 files, +102/-36.
  - `.claude/settings.json` + `templates/base/.claude/settings.json` (P3 ACTION_REQUIRED): `PostToolUseFailure` matcher `Bash|Edit|Write` → `Bash|Edit|Write|MultiEdit`.
  - `scripts/verify.local.sh` (P2 WORTH_CONSIDERING): `HARNESS_VERIFY_MODE` branch (`static`/`test`/`all`) + positional-parameter accumulation replacing the `SC2086`-disabled word-split builder.
  - `tests/test-check-mojibake.sh` (P1 hardening): Case E link set extended from `sh bash dash cat grep sed mkdir rm cd command pwd printf` to add `dirname env ln test`.
  - `docs/reports/codex-triage-mojibake-postedit-guard.md`: new triage artifact.

### Evidence gathered for the re-review

- `cmp .claude/settings.json templates/base/.claude/settings.json` → exit 0 (byte-for-byte identical after the Codex fix). `ls -la` shows both files are 3.4k and have the same mtime.
- `bash tests/test-check-mojibake.sh` → **11/11 PASS** (A, B, C, D, E, F.{edit,write,multiedit} × {clean,dirty}). No regression from the prior clean run.
- `HARNESS_VERIFY_MODE=static bash scripts/verify.local.sh` → runs shellcheck-skip, `sh -n` on 18 hook scripts, `jq -e` on 2 settings.json files, `check-sync.sh`. Does NOT run `tests/test-check-mojibake.sh`. Exit 0.
- `HARNESS_VERIFY_MODE=test bash scripts/verify.local.sh` → runs ONLY `tests/test-check-mojibake.sh`. Does NOT run `sh -n`, `jq -e`, or `check-sync.sh`. Exit 0.
- `HARNESS_VERIFY_MODE=all bash scripts/verify.local.sh` → runs all static checks, then hook tests. Exit 0.
- `HARNESS_VERIFY_MODE=bogus bash scripts/verify.local.sh` → emits `verify.local.sh: unknown HARNESS_VERIFY_MODE=bogus (expected static\|test\|all)` to stderr and exits 2 (the standard "misuse" code from `run()`-local `status=1` plus an explicit `exit 2`).
- Mutual-exclusivity probe (grep on labeled output): static-mode output contains `sh -n`, `jq -e`, `scripts/check-sync.sh` but NOT `tests/test-check-mojibake.sh`; test-mode output contains only `tests/test-check-mojibake.sh`. No overlap, no leak.
- `bash -n scripts/verify.local.sh` and `sh -n scripts/verify.local.sh` — both clean.
- `scripts/run-static-verify.sh` (`HARNESS_VERIFY_MODE=static exec ./scripts/run-verify.sh "$@"`) and `scripts/run-test.sh` (`HARNESS_VERIFY_MODE=test exec ./scripts/run-verify.sh "$@"`) correctly wire the new mode contract end-to-end. `run-verify.sh` already read and exported `HARNESS_VERIFY_MODE` (line 8-9), so the plumbing is honored without further changes.
- Link-set contamination probe for Case E: reproduced the `for tool in sh bash dash cat grep sed mkdir rm cd command pwd printf dirname env ln test; do ...` loop in a scratch directory and confirmed `jq` is NOT present in the resulting link set (ls of scratch dir contains no `jq` entry). The new additions `dirname env ln test` are all non-jq tools, so Case E still exercises the jq-missing branch as intended.
- HOOK_REPO_ROOT override continues to be honored in Case E (the test sets `HOOK_REPO_ROOT="$alt_root"` on line 101), so the marker write target remains `$alt_root/.harness/state/mojibake-jq-missing` — safely inside `$workdir` and cleaned up by the trap. The fix from 1321cd0 (don't delete the real repo's marker) is not regressed by 306b23a.
- The commit message claims `run-verify.sh all/static/test PASS` — I re-ran all three modes locally and confirm the claim.

### Findings from the Codex fix slice

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | In `verify.local.sh`, the order of internal static checks changed from (old) `1. shellcheck → 2. sh -n → 3. jq -e → 4. hook smoke tests → 5. check-sync` to (new) `static: 1. shellcheck → 2. sh -n → 3. jq -e → 4. check-sync` then `test: hook smoke tests`. The relative order of `check-sync` vs `hook smoke tests` is flipped in `all` mode (check-sync now runs before hook tests). This is a defensible reclassification (check-sync is static, hook tests are behavioral), and it is consistent with the documented mode split. However, the commit message does not call out the reorder. | `git show HEAD~1:scripts/verify.local.sh` vs `git show HEAD:scripts/verify.local.sh`, compared section numbers. | Not blocking. Consider a one-line note in the commit message or plan progress entry ("check-sync reclassified as static; now runs before hook tests in `all` mode") if someone diffs the ordering later. |
| LOW | accuracy of triage rationale | The DISMISSED entry in `docs/reports/codex-triage-mojibake-postedit-guard.md` states that "HOOK_REPO_ROOT override bypasses the dirname-dependent REPO_ROOT derivation in the hook." This is slightly off — HOOK_REPO_ROOT bypasses the *fallback* in `REPO_ROOT="${HOOK_REPO_ROOT:-$(cd "$HOOK_DIR/../.." && pwd)}"`, but the preceding line `HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"` still requires `dirname` unconditionally. So Codex's factual claim that "dirname is needed" is actually correct — Case E was only passing because `dirname` happened to be available via the shell builtin probe or via the bash interpreter's internal resolution. The P1 hardening (linking `dirname` explicitly) is a real fix, not pure defense-in-depth. The triage wording understates the value of the hardening. | `.claude/hooks/check_mojibake.sh` L35-36; `docs/reports/codex-triage-mojibake-postedit-guard.md` DISMISSED column. | Not blocking (the code change is correct; only the rationale wording is imprecise). Optional: reword the triage DISMISSED row from "false-positive" to "accepted as hardening" so future readers don't conclude that `dirname` was never required. |

Both findings are LOW. No CRITICAL/HIGH/MEDIUM.

### Positive notes (Codex fix slice)

- **Matcher symmetry restored**: `PostToolUse` and `PostToolUseFailure` matchers both now list `Bash|Edit|Write|MultiEdit`. The asymmetry introduced by the prior slice is fully resolved and both root+template files are byte-for-byte identical.
- **Positional-parameter refactor is cleaner**: `set --` followed by `set -- "$@" "$f"` inside the glob loop removes the need for `# shellcheck disable=SC2086` and the unquoted `$hook_scripts` word-split. It also handles the empty-glob case via `[ "$#" -gt 0 ]` gating the run call. This addresses LOW finding #5 from the prior pass *and* Codex P2 in one change.
- **Unknown-mode handling is strict and explicit**: `case "$mode" in static|test|all) ;; *) printf ... >&2; exit 2 ;; esac` rejects typos at the start of the script rather than silently running all checks. This is the right choice for a script that's invoked by orchestration (fail-fast on contract violations).
- **Mode contract matches the documented split**: `scripts/run-static-verify.sh` and `scripts/run-test.sh` wrappers were already setting `HARNESS_VERIFY_MODE`, and `run-verify.sh` was already exporting it. The P2 fix completes the end-to-end plumbing without touching the wrappers — it was one missing link in the chain, now filled.
- **Test still exercises the jq-missing branch**: I verified empirically that `jq` is NOT in the expanded link set (`dirname env ln test` are all non-`jq` tools), so the purpose of Case E (hook runs without jq, exits 0, writes marker) is preserved. The test's 11/11 PASS status confirms this.

### Updated recommendation

- **Merge**: YES.
- **Blockers**: none (no CRITICAL, HIGH, or MEDIUM in either the initial slice or the Codex fix slice).
- **Follow-ups** (non-blocking, LOW only):
  - Prior LOW follow-ups 1 (cleanup scope) was addressed in commit 1321cd0.
  - Prior LOW follow-up 5 (`$hook_scripts` word-split) was addressed in commit 306b23a via the positional-parameter refactor.
  - New LOW findings above are documentation-quality notes, not code defects.

The merge recommendation from the initial review stands, strengthened by the Codex fix slice: the PostToolUseFailure asymmetry is closed, the `HARNESS_VERIFY_MODE` contract is now implemented in the only verifier that honors a full mode split, and Case E is hardened with an explicit `dirname`/`env`/`ln`/`test` link set so we no longer depend on the test machine's shell-builtin resolution for `dirname`.

## Re-review after post_edit_verify fix (commit 29d71a2)

- Date: 2026-04-17
- Reviewer: reviewer subagent (Claude) — 3rd pass
- Scope: commit 29d71a2 only (Codex re-review P3-new fix). Diff quality only — spec compliance/tests are handled by /verify and /test.
- Commit contents (per `git show 29d71a2`): 5 files, +35/-9.
  - `.claude/hooks/lib_json.sh`: `extract_json_field` now interpolates `_field` into `jq -r ".${_field} // empty"` (supporting dotted paths); sed fallback now uses `_leaf="${_field##*.}"` so the leaf key is matched everywhere in the payload.
  - `.claude/hooks/post_edit_verify.sh`: call site `extract_json_field "$payload" "file_path"` → `"tool_input.file_path"` + a single-line contract comment.
  - Both mirrored into `templates/base/.claude/hooks/`.
  - `docs/reports/codex-triage-mojibake-postedit-guard.md`: added the re-review ACTION_REQUIRED row (documentation, not code).

### Evidence gathered for the re-review

- **Byte-for-byte mirror**: `cmp .claude/hooks/lib_json.sh templates/base/.claude/hooks/lib_json.sh` → exit 0; `cmp .claude/hooks/post_edit_verify.sh templates/base/.claude/hooks/post_edit_verify.sh` → exit 0. `git ls-files --stage` shows both file pairs share the same blob SHA (`5fb2678…` / `0325287…`) and mode `100755`. `chmod +x` was not dropped on either side.
- **Syntax**: `sh -n` and `bash -n` clean on both root and template copies of both files. `shellcheck` is not installed on the dev machine (confirmed via `command -v shellcheck` → 127); the pattern uses `sh`-legal constructs only (no bashisms: `${_field##*.}` is POSIX parameter expansion).
- **Smoke test (reproduced)**: `printf '{"tool_input":{"file_path":"/tmp/demo"}}' | ./.claude/hooks/post_edit_verify.sh` with clean state:
  - `.harness/state/edited-files.log` → contains `/tmp/demo` (previously always empty with jq installed).
  - `.harness/state/needs-verify` → created.
  - stdout: `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"Code file edited. Run ./scripts/run-verify.sh before claiming done. Save evidence to docs/evidence/."}}` — confirms the case-branch for code files fires (previously silent because `file_path` was empty and matched the `""` arm).
  - Exit 0.
  - Commit body claim reproduced.
- **All `extract_json_field` callers still work**:
  - `pre_bash_guard.sh` passes `"command"` (top-level identifier, no dot). jq path resolves as `jq -r ".command // empty"` → correct for top-level `{"command":"…"}`. Verified with `printf '{"command":"echo hi"}' | … extract_json_field "$(cat)" "command"` → `echo hi`.
  - `post_edit_verify.sh` passes `"tool_input.file_path"`. jq path resolves as `jq -r ".tool_input.file_path // empty"` → correct. sed fallback uses leaf `file_path` and matches the nested occurrence. Verified both paths.
  - Legacy top-level `"file_path"` still extracts (verified even though no caller uses it today).
  - Missing nested field returns empty (not an error).
  - Payloads with escaped `\"` inside string values parse correctly under jq (e.g. `{"tool_input":{"file_path":"a\"b.py"}}` → `a"b.py`).
- **sed fallback, jq-absent (simulated via `PATH` shim with only `sh`, `sed`, `cat`)**: all four caller shapes (top-level `command`, dotted `tool_input.file_path`, legacy top-level `file_path`, missing-field) return the expected values. The fallback's "first-occurrence-of-leaf-key" behavior is now an *intended* contract (documented in the new comment block) rather than an accidental side effect — the commit body acknowledges this honestly (`the sed fallback already accidentally matches nested`).
- **No regression in the mojibake test suite**: `bash tests/test-check-mojibake.sh` → **11/11 PASS** (A, B, C, D, E, F.{edit,write,multiedit} × {clean,dirty}). `check_mojibake.sh` does not source `lib_json.sh` (it calls `jq -r '.tool_input.file_path // empty'` directly at L47), so the hook under test is not touched by this commit.
- **Grep for literal U+FFFD in the diff**: `git show 29d71a2 | grep -P '\xef\xbf\xbd'` → no matches.
- **No secrets, debug prints, commented-out code, or TODO markers in the diff**.

### Findings from the post_edit_verify fix slice

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security / contract | `jq -r ".${_field} // empty"` interpolates `_field` directly into the jq expression. All current callers pass hardcoded literals (`"command"`, `"tool_input.file_path"`), so today this is safe. But the header comment does not warn future maintainers: a caller that ever passes an attacker-controlled or user-derived string (e.g. a field name extracted from a payload, or a CLI argument) gets jq-expression injection. Probe: `extract_json_field '{"file_path":"safe"}' 'file_path // "injected" \| . , "extra"'` → outputs both `safe` and `extra` (side-channel leak). The sed fallback is comparatively safer because `_leaf` is only interpolated into a regex literal, though metacharacters in `_leaf` (`.`, `*`, etc.) would need escaping for correctness. | `.claude/hooks/lib_json.sh` L18 (jq path), L20-21 (sed path). | Non-blocking (no caller violates the contract today). Add one line to the header comment: `# NOTE: _field is interpolated verbatim into the jq expression. Callers MUST pass a trusted literal, not user input.` This upgrades the implicit "internal callers only" contract to an explicit one. |
| LOW | compatibility | The old form `jq -r ".[\"$_field\"] // empty"` worked for any top-level key name, including names with hyphens, starting with digits, or containing shell-unsafe characters. The new form `jq -r ".${_field} // empty"` only works for identifier-shaped names (`[A-Za-z_][A-Za-z0-9_]*` separated by `.`). Today this is fine — `"command"`, `"tool_input"`, `"file_path"` are all identifiers. But the commit body's claim "behaviourally unchanged for non-dotted callers" is slightly overstated: a hypothetical caller passing `"a-b"` would have succeeded under the old code and silently returns empty under the new code. | Probe: `printf '{"a-b":"x"}' \| jq -r '.a-b // empty' 2>&1` → empty (silent parse error swallowed by `2>/dev/null`); the old `jq -r '.["a-b"] // empty'` would return `x`. | Non-blocking. Either (a) document the restriction in the header comment ("field names must match the jq identifier grammar"), or (b) wrap each segment with bracket-quoting at call time: `jq -r "[\"${_field//./\"][\"}\"] // empty"` — too clever for an sh function. Preference: option (a) — a single comment line, since no real caller is affected. |
| LOW | robustness | An empty `_field` causes `jq -r ". // empty"` to return the entire payload JSON. None of our callers pass an empty string, but defense-in-depth would be a leading guard: `[ -z "$_field" ] && return 0`. | Probe: `extract_json_field '{"file_path":"x"}' ""` → full JSON object. | Non-blocking (no caller regresses). Optional one-liner guard at the top of the function. |

No CRITICAL, HIGH, or MEDIUM findings. The fix is minimal, targeted, and correctly mirrored.

### Positive notes (post_edit_verify fix slice)

- **Correctness**: the commit actually repairs a real silent-no-op that predates the mojibake PR. The self-review would not have caught this during the initial slice because `post_edit_verify.sh` is pre-existing and was out-of-scope; it only surfaced because the MultiEdit matcher extension drew Codex's attention to the failure path. Good cross-model ROI.
- **Documentation inline with the code**: the new block comment in `lib_json.sh` (L4-12 post-fix) explains both the dotted-path contract and the sed fallback's ambiguity limitation. Future maintainers touching this function do not need to spelunk git blame.
- **Honesty about the sed path**: the "happens to work" comment (L10-12) is the right level of candor. A naive "also supports nested" would be misleading; the current wording correctly signals "upgrade to jq for correctness".
- **Mirror discipline held**: both files updated in both roots; `cmp` and `git ls-files --stage` confirm byte- and mode-identity. No `chmod` regression.
- **Commit message is precise**: the body correctly distinguishes jq (always broken top-level-only match) vs sed fallback (accidentally matched nested via first-occurrence heuristic). The root-cause explanation matches the code.
- **No collateral damage to `check_mojibake.sh`**: the mojibake hook does not source `lib_json.sh` (it calls `jq` directly), so this refactor cannot regress the mojibake scan.
- **Test suite unaffected**: `bash tests/test-check-mojibake.sh` still 11/11 PASS. The change is orthogonal to what those tests exercise.

### Updated recommendation

- **Merge**: YES.
- **Blockers**: none (no CRITICAL, HIGH, or MEDIUM in any slice reviewed so far — initial, 306b23a Codex fix, or 29d71a2 post_edit_verify fix).
- **Follow-ups** (non-blocking, LOW only):
  - Optional: add a one-line header-comment warning in `lib_json.sh` about the internal-callers-only contract for `_field` (finding 1).
  - Optional: document (or defensively guard) the identifier-shape restriction on `_field` (finding 2) and the empty-string case (finding 3).

Merge recommendation for the full branch (chore/mojibake-postedit-guard) remains **YES**. The 29d71a2 fix converts a previously silent-no-op hook into a working one without introducing any new defects; the only observations are documentation-quality and apply to future, hypothetical callers rather than current code.
