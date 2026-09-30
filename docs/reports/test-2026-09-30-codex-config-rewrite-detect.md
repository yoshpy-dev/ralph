# Test report: codex-config-rewrite-detect

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-30-codex-config-rewrite-detect.md
- Tester: tester subagent (Claude), cycle 1
- Scope: behavioral tests only (no static analysis), against worktree HEAD
  `8e7a678` (branch `fix/codex-config-rewrite-detect`), `git status --porcelain`
  empty throughout.
- Evidence: `docs/evidence/verify-2026-09-30-065349.log` (full-scope shell +
  Go suite run), plus scratch-only fixtures under this session's scratchpad
  (not committed, per the pointers-not-dumps convention for report bodies).

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (full shell suite + golang) | all `tests/test-*.sh` + 8 Go packages | all pass (see per-file counts below; 0 FAIL lines anywhere in the log) | 0 | 0 | ~single run, exit 0 |
| `sh tests/test-ralph-worktree.sh` — run 1 | 105 | 105 | 0 | 0 | seconds |
| `sh tests/test-ralph-worktree.sh` — run 2 | 105 | 105 | 0 | 0 | seconds |
| `sh tests/test-ralph-worktree.sh` — run 3 | 105 | 105 | 0 | 0 | seconds |
| `dash tests/test-ralph-worktree.sh` | 105 | 105 | 0 | 0 | seconds |
| `TMPDIR="<dir with a space>" sh tests/test-ralph-worktree.sh` | 105 | 105 | 0 | 0 | seconds |

`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` exit code: `0`. Grepping the
full log for `[0-9]+ failed` returns only `0 failed` occurrences; the golang
verifier section shows all 8 test-bearing packages `ok` (internal/cli,
internal/config, internal/insights, internal/org, internal/org/driver,
internal/org/protocol, internal/scaffold, internal/upgrade).
`tests/test-ralph-worktree.sh` itself reports `105 passed, 0 failed, 105
total` inside that run (evidence log line 803).

## End-to-end (real entry point, scratch clone)

Scratch clone made with `git clone --no-hardlinks <worktree> <scratch>/clone`
then `git -C <scratch>/clone checkout -q -B main`. One setup note: cloning
directly from the task worktree (whose own HEAD sits on
`fix/codex-config-rewrite-detect`) makes git set the clone's
`refs/remotes/origin/HEAD` to that branch, not `main` — a clone artifact of
testing from a feature-branch worktree, not a product defect (a real GitHub
clone's `origin/HEAD` points at the repo's actual default branch). Fixed
once with `git -C <clone> remote set-head origin main` before any case
below; `default_branch()` (`scripts/ralph-common.sh`) resolves off that ref,
so leaving it unfixed would have made every case fail for an unrelated
reason (`must start from clean default branch 'fix/codex-config-rewrite-detect'`).

| Case | Setup | Command | Expected | Observed |
| --- | --- | --- | --- | --- |
| (a) rewrite shape | `.codex/config.toml` comments+blanks stripped, `[shell_environment_policy]`/`[shell_environment_policy.set]` appended, unstaged | `ensure --id t185 --kind standard --branch test/t185 --path .claude/worktrees/t185 --cleanup-policy pr-success` | non-zero, specific message, no worktree, no state file | PASS — exit 1, specific message with the three recovery commands (`git -C <clone> diff/checkout/status`), `git worktree list` unchanged, `$(git rev-parse --git-common-dir)/ralph/worktrees/` empty both before and after |
| (a) recovery | printed `diff`/`checkout`/`status` commands run verbatim, then `ensure` re-run | same `ensure` invocation | clean after checkout, `ensure` then succeeds | PASS — `status --porcelain` empty after checkout; `ensure` exit 0, worktree created at `.claude/worktrees/t185` on branch `test/t185`; `cleanup --id t185 --force-branch` exit 0, worktree and branch removed |
| (b) rewrite + value change | same rewrite as (a), plus `approval_policy` value changed | `ensure --id t185b ...` | generic message only | PASS — exit 1, `has uncommitted changes` (generic), no rewrite-specific text; restored via plain `checkout --` |
| (c) staged rewrite | same rewrite as (a), `git add .codex/config.toml` (porcelain `M ` not `" M"`) | `ensure --id t185c ...` | generic message only | PASS — exit 1, generic message; restored via `git reset --` + `checkout --` |
| (d) subdirectory | same rewrite as (a), commands run from `<clone>/docs` | `validate-clean-base main` | specific message, root-relative recovery commands that work from the subdirectory | PASS — exit 1, specific message with absolute `-C <clone>` paths; running the printed `diff`/`checkout`/`status` commands from `<clone>/docs` restored clean; a follow-up `validate-clean-base main` (still run from `docs/`) then exited 0 |

