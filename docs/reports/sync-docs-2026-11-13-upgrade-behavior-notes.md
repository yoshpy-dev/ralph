# sync-docs report: upgrade-behavior-notes

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-upgrade-behavior-notes.md`(承認の digest `0de679dfe2a1`)。issue #317
- Branch: `feat/upgrade-behavior-notes`。base は origin/main `4220ea7c`(`f55823a2` で取り込み済み)。run 開始時の HEAD は `d8d4dd6e`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `e91aa2b3`(追記 `e328b7d7`。Merge: yes、LOW 5 件は `ae1724a8` で直した)、verify `f9280fdc`(pass。LOW 2 件 V1・V2)、test `d8d4dd6e`(pass。Test gaps G1〜G5)。テストのコードの最後の変更は `0cce026b`
- 見た範囲: `git diff origin/main...HEAD`(17 ファイル)。Go のコードとテストの変更は verify の HEAD `0cce026b` 以降に無い。help の文と実装の対応は verify が読んだ結果を使い、build し直していない
- 制約: 計画の本文は変えていない。変えたのは Progress の節だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `0de679dfe2a1`(`git diff --numstat` は 7 行追加、3 行削除)。Go のコード、テスト、skill は変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff origin/main...HEAD` は 17 ファイル、+1,717/-20(計画、report、insights を含む)。docs 以外は 12 ファイルで +1,307/-20、そのうちテストを除くと 8 ファイルで +385/-18(skill の 2 写しと README を含む)。walkthrough は書いていない

## Summary

verify の V1 は AGENTS.md の `internal/upgrade/` の行に 1 句、V2 は overlay-scaffold-v2 の spec の FR-1 に 2026-11-13 の追記を足して直した。test の Test gaps は G1・G2・G5 を tech-debt の 1 行にし、計画の範囲外の気づきのうち移行の報告の上書きをもう 1 行にした。既存の行の Trigger に当たるものは無かった。`/release` step 4 の字下げは行にしていない。Progress の節に self-review、verify、test の結果と sync-docs の行を足し、3 つの artifact の箱を付けた。ほかの文書は実装と合っていた。

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md` | Repo map の `internal/upgrade/` の行の `upgrade report writer (...)` の括弧に、`## Behavior changes` の節と、その元の埋め込みの注記 `internal/upgrade/notes/<X.Y.Z>.md`(`/release` で書く)を足した。1 行の差分(他のセッションが編集するので、この行だけ) |
| `docs/specs/2026-08-17-overlay-scaffold-v2.md` | FR-1 の末尾に `[2026-11-13 追記(#317): ...]` を足した。同じ版・同じ日の報告は上書きせず `-2.md`、`-3.md` … に書くこと、`Version:` の行の次の `## Behavior changes` の節、注記の置き場。この spec の前例(AC-10 の `[2026-08-20 追記: ...]`)の形。1 行の差分 |
| `docs/tech-debt/README.md` | 末尾に 2 行を足した(485、486 行目)。挿入だけの編集 |
| `docs/plans/active/2026-11-13-upgrade-behavior-notes.md` | Progress の節だけ。`Implementation started` の子の箇条に self-review、verify、test、sync-docs の 4 行を足し、`Review artifact created`・`Verification artifact created`・`Test artifact created` を `[x]` にして report の path を付けた。`PR created` は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-13-upgrade-behavior-notes.md` | この report |
| `docs/insights/events/2026-11-12-upgrade-behavior-notes.jsonl` | `sync_docs` の行(cycle 1、pass)。ファイル名の日付は UTC。`git status --short` は既存のこのファイルが `M` で、新しい `??` は出なかった |

commit は 1 つ(`docs: sync docs for upgrade-behavior-notes`)。report が自分の commit の SHA を書けないので、内容と report は分けていない。

## verify が挙げた drift

### V1: AGENTS.md の `internal/upgrade/` の行に注記が無い

`upgrade report writer (docs/reports/upgrade-*.md)` の glob は `-2.md` も含むので名前は合っていた。足したのは、報告に `## Behavior changes` の節が出ることと、その元が埋め込みの `internal/upgrade/notes/<X.Y.Z>.md` で `/release` で書くこと。`templates/base/.ralph/core/AGENTS.core.md` と `.ralph/core/AGENTS.core.md` には `internal/upgrade` の語が無い(grep -c で 0)ので、managed block の元は変えていない。

