# Test report: watch-alert-dedupe

- Date: 2026-11-12(run-test.sh は 12:46:19 から 13:08:32 JST)
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(`## Test plan`、AC1〜AC15)
- Tester: tester サブエージェント(`/test`、pipeline の run 4(cycle-count 3)。`cycle-count.json` は cycle 3、cap 3 のままで、insight event は `--cycle 4` で足した)
- Scope: `git diff 028a8f6d...HEAD`(HEAD `b305064a`)。run 3(HEAD `10a86e28`)のあとのコードの変更は S12 `14bc17a3` だけで、cycle 3 の cross-review の 1 件を直した。`evaluateCycle`(`watch.go`)が座席を評価する前に `reconcilePendingAlertsWithInbox`(`watch_recurrence.go`)を呼び、受信箱がすでに持つ pending(件の `alert_id`、または件の `recurred` の event の `alert_id` が同じもの)を `escalateAlert` に通して pending から消す。テストは `watch_test.go` に 4 本と helper の `pendingLivenessAlert` が増え、既存のテストの期待は変わっていない。ほかの commit は記録だけを変えた。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。worktree の production code とテストには手を入れていない。変異と probe は `go test -overlay` で当て、worktree のファイルは書き換えていない
- Evidence: `docs/evidence/test-2026-11-12-watch-alert-dedupe.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、meta と runner、打ち直し、`-v` と `-race` の回、coverage、変異の harness・定義・結果・失敗の文、probe の全文を足した。run 3 の evidence は `-cycle3.log` に名前を変えて残し、cycle 2 と cycle 1 のものは `-cycle2.log`、`-cycle1.log` のまま。wrapper 自身のログは `docs/evidence/verify-2026-11-12-034619.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,333 秒。1 分の load は開始時 8.99、終了直後 6.70 |
| └ Go: `internal/org` | 上位 1,637 本 | ok | 0 | — | 749.1 秒 |
| └ Go: `internal/cli` | 上位 1,079 本 | ok | 0 | — | 566.6 秒 |
| └ Go: `internal/config`、`cmd/ralph`、`internal/insights` | — | ok | 0 | — | 17.2、0.7、1.4 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | run-test.sh では `(cached)`。`-count=1` で打ち直して ok | 0 | — | 打ち直しは 1.8、1.2、0.5、0.5、1.4 秒 |
| この diff が足したテスト 53 本と期待を変えた 2 本(`-v -count=1`) | org 上位 52 本(subtest 込みの PASS 行 146)、cli 3 本 | 149 | 0 | 0 | 16.4 秒、0.9 秒 |
| `-race -count=3`(org は上の 52 本と、名前が `TestWatch`、`TestRunWatch`、`TestInbox` で始まるすべての合わせて 186 本。cli は上の 3 本) | org 上位 558 回(186 × 3、PASS 行 1,332)、cli 9 回 | 1,341 | 0 | 0 | 112.6 秒、2.1 秒。`DATA RACE` の行は 0 |
| coverage の回(`-run 'Watch\|Inbox\|Scope\|OwnRecord\|Porcelain\|Escalat\|Deadman\|Stall'`、`-coverprofile`) | — | ok | 0 | — | 45.7 秒 |
| 変異: S12 への 11 個と、修正前の 2 ファイルに戻した PRE(下の節) | 12 | 死んだ 11、生き残り 1(E07、等価) | — | — | 1 個 5〜62 秒 |
| 変異: 前の cycle の 43 個の打ち直し(`watch.go` と `watch_recurrence.go` の cycle 1 の 22 個と cycle 2 の 18 行、M30・M48・M50) | 43 | 死んだ 35、生き残り 8(run 3 と同じ 8 個) | — | — | 1 個 5〜49 秒 |
| probe(`go test -overlay`、null の pending、HEAD と E07 の下で) | 1 本(subtest 2 つ) | 下の節 | — | — | 1 回 1 秒 |

上位の本数は `go test -list '.*'` で数えた。org は run 3 の 1,633 本から S12 の 4 本だけ増え、cli は変わらない。この diff 全体では org に 50 本、cli に 3 本が増え、消えたテストは無い。期待を変えた base のテスト 2 本は run 3 までと同じ(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)。S12 はテストの期待をどれも変えておらず、assert も弱めていない。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(pid、start、end、rc を meta に書く)を Bash tool の `run_in_background` で起動し、`caffeinate -w <pid>` を perl の `alarm` で区切った前景の待ちで終わりを待った。`-v` の回、`-race` の回、変異、probe は run-test.sh の shell の段と Go の段に重ねて走らせ、coverage の回は run-test.sh が終わってから走らせた。重ねたあいだに読んだ 1 分の load は最大 25.96 だった。run-test.sh を含め、落ちたテストは 1 本も無かった。依頼にあった「落ちたら 1 回打ち直してから失敗と判断する」に当たる場面は無く、打ち直しはしていない。

run-test.sh で `(cached)` になったのは上の 5 パッケージで、`-count=1` で打ち直してすべて ok だった。run 3 で `(cached)` だった `cmd/ralph` と `internal/insights` はこの run では走ったが、同じく `-count=1` でも打ち直し、ok だった(0.4 秒、0.4 秒)。

