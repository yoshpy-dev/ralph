# Walkthrough: 印で始まる行を含む文も、送る前の確かめで入力欄を見分ける(send-input-box-marker)

- 計画: `docs/plans/archive/2026-11-13-send-input-box-marker.md`(PR の作成のあとに archive へ移す)
- issue: #322
- 差分: `git diff c53bf39c...HEAD`(41 ファイル、約 +2,120 / −38)。Go の本体は +110 / −22、Go のテストは +596、fixture は 11 ファイルで +253、残りは prompt・skill・文書・記録・計画・報告。c53bf39c(#329、PR #330)は bffc6934 で取り込んだもので、この差分には入らない

## 1. 入力欄の頭の選び直し(読む価値が一番高い)

`internal/org/send_screen.go`。

- `sendTypedInputBox`(321 行): 送る前の確かめ(`sendInputHolds`)が使う入力欄を探す。最初に `sendTypedMarkerLineBox` を試し、外れたら今までの処理(`findResumeInputBox` と、番号付きの行の道)に進む。
- `sendTypedMarkerLineBox`(377 行): 計画の D2 の 5 つの条件をすべて満たすときだけ、入力欄の頭を選び直す。
  1. 下から最初の印の行 L の、印の前に空白がある。
  2. L の印のあとの文が空でなく、打った文の一部(`sendTextInWant`)。
  3. L より上に 1 文字目が印の行 M があり、あいだの行は空行か 2 字下げの行だけ。
  4. M から取った入力欄が入力欄の形をしている。claude では M の真上が罫線で、M の下の最初の罫線が L より下。codex では footer があり、L はその上。
  5. M の文が番号付きの行なら、その文も打った文の一部。
  知らない driver では箱を取らない。
- `sendTextInWant`(346 行): 「印の行の文(空白を除き、codex は右端の `↑` を除く)が打った文の一部か」を 1 か所にまとめた。番号付きの道と、条件 2・5 が使う。空の文は通さない。
- `sendCodexFooterStart`(495 行): codex の footer の始まりを求める。`sendCodexHasFooter` はこれを呼ぶだけになった。
- 変えていないもの: `findResumeInputBox` と resume の画面の型(`resume_screens.go`)、打つ前の確かめ(`ClassifySendScreen`)、Enter のあとの確かめ(`SendScreenSubmitted`)のコード。`SendScreenSubmitted` は `sendInputHolds` を呼ぶので、送れた画面で true、文が残った画面で false になることをテストで固定した。

## 2. 実機の画面と fixture

- `docs/evidence/org-send-marker-probe-2026-11-13.md`: 分離した herdr で、claude(#319 からの bracketed paste)と codex(素の文)の入力欄に、印で始まる行を含む文を打った画面。どちらも入力欄の頭は印が 0 桁目にあり、文の中の行は 2 字下げで描かれた。claude は改行が 3 つ以上の文を `[Pasted text #N +M lines]` に畳むので、#322 が claude で起きるのは改行が 2 つ以下の文だけ。Enter のあとの画面も 1 回ずつ取った(claude は座席のアカウントの 5 時間の上限で返事の前に止まった)。
- `internal/org/testdata/resume/` の 11 の fixture は、その画面のバイト単位の写し。記録に名前、元の画面、sha256 の表がある。

## 3. テスト

- `internal/org/send_screen_marker_test.go`(新規): 11 の fixture ごとの `sendInputHolds` と `SendScreenSubmitted` の期待、番号付きの頭の行、拒否の縁(打った文の一部でない印の行、0 桁目の印の行がない、下の罫線より下の印の行、あいだに 0 桁目の行、footer より下、知らない driver、印のあとが空)。
- `internal/org/send_fallback_marker_test.go`(新規): 画面の道の `Send` を通した結合テスト(AC4b)。codex の座席に `›` の行を含む文を送ると、Enter が 1 回押され、`SubmitConfirmed` が true で、`sent` の Details に `via=screen` があり `submit_unconfirmed` がない。
- 既存のテストには、fixture をすべて読む表(`send_screen_test.go`、`resume_screens_test.go`、`usage_limit_screen_test.go`)に新しい fixture の行を足しただけ。tech-debt の 484 行のとおり、新しいテストは大きなファイルに足さずに新しいファイルに置いた。
- `internal/org/prompts_send_unsubmitted_test.go`(新規): leader と director の prompt に足した手順の文を固定する。

## 4. 送れずに止まったときの手順(prompt、skill、規則の文書)

- `internal/org/prompts/leader.md`(typed protocol の節): send が終了コード 1 で、エラーの文が `(text typed but not submitted)` で終わったら、同じ send を打ち直さない。座席の画面を `ralph org read` で読み、`ralph org escalate` で BLOCKED を上げる。座席の pane にキーは送らない。CLI の note の「消すか送る」は端末の前にいる人への助言で、leader はこの段落に従う。
- `internal/org/prompts/director.md` と `/director` skill の「答える順」の 2: 打ち直さず、leader の画面を読む。DECISION の全文が入力欄に残り、ほかの画面がないときだけ、今ある許可で Enter を 1 回押す。それ以外は Enter を押さず、resolve もせず、人に伝える。
- `/org` skill の `send` の行(4 つの写し)と `.claude/rules/ralph/agent-messaging.md`(2 つの写し)にも同じ手順を書いた。

## 5. 文書と記録

- `docs/specs/2026-08-01-org-runtime.md` に 2026-11-13 改訂(fix/send-input-box-marker)の節を足した。
- `docs/tech-debt/README.md`: この PR の Known gaps の行(a)〜(g)を足し、Trigger に当たった行に持ち越しの注を足した。
