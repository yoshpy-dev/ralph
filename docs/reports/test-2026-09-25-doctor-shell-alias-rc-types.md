# Test report: doctor-shell-alias-rc-types

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md` (issue #167)
- Tester: `tester` subagent (Claude Code, standard flow), cycle 1
- Branch: `fix/doctor-shell-alias-rc-types`, HEAD `904e5c1` (base `main`)
- Scope: behavioral tests only (changed-language scope: Go). No static
  analysis, linting, or drift checks — those already passed in
  `docs/reports/verify-2026-09-25-doctor-shell-alias-rc-types.md`; no diff-quality
  review is included (self-review already ran at `fadd3d8`/`b611661`).
- Evidence: `docs/evidence/verify-2026-09-25-042633.log`

## Overall verdict: PASS

`./scripts/run-test.sh` is green end to end (0 shell-suite failures, all 8 Go
packages `ok`). The `internal/cli` package's `ShellAlias`-scoped tests (153
`--- PASS` lines across 73 top-level funcs) pass under `-race`, under
`TMPDIR=/tmp`, and repeated 20x (plain and `-race`), with the three FIFO
tests never hanging. All 5 red/green mutations discriminate exactly as the
plan predicted (two of them a little more strongly than predicted — see the
mutation table). A live-binary demonstration confirms the FIFO-avoidance fix
completes in well under a second (plan asked for a 10s bound) and that a
relative `$ZDOTDIR`, including one containing `link/..` through a symlink,
resolves the same way the OS (and a real zsh `source`) resolves it. Edge
cases (`ZDOTDIR=/`, `.`, trailing `//`, relative+empty `Cwd`, a symlink to
`/dev/null`) all behave as the plan's Risks section anticipates. Working
tree is clean (`git status --porcelain` empty) and HEAD is unchanged from
`904e5c1`.

## 1. Execution

| # | Command | Result |
| --- | --- | --- |
| 1a | `./scripts/run-test.sh` | PASS — every shell suite `FAIL: 0`; `internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade` all `ok`. Evidence: `docs/evidence/verify-2026-09-25-042633.log` |
| 1b | `go test ./internal/cli/ -run 'ShellAlias' -count=1 -race -v` | PASS — 73 `--- PASS` top-level + 80 `    --- PASS` subtest lines (153 total), 0 FAIL, `ok` in 2.714s |
| 1c | `TMPDIR=/tmp go test ./internal/cli/... -count=1` | PASS — `ok` in 45.126s (full package, CI-like `TMPDIR`) |

`tests/test-ralph-dispatch.sh`'s "I. SIGTERM cleanup left stray
ralph-dispatch-* temp files" test — the known out-of-scope flake noted in
the handoff — **passed** in this run (`docs/evidence/verify-2026-09-25-042633.log:513`),
so no standalone rerun was needed.

## 2. Repeat

| Command | Result |
| --- | --- |
| `go test ./internal/cli/ -run 'ShellAlias' -count=20` | `ok` in 20.579s |
| `go test ./internal/cli/ -run 'ShellAlias' -count=20 -race` | `ok` in 23.504s |

The three unix FIFO tests (`TestCheckShellAliases_ZshrcIsFIFO_InfoAndCompletes`,
`TestCheckShellAliases_ZshrcSymlinksToFIFO_InfoNamedBySymlink`,
`TestCheckShellAliases_FIFOAndOtherRcAlias_WarnsAndNamesBoth`) never hung
across either repeat run — each returns near-instantly (well under their own
5s `checkShellAliasesWithin` guard), consistent with `scanShellAliasFile`
never calling `os.Open` on a non-regular candidate.

## 3. Red/green mutations

All mutations applied to the working tree, tested, then reverted with
`git checkout -- internal/cli/doctor_shell_alias.go`; `git status --porcelain`
was empty after every revert and `go build ./...` succeeded at the end.

