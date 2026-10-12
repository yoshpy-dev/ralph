# Test report: org-watch-gate

- Date: 2026-10-12(cycle 2。JST 18:06〜19:05)
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み、digest de5ca6b7e6f0)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 2(`cycle-count.json` は 2、上限 2)。cycle 1 の test(a0149a10)が実機で見つけた後退の直し(fdfb5b23)と、self-review cycle 2 の C2-1〜C2-3 の直し(3fb0fd1b)のあとの実行。cycle 1 の結果は末尾の「付録」に残してある
- Scope: HEAD 755e082d(base dc41c4f5)。cycle 1 の test のあとにコードを変えたのは fdfb5b23 と 3fb0fd1b で、`internal/cli/org_watch.go`、`internal/cli/org.go`、`internal/org/watchalive.go`(`DefaultWatchLockWait` を公開)、`internal/org/watchgate.go`(`CommandWithStateDir`)、`internal/org/watch.go`(コメント)の 5 ファイル(+98/-39)と、テスト 3 本(`git diff --shortstat a0149a10..HEAD -- '*_test.go'` は +572/-2)。behavioral test だけを実行した(静的解析は /verify の担当で、cycle 2 の verify は pass)。依頼のとおり、`run-test.sh` と `go test`(`-race` を含む)、`go test -overlay` の mutation(新しい枝と cycle 1 の生き残り 6 件)、後退の手順と base との対照の実機確認を行った
- Evidence: `docs/evidence/test-2026-10-12-org-watch-gate-c2.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore` で確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-12-090609.log`(時刻は UTC)。実機確認は `docs/evidence/org-watch-gate-live-2026-10-12.md` の「cycle 2」の節(コミットする。コードブロックは生の出力と byte 単位で同じ)
- テストは足していない。mutant の写しと probe のテストは scratchpad の `owg-test/c2/` に置き、`go test -overlay` で使った。worktree で書いたのは、このレポート、実機の記録、insight event だけ

## Latest run: cycle 2(HEAD 755e082d)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 40 本(PASS の行 4,081)、Go 8 パッケージ | すべて | 0 | shell 3 | 11 分 50 秒、rc 0 |
| `go test ./... -count=1 -v -coverprofile` | top-level 1,444、subtest 1,584 | top-level 1,441、subtest 1,584 | 0 | 3 | 219 s(cli 212.1 s、org 60.1 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 154 s(org 43.7 s、cli 148.2 s)、rc 0 |
| `go test -race -count=5 -run '<下の注>' -v`(org と cli) | top-level 1,020(204 本 × 5)、subtest 1,070 | すべて | 0 | 0 | 271 s、rc 0。`DATA RACE` は 0 件 |
| mutation(`go test -overlay`、HEAD のテスト) | 32 件 | red 29 件 | 生き残り 3 件(下の節) | - | 1 件 33〜186 s、合わせて 28 分 |
| probe のテスト(`-overlay` で足す 1 本、HEAD で `-count=3`) | 3 | 3 | 0 | 0 | 2 s |
| 実機確認(macOS、HEAD・base・cycle 1 の HEAD のビルド) | 下の「実機確認」 | すべて期待どおり | 0 | - | 84 s |

- shell の件数は cycle 1 と同じ数え方(集計の行を除く)。suite ごとの PASS・FAIL・SKIP の数は cycle 1 と `diff` で一致した。Skipped の 3 件も cycle 1 と同じ
- Go の top-level は cycle 1(1,426 本 PASS と 3 本 SKIP)から 15 本増え、消えたテストはない(名前の一覧を `comm` で比べた)。増えた 15 本は、cycle 1 の probe 5 本を取り込んだ 8 本(`TestOrgSpawn_WatchGate_TheProductionSettleIsTwoSeconds`、`TestOrgWatchDetach_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`、`TestOrgStart_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`、`TestOrgWatchDetach_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`、`TestOrgStart_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`、`TestOrgWatchDetach_TheFirstProbeJudgesAMarkWithoutDeadmanMinutesByTheCallersConfig`、`TestOrgWatchDetach_ALaunchThatFails_IsTheErrorAfterTheDetachPrefix`、`TestOrgStart_ALaunchThatFails_IsRefusedWithThePrefixAndNothingStarted`)と、後退の直しの 5 本(`TestOrgWatchDetach_AnotherWatchRuns_` の 4 本と `TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited`)と、C2-3 の 2 本(`TestOrgWatchDetach_TheCommandsRepeatAnExplicitStateDir`、`TestOrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir`)。テストの `-` の行は、案内の文に `--state-dir` の部分が入った期待の 2 行だけ
- `-race -count=5` の `-run` は cycle 1 の式に `OrgStart_` を足したもの: `Gate|Start.*Watch|Detach|Probe|Alive|Stopping|Once|OnLocked|Send|RenderRolePrompt_Leader_Watch|StatusCmd_OrgWatch|OrgWatch_|RunWatchAll_|OrgStart_`。この PR が足した 61 本(cycle 1 の 46 本と cycle 2 の 15 本)がすべて 204 本の中にあることを確かめた
- ほかのセッションの負荷で、load average は実行中に 7〜288 まで動いた(`run-test.sh` の終わりは 288、`go test ./...` の終わりは 274)。それでも落ちたテストはなく、cycle 1 で名前の出た 2 本(`TestCheckScaffoldIntegrity_…`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)も落ちなかった

### Coverage

- Statement: `internal/cli` 86.6%、`internal/org` 94.3%(cycle 1 は 86.5% と 94.3%)。ほかの 6 パッケージは cycle 1 と同じ
- Function(cycle 2 で変わった関数): `ensureOrgWatch`、`runOrgWatchDetach`、`orgWatchDetachRefusal`、`wireOrgStartWatchGate`、`CommandWithStateDir`、`watchGateCommand` は 100%。`wireOrgWatchGate` は 85.7% から 100% になった(本番の settle の既定の枝を、取り込まれた `TestOrgSpawn_WatchGate_TheProductionSettleIsTwoSeconds` が通る)。`launchAndAwaitOrgWatch` は 92.0% から 97.4% になり、残るのは子の要求(`orgWatchChildArgs` の `filepath.Abs`)が失敗する枝だけ
- Branch: Go の標準ツールには branch coverage がないので測っていない

### cycle 1 の「見つけたこと」は直ったか

直った。cycle 1 で後退を見つけた手順(同じ新しい台帳に `--detach` を 8 本同時に打ち、すべて返った直後に `--stop`)を HEAD のビルドで 3 回打った。

| | HEAD 755e082d(3 回) | base dc41c4f5(1 回) | cycle 1 の HEAD ae349cc4(1 回) |
| --- | --- | --- | --- |
| 8 本の rc | 3 回とも 8 本すべて 0、`started` は 1 本 | 0 が 6 本、1 が 2 本 | 8 本すべて 0 |
| すべて返った時点の見張りのプロセス | 3 回とも 1 | 1 | 7 |
| すぐあとの `--stop` | 3 回とも rc 0、0.4〜1.2 s | rc 0、0.7 s | rc 1、30.1 s で時間切れ |
| `--stop` のあとのプロセス | 3 回とも 0 | 0 | 1 |
| `watch.log` の banner | 1 行(ロックを取った見張りは 1 つ) | 3 行(base は banner をロックの前に出す) | 2 行(2 つの見張りが順にロックを取った) |

cycle 1 の HEAD のビルドでは同じ手順が今も後退を再現するので、手順が鈍ったのではなく、直しが効いている。HEAD は base の `--stop` の動きを保ったまま、tech-debt 206 の直し(8 本とも rc 0)も保っている。その代わりに、子がロックの取り合いに負けた `--detach` は、子のロックの待ち(2 秒)が終わるまで待ってから返る(実機で 2.3〜2.6 s。`started` の本と子を起動しなかった本は 0.4〜0.5 s)。テストでは、子の終わりを待つことを `TestOrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited` と `TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited` が、待たない形の mutant K1(直す前の形)で落ちることで固定している(下の表)。

残る穴は計画の Progress と self-review の C2-4 のとおり、`--detach` か start が返る前に、別の端末から `--stop` を打った場合である。ロックを待つ見張りがあとを取ることは base にもあり(cycle 1 の付録の「base との比較」の 1 行目。foreground の `watch --all` も同じ)、この PR では変わらない。

### AC とテストの対応(cycle 1 からの変更)

cycle 1 の表(付録)から変わった行だけを書く。ほかの AC は、コードもテストも cycle 1 のあと変わっていないので、付録の対応がそのまま使える。テストはすべて HEAD で通った。

| AC | 固定するテスト(cycle 2 で足したもの) | 補い | 穴 |
| --- | --- | --- | --- |
| AC4 running にならない子の回収、起動の失敗 | `TestOrgStart_ALaunchThatFails_IsRefusedWithThePrefixAndNothingStarted`(`internal/cli/org_start_watch_test.go:257`)、`TestOrgWatchDetach_ALaunchThatFails_IsTheErrorAfterTheDetachPrefix`(`internal/cli/org_watch_test.go:446`)、`TestOrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir`(`org_start_watch_test.go:278`) | - | なし(cycle 1 の E8 は閉じた) |
| AC5 子が先に終わってもほかの見張りが running なら成功 | `TestOrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited`(`org_watch_test.go:569`)、`…_AChildThatDoesNotExitInTimeIsStopped`(`:608`)、`…_AStoppedChildIsJudgedByAProbeAfterTheStop`(`:652`。止めたあとの判定 3 通り)、`…_AChildThatTakesTheLockMeanwhileIsStarted`(`:726`)、`TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited`(`org_start_watch_test.go:346`) | 実機 cycle 2(8 本の同時の `--detach` のあとの `--stop` が 3 回とも効く) | 本番の `orgWatchChildExitWait`(3 秒)を固定するテストはない(mutant K8。probe 6 が閉じる) |
| AC7 印の `deadman_minutes` | `TestOrgWatchDetach_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`(`org_watch_test.go:1308`)、`TestOrgWatchDetach_TheFirstProbeJudgesAMarkWithoutDeadmanMinutesByTheCallersConfig`(`:1326`)、`TestOrgStart_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`(`org_start_watch_test.go:588`) | - | なし(cycle 1 の E7・E7c・E7d は閉じた) |
| AC15 最初の印の前の stale を待つ | `TestOrgSpawn_WatchGate_TheProductionSettleIsTwoSeconds`(`internal/cli/org_watch_gate_test.go:248`)、`TestOrgWatchDetach_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`(`org_watch_test.go:956`)、`TestOrgStart_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`(`org_start_watch_test.go:550`) | - | なし(cycle 1 の S4 と E4 は閉じた) |
| C2-3 案内の `--state-dir`(AC1 と AC4 の文) | `TestOrgWatchDetach_TheCommandsRepeatAnExplicitStateDir`(`org_watch_test.go:871`。stale、stopping、ロックを放さない子 × `--state-dir` の有無)、`TestOrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir` | 実機 cycle 2(stale の案内に `--stop --state-dir '<台帳>'` が付き、`RALPH_ORG_STATE_DIR` で指したときは付かない。stopping の案内に `--detach --state-dir '<台帳>'` が付く) | なし |

### Mutation

`scratchpad/owg-test/c2/mut2.py` は cycle 1 の `mut.py` の写しに、cycle 2 の mutant(K1〜K10)を足したもの。書き換える文字列がファイルに 1 回だけあることを当てる前に確かめた。cycle 1 の mutant は、cycle 2 で変わった関数に当たるもの(E1、E1b、E2、E5、E6、L1、X1、X2、X3、T10)を走らせ直した。E2 は書き換え先の行がなくなったので、同じ意味(自分の子の pid のときだけ成功)の新しい形にし、T10 は新しい引数に合わせた。cycle 2 で変わっていないコードの mutant(G、W、P、D、M、B、S1、S3、N、T1〜T9、C、ST)は走らせ直していない。それを落としたテストは cycle 2 で消えも弱まりもしていない(テストの `-` の行は案内の文の 2 行だけ)。32 件ともコンパイルが通った。CLI の mutant は `internal/cli` の全体を、K10 は `internal/org` を走らせた。3 本に分けて並べて走らせた。

依頼の項目と、その mutant:

| 依頼 | mutant | 結果 |
| --- | --- | --- |
| 子の終了を待たない(直す前の形) | K1 | red |
| 待ちの期限で子を止めない | K2 | red |
| 待たずに止める | K3 | red |
| 子がロックを取った印を started にしない | K4a(launched を false に)、K4b(その枝を消す) | どちらも red |
| 止めたあとに判定し直さない(C2-1 の前の形) | K5 | red |
| 止めたあとに見張りがいなくなったときの文 | K6 | red |
| `--state-dir` を案内に付けない | K7a(`--detach` の全体)、K7b(start の起動の失敗)、K7c(stale の `--stop`)、K7d(stopping と `--once` の `--detach`)、K7e(`ralph status`)、K10(`CommandWithStateDir` が引用しない) | すべて red |
| cycle 1 の生き残り 6 件 | S4、E4、E7、E7c、E7d、E8 | すべて red(取り込まれたテストが落とす) |

全体は 32 件で、29 件が red、3 件(T2、K8、K9)が生き残った。T2 と K9 は等価、K8 は本当の穴で probe 6 が落とす。「落としたテスト」は `Test` を省いた。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| K1 | ほかの見張りが受け付けたら、子の終わりを待たずに返る(直す前の形) | red | `OrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited`、`OrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited` ほか 3 本 |
| K2 | 子の終わりの待ちの期限で、子を止めない | red | `OrgWatchDetach_AnotherWatchRuns_AChildThatDoesNotExitInTimeIsStopped`、`OrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop` |
| K3 | 子の終わりを待たずに止める(期限を今に) | red | 4 本(`OrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited` ほか) |
| K4a | 子の pid の印を launched と報告しない | red | 7 本(`OrgWatchDetach_AnotherWatchRuns_AChildThatTakesTheLockMeanwhileIsStarted`、`OrgStart_LaunchesTheWatchWhenNoneRuns` ほか) |
| K4b | 子の pid の印の枝を消す(ほかの見張りとして扱う) | red | 8 本(同上ほか) |
| K5 | 止めたあと、止める前の判定の見張りを返す(C2-1 の前の形) | red | `OrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop` |
| K6 | 止めたあとに見張りがいなくなったときの文を出さない(`exited before it was running` になる) | red | 同上 |
| K7a | `--detach` が案内に `--state-dir` を付けない | red | `OrgWatchDetach_TheCommandsRepeatAnExplicitStateDir` ほか 2 本 |
| K7b | start の起動の失敗の文に `--state-dir` を付けない | red | `OrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir` |
| K7c | stale の `--detach` の拒否の `--stop` に付けない | red | `OrgWatchDetach_TheCommandsRepeatAnExplicitStateDir` |
| K7d | stopping と `--once` の拒否の `--detach` に付けない | red | 同上ほか 2 本 |
| K7e | ロックが放されないときの `ralph status` に付けない | red | 同上と `OrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir` |
| K8 | `orgWatchChildExitWait` を 0 に(本番の値) | **生き残り**(cli の全体) | (probe 6 が落とす) |
| K9 | 止めたあとも、閉じた `exited` の channel を待ちに使い続ける | **生き残り**(等価) | - |
| K10 | `CommandWithStateDir` が `--state-dir` を引用しない | red | `WatchGateError_TheCommandsRepeatHintStateDir`(org) |
| S4 | `DefaultWatchGateSettle` を 0 に | red | `OrgSpawn_WatchGate_TheProductionSettleIsTwoSeconds` |
| E4 | stale を待つ間に持ち主が消えたら、起動せずに断る | red | `OrgWatchDetach_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`、`OrgStart_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch` |
| E7 | `ensureOrgWatch` の最初の判定だけが呼んだ側の deadman を見ない | red | `OrgWatchDetach_TheFirstProbeJudgesAMarkWithoutDeadmanMinutesByTheCallersConfig` |
| E7c | `--detach` の `ensureOrgWatch` が呼んだ側の deadman を見ない | red | `OrgWatchDetach_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig` ほか 1 本 |
| E7d | start の `ensureOrgWatch` が呼んだ側の deadman を見ない | red | `OrgStart_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig` |
| E8 | 起動の失敗を接頭辞なしで返す | red | `OrgWatchDetach_ALaunchThatFails_IsTheErrorAfterTheDetachPrefix`、`OrgStart_ALaunchThatFails_IsRefusedWithThePrefixAndNothingStarted` |
| T2 | start の検査が `ensureOrgWatch` のあとに検査をもう一度呼ばない | **生き残り**(等価、cycle 1 と同じ) | - |
| T10 | start の拒否を `WatchGateError` でなく `--detach` の文に | red | 4 本(`OrgStart_WatchGate_RefusesAStaleWatch` ほか) |
| E1 | stale を待たず、not_running と同じく起動する | red | 7 本(`OrgStart_WaitsForAWatchThatWritesItsFirstMarkLate` ほか) |
| E1b | stale を待たずに断る | red | 6 本(同上ほか) |
| E2 | 自分の子の pid のときだけ成功とする(子が終わったあとのほかの見張りを断る) | red | 7 本(`OrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` ほか) |
| E5 | 子が終わったら、stale の間も待たずに抜ける | red | `OrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning` |
| E6 | 期限で子を止めない | red | `OrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused`、`OrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| X3 | stale を待つ間に判定し直さない | red | 4 本(`OrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning` ほか) |
| L1 | 子が終わったあと、ロックを持つ受け付けない見張りへの拒否の枝を消す | red | 3 本(`OrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop` を含む) |
| X1 | `--detach` がどの見張りにも `started` と言う | red | 8 本(`OrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited` を含む) |
| X2 | `--detach` の `--once` への拒否を stopping の文に | red | `OrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused`、`OrgWatchDetach_AStoppingOrOnceWatch_StartsNothing` |

