# Test report: org-state-dir-local

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-org-state-dir-local.md`(`## Test plan` と AC1〜AC11)。Refs #276(A2)
- Tester: tester サブエージェント(`/test`、cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base c03647f9、HEAD e5d5a5a4)。Go は `internal/config`、`internal/org`、`internal/cli`、ほかに skill・文書・template。`./scripts/run-test.sh` は changed scope のまま打った。detector は `templates/base/ralph.toml` を分類できず full に落ち(`unclassified:templates/base/ralph.toml`)、選ばれた pack は golang だけだった。test mode なので `tests/test-*.sh` の 45 本もすべて走った。テストの関数はこの diff で 41 本増え、2 本が消えた(`TestResolveOrgStateDir_EnvWinsOverGitAndCwd` と `TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv`。env の段を外したため。個人用のファイルの段のテストに置き換わっている)
- Evidence: `docs/evidence/test-2026-10-30-org-state-dir-local.md`(要点。commit する)、`docs/evidence/test-2026-10-30-org-state-dir-local.log`(gitignore。`run-test.sh` の全出力 5,194 行、`go test -list` の本数、coverage の数字、変異 5 個の差分と FAIL の行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-30-034021.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 4,796)、Go 10 パッケージ ok | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.50.1 で、古い git は stub の行が見る) | 1,375 秒(12:40:21〜13:03:15 JST)。開始時の load 9.32 / 21.81 / 47.67、終了時 9.05 / 9.23 / 17.00。途中の見本は 18.7、9.6、8.6 |
| └ Go: `internal/org` | 1,285 本(トップレベル、`go test -list`) | ok | 0 | 0 | 721.6 秒 |
| └ Go: `internal/cli` | 910 本(同) | ok | 0 | 0 | 505.4 秒 |
| └ Go: `internal/config` | 90 本 | ok | 0 | 0 | 19.6 秒 |
| └ Go: `cmd/ralph`、`internal/{gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | 2、6、41、73、15、32、114 本 | ok | 0 | 0 | 0.9〜5.5 秒 |
| coverage の回: `go test -count=1 -coverprofile ./internal/config/` | 90 | ok | 0 | 0 | 45 秒(load 約 10) |
| coverage の回: `go test -count=1 -coverprofile -run 'ResolveOrgStateDir\|FeatureRepoRoot\|LedgerMainWorktreeRoot\|Escalat\|StateDir' ./internal/org/` | — | ok | 0 | 0 | 53 秒 |
| coverage の回: `go test -count=1 -coverprofile -run 'OrgLedger\|RetiredStateDirEnv\|RetiredEnv\|Insights\|Doctor\|Legacy\|ConfigGet\|OrgCleanup\|StateDirHint\|MergeGuarantee\|LocalOnlyKey\|OrgEscalate' ./internal/cli/` | — | ok | 0 | 0 | 385 秒 |
| 変異 5 個(`go test -overlay`、下の Regression checks) | 5 | 5 個とも落ちた | — | — | 1 個 31〜247 秒 |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。小さな runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打つ前の load は 1 分 9.32・5 分 21.81 で、lead の条件(30 未満)を満たしていた。shell の SKIP の行は前の回と同じ 1 行だけだった。

`run-test.sh` の `go test ./...` は 1 回で全パッケージを `(cached)` なしで打ち、30 分の時間切れにならなかった。そのため internal/org と internal/cli を `-list` で分けて打ち直す必要はなかった。既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` もこの回は通った。打ち直しはしていない。

### この diff に関わる shell のテスト

`scripts/verify.local.sh` の test mode は `tests/test-*.sh` を一覧ではなく glob で全部打つので、diff に関わるものも範囲に入っている。この回の結果:

| テスト | PASS の行 | 結果 | この diff との関わり |
| --- | --- | --- | --- |
| `tests/test-check-sync.sh` | 17 | OK | skill・recipe・template の写しの同期の検査そのもののテスト |
| `tests/test-check-skill-sync.sh` | 14 | OK | `/org`・`/director` skill の 4 つの写し |
| `tests/test-sync-skills.sh` | 23 | OK | 同上(写しの生成) |
| `tests/test-check-template.sh` | 60 | OK | `templates/base/ralph.toml` を含む scaffold(G 行は新しい `ralph init` の scaffold が通ることを見る) |
| `tests/test-template-purity.sh` | 11 | OK | template の文書の変更 |
| `tests/test-ralph-config.sh` | 19 | OK | config を読む script。`RALPH_ORG_STATE_DIR` を読む script は `scripts/` にもともとない(grep で 0 件) |
| `tests/test-detect-changed-languages.sh` | 84 | OK | `run-test.sh` の scope の判定 |

`tests/test-insights-append.sh` と `tests/test-ralph-worktree.sh` にも `--state-dir` や `state_dir` の文字があるが、それぞれ標準フローの pipeline の state と worktree の記録の場所で、org の台帳とは別のもの。どちらも OK だった。shell の 45 本の一覧と各 PASS の行数は evidence の md にある。

## Coverage

- Statement: `internal/config` はパッケージ全体で 96.7%。変更の中心の関数は、絞った回の profile で次のとおり。
  - config: `LocalOrgStateDir` 100%、`LocalOnlyKeys` 100%、`checkLocalOnlyKeys` 88.9%、`Load` 90.9%
  - org: `ResolveOrgStateDir` 100%、`LedgerMainWorktreeRoot` 100%、`EscalateNotRecorded` 100%、`MainWorktreeRoot` 87.5%、`FeatureRepoRoot` 85.7%、`mustAbs` 75.0%
  - cli: `resolveOrgLedger` 100%、`guardLegacyOrgStateDir` 100%、`checkRetiredEnv`(doctor)100%、`mergeGuaranteeEnvFromOS` 100%、`newInsightsCmd` 100%、`checkRetiredOrgStateDirEnv` 92.9%、`runDoctorStateDir` 95.9%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes: 打たれていないブロックは次のとおり。
  - `checkRetiredOrgStateDirEnv` の `filepath.Abs` の失敗(`internal/cli/org_legacy_ledger.go:113-114`)と `mustAbs` の失敗(`internal/org/statedir.go:222-223`)。どちらも `os.Getwd` が失敗したときだけ通る守りの道
  - `checkLocalOnlyKeys` の `toml.Unmarshal` の失敗(`internal/config/local.go:130-131`)。`Load` は同じ data を先に Unmarshal してから(`internal/config/config.go:728`)呼ぶので、Load からは届かない
  - `FeatureRepoRoot` の「主 worktree のない bare repository の linked worktree」のエラー(`internal/org/statedir.go:154-155`)。main からある分岐で、main では source `flag` と `env` のときに通り、この diff では `env` の代わりに `config` で通る。Test gaps の 1
  - `runDoctorStateDir` の 181-182 行と 203-204 行、`runInsights` の 73-74 行と 84-85 行は、この diff より前からある読み込みの失敗の道
  - org と cli のパッケージ全体の数字は、絞った回なので意味がない(org 22.8%、cli 55.4%)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

`run-test.sh`、coverage の 3 回のどれにも失敗はなかった。

## Test plan とテストの対応

