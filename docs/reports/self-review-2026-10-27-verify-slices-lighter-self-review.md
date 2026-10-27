# Self-review report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(D1〜D6、AC1〜AC7、Progress checklist の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、2 回目。1 回目は 90f4ecca)
- Scope: `git diff 44c14b97...HEAD`(HEAD bc56cc4f、41 ファイル、+918/-118、10 コミット)。前回の所見への直し cb094ae8 と bc56cc4f を先に読み、そのあとで diff 全体をこの PR 自身が絞った 6 項目(naming、readability、null safety、exception handling、security、maintainability)で見直した。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、範囲外のファイル)は implementer が各スライスで確かめた前提で、目に入った分だけ載せる。テスト、静的検査、spec 適合、文書のずれは見ていない(`/verify`・`/test`・`/sync-docs` の範囲)。

## Previous findings

前回は MEDIUM 1、LOW 6(1 回目の report は 90f4ecca)。7 件とも直っている。直しの過程で出た新しい指摘は、下の Findings の 3 件。

| 前回の所見 | 状態 | 根拠 |
| --- | --- | --- |
| MEDIUM: 手順 7 が implementer の self-check の中身を見ない(`or report it` だけで、直らなかった指摘の行き先がない) | 解決 | `.claude/skills/implement/SKILL.md:42` が「mechanical self-check result present, with a result for each of its five items and no item found and not fixed」と中身まで言い、「found and not fixed」は別スライスで直してから次へ進む。`.claude/agents/implementer.md:40,48` と `.codex/agents/implementer.toml` は結果を clean / fixed / found and not fixed の 3 状態にし、`or report it` を消した。`.claude/rules/ralph/model-routing.md:43-48` が orchestrator の動きを書く。2 つのテストが各文を固める。残った穴は Findings の 1 行目(秘密情報で止めたとき) |
| LOW: 手順 6 の `before dispatching` と手順 7 の inline の順序(stage 前に `git diff --cached` が使えない) | 解決 | 手順 6 は `(for an inline slice, before implementing it)` を足した。手順 7 の inline は「検証 → stage → `git diff --cached` で確かめる → 直したら検証を打ち直して stage し直す → 結果を Progress checklist に書く → commit」の順で implementer と同じになった(`SKILL.md:43`)。テストが文を固め、「Then stage the slice's files and commit」の復活を禁じる |
| LOW: implementer の定義の「`/self-review` no longer checks the first four items」 | 解決 | 「These are the checks `/self-review` leaves to the implementing side, so do not skip them.」に変わり、履歴の言い方も位置の指定もない(`implementer.md:40`、`.toml` も同じ)。`no longer` と `first four` が戻らないよう `assert_not_contains_any_case` がある |
| LOW: Completion gate に `/verify` と `/test` の verdict がない、末尾の文がどの検証か不明 | 解決 | gate に `- [ ] /verify and /test (step 13) passed` を足し、末尾は `If a slice's verification or the pipeline's checks have not run, say so explicitly`(`SKILL.md:80-90`)。旧い文の復活も禁じるテストがある |
| LOW: 「機械的な 5 項目」の言い方が場所ごとに違う | 解決 | model-routing の report contract が正本で、各所が同じ並びで繰り返す。全文検索で `typos` / `debug leftover` / `files outside` を洗い、implementer(`.md`・`.toml`)、reviewer(`.md`・`.toml`)、`self-review/SKILL.md`、手順 7 の inline が正本と一語違わない。折り返しは `join_lines` が吸収する。正本の文自体と各所の文は 2 つのテストが同じ定数で固める |
| LOW: 6 項目のテストの範囲が狭い(reviewer の定義、report の Area、implementer の報告の欄、手順 7 の委譲側) | 解決 | `review_areas` / `area_values` が SKILL.md・reviewer.md・reviewer.toml・template.md の 3 形式を読み、旧 4 項目の語(`unnecessary typo debug secret`)を大文字小文字を区別せずに禁じる。implementer 4 ファイルの報告の欄、手順 7 の委譲側 2 文も固めた。`assert_part_lacks_any_case` は part が空なら失敗にするので、マーカー行を直し忘れても通らない |
| LOW: 同名の `assert_contains` / `assert_not_contains` が 2 つのテストで大文字小文字の向きが逆 | 解決 | `test-slice-verification.sh` の関数を `*_exact` / `*_any_case` に改名し、冒頭のコメントに向きの違いを書いた。`test-self-review-scope.sh` の既存の 2 つ(基準の 44c14b97 から同じ向き)は変えず、新しい関数を挙動で名付けてコメントで区別した。ほかの tests の同名の関数(`test-plan-visual.sh` など)も向きが違うので、コメントの「other tests」は正しい |

