# Self-review report: resume-limit-later-line

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2)。今の判定はこの節の Recommendation の行で、Run 1 は見出しを 1 段下げて下に残す
- Scope: Run 1 の report のコミット `a3154042` のあとにこのブランチが足したコミット(HEAD `1042ccae`)。コードは `d470a77a`(Run 1 の M1・L2・L3・L4 の直し。どの self-review もまだ見ていない)と `1042ccae`(cross-review cycle 1 の ACTION_REQUIRED 1 の直し: `resumeScreenBlocks` が、prompt の行のあとで driver の `ItemGlyphs` で始まる行からも新しいまとまりを始める)。記録は `836cb19f`(verify)、`90d5d824`(test)、`1dda5f7f`(sync-docs: 仕様 2 本、rule の 2 写し、台帳、計画の Progress)、`b061c0c9`(triage)。`32440422` は origin/main `38932c13` の取り込み。静的解析、仕様との突き合わせ、文書のずれの点検はしていない。依頼で名指しされた仕様の 2 か所(54 行と 61 行)と、diff が触れた台帳の行だけ読んだ

### Evidence reviewed

- `git show d470a77a` と `git show 1042ccae` の全 hunk。HEAD の `resume_screens.go` で、画面モデルの前文、`ClassifyResumeScreen`、`resumeScreenBlocks`、`resumeMarkerItemGlyphs`、`findResumeLimitResult`、`resumeResultUnderToolOrNotice`、`resumeItemHead` を通して読んだ
- `resumeScreenBlocks` のほかの呼び出し元。`send_screen.go:608` の `sendScreenBusyCheck` が受け取る `region` は、`sendScreenRegion` が最後の prompt のまとまりより下から切り出した行で、prompt の行を含まない。footer にも prompt の行はない。どちらでも `afterPrompt` が真にならないので、項目の頭で切る新しい分岐は send の判定を変えない(下の probe でも、`ClassifySendScreen` の答えは 3 つの版で同じだった)。`usage_limit_screen.go:175` の `usageLimitRetryingTurn` は `usageLimitTurnBlocks` で最後の prompt のまとまりより下を今の手番とするので、prompt のすぐ下の `⏺` の行は今の手番に入る。実機の返答なら正しい向きで、M1 の形では引用が手番に入る(ただし retry の状態行が同じ時刻を持つときだけ上限と読む)
- `32440422`: main とこのブランチの両方が変えたファイルは `agent-messaging.md` の 2 写しだけで、`git show --cc` と `git show --remerge-diff` はどちらも空(手で解いた衝突はない)。2 写しは `cmp` で同一
- 記録のコミット(`836cb19f`、`90d5d824`、`b061c0c9`、`1dda5f7f`)の report にローカルのパス(`/Users/`、scratchpad)はない。d470a77a の改名の前の名前(`ResultHeads`、`resumeResultOfTool`、`resumeResultHead`、`resumeResultHeadForm`)は `internal/`、`docs/specs/`、`.claude/rules/` に残っていない
- 走らせたテスト: worktree で `go test -count=1 ./internal/org/ -run 'Resume|UsageLimit|SendScreen'`(ok、32.4 s)
- probe: HEAD、`d470a77a`、origin/main(base)を `git archive` で scratchpad に展開し、同じ probe のテストを置いて、claude の画面を `ClassifyResumeScreen` と `UsageLimitScreen` と `ClassifySendScreen` に通した。どの画面も入力欄は空で footer がある。worktree のファイルは触っていない

