# Verify evidence: ralph org merge-order CLI probe (org-merge-order, cycle 1)

- Date: 2026-10-14
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md` (Verify plan, "Evidence to capture")
- Report: `docs/reports/verify-2026-10-14-org-merge-order.md`
- How: a scratch bash script (not committed) builds the HEAD binary, makes a bare origin and a work repo under a `mktemp -d` dir, seeds `<tmp>/state/manifest.jsonl` with `scope_reserved` bindings and `gate_passed` / `gate_failed` records shaped like the fixtures of `internal/cli/org_merge_order_test.go` (`bind`, `mergeOrderGateDetails`), and runs `ralph org merge-order` with PATH set to a temp bin dir that holds only a git wrapper (`<tmp>/bin`), or the wrapper plus a fake `gh` that prints canned JSON per `--head` (`<tmp>/bin-gh`). HOME is an empty `<tmp>/home`, and git reads no global or system config.
- Redaction: the temp dir is `<tmp>` and `$HOME` is `~`. Commit SHAs and times are from the run.
- Observation (R6b): from a directory outside any repository that holds `origin.git`, git reads the name `origin` as the local path `./origin.git`, so merge-order lists that repository. That is git's lookup for a remote name that is not configured, not merge-order's code. R6 runs from an empty directory and exits 1.

## Setup

```
worktree HEAD: 346304fca7a29ef2110e41808d5d4e8ebc9d8350
binary: GOTOOLCHAIN=go1.25.8 go build -o <tmp>/ralph ./cmd/ralph
PATH dirs: <tmp>/bin = {git -> exec /opt/homebrew/bin/git}; <tmp>/bin-gh = {git, gh (fake, canned JSON per --head)}
GIT_CONFIG_GLOBAL=<tmp>/gitconfig (empty) GIT_CONFIG_NOSYSTEM=1
M0 (first main)    a825d2050c00d1d3ba64db18bb8770a173fb1838
M1 (main now)      12cb28a0aa0740e1837edea40a4556618b3bdc59
feat/alpha         3debdc2488d4f4e29a86f51246634ea120362cee (gated on M0)
feat/beta          4065acad27d339b8162da8b55f3c593a7d1e4e56 (gated on M1)
feat/eta           52f95658ff574392357204770a11d4f8a87800a5 (gated on M1)
feat/gamma gated   c6b1a3aa02138df527795d93f10be18eb1c0ba23, pushed after the gate: 29b532537a82c55bab6b8d8f9a63abb8ac1a2f81
feat/delta         4c0241d104cc090095ce2db1aa9058c649e71ee0 (gated on M1, never pushed)
feat/zeta          f0a37b8805927dd8d2bc8da2411eef5b3fe648aa (gate_passed, then gate_failed test_failed on the same head)
PATH=<tmp>/bin: command -v gh -> not found
$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'
ref: refs/heads/main	HEAD
12cb28a0aa0740e1837edea40a4556618b3bdc59	HEAD
3debdc2488d4f4e29a86f51246634ea120362cee	refs/heads/feat/alpha
4065acad27d339b8162da8b55f3c593a7d1e4e56	refs/heads/feat/beta
52f95658ff574392357204770a11d4f8a87800a5	refs/heads/feat/eta
29b532537a82c55bab6b8d8f9a63abb8ac1a2f81	refs/heads/feat/gamma
f0a37b8805927dd8d2bc8da2411eef5b3fe648aa	refs/heads/feat/zeta
12cb28a0aa0740e1837edea40a4556618b3bdc59	refs/heads/main
```

Seeded ledger (<tmp>/state/manifest.jsonl):

```
{"ts":"2026-10-14T00:00:01Z","org_id":"org-a","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-a/ split=demo feature=alpha digest=0123456789ab branch=feat/alpha approval=per-plan"}
{"ts":"2026-10-14T00:00:02Z","org_id":"org-a","seat_id":"","event":"gate_passed","details":"head=3debdc2488d4f4e29a86f51246634ea120362cee base=a825d2050c00d1d3ba64db18bb8770a173fb1838 base_ref=refs/remotes/origin/main branch=feat/alpha by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:03Z","org_id":"org-a","seat_id":"","event":"disbanded"}
{"ts":"2026-10-14T00:00:04Z","org_id":"org-b","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-b/ split=demo feature=beta digest=0123456789ab branch=feat/beta approval=per-plan"}
{"ts":"2026-10-14T00:00:05Z","org_id":"org-e","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-e/ split=demo feature=eta digest=0123456789ab branch=feat/eta approval=per-plan"}
{"ts":"2026-10-14T00:00:06Z","org_id":"org-g","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-g/ split=demo feature=gamma digest=0123456789ab branch=feat/gamma approval=per-plan"}
{"ts":"2026-10-14T00:00:07Z","org_id":"org-d","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-d/ split=demo feature=delta digest=0123456789ab branch=feat/delta approval=per-plan"}
{"ts":"2026-10-14T00:00:08Z","org_id":"org-z","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-z/ split=demo feature=zeta digest=0123456789ab branch=feat/zeta approval=per-plan"}
{"ts":"2026-10-14T00:00:09Z","org_id":"org-x","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-x/ split=demo feature=epsilon digest=0123456789ab branch=feat/epsilon approval=per-plan"}
{"ts":"2026-10-14T00:00:10Z","org_id":"org-a","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-a/ split=demo feature=alpha2 digest=0123456789ab branch=feat/alpha2 approval=per-plan"}
{"ts":"2026-10-14T00:00:11Z","org_id":"org-b","seat_id":"","event":"gate_passed","details":"head=4065acad27d339b8162da8b55f3c593a7d1e4e56 base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/beta by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:12Z","org_id":"org-e","seat_id":"","event":"gate_passed","details":"head=52f95658ff574392357204770a11d4f8a87800a5 base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/eta by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:13Z","org_id":"org-g","seat_id":"","event":"gate_passed","details":"head=c6b1a3aa02138df527795d93f10be18eb1c0ba23 base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/gamma by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:14Z","org_id":"org-d","seat_id":"","event":"gate_passed","details":"head=4c0241d104cc090095ce2db1aa9058c649e71ee0 base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/delta by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:15Z","org_id":"org-z","seat_id":"","event":"gate_passed","details":"head=f0a37b8805927dd8d2bc8da2411eef5b3fe648aa base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/zeta by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:16Z","org_id":"org-z","seat_id":"","event":"gate_failed","details":"reason=test_failed head=f0a37b8805927dd8d2bc8da2411eef5b3fe648aa base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/zeta by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T00:00:17Z","org_id":"org-x","seat_id":"","event":"gate_passed","dry_run":true,"details":"head=12cb28a0aa0740e1837edea40a4556618b3bdc59 base=12cb28a0aa0740e1837edea40a4556618b3bdc59 base_ref=refs/remotes/origin/main branch=feat/epsilon by=probe herdr_pane=unset log=\"/x\""}
{"ts":"2026-10-14T00:00:18Z","org_id":"org-i","seat_id":"","event":"scope_reserved","worktree":"<tmp>/work","details":"paths=internal/org-i/ split=demo feature=incomplete branch=feat/incomplete approval=per-plan"}
```

## Ledger and refs before the runs (AC9)

```
$ (cd <tmp>/state && find . -type f | sort | xargs shasum -a 256; wc -lc manifest.jsonl; /bin/ls -A)
9d9441cb6de754ecc89a915694dbad234a7df3901f1d685c85f96697091f7cdd  ./manifest.jsonl
      18    5624 manifest.jsonl
