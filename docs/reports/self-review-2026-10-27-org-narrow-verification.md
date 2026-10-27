# Self-review report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 と直し F1〜F6 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 / 2 の中の 2 回目の run。1 回目は dd2ffa05 に commit した LOW 5 件で、付録 A に残してある)
- Cycle: 2 / 2、run 2(最後の run。ここで直さなかった LOW は繰り越しになる)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 2f884ae1)のうち、1 回目のあとの 4 コミット(`git diff dd2ffa05..HEAD`、10 ファイル、+377/-114)を中心に読んだ。F5 a7d879d9(1 回目の所見 1・2・3・5 の直し)、F6 854be9db(止まったスライスの再開、`RESUME:`)、51cf59e2(inline。スライスを狭めるときの `RESUME:`)、2f884ae1(計画の Progress だけ)。残りの差分は 1 回目に読んだ範囲で、この 4 コミットが触っていない(`reviewer.md`、leader.md の他の項目、skill の `send` の行)。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: この run の所見は R2-1〜R2-5。1 回目(付録 A)の所見は 1〜5 の番号のまま残してある。テストのコメント(`internal/org/prompts_verification_predate_test.go:15`、`:56` の「cycle 2 self-review LOW 1」「LOW 2 and 3」)と計画の Progress の「LOW 4」は付録 A の番号を指す。付録 A が引く `implementer.md` の行番号は 50fc3faf の時点のもので、F5 のあとの `implementer.md` とは合わない。

## Evidence reviewed

- `git diff dd2ffa05..HEAD` の全体。`implementer.md` は項目 2 つ(`:15-27`、`:62-76`)とレポート契約の 1 文(`:154-156`)、`leader.md` は「ミッション」の 3(`:40-54`)、`/org` skill は implementer の箇条書き(`.claude/skills/org/SKILL.md:1336-1342`)と leader 運用の手順 3(`:1457-1460`)、テストは新しい `prompts_verification_predate_test.go`(273 行)と、そこへ移した分を削った `prompts_verification_multi_test.go`(90 行)。skill の 4 つの写しは `cmp` で一致。`git diff dd2ffa05..HEAD --check` は rc 0。
- 1 回目の所見 1 の直し。旧い言い方 `座席が始める前`・`スライスより前`・`既存の変更`・`編集を始める前に記録した`・`例外は 2 つ` を雛形、skill、`internal/org/*.go`、仕様、recipe に対して grep した。残るのはテストの「もう出さない」照合(`prompts_verification_predate_test.go:47`、`:139`)だけ。
- `markdownItem` の切れ目(trim した行が `- ` か番号で始まる所で切る)に対して、`implementer.md` の 2 つの項目(`:15-27`、`:57-76`)の中にその形の行がないことを読んだ。3 つの場合を並べた文がサブ箇条になっていないので、テストには項目の全文が渡る。
- F6 の設計を 3 つの入力で読んだ。(1) 秘密情報で止まったスライス: implementer が unstage して値を消し BLOCKED、leader が `git diff HEAD` で消えたことを確かめて `RESUME:` に並べ、implementer が自分の変更として引き継ぎ、自己確認の秘密情報の検査をその変更にもう一度かける。(2) `RESUME:` なしの頼み直し: 前の TASK の残りが前からあった変更になり、scope と重なるので BLOCKED(`leader.md:45-46` の「この行がないと…」と一致)。(3) スライスを狭める(51cf59e2)。F5 の定義(TASK ごとの記録)だけだと、止まったスライス自身の残りも前からあった変更になって再開の道がなくなる。F6 はその道を `RESUME:` に挙がったパスだけに開けている。
- 51cf59e2(inline)の判定。leader.md 1 本とそのテストの 1 句(2 ファイル、+5/-1)で、F6 の implementer が挙げた穴への 1 文なので、軽い編集の例外の範囲に収まる。例外のとき見る周りの 3 つ: 直前の文(挿入で折り返しが崩れた。R2-4)、同じ内容を言う別の面(skill の 4 つの写しに狭める文がない。R2-2)、近い兄弟(implementer 側に「元に戻す」の対応する規則はない。`RESUME:` のパスは scope の中で自分の変更なので、scope の中の編集として足りると読める)。
- 確かめていない: テストと静的検査の実行、変異の再現(計画の記録では F5 で 3 つ、F6 で 3 つ、51cf59e2 で 1 つを `-overlay` で確かめている)、座席の実機での再開。

## 前回の所見の扱い

