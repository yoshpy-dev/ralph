# Test report: org-cross-watch

- Date: 2026-10-11(JST。実行は 18:46〜19:30)
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: HEAD dd63a414(base 577ebeaf)。behavioral test だけを実行した(静的解析は /verify の担当)。依頼された 4 つを行った: `run-test.sh` と `go test`(`-race` を含む)、AC とテストの対応、`go test -overlay` の mutation、一時の台帳での実機確認。あわせて、計画の Edge cases のうちテストのないものを probe のテストで確かめ、Linux の setsid を Docker のコンテナで確かめた
- Evidence: `docs/evidence/test-2026-10-11-org-cross-watch.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-11-094617.log`(時刻は UTC)。実機確認は `docs/evidence/org-cross-watch-live-2026-10-11.md`(コミットする)
- テストは足していない。mutant の写しと probe のテストは scratchpad に置き、`go test -overlay` で使った。worktree の追跡ファイルは書き換えていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,972)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 7 分 1 秒、rc 0 |
| `go test ./... -count=1 -v` | top-level 1,368(org 551、cli 541)、subtest 1,447 | すべて | 0 | 3(下の注) | 92 s(cli 91.4 s、org 27.7 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 97 s(org 31.0 s、cli 92.5 s)、rc 0 |
| `go test -race -run 'Watch\|Probe\|Alive\|Deadline\|Status\|Inbox\|Escalate' -count=3`(org と cli) | top-level 627(org 124 本 × 3、cli 85 本 × 3) | すべて | 0 | 3(1 本 × 3) | 59 s、rc 0。`DATA RACE` は 0 件 |
| `go test -coverprofile`(org、cli、config) | 3 パッケージ | 3 | 0 | 0 | 92 s、rc 0 |
| Linux の `ubuntu:24.04`(aarch64)で、クロスビルドした test binary | cli の watch と status 22 本、org の watch・期限・受信箱 126 本 | 147 | 1(環境) | 0 | cli 2 s、org 4 s |
| probe のテスト(`-overlay` で足す 9 本、`-race -count=3`) | 27 | 27 | 0 | 0 | 7 s |
| mutation(`go test -overlay`、HEAD のテスト) | 82 件 | red 73 件 | 生き残り 9 件(下の節) | - | 1 件 30〜140 s |
| 実機確認(macOS 2 本、Linux 1 本) | 下の「実機確認」 | すべて期待どおり | 0 | - | 合わせて 1 分ほど |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた(`PASS: 29` のような集計行は除いた)。40 本すべてが `OK` で終わり、FAIL の行は 0。Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)
- `run-test.sh` の言語の scope は、`scripts/ralph-config.sh` が言語に分類されないので full に落ちた(`Language scope: full fallback`)。中の `go test ./...` は `internal/org` だけを実行し、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した。go は 1.26.0(darwin/arm64)
- Go の Skipped の 3 件は、`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin では skip する。Linux のコンテナでは走って通った)と、`internal/scaffold` の前からある 2 件(`TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`)
- この枝で足した top-level のテストは 62 本(`internal/cli/org_watch_test.go` 15、`internal/cli/status_test.go` 6、`internal/config/config_test.go` 4、`internal/org/deadline_test.go` 11、`internal/org/watch_all_test.go` 15、`internal/org/watch_test.go` 4、`internal/org/watchalive_test.go` 7)。62 本とも通った。消えた 4 本は `TestOrgWatch_RequiresOrgID`(拒否のテストに移った)と、`RunWatch` の 3 本(`RunWatchAll` の 3 本に置き換わった)
- Linux の 1 件の失敗は `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` で、コンテナに git がない(`exec: "git": executable file not found in $PATH`)ため。macOS では通っている。コードの失敗ではない
- 実行中、ほかのセッションの負荷で load average は 4〜93(12 コア)だった。既知の flaky なテスト(tester の MEMORY.md の 3 本)はどれも落ちなかった

## Coverage

- Statement: `internal/org` 94.0%、`internal/cli` 86.3%、`internal/config` 93.2%(前の計画の時点は 93.8%、86.0%、92.8%)。ほかの 5 パッケージは測り直していない
- Function(この枝が足した関数のうち 100% でないもの): `launchOrgWatch` 0.0%(本物の ralph を exec する本番の起動。TestMain が panic する偽物に差し替えるので、どのテストも通らない。これで AC12 の「テストは test binary を exec しない」を確かめた。本物の起動は実機確認で通した)、`startDetachedProcess` 85.7%(ログやディレクトリを開けない枝)、`runOrgWatchStop` 85.7%(頼みを書けない枝)、`runOrgWatchDetach` 95.7%、`acquireHeldFileLock` 81.2%・`fileLockHeld` 85.7%・`AcquireWatchLock` 83.3%(flock や open の失敗、印の読めない拒否)、`newWatchAllRun` 71.4%(引数が空のときの既定値)、`cycle` 85.7%、`pulse` 90.9%、`writeMark` 66.7%・`finish` 66.7%(印を書けない枝)、`dropOldStopRequest` 80.0%、`ReadWatchStopRequest` 87.5%、`RemoveWatchStopRequest` 75.0%、`writeJSONFileAtomic` 85.7%
- 100% の関数: `deadline.go` の 7 関数すべて(`InboxDeadlineDue`、`SweepInboxDeadlines` ほか)、`ProbeWatch`、`watchAliveWindow`、`AppliesTo`、`RunWatchAll`、`pulseOrg`、`waitForNextCycle`、`callWithTimeout`、`realGitStatus`、`checkOrgWatchFlags`、`runOrgWatchAll`、`orgWatchChildArgs`、`statusDeadmanMinutes`、`orgWatchLine`、`buildStatusOrgWatchJSON`
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 行の coverage は「条件を外しても気づくか」を示さない。そのため下の mutation で確かめた。`ProbeWatch` は 100% だが、確かめられないロックの扱い(P2)は mutation で生き残った

## AC とテストの対応

テストはすべて HEAD で通った。「実機」は `docs/evidence/org-cross-watch-live-2026-10-11.md` の確認、「probe」は下の「Probe のテスト」(コミットしていない)の番号。

| AC | 固定するテスト | 補い |
| --- | --- | --- |
| AC1 設定の既定、検証、3 面 | `TestDefault_OrgDirector`、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgDirectorRoundTrip`(resolve_minutes が ack_minutes より短い設定を含む)、`TestLoad_OrgDirectorRejects`(2 キー × 0 と -1)、`TestDefaultsLockStep` の 2 行 | なし |
| AC2 期限の判定表 | `TestInboxDeadlineDue`(22 行と、時刻の読めない件 8 通り)。表の 7 項目、期限ちょうど、時計の逆行を含む | probe 1(resolve_minutes 5 < ack_minutes 10 の判定と掃き出し) |
| AC3 人への経路と状態の不変 | `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman`、`_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral`、`_SkipsResolvedItemsAndKeepsIDOrder` | 本物の osascript は確かめていない(Test gaps) |
| AC4 書けない台帳での送り直しの間隔 | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes`、`_NotifiedUnwritableIsHeldBack`、`_HoldBackIsPerReason` | なし |
| AC5 起動し直したときの計算し直し | `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`、`TestRunWatchAll_SendsAnInboxItemPastItsDeadlineInTheFirstCycleOnce`、`TestOrgWatch_Once_SendsAnInboxItemPastItsDeadline` | probe 2(`inbox.jsonl` のない台帳) |
| AC6 escalate はすぐ送り、二度目は resolve の期限だけ | `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline`。escalate の既存のテスト(名前に Escalate を含む 25 本)も通る | なし |
| AC7 全 org を 1 プロセスで、disband した org は見ない | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`、`TestRunWatchAll_MultipleOrgs_SeparateStatusFiles`、1 org のサイクルの既存のテスト(`TestWatch_*` 28 本) | probe 5(30 席・5 org の 1 サイクルの時間) |
| AC7b 返らない呼び出し、書けない状態ファイル | `TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn`、`_EveryHungDriverCallOfTheFirstOrgIsCutOff`、`_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg`、`_AManifestThatCannotBeReadSkipsOnlyThePulse`、`_AnInboxThatCannotBeSweptDoesNotStopThePulse`、`TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` | probe 8(受信箱の期限をパルスより先に見る順)。本物の herdr が返らない場合は確かめていない(Test gaps) |
| AC8 `--org-id` と `--all` なしの拒否 | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing`(9 通り、stdout が空で state dir を作らない) | なし |
| AC9 2 つ目の見張りの拒否、終わればロックが放される | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`、`TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`、`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` | 実機 1(別プロセスの `--once` が pid を名指しして rc 1。`kill -9` のあとロックが放され、次の `--detach` が通る。verify が挙げた kill の穴)、実機 2(8 つ同時の `--detach` で、動く見張りは 1 つ) |
| AC10 印の書き込み、SIGINT と SIGTERM | `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd`、`TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark`、`TestWriteWatchAlive_ReplacesThroughARenameAndLeavesNoTemporaryFile` | probe 3(サイクルの途中で ctx が取り消される)、probe 4(stopped の印を書いてからロックを放す順)、実機 1(間隔 2 秒で `ts` が進む) |
| AC11 判定関数の 3 状態 | `TestProbeWatch_TellsRunningStaleAndNotRunningApart`(17 行、窓の境界)、`TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`、`TestReadWatchAlive_MissingAndCorrupt`、`TestAcquireWatchLock_WaitsOutAShortSharedLock` | probe 6(確かめられないロックは持たれているとみなす) |
| AC12 `--detach` | `TestOrgWatchDetach_AlreadyRunning_StartsNothing`、`_Stale_StartsNothingAndSaysHowToStopIt`、`_StartsTheChildAndWaitsForItsRunningMark`、`_ChildExitsFirst_ExitsOneNamingTheLog`、`_DeadmanMinutesZero_StartsTheChild`、`_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`。test binary を exec しないことは `launchOrgWatch` の coverage 0.0% で確かめた | 実機 1・3(本物の子、macOS と Linux の setsid)、probe 7(ほかの見張りの印を自分の子と取り違えない) |
| AC12b 待ちきれなかった子の回収 | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock`(ロックが放される場合と残る場合) | なし |
| AC13 `--stop` | `TestOrgWatchStop_NotRunning_SaysSoAndWritesNoRequest`、`_StopsARunningWatch`、`_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`、`TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt` | 実機 1・3(本物のプロセスが 0.43 s で止まる) |
| AC13b 古い頼みに従わない、シグナルを送らない | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`(前・同時刻・1 ns 後)、`TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt`、`TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`(印が生きている別プロセスを指す) | probe 9(ロックを待つ間に書かれた頼みに従う) |
| AC14 `ralph status` の行と `org_watch` | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState`、`_OrgWatchLineFollowsTheStateDirLine`、`_EmptyRosterStillShowsTheOrgWatch`、`_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml`、`_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault`、`_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad` | 実機 1(running、kill のあとの not running と残った印、stopped) |
| AC15 spawn・send・start・stop・disband・escalate は変わらない | それらのテストファイルは差分に入っていない(`git diff --stat 577ebeaf...HEAD -- '*_test.go'` の 12 本に含まれない)。名前に Spawn・Send・Start・Stop・Disband を含む top-level のテスト 313 本と、Escalate を含む 25 本が通った | なし |
| AC16 tech-debt (c)〜(h) は動作を変えない | `internal/cli/org_inbox_test.go` と `internal/org/inbox_test.go` は無変更で通る。(h) を拾う `TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne` と `TestInboxCommand` も通る。`escalate_test.go` の 9 か所の呼び出しの形の変更は、verify が assertion は変わっていないと読んだ | なし |
| AC17 文書 | behavioral test はない。`check-sync.sh` と `check-skill-sync.sh` は verify の静的な検査が持つ | 対象外(/verify と /sync-docs) |

behavioral test のない AC は AC17 だけで、文書の AC なので /test の対象外とした。

## Mutation

`scratchpad/ocw-test/mut.py` は、`internal/org/watchalive.go` などの写しを書き換えて `go test -overlay` で走らせる。書き換える文字列がファイルに 1 回だけあることを、当てる前に確かめた。org 層の mutant は `internal/org` の全体を走らせ、生き残ったら `internal/cli` の全体も走らせた。CLI の mutant はまず `-run 'OrgWatch|StatusCmd'` で走らせ、生き残ったものを `internal/cli` の全体で流し直した。

依頼された 9 点と、その mutant:

| 依頼 | mutant | 結果 |
| --- | --- | --- |
| ロック(`AcquireWatchLock` が持ち続ける) | L1、L2、L3、L4、L5b | 5 件とも red |
| probe のロックの確認 | P1、P2、P3、P4、P5 | P1・P3・P4 は red。**P2 が生き残り**(確かめられないロックの扱い)。P5 は等価 |
| 窓の規則(deadman の下限と間隔の 2 倍) | W1(下限)、W2(2 倍の項)、W3、W4、W5、W6、W7 | 7 件とも red |
| `InboxDeadlineDue` の ack と resolve の順 | D1 | red |
| ack の期限の open の条件 | D2 | red |
| `RunWatchAll` の org ごとの分離 | A1、A2、A3、A5、A7 | 5 件とも red。分離とは別の A4 は等価、A6(順)は生き残り |
| 止める頼みに従うこと | S1、S2、S3b、S4、S5b、S6、S7、S8 | S6 以外は red。**S6 が生き残り**(開始の時刻を読む位置) |
| `--detach` の期限切れの子の回収 | X1、X2 | どちらも red |
| `--org-id` の拒否 | F1 | red |

全体は 82 件で、73 件が red、9 件が生き残った。生き残りの内訳は、本物の穴が 5 つ(P2、X3、L6b と L6c、A6、S6。L6b と L6c は同じ順の入れ替えの 2 通り)、等価なものが 3 つ(P5、A4、X12)。下の表の「落としたテスト」は、CLI の mutant については `-run 'OrgWatch|StatusCmd'` の 1 回目のもの。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| L1 | `acquireHeldFileLock` が flock を取った直後に外す(ロックを持たない) | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| L2 | `RunWatchAll` が最初のサイクルの前にロックを放す | red | `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`, `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| L3 | `AcquireWatchLock` が待たない(1 回だけ試す) | red | `TestAcquireWatchLock_WaitsOutAShortSharedLock` |
| L4 | 拒否のエラーが `ErrWatchRunning` を包まない | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds` |
| L5b | 拒否のエラーが印の pid を名指ししない | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds` |
| L6b | `finish` がロックを先に放し、50 ms おいて stopped の印を書く | **生き残り** | (probe 4 が落とす) |
| L6c | `finish` がロックを先に放し、すぐ stopped の印を書く | **生き残り** | (probe 4 が落とす) |
| P1 | `ProbeWatch` がロックを見ない(running の印だけで running) | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart`, `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| P2 | `ProbeWatch` が、確かめられないロックを空いているとみなす | **生き残り** | (probe 6 が落とす) |
| P3 | `fileLockHeld` が、ロックのファイルがないときに持たれているとみなす | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| P4 | `fileLockHeld` が O_CREATE で開く(watch.lock を作る) | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding` |
| P5 | `fileLockHeld` が共有ロックを LOCK_UN で外さない | **生き残り**(等価) | |
| W1 | 窓: `deadman_minutes` が 1 未満のときに既定の 10 にしない | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W2 | 窓: 間隔の 2 倍の項を外す | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W3 | 窓: 間隔の 2 倍を 1 倍にする | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W4 | 窓: ちょうど窓の年齢を stale にする(`>` を `>=`) | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W5 | 窓: `deadman_minutes` を見ず、いつも既定の 10 | red(org では生き残り、cli で red) | `TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml` |
| W6 | status: `./ralph.toml` ではなく既定の deadman で判定する | red | `TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml` |
| W7 | `ProbeWatch`: ロックが持たれていて印が stopped なら running | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| D1 | 期限: ack を resolve より先に見る(順を入れ替える) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_HoldBackIsPerReason` |
| D2 | ack の期限: open の条件を外す(acked にも効く) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_SkipsResolvedItemsAndKeepsIDOrder` |
| D3 | ack の期限: 「notified が 1 つもない」を外す | red | `TestInboxDeadlineDue` ほか 5 本 |
| D4 | ack の期限: ack の期限の notified だけが止める | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D5 | resolve の期限: notified を見ず毎回送る | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D6 | resolve の期限: どの notified でも止まる(escalate した件に届かない) | red | `TestInboxDeadlineDue` ほか 3 本 |
| D7 | resolve の期限: ちょうど resolve_minutes で送る(`>` を `>=`) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D8 | ack の期限: ちょうど ack_minutes で送る(`>` を `>=`) | red | `TestInboxDeadlineDue` |
| D9 | resolved の件にも resolve の期限が効く | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_SkipsResolvedItemsAndKeepsIDOrder` |
| D10 | 送り直しの間隔を resolve_minutes にする | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes` |
| D11 | 送り直しの間隔のちょうどでも控える(`<` を `<=`) | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes` |
| D12 | 送れなかった件を覚えない(毎サイクル送り直す) | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes`, `TestSweepInboxDeadlines_HoldBackIsPerReason`, `TestSweepInboxDeadlines_NotifiedUnwritableIsHeldBack` |
| D13 | 控える鍵に reason を入れない | red | `TestSweepInboxDeadlines_HoldBackIsPerReason` |
| D14 | 時刻の読めない件の注意を毎回出す | red | `TestSweepInboxDeadlines_WarnsOnceAboutAnEscalatedAtThatDoesNotParse` |
| D15 | 1 分未満の期限を断らない | red | `TestSweepInboxDeadlines_ReturnsAnErrorWhenItCannotStart` |
| D16 | 期限の送信の reason を `inbox_no_director` にする | red | `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman` ほか 10 本 |
| D17 | ack の期限でも resolve の期限の文面にする | red | `TestSweepInboxDeadlines_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral`, `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman` |
| A1 | 1 つ目の org の失敗でパルスを終える(ほかの org を回さない) | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg` |
| A2 | すべての org が 1 つの状態を共有する | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_MultipleOrgs_SeparateStatusFiles` ほか 4 本 |
| A3 | すべての org が 1 つの状態ファイル名を使う | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_MultipleOrgs_SeparateStatusFiles` ほか 7 本 |
| A4 | 走らなくなった org の状態をメモリから消さない | **生き残り**(等価) | |
| A5 | disband した org も含め manifest のすべての org を回す | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated` |
| A6 | パルスを受信箱の期限より先に回す | **生き残り** | (probe 8 が落とす) |
| A7 | 受信箱を掃き出せないとき、パルスと印を飛ばす | red | `TestRunWatchAll_AnInboxThatCannotBeSweptDoesNotStopThePulse` |
| A8 | 最初のサイクルの前に印を書かない | red | `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| A9 | サイクルのあとに印を書き直さない | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| A10 | `callWithTimeout` に期限がない | red | panic: `TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn` の 10 s の guard(下の注) |
| A11 | `realGitStatus` が git を context つきで起動しない | red | `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` |
| A12 | 状態ファイル名にする前に org_id を確かめない | red | `TestRunWatchAll_AnOrgIDThatIsNotAnIdentifierIsSkipped` |
| A13 | 読めない状態ファイルを毎サイクル言う | red | `TestRunWatchAll_AStatusFileThatCannotBeReadIsSaidOncePerOrg` |
| A14 | 読めた状態をメモリに残さない(毎サイクル読み直す) | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg` |
| S1 | サイクルの間の待ちで止める頼みを読まない | red | `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S2 | 開始と同時刻の頼みにも従う | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S3b | どの頼みにも従う(古いものも) | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S4 | 起動のときに古い頼みを消さない | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S5b | 起動のときに、自分宛ての頼みも消す | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S6 | 開始の時刻をロックを取ったあとに読む | **生き残り** | (probe 9 が落とす) |
| S7 | 従った頼みを `finish` が消さない | red | `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S8 | 頼みの時刻を秒で比べる(ナノ秒を落とす) | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S9 | `--stop` が印の pid に SIGTERM も送る(安全のため、このプロセスの子だけに) | red | `TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` |
| X1 | `--detach` の期限切れで子を止めない | red | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| X2 | `--detach` の期限切れでロックが放されるのを待たない | red | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| X3 | `--detach` が、印の pid が自分の子かを見ずに成功とする | **生き残り**(cli 全体でも) | (probe 7 が落とす) |
| X4 | `--detach`: stale でも起動する | red | `TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt` |
| X5 | `--detach`: running でも起動する | red | `TestOrgWatchDetach_AlreadyRunning_StartsNothing` |
| X6 | 子の引数に `--config` を渡さない | red | `TestOrgWatchDetach_StartsTheChildAndWaitsForItsRunningMark` |
| X7 | 子の引数にいつも `--interval-seconds` を渡す | red | `TestOrgWatchDetach_StartsTheChildAndWaitsForItsRunningMark` |
| X8 | 子を setsid で起動しない | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X9 | `watch.log` を追記でなく切り詰めて開く | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X10 | 子が先に終わっても期限まで待つ | red | `TestOrgWatchDetach_ChildExitsFirst_ExitsOneNamingTheLog` |
| X11 | 子を止めるとき SIGTERM を待たずに SIGKILL | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X12 | 子の stdin を /dev/null にしない(親から継ぐ) | **生き残り**(等価、cli 全体でも) | |
| F1 | `--org-id` を断らない | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F2 | `--all` を求めない | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` のサブテスト `no --all`(`org_watch_test.go:174`)(下の注) |
| F3 | `--detach` と `--once` を受ける | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F4 | `--stop` と `--detach` を受ける | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F5 | フラグの検査を runtime を作ったあとにする(断る前に state dir を作る) | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| G1 | SIGTERM を扱わない(SIGINT だけ) | red | `TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark` の `terminated`(テストのプロセスが SIGTERM で終わった) |
| G2 | 長く動く見張りが止まったとき watcher の呼び出しを取り消さない | red | `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` |
| T1 | 空の roster の `--json` に `org_watch` を入れない | red | `TestStatusCmd_EmptyRosterStillShowsTheOrgWatch`, `TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad` |
| T2 | 表の側で `org watch:` の行を出さない | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState`, `TestStatusCmd_OrgWatchLineFollowsTheStateDirLine` |
| T3 | 残った running の印の行から pid を落とす | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |
| T4 | `--json` に `mark_state` を写さない | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |

### 生き残った mutant

本物の穴で、コミットされたテストが固定しないもの(probe で閉じることを確かめた):

- **P2**: `ProbeWatch` の doc は「確かめられないロックは持たれているとみなす」と書くが、それを固定するテストがない。`fileLockHeld` がエラーを返す場合(`watch.lock` が開けない)を、どのテストも作らない。probe 6(mode 000 の `watch.lock`)は HEAD で通り、P2 で落ちる(`not_running` を返す)
- **X3**: `--detach` が「印の pid が自分の子のもの」を見ずに成功とする。すべての `--detach` のテストで、running の印は子(か、子より前からの見張り)のものだけなので、ほかの見張りが先にロックを取った形がない。実機 2 の同時の `--detach` がまさにその形で、HEAD は正しく rc 1 を返した。probe 7 は HEAD で通り、X3 で落ちる(子が終わったのに `started ralph org watch --all (pid 4242)` を出す)
- **L6b・L6c**: `finish` の「stopped の印を書いてからロックを放す」順(doc は「`--stop` がロックの空きを見たら stopped の印も見える」と書く)。順を入れ替えると、ロックを取った新しい見張りの running の印を、古い見張りが stopped で上書きしうる。`TestOrgWatchStop_StopsARunningWatch` は見張りが返ったあとで印を読むので、順を見分けない。probe 4 は HEAD で 10 回 × 20 周すべて通り、L6c(sleep なし)は 10 回とも 1 周目で落ちた
- **A6**: 受信箱の期限をパルスより先に見る順(Codex plan advisory の指摘 1)。AC7b のテストは「同じサイクルで届く」ことを見るので、パルスのあとに送っても通る。順を入れ替えると、herdr が返らない org があるときに、期限の通知が打ち切りの合計(席の数 × 10 秒まで)だけ遅れる。probe 8(最初の herdr の呼び出しの時点で `escalations.jsonl` に行がある)は HEAD で通り、A6 で落ちる
- **S6**: 開始の時刻をロックを待つ前に読むこと(計画の S3a のずれ)。テストの時計は止まっているので、ロックの前に読んでもあとに読んでも同じ値になり、見分けられない。probe 9 は、ロックを持つ別の holder を置き、待っている間に時計を進めて頼みを書く。HEAD は 1 サイクルで止まり、S6 は頼みを古いものとして消して 3 サイクル回る。ただし probe 9 は「`RunWatchAll` が開始を読んだあと」を 100 ms の sleep で作っているので、コミットするなら時計の差し替えで同期させる

等価(穴ではない):

- **P5**: `fileLockHeld` が共有ロックを `LOCK_UN` で外さなくても、続く `Close` がロックを放す
- **A4**: 走らなくなった org の状態をメモリに残しても、その org がまた走るときは、HEAD でも毎サイクル保存した同じ内容をファイルから読み直す。違いはメモリの量だけ
- **X12**: `go test` は test binary の stdin をいつも `/dev/null` にする(stdin にパイプを渡しても変わらないことを確かめた)ので、子が親の stdin を継いでも `stdin: eof` になる。見分けるには、テストの中で `os.Stdin` をパイプに差し替える必要がある。本番の見張りは stdin を読まない

依頼どおり、テストは足していない。足すなら probe 4・6・7・8・9 の 5 本で、それぞれ 1 本ずつで生き残りを閉じる(scratchpad の `zz_probe_test.go` と `zz_probe_cli_test.go` にある)。足すかは orchestrator が決める。

mutant の当て方の注:

- L5 と S5 は書き換えで変数が使われなくなり、コンパイルに失敗した。`_ = mark`、`_ = req` を足した L5b と S5b で走らせ直した
- L6 は 2 回目の `Release` の stderr の 1 行(`file already closed`)だけで落ちた。順の違いで落ちたのではないので、`Release` を 1 回にした L6b(50 ms おく)と L6c(おかない)で走らせ直し、どちらも生き残った
- F2 と A10 は、終わらないテストが残ったので、そのテストのプロセスを SIGQUIT で止めた。F2 はその前にサブテストが落ちている。A10 は AC7b のテストだけで流し直し、10 s の guard の panic で落ちることを確かめた
- 実行中は load average が 6〜93 だった。落ちたテストはどれも、書き換えた動きを名指しする assertion で落ちていることをログで見た(時間切れだけで落ちたものはない)

## Probe のテスト

計画の Edge cases と mutation の生き残りのうち、コミットされたテストが固定しないものを、scratchpad の `zz_probe_test.go`・`zz_probe_timing_test.go`(`internal/org`)と `zz_probe_cli_test.go`(`internal/cli`)に書き、`go test -overlay` で足して走らせた。コミットしていない。HEAD では 9 本とも、`-race -count=3` で通った。

| # | probe | HEAD | 確かめたこと |
| --- | --- | --- | --- |
| 1 | `TestZZProbe_ResolveMinutesShorterThanAckMinutes` | 通る | ack 10・resolve 5 では、5 分ちょうどまでは何も送らず、5 分 1 秒で resolve の期限を 1 回だけ送り、その `notified` で ack の期限も止まる(11 分、3 時間でも鳴らない)。計画の Edge cases「resolve_minutes が ack_minutes より短い設定」は、設定のテストだけで判定のテストがなかった |
| 2 | `TestZZProbe_SweepWithNoInboxFile` | 通る | `inbox.jsonl` のない台帳の掃き出しはエラーなしで何も送らず、`escalations.jsonl` も作らない |
| 3 | `TestZZProbe_ContextCancelledMidCycleFinishesTheCycle` | 通る | 1 つ目の org の git status の最中に ctx を取り消しても、そのサイクルは最後まで回り(2 つ目の org も 1 サイクル、生存の偽の ALERT なし)、そのサイクルの印を書いてから stopped で終わる。計画の Edge cases「SIGTERM がサイクルの途中で届く」の org 層 |
| 4 | `TestZZProbe_TheLockIsFreeOnlyAfterTheStoppedMark` | 通る(10 回 × 20 周) | ロックが空いたのを見た時点で、印はもう stopped。L6b と、sleep を抜いた同じ並べ替え(L6c)は 10 回とも 1 周目で落ちた |
| 5 | `TestZZProbe_CycleTimeWith30Seats` | 通る | 30 席・5 org、全席が毎サイクル担当範囲を変える(毎サイクル ALERT)。herdr と agmsg は偽物ですぐ答える。1 サイクルは 13〜42 ms(`-race` では 61〜222 ms)。本物の herdr と agmsg のプロセスの時間は含まない |
| 6 | `TestZZProbe_ALockThatCannotBeCheckedCountsAsHeld` | 通る | mode 000 で開けない `watch.lock` は持たれているとみなす(印がなければ stale、新しい running の印なら running)。P2 で落ちる |
| 7 | `TestZZProbe_DetachDoesNotTakeAnotherWatchsMarkForItsChild` | 通る | `--detach` の子がロックを取れずに終わる間に、ほかの見張りが新しい running の印を書いていても、`--detach` は自分の子を「started」とは言わず rc 1。X3 で落ちる |
| 8 | `TestZZProbe_InboxDeadlinesComeBeforeThePulse` | 通る | サイクルの最初の herdr の呼び出しの時点で、期限を過ぎた件の `escalations.jsonl` の行がもうある(受信箱の期限をパルスより先に見る)。A6 で落ちる |
| 9 | `TestZZProbe_AStopWrittenDuringTheLockWaitIsFollowed` | 通る | ほかの holder がロックを持つ間に `RunWatchAll` を起こし、待っている間に時計を進めて頼みを書く。ロックが空くと 1 サイクルで止まる。S6 で落ちる(3 サイクル回る)。「開始を読んだあと」は 100 ms の sleep で作っている |

## 実機確認

`docs/evidence/org-cross-watch-live-2026-10-11.md` にコマンドと出力を残した。依頼の規則どおり、台帳は scratchpad の下だけ、manifest も受信箱も置かず(herdr・agmsg・osascript は呼ばれない)、ralph は `env -i` で空の PATH と scratchpad の HOME で起こした。利用者の herdr サーバー、`~/.config/herdr`、本物の台帳には触れていない。最後の `pgrep -fl 'org watch --all'` は何も返さなかった。

1. macOS(依頼の流れ): `--all --detach` は rc 0 で pid 9450 を出し、0.13 s で返った。子は新しいセッション(`PGID == PID`、`TTY ??`、`STAT Ss`、`PPID 1`)。`ralph status` と `--json` は running。`kill -9 9450` のあと、`ralph status` は `org watch: not running (a running mark is left from pid 9450, last cycle …)`、`--json` は `state: not_running`、`mark_state: running`。もう一度の `--detach --interval-seconds 2` は rc 0 で新しい pid 9606(印の `interval_seconds` は 2、4.5 s で `ts` が進んだ)。そのあいだの `--all --once` は rc 1 で `(pid 9606, from …/watch-alive.json)` を名指しした。`--stop` は rc 0、0.43 s で返り、status は `not running (stopped at …)`、`watch-stop.json` は残らなかった
2. macOS(足した確認): 新しい台帳に `--detach` を 8 つ同時に、4 回打った。どの回も動いている見張りは 1 つだけで、`started` 1 本、`already running` 5〜6 本、exit 1 が 1〜2 本(`the watch (pid N) exited before it was running`)。exit 1 の子はロックを 2 秒待ってから `another ralph org watch --all is running …` を `watch.log` に書いて終わっていた
3. Linux(足した確認、`ubuntu:24.04` のコンテナ、一般ユーザー): 同じ流れが通った。`--detach` の子は `SID 20`(自分の pid)で、起こしたシェルの SID 11 と違う。setsid は Linux でも効く

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし(HEAD の macOS の実行) | - | - | - |
| `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`(Linux のコンテナだけ) | `exec: "git": executable file not found in $PATH` | コンテナの `ubuntu:24.04` に git がない。テストの前提の環境の違いで、コードの失敗ではない | なし。CI(git のある Linux)では走る |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 1 org のパルスのサイクル(stall・生存・担当範囲・デッドマン) | 通る | `TestWatch_*` 28 本(assertion は変わっていないと verify が読んだ)。AC7 のバイト一致のテストが `RunWatchAll` と 1 org のサイクルを比べる |
| escalate・inbox・`wait --inbox` | 通る | 名前に Escalate を含む 25 本、Inbox を含む 60 本、Wait を含む 17 本 |
| spawn・send・start・stop・disband | 通る | 名前にそれらを含む top-level 313 本。テストファイルは無変更 |
| 旧い台帳の検査(`org_legacy_ledger_test.go` の watch の 1 行の書き換え) | 通る | `TestOrgLegacyLedger_*` 5 本 |
| `ralph status` の既存の表示と `--json` の形 | 通る | `TestStatusCmd_*` 25 本(うち新しい 6 本) |
| 設定の lock-step | 通る | `TestDefaultsLockStep` |

## Test gaps

- 本物の osascript の通知を、切り離した子から出すこと: 依頼どおり確かめていない(実機の台帳に期限を過ぎた件を置いていない)。期限の通知の AppleScript 3 通り(ack の期限、resolve の期限、`<invalid>` に置き換えた件)は、`osacompile` でコンパイルだけを通した(実行していない)。子は `TTY ??` の新しいセッションで動くので、ログインセッションの中で通知が出るかは、計画の Open questions のまま残る
- Linux の setsid: Docker の `ubuntu:24.04`(aarch64)で `SID == PID` を確かめた。CI の Linux では、`TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` が process group を見るだけで、SID は見ない
- 本物の herdr が返らない場合: テストは偽物の herdr と agmsg で打ち切りを確かめる。本物の driver が `exec.CommandContext` と 2 秒の `WaitDelay` で切れることは、verify がコードを読んだだけで、実機では確かめていない。30 席の 1 サイクルは ralph の側だけで 13〜42 ms(probe 5)で、本物の herdr の `agent get` と agmsg のプロセスの時間は入っていない
- プロセスをまたぐロックの取り合い: 実機で、別プロセスの 2 つ目の見張り、`kill -9` のあとの解放、8 つ同時の `--detach`(4 回)を確かめた。コミットされたテストは、同じプロセスの中の 2 つの open file で排他を見る(flock は open file ごとに効くので、別プロセスと同じ意味を持つ)。`ralph status` の共有ロックと見張りの起動の取り合いを、プロセスをまたいで何度も試したことはない
- mutation の生き残りのうち本物の穴 5 件(P2、X3、L6b と L6c、A6、S6)は、テストで固定されていない。probe 4・6・7・8・9 がそれぞれを閉じることを確かめた。足すかは orchestrator に任せる
- 同時の `--detach` の 1〜2 本が、見張りが動いているのに rc 1 になる(下の「参考」)。テストはこの形を作らない(X3 の probe 7 が作る)
- `--config` で別の `deadman_minutes` を渡した見張りと、`./ralph.toml` で判定する `ralph status` のずれは、tech-debt の行のとおりでテストはない。self-review の N2(読めない状態ファイルの org を黙って飛ばし続ける)と N3(`--stop` で取り消した watcher の呼び出しを失敗として記録する)も tech-debt にあり、テストはその今の動きを固定している
- Linux のコンテナには git がないので、`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` だけはコンテナで走らせられなかった(macOS では通る)

## 参考(テストの失敗ではない)

- 2 つ目の見張りは、ロックを取る前に `watching every org (interval=30s once=true …)` の banner を stdout に出してから断る(実機 1 の手順 4、Linux でも同じ)。banner は `runOrgWatchAll` が `RunWatchAll` の前に出すため。終了コードと pid の名指しは AC9 のとおり
- 同時の `--detach` のうち 1〜2 本は、見張りが動いているのに rc 1 になる(実機 2)。計画の AC12 の「子が先に終わったら終了コード 1」のとおりの動き。後半の PR で `ralph org start` が見張りを起動するとき、複数の start が同時に走ると、start が見張りの起動の失敗と読みうる。そのときは、rc 1 のあとに `ProbeWatch` が running なら成功として扱う、などを考えるとよい

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail、`-race`(org と cli)と、watch・判定・期限・受信箱・status のテストの `-race -count=3` も 0 fail。AC1〜AC16 と AC7b・AC12b・AC13b に behavioral test があり、すべて通る(AC17 は文書の AC で対象外)。依頼の 9 点の mutant は、P2 と S6 のほかはすべて落ち、82 件のうち 73 件が落ちた。実機では、`--detach`・`kill -9` のあとの判定と起動し直し・2 つ目の見張りの拒否・`--stop` が期待どおりに動き、プロセスは残らなかった(Linux のコンテナでも同じ)
- Fail: 0
- Blocked: なし。生き残った本物の穴 5 つ(P2、X3、L6、A6、S6)と Test gaps の残りは merge を止めない。probe 4・6・7・8・9 をテストに足すかは orchestrator が決める
