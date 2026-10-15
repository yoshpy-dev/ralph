# Test evidence: org-director-start (cycle 1)

- Date: 2026-10-15 (JST 18:23-19:25; the UTC timestamps in the output read 2026-10-15T09:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD b6c86188 (base 9cbf7080, 17 commits)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0, herdr 0.7.5 (only in the isolated server of section 4). Load average 8-12 during the runs (other sessions)
- Redaction: the session scratch directory is `$SCRATCH`, the home directory `~`, the Go test temp directory `$TMPDIR/`, and the stub probe's run directory `<run>`. The wrapper's full log `docs/evidence/verify-2026-10-15-092359.log` and the raw copy `docs/evidence/test-2026-10-15-org-director-start.log` are gitignored and not committed
- Trailing blanks at the ends of output lines were stripped; nothing else in the raw output was changed apart from the redaction
- Nothing in the worktree was edited for the tests. Mutants and probe tests lived in `$SCRATCH` and reached the build only through `go test -overlay`

## 1. ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; templates/base/ralph.toml widened it to full)

Command: `./scripts/run-test.sh > runtest.out 2>&1` (rc 0, 819 s wall). Filtered from the 4,514-line raw output with:
`sed -n 1,5p; grep -nE '^==> tests/|tests: [0-9]+ passed|Results?: [0-9]+/[0-9]+|^\s*PASS: [0-9]+( / [0-9]+)?$|^\s*FAIL: [0-9]+$|^\s*SKIP|passed, [0-9]+ failed|^    OK$|^    FAIL|^ok |^\? |^FAIL|Evidence saved|All verifiers|^==> Language scope'`.
Numbers before ':' are line numbers in the raw output. 41 suites, 41 "OK", every "FAIL:" line is 0 (25 of them, no other FAIL line). 4,143 lines start with PASS. The one real SKIP is tests/test-secret-scan-branch.sh's (git 2.49.0).

```text
# Verification run
- Timestamp: 2026-10-15T09:23:59Z
- Mode: test
- Requested scope: changed

7:==> tests/test-agent-models.sh
60:  PASS: 47 / 47
61:  FAIL: 0
62:    OK
63:==> tests/test-agent-phase-boundaries.sh
110:  PASS: 44 / 44
111:  FAIL: 0
112:    OK
113:==> tests/test-archive-plan.sh
140:archive-plan tests: 22 passed, 0 failed, 22 total
141:    OK
142:==> tests/test-branch-name.sh
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
562:ensure-pr-ready tests: 7 passed, 0 failed, 7 total
563:    OK
564:==> tests/test-ensure-pr-title-prefix.sh
580:ensure-pr-title-prefix tests: 13 passed, 0 failed, 13 total
581:    OK
582:==> tests/test-gc-artifacts.sh
621:gc-artifacts tests: 11 passed, 0 failed, 11 total
622:    OK
623:==> tests/test-hook-wiring.sh
695:  PASS: 68
696:  FAIL: 0
697:    OK
698:==> tests/test-insights-append.sh
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
964:new-feature-plan tests: 21 passed, 0 failed, 21 total
965:    OK
966:==> tests/test-no-loop-references.sh
968:    OK
969:==> tests/test-plan-visual.sh
1078:plan-visual tests: 103 passed, 0 failed, 103 total
1079:    OK
1080:==> tests/test-post-edit-verify.sh
1108:  PASS: 24
1109:  FAIL: 0
1110:    OK
1111:==> tests/test-pr-owner-lookup.sh
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
4496:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
4500:?   	github.com/yoshpy-dev/ralph	[no test files]
4501:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4502:ok  	github.com/yoshpy-dev/ralph/internal/cli	221.484s
4503:ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
4504:ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
4505:ok  	github.com/yoshpy-dev/ralph/internal/org	87.505s
4506:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	1.129s
4507:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4508:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4509:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4511:==> All verifiers passed.
4513:Evidence saved to: docs/evidence/verify-2026-10-15-092359.log
```

The five `(cached)` packages ran with `-count=1` in section 2.

## 2. Go tests

Every Go run below had `$SCRATCH/canary/bin` first on PATH (section 2.4).

### 2.1 Per package, `-count=1 -v`

Counts are from the `--- PASS/FAIL/SKIP` lines (top-level tests, then subtests).

go1.25.8 (`GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile=cover125.out ./internal/...`, rc 0, 255 s wall):

```text
internal/cli	ok	249.965s	top 699 (pass 699 fail 0 skip 0)	sub 639 (pass 639 fail 0 skip 0)
internal/config	ok	1.975s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	1.614s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	114.933s	top 800 (pass 799 fail 0 skip 1)	sub 1682 (pass 1682 fail 0 skip 0)
internal/org/driver	ok	2.066s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	3.770s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	3.373s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	3.469s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.26.0 (`go test -count=1 -v ./internal/...`, rc 0, 220 s wall):

```text
internal/cli	ok	219.211s	top 699 (pass 699 fail 0 skip 0)	sub 639 (pass 639 fail 0 skip 0)
internal/config	ok	0.554s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	2.806s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	83.341s	top 800 (pass 799 fail 0 skip 1)	sub 1682 (pass 1682 fail 0 skip 0)
internal/org/driver	ok	2.729s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	3.998s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	1.452s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	2.344s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

The sorted `<name> <result>` lines of the two runs (4,301 each) are identical. The three SKIPs are `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` (org, skipped on darwin by design) and scaffold's `TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS`.

Earlier flaky tests, both runs (go1.25.8, then go1.26.0):

```text
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.83s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.67s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (4.33s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.49s)
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.79s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.03s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.48s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.48s)
```

### 2.2 Tests this diff added and removed

`git diff 9cbf7080...HEAD -- '*_test.go'`: 67 `func Test` added, none removed. All 67 PASS on both toolchains, with 187 subtests under them.

```text
TestDirector_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration
TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther
TestDirectorAttachAndAttachHeadless_ConcurrentlyOnlyOneRegisters
TestDirectorAttachHeadless_AfterADetachTheOldOneStaysOpen
TestDirectorAttachHeadless_RecordsTheLaunchAndItsEvents
TestDirectorAttachHeadless_RefusesBadInputBeforeReadingAnything
TestDirectorAttachHeadless_RefusesWhileARegistrationIsCurrent
TestDirectorAttachHeadless_TokensAndTheNamesBuiltFromThem
TestDirectorHeadlessWriters_ACutOffWriteIsRestored
TestDirectorMarkForceStopped_OnlyTheStoppedLineCarriesForce
TestDirectorMarkLaunched_OnlyWhileTheRegistrationIsStillCurrent
TestDirectorMarkStopped_ARegistrationNoLongerCurrentGetsStoppedOnly
TestDirectorMarkStopped_DetachedFirstSoAWriteThatCannotBeCutBackStaysRecoverable
TestDirectorMarkStopped_RefusesWithNoOpenHeadlessRegistration
TestDirectorMarkStopped_TheCurrentRegistrationGetsDetachedAndStoppedInOneWrite
TestDirectorRead_AStage8a1FileReadsAsBefore
TestDirectorRead_FoldsHeadlessRegistrations
TestDirectorRecordHerdrIDs_Refusals
TestDirectorRoleInputErr_IsExactAndCaseSensitive
TestOrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop
TestOrgDirectorAttachReplace_AHeadlessDirectorNamesItsStop
TestOrgDirectorDetach_AHeadlessDirectorNamesItsStop
TestOrgDirectorStart_AFailedAgentStartIsCleanedUp
TestOrgDirectorStart_AnAttachedDirectorIsRefusedBeforeTheWatchGate
TestOrgDirectorStart_LaunchesRegistersAndStops
TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
TestOrgDirectorStart_RefusedWithNothingWritten
TestOrgDirectorStart_WatchGate
TestOrgDirectorStatus_ShowsHeadlessDirectors
TestOrgDirectorStop_ACloseThatFailsNamesForce
TestOrgDirectorStop_ADetachedHeadlessDirector
TestOrgDirectorStop_AStartingRegistrationAndABrokenRalphToml
TestOrgDirectorStop_Refusals
TestOrgDirectorStop_RefusedInsideTheDirectorsPaneOrWorkspace
TestOrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced
TestOrgSpawn_DirectorRole_RejectedBeforeAnyManifestWrite
TestOrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback
TestOrgStopAll_AnUnreadableDirectorFileChangesNothing
TestOrgStopAll_NamesTheOpenHeadlessDirectors
TestOrgStopAll_ReportsOpenHeadlessDirectorsAndLeavesThem
TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure
TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders
TestRenderDirectorPrompt_SubstitutesEveryPlaceholder
TestRenderRolePrompt_DirectorIsNotASeatTemplate
TestStartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly
TestStartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop
TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade
TestStartDirector_ARegistrationAfterTheFirstLookIsRefusedUnderTheLock
TestStartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch
TestStartDirector_BoundsTheAgentStart
TestStartDirector_LaunchesInAWorkspaceOfItsOwn
TestStartDirector_PermissionModeOfRoleDirector
TestStartDirector_RefusesBeforeWritingOrCallingHerdr
TestStatusCmd_DirectorLineShowsHeadlessDirectors
TestStatusCmd_DirectorStopCommandRepeatsTheStateDir
TestStopDirector_ACloseThatFailsRecordsNothing
TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce
TestStopDirector_ALabelNotThisLaunchsSendsNothing
TestStopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch
TestStopDirector_ClosesThePaneAndWorkspaceAndRecordsTheStop
TestStopDirector_ForceAfterACloseThatFailsRecordsWhatWasLeftOpen
TestStopDirector_ForceRecordsAStopWhoseLabelsAreAnotherLaunchs
TestStopDirector_ForceWhenAGetTimesOutSendsNothing
TestStopDirector_ForceWithLabelsThatMatchClosesAsWithoutIt
TestStopDirector_Refusals
TestStopDirector_RefusesInsideTheDirectorsWorkspace
TestStopDirector_WhatHerdrNoLongerKnowsCountsAsClosed
```

### 2.3 Race detector (go1.26.0)

`go test -race -count=1 ./internal/org/ ./internal/cli/` (whole packages), rc 0, 445 s wall, 0 `DATA RACE` lines:

```text
ok  	github.com/yoshpy-dev/ralph/internal/org	121.782s
ok  	github.com/yoshpy-dev/ralph/internal/cli	308.007s
```

### 2.4 AC13: no real herdr or osascript, nothing in the repo's ledger

- `$SCRATCH/canary/bin/{herdr,osascript}` log their argv to `$SCRATCH/canary/calls.log` and exit 97. They were first on PATH for both `-count=1` runs, `-race`, all 103 mutation runs (86 mutants and 2 controls filtered, 15 whole-package), the probe tests, and the merged-tree run of section 6. `calls.log` has 0 lines at the end. Typed by hand from the same PATH, both canaries log (PATH lookup reaches them)
- After the runs: `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` and `.../.claude/worktrees/org-director-start/.harness/state/org` do not exist; `find` finds no `director.jsonl` under the main checkout
- During the full-package runs, `pgrep -fl herdr` shows only the CLI tests' stub `herdr` scripts under `$TMPDIR/` (for example `$TMPDIR/TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStartstarted_per.../001/herdr agent start ...`)

### 2.5 Coverage (go1.25.8, each package's own tests)

Package totals: cli 87.7%, config 93.3%, insights 86.1%, org 94.6%, org/driver 93.1%, org/protocol 100.0%, scaffold 75.7%, upgrade 91.2% (`go tool cover -func` total 91.1%).

Per file (statements covered/total, from `cover125.out`):

```text
internal/org/director_start.go 212/228 93.0%
internal/org/director_stop.go 132/138 95.7%
internal/cli/org_director_headless.go 94/98 95.9%
internal/org/director.go 193/194 99.5%
internal/cli/org_director.go 177/180 98.3%
internal/org/spawn.go 460/500 92.0%
internal/org/prompts.go 37/38 97.4%
internal/org/verbs_all.go 106/107 99.1%
internal/org/permissions.go 28/31 90.3%
internal/cli/org.go 544/568 95.8%
internal/cli/status.go 230/239 96.2%
```

Uncovered blocks of the three new files (line: what it is):

```text
director_start.go 59    rand.Read fails (newDirectorToken)
director_start.go 129   AttachHeadless: readRaw fails under the lock
director_start.go 137   AttachHeadless: fold.nextID fails
director_start.go 204   recordHerdrID: readRaw fails
director_start.go 249   MarkLaunched: readRaw fails
director_start.go 322   markStopped: readRaw fails
director_start.go 470   StartDirector: AttachHeadless returns an error other than the refusal (mutant E1)
director_start.go 489   StartDirector: switch case err != nil after MarkLaunched
director_start.go 532   checkDirectorStart: the early Store.Read fails (mutant D4)
director_start.go 541   checkDirectorStart: absPath of the state dir fails
director_start.go 544   checkDirectorStart: absPath of the cwd fails
director_start.go 605   writePrompt: RenderDirectorPrompt fails
director_start.go 634   replacedError: case err != nil
director_start.go 652   abort: MarkStopped fails after a finished close
director_start.go 659   abort: Detach refused (ErrDirectorIDMismatch / ErrDirectorNotAttached)
director_start.go 661   abort: default (Detach fails otherwise)
director_stop.go 169    recordDirectorStop: case err != nil
director_stop.go 187    openHeadlessRegistration: readRaw fails
director_stop.go 239    DirectorCleanupError.Error: CleanupErr wraps ErrDirectorWriteNotRestored
director_stop.go 243    DirectorCleanupError.Error: case e.Err != nil
director_stop.go 372    closeDirectorPane: the C-c send fails
director_stop.go 468    directorShellWord: shellQuote branch
org_director_headless.go 163  startDirector: resolveOrgConfig fails
org_director_headless.go 175  startDirector: resolveModelOrWarn fails
org_director_headless.go 185  startDirector: os.Getwd fails
org_director_headless.go 233  startAttachedError: cur == nil (mutant P2)
```

Function coverage of the new files is in `go tool cover -func`: every function of `director_stop.go` and `org_director_headless.go` is at 80% or more except `directorShellWord` (66.7%); in `director_start.go` the lowest are `newDirectorToken` and `abort` (75.0%), `replacedError` (83.3%), `writePrompt` (85.7%), `checkDirectorStart` (87.5%).

## 3. HEAD binary against a stub herdr

`$SCRATCH/bin/ralph` is HEAD b6c86188 built with go1.25.8. `$SCRATCH/live/bin` holds that ralph (symlink), the CLI tests' stateful stub `herdr` (from `internal/cli/org_test.go`'s `herdrStub`, the same `ORG_STUB_FAIL`, `ORG_STUB_CLOSE_FAIL_IDS`, `ORG_STUB_HERDR_STATE` knobs) plus one knob, `ORG_STUB_AGENT_START_SLEEP` (`exec sleep <n>` in `agent start`), and a logging stub `osascript`. Every command runs under `env -i` with PATH = that dir + `/usr/bin:/bin`, so no real herdr, agmsg, or osascript is reachable and USER is unset.