| 1 回目の所見 | 扱い | 理由 |
| --- | --- | --- |
| 1 naming(呼び方が 3 通り) | 解消。雛形をまたぐ説明の食い違いが R2-3 に残る | 定義は `implementer.md:15-18` に 1 度(「以下「前からあった変更」と呼ぶ」)。BLOCKED に書く語も同じ(`:73-74`「前からあった変更だと書き」)。定義が 1 度だけであることをテストが数える(`prompts_verification_predate_test.go:44`) |
| 2 readability(「例外は 2 つ」の係り先) | 解消 | `:66-70` が「コミットしないのは次の 3 つのときで…、どれも BLOCKED を leader に返す: A とき、B とき、C とき」と 3 つを 1 文に並べ、`:70-76` が番号で受ける。サブ箇条にしていない |
| 3 security(unstage の形) | 解消 | `:64-66` が `git restore --staged -- <このスライスのパス>...` を指定し、パスを付けない `git reset` を禁じる(理由は「前からあった変更の stage まで外れる」)。`git restore --staged .` は名指しされないが、肯定の形が「このスライスのパス」なので足りる |
| 4 maintainability(標準フローの面) | 範囲外のまま(計画の決定)。台帳の行はまだない | `docs/tech-debt/README.md` に `unstage`・`RESUME` はない(grep)。`/sync-docs` が足す。下の案 A |
| 5 maintainability(テストの置き場) | 一部解消。残りが R2-5 | F4 の 2 本は `prompts_verification_predate_test.go` に移り、`…_multi_test.go` は複数スライスの 2 本だけに戻った。移した先と本体のあいだに残りがある |

