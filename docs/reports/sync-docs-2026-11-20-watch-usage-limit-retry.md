# sync-docs report: watch-usage-limit-retry

- Date: 2026-11-20(パイプラインの cycle 2。`cycle-count.json` は `cycle: 2`)。cycle 1 の run は下の「## Run 1」に、1 バイトも変えずに残した
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(承認の digest `70b063e49ad8`。この run の前後とも同じ)。issue #353
- Branch: `fix/watch-usage-limit-retry`。run 開始時の HEAD は `0e05904e`(origin/main の `26b9925c`、#356 を取り込んだ merge)
- 先行 report: cross-review の triage `bd55da61`(cycle 1、ACTION_REQUIRED 1)、self-review の Run 2 `9d5ae3a4`(LOW 5: L7〜L11)、verify の cycle 2 `0ea15fd2`(pass、LOW 4: V7〜V10)、test の cycle 2 `ddeb39f7`(pass)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: cycle 1 の sync-docs のあとの、この branch 自身の Go の commit は 2 つ。`51fe7ddf` は cross-review の ACTION_REQUIRED 1 の直しで、3 つ目の読み方が、入力欄の上の最後のブロックの始まりも求める(`claudeStatusLineStart`。記号の組 `claudeStatusSymbols` は send と共有)。`a5ec5464` はコメントと `ralph org watch --help` の改行だけ。`d729352e` など、ほかの Go の commit は main の #356 から入ったもので、この report が数える 9 つのファイルのどれにも触れていない(9 ファイルに絞った `git diff --stat a5ec5464 HEAD` が空)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go のファイルは変えていない。計画の本文(digest に入る節)は変えず、Progress に行を足しただけ
- 差分の大きさ(`/pr` が walkthrough を決める材料。この run の commit の前の数): `git diff 26b9925c...0e05904e` は 31 ファイル、+1,857/-79(計画、report、insights を含む)。Go のテストでないコードは +334/-48(382 行)、Go のテストは +561/-14、fixture は +15 行。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary(cycle 2)

verify の V7〜V10 と、self-review Run 2 の L7(register の部分)・L8・L9 を直した。依頼の 7 項目のうち 6 つは文書を直し、7 つ目(README、`agent-messaging.md`、director skill)は確かめて変えなかった。

自分の掃引で、依頼の一覧にない 2 点も直した。(1) tech-debt の 490 行と 494 行の注にある「at 29b5d5d6, the last Go commit of the branch」は、`51fe7ddf` と `a5ec5464` のあとは言い方が古い(数は変わらない)ので、「and the same at a5ec5464」を足した。(2) 515 行の Impact の「none makes a wrong `usage_limited` item」は外れる側の話なので、取り違えが別の行にあることを指す 1 文を足した。

README、`agent-messaging.md`、director skill の 4 つの写し、resume の仕様、`/org` skill は変えていない。skill を変えていないので `sync-skills.sh` は回していない。

## Changes made(cycle 2)

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-20 改訂の (a)(1) に始まりの条件、(b) に引用した返答が stall に戻ることと、残る取り違え | `23ff1bdb` |
| `docs/tech-debt/README.md` | 持ち越しの注 2 行(send-screen-fallback の findings、send-codex-paste の (h)。(h) の 788 行を 798 行に)、cycle 1 の注の数の直し(362、490、494、517 と、800 行を超えたテストの行)、515 の指す文、新しい 2 行 | `b3ce2595` |
| `docs/plans/active/2026-11-20-watch-usage-limit-retry.md` | Progress に 6 行(cross-review cycle 1、51fe7ddf、self-review cycle 2、verify cycle 2、test cycle 2、sync-docs cycle 2)。`git diff --numstat` は 6/0 で、digest は `70b063e49ad8` のまま | `799249d5` |
| `docs/reports/sync-docs-2026-11-20-watch-usage-limit-retry.md` | この report(cycle 2 を上、cycle 1 を下) | この report の commit |
| `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` | `sync_docs` の行(cycle 2、pass) | この report の commit |

tech-debt の行は、HEAD で 362、488、490、494、515、517、524、542 が変わり、543 と 544 が新しい。行番号は main の merge で動くので、見つけるときは行の最初の語を使う。

## 依頼の 7 項目

### 1. V7: 仕様

- org runtime の仕様の 2026-11-20 改訂(`23ff1bdb`)。(a)(1) は「再試行の状態行(`Retrying in` と `attempt` を両方含む)」だけだったので、最後のブロックの文(行を trim して空白 1 つでつないだもの)が `*·✢✳✶✻✽` のどれかと、空白か文の終わりで始まる条件を足した。記号の組が `send_screen.go` の `claudeStatusSymbols` で、send の 2 つの正規表現と同じ組であること、issue の画面の状態行が折り返しで `✻` 1 字の行から始まるが、ブロックの文では `✻ Weekly …` になって条件を通ることも書いた
- (b) には、issue の 2 行を引用して手番を終えた返答の画面が stall のままであること(`⏺` で始まる返答、`⎿` から引用する段落、語で始まる段落が引用を含む画面が、(a)(1) で外れる)と、知られた取り違えを足した。取り違えは、最後のブロックが記号と空白で始まる返答の段落(`* ` や `· ` の箇条書き、引用した再試行の行そのもの)が、再試行の語と `(<時刻>)` を持ち、同じ手番にその時刻の上限の文があれば、今も `usage_limited` の件になる、というもの。resume は何も打たず、理由の文だけが変わる。テストの行がないことと、Claude Code が返答の箇条書きを `*` で描くかは未確認であることも書き、tech-debt の新しい行(下の 5)を指した
- resume の仕様の FR-3 の改訂(2026-11-20)は変えなかった。「`Retrying in … (<時刻>) · attempt n/m` の状態行」と書くだけで、2 つの語による定義も始まりの条件も書かないので、偽にならず、条件を書く場所でもない。verify の V7 も「任意」とした

### 2. V8: 計画の Progress

6 行を足した(`799249d5`)。cross-review cycle 1(codex の P2 を ACTION_REQUIRED 1 件に、triage の commit)、`51fe7ddf`(計画からのずれ(Scope の 3)として。Scope の 3 は状態行を「`Retrying in` と `attempt` を両方含む」とだけ書くので、ブロックの始まりも求めたことを書き、Scope は digest に入るので書き換えていないことを添えた)、self-review cycle 2(L7〜L11、`a5ec5464` が L7・L10・L11 をコメントと help の改行で直した)、verify cycle 2(pass、V7〜V10)、test cycle 2(pass、変異 3 つ、run-test.sh の rc 1 は無関係の `TestGate_TwoGatesRunTheirScriptsOneAtATime` だけ)、sync-docs cycle 2。`./scripts/plan-visual.sh digest` は編集の前後とも `70b063e49ad8`、`git diff --numstat` は 6 行足して 0 行消した。

### 3. V9: 持ち越しの注

どちらも `51fe7ddf` の `send_screen.go` の変更が Trigger を満たした。形は 488 行の #326 の注と 524 行の既存の注に合わせた(Trigger のセルの末尾に日付つきの注、Related に計画の active のパス)。

| 行(先頭の語) | 当たった Trigger | 注に書いたこと |
|------|------|------|
| 488(Seven small findings of fix/send-screen-fallback) | (L2, L7, L9, L10) の「the next change to `send_fallback.go`, `send_screen.go`, or `status_live.go`」 | `claudeStatusSymbols` を足し、2 つの正規表現をそこから組んだ(文字列は同じ、`ClassifySendScreen` は触っていない)。払わなかった理由は、計画の Scope が上限の読み方で、4 項目のどれも状態行の記号の話でなく、定数は「driver に画面の規則があるか」を問う場所を増やさない(L7 に新しい場所はない)こと。(L6) の問い(返答の箇条書きを Claude Code がどう描くか)が、新しい行(下の 5)の未確認の点と同じで、1 回の capture で両方に答えが出ることも書いた |
| 524(Known gaps of fix/send-codex-paste)の (h) | 「The next change to `send_screen.go`: move the paste rules ... before the next rule is added」 | 払わなかった理由は、move-only の分割が計画の Scope の外であること。この commit でファイルが 788 から 798 行になり、800 行の目安まで 2 行であること。1 つめのセルの (h) の「788 lines」は 798 に直した(`wc -l` を `a5ec5464` で。788 は #326 の branch の終わりと、この branch の merge-base `1d05c9ac`、735 は #326 の前) |

### 4. V10 と L8: cycle 1 の注の数

数え直した commit は `a5ec5464`、この branch 自身の最後の Go の commit(`git log` の先頭は main から入った Go の commit だが、数えるファイルには触れていない。上のとおり)。cycle 1 の数え直しの commit `29b5d5d6` のあと、9 つのファイルのうち動いたのは `51fe7ddf` が変えた `usage_limit_screen.go`、`usage_limit_screen_test.go`、`send_screen.go` と、`a5ec5464` が変えた `usage_limit_screen.go` と `org_watch.go`(コメントと改行だけ)。

| ファイル | cycle 1 の注 | `a5ec5464` の `wc -l` |
|------|------|------|
| `usage_limit_screen.go` | 317(base 152) | 353 |
| `usage_limit_screen_test.go` | 892(base 606) | 960 |
| `send_screen.go` | 788 | 798 |
| `resume_test.go` | 869 | 869 |
| `watch_usage_limit_test.go` | 1035 | 1035 |
| `watch_usage_limit.go` | 485 | 485 |
| `send_screen_test.go`、`send_fallback_test.go` | 885、835 | 885、835 |

- 362 行: 数を 353 と 960 に直し、「at a5ec5464」に変えて、cycle 1 の値を括弧に残した。`51fe7ddf` と `a5ec5464` が見張りの escalation の code に触れていないことも 1 文で書いた(この行の Trigger の境目)
- 517 行: 「606 to 892 lines」を「606 to 960 lines」に直し、足したテストの列に `TestUsageLimitScreen_ClaudeStatusLineStart` を足した。`51fe7ddf` が足した 3 つの引用の行と `✶` の行も書いた。`51fe7ddf` と `a5ec5464` が Trigger の 3 つ(`UsageLimitScreen` の変更、`usage_limit_screen_test.go` へのテストの追加、`newOrgWatchCmd` の段落)をもう一度満たしたこと、払わなかったことを書いた
- 542 行(cycle 1 の run では 540 行。main の merge で 2 行増えて動いた): 606 から 960(+354)、#353 の block の範囲を 526〜831(約 306 行)、`TestUsageLimitScreen_RetryWaitShapes` を 30 subtests・約 210 行、移すテストを 3 つにした。移したあとの大きさ「約 650 行」は 960 から 306 を引いて 654 なので、そのまま。Trigger の移すものの列、Related(verify の V10 と self-review の L8)も直した
- 490 行、494 行: ファイルの数(885 と 1035)は変わらない。「at 29b5d5d6, the last Go commit of the branch」は古い言い方になったので、「and the same at a5ec5464」を足した(依頼の一覧にない、自分の掃引で見つけたもの)

### 5. L7 の register の部分: 残る取り違え

新しい行を足した(543 行目、行の最初の語は「Known gap of fix/watch-usage-limit-retry (#353), the residual misreading of the third way of `UsageLimitScreen`」)。内容は、始まりの条件のあとも、記号と空白で始まる返答の段落が再試行の語と `(<時刻>)` を持ち、同じ手番にその時刻の上限の文があれば、今も上限と読むこと。cycle 2 の test report の probe が 2 つの形(引用した再試行の行で始まる段落と、`⏺` のブロックの後ろの `* ` の段落)で、`ClassifyResumeScreen` が waiting、`UsageLimitScreen` が上限の行を返すことを確かめている。Impact は、誤った `usage_limited` の件が人と director にすぐ届くこと、resume は何も打たないこと。Trigger は、引用の `usage_limited` の件、画面の capture、`usageLimitRetryingTurn` か `claudeStatusLineStart` の次の変更。Debt の最後は、`usageLimitRetryingTurn` のコメントに書いてあること(`a5ec5464`)。

515 行は、外れる側(stall に戻る形)の記述のままにした。Impact の末尾に、誤った件を作る読み方は新しい行にあることを指す 1 文だけを足した。

### 6. L9: Run 1 の L3、L4、L6

新しい行を 1 つ足した(544 行目、行の最初の語は「Three LOW findings of the cycle 1 self-review of fix/watch-usage-limit-retry (#353), L3, L4, and L6」)。HEAD で 3 つとも残っていることを確かめた。L3: `usage_limit_screen.go` の `usageLimitResetsWord`、`usageLimitResetsIn`、`usageLimitResetTime` と型 `usageLimitReset`。L4: `internal/cli/org_resume.go` の `strings.HasPrefix(t.Reason, org.ResumeReasonUsageLimitWait)`。L6: `agent-messaging.md` の、待ちの括弧の中に入れ子になった再試行の括弧(read で確かめた。`templates/base` の写しは `check-sync.sh` が同一と判定)。Trigger は 3 項目それぞれの次の変更で、self-review の勧めの形にした。

### 7. README、`agent-messaging.md`、director skill

変えなかった。3 つとも、この読み方を「claude's retry status line」「再試行の行」と書き、`Retrying in` と `attempt` の 2 語で定義も、始まりの条件の列挙もしない。始まりの条件が外すのは、引用した返答の段落だけで、本物の状態行は今までどおり当たるので、どの文も偽にならない。`agent-messaging.md`(`usage_limited` の段落)と director skill(`### 件の処理`)の例には先頭の `✻` がないが、例は条件の定義ではない(verify の INFO と同じ読み)。条件を書く場所は org runtime の仕様の (a)(1) だけで、そこは直した。README の 302 行は「a retry line above the input box whose time is the reset time」と書くだけで、同じ理由で変えない。

`check-sync.sh` は PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-skill-sync.sh` は 14 skill が lock-step。skill に触れていないので、`sync-skills.sh` も `templates/base` への `cp` も要らなかった。

## Trigger の掃引(自分でやり直した)

python で register の Trigger のセル(5 セルの 4 つめ)を、取り消し線の部分を除いて走査した。トークンは、cycle 2 の Go が触った名前(`send_screen`、`sendClaude`、`claudeStatus`、`ClassifySendScreen`、`usage_limit_screen`、`UsageLimitScreen`、`usageLimit`、`newOrgWatchCmd`、`org_watch`、`status line`)と、この run が編集した文書(`org-runtime`、`runtime spec`、`docs/specs`、`docs/tech-debt`、`docs/plans/active`、`Progress`)。cycle 2 が触った名前(`sendClaudeBusyLine`、`sendClaudeTurnEnd`、`claudeStatusSymbols`、`claudeStatusLineStart`、`usageLimitRetryingTurn`、`usageLimitRetryWords`、`isClaudeStatusBlock`)を register の全文から探すと、この run の前に出ていたのは 515 と 517 だけだった。

- 当たって注を足した: 488、524(verify の V9)
- 当たり、cycle 1 の注を直した: 362、517(数と cycle 2 の 1 文)。490、494(言い方だけ)
- 当たらなかった(Trigger を読んで外した): 489(6)の「the next change to `ClassifySendScreen`, ...」(`ClassifySendScreen` は変わらず、使う 2 つの正規表現の文字列も base と同じ)、518(`ClassifySendScreen` が見つけない箱の話)、526(`sendClaudePlaceholder` は変えていない)、490 の Trigger(send のテストは触っていない)、370(Trigger の +300 行に届かない。`send_screen.go` は +10 行)、213・462・491・492(cycle 2 が触った名前が前の注か本文に出るだけ)、455・456・457・465・466(`resume_screens.go` は cycle 2 で変わっていない)
- 既に cycle 1 の注がある行(483、464、493、516、514、267): `51fe7ddf` は 3 つ目の読み方を狭めただけで、どの注の記述も偽にならないので、足さなかった
- 自分の編集が当たる行: 仕様、計画、tech-debt の編集を Trigger に挙げる行を探した。376 行の Trigger は `docs/evidence/` の引用の言い方の話で、この run は evidence を引いていない。387 行の「runtime spec」は receipts の話。79 行は取り消し線つき。どれも当たらない

## Run 1 の記述の訂正

Run 1 の本文は変えていない。今は合わない記述は次のとおり。

| Run 1 の記述 | 今 |
|------|------|
| 「見た範囲」の「Go の commit は 4 つで、このあと Go は変わっていない」 | この branch 自身の Go の commit は 6 つになった(`51fe7ddf` と `a5ec5464` が増えた) |
| 「V5」の `usage_limit_screen_test.go` 606 → 892、`TestUsageLimitScreen_RetryWaitShapes` が 26 subtests・約 185 行、#353 の block が 526〜768 行・約 240 行 | 960、30 subtests・約 210 行、526〜831 行・約 306 行(上の 4)。移したあとの大きさ「約 650 行」は変わらない。この行は 540 行目から 542 行目に動いた |
| 「V4」の表の 362 行の注が数える `usage_limit_screen.go` 317、`usage_limit_screen_test.go` 892 | 353、960(上の 4) |
| 「掃引で当たらなかった候補」の 488・489(「fixture という語はあるが、実機の画面を取る Trigger」)と、518・524(「Trigger の `agent-messaging.md` は Ctrl+C の手順」) | 488 の (L2, L7, L9, L10) と 524 の (h) は、`51fe7ddf` が `send_screen.go` を変えて当たった。cycle 1 の run が見た commit では、`send_screen.go` に触れていなかった。489 と 518 は今も当たらない |
| 「V2」の org runtime の仕様の改訂 (a) の 3 つの条件 | (a)(1) に始まりの条件が入った(上の 1)。条件の数は 3 つのまま |

## 確かめたこと(cycle 2)

| コマンド | 結果 |
|------|------|
| `./scripts/plan-visual.sh digest <plan>` | `70b063e49ad8`(編集の前後とも、`- Approved:` の行と同じ) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。仕様、計画、tech-debt の編集のあと |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0。末尾に `tech-debt README plan references` が `OK`(欠けた計画の参照なし) |
| `git show HEAD:docs/tech-debt/README.md` との照合(python) | 変わった行は 362、488、490、494、515、517、524、542 と、末尾の 2 つの新しい行(543、544)だけ。どの行も 5 つのセル、バッククォートと取り消し線は各セルで偶数、U+FFFD は 0、足した字に `file:line` の参照はない。元の各セルは挿入か置換の差分だけで、削った文は 362 行の数の直しだけ |
| 仕様の diff | 2 行が変わった(改訂の (a)(1) と (b))。U+FFFD は 0 |

回していないもの: `./scripts/run-test.sh`(依頼の指示)、`./scripts/run-verify.sh` の全体、`gofmt`・`go vet`・`go test`(Go は変えていない)。

## 残したもの

- test report が勧めた `TestGate_TwoGatesRunTheirScriptsOneAtATime` の競合(gate.lock の待ち)の行か issue: 依頼の 7 項目になく、この branch の外の問題なので、tech-debt に足していない(`grep` で `TwoGates` と `not written yet` は register に 0 件)。足すかどうかは orchestrator が決める
- 残る取り違えを固めるテストの行、返答の箇条書きを Claude Code が `*` で描くかの確認: コードの変更になるので、行(543)に未確認として書くだけにした
- `/org` skill の `usage_limited` の項、resume の仕様の FR-3 の改訂、README の 302 行: 上の理由で変えていない

## Insight event(cycle 2)

`./scripts/insights-append.sh --slug watch-usage-limit-retry --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` に 1 行を足した(`cycle` 2、ts は 2026-11-20T11:56:04Z)。exit 0。`git status --short` は、この行を足した既存の jsonl が `M` で、ほかに新しいファイルはなかった(slug に日付を付けていない)。

---

## Run 1(cycle 1。sync-docs の commit `4ae843df`・`48cffdb9`・`d73be884`・`46b6b4b1`、この report の commit `944a7820`)

以下は cycle 1 の report を 1 バイトも変えずに残したもの。数と行番号は、その run を行った時点のものである(今の値は上の「Run 1 の記述の訂正」)。

# sync-docs report: watch-usage-limit-retry

- Date: 2026-11-20
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(承認の digest `70b063e49ad8`)。issue #353
- Branch: `fix/watch-usage-limit-retry`。merge-base は `1d05c9ac`。origin/main の `a672b72a`(別の spec 1 つだけを足した #355)は `f4b3a7c4` で取り込み済み。run 開始時の HEAD は `f4b3a7c4`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `94fa8031`(MEDIUM 1、LOW 6。M1 と L1 は `cfe09a8f` で直した)、verify `f8303063`(pass。LOW の V1〜V6)、test `f246da8f`(pass)。`29b5d5d6` は verify の V1 と V3 を直した commit。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff 1d05c9ac...HEAD`。Go の commit は `10541be6`、`e57ddb65`、`cfe09a8f`、`29b5d5d6` の 4 つで、このあと Go は変わっていない
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go のファイルは変えていない。計画の本文は変えず、AC の 6 つの箱と Progress だけを変えた(下の V6)
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff 1d05c9ac...f4b3a7c4` は 26 ファイル、+1,337/-57(計画、report、evidence、insights を含む)。Go は +770/-50、そのうちテストが +493/-14、テストでないコードが +277/-36(313 行)、fixture が +15 行。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify の V2、V4、V5、V6 は直した。V1 と V3 は `29b5d5d6` で直し済みなので、そのとおり記録するだけにした(コードで確かめた: `internal/cli/org_watch.go` の help の括弧に 3 つ目の読み方があり、`ResumeReasonUsageLimitWait` の doc に「herdr が working で画面が上限で止まっているときは `herdr reports working` のまま」がある)。

Trigger の掃引は自分でもやり直した。verify が挙げた 6 行(464、490、493、494、516、517)はどれも当たっていた。ほかに 3 行(267、362、483)が当たっていて、verify と handoff はこの 3 行を挙げていなかった。9 行に持ち越しの注を足した。

README の 302 行に 1 文足した。この 1 文は handoff の一覧にない。`/org` skill は変えなかった(理由は下の「`/org` skill」)。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-11-20 改訂(fix/watch-usage-limit-retry)」の項を、最後の改訂の項(2026-11-15 feat/reseat-leader-progress)の次、`### 運用ノート` の前に足した。(a) 手番の中の再試行の待ちを `usage_limited` の対象にしたこと(3 つの条件)、(b) 通信断の再試行は stall のままであること、(c) 変えていないもの、と「本文は書き換えない」の項。`4ae843df`。下の V2 |
| `docs/specs/2026-11-07-org-seat-resume.md` | FR-3 の末尾に「改訂(2026-11-20、fix/watch-usage-limit-retry、issue #353)」の段落を 1 つ。同じ `4ae843df` |
| `docs/tech-debt/README.md` | 持ち越しの注 9 行(464、490、493、494、516、517、267、362、483。Related にも計画)、494 の Debt の末尾に 1035 行の 1 文、新しい行(540 行目。800 行を超えた 2 つのテストファイル)。`48cffdb9`。下の V4、V5 |
| `README.md` | 302 行(`[org].api_wait_minutes` の段落)の末尾に 1 文。`d73be884` |
| `docs/plans/active/2026-11-20-watch-usage-limit-retry.md` | AC1〜AC6 の 6 つの箱を `[x]`、Progress の `Test artifact created` を `[x]`(report と SHA と要点つき)、`Sync-docs: pass` の行を足した。同じ `d73be884`。下の V6 |
| `docs/reports/sync-docs-2026-11-20-watch-usage-limit-retry.md` | この report |
| `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` | `sync_docs` の行(cycle 1、pass) |

commit は 5 つ。`4ae843df`(2 つの仕様)、`48cffdb9`(tech-debt)、`d73be884`(README と計画)、`46b6b4b1`(tech-debt の新しい行の行番号を名前に直した)、そしてこの report と insights の commit。

## verify が挙げた drift

### V1、V3(`29b5d5d6` で直し済み)

どちらも Go の中の文字(help の括弧、doc comment)で、`29b5d5d6` が直した。この pass では触れていない。V1 の help を固めるテストはない(register の 517 行の (d)。下の V4 で注に書いた)。

### V2: 仕様の改訂

- org runtime の仕様: 2026-11-13 改訂(fix/claude-5h-usage-limit)の (b) は書き換えず、新しい項の「本文は書き換えない」で「この改訂の (a) の 3 つがそろう座席を除いて読む」とした。その項は同じ改訂の (c)(待ちの文が入力欄より上にだけある画面は対象外)にも「(a) の形を除いて」の但し書きを付けた。位置は handoff の「:98 のあたり」ではなく、日付の順に最後の改訂の項の次にした(この仕様の改訂の項は日付順に並んでいる)
- 項の中で、2026-11-07 改訂 (c) の「claude の座席が再試行で待っているあいだは、herdr が `working` と報告して `herdr agent get` の出力が変わり続けるので、見張りは stall の ALERT を出さない」が、issue #353 の週の上限の待ちでは当てはまらず stall が立ったことを書いた。これは issue の本文から読んだことで、見張りの状態ファイルは見ていない。通信断の待ちで同じ文が成り立つかは確かめていない。handoff の指示にない句だが、次の 516 行の注と同じ事実なので、仕様にも 1 か所残した
- resume の仕様: FR-3 の「待ち中」の項の直後に段落を足した。内容は、見張りが読む 3 つ目の形(入力欄のすぐ上の再試行の行の括弧の時刻が、今の手番の上限の文の `resets <時刻>` と同じ)、resume の結果は「待ち中」のまま何も打たないこと、理由が `waiting for the usage limit to reset: <上限の行>` になること。`internal/org/usage_limit_screen.go` と `internal/org/resume.go`(`resumeLook`、`resumeWorkingReason`)と `internal/cli/org_resume.go`(`resumeTargetText`)を読んで書いた。コードから読んだ、handoff にない点が 3 つある。(1) 理由の変更は 5 時間の上限の footer の待ち(`Continuing automatically at`)にも及ぶ(それまでは `the CLI is retrying or waiting`。計画の Rollout にも書いてある)。(2) herdr が working と報告する座席は、これまで pane を読まなかったが、今は座席ごとに 1 回読む。(3) 理由が `herdr reports working` のままのものが 3 つある: 上限で止まった画面(herdr が手番の終わりを取り逃がした)、通信断の再試行、pane の読みの失敗

### V4: Trigger に当たった行の持ち越しの注

verify の 6 行は、行頭の語で探して、番号が HEAD と同じであることを確かめた。形は 517 行の #322・#326 の注と、490・493・494 行の #329 の注に合わせた(Trigger のセルの末尾に日付つきの注、Related に計画の active のパス)。

| 行(先頭の語) | 当たった所 | 注に書いたこと |
|------|------|------|
| 464(Seven LOW findings of the cycle 1 self-review of org-seat-resume) | `resumeTargetText` の分岐、`resumeLook` を通る読み直し | L3〜L9 は計画の Scope の外で、どれも触れていない |
| 490(`send_screen_test.go` と `send_fallback_test.go` の大きさ) | `send_screen_test.go` に fixture の表の 1 行 | 878 → 885 行(`wc -l` を merge-base `1d05c9ac` と最後の Go の commit `29b5d5d6` で)。移さなかった理由は Scope |
| 493(Test gaps of feat/watch-usage-limited) | `watch_usage_limit.go` のコメント 2 つと新しいテスト 1 つ | (a)、`recur` の失敗の分岐、(d) の 2 つ目の acked の行は足していない |
| 494(`watch_usage_limit_test.go` の大きさ) | 1000 → 1035 行 | 古くなった「1000」を 1035 に直した。Trigger の注に加えて、Debt のセルの末尾にも 1 文(#329 の「grew it to 1000 lines」と同じ形) |
| 516(待ちのあいだ `herdr agent get` の出力が変わらないか) | issue #353 が stall の ALERT の報告に当たる | 下に詳しく |
| 517(#329 のテストの穴) | `UsageLimitScreen` の変更、`newOrgWatchCmd` の段落(`29b5d5d6`)、2 つのテストファイルへの追加 | (a)〜(d) は足していない。(d) の help の新しい句も固めていない |

516 行の注: issue #353 は、`CLAUDE_CODE_RETRY_WATCHDOG=1` の claude の leader が週の上限の解除を手番の中で再試行して待ち、見張りが `CONDITION: stall`・`REASON: leader_is_anomaly_subject` の件を上げ、`ralph org resume --dry-run` が `herdr reports working` と出した、という報告である。stall の件が上がるのは stall の判定が立ったとき(`agent get` の出力が前の周期と同じで、座席の最後の manifest の記録が `stall_minutes` より古い)なので、この再試行の形では出力が変わらなかったことになる。これは issue の本文から読んだことで、座席の `watch-status-<org_id>.json` の `agent_get` は見ていない、と注に書いた。払わなかった理由は、計画の Scope が「stall の判定が立ったあとの画面の読み方」で、判定が立つかどうかではないこと。この issue の形は手番の中の再試行の行で、この行の本来の対象(footer の `Continuing automatically at`)ではないので、footer の待ちでも出力が変わらないかは今も未確認、と書いた。

掃引で見つけた 3 行(verify と handoff は挙げていない):

- 267(Three test gaps of the org-director-attach PR): Trigger は「the inbox section (`受信箱の処理`) of the `/director` skill の変更」。この branch の S3 が `### 件の処理` の `usage_limited` の文を 4 つの写しで直した(自分の編集が当たる型)。`director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は変えていないので (1)〜(3) は開いたまま
- 362(`watch.go` が 1848 行、の行): Trigger は「the watch's escalation or its tests の次の変更」。#329 は同じ種類の変更(`watch_usage_limit.go` の文言)を「at its boundary」と書いた。この branch もコメント 2 つ(`usageLimitedWhy` のコメントなど)とテスト 1 つで、境目に当たる。この行の注が数えている 4 つのファイルの行数が動いたので、新しい数を書いた(`watch_usage_limit.go` 485、`usage_limit_screen.go` 317、`usage_limit_screen_test.go` 892、`watch_usage_limit_test.go` 1035)
- 483(Codex hook trust の check): Trigger は「`resume_screens.go` の画面の規則、または見張りの ALERT の条件の次の変更」。#312a と #329 が同じ型で当たったと注に書いている。この branch は `resume_screens.go` の claude の表のコメントを直し、見張りの `usage_limited` の条件に 3 つ目の読み方を足したので、また当たる。「Hooks need review」の条件は足していない

掃引で当たらなかった候補(Trigger のセルを読んで外した): 462(`ClassifyResumeScreen` の本体は変えていない。新しいコードは呼ぶだけで、`git diff` の hunk を読んだ)、457(「最初の利用上限の stop」の画面を保存する Trigger。issue の画面は再試行の形で stop ではなく、行の未確認の項目は変わらない)、455・456(`ralph status` と codex の通信断の待ち)、465(codex の footer の認識。`resume_screens.go` の変更は claude の表のコメントだけ)、491・492・495・507(`raiseUsageLimited`、`recordUsageLimitRecurrence`、`readUsageLimitScreen`、`usageLimitUnreadNote`、`usageLimitRecurNote`、`fitUsageLimitedSeats` はどれも変えていない)、467・88・358(`watch.go` は変えていない)、470(row 469 の RESOLVED のコメントがある。Trigger のセルは取り消し線がないが、docs/director-notice-handling で払い済み)、213(`newOrgWatchCmd` は `newOrgRuntimeAt` の doc が挙げる呼び出し元として出る名前で、help の段落の編集とは別)、274・370・385・386・427・509・521(director skill の別の節かテストファイルの Trigger。S3 は `### 件の処理` の `usage_limited` の文だけを変えた)、479・485・488・489(fixture という語はあるが、実機の画面を取る Trigger)、118・182(quality-gates の編集と `/org` skill の表。どちらも触れていない)、518・524(Trigger の `agent-messaging.md` は Ctrl+C の手順で、この branch が直したのは `usage_limited` の段落)、179・387(spec の別の項目)。

### V5: 800 行を超えた 2 つのテストファイル

新しい行を最後(540 行目)に足した。5 つのセル。`usage_limit_screen_test.go` は 606 → 892 行(+286)、`resume_test.go` は 800 → 869 行(+69)(`wc -l` を merge-base `1d05c9ac` と `29b5d5d6` で)。分け方の案は 2 つ: `usage_limit_screen_test.go` の #353 の定数・helper・2 つのテスト(`29b5d5d6` で 526〜768 行、約 240 行)を `internal/org/usage_limit_screen_retry_test.go` に移すと約 650 行になる(`usageLimitRetryingLine` は `TestUsageLimitScreen_EveryFixture` が呼ぶので上に残す)。`resume_test.go` の `TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit`(370〜436 行)を `internal/org/resume_usage_limit_test.go` に移すと約 802 行で、merge-base の 800 行の近くに戻る(800 を割るには足りない、と行に書いた)。行番号は Debt のセルに 1 回だけ、測った commit(`29b5d5d6`)を添えて書いた。Trigger と `resume_test.go` の側は、後の commit で行がずれても読めるように、関数と定数の名前で書いた(register の「関数名で書く」の規則)。計画への参照は active のパス。この行の直しは tech-debt の 2 つ目の commit(下の commit の一覧)にある。

### V6: 計画の箱

AC1〜AC6 を `[x]` にした。AC1〜AC5 は test report が PASS で、AC6 は verify の報告が Met で、この pass でも `check-sync.sh` と `check-skill-sync.sh` が通った。`./scripts/plan-visual.sh digest` は編集の前後とも `70b063e49ad8` で、計画の `- Approved:` の行と同じ。`git diff --numstat` は 8/7 行で、消えた 7 行はどれも `- [ ]` の行(AC の 6 つと Test artifact)。足した 8 行のうち 7 行は `- [x]` に変わった行、1 行は `Sync-docs: pass` の行(Progress の中)。

handoff は `Test artifact created` に触れていなかった。test report は commit 済み(`f246da8f`、pass)で、箱だけが `[ ]` のまま残っていたので `[x]` にして、report と要点を書いた。orchestrator が別の言い方で付け直す場合は、この行を差し替えてよい。

## `/org` skill(handoff の最後の項目)

変えなかった。理由は 3 つ。

1. `.claude/skills/org/SKILL.md` の `usage_limited` の項(686〜689 行付近)と watch の行(383 行)は、「`ralph org resume` が利用上限と読む文で止まっていれば、stall を立てない」と書く。これは十分条件の書き方で、3 つ目の読み方を足しても偽にならない(#329 の 5 時間の footer の待ちも、同じ項が書いていないまま)。verify の INFO も「偽ではない」とした
2. この形の契約は、座席と leader が読む `agent-messaging.md`(この branch で更新済み、2 つの写し)と、director が件を扱うときに読む `/director` skill の `### 件の処理`(4 つの写し、S3 で更新済み)にある。`/org` skill は動詞の表と人の受信箱の手順で、`usage_limited` の読み方で手順が分かれることはない
3. 直すには 4 つの写し(`sync-skills.sh` を 2 回)と、`/org` skill の表に触れる tech-debt の行(182 行の (a) など)への注が要る。偽でない文のためには釣り合わない

穴は残る: `/org` skill の項は 5 時間の footer の待ちと手番の中の再試行の待ちを書かない。register に新しい行は足していない(#329 の sync-docs も同じ判断で、verify の INFO に残っている)。次に `/org` skill の `usage_limited` の項を直すときに 2 つの待ちを足せばよい。

## そのほかの確認(変更なし)

- `AGENTS.md`: `internal/org/` の行の watch と resume の一節は「止まっているか、解除を待っている」と書き、読み方の数を言わない。resume が各座席の pane を読むことも既に書いてある。変更なし
- `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し: この branch の `5ddd4ef9` で更新済み。`cmp` で同じ(check-sync が IDENTICAL)。コードと突き合わせた verify の確認(入力欄のすぐ上の再試行の行、今の手番、`⎿` か `⏺`、同じ時刻、時刻のない再試行は読まない)を信用し、この pass では再確認していない
- `/director` skill の 4 つの写し: `check-skill-sync.sh` が 14 skill で lock-step。`templates/base/` の 2 つとの差は `check-sync.sh` が DRIFTED 0
- `docs/quality/quality-gates.md`(root と template の 2 つ): watch の行は `usage_limited` を名前で挙げるだけで、読み方は書かない。変更なし
- `docs/recipes/`(2 つ)と README の 302 行の「`api_wait_minutes` ほど待ってから手番を終える」: verify の INFO の 3 つ目の件。この PR の前からの文で、再試行が上限の解除より先に尽きたときの画面は確かめていない。README の 302 行には、issue #353 に当たる 1 文だけ足した(上の Changes made)。recipe の 2 つは変えていない
- 役割 prompt(`internal/org/prompts/`): 利用上限の言及は resume の続行の文を説明する 3 か所だけで、見張りの読み方には触れない。変更なし
- `docs/architecture/repo-map.md`、`.claude/settings.json`、hooks、language pack: この branch は触れていない。新しい script もない

## 確かめたこと

| コマンド | 結果 |
|------|------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。README、仕様、tech-debt、計画の編集のあと |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step。skill は変えていない |
| `./scripts/plan-visual.sh digest <plan>` | `70b063e49ad8`(`- Approved:` の行と同じ) |
| tech-debt の plan 参照の検査(`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ `grep`) | 欠けた参照なし |
| `git show HEAD:docs/tech-debt/README.md` との照合(python) | 変わった行は 267、362、464、483、490、493、494、516、517 と、末尾の新しい行だけ。どの行も 5 つのセル、バッククォートは各セルで偶数、取り消し線は偶数、足した字に `file:line` の参照はない。元の各セルが新しいセルの先頭に残っている |
| 編集した 5 つのファイルの U+FFFD | 0 |

回していないもの: `./scripts/run-test.sh`(依頼の指示)、`./scripts/run-verify.sh` の全体、`gofmt`・`go vet`(Go は変えていない)。

## 残したもの

- `/org` skill の `usage_limited` の項(上のとおり。偽ではない)
- README の 302 行と recipe の 2 つの「`api_wait_minutes` ほど待ってから手番を終える」: 再試行が上限の解除より先に尽きたときの挙動は未確認(verify の INFO)
- tech-debt の 515 行(#353 の穴、reviewer が足した行)と 514 行(この branch の S3 の注)は、この pass で変えていない

## Insight event

`./scripts/insights-append.sh --slug watch-usage-limit-retry --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` に 1 行を足した(cycle 1、ts は 2026-11-20T09:26:34Z)。exit 0。`git status --short` は、この行を足した既存の jsonl が `M` で、ほかに新しいファイルはなかった(slug に日付を付けていない)。
