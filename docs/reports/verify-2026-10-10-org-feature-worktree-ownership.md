# Verify report: org-feature-worktree-ownership

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-org-feature-worktree-ownership.md(承認済み、digest feadcc05c21f)
- Verifier: verifier subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: 仕様への適合(AC1〜AC6 と AC3b)、静的解析、文書のずれ。対象は `git diff be050681...HEAD`(HEAD b0b0e640、8 コミット、10 ファイル、+1120/-147)。cross-review-triage-org-feature-worktree.md の cycle 2 の WORTH_CONSIDERING #1・#2 が閉じたかも確かめた。テストは実行していない(`/test` の担当)。テストは、何を固定しているかを読んで確かめた
- Evidence: `docs/evidence/verify-2026-10-10-org-feature-worktree-ownership.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。runner 自身のログは `docs/evidence/verify-2026-10-10-073902.log`(時刻は UTC)

## Spec compliance

計画の承認: `./scripts/plan-visual.sh digest` は HEAD で `feadcc05c21f` を返し、`- Approved:` の値と一致する。承認のあと計画を触ったコミットは c8fe902d と fc50ad10 の 2 つで、どちらも AC のチェックボックス(digest は `- [x]` を読まない)と `## Progress checklist` の中だけを変えている。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `canonical_ref` が `split:<分割計画の絶対パス>#<slug>`、同じ計画の同じ機能は承認し直しても同じ worktree | 満たす | `featureCanonicalRef`(`internal/org/feature.go:152`)は `"split:" + splitPath + "#" + slug` を返し、呼び出しは `StartFeature` の 209 行の 1 か所で、引数は `plan.Path`。`plan.Path` は `ResolveSplitPlanPath`(`split.go:171-200`)が `filepath.EvalSymlinks` で解決した値で、`LoadSplitPlan` は同じ値に `filepath.Abs` をかけて持つ(`split.go:230`、`246`)。`TestStartFeature_CanonicalRefIsTheResolvedPlanPath`(`feature_test.go:333`)が、文字列の形と、symlink 経由の `--plan` が解決後のパスで記録され、直接のパスで打ち直すと使い回すことを見る。承認し直しは `TestStartFeature_AfterDisband` の「the same feature approved again」(775 行)。空白を含むパスは本物のスクリプトのテスト `TestStartFeature_RealWorktreeScript_OtherLedger`(`ledger b`、1483 行)で、記録の読み戻しと打ち直しまで通る |
| AC2 別の台帳の同じ id・同じ slug は使い回さずに拒否(本物のスクリプト) | 満たす | `TestStartFeature_RealWorktreeScript_OtherLedger`(`feature_test.go:1483`)。台帳 A で start と disband をしたあと、台帳 B の同じ id と slug の start が `canonical_ref` の違いで拒否される。呼ばれるのは `lookup:org-auth-core` だけで `ensure` はなく、台帳 B のスナップショットは空 |
| AC3 `ensure` のあとに記録が別の計画のものなら `Spawn` の前に拒否。leader なし、台帳に書かない。案内は slug の変更か待つことで、`cleanup` と別の `--org-id` を出さない | 満たす(計画からのずれは Progress に記録済み) | `checkEnsuredFeatureWorktree`(`feature.go:443-469`)の呼び出しは 227 行で、`o.Spawn`(232 行)より前にある。拒否は `refuse` を通るので outcome は `rejected`、台帳には何も書かない。`TestStartFeature_RecordChangedDuringEnsureRefused`(951 行)が、`ensure` の中で別の計画の記録を作り、エラー全文、`cleanup` と `--org-id` が文に出ないこと、呼び出しの列(lookup、ensure、lookup)、スナップショットが空であること、相手の記録が変わらないことを見る。記録が読めない・消えた・ブランチが読めない場合の「run start again」も同じテストにある。計画からのずれは 2 つで、どちらも Progress の 118 行と 120 行に書いてある。(1) 案内は「slug(か Type)」ではなく slug だけ(Type だけ変えても org_id が変わらず同じ記録に当たるため)。(2) `--org-id` を明示した start では、別の `--org-id`(または `--org-id` を外す)を案内する(self-review F-1 の直し)。この枝は `TestStartFeature_OtherRecordWithOrgID`(1019 行)と、本物のスクリプトの `TestStartFeature_RealWorktreeScript_OrgIDOtherPlan`(1542 行)が見る。既定の org_id で `ensure` と競合する場合を本物のスクリプトで通すテストはない(Coverage gaps) |
| AC3b 使い回しの拒否とブランチ衝突の案内は slug の変更を先に書き、`cleanup` やブランチの変更・削除は使っていないときに限る。slug を変えて承認し直すと通り、先の org の状態は変わらない | 満たす | 使い回しの拒否で `canonical_ref` が違う場合は `feature.go:417-421` で、`otherRecordFix` を先に置き、`cleanup` は「if that worktree and its branch are no longer needed」の後ろにある。ブランチ衝突は `ensureFailureErr` の 279 行の枝で、slug(か Type)の変更を先に書き、`git branch -m` / `-D` は「only once you have made sure nothing uses that branch」のあと。`TestStartFeature_WorktreeRecordMismatchRefused`(672 行)の `otherRefHint` と `TestStartFeature_EnsureFailureRefused`(846 行)の `branchInTheWayHint` が全文で固定する。本物のスクリプトの `_OtherLedger` は、slug の変更が `cleanup` より前にあることを文字の位置で比べ、slug を `auth-store` にして承認し直した start が通ること、先の org の記録・ブランチの commit・チェックアウトが変わらないこと(`featureWorktreeState`)を見る。`_OrgIDOtherPlan` は `--org-id` を明示した場合に同じことを見る(`last_seen_at` だけを除いて比べる) |
| AC4 既存の `StartFeature` と CLI のテストは新しい形の期待で通る | 読んだ範囲では満たす(実行は `/test`) | `internal` 以下のテストで古い形 `split:<id>#<slug>` を期待するのは、PR #216 の記録を拒否するケース(`feature_test.go:691-693`)だけで、意図した使い方。ほかの期待は `st.ref` / `splitRef` を通して新しい形を作る。`internal/cli/org_feature_test.go` は `split:` を 0 件しか含まず、`canonical_ref` の形に依存しない |
| AC5 `/org` skill(4 面)と tech-debt の行が新しい挙動と合い、`check-skill-sync.sh` と `check-sync.sh` が通る | 一部(tech-debt の行が古い。`/sync-docs` で直す予定) | skill の 4 面は `cmp` で同一。手順の 5(389〜392 行)と 6、拒否の一覧(418 行から)の各枝は、`feature.go` のエラー文と合う(下の「Documentation drift」)。`check-skill-sync.sh` は 13 skill が lock-step、`check-sync.sh` は DRIFTED 0。tech-debt の同時の start の行(`docs/tech-debt/README.md:192`)の RESOLVED の注は、F-1・F-2 の直しのあとの案内と合わない(V-1) |
| AC6 `./scripts/run-verify.sh` が通る | 静的な部分は満たす(テストの部分は `/test`) | `./scripts/run-static-verify.sh` は rc 0。`run-verify.sh` のテストの部分はこの段では回していない |