### 3.1 Script (`$SCRATCH/live/probe.sh`)

```bash
#!/bin/bash
# HEAD-binary checks of ralph org director start/stop against a stub herdr
# (the CLI tests' stateful stub plus ORG_STUB_AGENT_START_SLEEP). env -i, no
# real herdr or osascript on PATH, scratch --state-dir and --config.
set -u
S=$SCRATCH
B=$S/live/bin
R=$S/live/run
rm -rf "$R"
mkdir -p "$R/home" "$R/herdr" "$R/ledger" "$R/cwd"
L=$R/ledger
C=$R/ralph.toml
cat > "$C" <<'EOF'
[org.director]
enabled = true

[org.permissions]
default = "autonomous"

[org.permissions.roles]
director = "guarded"
EOF
cd "$R/cwd" || exit 1
E=(env -i HOME="$R/home" PATH="$B:/usr/bin:/bin" ORG_HERDR_LOG="$R/herdr.log" ORG_STUB_HERDR_STATE="$R/herdr")
red() { sed -e "s#$R#<run>#g"; }
step() { printf '\n### %s\n' "$*"; }
run() { # run [VAR=val ...] -- ralph args...
  local pre=()
  while [ "$1" != "--" ]; do pre+=("$1"); shift; done
  shift
  printf '$ %s ralph %s\n' "${pre[*]+${pre[*]}}" "$*" | red
  "${E[@]}" ${pre[@]+"${pre[@]}"} ralph "$@" > "$R/o" 2> "$R/e"
  local rc=$?
  sed 's/^/  out: /' "$R/o" | red
  sed 's/^/  err: /' "$R/e" | red
  echo "  rc=$rc"
}
events() { printf '  director.jsonl: '; python3 -c '
import json,sys
print(", ".join(e["event"]+" "+e["id"]+("+force" if e.get("force") else "")+(" ws="+e["workspace_id"] if e.get("workspace_id") else "")+(" pane="+e["pane_id"] if e.get("pane_id") else "") for e in map(json.loads, open(sys.argv[1]))))' "$L/director.jsonl"; }
herdrsince() { tail -n +"$(( $1 + 1 ))" "$R/herdr.log" 2>/dev/null | red | sed 's/^/  herdr: /'; }
hl() { wc -l < "$R/herdr.log" 2>/dev/null | tr -d ' ' || echo 0; }
waitev() { # waitev <event> <id>: until director.jsonl has it, up to 10 s
  for _ in $(seq 1 100); do
    python3 -c 'import json,sys; sys.exit(0 if any(e["event"]==sys.argv[2] and e["id"]==sys.argv[3] for e in map(json.loads, open(sys.argv[1]))) else 1)' "$L/director.jsonl" "$1" "$2" 2>/dev/null && return 0
    sleep 0.1
  done; return 1; }

: > "$R/herdr.log"
step "K0 the long-running watch"
run -- org watch --all --detach --state-dir "$L" --config "$C"
sleep 2
run -- status --state-dir "$L"

step "K1 starting is shown while the agent start waits; the agent start times out and start cleans up (no USER in env)"
n=$(hl)
"${E[@]}" ORG_STUB_AGENT_START_SLEEP=6 ralph org director start --driver claude --model haiku --timeout-ms 2500 --state-dir "$L" --config "$C" > "$R/k1.out" 2> "$R/k1.err" &
pid=$!
t0=$(date +%s)
waitev pane_created d1 && echo "  pane_created d1 recorded"
run -- status --state-dir "$L"
run -- org director status --state-dir "$L" --json
wait $pid; rc=$?
echo "  start rc=$rc after $(( $(date +%s) - t0 )) s"
sed 's/^/  out: /' "$R/k1.out" | red; sed 's/^/  err: /' "$R/k1.err" | red
herdrsince "$n"; events
run -- status --state-dir "$L"

step "K2 a start killed during its agent start leaves d2 starting; start and attach refuse naming stop; stop closes what was recorded"
n=$(hl)
"${E[@]}" ORG_STUB_AGENT_START_SLEEP=617 ralph org director start --driver claude --model haiku --timeout-ms 60000 --state-dir "$L" --config "$C" > "$R/k2.out" 2> "$R/k2.err" &
pid=$!
waitev pane_created d2 && echo "  pane_created d2 recorded"; sleep 0.3
kill -9 $pid; wait $pid 2>/dev/null; echo "  start killed (SIGKILL), rc=$?"
pkill -f '^sleep 617$' && echo "  the stub's orphaned agent-start sleep killed"
herdrsince "$n"; events
run -- status --state-dir "$L"
n=$(hl)
run -- org director start --driver codex --model gpt-6-astra --state-dir "$L" --config "$C"
run -- org director attach --driver claude --state-dir "$L" --config "$C"
echo "  herdr calls of the two refusals: $(( $(hl) - n ))"
run -- org director stop --id d2 --state-dir "$L"
herdrsince "$n"; events

step "K3 the cleanup cannot close the pane: detached only, the herdr and stop commands; stop refuses until herdr answers"
n=$(hl)
run ORG_STUB_FAIL=agent:start ORG_STUB_CLOSE_FAIL_IDS=pane-stub-3 -- org director start --driver claude --model haiku --state-dir "$L" --config "$C"
herdrsince "$n"; events
run -- status --state-dir "$L"
n=$(hl)
run ORG_STUB_CLOSE_FAIL_IDS=pane-stub-3 -- org director stop --id d3 --state-dir "$L"
run -- org director stop --id d3 --state-dir "$L"
herdrsince "$n"; events
run -- status --state-dir "$L"

step "K4 stop from inside the director's pane or workspace is refused; another workspace id is not"
run -- org director start --driver claude --model haiku --state-dir "$L" --config "$C"
n=$(hl)
run HERDR_PANE_ID=pane-stub-4 -- org director stop --id d4 --state-dir "$L"
run HERDR_WORKSPACE_ID=ws-stub-4 -- org director stop --id d4 --state-dir "$L"
run HERDR_WORKSPACE_ID=ws-stub-4 -- org director stop --id d4 --force --state-dir "$L"
echo "  herdr calls of the three refusals:"; herdrsince "$n"
events
run HERDR_WORKSPACE_ID=ws-stub-9 -- org director stop --id d4 --state-dir "$L"
events

step "K9 cleanup"
run -- org watch --stop --state-dir "$L"
sleep 1
echo "  processes naming the run dir left: $(pgrep -f "$R" | wc -l | tr -d ' ')"
echo "  sleep 617 left: $(pgrep -f '^sleep 617$' | wc -l | tr -d ' ')"
echo "  osascript calls: $(cat "$S/live/osascript.log" 2>/dev/null | wc -l | tr -d ' ')"
echo "  ledger: $(ls "$L" | tr '\n' ' ')"
```

