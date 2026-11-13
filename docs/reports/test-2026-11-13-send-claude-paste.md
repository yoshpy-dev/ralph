# Test report: send-claude-paste

- Date: 2026-11-13。run-test.sh は 14:14:54 から 14:40:59 JST(1,565 秒)に回した。変異 7 個、CLI の probe、絞り込みの coverage の回は 14:16 から 14:26 JST に回した
- Plan: `docs/plans/active/2026-11-13-send-claude-paste.md`(`## Test plan`、AC1〜AC7。issue #319)
- Tester: tester サブエージェント(`/test`、パイプライン 1 回目。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(base は origin/main `e902705c`、HEAD `3710ab21`)。16 ファイル、+1279/-21。Go は `internal/org/send_paste.go`(新規)、`verbs.go`、`send_paste_test.go`(新規)、`send_fallback_test.go`、`pane_lock_test.go`、`internal/cli/org.go`(help の 1 文)。run-test.sh は既定の changed scope で回った(`==> Language scope: changed (changed_languages)`、選ばれた pack は golang だけ)
- Evidence: `docs/evidence/test-2026-11-13-send-claude-paste.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力のあとに、30 秒ごとの load、変異 7 個の diff と出力、coverage の集計、CLI の probe の台本と出力を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-13-051454.log`
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、Bash ツールの `run_in_background` で scratchpad の runner から起動し、runner が開始・終了・rc を meta ファイルに書いた。終わりは `caffeinate -w <pid>`(1 回 175 秒まで)を繰り返して待った。回したのは 1 回だけで、回し直しはしていない。終わったあと、runner、load の見本取り、`go test` の process が残っていないことを `pgrep` で確かめた
- AC5(分離した herdr での実機の確かめ)は orchestrator が済ませた(`docs/evidence/org-send-paste-live-2026-11-13.md`)。この run では herdr、claude、codex を起動していない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,565 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 980.5 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 842.5 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/upgrade`、`internal/scaffold`、`internal/org/protocol`、`internal/org/driver`、`internal/insights`、`cmd/ralph` | パッケージ全体 | ok。`(cached)` は 0 | 0 | — | 128.5、2.7、2.5、2.3、1.6、1.4、1.3、0.7 秒 |
| 変異: production code の 7 個(下の「変異」) | 7 | 7 個とも死んだ | — | — | 1 個 13〜35 秒(build 込み) |
| coverage: org を `-run 'TestOrgSend_\|TestSendTextForDriver\|TestCheckNoPasteMarkers\|TestOrgSendScreen_\|TestOrgResume_SendWaits' -coverprofile` | 名前で絞った回 | ok | 0 | — | 27.5 秒(build 込み 34 秒) |
| CLI の probe: この branch で build した `ralph` の `org send`(herdr なし) | 4 | 4(期待どおり) | 0 | 0 | 数秒 |

新しい `internal/org/send_paste_test.go` はトップレベル 7 本、subtest 38 個(8 + 10 + 11 + 5 + 0 + 2 + 2)で、run-test.sh の org の回で通った。期待を書き換えた `send_fallback_test.go` の 2 本と `pane_lock_test.go` の 2 本も同じ回で通った。run-test.sh は `go test` を `-v` なしで回すので、件数はテストのコードから数えた。

Skipped の 2 行は、`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがあると起動してしまうため)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらも手元の環境によるもので、この diff と関係がない。前の run(watch-usage-limited)と同じ 2 行である。

30 秒ごとに 53 回見た 1 分の load は、shell の段(14:24 ごろまで)で 4.4 から 15.6、Go の段で 4.5 から 8.4 だった。shell の段の最大 15.6 は、自分の変異の回と重なった 14:20 ごろの値である。Go の段と重なった自分の回は、34 秒の絞り込みの coverage の回だけである。org は 980.5 秒で、golang の pack の上限(1 パッケージ 30 分、`packs/languages/golang/verify.sh` の `-timeout=30m`)の 54% だった。空きディスクは run の前後で 75 GiB と 74 GiB で、`no space left` は 0 行だった。

## Coverage

- Statement: 絞り込みの回で、`sendTextForDriver` と `checkNoPasteMarkers`(`send_paste.go`)はどちらも 100.0%、`Send`(`verbs.go:446`)は 96.4%。この diff で足した実行文の行は `verbs.go` が 4 行中 4 行(印の確かめの 3 行と `typeAndSubmit` の呼び出しの行)、`send_paste.go` が 8 行中 8 行で、すべて通った(`git diff -U0 origin/main...HEAD` の足した行と coverage の profile の block を突き合わせた)
- Branch: Go のツールでは測っていない。分かれ道は `driver != "claude"` と `strings.Contains` の 2 本で、どちらの向きもテストの行がある(下の対応表と変異)
- Function: 新しい関数 2 つはどちらも通った
- Notes: run-test.sh は coverage を出さない。パッケージ全体の数字は測っていない(絞り込みの回の 14.3% は、名前で絞ったテストだけの値)

## 受け入れ条件・Test plan とテスト

| 項目 | テスト | 結果 |
| --- | --- | --- |
| AC1 herdr の道: claude の座席には印で挟んだ文を 1 回の `PaneSendText` で打つ | `TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude` の claude の 2 行(短い文と 800 バイトを超える文)。打った文は印付きの 1 回、Enter は 1 回 | pass |
| AC1 画面の道 | `TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste`(短い文、`[Pasted text #1 +20 lines]` の置き換え)、`TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen/claude`、`TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed` の claude の 3 行 | pass |
| AC2 codex、空、ほかの値の座席には素の文 | `TestSendTextForDriver_WrapsOnlyClaude`(codex、空、`gemini`、大文字の `Claude`)、herdr の道のテストの codex、空、`gemini` の行、`send_fallback_test.go` の codex の行(画面の道) | pass |
| AC3 `sent` の Details、dry-run の Details、画面の道の Enter の前の確かめは印のない文 | herdr の道のテストの Details の完全一致と ESC の不在、`TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers`、画面の道の `via=screen herdr_state=working ` に元の文が続く Details と、元の文の入力欄で Enter が押されること | pass |
| AC4 印を含む文は driver によらず、`--raw` でも dry-run でも断り、何も打たず何も記録しない | `TestOrgSend_PasteMarker_RefusedWithNothingSent` の 11 行、`TestCheckNoPasteMarkers_RefusesEitherMarkerAnywhere` の 10 行。CLI の probe でも、`--raw --dry-run`、`--raw`、DECISION の 3 つが終了コード 1 と同じエラーで断られ、manifest に何も書かれなかった | pass |
| AC7 claude の座席で `PaneSendText` が失敗すると Enter を押さずに `SendProgressTextUnacknowledged` | `TestOrgSend_ClaudeSeat_SendTextFails_NoEnter` の herdr の道と画面の道。送ったのは `text pane-1` だけ、試した文は印付きの 1 回、新しい記録なし、pane のロックは空いた | pass |
| Edge: 印の片方だけ、文の途中の印、改行のない短い文 | 断りの表の片方だけの行と途中の行、`TestSendTextForDriver_WrapsOnlyClaude` の `TYPE: STOP` と空文字の行 | pass |
| Regression: 打った文を見ている既存のテスト | `send_fallback_test.go` の 2 本と `pane_lock_test.go` の 2 本。claude の行だけ期待が印付きになり、codex の行は `typedForTest` で素の文のまま | pass |
| Regression: `internal/cli` の `ralph org send` のテストは変わらない | `internal/cli` はテストのコードに変更がなく、パッケージ全体が ok(842.5 秒)。`org_test.go`、`org_resume_test.go`、`org_send_screen_fallback_test.go` は send-text の行を pane までで切るか接頭辞で見ている | pass |
| AC5、AC6 | テストの対象ではない。AC5 は orchestrator の実機の記録、AC6 は `/verify` の範囲 | — |

## 変異

本体を overlay(`go test -overlay`)で 1 か所ずつ書き換え、上の絞り込みと同じ名前の集合で回した。worktree のファイルは書き換えていない(回のあとの `git status --porcelain` は空)。

| # | 書き換え | 結果 |
| --- | --- | --- |
| V1 | 画面の道の確かめに印付きの文を渡す(`sendScreenSubmitChecks(..., sendTextForDriver(seat.Driver, p.Text))`) | 死んだ。新しい画面の道のテストの短い文の行、`HerdrWorkingAndScreenWaiting/claude`、`MultiLineMessageTallerThanTheBox` の claude の 2 行が落ちた |
| V2 | `sent` の Details に印付きの文を使う | 死んだ。新しい herdr の道と画面の道のテスト、既存の `TestOrgSendScreen_*` の 6 本と `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` |
| V3 | dry-run の Details に印付きの文を使う | 死んだ。`TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers` と既存の `TestOrgSend_DryRun_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` |
| V4 | `--raw` のとき印の確かめを飛ばす | 死んだ。断りの表の raw の 4 行 |
| V5 | dry-run のとき印の確かめを飛ばす | 死んだ。断りの表の `dry run` と `raw dry run` |
| V6 | 開始の印を見ない(終わりの印だけ確かめる) | 死んだ。`TestCheckNoPasteMarkers` の開始の印の 2 行と、断りの表の 4 行 |
| V7 | 印の確かめを protocol の検査より前に動かす | 死んだ。断りの表の `an invalid message with a marker` の 1 行だけが落とした |

V1 で、新しい画面の道のテストの置き換えの行(`[Pasted text #1 +20 lines]`)は単独では落ちない。claude の置き換えの確かめは文の改行の数を比べ、印は改行を足さないので、印付きの文でも通る。同じテストの短い文の行と既存の 3 行が落とすので、穴ではない。V7 を落とすのが 1 行だけなのは、計画の D3 の順(protocol の検査が先)をその行が固定しているためで、意図どおりである。S1 の implementer の 6 個の変異は中身が記録にないので、ここでは別に数えた。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh で落ちたテストはなく、load に弱いことが分かっているテスト(tester の記録の一覧)も今回は落ちなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #319: claude の座席に約 1 KB を超える文を素で打つと頭が欠ける | 単体テストでは再現できない(端末の挙動)。打つ文が印付きになることはテストで、届いた文がバイト単位で同じことは AC5 の実機の記録で確かめた | `TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude`、`docs/evidence/org-send-paste-live-2026-11-13.md` |
| `ralph org resume` の決まった文(`ResumeText`)は印で挟まない(計画の Non-goals) | 変わらない | `TestOrgResume_SendWaitsForAResumeHoldingThePaneLock`(claude の座席で、1 つ目の打った文が `ResumeText` と完全一致) |
| 画面の道の Enter の前の確かめと Enter のあとの確かめ(#322 の側の規則) | 変わらない | `TestOrgSendScreen_*` がすべて通った。V1 と V2 で、確かめと Details が元の文に縛られていることを確かめた |
| `PaneSendText` の失敗で Enter を押さない向き | 変わらない | `TestOrgSend_ClaudeSeat_SendTextFails_NoEnter` |

## Test gaps

1. CLI を通した印の断りと help の 1 文を固定する自動テストがない(verify の報告と同じ指摘)。今回は手で確かめた: HEAD から build した `ralph` を `env -i`、PATH に herdr のない状態、scratch の `--state-dir` で回し、印を含む 3 つ(`--raw --dry-run` の終わりの印、`--raw` の開始の印、DECISION の終わりの印)は終了コード 1 で `org: send: the message contains a bracketed paste marker (ESC [200~ or ESC [201~), ...` を出し、manifest に何も書かなかった。印のない `--raw --dry-run` の対照は終了コード 0 で、manifest に 1 行書いた。`--help` には新しい 1 文が出た。足すなら、`internal/cli` に `ralph org send --raw --dry-run` で `\x1b[201~` を含む文を渡し、終了コード 1、エラーの接頭辞、manifest が空であることを見る 1 本が最小
2. 端末の側の挙動は単体テストの外にある: Claude Code 2.1.316 と herdr 0.7.5 のほかの版、server が pane に書いている最中に止まった打ち込み、開いた貼り付けへの Enter、10 KB を超える `--raw`、8 ビットの CSI(U+009B)。どれも `docs/tech-debt/README.md` の #319 の行の (a)〜(f) にある。AC5 の実機の確かめは orchestrator のもので、この run ではやり直していない
3. codex の座席の長い文が 0.75 秒後の Enter で送られない件は計画の Non-goals(issue #326)で、テストもない
4. `-race` は回していない。この diff は goroutine もロックの扱いも変えていない(`Send` が `typeAndSubmit` に渡す文が変わっただけ)

## Verdict

- Pass: run-test.sh(rc 0、shell 45 本と Go 10 パッケージ、落ちたテスト 0)。AC1〜AC4 と AC7、Test plan の unit、integration、regression、edge の各項目。変異 7 個はすべて死んだ。変更行の coverage は 12 行中 12 行
- Fail: 0
- Blocked: なし。CLI を通した断りの自動テストがないこと(gap 1)と端末の側の未確認の挙動(gap 2)は残るが、PR を止めるものではない