#### 生き残った mutant

- **K8**: 本番の `orgWatchChildExitWait`(`org.DefaultWatchLockWait` + 1 秒 = 3 秒)を固定するテストはない。子の終わりを待つテストはどれも値を自分で設定する(3 s、200 ms、100 ms)。0 にすると、ほかの見張りが受け付けた次の判定で子を止める。本番の `stop` は子が終わるまで待つので、返るまでの時間はほぼ変わらないが、待つ間にほかの見張りが終わって子がロックを取る場合に、`started` ではなく「止めた子のあとに見張りがいない」の拒否になる。probe 6 は値を設定せずに、300 ms 後に終わる子を止めずに待つことと、値が子のロックの待ち以上であることを見る(HEAD で `-count=3` が通り、K8 で落ちる)。実機の 2.3〜2.6 s は子が自分で終わった時間で、この値の確かめにはならない
- **K9(等価)**: 止めたあとに `exited` を nil にしないと、次の待ちが閉じた channel で 1 回だけ待たずに抜ける。本番の `stop` も偽の子の `stop` も、`exited` が閉じてから返るので、違いは判定が 1 回早まることだけ
- **T2(等価)**: cycle 1 と同じ(付録)

#### Probe のテスト

cycle 1 の probe 1〜5(付録)は fdfb5b23 で 8 本のテストとして取り込まれ、どれも cycle 1 の生き残りを落とした(上の表の S4、E4、E7、E7c、E7d、E8)。続きの番号で 1 本を書いた。scratchpad の `owg-test/c2/probe/zz_probe_owg_c2_test.go`(`internal/cli`)にあり、`go test -overlay`(`c2/probe-head.json`)で足した。コミットしていない。