Skipped の 2 行は run 3 までと同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-v` の回で PASS した。「変異」の列の M 番号は cycle 1 の定義(`-cycle1.log`)、C 番号は cycle 2 の定義(下の「cycle 2 の記録」の表)、D 番号は run 3 の定義(「run 3 の記録」の表)、E 番号は run 4 の定義(「変異」の節)である。

| AC | テスト(ファイル) | 変異で確かめたこと |
| --- | --- | --- |
| AC1 件は 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go`)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestInboxAppendRecurred_RecordsOnOpenAndAckedALERTItems`(`inbox_test.go`)、`TestOrgInbox_ShowsTheRecurrences`、`TestOrgInbox_ShowsOneRecurrenceAsOneTime`(`org_inbox_test.go`)、`TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew` | cycle 1 の M01、M08、M38、M39、M42〜M45、M59。M01、M08、C14〜C16 は run 4 の打ち直しでも死んだ |
| AC2 resolved のあとは新しい件 | flapping のテストの後半、`TestInboxAppendRecurred_RefusesAndWritesNothing`、`TestWatch_Recurrence_ARecurrenceThatIsNotRecordedSendsANewAlert`、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`、escalation の打ち直しのテストの resolved の行。S12 で `TestWatch_Recurrence_APendingAlertAStoppedWatchEscalatedTakesNoNewFiring` と `_APendingAlertOnARecurredEventIsDroppedBeforeTheSeats` の resolved の行(件を記録したあと状態ファイルを保存する前に watch が止まっても、resolved のあとの発火は新しい ALERT になる) | cycle 1 の M02、M03。C14。S12 の前の形(PRE)と E01、E03〜E05、E09、E10 で S12 のテストの resolved の行が落ちた(下の節) |
| AC3 続きのあいだは leader に ALERT を送らない | flapping のテスト(ALERT 1 通、watcher の起動 1 回)、`TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert`。S12 の 2 本の open の行(open の件の再発は ALERT を送らず件に `recurred` を足す)と `TestWatch_Recurrence_AnInboxThatCannotBeReadLeavesThePendingAlertsAsTheyAre` | cycle 1 の M01。C01、C03、C06、C08b は run 4 の打ち直しでも死んだ |
| AC4 deadman の前の再発は pending を増やさず、件に回数が残る | `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`、期待を変えた `TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、leader が動いたかのテストの「何もしなかった」行。S12 で `TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept`(受信箱が持たない pending は、同じ鍵の件があっても消さない) | cycle 1 の M04、M05、M07。C09〜C13 は run 4 の打ち直しでも死んだ。E02、E02k は `_IsKept` で死んだ。C24 は生き残った(Test gaps 3) |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | `TestWatch_Recurrence_ANewWatchFindsTheOpenItemOfTheKeyInTheInbox`、`TestWatch_Recurrence_AnEscalationWhoseKeyHasAnItemThatIsNotResolvedIsARecurrenceOfIt`、escalation の打ち直しのテスト。S12 の 2 本の open の行(止まった watch が残した pending を消したあとの発火が、open の件の `recurred` になる) | cycle 1 の M06、M08、M09、M10、M52、M53。C14〜C16、C23。PRE と E01 で S12 のテストの open の行が「recurred events = []」で落ちた |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たない | `TestWatch_LeaderStall_IsNotRaisedWhileItsOrgHasAnItemItsLeaderRaised`、`TestWatch_LeaderStall_IsRaisedWhenItsOrgHasNoItemItsLeaderRaisedOpen` | cycle 1 の M11〜M15 は run 4 の打ち直しでも死んだ |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば立つ | `TestWatch_LeaderStall_IsNotRaisedWhileAnotherSeatIsWorking`、`_IsRaisedWhenNoOtherSeatIsWorking`、`_IsRaisedWhenANewSeatsFirstReplySaysWorking` | cycle 1 の M16〜M19 は run 4 の打ち直しでも死んだ。M56 は生き残った(Test gaps 6) |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | `TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing`、`_TheOrgsOwnNewPlanAndReportsRaiseNothing`(`watch_scope_test.go`、実際の git)、`TestScopePathInside_FollowsTheReservationRules`、`TestParsePorcelainZ_EntriesInOrderWithBothSidesOfARename`、`_CommitPathsStayRootRelativeUnderDiffRelative` | cycle 1 の M20、M27、M33、M46、M60。D01〜D03。`watch_scope.go` は run 3 のあと変わっていない。M33(`watch.go`)は run 4 の打ち直しでも死んだ |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | `TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths`(7 つの場合、実際の git)、`_OnlyANewPathOrACommitOutsideRaisesScopeChange`、`_WithNoOriginRefACommitOutsideRaisesScopeChange`、`_AConflictResolvedOutsideInAMergeRaisesScopeChange`、`_ADanglingOriginHEADFallsBackToMainThenToTheDiff`、`_CommitPathsStayRootRelativeUnderDiffRelative` の最後の段 | cycle 1 の M21〜M24(M23 は M23b)、M31、M37、M51、M61、C17、C19〜C22(run 3 の打ち直しで死んだ)。C18 は git の引数を固定する行でだけ死んだ(Test gaps 7) |
| AC10 予約のない org は今までどおり | `TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus` と既存の scope_change のテスト | 予約のない org の道を変える変異は当てていない。S12 はこの道を変えていない |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | `TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite` | cycle 1 の M26、M27 |
| AC15 rebase で入った upstream の commit では立たない | `TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing`(rebase と merge)、`_ARebasedOwnCommitOutsideRaisesScopeChange` と S8 の 2 本 | cycle 1 の M25。C19〜C22 |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | `TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange`(8 つの場合)、`TestOwnRecordCandidate_PlansAndReportsButNotTheirTemplates` | cycle 1 の M26、M28b、M29。M30 は run 4 の打ち直しでも生き残った(Test gaps 1) |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | `TestWatch_StatusFile_WrittenBeforeRecurrencesLoadsAndItsPendingAlertTakesARecurrence`、`TestWatch_StatusFile_APendingAlertWithLastRecurredTSLoadsAndDropsIt`、`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads`、`TestInboxRead_LinesWithoutWokenFoldAsBefore` | cycle 1 の M32、M42。C04 は生き残った(Test gaps 4) |
| AC13 help と文書 | `TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope`(help の文を固定する) | 文書の側は /verify と /sync-docs の範囲で、テストは無い |

計画の L1(commit の一覧に失敗したら前の HEAD を据え置く)は `TestWatch_ReservedScope_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle`、`_AFailedCommitListingAsksWhetherGitHasThePreviousHead`、`_APreviousHeadGitDoesNotHaveIsSkippedQuietly` が固定する。`watch_scope.go` とそのテストは run 3 のあと変わっていないので、L1 の変異 M34、M35、C21、C22 は run 4 では打ち直していない(run 3 で死んだ)。

### cycle 3 の cross-review の 1 件(S12)とテスト

| S12 の直し | テスト | 直しの前の形に戻したとき |
| --- | --- | --- |
| 受信箱がすでに持つ pending を、周期の初め(座席の評価の前)に escalate して pending から消す。件の `alert_id` が同じ場合 | `TestWatch_Recurrence_APendingAlertAStoppedWatchEscalatedTakesNoNewFiring`(e1 が resolved と open の 2 行。deadman の掃除で A を e1 にしたあと、1 周期前の状態ファイルを書き戻して読み直し、止まった watch を再現する) | PRE(`14bc17a3^` の `watch.go` と `watch_recurrence.go` を overlay し、テストは HEAD のもの)と E01(呼び出しを消す): resolved の行は「1 ALERTs to the leader; want a new one for the firing after e1 was resolved」と「pending alerts = map[]; want only the new org-a/seat-1/stall@…, with no recurrence」で落ちた。open の行は「e1 recurred events = []; want one with alert_id … and count 1」と、再発の stderr の行が無いことで落ちた |
| 同じく、件の `recurred` の event の `alert_id` が同じ場合 | `TestWatch_Recurrence_APendingAlertOnARecurredEventIsDroppedBeforeTheSeats`(open と resolved の 2 行。deadman はまだ遠い) | PRE と E01: 両方の行で、周期の初めの escalation の stderr の行が無く(「stderr = ""; want "WATCHDOG ESCALATION: … already in the inbox as a recurrence on e1 …"」)、pending が残った。resolved の行は「0 ALERTs to the leader」でも落ちた |
| 受信箱が持たない pending は残す。照らすのは alert_id で、鍵ではない | `TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept`(同じ鍵の open の件と、別の alert_id の `recurred` がある) | PRE と E01 では通った(残す側の形なので、修正の前も同じ)。alert_id の照合を外す E02 と、鍵で照らす E02k で落ちた(下の節) |
| 受信箱が読めないときは何もしない | `TestWatch_Recurrence_AnInboxThatCannotBeReadLeavesThePendingAlertsAsTheyAre` | PRE と E01 では通った(修正の前と同じ挙動を固定するテスト)。読めないときに stderr へ 1 行書く E08 で落ちた |

## 変異

S12 の変異は `go test -overlay` で当てた。`apply.pl` が、元の文字列がファイルにちょうど 1 回出ることと元と新が違うことを確かめてから写しを作り、overlay の JSON で worktree のファイルをその写しに置き換えて `go test -overlay=<json> ./internal/org/ -run <filter> -count=1 -vet=off -timeout 8m` を打った。`-vet=off` は `false &&` の形の変異で vet が止めないようにするためである。変異なしの対照の回は、S12 の 4 本の filter でも、広い filter(`Watch|Inbox|Scope|OwnRecord|Porcelain|Escalat|Deadman|Stall`、42.1 秒)でも ok だった。変異のあいだ、worktree の `git status --porcelain --ignored` は、run-test.sh が書く `.harness/cache/go-build/` の下の 44 行を除いて変わらなかった。

### S12 への 11 個と PRE

| 変異 | 変えたこと | S12 の 4 本の回 | 広い filter の回 |
| --- | --- | --- | --- |
| PRE | `watch.go` と `watch_recurrence.go` を `14bc17a3^` のものに戻す(テストは HEAD) | 死んだ: `_AStoppedWatch…` と `_OnARecurredEvent…` の 4 つの subtest。`_IsKept` と `_AnInboxThatCannotBeRead…` は通った | 打っていない |
| E01 | `evaluateCycle` の `w.reconcilePendingAlertsWithInbox(status, now)` を消す | PRE と同じ | 同じ 2 本だけが落ちた |
| E05 | その呼び出しを座席の評価のあと(deadman の掃除の前)に移す | PRE と同じ | 同じ 2 本だけが落ちた |
| E02 | alert_id の照合を外す(`held := true`。受信箱が何を持っていても escalate する) | `_AStoppedWatch…` の 2 行(回復のあとに A が pending に残らない)と `_IsKept` | 7 本が落ちた(`TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`、`_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem` など) |
| E02k | 件の `alert_id` と `recurred` の `alert_id` の両方を、鍵(`alertIDKey`)で照らす | `_IsKept` だけ | `_IsKept` だけ |
| E03 | 件の `alert_id` の照合だけを外す(`false &&`) | `_AStoppedWatch…` の 2 行だけ | 同じ 1 本だけ |
| E04 | `recurred` の event の照合だけを外す | `_OnARecurredEvent…` の 2 行だけ | 同じ 1 本だけ |
| E06 | `w.org.Inbox == nil` の確かめを外す | 通った | `TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires` が nil pointer の panic で落ちた(受信箱の無い watch で pending が残る周期) |
| E07 | `pending == nil` の `continue` を外す | 通った | 通った。生き残り(等価、Test gaps 14) |
| E08 | 受信箱が読めないとき stderr に 1 行書いてから戻る | `_AnInboxThatCannotBeRead…` | それと `TestWatch_Escalation_WithoutTheInboxWritesTheBannerThenNotifiesThenWritesOneLine/an_inbox_that_cannot_be_read` |
| E09 | `if held` を `if !held` に | 3 本(`_IsKept` を含む) | 7 本 |
| E10 | `escalateAlert` を呼ばずに pending から消すだけにする | `_AStoppedWatch…` と `_OnARecurredEvent…` | 5 本(`TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem`、`TestRunWatchAll_AOnceRunSendsTheUnnotifiedAlertItemOfAPendingAlertUnderItsOwnID` など) |

依頼の 2 つの確かめはこの表のとおりに出た。

- 呼び出しを消す(E01): `TestWatch_Recurrence_APendingAlertAStoppedWatchEscalatedTakesNoNewFiring` の resolved の行が、leader への ALERT が 1 通のまま(新しい ALERT が出ない)で落ちた。open の行も、e1 に `recurred` が足されないことで落ちた。どちらも cross-review が挙げた形(発火が A に畳まれ、deadman の replay で A ごと消える)である。`_IsKept` は E01 では通った。このテストは何も消さないことを見るものなので、呼び出しが無くても通るのが正しい
- alert_id の照合を外す(E02 と E02k): `TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept` が「pending alerts = map[]; want only org-a/seat-1/liveness@… as it was」「escalated = map[…:true]; want none」で落ちた。E02k ではさらに、pending が e1 の再発として escalate され(stderr「… recorded as a recurrence on inbox item e1 (state=open) …」)、受信箱も書き換わった

E02k、E03、E04 はそれぞれ S12 のテストの 1 本だけで死んだ。held の 2 つの照合と alert_id で照らすことは、それぞれ 1 本のテストが固定している(Test gaps 15)。

### 前の cycle の変異の打ち直し

S12 は `watch.go` と `watch_recurrence.go` を変えたので、この 2 つのファイルに当てた前の cycle の変異を、HEAD で同じ定義と filter のまま打ち直した。cycle 1 の 22 個(M01〜M19、M33、M52、M53。M03 は `inbox.go` の行と組)と、run 3 の定義の 29 行のうち `watch.go` と `watch_recurrence.go` の 18 行(M56、C01〜C04、C06、C09〜C16、C23、C07b、C08b、C24)、tech-debt が引く M30、M48、M50 の 43 個である。元の文字列はどれも HEAD でちょうど 1 回出た(`APPLY_FAILED` は 0)。harness は S12 と同じ overlay の形で、定義の `<NL>`/`<TAB>` を読む `apply2.pl` を使った。

- 死んだのは 35 個(cycle 1 の 22 個すべてと、C01、C03、C06、C08b、C09〜C16、C23)
- 生き残ったのは 8 個(M30、M48、M50、M56、C02、C04、C07b、C24)で、run 3 と同じ顔ぶれ
- S12 の 4 本は、前からの変異も殺した。C14 は `_OnARecurredEvent…` でも、C16 は `_IsKept` でも死ぬようになった。M01、M03、M04、M08、C09〜C12 にも、S12 のテストのどれかが落ちる側に加わった

生き残った 8 個のパッケージ全体での打ち直しは、この run では打っていない。cycle 2 で `internal/org` と `internal/cli` 全体でも生き残ったことを確かめた。そのあとに増えたテストは S10 の 1 本(M30、M48、M50 の filter `TestWatch_ReservedScope|…` に入る)と S12 の 4 本(M56、C02、C04、C07b の filter の `TestWatch_Recurrence` と、C24 の filter の `Watch` に入る)だけで、どれも上の filter の中で走った。

`watch_scope.go`、`inbox.go`、`internal/cli/` の変異(cycle 1 の M20〜M32、M34〜M51、M54、M55、M57〜M61 と C17〜C22)は、run 3 のあとそのファイルもテストも変わっていないので打ち直していない。

### probe

`go test -overlay` で `internal/org/` にテストを足して確かめた(コミットしていない。全文は evidence にある)。

- null の pending: watch-status に `"<alert_id>": null` の pending がある周期を、受信箱がその alert_id を持つ場合と持たない場合で打った。HEAD でも E07 の下でも、どちらの場合も `evaluateCycle` が nil pointer で panic した。HEAD の stack は `checkDeadman`(`watch.go:1721`)から呼ぶ `leaderActedSinceAlert`(`watch.go:1769` の `pending.ManifestLen`)である。base `028a8f6d` の `checkDeadman` も `pending.ManifestLen` を nil の確かめなしで読む(1571 行)。E07 の `continue` は、どちらにしても同じ周期で panic する入力にだけ効くので、外から見える違いは無い(Test gaps 14)

## Coverage

- Statement: coverage の回は filter を付けたので、パッケージ全体の数字(40.2%)は意味を持たない。この diff が足したり変えたりした production の行(`git diff -U0 028a8f6d..HEAD` の行と profile の block を突き合わせた)にかかる block は 355 statement 中 343(96.6%)。S12 の行(`14bc17a3^..HEAD`)では 15 statement 中 14(93.3%)
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: `reconcilePendingAlertsWithInbox` 92.9%、`evaluateCycle` 97.1%、`escalateAlert` と `checkDeadman` 100%、`escalateToInbox` 93.8%。`watch_recurrence.go` のほかの関数は `escalateAsRecurrence` の 85.7% を除いて 100%。`watch.go` の `raiseOrClear`、`leaderIsWaiting`、`orgHasOpenLeaderItem`、`seatBusy`、`evaluateSeat` は 100%、`leaderActedSinceAlert` 88.9%。`watch_scope.go` は run 3 と同じで、`scopeRecordsAtBase` 76.9%、`dropOwnRecords` 92.9%、`evaluateReservedScope` 95.0%、`runGitBounded` 92.3%、ほかは 100%
- Notes: 打たれていない 12 statement のうち 11 は run 3 と同じ場所で、S12 の行の追加で `watch.go` と `watch_recurrence.go` の行番号がずれた(run 3 の `watch.go:1766`、`:1893`、`watch_recurrence.go:206` は HEAD の `watch.go:1774`、`:1902`、`watch_recurrence.go:265`)。中身は `watch.go:1902`(新しい件を記録したあと pending の回数を移せなかったときの stderr の行)、`watch_recurrence.go:265`(`escalateAsRecurrence` の追記が resolved 以外の理由で失敗したときの行)、`watch_scope.go:325`、`:329`、`:333`(merge-base と ls-tree の失敗、M30 の道)、`:365`(同じ失敗を受けて何も除かない枝)、`:419`(`gitRun` が nil のときの既定)、`:423`(porcelain が読めない周期)、`:51`(`ErrWaitDelay` を成功とみなす枝)、`watch.go:424`(`realGitStatus`。filter の外の `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` が打つ)、`watch.go:1774`(`leaderActedSinceAlert` の herdr の `agent get` の文字列が基準と違うときの `return true`、C24)。新しい 1 つは `watch_recurrence.go:129`(`reconcilePendingAlertsWithInbox` の `pending == nil` の `continue`、E07)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、`(cached)` の 5 パッケージと `cmd/ralph`、`internal/insights` の打ち直し、この diff のテストの `-v` の回、`-race` の回、coverage の回、変異の対照の回で、落ちたテストも、打ち直しが要ったテストも無かった。flaky と分類したものも無い | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #307: 同じ条件の ALERT が立つたびに受信箱の件が 1 つ増える | 直っている | flapping のテスト(件 1、`recurred` 4、ALERT 1 通)。cycle 1 で、修正前の 2 つの道を外す変異(M01 + M06)では 5 回で 5 件になることを確かめた。M01 と M06 は run 4 の打ち直しでも死んだ |
| issue #307: 待っているだけの leader の stall | 直っている | AC6 と AC7 のテスト。M11〜M19 は run 4 の打ち直しでも死んだ |
| issue #307: 予約の中だけの変更の scope_change | 直っている | AC8 のテスト。AC15 で rebase と merge の誤報も固定されている |
| cross-review cycle 1 の 4 件(S8) | 直っている | 「cycle 2 の記録」の S8 の表。run 4 の打ち直しでも C03、C14 は死んだ(C17、C20 は `watch_scope.go` で、run 3 で死んだ) |
| cross-review cycle 2 の 1 件(S10)と self-review cycle 3 の C3-1(S11) | 直っている | 「run 3 の記録」。`watch_scope.go` とそのテストは run 3 のあと変わっていない |
| cross-review cycle 3 の 1 件(watch が件を記録したあと状態ファイルを保存する前に止まると、次の watch が残った pending に新しい発火を畳み、resolved のあとの発火が失われる) | 直っている | S12 の 2 本が、S12 の前の 2 ファイル(PRE)と呼び出しを消す E01 の下で 4 つの subtest すべて落ち、HEAD で通る |
| 既存の deadman(leader の活動の 3 つの情報源と、probe の「使えない」印の扱い) | 変わっていない | 既存の `TestWatch_Deadman_*` は期待を変えずに通り、C09〜C13 は run 4 の打ち直しでも死んだ。herdr の `agent get` の情報源だけは前から固定されていない(Test gaps 3) |
| 既存の liveness、watch の立て直し(同じ alert_id の件を二重に作らない)、予約のない org の scope_change | 変わっていない | `internal/org` と `internal/cli` 全体が ok。期待を変えた既存のテストは cycle 1 の 2 本だけで、S12 は期待を変えていない |
| 受信箱の無い watch と、受信箱が読めない周期 | 変わっていない | E06 は受信箱の無い既存のテストで panic して死に、受信箱が読めない周期は `_AnInboxThatCannotBeRead…` と既存の `TestWatch_Escalation_WithoutTheInbox…` が E08 で落ちる |
| 古い watch-status と受信箱 | 読める | AC12 のテスト |

## Test gaps

1〜11 は cycle 2 の report(commit `fe4eff60`)の番号のまま、12 と 13 は run 3(commit `24ca3804`)の番号のまま残した。tech-debt の 462 行はこの番号で M30、M50、M48、C24、C04 を引く。14〜16 はこの run で足した。

1. M30(LOW、cycle 1 から): origin の既定のブランチはあるが merge-base が読めないとき(共通の祖先が無い、git が期限で切られた)に、何も除かない枝が固定されていない。変異の下では、自分の記録の候補をすべて除き、予約の外の変更を知らせない側に倒れる。cycle 1 の probe の形の 1 本で死ぬ。run 4 の打ち直しでも生き残った
2. M50(LOW、cycle 1 から): HEAD が読めない周期に前の HEAD を据え置く条件(`head != ""`)が固定されていない。変異の下では、そのあいだの予約の外の commit が知らされない。cycle 1 の probe の形の 1 本で死ぬ。run 4 の打ち直しでも生き残った
3. C24(LOW、この diff の前からある穴): leader の活動の情報源のうち、herdr の `agent get` の文字列が基準と違えば活動とみなす `return true`(HEAD の `watch.go:1774`、run 3 では 1766)を、org と cli のどのテストも固定していない。S8 でこの判定は `foldIntoPendingAlert` も使うようになったので、その情報源だけで leader が動いた場合に、再発が新しい ALERT になるか続きに畳まれるかも固定されていない。cycle 2 の probe の形の 1 本(leader の文字列を 1 回変え、seat の pending が消えること)で死ぬ。run 4 の打ち直しでも生き残った
4. C04(LOW): 1 つの鍵に pending を 2 つ持つ status file(この計画の前のビルドが書いたもの)で、古いほうが答えられたあと新しいほうを試す道が固定されていない。変異の下では、まだ答えられていない pending があるのに新しい ALERT を送る。cycle 2 の probe の形の 1 本で死ぬ。run 4 の打ち直しでも生き残った
5. C02(INFO): leader についての pending には leader が動いたかの判定をかけない分岐が固定されていない。leader についての pending は送った周期の deadman の掃除で escalate されて消えるので、畳むときに残っているのは、その周期が掃除の前に終わったうえで leader の条件がいったん消えてまた立った場合に限られる。run 4 の打ち直しでも生き残った
6. M48(INFO、cycle 1 から): `scopeDefaultBranchRef` が `symbolic-ref` の答えを `refs/remotes/origin/` の下のものに限る検査が固定されていない。cycle 2 で、origin/HEAD を作業中のブランチに向ける probe で死ぬことを確かめた。M56(INFO、等価)は、leader は org に 1 席なので受信箱を周期に 1 回だけ読む印の違いが外から見えない
7. C18(INFO): `--cc` を `-c` にした変異は、git の引数を文字列で固定する行でだけ死んだ。`-c` と `--cc` で出るパスが違う merge はテストに無い
8. C07b(INFO、等価): manifest を一度だけ読む印を無視しても、同じ周期にもう一度読むだけで結果は変わらない
9. 打たれていない失敗の道(Coverage の Notes): 新しい件を記録したあと pending の回数を移せなかったときの行(HEAD の `watch.go:1902`)は stderr の 1 行だけを書く
10. Test plan の Integration の項(`watch_all_test.go` の流れで件数を照らす)は、その形のテストになっていない。新しいテストは `evaluateCycle` と `escalateToInbox` を直に打つ。INFO
11. 実機の道: 新しい binary で機能の org を走らせて受信箱の件を数え直すことはしていない。テストは偽の herdr と一時 repo の実際の git だけで確かめている。S9 の記録にある「再発を足した直後に watch が止まると、立て直した watch が回数を 1 多く数える」穴にはテストが無い。cycle 2 の /sync-docs で、この穴は tech-debt の 462 行の (7) に、C24 と C04 は (8) に入った
12. S10 の新しいテストの 2 つの向き(INFO、run 3 から): `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative` は予約の中の commit を先に見て `t.Fatalf` で止まるので、S10 の前の形では、予約の外の commit を見落とす向きの assert まで進まない。この向きは run 3 の probe で確かめた
13. git の版(INFO、run 3 から): Go のテストは手元の git 2.49.0 だけで走る。S11 の理由は git 2.26.2 と 2.49.0 の probe で確かめたが、2.28 ちょうどの git は打っていない
14. E07(INFO、等価、この run から): `reconcilePendingAlertsWithInbox` の `pending == nil` の `continue`(`watch_recurrence.go:129`)は、どのテストも打たず、外しても生き残る。null の pending を持つ status file では、HEAD でも同じ周期の `checkDeadman` が `leaderActedSinceAlert` で nil pointer の panic を起こすので(probe)、この `continue` があっても無くても外から見える結果は変わらない。この panic は base からある道で、watch 自身は null の pending を書かない。手で書き換えた status file にだけ起きる
15. 1 本のテストだけが殺す変異(INFO、この run から): 鍵で照らす E02k は `_IsKept` だけ、件の `alert_id` の照合を外す E03 は `_AStoppedWatch…` だけ、`recurred` の照合を外す E04 は `_OnARecurredEvent…` だけで死ぬ。受信箱の無い watch で何もしない確かめを外す E06 は、受信箱の無い既存のテストの nil pointer の panic でだけ死ぬ(「受信箱が無ければ pending をそのまま残す」を言う assert は無い)。どれも今は死ぬので穴ではないが、これらのテストを書き換えるときは、上の変異を当て直す必要がある。/verify が挙げた「`escalateToInbox` と `reconcilePendingAlertsWithInbox` の held の判定がそろっていることを見る決定的な検査が無い」もこれに近い。片方だけを変える変異(reconcile の側の E03・E04、`escalateToInbox` の側の C14)はどれも死ぬが、2 つの判定を 1 つの表で照らすテストは無い
16. 止まる窓の再現(INFO、この run から): S12 の 2 本は、1 周期前に保存した status file を書き戻して `loadWatchStatus` で読み直すことで、`escalateAlert` が受信箱に書いたあと状態ファイルを保存する前に watch が止まった場合を再現する。`RunWatchAll` を本当にその窓で止めて立て直すテストは無い

## Verdict

- Pass: 6(6 種類の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通り、`(cached)` だった 5 パッケージも `-count=1` で通った。この diff のテスト 53 本と期待を変えた 2 本の `-v` の回、S12 の 4 本を含む watch と受信箱のテスト 186 本の `-race -count=3` の回(`DATA RACE` 0)、coverage の回、変異の対照の回も 0 failed。AC13 を除く 14 個の AC と AC13 の help の文には、どれも PASS するテストがあり、名前は上の表にある。cross-review cycle 3 の 1 件の直し(S12)は、S12 の前の 2 ファイルに戻すと新しいテスト 2 本の 4 つの subtest がすべて落ち、呼び出しを消す変異(E01)でも同じだった。alert_id の照合を外す変異(E02、E02k)では `_IsKept` が落ちた。S12 への変異 11 個のうち 10 個が死に、生き残った E07 は外から見える違いの無い等価な変異である。前の cycle の変異 43 個の打ち直しでは 35 個が死に、生き残った 8 個は run 3 と同じ LOW か INFO の穴(Test gaps 1〜8)である。変えた production の行の coverage は 355 statement 中 343、S12 の行では 15 statement 中 14)
- Fail: 0
- Blocked: 0。Test gaps 1〜4 のテストを足すか、tech-debt の行に残すかは orchestrator が決める

## run 3 の記録

run 3(cycle 3、HEAD `10a86e28`、report のコミット `24ca3804`)は pass だった。run-test.sh は rc 0 で 1,304 秒(org 703.4 秒、cli 504.9 秒)、この diff のテスト 49 本と期待を変えた 2 本の `-v` と `-race -count=3` も通った。S10・S11 への変異 5 個はすべて死に、cycle 2 の 29 行は死んだ 19、生き残り 8、build 失敗 2(C07、C08)、cycle 1 の `watch_scope.go` の 24 個はすべて死んだ。run 3 の report の全文は `git show 24ca3804:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。D 番号の定義と git の版の probe は下の 2 つの表のとおりである。

