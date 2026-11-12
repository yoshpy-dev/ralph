# Test report: watch-alert-dedupe

- Date: 2026-11-12(run-test.sh は 09:41:22 から 10:03:45 JST)
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(`## Test plan`、AC1〜AC15)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2)
- Scope: `git diff 028a8f6d...HEAD`(HEAD `0a957745`)。cycle 1(HEAD `b8058948`)のあとの production の変更は S8 `907afde3`(cross-review cycle 1 の 4 件の直し)と S9 `5393a316`(片付け)、テストの変更はその 2 つと `3bb1828b`(コメント)である。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。worktree の production code とテストには手を入れていない。変異は `git archive HEAD` で作った scratchpad の写しに当て、gap の probe は `go test -overlay` で足した
- Evidence: `docs/evidence/test-2026-11-12-watch-alert-dedupe.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、meta、`(cached)` だった 7 パッケージの打ち直し、新しいテストの PASS 行、`-race` の回、coverage、変異の harness・定義・結果、生き残った変異のパッケージ全体での打ち直し、gap の probe の全文と結果を足した。cycle 1 の evidence は `docs/evidence/test-2026-11-12-watch-alert-dedupe-cycle1.log` に残した。wrapper 自身のログは `docs/evidence/verify-2026-11-12-004122.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,343 秒。1 分の load は開始時 12.05、終了直後 6.48 |
| └ Go: `internal/org` | 上位 1,632 本 | ok | 0 | — | 722.1 秒 |
| └ Go: `internal/cli` | 上位 1,079 本 | ok | 0 | — | 536.0 秒 |
| └ Go: `internal/config` | — | ok | 0 | — | 12.0 秒 |
| └ Go: `cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | run-test.sh では `(cached)`。`-count=1` で打ち直して ok | 0 | — | 打ち直しは 1.0、2.5、0.8、3.2、2.2、1.6、2.3 秒 |
| この diff が足したテスト 48 本と期待を変えた 2 本(`-v -count=1`) | org 上位 47 本(subtest 込みの PASS 行 135)、cli 3 本 | 138 | 0 | 0 | 20 秒、4 秒 |
| 同じテストを `-race -count=3` | org の PASS 行 405、cli 9 | 414 | 0 | 0 | 91 秒、16 秒。`DATA RACE` の行は 0 |
| coverage の回(`-run 'Watch\|Inbox\|Scope\|OwnRecord\|Porcelain\|Escalat\|Deadman\|Stall'`、`-coverprofile`) | — | ok | 0 | — | 58 秒 |
| 変異: cycle 1 の生き残り 4 個の打ち直しと、S8・S9 の変更への 23 個(下の節) | 27 | 新しい 23 個のうち 19 個は既存のテストで死んだ | — | — | 1 個 6〜47 秒 |
| 生き残った 8 個を同時に当てて `internal/org` と `internal/cli` 全体 | — | ok(8 個とも生き残った) | 0 | — | org 716.0 秒、cli 429.4 秒 |
| cycle 1 の変異のうち死んだ 57 個を HEAD の写しで打ち直し | 57 | 57 個とも死んだ(M23 は M23b に移して打った) | — | — | 1 個 5〜22 秒 |
| gap の probe 5 本(`go test -overlay`、HEAD と 5 つの変異の下で) | 5 本(subtest 6 つ) | HEAD ですべて通った。各変異の下では、その変異に向けた probe だけが落ちた | 0 | 0 | 1 回 1〜5 秒 |

上位の本数は `go test -list '.*'` で数えた。cycle 1 の 1,627 本から 5 本増えた。S8 が足したのは org の 6 本(`watch_test.go` 4、`watch_scope_test.go` 2)で、S9 がそのうち 1 本(本番では当たらない `recordRecurrence` の打ち直しの確かめ)を消した。この diff 全体では org に 45 本、cli に 3 本が増え、消えたテストは無い。期待を変えた base のテスト 2 本は cycle 1 と同じ(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)で、S8 と S9 は base のテストの期待を変えていない。S8 はこの branch のテスト 2 本(`_RunsOnlyTheGitCallsItNeeds`、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`)に、`--cc` の引数、origin/HEAD の指す先の確かめ、`alertItemWithRecurredID` と `pendingAlertForKey` の返す alert_id の assert を足した。diff を読むと、書き換えた assert は `git log` の期待の文字列に `--cc` を入れたものと、`pendingAlertForKey` の期待を「同じ alert_id」から「同じ alert_id と同じ pointer」にしたもので、弱めたものは無い。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(rc と経過秒を meta に書く)を Bash tool の `run_in_background` で起動し、`caffeinate -w <pid>` を perl の `alarm` で 570 秒に区切った前景の待ちをくり返した。変異と生き残りの打ち直しは run-test.sh の shell の段と Go の段に重ねて走らせた。そのあいだに読んだ 1 分の load は最大 17.86 だった。run-test.sh は落ちず、打ち直しが要ったテストも無かった。

