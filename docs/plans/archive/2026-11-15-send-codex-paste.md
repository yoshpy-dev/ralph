# send-codex-paste

- Status: Approved
- Approved: 2026-11-15 sha256:f17371b28b9f
- Owner: Claude Code
- Date: 2026-11-15
- Related request: issue #326(org send: codex の座席に長い文を打つと、750 ms 後の Enter で送られない)
- Related issue: 326
- Type: fix
- Branch: fix/send-codex-paste

## Objective

`ralph org send` が codex の座席に typed protocol の文(本文 2000 字まで)を送ると、
1 回の Enter で送られ、全文が届くようにする。画面の道でも同じ文に Enter を押せる
ようにする。

## Scope

- codex の座席にも、claude と同じく文を bracketed paste の印で挟んで 1 回で打つ
  (`sendTextForDriver`、`internal/org/send_paste.go`)。
- 画面の道の Enter の前の確かめ(`sendInputHolds`、`internal/org/send_screen.go`)
  に、codex の置き換え `[Pasted Content <N> chars]` の規則を足す(D2)。
- 実機の画面の fixture とテスト。打った文を見ている既存のテストの期待を、codex の
  座席では印で挟んだ文にする。
- 文書(`agent-messaging.md` の 2 つの写し、`AGENTS.md` の `internal/org/` の行、
  `/org` skill の `send` の行の 4 つの写し)。
- 実機の確かめ: この branch で作った `ralph` の `ralph org send` で、分離した herdr の
  codex の座席に本文 2000 字の文を 1 回送る。

## Non-goals

- Enter の前の待ち(`--enter-delay-ms`、既定 750 ms)を driver や文の大きさで
  変えること。
- 本文 2000 字の上限を超える `--raw` の文の扱い。
- resume の続行の文(`ResumeText`)の打ち方。resume は `typeAndSubmit` を直接呼び、
  今までどおり素の文で打つ。
- claude と codex のほかの driver。

## Assumptions

- 確かめの記録 `docs/evidence/org-send-codex-paste-probe-2026-11-15.md`
  (codex-cli 0.166.0、herdr 0.7.5): bracketed paste で打った 49〜4925 バイトの
  5 つの文は、どれも 750 ms 後の Enter で送られ、バイト単位で同じに届いた。
  651 バイトまでは畳まれずに入力欄に出て、1698 バイト以上は
  `› [Pasted Content <N> chars]` の 1 行になり、N は文の字数だった。
- #319 の確かめ(`docs/evidence/org-send-paste-probe-2026-11-13.md` の X1)では、
  5340 バイトの素の文は 750 ms 後の Enter で送られなかった。
- 印を含む文の断り(`checkNoPasteMarkers`)は #319 からすべての driver にかかる
  ので、codex に貼り付けで打っても、文の中の印で貼り付けが途中で閉じることはない。
- herdr の道は Enter のあと `confirmSubmitted` で working か blocked を待つ。確かめ
  では、どの文でも Enter から 0.3 秒後に送られていた(`• Working` か履歴)。

## Affected areas

- `internal/org/send_paste.go`: `sendTextForDriver` が claude と codex の両方で印で
  挟む。doc comment。
- `internal/org/send_screen.go`: `sendInputHolds` に codex の置き換えの規則と、その
  正規表現。doc comment。
- `internal/org/verbs.go`: `Send` の doc comment の、claude だけと書いた箇所。
- `internal/org/send_paste_test.go`、`internal/org/send_fallback_test.go`、
  `internal/org/pane_lock_test.go`: codex の座席の期待を印で挟んだ文にする。
- 新しいテストのファイル(例: `internal/org/send_codex_paste_test.go`)と、
  `internal/org/testdata/resume/` の新しい fixture。名前は #322 の表の名前の型
  (`-input-typed-(marker|paste)-`)に当たらないものにする(例:
  `codex-input-pasted-placeholder.txt`)。fixture をすべて読む既存の表
  (`send_screen_test.go`、`resume_screens_test.go`、`usage_limit_screen_test.go`)
  には行だけを足す。
- `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md`、
  `AGENTS.md`、`/org` skill の 4 つの写し。
- `docs/evidence/org-send-codex-paste-probe-2026-11-15.md`(計画と一緒にコミット)と、
  実機の確かめの記録 `docs/evidence/org-send-codex-paste-live-2026-11-15.md`(新規)。