| # | probe | 落とす mutant | 確かめたこと |
| --- | --- | --- | --- |
| 6 | `TestProbeOWG2_TheProductionChildExitWaitOutlastsTheChildsLockWait` | K8 | `orgWatchChildExitWait` を設定せず(本番の 3 秒のまま)、ほかの見張りが動いている台帳で、300 ms 後に自分で終わる子を止めずに待ち、`already running` を返す。あわせて値が `org.DefaultWatchLockWait` 以上であることを見る |

依頼のとおり、テストは足していない。足すかは orchestrator が決める。

### 実機確認

`docs/evidence/org-watch-gate-live-2026-10-12.md` の「cycle 2」の節に、スクリプト(`live2.sh`)と出力を byte 単位でそのまま貼った(貼ったあとでコードブロックを取り出して元のファイルと比べ、cycle 1 の 5 つと cycle 2 の 2 つがすべて一致した)。隔離は cycle 1 と同じで、`env -i`、scratch の HOME と XDG、`osascript` を外した PATH(herdr と agmsg はない)、台帳はすべて repo の外の `--state-dir`。利用者の herdr サーバーと `~/.config/herdr` には触れていない。

| # | 確認 | 結果 |
| --- | --- | --- |
| 1 | 後退の手順(8 本の同時の `--detach` のあと、すぐ `--stop`)を HEAD で 3 回 | 上の「cycle 1 の『見つけたこと』は直ったか」の表のとおり、3 回とも `--stop` が rc 0 で、何も残らない |
| 2 | 対照: base dc41c4f5 と cycle 1 の HEAD ae349cc4 で 1 回ずつ | base は `--stop` が効き(rc 1 の `--detach` が 2 本)、ae349cc4 は後退を再現する(7 プロセス、`--stop` が 30 s で時間切れ、1 つ残る) |
| 3 | `--detach` が返るまでの時間 | 新しい台帳で 0.3 s(`started`)、動いている台帳への 2 回目は 0.1 s(`already running`、子を起動しない)。8 本の同時のうち子が負けた本は 2.3〜2.6 s |
| 4 | 古い見張りへの `--detach --state-dir <台帳>` | rc 1、10.1 s。案内は `stop it with ralph org watch --stop --state-dir '<台帳>'`。同じ台帳を `RALPH_ORG_STATE_DIR` で指したときは `stop it with ralph org watch --stop` で、`--state-dir` は付かない |
| 5 | 止まりかけの印(手で置いたもの)への `--detach --state-dir <台帳>` | rc 1、0.1 s。案内は `run ralph org watch --all --detach --state-dir '<台帳>' again once it has ended` |
| 6 | 終わり | どの台帳にも `--stop` を打ったあと、3 つのバイナリのどれについても `pgrep -fl '<バイナリ> org watch'` は何も返さず rc 1 |

本物の見張りの後始末の間(stopping)と、C2-1 の流れ(子を止めている間に、ほかの見張りが終わるか替わる)は、実機では作れなかった。どちらもテスト(偽の子と手で置いた印)だけが見ている。

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | テストの失敗はない。load average が 200 を超えた時間帯の実行でも落ちなかった | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の「見つけたこと」: 同時の `--detach` のすぐあとの `--stop` が効かず、別の見張りが残る | 直った | 実機 cycle 2 の 1 と 2(HEAD 3 回とも効く、ae349cc4 は再現する)、mutant K1 を落とすテスト 5 本 |
| tech-debt 206: 同時の `--detach` の 1〜2 本が rc 1 | 直ったまま | 実機 cycle 2 の 1(HEAD は 3 回とも 8 本すべて rc 0、base は 2 本が rc 1) |
| self-review C2-1: 子を止める前の判定の見張りを返す | 直った | `TestOrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop`、mutant K5・K6 |
| self-review C2-3: `--detach` の拒否と起動の失敗の案内に `--state-dir` がない | 直った | `TestOrgWatchDetach_TheCommandsRepeatAnExplicitStateDir`、mutant K7a〜K7e、実機 cycle 2 の 4 と 5 |
| cycle 1 の生き残り 6 件(S4、E4、E7、E7c、E7d、E8) | テストが落とす | 上の Mutation の表 |
| cycle 1 で確かめた動き(spawn と TASK の拒否、start の起動、stale の待ち、`--once` と stopping への拒否など) | 変わらない | 名前のまま全テストが通る。`-race -count=5` の 204 本 |

### Test gaps

