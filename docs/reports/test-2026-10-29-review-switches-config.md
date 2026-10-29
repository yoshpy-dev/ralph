# Test report: review-switches-config

- Date: 2026-10-29(JST 05:36〜06:00、UTC では 2026-10-28 20:36〜21:00)
- Plan: `docs/plans/active/2026-10-29-review-switches-config.md`(AC1、AC2、AC5、AC6 と Test plan)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1
- Scope: HEAD 53c4ceeb(base da9d1b3a)。behavioral test だけを実行した(静的解析と同期の検査は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1
- Evidence: `docs/evidence/test-2026-10-29-review-switches-config.log`(1: `run-test.sh`、2: 名指しの `-v` 実行、3: mutant の置き換えと落ちた行、4: 古い `config.Load` の確認、5: build した binary の実行、6: coverage)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `go test -overlay` で、scratchpad の写しに置き換えた。テンプレートと `scripts/ralph-config.sh` を読むテストの mutant では、`repoRoot` が環境変数 `T265_ROOT` を読むようにした `defaults_sync_test.go` の写しを overlay し、`git archive HEAD` の写し(またはそれを変えた写し)を読ませた(S1 の implementer と同じ形)

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` | shell 45 本 + Go 9 パッケージ | shell 45 本、Go 9 パッケージ | 0 | 0(shell の既存の SKIP 行が 1 つ、下) | rc 0、1,046 秒(20:36:16〜20:53:42Z) |
| 2 | `go test -count=1 -v ./internal/config/ ./internal/org/driver/` | トップレベル 138(サブテスト 299) | 138(299) | 0 | 0 | rc 0、`ok … 0.657s` と `ok … 1.739s` |
| 3 | `go test -count=1 -v ./internal/cli/ -run 'TestConfig\|TestExitCode\|TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr\|TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse\|TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv'` | 13(9) | 13(9) | 0 | 0 | rc 0、3.3 秒 |
| 4 | `go test -count=1 -v ./internal/org/ -run 'TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver\|TestStartDirector_LaunchesInAWorkspaceOfItsOwn'` | 2(6) | 2(6) | 0 | 0 | rc 0、0.8 秒 |
| 5 | mutant 19 個と対照 2 つ(下の「mutant」) | 21 実行 | 対照 2 つは通り、mutant 18 個が落ちた | 生き残り 1 個(k、lead の名指しの外) | - | 1 回 1〜7 秒 |
| 6 | da9d1b3a の `config.Load` で HEAD のテンプレートとルートの `ralph.toml` を読む(`git archive` の写しに置いた使い捨てのテスト) | 1 | 1 | 0 | 0 | rc 0 |
| 7 | HEAD で build した binary の `ralph config get`(ルート、`ralph.toml` のない dir、正しくない `[org]`) | 10 実行 | 10 | 0 | - | - |

- 1 の scope は `Requested scope: changed` から `Language scope: full fallback (unclassified:ralph.toml)` に落ち、golang の pack が選ばれた(lead の想定どおり)
- 1 の shell: `==> tests/` の 45 本がすべて `OK` で終わった。`PASS` で始まる行は 4,796 行で、0 でない `FAIL:` の要約行はない。SKIP の行は `the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)` の 1 つで、この branch と関係のない既存の行である
- 1 の Go: `cmd/ralph` 1.030 秒、`internal/cli` 323.806 秒、`internal/config` 2.198 秒、`internal/insights` 1.868 秒、`internal/org` 421.429 秒、`internal/org/driver` 3.850 秒、`internal/org/protocol` 3.417 秒、`internal/scaffold` 4.291 秒、`internal/upgrade` 2.313 秒。`(cached)` はない
- 30 分の timeout も、既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`)の失敗も起きなかったので、打ち直しはしていない。1 の間の 1 分の load average は 3.2(始まり)、6.2(05:41 JST)、3.6(終わり)。2〜7 の多くは 1 と並べて走らせた
- 7 の結果: ルートで `standard.plan_advisory`、`standard.cross_review`、`org.review.plan_advisory`、`org.review.cross_review` が `true`、`org.review.max_cross_review_cycles` が `2`、`org.max_seats` が `5`(どれも rc 0)。`org.review` と `nope.key` は rc 2。`ralph.toml` のない dir では `standard.cross_review` が `false` で rc 0。`[standard] cross_review = true` と `[org] max_seats = 0` を書いた file を `--config` で渡すと rc 3 で、`[org].max_seats must be >= 1, got 0` を出す

## lead が名指ししたテスト

2〜4 の `-v` の出力で、次のテストが走って通ったことを名前で確かめた。2〜4 の SKIP は 0 で、`…Exits3/unreadable` も euid が 0 でないので skip されずに走った。

