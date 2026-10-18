# Test evidence: org-director-liveness

- HEAD 7e43f062 (base origin/main 88e37b1d), macOS arm64 (Darwin 25.5.0), go1.26.0 and go1.25.8 (`GOTOOLCHAIN`)
- 2026-10-19 05:26 to 06:20 JST. Load average 3 to 30
- `<scratch>` is the session scratchpad outside the worktree; `$TMPDIR` replaces the per-user temp path; `~` is `$HOME`
- No test or code file was changed. Mutants were overlay files under `<scratch>` read through `go test -overlay`; `git status --porcelain --ignored` (without `.harness/cache/`) was the same before and after every run

## 1. `./scripts/run-test.sh` (go1.26.0)

```
# Verification run
- Timestamp: 2026-10-18T20:26:33Z
- Mode: test
- Requested scope: changed
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
ok  	github.com/yoshpy-dev/ralph/internal/cli	383.998s
ok  	github.com/yoshpy-dev/ralph/internal/config	0.491s
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	502.451s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)

==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-10-18-202634.log
rc=0 dur=956s
```

Shell suites (local verifier), 41, each ending `OK`; every `FAIL:` tally line reads `FAIL: 0`; one skip (`tests/test-secret-scan-branch.sh`: the real git older than 2.41 case, git is 2.49.0):

```
tests/test-agent-models.sh tests/test-agent-phase-boundaries.sh tests/test-archive-plan.sh
tests/test-branch-name.sh tests/test-check-mojibake.sh tests/test-check-skill-sync.sh
tests/test-check-template.sh tests/test-codex-exec-invocation.sh tests/test-detect-changed-languages.sh
tests/test-detect-languages-terraform.sh tests/test-ensure-pr-ready.sh tests/test-ensure-pr-title-prefix.sh
tests/test-gc-artifacts.sh tests/test-hook-wiring.sh tests/test-insights-append.sh
tests/test-language-pack-monorepo-roots.sh tests/test-lib-json.sh tests/test-new-feature-plan.sh
tests/test-no-loop-references.sh tests/test-plan-visual.sh tests/test-post-edit-verify.sh
tests/test-pr-owner-lookup.sh tests/test-pre-bash-guard.sh tests/test-ralph-config.sh
tests/test-ralph-dispatch.sh tests/test-ralph-worktree.sh tests/test-run-verify-branch-secret-scan.sh
tests/test-run-verify-concurrency.sh tests/test-run-verify-scope.sh tests/test-secret-scan-branch.sh
tests/test-secret-scan.sh tests/test-self-review-scope.sh tests/test-skill-insight-cycle.sh
tests/test-sync-skills.sh tests/test-template-purity.sh tests/test-terraform-gitignore.sh
tests/test-terraform-pack-verify.sh tests/test-terraform-rule-frontmatter.sh
tests/test-verify-local-hook-tests.sh tests/test-verify-mode-split.sh tests/test-xreview-helpers.sh
```

`(cached)` uses the worktree's `.harness/cache/go-build`; the go1.25.8 run below re-ran all 8 packages with `-count=1`.

## 2. `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -timeout 30m -v`

Run beside the `-race` run of section 3 (started 5 minutes before it).

```
go version go1.25.8 darwin/arm64
ok  	github.com/yoshpy-dev/ralph/internal/cli	449.878s
ok  	github.com/yoshpy-dev/ralph/internal/config	2.250s
ok  	github.com/yoshpy-dev/ralph/internal/insights	0.997s
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/org	593.740s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	2.802s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.559s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.166s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	3.972s
FAIL
rc=1 dur=597s
top-level: PASS 2159, FAIL 1, SKIP 3; all "--- PASS" lines (subtests included): 5848
SKIP: TestEscalate_NoDesktopNotificationOffDarwinIsSkipped, TestBaseFS_WithMockFS, TestAvailablePacks_WithMockFS
the 67 test functions this diff adds: 67 "--- PASS" at top level
```

The one failure (known flaky test, recorded in tester memory 2026-10-18; `internal/org/regate*.go` is not in this diff):

