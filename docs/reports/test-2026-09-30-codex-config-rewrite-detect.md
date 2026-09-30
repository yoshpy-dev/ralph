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
