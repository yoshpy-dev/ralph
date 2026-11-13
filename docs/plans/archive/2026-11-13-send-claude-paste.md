# send-claude-paste

- Status: Approved
- Approved: 2026-11-13 sha256:2ad0b91591a9
- Owner: Claude Code
- Date: 2026-11-13
- Related request: issue #319(`ralph org send` で claude の座席に約 1 KB を超える文を打つと、頭が欠けて届く)
- Related issue: 319
- Type: fix
- Branch: fix/send-claude-paste

## Objective

`ralph org send` が claude の座席に打つ文を、長さによらず頭から終わりまで
そのまま届ける。typed protocol の上限(本文 2000 字)までの文を claude の
座席に送ったとき、座席が受け取る文は送った文と同じになる。

## Scope

- claude の座席への `ralph org send` の打ち方を変える。文を bracketed paste の
  印(`ESC [200~` と `ESC [201~`)で挟み、今と同じ 1 回の
  `herdr pane send-text` で打つ。herdr の道と画面の道の両方に当てる。
- 文の中に印そのものがあると貼り付けが途中で切れるので、印を含む文は driver に
  よらず断る。`--raw` も dry-run も同じにする。何も打たず、何も記録しない。
- `sent` の Details、dry-run の Details、画面の道の Enter の前の確かめには、
  印のない元の文を使う。
