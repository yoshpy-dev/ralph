# Test report: secret-scan-git-config (cycle 1)

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-25-secret-scan-git-config.md` (issue #176)
- Tester: `tester` subagent (Claude Code)
- HEAD: `b76d9dab98eb8fc8a4f0fdf16ec0b5446c32f1be` (worktree `git status --porcelain` empty at start and end; confirmed again after every mutation revert)
- Scope: behavioral tests only. Static analysis already passed in `/verify` (`docs/reports/verify-2026-09-25-secret-scan-git-config.md`, PASS, evidence `docs/evidence/verify-2026-09-25-073053.log`). No diff-quality review here (self-review already passed, `docs/reports/self-review-2026-09-25-secret-scan-git-config.md`).

## Verdict: PASS

`tests/test-secret-scan.sh` (61/61), `tests/test-secret-scan-branch.sh` (108/108), and `tests/test-run-verify-branch-secret-scan.sh` (32/32) all pass, stable across 3 shells, 10 repeat runs each, a hostile outer git config, `TMPDIR=/tmp`, and a spaced `TMPDIR`. 21 red/green mutations against the new pins/checks all discriminated exactly as predicted (or, for the two that showed no observable difference, the reason is now proven, not just asserted — see "Mutation results" and "Coverage"). A 9-scenario live demonstration reproduced every row of the plan's reproduction table: the pre-fix scanner (`git show main:scripts/secret-scan.sh`) misses all of them (false-negative "clean"), the post-fix scanner (this worktree) catches all of them. CI parity holds exactly on 4 independent ranges under default config. No regressions, no fixes needed.

## 1. Execution

| Command | Result |
|---|---|
| `./scripts/run-test.sh` (full changed-scope run) | PASS — "All verifiers passed." Evidence: `docs/evidence/verify-2026-09-25-073650.log`. `tests/test-secret-scan.sh` 61/61, `tests/test-secret-scan-branch.sh` 108/108, `tests/test-run-verify-branch-secret-scan.sh` 32/32; full shell suite and `go test ./...` (8 packages) also green. |
| `sh tests/test-secret-scan.sh` (standalone) | 61/61 |
| `sh tests/test-secret-scan-branch.sh` (standalone) | 108/108 |
| `sh tests/test-run-verify-branch-secret-scan.sh` (standalone) | 32/32 |
| Same 3, with `TMPDIR=/tmp` (CI-like) | 61/61, 108/108, 32/32 — identical |
| `tests/test-ralph-dispatch.sh` case I ("SIGTERM cleanup left stray ralph-dispatch-\* temp files") | Passed cleanly in the full run (26/26); not a flake this cycle, no rerun needed. |

## 2. Environment matrix

| Variant | `test-secret-scan.sh` | `test-secret-scan-branch.sh` |
|---|---|---|
| `/bin/sh` | 61/61 | 108/108 |
| `/bin/bash --posix` | 61/61 | 108/108 |
| `/bin/dash` | 61/61 | 108/108 |
| Hostile OUTER `HOME`/`GIT_CONFIG_GLOBAL` (a HOME distinct from the suite's own, `.gitconfig` with `color.ui=always`, `diff.relative=true`, `diff.renames=copies`, `diff.algorithm=histogram`, `log.showSignature=true`, `core.bigFileThreshold=1`) | 61/61 | 108/108 |
| `TMPDIR` containing a space | 61/61 | 108/108 |

Both files' own hermetic-section setup (`HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM=/dev/null`/`GIT_CONFIG_NOSYSTEM=1`, unset `XDG_CONFIG_HOME`) reassigns and exports over whatever the outer process passed in, so the outer hostility never reaches any fixture repo — confirmed empirically, not just by reading the code.

## 3. Repeat

10 sequential runs each, `sh`:

- `tests/test-secret-scan.sh`: 61/61 every time (10/10 runs clean).
- `tests/test-secret-scan-branch.sh`: 108/108 every time (10/10 runs clean).
- The TERM test specifically (`a TERM during the scan exits 143` / `... still removes the temp dir`) was grepped out of each of the 10 `test-secret-scan.sh` runs individually: both assertions PASS in all 10, no flake.

## 4. Red/green mutations

All mutations were applied in place to `scripts/secret-scan.sh` or `scripts/secret-scan-branch.sh` with a Python single-occurrence string replace (fails loudly if the anchor text isn't unique, so a mutation can never silently apply to the wrong spot), tested, then reverted with `git checkout -- <file>`. `git status --porcelain` and a byte-for-byte `diff` against a pre-mutation backup were both confirmed clean after every single revert, and again at the end of the whole round.

### `scan_range` pins (drop one at a time)

| Dropped | Failing test(s) | Match? |
|---|---|---|
| `--no-replace-objects` | `range scan ignores a local replace ref for the leaking commit` (1) | Exact |
| `-c diff.relative=false` | `AC-2: range scan with diff.relative=true from a subdirectory finds a token outside it` (1) | Exact |
| `-c diff.renames=true` | `AC-4: range scan with diff.renames=copies finds the token in a copied file`, `AC-4: pure rename with diff.renames=false matches the default result` (2) | Exact |
| `-c diff.renameLimit=1000` | `renamed-and-edited files with diff.renameLimit=1 match the default result` (1) | Exact |
| `-c core.bigFileThreshold=512m` | `AC-16: range scan with core.bigFileThreshold=1 still reads the text file` (1) | Exact |
| `-c core.attributesFile=/dev/null` | `range scan ignores a user-level attributes file marking the file -diff` (1) | Exact |
| `-c log.showRoot=true` | `range scan with log.showRoot=false still reads a root commit's diff` (1) | Exact |
| `-c log.showSignature=false` | **0 failures** | Matches the plan's own prediction — self-review already named this as untested (Deviation notes: "`--no-show-signature`はテストなし") |
| `--no-color` | **0 failures** | Matches the task's own prediction — `scan_diff_stream`'s ANSI strip is a second, independent defence (confirmed separately below) |
| `--no-textconv` | `AC-3: range scan ignores a local textconv for a committed diff driver` (1) | Exact |
| `--no-ext-diff` | **0 failures** | Not a coverage gap — see "Coverage" below for why this is provably a no-op for `git log -p` |
| `--diff-algorithm=default` | `AC-5: moved token line with diff.algorithm=histogram matches the default result`, `AC-5: moved token line with diff.algorithm=patience matches the default result` (2) | Exact |
| `--submodule=short` | `submodule bump with diff.submodule=diff matches the default result` (1) | Exact |

