# Test report: org-role-models

- Date: 2026-11-05(cycle 1)、2026-11-06(cycle 2。run-test.sh は 00:35:51 から 01:01:00 JST)
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(`## Test plan`、AC1〜AC13、Progress の S8)。spec は `docs/specs/2026-11-05-org-role-models.md`
- Tester: tester サブエージェント(`/test`)。cycle 2 は、cross-review cycle 1 の ACTION_REQUIRED を直した b4cfa641(S8)のあとの、pipeline の 2 回目
- Evidence: `docs/evidence/test-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 1 の記録の後ろに、cycle 2 の節 C2-1〜C2-7 を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-05-153551.log`(同じく対象外。名前の時刻は UTC)
- この report は cycle ごとに書き足す。上が cycle 2 で、`## Verdict` は cycle 2 の判定。cycle 1 の記録は下の「Cycle 1」に、書いたときのまま残した(見出しの段だけを下げた)

## Cycle 2(HEAD bc7f1080)

### Scope(cycle 2)

- `git diff d631e9f8...HEAD`(HEAD bc7f1080)。cycle 1 の HEAD fd107026 のあとに Go のファイルを変えたコミットは、cycle 1 で足したテスト 2 本を入れた 19aa4144 と、S8 の b4cfa641 の 2 つだけ。ほかは記録と文書のコミット
- `./scripts/run-test.sh` は changed scope のまま打った。cycle 1 と同じく `templates/base/ralph.toml` を分類できないので full fallback になり(`.harness/state/verify-scope` は `scope=full`、`reason=unclassified:templates/base/ralph.toml`、`languages=golang`)、Go の全パッケージと `tests/test-*.sh` の 45 本が走った
- 見たもの: 依頼にあった S8 の 6 つの動作(下の「S8 の動作とテストの対応」)。計画の Test plan の項目のうち S8 が触らなかったものは、cycle 1 の対応表(下の「Cycle 1」)のまま
- production code は変えていない。gap を埋めるテストを 1 ファイル足した(下の「足したテスト(cycle 2)」。未追跡で、コミットはしていない)。run-test.sh の Go の段が終わるまでは worktree に置かず、`go test -overlay` で打った。run-test.sh は HEAD だけを打っている

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope → full fallback、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(cycle 1 と同じ) | 1,509 秒。load は開始時 6.52 / 4.75 / 4.84、終了時 9.63 / 19.62 / 17.60。1 分の値の最大は 76.30(00:53:52) |
| └ Go: `internal/org` | — | ok | 0 | 0 | 977.6 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 746.3 秒 |
| └ Go: `internal/config`、`internal/org/driver`、`internal/insights`、`cmd/ralph` | — | ok | 0 | 0 | 14.6 秒、3.5 秒、2.0 秒、1.0 秒 |
| └ Go: `internal/gitroot`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok `(cached)` | 0 | 0 | — |
| 上の `(cached)` の 4 つを `-count=1` で打ち直し | — | 4 パッケージ ok | 0 | 0 | 3.0 秒、0.7 秒、2.1 秒、2.4 秒 |
| 足したテスト 1 ファイル(2 本、subtest 10)。overlay で 1 回と `-count=3`、worktree に置いて 1 回 | 2 本 × 5 回 | すべて PASS | 0 | 0 | `-count=3` で 19.5 秒 |
| 変異 30 個(`go test -c -overlay`、絞った `-run`) | 30 | 30 個とも死んだ。既存のテストが 26 個、足したテストだけが 4 個 | — | — | 1 つ 101〜289 秒(3 つずつ並べた) |
| 足したテストだけが殺した 4 つを 1 つにまとめた変異、既存のテストだけで `internal/org` と `internal/cli` の全体 | 2 パッケージ | 2 パッケージとも ok(変異は生き残った) | — | — | org 844.1 秒、cli 619.4 秒 |
| 変更行の coverage の回(HEAD、絞った `-run`、足したテストなし) | — | 2 パッケージ ok | 0 | 0 | org 36.5 秒、cli 122.3 秒 |

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、`run_in_background` で runner(`/tmp/orm-c2/runner.sh`)を起動し、log、1 分ごとの load、開始・終了・rc の meta を書かせた。同じ手番の中で `caffeinate -w` で待ち、meta を読んだ。走っているあいだに変異を 3 つずつ並べて打った。ほかのセッションの `swiftpm-testing-helper` も同じ時間に CPU を使っていて、1 分の load は 00:53 に 76.30 まで上がった。それでも `internal/org` は 977.6 秒で、30 分の時間切れの 54% に収まった。cycle 1(711.5 秒)より 266 秒長い。S8 が `internal/org` に足したテストは数秒で終わるので、差の大半は load によるとみられる。内訳は確かめていない。

