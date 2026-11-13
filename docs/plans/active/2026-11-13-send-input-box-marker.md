# send-input-box-marker

- Status: Approved
- Approved: 2026-11-13 sha256:6dc6132083c1
- Owner: Claude Code
- Date: 2026-11-13
- Related request: issue #322(org send: 送る文に入力欄の印(❯/›)で始まる行があると、画面の道で Enter が押されず文が入力欄に残る)
- Related issue: 322
- Type: fix
- Branch: fix/send-input-box-marker

## Objective

`ralph org send` の画面の道で、送る文に入力欄の印(claude は `❯`、codex は `›`)で
始まる行があっても、Enter の前の確かめが入力欄を正しく見つけ、Enter を押して
文を届ける。本物の選択の画面と、文が入力欄に入らなかった画面では、今までどおり
Enter を押さない。send が「打ったが送っていない」で終わったときに、leader と
director が打ち直さず pane を読む手順を、role prompt と skill に書く。

## Scope

- Enter の前の確かめ(`sendTypedInputBox`、`internal/org/send_screen.go`)で、
  最後の印の行が文の中の行(字下げされた印の行)のときに、入力欄の頭の行を
  選び直す(Design decisions の D1・D2)。
- 分離した herdr で取った実機の画面(`docs/evidence/org-send-marker-probe-2026-11-13.md`)
  を testdata の fixture にし、テストを足す。
- leader の role prompt(`internal/org/prompts/leader.md`)、director の role prompt
  (`internal/org/prompts/director.md`)、`/org` skill と `/director` skill
  (それぞれ 4 つの写し)に、send が `text typed but not submitted` で終わった
  ときの手順を足す(D3)。prompt の文はテストで固定する。

## Non-goals

- resume の画面の型(`findResumeInputBox`、`ClassifyResumeScreen`、
  `ResumeInputHolds`)と、Enter のあとの確かめ(`SendScreenSubmitted`)の入力欄の
  探し方は変えない。文が入力欄に残っていれば、どちらも今までどおり「入力欄が
  ない」か「送れていない」とみる(D1)。
- 打つ前の確かめ(`ClassifySendScreen`)で、前の send の残りの文が入力欄に
  あるときの扱い。今は入力欄を見つけられず打たないので、知らせる側に倒れる。
