# Test report: upgrade-behavior-notes

- Date: 2026-11-13(run-test.sh は 02:41:27 から 03:03:10 JST)
- Plan: `docs/plans/active/2026-11-13-upgrade-behavior-notes.md`(`## Test plan`、AC1〜AC10、issue #317)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1。`.harness/state/standard-pipeline/cycle-count.json` が cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `f9280fdc`、origin/main `4220ea7c` は `f55823a2` で取り込み済み)。新しいテストは 26 本(`internal/upgrade/behavior_notes_test.go` に 12 本、`internal/upgrade/report_test.go` に 9 本、`internal/cli/upgrade_v2_test.go` に 5 本)。既存の `TestRunMigrateLegacy_HappyPath_Yes` に AC5 の確かめを足した。消えたテストと、既存の確かめを弱めた変更は無い(テストファイルの削除行は `migrate_test.go` のコメント 2 行だけ)。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。依頼に従い、変異のテストは流していない。worktree の production code とテストには手を入れていない。隙間を確かめる 3 本は `go test -overlay` で worktree の外のファイルを当てて流し、前後で `git status --porcelain` が空のままであることを確かめた
- Evidence: `docs/evidence/test-2026-11-13-upgrade-behavior-notes.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。runner と meta、run-test.sh の全出力、名前を絞った 2 回の `-v` の出力、coverage、隙間を確かめた 3 本のテストの本文と結果を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-12-174128.log`(名前の日付と時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,303 秒 |
| └ Go: `internal/cli` | 上位 1,195 本 | ok | 0 | — | 551.5 秒 |
| └ Go: `internal/upgrade` | 上位 137 本 | ok | 0 | — | 2.5 秒 |
| └ Go: `internal/org` | — | ok | 0 | — | 722.6 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights` | — | ok | 0 | — | 12.4、5.5、1.7 秒 |
| └ Go: `internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`cmd/ralph` | — | ok | 0 | — | 3.7、2.7、2.0、0.8 秒 |
| `go test ./internal/upgrade/ -run 'BehaviorNotes\|NextUpgradeReportRelPath\|RenderUpgradeReport' -v -count=1` | 上位 26 本、subtest を含めて 61 | 61 | 0 | 0 | 0.43 秒(wall 0.70 秒) |
| `go test ./internal/cli/ -run 'BehaviorNotes\|SameDayRerun\|MigrateLegacy_HappyPath\|ScaffoldVersion' -v -count=1` | 上位 23 本、subtest を含めて 55 | 55 | 0 | 0 | 4.2 秒(wall 4.7 秒) |
| coverage: `internal/upgrade` 全体と、`internal/cli` の `-run 'Upgrade\|Migrat\|ScaffoldVersion'`(`-coverprofile`) | — | ok | 0 | — | 1.4 秒、11.1 秒 |
| 隙間の確かめ 3 本(`-overlay` で足した throwaway のテスト、下の Test gaps) | 3 | 3 | 0 | 0 | 各 1 秒前後 |

run-test.sh の Go の段では、10 パッケージとも `(cached)` にならずに走った。1 分の load は開始時 8.62、走っているあいだ 5.86〜12.20、終了時 10.87 で、ほかのセッションも同じマシンを使っていた。名前を絞った 2 回、coverage の 2 回、隙間の確かめは shell の段のあいだに流し(どれも 15 秒以内)、Go の段には何も重ねていない。shell の段は 02:50 にはまだ走っていて、02:53 には Go の段に入っていた。空き容量は開始時 87 GiB、終了後も 87 GiB だった。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start、end、rc を meta に書く)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm 175` で区切った前景の待ちで終わりを待った。

Skipped の 2 行は前の PR の /test と同じで、`tests/test-plan-visual.sh` の browser detection(`/Applications` に browser がある手元の環境による)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらもこの diff と関係が無い。

## 受け入れ条件とテスト

どのテストも run-test.sh と、名前を絞った `-v` の回で PASS した。

| AC | テスト | 確かめている内容 |
| --- | --- | --- |
| AC1 8.1.0 → 8.3.0 で `### 8.2.0`、`### 8.3.0` の順。8.2.0 → 8.3.0 なら 8.3.0 だけ | `TestBehaviorNotesBetween_SelectsTheReleasesAfterFromUpToTo`(18 case。`two releases`、`only the upper end` を含む)、`TestBehaviorNotesBetween_OrdersByVersionNotByName`、`TestBehaviorNotesBetween_PrereleaseComesBeforeItsRelease`、`TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine`、通しの `TestRunUpgradeV2_BehaviorNotes_InRangeNotesInReportAndStdout` | 版の古い順(名前の順ではない)、下端を含まず上端を含む範囲、`v` の有無、前後の空白、build metadata、prerelease の to に正式版の注記を含めないこと |
| AC2 版が読めない、from ≥ to、範囲に注記なしのときは節なし | 上の表の `same version`、`downgrade`、`from is dev`、`from is empty`、`to is dev`、`to is empty`、`no note in the range`、通しの `TestRunUpgradeV2_BehaviorNotes_NoneInRange_NoSectionNoLine` | 注記も問題の行も返らないこと |
| AC3 節は `Version:` の次、`## Summary` の前。注記なしならバイト単位で同じ | `TestRenderUpgradeReport_NoBehaviorNotes_ByteIdentical`(nil と空の slice を変更前の形の literal と比べる)、`TestRenderUpgradeReport_BehaviorNotes_OnlyAddTheSection`、`TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine`、`TestRenderUpgradeReport_BehaviorNotes_EmptyFromVersionStillRendered` | 既存の `TestRenderUpgradeReport_FullData_SectionsInOrder` などは変更されずに通った |
| AC4 stdout の `behavior changes` の 1 行 | 通しの InRange(`2 releases`)、`TestRunUpgradeV2_BehaviorNotes_UnusableNoteSkippedWithANote`(`1 release`)、NoneInRange(出ない)、`TestRunUpgradeV2_BehaviorNotes_DryRunAndNoOpPrintNone`(dry run と no-op で出ない) | `report:` の行の直後に並ぶこと |
| AC5 旧レイアウトの移行のあとも移行前の版から範囲を求める | `TestRunMigrateLegacy_HappyPath_Yes`(0.9.0-test → 2.0.0-test で 1.5.0 だけ、0.8.0 と 2.0.0 は入らない、stdout に `1 release`) | 作り直した manifest の 2.0.0-test から求めると範囲が空になり、このテストは落ちる(fixture の値から読んだ。変異は流していない) |
| AC6 注記の形のテスト | `TestValidateBehaviorNotes_EmbeddedNotesPass`(本物の埋め込み)、`TestValidateBehaviorNotes_RefusesBadNotes`、`TestValidateBehaviorNotes_AcceptsWellFormedNotes`、`TestValidateBehaviorNotes_ReportsEveryBadNote`、実行時の側の `TestBehaviorNotesBetween_SkipsBadNotesInTheRangeWithAProblemLine`、`..._UnreadableNoteIsAProblemLine`、`..._UnreadableDirectoryIsAProblemLine`、`..._SkipsFilesThatAreNotNotesSilently` | semver でない名前、`v` 付き、短い版、build metadata、空、空白だけ、見出し 3 種、ディレクトリ |
| AC7、AC8、AC10 /release の段 4b と README | 振る舞いのテストの対象外(skill と README の文書) | verify の報告が file:line と pathspec の dry run で確かめた。shell の段の `test-check-skill-sync.sh`、`test-sync-skills.sh` は通った |
| AC9 同じ日の 2 回目は `-2.md`、最初の報告は残る | `TestRunUpgradeV2_SameDayRerun_KeepsEarlierReports`(3 回の実行、`-2.md` と `-3.md`、drift のエラーと stdout の `report:` が書いた名前を出す、前の報告がバイト単位で変わらない)、`TestNextUpgradeReportRelPath_TakesTheFirstFreeName`、`..._AnyEntryCountsAsTaken`(ファイル、ディレクトリ、symlink、dangling symlink)、`..._GivesUpAfterTheLimit`(上限の両側)、`..._SymlinkedReportsDir_Refused`、`..._SanitizesLikeUpgradeReportRelPath` | 既存の `TestRunUpgradeV2_SymlinkedReportsDir_RejectedNoEscape` も通り、外への書き込みは起きない |

## Coverage

- Statement: `internal/upgrade` 91.7%(パッケージ全体)。変更した関数は `BehaviorNotesBetween`、`behaviorNoteVersion`、`readBehaviorNote`、`SemverWithV`、`renderBehaviorNotesSection`、`RenderUpgradeReport`、`UpgradeReportRelPath`、`upgradeReportRelPathN` が 100%、`ValidateBehaviorNotes` 92.9%、`NextUpgradeReportRelPath` 90.9%、`embeddedBehaviorNotesDir` 75.0%。`internal/cli` は `-run 'Upgrade|Migrat|ScaffoldVersion'` に絞った回で `writeBehaviorNotesLine` 100%、`scaffoldVersionOlder` 100%、`runUpgradeV2` 85.7%
- Branch: 計測していない(Go の標準の道具は statement だけ)
- Function: 上の関数ごとの値
- Notes: 変更した行で通っていない block は 4 つ。`behavior_notes.go:36`(`fs.Sub` の panic、到達しない)、`behavior_notes.go:118`(`ValidateBehaviorNotes` の `ReadDir` の失敗)、`report.go:304`(`NextUpgradeReportRelPath` の Lstat の ENOENT 以外の失敗)、`upgrade_v2.go:223`(`WriteUpgradeReport` の失敗で返る行)。最後の 2 つは Test gaps の G1、G2。`runUpgradeV2` のほかの未到達(`:180`、`:227`、`:243`)は変更前からの行である

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | 失敗なし | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| symlink の `docs/reports` を通して外に報告を書かない(`TestRunUpgradeV2_SymlinkedReportsDir_RejectedNoEscape`) | 通る | 今は `NextUpgradeReportRelPath` の `ValidateRealParentChain` が先に断る。throwaway のテストで確かめたエラーは `upgrade report path "...": refusing to write through non-directory path component ...` で、外のディレクトリは空のまま |
| 注記の無い報告の形(#313 の `Version:` の行を含む) | 通る | `TestRenderUpgradeReport_NoBehaviorNotes_ByteIdentical` の literal、既存の `TestRenderUpgradeReport_FullData_SectionsInOrder`、`TestRenderUpgradeReport_FromVersion_VersionLineAfterGenerated`、`TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine` |
| 古い雛形の知らせ(#313、`semverWithV` を `upgrade.SemverWithV` に移した) | 通る | `ScaffoldVersion` の上位 17 本(名前を絞った cli の回) |
| `internal/upgrade` と `internal/cli` の全体 | 通る | run-test.sh の Go の段(`(cached)` なし) |

## Test gaps

- G1(LOW): `runUpgradeV2` が `WriteUpgradeReport` の失敗で返る行(`internal/cli/upgrade_v2.go:222-224`)を、どのテストも通らなくなった。変更前はこの行を `TestRunUpgradeV2_SymlinkedReportsDir_RejectedNoEscape` が通していたが、今は前の `NextUpgradeReportRelPath` が同じ検査で先に止める。`runUpgradeIO` を呼ぶ名前の範囲外の 3 本を足した coverage の回でも 0 だった。throwaway のテスト(`-overlay`)で、`docs/reports` を 0555 にすると `writing upgrade report: writing upgrade report "...": open ...: permission denied` で失敗し、stdout に `Upgrade complete` が出ないことを確かめた。足すなら `upgrade_v2_test.go` に読み取り専用の `docs/reports` の 1 本
- G2(LOW): `NextUpgradeReportRelPath` の Lstat が ENOENT 以外で失敗する分岐(`internal/upgrade/report.go:303-305`)にテストが無い。throwaway のテストで、`docs/reports` を 0600(検索の権限なし)にすると、名前を返さずに `lstat: ... permission denied` のエラーを返すことを確かめた
- G3(INFO): `ValidateBehaviorNotes` の `ReadDir` の失敗(`behavior_notes.go:117-119`)は通っていない。実行時の側の同じ失敗は `TestBehaviorNotesBetween_UnreadableDirectoryIsAProblemLine` が見ている。`embeddedBehaviorNotesDir` の panic は到達しない
- G4(計画どおり): AC1 の実物の証拠(本物の注記がある版をまたいだ upgrade の報告)は、最初の注記が入る次の release のあとに取る。/release の段 4b の対話の分かれ目(`AskUserQuestion` の 2 択、注記の PR のあとのやり直し)は Go のテストの対象外で、流していない
- G5(INFO、変更前からの形): 新しい通しのテストは報告の名前を `time.Now().UTC()` で別に求める。`runUpgradeV2` の `now` と UTC の日付の境目(09:00 JST)をまたぐと名前がずれて落ちうる。同じ形が origin/main の `upgrade_v2_test.go` に 11 か所あり、`TestRunUpgradeV2_SameDayRerun_KeepsEarlierReports` は 3 回の実行で窓が少し広い。今回の回では起きていない
- 計画の範囲外(計画の Progress に記録済み): 移行の報告 `ralph-migration-<date>.md` は同じ日の再移行で今も上書きされる
- 変異のテストは依頼に従い流していない。AC5 の識別力は fixture の値から読んだだけである

## Verdict

- Pass: run-test.sh rc 0(1,303 秒、02:41:27〜03:03:10 JST、1 分の load 5.86〜12.20)。shell 45 本すべて OK(PASS を含む行 5,195)、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok で `(cached)` なし: `internal/org` 722.6 秒、`internal/cli` 551.5 秒、`internal/config` 12.4 秒、`internal/gitroot` 5.5 秒、`internal/org/driver` 3.7 秒、`internal/org/protocol` 2.7 秒、`internal/upgrade` 2.5 秒、`internal/scaffold` 2.0 秒、`internal/insights` 1.7 秒、`cmd/ralph` 0.8 秒。名前を絞った回は `internal/upgrade` が PASS 61・SKIP 0(上位 26 本)、`internal/cli` が PASS 55・SKIP 0(上位 23 本)
- Fail: 0
- Blocked: 0
- Verdict: pass。G1 と G2(エラーの分岐 2 つにテストが無い。どちらも throwaway のテストで正しく失敗することは確かめた)は、/pr の Known gaps に書くか、follow-up で 1 本ずつ足す扱いを提案する
