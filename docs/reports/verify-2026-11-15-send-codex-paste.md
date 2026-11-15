# Verify report: send-codex-paste

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-send-codex-paste.md`(issue #326)。`- Approved:` の digest `sha256:f17371b28b9f` は、`./scripts/plan-visual.sh digest` で計算した今の計画の digest と一致した
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は 1)
- Scope: `git diff 3c75e2e0...HEAD`(HEAD `7ecd4498`。取り込みのマージ `1f3146f5` は変更に入れていない)。28 ファイル、1176 行追加、96 行削除。Go の本体は 4 ファイル(`send_paste.go`、`send_screen.go`、`send_fallback.go` と `verbs.go` は doc comment だけ)、テストは新規 1(`send_codex_paste_test.go`)と既存 6、fixture 5、文書は規則 2、skill 4、`AGENTS.md`。self-review のあとの `7ecd4498`(L1、L2、L4、L5、L7 の直し)も範囲に入れた。テストは読んだだけで、回していない(`/test` の範囲)。AC5 の実機の確かめはやり直していない
- Evidence: `docs/evidence/verify-2026-11-15-send-codex-paste.log`(gitignore の対象。`run-static-verify.sh` が自分で書いた写しは `docs/evidence/verify-2026-11-15-124924.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: codex の座席には印で挟んだ文を 1 回の `PaneSendText` で打つ(herdr の道と画面の道)。claude は今のまま。空やほかの driver には素の文 | Met | `send_paste.go:36-41` は driver が `"claude"` か `"codex"` のときだけ印で挟み、ほかは文をそのまま返す。`Send` が打つ文は `verbs.go:621` の `sendTextForDriver(seat.Driver, p.Text)` の 1 つだけで、herdr の道と画面の道で同じ(画面の道で変わるのは `beforeEnter` と `confirm`)。テスト: `TestSendTextForDriver_WrapsClaudeAndCodex`(codex の短い文、複数行、空の文、大文字の `Codex`、空の driver、`gemini`)、`TestOrgSend_HerdrPath_TypesABracketedPasteForClaudeAndCodex`(codex の短い文と 800 バイト超、空と `gemini` は素の文)、画面の道は `TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen` の codex の行(`typedForTest` が codex も印付きにした)、`TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed`(`send_fallback_marker_test.go:42`)、AC4 の結合テスト。resume は `resume.go:518` で `ResumeText` を `typeAndSubmit` に直接渡すので素の文のまま(Non-goals のとおり) |
| AC2: `sent` と dry-run の Details、画面の道の確かめは、codex でも印のない文を使う | Met | dry-run の Details は `verbs.go:485`、`sent` の Details は `verbs.go:668` で、どちらも `truncateForDetails(p.Text)`。画面の確かめは `verbs.go:620` の `sendScreenSubmitChecks(screenPath, seat, p.Text)`。テスト: herdr の道のテストが Details と元の文の一致と ESC がないことを codex の行でも見る。`TestOrgSend_DryRun_CodexSeat_DetailsWithoutMarkers`、`TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed`(Details が `via=screen herdr_state=working ` に元の文が続く形と等しい)。`TestSendInputHolds_CodexPlaceholderRefuses` の `p4 typed with the markers counted` の行は、印付きの文を渡すと規則 4 が通らないことを固定する |
| AC3: 実機の置き換えの画面で `sendInputHolds` が true、N が違う・ほかに文がある・置き換えが 2 つで false、p1・p2 で true のまま、Enter から 0.3 秒後の画面で `SendScreenSubmitted` が true、置き換えが残った画面で false | Met | 本体: `send_screen.go:322` の codex の分岐に規則 4 を足し、`sendCodexPasteHolds`(`:487`)は一致が `got` の先頭から末尾までの 1 つで、数が `utf8.RuneCountInString(typed)` と等しいときだけ true。Enter のあとは `sendInputHasPastePlaceholder`(`:558`)が codex の置き換えを数によらず「まだ入力欄にある」と読み、`SendScreenSubmitted`(`:249`)は false。テスト: `TestSendInputHolds_CodexPastedFixtures`(実機の 4 画面で true、規則 4 が通るのは置き換えの 2 画面だけ、`SendScreenSubmitted` は false、`codex-submitted-pasted.txt` で true)、`TestSendInputHolds_CodexPlaceholderRefuses`(N が ±1、UTF-16 の 2017、バイトの 4925、置き換え 2 つ(空白あり・なし)、後ろや次の行や前に文、claude の形、別の文の画面の 14 行)、`TestSendScreenSubmitted_CodexPlaceholderStillInTheBox`(working の行の下でも置き換えが残れば false、空の入力欄が対照) |
| AC4: 画面の道の `Send` を通した結合テスト(codex、日本語 2000 字の本文、実機の置き換えの画面と Enter のあとの画面) | Met | `TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed`(`send_codex_paste_test.go:272`)。文は p4(`TYPE: DECISION` と空行の 16 字、本文 2000 字、全体 2016 字)。herdr の呼び出しの列に `keys pane-1 Enter` が 1 回、打った文は `pastedForTest(msg)` の 1 回、`SubmitConfirmed` が true、Details に `submit_unconfirmed` も ESC もない。否定の側は `TestOrgSendScreen_CodexPastePlaceholderOfAnotherSize_EnterNotPressed`(数が 2017 の画面で Enter を押さず、`SendProgressTextTyped`、何も記録しない、ロックが空く)。p4 の文が実機で打った文と同じことは `TestSendCodexPasteProbeMessages_AsTyped` が SHA-256 で固定する |
| AC5: 分離した herdr の codex の座席に、この branch の `ralph` で日本語 2000 字の DECISION を 1 回送る | Met(記録と生データで確かめた。実機はやり直していない) | `docs/evidence/org-send-codex-paste-live-2026-11-15.md`。orchestrator の scratchpad の生データ `s3-326/` を読み直した。`msg.txt` と `user-entry.txt` はどちらも 3938 バイトで sha256 `f4ba222f…b342210` が一致し、user の文に ESC はない。文は 2016 字で、`TYPE: DECISION` と空行のあとの本文が 2000 字・29 行。`sent.jsonl` は `sent` が 1 件で、`submit_unconfirmed` も `via=` も ESC もない(herdr の道)。`send.out` は `sent message to seat "worker"`。`manifest.jsonl` の `sent` は 1 件だけ。`after.txt` に `• 受信326` と `Worked for 5s` が 1 回ずつある。使った build は 4403bef4 で、`git diff 4403bef4 HEAD` の本体の Go はコメントだけ(テストは名前とコメント)なので、確かめた動きは HEAD と同じ。claude の座席を使わなかったことは記録の記述による |
| AC6: `agent-messaging.md` の 2 つの写し、`AGENTS.md` の行、`/org` skill の `send` の行が、codex にも貼り付けで打つことと codex の置き換えの形を書いている | 一部(字義) | `agent-messaging.md:115-127`(2 つの写しは `cmp` で同じ)は、codex にも印で挟んで打つこと、`[Pasted Content <N> chars]` と N が文の字数であることを書く。`/org` skill の `send` の行(4 つの写しは `cmp` で同じ)も両方を書く。`AGENTS.md` の `internal/org/` の行は「types a message to a claude or codex seat between the markers」を書くが、codex の置き換えの形は書いていない(V1) |

