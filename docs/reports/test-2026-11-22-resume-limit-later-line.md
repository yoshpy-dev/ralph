# Test report: resume-limit-later-line

## Run 2 (cycle 2)

- Date: 2026-11-22(run-test.sh は 18:21:37〜18:41:09 JST。パイプラインの cycle 2、`cycle-count.json` は `cycle: 2`)
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373)
- Tester: tester サブエージェント(`/test`)。今の判定は下の `## Verdict` で、Run 1 は見出しを 1 段下げて `---` の下に残した
- Scope: Run 1(HEAD `836cb19f`)のあとにコードを変えた 2 コミット。`1042ccae` は prompt のすぐ下の返答の頭が、引用した結果に印を付けないようにした。`e9017d7f` は項目の頭を 1 桁目の `⏺` の行に限り、`resumeScreenBlocks(lines, driver)` にし、send のコメントを直した。テストの行を 6 つ足した `7a7afdca`(下の「足したテスト」)の上で、`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を worktree のルートから 1 回だけ回した(`Requested scope: full`、`Language scope: full`、選ばれた pack は golang だけ)。打つ前に `ps -axo pid,command | grep -E 'run-test.sh|run-verify.sh'` で、ほかの run がないことを確かめた。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 175 s ずつ待った。`nohup` は使っていない。ほかに依頼の `-race` の run、足した行を確かめる変異 4 個、send の probe 1 本、狭い coverage の run を回した。変異と `-race` は run-test.sh の前に、coverage と打ち直しはあとに回し、run-test.sh とは重ねていない
- Evidence: `docs/evidence/test-2026-11-22-resume-limit-later-line-run2.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力の後ろに、1 分ごとの load、`(cached)` だった 5 パッケージの打ち直し、`-race` の run、`-v` の run の要約、変異ごとの diff と落ちたテスト、coverage の要約を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-092137.log`(同じく追跡外)。Run 1 の本文が引く Run 1 の log には足していない

### Test execution (cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(HEAD `7a7afdca`) | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。Run 1 と同じ 2 つ) | 18:21:37〜18:41:09 JST、1,172 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合 | 約 9 分(18:21:37〜18:30 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。5 パッケージは `(cached)`(下の注) | 約 10.5 分(18:30 ごろ〜18:41:09)、下の表 |
| `go test -count=1`(`(cached)` だった 5 パッケージ) | 5 | 5 | 0 | — | 壁時計 16 s |
| `go test -count=1 -race ./internal/org/ -run 'Resume\|UsageLimit\|SendScreen'` | — | ok | 0 | — | 34.436 s(壁時計 50 s)。`DATA RACE` は 0 行 |
| `go test -count=1 -v ./internal/org/ -run 'Resume\|UsageLimit\|SendScreen'` | トップレベル 101、`--- PASS` 696 行 | 696 | 0 | 0 | 32.123 s |

| パッケージ | run-test.sh | `-count=1` の打ち直し |
| --- | --- | --- |
| `cmd/ralph` | ok 0.728 s | — |
| `internal/cli` | ok 607.935 s | — |
| `internal/config` | ok 9.997 s | — |
| `internal/gitroot` | ok (cached) | ok 1.655 s |
| `internal/insights` | ok 1.168 s | — |
| `internal/org` | ok 623.257 s | — |
| `internal/org/driver` | ok 1.764 s | — |
| `internal/org/protocol` | ok (cached) | ok 0.687 s |
| `internal/scaffold` | ok (cached) | ok 1.746 s |
| `internal/toolchain` | ok (cached) | ok 15.939 s |
| `internal/upgrade` | ok (cached) | ok 2.177 s |

cycle 2 のコミットは `internal/org` と文書しか変えていないので、残りの 5 パッケージは go が前の結果を使った。`-count=1` で打ち直し、どれも ok だった。`internal/org` と `internal/cli` は golang pack の 1 パッケージの上限 30 分の 35% ほどで終わった。

`-v` の run で足した行のあるテストの subtest は、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` が 37 行、`TestClassifyResumeScreen_Edges` が 69 行、`TestResumeScreenBlocks_ItemHeadUnderAPrompt` が 5 行で、SKIP は 0 だった。

