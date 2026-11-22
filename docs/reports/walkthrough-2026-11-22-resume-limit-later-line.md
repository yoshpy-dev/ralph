# Walkthrough: resume-limit-later-line

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-resume-limit-later-line.md
- Issue: #373(fixture は issue に貼られた画面の文字から作った。実機の `herdr pane read` ではない。9 節)
- Base: main 38932c13

差分は 24 ファイル、+2,055 行、-69 行(この walkthrough の commit の前の数)。内訳は、記録(計画、self-review・verify・test・sync-docs・cross-review triage の report、insight)が 7 ファイルの +1,036 行。Go が 7 ファイルの +919 行、-55 行で、うちテストが 4 ファイルの +629 行、-6 行、テスト以外が 3 ファイルの +290 行、-49 行。fixture が 5 ファイルの +63 行。文書(rule の 2 写し、仕様 2 本、tech-debt の台帳)が 5 ファイルの +37 行、-14 行。規則の本体は `internal/org/resume_screens.go` の +265 行、-35 行で、`send_screen.go` と `usage_limit_screen.go` はコメントと呼び出しの引数の直しだけ。下の順に読むと早い。

## 1. 何が報告されたか

2026-11-22 の 04:05 JST ごろ、座席の Claude Code 2.1.316 が 5 時間の上限(`You've hit your 5-hour limit`)に当たり、3 つの org の 5 席(leader 3、reviewer 1、implementer 1)が止まった。04:40 に上限が解けたあと `ralph org resume --dry-run` を打つと、5 席とも `skipped … unknown screen: no known stop sentence above the input box` になった。resume は使えず、director が leader に DECISION で続行を伝えた。ralph は 8.2.1。

上限の文は `⎿` の行にあるが、候補のまとまりの先頭ではなかった。issue の画面は 3 つ。ツールの実行のあと(`Ran 1 shell command`)、背景の知らせのあと(`⏺ Background command "…" failed with exit code 1`。上限の文が pane の幅で折り返されている)、prompt のあと(`Read 2 files`)。#329 が足した `You've hit your 5-hour limit` の文は表にある。issue が求めたのは、最後の `⎿` の行が上限の文で始まるときも stopped と読むこと、prompt のまとまりでもその下の `⎿` の行を読むこと、見張りの `usage_limited` も同じ見分けにそろえること。

## 2. 調べて分かったこと

- 手順 4 の (c) は、状態の行を 1 つ飛ばした候補のまとまりの先頭だけを見る。3 つの画面は上限の文がまとまりの 2 行目以降にあるので、(c) に当たらない
- prompt のまとまりを読む案は要らない。prompt の行のあとの最初の `⎿` の行で別のまとまりが始まる(手順 1)ので、`⎿` の行は prompt のまとまりに入らない。計画の Non-goals に入れた
- 計画の調べ(scratchpad の probe、commit していない)では、今のコードは画面 1 と 2 を unknown にした。画面 3 は、issue に貼られた文字どおり(prompt の行のすぐ下に `Read 2 files`)なら stopped で、prompt の下に空行がある実機の形なら unknown だった。3 つとも (c2) で stopped になる
- `⎿` の行ならどれでも上限と数えると、返答が座席の画面を引用しただけの画面(`⏺ The seat showed:` の下に `⎿  You've hit your weekly limit · resets …`)も stopped になる。leader や director は座席の画面を引用することがある。計画の助言(codex、MEDIUM 1 件)がこれを挙げ、consult が 3 点を足した。表に `⏺ Monitor "` がない、すぐ上の行だけを見ると知らせの行の折り返しを取りこぼす、prompt のすぐ下の結果のまとまりを数える必要がある。規則は「上限の行から上へ、`⎿` の行と折り返しの行を飛ばして最初に当たる項目の頭が、ツールか知らせの形のとき」になった

## 3. 規則 (c2)(読む順)

`ClassifyResumeScreen`(`internal/org/resume_screens.go`)の手順 4 で、(c) の後ろで `findResumeLimitResult` を呼ぶ。claude だけに掛かる。次の 3 つをすべて満たすと stopped で、`Line` は上限の `⎿` の行からまとまりの終わりまで(`/usage-credits …` の行とリセットの時刻を含む)。