Skipped の 2 行は cycle 1 と同じで、どちらもこの diff と関係のない手元の環境による(`tests/test-plan-visual.sh` の browser detection、`tests/test-secret-scan-branch.sh` の git 2.41 より古い場合)。shell の 45 本、PASS の行の数(5,195)、log の行数(5,617)も cycle 1 と同じ。`(cached)` の 4 パッケージは、`go list -deps` で見ると `internal/org`・`internal/config`・`internal/cli`・`internal/insights` のどれも import していない。念のため `-count=1` で打ち直し、4 つとも ok だった。

### Coverage(cycle 2)

- 変更した行(b4cfa641 だけ): `git diff -U0 b4cfa641^ b4cfa641` で足された行(テストを除く `.go`)を含む profile のブロックは 53 statement で、53 とも打たれていた(100%)。profile は HEAD で `-run` を絞った 2 回(`internal/org` の 146 本と、`-coverpkg=./internal/cli/,./internal/org/` を付けた `internal/cli` の 114 本)を合わせたもので、足したテストは入れていない。絞った回の値なので下限にあたる。行が足されたファイルは 10 で、そのうち `internal/cli/doctor.go` と `internal/config/config.go` はコメントだけ
- パッケージ全体の statement coverage は、このサイクルでは測り直していない(cycle 1 は config 96.6%、org 95.0%、driver 96.1%、protocol 100.0%、cli 90.4%)。S8 は `ResolveRoleModel` の検査を `ValidateSpawnEnvelope` と `checkDirectorStart` に移したもので、足した statement は上の 53 の範囲に入る
- Branch: 計測なし(Go の道具は statement だけ)

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、打ち直し、足したテストの各回、coverage の回のどれにも失敗はなかった。flaky として memory にあるテストも落ちなかった。

### S8 の動作とテストの対応

テスト名の位置は HEAD bc7f1080 のもの。既存のテストはどれも run-test.sh の中で通った。

