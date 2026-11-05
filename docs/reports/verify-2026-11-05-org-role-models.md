# Verify report: org-role-models

- Date: 2026-11-05
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(AC1〜AC13、D1〜D9、Progress の S1〜S7)。spec は `docs/specs/2026-11-05-org-role-models.md`(FR-1〜FR-9、受け入れ条件 15 件)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff d631e9f8...HEAD`(base main d631e9f8、HEAD eae6dd39、85 ファイル、+6247 / -2476)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない(`go test` は 1 度も打っていない)。HEAD から `go build` した binary を一時ディレクトリに置き、`ralph doctor`、3 つの動詞の `--help`、一時の台帳での `spawn --dry-run`・`status`・`stop --dry-run`・`disband --dry-run` を打った。herdr と agmsg は PATH になく、見張りは一時の台帳に `--detach` で立てて trap で止めた(終わったあとに残っていないことを `pgrep` で確かめた)。worktree には report と insight event のほかに何も書いていない。
- Evidence: `docs/evidence/verify-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。中身は run-static-verify の全出力と、末尾の Probe 1〜6(doctor の 4 通りの設定、`--help` の flag の行、残った語の `git grep`、`events[len(events)-1]` の数、`spawn --dry-run` の 6 通り、`[org.roles]` を残した設定での status・stop・disband)。run-verify 自身のログは `docs/evidence/verify-2026-11-05-123137.log`(同じく対象外)。
- 計画の digest: `./scripts/plan-visual.sh digest` は `82b7b97155f4` で、`- Approved:` の行と同じ。
- Insight event: `docs/insights/events/2026-11-05-org-role-models.jsonl` に 1 行足した(phase verify、cycle 1 は `cycle-count.json` と同じ、verdict pass、low 2)。

## Spec compliance

