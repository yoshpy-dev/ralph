# Walkthrough: org-role-models(役割ごとのモデルを `[org.role_models]` で決め、`model_pool` を廃止する)

- PR: https://github.com/yoshpy-dev/ralph/pull/302
- Plan: `docs/plans/archive/2026-11-05-org-role-models.md`
- Spec: `docs/specs/2026-11-05-org-role-models.md`
- Base: main `d631e9f8`。97 ファイル、+8,124 / −2,515(docs を除くと約 84 ファイル)

読む順は、設定(1)、解決と起動(2)、CLI(3)、冪等の打ち直し(4)、doctor と init(5)、残ったキー(6)、文書(7)です。コミットはスライスごとに 1 つで、括弧の中に書きます。

## 1. 設定(S1 `ffd614f8`、S5 `14c77c43`、S5b `b23e6af5`)

- `internal/config/role_models.go`: `OrgRoleModel{Driver, Model}`、`DefaultRoleModels(driverPool)`(claude の組と codex の組)、ファイルごとのエントリの検査(`checkRoleModelEntries`)、残ったキーの記録(`legacyModelKeys`、`OrgConfig.LegacyModelKeys`、`toml:"-"`)。
- `config.Load` は、2 つのファイルを読む前に `RoleModels` を空にする。読んだあと、どちらにも書かれていない役割を、実際に使われる `driver_pool` で選んだ組で埋める。go-toml/v2 は既存の map に役割ごとにマージし、インライン表はエントリを丸ごと置き換え、表見出しはキーごとにマージする。そのため、どのファイルのエントリにも `driver` と `model` の両方を要する。
- エントリの `driver` と `driver_pool` の突き合わせは `Load` では行わない(計画 D9)。S5 で `ModelPool`・`Roles`・`orgPoolKeysPresent`・`filterModelPoolByDrivers` を消した。プールの絞り込みを通して間接的に効いていた `driver_pool` の検査(空、未知の driver)は、S5b で `validateLoaded` に直接置いた。
- テンプレートは、既定の 2 組をコメントの 2 つのブロックで示す。`defaults_sync_role_models_test.go` が、そのブロックと `DefaultRoleModels` を突き合わせる。

## 2. 解決と起動(S2 `bf1f4727`、S8 `b4cfa641`)

- `internal/org/role_models.go` の `ResolveRoleModel` は、spec FR-3 の規則で `ResolvedRoleModel{Driver, Model, DriverFromRoleEntry}` を返す。`LegacyRolesErr` と `LegacyModelPoolNotice` も同じファイルにある。
- モデルが `""` のときは、座席の argv、director の herdr の argv、`launch_cmd` の起動スクリプトのどれにも `--model` を付けない(`directorAgentArgs`)。`spawn_started` には `model_cli_default` を付ける(`internal/org/manifest.go`)。`observeStopModelReceipt` は、この印がある event では観測を続け、honored を `unknown` にして観測したモデルを記録する。
- `RetiredRoleConfigKeys` は `[org.role_models].lead` を見る。`EnvelopeSummary` は役割ごとの driver とモデルを出す。

## 3. CLI(S3 `1bc70f0b`、S7 `e401d1a6`)

- `internal/cli/org.go`: spawn・start `<task>`・start `--plan` が `ResolveRoleModel` を呼ぶ。`--driver` は省略できる。`resolveModelOrWarn` とその警告は消した。`applyMainWorktreeOrgLimits` と `withMainWorktreeDirectorStartConfig` は、`RoleModels`・`DriverPool`・`LegacyModelKeys` を main worktree の設定から取る。
- `org_director_headless.go`: director start は director の役割のエントリを使い、`[org.roles]` が残っていれば `LegacyRolesErr` で断る。
- 表示: `ModelOrCLIDefault` が、空のモデルを `(CLI default)` と出す。spawn の出力行、director start、`director status`、`ralph org status`、`ralph status`、`ralph org report`、`ralph insights`(S8)で使う。

