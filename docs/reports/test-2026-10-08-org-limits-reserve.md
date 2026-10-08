# Test report: org-limits-reserve

- Date: 2026-10-09(JST。実行の記録は UTC の 2026-10-08 15:44〜16:01。ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 2(`cycle-count.json` は 2。既定の上限の最後の回)
- Scope: HEAD f144021d。cycle 1 の test(fff1ec23)からのコードの差分は 3 つの commit にある。76d1cf1c は `internal/cli/org.go` の help の文言(`orgWideLimitsHelp`、`--config` の説明、`status` の `Long`)。975df92b と afcbc6c2 は `internal/org` の修正で、動いていない leader の予約の前に max_orgs を判定し、その判定を `validateMaxOrgs` に切り出して `ValidateOrgWideCapacity` と共有する(`envelope.go`、`spawn.go`、テストは `spawn_test.go` の 1 本)。ほかは文書の commit。behavioral test だけを実行した(静的解析は /verify の担当)。重点は、AC1 と AC14 にかかる新しい枝、self-review C2-3 の `if !seat.Active`、cycle 1 の安全側の mutation の再確認
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log` の末尾の「cycle 2」の節(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-154409.log`
- 番号の付け方: cycle 1 の Test gaps 1〜8 と mutation の番号(M1〜M14、X1〜X24、P1)は、付録 A に原文のまま残す。tech-debt の行 172 と 176 がその番号を指しているため。この回の新しい mutation は N1 から、新しい Test gaps は T2-1 から振る。cycle 1 の mutation を流し直したものは cycle 1 の番号で書く

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(f144021d) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | shell 1(環境による。下の注) | 319 s、rc 0 |
| `go test ./... -count=1 -coverprofile` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 79 s(`internal/cli` 77 s)、rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1` | 4 パッケージ | 4 | 0 | 0 | 86 s(`internal/cli` 82 s)、rc 0 |
| 変わったテストを `-count=3`(`internal/org`) | top-level 12(4 本 × 3)、subtest 60 | 72 | 0 | 0 | rc 0 |
| 変わったテストを `-count=3`(`internal/cli`) | top-level 24(8 本 × 3)、subtest 24 | 48 | 0 | 0 | rc 0 |
| mutation(`go test -overlay`。worktree には触れていない) | 26 件 | red 23 件 | 生き残り 3 件(N3、N8、X18) | - | 1 件 12〜82 s |
| 確かめのテスト(overlay で足しただけで commit しない。HEAD、N8、N3 で 1 回ずつ) | 1 本 × 3 | HEAD と N3 で pass | N8 で red | - | 各 1 s 未満 |
| 実バイナリの help(`ralph org`、`org spawn`、`org start`、`org status` の `--help`) | 4 | 4(rc 0) | 0 | 0 | - |

- `-count=3` の正規表現は、`internal/org` が `^(TestOrgSpawn_Reserve_InactiveLeaderChecksMaxOrgs|TestValidateOrgWideCapacity|TestOrgSpawn_Reserve_ExistingLeader|TestOrgSpawn_Reserve_ExistingLeader_Phase2)$`、`internal/cli` が `^(TestOrgSpawn_DeprecatedDriverFlagAlias_HiddenFromHelp|TestOrgStatus_.*|TestOrgStart_Reserve_SameListPassesDifferentListRejected|TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml)$`。help の文言を固定するテストはない(`spawn --help` を打つのは `HiddenFromHelp` だけで、`--leader-driver` の有無しか見ない)。そのため、76d1cf1c の 3 か所は実バイナリの出力で確かめた(Regression checks の 4 行目)
- shell の件数は suite ごとに数えた。cycle 1 の 2 回目の `run-test.sh`(`docs/evidence/verify-2026-10-08-124051.log`)と、39 本すべての pass・fail・skip の数が一致した(`diff` が空)。この差分は shell のテストを変えていない
- Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」で、手元の git が 2.49.0 なので毎回 SKIP になる。suite ごとの数が一致するので、cycle 1 の 2 回の実行にも同じ SKIP があった。付録 A の表の Skipped の 0 は、この 1 件を数えていない。`tests/test-pre-bash-guard.sh` の `SKIP: 0` は集計行なので数えていない
- `run-test.sh` の中の `go test ./...` は、`internal/org` だけを実行し、ほかはキャッシュの結果だった(cycle 1 と同じ)。そのため 2 行目で全パッケージを `-count=1` で流し直した
- 依頼にあった `tests/test-secret-scan.sh` の flake は出なかった。失敗がないので、単独の再実行はしていない
- 実行のあと、worktree と main のチェックアウトの `git status --porcelain` はどちらも空で、main の `.harness/state/org` もできていない

## Coverage

- Statement: `internal/org` 93.1%(cycle 1 は 93.0%)。ほかの 7 パッケージは cycle 1 と同じ(`internal/cli` 85.3%、`internal/config` 92.8%、`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数と、cycle 1 の一覧):