`git worktree list` and `$(git rev-parse --git-common-dir)/ralph/worktrees/`
were checked before/after every `ensure` call above and never showed a
leaked entry outside the one expected in the (a)-recovery case, which
`cleanup --force-branch` removed cleanly.

## Mutation testing

Scratch mutation root built by copying `scripts/ralph-worktree.sh`,
`scripts/ralph-common.sh`, and `tests/test-ralph-worktree.sh` into a fresh
scratch directory (so `PROJECT_ROOT` in the test script resolves there);
tracked files in the worktree were never edited. Baseline (unmutated copy)
run: `105 passed, 0 failed, 105 total`, confirming the copy itself is
faithful before mutating.

| Mutation | Description | Result | Failing cases |
| --- | --- | --- | --- |
| (a) always MATCH | `END{}` block in `codex_config_external_rewrite_only`'s awk short-circuits to `print "MATCH"; exit` as its first statement, before any real comparison | 8 cases fail | case 4 (value change alongside match shape), case 6 (non-policy table appended), case 10 (multi-line string guard), case 11 (CRLF), case 14 (comment content changed), case 15 (comment added), case 16 (indentation-only change), case 17 (trailing-comment header after a policy table) |
| (b) drop porcelain guard | `validate_clean_base`'s `if [ "$dirty" = " M .codex/config.toml" ] && ...` narrowed to `if codex_config_external_rewrite_only; then` (classifier called regardless of the exact-porcelain shape) | 4 cases fail | case 5 (another dirty tracked file), case 7 (unrelated untracked file only), case 8 (staged rewrite), case 9 (partially staged rewrite) |
| (c) allow comment in appended region | Trailing-region loop's `if (is_comment(nl)) { print "NOMATCH"; exit }` changed to `{ continue }`, so a brand-new standalone comment line inside the appended `[shell_environment_policy]` table no longer disqualifies a match | **0 cases fail — coverage gap** (see below) | — |
| (d) accept `[[shell_environment_policy]]` | `is_sep_header` regex widened to also accept the TOML array-of-tables double-bracket form (`[[shell_environment_policy]]` / `[[shell_environment_policy.<name>]]`) as a valid appended header | **0 cases fail — coverage gap** (see below) | — |

Mutations (a) and (b) match the two mutation kinds the plan and the verify
report predicted (AC-5 detector-always-false / table-name-detection removed,
and AC-2(b)/(e) porcelain-guard loosening), and the failing-case sets line
up with those reports' predictions (verify report's equivalent porcelain
mutation found cases 5/8/9; this run's slightly different mutation — calling
the classifier unconditionally rather than substring-matching the porcelain
line — additionally catches case 7, since an untracked-only dirty state
still leaves `.codex/config.toml` byte-identical to HEAD, which the
classifier's own docstring already flags as an input it does not
special-case). Mutations (c) and (d) were the two "informational" checks
the plan asked for; both found real gaps, not previously reported.

## Coverage gaps

- **Standalone new comment inside the appended table is undetected by any
  test.** No case constructs a rewrite where the appended
  `[shell_environment_policy]`/`.set` block itself contains an added
  comment line (e.g. `# note` between `inherit = "core"` and
  `[shell_environment_policy.set]`), as opposed to case 17's comment
  attached to a *non-policy* header on the same line (`[features2] # c`,
  which is caught by `is_sep_header` failing regardless of the comment).
  Mutation (c) shows the trailing-region `is_comment` check
  (`scripts/ralph-worktree.sh:151`) is currently load-bearing but silently
  so — nothing fails if it is weakened to `continue`. Recommend a case:
  HEAD unchanged aside from comments/blanks stripped, appended table =
  `[shell_environment_policy]` / `inherit = "core"` / `# injected` /
  `[shell_environment_policy.set]` / `SOME_VAR = "1"` -> expect the generic
  message.
- **TOML array-of-tables syntax (`[[shell_environment_policy]]`) is not
  exercised.** `is_sep_header` (`scripts/ralph-worktree.sh:113-115`) matches
  only the single-bracket form the known rewrite actually produces; no test
  proves the double-bracket form is rejected as a non-policy header. This is
  lower-risk than the comment gap (the known external rewrite has never
  been observed to emit array-of-tables syntax, per the plan's own
  Assumptions section), but a mutation widening the regex to accept it goes
  undetected today. Recommend a case: append `[[shell_environment_policy]]`
  instead of `[shell_environment_policy]` -> expect the generic message
  (this is a "some other table shape" case, structurally close to existing
  case 6).

