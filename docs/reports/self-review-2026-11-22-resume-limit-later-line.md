# Self-review report: resume-limit-later-line

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `b263b1f3` から HEAD `444adf11` までの `git diff origin/main...HEAD`(15 ファイル)のうち、diff の質を見るもの。コードは `internal/org/resume_screens.go`(`findResumeLimitResult` と補助の 4 関数、表の 3 つの欄、`resumeBlock.lines`、`findResumeStop` の `resumeAfterGlyph` への置き換え)と `usage_limit_screen.go` のコメント。テストは `resume_screens_test.go`・`usage_limit_screen_test.go`・`send_screen_test.go`・`internal/cli/org_resume_test.go` と fixture 5 件。文書は仕様 FR-2 の改訂と rule の 2 写し。静的解析、仕様との突き合わせ、文書のずれは見ていない。テストは依頼にあった 2 本だけを走らせた

## Evidence reviewed

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

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | 頭のない結果のまとまりを許す条件が、prompt のすぐ下かどうかを見ていない。`resumeResultOfTool` は、頭が見つからないとき `above[0]` が `⎿` で始まれば true を返す。前文、関数の doc、仕様 FR-2 の改訂はこれを「prompt の行のすぐ下の結果のまとまり」と書く。しかし `resumeBlock` は prompt のすぐ下で切られたかを持たないので、空行のあとに `⎿` で始まるまとまりならどれでも通る。返答が空行を置いて座席の画面の `⎿` の行を 2 行以上引用し、そのまま手番を終えた画面(P1)は stopped になり、見張りも上限と読む。base では unknown だった。計画が codex の計画助言を受けて入れた守り(頭が返答の文の行なら数えない)が止めるのは、引用が返答の 1 行目のすぐ下に空行なしで続く形(AC3 のテスト 2 件)だけである。Claude Code が返答の段落やコードの囲みの前に空行を置くなら(未確認)、引用は P1 の形で描かれることが多い。空行のあとの引用がまとめた行で始まる形(P2)も stopped になるが、こちらは issue の画面 3 の実機の形(prompt、空行、`Read 2 files`)と区別できない。1 行だけの引用(P1b)は前から (c) で stopped なので、この PR で増えたのは 2 行以上の引用である。読み違えると、resume は普通に手番を終えた director や leader に続行の文を打つ | `resume_screens.go:817`(`return resumeStartsWithAny(above[0], table.ResultGlyphs)`)、前文 98 行目と `resumeResultOfTool` の doc の「a result block right under a prompt」、`docs/specs/2026-11-07-org-seat-resume.md:61`、`resumeScreenBlocks` の切れ目(`:696` の `afterPrompt && strings.HasPrefix(t, "⎿")`)、probe の P1・P2 | `resumeScreenBlocks` が「prompt のあとの最初の `⎿` の行」で切ったまとまりに印(たとえば `underPrompt bool`)を付け、頭がないときの許しをその印で決める。前文、doc、仕様の文はそのままで合う。P1 の形の否定のテストを 1 行足す。P2 は印では止まらないので、計画の Risks と台帳の Known gaps に「空行のあとでまとめた行から始まる引用」を足す |
| M2 | MEDIUM | maintainability | 台帳の Trigger を満たしたのに、繰り越しを 1 行も書いていない。(a) 行 542 の Trigger は「`usage_limit_screen_test.go` か `resume_test.go` の次の変更では、先に #353 の塊を `usage_limit_screen_retry_test.go` へ移す」。この PR は #353 の塊の中の `TestUsageLimitScreen_RetryWaitShapes` の 1 行を直し、テストを 1 つ足した(960 → 1013 行)。(b) 行 517 の Trigger「`usage_limit_screen_test.go` に次のテストを足すとき」も満たす(`TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`)。行が求める (a)〜(d) のテストは足していない。(c) 行 490 の Trigger「`send_screen_test.go` の次の変更」も満たす(885 → 893)。(d) 行 544 の (c) は「`agent-messaging.md` の usage_limited の段落を次に変えるとき、retry の場合を一覧のあとの別の文に移す(2 写し)」で、この PR はその段落を変えたが、移さずに括弧の中を長くした。(e) `resume_screens.go`(705 → 882)と `resume_screens_test.go`(781 → 1056)は、この PR で 800 行の目安を越えたのに行がない。(f) 計画の Risks の 1 つ目と 2 つ目は「Known gaps に書く」としているが、まだどこにも書いていない | 台帳の行 490・517・542・544 の Trigger のセル。`git diff --stat` に `docs/tech-debt/README.md` がない。上の `wc -l` | 各行の Trigger の末尾に「2026-11-22: fix/resume-limit-later-line (#373) hit this trigger with …, did not pay it: …. Carried over to the next change that meets this trigger」を足す。(e) は 2 ファイルの行を 1 つ足す。(f) は下の「Tech debt identified」の行案を使う。台帳の編集は records-only の例外に入らないので、sync-docs か直しのコミットで `/pr` の前に入れる |
| L1 | LOW | maintainability | 計画の Risks の 2 つ目は、誤読の範囲を「ツールの出力の 1 行目が上限の文で始まり(`cat` した文書など)」と書く。コードは、ツールの出力のうち字下げを除くと `⎿` で始まる行ならどれも結果の行として読む。leader や director が打つ `ralph org read` や `herdr pane read` で、上限に当たった claude の座席の pane を読むと、その座席の `⎿  You've hit …` の行が出力の中に入る。そのあと手番が黙って終われば stopped になり(P3)、`Line` は `… +5 lines (ctrl+o to expand)` で終わる。黙って終わることが前提なのは Risks の 2 つ目と同じだが、文は実際より狭い | `findResumeLimitResult`(`resume_screens.go:778`)は行を trim してから読む(`resumeScreenBlocks` の `strings.TrimSpace`)。probe の P3 | Known gaps の文に入れ子の形(pane の読み取りの出力)を足す。直すなら、生の行の字下げを `resumeBlock` に残して比べる手がある。issue と fixture の上限の行はどれも 2 桁目の `⎿` で、入れ子の行はそれより深い。ただし実機で確かめた形は 5 つだけ |
| L2 | LOW | maintainability | (c2) は上限の文を `⎿` の行 1 行だけで照合し、(c) はまとまりの行をつないだ文で照合する。pane が狭く(およそ 33 桁より狭い)上限の文が途中で割れると、(c2) は unknown になる(P7)。同じ割れ方でも prompt のすぐ下なら (c) が読む(P7b)。打たない向きなので安全側だが、doc にも仕様にも書いていない。issue の画面 2 は 49 桁ほどで `(Asia/Tokyo)` が折り返しており、狭い pane は実際にある | `resume_screens.go:785-786`(`resumeAfterGlyph(line, …)` と `resumeStartsWithAny(text, table.UsageLimit)`)。probe の P7・P7b | 上限の行とその下の行をつないだ文(`strings.Join(block.lines[i:], " ")`、`Line` と同じ文)で glyph のあとの文を照合する。`matchedStopSentence` もこの文を読むので食い違わない。直さないなら doc に 1 文足す |
| L3 | LOW | maintainability | `ResultGlyphs` は `StopGlyphs` に含まれる、という前提が暗黙になっている。`UsageLimitScreen` は (c2) の `Line` を `matchedStopSentence` に渡し、`matchedStopSentence` は `StopGlyphs` で glyph を落とす。`StopGlyphs` にない glyph が `ResultGlyphs` に入ると(codex の画面を見て `└` などを足したとき)、resume は stopped として打つが、見張りは上限と読まず stall を上げる。その driver の (c2) の fixture がないと、テストは落ちない | `usage_limit_screen.go:327`(`StopGlyphs: table.StopGlyphs`)、`ResultGlyphs` の doc(`resume_screens.go` の表の型) | `ResultGlyphs` の doc に「each is also one of StopGlyphs」と 1 文足し、`TestUsageLimitSentences_AreStoppedSentences` と同じ形の表のテストで固定する |
| L4 | LOW | naming | 名前が中身より狭い。`resumeResultOfTool` と `resumeItemHead` の 2 つ目の戻り値 `tool` は、知らせ(`Background command "`、`Monitor "`)でも true になる。`resumeResultOfTool` は頭のないまとまりでも true を返す。欄の名前 `ResultHeads` は結果の行の形ではなく項目の頭の形を持つので、grep で読むと「結果の行の頭」と取り違えやすい | `resume_screens.go:808`、`:827`、表の型の `ResultHeads` | 次にこの辺りを直すとき、`ToolOrNoticeHeads`、`resumeResultUnderToolOrNotice`、`(head, toolOrNotice bool)` のような名前にする。今回は許容 |