1. 候補のまとまり(prompt のまとまりは除く)の 2 行目以降で、`⎿` で始まる行、`⏺` で始まる行、まとめた行のうち最後のものが、`⎿`、空白、`UsageLimit` の文の順で始まる。上限の行より下に項目の頭があれば、上限のあとも手番が続いたので当てない(計画にない守りで、実装者が足した)。上限の行の下の `⎿  Interrupted` やツールの出力の行も、最後の `⎿` の行が上限の文でなくなるので当てない。照合は、その行とまとまりの残りを空白 1 つでつないだ文で行うので、折り返された上限の文も読む(self-review 周回 1 の L2)
2. 上限の行の項目の頭が、ツールか知らせの形(`resumeResultUnderToolOrNotice`)。頭は、上限の行から上へ `⎿` の行と記号のない行を飛ばして最初に当たる `⏺` の行かまとめた行。形は表の欄 `ToolOrNoticeHeads` にある `⏺ <ツール名>(`、`⏺ Monitor "`、`⏺ Background command "`、`Ran <n> shell command`、`Read <n> file`(後の 2 つは `⏺` のない形もある)。まとめた行の言葉は実機の画面で見た `Ran` と `Read` だけ。頭が返答の文(`⏺ The seat showed:`)なら数えない
3. 頭がまとまりの先頭まで見つからないときは、そのまとまりが prompt の行のすぐ下の結果のまとまり(`resumeBlock` の `resultUnderPrompt`)のときだけ数える(issue の画面 3)

API Error の文は今までどおり、候補の先頭でだけ数える。codex の表には結果の記号の欄がないので (c2) は働かない。実機の codex の画面を見ていないため。

## 4. 直し 2 回

- self-review 周回 1 の M1(`d470a77a`): 頭のない結果のまとまりの許しが、prompt のすぐ下の `⎿` で切ったまとまりかを見ず、空行のあとで `⎿` から始まる引用も stopped と読んだ。`resumeBlock` に `resultUnderPrompt` を足し、許しをその印だけにした
- cross-review 周回 1(codex、指摘 1 件、ACTION_REQUIRED。`1042ccae`): prompt のすぐ下に空行なしで返答の頭(`⏺ The seat showed:`)が来ると、頭が prompt のまとまりに入って捨てられ、下の引用の `⎿` の行に `resultUnderPrompt` が付いて、引用しただけの返答が stopped になった。`resumeScreenBlocks` が、prompt のあとの項目の頭の行からも新しいまとまりを始めるようにした
- self-review 周回 2 の M1(`e9017d7f`): `1042ccae` の切れ目は字下げを見ず、座席の画面を写した prompt の字下げした `  ⏺ API Error: …` の行も項目の頭と読んだ。Claude Code は項目の `⏺` を 1 桁目に描き、prompt の折り返しの行を字下げする(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 2 節、`docs/evidence/org-seat-resume-live-2026-11-07.md` の 0 節)ので、字下げを落とす前の行が `⏺` で始まるときだけ頭にした。直す前は、そうした prompt のあとで手番が中断された画面が stopped になり、見張りも上限と読んだ(self-review の probe P1、P2、P9、P12)。同じ commit で `resumeScreenBlocks` の引数を marker から driver に変えた(L2)
- cross-review 周回 2(HEAD `2e4601ac`)は指摘なし。周回は 2/2 で、上限に届いている

## 5. 挙動の変化

新しい binary で変わる画面は 4 つある。