## Evidence reviewed

- 直しの 2 コミット(`git diff 90f4ecca bc56cc4f`)を root の定義・skill・rules・テストで通読した。`templates/base/` と `.agents/` の写しは `cmp` / `diff` で比べた
  - `.claude/agents/{implementer,reviewer}.md`、`.claude/skills/implement/SKILL.md`、`.codex/agents/{implementer,reviewer}.toml` は `templates/base/` の同名と完全に一致した
  - `.claude/rules/ralph/model-routing.md` は KNOWN_DIFFS の箇所(org receipts の節)だけが違う。cb094ae8 の hunk は root と `templates/base/` で同一
  - `.claude/skills/{implement,plan,self-review}` の `SKILL.md` と `template.md` は `.agents/skills/` と `templates/base/` の両方の写しと一致した。`self-review/SKILL.md` の違いは `allowed-tools` の 1 行だけ(`.agents` 側の生成の既存の挙動)
- 5 項目の語の洗い出し: `typos` / `debug leftover` / `files outside` を `.claude`・`.codex`・`.agents`・`docs/plans/templates`・`scripts` で grep し、すべての言及が正本と同じ並びだった。旧い `checked and clean` は残っていない
- TOML: `.codex/agents/implementer.toml` と `reviewer.toml` にバックスラッシュはなく、`"""` は開始と終了の 2 つだけ。`git diff 44c14b97...HEAD --check` は空
- 2 つのテストの helper(`review_areas`、`area_values`、`block`、`join_lines`、`assert_part_*`)の動きを awk の前方一致、`tr -s ' '`、`-F` の固定文字列で追った。reviewer.md は 1 行の Focus 行、reviewer.toml は `- ` の箇条書き、SKILL.md は番号付きで、いずれも 1 つの awk で同じ範囲に収まる
- 計画: bc56cc4f が変えたのは `## Progress checklist` だけ(digest が読まない節)
- 確かめていない: テストと静的検査の実行(計画の記録では scope 198/198、agent-models 47/47)、各テストの変異の再現

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | 「範囲内で消せない秘密情報はコミットせずに止める」例外(bc56cc4f)が implementer の定義にしかなく、orchestrator 側の文に届いていない。(1) 手順 7 の delegated と model-routing は「found and not fixed は別スライスで直してから次へ」とだけ言い、前提はコミット済みの指摘。止めたときはコミットがなく、「commit SHA が新しい `HEAD`」の確認も implementer の報告の commit SHA の欄も成り立たない。orchestrator が何をするか(別の handoff で消すか、人に聞くか)が書かれていない。(2) 手順 7 の inline は「別スライスで直す」のあとに `Then commit` と続くので、読んだ順に動くと秘密情報を含むままコミットできる。inline は orchestrator 自身が消せるので実害は小さいが、向きが implementer と逆。(3) `before the next slice` は最後のスライスでは参照先がない | `.claude/agents/implementer.md:40`(`do not commit; unstage it and stop`)、`.codex/agents/implementer.toml:50-52`。`.claude/skills/implement/SKILL.md:42`(delegated)・`:43`(inline、`fix an item found and not fixed as its own slice before the next slice. Then commit`)。`.claude/rules/ralph/model-routing.md:46-48`。報告の欄は `.claude/agents/implementer.md:48,50`(`mechanical self-check result` と `commit SHA`) | 手順 7 の delegated に 1 文足す。例: 「implementer が秘密情報でコミットせずに止めたとき(commit なし)は、同じく別の handoff で消すまで次へ進まない」。inline は「秘密情報は found and not fixed のままコミットしない」と一言足す。`before the next slice` は `(or step 12, after the last slice)` を足す。model-routing の 1 文と、テストで固める文もそろえる |
| LOW | readability | 計画の `## Progress checklist` に「Review artifact created」が 2 行ある。bc56cc4f が足した `[x]` の行(127 行目)の下に、元の `[ ]` の行(130 行目)が残っている。Completion gate の「progress checklist is fully updated」を満たさない形で、`[ ]` の行は review が未実施に読める。また 129 行目の「(このコミット)」は、計画が `/pr` で archive されたあとは参照先がない | `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md:127,130,129` | 元の `[ ]` の行を `[x]` にして 127 行目の説明をそこへ移し、127 行目を消す。「このコミット」は SHA に置き換える(今回の直しのコミットを待って書く) |
| LOW | maintainability | `join_lines` と `assert_joined_contains_exact` が 2 つのテストに同じ名前である。中身は同じではない: `test-slice-verification.sh` の版は `[ ! -f "$_file" ]` の分岐を持ち「no such file」と出すが、`test-self-review-scope.sh` の版は持たない。ファイルが消えたときは shell の redirect のエラーと「missing: 文」が出るだけで、原因が文の欠けに読める。前回の LOW(同名で向きが逆)と同じ種類の罠が、向きでなく guard の有無で残った | `tests/test-self-review-scope.sh:68-83`、`tests/test-slice-verification.sh:41-60` | `test-self-review-scope.sh` の版にも `[ ! -f ]` の分岐を足してそろえる。共有の helper ファイルは tests にまだないので、まずコピーの内容をそろえるだけでよい |