- `--detach` か start が返る前に、別の端末から `--stop` を打つと、ロックを待つ子があとを取りうる(base にもある。foreground の `watch --all` も同じ)。計画と self-review の C2-4 のとおり tech-debt に書く予定のもので、テストも実機の確かめもない
- 本番の `orgWatchChildExitWait` の値(mutant K8)。probe 6 が閉じる(コミットしていない)
- 子がロックの取り合いに負けた `--detach` と start は、子が終わるまでの約 2 秒(本番では最大 3 秒と `stop` の時間)遅れて返る。仕様の上限はないが、`ralph org start` を同時に打つ使い方では毎回その分だけ待つ
- C2-1 の流れ(子を止めている間に、ほかの見張りが終わる、止まりかける、替わる)は偽の子のテストだけで、実機では見ていない
- cycle 1 の Test gaps のうち、本物の後始末の間の拒否、AC8 の座席の `wait` と inbox の動詞、AC6 の main worktree の台帳での実機の起動、Linux、herdr のある実機は、そのまま残る(付録)

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail(top-level 1,441 本が PASS、cycle 1 より 15 本多く、消えたテストはない)。`-race`(org と cli)と、この PR のテスト 61 本を含む 204 本の `-race -count=5` も 0 fail で、DATA RACE は 0 件。依頼の mutant(新しい枝の K1〜K7e と cycle 1 の生き残り 6 件)はすべて落ちた。32 件のうち 29 件が red で、生き残りは等価の 2 件と、probe 6 が閉じる K8 の 1 件。cycle 1 で見つけた後退は直った: 同じ手順で HEAD は 3 回とも `--stop` が効いて何も残らず、cycle 1 の HEAD のビルドでは今も再現し、base とも合う。案内の `--state-dir` も実機で確かめた
- Fail: 0
- Blocked: なし。K8 の probe を足すかと、Test gaps の残る穴(返る前の別の端末からの `--stop`)を tech-debt に書くのは orchestrator が決める

## 付録: cycle 1 の結果(HEAD ae349cc4 の時点)

cycle 1 の header(Date から、テストを足していない旨の行まで)と本文を、そのとき書いたまま残す。見出しは 1 段下げて「(cycle 1)」を付け、判定の見出しは `## Verdict` から「判定(cycle 1)」に変えた。cycle 1 の「見つけたこと」は cycle 2 で直った(上の節)。

- Date: 2026-10-12(JST 14:10〜16:50。15:13 から 16:22 まで、API の接続エラーで中断した)
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み、digest de5ca6b7e6f0)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1(上限 2)
- Scope: HEAD ae349cc4(base dc41c4f5)。behavioral test だけを実行した(静的解析は /verify の担当で、`docs/reports/verify-2026-10-12-org-watch-gate.md` が pass)。依頼のとおり、`run-test.sh` と `go test`(`-race` を含む)、AC1〜AC15 とテストの対応、`go test -overlay` の mutation、verify が「テストがない」とした `ensureOrgWatch` の分岐の probe、ビルドしたバイナリでの実機確認を行った
- Evidence: `docs/evidence/test-2026-10-12-org-watch-gate.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore` で確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-12-051033.log`(時刻は UTC)。実機確認は `docs/evidence/org-watch-gate-live-2026-10-12.md`(コミットする。コードブロックは生の出力と byte 単位で同じ)
- テストは足していない。mutant の写しと probe のテストは scratchpad の `owg-test/` に置き、`go test -overlay` で使った。worktree で書いたのは、このレポート、実機の記録、insight event だけ

### Test execution(cycle 1)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 40 本(PASS の行 4,081)、Go 8 パッケージ | すべて | 0 | shell 3(下の注) | 8 分 22 秒、rc 0 |
| `go test ./... -count=1 -v -coverprofile` | top-level 1,429、subtest 1,569 | top-level 1,426、subtest 1,569 | 0 | 3 | 132 s(cli 128.9 s、org 32.8 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 142 s(org 35.7 s、cli 138.0 s)、rc 0 |
| `go test -race -count=5 -run '<下の注>' -v`(org と cli) | top-level 870(174 本 × 5)、subtest 930 | すべて | 0 | 0 | 185 s、rc 0。`DATA RACE` は 0 件 |
| mutation(`go test -overlay`、HEAD のテスト) | 64 件 | red 57 件 | 生き残り 7 件(下の節) | - | 1 件 26〜229 s |
| probe のテスト(`-overlay` で足す 5 本、HEAD で `-count=3`) | 15 | 15 | 0 | 0 | 18 s |
| 実機確認(macOS、ビルドしたバイナリ) | 下の「実機確認」 | 期待どおり(7 の `--stop` を除く) | 見つけたこと 1 件 | - | 3 分(base との比較は別に 3 分) |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた。`PASS: 29` と `PASS: 47 / 47` の形の集計の行と、`SKIP: 0` は除いた。40 本すべてが `OK` で終わり、FAIL の行は 0。前の PR の cycle 2(3,962)との差 119 は `tests/test-pre-bash-guard.sh` だけで、base に入った guard の PR の行である。このブランチは `tests/` と `scripts/` を変えていない
- Skipped の shell 3 件は、`tests/test-lib-json.sh`、`tests/test-pre-bash-guard.sh`、`tests/test-secret-scan-branch.sh` の環境による 1 件ずつで、前の cycle と同じ
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し、ほかはキャッシュの結果だった。2 行目で全パッケージを `-count=1` で流し直した
- Go の top-level は base(dc41c4f5 の `func Test`、`TestMain` を除いて 1,384 本)から 46 本増え、1 本が名前を変えた(`TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` が `…SaysStopping` に。AC10 が許す変更)。名前の一覧を `comm` で比べ、ほかに消えたテストがないことを確かめた。Skipped の 3 件は前からある `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` と `internal/scaffold` の 2 件
- `-race -count=5` の `-run` は、依頼の `Gate|WatchGate|Start.*Watch|Detach|Probe|Alive|Stopping|Once|OnLocked|Send` に `RenderRolePrompt_Leader_Watch|StatusCmd_OrgWatch|OrgWatch_|RunWatchAll_` を足した。依頼の式は、増えた 46 本のうち `TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry` と `TestStatusCmd_OrgWatch…` の 2 本に当たらなかったため。46 本すべてが 174 本の中にあることを確かめた
- 依頼にあった 2 本(`TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)は、`run-test.sh`、`-count=1` の全体、`-race` のどれでも落ちなかったので、単独の再実行はしていない。全体の実行の終わりの load average は 55(ほかのセッションの負荷)

### Coverage(cycle 1)

- Statement: `internal/cli` 86.5%、`internal/org` 94.3%(前の PR の cycle 2 は 86.3% と 94.1%)。ほかは config 93.2%、insights 86.1%、driver 93.1%、protocol 97.9%、scaffold 75.7%、upgrade 91.2% で変わらない
- Function(この PR が足した関数と変えた関数): 100% は `NewWatchGate`、`WatchAcceptsWork`、`checkWatchGate`、`WatchKillHint`、`WatchPIDText`、`watchGateCommand`、`ProbeWatch`、`watchAliveWindow`、`RunWatchAll`、`drainBeforeStop`、`finish`、`ensureOrgWatch`、`runOrgWatchDetach`、`orgWatchDetachRefusal`、`wireOrgStartWatchGate`、`runOrgWatchAll`、`orgWatchLine`、`buildStatusOrgWatchJSON`、`taskNotSentError` の 2 つ。100% でないのは `WatchGateError.Error` 88.9%(どの状態にも当たらないときの最後の行で、届かない)、`wireOrgWatchGate` 85.7%(本番だけが通る settle の既定。mutant S4)、`launchAndAwaitOrgWatch` 92.0%(子の要求と起動が失敗したときの 2 つの枝。mutant E8)、`orgWatchChildArgs` 90.9%(`filepath.Abs` の失敗)、`orgWatchChild` 93.8%(その失敗の伝搬)、`launchOrgWatch` 0%(本番の起動。テストは TestMain で差し替える)、`Send` 96.5%、`Spawn` 91.9%、`StartFeature` 92.9%(どれも前からある枝)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: `ensureOrgWatch` は 100% だが、待つ間に持ち主が消えたあとの起動を固定するテストはなかった(mutant E4)。行が通ることと、その結果が確かめられていることは別

### AC とテストの対応(cycle 1)

