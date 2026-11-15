# Test report: send-codex-paste(issue #326、codex の座席への bracketed paste)

- Date: 2026-11-15 JST(run-test.sh は 22:00:25 から 22:23:37、UTC では 13:00:25Z から 13:23:37Z)
- Plan: `docs/plans/active/2026-11-15-send-codex-paste.md`(`## Test plan`、AC1〜AC4、D2。AC5 は実機の記録 `docs/evidence/org-send-codex-paste-live-2026-11-15.md` の担当で、ここではやり直していない。herdr、claude、codex は起動していない)
- Tester: tester サブエージェント(`/test`)。パイプラインの 1 回目(`cycle-count.json` は `{"cycle": 1}`)
- Scope: HEAD 1b847444(origin/main 24e6723c を取り込んだ merge)。`git diff origin/main...HEAD` は 29 ファイル。本体の Go の変更は `send_paste.go`(`sendTextForDriver` が codex も印で挟む)と `send_screen.go`(`sendInputHolds` の規則 4、`sendCodexPasteHolds`、`sendCodexPastePlaceholder`、`sendInputHasPastePlaceholder` の codex の分岐)で、`send_fallback.go` と `verbs.go` はコメントだけ。run-test.sh は既定の scope で 1 回だけ回した。求めた scope は changed だが、分類のない fixture の path で language scope が full に落ち(`Language scope: full fallback (unclassified:internal/org/testdata/resume/codex-input-pasted-medium.txt)`)、golang の pack が選ばれた。shell のテストは `scripts/verify.local.sh` が scope によらず回す
- Evidence: `docs/evidence/test-2026-11-15-send-codex-paste.log`(run-test.sh の出力の写し。`docs/evidence/*.log` は gitignore の対象なのでコミットしていない)。run-verify.sh が自分で書いた写しは `docs/evidence/verify-2026-11-15-130026.log`(同じく gitignore)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(1 回、既定の scope、`run_in_background` で起動) | 下の 2 行 | rc 0、`==> All verifiers passed.` | 0 | 下の行 | 1,392 s(load 4.3〜8.3) |
| shell(`verify.local.sh`、`tests/*.sh` 45 ファイル) | `PASS` の行 5,195 | 5,195 | 0(各ファイルの `FAIL: 0`、`N passed, 0 failed`) | `SKIP:` の行 2(ホストの事情で飛ばす行: `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の git 2.41 未満の実物の git) | run-test.sh の前半、約 11 分 |
| Go(golang pack の `go test ./...`、package ごとに `-timeout=30m`) | test を持つ 11 package | 11 `ok`(`(cached)` なし) | 0 | 0 | `internal/org` 779.841 s(上限 30 分の 43%)、`internal/cli` 680.270 s、`internal/toolchain` 24.958 s、`internal/config` 12.378 s、ほかは 5 s 未満 |
| 新しいテストと期待を変えたテストを名指しで `-v`(`go test ./internal/org/ -run '<下の 25 本>' -count=1 -v`) | top-level 25、subtest を含む `--- PASS` 338 | 338 | 0 | 0 | 7.1 s |
| 変異の対照(`go test ./internal/org/ -run 'Paste\|Screen\|SendInputHolds\|Send\|Resume\|UsageLimit\|Marker\|PaneLock' -count=1`、計画の slice の確かめと同じ絞り) | top-level 217 | 217 | 0 | 0 | 45 s |
| 変異 14 個(1 個ずつ `-overlay`、上と同じ絞り) | 14 | 12 個を殺した | — | — | 1 個あたり 49〜58 s |
| 生き残った 2 個を package の全体で(`nice -n 10 go test -overlay ... ./internal/org/ -count=1 -timeout 30m`、2 本を並べて。M13 は続けて `./internal/cli/ -run Resume`) | `internal/org` の全体 ×2、cli の Resume | 3 本とも `ok`(どちらも殺されない) | 0 | 0 | org 814.8 s と 814.4 s、cli 16.2 s(22:39〜22:53、load 6〜23) |
| 使い捨ての overlay のテスト 2 本(G1、G2 の提案の行。repo には入れていない) | 2 | base で 2 本 PASS。M11 の overlay で G1 の 1 本、M13 の overlay で G2 の 1 本だけが FAIL | — | — | 各 20 s 前後 |

名指しの 25 本: `TestSendCodexPasteProbeMessages_AsTyped`、`TestSendInputHolds_CodexPastedFixtures`、`TestSendInputHolds_CodexPlaceholderRefuses`、`TestSendScreenSubmitted_CodexPlaceholderStillInTheBox`、`TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed`、`TestOrgSendScreen_CodexPastePlaceholderOfAnotherSize_EnterNotPressed`、`TestOrgSend_DryRun_CodexSeat_DetailsWithoutMarkers`、`TestOrgSend_CodexSeat_SendTextFails_NoEnter`(以上 `send_codex_paste_test.go` の 8 本)、`TestSendTextForDriver_WrapsClaudeAndCodex`、`TestOrgSend_HerdrPath_TypesABracketedPasteForClaudeAndCodex`(`send_paste_test.go`)、`TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen`、`TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed`(`send_fallback_test.go`)、`TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed`(`send_fallback_marker_test.go`)、fixture の表 `TestClassifyResumeScreen_Fixtures`、`TestClassifySendScreen_EveryFixture`、`TestClassifySendScreen_Edges`、`TestUsageLimitScreen_EveryFixture` ほか `TestUsageLimitScreen_*` 4 本、resume の回帰 `TestOrgResume_StoppedClaudeSeat_TypesTheSentenceAndEnterOnce`、`TestOrgResume_StoppedCodexSeat_TypesTheSentence`、`TestOrgResume_SendWaitsForAResumeHoldingThePaneLock`、`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`。

### Test plan の項目とテストの対応

| Test plan の項目 | テスト | 結果 |
| --- | --- | --- |
| Unit: `sendTextForDriver` の driver ごとの期待 | `TestSendTextForDriver_WrapsClaudeAndCodex`(claude と codex の短い文・複数行・空の文は印で挟む。空の driver、`gemini`、`Claude`、`Codex` は素の文) | PASS。変異 M1 を殺す |
| Unit: codex の置き換えの規則の通る画面と通らない画面(AC3) | `TestSendInputHolds_CodexPastedFixtures`(実機の 4 画面で true、規則 4 が通るのは置き換えの 2 画面だけ、Enter のあとの画面で `SendScreenSubmitted` が true)、`TestSendInputHolds_CodexPlaceholderRefuses`(14 行)、`TestSendScreenSubmitted_CodexPlaceholderStillInTheBox`(working の行の下に置き換えが残る 7 行と、空の入力欄の対照) | PASS。M2〜M10 を殺す |
| Integration: AC4 の 1 本 | `TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed`(p4、日本語 2000 字の本文の DECISION。打った文は `pastedForTest(msg)` の 1 回、Enter 1 回、`SubmitConfirmed`、Details は `via=screen herdr_state=working <元の文>` と等しい)と否定の側 `..._CodexPastePlaceholderOfAnotherSize_EnterNotPressed` | PASS |
| Integration: herdr の道の codex の座席で印で挟まれる 1 本 | `TestOrgSend_HerdrPath_TypesABracketedPasteForClaudeAndCodex` の `codex, short` と `codex, more than 800 bytes`、`TestOrgSend_CodexSeat_SendTextFails_NoEnter` | PASS。M1 を殺す |
| Regression: claude の座席の期待 | 同じ表の claude の行、`TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste` ほか、`internal/org` の全体の実行 | PASS |
| Regression: 既存の send と resume の画面のテスト | `internal/org` の全体(779.841 s)、fixture の表 3 つに足した 5 行ずつ | PASS |
| Regression: `ResumeText` が素の文で打たれること | claude の座席: `resume_test.go:289`、`pane_lock_test.go:518`、`:601`。codex の座席: 打った文を見るテストがない | claude は PASS で M14 を殺す。codex は M13 が生き残る(Test gaps の G2) |
| Edge: N が 1 違う置き換え、置き換え 2 つ、置き換えのあとに文、絵文字を含む文の N | `TestSendInputHolds_CodexPlaceholderRefuses` の p3 と p4 の ±1、UTF-16 の 2017、バイトの 4925、置き換え 2 つ(空白あり・なし)、後ろ・次の行・前の文 | PASS。M2〜M6 を殺す |
| 計画の文 4 つが実機で打った文と同じこと | `TestSendCodexPasteProbeMessages_AsTyped`(バイト数、rune、UTF-16 の単位、改行、SHA-256) | PASS |

## Coverage

- Statement: package 全体の数字は取っていない(run-test.sh の Go は coverprofile を書かない。`internal/org` 全体の計測は 800 s ほどかかる)。上の絞った実行(top-level 217 本)の profile では `internal/org` の 38.6%(絞った部分集合の値で、package の値ではない)
- Branch: Go の標準の計測にはない
- Function: 絞った profile で、変わった関数は `sendTextForDriver` 100%、`checkNoPasteMarkers` 100%、`sendCodexPasteHolds` 100%、`SendScreenSubmitted` 100%、`sendScreenSubmitChecks` 100%、`sendCodexTypedInputText` 100%、`sendInputHolds` 94.1%(通らないのは知らない driver の最後の `return false`。base からある行)、`sendInputHasPastePlaceholder` 80.0%
- Notes: 変わった行で見ると、本体の Go に足した行のうち cover profile の block に入る 13 行中 12 行を通った(`git diff -U0 origin/main...HEAD` の追加行と profile の block の突き合わせ)。通らない 1 行は `send_screen.go:566` の `return false`(claude と codex のほかの driver)。呼ぶ側の `SendScreenSubmitted` は `resumeScreenSentences` にない driver で先に返るので、今の呼び出しからは届かない

### 変異(1 個ずつ `go test -overlay`、絞りは対照と同じ。build の失敗と `no space left` は 0 件)

| ID | 変異(ファイル) | 見る条件 | 結果 | 落としたテスト |
| --- | --- | --- | --- | --- |
| M1 | `driver != "claude" && driver != "codex"` → `driver != "claude"`(codex に素の文、`send_paste.go`) | AC1 | 殺した | `--- FAIL` 15 行: `TestSendTextForDriver_WrapsClaudeAndCodex` の codex 3 行、herdr の道の codex 2 行、画面の道の codex の行(`HerdrWorkingAndScreenWaiting`、`MultiLineMessageTallerThanTheBox` 2 行、`MessageWithAMarkerLine`)、AC4 の結合テスト、`CodexSeat_SendTextFails_NoEnter` |
| M2 | N を rune ではなくバイト数 `len(typed)` と比べる(`send_screen.go`) | D2 | 殺した | `CodexPastedFixtures/codex-input-pasted-placeholder.txt`、`CodexPlaceholderRefuses/p4, its bytes`、AC4 の結合テスト |
| M3 | N を UTF-16 の単位数と比べる | D2 | 殺した | 置き換えの日本語の画面、`p4, its UTF-16 units`、AC4、`..._OfAnotherSize_EnterNotPressed` |
| M4 | `loc[0] != 0` を外す(置き換えの前の文を許す) | D2 | 殺した | `CodexPlaceholderRefuses/p4, the placeholder preceded by text` の 1 行だけ |
| M5 | `loc[1] != len(got)` を外す(後ろの文、2 つ目の置き換えを許す) | D2 | 殺した | 置き換え 2 つの 2 行、後ろの文の 2 行 |
| M6 | 数の比べを外す(`n > 0`) | D2 | 殺した | `CodexPlaceholderRefuses` の 8 行、`..._OfAnotherSize_EnterNotPressed` |
| M7 | 規則 4 に空白を除いた `want` を渡す(`sendCodexPasteHolds(got, want)`) | 配線 | 殺した | 置き換えの 2 画面、AC4 |
| M8 | 規則 4 を外す(`&& false`) | AC3 | 殺した | 同じ 3 本 |
| M9 | 規則 4 も `↑` の付いた入力欄に限る | AC3 | 殺した | 同じ 3 本 |
| M10 | `sendInputHasPastePlaceholder` の codex の分岐を `false` に(Enter のあとに置き換えを見ない) | AC3 | 殺した | `CodexPlaceholderStillInTheBox` の 6 行(2016 の行は `sendInputHolds` が先に止める) |
| M11 | 置き換えの正規表現が claude の形も受ける(`\[Pasted(?:Content\|text#\d+\+)(\d+)(?:chars\|lines)\]`) | D2(codex の形だけ) | 生き残った(絞った実行と、`internal/org` の全体 814.798 s の両方) | 既存の行 `p4, claude's form of placeholder` は `[Pasted text #1 +32 lines]` で、32 が rune 数 2016 と違うので、形によらず数の比べで落ちる。使い捨ての overlay のテスト(`[Pasted text #1 +2016 lines]` の codex の入力欄で `sendInputHolds` が false)は base で PASS、M11 で FAIL(G1) |
| M12 | `Send` が画面の道の確かめに印付きの文を渡す(`sendScreenSubmitChecks(..., sendTextForDriver(seat.Driver, p.Text))`、`verbs.go`) | AC2 | 殺した | `--- FAIL` 18 行(claude と codex の画面の道、AC4 を含む) |
| M13 | resume が codex の座席だけ `ResumeText` を印で挟む(`resume.go`) | Non-goals、回帰 | 生き残った(絞った実行、`internal/org` の全体 814.373 s、`internal/cli` の `-run Resume` 16.2 s) | `TestOrgResume_StoppedCodexSeat_TypesTheSentence` は送った回数(`text`、`Enter`)を見るが、打った文を見ない。入力欄の確かめも `ResumeText` を印なしで読むので通る。使い捨ての overlay のテスト(同じ組み立てで `h.typed()` が `ResumeText` そのものか)は base で PASS、M13 で FAIL(G2) |
| M14 | resume がすべての driver で `sendTextForDriver(t.driver, ResumeText)` を打つ(M13 の対照) | 回帰 | 殺した | claude の座席の 3 本(`StoppedClaudeSeat_TypesTheSentenceAndEnterOnce`、`SendWaitsForAResumeHoldingThePaneLock`、`AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`) |

使い捨てのテストと変異のファイルは scratchpad に置き、overlay で読ませた。worktree の `git status --porcelain` は変異の前後とも空だった。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | run-test.sh は rc 0。2 回目の実行はしていない(落ちたテストがないので、flake と本物の失敗を分ける必要がなかった) |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| codex-cli 0.166.0 が、素で打った 5340 バイトの文を 750 ms 後の Enter で送らない(#326、`docs/evidence/org-send-paste-probe-2026-11-13.md` の X1) | 単体では固定、実機は AC5 の記録 | M1(codex に素の文)をテスト 15 行が落とす。実機の確かめは `docs/evidence/org-send-codex-paste-live-2026-11-15.md`(この回ではやり直していない) |
| Claude Code 2.1.316 が素で打った 1117 バイトの文の頭を失う(#319) | 保たれている | claude の行(`TestSendTextForDriver_WrapsClaudeAndCodex`、herdr の道、`TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste`)が PASS。M12 で claude の画面の道の行も落ちる |
| 印を含む文の断り(すべての driver、`--raw` と dry-run も) | 保たれている | `TestOrgSend_PasteMarker_RefusedWithNothingSent` を含む `internal/org` の全体が PASS |
| 先頭が `›` の行を含む文の画面の道(#322) | 保たれている | `TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed` の codex の期待を印付きにして PASS |
| `ralph org resume` の続行の文は素で打つ | claude は固定、codex は未固定 | M14 は claude の 3 本が落とす。M13(codex だけ印付き)は生き残る(G2) |
| 画面の道の確かめと Details は印のない文を使う(AC2) | 固定 | M12 をテスト 18 行が落とす。`TestOrgSend_DryRun_CodexSeat_DetailsWithoutMarkers`、AC4 の Details の完全一致 |

## Test gaps

- G1(M11 の生き残り。`internal/org` の全体でも殺されない): codex の入力欄に claude の形の置き換えがあり、その数が文の rune 数と等しい画面を固定していない。`TestSendInputHolds_CodexPlaceholderRefuses` の `p4, claude's form of placeholder` の行は数が 32 なので、形の規則がなくても数の比べで落ちる。行を `[Pasted text #1 +2016 lines]` にするか、その行を 1 つ足せば M11 を殺せる(使い捨ての overlay のテストで確かめた)
- G2(M13 の生き残り。`internal/org` の全体と `internal/cli` の resume のテストでも殺されない。cli の resume のテストの codex の座席は選択の画面で飛ばされ、何も打たれない): codex の座席に `ResumeText` が素で打たれることを見るテストがない。`TestOrgResume_StoppedCodexSeat_TypesTheSentence` に `h.typed()` が `[]paneSendTextCall{{paneID: "pane-1", text: ResumeText}}` と等しいかの 1 行を足せば殺せる(同じく確かめた)。resume は `typeAndSubmit` に `ResumeText` を直接渡すので今の動きは正しい。Non-goals の「resume の打ち方は変えない」を codex で固定するものがないという穴である
- M4 を落とすのは `p4, the placeholder preceded by text` の 1 行だけ。穴ではないが、前に文がある形の行はこれ 1 つである
- N の数え方は、絵文字 1 つ(rune 1、UTF-16 の 2 単位)の文で UTF-16 との違いだけを固定している。結合文字や ZWJ の絵文字で rune と書記素の数が違う文は、実機で codex が何を数えるか確かめておらず、テストもない(計画の Open questions)。違えば規則 4 は通らず、Enter を押さずに止まる(知らせる側に倒れる)
- codex が畳み始める大きさ(651 バイトと 1698 バイトのあいだ)と、置き換えと文が混ざる形は、実機の画面がないのでテストもない(計画の Risks)
- 画面の道を実機で通しで走らせていない(AC5 は herdr の道を通った)。画面の道の codex の規則を固定しているのは、実機の画面の fixture を読むテストだけである
- `internal/cli` には、codex の座席に打った文を見るテストがない。`ralph org send` は `Send` をそのまま呼ぶので、変更の範囲は `internal/org` のテストで見ている
- 本文 2000 字を超える `--raw` の文、途中で切れた貼り付けのあとの codex の入力欄は、計画の Known gaps のとおりテストしていない

## Verdict

- Pass: run-test.sh 1 回(rc 0、shell 45 ファイル 0 FAIL、Go 11 package `ok`)。AC1〜AC4 と AC2 の Details を見るテストはすべて PASS。変異 14 個のうち 12 個を殺した
- Fail: 0
- Blocked: なし。G1 と G2 はテストの穴で、今の動きの誤りではない。足すなら 1 行ずつで、`/pr` を止める理由にはしない
