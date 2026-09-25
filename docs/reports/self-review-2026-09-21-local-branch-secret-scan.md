# Review report: local-branch-secret-scan (self-review, cycle 1)

- Date: 2026-09-21
- Plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD` at 99d3bb6 (code commits 2ca73aa, 335dddf, d4b9336, eda0248). No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- `git diff main...HEAD` (19 files, +1345/-53); worktree HEAD 99d3bb6, `git status --porcelain` empty at review start.
- Full reads: `scripts/secret-scan-branch.sh`, `scripts/secret-scan.sh`, `scripts/xreview-helpers.sh`, `scripts/run-verify.sh` (head + new block + summary tail), `scripts/archive-plan.sh`, `.claude/skills/pr/SKILL.md`, both new test files, the plan.
- `shellcheck -s sh scripts/secret-scan-branch.sh` — only SC2329 (cleanup invoked via trap) and SC1091 (sourced file not followed); both are informational and expected. `sh -n` / `bash -n` clean.
- `./scripts/secret-scan-branch.sh --strict` on this branch: `scanned a0d5bf5..99d3bb6 against origin/main: clean` (rc 0). Explicit `./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"`: rc 0. So nothing in this diff — script, tests, plan, skill text — matches a scanner pattern.
- `./scripts/check-template-purity.sh`: rc 0.
- Mirror parity by `cmp`: `scripts/secret-scan-branch.sh`, `scripts/run-verify.sh`, `scripts/secret-scan.sh`, `scripts/check-template.sh` byte-identical to their `templates/base/` copies, mode `100755` on both sides for the new script. All four `/pr` SKILL.md faces (`.claude/`, `.agents/`, and both `templates/base/`) byte-identical. `docs/quality/quality-gates.md` diverges, which is covered by the pre-existing whole-file `KNOWN_DIFFS` entry at `scripts/check-sync.sh:101`.
- Behavioural probes in scratch git repos **outside** the worktree (fixtures assembled at runtime, never written into the repo): strict-mode exit codes for a base override naming the current branch, a local base fast-forwarded onto the branch tip, execution from a subdirectory, and a stub scanner returning 130.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | `scripts/run-verify.sh:227-241` | When the branch scan is the only failing check and no language verifier ran, the run exits non-zero but the closing summary prints only the reassuring docs-only line. `==> Some verifiers failed.` lives in the `else` arm of `[ "$ran_any" -eq 0 ]`, so a docs-only branch with a finding ends with `No language verifier ran. This appears to be docs or scaffold-level work only.` followed by `Evidence saved to: ...`. The new test pins this (`tests/test-run-verify-branch-secret-scan.sh:170`) under a section header that claims the opposite: `the "docs-only" summary must not paper over a real failure`. Secondary: in the `docs_only=0` sub-branch, `status=2` at line 233 overwrites the scan's `status=1` (cosmetic today — no consumer distinguishes them; `scripts/run-static-verify.sh` is a 5-line `exec` and `run-test.sh` has no `-eq 2` branch). | Read of `scripts/run-verify.sh:218-241`; the failing-status line is unreachable when `ran_any=0`. | Print a failure verdict whenever `status -ne 0`, independent of `ran_any` (e.g. an `if [ "$status" -ne 0 ]` line before the `ran_any` block), and align the test's section comment with what it asserts. |
| MEDIUM | `.claude/skills/pr/SKILL.md:29,51` (+3 mirrors) | The gate reads `Exit 0: continue.` and `exited 0 on the pushed HEAD`, but exit 0 covers three different states: "scanned, clean", "HEAD is the base branch", and "no commits between `<base>` and HEAD". The last two return 0 **even under `--strict`** (`scripts/secret-scan-branch.sh:70-75`), which is exactly the distinction `--strict` was added to make. Probed in a scratch repo whose branch carries a GitHub-token-pattern fixture: `RALPH_XREVIEW_BASE=<current branch> secret-scan-branch.sh --strict` → rc 0, `nothing to scan, HEAD is the base branch (feature)`; same for `GITHUB_BASE_REF`. And with `refs/heads/main` fast-forwarded onto the branch tip while `refs/remotes/origin/main` is absent → rc 0, `nothing to scan, no commits between main and HEAD`, while the branch still has unpushed commits carrying the fixture. Neither is reachable by an agent that just follows the skill in this repo (`origin/main` exists and nothing sets those vars), so this is operator-reachable, not accident-reachable. | Probe transcripts above; `scripts/secret-scan-branch.sh:70-75` and `:96-98`, `:112-115`. | Make the gate require the positive line rather than the exit code: have `/pr` confirm the `scanned <a>..<b> against <ref>: clean` line, or give `nothing_to_scan` its own strict exit code so "there was nothing to look at" is not spelled the same as "I looked and it was clean". |
| LOW | `scripts/run-verify.sh:214` | `[ -n "${RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN:-}" ]` means `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=0` also skips the scan. This matches the repo idiom (`RALPH_COMMON_SOURCED`, `RALPH_XREVIEW_BASE` both use `-n`) and the echo is honest ("... set"), but for a leak gate the "any value, including 0, disables it" direction is the unforgiving one. | `grep` of `-n "${RALPH_` across `scripts/*.sh`. | Optional one-line tightening to `= "1"`, or leave and note the semantics in the message. |
| LOW | `scripts/secret-scan-branch.sh:149-152` | Every non-zero scanner exit is reported as `findings`. Probed with a stub scanner returning 130 (the interrupt case): output was `secret-scan-branch: scanned <a>..<b> against main: findings`, rc 130. Fail-closed, which is the right direction, but the label sends the operator looking for a leak that does not exist. | Stub-scanner probe in a scratch repo. | Branch on the code: `1` → findings, anything else → "scanner exited `<rc>`". |
| LOW | `scripts/secret-scan-branch.sh:52` | `trap cleanup EXIT HUP INT TERM` installs a handler that does not exit. POSIX sh resumes the script after a signal trap action completes, so a SIGINT delivered between `mktemp` (`:127`) and the `git show` redirect (`:129`) deletes the temp allowlist and then the `>` redirect recreates it under the umask instead of `mktemp`'s 0600. The window is small and the content is the already-committed allowlist, so there is no leak — but the trap does not do what its shape suggests. Verified no temp files are left behind in `$TMPDIR` after normal runs. | `scripts/secret-scan-branch.sh:47-52,127-132`; the same shape is a recorded finding from the dispatcher work. | `trap 'cleanup; exit 130' INT` / `143` TERM / `129` HUP, keeping `trap cleanup EXIT`. |
| LOW | `scripts/secret-scan-branch.sh:15` | The header documents exit 3 as `could not determine what to scan`, but one of the four `cannot_scan` callers is `scanner not found or not executable` (`:117-119`), which is "could not run the scan". The `/pr` skill's own wording (`SKILL.md:29`, "no base ref, no merge-base, scanner missing") is accurate; the script header is the copy that is not. | `scripts/secret-scan-branch.sh:15` vs `:118`. | Reword to "could not scan". |
| LOW | `scripts/check-template.sh:25` | `scripts/secret-scan-branch.sh` is now a required file, and it hard-depends on `scripts/xreview-helpers.sh` (`secret-scan-branch.sh:84-90`), which is **not** in `required_files`. A downstream project that has the required script but not the helper turns the `/pr` strict gate into a permanent exit 3. The helper is shipped (`templates/base/scripts/xreview-helpers.sh` exists, mode 755), so this is a consistency gap rather than a live break. Related: coupling a secret gate to a file whose own header calls itself "driver-agnostic helpers for the /cross-review skill" means a future `/cross-review` refactor can now break the push gate. | `scripts/check-template.sh:12-30`; `scripts/xreview-helpers.sh:2`. | Add `scripts/xreview-helpers.sh` to `required_files`, or inline the ~15 lines of base resolution so the gate has no cross-surface dependency. |
| LOW | `.claude/skills/pr/SKILL.md:29` (+3 mirrors) | The "Run it again before every later push to this branch (for example the push that carries the plan-archive commit)" sentence guards a push that no numbered step describes: `scripts/archive-plan.sh` only `mv`s the file (no `git add`/`commit`/`push` anywhere in it), so the commit-and-push after Step 8 is implicit in the skill. The instruction is correct but sits five steps away from the action it governs, and the completion-gate item "on the pushed HEAD" is ambiguous about which push once a later one exists. | `scripts/archive-plan.sh` (verified: no git commands); `SKILL.md:29` vs `:51`. | Repeat the one-line re-scan requirement at Step 8, where the later commit is created. |
| LOW | `tests/test-secret-scan-branch.sh:292-293` | `assert_exit ... 0` + `assert_stderr_contains "origin/main"` is satisfiable by a regression the test is meant to catch: the `cannot_scan` message is `no base ref for '<base>' (checked refs/remotes/origin/<base> and refs/heads/<base>)`, which contains `origin/main` and exits 0 in default mode. Probed: a repo with no `main` at all printed `cannot scan, no base ref for 'master' (checked refs/remotes/origin/master and refs/heads/master)` with rc 0. Three more exit-0 assertions do not pin a reason line either (`:462`, `:511`, `:537`), so a silent regression into "skip" reads as green. | Probe transcript; `scripts/secret-scan-branch.sh:105` vs `:147`. | Assert on `against origin/main` (the `against ` prefix appears only on the scanned line), and add a `"scanned"`/`"clean"` assertion to the other three. |
| LOW | `tests/test-secret-scan-branch.sh:141-153` | `assert_stderr_line_count` matches with `grep -c -- "$_prefix"` — a substring match anywhere in the line — while its own failure message says "lines starting with '<prefix>'". It happens to be correct here because `secret-scan-branch:` only ever appears at line start, but the assertion is weaker than it reads. Also, `grep -c ... || true` produces an empty `_got` if grep exits >= 1 with no stdout, and `[ "" -eq N ]` is a shell error under `set -eu`. | Read of `:145-146`. | Use `grep -c "^$_prefix"` and default `_got` to 0. |
| LOW | `tests/test-run-verify-branch-secret-scan.sh:58` | The doc comment says `run_verify <repo> <mode> [env=val ...]`, but the third positional is the *value* assigned to `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` (`:66`, `:75`), not an `env=val` pair. Passing the documented form would still "work" by accident, since any non-empty string skips. | Read of `:58-79` against the call site at `:206`. | Fix the comment to `[skip_value]`. |
| LOW | `docs/quality/quality-gates.md:47` | The root copy ends the new bullet with `(issue #169)`; the template copy correctly omits it. The file is already a whole-file `KNOWN_DIFFS` entry so nothing fails, but the issue number is the only thing that makes this particular line diverge, and it tells a reader running the gate nothing they can act on. | `git diff` of both copies; `scripts/check-sync.sh:98-101`. | Drop `(issue #169)` so the two copies of this line stay identical. |

