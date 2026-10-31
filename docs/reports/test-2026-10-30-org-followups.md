# Test report: org-followups

- Date: 2026-11-01(cycle 1 と cycle 2 の実行日。ファイル名は計画の日付)
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(`## Test plan` と AC1〜AC8、Progress の S1〜S5 と F1〜F6)。Refs #270
- Tester: tester サブエージェント(`/test`)。cycle 2 の節はパイプラインの cycle 2(`cycle-count.json` は 2)、cycle 1 の節は cycle 1
- 構成: 先頭が cycle 2 の節、次がこのファイルの判定の `## Verdict`(cycle 2 のもの)、最後が cycle 1 の節。cycle 1 の節は中身を変えず、見出しを 1 段下げ、判定の見出しを「Verdict(cycle 1)」にし、冒頭の Scope と Evidence の行をその節に移した。ファイルの `## Verdict` は cycle 2 の 1 つだけ
- Evidence: `docs/evidence/test-2026-10-30-org-followups.md`(要点。commit する。先頭が cycle 2、続いて cycle 1)、`docs/evidence/test-2026-10-30-org-followups.log`(gitignore。cycle 2 の分は末尾の `cycle 2 (HEAD 543d29db)` の区切りのあと)。wrapper 自身の写しは cycle 2 が `docs/evidence/verify-2026-10-31-182544.log`(同じく gitignore)

## Cycle 2(HEAD 543d29db)

### Scope(cycle 2)

- ブランチ全体は `git diff origin/main...HEAD`(merge-base 1cc84ffa、HEAD 543d29db)。cycle 1(HEAD a4cab358)のあとのコードの変更は F5 25c9fa77 と F6 12d28dae だけで、ほかは文書
  - F5: follow-up だけの経路の `gh_available`。`lookupFollowupMerges` が gh を見つけたかを返す(`internal/org/mergeorder_followup.go:173`。gh がなければ `:202` で false に戻し、gh が答えれば `:205` で true)。`mergeOrderFollowups` はそれを `GHAvailable` に OR で足す(`:134-136`)。足したテストは `TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch`(`internal/org/mergeorder_followup_test.go:661`、6 行)、`TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable`(`:725`)、`TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable`(`internal/cli/org_merge_order_followup_test.go:282`)
  - F6: コメントの追記と、merge-order の help の折り返し(`internal/cli/org_merge_order.go:102-106`)。文言は変わらない
