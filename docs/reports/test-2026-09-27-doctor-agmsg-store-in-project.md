# Test report: doctor-agmsg-store-in-project

- Date: 2026-09-27
- Plan: `docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md` (issue #170)
- Tester: `tester` subagent (Claude Code, standard flow), cycle 1 — this run
  resumed a session that was interrupted mid-`/test` by an infrastructure
  error; every result below comes from commands executed in this resumed
  session against the worktree's actual HEAD, not from any earlier partial
  or draft report.
- Branch: `fix/doctor-agmsg-store-in-project`, HEAD `f520de5` (base `main` at
  `88e7429`). `git status --porcelain` was empty at the start and remains
  empty now.
- Scope: behavioral tests only (changed-language scope: Go). No static
  analysis, linting, or drift checks — those already passed in
  `docs/reports/verify-2026-09-27-doctor-agmsg-store-in-project.md`
  (`ebb112d`); no diff-quality review is included (self-review already ran
  at `f558dd3`, addressed in the implementer's follow-up commit `a2faa60`,
  which is folded into this HEAD).
- Evidence: `docs/evidence/verify-2026-09-27-052255.log` (full `run-test.sh`
  run, executed in this session).

## Overall verdict: PASS

`./scripts/run-test.sh` is green end to end: every shell test suite reports
`FAIL: 0` (617 total shell-suite `PASS` assertions summed across all
`-- Summary --`/`── Summary ──` blocks plus the `ralph-worktree` suite's own
summary line), and all 8 Go test-bearing packages report `ok`. The check's
own tests pass under `-race`, under `TMPDIR=/tmp`, and repeated 20 times
(plain and once more with `-race`). All 5 red/green mutations discriminate
exactly as predicted by the self-review's "Coverage gaps" section and the
plan: the two gaps the self-review named (the unpinned `doctor.go` wiring,
and the missing negative "add" assertion) are both closed by tests already
present in this HEAD's follow-up commit (`a2faa60`), confirmed here by
reverting the fix and observing exactly the predicted, narrow failure in
each case. A live-binary demonstration against a built `ralph-new` (this
branch) and a `ralph-main` (built from a scratch `main` worktree) confirms
all three of the plan's Verify-plan scenarios (in-project, `.agents`-
protected, outside-project) plus every edge case named in the plan's Test
plan, with two cases byte-compared programmatically. `checkCodexAgmsgWritableRoot`
and `codexWritableRootDetail` are both at 100% line coverage under the
check's own test file. One narrow, pre-existing coverage gap unrelated to
this change was found and is not fixed here (§7, item E's exception).

## 1. Execution

| # | Command | Result |
| --- | --- | --- |
| 1a | `./scripts/run-test.sh` | PASS — every shell suite `FAIL: 0` (617 total `PASS` assertions: 14 `-- Summary --`/`── Summary ──` blocks summing to 588, plus the `ralph-worktree` suite's own "29 passed, 0 failed" line); all 8 Go packages `ok` (`internal/cli` 61.122s not cached, `internal/org` 11.929s not cached, the rest cached from an unchanged prior run). Evidence: `docs/evidence/verify-2026-09-27-052255.log` |
| 1b | `go test ./internal/cli/ -run 'CodexAgmsgWritableRoot\|Doctor' -count=1 -race -v` | PASS — every printed test/subtest is `--- PASS`, 0 `FAIL`, `ok` in 8.265s |
| 1c | `TMPDIR=/tmp go test ./internal/cli/... -count=1` | PASS — `ok` in 53.125s |

`tests/test-ralph-dispatch.sh`'s case I (SIGTERM cleanup, the handoff's named
flake watch) passed all 6 of its assertions inside the 1a run
(`docs/evidence/verify-2026-09-27-052255.log:509-514`), so per the handoff's
own condition no standalone rerun was required.

## 2. Repeat

| Command | Result |
| --- | --- |
| `go test ./internal/cli/ -run 'CodexAgmsgWritableRoot\|Doctor' -count=20` | `ok` in 112.233s, 0 failures |
| `go test ./internal/cli/ -run 'CodexAgmsgWritableRoot\|Doctor' -count=20 -race` | `ok` in 137.045s, 0 failures |

No flakiness observed across 20 iterations of the doctor/writable-root test
set, plain or race-instrumented.

## 3. Red/green mutations

All mutations were applied directly to `internal/cli/doctor_codex_writable_root.go`
or `internal/cli/doctor.go`, tested, then reverted with
`git checkout -- <file>`; `git status --porcelain` was empty after every
single revert (verified individually, not just at the end), and `git rev-parse HEAD`
stayed `f520de5` throughout.

| # | Mutation | Predicted | Actual |
| --- | --- | --- | --- |
| A | `containingProject` decision forced always-true (drop the `covers && !blocked` guard, unconditionally assign `containingProject = projectDir`) | Kills every test that expects the unconditional/AC-2/AC-3 wording to survive when the store is not genuinely, unblockedly inside the project | FAIL: `TestCheckCodexAgmsgWritableRoot_StoreUnderProjectProtectedDir_NoSupplement`, `TestCheckCodexAgmsgWritableRoot_ProjectDotAgentsSymlinked_NoSupplement`, `TestCheckCodexAgmsgWritableRoot_StoreOutsideProject_NoInProjectPhrase`. Exactly these 3 — matches prediction. |
| B | `containingProject` decision forced always-false (`if false && projectDir != ""`) | Kills every AC-1/AC-5/AC-6/edge-case test that expects the in-project wording | FAIL: `TestCheckCodexAgmsgWritableRoot_StoreInsideProject_Warn`, `TestCheckCodexAgmsgWritableRoot_ProjectDirSymlinked_SameAsResolved`, `TestCheckCodexAgmsgWritableRoot_StorageOverrideInsideProject_Warn`, `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn`. Exactly these 4 — matches prediction. |
| C | Drop the `!blocked` half of the guard (`if covers { containingProject = projectDir }`) | Kills the AC-2 protected-directory tests, which rely on `blocked` suppressing the supplement | FAIL: `TestCheckCodexAgmsgWritableRoot_StoreUnderProjectProtectedDir_NoSupplement`, `TestCheckCodexAgmsgWritableRoot_ProjectDotAgentsSymlinked_NoSupplement`. Exactly these 2, everything else green — matches prediction and closes the self-review's implicit "mutation reasoning read from the code, not executed" gap by actually running it. |
| D | `doctor.go`'s call site passes `""` instead of `projectDir` (self-review coverage gap (a)) | Kills only `TestRunDoctorOpts_CodexSandboxCheck_StoreInsideProjectThroughTheSeam`; every other test in the file (including the direct-call `checkCodexAgmsgWritableRoot` tests, which pass `projectDir` themselves and don't go through `doctor.go`) stays green | FAIL: exactly `TestRunDoctorOpts_CodexSandboxCheck_StoreInsideProjectThroughTheSeam`, 0 other failures — matches the plan's and the test's own doc comment's prediction precisely, confirming this test genuinely pins the wiring (the self-review's named coverage gap is closed). |
| E | Swap the "contains it can already write it" / "does not contain it" halves in both in-project format strings (`codexWritableRootDetail`'s two `containingProject != ""` branches) | Kills tests that assert the literal substring `"working directory contains it can already write it"` | FAIL: `TestCheckCodexAgmsgWritableRoot_StoreInsideProject_Warn`, `TestCheckCodexAgmsgWritableRoot_ProjectDirSymlinked_SameAsResolved`, `TestCheckCodexAgmsgWritableRoot_StorageOverrideInsideProject_Warn`. `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn` stays green under this mutation — it only asserts the `"the agmsg store is inside this project (...)"` substring, not the contains/does-not-contain wording, so it does not discriminate this specific swap (recorded as a coverage gap, §7 item E). |

## 4. Live binary demonstration

Built `ralph-new` from this worktree's HEAD (`go build -o <scratch>/ralph-new
./cmd/ralph`) and `ralph-main` from a scratch, detached `git worktree add
--detach <scratch> main` (`88e7429`), removed with `git worktree remove
--force` immediately after the build. All runs used
`env -u ZDOTDIR -u TMPDIR HOME=<fake home> CODEX_HOME=<fake home>/.codex`
against a scratch project's own `ralph.toml`
(`driver_pool = ["codex"]`, a `codex` `model_pool` entry,
`[org.permissions] default = "edits", codex_verified = true`), never
touching the real `HOME`, `~/.codex`, or the real agmsg store.

Fixtures initially lived under this session's scratchpad, which resolves
under `/private/tmp` — codex's own hardcoded fixed temp root
(`SlashTmpDir = "/tmp"` in production, since `/tmp` symlinks to
`/private/tmp`) — which silently turned every fixture into an implicit-root
pass and hid the warn path entirely (the same collision the self-review's
own live run hit and reported for the config-absent case). Fixed by moving
fixtures to a location under the real macOS `$TMPDIR`
(`/private/var/folders/.../T/claude-i170-livedemo`, confirmed via
`os.path.realpath` to start with neither `/tmp` nor `/private/tmp`) and
explicitly unsetting `TMPDIR` for the child process (`env -u TMPDIR`) so
neither of workspace-write's two implicit roots (the hardcoded `/tmp`, and
`$TMPDIR` itself) could cover the fixture — this let every scenario reach
the check's real warn/pass decision without needing an `exclude_slash_tmp`
config workaround, including the config-absent form the self-review's own
live run could not exercise.

| Case | Setup | `ralph-new` | `ralph-main` |
| --- | --- | --- | --- |
| (a) store inside project via `AGMSG_STORAGE_PATH`, config absent | `AGMSG_STORAGE_PATH=<proj>/custom-agmsg-store` (never created ahead of time) | warn, in-project wording ("agmsg store is inside this project", "working directory contains it can already write it", "does not contain it ... cannot send RESULT ... unless ... is added") | warn, unconditional wording ("cannot send RESULT to lead"; "; add ... to") |
| (a) same, config exists (`[sandbox_workspace_write]` present, empty) | same store, `~/.codex/config.toml` written | warn, in-project wording | warn, unconditional wording |
| (b) store under `<proj>/.agents/agmsg-home/db` (protected) | `RALPH_ORG_AGMSG_HOME=<proj>/.agents/agmsg-home` | warn, unconditional wording (no supplement) | identical | **Byte-compared programmatically: identical.** |
| (c) store outside the project entirely | `RALPH_ORG_AGMSG_HOME=<outside dir>` | warn, unconditional wording | identical | **Byte-compared programmatically: identical (AC-3 regression proof).** |
| (d) doctor run from a project subdirectory with no `ralph.toml` of its own | `cd <proj>/subdir` | pass, "not needed: ... codex_verified is false ..." — `ralph.toml` was never found (no upward search; `runDoctorFull(".")` always uses cwd literally), so config silently defaults and the check never reaches the warn path at all | n/a (behavior predates #170; see §6) |

## 5. Edge cases

| Case | Result |
| --- | --- |
| Store == project directory itself (`AGMSG_STORAGE_PATH=<proj>`) | Live and unit (`TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn`): warn, in-project wording present. |
| Non-existent store path inside the project | Live: the `custom-agmsg-store` directory used throughout case (a) never existed on disk before any run (confirmed with `test -e`); `resolveNearestExisting` walked to the nearest existing ancestor and the in-project decision was still correct. |
| Project path through a symlink (`proj_a_link` → `proj_a`) | Live: warn, in-project wording, with the project displayed via its logical (symlinked) path while the store is displayed via its real path — no shared string prefix between the two, exactly the self-review's "Observations" note; the underlying `covers`/`blocked` decision itself was still correct. Also pinned by `TestCheckCodexAgmsgWritableRoot_ProjectDirSymlinked_SameAsResolved`. |
| Relative `targetDir` | Not independently exercised as a distinct case: `runDoctorFull` is always invoked with the literal `"."` (see `internal/cli/doctor.go:33`, `newDoctorCmd`'s `RunE`), so every real invocation of `ralph doctor` already passes a relative `targetDir` by construction — case (d) above and every other live run in this report exercised this path already, via `filepath.Abs(".")`. A `filepath.Abs` failure itself (leaving `projectDir` `""`) is not reachable from a real process (`os.Getwd` failing) and is not separately fixture-able; the code's own fallback (leave `projectDir` `""`, keep unconditional wording) was confirmed by reading, matching the self-review's own note on this branch. |

## 6. Failure analysis

No test failures in the final state. The two failures observed during
mutation testing (§3) were both intentional and reverted; neither indicates
a real bug in the current HEAD.

Case (d) above is not a failure but a real, pre-existing, unrelated
behavior worth naming for the record: `ralph doctor` (via
`runDoctorFull(".")`) always treats the current working directory as "the
project" with no upward search for `ralph.toml`. Running it from a
subdirectory that has no `ralph.toml` of its own silently falls back to a
zero-value `Config` (via `fs.ErrNotExist`, which `runDoctorFull` treats as
"use defaults" rather than surfacing a warning), so `codex_verified`
defaults to `false` and the codex-sandbox check never even reaches the warn
branch this plan modifies — it reports pass/"not needed" instead. This
predates and is unrelated to the #170 change (the same would happen for
every other doctor check that reads `cfg.Org`); it is out of this plan's
scope (the plan's Non-goals explicitly exclude changing how `projectDir` is
determined), so it is recorded here as an observation rather than a defect
to fix.

## 7. Regression

- Full `run-test.sh` (§1a): 0 failures across every shell suite and all 8
  Go packages.
- `TestCheckCodexAgmsgWritableRoot_*` regression set (every pre-#170 test in
  `doctor_codex_writable_root_test.go` and the `_unix_test.go` sibling, all
  53 rewritten call sites): all pass under `-race`, `-count=20`, and
  `TMPDIR=/tmp`.
- Live case (c) (§4) is a direct AC-3 regression proof: `ralph-new` and
  `ralph-main` produce byte-identical Detail text when the store lies
  outside the project.
- Coverage gaps carried forward from self-review, now checked:
  - **Closed** — the unpinned `doctor.go` wiring: mutation D (§3) confirms
    `TestRunDoctorOpts_CodexSandboxCheck_StoreInsideProjectThroughTheSeam`
    (added in `a2faa60`) is the only test that fails when the wiring is
    reverted to `""`.
  - **Closed** — the missing negative "add" assertion: read
    `TestCheckCodexAgmsgWritableRoot_StoreInsideProject_Warn`'s
    `assertInProjectWarn` helper (`internal/cli/doctor_codex_writable_root_test.go:2051-2052`)
    — it explicitly checks
    `!strings.Contains(r.Detail, fmt.Sprintf(codexAddWritableRootUnconditionalPhrase, store))`,
    i.e. the unconditional `"; add <store> to [sandbox_workspace_write].writable_roots"`
    phrase, alongside the existing `codexWorkspaceWriteUnconditionalPhrase`
    check. Both halves of AC-1's "no unconditional wording remains" claim
    are now asserted, not just one.
- **New, narrow gap found in this cycle** (mutation E, §3 item E): none of
  the existing tests distinguish which of "contains"/"does not contain" is
  attached to which clause — only that the phrase
  `"working directory contains it can already write it"` is present
  somewhere in the Detail. `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn`
  in particular only checks for the `"the agmsg store is inside this
  project (...)"` substring and would stay green even if the two halves
  were swapped for that one test's fixture. This is a real but very narrow
  gap (an implementation that swapped the two clauses' subjects would still
  fail the *other* three in-project tests, so it cannot ship undetected) —
  recorded rather than fixed per this agent's no-source-edits mandate; see
  §9.

## 8. Coverage

`go test ./internal/cli/ -run 'CodexAgmsgWritableRoot|Doctor|RunDoctor' -coverprofile=...`
then `go tool cover -func`:

```
internal/cli/doctor_codex_writable_root.go:817:  codexWritableRootDetail        100.0%
internal/cli/doctor_codex_writable_root.go:921:  checkCodexAgmsgWritableRoot    100.0%
```

Both of the two functions this plan's Scope names are at 100% line coverage
under the check's own test file (narrowed run: 34.9% of the whole
`internal/cli` package's statements, as expected for a filtered subset).

## 9. Gaps (not fixed here; no source files edited by this agent)

- The wording-half-swap gap named in §3/§7 (mutation E): no existing test
  would fail if the "contains it can already write it" and "does not
  contain it ... cannot send RESULT" clauses were both attached to the
  wrong seat category **in only one** of the two in-project format strings
  while the other one stayed correct in a self-consistent way — the current
  tests catch a full, symmetric swap (3 of 4 tests fail) but
  `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn` does not add
  discriminating power beyond the other three. Suggested follow-up (for a
  future cycle, not this one): strengthen that one test's assertion to also
  check for `"working directory contains it can already write it"` (matching
  the other three in-project tests), or add an explicit assertion that
  "does not contain" appears strictly after "contains" in the Detail string.
- Case (d)'s subdirectory behavior (§6) is a pre-existing, out-of-scope
  observation, not a gap in this plan's own tests.
- The plan's edge case "relative targetDir" is exercised only indirectly
  (every real `ralph doctor` invocation already passes `"."`); a
  `filepath.Abs` failure itself has no test and is not practically
  fixture-able (would require a broken `os.Getwd`), matching the self-review's
  own note on this branch.

## Verdict

**PASS.** `./scripts/run-test.sh` is green, the targeted test set is stable
under `-race` and 20x repetition, all 5 red/green mutations discriminate as
predicted (including both of the self-review's named coverage gaps, now
confirmed closed), the live binary demonstration matches the unit tests and
the plan's Verify-plan scenarios exactly, and both named functions are at
100% line coverage. One narrow, low-risk coverage gap (§7/§9) and one
unrelated pre-existing behavior (§6) are recorded for the record; neither
blocks proceeding to `/sync-docs`. Working tree is clean
(`git status --porcelain` empty) and HEAD is unchanged at `f520de5`.