## Positive notes

- The fixture discipline the plan asked for is actually held: every secret-shaped value in both test files is assembled at runtime from split pieces (`printf 'ghp_%s' '...'`, `printf 'api%s' '_key'`), and the full range scan over this branch's own history comes back clean. The risk register's first row is the one that caused #168, and it was closed properly.
- Test environment isolation is deliberate and documented where it is non-obvious: `GITHUB_BASE_REF` / `RALPH_XREVIEW_BASE` / `RALPH_SECRET_ALLOWLIST` are unset per invocation with a comment saying why (this suite's own CI sets the first for real), and `RALPH_VERIFY_SCOPE=full` is pinned against `run-test.sh` exporting `changed` for itself.
- `test_origin_preference_and_fallback` constructs the origin-preference proof with `git update-ref refs/remotes/origin/main <leak-commit>` so that the two candidate bases give *opposite* verdicts. That is the right way to prove a precedence rule, and it is rare to see.
- The allowlist-from-HEAD decision is the correct read of what CI sees, and the "ignoring ..." notice means the override is never silent.
- `test_mode_routing_clean` asserts the absence of the string `branch secret scan` in test mode, not just the absence of the "Running" line — the stronger of the two available assertions.
- Base resolution never reaches `git` as a bare argument: every candidate is embedded in a `refs/heads/` or `refs/remotes/origin/` prefix for `rev-parse --verify`, so a base name containing `..` or a leading `-` fails ref resolution rather than becoming a revision range or an option. Path handling is quoted throughout, and `CDPATH='' cd -- "$(dirname -- "$0")"` handles both a space and a leading dash. Verified by running from a subdirectory (rc 1, correct range) and by the suite's own space-path fixture.

## Coverage gaps

- I did not run either new test file end to end (that belongs to `/test`); the assertions above were read, and the product behaviour was probed directly in scratch repos instead.
- Fixture repos call `git init` + `git config user.email/user.name` without pinning `commit.gpgsign`. `init.defaultBranch` is neutralised by `git checkout -q -B main`, but a developer with global commit signing would have these fixtures attempt to sign. This matches `tests/test-secret-scan.sh` exactly, so it is a pre-existing suite-wide convention rather than something this diff introduced — flagged here, not as a finding. `GIT_DIR` / `GIT_WORK_TREE` are likewise not unset in the subshells.
- I did not measure `git log -p` cost on a long-lived branch. No hook runs `run-verify.sh` automatically (`post_edit_verify.sh` and `session_start_context.sh` only print reminders naming it), so the new step's cost lands on explicit verify runs only.
- CI now runs the same range scan twice on a pull request — once in the dedicated `Secret scan` step and once inside `run-verify.sh`. Duplicated work, not a defect.

## Answers to the requested points

1. **Shell correctness / portability.** `xreview-helpers.sh` tolerates `set -u` (all reads are `${VAR:-}`-guarded) and sourcing it has no side effects — it defines three functions and executes nothing. The temp file is removed on every normal path (verified: zero leftovers in `$TMPDIR`); the signal caveat is the LOW above. `diff -q` is the established idiom in this repo (`check-sync.sh`, `check-skill-sync.sh`, `test-hook-wiring.sh` all use it), so no portability finding. `git show HEAD:.gitallowed` on a symlink would yield the link target as a one-line regex — harmless and not worth code. Bare repo: `git rev-parse --show-toplevel` fails, so `cannot_scan` fires. Linked worktree: verified working on this very worktree. Detached HEAD (CI): `git symbolic-ref` fails, `current_branch` becomes `""`, the base-branch short-circuit does not fire, and the scan proceeds — correct. Space in the path and running from a subdirectory: both verified.
2. **Security of the gate.** See the MEDIUM above. `..` and leading-`-` base names are neutralised by the `refs/...` prefixes. The reachable-by-accident set is empty for an agent following the skill in a repo that has `origin/<base>`; the two bypasses need either a deliberately-exported base variable or a moved local base *plus* a missing origin ref. "Nothing to scan → 0" is right for the genuine empty-range case and wrong as a spelling for it, which is why the recommendation is to make `/pr` read the line rather than to change the exit code semantics. CI differs in one respect worth recording: it always uses `origin/${GITHUB_BASE_REF}` and never falls back to a local head, so the local fallback is the only place the two can disagree; when `origin/<base>` is merely stale the local range is a superset of CI's, which is the safe direction.
3. **Default-mode skip visibility.** The reason line is printed, always, exactly once, on stderr, and `run-verify.sh` puts it in the evidence log. For the default (non-strict) call that is proportionate — it is an early-warning gate. The visibility problem is not the skip line, it is the *failure* line (MEDIUM 1): a run that exits 1 can end with a reassuring summary.
4. **Allowlist handling.** No legitimate caller is broken: the only non-template consumers of `RALPH_SECRET_ALLOWLIST` are `secret-scan.sh` itself and the test suites, and the tests set it on direct `secret-scan.sh` invocations, not through the branch script. The notice's "and/or" is vague but not wrong; naming which of the two fired would be a small improvement. The implementer's model of the self-matching allowlist line is accurate: `secret-scan.sh` applies **one** allowlist (HEAD's) uniformly across every commit in the range, so a rule that matches itself suppresses its own introducing commit for as long as HEAD still carries it, and surfaces the moment a later commit changes the line — the PR #168 shape. That mechanism is explained in the scanner's guidance text and in the test's header comment; it is *not* explained in `quality-gates.md` or the `/pr` skill, which both only say "written so the line itself does not match a scanner pattern". That is adequate as an instruction and thin as an explanation, but documentation depth is `/sync-docs` territory, so I am recording it here rather than as a finding.
5. **`run-verify.sh` placement.** The block sits after docs-only detection and before the summary, which is the right place, and not counting it in `ran_any` is correct. The closing lines are *not* truthful in the `ran_any=0, docs_only=1, finding` case — MEDIUM 1. The scanner emits `  - <range>:<n> [<pattern name>]` and never the matched text (`secret-scan.sh:55`), so the evidence log carries pattern names and line numbers only; `docs/evidence/*.log` is gitignored (`.gitignore:58`). Acceptable.
6. **Guidance text.** Accurate, and nothing in this diff matches a scanner pattern — `./scripts/secret-scan-branch.sh --strict` and the explicit `--range` scan both came back clean at 99d3bb6.
7. **Tests.** Isolation from `GITHUB_BASE_REF` / `RALPH_XREVIEW_BASE` / `RALPH_SECRET_ALLOWLIST` / `RALPH_VERIFY_SCOPE` is handled and commented; `init.defaultBranch` is neutralised; commit signing and `GIT_*` are not pinned (see Coverage gaps). Temp dirs are cleaned by `trap ... EXIT HUP INT TERM` in both files. Four exit-0 assertions can pass for a skip instead of a clean scan, one of them in the test that carries the file's strongest claim — LOW above.
8. **`/pr` step.** The wording is clear and the renumbering is complete and correct: every forward reference now points at the right step, including the pre-existing Step 1 → "Step 6" error, which is now "Step 8" and accurate. All four skill faces are byte-identical. An agent following it literally cannot push unscanned commits on the *first* push; the later-push case is covered by the sentence added in eda0248, with the findability caveat in the LOW above.
9. **Template side effects.** `xreview-helpers.sh` is shipped (`templates/base/scripts/xreview-helpers.sh`, mode 755), so the new script's only dependency reaches downstream — the gap is that it is not in the required-file list (LOW). `templates/base/docs/quality/quality-gates.md` is free of meta-repo references: it omits the issue number and names only paths that exist downstream. `check-template-purity.sh` passes.
10. **Comments.** No history narration and no bare finding ids anywhere in the shipped code. The one incident reference is `PR #168 shape` in a test header comment explaining *why* the fixture needs two commits — that earns its place. Comment volume on the new script is high (22 header lines for ~130 lines of code) but every block explains a decision a reader would otherwise have to reconstruct (why default mode does not fail, why the allowlist comes from HEAD, why the mode gate excludes `test`). I would not cut any of it.

## Recommendation

- Merge: **yes, with the two MEDIUM findings addressed first.** No CRITICAL and no HIGH. Both MEDIUMs are small, localised edits — a failure line in `run-verify.sh` and a gate that reads the status line instead of the exit code in the four `/pr` faces — and both sit on the truthfulness of a gate that exists to be trusted, which is the wrong place to carry a known gap.
- Follow-ups: the LOW items are individually optional; the three worth batching if they are not taken now are the required-file asymmetry (`xreview-helpers.sh`), the `findings` label on a non-`1` scanner exit, and the four unpinned exit-0 test assertions. If any LOW is deliberately deferred, record it as one tech-debt row rather than letting it evaporate.
- Known gaps: no end-to-end run of the two new test files (deferred to `/test`); no timing measurement of the range scan on a long branch; commit-signing hermeticity in the fixture repos is a pre-existing suite-wide convention, not addressed here.

## Cycle 2

- Date: 2026-09-24
- HEAD: e71feb0 (`git status --porcelain` empty at review start and at review end)
- Plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md (Scope row 1, AC-2d, AC-3b)
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for the cycle-2 delta `git diff 814abba...HEAD` (4 files, +555/-14: `scripts/secret-scan-branch.sh`, `templates/base/scripts/secret-scan-branch.sh`, `tests/test-secret-scan-branch.sh`, the plan), read against the whole branch `git diff main...HEAD` and the full current script. No spec-compliance verdict, no test verdict, no documentation-drift audit. Cycle-1 findings are not re-opened; none was reintroduced.
- Finding ids in this section are prefixed `C2-` so they cannot be confused with cycle 1's `MEDIUM 1` / `LOW n` numbering in the same file.

### Evidence reviewed

- `git log --format='%h parent=%p'` over the delta: `e71feb0 parent=dbba825`, `dbba825 parent=814abba` — linear, and the fix commit precedes the plan commit, so the plan's AC-2d / AC-3b ticks describe the tree that ships.
- Full read of `scripts/secret-scan-branch.sh` at HEAD (286 lines), the delta diff for all four files, and the cycle-1 cross-review triage report's AR-1 / AR-2 rows.
- `cmp scripts/secret-scan-branch.sh templates/base/scripts/secret-scan-branch.sh` — byte-identical. `git diff --stat 814abba...HEAD` — nothing outside the three code files and the plan.
- `sh tests/test-secret-scan-branch.sh` — PASS 87, FAIL 0.
- `./scripts/secret-scan-branch.sh --strict` run in this worktree: `scanned a0d5bf5..e71feb0 against origin/main: clean` (rc 0). The product scanning its own branch is the evidence that the new fixtures carry no secret-shaped literal in committed history; a source grep for whole-token shapes also returns nothing, and every fixture value is assembled at runtime from split `printf` pieces (21 sites).
- **`git ls-tree` / `git show` probes** in a throwaway fixture repo (deleted afterwards, worktree untouched):

  | pathspec | `git ls-tree --full-tree HEAD --` output | mode | path equals query? |
  |---|---|---|---|
  | `rules` (bare directory) | one line, the tree entry itself | `040000` | **yes** |
  | `rules/` (trailing slash) | three lines, the directory's children | (child's) | no |
  | `.` | three lines, the root's children | (child's) | no |
  | `/rules` (absolute) | `fatal: ... is outside repository`, rc 128 | — | — |
  | `rules/<non-ascii>` | one line, path printed C-quoted | `100644` | no |
  | `./README.md` | one line, path printed normalised as `README.md` | `100644` | no |

  `git show HEAD:rules` and `git show HEAD:rules/` both exit 0 and print a tree listing.