run の間の 1 分 load は 2.58〜5.75(18:22:37 から 1 分ごとに 20 回測った)。打つ前に CPU 50% を超えるプロセスは Orca の helper 1 つだけで、Run 1 に重なっていた空回りの zsh はなかった。落ちたテストはなく、打ち直したのは `(cached)` の 5 パッケージだけである。shell の段の `PASS` の行は Run 1 と同じ 5,390 だった。

### 足したテスト(`7a7afdca`)

`internal/org/resume_screens_test.go` に 6 行を足した。既存の行の期待は 1 つも変えていない。

| テスト | 行 | 画面(入力欄より上) | 期待 | 埋めた穴 |
| --- | --- | --- | --- | --- |
| `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` | a reply head right under a prompt that quotes one ⎿ line with a limit | `❯ what did the seat show?`、`⏺ The seat showed:`、`  ⎿  You've hit your weekly limit · resets …` | unknown(no stop)、`UsageLimitScreen` は false | verify cycle 2 の Coverage gaps の 2 つ目。self-review Run 2 の P6 |
| 同上 | 同じ形で、空行と状態行が下にある | 上の 3 行、空行、`✻ Worked for 2s · done 3:40 PM` | 同上 | 同上 |
| `TestClassifyResumeScreen_Edges` | a reply head right under a prompt that quotes one ⎿ API error line, above a status line, is unknown | `❯ what did the seat show?`、`⏺ The seat showed:`、`  ⎿  API Error: Connection error.`、空行、状態行 | unknown(no stop) | 同じく P10 |
| 同上 | a waiting sentence in an item at the first column right under a prompt is waiting | `❯ go on` のすぐ下に 1 桁目の `⏺ You've hit your Opus limit · resets 3:45pm` と `  Continuing shortly · Esc to cancel` | waiting | verify cycle 2 の Coverage gaps の 3 つ目(手順 1 の切れ目は (a)・(b) にも効く) |
| 同上 | a waiting sentence a prompt quotes in indented lines does not count | 上の 2 行を字下げして prompt の続きにした形 | unknown(prompt_last) | 上の行の対照 |
| `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` | codex limit on a later ⎿ line of a result block right under a prompt | `› go on` のすぐ下に `⎿  Read notes.md` と `⎿  You've hit your usage limit. Try again at 3:05 PM.` | unknown(no stop)、`UsageLimitScreen` は false | Run 1 の Test gaps の 1(M4b が残った) |

P6 と P10 の 3 行は、`1042ccae` で stopped から unknown に変わった形である。`1042ccae` の前は、返答の頭が prompt のまとまりに入り、その下の 1 行の `⎿` が手順 4 (c) で stop と読まれた。waiting の行も `1042ccae` で答えが変わった形で、前は prompt のまとまりに入って `prompt_last` だった。画面モデルの手順 1 は、1 桁目の項目の glyph で始まる行が始めるまとまりを prompt のまとまりではないとし、(a)・(b) が飛ばすのは prompt のまとまりだけなので、waiting はコードと doc の両方から出る答えである。prompt のすぐ下に待ちの文が出る画面を実機で見たことはない(Test gaps の 2)。

### 変異 (cycle 2)

`7a7afdca` の worktree はそのままにし、`internal/org/resume_screens.go` の写しを `$TMPDIR` の下に作って perl で 1 か所を替え(置き換える文字列がファイルに 1 回だけあることを確かめてから替えた)、`go test -overlay` で差し替えた。worktree の `resume_screens.go` は HEAD と `cmp` で同じまま。`-run` は `'Resume|UsageLimit|SendScreen'`。対照(変異なし)は rc 0、どの変異も `build failed` は 0 件。