### V2: spec の FR-1 (7) に `-2.md` の名前が無い

(7) の括弧は変えず、FR-1 の項の末尾に追記を足した。verify が書いたとおり、報告の中身(#313 の `Version:` の行を含む)はこの spec に無いので、節の説明は 1 文にとどめた。anchor は `python3` で `content.count(old) == 1` を確かめてから Edit した。

## tech-debt

### 既存の行の Trigger

`docs/tech-debt/README.md` の全行を、変更された file と関数の名前で python から引いた(`upgrade_v2`、`runUpgradeV2`、`WriteUpgradeReport`、`UpgradeReportRelPath`、`RenderUpgradeReport`、`internal/upgrade`、`scaffold_version_notice`、`semverWithV`、`printScaffoldVersionNotice`、`migrate_test`、`ralph-migration`、`/release`、`800 lines`、`time.Now().UTC` など)。当たった行の Trigger の列を読んだ。

| 行(先頭の語) | 判断 |
|----|----|
| 99「5 unfixed cycle-1 LOW findings from overlay-scaffold-v2」の (3)(`WriteUpgradeReport` の `0644`/`0755`) | 当たらない。`WriteUpgradeReport` の本体も呼び出しの行も変わっていない(変わったのは直前の名前を決める行と doc の文だけ)。`NextUpgradeReportRelPath` は権限を決めるところを足していない |
| 104、111、114、119(`PlanCoreReplaceDesired`、`buildDesiredStateV2`、`writeDriftGuidanceV2`) | 当たらない。どの関数も diff の hunk に無い。変わったのは `runUpgradeV2` の本体だけ |
| 465「`checkManifestVersion`」、466・467(#313 の `orgScaffoldRoot`、`printScaffoldVersionNotice` の引数) | 当たらない。Trigger の関数は変わっていない。`scaffold_version_notice.go` の変更は `scaffoldVersionOlder` の 1 行(`semverWithV` を `upgrade.SemverWithV` に替えた)と、写しの削除だけ。`semverWithV` の名前を書いた tech-debt の行は無い(`grep -c` で 0) |
| 88、195、201、261(`printScaffoldVersionNotice` の呼び出し) | 当たらない。呼び出しは変わっていない |
| 800 行の行(292、306、319、357、364、365、390、394、405、410、415、421 ほか) | 当たらない。どれも `internal/org`、`internal/cli/org_*`、`internal/config` の file の行で、この PR が触れた file は無い |

verify の判断(変更不要)と同じ結果になった。`internal/cli/upgrade_v2.go`(959 から 993 行)、`internal/cli/upgrade_v2_test.go`(2026 から 2309)、`internal/cli/migrate_test.go`(2906 から 2930)は前から 800 行を超えていて、行は無い。この PR が 800 を超えさせた file は無い(`git show origin/main:<f> | wc -l` と `wc -l` を、変えたすべての Go の file で比べた。`report_test.go` は 246 から 524、`behavior_notes_test.go` は新しく 335)。新しい行の Trigger に、G1 のテストを新しい file に置く指示を入れた。

### 新しい 2 行

- 485 行目: 「Test gaps of feat/upgrade-behavior-notes (#317), the test report's G1, G2, and G5」。(a) G1 `runUpgradeV2` の `WriteUpgradeReport` が失敗して返る行、(b) G2 `NextUpgradeReportRelPath` の `Lstat` が not-exist 以外で失敗する分岐、(c) G5 テストの日付が `time.Now().UTC()` を別に読むので UTC の 0 時をまたぐと落ちる(`upgrade_v2_test.go` は origin/main で 11 か所、いまは 15 か所。`grep -c` で数えた)。G5 の行はそれまで register に無かった(`time.Now().UTC`、`UTC date`、`midnight` などで引いた。当たったのは `run-verify.sh` の log 名の行 232 だけで、別の話)。G3(`ValidateBehaviorNotes` の `ReadDir` の失敗。実行時の側の同じ失敗は別のテストが見ている)と G4(計画どおり、次の release のあと)は INFO と計画どおりの扱いなので行にしていない。
- 486 行目: 移行の報告 `ralph-migration-<date>.md`(`migrationReportRelPath`)が同じ日の再移行で上書きされる。`runMigrateLegacy` が v3 の manifest を報告より先に書くので、再移行は manifest を旧版へ戻したときだけ起きる(self-review の観察)。LOW。

どちらも Related は `docs/plans/active/2026-11-13-upgrade-behavior-notes.md` と書いた。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`tdcheck` 相当の確かめ(`git show HEAD:` との比較)の結果は、元の file が新しい file の byte の prefix、追加は 2 行、どちらも 5 列(`|` で割ると 7 片)、列ごとの backtick は偶数、`file:line` の形の参照は 0、U+FFFD は 0。

### `/release` step 4 の字下げを行にしなかった理由

step 4 の子の箇条は 3 字下げで、番号つきの項目の中身の位置に合っている。計画が「既存の字下げのずれ」と呼んだのは、`4b.` が番号つきの項目ではなく段落になったことで、`5.` から新しい list が始まる点を指すと読んだ(self-review の観察と同じ)。CommonMark の描画で step の番号は読める。見た目だけで、動作にも読み手の判断にも影響せず、直す Trigger も「次に /release を編集するとき」でしかない。skill はこの run で触らない(依頼)。register の行は払うべき負債に取っておくので、行にしていない。直す人は `/release` の SKILL.md を次に編集する PR で見つけられる(`4b.` の行)。

## 実装と合っていると確かめたもの

| Doc / contract | 結果 |
|----|----|
| `README.md` の `ralph upgrade` の節(164〜167 行) | 合っている。範囲(`<from>` < version ≤ `<to>`、古い順)、節の出ない条件、stdout の `1 release` / `<n> releases`、`-2.md`・`-3.md`、`report:` の行が書いた名前を出すこと |
| `.claude/skills/release/SKILL.md` と `.agents/skills/release/SKILL.md` | `check-skill-sync.sh` が 14 skill で ok。この run では触っていない |
| `internal/upgrade/notes/README.md` | verify が確かめたとおり、規則とテストの説明がコードと一致する |
| `CLAUDE.md` の `/release` の行 | 手動起動だけで、変更なし |
| `.claude/skills/org/SKILL.md` ほか 4 つの写しの `docs/reports/upgrade-*.md` | glob が `-2.md` を含むので合っている |
| `docs/specs/2026-08-01-org-runtime.md` の「配布は 2 段階」 | 変わらない(binary を更新してから `ralph upgrade`) |
| `docs/architecture/repo-map.md`、`docs/quality/*.md` | `upgrade` の語が 0 件(grep)。変える所が無い |
| `templates/`、`.claude/rules/`、`.claude/hooks/`、`.codex/`、`scripts/` | この PR は触れていない(`git diff --name-only` で 0 件)。root と雛形のずれは生まれない |
| `--help`(`ralph upgrade`) | Long の文は報告の名前も節も書いていない(verify が読んだ) |

## 実行したもの

- `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && git diff --check`(commit の前、5 ファイルの変更を含めて): すべて exit 0。`check-sync.sh` は `PASS: all files in sync.`、`check-skill-sync.sh` は `14 skill(s) in lock-step`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-upgrade-behavior-notes.md`: 編集の前後とも `0de679dfe2a1`
- `./scripts/insights-append.sh --slug upgrade-behavior-notes --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`: exit 0。`--slug` は日付なしの slug
- テストと `run-static-verify.sh` は走らせていない(Go のコードを変えていない。verify と test の結果が有効)

## 残り

- AC1〜AC10 のチェックボックスは `[ ]` のまま(依頼が Progress の節だけだった。verify の INFO)。`/pr` の前に付ける
- AC1 の実物の注記での証拠は、最初の注記が入る次の release のあとに取る(計画の Rollout)
- self-review の報告は N1・N2 を未解決と書いたままで、計画の Progress に閉じた記録を足した。`0cce026b` 後のテストの doc comment は、限度が 100 のときの例の数(`-99`、`-100`)を括弧で書いている。動作には関係しない
- 計画の Test gaps G1・G2 は tech-debt の行にした。`/pr` の Known gaps に同じ名前で書ける