- **Parser probe** of `${entry#*"$tab"}` / `${entry%% *}` / the `*"$newline"*` case under `/bin/sh` (macOS) and `/bin/dash`, over six synthetic entries: identical results in both shells, all correct.
- **Guard deletion matrix** (`git archive HEAD | tar -x` into the scratchpad, one mutation at a time, full suite each run, copy deleted afterwards):

  | mutation | suite result |
  |---|---|
  | drop the multi-line (`newline`) check | PASS 87 / FAIL 0 — **not pinned** |
  | drop the exact-path check | PASS 85 / FAIL 2 — pinned |
  | drop `--full-tree` | PASS 84 / FAIL 3 — pinned |
  | drop the `allowlist_target_is_safe` call | PASS 87 / FAIL 0 — **not pinned** |
  | drop the `./`-stripping loop | PASS 87 / FAIL 0 — **not pinned** |
  | widen the mode case to accept `040000` | PASS 87 / FAIL 0 — **not pinned** |
  | baseline | PASS 87 / FAIL 0 |
- **Fail-open demonstration** for the last mutation: with the mode case widened to accept `040000` and every other guard intact, a fixture whose committed `.gitallowed` is a symlink to a bare directory name runs `--strict` to **rc 0, "scanned ... clean"** while the planted finding is present. The allowlist file then holds the tree listing, and one of its lines matches the leak line as a regex.
- **Pre-fix contrast run** of the same symlink fixture against `814abba:scripts/secret-scan-branch.sh`: rc 1 with the "ignoring uncommitted .gitallowed edits" notice, versus rc 0 and no notice at HEAD.

