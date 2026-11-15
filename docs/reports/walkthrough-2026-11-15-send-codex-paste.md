# Walkthrough: codex の座席にも貼り付けで打ち、1 回の Enter で届ける(send-codex-paste)

- 計画: `docs/plans/archive/2026-11-15-send-codex-paste.md`(PR の作成のあとに archive へ移す)
- issue: #326
- 差分: `git diff df186374...HEAD`(34 ファイル、約 +1,580 / −110)。Go の本体は +111 / −47(多くはコメント)、Go のテストは +462 / −24、fixture は 5 ファイルで +115、残りは文書・記録・計画・報告。取り込んだ #311・#312・#318 はこの差分に入らない

## 1. 打つ文(読む価値が一番高い)

`internal/org/send_paste.go` の `sendTextForDriver`(36 行)。driver がちょうど `"claude"` か `"codex"` のとき、文を `ESC [200~` と `ESC [201~` で挟む。ほかの driver(空、`"Claude"`、`"Codex"`、`"gemini"`)には素の文を打つ。印を含む文の断り(`checkNoPasteMarkers`)は #319 からすべての driver にかかるので、変えていない。`Send`(`internal/org/verbs.go`)は doc comment だけを直した。`resume.go` は `typeAndSubmit` を直接呼ぶので、`ResumeText` は今までどおり素の文で打つ。

## 2. 画面の道の確かめ

`internal/org/send_screen.go`。

- `sendCodexPastePlaceholder`(194 行): 空白を除いた入力欄の文の `[PastedContent<N>chars]`。
- `sendCodexPasteHolds`(487 行): `sendInputHolds` の codex の 4 つ目の規則。入力欄の文が置き換え 1 つだけで、N が打った文(印を除く、見出しを含む文全体)の字数(`utf8.RuneCountInString`)と等しいときに通す。今の 2 つの規則(文の頭、`↑` 付きの入力欄に文の終わり)のあとに試す。畳まれない大きさの文には、今の規則がそのまま当たる。
- `sendInputHasPastePlaceholder`(558 行): Enter のあとの確かめで、codex の置き換えが入力欄に残っていれば、N によらず送れていないとみる。
- `sendScreenSubmitChecks`(`send_fallback.go`)などの doc comment も codex の置き換えを書くようにした。

## 3. 実機の画面とテスト

- `docs/evidence/org-send-codex-paste-probe-2026-11-15.md`: 分離した herdr の codex の座席に、49〜4925 バイトの 5 つの文を貼り付けで打った記録。どれも 750 ms 後の Enter で送られ、バイト単位で同じに届いた。651 バイトまでは文が入力欄に出て、1698 バイトと 4925 バイトは `[Pasted Content <N> chars]` になり、N は字数だった(絵文字を 1 つ含む文で、UTF-16 の単位なら 2017 のところ 2016)。
- fixture 5 つ(`internal/org/testdata/resume/codex-input-pasted-*.txt`、`codex-submitted-pasted.txt`)は、その画面のバイト単位の写し。
- `internal/org/send_codex_paste_test.go`(新規): driver ごとの打つ文、置き換えの規則の通る画面と通らない画面(N±1、UTF-16 の数、バイト数、置き換え 2 つ、前後の文、claude の形)、Enter のあとの確かめ、画面の道の `Send` を通した結合テスト(Enter が 1 回、`SubmitConfirmed` が true、`submit_unconfirmed` がない)、数が違うときに Enter を押さないこと、send-text が失敗したときに Enter を押さないこと。
- 既存のテストは、codex の座席の期待を印付きの文にしただけ(`send_paste_test.go`、`send_fallback_test.go`、#322 の `send_fallback_marker_test.go`)。fixture をすべて読む 3 つの表には行を足した。

## 4. 実機の確かめ(AC5)

`docs/evidence/org-send-codex-paste-live-2026-11-15.md`。この branch の `ralph` で、分離した herdr の codex の座席に本文 2000 字(3938 バイト)の DECISION を 1 回送った。send は exit 0 で `submit_unconfirmed` はなく、届いた文は送った文とバイト単位で同じで、codex は求めた語を返した。send は herdr の道を通った。

## 5. 文書と記録

- `.claude/rules/ralph/agent-messaging.md`(2 つの写し)、`AGENTS.md` の `internal/org/` の行、`/org` skill の `send` の行(4 つの写し): codex にも貼り付けで打つこと、codex の置き換えの形、N を文全体の字数と比べること。Ctrl+C の文は、確かめた範囲(文が入った入力欄)に合わせた。
- `docs/specs/2026-08-01-org-runtime.md` に 2026-11-15 改訂の節を足した。
- `docs/tech-debt/README.md`: #326 の Known gaps の行(a)〜(i)、#319 の行の (e) の注、#322 の行の (f) の解消、Trigger に当たった行の持ち越しの注。