1. 意図した変更。issue の 3 画面と、#353 の Monitor の知らせの下の weekly limit(retry の行なし)が、unknown から stopped になる。`ralph org resume` は続行の文を打つ対象にし、見張りは stall の代わりに `usage_limited` を上げる
2. 計画になかった変更。prompt のすぐ下(空行なし)の 1 桁目の `⏺ API Error: …` が、unknown から stopped になる。以前は prompt のまとまりに入って prompt_last の unknown だった。今は独立したまとまりの先頭の通信断の文として止まったと読み、resume は続行の文を打つ。同じ位置の 1 桁目の `⏺ You've hit your weekly limit …` は、見張りも上限と読む。手順 1 の新しい切れ目の帰結で、(c2) の外の変化にあたる。空行のある実機の描き方(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 2 節。base でも stopped)と答えがそろう。仕様 FR-2 の 4(`docs/specs/2026-11-07-org-seat-resume.md:54`)が言い、`internal/org/resume_screens_test.go:522` の行が固定する。verify の V18
3. stopped から unknown に変わる向き。返答の頭(`⏺ The seat showed:`)のすぐ下に、上限か API Error を 1 行だけ `⎿` で引用した画面は、base では stopped だった誤読で、今は unknown(self-review 周回 2 の probe P6、P10)
4. prompt のすぐ下に 1 桁目の `⏺` の待ちの文(`⏺ You've hit your Opus limit · resets 3:45pm` と `Continuing shortly · Esc to cancel`)がある画面は waiting になる(以前は prompt のまとまりに入って unknown)。実機でこの形は見ていない。`TestClassifyResumeScreen_Edges` の行がコードと仕様の答えを固定しただけ(9 節)

## 6. 見張り、send、文書、配布