run-test.sh で `(cached)` になった 7 パッケージのうち、`cmd/ralph` と `internal/insights` は `internal/org` に依存する(`go list -deps`)。Go の test cache は依存の中身を鍵に含むので、同じ source で通った結果のはずだが、`-count=1` で打ち直してすべて ok だった。

Skipped の 2 行は cycle 1 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-v` の回で PASS した。「変異」の列の M 番号は cycle 1 の定義(`-cycle1.log`)、C 番号はこの cycle の定義である。

| AC | テスト(ファイル) | 変異で確かめたこと |
| --- | --- | --- |
| AC1 件は 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go`)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestInboxAppendRecurred_RecordsOnOpenAndAckedALERTItems`(`inbox_test.go`)、`TestOrgInbox_ShowsTheRecurrences`、`TestOrgInbox_ShowsOneRecurrenceAsOneTime`(`org_inbox_test.go`)。S8 で `TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew`(escalation の打ち直しで `recurred` を二重に数えない) | cycle 1 の M01、M08、M38、M39、M42〜M45、M59。C14、C15、C16 は死んだ |
| AC2 resolved のあとは新しい件 | flapping のテストの後半、`TestInboxAppendRecurred_RefusesAndWritesNothing`、`TestWatch_Recurrence_ARecurrenceThatIsNotRecordedSendsANewAlert`、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`。S8 の escalation の打ち直しのテストの resolved の行(resolved の件の `recurred` に載った alert_id では新しい件を作らない) | cycle 1 の M02、M03。C14 は resolved の行で新しい件 `e2` を作って死んだ |
| AC3 続きのあいだは leader に ALERT を送らない | flapping のテスト(ALERT 1 通、watcher の起動 1 回)。S8 で `TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`(leader が動いた周期の再発は続きでなく新しい ALERT。何もしなければ続き)、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert`(manifest が読めなければ知らせる側に倒す) | cycle 1 の M01。C01、C03、C06、C08b は死んだ |
| AC4 deadman の前の再発は pending を増やさず、件に回数が残る | `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`、期待を変えた `TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`。S8 の leader が動いたかのテストの「何もしなかった」行(pending 1 つ、回数 1、deadman で件に回数 1) | cycle 1 の M04、M05、M07。leader の活動の判定(`leaderActedSinceAlert`、checkDeadman と共有)の C09〜C13 は死んだ。C24 は生き残った(Test gaps 3) |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | `TestWatch_Recurrence_ANewWatchFindsTheOpenItemOfTheKeyInTheInbox`、`TestWatch_Recurrence_AnEscalationWhoseKeyHasAnItemThatIsNotResolvedIsARecurrenceOfIt`。S8 の escalation の打ち直しのテスト(`recurred` を記録したあと status を保存する前に止まった watch の続き) | cycle 1 の M06、M08、M09、M10、M52、M53。C14〜C16、C23 は死んだ |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たない | `TestWatch_LeaderStall_IsNotRaisedWhileItsOrgHasAnItemItsLeaderRaised`、`TestWatch_LeaderStall_IsRaisedWhenItsOrgHasNoItemItsLeaderRaisedOpen` | cycle 1 の M11〜M15。S8・S9 はこの道を変えていない |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば立つ | `TestWatch_LeaderStall_IsNotRaisedWhileAnotherSeatIsWorking`、`_IsRaisedWhenNoOtherSeatIsWorking`、`_IsRaisedWhenANewSeatsFirstReplySaysWorking` | cycle 1 の M16〜M19。M56 は cycle 1 と同じく生き残った(Test gaps 6) |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | `TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing`、`_TheOrgsOwnNewPlanAndReportsRaiseNothing`(`watch_scope_test.go`、実際の git)、`TestScopePathInside_FollowsTheReservationRules`、`TestParsePorcelainZ_EntriesInOrderWithBothSidesOfARename` | cycle 1 の M20、M27、M33、M46、M60 |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | `TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths`(7 つの場合、実際の git)、`_OnlyANewPathOrACommitOutsideRaisesScopeChange`、`_WithNoOriginRefACommitOutsideRaisesScopeChange`。S8 で `_AConflictResolvedOutsideInAMergeRaisesScopeChange`(merge 自身の編集は立つ)、`_ADanglingOriginHEADFallsBackToMainThenToTheDiff`(指す先の無い origin/HEAD を飛ばす) | cycle 1 の M21〜M24、M31、M37、M51、M61。C17、C19〜C22 は死んだ。C18 は git の引数を固定する行でだけ死んだ(Test gaps 7) |
| AC10 予約のない org は今までどおり | `TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus` と既存の scope_change のテスト | cycle 1 と同じく、予約のない org の道を変える変異は当てていない |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | `TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite` | cycle 1 の M26、M27 |
| AC15 rebase で入った upstream の commit では立たない | `TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing`(rebase と merge)、`_ARebasedOwnCommitOutsideRaisesScopeChange`。S8 の 2 本(merge で upstream から入るだけの `other/new.go` は本文に出ない。origin/HEAD が指す先の無いときも origin/main で upstream を除く) | cycle 1 の M25。C19(`--cc` を `-m` に)は upstream のパスを数えて死んだ。C20〜C22 も死んだ |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | `TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange`(8 つの場合)、`TestOwnRecordCandidate_PlansAndReportsButNotTheirTemplates` | cycle 1 の M26、M28b、M29。M30 は生き残った(Test gaps 1) |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | `TestWatch_StatusFile_WrittenBeforeRecurrencesLoadsAndItsPendingAlertTakesARecurrence`、`TestWatch_StatusFile_APendingAlertWithLastRecurredTSLoadsAndDropsIt`、`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads`、`TestInboxRead_LinesWithoutWokenFoldAsBefore` | cycle 1 の M32、M42。古い status file が 1 つの鍵に pending を 2 つ持つ場合の C04 は生き残った(Test gaps 4) |
| AC13 help と文書 | `TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope`(help の文を固定する) | 文書の側は /verify と /sync-docs の範囲で、テストは無い |

計画の L1(commit の一覧に失敗したら前の HEAD を据え置く)は `TestWatch_ReservedScope_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle`、`_AFailedCommitListingAsksWhetherGitHasThePreviousHead`、`_APreviousHeadGitDoesNotHaveIsSkippedQuietly` が固定する。git を打つ回数と引数は `_RunsOnlyTheGitCallsItNeeds` が固定し、S8 で `--cc` と origin/HEAD の指す先の確かめ(`rev-parse --verify --quiet <ref>^{commit}`)の行が足された。

### cycle 1 の cross-review の 4 件(S8)とテスト

| S8 の直し | テスト | 直しの前の形を変異で戻したとき |
| --- | --- | --- |
| (1) leader が動いた周期の再発を古い pending に畳まず、新しい ALERT にする | `TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`(manifest の `sent`、agmsg の履歴、何もしない、の 3 行)、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert` | C03(判定を外し、S8 の前と同じく必ず畳む): `sent` と履歴の行で「1 ALERTs to the leader; want 2」「pending alerts = map[]」。再発は消えた pending と一緒に失われ、leader に届かない。修正後は ALERT 2 通、新しい pending 1 つ、deadman で件 1 つ |
| (2) `git log` に `--cc` を足し、merge 自身の編集を数える | `TestWatch_ReservedScope_AConflictResolvedOutsideInAMergeRaisesScopeChange`(実際の git で衝突を予約の外で解く) | C17(`--cc` を外す): 「scope_change ALERTs = []; want one naming ["other/x.go"]」。予約の外の衝突の解決が知らされない |
| (3) 指す先の無い origin/HEAD を飛ばして main・master・`git diff` に落とす | `TestWatch_ReservedScope_ADanglingOriginHEADFallsBackToMainThenToTheDiff`(origin/main がある場合と無い場合) | C20(指す先を確かめない): 2 つの行とも「scope_change ALERTs = []」。`git log --not <消えた ref>` が毎周期失敗し、commit が知らされない |
| (4) `recurred` の event に載った alert_id の escalation の打ち直しで何も足さない | `TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew`(件が open と resolved) | C14(この確かめを外す): open の行では同じ件に `recurred` がもう 1 つ足され、resolved の行では新しい件 `e2` ができて人に通知が飛んだ |