| テスト | パッケージ | 結果 |
| --- | --- | --- |
| `TestDefault_ReviewSwitches` | config | PASS |
| `TestLoad_ReviewSwitches`(5 行: `ralph.toml` なし、表なし、キーのない表、すべて既定と違う値、1 つだけ設定) | config | PASS |
| `TestLoad_OrgReviewMaxCrossReviewCyclesBoundary`(0 と -1 を断る、1 を受ける) | config | PASS |
| `TestLoad_IgnoresUnknownReviewSwitchKeys` | config | PASS |
| `TestDefaultsLockStep`(`org`、`standard`、`shell_and_skills`) | config | PASS(3 つとも) |
| `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(22 行。`org.review.*` の 3 行を含む) | config | PASS |
| `TestStandardTemplateDrift_EachKeyIsComparedOnBothSides`(2 行) | config | PASS |
| `TestConfigGet_*` の 8 本(既定、値あり、`--config`、exit 2、exit 3 の 5 行、`[org]` の回帰、cobra の誤りの exit 1、help の約束) | cli | PASS |
| `TestExitCode_ConfigGet` | cli | PASS |
| `TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver`(claude と codex の worker、claude と codex の leader) | org | PASS |
| `TestStartDirector_LaunchesInAWorkspaceOfItsOwn`(claude、codex。extraEnv がないことの確かめを含む) | org | PASS |
| `TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv`(3 行)、`TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause` | driver | PASS |
| `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(spawn、director start) | cli | PASS |
| `TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse` | cli | PASS |

## AC ごとの確認

