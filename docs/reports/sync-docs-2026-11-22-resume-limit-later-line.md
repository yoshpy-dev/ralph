# sync-docs report: resume-limit-later-line

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(承認の digest `9683a080e69b`。この run の前後とも同じ)。issue #373
- Branch: `fix/resume-limit-later-line`。run 開始時の HEAD は `32440422`(origin/main `38932c13` = #361 と #374 の merge を取り込んだ merge commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit): self-review `a3154042`(Merge: yes、MEDIUM 2、LOW 4。M1・L2・L3・L4 は `d470a77a` で直った。M2 は台帳なので今回、L1 は Known gap)、verify `836cb19f`(pass、LOW 4・INFO 4)、test `90d5d824`(pass、`run-test.sh` rc 0、変異 7 個のうち 6 個が落ち、codex の M4b が残った)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。回したのは `check-sync.sh`、`check-skill-sync.sh`、`plan-visual.sh digest`、狭い Go テスト(下の「確かめたもの」)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 19 ファイル、+1,177/-41。Go は全体で +680/-39、テスト以外の Go が +236/-34(`resume_screens.go` +224/-27、`usage_limit_screen.go` +12/-7)。walkthrough を書くかどうかは `/pr` が決める

## Summary

verify の引き継ぎ 5 項目と、test report の Test gap 1 を片づけた。台帳は、Trigger を満たした既存の 7 行に日付つきの繰り越しの注を足し(verify の 6 行に、掃き出しで見つけた行 567 を足した)、新しい行を 5 つ(大きさ、既知の穴 3 つ、テストの穴 1 つ)足した。仕様 `2026-11-07-org-seat-resume.md` は、verify の V2・V3・V4 が指した 59・60・61 行を直した。計画の Progress は S1 の古い名前と `d470a77a` の行を直し、`Sync-docs artifact created` の行を足して、AC1〜AC5 に `[x]` を付けた。任意の 2 点(rule の頭の例、`2026-08-01-org-runtime.md` の 1 文)も入れた。コードは触っていない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-11-07-org-seat-resume.md` | FR-2 の 4 の 2026-11-22 の改訂の 3 か所。(1) 59 行: 最後の行を選ぶときは上限の行の下の印のない行を数えないが、照合はその行とまとまりの残りを空白 1 つでつないだ文で行い、pane の幅で割れた上限の文も読む(V3、`d470a77a` の L2)。(2) 60 行: 返答が空行を置いて、まとめた行と `⎿` の上限の行を引用した形は、まとめた行が頭になって止まったと読む。issue の画面 3 と区別できないので既知の穴で、テストが固定している(V4、P2)。(3) 61 行: 頭がないときの条件を「prompt の行のあとの最初の `⎿` の行で始まったまとまり」にし、空行のあとで `⎿` から始まるまとまりは数えない、とした(V2、`d470a77a` の M1) | この report の commit |
| `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md` | usage_limited の段落の頭の例を「under a head such as `Ran 1 shell command`, `Read 2 files`, `⏺ Bash(...)`, `⏺ Monitor "..."`, or `⏺ Background command "..."`」にした(V5)。2 写しは `cmp` で同一 | この report の commit |
| `docs/specs/2026-08-01-org-runtime.md` | 89 行((a) の usage_limited の項)の末尾に `2026-11-22 追記` の 1 文: (c2) の画面の座席も利用上限の対象になる(V7)。同じ行の「止まったと読み、決め手の文が利用上限の文」は (c2) のあとも正しいので、本文は変えていない | この report の commit |
| `docs/tech-debt/README.md` | Trigger のセルの末尾に `2026-11-22: fix/resume-limit-later-line (#373) ...` の注、Related のセルの末尾に `docs/plans/active/2026-11-22-resume-limit-later-line.md` を、行 462・483・490・517・542・544・567 に足した。新しい行 5 つ(570〜574 行目): 大きさ、(c2) が引用を読む形(P2)、入れ子の `⎿` の行(L1)、見ていない頭、この branch のテストの穴 | この report の commit |
| `docs/plans/active/2026-11-22-resume-limit-later-line.md` | Progress の S1 の行の `ResultHeads` を `ToolOrNoticeHeads` にし、改名が `d470a77a` だと書いた。`Implementation started` の下に `d470a77a` の子の項目。`Test artifact created` の次に `Sync-docs artifact created`。AC1〜AC5 に `[x]`。本文は変えていない。digest は前後とも `9683a080e69b` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-resume-limit-later-line.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-resume-limit-later-line.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 | この report の commit |

コードに触れていない。仕様、rule、台帳、計画を直したので、`post-implementation-pipeline.md` の「記録だけの修正」の例外には当たらない。sync-docs の次は cross-review で、self-review・verify・test を回し直すかは lead が決める(rule の文は変えたが、コードとテストは変えていないので、`resume_screens.go` の挙動の確かめは前の run のままである)。

## 渡された一覧の行き先

### 1. 台帳の繰り越しの文

行は最初の語で探した。verify の 6 行(行 490・517・542・544・462・483)は、self-review と verify の行番号が merge のあとの HEAD でも同じだった。注は「`<何を変えたか>` で Trigger を満たしたが、払わなかった: `<理由>`。Carried over to the next change that meets this trigger」の形で、Related の末尾に計画を `docs/plans/active/...` の path で足した(`/pr` の archive が `archive/` に書き換える)。

| 行(最初の語) | 満たした Trigger | この PR が触ったもの | 扱い |
|------|------|------|------|
| 490 `` `internal/org/send_screen_test.go` (662 → 831 lines) `` | `send_screen_test.go` の次の変更 | `sendScreenFixtureWants` に 5 行。885 → 893 行(origin/main `38932c13` と `d470a77a` で数えた)。`send_fallback_test.go`(835)は触っていない | 繰り越し。数字の行のテストは移していない。計画の Progress の S1 が、最初の implementer がこのテストで止まり範囲を足したと書いているので、それを理由に引いた |
| 517 `Test gaps of fix/claude-5h-usage-limit (#329)` | 「`usage_limit_screen_test.go` に次のテストを足すとき」と `UsageLimitScreen` の次の変更(境界) | `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`、`usageLimitFixtureWants` に 5 行、`TestUsageLimitScreen_RetryWaitShapes` の 1 行の期待。`UsageLimitScreen` はコメントだけで、コードは変えていない。960 → 1013 行 | 繰り越し。(a)〜(d) は足していない |
| 542 `` `internal/org/usage_limit_screen_test.go` and `internal/org/resume_test.go` went past `` | 次の変更で先に #353 の塊を移す | 同じ 960 → 1013 行。塊の中の 1 行(Monitor の知らせの下の上限)の期待を直した。`resume_test.go`(869)は触っていない | 繰り越し。塊は移していない |
| 544 `Three LOW findings of the cycle 1 self-review of fix/watch-usage-limit-retry (#353)` | (c) `usage_limited` の段落の次の変更 | rule の段落を 2 写しとも変えた(S2 の `90a93764` と、この sync-docs の頭の例)。(a)(b)の名前は触っていない | 繰り越し。retry の場合は入れ子の括弧のまま。**自分の編集でも当たった行**(V5 を直したため) |
| 462 `Seven mutants of `ralph org resume` survive` | `ClassifyResumeScreen` の次の変更 | (c2) の呼び出し(`2c418694`)と `resumeScreenBlocks` の印(`d470a77a`)。表のテストの 5 行と新しいテスト | 繰り越し。(d)(両方の文を出す fixture)は新しい fixture でも満たしていない。(a)(b)(f)(g)は計画の Scope の外 |
| 483 `A hook that was changed after codex approved it` | `resume_screens.go` の画面の規則の次の変更 | (c2)(表の 3 つの新しい欄)。`readUsageLimitScreen` は #353 のまま | 繰り越し。"Hooks need review" の条件は足していない |
| 567 `A claude reviewer seat runs a feature org's gate with `run_in_background`` | 「見張りが座席の画面を読む方法(`readUsageLimitScreen`、`resume_screens.go`)の次の変更」 | `resume_screens.go` の (c2)。見張りの `UsageLimitScreen` は同じ画面を読む | 繰り越し。**handoff にも verify にもない行**(掃き出しで見つけた。下)。(c2) は `⏺ Background command "…"` の知らせの下の上限を読むが、これは終わった command の知らせで、動いている背景の task の画面の規則ではない、と書いた |

行 490 の `send_screen_test.go` の数(885 → 893)と `resume_screens*` の数(705 → 902、781 → 1,119)は、`git show origin/main:<file> | wc -l` と `git show d470a77a:<file> | wc -l` と HEAD の `wc -l` で測り直した。merge で入った main の変更はこの 4 ファイルに及んでいないので、`d470a77a` と HEAD は同じ数だった(`internal/org/usage_limit_screen_test.go` は 960 → 1013)。

### 2. 台帳の新しい行

| 行 | 内容 | 出どころ |
|------|------|------|
| 570 | 大きさ。`resume_screens.go` 705 → 902、`resume_screens_test.go` 781 → 1,119(self-review の 882・1,056 は `d470a77a` の前の数なので、両方書いた)。分け方は (c2) の 5 つの関数(約 90 行)を `resume_screens_limit.go` へ、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` と issue #373 の定数と `claudeTightResumeScreen`(約 260 行)を `resume_screens_limit_test.go` へ。移したあとも約 810 行と約 860 行で 800 を少し越える、と Impact に書いた(`sed` で関数の範囲を測った) | self-review M2 (e)、verify の引き継ぎ 2 |
| 571 | 既知の穴 P2: 空行のあとにまとめた行で始まる引用が、issue の画面 3 と区別できず stopped になる。self-review の行案 1 の「M1 を直さない場合は `⎿` から始まる 2 行以上の引用(P1)」は、M1 が `d470a77a` で直ったので落とした。Trigger の末尾で、返答の段落の描かれ方を知る同じ取得(箇条書きの返答つきの実機の画面)を、行 488 の (L6) と行 543 が求めていることに触れた | self-review の行案 1、verify V4 |
| 572 | 既知の穴: ツールの出力の中の入れ子の `⎿` の行(L1)と、出力の 1 行目が上限の文(計画の Risks の 2 つ目)。`resumeScreenBlocks` が行を trim して字下げを捨てることを原因に書いた | self-review の行案 2、L1 |
| 573 | 既知の穴: 見ていない頭(`Searched for …`、MCP、`Web Search(`)は unknown。行案 3 にあった「pane が狭く上限の文が割れた行」は、`d470a77a` の L2 で頭のわかる形は読めるようになったので、その旨に書き換えた | self-review の行案 3、計画の Risks の 1 つ目、verify の Coverage gaps |
| 574 | テストの穴(依頼の「この branch の行にテストの穴 1 を足す」): (a) codex の行が「(c2) は claude だけ」を挙動で固定せず、M4b が残る(overlay の probe の形も書いた)、(b) 仕様と rule の文を読むテストがない、(c) fixture は実機の pane の読み取りではなく、ブロックの間の空行は仮定。(b) と (c) は test report の Test gaps 3・4。Test gap 2 は行 571〜573 の穴で、この行の冒頭が名指しする | test の Test gaps 1・3・4、verify の Coverage gaps の最後の項 |

新しい行はどれも `file:line` の参照を持たず、関数とテストの名前で書いた。

### 3. 仕様 `2026-11-07-org-seat-resume.md`

行番号は verify の値(59・61 行)が HEAD でも同じで、最初の語でも見つけた(59 行は `まとまりの 2 行目以降で`、61 行は `まとまりの先頭まで頭がないときは`)。P2 の 1 文は handoff の指示どおり 60 行(頭の形を言う項目)の末尾に足した。`resume_screens.go` の前文の (c2) の段落は `d470a77a` で直っていて、仕様の 3 文はそれと同じ条件を言う(「prompt の行のあとの最初の `⎿` の行で始まったまとまり」)。Open questions の補足(205 行)は、この問いの形が prompt のすぐ下の結果のまとまりであることを言うだけで、`d470a77a` のあとも合うので変えていない。

### 4. 計画の Progress

S1 の行は `ResultHeads` を `ToolOrNoticeHeads` に直し、`2c418694` の時点の名前が `ResultHeads` で、`d470a77a` が改名したと書いた(記録は残した)。`d470a77a` の子の項目は S2 の次に置き、M1(`resultUnderPrompt`)、L2(つないだ文の照合)、L3(glyph の部分集合のテスト)、L4(改名 4 つ)と、残したもの(M2 は台帳、L1 と P2 は台帳の行と Known gaps)を書いた。この計画の Progress には sync-docs の項目がなかったので、ほかの計画の最近の形(`Test artifact created` の次)で `Sync-docs artifact created` の行を足した。AC は verify の表のとおり 5 つとも Met で、AC5 は仕様の 2 文を直し `check-sync.sh` が通ったあとに付けた。

### 5. 任意の 2 点

rule の頭の例は「such as」にし、`Read 2 files` と `⏺ Monitor "..."` を足した(コードの `ToolOrNoticeHeads` と同じ 5 形のうち、`⏺ <ツール名>(` は `⏺ Bash(...)` で例示)。この段落は台帳の行 544 (c) の「次の変更」にも当たるので、注を足した(上の表)。`2026-08-01-org-runtime.md` は 89 行の末尾に日付つきの 1 文を足した(上の表)。

## 掃き出し(自分で全部の open な行の Trigger を読んだ)

Trigger のセルを python で `(?<!\\) \| ` で切って 4 つ目のセルを読み、全体が `~~` の行を飛ばし、この branch が触ったファイル・関数・文書(`resume_screens.go`、`ClassifyResumeScreen`、`UsageLimitScreen`、`usage_limit_screen*.go`、`send_screen_test.go`、`org_resume_test.go`、`testdata/resume`、`agent-messaging.md`、`usage_limited`、`org-seat-resume`、`org-runtime`、`StopGlyphs`、`sendScreenFixtureWants` など 38 語)で照らした。59 行が語に当たり、Trigger の頭を読んで次のとおり分けた。

- 当たった(注を足した): 462・483・490・517・542・544・567(上の表)。567 が handoff と verify にない行
- 当たらない(理由): 457(「最初の利用上限の止まり」。claude の 5 時間の文は #329 で `seen:` に済み、残る `docs:` の文と codex の 2 文の画面ではない)、464(`resumeOne`、`org_resume.go` の結果の行は触っていない。`org_resume_test.go` だけ)、465・466(`resumeCodexFooterLine`、`StopGlyphs`(取り消し線)、`empty` の placeholder の規則は触っていない)、479・488・489(`send_screen.go`、`send_fallback.go`、`status_live.go` は触っていない。488 の (L6) の Trigger は実機の箇条書きの返答つきの画面の取得で、この branch は取っていない)、491・492・493・494・495・507(`watch_usage_limit.go` は触っていない)、514・516・543(`usageLimitWaitingFooter`、`usageLimitRetryingTurn`、`claudeStatusLineStart` は触っていない)、566(`agent-messaging.md` は TASK_ID を足す変更の行き先として名指しされるだけで、Trigger は別)、88・118・182・267・358・362・370・420・441・442・447・448・470(`usage_limited` か `org-seat-resume` が古い日付の注の中に出るだけ)、455・456・463(Trigger は別の出来事で、`resume_screens.go` は行き先に挙がるだけ。455 は長い無音の報告か phase 2 の開始、456 は長い停電のあとの codex の座席、463 はリセットのあとの AC-10 の実機の確かめ)、453(全体が取り消し線で閉じている)
- 自分の編集が当たる行: `agent-messaging.md` の編集は 544(上で注を足した)。`2026-08-01-org-runtime.md` と `2026-11-07-org-seat-resume.md` の編集は、`org-runtime` と `org-seat-resume` を Trigger に持つ行を `grep` で全部見て、どれも古い日付の注の中の言及で、足す注はなかった。`docs/tech-debt/README.md` 自体を Trigger に持つ行は、新しい行 5 つの書き方(関数名で書く)で足りている

## 確かめたもの

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。rule の 2 写しは `cmp` で同一
- `./scripts/check-skill-sync.sh`: rc 0(skill は触っていない)
- `./scripts/plan-visual.sh digest`: `9683a080e69b`(編集の前後とも同じで、計画の `- Approved:` の行と一致)
- `go test -count=1 ./internal/org/ -run 'DocsMatch|AgentMessaging|Resume|UsageLimit'`: ok(7.4 s)。`go test -count=1 ./internal/cli/ -run 'Resume|DocsMatch'`: ok(16 s)。仕様と rule の編集で落ちる固定はなかった
- 台帳: 変えた行 7 つと新しい行 5 つが 5 つのセル(`(?<!\\) \| ` で切った)、セルごとにバッククォートと `~~` の数が偶数、行末が ` |`。`git show HEAD:` と比べて、変わった既存の行は 462・483・490・517・542・544・567 だけで、どれも挿入のみ(1 行ごとに difflib の操作が insert だけ)。新しい行に `file:line` の参照はない

## 確かめていないもの

- `./scripts/run-test.sh` と `run-verify.sh` は回していない(依頼の指示)
- 実機の Claude Code の画面での (c2) の判定、`ralph org resume` と見張りの実機での動き(test report と同じ)
- 台帳の新しい行の「移したあとの約 810 行・約 860 行」は関数とテストの範囲の行数から引いた見積もりで、移して `wc -l` を取った値ではない

## PR 本文の Known gaps に入れる点

1. P2: 返答が空行を置いてまとめた行(`Ran 1 shell command`)と `⎿` の上限の行を引用した画面は、issue の画面 3 と区別できず stopped と読む(台帳 571)。resume は続行の文を打ち、見張りは stall の代わりに usage_limited を上げる
2. ツールの出力の中の入れ子の `⎿` の上限の行と、出力の 1 行目が上限の文で手番が黙って終わった画面を stopped と読む(台帳 572、L1)
3. 見ていない頭(`Searched for …`、MCP、`Web Search(`)の下の上限は unknown で、resume は打たない(台帳 573)
4. codex の行が「(c2) は claude だけ」を挙動で固定していない。codex の表に `⎿` を包含つきで足す変異 M4b が残る(台帳 574 (a))
5. 仕様と rule の文を読むテストがない(574 (b))。fixture は実機の pane の読み取りではなく、ブロックの間の空行は仮定(574 (c))
6. `resume_screens.go`(902 行)と `resume_screens_test.go`(1,119 行)が 800 行を越えた(台帳 570)
7. 満たした Trigger を払わなかった行 7 つ(462・483・490・517・542・544・567)に繰り越しの注を足した

## Insight event

`./scripts/insights-append.sh --slug resume-limit-later-line --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を回した。rc 0。`docs/insights/events/2026-11-22-resume-limit-later-line.jsonl` は 3 行から 4 行になり、`git status --short` に迷子の jsonl はない(slug は日付なし)。
