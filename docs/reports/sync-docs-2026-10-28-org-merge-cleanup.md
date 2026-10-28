# sync-docs report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(承認の digest `0f38a9718a8b` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値は 1)。branch は `feat/org-merge-cleanup`、base は `15e301c8`。run 開始時の HEAD は `a36adbe2`(この report を含む commit の SHA ではない)。この run の直しの commit は `7e72d11e`
- 先行 report: `docs/reports/self-review-2026-10-28-org-merge-cleanup.md`(run 3 と F4 の追記まで、CRITICAL・HIGH・MEDIUM は 0)、`docs/reports/verify-2026-10-28-org-merge-cleanup.md`(pass、所見は LOW-1 の 1 つと Info)、`docs/reports/test-2026-10-28-org-merge-cleanup.md`(pass、`run-test.sh` rc 0、変異 26 個のうち 25 個が落ち 1 個は等価)
- 差分の大きさ(`git diff 15e301c8...HEAD --shortstat`、この report の前): 52 ファイル、+5960/-293。walkthrough を書くかどうかは `/pr` が決める

## Summary

verify の report の LOW-1 と Info 1 つを直し、新しい動詞 `ralph org cleanup` を並べるべき他の文書を探した。

LOW-1 は、既存の動詞の列挙 2 つに `cleanup` がなかったこと。台帳を書き換える動詞の一覧(古い台帳の拒否)と、見張りがなくても使える動詞の一覧である。`/org` skill の 4 つの写しと Go の注釈を直した。Info は `featureWorktreeCleanupPolicy` の注釈で、「merge のあとは人が消す」前提のまま残っていた。