計画の AC は spec の FR と受け入れ条件を束ねているので、AC ごとに対応する FR と spec の受け入れ条件(spec AC の番号は spec の並び順)を書いた。テストは中身を読んで、何を assert しているかを確かめた。結果は /test が出す。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 設定と既定(FR-1、FR-2、spec AC 1・2) | Met | 既定の 2 組は `DefaultRoleModels`(`internal/config/role_models.go:83-94`)。`Default()` は claude の組(`internal/config/config.go:759`)。`Load` は表を空の map から読み(`config.go:864`)、2 つのファイルを当てたあとで `fillDefaultRoleModels`(`config.go:900`、`role_models.go:100-109`)が使われる `driver_pool` の組で足りない役割を埋める。テストは `TestDefaultRoleModels_PicksTheSetByDriverPool`、`TestDefault_RoleModelsIsTheClaudeSet`、`TestLoad_RoleModelsFollowTheDriverPoolInEffect`、`TestLoad_RoleModelsKeepWhatTheFilesSet`(`internal/config/role_models_test.go`)。Probe 1 の case a と b で、doctor の envelope の行が claude の組と codex の組をそのとおりに出した。 |
| AC2 ファイルごとの検査(FR-5、spec AC 7) | Met | `checkRoleModelEntries`(`role_models.go:134-159`)は pointer の probe でキーの有無を見て、`ralph.toml`(`config.go:872`)と `ralph.local.toml`(`config.go:891`)のそれぞれで呼ばれる。エラーはファイルの path と役割を名指す。`TestLoad_RoleModelsEntryMustSetBothKeys` がインラインと表見出しの両方を打ち、`TestLoad_RoleModelsKeepWhatTheFilesSet` が `ralph.local.toml` の 1 役割の置き換えを打つ。 |
| AC3 値の検査(FR-5、spec AC 8・12) | Met | driver の値は `role_models.go:151-153`、モデルの文字は `CheckOrgModelName`(`role_models.go:118-123`、`reviewerWordPattern` と先頭の `-`、`""` は通す)。`Load` は driver と `driver_pool` を突き合わせない(D9)。`driver_pool` そのものは `validateLoaded` が空と未知の値を断る(`config.go:940-947`、S5b)。テストは `TestLoad_RoleModelsValues`、`TestCheckOrgModelName`、`TestLoad_RoleModelsDriverOutsideDriverPoolLoads`、`TestLoad_DriverPoolMustNameKnownDrivers`。spawn 側の拒否は AC4 の行。 |
| AC4 解決(FR-3) | Met | `ResolveRoleModel`(`internal/org/role_models.go:52-87`)の switch を spec FR-3 の 4 つの規則と 1 つずつ突き合わせた。両方のフラグ、`--model` だけ(エントリの driver)、エントリと同じ `--driver` だけ、違う `--driver` だけ(エントリのモデルが `""` なら通り、そうでなければ `--model` を名指して断る)、エントリのない役割(役割と `[org.role_models].<role>` を名指して断る)、文字の規則に合わない `--model`、解決した driver が `driver_pool` にないとき(エントリ由来と `--driver` 由来で文が分かれる)。`TestResolveRoleModel` の 23 ケースがこれをすべて打つ。spec AC 3 は、エントリと `--model` の値を入れ替えた形(エントリ gpt-6.1-sol、`--model gpt-6-astra`)で打たれていて、通る道は同じ。 |
| AC5 動詞(FR-3、spec AC 1・4・5・6) | Met | spawn(`internal/cli/org.go:940-944`)、`start <task>`(`:1152-1156`)、`start --plan`(`:1270`)、director start(`internal/cli/org_director_headless.go:333`)が `ResolveRoleModel` を呼んでから動く。spawn と director start の `--driver` は必須でなく、start の `--driver` の既定は `""`(`org.go:957`、`:1172`)。`resolveModelOrWarn` は消えた。テストは `TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel`(9 ケース、stderr が空であることも見る)、`TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`(manifest、receipt、herdr、agmsg が空)、`TestOrgStart_RoleModels_LeaderEntryWithoutDriverOrModel`(`driver_pool = ["codex"]` で leader が codex になる)、`TestOrgDirectorStart_RoleModels_NoFlagsLaunchesTheCLIDefault`、`TestOrgSpawnStartAndDirectorStartHelp_RoleModels`(`--driver` の行に required も既定値もない)。HEAD の binary の spawn では Probe 5 の 6 通りが同じ結果になった(Observational checks)。 |
| AC6 モデルが `""`(FR-1、D5、spec AC 5) | Met | 座席の argv は `if p.Model != ""` のときだけ `--model` を足す(`internal/org/spawn.go:1077-1079`)。`spawn_started` の印は本番(`:1465`)と dry-run(`:2302`)の 2 か所。herdr の director と `launch_cmd` の起動スクリプトは同じ `directorAgentArgs`(`internal/org/director_start.go:953-959`、`internal/org/director_cmd.go:324`)。登録は `AttachHeadless`(`director_start.go:134`)が `CheckOrgModelName` で `""` を通す。receipt は `codexFoundReceipt`(`spawn.go:1992`)が指示なしのとき honored を `unknown` にし観測したモデルを入れ、stop は印のある event だけ観測する(`internal/org/verbs.go:1904`)。テストは `internal/org/spawn_role_models_test.go` の 7 本、`internal/org/director_role_models_test.go` の 6 本、`TestOrgDirectorStart_RoleModels_LaunchCmdWithoutModel`、表示の `internal/cli/org_model_display_test.go` の 3 本。印のない古い event は `TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation` と既存の `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved` が固定する。 |
| AC7 読み先(FR-4、spec AC 9・13) | Met | `applyMainWorktreeOrgLimits`(`org.go:236-243`)が `RoleModels`・`DriverPool`・`LegacyModelKeys` を main worktree の値で丸ごと置き換え、spawn と `start <task>`(`newOrgSpawnRuntimeAt`)と `start --plan`(`resolveOrgStartConfig`)が共有する。director start は `withMainWorktreeDirectorStartConfig`(`org.go:301`)。`TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml` の 6 ケースが、`--config` なしで main の値、`--config` でそのファイルの値、写しの古い `driver_pool = ["claude"]` と main の `ralph.local.toml` の codex の reviewer の組み合わせで codex を打つ。`TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml` が `start --plan` を打つ。 |
| AC8 残ったキー(FR-6、spec AC 10・11) | Met(`status`・`stop`・`disband` が通る半分は Probe 6 とコードで確認。Go のテストはない) | Probe 5: `model_pool` を残した `--config` で spawn は通り、stderr に 1 行だけ出す。`[org.roles]` を残した `--config` では rc 1 で、manifest の行数は変わらない。Probe 6: 同じ `[org.roles]` の設定で `status` は rc 0 で座席を出し、`stop --dry-run` と `disband --dry-run` は rc 0 で `stopped` と `disbanded` を書いた。`config.Load` は記録するだけで止まらない(`legacyModelKeys`、`role_models.go:167-182`、`TestLoad_LegacyModelKeysLoadInAnyShape`)。`[org.roles]` の拒否 `LegacyRolesErr` を呼ぶのは `retiredRoleConfigErr`(`spawn.go:1711`)と director start(`org_director_headless.go:330`)だけで、`retiredRoleConfigErr` は Spawn の本番(冪等の return のあと)、dry-run、`start --plan` の事前検査 `spawnPrecheckErr`(worktree を作る前)の 3 か所から呼ばれる。stop・disband・status の道にはない。テストは `TestOrgSpawn_LegacyOrgRolesRefused`(CLI、本番と dry-run で何も書かない)、`TestOrgSpawn_LegacyRolesTable_*` の 3 本、`TestSpawnPrecheckErr_MatchesSpawn` の「an [org.roles] table」、director start の `org_director_headless_test.go:257` のケース。`model_pool` の 1 行は `TestOrgSpawn_RoleModels_LegacyModelPoolNoticeOnce` と、start・director start の各テストの stderr の比較。`[org.role_models].lead` は `RetiredRoleConfigKeys` が拾う(`prompts_test.go` の 2211 行からのケース)。 |
| AC9 doctor(FR-7、spec AC 8・10・11・14) | Met | `checkCodexModelSlugs`(`internal/cli/doctor_codex_models.go:314`)は codex のエントリでモデルが空でないものだけを重複なしで見る。`checkOrgModelProbes`(`internal/cli/doctor.go:1120`)は組を重複なしで起動し、`ProbeModel`(`internal/org/driver/probe.go:22`)は `""` のとき `--model` を付けない。`codexSeatPossible`(`doctor_codex_writable_root.go:279-281`)は `driver_pool` に codex があるかだけを見る。envelope の行は `EnvelopeSummary` の役割ごとの組。残ったキーは `checkOrgLegacyModelKeys`(`doctor.go:1058`)、`driver_pool` にない driver は `checkOrgRoleModelDrivers`(`doctor.go:1088`)。Probe 1 の case c で残ったキーの warn 2 行、case d で driver の warn と cache にないスラッグの warn が出た。テストは `internal/cli/doctor_role_models_test.go` の 5 本と `TestDoctorCheckOrgEnvelope_ReportsRoleModelsAndMaxSeats`。 |
| AC10 init(FR-8) | Met | `initSettingsToTOML`(`internal/cli/init_settings.go:143`)は `driver_pool` だけを書き、`initTOMLCommentModelPool` とその 3 つのエラーは消えた。`TestInitSettings_EveryAnswerLoadsFromTheTemplateWithTheTableValues` は 3 通りの出力を `config.Load` で読み、`model_pool` と `[org.roles]` の行がないこと、役割の既定が `DefaultRoleModels` と一致することを見る。`TestRunDoctorOpts_InitSettingsForEachCLIPass` は leader の driver を見る。 |
| AC11 ロックステップ(spec AC 15) | Met | `TestDefaultsLockStep` に「role_models」の sub-test(`roleModelsLockStepDrift`)が入り、`templates/base/ralph.toml:176-186` のコメントの 2 組を `DefaultRoleModels` と比べる。`TestRoleModelsTemplateDrift_*` の 2 本が、値の変更・役割の削除・追加・表の生書き・壊れた組でずれを検出することを固定する。 |
| AC12 雛形と文書(FR-9) | Met | leader の雛形から「`--model` は必ず明示する」とプール先頭の文が消え、手順 2・3・5・14 が「前文に指定があるときだけ渡す」になった(`internal/org/prompts/leader.md`)。`/org` skill は「役割ごとのモデル(`[org.role_models]`)」節に置き換わった。残った語の `git grep`(Probe 3)は、履歴(spec の背景と改訂注記、計画)と、残ったキーを扱うコード・テスト・文書だけで、該当しない行は見つからなかった。`check-skill-sync.sh` と `check-sync.sh` は通った。 |
| AC13 全体 | 静的な半分は Met。テストの半分は /test | `./scripts/run-static-verify.sh` は rc 0。`run-test.sh` は打っていない。 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | `templates/base/ralph.toml` が未分類なので full fallback になり、golang pack を選んだ。範囲は `RALPH_VERIFY_SCOPE=full` と同じ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さない(通ったときの形)。`Skipping` の行はない。手元の Go は go1.27.3 で、`go.mod` の go の行と同じ。 |
| shellcheck(hook と verify の script)、hook の `sh -n`、guard の awk、settings の `jq -e` | pass | |
| `scripts/check-sync.sh` | pass | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6(`model-routing.md` は KNOWN_DIFF)。 |
| `check-pipeline-sync.sh` / `check-skill-sync.sh`(14 skill)/ `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | d631e9f8..eae6dd39 は clean。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org spawn --help`・`start --help`・`director start --help` | Yes | Probe 2。3 つとも `--driver` の既定は役割のエントリの driver、`--model` の既定はエントリのモデルで、`""` なら `--model` なしと書く。spawn と start の Long は FR-3 の規則、`[org.roles]` の拒否、`model_pool` の 1 行、読む場所を書く。director start の Long も `[org.roles]` と `model_pool` を書く。 |
| `/org` skill(4 つの写し) | Yes | `spawn`・`start`・`director` の行と「役割ごとのモデル」節がコードと合う。写しは `check-skill-sync.sh` と `check-sync.sh` でそろっている。 |
| `/director` skill(4 つの写し) | Yes(I2) | 前文に書くのは `ralph.toml` と違う座席だけ、という手順になった。`start --plan` の例は 2 つあり、2 つ目に `--driver claude --model opus` が残る(I2)。 |
| `internal/org/prompts/leader.md` | Yes | AC12 の行。 |
| `internal/org/prompts/director.md` | Yes(I3) | モデルや前文の文がなく、直すところがない。計画の Affected areas は名前を挙げていた。 |
| `.claude/rules/ralph/model-routing.md`(2 つの写し) | Yes | 「Org runtime model receipts」節が `[org.role_models]`、`""` の receipt(honored が `unknown`)、doctor のスラッグの検査を書く。「Codex has no model aliases」は、doctor が `[reviewer].codex_model` を見なくなったことを書く(tech-debt の 429 行と合う)。 |
| `README.md` | Yes | `ralph.local.toml` の段落と、役割のモデルの段落(既定の 2 組、`model_pool` の 1 行、`[org.roles]` の拒否)がコードと合う。 |
| `docs/recipes/codex-seat-permissions.md`・`director-launch-cmd.md` と `templates/base/` の写し | Yes | 写しは `cmp` で同じ。`--model` を付けない座席の alias の扱いと、writable root の条件(`driver_pool` に codex)がコードと合う。 |
| `docs/specs/2026-08-01-org-runtime.md` | Yes | 2026-11-05 改訂の節 (a)〜(c) があり、FR-1・FR-2・FR-7 の行、2026-09-16 改訂の (c)(d)(e)、運用ノートに改訂の印がある。 |
| `docs/specs/2026-10-07-org-multi-org-director.md` | Yes | 2026-11-02 の決定の末尾に、init が `model_pool` を扱わなくなったことを足した。 |
| `AGENTS.md` の repo map | Yes(I5) | `internal/config/` の行に `role_models.go` が入った。`internal/org/` の行は `role_models.go` を名指さない(I5)。 |
| `templates/base/ralph.toml` | Yes | `model_pool` の配列と `[org.roles]` のコメント例が消え、`[org.role_models]` の 2 組がコメントで入った。`driver_pool` のコメントが既定の組の選び方を書く。 |
| `docs/tech-debt/README.md` | No(V1、V2) | 閉じた 3 行と足した 3 行、148・156・201・255 行の carry-over の注は合っている。発火した Trigger のうち 5 行に注がない(V1)。152 行の数が違う(V2)。 |
| `ralph doctor` が読む設定(FR-7 の「実際に使われる」) | 注記(I1) | doctor は `./ralph.toml` を読み、main worktree の値を取らない。 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt の Trigger のうち、この branch が発火させたのに carry-over の注がない行が 5 つある。この branch は 148・156・201・255 行には「2026-11-05: docs/spec-org-role-models hit this trigger ... Carried over」の注を足しているので、同じ書き方が抜けている。(1) 135 行: Trigger は「`doctor_codex_writable_root.go` の次の変更」で、S4 が `codexSeatPossible` を入れ outcome の番号を振り直した(102 行の変更)。(2) 188 行: (a) は「leader prompt の次の変更」、(b) は「`EnvelopeSummary` か `checkOrgEnvelope` の次の変更」で、leader.md と `EnvelopeSummary` がどちらも変わった。(3) 190 行 (d) と (4) 223 行 (b): 「`withMainWorktreeOrgLimits` の次の変更」で、148 行に書いた同じ変更が当たる。223 行 (b) の改名の理由は、関数が `RoleModels`・`DriverPool`・`LegacyModelKeys` も運ぶようになって強まった。(5) 414 行: 「5 つのファイルのどれかの次の変更」で、`internal/cli/doctor.go` は 1519 行から 1613 行に増え、`internal/config/config.go` は 1194 行から 1120 行に減った。 | 行番号は HEAD eae6dd39 の `docs/tech-debt/README.md`。行を `\|` で分けて 4 列目(Trigger)を python で読み、2026-11-05 の注があるのは 148・156・201・255 行だけだった。行数は `git show d631e9f8:<file> \| wc -l` と HEAD の `wc -l`。 | /sync-docs で 5 行に carry-over の注を足す(何を変えたか、払わなかった理由、414 行は新しい行数)。払うかは判断に任せる。 |
| V2 | LOW | 152 行(閉じた `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` の行)は「ten other tests in `internal/cli/org_test.go` index `events[len(events)-1]` ... (counted 2026-11-05)」と書くが、len の確かめのない添字は 12 テストに 1 か所ずつある。この行を書いた 0ee2bbaf でも 12 で、そのあとに `org_test.go` を変えたコミットはない。Trigger の列も「one of the ten others」と書く。 | Probe 4: `org_test.go` の 518・551・592・627・806・1198・1589・1691・2524・2571・2654・2675 行。どれも直前に `len(events)` の確かめがない(代入の行から添字の行までを python で調べた)。 | 152 行の 2 か所の「ten」を「twelve」にする(数え方を変えたのなら、その数え方を書く)。 |
| I1 | INFO | `ralph doctor` は `<targetDir>/ralph.toml`(`.`)だけを読む(`doctor.go:110`)。spawn と start は台帳が main worktree のものなら `[org.role_models]` と `driver_pool` を main の `ralph.toml` から読む(FR-4)ので、feature worktree で打った doctor の「Org envelope」「Org role model drivers」「Org codex model slugs」「Org legacy model keys」は写しの値を表す。FR-7 の「実際に使われる `[org.role_models]`」と食い違うのはこの場合だけで、main worktree の root で打てば同じになる。`max_orgs` などで前からある形で、今回の変更が持ち込んだものではない。 | `runDoctorStateDir` の `config.Load(filepath.Join(targetDir, "ralph.toml"))`。doctor の Go に `loadMainWorktree` を呼ぶ所はない。 | 直さないなら、`/org` skill の doctor の段落に「main worktree の root で打つ」と 1 句足すか、tech-debt の 148 行(設定と台帳の読み方の違い)に doctor を足す。 |
| I2 | INFO | FR-9 は「`ralph org start --plan` の例から `--driver` と `--model` を外す」と言うが、`/director` skill の 2 つ目の例は `--driver claude --model opus` を持つ。1 つ目の例はフラグなしで、2 つ目は「`ralph.toml` の leader が codex で、前文で leader を claude に決めたとき」の例として説明が付いている。計画の Progress(S6)は、`internal/org/skill_split_questions_test.go` が文を固定するので残したと書く。 | `.claude/skills/director/SKILL.md:597-602`。 | 記録済みの逸脱で、例の意味は新しい規則に合う。対応は要らない。 |
| I3 | INFO | 計画の Affected areas と S6 は `internal/org/prompts/director.md` を挙げるが、変わっていない。この雛形にはモデル、`--driver`・`--model`、分割計画の前文の文がない(`grep` で 3 行目の `driver: {{DRIVER}}` だけ)ので、直すところがない。 | `internal/org/prompts/director.md`。 | 対応は要らない。 |
| I4 | INFO | 計画の AC1〜AC13 と spec の FR・受け入れ条件のチェックボックスは `[ ]` のまま。Progress の「Review artifact created」も、self-review の report があるのに `[ ]`。 | 計画の 87〜99 行と 164 行。 | /pr の前に orchestrator が付ける。 |
| I5 | INFO | `AGENTS.md` の `internal/org/` の行は `role_models.go`(`ResolveRoleModel`、`LegacyRolesErr`、`LegacyModelPoolNotice`)を名指さない。計画の Verify plan が見るよう求めたのは `internal/config/` の行だけで、そちらは入っている。 | `AGENTS.md` の repo map。 | 任意。足すなら `internal/org/` の行の envelope の説明の近くに 1 句。 |