```
--- FAIL: TestRegate_FetchFailures (6.25s)
    --- PASS: TestRegate_FetchFailures/origin_is_gone (1.81s)
    --- PASS: TestRegate_FetchFailures/origin_has_no_feature_branch (1.70s)
    --- PASS: TestRegate_FetchFailures/origin_has_no_default_branch (1.31s)
    --- FAIL: TestRegate_FetchFailures/the_fetch_runs_past_the_bound (1.43s)
    regate_test.go:771: refusal "org: regate: git fetch origin +refs/heads/main:refs/remotes/origin/main in $TMPDIR/TestRegate_FetchFailuresthe_fetch_runs_past_the_bound1049255151/001/wt-auth failed (exit 128), so nothing was recorded: check origin (git remote get-url origin) and its credentials and run ralph org regate --branch feat/auth again; git said:\n  fatal: Could not read from remote repository.\n  Please make sure you have the correct access rights\n  and the repository exists." does not say "... ran past 300ms and was stopped, so nothing was ..."
```

Re-runs:

```
GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'TestRegate_FetchFailures$' -count=10 -v
rc=0 dur=111s; 10 x "--- PASS: TestRegate_FetchFailures", 0 FAIL (load 20)

GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -count=1 -timeout 30m -v
ok  	github.com/yoshpy-dev/ralph/internal/org	593.317s
rc=0 dur=597s; top-level PASS 1067, FAIL 0, SKIP 1
(ran beside the filtered cli coverage run of section 4 for its first 160 s)
```

`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` passed in every run that ran it: run-test.sh (package ok), go1.25.8 (0.47s), the go1.25.8 org re-run (0.56s), `-race` (0.67s).

## 3. `go test -race ./internal/org/ ./internal/cli/ -run 'Director|Watch|Status' -count=1 -timeout 30m -v` (go1.26.0)

```
ok  	github.com/yoshpy-dev/ralph/internal/org	108.022s
ok  	github.com/yoshpy-dev/ralph/internal/cli	197.507s
rc=0 dur=211s; top-level PASS 550, FAIL 0, SKIP 0; all "--- PASS" lines 1584; "WARNING: DATA RACE" 0
new test functions run by the filter: 62 of 67
```

The 5 the filter does not match, with `-race`:

```
go test -race -count=1 ./internal/org/ -run 'TestMarkStoppedAtVersion' -v
--- PASS: TestMarkStoppedAtVersion_UnchangedWritesWhatMarkStoppedIfUnchangedWrites (0.01s)
--- PASS: TestMarkStoppedAtVersion_RefusesAVersionThatChanged (0.01s)
ok  	github.com/yoshpy-dev/ralph/internal/org	1.431s
go test -race -count=1 ./internal/org/driver/ -v -run 'AgentNotFound|AgentGet|AgentStart'
--- PASS: TestIsAgentNotFound (0.00s)
--- PASS: TestHerdr_AgentGet_AgentNotFoundIsReadable (0.00s)
--- PASS: TestHerdr_AgentStartAndWait_KeepTheEnvelopeAsText (0.00s)
(and 5 older AgentGet/AgentStart tests)
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	4.253s
rc=0 both; "WARNING: DATA RACE" 0
```

## 4. Coverage (filtered runs, go1.26.0)

`go test ./internal/org/ -count=1 -run 'Director|RunWatchAll|Wake|MarkStopped' -coverprofile` (rc 0, 71 s, 36.4% of the package):

