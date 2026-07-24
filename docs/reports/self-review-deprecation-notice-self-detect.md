# Self-review report: deprecation-notice-self-detect

- Date: 2026-07-24
- Plan: docs/plans/active/2026-07-24-deprecation-notice-self-detect.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff ef12eaae...HEAD` — diff quality only (naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability). No spec compliance, tests, static analysis, or doc drift.

## Evidence reviewed

- `scripts/ralph` deprecation-notice block (lines 647–652)
- `templates/base/scripts/ralph` (byte-identical mirror; `cmp` exit 0 confirmed)
- `tests/test-ralph-deprecation-notice.sh` (new case 5, lines 115–133)
- Sibling-source mechanism: `SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"` at lines 13–15 — PATH-independent, so test case 5's PATH override does not break `ralph-config.sh`/`ralph-common.sh` sourcing.
- Behavioral probes (in a scratch dir, not the repo):
  - `set -euo pipefail` + `[ -n "$x" ] && ! [ "$x" -ef "$0" ]`: a false `-ef` result does not abort the script (compound is a tested condition, exit stays 0). Confirmed.
  - Under `PATH="$PWD/scripts:/usr/bin:/bin"`, `command -v ralph` resolves to the real `scripts/ralph`, which `-ef "$0"` matches → notice correctly suppressed. Confirmed.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | naming | Local temporaries `_ralph_resolved` use the underscore-prefix convention consistent with the rest of the script (`_plan`, `_dry_run`, etc.); grep-able and clear. No change needed — noted as confirmation. | `scripts/ralph:648` | None. |
| LOW | maintainability | The `-ef` self-exclusion relies on `$0` holding a real filesystem path. If the script were ever invoked in a way where `$0` is not a path to the file (e.g. sourced, or via a shell that sets `$0` to the shell name), `-ef` would fail the comparison and the notice could mis-fire. This is out of the documented usage contract (the script is an executable entrypoint, never sourced — plan Assumptions + Risks cover it), so it is acceptable, but the assumption is implicit in the code. | `scripts/ralph:649`; plan lines 34–35, 97 | Optional: a one-line comment above the `if` noting "`-ef` compares inode identity to exclude self when scripts/ is on PATH" would make the intent self-documenting for future readers. Non-blocking. |

No CRITICAL, HIGH, or MEDIUM findings.

### Checklist notes (no findings)

- **Unnecessary changes**: Diff is minimal and on-topic. The 3-line notice block became a 5-line guarded block; the two docs commits only touch the plan file. No stray formatting or unrelated edits.
- **Typos / copy-paste**: Notice message text is unchanged (byte-identical to prior). The two script files are byte-identical (`cmp` exit 0), satisfying the mirror-discipline contract. Test labels are accurate and specific.
- **Null safety**: `command -v ralph 2>/dev/null || true` cannot fail the `set -e` pipeline; `[ -n "$_ralph_resolved" ]` guards the empty case before `-ef` is evaluated, so `-ef` never runs against an empty operand.
- **Debug code / secrets**: None introduced.
- **Exception handling**: `|| true` deliberately neutralizes `command -v` non-zero exit under `set -e`; this is the correct idiom here, not a swallowed error (the empty-string branch is handled explicitly).
- **Security**: No injection surface. `_ralph_resolved` is only used as a `test -ef` operand (no eval, no unquoted expansion into a command). Values are quoted throughout.
- **Readability**: Nesting is 2 levels, block is ~6 lines. The `[ -n ... ] && ! [ ... -ef ... ]` compound is idiomatic POSIX-ish test usage and reads clearly.

## Positive notes

- The `-ef` (same-inode) approach is the right call over `realpath`/`readlink -f` string normalization: it sidesteps macOS bash 3.2 vs GNU differences and handles symlink-to-self transparently. The plan's rationale (lines 48, 96) matches the code.
- Test case 5 is well-constructed: it does not just assert notice-absence (which a broken early-exit could satisfy as a false green). It adds a positive assertion (`=== Ralph Pipeline Status ===` present) and a negative assertion (no `No such file or directory`) to prove the script actually started and sourced its siblings. This directly addresses the Codex plan-advisory HIGH finding about single-file-copy false greens. Strong regression coverage.
- Mirror discipline maintained: `scripts/ralph` and `templates/base/scripts/ralph` are byte-identical, both mode 100755.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (none) | — | — | — | — |

No new tech debt. The legacy shell CLI retirement itself remains tracked in `docs/tech-debt/README.md` (pre-existing, explicitly a non-goal here).

## Recommendation

- Merge: Yes (no blocking findings). Two LOW notes are optional polish; the second (an explanatory comment) is a reasonable follow-up but not required.
- Follow-ups: Optionally add a one-line intent comment above the notice guard. Proceed to `/verify`.
