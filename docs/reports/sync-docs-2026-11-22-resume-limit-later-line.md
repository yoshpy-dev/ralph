# sync-docs report: resume-limit-later-line

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(承認の digest `9683a080e69b`。この run の前後とも同じ)。issue #373
- Branch: `fix/resume-limit-later-line`。run 開始時の HEAD は `56319352`(push 済み。origin/main `38932c13` から分岐していて、main は進んでいない)
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2)。今の内容はこの節で、Run 1 は見出しを 1 段下げて `---` の下に残した
- 対象: Run 1 の report のコミット `1dda5f7f` のあとに足されたコミット。コードは `1042ccae`(cross-review cycle 1 の ACTION_REQUIRED 1 の直し)と `e9017d7f`(self-review Run 2 の M1・L1・L2 の直し)、テストは `7a7afdca`(6 行)。記録は `b061c0c9`(triage)、`c4e6f979`(self-review Run 2)、`f3956eb6`(verify Run 2)、`56319352`(test Run 2)
- 先行 report: self-review Run 2(Merge: yes、MEDIUM 1・LOW 3。M1・L1・L2 は `e9017d7f` で直った。L3 と Tech debt の 2 行目は今回)、verify Run 2(pass、LOW 8(うち依頼で既知の 3)・INFO 4。V9〜V20 は今回)、test Run 2(pass、`run-test.sh` rc 0、変異 4 個すべて落ちた。Test gaps 2・3 は今回)
- 制約: Go のコードとテストは触っていない。`run-test.sh` と Go のテストは回していない(依頼の指示)。回したのは `check-sync.sh`、`run-static-verify.sh`、`plan-visual.sh digest` と台帳の整合の確かめ

### Summary

cycle 1 の sync-docs のあとに入ったコード 2 本とテスト 1 本に、仕様、台帳、計画の Progress を合わせた。仕様 FR-2 の 4 は、54 行に 1 桁目の `⏺` の行での区切りと、prompt のすぐ下の 1 桁目の `⏺ API Error` が unknown から stopped に変わったこと(V18)、prompt の後ろの段落の既知の穴を足し、61 行に「あいだに項目の頭がない」を足した。台帳は、既存の 13 行に数の直しか繰り越しの注を足し(verify が名指しした 12 行と、掃き出しで見つけた 543)、新しい行を 2 つ(575、576)足した。574 の (a) は `7a7afdca` で閉じたので取り消し線にした。計画の Progress は `1042ccae` と `7a7afdca` の項目を足し、verify・test・sync-docs の行に cycle 2 を足した。rule の usage_limited の段落(2 写し)と `2026-08-01-org-runtime.md` の 89 行は cycle 2 のあとも正しいので変えていない。

### Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-11-07-org-seat-resume.md` | FR-2 の 4 の 2 か所。(1) 54 行の末尾に `2026-11-22 追記(issue #373 の cross-review と self-review の直し)`: まとまりは prompt のあとの 1 桁目の `⏺` の行でも区切る。その行が始めるまとまりは prompt のまとまりではなく、下の `⎿` の行はそのまとまりに入る。字下げした `⏺` の行(座席の画面の行を写した prompt の続き)は prompt のまとまりに残る。prompt のすぐ下(空行なし)の 1 桁目の `⏺ API Error: …` は、以前は不明で、今は止まったと読む(V18)。待ち中と Enter だけの文もそのまとまりにあれば読む。prompt の中の空行より下の段落と `⎿` で始まる折り返しの行はふつうのまとまりとして読まれる(既知の穴、台帳に行がある。V19)。(2) 61 行: 頭がないときの条件に「prompt の行と `⎿` の行のあいだに 1 桁目の `⏺` の行のような項目の頭がない」を足し、prompt のすぐ下の返答の頭の下の `⎿` の行は頭のまとまりに入って頭が決める、と足した(V9) | この report の commit |
| `docs/tech-debt/README.md` | 既存の 13 行(462、483、488、490、517、524、542、543、544、567、570、572、574)と新しい行 2 つ(575、576)。内訳は下の表 | この report の commit |
| `docs/plans/active/2026-11-22-resume-limit-later-line.md` | Progress だけ。`Implementation started` の下に `1042ccae` の項目(V20)と `7a7afdca` の項目を足し、Verification・Test・Sync-docs の行に cycle 2 を足した。本文は変えていない。digest は前後とも `9683a080e69b` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-resume-limit-later-line.md` | この report(Run 2 を先頭に足し、Run 1 は見出しを 1 段下げて残した) | この report の commit |
| `docs/insights/events/2026-11-22-resume-limit-later-line.jsonl` | `sync_docs` の行(cycle 2、pass)。8 行から 9 行 | この report の commit |

台帳の行ごとの変更(行は 1 つ目のセルの書き出しで探せる。番号は HEAD のもの)。

| 行(最初の語) | 何を足したか | 出どころ |
|------|------|------|
| 462 `Seven mutants of` | Trigger の 2026-11-22 の注に cycle 2 の文: `resumeScreenBlocks` の変更で `ClassifyResumeScreen` の読みが変わったこと(1 桁目の `⏺ API Error` は stopped、返答の頭の下の `⎿` は (c2) に入らない)。未解決の項目は 1 つも払っていない | V16 |
| 483 `A hook that was changed` | 同じく cycle 2 の文。`readUsageLimitScreen` は #353 のまま、「Hooks need review」の条件は足していない | V16 |
| 488 `Seven small findings` | 新しい注(`send_screen.go` のコメント 2 か所と `sendScreenBusyCheck` の呼び出しで (L2, L7, L9, L10) の Trigger を満たしたが、どれも払っていない)と Related に計画 | V13 |
| 490 `internal/org/send_screen_test.go` | 数を測ったコミットを `7a7afdca` にし、ファイルは cycle 2 で触っていない(893 は `d470a77a` でも同じ)と書いた | V17 |
| 517 `Test gaps of fix/claude-5h-usage-limit` | 同じ。`1013` は `d470a77a` でも同じで、`e9017d7f` が補助 `usageLimitRetryingLine` の 1 行を変えただけでテストは足していない | V17 |
| 524 `Known gaps of fix/send-codex-paste` | (h) の数に「`7a7afdca` で 804 行、目安を越えた(`e9017d7f` の 2 つのコメントで 798 から)」。Trigger の (h) の注(払っていない、規則は足していないので「次の規則の前」には反しない)。Related に計画 | V13 |
| 542 `internal/org/usage_limit_screen_test.go and` | 測ったコミットを `7a7afdca` にし、`usageLimitRetryingLine` の 1 行は塊の外と書いた | V17(下の訂正) |
| 543 `Known gap of fix/watch-usage-limit-retry (#353), the residual` | 新しい注: `usageLimitRetryingTurn` の呼び出しが `driver` を取るようになり、手順 1 の変更で今の手番に入るまとまりが変わった(prompt のすぐ下の 1 桁目の `⏺` の行)。払っていない。Related に計画 | 掃き出し(verify・self-review にない行) |
| 544 `Three LOW findings` | cycle 1 の注の「(a) and (b) were not met: `usageLimitRetryingTurn` … are as they were」が偽になったので、「(b) は満たさず、(a) は `e9017d7f` が満たして払っていない」に直した | V14 |
| 567 `A claude reviewer seat runs` | cycle 2 の文(prompt のすぐ下の 1 桁目の `⏺ You've hit …` は上限と読まれ、字下げした引用は読まれない)。背景 task の画面の規則は足していない | V16 |
| 570 `internal/org/resume_screens.go (705` | 数を 935 と 1,303 にし、測ったコミットを `7a7afdca` にして途中の数(902・1,119、932・1,205、935・1,261)を書いた。分けたあとの見積もりを約 850 と約 980 にし(旧: 約 810 と約 860)、「約 260 行」に `7a7afdca` で約 320 を添えた。Trigger を cycle 2 で満たしたが払っていない、と足した | V10 |
| 572 `Known gap of fix/resume-limit-later-line (#373, self-review L1` | 1 つ目のセルの「trims every line before …」を、ブロックは字下げを落とした行を持つ(`resumeBlock.lines`)、手順 1 は生の行を項目の頭の判定だけに読む、に直した。Trigger に cycle 2 の注(払ったのは `e9017d7f` の変更に要った部分だけ) | V11・V12 |
| 574 `Test gaps of fix/resume-limit-later-line` | (a) を取り消し線にして `Closed in 7a7afdca`(codex の 4 行目の row で M4b が落ちる)。Impact と Trigger の (a) も取り消し線。Trigger の末尾に cycle 2 の注: (a) は払われ、(b) は `1042ccae`・`e9017d7f` が満たして払っていない(仕様の 54・61 行がまたコードに遅れた) | V15、test Run 2 の Test gap 1・4 |
| 575(新) | 既知の穴: prompt の中の空行より下の段落(P13)と、prompt の折り返しの行のうち `⎿` で始まる行(P4)は prompt のまとまりに入らず、普通のまとまりとして読まれる。base からある形で、`e9017d7f` が前文の最後の段落を「成り立つ範囲」に狭めた。Trigger は実機の描き方の取得 | self-review Run 2 の Tech debt の 2 行目、V19 |
| 576(新) | cycle 2 のテストの穴: (a) 1 桁目の項目の待ちの文の行は、コードと doc から出る答えを固定するだけで、実機の画面も Enter だけの文の行もない。(b) send の prompt のまとまりが項目の頭で切れないことは overlay の probe で 1 回確かめただけで、テストの行がない | test Run 2 の Test gaps 2・3 |

### 渡された所見の行き先

| 所見 | 行き先 |
|------|------|
| self-review Run 2 の M1・L1・L2 | `e9017d7f` で直った(コード)。台帳の Tech debt の 1 行目(M1 を直さない場合の行)は払われたので入れていない |
| self-review Run 2 の L3 (a)(b)(c) | (a) 台帳 570 行の数と測ったコミット、(b) 572 行の注、(c) 仕様 54 行(61 行も) |
| self-review Run 2 の Tech debt の 2 行目(P4・P13) | 台帳 575 行(当座の手当ては前文の最後の段落の書き直しで、行に書いた) |
| verify V9 | 仕様 54・61 行 |
| V10 | 台帳 570 行 |
| V11・V12 | 台帳 572 行の Trigger の注と 1 つ目のセル |
| V13 | 台帳 524 行((h) の数と注)と 488 行(注) |
| V14 | 台帳 544 行(偽になった「(a) and (b) were not met」を直した) |
| V15 | 台帳 574 行((a) を閉じ、(b) の繰り越し) |
| V16 | 台帳 462・483・567 行 |
| V17 | 台帳 490・517・542 行。V17 は `usageLimitRetryingLine` を「塊の中の補助」と書いたが、その関数は `usage_limit_screen_test.go` の 245 行目で、#353 の塊(`TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit` が 580 行目から)の外にある。542 行には「塊の外」と書いた |
| V18 | 仕様 54 行(挙動の変化の帰結) |
| V19 | 台帳 575 行と仕様 54 行の 1 文 |
| V20 | 計画の Progress(`1042ccae` の項目) |
| test Run 2 の Test gap 1(codex の M4b) | 台帳 574 の (a) を閉じた |
| Test gaps 2・3 | 台帳 576 行 |
| Test gap 4(仕様と rule の文を固定するテストがない) | 台帳 574 の (b) の注。テストは足していない(`/test` の範囲外、コードとテストは触らない決まり) |
| Test gap 5(P2 と fixture が実機でない) | 変わらない(台帳 571 行と 574 の (c)) |

### 掃き出し(cycle 2)

Trigger のセルを python で `(?<!\\) \| ` で切って 4 つ目のセルを読み、取り消し線の部分と日付つきの注を除いて、cycle 2 のコードが触った語(`resume_screens.go`、`resumeScreenBlocks`、`ClassifyResumeScreen`、`send_screen.go`、`sendScreenRegion`、`sendScreenBusyCheck`、`usage_limit_screen.go`、`usageLimitRetryingTurn`、`usageLimitRetryingLine`、`usageLimitTurnBlocks`、`ItemGlyphs`、`resumeBlock`、`prompt block`、`screen model` など 35 語)で照らした。23 行が語に当たり、Trigger の頭を読んで次のとおり分けた。`git diff -U0 b061c0c9..HEAD` の hunk を読み、hunk の見出しの関数名は使っていない。

- 当たった: 462・483・490・517・542・544・567・570・572・574(cycle 1 の注があるか、この branch の行)、488・524(V13)、543(どの report にもない行。`e9017d7f` が `usageLimitRetryingTurn` の本文の 1 行を変えた)
- 当たらない(理由): 194(Trigger は `idempotentRespawnDecision` など。`send-screen` は別の文脈)、455・456・457(Trigger は実機の報告か将来の取得で、`resume_screens.go` は書き足す場所として挙がるだけ)、466(`resumeCodexFooterLine`、取り消し線の `StopGlyphs`、`empty` の placeholder の規則は触っていない)、489(6 の `ClassifySendScreen` は関数の本文が変わっていない。`send_screen.go` の hunk はコメント 2 か所と `sendScreenBusyCheck` の呼び出し、`sendScreenRegion` の doc だけ)、492(`usageLimitUnreadNote`、`readUsageLimitScreen` は触っていない)、518(Trigger は CLI の更新と報告、`sendTypedMarkerLineBox` は触っていない)、529(`send_fallback.go` と `reseatProgressSentTask` は触っていない)、571(Trigger の `findResumeLimitResult` と `resumeResultUnderToolOrNotice` は cycle 2 の hunk にない)
- 自分の編集が当たる行: 仕様 `2026-11-07-org-seat-resume.md` を Trigger に持つ行は、`org-seat-resume` の語で見て新しく当たる行はなかった。台帳自体を Trigger に持つ行は、新しい行 2 つの書き方(関数とテストの名前、`file:line` なし)で足りている

### 確かめたもの

- `./scripts/check-sync.sh`: rc 0、PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。rule の 2 写し(`.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し)は `cmp` で同一
- `./scripts/run-static-verify.sh`: rc 0。`.txt` の fixture を分類できず full にフォールバックし、選ばれた pack は golang だけ。check-sync、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の plan 参照、gofmt(ok)、golangci-lint(0 issues)、branch の secret scan(`38932c13..56319352` clean)が通った。出力は `docs/evidence/verify-2026-11-22-095559.log`(git の管理外)
- `./scripts/plan-visual.sh digest`: `9683a080e69b`(編集の前後とも同じで、計画の `- Approved:` の行と一致)
- 台帳: `git show HEAD:` と比べて、変わった既存の行は 13 行(462、483、488、490、517、524、542、543、544、567、570、572、574)だけ。462・483・488・524・543・567・574 は挿入のみ、490・517・542(コミットの名前)、544(文の書き直し)、570(数)、572(1 つ目のセルの文)は意図した置き換え。新しい行 2 つを含めて 5 つのセル(`(?<!\\) \| ` で切った)、セルごとにバッククォートと `~~` の数が偶数、行末が ` |`。新しい行に `file:line` の参照はない
- コードの確かめ: 仕様 54・61 行の文は `resumeScreenBlocks` の本文(`resume_screens.go` の 743 行目付近、`resumeStartsWithAny(l, itemGlyphs)`)と前文の手順 1・手順 4 の (c2)・最後の段落を読んで書いた。台帳の `wc -l` は `git show <commit>:<file>` で 5 時点(origin/main、`d470a77a`、`1042ccae`、`e9017d7f`、HEAD)を測った

### 確かめていないもの

- Go のテスト、`run-test.sh`、変異の run(依頼の指示)。仕様と rule の文を固定するテストはないので(台帳 574 の (b))、Go のテストを回しても今回の文書の編集は確かめられない
- 台帳 570 行の「約 850・約 980」は関数とテストの範囲の行数から引いた見積もりで、移して `wc -l` を取った値ではない
- 実機の Claude Code の画面での手順 1 の切れ目と (c2) の判定、`ralph org resume` と見張りの実機での動き(test report と同じ)

### PR 本文の Known gaps に入れる点

Run 1 の一覧からの変更だけを書く。

1. 外れた項目: codex の行が「(c2) は claude だけ」を固定していない点(Run 1 の 4)。`7a7afdca` の 4 行目の codex の row で閉じた(台帳 574 の (a))
2. 足す項目: prompt の中の空行より下の段落と、prompt の折り返しの行のうち `⎿` で始まる行は、prompt のまとまりに入らず、普通のまとまりとして読まれる(base からの形。台帳 575、P4・P13)
3. 足す項目: 1 桁目の項目の待ちの文の行は実機の画面を見ていない。send の prompt のまとまりが項目の頭で切れないことにテストの行がない(台帳 576)
4. 挙動の変化: prompt のすぐ下(空行なし)の 1 桁目の `⏺ API Error: …` が、unknown から stopped になった(`resume` は続行の文を打つ)。実機の空行のある描き方と答えがそろう(仕様 54 行、V18)
5. サイズ: `resume_screens.go` は 935 行、`resume_screens_test.go` は 1,303 行(Run 1 は 902・1,119)。分けても約 850 行と約 980 行で、どちらも 800 を越える(台帳 570)
6. 残りは Run 1 のとおり: P2(台帳 571)、入れ子の `⎿`(572)、見ていない頭(573)、仕様と rule を読むテストがない・fixture は実機でない(574 の (b)(c))
7. 満たした Trigger を払わなかった行は、cycle 1 の 7 行(462・483・490・517・542・544・567)に 488・524・543・570・572・574 を足した 13 行(574 の (a) は払った)

### Insight event

`./scripts/insights-append.sh --slug resume-limit-later-line --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を回した。rc 0、`cycle` は 2。`docs/insights/events/2026-11-22-resume-limit-later-line.jsonl` は 8 行から 9 行になり、`git status --short` に迷子の jsonl はない(slug は日付なし)。

---

## Run 1 (cycle 1、履歴)

以下は cycle 1(HEAD `32440422` の木)の report で、見出しを 1 段下げたほかは元のまま。今の内容は上の `## Run 2 (cycle 2)`。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(承認の digest `9683a080e69b`。この run の前後とも同じ)。issue #373
- Branch: `fix/resume-limit-later-line`。run 開始時の HEAD は `32440422`(origin/main `38932c13` = #361 と #374 の merge を取り込んだ merge commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit): self-review `a3154042`(Merge: yes、MEDIUM 2、LOW 4。M1・L2・L3・L4 は `d470a77a` で直った。M2 は台帳なので今回、L1 は Known gap)、verify `836cb19f`(pass、LOW 4・INFO 4)、test `90d5d824`(pass、`run-test.sh` rc 0、変異 7 個のうち 6 個が落ち、codex の M4b が残った)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。回したのは `check-sync.sh`、`check-skill-sync.sh`、`plan-visual.sh digest`、狭い Go テスト(下の「確かめたもの」)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 19 ファイル、+1,177/-41。Go は全体で +680/-39、テスト以外の Go が +236/-34(`resume_screens.go` +224/-27、`usage_limit_screen.go` +12/-7)。walkthrough を書くかどうかは `/pr` が決める

### Summary

verify の引き継ぎ 5 項目と、test report の Test gap 1 を片づけた。台帳は、Trigger を満たした既存の 7 行に日付つきの繰り越しの注を足し(verify の 6 行に、掃き出しで見つけた行 567 を足した)、新しい行を 5 つ(大きさ、既知の穴 3 つ、テストの穴 1 つ)足した。仕様 `2026-11-07-org-seat-resume.md` は、verify の V2・V3・V4 が指した 59・60・61 行を直した。計画の Progress は S1 の古い名前と `d470a77a` の行を直し、`Sync-docs artifact created` の行を足して、AC1〜AC5 に `[x]` を付けた。任意の 2 点(rule の頭の例、`2026-08-01-org-runtime.md` の 1 文)も入れた。コードは触っていない。

### Changes made

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

### 渡された一覧の行き先

#### 1. 台帳の繰り越しの文

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

#### 2. 台帳の新しい行

| 行 | 内容 | 出どころ |
|------|------|------|
| 570 | 大きさ。`resume_screens.go` 705 → 902、`resume_screens_test.go` 781 → 1,119(self-review の 882・1,056 は `d470a77a` の前の数なので、両方書いた)。分け方は (c2) の 5 つの関数(約 90 行)を `resume_screens_limit.go` へ、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` と issue #373 の定数と `claudeTightResumeScreen`(約 260 行)を `resume_screens_limit_test.go` へ。移したあとも約 810 行と約 860 行で 800 を少し越える、と Impact に書いた(`sed` で関数の範囲を測った) | self-review M2 (e)、verify の引き継ぎ 2 |
| 571 | 既知の穴 P2: 空行のあとにまとめた行で始まる引用が、issue の画面 3 と区別できず stopped になる。self-review の行案 1 の「M1 を直さない場合は `⎿` から始まる 2 行以上の引用(P1)」は、M1 が `d470a77a` で直ったので落とした。Trigger の末尾で、返答の段落の描かれ方を知る同じ取得(箇条書きの返答つきの実機の画面)を、行 488 の (L6) と行 543 が求めていることに触れた | self-review の行案 1、verify V4 |
| 572 | 既知の穴: ツールの出力の中の入れ子の `⎿` の行(L1)と、出力の 1 行目が上限の文(計画の Risks の 2 つ目)。`resumeScreenBlocks` が行を trim して字下げを捨てることを原因に書いた | self-review の行案 2、L1 |
| 573 | 既知の穴: 見ていない頭(`Searched for …`、MCP、`Web Search(`)は unknown。行案 3 にあった「pane が狭く上限の文が割れた行」は、`d470a77a` の L2 で頭のわかる形は読めるようになったので、その旨に書き換えた | self-review の行案 3、計画の Risks の 1 つ目、verify の Coverage gaps |
| 574 | テストの穴(依頼の「この branch の行にテストの穴 1 を足す」): (a) codex の行が「(c2) は claude だけ」を挙動で固定せず、M4b が残る(overlay の probe の形も書いた)、(b) 仕様と rule の文を読むテストがない、(c) fixture は実機の pane の読み取りではなく、ブロックの間の空行は仮定。(b) と (c) は test report の Test gaps 3・4。Test gap 2 は行 571〜573 の穴で、この行の冒頭が名指しする | test の Test gaps 1・3・4、verify の Coverage gaps の最後の項 |

新しい行はどれも `file:line` の参照を持たず、関数とテストの名前で書いた。

#### 3. 仕様 `2026-11-07-org-seat-resume.md`

行番号は verify の値(59・61 行)が HEAD でも同じで、最初の語でも見つけた(59 行は `まとまりの 2 行目以降で`、61 行は `まとまりの先頭まで頭がないときは`)。P2 の 1 文は handoff の指示どおり 60 行(頭の形を言う項目)の末尾に足した。`resume_screens.go` の前文の (c2) の段落は `d470a77a` で直っていて、仕様の 3 文はそれと同じ条件を言う(「prompt の行のあとの最初の `⎿` の行で始まったまとまり」)。Open questions の補足(205 行)は、この問いの形が prompt のすぐ下の結果のまとまりであることを言うだけで、`d470a77a` のあとも合うので変えていない。

#### 4. 計画の Progress

S1 の行は `ResultHeads` を `ToolOrNoticeHeads` に直し、`2c418694` の時点の名前が `ResultHeads` で、`d470a77a` が改名したと書いた(記録は残した)。`d470a77a` の子の項目は S2 の次に置き、M1(`resultUnderPrompt`)、L2(つないだ文の照合)、L3(glyph の部分集合のテスト)、L4(改名 4 つ)と、残したもの(M2 は台帳、L1 と P2 は台帳の行と Known gaps)を書いた。この計画の Progress には sync-docs の項目がなかったので、ほかの計画の最近の形(`Test artifact created` の次)で `Sync-docs artifact created` の行を足した。AC は verify の表のとおり 5 つとも Met で、AC5 は仕様の 2 文を直し `check-sync.sh` が通ったあとに付けた。

#### 5. 任意の 2 点

rule の頭の例は「such as」にし、`Read 2 files` と `⏺ Monitor "..."` を足した(コードの `ToolOrNoticeHeads` と同じ 5 形のうち、`⏺ <ツール名>(` は `⏺ Bash(...)` で例示)。この段落は台帳の行 544 (c) の「次の変更」にも当たるので、注を足した(上の表)。`2026-08-01-org-runtime.md` は 89 行の末尾に日付つきの 1 文を足した(上の表)。

### 掃き出し(自分で全部の open な行の Trigger を読んだ)

Trigger のセルを python で `(?<!\\) \| ` で切って 4 つ目のセルを読み、全体が `~~` の行を飛ばし、この branch が触ったファイル・関数・文書(`resume_screens.go`、`ClassifyResumeScreen`、`UsageLimitScreen`、`usage_limit_screen*.go`、`send_screen_test.go`、`org_resume_test.go`、`testdata/resume`、`agent-messaging.md`、`usage_limited`、`org-seat-resume`、`org-runtime`、`StopGlyphs`、`sendScreenFixtureWants` など 38 語)で照らした。59 行が語に当たり、Trigger の頭を読んで次のとおり分けた。

- 当たった(注を足した): 462・483・490・517・542・544・567(上の表)。567 が handoff と verify にない行
- 当たらない(理由): 457(「最初の利用上限の止まり」。claude の 5 時間の文は #329 で `seen:` に済み、残る `docs:` の文と codex の 2 文の画面ではない)、464(`resumeOne`、`org_resume.go` の結果の行は触っていない。`org_resume_test.go` だけ)、465・466(`resumeCodexFooterLine`、`StopGlyphs`(取り消し線)、`empty` の placeholder の規則は触っていない)、479・488・489(`send_screen.go`、`send_fallback.go`、`status_live.go` は触っていない。488 の (L6) の Trigger は実機の箇条書きの返答つきの画面の取得で、この branch は取っていない)、491・492・493・494・495・507(`watch_usage_limit.go` は触っていない)、514・516・543(`usageLimitWaitingFooter`、`usageLimitRetryingTurn`、`claudeStatusLineStart` は触っていない)、566(`agent-messaging.md` は TASK_ID を足す変更の行き先として名指しされるだけで、Trigger は別)、88・118・182・267・358・362・370・420・441・442・447・448・470(`usage_limited` か `org-seat-resume` が古い日付の注の中に出るだけ)、455・456・463(Trigger は別の出来事で、`resume_screens.go` は行き先に挙がるだけ。455 は長い無音の報告か phase 2 の開始、456 は長い停電のあとの codex の座席、463 はリセットのあとの AC-10 の実機の確かめ)、453(全体が取り消し線で閉じている)
- 自分の編集が当たる行: `agent-messaging.md` の編集は 544(上で注を足した)。`2026-08-01-org-runtime.md` と `2026-11-07-org-seat-resume.md` の編集は、`org-runtime` と `org-seat-resume` を Trigger に持つ行を `grep` で全部見て、どれも古い日付の注の中の言及で、足す注はなかった。`docs/tech-debt/README.md` 自体を Trigger に持つ行は、新しい行 5 つの書き方(関数名で書く)で足りている

### 確かめたもの

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。rule の 2 写しは `cmp` で同一
- `./scripts/check-skill-sync.sh`: rc 0(skill は触っていない)
- `./scripts/plan-visual.sh digest`: `9683a080e69b`(編集の前後とも同じで、計画の `- Approved:` の行と一致)
- `go test -count=1 ./internal/org/ -run 'DocsMatch|AgentMessaging|Resume|UsageLimit'`: ok(7.4 s)。`go test -count=1 ./internal/cli/ -run 'Resume|DocsMatch'`: ok(16 s)。仕様と rule の編集で落ちる固定はなかった
- 台帳: 変えた行 7 つと新しい行 5 つが 5 つのセル(`(?<!\\) \| ` で切った)、セルごとにバッククォートと `~~` の数が偶数、行末が ` |`。`git show HEAD:` と比べて、変わった既存の行は 462・483・490・517・542・544・567 だけで、どれも挿入のみ(1 行ごとに difflib の操作が insert だけ)。新しい行に `file:line` の参照はない

### 確かめていないもの

- `./scripts/run-test.sh` と `run-verify.sh` は回していない(依頼の指示)
- 実機の Claude Code の画面での (c2) の判定、`ralph org resume` と見張りの実機での動き(test report と同じ)
- 台帳の新しい行の「移したあとの約 810 行・約 860 行」は関数とテストの範囲の行数から引いた見積もりで、移して `wc -l` を取った値ではない

### PR 本文の Known gaps に入れる点

1. P2: 返答が空行を置いてまとめた行(`Ran 1 shell command`)と `⎿` の上限の行を引用した画面は、issue の画面 3 と区別できず stopped と読む(台帳 571)。resume は続行の文を打ち、見張りは stall の代わりに usage_limited を上げる
2. ツールの出力の中の入れ子の `⎿` の上限の行と、出力の 1 行目が上限の文で手番が黙って終わった画面を stopped と読む(台帳 572、L1)
3. 見ていない頭(`Searched for …`、MCP、`Web Search(`)の下の上限は unknown で、resume は打たない(台帳 573)
4. codex の行が「(c2) は claude だけ」を挙動で固定していない。codex の表に `⎿` を包含つきで足す変異 M4b が残る(台帳 574 (a))
5. 仕様と rule の文を読むテストがない(574 (b))。fixture は実機の pane の読み取りではなく、ブロックの間の空行は仮定(574 (c))
6. `resume_screens.go`(902 行)と `resume_screens_test.go`(1,119 行)が 800 行を越えた(台帳 570)
7. 満たした Trigger を払わなかった行 7 つ(462・483・490・517・542・544・567)に繰り越しの注を足した

### Insight event

`./scripts/insights-append.sh --slug resume-limit-later-line --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を回した。rc 0。`docs/insights/events/2026-11-22-resume-limit-later-line.jsonl` は 3 行から 4 行になり、`git status --short` に迷子の jsonl はない(slug は日付なし)。
