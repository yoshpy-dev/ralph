# Test report: send-screen-fallback

- Date: 2026-11-13(run-test.sh は 00:13:48 から 00:40:25 JST。変異、`-count=30`、coverage、probe は同じ時間帯に回した)
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(`## Test plan`、AC1〜AC11。issue #308)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2。`cycle-count.json` は `{"cycle":2}`)
- Scope: `git diff origin/main...HEAD`(origin/main `0936aaea`、HEAD `89585818`、78 ファイル、+6913/-212)。cycle 1 の test(`d7086a68`)のあとの変更は、f24290b5(番号付きの入力欄の行と選択の画面を Enter の前に見分ける `sendTypedInputBox`、`findResumeInputBox` の分割、fixture `*-input-typed-numbered-steps.txt` 2 つ、M02 のテスト)、cea96ea4(`resumeMarkerLineBox` の戻り値の確かめ)、origin/main の取り込み(39004c89)である。run-test.sh は changed scope を求めたが、testdata の `.txt` がどの言語にも分けられないので full に落ち(`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-folder-trust-unnumbered.txt)`)、選ばれた pack は golang だけだった。worktree の production code とテストには手を入れていない。変異と probe は `git archive HEAD` の写しに `go test -overlay` で当てた
- Evidence: `docs/evidence/test-2026-11-12-send-screen-fallback.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、変異の harness と定義と結果、coverage、`-count=30` の回、probe を入れた。cycle 1 の evidence は `docs/evidence/test-2026-11-12-send-screen-fallback.cycle1.log` に移した。wrapper 自身のログは `docs/evidence/verify-2026-11-12-151348.log`
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った
- 打ち直し: なし。ディスクの空きは run-test.sh の前後で 69 GiB と 67 GiB で、`no space left` は run-test.sh の出力にも変異の出力にも 0 行だった。build の失敗も無かった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,597 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 1,007.0 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 759.6 秒 |
| └ Go: `internal/config`、`internal/org/driver`、`internal/scaffold`、`internal/insights`、`cmd/ralph` | パッケージ全体 | ok | 0 | — | 13.4、3.7、3.3、1.9、0.7 秒 |
| └ Go: `internal/gitroot`、`internal/org/protocol`、`internal/upgrade` | パッケージ全体 | ok。`(cached)` で、同じ入力で前に通った結果 | 0 | — | — |
| 変異の対照: 写しを overlay なしで `-run 'Send\|Screen\|Live\|Resume\|PaneLock'` | 名前で絞った回 | ok | 0 | — | 42 秒 |
| 変異: production code の 5 個(下の節) | 5 | 5 個が死んだ | — | — | 1 個 42〜49 秒 |
| `-count=30 -v`、org: cycle 2 で足したか行を足した 4 本(`TestOrgSendScreen_TwoReadsAreTheReadGapApart`、`TestSendInputHolds_NumberedStepOnTheMarkerLine`、`_NumberedMarkerLineRefuses`、`TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed`) | 上位 120 回(subtest 込みの PASS 行 750) | 750 | 0 | 0 | 138 秒。load 7〜18 |
| coverage: org `-run 'Send\|Screen\|Live\|Resume\|PaneLock'` | 名前で絞った回 | ok | 0 | — | 44 秒 |
| probe: overlay で足したテスト 1 本(Test gaps 8) | 1 | 1 | 0 | 0 | 8 秒 |

Skipped の 2 行は cycle 1 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

run-test.sh の Go の段のあいだ、1 分の load は 6 回の見本で 13 から 23 だった。ほかの session が `go test ./...`、cli の `-run Gate|Regate`、overlay の変異を同時に回していた。org は 1,007 秒で、30 分の上限の 56% である(cycle 1 は load 7〜13 で 790 秒)。落ちたテストは無い。

## 受け入れ条件とテスト

どのテストも run-test.sh で PASS した。「変異」の列は、そのテストが落とした変異の id である。M01、M03、M04、M06、M07、K01 は cycle 1(HEAD `88742506`)の結果で、この回は回していない。

| AC | テスト(ファイル) | 変異 |
| --- | --- | --- |
| AC1 herdr が working のまま、画面が入力待ちの座席へ打ち、`sent` に `via=screen` | `TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen`(claude、codex)、`_AgentGetHangs_EndsAtTheDriverCallTimeout`、`_MultiLineMessageTallerThanTheBox_EnterPressed`(cycle 2 で、入力欄の 1 行目が番号付きの手順になる claude と codex の 2 行を足した)。cli の `TestOrgSendScreenPath_ScreenWaitsForInput_TypesAndRecordsViaScreen` | 当てていない |
| AC2 入力待ちでない画面では打たず、エラーに herdr の状態・判定と理由・次の手 | `TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy` の 7 行、cycle 2 で足した `TestOrgSendScreen_TwoReadsAreTheReadGapApart`(2 回目の読みが 1 回目から読みの間隔以上あとであること)。cli の `TestOrgSendScreenPath_ScreenBusy_TypesNothingAndSaysWhy` | M02 が死んだ(cycle 1 では生き残った)。M01(cycle 1) |
| AC3 ロックの下の読み直しで入力待ちでなければ打たない | `TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing` | 当てていない |
| AC4 Enter の前に文が入力欄に無ければ Enter を押さない | `TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed`、`TestSendInputHolds_TypedMultiLineFixtures`、`_Refuses`、`_ClaudePasteBoxes`。cycle 2 で `TestSendInputHolds_NumberedStepOnTheMarkerLine`(番号付きの行の入力欄で send の確かめは通り、resume と Enter のあとの確かめは選択の画面のまま)と `_NumberedMarkerLineRefuses`(本物の選択の画面 4 つ、別の文、罫線や footer を外した画面、文に無い番号付きの行の 14 行)を足した | N1、N2、N3 が死んだ。M03(cycle 1) |
| AC10 画面の道の Enter のあとは画面で確かめ、文が残れば `submit_unconfirmed=true` | `TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly`、`TestSendScreenSubmitted`(cycle 2 で、番号付きの行が入力欄に残る 4 行を足した) | M04(cycle 1) |
| AC11 確かめのあと、残りの時間で herdr の待ちに戻る | `TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath`、`_PaneReadHangs_HerdrsWaitGetsTheRest`、`_HerdrDoneAtTheCheck_ScreenNotRead` | M06、M07(cycle 1) |
| AC5 `--no-screen-fallback` と、idle か done の道 | `TestOrgSendScreen_NoScreenFallback_HerdrOnly`、`_HerdrIdleAtOnce_ScreenNotRead`、`_ReserveTooShortForTheCheck_HerdrOnly`、`TestSendScreenReserve_Table`。cli の `TestOrgSend_NoScreenFallbackFlag_ReachesSend`、`TestOrgSendScreenPath_NoScreenFallback_ReadsNoScreen` | 当てていない |
| AC6 `status --live` の列、食い違いの印、`--live` なしは今と同じ | `TestLiveSeatStates_*` の 6 本と `TestLiveSeatState_TextOfTheColumns`。cli の `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`、`TestOrgStatusLive_*` の 3 本、`TestOrgStatus_HelpDescribesLive` | K01(cycle 1) |
| AC7 codex 0.166 の footer で入力欄が見つかり、既存の fixture の判定は変わらない | `TestFindResumeInputBox_Codex0166Footer`、`TestResumeCodexFooterLine`、`TestClassifyResumeScreen_Fixtures`、`TestResumeInputHolds`(cycle 2 で、入力欄に文の頭だけが見える行を足した) | R1 が死んだ(`findResumeInputBox` の分割で、番号付きの行の判定を形の確かめより前に置く順) |
| AC8 入力待ちの判定が、採った画面と既存の testdata のすべてで分かれる | `TestClassifySendScreen_EveryFixture`(63 の fixture。cycle 2 の 2 つは選択の画面と判定される)、`TestClassifySendScreen_Edges`、`TestSendScreenBoxCheck_ChoiceDialogWithNoMarkerLine` | R1 が死んだ |
| AC9 help、skill、recipe、AGENTS.md | `TestOrgSend_HelpDescribesTheScreenFallback`、`TestOrgStatus_HelpDescribesLive`。skill、recipe、AGENTS.md の文はテストで固定していない。/verify が読んで確かめた | — |

## 変異

orchestrator の指示(ディスクのため変異は 5 個まで)に合わせ、指示の優先順に当てた。M02 を新しいテストで当て直し、番号付きの行を入力欄とみなす 2 つの条件(印の行の文が打った文の一部であること、入力欄の形)に 3 個、resume の判定が分割のあとも変わらないことに 1 個である。各変異は文字列の置き換え 1 つで、置き換える文字列が写しのファイルに 1 回だけ現れることを確かめてから当てた。どれも org を `-run 'Send|Screen|Live|Resume|PaneLock'` で絞って回した。同じ絞り方の対照は overlay なしで通った(42 秒)。

| 変異 | guard | 置き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- | --- |
| M02 | 2 回の読みのあいだの待ち(cycle 1 の Test gaps 1) | `send_fallback.go` の `waitOrCtxDone(ctx, gap)` を `waitOrCtxDone(ctx, 0)` に | 死んだ | `TestOrgSendScreen_TwoReadsAreTheReadGapApart`(`the screen check's two reads were 20.583µs apart; want at least the read gap, 200ms`) |
| N1 | 印の行の文が打った文の一部であること | `send_screen.go` の `if !strings.Contains(want, resumeDropSpace(text)) {` を `if false && !strings.Contains(...) {` に | 死んだ | `TestSendInputHolds_NumberedMarkerLineRefuses` の「claude, a numbered line not of the message above its start」と codex の同じ行(`sendInputHolds = true, want false`) |
| N2 | 入力欄の形(claude: 印の行の上下の罫線) | `if reason != "" \|\| (driver == "codex" && ...)` を `if false \|\| (driver == "codex" && ...)` に | 死んだ | `_NumberedMarkerLineRefuses/claude-folder-trust.txt, a message made of it` で panic(`slice bounds out of range [1:0]`) |
| N3 | 入力欄の形(codex: footer) | `(driver == "codex" && !sendCodexHasFooter(lines, box))` を `(driver == "codex" && false)` に | 死んだ | `_NumberedMarkerLineRefuses` の「codex-hooks-review.txt, a message made of it」「codex-update-available.txt, a message made of it」「codex, the numbered fixture with no footer」 |
| R1 | resume の番号付きの行の判定の順 | `resume_screens.go` の `findResumeInputBox` で、`resumeInputBoxShape` の確かめを番号付きの行の判定より前に置く | 死んだ | `TestClassifyResumeScreen_Fixtures` の claude の 2 行(reason が `no input box` に)、`TestClassifySendScreen_EveryFixture` の `claude-dialog-below-old-input.txt`(kind が unknown に)と `claude-folder-trust.txt`(line が `Enter to confirm · Esc to cancel` に)、`TestOrgResume_ScreenChangedUnderTheLock_TypesNothing/an_approval_dialog` |

変異から分かったこと:

- N2 は assert の行ではなく panic で死んだ。形が合わないとき `resumeInputBoxShape` は空の box を返し、この条件を外すとその box が `holdsHead` に渡り、`resumeInputBox.text` が `lines[1:0]` で範囲外になる。claude の形の条件は、誤って打たないためというより、空の box を先に渡さないために要る。panic でテストの binary が止まったので、同じテストのそれより後の行(「claude, the numbered fixture with no rule above the box」など)はこの変異の下では回っていない。読む限り、その行も同じ panic になる
- 「the numbered fixture with another message」の 2 行は N1 で落ちなかった。別の文では、入力欄が見つかっても 3 つの規則のどれにも合わないので、印の行の文の確かめを外しても結果は false のままである。N1 を固定しているのは「a numbered line not of the message above its start」の 2 行だけである
- R1 で、resume の判定(`ClassifyResumeScreen`)、send の判定(`ClassifySendScreen`)、resume のロックの下の読み直しの 3 か所が、分割の前と同じ順を固定していることを確かめた

## Coverage

- Statement: 名前で絞った org の回の profile と `git diff -U0` の足した行を突き合わせた。cycle 2 で足した行は、`send_screen.go` 34/38、`resume_screens.go` 20/20、`send_fallback.go` 0/0(block に入る行が無い)。branch 全体では `send_screen.go` 234/239、`resume_screens.go` 20/20、`send_fallback.go` 156/163、`status_live.go` 49/49、`verbs.go` 28/28、`resume.go` 1/1。`internal/cli/org.go` は cycle 1 のあと変わっていない(cycle 1 で 75/75)。名前で絞った回なので、パッケージ全体の数字(org 34.6%)は意味を持たない
- 通っていない行:
  - `send_screen.go:293`: `sendInputHolds` の `switch` の後の `return false`。f24290b5 で、知らない driver を関数の頭で断るようになったので、今は届かない
  - `send_screen.go:322〜323`: cea96ea4 の確かめ。`findResumeInputBox` が選択の画面を返すのは印の行を見つけたあとだけなので、今は届かない
  - `send_screen.go:415〜416`: `sendCodexHasFooter` の、`codexFooterStart` が見つけた footer で true を返す道。テストの codex の番号付きの入力欄は、どれも footer に `⚠ 11 · f2` の行があり、`sendCodexFooterInInput` の道で footer を見つける(Test gaps 8)
  - `send_fallback.go:199〜201`、`241〜242`、`254〜255`: cycle 1 と同じ(Test gaps 2、3)
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: cycle 2 で変えた関数は、`ResumeInputHolds`、`holdsHead`、`findResumeInputBox`、`resumeMarkerLineBox`、`resumeInputBoxShape`、`SendScreenSubmitted`、`sendCodexTypedInputText`、`sendCodexFooterInInput` が 100%、`sendInputHolds` 94.1%、`sendTypedInputBox` 93.3%、`sendCodexHasFooter` 75.0% である。足りないのは上の通っていない行の分だけである
- Notes: 415〜416 は行の coverage で見えている穴で、この道の変異はどれも生き残る。下の probe で、production code はこの画面で正しく true を返すことを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh、変異の対照、`-count=30`、coverage、probe のどれでも落ちたテストは無い。変異の 5 個の失敗は上の変異の節に書いた。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の send、pane ロック、resume、承認の確かめのテスト(計画の Regression tests) | 通った | `internal/org` と `internal/cli` のパッケージ全体を run-test.sh で通した。cycle 2 は既存のテストの期待を消していない(diff は行と関数の追加だけ) |
| resume の判定が `findResumeInputBox` の分割で変わらないこと | 通った | `TestClassifyResumeScreen_Fixtures` と `TestResumeInputHolds` の既存の行がそのまま通る。`TestSendInputHolds_NumberedStepOnTheMarkerLine` が、番号付きの 2 つの fixture で `ResumeInputHolds` が false、`ClassifyResumeScreen` が選択の画面のままであることを見る。R1 で、判定の順が 3 つのテストに固定されていることを確かめた |
| cross-review(cycle 1)の指摘: 入力欄の 1 行目が番号付きの手順だと Enter が押されない | 直った | `TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed` の番号付きの 2 行(claude、codex)が、Enter を 1 回押し、画面で送れたと確かめ、`via=screen` を残す。N1〜N3 で、本物の選択の画面には打たないことを確かめた |
| cycle 1 の生き残った変異 M02 | 直った | `TestOrgSendScreen_TwoReadsAreTheReadGapApart` が M02 を落とした |
| `ralph org status` の `--live` なしの出力 | 通った | `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`(cycle 1 の K01 で変化を捕まえることを確かめた) |
| 時間に頼るテストの余裕 | 通った | run-test.sh の org のパッケージ全体の回は load 13〜23 で通った(cycle 1 は 7〜13)。`TestOrgSendScreen_TwoReadsAreTheReadGapApart` は `-count=30` で 30/30(load 7〜18) |

## cycle 1 の Test gaps の今

| cycle 1 の gap | 今 | 根拠 |
| --- | --- | --- |
| 1. 2 回の読みのあいだの待ちを固定するテストが無い(M02) | 閉じた | `TestOrgSendScreen_TwoReadsAreTheReadGapApart`。M02 が死んだ |
| 2. 確かめの時間が 2 回の読みのあいだで切れる道(`send_fallback.go:199〜201`) | 開いたまま | coverage で 199〜201 は通っていない |
| 3. `sendHerdrState` が `unknown` を返す 2 つの道 | 開いたまま | coverage で 241〜242、254〜255 は通っていない |
| 4. `Send` の確かめで herdr が blocked を返す道 | 開いたまま | テストは足されていない |
| 5. 実機の herdr、claude、codex で画面の道を通していない。既定の 30000 ms で画面の道を通すテストが無い | 開いたまま | テストは足されていない |
| 6. AC3、AC5、AC6、AC7、AC8 の guard に変異を当てていない | 一部 | R1 で `findResumeInputBox` の判定の順(AC7、AC8 の選択の画面の扱い)を確かめた。AC3、AC5、AC6、footer の正規表現の錨(F2、F3、F6)は当てていない |
| 7. 時間に頼るテストの余裕は load 13 までしか見ていない | 狭まった | load 23 までのパッケージ全体の回で通った。CI のもっと重い load は見ていない |

## Test gaps

1. (cycle 1 の 2)確かめの時間が 2 回の読みのあいだで切れる道を通るテストが無い。足し方は cycle 1 の報告と同じで、`sendFallbackOrg` に読みの間隔 100 ms、`DriverCallTimeout` 100 ms、`blockGet` を足し、verdict が `the screen check ran out of time between its two reads` になり、残りが herdr の待ちに渡ることを見る行である。回していない
2. (cycle 1 の 3)`sendHerdrState` が `unknown` を返す 2 つの道を `Send` の側で通すテストが無い。本番の herdr の client は `AgentGet` を持つので、影響は小さい
3. (cycle 1 の 4)`Send` の確かめで herdr が blocked を返す道のテストが無い
4. (cycle 1 の 5)実機の herdr、claude、codex で画面の道を通していない。番号付きの 2 つの fixture も、実機で採った画面ではなく、中くらいの文の画面から作った合成である(テストのコメントのとおり)。既定の `--timeout-ms` 30000 で画面の道を通すテストも無い
5. (cycle 1 の 6)AC3(ロックの下の読み直し)、AC5(フラグ、driver、reserve の算術)、AC6(読む座席の選び方)、footer の正規表現の錨(F2、F3、F6)には変異を当てていない
6. (cycle 1 の 7)時間に頼るテストの余裕。`PaneReadHangs_HerdrsWaitGetsTheRest` の 2 回目の待ちの下限 100 ms に加え、cycle 2 の `TestOrgSendScreen_TwoReadsAreTheReadGapApart` も余裕は約 100 ms である。確かめに使えるのは reserve 600 ms の半分の 300 ms で、`agent get`、1 回目の読み、200 ms の待ちがそこに入らないと、確かめが読みのあいだで切れて Send が失敗する。load 23 までは通った。CI でもっと重い load になれば落ちうる。未確認です
7. `sendTypedInputBox` の claude の形の条件(N2)は、assert ではなく `resumeInputBox.text` の panic で固定されている。今の `text` は空の box で必ず範囲外になるので N2 は落ちる。`text` が空の box を許すように変われば、この条件を外しても `sendInputHolds` の結果は false のままで、テストは何も言わなくなる。そのときは、この条件を外した変異が生き残る形になる
8. (cycle 2 で見つけた)`sendCodexHasFooter` の、`codexFooterStart` が見つけた footer の道(`send_screen.go:415〜416`)を通るテストが無い。codex の番号付きの入力欄のテストは、どれも footer に `⚠ 11 · f2` の行がある画面である。overlay で足した probe(`codexResumeScreen` の 0.165 の形の footer、入力欄の 1 行目が `2. Read the files the plan lists for the slice.    ↑`、文は `sendNumberedStepsMsg`)で、`codexFooterStart` が footer を見つけ(`footerStart=17`、`len=20`)、`sendInputHolds` が true、`SendScreenSubmitted` が false になることを確かめた。production code に問題は無く、テストが無いだけである。いちばん小さい足し方は、この画面を `TestSendInputHolds_NumberedStepOnTheMarkerLine` か同じ形の表に 1 行足すことである。この行が `if box.footerStart < len(lines) { return true }` を外した変異を落とすかは、変異の上限のため回していない(読む限り、外すと `sendCodexFooterInInput` が false を返すので落ちる)

## Verdict

- Pass: run-test.sh(shell 45 本、Go 10 パッケージ、1,597 秒)、変異の対照、`-count=30`、coverage の回、probe のすべてが通った。cycle 2 で当てた変異 5 個(M02、N1、N2、N3、R1)はすべて死んだ。cycle 1 で生き残った M02 は、新しいテストが落とすようになった(cycle 1 の Test gaps 1 は閉じた)。cross-review の指摘(番号付きの行の入力欄)は、send の確かめが通る行と、本物の選択の画面を断る行の両方がテストで固定されている。残る穴は Test gaps の 8 項目で、どれも通るべき挙動の失敗ではない
- Fail: 0
- Blocked: なし