## 変異

harness は cycle 1 と同じ形で、`mutate.sh` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることと元と新が違うことを確かめてから置き換え、`go test ./internal/org/ -run <filter> -count=1 -timeout 8m` を打ち、写しの元のファイルに戻して `cmp` で確かめた。変異を当てる前に、filter をすべて合わせた回を変異なしの写しで打ち、ok だった(26.3 秒)。C07 と C08 は変数が使われなくなって build が通らなかったので、同じ意味の C07b と C08b に替えた。変異のあいだ、worktree の `git status --porcelain --ignored` は、run-test.sh が書く `.harness/cache/go-build/` の下を除いて変わらなかった。

### cycle 1 の生き残り 4 個

M30、M48、M50、M56 は cycle 1 と同じ定義で、元の文字列は HEAD でもそれぞれ 1 回だけ出た。4 個とも filter の回で生き残り、下の 8 個同時のパッケージ全体の回でも生き残った。S8 は M48 の隣(`scopeDefaultBranchRef` の origin/HEAD の分岐)に指す先の確かめを足したが、`refs/remotes/origin/` の下に限る検査はまだどのテストも固定していない。

### S8・S9 の変更への 23 個

| 変異 | 変えたこと | 結果 |
| --- | --- | --- |
| C01 | `foldIntoPendingAlert` で leader が動いたかを見ない(`false &&`) | 死んだ(leader が動いたかのテストの 2 行) |
| C02 | leader についての pending にも leader が動いたかの判定をかける(`if true`) | 生き残った(Test gaps 5) |
| C03 | どの pending にも判定をかけない(`if false`、S8 の前の形) | 死んだ(上の 2 本) |
| C04 | 答えられた pending を消したあと、同じ鍵の次の pending を試さずに抜ける(`continue` を `return false` に) | 生き残った(Test gaps 4) |
| C06 | manifest が読めないとき畳む側に倒す(`return true`) | 死んだ |
| C07b | manifest を一度だけ読む印を無視する(`!read \|\| true`) | 生き残った(等価。同じ周期に manifest をもう一度読むだけ) |
| C08b | 読んだ manifest の event を捨てる(`rr.Events[:0]`) | 死んだ |
| C09、C12 | `leaderActedSinceAlert` の `>` を `>=` に(manifest の数、履歴の行数) | 死んだ(既存の deadman のテスト、C09 は 12 本、C12 は 11 本。どちらも再発のテスト 3 本でも) |
| C10、C11 | probe の基準が「使えない」印のときも比べる(herdr の `agent get`、履歴の行数) | 死んだ(C10 は `TestWatch_Deadman_ProbeOutage*` など 3 本、C11 は `TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop` 1 本) |
| C13 | checkDeadman が `leaderActedSinceAlert` を呼ばない | 死んだ(deadman のテスト 10 本) |
| C24 | herdr の `agent get` の文字列が基準と違っても活動とみなさない | 生き残った(Test gaps 3) |
| C14 | `escalateToInbox` で `recurred` に載った alert_id を見ない | 死んだ |
| C15、C16 | `alertItemWithRecurredID` の event の種類の条件、alert_id の条件を外す | 死んだ(`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`。C16 は flapping のテストでも) |
| C23 | `pendingAlertForKey` が alert_id を返さない | 死んだ |
| C17 | `git log` から `--cc` を外す | 死んだ(衝突のテストと git の呼び出しのテスト) |
| C18 | `--cc` を `-c` に | git の呼び出しのテストの引数の行でだけ死んだ(Test gaps 7) |
| C19 | `--cc` を `-m` に | 死んだ(衝突のテスト、upstream を除くテスト、git の呼び出しのテスト) |
| C20 | origin/HEAD の指す先を確かめない | 死んだ |
| C21 | `namesCommit` がいつも真 | 死んだ(6 本) |
| C22 | main・master を確かめずに使う | 死んだ(6 本) |