## 4. 冪等の打ち直し(S8 `b4cfa641`、cross-review cycle 1 の ACTION_REQUIRED)

- S3 の形では、CLI の `ResolveRoleModel` が `org.Spawn` より前に `driver_pool` とモデル名を検査していた。そのため、既に立っている座席の打ち直しが断られることがあった。`spawn.go` の AC-3(冪等の no-op は検査で失敗してはならない)に反する。
- S8 で、2 つの検査を `ValidateSpawnEnvelope`(冪等の return のあと)と `checkDirectorStart` に移した。共通の文は `envelope.go` の `driverPoolErr` で作り、`DriverFromRoleEntry` があれば役割とエントリを名指す。`start --plan` には `StartFeatureParams.DriverFromRoleEntry` で届ける(`internal/org/feature.go`)。
- テスト: `internal/cli/org_role_models_retry_test.go`、`internal/org/spawn_role_models_retry_test.go`、`internal/cli/org_role_models_s8_gaps_test.go`(`fd573a50`)。
- 残る穴: `[org.role_models]` を最初の spawn のあとに書き換えた場合に限り、`ResolveRoleModel` の 2 つの断り(エントリのない役割でフラグが欠ける、`--driver` だけでエントリのモデルが空でない)が打ち直しの前に残る(self-review N3、tech-debt)。

## 5. doctor と init(S4 `39b50712`、S5 `14c77c43`、S7 `e401d1a6`)

- 「Org codex model slugs」と `--probe-models` は `[org.role_models]` を見る。モデルが `""` の組は `--model` なしで probe する(`internal/org/driver/probe.go`)。
- codex writable root の検査は、`driver_pool` に codex があるかで決める。役割ごとの許可リストの扱いは消した。
- 新しい検査は 2 つ。残ったキーを出す「Org legacy model keys」と、`driver_pool` にない driver のエントリを出す「Org role model drivers」。
- alias の warn(`doctor_shell_alias.go`)は、「`--model` を常に渡す」から「model が `""` でない座席にだけ渡す」に直した。
- `init_settings.go` は `driver_pool` だけを書き換える。

## 6. 残ったキー

- `model_pool`: 読み込みは通る。spawn・start・director start は stderr に 1 行出して進み、doctor は warn を出す。
- `[org.roles]`: 読み込みは通る。spawn・start・director start は何も記録せずに断る。`status`・`stop`・`disband` は動く(`internal/cli/org_legacy_model_keys_cleanup_test.go`)。doctor は warn を出す。

## 7. 文書(S6 `0ee2bbaf`、sync-docs `2cabff9a`・`a1480809`)

- leader の雛形: 「`--model` は必ず明示する」を消した。座席は `--driver` と `--model` を省いて立て、分割計画の前文に指定があるときだけ渡す。
- `/org` skill: 「既定の model_pool」節を「役割ごとのモデル(`[org.role_models]`)」節に置き換えた。`/director` skill: 前文に書く driver とモデルは、`ralph.toml` と違うものにする座席だけにした。
- `model-routing.md`、README、recipes、AGENTS.md、org-runtime spec の改訂注記、tech-debt の引き継ぎの注(25 行)。

## 8. 確かめたことと残したこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` は、cycle 1・2 とも rc 0 だった。golangci-lint は 0 issues。
  - verifier は HEAD の binary で、doctor、`spawn --dry-run`、打ち直し、`stop`・`disband` を確かめた。
  - tester の変異 30 個は、足したテストを含めてすべて殺された。
- 残したこと:
  - herdr の実機での起動。
  - `[reviewer].codex_model` のスラッグの doctor の検査(tech-debt)。
  - self-review の LOW(L4、L6 の後半、N2、N3)と verify の V2-1(tech-debt の 1 行)。
  - leader の雛形の手順 2 の規則のテスト。
