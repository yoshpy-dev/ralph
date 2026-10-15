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

## Cycle 2

- Date: 2026-10-15 (JST 21:38-22:20; the UTC timestamps in the output read 2026-10-15T12:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD 5dce71de (origin/main f21df66c merged in at 1cdddbbb; cycle 1 ran at b6c86188). Go changed since cycle 1 in 3 commits: 3243c499 and a539a765 (the --config carried into the director prompt), 4ef33195 (tests for cycle 1's survivors D4, E1, E12, E13, P2)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0. No real herdr ran in this cycle. Load average 4-17 during the runs (other sessions)
- Redaction: as in cycle 1 (`$SCRATCH`, `~`, `$TMPDIR/`, `<run>`). Cycle 2's scratch files are under `$SCRATCH/tds2`. The wrapper's full log `docs/evidence/verify-2026-10-15-123924.log` and the raw outputs appended to `docs/evidence/test-2026-10-15-org-director-start.log` (from the line `===== /test cycle 2`) are gitignored and not committed
- Trailing blanks at the ends of output lines were stripped; nothing else in the output was changed apart from the redaction and the field-line filter named in C2-3.2
- Nothing in the worktree was edited for the tests. Mutants and probe tests lived in `$SCRATCH/tds2` and reached the build only through `go test -overlay`

### C2-1 ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; templates/base/ralph.toml widened it to full)

Command: `PATH=$SCRATCH/tds2/canary/bin:$PATH ./scripts/run-test.sh > runtest.out 2>&1` (rc 0, 701 s wall, 4,654 lines). Filtered with the same command as section 1 (168 lines). Against section 1's 168 lines, with the line numbers, the `N.NNs` timings, and the timestamp dropped, `diff` gives:

```text
73c73
< plan-visual tests: 103 passed, 0 failed, 103 total
---
> plan-visual tests: 243 passed, 0 failed, 243 total
168c168
< Evidence saved to: docs/evidence/verify-2026-10-15-092359.log
---
> Evidence saved to: docs/evidence/verify-2026-10-15-123924.log
```

PR #230 (merged in at 1cdddbbb) added the 140 plan-visual cases. 41 suites, 41 "OK", every "FAIL:" line is 0 (25 of them, no other FAIL line). 4,283 lines start with PASS after leading blanks (cycle 1: 4,143; the 140 plan-visual cases are the difference). The one real SKIP is tests/test-secret-scan-branch.sh's (git 2.49.0). The Go part:

```text
4637:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
4641:?   	github.com/yoshpy-dev/ralph	[no test files]
4642:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4643:ok  	github.com/yoshpy-dev/ralph/internal/cli	213.707s
4644:ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
4645:ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
4646:ok  	github.com/yoshpy-dev/ralph/internal/org	82.725s
4647:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	0.895s
4648:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4649:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4650:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4652:==> All verifiers passed.
4654:Evidence saved to: docs/evidence/verify-2026-10-15-123924.log
```

The five `(cached)` packages ran with `-count=1` in C2-2.1 on both toolchains.

### C2-2 Go tests

Every Go run below had `$SCRATCH/tds2/canary/bin` first on PATH (C2-2.3).

#### C2-2.1 Per package, `-count=1 -v`

Counts are from the `--- PASS/FAIL/SKIP` lines (top-level tests, then subtests).

go1.25.8 (`GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile=cover125.out ./internal/...`, rc 0, 250 s wall):

```text
internal/cli	ok	245.612s	top 700 (pass 700 fail 0 skip 0)	sub 641 (pass 641 fail 0 skip 0)
internal/config	ok	4.147s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	4.080s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	95.866s	top 804 (pass 803 fail 0 skip 1)	sub 1692 (pass 1692 fail 0 skip 0)
internal/org/driver	ok	6.767s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	5.262s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	5.665s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	8.597s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.26.0, org and cli with the race detector (`go test -race -count=1 -v ./internal/org/ ./internal/cli/`, rc 0, 249 s wall, 0 `DATA RACE` lines):

```text
internal/org	ok	102.012s	top 804 (pass 803 fail 0 skip 1)	sub 1692 (pass 1692 fail 0 skip 0)
internal/cli	ok	241.835s	top 700 (pass 700 fail 0 skip 0)	sub 641 (pass 641 fail 0 skip 0)
```

go1.26.0, the other six packages (`go test -count=1 -v ./internal/config/ ./internal/insights/ ./internal/org/driver/ ./internal/org/protocol/ ./internal/scaffold/ ./internal/upgrade/`, rc 0):

```text
internal/config	ok	0.643s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	1.426s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org/driver	ok	2.061s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	2.115s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	0.957s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	3.022s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

The sorted `<package> <name> <result>` lines of the go1.25.8 run (4,318) and of the two go1.26.0 runs together (4,318) are identical. Against cycle 1's 4,301 lines (section 2.1), 17 lines were added and none removed:

```text
TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus PASS
TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus/with_--state-dir PASS
TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus/without_--state-dir PASS
TestPlanDigest_MatchesScript/archive/2026-10-15-plan-visual-easy.md PASS
TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig PASS
TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig/with_--config PASS
TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig/without_--config PASS
TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders/a_config_path_that_holds_a_placeholder PASS
TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders/a_relative_config_path PASS
TestStartDirector_AFailedCreatesOutputIsNotAnID PASS
TestStartDirector_AFailedCreatesOutputIsNotAnID/tab_create PASS
TestStartDirector_AFailedCreatesOutputIsNotAnID/workspace_create PASS
TestStartDirector_AFailedWriteOfAttachedCallsNoHerdr PASS
TestStartDirector_CarriesTheConfigIntoThePrompt PASS
TestStartDirector_CarriesTheConfigIntoThePrompt/a_relative_config_path PASS
TestStartDirector_CarriesTheConfigIntoThePrompt/no_config_path PASS
TestStartDirector_RefusesBeforeWritingOrCallingHerdr/director.jsonl_unreadable PASS
```

16 of them are the tests cycle 2 added (5 top-level, 11 subtests: org 4 and 9, cli 1 and 2). `TestPlanDigest_MatchesScript/archive/2026-10-15-plan-visual-easy.md` is the plan PR #230 archived, which the merge brought in. Removed lines: none. The three SKIPs are the same as cycle 1's.

Earlier flaky tests (go1.25.8, then go1.26.0 with -race):

```text
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (1.54s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.31s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.51s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.62s)
--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.61s)
--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.89s)
--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.17s)
--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.76s)
```

#### C2-2.2 Race detector (go1.26.0)

The org and cli run above, rc 0, 249 s wall (the two packages ran in parallel), 0 `DATA RACE` lines:

```text
ok  	github.com/yoshpy-dev/ralph/internal/org	102.012s
ok  	github.com/yoshpy-dev/ralph/internal/cli	241.835s
```

#### C2-2.3 AC13: no real herdr or osascript, nothing in the repo's ledger

- `$SCRATCH/tds2/canary/bin/{herdr,osascript}` log their argv to `$SCRATCH/tds2/canary/calls.log` and exit 97 (unlike cycle 1's canary, the herdr one has no `--version` pass-through). They were first on PATH for run-test.sh, the three Go runs above, every mutation run, and the probe tests. `calls.log` had 0 lines after all of them. Typed by hand afterwards from the same PATH, both log and exit 97:

```text
$ PATH=$SCRATCH/tds2/canary/bin:$PATH sh -c 'herdr --version; echo "herdr rc=$?"; osascript -e 1; echo "osascript rc=$?"'
herdr rc=97
osascript rc=97
$ cat calls.log
1792069323 herdr --version
1792069324 osascript -e 1
```

- After the runs: `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` and `.../.claude/worktrees/org-director-start/.harness/state/org` do not exist; `find ~/MyDev/github.com/yoshpy-dev/ralph -name director.jsonl` finds none

#### C2-2.4 Coverage (go1.25.8, each package's own tests)

From the go1.25.8 run (`go tool cover -func` total 91.2%):

```text
ok  	github.com/yoshpy-dev/ralph/internal/cli	245.612s	coverage: 87.8% of statements
ok  	github.com/yoshpy-dev/ralph/internal/config	4.147s	coverage: 93.3% of statements
ok  	github.com/yoshpy-dev/ralph/internal/insights	4.080s	coverage: 86.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org	95.866s	coverage: 94.6% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	6.767s	coverage: 93.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	5.262s	coverage: 100.0% of statements
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	5.665s	coverage: 75.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	8.597s	coverage: 91.2% of statements
```

Statements of the changed files, from `cover125.out` (each block counted once):

```text
internal/org/director_start.go: 216/231 (93.5%); uncovered blocks (lines): 59-61, 129-131, 137-139, 204-206, 249-251, 322-324, 497-498, 549-551, 552-554, 556-558, 618-620, 647-648, 665-665, 672-673, 674-675
internal/org/prompts.go: 45/46 (97.8%); uncovered blocks (lines): 169-171
internal/cli/org_director_headless.go: 95/98 (96.9%); uncovered blocks (lines): 170-172, 182-184, 192-194
internal/org/director_stop.go: 132/138 (95.7%); uncovered blocks (lines): 169-170, 187-189, 239-240, 243-244, 372-374, 468-468
```

`director_start.go` 556-558 is the `return` of the `--config` path's `absPath` failure (mutant K15). The other uncovered blocks are cycle 1's (the report's Coverage section), less the two cycle 1's tests now reach: the first `Store.Read` failure (D4) and the failed write of attached (E1); `org_director_headless.go`'s `startAttachedError` with `cur == nil` (P2) is now reached too. `prompts.go` 169-171 is the embedded template's read failure.

### C2-3 HEAD binary against a stub herdr

`GOTOOLCHAIN=go1.25.8 go build -o $SCRATCH/tds2/bin/ralph ./cmd/ralph  # at 5dce71de; bin/ralph version: ralph dev (unknown unknown)`. `$SCRATCH/tds2/live/bin` holds `ralph` (a symlink to that binary), cycle 1's stateful stub `herdr` (section 3, copied unchanged), and an `osascript` that logs and exits 0. The script runs every ralph under `env -i` with that directory, `/usr/bin`, and `/bin` on PATH; the ledger is `<run>/ledger` (`--state-dir`), start's cwd `<run>/cwd` (no ralph.toml), and the rendered lines run with `sh -c` from `<run>/elsewhere/sub`.

#### C2-3.1 Script (`$SCRATCH/tds2/live/probe.sh`)

```bash
#!/bin/bash
# /test cycle 2 of org-director-start: HEAD-binary check of the director
# prompt that `ralph org director start` renders, without and with --config,
# against the CLI tests' stateful stub herdr. env -i, no real herdr or
# osascript on PATH, scratch --state-dir; the repo's .harness/state/org is
# never named. The rendered status and stop lines are run as written with
# sh -c from a directory other than start's.
set -u
T=$SCRATCH/tds2
B=$T/live/bin
R=$T/live/run
rm -rf "$R"
mkdir -p "$R/home" "$R/herdr" "$R/ledger" "$R/cwd" "$R/elsewhere/sub" "$R/cfg dir"
L=$R/ledger
C="$R/cfg dir/it's ralph.toml"
cat > "$C" <<'EOF'
[org.director]
enabled = true

[org.permissions]
default = "autonomous"

[org.permissions.roles]
director = "guarded"
EOF
: > "$T/live/osascript.log"
: > "$R/herdr.log"
cd "$R/cwd" || exit 1
E=(env -i HOME="$R/home" PATH="$B:/usr/bin:/bin" ORG_HERDR_LOG="$R/herdr.log" ORG_STUB_HERDR_STATE="$R/herdr")
red() { sed -e "s#$R#<run>#g" -e "s#$HOME#~#g"; }
step() { printf '\n### %s\n' "$*"; }
run() { # run [VAR=val ...] -- ralph args...
  local pre=()
  while [ "$1" != "--" ]; do pre+=("$1"); shift; done
  shift
  printf '$ (cwd %s) %s ralph %s\n' "$(pwd)" "${pre[*]+${pre[*]}}" "$*" | red
  "${E[@]}" ${pre[@]+"${pre[@]}"} ralph "$@" > "$R/o" 2> "$R/e"
  local rc=$?
  sed 's/^/  out: /' "$R/o" | red
  sed 's/^/  err: /' "$R/e" | red
  echo "  rc=$rc"
}
sh_line() { # sh_line <line> [dir]: run a line copied from the prompt with sh -c, from $R/elsewhere/sub (or dir)
  local d=${2:-$R/elsewhere/sub}
  printf '$ (cwd %s) sh -c %s\n' "$d" "$1" | red
  (cd "$d" && "${E[@]}" sh -c "$1") > "$R/o" 2> "$R/e"
  local rc=$?
  sed 's/^/  out: /' "$R/o" | red
  sed 's/^/  err: /' "$R/e" | red
  echo "  rc=$rc"
}
latest_prompt() { ls -t "$L"/prompts/director_*.md | head -1; }
show_prompt() { # the header lines, every line naming a ralph command, and the --config count
  local p=$1
  echo "  prompt file: $(printf '%s' "$p" | red)"
  sed -n '1,7p' "$p" | sed 's/^/  | /' | red
  echo "  lines naming ralph (line number: text):"
  grep -n 'ralph org\|ralph status' "$p" | sed 's/^/  | /' | red
  echo "  lines holding --config: $(grep -c -- '--config' "$p")"
  echo "  occurrences of --config: $(grep -o -- '--config' "$p" | wc -l | tr -d ' ')"
}
first_status_line() { awk 'prev=="```sh" && /^ralph org director status/ {print; exit} {prev=$0}' "$1"; }
stop_line() { sed -n 's/^  \(ralph org director stop --id .*\)$/\1/p' "$1"; }

step "W0 the long-running watch"
run -- org watch --all --detach --state-dir "$L"
sleep 2
run -- status --state-dir "$L"

step "W1 start without --config"
run -- org director start --driver claude --model haiku --state-dir "$L"
P1=$(latest_prompt)
show_prompt "$P1"
step "W1a the first-step status line, run as written from another directory"
sh_line "$(first_status_line "$P1")"
step "W1b the stop line, run as written from another directory"
sh_line "$(stop_line "$P1")"

step "W2 start with a relative --config (a path with a space and a single quote)"
run -- org director start --driver claude --model haiku --state-dir "$L" --config "../cfg dir/it's ralph.toml"
P2=$(latest_prompt)
show_prompt "$P2"
echo "  the status lines' --config word equals the file start was given, made absolute:"
printf '    %s\n' "$(cd "$R/cwd" && python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "../cfg dir/it's ralph.toml")" | red
step "W2a the first-step status line, run as written from another directory"
sh_line "$(first_status_line "$P2")"
step "W2b control: the same status with the relative --config as start was given it, from that other directory (where it names <run>/elsewhere/cfg dir/it's ralph.toml, which does not exist)"
sh_line "ralph org director status --state-dir '$L' --config \"../cfg dir/it's ralph.toml\""
step "W2d the rendered status reads start's file: flip its [org.director].enabled to false; the rendered line says false, a status without --config in start's cwd (no ralph.toml) says true"
cp "$C" "$R/cfg.bak"
printf '[org.director]\nenabled = false\n' > "$C"
sh_line "$(first_status_line "$P2")"
run -- org director status --state-dir "$L"
cp "$R/cfg.bak" "$C"
echo "  restored the --config file: $(cmp -s "$C" "$R/cfg.bak" && echo same bytes as before)"
step "W2c the stop line, run as written from another directory"
sh_line "$(stop_line "$P2")"

step "W3 start with an absolute --config"
run -- org director start --driver claude --model haiku --state-dir "$L" --config "$C"
P3=$(latest_prompt)
show_prompt "$P3"
echo "  W2 and W3 prompts differ only in id, token, and prompt-file name lines:"
diff <(sed -e 's/d[0-9]\{1,\}/dN/g' "$P2") <(sed -e 's/d[0-9]\{1,\}/dN/g' "$P3") > "$R/d23" && echo "    no difference after d<N> is normalised" || sed 's/^/    /' "$R/d23" | red
run -- org director stop --id d3 --state-dir "$L"

step "W9 herdr calls, events, cleanup"
echo "  herdr calls (stub log):"; sed 's/^/  herdr: /' "$R/herdr.log" | red
printf '  director.jsonl: '; python3 -c '
import json,sys
print(", ".join(e["event"]+" "+e["id"] for e in map(json.loads, open(sys.argv[1]))))' "$L/director.jsonl"
run -- org watch --stop --state-dir "$L"
sleep 1
echo "  processes naming the run dir left: $(pgrep -f "$R" | wc -l | tr -d ' ')"
echo "  osascript calls: $(wc -l < "$T/live/osascript.log" | tr -d ' ')"
echo "  ledger: $(ls "$L" | tr '\n' ' ')"
```

#### C2-3.2 Output

`$` lines are what the script ran, `out:` and `err:` lines its stdout and stderr. Outside W1a, the eleven per-registration field lines of `director status` (`driver:` through `pane_id:`) are dropped; they match W1a's apart from the id's values.

```text
### W0 the long-running watch
$ (cwd <run>/cwd)  ralph org watch --all --detach --state-dir <run>/ledger
  out: started ralph org watch --all (pid 44016); log: <run>/ledger/watch.log
  rc=0
$ (cwd <run>/cwd)  ralph status --state-dir <run>/ledger
  out: no org runtime state found (state-dir=<run>/ledger (source: flag)) — run `ralph org spawn` to start one.
  out: org watch: running (pid 44016, mark updated 2026-10-15T12:44:04Z)
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
  out: merge guarantee: unknown (not checked: run ralph doctor)
  out: Run `ralph doctor` to check environment readiness.
  rc=0

### W1 start without --config
$ (cwd <run>/cwd)  ralph org director start --driver claude --model haiku --state-dir <run>/ledger
  out: started d1 (driver=claude, model=haiku, permission=autonomous, pane=pane-stub-1, workspace=ws-stub-1)
  out: talk to the director in herdr: pane pane-stub-1, in the tab director_032eed3a3b69 of the workspace labelled ralph_director_032eed3a3b69
  out: stop it with: ralph org director stop --id d1 --state-dir '<run>/ledger'
  rc=0
  prompt file: <run>/ledger/prompts/director_032eed3a3b69.md
  | # 役割: director(headless)
  |
  | - id: d1 / driver: claude
  | - 台帳: '<run>/ledger'
  | - 設定: main worktree の ralph.toml(台帳が main worktree のものでなければ、cwd の ralph.toml)
  | - cwd: <run>/cwd
  |
  lines naming ralph (line number: text):
  | 13:ralph org director status --state-dir '<run>/ledger'
  | 23:あなたは `ralph org director start` が立てた headless の director です。
  | 27:- skill の「昇格」の手順のうち、見張りの確かめ(`ralph status` の
  | 28:  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
  | 30:- 会話の要約のあとに `ralph org director status --state-dir '<run>/ledger'` で
  | 33:- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '<run>/ledger'` を付ける。
  | 34:  `ralph status` には `--state-dir '<run>/ledger'` だけを付ける。cwd が
  | 44:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
  | 51:  ralph org director stop --id d1 --state-dir '<run>/ledger'
  lines holding --config: 0
  occurrences of --config: 0

### W1a the first-step status line, run as written from another directory
$ (cwd <run>/elsewhere/sub) sh -c ralph org director status --state-dir '<run>/ledger'
  out: director: attached d1 (driver=claude, headless running, model=haiku, pane pane-stub-1, workspace ws-stub-1, attached at 2026-10-15T12:44:06Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: enabled: true
  out: id: d1
  out: driver: claude
  out: wake: pull
  out: attached_at: 2026-10-15T12:44:06Z
  out: user: -
  out: cwd: <run>/cwd
  out: herdr_pane: no
  out: launch: herdr
  out: state: running
  out: model: haiku
  out: permission_mode: autonomous
  out: token: 032eed3a3b69
  out: workspace_id: ws-stub-1
  out: pane_id: pane-stub-1
  rc=0

### W1b the stop line, run as written from another directory
$ (cwd <run>/elsewhere/sub) sh -c ralph org director stop --id d1 --state-dir '<run>/ledger'
  out: stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
  out: the director is no longer attached
  rc=0

### W2 start with a relative --config (a path with a space and a single quote)
$ (cwd <run>/cwd)  ralph org director start --driver claude --model haiku --state-dir <run>/ledger --config ../cfg dir/it's ralph.toml
  out: started d2 (driver=claude, model=haiku, permission=guarded, pane=pane-stub-2, workspace=ws-stub-2)
  out: talk to the director in herdr: pane pane-stub-2, in the tab director_6212d182a9aa of the workspace labelled ralph_director_6212d182a9aa
  out: stop it with: ralph org director stop --id d2 --state-dir '<run>/ledger'
  rc=0
  prompt file: <run>/ledger/prompts/director_6212d182a9aa.md
  | # 役割: director(headless)
  |
  | - id: d2 / driver: claude
  | - 台帳: '<run>/ledger'
  | - 設定: '<run>/cfg dir/it'\''s ralph.toml'(start の --config)
  | - cwd: <run>/cwd
  |
  lines naming ralph (line number: text):
  | 13:ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'
  | 23:あなたは `ralph org director start` が立てた headless の director です。
  | 27:- skill の「昇格」の手順のうち、見張りの確かめ(`ralph status` の
  | 28:  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
  | 30:- 会話の要約のあとに `ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'` で
  | 33:- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'` を付ける。
  | 34:  `ralph status` には `--state-dir '<run>/ledger'` だけを付ける。cwd が
  | 44:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
  | 51:  ralph org director stop --id d2 --state-dir '<run>/ledger'
  lines holding --config: 4
  occurrences of --config: 4
  the status lines' --config word equals the file start was given, made absolute:
    <run>/cfg dir/it's ralph.toml

### W2a the first-step status line, run as written from another directory
$ (cwd <run>/elsewhere/sub) sh -c ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'
  out: director: attached d2 (driver=claude, headless running, model=haiku, pane pane-stub-2, workspace ws-stub-2, attached at 2026-10-15T12:44:06Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: enabled: true
  out: id: d2
  out: state: running
  out: permission_mode: guarded
  rc=0

### W2b control: the same status with the relative --config as start was given it, from that other directory (where it names <run>/elsewhere/cfg dir/it's ralph.toml, which does not exist)
$ (cwd <run>/elsewhere/sub) sh -c ralph org director status --state-dir '<run>/ledger' --config "../cfg dir/it's ralph.toml"
  out: director: attached d2 (driver=claude, headless running, model=haiku, pane pane-stub-2, workspace ws-stub-2, attached at 2026-10-15T12:44:06Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: enabled: true
  out: id: d2
  out: state: running
  out: permission_mode: guarded
  rc=0

### W2d the rendered status reads start's file: flip its [org.director].enabled to false; the rendered line says false, a status without --config in start's cwd (no ralph.toml) says true
$ (cwd <run>/elsewhere/sub) sh -c ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'
  out: director: disabled ([org.director].enabled = false; registration d2 is still recorded, headless running, pane pane-stub-2, workspace ws-stub-2: stop it with ralph org director stop --id d2 --state-dir '<run>/ledger') -- ralph org escalate sends every item to the human at once
  out: enabled: false
  out: id: d2
  out: state: running
  out: permission_mode: guarded
  rc=0
$ (cwd <run>/cwd)  ralph org director status --state-dir <run>/ledger
  out: director: attached d2 (driver=claude, headless running, model=haiku, pane pane-stub-2, workspace ws-stub-2, attached at 2026-10-15T12:44:06Z) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
  out: enabled: true
  out: id: d2
  out: state: running
  out: permission_mode: guarded
  rc=0
  restored the --config file: same bytes as before

### W2c the stop line, run as written from another directory
$ (cwd <run>/elsewhere/sub) sh -c ralph org director stop --id d2 --state-dir '<run>/ledger'
  out: stopped d2 (ctrl_c=ok, pane=closed, workspace=closed)
  out: the director is no longer attached
  rc=0

### W3 start with an absolute --config
$ (cwd <run>/cwd)  ralph org director start --driver claude --model haiku --state-dir <run>/ledger --config <run>/cfg dir/it's ralph.toml
  out: started d3 (driver=claude, model=haiku, permission=guarded, pane=pane-stub-3, workspace=ws-stub-3)
  out: talk to the director in herdr: pane pane-stub-3, in the tab director_7320fb2b1ef0 of the workspace labelled ralph_director_7320fb2b1ef0
  out: stop it with: ralph org director stop --id d3 --state-dir '<run>/ledger'
  rc=0
  prompt file: <run>/ledger/prompts/director_7320fb2b1ef0.md
  | # 役割: director(headless)
  |
  | - id: d3 / driver: claude
  | - 台帳: '<run>/ledger'
  | - 設定: '<run>/cfg dir/it'\''s ralph.toml'(start の --config)
  | - cwd: <run>/cwd
  |
  lines naming ralph (line number: text):
  | 13:ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'
  | 23:あなたは `ralph org director start` が立てた headless の director です。
  | 27:- skill の「昇格」の手順のうち、見張りの確かめ(`ralph status` の
  | 28:  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
  | 30:- 会話の要約のあとに `ralph org director status --state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'` で
  | 33:- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'` を付ける。
  | 34:  `ralph status` には `--state-dir '<run>/ledger'` だけを付ける。cwd が
  | 44:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
  | 51:  ralph org director stop --id d3 --state-dir '<run>/ledger'
  lines holding --config: 4
  occurrences of --config: 4
  W2 and W3 prompts differ only in id, token, and prompt-file name lines:
    no difference after d<N> is normalised
$ (cwd <run>/cwd)  ralph org director stop --id d3 --state-dir <run>/ledger
  out: stopped d3 (ctrl_c=ok, pane=closed, workspace=closed)
  out: the director is no longer attached
  rc=0

### W9 herdr calls, events, cleanup
  herdr calls (stub log):
  herdr: workspace create --cwd <run>/cwd --label ralph_director_032eed3a3b69
  herdr: tab create --workspace ws-stub-1 --cwd <run>/cwd --label director_032eed3a3b69
  herdr: agent start director_032eed3a3b69 --kind claude --pane pane-stub-1 --timeout 60000 -- --permission-mode bypassPermissions --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_032eed3a3b69.md
  herdr: pane get pane-stub-1
  herdr: tab get tab-stub-1
  herdr: workspace get ws-stub-1
  herdr: workspace get ws-stub-1
  herdr: pane send-keys pane-stub-1 C-c
  herdr: pane close pane-stub-1
  herdr: workspace close ws-stub-1
  herdr: workspace create --cwd <run>/cwd --label ralph_director_6212d182a9aa
  herdr: tab create --workspace ws-stub-2 --cwd <run>/cwd --label director_6212d182a9aa
  herdr: agent start director_6212d182a9aa --kind claude --pane pane-stub-2 --timeout 60000 -- --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_6212d182a9aa.md
  herdr: pane get pane-stub-2
  herdr: tab get tab-stub-2
  herdr: workspace get ws-stub-2
  herdr: workspace get ws-stub-2
  herdr: pane send-keys pane-stub-2 C-c
  herdr: pane close pane-stub-2
  herdr: workspace close ws-stub-2
  herdr: workspace create --cwd <run>/cwd --label ralph_director_7320fb2b1ef0
  herdr: tab create --workspace ws-stub-3 --cwd <run>/cwd --label director_7320fb2b1ef0
  herdr: agent start director_7320fb2b1ef0 --kind claude --pane pane-stub-3 --timeout 60000 -- --model haiku 役割指示を読み込んで従ってください: <run>/ledger/prompts/director_7320fb2b1ef0.md
  herdr: pane get pane-stub-3
  herdr: tab get tab-stub-3
  herdr: workspace get ws-stub-3
  herdr: workspace get ws-stub-3
  herdr: pane send-keys pane-stub-3 C-c
  herdr: pane close pane-stub-3
  herdr: workspace close ws-stub-3
  director.jsonl: attached d1, workspace_created d1, pane_created d1, launched d1, detached d1, stopped d1, attached d2, workspace_created d2, pane_created d2, launched d2, detached d2, stopped d2, attached d3, workspace_created d3, pane_created d3, launched d3, detached d3, stopped d3
$ (cwd <run>/cwd)  ralph org watch --stop --state-dir <run>/ledger
  out: stopped ralph org watch --all (pid 44016)
  rc=0
  processes naming the run dir left: 0
  osascript calls: 0
  ledger: director.jsonl director.lock prompts watch-alive.json watch.lock watch.log
```

### C2-4 Mutation (go1.26.0, `go test -overlay`)

#### C2-4.1 Harness

`$SCRATCH/tds2/mut/mutate.py` is cycle 1's harness (section 5.1) with its output under `$SCRATCH/tds2/mut/out/` and the cycle 2 canary; `mutants.py` holds the edits below. An edit is refused unless `old` occurs exactly once in the file. `-run` regexes are cycle 1's (org: `Director|StopAll|Spawn_DirectorRole|RenderRolePrompt|CodexWritableRoot`; cli: `OrgDirector|StatusCmd_Director|OrgStopAll|OrgSpawn_RoleDirector|OrgLegacyLedger`). `prompts/director.md` is go:embed'ed; the overlay replaces it in the embed as well (K3, K8-K11, K14 are red through it). X0, X0t, and X0c pass an unchanged file through the overlay as controls. K1c, K3c, K2cc, and K15c are K1, K3, K2c, and K15's edits run against `./internal/cli`, whose tests read the prompt file the org package renders. All edits applied (no BADEDIT, no BUILD failure).

#### C2-4.2 Results

26 mutants: 24 red, 2 survived (K15, K16); the 3 cross-package reruns of red mutants are red; the 3 controls pass. "failing top-level tests" drops the `Test` prefix.

| ID | pkg | file | mutation | result (filtered) | failing top-level tests |
| --- | --- | --- | --- | --- | --- |
| X0 | org | director_start.go | org: director_start.go unchanged through -overlay | SURVIVED 2s |  |
| X0t | org | prompts/director.md | org: prompts/director.md unchanged through -overlay | SURVIVED 3s |  |
| X0c | cli | org_director_headless.go | cli: org_director_headless.go unchanged through -overlay | SURVIVED 41s |  |
| K1 | org | director_start.go | writePrompt drops ConfigPath from the prompt vars | red 7s | StartDirector_CarriesTheConfigIntoThePrompt |
| L1 | cli | org_director_headless.go | the CLI does not pass --config to StartDirector | red 41s | OrgDirectorStart_ReadsTheMainWorktreeRalphToml |
| K2 | org | director_start.go | checkDirectorStart does not make ConfigPath absolute (render refuses the relative path) | red 6s | StartDirector_CarriesTheConfigIntoThePrompt |
| K2b | org | director_start.go, prompts.go | K2 and K4 together: a relative ConfigPath reaches the prompt as given | red 6s | RenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders, StartDirector_CarriesTheConfigIntoThePrompt |
| K2c | org | director_start.go | a relative ConfigPath is made absolute against the director's Cwd, not the caller's directory | red 5s | StartDirector_CarriesTheConfigIntoThePrompt |
| L3 | cli | org_director_headless.go | the CLI joins a relative --config to the director's cwd (the main worktree root) before StartDirector | red 39s | OrgDirectorStart_ReadsTheMainWorktreeRalphToml |
| K3 | org | prompts/director.md | the template's stop line renders {{ORG_FLAGS}} (gets --config) | red 3s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig, StartDirector_CarriesTheConfigIntoThePrompt |
| K4 | org | prompts.go | RenderDirectorPrompt accepts a relative ConfigPath | red 6s | RenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders |
| K5 | org | prompts.go | the config path is not shell-quoted | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig, StartDirector_CarriesTheConfigIntoThePrompt |
| K6 | org | prompts.go | --config before --state-dir in {{ORG_FLAGS}} | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig, StartDirector_CarriesTheConfigIntoThePrompt |
| K7 | org | prompts.go | the header keeps the no-config text with a ConfigPath | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig |
| K8 | org | prompts/director.md | the status after a summary keeps --state-dir only | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig |
| K9 | org | prompts/director.md | the first-step status keeps --state-dir only | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig, StartDirector_CarriesTheConfigIntoThePrompt |
| K10 | org | prompts/director.md | the instruction sentence says --state-dir only | red 7s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig |
| K11 | org | prompts/director.md | ralph status is told to carry {{ORG_FLAGS}} | red 7s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig |
| K12 | org | director_start.go | an empty ConfigPath is made absolute too (becomes the cwd) | red 8s | StartDirector_CarriesTheConfigIntoThePrompt |
| K13 | org | prompts.go | RenderDirectorPrompt takes the --config branch for an empty ConfigPath | red 5s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig, RenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure, StartDirector_CarriesTheConfigIntoThePrompt |
| K14 | org | prompts/director.md | the template's 設定 header line is removed | red 6s | RenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig |
| D4 | org | director_start.go | an unreadable director.jsonl at the early look is ignored (the gate runs) | red 6s | StartDirector_RefusesBeforeWritingOrCallingHerdr |
| E1 | org | director_start.go | a failed attached write does not stop start (herdr is called) | red 7s | StartDirector_AFailedWriteOfAttachedCallsNoHerdr |
| E12 | org | director_start.go | a failed workspace create keeps herdr's output as the workspace id | red 7s | StartDirector_AFailedCreatesOutputIsNotAnID |
| E13 | org | director_start.go | a failed tab create keeps herdr's output as the pane id | red 6s | StartDirector_AFailedCreatesOutputIsNotAnID |
| P2 | cli | org_director_headless.go | start refused, current read failed or none: status without --state-dir | red 31s | OrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus |
| K1c | cli | director_start.go | writePrompt drops ConfigPath from the prompt vars (cli package) | red 70s | OrgDirectorStart_ReadsTheMainWorktreeRalphToml |
| K3c | cli | prompts/director.md | the template's stop line renders {{ORG_FLAGS}} (gets --config) (cli package) | red 65s | OrgDirectorStart_ReadsTheMainWorktreeRalphToml |
| K2cc | cli | director_start.go | a relative ConfigPath is made absolute against the director's Cwd, not the caller's directory (cli package) | red 71s | OrgDirectorStart_ReadsTheMainWorktreeRalphToml |
| K15 | org | director_start.go | a --config path that cannot be made absolute does not stop start | SURVIVED 7s |  |
| K15c | cli | director_start.go | a --config path that cannot be made absolute does not stop start (cli package) | SURVIVED 266s, whole package only |  |
| L4 | cli | org_director_headless.go | the stop command start prints gets --config when start had one | red 69s | OrgDirectorStart_LaunchesRegistersAndStops |
| K16 | org | prompts.go, prompts_test.go | the 設定 line's marker renamed in prompts.go and prompts_test.go together (the skills keep the old one) | SURVIVED 6s |  |

Survivors against whole packages (`-run` dropped):

| run | result | time |
| --- | --- | --- |
| K15 org | SURVIVED | 115s |
| K15c cli | SURVIVED | 266s |
| K16 org | SURVIVED | 80s |

K16 was not run against cli: no cli test reads the prompt's 設定 line (`grep 設定 internal/cli/*_test.go` finds none), and no shell test or script names the marker (`grep -rl 'start の --config\|設定:' tests/ scripts/` finds none).

#### C2-4.3 Edits

- K1 (./internal/org): writePrompt drops ConfigPath from the prompt vars
  - `internal/org/director_start.go`: `'StateDir: r.stateDir, ConfigPath: r.p.ConfigPath, Cwd: r.p.Cwd})'` -> `'StateDir: r.stateDir, Cwd: r.p.Cwd})'`
- L1 (./internal/cli): the CLI does not pass --config to StartDirector
  - `internal/cli/org_director_headless.go`: `'Cwd: cwd, ConfigPath: configPath, TimeoutMS: timeoutMS'` -> `'Cwd: cwd, TimeoutMS: timeoutMS'`
- K2 (./internal/org): checkDirectorStart does not make ConfigPath absolute (render refuses the relative path)
  - `internal/org/director_start.go`: `'\tif p.ConfigPath != "" {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'` -> `'\tif p.ConfigPath != "" && false {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'`
- K2b (./internal/org): K2 and K4 together: a relative ConfigPath reaches the prompt as given
  - `internal/org/director_start.go`: `'\tif p.ConfigPath != "" {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'` -> `'\tif p.ConfigPath != "" && false {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'`
  - `internal/org/prompts.go`: `'\tif vars.ConfigPath != "" && !filepath.IsAbs(vars.ConfigPath) {'` -> `'\tif false && vars.ConfigPath != "" && !filepath.IsAbs(vars.ConfigPath) {'`
- K2c (./internal/org): a relative ConfigPath is made absolute against the director's Cwd, not the caller's directory
  - `internal/org/director_start.go`: `'\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'` -> `'\t\tif p.ConfigPath, err = func() (string, error) {\n\t\t\tif filepath.IsAbs(p.ConfigPath) {\n\t\t\t\treturn p.ConfigPath, nil\n\t\t\t}\n\t\t\treturn filepath.Join(p.Cwd, p.ConfigPath), nil\n\t\t}(); err != nil {'`
- L3 (./internal/cli): the CLI joins a relative --config to the director's cwd (the main worktree root) before StartDirector
  - `internal/cli/org_director_headless.go`: `'Cwd: cwd, ConfigPath: configPath, TimeoutMS: timeoutMS'` -> `'Cwd: cwd, ConfigPath: func() string {\n\t\t\tif configPath == "" || strings.HasPrefix(configPath, "/") {\n\t\t\t\treturn configPath\n\t\t\t}\n\t\t\treturn cwd + "/" + configPath\n\t\t}(), TimeoutMS: timeoutMS'`
- K3 (./internal/org): the template's stop line renders {{ORG_FLAGS}} (gets --config)
  - `internal/org/prompts/director.md`: `'  ralph org director stop --id {{ID}} --state-dir {{STATE_DIR}}\n'` -> `'  ralph org director stop --id {{ID}} {{ORG_FLAGS}}\n'`
- K4 (./internal/org): RenderDirectorPrompt accepts a relative ConfigPath
  - `internal/org/prompts.go`: `'\tif vars.ConfigPath != "" && !filepath.IsAbs(vars.ConfigPath) {'` -> `'\tif false && vars.ConfigPath != "" && !filepath.IsAbs(vars.ConfigPath) {'`
- K5 (./internal/org): the config path is not shell-quoted
  - `internal/org/prompts.go`: `'\t\tconfig := shellQuote(vars.ConfigPath)'` -> `'\t\tconfig := vars.ConfigPath'`
- K6 (./internal/org): --config before --state-dir in {{ORG_FLAGS}}
  - `internal/org/prompts.go`: `'\t\torgFlags += " --config " + config'` -> `'\t\torgFlags = "--config " + config + " " + orgFlags'`
- K7 (./internal/org): the header keeps the no-config text with a ConfigPath
  - `internal/org/prompts.go`: `'\t\tconfigLine = config + "(start の --config)"\n'` -> `''`
- K8 (./internal/org): the status after a summary keeps --state-dir only
  - `internal/org/prompts/director.md`: `` '会話の要約のあとに `ralph org director status {{ORG_FLAGS}}` で' `` -> `` '会話の要約のあとに `ralph org director status --state-dir {{STATE_DIR}}` で' ``
- K9 (./internal/org): the first-step status keeps --state-dir only
  - `internal/org/prompts/director.md`: `` '```sh\nralph org director status {{ORG_FLAGS}}\n```' `` -> `` '```sh\nralph org director status --state-dir {{STATE_DIR}}\n```' ``
- K10 (./internal/org): the instruction sentence says --state-dir only
  - `internal/org/prompts/director.md`: `` 'どれも `{{ORG_FLAGS}}` を付ける。' `` -> `` 'どれも `--state-dir {{STATE_DIR}}` を付ける。' ``
- K11 (./internal/org): ralph status is told to carry {{ORG_FLAGS}}
  - `internal/org/prompts/director.md`: `` '`ralph status` には `--state-dir {{STATE_DIR}}` だけを付ける。' `` -> `` '`ralph status` には `{{ORG_FLAGS}}` だけを付ける。' ``
- K12 (./internal/org): an empty ConfigPath is made absolute too (becomes the cwd)
  - `internal/org/director_start.go`: `'\tif p.ConfigPath != "" {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'` -> `'\tif true {\n\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'`
- K13 (./internal/org): RenderDirectorPrompt takes the --config branch for an empty ConfigPath
  - `internal/org/prompts.go`: `'\tif vars.ConfigPath != "" {\n\t\tconfig := shellQuote(vars.ConfigPath)'` -> `'\tif true {\n\t\tconfig := shellQuote(vars.ConfigPath)'`
- K14 (./internal/org): the template's 設定 header line is removed
  - `internal/org/prompts/director.md`: `'- 設定: {{CONFIG_LINE}}\n'` -> `''`
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
- K15 (./internal/org): a --config path that cannot be made absolute does not stop start
  - `internal/org/director_start.go`: `'\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil {'` -> `'\t\tif p.ConfigPath, err = absPath(p.ConfigPath); err != nil && false {'`
- L4 (./internal/cli): the stop command start prints gets --config when start had one
  - `internal/cli/org_director_headless.go`: `'rt.command("ralph org director stop --id "+inboxField(reg.ID)))'` -> `'rt.command("ralph org director stop --id "+inboxField(reg.ID)+map[bool]string{true: " --config " + configPath}[configPath != ""]))'`
- K16 (./internal/org): the 設定 line's marker renamed in prompts.go and prompts_test.go together (the skills keep the old one)
  - `internal/org/prompts.go`: `'configLine = config + "(start の --config)"'` -> `'configLine = config + "(--config of start)"'`
  - `internal/org/prompts_test.go`: `'"- 設定: " + configArg + "(start の --config)\\n"'` -> `'"- 設定: " + configArg + "(--config of start)\\n"'`

#### C2-4.4 Probe tests for the 2 survivors

`$SCRATCH/tds2/probe/zz_probe_c2_test.go`, read through the overlay as `internal/org/zz_probe_c2_test.go`, run with `-run 'TestProbe_C2_|TestRenderDirectorPrompt_' -v ./internal/org/` at HEAD, under K15, and under K16 (each mutant is the other probe's control), plus K7 for the first probe:

```go
package org

// Probe test for a /test cycle 2 gap of org-director-start. It is read
// through go test -overlay only and is never committed.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A --config path that cannot be made absolute ends start before anything
// is written or herdr is called, with the resolve error.
func TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	p := f.params(DirectorDriverClaude, "opus")
	p.ConfigPath = "my ralph.toml"
	orig := absPath
	absPath = func(s string) (string, error) {
		if s == "my ralph.toml" {
			return "", errors.New("stub: abs failed")
		}
		return orig(s)
	}
	defer func() { absPath = orig }()
	_, err := f.o.StartDirector(p)
	if err == nil || !strings.HasPrefix(err.Error(), "org: director: start: resolve the --config path: ") || !strings.Contains(err.Error(), "stub: abs failed") {
		t.Fatalf("err = %v, want the --config resolve error", err)
	}
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", f.h.calls)
	}
	if got := directorFileState(t, f.store); got != "<missing>" {
		t.Fatalf("director.jsonl = %q, want it missing", got)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "prompts")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a refused start made the prompts dir (stat err %v)", err)
	}
}

// The /director skill tells a session to read the prompt's 設定 line and to
// carry its path as --config when the line ends with the marker
// RenderDirectorPrompt writes after the path. Each of the four SKILL.md
// copies must name that marker as the prompt writes it.
func TestProbe_C2_DirectorSkillNamesTheConfigLineMarker(t *testing.T) {
	vars := testDirectorPromptVars()
	vars.ConfigPath = "/x/ralph.toml"
	text := mustRenderDirectorPrompt(t, vars)
	prefix := "\n- 設定: " + shellQuote(vars.ConfigPath)
	i := strings.Index(text, prefix)
	if i < 0 {
		t.Fatalf("rendered prompt has no 設定 line naming %s:\n%s", shellQuote(vars.ConfigPath), text)
	}
	rest := text[i+len(prefix):]
	marker := rest[:strings.Index(rest, "\n")]
	if marker == "" {
		t.Fatalf("the 設定 line ends with no marker after the path:\n%s", text)
	}
	root := directorSkillRepoRoot(t)
	for _, rel := range []string{
		filepath.Join(".claude", "skills", "director", "SKILL.md"),
		filepath.Join(".agents", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".claude", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".agents", "skills", "director", "SKILL.md"),
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v)", rel, err)
		}
		if !strings.Contains(string(data), "`"+marker+"`") {
			t.Errorf("%s does not name the 設定 line's marker `%s`", rel, marker)
		}
	}
}
```

Results (`--- ` lines, the probe file's log lines, and the package line):

```text
== head
--- PASS: TestRenderDirectorPrompt_SubstitutesEveryPlaceholder (0.00s)
--- PASS: TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure (0.00s)
--- PASS: TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig (0.00s)
--- PASS: TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders (0.00s)
--- PASS: TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused (0.00s)
--- PASS: TestProbe_C2_DirectorSkillNamesTheConfigLineMarker (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	0.724s
== K15
--- PASS: TestRenderDirectorPrompt_SubstitutesEveryPlaceholder (0.00s)
--- PASS: TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure (0.00s)
--- PASS: TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig (0.00s)
--- PASS: TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders (0.00s)
    zz_probe_c2_test.go:31: err = <nil>, want the --config resolve error
--- FAIL: TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused (0.00s)
--- PASS: TestProbe_C2_DirectorSkillNamesTheConfigLineMarker (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/org	0.666s
FAIL
== K16
--- PASS: TestRenderDirectorPrompt_SubstitutesEveryPlaceholder (0.00s)
--- PASS: TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure (0.00s)
--- PASS: TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig (0.00s)
--- PASS: TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders (0.00s)
--- PASS: TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused (0.00s)
    zz_probe_c2_test.go:74: .claude/skills/director/SKILL.md does not name the 設定 line's marker `(--config of start)`
    zz_probe_c2_test.go:74: .agents/skills/director/SKILL.md does not name the 設定 line's marker `(--config of start)`
    zz_probe_c2_test.go:74: templates/base/.claude/skills/director/SKILL.md does not name the 設定 line's marker `(--config of start)`
    zz_probe_c2_test.go:74: templates/base/.agents/skills/director/SKILL.md does not name the 設定 line's marker `(--config of start)`
--- FAIL: TestProbe_C2_DirectorSkillNamesTheConfigLineMarker (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/org	0.732s
FAIL
== K7
--- PASS: TestRenderDirectorPrompt_SubstitutesEveryPlaceholder (0.00s)
--- PASS: TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure (0.00s)
--- FAIL: TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig (0.00s)
--- PASS: TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders (0.00s)
--- PASS: TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused (0.00s)
    zz_probe_c2_test.go:55: rendered prompt has no 設定 line naming '/x/ralph.toml':
--- FAIL: TestProbe_C2_DirectorSkillNamesTheConfigLineMarker (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/org	1.169s
FAIL
```

The first probe fails only under K15 (`err = <nil>`: K15's start succeeds with the ConfigPath emptied). The second fails under K16 for all four SKILL.md copies, while every existing `TestRenderDirectorPrompt_*` test passes under K16; under K7 it fails too, as K7's header has no path (K7 is red without it). Each passes under the other's mutant. Calls to the canary during the probe runs: 0 (the 2 lines in `calls.log` are C2-2.3's typed-by-hand check).

## Cycle 3

- Date: 2026-10-16 (JST 00:44-01:25; the UTC timestamps in the output read 2026-10-15T15:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD 3504ee30 (cycle 2 ran at 5dce71de, its report is 6556bce5). Go changed since cycle 2 in 2 commits: caf1d025 (D12: `ErrDirectorCodexWorkspaceWrite` in `checkDirectorStart`, `codexWritableRootArgs` leader-only again; D13: `checkStartAgmsgHome`, `agmsgHomeIsRelative` shared with `orgWatchChildAgmsgHome`, the `RALPH_ORG_AGMSG_HOME` warning) and 83287d39 (`CodexModeRunsUnderWorkspaceWrite`, doctor's `codexSeatModesPossible` no longer counts role director's edits/autonomous, the D13 refusal pinned in three places, help and wording)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0. No real herdr, codex, claude, or osascript ran. Load average 4-11 during the runs (other sessions)
- Redaction: as in cycles 1 and 2 (`$SCRATCH`, `~`, `$TMPDIR/`, `<run>`). Cycle 3's scratch files are under `$SCRATCH/ods3`. The wrapper's full log `docs/evidence/verify-2026-10-15-154510.log` and the raw outputs appended to `docs/evidence/test-2026-10-15-org-director-start.log` (from the line `===== /test cycle 3`) are gitignored and not committed
- Trailing blanks at the ends of output lines were stripped; nothing else in the output was changed apart from the redaction
- Nothing in the worktree was edited for the tests. Mutants and probe tests lived in `$SCRATCH/ods3` and reached the build only through `go test -overlay`

### C3-1 ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; templates/base/ralph.toml widened it to full)

Command: `PATH=$SCRATCH/ods3/canary/bin:$PATH ./scripts/run-test.sh > runtest.out 2>&1` (rc 0, 818 s wall, 4,654 lines). Filtered with the same command as section 1 (168 lines). Against C2-1's 168 lines, with the line numbers, the `N.NNs` timings, and the timestamp dropped, `diff` gives:

```text
168c168
< Evidence saved to: docs/evidence/verify-2026-10-15-123924.log
---
> Evidence saved to: docs/evidence/verify-2026-10-15-154510.log
```

41 suites, 41 "OK", every "FAIL:" line is 0 (25 of them, no other FAIL line). 4,283 lines start with PASS after leading blanks (cycle 2: 4,283). The one real SKIP is tests/test-secret-scan-branch.sh's (git 2.49.0); the other two `SKIP` matches are `SKIP: 0` summary lines. The Go part:

```text
4637:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
4641:?   	github.com/yoshpy-dev/ralph	[no test files]
4642:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4643:ok  	github.com/yoshpy-dev/ralph/internal/cli	317.283s
4644:ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
4645:ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
4646:ok  	github.com/yoshpy-dev/ralph/internal/org	128.213s
4647:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	0.896s
4648:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4649:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4650:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4652:==> All verifiers passed.
4654:Evidence saved to: docs/evidence/verify-2026-10-15-154510.log
```

`internal/insights` imports `internal/org`, so its `(cached)` is a result for the same org code from an earlier run, not proof that it is unaffected. All five `(cached)` packages ran with `-count=1` in C3-2.1 on both toolchains.

### C3-2 Go tests

Every Go run below had `$SCRATCH/ods3/canary/bin` first on PATH (C3-2.3).

#### C3-2.1 Per package, `-count=1 -v`

Counts are from the `--- PASS/FAIL/SKIP` lines (top-level tests, then subtests).

go1.25.8 (`GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile=cover125.out ./internal/...`, rc 0, 230 s wall):

```text
internal/cli	ok	226.077s	top 706 (pass 706 fail 0 skip 0)	sub 658 (pass 658 fail 0 skip 0)
internal/config	ok	1.221s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	2.082s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	100.828s	top 804 (pass 803 fail 0 skip 1)	sub 1696 (pass 1696 fail 0 skip 0)
internal/org/driver	ok	1.850s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	2.813s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	3.304s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	2.856s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.26.0, org and cli with the race detector (`go test -race -count=1 -v ./internal/org/ ./internal/cli/`, rc 0, 365 s wall, 0 `DATA RACE` lines):

```text
internal/org	ok	133.399s	top 804 (pass 803 fail 0 skip 1)	sub 1696 (pass 1696 fail 0 skip 0)
internal/cli	ok	355.563s	top 706 (pass 706 fail 0 skip 0)	sub 658 (pass 658 fail 0 skip 0)
```

go1.26.0, the other six packages (`go test -count=1 -v ./internal/config/ ./internal/insights/ ./internal/org/driver/ ./internal/org/protocol/ ./internal/scaffold/ ./internal/upgrade/`, rc 0):

```text
internal/config	ok	0.510s	top 37 (pass 37 fail 0 skip 0)	sub 34 (pass 34 fail 0 skip 0)
internal/insights	ok	1.480s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org/driver	ok	2.189s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	2.235s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	2.698s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	1.774s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.25.8, org and cli with the race detector (`GOTOOLCHAIN=go1.25.8 go test -race -count=1 -v ./internal/org/ ./internal/cli/`, rc 0, 323 s wall, 0 `DATA RACE` lines; it overlapped with the mutation runs of C3-4):

```text
internal/org	ok	119.029s	top 804 (pass 803 fail 0 skip 1)	sub 1696 (pass 1696 fail 0 skip 0)
internal/cli	ok	316.002s	top 706 (pass 706 fail 0 skip 0)	sub 658 (pass 658 fail 0 skip 0)
```

The sorted `<package> <name> <result>` lines of the go1.25.8 run (4,345) and of the two go1.26.0 runs together (4,345) are identical, and so are the org and cli lines of the two race runs (3,864 each). Against cycle 2's 4,318 lines (C2-2.1), 29 lines were added and 2 removed:

```text
24a25,29
> internal/cli TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted PASS
> internal/cli TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted/a_director_in_autonomous PASS
> internal/cli TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted/a_director_in_edits PASS
> internal/cli TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted/a_guarded_director_under_an_autonomous_default PASS
> internal/cli TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted/control:_a_seat_role_in_edits PASS
353a359,361
> internal/cli TestCodexSeatModesPossible/a_director_in_autonomous_under_a_guarded_default_is_not_counted PASS
> internal/cli TestCodexSeatModesPossible/a_director_in_edits_under_a_guarded_default_is_not_counted PASS
> internal/cli TestCodexSeatModesPossible/a_guarded_director_under_an_autonomous_default_still_counts_as_guarded PASS
461a470,481
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/a_./_path PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/a_bare_~ PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/a_relative_path PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/a_~/_path PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/an_absolute_path PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused/empty PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns/in_a_linked_worktree PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns/in_the_main_worktree_root PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns/on_a_ledger_with_no_main_worktree PASS
> internal/cli TestOrgDirectorStart_ARelativeAgmsgHomeWithoutConfigIsNotRefused PASS
467a488
> internal/cli TestOrgDirectorStart_HelpStatesTheCodexAndAgmsgHomeRules PASS
481a503
> internal/cli TestOrgDirectorStart_RefusedWithNothingWritten/codex_edits_with_codex_verified PASS
483a506
> internal/cli TestOrgDirectorStart_WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector PASS
3676d3698
< internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_autonomous,_ledger_inside_the_cwd PASS
3678c3700,3701
< internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_edits,_ledger_outside_the_cwd PASS
---
> internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_autonomous_with_codex_verified PASS
> internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_autonomous_with_codex_verified,_ledger_inside_the_cwd PASS
3679a3703
> internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_edits_with_codex_verified,_ledger_outside_the_cwd PASS
3680a3705,3706
> internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_guarded_with_codex_verified,_ledger_inside_the_cwd PASS
> internal/org TestStartDirector_PermissionModeOfRoleDirector/codex_guarded_with_codex_verified,_ledger_outside_the_cwd PASS
3692a3719
> internal/org TestStartDirector_RefusesBeforeWritingOrCallingHerdr/permission_rule_with_codex_verified PASS
```

The 2 removed lines are the two codex rows of `TestStartDirector_PermissionModeOfRoleDirector` that caf1d025 renamed (they expected a start that D12 now refuses). Added: 6 top-level tests (cli 6) and 23 subtests (cli 17, org 6). The three SKIPs are cycle 2's.

Earlier flaky tests (go1.25.8, go1.26.0 -race, go1.25.8 -race):

```text
go125.out: --- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.92s)
go125.out: --- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.86s)
go125.out: --- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (3.27s)
go125.out: --- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.69s)
race.out: --- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (2.64s)
race.out: --- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (1.84s)
race.out: --- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.95s)
race.out: --- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (7.04s)
race125.out: --- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (2.49s)
race125.out: --- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (2.18s)
race125.out: --- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (1.70s)
race125.out: --- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (7.07s)
```

The slowest `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` subtest was SIGTERM at 4.64 s (go1.25.8, -race), under the 5 s `gateKillGrace` and the 15 s budget.

#### C3-2.2 AC13: no real herdr or osascript, nothing in the repo's ledger

- `$SCRATCH/ods3/canary/bin/{herdr,osascript}` log their argv to `$SCRATCH/ods3/canary/calls.log` and exit 97 (no pass-through). They were first on PATH for run-test.sh, the four Go runs above, every mutation run, and the probe tests. `calls.log` had 0 lines after all of them. Typed by hand afterwards from the same PATH, both log and exit 97:

```text
$ PATH=$SCRATCH/ods3/canary/bin:$PATH sh -c 'herdr --version; echo "herdr rc=$?"; osascript -e 1; echo "osascript rc=$?"'
herdr rc=97
osascript rc=97
$ cat calls.log
1792081283 herdr --version
1792081284 osascript -e 1
```

- After the runs: `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` and `.../.claude/worktrees/org-director-start/.harness/state/org` do not exist; `find ~/MyDev/github.com/yoshpy-dev/ralph -name director.jsonl` finds none
- `~/.codex/config.toml` before the first run and after the last: mtime 1792054288 (Oct 15 17:51:28 2026), 33,673 bytes, sha256 `ac1a4821574050d3630118510c7dd591bf1839b947f2bb35fe4106e41b61796f` both times

#### C3-2.3 Coverage (go1.25.8, each package's own tests)

```text
ok  	github.com/yoshpy-dev/ralph/internal/cli	226.077s	coverage: 87.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/config	1.221s	coverage: 93.3% of statements
ok  	github.com/yoshpy-dev/ralph/internal/insights	2.082s	coverage: 86.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org	100.828s	coverage: 94.6% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	1.850s	coverage: 93.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.813s	coverage: 100.0% of statements
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.304s	coverage: 75.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.856s	coverage: 91.2% of statements
```

Statements of the changed files, from `cover125.out` (each block counted once):

```text
internal/org/director_start.go: 216/232 (93.1%); uncovered blocks (lines): 59-61, 129-131, 137-139, 204-206, 249-251, 322-324, 504-505, 549-552, 566-568, 569-571, 573-575, 635-637, 664-665, 682-682, 689-690, 691-692
internal/org/permissions.go: 29/32 (90.6%); uncovered blocks (lines): 130-131, 150-151, 203-205
internal/cli/org_director_headless.go: 102/105 (97.1%); uncovered blocks (lines): 189-191, 204-206, 214-216
internal/cli/org_watch.go: 223/240 (92.9%); uncovered blocks (lines): 91-93, 287-289, 327-329, 336-338, 381-383, 401-403, 403-405, 406-406, 417-419, 421-423, 426-428, 435-437, 542-544, 677-679, 685-687, 788-791
internal/cli/doctor_codex_writable_root.go: 271/276 (98.2%); uncovered blocks (lines): 140-140, 199-199, 212-214, 483-485, 571-573
```

The D12 block (`director_start.go` 541-548), `CodexModeRunsUnderWorkspaceWrite` (`permissions.go` 85-87), `checkStartAgmsgHome` (`org_director_headless.go` 253-265), `agmsgHomeIsRelative` (`org_watch.go` 395-397), and `codexSeatModesPossible` (`doctor_codex_writable_root.go` 344-368) are fully covered. `director_start.go` 549-552 is the error return of `permissionArgsForDriver` in `checkDirectorStart`: cycle 2's tests reached it with a codex director before codex_verified, which D12 now refuses first, so after caf1d025 only a mode outside autonomous, edits, and guarded could reach it, and `config.Load` refuses such a mode. The other uncovered blocks of these files are cycle 2's (shifted line numbers) or outside the diff (`launchOrgWatch` and the other production launchers in `org_watch.go`).

### C3-3 HEAD binary: `ralph doctor` and role director's modes

`GOTOOLCHAIN=go1.25.8 go build -o $SCRATCH/ods3/bin/ralph ./cmd/ralph` at 3504ee30, and the same from `git archive 83287d39^ | tar -x` (0bfbb054: caf1d025's Go, before doctor stopped counting role director) into `$SCRATCH/ods3/pre/bin/ralph`. Each case is a `git init` directory with its own `ralph.toml` (codex and claude in the pool, an absolute agmsg home holding `scripts/send.sh`). `ralph doctor` runs under `env -i` with a scratch HOME, CODEX_HOME, and TMPDIR, and PATH = the binary dir, `/usr/bin`, `/bin`. The codex configs exclude `/tmp` and `$TMPDIR` as writable roots, since `$SCRATCH` is under `/private/tmp` and would otherwise cover the agmsg store in every case.

#### C3-3.1 Script (`$SCRATCH/ods3/doctor/probe.sh`)

```bash
#!/bin/bash
# /test cycle 3 probe: `ralph doctor`'s "Codex sandbox (agmsg writable root)"
# check on the HEAD binary, for role director's edits/autonomous (not counted,
# C3-L4 of 83287d39) against a seat role's edits (counted) and a guarded
# director under an autonomous default with sandbox_mode = "workspace-write"
# (counted as guarded, reason (b)). Every ralph runs under env -i with a
# scratch HOME and CODEX_HOME, PATH = the binary dir + /usr/bin + /bin (no
# claude, codex, gh, herdr, osascript). The codex config excludes the two
# implicit writable roots (/tmp, $TMPDIR), since the scratchpad lives under
# /private/tmp and would otherwise cover the agmsg store in every case.
set -u
S=$SCRATCH/ods3
P=${PROBE_RUN:-$S/doctor/run}
B=${PROBE_BIN:-$S/bin}
echo "binary: $B/ralph"
rm -rf "$P"
mkdir -p "$P/home" "$P/tmp" "$P/agmsg/scripts" "$P/codex-plain" "$P/codex-ww"
printf '#!/bin/sh\n' > "$P/agmsg/scripts/send.sh"
chmod +x "$P/agmsg/scripts/send.sh"
printf '[sandbox_workspace_write]\nexclude_slash_tmp = true\nexclude_tmpdir_env_var = true\n' > "$P/codex-plain/config.toml"
printf 'sandbox_mode = "workspace-write"\n[sandbox_workspace_write]\nexclude_slash_tmp = true\nexclude_tmpdir_env_var = true\n' > "$P/codex-ww/config.toml"

run_case() { # name codex_dir verified default roles_toml
  local name=$1 codex=$2 verified=$3 def=$4 roles=$5 d="$P/$1"
  mkdir -p "$d"
  git -C "$d" init -q
  cat > "$d/ralph.toml" <<EOF
[org]
agmsg_home = "$P/agmsg"
driver_pool = ["claude", "codex"]
model_pool = [
  { driver = "codex", model = "gpt-6-astra" },
  { driver = "claude", model = "opus" },
]

[org.permissions]
codex_verified = $verified
default = "$def"

[org.permissions.roles]
$roles
EOF
  echo "### $name (codex config: ${codex##*/}, codex_verified = $verified, default = $def, roles: $roles)"
  (cd "$d" && env -i HOME="$P/home" CODEX_HOME="$P/$codex" TMPDIR="$P/tmp" PATH="$B:/usr/bin:/bin" ralph doctor > "$d/doctor.out" 2>&1; echo "doctor rc=$?")
  grep -n 'agmsg writable root' "$d/doctor.out" | sed "s#$P#<run>#g"
  grep -n '✗' "$d/doctor.out" | sed "s#$P#<run>#g"
  echo
}

run_case D1-director-edits codex-plain true guarded 'director = "edits"'
run_case D2-director-autonomous codex-plain true guarded 'director = "autonomous"'
run_case D3-control-implementer-edits codex-plain true guarded 'implementer = "edits"'
run_case D4-guarded-director-autonomous-default codex-ww false autonomous 'director = "guarded"'
run_case D5-control-no-roles-guarded codex-plain true guarded ''
run_case D6-control-autonomous-default-no-roles codex-ww false autonomous ''
ls -A "$P/home"
```

The pre-fix run is the same script with `PROBE_RUN=$SCRATCH/ods3/doctor/run-pre PROBE_BIN=$SCRATCH/ods3/pre/bin`.

#### C3-3.2 Output, HEAD binary

```text
binary: $SCRATCH/ods3/bin/ralph
### D1-director-edits (codex config: codex-plain, codex_verified = true, default = guarded, roles: director = "edits")
doctor rc=1
21:  ✓ Codex sandbox (agmsg writable root): pass — not needed: no role that can use a codex model resolves to edits or autonomous and <run>/codex-plain/config.toml does not set sandbox_mode = "workspace-write"
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D2-director-autonomous (codex config: codex-plain, codex_verified = true, default = guarded, roles: director = "autonomous")
doctor rc=1
21:  ✓ Codex sandbox (agmsg writable root): pass — not needed: no role that can use a codex model resolves to edits or autonomous and <run>/codex-plain/config.toml does not set sandbox_mode = "workspace-write"
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D3-control-implementer-edits (codex config: codex-plain, codex_verified = true, default = guarded, roles: implementer = "edits")
doctor rc=1
21:  ⚠ Codex sandbox (agmsg writable root): warn — no writable root in <run>/codex-plain/config.toml covers the agmsg store <run>/agmsg/db, so a codex seat under workspace-write cannot send RESULT to leader ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true and a role that can use a codex model resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats); add <run>/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D4-guarded-director-autonomous-default (codex config: codex-ww, codex_verified = false, default = autonomous, roles: director = "guarded")
doctor rc=1
21:  ⚠ Codex sandbox (agmsg writable root): warn — no writable root in <run>/codex-ww/config.toml covers the agmsg store <run>/agmsg/db, so a codex seat under workspace-write cannot send RESULT to leader ("attempt to write a readonly database"); needed because sandbox_mode = "workspace-write" in <run>/codex-ww/config.toml and a role that can use a codex model resolves to guarded (guarded codex seats inherit it); add <run>/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D5-control-no-roles-guarded (codex config: codex-plain, codex_verified = true, default = guarded, roles: )
doctor rc=1
21:  ✓ Codex sandbox (agmsg writable root): pass — not needed: no role that can use a codex model resolves to edits or autonomous and <run>/codex-plain/config.toml does not set sandbox_mode = "workspace-write"
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D6-control-autonomous-default-no-roles (codex config: codex-ww, codex_verified = false, default = autonomous, roles: )
doctor rc=1
21:  ✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and no role that can use a codex model resolves to guarded
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

```

`doctor rc=1` comes from the one failing check, `Claude Code CLI` (no claude on PATH); `ls -A` of the scratch HOME printed nothing.

#### C3-3.3 Output, pre-fix binary (0bfbb054)

D1 and D2 differ from HEAD; D3 to D6 are byte-identical to C3-3.2:

```text
binary: $SCRATCH/ods3/pre/bin/ralph
### D1-director-edits (codex config: codex-plain, codex_verified = true, default = guarded, roles: director = "edits")
doctor rc=1
21:  ⚠ Codex sandbox (agmsg writable root): warn — no writable root in <run>/codex-plain/config.toml covers the agmsg store <run>/agmsg/db, so a codex seat under workspace-write cannot send RESULT to leader ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true and a role that can use a codex model resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats); add <run>/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH

### D2-director-autonomous (codex config: codex-plain, codex_verified = true, default = guarded, roles: director = "autonomous")
doctor rc=1
21:  ⚠ Codex sandbox (agmsg writable root): warn — no writable root in <run>/codex-plain/config.toml covers the agmsg store <run>/agmsg/db, so a codex seat under workspace-write cannot send RESULT to leader ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true and a role that can use a codex model resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats); add <run>/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)
3:  ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH
```

### C3-4 Mutation (go1.26.0, `go test -overlay`)

#### C3-4.1 Harness

`$SCRATCH/ods3/mut/mutate.py` is cycle 2's harness (C2-4.1) with its output under `$SCRATCH/ods3/mut/out/`, the cycle 3 canary, a 20 m `-timeout`, multi-package mutants, and the mutant module picked by `MUTANTS_MODULE`. `-run` regexes are cycle 2's plus the changed code's tests (org: `Director|StopAll|Spawn_DirectorRole|RenderRolePrompt|CodexWritableRoot|PermissionArgs|ResolvePermissionMode`; cli: `OrgDirector|StatusCmd_Director|OrgStopAll|OrgSpawn_RoleDirector|OrgLegacyLedger|CodexAgmsgWritableRoot|CodexSeatModesPossible|ARelativeAgmsgHome`). `mutants.py` holds the edits below; A2, D7, and D8 move whole blocks that the module cuts out of the HEAD files and asserts once each. Every mutant compiled (`go build -overlay`) before the runs; no BADEDIT, no BUILD failure. X0, X0p, X0c, and X0d pass an unchanged file through the overlay as controls. Phase 1 ran with `-run` and 2 jobs; the survivors then ran against whole packages (`--full`, 3 jobs).

#### C3-4.2 Results (filtered runs)

30 mutants: 24 red, 6 survived (B2, C3, D3, D3b, D8, E2); the 4 controls pass. "failing top-level tests" drops the `Test` prefix.

| ID | pkg | file | mutation | result (filtered) | failing top-level tests |
| --- | --- | --- | --- | --- | --- |
| X0 | org | director_start.go | org: director_start.go unchanged through -overlay | SURVIVED 7s |  |
| X0p | org+cli | permissions.go | org: permissions.go unchanged through -overlay | SURVIVED 63s |  |
| X0c | cli | org_director_headless.go | cli: org_director_headless.go unchanged through -overlay | SURVIVED 55s |  |
| X0d | cli | doctor_codex_writable_root.go | cli: doctor_codex_writable_root.go unchanged through -overlay | SURVIVED 60s |  |
| A1 | org+cli | director_start.go | D12 removed | red 68s | OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| A2 | org+cli | director_start.go | D12 moved after permissionArgsForDriver | red 73s | OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| A3 | org+cli | director_start.go | D12 refuses autonomous only (edits passes) | red 70s | OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| A4 | org+cli | director_start.go | D12 for any driver (claude refused too) | red 57s | 41 top-level tests, among them Director_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration, Director_TwoLedgersOnOneHerdrServerNeverCloseEachOther, OrgDirectorStart_AFailedAgentStartIsCleanedUp (all in $SCRATCH/ods3/mut/out/A4/out.txt) |
| A5 | org+cli | director_start.go | D12 does not wrap ErrDirectorCodexWorkspaceWrite (%w -> %v) | red 71s | OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| A6 | org+cli | director_start.go | D12 only while codex_verified is false (verified unlocks again) | red 59s | OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| B1 | org+cli | permissions.go | director branch restored in codexWritableRootArgs | red 57s | CodexWritableRootArgs |
| B2 | org+cli | director_start.go | StartDirector calls codexWritableRootArgs again (branch not restored) | SURVIVED 57s |  |
| B3 | org+cli | director_start.go, permissions.go | B1 + B2: cycle 2's director --add-dir wiring back, D12 kept | red 55s | CodexWritableRootArgs |
| C1 | org+cli | permissions.go | CodexModeRunsUnderWorkspaceWrite true for guarded | red 49s | 13 top-level tests, among them CheckCodexAgmsgWritableRoot_BothReasons_WarnDetailNumbersEachReason, CheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted, CheckCodexAgmsgWritableRoot_EmptyStringRoleKeyDoesNotSuppressWarn (all in $SCRATCH/ods3/mut/out/C1/out.txt) |
| C2 | org+cli | permissions.go | CodexModeRunsUnderWorkspaceWrite false for edits | red 50s | CheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted, CodexSeatModesPossible, CodexWritableRootArgs, OrgDirectorStart_RefusedWithNothingWritten, StartDirector_PermissionModeOfRoleDirector, StartDirector_RefusesBeforeWritingOrCallingHerdr |
| C3 | org+cli | permissions.go | CodexModeRunsUnderWorkspaceWrite = mode != guarded (differs only for an unknown/empty mode) | SURVIVED 44s |  |
| D1 | cli | org_director_headless.go | D13 removed | red 43s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused, OrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns |
| D2 | cli | org_director_headless.go | D13 only when the director's cwd differs from start's (unresolved paths) | red 51s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused, OrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns |
| D2b | cli | org_director_headless.go | D2 comparing symlink-resolved paths | red 54s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused, OrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns |
| D3 | cli | org_director_headless.go | D13 without the Changed("config") condition | SURVIVED 61s |  |
| D3b | cli | org_director_headless.go | D13 without configPath != "" (matters for --config "" only) | SURVIVED 56s |  |
| D4 | cli | org_director_headless.go | RALPH_ORG_AGMSG_HOME warning removed | red 64s | OrgDirectorStart_WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector |
| D5 | cli | org_director_headless.go | the warning goes to stdout | red 60s | OrgDirectorStart_WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector |
| D6 | cli | org_director_headless.go | the warning only with --config | red 60s | OrgDirectorStart_WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector |
| D7 | cli | org_director_headless.go | checkStartAgmsgHome after resolveModelOrWarn (the --model fallback warns first) | red 65s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused |
| D8 | cli | org_director_headless.go | checkStartAgmsgHome before the enabled check | SURVIVED 54s |  |
| D9 | cli | org_director_headless.go | D13 uses !filepath.IsAbs instead of the shared predicate (~/ refused) | red 49s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused, OrgDirectorStart_LaunchesRegistersAndStops, OrgDirectorStart_ReadsTheMainWorktreeRalphToml, OrgDirectorStart_RefusedWithNothingWritten, OrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced |
| E1 | cli | org_watch.go | agmsgHomeIsRelative: a bare ~ counts as not relative | red 40s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused |
| E2 | cli | org_watch.go | agmsgHomeIsRelative: empty counts as relative | SURVIVED 40s |  |
| E3 | cli | org_watch.go | agmsgHomeIsRelative: ~/ counts as relative | red 39s | OrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused, OrgDirectorStart_LaunchesRegistersAndStops, OrgDirectorStart_ReadsTheMainWorktreeRalphToml, OrgDirectorStart_RefusedWithNothingWritten, OrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced, OrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild |
| F1 | cli | doctor_codex_writable_root.go | doctor counts role director's edits/autonomous again | red 42s | CheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted, CodexSeatModesPossible |
| F2 | cli | doctor_codex_writable_root.go | doctor skips role director entirely (a guarded director no longer counts) | red 43s | CheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted, CodexSeatModesPossible |
| F3 | cli | doctor_codex_writable_root.go | doctor applies the defaults as role director | red 43s | 30 top-level tests, among them CheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded, CheckCodexAgmsgWritableRoot_BlockedAncestorAndExplicitStoreBothListed_PassNamingExplicit, CheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn (all in $SCRATCH/ods3/mut/out/F3/out.txt) |
| H1 | cli | org_director_headless.go | help drops 'wherever it runs' | red 48s | OrgDirectorStart_HelpStatesTheCodexAndAgmsgHomeRules |

Subtests behind the kills the handoff named: A1 and A2 fail `PermissionModeOfRoleDirector`'s `codex autonomous before codex_verified` and `codex edits before codex_verified` (A2 gives the codex_verified error instead of `ErrDirectorCodexWorkspaceWrite`); A6 fails the three `with codex_verified` rows; B1 fails `CodexWritableRootArgs/director_codex_autonomous,_state_dir_outside_cwd` and `director_codex_edits,_state_dir_outside_cwd`; D2 and D2b fail `ARelativeAgmsgHomeInTheConfigIsRefused`'s three refused rows and `ARelativeAgmsgHomeIsRefusedWhereverStartRuns/in_the_main_worktree_root` and `on_a_ledger_with_no_main_worktree`, not `in_a_linked_worktree`; F1 fails `a_director_in_edits` and `a_director_in_autonomous` of both doctor tests; C1 fails, among others, the three codex guarded rows of `PermissionModeOfRoleDirector` and `CodexWritableRootArgs/leader_codex_guarded,_state_dir_outside_cwd`.

#### C3-4.3 The 6 survivors against whole packages (`-run` dropped, `--full`)

```text
B2	SURVIVED	315s
C3	SURVIVED	313s
D3	SURVIVED	313s
D3b	SURVIVED	264s
D8	SURVIVED	265s
E2	SURVIVED	265s
```

Why each survives:

- B2: at HEAD `codexWritableRootArgs` returns nil for role director, so calling it again adds nothing to the agent args (equivalent by construction; B3, which also restores the branch, is red)
- C3: `config.Load` accepts only autonomous, edits, and guarded for `[org.permissions].default` and every `[org.permissions.roles]` value (`internal/config/config.go` 405-412), and `ResolvePermissionMode` returns one of them, so `mode != guarded` equals "autonomous or edits" for every loaded config (equivalent for loaded configs)
- D3: `--config` defaults to "" (`internal/cli/org.go` 52), so `Changed("config")` false implies `configPath == ""` (equivalent, like cycle 3 of org-watch-gate's A8a)
- E2: neither caller passes "": `config.Load` backfills an empty `agmsg_home` with the default (`config.go` 392-394), and `orgWatchChildAgmsgHome` returns before the predicate when the variable is empty and no `--config` was given (equivalent)
- D3b: differs only for `--config ""` (the flag given, empty). No test passes it. Probe PD3b below
- D8: no test gives a `--config` with both `[org.director].enabled = false` and a relative agmsg_home, so the order of the two refusals is unpinned. The plan does not rank them. Probe PD8 below

#### C3-4.4 Probe tests for D3b and D8 (overlay only, not committed)

`$SCRATCH/ods3/probe/zz_probe_c3_test.go`, mapped to `internal/cli/zz_probe_c3_test.go` and merged with each mutant's overlay:

```go
package cli

// Scratch-only probe tests for /test cycle 3 of org-director-start, read
// through go test -overlay; never committed. Each targets one surviving
// mutant of the cycle-3 change (mutants.py).

import (
	"path/filepath"
	"strings"
	"testing"
)

// D3b: `--config ""` (the flag given, empty) reads ./ralph.toml like no
// --config; a relative [org].agmsg_home there is not refused, since the
// director and the watch both resolve it as without --config.
func TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome(t *testing.T) {
	stateDir, herdrLog, cwd := directorStartLedger(t)
	writeTestFile(t, filepath.Join(cwd, "ralph.toml"), "[org]\nagmsg_home = \"agmsg/home\"\n")

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", "")
	if err != nil {
		t.Fatalf("director start --config '': %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if got := directorEventKinds(directorEventsIn(t, stateDir)); len(got) != 4 || got[3] != "launched d1" {
		t.Errorf("director.jsonl = %v; want d1 launched", got)
	}
	if lines := readLogLines(t, herdrLog); len(lines) == 0 {
		t.Errorf("no herdr call; want the director started")
	}
}

// D8: with [org.director].enabled = false and a relative [org].agmsg_home
// in the --config, the enabled refusal is the one start gives (records
// HEAD's order; the plan does not rank the two refusals).
func TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	cfg := writeDirectorConfig(t, "[org]\nagmsg_home = \"agmsg/home\"\n\n[org.director]\nenabled = false\n")

	_, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", cfg)
	if want := "org: director start: [org.director].enabled is false in the --config file "; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("director start = %v; want the enabled refusal (%q...)", err, want)
	}
}
```

Runs (`go test -overlay=<ov> -count=1 -v -run '^TestProbe_C3_' ./internal/cli/` on HEAD, D3b, D8, and the controls D3 and D1):

```text
=== head
--- PASS: TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome (1.67s)
--- PASS: TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	3.852s
=== D3b
    zz_probe_c3_test.go:22: director start --config '': org: director start: [org].agmsg_home "agmsg/home" in $TMPDIR/TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome2594459915/004/ralph.toml is a relative path: the director and the watch may resolve it from different directories and so use different agmsg installs (the director runs in the ledger's main worktree root when the ledger has one, while the watch resolves it from this directory), so start refuses it wherever it runs; give an absolute path (or one that starts with ~/) (stdout:  stderr: )
--- FAIL: TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome (0.02s)
--- PASS: TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/cli	1.831s
FAIL
=== D8
--- PASS: TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome (1.28s)
    zz_probe_c3_test.go:41: director start = org: director start: [org].agmsg_home "agmsg/home" in the --config file $TMPDIR/TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome4073561308/006/ralph.toml is a relative path: the director and the watch may resolve it from different directories and so use different agmsg installs (the director runs in the ledger's main worktree root when the ledger has one, while the watch resolves it from this directory), so start refuses it wherever it runs; give an absolute path (or one that starts with ~/); want the enabled refusal ("org: director start: [org.director].enabled is false in the --config file "...)
--- FAIL: TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome (0.00s)
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/cli	3.331s
FAIL
=== D3
--- PASS: TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome (1.61s)
--- PASS: TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	3.415s
=== D1
--- PASS: TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome (0.59s)
--- PASS: TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	3.088s
```

Each probe passes on HEAD, fails under its target mutant, and passes under the other target and under the controls D3 (the equivalent survivor) and D1 (D13 removed: neither probe's start reaches a D13 refusal at HEAD).

#### C3-4.5 Cycle 2's mutant list at HEAD

Cycle 2's `mutants.py` (C2-4.2), copied as `mutants_c2.py` and rerun at HEAD with ids prefixed `R2-` (`MUTANTS_MODULE=mutants_r2`, `-run` as in C3-4.1, 3 jobs, 186 s). Every `old` string still occurs exactly once in the HEAD files. The verdicts equal cycle 2's: the 3 controls pass, 27 are red (incl. the cross-package reruns K1c, K3c, K2cc and L4, which caf1d025's rewrite of `LaunchesRegistersAndStops`'s codex row to guarded still kills), and K15, K15c, K16 survive:

```text
R2-X0	SURVIVED	3s
R2-X0t	SURVIVED	3s
R2-X0c	SURVIVED	49s
R2-K1	red	6s	TestStartDirector_CarriesTheConfigIntoThePrompt
R2-L1	red	49s	TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
R2-K2	red	5s	TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K2b	red	5s	TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K2c	red	8s	TestStartDirector_CarriesTheConfigIntoThePrompt
R2-L3	red	53s	TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
R2-K3	red	5s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K4	red	5s	TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders
R2-K5	red	5s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K6	red	7s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K7	red	6s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
R2-K8	red	6s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
R2-K9	red	7s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K10	red	5s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
R2-K11	red	5s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
R2-K12	red	6s	TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K13	red	7s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig,TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure,TestStartDirector_CarriesTheConfigIntoThePrompt
R2-K14	red	6s	TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
R2-D4	red	6s	TestStartDirector_RefusesBeforeWritingOrCallingHerdr
R2-E1	red	7s	TestStartDirector_AFailedWriteOfAttachedCallsNoHerdr
R2-E12	red	5s	TestStartDirector_AFailedCreatesOutputIsNotAnID
R2-E13	red	5s	TestStartDirector_AFailedCreatesOutputIsNotAnID
R2-P2	red	45s	TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus
R2-K1c	red	45s	TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
R2-K3c	red	45s	TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
R2-K2cc	red	45s	TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml
R2-K15	SURVIVED	5s
R2-K15c	SURVIVED	45s
R2-L4	red	44s	TestOrgDirectorStart_LaunchesRegistersAndStops
R2-K16	SURVIVED	5s
```

Calls to the canary during all mutation and probe runs: 0 (the 2 lines in `calls.log` are C3-2.2's typed-by-hand check).

## Cycle 4

- Date: 2026-10-16 (JST 04:28-05:30; the UTC timestamps in the output read 2026-10-15T19:xx and 20:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD 905fc0a7 at the start (cycle 3 ran at 3504ee30, its report is b0adfc63), then 10efa82c, the tests this cycle added (C4-5). Go changed since cycle 3 in 3 commits: b888d637 (D14: `MarkStoppedIfUnchanged` and `ErrDirectorChangedDuringStop`, stop's record compares the workspace and pane ids under the lock; D15: the `/director` skill's description and `internal/config/skill_frontmatter_test.go`), 5097b67e (doc lists, the refusal's notes once, on the CLI's stderr; the frontmatter test renamed with its limits in the doc and the failure text), and 44b80391 (`DirectorStopResult.AlreadyStopped`: a stop whose record finds the registration already stopped succeeds; `directorRecordStopError`, so "org: director: " appears once; writer rows for `MarkStoppedIfUnchanged` and `MarkForceStopped`)
- Environment: macOS arm64, go1.26.0 (default) and go1.25.8 (`GOTOOLCHAIN=go1.25.8`), git 2.49.0, Ruby 3.4.10 (Psych) for the YAML check of C4-3. No real herdr, codex, claude, or osascript ran. Load average 4-14 during the runs (other sessions)
- Redaction: as in cycles 1-3 (`$SCRATCH`, `~`, `$TMPDIR/`). Cycle 4's scratch files are under `$SCRATCH/ods4`. The wrapper's full logs `docs/evidence/verify-2026-10-15-192839.log` and `docs/evidence/verify-2026-10-15-195306.log` and the raw outputs appended to `docs/evidence/test-2026-10-15-org-director-start.log` (from the line `===== /test cycle 4`) are gitignored and not committed
- Trailing blanks at the ends of output lines were stripped; nothing else in the output was changed apart from the redaction, and in C4-3.2 the fixed tail of the failure message, which the probe cut at 260 characters, is written out from the format string at `skill_frontmatter_test.go:197`
- The only worktree edit for the tests is commit 10efa82c (two tests, C4-5). Mutants and probe tests lived in `$SCRATCH/ods4` and reached the build only through `go test -overlay`

### C4-1 ./scripts/run-test.sh (RALPH_VERIFY_SCOPE unset = changed; templates/base/ralph.toml widened it to full)

Run 1 at 905fc0a7 (`PATH=$SCRATCH/ods4/canary/bin:$PATH ./scripts/run-test.sh`, rc 0, 665 s wall, 4,654 lines) is not counted as HEAD's: its Go part started about 04:36 and the first test of C4-5 was written into `internal/org/director_stop_test.go` at 04:38:24, while it ran. Its org binary was most likely built before that (org ran 81.966 s from the start of the Go part); that is an estimate, not checked. Run 2 at 10efa82c is the gate: the same command, rc 0, 511 s wall, 4,654 lines. Filtered with the command of section 1 (168 lines each). Against C3-1's 168 lines, with the line numbers, the `N.NNs` timings, and the timestamp dropped, `diff` gives for run 2:

```text
159,160c159,160
< ok  	github.com/yoshpy-dev/ralph/internal/cli	N.NNs
< ok  	github.com/yoshpy-dev/ralph/internal/config	(cached)
---
> ok  	github.com/yoshpy-dev/ralph/internal/cli	(cached)
> ok  	github.com/yoshpy-dev/ralph/internal/config	N.NNs
163c163
< ok  	github.com/yoshpy-dev/ralph/internal/org/driver	N.NNs
---
> ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
168c168
< Evidence saved to: docs/evidence/verify-2026-10-15-154510.log
---
> Evidence saved to: docs/evidence/verify-2026-10-15-195306.log
```

and run 1 against run 2:

```text
159c159
< ok  	github.com/yoshpy-dev/ralph/internal/cli	N.NNs
---
> ok  	github.com/yoshpy-dev/ralph/internal/cli	(cached)
163c163
< ok  	github.com/yoshpy-dev/ralph/internal/org/driver	N.NNs
---
> ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
168c168
< Evidence saved to: docs/evidence/verify-2026-10-15-192839.log
---
> Evidence saved to: docs/evidence/verify-2026-10-15-195306.log
```

41 suites, 41 "OK", every "FAIL:" line is 0 (25 of them, no other FAIL line). 4,283 lines start with PASS after leading blanks (cycle 3: 4,283). The one real SKIP is tests/test-secret-scan-branch.sh's (git 2.49.0). The Go part of run 2:

```text
4637:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
4641:?   	github.com/yoshpy-dev/ralph	[no test files]
4642:?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
4643:ok  	github.com/yoshpy-dev/ralph/internal/cli	(cached)
4644:ok  	github.com/yoshpy-dev/ralph/internal/config	1.027s
4645:ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
4646:ok  	github.com/yoshpy-dev/ralph/internal/org	81.134s
4647:ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
4648:ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
4649:ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
4650:ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
4652:==> All verifiers passed.
4654:Evidence saved to: docs/evidence/verify-2026-10-15-195306.log
```

`internal/cli (cached)` in run 2 is run 1's go1.26.0 result (212.147 s) for the same inputs: between the two runs only `_test.go` files of `internal/org` changed, which are not in cli's build. The other `(cached)` packages are cached the same way; `internal/insights` imports `internal/org`, whose non-test code did not change. Every package ran with `-count=1` on both toolchains in C4-2.1, cli and org with the race detector too.

### C4-2 Go tests

Every Go run below had `$SCRATCH/ods4/canary/bin` first on PATH (C4-2.4).

#### C4-2.1 Per package, `-count=1 -v`

Counts are from the `--- PASS/FAIL/SKIP` lines (top-level tests, then subtests).

go1.25.8 at 905fc0a7 (`GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -v -coverprofile=cover125.out`, rc 0, 234 s wall; it ran 04:29:58-04:33:52, before any edit):

```text
internal/cli	ok	230.898s	top 708 (pass 708 fail 0 skip 0)	sub 662 (pass 662 fail 0 skip 0)
internal/config	ok	1.931s	top 39 (pass 39 fail 0 skip 0)	sub 53 (pass 53 fail 0 skip 0)
internal/insights	ok	1.485s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	96.713s	top 810 (pass 809 fail 0 skip 1)	sub 1710 (pass 1710 fail 0 skip 0)
internal/org/driver	ok	2.523s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	1.677s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	2.583s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	3.525s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.25.8 at 10efa82c (the same command into `cover125b.out`, rc 0, 218 s wall):

```text
internal/cli	ok	217.091s	top 708 (pass 708 fail 0 skip 0)	sub 662 (pass 662 fail 0 skip 0)
internal/config	ok	0.538s	top 39 (pass 39 fail 0 skip 0)	sub 53 (pass 53 fail 0 skip 0)
internal/insights	ok	0.956s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org	ok	81.949s	top 812 (pass 811 fail 0 skip 1)	sub 1712 (pass 1712 fail 0 skip 0)
internal/org/driver	ok	1.780s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	1.824s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	2.730s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	2.756s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

go1.26.0 at 10efa82c, org and cli with the race detector (`go test -race -count=1 -v ./internal/org/ ./internal/cli/`, rc 0, 232 s wall, 0 `DATA RACE` lines):

```text
internal/org	ok	86.892s	top 812 (pass 811 fail 0 skip 1)	sub 1712 (pass 1712 fail 0 skip 0)
internal/cli	ok	224.675s	top 708 (pass 708 fail 0 skip 0)	sub 662 (pass 662 fail 0 skip 0)
```

go1.25.8 at 10efa82c, org and cli with the race detector (`GOTOOLCHAIN=go1.25.8 go test -race -count=1 -v ./internal/org/ ./internal/cli/`, rc 0, 225 s wall, 0 `DATA RACE` lines):

```text
internal/org	ok	85.168s	top 812 (pass 811 fail 0 skip 1)	sub 1712 (pass 1712 fail 0 skip 0)
internal/cli	ok	218.353s	top 708 (pass 708 fail 0 skip 0)	sub 662 (pass 662 fail 0 skip 0)
```

go1.26.0 at 10efa82c, the other six packages (`go test -count=1 -v ./internal/config/ ./internal/insights/ ./internal/org/driver/ ./internal/org/protocol/ ./internal/scaffold/ ./internal/upgrade/`, rc 0):

```text
internal/config	ok	1.610s	top 39 (pass 39 fail 0 skip 0)	sub 53 (pass 53 fail 0 skip 0)
internal/insights	ok	0.653s	top 36 (pass 36 fail 0 skip 0)	sub 7 (pass 7 fail 0 skip 0)
internal/org/driver	ok	1.915s	top 48 (pass 48 fail 0 skip 0)	sub 89 (pass 89 fail 0 skip 0)
internal/org/protocol	ok	2.410s	top 15 (pass 15 fail 0 skip 0)	sub 56 (pass 56 fail 0 skip 0)
internal/scaffold	ok	1.983s	top 32 (pass 30 fail 0 skip 2)	sub 0 (pass 0 fail 0 skip 0)
internal/upgrade	ok	3.344s	top 114 (pass 114 fail 0 skip 0)	sub 13 (pass 13 fail 0 skip 0)
```

At 10efa82c the sorted `<package> <name> <result>` lines of the go1.25.8 run (4,396) and of the two go1.26.0 runs together (4,396) are identical (`LC_ALL=C sort` on both sides), and so are the org and cli lines of the two race runs (3,894 each). The three SKIPs are cycle 3's (`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`, scaffold's `TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS`).

Earlier flaky tests (go1.25.8 at 905fc0a7 and 10efa82c, then the go1.26.0 and go1.25.8 race runs):

```text
go125.out:--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.78s)
go125.out:--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.64s)
go125.out:--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (4.99s)
go125.out:--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.45s)
go125b.out:--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.78s)
go125b.out:--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.66s)
go125b.out:--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.17s)
go125b.out:--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.36s)
race.out:--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.34s)
race.out:--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.83s)
race.out:--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.75s)
race.out:--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.63s)
race125.out:--- PASS: TestRunWatcher_TimeoutIndependentOfSmallInterval (1.49s)
race125.out:--- PASS: TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess (0.82s)
race125.out:--- PASS: TestRunDoctorFull_StrictFlipsExitCode_DriftedCore (0.70s)
race125.out:--- PASS: TestOrgGate_InterruptStopsTheScriptAndRecordsNothing (2.17s)
```

The slowest `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` subtest was SIGHUP at 2.49 s (go1.25.8 at 905fc0a7), under the 5 s `gateKillGrace` and the 15 s budget.

#### C4-2.2 Tests added since cycle 3

Against cycle 3's 4,345 lines (C3-2.1), go1.25.8 at 905fc0a7 adds 47 lines and removes none:

```text
515a516,518
> internal/cli TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain PASS
> internal/cli TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain/force_false PASS
> internal/cli TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain/force_true PASS
516a520,522
> internal/cli TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun PASS
> internal/cli TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/a_forced_stop_whose_workspace_close_failed PASS
> internal/cli TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/the_workspace_closed PASS
1435a1442,1462
> internal/config TestPlainFrontmatterFindings PASS
> internal/config TestPlainFrontmatterFindings/CRLF_line_ends PASS
> internal/config TestPlainFrontmatterFindings/a_colon_followed_by_a_tab PASS
> internal/config TestPlainFrontmatterFindings/a_colon_or_a_#_with_no_whitespace PASS
> internal/config TestPlainFrontmatterFindings/a_colon_that_ends_the_value PASS
> internal/config TestPlainFrontmatterFindings/a_column_counted_in_characters PASS
> internal/config TestPlainFrontmatterFindings/a_comment_line PASS
> internal/config TestPlainFrontmatterFindings/a_frontmatter_never_closed PASS
> internal/config TestPlainFrontmatterFindings/a_space_and_a_# PASS
> internal/config TestPlainFrontmatterFindings/a_tab_and_a_# PASS
> internal/config TestPlainFrontmatterFindings/a_value_that_starts_with_a_backquote,_which_it_does_not_check PASS
> internal/config TestPlainFrontmatterFindings/an_empty_value,_and_a_nested_value_it_does_not_check PASS
> internal/config TestPlainFrontmatterFindings/block_scalars_and_their_lines PASS
> internal/config TestPlainFrontmatterFindings/no_frontmatter PASS
> internal/config TestPlainFrontmatterFindings/one_finding_per_line,_every_line PASS
> internal/config TestPlainFrontmatterFindings/quoted_values PASS
> internal/config TestPlainFrontmatterFindings/the_/director_description_after_the_fix PASS
> internal/config TestPlainFrontmatterFindings/the_/director_description_before_the_fix PASS
> internal/config TestPlainFrontmatterFindings/the_body PASS
> internal/config TestPlainFrontmatterFindings/the_first_of_two_shapes_on_a_line PASS
> internal/config TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash PASS
1675a1703
> internal/org TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkForceStopped_of_the_current_registration PASS
1676a1705
> internal/org TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkStoppedIfUnchanged_of_the_current_registration PASS
1694a1724,1729
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead PASS
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/both_at_the_read,_then_launched PASS
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/nothing_at_the_read,_then_the_workspace,_the_pane,_and_launched PASS
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/nothing_at_the_read_and_nothing_after PASS
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/the_pane_at_the_read,_then_the_workspace PASS
> internal/org TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/the_workspace_at_the_read,_then_the_pane_and_launched PASS
3670a3706
> internal/org TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce PASS
3802a3839
> internal/org TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile PASS
3809a3847
> internal/org TestStopDirector_ARecordThatFailsSaysItsPrefixOnce PASS
3838a3877,3885
> internal/org TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns PASS
> internal/org TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/everything,_before_stop_asks_herdr_anything PASS
> internal/org TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/the_pane,_while_stop_closes_the_workspace PASS
> internal/org TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/the_pane,_while_stop_closes_the_workspace,_with_force PASS
> internal/org TestStopDirector_SucceedsWhenTheStopWasRecordedFirst PASS
> internal/org TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/another_stop_recorded_it_first PASS
> internal/org TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it PASS
> internal/org TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it,_with_force PASS
> internal/org TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it_while_a_forced_stop's_workspace_close_failed PASS
```

That is 10 top-level tests (cli 2, config 2, org 6) and 37 subtests (cli 4, config 19, org 14). 10efa82c adds 4 more (C4-5):

```text
3722a3723
> internal/org TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord PASS
3846a3848,3850
> internal/org TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt PASS
> internal/org TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt/start's_cleanup PASS
> internal/org TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt/stop PASS
```

#### C4-2.3 The new tests by name, `-v`

The tests the handoff named and the D14 tests (`grep -n ErrDirectorChangedDuringStop` in the `_test.go` files finds `wantChangedDuringStop` in `internal/org/director_stop_test.go`, `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead` in `internal/org/director_test.go`, and `TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain` in `internal/cli/org_director_headless_test.go`; `wantChangedDuringStop` is the helper of `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns`), with the two existing tests 44b80391 and b888d637 changed (`TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce`, `TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade`), run by name at 905fc0a7 on go1.26.0 (`go test -count=1 -v -run '<names>'` per package, rc 0 for org, cli, and config). None is skipped, and all of them pass again in the go1.25.8 run at 10efa82c (C4-2.2 lists no name that went away):

```text
--- PASS: TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade (0.00s)
    --- PASS: TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade/workspace_created (0.00s)
    --- PASS: TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade/pane_created (0.00s)
