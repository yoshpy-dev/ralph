# Test report: secret-scan-attr-parity (cycle 1)

- Date: 2026-09-28
- Plan: `docs/plans/active/2026-09-27-secret-scan-attr-parity.md`
- Worktree: `.claude/worktrees/secret-scan-attr-parity`, branch `fix/secret-scan-attr-parity`
- HEAD at test time: `49e4684ffa8a504b3847aabbb4c91f6f01c1c8fb` (confirmed clean `git status --porcelain` before and after this cycle)
- Scope: behavioral tests only (static analysis already passed in `/verify` at `733b727`; no diff-quality review this cycle)

## Verdict: PASS

`./scripts/run-test.sh`, `tests/test-secret-scan.sh` (123/123), `tests/test-secret-scan-branch.sh` (190/190), and
`tests/test-run-verify-branch-secret-scan.sh` (32/32) all pass. Stable across three shells (`/bin/sh`, `bash --posix`,
`dash`), 10 repeat runs of each secret-scan file (0 flakes), `TMPDIR=/tmp`, and a hostile outer git config (poisoned
driver/merge config, poisoned `GIT_ATTR_SOURCE`/`RALPH_SECRET_SCAN_ATTR_SOURCE`) that proved the suites' own
hermeticity. All 10 requested red/green mutations discriminated exactly as predicted, closing the loop on every new
AC (AC-1, AC-2, AC-3, AC-4, AC-5, AC-5b, AC-5c, AC-6, AC-7). A 15-scenario live demonstration against `main`'s
pre-fix scanner reproduced every row of the plan's own reproduction table. CI parity holds byte-for-byte over this
repository's whole history (394,454 raw `git log -p` lines, identical) and on rename/submodule fixtures. Five real
but narrow coverage gaps were found in a REAL (not stubbed) old-git environment and in two defensive `git config`/
`git diff` failure branches — all recorded below, none blocking this verdict per the plan's Non-goals and the
existing tech-debt entry for the same class of gap.

## 1. Execution

| Command | Result |
|---|---|
| `./scripts/run-test.sh` (full wrapper) | exit 0; `tests/test-secret-scan.sh` 123/123, `tests/test-secret-scan-branch.sh` 190/190, all other shell suites green, `Language scope: changed (no_changes)` |
| `sh tests/test-secret-scan.sh` | 123/123 |
| `sh tests/test-secret-scan-branch.sh` | 190/190 |
| `sh tests/test-run-verify-branch-secret-scan.sh` | 32/32 |
| Same three, `TMPDIR=/tmp` | 123/190/32, identical |

`tests/test-ralph-dispatch.sh` (case I flake watch): 0 `FAIL` in the full `run-test.sh` run; no standalone re-run needed.

## 2. Shell / hostile-config matrix

| Shell | `test-secret-scan.sh` | `test-secret-scan-branch.sh` |
|---|---|---|
| `/bin/sh` | 123/123 | 190/190 |
| `bash --posix` | 123/123 | 190/190 |
| `dash` | 123/123 | 190/190 |

**Hostile outer git config, `env -i`, distinct `HOME`** (`diff.hostiledriver.binary=true`, `merge.default=hostilemerge`,
`merge.hostilemerge.driver=false` in the poisoned global config):

- With `GIT_ATTR_SOURCE`/`RALPH_SECRET_SCAN_ATTR_SOURCE` also poisoned to garbage strings: `test-secret-scan-branch.sh`
  stayed 190/190 (its `HOME`/config isolation and `unset GIT_ATTR_SOURCE RALPH_SECRET_SCAN_ATTR_SOURCE` run before any
  fixture). `test-secret-scan.sh` failed at exit 128 (`fatal: bad --attr-source or GIT_ATTR_SOURCE`) — **by design, not
  a hermeticity gap**: its own header comment states cases before its hermetic section (lines ~40-90, `--file`/`--stdin`/
  `--staged` fixtures) intentionally run "under the caller's own config", and a native git env var poisoned process-wide
  breaks *any* git invocation before that file's own `unset` line runs (matches `[[secret_scan_fixture_gotchas]]`
  gotcha 7 exactly).