manifest.jsonl
$ git -C <tmp>/work for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
886cfa6bc65074c6dcbf08f232cfcd31ff7476b023cfc0f5b3ea4ee05188215f  -
```

### R1 text, no gh (AC1 ready/rebase/regate, AC2 failed/no_gate/no_remote, AC3, AC4 dry-run/incomplete, AC5 no gh, AC6)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:02Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/beta	-	4065acad27d3	12cb28a0aa07	org-b/beta	the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	ready	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	the gate passed on 52f95658ff57, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
3	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	the gate failed (test_failed): fix it and run ralph org gate again
6	no_remote	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	origin has no branch feat/delta: it was not pushed, or was deleted after a merge
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase (rebase them on origin/main and run ralph org gate again; the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T20:17:02Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### R2 --json, no gh (AC8)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --json --state-dir <tmp>/state
--- stdout
{
  "default_branch": "main",
  "default_head": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
  "listed_at": "2026-10-13T20:17:02Z",
  "gh_available": false,
  "entries": [
    {
      "rank": 1,
      "class": "ready",
      "branch": "feat/beta",
      "remote_head": "4065acad27d339b8162da8b55f3c593a7d1e4e56",
      "detail": "the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "4065acad27d339b8162da8b55f3c593a7d1e4e56",
        "base": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T00:00:11Z"
      },
      "binding": {
        "org_id": "org-b",
        "feature": "beta",
        "split": "demo",
        "ts": "2026-10-14T00:00:04Z"
      }
    },
    {
      "rank": 2,
      "class": "ready",
      "branch": "feat/eta",
      "remote_head": "52f95658ff574392357204770a11d4f8a87800a5",
      "detail": "the gate passed on 52f95658ff57, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "52f95658ff574392357204770a11d4f8a87800a5",
        "base": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T00:00:12Z"
      },
      "binding": {
        "org_id": "org-e",
        "feature": "eta",
        "split": "demo",
        "ts": "2026-10-14T00:00:05Z"
      }
    },
    {
      "rank": 3,
      "class": "rebase",
      "branch": "feat/alpha",
      "remote_head": "3debdc2488d4f4e29a86f51246634ea120362cee",
      "detail": "origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "3debdc2488d4f4e29a86f51246634ea120362cee",
        "base": "a825d2050c00d1d3ba64db18bb8770a173fb1838",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T00:00:02Z"
      },
      "binding": {
        "org_id": "org-a",
        "feature": "alpha",
        "split": "demo",
        "ts": "2026-10-14T00:00:01Z"
      }
    },
    {
      "rank": 4,
      "class": "regate",
      "branch": "feat/gamma",
      "remote_head": "29b532537a82c55bab6b8d8f9a63abb8ac1a2f81",
      "detail": "pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "c6b1a3aa02138df527795d93f10be18eb1c0ba23",
        "base": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T00:00:13Z"
      },
      "binding": {
        "org_id": "org-g",
        "feature": "gamma",
        "split": "demo",
        "ts": "2026-10-14T00:00:06Z"
      }
    },
    {
      "rank": 5,
      "class": "failed",
      "branch": "feat/zeta",
      "remote_head": "f0a37b8805927dd8d2bc8da2411eef5b3fe648aa",
      "detail": "the gate failed (test_failed): fix it and run ralph org gate again",
      "gate": {
        "event": "gate_failed",
        "readable": true,
        "head": "f0a37b8805927dd8d2bc8da2411eef5b3fe648aa",
        "base": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
        "base_ref": "refs/remotes/origin/main",
        "reason": "test_failed",
        "ts": "2026-10-14T00:00:16Z"
      },
      "binding": {
        "org_id": "org-z",
        "feature": "zeta",
        "split": "demo",
        "ts": "2026-10-14T00:00:08Z"
      }
    },
    {
      "rank": 6,
      "class": "no_remote",
      "branch": "feat/delta",
      "remote_head": "",
      "detail": "origin has no branch feat/delta: it was not pushed, or was deleted after a merge",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "4c0241d104cc090095ce2db1aa9058c649e71ee0",
        "base": "12cb28a0aa0740e1837edea40a4556618b3bdc59",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T00:00:14Z"
      },
      "binding": {
        "org_id": "org-d",
        "feature": "delta",
        "split": "demo",
        "ts": "2026-10-14T00:00:07Z"
      }
    },
    {
      "rank": 7,
      "class": "no_gate",
      "branch": "feat/epsilon",
      "remote_head": "",
      "detail": "no gate record: run ralph org gate in the feature worktree before the PR is merged",
      "binding": {
        "org_id": "org-x",
        "feature": "epsilon",
        "split": "demo",
        "ts": "2026-10-14T00:00:09Z"
      }
    },
    {
      "rank": 8,
      "class": "no_gate",
      "branch": "feat/alpha2",
      "remote_head": "",
      "detail": "no gate record: run ralph org gate in the feature worktree before the PR is merged",
      "binding": {
        "org_id": "org-a",
        "feature": "alpha2",
        "split": "demo",
        "ts": "2026-10-14T00:00:10Z"
      }
    }
  ],
  "notes": [
    "gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked",
    "2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase (rebase them on origin/main and run ralph org gate again; the gate runs only while the feature org runs)",
    "the order is as of the ls-remote at 2026-10-13T20:17:02Z and is not a guarantee: origin/main can move before a merge"
  ]
}
--- stderr
--- exit=0
```