### 3.2 Output

```text
### K0 the long-running watch
$  ralph org watch --all --detach --state-dir <run>/ledger --config <run>/ralph.toml
  out: started ralph org watch --all (pid 53182); log: <run>/ledger/watch.log
  rc=0
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0

### K1 starting is shown while the agent start waits; the agent start times out and start cleans up (no USER in env)
  pane_created d1 recorded
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: attached d1 (driver=claude, headless starting, model=haiku, pane pane-stub-1, workspace ws-stub-1, attached at 2026-10-15T09:36:49Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0
$  ralph org director status --state-dir <run>/ledger --json
  out: {
  out:   "enabled": true,
  out:   "state": "attached",
  out:   "registration": {
  out:     "id": "d1",
  out:     "driver": "claude",
  out:     "wake": [
  out:       "pull"
  out:     ],
  out:     "pane_id": "pane-stub-1",
  out:     "launch": "herdr",
  out:     "state": "starting",
  out:     "token": "95b27157cd6c",
  out:     "model": "haiku",
  out:     "permission_mode": "guarded",
  out:     "workspace_id": "ws-stub-1",
  out:     "attached_at": "2026-10-15T09:36:49Z",
  out:     "user": "",
  out:     "cwd": "<run>/cwd",
  out:     "herdr_pane": false
  out:   },
  out:   "corrupt_lines": 0,
  out:   "ignored_events": 0
  out: }
  rc=0
  start rc=1 after 3 s
  err: org: director: start d1: herdr agent start director_95b27157cd6c (agent_start_retries=0): herdr: timed out: context deadline exceeded; start closed what it had made in herdr, and d1 is recorded stopped
  herdr: workspace create --cwd <run>/cwd --label ralph_director_95b27157cd6c
  herdr: tab create --workspace ws-stub-1 --cwd <run>/cwd --label director_95b27157cd6c
  herdr: agent start director_95b27157cd6c --kind claude --pane pane-stub-1 --timeout 2500 -- --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_95b27157cd6c.md
  herdr: pane get pane-stub-1
  herdr: tab get tab-stub-1
  herdr: workspace get ws-stub-1
  herdr: workspace get ws-stub-1
  herdr: pane send-keys pane-stub-1 C-c
  herdr: pane close pane-stub-1
  herdr: workspace close ws-stub-1
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0

### K2 a start killed during its agent start leaves d2 starting; start and attach refuse naming stop; stop closes what was recorded
  pane_created d2 recorded
  start killed (SIGKILL), rc=137
  the stub's orphaned agent-start sleep killed
  herdr: workspace create --cwd <run>/cwd --label ralph_director_2b0c3f710668
  herdr: tab create --workspace ws-stub-2 --cwd <run>/cwd --label director_2b0c3f710668
  herdr: agent start director_2b0c3f710668 --kind claude --pane pane-stub-2 --timeout 60000 -- --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_2b0c3f710668.md
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: attached d2 (driver=claude, headless starting, model=haiku, pane pane-stub-2, workspace ws-stub-2, attached at 2026-10-15T09:36:53Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0
$  ralph org director start --driver codex --model gpt-6-astra --state-dir <run>/ledger --config <run>/ralph.toml
  err: org: director: start: a director is already attached: d2 (driver claude, attached 2026-10-15T09:36:53Z, headless starting): stop it with ralph org director stop --id d2 --state-dir '<run>/ledger' from a terminal outside herdr, then run start again (start has no --replace)
  rc=1
$  ralph org director attach --driver claude --state-dir <run>/ledger --config <run>/ralph.toml
  err: org: director: attach: a director is already attached: d2 (driver claude, attached 2026-10-15T09:36:53Z, headless starting): stop it with ralph org director stop --id d2 --state-dir '<run>/ledger' from a terminal outside herdr, which closes its herdr pane, or replace it with ralph org director attach --driver claude --replace --state-dir '<run>/ledger', which leaves that pane open until such a stop
  rc=1
  herdr calls of the two refusals: 0
$  ralph org director stop --id d2 --state-dir <run>/ledger
  out: stopped d2 (ctrl_c=ok, pane=closed, workspace=closed)
  out: the director is no longer attached
  rc=0
  herdr: pane get pane-stub-2
  herdr: tab get tab-stub-2
  herdr: workspace get ws-stub-2
  herdr: workspace get ws-stub-2
  herdr: pane send-keys pane-stub-2 C-c
  herdr: pane close pane-stub-2
  herdr: workspace close ws-stub-2
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2, detached d2, stopped d2

### K3 the cleanup cannot close the pane: detached only, the herdr and stop commands; stop refuses until herdr answers
$ ORG_STUB_FAIL=agent:start ORG_STUB_CLOSE_FAIL_IDS=pane-stub-3 ralph org director start --driver claude --model haiku --state-dir <run>/ledger --config <run>/ralph.toml
  err: org: director: start d3: herdr agent start director_2b344f7e9bc3 (agent_start_retries=0): herdr: exit status 1: stub failure: agent start; its cleanup did not finish: close pane "pane-stub-3": herdr pane close: herdr: exit status 1: stub failure: pane close pane-stub-3; d3 was recorded detached only, so it is no longer the current registration but stays open; close it by hand: herdr pane close pane-stub-3, then herdr workspace close ws-stub-3; afterwards record the stop with ralph org director stop --id d3 --state-dir '<run>/ledger'
  rc=1
  herdr: workspace create --cwd <run>/cwd --label ralph_director_2b344f7e9bc3
  herdr: tab create --workspace ws-stub-3 --cwd <run>/cwd --label director_2b344f7e9bc3
  herdr: agent start director_2b344f7e9bc3 --kind claude --pane pane-stub-3 --timeout 60000 -- --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_2b344f7e9bc3.md
  herdr: pane get pane-stub-3
  herdr: tab get tab-stub-3
  herdr: workspace get ws-stub-3
  herdr: workspace get ws-stub-3
  herdr: pane send-keys pane-stub-3 C-c
  herdr: pane close pane-stub-3
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2, detached d2, stopped d2, attached d3, workspace_created d3 ws=ws-stub-3, pane_created d3 pane=pane-stub-3, detached d3
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work; the headless director d3 (starting, pane pane-stub-3, workspace ws-stub-3) is still open but not attached: stop it with ralph org director stop --id d3 --state-dir '<run>/ledger'
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0
$ ORG_STUB_CLOSE_FAIL_IDS=pane-stub-3 ralph org director stop --id d3 --state-dir <run>/ledger
  err: stop d3: ctrl_c=ok, pane=close failed: herdr pane close: herdr: exit status 1: stub failure: pane close pane-stub-3, workspace=not closed: the pane was left open
  err: org: director: stop d3: close pane "pane-stub-3": herdr pane close: herdr: exit status 1: stub failure: pane close pane-stub-3; nothing was recorded; close it by hand: herdr pane close pane-stub-3, then herdr workspace close ws-stub-3; afterwards record the stop with ralph org director stop --id d3 --state-dir '<run>/ledger'; or record the stop and leave them open with ralph org director stop --id d3 --force --state-dir '<run>/ledger'
  rc=1
$  ralph org director stop --id d3 --state-dir <run>/ledger
  out: stopped d3 (ctrl_c=ok, pane=closed, workspace=closed)
  rc=0
  herdr: pane get pane-stub-3
  herdr: tab get tab-stub-3
  herdr: workspace get ws-stub-3
  herdr: workspace get ws-stub-3
  herdr: pane send-keys pane-stub-3 C-c
  herdr: pane close pane-stub-3
  herdr: pane get pane-stub-3
  herdr: tab get tab-stub-3
  herdr: workspace get ws-stub-3
  herdr: workspace get ws-stub-3
  herdr: pane send-keys pane-stub-3 C-c
  herdr: pane close pane-stub-3
  herdr: workspace close ws-stub-3
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2, detached d2, stopped d2, attached d3, workspace_created d3 ws=ws-stub-3, pane_created d3 pane=pane-stub-3, detached d3, stopped d3
$  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 53182, mark updated 2026-10-15T09:36:47Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0

### K4 stop from inside the director's pane or workspace is refused; another workspace id is not
$  ralph org director start --driver claude --model haiku --state-dir <run>/ledger --config <run>/ralph.toml
  out: started d4 (driver=claude, model=haiku, permission=guarded, pane=pane-stub-4, workspace=ws-stub-4)
  out: talk to the director in herdr: pane pane-stub-4, in the tab director_fe401042a1c5 of the workspace labelled ralph_director_fe401042a1c5
  out: stop it with: ralph org director stop --id d4 --state-dir '<run>/ledger'
  rc=0
$ HERDR_PANE_ID=pane-stub-4 ralph org director stop --id d4 --state-dir <run>/ledger
  err: org: director: stop d4: HERDR_PANE_ID is its pane "pane-stub-4", and closing that pane would end this command before it records the stop: run ralph org director stop from a terminal outside the director's pane
  rc=1
$ HERDR_WORKSPACE_ID=ws-stub-4 ralph org director stop --id d4 --state-dir <run>/ledger
  err: org: director: stop d4: HERDR_WORKSPACE_ID is its workspace "ws-stub-4", labelled ralph_director_fe401042a1c5, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
  rc=1
$ HERDR_WORKSPACE_ID=ws-stub-4 ralph org director stop --id d4 --force --state-dir <run>/ledger
  err: org: director: stop d4: HERDR_WORKSPACE_ID is its workspace "ws-stub-4", labelled ralph_director_fe401042a1c5, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
  rc=1
  herdr calls of the three refusals:
  herdr: pane get pane-stub-4
  herdr: tab get tab-stub-4
  herdr: workspace get ws-stub-4
  herdr: workspace get ws-stub-4
  herdr: pane get pane-stub-4
  herdr: tab get tab-stub-4
  herdr: workspace get ws-stub-4
  herdr: workspace get ws-stub-4
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2, detached d2, stopped d2, attached d3, workspace_created d3 ws=ws-stub-3, pane_created d3 pane=pane-stub-3, detached d3, stopped d3, attached d4, workspace_created d4 ws=ws-stub-4, pane_created d4 pane=pane-stub-4, launched d4
$ HERDR_WORKSPACE_ID=ws-stub-9 ralph org director stop --id d4 --state-dir <run>/ledger
  out: stopped d4 (ctrl_c=ok, pane=closed, workspace=closed)
  out: the director is no longer attached
  rc=0
  director.jsonl: attached d1, workspace_created d1 ws=ws-stub-1, pane_created d1 pane=pane-stub-1, detached d1, stopped d1, attached d2, workspace_created d2 ws=ws-stub-2, pane_created d2 pane=pane-stub-2, detached d2, stopped d2, attached d3, workspace_created d3 ws=ws-stub-3, pane_created d3 pane=pane-stub-3, detached d3, stopped d3, attached d4, workspace_created d4 ws=ws-stub-4, pane_created d4 pane=pane-stub-4, launched d4, detached d4, stopped d4

### K9 cleanup
$  ralph org watch --stop --state-dir <run>/ledger
  out: stopped ralph org watch --all (pid 53182)
  rc=0
  processes naming the run dir left: 0
  sleep 617 left: 0
  osascript calls: 0
  ledger: director.jsonl director.lock prompts watch-alive.json watch.lock watch.log
```