生き残った 8 個(M30、M48、M50、M56、C02、C04、C07b、C24)は、同時に当てて `go test ./internal/org/ -count=1 -timeout 30m`(ok、716.0 秒)と `go test ./internal/cli/ -count=1 -timeout 30m`(ok、429.4 秒)で打ち直し、パッケージ全体でも生き残った。1 個ずつのパッケージ全体の回は打っていないので、組み合わせが互いの赤を隠した可能性は残る。C02、C04、C07b は同じ `foldIntoPendingAlert` の別の行を変え、ほかの 5 個は別々の関数を変える。

cycle 1 の変異のうち死んだ 57 個(build の通らなかった M28 の代わりの M28b を含む。生き残った 4 個は上で打ち直した)を、同じ定義と filter のまま HEAD の写しで打ち直した。S8 と S9 が変えた `recordRecurrence`、`pendingAlertForKey`、`escalateToInbox`、`scopeDefaultBranchRef`、`scopeCommittedPaths` を通る変異も含め、57 個とも死んだ。M23(`git log` から `--no-renames` を外す)だけは、S8 が `--cc` を足して元の文字列がファイルに無くなったので、同じ変更を今の引数に当てる M23b に移した。M23b は `_AChangeOutsideRaisesScopeChangeNamingItsPaths` と `_RunsOnlyTheGitCallsItNeeds` で死んだ。S8・S9 のあとも、cycle 1 で確かめたテストの強さは落ちていない。

