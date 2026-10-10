# Cross-review triage report: org-feature-worktree

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-09-org-feature-worktree.md(AC1〜AC14、承認 digest 56e435bfa976)
- Self-review report: docs/reports/self-review-2026-10-09-org-feature-worktree.md(cycle 2、Merge 可、LOW の C2-1〜C2-6)
- Verify report: docs/reports/verify-2026-10-09-org-feature-worktree.md(cycle 2、pass、V2-1〜V2-3)
- Implementation context summary: cycle 1 の ACTION_REQUIRED 2 件は d49bbc34(leader の task に台帳の行、分割計画のコードフェンス)、07d38e6d(main の台帳を指す `--state-dir` でも main の上限を使う)、9c1d447f(その help)で直した。実機の Run 4 で、既定でない台帳でも leader が `--state-dir` を付けることを確かめた。この回の review は HEAD e8ff0fdb に対して read-only の sandbox で動き、Go のテストは回っていない(本人の申告)。worktree の記録の `canonical_ref` は `split:<分割計画の id>#<slug>` で、id はファイル名から `.md` を除いたもの。`scripts/ralph-worktree.sh ensure` は既存の記録のパス・ブランチ・kind だけを比べ、`canonical_ref` は見ない

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 同じ repo で 2 つの `--state-dir` を使い、それぞれの台帳に同じ名前の分割計画(`splits/auth.md`)と同じ slug・Type の機能があると、1 つ目の org を disband したあとに残る worktree を、2 つ目の start が使い回しの検査を通って引き継ぐ。どちらも `canonical_ref` が同じになるため。2 つ目の org は別の計画のチェックアウトとコミットを受け継ぐ。持ち主の記録に分割計画のパスか台帳の名前空間を入れるべき | 本物の問題と判断した。`featureCanonicalRef` は分割計画の id と slug だけで作るので、別の台帳の同じ名前の分割計画と見分けられない。起きるのは、同じ repo で台帳を 2 つ以上使い(1 段目の「repo に共通の台帳 1 つ」から外れる使い方)、2 つの台帳に同じ名前の分割計画と同じ slug があり、前の org の worktree を消していないときに限られる。直すのは安く、`canonical_ref` に分割計画の絶対パスを入れれば足りる。ただパイプラインの上限に達しているので、直すかどうかはユーザーが決める | internal/org/feature.go(featureCanonicalRef、checkFeatureWorktreeReuse) |
| 2 | [P2] 別々の分割計画から同じ機能・org_id の start を同時に打つと、どちらも最初の記録の確認を通る。先に作った方の記録を、後の方の `ensure` がそのまま返す(スクリプトはパス・ブランチ・kind しか見ない)。後の方が先に `Spawn` に着くと、台帳の結びつきは自分の計画、worktree の記録は相手の計画を指し、打ち直しが失敗し、別の計画がそのコミットを使い回しうる。`ensure` のあとに記録を確かめ直すか、確認と作成を直列にすべき | 本物の問題と判断した。`StartFeature` は `ensure` の前に一度だけ記録を確かめ、`ensure` が返したあとはパスが同じかしか見ない。tech-debt に送った「同じ機能の `start --plan` の同時実行」(計画の進捗の (4))と同じ競合で、相手が別の分割計画のときに結びつきと記録が食い違う点が新しい。起きるのは、同じ org_id の start を別の分割計画から同時に打ったときだけ。直すのは安く、`ensure` のあとに `checkFeatureWorktreeReuse` を打ち直して、違えば `Spawn` の前に拒否すればよい(確認と作成の直列化はスクリプトにロックがないので重い)。パイプラインの上限に達しているので、直すかどうかはユーザーが決める | internal/org/feature.go(StartFeature) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## 付録 A: cycle 1 の仕分け

- Cycle 1 counts: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0
- review は HEAD 6312b5d6 に対して動いた。ユーザーが「直す」を選び、d49bbc34・07d38e6d・9c1d447f で直したあと、パイプラインを 2 回目として回した

### Triage context(cycle 1)

- Self-review report: cycle 1 のかけ直しのあと Merge 可、LOW の L5・N1〜N3
- Verify report: cycle 1 は pass、V-1〜V-4(V-1・V-2・V-4 は sync-docs で直した)
- Implementation context summary: S1 の implementer は、コードフェンスの中の行も見出しやフィールドとして読むと報告していた(計画の進捗の S1 の行)。leader の task の文(`featureLeaderTask`)には台帳の場所を書いていなかった

### ACTION_REQUIRED(cycle 1)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `start --plan` を既定でない `--state-dir` で打つと、leader の座席と予約はその台帳に記録されるが、leader への引き継ぎに台帳の場所がない。雛形は leader に素の `ralph org spawn/report/disband` を打たせるので、leader の座席は既定の台帳を使い、座席が別の台帳に入って予約と上限の数え方を外れ、打った人の status と片付けからも見えなくなる。ralph の版が同じでも起きる | 本物の問題と判断した。`featureLeaderTask` の行は分割計画・機能・worktree・ブランチ・予約・依存だけで、台帳を書かない。`RALPH_ORG_STATE_DIR` を打った人の環境で付けても、ralph はそれを herdr の pane に渡さない。既定の台帳を使うときは今の解決で同じ台帳になるが、`--state-dir` か env を使った start で台帳が分かれる。直すのは安い。leader の task に台帳の絶対パスを書き、`ralph org` のコマンドに `--state-dir` を付けさせればよい。そうすれば、tech-debt に送った「pane の ralph が古い版だと台帳が分かれる」(実機の run 1)も `--state-dir` の明示で防げる。d49bbc34 で直した | internal/org/feature.go(featureLeaderTask、startFeatureLeaderParams)、internal/org/prompts/leader.md、/org skill |
| 2 | [P2] 機能の本文のコードフェンスの中に `## Usage` のような行があると、`split.go` の見出しの判定が無条件なので、そこで機能の節(と `## Features`)が閉じる。承認の検査は通るのに、例の残り・受け入れ条件・後ろの機能が読み込みから落ち、leader の task に入らない | 本物の問題と判断した。digest は本文の全体を読むので承認は通るが、leader に渡る本文は黙って短くなる。承認した内容と実行に渡る内容が食い違うのは、計画の Codex plan advisory 2 と同じ種類の穴。直すのは安い。` ``` ` と `~~~` のフェンスを数え、中の行は見出しにもフィールドにもしない。digest が読まない行(`- Branch:` など)の拒否は、フェンスの中でも今どおり拒否する。d49bbc34 で直した | internal/org/split.go(parseSplitPlan)、internal/org/split_test.go、/org skill の分割計画の形式の説明 |