The 8 herdr calls listed under "the three refusals" of K4 are the 4 label checks of each `HERDR_WORKSPACE_ID` refusal; the `HERDR_PANE_ID` refusal made none.

## 4. HEAD binary against an isolated real herdr server

- herdr 0.7.5 (`/opt/homebrew/bin/herdr`, symlinked into `$SCRATCH/real/bin` with `ralph`). The server was started with `env -i`, `HOME=/tmp/rdt9/home`, `XDG_CONFIG_HOME=/tmp/rdt9/cfg`, `XDG_STATE_HOME=/tmp/rdt9/state`, `HERDR_SOCKET_PATH=/tmp/rdt9/h.sock` (`herdr server` printed `api socket: /tmp/rdt9/h.sock`, `client socket: /tmp/rdt9/h-client.sock`, `logs: /tmp/rdt9/cfg/herdr/herdr-server.log`). `lsof` on the server process showed only files under `/tmp/rdt9` and `$SCRATCH/real`. No herdr server of the user ran before or after
- The pane agent was claude (2.1.291) with an empty `CLAUDE_CONFIG_DIR` under `/tmp/rdt9`, model haiku, permission mode guarded (`$SCRATCH/real/ralph.toml`: `[org] model_pool = [{ driver = "claude", model = "haiku" }]`, `[org.permissions] default = "autonomous"`, `[org.permissions.roles] director = "guarded"`, `[org.director] enabled = true`)

### 4.1 Script (`$SCRATCH/real/real.sh`)

