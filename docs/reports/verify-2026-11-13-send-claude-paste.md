# Verify report: send-claude-paste

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-send-claude-paste.md`(issue #319)。`- Approved:` の digest `sha256:2ad0b91591a9` は、`./scripts/plan-visual.sh digest` で計算した今の計画の digest と一致した
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は 1)
- Scope: `git diff bb416c48...HEAD`(HEAD `8509497c`)。15 ファイル、1205 行追加、21 行削除。Go は 6 ファイル(`internal/org/send_paste.go`、`verbs.go`、`send_paste_test.go`、`send_fallback_test.go`、`pane_lock_test.go`、`internal/cli/org.go`)。テストは読んだだけで、回していない(`/test` の範囲)。AC5 の実機の確かめはやり直していない
- Evidence: `docs/evidence/verify-2026-11-13-send-claude-paste.log`(gitignore の対象。`run-static-verify.sh` が自分で書いた写しは `docs/evidence/verify-2026-11-13-045119.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: claude の座席には印で挟んだ文を 1 回の `PaneSendText` で打つ。herdr の道でも画面の道でも同じ | Met | `verbs.go:617` が `typeAndSubmit` に渡す文は `sendTextForDriver(seat.Driver, p.Text)` の 1 つだけ。`typeAndSubmit` は `PaneSendText` を 1 回呼ぶ(`verbs.go:849`)。画面の道で変わるのは `beforeEnter` と `confirm` だけで、打つ文は同じ。`send_paste.go:30-35` は driver が `"claude"` のときだけ印で挟む。テスト: `TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude`(claude の短い文と 1319 バイトの文で、打った文が印付きの 1 回)、`TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste`(`sendFallbackOrg` が herdr を `working` にするので、打つのは画面の道だけ)。既存の `send_fallback_test.go` の 2 つと `pane_lock_test.go` の 2 か所は、claude の行の期待を印付きに直した |
| AC2: codex、空、ほかの値の座席には素の文 | Met | `driver != "claude"` なら文をそのまま返す(`send_paste.go:31`)。`TestSendTextForDriver_WrapsOnlyClaude` に codex、空、`gemini`、大文字の `Claude` の行がある。herdr の道のテストに codex、空、`gemini` の行がある。`send_fallback_test.go` の codex の行は `typedForTest(tc.driver, …)` で素の文のまま |
| AC3: `sent` の Details、dry-run の Details、画面の道の Enter の前の確かめは印のない文を使う | Met | dry-run の Details は `truncateForDetails(p.Text)`(`verbs.go:482`)、`sent` の Details は `screenPath.details()` のあとに `truncateForDetails(p.Text)`(`verbs.go:664`)、画面の確かめは `sendScreenSubmitChecks(screenPath, seat, p.Text)`(`verbs.go:616`)。テスト: herdr の道で Details が元の文と等しく ESC を含まない、`TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers`、画面の道で `via=screen herdr_state=working ` に元の文が続き、入力欄に元の文(または `[Pasted text #1 +20 lines]`)が出た画面で Enter が押される。実機の `sent` の記録にも ESC はなかった(AC5 の生データ) |
| AC4: 印を含む文は driver によらず、`--raw` でも dry-run でも、何も打たず何も記録せず断る。エラーが印を示す | Met(エッジに注記) | `checkNoPasteMarkers(p.Text)`(`verbs.go:459`)は protocol の検査のすぐあと、TASK の見張りの確かめ、dry-run の分岐、`findSeat` より前にある。エラーの `errSendPasteMarker` は `ESC [200~ or ESC [201~` を名指す。`TestOrgSend_PasteMarker_RefusedWithNothingSent` の 11 行(開始と終わりの印、claude と codex、raw、dry run、raw の dry run、manifest にない座席、見張りの前の TASK)が、`SendProgressNothingSent`、manifest の件数が変わらないこと、2 つの fake herdr の呼び出しが 0 件、見張りが呼ばれないことを見る。`TestCheckNoPasteMarkers_RefusesEitherMarkerAnywhere` は文の頭、途中、終わりの印を断り、ESC やチルダが欠けた断片を通す。注記: `--raw` でない不正な文に印があると、D3 の順のとおり protocol の検査のエラーが先に返る(最後の行が固定している)。このときも何も打たず何も記録しない |
| AC5: 分離した herdr の claude の座席に、この branch の `ralph` で日本語 2000 字の DECISION を 1 回送る | Met(記録と生データで確かめた。実機はやり直していない) | `docs/evidence/org-send-paste-live-2026-11-13.md`。orchestrator の scratchpad の生データ `s3/` を読み直した。`msg.txt` は 4414 バイト、改行 40、本文 2000 字、sha256 `bf2a1da6…af217f7`。セッションの記録の user の文 `user-entry.txt` は `<pasted_content>` が 1 つで、その外は前の `\n\n` と後ろの `\n` だけ。開きの札と閉じの札のあいだは `msg.txt` とバイト単位で同じで、ESC を含まない。`sent.jsonl` は `sent` が 1 件で、`submit_unconfirmed` も `via=` も ESC もない。`send.out` は `sent message to seat "worker"`。実機で使った build は 331eebd3 で、`git diff 331eebd3 HEAD -- '*.go'` はコメントと `internal/cli/org.go` の help の文字列だけなので、確かめた動きは HEAD と同じ。返事の語「受信319」と手番が 1 回だったことは記録の記述によるもので、生データでは見ていない |
| AC6: `agent-messaging.md` の 2 つの写しが同じで、打ち方、断り、残りうる置き換えと Ctrl+C の消し方を書く。`AGENTS.md` と tech-debt が今の動きと合う | Met(隣の文書の食い違いは下の表の V1、V2、V4) | `cmp` で 2 つの写しは同じ。`agent-messaging.md:113` からの「How send types a message」に、claude の座席への打ち方、印を含む文の断り(`--raw` と `--dry-run` も)、`[Pasted text #N +M lines]` が残りうること、`herdr pane send-keys <pane_id> ctrl+c` を 1 回押すことがある。`:91` の `--raw` の文も断りの例外を書く。`AGENTS.md` の `internal/org/` の行に `send_paste.go` の一節がある。tech-debt の #308 の行の (c) は 4 つのセルで取り消され、RESOLVED の注がある。#319 の新しい行の (a)〜(f) は probe、実機の記録、計画と合う。`check-sync.sh` は PASS |
| AC7: claude の座席で `PaneSendText` が失敗すると Enter を押さず `SendProgressTextUnacknowledged` を返す | Met | `typeAndSubmit` は `PaneSendText` の失敗で Enter の前に戻り(`verbs.go:849-851`)、`Send` は記録を書かずにそれを返す(`verbs.go:620-630`)。`TestOrgSend_ClaudeSeat_SendTextFails_NoEnter` は herdr の道と画面の道の両方で、送ったのが `text pane-1` だけであること、試した文が印付きの 1 回であること、新しい記録がないこと、pane のロックが空いたことを見る |

