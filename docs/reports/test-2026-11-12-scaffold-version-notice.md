# Test report: scaffold-version-notice

- Date: 2026-11-12(run-test.sh は 18:54:16 から 19:16:39 JST)
- Plan: `docs/plans/active/2026-11-12-scaffold-version-notice.md`(`## Test plan`、`## Risks and mitigations`、AC1〜AC8)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1)
- Scope: `git diff 9ac72c50...HEAD`(HEAD `a80cbbe9`、origin/main は `6ac8fde4` で取り込み済み)。テストが足されたのは `internal/cli` と `internal/upgrade` で、新しいテストは 20 本(`scaffold_version_notice_test.go` に 10 本、`scaffold_version_notice_cli_test.go` に 7 本、`upgrade_v2_test.go` に 1 本、`internal/upgrade/report_test.go` に 2 本)。既存の 3 本に確かめを足した(`TestRunUpgradeV2_FullPass` と `TestRunMigrateLegacy_HappyPath_Yes` に版の行、`TestStatusCmd_ScaffoldSection_LegacyManifest` は `Version` をテストの中で決める形に変えた)。消えたテストは無い。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。worktree の production code とテストには手を入れていない。変異は `go test -overlay` で当て、前後で `git status --porcelain --ignored` が変わらないことを確かめた
- Evidence: `docs/evidence/test-2026-11-12-scaffold-version-notice.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、runner と meta、1 分ごとの load、対照の回、shuffle の回、変異の harness・差分・結果、coverage を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-12-095417.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,343 秒 |
| └ Go: `internal/cli` | 上位 1,105 本 | ok | 0 | — | 499.9 秒 |
| └ Go: `internal/upgrade` | 上位 116 本 | ok | 0 | — | 3.1 秒 |
| └ Go: `internal/org` | — | ok | 0 | — | 661.2 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights` | — | ok | 0 | — | 13.8、5.7、3.4 秒 |
| └ Go: `internal/scaffold`、`internal/org/driver`、`internal/org/protocol`、`cmd/ralph` | — | ok | 0 | — | 3.8、2.7、1.7、0.7 秒 |
| 対照の回(`-count=1 -v`、`-run 'ScaffoldVersion\|TestStatusCmd_Scaffold\|TestRunUpgradeV2_FullPass\|TestRunUpgradeV2_EmptyManifestVersion\|TestRunMigrateLegacy_HappyPath_Yes'`) | PASS 行 60(subtest 込み) | 60 | 0 | 0 | 5.6 秒 |
| shuffle の回(`-count=3 -shuffle=on`、上の対照の範囲に `TestOrgRetiredStateDirEnv` と `TestRunDoctorOpts_RetiredOrgStateDirEnv` を足した、seed `1794478665603577000`) | 上位 37 本 × 3 | 111 | 0 | 0 | 49.8 秒 |
| 変異 10 個(6 回の overlay の実行と、生き残りの package 全体の 1 回。下の節) | 10 | 死んだ 8、生き残り 2 | — | — | 1 回 2〜14 秒、package 全体は 511.4 秒 |
| coverage(対照と同じ範囲の `internal/cli` と `internal/upgrade` 全体、`-coverprofile`) | — | ok | 0 | — | 数秒 |

