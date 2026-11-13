# Walkthrough: claude の座席への send を bracketed paste で打つ(send-claude-paste)

- 計画: `docs/plans/archive/2026-11-13-send-claude-paste.md`(PR の作成のあとに archive へ移す)
- issue: #319
- 差分: `git diff e902705c...HEAD`(24 ファイル、約 +1,570 / −29)。Go の本体は約 +80、Go のテストは約 +340 で、残りは文書・記録・計画・報告。e902705c(#312a、PR #327)は 3710ab21 で取り込んだもので、この差分には入らない

## 1. 打つ文を選ぶところ(読む価値が一番高い)

`internal/org/send_paste.go`(新規)。

- `bracketedPasteStart` は `"\x1b[200~"`、`bracketedPasteEnd` は `"\x1b[201~"`。
- `sendTextForDriver(driver, text)`: driver がちょうど `"claude"` のときだけ、文を 2 つの印で挟んで返す。`"codex"`、空、ほかの値(大文字の `"Claude"` も)には文をそのまま返す。
- `checkNoPasteMarkers(text)`: 文のどこかに印のどちらかがあれば `errSendPasteMarker` を返す。文の中の終わりの印は、貼り付けをそこで閉じ、残りをキーとして打たせてしまうため。

## 2. Send の中の 2 か所

`internal/org/verbs.go` の `Send`。

- 断り: protocol の確かめ(`--raw` なら飛ばす)の直後、`isTask` の見張りと承認の確かめ、dry-run の記録、`findSeat` より前に `checkNoPasteMarkers(p.Text)` を呼ぶ。`--raw` でも `--dry-run` でも、座席の driver によらず、何も打たず、何も記録せず、herdr も呼ばない。
- 打つ文: `typeAndSubmit` に `sendTextForDriver(seat.Driver, p.Text)` を渡す。herdr の道でも画面の道でも、`PaneSendText` は今と同じ 1 回。
- 変えないもの: dry-run の Details、`sent` の Details、画面の道の確かめ(`sendScreenSubmitChecks`)は、印のない `p.Text` を使う。`ResumeText` を打つ `resume.go` は `typeAndSubmit` を直接呼ぶので、印で挟まれない。
- `SendParams.Raw` と `Send` の doc comment に、印の断りは `--raw` でも残ることと、なぜ claude だけかを書いた。

## 3. テスト

- `internal/org/send_paste_test.go`(新規、トップレベル 7 本): 打つ文の選び方(AC1・AC2)、Details と画面の確かめは元の文(AC3)、印を含む文の断り(AC4。印の片方だけ、文の途中、`--raw`、`--dry-run`、不正な文に印があるときは protocol のエラーが先)、`PaneSendText` が失敗したら Enter を押さない(AC7、herdr の道と画面の道)。
- `send_fallback_test.go` と `pane_lock_test.go`: 打った文を見ていた 4 か所の期待を、claude の座席だけ印で挟んだ文にした。codex の座席と `ResumeText` の期待は変えていない。
- 変異 7 個(画面の確かめや Details に印付きの文を渡す、`--raw` や dry-run で断りを飛ばす、開始の印を見ない、断りを protocol の確かめより前に動かす)は、どれもテストが落とした(test の報告)。

## 4. help と文書

- `internal/cli/org.go`: `ralph org send` の help の「`--raw` で validation を丸ごと飛ばす」を、protocol の確かめだけを飛ばし、印を含む文は `--raw` でも断る、に直した。
- `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し: 節「How send types a message」を足した。claude の座席への打ち方、印の断り、打ち込みが途中で失敗した send のあとに入力欄に残りうるもの(`[Pasted text #N +M lines]`)と、その消し方(人が pane で Ctrl+C を 1 回)。`--raw` の段落も直した。
- `/org` skill の `send` の行(4 つの写し)、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-13 改訂の項、`AGENTS.md` の `internal/org/` の行。
- `docs/tech-debt/README.md`: #308 の行の (c) に取り消し線を引いて解消とした。表の末尾に #319 の Known gaps の行(a)〜(g)を足し、Trigger に当たった 4 つの行に持ち越しの注を足した。

## 5. 確かめの記録

- `docs/evidence/org-send-paste-probe-2026-11-13.md`: 直し方を決める前の確かめ。素の 1117 バイトの文は先頭 1022 バイトが消え、印で挟むと 1117 バイトも 5340 バイトも欠けずに届いた。打ち込みを途中で止めた 12 回は、どれも 1 バイトも届かなかった。
- `docs/evidence/org-send-paste-live-2026-11-13.md`: AC5。この branch の `ralph` で、分離した herdr の claude の座席に本文 2000 字の DECISION を 1 回送った。届いた文は送った文とバイト単位で同じで、座席は求めた語を返した。claude の手番は 1 回。