### Findings

#### C2-M1 (MEDIUM) — the function comment credits the wrong guard for a bare directory target

`scripts/secret-scan-branch.sh:194-199` (and the byte-identical `templates/base/scripts/secret-scan-branch.sh:194-199`; the same claim is in the body of commit dbba825).

The comment reads: "Requiring a single entry whose path matches exactly guards against a directory pathspec (bare, or with a trailing slash): `git ls-tree` lists a directory's CHILDREN instead of failing, so without this check a symlink target naming a directory could pass the mode check via its first child's mode."

All three clauses are true for `rules/` and false for `rules`. For the bare form `git ls-tree --full-tree HEAD -- rules` prints exactly one line, the tree entry itself, whose recorded path is `rules` — so it passes the single-line check at `:203-205` and the exact-path check at `:206`, and no child's mode is ever involved. The guard that actually rejects it is the mode `case` at `:225` (top level) and `:237` (symlink target), because `040000` is not in `100644|100755`.

Why it matters: the mode list at `:237` is the only thing standing between a bare-directory symlink target and a fail-open, and the deletion matrix shows no test pins it. The comment directs a future maintainer away from that line — and the correct taxonomy already sits twenty lines below, at `:219-221` ("Anything else (no entry, a directory, a submodule, or a symlink target that cannot be resolved to a readable file) falls back to an empty allowlist"), so the file contradicts itself.

