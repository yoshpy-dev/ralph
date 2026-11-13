# Verify report: claude-5h-usage-limit

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(issue #329)。`- Approved:` の digest `sha256:be05f6884990` は、`./scripts/plan-visual.sh digest` で計算した今の計画の digest と一致した
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は 1)
- Scope: `git diff origin/main...HEAD`(base `bbe2807d`、HEAD `fd81cbd9`)。19 ファイル、875 行追加、79 行削除。Go の本体は 4 ファイル(`internal/org/resume_screens.go`、`usage_limit_screen.go`、`watch_usage_limit.go`、`internal/cli/org_watch.go`)、テストは 4 ファイル、fixture は 1 つ。テストは読んだだけで、回していない(`/test` の範囲。この機械では別のセッションが全体のテストを回している)
- Evidence: `docs/evidence/verify-2026-11-13-claude-5h-usage-limit.log`(gitignore の対象。`run-static-verify.sh` が自分で書いた写しは `docs/evidence/verify-2026-11-13-082526.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 実機の画面を `UsageLimitScreen("claude", …)` が上限の画面と答え、行に `limit resets 3:50pm` と `Continuing automatically at 3:50pm` が入る | Met | `UsageLimitScreen` の waiting の分岐(`usage_limit_screen.go:78-83`)が `usageLimitWaitingFooter`(`:95-112`)を呼び、footer のブロックのうち `UsageLimitWaiting` の文(`resume_screens.go:246` の `Continuing automatically at` と `Continuing shortly`)を含む最初のブロックを 1 行にして返す。テストは `TestUsageLimitScreen_EveryFixture`(`usage_limit_screen_test.go:194-244`)の `claude-5h-limit-continuing.txt` の期待(`:32-35`)。ok、行の全体の一致、`has` の 2 つ(`limit resets 3:50pm`、`Continuing automatically at 3:50pm`)、行の形、waiting なら行が footer のブロックで印の文を含むこと(`:228-236`)を見る。AC の言うことと一致する |
| AC2: `ClassifyResumeScreen` は実機の画面も `claude-continuing-automatically.txt` も waiting と読む | Met | `ClassifyResumeScreen` の変更は footer の切り出しを `box.footerBlocks(driver, lines)` に替えた 1 行だけ(`resume_screens.go:358`)。新しい method(`:553-555`)は前と同じ `resumeScreenBlocks(lines[b.footerStart:], resumeMarker(driver))` を返す。テストは `TestClassifyResumeScreen_Fixtures` の 2 行(`resume_screens_test.go:89-103`)で、どちらも kind waiting、理由 `the CLI is retrying or waiting`、行の全体を固める。resume は waiting の画面に何も打たない(resume の仕様の「待ち中(打たない)」) |
| AC3: 待ちの文が入力欄より上にだけある画面と、`Retrying in`・`Waiting for API response` の待ちは上限の画面ではない | Met | `usageLimitWaitingFooter` は `box.footerBlocks` だけを読み、入力欄より上は読まない。`claude-continuing-automatically.txt` の期待は `{}`(`usage_limit_screen_test.go:61`)。`TestUsageLimitScreen_WaitingForALimitToReset`(`:394-455`)の、返事の最後の段落の引用(`usageLimitQuotedWaitScreen`、`:190-192`)、ツールの出力、footer の `Retrying in`、footer の `Waiting for API response` の 4 行が、`ClassifyResumeScreen` は waiting、`UsageLimitScreen` は `("", false)` を固める。`claude-retrying.txt` も `{}`(`:62`)。API の再試行の否定を footer に置いたのは、入力欄の上では待ちの道に入らず否定として弱いためで、計画の Progress の S1 に記録がある |
| AC4: `⎿  You've hit your 5-hour limit · resets …` が候補の頭にある画面は stopped で、上限の画面 | Met | `resume_screens.go:215`(`Stopped`、出典 `seen: Claude Code 2.1.316, …`)と `:222`(`UsageLimit`)。`TestUsageLimitScreen_FiveHourLimitUnderThePromptIsAStop`(`usage_limit_screen_test.go:371-383`)は `⎿` のあとに空白と U+00A0 を置いた画面で、stopped と、行 `⎿ You've hit your 5-hour limit · resets 3:50pm (Asia/Tokyo)` を固める。`TestUsageLimitScreen_EachStoppedSentence` も新しい文をグリフごとに通す |
| AC5: `UsageLimitWaiting` のどの文も同じ driver の `Waiting` の文。`UsageLimit` と `Stopped` の関係のテストは変えずに通る | Met | `TestUsageLimitWaitingSentences_AreWaitingSentences`(`usage_limit_screen_test.go:501-526`)が、印ごとに `Waiting` の文がちょうど 1 つ、重複なし、claude は 2 つ、codex は nil を固める。`TestUsageLimitSentences_AreStoppedSentences` は検査の中身が base と同じで、期待の一覧に `You've hit your 5-hour limit` が 1 つ増えただけ(`:464`)。計画の Scope が「文の一覧の期待(316 行あたり)」を直すと書き、D5 が求める変更なので、AC の「変えずに」は検査の中身を指すと読んだ |
| AC6: 見張りが実機の画面の座席を `usage_limited` に入れ、返事が引用した画面の座席は 2 周期以上 stall のまま | Met | `TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited`(`watch_usage_limit_test.go:700-721`): 2 周期で画面の読みは 1 回、stall の ALERT も active な stall もなく、`usage_limited` が implementer で立ち、件は 1 つで、本文の行は `implementer (claude): ` と footer の行。`TestWatchUsageLimit_ASeatWhoseReplyQuotesTheWaitKeepsItsStall`(`:729-751`): 2〜4 周期のそれぞれで stall が active、`usage_limited` の条件・ALERT・件・座席はなし、読みは `n-1` 回、stall の ALERT は 1 通。実機の見張りでこの道に入るかは、待ちのあいだ `herdr agent get` の出力が変わらないかによる(register 496 行、下の Coverage gaps) |
| AC7: `send_screen_test.go` の fixture の表に新しい行があり、`TestClassifySendScreen_EveryFixture` が通る | Met(通ることはこの run では見ていない) | 行は `send_screen_test.go:97-100`(busy、理由 waiting、行の全体)。`TestClassifySendScreen_EveryFixture`(`:141-179`)は行のない fixture で落ちる。`go vet ./...` がテストをコンパイルして通った。最後に回したのは self-review の 1 回目(`a851a653`、`-run 'ResumeScreenSentences|ClassifySendScreen|WatchUsageLimit|SendInputHolds'` が ok)。そのあとの commit は `send_screen.go`、`send_screen_test.go`、fixture、文の表を変えていない(`6cf988d5` は `ClassifyResumeScreen` の footer の切り出しを変えたが、`ClassifySendScreen` はそれを呼ばない) |
| AC8: 仕様 2 つ、`agent-messaging.md` の 2 写し、tech-debt の 452 行がこの振る舞いに合い、`check-sync.sh` が通る | Met(隣の文書の食い違いは下の V1〜V5) | `check-sync.sh` は PASS(IDENTICAL 172、DRIFTED 0)。`cmp` で 2 写しは同じ。org runtime の仕様の 2026-11-13 改訂(fix/claude-5h-usage-limit)の (a)〜(c)、resume の仕様の止まった文の一覧(58 行)と待ち中の行(60 行)、452 行の Trigger の列の #329 の注が、コードと合う |

### issue #329 の「直す方向の候補」と D1〜D5

| 候補 | 採ったか | 理由 |
| --- | --- | --- |
| claude の `UsageLimit` に `You've hit your 5-hour limit` を足し、`⎿` の行で見分ける | 採った(D5)。ただしこれだけでは直らない(D1) | `Stopped` と `UsageLimit` の両方に足した。実機の画面では `⎿` の行は候補にならず(候補は入力欄の上の `⏺ Usage limit reached · continuing …` の段落)、footer の `Continuing automatically at` が先に waiting を決めるので、この文が効くのは `⎿` の行が候補の頭に来る画面(AC4)だけ |
| `UsageLimit` は `Stopped` の文でもある決まりのまま、waiting の画面でも上限と見分ける。resume は打たないまま | 採った(D1、D2) | 待ち中の文に印を付ける欄 `UsageLimitWaiting` を足し、`Waiting` の文であることをテストで固めた(AC5)。resume の判定は変えていない(AC2)。注: 計画の Non-goals の見出し「`ralph org resume` の振る舞いを変えること」より、実際に守ったのは「待ちの画面に打たない」で、D5 により `⎿` の行が候補の頭に来る 5 時間の画面は unknown から stopped に変わり、resume が続行の文を打つ。D5 と resume の仕様の止まった文の一覧に書いてある意図した変更(INFO) |
| `⚠ Usage limit reached · limit resets …`(footer)と `⏺ Usage limit reached · continuing …` を claude の上限の印に使う | 印としては採らなかった。footer の `⚠` の行は返す行に入る(D4) | 印は同じ footer のブロックにある `Continuing automatically at` にした。`⏺` の段落は入力欄の上で引用と区別できず(D2)、待ちが終わっても小文字の形が履歴に残る(Assumptions の 3 つ目)。この候補を退けた理由は、計画に独立した決定としては書かれていない。上の 2 つから読み取れる(INFO) |
| `You've hit your … limit` の形でまとめて見分けるか、1 つずつ持つか | 1 つずつ(D3) | まとめると引用やほかの文と取り違える幅が広がる。#312a と同じく表の 1 行ずつにして出典を書く |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | PASS(rc=0) | fixture の `.txt` が言語に分類されないので、言語の範囲は full に落ちた(`unclassified:internal/org/testdata/resume/claude-5h-limit-continuing.txt`)。選ばれた pack は golang。hook の shellcheck、`sh -n`、awk の構文、`jq -e`: OK。`check-sync.sh`: PASS(DRIFTED 0)。`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照: OK。gofmt: ok。`go vet ./...`: 出力なし。golangci-lint: 0 issues。staticcheck: 出力なし(`Skipping` の行はない)。`secret-scan-branch` の `bbe2807d..fd81cbd9`: clean |
| `cmp .claude/rules/ralph/agent-messaging.md templates/base/.claude/rules/ralph/agent-messaging.md` | 同じ | AC8 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-claude-5h-usage-limit.md` | `be05f6884990` | 計画の `- Approved:` の行と一致 |
| fixture の `shasum -a 256` と行数、証拠の文書の 2 節の写しとの比較(python) | 一致 | sha256 `2b9742d9…8dd27` と 23 行は証拠の文書のとおり。2 節の写しは fixture の U+00A0(1 つ、8 行目)を空白にしたものと文字単位で同じ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `UsageLimitScreen` と `usageLimitWaitingFooter` の doc、ファイルの前書き(`usage_limit_screen.go`) | Yes | 2 つの道、footer だけを読むこと、返す行(待ちでは footer のブロックの全部)が本体と合う |
| `resume_screens.go` の `UsageLimitWaiting` の欄の doc と表のコメント | Yes | |
| `ralph org watch --help`(`internal/cli/org_watch.go:102-110`) | Yes | 止まった文と footer の待ちの 2 つを書く。codex に待ちの道はないが、括弧が footer と限っているので読み違えない |
| 件の本文、leader への ALERT、recurred の note、stderr の 2 行(`watch_usage_limit.go:47`・`280`・`285`・`297`・`310`) | Yes | どれも「hit a usage limit」。tech-debt 489 行の stderr の引用も合う |
| `agent-messaging.md` の 2 写しの `usage_limited` の段落 | Yes | |
| `AGENTS.md` の `internal/org/` の行 | Yes | 「reads as stopped on a usage limit or as waiting for one to reset」 |
| 証拠の文書 `docs/evidence/claude-5h-limit-screen-2026-11-13.md` | Yes | 4 節が挙げるテスト 4 つは HEAD に実在する。送った文の 126 バイトも計算で合う |
| tech-debt の 452 行、新しい 495 行と 496 行、489 行 | Yes(495 行の言い方は V4) | 452 行の注は Trigger の列(5 つのセルの 4 つめ)にある。495 行と 496 行はどちらも 5 セルで、互いに矛盾しない |
| org runtime の仕様の 2026-11-13 改訂(fix/claude-5h-usage-limit) | 一部(V4、V5) | |
| resume の仕様(`docs/specs/2026-11-07-org-seat-resume.md:58`・`60`) | Yes(INFO あり) | 60 行の claude の待ち中の一覧に `Waiting for API response` がない。表には `1f298056`(resume の最初の実装)からあり、この branch より前からの抜け |
| `/director` skill の「件の処理」の `usage_limited` の扱い(4 写し) | No | V1 |
| `/org` skill の「受信箱」節の `usage_limited` の項(4 写し) | No | V2 |
| tech-debt の、この PR が Trigger を満たした行 | No | V3 |

指摘はどれも LOW で、文書だけ。挙動と AC の判定は変わらない。`/sync-docs` で直すか、PR 本文の Known gaps に書くかを決める。

- V1(LOW): `/director` skill の「件の処理」(`.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/` の 2 写し)に、上限の解除を待つ座席について偽になった文が 2 つある。(1) 1037-1038 行「履歴の `recurred` の行の `note=` に座席が出ていれば、その座席も上限で止まったということ」: recurred で足された座席が待ちの座席なら、止まっていない(CLI が解除の時刻に自分で続ける)。note の本文は今「hit a usage limit: …」と言う。(2) 992-993 行の `REASON` の説明「`usage_limit` は org の座席が利用上限で止まったこと」: 待ちの座席だけの件でも `REASON` は `usage_limit` になる。1030 行の「画面が利用上限の文で止まっていると」と 1032-1033 行の「止まった座席は `SEATS:` の行に並ぶ」は、待ちの座席が抜けているが偽ではない。害は小さい: 座席の行に `Continuing automatically at …` が出て、resume は待ちの画面に何も打たない。ただ、同じ PR が rule 文書と件の本文の「stopped」を「hit」に直した理由(self-review N1、N2)は、この skill にも当てはまる。計画の Non-goals は skill を変えないと決めていて(レーン A の #318 とレーン B の #322 も skill を直している)、その判断は尊重する。直すなら、その 2 つが merge されたあとに「止まった」を「上限に当たった(止まったか、解除を待っている)」にする 1 句で足りる。直さないなら、PR 本文の Known gaps に 1 行
- V2(LOW): `/org` skill の `usage_limited` の項(`.claude/skills/org/SKILL.md:647-654` と 3 写し)は、条件の立ち方を「`ralph org resume` が利用上限と読む文で止まっていれば」とだけ書き、footer の待ちの道を書いていない。偽ではないが、director を置かない運用(`[org.director].enabled = false`)で人が読む契約の説明として欠ける。612-613 行の「org の座席が利用上限で止まったとき(`usage_limit`)」も同じ。計画の Scope にも Non-goals にも名前がない。V1 と同じ回に、同じ 1 句で直せる
- V3(LOW): この PR が Trigger を満たしたが、持ち越しの注がない行が 8 つある。前例(#312a の 477 行の注、#319 の 484 行の注)は「hit this trigger … but did not pay it … Carried over」の注を足している。
  - 457 行(`ralph org resume` の生き残りの変異 7 つ): Trigger は「the next change to … `ClassifyResumeScreen`」。`6cf988d5` が footer の切り出しを `footerBlocks` に替えた(挙動は同じ)。(d) は enter-only と待ちの両方を出す fixture を求めるが、新しい fixture は止まった文と待ちの文の組で、払っていない
  - 477 行(Hooks need review): Trigger は「the next change to the screen rules of `internal/org/resume_screens.go` or to the ALERT conditions of the watch」。表に 5 時間の上限の文と `UsageLimitWaiting` を足し、`usage_limited` が待ちの画面でも立つようにした
  - 484 行(`send_screen_test.go` と `send_fallback_test.go` の大きさ): Trigger は「The next change to either file」。`send_screen_test.go` は fixture の行 1 つで 831 行から 839 行
  - 485 行(読みが 1 回失敗した座席): Trigger は「the next change to … `recordUsageLimitRecurrence`」。`dd3b1e58` がその中の stderr の 2 行の語を変えた
  - 487 行(#312a のテストの穴): Trigger は「The next change to `internal/org/watch_usage_limit.go` or its test file」。どちらも変えた。(a) のテストは足しておらず、(c) の `recur` が失敗したときの stderr の行(`watch_usage_limit.go:280`、この PR が語を変えた)は今もどのテストも固めていない(race テストは出ないことだけを見る)
  - 488 行(`watch_usage_limit_test.go` の大きさ): Trigger は「The next change to this file」。936 行から 1000 行になり、fixture と助けの関数は移していない。行の「is 936 lines」と「this file did not grow」は今は古い
  - 489 行(grown の recurred の note の切り詰め): Trigger は「the next change to `usageLimitRecurNote` or `fitUsageLimitedSeats`」。`usageLimitRecurNote` の頭の文を「stopped on a usage limit: 」(26 文字)から「hit a usage limit: 」(19 文字)に変えた。この PR はこの行の stderr の引用だけを直し、注は足していない
  - 492 行: (a) の Trigger「The next change to `recordUsageLimitRecurrence`」を上の stderr の変更で、(b) の Trigger「the next edit of the `recurred`-note row」を 489 行の編集で満たした。489 行の Related の列は今も triage を「(WORTH_CONSIDERING 2)」と番号だけで指す
- V4(LOW): 理由の書き方が広すぎる。org runtime の仕様の改訂 (c)(91 行)は「resume の判定は待ちの文をどのブロックでも数える」、tech-debt 495 行の Why deferred は「the resume check counts a waiting sentence in any block」と書く。`ClassifyResumeScreen` が待ちの文を探すのは、入力欄のすぐ上のブロック、候補(claude の手番の終わりの行の 1 つ上)、footer のブロックだけ(`resume_screens.go:345-358`、画面の model の説明の 78-80 行の (b))。返事の古い段落の引用は数えない。結論(返事の最後の段落やツールの出力の引用で waiting になるので、入力欄の上を数えると stall が止まる)は変わらない。「入力欄のすぐ上のブロックと候補のどこにあっても数える」に直せば正確になる。計画の D2 と Progress も同じ言い方だが、計画の本文は digest の対象なので直さなくてよい
- V5(LOW): org runtime の仕様の改訂の最後の項「上の feat/watch-usage-limited の改訂 (a) と FR-8 の行は、この改訂の (a) と (b) を足して読む」は、feat/watch-usage-limited の改訂 (b) を挙げていない。その (b) は「`SEATS` の見出しに止まった座席が並び、本文の説明は org の座席が利用上限で止まったことを言う」と書くが、この PR のあと `SEATS` には待ちの座席も並び、本文の説明は「hit a usage limit (stopped, or waiting for it to reset)」になった。人と leader に届く文が変わったことは、仕様の改訂のどこにも書かれていない。FR-8 の印(142 行)も「止まった」と「2026-11-13 改訂 (a)〜(c)」のままで、同じ日の改訂が 2 つ以上になったので、どちらの (a)〜(c) かが読み分けにくい。最後の項に「(b)」と、本文の言い方が変わったことを 1 文足せば足りる

INFO(直さなくてよいもの):

- `docs/specs/2026-10-07-org-multi-org-director.md:119` の #312a の決定の注は「止まっているとき」と書く。日付のある決定の記録で、その日の時点では正しい
- `internal/org/watch.go` の comment 5 か所(254・316・1267・1315・2163 行)と `watch_usage_limit.go:152` は「stopped」のまま。`watch_usage_limit.go` の前書き(30-34 行)が「この comment の stopped は `UsageLimitScreen` が true の座席」と定めるが、その範囲は同じファイルの中だけ
- `send_screen_test.go` の名前の見張り `sendScreenNeverIdleName`(`:137`)は `continuing-automatically` を見るが、新しい fixture の名前 `claude-5h-limit-continuing.txt` には当たらない。行の期待が busy を固めているので穴ではない

## Observational checks

- `gh issue view 329 --repo yoshpy-dev/ralph` で issue の本文を読み、「直す方向の候補」の 4 つを D1〜D5 と突き合わせた(上の表)
- 証拠の文書の 2 節の写しと fixture を python で比べ、U+00A0 を空白にすると一致することを確かめた。base での結果(`ok=false`、waiting)は証拠の文書の記述によるもので、この run では `go test -overlay` を回していない

## Coverage gaps

- テストは回していない。AC1〜AC7 のテストが通ることは `/test` の側。`go vet ./...` がテストのコンパイルまでを確かめた
- 実機の見張りで、待ちの座席が `usage_limited` に入るか。`readUsageLimitScreen` は stall の判定が立つ座席だけに呼ばれ、待ちのあいだ `herdr agent get` の出力が変わらないかは確かめられていない(register 496 行)。変わるなら、この道は実機では動かない(stall の ALERT も出ない、#329 の前と同じ黙り方)
- `Continuing shortly` の画面と、ほかの幅、版、モデルの画面は実機で見ていない(証拠の文書の「確かめていないこと」、452 行)
- 待ちの道の行には footer のブロックの全部が入るので、`statusLine` の出力も件の本文に写る(self-review L4、計画 D4 で受け入れ済み)。同じ理由で、`statusLine` が印の文(`Continuing shortly` など)をそのまま出す設定なら、その座席は waiting の画面で `usage_limited` に入る。その場合も件はすぐ記録されるので黙る向きではない。おそらく起きにくい。未確認
- 計画の Test plan の変異の確かめ(印の照合を外す、footer の限定を外す、`Retrying in` に印を付ける)は `/test` の側。self-review が読みで確かめた `footerBlocks` を入力欄の上まで読む変異も、走らせてはいない

確信を上げる最小の追加の確かめ: `/test` で計画の S1 の検証コマンドと、`footerBlocks` を `resumeScreenBlocks(lines, marker)`(画面の全体)に替える一時の変異を 1 つ回す。`claude-continuing-automatically.txt` の `{}`、引用の 2 行、見張りの 4 周期のテストが落ちれば、AC3 と AC6 の否定の側がテストで守られていることが分かる。

## Verdict

- Verdict: pass
- Verified: AC1〜AC8(テストの中身は読みで、静的解析は `run-static-verify.sh` の rc=0 で)。計画の digest の一致。issue #329 の候補と D1〜D5 の対応。`agent-messaging.md` の 2 写しの一致、`check-sync.sh` の PASS。証拠の文書と fixture の一致
- Partially verified: AC7 の「通る」(行と網羅の検査は読んだ。最後に回ったのは `a851a653` の self-review で、そのあと関係するファイルは変わっていない)。文書の食い違いは V1〜V5 の LOW 5 件で、どれも `/sync-docs` か PR 本文で扱える
- Not verified: テストの実行、変異の確かめ、実機の見張りでの待ちの座席の扱い(496 行)、`Continuing shortly` と別の版の画面