## Observational checks

- Probe 1(HEAD の binary の `ralph doctor`、一時ディレクトリの 4 通りの設定。HOME、ZDOTDIR、CODEX_HOME を一時の場所にし、手元の `models_cache.json` の写しを置き、`RALPH_LOCAL_CONFIG=off`)。
  - 空の `ralph.toml`: `role_models: director=claude/(CLI default), implementer=claude/opus, leader=claude/opus, reviewer=claude/opus`。残ったキーと driver は pass、スラッグは「codex のエントリがない」の info。
  - `driver_pool = ["codex"]`: 3 役が `codex/gpt-6.1-sol`、director が `codex/(CLI default)`。スラッグの検査は pass で、`gpt-6.1-sol` が手元の cache にあることも確かめられた(spec の Dependencies)。
  - `model_pool` と `[org.roles]` を残した設定: 「Org legacy model keys」が 2 行の warn を出す。`[org.roles]` の行は、3 つの動詞が断ることと、消すか移すかを書く。
  - `driver_pool = ["claude"]` と codex の reviewer(cache にないスラッグ): 「Org role model drivers」の warn と「Org codex model slugs」の warn が出る(spec AC 8・14)。
- Probe 2: 3 つの動詞の `--help` の `--driver` と `--model` の行(Documentation drift の表)。
- Probe 5(計画の Verify plan の「フラグなしの `--dry-run` の spawn の行」)。spawn は `--dry-run` でも見張りを要る(`spawn.go:608` の `checkWatchGate`)ので、一時の台帳に `ralph org watch --all --detach` を立てた。座席がないので herdr は呼ばれない。reviewer を `--driver` と `--model` なしで打った結果は次のとおり。
  - 既定: `driver=claude model=opus`、stderr は空。
  - `driver_pool = ["codex"]`: `driver=codex model=gpt-6.1-sol`。
  - `model_pool` を残した設定: `driver=claude model=opus` のまま、stderr に `org: <path> sets [org].model_pool, which ralph no longer reads: ... remove model_pool to drop this note` の 1 行。
  - `[org.roles]` を残した設定: rc 1、`rejected: org: <path> sets [org.roles], ...`、manifest の行数は変わらない。
  - `--model sonnet` だけ: `driver=claude model=sonnet`。
  - `--driver codex` だけ: rc 1、`org: role "reviewer" runs on claude in [org.role_models]; pass --model for --driver codex`、manifest の行数は変わらない。