### S10・S11 への 5 個(run 3)

| 変異 | 変えたこと | filter | 結果 |
| --- | --- | --- | --- |
| D01n | 予備の `diff` から `-c diff.relative=false` を外す | 新しいテストだけ | 死んだ(`diff` の行だけ) |
| D02n | `log` から `-c diff.relative=false` を外す | 新しいテストだけ | 死んだ(`log` の行だけ) |
| D01 | D01n と同じ | scope の filter 全体 | 死んだ(新しいテストと `_RunsOnlyTheGitCallsItNeeds`) |
| D02 | D02n と同じ | scope の filter 全体 | 死んだ(同じ 2 本) |
| D03 | 両方から外す(S10 の前の形) | scope の filter 全体 | 死んだ(新しいテストの 2 行と、`_RunsOnlyTheGitCallsItNeeds` の引数の 4 か所) |

D03 の下では、`diff.relative=true` と `internal/org` の cwd で予約の外の `cmd/y.go` だけを commit する probe も、`log` と `diff` の両方の行で「scope_change ALERTs = []」になった(Test gaps 12)。

### git の版の probe(run 3)

| git | `-c diff.relative=false` の `log` と `diff` | `--no-relative` の `log` と `diff` | 何も付けない `log` と `diff` |
| --- | --- | --- | --- |
| 2.49.0(手元) | rc 0、`cmd/y.go` と `internal/org/x.go` | rc 0、同じ 2 つ | rc 0、`x.go` だけ(cwd からの相対で、cwd の外が落ちる) |
| 2.26.2(docker の `alpine/git:v2.26.2`) | rc 0、`cmd/y.go` と `internal/org/x.go` | `log` は rc 128(`fatal: unrecognized argument: --no-relative`)、`diff` は rc 129(usage) | rc 0、`cmd/y.go` と `internal/org/x.go`(2.26 は `diff.relative` を読まない) |

