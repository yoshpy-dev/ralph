# Test evidence: org-director-attach (cycle 1)

- Date: 2026-10-15 (JST 04:08-04:40; the UTC timestamps in the output read 2026-10-14T19:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach, HEAD a5f5353b (base ef575748)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0. Load average was 222 when the session began (another session's iOS simulator) and 5-15 during the runs below
- Redaction: the session scratch directory is `$SCRATCH`, the home directory `~`, and the login name `$USER`. The wrapper's full log `docs/evidence/verify-2026-10-14-190833.log` is gitignored and not committed
- Trailing spaces at the ends of output lines were stripped (`git diff --check`); nothing else in the raw output was changed apart from the redaction
- Nothing in the worktree was edited for the tests. Mutants and probe tests lived in `$SCRATCH` and reached the build only through `go test -overlay`

## 1. ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; scripts/ralph-config.sh widened it to full)

Command: `./scripts/run-test.sh > run-test.out 2>&1` (rc 0, 676.6 s wall). Filtered from the 4,517-line raw output with:
`sed -n 1,6p; grep -nE '^==> |tests: [0-9]+ passed|Results?: [0-9]+/[0-9]+|^\s*PASS: [0-9]+( / [0-9]+)?$|^\s*FAIL: [0-9]+$|^\s*SKIP|passed, [0-9]+ failed|^    OK$|^    FAIL|^ok |^\? |^FAIL|^rc=|^real |Evidence saved|All verifiers'`
(the `==> sh -n`, `awk parse`, `jq -e`, `Codex`, and `scripts/check` step lines dropped). Numbers before ':' are line numbers in the raw output. 41 suites, 41 "OK", every "FAIL:" line is 0. The one real SKIP is tests/test-secret-scan-branch.sh's (git 2.49.0). With line numbers, timing, and the Go lines removed, the suite lines equal those of `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-run-test.txt` except the scope line (`full fallback (unclassified:scripts/ralph-config.sh)` here, `changed` there).

```text
# Verification run
- Timestamp: 2026-10-14T19:08:33Z
- Mode: test
- Requested scope: changed

==> Running local verifier
6:==> Running local verifier
7:==> tests/test-agent-models.sh
60:  PASS: 47 / 47
61:  FAIL: 0
62:    OK
63:==> tests/test-agent-phase-boundaries.sh
110:  PASS: 44 / 44
111:  FAIL: 0
112:    OK
113:==> tests/test-archive-plan.sh
114:==> archive-plan.sh rewrites tech-debt README links before moving the plan
133:==> verify.local.sh checks the plan links in docs/tech-debt/README.md
140:archive-plan tests: 22 passed, 0 failed, 22 total
141:    OK
142:==> tests/test-branch-name.sh
143:==> branch-name.sh generation
150:==> branch-name.sh validation
168:==> plan creation type metadata
173:branch-name tests: 26 passed, 0 failed, 26 total
174:    OK
175:==> tests/test-check-mojibake.sh
194:  PASS: 15
195:  FAIL: 0
196:    OK
197:==> tests/test-check-skill-sync.sh
212:  PASS: 13
213:  FAIL: 0
214:    OK
215:==> tests/test-check-template.sh
277:test-check-template: 60 passed, 0 failed, 0 skipped
278:    OK
279:==> tests/test-codex-exec-invocation.sh
446:Results: 144/144 passed
448:    OK
449:==> tests/test-detect-changed-languages.sh
535:  PASS: 83 / 83
536:  FAIL: 0
537:    OK
538:==> tests/test-detect-languages-terraform.sh
549:  PASS: 8 / 8
550:  FAIL: 0
551:    OK
552:==> tests/test-ensure-pr-ready.sh
553:==> ensure-pr-ready.sh
562:ensure-pr-ready tests: 7 passed, 0 failed, 7 total
563:    OK
564:==> tests/test-ensure-pr-title-prefix.sh
565:==> ensure-pr-title-prefix.sh
580:ensure-pr-title-prefix tests: 13 passed, 0 failed, 13 total
581:    OK
582:==> tests/test-gc-artifacts.sh
583:==> gc-artifacts: dry-run mode
592:==> gc-artifacts: --apply with date-based reports
598:==> gc-artifacts: README.md is always preserved
604:==> gc-artifacts: evidence keep-newest-20
613:==> gc-artifacts: dry-run output contains candidates
616:==> gc-artifacts: exit 0 when nothing to do
618:==> gc-artifacts: --help
621:gc-artifacts tests: 11 passed, 0 failed, 11 total
622:    OK
623:==> tests/test-hook-wiring.sh
695:  PASS: 68
696:  FAIL: 0
697:    OK
698:==> tests/test-insights-append.sh
700:==> Case 1: valid append — single line, all required fields
717:==> Case 2: second append to same slug → 2 lines in file
722:==> Case 3: missing required flag → exit 1, no file
727:==> Case 4: invalid enum values → exit 1
734:==> Case 5: counts land in findings and triage objects
743:==> Case 6: --events-dir override
748:==> Case 7: optional fields absent when not provided
753:==> Case 8: --cycle auto reads the pipeline cycle from the state dir
767:insights-append tests: 51 passed, 0 failed
768:    OK
769:==> tests/test-language-pack-monorepo-roots.sh
801:  PASS: 29 / 29
802:  FAIL: 0
803:    OK
804:==> tests/test-lib-json.sh
934:  PASS: 126
935:  FAIL: 0
936:  SKIP: 0
937:    OK
938:==> tests/test-new-feature-plan.sh
939:==> plan templates stay identical
941:==> new-feature-plan.sh output
958:==> plan-visual.sh digest on the created plan
964:new-feature-plan tests: 21 passed, 0 failed, 21 total
965:    OK
966:==> tests/test-no-loop-references.sh
968:    OK
969:==> tests/test-plan-visual.sh
970:==> usage
984:==> open
1008:==> shot
1054:==> digest
1078:plan-visual tests: 103 passed, 0 failed, 103 total
1079:    OK
1080:==> tests/test-post-edit-verify.sh
1108:  PASS: 24
1109:  FAIL: 0
1110:    OK
1111:==> tests/test-pr-owner-lookup.sh
1112:==> expressions found in .claude/skills/pr/SKILL.md
1118:==> the four faces carry the same expressions
1122:==> owner from origin URL
1130:==> lookup into the intended base (--base <base>)
1134:==> lookup into any base (no --base)
1138:pr-owner-lookup tests: 20 passed, 0 failed, 20 total
1139:    OK
1140:==> tests/test-pre-bash-guard.sh
3386:  PASS: 2227
3387:  FAIL: 0
3388:  SKIP: 0
3389:    OK
3390:==> tests/test-ralph-config.sh
3421:Results: 19/19 passed
3423:    OK
3424:==> tests/test-ralph-dispatch.sh
3461:  PASS: 33
3462:  FAIL: 0
3463:    OK
3464:==> tests/test-ralph-worktree.sh
3465:==> ralph-worktree.sh state paths
3470:==> ralph-worktree.sh ensure/resume
3480:==> ralph-worktree.sh collision and dirty-base checks
3483:==> ralph-worktree.sh cleanup
3488:==> ralph-worktree.sh gc
3501:==> ralph-worktree.sh codex config external-rewrite detection
3617:ralph-worktree tests: 143 passed, 0 failed, 143 total
3618:    OK
3619:==> tests/test-run-verify-branch-secret-scan.sh
3654:  PASS: 32
3655:  FAIL: 0
3656:    OK
3657:==> tests/test-run-verify-concurrency.sh
3702:  PASS: 37 / 37
3703:  FAIL: 0
3704:    OK
3705:==> tests/test-run-verify-scope.sh
3727:  PASS: 19 / 19
3728:  FAIL: 0
3729:    OK
3730:==> tests/test-secret-scan-branch.sh
3924:  SKIP: the real git older than 2.41 case (git 2.49.0 is 2.41 or later; the stubbed old git covers it)
3980:  PASS: 246
3981:  FAIL: 0
3982:    OK
3983:==> tests/test-secret-scan.sh
4109:  PASS: 123
4110:  FAIL: 0
4111:    OK
4112:==> tests/test-self-review-scope.sh
4179:  PASS: 64 / 64
4180:  FAIL: 0
4181:    OK
4182:==> tests/test-skill-insight-cycle.sh
4183:==> insight-event command passes --cycle auto
4205:skill-insight-cycle tests: 20 passed, 0 failed, 20 total
4206:    OK
4207:==> tests/test-sync-skills.sh
4231:  PASS: 22
4232:  FAIL: 0
4233:    OK
4234:==> tests/test-template-purity.sh
4246:  PASS: 10
4247:  FAIL: 0
4248:    OK
4249:==> tests/test-terraform-gitignore.sh
4305:  PASS: 47
4306:  FAIL: 0
4308:    OK
4309:==> tests/test-terraform-pack-verify.sh
4348:  PASS: 36 / 36
4349:  FAIL: 0
4350:    OK
4351:==> tests/test-terraform-rule-frontmatter.sh
4365:  PASS: 11 / 11
4366:  FAIL: 0
4367:    OK
4368:==> tests/test-verify-local-hook-tests.sh
4369:==> run_hook_tests treats a test that would not run in CI as FAIL
4376:verify-local-hook-tests tests: 5 passed, 0 failed, 5 total
4377:    OK
4378:==> tests/test-verify-mode-split.sh
4440:  PASS: 59 / 59
4441:  FAIL: 0
4442:    OK
4443:==> tests/test-xreview-helpers.sh
4493:  PASS: 29
4494:  FAIL: 0
4495:    OK
4496:==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)
4497:==> Language packs selected: golang
4498:==> Running golang verifier
4499:==> Go project root: .
4500:?   	github.com/yoshpy-dev/ralph	[no test files]
4501:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4502:ok  	github.com/yoshpy-dev/ralph/internal/cli	195.357s
4503:ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
4504:ok  	github.com/yoshpy-dev/ralph/internal/insights	1.607s
4505:ok  	github.com/yoshpy-dev/ralph/internal/org	87.095s
4506:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
4507:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4508:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4509:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4511:==> All verifiers passed.
4513:Evidence saved to: docs/evidence/verify-2026-10-14-190833.log
4514:real 676.56
4517:rc=0
```

## 2. Go tests

### 2.1 Per package (`go test -count=1 -v ./internal/...`)

go1.25.8 ran with `-coverprofile`; go1.26.0 without. Both had `$SCRATCH/canary` first on PATH (2.4). "top" counts top-level `--- PASS/FAIL/SKIP` lines, "sub" every indented one.

go1.25.8 (04:13:01-04:16:36, rc 0):
```text
ok  	github.com/yoshpy-dev/ralph/internal/cli	209.543s	coverage: 87.3% of statements
ok  	github.com/yoshpy-dev/ralph/internal/config	1.214s	coverage: 93.3% of statements
ok  	github.com/yoshpy-dev/ralph/internal/insights	2.159s	coverage: 86.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org	96.954s	coverage: 94.5% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	3.892s	coverage: 93.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	4.305s	coverage: 100.0% of statements
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.857s	coverage: 75.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	3.453s	coverage: 91.2% of statements

internal/cli           top PASS 678 FAIL 0 SKIP 0 | sub PASS 599 FAIL 0 SKIP 0 | 209.543s
internal/config        top PASS 37 FAIL 0 SKIP 0 | sub PASS 34 FAIL 0 SKIP 0 | 1.214s
internal/insights      top PASS 36 FAIL 0 SKIP 0 | sub PASS 7 FAIL 0 SKIP 0 | 2.159s
internal/org           top PASS 746 FAIL 0 SKIP 1 | sub PASS 1523 FAIL 0 SKIP 0 | 96.954s
internal/org/driver    top PASS 48 FAIL 0 SKIP 0 | sub PASS 89 FAIL 0 SKIP 0 | 3.892s
internal/org/protocol  top PASS 15 FAIL 0 SKIP 0 | sub PASS 56 FAIL 0 SKIP 0 | 4.305s
internal/scaffold      top PASS 30 FAIL 0 SKIP 2 | sub PASS 0 FAIL 0 SKIP 0 | 3.857s
internal/upgrade       top PASS 114 FAIL 0 SKIP 0 | sub PASS 13 FAIL 0 SKIP 0 | 3.453s
```

go1.26.0 (04:20:13-04:23:44, rc 0):
```text
ok  	github.com/yoshpy-dev/ralph/internal/cli	210.794s
ok  	github.com/yoshpy-dev/ralph/internal/config	1.026s
ok  	github.com/yoshpy-dev/ralph/internal/insights	1.762s
ok  	github.com/yoshpy-dev/ralph/internal/org	84.587s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	2.358s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	3.401s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.130s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	4.384s

internal/cli           top PASS 678 FAIL 0 SKIP 0 | sub PASS 599 FAIL 0 SKIP 0 | 210.794s
internal/config        top PASS 37 FAIL 0 SKIP 0 | sub PASS 34 FAIL 0 SKIP 0 | 1.026s
internal/insights      top PASS 36 FAIL 0 SKIP 0 | sub PASS 7 FAIL 0 SKIP 0 | 1.762s
internal/org           top PASS 746 FAIL 0 SKIP 1 | sub PASS 1523 FAIL 0 SKIP 0 | 84.587s
internal/org/driver    top PASS 48 FAIL 0 SKIP 0 | sub PASS 89 FAIL 0 SKIP 0 | 2.358s
internal/org/protocol  top PASS 15 FAIL 0 SKIP 0 | sub PASS 56 FAIL 0 SKIP 0 | 3.401s
internal/scaffold      top PASS 30 FAIL 0 SKIP 2 | sub PASS 0 FAIL 0 SKIP 0 | 3.130s
internal/upgrade       top PASS 114 FAIL 0 SKIP 0 | sub PASS 13 FAIL 0 SKIP 0 | 4.384s
```

- The 4,028 result lines (name and verdict, durations removed, sorted) are identical across the two versions: `diff names125.txt names126.txt` is empty
- SKIP lines (both versions): `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` (org; skipped on darwin by design), `TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS` (scaffold). None is in this diff
- Tests that were flaky before (tester memory), both versions:

```text
go125:
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.99s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.49s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.54s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.86s)
go126:
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.77s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.66s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (7.53s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.47s)
```

### 2.2 Tests this diff added and removed

`git diff ef575748...HEAD -- '*_test.go'`: 70 `+func Test`, 7 `-func Test`. All 70 added tests are `--- PASS` in both runs (99 direct subtests). The 7 removed functions are 7a tests whose names or assertions said approve resolves the item (plan D7), and the old name of the status-line order test; their replacements are among the 70 (`TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve`, `TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell`, `TestOrgApprove_RecordsAndLeavesTheItemAndThePlan`, `TestOrgApprove_ASecondApproveOfTheItemIsRefused`, `TestOrgApprove_RefusesAnItemAlreadyApproved`, `TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines`).

Removed:
```text
TestOrgApprove_RecordsResolvesAndPrintsTheDecision
TestOrgApprove_TheDecisionCommandRunsInAPOSIXShell
TestOrgApprove_AResolveThatFailsKeepsTheRecordAndPrintsIt
TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchLine
TestOrgApprove_RecordsResolvesAndLeavesThePlan
TestOrgApprove_RefusesAnItemItCannotResolve
TestOrgApprove_ResolveFailureKeepsTheRecord
```

Added (all PASS on go1.25.8 and go1.26.0):
```text
PASS/PASS TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve
PASS/PASS TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell
PASS/PASS TestOrgApprove_PassesWithAnInboxItCannotWrite
PASS/PASS TestOrgApprove_ASecondApproveOfTheItemIsRefused
PASS/PASS TestPrintApprovedPlans_NoResolveCommandForAnItemResolvedElsewhere
PASS/PASS TestOrgDirector_AttachReplaceAndDetachByID
PASS/PASS TestOrgDirector_DetachNeedsExactlyOneOfIDAndForce
PASS/PASS TestOrgDirector_DetachForce
PASS/PASS TestOrgDirector_AttachRefusesABadDriverBeforeReadingAnything
PASS/PASS TestOrgDirector_RefusesOrgID
PASS/PASS TestOrgDirector_UnknownVerbIsAnError
PASS/PASS TestOrgDirector_EnabledFromTheMainWorktreeRalphToml
PASS/PASS TestOrgDirector_MainWorktreeRalphTomlLoadError
PASS/PASS TestOrgDirector_DetachReadsNoConfig
PASS/PASS TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork
PASS/PASS TestOrgDirector_LegacyLedgerGuard
PASS/PASS TestOrgDirector_StatusWarnsAboutDamageAndEscapesValues
PASS/PASS TestOrgDirector_StatusOfAnUnreadableFile
PASS/PASS TestOrgDirector_StatusDisabledNamesALeftRegistration
PASS/PASS TestOrgDirectorHelp_DescribesTheRegistration
PASS/PASS TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly
PASS/PASS TestOrgEscalate_ProbesTheLedgersWatch
PASS/PASS TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml
PASS/PASS TestOrgEscalate_MainWorktreeRalphTomlLoadError_AlertsTheHumanAndExitsOne
PASS/PASS TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch
PASS/PASS TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines
PASS/PASS TestStatusCmd_DirectorLinePerState
PASS/PASS TestStatusCmd_DirectorEnabledFromTheMainWorktree
PASS/PASS TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad
PASS/PASS TestStatusCmd_DirectorDamageWarnsOnStderr
PASS/PASS TestStatusCmd_HelpDescribesTheDirectorLine
PASS/PASS TestOrgApprove_RecordsAndLeavesTheItemAndThePlan
PASS/PASS TestOrgApprove_RefusesAnItemWhoseResolveNoteIsTooLong
PASS/PASS TestOrgApprove_TakesNoInboxLock
PASS/PASS TestOrgApprove_RefusesAnItemAlreadyApproved
PASS/PASS TestOrgApprove_AnApproveThatStopsBeforeTheSendReachesAHumanAtADeadline
PASS/PASS TestOrgApprove_AnInboxItCannotReadAgainKeepsTheRecords
PASS/PASS TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline
PASS/PASS TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline
PASS/PASS TestDirectorStore_LivesUnderTheGivenStateDir
PASS/PASS TestDirectorRead_MissingStateDirIsNoRegistration
PASS/PASS TestDirectorAttach_RecordsOneAttachedWithAnID
PASS/PASS TestDirectorRegistration_JSONShape
PASS/PASS TestDirectorAttach_RefusesASecondAttachWithoutReplace
PASS/PASS TestDirectorAttach_EscapesAHandEditedAttachTimeInItsError
PASS/PASS TestDirectorAttach_ReplaceWritesTheDetachedAndANewAttached
PASS/PASS TestDirectorAttach_AReplaceThatCannotWriteRecordsNothing
PASS/PASS TestDirectorAttach_ReplaceWithNoRegistrationOnlyAttaches
PASS/PASS TestDirectorDetach_WithTheCurrentIDEndsTheRegistration
PASS/PASS TestDirectorDetach_WithNoRegistrationIsRefused
PASS/PASS TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor
PASS/PASS TestDirectorDetach_ForceEndsTheCurrentRegistrationWhateverItsID
PASS/PASS TestDirectorDetach_NeedsExactlyOneOfAnIDAndForce
PASS/PASS TestDirectorAttach_RefusesAnUnknownDriverBeforeReadingAnything
PASS/PASS TestDirectorAttach_NeverReusesTheIDOfADamagedLine
PASS/PASS TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable
PASS/PASS TestDirectorRead_FoldsTheRegistrationFromItsAttachedEvent
PASS/PASS TestDirectorRead_SkipsAndCountsLinesItCannotUse
PASS/PASS TestDirectorAttach_FailsWhenNoIDIsLeft
PASS/PASS TestDirectorAttach_ConcurrentAttachesWithoutReplace_OnlyOneSucceeds
PASS/PASS TestDirectorAttach_ConcurrentReplacesEachEndTheOneBefore
PASS/PASS TestDirectorLock_IsItsOwnLockAndSerializesWrites
PASS/PASS TestDirectorStore_ReportsAnUnreadableFileAndAnUnavailableLock
PASS/PASS TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch
PASS/PASS TestEscalate_DeferredBanner
PASS/PASS TestEscalate_RefusalProbesNoWatch
PASS/PASS TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason
PASS/PASS TestNotifyInboxItem_SendsADeferredItemNow
PASS/PASS TestInboxEscalate_WritesAndFoldsTheDelivery
PASS/PASS TestInboxEscalate_RefusesAnUnknownDelivery
```

### 2.3 Race detector (go1.26.0)

```text
$ go test -race -count=1 -run 'Director|Escalate|Inbox|Approve|Deadline|NotifyInboxItem' ./internal/org/
ok  	github.com/yoshpy-dev/ralph/internal/org	7.114s
rc=0 12s
$ go test -race -count=1 -run 'OrgDirector|OrgEscalate|OrgInbox|OrgApprove|StatusCmd|PrintApprovedPlans' ./internal/cli/
ok  	github.com/yoshpy-dev/ralph/internal/cli	17.709s
rc=0 23s
DATA RACE lines: 0
```

### 2.4 AC9: no real osascript or herdr, no director.jsonl in the repo's ledger

Every Go run in 2.1 and 2.3 and the 44 runs of mutate.py (4.1) had `$SCRATCH/canary` first on PATH; the follow-up runs of 4.2 did not. The canary `osascript` logs its call and exits 0; the canary `herdr` logs its call, passes `--version` to the real `/opt/homebrew/bin/herdr`, and exits 97 otherwise. Sensitivity: run by hand through the same PATH, `osascript -e 'sensitivity check'`, `herdr --version`, and `herdr pane list` each wrote a line (and `herdr pane list` exited 97). After the Go runs the log had 0 lines.

```text
canary log lines after go1.25.8: 0 (checked before it was emptied for the go1.26.0 run)
canary log lines after go1.26.0, -race, and the 44 runs of mutate.py (section 4.1): 0
$ ls ~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach/.harness/state/org
both: No such file or directory
$ find ~/MyDev/github.com/yoshpy-dev/ralph -name director.jsonl
(no output)
$ git status --porcelain   (worktree and main checkout)
(no output, both)
```

`ralph org director attach` and `escalate` create the ledger directory (withFileLock makes it), so a test that wrote to the repo's ledger would have left `.harness/state/org` behind; its absence is not vacuous. No `RALPH_ORG_STATE_DIR` or `HERDR_*` variable was set in the environment the tests ran in.

### 2.5 Coverage (go1.25.8, each package's own tests)

Package totals: cli 87.3%, config 93.3%, insights 86.1%, org 94.5%, org/driver 93.1%, org/protocol 100.0%, scaffold 75.7%, upgrade 91.2% (the profile's total 90.8%).

Statements per changed file:
```text
internal/org/director.go: 129/135 statements (95.6%)
internal/cli/org_director.go: 135/138 statements (97.8%)
internal/org/escalate.go: 117/127 statements (92.1%)
internal/org/inbox.go: 170/176 statements (96.6%)
internal/org/deadline.go: 62/62 statements (100.0%)
internal/org/approve.go: 159/169 statements (94.1%)
internal/cli/org_inbox.go: 170/173 statements (98.3%)
internal/cli/org_approve.go: 20/20 statements (100.0%)
internal/cli/status.go: 230/239 statements (96.2%)
internal/cli/org.go: 517/541 statements (95.6%)
internal/config/config.go: 70/75 statements (93.3%)
```

Functions of the two new files:
```text
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:24:			newOrgDirectorCmd			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:63:			newOrgDirectorAttachCmd			96.6%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:147:			newOrgDirectorDetachCmd			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:201:			newOrgDirectorStatusCmd			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:257:			newOrgDirectorRuntime			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:286:			command					100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:296:			attachedError				100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:309:			orgDirectorEnabledFile			85.7%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:332:			orgDirectorCaller			75.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:342:			orgDirectorWatchGap			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:383:			readOrgDirectorReport			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:405:			orgDirectorLine				100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:436:			warnDirectorDamage			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:450:			printOrgDirectorReport			100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:110:			describe				100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:158:			DirectorPathIn				100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:164:			NewDirectorStore			100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:169:			Path					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:176:			Read					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:196:			Attach					96.4%
github.com/yoshpy-dev/ralph/internal/org/director.go:252:			Detach					96.4%
github.com/yoshpy-dev/ralph/internal/org/director.go:299:			withLock				100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:303:			now					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:312:			readRaw					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:330:			appendLocked				76.5%
github.com/yoshpy-dev/ralph/internal/org/director.go:357:			registrationOf				100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:378:			foldDirector				100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:400:			seeN					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:406:			apply					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:436:			nextID					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:443:			result					100.0%
github.com/yoshpy-dev/ralph/internal/org/director.go:450:			directorIDNumber			100.0%
```

Functions changed in other files:
```text
github.com/yoshpy-dev/ralph/internal/cli/org.go:172:				withMainWorktreeOrgLimits		100.0%
github.com/yoshpy-dev/ralph/internal/cli/org.go:201:				withMainWorktreeDirectorEnabled		100.0%
github.com/yoshpy-dev/ralph/internal/cli/org.go:219:				loadMainWorktreeOrgConfig		100.0%
github.com/yoshpy-dev/ralph/internal/cli/org.go:403:				wireOrgEscalateWatch			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org.go:413:				orgEscalateWatchProbe			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_approve.go:119:			printApprovedPlans			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_inbox.go:30:			newOrgEscalateCmd			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_inbox.go:402:			inboxNotifiedColumn			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_inbox.go:412:			inboxItemDeferred			100.0%
github.com/yoshpy-dev/ralph/internal/cli/org_inbox.go:421:			printInboxItem				100.0%
github.com/yoshpy-dev/ralph/internal/cli/status.go:112:				runStatus				93.3%
github.com/yoshpy-dev/ralph/internal/cli/status.go:216:				statusOrgConfig				100.0%
github.com/yoshpy-dev/ralph/internal/cli/status.go:237:				statusDirectorEnabled			100.0%
github.com/yoshpy-dev/ralph/internal/cli/status.go:401:				printStatusEmpty			94.1%
github.com/yoshpy-dev/ralph/internal/cli/status.go:586:				printStatusJSONAllOrgs			100.0%
github.com/yoshpy-dev/ralph/internal/cli/status.go:622:				printStatusTableAllOrgs			96.0%
github.com/yoshpy-dev/ralph/internal/org/approve.go:112:			Approve					89.3%
github.com/yoshpy-dev/ralph/internal/org/approve.go:163:			recordUnderLock				93.3%
github.com/yoshpy-dev/ralph/internal/org/approve.go:306:			checkNotApprovedYet			100.0%
github.com/yoshpy-dev/ralph/internal/org/approve.go:408:			ApprovalDecisionCommand			85.7%
github.com/yoshpy-dev/ralph/internal/org/approve.go:427:			ApprovalResolveCommand			100.0%
github.com/yoshpy-dev/ralph/internal/org/approve.go:442:			checkResolvedMeanwhile			100.0%
github.com/yoshpy-dev/ralph/internal/org/deadline.go:58:			InboxDeadlineDue			100.0%
github.com/yoshpy-dev/ralph/internal/org/escalate.go:173:			Escalate				95.2%
github.com/yoshpy-dev/ralph/internal/org/escalate.go:231:			NotifyInboxItem				86.7%
github.com/yoshpy-dev/ralph/internal/org/escalate.go:310:			escalateDelivery			100.0%
github.com/yoshpy-dev/ralph/internal/org/escalate.go:328:			inboxDeferredBanner			100.0%
github.com/yoshpy-dev/ralph/internal/org/inbox.go:214:				Escalate				95.7%
github.com/yoshpy-dev/ralph/internal/org/inbox.go:423:				foldInbox				100.0%
github.com/yoshpy-dev/ralph/internal/org/inbox.go:451:				apply					100.0%
```

Blocks with count 0 in the new code: `director.go` 231 (append failure of a plain attach), 287-289 (append failure of a detach), 337-339 (marshal failure), 346-352 (write and close failures); `org_director.go` 121 (a store error other than ErrDirectorAttached), 319 (the `./ralph.toml` fallback when `filepath.Abs` fails), 334-336 (`os.Getwd` failure). The rest of the count-0 blocks in the changed files are lines from before this diff (I/O error paths, `o.Inbox == nil`).

## 3. HEAD binary probe (scratch repo; PATH holds only git, ralph, and a stub osascript)

The binary was built from HEAD with `GOTOOLCHAIN=go1.25.8 go build -o $SCRATCH/bin/ralph ./cmd/ralph`. Every command ran under `env -i HOME=$SCRATCH/probe/home PATH=$SCRATCH/probe/pbin USER=$USER TMPDIR=... GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null`, so no herdr, agmsg, or real osascript could run; the stub `osascript` appends its arguments to `osascript.log` and exits 0, which stands for the human path. The scratch repo's main worktree `ralph.toml` sets `[org.watchdog].interval_seconds = 5` and `[org.director]` `enabled = true`, `ack_minutes = 1`, `resolve_minutes = 2`. The watch ran detached (`ralph org watch --all --detach`) and was stopped with `--stop`; afterwards `pgrep -fl $SCRATCH/bin/ralph` found nothing.

### 3.1 Script

```bash
#!/bin/bash
# Behavioral probe of org-director-attach with the HEAD binary.
# Scratch repo only; PATH holds git, ralph, and a stub osascript that logs.
set -u
T=$SCRATCH
S=$T/probe
rm -rf "$S"
mkdir -p "$S/home" "$S/pbin" "$S/tmp"
ln -s /opt/homebrew/bin/git "$S/pbin/git"
ln -s "$T/bin/ralph" "$S/pbin/ralph"
cat > "$S/pbin/osascript" <<EOF
#!/bin/sh
printf '%s osascript %s\n' "\$(/bin/date -u +%Y-%m-%dT%H:%M:%SZ)" "\$*" >> "$S/osascript.log"
exit 0
EOF
chmod +x "$S/pbin/osascript"
: > "$S/osascript.log"

E() { env -i HOME="$S/home" PATH="$S/pbin" USER="$USER" TMPDIR="$S/tmp" GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$@"; }
# r <label-free> args...: run ralph in the current directory, show stdout, stderr, exit
r() {
  printf '\n$ (cwd=%s) ralph' "$(basename "$PWD")"; printf ' %q' "$@"; printf '\n'
  E ralph "$@" > "$S/o" 2> "$S/e"; rc=$?
  if [ -s "$S/o" ]; then echo "[stdout]"; cat "$S/o"; fi
  if [ -s "$S/e" ]; then echo "[stderr]"; cat "$S/e"; fi
  echo "[exit $rc]"
}
show() { for f in "$@"; do echo; echo "--- ${f#$S/}"; if [ -f "$f" ]; then cat "$f"; else echo "(absent)"; fi; done; }
now() { /bin/date -u +%Y-%m-%dT%H:%M:%SZ; }
L=$S/repo/.harness/state/org

mkdir -p "$S/repo" && cd "$S/repo"
E git init -q -b main
cat > ralph.toml <<'EOF'
[org.watchdog]
interval_seconds = 5

[org.director]
enabled = true
ack_minutes = 1
resolve_minutes = 2
EOF
E git add ralph.toml
E git -c user.name=t -c user.email=t@example.invalid commit -q -m init
echo "main ralph.toml:"; cat ralph.toml

echo; echo "## P1. no watch: attach warns; status and director status show the registration; escalate sends at once"
r org director attach --driver claude
r status
r status --json
r org director status
r org director status --json
r org escalate --org-id org-a --text $'TYPE: QUESTION\n\nno watch: sent at once?'
show "$L/director.jsonl" "$L/inbox.jsonl" "$L/escalations.jsonl" "$S/osascript.log"

echo; echo "## P2. a long-running watch (--detach): escalate records only; the ack deadline sends it"
r org watch --all --detach
r status
P2_T0=$(/bin/date +%s); echo "T0 $(now)"
r org escalate --org-id org-a --text $'TYPE: QUESTION\n\nleft while d1 is attached'
# A -> B (--replace, no USER, HERDR_PANE_ID set) -> A detach refused -> B detach
printf '\n$ (no USER, HERDR_PANE_ID=w1:p1) ralph org director attach --driver codex --replace\n'
env -i HOME="$S/home" PATH="$S/pbin" TMPDIR="$S/tmp" GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null HERDR_PANE_ID=w1:p1 \
  ralph org director attach --driver codex --replace > "$S/o" 2> "$S/e"; rc=$?
echo "[stdout]"; cat "$S/o"; [ -s "$S/e" ] && { echo "[stderr]"; cat "$S/e"; }; echo "[exit $rc]"
r org director detach --id d1
r org director status
r org director detach
r org director detach --id d2 --force
r org director attach --driver claude --org-id org-a
r org director detach --id d2
r org director detach --id d2
r org director status --json
r status
r org escalate --org-id org-a --text $'TYPE: BLOCKED\nTASK_ID: org-a\n\nleft with no director attached'
r org escalate --org-id org-a --text $'TYPE: QUESTION\n\nacked at once, never resolved'
r org inbox ack e4
r org inbox
r org inbox show e2
show "$L/escalations.jsonl" "$S/osascript.log"
echo; echo "poll: NOTIFIED column of e2..e4 every 5 s (elapsed seconds since T0)"
sent_ack=""
while :; do
  el=$(( $(/bin/date +%s) - P2_T0 ))
  line=$(E ralph org inbox 2>&1 | /usr/bin/awk -F'\t' '$1 ~ /^e[234]$/ {printf "%s=%s/%s ", $1, $2, $7}')
  echo "t+${el}s $(now) $line"
  case "$line" in *"e2=open/yes"*"e3=open/yes"*) [ -z "$sent_ack" ] && sent_ack=$el ;; esac
  [ -n "$sent_ack" ] && [ "$el" -ge $((sent_ack + 10)) ] && break
  [ "$el" -ge 110 ] && break
  /bin/sleep 5
done
echo "ack-deadline send observed at t+${sent_ack:-never}s"
r org inbox
r org inbox show e2
show "$L/inbox.jsonl" "$L/escalations.jsonl" "$S/osascript.log"
echo; echo "wait for the resolve deadline (resolve_minutes = 2) of e1..e4"
while :; do
  el=$(( $(/bin/date +%s) - P2_T0 ))
  n=$(/usr/bin/grep -c inbox_resolve_deadline "$L/escalations.jsonl" 2>/dev/null || true)
  echo "t+${el}s $(now) resolve-deadline lines in escalations.jsonl: ${n:-0}"
  [ "${n:-0}" -ge 4 ] && break
  [ "$el" -ge 175 ] && break
  /bin/sleep 5
done
r org inbox
show "$L/inbox.jsonl" "$L/escalations.jsonl" "$S/osascript.log"
r org watch --stop
show "$L/watch.log"

echo; echo "## P3. the watch stopped: escalate sends at once again (inbox_no_watch)"
r status
r org escalate --org-id org-a --text $'TYPE: QUESTION\n\nafter the watch stopped'
show "$L/escalations.jsonl"

echo; echo "## P4. main worktree's ralph.toml decides [org.director].enabled (AC5); a feature worktree's copy does not"
E git worktree add -q -b feat "$S/feat"
cd "$S/feat"
/usr/bin/sed 's/^enabled = true/enabled = false/' ralph.toml > ralph.toml.new && /bin/mv ralph.toml.new ralph.toml
echo "feat ralph.toml:"; cat ralph.toml
echo "### P4a. main enabled = true, feature enabled = false; run from the feature worktree"
r org director attach --driver claude
r org escalate --org-id org-f --text $'TYPE: QUESTION\n\nfrom the feature worktree, main enabled'
r status
r org director detach --id d3
cd "$S/repo"
/usr/bin/sed 's/^enabled = true/enabled = false/' ralph.toml > ralph.toml.new && /bin/mv ralph.toml.new ralph.toml
cd "$S/feat"
/usr/bin/sed 's/^enabled = false/enabled = true/' ralph.toml > ralph.toml.new && /bin/mv ralph.toml.new ralph.toml
echo "### P4b. main enabled = false, feature enabled = true; run from the feature worktree"
echo "main:"; cat "$S/repo/ralph.toml"; echo "feat:"; cat ralph.toml
r org director attach --driver claude
r status
r status --state-dir "$L"
r status --json --state-dir "$L"
r org director status
cd "$S/repo"
echo "### P4c. main enabled = false with a watch running: escalate still sends at once (inbox_director_disabled)"
r org watch --all --detach
r status
r org escalate --org-id org-a --text $'TYPE: QUESTION\n\ndisabled while a watch runs'
r org inbox
r org watch --stop
show "$L/director.jsonl" "$L/escalations.jsonl" "$S/osascript.log"
echo; echo "## P5. what the probe left: no herdr/agmsg dirs, files in the ledger"
/bin/ls -1 "$L"
/bin/ls -A "$S/home"
echo "done $(now)"
```

### 3.2 Output

Phases: P1 no watch; P2 a long-running watch, the ack deadline (1 min) and the resolve deadline (2 min) observed live, attach / `--replace` / detach refusals in between; P3 after `--stop`; P4 the main worktree's `ralph.toml` against a linked feature worktree's (AC5), including `ralph status --state-dir` from the linked worktree (the `git worktree list --porcelain` branch) and `enabled = false` with a watch running.

```text
main ralph.toml:
[org.watchdog]
interval_seconds = 5

[org.director]
enabled = true
ack_minutes = 1
resolve_minutes = 2

## P1. no watch: attach warns; status and director status show the registration; escalate sends at once

$ (cwd=repo) ralph org director attach --driver claude
[stdout]
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1
[stderr]
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/repo/.harness/state/org (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach
[exit 0]

$ (cwd=repo) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running
director: attached d1 (driver=claude, attached at 2026-10-14T19:11:37Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=repo) ralph status --json
[stdout]
{
  "state_dir": "$SCRATCH/probe/repo/.harness/state/org",
  "state_dir_source": "git-main-worktree",
  "orgs": [],
  "org_watch": {
    "state": "not_running"
  },
  "director": {
    "enabled": true,
    "state": "attached",
    "registration": {
      "id": "d1",
      "driver": "claude",
      "wake": [
        "pull"
      ],
      "attached_at": "2026-10-14T19:11:37Z",
      "user": "$USER",
      "cwd": "$SCRATCH/probe/repo",
      "herdr_pane": false
    },
    "corrupt_lines": 0,
    "ignored_events": 0
  },
  "merge_guarantee": {
    "kind": "unknown",
    "reason": "not_checked",
    "detail": "not checked: run ralph doctor"
  }
}
[exit 0]

$ (cwd=repo) ralph org director status
[stdout]
director: attached d1 (driver=claude, attached at 2026-10-14T19:11:37Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-14T19:11:37Z
user: $USER
cwd: $SCRATCH/probe/repo
herdr_pane: no
[exit 0]

$ (cwd=repo) ralph org director status --json
[stdout]
{
  "enabled": true,
  "state": "attached",
  "registration": {
    "id": "d1",
    "driver": "claude",
    "wake": [
      "pull"
    ],
    "attached_at": "2026-10-14T19:11:37Z",
    "user": "$USER",
    "cwd": "$SCRATCH/probe/repo",
    "herdr_pane": false
  },
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nno watch: sent at once?'
[stdout]
escalated e1 (org=org-a type=QUESTION)
[stderr]
ORG ESCALATION: org=org-a item=e1 type=QUESTION -- recorded in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e1
[exit 0]

--- repo/.harness/state/org/director.jsonl
{"ts":"2026-10-14T19:11:37Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/repo","herdr_pane":false}

--- repo/.harness/state/org/inbox.jsonl
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nno watch: sent at once?","delivery":"now"}
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"ok"}

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}

--- osascript.log
2026-10-14T19:11:38Z osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"

## P2. a long-running watch (--detach): escalate records only; the ack deadline sends it

$ (cwd=repo) ralph org watch --all --detach
[stdout]
started ralph org watch --all (pid 42439); log: $SCRATCH/probe/repo/.harness/state/org/watch.log
[exit 0]

$ (cwd=repo) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: running (pid 42439, mark updated 2026-10-14T19:11:38Z)
director: attached d1 (driver=claude, attached at 2026-10-14T19:11:37Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]
T0 2026-10-14T19:11:38Z

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nleft while d1 is attached'
[stdout]
escalated e2 (org=org-a type=QUESTION)
[stderr]
ORG ESCALATION RECORDED: org=org-a item=e2 type=QUESTION -- left in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl for the director; if it is still open after the ack deadline of ralph org watch --all ([org.director].ack_minutes), the watch sends it to the human; read it with: ralph org inbox show e2
[exit 0]

$ (no USER, HERDR_PANE_ID=w1:p1) ralph org director attach --driver codex --replace
[stdout]
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-14T19:11:37Z)
detach it with: ralph org director detach --id d2
[exit 0]

$ (cwd=repo) ralph org director detach --id d1
[stderr]
org: director: detach d1: the id is not the current registration's: the current registration is d2 (driver codex, attached 2026-10-14T19:11:39Z), so the session that attached d1 was replaced (or d1 has already been detached) and does not need to detach
[exit 1]

$ (cwd=repo) ralph org director status
[stdout]
director: attached d2 (driver=codex, attached at 2026-10-14T19:11:39Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d2
driver: codex
wake: pull
attached_at: 2026-10-14T19:11:39Z
user: -
cwd: $SCRATCH/probe/repo
herdr_pane: yes
[exit 0]

$ (cwd=repo) ralph org director detach
[stderr]
org: director detach: give --id <id>, the id attach printed, or --force
[exit 1]

$ (cwd=repo) ralph org director detach --id d2 --force
[stderr]
org: director detach: --id and --force cannot be used together
[exit 1]

$ (cwd=repo) ralph org director attach --driver claude --org-id org-a
[stderr]
org: director attach: --org-id does not apply: the director registration belongs to the whole ledger, not to one org
[exit 1]

$ (cwd=repo) ralph org director detach --id d2
[stdout]
detached d2
[exit 0]

$ (cwd=repo) ralph org director detach --id d2
[stderr]
org: director: detach d2: no director is attached
[exit 1]

$ (cwd=repo) ralph org director status --json
[stdout]
{
  "enabled": true,
  "state": "not_attached",
  "registration": null,
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]

$ (cwd=repo) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: running (pid 42439, mark updated 2026-10-14T19:11:38Z)
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: BLOCKED\nTASK_ID: org-a\n\nleft with no director attached'
[stdout]
escalated e3 (org=org-a type=BLOCKED)
[stderr]
ORG ESCALATION RECORDED: org=org-a item=e3 type=BLOCKED -- left in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl for the director; if it is still open after the ack deadline of ralph org watch --all ([org.director].ack_minutes), the watch sends it to the human; read it with: ralph org inbox show e3
[exit 0]

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nacked at once, never resolved'
[stdout]
escalated e4 (org=org-a type=QUESTION)
[stderr]
ORG ESCALATION RECORDED: org=org-a item=e4 type=QUESTION -- left in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl for the director; if it is still open after the ack deadline of ralph org watch --all ([org.director].ack_minutes), the watch sends it to the human; read it with: ralph org inbox show e4
[exit 0]

$ (cwd=repo) ralph org inbox ack e4
[stdout]
acked e4
[exit 0]

$ (cwd=repo) ralph org inbox
[stdout]
ID	STATE	ORG	TYPE	TASK_ID	ESCALATED_AT	NOTIFIED	SUMMARY
e1	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	no watch: sent at once?
e2	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	deferred	left while d1 is attached
e3	open	org-a	BLOCKED	org-a	2026-10-14T19:11:39Z	deferred	left with no director attached
e4	acked	org-a	QUESTION	-	2026-10-14T19:11:39Z	deferred	acked at once, never resolved
[exit 0]

$ (cwd=repo) ralph org inbox show e2
[stdout]
id: e2
state: open
org: org-a
type: QUESTION
escalated_at: 2026-10-14T19:11:38Z
notified: deferred (left to the director, or to the inbox deadlines of ralph org watch --all; to send it to the human now: ralph org inbox notify e2)
message:
  TYPE: QUESTION

  left while d1 is attached
events:
  2026-10-14T19:11:38Z escalated
[exit 0]

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}

--- osascript.log
2026-10-14T19:11:38Z osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"

poll: NOTIFIED column of e2..e4 every 5 s (elapsed seconds since T0)
t+1s 2026-10-14T19:11:39Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+6s 2026-10-14T19:11:45Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+12s 2026-10-14T19:11:50Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+17s 2026-10-14T19:11:55Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+22s 2026-10-14T19:12:00Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+27s 2026-10-14T19:12:05Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+32s 2026-10-14T19:12:10Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+37s 2026-10-14T19:12:15Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+42s 2026-10-14T19:12:20Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+47s 2026-10-14T19:12:25Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+52s 2026-10-14T19:12:30Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+57s 2026-10-14T19:12:36Z e2=open/deferred e3=open/deferred e4=acked/deferred
t+63s 2026-10-14T19:12:41Z e2=open/yes e3=open/deferred e4=acked/deferred
t+68s 2026-10-14T19:12:46Z e2=open/yes e3=open/yes e4=acked/deferred
t+73s 2026-10-14T19:12:51Z e2=open/yes e3=open/yes e4=acked/deferred
t+78s 2026-10-14T19:12:56Z e2=open/yes e3=open/yes e4=acked/deferred
ack-deadline send observed at t+68s

$ (cwd=repo) ralph org inbox
[stdout]
ID	STATE	ORG	TYPE	TASK_ID	ESCALATED_AT	NOTIFIED	SUMMARY
e1	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	no watch: sent at once?
e2	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	left while d1 is attached
e3	open	org-a	BLOCKED	org-a	2026-10-14T19:11:39Z	yes	left with no director attached
e4	acked	org-a	QUESTION	-	2026-10-14T19:11:39Z	deferred	acked at once, never resolved
[exit 0]

$ (cwd=repo) ralph org inbox show e2
[stdout]
id: e2
state: open
org: org-a
type: QUESTION
escalated_at: 2026-10-14T19:11:38Z
notified: yes (2026-10-14T19:12:38Z)
message:
  TYPE: QUESTION

  left while d1 is attached
events:
  2026-10-14T19:11:38Z escalated
  2026-10-14T19:12:38Z notified reason=inbox_ack_deadline osascript=ok
[exit 0]

--- repo/.harness/state/org/inbox.jsonl
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nno watch: sent at once?","delivery":"now"}
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"ok"}
{"ts":"2026-10-14T19:11:38Z","id":"e2","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nleft while d1 is attached","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e3","event":"escalated","org_id":"org-a","type":"BLOCKED","task_id":"org-a","body":"TYPE: BLOCKED\nTASK_ID: org-a\n\nleft with no director attached","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e4","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nacked at once, never resolved","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e4","event":"acked"}
{"ts":"2026-10-14T19:12:38Z","id":"e2","event":"notified","reason":"inbox_ack_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:12:43Z","id":"e3","event":"notified","reason":"inbox_ack_deadline","osascript":"ok"}

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:12:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:12:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_ack_deadline"}

--- osascript.log
2026-10-14T19:11:38Z osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"
2026-10-14T19:12:38Z osascript -e display notification "org org-a: e2 (QUESTION) not acked within 1 min" with title "ralph org watch"
2026-10-14T19:12:43Z osascript -e display notification "org org-a: e3 (BLOCKED) not acked within 1 min" with title "ralph org watch"

wait for the resolve deadline (resolve_minutes = 2) of e1..e4
t+78s 2026-10-14T19:12:56Z resolve-deadline lines in escalations.jsonl: 0
t+83s 2026-10-14T19:13:01Z resolve-deadline lines in escalations.jsonl: 0
t+88s 2026-10-14T19:13:06Z resolve-deadline lines in escalations.jsonl: 0
t+93s 2026-10-14T19:13:11Z resolve-deadline lines in escalations.jsonl: 0
t+98s 2026-10-14T19:13:16Z resolve-deadline lines in escalations.jsonl: 0
t+104s 2026-10-14T19:13:22Z resolve-deadline lines in escalations.jsonl: 0
t+109s 2026-10-14T19:13:27Z resolve-deadline lines in escalations.jsonl: 0
t+114s 2026-10-14T19:13:32Z resolve-deadline lines in escalations.jsonl: 0
t+119s 2026-10-14T19:13:37Z resolve-deadline lines in escalations.jsonl: 0
t+124s 2026-10-14T19:13:42Z resolve-deadline lines in escalations.jsonl: 2
t+129s 2026-10-14T19:13:47Z resolve-deadline lines in escalations.jsonl: 4

$ (cwd=repo) ralph org inbox
[stdout]
ID	STATE	ORG	TYPE	TASK_ID	ESCALATED_AT	NOTIFIED	SUMMARY
e1	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	no watch: sent at once?
e2	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	left while d1 is attached
e3	open	org-a	BLOCKED	org-a	2026-10-14T19:11:39Z	yes	left with no director attached
e4	acked	org-a	QUESTION	-	2026-10-14T19:11:39Z	yes	acked at once, never resolved
[exit 0]

--- repo/.harness/state/org/inbox.jsonl
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nno watch: sent at once?","delivery":"now"}
{"ts":"2026-10-14T19:11:38Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"ok"}
{"ts":"2026-10-14T19:11:38Z","id":"e2","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nleft while d1 is attached","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e3","event":"escalated","org_id":"org-a","type":"BLOCKED","task_id":"org-a","body":"TYPE: BLOCKED\nTASK_ID: org-a\n\nleft with no director attached","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e4","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nacked at once, never resolved","delivery":"deadline"}
{"ts":"2026-10-14T19:11:39Z","id":"e4","event":"acked"}
{"ts":"2026-10-14T19:12:38Z","id":"e2","event":"notified","reason":"inbox_ack_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:12:43Z","id":"e3","event":"notified","reason":"inbox_ack_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:13:38Z","id":"e1","event":"notified","reason":"inbox_resolve_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:13:38Z","id":"e2","event":"notified","reason":"inbox_resolve_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:13:44Z","id":"e3","event":"notified","reason":"inbox_resolve_deadline","osascript":"ok"}
{"ts":"2026-10-14T19:13:44Z","id":"e4","event":"notified","reason":"inbox_resolve_deadline","osascript":"ok"}

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:12:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:12:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:44Z","org_id":"org-a","inbox_id":"e4","subject":"leader","reason":"inbox_resolve_deadline"}

--- osascript.log
2026-10-14T19:11:38Z osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"
2026-10-14T19:12:38Z osascript -e display notification "org org-a: e2 (QUESTION) not acked within 1 min" with title "ralph org watch"
2026-10-14T19:12:43Z osascript -e display notification "org org-a: e3 (BLOCKED) not acked within 1 min" with title "ralph org watch"
2026-10-14T19:13:38Z osascript -e display notification "org org-a: e1 (QUESTION) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:38Z osascript -e display notification "org org-a: e2 (QUESTION) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:44Z osascript -e display notification "org org-a: e3 (BLOCKED) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:44Z osascript -e display notification "org org-a: e4 (QUESTION) not resolved within 2 min" with title "ralph org watch"

$ (cwd=repo) ralph org watch --stop
[stdout]
stopped ralph org watch --all (pid 42439)
[exit 0]

--- repo/.harness/state/org/watch.log
watching every org (interval=5s once=false state-dir=$SCRATCH/probe/repo/.harness/state/org (source: flag))
ORG INBOX DEADLINE: org=org-a item=e2 type=QUESTION -- not acked within 1m (inbox_ack_deadline); read it with: ralph org inbox show e2 --state-dir '$SCRATCH/probe/repo/.harness/state/org'
ORG INBOX DEADLINE: org=org-a item=e3 type=BLOCKED -- not acked within 1m (inbox_ack_deadline); read it with: ralph org inbox show e3 --state-dir '$SCRATCH/probe/repo/.harness/state/org'
ORG INBOX DEADLINE: org=org-a item=e1 type=QUESTION -- not resolved within 2m (inbox_resolve_deadline); read it with: ralph org inbox show e1 --state-dir '$SCRATCH/probe/repo/.harness/state/org'
ORG INBOX DEADLINE: org=org-a item=e2 type=QUESTION -- not resolved within 2m (inbox_resolve_deadline); read it with: ralph org inbox show e2 --state-dir '$SCRATCH/probe/repo/.harness/state/org'
ORG INBOX DEADLINE: org=org-a item=e3 type=BLOCKED -- not resolved within 2m (inbox_resolve_deadline); read it with: ralph org inbox show e3 --state-dir '$SCRATCH/probe/repo/.harness/state/org'
ORG INBOX DEADLINE: org=org-a item=e4 type=QUESTION -- not resolved within 2m (inbox_resolve_deadline); read it with: ralph org inbox show e4 --state-dir '$SCRATCH/probe/repo/.harness/state/org'

## P3. the watch stopped: escalate sends at once again (inbox_no_watch)

$ (cwd=repo) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-14T19:13:48Z)
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nafter the watch stopped'
[stdout]
escalated e5 (org=org-a type=QUESTION)
[stderr]
ORG ESCALATION: org=org-a item=e5 type=QUESTION -- recorded in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e5
[exit 0]

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:12:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:12:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:44Z","org_id":"org-a","inbox_id":"e4","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:48Z","org_id":"org-a","inbox_id":"e5","subject":"leader","reason":"inbox_no_watch"}

## P4. main worktree's ralph.toml decides [org.director].enabled (AC5); a feature worktree's copy does not
feat ralph.toml:
[org.watchdog]
interval_seconds = 5

[org.director]
enabled = false
ack_minutes = 1
resolve_minutes = 2
### P4a. main enabled = true, feature enabled = false; run from the feature worktree

$ (cwd=feat) ralph org director attach --driver claude
[stdout]
attached d3 (driver=claude)
detach it with: ralph org director detach --id d3
[stderr]
warning: director attach: d3 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/repo/.harness/state/org (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach
[exit 0]

$ (cwd=feat) ralph org escalate --org-id org-f --text $'TYPE: QUESTION\n\nfrom the feature worktree, main enabled'
[stdout]
escalated e6 (org=org-f type=QUESTION)
[stderr]
ORG ESCALATION: org=org-f item=e6 type=QUESTION -- recorded in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e6
[exit 0]

$ (cwd=feat) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-14T19:13:48Z)
director: attached d3 (driver=claude, attached at 2026-10-14T19:13:48Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=feat) ralph org director detach --id d3
[stdout]
detached d3
[exit 0]
### P4b. main enabled = false, feature enabled = true; run from the feature worktree
main:
[org.watchdog]
interval_seconds = 5

[org.director]
enabled = false
ack_minutes = 1
resolve_minutes = 2
feat:
[org.watchdog]
interval_seconds = 5

[org.director]
enabled = true
ack_minutes = 1
resolve_minutes = 2

$ (cwd=feat) ralph org director attach --driver claude
[stderr]
org: director attach: [org.director].enabled is false in the main worktree's $SCRATCH/probe/repo/ralph.toml, so ralph org escalate already sends every item to the human at once and leaves nothing to a director; to use one, set [org.director].enabled = true there
[exit 1]

$ (cwd=feat) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-14T19:13:48Z)
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=feat) ralph status --state-dir $SCRATCH/probe/repo/.harness/state/org
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-14T19:13:48Z)
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=feat) ralph status --json --state-dir $SCRATCH/probe/repo/.harness/state/org
[stdout]
{
  "state_dir": "$SCRATCH/probe/repo/.harness/state/org",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "not_running",
    "pid": 42439,
    "started_at": "2026-10-14T19:11:38.754546Z",
    "updated_at": "2026-10-14T19:13:48Z",
    "stopped_at": "2026-10-14T19:13:48Z",
    "mark_state": "stopped"
  },
  "director": {
    "enabled": false,
    "state": "disabled",
    "registration": null,
    "corrupt_lines": 0,
    "ignored_events": 0
  },
  "merge_guarantee": {
    "kind": "unknown",
    "reason": "not_checked",
    "detail": "not checked: run ralph doctor"
  }
}
[exit 0]

$ (cwd=feat) ralph org director status
[stdout]
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
enabled: false
[exit 0]
### P4c. main enabled = false with a watch running: escalate still sends at once (inbox_director_disabled)

$ (cwd=repo) ralph org watch --all --detach
[stdout]
started ralph org watch --all (pid 66629); log: $SCRATCH/probe/repo/.harness/state/org/watch.log
[exit 0]

$ (cwd=repo) ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: running (pid 66629, mark updated 2026-10-14T19:13:49Z)
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ (cwd=repo) ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\ndisabled while a watch runs'
[stdout]
escalated e7 (org=org-a type=QUESTION)
[stderr]
ORG ESCALATION: org=org-a item=e7 type=QUESTION -- recorded in $SCRATCH/probe/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e7
[exit 0]

$ (cwd=repo) ralph org inbox
[stdout]
ID	STATE	ORG	TYPE	TASK_ID	ESCALATED_AT	NOTIFIED	SUMMARY
e1	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	no watch: sent at once?
e2	open	org-a	QUESTION	-	2026-10-14T19:11:38Z	yes	left while d1 is attached
e3	open	org-a	BLOCKED	org-a	2026-10-14T19:11:39Z	yes	left with no director attached
e4	acked	org-a	QUESTION	-	2026-10-14T19:11:39Z	yes	acked at once, never resolved
e5	open	org-a	QUESTION	-	2026-10-14T19:13:48Z	yes	after the watch stopped
e6	open	org-f	QUESTION	-	2026-10-14T19:13:48Z	yes	from the feature worktree, main enabled
e7	open	org-a	QUESTION	-	2026-10-14T19:13:49Z	yes	disabled while a watch runs
[exit 0]

$ (cwd=repo) ralph org watch --stop
[stdout]
stopped ralph org watch --all (pid 66629)
[exit 0]

--- repo/.harness/state/org/director.jsonl
{"ts":"2026-10-14T19:11:37Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/repo","herdr_pane":false}
{"ts":"2026-10-14T19:11:39Z","id":"d1","event":"detached","user":"","cwd":"$SCRATCH/probe/repo","herdr_pane":true}
{"ts":"2026-10-14T19:11:39Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"user":"","cwd":"$SCRATCH/probe/repo","herdr_pane":true}
{"ts":"2026-10-14T19:11:39Z","id":"d2","event":"detached","user":"$USER","cwd":"$SCRATCH/probe/repo","herdr_pane":false}
{"ts":"2026-10-14T19:13:48Z","id":"d3","event":"attached","driver":"claude","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/feat","herdr_pane":false}
{"ts":"2026-10-14T19:13:48Z","id":"d3","event":"detached","user":"$USER","cwd":"$SCRATCH/probe/feat","herdr_pane":false}

--- repo/.harness/state/org/escalations.jsonl
{"ts":"2026-10-14T19:11:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:12:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:12:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_ack_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:38Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:43Z","org_id":"org-a","inbox_id":"e3","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:44Z","org_id":"org-a","inbox_id":"e4","subject":"leader","reason":"inbox_resolve_deadline"}
{"ts":"2026-10-14T19:13:48Z","org_id":"org-a","inbox_id":"e5","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:13:48Z","org_id":"org-f","inbox_id":"e6","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T19:13:49Z","org_id":"org-a","inbox_id":"e7","subject":"leader","reason":"inbox_director_disabled"}

--- osascript.log
2026-10-14T19:11:38Z osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"
2026-10-14T19:12:38Z osascript -e display notification "org org-a: e2 (QUESTION) not acked within 1 min" with title "ralph org watch"
2026-10-14T19:12:43Z osascript -e display notification "org org-a: e3 (BLOCKED) not acked within 1 min" with title "ralph org watch"
2026-10-14T19:13:38Z osascript -e display notification "org org-a: e1 (QUESTION) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:38Z osascript -e display notification "org org-a: e2 (QUESTION) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:44Z osascript -e display notification "org org-a: e3 (BLOCKED) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:44Z osascript -e display notification "org org-a: e4 (QUESTION) not resolved within 2 min" with title "ralph org watch"
2026-10-14T19:13:48Z osascript -e display notification "org org-a raised e5 (QUESTION)" with title "ralph org escalate"
2026-10-14T19:13:48Z osascript -e display notification "org org-f raised e6 (QUESTION)" with title "ralph org escalate"
2026-10-14T19:13:49Z osascript -e display notification "org org-a raised e7 (QUESTION)" with title "ralph org escalate"

## P5. what the probe left: no herdr/agmsg dirs, files in the ledger
director.jsonl
director.lock
escalations.jsonl
inbox.jsonl
inbox.lock
watch-alive.json
watch.lock
watch.log
done 2026-10-14T19:13:50Z
probe rc=0
```

### 3.3 Environment variable, damaged line, unreadable file

```bash
#!/bin/bash
T=$SCRATCH
P=$T/probe2; rm -rf $P; mkdir -p $P/home $P/pbin $P/tmp $P/repo
ln -s /opt/homebrew/bin/git $P/pbin/git; ln -s $T/bin/ralph $P/pbin/ralph
printf '#!/bin/sh\nprintf "%%s\\n" "osascript $*" >> "%s/osascript.log"\nexit 0\n' "$P" > $P/pbin/osascript; chmod +x $P/pbin/osascript
L=$P/repo/.harness/state/org
E() { env -i HOME=$P/home PATH=$P/pbin TMPDIR=$P/tmp USER=u GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$@"; }
cd $P/repo && E git init -q -b main
echo "## Q1. RALPH_ORG_DIRECTOR_ENABLED in the environment is not read (only ralph.toml decides)"
printf '[org.director]\nenabled = false\n' > ralph.toml
for v in true false; do echo; echo "\$ RALPH_ORG_DIRECTOR_ENABLED=$v ralph org director status   # ralph.toml: enabled = false"; E RALPH_ORG_DIRECTOR_ENABLED=$v ralph org director status 2>&1; echo "[exit $?]"; done
printf '[org.director]\nenabled = true\n' > ralph.toml
echo; echo "\$ RALPH_ORG_DIRECTOR_ENABLED=false ralph org escalate ...   # ralph.toml: enabled = true, no watch"
E RALPH_ORG_DIRECTOR_ENABLED=false ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nenv var ignored?' 2>&1; echo "[exit $?]"
echo "--- escalations.jsonl"; cat $L/escalations.jsonl
echo; echo "## Q2. a damaged director.jsonl line holding d7: attach takes d8; status warns on stderr"
printf '{"ts":"2026-10-14T00:00:00Z","id":"d7","event":"attached","driver":"claude"\n' > $L/director.jsonl
echo "--- director.jsonl before"; cat $L/director.jsonl
echo; echo '$ ralph org director attach --driver claude'; E ralph org director attach --driver claude 2>&1; echo "[exit $?]"
echo; echo '$ ralph status'; E ralph status > $P/o 2> $P/e; rc=$?; echo "[stdout]"; cat $P/o; echo "[stderr]"; cat $P/e; echo "[exit $rc]"
echo "--- director.jsonl after"; cat $L/director.jsonl
echo; echo "## Q3. director.jsonl unreadable (mode 000): status says unreadable; escalate is unaffected"
chmod 000 $L/director.jsonl
echo '$ ralph status'; E ralph status 2>&1; echo "[exit $?]"
echo; echo '$ ralph status --json   (the director key)'; E ralph status --json 2>/dev/null | /usr/bin/sed -n '/"director"/,/^  },/p'
echo; echo '$ ralph org director status --json'; E ralph org director status --json 2>&1; echo "[exit $?]"
echo; echo '$ ralph org escalate ...'; E ralph org escalate --org-id org-a --text $'TYPE: QUESTION\n\nunreadable director.jsonl' 2>&1; echo "[exit $?]"
echo "--- escalations.jsonl (last line)"; tail -1 $L/escalations.jsonl
chmod 644 $L/director.jsonl
echo "--- osascript.log"; cat $P/osascript.log
```

```text
## Q1. RALPH_ORG_DIRECTOR_ENABLED in the environment is not read (only ralph.toml decides)

$ RALPH_ORG_DIRECTOR_ENABLED=true ralph org director status   # ralph.toml: enabled = false
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
enabled: false
[exit 0]

$ RALPH_ORG_DIRECTOR_ENABLED=false ralph org director status   # ralph.toml: enabled = false
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
enabled: false
[exit 0]

$ RALPH_ORG_DIRECTOR_ENABLED=false ralph org escalate ...   # ralph.toml: enabled = true, no watch
ORG ESCALATION: org=org-a item=e1 type=QUESTION -- recorded in $SCRATCH/probe2/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e1
escalated e1 (org=org-a type=QUESTION)
[exit 0]
--- escalations.jsonl
{"ts":"2026-10-14T19:36:14Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}

## Q2. a damaged director.jsonl line holding d7: attach takes d8; status warns on stderr
--- director.jsonl before
{"ts":"2026-10-14T00:00:00Z","id":"d7","event":"attached","driver":"claude"

$ ralph org director attach --driver claude
attached d8 (driver=claude)
detach it with: ralph org director detach --id d8
warning: director attach: d8 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe2/repo/.harness/state/org (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach
[exit 0]

$ ralph status
[stdout]
no org runtime state found (state-dir=$SCRATCH/probe2/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running
director: attached d8 (driver=claude, attached at 2026-10-14T19:36:15Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[stderr]
warning: 1 unreadable director line(s) skipped in $SCRATCH/probe2/repo/.harness/state/org/director.jsonl
[exit 0]
--- director.jsonl after
{"ts":"2026-10-14T00:00:00Z","id":"d7","event":"attached","driver":"claude"
{"ts":"2026-10-14T19:36:15Z","id":"d8","event":"attached","driver":"claude","wake":["pull"],"user":"u","cwd":"$SCRATCH/probe2/repo","herdr_pane":false}

## Q3. director.jsonl unreadable (mode 000): status says unreadable; escalate is unaffected
$ ralph status
no org runtime state found (state-dir=$SCRATCH/probe2/repo/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.
org watch: not running
director: unreadable (org: director: read $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: open $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: permission denied)
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph status --json   (the director key)
  "director": {
    "enabled": true,
    "state": "unreadable",
    "registration": null,
    "error": "org: director: read $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: open $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: permission denied",
    "corrupt_lines": 0,
    "ignored_events": 0
  },

$ ralph org director status --json
{
  "enabled": true,
  "state": "unreadable",
  "registration": null,
  "error": "org: director: read $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: open $SCRATCH/probe2/repo/.harness/state/org/director.jsonl: permission denied",
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]

$ ralph org escalate ...
ORG ESCALATION: org=org-a item=e2 type=QUESTION -- recorded in $SCRATCH/probe2/repo/.harness/state/org/inbox.jsonl; read it with: ralph org inbox show e2
escalated e2 (org=org-a type=QUESTION)
[exit 0]
--- escalations.jsonl (last line)
{"ts":"2026-10-14T19:36:15Z","org_id":"org-a","inbox_id":"e2","subject":"leader","reason":"inbox_no_watch"}
--- osascript.log
osascript -e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"
osascript -e display notification "org org-a raised e2 (QUESTION)" with title "ralph org escalate"
```

## 4. Mutation (go1.26.0, `go test -overlay`)

### 4.1 mutate.py

Each mutant replaces one file through an overlay and runs the package's tests filtered by `-run` (org: `Director|Escalate|Inbox|Deadline|Approve|NotifyInboxItem|EscalateWatch`; cli: `OrgDirector|OrgEscalate|OrgInbox|OrgApprove|StatusCmd|PrintApprovedPlans`; config: all). X0 and X0c pass an unchanged copy through the overlay as controls. 42 mutants, 2 controls; 493 s.

| ID | File | Mutation | Verdict | First failing tests |
| --- | --- | --- | --- | --- |
| X0 | `director.go` | control: unchanged file through the overlay | SURVIVED | - |
| X0c | `org_director.go` | control: unchanged file through the overlay | SURVIVED | - |
| D1 | `director.go` | a second attach without --replace silently replaces | RED | `TestDirectorAttach_RefusesASecondAttachWithoutReplace`, `TestDirectorAttach_EscapesAHandEditedAttachTimeInItsError`, `TestDirectorAttach_ConcurrentAttachesWithoutReplace_OnlyOneSucceeds` |
| D2 | `director.go` | --replace writes no detached for the old registration | RED | `TestDirectorAttach_ReplaceWritesTheDetachedAndANewAttached`, `TestDirectorDetach_WithTheCurrentIDEndsTheRegistration`, `TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor` |
| D3 | `director.go` | detach --id of a replaced id ends the successor | RED | `TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor` |
| D4 | `director.go` | ids in damaged lines are not counted (id reuse) | RED | `TestDirectorAttach_NeverReusesTheIDOfADamagedLine`, `TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable` |
| D5 | `director.go` | a detached of any id ends the current registration | SURVIVED | - |
| D6 | `director.go` | an attached while another is current replaces it in the fold | RED | `TestDirectorRead_SkipsAndCountsLinesItCannotUse` |
| D7 | `director.go` | no director lock | RED | `TestDirectorStore_LivesUnderTheGivenStateDir`, `TestDirectorAttach_ConcurrentReplacesEachEndTheOneBefore`, `TestDirectorLock_IsItsOwnLockAndSerializesWrites` |
| D8 | `director.go` | a torn last line is not closed before the next write | RED | `TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable` |
| D9 | `director.go` | an attached that reuses an id is applied | RED | `TestDirectorRead_SkipsAndCountsLinesItCannotUse` |
| E1 | `escalate.go` | enabled = false is ignored | RED | `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline`, `TestEscalate_RecordsSendsToTheHumanPathAndMarksNotified`, `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| E2 | `escalate.go` | a running --once watch counts as accepting work | RED | `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| E3 | `escalate.go` | no-watch sends carry the disabled reason | RED | `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| E4 | `escalate.go` | a deferred item is sent at once anyway | RED | `TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline`, `TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline`, `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| E5 | `escalate.go` | the deferred item is recorded with delivery now | RED | `TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline`, `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch`, `TestNotifyInboxItem_SendsADeferredItemNow` |
| E6 | `escalate.go` | inbox notify records the no-watch reason | RED | `TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline`, `TestEscalate_EscalationsUnwritable_ReportsTheIDAndInboxNotifyCompletesIt`, `TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason` |
| E7 | `escalate.go` | no probe wired means deferred | RED | `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| E8 | `escalate.go` | EscalateResult does not say the item was deferred | RED | `TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline`, `TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline`, `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` |
| I1 | `inbox.go` | an old line with no delivery folds as deadline | RED | `TestInboxRead_FoldsEventsIntoItemsInIDOrder`, `TestInboxEscalate_WritesAndFoldsTheDelivery` |
| DL1 | `deadline.go` | the ack deadline skips deferred items | RED | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline` |
| DL2 | `deadline.go` | the ack deadline also re-sends items sent at once | RED | `TestInboxDeadlineDue` |
| A1 | `approve.go` | a second approve of an item records again | RED | `TestOrgApprove_RefusesAnItemAlreadyApproved` |
| A2 | `approve.go` | an item resolved meanwhile is not flagged | BUILD | - |
| A3 | `approve.go` | the resolve command loses its note | RED | `TestOrgApprove_RefusesAnItemAlreadyApproved` |
| A4 | `approve.go` | approve resolves the item again (the 7a behavior) | RED | `TestOrgApprove_RecordsAndLeavesTheItemAndThePlan`, `TestOrgApprove_Batch`, `TestOrgApprove_RecordsTheFileAsItIsThen` |
| C1 | `org.go` | the main worktree's enabled is ignored | RED | `TestOrgDirector_EnabledFromTheMainWorktreeRalphToml`, `TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml` |
| C2 | `org_director.go` | attach is not refused when disabled | RED | `TestOrgDirector_EnabledFromTheMainWorktreeRalphToml` |
| C3 | `org_director.go` | --org-id is not refused | RED | `TestOrgDirector_RefusesOrgID` |
| C4 | `org_inbox.go` | a deferred item already sent still shows deferred | RED | `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly` |
| C5 | `org_inbox.go` | NOTIFIED shows no for a deferred item | RED | `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly` |
| C6 | `org_approve.go` | approve prints the resolve before the send | RED | `TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve`, `TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell`, `TestOrgApprove_TwoItemsAtOnce` |
| C7 | `org.go` | escalate gets no watch probe (always sends at once) | RED | `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly`, `TestOrgEscalate_ProbesTheLedgersWatch`, `TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml` |
| C8 | `status.go` | no director line when there is no org | RED | `TestOrgDirector_EnabledFromTheMainWorktreeRalphToml`, `TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines` |
| C9 | `status.go` | status never reads the main worktree's enabled | RED | `TestOrgDirector_EnabledFromTheMainWorktreeRalphToml`, `TestStatusCmd_DirectorEnabledFromTheMainWorktree` |
| C10 | `org_director.go` | attach never warns about a missing watch | RED | `TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork` |
| C11 | `org_director.go` | attach does not warn about a running --once watch | RED | `TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork` |
| C12 | `status.go` | no director line in the org table | RED | `TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines`, `TestStatusCmd_OrgWatchLineFollowsTheStateDirLine`, `TestStatusCmd_DirectorLinePerState` |
| C13 | `status.go` | no director key in --json with orgs | RED | `TestStatusCmd_DirectorLinePerState` |
| C14 | `status.go` | no director key in --json with no org | RED | `TestStatusCmd_DirectorLinePerState` |
| C15 | `org_director.go` | the director line says 'by -' when USER was not set | RED | `TestStatusCmd_DirectorLinePerState` |
| C16 | `org_director.go` | attach never records herdr_pane | RED | `TestOrgDirector_AttachReplaceAndDetachByID` |
| CF1 | `config.go` | the default of enabled is false | RED | `TestDefault_OrgDirector`, `TestLoad_OrgDirectorMissingSectionBackfillsDefault`, `TestLoad_OrgDirectorRoundTrip` |
| I2 | `inbox.go` | Escalate accepts an unknown delivery | RED | `TestInboxEscalate_RefusesAnUnknownDelivery` |

Raw lines:

```text
X0 SURVIVED 5s control: unchanged file through the overlay |
X0c SURVIVED 17s control: unchanged file through the overlay |
D1 RED 8s a second attach without --replace silently replaces | --- FAIL: TestDirectorAttach_RefusesASecondAttachWithoutReplace (0.00s); --- FAIL: TestDirectorAttach_EscapesAHandEditedAttachTimeInItsError (0.00s); --- FAIL: TestDirectorAttach_ConcurrentAttachesWithoutReplace_OnlyOneSucceeds (0.24s)
D2 RED 8s --replace writes no detached for the old registration | --- FAIL: TestDirectorAttach_ReplaceWritesTheDetachedAndANewAttached (0.00s); --- FAIL: TestDirectorDetach_WithTheCurrentIDEndsTheRegistration (0.00s); --- FAIL: TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor (0.00s); --- FAIL: TestDirectorDetach_ForceEndsTheCurrentRegistrationWhateverItsID (0.00s); --- FAIL: TestDirectorAttach_ConcurrentReplacesEachEndTheOneBefore (0.22s)
D3 RED 8s detach --id of a replaced id ends the successor | --- FAIL: TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor (0.00s)
D4 RED 8s ids in damaged lines are not counted (id reuse) | --- FAIL: TestDirectorAttach_NeverReusesTheIDOfADamagedLine (0.01s); --- FAIL: TestDirectorAttach_NeverReusesTheIDOfADamagedLine/a_damaged_line_holds_d7 (0.00s); --- FAIL: TestDirectorAttach_NeverReusesTheIDOfADamagedLine/spaces_around_the_colon (0.00s); --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable (0.00s); --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable/cut_after_the_id (0.00s); --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable/cut_inside_the_id (0.00s)
D5 SURVIVED 8s a detached of any id ends the current registration |
D6 RED 7s an attached while another is current replaces it in the fold | --- FAIL: TestDirectorRead_SkipsAndCountsLinesItCannotUse (0.00s)
D7 RED 7s no director lock | --- FAIL: TestDirectorStore_LivesUnderTheGivenStateDir (0.00s); --- FAIL: TestDirectorAttach_ConcurrentReplacesEachEndTheOneBefore (0.00s); --- FAIL: TestDirectorLock_IsItsOwnLockAndSerializesWrites (0.00s); --- FAIL: TestDirectorStore_ReportsAnUnreadableFileAndAnUnavailableLock (0.00s); --- FAIL: TestDirectorStore_ReportsAnUnreadableFileAndAnUnavailableLock/director.lock_is_a_directory (0.00s)
D8 RED 8s a torn last line is not closed before the next write | --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable (0.00s); --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable/cut_after_the_id (0.00s); --- FAIL: TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable/cut_inside_the_id (0.00s)
D9 RED 8s an attached that reuses an id is applied | --- FAIL: TestDirectorRead_SkipsAndCountsLinesItCannotUse (0.00s)
E1 RED 7s enabled = false is ignored | --- FAIL: TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline (0.00s); --- FAIL: TestEscalate_RecordsSendsToTheHumanPathAndMarksNotified (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/disabled,_a_watch_that_accepts_work,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/disabled,_a_watch_that_accepts_work,_director_attached_true (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/disabled,_no_EscalateWatch,_director_attached_false (0.00s)
E2 RED 8s a running --once watch counts as accepting work | --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_--once_run,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_--once_run,_director_attached_true (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_running_with_no_mark,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_running_with_no_mark,_director_attached_true (0.00s)
E3 RED 8s no-watch sends carry the disabled reason | --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_no_EscalateWatch,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_no_EscalateWatch,_director_attached_true (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_not_running,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_not_running,_director_attached_true (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_stale,_director_attached_false (0.00s)
E4 RED 8s a deferred item is sent at once anyway | --- FAIL: TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline/acked (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline/inbox_notify (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_watch_that_accepts_work,_director_attached_false (0.00s)
E5 RED 8s the deferred item is recorded with delivery now | --- FAIL: TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.04s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_watch_that_accepts_work,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_watch_that_accepts_work,_director_attached_true (0.00s); --- FAIL: TestNotifyInboxItem_SendsADeferredItemNow (0.00s)
E6 RED 8s inbox notify records the no-watch reason | --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline/inbox_notify (0.00s); --- FAIL: TestEscalate_EscalationsUnwritable_ReportsTheIDAndInboxNotifyCompletesIt (0.00s); --- FAIL: TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason (0.00s); --- FAIL: TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason/director_disabled (0.00s); --- FAIL: TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason/no_watch (0.00s)
E7 RED 8s no probe wired means deferred | --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_no_EscalateWatch,_director_attached_false (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_no_EscalateWatch,_director_attached_true (0.00s)
E8 RED 8s EscalateResult does not say the item was deferred | --- FAIL: TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline/acked (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline/inbox_notify (0.00s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch (0.03s); --- FAIL: TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch/enabled,_a_watch_that_accepts_work,_director_attached_false (0.00s)
I1 RED 8s an old line with no delivery folds as deadline | --- FAIL: TestInboxRead_FoldsEventsIntoItemsInIDOrder (0.00s); --- FAIL: TestInboxEscalate_WritesAndFoldsTheDelivery (0.00s)
DL1 RED 8s the ack deadline skips deferred items | --- FAIL: TestInboxDeadlineDue (0.00s); --- FAIL: TestInboxDeadlineDue/open,_not_notified,_just_past_the_ack_deadline (0.00s); --- FAIL: TestInboxDeadlineDue/open,_not_notified,_exactly_at_the_resolve_deadline:_ack_only (0.00s); --- FAIL: TestInboxDeadlineDue/escalated_at_with_an_offset_is_the_same_instant (0.00s); --- FAIL: TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline (0.00s)
DL2 RED 8s the ack deadline also re-sends items sent at once | --- FAIL: TestInboxDeadlineDue (0.00s); --- FAIL: TestInboxDeadlineDue/notified_by_escalate,_past_the_ack_deadline (0.00s); --- FAIL: TestInboxDeadlineDue/notified_by_escalate,_exactly_at_the_resolve_deadline (0.00s); --- FAIL: TestInboxDeadlineDue/notified_by_escalate_with_no_watch,_past_the_ack_deadline (0.00s); --- FAIL: TestInboxDeadlineDue/sent_by_inbox_notify,_past_the_ack_deadline (0.00s); --- FAIL: TestInboxDeadlineDue/notified_before_stage_8a_(inbox_no_director),_past_the_ack_deadline (0.00s)
A1 RED 8s a second approve of an item records again | --- FAIL: TestOrgApprove_RefusesAnItemAlreadyApproved (0.00s)
A2 BUILD 0s an item resolved meanwhile is not flagged |
A3 RED 8s the resolve command loses its note | --- FAIL: TestOrgApprove_RefusesAnItemAlreadyApproved (0.00s)
A4 RED 8s approve resolves the item again (the 7a behavior) | --- FAIL: TestOrgApprove_RecordsAndLeavesTheItemAndThePlan (0.00s); --- FAIL: TestOrgApprove_Batch (0.00s); --- FAIL: TestOrgApprove_RecordsTheFileAsItIsThen (0.00s); --- FAIL: TestOrgApprove_Tripwire (0.01s); --- FAIL: TestOrgApprove_Tripwire/HERDR_PANE_ID_unset_passes (0.00s); --- FAIL: TestOrgApprove_Tripwire/another_pane_passes (0.00s)
C1 RED 18s the main worktree's enabled is ignored | --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml (1.30s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_from_the_linked_worktree (0.26s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_the_worktree_says_true (0.24s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_--state-dir_names_the_shared_ledger (0.21s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/no_main_ralph.toml,_the_worktree_says_false (0.20s); --- FAIL: TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml (1.21s)
C2 RED 17s attach is not refused when disabled | --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml (1.70s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_from_the_linked_worktree (0.45s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_the_worktree_says_true (0.44s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_--state-dir_names_the_shared_ledger (0.20s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/--config_says_false (0.21s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/the_worktree_says_false,_--state-dir_names_another_ledger (0.21s)
C3 RED 16s --org-id is not refused | --- FAIL: TestOrgDirector_RefusesOrgID (0.13s)
C4 RED 18s a deferred item already sent still shows deferred | --- FAIL: TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly (0.06s)
C5 RED 28s NOTIFIED shows no for a deferred item | --- FAIL: TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly (0.09s)
C6 RED 17s approve prints the resolve before the send | --- FAIL: TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve (0.03s); --- FAIL: TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell (0.03s); --- FAIL: TestOrgApprove_TwoItemsAtOnce (0.06s); --- FAIL: TestOrgApprove_TheDecisionCommandRepeatsAnExplicitStateDir (0.07s); --- FAIL: TestOrgApprove_TheDecisionCommandRepeatsAnExplicitStateDir/flag (0.03s); --- FAIL: TestOrgApprove_TheDecisionCommandRepeatsAnExplicitStateDir/env (0.03s)
C7 RED 17s escalate gets no watch probe (always sends at once) | --- FAIL: TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly (0.03s); --- FAIL: TestOrgEscalate_ProbesTheLedgersWatch (0.13s); --- FAIL: TestOrgEscalate_ProbesTheLedgersWatch/a_running_watch (0.03s); --- FAIL: TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml (1.28s); --- FAIL: TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml/no_main_ralph.toml,_the_worktree_says_false (0.15s); --- FAIL: TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml/main_says_true,_the_worktree_says_false,_--state-dir_names_the_shared_ledger (0.14s)
C8 RED 17s no director line when there is no org | --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml (1.44s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_from_the_linked_worktree (0.24s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_the_worktree_says_true (0.26s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_--state-dir_names_the_shared_ledger (0.25s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/the_worktree_says_false,_--state-dir_names_another_ledger (0.26s); --- FAIL: TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines (0.11s)
C9 RED 13s status never reads the main worktree's enabled | --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml (1.35s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_from_the_linked_worktree (0.23s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_the_worktree_says_true (0.24s); --- FAIL: TestOrgDirector_EnabledFromTheMainWorktreeRalphToml/main_says_false,_--state-dir_names_the_shared_ledger (0.24s); --- FAIL: TestStatusCmd_DirectorEnabledFromTheMainWorktree (0.23s)
C10 RED 17s attach never warns about a missing watch | --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork (0.19s); --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork/no_watch (0.04s); --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork/a_--once_watch (0.03s); --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork/a_stale_watch (0.03s); --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork/a_stopping_watch (0.03s)
C11 RED 17s attach does not warn about a running --once watch | --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork (0.48s); --- FAIL: TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork/a_--once_watch (0.07s)
C12 RED 17s no director line in the org table | --- FAIL: TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines (0.13s); --- FAIL: TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines/two_orgs (0.05s); --- FAIL: TestStatusCmd_OrgWatchLineFollowsTheStateDirLine (0.04s); --- FAIL: TestStatusCmd_DirectorLinePerState (0.31s); --- FAIL: TestStatusCmd_DirectorLinePerState/not_attached,_orgs_true (0.02s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached,_orgs_true (0.01s)
C13 RED 16s no director key in --json with orgs | --- FAIL: TestStatusCmd_DirectorLinePerState (0.37s); --- FAIL: TestStatusCmd_DirectorLinePerState/not_attached,_orgs_true (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached,_orgs_true (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached_with_a_pane_and_no_user,_orgs_true (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/disabled,_orgs_true (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/disabled_with_a_registration_left,_orgs_true (0.02s)
C14 RED 17s no director key in --json with no org | --- FAIL: TestStatusCmd_DirectorLinePerState (0.43s); --- FAIL: TestStatusCmd_DirectorLinePerState/not_attached,_orgs_false (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached,_orgs_false (0.05s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached_with_a_pane_and_no_user,_orgs_false (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/disabled,_orgs_false (0.03s); --- FAIL: TestStatusCmd_DirectorLinePerState/disabled_with_a_registration_left,_orgs_false (0.03s)
C15 RED 18s the director line says 'by -' when USER was not set | --- FAIL: TestStatusCmd_DirectorLinePerState (0.47s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached_with_a_pane_and_no_user,_orgs_false (0.02s); --- FAIL: TestStatusCmd_DirectorLinePerState/attached_with_a_pane_and_no_user,_orgs_true (0.02s)
C16 RED 22s attach never records herdr_pane | --- FAIL: TestOrgDirector_AttachReplaceAndDetachByID (0.24s)
CF1 RED 1s the default of enabled is false | --- FAIL: TestDefault_OrgDirector (0.00s); --- FAIL: TestLoad_OrgDirectorMissingSectionBackfillsDefault (0.00s); --- FAIL: TestLoad_OrgDirectorRoundTrip (0.00s); --- FAIL: TestLoad_OrgDirectorRoundTrip/one_key_set_leaves_the_other_at_default (0.00s); --- FAIL: TestDefaultsLockStep (0.00s)
I2 RED 8s Escalate accepts an unknown delivery | --- FAIL: TestInboxEscalate_RefusesAnUnknownDelivery (0.01s); --- FAIL: TestInboxEscalate_RefusesAnUnknownDelivery/later (0.00s); --- FAIL: TestInboxEscalate_RefusesAnUnknownDelivery/NOW (0.00s); --- FAIL: TestInboxEscalate_RefusesAnUnknownDelivery/_deadline (0.00s); --- FAIL: TestInboxEscalate_RefusesAnUnknownDelivery/now_ (0.00s)
rc=0 493s
```

### 4.2 Follow-ups: A2 rewritten, the D5 survivor, two probe tests

- A2 did not build (`declared and not used: j`), so it was rewritten as A2b (`|| true` appended to the condition)
- D5 was run again against the whole org package (746 top-level tests and 1,523 subtests, all PASS): it survives there too. `TestDirectorRead_SkipsAndCountsLinesItCannotUse` has a `detached d2` while d1 is current, but under D5 that line ends d1 and the later `detached d1` becomes the ignored one, so the current registration (d4) and the ignored count (13) stay the same: the two lines compensate. The probe PD (a file with `attached d1` then `detached d2` only) passes at HEAD and fails under D5
- PA: two concurrent `Approve` calls on one item, 30 rounds (the self-review and verify reports left this unverified). It passes at HEAD and with `-race`, and fails under A1 (the check removed), so it detects a double record. Both goroutines share one `*Org`, but each `withManifestLock` call opens its own descriptor of `manifest.lock`, so flock serializes them as it would two processes
- The probe files were overlaid as `internal/org/zz_probe_*_test.go`; `ls internal/org | grep -c zz_probe` is 0 afterwards

```text
== A2b
--- FAIL: TestOrgApprove_ResolvedBySomeoneElse (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	5.659s
== D5-fullpkg
ok  	github.com/yoshpy-dev/ralph/internal/org	91.757s
== PD-HEAD
--- PASS: TestProbe_DetachedOfAnotherIDLeavesTheCurrentRegistration (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	1.247s
== PD-D5
    zz_probe_director_fold_test.go:12: current = <nil>, ignored 0; want d1 current and the detached of d2 ignored
--- FAIL: TestProbe_DetachedOfAnotherIDLeavesTheCurrentRegistration (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.093s
== PA-HEAD
--- PASS: TestProbe_ConcurrentApproveOfOneItem_OnlyOneRecords (0.92s)
ok  	github.com/yoshpy-dev/ralph/internal/org	2.104s
== PA-HEAD-race
--- PASS: TestProbe_ConcurrentApproveOfOneItem_OnlyOneRecords (0.93s)
ok  	github.com/yoshpy-dev/ralph/internal/org	3.148s
== PA-A1
    zz_probe_approve_race_test.go:39: round 0: 2 of the 2 concurrent approves recorded; want exactly 1
--- FAIL: TestProbe_ConcurrentApproveOfOneItem_OnlyOneRecords (0.03s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.149s
```

Probe sources:

```go
package org

import "testing"

// Probe (not committed): kills D5. A detached whose id is not the current
// registration's is ignored by the fold; the current registration stays.
func TestProbe_DetachedOfAnotherIDLeavesTheCurrentRegistration(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, directorAttachedLine("d1", "claude")+"\n"+directorDetachedLine("d2")+"\n")
	res := mustReadDirector(t, s)
	if res.Current == nil || res.Current.ID != "d1" || res.IgnoredEvents != 1 {
		t.Fatalf("current = %+v, ignored %d; want d1 current and the detached of d2 ignored", res.Current, res.IgnoredEvents)
	}
}
```

```go
package org

import (
	"errors"
	"sync"
	"testing"
)

// Probe (not committed): two concurrent Approve calls on one inbox item.
// checkNotApprovedYet runs under the manifest lock, so exactly one records
// and the other is refused with ErrPlanAlreadyApproved.
func TestProbe_ConcurrentApproveOfOneItem_OnlyOneRecords(t *testing.T) {
	for round := 0; round < 30; round++ {
		s := newApproveSetup(t)
		results := make([]ApproveResult, 2)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range results {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				results[i] = s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/w"})
			}(i)
		}
		close(start)
		wg.Wait()
		recorded := 0
		for i, r := range results {
			switch {
			case r.Err == nil && len(r.Approved) == 1:
				recorded++
			case errors.Is(r.Err, ErrPlanAlreadyApproved) && len(r.Approved) == 0:
			default:
				t.Fatalf("round %d: Approve %d = %+v; want recorded or refused as already approved", round, i, r)
			}
		}
		if recorded != 1 {
			t.Fatalf("round %d: %d of the 2 concurrent approves recorded; want exactly 1", round, recorded)
		}
		if got := planApprovedEvents(t, s.o); len(got) != 1 {
			t.Fatalf("round %d: %d plan_approved lines; want 1", round, len(got))
		}
	}
}
```

mutate.py (the mutant list is the source of truth for the table above):

```python
#!/usr/bin/env python3
"""Overlay mutants for org-director-attach. Never edits the worktree."""
import json, os, subprocess, sys, time

W = "~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach"
T = "$SCRATCH/mut"
os.makedirs(T, exist_ok=True)

ORG_RUN = "Director|Escalate|Inbox|Deadline|Approve|NotifyInboxItem|EscalateWatch"
CLI_RUN = "OrgDirector|OrgEscalate|OrgInbox|OrgApprove|StatusCmd|PrintApprovedPlans"
CFG_RUN = "."

# id, package, file, old, new, run-regex, intent
M = [
    ("X0", "org", "internal/org/director.go", None, None, ORG_RUN, "control: unchanged file through the overlay"),
    ("X0c", "cli", "internal/cli/org_director.go", None, None, CLI_RUN, "control: unchanged file through the overlay"),
    ("D1", "org", "internal/org/director.go", "if cur != nil && !a.Replace {", "if cur != nil && false {", ORG_RUN,
     "a second attach without --replace silently replaces"),
    ("D2", "org", "internal/org/director.go", "\t\tif cur != nil {\n\t\t\tevents = append(events, DirectorEvent{", "\t\tif false {\n\t\t\tevents = append(events, DirectorEvent{", ORG_RUN,
     "--replace writes no detached for the old registration"),
    ("D3", "org", "internal/org/director.go", "if !d.Force && d.ID != cur.ID {", "if false {", ORG_RUN,
     "detach --id of a replaced id ends the successor"),
    ("D4", "org", "internal/org/director.go", "directorRawIDPattern.FindAllSubmatch(raw, -1)", "directorRawIDPattern.FindAllSubmatch(nil, -1)", ORG_RUN,
     "ids in damaged lines are not counted (id reuse)"),
    ("D5", "org", "internal/org/director.go", "if f.current == nil || ev.ID != f.current.ID {", "if f.current == nil {", ORG_RUN,
     "a detached of any id ends the current registration"),
    ("D6", "org", "internal/org/director.go", "|| f.used[ev.ID] || f.current != nil {", "|| f.used[ev.ID] {", ORG_RUN,
     "an attached while another is current replaces it in the fold"),
    ("D7", "org", "internal/org/director.go", "return withFileLock(filepath.Dir(s.path), directorLockFile, \"director\", fn)", "_ = os.MkdirAll(filepath.Dir(s.path), 0o755)\n\treturn fn()", ORG_RUN,
     "no director lock"),
    ("D8", "org", "internal/org/director.go", "if len(raw) > 0 && raw[len(raw)-1] != '\\n' {", "if false {", ORG_RUN,
     "a torn last line is not closed before the next write"),
    ("D9", "org", "internal/org/director.go", "|| f.used[ev.ID] || f.current != nil {", "|| f.current != nil {", ORG_RUN,
     "an attached that reuses an id is applied"),
    ("E1", "org", "internal/org/escalate.go", "if !o.Config.Director.Enabled {\n\t\treturn InboxDeliveryNow, InboxReasonDirectorDisabled", "if false {\n\t\treturn InboxDeliveryNow, InboxReasonDirectorDisabled", ORG_RUN,
     "enabled = false is ignored"),
    ("E2", "org", "internal/org/escalate.go", "if o.EscalateWatch == nil || !WatchAcceptsWork(o.EscalateWatch()) {", "if o.EscalateWatch == nil || o.EscalateWatch().State != WatchLivenessRunning {", ORG_RUN,
     "a running --once watch counts as accepting work"),
    ("E3", "org", "internal/org/escalate.go", "\t\treturn InboxDeliveryNow, InboxReasonNoWatch", "\t\treturn InboxDeliveryNow, InboxReasonDirectorDisabled", ORG_RUN,
     "no-watch sends carry the disabled reason"),
    ("E4", "org", "internal/org/escalate.go", "\tif delivery == InboxDeliveryDeadline {\n\t\t_, _ = fmt.Fprintln(banner", "\tif false {\n\t\t_, _ = fmt.Fprintln(banner", ORG_RUN,
     "a deferred item is sent at once anyway"),
    ("E5", "org", "internal/org/escalate.go", "Body: p.Text, Delivery: delivery})", "Body: p.Text, Delivery: InboxDeliveryNow})", ORG_RUN,
     "the deferred item is recorded with delivery now"),
    ("E6", "org", "internal/org/escalate.go", "InboxReasonManualNotify, bannerOrStderr(banner)", "InboxReasonNoWatch, bannerOrStderr(banner)", ORG_RUN,
     "inbox notify records the no-watch reason"),
    ("E7", "org", "internal/org/escalate.go", "if o.EscalateWatch == nil || !WatchAcceptsWork(o.EscalateWatch()) {", "if o.EscalateWatch != nil && !WatchAcceptsWork(o.EscalateWatch()) {", ORG_RUN,
     "no probe wired means deferred"),
    ("E8", "org", "internal/org/escalate.go", "\t\tres.Deferred = true\n", "\t\tres.Deferred = false\n", ORG_RUN,
     "EscalateResult does not say the item was deferred"),
    ("I1", "org", "internal/org/inbox.go", "\t\tif delivery != InboxDeliveryDeadline {\n\t\t\tdelivery = InboxDeliveryNow", "\t\tif delivery == \"\" {\n\t\t\tdelivery = InboxDeliveryDeadline", ORG_RUN,
     "an old line with no delivery folds as deadline"),
    ("I2", "org", "internal/org/inbox.go", "\tcase InboxDeliveryNow, InboxDeliveryDeadline:\n\tdefault:", "\tdefault:", ORG_RUN,
     "Escalate refuses every explicit delivery (sanity) -- replaced below"),
    ("DL1", "org", "internal/org/deadline.go", "since > time.Duration(ackMinutes)*time.Minute && !item.Notified {", "since > time.Duration(ackMinutes)*time.Minute && !item.Notified && item.Delivery != InboxDeliveryDeadline {", ORG_RUN,
     "the ack deadline skips deferred items"),
    ("DL2", "org", "internal/org/deadline.go", "since > time.Duration(ackMinutes)*time.Minute && !item.Notified {", "since > time.Duration(ackMinutes)*time.Minute && (!item.Notified || item.Delivery == InboxDeliveryNow) {", ORG_RUN,
     "the ack deadline also re-sends items sent at once"),
    ("A1", "org", "internal/org/approve.go", "c.err = checkNotApprovedYet(rr.Events, c, o.InboxHintStateDir)", "c.err = nil", ORG_RUN,
     "a second approve of an item records again"),
    ("A2", "org", "internal/org/approve.go", "if j < 0 || inbox.Items[j].State != InboxStateResolved {", "if true {", ORG_RUN,
     "an item resolved meanwhile is not flagged"),
    ("A3", "org", "internal/org/approve.go", "return InboxCommand(\"resolve\", id, stateDir) + \" --note \" + shellQuote(approveNote(ref))", "return InboxCommand(\"resolve\", id, stateDir)", ORG_RUN,
     "the resolve command loses its note"),
    ("A4", "org", "internal/org/approve.go", "\tres.Err = errors.Join(res.Err, o.checkResolvedMeanwhile(res.Approved))", "\tfor _, a := range res.Approved {\n\t\t_ = o.Inbox.Resolve(a.ID, \"approved\")\n\t}\n\tres.Err = errors.Join(res.Err, o.checkResolvedMeanwhile(res.Approved))", ORG_RUN,
     "approve resolves the item again (the 7a behavior)"),
    ("C1", "cli", "internal/cli/org.go", "\tcfg.Director.Enabled = mainCfg.Director.Enabled\n", "\t_ = mainCfg\n", CLI_RUN,
     "the main worktree's enabled is ignored"),
    ("C2", "cli", "internal/cli/org_director.go", "if !rt.cfg.Director.Enabled {", "if false {", CLI_RUN,
     "attach is not refused when disabled"),
    ("C3", "cli", "internal/cli/org_director.go", "if cmd.Flags().Changed(\"org-id\") {", "if false {", CLI_RUN,
     "--org-id is not refused"),
    ("C4", "cli", "internal/cli/org_inbox.go", "return item.Delivery == org.InboxDeliveryDeadline && !item.Notified && item.State != org.InboxStateResolved", "return item.Delivery == org.InboxDeliveryDeadline && item.State != org.InboxStateResolved", CLI_RUN,
     "a deferred item already sent still shows deferred"),
    ("C5", "cli", "internal/cli/org_inbox.go", "\tif inboxItemDeferred(item) {\n\t\treturn \"deferred\"", "\tif false {\n\t\treturn \"deferred\"", CLI_RUN,
     "NOTIFIED shows no for a deferred item"),
    ("C6", "cli", "internal/cli/org_approve.go",
     "\t\t_, _ = fmt.Fprintf(stdout, \"approve sends no message and resolves nothing; tell the leader of org %s with:\\n%s\\n\",\n\t\t\torgID, org.ApprovalDecisionCommand(a.OrgID, ref, hintStateDir))\n\t\tif !a.ResolvedElsewhere {\n\t\t\t_, _ = fmt.Fprintf(stdout, \"once that send succeeded, resolve the item with:\\n%s\\n\",\n\t\t\t\torg.ApprovalResolveCommand(a.ID, ref, hintStateDir))\n\t\t}\n",
     "\t\tif !a.ResolvedElsewhere {\n\t\t\t_, _ = fmt.Fprintf(stdout, \"once that send succeeded, resolve the item with:\\n%s\\n\",\n\t\t\t\torg.ApprovalResolveCommand(a.ID, ref, hintStateDir))\n\t\t}\n\t\t_, _ = fmt.Fprintf(stdout, \"approve sends no message and resolves nothing; tell the leader of org %s with:\\n%s\\n\",\n\t\t\torgID, org.ApprovalDecisionCommand(a.OrgID, ref, hintStateDir))\n",
     CLI_RUN, "approve prints the resolve before the send"),
    ("C7", "cli", "internal/cli/org.go", "\trt.EscalateWatch = orgEscalateWatchProbe(resolvedStateDir, rt.Config.DeadmanMinutes)\n", "\t_ = orgEscalateWatchProbe\n", CLI_RUN,
     "escalate gets no watch probe (always sends at once)"),
    ("C8", "cli", "internal/cli/status.go", "\t_, _ = fmt.Fprintln(out, orgWatchLine(watch))\n\t_, _ = fmt.Fprintln(out, orgDirectorLine(director))\n\t_, _ = fmt.Fprintln(out, mergeGuaranteeLine(guarantee))\n\tif corruptLines > 0 {", "\t_, _ = fmt.Fprintln(out, orgWatchLine(watch))\n\t_, _ = fmt.Fprintln(out, mergeGuaranteeLine(guarantee))\n\tif corruptLines > 0 {", CLI_RUN,
     "no director line when there is no org"),
    ("C9", "cli", "internal/cli/status.go", "\tif !loaded {\n\t\treturn cfg.Director.Enabled\n\t}\n\tfallback", "\tif true {\n\t\treturn cfg.Director.Enabled\n\t}\n\tfallback", CLI_RUN,
     "status never reads the main worktree's enabled"),
    ("C10", "cli", "internal/cli/org_director.go", "if !org.WatchAcceptsWork(probe) {", "if false {", CLI_RUN,
     "attach never warns about a missing watch"),
    ("C11", "cli", "internal/cli/org_director.go", "if !org.WatchAcceptsWork(probe) {", "if probe.State != org.WatchLivenessRunning {", CLI_RUN,
     "attach does not warn about a running --once watch"),
    ("C12", "cli", "internal/cli/status.go", "\tprintStateDirLine(out, stateDir, stateDirSource)\n\t_, _ = fmt.Fprintln(out, orgWatchLine(watch))\n\t_, _ = fmt.Fprintln(out, orgDirectorLine(director))\n", "\tprintStateDirLine(out, stateDir, stateDirSource)\n\t_, _ = fmt.Fprintln(out, orgWatchLine(watch))\n", CLI_RUN,
     "no director line in the org table"),
    ("C13", "cli", "internal/cli/status.go", "Orgs: orgsJSON, CorruptLines: corruptLines, OrgWatch: buildStatusOrgWatchJSON(watch), Director: director,", "Orgs: orgsJSON, CorruptLines: corruptLines, OrgWatch: buildStatusOrgWatchJSON(watch),", CLI_RUN,
     "no director key in --json with orgs"),
    ("C14", "cli", "internal/cli/status.go", "Orgs: []statusOrgJSON{}, CorruptLines: corruptLines, OrgWatch: buildStatusOrgWatchJSON(watch), Director: director,", "Orgs: []statusOrgJSON{}, CorruptLines: corruptLines, OrgWatch: buildStatusOrgWatchJSON(watch),", CLI_RUN,
     "no director key in --json with no org"),
    ("C15", "cli", "internal/cli/org_director.go", "\t\tif reg.User != \"\" {\n\t\t\tdetails += \", by \"", "\t\tif true {\n\t\t\tdetails += \", by \"", CLI_RUN,
     "the director line says 'by -' when USER was not set"),
    ("C16", "cli", "internal/cli/org_director.go", "HerdrPane: os.Getenv(\"HERDR_PANE_ID\") != \"\"}", "HerdrPane: false}", CLI_RUN,
     "attach never records herdr_pane"),
    ("CF1", "config", "internal/config/config.go", "\t\t\t\tEnabled:        true,\n\t\t\t\tAckMinutes:     10,", "\t\t\t\tEnabled:        false,\n\t\t\t\tAckMinutes:     10,", CFG_RUN,
     "the default of enabled is false"),
]
# I2 above is replaced by a meaningful one: an unknown delivery is accepted.
M = [m for m in M if m[0] != "I2"] + [
    ("I2", "org", "internal/org/inbox.go", "\tdefault:\n\t\treturn \"\", fmt.Errorf(\"org: inbox: escalate: delivery %q is not %s or %s\", in.Delivery, InboxDeliveryNow, InboxDeliveryDeadline)", "\tdefault:", ORG_RUN,
     "Escalate accepts an unknown delivery"),
]

only = set(sys.argv[1:])
results = []
for mid, pkg, rel, old, new, run, intent in M:
    if only and mid not in only:
        continue
    src = os.path.join(W, rel)
    text = open(src).read()
    if old is not None:
        n = text.count(old)
        if n != 1:
            results.append((mid, "SKIP", f"old string found {n} times"))
            print(mid, "SKIP", n, flush=True)
            continue
        text = text.replace(old, new)
    mpath = os.path.join(T, f"{mid}.go")
    open(mpath, "w").write(text)
    ov = os.path.join(T, f"{mid}.json")
    json.dump({"Replace": {src: mpath}}, open(ov, "w"))
    cmd = ["go", "test", "-count=1", f"-overlay={ov}", "-run", run, f"./internal/{pkg}/"]
    t0 = time.time()
    try:
        p = subprocess.run(cmd, cwd=W, capture_output=True, text=True, timeout=900)
        out, rc = p.stdout + p.stderr, p.returncode
    except subprocess.TimeoutExpired as e:
        out, rc = "TIMEOUT", -1
    dt = time.time() - t0
    open(os.path.join(T, f"{mid}.out"), "w").write(out)
    if "[build failed]" in out or "setup failed" in out:
        verdict = "BUILD"
    elif rc == 0:
        verdict = "SURVIVED"
    else:
        verdict = "RED"
    fails = [l.strip() for l in out.splitlines() if l.strip().startswith("--- FAIL")][:6]
    results.append((mid, verdict, f"{dt:.0f}s {intent}; fails: {', '.join(f.replace('--- FAIL: ', '') for f in fails)}"))
    print(mid, verdict, f"{dt:.0f}s", intent, "|", "; ".join(fails), flush=True)

with open(os.path.join(T, "summary.txt"), "a") as f:
    for r in results:
        f.write("\t".join(r) + "\n")
```

## Cycle 2

- Date: 2026-10-15 (JST 06:02-06:32; the UTC timestamps in the output read 2026-10-14T21:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach, HEAD 9548a8cd (base ef575748; cycle 1 tested a5f5353b)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0. Load average was 2.76 / 4.53 / 15.37 when the runs began; other sessions pushed the 1-minute figure to about 40 during run-test.sh's shell phase
- Redaction: as in cycle 1, the session scratch directory is `$SCRATCH` (a path cut off inside it reads `$SCRATCH…`) and the home directory `~`. The probes ran with `USER=u`. Runs of padding `p` in the disk-image probe are shortened to `p…p`. The wrapper's full log `docs/evidence/verify-2026-10-14-210346.log` is gitignored and not committed
- Trailing spaces at the ends of output lines were stripped; nothing else in the raw output was changed apart from the redaction
- Nothing in the worktree was edited for the tests. Mutants and the probe test lived in `$SCRATCH` and reached the build only through `go test -overlay`. The disk image lived in `$SCRATCH/img`, was mounted at `$SCRATCH/mnt` with `-nobrowse`, and was detached afterwards
- Order: run-test.sh, then the two full Go runs, the focused run, and `-race`, one after another (`runall.sh`). The narrowed mutation runs (C2-3.1) ran 06:04-06:07, during run-test.sh's shell phase; the whole-package survivor runs and the coverage runs ran after `-race`

### C2-1. ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; scripts/ralph-config.sh widened it to full)

Command: `./scripts/run-test.sh > runtest.out 2>&1` (06:03:46-06:15:05, rc 0, 679 s wall, 4,513 lines). Filtered with the same command as cycle 1's section 1 (`sed -n 1,6p` plus the same `grep -nE` and the same dropped step lines), which gives 215 lines. With the line numbers, the durations, `(cached)`, the `ok`/`?` lines, the timestamp, and the evidence-path line removed from both, that excerpt equals cycle 1's section 1 line for line; the only difference is the two lines cycle 1 appended by hand (`rc=0`, `real 676.56`):

```text
$ diff <(norm < cycle1-section1.txt) <(norm < runtest.excerpt)
204,205d203
< real 676.56
< rc=0
```

The lines that carry the verdict:

```text
# Verification run
- Timestamp: 2026-10-14T21:03:46Z
- Mode: test
- Requested scope: changed

$ grep -c '^    OK$' runtest.out
41
$ grep -cE '^==> tests/' runtest.out
41
$ grep -nE '^\s*FAIL: [1-9]|^    FAIL|^FAIL' runtest.out
(no output)
$ grep -n 'SKIP: the' runtest.out
3924:  SKIP: the real git older than 2.41 case (git 2.49.0 is 2.41 or later; the stubbed old git covers it)

4496:==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)
4497:==> Language packs selected: golang
4498:==> Running golang verifier
4499:==> Go project root: .
4500:?   	github.com/yoshpy-dev/ralph	[no test files]
4501:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4502:ok  	github.com/yoshpy-dev/ralph/internal/cli	199.412s
4503:ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
4504:ok  	github.com/yoshpy-dev/ralph/internal/insights	1.071s
4505:ok  	github.com/yoshpy-dev/ralph/internal/org	81.407s
4506:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
4507:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4508:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4509:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4511:==> All verifiers passed.
4513:Evidence saved to: docs/evidence/verify-2026-10-14-210346.log
```

### C2-2. Go tests

#### C2-2.1 Per package (`go test -count=1 -v ./internal/...`)

Both runs had `$SCRATCH/canary` first on PATH (C2-2.5). Counted with cycle 1's `count.awk` ("top" counts top-level `--- PASS/FAIL/SKIP` lines, "sub" every indented one).

go1.26.0 (06:15:05-06:18:24, rc 0, 199 s):
```text
internal/cli           top PASS 679 FAIL 0 SKIP 0 | sub PASS 601 FAIL 0 SKIP 0 | 198.864s
internal/config        top PASS 37 FAIL 0 SKIP 0 | sub PASS 34 FAIL 0 SKIP 0 | 0.544s
internal/insights      top PASS 36 FAIL 0 SKIP 0 | sub PASS 7 FAIL 0 SKIP 0 | 1.132s
internal/org           top PASS 751 FAIL 0 SKIP 1 | sub PASS 1525 FAIL 0 SKIP 0 | 78.430s
internal/org/driver    top PASS 48 FAIL 0 SKIP 0 | sub PASS 89 FAIL 0 SKIP 0 | 3.310s
internal/org/protocol  top PASS 15 FAIL 0 SKIP 0 | sub PASS 56 FAIL 0 SKIP 0 | 3.266s
internal/scaffold      top PASS 30 FAIL 0 SKIP 2 | sub PASS 0 FAIL 0 SKIP 0 | 2.308s
internal/upgrade       top PASS 114 FAIL 0 SKIP 0 | sub PASS 13 FAIL 0 SKIP 0 | 2.320s
```

go1.25.8 (06:18:24-06:21:50, rc 0, 206 s):
```text
internal/cli           top PASS 679 FAIL 0 SKIP 0 | sub PASS 601 FAIL 0 SKIP 0 | 205.080s
internal/config        top PASS 37 FAIL 0 SKIP 0 | sub PASS 34 FAIL 0 SKIP 0 | 5.435s
internal/insights      top PASS 36 FAIL 0 SKIP 0 | sub PASS 7 FAIL 0 SKIP 0 | 5.118s
internal/org           top PASS 751 FAIL 0 SKIP 1 | sub PASS 1525 FAIL 0 SKIP 0 | 85.732s
internal/org/driver    top PASS 48 FAIL 0 SKIP 0 | sub PASS 89 FAIL 0 SKIP 0 | 3.818s
internal/org/protocol  top PASS 15 FAIL 0 SKIP 0 | sub PASS 56 FAIL 0 SKIP 0 | 3.637s
internal/scaffold      top PASS 30 FAIL 0 SKIP 2 | sub PASS 0 FAIL 0 SKIP 0 | 4.845s
internal/upgrade       top PASS 114 FAIL 0 SKIP 0 | sub PASS 13 FAIL 0 SKIP 0 | 5.914s
```

- The 4,038 result lines (name and verdict, durations removed, sorted) are identical across the two versions: `diff names126.txt names125.txt` is empty
- SKIP lines (both versions): `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` (org), `TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS` (scaffold), as in cycle 1
- Tests that were flaky before (tester memory):

```text
go126:
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.80s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.66s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.59s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.31s)
go125:
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.80s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.35s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.69s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.64s)
```

#### C2-2.2 Tests added since cycle 1

```text
$ git diff a5f5353b..HEAD -- '*_test.go' | grep -E '^[+-]func Test'
+func TestOrgDirector_NotRestoredErrorNamesTheStatusCommand(t *testing.T) {
+func TestDirectorAttach_AReplaceCutOffPartwayIsRestored(t *testing.T) {
+func TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo(t *testing.T) {
+func TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored(t *testing.T) {
+func TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored(t *testing.T) {
+func TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration(t *testing.T) {
```

The sorted result lines of cycle 1's go1.25.8 run against this one (go1.25.8): ten added, none removed.

```text
$ diff <(sort cycle1/names125.txt) <(sort names125.txt)
258a259,260
>     --- PASS: TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/attach_into_a_missing_file
>     --- PASS: TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach
1029a1032,1033
>     --- PASS: TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/with_--state-dir
>     --- PASS: TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/without_--state-dir
2652a2657,2659
> --- PASS: TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored
> --- PASS: TestDirectorAttach_AReplaceCutOffPartwayIsRestored
> --- PASS: TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo
2670a2678
> --- PASS: TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration
2675a2684
> --- PASS: TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored
3063a3073
> --- PASS: TestOrgDirector_NotRestoredErrorNamesTheStatusCommand
```

#### C2-2.3 Focused run (go1.26.0)

```text
$ go test -count=1 -run 'Director|NotRestored' -v ./internal/org/ ./internal/cli/    (06:21:50-06:22:01, rc 0)
internal/org           top PASS 37 FAIL 0 SKIP 0 | sub PASS 44 FAIL 0 SKIP 0 | 1.165s
internal/cli           top PASS 28 FAIL 0 SKIP 0 | sub PASS 43 FAIL 0 SKIP 0 | 9.497s
$ grep -E '^--- PASS' focused.out | grep -E 'CutOff|CutBack|CloseFailure|ADetachedForAnother|NotRestored'
--- PASS: TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s)
--- PASS: TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s)
--- PASS: TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s)
--- PASS: TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
--- PASS: TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration (0.00s)
--- PASS: TestOrgDirector_NotRestoredErrorNamesTheStatusCommand (0.00s)
```

#### C2-2.4 Race detector (go1.26.0)

```text
$ go test -count=1 -race ./internal/org/ ./internal/cli/ -run 'Director'    (06:22:01-06:22:18, rc 0)
ok  	github.com/yoshpy-dev/ralph/internal/org	2.958s
ok  	github.com/yoshpy-dev/ralph/internal/cli	10.841s
DATA RACE lines: 0
$ go test -count=1 -race -run 'Director' -list 'Director' ./internal/org/ ./internal/cli/ | grep -c '^Test'
65
```

#### C2-2.5 AC9: no real osascript or herdr, no director.jsonl in the repo's ledger

Every Go run in C2-2.1 to C2-2.4, every mutation run of C2-3, the probe test, and the coverage runs had `$SCRATCH/canary` first on PATH. The canary `osascript` logs its call and exits 0; the canary `herdr` logs its call, passes `--version` to the real `/opt/homebrew/bin/herdr`, and exits 97 otherwise. Sensitivity, run by hand through the same PATH before the runs:

```text
$ $SCRATCH/canary/osascript -e x; PATH=$SCRATCH/canary:$PATH herdr pane list; echo "herdr rc=$?"; cat canary.log
herdr rc=97
osascript ppid=53347 args=-e x
/bin/zsh
herdr ppid=53347 args=pane list
/bin/zsh
(the log was emptied before the runs)
```

After the runs:

```text
canary log lines after go1.26.0: 0
canary log lines after every run (go1.25.8, focused, -race, mutation, probe test, coverage): 0
$ ls ~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach/.harness/state/org
both: No such file or directory (os error 2)
$ find ~/MyDev/github.com/yoshpy-dev/ralph -name director.jsonl
(no output)
$ git status --porcelain   (worktree and main checkout)
(no output, both)
```

#### C2-2.6 Coverage (go1.25.8, each package's own tests, 06:27:07-06:31:43)

```text
$ GOTOOLCHAIN=go1.25.8 go test -count=1 -coverprofile=cover-org.out ./internal/org/
ok  	github.com/yoshpy-dev/ralph/internal/org	79.117s	coverage: 94.6% of statements
$ GOTOOLCHAIN=go1.25.8 go test -count=1 -coverprofile=cover-cli.out ./internal/cli/
ok  	github.com/yoshpy-dev/ralph/internal/cli	191.925s	coverage: 87.3% of statements

$ go tool cover -func=cover-org.out | grep org/director.go   (functions not at 100.0%)
github.com/yoshpy-dev/ralph/internal/org/director.go:359:		appendLocked				95.0%
(the other 19 functions of director.go, restoreLocked and directorAppendFailed included: 100.0%)
$ go tool cover -func=cover-cli.out | grep cli/org_director.go   (functions not at 100.0%)
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:63:			newOrgDirectorAttachCmd			96.6%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:325:			orgDirectorEnabledFile			85.7%
github.com/yoshpy-dev/ralph/internal/cli/org_director.go:348:			orgDirectorCaller			75.0%
(the other 12 functions, notRestoredError included: 100.0%)

statements per file (blocks not reached):
internal/org/director.go 145/146 99.3% [('366.17,368.4', 1)]
internal/cli/org_director.go 138/141 97.9% [('121.5,121.36', 1), ('335.2,335.23', 1), ('350.16,352.3', 1)]
```

`director.go:366-368` is the marshal failure in `appendLocked`. `org_director.go:121` is attach's `return rt.notRestoredError(err)`; `:335` and `:350-352` are the `filepath.Abs` and `os.Getwd` failures (as in cycle 1). detach's `return rt.notRestoredError(err)` (`:183`) is reached, by refusals such as "no director is attached".

### C2-3. Mutation (go1.26.0, `go test -overlay`)

#### C2-3.1 mutate2.py (narrowed runs: `-run Director` for org, `-run OrgDirector` for cli)

```text
X0 SURVIVED 2s control: unchanged director.go through the overlay |
X0c SURVIVED 6s control: unchanged org_director.go through the overlay |
R1 RED 4s skip the truncate on a write failure | TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s); TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/attach_into_a_missing_file (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach (0.00s)
R2 RED 5s truncate to 0 instead of len(raw) (both branches) | TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s); TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach (0.00s); TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R2w RED 5s truncate to 0 instead of len(raw) on a write failure only | TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s); TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach (0.00s)
R2c RED 4s truncate to 0 instead of len(raw) on a close failure only | TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R3 RED 4s return the restored error (no sentinel) when the truncate fails | TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s)
R3b RED 5s same text, but the sentinel is not wrapped (%v) | TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s)
R4 RED 4s treat a close failure as success | TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R5 RED 4s skip the truncate on a close failure (error still returned) | TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R6 RED 4s production truncate is a no-op that reports success | TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/attach_into_a_missing_file (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach (0.00s); TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R7 RED 4s directorAppendFailed says nothing was recorded even when not restored | TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s)
R8 RED 4s off by one: keep one byte of the failed write | TestDirectorAttach_AReplaceCutOffPartwayIsRestored (0.00s); TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/attach_into_a_missing_file (0.00s); TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored/detach (0.00s); TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored (0.00s)
R11 SURVIVED 4s restore keeps the newline added before a torn last line |
D5 RED 4s cycle-1 survivor: a detached of any id ends the current registration | TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration (0.00s)
H1 RED 8s drop the not-restored hint in notRestoredError | TestOrgDirector_NotRestoredErrorNamesTheStatusCommand (0.00s); TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/without_--state-dir (0.00s); TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/with_--state-dir (0.00s)
H2 RED 6s drop the --state-dir part of the hint | TestOrgDirector_NotRestoredErrorNamesTheStatusCommand (0.00s); TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/with_--state-dir (0.00s)
H3 RED 8s add the hint to every store error | TestOrgDirector_NotRestoredErrorNamesTheStatusCommand (0.00s); TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/without_--state-dir (0.00s); TestOrgDirector_NotRestoredErrorNamesTheStatusCommand/with_--state-dir (0.00s)
H4 SURVIVED 9s attach RunE returns the store error without the helper |
H5 SURVIVED 8s detach RunE returns the store error without the helper |
```

"SURVIVED" on X0 and X0c means the control passed. Every old string was found exactly once (the harness skips a mutant otherwise; none was skipped).

#### C2-3.2 The survivors against the whole package (`--wide`: `-run .`)

```text
R11-wide SURVIVED 84s restore keeps the newline added before a torn last line |      (06:22:19-06:23:43, all of internal/org)
H4-wide SURVIVED 203s attach RunE returns the store error without the helper |       (06:23:43-06:27:07, all of internal/cli; H4 and H5 in parallel)
H5-wide SURVIVED 203s detach RunE returns the store error without the helper |
```

#### C2-3.3 Probe test PT for R11 (overlay only, never committed)

`$SCRATCH/probe-go/probe_director_torn_test.go`, added to `internal/org` through the overlay:

```go
package org

import "testing"

// Probe (cycle-2 /test, never committed): an append that fails after a torn
// last line is cut back to the bytes before it, the newline appendLocked put
// in front of its lines included.
func TestProbeDirector_AFailedAppendAfterATornLineIsCutBackExactly(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	before := directorAttachedLine("d1", "claude") + "\n" + `{"ts":"2026-10-15T00:00:02Z","id":"d2","ev`
	writeDirectorFile(t, s, before)
	var onDisk string
	s.write = directorPartialWrite(t, s, &onDisk)
	if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true}); err == nil {
		t.Fatal("Attach with replace succeeded; want the partial write to fail it")
	}
	if len(onDisk) <= len(before) || onDisk[len(before)] != '\n' {
		t.Fatalf("the partial write = %q, want the old bytes, a newline, and part of the lines", onDisk)
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl = %q, want it cut back to %q", got, before)
	}
}
```

```text
$ go test -count=1 -overlay=head.json -run TestProbeDirector_ -v ./internal/org/   # HEAD
=== RUN   TestProbeDirector_AFailedAppendAfterATornLineIsCutBackExactly
--- PASS: TestProbeDirector_AFailedAppendAfterATornLineIsCutBackExactly (0.00s)
PASS
ok  	github.com/yoshpy-dev/ralph/internal/org	1.100s
rc=0

$ go test -count=1 -overlay=r11.json -run TestProbeDirector_ -v ./internal/org/   # R11
=== RUN   TestProbeDirector_AFailedAppendAfterATornLineIsCutBackExactly
    probe_director_torn_test.go:21: director.jsonl = "{\"ts\":\"2026-10-15T00:00:00Z\",\"id\":\"d1\",\"event\":\"attached\",\"driver\":\"claude\",\"wake\":[\"pull\"],\"user\":\"u\",\"cwd\":\"/w\",\"herdr_pane\":false}\n{\"ts\":\"2026-10-15T00:00:02Z\",\"id\":\"d2\",\"ev\n", want it cut back to "{\"ts\":\"2026-10-15T00:00:00Z\",\"id\":\"d1\",\"event\":\"attached\",\"driver\":\"claude\",\"wake\":[\"pull\"],\"user\":\"u\",\"cwd\":\"/w\",\"herdr_pane\":false}\n{\"ts\":\"2026-10-15T00:00:02Z\",\"id\":\"d2\",\"ev"
--- FAIL: TestProbeDirector_AFailedAppendAfterATornLineIsCutBackExactly (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.086s
FAIL
rc=1
```

After the runs, `ls internal/org/probe*` in the worktree matched nothing.

#### C2-3.4 mutate2.py

```python
#!/usr/bin/env python3
"""Cycle-2 overlay mutants for org-director-attach (restore path + CLI hint). Never edits the worktree."""
import json, os, subprocess, sys, time

W = "~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach"
T = "$SCRATCH/mut"
os.makedirs(T, exist_ok=True)
ORG = "internal/org/director.go"
CLI = "internal/cli/org_director.go"
ORG_RUN = "Director"
CLI_RUN = "OrgDirector"
WB = 'return s.restoreLocked(len(raw), fmt.Errorf("append to %s: %w", s.path, err))'
CB = 'return s.restoreLocked(len(raw), fmt.Errorf("close %s: %w", s.path, err))'

# id, package, file, [(old, new), ...], run-regex, intent
M = [
    ("X0", "org", ORG, [], ORG_RUN, "control: unchanged director.go through the overlay"),
    ("X0c", "cli", CLI, [], CLI_RUN, "control: unchanged org_director.go through the overlay"),
    ("R1", "org", ORG, [(WB, 'return fmt.Errorf("append to %s: %w", s.path, err)')], ORG_RUN,
     "skip the truncate on a write failure"),
    ("R2", "org", ORG, [("truncate(s.path, int64(size))", "truncate(s.path, 0)")], ORG_RUN,
     "truncate to 0 instead of len(raw) (both branches)"),
    ("R2w", "org", ORG, [(WB, WB.replace("len(raw)", "0"))], ORG_RUN,
     "truncate to 0 instead of len(raw) on a write failure only"),
    ("R2c", "org", ORG, [(CB, CB.replace("len(raw)", "0"))], ORG_RUN,
     "truncate to 0 instead of len(raw) on a close failure only"),
    ("R3", "org", ORG, [('return fmt.Errorf("%w, and %w (%w), so it may hold part of the write", failed, ErrDirectorWriteNotRestored, err)',
                         "_ = err\n\t\treturn failed")], ORG_RUN,
     "return the restored error (no sentinel) when the truncate fails"),
    ("R3b", "org", ORG, [('"%w, and %w (%w), so it may hold part of the write"', '"%w, and %v (%w), so it may hold part of the write"')], ORG_RUN,
     "same text, but the sentinel is not wrapped (%v)"),
    ("R4", "org", ORG, [("\tif err := f.Close(); err != nil {\n\t\t" + CB + "\n\t}", "\t_ = f.Close()")], ORG_RUN,
     "treat a close failure as success"),
    ("R5", "org", ORG, [(CB, 'return fmt.Errorf("close %s: %w", s.path, err)')], ORG_RUN,
     "skip the truncate on a close failure (error still returned)"),
    ("R6", "org", ORG, [("truncate := os.Truncate", "truncate := func(string, int64) error { return nil }")], ORG_RUN,
     "production truncate is a no-op that reports success"),
    ("R7", "org", ORG, [("\tcase errors.Is(err, ErrDirectorWriteNotRestored):\n\t\treturn fmt.Errorf(\"org: director: %s: %w\", verb, err)\n", "")], ORG_RUN,
     "directorAppendFailed says nothing was recorded even when not restored"),
    ("R8", "org", ORG, [("truncate(s.path, int64(size))", "truncate(s.path, int64(size)+1)")], ORG_RUN,
     "off by one: keep one byte of the failed write"),
    ("R11", "org", ORG, [("\tif len(raw) > 0 && raw[len(raw)-1] != '\\n' {\n\t\tlines = append(lines, '\\n')\n\t}",
                          "\tsize := len(raw)\n\tif len(raw) > 0 && raw[len(raw)-1] != '\\n' {\n\t\tlines = append(lines, '\\n')\n\t\tsize++\n\t}"),
                         (WB, WB.replace("len(raw)", "size")), (CB, CB.replace("len(raw)", "size"))], ORG_RUN,
     "restore keeps the newline added before a torn last line"),
    ("D5", "org", ORG, [("if f.current == nil || ev.ID != f.current.ID {", "if f.current == nil {")], ORG_RUN,
     "cycle-1 survivor: a detached of any id ends the current registration"),
    ("H1", "cli", CLI, [("\tif !errors.Is(err, org.ErrDirectorWriteNotRestored) {\n\t\treturn err\n\t}", "\tif true {\n\t\treturn err\n\t}")], CLI_RUN,
     "drop the not-restored hint in notRestoredError"),
    ("H2", "cli", CLI, [('rt.command("ralph org director status")', '"ralph org director status"')], CLI_RUN,
     "drop the --state-dir part of the hint"),
    ("H3", "cli", CLI, [("\tif !errors.Is(err, org.ErrDirectorWriteNotRestored) {\n\t\treturn err\n\t}", "")], CLI_RUN,
     "add the hint to every store error"),
    ("H4", "cli", CLI, [("\t\t\t\treturn rt.notRestoredError(err)\n\t\t\t}\n\t\t\tline :=", "\t\t\t\treturn err\n\t\t\t}\n\t\t\tline :=")], CLI_RUN,
     "attach RunE returns the store error without the helper"),
    ("H5", "cli", CLI, [("\t\t\t\treturn rt.notRestoredError(err)\n\t\t\t}\n\t\t\tif force {", "\t\t\t\treturn err\n\t\t\t}\n\t\t\tif force {")], CLI_RUN,
     "detach RunE returns the store error without the helper"),
]

args = sys.argv[1:]
wide = "--wide" in args
only = set(a for a in args if a != "--wide")
results = []
for mid, pkg, rel, reps, run, intent in M:
    if only and mid not in only:
        continue
    if wide:
        run = "."
    src = os.path.join(W, rel)
    text = open(src).read()
    bad = None
    for old, new in reps:
        n = text.count(old)
        if n != 1:
            bad = f"old string found {n} times: {old[:60]!r}"
            break
        text = text.replace(old, new)
    if bad:
        results.append((mid, "SKIP", bad))
        print(mid, "SKIP", bad, flush=True)
        continue
    tag = mid + ("-wide" if wide else "")
    mpath = os.path.join(T, f"{mid}.go")
    open(mpath, "w").write(text)
    ov = os.path.join(T, f"{mid}.json")
    json.dump({"Replace": {src: mpath}}, open(ov, "w"))
    cmd = ["go", "test", "-count=1", f"-overlay={ov}", "-run", run, f"./internal/{pkg}/"]
    t0 = time.time()
    try:
        p = subprocess.run(cmd, cwd=W, capture_output=True, text=True, timeout=1200)
        out, rc = p.stdout + p.stderr, p.returncode
    except subprocess.TimeoutExpired:
        out, rc = "TIMEOUT", -1
    dt = time.time() - t0
    open(os.path.join(T, f"{tag}.out"), "w").write(out)
    if "[build failed]" in out or "setup failed" in out:
        verdict = "BUILD"
    elif rc == 0:
        verdict = "SURVIVED"
    else:
        verdict = "RED"
    fails = [l.strip().replace("--- FAIL: ", "") for l in out.splitlines() if l.strip().startswith("--- FAIL")][:8]
    results.append((tag, verdict, f"{dt:.0f}s", intent, ", ".join(fails)))
    print(tag, verdict, f"{dt:.0f}s", intent, "|", "; ".join(fails), flush=True)

with open(os.path.join(T, "summary.txt"), "a") as f:
    for r in results:
        f.write("\t".join(r) + "\n")
```

### C2-4. HEAD binary probe (temp `--state-dir`; PATH holds only git, ralph, and a stub osascript)

`GOTOOLCHAIN=go1.25.8 go build -o $SCRATCH/bin/ralph ./cmd/ralph` at HEAD 9548a8cd (`ralph version`: `ralph dev (unknown unknown)`). Every call ran under `env -i` with `HOME`, `PATH`, and `TMPDIR` in `$SCRATCH/probe`, `USER=u`, from a scratch git repo with no `ralph.toml` (so `[org.director]` has its default `enabled = true`), and every call passed a temp `--state-dir`. herdr, agmsg, and the real osascript did not run (the stub's log has 0 lines).

#### C2-4.1 probe3.sh (normal disk)

```bash
#!/bin/bash
# Cycle-2 binary probe of ralph org director with the HEAD binary.
# Temp --state-dir only; PATH holds git, ralph, and a stub osascript that logs.
set -u
T=$SCRATCH
P=$T/probe; rm -rf "$P"; mkdir -p "$P/home" "$P/pbin" "$P/tmp" "$P/work"
ln -s /opt/homebrew/bin/git "$P/pbin/git"; ln -s "$T/bin/ralph" "$P/pbin/ralph"
printf '#!/bin/sh\nprintf "%%s\\n" "osascript $*" >> "%s/osascript.log"\nexit 0\n' "$P" > "$P/pbin/osascript"; chmod +x "$P/pbin/osascript"; : > "$P/osascript.log"
SD=$P/state
E() { env -i HOME="$P/home" PATH="$P/pbin" TMPDIR="$P/tmp" USER=u GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$@"; }
r() {
  printf '\n$ ralph'; printf ' %q' "$@"; printf '\n'
  E ralph "$@" > "$P/o" 2> "$P/e"; rc=$?
  if [ -s "$P/o" ]; then echo "[stdout]"; cat "$P/o"; fi
  if [ -s "$P/e" ]; then echo "[stderr]"; cat "$P/e"; fi
  echo "[exit $rc]"
}
showf() { echo "--- director.jsonl ($(wc -c < "$1" | tr -d ' ') bytes)"; cat "$1"; }
cd "$P/work" && E git init -q -b main   # no ralph.toml: [org.director] defaults (enabled = true)

echo "## A1. attach, status, status --json"
r org director attach --driver claude --state-dir "$SD"
r org director status --state-dir "$SD"
r org director status --json --state-dir "$SD"
showf "$SD/director.jsonl"

echo; echo "## A2. attach --replace, the replaced id's detach is refused, detach --id of the current one"
r org director attach --driver codex --replace --state-dir "$SD"
cp "$SD/director.jsonl" "$P/before"
r org director detach --id d1 --state-dir "$SD"
cmp "$P/before" "$SD/director.jsonl" && echo "cmp: director.jsonl unchanged by the refused detach"
r org director detach --id d2 --state-dir "$SD"
r org director status --state-dir "$SD"
showf "$SD/director.jsonl"

echo; echo "## A3. director.jsonl read-only (mode 0444): attach, attach --replace, detach"
r org director attach --driver claude --state-dir "$SD"
chmod 0444 "$SD/director.jsonl"; ls -l "$SD/director.jsonl" | awk '{print $1}'
cp "$SD/director.jsonl" "$P/before"
r org director attach --driver codex --replace --state-dir "$SD"
r org director detach --id d3 --state-dir "$SD"
r org director detach --force --state-dir "$SD"
cmp "$P/before" "$SD/director.jsonl" && echo "cmp: director.jsonl unchanged by the three failed appends"
chmod 0644 "$SD/director.jsonl"
r org director status --state-dir "$SD"
r org director detach --id d3 --state-dir "$SD"
echo "--- osascript.log ($(wc -l < "$P/osascript.log" | tr -d ' ') lines)"; cat "$P/osascript.log"
```

Output (06:07, rc 0):

```text
## A1. attach, status, status --json

$ ralph org director attach --driver claude --state-dir $SCRATCH/probe/state
[stdout]
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1 --state-dir '$SCRATCH/probe/state'
[stderr]
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/state (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/probe/state'
[exit 0]

$ ralph org director status --state-dir $SCRATCH/probe/state
[stdout]
director: attached d1 (driver=claude, attached at 2026-10-14T21:07:00Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-14T21:07:00Z
user: u
cwd: $SCRATCH/probe/work
herdr_pane: no
[exit 0]

$ ralph org director status --json --state-dir $SCRATCH/probe/state
[stdout]
{
  "enabled": true,
  "state": "attached",
  "registration": {
    "id": "d1",
    "driver": "claude",
    "wake": [
      "pull"
    ],
    "attached_at": "2026-10-14T21:07:00Z",
    "user": "u",
    "cwd": "$SCRATCH/probe/work",
    "herdr_pane": false
  },
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]
--- director.jsonl (278 bytes)
{"ts":"2026-10-14T21:07:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}

## A2. attach --replace, the replaced id's detach is refused, detach --id of the current one

$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/probe/state
[stdout]
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-14T21:07:00Z)
detach it with: ralph org director detach --id d2 --state-dir '$SCRATCH/probe/state'
[stderr]
warning: director attach: d2 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/state (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/probe/state'
[exit 0]

$ ralph org director detach --id d1 --state-dir $SCRATCH/probe/state
[stderr]
org: director: detach d1: the id is not the current registration's: the current registration is d2 (driver codex, attached 2026-10-14T21:07:01Z), so the session that attached d1 was replaced (or d1 has already been detached) and does not need to detach
[exit 1]
cmp: director.jsonl unchanged by the refused detach

$ ralph org director detach --id d2 --state-dir $SCRATCH/probe/state
[stdout]
detached d2
[exit 0]

$ ralph org director status --state-dir $SCRATCH/probe/state
[stdout]
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
enabled: true
[exit 0]
--- director.jsonl (1043 bytes)
{"ts":"2026-10-14T21:07:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}
{"ts":"2026-10-14T21:07:01Z","id":"d1","event":"detached","user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}
{"ts":"2026-10-14T21:07:01Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}
{"ts":"2026-10-14T21:07:01Z","id":"d2","event":"detached","user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}

## A3. director.jsonl read-only (mode 0444): attach, attach --replace, detach

$ ralph org director attach --driver claude --state-dir $SCRATCH/probe/state
[stdout]
attached d3 (driver=claude)
detach it with: ralph org director detach --id d3 --state-dir '$SCRATCH/probe/state'
[stderr]
warning: director attach: d3 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/state (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/probe/state'
[exit 0]
-r--r--r--@

$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/probe/state
[stderr]
org: director: attach: nothing was recorded, so d3 (driver claude, attached 2026-10-14T21:07:01Z) is still the current registration: open $SCRATCH/probe/state/director.jsonl: open $SCRATCH/probe/state/director.jsonl: permission denied
[exit 1]

$ ralph org director detach --id d3 --state-dir $SCRATCH/probe/state
[stderr]
org: director: detach d3: nothing was recorded, so d3 (driver claude, attached 2026-10-14T21:07:01Z) is still the current registration: open $SCRATCH/probe/state/director.jsonl: open $SCRATCH/probe/state/director.jsonl: permission denied
[exit 1]

$ ralph org director detach --force --state-dir $SCRATCH/probe/state
[stderr]
org: director: force detach: nothing was recorded, so d3 (driver claude, attached 2026-10-14T21:07:01Z) is still the current registration: open $SCRATCH/probe/state/director.jsonl: open $SCRATCH/probe/state/director.jsonl: permission denied
[exit 1]
cmp: director.jsonl unchanged by the three failed appends

$ ralph org director status --state-dir $SCRATCH/probe/state
[stdout]
director: attached d3 (driver=claude, attached at 2026-10-14T21:07:01Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d3
driver: claude
wake: pull
attached_at: 2026-10-14T21:07:01Z
user: u
cwd: $SCRATCH/probe/work
herdr_pane: no
[exit 0]

$ ralph org director detach --id d3 --state-dir $SCRATCH/probe/state
[stdout]
detached d3
[exit 0]
--- osascript.log (0 lines)
```

#### C2-4.2 The disk image

```text
$ hdiutil create -size 1m -fs HFS+ -layout NONE -volname d8a1c2 $SCRATCH/img/v.dmg
created: $SCRATCH/img/v.dmg
$ hdiutil attach -nobrowse -noautoopen -mountpoint $SCRATCH/mnt $SCRATCH/img/v.dmg
/dev/disk14         	                               	$SCRATCH/mnt
$ diskutil info $SCRATCH/mnt | grep -iE 'block size|File System Personality|Volume Free'
   File System Personality:   HFS+
   Device Block Size:         512 Bytes
   Volume Free Space:         905.2 KB (905216 Bytes) (exactly 1768 512-Byte-Units) (86.3%)
   Allocation Block Size:     4096 Bytes
... (probe4.sh and probe5.sh) ...
$ hdiutil detach $SCRATCH/mnt
"disk14" ejected.
$ mount | grep -c "$SCRATCH/mnt"
0
```

#### C2-4.3 probe4.sh (a real ENOSPC; then the truncate refused with `chflags uappnd`)

```bash
#!/bin/bash
# Cycle-2 probe B: a real ENOSPC on a full 1 MB HFS+ image mounted in scratch.
set -u
T=$SCRATCH
P=$T/probe; M=$T/mnt; SD=$M/state; SDN=$P/state-measure
E() { env -i HOME="$P/home" PATH="$P/pbin" TMPDIR="$P/tmp" USER=u GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$@"; }
r() {
  printf '\n$ ralph'; printf ' %q' "$@"; printf '\n'
  E ralph "$@" > "$P/o" 2> "$P/e"; rc=$?
  if [ -s "$P/o" ]; then echo "[stdout]"; cat "$P/o"; fi
  if [ -s "$P/e" ]; then echo "[stderr]"; cat "$P/e"; fi
  echo "[exit $rc]"
}
sz() { wc -c < "$1" | tr -d ' '; }
cd "$P/work"
echo "## B0. measure the two lines of an attach --replace on the normal disk (same cwd and USER)"
rm -rf "$SDN"
E ralph org director attach --driver claude --state-dir "$SDN" >/dev/null 2>&1
s0=$(sz "$SDN/director.jsonl")
E ralph org director attach --driver codex --replace --state-dir "$SDN" >/dev/null 2>&1
s1=$(sz "$SDN/director.jsonl")
LD=$(sed -n 2p "$SDN/director.jsonl" | wc -c | tr -d ' '); LA=$(sed -n 3p "$SDN/director.jsonl" | wc -c | tr -d ' ')
echo "replace appended $((s1-s0)) bytes: detached line $LD, attached line $LA (with newlines)"

echo; echo "## B1. ledger on the image; d1's attached line padded so the file ends $((LD + LA/2)) bytes before a 4096-byte block boundary"
rm -rf "$SD"
r org director attach --driver claude --state-dir "$SD"
python3 - "$SD/director.jsonl" $((4096 - LD - LA/2)) <<'PY'
import json, sys
path, target = sys.argv[1], int(sys.argv[2])
ev = json.loads(open(path).read())
base = len(json.dumps(ev, separators=(",", ":"))) + 1
ev["cwd"] = ev["cwd"] + "/" + "p" * (target - base - 1)
line = json.dumps(ev, separators=(",", ":")) + "\n"
assert len(line) == target, (len(line), target)
open(path, "w").write(line)
PY
echo "director.jsonl: $(sz "$SD/director.jsonl") bytes"
r org director status --state-dir "$SD"
echo "--- fill the volume"
dd if=/dev/zero of="$M/fill" bs=4096 2>&1 | tail -1
dd if=/dev/zero of="$M/fill2" bs=512 2>&1 | tail -1
df -k "$M" | tail -1 | awk '{print "df: used " $3 " KiB, available " $4 " KiB"}'
cp "$SD/director.jsonl" "$P/before"

echo; echo "## B2. attach --replace on the full volume (the truncate can work)"
r org director attach --driver codex --replace --state-dir "$SD"
echo "director.jsonl after: $(sz "$SD/director.jsonl") bytes (before: $(sz "$P/before"))"
cmp "$P/before" "$SD/director.jsonl" && echo "cmp: director.jsonl is byte-identical to before the attempt"
r org director status --state-dir "$SD"

echo; echo "## B3. the same with director.jsonl append-only (chflags uappnd): the write fails on the full volume and the truncate is refused"
chflags uappnd "$SD/director.jsonl"; ls -lO "$SD/director.jsonl" | awk '{print $1, $5}'
r org director attach --driver codex --replace --state-dir "$SD"
echo "director.jsonl after: $(sz "$SD/director.jsonl") bytes (before: $(sz "$P/before"))"
cmp "$P/before" "$SD/director.jsonl" && echo "cmp: director.jsonl is byte-identical to before the attempt"
echo "--- the bytes past the old end:"; tail -c +$(( $(sz "$P/before") + 1 )) "$SD/director.jsonl" | sed "s/pppp*/p…p/"; echo
r org director status --state-dir "$SD"
r org director detach --id d1 --state-dir "$SD"
echo "director.jsonl after the detach: $(sz "$SD/director.jsonl") bytes"

echo; echo "## B4. cleanup"
chflags nouappnd "$SD/director.jsonl"
rm -f "$M/fill" "$M/fill2"
df -k "$M" | tail -1 | awk '{print "df: used " $3 " KiB, available " $4 " KiB"}'
```

Output (06:08, rc 0):

```text
## B0. measure the two lines of an attach --replace on the normal disk (same cwd and USER)
replace appended 521 bytes: detached line 244, attached line 277 (with newlines)

## B1. ledger on the image; d1's attached line padded so the file ends 382 bytes before a 4096-byte block boundary

$ ralph org director attach --driver claude --state-dir $SCRATCH/mnt/state
[stdout]
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1 --state-dir '$SCRATCH/mnt/state'
[stderr]
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/mnt/state (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/mnt/state'
[exit 0]
director.jsonl: 3714 bytes

$ ralph org director status --state-dir $SCRATCH/mnt/state
[stdout]
director: attached d1 (driver=claude, attached at 2026-10-14T21:08:03Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-14T21:08:03Z
user: u
cwd: $SCRATCH/probe/work/p…p
herdr_pane: no
[exit 0]
--- fill the volume
901120 bytes transferred in 0.000820 secs (1098926829 bytes/sec)
dd: $SCRATCH/mnt/fill2: No space left on device
df: used 1024 KiB, available 0 KiB

## B2. attach --replace on the full volume (the truncate can work)

$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/mnt/state
[stderr]
org: director: attach: nothing was recorded, so d1 (driver claude, attached 2026-10-14T21:08:03Z) is still the current registration: append to $SCRATCH/mnt/state/director.jsonl: write $SCRATCH/mnt/state/director.jsonl: no space left on device
[exit 1]
director.jsonl after: 3714 bytes (before: 3714)
cmp: director.jsonl is byte-identical to before the attempt

$ ralph org director status --state-dir $SCRATCH/mnt/state
[stdout]
director: attached d1 (driver=claude, attached at 2026-10-14T21:08:03Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-14T21:08:03Z
user: u
cwd: $SCRATCH/probe/work/p…p
herdr_pane: no
[exit 0]

## B3. the same with director.jsonl append-only (chflags uappnd): the write fails on the full volume and the truncate is refused
-rw-r--r--@ uappnd

$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/mnt/state
[stderr]
org: director: attach: append to $SCRATCH/mnt/state/director.jsonl: write $SCRATCH/mnt/state/director.jsonl: no space left on device, and director.jsonl could not be restored to its size before the append (truncate $SCRATCH/mnt/state/director.jsonl: operation not permitted), so it may hold part of the write: read the registration with ralph org director status --state-dir '$SCRATCH/mnt/state'
[exit 1]
director.jsonl after: 4096 bytes (before: 3714)
cmp: EOF on $SCRATCH/probe/before
--- the bytes past the old end:
{"ts":"2026-10-14T21:08:03Z","id":"d1","event":"detached","user":"u","cwd":"$SCRATCH/probe/work","herdr_pane":false}
{"ts":"2026-10-14T21:08:03Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"user":"u","cwd":"$SCRATCH…

$ ralph org director status --state-dir $SCRATCH/mnt/state
[stdout]
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
enabled: true
[stderr]
warning: 1 unreadable director line(s) skipped in $SCRATCH/mnt/state/director.jsonl
[exit 0]

$ ralph org director detach --id d1 --state-dir $SCRATCH/mnt/state
[stderr]
org: director: detach d1: no director is attached
[exit 1]
director.jsonl after the detach: 4096 bytes

## B4. cleanup
df: used 144 KiB, available 880 KiB
```

#### C2-4.4 probe5.sh (detach's two endings on the full image)

```bash
#!/bin/bash
# Cycle-2 probe B5: detach's not-restored path on the full image (append-only director.jsonl).
set -u
T=$SCRATCH
P=$T/probe; M=$T/mnt; SD=$M/state
E() { env -i HOME="$P/home" PATH="$P/pbin" TMPDIR="$P/tmp" USER=u GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null "$@"; }
r() {
  printf '\n$ ralph'; printf ' %q' "$@"; printf '\n'
  E ralph "$@" > "$P/o" 2> "$P/e"; rc=$?
  if [ -s "$P/o" ]; then echo "[stdout]"; cat "$P/o"; fi
  if [ -s "$P/e" ]; then echo "[stderr]"; cat "$P/e"; fi
  echo "[exit $rc]"
}
sz() { wc -c < "$1" | tr -d ' '; }
cd "$P/work"
echo "## B5. detach --id d1 and detach --force: the detached line (244 bytes) is cut at the block boundary, 122 bytes in"
chflags nouappnd "$SD/director.jsonl" 2>/dev/null
python3 - "$SD/director.jsonl" "$P/before" $((4096 - 122)) <<'PY'
import json, sys
path, src, target = sys.argv[1], sys.argv[2], int(sys.argv[3])
ev = json.loads(open(src).read())
ev["cwd"] = ev["cwd"].rstrip("p").rstrip("/")
base = len(json.dumps(ev, separators=(",", ":"))) + 1
ev["cwd"] = ev["cwd"] + "/" + "p" * (target - base - 1)
line = json.dumps(ev, separators=(",", ":")) + "\n"
assert len(line) == target, (len(line), target)
open(path, "w").write(line)
PY
echo "director.jsonl: $(sz "$SD/director.jsonl") bytes"
cp "$SD/director.jsonl" "$P/before5"
dd if=/dev/zero of="$M/fill" bs=4096 2>&1 | tail -1
dd if=/dev/zero of="$M/fill2" bs=512 2>&1 | tail -1
df -k "$M" | tail -1 | awk '{print "df: used " $3 " KiB, available " $4 " KiB"}'

echo; echo "--- B5a. truncate allowed"
r org director detach --id d1 --state-dir "$SD"
cmp "$P/before5" "$SD/director.jsonl" && echo "cmp: director.jsonl is byte-identical to before the attempt"
r org director detach --force --state-dir "$SD"
cmp "$P/before5" "$SD/director.jsonl" && echo "cmp: director.jsonl is byte-identical to before the attempt"

echo; echo "--- B5b. append-only (truncate refused)"
chflags uappnd "$SD/director.jsonl"
r org director detach --id d1 --state-dir "$SD"
echo "director.jsonl after: $(sz "$SD/director.jsonl") bytes (before: $(sz "$P/before5"))"
echo "--- the bytes past the old end:"; tail -c +$(( $(sz "$P/before5") + 1 )) "$SD/director.jsonl"; echo
r org director status --state-dir "$SD"

echo; echo "## B6. cleanup"
chflags nouappnd "$SD/director.jsonl"
rm -f "$M/fill" "$M/fill2"
df -k "$M" | tail -1 | awk '{print "df: used " $3 " KiB, available " $4 " KiB"}'
```

Output (06:08, rc 0):

```text
## B5. detach --id d1 and detach --force: the detached line (244 bytes) is cut at the block boundary, 122 bytes in
director.jsonl: 3974 bytes
901120 bytes transferred in 0.000534 secs (1687490637 bytes/sec)
dd: $SCRATCH/mnt/fill2: No space left on device
df: used 1024 KiB, available 0 KiB

--- B5a. truncate allowed

$ ralph org director detach --id d1 --state-dir $SCRATCH/mnt/state
[stderr]
org: director: detach d1: nothing was recorded, so d1 (driver claude, attached 2026-10-14T21:08:03Z) is still the current registration: append to $SCRATCH/mnt/state/director.jsonl: write $SCRATCH/mnt/state/director.jsonl: no space left on device
[exit 1]
cmp: director.jsonl is byte-identical to before the attempt

$ ralph org director detach --force --state-dir $SCRATCH/mnt/state
[stderr]
org: director: force detach: nothing was recorded, so d1 (driver claude, attached 2026-10-14T21:08:03Z) is still the current registration: append to $SCRATCH/mnt/state/director.jsonl: write $SCRATCH/mnt/state/director.jsonl: no space left on device
[exit 1]
cmp: director.jsonl is byte-identical to before the attempt

--- B5b. append-only (truncate refused)

$ ralph org director detach --id d1 --state-dir $SCRATCH/mnt/state
[stderr]
org: director: detach d1: append to $SCRATCH/mnt/state/director.jsonl: write $SCRATCH/mnt/state/director.jsonl: no space left on device, and director.jsonl could not be restored to its size before the append (truncate $SCRATCH/mnt/state/director.jsonl: operation not permitted), so it may hold part of the write: read the registration with ralph org director status --state-dir '$SCRATCH/mnt/state'
[exit 1]
director.jsonl after: 4096 bytes (before: 3974)
--- the bytes past the old end:
{"ts":"2026-10-14T21:08:35Z","id":"d1","event":"detached","user":"u","cwd":"$SCRATCH…

$ ralph org director status --state-dir $SCRATCH/mnt/state
[stdout]
director: attached d1 (driver=claude, attached at 2026-10-14T21:08:03Z, by u) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-14T21:08:03Z
user: u
cwd: $SCRATCH/probe/work/p…p
herdr_pane: no
[stderr]
warning: 1 unreadable director line(s) skipped in $SCRATCH/mnt/state/director.jsonl
[exit 0]

## B6. cleanup
df: used 144 KiB, available 880 KiB
```