run-test.sh の Go の段では、10 パッケージとも `(cached)` にならずに走った。1 分の load は開始時 13.83、走っているあいだ 7.85〜18.65、終了時 13.02 で、ほかのセッションが同じマシンを使っていた。run-test.sh の段が走っているあいだに重ねたのは、shell の段に重ねた対照の回と変異の 6 回(どれも 15 秒以内)だけで、Go の段には何も重ねていない。変異の package 全体の回は run-test.sh が終わった 11 秒後に始めた。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start、end、rc、load を meta に書く)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm` で区切った前景の待ちで終わりを待った。19:04:17 の時点では shell の段、19:07:17 の時点では Go の段だった。

Skipped の 2 行は前の PR の /test と同じで、`tests/test-plan-visual.sh` の browser detection(`/Applications` に browser がある手元の環境による)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらもこの diff と関係が無い。

## 受け入れ条件とテスト

どのテストも run-test.sh と対照の回で PASS した。「変異」の列の M 番号は下の「変異」の節の定義である。

| AC | テスト(ファイル) | 変異で確かめたこと |
| --- | --- | --- |
| AC1 古い雛形の repo で `org start` の 2 つの形が stderr に 1 行を 1 回だけ出す | `TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce`、`TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold`(`scaffold_version_notice_cli_test.go`)。どちらも stderr 全体を 1 行と完全一致で比べる | MJ(task の形で 2 回出す)は task の形のテストで死んだ。MD(`--plan` の形が `.` を読む)は `--plan` のテストで死んだ |
| AC2 `status`(`--json` も)が stderr に 1 行を出し、stdout は変わらない | `TestScaffoldVersionStatus_NotesOnStderrAndLeavesStdout`(dev の CLI との stdout のバイト比較、`--json` の読み込み) | — |
| AC3 `director attach` と `director start` が同じ 1 行を出す | `TestScaffoldVersionDirectorAttach_ReadsTheMainWorktreeScaffold`(3 case)、`TestScaffoldVersionDirectorStart_NotesTheScaffoldOnce` | ME(attach が `.` を読む)は attach の 3 case すべてで死んだ |
| AC4 dev、semver でない版、読めない manifest、manifest 無し、同じか新しい雛形では出さない | `TestScaffoldVersionOlder_ComparesSemver`、`TestScaffoldVersionNotice_SilentCases`(`scaffold_version_notice_test.go`)、`TestScaffoldVersionStatus_SilentCases`(status で 6 case)、attach の「main current, worktree older」の case | 判定は 5 つの呼び出しとも `printScaffoldVersionNotice` → `scaffoldVersionNotice` の 1 か所を通る(verify の報告と同じ読み) |
| AC5 linked worktree から org の動詞を流すと main worktree の manifest で決まる。status は今いる directory の manifest で決まる | `TestScaffoldVersionOrgRoot_LedgerPathNamesTheRootWithoutGit`、`TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree`、`TestScaffoldVersionOrgRoot_OutsideGitIsTheWorkingDirectory`、attach の 3 case、`--plan` のテスト | MF(`orgScaffoldRoot` で toplevel を main worktree より先に見る)は `TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree` と attach の `--state-dir` の case で死んだ。MA(task の形が `.` を読む)と MB(director start が `.` を読む)は package 全体で生き残った。MC(status が org の root を読む)は status の振る舞いを見るテストでは死なず、git の呼び出し回数を数えるテストでだけ死んだ(Test gaps の G1、G2) |
| AC6 semver で比べる | `TestScaffoldVersionOlder_ComparesSemver` の 16 行(AC6 の 4 組、逆向きの prerelease、prerelease の CLI と雛形 `8.1.0`、`v` の片側、build metadata、前後の空白) | — |
| AC7 upgrade の報告の `Generated:` の次に `Version: <from> → <to>`。移行のあとも移行前の版。from が空なら行は無い | `TestRenderUpgradeReport_FromVersion_VersionLineAfterGenerated`、`TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine`(`report_test.go`)、`TestRunUpgradeV2_FullPass`(`1.0.0-test → 1.1.0-test`)、`TestRunUpgradeV2_EmptyManifestVersion_ReportHasNoVersionLine`、`TestRunMigrateLegacy_HappyPath_Yes`(`0.9.0-test → 2.0.0-test` があり、`2.0.0-test → 2.0.0-test` が無い) | MG(`upgrade.go` が CLI の版を from に渡す)は FullPass と EmptyManifestVersion で死んだ。MH(`migrate.go` が作り直した manifest の版を渡す)は migrate のテストで死んだ。MI(from が空でも行を出す)は cli の EmptyManifestVersion と `internal/upgrade` の EmptyFromVersion で死んだ |
| AC8 4 つの動詞の `--help` と README | `TestScaffoldVersionHelp_EachVerbDescribesTheNote`(4 つの Long に段落があること、段落が `run 'ralph upgrade' in <that directory>` を含むこと、1 行 76 文字以内)。README はテストが読まない(verify の照合で確かめ済み) | — |

## 変異

`go test -overlay` で production code の写しを差し替え、worktree のファイルは書き換えていない。harness は perl で、置き換える文字列がファイルに 1 回だけあることと、前後が同じでないことを確かめてから写しを作る。別々のテストにしか効かない変異を 1 回の実行にまとめ、落ちたテストの名前でどの変異が殺したかを分けた。どの実行の log にも `build failed` は無い。

| 変異 | 変えた所 | 結果 | 殺したテスト |
| --- | --- | --- | --- |
| MA | `org.go` の task の形の root を `orgScaffoldRoot(...)` から `"."` に | 生き残り(package 全体でも) | — |
| MB | `org_director_headless.go` の director start の root を `"."` に | 生き残り(package 全体でも) | — |
| MC | `status.go` の root を `"."` から `orgScaffoldRoot(stateDir, stateDirSource)` に | 絞った範囲では生き残り、package 全体では死んだ | `TestStatusCmd_MergeGuaranteeRunsNoGH` の「git rev-parse 4 times in one run; want at most 2」だけ。知らせの中身を見るテストではない |
| MD | `org.go` の `runOrgStartPlan` の root を `"."` に | 死んだ | `TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold` |
| ME | `org_director.go` の attach の root を `"."` に | 死んだ | `TestScaffoldVersionDirectorAttach_ReadsTheMainWorktreeScaffold` の 3 case |
| MF | `orgScaffoldRoot` で `gitroot.Toplevel` を `gitroot.MainWorktree` より先に | 死んだ | `TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree`、attach の `--state-dir` の case |
| MG | `upgrade.go` が `fromVersion` に `Version` を渡す | 死んだ | `TestRunUpgradeV2_FullPass`、`TestRunUpgradeV2_EmptyManifestVersion_ReportHasNoVersionLine` |
| MH | `migrate.go` が `newManifest.Meta.Version` を渡す | 死んだ | `TestRunMigrateLegacy_HappyPath_Yes` |
| MI | `report.go` の `if data.FromVersion != ""` を常に真に | 死んだ | cli の `TestRunUpgradeV2_EmptyManifestVersion_ReportHasNoVersionLine`、`internal/upgrade` の `TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine` |
| MJ | `org.go` の task の形で知らせを 2 回出す | 死んだ | `TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce` |

MA、MB、MC をまとめた overlay を `internal/cli` 全体に当てた回(`nice -n 19`、`-parallel 4`、511.4 秒)で落ちたのは `TestStatusCmd_MergeGuaranteeRunsNoGH` の 1 本だけだった。このテストは status だけを流すので、MA と MB は package 全体でも生き残ったと読める。org start の task の形と director start のテストは git の外で流すので、`orgScaffoldRoot` は `.` を返し、`.` を渡す変異と区別がつかない。

## Coverage

- Statement: 対照と同じ範囲に絞った `internal/cli` の回で、`scaffold_version_notice.go` の関数は `scaffoldVersionOlder` 100.0%、`semverWithV` 100.0%、`scaffoldVersionNotice` 90.9%、`orgScaffoldRoot` 85.7%、`printScaffoldVersionNotice` 100.0%。`internal/upgrade` 全体は 91.2%、`RenderUpgradeReport` は 100.0%
- Branch: 測っていない(Go の標準の道具に branch coverage は無い)
- Function: 上のとおり
- Notes: 実行されなかった block は 2 つ。`scaffold_version_notice.go:68-69`(`filepath.Abs` が失敗したときに `root` をそのまま使う所)と `:99-100`(main worktree が名指せないときに toplevel を返す所。verify の Coverage gaps 3 と同じ)。既存のファイルの変更行(5 つの呼び出し、`upgrade.go:157`、`migrate.go:993`、`upgrade_v2.go:185`、`report.go:61-62`)は、`git diff -U0` の追加行と profile の block を突き合わせて、すべて実行されていた。宣言と定数の行は block を持たない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | run-test.sh、対照、shuffle の回で落ちたテストは無い | — | — |

時間切れも打ち直しも無かった。依頼にあった「package 変数 `Version` が残った値のまま、手で書いた古い manifest で知らせが出て落ちる」型の失敗は、この run では 1 本も出ていない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 計画の Regression tests: `internal/cli` と `internal/upgrade` の全体が通る(ほかのテストが残した `Version` と手書きの古い manifest で知らせが stderr に混ざらない) | Pass | run-test.sh で `internal/cli` 499.9 秒、`internal/upgrade` 3.1 秒、どちらも ok で `(cached)` ではない |
| 同じ型の危険があるテストの洗い出し | 該当なし | manifest を書くテストファイルのうち、4 つの動詞を流すのは 5 つ(`cli_test.go`、`org_retired_state_dir_env_test.go`、`scaffold_version_notice_test.go`、`scaffold_version_notice_cli_test.go`、`status_scaffold_test.go`)。`status_scaffold_test.go` の manifest は `initV2Project`(`Version` を決めてから `executeInit`)か、S2 で直した `LegacyManifest`(`Version` を manifest と同じ版にして `t.Cleanup` で戻す)か、壊れた TOML で、どれもテストの順に依らずに黙る。`org_retired_state_dir_env_test.go` の status と org start は manifest の無い repo で流れる |
| テストの順に依らないこと | Pass | 上の範囲を `-shuffle=on -count=3` で流して 111/111。package 全体の shuffle は流していない(Test gaps の G4) |
| S2 で直した `TestStatusCmd_ScaffoldSection_LegacyManifest` | Pass | run-test.sh と対照の回。直し方は `Version` を manifest と同じ `0.9.0-test` にするもので、このテストが見る scaffold の節(layout と ownership)の assert は変わっていない |

## Test gaps

- G1(AC5): org start の task の形と director start が main worktree の manifest を読むことは、テストで確かめていない。どちらのテストも git の外で流すので、root に `.` を渡す変異(MA、MB)が `internal/cli` 全体で生き残る。`--plan` の形と attach は linked worktree から確かめているので、同じ `orgScaffoldRoot` を通ることはコードで分かる。足すなら、attach のテストと同じ `newLegacyLedgerRepo` で main 7.0.0、linked 8.1.0 を作り、linked worktree から task の形の `org start` と `director start` を流して main の root が名指されることを見る 1 case ずつが最小
- G2(AC5): status が今いる directory の manifest を読むことを、linked worktree から確かめるテストは無い(verify の Coverage gaps 2)。MC は知らせの中身を見るテストでは死なず、`TestStatusCmd_MergeGuaranteeRunsNoGH` が git の呼び出し回数で拾っただけだった。足すなら、linked worktree の manifest を 7.0.0、main を 8.1.0 にして status を流し、知らせが linked worktree の path を名指すことを見る 1 case
- G3: `orgScaffoldRoot` の toplevel の段(main worktree の無い bare の linked worktree)と、`filepath.Abs` の失敗の段は実行されていない
- G4: `internal/cli` の package 全体を `-shuffle=on` で流していない。`Version` を戻さない既存のテストは多く、順を変えたときの影響は、manifest と 4 つの動詞が出会うテストを洗い出して確かめた範囲(上の Regression checks)に限る
- G5: `org start` の 2 つの形と `director start` を本物の herdr と座席で流していない(verify の Not verified と同じ)。CLI のテストは stub の herdr で確かめている
- G6: `-race` は流していない。この diff は goroutine にも共有の状態にも触れない

## Verdict

- Pass: run-test.sh rc 0(1,343 秒、18:54:16〜19:16:39 JST、1 分の load 7.85〜18.65)。shell 45 本すべて OK、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok: `internal/cli` 499.9 秒、`internal/org` 661.2 秒、`internal/config` 13.8 秒、`internal/gitroot` 5.7 秒、`internal/scaffold` 3.8 秒、`internal/insights` 3.4 秒、`internal/upgrade` 3.1 秒、`internal/org/driver` 2.7 秒、`internal/org/protocol` 1.7 秒、`cmd/ralph` 0.7 秒。この diff の新しいテスト 20 本と確かめを足した 3 本も、対照の回と shuffle の回で通った。変異 10 個のうち 8 個が死んだ
- Fail: 0
- Blocked: 0
- Verdict: pass。G1 と G2(AC5 のうち org start の task の形、director start、status を linked worktree から見るテストが無い)は、/pr の Known gaps か follow-up に書く扱いを提案する