| 動作 | 既存のテスト | 足したテスト(cycle 2) |
| --- | --- | --- |
| 冪等の打ち直しは、`driver_pool` を狭めたあとでも既存の座席を返す(rc 0、`already spawned` の行、manifest の event・receipt・herdr・agmsg の数が変わらない) | spawn: `TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat`(`internal/cli/org_role_models_retry_test.go:97`)の「re-run」の 3 つ(同じ `--driver codex --model`、フラグなし、文字の規則に合わない `--model`)。start `<task>`: `TestOrgStart_RoleModels_LeaderRetryAfterDriverPoolNarrowed_ReturnsTheLeader`(`:152`)の同じ 3 つ。start --plan: `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`(`:210`)の「re-run」の 2 つ(フラグなし、同じフラグ)。org の層: `TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat`(`internal/org/spawn_role_models_retry_test.go:19`。`DriverFromRoleEntry` の印のあり・なし) | — |
| 文字の規則に合わない `--model` で打ち直しても、既存の座席を返す | spawn と start `<task>` は上の 3 つ目。org の層: `TestOrgSpawn_RetryWithAModelNotOneWord_ReturnsTheSeat`(`spawn_role_models_retry_test.go:74`) | start --plan は HEAD のテストになかった。`TestOrgStartPlan_RoleModels_NewLeaderRefusalsAndARetryWithABadModel` の「re-run」の 3 つ(codex を含むプール、狭めたプール、狭めたプールで `--driver codex` も付ける)が埋める |
| 新しい座席は断られ、`rejected` が残る | spawn: `TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat` の「new seat」の 3 つ(`--driver` 由来の文、エントリを名指す文、モデルを名指す文。最後の event が誤りの文を持つ `rejected`、herdr の呼び出しなし)。start `<task>`: 同じ形の「new org」の 3 つと、`TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit`(`internal/cli/org_test.go:2686`)の「--driver codex with --model」。receipt を台帳から読むのは `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused/spawn`(`internal/cli/org_role_models_codex_pool_test.go:27`。`rejected` 1 つと honored=false の receipt 1 つ)。org の層: `TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat` の 4 つの subtest(本番と dry-run、2 つの文)、`TestOrgSpawn_RetryWithAModelNotOneWord_ReturnsTheSeat` の後半、`TestValidateSpawnEnvelope_DriverPoolRefusalWording`(`internal/org/envelope_test.go:197`、5 case) | — |
| start --plan の新しい leader は、何かを作る前に断られ、何も記録されない | `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`(フラグなし、エントリを名指す文と完全一致)。文字の規則に合わない `--model` は org の層の `TestStartFeature_RefusedBeforeAnyRecord` の「a model that is not one word」(`internal/org/feature_test.go:1038`)だけだった | `TestOrgStartPlan_RoleModels_NewLeaderRefusalsAndARetryWithABadModel` の「new leader」の 3 つ。`--driver codex --model` を狭めたプールで打つと `--driver` 由来の文と完全一致、`--model "not a real model"` と `--driver claude --model=-x` はモデルを名指して断られる。どれも feature worktree・branch・`ralph-worktree.sh` の記録・manifest・herdr・agmsg に何も残さない(verify cycle 2 が挙げた case) |
| director start は書く前に断る | org の層: `TestStartDirector_RefusesBeforeWritingOrCallingHerdr`(`internal/org/director_start_test.go:443`)の S8 の 3 case(`:464` エントリを名指す文、`:474` `--driver` 由来の文、`:483` `launch_cmd`)と「a model that is not one word」(`:496`)など。watch gate を尋ねず、herdr を呼ばず、何も書かないことを見る。CLI: `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused/director_start`(エントリを名指す文。`director.jsonl` が変わらず、herdr の呼び出しも `prompts/` もない) | `TestOrgDirectorStart_RoleModels_DriverPoolAndModelRefusalsThroughTheCLI` の 4 つ。`--driver claude --model sonnet` と codex だけのプールで `--driver` 由来の文と完全一致(`[org.role_models]` を名指さない)、`--model` だけでエントリの driver がプールの外ならエントリを名指す文、文字の規則に合わない `--model` を `--driver` あり・なしで。CLI からの `--driver` 由来の文と文字の拒否は、verify cycle 2 では binary の Probe 9 だけで見ていた |
| start --plan は `DriverFromRoleEntry` を leader の spawn まで運ぶ | `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool` がエントリを名指す文を完全一致で見る。印が落ちる向き(下の M10f、M11f、M15f)はこれで死ぬ | 印がいつも立つ向き(M10t、M11t、M15t)は、既存のテストでは `internal/org` と `internal/cli` の全体でも生き残った。足したテストの「new leader: --driver codex --model outside driver_pool」が殺す |
| `ralph insights` が指示したモデルが空の座席を `(CLI default)` と出す | `TestInsightsCmd_ReceiptsNoCommandedModelShowsCLIDefault`(`internal/cli/insights_test.go:395`。text の 2 行、空の `commanded=  ` がないこと、`--json` の `commanded_models` が空のまま) | — |
| `ResolveRoleModel` は決めるだけで、文字と `driver_pool` を見ない | `TestResolveRoleModel`(`internal/org/role_models_test.go:38`、19 case。source の行で数えた)。文字の規則に合わない `--model` と、プールの外の driver をそのまま渡し、印をエントリ由来のときだけ立てることを見る | — |