## Positive notes

- 5 項目の語は model-routing の正本から implementer(2 形式)、reviewer(2 形式)、self-review の skill、手順 7 まで同じ並びで届き、2 つのテストが同じ定数で固めている。1 か所だけ語を変えると落ちる
- 前回の MEDIUM の直しは、clean / fixed / found and not fixed の 3 状態を implementer の定義・手順 7・model-routing・テストの全部に同じ向きで置いた。`or report it` のような抜け道が文面から消えた
- `review_areas` は 3 形式の定義(SKILL.md、reviewer.md、reviewer.toml)を 1 つの awk で同じ範囲に切り出し、`assert_part_lacks_any_case` が空の part を失敗にする。マーカー行を直し忘れて何も照合しないまま通る、という穴がない
- `no longer` と `first four` を禁じる検査が、文の直しだけでなく、履歴を前提にした書き方の復活も防ぐ
- 今回の 2 コミットが触ったのはこの PR の範囲(定義、skill、rules、2 つのテスト、計画の Progress checklist)だけで、範囲外のファイルはない
- 前回から続く良い点: root と写しが KNOWN_DIFFS 以外で一致している、`/self-review` を絞った理由が skill・reviewer・report の雛形に同じ向きで書かれスクリプト名がない(D5)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

なし(サイクル 1 の中。LOW 3 件は `/verify` の前に 1 コミットで直せる。直さずに進める場合は triage に回す)。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 3 件で、どれも文言と計画の記録とテストの helper の直し)
- Follow-ups: 表の 1 行目(秘密情報で止めたとき)は、実装側の例外が orchestrator 側と食い違う点なので、`/verify` に進む前に直す価値がある。2 行目と 3 行目は同じコミットで直せる。`.goreleaser.yml:36-39` の changelog は `docs:` と `test:` を落とし、この PR の 10 コミットはすべてその 2 種なので、downstream に届く skill と agent の変更は自動の changelog に Merge 行しか出ない。計画の Rollout は release ノートを約束していないが、`/pr` の本文と release の issue に 1 行(「`/implement` がフルの検査を打たない」「`/self-review` が 6 項目になった」)を足す価値がある
