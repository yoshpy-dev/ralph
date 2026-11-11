# Test report: watch-alert-dedupe

- Date: 2026-11-12(run-test.sh は 07:43:18 から 08:10:21 JST)
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(`## Test plan`、AC1〜AC15。digest `e5a18ef9477a`)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff 028a8f6d...HEAD`(HEAD `b8058948`、20 ファイル)。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。worktree の production code とテストには手を入れていない。変異と probe は `git archive HEAD` で作った scratchpad の写しに当てた
- Evidence: `docs/evidence/test-2026-11-12-watch-alert-dedupe.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、meta、新しいテストの PASS 行、`-race` の回、coverage、変異の harness・定義・結果、生き残った変異のパッケージ全体での打ち直し、gap の probe、修正前の件数の probe を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-11-224319.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,623 秒。1 分の load は開始時 9.56、終了時 10.89 |
| └ Go: `internal/org` | 上位 1,627 本 | ok | 0 | — | 910.6 秒 |
| └ Go: `internal/cli` | 上位 1,079 本 | ok | 0 | — | 673.0 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`、`cmd/ralph` | — | ok(`(cached)` は無い) | 0 | — | 27.7、7.4、3.4、4.6、2.5、5.4、2.2、0.7 秒 |
| 新しいテスト 43 本と期待を変えた 2 本(`-v -count=1`、org と cli) | org 上位 42 本(subtest 込み 123)、cli 3 本 | 126 | 0 | 0 | 16 秒、7 秒 |
| 同じテストを `-race -count=3` | org 369 行、cli 9 行 | 378 | 0 | 0 | 86 秒。`DATA RACE` の行は 0 |
| coverage の回(`-run 'Watch\|Inbox\|Scope\|OwnRecord\|Porcelain\|Escalat\|Deadman\|Stall'`、`-coverprofile`) | — | ok | 0 | — | 59 秒 |
| 変異 61 個(下の節) | 61 | 57 個は既存のテストで死んだ | — | — | 1 個 5〜41 秒 |
| 生き残った 4 個を同時に当てて `internal/org` と `internal/cli` 全体 | — | ok(4 個とも生き残った) | 0 | — | org 692.5 秒、cli 492.5 秒 |

上位の本数は `go test -list '.*'` で数えた。この diff が足したテストは org の 40 本(`inbox_test.go` 5、`watch_test.go` 13、`watch_scope_test.go` 22)と cli の 3 本(`org_inbox_test.go` 2、`org_watch_test.go` 1)で、消したテストは無い。

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(開始・終了・rc を meta に書く)を Bash tool の `run_in_background` で起動した。手番は終えず、`caffeinate -w <pid>` を perl の `alarm` で 560〜570 秒に区切った前景の待ちと meta の読み取りをくり返した。変異の harness も `run_in_background` で 1 個ずつ直列に走らせ、run-test.sh の shell の段と Go の段に重ねた。変異を 1 個打ち終えるたびに読んだ 1 分の load は最大 35.30 だった。run-test.sh は落ちず、打ち直しが要ったテストも無かった。

Skipped の 2 行は前の cycle の report と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-v` の回で PASS した。変異の名前は下の「変異」の節のもの。

