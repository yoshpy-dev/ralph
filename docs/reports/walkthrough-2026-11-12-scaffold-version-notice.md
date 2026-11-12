# Walkthrough: 雛形が CLI より古いと、日々の動詞が 1 行で知らせる(scaffold-version-notice)

- 計画: `docs/plans/archive/2026-11-12-scaffold-version-notice.md`(PR の作成時に archive へ移す)
- issue: #313。PR #316
- 差分: `git diff 9ac72c50...HEAD`(26 ファイル、約 +1,420 / −17。テストと `go.sum` を除く Go は約 +190、Go のテストは約 +650、残りは計画・報告・README・tech-debt)

## 1. 知らせの部品(読む価値が一番高い)

- `internal/cli/scaffold_version_notice.go`
  - `scaffoldVersionOlder`: 前後の空白を除き、`v` が無ければ足してから `golang.org/x/mod/semver` で比べる。どちらかが semver でなければ false(CLI の `dev`、空の `meta.version`)。
  - `scaffoldVersionNotice`: root の `.ralph/manifest.toml` を読み、読めなければ空。古いときだけ `note: <verb>: the scaffold in <dir> (.ralph/manifest.toml) is ralph <m>, older than this CLI (<c>), so its skills and rules may not match this CLI; run 'ralph upgrade' in <dir>` を返す。`<dir>` は絶対パス。
  - `orgScaffoldRoot`: org の動詞が読む directory。今いる repo の main worktree、なければ toplevel、なければ `.`。先頭の `org.MainWorktreeRoot` は、`git-main-worktree` の台帳から git を呼ばずに同じ root を読み戻す近道。
  - `printScaffoldVersionNotice`: stderr に 1 行。書き込みの失敗は無視する(`printLegacyModelPoolNotice` と同じ)。
  - help の 2 つの定数。
- テスト: `scaffold_version_notice_test.go`(比較の表、manifest の有無と版、root の決め方)。

## 2. 4 つの動詞へのつなぎ込み

- `internal/cli/org.go`: `ralph org start <task>` と `runOrgStartPlan`。どちらも `printLegacyModelPoolNotice` の直後で、watch と herdr の gate より前に出す。2 つの形は排他なので二重に出ない。
- `internal/cli/org_director.go`: attach。`newOrgDirectorRuntime` の直後、enabled の断りより前。
- `internal/cli/org_director_headless.go`: `startDirector`。`printLegacyModelPoolNotice` の直後。
- `internal/cli/status.go`: `buildScaffoldStatus(".")` の直後に root `.` で出す。stdout と `--json` は変わらない。
- テスト: `scaffold_version_notice_cli_test.go`(4 つの動詞、linked worktree から main の manifest を読むこと、`--state-dir` で台帳を外に置いたとき、status の stdout と `--json` が変わらないこと、黙る場合、help)。`status_scaffold_test.go` の 1 本は、直前のテストが残した package 変数 `Version` で新しい 1 行が出たので、テストの中で `Version` を manifest と同じ版に決めた。

## 3. upgrade の報告の版

- `internal/upgrade/report.go`: `UpgradeReportData.FromVersion`。空でなければ `Generated:` の次に `Version: <from> → <to>`。
- `internal/cli/upgrade_v2.go`: `runUpgradeV2(…, fromVersion, …)`。no-op の判定は今のまま。
- `internal/cli/upgrade.go` は読んだ manifest の版を、`internal/cli/migrate.go` は移行前の manifest の版を渡す。移行は manifest を今の版で作り直してから `runUpgradeV2` を呼ぶので、そこから取ると `<今> → <今>` になる。
- テスト: `report_test.go`(行の有無と位置)、`upgrade_v2_test.go`(`1.0.0-test → 1.1.0-test`、版が空なら行が無い)、`migrate_test.go`(`0.9.0-test → 2.0.0-test`)。

## 4. 文書

- README: org と status の行、`ralph upgrade` の節の報告の箇条。
- `docs/tech-debt/README.md`: この PR が触ったファイルの既存 4 行に note、末尾に新しい 4 行(director の扱い、doctor の比べ方、テストの穴 G1・G2、self-review の L1・L2)。

## 見どころ

- 知らせが消える向きの変化は無い。知らせは追加だけで、動詞の動きと終了コードは変えない。
- 読む雛形: org の動詞は台帳の場所によらず今いる repo の main worktree を読む。status は今いる directory を読む。古い main から切った task worktree で status を流すと、その worktree の雛形が古ければ知らせが出る。
- テストの穴: org start の task の形と director start は git の外でしか流していない(root に `.` を渡す変異が残る)。今のコードは正しい。