- 文書: `Send` の doc comment、`.claude/rules/ralph/agent-messaging.md` と
  その template の写し、`AGENTS.md` の `internal/org/` の行、
  `docs/tech-debt/README.md`(#308 の行の (c) と、この変更の Known gaps)。
- 実機の確かめ: 分離した herdr で、この branch で作った `ralph` の
  `ralph org send` を claude の座席に 1 回打つ。
- 直し方を決めた確かめの記録 `docs/evidence/org-send-paste-probe-2026-11-13.md`
  を入れる。

## Non-goals

- codex の座席の打ち方。codex では頭は欠けなかったが、長い文は 0.75 秒後の
  Enter で送られなかった。直すには `[Pasted Content N chars]` の画面の規則が
  要り、それは #322 が直している画面の規則の上に作る。issue #326 で扱う。
- 画面の規則(`internal/org/send_screen.go` の `sendInputHolds` など)と
  `resume_screens.go`。#322 が直している場所なので触らない。
- `ralph org resume` が打つ決まった文(`ResumeText`)と、見張りが director を
  起こす 1 行。どちらも短く、頭が欠ける大きさにならない。
- `--raw` の文の大きさの上限。今も上限はなく、この変更でも足さない。
- Enter の前の待ち(`--enter-delay-ms`、既定 750 ms)の変更。

## Assumptions

- 確かめた版は Claude Code 2.1.316、herdr 0.7.5。claude は bracketed paste の
  印を受け、終わりの印までを 1 つの貼り付けとして集める
  (`docs/evidence/org-send-paste-probe-2026-11-13.md` の B1、B2)。
- 座席の driver は manifest の `SeatStatus.Driver` に `claude` か `codex` で
  入っている。空やほかの値の座席には今と同じ素の文を打つ。
- 文全体が 1 つの貼り付けになっても、claude はそれを座席への依頼として扱う。
  確かめでは、B1 と B2 の座席は文の最後の行の指示どおりの語を返した。
- `herdr pane send-text` は、client が途中で止まると文を 1 バイトも打たない。
  300000 バイトの文で client を 0.01〜0.55 秒のあいだに `kill -9` した 12 回は、
  どれも 0 バイトだった。`--timeout-ms` の期限で ralph が client を止めるのも
  同じ SIGKILL である。server が pane に書いている最中に止まる場合は確かめて
  いない(確かめの記録の「打ち込みが途中で切れたときの確かめ」)。
- claude は終わりの印が来ない貼り付けを約 2 秒で閉じ、入力欄に
  `[Pasted text #N +M lines]` を残す。そのあとの入力は貼り付けに取り込まず、
  置き換えの後ろに並べる。Ctrl+C を 1 回押すと入力欄は空になる(同じ節)。

## Affected areas

- `internal/org/send_paste.go`(新規): 2 つの印、claude の座席に打つ文を作る
  関数、印を含む文の断り。
- `internal/org/verbs.go`: `Send` が印を含む文を断り、claude の座席には印で
  挟んだ文を `typeAndSubmit` に渡す。doc comment。
- `internal/org/send_paste_test.go`(新規)、`internal/org/send_fallback_test.go`、
  `internal/org/pane_lock_test.go`: 打った文を見ているテストの期待を、claude の
  座席では印で挟んだ文にする。
- `.claude/rules/ralph/agent-messaging.md`、
  `templates/base/.claude/rules/ralph/agent-messaging.md`。
- `AGENTS.md`(`internal/org/` の行)、`docs/tech-debt/README.md`。
- `internal/cli/org.go`(`ralph org send` の help の 1 文): `--raw` でも印を含む文は
  断るので、「validation を丸ごと飛ばす」を直す(self-review の L2 で足した。
  計画からのずれ)。
- `docs/evidence/org-send-paste-probe-2026-11-13.md`、
  `docs/evidence/org-send-paste-live-2026-11-13.md`(新規)。

## Visual review

- Page: `.harness/state/plan-visual/send-claude-paste.html`(かんたんな図 2 つ
  `overview`、`easy-scope`、承認の前に、折りたたみの中に `detail-overview` と
  `detail-send`)
- Lint: `./scripts/plan-visual.sh lint` は exit 0
- Self-check: ページ全体と 4 つの図を `shot` で撮り、すべての PNG を読んだ。
  文字の重なり、切れたラベル、ラベルを通る線はなかった。計画の助言のあとに
  「承認の前に」の箱の文だけを直し、lint(exit 0)とページ全体の `shot` を
  やり直して読んだ。図は変えていない

## Design decisions

### D1. claude の座席にだけ、文を bracketed paste の印で挟んで 1 回で打つ(自律で選んだ)

consult の上限が 11 月 14 日 23 時まで戻らないので、`/consult` を通さずに
自分で選んだ。レーン A(ralph-4f)の指示「bracketed paste(claude の座席だけ)
で進めてください。」とも合う。

確かめ(`docs/evidence/org-send-paste-probe-2026-11-13.md`)で比べた案は 3 つ。

| 案 | 確かめの結果 | 採否 |
|---|---|---|
| 今のまま 1 回の `send-text` | 1117 バイトの文の先頭 1022 バイトが消えた(A1) | 不採用 |
| 印で挟んで 1 回の `send-text` | 1117 バイトも 5340 バイトも 1 字も欠けずに届いた。124 バイトの文は素の文のまま届いた。置き換えの `+N lines` は文の改行の数と同じで、画面の道の今の確かめが通る | 採用 |
| 行の切れ目で 512 バイト以下に分け、150 ms あけて打つ | 届いたが、塊のあいだの時間に頼る。2000 字の文では `send-text` が 13 回になり、pane のロックを持つ時間が 2 秒ほど延びる | 不採用 |

ほかに、claude の座席には 800 バイトを超える文を断る案もある。typed protocol は
本文 2000 字まで認めているので、ふつうの TASK を断ることになり採らない。

### D2. 印で挟むのは Send の中で、typeAndSubmit の手前

`typeAndSubmit` は `ralph org resume` も使う。打つ文を変えるのは `Send` だけなので、
`Send` が `typeAndSubmit` に渡す文だけを変える。Details と画面の道の確かめは、
今と同じく `p.Text`(印のない文)を使う。

### D3. 印を含む文は driver によらず断る

文の中に `ESC [201~` があると、claude ではそこで貼り付けが終わり、残りが
1 字ずつの打鍵として入る。座席を見る前に断れば dry-run も同じ答えを返せるので、
protocol の確かめのすぐあと、見張りの確かめより前に、driver を見ずに断る。
codex の座席への文も断ることになるが、端末の制御の並びを座席に送る使い方は
ないとみている(未確認)。

Critical forks: D1 のみ(上のとおり自律で選んだ)。

## Acceptance criteria

- [x] AC1: claude の座席への `ralph org send` は、文を `ESC [200~` と
  `ESC [201~` で挟んだものを 1 回の `PaneSendText` で打つ。herdr の道でも
  画面の道でも同じ。
- [x] AC2: driver が codex、空、ほかの値の座席には、今と同じ素の文を打つ。
- [x] AC3: `sent` の Details(画面の道の `via=screen ...` を含む)、dry-run の
  Details、画面の道の Enter の前の確かめは、印のない元の文を使う。
- [x] AC4: 文に `ESC [200~` か `ESC [201~` を含む send は、driver によらず、
  `--raw` でも dry-run でも、何も打たず何も記録せずに断る。エラーは、文に
  bracketed paste の印があることを示す。
- [x] AC5: 実機で、分離した herdr(`docs/recipes/isolated-herdr-server.md`)の
  claude の座席に、この branch で作った `ralph` の `ralph org send` で、本文が
  日本語 2000 字の typed protocol の DECISION を 1 回送る。send は exit 0 で、
  `sent` の Details に `submit_unconfirmed=true` がない。座席のセッションの
  記録の user の文は 1 つの貼り付けで、その中身は送った文とバイト単位で同じ。
  座席は文の最後の行が求める語を返す。claude の手番は 1 回だけ使う。
- [x] AC6: `agent-messaging.md` の 2 つの写しは同じ中身で、claude の座席への
  打ち方と AC4 の断りを書いている。打ち込みが途中で失敗した send のあとに
  claude の座席の入力欄に残りうるもの(`[Pasted text #N +M lines]` の置き換え)
  と、その消し方(その pane で Ctrl+C を 1 回)も書いている。`AGENTS.md` の行と
  `docs/tech-debt/README.md` が今の動きと合っている。
- [x] AC7: claude の座席への send で `PaneSendText` が失敗すると、send は Enter を
  押さずに `SendProgressTextUnacknowledged` を返す(今と同じ)。印で挟んだ文でも
  この向きが変わらないことをテストで確かめる。

## Implementation outline

1. 打ち方の変更とテスト: `send_paste.go` を足し、`Send` で印を含む文を断り、
   claude の座席には印で挟んだ文を打つ。doc comment を直す。打った文を見ている
   既存のテストの期待を直し、AC1〜AC4 と AC7 のテストを足す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ && go test ./internal/cli/ -run 'OrgSend'`
2. 文書: `agent-messaging.md` の 2 つの写し、`AGENTS.md` の行、
   `docs/tech-debt/README.md` — verify: `cmp .claude/rules/ralph/agent-messaging.md templates/base/.claude/rules/ralph/agent-messaging.md && ./scripts/check-sync.sh`
3. 実機の確かめ(AC5)と記録 `docs/evidence/org-send-paste-live-2026-11-13.md`。
   herdr を分ける作業があるので、orchestrator が自分で行う — verify: `test -s docs/evidence/org-send-paste-live-2026-11-13.md && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、
  golangci-lint、staticcheck、check-sync、check-skill-sync)。
