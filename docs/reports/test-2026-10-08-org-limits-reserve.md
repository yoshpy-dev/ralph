# Test report: org-limits-reserve

- Date: 2026-10-08(JST。実行の記録は UTC の 2026-10-08 11:54〜12:49)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch feat/org-limits-reserve と base 51855166 の差分。コードの commit は 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI と `statedir.go`)、8dd19634(文言)。テスト開始時の HEAD は fbb04f83。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC11、AC13〜AC15 を見た。AC12 は文書の基準なので /verify の担当
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-115436.log`(テスト追加前)と `docs/evidence/verify-2026-10-08-124051.log`(追加後)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加前、fbb04f83) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 342 s、rc 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加後、c3a95c48) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 502 s、rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... ./internal/config/... -count=1`(追加前) | 5 パッケージ | 5 | 0 | 0 | `internal/cli` 94 s、rc 0 |
| `go test -race ./internal/org/ -count=1`(追加後) | 1 パッケージ | 1 | 0 | 0 | 14 s、rc 0 |
| 新しいテストを `-count=3`(追加前。下の正規表現) | top-level 168(56 本 × 3。PR の新しいテスト 41 本をすべて含む) | 168 | 0 | 0 | rc 0 |
| 新しいテストを `-count=3`(追加後。正規表現に `CloseDeferredSelfPane` を足した) | top-level 189(63 本 × 3。新しいテスト 43 本を含む)、subtest 468 | 657 | 0 | 0 | rc 0 |
| `go test ./... -count=1 -coverprofile`(追加後) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 99 s、rc 0 |
| mutation(依頼の 14 種を 16 件に分けたものと、追加の 24 件。`go test -overlay` で差し替え、worktree には触れていない) | 40 件と確認 1 件(P1) | red 39 件 | 生き残り 1 件(X18)。X24 は Phase 2 のテストを足す前は生き残り、足したあと red | - | 1 件 15〜140 s |
| M14(ロックなし)で `-run Concurrent -count=10` | 新しい競合のテスト 3 本 × 10 | - | 29 回 red / 30 回 | - | rc 1 |

- `-count=3` の正規表現は `Reserve|Reservation|OrgWide|RunningOrgs|MainWorktree|ZeroOrgWide|MaxOrgs|MaxTotalSeats|Concurrent|TotalActiveSeats|ReservationOnlyOrg|OrgFleetLimits|ScopeReservedDetails|NormalizeReservePaths|ReservePathsOverlap|ActiveReservation|CloseDeferredSelfWorkspace|ShowsReservation|ReserveFlag|Defaults` で、`internal/org`、`internal/cli`、`internal/config` に掛けた。base からの差分で足された `func Test` 41 本が、1 回目の実行の `--- PASS` にすべて出ることを `comm` で確かめた。既存の `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` なども含む。
- 2 回の `run-test.sh` は、どちらも `Requested scope: full`、`Language scope: full`、`Language packs selected: golang` で走った。2 回とも、中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため、テスト追加後に `-count=1` で全パッケージを流し直した(表の 7 行目)。`internal/cli` は 97 s で、前回の記録(70〜100 s)の範囲に入る。
- shell の件数は、各 suite の集計行(`PASS: 29`、`PASS: 47 / 47`、`22 passed, 0 failed` など)の合計 2,046 件と、集計行のない `tests/test-no-loop-references.sh` の 1 件を足したもの。org-watch-stop-failed の test レポートと同じ 2,047 件で、この差分は shell のテストを変えていない。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回とも出なかった。失敗したテストがないので、単独の再実行はしていない。

## Coverage

