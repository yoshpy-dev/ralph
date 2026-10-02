# Test report: drop-gpt-5-5-default-pool

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md
- Tester: tester subagent (Claude)、cycle 1(pipeline cap 2)
- Branch: chore/drop-gpt-5-5-default-pool、HEAD b30eec66(push 済み)、base main 7dd6911c
- Scope: 振る舞いのテストだけ。静的解析、formatter、linter、check-sync、check-skill-sync、doctor の probe は verifier の担当なので実行していない(AC-5 は verify report を参照)。コード、テスト、plan は編集していない。mutation は `git archive` で作った scratch の複製だけで行い、毎回 worktree から戻して `cmp` で一致を確認した
- Evidence: `docs/evidence/verify-2026-10-02-053904.log`(default スコープの `run-test.sh`)、`docs/evidence/verify-2026-10-02-054351.log`(`RALPH_VERIFY_SCOPE=full`)。どちらも `docs/evidence/*.log` が gitignore のため commit には含まれない。mutation と CLI probe の出力は repo の外の scratch に置いた
- 隔離: すべての probe で `HOME=<scratch>/home`、`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`CODEX_HOME=<scratch>/codex-home`。利用者の `~/.gitconfig`、`~/.codex`、HOME の rc、herdr server には触れていない。CLI probe の `herdr` / `agmsg` は PATH 先頭に置いた呼び出し記録だけの stub

## Test execution

default の `./scripts/run-test.sh` が選ぶもの: `==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)` → `Language packs selected: golang`。`./scripts/detect-changed-languages.sh` の直接実行も `scope=full`、`reason=unclassified:scripts/ralph-config.sh`、`golang_roots=.`。diff の Go 3 ファイルだけで選ばれたのではなく、未分類ファイルによる full fallback で選ばれている(verify report と同じ観察)。このため default でも `RALPH_VERIFY_SCOPE=full` でも実行内容は同じ。

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(default スコープ、フォアグラウンド) | shell 32 suite + Go 8 package | shell 32/32 suite OK、Go 8/8 package ok | 0 | 0 | 3:56 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(フォアグラウンド) | 同上 | shell 32/32 suite OK、Go 8/8 package ok | 0 | 0 | 3:47 |
| 上の shell suite のアサーション数(各 suite 末尾の集計行を足した値。集計行を出さない suite が `tests/test-no-loop-references.sh` の 1 つあり、OK 判定は出ている) | 件数を出す 31 suite の PASS 合計 1,427 | 1,427 | 0(`FAIL: N` 行は 22 行すべて 0、非 0 の FAIL 行なし) | 0 | — |
| `go test ./... -count=1 -v -cover`(uncached) | 1,397(subtest 込み、top-level 993) | 1,397 | 0 | 2(既存の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`。mock embed.FS 前提) | cli 46.1s、org 9.8s、protocol 3.3s、upgrade 3.5s、insights 2.1s、driver 1.9s、config 1.1s、scaffold 0.7s |
| `TMPDIR=/tmp go test ./internal/config/... ./internal/cli/... ./internal/org/... -count=1`(CI の ubuntu 相当) | 5 package | 5 ok | 0 | 0 | cli 43.5s、org 10.9s |