- Probe 6: `[org.roles]` を残した設定で、seed した座席の台帳に `status`(rc 0)、`stop --dry-run`(rc 0、`stopped`)、`disband --dry-run`(rc 0、`disbanded`)。

## Coverage gaps

- Go のテストの結果は /test の担当。この report の「Met」は、コードを読み、テストの assert を読んで、規則と合うことを確かめたもの。
- AC8 と spec AC 11 の「`[org.roles]` が残っても `status`・`stop`・`disband` が通る」を打つ Go のテストはない。Probe 6 は `--dry-run` の stop と disband で、herdr を呼ぶ本番の道は打っていない。設定を読むところは両方の道で同じで、`LegacyRolesErr` の呼び出し元は spawn と director start の道だけ。足すなら、Probe 6 と同じ形の CLI のテスト(`[org.roles]` を書いた `--config`、seed した座席、stub の herdr)が一番小さい。
- spec AC 2 は implementer の spawn を名指すが、CLI のテストは `driver_pool = ["codex"]` を reviewer の spawn と leader の start で打ち、implementer は config の層(`TestLoad_RoleModelsFollowTheDriverPoolInEffect`)で打つ。通る道は役割によらず同じ(`ResolveRoleModel` の map の引き)。
- 独自の役割にエントリを書き、フラグなしで立てる場合は、config の読み込み(`qa_bot`、`role_models_test.go:259`)と doctor(`doctor_org_test.go:1172`)にあり、spawn のテストはない。解決は map の引きで、組み込みの役割と同じ道を通る。
- leader の雛形の手順 2 の新しい文を固定するテストはない(計画の Progress が記録済み。手順 3 と 5 は `prompts_test.go:1442-1446` が固定する)。
- herdr が `--` のあとの argv を pane にどう渡すかと、`--model` を渡さない座席で alias のモデルが効くかは、実機で確かめていない(self-review と同じ範囲)。
- 自己レビューの残り(L4、L6、N1、N2)は tech-debt の 430 行に入っていることを確かめた。中身はこの report では評価し直していない。

## Verdict

pass

- Verified: AC1〜AC12 の実装と、それを固定するテストの存在と assert の中身。AC13 の静的な半分(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、secret scan)。HEAD の binary の doctor の 4 通り、3 つの `--help`、`spawn --dry-run` の 6 通り、`[org.roles]` を残した設定の status・stop・disband。計画の digest は承認のときと同じ。
- Partially verified: AC8 の stop と disband は `--dry-run` の道だけを打った(Go のテストはない)。spec AC 2 の implementer は config の層で確認。
- Not verified: Go のテストの結果と `run-test.sh`(/test)。herdr の pane で `--model` なしの座席が動くこと。
- 所見: LOW 2 件(V1、V2)、INFO 5 件(I1〜I5)。どれも merge を止めない。
- /sync-docs に回すもの: V1(tech-debt の 135・188・190・223・414 行の carry-over の注)、V2(152 行の数)。I1 と I5 は任意。
