# Test report: claude-5h-usage-limit

- Date: run-test.sh は 2026-11-13 08:49:33Z から 09:10:12Z(log の更新時刻)に回った。名前で絞った回、変異、probe は 2026-11-15 09:56Z から 10:05Z に回した
- Plan: `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(`## Test plan`、AC1〜AC8。issue #329)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: HEAD `70120d4f`(`git diff origin/main...HEAD`、merge-base `bbe2807d`、20 ファイル、+976/-79)。run-test.sh は changed scope を求めたが、fixture の `.txt` が言語に分類されないので full に落ちた(`==> Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-5h-limit-continuing.txt)`)。選ばれた pack は golang だけ
- run-test.sh を回した者: 前の tester が同じ HEAD `70120d4f` で回し、報告を書く前に API エラーで止まった。この報告はその log の数字を使い、run-test.sh を回し直していない(handoff の指示。この機械では別のセッションが全体の回を回していた)。その回の load は記録されていない
- Evidence:
  - `docs/evidence/test-2026-11-13-claude-5h-usage-limit.log`(run-test.sh の生の出力。`run-verify.sh` 自身の写しは `docs/evidence/verify-2026-11-13-084933.log` で、test の log の末尾 2 行(空行と `Evidence saved to:`)がないほかは同じ)
  - `docs/evidence/test-2026-11-13-claude-5h-usage-limit-filtered.log`(この報告の名前で絞った回、変異 7 個の diff と出力、probe のテストファイルと出力)
  - 3 つとも `docs/evidence/*.log` の gitignore の対象で、commit されない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(前の tester、rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0、`FAIL: <1 以上>` の行 0) | 2 行 | 約 1,239 秒(08:49:33Z から log の更新時刻 09:10:12Z まで) |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 565.3 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 439.2 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/upgrade`、`internal/insights`、`internal/scaffold`、`internal/org/driver`、`internal/org/protocol`、`cmd/ralph` | パッケージ全体 | ok。`(cached)` は 0 | 0 | — | 9.5、2.4、2.1、2.0、1.8、1.4、1.4、0.4 秒 |
| 名前で絞った回: org を `-count=1 -v -coverprofile -run 'UsageLimit\|ClassifyResumeScreen\|ResumeScreenSentences\|ClassifySendScreen\|SendInputHolds\|WatchUsageLimit'`(計画の S1 の検証コマンドの絞り込み) | トップレベル 34 本(subtest 込みの RUN 387) | 387 | 0 | 0 | テスト 1.47 秒(build 込み 3 秒) |
| 変異 7 個(下の節)。同じ絞り込みに `-overlay` と `-vet=off` | 7 | 7 個とも死んだ | — | — | 1 個あたり数秒 |
| 変異のあとの対照: 同じ絞り込みを overlay なしで | 387 | 387 | 0 | 0 | 1.10 秒 |
| probe: overlay で足した使い捨てのテスト 1 本(Test gaps 3) | 1(subtest 3) | base で 3 | 0 | 0 | 1 秒 |

Skipped の 2 行は前の cycle の報告と同じで、`tests/test-plan-visual.sh` の browser detection(`/Applications` のブラウザが起動してしまうため)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらも手元の環境によるもので、この diff とは関係がない。

org の 565.3 秒は golang の pack の上限 30 分(`packs/languages/golang/verify.sh` の `-timeout=30m`)の 31% である。

load: run-test.sh の回の load は前の tester が記録しておらず、分からない。この報告の絞った回のあいだに見た 1 分の load は 6.99 から 144.49 だった。09:54Z ごろから別の worktree(`send-input-box-marker`)の `run-verify.sh` が回っていて、変異 b2 の回で 144.49 になった。base の名前で絞った回は 8.67、対照の回は 18.32 のときに回った。絞った回のテストは偽の時計と偽の herdr で動き、変異の回の失敗はどれも期待の値の食い違いで、時間切れは 0 である。

### 名前で確かめたテスト(handoff の step 2)

名前で絞った回の `-v` の出力で、次の 10 本が `--- PASS` だった(`--- FAIL` と `--- SKIP` は 0)。

- `TestUsageLimitScreen_EveryFixture`(subtest `claude-5h-limit-continuing.txt` を含む)
- `TestUsageLimitScreen_WaitingForALimitToReset`(subtest 6 本: footer の `Continuing shortly`、footer の `Retrying in`、footer の `Waiting for API response`、返事の最後の段落の引用、ツールの出力、待ちが終わったあとの小文字の行)
- `TestUsageLimitScreen_FiveHourLimitUnderThePromptIsAStop`
- `TestUsageLimitWaitingSentences_AreWaitingSentences`
- `TestUsageLimitSentences_AreStoppedSentences`
- `TestClassifyResumeScreen_Fixtures`(subtest `claude_waiting_for_its_5-hour_limit_to_reset_is_waiting` を含む)
- `TestClassifySendScreen_EveryFixture`(subtest `claude-5h-limit-continuing.txt` を含む。verify が回さなかった AC7 の「通る」はここで確かめた)
- `TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited`
- `TestWatchUsageLimit_ASeatWhoseReplyQuotesTheWaitKeepsItsStall`
- `TestWatchUsageLimit_TheItemBodyStaysWithinTheCap`(subtest `6_seats`、`20_seats`)

run-test.sh の Go の段は `-v` なしで回るので、テストの名前は run-test.sh の log に出ない。上の 10 本が run-test.sh でも回ったことは、`internal/org` が `ok` で `-run` がないことから言える。

## 受け入れ条件とテスト

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 実機の画面が上限の画面で、行に `limit resets 3:50pm` と `Continuing automatically at 3:50pm` | `TestUsageLimitScreen_EveryFixture/claude-5h-limit-continuing.txt` | 通った。変異 b1 で落ちる |
| AC2 実機の画面と `claude-continuing-automatically.txt` は waiting | `TestClassifyResumeScreen_Fixtures` の 2 行 | 通った |
| AC3 入力欄の上だけの待ちの文、`Retrying in`、`Waiting for API response` は上限の画面ではない | `TestUsageLimitScreen_EveryFixture/claude-continuing-automatically.txt`、`TestUsageLimitScreen_WaitingForALimitToReset` の否定の 4 行、`TestUsageLimitScreen_Edges/a_limit_the_CLI_waits_out_on_its_own` | 通った。変異 a、a2、b2、c で落ちる |
| AC4 `⎿` の 5 時間の上限の行は stopped で上限の画面 | `TestUsageLimitScreen_FiveHourLimitUnderThePromptIsAStop` | 通った。変異 d、d2 で落ちる |
| AC5 `UsageLimitWaiting` は `Waiting` の文。`UsageLimit` と `Stopped` の関係のテストも通る | `TestUsageLimitWaitingSentences_AreWaitingSentences`、`TestUsageLimitSentences_AreStoppedSentences` | 通った。変異 c、d、d2 で落ちる |
| AC6 見張りが実機の画面の座席を `usage_limited` にし、引用の座席は 2 周期以上 stall のまま | `TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited`、`TestWatchUsageLimit_ASeatWhoseReplyQuotesTheWaitKeepsItsStall` | 通った。前者は変異 a、b1 で、後者は変異 a、a2、b2 で落ちる |
| AC7 send の fixture の表に新しい行があり、`TestClassifySendScreen_EveryFixture` が通る | `TestClassifySendScreen_EveryFixture/claude-5h-limit-continuing.txt` | 通った |
| AC8 文書と `check-sync.sh` | テストの範囲ではない(verify が PASS を記録) | — |

## 変異

worktree の追跡ファイルには手を入れていない。変異はどれも scratch の写しに当て、`go test -overlay` で回した。変異と probe を回したあとに `git status --porcelain` が空であることと、probe のテストファイルが worktree にないことを確かめた。絞り込みは上の名前で絞った回と同じ。行番号はどれも `internal/org/` のテストファイルの行である。

| 変異 | 変えたところ | 結果 | 落ちた行 |
| --- | --- | --- | --- |
| a(handoff (a)、verify の提案) | `usageLimitWaitingFooter` の `box.footerBlocks(driver, lines)` を `resumeScreenBlocks(lines, resumeMarker(driver))`(画面のすべてのブロック)に替えた | 死んだ | `usage_limit_screen_test.go:217`: `EveryFixture` の `claude-continuing-automatically.txt`(`{}` の期待に行が返る)と `claude-5h-limit-continuing.txt`。`:358`: `Edges/a_limit_the_CLI_waits_out_on_its_own`。`:448`: 返事の最後の段落の引用、ツールの出力、footer の `Continuing shortly`。`watch_usage_limit_test.go:720`: 実機の画面の座席の件の本文。`:737` と `:740`(周期 2、3、4 のそれぞれ): 引用の座席の stall が消え `usage_limited` が立つ。`:748`: stall の ALERT が 0 通 |
| a2(a を絞った版) | 同じところを、入力欄より上のブロック(`lines[:box.aboveEnd]`)と footer のブロックを足したものに替えた | 死んだ | `usage_limit_screen_test.go:217`(`claude-continuing-automatically.txt`)、`:358`(`Edges` の同じ行)、`:448`(返事の引用とツールの出力)、`watch_usage_limit_test.go:737`・`:740`(周期 2〜4)、`:748`。肯定の側の行(実機の画面、`Continuing shortly`、実機の画面の座席の見張りのテスト)は通った |
| b1(handoff (b)、照合が当たらない) | `if strings.Contains(b.text, s)` を `if false && …` にした | 死んだ | `usage_limit_screen_test.go:217`(`claude-5h-limit-continuing.txt` が `("", false)`)、`:448`(footer の `Continuing shortly`)、`watch_usage_limit_test.go:711`(stall の ALERT が出る)、`:714`(`usage_limited` が立たない)、`:718`(件が 0) |
| b2(handoff (b)、照合を外して footer のどのブロックも数える) | 同じ行を `if true \|\| …` にした | 死んだ | `usage_limit_screen_test.go:217`(`claude-continuing-automatically.txt` と `claude-retrying.txt` が状態の行を返す)、`:358`(`Edges` の同じ行)、`:448`(footer の `Retrying in`、footer の `Waiting for API response`、返事の引用、ツールの出力)、`watch_usage_limit_test.go:737`・`:740`(周期 2〜4)、`:748` |
| c(handoff (c)) | claude の `UsageLimitWaiting` に `"Retrying in"` を足した | 死んだ | `usage_limit_screen_test.go:448`(footer の `Retrying in` の行が `✻ Can't reach the API · Retrying in 4s · attempt 13/300 🤖 Haiku 5.5 \| ⚡ low` を返す)、`:523`(印の一覧が 3 つ) |
| d(handoff (d)) | `You've hit your 5-hour limit` を `Stopped` と `UsageLimit` の両方から消した | 死んだ | `usage_limit_screen_test.go:375`(画面が unknown、`no known stop sentence above the input box`)、`:380`(`("", false)`)、`:484`(`UsageLimit` の一覧) |
| d2(d の片側) | `UsageLimit` からだけ消した | 死んだ | `usage_limit_screen_test.go:380`、`:484`、`:488`(止まった文が上限の語を持つのに印がない) |

