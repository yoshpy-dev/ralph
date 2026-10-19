# Test evidence: org-director-wait

- HEAD 45f5d586 (base origin/main e38b89a9), macOS arm64 (Darwin 25.5.0, 12 CPUs), go1.26.0 and go1.25.8 (`GOTOOLCHAIN`)
- 2026-10-19 13:35 to 16:00 JST. Load average 4 to 45 during the runs (50 in the deliberate load run of section 6)
- `<scratch>` is the session scratchpad outside the worktree; `<tmp>` replaces the per-user temp path; `~` is `$HOME`
- No test or code file was changed. Mutants and probes were overlay files under `<scratch>` read through `go test -overlay`; `git status --porcelain` was empty before and after, and `git status --porcelain --ignored` (without `.harness/cache/`) differed only by the run-test.sh log `docs/evidence/verify-2026-10-19-043532.log`
- Go runs never overlapped as two whole-package runs: every command below ran alone, one after another
- The tester stopped once on an API connection error after the mutation run had started in the background; that run finished on its own (its record below is complete), and the work went on from the recorded results

## 1. `GOFLAGS=-timeout=30m ./scripts/run-test.sh` (go1.26.0)

```
# Verification run
- Timestamp: 2026-10-19T04:35:32Z
- Mode: test
- Requested scope: changed
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
ok  	github.com/yoshpy-dev/ralph/internal/cli	287.378s
ok  	github.com/yoshpy-dev/ralph/internal/config	0.528s
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	453.330s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)

==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-10-19-043532.log
rc=0 elapsed=887s
```

Shell suites (local verifier), 41, each ending `OK`; no `FAIL` line other than `FAIL: 0`; one skip (`tests/test-secret-scan-branch.sh`: the real git older than 2.41 case, git is 2.49.0):

```
tests/test-agent-models.sh tests/test-agent-phase-boundaries.sh tests/test-archive-plan.sh
tests/test-branch-name.sh tests/test-check-mojibake.sh tests/test-check-skill-sync.sh
tests/test-check-template.sh tests/test-codex-exec-invocation.sh
tests/test-detect-changed-languages.sh tests/test-detect-languages-terraform.sh
tests/test-ensure-pr-ready.sh tests/test-ensure-pr-title-prefix.sh tests/test-gc-artifacts.sh
tests/test-hook-wiring.sh tests/test-insights-append.sh tests/test-language-pack-monorepo-roots.sh
tests/test-lib-json.sh tests/test-new-feature-plan.sh tests/test-no-loop-references.sh
tests/test-plan-visual.sh tests/test-post-edit-verify.sh tests/test-pr-owner-lookup.sh
tests/test-pre-bash-guard.sh tests/test-ralph-config.sh tests/test-ralph-dispatch.sh
tests/test-ralph-worktree.sh tests/test-run-verify-branch-secret-scan.sh
tests/test-run-verify-concurrency.sh tests/test-run-verify-scope.sh
tests/test-secret-scan-branch.sh tests/test-secret-scan.sh tests/test-self-review-scope.sh
tests/test-skill-insight-cycle.sh tests/test-sync-skills.sh tests/test-template-purity.sh
tests/test-terraform-gitignore.sh tests/test-terraform-pack-verify.sh
tests/test-terraform-rule-frontmatter.sh tests/test-verify-local-hook-tests.sh
tests/test-verify-mode-split.sh tests/test-xreview-helpers.sh
```

## 2. `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -v -timeout 30m`

```
go version go1.25.8 darwin/arm64
ok  	github.com/yoshpy-dev/ralph/internal/cli	493.185s
ok  	github.com/yoshpy-dev/ralph/internal/config	0.903s
ok  	github.com/yoshpy-dev/ralph/internal/insights	1.153s
FAIL	github.com/yoshpy-dev/ralph/internal/org	573.810s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	3.359s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	3.090s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	1.893s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	3.929s
rc=1 elapsed=577s
```