- Spec compliance criteria to confirm: AC1〜AC7。#319 の受け入れ条件「頭が
  欠けずに届く(実機で確かめる)」は AC5 で見る。「届かない大きさがあるなら
  断る」は、typed protocol の上限までで届かない大きさが見つからなかったことを
  AC5 と確かめの記録で示す。
- Documentation drift to check: `Send` の doc comment と
  `agent-messaging.md` の 2 つの写し、`AGENTS.md` の行、
  `docs/tech-debt/README.md` の #308 の行の (c)。
- Evidence to capture: verify の報告、AC5 の記録。

## Test plan

- Unit tests(`send_paste_test.go`): 打つ文を作る関数が claude だけ印で挟み、
  codex、空、ほかの値はそのまま返す。印を含む文の断りが、どちらの印でも、
  `--raw` でも、dry-run でも効き、manifest に何も書かず、herdr を呼ばない。
- Integration tests(fake herdr): herdr の道で claude の座席に打った文が印で
  挟まれ、`sent` の Details は元の文のまま。codex の座席では素の文。画面の道でも
  同じで、Enter の前の確かめは元の文で通り Enter が押される。
- Regression tests: 打った文を見ている既存のテスト
  (`send_fallback_test.go` の画面の道の 2 つ、`pane_lock_test.go` の 2 か所)
  の期待を、claude の座席では印で挟んだ文にする。codex の座席の期待は変えない。
  `internal/cli` の `ralph org send` のテストは、打った文を pane までで切って
  見ているので変わらないことを確かめる。
- Edge cases: 印の片方だけを含む文、文の途中にある印、改行のない短い文。
  claude の座席で `PaneSendText` が失敗したとき、Enter を押さずに
  `SendProgressTextUnacknowledged` を返す(AC7)。
- Evidence to capture: test の報告。

## Risks and mitigations

- Claude Code が bracketed paste を受けなくなる版が出ると、印が打鍵として
  入る。確かめた版は 2.1.316 だけ。AC5 で今の版を確かめ、
  `docs/tech-debt/README.md` に版の前提を残す。
- 文全体が 1 つの貼り付けとして届くので、座席がそれを依頼として扱わない
  恐れがある。確かめでは座席は依頼どおりに答えた。今も 800 バイトを超える
  分は貼り付けとして届いている。
- `--raw` で 1 行が 800 バイトを超える文は、置き換えが `+N lines` のない形に
  なるかもしれない。そのとき画面の道の Enter の前の確かめは通らず、send は
  Enter を押さずに打っただけで止まって知らせる(今と同じ向き)。Known gaps に
  残す。