読み取れること:

- a は handoff の期待どおり、作った fixture の `{}`、引用の 2 つの画面、周期を重ねる見張りのテストで落ちた。肯定の側の行も落ちたが、それは画面の全体を 1 度に切ると入力欄の罫線と `❯` が footer と同じブロックに入り、返す行が変わるためである。a2 は肯定の側を通したまま否定の側だけで落ちたので、footer に限ることは否定の側のテストだけで守られている
- 「`UsageLimitWaiting` の検査を外す」は 2 通りに読めるので、両方を回した。照合が当たらない向き(b1)は AC1 と AC6 の肯定の側が、どのブロックでも数える向き(b2)は AC3 の否定の側と、計画より前からある `Edges` の行が落とす
- d では AC1 の実機の画面は上限の画面のまま通る。実機の画面を決めるのは footer の待ちの道で、5 時間の上限の文は `⎿` の行が候補の頭に来る画面にだけ効く(計画の D1 と D5)

## Coverage

- Statement: 名前で絞った回の profile で、パッケージ全体の 12.3%(絞った回なので全体の数字としては意味がない)
- 変えた関数(`go tool cover -func`): `UsageLimitScreen` 100%、`usageLimitWaitingFooter` 88.9%、`resumeInputBox.footerBlocks` 100%、`ClassifyResumeScreen` 100%、`usageLimitRecurNote` 100%、`usageLimitedAlertMessage` 100%、`recordUsageLimitRecurrence` 79.2%
- Branch / Function: Go の標準の道具には branch coverage がない。上の関数ごとの数字で代える
- Notes:
  - `usageLimitWaitingFooter` の通らない区間は 102〜103 行の、入力欄が見つからないときの `return "", false` だけ。`ClassifyResumeScreen` が waiting と読むなら入力欄は見つかっているので、届かない防御の分岐である(コードのコメントのとおり)
  - `recordUsageLimitRecurrence` の通らない区間のうち 280〜282 行は、この PR が語を変えた stderr の行(recur が失敗したとき)である。Test gaps 6
  - `matchedStopSentence` の 80% の残りは、最後の `return resumeSentence{}, false`(126 行)で、この PR は変えていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、名前で絞った回、対照の回のどれにも落ちたテストはない | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #329: claude 2.1.316 が 5 時間の上限の解除を待つ画面を、見張りが stall にしていた | 直った | `TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited` が通る。変異 b1(待ちの道が当たらない、直す前と同じ振る舞い)で落ちる |