issue #326 の受け入れ条件のうち「1 回の Enter で送られ、全文が届く(実機)」は AC5、「画面の道でも Enter を押せる。押せない形なら断る」は AC3、AC4 と D3 で見た。D3 のとおり大きさで断る規則は足していない。確かめでは本文 2000 字までのどの大きさも Enter で送られた(probe の 5 つと実機の 1 つ。#319 の証拠の X2、5341 バイトの貼り付けも送られた)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の scope) | PASS(rc=0) | 求めた scope は changed だが、`internal/org/testdata/resume/*.txt` が分類のない path なので language scope は full に落ち、Go の pack が選ばれた。hook の shellcheck、`sh -n`、awk の構文、`jq -e`: OK。`check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0)。`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照: OK。gofmt: ok。`go vet ./...`: 出力なし。golangci-lint: 0 issues。staticcheck: 出力なし(`Skipping` の行はない)。`secret-scan-branch` の `3c75e2e0..7ecd4498`: clean |
| `cmp` で `agent-messaging.md` の 2 つの写し、`/org` skill の 4 つの写し | 同じ | AC6 |
| `cmp` で fixture 5 つと scratchpad の `probe326/out/` | 5 つとも同じ | `codex-input-pasted-placeholder.txt` = `p4-japanese.typed.txt`、`-ascii` = `p3-long.typed.txt`、`-short` = `p1-short.typed.txt`、`-medium` = `p2-medium.typed.txt`、`codex-submitted-pasted.txt` = `p4-japanese.enter03.txt`。U+FFFD は 0 |
| `shasum -a 256` で `probe326/` の文 4 つ | テストの定数と一致 | p1 105、p2 651、p3 1698、p4 4925 バイト。`TestSendCodexPasteProbeMessages_AsTyped` の SHA-256 と同じ |
| `./scripts/plan-visual.sh digest` | `f17371b28b9f` | 計画の `- Approved:` の行と一致 |
| `git diff 4403bef4 HEAD -- 'internal/org/*.go' ':!*_test.go'` | コメントの行だけ | AC5 の build のあとに本体の動きは変わっていない |
| `git merge-tree --write-tree --name-only origin/main HEAD` | `AGENTS.md` が衝突 | origin/main は base のあとに #333(org-reseat、`24e6723c`)が入った。V8 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `sendTextForDriver` と `checkNoPasteMarkers` の doc(`send_paste.go`) | Yes | codex の X1(5340 バイトの素の文が 750 ms 後の Enter で送られない)と、probe の 5 つの大きさを観測のまま書く(self-review L1 の直しのあと) |
| `sendInputHolds`、`sendCodexPasteHolds`、`sendCodexPastePlaceholder`、`sendInputHasPastePlaceholder`、ファイル冒頭の注(`send_screen.go`) | Yes | 規則 4 の条件(規則 1 と 2 のあと、置き換え 1 つだけ、数は字数)が本体と合う。畳まれ始める大きさと書記素の数を確かめていないことも書く |
| `SendScreenSubmitted` の doc | Yes | 置き換えは数によらず「まだ入力欄」と読むことが本体と合う |
| `sendScreenSubmitChecks` の doc(`send_fallback.go:289-294`) | Yes | self-review L2 の直しで codex の置き換えが入った |
| `Send` の doc と本体の注(`verbs.go:284-294`、`:386-396`、`:615-619`) | Yes | |
| `agent-messaging.md` の 2 つの写し | Yes(V4 を除く) | |
| `/org` skill の `send` の行(4 つの写し) | Yes(V6 は INFO) | |
| `AGENTS.md` の `internal/org/` の行 | 一部 | V1 |
| org runtime の spec(`docs/specs/2026-08-01-org-runtime.md`) | No | V3 |
| tech-debt の #319 の行(503 行目)の (e) と、この PR の Known gaps の行 | No | V2 |
| tech-debt の、この PR が Trigger を満たした行 | No | V5 |
| 計画の Affected areas の `pane_lock_test.go` | INFO | V7 |

