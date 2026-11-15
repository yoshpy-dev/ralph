# Test report: org-reseat(`ralph org reseat`、#312c の 1 本目)、パイプラインの 4 回目(cycle 3 の cross-review の直しのあと)

- Date: 2026-11-15 JST(run-test.sh は 10:41:16Z から 11:05:26Z、JST では 19:41:16 から 20:05:26)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(`## Test plan`、AC-1〜AC-11、AC-15〜AC-21。AC-12 は実機の記録 `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、ここでは繰り返していない)
- Tester: tester サブエージェント(`/test`)。パイプラインの 4 回目で、cycle 3 の cross-review の ACTION_REQUIRED 2 件を直したあとの回である(`cycle-count.json` は `{"cycle": 3, "cap": 3}`。insight event の `--cycle auto` もこの値を読むので `cycle` は 3 になる)。この report は 3 回目の本文(c856fc27)を置き換える。3 回目の本文は `git show c856fc27:docs/reports/test-2026-11-12-org-reseat.md`、2 回目は `git show 2d37fa32:...`、1 回目は `git show 69908490:...` で読める
- Scope: HEAD b4215caf。3 回目の報告のあとに入ったのは、origin/main の取り込み(merge 46dfc5a8、merge base bbe2807d。PR #328 の `ralph org send` の bracketed paste)と、cross-review cycle 3 の直し 4 つとテスト 1 つである。直しは ffd32109(DECISION の `ralph org send` と、その前に出す `ralph org read` に `--config` を付ける。`ReseatDecisionCommand` の引数、`ReseatResult.DecisionRead`、`reseatLeaderReadCommand`)、84346015(立てる直前に台帳を読み直し、計画のあとに org が disband されていたら立てずに終わる。`ReseatPlan.LastDisbandedIndex`、`endIfDisbanded`、`reseatDisbandedNow`、`reseatDisbandedErr`、`ReseatResult.Disbanded`。disband で終わったときは `finish` が `Retry`・`Decision`・`DecisionRead` を空にする。新しい `internal/org/reseat_disband_test.go`)、c4f55aef(止める直前にも同じ確かめをする。`checkBeforeStop`、`reseatDisbandedIn`)、b568ca8c(disband で終わった reseat の出力で、止めなかった座席を `not stopped: <seat> (the org was disbanded meanwhile)` と書く。`orgReseatLeftLine`)で、f2ed6bdf は self-review の L23 を閉じるテスト `TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted` を足した。e16177bf と b4215caf は self-review と verify の報告だけで、コードを変えていない。run-test.sh は changed scope のまま golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない。変異は `go test -overlay` で当て、scratch は scratchpad の下に置いた
- Evidence: `docs/evidence/test-2026-11-12-org-reseat.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。末尾の「pipeline run 4」の節に、run-test.sh の全出力(wrapper のログは `docs/evidence/verify-2026-11-15-104116.log`)、load average と空き容量、`(cached)` だった 5 パッケージの打ち直し、取り込んだ #328 のテストの名前での回、`-race` の回、coverage の回と変更行の突き合わせ、変異の置き換えと結果を足した
- 起動の仕方: run-test.sh は Bash ツールの 600 秒の上限を超えるので、ツールの `run_in_background` で起動し(開始・終了・終了コードを別のファイルに書かせる)、`caffeinate -w <pid>` を 175 秒ずつ繰り返して前景で待った。shell の段の SKIP は 3 回目と同じ 2 行で、signal を使う shell のテストは SKIP していない。空き容量は 52〜54 GiB で、ENOSPC は出ていない(`grep -ci 'no space left'` は 0)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` の shell の段(`tests/test-*.sh`) | 45 本 | 45 | 0 | 2 行(browser の検出、2.41 より古い実 git の場合。どちらも同じ script の中で別の手段が見る) | 約 10 分(19:41:16〜19:51 ごろ) |
| `./scripts/run-test.sh` の Go の段(`go test ./...`、golang の verifier の既定の `-timeout=30m`、`GOFLAGS` は空) | 10 パッケージ(テストの無い root を除く) | 10(うち gitroot、org/driver、org/protocol、scaffold、upgrade は `(cached)`) | 0 | `-v` なしなので数えていない | org 752.8 s、cli 659.4 s。run-test.sh 全体は 1,450 s、終了コード 0 |
| `(cached)` だった 5 パッケージの打ち直し(`go test -count=1`) | 5 パッケージ | 5 | 0 | — | 4 s |
| 取り込んだ #328 のテストを名前で(`go test -count=1 -v -run 'PasteMarker\|BracketedPaste\|SendTextForDriver\|CheckNoPasteMarkers\|ClaudeSeat_SendTextFails\|DryRun_ClaudeSeat'`、org と cli) | org 7 本(cli に当たる名前は無い) | 7 | 0 | 0 | 4 s |
| `go test -race -count=1 -v -run 'Reseat\|Disband' ./internal/org/ ./internal/cli/` | トップレベル org 100 本、cli 26 本(subtest を含めて 313) | 126 | 0 | 0 | org 6.3 s、cli 44.1 s(build を含めて 61 s)。`WARNING: DATA RACE` は 0 件 |
| coverage の回(`go test -count=1 -run 'Reseat\|Disband' -coverprofile`、org と cli) | 2 パッケージ | 2 | 0 | 0 | org 2.9 s、cli 72.9 s |
| 変異(`go test -overlay`、下の節) | 11 個 | 9 個が死んだ | 生き残り 2 個(Q5b、Q6) | 0 | 1 個 3〜51 s |

注: 3 回目と違い、run-test.sh の 1 回の実行で Go の 10 パッケージがすべて通った。時間切れは無く、打ち直しもしていない。load average は、19:41〜19:49 に uptime で見た 1 分の値が 10.9〜16.6、19:52 から 30 秒ごとに取った値が 4.9〜14.7 で、Go の段の後半は 5〜7 だった。`-race`、coverage、変異はどれも shell の段か Go の段の初めのうちに `nice -n 19` で回し、Go の段の後半とは重ねていない。#328 の 7 本は run-test.sh の org のパッケージでも通っている。

## cycle 3 の cross-review の直しとテスト

cross-review cycle 3 の指摘 2 件の直しを、どのテストが固定するかで並べる。「変異」の列は、そのテストが落としたこの回の変異の id(下の節)である。

| 直したこと | コミット | テスト(`internal/org/reseat_disband_test.go`、`reseat_start_test.go`、`reseat_test.go`。CLI は `internal/cli/org_reseat_test.go`) | 変異 |
| --- | --- | --- | --- |
| 止めてから立てるまでに別のコマンドが org を disband すると、reseat は確かめ直さずに座席を立て、結びつきの無い座席(`Spawn`)や、元の承認を持たない新しい命(`StartFeature`)を作っていた(指摘 1)。立てる直前に台帳を読み直し、計画のときの最後の `disbanded` と違えば、その座席から先を立てずに終わる。打ち直し・DECISION・read は出さず、`ralph org start --plan` から立て直すことを理由に書く | 84346015 | `TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart`(implementer だけの reseat。止めたあと disband され、立てない。エラーの全文、`LastDisbandedIndex` が -1、`spawn_started` が無いこと、`reseated` が `disbanded` のあとに入ること)、`TestReseat_AnOrgDisbandedAfterTheLeaderStarted`(leader を立てたあとに disband され、残りを立てない。エラーが leader を「disband の前に立てた」と名指しする)、`TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted`(f2ed6bdf。leader が対象でなく、implementer を立てたあと reviewer の前に disband される。DECISION と read を出さない)、`TestReseat_AManifestUnreadableBeforeAStart`(読み直しに失敗したら立てずに終わり、`Disbanded` にはしない)、`TestReseat_AnOrgDisbandedBeforeTheReseatStartsEverySeat`(reseat の前に disband され立て直された org では、計画が読んだ `disbanded` のままなので、ただ 1 席の leader を立てる)、`TestReseatDisbandedErr`(エラーの文の 5 行)。どのテストも共通の `assertDisbandedResult` で `Retry`・`Decision`・`DecisionRead`・`Changed` が空であることを見る | Q1、Q3、Q3b |
| 止める直前にも同じ確かめをする。一度の読み直しで disband を先に見るので、disband を「別のコマンドが座席を変えた」と取り違えない | c4f55aef | `TestReseat_AnOrgDisbandedBeforeItsFirstStop`(検査のあと最初の停止の前に disband される。何も止めず、`reseated` も書かない)、`TestReseat_AnOrgDisbandedBeforeItsSecondStop`(reviewer を止めたあと implementer の前に disband される。implementer と leader は止めず、`Changed` にしない)、`TestReseat_AManifestUnreadableBeforeAStop`(読み直しに失敗したら `Disbanded` にも `Changed` にもせず、打ち直しは全座席を名指しする) | Q2、Q3b |
| disband で終わった reseat の出力で、止めなかった座席を「left running」と書かない | b568ca8c | `TestOrgReseatLeftLine`(結果ごとの 9 行。`not stopped, disbanded` の行が新しい文を固定し、`stopped, disbanded` と `not started, disbanded` の行が disband でも文が変わらないことを固定する) | Q5(Q5b は生き残った。Test gaps 12) |
| DECISION の `ralph org send` と、その前の `ralph org read` は `--config`、無ければ `./ralph.toml` を読むので、作業ディレクトリの `ralph.toml` が壊れていると失敗していた(指摘 2)。両方に、絶対パスにした `--config` を付ける | ffd32109 | `TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir`(send と read の 4 行。どちらも無し、`--state-dir` だけ、`--config` だけ、空白を含む両方)、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`(`conf/ralph.toml` を渡した reseat の `Decision` と `DecisionRead`、打ち直しの `DecisionRead`)、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`(出力の read と send の頭を文字列で固定し、壊れた `ralph.toml` のある subdirectory から打つと、出力どおりの read と send は通り、`--config` なしの read は `load config` で落ちる)、CLI の `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`(`--config` を渡さなければ出力に `--config` が無い) | Q4、Q4b、Q4c、Q4d |
| help に disband で終わることと、read と send も `--config` を付けることを書き、古い「read no ralph.toml」の文を消す | 84346015、c4f55aef、ffd32109 | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(足した文を文ごと固定し、`read no ralph.toml` が無いことも見る) | 文なので当てていない |

## 受け入れ条件とテスト

どのテストも run-test.sh の shell の段か Go の段で PASS し、`Reseat` か `Disband` を名前に含むものは `-race` でも PASS した。3 回目から変わった行だけ、テストと変異を書き足した。それ以外の行のテストと、1〜3 回目の変異(1 回目の M1〜M9 と C1、2 回目の N1〜N5、N4b、N4c、N8a〜N8c、N9、3 回目の K1〜K3、X1、X2、H1)は 69908490、2d37fa32、c856fc27 の報告のとおりである。4 回目の直しはそれらの判定を消していない(`git diff 7ea546da..HEAD -- internal/org/reseat.go` で消えた行は、コメントのほかは、`reseatLifeSeats` の命の始まりの計算を `reseatLastDisbanded` に移した部分、`ReseatPlan` を作る 1 行、止める前の読み直しと座席の比較を `checkBeforeStop` に移した部分、`ReseatDecisionCommand` の引数とその呼び出しだけである)。

| AC | テスト | 変異 |
| --- | --- | --- |
| AC-1 3 席を止め、leader → implementer → reviewer の順に立てる | `TestReseat_StopsAndStartsEverySeatInOrder`、`TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_StopsAndStartsEverySeat` | — |
| AC-2 leader が新しい workspace の root の pane | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace` | — |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | `TestReseat_OneSeatOnAnotherDriver`、`TestPlanReseat_DriverAndModelForOneSeat`、CLI の `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`TestReseat_ASeatAnotherCommandStartedOnTheSameSettings`。4 回目で、DECISION の send と read が `--config` を付けること(`TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir` と上の 2 本)と、disband で終わったら DECISION を出さないこと(`TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted`)を足した | N3、K1〜K3、H1、Q3、Q4、Q4b、Q4c、Q4d |
| AC-4 止める前の断り | `TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestReseat_ARefusalWritesNothing`、`TestPlanReseat_GatesRunOnceAfterTheChecks` | 1 回目の M9 は生き残ったまま(Test gaps 1) |
| AC-5 結びつきの無い org の断り | `TestPlanReseat_RefusesAnOrgWithoutABinding`、CLI の `TestOrgReseat_OrgWithoutBindingRefused` | N4、N4c、N8a |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方。打ち直すと立つ | `TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`、`TestReseat_RetryAfterAFailedStart` の leader の subtest | N1、N2、N4、N4c、N8c |
| AC-7 座席の pane から打った reseat の断り | `TestPlanReseat_RefusedFromASeatPane` | — |
| AC-8 pane が無い座席の立て直し | `TestReseat_SeatsWhosePanesAreGone` | — |
| AC-9 `--dry-run` の出力 | `TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-10 `stop_failed` で何も立てない | `TestReseat_AStopThatFailsStartsNothing` | 1 回目の C1 と同じ変異 Q6 は、この回も生き残った(Test gaps 3) |
| AC-11 フラグの組み合わせの断り | `TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、CLI の `TestOrgReseat_FlagRefusals` | — |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | `TestReseat_RecordsItsIDOnEveryStopAndStart`、`TestReseat_AStopThatFailsStartsNothing`、`TestReseat_RetryAfterAFailedStart`、`TestReseatNoteAndReseatedID`。4 回目の disband のテストが、disband で終わった reseat の `reseated`(`implementer:stopped`、`leader:started,implementer:stopped,reviewer:stopped`、`leader:not_stopped,implementer:not_stopped,reviewer:stopped`)が `disbanded` のあとに入ることと、何も止めずに終わったときは `reseated` を書かないこと(`TestReseat_AnOrgDisbandedBeforeItsFirstStop`)を固定する | Q1、Q2 |
| AC-16 help が動詞とアカウントの移し方を言う | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(4 回目で disband の文と `--config` の文を直した)。skill の 4 つの写しの文を固定するテストは無い | — |
| AC-17 `reseat.lock` と止める直前の読み直し | `TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop`、3 回目の割り込みの 4 本。4 回目で、止める直前と立てる直前の disband の確かめ(上の表の disband のテスト 8 本)を足した | K1〜K3、X1、X2、H1、Q1、Q2 |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | `TestReseat_RetryAfterAFailedStart`、`TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife`、`TestReseatRetryCommand`、CLI の `TestOrgReseat_AFailedStartPrintsTheRetry`、`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`。4 回目で、disband で終わったら打ち直しを出さないこと(`assertDisbandedResult`)と、読み直しに失敗したときは全座席の打ち直しを出すこと(`TestReseat_AManifestUnreadableBeforeAStop`)を足した | N1、N4、N4b、N4c、N5、N9、X1、X2、Q3b |
| AC-19 `--dry-run` は見張りも server も起動せず、何も書かない | `TestPlanReseat_DryRunStartsNoWatchOrServer`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-20 切れた reseat の座席が既定の対象になる。signal と SIGPIPE で終わらない | `TestPlanReseat_FinishesAReseatThatWasCutOff`、CLI の `TestOrgReseat_ARealReseatRunsThroughTheSignalHold`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` | — |
| AC-21 独自の役割の座席の断り | `TestPlanReseat_RefusesACustomRoleSeat` | N4、N8b |

## 回帰

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 止めてから立てるまでに org が disband されても、reseat が座席を立てた(cross-review cycle 3 の指摘 1) | Pass | `TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart`、`TestReseat_AnOrgDisbandedAfterTheLeaderStarted`、`TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted`。Q1 の下では FAIL |
| 検査のあと止める前に org が disband されると、reseat はそれを「別のコマンドが座席を変えた」と報告した | Pass | `TestReseat_AnOrgDisbandedBeforeItsFirstStop`、`TestReseat_AnOrgDisbandedBeforeItsSecondStop`。Q2 の下では FAIL |
| `--config` を渡した reseat の DECISION の send と read が、作業ディレクトリの壊れた `ralph.toml` を読んで失敗した(指摘 2) | Pass | `TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`。Q4〜Q4d の下では FAIL |
| 1〜3 回目の回帰の項(打ち直しと DECISION と `--config`、`--seat` が受け付ける座席、`StartFeature`、spawn の事前の判定、Note が空の Stop と Spawn の Details、`reseated` を古い binary が数えないこと、立てる時点の idempotent の設定違い、`Changed` と打ち直し) | Pass | run-test.sh の Go の段の org と cli の全体 |
| 取り込んだ #328(`ralph org send` の bracketed paste) | Pass | 新しい 7 本(`TestSendTextForDriver_WrapsOnlyClaude`、`TestCheckNoPasteMarkers_RefusesEitherMarkerAnywhere`、`TestOrgSend_PasteMarker_RefusedWithNothingSent`、`TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude`、`TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers`、`TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste`、`TestOrgSend_ClaudeSeat_SendTextFails_NoEnter`、どれも `internal/org`)は名前での `-v` の回で PASS し、run-test.sh の org のパッケージでも通った |

## Coverage

- Statement: 4 回目の直し 4 つが足した行を、HEAD の行番号で取った(`git diff -U0 7ea546da HEAD -- internal/org/reseat.go internal/cli/org_reseat.go`。この範囲でこの 2 ファイルを変えたのは ffd32109、84346015、c4f55aef、b568ca8c だけで、f2ed6bdf はテストだけ、merge 46dfc5a8 は範囲の外)。`-run 'Reseat|Disband'` で絞った org と cli の回の cover profile と突き合わせると(`addedcov.py`)、`reseat.go` は cover profile の block に入る足した行 92 行のうち 91 行、`org_reseat.go` は 4 行すべてを通った。通らなかったのは `reseat.go:1352`、`reseatSeatNow` の `return nil, false`(読み直した台帳の今の命に座席の状態の event が無いとき)である。c4f55aef で `checkBeforeStop` が disband を先に見るようになったので、ここに来るのは台帳を手で書き換えたときだけになった(コードのコメントもそう書く)。そのとき出る `reseatChangedErr` の「no state event since the org's last disband」の文を含むテストも無い(Test gaps 14)
- Branch: Go の標準ツールに branch coverage は無いので測っていない。`reseatDisbandedIn` の「変わった」と「変わらない」の両側は、disband のテストと `TestReseat_AnOrgDisbandedBeforeTheReseatStartsEverySeat` がそれぞれ通る。`reseatDisbandedErr` の started が 0・1・2 席の枝と、時刻の無い `disbanded` は `TestReseatDisbandedErr` の表が通る
- Notes: 名前で絞った回なので、パッケージ全体の数字(org 30.7%、cli 11.9%)は意味を持たない

## 変異

4 回目の直しが足した判定と配線に 10 個、1 回目の C1 の当て直しに 1 個の変異を当てた。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れ、置き換えたあとのファイルが元と違うことを確かめてから当てた(scratch の `mut.pl`)。`internal/org/reseat.go` の変異は `go test -count=1 -vet=off -v -overlay <mutant> -run 'Reseat|Disband' ./internal/org/`、`internal/cli/org_reseat.go` の変異と、Q4 と Q4b の CLI 側は `-run 'OrgReseat' ./internal/cli/` で回した。build に失敗した変異は無い(`build failed` と `no space left` は 0 件)。変異を回す前とあとで `git status --porcelain --ignored`(`.harness/cache` を除く)は同じだった。id は 1〜3 回目の M、C、N、K、X、H と重ならない Q にした。

| id | 変異 | 結果(落ちたテスト) | 時間 |
| --- | --- | --- | --- |
| Q1 | `startAll` の `if r.endIfDisbanded(i)` を `if false && r.endIfDisbanded(i)` にする(立てる前の確かめを呼ばない) | 死んだ。`TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart`、`TestReseat_AnOrgDisbandedAfterTheLeaderStarted`、`TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted`、`TestReseat_AManifestUnreadableBeforeAStart` | 9 s |
| Q2 | `checkBeforeStop` の disband の確かめを `false && changed` で止める | 死んだ。`TestReseat_AnOrgDisbandedBeforeItsSecondStop`、`TestReseat_AnOrgDisbandedBeforeItsFirstStop` | 9 s |
| Q3 | `finish` で、disband のときは `Retry` だけを空のままにし、`Decision` と `DecisionRead` は作る(`if r.res.Disbanded { return r.res }` を消し、`Retry` の行を `if !r.res.Disbanded { ... }` で囲む) | 死んだ。`TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted` だけ | 3 s |
| Q3b | `finish` の disband の `return` を `Retry` の行のあとに移す(`Retry` を作り、DECISION の前に返る) | 死んだ。`OneSeatOfAnOrgDisbandedBeforeItsStart`、`AnOrgDisbandedAfterTheLeaderStarted`、`NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted`、`AnOrgDisbandedBeforeItsSecondStop`、`AnOrgDisbandedBeforeItsFirstStop`(どれも `TestReseat_`) | 3 s |
| Q4 | `ReseatDecisionCommand` が `reseatVerbCommand` に設定のヒントを `""` で渡す | 死んだ。org の `TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir`(`the config only`、`both, quoted`)、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`。cli の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` | org 8 s、cli 31 s |
| Q4b | `reseatLeaderReadCommand` が同じく `""` を渡す | 死んだ。Q4 と同じ org の 2 本と cli の 1 本 | org 9 s、cli 51 s |
| Q4c | `finish` の `ReseatDecisionCommand` の呼び出しに `r.hintConfigPath` の代わりに `""` を渡す | 死んだ。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure` | 8 s |
| Q4d | `finish` の `reseatLeaderReadCommand` の呼び出しに同じく `""` を渡す | 死んだ。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure` | 9 s |
| Q5 | `orgReseatLeftLine` で disband の case を `left running` の case のあとに移す(disband の文に届かなくなる) | 死んだ。`TestOrgReseatLeftLine/not_stopped,_disbanded` | 33 s |
| Q5b | `printOrgReseatResult` の `orgReseatLeftLine` の呼び出しで `res.Disbanded` の代わりに `false` を渡す | 生き残った。`-run 'OrgReseat'` の 13 本がすべて PASS | 31 s |
| Q6 | 1 回目の C1 と同じ置き換え: 止める失敗の行の `verb = "stop failed"` を `"stopped"` にする | 生き残った。`-run 'OrgReseat'` の 13 本がすべて PASS | 35 s |

Q3 は、ハンドオフに書かれた形(disband のとき `Retry` は空のまま、DECISION と read は作る)で、f2ed6bdf のテスト 1 本だけが落とした。ほかの disband のテストは leader が対象か、leader のほかに立てた座席が無いので、DECISION が作られない。self-review の L23 の指摘(このテストが無いと、この変異はどのテストにも落とされない)はこの回でも成り立つ。一方、`return` を `Retry` の行のあとに移す形(Q3b)は `Retry` が空でなくなるので、`assertDisbandedResult` を通る disband のテスト 5 本が落とす。

Q5b と Q6 はパッケージ全体では回していない。理由は 2 つある。`res.Disbanded` が真になるのは、計画と停止・起動のあいだに disband が入ったときだけで、その割り込みは `internal/org` の中のテスト用のフックでしか起こせず、cli のテストからは届かない。また、`internal/cli` のテストのうち `the org was disbanded meanwhile` と `not stopped: ` を含むのは `TestOrgReseatLeftLine` だけである(`grep`)。Q6 の `stop failed` の行も、1 回目の報告のとおり `reseat` を打つ cli のテストは `org_reseat_test.go` だけで、止める失敗を起こすテストはその中に無い。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh の shell の段と Go の段、`(cached)` の打ち直し、#328 のテストの回、`-race` の回、coverage の回で落ちたテストは無い。時間切れも無かった。変異の FAIL は、上の節の意図した落ち方である。

## Test gaps

3 回目の Test gaps のうち、4(検査のあと止める前の disband)と 9(run-test.sh の 1 回の実行で Go の全パッケージが通った記録が無い)は、この回で閉じた。4 は `TestReseat_AnOrgDisbandedBeforeItsFirstStop` と `TestReseat_AnOrgDisbandedBeforeItsSecondStop` が固定し(Q2)、9 はこの回の run-test.sh が 10 パッケージとも通した(5 つは `(cached)` で、`-count=1` の打ち直しも通った)。番号は 3 回目のものを引き継ぎ、閉じた番号は空けておく。1〜3 は `docs/tech-debt/README.md` の reseat の Test gaps の行に載っている。

1. 非 leader の reseat の worktree の branch の検査(`reseatSeatsStartErr` の `checkFeatureWorktreeReuse`)を、leader を含まない reseat で通るテストが無い(1 回目の M9)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` の「the worktree has another branch checked out」は既定の対象(leader を含む)の行である
2. Note が空の spawn の Details を完全一致で固定するテストが無い(1 回目の M68)
3. CLI の `stop failed:` の行を出すテストが無い(1 回目の C1。この回の Q6 も生き残った)。`left stopped:`、`left as it is:`、`left running:`、`not started:`、`not stopped:` の文は、b568ca8c の `TestOrgReseatLeftLine` が関数の単体の表で固定するようになったが、`printOrgReseatResult` がそれらを出す流れを通す cli のテストは、止める失敗の形には無い
4. (閉じた)
5. I/O の失敗の戻りは行の coverage に入らない(`org_reseat.go` の `filepath.Abs` の失敗を返す行など)
6. `dryRunSpawn` の note(1 回目の M4)は届かない行で、固定されていない
7. AC-12 の実機は `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、4 回目でも繰り返していない。実機の binary は dd204370 で、1 回目の P9・P10、SIGPIPE の直し、2〜4 回目の直しより前である。disband で終わる reseat と、`--config` を付けた DECISION の send と read を実機で打った記録は無い
8. AC-16 の skill の 4 つの写しの文はテストで固定していない
9. (閉じた)
10. 立てる時点の検出を reseat を通して確かめるテストは、どれも model の違う割り込みである。role だけ、driver だけが違う割り込みは `TestReseatIdempotentMismatchErr` の表だけが見る(3 回目の K1、K2)
11. 割り込みは `reseatTestHooks.beforeStart` と `beforeStopCheck` のフックで、reseat と同じ goroutine の中で 1 回だけ起こしている。4 回目の disband も、本物の `Disband` を fake の driver で走らせるが、同じ goroutine の中である。別のプロセスが本当に並んで走る窓は再現しておらず、`-race` の回もこの窓を通っていない。読み直しから `Stop`・`Spawn`・`StartFeature` の書き込みまでの窓は、cross-review cycle 3 の triage のとおり残っている(`Disband` は `reseat.lock` を取らない)
12. 立てる時点で終わった reseat と、disband で終わった reseat の CLI の出力を見る cli のテストは無い。フックが `internal/org` の中にあり、cli のテストからは割り込めない。`printOrgReseatResult` が `res.Disbanded` を `orgReseatLeftLine` に渡す配線は、どのテストも固定していない(Q5b が生き残った)。`Retry`、`Changed`、`Disbanded`、`Err` の値は org のテストが固定し、行の文は `TestOrgReseatLeftLine` が固定する
13. help の文は、文ごとの部分一致で固定している。文の中の語を変える変異は、3 回目の 2 文にも、4 回目に直した disband の文と `--config` の文にも当てていない
14. `reseatSeatNow` が座席を見つけない枝(`reseat.go:1352`)と、そのとき `reseatChangedErr` が出す「no state event since the org's last disband」の文を通るテストが無い。c4f55aef のあと、ここに来るのは台帳を手で書き換えたときだけである

## Verdict

- Pass: run-test.sh(shell の段 45 本、Go の段 10 パッケージ)が 1 回の実行で通り、`(cached)` の 5 パッケージの打ち直し、取り込んだ #328 の 7 本、`-race` の回、coverage の回もすべて通った。cross-review cycle 3 の指摘 2 件の直しはどれもテストが固定し(help は文の pin)、ハンドオフの 5 個の変異(Q1〜Q5)を含む直しの変異 9 個が死んだ。生き残った Q5b(CLI の配線)と Q6(1 回目の C1 の当て直し)は、cli のテストから disband と止める失敗を起こせないことによる穴で、Test gaps 3 と 12 に書いた。足した行のうち cover profile の block に入る 96 行のうち 95 行を通った
- Fail: 0
- Blocked: なし