- Re-run with only the poisoned `HOME`/driver/merge config and **no** poisoned `GIT_ATTR_SOURCE`/`RALPH_SECRET_SCAN_ATTR_SOURCE`:
  both files fully pass (123/190), proving the driver-binary and merge-driver isolation holds regardless of ambient
  outer config.
- Worst-case direct probe: `scripts/secret-scan.sh --range HEAD` with `RALPH_SECRET_SCAN_ATTR_SOURCE=totally-bogus-garbage-ref-xyz`
  against a real fixture repo, hostile outer `HOME`: exit 3, reason `RALPH_SECRET_SCAN_ATTR_SOURCE (totally-bogus-garbage-ref-xyz)
  does not name a tree`, no `fatal:` leak.

## 3. Docker git-version matrix

`alpine:3.18` (git 2.40.4, older than the 2.41 `GIT_ATTR_SOURCE` requirement) and `alpine:3.19` (git 2.43.7, at CI's
minimum) via a minimal `scripts/`+`tests/` staging under `$HOME` (not `/private/tmp` — Docker Desktop does not share
that path; see `[[secret_scan_fixture_gotchas]]` gotcha 6).

| Image | git | `test-secret-scan.sh` | `test-secret-scan-branch.sh` |
|---|---|---|---|
| alpine:3.18 | 2.40.4 | 94/94 (28 `SKIP`, real-git-version-gated: `attributes-from-HEAD cases (git 2.40.4 is older than 2.41)`) | **166/185, 19 FAIL** (see Gap 1) |
| alpine:3.19 | 2.43.7 | 122/122 (1 `SKIP`, root-permission) | 185/185 (2 `SKIP`, root-permission) |

Root-permission `SKIP`s (`read-only object database`, `an unreadable .git/info/attributes`) are a container artifact
(root can always read/write), not a logic gap — 0 `FAIL` from them in either image.

## 4. Repeat (10x)

`tests/test-secret-scan.sh`: 10/10 runs, 123/123 every time. `tests/test-secret-scan-branch.sh`: 10/10 runs, 190/190
every time. Zero flakes.

## 5. Mutation testing

All mutations applied to `scripts/secret-scan.sh` / `scripts/secret-scan-branch.sh` only (never the `templates/base/`
copies, since only `scripts/` is exercised by the tests), each edited, tested, and reverted individually via
`git checkout -- <file>`; `git status --porcelain` confirmed empty and HEAD unchanged (`49e4684`) after every single
mutation and at the end of the full sequence.