```
director_check.go          directorVersionOf 100.0  readDirectorVersion 100.0  CheckDirector 100.0
                           checkDirectorOnce 100.0  checkDirectorStarter 100.0  checkDirectorAgent 100.0
                           checkDirectorProcess 100.0  directorAliveOf 100.0  Check 100.0
                           DirectorAlivePathIn 100.0  writeDirectorAlive 100.0  ReadDirectorAlive 100.0
                           DirectorAliveFor 100.0  CheckDirectorStarter 0.0 (called from internal/cli only)
director_start_cleanup.go  checkCurrentDirector 94.1  cleanUpEndedDirector 93.8
                           directorHerdrCleanupNotice 75.0  directorCmdCleanupNotice 100.0
director_start.go          AttachHeadless 95.7  MarkCmdLaunched 97.6  directorCheckInStarterEnded 100.0
                           MarkStoppedAtVersion 100.0  markStopped 97.1  directorVersionText 100.0
                           directorStarter 100.0  checkDirectorStart 89.5
director_wake.go           WakeDirector 93.3  directorCheckFoundEnded 100.0
watch.go                   cycle 88.9  checkDirector 100.0  wake 100.0
```

Blocks not run, in new code or code this diff moved:

```
director_start_cleanup.go:38   checkCurrentDirector: CheckDirector returns an error (director.jsonl unreadable at start's check)
director_start_cleanup.go:96   cleanUpEndedDirector: a launch cmd whose end stopDirectorProcess does not confirm
director_start_cleanup.go:125  directorHerdrCleanupNotice: the unrecorded-workspace sentence (run by the cli test below)
director_start.go:452          MarkCmdLaunched: the read under the lock fails
director_start.go:179,187,614  AttachHeadless and markStopped: the read and nextID under the lock fail (older paths)
director_wake.go:363           WakeDirector: mem == nil (older path)
watch.go:674                   cycle: sweep.Err (older path)
```

`go test ./internal/cli/ -count=1 -run 'Director|Status|OrgWatch' -coverprofile -coverpkg=./internal/cli/,./internal/org/` (rc 0, 160 s):

```
cli/org_director_check.go     attachedError 100.0  attachHeadlessChecks 100.0  orgDirectorCheckText 100.0
                              readOrgDirectorCheck 100.0  orgDirectorWatchCheckClause 100.0
cli/org_director.go           readOrgDirectorReport 100.0  orgDirectorLine 100.0
cli/org_director_headless.go  startDirector 90.5  startError 100.0  startAttachedError 100.0  directorLaunched 100.0
org/director_check.go         CheckDirectorStarter 100.0
org/director_start_cleanup.go directorHerdrCleanupNotice 100.0
```

## 5. Mutation (`go test -overlay`, 31 mutants, all red)

org: `go test -overlay=<scratch>/mut/<name>/overlay.json -count=1 -timeout 15m -v -run 'Director|RunWatchAll|Wake|MarkStopped' ./internal/org/` (52 to 65 s each, two at a time).
cli: the same with `-run 'Director|Status|OrgWatch' ./internal/cli/` (78 to 93 s each, two at a time).
Every mutant compiled (`-run '^$'` first). The failing assertion of each single-test kill was read: each is the asserted behavior, none a timeout.

