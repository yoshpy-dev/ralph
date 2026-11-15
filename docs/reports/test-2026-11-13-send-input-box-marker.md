# Test report: send-input-box-marker

- Date: 2026-11-15。run-test.sh は 19:02:25 から 19:30:11 JST(1,666 秒)に 1 回だけ回した。変異 19 個、survivor の probe、絞り込みの coverage の回は、run が終わったあとの 19:31 から 19:50 JST に回した
- Plan: `docs/plans/active/2026-11-13-send-input-box-marker.md`(`## Test plan`、AC1〜AC6 と AC4b、Design decisions の D2 の 5 条件。issue #322)
- Tester: tester サブエージェント(`/test`、パイプライン 1 回目。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff bbe2807d...HEAD`(base は origin/main `bbe2807d`、HEAD `9784fcbf`)。33 ファイル、+1713/-26。Go の本体は `internal/org/send_screen.go` だけで、テストは `send_screen_marker_test.go`(新規)、`send_fallback_marker_test.go`(新規)、`prompts_send_unsubmitted_test.go`(新規)、`send_screen_test.go`、`resume_screens_test.go`、`usage_limit_screen_test.go`、fixture 11。run-test.sh は既定の changed scope で回った(`- Requested scope: changed`、golang の pack だけが選ばれた)
- Evidence: `docs/evidence/test-2026-11-13-send-input-box-marker.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力のあとに、30 秒ごとの load と空きディスク、変異 19 個の書き換えと出力、probe の行と出力、coverage の集計を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-15-100225.log`
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、Bash ツールの `run_in_background` で起動し、開始・終了・rc を scratchpad の meta ファイルに書いた。終わりは `caffeinate -w <pid>`(1 回 175 秒まで)を繰り返して待った。回し直しはしていない。終わったあと、run-test.sh、load の見本取り、変異の runner、自分の `go test` が残っていないことを `ps` で確かめた(残っていた `go test` は別の worktree の別セッションのもの)。herdr、claude、codex は起動していない。実機の画面は `internal/org/testdata/resume/` の fixture(`docs/evidence/org-send-marker-probe-2026-11-13.md` の写し)だけを使った

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195、`FAIL: 0` の要約行 28、`FAIL: [1-9]` は 0 行)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,666 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 971.3 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 838.3 秒 |
| └ Go: `internal/config` | パッケージ全体 | ok | 0 | — | 13.4 秒 |
| └ Go: `cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | パッケージ全体 | ok `(cached)` | 0 | — | — |
| `go test -count=1 ./cmd/ralph ./internal/insights`(`(cached)` のうち `internal/org` を import する 2 つ、`go list -deps` で確かめた) | パッケージ全体 | ok | 0 | — | 0.9 秒、0.5 秒 |
| 変異: `send_screen.go` の 17 個と prompt の 2 個(下の「変異」) | 19 | 14 個が死んだ | — | — | 1 個 37〜41 秒(build 込み)、prompt は 8〜9 秒 |
| survivor の probe: overlay で足したテスト 1 本(3 行) | 3 | HEAD で 3 行とも pass | 0 | 0 | 7 秒 |
| coverage: org を `-run 'Screen\|SendInputHolds\|Resume\|SendMarker\|SendProbe' -coverprofile` | 名前で絞った回 | ok | 0 | — | 37.3 秒(build 込み 45 秒) |

この branch で足したテストは run-test.sh の org と cli の回で通った。数はテストのコードから数えた(run-test.sh は `go test` を `-v` なしで回す)。

- `send_screen_marker_test.go`: トップレベル 4 本。`TestSendMarkerProbeMessages_AsTyped`(10 の文の大きさ)、`TestSendInputHolds_TypedMarkerFixtures`(subtest 9)、`TestSendInputHolds_MarkerLineUnderANumberedHead`(subtest 2)、`TestSendInputHolds_MarkerLineRefuses`(subtest 16)
- `send_fallback_marker_test.go`: `TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed` の 1 本
- `prompts_send_unsubmitted_test.go`: leader の 1 本と director の 1 本(herdr の形と launch_cmd の形の subtest 2)
- 既存の表に足した行: `sendScreenFixtureWants` 11、`TestSendScreenSubmitted` 2、`TestClassifyResumeScreen_Fixtures` 11、`usageLimitFixtureWants` 11

Skipped の 2 行は、`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがあると起動してしまうため)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらも手元の環境によるもので、この diff と関係がない。前の run(send-claude-paste)と同じ 2 行である。

30 秒ごとに 78 回見た 1 分の load は、shell の段(19:13 ごろまで)で 8.3 から 22.0、Go の段で 4.8 から 111.4 だった。最大の 111.4(19:15:54)は Go の段の初めで、自分の回は run-test.sh だけだったので、ほかのセッションの負荷である。org の 971.3 秒は golang の pack の上限(1 パッケージ 30 分、`-timeout=30m`)の 54% だった。空きディスクは run の前後で 55 GB と 52 GB で、`no space left` は 0 行だった。

## Coverage

- Statement: 絞り込みの回で、`sendTypedInputBox`、`sendTextInWant`、`sendCodexFooterStart`、`sendCodexHasFooter`、`SendScreenSubmitted` は 100.0%、`sendTypedMarkerLineBox` は 96.0%、`sendInputHolds` は 94.1%。`send_screen.go` の足した行(110 行)に触れる coverage の block の文は 37 で、そのうち 36 を通った(`git diff -U0 bbe2807d...HEAD` の足した行と profile の block を突き合わせた)。通らなかった 1 文は `sendTypedMarkerLineBox` の `if head < 0 { return resumeInputBox{}, false }`(`send_screen.go:393`)で、変異 M3d と gap 1 で扱う。`sendInputHolds` の未通過は base からある末尾の `return false`(298 行、知らない driver は先に返るので届かない)
- Branch: Go のツールでは測っていない。D2 の各条件の両向きは、下の対応表と変異で見た
- Function: 新しい関数 3 つ(`sendTypedMarkerLineBox`、`sendTextInWant`、`sendCodexFooterStart`)はすべて通った
- Notes: run-test.sh は coverage を出さない。パッケージ全体の数字は測っていない(絞り込みの回の 15.9% は、名前で絞ったテストだけの値)

## 受け入れ条件・Test plan とテスト

| 項目 | テスト | 結果 |
| --- | --- | --- |
| AC1 codex の実機の 4 画面で `sendInputHolds` が true | `TestSendInputHolds_TypedMarkerFixtures` の codex の 4 行(`codex-input-typed-marker-line.txt`、`-line-last.txt`、`-line-scrolled.txt`、`codex-input-typed-marker-numbered.txt`)。箱の頭が 0 桁目の行であること、`sendInputHolds` が true、`SendScreenSubmitted` が false | pass |
| AC2 claude の改行 2 つと 1 つの文で true。改行 0 と placeholder も true のまま | 同じテストの claude の 5 行(two-nl、one-nl、wrapped、no-nl、paste-short) | pass |
| AC3 今まで false の画面は false のまま。3 つの縁 | 既存の `TestSendInputHolds_Refuses` と `TestSendInputHolds_NumberedMarkerLineRefuses`(diff で変わっていない)が通った。縁は `TestSendInputHolds_MarkerLineRefuses` の「not of the message」の 2 行、「no marker line at column 0 above it」の 2 行、「under the bottom rule」の 2 行(罫線が 0 桁目と 2 字下げ) | pass |
| AC4 文が残った画面で `SendScreenSubmitted` が false、送れた画面で true。resume と `ClassifySendScreen` の期待は変わらない | 残った画面は `TypedMarkerFixtures` の 9 行と numbered-head の 2 行。送れた画面は `TestSendScreenSubmitted` の 2 行(`claude-submitted-marker-line.txt`、`codex-submitted-marker-line.txt`)。`findResumeInputBox`、`ClassifyResumeScreen`、`ResumeInputHolds`、`ClassifySendScreen` のコードは diff にない。それらの既存のテストは期待を変えずに通り、新しい fixture の行(resume 11、send 11)は今の resume の判定を記録している | pass |
| AC4b 画面の道の `Send` で Enter 1 回、`SubmitConfirmed` true、`via=screen`、`submit_unconfirmed` なし | `TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed`。herdr の呼び出しの列を完全一致で見て(`keys pane-1 Enter` は 1 回)、Details も完全一致で見る。変異 M0 と M4f ではこのテストが `(text typed but not submitted)` で落ち、#322 の症状を再現した | pass |
| AC5 leader と director の prompt の文をテストで固定 | `TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates` と `TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen`。変異 P1 と P2 はそれぞれ死んだ。skill の 4 つの写しの文はテストで固定されていない(gap 3) | pass |
| AC6 確かめの記録と fixture | テストの対象ではない(`/verify` が fixture 11 と probe の画面を `cmp` で比べた)。`TestSendMarkerProbeMessages_AsTyped` が、テストの文の大きさを証拠の表の値と比べている | — |
| Edge: 印の行が最後(last)、番号付きの印の行(numbered) | `codex-input-typed-marker-line-last.txt` と `codex-input-typed-marker-numbered.txt` の行、`MarkerLineUnderANumberedHead` の 2 行 | pass |
| Regression: #319 の placeholder の規則 | `claude-input-typed-paste-short.txt`(`[Pasted text #1 +3 lines]`)で `sendInputHolds` が true、`SendScreenSubmitted` が false | pass |

## 変異

本体を `go test -overlay` で 1 か所ずつ書き換え、`-run 'Screen|SendInputHolds|Resume|SendMarker|SendProbe' -vet=off -count=1` で回した(書き換えなしの対照は rc 0、34 秒)。書き換えは scratchpad の perl スクリプトで文字列を 1 か所だけ置き換えた(1 回だけ現れることを確かめてから)。worktree のファイルは書き換えていない(前後の `git status --porcelain --ignored` は同じ)。

| # | D2 の条件 | 書き換え | 結果 |
| --- | --- | --- | --- |
| M0 | (全体) | `sendTypedInputBox` から `sendTypedMarkerLineBox` の呼び出しを外す | 死んだ。新しい道の 7 つの fixture、numbered-head の 2 行、AC4b の結合テスト |
| M1 | 1 | L の印の前に空白があることを見ない | 死んだ。「codex, a head at column 0 that holds a part of the message」の行(`sendTypedMarkerLineBox` と `sendInputHolds` の両方の assert) |
| M2 | 2 | L の文が打った文の一部かを見ない | 死んだ。「not of the message」の 2 行と「nothing after the indented marker」の 2 行 |
| M2b | 2 | `sendTextInWant` で空の文も一部とみる | 死んだ。「nothing after the indented marker」の 2 行 |
| M3a | 3 | M と L のあいだの 0 桁目の行を許す | 死んだ。「a line at column 0 between the head and it」 |
| M3b | 3 | あいだの行の字下げを 2 字から 1 字に緩める | 生き残った(gap 2) |
| M3c | 3 | M が 0 桁目でなくてもよい | 死んだ。「no marker line at column 0 above it」の 2 行 |
| M3d | 3 | M が見つからないときの `head < 0` の return を外す | 生き残った(gap 1)。probe の行では `index out of range [-1]` で panic した |
| M4a | 4 | `resumeInputBoxShape` の reason を見ない | 生き残った。等価: codex では reason が常に空、claude では失敗のとき空の箱が返り `inputEnd(0) <= last.marker` が同じ結果にする(self-review と verify の読みと同じ) |
| M4b | 4(claude) | `box.inputEnd <= last.marker` を見ない | 死んだ。「under a bottom rule with two spaces before it」の 1 行だけ。0 桁目の罫線の行は条件 3 が先に外すので、条件 4 を固定しているのは 2 字下げの罫線の行である |
| M4c | 4(codex) | footer がないことを許す | 死んだ。「codex, no footer under the box」 |
| M4d | 4(codex) | L が footer より下にあることを許す | 死んだ。「codex, the indented marker line under the footer」 |
| M4e | 4 | claude と codex 以外の driver を断らない | 死んだ。`gemini` の行の `sendTypedMarkerLineBox` の assert だけ。`sendInputHolds` は知らない driver を先に断るので、この分岐は二重の守りである |
| M4f | 4(codex) | footer を `codexFooterStart` だけで探す(入力に数えられた `⚠ 11 · f2` の footer を見ない) | 死んだ。codex の 4 つの fixture、numbered-head の codex の行、AC4b の結合テスト(Enter が押されない) |
| M4g | 4(codex) | `sendCodexFooterStart` で 2 つの footer の順を入れ替える | 生き残った。知られた画面では等価: 違いが出るのは、入力欄の中の文の最後の段落が codex の footer の形の行で始まるときだけ |
| M5 | 5 | 番号付きの M の文が打った文の一部かを見ない | 死んだ。「a numbered head not of the message」の claude と codex の 2 行 |
| M5b | 5 | 番号付きでない M の文も打った文の一部であることを求める(厳しくする) | 生き残った。知られた画面では等価: 受け入れる画面の頭の文はどれも打った文の一部 |
| P1 | (AC5) | leader.md の「同じ send を打ち直さないでください。」を「同じ send を打ち直してください。」に | 死んだ。leader のテスト |
| P2 | (AC5) | director.md の「それ以外は Enter を押さず、」を「それ以外も Enter を押し、」に | 死んだ。director のテストの両方の形 |

`send_screen.go` の 17 個のうち 12 個と prompt の 2 個が死んだ。生き残った 5 個のうち M4a は等価、M4g と M5b は知られた画面では等価で、穴として残るのは M3b と M3d である。この 2 つは overlay で足したテスト(`zz_probe_test.go`、repo には入れていない)で確かめた。HEAD では 3 行とも通り、M3d では「codex, only blank lines above the indented marker line」の行が panic し、M3b では「a line with one space before it between the head and it」の行で `sendInputHolds` が true になった。S1 の implementer の記録(「`reason != ""` を外す変異だけが生き残る」)とは、M3b と M3d の 2 つの分だけ違う。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh で落ちたテストはなく、load に弱いことが分かっているテスト(tester の記録の一覧)も今回は落ちなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #322: 印で始まる行を含む文で、画面の道が Enter を押さずに `text typed but not submitted` で終わる | 直った。新しい道を外す M0 と、`⚠` の footer を見ない M4f で、結合テストが同じエラーで落ちる | `TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed`、`TestSendInputHolds_TypedMarkerFixtures` |
| 選択の画面、入力欄の下の選択の画面、空の入力欄、ほかの文の入力欄では Enter を押さない | 変わらない | `TestSendInputHolds_Refuses`、`TestSendInputHolds_NumberedMarkerLineRefuses`(diff で変わらず通った) |
| resume の画面の型と、打つ前の `ClassifySendScreen` | 変わらない(コードは diff にない) | `TestClassifyResumeScreen_*`、`TestResumeInputHolds`、`TestClassifySendScreen_*` が通った。新しい fixture の行は今の判定(`input_has_text`、`no_input_box`、`choice_dialog` など)を記録している |
| #319 の placeholder の規則(`[Pasted text #1 +3 lines]`) | 変わらない | `claude-input-typed-paste-short.txt` の行、`TestSendInputHolds_ClaudePasteBoxes` |
| 使用量の上限の画面の読み取り | 新しい fixture で上限の行を拾わない | `TestUsageLimitScreen_EveryFixture` の 11 行 |

## Test gaps

1. `sendTypedMarkerLineBox` の `head < 0` の return(`send_screen.go:393`)を通るテストがない。L より上がすべて空行か 2 字下げの行の画面でだけ通る分岐で、外すと `lines[-1]` で panic する(M3d、probe で確かめた)。今のコードは正しく、テストだけが足りない。足すなら `TestSendInputHolds_MarkerLineRefuses` に、`"\n\n  › explain this\n  Reply with the single word: alpha.\n\n  GPT-6-Astra · xhigh · Context 0% used\n  ? for shortcuts\n"` を codex、`sendMarkerCodexShortMsg` で false とみる 1 行が最小
2. 条件 3 の「2 つの空白で始まる」を、1 つの空白で始まる行と分けるテストの行がない(M3b)。足すなら、codex の short fixture の M と L のあいだに ` • Hi!`(1 字下げ)を入れて false とみる 1 行。実機で 1 字下げの行がそこに出る画面は見つかっていないので、優先度は低い
3. skill の文(`/org` skill の `send` の行、`/director` skill の「答える順」の手順 2)は、4 つの写しが同じことを `check-skill-sync.sh` が見るだけで、文を固定するテストはない。AC5 がテストを求めるのは prompt の文だけなので、受け入れ条件は満たしている
4. prompt のテストの `sendTypedNotSubmittedMark`("text typed but not submitted")は、`verbs.go` のエラー文とは別に書かれた文字列である。`verbs.go` の文を変えると `verbs_test.go` と `send_fallback_test.go` は落ちるが、prompt のテストは通り続け、prompt と skill の文が古くなっても気づけない。`Send` の返すエラーがこの定数で終わることを見る assert を 1 つ足すと、両方がつながる
5. 実機の側は単体テストの外にある: Claude Code 2.1.316 と codex-cli 0.166.0 以外の版、幅 53 桁以外の pane、claude が上限に当たらずに返事をしたあとの画面(計画の Open questions、verify の V2 の (a)(b))。この run では herdr、claude、codex を起動していない
6. 打つ前の確かめで、前の send の残りの文が入力欄にある場合(計画の Non-goals)と、codex への bracketed paste と長い文(#326)はテストもない
7. `-race` は回していない。この diff は goroutine もロックの扱いも変えていない

## Verdict

- Pass: run-test.sh(rc 0、shell 45 本と Go 10 パッケージ、落ちたテスト 0)。AC1〜AC5 と AC4b、Test plan の unit、integration、regression、edge の各項目。D2 の 5 条件はどれも、少なくとも 1 つの変異をテストが落とした。変更行の coverage は 37 文中 36 文
- Fail: 0
- Blocked: なし。`head < 0` の分岐のテストがないこと(gap 1)と、字下げの幅の行がないこと(gap 2)は残るが、どちらも今のコードの誤りではなく、PR を止めるものではない
