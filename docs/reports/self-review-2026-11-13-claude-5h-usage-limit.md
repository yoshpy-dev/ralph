# Self-review report: claude-5h-usage-limit

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(issue #329)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目
- Scope: `git diff origin/main...HEAD`(base `bbe2807d`、HEAD `a851a653`)。15 ファイル、684 行追加、33 行削除。Go は本体 2 ファイル(`resume_screens.go` と `usage_limit_screen.go`)とテスト 4 ファイル、fixture 1 つ。残りは文書、計画、証拠。コミットは 4 つ(`c610109f` 計画、`fddda578` 本体とテスト、`6a8977ca` 文書、`a851a653` 計画の進捗)で、すべて見た。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは `/verify` と `/test` と `/sync-docs` の側

## Evidence reviewed

読んだもの。

- `internal/org/usage_limit_screen.go` の全部(`UsageLimitScreen`、新しい `usageLimitWaitingFooter`、`matchedStopSentence`、`usageLimitLine`)と、`internal/org/resume_screens.go` の全部(画面の model の説明、表、`ClassifyResumeScreen`、`findResumeInputBox` から `resumeScreenBlocks`、`findResumeSentence`)
- テストの差分の全部: `usage_limit_screen_test.go`(`usageLimitFooterLine`、`claudeFooterScreen`、`usageLimitQuotedWaitScreen`、`TestUsageLimitScreen_WaitingForALimitToReset`、`TestUsageLimitWaitingSentences_AreWaitingSentences`、`TestUsageLimitScreen_EveryFixture` の書き換え)、`watch_usage_limit_test.go` の 2 つの新しいテスト、`resume_screens_test.go` と `send_screen_test.go` の表の行。テストの助けの関数 `claudeResumeScreen` と `countString`
- fixture `claude-5h-limit-continuing.txt` を `cat -A` で全行。`shasum -a 256` が証拠の文書の値(`2b9742d9…8dd27`)と一致し、23 行で、先頭 2 行が空行。U+00A0 が 8 行目の `⎿` のあとにある
- `docs/evidence/claude-5h-limit-screen-2026-11-13.md` の全部。secret、token、メール、`/Users/` のパス、`$HOME` を `grep` で探して 0 件
- 文書: `docs/specs/2026-08-01-org-runtime.md` の新しい項(a から c)、`docs/specs/2026-11-07-org-seat-resume.md` の 2 行、`.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し(`cmp` で同じ)、`AGENTS.md` の差分(語単位)、`docs/tech-debt/README.md` の 452 行の差分(語単位)と新しい 495 行(区切りの数 5、セルの中に生の `|` なし)
- 比べた先: `watch_usage_limit.go` の文字列と doc comment、`watch.go` の `evaluateSeat`(`if stalled {` の 1324 行)、`internal/cli/org_watch.go` の `--help`、`.claude/skills/director/SKILL.md` の `usage_limited` の項、`send_screen.go` の `footerStart` の使い方
- 絞ったテストを 2 回実行(`-run 'UsageLimit|ClassifyResumeScreen'` と `-run 'ResumeScreenSentences|ClassifySendScreen|WatchUsageLimit|SendInputHolds'`)。どちらも ok。全体のテストは回していない

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- `usageLimitWaitingFooter` は `ClassifyResumeScreen` と同じ入力欄と footer を読む。同じ `lines`、同じ `findResumeInputBox`、同じ `lines[box.footerStart:]` を `resumeScreenBlocks` に渡す。claude の `footerStart` は `lower+1`、codex の `codexFooterStart` は `len(lines)` 以下なので、空の footer でも切り出しは落ちない
- 入力欄より上は読めない。`resumeScreenBlocks` に渡るのは `lines[footerStart:]` だけ。footer に prompt のブロックは出ない。入力欄は marker で始まる最後の行なので、それより下の行は marker で始まらず、`isResumePromptLine` に当たらない。だから `findResumeSentence` のように prompt を飛ばす処理は要らない
- codex は false になる。codex の `UsageLimitWaiting` は nil で、内側の loop は回らない。`codex-network-waiting.txt` の期待が `{}` で固めていて、`TestUsageLimitWaitingSentences_AreWaitingSentences` が `"codex": nil` を固めている
- `reason != ""` の分岐は待ちの道では届かない。`ClassifyResumeScreen` が waiting を返したのは同じ `findResumeInputBox` が成功したあとだけ。届いても `false` で、黙る向きではなく stall に戻る側
- 順番は変わらない。enter_only、waiting、stopped の順で、5 時間の上限の実機の画面は footer の `Continuing automatically at` が先に waiting を決め、待ちの道に入る。stopped の道は `You've hit your 5-hour limit` が `Stopped` と `UsageLimit` の両方に入っただけで、ほかは同じ
- 書き換えた期待は弱くなっていない。前は「上限の画面は必ず stopped」。今は「stopped、または waiting で、返した行が footer のブロックを 1 行にしたもので、`UsageLimitWaiting` の文を含む」。`usageLimitFooterLine` は本体と別に footer を切り出して比べるので、入力欄より上を読む変異は行が footer のブロックと一致せず落ちる。stopped の道の期待(`line` と `has`)は変わらない
- 引用の画面は claude の描き方と合う。返事の引用は `⏺ It said:`、空行、2 桁インデントの段落。ツールの出力は `⏺ Bash(cat banner.txt)` の下の `⎿  ` の行と 5 桁インデントの続き。どちらも入力欄の上で、resume の判定は waiting、`UsageLimitScreen` は false
- 見張りの複数周期のテストは「stall が残る」ことを示す。2 周期目から 4 周期目まで、各周期で active な stall が implementer のものだけで、`usage_limited` の ALERT と件がなく、画面を毎周期 1 回ずつ読み(`reads == n-1`)、stall の ALERT は 1 通のまま。footer の限定を外すと 2 周期目に落ちる
- secret と個人情報なし。証拠の写しにあるパスは `/tmp/ml308/` と scratch の repo だけ。証拠は「確かめていないこと」の節で、解除の時刻のあとの画面、`Continuing shortly`、ほかの幅や版、`agent get` の出力が待ちのあいだ変わらないか、を挙げている
- 文書は互いに合う。2 つの仕様の新しい文、`agent-messaging.md` の文、`AGENTS.md` の句、tech-debt の 452 行と 495 行が、同じ範囲(footer だけ、`Retrying in` と `Waiting for API response` は対象外、入力欄の上にだけ待ちの文を出す版は stall のまま)を言っている

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: footer のブロックの切り出し `resumeScreenBlocks(lines[box.footerStart:], marker)` が本体に 2 つ(`ClassifyResumeScreen` と `usageLimitWaitingFooter`)あり、「`ClassifyResumeScreen` と同じ道で見つける」は 2 つの式が同じであることに頼っている。`send_screen.go` も `footerStart` で別の切り方をしているので、footer の扱いを変える人は 3 か所を見る。待ちの道では届かない `reason != ""` の分岐に、届かない理由の注釈もない | `internal/org/resume_screens.go:358`、`internal/org/usage_limit_screen.go:96-109`(分岐は 99 行)、テストの助け `internal/org/usage_limit_screen_test.go:160-176` にも同じ式 | 小さな関数(例: `resumeFooterBlocks(lines, box, driver)`)にして 2 か所から呼ぶか、`usageLimitWaitingFooter` の doc に「呼ぶのは waiting と読んだあとだけで、分岐は届かない」と 1 文足す。直さなくても動きは変わらない |
| LOW | readability | L2: 「止まった(stopped)」の語が、新しく true を返す待ちの画面を含まないまま残っている。待ちの座席は止まっておらず、CLI が解除の時刻に自分で続け、`ralph org resume` は何も打たない。`--help` の「a sentence ralph org resume reads as one」は事実としても欠けた: resume はその文を待ち中と読む。人と leader に届く文(件の本文の `usageLimitedWhy`、recurred の note の `lead`、leader への ALERT)も「stopped on a usage limit」のまま。座席の行に `Continuing automatically at …` が出るのが唯一の手がかり。計画の Non-goals は director の skill を直さないと決めているので、skill の「リセットの時刻まで待ってから座席を続ける」は待ちの座席には当てはまらないまま | `internal/cli/org_watch.go:103-104`(`--help`)、`internal/org/watch_usage_limit.go:38`・`286`・`298`(人と leader に届く文)・`270`・`275`(stderr)、同ファイルの doc comment `20`・`53`・`88`・`126`・`159`、`internal/org/watch.go:254`・`316`・`1267`・`1315`。コメントは「止まった」を「ターンが終わって動いていない」と広く読めば偽ではない | `usageLimitedWhy` と ALERT の文に「または上限の解除を待っている(CLI は解除の時刻に自分で続ける)」の 1 句を足し、`--help` の括弧を「resume が上限の文と読む画面、または footer の解除待ち」に直す。コメントは `UsageLimitScreen` の doc がすでに 2 通りを書いているので、1 か所に「待ちの座席も含む」と書けば足りる。直すと本体と文字列に触れるので、記録だけの修正の例外には入らず、パイプライン全体の再走になる。ほかの理由で再走するなら一緒に、そうでなければ PR 本文の Known gaps に 1 行 |
| LOW | maintainability | L3: 証拠の文書が、まだこの branch にも main にもないファイルに、環境と分離の詳細を預けている。`docs/evidence/org-send-marker-probe-2026-11-13.md` は origin の `fix/send-input-box-marker`(`40d7a240`、レーン B の #322)にだけある。#322 より先にこの PR が merge されると、main の証拠が実在しないファイルを指す。#322 が別の名前で merge されても同じ | `docs/evidence/claude-5h-limit-screen-2026-11-13.md:31`。`git cat-file -e origin/fix/send-input-box-marker:docs/evidence/org-send-marker-probe-2026-11-13.md` は成功、この tree には `ls` で見つからない | PR 本文に merge の順(B が先)を 1 行書くか、B が先に入るまで待つ。または証拠の「環境と分離」の要点(すでに 5 項目ある)だけで足りると見て、指す文を「その branch の記録」から外す |
| LOW | security | L4: 待ちの道が返す行は、footer のブロックの全部(D4)で、モデル、effort、branch の状態の行と権限の行も入る。その状態の行は Claude Code の標準の footer ではなく、ユーザーが設定した `statusLine` の出力に見える(標準の footer にモデルと branch の行があるとは確かめられなかった。未確認)。`statusLine` はコマンドの出力を自由に描けるので、組織名、費用、パスなど、設定した人が足した文字が 300 文字まで件の本文、leader への ALERT、watch の stderr に写る。`usageLimitLine` が制御文字を落とし長さを切るので注入にはならず、止まった道も返事のブロックを写すので信頼の水準は同じ。新しいのは `statusLine` の内容が出る点だけ。計画は D4 で状態の行を切らないと決めている | `internal/org/usage_limit_screen_test.go:36-39` の期待の行(`🤖 Haiku 5.5` から `⏵⏵ bypass permissions on (shift+tab to cycle)` まで入る)、`internal/org/watch_usage_limit_test.go` の `ulClaude5hWaitLine`、`internal/org/usage_limit_screen.go:65-90` | 今のままでよい。気になるなら、待ちの文のある行とその折り返しで切る(状態の行の前まで)。実機の `statusLine` の設定がある座席で、件の本文に何が出るかを最初の実運用で見る |
| LOW | maintainability | L5: 「待ちのあいだ stall の判定が立つか」が未確認で、register に行がない。`readUsageLimitScreen` は `if stalled {` の中でだけ呼ばれる。待ちのあいだ `herdr agent get` の出力が変わり続けるなら、画面は読まれず、この変更は実機で何も起こさない。テストの fake herdr は `agent get` の出力が一定なので、そのときも緑のまま。計画の Assumptions と証拠の「確かめていないこと」には書いてあるが、新しい 495 行は「入力欄の上にだけ出る版」の穴だけを持つ | `internal/org/watch.go:1324`、`docs/evidence/claude-5h-limit-screen-2026-11-13.md:159-160`、`docs/plans/active/2026-11-13-claude-5h-usage-limit.md` の Assumptions の 2 つ目、`docs/tech-debt/README.md:495` | 495 行に「待ちのあいだの `agent get` の出力は未確認。変わるなら条件は立たず、stall のまま(黙る向きではない)」を足すか、PR 本文の Known gaps に書く。最初の実運用の上限で、watch の周期ごとの `agent get` を 1 度記録する |
| LOW | readability | L6: 小さな読みにくさが 3 つ。(a) `TestUsageLimitScreen_WaitingForALimitToReset` の最初の case は待ちでなく止まった画面(5 時間の上限の `⎿` の行)で、テスト名と合わない(doc comment が理由を書いている)。(b) `usageLimitFixtureWants` の見出しコメント「The usage limits. The weekly limit is a ⎿ line …」と、その下の最初の entry(5 時間)のあいだに 5 時間の説明の段落が挟まり、weekly と codex の説明が先に来て 5 時間の entry の前に置かれたように読める。(c) `UsageLimitScreen` の doc の「So is every other screen: one that waits for Enter …」は主語が前の文の「false」を受けて読みにくい | (a) `internal/org/usage_limit_screen_test.go:378` と case の名前 `:388`。(b) 同ファイル `:26-40`。(c) `internal/org/usage_limit_screen.go:48-56` | どれも直さなくても動きは変わらない。直すなら (a) は case を別のテスト関数に、(b) は 5 時間の説明を entry の直前に、(c) は「次の画面も false である: …」と言い直す |

## Positive notes

- 待ちの道を footer だけに限った判断(D2)が、本体の doc、表の欄の doc、2 つの仕様、tech-debt の行まで同じ理由で通っている。返事やツールの出力が引用した文では stall の知らせが止まる、という codex の計画助言の MEDIUM が、本体の形とテスト(返事、ツールの出力、4 周期の見張りのテスト)に反映されている
- `UsageLimitWaiting` が `Waiting` の文であることをテストが固め(`TestUsageLimitWaitingSentences_AreWaitingSentences`)、`Retrying in` と `Waiting for API response` を印の外に置いた理由が表のコメントに書かれている。`UsageLimit` と `Stopped` の関係のテストと同じ形で、読む人が迷わない
- 実機の画面を fixture に byte 単位で写し、hash を証拠の文書に残している。base での結果(`ok=false`、`waiting`)と、変更後の結果を並べ、確かめていないことを別の節に分けて書いた
- `usageLimitWaitingFooter` は小さく、名前から探せ、副作用がなく、エラーを握りつぶす箇所がない(届かない分岐は false に倒れる)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 待ちのあいだ `herdr agent get` の出力が変わるか(L5) | 変わるなら、解除待ちの座席は stall の判定に届かず、`usage_limited` に入らない。stall の ALERT も立たないので、座席の状態は今と同じ | この run の commit は報告 1 件に限られ、register に書けない。確かめには実機の上限が要る | 最初の実運用の 5 時間の上限、または上限に当てる確かめ。495 行に足す | `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(Assumptions) |
| 人と leader に届く文と `--help` の「stopped」の語(L2) | 待ちの座席を止まったと読んだ人が、CLI が自分で続くのに resume や立て直しを頼む | 直すと本体の文字列に触れ、パイプライン全体の再走になる | 次に `usageLimitedWhy` か ALERT の文か `org watch --help` に触るとき | 同上(Non-goals の director の skill) |

_(register には書いていない。この run は報告 1 件だけを commit する指示のため。L5 を 495 行に足すか、L2 を直すか PR 本文に残すかは作者が決める。L1、L3、L4、L6 は register に載せる性質ではなく、直すか、そのまま PR に残すかの判断。)_

## Recommendation

- Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6
- Follow-ups: LOW 6 件はどれも直さなくても動きは変わらない。L2 は人に届く文字列と `--help` で、`/sync-docs` が文書の側から拾う可能性が高い。直す場合はコードの文字列に触れるので、記録だけの修正の例外には入らない。L3 は merge の順の 1 行。L5 は 495 行への 1 文か PR 本文の Known gaps。L1、L4、L6 は任意
- 未確認: 実機で見張りが待ちの座席を `usage_limited` に入れること(L5 の `agent get`)、`Continuing shortly` の画面、`go vet`、`gofmt`、golangci-lint(`/verify` の側)、全体のテスト(`/test` の側)、`statusLine` の設定の有無(L4)