- codex の座席への bracketed paste と、長い文が Enter で送られない件(#326)。
  この計画のあとに別の PR で扱う。
- 画面の道を ralph で通しで実機で走らせること。herdr が手番の終わった座席を
  `working` と言い続ける状態を作れない(#308 の Known gaps (a) と同じ)。

## Assumptions

- 実機の画面(`docs/evidence/org-send-marker-probe-2026-11-13.md`、Claude Code
  2.1.316、codex-cli 0.166.0、herdr 0.7.5)では、どちらの CLI も入力欄の頭の行は
  印が 0 桁目にあり、文の 2 行目からは 2 字下げで描いた。文の中の印で始まる行も
  2 字下げになった。#308 の証拠(`docs/evidence/org-send-screen-probe-2026-11-12.md`
  の 10 節)の複数行の文の画面も同じ形だった。
- claude には #319 から bracketed paste で打つ。改行が 3 つ以上の文は
  `[Pasted text #N +M lines]` に畳まれ、印の行は入力欄に出ない。#322 が claude で
  起きるのは改行が 2 つ以下の文(`TYPE:` の行、空行、1 行の本文)と、`--raw` で
  打つ 3 行以下の文。
- codex には素の文を打つ(#326 で変わる)。文の中の `›` の行は、短い文でも
  長い文(入力欄がスクロールし、1 行目の右端に `↑`)でも入力欄に出る。
- Enter で送ったあとは、どちらの CLI でも送った文が履歴に移り、文の中の印の行は
  履歴でも 2 字下げで描かれた。下から最初の印の行は、空になった入力欄の頭の行
  (0 桁目)だった(確かめの記録の「Enter のあとの画面」)。

## Affected areas

- `internal/org/send_screen.go`: `sendTypedInputBox` に、入力欄の頭の行を選び
  直す道を足す。doc comment。
- `internal/org/send_screen_test.go`: 新しい fixture のテスト。必要なら、
  fixture の一覧を持つ表(`ClassifySendScreen` の期待の表など)に新しい fixture の
  行を足す。
- `internal/org/testdata/resume/`(新規の fixture 11): `codex-input-typed-marker-line.txt`、
  `codex-input-typed-marker-line-last.txt`、`codex-input-typed-marker-line-scrolled.txt`、
  `codex-input-typed-marker-numbered.txt`、`claude-input-typed-marker-line.txt`、
  `claude-input-typed-marker-line-one-nl.txt`、`claude-input-typed-marker-no-nl.txt`、
  `claude-input-typed-marker-line-wrapped.txt`、`claude-input-typed-paste-short.txt`、
  `claude-submitted-marker-line.txt`、`codex-submitted-marker-line.txt`。
- fixture をすべて読む既存のテスト(`internal/org/resume_screens_test.go` など)が
  あれば、その表に新しい fixture の行を足す。
- `internal/org/prompts/leader.md`、`internal/org/prompts/director.md`、新しい
  prompt のテスト(`internal/org/prompts_send_unsubmitted_test.go`)。
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と、それぞれの
  `.agents/skills/` と `templates/base/` の写し。
- `docs/evidence/org-send-marker-probe-2026-11-13.md`(新規、計画と一緒にコミット)。
- `docs/tech-debt/README.md`(この PR の Known gaps の行)。

## Visual review

- Page: `.harness/state/plan-visual/send-input-box-marker.html`
  (かんたん版の図 3 つ: `overview`、`easy-refuse`、`easy-docs`。折りたたみの
  中に `detail-overview` と `detail-pick`)
- Lint: `./scripts/plan-visual.sh lint` は exit 0。
- Self-check: 全体と 5 つの図を `shot` で撮って読んだ。詳しい図 2 の画面の
  2 字下げが詰まって見えたので、空白を NBSP にして撮り直した。そのあとは、どの
  図にも重なり、はみ出し、線の通り抜けはない。

## Design decisions

Critical forks: 1 つ(D1)。consult のアドバイザーが 11 月 14 日 23 時まで使えない
ので、レーンの指示「推奨案を自動承認」により自分で選んだ(自律で選んだ)。

- D1(直す場所): Enter の前の確かめ(`sendTypedInputBox`)だけで入力欄の頭の行を
  選び直す。
  - 選んだ理由: 文が入力欄に残っているかを見るのは、この確かめだけ。
    `findResumeInputBox` は resume の画面の型と、打つ前と Enter のあとの確かめが
    共有している。そこを変えると、resume の判定と、`ClassifySendScreen` の待ちの
    判定まで動く。
  - ほかの案 1: `findResumeInputBox` を、claude は罫線の境、codex は footer から
    上で入力欄を取る形に書き換える(issue の案)。すべての呼び手が変わり、
    resume の fixture の期待もすべて見直しになる。
  - ほかの案 2: 印で始まる行を含む文を send が断る。正しい文を送れなくなる。
  - 戻しにくさ: 案 1 で始めて途中で戻すと、resume のテストの見直しが 1 slice を
    超える。
- D2(選び直す条件): 次のすべてを満たすときだけ、入力欄の頭を行 M にする。
  1. 下から最初の印の行 L(今の `resumeMarkerLineBox`)の、印の前に空白がある
     (0 桁目ではない)。
  2. L の印のあとの文(空白を除く)が、打った文(空白を除く)の一部。
  3. L より上に、1 文字目が印の行 M がある(0 桁目)。M と L のあいだの行は、
     どれも空行か、2 つの空白で始まる。
  4. M から取った入力欄が入力欄の形をしている(`resumeInputBoxShape`)。
     claude では M の真上が罫線で、M の下の最初の罫線が L より下にある。
     codex では footer がある(`sendCodexHasFooter`)。L は footer より上にある。
  5. M の文が番号付きの行なら、今の規則のとおり、その文も打った文の一部。

  選び直した入力欄には、今の 3 つの規則(頭、終わり、claude の placeholder)を
  そのまま当てる。どれかを満たさなければ、今までどおり入力欄はないとみて、
  Enter を押さない。入力欄の下に別の画面(選択の画面)が出ている場合は、L が
  その画面の行になる。条件 2 か 4 で外れるので、Enter は押さない。
- D3(文書の手順): send が `text typed but not submitted` で終わったとき。
  - leader: send を打ち直さない。`ralph org read` で座席の pane を読み、
    `ralph org escalate` で BLOCKED を上げる。読んだ pane のポインタを本文に書く。
    座席の pane にキーは送らない。
  - director: send を打ち直さない。`ralph org read` で leader の pane を読む。
    DECISION の文がそのまま全部、leader の入力欄に残っていて、ほかの画面が
    出ていないときだけ、今ある許可(入力欄に残った DECISION への Enter 1 回)で
    Enter を 1 回押し、もう一度読む。それ以外は Enter を押さず、人に伝える。

## Acceptance criteria

- [x] AC1: codex で、実機の 4 つの画面(`›` で始まる行が文の途中、最後、
  スクロールした入力欄の中、番号付きの行)について、`sendInputHolds` が true を
  返す(Enter を押す)。テストで固定する。
- [x] AC2: claude で、実機の画面のうち改行 2 つと 1 つの文(`❯` で始まる行が
  入力欄に出る)について、`sendInputHolds` が true を返す。改行 0 の文
  (`❯ ❯ explain this`)と、4 行の文の placeholder(`[Pasted text #1 +3 lines]`)
  も true のまま。テストで固定する。
- [x] AC3: 今まで Enter を押さなかった画面では、今までどおり false になる。
  既存の拒否の fixture(選択の画面、入力欄の下の選択の画面、空の入力欄、
  ほかの文が入った入力欄)に加え、次の 3 つの縁もテストで固定する。
  - 字下げされた印の行の文が、打った文の一部でない。
  - 0 桁目の印の行が上にない。
  - claude で、字下げされた印の行が下の罫線より下にある。
- [x] AC4: Enter のあとの確かめ(`SendScreenSubmitted`)は、変えた
  `sendInputHolds` を呼ぶので、次の両方をテストで固定する。
  - 新しい fixture で文が入力欄に残っている画面では false(送れていない)。
  - 実機の Enter から 0.3 秒後の画面(`claude-submitted-marker-line.txt`、
    `codex-submitted-marker-line.txt`、送った文の中の印の行が履歴にある)では
    true(送れた)。
  `findResumeInputBox`、`ClassifyResumeScreen`、`ResumeInputHolds`、
  `ClassifySendScreen` のコードは変えず、それらの既存のテストは期待を変えずに通る。
- [x] AC4b: 画面の道の `Send` を通した結合テスト(codex の座席、`›` の行を含む文、
  fake の herdr が実機の Enter の前と後の画面を返す)で、Enter が 1 回押され、
  `SubmitConfirmed` が true で、`sent` の Details に `via=screen` があり
  `submit_unconfirmed` がない。
- [x] AC5: leader と director の role prompt、`/org` skill と `/director` skill
  (4 つの写しずつ)に、D3 の手順が書かれている。leader と director の prompt の
  文はテストで固定する。
- [x] AC6: 確かめの記録(`docs/evidence/org-send-marker-probe-2026-11-13.md`)が
  あり、fixture はその画面の写しになっている。

## Implementation outline

1. S1(implementer、opus): `sendTypedInputBox` の選び直し(D1・D2)と doc comment、
   実機の画面の 11 の fixture、テスト(AC1〜AC4b)。fixture は scratchpad の
   `probe322/out/` の画面の写しから作る(handoff に対応を書く) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'Screen|SendInputHolds|Resume|Send' -count=1`
2. S2(implementer、sonnet): D3 の手順を leader と director の prompt、`/org` と
   `/director` の skill に足し、prompt のテストを足す(AC5)。A の #318 も同じ
   ファイルに触れるが、触る節は別(#318 は leader.md の 248・932・1207 行と末尾、
   `/director` skill の 89・636・955・1456・2004 行、director.md の 53 行。S2 は
   leader.md の typed protocol の節、`/director` skill の「答える順」の手順 2、
   director.md のミッションの箇条の途中)。PR の前に origin/main を取り込み、
   衝突は両方を残して解く — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'RenderRolePrompt' -count=1 && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、
  golangci-lint、staticcheck、check-sync、check-skill-sync)。
- Spec compliance criteria to confirm: AC1〜AC6(AC4b を含む)。D2 の 5 つの条件がコードの
  分岐とテストの両方にあること。
- Documentation drift to check: `sendTypedInputBox` と `sendInputHolds` の
  doc comment、`/org` skill の `send` の行、AGENTS.md の `internal/org/` の行の
  screen fallback の項、spec の改訂の項。
- Evidence to capture: verify の報告。

## Test plan

- Unit tests: 新しい fixture ごとの `sendInputHolds` の期待(AC1・AC2)と、
  文が残った画面と送れた画面の `SendScreenSubmitted` の期待(AC4)。
- Integration tests: AC4b の 1 本。`via=screen` があるだけでは送れたことに
  ならない(確かめが外れても `submit_unconfirmed=true` 付きで記録される)ので、
  `SubmitConfirmed` と `submit_unconfirmed` がないことも見る。今の画面の道の
  テストの形に合わせる。
- Regression tests: 既存の send と resume の画面のテストを期待を変えずに通す。
  #319 の placeholder の規則(`[Pasted text #1 +3 lines]`)も固定する。
- Edge cases: AC3 の 3 つの縁。印の行が入力欄の最後の行にある画面(last)。
  番号付きの印の行(numbered)。
- Evidence to capture: test の報告、変異(D2 の条件を 1 つずつ外す)の結果。

## Risks and mitigations

- 入力欄の頭が 0 桁目にあることに頼る。ほかの版の CLI が入力欄を字下げすると、
  選び直しは働かず、今までどおり Enter を押さない(知らせる側)。Known gaps に
  残す。
- 選び直しが本物の選択の画面を入力欄とみる恐れ。D2 の条件 2 と 4 で防ぎ、AC3 の
  テストで固定する。
- S2 は A の #318 と同じファイル(leader.md、director.md、`/director` skill の
  4 つの写し)に触れる。触る節は別なので先に書き、PR の前に origin/main を
  取り込む。衝突したら両方の文を残し、`check-skill-sync.sh` と prompt の
  テストで確かめる。
- #326 で codex にも bracketed paste で打つようになると、長い文の codex の画面は
  placeholder になる。この計画の規則は短い文で効き続け、長い文では使われない
  だけになる。

## Rollout or rollback notes

- revert で戻せる。設定、台帳の形、CLI の flag は変えない。
- 戻すと、印で始まる行を含む文は、画面の道でまた `text typed but not submitted`
  で止まる(前と同じ)。

## Open questions

- claude が上限に当たらずに返事をしたあとの画面(確かめでは座席のアカウントが
  5 時間の上限に当たり、返事の前に止まった)。Enter から 0.3 秒後の画面で
  送れたことは見えているので、Enter のあとの確かめには足りるとみる。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex — [MEDIUM] 結合テストは
  `via=screen` だけを見るので、確かめが外れて `submit_unconfirmed=true` でも
  通ってしまう。Enter のあとの画面は確かめておらず、`SendScreenSubmitted` は
  変えた `sendInputHolds` を呼ぶので、ほかの確かめと切り離せていない → 計画を
  更新(自律で選んだ)。分離した herdr で claude と codex に 1 回ずつ Enter で
  送り、Enter のあとの画面を取った(確かめの記録の「Enter のあとの画面」。
  claude は座席のアカウントの 5 時間の上限で返事の前に止まったが、0.3 秒後の画面で
  送れたことは見えた)。AC4 に送れた画面で true を足し、AC4b(`SubmitConfirmed`
  が true で `submit_unconfirmed` がない結合テスト)を足し、fixture を 11 にした
- [x] Plan approved(レーンの指示「推奨案を自動承認」による。2026-11-13)
- [x] Branch created
- [x] Implementation started
  - S1(implementer、opus のあと sonnet で再開): a953ff11。1 回目の implementer は
    `usage_limit_screen_test.go` の全 fixture の表(範囲の外)で確かめが落ちたので
    コミットせずに止まった。表を範囲に足し、残った変更をこの slice のものとして
    再開した。slice の確かめのコマンドは exit 0(`internal/org` 62.994s)。D2 の
    条件を 1 つずつ外す変異はどれもテストが落とした(`reason != ""` を外す変異
    だけは結果が変わらず生き残る)。自己点検の 5 項目はすべて clean
  - 計画からのずれ: S2 を #318 の merge を待たずに始めるよう、Implementation
    outline と Risks を直して承認し直した(#318 と触る節が別。図解ページの
    「見ておくこと」の 1 行も直した。図は変わらない。レーンの指示「推奨案を自動
    承認」による。2026-11-13、sha256:6dc6132083c1)
  - S2(implementer、sonnet): 98b243bb。leader.md、director.md、`/org` と
    `/director` の skill の 4 つの写しずつ、新しい prompt のテスト。slice の
    確かめのコマンドは exit 0(`internal/org` 160.821s、check-skill-sync と
    check-sync は PASS)。prompt を戻すと新しいテスト 2 本が落ちた。director.md の
    箇条は、launch_cmd 形の prompt に `pane` の語を出さない既存のテストに合わせて
    「leader の画面」と書いた。自己点検の 5 項目はすべて clean
- [x] Review artifact created — self-review は MEDIUM 1、LOW 6(40d7a240)。
  M1 と L1〜L5 は verify の前に S3 で直した。L6(テストの helper の置き場所)は
  書き方の好みなので残す
  - S3(implementer、opus): 0c41fbf8。この branch で足したテストを
    `send_screen_marker_test.go` と `send_fallback_marker_test.go` に移した
    (`send_screen_test.go` は 859 行で、増えたのは既存の表の 28 行だけ。
    `send_fallback_test.go` は bbe2807d と同じ)。leader の規則が CLI の note より
    優先すると書いた。規則を `sendTextInWant` にまとめた。知らない driver と、
    印のあとが空の行は箱を取らないようにした。計画はコメントで名前の形に直した。
    確かめのコマンドは exit 0(`internal/org` 278.151s)。L3・L4・L1 の変異は
    どれもテストが落とした。自己点検の 5 項目は 1 つ直して、あとは clean
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
