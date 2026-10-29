# sync-docs report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(承認の digest `0f38a9718a8b` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値は 1)。branch は `feat/org-merge-cleanup`、base は `15e301c8`。run 開始時の HEAD は `a36adbe2`(この report を含む commit の SHA ではない)。この run の直しの commit は `7e72d11e`
- 先行 report: `docs/reports/self-review-2026-10-28-org-merge-cleanup.md`(run 3 と F4 の追記まで、CRITICAL・HIGH・MEDIUM は 0)、`docs/reports/verify-2026-10-28-org-merge-cleanup.md`(pass、所見は LOW-1 の 1 つと Info)、`docs/reports/test-2026-10-28-org-merge-cleanup.md`(pass、`run-test.sh` rc 0、変異 26 個のうち 25 個が落ち 1 個は等価)
- 差分の大きさ(`git diff 15e301c8...HEAD --shortstat`、この report の前): 52 ファイル、+5960/-293。walkthrough を書くかどうかは `/pr` が決める

この下は cycle 1 の記録で、行番号もその時点のもの。パイプラインの cycle 2(cross-review cycle 1 の指摘の直し、HEAD 1341a3d4)の sync-docs は、末尾の「cycle 2(HEAD 1341a3d4)」の節にある。

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

## cycle 2(HEAD 1341a3d4)

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(承認の digest `0f38a9718a8b` はこの run の前後で同じ。変えたのは `## Progress checklist` の行だけ)
- Pipeline cycle: 2(`.harness/state/standard-pipeline/cycle-count.json` の値は 2)。run 開始時の HEAD は `1341a3d4`(この report を含む commit の SHA ではない)
- 対象の変更: cross-review cycle 1 の直し 3 コミット。8e46db0f(S-X1: cleanup の PR の読み方を origin の repository に固定し、origin が読めなければ断る)、d3452a9d(S-X2)と 54fa6351(C2-F): ディレクトリの消えた worktree の登録に確かめ 6 をかけ、登録が binding のブランチにいなければ断って、手で消すコマンドを出す
- 先行 report: `docs/reports/verify-2026-10-28-org-merge-cleanup.md`(cycle 2 は pass。この run に渡された所見は C2-D1〜C2-D5)、`docs/reports/test-2026-10-28-org-merge-cleanup.md`(cycle 2 は pass)

### Summary

verify の cycle 2 が渡した所見のうち、C2-D1・C2-D2(LOW)と C2-D3・C2-D4(Info)を直し、C2-D5(Info、計画の Progress)も 2 行足して解消した。

cycle 2 の動きは 2 つあり、どちらも文書の列挙に入っていなかった。cleanup の PR の読み方は origin の repository に固定され、origin が GitHub の repository を名指さないか読めなければ、台帳を読む前に何も消さず 1 で断る。ディレクトリの消えた worktree の登録は、detached HEAD・別のブランチ・消えたブランチ・読めない HEAD のどれかなら断り、手で消すコマンド(`git worktree remove <worktree> && <script の cleanup>`)を出す。この断りは ralph が登録を変えられないので、打ち直しても同じ断りになる。

help、`/org` skill、`/director` skill、README、仕様の注、tech-debt の行、AGENTS.md の地図と、文を固定するテストを直した。コードの動作は変えていない(Go の変更は help の文字列とテストの期待だけ)。

### Changes made

