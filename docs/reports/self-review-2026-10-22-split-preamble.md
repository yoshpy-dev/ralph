# Self-review report: split-preamble

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-split-preamble.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff a7bdc884...HEAD`(HEAD 7cc48f33、21 ファイル、+791/-82)。S1 03651616(`SplitPlan.Preamble`)、S2 ef33170f と S2b b10c6f25(`featureLeaderTask` の前文のブロックと `fenceFor`、leader.md の手順 1・3・5、start の help とエラー、テスト)、S3 1b65bf49 と 546ad029(`/org`・`/director` skill 4 写し、仕様 FR-4、README)、S4 c407bd67 と 7cc48f33(記録)、計画 2 コミット。diff の質だけを見た

## Evidence reviewed

- `internal/org/split.go` の diff を `parseSplitPlan` の全文(300〜390 行)と突き合わせた。前文に入る行は 2 か所だけで、見出しの行(`## ` で `## Features` でなく、`featuresLine == 0`)と、見出しのあとの本文の行(`!inFeatures` かつ `featuresLine == 0`)。フェンスの中の `## Features` は `fenced` が true なので見出しとして読まれず、`inHeader` が false のあとは前文の行になる。`## Features` より後の節は `featuresLine != 0` で落ちる。`- Status:` などの行は前文の追加より前の `isStatusOrApprovedLine` が拒否するので、前文に digest の読まない行は入らない。`LoadSplitPlan` はファイルを 1 回読み、前文・`Digest` ともその同じ `data` から作る(`split.go:249-257`)ので、承認の digest と前文のあいだに読み直しの隙間はない。CRLF は `strings.TrimSuffix(raw, "\r")` のあとの行を足すので `\r` を持たない。`finishSplitPlan` の呼び出しは 1 か所(`:386`)で、引数の順(header、preamble、featuresLine、drafts)が宣言と合う
- `internal/org/feature.go` の diff を読んだ。`fenceFor` は最長のバッククォートの並び + 1(最低 3)で、前文の中のどの並びも外のフェンスを閉じられない(閉じるには同じ長さ以上が要る)。バイトの走査だが、バッククォートは ASCII なので UTF-8 の多バイト文字の途中に当たらない。`strings.TrimSpace(plan.Preamble) != ""` は、パーサが空白行を詰める前提では二重だが、手で組んだ `SplitPlan`(テスト)に備える。`prompts.go:87-100` の `strings.NewReplacer` は 1 回の走査なので、前文の中の `{{ORG_ID}}` や `{{TASK}}` は置き換わらない。`Spawn` の `Task` は role prompt の `{{TASK}}` にだけ入る(`spawn.go:1005`・`:2261`、`prompts.go:97`)ので、2,000 字の上限と shell の引用の問題は前文に及ばない
- `internal/org/prompts/leader.md` の diff を手順 1〜7 の全文と読んだ。手順 1(囲みの行ごと原文のまま写す)、手順 3(承認の前の reviewer も同じ driver と model)、手順 5(driver と model は spawn の引数、ほかの決まりは TASK の本文に写さず計画の節を読ませる)は、同じ語「全機能の座席に適用する決まり」を使い、`featurePreambleLead` の先頭と一字一句合う。手順番号の相互参照(手順 1 の「手順 5」、手順 3 の「手順 5」)は実在する。追加行の幅は最大 78 列で、周りの行(72〜78 列)と同じ
- テストの diff を全部読んだ。`TestLoadSplitPlan_Preamble` は前文の境界(フェンスの中の `## Features`、4 連の中の 3 連、チルダ、`###` とフィールドの形の行、`## Features` の末尾の空白、`## Features` より後の節、CRLF)を別々の行で打つ。`TestFeatureLeaderTask_Preamble` は空白行だけの前文、3 連・4 連の境界、本文なしを打つ。`TestStartFeature_LeaderPromptHoldsPreamble` は 2,000 字超の前文を `utf8.RuneCountInString` で作り、`## タスク` の節を次のテンプレートの見出しまでで切る(`markdownSection` が前文の中の見出しで終わるため、とテストの doc が理由を書く)。`TestRenderRolePrompt_Leader_FeatureOrgSeatRules` は 3 手順の文と順序を固める。`TestLoadSplitPlan_TrialPreamble` が引く `docs/evidence/org-trial-2026-10-20/splits/trial.md` の文字(`## 目的`、`座席の決まり(両方の機能に共通):`、`- 予約の外のファイルは編集しない`、2 機能の本文の先頭と末尾)を実ファイルと突き合わせ、すべて実在する(`splitTestRepoRoot` は `runtime.Caller` で root を引き、ほかのテストと同じ作り)
- start の help とエラーの diff(`internal/cli/org.go:918-922`、`:1044`)と、それを固めるテスト 2 か所(`TestOrgStartHelp_TaskSource`、`TestOrgStartPlan_FlagCombinationsRefused` の期待)を読んだ。help の改行は `strings.Fields` で畳んで比べるので折り返しに依らない
- skill の `org`・`director` の 4 写し(`.claude/skills/`、`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)が `cmp` でバイト一致なことを確かめた。`templates/base/` の追加行に `internal/`・`cmd/`・`docs/plans`・`docs/evidence`・`docs/tech-debt`・`docs/specs` の参照はない(`docs/plans/active/` は既存の行を直した 2 行の中の既存の語)。追加行の幅は中央値 70 列、最大 78 列
- 記録の diff を読んだ。tech-debt の変更行 11 本を `re.split(r'(?<!\\)\|', ...)` で割ると、表の行はすべて 7 要素(5 欄)で base と同じ形。F5 の行は取り消し線・`(RESOLVED ...)`・直上の RESOLVED コメント・Trigger 欄の `~~...~~ — RESOLVED` がそろい、新しい open の行(347)の Related 欄は「the F5 row above」で閉じた行を指す。`fix/seat-wip-checkpoint` は `git grep`(archive 済みの計画と過去の reports を除く)でこの PR の計画の 3 行だけになり、仕様・根拠の表・tech-debt から消えて `#250` になっている。根拠の表の変更行 5 本(24、30、62、102、103 行)の欄の数は base と同じ。`#250` の計画と 4 つの report は `test -f` で実在する。仕様の PR 一覧は 33 本で、NFR の行の 33 本・30 本(33 から #205・#241・#242 を除く)・25 本(30 から #246〜#250 を除く)と算術が合う。RESOLVED コメントが挙げるテスト名 7 本は、すべてこの diff にある
- 追加行を `/Users/`、`TODO`・`FIXME`、`fmt.Print`・`println`・`t.Log`、`sk-`・`ghp_`・`api_key`・`token=` で grep し、diff 全体を U+FFFD で調べた。当たりなし。`git diff --check a7bdc884...HEAD` は空
- 確かめていない: Go とシェルのテストの実行(`/test` の範囲)、`go vet`・`gofmt`・shellcheck・`check-sync`・`check-skill-sync`(`/verify` の範囲。run-verify が 7cc48f33 で rc 0 と依頼にある)、仕様の `[x]`・`[ ]` の数と根拠の表の突き合わせ、実機で leader が前文を計画に写すか

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | maintainability | `featurePreambleLead` の doc が、ブロックを前置きの語で呼ぶ手順を「手順 1 と 5」と書く。S2b(b10c6f25)が手順 3 にも同じ語を足したので、3 つ目が抜けている。同じ PR の `prompts_test.go` の doc は「All three name the block」と書き、手順 3 を数えている | `internal/org/feature.go:371-375`(「Steps 1 and 5 of the 機能ごとの org section ... name the block by its first words」)、`internal/org/prompts/leader.md:134-136`(手順 3 の「全機能の座席に適用する決まり」)、`internal/org/prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgSeatRules` の doc | 「Steps 1, 3 and 5」に直す。計画の D2(「leader.md の手順 5 の 1 文と同じ言葉」)は記録なので触らない |
| LOW (L2) | readability | `/org` skill の Leader のサイクル 1 の再折り返しが半端。既存の行に前文の 2 行を継ぎ足したため、43 列の短い行が 78 列の行の前に残る(周りの中央値は 70 列)。4 写しで同じ | `.claude/skills/org/SKILL.md:1217`(`囲みの行ごと写す(「分割計画」)。タスクの`)と次の 1218 行(`` `- 承認方式:` が `per-plan` なら、図解ページを書き、`PLAN:` と `VISUAL:` の ``)。写しは `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行 | 1215〜1220 行を折り直す。`scripts/sync-skills.sh` で写しを作り直し、`cmp` で 4 写しの一致を確かめる |
| LOW (L3) | typo | tech-debt の RESOLVED コメントの括弧が、1 つの slice の 2 コミットをコンマなしで並べる。同じ括弧の前の 3 件(`S1 03651616, S2 ef33170f, S2b b10c6f25,`)と、#250 のコメント(`S1 88706897, S2 416993cf, ... S3 3dedaef9, S4 b5b12ae9`)はコンマで区切るので、`S3 1b65bf49 546ad029` は 1 つの id に続く語に読める | `docs/tech-debt/README.md:322`(`(S1 03651616, S2 ef33170f, S2b b10c6f25, S3 1b65bf49 546ad029)`) | `S3 1b65bf49, 546ad029` にする。S4 の id はコメントを書くコミット自身なので、ないのは正しい |
| LOW (L4) | readability | 前置きの行の「原文のまま下に囲む」は、「囲む」が他動詞で目的語を取るので、「下に」と並ぶと読みにくい。この行は leader が毎回読み、手順 1 の「囲みの行ごと写す」の根拠になる。自然さの判断で、確認済みの誤りではない。文字は `feature.go` の定数と `feature_test.go` の 2 か所(`lead`)に同じ形で書かれているので、直すなら 3 か所を同じコミットで直す | `internal/org/feature.go:375`、`internal/org/feature_test.go` の `TestFeatureLeaderTask_Preamble` と `TestStartFeature_LeaderPromptHoldsPreamble` の `const lead` | 「原文のまま下のコードブロックに入れる」のように、囲むものを目的語に取る形にする。`prompts_test.go` は先頭の「全機能の座席に適用する決まり(」だけを引くので、直しの影響は受けない |

## Positive notes

- パーサの変更は 3 か所の追加(`preamble` の slice、見出しの行、本文の行)で、既存の判定(フェンス、`isStatusOrApprovedLine`、機能の本文)の文字を変えていない。前文は承認の digest がもとから覆う範囲で、読み込みも 1 回なので、承認と前文のあいだに隙間がない。`## Features` より後の節と先頭の自由文は今までどおりどこにも渡らず、拒否もしない(既存の分割計画が読める)
- `featureLeaderTask` は、前文が空ならバイトで前と同じタスクを返す。フェンスは前文の最長のバッククォートより 1 つ長い。前文は `NewReplacer` の 1 回の走査と role prompt のファイルを通るだけで、agmsg の 2,000 字の上限にも shell の引用にも触れない。座席への TASK には前文を写さず計画の節を読ませる(D5)ので、上限に当たる経路が閉じている
- テストが挙動と文字の両方を固める。境界の表は 1 行 1 事象で、4 連の中の 3 連、チルダ、`###` とフィールドの形の行、CRLF まで打つ。role prompt の節が 1 つずつ見つかることと、前文の `## 目的` が囲みの中にだけあることを別に確かめる。試行の実ファイルを読むテストが、計画の AC1 を実データで固める
- 記録は、できていないこと(leader が写したか・models に従ったかを ralph は確かめない、実機の確かめなし)を RESOLVED コメントと新しい open の行の両方に書く。open の行は、直す案(分割計画に ralph が読める driver と model の節を足す)と、それをこの PR でしなかった理由(D4、D5、Non-goals)を持つ。`#250` への書き換えは、仕様・根拠の表・tech-debt の 3 か所で数がそろう
- skill の 4 写しは `cmp` で一致し、`templates/base/` に repo 内部の参照は入っていない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(L1〜L4 はすべて文面の LOW)。依頼どおりこの報告書以外のファイルは変えていないので、`docs/tech-debt/README.md` には足していない。最後の cycle で直さない LOW が残れば、1 行にまとめて足す。この PR が自分で足した open の行(347 行、leader が前文に従ったかを ralph が確かめない)は、diff の一部として読んだ | - | - | - | `docs/plans/active/2026-10-22-split-preamble.md` |

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)
- Follow-ups: 4 件とも文面の LOW。L1 と L4 はコードの doc と leader に出す文字列を直すので、直すなら L4 の 3 か所(定数とテスト 2 か所)を同じコミットで直し、パイプラインを `/self-review` から回し直す(今は cycle 1 で、次が最後の cycle)。L2 は skill の 4 写し、L3 は tech-debt の記録の文字で、`post-implementation-pipeline.md` の例外(`/cross-review` だけが出す、この task の report と insight への修正)の対象ではないので、L1・L4 と同じ回し直しに入れる。直さない LOW は 1 行にまとめて tech-debt に足す。insight event は未実行(依頼が他のファイルを変えないとしたため): `./scripts/insights-append.sh --slug split-preamble --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 4 --cycle auto --source skill`