| #312a の上限の画面(session、weekly、codex の usage limit)は今までどおり上限の画面 | 変わらない | `TestUsageLimitScreen_EveryFixture` の 3 つの fixture、`TestUsageLimitScreen_EachStoppedSentence` が通る |
| 作った fixture `claude-continuing-automatically.txt` と `claude-retrying.txt` は上限の画面ではなく、resume は waiting と読む | 変わらない | `TestUsageLimitScreen_EveryFixture` の `{}` の 2 行、`TestClassifyResumeScreen_Fixtures` |
| send の画面の判定は fixture の全部で変わらない | 変わらない | `TestClassifySendScreen_EveryFixture`、`TestSendInputHolds*` |
| 件の本文が 2,000 字の上限に収まる(S3 で本文が長くなり、20 席のテストが 1 度落ちた) | 収まる | `TestWatchUsageLimit_TheItemBodyStaysWithinTheCap` の 6 席と 20 席 |
| パッケージ全体 | 通る | run-test.sh の Go 10 パッケージ(前の tester の回) |

## Test gaps

1. 実機の見張りで、上限の解除を待つ座席が `usage_limited` に入るか。`readUsageLimitScreen` は stall の判定が立つ座席にだけ呼ばれ、stall は `herdr agent get` の出力が `stall_minutes` のあいだ変わらないことで立つ。テストは偽の herdr の固定の出力で stall を立てるので、待ちのあいだに実機の出力が変わるかは確かめられない(tech-debt 496 行、計画の Assumptions の 2 つ目)。未確認
2. `Continuing shortly` の実機の画面と、ほかの幅、版、モデルの画面。`Continuing shortly` は作った画面の 1 行(`TestUsageLimitScreen_WaitingForALimitToReset` の最初の行)だけで、実機の画面は Haiku、幅 53 桁の 1 つだけ(tech-debt 452 行)
3. 計画の Test plan の edge case「プロンプトが待ちの文を引用した画面」に専用の行がない。overlay で足した使い捨てのテストで、base が 3 つの画面のどれも上限の画面と答えないことを確かめた。最後のプロンプトだけが引用する画面と、引用のあとに返事が続く画面は、`ClassifyResumeScreen` が unknown と読むので待ちの道に入らず、footer に限ることを試す行にならない(変異 a と a2 でも通る)。footer に `Retrying in` があり、プロンプトが `Continuing automatically at` を引用した画面は waiting と読まれ、変異 a と a2 で落ちる(`❯ It said Continuing automatically at 3:50pm · Esc to cancel` を返す)。a と a2 は既存の行でも死ぬので、足すとしても優先度は低い。足すなら `TestUsageLimitScreen_WaitingForALimitToReset` にこの画面の行を 1 つ
4. 止まった座席と待ちの座席が同じ件に入る組み合わせと、待ちの座席が recurred で足されるときの note。既存のテストの組み合わせと recurred は、止まった座席の行だけを使う。待ちの座席の行(実機の画面で 169 字)も同じ `usageLimitLine` と `fitUsageLimitedSeats` を通るので、おそらく同じ振る舞い。未確認
5. `statusLine` が印の文(`Continuing shortly` など)をそのまま出す設定では、その座席は waiting の画面で `usage_limited` に入る(verify の Coverage gaps)。テストはない。件はすぐ記録されるので、黙る向きにはならない
6. `recordUsageLimitRecurrence` の recur が失敗したときの stderr の行(`watch_usage_limit.go:280`)は、この PR が語を「hit a usage limit」に変えたが、どのテストも固めていない(名前で絞った回で通らない区間。tech-debt 487 行の (c)、この branch より前からの穴)
7. `ralph org watch --help` の usage_limited の段落(`internal/cli/org_watch.go:102-110`)は語を変えただけで、文を固めるテストはない
8. run-test.sh の回の load は記録がなく、`internal/org` の 565.3 秒と `internal/cli` の 439.2 秒がどれだけ混んだ機械での数字かは分からない

## Verdict

- Pass: 前の tester が HEAD `70120d4f` で回した run-test.sh(changed を求め full に落ちた scope、shell 45 本、Go 10 パッケージ、約 1,239 秒、rc 0、org 565.3 秒、cli 439.2 秒)、名前で絞った回(トップレベル 34 本、RUN 387)、対照の回が通った。handoff が名前を挙げた 10 本はどれも `--- PASS` で、AC7 の `TestClassifySendScreen_EveryFixture` も通った。変異 7 個(handoff の a〜d と、その読み分けの a2、b1/b2、d2)はすべて死に、AC1、AC3、AC4、AC5、AC6 のどれも、少なくとも 1 つの変異で落ちるテストで守られている。残る穴は Test gaps の 8 項目で、どれも通るべき挙動が落ちたものではない
- Fail: 0
- Blocked: なし
