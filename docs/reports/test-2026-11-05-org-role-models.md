# Test report: org-role-models

- Date: 2026-11-05(run-test.sh は 21:49:07 から 22:14:01 JST)
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(`## Test plan`、AC1〜AC13)。spec は `docs/specs/2026-11-05-org-role-models.md`
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff d631e9f8...HEAD`(HEAD fd107026)。`./scripts/run-test.sh` は changed scope のまま打った。`templates/base/ralph.toml` を分類できないので full fallback になり(`.harness/state/verify-scope` は `scope=full`、`reason=unclassified:templates/base/ralph.toml`、`languages=golang`)、Go の全パッケージと `tests/test-*.sh` の 45 本が走った。production code は変えていない。gap を埋めるテストを 2 本、新しいファイルに書いた(下の「足したテスト」。どちらも未追跡で、コミットはしていない)
- Evidence: `docs/evidence/test-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、足したテストの 3 回の結果、5 つの変異の定義と結果、coverage の数字と変更した行の coverage を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-05-124908.log`(同じく対象外。名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope → full fallback、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 1,494 秒。load は開始時 6.32 / 13.18 / 17.92、終了時 5.76 / 14.74 / 19.85 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 711.5 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 523.4 秒 |
| └ Go: `internal/config`、`internal/insights`、`internal/upgrade`、`internal/gitroot` | — | ok | 0 | 0 | 13.2 秒、4.1 秒、4.0 秒、3.8 秒 |
| └ Go: `internal/scaffold`、`internal/org/driver`、`internal/org/protocol`、`cmd/ralph` | — | ok | 0 | 0 | 3.1 秒、3.1 秒、1.6 秒、0.8 秒 |
| coverage の回: `go test -count=1 -coverprofile ./internal/config/ ./internal/org/... ./internal/cli/`(HEAD の `git archive` の写し。足したテストは入っていない) | — | 5 パッケージ ok | 0 | 0 | 805 秒(org 796.1 秒、cli 486.3 秒) |
| 足したテスト 2 本を単独で `-count=3` | 2 本(subtest 5 つ)× 3 | すべて PASS | 0 | 0 | 12.7 秒 |
| 変異 5 つ(`go test -overlay`、下の「足したテスト」) | 5 | 足したテストがすべて殺した | — | — | 1 つ 2〜3 秒 |

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、`run_in_background` で runner(`/tmp/test-orm/runner.sh`)を起動し、log と開始・終了・rc の meta を書かせた。同じ手番の中で runner の pid を `caffeinate -w` で待ち、meta を読んだ。走っているあいだに、変異を `-run` で絞った回(長いもので 249 秒)を並べて打ったので、Go の段の途中で 1 分の load が 43.78 まで上がった。それでも `internal/org` は 711.5 秒で、30 分の時間切れの 40% に収まった。

Skipped の 2 行は、どちらもこの diff と関係のない手元の環境による。2026-11-02 の各回と同じ 2 行。

- `tests/test-plan-visual.sh` の「browser detection (a browser in /Applications would start)」。本物のブラウザを起こさないための SKIP
- `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git は 2.49.0 で、古い git は stub の行が見ている

Go の 10 パッケージはどれも `(cached)` ではなかった。shell の 45 本と PASS の行の数(5,195)、log の行数(5,617)は 2026-11-02 の standard-config-gaps の回と同じで、この diff は shell のテストに何も足していない。

## Coverage

