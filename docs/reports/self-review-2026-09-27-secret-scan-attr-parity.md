# Review report: secret-scan-attr-parity (self-review, cycle 1)

- Date: 2026-09-27
- Plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD` at ed9f588 (code commits 7e0c529 slice A, beb19f1 slice B, ad8c475 slice C; plan commits 8335ccb, 972c472, ed9f588), plus full reads of `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh`. No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- `git diff main...HEAD`: 14 files, +1131/-74. HEAD ed9f588 and `git status --porcelain` empty at review start and after every probe. `cmp` confirms `scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`, and the four `/pr` skill copies are byte-identical to their mirrors. The two `quality-gates.md` copies differ only in the pre-existing KNOWN_DIFFS lines (the template has no tech-debt pointer and no `check-sync.sh` row, as on main).
- Callers read: `.github/workflows/verify.yml:8-34` (checkout `fetch-depth: 0`, the Secret scan step, `run-verify.sh`), `scripts/run-verify.sh:217-253`, `scripts/prepare-commit-msg-secret-guard.sh`, `internal/cli/git_hooks.go:27` (the guard is installed by `ralph init`).
- Suites run as requested: `sh tests/test-secret-scan.sh` passed 123/123 and `sh tests/test-secret-scan-branch.sh` passed 163/163 (macOS `/bin/sh`, which is bash 3.2 here, git 2.49.0). Both also passed (123/123, 163/163) in `ubuntu:24.04`, where `/bin/sh` is dash and git is 2.43.0, run as a non-root user.
- Product run on this branch: `./scripts/secret-scan-branch.sh --strict` printed `scanned 03f3e8a..ed9f588 against origin/main: clean` (rc 0).
- Mutation spot-checks, each on a scratch copy of `scripts/` and `tests/` outside the worktree, one mutation per copy:

  | Mutation | Tests that went red |
  | --- | --- |
  | scanner: drop the `"$@"` pin list from `git log` | 5 (every `diff.<driver>.binary` case) |
  | scanner: drop the `check_info_attributes` call | 6 (every info/attributes case) |
  | scanner: ignore `RALPH_SECRET_SCAN_ATTR_SOURCE` | 6 (every attribute-source case) |
  | branch: pathspec `:(top).gitattributes` (root only) | 4 (nested change, both cwd variants) |
  | branch: never call `use_merge_attributes` | 31 |
  | branch: merge-driver listing rc 0 not recognized | 4 (the reason assertions) |
  | branch: a conflicting merge (rc 1) uses the tree | 3 (every conflict case) |
  | scanner: pin `auto` instead of `false` (candidate fix for M1) | none, 123/123 |
  | scanner: add `GIT_ATTR_NOSYSTEM=1` (candidate fix for M2) | none, 123/123 |

  The first four scanner mutations were first run in parallel and produced extra failures unrelated to the mutation. `expect_exit` in `tests/test-secret-scan.sh` writes to fixed `/tmp/ralph-secret-scan-test.{out,err}` paths, so concurrent runs overwrite each other. That is pre-existing and not in this diff. The table above is from a sequential re-run.
- Probes (fixture repos under the scratchpad, isolated HOME and global config; containers for system-level files):
  - A file with a NUL byte and a token line under a committed `diff=foo`, with local `diff.foo.binary=true`. CI-like run (no driver config): scanner rc 0 and `git log -p` prints `Binary files ... differ`. This branch's scanner: rc 1. `-c diff.foo.binary=auto` gives the same output as no config on git 2.49.0, 2.45.4 (`alpine:3.20`), and 2.8.6 (`alpine:3.4`).
  - `/etc/gitattributes` holding `*.txt -diff` (git's system-level attributes file) in `alpine:3.20` and `alpine:3.4`: the scanner returns rc 0 for a token in `leak.txt`, and rc 1 with `GIT_ATTR_NOSYSTEM=1`.
  - Merge guard: a real `git merge --no-edit other` with the guard installed as `prepare-commit-msg` and `.git/info/attributes` holding one `*.md linguist-documentation` line. The merge commit is not created (`Not committing merge`, rc 1), and the same merge succeeds once the file is removed.
  - Driver names `MyDrv` and `q"t` (the plain log hides the token, the scanner pins and finds it, rc 1); config keys holding a space or a backslash are accepted by `-c` (no exit 3).
  - Config listing covers `includeIf.gitdir:` includes, `GIT_CONFIG_PARAMETERS` (which is where `--config-env` lands), and `GIT_CONFIG_COUNT`; a later `-c` wins over an inherited `GIT_CONFIG_COUNT` key.
  - `.git/info/attributes` shapes: a symlink to a directory exits 3; CR-only and whitespace-only lines scan; a UTF-8 BOM before a comment, a rule line over 2048 bytes (which git ignores with a warning), and a vertical-tab-prefixed line all exit 3 (over-blocks only); from a subdirectory `--git-path` prints a relative path that still resolves.
  - Base-change detection: a rename of `d/.gitattributes` away, a mode-only change, and `a/b/c/.gitattributes` all count as changes; `.gitattributes.bak` does not. A branch where only HEAD changed `.gitattributes` scans clean without running merge-tree (a stub that fails on merge-tree confirmed it is never called).
  - `git_version_at_least 2 41`, extracted verbatim: `2.39.5 (Apple Git-154)` no, `2.40.1` no, `2.41.0` yes, `2.41.0.windows.1` yes, `2.100.0` yes, `3.0.0` yes, `2`, `2.x`, and empty all no.