| AC | テストと mutant | 結果 |
| --- | --- | --- |
| AC1 | 既定値は `TestDefault_ReviewSwitches`、Load は `TestLoad_ReviewSwitches`、0 以下のエラーは `TestLoad_OrgReviewMaxCrossReviewCyclesBoundary`。既定を 3 にする mutant b1・b2 で前の 2 本が落ちる | 満たす |
| AC2 | `TestDefaultsLockStep` の `org` と `standard`、照合の表の 2 本。テンプレートの写しからキーを消す mutant a3(`max_cross_review_cycles`)と a4(`[standard]` の `plan_advisory`)で lock-step が落ちる。上限と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` がずれる mutant b1 と b3 で `shell_and_skills` が落ちる | 満たす |
| AC5 | org: `TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver`。CLI から見た argv: `TestOrgSpawn_HappyPath_…` の `tab create` が `--env RALPH_PRIMARY_CLI=claude` で終わる。`--env` のない herdr: `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` と driver の 2 本。mutant e、i、f、g、h、h2、h3 がすべて落ちる | 満たす(stub の herdr で確かめた。本物の herdr では走らせていない) |
| AC6 | 4 パッケージ(`internal/config`、`internal/cli`、`internal/org`、`internal/org/driver`)は 1 で通った。`defaults_sync_test.go` の同期の検査は 2 で通った。`check-sync.sh` と `check-skill-sync.sh` は静的な検査なので走らせていない。verify の報告(`docs/reports/verify-2026-10-29-review-switches-config.md`)が d017007c で rc 0 を記録しており、その後の 2 コミット(5f38f0c9、53c4ceeb)が変えたのは verify の報告、計画の記録、insight event の 3 ファイルだけである。1 の中では、2 つのスクリプト自体のテスト(`tests/test-check-sync.sh`、`tests/test-check-skill-sync.sh`)が通った | テストの部分は満たす |

## mutant

lead が名指しした (a)〜(f) は、下の a1・a2、b1、c、d、e、f にあたる。置き換えは perl で 1 回だけ現れる字面を変えた(0 回か 2 回以上なら止める)。build の失敗は 0 件。

| id | 変更 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| a1 | `orgTemplateDrift` から `org.review.max_cross_review_cycles` の `valueDrift` を外す | 落ちる | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides/org.review.max_cross_review_cycles`(`Default()` の変更、テンプレートの値の変更、キーの削除の 3 つとも drift が空になる) |
| a2 | a1 に加えて `orgTemplateProbe` から `MaxCrossReviewCycles` を外す | 落ちる | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(行のキーと probe のキーの照合で止まる) |
| a3 | テンプレートの写しから `max_cross_review_cycles = 2` を消す | 落ちる | `TestDefaultsLockStep/org`(`org.review.max_cross_review_cycles: missing from the template (Default() = 2)`)。実際のテンプレートが `Default()` と合うことを前提にする 2 本も落ちる |
| a4 | テンプレートの写しの `[standard]` から `plan_advisory = false` を消す | 落ちる | `TestDefaultsLockStep/standard`、`TestStandardTemplateDrift_EachKeyIsComparedOnBothSides` |
| b1 | `Default()` の上限を 3 にし、テンプレートの写しも 3 にする(org の照合は通る) | 落ちる | `TestDefaultsLockStep/shell_and_skills`(`RALPH_STANDARD_MAX_PIPELINE_CYCLES default = "2"`、`config.Default() = "3"`)、`TestDefault_ReviewSwitches`、`TestLoad_ReviewSwitches/one_key_set_leaves_the_others_at_default` |
| b2 | `Default()` の上限を 3 にする(テンプレートはそのまま) | 落ちる | b1 の 3 本に加えて `TestDefaultsLockStep/org` と、テンプレートを前提にする 2 本 |
| b3 | `scripts/ralph-config.sh` の写しで既定を `:-3` にする | 落ちる | `TestDefaultsLockStep/shell_and_skills` だけ |
| c | `ExitCode` の 3 の case から `ErrConfigLoad` を外す(1 になる) | 落ちる | `TestConfigGet_RalphTomlThatCannotBeReadOrIsInvalidExits3`(5 行)、`TestConfigGet_InvalidOrgSettingDoesNotTurnCrossReviewOff`、`TestExitCode_ConfigGet` |
| j | `ErrConfigKey` を 1 にする | 落ちる | `TestConfigGet_KeyThatIsNotOneValueExits2`、`TestExitCode_ConfigGet` |
| d | Load のエラーで `config.Default()` を使う | 落ちる | `…Exits3`(5 行)、`TestConfigGet_InvalidOrgSettingDoesNotTurnCrossReviewOff`(AC4 の回帰) |
| d2 | Load のエラーを `ErrConfigLoad` で包まない | 落ちる | d と同じ |
| k | キーの照合を Load の前に `Default()` で行う(壊れた `ralph.toml` と知らないキーの組で 3 でなく 2) | 生き残る | - |
| e | spawn の `TabCreate` から extraEnv を外す | 落ちる | org: `TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver`(4 行)。cli: `TestOrgSpawn_HappyPath_…`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr/spawn` |
| i | spawn が driver によらず `PrimaryCLIEntry("claude")` を渡す | 落ちる | org: `…/codex_worker`、`…/codex_leader`(cli の 2 本は claude の座席だけなので通る) |
| f | director の `TabCreate` に `RALPH_PRIMARY_CLI=<driver>` を渡す | 落ちる | org: `TestStartDirector_LaunchesInAWorkspaceOfItsOwn`(claude、codex)。cli: `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr/director_start` |
| g | `paneEnvConsequence` の `RALPH_PRIMARY_CLI=` の case に `&& false` を足す | 落ちる | `TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`(2 行)、`TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause` |
| h | env を extraEnv、PaneEnv の順にする | 落ちる | `TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv`(2 行)、`…ExtraEnvWithoutEnvFlagWarnsWithItsClause` |
| h2 | `--env` を足す条件を `len(h.PaneEnv) > 0` にする | 落ちる | `TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv/a_literal_with_no_PaneEnv` |
| h3 | 警告に PaneEnv だけを渡す | 落ちる | `TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause` |

- 走らせた範囲: config の mutant は `./internal/config/` 全体。cli の c・j・d・d2・k は `-run 'TestConfig|TestExitCode'`。e・i・f は `internal/org` で 4 と同じ `-run`、`internal/cli` で `-run 'TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse|TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr'`。driver の mutant は `./internal/org/driver/` 全体
- 対照: config は overlay だけ(`T265_ROOT` に変えていない写し)で rc 0、FAIL 0、SKIP 0。cli は `-run 'TestConfig|TestExitCode'` で 15 本が通った。e・i・f の対照は 3 と 4 の実行(同じか、より広い `-run`)
- k が生き残る理由: 壊れた `ralph.toml` に知らないキーを渡すテストがない。`…Exits3` の 5 行はどれも正しいキー `standard.plan_advisory` を使う。`config get` を呼ぶテストは `internal/cli/config_test.go` だけなので(`grep` で確かめた)、パッケージ全体で走らせても結果は変わらない。skill から見ると 2 も 3 も「止まって報告する」で、off として読まれることはない。違いは終了コードと文言だけである

## Test plan の端のケース

| 端のケース | テスト |
| --- | --- |
| `ralph.toml` がない | `TestLoad_ReviewSwitches/no_ralph.toml`、`TestConfigGet_NoRalphTomlPrintsTheDefaults`(7 キー)、7 の binary |
| キーがない(表なし、キーのない表、1 つだけ設定) | `TestLoad_ReviewSwitches` の 3 行、`TestConfigGet_PrintsTheValueRalphTomlSets`(設定していないキーは既定を出す) |
| 読めない `ralph.toml` | `TestConfigGet_RalphTomlThatCannotBeReadOrIsInvalidExits3`(unreadable、ディレクトリ、parse できない、検証で落ちる、`--config` の parse できない file) |
| 知らないキー | 引数: `TestConfigGet_KeyThatIsNotOneValueExits2`(`nope.key`、`standard.nope`、大文字、空、末尾の点)。file の中: `TestLoad_IgnoresUnknownReviewSwitchKeys` |
| 表の名前 | `TestConfigGet_KeyThatIsNotOneValueExits2`(`standard`、`org.review`、`org.roles`、配列 2 つ、表と値の下のキー)。7 の binary で `org.review` が rc 2 |
| `--env` のない herdr | driver: `TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause`、`TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence` の 3 行。CLI: `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(spawn は 3 つの entry と clause、director は 2 つで clause なし) |
| 古い binary が新しいテンプレートを読む | `TestLoad_IgnoresUnknownReviewSwitchKeys`(知らないキーと、知らない子の表 `[org.review.future_table]`)。6 で、da9d1b3a の `config.Load`(`config.go` に `Standard` も `Review` もない)が HEAD のテンプレートとルートの `ralph.toml` を rc 0 で読んだ |
| 上限の境界 | `TestLoad_OrgReviewMaxCrossReviewCyclesBoundary`(0、-1、1) |

