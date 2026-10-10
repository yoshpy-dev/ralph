# sync-docs report: org-feature-worktree-ownership

## Cycle 1

- Date: 2026-10-10
- Plan: `docs/plans/active/2026-10-10-org-feature-worktree-ownership.md`(承認済み、digest `feadcc05c21f`。編集の前後で同じ値)
- Pipeline cycle: 1(上限 2)。差分は origin/main(`be050681`)から branch HEAD `0ec79aae`(fix/org-feature-worktree-ownership)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-10-org-feature-worktree-ownership.md`(`7bd95ebd`、`b0b0e640` で再実行の結果に更新。Merge 可、CRITICAL 0・HIGH 0・MEDIUM 0、LOW は F-3 と N1〜N5)、
  `docs/reports/verify-2026-10-10-org-feature-worktree-ownership.md`(`9bd90993`。pass、V-1〜V-4)、
  `docs/reports/test-2026-10-10-org-feature-worktree-ownership.md`(`0ec79aae`。pass、Test gaps 5 件)

## Summary

古くなっていたのは、tech-debt の同時の start の行の注(self-review N1、verify V-1)、`/org` skill の 2 か所(verify V-2 と V-4)、plan の進捗のチェックボックスだった。tech-debt に入れていなかった項目は 3 行にして足した(F-3 と V-2、N2〜N5、test report の残りの gap)。コードは触っていない。

差分の大きさ(`git diff origin/main...HEAD --stat`、`0ec79aae` 時点、この sync-docs の commit を含まない): 12 files changed, 1417 insertions(+), 147 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(同時の start の行、RESOLVED の注。N1、V-1) | `internal/org/feature.go` の `checkEnsuredFeatureWorktree` と `otherRecordFix` を読んで、52800bae のあとの案内に合わせた。(1) 直した commit に 52800bae(案内の分)を足した。(2) 「`cleanup` も別の `--org-id` も案内しない」を、「`cleanup` は案内しない。案内は `otherRecordFix` の直し方か、先の org の持ち主がその org の PR の merge のあとに worktree を消すのを待つこと」に直した。(3) 既定の org_id(slug と同じ値)では slug の変更だけを案内し、別の `--org-id` は案内しない。明示した `--org-id` が別の値のときは別の `--org-id`(か、外して slug を org_id にすること)を案内し、記録が同じブランチにもあるときは slug の変更も足す。slug と同じ値を明示した start は既定の枝に入る、と書いて、小さな指摘の行の (b) を指した。(4) Related に self-review の F-1・F-2・N1、verify の V-1、計画の Progress を足した。同じ行の「Still open」の文は変えていない |
| `docs/tech-debt/README.md`(新しい行 1。F-3、V-2) | `canonical_ref` の大文字小文字。`ResolveSplitPlanPath` の `filepath.EvalSymlinks` は symlink と相対パスを解決するが大文字小文字は直さない(self-review の probe: macOS の APFS で `Foo/a.md`・`foo/a.md`・`FOO/A.MD` が渡した綴りのまま返る)。大文字小文字を区別しない FS で綴りを変えて打つと別の `canonical_ref` になり、使い回しの検査が別の分割計画のものとして拒否する(fail closed)。`featureCanonicalRef` の doc の「`--plan` の書き方によらず同じ」は相対パスと symlink にだけ正しい。トリガーは `featureCanonicalRef` か `ResolveSplitPlanPath` の次の変更、または 2 つの `canonical_ref` が大文字小文字だけ違う拒否の報告 |
| `docs/tech-debt/README.md`(新しい行 2。N2〜N5) | org-feature-worktree-ownership の小さな指摘を 1 行にした。(a) N2: `feature_test.go` のテストの doc 2 か所。(b) N3: `--org-id` に slug と同じ値を明示した start は既定の枝に入り、効かない slug の変更を案内されて、次の拒否で正しい案内に着く。(c) N4: 古い `active/` の参照を持つ Go のコメント 6 か所(`split.go`、`reserve.go`、`internal/cli/org.go` の `newOrgStartCmd`、`internal/cli/org_feature_test.go`、`internal/org/spawn_feature_test.go`、`internal/org/prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgProcedure`)。(d) N5: `checkFeatureWorktreeReuse` と `checkEnsuredFeatureWorktree` の `root, orgID, slug string`。ファイル名と関数名で書き、`file:line` は使っていない。(c) の旧い参照の綴りは、plan の参照の検査(`docs/plans/(active\|archive)/` の形を読む)に掛からないよう、`docs/` を付けずに書いた |
| `docs/tech-debt/README.md`(新しい行 3。test report の Test gaps) | 既存の行に入っていなかった 1 件だけを足した: symlink を通した `--plan` は fake の `FeatureWorktrees` でだけ確かめていて、本物のスクリプトでは確かめていない(`TestStartFeature_CanonicalRefIsTheResolvedPlanPath` が fake)。大文字小文字の件は新しい行 1 を指す。並行の start(同時の start の行)、`scriptFeatureWorktrees` のエラーの枝(org-feature-worktree の Test gaps の行の (f)、T-6)、実機の leader((a)、(g))は既存の行にあるので、行は足さず、行の中で指した |
| `.claude/skills/org/SKILL.md` と 3 つの写し(V-4、N3) | 使い回しの拒否の枝の見出しを「org_id が slug のとき(`--org-id` を渡さない既定)」から「org_id が slug と同じとき(`--org-id` を渡さない既定。`--org-id` に slug と同じ値を明示した場合も、コードは値が同じかどうかだけを見るのでここに入る)」にし、ensure のあとの枝(「org_id が slug と同じなら」)とコードの条件 `orgID == slug` に合わせた。明示した org_id は slug を変えてもそのままなので、その場合は打ち直したときの拒否が「`--org-id` で slug と違う org_id を渡したとき」の枝の案内になる、と 1 文足した |
| 同上(V-2) | 「`--plan` をどう書いても同じ計画なら同じになり」を「`--plan` を相対パスや symlink 経由で書いても同じ計画なら同じになり」に直し、「解決は大文字小文字を直さないので、大文字小文字を区別しない FS で `--plan` のパスの大文字小文字を前回と変えて打つと別の値になり、別の計画の記録として拒否される(使い回さない側に倒れる。前回と同じ綴りなら同じ値になる)」を足した。編集は `.claude/skills/org/SKILL.md` に 1 回だけ行い、`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り、2 つを `templates/base/` の 2 つに `cp` した |
| `docs/plans/active/2026-10-10-org-feature-worktree-ownership.md`(`## Progress checklist` だけ) | 「Verification artifact created」と「Test artifact created」を `[x]` にした。「Review artifact created」も `[x]` にした(依頼になかったが、self-review の report は `7bd95ebd` からある)。verify の V-3(Risks の 2 つ目が AC3b より前の書き方)と、この sync-docs の 2 行を足した。計画の本文は変えていない。digest は `feadcc05c21f` のまま |
| `docs/insights/events/2026-10-10-org-feature-worktree-ownership.jsonl` | `sync_docs` の event を 1 行足した(`--cycle 1`、verdict pass) |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`、`AGENTS.md` の repo map | `canonical_ref` にも start の拒否にも触れていない。変更なし |
| `docs/quality/quality-gates.md` と `templates/base/` の写しの「Split plan approval」の行 | 「`ralph-worktree.sh` の record と checkout が合うときだけ使い回す」。今も正しい。`ensure` のあとにもう一度読む、までは書いていないが、行の粒度では足りている。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-4 の 48 行) | 使い回しの条件を `canonical_ref`・パス・ブランチ・kind・チェックアウトで述べ、値の形(`split:<...>`)は書いていない。変更なし |
| `internal/cli` の `ralph org start --help`、`internal/org/prompts/*.md` | `canonical_ref` を含まない(`grep -rln canonical_ref` の結果にテスト以外では `internal/org/feature.go` と `scripts/ralph-worktree.sh`、仕様、skill、tech-debt、計画、報告だけが出る)。変更なし |
| `/org` skill の stop と disband の表のセル(tech-debt の F-8 の行が長さを数えている) | 触っていない。差分は `start --plan` の節の 2 か所だけ |
| `/org` skill の 4 面 | `cmp` で同一 |
| 計画の Risks の 2 つ目(verify V-3) | 古い書き方だが、計画の本文は承認の digest に入るので変えず、Progress に 1 行で記録した |

## Found but left

- Go の doc コメント(`featureCanonicalRef` の「`--plan` の書き方によらず同じ」)は直していない。Go のファイルの編集になり、pipeline が `/self-review` からの回し直しになるため(今は cycle 1 で、上限 2 に 1 回の余裕はあるが、回すかどうかは lead の判断)。直す条件と直し方は、tech-debt の新しい行 1 のトリガーに書いた
- N2〜N5(テストの doc、`--org-id` に slug と同じ値の枝、`active/` の参照 6 か所、`string` の並び)は、self-review の判断のとおり直さずに tech-debt の行 2 にした
- 計画の AC3 の本文(「slug(か Type)」「別の `--org-id` を案内しない」)は、承認の digest に入るので変えていない。ずれは計画の Progress の 2026-10-10 の 2 行(S1、self-review の直し)に記録済み
- `docs/quality/quality-gates.md` の行に「`ensure` のあとにもう一度読む」を足すことは、しなかった(上の表の理由)
- 計画の `- Approved:` の行と本文は触っていない

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh` | `done: 13 skill(s) mirrored to .agents/skills` |
| `cmp .claude/skills/org/SKILL.md` と、`.agents/`、`templates/base/.claude/`、`templates/base/.agents/` の 3 つ | すべて同一 |
| `./scripts/check-skill-sync.sh` | `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。編集のあとに実行 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-10-org-feature-worktree-ownership.md` | `feadcc05c21f`(編集の前後で同じ。`- Approved:` の行と一致) |
| tech-debt の表の形(scratchpad の `td_anchors.py` と `tdcheck.py`) | 編集の前に、計画した 4 つの anchor がすべて 1 回だけ出ることを確かめた。変わった行は 192 と、足した 195・196・197 だけ(`HEAD` と比べて)。4 行とも 7 つの区切り(5 列)、列ごとのバッククォートと `~~` は偶数、`file:line` の参照なし |
| tech-debt の plan 参照の検査(`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ正規表現を手で) | `docs/plans/(active\|archive)/` の参照はすべて存在する。最初の版は行 2 の (c) に `docs/plans/active/2026-10-09-...` を書いたので不在の参照になりかけ、`docs/` を外して直した |
| 新しい skill の行の表示幅(全角を 2 と数える) | 最大 77(既存の行は 80 まで) |
| `grep -rn 'plans/active/2026-10-09-org-feature-worktree' internal` | 6 件(N4 の 6 か所と一致)。`archive/2026-10-09-...` に直してあるのは `feature.go` の先頭と `feature_test.go` の先頭の 2 件 |
| `./scripts/run-verify.sh` | 下の「run-verify の結果」 |

## Not verified

- skill に足した大文字小文字の文は、`ResolveSplitPlanPath`・`featureWorktreeMismatch` のコードと self-review の APFS の probe からの推論で、大文字小文字を区別しない FS で start を打って確かめてはいない
- 「明示した org_id は slug を変えてもそのままなので、打ち直したときの拒否が別の枝の案内になる」は、`otherRecordFix` を読んでの推論と、self-review の N3(「2 回で収束する」)による。本物のスクリプトで打ってはいない
- この sync-docs の commit を含む range の secret scan は、commit の前には流せない。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、結果は lead への報告に書く

## run-verify の結果

`./scripts/run-verify.sh`(mode all、scope full)を、skill の 4 面、tech-debt、計画、insights の event、この report(この節を除く)の編集がすべて終わった状態で流した。開始は 2026-10-10T08:13:18Z(JST 17:13)、終了は 08:20:21Z で、rc 0、最後の行は「All verifiers passed.」。この節(結果の記録)は、実行のあとにこの report へ足した。流した時点の report には、この節だけがない。

- 静的: shellcheck、hook の `sh -n`(root と templates)、guard の awk の構文、`jq -e` の settings、Codex の hook のガード、`check-sync.sh`(PASS、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt README の plan の参照(OK)
- shell のテスト: `FAIL: <n>` の集計行 24 本はすべて `FAIL: 0`
- Go(scope は full): `gofmt: ok`、`0 issues.`(golangci-lint)、`go test ./...` は 8 パッケージすべて ok(`internal/org` は 24.3 秒で流し直し、残りはキャッシュ)
- 分岐の secret scan: `be050681..0ec79aae` は clean。この sync-docs の commit は、実行の時点でまだ range に入っていない
- evidence: `docs/evidence/verify-2026-10-10-081318.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)