```bash
#!/bin/bash
# Real herdr (isolated server under /tmp/rdt9): director stop typed into a
# pane of the director's own workspace (HERDR_WORKSPACE_ID set by herdr), and
# from a pane of another workspace.
set -u
S=$SCRATCH
D=/tmp/rdt9
L=$S/real/ledger
C=$S/real/ralph.toml
E=(env -i HOME="$D/home" PATH="$S/real/bin:/usr/bin:/bin" HERDR_SOCKET_PATH="$D/h.sock")
red() { sed -e "s#$S#\$SCRATCH#g" -e "s#$HOME#~#g"; }
step() { printf '\n### %s\n' "$*"; }
run() { printf '$ %s\n' "$*" | red; "${E[@]}" "$@" > "$D/o" 2> "$D/e"; local rc=$?; sed 's/^/  out: /' "$D/o" | red; sed 's/^/  err: /' "$D/e" | red; echo "  rc=$rc"; }
inpane() { # inpane <pane> <outfile> <ralph args...>: type the command into the pane, wait for it
  local pane=$1 out=$2; shift 2
  rm -f "$out"
  local cmd="ralph $* > $out 2>&1; echo rc=\$? >> $out; env | grep '^HERDR_' | sort >> $out"
  printf '[typed into %s] %s\n' "$pane" "$cmd" | red
  "${E[@]}" herdr pane send-text "$pane" "$cmd" > /dev/null
  "${E[@]}" herdr pane send-keys "$pane" Enter > /dev/null
  for _ in $(seq 1 150); do grep -q '^rc=' "$out" 2>/dev/null && break; sleep 0.2; done
  sed 's/^/  pane: /' "$out" | red
}
md5f() { md5 -q "$L/director.jsonl"; }

step "R1 watch and director start (claude, guarded, empty CLAUDE_CONFIG_DIR)"
run ralph org watch --all --detach --state-dir "$L" --config "$C"
sleep 2
cd "$S/real/cwd" || exit 1
run ralph org director start --driver claude --model haiku --state-dir "$L" --config "$C"
run herdr workspace list
run herdr tab list --workspace w1
run ralph status --state-dir "$L"

step "R2 stop typed into the first tab of the director's workspace (w1:p1)"
before=$(md5f)
inpane w1:p1 "$D/inws1.out" org director stop --id d1 --state-dir "$L"
inpane w1:p1 "$D/inws2.out" org director stop --id d1 --force --state-dir "$L"
echo "  director.jsonl unchanged: $([ "$(md5f)" = "$before" ] && echo yes || echo NO)"
run herdr workspace list

step "R3 stop typed into a pane of another workspace"
run herdr workspace create --cwd "$S/real/cwd" --label probe_other
inpane w2:p1 "$D/other.out" org director stop --id d1 --state-dir "$L"
run herdr workspace list
run herdr agent list
python3 -c 'import json,sys; print("  director.jsonl: "+", ".join(e["event"]+" "+e["id"] for e in map(json.loads, open(sys.argv[1]))))' "$L/director.jsonl"
run ralph status --state-dir "$L"
```

### 4.2 Output

```text
### R1 watch and director start (claude, guarded, empty CLAUDE_CONFIG_DIR)
$ ralph org watch --all --detach --state-dir $SCRATCH/real/ledger --config $SCRATCH/real/ralph.toml
  out: started ralph org watch --all (pid 82251); log: $SCRATCH/real/ledger/watch.log
  rc=0
$ ralph org director start --driver claude --model haiku --state-dir $SCRATCH/real/ledger --config $SCRATCH/real/ralph.toml
  out: started d1 (driver=claude, model=haiku, permission=guarded, pane=w1:p2, workspace=w1)
  out: talk to the director in herdr: pane w1:p2, in the tab director_61ea4337317f of the workspace labelled ralph_director_61ea4337317f
  out: stop it with: ralph org director stop --id d1 --state-dir '$SCRATCH/real/ledger'
  rc=0
$ herdr workspace list
  out: {"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w1:t1","agent_status":"idle","focused":true,"label":"ralph_director_61ea4337317f","number":1,"pane_count":2,"tab_count":2,"workspace_id":"w1"}]}}
  rc=0
$ herdr tab list --workspace w1
  out: {"id":"cli:tab:list","result":{"tabs":[{"agent_status":"unknown","focused":true,"label":"1","number":1,"pane_count":1,"tab_id":"w1:t1","workspace_id":"w1"},{"agent_status":"idle","focused":false,"label":"director_61ea4337317f","number":2,"pane_count":1,"tab_id":"w1:t2","workspace_id":"w1"}],"type":"tab_list"}}
  rc=0
$ ralph status --state-dir $SCRATCH/real/ledger
  out: no org runtime state found (state-dir=$SCRATCH/real/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 82251, mark updated 2026-10-15T09:40:50Z)
  out: director: attached d1 (driver=claude, headless running, model=haiku, pane w1:p2, workspace w1, attached at 2026-10-15T09:40:52Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0

### R2 stop typed into the first tab of the director's workspace (w1:p1)
[typed into w1:p1] ralph org director stop --id d1 --state-dir $SCRATCH/real/ledger > /tmp/rdt9/inws1.out 2>&1; echo rc=$? >> /tmp/rdt9/inws1.out; env | grep '^HERDR_' | sort >> /tmp/rdt9/inws1.out
  pane: org: director: stop d1: HERDR_WORKSPACE_ID is its workspace "w1", labelled ralph_director_61ea4337317f, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
  pane: rc=1
  pane: HERDR_ENV=1
  pane: HERDR_PANE_ID=w1:p1
  pane: HERDR_SOCKET_PATH=/tmp/rdt9/h.sock
  pane: HERDR_TAB_ID=w1:t1
  pane: HERDR_WORKSPACE_ID=w1
[typed into w1:p1] ralph org director stop --id d1 --force --state-dir $SCRATCH/real/ledger > /tmp/rdt9/inws2.out 2>&1; echo rc=$? >> /tmp/rdt9/inws2.out; env | grep '^HERDR_' | sort >> /tmp/rdt9/inws2.out
  pane: org: director: stop d1: HERDR_WORKSPACE_ID is its workspace "w1", labelled ralph_director_61ea4337317f, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
  pane: rc=1
  pane: HERDR_ENV=1
  pane: HERDR_PANE_ID=w1:p1
  pane: HERDR_SOCKET_PATH=/tmp/rdt9/h.sock
  pane: HERDR_TAB_ID=w1:t1
  pane: HERDR_WORKSPACE_ID=w1
  director.jsonl unchanged: yes
$ herdr workspace list
  out: {"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w1:t1","agent_status":"idle","focused":true,"label":"ralph_director_61ea4337317f","number":1,"pane_count":2,"tab_count":2,"workspace_id":"w1"}]}}
  rc=0

### R3 stop typed into a pane of another workspace
$ herdr workspace create --cwd $SCRATCH/real/cwd --label probe_other
  out: {"id":"cli:workspace:create","result":{"root_pane":{"agent_status":"unknown","cwd":"$SCRATCH/real/cwd","focused":false,"foreground_cwd":"$SCRATCH/real/cwd","pane_id":"w2:p1","revision":0,"scroll":{"max_offset_from_bottom":0,"offset_from_bottom":0,"viewport_rows":23},"tab_id":"w2:t1","terminal_id":"term_65dddd83446e13","workspace_id":"w2"},"tab":{"agent_status":"unknown","focused":false,"label":"1","number":1,"pane_count":1,"tab_id":"w2:t1","workspace_id":"w2"},"type":"workspace_created","workspace":{"active_tab_id":"w2:t1","agent_status":"unknown","focused":false,"label":"probe_other","number":2,"pane_count":1,"tab_count":1,"workspace_id":"w2"}}}
  rc=0
[typed into w2:p1] ralph org director stop --id d1 --state-dir $SCRATCH/real/ledger > /tmp/rdt9/other.out 2>&1; echo rc=$? >> /tmp/rdt9/other.out; env | grep '^HERDR_' | sort >> /tmp/rdt9/other.out
  pane: stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
  pane: the director is no longer attached
  pane: rc=0
  pane: HERDR_ENV=1
  pane: HERDR_PANE_ID=w2:p1
  pane: HERDR_SOCKET_PATH=/tmp/rdt9/h.sock
  pane: HERDR_TAB_ID=w2:t1
  pane: HERDR_WORKSPACE_ID=w2
$ herdr workspace list
  out: {"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w2:t1","agent_status":"unknown","focused":true,"label":"probe_other","number":1,"pane_count":1,"tab_count":1,"workspace_id":"w2"}]}}
  rc=0
$ herdr agent list
  out: {"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}
  rc=0
  director.jsonl: attached d1, workspace_created d1, pane_created d1, launched d1, detached d1, stopped d1
$ ralph status --state-dir $SCRATCH/real/ledger
  out: no org runtime state found (state-dir=$SCRATCH/real/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 82251, mark updated 2026-10-15T09:40:50Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0
```

### 4.3 Cleanup

- Under the same `env -i` environment: `herdr workspace close w2` rc 0, `ralph org watch --stop --state-dir $SCRATCH/real/ledger` rc 0, `herdr server stop` rc 0
- Afterwards no `herdr server` process, no process naming `/tmp/rdt9` or `$SCRATCH/real`, and `/tmp/rdt9` removed
- The user's herdr state, read only: `~/.config/herdr/session.json` sha256 prefix 004173699fb3c932 (mtime Sep 19 14:09:37), `~/.config/herdr/herdr-server.log` sha256 prefix fe019d6d40e422ca (mtime Oct 1 14:35:45), the same values before and after

## 5. Mutation (go1.26.0, `go test -overlay`)

### 5.1 Harness

