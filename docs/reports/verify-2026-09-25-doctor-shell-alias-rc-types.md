# Verify report: doctor-shell-alias-rc-types

- Date: 2026-09-25
- Plan: docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md
- Verifier: verifier subagent (cycle 1)
- HEAD: 413e4b47fd97e109f84634f673379cf8341f2928 (branch fix/doctor-shell-alias-rc-types)
- Worktree: /Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/doctor-shell-alias-rc-types
- `git status --porcelain` empty before and after this run

## Static analysis

- `./scripts/run-static-verify.sh` (changed-language scope): PASS — gofmt ok, 0 issues, `scripts/check-sync.sh` 0 DRIFTED, `scripts/check-skill-sync.sh` 13 skills in lock-step, `scripts/check-template-purity.sh` clean, branch secret scan (`a5aafc3..413e4b4` vs `origin/main`) clean. Evidence: `docs/evidence/verify-2026-09-25-042010.log`.
- `gofmt -l internal/cli` — no output (clean).
- `go vet ./internal/cli/...` — no output (clean).
- `./scripts/run-verify.sh` (full scope, run once as plan-authorized evidence): PASS, exit 0. `go test` ran the whole module (`internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade`) — all `ok`. `tests/test-ralph-dispatch.sh`'s "I. SIGTERM cleanup left no stray ralph-dispatch-* temp files in $TMPDIR" passed on this run (no re-run needed; the plan's known flake did not reproduce). Evidence: `docs/evidence/verify-2026-09-25-042128.log` (grep-checked: only `FAIL: 0` summary lines, no `FAIL` result lines).
- Direct scoped run: `TMPDIR=/tmp go test ./internal/cli/... -run TestCheckShellAliases -count=1 -v` — 14 top-level tests (+2 subtests) PASS, 0.783s.
- gopls "modernize" hints: none checked separately: `run-static-verify.sh`'s golang verifier reported "0 issues" and no modernize-hint output appeared in either evidence log, so nothing to attribute to lines this branch added.

## Spec compliance (AC-1 .. AC-9)

| AC | Verdict | Evidence |
|----|---------|----------|
| AC-1 (FIFO rc → info, <5s, "not a regular file") | PASS | `internal/cli/doctor_shell_alias.go:496-503` (`scanShellAliasFile` stats first, returns `Opened:false, Err: errors.New("not a regular file")` without `os.Open` when `!info.Mode().IsRegular()`); `internal/cli/doctor_shell_alias_unix_test.go:59-75` (`TestCheckShellAliases_ZshrcIsFIFO_InfoAndCompletes`, 5s guard via `checkShellAliasesWithin`, `internal/cli/doctor_shell_alias_unix_test.go:37-50`) |
| AC-2 (FIFO via symlink, named by symlink candidate) | PASS | `internal/cli/doctor_shell_alias_unix_test.go:82-102` (`TestCheckShellAliases_ZshrcSymlinksToFIFO_InfoNamedBySymlink`) |
| AC-3 (FIFO doesn't block scan of other candidates; other rc's codex alias still warns) | PASS | `internal/cli/doctor_shell_alias_unix_test.go:108-129` (`TestCheckShellAliases_FIFOAndOtherRcAlias_WarnsAndNamesBoth`) |
| AC-4 (directory candidate silently skipped, no "could not read") | PASS | `internal/cli/doctor_shell_alias.go:193-195` (`shellAliasRcCandidates` skips `info.IsDir()`); `internal/cli/doctor_shell_alias_test.go:1065-1081` (`TestCheckShellAliases_ZshrcIsDirectory_SkippedSilently`) |
| AC-5 (relative `$ZDOTDIR` + Cwd resolves and detects) | PASS | `internal/cli/doctor_shell_alias.go:104-115` (`shellAliasZdotdirBase`); `internal/cli/doctor_shell_alias_test.go:276-292` (`TestCheckShellAliases_RelativeZdotdir_Detected`, exact-item assertion) |
| AC-5b (`link/../rc` relative, resolved via symlink target, not lexically) | PASS | `internal/cli/doctor_shell_alias.go:74-76,104-115` (`shellAliasJoinRaw`, no `filepath.Join`); `internal/cli/doctor_shell_alias_test.go:315-339` (`TestCheckShellAliases_RelativeZdotdirWithDotDotThroughSymlink_FollowsOSResolution`) |
| AC-5c (Cwd itself through a symlink, `$ZDOTDIR=".."`) | PASS | `internal/cli/doctor_shell_alias_test.go:345-370` (`TestCheckShellAliases_CwdThroughSymlinkWithDotDotZdotdir_FollowsOSResolution`) |
| AC-5d (absolute `$ZDOTDIR` with `link/..`, Cwd irrelevant) | PASS | `internal/cli/doctor_shell_alias_test.go:376-400` (`TestCheckShellAliases_AbsoluteZdotdirWithDotDotThroughSymlink_FollowsOSResolution`, `Cwd=""` proves irrelevance) |
| AC-6 (relative `$ZDOTDIR`, empty Cwd → same as unset, no panic) | PASS | `internal/cli/doctor_shell_alias.go:111-113` (`env.Cwd == "" → return ""`); `internal/cli/doctor_shell_alias_test.go:297-308` (`TestCheckShellAliases_RelativeZdotdirEmptyCwd_SameAsUnset`) |
| AC-7 (`shellAliasEnvFromOS` fills Cwd via `os.Getwd()`; absolute/unset unaffected) | PASS | `internal/cli/doctor_shell_alias.go:35-42`; `internal/cli/doctor_shell_alias_test.go:461-516` (`TestShellAliasEnvFromOS` subtests, incl. `t.Chdir` + `filepath.EvalSymlinks` comparison for macOS `/tmp` symlink); existing absolute-`$ZDOTDIR`/unset tests (e.g. `TestCheckShellAliases_Zdotdir_Detected`) unmodified and passing |
| AC-8 ("would refuse" removed; comment states relative-`$ZDOTDIR` resolution + cwd approximation) | PASS | `grep -n "would refuse" internal/cli/doctor_shell_alias.go` → no match (re-grepped, exit 1); `internal/cli/doctor_shell_alias.go:134-153` (doc comment: "zsh resolves a relative $ZDOTDIR against the shell's own working directory... This check has no way to observe the herdr pane's actual working directory... approximates with doctor's own working directory instead") |
| AC-9 (`run-verify.sh` green, `go test ./internal/cli/...` pass) | PASS | `docs/evidence/verify-2026-09-25-042128.log` (exit 0, only `FAIL: 0` lines); `internal/cli` package `ok` in the same run; scoped `-run TestCheckShellAliases -v` run above also green |