| File | Change |
|------|--------|
| `internal/cli/org_cleanup.go`(`ralph org cleanup --help`) | 手で消すコマンドが出る場面の列挙に「worktree のディレクトリがなく、git の登録に消すと失う HEAD がある」を足した(C2-D2)。確かめ 6 の文のあとに、ディレクトリのない worktree は登録で確かめ、登録はそのブランチにいてブランチが残っていること、を 1 文足した(C2-D3)。S-X1 が足した origin の段落は変えていない |
| `internal/cli/org_merge.go`(`ralph org merge --help`) | 同じ 2 か所(C2-D2、C2-D3) |
| `internal/cli/org_cleanup_test.go`、`internal/cli/org_merge_test.go` | `TestOrgCleanup_Help` と `TestOrgMerge_Help` の期待を、上の文のとおりに変えた(列挙の文を直し、確かめ 6 の 1 文を足した) |
| `.claude/skills/org/SKILL.md` | `cleanup` の行(234 行目)に、PR は `merge` と同じく origin の repository だけで読み `gh pr list` を `-R <host>/<owner>/<repo>` に固定すること、別の repository の PR は使わず `SKIPPED:` か `--branch` の断りの理由に出すこと、終了コード 1 に origin の断りがあること、確かめ 6 が登録にかかること、を足した(C2-D1、C2-D3)。`cleanup` と `merge` の行(233、234 行目)の手で消すコマンドの列挙に登録の断りを足した(C2-D2)。「org の終わり方」は、断りの一覧の detached HEAD の項に登録の場合を足し(C2-D3)、手で消すコマンドの列挙を「3 つ」から「4 つ」にして、登録の断りの文を足した(C2-D2) |
| `.claude/skills/director/SKILL.md` | 「GitHub で merge した PR の後始末」の手順 5 に、origin が GitHub の repository を名指さないか読めないときも、何も消さず 1 で終わることを足した(C2-D1)。ほかの箇所は変えていない |
| `.agents/skills/org/SKILL.md`、`.agents/skills/director/SKILL.md` | `./scripts/sync-skills.sh` で作り直した |
| `templates/base/.claude/skills/{org,director}/SKILL.md`、`templates/base/.agents/skills/{org,director}/SKILL.md` | `sync-skills.sh` は `.agents/skills` だけを作り直すので、4 つとも root から `cp` で写した |
| `internal/org/org_skill_send_sync_test.go`、`internal/org/director_skill_sync_test.go` | `/org` skill の 3 つの固定(cleanup の行、merge の行、「org の終わり方」)の列挙の断片を新しい文にし、origin への固定、別の repository の PR、origin の断り、確かめ 6 の登録、登録の断りの 3 文の断片を足した。`/director` skill の手順 5 の断片を 1 つ足し、固定の説明の注釈を直した |
| `README.md`(278 行目) | cleanup の段落に、PR を origin の repository だけで読むこと(C2-D1)、手で消すコマンドの列挙の登録の断り(C2-D2)、確かめ 6 の登録(C2-D3)、終了コード 1 の origin の断りを足した |
| `docs/specs/2026-10-07-org-multi-org-director.md`(133 行目の FR-8 の注) | 同じ 4 点を足した。登録の断りには、HEAD を残す方法を理由に出すことと、打ち直しても同じ断りになることも書いた |
| `docs/tech-debt/README.md`(ssh の別名の行、247 行目) | Trigger の列に、cleanup も同じ限界を持つことを足した。ssh の別名の origin では gh が失敗し、全ブランチが `SKIPPED:` になり、`--branch` は断られて、人が手で片付ける。コードを読んだだけで測っていないと書いた。Related の列に `internal/org/mergeorigin.go` と計画の S-X1 を足した(計画は `docs/plans/active/` の形で、`/pr` の `archive-plan.sh` が書き換える) |
| `AGENTS.md`(90 行目) | `internal/org/` の項の cleanup に、PR を origin の repository だけで読むこと(`mergeorigin.go` の `ReadMergeOrigin`)を足した(C2-D4)。地図が `cleanup.go`・`mergeenforce.go` などの隣のファイルを挙げる形なので足した |
| `docs/plans/active/2026-10-28-org-merge-cleanup.md` | `## Progress checklist` の「パイプライン cycle 2」に、test cycle 2 と sync-docs cycle 2 の 2 行を足した(C2-D5)。digest の範囲の外 |
| `docs/insights/events/2026-10-29-org-merge-cleanup.jsonl` | `sync_docs` の cycle 2 の event を 1 行追記した |
| `docs/reports/sync-docs-2026-10-28-org-merge-cleanup.md` | この節 |

### Surfaces checked for drift(cycle 2)

| Surface | 結果 | 根拠 |
|---------|------|------|
| `ralph org cleanup --help` の origin の段落と終了コード(S-X1) | 変更なし(正確) | origin への固定、別の repository の PR、origin が名指さないか読めないときの断り、終了コード 1 の列挙が、コード(`ReadMergeOrigin` の分類と `foreignPRWhy`)の言うことと合う |
| `internal/org/prompts/leader.md` の手順 15 | 変更なし(正確) | leader が worktree を消さないことと、`ralph org merge` か `ralph org cleanup` が消すこと、未コミットの変更が後始末を止めることだけを書く。origin の読み方も登録の断りも書かない |
| `/director` skill の `CLEANUP:` の読み方(861〜874 行目)と手順 3、1025 行目の「手で消すコマンドを打たない」 | 変更なし(正確) | 手で消すコマンドの場面を列挙せず、「ralph が先に進めず人の判断が要るときだけ出る」と書く。登録の断りもこれに入る。`CLEANUP MANUAL:` のない断りだけを「打ち直せば続く」と書くので、打ち直しても変わらない登録の断りとは食い違わない |
| `ralph org --help`(`internal/cli/org.go`)、`docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`、`docs/recipes/` | 変更なし | origin の読み方も手で消すコマンドの場面も書かない(`git grep` で 0 件) |
| `templates/base/AGENTS.md`、`templates/base/.ralph/core/AGENTS.core.md` | 変更なし | `cleanup.go` の項は root の AGENTS.md の地図にだけある。`check-sync.sh` は DRIFTED 0 |
| tech-debt の #263 の 6 行(383〜388 行目) | 変更なし | verify が self-review の扱いと合うと確かめた。この run は ssh の別名の行だけを足した |