`$SCRATCH/mut/mutate.py` applies each mutant as literal (old, new) edits; an edit is refused unless `old` occurs exactly once in the file. The mutated copy goes under `$SCRATCH/mut/out/<id>/` and an overlay JSON maps the worktree path to it. Each run is `go test -overlay=<json> -count=1 -run <regex> <pkg>` with the canary PATH. The org regex is `Director|StopAll|Spawn_DirectorRole|RenderRolePrompt|CodexWritableRoot`; the cli regex is `OrgDirector|StatusCmd_Director|OrgStopAll|OrgSpawn_RoleDirector|OrgLegacyLedger`. X0 and X0c pass an unchanged file through the overlay as controls. All edits applied (no BADEDIT, no BUILD failure).

### 5.2 Results (filtered runs)

86 mutants and 2 controls: 78 red, 8 survived, controls passed. "first failing tests" drops the `Test` prefix and shows two names, then how many more.

| ID | file | mutation | result | first failing tests |
| --- | --- | --- | --- | --- |
| X0 | director_stop.go | org: director_stop.go unchanged through -overlay | SURVIVED 3s |  |
| X0c | org_director_headless.go | cli: org_director_headless.go unchanged through -overlay | SURVIVED 38s |  |
| A1 | director_stop.go | start cleanup closes without the label check (closeDirectorResources ignores confirm's error) | red 2s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly |
| A2 | director_stop.go | stop closes whatever confirm said | red 5s | Director_TwoLedgersOnOneHerdrServerNeverCloseEachOther, StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly (+4) |
| A3 | director_stop.go | the recorded pane is not checked | red 6s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop (+11) |
| A4 | director_stop.go | the recorded workspace is not checked | red 5s | Director_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration, StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop (+9) |
| A5 | verbs.go | confirmSeatPane: the tab label is not compared (shared with seats) | red 5s | OrgStopAll_OwnPaneIDNotConfirmed_NotDeferred, StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly (+3) |
| A6 | verbs.go | confirmSeatPane: the pane's workspace label mismatch is ignored (shared) | red 5s | StopDirector_ALabelNotThisLaunchsSendsNothing |
| A7 | verbs.go | confirmOrgWorkspace: the label is not compared (shared) | red 5s | StopDirector_ALabelNotThisLaunchsSendsNothing, StopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch |
| A8 | director_stop.go | the two labels swapped in the pane check | red 7s | Director_TwoLedgersOnOneHerdrServerNeverCloseEachOther, StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly (+13) |
| A9 | verbs.go | tab label compared by length only (another token's label passes; shared) | red 7s | StopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce, StopDirector_ALabelNotThisLaunchsSendsNothing (+1) |
| A10 | verbs.go | workspace label compared by length only (shared) | red 5s | StopDirector_ALabelNotThisLaunchsSendsNothing, StopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch |
| A11 | verbs.go | a failed pane get counts as an already closed pane (shared) | red 6s | StopDirector_ForceWhenAGetTimesOutSendsNothing |
| A12 | verbs.go | a failed workspace get counts as an already closed workspace (shared) | SURVIVED 5s |  |
| A13 | verbs.go | a pane whose tab herdr lacks counts as closed (shared) | SURVIVED 6s |  |
| A14 | verbs.go | a pane whose workspace herdr lacks counts as confirmed (shared) | SURVIVED 5s |  |
| B1 | director_stop.go | with Force, stop closes what confirm did not confirm | red 6s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch (+2) |
| B2 | director_stop.go | after a failed pane close the workspace is closed anyway | red 6s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StopDirector_ACloseThatFailsRecordsNothing (+2) |
| B3 | director_stop.go | a forced stop is recorded without force | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch (+3) |
| B4 | director_stop.go | Force always marks the stop forced | red 5s | StopDirector_ForceWithLabelsThatMatchClosesAsWithoutIt |
| B5 | director_stop.go | without Force an unconfirmed stop is recorded anyway | red 6s | Director_TwoLedgersOnOneHerdrServerNeverCloseEachOther, StopDirector_ACloseThatFailsRecordsNothing (+2) |
| B6 | org_director_headless.go | CLI: --force is not passed to StopDirector | red 46s | OrgDirectorStop_ACloseThatFailsNamesForce, OrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced |
| B7 | director_stop.go | a forced stop's herdr commands name the recorded workspace id | red 4s | StopDirector_ForceAfterACloseThatFailsRecordsWhatWasLeftOpen, StopDirector_ForceRecordsAStopWhoseLabelsAreAnotherLaunchs (+1) |
| B8 | director_stop.go | an unconfirmed cleanup error names the recorded ids to close | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce (+1) |
| B9 | org_director_headless.go | CLI: no stderr warning after a forced stop | red 53s | OrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced |
| B11 | director_stop.go | a failed pane close sets no error (the stop is recorded as if closed) | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StopDirector_ACloseThatFailsRecordsNothing (+2) |
| C1 | director_stop.go | no self-pane refusal | red 5s | StopDirector_Refusals |
| C2 | director_stop.go | self-pane refusal without the own != "" guard | red 5s | Director_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration, StopDirector_RefusesInsideTheDirectorsWorkspace |
| C3 | director_stop.go | no self-workspace refusal | red 5s | StopDirector_RefusesInsideTheDirectorsWorkspace |
| C4 | director_stop.go | self-workspace refusal on the env id alone (unconfirmed label) | red 6s | StopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch |
| C5 | director_stop.go | self-workspace refusal of a workspace herdr no longer knows | red 6s | StopDirector_RefusesInsideTheDirectorsWorkspace |
| C6 | director_stop.go | self-workspace refusal without the recorded-id guard | red 6s | Director_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration |
| C7 | director_stop.go | self-workspace refusal reads HERDR_PANE_ID | red 9s | StopDirector_RefusesInsideTheDirectorsWorkspace |
| C8 | org_director_headless.go | CLI: the runtime reads no HERDR_* env (Getenv returns empty) | red 56s | OrgDirectorStart_RefusedWithNothingWritten, OrgDirectorStop_RefusedInsideTheDirectorsPaneOrWorkspace |
| C9 | director_start.go | start: no HERDR_PANE_ID refusal | red 8s | StartDirector_RefusesBeforeWritingOrCallingHerdr |
| C10 | director_stop.go | self-pane refusal after the label checks (herdr gets called first) | red 8s | StopDirector_Refusals |
| D1 | director_start.go | no early attached refusal (AttachHeadless under the lock still refuses) | red 7s | StartDirector_RefusesBeforeWritingOrCallingHerdr |
| D2 | director_start.go | the watch gate runs before the early attached look | red 7s | StartDirector_RefusesBeforeWritingOrCallingHerdr |
| D3 | director_start.go | the early refusal is a plain error, not *DirectorAttachedError | red 6s | Director_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| D4 | director_start.go | an unreadable director.jsonl at the early look is ignored (the gate runs) | SURVIVED 7s |  |
| D5 | director_start.go | AttachHeadless does not refuse a current registration under the lock | red 7s | DirectorAttachAndAttachHeadless_ConcurrentlyOnlyOneRegisters, DirectorAttachHeadless_RefusesWhileARegistrationIsCurrent (+1) |
| D6 | director_start.go | AttachHeadless's refusal is returned without the re-read *DirectorAttachedError | red 8s | StartDirector_ARegistrationAfterTheFirstLookIsRefusedUnderTheLock |
| D7 | org_director_headless.go | CLI: director start wires no watch gate | red 50s | OrgDirectorStart_WatchGate |
| D8 | org_director_headless.go | CLI: start's attached refusal gets no way-out command | red 46s | OrgDirectorStart_RefusedWithNothingWritten |
| E1 | director_start.go | a failed attached write does not stop start (herdr is called) | SURVIVED 5s |  |
| E2 | director_start.go | workspace_created is never written | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop (+14) |
| E3 | director_start.go | workspace_created written only after the tab create | red 5s | StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop, StartDirector_ARecordThatFailsStillClosesWhatHerdrMade |
| E4 | director_start.go | launched written before the agent start | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop (+1) |
| E5 | director_start.go | pane_created written only after the agent start | red 5s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly, StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop (+2) |
| E6 | director_start.go | the agent start runs under director.lock | red 15s | StartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch |
| E7 | director_start.go | MarkLaunched does not compare the current id | red 5s | DirectorMarkLaunched_OnlyWhileTheRegistrationIsStillCurrent |
| E8 | director_start.go | a replaced registration does not end start's own launch | red 6s | StartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch |
| E9 | director_start.go | start's cleanup closes but records no stop | red 6s | StartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop, StartDirector_ARecordThatFailsStillClosesWhatHerdrMade (+1) |
| E10 | director_start.go | start's cleanup that cannot close writes no detached | red 6s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly |
| E11 | director_start.go | start's cleanup records stopped although the close failed | red 7s | StartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly |
| E12 | director_start.go | a failed workspace create keeps herdr's output as the workspace id | SURVIVED 7s |  |
| E13 | director_start.go | a failed tab create keeps herdr's output as the pane id | SURVIVED 7s |  |
| P1 | org_director_headless.go | start's success line: stop without --state-dir | red 50s | OrgDirectorStart_LaunchesRegistersAndStops |
| P2 | org_director_headless.go | start refused, current read failed or none: status without --state-dir | SURVIVED 49s |  |
| P3 | org_director_headless.go | start refused by a headless director: stop without --state-dir | red 45s | OrgDirectorStart_RefusedWithNothingWritten |
| P4 | org_director_headless.go | start refused by a promoted session: detach without --state-dir | red 46s | OrgDirectorStart_RefusedWithNothingWritten |
| P5 | org_director_headless.go | a cleanup error's stop command without --state-dir | red 47s | OrgDirectorStart_AFailedAgentStartIsCleanedUp, OrgDirectorStop_ACloseThatFailsNamesForce (+1) |
| P6 | org_director_headless.go | stop's --force suggestion without --state-dir | red 47s | OrgDirectorStop_ACloseThatFailsNamesForce |
| P7 | org_director_headless.go | stop of an unknown id: status without --state-dir | red 41s | OrgDirectorStop_Refusals |
| P8 | org_director_headless.go | stop of a promoted session: detach without --state-dir | red 42s | OrgDirectorStop_Refusals |
| P9 | org_director.go | attach refused by a headless director: stop without --state-dir | red 42s | OrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop |
| P10 | org_director.go | detach / attach --replace of a headless director: stop without --state-dir | red 46s | OrgDirectorAttachReplace_AHeadlessDirectorNamesItsStop, OrgDirectorDetach_AHeadlessDirectorNamesItsStop |
| P11 | org_director.go | status disabled with a headless registration: stop without --state-dir | red 46s | StatusCmd_DirectorLineShowsHeadlessDirectors |
| P12 | org_director.go | status note, one open headless: stop without --state-dir | red 46s | OrgDirectorStart_AFailedAgentStartIsCleanedUp, OrgDirectorStatus_ShowsHeadlessDirectors (+3) |
| P13 | org_director.go | status note, several open headless: stop without --state-dir | red 39s | OrgDirectorStatus_ShowsHeadlessDirectors, StatusCmd_DirectorLineShowsHeadlessDirectors |
| P14 | org.go | stop --all note: stop without --state-dir | red 40s | OrgStopAll_NamesTheOpenHeadlessDirectors |
| P15 | status.go | ralph status with no org: director line without the hint | red 40s | StatusCmd_DirectorLineShowsHeadlessDirectors, StatusCmd_DirectorStopCommandRepeatsTheStateDir |
| P16 | status.go | ralph status with orgs: director line without the hint | red 34s | StatusCmd_DirectorLineShowsHeadlessDirectors |
| P17 | org.go | stop --all: the note gets no hint | red 36s | OrgStopAll_NamesTheOpenHeadlessDirectors |
| P18 | director_start.go | the prompt's state dir is the cwd | red 5s | StartDirector_LaunchesInAWorkspaceOfItsOwn |
| P19 | prompts.go | the prompt's state dir is not shell-quoted | red 5s | RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure, RenderDirectorPrompt_SubstitutesEveryPlaceholder (+1) |
| P20 | director.md | director.md: the first status command has no --state-dir | red 5s | RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure |
| P21 | director.md | director.md: the stop command has no --state-dir | red 5s | RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure |
| P22 | director.md | director.md: the re-read after a summary has no --state-dir | red 5s | RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure |
| P23 | director.md | director.md: the every-command --state-dir sentence loses its value | red 5s | RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure |
| G1 | spawn.go | Spawn does not call DirectorRoleInputErr | red 9s | OrgSpawn_DirectorRole_RejectedBeforeAnyManifestWrite |
| G2 | spawn.go | DirectorRoleInputErr is case-insensitive | red 9s | DirectorRoleInputErr_IsExactAndCaseSensitive |
| G3 | spawn.go | DirectorRoleInputErr never refuses | red 5s | DirectorRoleInputErr_IsExactAndCaseSensitive, OrgSpawn_DirectorRole_RejectedBeforeAnyManifestWrite |
| G3c | spawn.go | DirectorRoleInputErr never refuses, CLI tests | red 33s | OrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback |
| G4 | org.go | CLI spawn does not call DirectorRoleInputErr (Spawn still refuses) | red 31s | OrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback |
| G5 | prompts.go | RenderRolePrompt renders director.md for a seat | red 5s | RenderRolePrompt_DirectorIsNotASeatTemplate |
| Z1 | verbs_all.go | stop --all lists no open headless directors | red 5s | OrgStopAll_AnUnreadableDirectorFileChangesNothing, OrgStopAll_ReportsOpenHeadlessDirectorsAndLeavesThem |
| Z2 | permissions.go | codex director gets no --add-dir | red 5s | CodexWritableRootArgs, StartDirector_LaunchesInAWorkspaceOfItsOwn (+1) |

### 5.3 The 8 survivors against whole packages (`-run` dropped)

Org-layer survivors ran against both `./internal/org` and `./internal/cli`; P2 against `./internal/cli`.

| run | result | time | failing top-level tests |
| --- | --- | --- | --- |
| A12-orgfull | red | 93s | TestOrgStopDisband_UnansweredGet_LeavesItAlone |
| A12-clifull | SURVIVED | 259s |  |
| A13-orgfull | red | 92s | TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose |
| A13-clifull | SURVIVED | 240s |  |
| A14-orgfull | red | 106s | TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose |
| A14-clifull | SURVIVED | 243s |  |
| D4-orgfull | SURVIVED | 90s |  |
| D4-clifull | SURVIVED | 255s |  |
| E1-orgfull | SURVIVED | 99s |  |
| E1-clifull | SURVIVED | 261s |  |
| E12-orgfull | SURVIVED | 95s |  |
| E12-clifull | SURVIVED | 262s |  |
| E13-orgfull | SURVIVED | 98s |  |
| E13-clifull | SURVIVED | 238s |  |
| P2-clifull | SURVIVED | 234s |  |

A12, A13, and A14 are in `confirmSeatPane` / `confirmOrgWorkspace` of `verbs.go`, shared with the seats; seat tests kill them. Final count: 86 mutants, 81 red, 5 survivors (D4, E1, E12, E13, P2).

Edits of the 8:

- A12 (./internal/org): a failed workspace get counts as an already closed workspace (shared)
  - `internal/org/verbs.go`: `'\tcase err != nil:\n\t\treturn false, fmt.Errorf("check workspace %q: %w", workspaceID, err)'` -> `'\tcase err != nil:\n\t\treturn true, nil'`
- A13 (./internal/org): a pane whose tab herdr lacks counts as closed (shared)
  - `internal/org/verbs.go`: `'\t\treturn false, fmt.Errorf("check pane %q: herdr has no tab %q although it has the pane (%v)", paneID, tabID, err)'` -> `'\t\treturn true, nil'`
- A14 (./internal/org): a pane whose workspace herdr lacks counts as confirmed (shared)
  - `internal/org/verbs.go`: `'\tcase wsGone:\n\t\treturn false, fmt.Errorf("pane %q is not confirmed in %s: herdr has no workspace %q"'` -> `'\tcase wsGone && false:\n\t\treturn false, fmt.Errorf("pane %q is not confirmed in %s: herdr has no workspace %q"'`
- D4 (./internal/org): an unreadable director.jsonl at the early look is ignored (the gate runs)
  - `internal/org/director_start.go`: `'\tres, err := p.Store.Read()\n\tif err != nil {\n'` -> `'\tres, err := p.Store.Read()\n\tif err != nil && false {\n'`
- E1 (./internal/org): a failed attached write does not stop start (herdr is called)
  - `internal/org/director_start.go`: `'\tif err != nil {\n\t\treturn DirectorStartResult{}, err\n\t}\n\tr := &directorStartRun{'` -> `'\tif err != nil && false {\n\t\treturn DirectorStartResult{}, err\n\t}\n\tr := &directorStartRun{'`
- E12 (./internal/org): a failed workspace create keeps herdr's output as the workspace id
  - `internal/org/director_start.go`: `'\t\tr.reg.WorkspaceID = ""\n'` -> `''`
- E13 (./internal/org): a failed tab create keeps herdr's output as the pane id
  - `internal/org/director_start.go`: `'\t\tr.reg.PaneID = ""\n'` -> `''`
- P2 (./internal/cli): start refused, current read failed or none: status without --state-dir
  - `internal/cli/org_director_headless.go`: `'e, rt.command("ralph org director status"))'` -> `'e, "ralph org director status")'`

The real driver's `WorkspaceCreate` and `TabCreate` (`internal/org/driver/herdr.go` 123-126 and 153-156) return `out, err` when the command fails, while the tests' `fakeHerdr` returns `"", err`. That is why E12 and E13 survive: with the fake there is no output to keep.

### 5.4 Probe tests for the 5 survivors

Two scratch files read through the overlay as `internal/org/zz_probe_director_start_test.go` and `internal/cli/zz_probe_director_headless_test.go`, each run alone (`-run TestProbe_`) at HEAD, under its target mutant, and under a mutant it does not target (D1, P3) as a control. E2 is an incidental extra.

`zz_probe_director_start_test.go`:

```go
package org

// Probe tests for the /test cycle 1 survivors of org-director-start. They are
// read through go test -overlay only and are never committed.

import (
	"context"
	"errors"
	"os"
	"slices"
	"syscall"
	"testing"
)

// PD4 (kills D4): an unreadable director.jsonl at the early look refuses
// start before the watch gate, with no herdr call.
func TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	if err := os.Mkdir(f.store.Path(), 0o755); err != nil {
		t.Fatalf("make director.jsonl a directory: %v", err)
	}
	gateCalls := 0
	f.o.WatchGate = func() error { gateCalls++; return nil }
	_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
	if err == nil {
		t.Fatal("StartDirector succeeded with an unreadable director.jsonl")
	}
	if gateCalls != 0 {
		t.Fatalf("watch gate asked %d times, want 0 (err %v)", gateCalls, err)
	}
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", f.h.calls)
	}
}

// PE1 (kills E1): a failed attached write ends start before any herdr call.
func TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	f.store.write = func(*os.File, []byte) (int, error) { return 0, syscall.ENOSPC }
	_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("err = %v, want ENOSPC", err)
	}
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", f.h.calls)
	}
	// The failed append is cut back to the empty file it started from.
	if got := directorFileState(t, f.store); got != "" && got != "<missing>" {
		t.Fatalf("director.jsonl = %q, want empty or missing", got)
	}
}

// junkOutputHerdr returns herdr's stdout along with the error, as the real
// driver does when the command fails.
type junkOutputHerdr struct {
	*directorHerdr
}

func (h *junkOutputHerdr) WorkspaceCreate(ctx context.Context, cwd, label string) (string, error) {
	id, err := h.directorHerdr.WorkspaceCreate(ctx, cwd, label)
	if err != nil {
		return "Error: workspace create failed", err
	}
	return id, nil
}

func (h *junkOutputHerdr) TabCreate(ctx context.Context, workspaceID, cwd, label string) (string, error) {
	id, err := h.directorHerdr.TabCreate(ctx, workspaceID, cwd, label)
	if err != nil {
		return "Error: tab create failed", err
	}
	return id, nil
}

// PE12 / PE13 (kill E12 / E13): the output of a failed create is not an id;
// the cleanup closes only what herdr handed out.
func TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID(t *testing.T) {
	boom := errors.New("herdr: connection refused")
	cases := []struct {
		name      string
		inject    func(h *directorHerdr)
		wantCalls []string
		wantTrail []string
	}{
		{"workspace create", func(h *directorHerdr) { h.workspaceCreateErr = boom },
			[]string{"workspace_create"}, []string{"attached d1", "detached d1", "stopped d1"}},
		{"tab create", func(h *directorHerdr) { h.tabCreateErr = boom },
			[]string{"workspace_create", "tab_create", "workspace_get", "workspace_close"},
			[]string{"attached d1", "workspace_created d1", "detached d1", "stopped d1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			tc.inject(f.h)
			f.o.Herdr = &junkOutputHerdr{f.h}
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v, want it to wrap the create's error", err)
			}
			if !slices.Equal(f.h.calls, tc.wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, tc.wantCalls)
			}
			wantTrail(t, f.store, tc.wantTrail...)
			wantNoDirectorOpen(t, f.store)
		})
	}
}
```

`zz_probe_director_headless_test.go`:

```go
package cli

// Probe test for the /test cycle 1 survivor P2 of org-director-start. Read
// through go test -overlay only and never committed.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// PP2 (kills P2): start's refusal with no current registration on the
// re-read names status with this invocation's --state-dir.
func TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir(t *testing.T) {
	refusal := fmt.Errorf("org: director: start: %w: d1 (driver claude, attached 2026-10-15T09:00:00Z)", org.ErrDirectorAttached)
	for _, tc := range []struct{ name, hintStateDir, want string }{
		{"without --state-dir", "", refusal.Error() + ": read the current registration with ralph org director status, and run start again once none is attached (start has no --replace)"},
		{"with --state-dir", "/work/the ledger", refusal.Error() + ": read the current registration with ralph org director status --state-dir '/work/the ledger', and run start again once none is attached (start has no --replace)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &orgDirectorRuntime{hintStateDir: tc.hintStateDir}
			err := rt.startAttachedError(&org.DirectorAttachedError{Err: refusal})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("startAttachedError = %v; want %q", err, tc.want)
			}
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Fatalf("startAttachedError = %v; want it to wrap ErrDirectorAttached", err)
			}
		})
	}
}
```

The first PE1 draft read the file with `directorTrail`, which fails on the empty file the restore leaves; it was changed to `directorFileState` before the runs below.

Runs (the `--- PASS/FAIL` lines, the failure messages, and the package line; the parent lines of the table tests are dropped):

```text
== HEAD (org)
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	1.570s
== D4
    zz_probe_director_start_test.go:29: watch gate asked 1 times, want 0 (err org: director: read $TMPDIR/TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate1557092891/001/director.jsonl: read $TMPDIR/TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate1557092891/001/director.jsonl: is a directory)
--- FAIL: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.495s
== E1
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
    zz_probe_director_start_test.go:42: err = org: director: record the workspace: "" is not a registration id (d<N>, such as d1); its cleanup did not finish: record  stopped: org: director: stop: "" is not a registration id (d<N>, such as d1); nothing more was recorded; record the stop with ralph org director stop --id , want ENOSPC
--- FAIL: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.602s
== E12
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    zz_probe_director_start_test.go:101: herdr calls = [workspace_create workspace_get], want [workspace_create]
    --- FAIL: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.467s
== E13
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    zz_probe_director_start_test.go:101: herdr calls = [workspace_create tab_create pane_get workspace_get workspace_close], want [workspace_create tab_create workspace_get workspace_close]
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- FAIL: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.535s
== D1 (control)
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	1.663s
== E2 (incidental)
--- PASS: TestProbe_PD4_AnUnreadableDirectorFileRefusesBeforeTheGate (0.00s)
--- PASS: TestProbe_PE1_AFailedAttachedWriteCallsNoHerdr (0.00s)
    zz_probe_director_start_test.go:103: director.jsonl events = ["attached d1" "detached d1" "stopped d1"], want ["attached d1" "workspace_created d1" "detached d1" "stopped d1"]
    --- PASS: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/workspace_create (0.00s)
    --- FAIL: TestProbe_PE12_PE13_AFailedCreatesOutputIsNotAnID/tab_create (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.757s
== HEAD (cli)
    --- PASS: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/without_--state-dir (0.00s)
    --- PASS: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/with_--state-dir (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	1.514s
== P2
    zz_probe_director_headless_test.go:26: startAttachedError = org: director: start: a director is already attached: d1 (driver claude, attached 2026-10-15T09:00:00Z): read the current registration with ralph org director status, and run start again once none is attached (start has no --replace); want "org: director: start: a director is already attached: d1 (driver claude, attached 2026-10-15T09:00:00Z): read the current registration with ralph org director status --state-dir '/work/the ledger', and run start again once none is attached (start has no --replace)"
    --- PASS: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/without_--state-dir (0.00s)
    --- FAIL: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/with_--state-dir (0.00s)
FAIL	github.com/yoshpy-dev/ralph/internal/cli	1.489s
== P3 (control)
    --- PASS: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/without_--state-dir (0.00s)
    --- PASS: TestProbe_PP2_StartAttachedErrorWithNoCurrentNamesStatusWithTheStateDir/with_--state-dir (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	1.462s
```

## 6. This branch merged with today's main

`main` is 26 commits past the base (9cbf7080..f21df66c, PR #230 plan-visual-easy). `git merge-tree --write-tree main HEAD` gives tree 65c7c26a with one conflict, `docs/tech-debt/README.md` (both sides add lines); `internal/org/prompts_test.go` and `internal/org/prompts/leader.md` (main's only Go-side change: +3 and +6 lines) merge cleanly. The tree was extracted with `git archive 65c7c26a` into `$SCRATCH/merged` and tested there:

`cd $SCRATCH/merged && go test -count=1 ./internal/org/... ./internal/cli/` (go1.26.0, canary PATH), rc 0, 222 s wall:

```text
ok  	github.com/yoshpy-dev/ralph/internal/org	82.966s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	2.119s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.471s
ok  	github.com/yoshpy-dev/ralph/internal/cli	219.100s
```

The conflict in `docs/tech-debt/README.md` is a docs-only merge for /pr; the shell suites were not run on the merged tree.