### R3 fake gh, default (AC5 merged/closed hidden, fork PR skipped; wrong_base)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:02Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/beta	#11 https://example.invalid/pull/11	4065acad27d3	12cb28a0aa07	org-b/beta	the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
3	wrong_base	feat/eta	#13 https://example.invalid/pull/13	52f95658ff57	12cb28a0aa07	org-e/eta	PR #13 merges into release, not origin's default branch main: change the PR's base to main
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
6	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T20:17:02Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### R4 fake gh, --all (AC5 merged/closed listed last)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh ralph org merge-order --all --state-dir <tmp>/state
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:03Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/beta	#11 https://example.invalid/pull/11	4065acad27d3	12cb28a0aa07	org-b/beta	the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
3	wrong_base	feat/eta	#13 https://example.invalid/pull/13	52f95658ff57	12cb28a0aa07	org-e/eta	PR #13 merges into release, not origin's default branch main: change the PR's base to main
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
6	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
7	merged	feat/delta	#9 https://example.invalid/pull/9	4c0241d104cc	12cb28a0aa07	org-d/delta	PR #9 is merged
8	closed	feat/zeta	#10 https://example.invalid/pull/10	f0a37b880592	12cb28a0aa07	org-z/zeta	PR #10 was closed without a merge
note: the order is as of the ls-remote at 2026-10-13T20:17:03Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