### cross-review cycle 2 の WORTH_CONSIDERING

| # | 現況 | 根拠 |
| --- | --- | --- |
| 1 別の台帳の同じ名前の分割計画が、残った worktree を引き継ぐ | 閉じた | `canonical_ref` に分割計画の解決済みの絶対パスが入り、分割計画は `<state dir>/splits/` の直下に限られる(`ResolveSplitPlanPath` の 182 行の検査)ので、台帳が違えば値が違う。`TestStartFeature_RealWorktreeScript_OtherLedger` が本物のスクリプトでこの場面を再現し、拒否されることを見る |
| 2 別々の分割計画からの同時の start で、台帳の結びつきと worktree の記録が食い違う | 閉じた(同じ分割計画どうしの同時の start は残る。計画の Non-goals どおり) | `checkEnsuredFeatureWorktree` が `ensure` のあと `Spawn` の前に記録を読み直す。`scripts/ralph-worktree.sh` の `ensure_worktree` は、同じパス・ブランチ・種類の記録があれば `update_last_seen`(`.last_seen_at` だけを `jq` で書き換える)をして返すので、照合と `Spawn` の間に `canonical_ref` は変わらない(計画の前提どおり)。ブランチが違えば `state collision` で `ensure` が止まる。先読みと `ensure` の直列化はしておらず、tech-debt の行に残っている |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | changed scope。shellcheck、hook の `sh -n`(root と templates の 20 本)、guard の awk の構文、`jq -e` の settings 2 つ、Codex の hook の 3 つのガード、`check-sync.sh`(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の計画の参照。golang は `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、`staticcheck` は出力なし(どちらも導入済みで、skip ではない)。`secret-scan-branch` は be050681..b0b0e640 で clean |
| `cmp` で skill の 4 面 | 同一 | `.claude/skills/org/SKILL.md` と `.agents/`、`templates/base/.claude/`、`templates/base/.agents/` |
| `./scripts/plan-visual.sh digest <plan>` | `feadcc05c21f` | `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の `start --plan` の手順(4 面) | 合っている | 手順の 5(`ensure` のあとの照合)と 6(spawn)はコードの順(`feature.go:211` → `215` → `227` → `232`)と同じ。末尾の「6 のロックの下で拒否されたとき」も繰り下げに合っている |
| `/org` skill の「start が拒否するもの」(4 面) | 合っている(言い方の小さな食い違いが 2 つ) | 使い回しの拒否の 3 つの枝(org_id が slug、`--org-id` を渡した、それ以外)は `otherRecordFix` と `refuse` の文と合う。`ensure` のあとの照合の 2 つの枝(別の計画の記録、それ以外は打ち直し)と、`cleanup` を出さないこと、既定の org_id では別の `--org-id` を出さないことも合う。ブランチ衝突の枝は `ensureFailureErr` の文と合う。食い違いは V-2(「`--plan` をどう書いても」、436 行)と V-4(443 行の見出し) |
| `docs/tech-debt/README.md` の同時の start の行(192 行) | ずれている | V-1。self-review の N1 と同じ指摘で、`/sync-docs` が直す予定 |
| `docs/tech-debt/README.md` の F-3 の行 | まだない | V-2。計画の Progress(120 行)は「F-3 は tech-debt に送る」と書く。`/sync-docs` が足す予定 |
| `ralph org start --help`(`internal/cli/org.go:540-580`) | 合っている | `canonical_ref` に触れない。「Running the same start again reuses the worktree」は今も正しい |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:48` | 合っている | 使い回しの条件を `canonical_ref`・パス・ブランチ・kind・チェックアウトで述べ、値の形には触れない |
| `feature.go` の先頭の手順のコメント(22〜45 行) | 合っている | 手順 5 と 6、「before 6」の表記はコードの順と合う |
| 計画の Risks の 2 つ目(101 行) | 少しずれている | V-3 |
| archive 済みの計画 `2026-10-09-org-feature-worktree.md` の 31〜32 行 | 対象外 | 古い形 `split:<id>#<slug>` を書くが、当時の記録なので直さない |

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V-1 | LOW | doc drift | tech-debt の同時の start の行の RESOLVED の注が、F-1・F-2 の直しのあとの案内と合わない。注は「the error points at changing this feature's slug and approving again or at waiting until the other org is done, and at neither the cleanup nor another `--org-id`」と書く。今のコードは、`--org-id` を明示した start では別の `--org-id`(か `--org-id` を外すこと)を案内し、待つ相手を「the owner of the other org (org_id %s) removes that worktree once the org's PR is merged」と書く。self-review の N1 と同じ | `docs/tech-debt/README.md:192`、`feature.go:460-464`、`feature.go:481-493` | `/sync-docs` で、注の該当の 1 文を、既定の org_id のときと `--org-id` を渡したときの 2 通りと、新しい待つ相手に合わせて直す |
| V-2 | LOW | doc drift | `featureCanonicalRef` の doc(「the same plan has the same canonical_ref however --plan names it」)と skill の「`--plan` をどう書いても同じ計画なら同じになり」は、symlink と相対パスについてだけ正しい。`filepath.EvalSymlinks` は大文字小文字を正規化しないので、大文字小文字を区別しない FS で `--state-dir` の綴りが前回と違うと別の値になる(拒否の側に倒れる)。self-review の F-3 で、tech-debt の行はまだない | `feature.go:146-151`、`.claude/skills/org/SKILL.md:434-437`(4 面とも同じ) | `/sync-docs` で F-3 の tech-debt の行を足す。doc と skill の言い方は、その行で「symlink と相対パスは解決するが、大文字小文字の違いは別の値になる」と述べるか、次に該当の段を触る変更で直す |
| V-3 | LOW | doc drift(計画の記述) | 計画の Risks の 2 つ目は、分割計画のファイルを移した場合に「エラー文が `cleanup` か別の `--org-id` を案内する」と書く。AC3b のあとの使い回しの拒否(`canonical_ref` が違う場合)は、既定の org_id では slug の変更と、要らないときの `cleanup` を案内し、別の `--org-id` は出さない。計画の Scope(27 行)と Design decisions とは合っているので、Risks の 1 行だけが古い | 計画の 101 行、`feature.go:417-421`、`feature.go:482-485` | `/verify` は計画を書き換えない。`/pr` の前に計画を触る機会があれば Risks の文を直す。直さなくても、Scope と AC3b が挙動を正しく書いているので害はない |
| V-4 | LOW | doc drift(skill の中の言い方) | skill の使い回しの拒否の枝の見出しは「org_id が slug のとき(`--org-id` を渡さない既定)」、`ensure` のあとの枝は「org_id が slug と同じなら」と書き、2 か所で条件の言い方が違う。コードの条件は `orgID == slug`(値が同じ)なので、後者がコードと合う。`--org-id` に slug と同じ値を渡した start は前者の見出しに当てはまらないが、コードは slug の変更を案内する。self-review の N3 と同じ根で、N3 は tech-debt に送られる | `.claude/skills/org/SKILL.md:443`、`:477`、`feature.go:482` | N3 の扱いに合わせる。コードを直さないなら、443 行の見出しを「org_id が slug と同じとき(`--org-id` を渡さない既定など)」にすると 2 か所が揃う |

参考(指摘にはしない):

- `feature.go:430` とテストのコメント 9 か所が `docs/plans/archive/2026-10-10-org-feature-worktree-ownership.md` を指すが、計画はまだ `active/` にある。`/pr` が計画を archive に動かすと正しい参照になる。同じ 2026-10-09 の計画の `active/` 参照が 6 か所残る件は self-review の N4 で、tech-debt に送られる
- AC3 の本文(「slug(か Type)」「別の `--org-id` を案内しない」)は計画からのずれのとおりには書き換わっていないが、ずれは Progress の 118 行と 120 行に記録されている

## Observational checks

- `scripts/ralph-worktree.sh` の `ensure_worktree`(261〜353 行)と `update_last_seen`(251〜259 行)を読み、計画の前提 2 つ(既存の記録を返すときに変わるのは `last_seen_at` だけ、`--canonical-ref` は argv から `jq --arg` で書く)を確かめた
- `internal/cli` の `runOrgStartPlan` は `res.Spawn.Err` をそのまま返すので、`ensure` のあとの拒否もほかの拒否と同じく非 0 で終わる(skill の「終了コード 1」)
- `featureCanonicalRef`・`checkEnsuredFeatureWorktree`・`otherRecordFix`・`featureWorktreeMismatch` を `feature.go` の外から呼ぶのはテストの `featureCanonicalRef` の 1 か所だけ

## Coverage gaps

- テストは実行していない。AC1〜AC4 と AC3b は、テストが何を固定しているかを読んで確かめた。通るかどうかは `/test` が確かめる
- 既定の org_id(slug)で `ensure` と競合する場合は fake の `Ensure` でだけ通る(`TestStartFeature_RecordChangedDuringEnsureRefused`)。本物のスクリプトでの競合は `--org-id` を明示した `_OrgIDOtherPlan` だけ。fake は `ensure_worktree` と同じく `canonical_ref` を見ずに既存の記録を返すことをコードで確かめたので、おそらく本物でも同じ挙動になる。未確認です
- 2 つの start を本当に並行で走らせるテストはない(計画の Non-goals、tech-debt の行に残る)
- この PR のバイナリでの実機の確認(herdr の上の leader)はしていない。計画も求めていない
- 大文字小文字を区別しない FS での `--state-dir` の綴りの違い(V-2)は、self-review の probe 以上には確かめていない

増やすと確信が最も上がる確認: `/test` で、変更を 1 つずつ戻す 2 つの mutant を回す。(1) `feature.go:227` の `checkEnsuredFeatureWorktree` の呼び出しを外すと、`TestStartFeature_RecordChangedDuringEnsureRefused` と `_RealWorktreeScript_OrgIDOtherPlan` が落ちること。(2) `featureCanonicalRef` の引数を `plan.ID` に戻すと、`_RealWorktreeScript_OtherLedger` が落ちること。どちらも落ちれば、WORTH_CONSIDERING #1・#2 の直しがテストで固定されていると言える。

## Verdict

pass

- Verified: AC1、AC2、AC3(計画からのずれは Progress に記録済み)、AC3b はコードとテストの読み合わせで満たす。WORTH_CONSIDERING #1 は閉じ、#2 は別々の分割計画の場合について閉じた。`./scripts/run-static-verify.sh` は rc 0。skill の 4 面は同一で、コードのエラー文と合う。計画の承認 digest は一致する
- Partially verified: AC4(読んだ範囲。実行は `/test`)、AC5(tech-debt の行が古い。V-1 と V-2 は `/sync-docs` で直す予定)、AC6(静的な部分だけ)
- Not verified: テストの実行、本物のスクリプトでの既定の org_id の競合、並行の start、実機
