# Gate log excerpts (org recheck 2026-10-24)

The three logs of `ralph org gate` and `ralph org regate` (`gates/` of the recheck ledger, about 347 KB and
5320 lines each) are not committed. Below are, for each, its header, the times its two scripts started, the
branch secret scan, and the lines that give its verdict, each prefixed with its line number in the original
log. No gate failed, so there are no failure lines to show. The logs hold no `GATE:` line (the verb prints
that on its own output); the verdict recorded is the manifest event named above each excerpt. The last
section has the lines of `scripts/check-sync.sh` and `tests/test-check-sync.sh`, which are the same in all
three logs.

## backfill-na-20261025T025423Z.log

gate of backfill-na: `gate_passed` (manifest 03:09:33Z). Total lines in the log: 5320.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ralph-iso/ralph/.claude/worktrees/org-backfill-na
3: # branch: fix/backfill-na
4: # head: 5d72c2393a9eb4d29d2ed2b94570ea726395f67e
5: # base: refs/remotes/origin/main ffb3e271b92c54aa7f21bc3548ac3530882c79f4
7: ==> ralph org gate: running ./scripts/run-static-verify.sh
9: - Timestamp: 2026-10-25T02:54:23Z
429: secret-scan-branch: scanned ffb3e271..5d72c239 against origin/main: clean
431: ==> All verifiers passed.
435: ==> ralph org gate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org gate: running ./scripts/run-test.sh
439: - Timestamp: 2026-10-25T02:54:48Z
5305: ?   	github.com/yoshpy-dev/ralph	[no test files]
5306: ?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
5307: ok  	github.com/yoshpy-dev/ralph/internal/cli	343.620s
5308: ok  	github.com/yoshpy-dev/ralph/internal/config	0.589s
5309: ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
5310: ok  	github.com/yoshpy-dev/ralph/internal/org	399.859s
5311: ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
5312: ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
5313: ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
5314: ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
5316: ==> All verifiers passed.
5320: ==> ralph org gate: ./scripts/run-test.sh exited 0
```

## hooks-matcher-20261025T030201Z.log

gate of hooks-matcher: `gate_passed` (manifest 03:16:07Z). Total lines in the log: 5320. It ran beside
backfill-na's gate until 03:09:33Z.

```text
1: # ralph org gate
2: # worktree: /private/tmp/ralph-iso/ralph/.claude/worktrees/org-hooks-matcher
3: # branch: fix/hooks-matcher
4: # head: b31e623ac24761ca652b08e36c26c0e140f2ce5e
5: # base: refs/remotes/origin/main ffb3e271b92c54aa7f21bc3548ac3530882c79f4
7: ==> ralph org gate: running ./scripts/run-static-verify.sh
9: - Timestamp: 2026-10-25T03:02:01Z
429: secret-scan-branch: scanned ffb3e271..b31e623a against origin/main: clean
431: ==> All verifiers passed.
435: ==> ralph org gate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org gate: running ./scripts/run-test.sh
439: - Timestamp: 2026-10-25T03:02:19Z
5305: ?   	github.com/yoshpy-dev/ralph	[no test files]
5306: ?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
5307: ok  	github.com/yoshpy-dev/ralph/internal/cli	283.458s
5308: ok  	github.com/yoshpy-dev/ralph/internal/config	0.642s
5309: ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
5310: ok  	github.com/yoshpy-dev/ralph/internal/org	349.282s
5311: ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
5312: ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
5313: ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
5314: ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
5316: ==> All verifiers passed.
5320: ==> ralph org gate: ./scripts/run-test.sh exited 0
```

## hooks-matcher-20261025T140359Z.log

regate of hooks-matcher (director, through the wrapper): `gate_passed` via=regate (manifest 14:18:21Z).
Total lines in the log: 5320. The head is the merge commit of origin/main 536271a2 into b31e623a.

```text
1: # ralph org regate
2: # worktree: /private/tmp/ralph-iso/ralph/.claude/worktrees/org-hooks-matcher
3: # branch: fix/hooks-matcher
4: # head: 77c631b2a635faaa7f79600302da08b925770c58
5: # base: refs/remotes/origin/main 536271a2c0a274be93b124905a3f622c1ea73ad0
7: ==> ralph org regate: running ./scripts/run-static-verify.sh
9: - Timestamp: 2026-10-25T14:04:00Z
429: secret-scan-branch: scanned 536271a2..77c631b2 against origin/main: clean
431: ==> All verifiers passed.
435: ==> ralph org regate: ./scripts/run-static-verify.sh exited 0
437: ==> ralph org regate: running ./scripts/run-test.sh
439: - Timestamp: 2026-10-25T14:04:21Z
5305: ?   	github.com/yoshpy-dev/ralph	[no test files]
5306: ?   	github.com/yoshpy-dev/ralph/cmd/ralph	[no test files]
5307: ok  	github.com/yoshpy-dev/ralph/internal/cli	297.436s
5308: ok  	github.com/yoshpy-dev/ralph/internal/config	2.193s
5309: ok  	github.com/yoshpy-dev/ralph/internal/insights	1.656s
5310: ok  	github.com/yoshpy-dev/ralph/internal/org	379.822s
5311: ok  	github.com/yoshpy-dev/ralph/internal/org/driver	2.298s
5312: ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
5313: ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
5314: ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
5316: ==> All verifiers passed.
5320: ==> ralph org regate: ./scripts/run-test.sh exited 0
```

## check-sync in all three logs

Lines 366, 393 to 398, 400, and 652 to 669 of each log, here from backfill-na's (blank lines 392 and 399 left
out). The other two logs have the same lines, except the worktree's name in line 653
(`org-hooks-matcher`). Each gate ran on a head with the branch's reviewer report (`review-`) and org report
(`org-manifest-`) committed, and `ROOT_ONLY` stayed 0 (F8); lines 654 to 669 are the cases of
`tests/test-check-sync.sh` (G1).

```text
366: ==> scripts/check-sync.sh
393: === Sync Summary ===
394:   IDENTICAL:      172
395:   DRIFTED:        0
396:   ROOT_ONLY:      0
397:   TEMPLATE_ONLY:  11
398:   KNOWN_DIFF:     5
400: PASS: all files in sync.
652: ==> tests/test-check-sync.sh
653: == check-sync.sh over a scratch tree (/private/tmp/ralph-iso/ralph/.claude/worktrees/org-backfill-na/scripts/check-sync.sh) ==
654:   PASS: baseline tree passes (exit 0)
655:   PASS: review- and org-manifest- reports pass (exit 0)
656:   PASS: review-x.md is excluded
657:   PASS: org-manifest-x.md is excluded
658:   PASS: a report under another name fails (exit 1)
659:   PASS: other-x.md is ROOT_ONLY
660:   PASS: review-x.md is still excluded
661:   PASS: org-manifest-x.md is still excluded
662:   PASS: reviewer.md fails (exit 1)
663:   PASS: reviewer.md is ROOT_ONLY (review- matches only part of the name)
664:   PASS: x-review-y.md fails (exit 1)
665:   PASS: x-review-y.md is ROOT_ONLY (review- is not at the start of the name)
666:   PASS: an excluded prefix after another directory fails (exit 1)
667:   PASS: docs/old/docs/reports/review-x.md is ROOT_ONLY
668:   PASS: self-review- report passes (exit 0)
669:   PASS: self-review-x.md is excluded
```
