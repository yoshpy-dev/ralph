# Walkthrough: 手番の中で再試行して週の利用上限を待つ claude の座席を、usage_limited と resume の待ちとして読む(watch-usage-limit-retry)

- 計画: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(PR の作成のあとに archive へ移す)
- issue: #353
- 差分: `git diff main...HEAD`(この walkthrough を足す前で 31 ファイル、+2,047 / −81。base は main の 26b9925c で、merge base も同じ。branch には origin/main を 0e05904e で取り込んである)。Go の本体は 7 ファイルで +334 / −48(大半はコメント)、Go のテストは 6 ファイルで +561 / −14、fixture は +15 行。残りは rule・skill の写し・README・仕様・tech-debt と、計画・報告・insight(+1,071)
- パイプライン: 上限の 2 周。1 周目の cross-review(codex)は ACTION_REQUIRED 1 件(P2)で、2 周目は 51fe7ddf で直した。2 周目の cross-review は指摘 0 件
- 読む順は下の節の番号のとおり。1 〜 3 が判定の本体、4 〜 5 が resume と見張りの側、6 〜 7 が fixture とテスト、8 が文書、9 が残したもの

## 1. issue が何だったか

- 機能の org の leader(driver は claude)が週の上限に当たった。Claude Code 2.1.316(座席の環境に `CLAUDE_CODE_RETRY_WATCHDOG=1`)は手番を閉じず、再試行して解除を待った。上限の文は `⏺ Monitor "..." stream ended` のブロックの `⎿  You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` にあった。入力欄のすぐ上は状態行だけで、狭い pane に折り返された `Weekly li… · Retrying in 29m (Nov 21 at 3pm) · attempt 1/55` だった
- 見張りは `CONDITION: stall`・`REASON: leader_is_anomaly_subject` の件を上げ、`usage_limited` にしなかった。director は通常の停止として読んだ。`ralph org resume --dry-run` は `waiting <org>/leader claude herdr reports working` と出した
- 原因は base の `UsageLimitScreen` が 2 通りしか読まないことだった(`git show main:internal/org/usage_limit_screen.go` の `case ResumeScreenStopped` と `case ResumeScreenWaiting`)。この画面は `Retrying in` で waiting と読まれる(`resume_screens.go:232`)ので 1 つ目の「stopped」に入らない。2 つ目は入力欄の下の footer に `Continuing automatically at` などを探すが、この画面の footer にはない。issue は原因を確かめていないと書いている。ここはコードを追った結果である

## 2. 3 通りの読み方(`internal/org/usage_limit_screen.go`、読む価値が一番高い)

`UsageLimitScreen`(`:91`)は、見張りと `ralph org resume` が共有する唯一の判定である。画面が次のどれかなら利用上限と読み、上限の行を返す。1 と 2 は base のまま、3 が今回足したもの。

1. stopped(`:98-103`): `ClassifyResumeScreen` が stopped と読み、決め手の文が driver の表の `UsageLimit` にある
2. waiting と footer(`:105-107`、`usageLimitWaitingFooter` は `:296`): waiting と読み、入力欄の下の footer のブロックに `UsageLimitWaiting` の文(`Continuing automatically at`、`Continuing shortly`)がある。2026-11-13 の 5 時間の上限の対応で足したもの
3. 手番の中の再試行(`:108-115`、`usageLimitRetryingTurn` は `:167`): claude だけ。waiting と読み、次の 4 つがそろう

   - 入力欄の上の最後のブロックが、claude の状態行の記号と空白(か文の終わり)で始まる(`claudeStatusLineStart`、`:178`、`:207`)
   - そのブロックが `Retrying in` と `attempt` を両方含む(`usageLimitRetryWords`、`:130`、`:181-185`)。順番は見ない。狭い pane は 2 列に折り返して語が入れ替わる
   - 今の手番(最後の prompt のブロックより後、prompt が画面にないときは入力欄の上のすべてのブロック。`usageLimitTurnBlocks`、`:219`)のどこかに、`⏺` か `⎿`、空白、上限の文、そのあとの `resets <時刻>` がある(`usageLimitResetsIn`、`:246`。`<時刻>` は `usageLimitResetTime`、`:277` が ` (` かブロックの終わりまでで切る)
   - 最後のブロックが、その `<時刻>` を括弧に入れた `(<時刻>)` を持つ(`:189`)

   時刻の一致が、今の再試行が上限によるものだという証拠になる。通信断の再試行(`Retrying in 4s · attempt 13/300`)は時刻を持たないので当たらない。上限の文が手番に複数あれば、時刻が一致する最後のものを使う(`:187-193`)。返す行は、その glyph からブロックの終わりまでを `usageLimitLine`(`:338`)で 1 行にし、300 字で切ったもの