### gap の probe

HEAD で通り、その変異だけで赤になるテストを、`go test -overlay` で `internal/org/` に足して確かめた(コミットしていない。全文は evidence にある)。M30 と M50 の probe は cycle 1 のものをそのまま使った。

- M30: `refs/remotes/origin/main` を HEAD と共通の祖先の無い root commit に向け、予約の外に新しい計画を書く。HEAD では scope_change が 1 件立ち、M30 の下では 0 件になった。origin を分岐点に置いた対照の行は、どちらでも通った
- M50: 1 周期だけ `rev-parse --verify -q HEAD` を失敗させ、そのあいだに予約の外を commit する。HEAD では次の周期で 1 件立ち、M50 の下では `ScopeHead` が空になって 0 件になった
- M48(この cycle で足した): `refs/remotes/origin/HEAD` を `refs/heads/main`(作業中のブランチ自身)に向け、予約の外を commit する。HEAD ではその向き先を使わず origin/main で比べて 1 件立ち、M48 の下では `--not refs/heads/main` が自分の commit をすべて隠して 0 件になった
- C04(この cycle で足した): 1 つの鍵に pending を 2 つ持つ古い status file で、古いほうのあとに leader が動いた場合。HEAD では古いほうを消して新しいほうに畳み(回数 1、ALERT なし)、C04 の下では畳まずに false を返した。pending が 1 つで答えられた対照の行は、どちらでも false で通った
- C24(この cycle で足した): leader の `agent get` の文字列を、ALERT のときの `idle-A` から次の周期に `busy-B` へ変える。HEAD では deadman の掃除が seat-1 の pending を消し、C24 の下では残った。文字列を変えない対照の行は、どちらでも pending が残って通った

## Coverage