| probe | 入力欄より上の画面 | HEAD | d470a77a | base |
| --- | --- | --- | --- | --- |
| P1 | `❯ the reviewer seat showed this, what now?`、2 行目に字下げした `  ⏺ API Error: Connection error.`(座席の画面の行を写した prompt) | stopped | unknown(prompt_last) | unknown(prompt_last) |
| P2 | P1 の下に `  ⎿  Interrupted · What should Claude do instead?` | stopped | unknown | unknown |
| P9 | P2 の下に空行と `✻ Worked for 2s · done 3:40 PM` | stopped | unknown | unknown |
| P12 | `❯ The reviewer stopped with:`、`  ⏺ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`、`  ⎿  Interrupted …` | stopped(見張りも上限と読む) | unknown | unknown |
| P13 | `❯ The reviewer stopped with:`、空行、`  ⏺ API Error: Connection error.`、`  ⎿  Interrupted …`(prompt の 2 段落目) | stopped | stopped | stopped |
| P3 | `❯ the reviewer showed:`、`  ⏺ Bash(ralph org read reviewer)`、`    ⎿  You've hit your weekly limit …`、`  ⎿  Interrupted …` | unknown | stopped | stopped |
| P4 | P3 から `⎿  Interrupted` の行を除いた形 | stopped | stopped | stopped |
| P5 | `❯ say hi` のすぐ下(空行なし)に 1 桁目の `⏺ API Error: Can't reach the API server —` と折り返しの行 | stopped | unknown(prompt_last) | unknown(prompt_last) |
| P6 | `❯ what did the seat show?`、`⏺ The seat showed:`、`  ⎿  You've hit your weekly limit …`(1 行だけ) | unknown | stopped | stopped |
| P7 | `❯ say hi`、`  ⎿  You've hit your weekly limit …` | stopped | stopped | stopped |
| P8 | P1 の下に空行と `⏺ I will restart it.` | unknown | unknown | unknown |
| P10 | `❯ what did the seat show?`、`⏺ The seat showed:`、`  ⎿  API Error: Connection error.`、空行、状態行 | unknown | stopped | stopped |

- 実装者が意図した変更とした 3 つは表のとおり: P5(prompt のすぐ下の `⏺ API Error`)は stopped に、P6(返答の頭と 1 行の `⎿` の上限)は unknown になった。P10 は base から stopped だった誤読で、これも unknown になった。P5 は、空行のある実機の描き方(`docs/evidence/org-seat-resume-probe-2026-11-07.md:49-51`、(c) で stopped)と答えがそろうだけで、画面モデルの「prompt のまとまりは読まない」には反しない。`⏺` が 1 桁目にある限り、それは Claude Code が描いた項目で、利用者の文ではない。反するのは、3 つ目の「prompt の折り返しの行が `⏺` で始まる」場合で、下の M1 にした(P1・P2・P9・P12)

### Run 1 の所見の行き先

- M1: `d470a77a` で直った(`resumeBlock.resultUnderPrompt`。頭のない結果のまとまりの許しを、prompt の行のあとの最初の `⎿` の行で切ったまとまりに限った)。その印が、prompt のすぐ下に返答の頭が来る形を取りこぼしたのが cross-review の指摘で、`1042ccae` で直った(P6・P10 が unknown に)
- M2: `1dda5f7f` が、Trigger を満たした台帳の 7 行に繰り越しの注を足し、新しい行を 5 つ入れた(`docs/tech-debt/README.md` の 570〜574 行)
- L1: 台帳の 572 行(入れ子の `⎿` の行)
- L2: `d470a77a` で直った(上限の行とその下の行をつないだ文で照合し、`Line` と同じ文を読む)
- L3: `d470a77a` で直った(`ResultGlyphs` の doc と `TestResumeResultGlyphs_AreStopGlyphs`)
- L4: `d470a77a` で直った(`ToolOrNoticeHeads`、`resumeToolOrNoticeHead`、`resumeResultUnderToolOrNotice`、`toolOrNotice`)