| Mutant | Edit | Red by |
| --- | --- | --- |
| M01 check-no-reread | CheckDirector returns after the first try, with no read under the lock | TestCheckDirector_UnreadableDirectorFileIsAnError, _LaunchedAfterTheReadIsNotReadAsStartEnded, _ChangedOnBothTriesIsUnknown, _ReplacedAfterTheReadIsNothingToCheck; TestStartDirector_AStartThatFinishedDuringTheCheckIsNotCleanedUp; TestRunWatchAll_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn, _ALaunchedThatLandsDuringTheCheckIsNotMarkedStartEnded |
| M02 check-reread-unlocked | the second read is readRaw without director.lock | TestRunWatchAll_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn (an unopenable director.lock no longer fails the check) |
| M03 cleanup-rereads-registration | cleanUpEndedDirector replaces chk.Registration with store.Read().Current before closing | TestStartDirector_TheCleanupActsOnTheRegistrationTheCheckRead |
| M04 watch-check-after-wake | cycle: `w.wake(nil)` then `w.checkDirector()` | TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse, _TheDirectorCheckWritesOnlyItsMark, _ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn |
| M05 watch-check-before-deadlines | the check runs before SweepInboxDeadlines | TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse |
| M06 watch-check-after-pulse | pulse, then check, then wake | TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse (8b-1), TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse |
| M07 checkin-accepts-ended-starter | `if false && ended != "" ...` in MarkCmdLaunched | TestDirectorMarkCmdLaunched_ReadsTheStarterWithPSWhenNoReaderIsGiven, _RefusesACheckInOnceTheStartThatRecordedItHasEnded, _ReadsTheStarterWithTheDirectorLockFree, _UsesTheStarterItReadOnlyWhileTheRegistrationStillRecordsIt; TestStartDirector_ACheckInRefusedForItsStarterRunsNoAgent |
| M08 checkin-drops-same-starter-guard | under the lock, refuse on `ended != ""` alone | TestDirectorMarkCmdLaunched_UsesTheStarterItReadOnlyWhileTheRegistrationStillRecordsIt |
| M09 checkin-accepts-reused-starter-pid | directorCheckInStarterEnded returns "" for any alive read | TestDirectorMarkCmdLaunched_RefusesACheckInOnceTheStartThatRecordedItHasEnded, TestStartDirector_ACheckInRefusedForItsStarterRunsNoAgent |
| M10 checkin-accepts-zombie-starter | returns "" for a zombie | TestDirectorMarkCmdLaunched_ReadsTheStarterWithPSWhenNoReaderIsGiven, _RefusesACheckInOnceTheStartThatRecordedItHasEnded |
| M11 checkin-reads-starter-under-lock | the ps read moves inside withLock | TestDirectorMarkCmdLaunched_RefusesACheckInOnceTheStartThatRecordedItHasEnded, _ReadsTheStarterWithTheDirectorLockFree, _UsesTheStarterItReadOnlyWhileTheRegistrationStillRecordsIt |
| M12 mark-match-id-only | DirectorAliveFor compares the id only | TestDirectorAliveFor |
| M13 wake-check-match-id-only | directorCheckFoundEnded compares the id only | TestWakeDirector_AnEndedCheckOfAnotherRegistrationWakesAsBefore |
| M14 starter-reused-is-ended | starter_pid_reused returns ended | TestCheckDirector_StartingRows |
| M15 pane-unconfirmed-is-ended | pane_unconfirmed returns ended | TestCheckDirector_HerdrRunningRows, TestStartDirector_RefusesADirectorItsCheckFindsAliveOrCannotTell, TestWakeDirector_AnUnknownCheckWakesAsBefore |
| M16 cmd-pid-reused-is-ended | pid_reused returns ended | TestCheckDirector_CmdRunningRows, TestStartDirector_RefusesADirectorItsCheckFindsAliveOrCannotTell, TestWakeDirector_AnUnknownCheckWakesAsBefore |
| M17 wake-ignores-ended-check | `if false && directorCheckFoundEnded(...)` | TestWakeDirector_ARegistrationTheCheckFoundEndedIsNotWoken, TestRunWatchAll_TheDirectorCheckWritesOnlyItsMark, _ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn |
| M18 record-ignores-version | markStopped skips the version comparison | TestStartDirector_ARegistrationThatChangesBeforeTheRecordIsNotRecordedStopped, _TheCleanupActsOnTheRegistrationTheCheckRead, TestMarkStoppedAtVersion_RefusesAVersionThatChanged |
| M19 attached-drops-starter | AttachHeadless writes no starter fields | 8 tests (TestDirectorAttachHeadless_RecordsTheStarterOnTheAttachedOfEitherLaunch, TestStartDirector_RecordsItsOwnProcessAsTheStarter, _LaunchCmdRecordsItsOwnProcessAsTheStarter, _ACheckInRefusedForItsStarterRunsNoAgent, and 4 TestDirectorMarkCmdLaunched_*) |
| M20 starter-is-parent-pid | directorStarter uses os.Getppid() | TestStartDirector_RecordsItsOwnProcessAsTheStarter, _LaunchCmdRecordsItsOwnProcessAsTheStarter, _ACheckInRefusedForItsStarterRunsNoAgent, _CleansUpAnEndedLaunchCmdDirectorAndStarts, _CleansUpARegistrationWhoseStartDied |
| M21 watch-marks-nothing-checked | checkDirector writes the mark when nothing was checked | TestRunWatchAll_WritesNoDirectorMarkWithoutACurrentHeadlessRegistration |
| M22 start-cleans-unknown | start cleans up anything not alive | TestStartDirector_RefusesADirectorItsCheckFindsAliveOrCannotTell, _LaunchCmdKeepsTheOtherRefusals, _RefusesBeforeWritingOrCallingHerdr, TestDirector_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration |
| M23 version-ignores-event-count | directorVersionOf never counts events | TestDirectorVersionOf, TestStartDirector_TheCleanupActsOnTheRegistrationTheCheckRead, TestMarkStoppedAtVersion_RefusesAVersionThatChanged |
| M24 one-check-try | directorCheckAttempts = 1 | TestCheckDirector_LaunchedAfterTheReadIsNotReadAsStartEnded, _ChangedOnBothTriesIsUnknown, _ReplacedAfterTheReadIsNothingToCheck; TestStartDirector_AStartThatFinishedDuringTheCheckIsNotCleanedUp; TestRunWatchAll_ALaunchedThatLandsDuringTheCheckIsNotMarkedStartEnded |
| M25 check-a-stopped-registration | CheckDirector checks a current stopped registration | TestCheckDirector_NothingToCheck, TestStartDirector_RefusesWithNoCheckWhenThereIsNothingToCheck, TestRunWatchAll_WritesNoDirectorMarkWithoutACurrentHeadlessRegistration |
| M26 version-ignores-folded | directorVersionOf leaves folded empty | TestDirectorVersionOf (asserts the field) |
| C01 display-mark-match-id-only | DirectorAliveFor compares the id only (cli run) | TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector, TestOrgDirectorAttach_RefusalAddsTheWatchsMark |
| C02 disabled-line-names-start-cleanup | the enabled=false line passes cleanup=true | TestDirectorLine_EdgeCasesOfTheCheck |
| C03 status-reads-mark-of-stopped | readOrgDirectorCheck reads the mark of a stopped registration | TestDirectorLine_EdgeCasesOfTheCheck |
| C04 attach-refusal-ignores-mark-match | attachHeadlessChecks uses any mark | TestOrgDirectorAttach_RefusalAddsTheWatchsMark |
| C05 start-cleanup-unknown-via-cli | M22 through the cli | TestOrgDirectorStart_RefusesADirectorItsCheckFindsAliveOrCannotTell, _RefusedWithNothingWritten, _AnAttachedDirectorIsRefusedBeforeTheWatchGate |