issue #319 の受け入れ条件(2000 字までの文が頭を欠かずに届く。届かない大きさがあれば断る)は、AC5 と probe の B1、B2 で見ている。上限までで届かない大きさは見つからず、大きさで断る規則は足していない。計画の Verify plan のとおり。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | PASS(rc=0) | hook の shellcheck、`sh -n`、awk の構文、`jq -e`: OK。`check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0)。`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照: OK。gofmt: ok。`go vet ./...`: 出力なし。golangci-lint: 0 issues。staticcheck: 出力なし(`Skipping` の行はない)。`secret-scan-branch` の `bb416c48..8509497c`: clean |
| `cmp .claude/rules/ralph/agent-messaging.md templates/base/.claude/rules/ralph/agent-messaging.md` | 同じ | AC6 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-send-claude-paste.md` | `2ad0b91591a9` | 計画の `- Approved:` の行と一致 |
| `git diff 331eebd3 HEAD -- '*.go'` | コメントと help の文字列だけ | AC5 の build のあとに動きは変わっていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `Send` の doc comment、`SendParams.Raw` の doc(`verbs.go`) | Yes | 印で挟むこと、Details と画面の確かめは元の文、断りの位置と範囲が本体と合う |
| `send_paste.go` の doc | Yes | self-review L1 の直しのあと、1117 バイトの 1 回の観測をそのまま書いている。「約 1 KB の塊」と「後ろの塊の違いだけ」は issue #319 の本文(`gh issue view 319`)と合う |
| `agent-messaging.md` の 2 つの写し | Yes(V4 を除く) | コードと合う |
| `AGENTS.md` の `internal/org/` の行 | Yes | |
| `ralph org send --help` の `Long`(`internal/cli/org.go:1363`) | Yes | 断りの 1 文を足した。claude の座席に貼り付けで打つこと自体は help に出ない(INFO、計画は求めていない) |
| tech-debt の #308 の行の (c)(473 行目) | Yes | Debt、Impact、Why deferred、Trigger の 4 つのセルで取り消されている |
| tech-debt の #319 の新しい行(487 行目) | Yes | 版、途中で切れた打ち込み、開いた貼り付けへの Enter、`--raw` と 10 KB、codex への断り、8 ビットの CSI。probe と実機の記録の「確かめていないこと」と合う |
| `/org` skill の `send` の行(4 つの写し、`.claude/skills/org/SKILL.md:360` など) | No | V1 |
| org runtime の spec の改訂の記録(`docs/specs/2026-08-01-org-runtime.md`) | No | V2 |
| tech-debt の、この PR が Trigger を満たした行 | No | V3 |
| 実機の記録の行数の書き方 | 記録の文言 | V5 |

指摘(どれも LOW で、文書だけ。`/sync-docs` で直すかを決める)。

- V1(LOW): `/org` skill の動詞リファレンスの `send` の行(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 つ)は、見張りの確かめと計画の承認の断りを「`--raw` と `--dry-run` も含めて終了コード 1 で断られ、何も記録せず pane にも何も送らない」の形で並べているが、印を含む文の断りがない。claude の座席に bracketed paste で打つことも書いていない。「`--raw` で検証をバイパス」も、印の断りは `--raw` でも残るので、そのままでは広い。4 つの写しに 1 文ずつ足せば足りる(`scripts/sync-skills.sh` で同期)
- V2(LOW): `docs/specs/2026-08-01-org-runtime.md` には、`ralph org send` の動きを変えた PR ごとに日付の改訂の節がある(68 行目の「2026-11-12 改訂(fix/send-screen-fallback)」など)。この PR の変更(claude の座席への bracketed paste、印を含む文の断り)の節がない
- V3(LOW): この PR が Trigger を満たしたが、行が base と同じで、持ち越しの注がない行が 3 つある。(1) 365 行目「`internal/org/verbs.go` is 2277 lines」: Trigger は「The next change to `Send`」。`verbs.go` は 2597 行から 2615 行(+18。+300 の基準の 2597 の内)。分割はしていない。(2) 226 行目「Small findings of org-plan-approval」の (b): Trigger は「the next change to `Send`」。`internal/org/approval_test.go:945` の `approval gets` だけの行が残っている。(3) 484 行目「`internal/org/send_screen_test.go` (662 → 831 lines) and `internal/org/send_fallback_test.go`」: Trigger は「The next change to either file」で「first move」を求める。この PR は `send_fallback_test.go` の期待を 2 か所直した(835 行のまま)が、テストを移していない。どれも計画の Scope の外なので、払うのではなく「carried over by fix/send-claude-paste」の注を足すのが前例(#308 の同じ行の注)に合う。INFO: 184 行目「Four findings on the S8 compensation of org-stop-all」の C2-5(「`verbs.go` を次に直すときに block を移す」)も字義どおりには満たしたが、#308 などの前の PR も注を足していない
- V4(LOW): `agent-messaging.md:133-137` の Ctrl+C の手順は、誰が押すかを書いていない。`/director` skill の禁止事項(`.claude/skills/director/SKILL.md:2000`)は、director が座席の pane に文字やキーを送るのを `ralph org resume` と DECISION の送り方(`ralph org send` と、入力欄に残った DECISION への Enter 1 回)に限っている。この rule を読んだ director が、send の失敗のあとに Ctrl+C を押してよいかが 2 つの文書で食い違う。rule の側で押す人(人か、自分の座席に送った leader)を書くか、director の許可の一覧に足すかを決める
- V5(LOW、記録だけ): `docs/evidence/org-send-paste-live-2026-11-13.md:69` は「本文は 40 行」と書くが、生データでは空行のあとの本文は 39 行(番号付きの 38 行と最後の行)で、文全体が 41 行、改行が 40 個。バイト数、字数、改行の数、sha256 は合っており、AC5 の判定は変わらない

## Observational checks

- AC5 の生データ(orchestrator の scratchpad の `s3/`: `msg.txt`、`user-entry.txt`、`user-entry-inner.txt`、`sent.jsonl`、`manifest.jsonl`、`send.out`)を python で読み直し、上の表の数字を出した。生データは repo に入っていない
- `gh issue view 319` で issue の本文を読み、`send_paste.go` と `Send` の doc の観測の書き方と比べた

## Coverage gaps

- テストは回していない(指示と skill の範囲)。AC1〜AC4、AC7 は本体とテストの assertion を読んで判定した。テストが通るかは `/test` で見る
- `internal/cli` に、新しい help の 1 文と、CLI を通した印の断りを固定するテストがない(計画は `internal/cli` のテストを変えない方針)。足すなら、`ralph org send --raw --dry-run` に ESC `[201~` を含む文を渡して終了コード 1 とエラーの接頭辞 `org: send: the message contains a bracketed paste marker` を見る 1 件と、`org send --help` の部分文字列の確かめ 1 件が最小
- Claude Code 2.1.316 と herdr 0.7.5 のほかの版、server が書いている途中で止まった打ち込み、開いた貼り付けへの Enter、8 ビットの CSI は確かめていない。tech-debt の #319 の行の (a)〜(f) に残っている
- 返事の語と claude の手番の数は記録の記述だけによる(AC5 の注記)

## Verdict

- Verified: AC1〜AC4、AC6、AC7 の本体とテストの対応(file:line とテスト名)。AC5 の送った文と届いた文のバイト一致、`sent` の記録の中身、実機の build のあとに動きが変わっていないこと。`run-static-verify.sh` の全項目。計画の承認の digest
- Partially verified: AC5 の返事の語と手番の数(記録だけ)。AC6 は満たすが、隣の文書に V1、V2、V4 の食い違いがある
- Not verified: テストの実行結果(`/test`)。確かめた版のほかの Claude Code と herdr
- Verdict: pass。受け入れ条件 7 つはすべて満たした。静的検査はすべて通った。文書の指摘は LOW 5 件(V1: `/org` skill の `send` の行、V2: org runtime の spec の改訂の節、V3: Trigger を満たした tech-debt の 3 行、V4: Ctrl+C を押す人と `/director` skill の禁止事項、V5: 実機の記録の行数)で、どれも `/sync-docs` で直すかを決める