### Findings (Run 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | `1042ccae` の新しい切れ目は、利用者の prompt の 2 行目以降が `⏺` で始まるときも prompt のまとまりを終わらせる。行は字下げを落としてから比べる(`t := strings.TrimSpace(l)`)ので、座席の画面の行を写した prompt の続きの行(`  ⏺ API Error: …`)と、Claude Code が 1 桁目に描く項目の頭を区別しない。P1・P2・P9・P12 は d470a77a でも base でも unknown だったが、HEAD では stopped になり、P12 は見張りも上限と読む。P2 は既存の fixture `claude-prompt-quotes-error-interrupted.txt`(引用を含む prompt のすぐ下に `⎿  Interrupted`。unknown と固定している)の引用を、`⏺` で始まる 1 行に置き換えただけの形である。前文の「A prompt block is read by none of (a) to (c2), so a sentence the user typed or quoted never counts」と、新しい step 1 の「A line that starts with an item glyph heads an item (a reply, a tool call, a notice), not the prompt's text」は、この形では成り立たない。こうした prompt は、座席が `ralph org send` で leader に送る短い RESULT や BLOCKED(短い貼り付けは畳まれずに履歴に出る)や、人が director に打つ文が座席の画面の行を写すと現れる。そのあと手番が中断されるか返答がないと、resume は続行の文を打ち、見張りは stall の代わりに usage_limited を上げる | `internal/org/resume_screens.go:728`(`afterPrompt && resumeStartsWithAny(t, itemGlyphs)`)、前文 `:39-41` と `:130-131`、probe の P1・P2・P9・P12、`internal/org/testdata/resume/claude-prompt-quotes-error-interrupted.txt`。prompt の続きの行は 2 桁目から始まる(`docs/evidence/org-seat-resume-live-2026-11-07.md:88-90`)。項目の頭は 1 桁目にある(`docs/evidence/org-seat-resume-probe-2026-11-07.md:51`)。短い貼り付けが畳まれずに出ることは `docs/evidence/org-send-paste-probe-2026-11-13.md:90`(S1) | 項目の頭は、右の空白だけを落とした生の行 `l` が glyph で始まるとき(字下げなし)に限る: `resumeStartsWithAny(l, itemGlyphs)`。scratchpad の写しでこの 1 語を変えると、P1・P2・P9・P12 は unknown に戻り、P5・P6・P10 と、この PR のテスト(cross-review の 2 行、ツールの頭の 1 行、`TestResumeScreenBlocks_ItemHeadUnderAPrompt`)は今の答えのまま、`-run 'Resume\|UsageLimit\|SendScreen'` も通った(archive に `scripts/` がないため落ちる cleanup のテスト 1 つを除く)。P3 は base と同じ stopped に戻る(下の Tech debt の行の形)。P2 の形の否定のテストを `TestResumeScreenBlocks_ItemHeadUnderAPrompt` と fixture の表に 1 行ずつ足す。前文 `:130-131` の文は、成り立つ範囲に狭める。直さずに進めるなら、P1・P2・P12 を下の Tech debt の 1 行目として台帳と `/pr` の Known gaps に入れる |
| L1 | LOW | maintainability | `send_screen.go` の 2 か所のコメントが、send の prompt のまとまりを resume のまとまりと同じだと書く。画面モデルの 3 の「up to a blank line or a ⎿ line, as in resume's blocks」と、`sendScreenRegion` の doc の「which resume's blocks also set apart (step 1 of its screen model)」である。`1042ccae` のあと resume は prompt のすぐ下の `⏺` の行でも切るが、`sendScreenRegion` は空行と `⎿` でしか切らない。空行なしで `❯ hi`、`⏺ Hi!`、`✻ Worked …` と続く画面では、resume は prompt のまとまりを最後と見ないが、send は region を空と見て prompt_last の unknown にする(P5 で send は prompt_last)。send が打たない向きなので害はないが、コメントが食い違いを隠す | `internal/org/send_screen.go:37-39`、`:655-659`、`:666`(`t == "" \|\| strings.HasPrefix(t, "⎿")`) | 2 か所に「resume と違い項目の頭では切らない(send はその画面を unknown にして打たない)」と書く。`sendScreenRegion` を同じ規則にするのはコードとテストの変更なので、今回はコメントで足りる |
| L2 | LOW | maintainability | `resumeMarkerItemGlyphs` は marker から driver を逆に引く(`resumeScreenSentences` の map を回し、`resumeMarker(driver) == marker` の表の `ItemGlyphs` を返す)。`resumeScreenBlocks` の呼び出し元 5 か所(テストを含む)はどれも driver を持っていて、`resumeMarker(driver)` で marker を作って渡している。逆引きは marker と driver が 1 対 1 であることを前提にし、2 つの driver が同じ marker を持つと、map の順でどちらの表を返すかが変わる。`resumeMarker` は codex 以外のすべての driver に `❯` を返す | `internal/org/resume_screens.go:743-750`、呼び出し元 `:444`、`:655`、`send_screen.go:608`、`usage_limit_screen.go:175`、`resume_screens_test.go:1015` | `resumeScreenBlocks(lines, driver)` にして、中で `resumeMarker(driver)` と `resumeScreenSentences[driver].ItemGlyphs` を引く。そうすれば `resumeMarkerItemGlyphs` は要らなくなる |
| L3 | LOW | maintainability | `1042ccae` で古くなった記録が 3 つある(cycle 2 の sync-docs 向け)。(a) 台帳の 570 行は行数を「705 → 902」「781 → 1,119」とし、`d470a77a` を「the last Go commit of the branch」と書く。HEAD では 932 行と 1,205 行で、最後の Go のコミットは `1042ccae` である。(b) 572 行の Trigger「the next change to `resumeScreenBlocks` or `findResumeLimitResult`」を `1042ccae` が満たした(`resumeScreenBlocks` を変えた)が、払ってもいないし繰り越しの注もない。(c) 573 行の (b) が予告したとおり、仕様の 54 行(2026-11-07 の改訂)の「prompt のあとの最初の `⎿` の行でも区切る」は、prompt のすぐ下の `⏺` の行でも切る今の step 1 を言わない。61 行の「prompt の行のあとの最初の `⎿` の行で始まったもの」は今も正しいが、あいだに項目の頭がないことを言わない | `docs/tech-debt/README.md:570`、`:572`、`:573`、`docs/specs/2026-11-07-org-seat-resume.md:54`、`:61`、`wc -l`(HEAD: 932 と 1,205) | cycle 2 の sync-docs で (a) の数を直し、(b) に繰り越しの注を足し、(c) の 54 行に 1 文足す。M1 を直すなら、字下げの条件も同じ文に入れる。(b) は M1 の直しで一部を払える(切れ目で生の行を読む)が、572 行が求める `resumeBlock.lines` への字下げの保存は M1 の直しには要らない |