- 開始の印が届いて終わりの印が届く前に打ち込みが切れると、claude は貼り付けを
  集め続け、次の入力を前の文に取り込むかもしれない(計画の助言の所見)。
  確かめでは、herdr の client を止めた 12 回は文を 1 バイトも打たず、claude は
  終わりの印が来ない貼り付けを約 2 秒で閉じた。`PaneSendText` が失敗した send は
  Enter を押さず(AC7)、CLI は pane を `ralph org read` で見てから送り直すよう
  すでに知らせている。入力欄に残った置き換えは Ctrl+C で消せることを
  `agent-messaging.md` に書く(AC6)。server が pane に書いている最中に止まって
  文の一部だけが届いたとき、次の send の文がその後ろに並んで 1 つの文として
  送られる型は、素の文で打つ今も同じなので、Known gaps に残す。
- #322 も `Send` の近くを変えうる。計画の承認時と merge の直前に、A と
  ralph-47 に知らせる。

## Rollout or rollback notes

- 変更は `ralph org send` の中だけで、設定も記録の形も変えない。戻すときは
  この PR を revert する。
- release ノートに載せる項目: claude の座席への `ralph org send` は文を
  bracketed paste で打つ。文に印の並びがあると断る。

## Open questions

- 10 KB を超える `--raw` の文、入力欄に前の置き換えが残ったまま同じ文を
  貼ったときの形は確かめていない。Known gaps に残す。
- claude が終わりの印の来ない貼り付けを集めているあいだに Enter を押したときの
  動きと、herdr の server が pane に書いている最中に止まったときの届き方は
  確かめていない。Known gaps に残す。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex — [MEDIUM] 貼り付けの途中で
  打ち込みが切れたときの復旧の条件がない(開始の印のあと、本文の途中、終わりの
  印の途中で切る試験を足し、途中まで届かないことを確かめるか、復旧の手順を
  受け入れ条件に入れる)→ 計画を更新(自律で選んだ)。claude の手番を使わない
  確かめを分離した herdr で行い、記録の「打ち込みが途中で切れたときの確かめ」に
  入れた。herdr の client を途中で止めた 12 回はどれも 0 バイト、claude は
  終わりの印の来ない貼り付けを約 2 秒で閉じ、Ctrl+C で入力欄が空になった。
  Assumptions と Risks に足し、AC6 に消し方の文書、AC7 に打ち込みが失敗したとき
  Enter を押さないテストを足した。server が書いている最中に止まる場合は
  Known gaps に残す
- [x] Plan approved(レーンの指示「推奨案を自動承認」による。2026-11-13)
- [x] Branch created
- [x] Implementation started
  - S1(implementer、opus): 331eebd3。slice の検証コマンドは rc=0
    (`internal/org` 1269.693s、`internal/cli` 80.313s)。自己点検の 5 項目は
    すべて clean。本体をわざと壊した 6 通りで、どれも新しいテストが落ちた
  - S2(implementer、sonnet): 0d8f04e1。`cmp` と `check-sync.sh` は PASS。
    自己点検の 5 項目はすべて clean
  - S3(inline、orchestrator): 1e3dad38。`test -s` と `check-sync.sh` は PASS。
    実機で送った文と届いた文はバイト単位で同じで、claude の手番は 1 回。
    自己点検は、不要な変更・打ち間違い・デバッグの残り・secret・slice の外の
    ファイルとも clean(U+FFFD は 0)
  - S4(implementer、sonnet、self-review の L1〜L4): eea6424d。コメント、help の
    1 文、文書、tech-debt の (f) だけ。slice の検証コマンドは rc=0。自己点検の
    5 項目はすべて clean
  - S4b(inline、1 ファイルのコメント): `send_paste_test.go` のテスト用の文の
    説明にも L1 と同じ書き方が残っていたので直した。`gofmt -l` は空で、`go vet` は
    通った。自己点検は 5 項目とも clean
- [x] Review artifact created — self-review は LOW 6 件(b065f600)。L1〜L4 は
  verify の前に直す slice にした。L5・L6 は動きに関わらない書き方の好みなので残す。
  L2 で Affected areas に `internal/cli/org.go` を足したので、計画を承認し直した
  (図解ページの図には help の文が出てこないので、図は変えていない。レーンの指示
  「推奨案を自動承認」による。2026-11-13、sha256:2ad0b91591a9)
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created — https://github.com/yoshpy-dev/ralph/pull/328(cross-review は
  codex で指摘 0 件、cycle 1/2)