- Statement(HEAD の写し、足したテストなし): `internal/config` 96.6%、`internal/org` 95.0%、`internal/org/driver` 96.1%、`internal/org/protocol` 100.0%、`internal/cli` 90.4%
- 変更した行: `git diff -U0 d631e9f8...HEAD` で足された行(テストを除く `.go`)を含む profile のブロックは 320 statement で、そのうち 314 が打たれていた(98.1%)。行が足されたテスト以外の `.go` は 23 ファイルで、そのうち 21 ファイルの足された行が statement のブロックに入る(残りの `internal/cli/org_director.go` と `internal/org/manifest.go` はコメントと構造体の項目だけ)。打たれていない 6 つは次のとおり。
  - `internal/cli/org_director_headless.go:335-336`: director start で `org.ResolveRoleModel` が断ったときの return。届く道で、足したテスト `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused/director_start` が打つ(そのテストだけの profile で count 1 を確かめた)。
  - `internal/cli/doctor.go:1074`: 残ったキーの warn の `default` の分岐。キーは `[org].model_pool` と `[org.roles]` の 2 つだけなので、今は届かない。
  - `internal/config/config.go:896-897`、`internal/config/role_models.go:144-145`、`:170-171`: `ralph.local.toml` か各ファイルを probe 用に読み直したときの `toml.Unmarshal` の失敗。同じ data をその前に `toml.Unmarshal(…, &cfg)` で読んでいて、壊れた TOML はそこで先に失敗するので、届かない。
  - `internal/config/role_models.go:102-103`: `fillDefaultRoleModels` の `RoleModels == nil` の初期化。`Load` は表を空の map から読む(`config.go:864`)ので、`Load` からは届かない。
- Branch: 計測なし(Go の道具は statement だけ)
- Function: パッケージの数字のみ
- Notes: tester の memory にある前の値は、config 97.0%(2026-10-29)、org 95.0%(2026-10-23)、driver 95.3%(2026-10-29)、cli 88.4%(2026-10-22)。config の 0.4 ポイント減は、`internal/config` に入った届かない分岐 4 つ(上の `config.go` と `role_models.go` の分)によるものとみられる。消えたコードで分母も変わったので、内訳は確かめていない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、coverage の回、足したテストの 3 回のどれにも失敗はなかった。flaky として memory にあるテストも落ちなかった。

## Test plan とテストの対応

テスト名の位置は HEAD fd107026 のもの。どれも run-test.sh の中で通った。

### Unit: `internal/config`

| 計画の項目 | テスト |
| --- | --- |
| 既定の 2 組の選び方(AC1) | `TestDefaultRoleModels_PicksTheSetByDriverPool`(`internal/config/role_models_test.go:65`)、`TestDefault_RoleModelsIsTheClaudeSet`(`:89`)、`TestLoad_RoleModelsFollowTheDriverPoolInEffect`(`:109`。ファイルなし、codex だけの `ralph.toml`、`ralph.local.toml` で codex に絞る、claude を戻す) |
| 1 役割だけの上書き(AC1) | `TestLoad_RoleModelsKeepWhatTheFilesSet`(`:225`、「one role in ralph.toml」「one table-header role with driver_pool codex」) |
| `ralph.local.toml` のインラインと表見出し(AC2) | 同上の「ralph.local.toml overrides one role, inline / table header」 |
| キーが欠けたエントリ、ファイルごと(AC2) | `TestLoad_RoleModelsEntryMustSetBothKeys`(`:293`。インラインと表見出し、`ralph.toml` と `ralph.local.toml`、表見出しで 1 キーだけの上書き、キーのない表) |
| `claude`・`codex` 以外の driver は失敗、`driver_pool` にない driver は読み込みが通る(AC3、D9) | `TestLoad_RoleModelsValues`(`:328`)、`TestLoad_RoleModelsDriverOutsideDriverPoolLoads`(`:274`)。`driver_pool` 自体の検査は `TestLoad_DriverPoolMustNameKnownDrivers`(`:152`、S5b) |
| 文字の規則(空白、引用符、`;`、`$`、先頭の `-`、`""`)(AC3) | `TestCheckOrgModelName`(`:377`)、`TestLoad_RoleModelsValues`。「model empty」は `TestLoad_RoleModelsKeepWhatTheFilesSet` |
| 残ったキーの記録、どちらのファイルが書いたか(AC8、D1) | `TestLoad_LegacyModelKeysRecordsEachFile`(`:402`)、`TestLoad_LegacyModelKeysLoadInAnyShape`(`:476`) |
| 改名前の役割名 | config の層には専用のテストはない。`TestRetiredRoleConfigKeys`(`internal/org/prompts_test.go:2189`、`[org.role_models].lead`)と CLI の `TestOrgSpawnAndStart_RetiredLeaderConfigKey_Rejected`(`internal/cli/org_test.go:3206`)が見る |
| ロックステップ(AC11) | `TestDefaultsLockStep` の role_models の sub-test(`internal/config/defaults_sync_test.go:212`)、`TestRoleModelsTemplateDrift_EachRoleIsComparedOnBothSides`(`internal/config/defaults_sync_role_models_test.go:263`)、`TestRoleModelsTemplateDrift_DetectsASetTableAndMalformedBlocks`(`:333`) |

