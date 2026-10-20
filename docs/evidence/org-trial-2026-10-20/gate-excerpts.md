# Gate log excerpts (org trial 2026-10-20)

The six logs of `ralph org gate` and `ralph org regate` (`gates/` of the trial ledger, about 320 KB each)
are not committed. Below are, for each, its header, the lines that give its verdict, and the lines that
show why a failing gate failed, each prefixed with its line number in the original log. The logs hold no
`GATE:` line (the verb prints that on its own output); the verdict recorded is the manifest event named
above each excerpt. `$TMPDIR/` stands for the macOS per-user temporary directory.

## doctor-escape-20261020T045853Z.log

gate 1 of doctor-escape: `gate_failed` reason=static_failed (manifest 04:59:21Z), F8. Total lines in the log: 445.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-doctor-escape
3: # branch: fix/doctor-escape
4: # head: 8d6c11a69269fb2d3e309392f4c15ae05e5796a4
5: # base: refs/remotes/origin/main f0668004015323a2366ac7983a50e2d27017dfa2
7: ==> ralph org gate: running ./scripts/run-static-verify.sh
389: === Checking for root-only files not in templates/base/ ===
390:   ROOT_ONLY      docs/reports/org-manifest-doctor-escape-2026-10-20.md
391:   ROOT_ONLY      docs/reports/review-2026-10-20-doctor-escape.md
392: 
393: === Checking for packs/languages/ files not in templates/packs/ ===
394: 
395: === Sync Summary ===
396:   IDENTICAL:      171
397:   DRIFTED:        0
398:   ROOT_ONLY:      2
399:   TEMPLATE_ONLY:  11
400:   KNOWN_DIFF:     5
401: 
402: === Issues Found ===
403:   - ROOT_ONLY: docs/reports/org-manifest-doctor-escape-2026-10-20.md (exists at root but missing in templates/base/)
404:   - ROOT_ONLY: docs/reports/review-2026-10-20-doctor-escape.md (exists at root but missing in templates/base/)
405: 
406: FAIL: 2 sync issue(s) found.
441: ==> Some verifiers failed.
445: ==> ralph org gate: ./scripts/run-static-verify.sh exited 1
```

## doctor-escape-20261020T050127Z.log

gate 2 of doctor-escape: `gate_failed` reason=test_failed (manifest 05:33:54Z), F9. Total lines in the log: 5100.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-doctor-escape
3: # branch: fix/doctor-escape
4: # head: 9526f18800035c82c19cb8bd6a46936a43c6f088
5: # base: refs/remotes/origin/main f0668004015323a2366ac7983a50e2d27017dfa2
435: ==> ralph org gate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org gate: running ./scripts/run-test.sh
5080: ok  	github.com/yoshpy-dev/ralph/internal/cli	657.725s
5083: --- FAIL: TestRegate_FetchFailures (6.31s)
5084:     --- FAIL: TestRegate_FetchFailures/the_fetch_runs_past_the_bound (1.20s)
5085:         regate_test.go:771: refusal "org: regate: git fetch origin +refs/heads/main:refs/remotes/origin/main in $TMPDIR/TestRegate_FetchFailuresthe_fetch_runs_past_the_bound2157056441/001/wt-auth failed (exit 128), so nothing was recorded: check origin (git remote get-url origin) and its credentials and run ralph org regate --branch feat/auth again; git said:\n  fatal: Could not read from remote repository.\n  Please make sure you have the correct access rights\n  and the repository exists." does not say "git fetch origin +refs/heads/main:refs/remotes/origin/main in $TMPDIR/TestRegate_FetchFailuresthe_fetch_runs_past_the_bound2157056441/001/wt-auth ran past 300ms and was stopped, so nothing was recorded"
5086:         regate_test.go:771: refusal "org: regate: git fetch origin +refs/heads/main:refs/remotes/origin/main in $TMPDIR/TestRegate_FetchFailuresthe_fetch_runs_past_the_bound2157056441/001/wt-auth failed (exit 128), so nothing was recorded: check origin (git remote get-url origin) and its credentials and run ralph org regate --branch feat/auth again; git said:\n  fatal: Could not read from remote repository.\n  Please make sure you have the correct access rights\n  and the repository exists." does not say "check the network and origin's credentials"
5087: FAIL
5088: FAIL	github.com/yoshpy-dev/ralph/internal/org	751.744s
5093: FAIL
5095: ==> Some verifiers failed.
5099: ==> ralph org gate: ./scripts/run-test.sh exited 1
5100: ==> ralph org gate: stopped the processes ./scripts/run-test.sh left running after it exited
```

