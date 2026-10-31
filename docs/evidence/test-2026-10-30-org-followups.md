# Test evidence: org-followups(cycle 1)

- HEAD: a4cab358(merge-base 1cc84ffa = origin/main)
- 全出力: `docs/evidence/test-2026-10-30-org-followups.log`(gitignore、7,577 行)。`run-test.sh` の全出力、絞った回、`-race` の回、変異ごとの差分と出力、coverage の関数の行を順に入れた。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-163815.log`(gitignore)
- 道具: go1.27.3 darwin/arm64、git 2.49.0

## run-test.sh

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

## 絞った回(変異の基準)

| コマンド | 選ばれる本数 | rc | 時間 |
| --- | --- | --- | --- |
| `go test -count=1 -run 'Followup\|MergeOrder' ./internal/org/` | 100 | 0 | 6 秒 |
| `go test -count=1 -run 'Status\|OrgMergeOrder\|OrgFollowup' ./internal/cli/` | 116 | 0 | 108 秒 |

## -race

```
$ go test -race -count=3 ./internal/org/ -run 'Followup' -v
rc=0 17s
--- PASS の行 135(45 本 × 3)、--- FAIL 0、WARNING: DATA RACE 0
ok  	github.com/yoshpy-dev/ralph/internal/org	5.265s
```

## 変異

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

### 落ちたテスト(`--- FAIL` の行)

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

## coverage(絞った回の profile)

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