### `scan_range`'s exit-status check and `scan_diff_stream`'s ANSI strip

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Revert the file-then-check-`log_rc` pattern to a plain pipe (`git log ... \| scan_diff_stream ...`, no exit-status check) | All 6 AC-15 assertions: 3 scenarios (missing `diff.orderFile`, unresolved range, mid-range object failure) x 2 assertions each (exit code + stderr message) | Exact |
| Remove the ANSI-escape `gsub` from `scan_diff_stream`'s awk | `AC-10: --diff finds the token in a colored added line` (1) | Exact — confirms the ANSI strip is what AC-10 actually depends on, and confirms (by the `--no-color` result above) that `scan_range` doesn't need it since it never emits color in the first place |

### `scan_staged`

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Revert the whole function to the pre-fix `--name-only` + `git show ":$path"` shape (from `git show main:scripts/secret-scan.sh`) | 15: all 6 AC-7 name-quoting cases, `AC-8` (subdirectory), the symlink-replaced-by-file case, both replace-ref cases, `AC-9` (unreadable blob, 2 assertions), and all 3 malformed-raw-line cases | Exact |
| Drop the `is_object_id` validation call only (keep the blob-id-based rewrite otherwise) | 3: `staged: a raw line with no new object id exits 3`, `staged: an unparsable line names the entry`, `staged: an abbreviated all-zero object id exits 3` | Exact — isolates exactly what `is_object_id` itself is responsible for, out of the 15 the full-function revert breaks |

### `--file` readability check

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Drop the `[ ! -f "$2" ] \|\| [ ! -r "$2" ]` guard | `--file on a missing path exits 3`, `--file on a missing path names it`, `--file on an unreadable file exits 3` (3) | Exact |