## Refs and files after R1-R4 (no fetch: same refs digest as before; nothing newer than the marker)

```
$ git -C <tmp>/work for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
886cfa6bc65074c6dcbf08f232cfcd31ff7476b023cfc0f5b3ea4ee05188215f  -
$ find <tmp>/state <tmp>/work <tmp>/origin.git <tmp>/home -newer <tmp>/marker | wc -l
       0
```

### R5 repository with no origin (AC7 exit 1)

```
$ cd <tmp>/noorigin && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
--- stderr
org: merge-order: git ls-remote --symref origin HEAD 'refs/heads/*' in . failed (exit status 128: fatal: 'origin' does not appear to be a git repository
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.): origin's state is unknown, so nothing was classified; merge-order reads origin, so run it in a repository whose origin answers
--- exit=1
```

### R6 cwd outside any repository (AC7 exit 1)

```
$ cd <tmp>/norepo && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
--- stderr
org: merge-order: git ls-remote --symref origin HEAD 'refs/heads/*' in . failed (exit status 128: fatal: 'origin' does not appear to be a git repository
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.): origin's state is unknown, so nothing was classified; merge-order reads origin, so run it in a repository whose origin answers
--- exit=1
```

### R6b cwd outside any repository, next to <tmp>/origin.git (observation: git reads the name origin as the path ./origin.git)

```
$ cd <tmp> && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:04Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/beta	-	4065acad27d3	12cb28a0aa07	org-b/beta	the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	ready	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	the gate passed on 52f95658ff57, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
3	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	the gate failed (test_failed): fix it and run ralph org gate again
6	no_remote	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	origin has no branch feat/delta: it was not pushed, or was deleted after a merge
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase (rebase them on origin/main and run ralph org gate again; the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T20:17:04Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### R7 origin HEAD names a missing branch (AC7 exit 1)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
--- stderr
org: merge-order: git ls-remote --symref origin HEAD 'refs/heads/*' in . printed no HEAD symref, so origin's default branch is unknown and nothing was classified: point origin's HEAD at the default branch and run it again
--- exit=1
```

### R8 origin HEAD -> trunk, old main kept (AC11 rebase)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/trunk 12cb28a0aa07 (listed 2026-10-13T20:17:04Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate again (the gate runs only while the feature org runs)
2	rebase	feat/beta	-	4065acad27d3	12cb28a0aa07	org-b/beta	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate again (the gate runs only while the feature org runs)
3	rebase	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate again (the gate runs only while the feature org runs)
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	the gate failed (test_failed): fix it and run ralph org gate again
6	no_remote	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	origin has no branch feat/delta: it was not pushed, or was deleted after a merge
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: the order is as of the ls-remote at 2026-10-13T20:17:04Z and is not a guarantee: origin/trunk can move before a merge
--- stderr
--- exit=0
```

### R9 --org-id refused

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --org-id org-b --state-dir <tmp>/state
--- stdout
--- stderr
org: merge-order: --org-id does not apply: merge-order lists the feature branches of every org in the ledger
--- exit=1
```

Appended to a copy (<tmp>/state-unreadable): one gate_passed of org-b with no branch= and a non-SHA head.
Appended to another copy (<tmp>/state-corrupt): a cut-off last line with no newline.