cycle 1 の 10 件は、F5・F6 が `reviewer.md` と leader.md の他の項目に触れていないので解消のまま。ただし「複数のスライスの単数形」(cycle 1 の 1 回目 #4 と 2 回目 LOW 1)と同じ形が、名前のない 1 行の `RESUME:` として戻っている(R2-1)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R2-1 | LOW | maintainability | `RESUME:` の行にだけスライスの名前がない。複数のスライスを頼む TASK では `VERIFY:`・`COMMIT:`・`SELF-CHECK:` が行ごとにスライスの名前で始まる(cycle 1 の直しで揃えた形)のに、`RESUME:` は名前のない 1 行で、implementer は「このスライスの変更として引き継ぐ」。止まったスライス S1 と別のスライス S2 を 1 つの TASK で頼み、implementer が S2 から先に取りかかると、S1 のパスが S2 のコミットに入る。`leader.md:41` の「同じスライスをもう一度頼むとき」は単一のスライスを前提にするが、「`RESUME:` を付けるのは 1 つのスライスの TASK だけ」とは書かず、順序を決める規則もない。起きにくい。実機では見ていない(未確認) | `leader.md:37-38`(名前付きの `VERIFY:`)、`:41-45`(名前のない `RESUME:` と例)。`implementer.md:28-32`、`:161-164`(名前付きの行)、`:19-21`(「このスライスの変更として」) | `leader.md` に「`RESUME:` を付けるのは、止まったスライス 1 つだけを頼む TASK(ほかのスライスは別の TASK にする)」を 1 文、`implementer.md:19` の前後に同じ趣旨を 1 文。または `RESUME: S1: <パス>...` にして名前で始める。前者のほうが小さい。テストの句 1〜2 個(`prompts_verification_predate_test.go:187-196`、`:223-229`) |
| R2-2 | LOW | maintainability | skill の leader 手順 3 は「値が作業ツリーから消えたことを先に確かめる」で止まり、確かめて消えていなかったときの手がない。`leader.md:50-54` には「消えていないとき、また BLOCKED が秘密情報を前からあった変更にあると書いたときは、再開せず `ralph org escalate` で BLOCKED を上げる」がある。(A) 経路の昇格したセッションの leader は雛形を読まず skill だけを読む(計画 S4 の記録)ので、skill の文が唯一の指示になる。狭めるときの「止まった TASK が残したパスは外すものも含めて並べ、外すパスは元に戻すよう TASK に書く」(`leader.md:46-49`)もない。cycle 1 で skill が雛形より短いことを Cleared にしたのは unstage の手順の深さの話で、これは分岐の欠け。cycle 1 の 1 回目 #2(確かめるだけで、そうでないときの分岐がない)と同じ形 | `.claude/skills/org/SKILL.md:1457-1460`(4 つの写しで同一)。計画 `docs/plans/active/2026-10-27-org-narrow-verification.md:175`(昇格したセッションは skill だけを読む)。skill 側のテストは確かめるところまでを固定する(`prompts_verification_predate_test.go:260-262`) | skill の括弧を「(秘密情報で止まったあとは、値が作業ツリーから消えたことを先に確かめる。消えていないとき、または前からあった変更の秘密情報のときは、再開せず `escalate` で BLOCKED を上げる)」の形にし、狭める文を 1 文足す。4 つの写しとテストの句 1 個 |
| R2-3 | LOW | naming | `leader.md` の「前からあった変更」が、同じ項目の中で別の広さで使われている。`:48` の「並べなかったパスは前からあった変更として残り」は、止まった TASK 自身が残して `RESUME:` に並べなかったパスも含む(`implementer.md:15-24` の定義どおり)。`:52` の「前からあった変更(止まった TASK より前の変更。implementer は触らない)」は、それを除く書き方になる。implementer の BLOCKED が「前からあった変更にある秘密情報」と書いたとき、leader は、自分が並べ忘れた止まった TASK のパスなのか、止まった TASK より前の変更なのかを説明から切れない。切れないまま escalate に倒れるので安全側で、重さは低い。cycle 2 の 1 回目の所見 1(同じ集合に複数の呼び名)が、雛形をまたいで一つ残った形 | `leader.md:48`、`:52`。`implementer.md:15-24` | `:52` の括弧を implementer の定義に合わせ、「`RESUME:` に並べなかったパスの変更も含む。implementer は触らない」とする。括弧ごとテストが固定しているので(`prompts_verification_predate_test.go:194`)、句も一緒に直す |
| R2-4 | LOW | readability | 狭めるときの文(51cf59e2)の「残したパスはすべて並べ」の「残した」が 2 通りに読める。`leader.md:41-42` では「止まった TASK がコミットせずに残した」(置き去りにした)の意味。`:46-49` では「外すパス」と対になって、「狭めたスライスに残すパス」とも読める。後者に取ると外すパスは並べないことになり、続く括弧(並べなかったパスは…implementer がまた止まる)と食い違う。括弧まで読めば前者と分かる。挿入で折り返しも崩れ、`:49` が「止まった」で終わる 1 行だけ短い行になった(`implementer.md:156` も F6 の挿入で同じ) | `leader.md:41-42`、`:46-49`。テストの句 `prompts_verification_predate_test.go:192` | 「止まった TASK が残したパスは、外すものも含めてすべて並べ、外すパスは元に戻すよう TASK に書く」とし、前後を折り返し直す。テストの句 1 個 |
| R2-5 | LOW | maintainability | テストの置き場と書き方の残りが 4 点。(a) 同じ文が 2 つのファイルで固定されている。「コミットしないのは次の 3 つのときで…」と「1 つ目では…」を、`prompts_verification_test.go:97-98` と `prompts_verification_predate_test.go` の steps(`:84-97`)の両方が照合する。文を直すとき 2 か所を直す。(b) ファイル名が `predate` なのに、6 本のうち 3 本(`:176`、`:212`、`:253`)は `RESUME:` の再開の話。(c) `phrasesInOrder`(`:152`)は最初に欠けた句を返し、成功なら `""` を返すので、名前が真偽値のように読める。同じファイルの `SelfCheckLeaves…`(`:98-107`)は同じ順序の照合を手で書いていて、規則も違う(`at = -1`、`i < at`、句の終わりまで進めないので、重なった句や繰り返しの句が通る)。(d) テスト名の `…ThatPredateTheSlice`(`:73`、`:124`)は、雛形の語(TASK ごとの記録、`implementerPredatingChanges`)と合わず「スライス」と言っている | `prompts_verification_test.go:97-98`。`prompts_verification_predate_test.go:11`、`:73`、`:84-97`、`:124`、`:152-163`、`:176`、`:212`、`:253` | 急がない。次にこの項目の文を直すとき、(a) は本体の `MechanicalSelfCheckBeforeEachCommit` から重なる 2 句を外すか、predate 側を外すかの片方にする。(b) は `…_implementer_test.go` のような名前か、先頭に中身を 1 行。(c) は 1 つの補助(`firstPhraseOutOfOrder` など)にして両方で使う |

## Positive notes

- F6 は、F5 の定義(TASK ごとの記録)が自分で開けた穴を閉じている。記録だけだと、止まったスライス自身の残りも前からあった変更になり、再開が BLOCKED の繰り返しになる。`RESUME:` に挙がったパスだけを自分の変更として引き継ぎ(`implementer.md:19-21`)、挙がっていないパスは前からあった変更のまま守る(`:24`)。失敗しても BLOCKED にもう一度なるだけで、残りを黙って上書きしない。
- 引き継ぎの範囲が 3 方向で絞られている。scope の中のパスだけ(`implementer.md:22-23`)、引き継いだ変更も stage して自己確認の 5 項目に通す(`:21`、秘密情報の検査もかかる)、leader は秘密情報の値が消えたことを確かめてから送る(`leader.md:50-52`)。
- RESULT に、引き継いだパスを決定事項として書く(`implementer.md:154-156`)ので、leader が自分の `RESUME:` の行と突き合わせられる。
- 定義のテストが「定義が 1 度だけ」と「旧い呼び名が戻らない」の両方を固定する(`prompts_verification_predate_test.go:44`、`:47`)ので、F5 を戻せば落ちる。順序の照合 `phrasesInOrder` は一致した句の終わりまで進めるので、同じ句で 2 つの手順を満たすことがない。

## Cleared, do not raise again

- `leader.md:50-52` の `git diff HEAD`(未追跡のファイルは中身を読む)は、消えていない値を leader の画面に出しうる。確かめのために差分を読むことは避けられず、作業ツリーも同じ信頼の範囲にある。escalate の本文は「ポインタで書き、差分を貼らない」の原則の下にある。指摘にしない。
- 型付きプロトコルの節(`leader.md:464-466`、`:478-489`)が本文の行として挙げるのは `VERIFY:` だけ。`RESUME:` の「本文の行で、ヘッダにはしない」は「ミッション」の 3(`:44-45`)に書かれ、テストが固定している。指摘にしない。
- 51cf59e2 を inline で直したこと。2 ファイル、+5/-1 の文言で、軽い編集の例外の範囲(上の Evidence)。
- skill の 4 つの写しが雛形より短い(unstage の手順や値を消す手順がない)こと。cycle 1 から続く要約の程度。ただし分岐の欠けは R2-2。
- 付録 A の「Cleared」の項目は引き続き有効。

## Tech debt identified

この run が cycle 2 / 2 の最後なので、直さなかった LOW は繰り越しになる。この run の commit はレポートと insight event に限るので、台帳には書いていない。次の 2 行を `/sync-docs` か orchestrator が足す。付録 A の台帳案は、この 2 行で置き換える(1 回目の所見 1・2・3 は F5 で直った)。

案 A(1 回目の所見 4。直す直さないにかかわらず足す):

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 標準フローの implementer の秘密情報の片付けに、前からあった変更を除く文がない。`.claude/agents/implementer.md:40` と `.codex/agents/implementer.toml:54-56`(「unstage it and stop」)、`.claude/skills/implement/SKILL.md:42-43`(「take a secret out of the worktree」で、止まったスライスが作ったファイルに限る文がない)。写しは `.agents/skills/implement/SKILL.md`、`templates/base/.claude/agents/implementer.md`、`templates/base/.claude/skills/implement/SKILL.md`、`templates/base/.agents/skills/implement/SKILL.md`、`templates/base/.codex/agents/implementer.toml` | 前から stage されていた別の作業を、片付けが unstage するか消しうる。標準フローの作業ツリーは clean な base から作るので、共有の作業ツリーで起きる org より当たりにくい | #261 で merge 済みの文で、#262 の範囲の外 | 標準フローの implementer の定義か `/implement` の手順 7 を次に直すとき。または前から stage されていたパスを巻き込んだ報告が出たとき | `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(付録 A の所見 4)、`docs/reports/cross-review-triage-org-narrow-verification.md`(#1) |

案 B(この run の R2-1〜R2-5):

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org の再開 `RESUME:` の文言とテストの残り。R2-1: `RESUME:` にスライスの名前がなく、複数のスライスの TASK で引き継ぎ先が決まらない。R2-2: `/org` skill の leader 手順 3 に、値が消えていないときの escalate と、狭めるときの文がない。R2-3: `leader.md:52` の「前からあった変更」の説明が `implementer.md` の定義と違う。R2-4: `leader.md:46-49` の「残したパス」が 2 通りに読める。R2-5: テストで同じ文を 2 か所が固定し、`predate` のファイルに再開のテストが入り、順序の照合が 2 通りある | 再開の誤読は BLOCKED か escalate に倒れる(安全側)。昇格した leader だけが skill の欠けを踏む。動作を変えない文言とテストの整理 | cap 2 / 2 に当たり、再 run すると 5 件の文言直しのために全段を回し直す | org の leader か implementer の雛形、`/org` skill の leader 運用を次に直すとき。または複数のスライスの TASK で `RESUME:` を使う運用が出たとき | `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(R2-1〜R2-5)、`docs/plans/active/2026-10-27-org-narrow-verification.md`(F6) |

_(この review では `docs/tech-debt/README.md` を書き換えていない。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5)
- Follow-ups: 1 回目の所見 1・2・3 は解消、4 は範囲外で台帳へ(案 A)、5 は一部解消。F6 の再開の仕組みは安全側に倒れる設計で、新しい所見はどれも「誤読したときの遅れ」で、動作を止めるものはない。cap に届いているので回し直さず、案 B の 1 行にまとめて台帳へ足す。cap を上げて回し直すなら優先は R2-1(複数のスライス)、R2-2(skill の分岐)、R2-3 と R2-4(言い方)、R2-5(テスト)の順。
- inline 修正 51cf59e2 の判定: 範囲内。skill の写しに狭める文がないことだけを R2-2 に入れた。

## 付録 A: cycle 2 の 1 回目(HEAD 50fc3faf、コミット dd2ffa05)

1 回目のレポートの全文。見出しを 1 段下げ、`Merge:` の行を「1 回目の判定:」に替えた以外は書き換えていない(`git show dd2ffa05:docs/reports/self-review-2026-10-27-org-narrow-verification.md` と照合できる)。所見の番号 1〜5 はテストのコメントと計画が引いている。本文の行番号(`implementer.md:53-65` など)は 50fc3faf の時点のもので、F5 のあとは動いている。

### Self-review report: org-narrow-verification(1 回目)

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 と直し F1〜F4 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 / 2。cross-review の ACTION_REQUIRED 1 件を F4 ffafc9d2 で直したあとの再 run。cycle 1 の 2 回目は LOW 2 件)
- Cycle: 2 / 2(最後の run。ここで直さなかった LOW は繰り越しになる)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 50fc3faf、23 ファイル、+1987/-101、22 コミット)。中心は F4 ffafc9d2(7 ファイル、+102/-11): `internal/org/prompts/implementer.md` の機械的な確かめの項目、`/org` skill の implementer の箇条書き(4 つの写し)、テスト 3 か所(`prompts_verification_multi_test.go` に 2 本、`prompts_verification_test.go` の期待 2 行)。残りの差分は cycle 1 に読んだ範囲で、F4 が触っていない(`git show ffafc9d2 --stat` に `leader.md`、`reviewer.md`、docs、台帳はない)。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。

### Evidence reviewed

- `git show ffafc9d2` の全差分と、トリアージ `docs/reports/cross-review-triage-org-narrow-verification.md` の指摘 #1(stage 済みの scope 外の資格情報ファイルを、秘密情報の片付けが unstage して消しうる)。直し後の `implementer.md` 全文(ミッションの 15-18 行目の「記録」、48-65 行目の確かめの項目、86-91 行目、109-115 行目、139-155 行目)。
- 指摘 #1 への答えを 3 点に分けて雛形の文で確かめた。(1) 直すのはこのスライスの変更だけ(`implementer.md:53-54`)。(2) 記録した `git status --porcelain` に出ていた変更は stage 済みも含めて書き換えも削除も unstage もしない(`:54-56`)。ファイルを消すのは「このスライスで作った未追跡のファイル」だけ(`:59-60`)。(3) 前からあった変更の秘密情報と、前から stage されていた scope 外のパスは、触らずコミットせずに BLOCKED(`:61-65`)。コミットしない理由は「コミットには stage してあるものがすべて入る」(`:56-57`)で、`git commit` が index 全体を取る挙動と合う。
- skill の写し: 4 つが `cmp` で一致。変更行は `.claude/skills/org/SKILL.md:1336-1338`。雛形の言葉より短く、unstage と値を消す手順は書かず、「そのスライスの変更だけ」「前からあった変更の秘密情報と前から stage されていたパスには触らずコミットせずに BLOCKED」を言う。cycle 1 から同じ要約の程度。
- テスト: `squashSpace` は `strings.Fields` を空文字でつなぐので(`prompts_test.go:139`)、雛形の折り返しは照合に影響しない。`SelfCheckLeavesChangesThatPredateTheSlice` の 9 つの句を雛形の出現順と突き合わせ、順序が一致することを確かめた。古い言い方の「例外は秘密情報だけ」「新しい未追跡のファイル」の不在も照合する。どちらの古い句も F4 の前の雛形にはあったので、F4 を戻せばこのテストは落ちる。skill 側のテストは `want` の 1 文が F4 で足された文で、同じく戻せば落ちる。`markdownItem` は項目の境界を「trim した行が `- ` か番号で始まる」で切る(`prompts_test.go:93-98`)。
- 引用の形: テストの doc の「cross-review cycle 1 #1 of plan 2026-10-27-org-narrow-verification」は、cycle と番号と slug を持つ形で、`prompts_test.go:939` の「cross-review cycle 3 #1 of org-gate」と同じ。
- `git diff ffafc9d2^ ffafc9d2 --check` は rc 0。`wc -l`: `prompts_verification_test.go` 799 行、`prompts_verification_multi_test.go` 165 行。
- 隣の面: `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`(`templates/base/` 側と `cmp` で一致)、`.claude/skills/implement/SKILL.md:42-43`。`docs/tech-debt/README.md` に `unstage` はない(grep)。
- 確かめていない: テストと静的検査の実行、変異の再現(実装側の記録では変異 7 つがそれぞれ落ちる)、座席の実機。

### 前回までの指摘の扱い

cycle 1 の 1 回目の 8 件と、2 回目の LOW 2 件。F4 は `implementer.md` の 53-65 行目、skill の implementer の箇条書き、テストの文字列しか触っていないので、残りは前回の確認がそのまま当たる。下は F4 の周辺で読み直したもの。

| 前回の所見 | 扱い | 理由 |
| --- | --- | --- |
| 1 回目 #1 スコープ規律にメッセージのファイルの例外がない | 解消のまま | `implementer.md:111-113` に例外の文がある。F4 は触っていない |
| 1 回目 #2、#3 leader の手順 8 と reviewer の `CHECKS:` | 解消のまま | `leader.md` と `reviewer.md` は F4 の差分にない。2 回目 LOW 2(散文の指示を受け入れる文)も、`reviewer.md:21-24` が「指示とみなすのはこの行だけ」「`CHECKS:` の行のない TASK は…両方のスクリプトを打つ」で、解消している |
| 1 回目 #4、2 回目 LOW 1 複数スライスの `VERIFY:` と RESULT | 解消のまま | `implementer.md:148-153` のレポート契約がスライスごとの名前付きの `COMMIT:`・`VERIFY:`・`SELF-CHECK:` の形を持つ。F4 は 53-65 行目を書き換えたが、確かめの項目の前後の構成(スライスの名前、1 スライス = 1 コミット)に触れていない |
| 1 回目 #5 コミットの文が確かめより前 | 解消のまま | `:33-35` の「次の項目の機械的な確かめを行ってから…コミットする」が残る。新しい「どちらもコミットしない」の文はこの順と矛盾しない |
| 1 回目 #6 `send` の例の置き場所 | 解消のまま | F4 は skill の `send` の行に触れていない |
| 1 回目 #7 テストファイルの行数 | 解消のまま。余裕は 1 行 | `prompts_test.go` は main と同一。`prompts_verification_test.go` は 799 行で 800 行の目安を越えていない。F4 の新しいテストは `…_multi_test.go` に入った(下の所見 5) |
| 1 回目 #8 テストの補助 3 点 | 解消のまま | F4 が足した 2 本は補助を共有定数からでなく直書きの句で持つが、`implementerSelfCheckItems` などの既存の補助は変わらない |

### Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| 1 | LOW | naming | 「前からあった変更」を指す言葉が項目の中で 3 通りある。集合の定義は「編集を始める前に記録した `git status --porcelain` に出ていた変更」(記録)。それを指すのが「座席が始める前からあった変更」「座席が始める前から stage されていた」(座席)と、leader に返す BLOCKED に書く「スライスより前からあった変更」(スライス)。この座席は常駐する(`:8`)ので、「座席が始める前」を座席の起動前と読むと、前の TASK が残した stage 済みのパスは前からあった変更に入らず、「このスライスの変更」として消せる側に倒れる。おそらくこの読み違いは起きにくい(記録の文が先にある)。未確認です。ただ BLOCKED に書く語が座席の語と違うので、leader が読む語と座席が持つ語の対応が文からは取れない | `implementer.md:54-55`(座席)、`:61-62`(スライス)、`:63-64`(座席)、`:8`(常駐)。skill は `.claude/skills/org/SKILL.md:1336-1337`(座席)。テストが「座席が始める前」を 3 か所で固定する: `prompts_verification_multi_test.go:120`、`:126`(「座席が始める前から stage されていた」)、`:161`(skill 側)。同じファイルの doc に "predates the seat" | 言い方を 1 つにする。記録で定義した語に合わせ、たとえば「この TASK の編集を始める前に記録した…に出ていた変更」を定義の文だけに置き、あとは「記録にあった変更」と呼ぶ。BLOCKED に書く語も同じにする。直す場所は雛形 3 か所、skill の 4 つの写し、テストの句 3 つ |
| 2 | LOW | readability | 「例外は 2 つで、どちらもコミットしない」(`:56`)の「例外」の係り先が、直前の「前からあった変更は触らない」(`:54-56`)ではなく、2 文前の「scope の中で直せない項目はそのまま残し、RESULT に found and not fixed として書く」(`:52-53`)になる。読み手は直前の文に係ると取りやすく、そう取ると 1 つ目(このスライスの秘密情報を unstage して消す)は直前の規則の例外になっていない。コミットしない場合は実際には 3 つ(自分の秘密情報、前からあった変更の秘密情報、前から stage されていた scope 外のパス)あるのに、2 つ目の「前からあった変更の秘密情報」が 1 つ目の中の節(`:61-62`)に入って、数が合わない。項目は 18 行の 1 文の連なりで、座席が実行中に引いて読むには、場合分けが読み取りにくい | `implementer.md:52-65`。項目の境界の規則 `prompts_test.go:93-98` | 「例外は 2 つで」を「次のときはコミットしない」に替え、3 つの場合を同じ深さの文で並べる。サブ箇条にはしない(`markdownItem` が入れ子の `- ` で項目を切るので、テストの照合が項目の途中までになる)。テストの句 1 つ(`prompts_verification_multi_test.go:121`、`prompts_verification_test.go:97`)を一緒に直す |
| 3 | LOW | security | 「unstage」の仕方が書かれていない。コミットの側は `git add <path>...` を指定し `-A`・`-u`・`.` を禁じる(`:33`)のに、unstage の側は「このスライスのパスを unstage し」(`:58-59`)だけで、コマンドの形がない。引数なしの `git reset` や `git restore --staged .` は index 全体を空にするので、この直しが「触らない」と決めた前から stage されていたパスまで unstage する。内容は残るので失うのは index の状態だけで、重さは低い。ただトリアージの #1 が避けたい種類の作用で、座席が省略した形を選びやすい箇所でもある | `implementer.md:58-59`、`:65`。雛形 3 本に `git restore --staged` と `git reset` はない(grep) | 「パスを指定して unstage する(`git restore --staged -- <path>...`)。引数なしの `git reset` と `git restore --staged .` は使わない」を 1 文足す。skill は要約なので足さなくてよい。テストに 1 句 |
| 4 | LOW | maintainability | 同じ「stage した差分の秘密情報が消せなければ unstage して止まる」が、この計画の外の面に同じ形で残る。トリアージの #1 は、前から stage されていたパスを巻き込む穴だった。標準フローの implementer 定義(`.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`)は「unstage it and stop」で、前からあった変更を除く文がない。`/implement` の手順 7(`.claude/skills/implement/SKILL.md:42`)は「take a secret out of the worktree (edit the value out, or delete a new untracked file that holds it)」で、止まったスライスが作ったファイルに限る文がない。標準フローの作業ツリーは clean な base から作るので、共有の作業ツリーで起きる org より当たりにくい。計画の F4 の記録(`docs/plans/active/2026-10-27-org-narrow-verification.md:188`)は `/implement` の手順 7 だけを台帳に回すと書き、座席の定義 2 本(と `templates/base/` の写し)を名指していない。台帳に行はまだない(`docs/tech-debt/README.md` に `unstage` はない) | `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`、`.claude/skills/implement/SKILL.md:42-43`。写しは `.agents/skills/implement/SKILL.md`、`templates/base/.claude/agents/implementer.md`、`templates/base/.claude/skills/implement/SKILL.md`、`templates/base/.agents/skills/implement/SKILL.md`、`templates/base/.codex/agents/implementer.toml`。#261 で merge 済みの文で、この計画の範囲の外 | この計画では直さない(範囲の外)。`/sync-docs` が足す台帳の行は、`/implement` の手順 7 だけでなく、上の 3 種類の面(skill、Claude の agent 定義、Codex の agent 定義)と、それぞれの写しを名指す。「stage 済みの前からあった変更を除く文を足す」を trigger にする |
| 5 | LOW | maintainability | F4 のテスト 2 本が `prompts_verification_multi_test.go` に入ったが、このファイルは複数スライスの形(RESULT の名前付きの行、`CHECKS:` の行だけを読む reviewer)のためのもので、F4 のテスト(前からあった変更に触れないこと)は複数スライスと関係がない。同じ項目(機械的な確かめ)を固定するテストが 2 つのファイルに分かれた: `MechanicalSelfCheckBeforeEachCommit`(`prompts_verification_test.go:81`)と `SelfCheckLeavesChangesThatPredateTheSlice`(`…_multi_test.go:107`)。入れた理由は本体のファイルが 799 行で 800 行の目安に 1 行しか余裕がないことと読める(推測)が、ファイル名からは中身が引けない。`.claude/rules/ralph/architecture.md` の module の境界をパスから推せるようにする方針に合わない | `prompts_verification_multi_test.go:107`、`:153`、`prompts_verification_test.go:81`(799 行) | ファイル名を中身に合わせる(たとえば `prompts_verification_implementer_test.go` に implementer の確かめ 3 本を寄せ、本体から出す)か、先頭にファイルの中身を 1 行書く。急がない。次にテストを足すときに分ける |

### Positive notes

- 直しの規則を、新しい検出の仕組みでなく、雛形がすでに求めていた「編集を始める前の `git status --porcelain` の記録」で決めている。座席が新しく覚える手順がなく、削除を許す範囲も「このスライスで作った未追跡のファイル」に絞られた(`:59-60`)。
- 「コミットには stage してあるものがすべて入る」(`:56-57`)の 1 文が、コミットしない 2 つの場合の理由になっている。座席が `git commit` に pathspec を付けて避ける道を選ばず、BLOCKED にする筋が文から読める。
- 2 本目のテスト(skill 側)は、F4 で足した 1 文を `## 役割` の implementer の箇条書きに限って照合する。隣の箇条書きの一致で通ることがない。
- 1 本目のテストは、9 つの句の順序と、古い言い方の不在の両方を固定する。F4 の前の雛形にあった句を使うので、戻せば落ちる(F4 の前後の差分で確かめた)。
- 1 回目の 8 件と 2 回目の 2 件は、F4 を挟んでも解消のまま。F4 の差分が小さく(雛形 1 項目、skill の 1 文、テスト)、他の面に触っていない。

### Cleared, do not raise again

- テストの doc の「cross-review cycle 1 #1 of plan …」は、triage が cycle ごとに書き換えられても、cycle の名前を持つので指す先を取り違えにくい。既存の `prompts_test.go:939` と同じ形。
- 計画の S1(`docs/plans/active/2026-10-27-org-narrow-verification.md:25`)が F4 の前の規則のまま残るのは、F1〜F3 の直しも S の行を書き換えず Progress に記録した形(`:178-181`)と同じ。F4 の記録は `:186-189` にある。
- 仕様 `docs/specs/2026-08-01-org-runtime.md` の 2026-10-27 改訂 (a)(`:35`)は、秘密情報の片付けの規則を持たず、5 項目の列挙だけなので、F4 で直す文がない。
- skill が雛形より短く、unstage と値を消す手順を書かないのは、cycle 1 から続く要約の程度。
- 前回までの 10 件(上の表)。

### Tech debt identified

この run が cycle 2 / 2 の最後なので、直さなかった LOW は繰り越しになる。上の LOW 5 件のうち、このあと直されない分は、1 本の行にまとめて台帳へ足す。この run の commit はレポートと insight event に限るので、台帳には書いていない。次の行を `/sync-docs` か orchestrator が足す。所見 4 はこの計画の範囲の外なので、直す直さないにかかわらず足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org の implementer の確かめの文言の残り(所見 1: 「前からあった変更」の言い方が 3 通り、所見 2: 「例外は 2 つ」の係り先、所見 3: unstage のコマンドの形がない、所見 5: F4 のテストが multi のファイルに入った)と、標準フローの implementer の同じ形の穴(所見 4: `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`、`.claude/skills/implement/SKILL.md:42-43` と各写し) | 文言の読み違いと省略した unstage で、前から stage されていた別の作業の index を空にしうる。内容は失われない。標準フローは clean な base の作業ツリーなので、起きにくい | pipeline の cap 2 / 2 に当たり、再 run すると 5 件の文言直しのために全段を回し直すことになる。動作を変えない文言の直しで、座席は F4 の規則に従う | org の implementer 雛形か、標準フローの implementer 定義を次に直すとき。または前から stage されていたパスを巻き込んだ報告が出たとき | `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(cycle 2)、`docs/reports/cross-review-triage-org-narrow-verification.md`(cycle 1 の #1) |

_(この review では `docs/tech-debt/README.md` を書き換えていない。上の行は `/sync-docs` か orchestrator が足す。)_

### Recommendation

- 1 回目の判定: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5)
- Follow-ups: cross-review の ACTION_REQUIRED 1 件(前から stage されていた変更を巻き込む片付け)は、雛形と skill の 4 つの写しとテストで直っている。cycle 1 の 10 件は解消のまま。残る LOW 5 件は文言とテストの置き場所で、動作を変えない。cap に届いているので回し直さず、上の 1 行にまとめて台帳へ足す。回し直すなら、優先は所見 2(「例外」の係り先。直す場所が雛形の 1 項目とテストの句 1 つで小さい)、所見 3(unstage の形。1 文)、所見 1(言い方の統一)の順。所見 4 は直さず、台帳に標準フローの面を全部名指して足す。