- `GHAvailable` は `ralph org cleanup` も読む(`internal/cli/org_cleanup.go:256`)。ただし `Followups: true` を渡すのは merge-order だけ(`internal/cli/org_merge_order.go:191`)なので、F5 は cleanup の判定を変えない
- `./scripts/run-test.sh` は changed scope のまま打った(言語は golang)。test mode は `tests/test-*.sh` を全部打つので、shell の 45 本も走った。テストの関数は cycle 1 から 3 本増え(org 2 本、cli 1 本)、この diff 全体では 65 本増えた。消えたものはない

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok | 0 | 1(cycle 1 と同じ `test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行) | 1,578 秒(03:25:44〜03:52:02 JST)。load は開始 12.20 / 11.96 / 15.96、終了 6.12 / 16.33 / 25.28 |
| └ Go: `internal/org` | 1,398 本(トップレベル、`go test -list`) | ok | 0 | 0 | 599.6 秒 |
| └ Go: `internal/cli` | 954 本(同) | ok | 0 | 0 | 456.3 秒 |
| └ Go: `internal/config` | — | ok | 0 | 0 | 10.2 秒 |
| └ Go: `cmd/ralph`、`internal/{gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok `(cached)` | 0 | 0 | — |
| 上の 7 パッケージを `go test -count=1` で打ち直し | — | 7 パッケージ ok | 0 | 0 | 4 秒(1 パッケージ 1.0〜3.5 秒) |
| `go test -count=1 -run 'Followup\|MergeOrder' ./internal/org/`(変異の基準) | 102 本 | 102 | 0 | 0 | 7 秒 |
| `go test -count=1 -run 'Status\|OrgMergeOrder\|OrgFollowup' ./internal/cli/`(変異の基準) | 117 本 | 117 | 0 | 0 | 119 秒 |
| F5 に関わるテストを `-v` で(org 5 本、cli 3 本) | org 5 本(サブテスト 9 件)、cli 3 本 | すべて PASS | 0 | 0 | 2 秒、8 秒 |
| `go test -race -count=3 ./internal/org/ -run 'Followup\|MergeOrder'` | 102 本 × 3 | 306 | 0 | 0 | 58 秒。`WARNING: DATA RACE` は 0 |
| 変異 6 個(`go test -overlay`、上の 2 つの絞り方) | 6 | 4 個が落ちた。手順書の 2 個(mA、mB)はどちらも落ちた。2 個(mE、mF)は残った | — | — | org 13〜19 秒、cli 79〜134 秒 |
| coverage の回(org の絞り方に `-coverprofile`) | — | ok | 0 | 0 | 22 秒 |

`(cached)` の 7 パッケージのうち `cmd/ralph` と `internal/insights` は `internal/org` に依存する(`go list -deps -test`)。cache は同じ入力で前に通った結果だが、それに頼らないよう `-count=1` で打ち直し、7 パッケージとも通った。

起動の仕方は cycle 1 と同じ。`run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。`run-test.sh` が動いている間に、絞った回、F5 のテスト、変異、`-race` の回、coverage の回を前景で 1 つずつ打った。03:40 ごろに別のプロセス(iOS simulator と swift-frontend)で 1 分の load が 132.83 まで上がったが、時間切れも既知の flaky なテストの失敗もなく、打ち直しはしていない。shell の SKIP の行は cycle 1 と同じ 1 行だけだった。

### F5 の振る舞いとテスト(cycle 2)

| 場合 | テスト | 確かめた値 |
| --- | --- | --- |
| 機能のブランチがなく、gh が見つかって答える | `GHAvailableWithNoFeatureBranch/gh_answers`、cli `TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable` | `GHAvailable` true、f1 ready。cli の `--json` の `gh_available` は `true` で、gh の呼び出しは follow-up の検索の 1 回だけ |
| gh がすべてのブランチで失敗する | `…/gh_fails_for_the_branch`(ブランチは 1 本) | true、f1 unknown |
| gh がない | `…/gh_is_missing`、`TestMergeOrderFollowup_GHMissingMakesEveryFollowupUnknown`(`:311`)、cli `TestOrgMergeOrder_FollowupWithoutGHIsUnknown`(`internal/cli/org_merge_order_followup_test.go:260`) | false、すべて unknown |
| 1 本目に答えたあと gh が消える | `…/gh_goes_missing_after_an_answer` | false、f1 と f2 は unknown |
| open の follow-up がない | `…/no_record`、`…/only_a_done_follow-up` | false、gh の呼び出しは 0 |
| 機能のブランチがあり、follow-up の検索で gh が見つからない | `TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable` | entries の検索が出した true が残る。f1 は unknown で、DETAIL は「gh was not found, so no PR was read」 |
| 機能のブランチがあり、follow-up のある回とない回を比べる | `TestMergeOrderFollowup_EntriesAndTheirLookupsStayAsTheyWere`(`:271`) | `GHAvailable` も entries も notes も同じ |

どれも `run-test.sh` の中と、`-v` の回の両方で通った。

### Coverage(cycle 2)

- Statement(org の絞った回の profile。パッケージ全体は 13.2%): `lookupFollowupMerges` は 94.6% から 95.0% に上がった。`mergeOrderFollowups` は 100%、`MergeOrder` は 96.9% のまま。`internal/org/followup.go` の `Add` 94.4%、`appendLocked` 68.8%、`recordedMergeHead` 80.0% は cycle 1 と同じ
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり。cli は F5 と F6 で本体のコードが help の折り返ししか変わっていないので、cycle 1 の数字のまま(打ち直していない)
- Notes: follow-up のファイルで打たれていないブロックは cycle 1 と同じ 7 か所。`internal/org/followup.go:163`、`:324`、`:329`、`:332`、`:336`、`:554`、`internal/org/mergeorder_followup.go:213-214`(F5 で行がずれた。cycle 1 の `:202-203` と同じ箇所)

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

`run-test.sh`、打ち直しの 7 パッケージ、絞った 2 回、F5 のテスト、`-race` の回、coverage の回に失敗はなかった。

### Regression checks(cycle 2)

worktree のファイルは変えていない。cycle 1 と同じく、`internal/org/mergeorder_followup.go` を差し替えた写しを scratch に作り、`go test -overlay` で入れた(置換は「元の文字列がちょうど 1 回現れる」ことを確かめる perl の helper)。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じで、どの回にも `build failed` はない。mA と mB は手順書の 2 個、mC〜mF は F5 のほかの行を見るために足した。

| ID | 変更 | org(`Followup\|MergeOrder`) | cli(`Status\|OrgMergeOrder\|OrgFollowup`) |
| --- | --- | --- | --- |
| mA | `:136` の OR を代入に: `res.GHAvailable = res.GHAvailable \|\| ghFound` → `res.GHAvailable = ghFound` | 落ちた(13 秒)。`TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable`: `GHAvailable = false with entries [feat/a …]; want true and feat/a` | 落ちた(79 秒)。`TestOrgMergeOrder_AllListsMergedAndClosedPRsTheDefaultLeavesOut`(`internal/cli/org_merge_order_test.go:625`): `--all --json: gh_available false`。CLI は follow-up の記録がない台帳でも `Followups: true` で呼ぶので、entries の true が消える |
| mB | gh がないときに false へ戻す `:202` の `ghFound = false` を消す | 落ちた(13 秒)。`GHAvailableWithNoFeatureBranch/gh_goes_missing_after_an_answer`: `GHAvailable = true with entries []; want false` | 残った(127 秒)。cli には途中で gh が消える行がない |
| mC | `:205` の `ghFound = true` を、gh の失敗の `continue` のあと(PR が読めたときだけ)に移す | 落ちた(13 秒)。`…/gh_fails_for_the_branch`: `GHAvailable = false …; want true` | 残った(101 秒) |
| mD | `:136` の OR の右を `(ghFound && false)` に | 落ちた(13 秒)。`…/gh_answers` と `…/gh_fails_for_the_branch`: `want true` | 落ちた(134 秒)。`TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable`: `gh_available = false with entries []; want true` |
| mE | PR が origin の repository のものでないとき(`case why != "":`)に `ghFound = false` | 残った(18 秒) | 残った(93 秒) |
| mF | 1 本目の gh の途中で中断したとき(`case ctx.Err() != nil:`)に `ghFound = true` | 残った(19 秒) | 残った(88 秒) |

手順書の 2 個は、どちらも新しいテストで落ちた。mA は org の単体の行と、cycle 1 からある cli の行の両方で落ちる。mB は org の単体の行だけが落とす。mE と mF は Test gaps の 8 と 9 に書いた。

### Test gaps(cycle 2)

cycle 1 の 1〜7(下の cycle 1 の節)は変わらない。F5 のテストは `GHAvailable` と follow-up の状態だけを見るので、どれも閉じていない。4 の場所は F5 で行がずれ、今は `internal/org/mergeorder_followup.go:213-214`。cycle 2 で次の 3 つが加わった。

8. 中断のときの `GHAvailable` が固定されていない(mF が残った)。`TestMergeOrderFollowup_InterruptedLookupIsNoError`(`internal/org/mergeorder_followup_test.go:457`)は、feat/a に gh が答えたあとで feat/b の検索を止めるので、`GHAvailable` はどちらにしても true になり、しかもそのテストは `GHAvailable` を見ていない。`lookupFollowupMerges` のコメント(`:170-172`)の定義では、1 本目で中断すれば gh はまだ答えていないので false になる。`GHAvailableWithNoFeatureBranch` に「1 本目の gh の途中で中断する」行(期待は false)を足せば閉じられる
9. origin の repository のものでない PR が返ったときの `GHAvailable` が固定されていない(mE が残った)。gh は答えているので、コメントの定義では true になる。同じ表に「origin でない repository の PR」の行(期待は true)を足せば閉じられる
10. cli の層には、gh が失敗する場合と途中で消える場合の `gh_available` の行がない。mB と mC は org の単体の行だけが落とす。判定は org の層にあるので、影響は小さい

8〜10 で変わりうるのは merge-order の `--json` の `gh_available` の端の値だけで、follow-up の状態(ready・waiting・closed・unknown)と DETAIL は変わらない。cleanup は follow-up を渡さないので、その判定にも影響しない。

## Verdict

- Verdict(cycle 2、HEAD 543d29db): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ。`(cached)` の 7 パッケージは `-count=1` で打ち直して ok)、絞った 2 回(org 102 本、cli 117 本)、F5 に関わるテスト、`go test -race -count=3 ./internal/org/ -run 'Followup|MergeOrder'`(306 件、DATA RACE 0)。手順書の変異 2 個(mA、mB)はどちらも落ちた
- Fail: 0
- Blocked: 0
- 残った変異: mE と mF(Test gaps の 8 と 9)。どちらも `gh_available` の端の値だけで、計画の判定は変えない

## Cycle 1(HEAD a4cab358)

- Scope(cycle 1): `git diff origin/main...HEAD`(merge-base 1cc84ffa、HEAD a4cab358。verify のあとの F4 a4cab358 で merge-order の follow-up の表に SOURCE の列が入った版)。Go は `internal/org`、`internal/cli`。ほかは skill と写し、README、AGENTS.md、仕様、tech-debt。`./scripts/run-test.sh` は changed scope のまま打った(言語は golang)。test mode は `tests/test-*.sh` を glob で全部打つので、shell の 45 本もすべて走った。テストの関数はこの diff で 62 本増え、消えたものはない
- Evidence(cycle 1): `docs/evidence/test-2026-10-30-org-followups.md`(要点。commit する)、`docs/evidence/test-2026-10-30-org-followups.log`(gitignore。`run-test.sh` の全出力、追加の回の出力、変異の差分と出力、coverage の関数の行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-163815.log`

### Test execution

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

### Coverage

- Statement(絞った回の profile。パッケージ全体の数字は絞ったので意味がない: org 13.1%、cli 25.1%):
  - `internal/org/followup.go`: 26 本中 23 本が 100%。`Add` 94.4%、`appendLocked` 68.8%、`recordedMergeHead` 80.0%
  - `internal/org/mergeorder_followup.go`: 8 本中 7 本が 100%。`lookupFollowupMerges` 94.6%
  - `internal/cli/status_followup.go`: 6 本すべて 100%
  - `internal/cli/org_followup.go`: 7 本中 5 本が 100%。`newOrgFollowupAddCmd`・`newOrgFollowupListCmd` 95.0%
  - 変更した既存の関数: `MergeOrder` 96.9%、`runStatus` 93.5%、`buildStatusOrgJSON` 85.7%、`printStatusEmpty`・`printStatusJSONAllOrgs`・`printStatusTableAllOrgs`・`printMergeOrder`・`printMergeOrderFollowups` 100%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes: 打たれていないブロックは、書き込みの失敗(`internal/org/followup.go:163`、`:324`、`:329`、`:332`、`:336`)、head= のない Details(`:554`)、origin の確かめの途中の中断(`internal/org/mergeorder_followup.go:202-203`)、CLI の add と list の store のエラー(`internal/cli/org_followup.go:103`、`:143`)だけ。Test gaps の 2〜5 に書いた

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

`run-test.sh`、絞った 2 回、`-race` の回、coverage の 2 回に失敗はなかった。

### Test plan とテストの対応

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

### Regression checks

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

### Test gaps

1. Done の lock は確率でしか固定されていない。Done だけ lock を外す変異(m3b)は、`TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce`(`internal/org/followup_test.go:565`)が 20 回中 7 回落としただけだった。`TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites`(`:600`)は、`followups.lock` を持っている間に Add が返らないことしか見ていない。同じ形の Done の行(lock を持っている間は Done が返らず、離したあとに changed=true で返る)を足せば閉じられる。Add と Done の両方から lock を外す変異(m3)は、Add の行で決定的に落ちる
2. 書き込みの失敗の道が打たれていない: `Add` の appendLocked のエラーの return(`internal/org/followup.go:163`)と、appendLocked の marshal・open・write・close のエラー(`:324`、`:329`、`:332`、`:336`)。「followups.jsonl is a directory」の行(`internal/org/followup_test.go:650`)は読み出しで先に止まるので、ここには届かない
3. `recordedMergeHead` の head= のない Details(`internal/org/followup.go:554`)を打つ行がない。その event は M1 の除外の対象にならず、merge として数えられる。ralph の書き手(`recordMerge`、`cleanupDetails`)は必ず head= を書くので、手で直した manifest でだけ起きる
4. origin の確かめ(`git remote get-url origin`)の途中の中断(`internal/org/mergeorder_followup.go:201-203`)は打たれていない。`TestMergeOrderFollowup_InterruptedLookupIsNoError`(`internal/org/mergeorder_followup_test.go:457`)は gh の一覧の途中の中断だけを見ている
5. CLI の `followup add` で検証のあとに store が失敗する道(`internal/cli/org_followup.go:103`)と、`followup list` の読み出しの失敗(`:143`)は CLI の層では打たれていない。store の層では `TestFollowupStore_ReportsAnUnreadableRecordAndAnUnavailableLock` が見ている
6. status の CLI の行に、同じ after-branch を待つ follow-up が 2 つある場合と、追加より前の `pr_merged` のあとに別の head の `worktree_cleaned` がある場合がない(`seedStatusFollowups`、`internal/cli/status_followup_test.go:377` は 5 件とも別のブランチ)。判定は follow-up ごとに `org.FollowupRecordedMerge` が出し、後者はその単体の行(`internal/org/followup_recorded_merge_test.go` の「another head」)が見ているので、影響は小さい
7. 実物の gh と GitHub は使っていない。merge-order は偽の gh(`MergeOrderExec` の差し替えと、PATH の偽の gh)で見ている。status の CLI のテストは `pr_merged` と `worktree_cleaned` の Details を同じ形のヘルパーで組む(`internal/cli/status_followup_test.go:44`、`:52`)。org の単体のテストは本物の書き手(`recordMerge`、`cleanupDetails`)を使う

どれも計画の判定(ready・waiting・closed・unknown、D5 の時刻、M1 の除外、lock の中の読み直し、status が gh を呼ばないこと)を変えない。それらの判定は、上の変異で落ちることを確かめた。

### Verdict(cycle 1)

- Verdict(cycle 1、HEAD a4cab358): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、絞った 2 回、`go test -race -count=3 ./internal/org/ -run 'Followup'`(135 件、DATA RACE 0)、変異 8 個はすべて落ちた(m3b は 20 回中 7 回)
- Fail: 0
- Blocked: 0