Fix: narrow the parenthetical at `:195` to the trailing-slash form, and add one clause saying the mode `case` below is what rejects a bare directory (`040000`) and a submodule (`160000`). Apply to both copies; reword the commit body only if the commit is amended for another reason.

#### C2-L1 (LOW) — the regular-file arm's `git show` is unguarded next to a guarded sibling

`scripts/secret-scan-branch.sh:226` vs `:238`.

`git show "HEAD:.gitallowed" > "$tmp_allowlist"` at `:226` has no `if` and no `2>/dev/null`, while the same command for the symlink target at `:238` is wrapped in `if ...; then allowlist_resolved=1; fi`. Under `set -eu` a failure at `:226` (a damaged object, or a redirection failure because `$TMPDIR` disappeared) exits with git's or the shell's own status — 128, or 1 — and 1 is the code this script's own contract at `:24-29` reserves for "scanned and found something".

Not a regression: the pre-fix code was equally unguarded. The fix introduced the asymmetry by writing a guarded sibling twelve lines away without saying why the two differ.

Fix: give `:226` the same shape — on failure, `: > "$tmp_allowlist"` plus the existing notice — or add one sentence saying the mode check already proved the blob exists and a failure here is a broken repository.

#### C2-L2 (LOW) — the path is validated before it is normalised, so normalisation can undo the validation