### Positive notes (Run 2)

- `1042ccae` は、cross-review が見つけた空行のない形を、空行のある形(既存の否定のテスト)とそろえた。返答の頭が prompt のすぐ下に来ても空行のあとに来ても、引用の `⎿` の行は返答のまとまりに入り、(c2) は頭の形で決める。base から stopped だった P10(返答の頭の下の `⎿  API Error`)も unknown になった
- codex には `ItemGlyphs` がないので切れ目は変わらない。`TestResumeScreenBlocks_ItemHeadUnderAPrompt` がそれを codex の行で固定している
- そのテストは `lines`、`text`、`prompt`、`resultUnderPrompt` の 4 つを比べる。`⎿` で始まるまとまりの下の `⏺` の行が同じまとまりに入ること(flush で `afterPrompt` が落ちること)も固定している
- `resultUnderPrompt` と `ItemGlyphs` の doc は、新しい切れ目を言うように直してある。前文の step 4 (c2) にも、返答の頭の下の `⎿` の行は頭が決めることを 1 文足してある
- `d470a77a` の `findResumeLimitResult` は、照合の文と `Line` に同じ文(`strings.Join(block.lines[i:], " ")`)を使うので、`matchedStopSentence` と食い違わない

### Tech debt identified (Run 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (M1 を直さない場合)prompt の続きの行が `⏺` で始まると、その行から下が prompt のまとまりを出る(P1・P2・P9・P12)。`1042ccae` が足した切れ目は字下げを見ない | 座席の画面の行を写した prompt のあと手番が中断されるか返答がないと、resume は続行の文を打ち、見張りは usage_limited を上げる | cycle 2 はパイプラインの上限で、直すとコードの変更になり再実行が要る | 引用を含む prompt の座席への `resumed` か usage_limited の件、または `resumeScreenBlocks` の次の変更: 項目の頭を `l`(字下げなし)で判定し、P2 の形の否定のテストを足す | この report の Run 2 の M1 |
| base から残っている形: prompt の中の空行より下の段落(P13)と、prompt の続きの行のうち `⎿` で始まる行(P4)は、prompt のまとまりに入らない。前文 `:130-131` の「a sentence the user typed or quoted never counts」はこれを言わない | P13 の形(座席の画面の `⏺ API Error` を 2 段落目に写した prompt のあと中断)は stopped になり、resume が打つ | この PR より前からある形で、計画の Scope の外。prompt の続きの行も結果の行も 2 桁目から始まるので、字下げでは `⎿` の行を分けられない | 同じ形の `resumed` の報告、または `resumeScreenBlocks` の次の変更: Claude Code が prompt の中の空行をどう描くかを実機で見て、段落の続きを prompt のまとまりに残す規則を決める。それまで前文の文を成り立つ範囲に狭める | この report の Run 2 の probe P4・P13 |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report、計画の Progress、insight event の 1 コミットだからである。上の 2 行と L3 の直しは、cycle 2 の sync-docs で台帳に入れてほしい。_

### Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 3)
- Follow-ups:
  - M1 は 1 語の変更(`t` → `l`)と否定のテスト 2 行で直る。cycle 2 はパイプラインの上限なので、上限を上げて直すか、Known gaps に入れて進むかを orchestrator が決める。直さないなら Tech debt の 1 行目を台帳に入れる
  - L1 はコメント 2 か所、L2 は呼び出し元 5 か所の引数の変更で、どちらも次にこのあたりを変えるときでよい
  - L3 と Tech debt の 2 行は cycle 2 の sync-docs で入れる

