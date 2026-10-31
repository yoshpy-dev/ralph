# Test evidence: org-followups

先頭が cycle 2、続いて cycle 1。cycle 1 の節は中身を変えず、見出しを 1 段下げた。

## Cycle 2(HEAD 543d29db)

- HEAD: 543d29db(merge-base 1cc84ffa = origin/main)。cycle 1 のあとのコードの変更は F5 25c9fa77 と F6 12d28dae
- 全出力: `docs/evidence/test-2026-10-30-org-followups.log`(gitignore、13,313 行)。cycle 2 の分は 7,579 行目の `cycle 2 (HEAD 543d29db)` の区切りから。`run-test.sh` の meta と全出力、打ち直し、絞った回、F5 のテストの `-v`、変異ごとの差分と `--- FAIL` の行、`-race` の数え、coverage の関数の行の順。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-182544.log`(gitignore)
- 道具: go1.27.3 darwin/arm64、git 2.49.0

### run-test.sh(cycle 2)

- 起動: cycle 1 と同じ。perl の fork と setsid で切り離し、前景の `caffeinate -w <pid>` で待った
- meta:

```
start=2026-11-01T03:25:44+0900 epoch=1793471144 head=543d29db load={ 12.20 11.96 15.96 }
end=2026-11-01T03:52:02+0900 epoch=1793472722 rc=0 load={ 6.12 16.33 25.28 }
```

- 1,578 秒。途中の 1 分の load は 18.58、24.42、50.44、132.83(03:40 ごろ、別のプロセスの iOS simulator と swift-frontend)
- Go(`go test ./...`):

```
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/cli	456.279s
ok  	github.com/yoshpy-dev/ralph/internal/config	10.157s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	599.584s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)
```

- `(cached)` の 7 パッケージの打ち直し(`cmd/ralph` と `internal/insights` は `internal/org` に依存する):

```
$ go test -count=1 ./cmd/ralph ./internal/gitroot ./internal/insights ./internal/org/driver ./internal/org/protocol ./internal/scaffold ./internal/upgrade
rc=0 4s
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	1.042s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	3.451s
ok  	github.com/yoshpy-dev/ralph/internal/insights	1.859s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	2.731s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.735s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.200s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.795s
```

- shell: `tests/test-*.sh` の 45 本すべてに `    OK` の行がある。SKIP は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 行だけ
- テストの数(`go test -list .`): org 1,398 本(cycle 1 から +2)、cli 954 本(+1)

### 絞った回と F5 のテスト(cycle 2)

| コマンド | 選ばれる本数 | rc | 時間 |
| --- | --- | --- | --- |
| `go test -count=1 -run 'Followup\|MergeOrder' ./internal/org/` | 102 | 0 | 7 秒 |
| `go test -count=1 -run 'Status\|OrgMergeOrder\|OrgFollowup' ./internal/cli/` | 117 | 0 | 119 秒 |
| `go test -count=1 -v -run 'TestMergeOrderFollowup_(GHAvailableWithNoFeatureBranch\|LookupNeverClearsTheEntriesGHAvailable\|GHMissingMakesEveryFollowupUnknown\|GHFailureMakesOnlyItsBranchUnknown\|ReportedWithTargetsAndWithout)' ./internal/org/` | 5(サブテスト 9) | 0 | 2 秒 |
| `go test -count=1 -v -run 'TestOrgMergeOrder_(FollowupOnlyJSONSaysGHAvailable\|FollowupWithoutGHIsUnknown\|FollowupJSONHasEveryOpenFollowup)' ./internal/cli/` | 3 | 0 | 8 秒 |

`GHAvailableWithNoFeatureBranch` のサブテストは gh_answers、gh_fails_for_the_branch、gh_is_missing、gh_goes_missing_after_an_answer、no_record、only_a_done_follow-up の 6 件で、すべて PASS。

### -race(cycle 2)

```
$ go test -race -count=3 ./internal/org/ -run 'Followup|MergeOrder' -v
rc=0 58s
トップレベルの --- PASS 306(102 本 × 3)、--- FAIL 0、WARNING: DATA RACE 0
ok  	github.com/yoshpy-dev/ralph/internal/org	26.538s
```

### 変異(cycle 2)

worktree のファイルは書き換えていない。`internal/org/mergeorder_followup.go` を差し替えた写しを scratch に作り、`go test -overlay` で入れた。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じで、どの回も `build failed` はない。

| ID | 場所と変更 | org(`Followup\|MergeOrder`) | cli(`Status\|OrgMergeOrder\|OrgFollowup`) |
| --- | --- | --- | --- |
| mA | `:136` `res.GHAvailable = res.GHAvailable \|\| ghFound` → `res.GHAvailable = ghFound` | rc 1、13 秒 | rc 1、79 秒 |
| mB | `:202` の `ghFound = false` を消す | rc 1、13 秒 | rc 0、127 秒 |
| mC | `:205` の `ghFound = true` を `lookups[branch] = followupLookup{prs: prs, limitFull: limitFull}` の前に移す | rc 1、13 秒 | rc 0、101 秒 |
| mD | `:136` `\|\| ghFound` → `\|\| (ghFound && false)` | rc 1、13 秒 | rc 1、134 秒 |
| mE | `case why != "":` のあとに `ghFound = false` | rc 0、18 秒 | rc 0、93 秒 |
| mF | `case ctx.Err() != nil:` のあとに `ghFound = true` | rc 0、19 秒 | rc 0、88 秒 |

落ちたテスト:

```
mA org: TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable
          mergeorder_followup_test.go:740: GHAvailable = false with entries [{Rank:1 Class:ready Branch:feat/a ...}]; want true and feat/a