- Statement: coverage の回は filter を付けたので、パッケージ全体の数字(40.1%)は意味を持たない。この diff が足したり変えたりした production の行(`git diff -U0 028a8f6d..HEAD` の行と profile の block を突き合わせた)にかかる block は 340 statement 中 329(96.8%)。cycle 1 のあと(`b8058948..HEAD`)の行では 47 statement 中 46
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: `watch_recurrence.go` は `escalateAsRecurrence` の 85.7% を除いて 100%(S8 で足した `foldIntoPendingAlert` と `alertItemWithRecurredID` も 100%)。`watch.go` は `checkDeadman`、`raiseOrClear`、`leaderIsWaiting`、`orgHasOpenLeaderItem`、`seatBusy`、`evaluateSeat` が 100%、`evaluateCycle` 97.0%、`escalateToInbox` 93.8%、S8 で切り出した `leaderActedSinceAlert` 88.9%。`watch_scope.go` は `scopeRecordsAtBase` 76.9%、`dropOwnRecords` 92.9%、`evaluateReservedScope` 95.0%、`runGitBounded` 92.3% で、ほかは 100%
- Notes: 打たれていない 11 statement のうち 10 は cycle 1 と同じ場所(行番号は HEAD のもの)で、`watch.go:1893`(新しい件を記録したあと pending の回数を移せなかったときの stderr の行)、`watch_recurrence.go:206`(`escalateAsRecurrence` の追記が resolved 以外の理由で失敗したときの行)、`watch_scope.go:319`、`:323`、`:327`(merge-base と ls-tree の失敗、M30 の道)、`:359`(同じ失敗を受けて何も除かない枝)、`:413`(`gitRun` が nil のときの既定)、`:417`(porcelain が読めない周期)、`:51`(`ErrWaitDelay` を成功とみなす枝)、`watch.go:424`(`realGitStatus`。filter の外の `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` が打つ)である。新しく数えに入った 1 つは `watch.go:1766` で、`leaderActedSinceAlert` の herdr の `agent get` の文字列が基準と違うときの `return true` である。S8 が checkDeadman から動かした行で、C24 がパッケージ全体で生き残ったので、filter の外のテストもこの行の結果を固定していない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、`(cached)` の 7 パッケージの打ち直し、新しいテストの `-v` の回、`-race` の回、coverage の回で、落ちたテストも、打ち直しが要ったテストも無かった。flaky と分類したものも無い | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #307: 同じ条件の ALERT が立つたびに受信箱の件が 1 つ増える | 直っている | flapping のテスト(件 1、`recurred` 4、ALERT 1 通)。cycle 1 で、修正前の 2 つの道を外す変異(M01 + M06)では 5 回で 5 件になることを確かめた |
| issue #307: 待っているだけの leader の stall | 直っている | AC6 と AC7 のテスト |
| issue #307: 予約の中だけの変更の scope_change | 直っている | AC8 のテスト。AC15 で rebase と merge の誤報も固定されている |
| cross-review cycle 1 の 4 件 | 直っている | 上の S8 の表。4 件とも、直しの前の形に戻す変異で専用のテストが赤になる |
| 既存の deadman(leader の活動の 3 つの情報源と、probe の「使えない」印の扱い) | 変わっていない | S8 は判定を `leaderActedSinceAlert` に移した。既存の `TestWatch_Deadman_*` は期待を変えずに通り、C09〜C13 で赤になった。herdr の `agent get` の情報源だけは前から固定されていない(Test gaps 3) |
| 既存の liveness、watch の立て直し(同じ alert_id の件を二重に作らない)、予約のない org の scope_change | 変わっていない | `internal/org` と `internal/cli` 全体が ok。期待を変えた既存のテストは cycle 1 の 2 本だけ |
| 古い watch-status と受信箱 | 読める | AC12 のテスト |

## Test gaps