### Unit: `internal/org`

| 計画の項目 | テスト |
| --- | --- |
| `ResolveRoleModel` の表(AC4 の 6 通り、`model` が `""` で `--model` を渡す場合、`driver_pool` にない driver)(D4、D9) | `TestResolveRoleModel`(`internal/org/role_models_test.go:37`、20 ケース。`-v` の subtest の数で数えた)、`TestResolveRoleModel_DefaultsOfConfigDefault`(`:121`)、`TestDefaultRoleModels_KeysAreTheOrgRoles`(`:153`) |
| `ValidateSpawnEnvelope` の文字の検査 | `TestValidateSpawn`(`internal/org/envelope_test.go:83`。許可リストがないこと、`""` が通ること、5 つの悪い名前)、`TestOrgSpawn_Rejected_ModelNameNotOneWord`(`internal/org/spawn_test.go:876`) |
| 空のモデルの argv と receipt(AC6、D5) | `TestOrgSpawn_EmptyModel_PassesNoModelFlagAndMarksSpawnStarted`(`internal/org/spawn_role_models_test.go:126`)、`TestOrgSpawn_DryRun_EmptyModel_MarksSpawnStarted`(`:190`)、`TestOrgSpawn_Codex_EmptyModel_ReceiptUnknownWithObservedModel`(`:212`)、`TestCodexFoundReceipt_NoCommandedModel`(`:334`) |
| spawn の観測が時間切れになった座席の stop の receipt と、印のない古い event(AC6) | `TestOrgStop_Codex_EmptyModel_ObservedAtStop`(`:246`)、`TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation`(`:288`)、既存の `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved`(`internal/org/verbs_test.go:5223`) |
| `EnvelopeSummary` の決まった順の出力(AC9) | `TestEnvelopeSummary_ListsRoleModelsMaxSeatsAndPermissionDefault`(`internal/org/envelope_summary_test.go:17`)、`TestEnvelopeSummary_DefaultRoleModels`(`:40`)、`TestEnvelopeSummary_EmptyRoleModels_NoneConfiguredMarker`(`:48`) |
| `RetiredRoleConfigKeys` の role_models | `TestRetiredRoleConfigKeys`(`internal/org/prompts_test.go:2189`) |
| `[org.roles]` の拒否で manifest と receipt に何も書かれない(AC8、D7) | `TestOrgSpawn_LegacyRolesTable_RefusedBeforeAnyWrite`(`internal/org/spawn_role_models_test.go:29`)、`TestOrgSpawn_LegacyRolesTable_AfterTheSpawnKeepsTheRetryIdempotent`(`:69`)、`TestOrgSpawn_LegacyModelPoolRecord_DoesNotRefuse`(`:95`)、`TestOrgSpawn_LegacyRolesTable_ComesBeforeRetiredRoleKeys`(`:110`)、`TestLegacyRolesErr`(`internal/org/role_models_test.go:185`)、`TestLegacyModelPoolNotice`(`:225`)、`TestSpawnPrecheckErr_MatchesSpawn`(`internal/org/feature_test.go:1772`) |
| director の登録と起動スクリプトの空のモデル(AC6) | `TestDirectorAttachHeadless_EmptyModelIsRecorded`(`internal/org/director_role_models_test.go:24`)、`TestDirectorAttachHeadless_RefusesAModelThatIsNotOneWord`(`:56`)、`TestStartDirector_EmptyModel_HerdrAgentStartHasNoModelFlag`(`:89`)、`TestDirectorLaunchScript_EmptyModel_ExecsWithoutModelFlag`(`:109`)、`TestStartDirector_EmptyModel_LaunchCmdAgentGetsNoModel`(`:128`)、`TestDirectorAgentArgs`(`:154`) |
| 表示の `(CLI default)`(AC6) | `TestModelOrCLIDefault`(`internal/org/role_models_test.go:261`)、`TestBuildOrgReport_EmptyModelShowsCLIDefault`(`internal/org/report_test.go:243`) |
| probe の空のモデル(AC9) | `TestProbeModel_EmptyModelOmitsModelFlag`(`internal/org/driver/probe_test.go:39`) |

