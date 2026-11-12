# Test report: send-screen-fallback

- Date: 2026-11-12(数える回の run-test.sh は 22:08:48 から 22:33:06 JST。追加の回は 22:33 から 22:45 JST)
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(`## Test plan`、AC1〜AC11。issue #308)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(origin/main `d2e5c6e9`、HEAD `88742506`、71 ファイル、+6146/-176)。run-test.sh は changed scope を求めたが、testdata の `.txt` がどの言語にも分けられないので full に落ち(`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-folder-trust-unnumbered.txt)`)、選ばれた pack は golang だけだった。worktree の production code とテストには手を入れていない。変異は `git archive HEAD` の写しに `go test -overlay` で当てた
- Evidence: `docs/evidence/test-2026-11-12-send-screen-fallback.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。数える回の run-test.sh の全出力、使えなかった 1 回目の要約、変異の harness と定義と結果、coverage、`-count=10` と `-race` の回を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-12-130848.log`
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った
- 打ち直し: 1 回目の run-test.sh は、マシンのディスクが一杯になって(空き 136 MiB)Go の段で `no space left on device` を出し、`internal/cli` と `internal/org` が build できなかった。ほかの session が空きを作ったあと(33 GiB)、指示どおり 1 回だけ打ち直した。下の数字は 2 回目のもの。同じ時に回していた変異の 1 回目の束(M01〜M14)も、すべて同じ理由の `[build failed]` だったので捨て、打ち直した

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 2 回目(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,458 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 790.0 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 629.5 秒 |
| └ Go: `internal/config` | パッケージ全体 | ok | 0 | — | 11.6 秒 |
| └ Go: `cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | パッケージ全体 | ok。`(cached)` で、同じ tree の 1 回目に通った結果 | 0 | — | 1 回目で 0.9、2.2、2.1、3.4、1.0、2.5、2.4 秒 |
| `./scripts/run-test.sh` 1 回目(数えない) | — | — | ENOSPC。cli と org は `[build failed]`、config は `t.TempDir` の mkdir が失敗 | — | 564 秒 |
| `-count=10 -v`、org: `TestOrgSendScreen_` の 13 本と `TestLiveSeatStates_EachCallEndsAtTheDriverCallTimeout` | 上位 140 回(subtest 込みの PASS 行 340) | 340 | 0 | 0 | 204 秒。load 5〜9 |
| `-race -count=2 -v`、org: この diff が足した 34 本(下の正規表現) | 上位 68 回(PASS 行 436) | 436 | 0 | 0 | 57 秒。`DATA RACE` 0 |
| coverage: org `-run 'Send\|Screen\|Live\|Resume\|PaneLock'` | 名前で絞った回 | ok | 0 | — | 44.5 秒 |
| coverage: cli `-run 'TestOrgSend\|TestOrgStatus\|TestOrgResume'` | 名前で絞った回 | ok | 0 | — | 87 秒 |
| 変異: production code の 7 個(下の節) | 7 | 6 個が死んだ | — | — | 1 個 9〜45 秒 |

`-race` の回の正規表現は `TestOrgSendScreen_|TestLiveSeatStates|TestLiveSeatState_|TestClassifySendScreen|TestSendInputHolds|TestSendScreen|TestSendProbe|TestFindResumeInputBox_Codex0166Footer|TestResumeCodexFooterLine` である。この diff が org に足したテスト関数 34 本(`send_fallback_test.go` 17、`send_screen_test.go` 8、`status_live_test.go` 7、`resume_screens_test.go` 2)をすべて含む。cli に足した 10 本(`org_send_screen_fallback_test.go` 5、`org_status_live_test.go` 5)は run-test.sh と coverage の回で通した。

Skipped の 2 行は前の PR の回と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

2 回目の run-test.sh のあいだ、1 分の load は 6 回の見本で 7 から 13 だった。org は 790 秒で、30 分の上限の 44% である。2 回目で落ちたテストは無い。

## 受け入れ条件とテスト

どのテストも 2 回目の run-test.sh で PASS した。org のテストは `-race` の回でも PASS した。「変異」の列は、そのテストが落とした変異の id(下の節)である。

| AC | テスト(ファイル) | 変異 |
| --- | --- | --- |
| AC1 herdr が working のまま、画面が入力待ちの座席へ打ち、`sent` に `via=screen` | `TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen`(claude、codex。herdr の呼び出しの列、`via=screen herdr_state=working`、`--timeout-ms` のうちに終わること、ロックの解放)、`_AgentGetHangs_EndsAtTheDriverCallTimeout`、`_MultiLineMessageTallerThanTheBox_EnterPressed`(`send_fallback_test.go`)。cli の `TestOrgSendScreenPath_ScreenWaitsForInput_TypesAndRecordsViaScreen`(binary の道を 24000 ms で通し、exit 0、呼び出しの列、`sent` の Details) | 当てていない(M18 を定義したが、上限のため回していない) |
| AC2 入力待ちでない画面では打たず、エラーに herdr の状態・判定と理由・次の手 | `TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy` の 7 行(2 回の読みが違う、busy、2 回目が busy、choice、終わりの行が無い、読みの失敗、`agent get` の失敗)。cli の `TestOrgSendScreenPath_ScreenBusy_TypesNothingAndSaysWhy`(exit 1、`--state-dir` を付けた次の手) | M01 が死んだ。M02 は生き残った(Test gaps 1) |
| AC3 ロックの下の読み直しで入力待ちでなければ打たない | `TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing`(busy になった画面、読みの失敗) | 当てていない |
| AC4 Enter の前に文が入力欄に無ければ Enter を押さない | `TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed`、`TestSendScreen_InputHoldsAMultiLineProtocolMessage`、`TestSendInputHolds_TypedMultiLineFixtures`、`_Refuses`、`_ClaudePasteBoxes`、通る側の `_MultiLineMessageTallerThanTheBox_EnterPressed` | M03 が死んだ |
| AC10 画面の道の Enter のあとは画面で確かめ、文が残れば `submit_unconfirmed=true` | `TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly`(残ったまま、`raw=true` の並び、2 回目の読みで空になる)、`TestSendScreenSubmitted` | M04 が死んだ |
| AC11 確かめのあと、残りの時間で herdr の待ちに戻る | `TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath`、`_PaneReadHangs_HerdrsWaitGetsTheRest`(done が来る場合と来ない場合)、`_HerdrDoneAtTheCheck_ScreenNotRead` | M06(4 本が落とした)、M07 が死んだ |
| AC5 `--no-screen-fallback` と、idle か done の道 | `TestOrgSendScreen_NoScreenFallback_HerdrOnly`、`_HerdrIdleAtOnce_ScreenNotRead`、`_ReserveTooShortForTheCheck_HerdrOnly`、`TestSendScreenReserve_Table`(20250 で 6.75 秒、20249 で 0)。cli の `TestOrgSend_NoScreenFallbackFlag_ReachesSend`、`TestOrgSendScreenPath_NoScreenFallback_ReadsNoScreen` | 当てていない |
| AC6 `status --live` の列、食い違いの印、`--live` なしは今と同じ | `TestLiveSeatStates_*` の 6 本と `TestLiveSeatState_TextOfTheColumns`。cli の `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`、`TestOrgStatusLive_TableShowsHerdrAndTheScreenAndTheMismatch`、`_JSONAddsALiveObjectPerActiveSeat`、`_NoHerdrReachableStillExitsZero`、`TestOrgStatus_HelpDescribesLive` | K01 が死んだ |
| AC7 codex 0.166 の footer で入力欄が見つかり、既存の fixture の判定は変わらない | `TestFindResumeInputBox_Codex0166Footer`、`TestResumeCodexFooterLine`、`TestClassifyResumeScreen_Fixtures` と `TestResumeInputHolds` に足した行 | 当てていない(Test gaps 6) |
| AC8 入力待ちの判定が、採った画面と既存の testdata のすべてで分かれる | `TestClassifySendScreen_EveryFixture`(61 の fixture すべての kind、reason、line。表に無い fixture と、名前の `-idle-` や `-working-` に合わない kind も落とす)、`TestClassifySendScreen_Edges`、`TestSendScreenBoxCheck_ChoiceDialogWithNoMarkerLine` | 当てていない |
| AC9 help、skill、recipe、AGENTS.md | `TestOrgSend_HelpDescribesTheScreenFallback`、`TestOrgStatus_HelpDescribesLive`。skill、recipe、AGENTS.md の文はテストで固定していない。/verify が読んで確かめた | — |

## 計画の Test plan の項目とテスト

| 項目 | 状態 | テスト |
| --- | --- | --- |
| 入力待ちの判定の表(起動直後、入力待ち、考え中、ツール、折り返した動作中の行、積まれた文、送った直後、再試行中、通信待ち、選択の画面、hook の確認、自動の続行の待ち、入力欄に文がある、終わりの行が無い、判断できない) | ある | `TestClassifySendScreen_EveryFixture` の 61 行と `_Edges`。claude の終わりの行と動作中の行の取り違えは、`_Edges` の「✻ word… under a prompt is busy」「✻ word for 3s · done under a reply is idle」「turn-end line with the time taken only is not a turn end」の 3 行が固定している |
| 入力欄の見つけ方(codex 0.166 の footer、既存の判定は変わらない) | ある | AC7 の行 |
| `Send` の画面の道(打つ、打たない、ロックの下、Enter の前、Enter のあと、herdr の待ちに戻る、フラグ、herdr が idle) | ある | AC1〜AC5、AC10、AC11 の行 |
| 時間(最初の待ちは `--timeout-ms` から確かめの分を引いた長さ、全体は `--timeout-ms` を超えない) | ある | 各 `Send` のテストが `waitTimeouts` と経過時間を見る。`TestSendScreenReserve_Table`。cli では最初の待ちの `--timeout 16000`、`12200`、`18300` を呼び出しの記録で見る |
| `status --live`(列、食い違いの印と案内、herdr に届かない座席) | ある | AC6 の行。`_NoHerdrReachableStillExitsZero` と `TestLiveSeatStates_HerdrStateAgainstTheScreen` の失敗の行 |
| Integration(CLI の `send` と `status --live` を偽の herdr で通す) | ある | cli の 10 本。画面の道の 3 本は 88742506 で足された。読みのあいだの 3 秒を CLI から縮められないので、1 本 18〜20 秒かかる |
| Regression(既存の send、pane ロック、resume、承認の確かめ) | 通った | 下の Regression checks |
| Edge: `--timeout-ms` が短い | ある | `_ReserveTooShortForTheCheck_HerdrOnly`、`TestSendScreenReserve_Table` |
| Edge: pane が読めない | ある | 確かめの中、ロックの下、Enter の前の 3 か所で、それぞれ「the read fails」の行がある。`status --live` にもある |
| Edge: 空の画面 | 判定だけ | `TestClassifySendScreen_Edges` の「empty screen」(unknown、no input box)。`Send` では回していないが、ほかの unknown の画面と同じ道を通る |
| Edge: herdr が blocked を返す | `status --live` だけ | `TestLiveSeatStates_HerdrStateAgainstTheScreen` の「blocked and a choice dialog」。`Send` の確かめで herdr が blocked を返す場合のテストは無い(Test gaps 4) |

## Coverage

- Statement: この diff が変えた Go のコード(テストを除く)のうち、cover profile の block に入る足した行は 523 行で、516 行を通った(`git diff -U0 origin/main...HEAD` の足した行と、上の名前で絞った 2 つの回の profile を突き合わせた)。`send_fallback.go` 156/163、`send_screen.go` 207/207、`status_live.go` 49/49、`verbs.go` 28/28、`resume.go` 1/1、`internal/cli/org.go` 75/75。`resume_screens.go` の変更は package の変数の正規表現 1 行で、block に入らない。名前で絞った回なので、パッケージ全体の数字(org 35.8%、cli 11.2%)は意味を持たない
- 通っていない 7 行(すべて `send_fallback.go`):
  - 199〜201: 確かめの時間が 2 回の読みのあいだで切れた場合(`the screen check ran out of time between its two reads`)。self-review の L10 が「最小に近い reserve では起きうる」とした道である(Test gaps 2)
  - 241〜242: herdr の client が `AgentGet` を持たない場合
  - 254〜255: `agent get` の返事に状態が無い場合
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: `send_screen.go` と `status_live.go` の関数はすべて 100%。`send_fallback.go` は `checkSendScreen` 88.9%、`sendHerdrState` 85.7% で、ほかは 100%。`Send` は 97.6% で、通らないのはこの diff が変えていない行である(上の足した行の突き合わせで `verbs.go` は 28/28)
- Notes: 行はほぼすべて通るが、2 回の読みのあいだに待つことは行の coverage では見えない穴で、変異 M02 で見つけた(Test gaps 1)

## 変異

orchestrator の指示(1 個ごとに build cache が増えるので、変異は 6 個ほどに)に合わせ、ユーザーが名前を挙げた guard に 1 個ずつ当てた。M02 は指示が届く前に回し終えていたので、結果に含める。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた。org の変異は `-run 'Send|Screen|Live|Resume'`、K01 は `-run 'TestOrgStatus'` で絞って回した。定義、harness、出力は evidence にある。

| 変異 | guard | 置き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- | --- |
| M01 | 2 回の読みが同じであること | `send_fallback.go` の `case second != first:` を `case false && second != first:` に | 死んだ | `TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy/the_two_reads_differ` |
| M02 | 2 回の読みのあいだの待ち | `waitOrCtxDone(ctx, gap)` を `waitOrCtxDone(ctx, 0)` に | 生き残った | なし |
| M03 | Enter の前の `sendInputHolds` | `if !sendInputHolds(...)` を `if false && !sendInputHolds(...)` に | 死んだ | `TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed/the_input_box_is_empty` |
| M04 | Enter のあとの画面の確かめ | `err == nil && SendScreenSubmitted(...)` を `err == nil \|\| SendScreenSubmitted(...)` に | 死んだ | `TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly` の 3 つの subtest |
| M06 | 確かめのあと herdr の待ちに戻ること | 戻りの `AgentWait` を外し、すぐ `context.DeadlineExceeded` のエラーにする | 死んだ | `_ScreenNotWaiting_TypesNothingAndSaysWhy`、`_ScreenBusyThenHerdrDone_SendsByHerdrsPath`、`_HerdrDoneAtTheCheck_ScreenNotRead`、`_PaneReadHangs_HerdrsWaitGetsTheRest` |
| M07 | 確かめを reserve の半分で区切ること | `context.WithTimeout(ctx, reserve/2)` を `reserve` に | 死んだ | `_PaneReadHangs_HerdrsWaitGetsTheRest` の 2 つの subtest(`AgentWait timeouts = [800 1]`) |
| K01 | `--live` なしの出力は今と同じ | `internal/cli/org.go` の `if live {` を `if true \|\| live {` に | 死んだ | `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`、`TestOrgStatus_EmptyModelShowsCLIDefault` |

M02 が生き残った理由: 偽の herdr(`sendFallbackHerdr`)は、呼ばれた順に画面を返し、呼ばれた時刻を持たない。どのテストも、2 回目の読みが 1 回目の読みから `SendScreenReadGap` 以上あとであることを見ていない。cli の `_ScreenWaitsForInput_` は経過時間の上限だけを見る。確かめは、最初の待ちが自分の期限で切れたときだけ走る。`SendScreenReserve` か `SendScreenReadGap` を設定するテストは `send_fallback_test.go` だけで(`git grep`)、ほかに確かめに届くのは cli の画面の道の 3 本である。`TimeoutMS` に 10000 を渡す既存の `Send` のテストは reserve が 0 になる。そのため、おそらく cli とパッケージ全体でも生き残る。build を増やさないため、この 2 つの回では確かめていない。未確認です。

本番で待ちが無くなると、2 回の読みは数ミリ秒の間隔になる。計画の D1 と D4 が頼る「動作中の画面は 3 秒あけた 2 回の読みで必ず違った」(probe の evidence)が効かなくなり、動作中の行を出さない一瞬の画面を入力待ちと読む余地が広がる。動作中の行、終わりの行、ロックの下の読み直し、Enter の前の確かめは残るので、それだけで誤送信になるわけではない。

定義したが回していない変異が 57 個ある(M05、M08〜M24、C01〜C23、F01〜F03 と F06、L01〜L07、K02〜K06。64 個の定義はすべて evidence の末尾に写した)。これらの guard がテストで固定されているかは、テストを読んで出した見込みにとどまる。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| run-test.sh 1 回目の `internal/cli`、`internal/org`(`[build failed]`)、`internal/config` の `TestLoad_ScriptLocalOnlyKeysInRalphToml`、`TestLoad_LocalScriptKeys`、`TestLoad_LocalScriptValueValidation` ほか | `link: mapping output file failed: no space left on device`、`compile: writing output: ... no space left on device`、`TempDir: mkdir ...: no space left on device` | 環境。データの volume の空きが 136 MiB になっていた(`df -h /System/Volumes/Data`)。ほかの session が空きを作り、数分後に 33 GiB になった | 直すコードは無い。1 回だけ打ち直し、2 回目はすべて通った |
| 変異の 1 回目の束(M01〜M14)がすべて `[build failed]` | 同じ ENOSPC(`go test` の出力には `[build failed]` しか出ず、直接回して `compile: writing output: ... no space left on device` を見た) | 同じ | 捨てて、空きが戻ってから回し直した。数えたのは 2 回目の結果だけ |

2 回目の run-test.sh と、追加の回(coverage 2 つ、`-count=10`、`-race`)で落ちたテストは無い。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の send、pane ロック、resume、承認の確かめのテスト(計画の Regression tests) | 通った | `internal/org` と `internal/cli` のパッケージ全体を 2 回目の run-test.sh で通した。この diff は既存のテストの期待を消していない。cli の `org_resume_test.go` の偽の herdr は、`agent wait` を時間切れまで待たせる `keeps-working-` と、Enter のあとの画面の `entered-` の分岐を足しただけで、元の `typed-` の分岐は残っている |
| herdr がすぐ idle か done を返す道(画面を読まない) | 通った | `TestOrgSendScreen_HerdrIdleAtOnce_ScreenNotRead`、cli の `TestOrgSend_NoScreenFallbackFlag_ReachesSend`(pane read と agent get が 1 回も無いこと)。違いは最初の待ちの `--timeout` が `--timeout-ms` から reserve を引いた値になることだけで、計画どおり |
| `ralph org status` の `--live` なしの出力 | 通った | `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`。K01 で、この test が出力の変化を捕まえることを確かめた |
| codex 0.165 の画面の resume の判定 | 通った | `TestClassifyResumeScreen_Fixtures` と `TestResumeInputHolds` の既存の行はそのまま通り、足した行は 0.166 の画面だけ |
| 時間に頼るテストの余裕(self-review と verify が挙げた `PaneReadHangs_HerdrsWaitGetsTheRest` の約 100 ms) | 通った | `-count=10` で 10/10(load 5〜9)、run-test.sh のパッケージ全体の回(load 7〜13)でも通った。それより重い load では試していない |

## Test gaps

1. 2 回の読みのあいだに `SendScreenReadGap` だけ待つことを、どのテストも固定していない(M02 が生き残った)。いちばん小さい足し方は、`sendFallbackHerdr` に `PaneRead` が呼ばれた時刻を残し、`TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen` で 2 回目と 1 回目の差が `o.SendScreenReadGap`(20 ms)以上であることを見ることである。おそらくこれで M02 は落ちる。overlay の probe では確かめていない
2. 確かめの時間が 2 回の読みのあいだで切れる道(`send_fallback.go:199〜201`)を通るテストが無い。self-review の L10 が挙げた境目(reserve が最小に近いと、`agent get` と 1 回目の読みに使える時間は Enter の前の待ちの半分、既定で 375 ms)で起きる道で、エラーの文と、そのあと herdr の待ちに戻ることが固定されていない。足すなら、`sendFallbackOrg` に `SendScreenReadGap` 100 ms、`DriverCallTimeout` 100 ms、`blockGet` を足し(確かめは reserve 300 ms の半分の 150 ms。`agent get` が 100 ms で切れ、1 回目の読みのあとの 100 ms の待ちの途中で確かめが切れる)、verdict が `the screen check ran out of time between its two reads` になり、残りが herdr の待ちに渡ることを見る行がいちばん小さい。この数字は読んで出したもので、回していない
3. `sendHerdrState` が `unknown` を返す 2 つの道(`AgentGet` を持たない herdr の client、状態の無い返事)を `Send` の側で通すテストが無い(`send_fallback.go:241〜242`、`254〜255`)。本番の herdr の client は `AgentGet` を持つので、影響は小さい
4. 計画の Edge cases の「herdr が blocked を返す」は、`status --live` にしか行が無い。`Send` の確かめで herdr が blocked を返すと、画面は読まれ、入力待ちなら打って `herdr_state=blocked` を残す。この道のテストは無い
5. 実機の herdr、claude、codex では画面の道を通していない(/verify の Coverage gaps と同じ)。既定の `--timeout-ms` 30000 で画面の道を通すテストも無い。cli の 3 本は 24000 と 18300 で、30000 の回は herdr がすぐ idle を返す場合だけである
6. 変異は 7 個に絞ったので、AC3(ロックの下の読み直し)、AC5(フラグ、driver、reserve の算術)、AC7(footer の正規表現)、AC8(判定の規則)、AC6 の読む座席の選び方の guard には当てていない。とくに /verify の V2 が「`/test` の mutant で確かめる」とした footer の正規表現の錨(F2、F3、F6)のどれが赤になるかは、確かめていない
7. 時間に頼るテストの余裕は、load 13 までしか見ていない。`PaneReadHangs_HerdrsWaitGetsTheRest` の 2 回目の待ちの下限 100 ms は、CI でもっと重い load になれば落ちうる。おそらく `got[1] >= 1` に緩めても契約は守れる(self-review の観察と同じ)。未確認です

## Verdict

- Pass: 2 回目の run-test.sh(shell 45 本、Go 10 パッケージ)、`-count=10` と `-race -count=2` の回、coverage の 2 つの回のすべてが通った。AC1〜AC8、AC10、AC11 の挙動はテストが固定し、足した行のうち block に入る 523 行の 516 行を通った。名前を挙げられた guard の変異 7 個のうち 6 個が死んだ。生き残った M02(2 回の読みのあいだの待ち)は Test gaps 1 に書いた。AC9 は help のテストと /verify の文書の確かめ
- Fail: 0
- Blocked: なし