指摘。V1 は AC6 の字義に関わる。どれも文書だけで、`/sync-docs` で直すかを決める。V2 は `/pr` の前に済ませる。

- V1(LOW、AC6): AC6 は `AGENTS.md` の行にも codex の置き換えの形を書くことを求めるが、行は「the bracketed paste of send (`send_paste.go`: `ralph org send` types a message to a claude or codex seat between the markers …)」までで、`[Pasted Content <N> chars]` も、Enter の前の確かめがその数を文の字数と比べることも書いていない(この行は claude の置き換えも書いていない)。画面の道の一節(`sendInputHolds` for the check before Enter)に「codex's one paste placeholder, `[Pasted Content <N> chars]`, whose N must be the text's length in runes」程度の 1 句を足せば満たす。AC の文を狭める直しは計画の digest を変えるので、足すほうが安い
- V2(MEDIUM): 計画の Affected areas は `docs/tech-debt/README.md`(この PR の Known gaps と #319 の行の (e))を挙げ、Risks と D3 は「Known gaps に残す」と書くが、tech-debt は diff にない(self-review L3)。(1) #319 の行の (e) 「The marker refusal applies to every driver, codex included, although only a claude seat gets the paste」は、この PR で codex も貼り付けを受けるので成り立たない。Impact と Trigger の (e) も合わせて取り消し線で閉じる。(b) の「Claude Code closes the open paste by itself after about 2 seconds」と (f) の 8 ビットの印も claude だけの記述で、codex では確かめていない。(2) #322 の行(507 行目)の (f) 「A codex seat still gets the message as plain text; a bracketed paste for codex is #326」は Trigger が `#326` で、この PR がそれを満たし、払った。取り消し線で閉じる。(3) 足すべき Known gaps の行の項目: 確かめたのは codex-cli 0.166.0 と herdr 0.7.5 だけ(置き換えの形が変わると画面の道は Enter を押さずに止まる)、畳まれ始める大きさ(651 と 1698 バイトのあいだ)、N の数え方(符号位置と書記素)、貼り付けが途中で切れたときの codex の入力欄、本文 2000 字を超える `--raw` の文と 10 KB 超、前の置き換えが残る入力欄への貼り付け、画面の道を ralph で通しで走らせていない(実機は 15 秒で `idle` になり herdr の道を通った)、Ctrl+C で codex の入力欄が空になるか(V4)、8 ビットの終わりの印、`send_screen.go` が 788 行で 800 行の目安に近い(self-review L6。base は 735)
- V3(LOW): spec には `ralph org send` を変えた PR ごとに日付の改訂の節がある(94 行目の「2026-11-13 改訂(fix/send-claude-paste)」、101 行目の「2026-11-13 改訂(fix/send-input-box-marker)」)が、この PR の節がない。前の節の文も古くなった。96 行目 (a) の「driver が claude 以外(codex、空、ほかの値)の座席には、本文を今までどおりそのまま打つ。codex の座席の長い本文が 0.75 秒後の Enter で送られない件は、この改訂の外である(issue #326)」、98 行目 (c) の「claude の座席への send が本文を打つあいだに失敗したとき」、103 行目 (a) の「今の 3 つの規則(頭、終わり、claude の placeholder)」(今は 4 つ)。新しい節を足し、前の節の該当箇所に改訂の印を付けるのが前例に合う
- V4(LOW): `agent-messaging.md:133-146` の段落は、送信の失敗のあとの手順を「a claude or codex seat」に広げた。続く「The human presses Ctrl+C once in the pane …, which clears the input box」も codex の座席に読めるが、Ctrl+C で入力欄が空になったのは claude の観測だけ(`docs/evidence/org-send-paste-probe-2026-11-13.md:202`、`:217`)。codex の probe と実機の記録に Ctrl+C の確かめはない。codex では確かめていないと書くか、Known gaps に入れる
- V5(LOW): Trigger を満たしたのに持ち越しの注がない行。行は base と同じ(tech-debt は diff にない)。(1) 180 行目 (a)「The next edit to the `/org` skill's command table」: `send` の行を 4 つの写しで直した(表の 1 行が 4,086 字から 4,273 字。改行を除いて `3c75e2e0` と HEAD で数えた)。(2) 226 行目 (b)「the next change to `Send`」: `Send` の doc と本体の注を直した。(3) 365 行目「The next change to `Send`」: 同じ。`verbs.go` は 2615 行から 2619 行(+4)。(4) 460 行目「the next probe of codex's screens」: この PR の probe と実機で codex-cli 0.166.0 の画面を取った。footer は前と同じ形(`GPT-6-Luna · low · Context 7% used · main · No cha…` と `⚠ 11 · f2`)で、`% left` の形は出ていない。(5) 482 行目 (L2, L7, L9, L10)「the next change to `send_fallback.go`, `send_screen.go`」: 両方を直した。(6) 484 行目「The next change to either file」: `send_screen_test.go` に fixture の表の 5 行(867 行から 878 行)、`send_fallback_test.go` は失敗の文言を 2 行(835 行のまま)。数え方は `wc -l` を `3c75e2e0` と HEAD で。(7) 506 行目「the next test added to `usage_limit_screen_test.go`」: `usageLimitFixtureWants` に 5 行(598 行から 606 行)。#322 は同じ形の行の追加を「満たした」と注に書いた。どれも計画の Scope の外なので、払うのではなく「carried over by fix/send-codex-paste」の注を足すのが前例に合う。483 行目と 473 行目は調べたが、名指しの関数(`sendTypedInputBox`、`sendCodexHasFooter`、`checkSendScreen` など)も番号付きの TASK の送信もこの PR にないので満たしていない
- V6(INFO): `/org` skill の `send` の行は「N が本文の文字数」と書くが、同じ行の「本文 2,000 文字上限」の「本文」は protocol の本文(空行のあと)を指す。N は文全体の字数で(p4 は本文 2000 字、N は 2016)、`agent-messaging.md` の「the message's length in characters」はそのとおり書いている。skill は「文全体の文字数」と書くと読み違えがない
- V7(INFO): 計画の Affected areas は `internal/org/pane_lock_test.go` の codex の期待を直すと書くが、このファイルは変わっていない。`h.typed()` を見る 2 か所(518 行目、615 行目)はどちらも claude の座席で、すでに `pastedForTest` を使う。codex の座席の打った文を見る行がないので、直すものがなかった
- V8(INFO): base のあとに origin/main に #333(org-reseat)が入り、`git merge-tree` で `AGENTS.md` の `internal/org/` の行が衝突する(reseat の一節と、この PR の「or codex」)。`verbs.go` と `/org` skill の 4 つの写しは自動で merge できる。main の新しい文(reseat の spec、skill、`reseat.go`)に貼り付けや `send-text` の記述はなく、この PR の動きと食い違う文は入ってこない。`/pr` の前に origin/main を取り込み、両方の文を残して解く

## Observational checks

- AC5 の生データ(orchestrator の scratchpad の `s3-326/`: `msg.txt`、`user-entry.txt`、`sent.jsonl`、`manifest.jsonl`、`send.out`、`after.txt`)を `python3 -I` で読み直し、上の表の数字を出した。生データは repo に入っていない
- fixture 5 つは scratchpad の `probe326/out/` の画面とバイト単位で同じ。probe の文 4 つの SHA-256 はテストの定数と同じ。fixture の画面は読んだ(置き換えの 1 行、Enter のあとの空の入力欄と `• Working (0s • esc to interrupt)`、p2 の `↑` 付きの 1 行目)
- `docs/evidence/org-send-paste-probe-2026-11-13.md` の X1(5340 バイトの素の文)と X2(5341 バイトの貼り付けが `[Pasted Content 2147 chars]` になり送られた)を読み、`send_paste.go` の doc の X1 の記述と比べた

## Coverage gaps

- テストは回していない(指示と skill の範囲)。AC1〜AC4 は本体とテストの assertion を読んで判定した。通るかは `/test` で見る
- 画面の道を実機で通しで走らせていない(実機は herdr の道を通った)。画面の道の codex の規則は、実機の画面の fixture を読むテストだけが固定している。#308 と同じく、herdr が `working` と言い続ける状態は作れていない
- N の数え方は、絵文字 1 つ(符号位置 1、UTF-16 の 2 単位)を含む文で UTF-16 との違いだけを確かめた。結合文字や ZWJ の絵文字で符号位置と書記素が違う文は未確認。違えば規則 4 は通らず、Enter を押さずに止まる(知らせる側)
- 既存のテストの期待が codex の座席で素の文のまま残っている箇所がないかは、`h.typed()` を見る 6 ファイルを読んで確かめた。CLI(`internal/cli`)のテストは codex の座席の打った文を見ていない
- 静的検査の欠けはない。足すなら、`agent-messaging.md` の Ctrl+C の段落と `/org` skill の置き換えの文を読むテスト(V4 と V6 の文言の固定)が最小だが、どちらも文書の判断が先

## Verdict

- Verified: AC1〜AC4 の本体とテストの対応(file:line とテスト名)、AC2 の 3 つの記録と確かめが印のない文を使うこと、fixture 5 つのバイト一致と文 4 つの SHA-256、AC5 の送った文と届いた文のバイト一致、`sent` の記録の中身、返事の語と手番の数(生データの画面と manifest)、AC5 の build のあとに本体の動きが変わっていないこと、規則と skill の写しの一致、`run-static-verify.sh` の全項目、計画の承認の digest
- Partially verified: AC6(`AGENTS.md` の行に codex の置き換えの形がない、V1)。AC5 の claude の座席を使わなかったこと(記録の記述だけ)
- Not verified: テストの実行結果(`/test`)。画面の道の実機での通しの実行。codex-cli 0.166.0 と herdr 0.7.5 のほかの版。書記素と符号位置が違う文の N
- Verdict: partial-pass。AC1〜AC5 は満たし、静的検査はすべて通った。AC6 は `AGENTS.md` の行に 1 句が足りない(V1、LOW)。文書の指摘は V1、V2(MEDIUM、この PR の Known gaps の行と #319 の (e)、#322 の (f))、V3(LOW、spec の改訂の節)、V4(LOW、codex の Ctrl+C)、V5(LOW、Trigger を満たした tech-debt の 7 行)、V6〜V8(INFO)。どれも `/sync-docs` で直し、V1 と V2 は `/pr` の前に済ませる。V8 の `AGENTS.md` の衝突は `/pr` の前の取り込みで解く。insight event の `verdict` には partial の値がないので `pass` と記録した(`/pr` の前提は pass か partial-pass)