## Run 1 (cycle 1)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `b263b1f3` から HEAD `444adf11` までの `git diff origin/main...HEAD`(15 ファイル)のうち、diff の質を見るもの。コードは `internal/org/resume_screens.go`(`findResumeLimitResult` と補助の 4 関数、表の 3 つの欄、`resumeBlock.lines`、`findResumeStop` の `resumeAfterGlyph` への置き換え)と `usage_limit_screen.go` のコメント。テストは `resume_screens_test.go`・`usage_limit_screen_test.go`・`send_screen_test.go`・`internal/cli/org_resume_test.go` と fixture 5 件。文書は仕様 FR-2 の改訂と rule の 2 写し。静的解析、仕様との突き合わせ、文書のずれは見ていない。テストは依頼にあった 2 本だけを走らせた

### Evidence reviewed

- `git diff origin/main...HEAD` の全 hunk。あわせて `resume_screens.go` の画面モデルの前文、`ClassifyResumeScreen`、`resumeScreenBlocks`、`findResumeStop` の全文、`usage_limit_screen.go` の `UsageLimitScreen`・`matchedStopSentence`・`usageLimitRetryingTurn`・`usageLimitResetsIn`、`resume.go` の `resumeLook`・`resumeWorkingReason` を読んだ
- `findResumeStop` の置き換え: 前の形(glyph ごとに `CutPrefix` し、最初に当たった glyph で `break`)と `resumeAfterGlyph`(最初に当たった glyph で `return`)は同じ答えを返す
- `resumeBlock.lines` は flush で `cur` を渡したあと `cur = nil` にするので、次のまとまりと backing array を共有しない。`resumeBlock{}`(候補なし)と `matchedStopSentence` の `resumeBlock{text: block}` は `lines` が nil で、`findResumeLimitResult` のループは回らない。`resumeResultOfTool` の `above[0]` は、呼び出し元の `i >= 1` があるので空にならない
- 見張りが (c2) の `Line` を読めるか: `Line` は `⎿`、空白、`UsageLimit` の文の順で始まる。`matchedStopSentence` は claude の `StopGlyphs`(`⏺`・`⎿`)で読むので一致する。見張りの行は `usageLimitLine` を通る。herdr が working と報告する座席に resume は打たない(`resumeLook` の `case "working"`)ので、(c2) が打つ経路を増やすのは idle と done の座席だけ
- 走らせたテスト: `go test -count=1 ./internal/org/ -run 'Resume|UsageLimit|SendScreen'`(ok、34.1s)と `go test -count=1 ./internal/cli/ -run Resume`(ok、17.1s)
- scratchpad に HEAD と origin/main を `git archive` で展開し、両方に probe のテストを 1 つずつ置いて、14 の画面を `ClassifyResumeScreen` と `UsageLimitScreen` に通した。結果は次の表のとおり(base は origin/main での答え)

| probe | 画面 | HEAD | base |
| --- | --- | --- | --- |
| P1 | 返答 `⏺ The seat showed:`、空行、`⎿  Read notes.md` と `⎿  You've hit your weekly limit · resets 3pm` | stopped(見張りも上限と読む) | unknown |
| P1b | P1 の引用が `⎿` の 1 行だけ | stopped | stopped(既存の (c)) |
| P2 | 返答、空行、`Ran 1 shell command` と上限の `⎿` の行 | stopped | unknown |
| P3 | `⏺ Bash(ralph org read fx-rate/leader --lines 8)` の出力の中に入れ子の `⎿  You've hit your 5-hour limit …` と `… +5 lines (ctrl+o to expand)` があり、手番は黙って終わる | stopped(Line の終わりは `… +5 lines (ctrl+o to expand)`) | unknown |
| P4 | `⏺ Bash(cat notes.md)` の出力の 1 行目が上限の文で、手番は黙って終わる | stopped | unknown |
| P5 | 行頭の字下げと `⎿` のあとが U+00A0 | stopped | unknown |
| P7 | ツールの下で、上限の文が pane の幅で `You've hit your` と `5-hour limit · resets 4:40am` に割れる | unknown | unknown |
| P7b | P7 と同じ割れ方で、prompt のすぐ下 | stopped | stopped |
| P11 | MCP のツールの頭 `⏺ context7 - resolve-library-id (MCP)(…)` | unknown | unknown |
| P11b | `⏺ Web Search("go regexp")` | unknown | unknown |
| P12 | `Read 1 file, ran 2 shell commands` | stopped | unknown |
| P13 | 返答 `⏺ Note(1): the seat showed` のすぐ下の上限の `⎿` の行 | stopped | unknown |
| P14 | 上限の行の下に `… +3 lines (ctrl+o to expand)` | stopped | unknown |
| P15 | まとまりが印のない出力の続きで始まる(画面の上が切れた形) | unknown | unknown |