| # | 変異 | 結果 | 落ちた subtest |
| --- | --- | --- | --- |
| MR | 手順 1 の項目の頭の切れ目を外す(条件の頭に `false &&`。`1042ccae` の前の切り方) | 落ちた(9) | Edges の 3 行(1 桁目の API Error、足した P10、足した waiting)、表の 4 行(cross-review の `⎿` 2 行の 2 形、足した P6 の 2 形)、`TestResumeScreenBlocks_ItemHeadUnderAPrompt` の 2 行 |
| MT | 項目の頭を字下げを落とした行 `t` で比べる(`e9017d7f` の前) | 落ちた(7) | Edges の 3 行(足した waiting の対照、prompt が字下げした `⏺ API Error` を写す 2 形)、表の 2 行(prompt が字下げした上限を写す 2 形)、切り方のテストの 2 行 |
| MV | 頭は prompt のまとまりに残し、その下の `⎿` の行はまとまりを始めるが印を付けない(verify cycle 2 の Coverage gaps が挙げた形) | 落ちた(7) | Edges の 3 行(1 桁目の API Error、足した P10、足した waiting)、表の 2 行(足した P6 の 2 形)、切り方のテストの 2 行。cross-review の `⎿` 2 行の 2 形は verify の読みどおり通った |
| M4b | codex の `StopGlyphs` と `ResultGlyphs` の両方に `⎿` を足す(Run 1 で残った変異) | 落ちた(1) | 足した codex の行だけ(stopped、`UsageLimitScreen` は true) |

4 個とも落ちた。足した 6 行はどれも 1 個以上の変異で落ちる(P6 の 2 行と P10 と waiting は MR と MV、waiting の対照は MT、codex の行は M4b)。足す前の MV を落としていたのは、Edges の 1 桁目の API Error の行と切り方のテストだけで、P6・P10 の形を分類の答えで固定する行はなかった。

### send の probe

`e9017d7f` は send のコメントに「返答の頭と状態行が prompt のすぐ下に空行なしで来ると、send は unknown と読み、何も打たない」と書いた。この形を固定するテストはないので、使い捨てのテストを `-overlay` で足して `ClassifySendScreen("claude", …)` を確かめた。

| 入力欄より上の画面 | 結果 |
| --- | --- |
| `❯ say hi`、`⏺ Hi.`、`✻ Cooked for 3s · done 6:09 PM`(空行なし) | unknown(`the last block above the input box is a user prompt`) |
| `❯ say hi`、`⏺ Hi.`、空行、状態行 | idle(`the turn ended`) |
| `❯ say hi`、空行、`⏺ Hi.`、空行、状態行 | idle(`the turn ended`) |

コメントのとおりだった。send のコードはこの branch で変わっていないので、行は足していない(Test gaps の 3)。

### Coverage (cycle 2)

上の `-v` と同じ `-run` で `-coverprofile` を取った(`7a7afdca`)。

- Statement: `resume_screens.go` の 157 ブロックのうち count 0 は 0。`resumeScreenBlocks`、`ClassifyResumeScreen`、`findResumeStop`、`findResumeLimitResult`、`resumeResultUnderToolOrNotice`、`resumeItemHead`、`findResumeSentence` はどれも 100.0%。cycle 2 で引数を変えた呼び出し(`usage_limit_screen.go:175`、`send_screen.go:611`)のブロックも count 1 以上
- Branch: Go の coverage は分岐を数えない。手順 1 の切れ目の分岐は上の変異で見た
- Notes: パッケージ全体の 18.5% は絞り込みの run の数字で、意味はない

### Failure analysis (cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks (cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 1 の ACTION_REQUIRED 1: prompt のすぐ下の返答の頭が引用した `⎿` 2 行に印が付き、stopped と読まれた | 直った | 表の cross-review の 2 行が unknown を固定する。MR で 2 行とも落ちる |
| self-review Run 2 の M1: prompt が字下げして写した `⏺` の行を項目の頭と読み、中断した画面を stopped と読んだ | 直った | Edges、表、切り方のテストの 6 行と足した対照が、unknown と prompt のまとまりを固定する。MT で 7 行が落ちる |
| P6・P10: 返答の頭の下の 1 行の `⎿` を stop と読んだ(base から) | 直った | 足した 3 行。MR と MV で落ちる |
| issue #373 の画面を `ralph org resume` が skipped にした | 直ったまま | fixture の 5 行と CLI の `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice` が run-test.sh で通った |
| send の画面の読み(`ClassifySendScreen`) | 守られている | `TestClassifySendScreen_*` が狭い run と run-test.sh で通った。cycle 2 の send の変更は引数とコメントだけ |
| 既存の resume・usage limit・send・watch のテスト | 守られている | run-test.sh の `internal/org`(623 s)と `internal/cli`(608 s)が全件 PASS |