## cycle 2 の記録

cycle 2(HEAD `0a957745`、report のコミット `fe4eff60`)は pass だった。run-test.sh は rc 0 で 1,343 秒(org 722.1 秒、cli 536.0 秒)、この diff のテスト 48 本と期待を変えた 2 本の `-v` と `-race -count=3` も通った。S8・S9 への変異 23 個のうち 19 個が死に、cycle 1 からの 4 個と合わせた 8 個は `internal/org`(716.0 秒)と `internal/cli`(429.4 秒)全体を同時に打っても生き残った。cycle 1 で死んだ 57 個も HEAD で死んだ。gap の probe は M30、M50、M48、C04、C24 の 5 本。cycle 2 の report の全文は `git show fe4eff60:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。C 番号の定義は下の 2 つの表のとおりである。

### cycle 1 の cross-review の 4 件(S8)とテスト(cycle 2)

| S8 の直し | テスト | 直しの前の形を変異で戻したとき |
| --- | --- | --- |
| (1) leader が動いた周期の再発を古い pending に畳まず、新しい ALERT にする | `TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`(manifest の `sent`、agmsg の履歴、何もしない、の 3 行)、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert` | C03(判定を外し、S8 の前と同じく必ず畳む): `sent` と履歴の行で「1 ALERTs to the leader; want 2」「pending alerts = map[]」 |
| (2) `git log` に `--cc` を足し、merge 自身の編集を数える | `TestWatch_ReservedScope_AConflictResolvedOutsideInAMergeRaisesScopeChange`(実際の git で衝突を予約の外で解く) | C17(`--cc` を外す): 「scope_change ALERTs = []; want one naming ["other/x.go"]」 |
| (3) 指す先の無い origin/HEAD を飛ばして main・master・`git diff` に落とす | `TestWatch_ReservedScope_ADanglingOriginHEADFallsBackToMainThenToTheDiff`(origin/main がある場合と無い場合) | C20(指す先を確かめない): 2 つの行とも「scope_change ALERTs = []」 |
| (4) `recurred` の event に載った alert_id の escalation の打ち直しで何も足さない | `TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew`(件が open と resolved) | C14(この確かめを外す): open の行では同じ件に `recurred` がもう 1 つ足され、resolved の行では新しい件 `e2` ができた |

