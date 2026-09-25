# Review report: secret-scan-git-config (self-review, cycle 1)

- Date: 2026-09-25
- Plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD` at e0acc4e (code commits 014ab77 slice A, 3060815 slice B, a8adfc7 slice C; plan commits 7da850a, 0904a25, e0acc4e), plus full reads of `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh`. No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- `git diff main...HEAD`: 7 files, +892/-77. Worktree HEAD e0acc4e, `git status --porcelain` empty at review start. `cmp` confirms `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` are byte-identical to their `templates/base/scripts/` copies.
- Callers read to check how each one handles exit 3: `scripts/pre-commit-secret-guard.sh`, `scripts/pre-merge-commit-secret-guard.sh`, `scripts/prepare-commit-msg-secret-guard.sh`, `scripts/commit-msg-guard.sh`, `scripts/run-verify.sh:207-254`, `.github/workflows/verify.yml:11-14`.
- Suites run as requested: `sh tests/test-secret-scan.sh` passed 50/50 and `sh tests/test-secret-scan-branch.sh` passed 108/108. The same two suites also passed (50/50, 108/108) on a scratch copy of `scripts/` and `tests/` in which both scanners' shebangs were changed to `/bin/dash`.
- Product run on this branch: `./scripts/secret-scan-branch.sh --strict` printed `scanned c1785b8..e0acc4e against origin/main: clean` (rc 0). `git log main..HEAD --format=%B | ./scripts/secret-scan.sh --stdin` returned rc 0.
- Mutation spot-checks. All were run on the scratch copy, and the worktree was never modified. Each mutation was checked for one changed line before running.

  | Mutation | Tests that went red |
  | --- | --- |
  | drop `--diff-algorithm=default` | AC-5 histogram and patience |
  | drop `-c diff.renames=true` | AC-4 copies and AC-4 `diff.renames=false` |
  | `--diff-filter=d` changed to `ACMR` | type-change test |
  | drop the all-zero-id skip | unmerged test (so the fixture really is unmerged) |
  | drop `--no-abbrev` | AC-9 label assertion |
  | drop the ANSI `gsub` | AC-10 |
  | drop `--no-replace-objects` from staged | replace-HEAD test |
  | drop `diff.relative=false` from staged | AC-8 |
  | ignore `log_rc` | four AC-15 assertions |
  | branch: drop the sentinel | both AC-11 assertions |
  | branch: drop `core.quotePath=false` | all three AC-12 assertions |

  `--no-show-signature` and `--no-color` have no test that fails on their own, as the plan's Deviation notes already say.
- Probes in fixture repos under the scratchpad, each run under an isolated HOME, `GIT_CONFIG_GLOBAL`, and `GIT_CONFIG_SYSTEM`:
  - `diff.renameLimit=1` with three renamed-and-edited files: the default config gives rc 0 and `renameLimit=1` gives rc 1, along with git's "exhaustive rename detection was skipped" warning.
  - A local `diff.scrub.binary=true` for a committed `*.txt diff=scrub`: default rc 1, with the setting rc 0.
  - `--file` on a missing path gives rc 0. `--file` on a `chmod 000` file that holds a token gives rc 0, and the same file readable gives rc 1.
  - SIGTERM to `secret-scan.sh --file` 1 s into scanning a 400-line file of tokens gives rc 0. Unsignalled, the same run gives rc 1.
  - An added line whose content starts with `++ ` followed by a token: `--range` rc 0.
  - `git -c color.ui=always -c color.diff=always diff --cached --raw` prints no escape bytes.
  - An intent-to-add file does not appear in `diff --cached --raw`, and `--staged` gives rc 0.
  - With a missing `diff.orderFile`, `diff --cached` exits 128, so `--staged` exits 3.
  - Symlink `.gitallowed` targets `rulesx` and `x` both resolve (the scan prints `clean`, which means the rule was used). An empty symlink blob gets the could-not-read notice.
  - Timing of `--staged` with 300 staged files: 26.7 s before the change (main) and 25.8 s after.
- Portability probes:
  - The awk `gsub(/\033\[[0-9;:]*m/, "")` strips the sequence under macOS BWK awk 20200816, mawk (`ubuntu:24.04`, CI's awk), and busybox awk (`redis:7-alpine`).
  - git 2.8.6 (`alpine:3.4`, amd64): the full `scan_range` invocation fails with `fatal: unrecognized argument: --no-show-signature` (rc 128). Replacing that flag with `-c log.showSignature=false` makes the same invocation succeed. The staged listing invocation succeeds as written.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | `scripts/secret-scan.sh:182-190` | `diff.renameLimit` is not pinned, so the local range scan and CI still disagree for renamed files. This is the same class as AC-4: rename and copy detection decide whether content already on the base shows up again as added lines. The plan's Design decisions call this kind of disagreement wrong in both directions. A low local limit skips inexact rename detection, so renamed-and-edited files show as delete plus add, and a branch that CI passes is blocked locally (verified). A high local limit, which git's own warning tells users to set, detects renames that CI's default of 1000 skips once a branch has more than 1000 candidates, so the local scan misses what CI blocks. The plan lists `diff.renameLimit` as an unverified out-of-scope item. It is now reproduced. | Fixture with three renamed-and-edited files whose unchanged lines hold a token already on main: default rc 0, `diff.renameLimit=1` rc 1. `git help config`: "If not set, the default value is currently 1000." | Add `-c diff.renameLimit=1000` next to `-c diff.renames=true`, in the same style as `core.bigFileThreshold=512m`. Add a comment line (`diff.renameLimit=1000  git's default rename-candidate limit`) and one test built from the fixture above (`diff.renameLimit=1` gives the same result as the default config). If this is deferred instead, the tech-debt row should say the divergence is reproduced, with this fixture, not "未確認". |
| LOW | `scripts/secret-scan.sh:4-10` | The header says `scan_range` neutralizes "every local git setting that changes which added lines `git log -p` prints". Two local config settings are not neutralized: `diff.renameLimit` (above) and `diff.<driver>.binary=true` for a driver named in the committed `.gitattributes`. The plan's Deviation notes already list both as out of scope. A reader who trusts "every" will dismiss a report of divergence. | `diff.scrub.binary=true` fixture: rc 0 where the default config (and therefore CI) gives rc 1. | Change "every local git setting" to "the local git settings listed at scan_range". Add one sentence saying the known exceptions (a local `diff.<driver>.binary`, `.git/info/attributes`) are tracked in `docs/tech-debt/README.md`. |
| LOW | `scripts/secret-scan.sh:189`, `:174` | `--no-show-signature` requires git 2.10 or later. On an older git, every `--range` call fails with `unrecognized argument`, which blocks CI's secret-scan step, the prepare-commit-msg hook, `run-verify.sh`, and the `/pr` gate for everyone on that git. It fails closed, but every run fails. The plan's own Risks row says to prefer `-c` overrides because git ignores config keys it does not know. | git 2.8.6: the invocation as written gives rc 128. With `-c log.showSignature=false` in place of the flag, it succeeds and prints the added line. All other pinned options work on 2.8.6. | Replace `--no-show-signature` with `-c log.showSignature=false` in the `-c` block, and move its comment line into the `-c` group. |
| LOW | `scripts/secret-scan.sh:15-18`, `:39`, `:78` | The new header documents exit 0 as "scanned, nothing found". Two pre-existing paths return 0 without having scanned. (a) `--file` on a missing or unreadable path: `[ -s ]` treats a missing file as empty, and `grep`'s exit status 2 is lost in the `grep \| while` pipe. `commit-msg-guard.sh` checks `-f` first, so today's hook does not reach this. (b) `trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM`: POSIX sh resumes after a signal trap, so the scan continues without its temp dir and ends with an empty findings file. `secret-scan-branch.sh:70-78` already explains this pitfall and fixes it. | `--file` missing path: rc 0. `chmod 000` file holding a token: rc 0 (1 when readable). SIGTERM mid-scan on a file full of tokens: rc 0 (1 unsignalled). | Copy the sibling's traps: `trap 'rm -rf "$tmp_dir"' EXIT` plus `trap 'exit 129' HUP`, `trap 'exit 130' INT`, `trap 'exit 143' TERM`. In `--file`, exit 3 with a one-line reason unless `[ -f "$2" ] && [ -r "$2" ]`. Alternatively, narrow the header's exit-0 line. Either fix is small, or both can go to tech-debt, since the behavior is not new. |
| LOW | `scripts/secret-scan.sh:125-136` | The all-zero check `*[!0]*) ;; *) continue ;;` also skips a line whose `new_id` is empty, meaning a raw line that did not parse. This is the same silent skip that the plan's Design decisions name as the reason the non-ASCII miss stayed hidden. I found no git output that reaches it; `--raw --no-abbrev` always has five meta fields. | Reading of the `case`. An empty string does not match `*[!0]*`. | Add an explicit arm, `"") printf 'secret-scan: could not scan the staged changes: cannot parse %s\n' "$meta" >&2; exit 3 ;;`, ahead of the zero check. |
| LOW | `scripts/secret-scan.sh:155` (pre-existing), `:154` | `/^\+\+\+ / { next }` also drops an added line whose content starts with `++ `, in CI as well as locally. The new `gsub` runs first, so content that starts with an SGR sequence followed by `++ ` is now dropped too. Before this change, that line was scanned. CI and the local scan run the same awk, so their results do not diverge, but this is the same "which added lines are read" class the PR is about. | An added line `++ x <token>` on a branch: `--range` gives rc 0. | Record it in tech-debt with the two Non-goals rows. The fix is to skip `+++ ` only in a file header, from `diff --git` (or `--- `) up to the first `@@`, not in every line. |
| LOW | `scripts/secret-scan.sh:99-106` vs `:110-112`; `scripts/secret-scan-branch.sh:20-28` | Two comments do not match the code. (a) The option list for `scan_staged` omits `--raw`, `--no-abbrev`, and `--no-color`, which the invocation passes. `--no-color` does nothing for `--raw`, and `--no-abbrev` is what keeps the error message's blob id full length (see the mutation). (b) The exit-code list for `secret-scan-branch.sh` still says `3  (--strict only)`, but the paragraph right after it says default mode also exits 3 when the scanner exits 3. "could not scan" now means two different things: in this script it is exit 0 in default mode, and in `secret-scan.sh` it is exit 3 in both modes. | Reading of both headers. The `color.ui=always` probe of `--raw` printed no escapes. | (a) Drop `--no-color` from the staged invocation, or list it. Add `--no-abbrev  full ids, as cat-file and the error message use them`. (b) Rewrite the 3 line as: `3  --strict: could not determine what to scan, or nothing to scan; either mode: the scanner could not read the range (its own exit 3, propagated)`. |

No CRITICAL or HIGH findings.

## Positive notes

- Moving from a pipe to a temp file plus a `log_rc` check is the right fix for the fail-open pipe, and it covers a failure both before and part way through the output. The partial-failure fixture, which deletes the older commit's loose blob, tests the harder of the two cases.
- The `scan_range` comment lists its 12 pins in the same order as the invocation, one line each, and each line names the setting it neutralizes. Nine of the twelve have a test that turns red when the pin is removed.
- Reading staged content by blob id with `--no-replace-objects` removes path quoting from the problem instead of working around it. It also closes the type-change gap that the old `ACMR` filter had, and the replace-HEAD fixture is a thoughtful test of that.
- The sentinel read is correct at every edge I probed: a failure short-circuits `&&` and yields `""`; partial output on failure is thrown away; an empty blob gets the notice; targets `rulesx` and `x` keep their trailing `x`.
- Test fixtures build every value at runtime from split pieces. Test names state what is protected (for example "pure rename with diff.renames=false matches the default result"). `expect_exit` prints the exit code it got, the one it wanted, and both output streams. The new section is hermetic and leaves the older non-hermetic cases untouched, as the plan says.
- Each hook reacts to exit 3 by failing closed: the three hook wrappers pass the non-zero status to git, and `run-verify.sh` sets `status=1` and prints `Branch secret scan failed`.

## Coverage gaps

- Not probed: a symlink target that contains a NUL byte. Command substitution handles NUL differently in bash and dash, and git refuses to check such a target out. Such a blob can only be made by crafting it, and the allowlist is controlled by whoever commits anyway.
- The hermetic section does not unset `XDG_CONFIG_HOME`, so fixture commands still read `$XDG_CONFIG_HOME/git/attributes`. No new assertion depends on it, although a user-level `merge=union` would make the unmerged fixture merge cleanly and pass vacuously. It also does not set `GIT_CONFIG_NOSYSTEM=1`, and `GIT_CONFIG_SYSTEM` needs git 2.32 or later. Both are optional hardening.
- `--staged` costs about 87 ms per file (300 files in 26 s). The nine grep pipelines in `scan_file` account for most of it, the same as before the change, so this is not a regression. A large staged import is slow, as it already was.
- Default mode of `secret-scan-branch.sh` can now exit 3 through `run-verify.sh` when git cannot read the range, for example a blobless partial clone while offline. This is consistent with failing closed and with the existing "scanner failed" path, but it is an exception to the default mode's "never fails on environment gaps" framing. `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` is still available.

## Answers to the focus points

1. **Fail-closed.**
   - `--range`: a git failure before or during output, and a failed write of the log file (git's own stdout write check), both end in 3.
   - `--staged`: a failed listing or an unreadable blob ends in 3.
   - `--diff`/`--stdin`: no git involved.
   - Paths that still end in 0 without scanning: `--file` on a missing or unreadable path, and a HUP/INT/TERM during the scan (LOW, both pre-existing), plus the theoretical unparsed staged line (LOW).
   - `secret-scan-branch.sh`: 0 only after the scanner returns 0, or through the documented `cannot_scan`/`nothing_to_scan` calls in default mode. `--strict` turns those into 3.
   - The hooks and `run-verify.sh` treat 3 as a block (checked, no issue beyond the header wording in LOW-6).
2. **Over-blocking.** No over-block found in the diff:
   - `diff.renames=true`, `--submodule=short`, `log.showRoot=true`, `core.bigFileThreshold=512m`, and `--diff-algorithm=default` all equal git's defaults, and tests confirm them for rename, submodule, and algorithm.
   - The ANSI strip can change content only when committed content contains a literal SGR sequence. CI runs the same awk, so their results cannot diverge. The strip only joins text, so it cannot hide a real token. Its one side effect is the `++ ` case in LOW-5.
   - `--diff-filter=d` adds type changes, and `git log -p` shows those as delete plus add in CI too. Unmerged entries are skipped.
   - The one verified over-block is outside the diff: an unpinned `diff.renameLimit` (MEDIUM).
3. **Staged parsing.**
   - A leading space survives, because IFS is only a tab and the path is only a label.
   - A mode-only change scans the whole blob (as `M` did before).
   - A symlink scans its target string (as before).
   - An unmerged entry is `U` with a zero id and is skipped. The mutation proves the fixture is really unmerged.
   - A rename under `--no-renames` becomes D (filtered) plus A (scanned, labelled with the new name).
   - Intent-to-add entries do not appear at all.
   - One `cat-file` per file adds nothing measurable (26.7 s before, 25.8 s after, for 300 files). Checked, no issue apart from LOW-4.
4. **Sentinel.** A failure yields `""` and the notice, and partial output is thrown away. An empty blob yields `""` and the notice. A target ending in `x` is handled by `%x`, which removes exactly one character (probed `rulesx` and `x`). Checked, no issue.
5. **Portability.**
   - dash: both suites pass.
   - macOS `/bin/sh` (bash 3.2): the default run.
   - awk `\033` in the ERE works in BWK awk, mawk (CI), and busybox awk.
   - git: every pinned option works on 2.8.6 except `--no-show-signature`, which needs 2.10 or later (LOW-2). `-c` keys that git does not know are ignored (`diff.relative` needs 2.28 or later).
6. **Tests.** The section is hermetic for HOME and global/system config (XDG and git older than 2.32 are optional hardening, see Coverage gaps). The branch scan and the commit-message scan found no secret-looking literal. The names say what they protect, and a failing test prints the exit code it got, the one it wanted, and both output streams. Mutations back the red/green claims, with the two exceptions the plan already declares.
7. **Comments.** The `scan_range` option list matches the invocation exactly. The header claims "every" setting (LOW-1). The `scan_staged` list is missing three flags, and the exit-3 line in the branch header is out of date (LOW-6). Otherwise the comments describe the code rather than its history, and the only IDs are plan ACs and `#176`.

## Recommendation

**Pass**: no CRITICAL or HIGH findings, so the branch can go to `/verify`. Before `/pr`:

- MEDIUM-1: fix it now (one `-c` line and one test), or defer it with a tech-debt row that records the reproduction.
- LOW-2: fix now. It is one token.
- LOW-1 and LOW-6: comment wording. Fix these in the same pass.
- LOW-3, LOW-4, LOW-5: these fit the batch of tech-debt rows the plan already plans for out-of-scope findings, or they can be fixed cheaply now.