- 2 が 3 より先に評価される。footer に待ちの文があれば footer のブロックが行になる(`usage_limit_screen_test.go:663` の行)
- 3 は `ClassifyResumeScreen` が waiting と読む画面にだけ働く。上限の文が手番にあって再試行のない画面は、unknown なら上限と読まず(`usage_limit_screen_test.go:692`)、stopped なら 1 つ目の読み方が答える(`:698`)

## 3. 2 周目で足した、状態行の始まりの条件(51fe7ddf)

- 1 周目の cross-review(`docs/reports/cross-review-triage-watch-usage-limit-retry.md` の ACTION_REQUIRED 1)の指摘。最初の実装は、最後のブロックが claude の状態行かを見ず、2 つの語を含むかだけを見ていた。`ClassifyResumeScreen` は待ちの文をブロックのどこにあっても数える。手番の時間の行を出さない設定の座席が、issue の 2 行を引用して手番を終えると、その返答が最後のブロックになる。見張りは stall を `usage_limited` に替え、resume は解除待ちの理由を出す。この repo の座席は issue #353 そのものを扱うことがあるので、起こりうる
- 直し: 最後のブロックが記号と空白で始まることを足した(`usage_limit_screen.go:178`、`:207`)。issue の画面は折り返しで `✻` 1 字の行から始まるが、ブロックの文は行を trim して空白 1 つでつなぐので `✻ Weekly …` になり、通る。`⏺` や `⎿` で始まる返答、語で始まる段落は外れる
- 計画からのずれ: 計画の Scope の 3 は状態行を「`Retrying in` と `attempt` を両方含む」とだけ書き、始まりの条件を持たない。Scope は承認の digest(`70b063e49ad8`)に入るので書き換えず、計画の Progress と org runtime の仕様の改訂 (a) (1) に記録した
- 記号の組は `send_screen.go:145` の `claudeStatusSymbols` = `*·✢✳✶✻✽` の 1 か所に置いた。send の `sendClaudeBusyLine`(`:154`)と `sendClaudeTurnEnd`(`:160`)の正規表現もここから組む。組み立てた文字列が base とバイト単位で同じことは、2 周目の self-review が scratch の Go プログラムで `cmp` して確かめた。send の判定は変わらない。ただし 3 つの用途が 1 つの定数を共有するので、定数を変えると send の busy 判定も動く(コメントに、正規表現の文字クラスに入るので `]`・`\`・`^`・`-` は使えないと書いた)

## 4. `ralph org resume`(`internal/org/resume.go`、`internal/cli/org_resume.go`)

- 変わるのは理由の文だけ。上限と読めた座席は outcome が waiting のままで、何も打たず、何も記録しない。理由は `waiting for the usage limit to reset: <上限の行>`(`ResumeReasonUsageLimitWait`、`resume.go:89`)
- herdr が idle か done と報告する座席は、これまでも pane を読む。waiting と読んだ画面で `UsageLimitScreen` が上限と読めたら、この理由にする(`resumeLook`、`:442`。分岐は `:465-467`)
- herdr が working と報告する座席は、これまで pane を読まずに `herdr reports working` と返していた。今は `resumeWorkingReason`(`:484`)が pane を 1 回読む(`resumeReadPane`、`:500`)。理由を変えるのは、画面が waiting で、かつ上限と読めたときだけ。上限で止まった画面(herdr が手番の終わりを取り逃がしたもの。CLI は自分では続けない)、通信断の再試行、pane の読みの失敗は `herdr reports working` のまま。この制限は self-review の MEDIUM(M1)の直し(cfe09a8f)。pane の読みの失敗を出さないのは計画で決めたことで、self-review の L2 として残した
- 5 時間の上限の footer の待ち(`claude-5h-limit-continuing.txt`)も同じ理由になる。これまでは `the CLI is retrying or waiting` だった(計画の Rollout)。挙動の変わる箇所なので release ノートに入れる候補
- CLI は、この理由のときだけ画面の行の代わりに理由を出す(`resumeTargetText`、`org_resume.go:203`)。idle の座席の `Line` は再試行の状態行(`… attempt li…      1/55` のように折り返したもの)で、上限の文は理由の側にしかない。working の座席は `Line` を持たない。help の文は `:54-58`
- herdr の呼び出しが増える。working の座席ごとに `herdr pane read` が 1 回。AC5 の「意図した変更」として、`TestOrgResume_RetryingOrWorking_Waits` の working の行の呼び出しの列に `read pane-1 60` を足した(`resume_test.go:343`)

## 5. `ralph org watch`(判定の呼び出しは変えていない)

- stall の条件が立った座席だけ pane を 1 回読み、`UsageLimitScreen` が上限と読めば `cycle.usageLimited` に足して stall を下ろす。この流れは base のまま(`watch.go:1322-1333`、`watch_usage_limit.go:113-133`)。leader の待ちの除外は、この読みのあとにある(`:1333`)
- つまり見張りの変更は、`UsageLimitScreen` の答えが 3 つ目で変わることで入る。件の形(`REASON: usage_limit`、`SEATS`、本文の座席ごとの行)、`recurred` の規則、届け方は変えていない
- コードの変更は `ralph org watch --help` の文(`org_watch.go:102-110`)に 3 つ目の読み方を足したことと、`watch_usage_limit.go` のコメントの「second way」を second and third ways に直したことだけ。`resume_screens.go:245-249` のコメントにも、`Retrying in` は印を付けないが 3 つ目の読み方が例外として読む、と足した

## 6. fixture(`internal/org/testdata/resume/claude-weekly-limit-retrying.txt`、15 行)

- issue の 2 つの断片(`⏺ Monitor …` のブロックの 3 行と、折り返した状態行の 3 行)を、折り返しの形のまま 3〜5 行目と 7〜9 行目に置いた。1 行目は短い prompt、11〜15 行目の入力欄と footer は `claude-weekly-limit-under-prompt.txt` の末尾 5 行。実機の画面の全体の capture ではない(`usage_limit_screen_test.go:46-58` の表の注にも書いた)
- この fixture は 4 つのテストが読む。resume の判定の表(`resume_screens_test.go:104-114`、kind waiting)、send の画面の表(`send_screen_test.go:132-138`、busy)、上限の表(`usage_limit_screen_test.go:55`、行は `⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`)、`TestUsageLimitScreen_EveryFixture`(`:263`。waiting の分岐を footer の行か再試行の行かのどちらかを許す形に広げた)

## 7. 先に読むテスト

- `usage_limit_screen_test.go:554` の `TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`(AC1)。fixture が waiting で、footer に待ちがなく、行が上限と時刻と時間帯を含む
- `:586` の `TestUsageLimitScreen_RetryWaitShapes`。上限と読む形は `:603-663`(折り返した文、`⏺`、U+00A0、時刻の前後の空白、上限が 2 つ、`✶` の状態行、300 字の切り詰め)。AC4 の読まない形は `:670` 以降: API の再試行だけ(`:672`)、前の手番の上限(`:677`)、返答の文の途中(`:682`)、tool の出力が引いた上限と時刻のない通信断(`:713`)、時刻が違う再試行(`:719`、`:724`)、codex(`:754`)。2 周目の 3 行は、引用した返答の読み(`:766`、`:771`、`:776`)
- `:804` の `TestUsageLimitScreen_ClaudeStatusLineStart`。記号 7 つと空白、`✻` 1 字、U+00A0 は true。`⏺`、`⎿`、語、`❯`、codex の `•`、記号の直後に語が続く `✻Weekly` は false
- `resume_test.go:383` の `TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit`(AC3)。working と idle の 2 通り、dry run の有無、5 時間の footer、pane の読みの失敗、上限で止まった画面(`herdr reports working` のまま)。何も打たず、記録せず、pane のロックも作らない
- `internal/cli/org_resume_test.go:360` は CLI の出力の全文を固める。`watch_usage_limit_test.go:731` は implementer と leader の 2 通りで、1 周期の screen read が 1 回、stall が上がらず、受信箱に `usage_limited` の 1 件だけが入ることを見る(AC2)
- 変異(test report)。1 周目は時刻の一致を外すと `RetryWaitShapes` の 5 行が落ちた。2 周目は始まりの条件を外す(引用した返答の 3 行が落ちる)、時刻の一致を常に真にする、記号の組から `✻` を外す(issue の fixture の行と send の行が落ちる)の 3 つがどれもテストを落とした

## 8. 文書

- 仕様: `docs/specs/2026-08-01-org-runtime.md:128-133` の 2026-11-20 改訂 (a)〜(c) と本文を書き換えない旨(始まりの条件、通信断を stall のままにする理由、残る取り違えを含む)。`docs/specs/2026-11-07-org-seat-resume.md:62` に、resume の判定は waiting のまま理由の文だけ変わる旨の改訂
- `.claude/rules/ralph/agent-messaging.md:199-204`(`templates/base` の写しも)に、再試行の形を `usage_limited` の段落へ足した。director skill は 4 つの写しの 1051-1057 行に、手番の中で待つ座席の見え方と `waiting for the usage limit to reset: <上限の行>` を書いた(`.agents` の 2 写しは `sync-skills.sh` で作った)。`README.md:302` に 1 文
- `/org` skill は変えていない。`usage_limited` の項は十分条件の書き方で、3 つ目を足しても偽にならないため。5 時間の footer の待ちと手番の中の再試行の待ちを書かないままになる(sync-docs report の「`/org` skill」)
- `docs/tech-debt/README.md`: 新しい行 4 つ(515 行の実機の画面がない既知の穴、542 行の 800 行を超えた 2 つのテストファイル、543 行の残る取り違え、544 行の self-review の L3・L4・L6)と、既存の 12 行を書き換えた持ち越しの注(Trigger に当たった行)

## 9. 残したもの

- 残る取り違え(tech-debt 543 行、コメントは `usage_limit_screen.go:156-160`): ブロックを平文で読むので、最後のブロックが記号と空白で始まる返答の段落(`* ` や `· ` の箇条書き、引用した再試行の行そのもの)で、`Retrying in`・`attempt`・`(<時刻>)` を持ち、同じ手番にその時刻の上限の文があれば、今も上限と読む。結果は誤った `usage_limited` の件で、人にすぐ届き、director を待つか別のアカウントへ向ける。resume は何も打たないので、理由の文が変わるだけ。この形を固めるテストの行はなく、Claude Code が返答の箇条書きを `*` で始めて描くかは確かめていない。2 周目の test report の probe は、この 2 形を実際に上限と読んだ
- 実機の画面の全体がない(515 行)。fixture は issue の断片から組み立てた。括弧の時刻と上限のリセットの時刻が同じことも 1 つの画面でしか見ていない。pane の幅で時刻が割れる、時間帯の括弧のない上限の文に別の行が続く、再試行の行の形が違う版は、どれも一致せず stall の件に戻る(黙る向きではない)
- stall が立った理由は確かめていない。2026-11-07 の改訂は「再試行中は herdr が working で `agent get` の出力が変わり続けるので stall は出ない」と書いたが、issue では stall が立った。見張りの状態ファイル `watch-status-<org_id>.json` は見ておらず、issue の報告から読んだ。ほかの待ち(通信断)で同じことが起きるかも確かめていない
- 名前と構造(544 行): `usageLimitReset*` が 4 つ近い名前、`resumeTargetText` が `strings.HasPrefix(t.Reason, ...)` で「上限の待ち」を伝える点、`agent-messaging.md` の再試行の文が入れ子の括弧に入っている点。どれも挙動は正しい
- 800 行: `usage_limit_screen_test.go` は 606 → 960 行、`resume_test.go` は 800 → 869 行(542 行)。`watch_usage_limit_test.go` は 1,000 → 1,035 行で、この branch の前から 800 行を超えていた(別の既存の行がある)
- 2 周目の `./scripts/run-test.sh` は rc 1 だった。落ちたのはこの branch が触らない `TestGate_TwoGatesRunTheirScriptsOneAtATime`(gate.lock の待ちの競合)で、単独の 20 回と `internal/org` 全体の打ち直しは通った(`docs/reports/test-2026-11-20-watch-usage-limit-retry.md` の Failure analysis)。rc 0 の run-test.sh は 2 周目にない。2 周目のテストは 0ea15fd2 で走り、そのあとに origin/main を取り込んだ(0e05904e)。この branch が触る 14 のファイル(Go と fixture)は、最後の Go の commit a5ec5464 から HEAD まで変わっていない(`git diff --quiet a5ec5464 HEAD -- <14 ファイル>` で確かめた)
- 私はテストを回していない。上の結果は報告から読んだもの