1. M30(LOW、cycle 1 から): origin の既定のブランチはあるが merge-base が読めないとき(共通の祖先が無い、git が期限で切られた)に、何も除かない枝が固定されていない。変異の下では、自分の記録の候補をすべて除き、予約の外の変更を知らせない側に倒れる。上の probe の形の 1 本で死ぬ
2. M50(LOW、cycle 1 から): HEAD が読めない周期に前の HEAD を据え置く条件(`head != ""`)が固定されていない。変異の下では、そのあいだの予約の外の commit が知らされない。上の probe の形の 1 本で死ぬ
3. C24(LOW、この diff の前からある穴): leader の活動の情報源のうち、herdr の `agent get` の文字列が基準と違えば活動とみなす `return true`(`watch.go:1766`)を、org と cli のどのテストも固定していない。既存のテストがこの情報源について見るのは、基準が「使えない」印のときに比べないこと(C10)だけである。S8 でこの判定は `foldIntoPendingAlert` も使うようになったので、その情報源だけで leader が動いた場合に、再発が新しい ALERT になるか続きに畳まれるかも固定されていない。S8 は判定を動かしただけで、deadman のテストは base から変わっていないので、base でも同じ穴だったとみられる(base では変異を打っていない)。上の probe の形の 1 本(leader の文字列を 1 回変え、seat の pending が消えること)で死ぬことを確かめた
4. C04(LOW): 1 つの鍵に pending を 2 つ持つ status file(この計画の前のビルドが書いたもの)で、古いほうが答えられたあと新しいほうを試す道が固定されていない。`foldIntoPendingAlert` の doc はその動きを書いている。変異の下では、まだ答えられていない pending があるのに新しい ALERT を送る。上の probe の形の 1 本で死ぬ
5. C02(INFO): leader についての pending には leader が動いたかの判定をかけない分岐が固定されていない。leader についての pending は送った周期の deadman の掃除で escalate されて消える(`escalateAlert` は必ず消す)ので、畳むときに残っているのは、その周期が掃除の前に終わった(manifest の読み直しの失敗)うえで leader の条件がいったん消えてまた立った場合に限られる
6. M48(INFO、cycle 1 から): `scopeDefaultBranchRef` が `symbolic-ref` の答えを `refs/remotes/origin/` の下のものに限る検査が固定されていない。この cycle で、origin/HEAD を作業中のブランチに向ける probe で死ぬことを確かめた。M56(INFO、等価)は cycle 1 のとおりで、leader は org に 1 席なので受信箱を周期に 1 回だけ読む印の違いは外から見えない
7. C18(INFO): `--cc` を `-c` にした変異は、git の引数を文字列で固定する行でだけ死んだ。実際の git で衝突を解くテストは `-c` でも通った。`--name-only` では、このテストの場合 2 つの選択肢が同じパスを出す。`-c` と `--cc` で出るパスが違う merge はテストに無い
8. C07b(INFO、等価): manifest を一度だけ読む印を無視しても、同じ周期にもう一度読むだけで結果は変わらない
9. 打たれていない失敗の道(Coverage の Notes): cycle 1 の 5 と同じ。新しい件を記録したあと pending の回数を移せなかったときの行(`watch.go:1893`)は stderr の 1 行だけを書く
10. Test plan の Integration の項(`watch_all_test.go` の流れで件数を照らす)は、cycle 1 と同じくその形のテストになっていない。新しいテストは `evaluateCycle` と `escalateToInbox` を直に打つ。INFO
11. 実機の道: 新しい binary で機能の org を走らせて受信箱の件を数え直すことはしていない。テストは偽の herdr と一時 repo の実際の git だけで確かめている。S9 の記録にある「再発を足した直後に watch が止まると、立て直した watch が回数を 1 多く数える」穴にはテストが無い。計画は tech-debt の既知の穴の行に足すとしているが、HEAD の `docs/tech-debt/README.md` のその行(462 行、「Six gaps」)にはまだ無い(/sync-docs の範囲)。同じ行の (6) は cycle 1 の M30、M50、M48 を挙げていて、この cycle の C24 と C04 は入っていない

## Verdict

- Pass: 7(7 種類の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通り、`(cached)` だった 7 パッケージも `-count=1` で通った。この diff のテスト 48 本と期待を変えた 2 本の `-v` の回、その `-race -count=3` の回(`DATA RACE` 0)、coverage の回、生き残った変異 8 個をパッケージ全体で打ち直した org と cli の回、gap の probe の HEAD の回も 0 failed。AC13 を除く 14 個の AC と AC13 の help の文には、どれも PASS するテストがあり、名前は上の表にある。cross-review cycle 1 の 4 件の直しは、どれも直しの前の形に戻す変異で専用のテストが赤になる。S8・S9 への変異 23 個のうち 19 個は既存のテストが殺し、cycle 1 で死んだ 57 個も HEAD でまた死んだ。生き残った 4 個と cycle 1 からの 4 個はどれも LOW か INFO の穴(Test gaps 1〜8)で、LOW の 4 個(M30、M50、C24、C04)はどれも probe 1 本で死ぬ。変えた production の行の coverage は 340 statement 中 329)
- Fail: 0
- Blocked: 0。Test gaps 1〜4 のテストを足すか、tech-debt の行に足すかは orchestrator が決める

## cycle 1 の記録

cycle 1(HEAD `b8058948`、report のコミット `3506f99a`)は pass だった。run-test.sh は rc 0 で 1,623 秒(org 910.6 秒、cli 673.0 秒)、新しいテストの `-v` と `-race -count=3` も通った。変異 61 個のうち 57 個が死に、M30(LOW)、M50(LOW)、M48(INFO)、M56(INFO、等価)の 4 個が org と cli 全体でも生き残った。この cycle の再確認で、4 個とも HEAD でまだ生き残っている。M30 と M50 は cycle 1 の probe で、M48 はこの cycle の probe で死ぬ。cycle 1 の report の全文は `git show 3506f99a:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。