## Non-goals check

- No `source`-following added: `shellAliasRcCandidates`'s doc comment (`internal/cli/doctor_shell_alias.go:127-130`) still states the candidate list is static and `source`d files are not followed; no new code path opens a file discovered via a scanned `source` statement.
- No seat-cwd resolution from `ralph.toml`/org state: `shellAliasEnvFromOS` (`internal/cli/doctor_shell_alias.go:35-42`) only calls `os.Getwd()`; no `internal/config` or `internal/org` import was added to `doctor_shell_alias.go` (confirmed by reading the full file — import block at lines 3-12 is unchanged in kind: `bufio`, `errors`, `fmt`, `io/fs`, `os`, `path/filepath`, `strings`, `syscall`).
- No `/dev/null` special case: `scanShellAliasFile` treats any non-regular candidate uniformly via `info.Mode().IsRegular()`; no path-literal check for `/dev/null` exists.
- Severity rules unchanged: `checkShellAliases`'s status `switch` (`internal/cli/doctor_shell_alias.go:943-950`) is untouched by this diff — `git diff main...HEAD -- internal/cli/doctor_shell_alias.go` shows no hunk touching that block; all pre-existing severity-grading tests (`TestCheckShellAliases_CodexSandboxOnly_*`, `_ClaudePermissionMode_*`, etc.) still pass.
- No Windows-specific production code added: `doctor_shell_alias.go` carries no build tag (unix and windows both run it, as before); the only new file with unix-only content is `doctor_shell_alias_unix_test.go`, tagged `//go:build !windows` and living in a `_test.go` file (test-only, matches plan).

## Documentation drift

- `docs/recipes/codex-seat-permissions.md` (root and `templates/base/` copy): byte-identical (`diff` clean); content describes alias severity/flag behavior, not the candidate list or `$ZDOTDIR` resolution mechanics — no drift from this fix.
- `.claude/skills/org/SKILL.md` and its three mirrors (`.agents/skills/org/`, `templates/base/.claude/skills/org/`, `templates/base/.agents/skills/org/`): all four MD5-identical; same scope note as above — no drift.
- `docs/tech-debt/README.md:132` (the `checkShellAliases` config-aware-grading row): content is still accurate — this fix (FIFO handling + relative `$ZDOTDIR`) does not touch the flag-class/severity grading the row describes. However, the row's own trigger condition ("next touch to `internal/cli/doctor_shell_alias.go` for another reason") has now fired, per the plan's self-review deviation note (`docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md:105`, "このPRで発火した"). **This is a doc-drift item for `/sync-docs`**, not a verify failure: the row needs an explicit annotation (e.g. re-affirm as still-open, or note the trigger fired and was consciously deferred) rather than silent staleness.
- `docs/specs/`: no spec file references `ZDOTDIR`, `shellAliasRcCandidates`, or the "Shell aliases (codex/claude)" check name (grep clean) — nothing to update.
- `grep -rn "would refuse"` across the worktree: only historical/report docs (plan's own "Related request" and deviation-note prose describing this fix, the archived predecessor plan, the cross-review triage report, a walkthrough report) plus one unrelated self-review about a different file (`org-send-enter-timing`). No stray production-doc reference to the removed comment text.

## Diff hygiene

`git diff main...HEAD --stat`: 6 files — `internal/cli/doctor_shell_alias.go`, `internal/cli/doctor_shell_alias_test.go`, `internal/cli/doctor_shell_alias_unix_test.go` (the plan's affected areas), plus `docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md`, `docs/reports/self-review-2026-09-25-doctor-shell-alias-rc-types.md`, `docs/insights/events/2026-09-25-doctor-shell-alias-rc-types.jsonl`. No unrelated files touched.

## Verdict

**PASS.**

## Follow-ups

- `/test`: run the behavioral test suite (this report only re-used the plan-authorized single `run-verify.sh` evidence pass; the tester should run its own scoped pass per its own protocol).
- `/sync-docs`: annotate `docs/tech-debt/README.md:132`'s `checkShellAliases` row — its trigger condition fired with this PR; needs an explicit note per the plan's self-review deviation record, not silent staleness. No other doc changes identified.

## Known gaps

- gopls "modernize" hints were not independently re-run outside `run-static-verify.sh`'s own golang verifier pass (which reported 0 issues); if a separate modernize/staticcheck pass is part of a future static-verify gate, re-check then.