| AC | テスト(ファイル) | 変異で確かめたこと |
| --- | --- | --- |
| AC1 件は 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go`。5 回立って件 1、`recurred` 4、途中で ack しても続く)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestInboxAppendRecurred_RecordsOnOpenAndAckedALERTItems`(`inbox_test.go`)、`TestOrgInbox_ShowsTheRecurrences`、`TestOrgInbox_ShowsOneRecurrenceAsOneTime`(`org_inbox_test.go`) | M01、M08、M38、M39、M42〜M45、M59 は死んだ |
| AC2 resolved のあとは新しい件 | 上の flapping のテストの後半、`TestInboxAppendRecurred_RefusesAndWritesNothing`、`TestWatch_Recurrence_ARecurrenceThatIsNotRecordedSendsANewAlert`、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly` | M02(件の検索で resolved を飛ばさない)は死んだ。M03(`AppendRecurred` も resolved を断らない、両方を外す)も死んだ |
| AC3 続きのあいだは leader に ALERT を送らない | flapping のテスト(ALERT 1 通、watcher の起動 1 回) | M01 の下では ALERT が 6 通、watcher の起動が 6 回になった(修正後は resolve の前後で 1 通ずつ、計 2 通) |
| AC4 deadman の前の再発は pending を増やさず、件に回数が残る | `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`、期待を変えた `TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires` | M04(pending の回数を足さない)、M05、M07(escalate のときに回数を移さない)は死んだ |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | `TestWatch_Recurrence_ANewWatchFindsTheOpenItemOfTheKeyInTheInbox`、`TestWatch_Recurrence_AnEscalationWhoseKeyHasAnItemThatIsNotResolvedIsARecurrenceOfIt` | M06(escalate の側の検索を止める)、M08(立ち上がりの側の検索を止める)、M09、M10、M52、M53 は死んだ |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たない | `TestWatch_LeaderStall_IsNotRaisedWhileItsOrgHasAnItemItsLeaderRaised`(3 種類、BLOCKED は acked、resolve のあとは立つ)、`TestWatch_LeaderStall_IsRaisedWhenItsOrgHasNoItemItsLeaderRaisedOpen` | M11〜M15(抑止を外す、resolved も数える、RESULT を外す、QUESTION を外す、ほかの org の件も数える)は死んだ |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば立つ | `TestWatch_LeaderStall_IsNotRaisedWhileAnotherSeatIsWorking`、`_IsRaisedWhenNoOtherSeatIsWorking`、`_IsRaisedWhenANewSeatsFirstReplySaysWorking` | M16〜M19(固まった `working` も数える、最初の返事も数える、`working` 以外も数える、leader を先に評価する)は死んだ |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | `TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing`、`_TheOrgsOwnNewPlanAndReportsRaiseNothing`(`watch_scope_test.go`、実際の git)、`TestScopePathInside_FollowsTheReservationRules`、`TestParsePorcelainZ_EntriesInOrderWithBothSidesOfARename` | M20、M27、M33、M46、M60 は死んだ |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | `TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths`(7 つの場合、実際の git)、`_OnlyANewPathOrACommitOutsideRaisesScopeChange`、`_WithNoOriginRefACommitOutsideRaisesScopeChange` | M21〜M24(rename の片側を落とす、`--no-renames` を外す)、M31、M37、M51、M61 は死んだ |
| AC10 予約のない org は今までどおり | `TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus` と既存の scope_change のテスト | 予約のない org の道を変える変異は当てていない。その比較のコードはこの diff で変わっていない(`realGitStatus` が `runGitBounded` を使うようになっただけ) |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | `TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite` | M26、M27 は死んだ |
| AC15 rebase で入った upstream の commit では立たない | `TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing`(rebase と merge)、`_ARebasedOwnCommitOutsideRaisesScopeChange` | M25(`--not <ref>` を効かなくする)は死んだ |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | `TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange`(8 つの場合)、`TestOwnRecordCandidate_PlansAndReportsButNotTheirTemplates` | M26、M28b、M29 は死んだ。M30(merge-base が読めないときに候補をすべて除く)は生き残った(Test gaps 1) |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | `TestWatch_StatusFile_WrittenBeforeRecurrencesLoadsAndItsPendingAlertTakesARecurrence`、`TestWatch_StatusFile_APendingAlertWithLastRecurredTSLoadsAndDropsIt`、`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads`、`TestInboxRead_LinesWithoutWokenFoldAsBefore` | M32、M42 は死んだ |
| AC13 help と文書 | `TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope`(help の文を固定する) | 文書の側は /verify と /sync-docs の範囲で、テストは無い |

計画の L1(commit の一覧に失敗したら前の HEAD を据え置く)は `TestWatch_ReservedScope_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle`、`_AFailedCommitListingAsksWhetherGitHasThePreviousHead`、`_APreviousHeadGitDoesNotHaveIsSkippedQuietly` が固定する(M34、M35 は死んだ)。git を打つ回数は `_RunsOnlyTheGitCallsItNeeds` が固定する(M23〜M25、M27、M47、M49 で赤になった)。

Verify plan の「再発を模したテストの件数」: flapping のテストで、M01 と M06 を同時に当てる(修正前の 2 つの道を両方外す)と、5 回立って受信箱の ALERT の件が 5 つ(`e1`〜`e5`)になり、テストは「want one ALERT item」で落ちた。修正後は件 1 つ、`recurred` 4 つ、leader への ALERT 1 通である。

### 期待を変えた既存のテスト 2 本

`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires` と `TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess` は、pending のあいだの再発で 2 通目の ALERT を期待していた。D1 と AC3 でこれは再発の記録に変わるので、S2 で期待を「pending のあいだは ALERT 1 通、pending の回数 1、leader が動いて pending が消えたあとの stall で 2 通目」に直した(計画の Progress の S2 に記録がある)。diff を読むと、2 通目の ALERT と Join の再試行を見る assert は残り、pending の数と回数を見る assert が足されている。assert を緩めた変更ではない。

## 変異

`mutate.sh` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることと元と新が違うことを確かめてから置き換え、`go test <パッケージ> -run <filter> -count=1 -timeout 20m` を打ち、写しの元のファイルに戻して `cmp` で確かめた。filter は変異ごとに、その関数を通るテストの名前の前半(`TestWatch_Recurrence`、`TestWatch_LeaderStall`、`TestWatch_ReservedScope` など)にした。変異を当てる前に、filter をすべて合わせた回を変異なしの写しで打ち、ok だった(40 秒)。M28 は `existing` が使われなくなって build が通らなかったので、同じ意味の M28b に替えた。変異を当てているあいだ、worktree の `git status --porcelain` は空だった。

| 組 | 変異の数 | 結果 |
| --- | --- | --- |
| 受信箱(`inbox.go`): 回数の畳み込み、最後の時刻、JSON の欄の名前 2 つ、`AppendRecurred` の断り(ALERT でない件、空の alert_id、count、note) | 8 | すべて死んだ |
| 続きの扱い(`watch_recurrence.go`、`watch.go` の `raiseOrClear` と `escalateToInbox`。M03 は `inbox.go` の resolved の断りも同時に外す) | 12 | すべて死んだ |
| leader の stall(`watch.go` の `orgHasOpenLeaderItem`、`seatBusy`、`leaderIsWaiting`、評価の順) | 10 | 9 個は死んだ。M56(受信箱を周期に 1 回だけ読む印を無視する)が生き残った |
| 予約のある org の scope_change(`watch_scope.go` と `watch.go` の分岐。porcelain の解析、予約の判定、commit の一覧、origin の見つけ方、除く規則、立てる条件、HEAD の据え置き、本文と note のパス) | 28 | 25 個は死んだ。M30、M48、M50 が生き残った |
| 表示(`internal/cli/org_inbox.go`) | 3 | すべて死んだ |

### 生き残った 4 個

`-run` で絞った回で生き残った M30、M48、M50、M56 は、4 個を同時に当てて `go test ./internal/org/ -count=1 -timeout 30m`(ok、692.5 秒)と `go test ./internal/cli/ -count=1 -timeout 30m`(ok、492.5 秒)で打ち直した。4 個とも別の関数を変えるので、互いの赤を隠すことはない。パッケージ全体でも生き残った。

gap の probe: M30 と M50 には、HEAD で通り、その変異だけで赤になるテストを 1 本ずつ書き、写しに throwaway の `zz_gap_test.go` と `zz_gap2_test.go` を置いて確かめた(コミットしていない。全文は evidence にある)。

- M30 の probe: `refs/remotes/origin/main` を、HEAD と共通の祖先を持たない root commit(空の tree の `commit-tree`)に向け、予約の外に新しい計画 `docs/plans/active/2026-11-12-feat.md` を書く。HEAD では scope_change が 1 件立ち、そのパスを名指す。M30 の下では 0 件になった。origin を分岐点に置いた対照の行は、どちらでも 0 件で通った
- M50 の probe: 1 周期だけ `rev-parse --verify -q HEAD` を失敗させ、そのあいだに予約の外を commit する。HEAD では次の周期で scope_change が 1 件立つ。M50 の下では `ScopeHead` が空になり、0 件になった

## Coverage

- Statement: coverage の回は filter を付けたので、パッケージ全体の数字(39.9%)は意味を持たない。この diff が足したり変えたりした production の行(`git diff -U0 028a8f6d..HEAD` の行と profile の block を突き合わせた)にかかる block は 298 statement 中 288(96.6%)
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: `watch_recurrence.go` は `escalateAsRecurrence` の 85.7% を除いて 100%。`watch_scope.go` は `scopeRecordsAtBase` 76.9%、`dropOwnRecords` 92.9%、`evaluateReservedScope` 95.0%、`runGitBounded` 92.3% で、ほかは 100%。`watch.go` は `leaderIsWaiting`、`orgHasOpenLeaderItem`、`seatBusy`、`evaluateSeat`、`raiseOrClear` が 100%、`evaluateCycle` 93.9%、`escalateToInbox` 93.1%。`inbox.go` の `AppendRecurred` と `apply` は 100%
- Notes: 打たれていない 10 statement は次のとおりで、最後の `realGitStatus` を除けば失敗の道である。`watch.go:1858-1860`(新しい件を記録したあと pending の回数の `recurred` が書けなかったときの stderr の行)、`watch_recurrence.go:139-141`(`escalateAsRecurrence` の追記が resolved 以外の理由で失敗したときの行)、`watch_scope.go:308-317`(merge-base と ls-tree の失敗、M30 の道)、`:348-349`(同じ失敗を受けて何も除かない枝)、`:401-402`(`gitRun` が nil のときの既定)、`:405-406`(porcelain が読めない周期)、`:51-52`(`ErrWaitDelay` を成功とみなす枝)、`watch.go:424-425`(`realGitStatus`)。最後の 1 つは filter の外の `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout`(`watch_all_test.go`)が呼ぶので、run-test.sh の回では打たれている。ほかにも filter の外のテストが打つ行はありうる

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、新しいテストの `-v` の回、`-race` の回、coverage の回で、落ちたテストも、打ち直しが要ったテストも無かった。flaky と分類したものも無い | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #307: 同じ条件の ALERT が立つたびに受信箱の件が 1 つ増える | 直っている | flapping のテスト。修正前の 2 つの道を外す変異(M01 + M06)では 5 回で 5 件、修正後は 1 件 |
| issue #307: 待っているだけの leader の stall | 直っている | AC6 と AC7 のテスト |
| issue #307: 予約の中だけの変更の scope_change | 直っている | AC8 のテスト。AC15 で rebase の誤報(S4b で直したもの)も固定されている |
| 既存の deadman、liveness、watch の立て直し(同じ alert_id の件を二重に作らない)、予約のない org の scope_change | 変わっていない | `internal/org` と `internal/cli` 全体が ok。期待を変えた既存のテストは上の 2 本だけ |
| 古い watch-status と受信箱 | 読める | AC12 のテスト |

## Test gaps

1. M30(LOW): origin の既定のブランチはあるが merge-base が読めないとき(共通の祖先が無い、git が期限で切られた)に、何も除かない枝が固定されていない。変異の下では、自分の記録の候補をすべて除き、予約の外の変更を知らせない側に倒れる。計画の D5 と `dropOwnRecords` の doc は「base が読めなければ何も除かない」と書いている。上の probe の形の 1 本で死ぬ。`scopeRecordsAtBase` の失敗の 3 か所も打たれていない
2. M50(LOW): HEAD が読めない周期に前の HEAD を据え置く条件(`head != ""`)が固定されていない。変異の下では、その周期に `ScopeHead` が空になり、そのあいだの予約の外の commit が知らされない。`evaluateReservedScope` の doc は「HEAD が読めなければ前の HEAD を据え置く」と書いている。既存の `_AWorktreeWithNoCommitComparesItsChanges` は前の HEAD が空の場合しか通らない。上の probe の形の 1 本で死ぬ
3. M48(INFO): `scopeDefaultBranchRef` が `symbolic-ref` の答えを `refs/remotes/origin/` の下のものに限る検査が固定されていない。テストの repo では origin/HEAD がいつも `refs/remotes/origin/trunk` を指すので、検査を外しても答えは同じになる。手で `refs/heads/...` を指させた origin/HEAD か、偽の git の 1 行で固定できる
4. M56(INFO、等価): `leaderIsWaiting` の「受信箱を周期に 1 回だけ読む」印を無視しても、leader は org に 1 席なので 1 周期に 1 回しか呼ばれず、違いは外から見えない
5. 打たれていない失敗の道(Coverage の Notes): 新しい件を記録したあと pending の回数を移せなかったときの行(`watch.go:1858`)は、計画が既知の穴として記録した「件を作った直後に止まると回数が残らない」の隣の道で、stderr の 1 行だけを書く。害はその回数が件に残らないことだけである
6. Test plan の Integration の項(`watch_all_test.go` の流れで件数を照らす)は、その形のテストになっていない。新しいテストは続きと leader の stall を `evaluateCycle` と `escalateToInbox` から直に打ち、`RunWatchAll` を通すのは scope の GitRun の配線(`TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus`)だけである。既存の `watch_all_test.go` は変わらずに通った。INFO
7. 実機の道: 新しい binary で機能の org を走らせて受信箱の件を数え直すことはしていない。再発の原因(herdr の `agent get` の文字列の揺れ)は計画の Assumptions のとおり推測のままで、テストは偽の herdr と一時 repo の実際の git だけで確かめている

## Verdict

- Pass: 6(6 回の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通り、`(cached)` は無かった。新しいテスト 43 本と期待を変えた 2 本の `-v` の回、その `-race -count=3` の回(`DATA RACE` 0)、coverage の回、生き残った変異 4 個をパッケージ全体で打ち直した org と cli の回も 0 failed。AC13 を除く 14 個の AC と AC13 の help の文には、どれも PASS するテストがあり、名前は上の表にある。変異 61 個のうち 57 個は既存のテストが殺し、生き残った 4 個はどれも LOW か INFO の穴(Test gaps 1〜4)で、M30 と M50 はテストを 1 本ずつ足せば死ぬことを probe で確かめた。変えた production の行の coverage は 298 statement 中 288)
- Fail: 0
- Blocked: 0。Test gaps 1、2 のテストを足すか、tech-debt の行に足すかは orchestrator が決める