- Statement(追加後): `internal/org` 93.0%(org-stop-all cycle 2 の記録は 92.6%)、`internal/cli` 85.3%(同じ)、`internal/config` 92.8%(92.3%)。ほかの 5 パッケージは前回と同じ(`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数):

| 関数 | 位置 | Coverage | 通らない文 |
| --- | --- | --- | --- |
| `NormalizeReservePaths` / `normalizeReservePath` | `reserve.go:45` / `:61` | 100.0% / 100.0% | なし |
| `reservePathsOverlap` | `reserve.go:90` | 100.0% | なし |
| `RunningOrgs` | `reserve.go:233` | 100.0% | なし |
| `ActiveReservation` | `reserve.go:256` | 100.0% | なし |
| `TotalActiveSeats` | `reserve.go:265` | 100.0% | なし |
| `ValidateOrgWideCapacity` | `envelope.go:92` | 100.0% | なし |
| `spawnCapacityErr` | `spawn.go:1063` | 100.0% | なし |
| `reservationDecision` | `reserve.go:283` | 100.0% | なし |
| `idempotentRespawn` | `spawn.go:1085` | 87.5% | `scope_reserved` の追記の失敗(`:1092`) |
| `reserveAgain` | `verbs.go:1225` | 75.0% | すでに予約を持つときの戻り(`:1227`)と、追記の失敗(`:1233`) |
| `orgsToDisband` | `verbs_all.go:309` | 100.0% | なし |
| `withMainWorktreeOrgLimits` | `cli/org.go:148` | 100.0% | なし |
| `MainWorktreeRoot` | `statedir.go:81` | 100.0% | なし |

- そのほかの変わった関数: `checkCapacityAndStart` 69.2%(`scope_reserved` と、前からある `spawn_started` の追記の失敗)、`dryRunSpawn` 93.7%、`autonomousScopeGateErr` 100.0%、`CloseDeferredSelfWorkspace` 100.0%、`newOrgSpawnRuntime` 90.0%(`newOrgRuntimeAt` の失敗)、`orgReservation` 75.0%(台帳の読み込みの失敗)、`printStatusTable` / `printStatusJSON` 100.0%、`config.Load` 92.9%、`config.Default` 100.0%
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 通らない文は、`reserveAgain` の `:1227` を除くと、台帳への追記や読み込みが失敗する経路だけである。`:1227` は下の mutation X18 の生き残りにあたる

## Mutation

各 mutation は、対象のファイルの写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off <パッケージ>` で流した。パッケージは、そのファイルを持つパッケージと、CLI から届くものは `internal/cli` も含めた全体である(`-run` で絞っていない)。worktree のファイルは書き換えていない。M14 だけは、全体を流す代わりに競合のテストを `-count=10` で流した。

依頼の 14 種:

| # | Mutation | Result | Red になったテスト(主なもの) |
| --- | --- | --- | --- |
| M1 | `ValidateOrgWideCapacity` の max_orgs の判定を外す | red | `TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`MaxOrgs_StoppedOrgHoldsItsSlotUntilDisbanded`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、CLI の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml` ほか、計 9 本 |
| M2 | max_total_seats の判定を外す | red | `TestValidateOrgWideCapacity`、`MaxTotalSeats_Boundary`、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded`、CLI の `MainWorktreeRalphToml_OnlyOrgWideLimitsTaken`、追加した `CloseDeferredSelfPane_..._ReactivationCanExceedMaxTotalSeats` ほか、計 7 本 |
| M3 | `RunningOrgs` が `rejected` だけの座席の org も数える | red | `TestRunningOrgs`、`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded` ほか、計 6 本 |
| M4 | `currentOrgLives` が最後の `disbanded` より前の記録も数える | red | `TestRunningOrgs`、`TestActiveReservation`、`TestReservationBeforeLastDisband`、`Reserve_DisbandReleasesIt`、`DisbandAll_ReservationOnlyOrgDisbanded` ほか、計 8 本 |
| M5 | ディレクトリどうしの重なりで末尾の `/` を外して比べる(区切りの単位を無視) | red | `TestReservePathsOverlap`(`internal/auth/` と `internal/authz/`)、`TestReservationDecision`、`Reserve_OverlapWithRunningOrgRejected` |
| M5b | ディレクトリとファイルの重なりで同じことをする | red | `TestReservePathsOverlap`(`internal/authz.go`、`internal/auth`) |
| M6 | `..` の拒否を外す | red | `TestNormalizeReservePaths_Rejects`、`Reserve_InputRejectedBeforeAnyRecord` |
| M7 | leader 以外の座席の予約を通す | red | `Reserve_InputRejectedBeforeAnyRecord`、CLI の `TestOrgSpawn_Reserve_LeaderOnly` |
| M8 | `idempotentRespawn` が予約を判定しない | red | `Reserve_ExistingLeader`、追加した `Reserve_ExistingLeader_Phase2`、CLI の `TestOrgStart_Reserve_SameListPassesDifferentListRejected` |
| M9 | すでに予約がある org に違う一覧を通す | red | `TestReservationDecision`、`Reserve_SameSetPassesDifferentSetRejected`、`Reserve_ExistingLeader`、CLI の `SameListPassesDifferentListRejected` |
| M10 | `orgsToDisband` から `scope_reserved` を外す | red | `TestOrgsToDisband_Definition`(「reservation only, never disbanded」など) |
| M11 | `CloseDeferredSelfWorkspace` が `reserveAgain` を呼ばない | red | `CloseDeferredSelfWorkspace_CloseFails_ReservationRestored`、`..._RiskWindowClearedByRetry` |
| M12 | main worktree の `ralph.toml` を読むが値を使わない(打った場所の設定のまま) | red | CLI の `ReadFromMainWorktreeRalphToml`(サブディレクトリと linked worktree の 2 件)、`OnlyOrgWideLimitsTaken` |
| M12b | main worktree の `ralph.toml` を読まない | red | M12 の 2 本と `TestOrgStart_MainWorktreeRalphTomlLoadError` |
| M13 | `--state-dir` や env で台帳を決めたときも main の上限を使う | red | CLI の `ReadFromMainWorktreeRalphToml` の「--state-dir naming the shared ledger」と「RALPH_ORG_STATE_DIR naming the shared ledger」 |
| M14 | `withManifestLock` が flock を取らない | red(30 回中 29 回) | `-count=10` で `ConcurrentSpawns_MaxOrgsNeverExceeded` 10/10、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded` 10/10、`ConcurrentReservations_OnlyOneOfOverlappingWins` 9/10 が red。既存の `MaxSeatsNeverExceeded` も 9/10、`TestWithManifestLock_SerializesConcurrentCallers` は 10/10 |

追加の 24 件(境界、読み戻し、CLI の周辺。P1 は mutation ではなく、追加したテストが区別できるかの確認):

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| X1 / X2 | max_orgs / max_total_seats の `>=` を `>` にする | red / red | `TestValidateOrgWideCapacity`、`ZeroOrgWideLimits_Reject`、境界のテスト |
| X3 | max_orgs を走っている org にも掛ける | red | `MaxOrgs_NewOrgRejectedAtLimit`(org-a の 2 席目)、`TestValidateOrgWideCapacity` |
| X4 | dry-run の org の記録も数える | red | `TestRunningOrgs`、`TestActiveReservation`、`Reserve_DryRunPreviewsAndHoldsNothing` |
| X5 | 読めない `scope_reserved` を無視する(repo 全体として扱わない) | red | `TestActiveReservation`(「unreadable is the whole repo」) |
| X6 | scope のゲートが `Reserve` を見ない | red | `Reserve_SatisfiesAutonomousScopeGate`、`--scope` なしの `--reserve` で leader を立てる CLI のテスト 5 本 |
| X7 | `--config` を渡しても main の上限を使う | red | CLI の `ReadFromMainWorktreeRalphToml`(「--config with max_orgs = 10」) |
| X8 | main から `max_seats` も取る | red | CLI の `OnlyOrgWideLimitsTaken`(「max_seats stays the caller's」) |
| X9 | main の `ralph.toml` の読み込みの失敗を無視する | red | CLI の `MainWorktreeRalphTomlLoadError` |
| X10 | `status` が `reserved:` の行を出さない | red | CLI の `TestOrgStatus_ShowsReservation`、`ReservationWithoutSeatsAndAfterDisband` |
| X11 | 立っている leader への予約の拒否で `rejected` を書く | red | `Reserve_ExistingLeader`(記録が増えない、の検査)、CLI の `SameListPassesDifferentListRejected` |
| X12 | dry-run の trail に `scope_reserved` を出さない | red | `Reserve_DryRunPreviewsAndHoldsNothing` |
| X13 / X14 | `config.Load` が `max_orgs` / `max_total_seats` の 0 を通す | red / red | `TestLoad_OrgFleetLimitsBoundary`(X13 は CLI の `MainWorktreeRalphTomlLoadError` も) |
| X15 / X16 | 既定の 10 / 30 を 11 / 31 にする | red / red | `TestDefault_Org`、`TestDefaultsLockStep` |
| X17 | 予約のパスを並べ替えない | red | `TestNormalizeReservePaths`、`Reserve_RecordedBeforeSpawnStarted` ほか、計 4 本 |
| X18 | `reserveAgain` の「すでに予約を持つなら書かない」を外す | **生き残り** | なし(下の Test gaps 3) |
| X19 | `MainWorktreeRoot` の末尾の検査を外す | red | `TestMainWorktreeRoot` |
| X20 | `reservedPathsFromDetails` が注記(`restored: …`)を切らない | red | `TestScopeReservedDetails_RoundTrip`、`..._ReservationRestored` |
| X21 | `checkCapacityAndStart` が予約を記録しない | red | 14 本 |
| X22 | `org_workspace_closed` で workspace を外さない | red | `TestRunningOrgs`(「workspace created then closed」) |
| X23 | `reservationDecision` がほかの org の予約を見ない | red | 7 本 |
| X24 | Phase 2 の idempotent の戻り(`spawn.go:760`)で予約を判定しない | red(テスト追加前は生き残り) | 追加した `TestOrgSpawn_Reserve_ExistingLeader_Phase2` |
| P1 | (確認)pane の補償が max_total_seats に達していたら座席を戻さない、という仮の修正 | red | 追加した `CloseDeferredSelfPane_..._ReactivationCanExceedMaxTotalSeats`。このテストは今の振る舞い(上限を 1 つ超える)を固定しており、補償が上限を見るように変わると落ちる |

- mutation は 2 回に分けて流した。1 回目(M3〜M6、M10、M11、X1〜X5、X12、X14〜X20、X22〜X24)は、`verbs_test.go` に V-2 のテストを足したあと、`spawn_test.go` に Phase 2 のテストを足す前に流した。2 回目(M1、M2、M7〜M9、M12、M12b、M13、X6〜X11、X13、X21、X24、P1)と M14 は、2 本とも足したあとに流した。
- 1 回目で生き残ったのは X18 と X24 の 2 件である。X24 は Phase 2 のテストを足し、2 回目で red になることを確かめた。X18 にはテストを足していないので、流し直していない。
- M14 の 30 回中 1 回の pass は、goroutine の読みと書きがたまたま直列に並んだ回とみている。ロックがあれば `-count=3` の 3 回とも、`-race` でも通る。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 今ある `max_seats` の判定と、その競合のテスト | 保たれている | `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` が `-count=3` と `-race` で通る。`spawnCapacityErr` は `ValidateSpawnCapacity` を最初に呼ぶ |
| scope のゲート(AC-2b) | 保たれている | `--scope` も `--reserve` もない autonomous の spawn は今どおり拒否される(`Reserve_SatisfiesAutonomousScopeGate` の 2 つ目の検査)。X6 で red |
| spawn の idempotent(立っている座席は上限の判定の前に返す) | 保たれている | `MaxTotalSeats_Boundary` の上限ちょうどでの respawn がイベントを増やさない。Phase 2 の idempotent の既存テスト(`StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent`)も通る |
| 2 段目の自分の close の補償(workspace と pane) | 保たれている | 既存の `CloseDeferredSelfWorkspace_*` と `CloseDeferredSelfPane_*` がすべて通る。予約のない org では補償の記録が増えない(`..._ReservationRestored` の「no reservation, nothing restored」) |
| `disband --all` の対象 | 保たれている | `TestOrgsToDisband_Definition` の既存の場合がすべて通り、予約だけの org が加わった。M10 で red |

## Test gaps

1. 追加したテスト(c3a95c48、`internal/org/spawn_test.go` と `internal/org/verbs_test.go` だけ、+75 行。本番のコードは変えていない):
   - `TestOrgSpawn_Reserve_ExistingLeader_Phase2`: AC14 の Phase 2 側の穴。stale な saga の補償の間に別の呼び出しが leader を立てた場合、Phase 2 の新しい読みで見つけた leader にも予約を判定して記録する。既存の `afterStaleCompensation` の継ぎ目を使った。X24 が生き残ったので足し、red になることを確かめた
   - `TestOrgCloseDeferredSelfPane_CloseFails_ReactivationCanExceedMaxTotalSeats`: verify の V-2 の pane の経路を、今の振る舞いのまま固定したもの。座席の `stopped` から補償までの間に別の座席が立つと、補償は上限を見ずに座席を戻すので `max_total_seats` を 1 つ超える(1 席の上限で 2 席が動く)。新しい座席は拒否され、herdr が答えるようになってからほかの pane で stop をやり直すと 1 席に戻る。要件ではなく記録であり、補償が上限を見るように直すときは、このテストも一緒に直す。P1 で、そうした修正を入れるとこのテストが落ちることを確かめた
   - どちらも不要なら、この commit だけを戻せば元に戻る
2. V-2 の窓そのものは残る。workspace の経路は既存の `..._RiskWindowClearedByRetry` が、pane の経路は追加したテストが、上限を超えた状態と打ち直しで解けることを固定している。補償が上限や重なりを見るようにするかは、/sync-docs か cross-review の判断になる
3. X18(`reserveAgain` の `verbs.go:1226-1228`、すでに予約を持つ org には書かない)はテストがない。この戻りが効くのは、`disbanded` のあとに同じ org_id が `--reserve` で立て直され、それを `CloseDeferredSelfWorkspace` の冒頭の読みが見た場合だけである。`ralph org disband` は `Disband` のすぐあとに同じプロセスでこの読みをするので、その間に立て直しが入る窓は短い。現実に起きやすいのは、herdr の呼び出しの間(最大 3 回 × 10 秒)に立て直される場合で、そのときは読みが古いのでこの戻りでは防げない(self-review F-5)。そのため AC の穴とはみなさず、テストを足さなかった。F-5 を直す(追記の直前に読み直す)なら、そのテストでこの戻りも押さえられる
4. 別プロセスの競合は再現していない。AC4 の 3 本は同じプロセスの goroutine で flock を競わせる。flock は open ごとに掛かるので、プロセスが分かれても同じ仕組みで直列になるはずだが、未確認である
5. 台帳への追記と読み込みの失敗の経路(`checkCapacityAndStart` の `scope_reserved`、`idempotentRespawn`、`reserveAgain`、`orgReservation`)はテストがない。どれもエラーを返すだけの分岐で、前からある `spawn_started` の追記の失敗と同じ扱いである
6. 予約のパスは大文字小文字を区別して比べる(self-review の Coverage gaps)。macOS の既定のファイルシステムで `Internal/` と `internal/` が重ならないことは、テストでも固定していない。plan に書かれていない扱いで、tech-debt の候補として /sync-docs に渡っている
7. bare リポジトリの linked worktree(台帳の source が `git-toplevel`)で main の `ralph.toml` を読まないことは、`TestMainWorktreeRoot` の単体の場合だけで見ている。CLI からは打っていない(verify の V-1 と同じ)
8. 本物の herdr と agmsg では動かしていない

## Verdict

- Pass: pass。`run-test.sh`(full)は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`-race`、新しいテストの `-count=3`、`go test ./... -count=1` も 0 failed。mutation は依頼の 14 種(16 件)がすべて red で、追加の 24 件のうち X24 はテストを足して red になり、X18 だけが生き残った。AC1〜AC11 と AC13〜AC15 をテストの実行で確かめた
- Fail: なし
- Blocked: なし。Test gaps 2〜8 は merge を止めない