探した結果、同じ種類の漏れが 2 か所ほかにあった。README の動詞の表(`gate/merge-order/merge/regate/director` の並びと説明文)と、quality-gates の watch gate の行(見張りなしで使える動詞の一覧。root と template に同じ行がある)である。どちらも直した。コードの動作は変えていない。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`(92〜93 行目) | 古い台帳の拒否の一覧に `cleanup` を `merge` の後ろに足した。2 行の折り返しは、`approve` と `inbox ack` の間で組み直した |
| `.claude/skills/org/SKILL.md`(1643〜1644 行目) | 見張りがなくても使える動詞の一覧に `cleanup` を `merge` の後ろに足した。折り返しは `read` と `status` の間で組み直した |
| `.agents/skills/org/SKILL.md` | `./scripts/sync-skills.sh` で作り直した。上の 2 か所と同じ差分 |
| `templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `sync-skills.sh` は `.agents/skills` だけを作り直すので、template の 2 つは root から `cp` で写した。写す前に、2 つとも HEAD の root と `cmp` で同一だったことを確かめた |
| `internal/cli/org_legacy_ledger.go`(22〜27 行目) | `orgLedgerMutating` の注釈の動詞の一覧に `cleanup` を足した。`ralph org cleanup` は `guardLegacyOrgStateDir(..., orgLedgerMutating)` を打つ(`internal/cli/org_cleanup.go:108`)。注釈だけの変更 |
| `internal/org/feature.go`(59〜62 行目) | `featureWorktreeCleanupPolicy` の注釈を、「disband のあとも残り、PR が merge されると `ralph org merge` か `ralph org cleanup` が消す(どちらも `ralph-worktree.sh cleanup` を通す)。merge されずに閉じた PR では人が script を打つ」に直した。`--cleanup-policy manual` の指定は変えていない。注釈だけの変更 |
| `README.md`(124 行目) | `ralph org` の動詞の並びに `cleanup` を足し、`merge` の説明の終わりに、merge が機能の worktree・ローカルのブランチ・`ralph-worktree.sh` の記録を消すことと、GitHub で merge した PR には `cleanup [--branch <branch>]` が同じことをすることを 1 文で足した。詳しい段落は 270 行目にすでにある |
| `docs/quality/quality-gates.md`(80 行目)、`templates/base/docs/quality/quality-gates.md`(79 行目) | watch gate の行の「見張りなしで使える動詞」に `cleanup` を足した。2 つの行は同じ文面で、変更後も同じ(`diff` で確かめた)。この 2 つのファイルは `check-sync.sh` の KNOWN_DIFF だが、この行は差の外にある |
| `docs/plans/active/2026-10-28-org-merge-cleanup.md` | `## Progress checklist` の 3 つの artifact の行を `[x]` にし(report が 3 つとも pass で存在する)、sync-docs の行を足した。S5 の記録の「1137 行」を「1143 行」に直した(F2・F3 の追記で閉じた PR の案内の行が動いた)。digest の範囲の外 |
| `docs/insights/events/2026-10-28-org-merge-cleanup.jsonl` | `sync_docs` の event を 1 行追記した(下の Verification) |
| `docs/reports/sync-docs-2026-10-28-org-merge-cleanup.md` | この report |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `ralph org --help` の説明と `--org-id` の説明(`internal/cli/org.go:30-59`) | 変更なし(正確) | merge の片付けと `cleanup` の説明があり、`--org-id` を断る動詞の一覧にも `cleanup` が入っている |
| 見張りの gate の注釈(`internal/cli/org.go:435-440` の `wireOrgWatchGate`、`internal/org/spawn.go:288-293` の `WatchGate`) | 変更なし(正確) | 動詞を並べず、「spawn、start、send だけが gate を呼ぶ。ほかの動詞は見張りなしで動く」と書く。`cleanup.go` と `org_cleanup.go` に `WatchGate`・`wireOrgWatchGate` はない(grep で 0 件) |
| `AGENTS.md`(90 行目) | 変更なし(正確) | `cleanup.go` の項がある。watch gate の節は gate を受ける動詞だけを挙げ、免除する動詞は並べない |
| `README.md`(252、266、268、270 行目) | 変更なし(正確) | verify が確かめた。この run は 124 行目だけを足した |
| `.claude/skills/director/SKILL.md` と 3 つの写し | 変更なし(正確) | `ralph org regate` と `merge` を並べる箇所は、director の手順の説明で、動詞の一覧ではない。`cleanup` は 859〜874 行目、959〜987 行目、1023 行目にある |
| `docs/specs/2026-10-07-org-multi-org-director.md`(132 行目の FR-8 の注) | 変更なし(正確) | verify が確かめた。ほかの注(88、123、130、131 行目)は日付つきの履歴で、動詞の一覧ではない |
| `docs/quality/quality-gates.md`(82 行目の Quality pipeline gate の行) | 変更なし(正確) | `regate` の記述は gate の話で、`cleanup` は gate ではない |
| `docs/architecture/repo-map.md`、`.claude/rules/ralph/agent-messaging.md`、`.claude/rules/ralph/git-commit-strategy.md` | 変更なし | 動詞の一覧も、機能の worktree の後始末の文もない |
| `docs/recipes/isolated-herdr-server.md`(524〜535 行目)と template の写し | 変更なし(一致) | この PR の中ですでに直してあり、`cmp` で同一 |
| 人が消す案内の残り(`/org` skill の 727、729、777、1143 行目) | 変更なし(正確) | start の失敗の案内と、merge されずに閉じた PR の案内で、いまも人が打つ場面 |
| 固定するテスト | 当たらない | 動詞の一覧の文字列を固定するテストは見つからなかった(`grep -F` で、この run が直した並びを探した)。`orgSkillOldCleanupWords`(`internal/org/org_skill_send_sync_test.go:215-221`)は古い後始末の文が戻らないことを見るテストで、この run の文には当たらない。テストは打っていない |
| `docs/tech-debt/README.md` | 行を足していない | LOW-1 は直したので負債にならない。`internal/org/feature.go` と `internal/cli/org_legacy_ledger.go` の行番号を引く行は、tech-debt・仕様・README・AGENTS.md・quality-gates のどれにもない |
| `origin/main` の動き | 衝突なし | `15e301c8` から `4389396f`(PR #280)まで 19 ファイル進み、`README.md` と `docs/tech-debt/README.md` はこの branch とどちらも変えている。`git merge-tree --write-tree HEAD origin/main` は衝突のない tree を返した。rebase か merge するかは `/pr` が決める |

## Decisions on the reports' remaining items

| 出どころ | 項目 | 扱い |
|----------|------|------|
| verify | 【LOW-1 (a)】古い台帳を拒否する動詞の一覧に `cleanup` がない | 直した(Changes made)。`/org` skill の 4 つの写しと Go の注釈 |
| verify | 【LOW-1 (b)】見張りなしで使える動詞の一覧に `cleanup` がない | 直した。`/org` skill の 4 つの写し。同じ漏れの README と quality-gates も直した |
| verify | 【Info】`feature.go:57-60` の注釈 | 直した |
| verify | 【Info】Progress の「1137 行」 | 直した(1143 行) |
| verify | 【Info】self-review の run 3 の insight event がない | 直し済みだった。`44f13b7f` が足していて、いまのファイルに run 3 の行(`low: 3`)がある |
| verify | 【Info】issue の「手で消すときの打ち方を出す」と D2 の違い | 文書は直さない。計画の D2 と仕様の注に理由があるので、PR の本文に 1 文書く(下) |
| verify | 【Info】F4 の退避の名前に残る狭い場面 | 直さない。verify も直さなくてよいとした |
| verify、test | merge の側のテストがリモートのブランチを消さない | 直さない。squash merge と削除のあとの片付けは cleanup の側のテストが見て、merge の側も同じ `cleanupRun.remove` を通る |
| test | Test gaps 1〜5、7(別プロセスの flock、`worktree_cleaned` の書き込み失敗、`git worktree remove` の失敗、`runCleanupSteps` の中断、確かめ 6 の守りの道、実機) | tech-debt に行を足していない。test の report が、確かめ 0〜6 と「exit 0 は記録した merge」の判定を変えない道と読んでいる。Test gaps 2・3 は失敗の道を打つテストがない点で、行にするかは lead が決める(行にするなら Trigger は `cleanup.go` の `record` か `remove` を次に直すとき) |

## Plan drift notes

- Scope と Affected areas は、S1〜S5 と F1〜F4 のとおりに実行した。この run が足した README の動詞の表と quality-gates の watch gate の行は、Affected areas の一覧になかった文書である。動詞を並べる文書の取りこぼしで、計画の範囲を広げる変更ではない。Progress に 1 行で記録した
- AC1〜AC6 の `[ ]` は digest の範囲の中なので触れていない
- 残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける

## `/pr` と release に渡すこと

- release ノートに手で足す 1 行(CHANGELOG は編集していない): 「`ralph org merge` が、merge を記録したあとで機能の worktree・ローカルのブランチ・`ralph-worktree.sh` の記録を消すようになった。未コミットの変更や push していないコミットなど消すと失うものがあるときは、何も消さず、理由と次の手を `CLEANUP:` の行で出し、終了コードは 0 のままである。GitHub で直接 merge したブランチは、新しい `ralph org cleanup [--branch <ブランチ>]` で消せる。」
- release ノートに足す補足。leader の役割雛形は go:embed で、spawn のたびに展開される。古い binary で立てた leader の手順 15 は「人が消す」のままだが、merge のあとの片付けは merge を打つ側(新しい binary)が行うので、結果は同じ。台帳には `worktree_cleaned` の行が増える。古い binary はその行を読まずに無視する。片付けの間だけ、同じ org_id の `spawn` と `start` が `regate.lock` の断りで止まる
- 自動の changelog(`.goreleaser.yml` は `^docs:`、`^test:`、`^chore:` を除く)に載るのは `feat:` が 4 件、`fix:` が 5 件(この report の commit を除く)。`fix:` の 5 件はこの PR の `feat:` の直しで、同じ release に入る。手で書く release ノートでは新しい動きの一部として扱い、既存の不具合の修正とは書かない
- PR の本文に書くこと: (1) issue #263 の受け入れ条件 3 は「手で消すときの打ち方を出す」と書くが、計画の D2 は、打ち直せば続く断りでは手で消すコマンドを出さず `ralph org cleanup --branch <ブランチ>` の打ち直しを出す形にした。`--force-branch` の script を渡すと確かめ 5・6 を飛ばして push していないコミットまで消すため。(2) 既知の穴 2 つ。`runWorktreeScript` は script を時間の上限なしで打つ。`StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれる(Regate と同じ窓)。tech-debt の L5・L6 などの行に記録がある。(3) 記録だけの修正の例外は使っていない
- 実機の director と leader が新しい手順 15 と skill の文のとおりに動くことは確かめていない(verify と test の Not verified と同じ)
- tech-debt の行の Related 列は `docs/plans/active/` を指す。`/pr` の `archive-plan.sh` が `archive/` に書き換える

## Verification

| Check | Result |
|-------|--------|
| `./scripts/sync-skills.sh` | rc 0。14 skill を `.agents/skills` に写した。template の 2 つの写しは `cp` で写した |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `cmp`(`/org` skill の root と template の 2 組)、quality-gates の watch gate の行(root 80 行目と template 79 行目)の `diff` | 3 つとも同一 |
| `gofmt -l internal/cli/org_legacy_ledger.go internal/org/feature.go` | 出力なし(rc 0) |
| `git diff --check` | rc 0、出力なし |
| 変えたファイルの U+FFFD | 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-org-merge-cleanup.md` | `0f38a9718a8b`(`- Approved:` 行の `sha256:0f38a9718a8b` と同じ。Progress の編集のあとも同じ) |
| `git merge-tree --write-tree HEAD origin/main` | rc 0、衝突なし |
| `/natural-japanese` の lint(この report、`uv run --quiet ~/.claude/skills/natural-japanese/scripts/lint.py <report> --json`、1 回) | 指摘は 1 件で、和欧文間の半角空白(15 行、warn)。直さず残した。直前の sync-docs の report と同じ書き方で、`run`・`digest`・`merge` などの英単語と日本語の間の空白は語の切れ目を示し、コード片(バッククォート)との間も同じに揃えている |
| テスト | 打っていない。別の session が local CI を流している間の依頼で、`go test`・`go build`・golangci-lint・staticcheck・`run-verify.sh`・`run-static-verify.sh`・`run-test.sh` は避けた。この run が変えた Go のファイルは注釈の行だけで、動作は変わらない。コードの検査は `/verify` と `/test` の結果(pass)のまま |
| insight event | `./scripts/insights-append.sh --slug org-merge-cleanup --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を実行した。既存のファイルへの追記 |