## Findings

### M1 (MEDIUM): `diff.<driver>.binary` is pinned to `false`, which makes the local scan read binary content that CI skips

- Where: `scripts/secret-scan.sh:348` and `:355` (the pin), `:278-286` (comment: "CI has no driver config, so every driver is text there"), header `:6-10` ("a diff driver's binary flag ... to git's defaults"); `docs/quality/quality-gates.md:46` ("so a local run scans the same added lines as CI"); `docs/tech-debt/README.md:139` ("is set back to false").
- What: CI has no `diff.<driver>.*` config, so git falls back to its own content check. A file whose content holds a NUL byte is binary in CI, and `git log -p` prints `Binary files ... differ`. `false` forces text instead. Probe: a NUL-byte file carrying a token under a committed `diff=foo`, with a local `diff.foo.binary=true`. The CI-like scan returned rc 0 and this branch's scanner returned rc 1.
- Why it matters: the pin reads more lines than CI. `/pr --strict` can then block a push that CI passes, and the operator has to rewrite history or allowlist a binary file for no reason. The comment gives a wrong mechanism, since CI does not treat every driver as text, and the header and quality-gates claims ("git's defaults", "the same added lines") do not hold for this case.
- Fix: pin `auto`, which git's `diff.<driver>.binary` parser accepts (a tristate) and which restores the content check. It matches no-config output on git 2.8.6, 2.45.4, and 2.49.0, and the whole suite passes with it (123/123). Rename `RALPH_SECRET_SCAN_FALSE` to match (for example `RALPH_SECRET_SCAN_BINARY_AUTO`), correct the comment and the tech-debt wording, and add one fixture: a NUL-byte file with a token under the pinned driver, expecting rc 0. Today no test tells `false` from `auto`.

### M2 (MEDIUM): git's system-level attributes file is neither pinned nor listed as a gap