- `docs/tech-debt/README.md` を diff が触れた名前(`resume_screens.go`、`UsageLimitScreen`、`usage_limit_screen_test.go`、`send_screen_test.go`、`agent-messaging.md` の usage_limited の段落)で引き、Trigger のセルを `python3` の `re.split(r'(?<!\\)\|', 行)[4]` で読んだ。diff に台帳の hunk はない
- `wc -l`(origin/main → HEAD): `resume_screens.go` 705 → 882、`resume_screens_test.go` 781 → 1056、`usage_limit_screen_test.go` 960 → 1013、`send_screen_test.go` 885 → 893、`internal/cli/org_resume_test.go` 743 → 783
- rule の 2 写しは `cmp` で同一

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | 頭のない結果のまとまりを許す条件が、prompt のすぐ下かどうかを見ていない。`resumeResultOfTool` は、頭が見つからないとき `above[0]` が `⎿` で始まれば true を返す。前文、関数の doc、仕様 FR-2 の改訂はこれを「prompt の行のすぐ下の結果のまとまり」と書く。しかし `resumeBlock` は prompt のすぐ下で切られたかを持たないので、空行のあとに `⎿` で始まるまとまりならどれでも通る。返答が空行を置いて座席の画面の `⎿` の行を 2 行以上引用し、そのまま手番を終えた画面(P1)は stopped になり、見張りも上限と読む。base では unknown だった。計画が codex の計画助言を受けて入れた守り(頭が返答の文の行なら数えない)が止めるのは、引用が返答の 1 行目のすぐ下に空行なしで続く形(AC3 のテスト 2 件)だけである。Claude Code が返答の段落やコードの囲みの前に空行を置くなら(未確認)、引用は P1 の形で描かれることが多い。空行のあとの引用がまとめた行で始まる形(P2)も stopped になるが、こちらは issue の画面 3 の実機の形(prompt、空行、`Read 2 files`)と区別できない。1 行だけの引用(P1b)は前から (c) で stopped なので、この PR で増えたのは 2 行以上の引用である。読み違えると、resume は普通に手番を終えた director や leader に続行の文を打つ | `resume_screens.go:817`(`return resumeStartsWithAny(above[0], table.ResultGlyphs)`)、前文 98 行目と `resumeResultOfTool` の doc の「a result block right under a prompt」、`docs/specs/2026-11-07-org-seat-resume.md:61`、`resumeScreenBlocks` の切れ目(`:696` の `afterPrompt && strings.HasPrefix(t, "⎿")`)、probe の P1・P2 | `resumeScreenBlocks` が「prompt のあとの最初の `⎿` の行」で切ったまとまりに印(たとえば `underPrompt bool`)を付け、頭がないときの許しをその印で決める。前文、doc、仕様の文はそのままで合う。P1 の形の否定のテストを 1 行足す。P2 は印では止まらないので、計画の Risks と台帳の Known gaps に「空行のあとでまとめた行から始まる引用」を足す |
| M2 | MEDIUM | maintainability | 台帳の Trigger を満たしたのに、繰り越しを 1 行も書いていない。(a) 行 542 の Trigger は「`usage_limit_screen_test.go` か `resume_test.go` の次の変更では、先に #353 の塊を `usage_limit_screen_retry_test.go` へ移す」。この PR は #353 の塊の中の `TestUsageLimitScreen_RetryWaitShapes` の 1 行を直し、テストを 1 つ足した(960 → 1013 行)。(b) 行 517 の Trigger「`usage_limit_screen_test.go` に次のテストを足すとき」も満たす(`TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`)。行が求める (a)〜(d) のテストは足していない。(c) 行 490 の Trigger「`send_screen_test.go` の次の変更」も満たす(885 → 893)。(d) 行 544 の (c) は「`agent-messaging.md` の usage_limited の段落を次に変えるとき、retry の場合を一覧のあとの別の文に移す(2 写し)」で、この PR はその段落を変えたが、移さずに括弧の中を長くした。(e) `resume_screens.go`(705 → 882)と `resume_screens_test.go`(781 → 1056)は、この PR で 800 行の目安を越えたのに行がない。(f) 計画の Risks の 1 つ目と 2 つ目は「Known gaps に書く」としているが、まだどこにも書いていない | 台帳の行 490・517・542・544 の Trigger のセル。`git diff --stat` に `docs/tech-debt/README.md` がない。上の `wc -l` | 各行の Trigger の末尾に「2026-11-22: fix/resume-limit-later-line (#373) hit this trigger with …, did not pay it: …. Carried over to the next change that meets this trigger」を足す。(e) は 2 ファイルの行を 1 つ足す。(f) は下の「Tech debt identified」の行案を使う。台帳の編集は records-only の例外に入らないので、sync-docs か直しのコミットで `/pr` の前に入れる |
| L1 | LOW | maintainability | 計画の Risks の 2 つ目は、誤読の範囲を「ツールの出力の 1 行目が上限の文で始まり(`cat` した文書など)」と書く。コードは、ツールの出力のうち字下げを除くと `⎿` で始まる行ならどれも結果の行として読む。leader や director が打つ `ralph org read` や `herdr pane read` で、上限に当たった claude の座席の pane を読むと、その座席の `⎿  You've hit …` の行が出力の中に入る。そのあと手番が黙って終われば stopped になり(P3)、`Line` は `… +5 lines (ctrl+o to expand)` で終わる。黙って終わることが前提なのは Risks の 2 つ目と同じだが、文は実際より狭い | `findResumeLimitResult`(`resume_screens.go:778`)は行を trim してから読む(`resumeScreenBlocks` の `strings.TrimSpace`)。probe の P3 | Known gaps の文に入れ子の形(pane の読み取りの出力)を足す。直すなら、生の行の字下げを `resumeBlock` に残して比べる手がある。issue と fixture の上限の行はどれも 2 桁目の `⎿` で、入れ子の行はそれより深い。ただし実機で確かめた形は 5 つだけ |
| L2 | LOW | maintainability | (c2) は上限の文を `⎿` の行 1 行だけで照合し、(c) はまとまりの行をつないだ文で照合する。pane が狭く(およそ 33 桁より狭い)上限の文が途中で割れると、(c2) は unknown になる(P7)。同じ割れ方でも prompt のすぐ下なら (c) が読む(P7b)。打たない向きなので安全側だが、doc にも仕様にも書いていない。issue の画面 2 は 49 桁ほどで `(Asia/Tokyo)` が折り返しており、狭い pane は実際にある | `resume_screens.go:785-786`(`resumeAfterGlyph(line, …)` と `resumeStartsWithAny(text, table.UsageLimit)`)。probe の P7・P7b | 上限の行とその下の行をつないだ文(`strings.Join(block.lines[i:], " ")`、`Line` と同じ文)で glyph のあとの文を照合する。`matchedStopSentence` もこの文を読むので食い違わない。直さないなら doc に 1 文足す |
| L3 | LOW | maintainability | `ResultGlyphs` は `StopGlyphs` に含まれる、という前提が暗黙になっている。`UsageLimitScreen` は (c2) の `Line` を `matchedStopSentence` に渡し、`matchedStopSentence` は `StopGlyphs` で glyph を落とす。`StopGlyphs` にない glyph が `ResultGlyphs` に入ると(codex の画面を見て `└` などを足したとき)、resume は stopped として打つが、見張りは上限と読まず stall を上げる。その driver の (c2) の fixture がないと、テストは落ちない | `usage_limit_screen.go:327`(`StopGlyphs: table.StopGlyphs`)、`ResultGlyphs` の doc(`resume_screens.go` の表の型) | `ResultGlyphs` の doc に「each is also one of StopGlyphs」と 1 文足し、`TestUsageLimitSentences_AreStoppedSentences` と同じ形の表のテストで固定する |
| L4 | LOW | naming | 名前が中身より狭い。`resumeResultOfTool` と `resumeItemHead` の 2 つ目の戻り値 `tool` は、知らせ(`Background command "`、`Monitor "`)でも true になる。`resumeResultOfTool` は頭のないまとまりでも true を返す。欄の名前 `ResultHeads` は結果の行の形ではなく項目の頭の形を持つので、grep で読むと「結果の行の頭」と取り違えやすい | `resume_screens.go:808`、`:827`、表の型の `ResultHeads` | 次にこの辺りを直すとき、`ToolOrNoticeHeads`、`resumeResultUnderToolOrNotice`、`(head, toolOrNotice bool)` のような名前にする。今回は許容 |