- `docs/tech-debt/README.md`(この PR の Known gaps、#319 の行の (e))。

## Visual review

- Page: `.harness/state/plan-visual/send-codex-paste.html`(かんたん版の図 2 つ:
  `overview`、`easy-screen`。折りたたみの中に `detail-overview`)
- Lint: `./scripts/plan-visual.sh lint` は exit 0。
- Self-check: 全体と 3 つの図を `shot` で撮って読んだ。重なり、はみ出し、線の
  通り抜けはない。

## Design decisions

Critical forks: 1 つ(D1)。レーンの指示「推奨案を自動承認」により自分で選んだ
(自律で選んだ)。

- D1(直し方): codex の座席にも bracketed paste で打つ。
  - 選んだ理由: 確かめで、2000 字の本文の文まで 750 ms 後の Enter で送られ、
    全文が届いた。claude と同じ道(#319)なので、印の断りや Details の扱いが
    そのまま使える。
  - ほかの案: codex の Enter の前の待ちを文の大きさで延ばす。どれだけ延ばせば
    よいかを測っておらず、codex の描き足しの速さは機械の負荷でも変わる。
  - 戻しにくさ: 案を途中で変えると、画面の規則とテストの両方の作り直しになる。
- D2(画面の規則): codex の入力欄の文(空白を除く)が `[PastedContent<N>chars]` の
  1 つだけで、N が打った文(印を除く)の字数(`utf8.RuneCountInString`)と等しい
  ときに通す。置き換えのほかに文があるとき、置き換えが 2 つ以上のとき、N が
  違うときは通さず、今までどおり Enter を押さずに `text typed but not submitted`
  で止まる。畳まれない大きさの文には、今の 2 つの規則(文の頭、`↑` 付きの入力欄に
  文の終わり)がそのまま当たる。
- D3(「押せない形の文なら断る」): 確かめでは、typed protocol の上限(本文 2000 字)
  までのどの大きさでも Enter で送られたので、大きさで断る規則は足さない。上限を
  超える `--raw` の文は Known gaps に残す。

## Acceptance criteria

- [x] AC1: driver が codex の座席への `ralph org send` は、文を `ESC [200~` と
  `ESC [201~` で挟んで 1 回の `PaneSendText` で打つ(herdr の道と画面の道)。
  claude の座席は今のまま。driver が空やほかの値の座席には素の文を打つ。テストで
  固定する。
- [x] AC2: `sent` と dry-run の Details、画面の道の確かめは、codex の座席でも印の
  ない元の文を使う。テストで固定する。
- [x] AC3: codex の実機の置き換えの画面(`[Pasted Content 1698 chars]`、
  `[Pasted Content 2016 chars]`)で、`sendInputHolds` が true を返す。N が違う画面、
  置き換えのほかに文がある画面、置き換えが 2 つの画面では false。実機の畳まれない
  画面(p1、p2)では true のまま。Enter から 0.3 秒後の画面で
  `SendScreenSubmitted` が true、置き換えが残った画面で false。テストで固定する。
- [x] AC4: 画面の道の `Send` を通した結合テスト(codex の座席、日本語 2000 字の
  本文の文、fake の herdr が実機の置き換えの画面と Enter のあとの画面を返す)で、
  打った文が印で挟まれ、Enter が 1 回押され、`SubmitConfirmed` が true で、`sent` の
  Details に `submit_unconfirmed` がない。
- [x] AC5: 実機で、分離した herdr(`docs/recipes/isolated-herdr-server.md`)の codex の
  座席に、この branch で作った `ralph` の `ralph org send` で、本文が日本語 2000 字の
  typed protocol の DECISION を 1 回送る。send は exit 0 で、`sent` の Details に
  `submit_unconfirmed=true` がない。codex のセッションの記録の user の文は、送った文と
  バイト単位で同じ。座席は文の最後の行が求める語を返す。codex の手番は 1 回だけ
  使う。claude の座席は使わない。
- [x] AC6: `agent-messaging.md` の 2 つの写し、`AGENTS.md` の行、`/org` skill の
  `send` の行が、codex にも貼り付けで打つことと、codex の置き換えの形を書いている。

## Implementation outline

1. S1(implementer、opus): `sendTextForDriver` の driver の集合、codex の置き換えの
   規則、doc comment、fixture、新しいテスト、既存のテストの codex の期待(AC1〜AC4)。
   fixture は scratchpad の `probe326/out/` の画面の写しから作る — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'Paste|Screen|SendInputHolds|Send|Resume|UsageLimit|Marker|PaneLock' -count=1`
2. S2(implementer、sonnet): 文書(AC6) — verify: `cmp .claude/rules/ralph/agent-messaging.md templates/base/.claude/rules/ralph/agent-messaging.md && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`
3. S3(inline、orchestrator): 実機の確かめ(AC5)と記録 `docs/evidence/org-send-codex-paste-live-2026-11-15.md`。分離した herdr を立てる作業があるので、orchestrator が自分で行う — verify: `test -s docs/evidence/org-send-codex-paste-live-2026-11-15.md && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`。
- Spec compliance criteria to confirm: AC1〜AC6。#326 の受け入れ条件「1 回の Enter で
  送られ、全文が届く(実機)」は AC5、「画面の道でも Enter を押せる。押せない形なら
  断る」は AC3・AC4 と D3。
- Documentation drift to check: `sendTextForDriver`、`sendInputHolds`、`Send` の doc
  comment、`agent-messaging.md`、`AGENTS.md`、`/org` skill、spec、tech-debt の #319 の
  行の (e)。
- Evidence to capture: verify の報告、AC5 の記録。

## Test plan

- Unit tests: `sendTextForDriver` の driver ごとの期待。codex の置き換えの規則の
  通る画面と通らない画面(AC3)。
- Integration tests: AC4 の 1 本。herdr の道の codex の座席で、打った文が印で
  挟まれる 1 本。
- Regression tests: claude の座席の期待、既存の send と resume の画面のテスト、
  `ResumeText` が素の文で打たれること。
- Edge cases: N が 1 違う置き換え、置き換え 2 つ、置き換えのあとに文がある入力欄、
  字数と UTF-16 の単位が違う文(絵文字を含む文)の N。
- Evidence to capture: test の報告、変異(規則の条件を 1 つずつ外す)の結果。

## Risks and mitigations

- codex の別の版で置き換えの形が変わると、画面の道の確かめは通らず、Enter を
  押さずに止まる(知らせる側)。herdr の道は画面を読まないので影響しない。
- 畳まれ始める大きさは確かめていない。畳まれれば D2、畳まれなければ今の規則が
  当たる。どちらにも当たらない形(置き換えと文が混ざる)は通さない。
- 貼り付けが途中で切れたとき、codex の入力欄に何が残るかは確かめていない。
  claude と同じく、send は失敗した send-text のあとに Enter を押さない。Known gaps に
  残す。

## Rollout or rollback notes

- revert で戻せる。設定、台帳の形、CLI の flag は変えない。

## Open questions

- 字数と書記素の数が違う文での N(結合文字)。

## Progress checklist

- [x] Plan reviewed — plan advisory: no findings (codex)。「重大な指摘はありません。
  現行コードと実測記録に照らすと、変更範囲、受け入れ条件、失敗時の扱いは妥当です」
- [x] Plan approved(レーンの指示「推奨案を自動承認」による。2026-11-15)
- [x] Branch created
- [x] Implementation started
  - S1(implementer、opus のあと sonnet で再開): ad29a286。1 回目は #322 の結合テスト
    (`send_fallback_marker_test.go`)の codex の期待(素の文)が範囲の外で落ちたので
    コミットせずに止まった。そのファイルを範囲に足し、残った変更をこの slice の
    ものとして再開した。確かめのコマンドは exit 0(`internal/org` 70.172s)。変異 5 つ
    (codex を外す、N を UTF-16 で比べる、余計な文字つきの置き換えを通す、Enter の
    あとの確かめから置き換えを外す、規則 4 を外す)はどれもテストが落とした。
    自己点検の 5 項目はすべて clean
  - origin/main(3c75e2e0、#311、PR #332)を 1f3146f5 で取り込んだ。衝突はなく、
    絞ったテストと check-sync は通った
  - S2(implementer、sonnet): 4403bef4。agent-messaging.md の 2 つの写し、AGENTS.md、
    `/org` skill の 4 つの写し。確かめのコマンドは exit 0。自己点検はすべて clean
  - S3(inline、orchestrator): 1072522e。実機の確かめ(AC5)。send は exit 0、届いた
    文はバイト単位で同じ、codex は「受信326」を返した。codex の手番 1 回、claude の
    座席は使っていない。自己点検の 5 項目は clean(U+FFFD は 0)
  - self-review の直し(L1、L2、L4、L5、L7): 7ecd4498。コメント、テストの
    名前、実機の記録の 2 行だけで、本体の動きは変えていない
  - sync-docs(1 回目): 79ad7302。verify の V1〜V6 を直し、tech-debt に
    Known gaps の行と、Trigger に当たった 7 行の持ち越しの注を足した。報告は
    `docs/reports/sync-docs-2026-11-15-send-codex-paste.md`
- [x] Review artifact created — self-review は LOW 7 件(69179e35)。L1、L2、
  L4、L5、L7 は 7ecd4498 で直し、L3 と L6 は sync-docs で tech-debt の行にした
- [x] Verification artifact created — verify は partial-pass(5c274b0a)。AC6 は
  `AGENTS.md` の行に 1 句が足りなかった(V1)。V1〜V6 は sync-docs で直した
- [x] Test artifact created — test は pass(3598f296)。`run-test.sh` は rc 0 で、
  変異 14 個のうち 12 個を殺した。生き残った M11 と M13 は tech-debt の行の (g)
- [x] PR created — https://github.com/yoshpy-dev/ralph/pull/335(cross-review は
  codex で指摘 0 件、cycle 1/2)