| # | Mutation | Predicted | Actual |
| --- | --- | --- | --- |
| (a) | Remove the `IsRegular` check in `scanShellAliasFile` | FIFO tests fail by 5s timeout, package still finishes | Matched exactly: the 3 FIFO tests fail with `did not return within 5s`, package finishes in 17.0s (no hang) |
| (b) | `shellAliasJoinRaw` → `filepath.Join` in the `$ZDOTDIR` candidate loop | AC-5b/5c/5d fail | Matched, plus one extra: `TestCheckShellAliases_RelativeZdotdirWithDotDotThroughSymlink_FollowsOSResolution`, `TestCheckShellAliases_CwdThroughSymlinkWithDotDotZdotdir_FollowsOSResolution`, `TestCheckShellAliases_AbsoluteZdotdirWithDotDotThroughSymlink_FollowsOSResolution` (the 5b/5c/5d trio) **and** `TestCheckShellAliases_RelativeZdotdirResolvesToHome_DedupedToOne` all fail — stronger discrimination than predicted |
| (c) | Drop the relative-`$ZDOTDIR` branch (return `""` when not absolute) | AC-5 fails | Matched, plus related relative-ZDOTDIR tests: `TestCheckShellAliases_RelativeZdotdir_Detected` (AC-5 itself), `TestCheckShellAliases_RelativeZdotdirWithDotDotThroughSymlink_FollowsOSResolution`, `TestCheckShellAliases_CwdThroughSymlinkWithDotDotZdotdir_FollowsOSResolution`, `TestCheckShellAliases_RelativeZdotdirResolvesToHome_DedupedToOne` all fail (all are relative-`$ZDOTDIR` scenarios, so all correctly depend on the branch under test) |
| (d) | `p = filepath.Clean(p)` at the top of `displayPath` | AC-5b/5c/5d fail | Matched exactly the same 4 tests as (b) — expected, since (b) and (d) both defeat the "raw, unresolved `..`" property the same way |
| (e) | Revert `shellAliasRawParent`'s root case (`i==0` → old combined `if i>=0 { return path[:i] }`, making `/.zshrc`'s parent `""` instead of `/`) | Its table test fails | Matched exactly: `TestShellAliasRawParent//.zshrc` fails, all other `TestShellAliasRawParent` subtests pass |

## 4. Live demonstration

Built `ralph-new` via `go build -o <scratch>/ralph-new ./cmd/ralph`. Every
invocation below used an explicit fake `HOME` and either `env -u ZDOTDIR` or
an explicit `ZDOTDIR` set to a scratch path, so no real dotfile or real
`$ZDOTDIR` was ever read.

**(i) Fake HOME, FIFO `.zshrc`.** `env -u ZDOTDIR HOME=<fakehome1> ralph-new doctor`,
backgrounded with a 10s kill-guard (macOS has no `timeout`): exited 0 in
under a second (no hang), with:
```
ℹ Shell aliases (codex/claude): info — could not read: ~/.zshrc (not a regular file) — aliases there were not checked. scanned 0 shell rc file(s) (files they source are not followed)
```

**(ii) Fake HOME, scratch cwd, `ZDOTDIR=relative-rc`**, with
`<cwd>/relative-rc/.zshrc` holding `alias codex="codex --model fake-model"`:
```
⚠ Shell aliases (codex/claude): warn — alias codex in <cwd>/relative-rc/.zshrc:1 adds --model — herdr expands the alias in the seat's pane and codex rejects a flag given twice ("cannot be used multiple times"); ralph org spawn always passes --model, so every spawn fails; remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md). scanned 1 shell rc file(s): <cwd>/relative-rc/.zshrc (files they source are not followed)
```
The Detail names the fully resolved absolute path, as AC-5 requires.