| # | Mutation | File | Result |
|---|---|---|---|
| 1 | Drop the binary-pin loop body (no `-c`/`--config-env` added) | secret-scan.sh | **6 FAIL** — exactly the AC-1/AC-2 driver-pin tests (single driver, two drivers, dotted name, `=`-named driver, NUL-byte default-config parity) |
| 2 | `auto` → `false` for the binary pin | secret-scan.sh | **2 FAIL** — both NUL-byte-file tests ("skips the NUL-byte file as the default config does"), matching self-review's own M1 rationale exactly |
| 3 | Drop `GIT_ATTR_NOSYSTEM=1` | secret-scan.sh | **1 FAIL** — the dedicated `GIT_ATTR_NOSYSTEM=1` env-passthrough probe |
| 4 | Ignore `RALPH_SECRET_SCAN_ATTR_SOURCE` | secret-scan.sh | **13 FAIL** — 6 in `test-secret-scan.sh` (AC-6: tree/commit/blob/unresolvable resolution) + 7 in `test-secret-scan-branch.sh` (AC-4: the branch script's own consumption of the variable for merge attributes) |
| 5 | Drop the `.gitattributes`-changed check (always call `use_merge_attributes`) | secret-scan-branch.sh | **4 FAIL** — both "no merge-tree notice when unchanged" regression guards, plus "already merged" guard, plus the "prints exactly one status line" invariant |
| 6 | Treat every non-zero `merge-tree` rc as conflict (fold `*` into `1`) | secret-scan-branch.sh | **8 FAIL** — exactly the AC-5c fail-closed scenarios (exit 128, no `--write-tree`, read-only object database) |
| 7 | Drop the merge-driver refusal (`merge.default`/`merge.<name>.driver` check) | secret-scan-branch.sh | **14 FAIL** — exactly the AC-5b `merge.default` and `merge.keep.driver` scenarios (strict reason, remedy, default-mode fallback, both driver-naming shapes) |
| 8 | Drop the git-version gate (`git_version_at_least 2 41`) | secret-scan-branch.sh | **6 FAIL** — exactly the stubbed "git older than 2.41" scenario |
| 9 | Drop the `check_info_attributes` call | secret-scan-branch.sh | **15 FAIL** — exactly AC-3/AC-7 (rule line, unrelated rule, symlink-to-rules, directory, unreadable; two of these also flip from the expected default-mode exit 0 to exit 1 because the scan then proceeds and finds real content, consistent with the design) |
| 10 | Drop the remedy text from `attributes_unguaranteed` | secret-scan-branch.sh | **12 FAIL** — every scenario that names the remedy line, across AC-5b/AC-5c/AC-4/AC-7-adjacent stub-git assertions |

Every mutation's failure set matches its corresponding AC/design-decision exactly — no mutation was silently
absorbed, and no mutation produced failures outside its own intended scope. Final confirmation run after all 10
reverts: `test-secret-scan.sh` 123/123, `test-secret-scan-branch.sh` 190/190, `test-run-verify-branch-secret-scan.sh`
32/32, and a full `./scripts/run-test.sh` pass.

## 6. Live demonstration

Isolated `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` fixtures (token built at runtime from split pieces), comparing
`main`'s `secret-scan-branch.sh`/`secret-scan.sh` (via a scratch `git worktree add --detach main`, removed after) with
this worktree's scanners.

| Scenario | `main` (old) | new | Match to plan |
|---|---|---|---|
| (a) committed `diff=mydriver` + local `diff.mydriver.binary=true`, text content | exit 0, clean (false clean) | exit 1, finds the token | matches "main 0 (false clean), new 1" |
| (a) same driver, real NUL-byte content | exit 0, clean | exit 0, clean | matches "both 0" |
| (b) `.git/info/attributes` holds `*.txt -diff` | old: exit 0, clean (unaware of the file) | new: `--strict` exit 3, `cannot scan, .git/info/attributes holds attribute rules...` | matches "main strict 0 clean, new strict 3 with the reason" |
| (b) unrelated linguist rule in `.git/info/attributes` | — | new: `--strict` exit 3 (by design, presence-only check) | matches |
| (b) real `git merge` (non-fast-forward) with the merge guard (`prepare-commit-msg`) installed, `.git/info/attributes` rule present | — | merge completes, 2-parent merge commit created, no block | confirms AC-3's revised scope (scanner itself never refuses on `info/attributes`) |
| (c) base **adds** `-diff` after fork | old: exit 1 (misses base's fix, reads HEAD's own attrs) | new: exit 0, notice `attributes read from the merge of main and HEAD` | matches "main strict 1, new strict 0 with merge notice" |
| (c) base **removes** `-diff` after fork | old: exit 0, clean (misses the leak) | new: exit 1, finds it | matches "main 0, new 1" |
| (c) base **unchanged** | new: exit 1, finds it, **no** "attributes read from the merge" notice in stderr | matches "no merge-tree notice" |
| (d) base and HEAD conflict on `.gitattributes` | new: exit 0, notice `but the merge conflicts; attributes read from HEAD (CI does not run on a conflicting pull request)` | matches exactly |
| (e) local `merge.default` set, base changed `.gitattributes` | new `--strict`: exit 3, reason names the config + the merge/rebase remedy | new default: exit 1 (finds the token), notice + `attributes read from HEAD` + remedy, does not fail | matches AC-5b exactly, both modes |

All 10 scenario rows (15 individual scanner invocations) reproduced their predicted result exactly, including the
two negative assertions (no notice text when unchanged; merge succeeds despite the `info/attributes` rule).

## 7. CI parity

| Comparison | Result |
|---|---|
| Whole-repository history, `root..HEAD` (`868da02..49e4684`), full pin set including `GIT_ATTR_SOURCE=HEAD`, `GIT_ATTR_NOSYSTEM=1` (binary-pin loop is a no-op: no local `diff.*.binary` config exists in this environment) | 394,454 raw `git log -p` lines, **byte-for-byte identical** between the old and new pin sets |
| Full scanner (old vs. new), same range | Both exit 0, both stderr empty (no findings) — identical |
| A branch with a pure rename | Both exit 0; stdout+stderr byte-identical |
| A branch with a submodule pointer bump | Both exit 0; stdout+stderr byte-identical |

No behavior difference found under default configuration anywhere tested — AC-8 holds.

## 8. Edge cases

All of the requested edge cases are already covered by the existing (green) test suite; no gaps found here:

| Edge case | Covered by |
|---|---|
| Config key containing `=` for the binary pin | `test-secret-scan.sh`: "range scan pins a driver whose name holds '='" / "...on a git without `--config-env` exits 3" |
| `.git/info/attributes` as a symlink / as a directory | `test-secret-scan-branch.sh`: "symlinked to a rules file", "symlinked to a missing file", "a directory at .git/info/attributes" |
| Nested `sub/.gitattributes` changed on base only | `test-secret-scan-branch.sh`: `test_attr_nested_change_reads_merge` (root + subdirectory-cwd-with-`diff.relative=true` variant) |
| Base already merged into HEAD (merge-tree returns HEAD's own tree) | `test-secret-scan-branch.sh`: "base .gitattributes change already merged: strict scan reads HEAD's attributes without merge-tree" (asserts **no** "attributes read from" notice) |
| `RALPH_SECRET_SCAN_ATTR_SOURCE` naming a commit / a tree / a blob | `test-secret-scan.sh`: empty value, tree id, commit id (resolves via its tree), unresolvable name, blob id (rejected, exit 3) |
| A range not ending at HEAD with the variable set | `test-secret-scan.sh`: two "range not ending at HEAD ignores..." assertions plus "...still reads the working tree's attributes" |

## 9. Coverage

Branches of the new/changed code exercised by at least one test, and which are not:

**`scan_range` (secret-scan.sh):**
- Exercised: `keys_rc` 0 (drivers found) and the explicit `keys_rc` failure path (stubbed `git config` exit 5); the
  `*=*` (`--config-env`) and plain `-c key=auto` pin shapes; `attr_source` = HEAD-commit case with
  `RALPH_SECRET_SCAN_ATTR_SOURCE` unset/empty/tree/commit/blob/unresolvable; range-not-ending-at-HEAD (variable
  ignored).
- Not exercised, and **not a real gap by construction**: the `"") ;;` empty-`binary_key` arm of the pin loop. When
  `git config --name-only --get-regexp` finds zero keys (`keys_rc`=1, an empty file), `read -r` fails immediately on
  EOF and the loop body never runs at all; git's own `--name-only` output for a real multi-key match never emits a
  blank line between keys. Same defense-in-depth category as `[[secret_scan_fixture_gotchas]]` gotcha 5's
  `--no-ext-diff` finding — this arm cannot be reached by any real git config listing, tested or not.

**`check_info_attributes` (secret-scan-branch.sh):** exercised: absent (pass-through), rule line present, only
comments/blanks, a directory, a symlink to a rules file, a symlink to a missing file. **Not exercised (real, narrow
gaps):** the `git rev-parse --git-path info/attributes` failure arm (near-unreachable — the repo/work-tree checks
above it already validated a working repo) and the `grep` exiting with something other than 0/1 (reachable the same
way as `scan_range`'s tested `keys_rc`=5 case — a `grep` wrapper on `PATH` — just not yet built for this file).

**`use_merge_attributes` (secret-scan-branch.sh):** exercised: `drivers_rc`=0 for both `merge.default` and
`merge.<name>.driver`; `drivers_rc`=1 (implicit, the normal path throughout every AC-4 test); the git-version gate's
true branch (host's real 2.49 in every successful merge-tree test) and its stubbed-false branch (fake "git version
2.40.1"); `mktemp` failure; merge-tree rc=0 with a valid tree id; rc=1 (conflict); rc=128 and "no `--write-tree`"
variants of rc-other. **Not exercised (real, narrow gaps):** `drivers_rc`=other (the merge-driver-listing `git
config` call itself failing with an unexpected exit code) — the direct structural analogue of `scan_range`'s already-
tested `keys_rc`=other case, just not ported to this function; and the *outer* `attr_diff_rc`=other case (the
top-level `git diff --quiet ... .gitattributes` check that decides whether to call `use_merge_attributes` at all
failing with something other than 0/1) — both reachable via the same PATH-shadowing wrapper technique already used
twice in this suite.

**`attributes_unguaranteed`:** both branches (strict `cannot_scan` with remedy; default `report` + HEAD-fallback with
remedy) are exercised by every `use_merge_attributes` failure scenario above; mutation 10 independently confirmed the
remedy text is load-bearing in both branches.

**`git_version_at_least`:** true branch (host git 2.49) and a well-formed false branch (stubbed "2.40.1") are both
exercised. **Not exercised (real, low-severity gap):** the malformed/empty `git version` output arm (`gv_major`/
`gv_minor` failing to parse as digits), which produces the `${git_version_text:-an unreadable git version}` wording
in the reason — same fail-closed *outcome* as the tested "too old" case, just a different trigger, never
independently tested.

## 10. Failure analysis

No unexpected failures. The only failures observed in this cycle were: (a) all 10 intentional mutations (Section 5,
each discriminating exactly as predicted and fully reverted), and (b) the 19 `test-secret-scan-branch.sh` failures
under a genuinely old git in Docker (Section 3 / Gap 1 below), which trace to a test-suite design gap, not a
production defect — production's own fail-closed behavior under that same real old git was independently confirmed
correct (Gap 1).

## 11. Regression

None found. Full shell suite green via `./scripts/run-test.sh` (`Language scope: changed (no_changes)` — no Go
files touched). Whole-history CI parity is byte-for-byte identical to `main` under default configuration (Section
7). `templates/base/scripts/secret-scan.sh` and `templates/base/scripts/secret-scan-branch.sh` remain byte-identical
to their `scripts/` counterparts (`diff -q`, both empty).

## 12. Gaps

1. **`test-secret-scan-branch.sh` does not gracefully skip its merge-tree-path assertions under a REAL git older
   than 2.41** (unlike its sibling `test-secret-scan.sh`, which has an explicit, working `SKIP: attributes-from-HEAD
   cases (git X.Y is older than 2.41)` real-git-version gate). Under `alpine:3.18` (real git 2.40.4), 19 assertions
   fail with a *mismatched but production-correct* message: the genuine `git_version_at_least` gate fires first
   ("...changed .gitattributes since X, and git version 2.40.4 is not 2.41 or later, which GIT_ATTR_SOURCE needs"),
   before the assertion's own intended failure mode (mktemp failure, non-`--write-tree` git, a non-conflict
   merge-tree exit code, or a real base-changed-`.gitattributes` scenario) is ever reached — because those
   assertions assume the *host* git is 2.41+ and only stub the "old git" scenario narrowly via a fake `git version`
   wrapper for one dedicated test, not for the suite as a whole. This is a test-portability gap (would need the same
   kind of real-git-version `SKIP` gate `test-secret-scan.sh` already has), not a script defect — production's own
   behavior under this real old git is exactly as designed (fail-closed with an accurate reason). Not fixed this
   cycle per "do not edit code (record gaps instead)".
2. `use_merge_attributes`'s `drivers_rc`=other branch (Section 9) — untested, same class as the already-tested
   `scan_range` `keys_rc`=other case.
3. The top-level `attr_diff_rc`=other branch that gates whether `use_merge_attributes` is called at all (Section 9)
   — untested.
4. `check_info_attributes`'s `grep`-exits-other-than-0/1 branch (Section 9) — untested.
5. `git_version_at_least`'s malformed/empty-version-string branch (Section 9) — untested, low severity (same
   fail-closed outcome as the tested "too old" case).
6. Already recorded in the plan's own work log (not new this cycle): `git merge-tree`'s own use of
   `merge.renames`/similar config is not separately pinned; a probe found no observable difference in this
   environment, and it is filed as unreproduced tech-debt.

None of the above block the PASS verdict — all are narrow, fail-closed-safe, or test-suite-only gaps, consistent
with this plan's Non-goals (CI's own shared gaps) and existing tech-debt entries for the same class of issue.

## Insight event

```
./scripts/insights-append.sh --slug secret-scan-attr-parity --flow standard --phase test --cycle 1 --verdict pass --critical 0 --high 0 --medium 0 --low 0 --source skill
```