### Test gaps (cycle 2)

1. Run 1 の Test gaps の 1(codex の M4b)は、足した codex の行で閉じた
2. (LOW)waiting の行は、コードと画面モデルの doc から出る答えを固定しただけで、prompt のすぐ下に待ちの文が出る画面を実機で見たことはない。enter-only の文((a))で同じ形の行は足していない。(a) と (b) は同じ `findResumeSentence` に同じまとまりを渡すので、切れ目の効き方は 1 行で固定できると判断した
3. (LOW)send の prompt のまとまりが項目の頭で切れないこと(`e9017d7f` のコメント、check 3)は、上の probe で確かめただけでテストの行がない。足すなら `TestClassifySendScreen_*` に、空行のない `❯ say hi`、`⏺ Hi.`、状態行を unknown とする 1 行
4. 仕様と rule の文を固定するテストはない(Run 1 の Test gaps の 3、verify cycle 2 の Coverage gaps の 4 つ目)。`/test` の担当外
5. Run 1 の Test gaps の 2(空行のあとにまとめた行で始まる引用は stopped、計画どおりの known gap)と 4(fixture は実機の pane を `herdr pane read` で読んだものではない)は変わらない。本物の herdr のサーバーには触れていない

## Verdict

- Pass: run-test.sh(full)rc 0、1,172 s。shell 46 本と Go 11 パッケージが全部通り、SKIP は条件つきの 2 つだけ。`(cached)` の 5 パッケージは `-count=1` で打ち直して ok。依頼の `-race` の run は ok で `DATA RACE` は 0。足した 6 行は `7a7afdca` で通り、変異 4 個(Run 1 で残った M4b を含む)はどれも落ちた
- Fail: 0
- Blocked: 0

---

## Run 1 (cycle 1、履歴)

