# Walkthrough: upgrade の報告に release ごとの挙動の注記を出す(upgrade-behavior-notes)

- 計画: `docs/plans/archive/2026-11-13-upgrade-behavior-notes.md`(PR の作成のあとに archive へ移す)
- issue: #317。PR #325
- 差分: `git diff 4220ea7c...HEAD`(22 ファイル、約 +1,860 / −22。Go の本体は約 +340、Go のテストは約 +920、残りは skill・文書・計画・報告)

## 1. 注記の読み込みと選び方(読む価値が一番高い)

`internal/upgrade/behavior_notes.go`。

- `embeddedBehaviorNotes` は `//go:embed notes/*.md`。`BehaviorNotesFS` はその `notes/` を根にした `fs.FS` で、テストが差し替える。
- `behaviorNoteVersion`: ファイル名から版を取る。`<X.Y.Z>.md`(先頭に `v` なし、prerelease も可)で、`SemverWithV` を付けて semver として読めるものだけが注記。`README.md` はここで外れる。
- `BehaviorNotesBetween(from, to)`: from < v ≤ to の注記を古い順に返す。from か to が semver として読めない、または from ≥ to なら何も返さない。読めない注記は飛ばし、その理由を `problems` に返す(報告の Notes に 1 行ずつ入る)。
- `ValidateBehaviorNotes`: 名前が semver でない `.md`(README を除く)、空の本文、`#` で始まる行を含む本文を誤りにする。`behavior_notes_test.go` が埋め込みの本物の `notes/` をこれで確かめる。
- `SemverWithV` は `internal/cli/scaffold_version_notice.go` の `semverWithV` をここへ移したもの。挙動は同じ。

## 2. 報告の節と名前

`internal/upgrade/report.go`。

- `UpgradeReportData.BehaviorNotes` と `renderBehaviorNotesSection`: 注記があるときだけ、`Version:` の行の次に `## Behavior changes` と版ごとの `### <X.Y.Z>` を書く。無ければ何も書かないので、報告は前とバイト単位で同じ(テストが固定している)。
- `NextUpgradeReportRelPath`: `upgrade-<version>-<date>.md` が既にあれば `-2`、`-3` … と、`upgradeReportNameLimit`(100)個目まで空いた名前を探す。既にある名前が symlink やディレクトリでも「使用中」とみなす。Lstat が ENOENT 以外で失敗すれば誤りを返す。

## 3. upgrade の流れ

`internal/cli/upgrade_v2.go` の `runUpgradeV2`。

- `upgrade.BehaviorNotesBetween(fromVersion, Version)` を呼ぶ。`fromVersion` は #313 で足した移行前の版なので、旧レイアウトの移行のあとに続けて上げたときも、移行前の版から範囲を求める(`migrate_test.go` の期待を足した)。
- 報告の名前を `NextUpgradeReportRelPath` で決め、stdout の `report:` の行も実際に書いた名前を出す。
- `writeBehaviorNotesLine`: 注記があるときだけ、`report:` の行のあとに `  behavior changes: <n> release(s), see the report` を書く。`--dry-run` と何もしない実行では呼ばれない。

## 4. /release の段 4b

`.claude/skills/release/SKILL.md` と `.agents/skills/release/SKILL.md`(同じ文)。

- 版を決めた step 4 のあと、tag の step 5 の前。`git diff --stat <latest-tag>..HEAD -- templates/ internal/org/prompts/ ':(exclude)templates/base/docs/'` で、上げる側の挙動を変えうる変更を見せる。
- `git ls-tree --name-only HEAD internal/upgrade/notes/` で、計算した版の注記の有無と、ほかの未 release の版の注記が無いことを確かめる。
- 変更があって注記が無ければ、「注記を書く」(/release を終えて注記の PR を通し、やり直す)か「挙動の変わる点は無い」(step 9 の報告にその判断を書く)を選ばせる。完了の checklist にも 1 行足した。

## 5. 文書と記録

- README の `ralph upgrade` の節、AGENTS.md の `internal/upgrade/` の行、`docs/specs/2026-08-17-overlay-scaffold-v2.md` の FR-1 への 2026-11-13 の追記。
- `internal/upgrade/notes/README.md`: 注記の書き方(名前、本文の形、見出しを書かないこと)。
- `docs/tech-debt/README.md`: テストの穴(G1、G2、G5)と、移行の報告の上書きの行。

## 見どころ

- 範囲の境界(from は含まず to は含む、prerelease の to に正式版の注記を含めない)は `behavior_notes_test.go` の表で見る。
- 同じ日のやり直しは `upgrade_v2_test.go` の `TestRunUpgradeV2_SameDayRerun_KeepsEarlierReports` が 3 回走らせ、最初の報告の注記が残ることを確かめる。