- Where: `scripts/secret-scan.sh:340-347` (the subshell environment for `git log`); header `:5-18`; `docs/tech-debt/README.md:139` (the "Local-only ways" row); `docs/quality/quality-gates.md:46`.
- What: git reads `$(prefix)/etc/gitattributes` (`git var GIT_ATTR_SYSTEM`; `/opt/homebrew/etc/gitattributes` for this machine's git) under every setting the scanner pins. `core.attributesFile=/dev/null` covers only the user-level file. Probe in `alpine:3.20` (git 2.45.4) and `alpine:3.4` (git 2.8.6): `/etc/gitattributes` with `*.txt -diff` made the scanner return rc 0 for a token CI finds; `GIT_ATTR_NOSYSTEM=1` restored rc 1.
- Why it matters: this is the fail-open direction the PR exists to close (local clean, CI finding), and #181's objective is exactly the attribute sources. The header now enumerates what it pins and names `.git/info/attributes` as the one source it cannot pin. The tech-debt row presents the remaining local-only ways as a closed list (replace refs, git older than 2.41). Neither mentions this source, so a reader would conclude it is covered. The trigger is rare, since the file does not exist on this machine, but the fix is one line.
- Fix: `GIT_ATTR_NOSYSTEM=1; export GIT_ATTR_NOSYSTEM` in the `git log` subshell. It works on git 2.8.6, and the suite passes with it. CI runs the same scanner, so both sides drop the system file and parity holds by construction. Add it to the pin comment and header, and add a test that points `GIT_ATTR_NOSYSTEM`'s effect at a fixture. A test cannot write the real system file, so assert the variable reaches `git log` through a PATH wrapper that echoes it, the same way the existing wrappers work.

### M3 (MEDIUM): the `.git/info/attributes` refusal also blocks every local non-fast-forward merge, which no doc or plan item mentions

- Where: `scripts/secret-scan.sh:327` (`check_info_attributes` runs for every `--range`, not only ranges ending at HEAD); `scripts/prepare-commit-msg-secret-guard.sh:15` (the merge guard's `--range HEAD..<merge head>`); `internal/cli/git_hooks.go:27` (`ralph init` installs that guard in every scaffolded project).
- What: probe with the guard installed as `prepare-commit-msg` and one unrelated line (`*.md linguist-documentation`) in `.git/info/attributes`. `git merge --no-edit other` printed the scanner's reason and `Not committing merge`, and returned rc 1. The same merge succeeds without the file. `--no-verify` does not skip `prepare-commit-msg`, so a user cannot complete a merge (or a `git pull` that merges) until they empty the file.
- Why it matters: the refusal is consistent in logic, because the merge guard also reads attributes CI does not have. But it is a new blocking behavior on a far more frequent operation than `/pr`, and it ships to every downstream project. The plan's Risks line says only that "the local scan" stops. `quality-gates.md`, the `/pr` skill, and the tech-debt rows describe the push gate only. Tools that write `.git/info/attributes` by design (git-annex writes a `filter=annex` rule there) would turn every merge into a hard stop.
- Fix: decide the surface explicitly. Either (a) keep it and say so where operators look: the scanner header, `quality-gates.md` next to the pre-commit/merge hooks, and the plan's Risks. The reason line already says what to do. Or (b) limit the check to ranges ending at HEAD (the CI-parity callers) and record the merge-guard gap as a tech-debt row. Option (a) keeps fail-closed and needs only documentation plus a merge-guard test in `tests/test-secret-scan.sh`, next to the existing guard cases at `:264-270`.

### L1 (LOW): the new refusal reasons do not name the remedy that always works

- Where: `scripts/secret-scan-branch.sh:325-330` (`attributes_unguaranteed`), `:370`, `:381`, `:405`; `.claude/skills/pr/SKILL.md:29` (and its three mirrors).
- What: when the base changed `.gitattributes` and the merge cannot be computed like CI's (old git, a local merge driver, merge-tree failing), `--strict` exits 3 with a reason that names the cause but not a way out. Merging or rebasing onto the base always clears it, because the merge-base then equals the base tip and HEAD's attributes are the merge's (the "already merged" test pins this). On any git older than 2.41, every `/pr` after a base-side `.gitattributes` change stops here until the operator upgrades git or updates the branch. A global `merge.<name>.driver` (for example `merge.ours.driver=true`) stops it too, even when no committed attribute names that driver. The `/pr` skill's remedy list ("`git fetch origin <base>`, confirm the base, or move `.git/info/attributes` rules") does not cover these cases. Its "could not scan" list also omits merge-tree failing.
- Fix: append a hint to the strict reason, such as `; merging or rebasing onto <base> makes HEAD's attributes the merge's`, and add the same remedy to the `/pr` clause.

### L2 (LOW): the scanner test's hermetic block does not clear `RALPH_SECRET_SCAN_ATTR_SOURCE` or `GIT_ATTR_SOURCE`

- Where: `tests/test-secret-scan.sh:106` (`unset XDG_CONFIG_HOME RALPH_SECRET_ALLOWLIST`).
- What: the new "attribute source: unset reads HEAD's attributes" case and every range case ending at HEAD depend on the caller's environment. An exported `RALPH_SECRET_SCAN_ATTR_SOURCE` would make them read another tree or exit 3. An exported `GIT_ATTR_SOURCE` would change what `expect_plain_log_hides` (plain `git log`) validates. The branch test is not affected, because `secret-scan-branch.sh` always passes the variable.
- Fix: add both names to the `unset` on `:106`.

### L3 (LOW): a failed `mktemp` for the merge stderr file exits 1, the "findings" code

- Where: `scripts/secret-scan-branch.sh:384`.
- What: under `set -e`, a failing `mktemp` (an unwritable `TMPDIR`) ends the script with rc 1 and no status line. `/pr` reads exit 1 as "findings". This is the same shape as the pre-existing allowlist `mktemp` at `:254`, and the new site copies it. It fails closed, but with the wrong label.
- Fix: `tmp_merge_err="$(mktemp ...)" || cannot_scan "could not create a temporary file"`. Alternatively, drop the file: capture `2>&1` and read the first line only on a non-zero rc. A tree id that is not the first line on rc 0 already fails closed through the `rev-parse --verify` check.

## Positive notes

- Every new decision point has a test that goes red when it is reverted (the mutation table). The stubs for old git, merge-tree without `--write-tree`, and merge-tree failing are scoped to one command through `env PATH=...`, so they cannot leak.
- `--config-env` splits at the last `=` and `-c` at the first. The key-with-`=` test covers exactly that asymmetry, including the git-older-than-2.31 branch.
- The info/attributes check reproduces git's blank/comment rule (`" \t\r"`), fails closed on anything it cannot read, and treats a dangling symlink the way git does.
- The change detection uses the fully qualified base ref, a root-relative glob pathspec with `diff.relative=false`, and `--no-ext-diff --no-textconv`. It fails closed on any rc above 1, a deviation the plan records.
- merge-tree rc 1 is the only fallback that reads HEAD's attributes; every other rc is classified, and the tree id is re-verified with `^{tree}`.
- All fixture tokens are assembled from split pieces (`printf 'ghp_%s' '...'`); no secret-shaped literal appears in the diff.

## Coverage gaps

- M1 and M2: no fixture tells `false` from `auto`, and none exercises a system attributes file (the candidate fixes pass the suite unchanged).
- M3: no test runs the merge guard with a `.git/info/attributes` rule.
- Not probed: criss-cross histories with several merge-bases (`git merge-base` returns one, and the change detection compares against that one only); a race in CI where `origin/<base>` moved after GitHub built the merge commit, which can make `run-verify.sh`'s default-mode branch scan run merge-tree (notice-only, and the authoritative Secret scan step is unaffected).
- The 219,578-line whole-history comparison (AC-8) was not re-run; the default-config path was checked by reading (with no `diff.*.binary` keys the pin list is empty and `GIT_ATTR_SOURCE` is HEAD's commit, as on main).

## Answers to the requested checks

1. **Fail-closed.** The attr-source handling, the key enumeration (includes, `--config-env`, `GIT_CONFIG_COUNT`, key spellings), the info/attributes shapes, the pathspec, the merge-driver regex, the version parsing, the merge-tree rc classification, and the first-line parsing all hold; the one new clean-while-CI-finds path is git's system attributes file (M2).
2. **Over-blocking.** Three over-blocks: info/attributes rules unrelated to diff now stop `/pr` and every local merge (M3); the `false` pin blocks binary content CI skips (M1); the old-git and merge-driver refusals are proportionate given how rarely the base changes `.gitattributes`, but their reasons lack the remedy (L1).
3. **Shell correctness.** Checked, with no issue beyond L3: dash and bash-3.2-as-sh both pass, `set --` is scoped to the subshell, an empty `"$@"` is safe under `set -u`, keys with spaces stay one word, and the EXIT trap (with exiting signal traps) removes the merge stderr file on every path.
4. **CI parity.** Checked, no issue: CI's Secret scan step calls the scanner directly, and `run-verify.sh`'s default-mode branch scan sees merge-base equal to the base tip, so no attribute diff and no merge-tree; with no driver keys and no info/attributes in CI's clone, the scanner's default path is unchanged.
5. **Tests.** Hermetic apart from L2; the stubs cannot leak; no secret-shaped literal; names describe what they protect; seven reverts each went red. Gaps: `false` vs `auto` (M1), the system file (M2), the merge guard (M3).
6. **Comments and docs.** Accurate and proportional except three spots. The binary pin's mechanism and the "git's defaults" and "same added lines" claims are wrong (M1). The closed list of local-only gaps omits the system attributes file (M2). The `/pr` clause lacks the merge/rebase remedy (L1). No bare finding ids in the new code text, and no history narration in the headers.

## Recommendation

No CRITICAL or HIGH findings, so this is not a blocker. Fix M1 and M2 in this cycle: each is a one- or two-line code change with one fixture, and both candidate fixes already pass the full suite. Resolve M3 by documenting the merge-guard effect or by scoping the check, and record which one. L1 to L3 are small and can ride along or go into one batched tech-debt row.

## Cycle 2

- Date: 2026-09-28
- Scope: diff quality only for `git diff 5107fb2...HEAD -- scripts/ templates/ tests/ docs/` at d13da43. The code commits are f2ceeca (Slice D, the cycle-1 fixes) and fe4f383 (Slice E, cross-review cycle-1 AR-1 and AR-2); the rest of the delta is pipeline reports and plan notes. `git diff main...HEAD` was read for context only. HEAD d13da43 and `git status --porcelain` empty at the start and after every probe.
- Finding ids below are `C2-` plus a severity letter and a number, and they belong to this section only.

### Evidence reviewed

- Mirrors: `cmp` shows `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` byte-identical to their `templates/base/scripts/` copies, and the four `/pr` skill copies identical.
- Suites on the worktree (macOS `/bin/sh`, git 2.49.0): `sh tests/test-secret-scan.sh` 123/123, and `sh tests/test-secret-scan-branch.sh` 227/227 with one SKIP (the real-old-git case, which only runs below 2.41). The old-git runs in the plan (git 2.40.4: 189/0 with 6 SKIP) were not repeated here.
- Deletion matrix for the merge-tree subshell (`scripts/secret-scan-branch.sh:464-476`). Each row removes one pin from a scratch copy outside the worktree and runs the whole branch suite:

  | Pin removed | Failing assertions (of 227) |
  | --- | --- |
  | `GIT_ATTR_SOURCE=$head_commit`, so an inherited value flows through | 4 (working-tree union, inherited union) |
  | the same line replaced by `unset GIT_ATTR_SOURCE` | 2 (working-tree union) |
  | `GIT_ATTR_NOSYSTEM=1` | 0 |
  | `-c core.attributesFile=/dev/null` | 2 (user attributes file with union) |
  | `-c attr.tree=` | 0 |
  | `-c merge.renormalize=false` | 0 |
  | `-c merge.renames=true` | 0 |
  | `-c merge.directoryRenames=conflict` | 0 |
  | `-c merge.renameLimit=7000` | 0 |
  | `core.attributesFile` and `merge.renormalize` together | 3 (adds the renormalize case) |
  | the `check_info_attributes` call | 18 |

- Probes against git 2.49.0 directly, in fixture repos with an isolated HOME and global config:
  - A clean filter named by the committed `.gitattributes`, with a local `filter.hide.clean` and `merge.renormalize=true`: this branch's script returns rc 1 under `--strict`. The same script without `-c merge.renormalize=false` returns rc 0 and prints `scanned ... against main: clean`. Under the default config both return rc 1 with findings.
  - `merge.directoryRenames`: base renames `a/` to `b/`, the branch adds `a/.gitattributes`. `conflict` (the default) gives rc 1, while `true` and `false` both give rc 0, with the file at `b/` and at `a/` respectively.
  - `merge.renames=false` and `diff.renames=false` do not change the result of `git merge-tree --write-tree` for a base-side move of a nested `.gitattributes` that the branch edits (rc 0, the rename followed). Porcelain `git merge` behaves the same way.
  - `diff.algorithm`: a random search found a 3-way input that `git merge-file` merges cleanly under histogram and conflicts under myers. `git merge-tree --write-tree` on that input gives the same blob with `diff.algorithm` unset, myers, histogram, and patience. The comment's claim at `:430` holds on 2.49.
  - `GIT_ATTR_SOURCE` overrides `attr.tree`: `check-attr` reads the attr.tree tree alone (`diff: unset`), and prints `unspecified` once `GIT_ATTR_SOURCE=HEAD` is also set.
  - Default mode with one unrelated `.git/info/attributes` line (`*.md linguist-documentation`) and a token on the branch: main's script rc 1 with findings, cycle 1's (5107fb2) rc 3 (`scanner failed with exit 3`), and HEAD's rc 0 with only the `cannot scan` line.
  - A stub `mktemp` that fails only for the allowlist file: HEAD's script exits 1 under `--strict` and prints nothing.
  - A bare clone whose HEAD commits `*.txt -diff`: `git check-attr diff -- a.txt` in the bare repo prints `unspecified`, so vanilla git in a bare repository reads no in-tree attributes by default.

### Cycle-1 findings at HEAD

| Cycle-1 finding | Status at HEAD | Evidence |
| --- | --- | --- |
| M1 binary pin `false` | Fixed | `scripts/secret-scan.sh:326` pins `auto`; the NUL-byte fixtures at `tests/test-secret-scan.sh:424` and `:426` fail against the old `false` pin |
| M2 system attributes file | Fixed | `scripts/secret-scan.sh:318`; the PATH wrapper test at `tests/test-secret-scan.sh:865` records the value `git log` receives, even with the caller exporting 0 |
| M3 refusal blocks every merge | Fixed | the refusal moved to `check_info_attributes` (`scripts/secret-scan-branch.sh:206-225`); a real merge under the guard hook commits (`tests/test-secret-scan.sh:569-571`); the merge guard's remaining read of the file is recorded in `docs/tech-debt/README.md:139` |
| L1 remedy missing | Fixed | `attributes_unguaranteed` (`:374-380`) and the `/pr` clause name the merge-or-rebase remedy |
| L2 test isolation | Fixed | both suites unset `RALPH_SECRET_SCAN_ATTR_SOURCE` and `GIT_ATTR_SOURCE` |
| L3 mktemp exits 1 | Half-fixed | the merge-tree site at `:454` is guarded; the allowlist site at `:300` is not (C2-L2) |

### Findings

#### C2-M1 (MEDIUM): no test fails when the renormalize pin is removed, although the pin blocks a reachable false clean on its own

- Where: `scripts/secret-scan-branch.sh:471` (the pin); `tests/test-secret-scan-branch.sh:1733-1740` (the test named for it).
- What: the AR-1 regression test selects the clean filter through a user-level attributes file (`:1733-1734`). `-c core.attributesFile=/dev/null` already drops that file, so the case passes with the renormalize pin removed. It also passes with the attributes-file pin removed, and fails only when both are gone (deletion matrix). The renormalize pin also works alone. With the filter named in the committed `.gitattributes`, the script without the pin printed `scanned ... against main: clean` (rc 0) under `--strict`, where the default config reports findings. HEAD's script returns rc 1 there.
- Why it matters: this is the AR-1 failure mode, and the test whose name says "merge.renormalize with a local clean filter" does not guard it. A later edit that drops the pin as redundant with `core.attributesFile` passes 227/227 and reopens AR-1 for any repo that commits a `filter=` attribute.
- Fix: add a case whose filter comes from the committed `.gitattributes` (for example `.gitattributes filter=hide` on both sides, with the local `filter.hide.clean` and `merge.renormalize=true`), expecting rc 1. Keep the user-file case as the `core.attributesFile` test and rename it to say which pin it covers.

#### C2-L1 (LOW): default mode no longer scans at all while `.git/info/attributes` holds a rule, which main did

- Where: `scripts/secret-scan-branch.sh:206-225` (`check_info_attributes` uses `cannot_scan`); header `:37-42`; `tests/test-secret-scan-branch.sh:1807` pins "default mode does not scan"; plan AC-3 ("既定は exit 0").
- What: with one unrelated rule, `run-verify.sh`'s default-mode call now prints `cannot scan, ...` and exits 0 without reading the range. main's script scanned the same fixture and returned rc 1 with findings.
- Why it matters: the early-warning scan disappears for anyone whose `.git/info/attributes` holds a rule, and git-annex writes one by design. `/pr --strict` still stops with exit 3, so nothing gets pushed. The same script already handles the analogous "cannot match CI exactly" case differently: `attributes_unguaranteed` reports the reason and still scans in default mode.
- Fix (a lead decision, since AC-3 fixes the current behavior): in default mode, report the reason and scan anyway, reading the file as git does, like `attributes_unguaranteed`. `--strict` keeps exit 3. Otherwise record the loss of the early warning next to the merge-guard sentence in `docs/tech-debt/README.md:139`.

#### C2-L2 (LOW): the allowlist `mktemp` still exits 1 with no output, and a real temp-dir failure reaches it first

- Where: `scripts/secret-scan-branch.sh:300`; compare the guarded site at `:454-458`; `tests/test-secret-scan-branch.sh:329` (the stub comment).
- What: cycle-1 L3 named both sites. The fix guarded only the merge-tree one. The allowlist file is created about 150 lines earlier on every run, so an unwritable `TMPDIR` fails there first. Probe: `--strict` exits 1 and prints nothing, and `/pr` reads exit 1 as findings. The new guarded branch is reachable only when the temp directory breaks between the two calls, and the test reaches it only through a stub that fails for the merge file alone.
- Fix: `tmp_allowlist="$(mktemp ...)" || cannot_scan "could not create a temporary file for the allowlist"`, or a batched tech-debt row.

#### C2-L3 (LOW): five of the eight merge-tree pins have no test, two of them cannot change anything, and the comment explains the rename pins with behavior merge-tree does not have

- Where: `scripts/secret-scan-branch.sh:466`, `:470`, `:472-474` (pins); `:422` and `:425-429` (comment); `tests/test-secret-scan-branch.sh:1771`.
- What: per the deletion matrix and probes:
  - `GIT_ATTR_NOSYSTEM=1` has no test, although the scanner's suite already has a PATH-wrapper pattern for exactly this (`tests/test-secret-scan.sh:865`).
  - `merge.directoryRenames=conflict` changes the outcome: `false` and `true` turn the default conflict into rc 0. No test covers it. The effect is limited to merges that conflict under git's defaults, where CI does not run.
  - `-c attr.tree=` cannot change anything while `GIT_ATTR_SOURCE` is set, and it is always set here.
  - `merge.renames` is not honored by merge-tree on 2.49 for the probed shape (false still follows the rename), so the comment's "merge.renames otherwise follows diff.renames" describes config semantics that do not show up here. `merge.renameLimit` was not probed.
  - The case at `:1771` (inherited `GIT_ATTR_SOURCE` marking `*.txt -diff`) passes with every merge-tree pin removed. It exercises the scanner's own override, not Slice E.
- Why it matters: the deletion matrix leaves the next maintainer no way to tell load-bearing pins from belt-and-braces ones, and the comment suggests every pin was needed for the same reason.
- Fix: add a wrapper test for `GIT_ATTR_NOSYSTEM` in the merge-tree call, and a `merge.directoryRenames=false` case expecting the conflict notice. State in the comment that `attr.tree=` and the rename pins only restate git's defaults for versions that read them. Relabel `:1771` as a scanner-override check.

#### C2-L4 (LOW): the remedy is appended to reasons it does not address, and `attributes_unguaranteed`'s doc no longer covers all its callers

- Where: `scripts/secret-scan-branch.sh:375` (remedy), `:367-373` (doc), `:456`, `:460`, `:511`.
- What: every `attributes_unguaranteed` reason now ends with "merge or rebase <base> into this branch ... then re-run". That fits the merge-driver, old-git, and merge-tree reasons. It does not fit three other callers:
  - An unwritable temp directory (`:456`), which the scanner also needs.
  - `git diff` failing (`:511`). That caller does not even know whether the base changed `.gitattributes`, while the doc says "the base changed .gitattributes".
  - "HEAD does not resolve to a commit" (`:460`), which cannot happen once `git merge-base HEAD <base>` and `git rev-list` have succeeded above.
- Fix: pass the remedy only from the callers it fits, or phrase it as a fallback ("if the cause cannot be fixed, merge or rebase ..."), and widen the doc to "the base changed .gitattributes, or whether it did cannot be told".

#### C2-L5 (LOW): two doc pointers will go stale or overclaim

- Where: `docs/tech-debt/README.md:140` (evidence cell); `docs/quality/quality-gates.md:46` and its template copy.
- What: the row cites `docs/reports/cross-review-triage-secret-scan-attr-parity.md` for "#181's cycle-1 cross-review reproduced a local renormalization". `/cross-review` rewrites that per-slug file in place, and the cycle-2 run is next, so the pointer will no longer show that reproduction. In quality-gates, "the same added lines as CI, apart from `.git/info/attributes`" is a closed list. It omits the git-older-than-2.41 case, which the scanner header and the tech-debt row both state.
- Fix: cite the cycle and the reviewed HEAD in the row (for example "cycle-1 triage, reviewed HEAD af29652"), and add "on git 2.41 or later" to the quality-gates sentence.

### Answers to the cycle-2 focus areas

1. **merge-tree isolation.** The pins cover every local setting that was shown to change the merged `.gitattributes`: user and system attributes files, renormalization, directory renames, the working-tree and inherited attribute source. `GIT_ATTR_SOURCE` set to HEAD's commit is the right choice over `unset`: unset lets an uncommitted working-tree `merge=union` change the merge (two assertions go red), while CI has only committed content. Which committed tree GitHub's own merge reads is unknown (see Known gaps). Test coverage of the pins is uneven (C2-M1, C2-L3).
2. **renameLimit 7000 against the scanner's 1000.** Justified. 7000 is merge's default when neither `merge.renameLimit` nor `diff.renameLimit` is set, and 1000 is diff's. The comment gives the fallback chain, but it does not say in so many words that the two numbers differ on purpose. Not probed, since it needs a candidate set above the limit.
3. **Error paths.** A conflict (rc 1) reads HEAD's attributes with a notice, and every other merge-tree rc is classified. The temp-file guard works but covers the second `mktemp`, not the first (C2-L2). The HEAD guard is unreachable. The remedy wording is attached too broadly (C2-L4).
4. **Tests.** The real-git version gate is honest: it SKIPs the merge-derived cases below 2.41, runs the stubbed old-git case everywhere, and runs a real-old-git assertion only there. The four newly tested branches each assert their exact reason: the merge-driver listing failure, the attribute-diff failure, the grep failure on `.git/info/attributes`, and a version string that is malformed or empty. No assertion was weakened: the renamed info/attributes test asserts the new "cannot scan" wording, and `expect_attr_unguaranteed` gained assertions. No secret-shaped literal appears in the delta. Gaps: C2-M1, C2-L3.
5. **Slice D.** All six cycle-1 findings were addressed, and five are fully fixed (table above). Each new test goes red on its own revert: the NUL-byte cases, the `GIT_ATTR_NOSYSTEM` wrapper, the real merge under the guard hook, and the 18 info/attributes assertions. L3 is half-fixed (C2-L2).

### Positive notes

- The stubs now match on the whole argument list (`case " $* "`), because the script puts `-c` options before the subcommand. A stub keyed on `$1 $2` would have gone silent.
- The merge-guard test runs a real `git merge --no-ff` under the installed hook and checks `HEAD^2`, not only an exit code.
- `GIT_ATTR_SOURCE` and `GIT_ATTR_NOSYSTEM` are set inside the command substitution's subshell, so they do not leak into the scanner call that follows.
- The comment's claim that merge-tree reads no `diff.algorithm` survived a targeted probe.

### Known gaps

- Which attributes GitHub's server-side merge applies is not documented. Vanilla git in a bare repository reads no in-tree attributes by default (probe above), while this script uses HEAD's committed attributes. The two differ only when a committed `.gitattributes` gives `.gitattributes` itself a `merge=` attribute (`union` or `binary`). Worth one clause in the "local merge is not GitHub's" row at `docs/tech-debt/README.md:140`.
- `merge.renameLimit` was not probed, and git versions other than 2.49 were not run for this cycle.

### Recommendation

No CRITICAL or HIGH findings, so this is not a blocker. Fix C2-M1 before merge: one fixture closes the only reachable false clean that the suite does not guard. This is the last cycle, so each unfixed LOW is a deferral. Fix C2-L1 to C2-L5 in-cycle, or record them together in one batched `docs/tech-debt/README.md` row with a trigger ("next touch to `use_merge_attributes` or `check_info_attributes`").
