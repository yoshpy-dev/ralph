# Test report: org-followups

- Date: 2026-11-01(実行日。ファイル名は計画の日付)
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(`## Test plan` と AC1〜AC8、Progress の S1〜S5 と F1〜F4)。Refs #270
- Tester: tester サブエージェント(`/test`、cycle 1。`cycle-count.json` は 1)
- Scope: `git diff origin/main...HEAD`(merge-base 1cc84ffa、HEAD a4cab358。verify のあとの F4 a4cab358 で merge-order の follow-up の表に SOURCE の列が入った版)。Go は `internal/org`、`internal/cli`。ほかは skill と写し、README、AGENTS.md、仕様、tech-debt。`./scripts/run-test.sh` は changed scope のまま打った(言語は golang)。test mode は `tests/test-*.sh` を glob で全部打つので、shell の 45 本もすべて走った。テストの関数はこの diff で 62 本増え、消えたものはない
- Evidence: `docs/evidence/test-2026-10-30-org-followups.md`(要点。commit する)、`docs/evidence/test-2026-10-30-org-followups.log`(gitignore。`run-test.sh` の全出力、追加の回の出力、変異の差分と出力、coverage の関数の行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-163815.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.49.0) | 1,701 秒(01:38:14〜02:06:35 JST)。load は開始 8.97 / 7.92 / 8.95、終了 7.78 / 8.09 / 8.81 |
| └ Go: `internal/org` | 1,396 本(トップレベル、`go test -list`) | ok | 0 | 0 | 795.9 秒 |
| └ Go: `internal/cli` | 953 本(同) | ok | 0 | 0 | 652.6 秒 |
| └ Go: `cmd/ralph`、`internal/{config,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 1.4〜14.7 秒 |
| `go test -count=1 -run 'Followup\|MergeOrder' ./internal/org/`(変異の基準) | 100 本 | 100 | 0 | 0 | 6 秒 |
| `go test -count=1 -run 'Status\|OrgMergeOrder\|OrgFollowup' ./internal/cli/`(変異の基準) | 116 本 | 116 | 0 | 0 | 108 秒 |
| `go test -race -count=3 ./internal/org/ -run 'Followup'` | 45 本 × 3 | 135 | 0 | 0 | 17 秒。`WARNING: DATA RACE` は 0 |
| 変異 8 個(`go test -overlay`、上の 2 つの絞り方) | 8 | 8 個とも落ちた(1 個は 20 回中 7 回) | — | — | 1 個 11〜147 秒 |
| coverage の回(上の 2 つの絞り方に `-coverprofile`) | — | ok | 0 | 0 | 12 秒、94 秒 |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打つ前の load は 1 分 8.97・5 分 7.92 だった。

`run-test.sh` が動いている間に、絞った回、変異、`-race` の回を前景で 1 つずつ打った。別の worktree(script-local-config)の `go test ./...` も同じ時間に動いていた。それでも load は 6.3〜10.2 にとどまり、30 分の時間切れも既知の flaky なテストの失敗もなかった。打ち直しはしていない。shell の SKIP の行は前の回の報告と同じ 1 行だけだった。

## Coverage

- Statement(絞った回の profile。パッケージ全体の数字は絞ったので意味がない: org 13.1%、cli 25.1%):
  - `internal/org/followup.go`: 26 本中 23 本が 100%。`Add` 94.4%、`appendLocked` 68.8%、`recordedMergeHead` 80.0%
  - `internal/org/mergeorder_followup.go`: 8 本中 7 本が 100%。`lookupFollowupMerges` 94.6%
  - `internal/cli/status_followup.go`: 6 本すべて 100%
  - `internal/cli/org_followup.go`: 7 本中 5 本が 100%。`newOrgFollowupAddCmd`・`newOrgFollowupListCmd` 95.0%
  - 変更した既存の関数: `MergeOrder` 96.9%、`runStatus` 93.5%、`buildStatusOrgJSON` 85.7%、`printStatusEmpty`・`printStatusJSONAllOrgs`・`printStatusTableAllOrgs`・`printMergeOrder`・`printMergeOrderFollowups` 100%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes: 打たれていないブロックは、書き込みの失敗(`internal/org/followup.go:163`、`:324`、`:329`、`:332`、`:336`)、head= のない Details(`:554`)、origin の確かめの途中の中断(`internal/org/mergeorder_followup.go:202-203`)、CLI の add と list の store のエラー(`internal/cli/org_followup.go:103`、`:143`)だけ。Test gaps の 2〜5 に書いた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

`run-test.sh`、絞った 2 回、`-race` の回、coverage の 2 回に失敗はなかった。

## Test plan とテストの対応

| 計画の項目 | テスト |
| --- | --- |
| store の追加・完了・読み出し | `TestFollowupStore_AddReadAndDone`(`internal/org/followup_test.go:133`)、`TestFollowupRead_MissingStateDirIsEmptyAndCreatesNothing`(`:111`) |
| ID の採番(壊れた行を含む)、壊れた最後の行 | `TestFollowupAdd_NeverReusesTheIDOfADamagedLine`(`:290`)、`TestFollowupAdd_TornLastLineStaysSeparateAndTheNewEventIsReadable`(`:329`)、`TestFollowupAdd_CountsAnIDWrittenWithJSONEscapes`(`:362`)、`TestFollowupAdd_FailsWhenNoIDIsLeft`(`:376`) |
| 二度目の done、知らない ID | `TestFollowupDone_SecondDoneChangesNothing`(`:186`)、`TestFollowupDone_UnknownIDIsErrFollowupUnknownID`(`:209`)、CLI は `TestOrgFollowupDone_SecondDoneAndUnknownID`(`internal/cli/org_followup_test.go:267`) |
| 知らない event、壊れた行の数え | `TestFollowupRead_FoldsInIDOrderAndCountsWhatItCannotUse`(`internal/org/followup_test.go:228`)、`TestOrgFollowup_WarnsAboutDamageOnStderr`(`internal/cli/org_followup_test.go:410`) |
| 検証(issue、ブランチ名、note) | `TestValidateFollowupAdd`(`internal/org/followup_test.go:393`)、`TestFollowupNote_DecidesAsValidateInboxNote`(`:446`)、`TestFollowupAddAndDone_RefusedInputWritesNothing`(`:498`)、`TestOrgFollowupAdd_ValidationErrorsWriteNothing`(`internal/cli/org_followup_test.go:333`) |
| CLI の add・list・done(表、`--all`、`--json`、`--org-id` を断る) | `TestOrgFollowup_AddListDoneRoundTrip`(`:122`)、`TestOrgFollowup_ListJSON`(`:177`)、`TestOrgFollowup_EmptyRecord`(`:231`)、`TestOrgFollowup_RefusesOrgID`(`:383`)、`TestOrgFollowup_Help`(`:579`) |
| merge-order の節と JSON(偽の gh で merged・open・closed) | `TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs`(`internal/org/mergeorder_followup_test.go:158`、13 行)、`TestOrgMergeOrder_FollowupSectionFollowsTheNotes`(`internal/cli/org_merge_order_followup_test.go:138`)、`TestOrgMergeOrder_FollowupJSONHasEveryOpenFollowup`(`:180`) |
| merge-order の gh なし・gh の失敗 | `TestMergeOrderFollowup_GHMissingMakesEveryFollowupUnknown`(`internal/org/mergeorder_followup_test.go:311`)、`_GHFailureMakesOnlyItsBranchUnknown`(`:332`)、`TestOrgMergeOrder_FollowupWithoutGHIsUnknown`(`internal/cli/org_merge_order_followup_test.go:260`) |
| status の予約と follow-up、gh を走らせない | `TestStatusCmd_ReservationLinesInTheirGroups`(`internal/cli/status_followup_test.go:248`)、`_ReservationJSON`(`:291`)、`_ReservationOnlyLedgerShowsItsGroup`(`:338`)、`_FollowupsReadyFromTheLedgersMerges`(`:414`)、`_FollowupsRunNoGH`(`:668`) |
| 回帰: 今の merge-order・status・inbox・merge・cleanup・regate のテスト | `run-test.sh` の中で org 1,396 本、cli 953 本がすべて通った |
| 回帰: status の JSON の空と中身ありの形 | `TestStatusCmd_FollowupJSONOnBothPaths`(`:477`)、`TestStatusCmd_ReservationJSON`(`:291`) |
| 同じブランチを待つ follow-up が 2 つ | `TestMergeOrderFollowup_OneLookupPerAfterBranch`(`internal/org/mergeorder_followup_test.go:241`) |
| 追加より前に merge 済みのブランチ | `StateComesFrom…` の「merged before the record is waiting」、`TestFollowupRecordedMerge`(`internal/org/followup_recorded_merge_test.go:46`)の「a pr_merged before the add」 |
| 同じブランチの古い merged の PR と新しい open の PR | `StateComesFrom…` の「an older merged PR and a newer open one is waiting (AC8)」 |
| 追加より前の merge の cleanup(同じ head、別の head) | `TestFollowupRecordedMerge` の「a pr_merged before the add and the worktree_cleaned of its head after it」と「… of another head after it (the branch used again)」、status は `seedStatusFollowups` の f3(`internal/cli/status_followup_test.go:377`) |
| lock の取り合い | `TestFollowupAdd_ConcurrentCallsGetDistinctIDs`(`internal/org/followup_test.go:528`)、`TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce`(`:565`)、`TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites`(`:600`)、`TestFollowupStore_ReportsAnUnreadableRecordAndAnUnavailableLock`(`:649`) |
| skill の文 | `internal/org/skill_followup_test.go` の 4 本(`run-test.sh` の中で通った) |

## Regression checks

lead が挙げた 4 か所(ready の判定、lock、status からの gh、M1 の同じ head の除外)に、ready の判定を status 側でも外す変異、D5 の時刻の比較を外す変異 2 個、Done だけ lock を外す変異を足し、8 個を確かめた。worktree のファイルは変えていない。変異ごとに 1 ファイルを差し替えた写しを scratch に作り、`go test -overlay` で入れた。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じで、どの回にも `build failed` はない。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| m1 merge-order の ready の判定を外す(`internal/org/mergeorder_followup.go:295` `case after != nil:` に `&& false`) | 落ちた | org 7 本(サブテスト 6 件)、cli 2 本(`TestOrgMergeOrder_FollowupSectionFollowsTheNotes` は f1 が waiting になる) |
| m2 status の ready の判定を外す(`internal/cli/status_followup.go:161` `merged` に `&& false`) | 落ちた | cli 2 本(`TestStatusCmd_FollowupsReadyFromTheLedgersMerges`、`_FollowupsRunNoGH`) |
| m3 lock を外す(`internal/org/followup.go:289` `withLock` の中身を `return fn()` に) | 落ちた | org 4 本。lock を見て落ちたのは `TestFollowupAdd_ConcurrentCallsGetDistinctIDs`(`ID f2 was handed out twice`)、`TestFollowupLock_IsSeparate…`(`Add returned … while followups.lock was held`)、`…/followups.lock_is_a_directory`。ほかの org 1 本と cli 3 本は、lock が state dir を作らなくなった副作用で落ちた |
| m3b Done だけ lock を外す(`internal/org/followup.go:189`) | 落ちた(20 回中 7 回) | `-count=20` で `TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce` だけが 7 回落ちた(`5 Dones reported changed, want exactly 1` など)。1 回の実行では見逃しうる(Test gaps の 1) |
| m4 status から gh を呼ぶ(`statusFollowup` の先頭に `exec.Command("gh", "pr", "list", …).Run()`) | 落ちた | cli 1 本(`TestStatusCmd_FollowupsRunNoGH`: `status ran gh: ["pr list --head feat/merged" …]`) |
| m5 M1 の同じ head の除外を外す(`internal/org/followup.go:498` `mergedBefore[head]` に `&& false`) | 落ちた | org `TestFollowupRecordedMerge` のサブテスト 3 件、cli 1 本(`_FollowupsReadyFromTheLedgersMerges` の f3 feat/old が ready になる) |
| m6 status の D5 の時刻の比較を外す(`internal/org/followup.go:489` に `\|\| true`) | 落ちた | org `TestFollowupRecordedMerge` のサブテスト 7 件、cli 1 本 |
| m7 merge-order の D5 の時刻の比較を外す(`internal/org/mergeorder_followup.go:344` に `&& false`) | 落ちた | org `StateComesFrom…` のサブテスト 4 件(AC8 の行を含む)、cli 2 本(f4 feat/old が ready になる) |

## Test gaps

1. Done の lock は確率でしか固定されていない。Done だけ lock を外す変異(m3b)は、`TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce`(`internal/org/followup_test.go:565`)が 20 回中 7 回落としただけだった。`TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites`(`:600`)は、`followups.lock` を持っている間に Add が返らないことしか見ていない。同じ形の Done の行(lock を持っている間は Done が返らず、離したあとに changed=true で返る)を足せば閉じられる。Add と Done の両方から lock を外す変異(m3)は、Add の行で決定的に落ちる
2. 書き込みの失敗の道が打たれていない: `Add` の appendLocked のエラーの return(`internal/org/followup.go:163`)と、appendLocked の marshal・open・write・close のエラー(`:324`、`:329`、`:332`、`:336`)。「followups.jsonl is a directory」の行(`internal/org/followup_test.go:650`)は読み出しで先に止まるので、ここには届かない
3. `recordedMergeHead` の head= のない Details(`internal/org/followup.go:554`)を打つ行がない。その event は M1 の除外の対象にならず、merge として数えられる。ralph の書き手(`recordMerge`、`cleanupDetails`)は必ず head= を書くので、手で直した manifest でだけ起きる
4. origin の確かめ(`git remote get-url origin`)の途中の中断(`internal/org/mergeorder_followup.go:201-203`)は打たれていない。`TestMergeOrderFollowup_InterruptedLookupIsNoError`(`internal/org/mergeorder_followup_test.go:457`)は gh の一覧の途中の中断だけを見ている
5. CLI の `followup add` で検証のあとに store が失敗する道(`internal/cli/org_followup.go:103`)と、`followup list` の読み出しの失敗(`:143`)は CLI の層では打たれていない。store の層では `TestFollowupStore_ReportsAnUnreadableRecordAndAnUnavailableLock` が見ている
6. status の CLI の行に、同じ after-branch を待つ follow-up が 2 つある場合と、追加より前の `pr_merged` のあとに別の head の `worktree_cleaned` がある場合がない(`seedStatusFollowups`、`internal/cli/status_followup_test.go:377` は 5 件とも別のブランチ)。判定は follow-up ごとに `org.FollowupRecordedMerge` が出し、後者はその単体の行(`internal/org/followup_recorded_merge_test.go` の「another head」)が見ているので、影響は小さい
7. 実物の gh と GitHub は使っていない。merge-order は偽の gh(`MergeOrderExec` の差し替えと、PATH の偽の gh)で見ている。status の CLI のテストは `pr_merged` と `worktree_cleaned` の Details を同じ形のヘルパーで組む(`internal/cli/status_followup_test.go:44`、`:52`)。org の単体のテストは本物の書き手(`recordMerge`、`cleanupDetails`)を使う

どれも計画の判定(ready・waiting・closed・unknown、D5 の時刻、M1 の除外、lock の中の読み直し、status が gh を呼ばないこと)を変えない。それらの判定は、上の変異で落ちることを確かめた。

## Verdict

- Verdict(cycle 1、HEAD a4cab358): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、絞った 2 回、`go test -race -count=3 ./internal/org/ -run 'Followup'`(135 件、DATA RACE 0)、変異 8 個はすべて落ちた(m3b は 20 回中 7 回)
- Fail: 0
- Blocked: 0