### 足したテスト(cycle 2)

`internal/cli/org_role_models_s8_gaps_test.go`(新しいファイル、未追跡)。コミットするかは orchestrator に任せる。go test で compile して通したが、gofmt・vet・lint は /verify の担当なので打っていない。

| テスト | 見るもの |
| --- | --- |
| `TestOrgStartPlan_RoleModels_NewLeaderRefusalsAndARetryWithABadModel`(subtest 6 つ) | leader のエントリを codex にした設定で、新しい leader の start --plan を 3 通り断らせる(`--driver codex --model` を `driver_pool = ["claude"]` で、`--model "not a real model"` と `--driver claude --model=-x` を codex を含むプールで)。誤りの文、`[org.role_models]` を名指さないこと、feature の始まりが何も残らないこと(`assertNoFeatureStart`)、manifest が空、herdr と agmsg の呼び出しなしを見る。そのあと leader を立て、文字の規則に合わない `--model` の打ち直し 3 通りが既存の leader を返し、manifest・receipt・herdr・agmsg の数を変えないことを見る |
| `TestOrgDirectorStart_RoleModels_DriverPoolAndModelRefusalsThroughTheCLI`(subtest 4 つ) | director start を 4 通り断らせる(`--driver claude --model sonnet` と codex だけのプール、`--model sonnet` だけとエントリの claude がプールの外、`--model "not a real model"` だけ、`--driver claude --model=-x`)。誤りの文、`[org.role_models]` を名指すのがエントリ由来のときだけであること、stdout が空、`director.jsonl` が変わらない、herdr の呼び出しなし、`prompts/` なしを見る |

### 変異(cycle 2)

どの変異も production のファイルの写しを `go test -c -overlay` で差し替え、worktree のファイルは変えていない(perl で 1 か所だけ置き換え、1 か所でなければ止める)。overlay には足したテストも入れ、落ちたテストの名前で既存と足したテストを分けた。絞った `-run` は、org が `RoleModel|Resolve|ValidateSpawn|Retry|StartDirector|StartFeature|SpawnPrecheck|DirectorAttachHeadless`(146 本)、cli が `RoleModels|Retry|DirectorStart|Insights|ClaudeEntryOutside|OrgStart_|OrgStartPlan`(114 本)。HEAD では両方とも通る。`f` と `t` の付く変異は、印を `false` か `true` に固定したもの。