### Integration: `internal/cli`

| 計画の項目 | テスト |
| --- | --- |
| spawn の `--dry-run` をフラグなし・`--model` だけ・`--driver` だけで(AC5) | `TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel`(`internal/cli/org_role_models_test.go:64`、9 ケース。stderr が空であることも見る)、`TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`(`:131`、6 ケース) |
| start `<task>` と start --plan(AC5) | `TestOrgStart_RoleModels_LeaderEntryWithoutDriverOrModel`(`:192`)、`TestOrgStart_ModelFlagOmitted_TakesTheLeaderRoleModel`(`internal/cli/org_test.go:2629`)、`TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit`(`:2687`)、`TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml`(`org_role_models_test.go:475`) |
| director start(herdr と `launch_cmd`)(AC5、AC6) | `TestOrgDirectorStart_RoleModels_NoFlagsLaunchesTheCLIDefault`(`:287`)、`TestOrgDirectorStart_RoleModels_LaunchCmdWithoutModel`(`:343`) |
| main worktree と feature worktree の `ralph.toml` が違う台帳での読み先(AC7) | `TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml`(`:394`、6 ケース)、`TestOrgSpawn_MainWorktreeRalphToml_OrgWideValuesTakenMaxSeatsNot`(`internal/cli/org_reserve_test.go:496`)、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`(`internal/cli/org_director_headless_test.go:450`) |
| 写しの古い `driver_pool` と main の `ralph.local.toml` の codex の役割(AC7、計画の助言 1) | `TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml` の該当ケース |
| `model_pool` が残った設定の stderr の 1 行(AC8) | `TestOrgSpawn_RoleModels_LegacyModelPoolNoticeOnce`(`org_role_models_test.go:250`)、start と director start の各テストの stderr の比較 |
| `[org.roles]` が残った設定の拒否(AC8) | `TestOrgSpawn_LegacyOrgRolesRefused`(`internal/cli/org_test.go:819`、本番と dry-run)、`TestOrgDirectorStart_RefusedWithNothingWritten` の「a config that still sets [org.roles]」(`org_director_headless_test.go:243`) |
| その設定での `status`・`stop`・`disband`(AC8、spec 受け入れ条件 11) | このサイクルまで Go のテストがなかった(verify の Coverage gaps)。足したテスト `TestOrgCleanupVerbs_LegacyModelKeys_StillWork` が埋める |
| doctor の 4 つの検査と warn(AC9) | `TestDoctorCheckOrgEnvelope_ReportsRoleModelsAndMaxSeats`(`internal/cli/doctor_org_test.go:181`)、`TestCheckCodexModelSlugs_RoleModelsEmptyModelAndDuplicates`(`internal/cli/doctor_role_models_test.go:37`)、`TestCheckOrgModelProbes_RoleModelsPairsDedupedAndEmptyModelWithoutFlag`(`:101`)、`TestDoctorCheckOrgLegacyModelKeys`(`:211`)、`TestDoctorCheckOrgRoleModelDrivers`(`:260`)、`TestRunDoctorOpts_RoleModelsChecksWired`(`:310`)、`TestCodexSeatPossible`(`internal/cli/doctor_codex_writable_root_test.go:1451`)、`TestCheckCodexAgmsgWritableRoot_CodexInDriverPoolWithoutCodexRoleEntry_StillChecked`(`:880`) |
| init の 3 通りの出力を `config.Load` で読む(AC10) | `TestInitSettings_EveryAnswerLoadsFromTheTemplateWithTheTableValues`(`internal/cli/init_settings_test.go:102`)、`TestRunDoctorOpts_InitSettingsForEachCLIPass`(`internal/cli/init_settings_write_test.go:374`) |
| 表示の `(CLI default)`(AC6、S7 の L1) | `TestOrgStatus_EmptyModelShowsCLIDefault`(`internal/cli/org_model_display_test.go:45`)、`TestStatus_SeatTableShowsCLIDefaultForEmptyModel`(`:73`)、`TestOrgDirectorStatus_HeadlessEmptyModelLine`(`:121`) |
| flag の help(AC5) | `TestOrgSpawnStartAndDirectorStartHelp_RoleModels`(`org_role_models_test.go:539`) |

### Edge cases

| 計画の項目 | テスト |
| --- | --- |
| 役割の名前に改名前の名前を使う | `TestRetiredRoleConfigKeys`、`TestOrgSpawnAndStart_RetiredLeaderConfigKey_Rejected`、`TestOrgSpawn_RetiredRoleOrID_RefusedBeforeResolvingTheRoleModel`(`internal/cli/org_test.go:3350`) |
| `[org.role_models]` に director だけを書く | `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`(main と写しの両方が director だけを書く)。ほかの役割が既定で埋まることは、役割によらない道として `TestLoad_RoleModelsKeepWhatTheFilesSet` の「one role」が見る |
| `driver_pool = ["codex"]` で claude のエントリを明示する(D9 で「読み込みは通り、立てるときに断る」に変わった) | 逆向き(`driver_pool = ["claude"]` と codex のエントリ)だけがテストされていた。足したテスト `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused` がこの向きを spawn と director start で打つ |
| `--model ""` を明示する | `TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel` の「an empty --model is an omitted one」、`TestResolveRoleModel` の「blank flags count as omitted」 |
| `--driver` と役割の driver が同じで `--model` なし | `TestResolveRoleModel` の「only --driver, the entry's driver」、CLI の「--driver only, the role's driver」 |
| エントリのない独自の役割で `--driver` と `--model` を渡す | `TestResolveRoleModel` の「both flags for a role with no entry」、CLI の「a custom role with both flags」 |

## 足したテスト

どちらも `internal/cli` の新しいファイルで、未追跡のまま置いた。コミットするかは orchestrator に任せる。

| ファイル | テスト | 見るもの |
| --- | --- | --- |
| `internal/cli/org_legacy_model_keys_cleanup_test.go` | `TestOrgCleanupVerbs_LegacyModelKeys_StillWork`(subtest 3 つ: `[org.roles]`、`[org].model_pool`、両方) | 掃除の前の設定で座席を 2 つ立て、残ったキーを書いた `--config` で `status`・`stop --seat`・`disband` を本番の道(stub の herdr、`--dry-run` なし)で打つ。rc 0、status に 2 座席、seat-1 の最後の event が本物の `stopped`、herdr に `pane close <pane>`、`disbanded org "org-a"` の行、本物の `disbanded` が org-a の 1 つ、seat-2 も `stopped` になること |
| `internal/cli/org_role_models_codex_pool_test.go` | `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused`(subtest 2 つ: spawn、director start) | `driver_pool = ["codex"]` と claude のエントリで、フラグなしの spawn と director start が、役割・エントリ・`driver_pool` を名指す `ResolveRoleModel` の文で断られ、manifest・receipt・herdr・agmsg・`director.jsonl`・`prompts/` に何も書かれないこと |

変異で、2 本が何を見分けるかを確かめた。どの変異も production のファイルの写しを `go test -overlay` で差し替え、worktree のファイルは変えていない。

| 変異 | 足したテストの結果 | 既存のテストだけの結果 |
| --- | --- | --- |
| m1: `config.Load` が残ったキーを書いた `ralph.toml` を断る | cleanup の 3 つとも FAIL | — |
| m1r: m1 を `[org.roles]` だけに絞る | `[org.roles]` と両方が FAIL、`model_pool` は PASS | `-run 'Legacy\|Roles\|RoleModels\|Cleanup\|Status\|Disband\|Stop'` で doctor と spawn の 4 本が FAIL |
| m2: `newOrgRuntimeAt`(spawn も通る)で `LegacyRolesErr` を返す | `[org.roles]` と両方が FAIL | 同じ絞り方で `TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml` が FAIL |
| m3: `newOrgRuntime`(status・stop・disband が通り、spawn は通らない)で `LegacyRolesErr` を返す | `[org.roles]` と両方が FAIL | 同じ絞り方でも、`[org.roles]` を書くほかのテスト(director start、doctor)を足した回でも生き残った |
| m4: `ResolveRoleModel` の `driver_pool` の確かめを codex のときだけにする | spawn と director start の両方が FAIL | `internal/org` の `-run 'RoleModel\|Resolve\|Director\|Legacy'` と `internal/cli` の `-run 'RoleModels\|DriverPool\|Director.*Start\|Doctor.*RoleModel'` で生き残った |

m3 が生き残った回の範囲について。`internal/cli` のテストのうち `[org.roles]` を書く設定を使うのは、`git grep` で見たかぎり spawn(`org_test.go:828`、`org_role_models_test.go:414` と `:420`)、director start(`org_director_headless_test.go:257`、`:453`、`:475`)、doctor(`doctor_role_models_test.go`)、`ralph config get`(`config_test.go:180`)だけで、どれも `newOrgRuntime` を通らない。それで、パッケージ全体でも m3 は既存のテストでは死なないと判断した。パッケージ全体を m3 で打ってはいない。

m4 では、director start が `claude` の director を `driver_pool = ["codex"]` のまま起動し(`started d1 (driver=claude, model=(CLI default), …)`)、spawn は `ValidateSpawnEnvelope` の古い文で断ったうえ、manifest に `rejected` と receipt を書いた。director start の道で `driver_pool` を確かめるのは `ResolveRoleModel` だけなので、この確かめが弱まっても既存のテストは気づかなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--driver` と `--model` を明示する今までの spawn・start・director start のテストが同じ結果になる | OK | run-test.sh の `internal/cli` と `internal/org` が ok。たとえば `TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`、`TestOrgCleanupVerbs_RetiredLeaderConfigKey_StillWork` は明示のまま通った |
| 印のない古い `spawn_started` では stop が観測しない | OK | `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved`、`TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation` |
| model_pool の復旧のテストの扱い | OK(self-review が記録) | `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` は `TestOrgStatus_ConfigFlagRecoversFromARalphTomlThatFailsToLoad`(`internal/cli/org_test.go:940`)に置き換わった。消えた 34 本の行き先は self-review の report の「削除されたテストの扱い」にある。この report では 1 本ずつ評価し直してはいない |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は前の回と同じ 5,195 |