### Positive notes

- (c2) は (c) のあとに置かれ、(a)・(b)・(c) の答えは変わらない。API Error の文は今どおり候補の先頭でだけ数える(上限の下の `⎿  API Error: …` を unknown と固定するテストがある)
- 計画にない守り(上限の行より下に項目の頭があれば当てない)は、Design decisions の「最後の結果の行だけを見る」と同じ向きで、返答の行とまとめた行が下にある 2 つの否定のテストで固定している
- codex には (c2) を当てない。そのことを、`⎿` で始まる codex の画面を含む 3 件のテストで固定している。codex の表に `ResultGlyphs` を入れると、`⎿` の行の場合が落ちる
- 期待を直した既存の 2 件は意図を残している。まとまりがつながることは API Error で確かめ、prompt の下の上限は別の行にした
- `TestResumeScreenSentences_EveryEntryHasASource` が `ResultHeads` の出どころも見るようになった
- U+00A0 は、行頭の字下げ(`strings.TrimSpace`)と glyph のあと(`unicode.IsSpace`)の両方で通る(P5)
- 見ていないまとめた行と、空白を含むツールの頭(MCP の `(MCP)(`、`Web Search(`)は unknown に落ちる(P11・P11b)。取りこぼす向きで、今より悪くはならない
- CLI のテストは、dry run で打った記録と `resumed` の event がないことも確かめている

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (c2) が返答の引用を上限と読む形が残る。空行のあとでまとめた行(`Ran 1 shell command`、`Read 2 files`)から始まる引用(P2)と、M1 を直さない場合は `⎿` から始まる 2 行以上の引用(P1) | 普通に手番を終えた director や leader に resume が続行の文を打つ。見張りは stall の代わりに usage_limited を上げる | 画面 3 の実機の形(prompt、空行、`Read 2 files`)と区別できない。区別には返答の描かれ方の実機の記録が要る | 上限でない座席に resume が打った報告、引用を含む usage_limited の件、または `findResumeLimitResult` の次の変更 | この report の M1、計画の Risks |
| (c2) が、ツールの出力の中にある入れ子の `⎿` の行(`ralph org read` の出力に入った別の座席の上限の行など)を、手番が黙って終わった画面で上限と読む(L1)。計画の Risks の 2 つ目(出力の 1 行目が上限の文)を広げたもの | 黙って終わった座席に続行の文を 1 行打ち、usage_limited を上げる | 黙って終わる手番はまれで、字下げで見分けるには生の行を残す変更が要る | 同じ形の誤読の報告、または `resumeBlock` か `findResumeLimitResult` の次の変更 | この report の L1、計画の Risks |
| 見ていないまとめた行(`Searched for …`、`Listed …`)と、名前に空白や記号を含むツールの頭(MCP の `<server> - <tool> (MCP)(`、`Web Search(`)の下の上限は、unknown のまま打たない(計画の Risks の 1 つ目)。pane が狭く上限の文が割れた行も同じ(L2) | 上限が解けても resume で続けられず、人か director が手で続ける | 実機の画面がない。打たない向きなので今より悪くはならない | その形の上限の画面を実機で見たとき。`ResultHeads` に足して fixture を置く | この report の L2、計画の Risks |
| `internal/org/resume_screens.go`(705 → 882 行)と `internal/org/resume_screens_test.go`(781 → 1056 行)が、この PR で 800 行の目安を越えた | ファイルが長く、画面モデルの前文と表と関数が 1 か所に集まって読みにくい | 計画の Scope は (c2) の追加で、分ける変更は入れていない | 次にどちらかを変えるとき。(c2) の関数群を `resume_screens_limit.go` に、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` と issue #373 の定数を `resume_screens_limit_test.go` に、中身を変えずに移す | この report の M2 (e) |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report、計画の Progress、insight event の 1 コミットで、ほかのレーンも台帳の末尾に追記するため衝突する。上の 4 行と、M2 の各行への繰り越しの文は、`/pr` の前に台帳へ入れてほしい(M2 は台帳の編集そのもの)。_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 2、LOW 4)
- Follow-ups:
  - M1 は `resumeScreenBlocks` に印を 1 つ足すだけで直り、否定のテストを 1 行足せば固定できる。cross-review で再実行になるならそのときに直すのが安い。直さずに進めるなら、上の表の 1 行目に P1 も入れる
  - M2 は台帳の編集で、sync-docs か直しのコミットで入れる(`/pr` の前)。行 542 と 490 は、繰り越しではなく移動で払う手もある
  - L2 は 1 行の変更で直る(照合する文を `Line` と同じにする)。L3 は doc の 1 文と表のテスト 1 つ
  - 範囲外で見た点: ツールの頭の形 `^[A-Za-z][A-Za-z0-9_]*\(` は、返答の行 `⏺ Note(1): …` もツールの頭と読む(P13)。作為的な形で、所見にはしていない