--- PASS: TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce (0.00s)
--- PASS: TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce (0.00s)
    --- PASS: TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce/a_label_that_is_another_launch's (0.00s)
    --- PASS: TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce/a_pane_close_that_fails (0.00s)
--- PASS: TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns (0.01s)
    --- PASS: TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/the_pane,_while_stop_closes_the_workspace (0.00s)
    --- PASS: TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/the_pane,_while_stop_closes_the_workspace,_with_force (0.00s)
    --- PASS: TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns/everything,_before_stop_asks_herdr_anything (0.00s)
--- PASS: TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile (0.00s)
--- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst (0.01s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/another_stop_recorded_it_first (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it,_with_force (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it_while_a_forced_stop's_workspace_close_failed (0.00s)
--- PASS: TestStopDirector_ARecordThatFailsSaysItsPrefixOnce (0.00s)
--- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead (0.01s)
    --- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/nothing_at_the_read,_then_the_workspace,_the_pane,_and_launched (0.00s)
    --- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/the_workspace_at_the_read,_then_the_pane_and_launched (0.00s)
    --- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/the_pane_at_the_read,_then_the_workspace (0.00s)
    --- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/both_at_the_read,_then_launched (0.00s)
    --- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead/nothing_at_the_read_and_nothing_after (0.00s)
--- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored (0.01s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/AttachHeadless_into_a_missing_file (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/AttachHeadless_after_an_ended_registration (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/RecordWorkspace (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/RecordPane (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkLaunched (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkStopped_of_the_current_registration (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkStopped_of_a_detached_registration (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkStoppedIfUnchanged_of_the_current_registration (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkForceStopped_of_the_current_registration (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	0.636s
--- PASS: TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain (0.00s)
    --- PASS: TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain/force_false (0.00s)
    --- PASS: TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain/force_true (0.00s)
--- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun (0.00s)
    --- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/the_workspace_closed (0.00s)
    --- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/a_forced_stop_whose_workspace_close_failed (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/cli	0.572s
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- PASS: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
--- PASS: TestPlainFrontmatterFindings (0.00s)
    --- PASS: TestPlainFrontmatterFindings/the_/director_description_before_the_fix (0.00s)
    --- PASS: TestPlainFrontmatterFindings/the_/director_description_after_the_fix (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_colon_followed_by_a_tab (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_colon_that_ends_the_value (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_space_and_a_# (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_tab_and_a_# (0.00s)
    --- PASS: TestPlainFrontmatterFindings/the_first_of_two_shapes_on_a_line (0.00s)
    --- PASS: TestPlainFrontmatterFindings/one_finding_per_line,_every_line (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_column_counted_in_characters (0.00s)
    --- PASS: TestPlainFrontmatterFindings/CRLF_line_ends (0.00s)
    --- PASS: TestPlainFrontmatterFindings/quoted_values (0.00s)
    --- PASS: TestPlainFrontmatterFindings/block_scalars_and_their_lines (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_colon_or_a_#_with_no_whitespace (0.00s)
    --- PASS: TestPlainFrontmatterFindings/an_empty_value,_and_a_nested_value_it_does_not_check (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_value_that_starts_with_a_backquote,_which_it_does_not_check (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_comment_line (0.00s)
    --- PASS: TestPlainFrontmatterFindings/the_body (0.00s)
    --- PASS: TestPlainFrontmatterFindings/no_frontmatter (0.00s)
    --- PASS: TestPlainFrontmatterFindings/a_frontmatter_never_closed (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/config	0.496s
```

The two tests of C4-5, on go1.25.8 before the commit (the same content as 10efa82c):

```text
--- PASS: TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord (0.00s)
--- PASS: TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt (0.00s)
    --- PASS: TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt/stop (0.00s)
    --- PASS: TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt/start's_cleanup (0.00s)
ok  	github.com/yoshpy-dev/ralph/internal/org	0.412s
```

#### C4-2.4 AC13: no real herdr or osascript, nothing in the repo's ledger

- `$SCRATCH/ods4/canary/bin/{herdr,osascript}` log their argv to `$SCRATCH/ods4/canary/calls.log` and exit 97 (no pass-through). They were first on PATH for both run-test.sh runs, the Go runs above, every mutation run, and the probe tests. `calls.log` had 0 lines after all of them. Typed by hand afterwards from the same PATH, both log and exit 97:

```text
$ wc -l < calls.log   # after every run of C4-1, C4-2, C4-3, C4-4
0
$ PATH=$SCRATCH/ods4/canary/bin:$PATH sh -c 'herdr --version; echo "herdr rc=$?"; osascript -e 1; echo "osascript rc=$?"'
herdr rc=97
osascript rc=97
$ cat calls.log
1792095364 herdr --version
1792095365 osascript -e 1
$ ls -d ~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org .../.claude/worktrees/org-director-start/.harness/state/org
"~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org": No such file or directory (os error 2)
"~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start/.harness/state/org": No such file or directory (os error 2)
$ find ~/MyDev/github.com/yoshpy-dev/ralph -name director.jsonl | wc -l
0
$ stat -f '%m %Sm %z' ~/.codex/config.toml; shasum -a 256 ~/.codex/config.toml
1792054288 Oct 15 17:51:28 2026 33673
ac1a4821574050d3630118510c7dd591bf1839b947f2bb35fe4106e41b61796f  ~/.codex/config.toml
```

#### C4-2.5 Coverage (go1.25.8 at 10efa82c, each package's own tests)

```text
ok  	github.com/yoshpy-dev/ralph/internal/cli	217.091s	coverage: 87.8% of statements
ok  	github.com/yoshpy-dev/ralph/internal/config	0.538s	coverage: 93.3% of statements
ok  	github.com/yoshpy-dev/ralph/internal/insights	0.956s	coverage: 86.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org	81.949s	coverage: 94.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	1.780s	coverage: 93.1% of statements
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	1.824s	coverage: 100.0% of statements
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	2.730s	coverage: 75.7% of statements
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.756s	coverage: 91.2% of statements
```

Statements of the changed files, from `cover125b.out` (each block counted once):

```text
internal/org/director_start.go: 226/241 (93.8%); uncovered blocks (lines): 60-62, 130-132, 138-140, 205-207, 250-252, 346-348, 545-546, 590-593, 607-609, 610-612, 614-616, 676-678, 705-706, 734-735, 736-737
internal/org/director_stop.go: 146/149 (98.0%); uncovered blocks (lines): 263-265, 334-334, 456-458, 552-552
internal/org/director.go: 193/194 (99.5%); uncovered blocks (lines): 481-483
internal/cli/org_director_headless.go: 105/108 (97.2%); uncovered blocks (lines): 195-197, 210-212, 220-222
```

At 905fc0a7 (`cover125.out`) `director_stop.go` was 145/149 (97.3%) with blocks 319-320 uncovered as well: the not-restored case of `DirectorCleanupError.Error`, which the first test of C4-5 reaches. In the changed functions nothing else is uncovered: `markStopped`'s D14 check (`director_start.go` 354-357), `directorRecordedIDs` (380-389), `recordDirectorStop` (`director_stop.go` 198-230), `directorRecordStopError` (247-253), `DirectorCleanupError.Error`'s changed case (335-336), and `printOrgDirectorStop`'s AlreadyStopped branch (`org_director_headless.go` 400-402) are covered. `director_stop.go` 334 is the empty `case e.StopCommand == "":` of the new switch: `newDirectorCleanupError` always sets StopCommand, so only an error built by hand reaches it, as before the switch. `director_start.go` 346-348 is the read error under the lock in `markStopped` and 734-737 abort's detach failures (both S1b); the rest are cycle 3's.

### C4-3 AC16 with a YAML parser, and the frontmatter test against broken copies

#### C4-3.1 Ruby's `YAML.safe_load` on every tracked SKILL.md

`$SCRATCH/ods4/yamlcheck.rb` cuts each file's frontmatter (the lines between the first `---` and the next) and loads it with `YAML.safe_load`, failing on an error or a value that is not a mapping. `ruby yamlcheck.rb $(git ls-files '*SKILL.md')` (54 files), at 905fc0a7 and again at 10efa82c with the same output, rc 0 (10efa82c changes no SKILL.md and nothing under `internal/config`):

```text
.agents/skills/director/SKILL.md: ok name=director description=672 chars
.claude/skills/director/SKILL.md: ok name=director description=672 chars
templates/base/.agents/skills/director/SKILL.md: ok name=director description=672 chars
templates/base/.claude/skills/director/SKILL.md: ok name=director description=672 chars
ok=54 bad=0
```

The four `/director` copies before the fix (`git show b888d637^:<path>`):

```text
.claude/skills/director/SKILL.md: YAML ERROR Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 2 column 155
.agents/skills/director/SKILL.md: YAML ERROR Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 2 column 155
templates/base/.claude/skills/director/SKILL.md: YAML ERROR Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 2 column 155
templates/base/.agents/skills/director/SKILL.md: YAML ERROR Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 2 column 155
ok=0 bad=4
```

The column Psych names (line 2 of the frontmatter, column 155) is the finding of `TestPlainFrontmatterFindings`' first row (line 3 of the file, column 155).

#### C4-3.2 The frontmatter test against broken copies of the skills

`go test -overlay` does not reach the SKILL.md files the test reads at run time, so `$SCRATCH/ods4/fm/probe.sh` runs the test in `git archive HEAD` (905fc0a7; 10efa82c changes neither the skills nor this test) unpacked under `$SCRATCH/ods4/fm/base`, breaks one file at a time, and restores it. `repoRoot` is the test file's directory two levels up, so the copy's own skills are read.

```sh
#!/bin/bash
# Frontmatter-test sensitivity probes on a git-archive copy of HEAD (scratch only).
B=$SCRATCH/ods4/fm/base
PRE=$SCRATCH/ods4/prefix
run() { (cd $B && go test -count=1 -v -run 'TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash' ./internal/config/ 2>&1) | grep -E -- '--- (PASS|FAIL|SKIP)|skill_frontmatter_test.go:[0-9]+:|^(ok|FAIL)\s' | sed "s|$B/||g" | cut -c1-260; }
save() { cp "$B/$1" "$S_TMP/$(echo $1 | tr / _)"; }
S_TMP=$(mktemp -d "$SCRATCH/ods4/fm/tmp.XXXX")
echo "### F0 control: the copy as archived"; run
f=templates/base/.agents/skills/director/SKILL.md
echo "### F1 pre-fix description in $f only"; cp "$B/$f" "$S_TMP/f1"; cp "$PRE/$f" "$B/$f"; run; cp "$S_TMP/f1" "$B/$f"
f=.claude/skills/director/SKILL.md
echo "### F2 pre-fix description in $f only"; cp "$B/$f" "$S_TMP/f2"; cp "$PRE/$f" "$B/$f"; run; cp "$S_TMP/f2" "$B/$f"
f=.agents/skills/test/SKILL.md
echo "### F3 ' see #2' appended to the plain description of $f"; cp "$B/$f" "$S_TMP/f3"; perl -0pi -e 's/^(description: [^\n]*)/$1 see #2/m' "$B/$f"; grep -c ' see #2' "$B/$f"; run; cp "$S_TMP/f3" "$B/$f"
f=.claude/skills/director/SKILL.md
echo "### F4 pre-fix description double-quoted in $f (no finding expected)"; cp "$B/$f" "$S_TMP/f4"; cp "$PRE/$f" "$B/$f"; perl -0pi -e 's/^description: ([^\n]*)/description: "$1"/m' "$B/$f"; grep -c '^description: "' "$B/$f"; ruby $SCRATCH/ods4/yamlcheck.rb "$B/$f" | sed "s|$B/||"; run; cp "$S_TMP/f4" "$B/$f"
echo "### F5 templates/base/.agents/skills moved away"; mv "$B/templates/base/.agents/skills" "$S_TMP/skills5"; run; mv "$S_TMP/skills5" "$B/templates/base/.agents/skills"
echo "### F6 control after restore"; run
rm -rf "$S_TMP"
```

Output (the probe's ids F0-F6 are called Y0-Y6 here, apart from the mutants F1-F10 of C4-4):

```text
### Y0 control: the copy as archived
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- PASS: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.02s)
ok  	github.com/yoshpy-dev/ralph/internal/config	1.004s
### Y1 pre-fix description in templates/base/.agents/skills/director/SKILL.md only
    skill_frontmatter_test.go:197: templates/base/.agents/skills/director/SKILL.md:3:155: rule colon-space: the unquoted value of description holds ": ", which YAML reads as a mapping and rejects; quote it (or write it as a > block) or reword it -- see this test's doc comment
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- FAIL: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
FAIL	github.com/yoshpy-dev/ralph/internal/config	0.572s
### Y2 pre-fix description in .claude/skills/director/SKILL.md only
    skill_frontmatter_test.go:197: .claude/skills/director/SKILL.md:3:155: rule colon-space: the unquoted value of description holds ": ", which YAML reads as a mapping and rejects; quote it (or write it as a > block) or reword it -- see this test's doc comment
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- FAIL: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
FAIL	github.com/yoshpy-dev/ralph/internal/config	0.450s
### Y3 ' see #2' appended to the plain description of .agents/skills/test/SKILL.md
1
    skill_frontmatter_test.go:197: .agents/skills/test/SKILL.md:3:180: rule space-hash: the unquoted value of description holds " #", where YAML starts a comment and drops the rest of the value; quote it (or write it as a > block) or reword it -- see this test's doc comment
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- FAIL: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
FAIL	github.com/yoshpy-dev/ralph/internal/config	0.306s
### Y4 pre-fix description double-quoted in .claude/skills/director/SKILL.md (no finding expected)
1
.claude/skills/director/SKILL.md: ok name=director description=665 chars
ok=1 bad=0
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- PASS: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
ok  	github.com/yoshpy-dev/ralph/internal/config	0.651s
### Y5 templates/base/.agents/skills moved away
    skill_frontmatter_test.go:172: walk templates/base/.agents/skills: lstat templates/base/.agents/skills: no such file or directory
    skill_frontmatter_test.go:202: checked the frontmatter of 41 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- FAIL: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
FAIL	github.com/yoshpy-dev/ralph/internal/config	0.494s
### Y6 control after restore
    skill_frontmatter_test.go:202: checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills
--- PASS: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
ok  	github.com/yoshpy-dev/ralph/internal/config	0.466s
```

Each root is walked (Y1 breaks only the template's Codex copy, Y2 only the root Claude copy), the space-hash rule fires on a real skill (Y3), a quoted value of the same text passes and Psych reads it (Y4), and a root that is gone fails the test rather than leaving it silent (Y5).

### C4-4 Mutation (go1.26.0, `go test -overlay`)

#### C4-4.1 Harness

`$SCRATCH/ods4/mut/mutate.py` is cycle 3's harness with its paths moved to `ods4`: each mutant is a list of literal `(old, new)` edits, refused unless `old` occurs exactly once; the mutated copies and an overlay JSON go under `$SCRATCH/ods4/mut/out/<id>/`, and the tests run with `-run` (org: `Director|StopAll|Spawn_DirectorRole|RenderRolePrompt|CodexWritableRoot|PermissionArgs|ResolvePermissionMode`; cli: `OrgDirector|StatusCmd_Director|OrgStopAll|OrgSpawn_RoleDirector|OrgLegacyLedger|CodexAgmsgWritableRoot|CodexSeatModesPossible|ARelativeAgmsgHome`; config: `SkillFrontmatter|PlainFrontmatter`), or with `--full` against whole packages. The list is `$SCRATCH/ods4/mut/mutants.py` (and `mutants_h7.py`). F1-F10 mutate the helper inside the test file `skill_frontmatter_test.go`. The batch ran 04:40-04:49 against 905fc0a7 with the first test of C4-5 already in `director_stop_test.go` (written at 04:38:24) and before the second (04:50:07), so P1-P4 and P6 also list that test; the "h" variants cut it out again. H7 ran after the second test was written. Every edit applied exactly once and every mutant built (F4 and F6 needed a second form: the first ones left `i` and `unicode/utf8` unused).

#### C4-4.2 Results (filtered runs)

| ID | Mutation | Result | Time | Failing tests |
| --- | --- | --- | --- | --- |
| X0 | org: director_start.go unchanged through -overlay | SURVIVED | 2s | - |
| X0s | org: director_stop.go unchanged through -overlay | SURVIVED | 45s | - |
| X0c | cli: org_director_headless.go unchanged through -overlay | SURVIVED | 42s | - |
| X0f | config: skill_frontmatter_test.go unchanged through -overlay | SURVIVED | 1s | - |
| G1 | D14 check removed | red | 46s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G2 | D14 compares the workspace only | red | 48s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G3 | D14 compares the pane only | red | 48s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead` |
| G4 | D14 compares the state too (launched refuses) | red | 47s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead` |
| G5 | MarkStoppedIfUnchanged passes no read (no check) | red | 49s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G6 | stop's unforced record calls MarkStopped (no check) | red | 47s | `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G7 | the forced record checks the ids too | red | 46s | `TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile` |
| G8 | start's cleanup (abort) records with MarkStoppedIfUnchanged against r.reg | red | 50s | `TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` |
| G9 | D14 refusal does not wrap ErrDirectorChangedDuringStop (%w -> %v) | red | 50s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G10 | D14 refusal names the two reads the other way round | red | 49s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G11 | directorRecordedIDs: no workspace / no pane words swapped | red | 49s | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| H1 | AlreadyStopped branch removed (the stopped refusal is a cleanup error again) | red | 50s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H2 | AlreadyStopped only without force | red | 50s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H3 | AlreadyStopped only with force | red | 50s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H4 | AlreadyStopped not set (returns nil with the plain result) | red | 51s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H5 | AlreadyStopped also sets Forced for a forced stop | red | 50s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H6 | AlreadyStopped also sets WasCurrent | red | 49s | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| P1 | directorRecordStopError trims nothing | red | 48s | `TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`, `TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce`, `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`, `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| P2 | directorRecordStopError trims org: director: only | red | 49s | `TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`, `TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce`, `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`, `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| P3 | directorRecordStopError.Unwrap returns nil | red | 51s | `TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`, `TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce`, `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`, `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| P4 | start's cleanup (abort) back to fmt.Errorf(record %s stopped: %w) | red | 48s | `TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`, `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt` |
| P5 | stop's forced branch back to fmt.Errorf(record %s stopped: %w) | red | 49s | `TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce` |
| P6 | stop's general branch back to fmt.Errorf(record %s stopped: %w) | red | 49s | `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`, `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce` |
| P7 | the changed branch drops '; this stop closed only what it had read' | red | 50s | `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| P8 | DirectorCleanupError.Error drops the changed case (says 'record the stop with') | red | 49s | `TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain`, `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| C1 | printOrgDirectorStop ignores AlreadyStopped | red | 43s | `TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun` |
| C2 | printOrgDirectorStop drops the second AlreadyStopped line | red | 42s | `TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun` |
| C3 | stop help drops the D14 sentence | red | 42s | `TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain` |
| C4 | stop help drops the AC8 sentence | red | 46s | `TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun` |
| F1 | the frontmatter check always skips | SURVIVED | 1s | - |
| F2 | a double-quoted value is checked | red | 1s | `TestPlainFrontmatterFindings` |
| F3 | a > or | value is checked | SURVIVED | 1s | - |
| F4 | CR not trimmed | red | 1s | `TestPlainFrontmatterFindings` |
| F5 | colon-end check removed | red | 1s | `TestPlainFrontmatterFindings` |
| F6 | column in bytes | red | 1s | `TestPlainFrontmatterFindings` |
| F7 | the last shape on a line wins, not the first | SURVIVED | 1s | - |
| F8 | colon-space needle removed | red | 1s | `TestPlainFrontmatterFindings` |
| F9 | space-hash points at the space (at 0) | red | 1s | `TestPlainFrontmatterFindings` |
| F10 | indented lines are checked too (key regexp allows leading blanks) | red | 1s | `TestPlainFrontmatterFindings` |
| X0t | control: the new test cut out, nothing else | SURVIVED | 5s | - |
| P9 | DirectorCleanupError.Error drops the not-restored case (says nothing was recorded) | red | 47s | `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt` |
| P9h | P9 with HEAD's tests only | SURVIVED | 45s | - |
| P11 | DirectorCleanupError.Error checks e.Err (after Detached) before not-restored (start's cleanup says nothing more was recorded) | red | 46s | `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt` |
| P11h | P11 with HEAD's tests only | SURVIVED | 46s | - |
| H7org | abort treats a stop recorded first (ErrDirectorStopped) as a failed record (internal/org only, before the new start test existed) | SURVIVED | 5s | - |
| X0t2 | control: the new start test cut out, nothing else | SURVIVED | 7s | - |
| H7 | abort treats a stop recorded first (ErrDirectorStopped) as a failed record | red | 42s | `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord` |
| H7h | H7 with the new start test cut out (HEAD's tests and the first new test) | SURVIVED | 42s | - |

#### C4-4.3 The survivors against whole packages (`-run` dropped, `--full`)

| ID | Mutation | Result | Time | Failing tests |
| --- | --- | --- | --- | --- |
| F1 | the frontmatter check always skips | SURVIVED | 1s | - |
| F3 | a > or | value is checked | SURVIVED | 2s | - |
| F7 | the last shape on a line wins, not the first | SURVIVED | 2s | - |
| P9h | P9 with HEAD's tests only | SURVIVED | 307s | - |
| P11h | P11 with HEAD's tests only | SURVIVED | 307s | - |
| H7h | H7 with the new start test cut out (HEAD's tests and the first new test) | SURVIVED | 312s | - |

#### C4-4.4 Why the survivors survive

- F1 (the test always skips): the skip for a repo without `.claude/skills` is by design (the vendored-repo case). Nothing fails when it skips; the `checked the frontmatter of 54 SKILL.md files` log line shows that it ran (C4-2.3)
- F3 (a value starting with `>` or `|` is checked too): no row and no SKILL.md has a block-scalar header followed by `: ` or ` #` on the key's line (`a: > # note` is valid YAML), so the two only differ there, and in the direction of a false finding
- F7 (the last shape listed wins instead of the first on the line): the row "the first of two shapes on a line" (`x #y: z`) has the earlier shape listed later, so both pick it. Only the rule named in the failure differs (`x: y #z` would tell them apart); the line fails either way
- P9h and P11h (the not-restored case of `DirectorCleanupError.Error` dropped, or put after `e.Err != nil`): no test of HEAD records a stop, or start's cleanup, whose write cannot be cut back, so they survive HEAD's tests in both packages; the first test of C4-5 kills both (P9, P11)
- H7 (start's cleanup takes a stop recorded first, `ErrDirectorStopped`, as a failed record): AC4's rows end start with an attach --replace or a detach --force, which leave the registration open, so start's `MarkStopped` succeeds; no test lets a stop record first. H7org survived internal/org's tests before the second test of C4-5 existed, H7h survives both packages with that test cut out, and the test kills H7

#### C4-4.5 The texts of the not-restored record

`$SCRATCH/ods4/probe/zz_probe_test.go` (added through the overlay only) prints the two errors the first test of C4-5 checks by parts:

```text
    zz_probe_test.go:17: STOP: org: director: stop d1: record d1 stopped: append to $TMPDIR/TestProbe_C4_NotRestoredTexts4165139069/001/director.jsonl: no space left on device, and director.jsonl could not be restored to its size before the append (truncate refused), so it may hold part of the write; director.jsonl may hold part of the write; record the stop with ralph org director stop --id d1
    zz_probe_test.go:33: START: org: director: start d1: herdr agent start director_0123456789ab (agent_start_retries=0): agent start: exited; its cleanup did not finish: record d1 stopped: append to $TMPDIR/TestProbe_C4_NotRestoredTexts4165139069/003/director.jsonl: no space left on device, and director.jsonl could not be restored to its size before the append (truncate refused), so it may hold part of the write; director.jsonl may hold part of the write; record the stop with ralph org director stop --id d1
ok  	github.com/yoshpy-dev/ralph/internal/org	0.416s
```

Both say "may hold part of the write" twice (the store's text, then `DirectorCleanupError.Error`'s note), the same shape as the doubled "nothing was recorded" that S12 handed to /sync-docs. The text comes from S1b; directorRecordStopError only cut the doubled prefix. The test checks the parts and does not pin the doubled phrase.

### C4-5 The tests this cycle added (10efa82c)

`git show --stat 10efa82c`:

```text
10efa82c test: pin the director stop records that cannot be cut back or that a stop made first

 internal/org/director_start_test.go | 46 +++++++++++++++++++
 internal/org/director_stop_test.go  | 92 +++++++++++++++++++++++++++++++++++++
 2 files changed, 138 insertions(+)
```

- `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt` (`internal/org/director_stop_test.go`), two subtests: a stop of a running d1, and start's cleanup after a failed agent start, whose record's write stops halfway (`directorPartialWrite`) and whose cut back fails. The error is a `*DirectorCleanupError` that wraps `ErrDirectorWriteNotRestored`, ENOSPC, and the truncate error, starts with the prefix once, says `; director.jsonl may hold part of the write` and not that nothing (more) was recorded, names the stop and no herdr command. The file holds the whole detached and part of the stopped (one corrupt line): nothing is current and d1 is open. Then the stop it names, run with the write working and herdr no longer knowing the pane and the workspace, records the stopped and leaves nothing open. Kills P9 and P11
- `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord` (`internal/org/director_start_test.go`): a stop runs inside the fake agent start, closes ws-1 and pane-1, and records detached and stopped; start's `MarkLaunched` then finds no current registration, its cleanup finds that herdr no longer knows what it made and its `MarkStopped` gets `ErrDirectorStopped`. Start returns `ErrDirectorReplacedDuringStart` ending in `; start closed what it had made in herdr, and d1 is recorded stopped`, not a `*DirectorCleanupError`, sends no C-c and no close after the agent start, and the trail is attached, workspace_created, pane_created, detached, stopped. Kills H7