## 6. No real herdr, osascript, terminal, Orca, tmux, gh, or network (AC9)

`herdr`, `gh`, `tmux`, `orca` (`/opt/homebrew/bin`), `osascript` and `open` (`/usr/bin`) are installed on this machine. Canary scripts with those names (plus `agmsg`, `codex`, `claude`, `curl`), each logging its call and exiting 1, were put first on the inherited PATH:

```
PATH="<scratch>/canary:$PATH" go test -count=1 -timeout 30m -v -run '^(<the 67 new test functions>)$' ./internal/org/ ./internal/org/driver/ ./internal/cli/
ok  	github.com/yoshpy-dev/ralph/internal/org	6.230s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	0.484s
ok  	github.com/yoshpy-dev/ralph/internal/cli	9.951s
rc=0; top-level PASS 67, FAIL 0; canary calls: 0
```

Sensitivity check: an overlay of `internal/cli/org_test.go` whose `setupOrgStubPATH` no longer puts the stub dir on PATH, same command for internal/cli: rc 1, PASS 12, FAIL 4, canary calls 5 (all `herdr workspace create`). So a call that leaves the stubs reaches the canary, and at HEAD none does.

No `.harness/state/org/` and no `director-alive.json` appeared in the worktree, and nothing under `~/MyDev/github.com/yoshpy-dev/ralph/.git/ralph/` or the main checkout's `.harness/state/org/` changed during the runs (`find -newermt` from the run's start).