Top-level results per package (`--- PASS` / `--- FAIL` / `--- SKIP` at column 0): cli 810/0/0, config 56/0/0, insights 36/0/0, org 1094/1/1, org/driver 51/0/0, org/protocol 15/0/0, scaffold 30/0/2, upgrade 114/0/0. Total 2,210 top-level: 2,206 PASS, 1 FAIL, 3 SKIP; 5,981 PASS lines with subtests. The 44 new top-level tests are all among the PASS lines (`comm` of the new `func Test` names against them).

The one failure (`internal/org/director_cmd_test.go:310`, not changed by this diff):

```
=== RUN   TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime
    director_cmd_test.go:310: stopDirectorProcess = {outcome:ended signalled:true note:process=ended after SIGTERM err:<nil>}, want not ended within 300ms
--- FAIL: TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime (0.27s)
```

The 1- and 5-minute load averages right after this run were 36 and 45. The cause is the one `docs/tech-debt/README.md:279` records: the test sleeps 100 ms for `sh -c "trap '' TERM; exec sleep 30"` to reach the exec, and on a loaded machine the SIGTERM arrives first.

The same test alone, then the org package alone:

```
GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run '^TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime$' -count=10 -v -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	5.475s     (10 PASS lines)
rc=0 elapsed=9s

GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -count=1 -v -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	452.606s
rc=0 elapsed=454s            (top-level 1,095 PASS, 0 FAIL, 1 SKIP; 3,681 PASS lines)
```

## 3. `-race` (go1.26.0)

```
go test -race ./internal/org/ ./internal/cli/ -run 'Director|WaitInbox|OrgWait|ExitCode|SignalCause|OrgGate|Status|Escalate|Watch' -count=1 -v -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	79.937s     (388 PASS, 0 FAIL, 1 SKIP)
ok  	github.com/yoshpy-dev/ralph/internal/cli	162.373s    (275 PASS, 0 FAIL, 0 SKIP)
rc=0 elapsed=173s
WARNING: DATA RACE lines: 0; 1,852 PASS lines with subtests; all 44 new top-level tests PASS
```

The skip in sections 2 and 3 is `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`, which skips on darwin by design; the scaffold skips are `TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS`.

## 4. Isolation (AC9): canary on PATH

Real `herdr`, `gh`, `tmux`, `orca` are in `/opt/homebrew/bin` and `osascript`, `open`, `curl` in `/usr/bin` on this machine; `agmsg` is installed too. A canary for each of `herdr gh tmux orca osascript open agmsg codex claude curl` (logs `<name> <args>` to `<scratch>/canary/calls.log`, exits 1) went first on PATH:

```
PATH=<scratch>/canary/bin:$PATH sh -c 'command -v herdr gh tmux orca osascript open'
<scratch>/canary/bin/herdr ... <scratch>/canary/bin/open      (all six resolve to the canary)

PATH=<scratch>/canary/bin:$PATH go test ./internal/org/ -run '^(<the 25 new org tests>)$' -count=1 -v -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	4.074s
PATH=<scratch>/canary/bin:$PATH go test ./internal/cli/ -run '^(<the 19 new cli tests>)$' -count=1 -v -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/cli	6.042s
top-level PASS 44, FAIL 0; canary calls: 0
```

Sensitivity check: an overlay of `internal/org/director_wait.go` that runs `exec.Command("herdr", "--version")` before `begin`, run only with the canary first on PATH (the command checks `command -v herdr` first):

```
--- PASS: TestWaitInboxForDirector_AnOpenItemEndsWithItems (0.38s)
--- PASS: TestWaitInboxForDirector_TimesOutWithThePhraseAfterANote (0.24s)
ok  	github.com/yoshpy-dev/ralph/internal/org	1.528s
canary calls:
herdr --version
herdr --version
```

Ledger and signals: the default ledger `<main checkout>/.harness/state/org` and the worktree's `.harness/state/org` did not exist before or after; nothing under `<main checkout>/.git/ralph/` changed after 13:35 JST (`find -newermt`); under `<main checkout>/.harness/state/` only the hook files `edited-files.log`, `needs-verify`, and `tool_failures.count` changed; no `director-wait.*` file appeared in the worktree. The new tests send signals only to the test process with a catcher registered first (`signalSelf`) and to child copies of the test binary they start (`child.Process.Signal`, `child.Process.Kill`).