テストはすべて HEAD で通った。「実機」は `docs/evidence/org-watch-gate-live-2026-10-12.md` の節の番号、「probe」は下の「Probe のテスト」(コミットしていない)。

| AC | 固定するテスト | 補い | 穴 |
| --- | --- | --- | --- |
| AC1 spawn の拒否 | `TestOrgSpawn_WatchGate_RefusesAWatchThatAcceptsNoWork`(`internal/cli/org_watch_gate_test.go:154`。見張りなし、kill のあとの印、古い印、印なし、stopping、`--once` の 6 通り × 本番と `--dry-run`。manifest・receipt・herdr・agmsg を見る)、`TestOrgSpawn_WatchGate_ARunningWatchPasses`(`:181`)、`TestOrgWatchGate_TheCommandsRepeatAnExplicitStateDir`(`:458`)、`TestOrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing`(`internal/org/watchgate_positions_test.go:37`)、`TestOrgSpawn_WatchGate_RunsAfterTheInputChecksAndBeforeTheManifestRead`(`:82`)、`TestNewWatchGate_PassesOnlyAWatchThatAcceptsWork`(`internal/org/watchgate_test.go:77`)、`TestWatchGateError_TheCommandsRepeatHintStateDir`(`:211`) | 実機 1、3、4、6 | なし |
| AC2 TASK の send だけを断る | `TestOrgSend_WatchGate_RefusesOnlyATask`(cli `org_watch_gate_test.go:289`)、`TestOrgSend_WatchGate_ATaskPassesARunningWatch`(`:368`)、`TestOrgSend_WatchGate_RefusesATaskOnly`(`watchgate_positions_test.go:161`。前後の空白、先頭の空行、小文字、`task`、検証の前に断る不正な TASK) | 実機 1(型付きと `--raw` の TASK が rc 1、QUESTION が rc 0) | なし |
| AC3 start が見張りを起動する | `TestOrgStart_LaunchesTheWatchWhenNoneRuns`(`internal/cli/org_start_watch_test.go:152`。2 つの形。起動は 1 回、stderr の 1 行、200 ms 後の印を待ってから leader、start のあとも動き、2 回目は起動しない) | 実機 3(kill のあとの start が先に見張りを起動し、start のあとも動き続ける) | 実機では herdr がないので、leader が立つところまでは見ていない(テストが見る) |
| AC4 stale の拒否、running にならない子の回収 | `TestOrgStart_WatchGate_RefusesAStaleWatch`(`org_watch_gate_test.go:396`)、`TestOrgStartPlan_WatchGate_RefusesAStaleWatchBeforeTheWorktree`(`:421`)、`TestOrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused`(`org_start_watch_test.go:213`)、`TestStartFeature_WatchGate_ARefusalMakesNoWorktree`(`watchgate_positions_test.go:111`) | 実機 4(start が 10.1 s 待ってから stale の文で rc 1。verify が見ていなかった start の stale) | なし |
| AC5 子が先に終わってもほかの見張りが running なら成功 | `TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`(`internal/cli/org_watch_test.go:442`)、`TestOrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning`(`:467`)、`TestOrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused`(`:497`)、`TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn`(`org_start_watch_test.go:256`)、`TestOrgStart_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefusedAsTheGateRefuses`(`:291`) | 実機 7(同時の `--detach` 8 本が 3 回とも全部 rc 0) | 同時の起動のすぐあとの `--stop` が効かない(下の「見つけたこと」)。テストの偽の子は返った時点で終わっているか、ロックを持ったまま残るので、ロックを待つ子が残る形を通らない |
| AC6 子の作業ディレクトリ、`--config`、環境 | `TestOrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_start_watch_test.go:458`)、`TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_watch_test.go:670`。既定の台帳、main の台帳を名指す `--state-dir`、repo の外)、`TestOrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv`(`:748`) | 実機は repo の外の台帳だけ(打った場所で動く) | main worktree の中の台帳での実機の起動は見ていない |
| AC7 印の `deadman_minutes` | `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`(`internal/org/watch_all_test.go:861`)、`TestProbeWatch_JudgesTheWindowByTheDeadmanMinutesOfTheMark`(`internal/org/watchalive_test.go:267`)、`TestWatchAlive_DeadmanMinutesAndOnceRoundTrip`(`:304`)、`TestOrgSpawn_WatchGate_JudgesTheMarkByTheWatchsDeadmanMinutes`(`org_watch_gate_test.go:248`)、`TestStatusCmd_OrgWatchJudgesTheMarkByTheWatchsOwnDeadmanMinutes`(`internal/cli/status_test.go:862`)、`TestStatusCmd_OrgWatchOfAWatchWithAnotherConfigReadsByItsMark`(`:899`) | 実機 2(印に `deadman_minutes: 10`)、probe 3・4 | `--detach` と start が、値のない印を呼んだ側の設定で読むことはテストがない(mutant E7、E7c、E7d。probe 3・4 が閉じる) |
| AC8 見張りなしで使える動詞 | `TestOrgVerbsThatStartNoWork_WorkWithNoWatch`(`org_watch_gate_test.go:497`。`ralph status`、org status、read、report、escalate、inbox、`wait --inbox`、stop、disband、`stop --all`、`disband --all`) | 実機 1(見張りのない台帳で `ralph status`) | 座席の `wait` と `inbox show/ack/resolve/notify` はテストに入っていない(verify と同じ。配線がないことは grep だけ) |
| AC9 2 つ目の見張りは banner を出さない | `TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`(`org_watch_test.go:191`)、`TestRunWatchAll_CallsOnLockedOnceItHoldsTheLock`(`watch_all_test.go:736`) | 実機 7(E の各台帳の `watch.log` には、ロックを取った見張りの banner だけがあり、断られた子は拒否の 1 行だけ) | なし |
| AC10 既存のテストを書き換えない | base の 1,384 本のうち 1,383 本が名前のまま通り、1 本が AC10 の許す改名。test ファイルの `-` の行は、`org_watch_test.go` と `watch_all_test.go` の 6a の見張りのテストと、起動の差し替え口の型の書き換えだけ(verify の確認と同じ) | `go test ./...` の全体 | なし |
| AC11 leader の雛形 | `TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry`(`internal/org/prompts_test.go:705`) | - | なし |
| AC12 文書 | behavioral test はない(`check-sync.sh` などは /verify の担当) | - | 対象外 |
| AC13 止まりかけの見張り | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysStopping`(`watch_all_test.go:1222`)、`TestProbeWatch_AStoppingMarkIsStoppingOnlyWhileHeldAndFresh`(`watchalive_test.go:224`)、`TestWatchAcceptsWork_OnlyARunningWatchThatIsNotOnce`(`watchgate_test.go:48`)、`gateRefusals` の stopping の行(spawn)、`TestOrgStart_AStoppingOrOnceWatch_IsRefusedAndNothingLaunched`(`org_start_watch_test.go:345`)、`TestOrgWatchDetach_AStoppingOrOnceWatch_StartsNothing`(`org_watch_test.go:603`)、`TestStatusCmd_OrgWatchShowsStoppingAndOnce`(`status_test.go:777`)、`TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(`org_watch_test.go:1060`) | 実機 6 の D(手で置いた stopping の印) | 本物の見張りが後始末をしている間に spawn・TASK・start を打つテストも実機もない(verify と同じ) |
| AC14 `--once` の見張り | `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`、`gateRefusals` の `--once` の行、start と `--detach` の `--once` の行(AC13 と同じ 2 本)、`TestStatusCmd_OrgWatchShowsStoppingAndOnce` | 実機 6 の C(本物の `--once` の見張りが動いている間に、spawn、TASK、`--detach`、start が rc 1、status が `running (--once)`、`--json` が `once: true`。verify が見られなかったもの) | なし |
| AC15 最初の印の前の stale を待つ | `TestOrgStart_WaitsForAWatchThatWritesItsFirstMarkLate`(`org_start_watch_test.go:387`)、`TestOrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning`(`org_watch_test.go:640`)、`TestOrgSpawn_WatchGate_AStaleWatchThatTurnsRunningWithinTheSettlePasses`(`org_watch_gate_test.go:212`)、`TestNewWatchGate_AStaleWatchThatWritesItsMarkWithinTheSettle`(`watchgate_test.go:156`) | 実機 4(spawn は 2.1 s、`--detach` と start は 10 s 待ってから断る)、probe 2(本番の settle が 2 s) | 本番の settle の値(2 s)を固定するテストはない(mutant S4。probe 2 が閉じる) |

