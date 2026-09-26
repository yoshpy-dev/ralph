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

## Cycle 2

- Date: 2026-09-25
- HEAD: `092c5e37890bf853e527e35a9b4211a61cd41944` (worktree `git status --porcelain` empty at start and end; confirmed again after every mutation revert)
- Delta since cycle 1 (`b76d9da`): `f6d1b87` sync-docs, `f413a58` cycle-1 cross-review triage (AR-1), `ca8a212` **fix AR-1** (state-aware `+++ ` header rule: `scan_diff_stream` now skips `+++ ` only between a `diff ` line and the next `@@`, so every `+` line inside a hunk is added content — including one whose content starts with `++ ` or ESC`[32m++ `), `45ce71f` plan notes, `750d247` cycle-2 self-review, `443ffb7` **Slice F** (`GIT_ATTR_SOURCE=HEAD` on the `scan_range` git invocation, so an uncommitted working-tree `.gitattributes` or a local `attr.tree` cannot change what CI's checkout of HEAD would scan; git 2.41+, older git ignores it and reads the working tree as before; an unborn HEAD now exits 3), `5774b5f`/`092c5e3` plan/report notes. `tests/test-secret-scan.sh` grew 61 → 74.

### Verdict: PASS

`tests/test-secret-scan.sh` 74/74, unchanged 108/108 and 32/32 for the other two files, stable across 3 shells, 2 real awk implementations (mawk, busybox awk) beyond macOS's own, 10 repeat runs, and the cycle-1 hostile outer config extended with an `attr.tree` setting. 3 of 4 requested mutations discriminated exactly as predicted; the 4th (replacing the anchored while-loop SGR strip with an unanchored global `gsub`) showed **0 failures** — not a surprise once traced through: it generalizes a gap self-review had already disclosed (`while(sub(...))` vs a single `sub`) to the stronger claim that the strip's anchoring itself is currently undiscriminated, because the new `in_hunk` state machine alone already prevents the AR-1 regression class regardless of how the strip is written. All 4 live-demonstration scenarios matched the predicted exit codes exactly, including the intermediate (pre-AR-1-fix) scanner reproducing the regression. CI parity holds on all 4 cycle-1 fixtures plus a new full-history (1,607 commits, root..HEAD) added-line-set comparison: byte-identical, 213,915 lines each, 0 differences.

### 1. Execution

| Command | Result |
|---|---|
| `./scripts/run-test.sh` (full changed-scope run) | PASS — "All verifiers passed." Evidence: `docs/evidence/verify-2026-09-25-133950.log`. `tests/test-secret-scan.sh` 74/74, `tests/test-secret-scan-branch.sh` 108/108, `tests/test-run-verify-branch-secret-scan.sh` 32/32; full shell suite and `go test ./...` (8 packages) also green. |
| `sh tests/test-secret-scan.sh` (standalone) | 74/74 |
| `sh tests/test-secret-scan-branch.sh` (standalone) | 108/108 |
| `sh tests/test-run-verify-branch-secret-scan.sh` (standalone) | 32/32 |
| Same 3, with `TMPDIR=/tmp` | 74/74, 108/108, 32/32 — identical |

### 2. Environment matrix

| Variant | `test-secret-scan.sh` | `test-secret-scan-branch.sh` |
|---|---|---|
| `/bin/sh` | 74/74 | 108/108 |
| `/bin/dash` | 74/74 | 108/108 |
| `/bin/bash --posix` | 74/74 | 108/108 |
| Cycle-1 hostile outer config **+ `[attr] tree = 0000...0000`** | 74/74 (all 4 `GIT_ATTR_SOURCE`-related assertions individually confirmed: the uncommitted `.gitattributes` case, the local `attr.tree` case, the committed-not-working-tree case, and the unborn-HEAD case) | 108/108 |

The suite's own hermetic section (`HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM=/dev/null`) reassigns before any of the new `GIT_ATTR_SOURCE`-related fixtures run, so the outer `attr.tree` never reaches them — confirmed empirically, matching cycle 1's finding for the other hostile settings.

**awk portability (mawk, busybox awk):** rather than running the full suite inside a container (needs git + sh, more moving parts than the question warrants), I extracted the exact `scan_diff_stream` awk program and ran it directly under `docker run ubuntu:24.04 awk` (mawk 1.3.4) and `docker run busybox awk` against a hand-built fixture covering every branch: plain hunk content, a `+++ `-shaped hunk-content header look-alike, content starting with an escaped `ESC[32m++ `, content starting with two *stacked* SGR codes (`ESC[1mESC[32m`) both as file content and directly in front of the leading `+`, and a second file's own (also stacked-SGR-colored) header arriving after the first file's hunk (`in_hunk` reset via the `diff ` line). Both mawk and busybox awk produced output byte-identical to macOS's system awk (`awk version 20200816`) via `diff`. Docker Desktop's default bind-mount sharing does not cover `/private/tmp` on this machine — the mount silently became an empty directory instead of erroring, which cost one debugging round-trip; the fixture had to live under `$HOME` (cleaned up afterward) for the container mounts to see the actual file.

### 3. Repeat

10 sequential runs each, `sh`: `tests/test-secret-scan.sh` 74/74 every time (10/10), `tests/test-secret-scan-branch.sh` 108/108 every time (10/10). No flakes.

### 4. Red/green mutations

Applied in place to `scripts/secret-scan.sh` with the same single-occurrence Python replace used in cycle 1, tested, reverted with `git checkout --`, `git status --porcelain` confirmed clean after every revert and again at the end (byte-diff against a pre-mutation backup also clean).

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Drop the `in_hunk` state (`!in_hunk && /^\+\+\+ /` → `/^\+\+\+ /`, always skip `+++ `) | `range scan finds a token in content that starts with '++ '`, `--diff finds a token in hunk content that starts with '++ '` (2) | Exact |
| Drop the `diff ` reset (`/^diff / { in_hunk = 0; next }` removed, `@@`'s `in_hunk = 1` kept) | `range scan does not report a token-looking name in a +++ file header`, `--diff does not report a token-looking name in the +++ header after a hunk` (2) — `in_hunk` never resets for the second file, so its own `+++ ` header is no longer recognized as a header and gets reported as a false-positive finding instead | Exact |
| Replace `{ while (sub(/^\033\[[0-9;:]*m/, "")) {} }` with `{ gsub(/\033\[[0-9;:]*m/, "") }` (the *exact* pre-AR-1-fix strip, unanchored and global, reintroduced on top of the new `in_hunk` state machine) | **0 failures** | Not a surprise, but a real, now-more-precisely-characterized gap — see "Coverage gaps" below |
| Drop `GIT_ATTR_SOURCE=HEAD` from the `scan_range` git invocation | `range scan ignores an uncommitted working-tree .gitattributes marking the file -diff`, `range scan ignores a local attr.tree marking the file -diff`, `range scan reads the committed .gitattributes, not the working tree, as CI's checkout does`, `range scan with an unborn HEAD exits 3`, `range scan with an unborn HEAD names the unscanned range` (5) | Exact |

### 5. Live demonstration

Same isolated-fixture methodology as cycle 1 (isolated `HOME`/`GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM=/dev/null`, token from split pieces). Scanners: `main` (`git show main:scripts/secret-scan.sh`), the pre-AR-1-fix **intermediate** scanner (`git show f413a58:scripts/secret-scan.sh` — identical code to cycle-1's own HEAD, `b76d9da`, since the intervening commits up to `f413a58` are docs-only), and **new** (this worktree, `092c5e3`).

| Scenario | main | intermediate (pre-AR-1) | new |
|---|---|---|---|
| (a) committed line: content is ESC`[32m++ ` + token | exit=1, `secret-a.txt:1 [Stripe live secret key]` | **exit=0, clean — the regression** | exit=1, `secret-a.txt:1 [Stripe live secret key]` |
| (b) committed line: content is `++ ` + token (no escape) | exit=0, clean (pre-existing, CI-shared gap, now closed) | (not run — same code as cycle-1 baseline, already covered by (a)'s intermediate row) | exit=1, `secret-b.txt:1 [Stripe live secret key]` |
| (c) uncommitted working-tree `.gitattributes` (`*.txt -diff`) over a clean committed tree | (not applicable — pre-AR-1 scanner has no `GIT_ATTR_SOURCE` fix either way) | exit=0, clean (baseline with no `.gitattributes` edit: exit=1, detects normally — confirms the miss is specific to the edit, not a fixture error) | exit=1, `secret-c.txt:1 [Stripe live secret key]` |
| (d) a token-looking file name appearing only in a `+++ ` header | exit=0, clean | (not run) | exit=0, clean |

All 4 exactly as predicted. (a)/(b) together demonstrate the AR-1 fix closes both the colored and the plain `++ `-prefixed-content case; (a)'s three-way comparison specifically shows the fix is a genuine regression-fix against `main`'s own contemporary behavior, not just an improvement over the broken intermediate. (c) demonstrates Slice F end-to-end with an explicit before/after-edit baseline pair so the miss can't be mistaken for a fixture mistake. (d) confirms the new state machine doesn't newly break the pre-existing "don't scan header lines" behavior that `main` already had (both agree).

### 6. CI parity

**Cycle-1's 4 comparisons, re-run under the cycle-2 scanner (all still MATCH):**

| Range | main | new | Result |
|---|---|---|---|
| (a) plain leak, fresh fixture, default config | exit=1, 1 finding | exit=1, 1 finding | MATCH |
| (b) a branch with a pure rename | exit=0, 0 findings | exit=0, 0 findings | MATCH |
| (c) a branch with a submodule pointer bump | exit=0, 0 findings | exit=0, 0 findings | MATCH |
| (d) this repository's own range, `c1785b8..092c5e3` | exit=0, 0 findings | exit=0, 0 findings; full stdout+stderr byte-identical | MATCH |

**New this cycle — whole-repository history, added-line-set comparison (not just exit codes):** generated one canonical `git log -p` transcript for `root..HEAD` (root `868da02`, HEAD `092c5e3`, 1,607 commits, using the current `scan_range` pin set including `GIT_ATTR_SOURCE=HEAD`; 387,316 raw diff lines), then ran *only the awk extraction logic* (not the full scanner) from `main`'s `scan_diff_stream` and from the new one against that same transcript, and `diff`'d the two extracted added-line sets. **213,915 lines each, `diff` output empty — byte-identical.** This repository's own history contains zero instances of the widened case (a real added line whose own content happens to start with `++ ` or an SGR-escaped `++ `), so the AR-1 fix's "strict superset" widening is confirmed to change nothing about this repo's own historical scan results — consistent with cross-review's own 215,825-line oracle finding (a smaller number, from an earlier HEAD before this cycle's later commits).

No behavior difference found anywhere in this cycle's CI-parity checks.

### 7. Gaps (updated)

1. **`HUP`/`INT` signal-trap exit codes are still unexercised** (cycle-1 finding, re-confirmed unchanged at `092c5e3`: mutating only the `HUP` trap's exit-code literal again produces 0 failures across all 74 tests). No trap-related code was touched this cycle. Still not closed, same reasoning as cycle 1 (fixture cost vs. narrow, low-severity value).
2. **The leading-SGR strip's anchoring is unexercised, and this cycle's mutation 3 shows the gap is broader than self-review characterized it.** Self-review's own cycle-2 report names the gap as "`while(sub(...))` changed to a single `sub` — nothing red" (i.e., the *looping*, for stacked codes, is untested). This cycle's mutation went further — replacing the loop with a fully **unanchored global `gsub`** (the literal pre-AR-1-fix code) — and *also* produced 0 failures. Root cause, traced by hand: the `in_hunk` state machine added by the AR-1 fix is what actually prevents header-misdetection now; the strip's exact shape (looped vs. single, anchored vs. global) no longer changes whether a `+++ `-shaped hunk-content line gets mis-skipped, because `in_hunk` gates the skip rule regardless of what the stripped text looks like. This is a real, still-open, low-severity gap — not a new independent problem, but self-review's own characterization of it should be understood as the narrower case of this broader one. Recorded, not closed (per the "otherwise record gaps" instruction; closing it would mean constructing a scenario where anchoring specifically matters, and none was found to exist given `in_hunk`'s current coverage).
3. **`--no-ext-diff` — still not a gap, unchanged from cycle 1.** No cycle-2 change touched this pin or its rationale (git requires an explicit `--ext-diff` flag on `git log`, never passed by `scan_range`).
4. **Which git release introduced `GIT_ATTR_SOURCE` is still unconfirmed** — both self-review and this cycle verified its *effect* only on git 2.49 (this machine's version); neither checked the changelog for the exact minimum version (self-review's own header comment says "git 2.41+", not independently re-verified here). Low risk: an older git silently ignores the unknown env var and falls back to reading the working tree, matching pre-Slice-F behavior exactly (documented, tested via the version-gated `SKIP:` path in `tests/test-secret-scan.sh:184-195`, though that path itself is untested *on* an actual pre-2.41 git in this environment — this machine only has 2.49).

None of the above block the PASS verdict.

### Insight event (cycle 2)

```
./scripts/insights-append.sh --slug secret-scan-git-config --flow standard --phase test --cycle 2 --verdict pass --critical 0 --high 0 --medium 0 --low 0 --source skill
```

## Cycle 3 (final, run 3 of 3)

- Date: 2026-09-26
- HEAD: `28cd7c186087491463d475bf17d4527ff48cdc57` (worktree `git status --porcelain` empty at start and end; confirmed again after every mutation revert)
- Delta since cycle 2 (`b403b9f`): `dca6c2f` cycle-2 sync-docs, `df3193e` cycle-2 cross-review triage (**AR-2**: `GIT_ATTR_SOURCE=HEAD` broke `prepare-commit-msg-secret-guard.sh`'s merge-in-progress scan, `HEAD..$merge_head`, which does not end at HEAD; user raised the pipeline cap to 3), `004d99e` **Slice G** (attribute source = the range's own end commit, not always HEAD — fixed AR-2's exact shape but, per self-review run 3, only *moved* the misread to the mirror case, C3-H1), `63461d6` plan notes, `9e297e4` self-review run 3, `0e136fc` **Slice H** (final fix: pin `GIT_ATTR_SOURCE` to HEAD's own commit *only* when the range's resolved end equals HEAD; for any other end, explicitly `unset GIT_ATTR_SOURCE` — including one inherited from the caller's environment — and add `-c attr.tree=` so a local `attr.tree` can't step in either, falling back to the working tree's live attributes exactly as `main` did; a range end that doesn't resolve to a commit now exits 3 before `git log` even runs), `c2ad097` plan notes. `tests/test-secret-scan.sh` grew 74 → 86.

### Verdict: PASS

`tests/test-secret-scan.sh` 86/86, unchanged 108/108 and 32/32 for the other two files. Stable across 3 shells, the hostile outer config, 10 repeat runs, and two real git versions via Docker (2.40.4 — pre-`GIT_ATTR_SOURCE`, correctly version-gate-skips — and 2.43.7 — supports both `GIT_ATTR_SOURCE` and `attr.tree=`, fully exercises them). All 5 requested mutations discriminated exactly as predicted, including the two that reproduce the AR-2→C3-H1 arc directly (always-pin-the-end reproduces C3-H1's mirror miss; never-pin reproduces AR-2's original regression). All 4 live-demonstration scenarios matched the predicted exit codes exactly, once a self-caught fixture-construction mistake was fixed (see below). CI parity holds on all 4 cycle-1 fixtures, this repository's own range, and a new whole-history comparison for both a HEAD-ending and a non-HEAD-ending range — all byte-identical to `main`.

### 1. Execution

| Command | Result |
|---|---|
| `./scripts/run-test.sh` (full changed-scope run) | PASS — "All verifiers passed." Evidence: `docs/evidence/verify-2026-09-26-101048.log`. `tests/test-secret-scan.sh` 86/86, `tests/test-secret-scan-branch.sh` 108/108, `tests/test-run-verify-branch-secret-scan.sh` 32/32; full shell suite and `go test ./...` (8 packages) also green. |
| `sh tests/test-secret-scan.sh` / `test-secret-scan-branch.sh` / `test-run-verify-branch-secret-scan.sh` (standalone) | 86/86, 108/108, 32/32 |
| Same 3, with `TMPDIR=/tmp` | 86/86, 108/108, 32/32 — identical |

### 2. Environment matrix

| Variant | `test-secret-scan.sh` | `test-secret-scan-branch.sh` |
|---|---|---|
| `/bin/sh` | 86/86 | 108/108 |
| `/bin/dash` | 86/86 | 108/108 |
| `/bin/bash --posix` | 86/86 | 108/108 |
| Cycle-1/2 hostile outer config (`color.ui=always`, `diff.relative=true`, `diff.renames=copies`, `diff.algorithm=histogram`, `log.showSignature=true`, `core.bigFileThreshold=1`, `attr.tree=0000...0000`) | 86/86 (all merge-guard/attr-source assertions individually confirmed, including the two new merge-guard scenarios and both `A...B` cases) | 108/108 |
| git 2.40.4 (Docker, `alpine:3.18`, pre-`GIT_ATTR_SOURCE`) | 70/70 (the whole `attr_source_supported`-gated block correctly `SKIP:`s with "git 2.40.4 is older than 2.41", not a failure) | 108/108 |
| git 2.43.7 (Docker, `alpine:3.19`, supports `GIT_ATTR_SOURCE` **and** `attr.tree=`) | 85/85 (only the expected root-can-always-read `--file` skip; every attr-source assertion runs and passes) | 108/108 |

**Outer `GIT_ATTR_SOURCE` export — narrower than a full-suite run turned out to be meaningful.** Exporting a syntactically-invalid or merely-unresolvable `GIT_ATTR_SOURCE` (e.g. a nonexistent string) into the *whole test process's* environment breaks essentially all git operations, not just the scanner's own — confirmed by hand: `git add` itself fails with `fatal: bad --attr-source or GIT_ATTR_SOURCE` for any path that doesn't resolve in the *current* repo, and since `test-secret-scan.sh` builds dozens of independent throwaway fixture repos, no single external value can be simultaneously valid in all of them; a garbage or foreign-repo value just breaks fixture construction itself, which tests git's own eagerness, not the scanner. The meaningful, realistic version of this check — a **valid-but-wrong** commit id, from within the actual fixture repo the scan is running in — is exactly what the shipped test "an inherited `GIT_ATTR_SOURCE` does not choose the merge guard's attributes" already does (part of the 86/86 above). I additionally re-verified by hand, directly against `scan_range`'s subshell, that even a completely garbage/unresolvable inherited `GIT_ATTR_SOURCE` is fully neutralized for **both** branches: `unset GIT_ATTR_SOURCE`-branch (merge guard, range not ending at HEAD) → `exit=0`, no `fatal:`; `GIT_ATTR_SOURCE=$head_commit`-branch (CI/branch-scan shape) → `exit=0`, no `fatal:` — the explicit reassignment/unset inside the subshell overwrites the inherited value either way, regardless of how badly it was poisoned.

### 3. Repeat

10 sequential runs, `sh tests/test-secret-scan.sh`: 86/86 every time (10/10), no flakes. (This suite took longer than the 180s default timeout to run 10x back-to-back this cycle — the new real-`git merge`-based fixtures add real work per run — so it was moved to a background task; full output confirmed 10/10 clean.)

### 4. Red/green mutations

Applied in place to `scripts/secret-scan.sh`, tested, reverted with `git checkout --`, `git status --porcelain` and a byte-diff against a pre-mutation backup both confirmed clean after every revert.

| Mutation | Failing test(s) | Match? |
|---|---|---|
| Always pin the range end's own commit (Slice G behavior: `GIT_ATTR_SOURCE=$end_commit` unconditionally, no `if`/`else`) | `A...B with B other than HEAD reads the working tree's attributes`, `merge where HEAD's side drops -diff: HEAD..<merge head> range finds the token`, `merge where HEAD's side drops -diff: prepare-commit-msg guard finds the token`, `an inherited GIT_ATTR_SOURCE does not choose the merge guard's attributes`, `a local attr.tree does not replace the merge guard's working-tree attributes` (5) — exactly C3-H1's mirror-case tests; the *original* AR-2-shape merge tests stay green, since Slice G's own approach happens to fix that specific shape | Exact — reproduces C3-H1 |
| Never pin (`main` behavior: `unset GIT_ATTR_SOURCE` unconditionally) | `range scan ignores an uncommitted working-tree .gitattributes marking the file -diff`, `a ..HEAD range reads attributes from HEAD, not the working tree`, `a range with an empty end reads attributes from HEAD, not the working tree`, `range scan reads the committed .gitattributes, not the working tree, as CI's checkout does`, `A...B with B at HEAD reads HEAD's attributes` (5) — exactly the HEAD-pin tests; both merge-guard tests stay green, since that is exactly `main`'s own (correct-for-that-case) behavior | Exact — reproduces the original AR-2 regression |
| Drop the `else` branch's `unset GIT_ATTR_SOURCE` (keep the `if`, drop only the `else`) | `an inherited GIT_ATTR_SOURCE does not choose the merge guard's attributes` (1) | Exact |
| Drop `-c attr.tree=` | `a local attr.tree does not replace the merge guard's working-tree attributes` (1) — the HEAD-ending `attr.tree` test stays green, since `GIT_ATTR_SOURCE` already takes precedence over `attr.tree` on that branch regardless | Exact |
| Drop the range-end resolution `exit 3` check (fold it into a soft `\|\| end_commit=""` instead) | `a range whose end does not resolve names that end` (1 — the *specific* diagnostic message goes missing) | The exit-code guarantee itself does **not** regress — `a range whose end does not resolve exits 3` and both unborn-HEAD tests stay green, because an unresolvable range end still makes the later `git log` call fail, which the pre-existing `log_rc` check catches and reports (with a more generic message). The dedicated check's value is a faster, more specific diagnostic, not the exit-code contract itself. |

### 5. Live demonstration

Real merges, the real `scripts/prepare-commit-msg-secret-guard.sh` (copied into each fixture's own `scripts/` so it resolves the scanner under test), three scanners: `main` (`git show main:...`), **Slice G** (`git show 9e297e4:scripts/secret-scan.sh`), and **new** (this worktree, `28cd7c1`).

| Scenario | main | Slice G | new |
|---|---|---|---|
| AR-2 shape (HEAD keeps `-diff`, incoming drops it and adds-then-deletes a token) | range=1, guard=1 | range=1, guard=1 | range=1, guard=1 |
| mirror shape (HEAD's side drops `-diff`, incoming — forked before that — keeps it and adds-then-deletes a token) | range=1, guard=1 | **range=0, guard=0 (missed)** | range=1, guard=1 |
| `--range base..HEAD` with an uncommitted working-tree `-diff` (confirms the HEAD-pin still works) | exit=0 (misses — reads the live working tree, as `main` always does) | (not run) | exit=1 (detects — pinned to HEAD's own committed tree, which has no `.gitattributes` at all) |

Exactly as predicted: main 1/1, Slice G 1/0, new 1/1 for the two merge scenarios; main 0, new 1 for the HEAD-pin check.

**Self-caught fixture mistake, fixed before reporting:** my first attempt at the third scenario reused the *same* repo as the two merge scenarios above it, whose `main` branch had already accumulated a real, committed `*.txt -diff` from the AR-2-shape setup earlier in the script. Re-adding an "uncommitted" `.gitattributes` with the same content on top of that wasn't actually testing an uncommitted-attribute bypass — it was re-stating an attribute already legitimately present in HEAD's own committed tree, so both `main` and `new` correctly matched at `exit=0` for the wrong reason. Fixed by building the third scenario in a completely fresh, `.gitattributes`-history-free repo instead; the corrected run reproduces the predicted `main=0, new=1` split.

### 6. CI parity

**Cycle-1's 4 fixtures, re-run under the cycle-3 scanner (all still MATCH):** plain leak, pure rename, submodule pointer bump, this repository's own range (`c1785b8..28cd7c1`) — same table shape as cycles 1–2, all MATCH, `d`'s stdout+stderr byte-identical.

**Whole-history comparison, both range shapes:**

| Range | main | new | Result |
|---|---|---|---|
| Ends at HEAD: `868da02..HEAD` (root, 1,616 commits) | exit=0, 0 findings | exit=0, 0 findings; full output byte-identical | MATCH |
| Does **not** end at HEAD: `868da02..HEAD~50` | exit=0, 0 findings | exit=0, 0 findings; full output byte-identical | MATCH |

No behavior difference anywhere in this repository's own real history, for either range shape — the attribute-source fix changes nothing about what actually gets scanned here today, same conclusion as cycle 2's AR-1 finding.

### 7. Gaps (updated)

1. **`HUP`/`INT` signal-trap exit codes are still unexercised** (unchanged from cycles 1–2; re-confirmed at `28cd7c1`, 86-test file: mutating only the `HUP` trap's exit code again produces 0 failures). No trap-related code was touched this cycle.
2. **The leading-SGR strip's anchoring is still unexercised** (unchanged from cycle 2; re-confirmed at `28cd7c1`: replacing the anchored while-loop with the pre-AR-1-fix unanchored global `gsub` again produces 0 failures, for the same reason as cycle 2 — `in_hunk` fully gates header-misdetection regardless). No `scan_diff_stream` code was touched this cycle.
3. **New this cycle — the `A...B` residual is real, disclosed, and now concretely demonstrated (not just described), but remains low-priority and untested-by-name since no caller uses this form.** For a symmetric-difference range `A...B` where `B` is the resolved range end and equals HEAD, `GIT_ATTR_SOURCE` pins to `B`'s (HEAD's) tree for *every* commit shown, including ones reachable only from `A` — whose *own* tree may have a different attribute state. Built a direct fixture: `unique-a` (forked from a clean base, no `.gitattributes` anywhere in its history, adds a token) scanned directly (`base..unique-a`) detects the token (exit 1, as expected); the same commit scanned via `unique-a...head-with-diff` (with `head-with-diff`, which *does* commit `*.txt -diff`, checked out as HEAD) is **missed** (exit 0) — `head-with-diff`'s `-diff` attribute gets wrongly applied to `unique-a`'s own commit. This is the asymmetric-miss direction of the residual the plan/self-review only described in general terms; verified concretely this cycle. Two existing tests (`A...B with B other than HEAD reads the working tree's attributes`, `A...B with B at HEAD reads HEAD's attributes`) pin the *documented* behavior by name, but neither is actually the two-unrelated-branches-with-divergent-attribute-state shape that produces a real miss — so the miss itself remains unexercised by name, even though the underlying mechanism (HEAD's attributes apply to the whole symmetric-difference output) is fully intentional and disclosed (`scripts/secret-scan.sh:229-236`, `docs/tech-debt/README.md`'s new C3-L3 row). No caller in this repository passes an `A...B` range (`verify.yml`, `secret-scan-branch.sh`, and `prepare-commit-msg-secret-guard.sh` all pass plain `A..B`), so this stays a documented, low-priority residual rather than something to fix or add a dedicated regression test for.
4. **git 2.41.x/2.42.x specifically unrun.** This cycle exercised git 2.40.4 (pre-`GIT_ATTR_SOURCE`, correctly version-gates) and 2.43.7 (supports both `GIT_ATTR_SOURCE` and `attr.tree=`) via Docker/alpine. The narrower band that supports `GIT_ATTR_SOURCE` (2.41+) but not yet `attr.tree=` (2.43+) — i.e. exactly 2.41.x or 2.42.x — was not tested; no readily available alpine tag ships exactly that range (`3.18`→2.40, `3.19`→2.43). Low risk: `-c attr.tree=` on a git that doesn't understand `attr.tree` at all is simply ignored the same way any unknown `-c` key is (documented in the header, `scripts/secret-scan.sh:9-13`), and the version-gate test in `tests/test-secret-scan.sh` only checks `>= 2.41` for the `GIT_ATTR_SOURCE`-dependent assertions as a group, not the `attr.tree=`-specific sub-behavior separately — a git in the 2.41-2.42 band would take the version-gate's "supported" branch and could, in principle, behave differently for the `attr.tree=` cases specifically. Not confirmed either way this cycle.
5. **Which git release introduced `GIT_ATTR_SOURCE` is still unconfirmed** (unchanged from cycle 2 — both self-review and testing across all 3 cycles verified its *effect* only empirically, on 2.40.4/2.43.7/2.49.0, not against git's own changelog).

None of the above block the PASS verdict.

### Insight event (cycle 3)

```
./scripts/insights-append.sh --slug secret-scan-git-config --flow standard --phase test --cycle 3 --verdict pass --critical 0 --high 0 --medium 0 --low 0 --source skill
```