### R10 unreadable gate record (AC4)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state-unreadable
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:04Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	the gate passed on 52f95658ff57, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
3	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
4	unreadable	feat/beta	-	-	-	org-b/beta	the latest gate record (gate_passed at 2026-10-14T00:00:19Z) cannot be read: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
5	failed	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	the gate failed (test_failed): fix it and run ralph org gate again
6	no_remote	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	origin has no branch feat/delta: it was not pushed, or was deleted after a merge
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: the order is as of the ls-remote at 2026-10-13T20:17:04Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### R11 corrupt trailing line (AC4)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state-corrupt
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:17:05Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	unreadable	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	a corrupt manifest line follows the latest gate record (gate_passed at 2026-10-14T00:00:02Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
2	unreadable	feat/beta	-	4065acad27d3	12cb28a0aa07	org-b/beta	a corrupt manifest line follows the latest gate record (gate_passed at 2026-10-14T00:00:11Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
3	unreadable	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	a corrupt manifest line follows the latest gate record (gate_passed at 2026-10-14T00:00:12Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
4	unreadable	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	a corrupt manifest line follows the latest gate record (gate_passed at 2026-10-14T00:00:13Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
5	unreadable	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	a corrupt manifest line follows the latest gate record (gate_passed at 2026-10-14T00:00:14Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
6	unreadable	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	a corrupt manifest line follows the latest gate record (gate_failed at 2026-10-14T00:00:16Z) and may be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: the manifest has 1 corrupt line(s) that cannot be read: a branch whose latest gate record comes before one is unreadable until ralph org gate records a new one
note: the order is as of the ls-remote at 2026-10-13T20:17:05Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### R12 manifest that cannot be read (AC7 exit 1)

```
$ cp <tmp>/state/manifest.jsonl <tmp>/state-noperm/ && chmod 000 <tmp>/state-noperm/manifest.jsonl
$ cd <tmp>/work && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state-noperm
--- stdout
--- stderr
org: merge-order: read manifest: org: open <tmp>/state-noperm/manifest.jsonl: open <tmp>/state-noperm/manifest.jsonl: permission denied
--- exit=1
```

### R13 gh that fails for every branch (AC5 exit 0)

```
$ command cat <tmp>/bin-ghfail/gh
#!/bin/sh
echo "To get started with GitHub CLI, please run:  gh auth login" >&2
exit 4
$ cd <tmp>/work && PATH=<tmp>/bin-ghfail ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/main 12cb28a0aa07 (listed 2026-10-13T20:23:15Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/beta	-	4065acad27d3	12cb28a0aa07	org-b/beta	the gate passed on 4065acad27d3, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
2	ready	feat/eta	-	52f95658ff57	12cb28a0aa07	org-e/eta	the gate passed on 52f95658ff57, origin's tip of the branch, with base 12cb28a0aa07, origin/main's tip
3	rebase	feat/alpha	-	3debdc2488d4	a825d2050c00	org-a/alpha	origin/main moved: gate base a825d2050c00, now 12cb28a0aa07: rebase on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
4	regate	feat/gamma	-	c6b1a3aa0213	12cb28a0aa07	org-g/gamma	pushed after the gate: origin has 29b532537a82, the gate checked c6b1a3aa0213: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/zeta	-	f0a37b880592	12cb28a0aa07	org-z/zeta	the gate failed (test_failed): fix it and run ralph org gate again
6	no_remote	feat/delta	-	4c0241d104cc	12cb28a0aa07	org-d/delta	origin has no branch feat/delta: it was not pushed, or was deleted after a merge
7	no_gate	feat/epsilon	-	-	-	org-x/epsilon	no gate record: run ralph org gate in the feature worktree before the PR is merged
8	no_gate	feat/alpha2	-	-	-	org-a/alpha2	no gate record: run ralph org gate in the feature worktree before the PR is merged
note: gh pr list --head feat/alpha failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/beta failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/eta failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/gamma failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/delta failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/zeta failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/epsilon failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh pr list --head feat/alpha2 failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: 2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase (rebase them on origin/main and run ralph org gate again; the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T20:23:15Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

## Ledger after every run (AC9: same as before)

R12 and R13 were run as separate commands after this snapshot was first taken (after R11), so it was taken again after R13; both times it printed the lines below. `<tmp>/home` held 0 entries after R13 (`find <tmp>/home -mindepth 1 | wc -l`).

```
$ (cd <tmp>/state && find . -type f | sort | xargs shasum -a 256; wc -lc manifest.jsonl; /bin/ls -A)
9d9441cb6de754ecc89a915694dbad234a7df3901f1d685c85f96697091f7cdd  ./manifest.jsonl
      18    5624 manifest.jsonl
manifest.jsonl
```