## Test gaps

1. herdr を使った実機の起動はしていない。`--model` なしの座席と headless の director が本物の herdr の pane で、claude と codex の既定のモデルで動くこと、alias のモデルが効くか、本物の codex の session で receipt が honored `unknown` と観測したモデルになることは、stub と一時の session の記録でしか見ていない。verify と self-review も同じ範囲にとどまっている。
2. spec の受け入れ条件 2 は implementer の spawn を名指すが、`driver_pool = ["codex"]` の CLI のテストは reviewer の spawn と leader の start で、implementer は config の層(`TestLoad_RoleModelsFollowTheDriverPoolInEffect`)だけ。解決は役割によらない map の引きなので、残しても危険は小さい。
3. 独自の役割にエントリを書き、フラグなしで spawn するテストはない(config の読み込みと doctor にはある)。
4. leader の雛形の手順 2 の新しい文を固定するテストはない(計画の Progress が記録済み)。
5. 打たれていない変更行が 5 つ残る(Coverage の節)。どれも `Load` の道からは届かない防御の分岐で、テストを足す意味は薄い。
6. `ralph doctor` は `./ralph.toml` だけを読む(verify の I1)。feature worktree で打った doctor が写しの値を表すことを固定するテストも、main の値を読むテストもない。前からある形。
7. 版が混ざったとき、古い ralph が空のモデルの director の登録を読む場合(計画の R3)は、この repo のテストでは打てない。release ノートで扱う。
8. このサイクルでは `-race` を打っていない。計画の Test plan は求めていない。
9. 足したテスト 2 本はコミットされていない。コミットしないなら、AC8 の status・stop・disband の半分と、codex だけのプールで claude のエントリを断る向きは、Go のテストで固定されないまま残る(m3、m4 が既存のテストで生き残る)。

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ)、HEAD の写しの coverage の回(5 パッケージ ok)、足したテスト 2 本の 3 回。計画の Test plan の項目は、上の対応表のとおりテストがあり、run-test.sh の中で通った。AC13 のテストの半分は満たした
- Fail: 0
- Blocked: なし。herdr の実機の起動は gap 1 に記録した。足したテスト 2 本をコミットするかは orchestrator が決める