mA cli: TestOrgMergeOrder_AllListsMergedAndClosedPRsTheDefaultLeavesOut
          org_merge_order_test.go:625: --all --json: gh_available false, entries [{Rank:1 Class:ready Branch:feat/open ...
mB org: TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch/gh_goes_missing_after_an_answer
          mergeorder_followup_test.go:710: GHAvailable = true with entries []; want false and no entry
mC org: TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch/gh_fails_for_the_branch
          mergeorder_followup_test.go:710: GHAvailable = false with entries []; want true and no entry
mD org: TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch/{gh_answers, gh_fails_for_the_branch}
          mergeorder_followup_test.go:710: GHAvailable = false with entries []; want true and no entry
mD cli: TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable
          org_merge_order_followup_test.go:294: gh_available = false with entries []; want true and no entry
```

mE と mF はどちらの絞り方でも残った(報告の Test gaps の 8 と 9)。

### coverage(cycle 2、org の絞った回の profile)

```
$ go test -count=1 -run 'Followup|MergeOrder' -coverprofile=... ./internal/org/      22 秒、パッケージ全体 13.2%
```

| 関数 | cycle 1 | cycle 2 |
| --- | --- | --- |
| `lookupFollowupMerges` | 94.6% | 95.0%(打たれていないのは `internal/org/mergeorder_followup.go:213-214`、origin の確かめの途中の中断) |
| `mergeOrderFollowups` | 100% | 100% |
| `MergeOrder` | 96.9% | 96.9% |
| `(*FollowupStore).Add`、`appendLocked`、`recordedMergeHead` | 94.4%、68.8%、80.0% | 同じ |

## Cycle 1(HEAD a4cab358)

- HEAD: a4cab358(merge-base 1cc84ffa = origin/main)
- 全出力: `docs/evidence/test-2026-10-30-org-followups.log`(gitignore、cycle 1 の時点で 7,577 行)。`run-test.sh` の全出力、絞った回、`-race` の回、変異ごとの差分と出力、coverage の関数の行を順に入れた。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-163815.log`(gitignore)
- 道具: go1.27.3 darwin/arm64、git 2.49.0

### run-test.sh

- 起動: Bash tool の `run_in_background` は使っていない。perl の fork と setsid で切り離し、INT・HUP・QUIT・TERM を既定に戻し、標準入出力を /dev/null にした。runner が log と meta を書き、前景の `caffeinate -w <pid>` で待った
- meta:

```
start=2026-11-01T01:38:14+0900 epoch=1793464694 load={ 8.97 7.92 8.95 }
end=2026-11-01T02:06:35+0900 epoch=1793466395 rc=0 load={ 7.78 8.09 8.81 }
```

- 1,701 秒。途中で、下の絞った回・変異・`-race` の回を前景で並べて打った。別の worktree(script-local-config)の `go test ./...` も同じ時間に動いていた。途中の load の見本は 6.28、7.99、8.52、8.79、10.15
- Go(`go test ./...`、`(cached)` はない):

```
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	1.356s
ok  	github.com/yoshpy-dev/ralph/internal/cli	652.572s
ok  	github.com/yoshpy-dev/ralph/internal/config	14.721s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	4.289s
ok  	github.com/yoshpy-dev/ralph/internal/insights	3.839s
ok  	github.com/yoshpy-dev/ralph/internal/org	795.902s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	5.564s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	4.652s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.220s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.807s
```

- shell: `tests/test-*.sh` の 45 本すべてに `    OK` の行がある(45 行)。SKIP は `tests/test-secret-scan-branch.sh` の `SKIP: the real git older than 2.41 case (git 2.49.0 is 2.41 or later; the stubbed old git covers it)` の 1 行だけ
- テストの数(`go test -list .`): org 1,396 本、cli 953 本、config 100 本。この diff で増えたテストの関数は 62 本、消えたものはない

### 絞った回(変異の基準)

| コマンド | 選ばれる本数 | rc | 時間 |
| --- | --- | --- | --- |
| `go test -count=1 -run 'Followup\|MergeOrder' ./internal/org/` | 100 | 0 | 6 秒 |
| `go test -count=1 -run 'Status\|OrgMergeOrder\|OrgFollowup' ./internal/cli/` | 116 | 0 | 108 秒 |

### -race

```
$ go test -race -count=3 ./internal/org/ -run 'Followup' -v
rc=0 17s
--- PASS の行 135(45 本 × 3)、--- FAIL 0、WARNING: DATA RACE 0
ok  	github.com/yoshpy-dev/ralph/internal/org	5.265s
```

### 変異

worktree のファイルは書き換えていない。変異ごとに 1 ファイルを差し替えた写しを scratch に作り、`go test -overlay` で入れた(置換は「元の文字列がちょうど 1 回現れる」ことを確かめる perl の helper)。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じだった。どの回も `build failed` はない。

| ID | 場所と変更 | org(`Followup\|MergeOrder`) | cli(`Status\|OrgMergeOrder\|OrgFollowup`) |
| --- | --- | --- | --- |
| m1 | `internal/org/mergeorder_followup.go:295` `case after != nil:` → `case after != nil && false:` | rc 1、13 秒 | rc 1、114 秒 |
| m2 | `internal/cli/status_followup.go:161` `; merged {` → `; merged && false {` | — | rc 1、134 秒 |
| m3 | `internal/org/followup.go:289` `return withFileLock(filepath.Dir(s.path), followupLockFile, "followup", fn)` → `return fn()` | rc 1、13 秒 | rc 1、87 秒 |
| m3b | `internal/org/followup.go:189` Done だけ lock を外す: `err = s.withLock(func() error {` → `err = func(fn func() error) error { return fn() }(func() error {` | `-count=20 -run 'Followup'` で rc 1、17 秒。落ちたのは 20 回のうち 7 回 | — |
| m4 | `internal/cli/status_followup.go:153` の `statusFollowup` の先頭に `_ = exec.Command("gh", "pr", "list", "--head", f.AfterBranch).Run()`(import に `os/exec`) | — | rc 1、147 秒 |
| m5 | `internal/org/followup.go:498` `&& mergedBefore[head] {` → `&& mergedBefore[head] && false {` | rc 1、12 秒 | rc 1、96 秒 |
| m6 | `internal/org/followup.go:489` `case !ts.Before(added):` → `case !ts.Before(added) \|\| true:` | rc 1、11 秒 | rc 1、115 秒 |
| m7 | `internal/org/mergeorder_followup.go:344` `case at.Before(added):` → `case at.Before(added) && false:` | rc 1、12 秒 | rc 1、112 秒 |

#### 落ちたテスト(`--- FAIL` の行)

m1(merge-order の ready の判定を外す):

```
org: TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs/{merged_after_the_record_is_ready, merged_at_the_time_of_the_record_is_ready, merged_after_the_record_under_a_newer_open_PR_is_ready}
     TestMergeOrderFollowup_OneLookupPerAfterBranch, _EntriesAndTheirLookupsStayAsTheyWere, _GHFailureMakesOnlyItsBranchUnknown(3 件),
     _InterruptedLookupIsNoError, _JSONCarriesEveryFieldAndANullPR, _PinnedOriginReadsItsOwnPRs
cli: TestOrgMergeOrder_FollowupSectionFollowsTheNotes
       follow-up row 1 = ["f1" "270" "feat/done" "waiting" "gh" ...] want [... "ready" ...]
     TestOrgMergeOrder_FollowupJSONHasEveryOpenFollowup
```

m2(status の ready の判定を外す):

```
cli: TestStatusCmd_FollowupsReadyFromTheLedgersMerges(f1 feat/merged が waiting)
     TestStatusCmd_FollowupsRunNoGH(f1 が waiting)
```

m3(lock を外す。Add と Done の両方):

```
org: TestFollowupAdd_ConcurrentCallsGetDistinctIDs
       followup_test.go:549: ID f2 was handed out twice: [f3 f2 f4 f5 f2 f2 f5 f4 f6 f1]
     TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites
       followup_test.go:637: Add returned {id:f4 err:<nil>} while followups.lock was held
     TestFollowupStore_ReportsAnUnreadableRecordAndAnUnavailableLock/followups.lock_is_a_directory
       followup_test.go:672: Add err = <nil>, want org: open followup lock file ...
     TestFollowupStore_LivesUnderTheGivenStateDir(副作用: lock が state dir を作らないので open が失敗)
cli: TestOrgFollowupAdd_ValidationErrorsWriteNothing, TestOrgFollowup_LegacyLedgerGuard,
     TestOrgMergeOrder_FollowupWithoutGHIsUnknown(3 本とも同じ副作用: followups.jsonl の open が no such file or directory)
```

m3b(Done だけ lock を外す、`-count=20`):

```
7 --- FAIL: TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce
  followup_test.go:587: 5 Dones reported changed, want exactly 1(ほかの回は 6、2、3、3、4 …)
ほかのテストは 20 回とも通った
```

m4(status から gh を呼ぶ):

```
cli: TestStatusCmd_FollowupsRunNoGH
       status_followup_test.go:705: status ran gh: ["pr list --head feat/merged" "pr list --head feat/open" ...]
```

m5(self-review M1 の同じ head の除外を外す):

```
org: TestFollowupRecordedMerge/a_pr_merged_before_the_add_and_the_worktree_cleaned_of_its_head_after_it
     TestFollowupRecordedMerge/the_earlier_merge's_pr_merged_after_its_worktree_cleaned_in_file_order
     TestFollowupRecordedMerge/the_earlier_merge's_worktree_cleaned,_then_a_pr_merged_of_another_head_after_the_add
cli: TestStatusCmd_FollowupsReadyFromTheLedgersMerges(f3 feat/old が ready になる)
```

m6(status の D5 の時刻の比較を外す):

```
org: TestFollowupRecordedMerge の 7 件(a_pr_merged_before_the_add_(a_branch_name_used_again)、times_compare_as_instants,_not_as_text など)
cli: TestStatusCmd_FollowupsReadyFromTheLedgersMerges
```

m7(merge-order の D5 の時刻の比較を外す):

```
org: TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs/{merged_before_the_record_is_waiting,
       an_older_merged_PR_and_a_newer_open_one_is_waiting_(AC8), closed_after_an_older_merged_one_is_closed,
       gh's_limit_of_PRs_with_none_merged_after_is_unknown}
cli: TestOrgMergeOrder_FollowupSectionFollowsTheNotes
       follow-up row 4 = ["f4" "273" "feat/old" "ready" ...] want [... "waiting" ...]
     TestOrgMergeOrder_FollowupJSONHasEveryOpenFollowup
```

### coverage(絞った回の profile)

```
$ go test -count=1 -run 'Followup|MergeOrder' -coverprofile=... ./internal/org/      12 秒、パッケージ全体 13.1%
$ go test -count=1 -run 'Status|OrgMergeOrder|OrgFollowup' -coverprofile=... ./internal/cli/   94 秒、パッケージ全体 25.1%
```

100% でない関数と、打たれていないブロック:

| 関数 | statement | 打たれていないブロック |
| --- | --- | --- |
| `(*FollowupStore).Add` | 94.4% | `internal/org/followup.go:163`(appendLocked のエラーの return) |
| `(*FollowupStore).appendLocked` | 68.8% | `:324`(marshal)、`:329`(open)、`:332`(write)、`:336`(close)のエラー |
| `recordedMergeHead` | 80.0% | `:554`(head= のない Details) |
| `lookupFollowupMerges` | 94.6% | `internal/org/mergeorder_followup.go:202-203`(origin の確かめの途中の中断) |
| `newOrgFollowupAddCmd` | 95.0% | `internal/cli/org_followup.go:103`(store.Add のエラー) |
| `newOrgFollowupListCmd` | 95.0% | `internal/cli/org_followup.go:143`(store.Read のエラー) |

そのほかの新しい関数(`followup.go` の 26 本中 23 本、`mergeorder_followup.go` の 8 本中 7 本、`status_followup.go` の 6 本、`org_followup.go` の 7 本中 5 本、`printMergeOrderFollowups`)は 100%。変更した既存の関数は `MergeOrder` 96.9%、`runStatus` 93.5%、`buildStatusOrgJSON` 85.7%、`printStatusEmpty`・`printStatusJSONAllOrgs`・`printStatusTableAllOrgs`・`printMergeOrder` 100%。