## Positive notes

- (c2) は (c) のあとに置かれ、(a)・(b)・(c) の答えは変わらない。API Error の文は今どおり候補の先頭でだけ数える(上限の下の `⎿  API Error: …` を unknown と固定するテストがある)
- 計画にない守り(上限の行より下に項目の頭があれば当てない)は、Design decisions の「最後の結果の行だけを見る」と同じ向きで、返答の行とまとめた行が下にある 2 つの否定のテストで固定している
- codex には (c2) を当てない。そのことを、`⎿` で始まる codex の画面を含む 3 件のテストで固定している。codex の表に `ResultGlyphs` を入れると、`⎿` の行の場合が落ちる
- 期待を直した既存の 2 件は意図を残している。まとまりがつながることは API Error で確かめ、prompt の下の上限は別の行にした
- `TestResumeScreenSentences_EveryEntryHasASource` が `ResultHeads` の出どころも見るようになった
- U+00A0 は、行頭の字下げ(`strings.TrimSpace`)と glyph のあと(`unicode.IsSpace`)の両方で通る(P5)
- 見ていないまとめた行と、空白を含むツールの頭(MCP の `(MCP)(`、`Web Search(`)は unknown に落ちる(P11・P11b)。取りこぼす向きで、今より悪くはならない
- CLI のテストは、dry run で打った記録と `resumed` の event がないことも確かめている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (c2) が返答の引用を上限と読む形が残る。空行のあとでまとめた行(`Ran 1 shell command`、`Read 2 files`)から始まる引用(P2)と、M1 を直さない場合は `⎿` から始まる 2 行以上の引用(P1) | 普通に手番を終えた director や leader に resume が続行の文を打つ。見張りは stall の代わりに usage_limited を上げる | 画面 3 の実機の形(prompt、空行、`Read 2 files`)と区別できない。区別には返答の描かれ方の実機の記録が要る | 上限でない座席に resume が打った報告、引用を含む usage_limited の件、または `findResumeLimitResult` の次の変更 | この report の M1、計画の Risks |
| (c2) が、ツールの出力の中にある入れ子の `⎿` の行(`ralph org read` の出力に入った別の座席の上限の行など)を、手番が黙って終わった画面で上限と読む(L1)。計画の Risks の 2 つ目(出力の 1 行目が上限の文)を広げたもの | 黙って終わった座席に続行の文を 1 行打ち、usage_limited を上げる | 黙って終わる手番はまれで、字下げで見分けるには生の行を残す変更が要る | 同じ形の誤読の報告、または `resumeBlock` か `findResumeLimitResult` の次の変更 | この report の L1、計画の Risks |
| 見ていないまとめた行(`Searched for …`、`Listed …`)と、名前に空白や記号を含むツールの頭(MCP の `<server> - <tool> (MCP)(`、`Web Search(`)の下の上限は、unknown のまま打たない(計画の Risks の 1 つ目)。pane が狭く上限の文が割れた行も同じ(L2) | 上限が解けても resume で続けられず、人か director が手で続ける | 実機の画面がない。打たない向きなので今より悪くはならない | その形の上限の画面を実機で見たとき。`ResultHeads` に足して fixture を置く | この report の L2、計画の Risks |
| `internal/org/resume_screens.go`(705 → 882 行)と `internal/org/resume_screens_test.go`(781 → 1056 行)が、この PR で 800 行の目安を越えた | ファイルが長く、画面モデルの前文と表と関数が 1 か所に集まって読みにくい | 計画の Scope は (c2) の追加で、分ける変更は入れていない | 次にどちらかを変えるとき。(c2) の関数群を `resume_screens_limit.go` に、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` と issue #373 の定数を `resume_screens_limit_test.go` に、中身を変えずに移す | この report の M2 (e) |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report、計画の Progress、insight event の 1 コミットで、ほかのレーンも台帳の末尾に追記するため衝突する。上の 4 行と、M2 の各行への繰り越しの文は、`/pr` の前に台帳へ入れてほしい(M2 は台帳の編集そのもの)。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 2、LOW 4)
- Follow-ups:
  - M1 は `resumeScreenBlocks` に印を 1 つ足すだけで直り、否定のテストを 1 行足せば固定できる。cross-review で再実行になるならそのときに直すのが安い。直さずに進めるなら、上の表の 1 行目に P1 も入れる
  - M2 は台帳の編集で、sync-docs か直しのコミットで入れる(`/pr` の前)。行 542 と 490 は、繰り越しではなく移動で払う手もある
  - L2 は 1 行の変更で直る(照合する文を `Line` と同じにする)。L3 は doc の 1 文と表のテスト 1 つ
  - 範囲外で見た点: ツールの頭の形 `^[A-Za-z][A-Za-z0-9_]*\(` は、返答の行 `⏺ Note(1): …` もツールの頭と読む(P13)。作為的な形で、所見にはしていない