`scripts/secret-scan-branch.sh:231` (validate) runs before `:232-234` (strip leading `./`).

A symlink target of `.//x` passes `allowlist_target_is_safe` (it does not begin with `/`, and it holds no `..` component), and the loop then strips one `./` to yield `/x`, an absolute path the validator exists to reject. Probed under `/bin/sh`: `target=[.//x] SAFE stripped=[/x]`.

Harmless at HEAD only because `git ls-tree --full-tree HEAD -- /x` exits 128 with "is outside repository", so `allowlist_ls_tree_mode` returns 1 and the scan falls back to an empty allowlist. The protection is git's, not the script's.

Fix: move the stripping loop above the `allowlist_target_is_safe` call so the validator sees the value that is actually used.

#### C2-L3 (LOW) — the multi-line guard is unreachable as the deciding check and is not pinned

`scripts/secret-scan-branch.sh:203-205`.

Deleting the `case "$entry" in *"$newline"*) return 1` arm leaves the suite at 87/0. I could not construct an input where it decides: a multi-line `entry` makes `${entry#*"$tab"}` a multi-line string, which cannot equal the single-line pathspec passed as `$1`, so `:206` rejects the same inputs. The comment at `:182-183` presents both variables as parsing machinery, which reads as if the guard is load-bearing.

Keep the guard — it is cheap and it is correct. Fix: one clause at `:203` saying it is belt-and-braces ahead of the exact-path check, so nobody later relies on it as the sole protection.

#### C2-L4 (LOW) — eight of nine new test names lead with a cross-review finding id that will be reassigned

`tests/test-secret-scan-branch.sh:487, 528, 792, 821, 850, 892, 968, 1002`.

Eight new test functions are prefixed `test_ar1_` / `test_ar2_`. Those ids live in `docs/reports/cross-review-triage-local-branch-secret-scan.md`, which `.claude/skills/cross-review/SKILL.md:97` writes to a single per-slug path — cycle 2's `/cross-review` run will overwrite it with its own AR numbering, at which point `ar1` and `ar2` in these names point at different findings.

The file already has the stable alternative: `test_ac1_deleted_secret_still_found` (pre-existing) cites a plan acceptance criterion, and each new test's own comment already leads with `AC-3b (...)` or `AC-2d (...)` before naming the AR. Severity stays LOW because every one of these comments restates the defect in full prose, so the id is decoration rather than the only pointer.

