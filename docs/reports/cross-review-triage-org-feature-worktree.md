# Cross-review triage report: org-feature-worktree

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-09-org-feature-worktree.md(AC1〜AC14、S1〜S5、承認 digest 56e435bfa976)
- Self-review report: docs/reports/self-review-2026-10-09-org-feature-worktree.md(かけ直しのあと Merge 可、LOW の L5・N1〜N3)
- Verify report: docs/reports/verify-2026-10-09-org-feature-worktree.md(pass、V-1〜V-4。V-1・V-2・V-4 は sync-docs で直した)
- Implementation context summary: review は HEAD 6312b5d6 に対して read-only の sandbox で動いた。Go のテストは sandbox が一時ディレクトリを作れず回っていない(本人の申告)。S1 の implementer は、コードフェンスの中の行も見出しやフィールドとして読むと報告していた(計画の進捗の S1 の行)。leader の task の文(`featureLeaderTask`)には台帳の場所を書いていない

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `start --plan` を既定でない `--state-dir` で打つと、leader の座席と予約はその台帳に記録されるが、leader への引き継ぎに台帳の場所がない。雛形は leader に素の `ralph org spawn/report/disband` を打たせるので、leader の座席は既定の台帳を使い、座席が別の台帳に入って予約と上限の数え方を外れ、打った人の status と片付けからも見えなくなる。ralph の版が同じでも起きる | 本物の問題と判断した。`featureLeaderTask` の行は分割計画・機能・worktree・ブランチ・予約・依存だけで、台帳を書かない。`RALPH_ORG_STATE_DIR` を打った人の環境で付けても、herdr の pane はサーバーの環境を引き継ぐので leader には届かない。既定の台帳を使うときは今の解決で同じ台帳になるが、`--state-dir` か env を使った start で台帳が分かれる。直すのは安い。leader の task に台帳の絶対パスを書き、`ralph org` のコマンドに `--state-dir` を付けさせればよい。そうすれば、tech-debt に送った「pane の ralph が古い版だと台帳が分かれる」(実機の run 1)も `--state-dir` の明示で防げる | internal/org/feature.go(featureLeaderTask、startFeatureLeaderParams)、internal/org/prompts/leader.md、/org skill |
| 2 | [P2] 機能の本文のコードフェンスの中に `## Usage` のような行があると、`split.go` の見出しの判定が無条件なので、そこで機能の節(と `## Features`)が閉じる。承認の検査は通るのに、例の残り・受け入れ条件・後ろの機能が読み込みから落ち、leader の task に入らない | 本物の問題と判断した。digest は本文の全体を読むので承認は通るが、leader に渡る本文は黙って短くなる。承認した内容と実行に渡る内容が食い違うのは、計画の Codex plan advisory 2 と同じ種類の穴。S1 の implementer が「コードフェンスの中の行も見出しやフィールドとして読む」と決めたまま残っていた。直すのは安い。` ``` ` と `~~~` のフェンスを数え、中の行は見出しにもフィールドにもしない。digest が読まない行(`- Branch:` など)の拒否は、digest がフェンスを知らないので、フェンスの中でも今どおり拒否する | internal/org/split.go(parseSplitPlan)、internal/org/split_test.go、/org skill の分割計画の形式の説明 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