### Mutation(cycle 1)

`scratchpad/owg-test/mut.py` は前の PR の `ocw2/mut2.py` と同じ作りで、`-timeout 240s` にし、probe を足す `EXTRA` を加えた。書き換える文字列がファイルに 1 回だけあることを当てる前に確かめ、64 件ともコンパイルが通ることを `CHECK=1` で確かめた(C1 は `slices` を残すため `false &&` の形にした)。org 層の mutant は `internal/org` の全体を走らせ、生き残ったら `internal/cli` の全体も走らせた。CLI の mutant は `internal/cli` の全体を走らせた。3 本に分けて並べて走らせ、そのあと E7c と E7d を 1 本で走らせ直した(1 回目は probe の実行とログのファイル名が重なったため。結果は同じ)。

依頼の項目と、その mutant:

| 依頼 | mutant | 結果 |
| --- | --- | --- |
| Spawn の検査を dry-run の枝のあと、Manifest.Read のあとに | G1a、G1b(ほかに G0 削除、G1c 入力の検査の前) | すべて red |
| StartFeature の検査を消す | G2(ほかに G2b spawnPrecheckErr のあと) | red |
| Send ですべての TYPE を検査する | G3 | red |
| raw の TYPE を読まない | G4 | red |
| `WatchAcceptsWork` が once を見ない | W1 | red |
| `ProbeWatch` が stopping を返さない | P1a(running と読む)、P1b(stale と読む) | どちらも red |
| `drainBeforeStop` が stopping を書かない | D1(ほかに D2 書き直しが running) | red |
| 印の deadman を見ない | P2 | red |
| `NewWatchGate` の settle を 0 に | S1 | red |
| `ensureOrgWatch` が stale で起動する | E1(ほかに E1b 待たずに断る) | red |
| 成功に子の pid を求める | E2 | red |
| HERDR_* を残す | C1 | red |
| 相対の `--config` をそのまま渡す | C2 | red |
| banner をロックの前に出す | B1(CLI)、B2(org 層の `OnLocked` をロックの前に) | どちらも red |
| L1 の枝を消す | L1 | red |
| bda15444 の dry-run の追記のエラーに接頭辞を付けない | N3 | red |