| 計画の項目 | テスト(この diff で増えたもの) |
| --- | --- |
| config: `[org].state_dir` の読み込み(絶対、`~/`、空)、相対の拒否、`~` だけ、`~user/…` | `TestLocalOrgStateDir_Values`(`internal/config/local_test.go:713`)、`TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute`(`:750`、`~` と `~user/ledger` と先頭の空白を含む)、`TestLocalOrgStateDir_FileErrorsNameItsPath`(`:768`、TOML でない、文字列でない、home がない) |
| config: `ralph.toml` の LocalOnlyKeys のエラー、template が持たない | `TestLoad_LocalOnlyKeyInRalphToml`(`:580`)、`TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults`(`:615`)、`TestLocalOnlyKeys`(`:672`)、`TestTemplateDriftProbesHoldNoLocalOnlyKey`(`internal/config/defaults_sync_test.go:588`)、`TestConfigGet_RalphTomlThatSetsALocalOnlyKeyExits3`(`internal/cli/config_test.go:338`) |
| config: `RALPH_LOCAL_CONFIG=off`(AC9) | `TestLocalOrgStateDir_LocalConfigEnv`(`internal/config/local_test.go:820`。off では git も打たない) |
| org: `ResolveOrgStateDir` の各段と順、error | `TestResolveOrgStateDir_PersonalFileTier`(`internal/org/statedir_test.go:277`)、`_PersonalFileOrder`(`:330`)、`_PersonalFileErrors`(`:392`)、`_RetiredEnvIsIgnored`(`:89`) |
| org: `LedgerMainWorktreeRoot`・`FeatureRepoRoot` の source `config`(AC8) | `TestFeatureRepoRoot`(`:515`)と、既存の `TestLedgerMainWorktreeRoot` に足した config の行 |
| cli: 個人用のファイルで `ralph org status`・`ralph status`・`ralph insights` が同じ台帳を読む。linked worktree とサブディレクトリ、`--config` | `TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree`(`internal/cli/org_ledger_config_test.go:62`)、`TestOrgLedger_ConfigFlagDoesNotChooseThePersonalFile`(`:109`)、`TestInsights_PersonalFileNamesTheReceiptsLedger`(`:415`)、`TestLocalOrgStateDir_ReadsTheMainWorktreesFile`(`internal/config/local_test.go:884`、bare repository の linked worktree を含む) |
| cli: doctor の fail の検査と続行、`--state-dir`、env の warn | `TestRunDoctorFull_PersonalFileThatCannotResolveTheLedgerIsAFailLine`(`org_ledger_config_test.go:273`)、`TestDoctor_StateDirFlagChoosesTheLedgerOverThePersonalFile`(`:341`)、`TestMergeGuaranteeEnvFromOS_StateDirFlagNamesTheLedgerWithoutGit`(`internal/cli/doctor_merge_guarantee_test.go:399`)、`TestRunDoctorOpts_RetiredOrgStateDirEnv`(`internal/cli/org_retired_state_dir_env_test.go:411`)、`TestCheckRetiredEnv`(`:368`)、`TestRunDoctorOpts_LocalOnlyKeyInRalphTomlUsesDefaults`(`internal/cli/doctor_org_test.go:1258`) |
| 逃げ道(AC3、AC5): 相対の値と TOML でないファイルで、`--state-dir` 付きの `org stop` と `org status` | `TestOrgLedger_RelativePersonalStateDirFailsUntilStateDirIsPassed`(`org_ledger_config_test.go:133`)、`TestOrgLedger_UnparsablePersonalFileNamesItsPath`(`:175`)、`TestInsights_BrokenPersonalFileFailsUntilStateDirOrReceiptsIsPassed`(`:439`)、`TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman`(`:206`) |
| env の拒否(AC6): start・spawn・send(TASK)が exit 1、manifest・見張り・driver に触れない。status と org status は注意で続く。`--state-dir`・個人用のファイル・同じ場所なら注意だけ | `TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger`(`org_retired_state_dir_env_test.go:143`)、`_ReadOnlyVerbsShowTheDefaultLedgerWithANote`(`:101`)、`_OnlyANoteWhenTheLedgerIsChosenOrTheSame`(`:206`)、`_EscalateStillReachesTheHuman`(`:279`)、`_NamingTheLegacyLedgerRefusesStop`(`:336`)、`TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger`(`internal/cli/org_cleanup_test.go:597`) |
| AC7: insights と doctor の `--state-dir`、`--receipts` が勝つ | `TestInsightsCmd_StateDirFlagNamesTheReceiptsLedger`(`internal/cli/insights_test.go:319`)、`TestInsightsCmd_ReceiptsFlagOverridesStateDir`(`:337`)、上の doctor の 2 本 |
| 回帰: hint の `--state-dir`、cleanup の台帳 | `TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromThePersonalFile`(`internal/cli/org_test.go:2234`)、`TestOrgCleanup_PersonalFileNamesTheLedger`(`org_cleanup_test.go:646`)、`TestEscalateNotRecorded_WithNoLedgerWritesNothing`(`internal/org/escalate_test.go:1287`) |
| 回帰: 既存の org と cli のテスト | `run-test.sh` の中で org 1,285 本、cli 910 本がすべて通った |
| config get | `TestConfigGet_OrgStateDirPrintsThePersonalFilesValueAsWritten`(`internal/cli/config_test.go:321`) |