| 関数 | 位置 | Coverage | cycle 1 との違い | 通らない文 |
| --- | --- | --- | --- | --- |
| `validateMaxOrgs` | `envelope.go:107` | 100.0% | 新しい関数 | なし |
| `ValidateOrgWideCapacity` | `envelope.go:92` | 100.0% | 同じ | なし |
| `idempotentRespawn` | `spawn.go:1098` | 90.9% | 87.5% から上がった(足した文はすべて通る) | `scope_reserved` の追記の失敗(`:1110-1112`)。cycle 1 と同じ文 |
| `NormalizeReservePaths` / `normalizeReservePath` | `reserve.go:45` / `:61` | 100.0% / 100.0% | 同じ | なし |
| `reservePathsOverlap` | `reserve.go:90` | 100.0% | 同じ | なし |
| `RunningOrgs` | `reserve.go:233` | 100.0% | 同じ | なし |
| `ActiveReservation` | `reserve.go:256` | 100.0% | 同じ | なし |
| `TotalActiveSeats` | `reserve.go:265` | 100.0% | 同じ | なし |
| `spawnCapacityErr` | `spawn.go:1065` | 100.0% | 同じ | なし |
| `reservationDecision` | `reserve.go:283` | 100.0% | 同じ | なし |
| `reserveAgain` | `verbs.go:1225` | 75.0% | 同じ | すでに予約を持つときの戻りと、追記の失敗 |
| `orgsToDisband` | `verbs_all.go:309` | 100.0% | 同じ | なし |
| `withMainWorktreeOrgLimits` | `cli/org.go:148` | 100.0% | 同じ | なし |
| `MainWorktreeRoot` | `statedir.go:81` | 100.0% | 同じ | なし |

- そのほかの関数: `checkCapacityAndStart` 69.2%、`dryRunSpawn` 93.7%、`autonomousScopeGateErr` 100.0%、`CloseDeferredSelfWorkspace` 100.0%、`newOrgSpawnRuntime` 90.0%、`orgReservation` 75.0%、`printStatusTable` / `printStatusJSON` 100.0%、`config.Load` 92.9%、`config.Default` 100.0% で、どれも cycle 1 と同じ。help を変えた関数は `newOrgCmd` 100.0%、`newOrgSpawnCmd` 95.0%、`newOrgStartCmd` 100.0%、`newOrgStatusCmd` 90.0%(通らないのは `Status` と `orgReservation` のエラーの戻り)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 新しい枝(`spawn.go:1100-1103`)の文はすべて通る。行の coverage からは `if !seat.Active` を外せるかどうかは分からないので、下の N3 で確かめた。tech-debt の行 176 の (b) は `idempotentRespawn` を 87.5% と書くが、今は 90.9% である。通らない文は変わっていない(self-review C2-5 の指摘と同じ)

## Mutation