### Signal traps

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Revert to the pre-fix single `trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM` (drop the per-signal `exit 129`/`130`/`143`) | `a TERM during the scan exits 143` (1; `... still removes the temp dir` stays PASS, as expected — the EXIT trap still fires, just with the wrong exit code) | Exact for the one signal the suite tests (TERM) |
| (follow-up, not part of the requested list) Change only the `HUP` trap's exit code (`129` → `42`), leaving `INT`/`TERM` untouched | **0 failures** | Confirms `HUP` (and by the same code shape, `INT`) is entirely unexercised — see "Coverage" |

### `secret-scan-branch.sh`

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Drop the `printf x` / `${var%x}` sentinel around the symlink-target read (revert to plain command substitution) | Both `AC-11 target ending with a newline` assertions (exit code + could-not-read notice) | Exact |
| Drop `core.quotePath=false` from `allowlist_ls_tree_mode`'s `ls-tree` call | All 3 `AC-12 non-ASCII target` assertions | Exact |

## 5. Live demonstration

Isolated scratch fixture repo (`HOME`/`GIT_CONFIG_GLOBAL` pointed at a throwaway dir, `GIT_CONFIG_SYSTEM=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, `XDG_CONFIG_HOME` unset). Pre-fix scanner extracted via `git show main:scripts/secret-scan.sh`; post-fix scanner is this worktree's `scripts/secret-scan.sh` / `scripts/secret-scan-branch.sh`. Fixture secret assembled at runtime from split pieces (`printf 'sk_live_%s' 'abcdefghijklmnopqrstuv'`), matching the existing test convention; only file:line/pattern-name labels are reproduced below, never the matched text.

| Scenario | OLD (pre-fix) | NEW (post-fix) |
|---|---|---|
| A: `color.ui=always` (range) | exit=0, clean (miss) | exit=1, `secret-a.txt:1 [Stripe live secret key]` |
| B: `diff.relative=true`, run from a subdirectory (range) | exit=0, clean (miss) | exit=1, `secret-b.txt:1 [Stripe live secret key]` |
| C: committed `.gitattributes` diff driver + local `textconv` (range) | exit=0, clean (miss) | exit=1, `secret-c.enc:2 [Stripe live secret key]` |
| D: `diff.renames=copies`, duplicate an existing (same-commit-touched) base file (range) | exit=0, clean (miss) | exit=1, finding on the copy |
| E: `core.bigFileThreshold=1` (range) | exit=0, clean (miss) | exit=1, `secret-e.txt:2 [Stripe live secret key]` |
| F: nonexistent `diff.orderFile` — `git log` itself fails (range) | exit=0, clean (**false** "clean" — `git log` actually failed with `fatal: failed to read orderfile ...`, but the old pipe-based `scan_range` never checked) | exit=3, `secret-scan: could not scan main..feat-orderfile: git log exited with 128, so the range was not (fully) scanned` |
| G: staged file with a non-ASCII name, default config (staged) | exit=0, clean (miss — `--name-only`'s quoted-octal name doesn't match `git show ":$path"`) | exit=1, `café.txt:1 [Stripe live secret key]` |
| H1: `secret-scan-branch.sh --strict`, `color.ui=always`, end-to-end | (not run — this scenario is the current worktree's own AR-3 fix, not a standalone pre-fix reproduction) | exit=1, `secret-scan-branch: scanned <base>..<head> against main: findings` |
| H2: `secret-scan-branch.sh --strict`, nonexistent `diff.orderFile`, end-to-end | (not run) | exit=3, three-line stderr: `fatal: failed to read orderfile ...` → `secret-scan: could not scan <range>: git log exited with 128, ...` → `secret-scan-branch: scanner failed with exit 3 on <range>` |

Scenario D required matching the actual AC-4 test fixture's construction exactly (modify `orig.txt` in the *same* commit that adds the copy, not just add the copy alone): plain `-C` copy detection (what `diff.renames=copies` enables, as opposed to `--find-copies-harder`) only considers files touched in the same changeset as candidate copy sources. An initial attempt without touching `orig.txt` produced no copy detection at all (both OLD and NEW showed exit=1, an ordinary add) — a self-caught fixture error, not a scanner behavior, fixed before this table was produced.

## 6. CI parity (AC-6)

Under isolated **default** git config (empty global, `GIT_CONFIG_SYSTEM=/dev/null`), OLD vs NEW on 4 ranges:

| Range | OLD | NEW | Result |
|---|---|---|---|
| (a) plain leak, fresh fixture | exit=1, 1 finding | exit=1, 1 finding | MATCH |
| (b) a branch with a pure rename (content unchanged) | exit=0, 0 findings | exit=0, 0 findings | MATCH |
| (c) a branch with a submodule pointer bump | exit=0, 0 findings | exit=0, 0 findings | MATCH |
| (d) this repository's own range, `$(git merge-base HEAD origin/main)..HEAD` (`c1785b8..b76d9da`) | exit=0, 0 findings | exit=0, 0 findings; `diff` of the two scanners' full stdout+stderr is byte-identical | MATCH |

No behavior difference under default config anywhere tested — AC-6 holds.

## 7. Edge cases

| Case | Source | Result |
|---|---|---|
| Empty staging area | Existing test (`staged: an empty staging area exits 0`) | Covered, passing |
| A staged symlink (mode `120000`, not "replaced by a file" — a genuinely new symlink whose target is an ordinary filename) | New, manual | `--staged` reads the symlink's blob (the target-path string) as content without erroring; target text didn't match a pattern, exit=0. No crash, no special-case needed — the existing gitlink/all-zero-id skips are the only special cases, and neither applies to `120000`. |
| A staged mode-only change (`chmod +x` on an already-committed file whose *content* (unchanged) holds a secret) | New, manual | Raw diff line: `:100644 100755 <same-id> <same-id> M	modefile.txt` (status `M`, included by `--diff-filter=d`). Scanner re-scans and detects it (exit=1) — mode-only changes are not skipped. |
| An unmerged path during a merge conflict (no staged blob) | Existing test (`staged: an unmerged path (no staged blob) does not fail the scan`) | Covered, passing (exit 0, not a failure) |
| 300 staged files (one holding the token) | New, manual | OLD: 26.25s total (4.86s user / 9.21s sys), 1 finding. NEW: 26.04s total (4.96s user / 9.39s sys), 1 finding. No runtime regression — both dominated by ~300 per-file git subprocess spawns in this environment; NEW is marginally faster, not slower. |
| A file name with a newline | Existing test (`AC-7: staged file whose name has a newline is scanned` in the AC-7 family) | Covered, passing |

## 8. Coverage

Branches exercised by at least one test, vs. not:

**`scan_range` (all pins exercised except two, both explained, neither a live gap):**
- Exercised: `--no-replace-objects`, `-c diff.relative=false`, `-c diff.renames=true`, `-c diff.renameLimit=1000`, `-c core.bigFileThreshold=512m`, `-c core.attributesFile=/dev/null`, `-c log.showRoot=true`, `--no-textconv`, `--diff-algorithm=default`, `--submodule=short`, the `log_rc` exit-status check (AC-15, 3 sub-scenarios).
- **Not exercised, but confirmed provably a no-op for this invocation, not a gap: `--no-ext-diff`.** `git log -p` (what `scan_range` calls, with or without `--no-ext-diff`) never invokes an external diff driver unless the caller *also* passes `--ext-diff` — confirmed empirically this cycle: a fixture with a committed `.gitattributes` diff driver and a local `diff.<driver>.command` produces the *real* content under plain `git log -p` and under `git log -p --no-ext-diff` alike, and only the *real* content is redacted when `--ext-diff` is explicitly added (`git help diff`: "If you set an external diff driver with gitattributes(5), you need to use this option with git-log(1) and friends."). So `--no-ext-diff` cannot currently be bypassed by any local config through `scan_range`; it is defense-in-depth against a future change to the invocation (e.g. someone adding `--ext-diff` later), the same category as `--no-color` but with a git-documented mechanical reason rather than only "the ANSI strip already covers it."
- **Not exercised, and genuinely untested by design choice (self-review's own note): `-c log.showSignature=false`.**

**`scan_staged` (fully exercised):** the raw-id parse (`is_object_id`), the gitlink skip (`160000`), the all-zero-id skip (unmerged paths), the unreadable-blob exit 3, and non-ASCII/space/tab/quote/backslash/newline names are each independently pinned by a distinct test.

**`--file` check (fully exercised):** missing path, unreadable path, existing-empty-file, both message and exit code.

**Signal traps (partially exercised):** `TERM` is directly tested (kill -TERM mid-scan, FIFO-based). **`HUP` and `INT` are not exercised by any test** — confirmed this cycle by mutating only the `HUP` trap's exit-code literal (129 → 42) with `INT`/`TERM` left untouched: 0 test failures. This is low-risk (the code being defended — POSIX sh resuming after a trap instead of stopping — is the same three-line shape for all three signals, and the mechanism itself is proven by the TERM case) but is a real, previously-unrecorded blind spot. Not closed this cycle: replicating the existing TERM test (FIFO + up-to-20×1s poll loop) for both HUP and INT would roughly triple that section's fixture cost for a narrow, low-severity gap — recorded here per the "otherwise record gaps" instruction rather than added.

**`secret-scan-branch.sh`'s symlink-target read (mostly exercised, two pre-existing inherited gaps, unchanged by this PR):** the newline-sentinel (AC-11), `core.quotePath=false` (AC-12), a bare-directory target, an absolute target, a `..`-containing target, a leading-`./` target, and a target ending in a non-existent path are all tested. Two gaps already known and recorded from before this PR (not touched or affected by it — the mode-dispatch `case` block's shape is unchanged): a `120000` (symlink) target that itself resolves to a `160000` (submodule) or another `120000` (symlink-to-symlink) — both confirmed fail-closed by hand in the prior local-branch-secret-scan cycle, not covered by a dedicated test (see `docs/reports/test-2026-09-24-local-branch-secret-scan.md`).

## Regression checks

- `AC-6`: default-config range results unchanged — confirmed by CI parity (a) and (d) above, plus 61/61 on `test-secret-scan.sh`'s pre-hermetic-section cases (lines 48-85), which run under the *caller's own* config, not the isolated hermetic section.
- No test file was edited this cycle (all mutation testing was apply → test → `git checkout --` revert; confirmed clean every time). No production script was edited. `git status --porcelain` at the end of this cycle is empty except for this report and the insight-events file (staged separately below).
- `templates/base/scripts/secret-scan.sh` and `templates/base/scripts/secret-scan-branch.sh` were not touched this cycle (already verified byte-identical to `scripts/` in `/verify`); not re-verified here since no script was edited.

## Test gaps

1. **`HUP`/`INT` signal-trap exit codes are unexercised** (see "Coverage" above). Low severity — same three-line shape as the tested `TERM` case, same underlying POSIX-sh-resume defense. Recorded, not closed (fixture cost vs. value judged not worth it this cycle).
2. **`-c log.showSignature=false` is unexercised** — already known and explicitly accepted in the plan's own Deviation notes (self-review, Slice D). Not a new finding.
3. **`--no-ext-diff` is unexercised, but not a gap** — provably a no-op for `scan_range`'s exact invocation shape (git requires an explicit `--ext-diff` flag for `git log`, which `scan_range` never passes). See "Coverage" for the empirical proof. No test needed; recorded here only so this doesn't get re-investigated as if it were an open question.
4. **Two pre-existing, inherited allowlist-symlink-target gaps** (submodule target, symlink-to-symlink target) in `secret-scan-branch.sh` — unchanged by this PR, already recorded in the prior cycle's report (`docs/reports/test-2026-09-24-local-branch-secret-scan.md`).

None of the above block the PASS verdict: all are either explicitly accepted by the plan/self-review already, proven not to be reachable, or low-severity and mechanically near-duplicate of already-tested code.

## Insight event

```
./scripts/insights-append.sh --slug secret-scan-git-config --flow standard --phase test --cycle 1 --verdict pass --critical 0 --high 0 --medium 0 --low 0 --source skill
```