## Coverage

- Statement:
  - `internal/config` 97.0%(`Default` 100%、`Load` 95.7%。通らない 4 ブロックは、この diff の外にある既存のエラーの道)
  - `internal/org/driver` 95.3%(`PrimaryCLIEntry`、`paneEnvConsequence`、`warnNoPaneEnv`、`paneEnvSupported` は 100%、`TabCreate` は 84.2%。通らないのは run、envelope、parse のエラーと pane id の欠落で、どれも既存の道)
  - `internal/cli` の新しい関数(`-run 'TestConfig|TestExitCode'` で測った): `newConfigGetCmd`、`configValue`、`configField`、`ExitCode` は 100%、`newConfigCmd` は 83.3%(`ralph config` を引数なしで打ったときの `cmd.Help()`、`config.go:35`)、`configKindName` は 85.7%(今の `Config` にない kind の default、`config.go:145`)
- Branch: 測っていない
- Function: 上の行のとおり
- Notes: `internal/org` の `spawn.go` と `director_start.go` の変わった行は、mutant e、i、f が落ちることで、テストがその行を通ることを確かめた。`internal/org` と `internal/cli` のパッケージ全体の coverage は測っていない(1 回 5〜7 分かかる)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の pane の env(`AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off`)が変わらない | 通る | `TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`、`TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv` の extraEnv なしの呼び出し(PaneEnv だけで、PaneEnv 自体も変わらない)、`TestOrgVerbs_…/director_start`(2 つだけ)、`TestOrgSpawn_HappyPath_…` の末尾 |
| 照合の表と `TestDefaultsLockStep` が新しいキーを見る | 通る | mutant a1〜a4、b1〜b3 |
| headless の director の pane に `RALPH_PRIMARY_CLI` が入らない(R4) | 通る | mutant f |
| 正しくない `[org]` と `standard.cross_review = true` の同居で exit 3(AC4 の回帰) | 通る | `TestConfigGet_InvalidOrgSettingDoesNotTurnCrossReviewOff`、mutant c・d・d2、7 の binary |
| 既存の終了コード(3 = upgrade の drift、4、5) | 通る | `TestExitCode` |

## Test gaps

- mutant k(LOW): 壊れた `ralph.toml` では知らないキーでも 3 になる、という順序(計画の S2 の注記)を押さえるテストがない。`TestConfigGet_RalphTomlThatCannotBeReadOrIsInvalidExits3` に、キーを `nope.key` にした行を 1 つ足せば k が落ちる。skill にとっては 2 も 3 も止まる側なので、off として読まれる危険はない
- `ralph config` を引数なしで打ったとき(help を出して exit 0)のテストがない(LOW)。打ち間違いの `ralph config gte …` が exit 1 になることは `TestConfigGet_CommandLineErrorsExit1` が見ている
- `RALPH_PRIMARY_CLI` は stub の herdr の argv で確かめた。本物の herdr で pane を作り、座席のプロセスや `pick_reviewer` にその値が届くかは走らせていない
- skill が設定を読む側(#266、#267、#269)と、PATH に `ralph` がないときの exit 127(verify の V1)は、この PR の範囲の外
- macOS の go1.27.3 だけで走らせた。CI の Linux では走らせていない

## Verdict

- Pass: 1(rc 0、shell 45 本、Go 9 パッケージ)、2〜4(名指しのテストすべて、SKIP 0)、lead の (a)〜(f) を含む mutant 19 個のうち 18 個、6、7
- Fail: なし
- Blocked: なし
- Verdict: pass