| 変異 | 変えたところ | 殺した既存のテスト | 足したテスト |
| --- | --- | --- | --- |
| M1 | `Spawn` の本番の道で、冪等の return の前に `ValidateSpawnEnvelope` を打つ(cycle 1 の不具合を戻す) | org と cli の打ち直しのテスト 5 本、`TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused`、`TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit`、`TestOrgSpawn_InFlightSeat_RefusedRetryWritesNothing`、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn` | 落ちた |
| M2 | `spawnPrecheckErr` で、冪等の判断の前に `ValidateSpawnEnvelope` を打つ | `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn`、`TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir` | 落ちた |
| M20 | `ResolveRoleModel` に `--model` の文字の検査を戻す | `TestResolveRoleModel`、cli の spawn と start の打ち直し | 両方落ちた |
| M21 | `ResolveRoleModel` のフラグなしの case に `driver_pool` の検査を戻す | `TestResolveRoleModel`、打ち直しのテスト 3 本、codex のプールのテスト | — |
| M3 | `driverPoolErr` の `if fromEntry` を反転 | 打ち直しのテスト、文の表、director、codex のプールなど 8 本 | 両方落ちた |
| M4 | `ValidateSpawnEnvelope` が印を捨てる | `TestValidateSpawnEnvelope_DriverPoolRefusalWording`、打ち直しのテスト 4 本、codex のプールのテスト | — |
| M5、M5t | `checkDirectorStart` が印を `false`、`true` に固定 | `TestStartDirector_RefusesBeforeWritingOrCallingHerdr`(両方)、M5 は codex のプールのテストも | director の 1 本が、どちらの変異でも落ちた |
| M6 | `checkDirectorStart` の `driver_pool` の検査を外す | `TestStartDirector_RefusesBeforeWritingOrCallingHerdr`、codex のプールのテスト | 落ちた |
| M7 | `checkDirectorStart` のモデルの検査を外す | `TestStartDirector_RefusesBeforeWritingOrCallingHerdr` だけ(watch gate を 1 回尋ねてから断る。`launch_cmd` では claude を探しに行く) | 通った。CLI では `AttachHeadless` の同じ検査が書く前に同じ文で断り、TestMain の watch gate はいつも通すので、違いが見えない |
| M8f、M8t | spawn の CLI の呼び出しの印 | `TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat`(両方)、M8f は codex のプールのテストも | — |
| M9f、M9t | start `<task>` の CLI の呼び出しの印 | `TestOrgStart_RoleModels_LeaderRetryAfterDriverPoolNarrowed_ReturnsTheLeader`(両方)、M9t は `TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit` も | — |
| M10f、M10t | start --plan の CLI の呼び出し(`runOrgStartPlan`)の印 | M10f は `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`。M10t は既存のテストでは死なない | M10t を殺した |
| M11f、M11t | `startFeatureLeaderParams` の印 | M11f は同上。M11t は死なない | M11t を殺した |
| M12f、M12t | director start の CLI の呼び出しの印 | M12f は codex のプールのテスト。M12t は死なない | 両方殺した |
| M13f、M13t | `Spawn` の dry-run の `SpawnRequest` の印 | `TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat` の dry-run の subtest(org の層だけ。cli の絞った回は通った) | — |
| M14f、M14t | `Spawn` の本番の `SpawnRequest` の印 | org と cli の打ち直しのテスト。M14f は codex のプール、M14t は `TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit` も | — |
| M15f、M15t | `spawnPrecheckErr` の `SpawnRequest` の印 | M15f は `TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`。M15t は死なない | M15t を殺した |
| M16 | `ResolveRoleModel` の `--model` だけの case で印を落とす | `TestResolveRoleModel` | 落ちた |
| M17 | フラグなしの case で印を落とす | `TestResolveRoleModel`、`TestResolveRoleModel_DefaultsOfConfigDefault`、打ち直しのテスト 3 本、codex のプールのテスト | — |
| M18 | エントリと同じ `--driver` だけの case に印を立てる | `TestResolveRoleModel` だけ | — |
| M19 | `ralph insights` の `org.ModelOrCLIDefault` を外す | `TestInsightsCmd_ReceiptsNoCommandedModelShowsCLIDefault` | — |

足したテストだけが殺した M10t、M11t、M12t、M15t は、絞った `-run` の外のテストが殺すかもしれない。そこで 4 つを 1 つの overlay にまとめ、足したテストを入れずに `internal/org` と `internal/cli` の全体を `-count=1` で打った。両方とも ok で、4 つは既存のテストのどこでも死なないことがわかった。4 つとも印を `true` に固定する変異で、まとめても互いの効き目を消さない。この 4 つのもとでは、`--driver` と `--model` を両方渡した start --plan と director start の拒否が、渡していないエントリを名指す文になる(たとえばエントリが claude でも「runs on driver "codex" in [org.role_models].leader」と出る)。断ること自体は変わらないので、害は文の誤りにとどまる。

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 1 の所見: 既に立っている座席を `driver_pool` を狭めたあと(または文字の規則に合わない `--model`)で打ち直すと断られる | OK | 上の対応表の 1 行目と 2 行目のテストが run-test.sh の中で通った。不具合を戻す M1、M2、M20、M21 はどれも既存のテストで死んだ |
| 座席の途中の spawn(`spawn_started` だけ)の打ち直しは、何も書かずに断る | OK | M1 が `TestOrgSpawn_InFlightSeat_RefusedRetryWritesNothing` と `TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn` で死んだ。S8 の前からの形 |
| cycle 1 で足したテスト 2 本(19aa4144 でコミット) | OK | `TestOrgCleanupVerbs_LegacyModelKeys_StillWork` と `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused` は run-test.sh の `internal/cli` の中で通った。後者の spawn の subtest は S8 で「何も記録しない」から「`rejected` 1 つと honored=false の receipt 1 つ」に期待が変わった |
| S8 が期待を変えた既存のテスト | OK(意図の変更で、弱めてはいない) | `TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`(`internal/cli/org_role_models_test.go:132`)から 2 case(プールの外のエントリの driver、空白を含む `--model`)が抜けた。どちらも S8 から `Spawn` の拒否で、`rejected` を残すので、期待を変えて `refusalCases` に移った。`TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit` は「--driver codex with --model」で `rejected` 1 つを求めるようになった。新しい座席の拒否が base と同じ形(`rejected` と receipt)に戻ることは計画の D4・D9 と verify cycle 2 の「新しい座席の拒否の記録」の行が書いていて、assert はどれも数と中身を名指す形に強まった |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は cycle 1 と同じ 5,195 |

### Test gaps(cycle 2)

1. 足したテストはコミットされていない。コミットしないなら、start --plan と director start で `--driver` 由来の拒否の文を固定するテストがなく、M10t、M11t、M12t、M15t は既存のテストの全体で生き残る。start --plan の文字の規則に合わない `--model`(新しい leader の拒否と、既存の leader の打ち直し)も、CLI のテストがないままになる
2. self-review N3(`[org.role_models]` を最初の spawn のあとに書き換えたとき、`ResolveRoleModel` の 2 つの拒否が打ち直しの前に残る)を打つテストはない。tech-debt の 430 行の (e) にある
3. M7 は CLI のテストでは見分けられない。TestMain の watch gate がいつも通すので、`checkDirectorStart` と `AttachHeadless` のどちらが断っても CLI の結果は同じになる。org の層のテストが固定しているので、足していない
4. M13f と M13t(`Spawn` の dry-run の文)は org の層だけが見る。`--dry-run` の spawn をプールの外の driver で打つ CLI のテストはない。dry-run と本番は同じ `driverPoolErr` を呼ぶので、残しても危険は小さい
5. パッケージ全体の statement coverage は測り直していない(Coverage の節)
6. herdr を使った実機の起動はしていない(cycle 1 の gap 1 と同じ)。既存の座席の打ち直しも、stub の herdr と台帳の `spawned` の event に対してだけ打った
7. このサイクルでも `-race` は打っていない。計画の Test plan は求めていない
8. cycle 1 の gap 2・3・4・6・7(spec AC 2 の implementer、独自の役割のフラグなしの spawn、leader の雛形の手順 2、doctor が `./ralph.toml` だけを読むこと、版の混在)は変わっていない。cycle 1 の gap 9(足したテスト 2 本が未コミット)は 19aa4144 で解消した

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ。`(cached)` の 4 パッケージも `-count=1` で ok)。依頼にあった S8 の 6 つの動作には、どれも run-test.sh の中で通ったテストがある(上の対応表)。b4cfa641 の変更行は絞った回で 53 statement すべてが打たれた。変異 30 個のうち 26 個は既存のテストが殺し、残る 4 個(印を `true` に固定する M10t、M11t、M12t、M15t)は既存のテストの全体で生き残り、足したテストが殺した。足したテスト 2 本(subtest 10)は 5 回とも通った
- Fail: 0
- Blocked: なし。足したテスト `internal/cli/org_role_models_s8_gaps_test.go` をコミットするかは orchestrator が決める(gap 1)。herdr の実機の起動は gap 6 に記録した

## Cycle 1(HEAD fd107026)

- Date: 2026-11-05(run-test.sh は 21:49:07 から 22:14:01 JST)
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(`## Test plan`、AC1〜AC13)。spec は `docs/specs/2026-11-05-org-role-models.md`
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff d631e9f8...HEAD`(HEAD fd107026)。`./scripts/run-test.sh` は changed scope のまま打った。`templates/base/ralph.toml` を分類できないので full fallback になり(`.harness/state/verify-scope` は `scope=full`、`reason=unclassified:templates/base/ralph.toml`、`languages=golang`)、Go の全パッケージと `tests/test-*.sh` の 45 本が走った。production code は変えていない。gap を埋めるテストを 2 本、新しいファイルに書いた(下の「足したテスト」。どちらも未追跡で、コミットはしていない)
- Evidence: `docs/evidence/test-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、足したテストの 3 回の結果、5 つの変異の定義と結果、coverage の数字と変更した行の coverage を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-05-124908.log`(同じく対象外。名前の時刻は UTC)

### Test execution(cycle 1)

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

### Coverage(cycle 1)

- Statement(HEAD の写し、足したテストなし): `internal/config` 96.6%、`internal/org` 95.0%、`internal/org/driver` 96.1%、`internal/org/protocol` 100.0%、`internal/cli` 90.4%
- 変更した行: `git diff -U0 d631e9f8...HEAD` で足された行(テストを除く `.go`)を含む profile のブロックは 320 statement で、そのうち 314 が打たれていた(98.1%)。行が足されたテスト以外の `.go` は 23 ファイルで、そのうち 21 ファイルの足された行が statement のブロックに入る(残りの `internal/cli/org_director.go` と `internal/org/manifest.go` はコメントと構造体の項目だけ)。打たれていない 6 つは次のとおり。
  - `internal/cli/org_director_headless.go:335-336`: director start で `org.ResolveRoleModel` が断ったときの return。届く道で、足したテスト `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused/director_start` が打つ(そのテストだけの profile で count 1 を確かめた)。
  - `internal/cli/doctor.go:1074`: 残ったキーの warn の `default` の分岐。キーは `[org].model_pool` と `[org.roles]` の 2 つだけなので、今は届かない。
  - `internal/config/config.go:896-897`、`internal/config/role_models.go:144-145`、`:170-171`: `ralph.local.toml` か各ファイルを probe 用に読み直したときの `toml.Unmarshal` の失敗。同じ data をその前に `toml.Unmarshal(…, &cfg)` で読んでいて、壊れた TOML はそこで先に失敗するので、届かない。
  - `internal/config/role_models.go:102-103`: `fillDefaultRoleModels` の `RoleModels == nil` の初期化。`Load` は表を空の map から読む(`config.go:864`)ので、`Load` からは届かない。
- Branch: 計測なし(Go の道具は statement だけ)
- Function: パッケージの数字のみ
- Notes: tester の memory にある前の値は、config 97.0%(2026-10-29)、org 95.0%(2026-10-23)、driver 95.3%(2026-10-29)、cli 88.4%(2026-10-22)。config の 0.4 ポイント減は、`internal/config` に入った届かない分岐 4 つ(上の `config.go` と `role_models.go` の分)によるものとみられる。消えたコードで分母も変わったので、内訳は確かめていない。

### Failure analysis(cycle 1)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、coverage の回、足したテストの 3 回のどれにも失敗はなかった。flaky として memory にあるテストも落ちなかった。

### Test plan とテストの対応(cycle 1)

テスト名の位置は HEAD fd107026 のもの。どれも run-test.sh の中で通った。

#### Unit: `internal/config`

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

#### Unit: `internal/org`

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

#### Integration: `internal/cli`

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

#### Edge cases

| 計画の項目 | テスト |
| --- | --- |
| 役割の名前に改名前の名前を使う | `TestRetiredRoleConfigKeys`、`TestOrgSpawnAndStart_RetiredLeaderConfigKey_Rejected`、`TestOrgSpawn_RetiredRoleOrID_RefusedBeforeResolvingTheRoleModel`(`internal/cli/org_test.go:3350`) |
| `[org.role_models]` に director だけを書く | `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`(main と写しの両方が director だけを書く)。ほかの役割が既定で埋まることは、役割によらない道として `TestLoad_RoleModelsKeepWhatTheFilesSet` の「one role」が見る |
| `driver_pool = ["codex"]` で claude のエントリを明示する(D9 で「読み込みは通り、立てるときに断る」に変わった) | 逆向き(`driver_pool = ["claude"]` と codex のエントリ)だけがテストされていた。足したテスト `TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused` がこの向きを spawn と director start で打つ |
| `--model ""` を明示する | `TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel` の「an empty --model is an omitted one」、`TestResolveRoleModel` の「blank flags count as omitted」 |
| `--driver` と役割の driver が同じで `--model` なし | `TestResolveRoleModel` の「only --driver, the entry's driver」、CLI の「--driver only, the role's driver」 |
| エントリのない独自の役割で `--driver` と `--model` を渡す | `TestResolveRoleModel` の「both flags for a role with no entry」、CLI の「a custom role with both flags」 |

### 足したテスト(cycle 1)

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

### Regression checks(cycle 1)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--driver` と `--model` を明示する今までの spawn・start・director start のテストが同じ結果になる | OK | run-test.sh の `internal/cli` と `internal/org` が ok。たとえば `TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`、`TestOrgCleanupVerbs_RetiredLeaderConfigKey_StillWork` は明示のまま通った |
| 印のない古い `spawn_started` では stop が観測しない | OK | `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved`、`TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation` |
| model_pool の復旧のテストの扱い | OK(self-review が記録) | `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` は `TestOrgStatus_ConfigFlagRecoversFromARalphTomlThatFailsToLoad`(`internal/cli/org_test.go:940`)に置き換わった。消えた 34 本の行き先は self-review の report の「削除されたテストの扱い」にある。この report では 1 本ずつ評価し直してはいない |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は前の回と同じ 5,195 |

