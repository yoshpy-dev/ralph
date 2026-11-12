# sync-docs report: scaffold-version-notice

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-scaffold-version-notice.md`(承認の digest `6ac68081cc35`)。issue #313
- Branch: `feat/scaffold-version-notice`。base は origin/main `9ac72c50`(`6ac8fde4` で取り込み済み)、run 開始時の HEAD は `90cbd303`(test の report のコミット)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `71c7b31a`(Merge: yes、LOW 4 件 L1〜L4)、verify `a80cbbe9`(pass。LOW 3 件 V1〜V3、INFO 6 件 V4〜V9)、test `90cbd303`(pass。Test gaps G1〜G6)
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `6ac68081cc35`。コードとテスト、director と org の skill、README は変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit の前の数): `git diff origin/main...HEAD` は 23 ファイル、+1,248/-13(計画、report、insights を含む)。docs 以外は +838/-13、そのうちテストと `go.sum` を除くと +193/-12。walkthrough は書いていない

## Summary

verify の V1〜V3 に当たる 3 行に、この PR(#313)が Trigger を引いて払わなかった note を足した。掃引でもう 1 行(Trigger に `newOrgStartCmd` の `RunE` を挙げる行)が当たったので、同じ形で足した。新しい tech-debt の行は 4 つで、director の skill と prompt、`checkManifestVersion`、テストの穴 G1/G2、self-review の L1/L2 に当たる。README の追記 3 か所は実装と合っていた。ほかの文書に直す所は無かった。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 既存の 4 行の Trigger の末尾に 2026-11-12 の note と、Related に計画の path(`docs/plans/active/...`)を足した。新しい 4 行を末尾に足した(464〜467 行)。変わった行は 88、195、201、261 と、追加の 4 行だけ。下の「tech-debt」 |
| `docs/plans/active/2026-11-12-scaffold-version-notice.md` | `## Progress checklist` の artifact の 3 つの箱を `[x]` に、後段(self-review、verify、test の commit と結果)と sync-docs の 2 行を「Implementation started」の下に足した。「PR created」と AC の箱は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-12-scaffold-version-notice.md` | この report |
| `docs/insights/events/2026-11-12-scaffold-version-notice.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 |

commit は 1 つ(`docs: sync docs for scaffold-version-notice`)。

## tech-debt

行は先頭の語で見つけた。

### 依頼 1: Trigger を引いて払わなかった 3 行と、掃引で見つけた 1 行

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 「6 batchable LOW findings from org-debt-batch」(V1) | 項目 (4)。`internal/cli/status.go` の `runStatus` に `printScaffoldVersionNotice` の呼び出し 1 行と、`--help` の段落 | `status.go` は 966 → 971 行(origin/main `9ac72c50` と HEAD を `wc -l` で数えた)。`printStateDirLine` は 1 か所のまま、`state_dir_source` の `omitempty` も残る。計画の Scope は stderr の 1 行 |
| 「Test gaps left by org-feature-worktree」(V2) | (e) と (f)。`runOrgStartPlan` の呼び出し 1 行と doc の 4 行 | (e) の `org.ResolveRoleModel` の拒否と (f) の防御の枝は今も走らない。新しい呼び出しは `TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold` が走らせる |
| 「Two test gaps of the org-director-attach PR」(V3) | H4 と H5。`attach` の `RunE` に呼び出し 1 行と `--help` の段落 | store の `write` と `truncate` の seam は非公開のままで、CLI のテストは `ErrDirectorWriteNotRestored` を出させられない。計画の Scope は store に触れない |
| 「A `ralph` of another version on the pane's `PATH`」(掃引で見つけた。handoff、self-review、verify のどれも挙げていない) | Trigger の「Also the next change to the `RunE` of `newOrgStartCmd`, for the `start <task>` gap」。`start <task>` の形の呼び出し 1 行と `--help` の段落 | 知らせはその CLI が見る雛形の版の比べで、operator の stderr に出る。leader のタスクには入らないので、`start <task>` の leader に台帳の行が付かないことと、pane の `ralph` と org を起こした `ralph` の版のずれは前のまま。2026-10-30 と 2026-11-06 の note と同じ形 |

4 行とも Related の末尾に `docs/plans/active/2026-11-12-scaffold-version-notice.md`(Scope; the carry-over note)を足した。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`docs/plans/archive/` とは書いていない。

### 依頼 2: 新しい 4 行(464〜467 行)

| 行(先頭の語) | 内容 |
|----|----|
| 「The `/director` skill (`.claude/skills/director/SKILL.md`, in four copies) and the headless director's role prompt」 | (a)。director の skill(4 つの写し)と `internal/org/prompts/director.md` が、知らせを見たときに人に伝えるか自分で `ralph upgrade` を流すかを決めていない。root の 2 つの文書に `scaffold`、`manifest.toml`、`ralph upgrade` は 0 件(`git grep -c -i`)。Impact に STOIC の例(8.1.0 の CLI と 7.0.0 の skill)と、`ralph upgrade` が人の repository の追跡ファイルを書き換えるので、director が頼まれずに流すかは製品の判断であることを書いた。Trigger は skill か prompt の次の変更 |
| 「`ralph doctor`'s `checkManifestVersion`」 | (b)。文字列一致なので、雛形が CLI より新しいときと CLI が `dev` のときも `run 'ralph upgrade'` と言う。この 2 つで新しい知らせ(semver)は黙る。Impact の「逆向きの助言」は仕様の配布 2 段階の記述と `checkManifestVersion` の読みからの推論で、動かしてはいない(行にもそう書いた)。計画の Non-goals |
| 「Test gaps of feat/scaffold-version-notice (#313), the test report's G1 and G2」 | (c)。G1: `org start <task>` と `director start` のテストは git の外で流れ、root が `.` になるので、`.` を渡す変異 MA、MB が package 全体で生き残る。G2: status を linked worktree から流すテストが無く、変異 MC は `TestStatusCmd_MergeGuaranteeRunsNoGH` の git の呼び出し回数でだけ死ぬ。Trigger に test report が示した最小の足し方を写した |
| 「Two small findings of feat/scaffold-version-notice (#313), the self-review's L1 and L2」 | (d)。L1: `orgScaffoldRoot` が `org.FeatureRepoRoot` の最初の 2 段の写しで、doc comment だけで順を保つ。L2: help、README、2 つの doc comment が「main worktree」とだけ言い、コードは toplevel、`.` に落ちる。toplevel の段は走るテストが無い(test report G3)。L2 の直し方に、help の 76 桁の上限と `TestScaffoldVersionHelp_EachVerbDescribesTheNote` が固定する文を書いた |

行の中の名前はすべて関数とテストの名前で、新しい `file:line` は無い。参照した名前は `git grep` で実在を確かめた(`scaffoldVersionNotice`、`scaffoldVersionOlder`、`orgScaffoldVersionNoticeHelp`、`FeatureRepoRoot`、`ErrDirectorWriteNotRestored`、`newLegacyLedgerRepo`、対象の 4 つのテスト名ほか)。

### 掃引

Trigger の欄を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、全体が `~~` の行は飛ばす)、変えたファイルと関数の名前(`status.go`、`org.go`、`org_director.go`、`org_director_headless.go`、`upgrade.go`、`upgrade_v2.go`、`migrate.go`、`report.go`、`runStatus`、`runOrgStartPlan`、`newOrgStartCmd`、`startDirector`、`runUpgradeV2`、`RenderUpgradeReport`、`UpgradeReportData`、`checkManifestVersion`、`go.mod` ほか)で当たりを集め、当たった Trigger を 1 つずつ読んだ。Debt の欄に名前があって Trigger が「any of」の行も、`upgrade_v2`、`report.go`、`runUpgradeIO` などで探した。`org.go`、`status.go`、`migrate.go`、`upgrade.go`、`upgrade_v2.go`、`report.go`、README の hunk は `git diff origin/main...HEAD` で読み直した。

当たらない行と理由:

| 行(先頭の語) | 判断 |
|----|----|
| 「`runUpgradeIO` positional-parameter creep」 | `runUpgradeV2` に `fromVersion` が増えたが、`runUpgradeIO` は変わらず、`fromVersion` は表示や IO の knob でなく版の値(self-review も同じ読み) |
| 「5 unfixed cycle-1 LOW findings from overlay-scaffold-v2」 | Trigger の「listed call sites」(`WriteUpgradeReport`、`writeFileV2`、`writeMigrationFile`)は変わっていない。`report.go` で変えたのは `UpgradeReportData` と `RenderUpgradeReport` |
| 「`PlanCoreReplace` cannot handle template path shape transitions」、「`relocationOutcome`」の行、`.codex/AGENTS.override.md` の行 | Trigger は `replaceplan.go`、`relocationOutcome`、`buildMigratedManifest`、`classifyLegacyPath`。`migrate.go` の hunk は `runMigrateLegacy` の呼び出しと comment だけ |
| 「Two documentation findings of org-stop-all」 | Trigger の (a) は `/org` skill の表で、触っていない。(b) の `newOrgStartCmd` は `~~` で閉じている |
| 「Small findings of org-inbox-escalate」、「Small findings of org-gate cycles 3 and 4」、`runOrgStartPlan` を挙げる 2 行(「`ralph org start --plan` reads the main worktree's `ralph.toml` twice」ほか) | 当たる項目はどれも `~~` で閉じている。`go.mod` は `go` の行でなく `require` を 1 行足しただけ |
| 「The next line added to the `ralph status` header」(260 行)、「`internal/cli/status.go` grew from 901 to 963 lines」(417 行) | 知らせは stderr の 1 行で、status の header の行でも節でもない。417 行の数字(901 → 963)は触っていない(今は 971 行) |
| 「Two mutants of the org-director-start PR's D13 check」(275 行) | 判断が分かれる。`startDirector` に知らせの 1 行と doc の言い足しが入ったが、Trigger の「the order of refusals」は同じ。知らせは断りでなく、doc は順を変えずに知らせを足しただけ。verify も当たらないとした。当たると読むなら、note を足せる |
| 「A seat that a spawn left through `spawn_close_failed`」、「seat table of `ralph status`」の行、`ralph org status` の JSON を挙げる行、`resume_screens.go` の行 | status の seat table も JSON も `Waiting` の画面も変えていない。`status.go` の hunk は `runStatus` の scaffold の直後と Long の段落だけ |
| `ralph doctor`、`ralph upgrade` の語を拾った行(「Residual risks of `/cross-review`'s codex reviewer」、「A project scaffolded before `templates/base/.gitignore`」ほか)、`attach` や `README.md` の語を拾った行 | Trigger は別の変更を指す。README の hunk は `ralph org` の行、`ralph status` の行、`ralph upgrade` の節の 1 文で、`[org].state_dir` の段落(401 行の Trigger)ではない |

## 確かめたこと(文書が実装と合っていた所)

- README の 3 か所(`ralph org` の行、`ralph status` の行、`ralph upgrade` の節)。説明の中身は、5 つの呼び出し、semver の比べ、stdout と `--json` が変わらないこと、`Version: <from> → <to>`、移行のあとは移行前の版、版が空なら行を出さないこと、と一致する。「main worktree」の言い方は (d) の L2 と同じ。README は書き直していない
- 4 つの verb の `--help`(`orgScaffoldVersionNoticeHelp`、`statusScaffoldVersionNoticeHelp`)は verify が 8.1.0 の binary で確かめた
- `AGENTS.md` の repo map は `internal/cli/` を個別のファイルで挙げず、新しいファイルの行は要らない。`internal/upgrade/` の行は報告の path だけを書く
- `docs/architecture/`、`docs/quality/`、`docs/recipes/`、`templates/base/docs/`、`.claude/rules/`、skill では、変えたファイル、upgrade の報告の中身、4 つの verb の stderr を書いた所を `git grep` で探して 0 件
- `docs/specs/2026-08-01-org-runtime.md` の運用ノート (c)(配布は 2 段階)は、`ralph upgrade` が入っている binary の埋め込みテンプレートを使う点を書く。新しい知らせはそれと矛盾しないので、変えていない。`docs/specs/2026-08-17-overlay-scaffold-v2.md` は報告の path だけを書く
- 計画の D1〜D6 と AC1〜AC8 は verify が照合済み。この pass で本文は変えていない

## 残りの項目の扱い

| 項 | 扱い |
|----|----|
| V4、L2(help と README の言い方) | (d) の行にした。help は定数 1 か所とテストの固定があり、README だけ直すと食い違うので、文書の同期では直さない |
| V5(semver の短縮形 `8.1`) | 変えない。ralph は 3 つの数の版を書く |
| V6 | drift でない |
| V7 | Progress の 3 つの箱を `[x]` にした。AC の箱は `[ ]` のまま(lead か `/pr`) |
| V8、L3(テストの comment が `docs/plans/active/...` を引く) | 行にしていない。`/pr` が計画を archive に移すと path が古くなる comment 1 行。依頼に無いので lead に知らせる |
| V9 | (a) の行 |
| L4(体裁の 3 点) | 行にしていない。動作に影響しない。依頼に無い |
| G1、G2 | (c) の行 |
| G3 | (d) の行 |
| G4(package 全体の shuffle)、G5(本物の herdr)、G6(`-race`) | 行にしていない。test report が環境と diff の性質から理由を書いている |

## 残っていること(lead へ)

- コードとテストは変えていない。パイプラインの再走は要らない(記録だけの変更)
- 払っていないもの: 4 行の Trigger と、新しい 4 行の中身(どれもコードかテストの変更、または製品の判断)
- 計画の Non-goals に、merge のあとに「版ごとの挙動の注記」の別 issue を起票して番号を ralph-d1 に知らせる、とある。まだ済んでいない
- 次の release ノートに入れる 2 項目(計画の Rollout): 4 つの verb が stderr に 1 行出すこと、upgrade の報告に from → to が出ること
- 275 行(`startDirector` の断る順)は当たらないと読んだ。当たると読むなら、note は 1 文で足せる
- 同じファイルを別のセッションも編集している。この commit は 4 行への挿入と末尾の 4 行の追加だけなので、merge の衝突は末尾の追加行で起きうる

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 88、195、201、261(挿入だけ)と、末尾の新しい 4 行(行数は 464 から 468 で、末尾の空行が 1 つ)。変わった行も新しい行も列は 5、列ごとのバッククォートと `~~` は偶数。新しい `file:line` は 0(diff に出る 4 件は行 88 の既存の文)
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-scaffold-version-notice.md` は編集の前後とも `6ac68081cc35`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`git diff --check` は rc 0
- 実行していないもの: テスト、`run-verify.sh` の全体(コードとテストを変えていない)

## Insight event

```
./scripts/insights-append.sh --slug scaffold-version-notice --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-scaffold-version-notice.jsonl` が 3 行から 4 行になり、足した行は cycle 1、pass、ts は 2026-11-12T10:37:21Z。ファイル名の日付は UTC で、`date -u` も 2026-11-12 だった。この時点の `git status --short` は既存の jsonl、計画、tech-debt の `M` だけで、迷子のファイルは無かった。