- Date: 2026-11-22(run-test.sh は 16:27:14〜16:49:43 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `836cb19f`(worktree は clean、base は origin/main `b263b1f3`)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を worktree のルートから 1 回だけ回した(`Requested scope: full`、`Language scope: full`、選ばれた pack は golang だけ)。打つ前に `ps -axo pid,command | grep -E 'run-test.sh|run-verify.sh'` を見ると、別の worktree(`director-wait-types-note`)の run-verify.sh が 16:03:45 から走っていた。それが 16:27:08 に終わり、何も返さなくなってから打った。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。そのほかに、足したテストを `-count=1 -v` で回し直し、変異 7 個、codex の probe 1 本、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-22-resume-limit-later-line.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力の後ろに、1 分ごとの load、`-v` の run の要約、変異と落ちたテスト、probe、coverage の要約、`~/.codex/config.toml` と herdr のプロセスの前後を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-072715.log`(同じく追跡外)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 16:27:14〜16:49:43 JST、1,349 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0。stub の古い git の行は通る) | 約 11 分(16:27:15〜16:38 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 11.5 分(16:38 ごろ〜16:49:43)、下の表 |
| `go test -count=1 -v ./internal/org/ -run '<下の注>'` | `--- PASS` 602 行 | 602 | 0 | 0 | 0.690 s |
| `go test -count=1 -v ./internal/cli/ -run 'TestOrgResume_'` | 18 | 18 | 0 | 0 | 17.029 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.775 s |
| `internal/cli` | ok | 757.774 s |
| `internal/config` | ok | 14.529 s |
| `internal/gitroot` | ok | 3.054 s |
| `internal/insights` | ok | 2.167 s |
| `internal/org` | ok | 772.336 s |
| `internal/org/driver` | ok | 3.858 s |
| `internal/org/protocol` | ok | 3.572 s |
| `internal/scaffold` | ok | 4.428 s |
| `internal/toolchain` | ok | 21.580 s |
| `internal/upgrade` | ok | 4.614 s |

`(cached)` のパッケージは 1 つもなかったので、`-count=1` の打ち直しは要らなかった。`internal/cli` と `internal/org` は golang pack の 1 パッケージの上限 30 分の 43% ほどで終わった。

`-v` の run の `-run` は `TestClassifyResumeScreen|TestResumeResultGlyphs|TestResumeScreenSentences|TestResumeInputHolds|TestUsageLimit|TestClassifySendScreen|TestSendScreen|TestSendInputHolds|TestSendProbe`。依頼で名前の挙がったテストはどれも PASS で、SKIP は 0 だった。

- `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine`: subtest 29 行(stopped 10、unknown 19)
- `TestResumeResultGlyphs_AreStopGlyphs`: PASS(claude の `⎿` で中身を確かめた。claude に `ResultGlyphs` がないときに止める先頭の `t.Fatal` には当たっていない)
- `TestClassifyResumeScreen_Fixtures`: 新しい fixture の 5 行
- `TestClassifyResumeScreen_Edges`: 直した 1 行と足した 1 行
- `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`、`TestUsageLimitScreen_RetryWaitShapes` の Monitor の行(「a limit in the turn above a turn-end status line, with no retry」)
- `TestUsageLimitScreen_EveryFixture` と `TestClassifySendScreen_EveryFixture` の新しい fixture の各 5 行
- `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice`: PASS(0.44 s)

run の間の 1 分 load は 9.28〜18.32(16:28:15 から 1 分ごとに 23 回測った)。この run には、別のレーンの load の試験が残した nice つきの空回りの zsh が 6 つ(pid 27202〜27207、親は pid 1、15:21 ごろから各 90% 前後の CPU)が重なっていた。私のものではないので止めていない。落ちたテストはなく、打ち直しもしていない。私の変異の run(M1〜M4b の絞り込み)は run-test.sh の前、別の worktree の run と重なる時間に回した。M4b のパッケージ全体の run は run-test.sh が終わってから回した。

shell の段の `PASS` の行は 5,390 で、org-start-issue の run(`docs/reports/test-2026-11-22-org-start-issue.md`)と同じ数だった。この branch は shell のテストを足していない。

run と変異の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じで、書かれていない。herdr のプロセス(`herdr server` の pid 59855 と、ほかの 2 つ)も前後で同じで、本物の herdr のサーバーには触れていない。CLI のテストは `stubResumeHerdr` の fake の herdr を使う。

#### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestClassifyResumeScreen_Fixtures` の 5 行(fixture 5 件)、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` の stopped の 10 行(issue の 3 画面をそのまま、`⏺ Bash(` の出力のあと、折り返した Monitor の呼び出し、`⏺ Ran 2 shell commands`、U+00A0、1 行だけの候補、上限の文の折り返し、P2 の形)、CLI の `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice` | PASS。M1 で 20 行、M3a で 14 行が落ちる。CLI のテストは M1 で issue と同じ `skipped  org-a/implementer  claude  unknown screen: no known stop sentence above the input box` を 2 行出して落ちる |
| AC2 | `TestUsageLimitScreen_EveryFixture` の 5 行(`line` と `has` のリセットの時刻)、表のテストの `UsageLimitScreen` の列、`TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop` | PASS。M1 と M3a で落ちる |
| AC3 | 表のテストの unknown の 19 行(claude 16、codex 3) | PASS。M2a で 3 行(空行のあとの `⎿` 2 行の引用の 2 形、頭のない普通の行の下の上限)が落ちる。codex の 3 行は M4b を落とせない(Test gaps の 1) |
| AC4 | run-test.sh の `internal/org`(772 s)と `internal/cli`(758 s)の全件。直した 2 件(Edges の「only the first ⎿ line under a prompt starts a block, the next ones join it」と RetryWaitShapes の Monitor の行) | PASS。Edges で足した行(プロンプトのすぐ下の結果のまとまりの上限)は M2b で落ちる |
| AC5 | 仕様と rule の文を読むテストはない(下の Test gaps の 3)。rule の 2 写しの同期は local verifier の中の `tests/test-check-sync.sh` がスクリプトの側を確かめ、中身の `check-sync.sh` は `/verify` の静的検証が通している | 文の中身は `/verify` の担当 |

計画の Test plan の Integration tests(`ralph org resume --dry-run` の fake の画面で issue の画面が続ける対象になる)は、上の CLI のテストが満たす。Regression tests の見張りの usage_limited は、run-test.sh の `internal/org` の中の `watch_usage_limit_test.go` と `watch_usage_limit_race_test.go` が通った。

#### 変異

HEAD の `git archive` の写しをセッションの scratchpad の下に作り、`internal/org/resume_screens.go` の 1 か所を perl で替えた(置き換える文字列がファイルに 1 回だけあることを確かめてから替えた)。毎回元に戻して `cmp` で HEAD と同じことを確かめた。どの変異も `build failed` は 0 件。対照(変異なし)は上の `-v` と同じ `-run` で rc 0 だった。

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| M1 | (c2) を外す(`ClassifyResumeScreen` の `findResumeLimitResult` の呼び出しを消す) | 落ちた | fixture の 5 行、Edges の 1 行、表のテストの stopped の 8 行、`TestUsageLimitScreen_EveryFixture` の 5 行、RetryWaitShapes の Monitor の行。CLI の dry-run のテスト |
| M2a | `resultUnderPrompt` を外す(`return block.resultUnderPrompt` を `return true` に。`d470a77a` の前の許し方) | 落ちた | 表のテストの 3 行(空行のあとの `⎿` 2 行の引用の状態行あり・なし、頭のない普通の行の下の上限) |
| M2b | 頭のない結果のまとまりを許さない(同じ行を `return false` に) | 落ちた | Edges の「a usage limit on the last ⎿ line of a result block under a prompt is stopped」 |
| M3a | つないだ文でなく 1 行だけを使う(`text := line`。照合も `Line` も 1 行になる) | 落ちた | fixture の 5 行、表のテストの 3 行(issue の画面 1・2 の文字どおりの形、上限の文の折り返し)、`TestUsageLimitScreen_EveryFixture` の 5 行、RetryWaitShapes の Monitor の行 |
| M3b | 照合だけを 1 行にする(`resumeAfterGlyph(line, …)`。`Line` はつないだ文のまま) | 落ちた | 表のテストの「a limit sentence the pane's width wrapped onto the next line, under a tool call」 |
| M4a | codex に `ResultGlyphs: []string{"⎿"}` を足す | 落ちた | `TestResumeResultGlyphs_AreStopGlyphs` だけ(「result glyph "⎿" is not one of StopGlyphs」)。codex の挙動の行では落ちない |
| M4b | codex の `StopGlyphs` と `ResultGlyphs` の両方に `⎿` を足す(M4a で glyph の包含を保った形) | 残った | 絞り込みの run は rc 0。run-test.sh のあとに回した `internal/org` 全体(`-count=1 -timeout 30m`、673.486 s)と CLI の `-run 'TestOrgResume\|TestOrgStatus\|TestOrgWatch'`(30.001 s)も rc 0 |

M1、M2a、M2b、M3a、M3b、M4a の 6 個は落ち、M4b の 1 個が残った。M4b は依頼の 4 つ目(codex に `⎿` を結果の記号として与える)を、glyph の包含の確かめに掛からない形にしたもの。

#### codex の probe

M4b が残る理由を確かめるため、使い捨てのテストを `-overlay` で写しに足した。codex の画面で、プロンプトの行 `› go on` のすぐ下に `⎿  Read notes.md` と `⎿  You've hit your usage limit. Try again at 3:05 PM.` が並ぶ形にした。

| 写し | `ClassifyResumeScreen("codex", …)` | `UsageLimitScreen` |
| --- | --- | --- |
| HEAD | unknown(`no known stop sentence above the input box`) | false |
| HEAD + M4b | stopped(`Line` は `⎿  You've hit your usage limit. Try again at 3:05 PM.`) | true |

codex の 3 行は、どれもプロンプトの行も項目の頭も持たない。そのため、codex に結果の記号を足しても `resumeResultUnderToolOrNotice` は `resultUnderPrompt` の false を返し、答えは変わらない。プロンプトのすぐ下の結果のまとまりを持つ 1 行を足せば M4b を落とせる。

#### Coverage

上の `-v` と同じ `-run` で `-coverprofile` を取った(HEAD の写しで)。

- Statement: `resume_screens.go` の 155 ブロックのうち count 0 は 0。足した関数 `findResumeLimitResult`、`resumeResultUnderToolOrNotice`、`resumeItemHead`、`resumeToolOrNoticeHeadForm`、`resumeStartsWithAny`、`resumeAfterGlyph` と、変えた `ClassifyResumeScreen`、`resumeScreenBlocks`、`findResumeStop` はどれも 100.0%
- Branch: Go の coverage は分岐を数えない。上の変異で分岐ごとに見た
- Function: 上のとおり
- Notes: パッケージ全体の数字は絞り込みの run なので意味がない(4.3%)。M4b の残りが示すように、行が覆われていても codex の表の欄の中身は固定されていない

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #373: 上限の文が候補の 2 行目以降の `⎿` の行にある claude の画面を `ralph org resume` が `skipped … unknown screen` にした | 直った | fixture の 5 行と CLI の dry-run のテストが `would_resume` を固定する。M1(直す前と同じ読み方)で CLI のテストが issue と同じ skipped の 2 行を出して落ちる |
| #353 の Monitor の知らせの下の上限(retry の文なし)を見張りが読まなかった | 直った。3 つ目の読み方は前のまま | `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop` が、(c2) の stopped、retry の文がないので `usageLimitRetryingTurn` が false、`UsageLimitScreen` が #353 の行を返すことを固定する |
| 返信が座席の画面を `⎿` の形で引用しただけの画面を stopped と読まない(計画の助言と self-review の M1) | 守られている | 表のテストの unknown の 4 行。M2a で 2 行が落ちる |
| send の画面の読み(`ClassifySendScreen`)が新しい fixture で変わらない | 守られている | `TestClassifySendScreen_EveryFixture` の 5 行が idle と turn_ended を固定する |
| 既存の resume・usage limit・send・watch のテスト | 守られている | run-test.sh の `internal/org` と `internal/cli` が全件 PASS |

### Test gaps

1. (LOW)codex の 3 行は「(c2) は claude だけ」を挙動で固定していない。今の固定は `TestResumeResultGlyphs_AreStopGlyphs` の glyph の包含だけで、包含を保って codex に `⎿` を足す M4b は、`internal/org` 全体と CLI の resume・status・watch のテストでも残った。codex の行にはプロンプトの行も項目の頭もないので、codex の表に結果の記号を足しても答えが変わらない。上の probe の形(`› go on` のすぐ下の `⎿` 2 行、最後が上限)を unknown と固定する 1 行を足せば落ちる。codex は実機で `⎿` を出したことがないので、今の害はない
2. (既知、計画どおり)空行のあとにまとめた行(`Ran 1 shell command`)で始まる引用は、issue の画面 3 の実機の形と区別できず stopped になる(self-review の M1、P2 の形)。表のテストが stopped で固定し、コメントで known gap と書いている。実機で見ていないまとめた行(`Searched for 1 pattern`)の下の上限が unknown のまま、ツールの出力の 1 行目が上限の文で始まり手番が黙って終わった座席を stopped と読む、の 2 つも計画の Risks のとおり残る
3. 仕様 FR-2 の 4 と rule の usage_limited の段落の文を読むテストはない。2 写しの同一は `check-sync.sh`、中身は `/verify` が読んで確かめた。`ralph org resume --help` と `/director` skill は変えていない
4. fixture は issue の文字に、手元の Claude Code 2.1.316 の画面の空行と入力欄と footer を組み合わせたもので、上限に当たった実機の pane を `herdr pane read` で読んだものではない。本物の herdr のサーバーには触れない約束なので、実機の画面での確かめはしていない

### Verdict

- Pass: run-test.sh(full)rc 0、1,349 s。shell 46 本と Go 11 パッケージが全部通り、SKIP は条件つきの 2 つだけ。足したテストと直したテストは `-count=1 -v` で SKIP なしに通った。依頼の 4 つの変異(とその変形)は 6 個が落ち、1 個(M4b)が残った
- Fail: 0
- Blocked: 0