Both gaps are test-only additions (no production-code change implied); they
do not block this cycle's pass verdict per the post-implementation pipeline
contract (`/test` reports behavioral pass/fail and coverage gaps, it does
not gate on closing them in the same cycle).

## Failure analysis

None — no unexpected failures observed in any run.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `ensure`/`validate-clean-base` used to stop with only the generic "has uncommitted changes" message for the known external rewrite, giving no recovery path (issue #185) | Fixed | End-to-end case (a) above: specific message + working recovery commands, confirmed through the real CLI entry point on a scratch clone, not just the unit-test fixtures |
| Clean base still passes (no regression to the existing gate) | Confirmed unchanged | `tests/test-ralph-worktree.sh` case 13 (clean base) passes in all 5 runs above; end-to-end case (a)-recovery's second `ensure` call succeeds after restoring a clean file |

## Verdict

- Pass: yes — all 5 `tests/test-ralph-worktree.sh` runs (3x `sh`, 1x `dash`,
  1x space-containing `TMPDIR`) report `105 passed, 0 failed`; the full
  `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` run exits `0` with no
  failures anywhere in the shell or Go output; all 4 end-to-end scratch-clone
  cases behave as the plan specifies; mutations (a) and (b) are correctly
  caught by the existing test set.
- Fail: none.
- Blocked: none.
- Known gaps carried forward (not blocking): the two coverage gaps above
  (standalone comment inside the appended table; `[[shell_environment_policy]]`
  array-of-tables form), both test-only additions if picked up later.
- Post-run confirmation: `git status --porcelain` empty in both the task
  worktree and the main checkout after all testing; the scratch clone's
  `git worktree list` and state files were never visible to the real repo's
  `$(git rev-parse --git-common-dir)/ralph/worktrees/`.

## Cycle 2

- Date: 2026-09-30
- Tester: tester subagent (Claude), cycle 2 (final)
- Scope: behavioral tests only, against worktree HEAD `8fc3eb3`
  (`docs: verify cycle 2 for codex-config-rewrite-detect`), `git status
  --porcelain` empty before and after. Re-verifies the cycle-1-to-cycle-2
  delta: Slice C `d84cdab` (cases 20-21, closing cycle 1's two known gaps),
  cross-review AR-1 + Slice D `ee397e9` (indented headers, cases 22-24),
  self-review cycle 2 + Slice E `a2c721d` (at-least-one-known-change guard,
  single-quote root quoting, cases 25-28). Mutation re-run for this delta
  (verify cycle 2 did not re-run mutations).
- Evidence: `docs/evidence/verify-2026-09-30-095434.log` (full-scope run,
  this cycle's run 1).

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | all `tests/test-*.sh` + 8 Go packages | all pass (0 `FAIL` lines, 0 non-zero `failed` counts anywhere in the log) | 0 | 0 | exit 0 |
| `sh tests/test-ralph-worktree.sh` — run 1 | 143 | 143 | 0 | 0 | seconds |
| `sh tests/test-ralph-worktree.sh` — run 2 | 143 | 143 | 0 | 0 | seconds |
| `sh tests/test-ralph-worktree.sh` — run 3 | 143 | 143 | 0 | 0 | seconds |
| `dash tests/test-ralph-worktree.sh` | 143 | 143 | 0 | 0 | seconds |
| `TMPDIR="<space + 日本語>" sh tests/test-ralph-worktree.sh` | 143 | 143 | 0 | 0 | seconds |

`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` exit code: `0`;
`tests/test-ralph-worktree.sh` inside that run reports `143 passed, 0
failed, 143 total`; all 8 Go packages `ok`.

**TMPDIR note:** on this host, macOS's `mktemp -d` called with no template
(the test file's top-level `_tmp="$(mktemp -d)"`, `tests/test-ralph-worktree.sh:50`)
silently ignores `TMPDIR` — confirmed by direct probe (`TMPDIR="<dir>" sh -c
'mktemp -d'` returned a path under the standard `/var/folders/.../T/`
location, not under the given directory) and by finding nothing left under
the custom `TMPDIR` after the run. The per-repo fixture directories
(`mktemp -d "$_tmp/codex-repo.XXXXXX"`, an explicit template rooted at
`$_tmp`) are consequently also unaffected by `TMPDIR` on this host — the
space/non-ASCII path coverage for the detector itself comes entirely from
case 19 (space, cycle 1) and case 28 (Japanese + literal quote, cycle 2)
building their own explicit target directories via `_cx_new_repo`'s second
argument, not from the environment variable. The run still exercises the
full suite unchanged (143/0); it is a no-op with respect to `TMPDIR`
specifically on this host, not a gap in the suite.

**`gawk` substitution:** not run — `command -v gawk` exits 1 (not
installed on this host); nothing was installed per the report-only
constraint.

### End-to-end (real entry point, scratch clone)

Scratch clone(s) built the same way as cycle 1 (`git clone --no-hardlinks`,
`checkout -q -B main`, then `git remote set-head origin main` to correct
the clone-from-feature-branch-worktree `origin/HEAD` artifact documented in
cycle 1, before any case).

| Case | Setup | Command(s) | Expected | Observed |
| --- | --- | --- | --- | --- |
| (a) recorded rewrite shape | comments/blanks stripped + `[shell_environment_policy]`/`.set` appended, unstaged | `ensure --id t185ca ...` | specific message; printed commands restore; `ensure` then succeeds; `cleanup --force-branch` removes it | PASS — exit 1 specific message (single-quoted `-C` paths); `diff`/`checkout`/`status` run verbatim restored clean; re-run `ensure` exit 0, worktree created; `cleanup --force-branch` exit 0, worktree+branch removed |
| (b) non-ASCII + quote path | same clone re-created under `<scratch>/日本語 clone's dir` (Japanese component + literal `'`), same rewrite | `validate-clean-base main`, then the printed checkout line pasted into `bash -c` and `zsh -c` | specific message with correctly single-quote-escaped path; checkout line round-trips through both shells | PASS — message uses `'...\''...'`-escaped path (embedded `'` correctly escaped); `bash -c "<line>"` exit 0, status clean, `validate-clean-base` then exit 0; re-dirtied, `zsh -c "<line>"` exit 0, status clean, `validate-clean-base` then exit 0. (One self-caught harness mistake: the first attempt at the `zsh -c` sub-case ran `validate-clean-base` without `cd`-ing into the Japanese-path clone first — each Bash call resets cwd — so it silently validated the unrelated, already-clean main checkout instead; caught immediately via the mismatch between an expected dirty status and an unexpected exit 0/empty output, corrected by re-running with an explicit `cd` into the clone, which reproduced the correct specific-message behavior. Not a product defect.) |
| (c) indented non-policy header | recorded rewrite shape plus `\n  [features]\n  hooks = false\n` appended | `validate-clean-base main` | generic message | PASS — exit 1, generic `has uncommitted changes` message, no rewrite-specific text; `git checkout --` restored clean |
| (d) mode-only change | `chmod +x .codex/config.toml` on an otherwise-clean clone, no content change | `validate-clean-base main` | generic message | PASS — exit 1, generic message; `chmod -x` restored clean |

`git worktree list` / `$(git rev-parse --git-common-dir)/ralph/worktrees/`
checked before/after case (a)'s `ensure` calls; no leaked entry survived
`cleanup --force-branch`.

### Mutation testing

Scratch mutation root rebuilt fresh at cycle-2 HEAD the same way as cycle 1
(copy `ralph-worktree.sh` + `ralph-common.sh` + `test-ralph-worktree.sh`
into a scratch directory so `PROJECT_ROOT` resolves there; tracked files
never edited). Baseline (unmutated copy): `143 passed, 0 failed, 143
total`.

| Mutation | Description | Result | Failing cases |
| --- | --- | --- | --- |
| (a) revert header test to `/^\[/` | Appended-region header check narrowed back from `/^[ \t]*\[/` to the cycle-1 unindented-only form | 3 fail | case 22 (indented non-policy header, spaces), case 23 (same, tab), case 24 (indented policy header) |
| (b) drop the at-least-one-known-change guard | Removed `if (!dropped && !saw_policy_header) { print "NOMATCH"; exit }` | 3 fail | case 25 (mode-only change), case 26 (trailing newline removed), case 27 (blank/whitespace-only lines appended) |
| (c) revert `qroot` quoting to `printf -v qroot '%q'` | Replaced the POSIX single-quote escaping (`sq="'\\''"; qroot="'${root//\'/$sq}'"`) with the pre-cycle-2 `printf -v qroot '%q' "$root"` | 19 assertions fail across 5 cases | case 1, case 2, case 3, case 18 (each: "has diff/checkout/status command", 3 assertions per case — the literal unescaped path string is no longer present verbatim in `%q`'s escaped stderr text); case 28 (all of the above plus "status is clean after bash -c checkout", "validate-clean-base passes after bash -c recovery", "status is clean after zsh -c checkout", "validate-clean-base passes after zsh -c recovery" — the mutated `%q` form does not paste back into a runnable command for a non-ASCII path under this shell/locale) |
| (d) allow a comment line in the appended region | `if (is_comment(nl)) { print "NOMATCH"; exit }` → `{ continue }` (cycle 1's gap b) | 1 fails | case 20 (comment injected inside the appended policy table) — cycle 1's gap now closed |
| (e) accept `[[shell_environment_policy]]` as a header | Widened `is_sep_header`'s regex to also match the double-bracket array-of-tables form (cycle 1's gap d) | 1 fails | case 21 (array-of-tables header) — cycle 1's gap now closed |

All 5 mutations behaved exactly as the plan's Slice D/E deviation notes
predicted; no mutation survived undetected in this cycle.

### Findings

- Both coverage gaps flagged in cycle 1 (comment injected inside the
  appended table; `[[shell_environment_policy]]` accepted as a header)
  are now closed by Slice C's cases 20/21 — confirmed via mutations (d)
  and (e) above, not just by the cases' existence.
- No new coverage gap found in this cycle's mutation sweep (5/5
  discriminated as predicted).
- One self-corrected test-harness mistake during the case-(b) `zsh -c`
  sub-case (missing `cd` into the scratch clone before an entry-point call
  that reads `git status --porcelain` without `-C`) — caught immediately
  via an unexpected exit code, not a product defect. Noted as a
  reminder for future cycles: any call to `ralph-worktree.sh` without
  `-C`/an absolute cwd requires an explicit `cd` in the same Bash
  invocation, since cwd resets between tool calls.
- `mktemp -d` (no template) ignoring `TMPDIR` on this host means the
  `TMPDIR`-with-space-and-non-ASCII-component run is not, by itself, proof
  that such a `TMPDIR` value is handled — that coverage instead comes from
  cases 19 and 28's own explicit target-directory construction. Worth
  keeping in mind if a future cycle wants dedicated `TMPDIR`-driven
  coverage: it would need an explicit `mktemp -d "$TMPDIR/tmp.XXXXXX"`-style
  template, not a bare `mktemp -d`.

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cross-review AR-1 (cycle 1): an indented non-policy header after the policy table was misread as absorbed into the policy table, so a real added setting fell through as "known rewrite" | Fixed | Mutation (a) above; end-to-end case (c) |
| Self-review cycle 2 C2-1: a diff with no dropped comment/blank line and no appended policy header (mode-only, trailing-newline, blank-only-append) was wrongly treated as the known rewrite | Fixed | Mutation (b) above; end-to-end case (d) |
| Self-review cycle 2 C2-2: `%q` renders a non-ASCII root path as a mix of raw bytes and octal escapes under bash 3.2 + UTF-8, which does not paste back through a terminal | Fixed | Mutation (c) above; end-to-end case (b), confirmed round-tripping through both `bash -c` and `zsh -c` |
| Cycle-1 gap: comment inside the appended table went undetected | Fixed | Mutation (d) above |
| Cycle-1 gap: `[[shell_environment_policy]]` array-of-tables form went undetected | Fixed | Mutation (e) above |

### Verdict

- Pass: yes — all 5 `tests/test-ralph-worktree.sh` runs report `143
  passed, 0 failed`; the full `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`
  run exits `0` with no failures anywhere; all 4 end-to-end scratch-clone
  cases (including the non-ASCII/quote path under both `bash -c` and `zsh
  -c`) behave as the plan specifies; all 5 mutations for the cycle-2 delta
  are correctly caught by the existing/new test cases, closing both of
  cycle 1's known gaps.
- Fail: none.
- Blocked: none (`gawk` substitution skipped — not installed, nothing
  installed per constraints; noted above, not a blocker).
- Known gaps carried forward: none — cycle 1's two gaps are closed;
  cycle 2's mutation sweep found no new gap.
- Post-run confirmation: `git status --porcelain` empty in both the task
  worktree and the main checkout after all testing; no leaked fixture
  repos outside the scratchpad; no orphaned worktree/state entries in
  `$(git rev-parse --git-common-dir)/ralph/worktrees/`.