全体は 64 件で、57 件が red、7 件(S4、T2、E4、E7、E7c、E7d、E8)が生き残った。T2 は等価、ほかの 6 件は本当の穴で、どれも probe が落とす。「落としたテスト」は最初に落ちたパッケージのもので、`Test` を省いた。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| G0 | Spawn の検査を消す | red | `OrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing` ほか 2 本 |
| G1a | Spawn の検査を dry-run の枝のあと(本番は Manifest.Read のあと)に | red | `OrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing`、`OrgSpawn_WatchGate_RunsAfterTheInputChecksAndBeforeTheManifestRead` |
| G1b | Spawn の検査を Manifest.Read のあとに(dry-run の枝の前) | red | `OrgSpawn_WatchGate_RunsAfterTheInputChecksAndBeforeTheManifestRead` |
| G1c | Spawn の検査を `checkSpawnInput` の前に | red | 同上 |
| G2 | StartFeature の検査を消す(Spawn の中の検査は worktree のあと) | red | `StartFeature_WatchGate_ARefusalMakesNoWorktree` |
| G2b | StartFeature の検査を `spawnPrecheckErr` のあとに | red | 同上 |
| G3 | Send ですべての TYPE を検査する | red | `OrgSend_WatchGate_RefusesATaskOnly` |
| G4 | raw のメッセージの TYPE を読まない(型付きの TASK だけ検査) | red | 同上 |
| G5 | Send の検査を dry-run の枝のあとに | red | 同上 |
| G6 | TYPE を大文字小文字を無視して比べる | red | 同上 |
| G7 | Send の検査を protocol の検証の前に | red | 同上 |
| W1 | `WatchAcceptsWork` が once を見ない | red | `NewWatchGate_PassesOnlyAWatchThatAcceptsWork`、`WatchAcceptsWork_OnlyARunningWatchThatIsNotOnce` |
| W2 | `WatchAcceptsWork` が印の nil を見ない | red | `WatchAcceptsWork_OnlyARunningWatchThatIsNotOnce` |
| P1a | `ProbeWatch` が新しい stopping の印を running と返す | red | 6 本(`ProbeWatch_AStoppingMarkIsStoppingOnlyWhileHeldAndFresh` ほか) |
| P1b | `ProbeWatch` が stopping の印を stale と返す | red | 6 本(同上) |
| P2 | `ProbeWatch` が印の `deadman_minutes` を見ない | red | `ProbeWatch_JudgesTheWindowByTheDeadmanMinutesOfTheMark`、`RunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce` |
| P3 | 古い stopping の印も stopping と返す | red | `ProbeWatch_AStoppingMarkIsStoppingOnlyWhileHeldAndFresh` |
| D1 | `drainBeforeStop` が stopping を書かない | red | `RunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysStopping` |
| D2 | 後始末の間の書き直しが running を書く | red | 同上 |
| M1 | 印に once を書かない | red | 同上と `RunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce` |
| M2 | 印に `deadman_minutes` を書かない | red | `RunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce` |
| M3 | 1 未満の `deadman_minutes` を 1 にして書く | red | 同上 |
| B2 | `RunWatchAll` が `OnLocked` をロックの前に呼ぶ | red | `RunWatchAll_CallsOnLockedOnceItHoldsTheLock` |
| B3 | `OnLocked` を最初の印のあとに呼ぶ | red | 同上 |
| S1 | `NewWatchGate` の settle を 0 に | red | `NewWatchGate_AStaleWatchThatWritesItsMarkWithinTheSettle`、`NewWatchGate_PassesOnlyAWatchThatAcceptsWork` |
| S3 | `NewWatchGate` が stale 以外の状態でも待つ | red | 同上 |
| S4 | `DefaultWatchGateSettle` を 0 に(本番の既定) | **生き残り**(org と cli の全体) | (probe 2 が落とす) |
| N2 | CLI の send がエラーに `org: send:` をもう一度付ける | red | `OrgSend_AnErrorWithoutTheSendPrefixGetsItOnce`、`OrgSend_WatchGate_RefusesOnlyATask` |
| N3 | dry-run の追記のエラーに接頭辞を付けない(bda15444 を戻す) | red | `OrgSend_DryRun_AppendFails_ErrorStartsWithTheSendPrefix` |
| N4 | `taskNotSentError` が検査の `org: ` を残す | red | `OrgSend_WatchGate_RefusesATaskOnly` |
| N5 | `taskNotSentError` が Unwrap しない | red | 同上 |
| T1 | start が起動の 1 行を stderr に出さない | red | `OrgStart_LaunchesTheWatchWhenNoneRuns`、`OrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv` |
| T2 | start の検査が `ensureOrgWatch` のあとに検査をもう一度呼ばない | **生き残り**(等価) | - |
| T3 | `start <task>` が起動しない検査を使う | red | 7 本(`OrgStart_LaunchesTheWatchWhenNoneRuns` ほか) |
| T4 | `start --plan` が起動しない検査を使う | red | 4 本(同上ほか) |
| T5 | spawn に検査を配線しない | red | 4 本(`OrgSpawn_WatchGate_RefusesAWatchThatAcceptsNoWork` ほか) |
| T6 | send に検査を配線しない | red | `OrgSend_WatchGate_RefusesOnlyATask`、`OrgWatchGate_TheCommandsRepeatAnExplicitStateDir` |
| T8 | 検査の文に `--state-dir` を付けない | red | `OrgWatchGate_TheCommandsRepeatAnExplicitStateDir` |
| T9 | 検査が呼んだ側の `deadman_minutes` を見ない(10 に固定) | red | `OrgSpawn_WatchGate_JudgesTheMarkByTheWatchsDeadmanMinutes` |
| T10 | start の拒否を `WatchGateError` でなく `--detach` の文に | red | 4 本(`OrgStart_WatchGate_RefusesAStaleWatch` ほか) |
| E1 | `ensureOrgWatch` が stale を待たず、not_running と同じく起動する | red | 5 本(`OrgStart_WaitsForAWatchThatWritesItsFirstMarkLate` ほか) |
| E1b | `ensureOrgWatch` が stale を待たずに断る | red | 4 本(同上ほか) |
| E2 | 起動の成功に自分の子の pid を求める | red | `OrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn` ほか 2 本 |
| E4 | stale を待つ間に持ち主が消えたら、起動せずに断る | **生き残り**(cli の全体) | (probe 1 が落とす) |
| E5 | 子が終わったら、stale の間も待たずに抜ける | red | `OrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning` |
| E6 | 期限で子を止めない | red | `OrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused`、`OrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| E7 | `ensureOrgWatch` の最初の判定だけが呼んだ側の deadman を見ない(0、つまり既定の 10) | **生き残り**(cli の全体) | (probe 5 が落とす) |
| E7c | `--detach` の `ensureOrgWatch` がすべての判定で呼んだ側の deadman を見ない | **生き残り**(cli の全体) | (probe 3 の `--detach` が落とす) |
| E7d | start の `ensureOrgWatch` がすべての判定で呼んだ側の deadman を見ない | **生き残り**(cli の全体) | (probe 3 の `start <task>` が落とす) |
| E8 | 起動の失敗を接頭辞なしで返す | **生き残り**(cli の全体) | (probe 4 が落とす) |
| X3 | stale を待つ間に判定し直さない | red | `OrgStart_WaitsForAWatchThatWritesItsFirstMarkLate`、`OrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning` |
| L1 | 子が終わったあと、ロックを持つ受け付けない見張りへの拒否の枝を消す | red | `OrgStart_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefusedAsTheGateRefuses`、`OrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused` |
| C1 | 子の環境に HERDR_* を残す | red | `OrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`、`OrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv` |
| C2 | 相対の `--config` をそのまま渡す | red | 同上 |
| C3 | 子の PWD を新しい作業ディレクトリにしない | red | 同上 |
| C3b | 子の環境に元の PWD も残す | red | 同上 |
| C4 | 子の作業ディレクトリを設定しない | red | 同上 |
| C5 | `startDetachedProcess` が要求の環境を使わない | red | `OrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv` |
| B1 | CLI の banner をロックの前に出す | red | `OrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` |
| X1 | `--detach` がどの見張りにも `started` と言う | red | 4 本(`OrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` ほか) |
| X2 | `--detach` の `--once` への拒否を stopping の文に | red | `OrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused`、`OrgWatchDetach_AStoppingOrOnceWatch_StartsNothing` |
| ST1 | status の stopping の行を stale と出す | red | `StatusCmd_OrgWatchShowsStoppingAndOnce` |
| ST2 | status の `(--once)` を出さない | red | 同上 |
| ST3 | `--json` に once を写さない | red | 同上 |

#### 生き残った mutant

- **T2(等価)**: `ensureOrgWatch` は `WatchAcceptsWork` を見てから成功を返し、そのすぐあとの `check()` も同じ判定をする。違いが出るのは、その間(マイクロ秒)に見張りの状態が変わったときだけ。`start --plan` では、そのあと Spawn の中でもう一度検査が走る。probe 5 本のもとでも生き残った
- **S4**: 本番の settle(`DefaultWatchGateSettle`、2 秒)を固定するテストはない。TestMain が `orgWatchGateSettleOverride` を 50 ms にし、settle を見るテストはどれも値を自分で設定する。実機 4 の 2.1 s が本番の値を示す。probe 2 は差し替え口を 0 にして本物の既定を通し、1.5〜6 s の間に断ることを見る
- **E4**: verify の報告の Coverage gaps の分岐。`ensureOrgWatch` が stale を待つ間に持ち主がロックを放すと、HEAD は起動へ進む(実機 5 の B2・B3 で確かめた)。テストはこの流れを通らない。probe 1 は印のない stale のロックを 200 ms 後に放し、`--detach`(in-process の見張り)と 2 つの形の start が 1 回起動して成功することを見る。E4 では 3 つとも落ちる
- **E7、E7c、E7d**: `--detach` と start の `ensureOrgWatch` は、`deadman_minutes` のない印(前半の版の印)を呼んだ側の設定で読む。テストの印(`runningMark`)はどれも `deadman_minutes` を持たず、設定は既定の 10 なので、0(既定の 10 で読む)と区別がつかない。検査(spawn と send)の側は `TestOrgSpawn_WatchGate_JudgesTheMarkByTheWatchsDeadmanMinutes` が見るが、`ensureOrgWatch` の 2 か所の呼び手(`runOrgWatchDetach` と `wireOrgStartWatchGate`)にはない。probe 3 は設定 30 分で 20 分前の印を running と読むこと(E7c と E7d をそれぞれの subtest で落とす)、probe 5 は設定 5 分で 7 分前の印を stale と読むこと(最初の判定だけを変える E7 を落とす)を見る。前半の版の見張りと台帳を分け合うときだけの話で、計画の Rollout は更新の前に見張りを止めるとしている
- **E8**: 子の要求と起動が失敗したときの 2 つの枝(`org_watch.go:461-467`)は coverage 0。起動の失敗の文が `org: watch --detach: ` と `org: start: the cross-org watch did not start, so no leader was started: ` で始まることを見るテストはない。probe 4 は起動の差し替え口にエラーを返させ、`--detach` と start のエラーの文全体と、start が何も記録しないことを見る

### Probe のテスト(cycle 1)

scratchpad の `owg-test/probe/zz_probe_owg_test.go`(`internal/cli`)に書き、`go test -overlay`(`probe-head.json`)で足した。コミットしていない。5 本とも HEAD で `-count=3` が通った。

| # | probe | 落とす mutant | 確かめたこと |
| --- | --- | --- | --- |
| 1 | `TestProbeOWG_StaleHolderGoneDuringTheWait_Launches` | E4(3 つの subtest すべて) | 印のない stale のロックを 200 ms 後に放す。`--detach` は 150 ms〜3 s の間に `started` を返して 1 回起動し、`start <task>` と `start --plan` は起動の 1 行を出して leader を立てる |
| 2 | `TestProbeOWG_TheProductionSettleIsTwoSeconds` | S4 | `orgWatchGateSettleOverride` を 0 にして、1 時間前の印の stale を spawn `--dry-run` が 1.5〜6 s の間に断る(HEAD は 2.0〜2.1 s) |
| 3 | `TestProbeOWG_EnsureJudgesAMarkWithoutDeadmanByTheCallersConfig` | E7c(`--detach`)、E7d(`start <task>`) | 設定の `deadman_minutes` 30 で、`deadman_minutes` のない 20 分前の印を running と読み、`--detach` は `already running`、start は起動せずに leader を立てる |
| 4 | `TestProbeOWG_ALaunchErrorCarriesThePrefix` | E8(2 つの subtest) | 起動の差し替え口がエラーを返すと、`--detach` は `org: watch --detach: <エラー>`、start は `org: start: the cross-org watch did not start, so no leader was started: <エラー>` で、start は何も記録しない |
| 5 | `TestProbeOWG_EnsureFirstProbeJudgesByTheCallersConfig` | E7 | 設定の `deadman_minutes` 5 で、`deadman_minutes` のない 7 分前の印を stale と読み、`--detach` は(短くした)期限まで待ってから stale の文で断る |

依頼のとおり、テストは足していない。足すなら probe 1〜5 の 5 本で、それぞれ上の mutant を閉じる。足すかは orchestrator が決める。

### 実機確認(cycle 1)

`docs/evidence/org-watch-gate-live-2026-10-12.md` に、スクリプト(`live.sh`、`holder.py`、`takeover.sh`)と出力を byte 単位でそのまま貼った(貼ったあと、コードブロックを取り出して元のファイルと比べ、5 つとも一致した)。ビルドした HEAD のバイナリを、`env -i`、scratch の HOME と XDG、`osascript` を外した PATH で動かした。台帳はすべて repo の外の `--state-dir`。herdr の座席は立てていない。

| # | 確認 | 結果 |
| --- | --- | --- |
| 1 | 見張りなし: `spawn --dry-run`、型付きと `--raw` の TASK の `send --dry-run` | どれも rc 1 で、台帳のディレクトリもできない。QUESTION は rc 0 で、manifest はその 1 行だけ |
| 2 | `--detach` を 2 回 | `started (pid 45166)` と `already running (pid 45166)`。印に `deadman_minutes: 10` |
| 3 | `kill -9` のあと | spawn は not_running の文で rc 1。`start <task>` は先に見張り(pid 45256)を起動して stderr に pid と `watch.log` を出し、そのあと herdr がないので rc 1。見張りは start のあとも動き続け、`--stop` で止まった |
| 4 | 古い見張り(1 時間前の印でロックを持つ) | spawn と TASK は 2.0〜2.1 s、`--detach` と start は 10.0〜10.1 s 待ってから rc 1。見張りは起動しない |
| 5 | stale を待つ間に持ち主が消える | `--detach` も `start <task>` も 3.2 s で見張りを起動した(E4 の分岐が本番でも動く) |
| 6 | `--once` の見張りが動いている間(本物)と、stopping の印(手で置いたもの) | spawn、TASK、`--detach`、start はどれも待たずに rc 1。status は `running (--once)` と `stopping`、`--json` は `once: true` と `stopping` |
| 7 | 同時の `--detach` 8 本 × 3 回 | 3 回とも 8 本すべて rc 0 で、`started` は 1 本。ロックを持つ見張りは 1 つ。ただし、すぐあとの `--stop` は 3 回とも 30 s で時間切れ(下の「見つけたこと」) |
| 8 | 終わり | どの台帳にも `--stop` を打ったあと、`pgrep -fl '<scratch>/ralph org watch'` は何も返さず rc 1 |

後始末の間の本物の stopping は、watcher の呼び出しがないと一瞬で stopped に変わるので、捕まえられなかった(6 は手で置いた印で読み手の側だけを見た)。

#### 見つけたこと: 同時の起動のすぐあとの `--stop` が効かず、別の見張りが残る(中)

同時の `--detach` 8 本がすべて rc 0 で返った直後に `--stop` を打つと、3 回とも `the watch did not stop within 30s` で rc 1 になり、`started` と言われたのとは別の pid の見張りが動き続けた。各台帳の `watch.log` には banner が 2 行(ロックを取った見張りが 2 つ)あり、残りの子は 2 つ目の pid を名指しして断られている。

起きる順(実機記録の 7・8 と `watch.log` から):

1. ロックの取り合いに負けた子は、`AcquireWatchLock` の待ち(既定 2 秒)の間、生きている。すべての `--detach` が返った時点で、4〜6 個の `ralph org watch --all` のプロセスがあった
2. HEAD の `--detach` と start は、ほかの見張りの印が running になると rc 0 で返り、自分の子が終わるのを待たない(AC5、tech-debt 206 の直し)。base は子が断られて終わるまで待って rc 1 を返していたので、返った時点で子は残っていなかった
3. その 2 秒の間に `--stop` を打つと、勝った見張りは止まり、`stopAndRelease` で頼み(`watch-stop.json`)を消してからロックを放す
4. ロックを待っていた子がロックを取る。頼みはもう消えているので、その子は長く動く見張りとして動き続け、`--stop` はロックが放されるのを 30 秒待って時間切れになる

base dc41c4f5 との比較(`takeover.sh`): ロックを待つ foreground の見張りがあとを取ること自体は base にもある(base も HEAD も `--stop` が 30 s で時間切れ)。同時の `--detach` 8 本の直後の `--stop` は、base では 0.97 s で rc 0、そのあと見張りのプロセスは 0。HEAD では 30.06 s で rc 1、そのあと 1 つ動き続ける。なので、この PR の AC5 の直しで新しく表に出た動きである。

影響: 窓はロックの待ちの 2 秒で、同時の start か `--detach` のあと 2 秒以内に `--stop` を打ったときだけ起きる。止めたい人の `--stop` が失敗と言い、見張りが 1 つ残る(見張りがないまま作業が始まる側には転ばない)。AC と計画の範囲の外なので、判定は pass のままにし、この PR で直すかは orchestrator に任せる。直し方の候補(試していない): (a) `launchAndAwaitOrgWatch` が、ほかの見張りを成功とするとき、自分の子が終わるのを(ロックの待ちの分だけ)待ってから返す、(b) `--detach` と start の子はロックを待たずに断る(起動する前の判定で not_running を見ているので、待つ理由は取り合いだけ)、(c) 止まる見張りが頼みを消すのを、ロックを待つ見張りの開始より前の頼みに限る。どれを選んでも、ロックを待つ子が残る形のテスト(今の偽の子は通らない)が要る。

### Failure analysis(cycle 1)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | テストの失敗はない。依頼にあった 2 本(`TestCheckScaffoldIntegrity_…`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)も、どの実行でも落ちなかった | - |

### Regression checks(cycle 1)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| tech-debt 206: 同時の `--detach` の 1〜2 本が rc 1 | 直った | 実機 7(3 回とも 8 本すべて rc 0)、`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` ほか。ただし上の「見つけたこと」 |
| tech-debt 204 (e): 断られる 2 つ目の見張りが banner を出す | 直った | `TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`、mutant B1・B2・B3、実機 7 の `watch.log`(断られた子に banner がない) |
| tech-debt 201: `ralph status` が呼んだ側の deadman で印を読む | 直った | `TestStatusCmd_OrgWatchOfAWatchWithAnotherConfigReadsByItsMark`、mutant P2・M2 |
| tech-debt 205: 子が打った worktree と herdr の pane の環境を引き継ぐ | 直った | `TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`、mutant C1〜C5 |
| 7d2e4bbb: send のエラーに `org: send:` が重なる | 直った | `TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce`、mutant N2・N3 |
| 6a の見張りの動き(ロック、印、`--stop`、BeforeStop) | 変わらない(stopping への変更は AC10 の範囲) | base の 1,383 本が名前のまま通る。`-race -count=5` の `OrgWatch_` と `RunWatchAll_` |

### Test gaps(cycle 1)

- 同時の start か `--detach` のすぐあとの `--stop` が効かない(上の「見つけたこと」、中)。テストの偽の子は、ロックを待ったまま残る子の形を持たないので、この流れを通らない
- mutant の生き残り 6 件(S4、E4、E7、E7c、E7d、E8)。どれも probe 1〜5 が閉じる(コミットしていない)。E7 系は前半の版の印だけの話
- 本物の見張りが後始末(stopping)をしている間に、spawn・TASK・start を打つテストも実機もない(印の書き手と読み手を別々に固定している。verify と同じ)
- AC8 の座席の `wait` と `inbox show/ack/resolve/notify` は、見張りなしで動くことをテストで見ていない(verify と同じ)
- AC6 の main worktree の中の台帳での起動は、テスト(偽の起動)だけで、実機では見ていない。Linux も見ていない
- 実機で leader が立つところ(herdr あり)は見ていない。start が見張りを起動したあとの leader の spawn はテストだけが見る

### 判定(cycle 1)

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail(top-level 1,426 本が PASS、base より 46 本多く、消えたのは AC10 の許す改名 1 本だけ)。`-race`(org と cli)と、増えたテストと変わったテストを含む 174 本の `-race -count=5` も 0 fail で、DATA RACE は 0 件。依頼の mutant はすべて落ち、64 件のうち 57 件が red。生き残りは等価の 1 件と、probe が閉じる 6 件。実機では、見張りなしの拒否と記録のなさ、`--detach`、kill のあとの start の起動、stale の待ち時間、持ち主が消えたあとの起動、`--once` と stopping への拒否が期待どおりで、最後に見張りのプロセスは残らなかった
- Fail: 0
- Blocked: なし。ただし「見つけたこと」(同時の起動のすぐあとの `--stop` が効かない、中)は、この PR の AC5 の直しで新しく出た動きなので、/pr の前に直すかを orchestrator が決める。probe 1〜5 を足すかも orchestrator が決める