各 mutation は、対象のファイルの写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off` でパッケージ全体を流した(`-run` で絞っていない)。まず 26 件すべてを `internal/org` で流した。生き残った N3 と N8、それに N13 は、`internal/org` と `internal/cli` の両方で流し直した。worktree のファイルは書き換えていない。

この回のコード(975df92b、afcbc6c2)の mutation:

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| N1 | `idempotentRespawn` が `validateMaxOrgs` を呼ばない(依頼) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1, max_orgs 1」と「phase 2, max_orgs 1」だけ |
| N2 | `validateMaxOrgs` の `!slices.Contains(runningOrgs, orgID)` を外す(依頼。新しい spawn と idempotent の両方に効く) | red | `TestValidateOrgWideCapacity`(「running org at / over max_orgs」)、`MaxOrgs_NewOrgRejectedAtLimit`、`OrgWideLimits_DryRunPredictsTheSameRejection`、`Reserve_InactiveLeaderChecksMaxOrgs`(「org running through another seat」) |
| N2b | idempotent の呼び出しだけ、org_id が `runningOrgs` に合わないようにする(`p.OrgID+"#"` を渡す) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の 3 つの subtest |
| N3 | `if !seat.Active` を外し、いつも max_orgs を判定する(依頼。self-review C2-3) | **生き残り**(等価) | なし。下の T2-2 |
| N3b | 条件を逆にする(`if seat.Active`) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1 / phase 2, max_orgs 1」 |
| N4 | `ValidateOrgWideCapacity` で max_total_seats を max_orgs より先に判定する(依頼) | red | `TestValidateOrgWideCapacity` の「both reached reports max_orgs first」 |
| N5 | `validateMaxOrgs` の `>=` を `>` にする | red | 8 本(`TestValidateOrgWideCapacity`、`ZeroOrgWideLimits_Reject`、`Reserve_InactiveLeaderChecksMaxOrgs` ほか) |
| N6 | 新しい枝の拒否を `SpawnOutcomeFailed` で返す | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「max_orgs 1」の 2 つ |
| N7 | 新しい枝の拒否で `rejected` と receipt を書く(`o.reject`) | red | 同じ 2 つ(最後の記録が `disbanded` のまま、の検査) |
| N8 | `idempotentRespawn` で予約の判定を max_orgs より先にする | **生き残り** | なし。下の T2-1 |
| N9 | 走っている org がないときの `(none)` を空にする | red | `TestValidateOrgWideCapacity`(「zero max_orgs」)、`ZeroOrgWideLimits_Reject` |
| N10 | `ValidateOrgWideCapacity` が org_id の代わりに seat id を渡す | red | `TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`OrgWideLimits_DryRunPredictsTheSameRejection` |
| N11 | `ValidateOrgWideCapacity` だけが `validateMaxOrgs` を呼ばない(新しい spawn の側) | red | 8 本(`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、`ReservationRestored_RiskWindowClearedByRetry` ほか) |
| N12a | Phase 1 の呼び出し(`spawn.go:606`)だけ、座席を Active として渡す | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1, max_orgs 1」 |
| N12b | Phase 2 の呼び出し(`spawn.go:762`)だけ、座席を Active として渡す | red | 同じテストの「phase 2, max_orgs 1」 |
| N13 | (比べるため)新しい spawn の `spawnCapacityErr` で、予約を全 org の上限より先に判定する | red | `TestOrgDisbandAll_ReservationOnlyOrgDisbanded`(max_orgs と重なりの両方に当たる spawn に max_orgs の拒否を期待する、`verbs_all_test.go:588`) |
| N14 | `idempotentRespawn` が `reservationDecision` だけを飛ばす(max_orgs は残す) | red | `Reserve_ExistingLeader` の 3 つの subtest、`Reserve_ExistingLeader_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` の「max_orgs 2」と「through another seat」 |

cycle 1 の mutation の流し直し(依頼の安全側のもの。番号は付録 A と同じ):

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| M1 | max_orgs の判定を外す。判定が `validateMaxOrgs` に移ったので、新しい spawn と idempotent の両方に効く | red | 9 本(`TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、`Reserve_InactiveLeaderChecksMaxOrgs` ほか) |
| M2 | max_total_seats の判定を外す | red | 6 本(`MaxTotalSeats_Boundary`、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded`、`..._ReactivationCanExceedMaxTotalSeats` ほか) |
| M3 | `RunningOrgs` が `rejected` だけの座席の org も数える | red | 6 本(`TestRunningOrgs`、`MaxOrgs_NewOrgRejectedAtLimit` ほか) |
| M5 | ディレクトリどうしの重なりで末尾の `/` を外して比べる | red | `TestReservePathsOverlap`、`TestReservationDecision`、`Reserve_OverlapWithRunningOrgRejected` |
| X23 | `reservationDecision` がほかの org の予約を見ない(重なりの判定がない) | red | 7 本(`ConcurrentReservations_OnlyOneOfOverlappingWins`、`Reserve_ExistingLeader` ほか) |
| M8 | `idempotentRespawn` が `Reserve` の扱いを丸ごと飛ばす | red | `Reserve_ExistingLeader`、`_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` |
| X24 | Phase 2 の idempotent の戻りが `idempotentRespawn` を通らない | red | `Reserve_ExistingLeader_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` の「phase 2」の 2 つ |
| X11 | 立っている leader への予約の拒否で `rejected` を書く | red | `Reserve_ExistingLeader` |
| X18 | `reserveAgain` の「すでに予約を持つなら書かない」を外す | 生き残り(cycle 1 と同じ) | なし。`verbs.go` はこの回変わっていない |

- 依頼の 4 件(N1、N2、N3、N4)のうち、N3 だけが生き残った。cycle 1 の安全側の mutation は、X18 を除いてすべて red のまま。
- N1 を落とすのは新しいテストだけである。新しいテストを戻すと、この回の修正は守られなくなる。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の安全側の判定(max_orgs、重なり、`rejected` だけの org、立っている leader への予約) | 保たれている | M1、M3、M5、X23、M8、X24、X11 が HEAD で red |
| `ValidateOrgWideCapacity` の条件、判定の順、文言(`validateMaxOrgs` への切り出しの前後) | 保たれている | `TestValidateOrgWideCapacity` の 10 行が `-count=3` で通る。判定の順は「both reached reports max_orgs first」が、`(none)` の文言は「zero max_orgs」が固定していて、N4、N9、N10、N11 が red |
| 予約のない再試行は既存の座席を返す(AC14 の後半。新しい枝で拒否したあと) | 保たれている | 新しいテストが、拒否のあとも leader が `spawned` のままで、最後の記録が古い `disbanded` のままであることを見る。`rejected` を書く N7 が red。`rejected` が書かれると座席の最新の状態が変わり、次の再試行が idempotent の戻りに入らなくなる(verify の C2-2 の判断) |
| help の 3 か所(76d1cf1c) | 表示を確かめた | 実バイナリで、`org spawn --help` と `org start --help` に `orgWideLimitsHelp` の段落が 1 回ずつ出る。`org status --help` に `Long` が出る。4 つの help の `--config` の説明が新しい文になる。どれも rc 0。`status` の `reserved: <path>, ...` の形は、`printStatusTable` の `strings.Join(reservation, ", ")` と、`TestOrgStatus_ShowsReservation` の期待(`reserved: docs/x.md, internal/auth/`)に合う。文言の内容が正しいかは /verify の担当 |
| 今ある `max_seats`、scope のゲート、spawn の idempotent | 保たれている | `run-test.sh` と `go test ./... -count=1` がすべて通る |

## Test gaps

この回に新しく見つけたもの。cycle 1 の Test gaps 1〜8 は付録 A にある。1 で足した 2 本のテストは今も通り、2〜8 はこの回の差分に関係しないので変わらない(X18 は流し直しても生き残る)。

- T2-1(N8): idempotent の経路では、max_orgs と重なりのどちらを先に判定するかがテストで固定されていない。古い台帳の動いていない leader が max_orgs に当たり、しかも頼んだ範囲がほかの走っている org の予約に重なるとき、今のコードは max_orgs の拒否を返す。これは新しい org が同じ場合に受け取る拒否と同じで、新しい spawn の側の順は `TestOrgDisbandAll_ReservationOnlyOrgDisbanded` が固定している(N13 が red)。`idempotentRespawn` の順を入れ替えても(N8)、どちらの順でも拒否になり何も記録されないので、変わるのは出てくる理由の文だけである。`idempotentRespawn` の doc と新しいテストのコメントは「the same error a new org gets」と書くが、判定の順を決めた AC はない。依頼に従い、AC の穴とはみなさずテストを足さなかった。overlay だけで足した確かめのテストで、subtest 1 つで塞げることを確かめた。org-b の leader が `internal/` を予約し、max_orgs が 1 で、org-a の leader に古い disband があるとき、org-a の leader の `internal/x.go` の予約に `max_orgs 1 reached` を期待する。このテストは HEAD と N3 で通り、N8 で落ちる(`docs/evidence/` の cycle 2 の節に本文と出力がある)
- T2-2(N3、self-review C2-3): `if !seat.Active` を外しても結果は変わらない。これは等価な mutation で、今の呼び出し元のままではどのテストでも落とせない。2 つの呼び出し元(`spawn.go:606` と `:762`)は、座席を `Roster(events, RosterOptions{})` から取り、同じ `events` を `idempotentRespawn` に渡す。`RunningOrgs` は同じ roster の Active な座席の org を「走っている」に数えるので、Active な座席なら `validateMaxOrgs` は MaxOrgs の値にかかわらず nil を返す。条件で省けるのは `RunningOrgs` を 1 回計算することだけである。N12a と N12b(呼び出し元が動いていない座席を Active として渡す)は red になるので、条件が効くのは座席と `events` が食い違うときに限られる。今の呼び出し元では、そうはならない。C2-3 の読みは、実行でも合っていた。外すかどうかはコードの形の判断で、外すなら `internal/org` の変更なのでパイプラインの再実行になる
- T2-3: help の文言(`orgWideLimitsHelp`、`status` の `Long`、`--config` の説明)を固定するテストはない。ほかの help の文も、`HiddenFromHelp` を除いて固定されていない。この回は実バイナリの出力で表示だけを確かめた
- T2-4: 新しいテストの古い台帳は、`spawned` と `disbanded` の 2 行を手で書いて作っている。古い ralph のバイナリが実際に書いた台帳では確かめていない(verify の Coverage gaps と同じ)。CLI から古い台帳に `ralph org start --reserve` を打つテストもなく、CLI の経路は verify の実バイナリの確認だけである

## Verdict

- Pass: pass。`run-test.sh`(full)は rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`go test ./... -count=1`、`-race`、変わったテストの `-count=3` も 0 failed。mutation は 26 件のうち 23 件が red。依頼の 4 件では N1、N2、N4 が red で、N3 は等価なので生き残った(T2-2)。cycle 1 の安全側の mutation は、X18 を除いてすべて red のまま。N8 の生き残り(T2-1)は理由の文の違いだけで、AC の穴ではない。AC1 と AC14 の新しい枝を、Phase 1 と Phase 2 の両方でテストの実行で確かめた
- Fail: なし
- Blocked: なし。T2-1〜T2-4 と、付録 A の Test gaps 2〜8 は merge を止めない

## 付録 A: cycle 1 の test レポート(fff1ec23 時点の原文)

cycle 1 の本文を、見出しを 1 段下げただけでそのまま残す(`## Verdict` が 2 つにならないようにするため)。tech-debt の行 172 と 176 は、ここの Test gaps の番号、Coverage、mutation X18 を指している。`file:line` と coverage の値は fff1ec23 時点のもの(`idempotentRespawn` の 87.5% は、この回は 90.9%)。

- Date: 2026-10-08(JST。実行の記録は UTC の 2026-10-08 11:54〜12:49)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch feat/org-limits-reserve と base 51855166 の差分。コードの commit は 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI と `statedir.go`)、8dd19634(文言)。テスト開始時の HEAD は fbb04f83。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC11、AC13〜AC15 を見た。AC12 は文書の基準なので /verify の担当
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-115436.log`(テスト追加前)と `docs/evidence/verify-2026-10-08-124051.log`(追加後)

### Test execution

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

### Coverage

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

### Mutation

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

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 今ある `max_seats` の判定と、その競合のテスト | 保たれている | `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` が `-count=3` と `-race` で通る。`spawnCapacityErr` は `ValidateSpawnCapacity` を最初に呼ぶ |
| scope のゲート(AC-2b) | 保たれている | `--scope` も `--reserve` もない autonomous の spawn は今どおり拒否される(`Reserve_SatisfiesAutonomousScopeGate` の 2 つ目の検査)。X6 で red |
| spawn の idempotent(立っている座席は上限の判定の前に返す) | 保たれている | `MaxTotalSeats_Boundary` の上限ちょうどでの respawn がイベントを増やさない。Phase 2 の idempotent の既存テスト(`StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent`)も通る |
| 2 段目の自分の close の補償(workspace と pane) | 保たれている | 既存の `CloseDeferredSelfWorkspace_*` と `CloseDeferredSelfPane_*` がすべて通る。予約のない org では補償の記録が増えない(`..._ReservationRestored` の「no reservation, nothing restored」) |
| `disband --all` の対象 | 保たれている | `TestOrgsToDisband_Definition` の既存の場合がすべて通り、予約だけの org が加わった。M10 で red |

### Test gaps

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

### Verdict

- Pass: pass。`run-test.sh`(full)は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`-race`、新しいテストの `-count=3`、`go test ./... -count=1` も 0 failed。mutation は依頼の 14 種(16 件)がすべて red で、追加の 24 件のうち X24 はテストを足して red になり、X18 だけが生き残った。AC1〜AC11 と AC13〜AC15 をテストの実行で確かめた
- Fail: なし
- Blocked: なし。Test gaps 2〜8 は merge を止めない