どれも `run-test.sh` の中で通った。

## Regression checks

実装の各スライスで implementer が変異 41 個を確かめている(計画の Progress)。この回では、計画の中心の判定 5 か所を選び、独立に確かめ直した。worktree のファイルは変えず、1 か所だけの文字列の置き換えを `go test -overlay` で渡し、関係するテストに `-run` で絞った。前後の `git status --porcelain --ignored` は同じだった。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| m1 個人用のファイルの段を飛ばす(`statedir.go:80` の `if local != ""` を `if false && …` に) | 落ちた | `TestResolveOrgStateDir_PersonalFileTier` とサブテスト 8 件 |
| m2 env が別の台帳を指しても台帳を変えるコマンドを断らない(`org_legacy_ledger.go:121`) | 落ちた | `TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger`、`_EscalateStillReachesTheHuman`、`_NamingTheLegacyLedgerRefusesStop`、`TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger` |
| m3 source `config` を注意だけの側から外す(`org_legacy_ledger.go:116`) | 落ちた | `TestOrgRetiredStateDirEnv_OnlyANoteWhenTheLedgerIsChosenOrTheSame/[org].state_dir_naming_another_ledger` |
| m4 `LedgerMainWorktreeRoot` が source `config` を `flag` と同じに扱わない(`statedir.go:127`、AC8) | 落ちた | `TestLedgerMainWorktreeRoot` |
| m5 個人用のファイルのエラーを捨てて次の段に進む(`statedir.go:77`、D4) | 落ちた | `TestResolveOrgStateDir_PersonalFileErrors` の 4 件(相対、`~` だけ、`~user`、TOML でない) |

## Test gaps

1. `FeatureRepoRoot` の bare repository のエラーの分岐(`internal/org/statedir.go:154-155`)を打つテストがない。main からある分岐で、この diff では `--state-dir` か `[org].state_dir` で台帳を決めたときに通る(main では `--state-dir` か env のとき)。`TestResolveOrgStateDir_BareDotGitWorktreeFallsBackToToplevel` と同じ fixture で `FeatureRepoRoot(<dir>, "config")` がエラーを返すことを見る 1 行で閉じられる
2. `filepath.Abs` の失敗(`org_legacy_ledger.go:113-114`、`statedir.go:222-223`)は打たれていない。cwd が消えたときだけ起きる守りの道
3. 実物の `ralph.local.toml` を置いた開発機での手動の確かめ(実際の HOME の `~/` 展開、別の端末の env が残った状態での start の拒否)はしていない。どれも temp の repository と `t.Setenv("HOME", …)` のテストで見ている
4. release のノートに書く「`--state-dir` 付きで起こした見張りの子が env を受け継ぐと watch.log に注意が 1 行出る」(S4 の報告)を、実際の `ralph org watch --detach` で見たテストはない。害のない注意の行

どれも計画の判定(台帳の決め方の順、env の拒否、source `config` の扱い)を変えない道で、その判定は上の変異で落ちることを確かめた。

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 45 本 OK、Go 10 パッケージ ok)、coverage の 3 回 ok、変異 5 個すべてがテストで落ちた。AC11 のうちテストの部分は満たす(静的の部分は verify の report で pass)
- Fail: 0
- Blocked: 0