### Decisions on the verify items

| 所見 | 扱い |
|------|------|
| C2-D1(LOW)cleanup の PR の読み方に origin への固定と断りがない | 直した。`/org` skill の cleanup の行と 3 つの写し、README、仕様の注、`/director` skill の手順 5 と 3 つの写し、tech-debt の ssh の別名の行 |
| C2-D2(LOW)手で消すコマンドの場面に登録の断りがない | 直した。2 つの help、`/org` skill の merge と cleanup の行と「org の終わり方」(数の「3 つ」)と写し、README、仕様の注。固定するテストも新しい文に変えた。`/director` skill は場面を列挙しないので変えていない |
| C2-D3(Info)確かめ 6 の言い方 | 直した。2 つの help の 1 文、`/org` skill の cleanup の行と「org の終わり方」の detached HEAD の項、README、仕様の注 |
| C2-D4(Info)AGENTS.md の地図に `mergeorigin.go` がない | 足した |
| C2-D5(Info)計画の Progress | 足した。self-review cycle 2 run 2 の結果と台帳の行、verify cycle 2 の行はすでにあったので、test cycle 2 と sync-docs cycle 2 の 2 行を足した |
| verify の関連: cycle 1 の cleanup は `-R` なしで gh に repository を選ばせたので、ssh の別名でも PR を読めたかもしれない | tech-debt の行には書かなかった。gh が別名をどう解くかを確かめていないため。この PR は cleanup の cycle 1 を release していないので、release ノートにも要らない |

### Verification(cycle 2)

| Check | Result |
|-------|--------|
| `gofmt -l internal/org internal/cli` | 出力なし(rc 0) |
| `go vet ./internal/org/ ./internal/cli/` | rc 0 |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgCleanup\|OrgMerge\|Help'` | `ok`、189.6 s |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|Prompt'` | `ok`、8.9 s |
| `./scripts/sync-skills.sh` | rc 0。14 skill を `.agents/skills` に写した。template の 4 つの写しは `cp` で写した |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。`PASS: all files in sync.` cycle 1 の記録(TEMPLATE_ONLY 11、KNOWN_DIFF 5)と数が違う。この run が足したファイルはないので、cycle 1 のあとに入った変更(origin/main の merge など)による数の動きとみられる。原因は調べていない |
| `git diff --check` | rc 0、出力なし |
| 変えたファイルの U+FFFD | 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-org-merge-cleanup.md` | `0f38a9718a8b`(`- Approved:` 行の `sha256:0f38a9718a8b` と同じ。Progress の編集のあとも同じ) |
| 範囲外 | `go test ./...` の全体、`run-verify.sh`、`run-static-verify.sh`、`run-test.sh` は打っていない。この run が変えた Go のファイルは help の文字列とテストの期待だけで、動作は変わらない。コードの検査は `/verify` と `/test` の cycle 2 の結果(pass)のまま |
| insight event | `./scripts/insights-append.sh --slug org-merge-cleanup --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を実行した(rc 0)。cycle 2、`2026-10-29-org-merge-cleanup.jsonl` への追記 |

### `/pr` と release に渡すこと

- release ノートの新しい 1 行は要らない。cleanup は同じ PR で足す新しい動詞で、origin への固定と登録の断りはその動きの一部である。cycle 1 の release ノート案(上の cycle 1 の節)の「新しい `ralph org cleanup [--branch <ブランチ>]` で消せる」に、origin の repository だけで PR を読むことを 1 文足すと、ssh の別名の origin で使えない理由が読める
- PR の本文の既知の穴に、ssh の別名の origin では cleanup が全ブランチを `SKIPPED:` にすることを足す(tech-debt の ssh の別名の行に記録した。コードを読んだだけで、実際の gh では確かめていない)
- 記録だけの修正の例外は使っていない