**(iii) `ZDOTDIR=link/../rc`**, `<cwd>/link` a symlink to `<X>/config`, alias
placed at `<X>/rc/.zshrc` (not `<cwd>/rc/.zshrc`, which was confirmed to not
even exist):
```
⚠ Shell aliases (codex/claude): warn — alias codex in <cwd>/link/../rc/.zshrc:1 adds --model — ...
```
The Detail names the **raw** candidate path (`link/../rc/.zshrc`, `..` left
unresolved), matching AC-5b's "report the resolved-in-substance, unresolved-
in-text" behavior, while the alias it actually found was read from
`<X>/rc/.zshrc` — the OS-resolved-through-the-symlink target, not the
lexical `<cwd>/rc/.zshrc`.

**zsh cross-check.** This host's `/etc/zshenv` unconditionally sets
`ZDOTDIR=$HOME/.config/zsh` (`cat /etc/zshenv` → `ZDOTDIR=$HOME/.config/zsh`,
no guard), which zsh always sources first regardless of an inherited
`$ZDOTDIR` env var, so a genuine end-to-end "real login-zsh pane reads our
custom `$ZDOTDIR`" demo isn't possible on this host without editing
`/etc/zshenv` (out of scope). Instead, confirmed the narrower, still
zsh-specific claim directly: from `<cwd3>` (case iii's fixture, still on
disk), `zsh -f -c 'source link/../rc/.zshrc; alias codex'` printed
`codex='codex --model fake-model'` — zsh's own `source` builtin resolves the
raw candidate string to the exact same physical file
(`<X>/rc/.zshrc`, confirmed separately by `realpath`) that the built binary
scanned. This validates the OS-resolution assumption the plan's Design
decisions section rests on, decoupled from the local `/etc/zshenv` override.

## 5. Edge cases

Verified via a throwaway `internal/cli/zz_scratch_edge_test.go` (written,
run, then deleted — not part of any commit; `git status --porcelain` was
empty again immediately after):

| Edge case | Result |
| --- | --- |
| `ZDOTDIR=/` (candidate `/.zshenv` etc.) | `shellAliasZdotdirBase` returns `"/"`; `shellAliasRcCandidates` returns an empty candidate list on this real machine (none of `/.zshenv`/`/.zshrc`/etc. exist at real root) — no panic; sanity-checked `shellAliasJoinRaw("/", ".zshenv") == "/.zshenv"` |
| `ZDOTDIR=.` | Base resolves to `<Cwd>/.` (left unclean, by design) — no panic |
| `ZDOTDIR=/a//` (trailing `//`) | Base stays `"/a//"` unchanged (not cleaned), but `shellAliasJoinRaw("/a//", ".zshrc")` still collapses correctly to `/a/.zshrc` |
| Relative `ZDOTDIR` + empty `Cwd` (unit-level via the env struct) | `shellAliasZdotdirBase` returns `""`; the resulting candidate list is byte-for-byte identical to `ZDOTDIR` fully unset — confirms AC-6 |
| rc is a symlink to `/dev/null` | `scanShellAliasFile` returns `Opened=false`, `Err.Error() == "not a regular file"`; `checkShellAliases` status is **`info`**, not `warn`/`fail` — matches the plan's Risks note exactly |

## 6. Coverage

`go test ./internal/cli/ -coverprofile=... -count=1`: package `ok`, **84.4%**
of statements (whole package).

Per-function coverage for the six functions named in the handoff:

| Function | Coverage |
| --- | --- |
| `shellAliasJoinRaw` | 100.0% |
| `shellAliasRawParent` | 100.0% |
| `shellAliasZdotdirBase` | 100.0% |
| `shellAliasRcCandidates` | 97.1% |
| `scanShellAliasFile` | 97.0% |
| `shellAliasEnvFromOS` | 100.0% |

The five newly-added or changed pieces from this plan's Slice B
(`shellAliasEnv.Cwd`, `shellAliasJoinRaw`, `shellAliasRawParent`,
`shellAliasZdotdirBase`, the `$ZDOTDIR`-candidate loop) are all at 100%
except `shellAliasRcCandidates`'s one uncovered line (below, pre-existing
defensive code, not part of this plan's diff behavior). See §7 for exactly
which lines are uncovered and why.