この branch が足したテスト(`git diff main...HEAD -- '*_test.go'`): 新規 top-level 2 本(`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel`、サブテスト 3 つ。`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit`)。既存の追従 3 本(`TestDefault_Org` に `gpt-5.5` 不在のループ、`TestLoad_OrgRolesEmpty` の 9 → 8、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder` の期待スラッグから `gpt-5.5` を削除)。削除したテストはない。

### Flakiness(新規・変更したテスト)

対象: `TestDefault_Org`、`TestLoad_OrgRolesEmpty`、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors`、`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit`、`TestDefaultsLockStep`(config)、`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel`(cli)。

| 実行 | 結果 |
| --- | --- |
| config 6 本 `-count=3 -v` | 18/18 PASS、FAIL 0 |
| cli のサブテスト 3 つ込みの 1 本 `-count=3 -v` | 親 3/3、各サブテスト 3/3 PASS |
| config 6 本 `-race -count=50` | ok(2.1s) |
| cli の 1 本 `-race -count=50`。同じシェルで `go test ./internal/cli/ -count=1`(suite 全体)をバックグラウンドで同時に走らせた負荷下 | ok(2.6s)。バックグラウンドの suite も ok(41.4s) |

flaky なものはなかった。新規テストは時計、ネットワーク、実プロセスに依存せず(`PATH=""` で git を隠し、`t.TempDir()` と `t.Chdir()` だけを使う)、flake の経路がない。memory にある既知の flake(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)は今回の実行では出なかった。

## Coverage

- Statement(`go test -cover`、HEAD。同じ機械・同じ手順で main の archive を測った値を括弧に入れた): config 92.3%(92.3%)、cli 84.6%(84.4%)、org 90.5%(90.5%)、org/driver 92.0%、org/protocol 97.9%、insights 86.1%、scaffold 75.7%、upgrade 91.2%
- Branch / Function: 計測していない(Go の標準ツールは statement のみ。shell は計装の手段なし)
- Notes:
  - 変更した本番コードは `internal/config/config.go` のデフォルト値 1 行の削除だけで、新しい分岐も新しい関数もない。config の coverage が動かないのは想定どおり
  - cli の +0.2pp は、この branch が `internal/cli` の本番コードを触っていないので差分の原因にはならない。時間依存の分岐の実行回数による run-to-run の揺れと判断した(未確認。再測定はしていない)
  - 新規 2 本は coverage を増やすためではなく、既存行の判別力を上げるためのテスト。判別力は下の mutation 表で測った

## Mutation testing(scratch の複製、tracked ファイルは未変更)

各 mutation のあと `go test ./internal/config/ ./internal/cli/ ./internal/org/ -count=1` を実行し、さらに新規テストを単独で実行した(`TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` が index out of range で panic すると package 全体が止まり、後ろのテストが走らないため。mutation e で実際に起きた)。

| ID | Mutation | 落ちたテスト | 判定 |
| --- | --- | --- | --- |
| a | `Default()` にだけ `gpt-5.5` を戻す | `TestDefault_Org`、`TestDefaultsLockStep`、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit`、`TestLoad_OrgRolesEmpty`、`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` | 検出 |
| b | `templates/base/ralph.toml` にだけ戻す | `TestDefaultsLockStep` | 検出 |
| c | `scripts/ralph-config.sh`(root)にだけ戻す | `TestDefaultsLockStep` | 検出 |
| c2 | `templates/base/scripts/ralph-config.sh` にだけ戻す(plan の (c) に含まれない追加) | なし(config / cli / org のどのテストにも出ない) | 生存。root と template の byte 一致は `scripts/check-sync.sh`(verifier の静的 gate、verify report では `cmp` と合わせて green)の担当で、振る舞いのテストは見ない。実行はしていない |
| d | `newOrgRuntimeAt` が `--config` を無視する(`resolveOrgConfig("")`) | AC-7 のテスト(親 + 3 サブテスト)、ほか cli の 10 本(`TestOrgSpawn_*`、`TestOrgStart_*`、`TestOrgStop_CLI_Codex*`) | 検出 |
| e | state dir を設定ファイルのディレクトリから決める | AC-7 のテスト(親 + 3 サブテスト、`got: no seats`)、ほか `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry`(panic)、`TestOrgSpawn_Rejection_ModelOutOfPoolAndMaxSeatsWithOrgIsolation` | 検出(AC-7 は単独実行で確認。全 package 実行は panic で途中終了) |
| f | `ResolveOrgStateDir` の `RALPH_ORG_STATE_DIR` の段を外す | AC-7 の `state_dir_from_RALPH_ORG_STATE_DIR_without_--state-dir`、`TestResolveOrgStateDir_EnvWinsOverGitAndCwd`、`TestInsightsCmd_ReceiptsDefaultPathFromOrgStateDir` | 検出 |
| g | 明示した `model_pool` を既定と dedup マージする(置き換えをやめる) | `TestLoad_OrgFullRoundTrip`、`TestLoad_OrgRejects`(`explicitly_empty_model_pool`、`roles_reference_unknown_model`)、`TestOrgSpawn_ModelFlagOmitted_RespectsRoleAllowlist`、`TestOrgSpawn_Rejection_ModelOutOfPoolAndMaxSeatsWithOrgIsolation`。新規 2 本は通る | 検出(既存テストが検出。skill の「明示した model_pool は置き換え」の主張はこの既存テストが守っている) |
| g2 | 明示した `model_pool` を既定の後ろにそのまま足す | 上に加えて `TestDefaultsLockStep`、`TestLoad_TemplateRalphToml`、新規 2 本、`TestOrgSpawn_*` / `TestOrgStart_*` 数本 | 検出 |
| h | `driver_pool` だけの上書きのとき、`[org.roles]` を絞る前の既定プール全体で検証する | なし | 生存。実際のギャップ(下の G-1)。main では同じ mutation を `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors` が検出するので、この branch で判別力が落ちた |
| l | 3 面で `gpt-5.6-luna` を落とす(別のスラッグを誤って外した想定) | `TestDefault_Org`、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`TestLoad_OrgRolesEmpty` | 検出 |
| m | 3 面すべてで `gpt-5.5` を戻す(lock-step を素通りする回帰) | `TestDefault_Org`、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit`、`TestLoad_OrgRolesEmpty`、AC-7 のテスト。`TestDefaultsLockStep` は通る(3 面が一致するため) | 検出(lock-step の外で検出できている) |
| o | `resolveOrgConfig` が読み込みエラーを握りつぶして `Default().Org` を返す | AC-7 のテスト(壊れた設定で status が失敗することの確認) | 検出 |
| q | `[org.roles]` のモデル検証を飛ばす | `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors`、`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit`、`TestLoad_OrgRejects/roles_reference_unknown_model`、AC-7 のテスト | 検出 |
| r | 3 面で `gpt-6-astra` と `gpt-5.6-sol` の順序を入れ替える(codex の省略時フォールバック先が変わる) | `TestDefault_Org`、`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`TestDefaultModelForDriver_DefaultPoolHeads`(org) | 検出 |
| s | `driver_pool` の絞り込みが全エントリを残す | `TestLoad_DriverPoolOnlyOverride_FiltersInheritedDefaultModelPool`、`..._Codex_KeepsOnlyCodexDefaultEntriesInOrder`、`..._NoDefaultModels_Errors`、`..._RolesReferencingFilteredModelErrors` | 検出 |
| t | org skill の表に `gpt-5.5` の行を 4 面そろえて戻す(文書だけ) | なし(config / cli / org / scaffold) | 生存。文書の表を `Default()` に結びつけるテストはない。4 面そろえた変更は `check-skill-sync` でも見えない。AC-4 の `git grep` による人手の確認が唯一の gate(下の G-2) |

17 件中、検出 14、生存 3(c2、h、t)。生存のうち、この変更が原因で判別力が落ちたのは h だけ。c2 と t は既存の設計上の空白で、この branch で新しく生まれたものではない。

h を再現した手順: (1) main の archive に同じ mutation を当てると `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors` が `Load: expected error, got nil` で落ちる。(2) HEAD では通る。(3) scratch でこのテストの fixture の `reviewer = ["gpt-5.5"]` と期待文字列を `gpt-6-astra`(`driver_pool = ["claude"]` で絞られ、既定のプールには残るスラッグ)に変えると、無変更では通り、mutation h では `Load: expected error, got nil` で落ちる。

## Edge cases(plan の Test plan から。ビルドは HEAD、隔離は上記のとおり)

| ケース | 結果 |
| --- | --- |
| `driver_pool = ["codex"]` だけで `model_pool` なし → 絞った既定は codex の 4 スラッグそのもの | `TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder` が長さ 4、順序 `gpt-6-astra`、`gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna`、すべて codex、claude なしを確認(pass)。CLI でも同じ設定で `org spawn --dry-run` を 4 スラッグそれぞれ rc 0、`gpt-5.5` と `fable` は `org: model "<m>" not in [org].model_pool for driver "codex"` で rc 1、`--driver claude` は `driver "claude" not in [org].driver_pool [codex]` で rc 1 |
| `[org.roles]` が `gpt-5.5` を指定し、プールは既定 → 検証エラー | `org: load config: [org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool`、rc 1。`--config <file>` でも cwd の `./ralph.toml` からの暗黙の読み込みでも同じ。壊れた cwd の `ralph.toml` では、`gpt-5.6-sol` という有効なモデルを渡した `org spawn --dry-run` も同じ文面で止まる(全 verb が設定を読む、という復旧の段落の主張と一致)。config 層では `TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit` が同じ文面を完全一致で確認し、明示した `model_pool` に `gpt-5.5` を入れた設定は読み込めることも確認 |
| `ralph org spawn --model gpt-5.5` + 既定のプールは herdr に触る前に拒否される | `--dry-run` あり・なしの両方で `rejected: org: model "gpt-5.5" not in [org].model_pool for driver "codex"`、rc 1。stub の `herdr` / `agmsg` の呼び出し記録は空(`gpt-5.5` の拒否 3 回と、有効なモデルの `--dry-run` 1 回のあとも 0 行)。対照: 有効なモデル `gpt-5.6-sol` の実 spawn(非 dry-run)では stub の `herdr` が 3 回呼ばれた(`workspace create`、`tab create`、`agent start`)。stub が配線されていて、拒否のときだけ呼ばれないことを区別できている。有効なモデルの `--dry-run`(guarded 権限の設定)は rc 0 で `spawned seat "r3" (... model=gpt-5.6-sol ... dry_run=true)` |
| 復旧の経路(`--config` と `RALPH_ORG_STATE_DIR`) | AC-7 のテストが、壊れた設定で `status` が上のエラーで失敗し、直した設定を `--config` で渡すと同じ state dir の座席 `seat-1` が見えることを、`--state-dir` 経路 2 通りと `RALPH_ORG_STATE_DIR` 経路 1 通りで確認(pass)。mutation d、e、f、o がそれぞれ別のサブテストで落ちることを確認済み |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | 全 suite が green。mutation の実行中に落ちたテストは、すべて意図した mutation による期待どおりの失敗 | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既定のプールに `gpt-5.5` が含まれる(#156、retiring の予告と "Legacy" 表示) | 解消。再発は a、m で検出 | `TestDefault_Org`、`TestLoad_OrgRolesEmpty`(8 件)、`TestDefaultsLockStep`、`go test ./... -count=1` green |
| 3 面のうち 1 つだけ更新を忘れる | 検出 | mutation b、c(`TestDefaultsLockStep`)。template の `ralph-config.sh` だけは check-sync の担当(c2) |
| 既定にないモデルを `[org.roles]` や `--model` で使うと止まる(復旧の段落が前提にする挙動) | 再現と復旧を確認 | edge case の 2 行目、3 行目、4 行目 |
| 明示した `model_pool` が既定を丸ごと置き換える(復旧の段落が前提にする挙動) | 既存テストが守っている | mutation g、g2(`TestLoad_OrgFullRoundTrip` ほか) |
| org の state dir が設定ファイルの場所で変わらない(復旧の段落が前提にする挙動) | テストが守っている | mutation e、f |

## Test gaps

- G-1(MEDIUM、この変更で判別力が落ちた): `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors`(`internal/config/config_test.go:819`)は fixture で `driver_pool = ["claude"]` と `reviewer = ["gpt-5.5"]` を使っている。`gpt-5.5` が既定のプールから消えたため、「`driver_pool` で絞られたプールに対して `[org.roles]` を検証する」と「絞る前の既定プール全体に対して検証する」の区別がつかなくなった(mutation h が生存。main では同じ mutation をこのテストが検出していた)。テストのコメントにある `(or that was never in the default pool)` の側だけを検査するテストに変わっている。提案: fixture の `gpt-5.5`(2 か所: TOML と `want`)を、既定のプールに残る codex スラッグ(`gpt-6-astra`)に変える。scratch で、無変更は pass、mutation h は fail になることを確認済み。テスト fixture だけの 1 か所の変更で、本番コードは動かない。`.claude/rules/ralph/testing.md` の「意図を弱めたテストの変更は plan と report に書く」にも関わるので、直さない場合は known gap として PR に書く
- G-2(LOW、既存の空白): org skill の「既定の model_pool」の表(4 面)は `config.Default()` に結びつけたテストがない(mutation t)。4 面そろえた変更は `check-skill-sync` でも検出できない。この変更では AC-4 の `git grep` を人手で当てた(verify report)。表のスラッグを `Default()` と突き合わせるテスト(`send_defaults_sync_test.go` と同じ型)を足せば機械化できるが、この変更の範囲外
- G-3(LOW、既存の空白): `templates/base/scripts/ralph-config.sh` と root の byte 一致は `scripts/check-sync.sh` だけが見る。`defaults_sync_test.go` は root の `scripts/ralph-config.sh` しか読まない(mutation c2)。静的 gate が必ず走る前提なら問題にならない。この `/test` では check-sync を実行していない(verifier の担当)
- G-4(INFO、既存): `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry`(`internal/cli/org_test.go:710-711`)は `events[len(events)-1]` の前に長さを確認しない。回帰が起きると `t.Fatalf` ではなく `index out of range` の panic になり、cli package の残りのテストが走らず結果が見えなくなる(mutation e で実際に起きた)。この変更とは無関係。
- 実行していないもの: `ralph doctor` の probe(AC-5。verify report が HEAD と main の対照で確認済みなのでそれに依拠)、利用者の実際の codex cache での確認(plan の取り決めで合否に使わない)、`ralph org start` の CLI 実行(コードの経路は spawn と同じ。AC-7 の段落が `start` を名指す部分は spawn の CLI probe とコードで確認した verify report に依拠)

## Verdict

- Pass: 全 suite が green。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` と default の `./scripts/run-test.sh` は shell 32/32 suite、Go 8/8 package(uncached の `go test ./... -count=1 -v`: 1,397 PASS / 0 FAIL / 2 SKIP(既存))。`TMPDIR=/tmp` も 5 package ok。新規・変更したテストは `-count=3`、`-race -count=50`、負荷下で flaky なし。AC-6 の振る舞い側は確認できた(静的側は verify report)。AC-7 のテストは mutation a、d、e、f、m、o、q を検出し、d、e、f、o はそれぞれ別のサブテストが落ちる。edge case 4 件はすべて期待どおり
- Fail: なし
- Blocked: なし
- 推奨: G-1 は fixture の 1 か所の変更で直せるので、PR の前に直すことを勧める(直すなら fix → `/self-review` → `/verify` → `/test` → `/sync-docs` の再実行が必要で、cycle 2 が最後の実行になる)。直さずに進める場合は PR に known gap として書く。G-2 から G-4 は既存の空白で、この変更では扱わない