Fix: rename to the AC or to the behaviour — `test_ac3b_tag_shadows_base_branch`, `test_ac2d_symlinked_allowlist_resolves_target`, and so on — and leave the AR citation in the comment where it already is. The ninth new test, `test_ls_tree_mode_check_is_root_relative:925`, already does this correctly.

#### C2-L5 (LOW) — the ref-disambiguation rationale is told twice at full length, and the notice string is duplicated verbatim

`scripts/secret-scan-branch.sh:9-14` and `:144-150`; `:246` and `:254`.

The header paragraph and the inline comment above the `merge-base` call state the same git resolution order (`refs/<name>`, `refs/tags/<name>`, `refs/heads/<name>`, `refs/remotes/<name>`) and the same consequence, in six and seven lines respectively. Separately, the notice `"committed .gitallowed could not be read as a file; scanning without allowlist exceptions"` is written out in full at `:246` and again at `:254`.

Cycle 1 examined comment volume on this script and explicitly declined to cut any of it, so this is not a re-open of that: the delta added a *second full telling* of one rationale, taking the file from 52 to 100 comment lines against 111 to 163 code lines. Two tellings drift independently, which is the mechanism behind C2-M1.

Fix: keep the ordering rationale at the call site (`:144-150`) and shorten the header to the contract sentence it already has at `:9-11`; hoist the notice into a one-line helper or a variable.

### Positive notes

1. **Both cross-review findings are genuinely closed, and the tests discriminate.** For AR-2 I traced both fixtures by hand: in the tag test the pre-fix `merge-base HEAD main` resolves through `refs/tags/main` to a commit *after* the planted leak, so the pre-fix range excludes it and the scan reports clean; post-fix `refs/heads/main` restores the full range. The `origin/<base>` local-branch test has the same shape through `refs/heads/origin/main`. For AR-1, `test_ar1_symlinked_allowlist_target_name_not_used_as_regex:892` is the exact reproduction — pre-fix the link target string becomes the allowlist rule and matches the planted line.
2. **`base_ref_full` is used at every resolving call, and nowhere else.** `git merge-base` at `:151` takes it; `git rev-parse --verify` at `:134` and `:137` were already fully qualified; `git rev-list` at `:155` and `git rev-parse --short` at `:164` take the merge-base SHA. `base_ref` (short form) appears only inside `cannot_scan` / `nothing_to_scan` / `report` strings. No short-name resolution is left in the file.
3. **The fix silently closed the cycle-1 `/test` report's gap (b).** That report recorded the "ignoring uncommitted .gitallowed edits" notice as potentially misfiring for a symlinked `.gitallowed`. Running the same fixture against `814abba:scripts/secret-scan-branch.sh` reproduces the misfire (plus a real false positive, since the link target string was the allowlist); at HEAD the same fixture is clean with no notice. That gap was recorded in the report only and never entered `docs/tech-debt/README.md`, so there is no register row left asserting a state this PR removed. Flagging it for the cycle-2 `/test` re-run rather than editing the cycle-1 artifact.
4. **Pathspec magic fails closed, which the comment does not claim but which holds.** Targets such as `:/rules/deploy` and `:(glob)*.md` pass `allowlist_target_is_safe`, but `git ls-tree` then reports the *resolved* path, which never equals the magic string, so `:206` rejects them. Same mechanism protects a glob target and a C-quoted non-ASCII target.
5. **Comment reflow is consistent.** No comment line added by this delta exceeds 75 columns; the only over-wide comment lines in the file (`:4` at 81, `:39` at 76) are pre-existing.
6. **Exit and cleanup structure is untouched and still correct.** The EXIT trap is installed before `tmp_allowlist` is assigned, `cleanup` guards the empty value and ends with `return 0`, and the signal traps at `:77-79` still exit explicitly.
7. **The commit message of dbba825 describes what the code does**, in the right order (AR-2 then AR-1 then the two adjudication amendments), with the one over-generalisation noted in C2-M1. The plan's own deviation note is more precise than the commit body: it restricts the directory case to the trailing-slash form, which is correct.
8. **Fixture hygiene holds for the new tests, including the symlink target names.** The one test that makes a token the *symlink target* still builds it from split pieces at `:892`, so nothing secret-shaped appears on a source line.

### Coverage gaps

These belong to `/test`, not to this report's verdict; recording them because the deletion matrix makes them concrete.