## 7. Failure analysis

No unexpected failures anywhere in this cycle. Every FAIL observed was a
deliberately-applied, deliberately-reverted mutation (§3).

Five statement ranges in `doctor_shell_alias.go` are uncovered (0 hits in
the coverage profile), none touched by this plan's mutations or by the
plan's own AC list — recorded as gaps in §8, not failures:

1. `internal/cli/doctor_shell_alias.go:196-198` — the `filepath.EvalSymlinks`
   failure fallback (`real = c`) inside `shellAliasRcCandidates`. Only
   reachable if `EvalSymlinks` fails on a path that just passed `os.Stat`
   (a TOCTOU race or an exotic permission edge) — pre-existing, unrelated
   to this plan.
2. `internal/cli/doctor_shell_alias.go:321-323` — the unquoted,
   outside-any-quote backslash-newline branch in `shellAliasStatements`.
   The existing "backslash-newline joins across the join point" test
   exercises this inside a double-quoted value (a different branch,
   covered); no test hits the bare top-level case.
3. `internal/cli/doctor_shell_alias.go:378-380` — `parseAliasStatements`'s
   marker-skip loop over an *unterminated* trailing statement (e.g. a
   dangling `then alias codex='unterminated`). All existing "open" tests
   have `alias` as the very first word of the trailing statement.
4. `internal/cli/doctor_shell_alias.go:497-499` — `scanShellAliasFile`'s own
   `os.Stat` error branch. Unreachable in practice today:
   `shellAliasRcCandidates` only calls `scanShellAliasFile` on paths that
   already passed its own `os.Stat`, so this is defensive code for the same
   TOCTOU race the plan's Non-goals explicitly excludes.
5. `internal/cli/doctor_shell_alias.go:926` — the plural `"have"` branch
   when `len(harmlessItems) > 1`. No existing test has two or more harmless
   (no-conflicting-flags) alias findings in the same run.

## 8. Regression checks

- `./scripts/run-test.sh` ran all 8 Go packages (not just `internal/cli`),
  all `ok`: `internal/config`, `internal/insights`, `internal/org` (9.243s),
  `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`,
  `internal/upgrade`.
- `TMPDIR=/tmp go test ./internal/cli/... -count=1` (§1c) is itself a
  regression check for the pre-existing `runDoctor*`-based tests under a
  CI-like `TMPDIR`.
- The pre-existing `TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault`
  and `TestRunDoctorOpts_ShellAliasesCheck_WarnsThroughTheSeam` integration
  tests (both scaffold a real project into a temp dir) pass unchanged.
- No secret-looking literal appears in any test, the live-demo output
  captured above, or this report — alias values used throughout are the
  placeholder `fake-model`, never a real model slug or credential.

## 9. Test gaps

- The five uncovered statement ranges in §7 — none required by this plan's
  ACs, but worth a future note if `doctor_shell_alias.go` is touched again
  for an unrelated reason (same convention as the plan's own tech-debt
  pointer for this file).
- `TestCheckShellAliases_RelativeZdotdirResolvesToHome_DedupedToOne`
  discriminates mutations (b), (c), and (d) simultaneously (§3) — not a
  gap, but worth noting it is currently doing more discrimination work than
  its name suggests; a future maintainer renaming/removing it should be
  aware it also covers part of AC-5b/5c/5d's intent.
- Per the plan's own Non-goals, the stat/open TOCTOU race (a candidate
  swapped for a FIFO between `shellAliasRcCandidates`'s stat and
  `scanShellAliasFile`'s stat) is intentionally untested — matches gap #1/#4
  above.
- Windows is out of scope (plan Non-goals); no FIFO-equivalent test exists
  for it, by design (the FIFO tests live in a `//go:build !windows` file).

## Commit

- `docs/reports/test-2026-09-25-doctor-shell-alias-rc-types.md` and the
  matching insight event are the only files staged by this cycle; no
  source file was edited (the scratch edge-case test file was deleted
  before this report was written).