### Test gaps(cycle 1)

1. herdr を使った実機の起動はしていない。`--model` なしの座席と headless の director が本物の herdr の pane で、claude と codex の既定のモデルで動くこと、alias のモデルが効くか、本物の codex の session で receipt が honored `unknown` と観測したモデルになることは、stub と一時の session の記録でしか見ていない。verify と self-review も同じ範囲にとどまっている。
2. spec の受け入れ条件 2 は implementer の spawn を名指すが、`driver_pool = ["codex"]` の CLI のテストは reviewer の spawn と leader の start で、implementer は config の層(`TestLoad_RoleModelsFollowTheDriverPoolInEffect`)だけ。解決は役割によらない map の引きなので、残しても危険は小さい。
3. 独自の役割にエントリを書き、フラグなしで spawn するテストはない(config の読み込みと doctor にはある)。
4. leader の雛形の手順 2 の新しい文を固定するテストはない(計画の Progress が記録済み)。
5. 打たれていない変更行が 5 つ残る(Coverage の節)。どれも `Load` の道からは届かない防御の分岐で、テストを足す意味は薄い。
6. `ralph doctor` は `./ralph.toml` だけを読む(verify の I1)。feature worktree で打った doctor が写しの値を表すことを固定するテストも、main の値を読むテストもない。前からある形。
7. 版が混ざったとき、古い ralph が空のモデルの director の登録を読む場合(計画の R3)は、この repo のテストでは打てない。release ノートで扱う。
8. このサイクルでは `-race` を打っていない。計画の Test plan は求めていない。
9. 足したテスト 2 本はコミットされていない。コミットしないなら、AC8 の status・stop・disband の半分と、codex だけのプールで claude のエントリを断る向きは、Go のテストで固定されないまま残る(m3、m4 が既存のテストで生き残る)。

### Verdict(cycle 1)

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ)、HEAD の写しの coverage の回(5 パッケージ ok)、足したテスト 2 本の 3 回。計画の Test plan の項目は、上の対応表のとおりテストがあり、run-test.sh の中で通った。AC13 のテストの半分は満たした
- Fail: 0
- Blocked: なし。herdr の実機の起動は gap 1 に記録した。足したテスト 2 本をコミットするかは orchestrator が決める