- 見張り: `UsageLimitScreen`(`internal/org/usage_limit_screen.go`)は resume の判定が stopped のときだけ上限を読み、`Line` を `matchedStopSentence` で照合する。(c2) で同じ画面の座席を見張りも上限と読み、`usage_limited` がそろう。読み方のコードは変えておらず、コメントと `resumeScreenBlocks` の引数だけを直した。retry の行を読む 3 つ目の道は、retry の状態行が最後のまとまりのときだけ働く(#353 の計画の AC4。`internal/org/usage_limit_screen_test.go:606`)
- send: `send_screen.go` の `sendScreenBusyCheck` の呼び出しの引数が driver になった。send の prompt のまとまりは項目の頭では切れないので、send の判定は変わらない(self-review 周回 2 の probe は 3 つの版で同じ答え)。この違いはコメント 2 か所(L1)に書いた
- 文書: 仕様 `2026-11-07-org-seat-resume.md` の FR-2 の 4 に 2026-11-22 の改訂(58 行から 61 行の (c2) の 3 条件)と、54 行の 1 桁目の `⏺` の区切りの追記。205 行の Open questions に 1 項目。rule `.claude/rules/ralph/agent-messaging.md:255` の `usage_limited` の段落に頭の例(2 写し、`cmp` で一致)。`docs/specs/2026-08-01-org-runtime.md:89` に 1 文。`/director` skill と `ralph org resume --help` は「既知の文で終わる画面」と書いていて (c2) でも正しいので変えていない
- 配布: 新しい binary が要る。動いている見張りは立て直すまで前の読み方で、(c2) の画面の座席に stall を上げる。revert すると、上限の文が候補の先頭にある画面だけを読む前の形に戻る

## 7. 先に読むテスト

- `internal/org/resume_screens_test.go:776` の `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine`。issue の 3 画面を定数(`resumeIssue373Screen1` から `3`)にして、入力欄と footer を足した形(`claudeTightResumeScreen`)で読む。否定の行は、API Error が最後の `⎿`、上限の下の `⎿  Interrupted`、返答の文の中の引用、返答の頭の下の引用(状態行のある形とない形)、記号のない行の上限の文、見ていないまとめた行(`Searched for 1 pattern`)、codex の表。codex の `⎿` の行が unknown であることは 7a7afdca の行が固定する
- `internal/org/resume_screens_test.go:49` の `TestClassifyResumeScreen_Fixtures`(fixture 5 件、`internal/org/testdata/resume/claude-5h-limit-after-*.txt` 4 件と `claude-weekly-limit-after-monitor.txt`)。同じ 5 件を `internal/org/send_screen_test.go` の表が send で idle と固定し、`internal/org/usage_limit_screen_test.go` の表が上限の行とリセットの時刻を固定する
- `internal/org/resume_screens_test.go:1058` の `TestResumeScreenBlocks_ItemHeadUnderAPrompt`(切れ目)と `:1286` の `TestResumeResultGlyphs_AreStopGlyphs`(結果の記号が止まりの記号に含まれる)
- `internal/cli/org_resume_test.go:426` の `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice`。fake の herdr で issue の画面 1 と 2 が `would_resume` 2 件になる。直す前のコードでは issue と同じ `skipped … unknown screen` で落ちることを確かめた
- 期待を直した既存の 2 件。`internal/org/resume_screens_test.go:500` の「only the first ⎿ line under a prompt starts a block, the next ones join it」は、2 つ目の `⎿` を上限から API Error に変えて「つながる」意図を残し、上限の形は次の行が stopped を期待する。`internal/org/usage_limit_screen_test.go:745` の「a limit in the turn above a turn-end status line, with no retry」(Monitor の知らせの下の上限)は、unknown から stopped と上限の行の期待にした

## 8. 確かめたこと

- `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 0。周回 1 は HEAD `836cb19f` で 1,349 秒。周回 2 は HEAD `7a7afdca` で 1,172 秒(shell 46 本、`PASS` の行 5,390、Go 11 パッケージ、SKIP は条件つきの 2 つだけ)。`-race -count=1` で `Resume|UsageLimit|SendScreen` は ok、data race 0。足した行は `-v` で SKIP 0
- 変異: implementer の 15 個のうち 14 個が落ち、1 個は等価。tester の周回 1 は 7 個のうち 6 個が落ち、codex の表に `⎿` を足す M4b が残ったが、`7a7afdca` の codex の行で落ちるようになった。周回 2 の 4 個(手順 1 の切れ目を外す、字下げを落とした行で比べる、頭を prompt に残し印を付けない、M4b)はすべて落ちた
- `./scripts/run-static-verify.sh` は周回 1 と 2 の verify で rc 0(`check-sync.sh` は PASS)。AC1 から AC5 は Met。周回 2 の AC5 は、仕様 54・61 行が cycle 2 の切れ目を言わない点が一部だったが、周回 2 の sync-docs で直した
- self-review は周回 1 が MEDIUM 2、LOW 4、周回 2 が MEDIUM 1、LOW 3。コード側は `d470a77a`、`e9017d7f` で直し、記録側は sync-docs で直した。cross-review は read-only の sandbox で回したので、codex は Go のテストを走らせていない(テストは `/test` の回)

## 9. 残したもの

tech-debt の台帳(`docs/tech-debt/README.md` の 570 行から 576 行)に 7 行を足した。

- 実機の画面がない(574 の (c))。fixture は issue の文字に、既存の 2.1.316 の fixture の入力欄と footer と空行を足したもの。まとまりのあいだの空行は仮定。実機の `herdr pane read` を `docs/evidence/` に置いて fixture に足すのが次
- 571: 返答が空行を置いて `Ran 1 shell command` などのまとめた行と上限の `⎿` の行を引用する形(P2)は、issue の画面 3 と画面の上で区別できないので stopped のまま。テストが既知の穴として固定している
- 572: ツールの出力が入れ子の `⎿` の行を持つ形(`ralph org read` の出力など)と、出力の 1 行目が上限の文の形(`cat` した文書など)を stopped と読む。まとまりが行を trim するので字下げの深さを見られない。どちらも、そのあと手番が返答なしで終わるときだけ
- 573: 見ていない頭(`Searched for 1 pattern`、MCP のツールの `⏺ context7 - resolve-library-id (MCP)(…)`、`⏺ Web Search("…")`)の下の上限は unknown のまま。取りこぼす向きで、前より悪くならない
- 574 と 576 のテストの穴: 仕様と rule の文を読むテストがない(`check-sync.sh` は写しの一致だけ)。5 節の 4 の waiting の行は実機の画面がない。send の prompt のまとまりが項目の頭で切れないことは overlay の probe で 1 回確かめただけで、テストの行がない
- 575: prompt の 2 段落目と、`⎿` で始まる折り返しの行は、base からあった形で、普通のまとまりとして読まれて stopped になりうる
- 570: ファイルの大きさ。`resume_screens.go` は 705 から 935 行、`resume_screens_test.go` は 781 から 1,303 行で、どちらも 800 行を超えた。(c2) の 5 関数とテストのまとまりを移しても約 850 行と約 980 行で、800 行を超える。移動だけの split は計画の Scope の外。`send_screen.go` もコメント 2 か所で 798 から 804 行になった(524 行の (h))
- 既存の行では、Trigger を満たした行に繰り越しの注を足した(周回 1 で 7 行、周回 2 で 13 行)