1. **No test covers a bare-directory symlink target** (`ln -s rules .gitallowed`). The only directory test uses the trailing slash, which a different guard rejects. Widening the mode case to accept `040000` keeps the suite at 87/0 while turning `--strict` into a silent rc 0 on a branch that carries a finding. A fixture mirroring `test_ar1_symlinked_allowlist_directory_target_rejected:968` without the trailing slash would pin it.
2. **`allowlist_target_is_safe` is entirely unpinned** — removing the call leaves the suite green. Neither the absolute-path rejection nor the `..` rejection nor the empty-target rejection has a fixture. Today git's own pathspec handling refuses those inputs, so the guard is defence in depth; that is exactly the kind of guard a later refactor removes.
3. **The `./`-stripping loop is unpinned.** No fixture uses a `./`-prefixed target. Without the loop such a target degrades to an empty allowlist (fail-closed, false-positive direction), so the loop adds working behaviour that nothing exercises.
4. **Two tree-entry modes reachable through the symlink branch are untested**: a submodule (`160000`) and a symlink pointing at another symlink (`120000`). Both fall to the empty-allowlist path by the mode check alone.
5. **`test_ar1_symlinked_allowlist_denies_fixture:821` does not discriminate.** Pre-fix the allowlist content would be the literal target path, which also fails to match the planted line, so the test exits 1 either way. It is a useful negative control for the resolving test; it does not protect the fix.

### Answers to the requested points

1. **Strict exit 0 reachable only through the clean line — confirmed.** Every `exit` in the file is at `:48` (usage, 2), `:77-79` (signal traps), `:96`/`:98` inside `report_and_exit` (3 under `--strict`, 0 otherwise), `:272` (the clean line) and `:285` (the scanner's own code). The new allowlist block contains no `exit` and no early `return` outside the two helper functions, and it runs after `mktemp`, so the EXIT trap still owns cleanup. One caveat, C2-L1: the unguarded `git show` at `:226` can exit with git's own status under `set -e`.
2. **`allowlist_ls_tree_mode` inputs — one comment defect, no behaviour defect.** A bare directory passes both the single-line and exact-path checks and is stopped by the mode case (C2-M1). A trailing-slash directory, `.`, a submodule, a second-level symlink, a C-quoted path (non-ASCII, quote, backslash) and a path containing a newline all end at an empty allowlist plus the one notice line, which is the intended direction. The `newline` and `tab` definitions and both parameter expansions behave identically under `/bin/sh` and `/bin/dash` (probe table above).
3. **`allowlist_target_is_safe` and the `./` loop — one ordering defect (C2-L2).** Absolute, empty, `..`, `a/../b` are rejected; `.` and `dir/` are accepted by the validator and then rejected downstream by the exact-path check; `.//x` becomes absolute after stripping and is saved only by git refusing the pathspec.
4. **The notice block stays sensible for a symlinked worktree `.gitallowed`, and the cycle-1 misfire is gone — checked, no issue.** `test -f` and `diff -q` both dereference, so a worktree symlink is compared by its target's content against the resolved committed content, which is the right comparison. I could not construct a case where the notice fires without a genuine committed-versus-worktree difference, and the one case that always misfired pre-fix now does not (positive note 3).
5. **`base_ref_full` is used for every resolving call and `base_ref` only for display — checked, no issue** (positive note 2). No other short-name resolution remains in the file.
6. **Tests — hermetic, correctly named except for the id prefix, and mostly discriminating.** No secret-shaped literal anywhere in the source, including symlink target names; the branch's own strict scan is clean. `HOME` / `GIT_CONFIG_GLOBAL` / `GIT_CONFIG_SYSTEM` are isolated at the top of the file and every new test builds its own repo, so none depends on host refs. The deletion matrix shows the exact-path check and `--full-tree` are pinned; the other three new guards are not (coverage gaps 1-3). Test names are specific but lead with a volatile id (C2-L4). Suite run: PASS 87, FAIL 0.
7. **Comments — one inaccurate claim (C2-M1), one duplicated rationale (C2-L5), otherwise clean.** No history narration, no bare finding ids in either shipped script, no reference to in-branch intermediate designs. Density rose from 0.46 to 0.61 comment lines per code line; cycle 1 ruled on volume and I am not re-opening that, only the second telling. The header at `:9-20` matches behaviour. The commit message matches the code with the C2-M1 over-generalisation.
8. **Template copy byte-identical and delta scope clean — checked, no issue.** `cmp` reports no difference; `git diff --stat 814abba...HEAD` lists exactly the three code files and the plan.

### Recommendation

- **Merge after fixes.** No CRITICAL, no HIGH. The one MEDIUM (C2-M1) is a two-line comment edit in two byte-identical files, and it sits on the one guard in the new code that nothing tests, which is the wrong place to leave a misleading pointer.
- **This is the last cycle at the default cap.** C2-L1 through C2-L5 are individually optional, but an unfixed LOW at the cap is a deferral, and `/pr` archives the plan. If they are not taken now, record them as one tech-debt row in `docs/tech-debt/README.md` citing this section, together with coverage gaps 1-4 — the bare-directory fixture is the item with real protective value.
- **Known gaps:** no test run beyond this suite (`/test` owns that); no spec-compliance verdict and no documentation-drift audit (`/verify` and `/sync-docs` own those); the cycle-1 `/test` report's gap (b) is stale at HEAD and is handed to the cycle-2 `/test` run rather than edited here.