## 5. Coverage of the new code (filtered runs)

```
go test ./internal/org/ -run 'DirectorWait|WaitInbox' -count=1 -coverprofile=<scratch>/cover-org.out -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	6.124s	coverage: 9.5% of statements
go test ./internal/cli/ -run 'OrgWaitDirector|OrgWaitInbox|DirectorWaitLine|ExitCode|SignalCause|OrgDirectorStatus|DirectorLine|StatusCmd_Director' \
  -count=1 -coverpkg=./internal/cli/,./internal/org/ -coverprofile=<scratch>/cover-cli.out -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/cli	15.852s	coverage: 18.8% of statements in ./internal/cli/, ./internal/org/
```

`go tool cover -func`, org run, `internal/org/director_wait.go`:

```
DirectorWaitPathIn 100.0%   ReadDirectorWait 100.0%    DirectorWaitFor 100.0%
CheckDirectorWaitID 100.0%  WaitInboxForDirector 86.7%  directorWaitCheckCurrent 100.0%
newDirectorWaitRun 54.5%    begin 96.4%                loop 97.6%
probe 100.0%                probeAgain 92.9%           refresh 100.0%
endNoWatch 100.0%           endError 100.0%            endSignal 100.0%
end 100.0%                  save 100.0%                owns 100.0%
sayLost 66.7%               sayWriteFailed 100.0%      directorWaitItemsDetail 66.7%
directorWaitWatchFacts 100.0%  directorWaitWatchState 87.5%
directorWaitTSOf 66.7%      directorWaitMarkTS 66.7%
escalate.go: WaitInbox 95.7%  openInboxItems 100.0%
```

cli run: `ExitCode` 100.0%, `readOrgDirectorWait` / `orgDirectorWaitFacts` / `orgDirectorWaitLine` / `orgDirectorWaitClause` 100.0%, `readOrgDirectorReport` / `orgDirectorLine` / `printOrgDirectorReport` 100.0%, `notIgnoredSignals` 100.0%, `runOrgWaitInboxDirector` 88.9%, `runOrgWaitInbox` 90.9%, `printInboxWaitItems` 100.0%, `signalCauseContext` 95.8%. `reraiseSignal` shows 0.0% because it runs only in the child copy of the test binary (`TestOrgWaitDirector_ASignalEndsTheProcessByIt`), whose coverage is not merged.

Blocks of `director_wait.go` with count 0 in both runs (the `newDirectorWaitRun` defaults at 357-371 are covered by the cli run):

```
275-280  nil Inbox / empty StateDir guards
417-419  begin: decide() when director-wait.lock could not be taken
439-441  loop: InboxPollInterval <= 0 default
531-533  probeAgain: ctx done during a round's settle
649-653  sayLost with no mark (the file was removed)
672      directorWaitItemsDetail: "<n> open inbox items" (n > 1)
708-709  not_running with a mark state other than running/stopping/stopped
725      the fallback "the watch is <state>"
730-732  directorWaitTSOf(nil)
739-741  directorWaitMarkTS("")
```

## 6. Timing under load

16 `perl -e '1 while 1'` loops on 12 CPUs (killed by script path afterwards; `pgrep` found none left), load average 9 at the start and 50 at the end:

```
go test ./internal/org/ -run '^(<the 25 new org tests>)$' -count=5 -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/org	18.254s
go test ./internal/cli/ -run '^(<the 19 new cli tests>)$' -count=5 -timeout 30m
ok  	github.com/yoshpy-dev/ralph/internal/cli	53.022s
elapsed=72s
```

## 7. Mutation

56 overlay mutants, 50 killed on the filtered runs; the 6 survivors also survive their whole package; 5 of them are killed by one probe test each, and 1 is equivalent. The table, the first failing assertion of each kill, the whole-package reruns, and the probe source are in `docs/evidence/test-2026-10-19-org-director-wait-mutation.txt`.