### cycle 2 の変異(S8・S9 の変更への 23 個)

| 変異 | 変えたこと | 結果(cycle 2。run 3 と run 4 の打ち直しでも同じ) |
| --- | --- | --- |
| C01 | `foldIntoPendingAlert` で leader が動いたかを見ない(`false &&`) | 死んだ |
| C02 | leader についての pending にも leader が動いたかの判定をかける(`if true`) | 生き残った(Test gaps 5) |
| C03 | どの pending にも判定をかけない(`if false`、S8 の前の形) | 死んだ |
| C04 | 答えられた pending を消したあと、同じ鍵の次の pending を試さずに抜ける(`continue` を `return false` に) | 生き残った(Test gaps 4) |
| C06 | manifest が読めないとき畳む側に倒す(`return true`) | 死んだ |
| C07b | manifest を一度だけ読む印を無視する(`!read \|\| true`) | 生き残った(等価、Test gaps 8) |
| C08b | 読んだ manifest の event を捨てる(`rr.Events[:0]`) | 死んだ |
| C09、C12 | `leaderActedSinceAlert` の `>` を `>=` に(manifest の数、履歴の行数) | 死んだ |
| C10、C11 | probe の基準が「使えない」印のときも比べる(herdr の `agent get`、履歴の行数) | 死んだ |
| C13 | checkDeadman が `leaderActedSinceAlert` を呼ばない | 死んだ |
| C24 | herdr の `agent get` の文字列が基準と違っても活動とみなさない | 生き残った(Test gaps 3) |
| C14 | `escalateToInbox` で `recurred` に載った alert_id を見ない | 死んだ |
| C15、C16 | `alertItemWithRecurredID` の event の種類の条件、alert_id の条件を外す | 死んだ |
| C23 | `pendingAlertForKey` が alert_id を返さない | 死んだ |
| C17 | `git log` から `--cc` を外す | 死んだ(run 4 では打ち直していない) |
| C18 | `--cc` を `-c` に | git の呼び出しのテストの引数の行でだけ死んだ(Test gaps 7。run 4 では打ち直していない) |
| C19 | `--cc` を `-m` に | 死んだ(run 4 では打ち直していない) |
| C20 | origin/HEAD の指す先を確かめない | 死んだ(run 4 では打ち直していない) |
| C21 | `namesCommit` がいつも真 | 死んだ(run 4 では打ち直していない) |
| C22 | main・master を確かめずに使う | 死んだ(run 4 では打ち直していない) |

C07(`if !read {` を `if true {`)と C08(`events = rr.Events` を `nil`)は変数が使われなくなって build が通らなかったので、同じ意味の C07b と C08b に替えた。cycle 2 の gap の probe は、M30(origin/main を HEAD と共通の祖先の無い root commit に向ける)、M50(1 周期だけ `rev-parse --verify -q HEAD` を失敗させる)、M48(origin/HEAD を作業中のブランチに向ける)、C04(1 つの鍵に pending を 2 つ持つ古い status file)、C24(leader の `agent get` の文字列を 1 回変える)の 5 本で、どれも HEAD で通り、対象の変異の下でだけ落ちた。

## cycle 1 の記録

cycle 1(HEAD `b8058948`、report のコミット `3506f99a`)は pass だった。run-test.sh は rc 0 で 1,623 秒(org 910.6 秒、cli 673.0 秒)、新しいテストの `-v` と `-race -count=3` も通った。変異 61 個のうち 57 個が死に、M30(LOW)、M50(LOW)、M48(INFO)、M56(INFO、等価)の 4 個が org と cli 全体でも生き残った。4 個とも run 4 の HEAD でまだ生き残っている。cycle 1 の report の全文は `git show 3506f99a:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。
