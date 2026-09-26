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

## Cycle 2

- Date: 2026-09-25
- Plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff f6a6c82...HEAD` at 45ce71f, read in full with the whole branch as context. The range contains the code commits aecee05 (Slice D, fixes for this report's cycle-1 findings) and ca8a212 (Slice E, cross-review AR-1), and the documentation commits 95a02f9, b76d9da, 237dc14, f6d1b87, f413a58, and 45ce71f. Cycle-1 findings are re-opened only where they came back or got worse. This is cycle 2 of 2, so any LOW left unfixed here becomes a deferral.

### Evidence reviewed

- Starting state: worktree HEAD 45ce71f, `git status --porcelain` empty. Both scanners are still `cmp`-identical to their `templates/base/scripts/` copies. The DAG is linear: ca8a212's parent is f413a58, and aecee05's parent is f6a6c82.
- Test suites: `sh tests/test-secret-scan.sh` passes 69/69 and `sh tests/test-secret-scan-branch.sh` passes 108/108. Both also pass on a scratch copy with both scanners run under `/bin/dash`. The TERM test passed in 3 of 3 repeated runs.
- **State machine against a hunk-counting oracle.** The oracle is a small Python parser that reads `@@ -a,b +c,d @@` and counts lines down each hunk; `\` lines do not count. I compared its output with three awk programs taken byte for byte from `git show <rev>:scripts/secret-scan.sh`: the new awk (45ce71f), main's (c1785b8), and cycle 1's (f6a6c82).
  - Shape fixture, a five-commit `main..feature` range. It covers:
    - several files in one commit;
    - `commit <sha>` lines between commits, and an empty commit;
    - a binary file (`Binary files ... differ`);
    - a mode-only change;
    - a pure rename, and a rename with an edit;
    - a new empty file;
    - `\ No newline at end of file`;
    - a submodule pointer change under `--submodule=short`;
    - CRLF content;
    - a deletion;
    - a file with two hunks;
    - content lines that look like headers: `++ `, `+++ `, `diff --git`, `@@`, `--- `, `-- `, and one that starts with a literal ESC `[32m`;
    - file names that look like headers or tokens: a token-shaped name, `@@x`, `diff y`, `+plus.txt`.

    Results:
    - new awk: identical to the oracle (24 lines).
    - main: drops exactly the three `++ ` content lines.
    - cycle 1: also removes the content's own escape sequence.
    - `--color=always` output of the same range: after removing SGR codes from the awk's output, it equals the oracle.
    - The real scanner gives rc 0 on this range and on its colored log: the token-shaped name is never reported.
  - This repo's full history at HEAD (1,604 commits, 24.7 MB of `git log -p` with `scan_range`'s pins): main awk, new awk, and the oracle each extract 215,825 lines, and all three outputs are byte-identical.
- **awk portability of `while (sub(/^\033\[[0-9;:]*m/, "")) {}`.** I ran the program on the plain and colored shape logs, and on a file with several SGR codes in front of one line, under:
  - BWK awk 20200816 (macOS),
  - mawk (`ubuntu:24.04`, CI's awk),
  - busybox awk (`redis:7-alpine`),
  - gawk (installed in the same image).

  All four produce identical bytes.
- **git 2.8.6** (`alpine:3.4`, busybox sh and awk): the HEAD scanner's `--range` gives rc 1 with both findings (a plain token line and a `++ ` line), also rc 1 with 2 findings under `color.ui=always`. `--staged` gives rc 1, `--file` on a missing path gives rc 3, and `--diff` on a `++ ` hunk line gives rc 1. The header's claim "every option used works on git 2.8" holds.
- **The `diff.renameLimit=1000 (git's default since 2.33)` comment:** with 500 renamed-and-edited files, git 2.32.7 (`alpine:3.14`) skips inexact rename detection (0 renames, plus the warning), and git 2.34.8 (`alpine:3.15`) detects all 500. This is consistent with a default of 400 before 2.33 and 1000 from 2.33.
- Mutations, each run on a scratch copy and checked for its changed-line count before running. The worktree was never modified.

  | Mutation | Result |
  | --- | --- |
  | drop the `diff ` reset | both token-name tests red |
  | skip every `+++ ` line again | both `++ ` tests red |
  | never enter a hunk | both `++ ` tests red |
  | drop the leading-SGR strip | AC-10 and the colored-hunk test red |
  | cycle-1 single-line trap | TERM test red, exit 0 where 143 is wanted |
  | drop `-c diff.renameLimit=1000` | the renameLimit test red |
  | drop the `--file` check | three tests red |
  | drop `is_object_id` | three tests red |
  | `while (sub(...))` changed to a single `sub` | nothing red (see Coverage gaps) |
  | drop `-c log.showSignature=false` | nothing red (already recorded in tech-debt) |

- Probes of the tech-debt rows:
  - An uncommitted working-tree `.gitattributes` holding `*.txt -diff` turns a range that gives rc 1 into rc 0. Setting `GIT_ATTR_SOURCE=HEAD` on the same run brings it back to rc 1.
  - `diff.external` and `GIT_EXTERNAL_DIFF` are not invoked by `git log -p` without `--ext-diff`, which confirms the row's "structural no-op" claim.
  - `--diff` on a unified diff with two files and no `diff ` line between them (plain `diff -u` concatenation, or svn-style `Index:` sections) reports the second file's token-shaped name (rc 1).

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | `docs/quality/quality-gates.md:46`, `templates/base/docs/quality/quality-gates.md:46` | Cycle-1 LOW-1 has come back on a new, shipped surface. The script header now correctly lists the local-only gaps it cannot pin. But `/sync-docs` added "so a local run scans the same added lines as CI **regardless of local git config**" to both copies of the quality gates, and one of those gaps, a local `diff.<driver>.binary=true`, is itself a local git config setting. The `templates/base/` copy ships to scaffolded projects. `docs/reports/sync-docs-2026-09-25-secret-scan-git-config.md:14` repeats the claim. | Cycle-1 probe: `diff.scrub.binary=true` gives rc 0 where CI gives rc 1. `scripts/secret-scan.sh:10-13` names the same gap. | Say "regardless of the local git settings it pins (the scanner header lists the known exceptions)" in both copies. |
| LOW | `scripts/secret-scan.sh:10-13`; `docs/tech-debt/README.md:139` | The header's list of "Known local-only gaps it cannot pin" names two gaps. A third one reproduces: an uncommitted working-tree `.gitattributes` with `*.txt -diff` makes the range scan clean where CI's checkout of HEAD finds the token. It can also be pinned: `GIT_ATTR_SOURCE=HEAD` reads attributes from HEAD's tree, which is what CI's worktree holds. The tech-debt row covering it has four inaccuracies. It says the item is "plausible mechanism, unverified this cycle". Its Justification says "Reproduced during #176's investigation". It opens with "none pinnable by `-c` or a command-line option". Its own fourth item says "closable with `GIT_NO_REPLACE_OBJECTS=1`", which is the env form of the same `--no-replace-objects` option `scan_range` already passes. Impact is limited: `/pr` step 2 commits before the strict scan runs, so the gap mainly affects the default-mode `run-verify.sh` run. | Fixture: rc 1 at baseline, rc 0 with the uncommitted `.gitattributes`, rc 1 again with `GIT_ATTR_SOURCE=HEAD`. Reading of the row's cells. | Either add `GIT_ATTR_SOURCE=HEAD` to the `scan_range` git call (an unknown env var is ignored by an older git; it applies only to the `--range` path, where HEAD exists), or add the gap to the header list. Rewrite the row: item 3 reproduced (with this fixture), items 3 and 4 pinnable (`GIT_ATTR_SOURCE=HEAD`, `GIT_NO_REPLACE_OBJECTS=1`), and drop "none pinnable". I did not check which git version added `GIT_ATTR_SOURCE`. |
| LOW | `scripts/secret-scan.sh:174-180`, `:188-190` | In `--diff` mode, the state machine resets the header state only on a `diff ` line. For a unified diff with several files and no `diff ` line between them (concatenated `diff -u` output, svn-style `Index:` sections), the second and later files' `+++ ` headers arrive while `in_hunk` is still 1. They are then read as added content, so a token-shaped file name is reported. Main skipped every `+++ ` line, so this is a new false positive, though only an over-report. `--range` is not affected (git always prints `diff --git`, and the full-history oracle comparison is identical), and no caller in the repo uses `--diff`. The comment states the rule accurately. | `--diff` on `--- a/x`, `+++ b/x`, `@@`, `+clean`, `--- a/<token-name>`, `+++ b/<token-name>`, `@@`, `+clean`: one finding, rc 1. The svn-style input also gives rc 1. | Record it in tech-debt. Either say in the usage text or comment that `--diff` expects git-style input with one `diff ` line per file, or count hunk lines from the `@@ -a,b +c,d @@` header (this review's oracle does that, and it matched the new awk on the full history), so a header after a hunk is recognized without a `diff ` line. |
| LOW | `scripts/secret-scan.sh:22-23`, `:127`; `docs/tech-debt/README.md:140` | Three small wording issues. (a) The exit-3 parenthetical "(a missing or unreadable --file, or git failed)" leaves out the third cause the same commit added: a staged entry that does not parse (`:152-155`). (b) `--no-abbrev` is explained as "full object ids, as cat-file reads them", but `cat-file` accepts abbreviated ids. The reader that needs full ids is `is_object_id` (40 or 64 hex), and without the flag every staged file exits 3. (c) The new test-gaps row attributes an observation to "self-review Slice D", but Slice D is an implementer slice. The cycle-1 self-review noted that `--no-show-signature` was untested. | Reading of the lines cited. | (a) Add "or a staged entry that does not parse". (b) Say "full object ids, which is_object_id requires". (c) Say "the cycle-1 self-review". |

No CRITICAL, HIGH, or MEDIUM findings.

Status of the cycle-1 findings at HEAD:
- MEDIUM-1: fixed, with a discriminating test.
- LOW-2, LOW-3, LOW-4: fixed, each with a discriminating test.
- LOW-5: fixed by ca8a212, and the "++ " item was correctly removed from the CI-shared tech-debt row.
- LOW-6: fixed.
- LOW-1: fixed in the script header, but it came back in the quality gates (the first finding above).

### Positive notes

- The state-aware header rule matches a line-counting oracle exactly, both on real git output (215,825 lines across the whole history) and on a fixture built to contain every header-looking content line and header-looking file name I could think of. It is also a strict superset of main's reads: the only lines it adds are hunk content that starts with `++ `.
- The leading-only SGR strip removes the root cause of AR-1 instead of patching around it. A file's own escape bytes now reach the patterns unchanged, and a colored `+++ b/<name>` header is still skipped.
- Every Slice D and Slice E fix came with a test that turns red when the fix is reverted. The TERM test is built carefully: the FIFO holds the scanner in `cat`, the signal is sent only after the findings file exists, and it measures the scanner rather than the harness. It exits 0 under the cycle-1 trap and 143 under the fixed one. The fake `git` wrapper for the unparsed-line test intercepts only `--raw` and passes every other call through.
- The fixes follow the cycle-1 recommendations as mechanisms, not just instances. The header now says "listed there" and names its exceptions instead of claiming "every". `-c log.showSignature=false` is a portable pin. The traps copy the sibling script's pattern and its comment.
- The branch-script header's exit-code list is now self-consistent, and the `/pr` skill's exit-3 text (all four copies) matches it.

### Coverage gaps

- No test depends on the `while` in the SGR strip: a single `sub` keeps every test green. git's own colored output never puts two SGR codes in front of a line (checked with `color.diff.new='green bold reverse'`, `diff.colorMoved=zebra`, `diff.wsErrorHighlight=all`). The loop only matters for `--diff` input from other tools. A stacked-code case in the colored `--diff` test would pin it.
- `GIT_ATTR_SOURCE`: I verified its effect on git 2.49 only. I did not check which git release introduced it.
- Combined-diff input (`@@@`, lines that start with ` +`) under `--diff` is not handled, the same as on main. `git log -p` prints no merge diffs, which is already a Non-goal with a tech-debt row.
- The cycle-1 `sync-docs` report (lines 53-60) still describes the CI-shared row as including the `++ ` item. ca8a212 changed that. The cycle-2 `/sync-docs` should say so.

### Answers to the focus points

1. **State machine.** For `git log -p` output it reads exactly the added lines: the oracle comparison is identical on the shape fixture and on the full history.
   - Every shape listed was checked: several files per commit, several commits with `commit <sha>` lines, binary, mode-only, rename with and without an edit, new empty file, `\ No newline`, `--submodule=short`, CRLF, deletion, multi-hunk, and colored `--diff` input.
   - No real added line is skipped, because a line inside a hunk can never start with `diff ` or `@@`: every content line carries its prefix.
   - No header line is scanned for `--range`, and token-shaped file names are never reported (rc 0 on the fixture).
   - The one header-as-content case is `--diff` input without `diff ` separator lines (LOW, third finding).
2. **Leading-SGR loop.** BWK awk, mawk, busybox awk, and gawk give identical bytes on plain, colored, and stacked-code inputs. The anchored pattern needs at least 3 bytes, so the loop always ends. It has no discriminating test (Coverage gaps).
3. **CI parity.**
   - For `--range`, the new rule reads more than main in exactly one class: hunk lines whose content starts with `++ `, which is intended. Nothing else changes: on the full history all three extractions are byte-identical, and every other difference is a line that does not start with `+`, which neither version prints.
   - It can newly block a branch only if that branch adds a line starting with `++ ` that also matches a secret pattern. That is a true positive. False positives are unlikely: git shows a committed patch's own `+++ ` headers as `++++ `, which main already scanned. So the change is acceptable.
   - `--diff` also reads more for colored input (intended) and for multi-file input without `diff ` lines (the third finding). No CI caller uses `--diff`.
4. **Tech-debt rows and docs.**
   - The CI-shared row matches the code after the `++ ` removal.
   - The "Four local-only ways" row is inaccurate on reproduction and on pinnability (second finding).
   - The test-gaps row's no-op claim is correct, and its attribution is off (fourth finding).
   - The quality gates overclaim (first finding).
   - `/pr` SKILL.md (four copies) and `repo-map.md` match the code.
5. **Comments and tests.**
   - The `scan_diff_stream` comment describes the rule exactly. The `scan_range` option list still matches the invocation in order, and the "since 2.33" claim holds (2.32.7 against 2.34.8).
   - Three wording nits (fourth finding).
   - The tests are hermetic: `GIT_CONFIG_NOSYSTEM=1` and `unset XDG_CONFIG_HOME` were added in both files. Their names say what they protect. They build their token fixtures at runtime and discriminate every fix under mutation, apart from the SGR loop.

### Recommendation

**Pass**: no CRITICAL, HIGH, or MEDIUM findings, so the branch can proceed to `/verify` → `/test` → `/sync-docs` → `/cross-review` → `/pr`.

This is the final cycle, so each of the four LOWs must be fixed now or recorded:
- Findings 1 and 4 are wording changes, cheap to fix in the `/sync-docs` pass.
- For finding 2, correct the existing "Four local-only ways" row in the same pass (reproduced; `GIT_ATTR_SOURCE=HEAD` and `GIT_NO_REPLACE_OBJECTS=1` as the known closures), or add the one-line env pin.
- For finding 3, either state git-style input in the `--diff` usage, or add it to the tech-debt register together with the untested SGR loop.

## Cycle 3

- Date: 2026-09-26
- Plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff 750d247...HEAD` at 63461d6, read in full with the whole branch as context. The code commits are 443ffb7 (Slice F: the fixes for cycle 2's findings plus `GIT_ATTR_SOURCE=HEAD`) and 004d99e (Slice G: the fix for cross-review AR-2). No self-review had seen 443ffb7 before, so it is reviewed here together with 004d99e. The docs commits are 5774b5f, 092c5e3, b403b9f, dca6c2f, df3193e, and 63461d6. This is the final run (the cap was raised to 3), so any finding left unfixed becomes a deferral.

### Evidence reviewed

- Starting state: worktree HEAD 63461d6, `git status --porcelain` empty. The DAG is linear from 750d247. Both scanners are still `cmp`-identical to their `templates/base/scripts/` copies.
- Test suites:
  - `sh tests/test-secret-scan.sh` passes 81/81, and gives the same 81/81 on a scratch copy with the scanners run under `/bin/dash`.
  - `sh tests/test-secret-scan-branch.sh` passes 108/108.
  - After the runs, the real worktree has no `MERGE_HEAD`, no `info/attributes`, no local `attr.*`/`diff.*` config, and its hooks are unchanged.
- **Mirror of AR-2** (fixture `scratchpad/c3/mirror.sh`). The base commit has `*.txt -diff`. The incoming branch, forked from base, adds a token and deletes it again. `main` then removes `.gitattributes`. During `git merge --no-ff --no-commit incoming`, the working tree (the merge in progress) has no `.gitattributes`, while the incoming side's tree still has `*.txt -diff`.

  | Check | rc |
  | --- | --- |
  | `prepare-commit-msg-secret-guard.sh` with main's scanner (c1785b8) | 1 |
  | the same guard with the HEAD scanner | **0** |
  | `--staged` (the pre-merge-commit guard) | 0 |
  | after committing, a range ending at the merge commit (what CI reads) | 1 |
  | after committing, a `base..HEAD` branch-style scan | 1 |

  With `GIT_ATTR_SOURCE=<merge head>`, `git log -p` prints `Binary files /dev/null and b/leak.txt differ` for both incoming commits.
- **Prototype fix** (scratch copy only): set `GIT_ATTR_SOURCE` only when the range end resolves to HEAD's commit; otherwise read the working tree, as main did.
  - The mirror case gives rc 1.
  - The whole test file gives 80/81, and the AR-2 merge and guard tests stay green.
  - The one failure is `a symmetric A...B range reads attributes from B`, the test that pins the choice this finding questions.
- **Range forms** (fixture `c3/forms.sh`, HEAD = `feature`, with an upstream set). Each form below was run through `git log -p` (which accepted all of them) and through the scanner:

  | Form | Scanner |
  | --- | --- |
  | `main..feature`, `main...feature`, `main..`, `feature`, `main..@{u}`, `@{u}`, `main..HEAD@{0}`, `main..feature~0`, `main..feature^{/fix}` | rc 1 (resolved) |
  | `..feature` | rc 0 (the range is empty) |
  | `^main` | rc 0 (git log prints nothing; `rev-parse --verify` returns `^<sha>`, which git would reject with `bad --attr-source` only if it needed attributes) |
  | `feature^!`, `feature^@`, `feature^-`, `feature^-1`, `main..:/fix: add` | **rc 3**, "does not name a commit" |
  | `main..:/add..x` | git log itself fails (rc 128); scanner rc 3 |

- **CI parity.** `.github/workflows/verify.yml` scans `$base..HEAD`, and `secret-scan-branch.sh` scans `$merge_base..HEAD`: both resolve the end to HEAD. The fixture `c3/prmerge.sh` simulates CI's `pull_request` checkout of the PR merge commit, with base carrying `-diff`, main later dropping it, and the branch adding a token:

  | Scan | rc |
  | --- | --- |
  | CI-style `base..<PR merge>` | 1 |
  | local branch scan, HEAD scanner | 0 |
  | local branch scan, main's scanner | 0 |

  So this divergence predates the PR and is reproduced here.
- **`.git/info/attributes` under `GIT_ATTR_SOURCE`:** rc 1 at baseline and rc 0 with `*.txt -diff` in `info/attributes`. This confirms the tech-debt row's claim that the variable does not reach it.
- **Mutations for Slice G** (scratch copy; each checked for its changed-line count):

  | Mutation | Tests that went red |
  | --- | --- |
  | back to `GIT_ATTR_SOURCE=HEAD` | the `A...B`, merge-head and guard tests |
  | drop `GIT_ATTR_SOURCE` | six attribute tests |
  | drop the empty-end-to-HEAD default | the `main..` test (rc 3) |
  | replace the exit 3 with `attr_source=HEAD` | only the stderr assertion (git log still fails for a missing end) |
  | split at the first `..` | the `A...B` test (rc 3) |

- Not run: git 2.41 or 2.42. `alpine/git` and `bitnami/git` have no such tags, so the tech-debt row's "2.41 and 2.42 themselves were not run" still stands.

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-H1 | HIGH | `scripts/secret-scan.sh:221-224`, `:230-238`; `tests/test-secret-scan.sh:263`, `:273`, `:276` | The AR-2 fix swaps which side of a merge is misread; it does not stop the misreading. The merge guard scans `HEAD..<merge head>` before HEAD advances. What CI will read later, and what main's guard read, is the attribute set of the **merge result**, which is the working tree during the merge. It is neither HEAD's tree (the AR-2 regression) nor the merge head's tree (this one). When HEAD's side has removed a `-diff` that the incoming side still carries from the fork point, the incoming side's add-then-delete token is now hidden. main's guard and the post-merge scan detect it. The staged guard cannot catch it, because the token is deleted before the merge. The comment "the merge guard ends at the merge head, whose attributes are the incoming side's" states the rationale that fails here. Mitigations are the same as for AR-2: after the merge commit, `secret-scan-branch.sh` (`/pr --strict`, `run-verify.sh`) and CI read the merge result and catch it before or at push. The trigger is as narrow as AR-2's, but this is the same kind of fail-open regression versus main, in the same guard, and AR-2 was fixed for exactly that reason. | Mirror fixture: main's guard rc 1, HEAD guard rc 0, staged rc 0, post-merge rc 1. The prototype described below gives rc 1 and keeps AR-2 fixed. | Set `GIT_ATTR_SOURCE` only when the resolved range end is HEAD's commit. That covers CI and the branch scan, and all Slice F cases stay pinned. For any other end, leave attributes to the working tree, which during a merge is the merge result and was main's behavior. Alternatively, have the guard pass the merged index's tree (`git write-tree` at prepare-commit-msg time, when the index is fully merged) as the source. Add the mirror fixture as a test next to the AR-2 one. Drop or rewrite the `A...B reads B` test and the "incoming side's" comment. If this is deferred instead, it needs the user's explicit acceptance and a tech-debt row, since it is a regression versus main. |
| C3-L1 | LOW | `scripts/secret-scan.sh:36`, `:221-224`, `:230-232` | The range-end parser accepts fewer forms than git. `X^!`, `X^@`, `X^-`, `X^-N`, and a `:/<regex>` end now exit 3 where `git log` accepts them. For `:/<regex>`, the appended `^{commit}` becomes part of the regex. All of these fail closed, and no caller in the repo passes them. Two forms resolve to a commit other than the one whose history is being read. `A...B` has two tips, but B's attributes are applied to A's side too, and a test pins this. The parser also splits at the **last** `..` while git splits at the first. These differ only when the end revision itself contains `..` (`:/<regex>`), and in the probes such a form failed closed. | Forms probe table above. | Say in the usage line or the `scan_range` comment which forms are supported: `A..B`, `A..`, `..B`, or a single revision. Other forms exit 3. `A...B` either exits 3 or falls back to the working tree once C3-H1 is fixed as recommended. |
| C3-L2 | LOW | `scripts/secret-scan.sh:5-9`, `:15-17` | Two header sentences 443ffb7 extended are now inaccurate. (a) "working-tree attributes … to git's defaults": attributes are pinned to the range end's tree, not to a git default. (b) "a -c key or environment variable an older git does not know is ignored, and such a git has no such setting to neutralize" is true for the `-c` keys but false for `GIT_ATTR_SOURCE`. git 2.8 through 2.40 does read working-tree attributes and ignores the variable, so on those versions the gap stays open. The tech-debt test-gaps row says so ("falls back to the pre-Slice-F working-tree read"). | Reading of the header against `docs/tech-debt/README.md:140`. | (a) "… replace refs, signature output) to git's defaults, working-tree attributes to the range end's tree, …". (b) "a -c key an older git does not know is ignored, and such a git has no such setting; `GIT_ATTR_SOURCE` needs git 2.41, and an older git keeps reading the working tree's attributes." |
| C3-L3 | LOW | `docs/plans/active/2026-09-25-secret-scan-git-config.md:133` (Slice F bullet); `docs/tech-debt/README.md`; `docs/quality/quality-gates.md:46` | The pre-existing divergence that CI's `pull_request` checkout of the PR merge commit reads that commit's attributes, while the local branch scan reads the branch tip's, is recorded only as "未確認の注意点" in a plan bullet. `/pr` archives the plan, so this final run is the last chance to register it. It reproduces: when main drops a `-diff` after the branch forked, CI finds the branch's token and every local scan, including main's scanner, reports clean. The quality-gates sentence names only "local-only attribute sources" as exceptions, so it does not cover this case either. | Fixture `c3/prmerge.sh`: CI-style rc 1, local rc 0 with both scanners. | Add a tech-debt row, or extend the local-only row: pre-existing, reproduced, with its trigger. Make the quality-gates exception wording general enough to include it, for example "except for the attribute differences listed in …". |

No CRITICAL or MEDIUM findings.

Status of the cycle-2 findings at HEAD:
- All four were fixed in 443ffb7.
- Finding 1: the quality-gates wording now names the exceptions.
- Finding 2: the uncommitted `.gitattributes` gap is now pinned. The tech-debt row was rewritten, and its claims hold: `info/attributes` is still read (probe above), and `GIT_NO_REPLACE_OBJECTS=1` closes the replace-ref item.
- Finding 3: the `--diff` input limit is in the usage text, the comment, and tech-debt.
- Finding 4: all three wording items are fixed.

### Positive notes

- Slice G fails closed where it cannot decide: an end that does not resolve to a commit, or an unborn HEAD, exits 3 with a message naming the end. `GIT_ATTR_SOURCE` receives the resolved commit id, not the name.
- The new tests exercise the real mechanism. They run a real `git merge --no-commit` and the real `prepare-commit-msg-secret-guard.sh` inside a fixture repo that holds a copy of the scanner. They skip cleanly on git older than 2.41, and they tidy up with `git merge --abort` and `rm -rf scripts`. Nothing touches the real repo's hooks, config, or git dir, which I checked after the runs.
- 443ffb7's rewrite of the tech-debt row is precise where precision matters. "No value restores git's own binary detection" is correct for `diff.<driver>.binary`, where `false` forces text rather than switching to auto-detection. The row also states where `GIT_ATTR_SOURCE` does and does not reach.

### Coverage gaps

- git 2.41 and 2.42 are still not run for `GIT_ATTR_SOURCE`. No image with those versions was found; the tech-debt row already states this.
- No test covers the mirror merge case (C3-H1) or a merge whose two sides both edit `.gitattributes`.
- The `^main` form yields `GIT_ATTR_SOURCE=^<sha>`. It is harmless today, because a range with only negative revisions prints nothing and git never resolves the source. If attributes are ever needed on that path, git will die with `bad --attr-source` (rc 128, then exit 3), so it fails closed.

### Answers to the focus points

1. **443ffb7 together with 004d99e.**
   - 443ffb7's four cycle-2 fixes are correct. Its `GIT_ATTR_SOURCE=HEAD` produced AR-2.
   - 004d99e fixes AR-2 but opens its mirror (C3-H1).
   - 443ffb7's header wording is partly inaccurate (C3-L2).
   - 443ffb7's tech-debt rewrite is accurate against probes.
2. **Range-end parsing.**
   - Resolved correctly: `A..B`, `A...B` (to B), `A..` and `A...` (to HEAD), `..B`, a single revision, and `@{u}`, `HEAD@{n}`, `~n` and `^{/text}` ends.
   - Exit 3 where git log accepts: `X^!`, `X^@`, `X^-`, `X^-N`, and `:/<regex>` ends. All fail closed.
   - A different commit than the one being read: `A...B`, whose A side is read with B's attributes, and the merge-guard case (C3-H1, which fails open). The last-`..` split differs from git only when the end itself contains `..`, and that failed closed in the probe.
3. **`GIT_ATTR_SOURCE` semantics.**
   - A commit id works as the tree-ish.
   - git older than 2.41 ignores the variable and reads the working tree, as the tech-debt row states; the header sentence says otherwise (C3-L2). 2.41 and 2.42 themselves were not run.
   - An unborn HEAD with a `..HEAD` or empty end exits 3, and a test covers it.
   - Shallow clones: the range end is the checked-out HEAD (CI, with `fetch-depth: 0`, and the branch scan) or a fetched `MERGE_HEAD` (the guard), so it is always present.
4. **CI parity.** verify.yml and the branch scan both end at HEAD, so attributes come from HEAD's tree. In CI that is the checked-out PR merge commit, identical to its worktree. The remaining divergence is local branch tip against CI's merge commit when the base changed its attributes after the branch forked. It is pre-existing and reproduced, but not registered (C3-L3).
5. **Tech-debt rows.**
   - The local-only row is accurate: `info/attributes` and `diff.<driver>.binary` are unpinnable, the replace ref is closable, and the `GIT_ATTR_SOURCE` note is correct.
   - The test-gaps row's version claim is honest about 2.41 and 2.42 being unrun.
   - The CI-shared row is unchanged and still accurate.
   - Missing: a row for C3-L3, and for C3-H1 if it is deferred.
6. **Tests.** Hermetic (isolated HOME, global and system config, `GIT_CONFIG_NOSYSTEM`, no XDG). Tokens are built at runtime, and a scan of the report is clean. The merge fixture and the guard run stay inside `$workdir` and use a copied scanner. Each Slice G mutation turns at least one test red. The `A...B` test pins behaviour that C3-H1 recommends changing.

### Recommendation

**No merge until C3-H1 is fixed or explicitly accepted by the user.** The verdict is pass by the pipeline's stop rule (no CRITICAL).
- C3-H1 is a verified fail-open regression versus main in the same guard AR-2 was about, and it was introduced by the AR-2 fix. The prototype fix is a two-line change: `GIT_ATTR_SOURCE` only when the end is HEAD. With it, 80 of 81 existing tests stay green, and the one that fails pins the questionable choice. A fix needs another pipeline run beyond the raised cap of 3; accepting the regression needs a tech-debt row that states it is a regression versus main.
- The three LOWs are documentation and registration work, cheap in either path.
