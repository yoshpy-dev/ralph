# Walkthrough: org-feature-worktree-ownership

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-10-org-feature-worktree-ownership.md(PR 作成時に active から移す)
- Branch: fix/org-feature-worktree-ownership(base be050681、PR #216 の後続)

差分は 15 ファイル、+1570/-147 で、コードは `internal/org/feature.go`(+173)だけになる。残りはテスト(+676)、`/org` skill の 4 面、レポートと記録。

## 読む順

1. `featureCanonicalRef`(feature.go)。worktree の記録の `canonical_ref` を `split:<分割計画の絶対パス>#<slug>` にした。パスは `ResolveSplitPlanPath` が symlink を解決した値で、台帳ごとに違う。4 段目の cross-review cycle 2 の WORTH_CONSIDERING #1(別の台帳の同じ名前の分割計画が、残った worktree を引き継げる)を閉じる
2. `featureWorktreeMismatch`。照合の規則はここ 1 か所にある。比べるのは `canonical_ref`・パス・ブランチ・kind で、ディレクトリと実際のチェックアウトも見る
3. `checkFeatureWorktreeReuse`(`ensure` の前)と `checkEnsuredFeatureWorktree`(`ensure` のあと、`Spawn` の前)。どちらも 2 の規則を呼び、エラー文だけを出し分ける。後者が WORTH_CONSIDERING #2(別の分割計画からの同時の start)を閉じる。拒否のときは leader を立てず、台帳にも書かない
4. `otherRecordFix`。記録がぶつかったときの案内を作る
   - org_id が slug と同じとき: slug を変えて承認し直す
   - `--org-id` を明示したとき: 別の `--org-id` を使うか外す。ブランチも同じなら slug も変える
   - `ensure` のあとの拒否: `cleanup` と別の `--org-id` を勧めない。先の org がその worktree とブランチを使っているかもしれないため
5. `ensureFailureErr` の「ブランチがすでにある」。slug(か Type)の変更を先に書き、名前の変更や削除は、何も使っていないと確かめたときに限る
6. テスト(feature_test.go)。本物のスクリプトのテストが 4 本ある
   - `_OtherLedger`: 2 つの台帳
   - `_OrgIDOtherPlan` と `_DefaultOrgIDOtherPlan`: `ensure` の中で別の start が走る
   - `RealWorktreeScript` の「ブランチがすでにある」のケース
   - あわせて、案内どおりに直すと通り、先の org の記録・ブランチ・チェックアウトが変わらないことを確かめる
7. `/org` skill(4 面): `start --plan` の手順に 5(`ensure` のあとの照合)を足し、拒否の一覧を新しい案内に合わせた
8. `docs/tech-debt/README.md`: 同時の start の行を更新し、残りの LOW と test の穴を足した。`docs/insights/events/` には、この PR と 4 段目の cross-review の記録を足した(4 段目の分は抜けていた)

## パイプライン

- self-review: MEDIUM の F-1(`--org-id` を明示した start で slug の変更を案内しても効かない)を 52800bae で直し、かけ直しで Merge 可
- verify: pass。test: pass(mutant 15 件をすべて検出)。cross-review: 指摘 0