## report-escape-20261020T053326Z.log

gate of report-escape: `gate_passed` (manifest 05:56:21Z). Total lines in the log: 5093.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-report-escape
3: # branch: fix/report-escape
4: # head: 82c4483db7f291a982b10743769e4057a5088694
5: # base: refs/remotes/origin/main f0668004015323a2366ac7983a50e2d27017dfa2
431: ==> All verifiers passed.
435: ==> ralph org gate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org gate: running ./scripts/run-test.sh
5080: ok  	github.com/yoshpy-dev/ralph/internal/cli	434.700s
5083: ok  	github.com/yoshpy-dev/ralph/internal/org	775.290s
5089: ==> All verifiers passed.
5093: ==> ralph org gate: ./scripts/run-test.sh exited 0
```

## doctor-escape-20261020T053719Z.log

gate 3 of doctor-escape: `gate_passed` (manifest 06:05:03Z). Total lines in the log: 5093.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-doctor-escape
3: # branch: fix/doctor-escape
4: # head: 9526f18800035c82c19cb8bd6a46936a43c6f088
5: # base: refs/remotes/origin/main f0668004015323a2366ac7983a50e2d27017dfa2
431: ==> All verifiers passed.
435: ==> ralph org gate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org gate: running ./scripts/run-test.sh
5080: ok  	github.com/yoshpy-dev/ralph/internal/cli	(cached)
5083: ok  	github.com/yoshpy-dev/ralph/internal/org	968.715s
5089: ==> All verifiers passed.
5093: ==> ralph org gate: ./scripts/run-test.sh exited 0
```

## doctor-escape-20261020T062210Z.log

regate 1 of doctor-escape (director, no GOFLAGS): `gate_failed` reason=test_failed via=regate (manifest 06:55:20Z), F10. Total lines in the log: 6425.

```text
1: # ralph org regate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-doctor-escape
3: # branch: fix/doctor-escape
4: # head: 36d3491fccd4960177a152b12b9180ef6c5c9431
5: # base: refs/remotes/origin/main a9e8ea04423dff2dca7cb854d41328291e00d64c
435: ==> ralph org regate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org regate: running ./scripts/run-test.sh
6168: panic: test timed out after 10m0s
6169: 	running tests:
6170: 		TestOrgWatch_RunsTheWakeCmdOfTheMainWorktreeInItsRoot (0s)
6307: FAIL	github.com/yoshpy-dev/ralph/internal/cli	601.405s
6310: panic: test timed out after 10m0s
6311: 	running tests:
6312: 		TestRegate_RefusedBeforeTheFetch (13s)
6313: 		TestRegate_RefusedBeforeTheFetch/an_untracked_file (0s)
6414: FAIL	github.com/yoshpy-dev/ralph/internal/org	602.724s
6419: FAIL
6421: ==> Some verifiers failed.
6425: ==> ralph org regate: ./scripts/run-test.sh exited 1
```

## doctor-escape-20261020T065639Z.log

regate 2 of doctor-escape (director, GOFLAGS=-timeout=30m): `gate_passed` via=regate (manifest 07:09:51Z). Total lines in the log: 5094.

```text
1: # ralph org regate
2: # worktree: /private/tmp/ot/ralph/.claude/worktrees/org-doctor-escape
3: # branch: fix/doctor-escape
4: # head: c5c65f5f38b1fe5aebc45bfbae73217f6be7ae84
5: # base: refs/remotes/origin/main a9e8ea04423dff2dca7cb854d41328291e00d64c
431: ==> All verifiers passed.
435: ==> ralph org regate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org regate: running ./scripts/run-test.sh
5080: ok  	github.com/yoshpy-dev/ralph/internal/cli	230.333s
5083: ok  	github.com/yoshpy-dev/ralph/internal/org	296.800s
5089: ==> All verifiers passed.
5093: ==> ralph org regate: ./scripts/run-test.sh exited 0
5094: ==> ralph org regate: stopped the processes ./scripts/run-test.sh left running after it exited
```
