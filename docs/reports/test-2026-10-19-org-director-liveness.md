# Test report: org-director-liveness

- Date: 2026-10-19(JST 05:26〜06:20)
- Plan: docs/plans/active/2026-10-18-org-director-liveness.md(8 段目の 8b-2a。headless の director の確かめと、start の片付け)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 7e43f062(base origin/main 88e37b1d、23 コミット、51 ファイル、+7,003/-330)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)。実行中の load average は 3〜30
- Evidence: `docs/evidence/test-2026-10-19-org-director-liveness.md`(各コマンドの結果の行、落ちたテストの出力と再実行、coverage、mutant 31 個の表、canary の確かめ)。run-test.sh の全ログは `docs/evidence/verify-2026-10-18-202634.log`(gitignore の対象)。`<scratch>` はセッションの scratchpad
- テストもコードも変えていない。mutant と canary の overlay は scratchpad に置き、`go test -overlay` で読ませた。`git status --porcelain --ignored`(`.harness/cache/` を除く)は、すべての実行の前とあとで同じだった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。golang を選んだ。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 956 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -timeout 30m -v` | top-level 2,163 | 2,159(PASS の行は subtest を含め 5,848) | 1(`TestRegate_FetchFailures`、下の Failure analysis) | 3 | 597 秒、rc 1 |
| 同じ失敗の再実行 `-run 'TestRegate_FetchFailures$' -count=10`(go1.25.8) | 10 | 10 | 0 | 0 | 111 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -count=1 -timeout 30m -v`(パッケージ全体の再実行) | top-level 1,068 | 1,067 | 0 | 1 | 597 秒、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -run 'Director\|Watch\|Status' -count=1 -timeout 30m`(go1.26.0) | top-level 550 | 550(PASS の行 1,584) | 0 | 0 | 211 秒、rc 0、DATA RACE 0 |
| 上の filter に入らない新しいテスト 5 本の `-race`(`TestMarkStoppedAtVersion_*` 2 本、driver の 3 本) | 5(driver は既存の 5 本も) | すべて | 0 | 0 | rc 0、DATA RACE 0 |
| うちこの差分で足したテスト(org 48、org/driver 3、cli 16) | top-level 67 | go1.25.8 の回で 67 すべて PASS。`-race` は 62 + 5 | 0 | 0 | - |
| canary を PATH の先頭に置いて新しいテスト 67 本(AC9、下の「隔離」) | 67 | 67 | 0 | 0 | 11 秒、canary の呼び出し 0 |
| mutation(`go test -overlay`。org 26、cli 5) | mutant 31 | 赤 31 | 生き残り 0 | - | org 52〜65 秒、cli 78〜93 秒(2 本ずつ並べた) |

- run-test.sh の shell は 41 本すべてが `OK` で終わり、`FAIL:` の集計の行はどれも `FAIL: 0` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- run-test.sh の Go は cli(384.0 s)、config、org(502.5 s)を流し直し、insights、org/driver、org/protocol、scaffold、upgrade は worktree の `.harness/cache/go-build` で `(cached)` だった。go1.25.8 の回は `-count=1` で 8 パッケージすべてを流し直し、driver も ok だった
- go1.25.8 の回は `-race` の回と並べて走らせ、その中で `TestRegate_FetchFailures` が 1 件落ちた。org の再実行は cli の coverage の回と最初の 160 秒だけ重なった
- Skipped の 3 件は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin で飛ばす作り)、scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS` で、この差分と関係ない
- 消したテストはない。既存のテストで期待値を変えたのは、headless の `director:` の行の末尾(`org_director_headless_test.go`、`org_director_wake_test.go`、`status_test.go`)と、`ralph status` と `director status` の help の文(`status_test.go`、`org_director_test.go`)で、どれも D8 の文の変更に合わせたもの(計画の Progress の S4)
- 知られた揺れのテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は、走らせたすべての回で PASS だった(go1.25.8 0.47 s、org の再実行 0.56 s、`-race` 0.67 s、run-test.sh はパッケージが ok)

## Coverage

- Statement: `-run` で絞った回の `-coverprofile` で、足した関数と動かした関数を見た(数字はパッケージ全体の値ではない)
  - org(`-run 'Director|RunWatchAll|Wake|MarkStopped'`): `director_check.go` は 14 関数のうち 13 が 100%。残りの `CheckDirectorStarter` は org では 0% で、cli の回で 100%。`directorCheckInStarterEnded`、`MarkStoppedAtVersion`、`directorStarter`、`directorCheckFoundEnded`、watch の `checkDirector` と `wake` は 100%。`checkCurrentDirector` 94.1%、`cleanUpEndedDirector` 93.8%、`MarkCmdLaunched` 97.6%、`markStopped` 97.1%、`AttachHeadless` 95.7%、`checkDirectorStart` 89.5%、`WakeDirector` 93.3%、`cycle` 88.9%
  - cli(`-run 'Director|Status|OrgWatch'`、`-coverpkg` に org を含める): `org_director_check.go` の 5 関数、`readOrgDirectorReport`、`orgDirectorLine`、`startError`、`startAttachedError`、`directorLaunched` は 100%、`startDirector` 90.5%。org の `directorHerdrCleanupNotice` は cli の回で 100%
- Branch: 計っていない(Go の標準の道具にない)。新しいコードで通らなかった block は 2 つ
  - `director_start_cleanup.go:38`: start の確かめで `CheckDirector` がエラーを返す(その場で director.jsonl が読めない)
  - `director_start_cleanup.go:96`: launch_cmd の片付けで、`stopDirectorProcess` が終わりを確かめられない(確かめでは gone か zombie と読み、片付けの ps が失敗するか別の答えを返すとき)
  - ほかに通らなかった block(`director_start.go:179`、`:187`、`:452`、`:614` のロックの下の読み込みエラー、`director_wake.go:363` の `mem == nil`、`watch.go:674` の `sweep.Err`)は、前からある道か、この差分で移しただけの道
- Function: 上のとおり
- Notes: 数字は block が通るかを見るために使った

## Test plan coverage

| 計画の項目 | テスト | mutation |
| --- | --- | --- |
| 単体: D3 の確かめの表(偽の herdr、偽の `DirectorProcessReader`) | org の `TestCheckDirector_StartingRows`、`_HerdrRunningRows`、`_NoAgentGet`、`_CmdRunningRows`、`_NothingToCheck`、`_UnreadableDirectorFileIsAnError`。driver の `TestIsAgentNotFound`、`TestHerdr_AgentGet_AgentNotFoundIsReadable` | M14〜M16、M25 は赤 |
| 単体: 印の書き込みと読み取り(id、壊れた印、ない印、一時ファイルと rename) | `TestDirectorAlive_WriteAndReadBack`、`TestReadDirectorAlive_MissingMalformedAndAsRead`、`TestWriteDirectorAlive_AFailedWriteLeavesNoTemporaryFile`、`TestDirectorAliveFor` | M12 は赤 |
| 単体: starter の fold(新しい行、古い行) | `TestDirectorAttachHeadless_RecordsTheStarterOnTheAttachedOfEitherLaunch`、`TestDirectorRead_FoldsTheStarterOfAHeadlessAttachedOnly`、`TestStartDirector_RecordsItsOwnProcessAsTheStarter`、`_LaunchCmdRecordsItsOwnProcessAsTheStarter`、`_ARefusedStartReadsNoProcess` | M19、M20 は赤 |
| 単体: check-in の断りの表(gone、zombie、起動時刻の違い、生きている、ps の失敗、starter なし) | `TestDirectorMarkCmdLaunched_RefusesACheckInOnceTheStartThatRecordedItHasEnded`、`_ReadsTheStarterOnlyAfterItsOtherChecks`、`_ReadsTheStarterWithTheDirectorLockFree`、`_UsesTheStarterItReadOnlyWhileTheRegistrationStillRecordsIt`、`_ReadsTheStarterWithPSWhenNoReaderIsGiven`(本物の ps、自分の子だけ)、`TestStartDirector_ACheckInRefusedForItsStarterRunsNoAgent`、cli の `TestOrgDirectorLaunched_RefusesOnceTheStartThatRecordedItHasEnded` | M07〜M11 は赤 |
| 結合: 見張りのサイクルの順と印 | `TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse`、`_TheDirectorCheckWritesOnlyItsMark`、`_WritesNoDirectorMarkWithoutACurrentHeadlessRegistration`、`_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn`、`_ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn`、cli の `TestOrgWatch_HelpDescribesTheDirectorCheck` | M04〜M06、M21 は赤 |
| 結合: ended に起こす 1 行を送らない、stderr に 1 回 | `TestWakeDirector_ARegistrationTheCheckFoundEndedIsNotWoken`、`_AnUnknownCheckWakesAsBefore`、`_AnEndedCheckOfAnotherRegistrationWakesAsBefore`、`TestRunWatchAll_TheDirectorCheckWritesOnlyItsMark`(3 サイクルで 1 回) | M13、M17 は赤 |
| 結合: start の片付け(agent_ended、pane_closed、process_ended、start_ended)、alive と unknown の断り、昇格したセッションの断り、記録まで届かない片付け | org の `TestStartDirector_CleansUpAnEndedHerdrDirectorAndStarts`、`_CleansUpAnEndedLaunchCmdDirectorAndStarts`、`_CleansUpARegistrationWhoseStartDied`、`_ACleanupIsReportedWhenTheWatchGateRefusesAfterwards`、`_RefusesADirectorItsCheckFindsAliveOrCannotTell`、`_RefusesWithNoCheckWhenThereIsNothingToCheck`、`_ACleanupThatCannotFinishRefusesStart`。cli の `TestOrgDirectorStart_CleansUpAnEndedHeadlessDirectorAndStarts`、`_CleansUpARegistrationWhoseStartDied`、`_RefusesADirectorItsCheckFindsAliveOrCannotTell`、`_ACleanupThatCannotCloseNamesTheStop`、`_ARegistrationChangedSinceTheCheckNamesStatus`、`_AnAttachedRefusalAddsTheCheck`、`_HelpStatesTheCleanup` | M03、M18、M22、C05 は赤 |
| 結合: attach の断りの文、`ralph status` と `director status` の表示と `--json` | cli の `TestOrgDirectorAttach_RefusalAddsTheWatchsMark`、`_RefusalAddsTheStartersCheck`、`_APromotedSessionsRefusalIsUnchanged`、`_HelpStatesTheChecksInTheRefusal`、`TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector`、`_APromotedSessionIsNotChecked`、`_EdgeCasesOfTheCheck` | C01〜C04 は赤 |
| 回帰: 8a と 8b-1 のテストがそのまま通る。見張りは確かめのために director.jsonl を書かず、herdr の close を打たない | 3 回の実行で org と cli の既存のテストがすべて PASS(go1.25.8 の `TestRegate_FetchFailures` の 1 件は下)。書かないこと、get だけを打つことは `TestRunWatchAll_TheDirectorCheckWritesOnlyItsMark` が 3 サイクルで director.jsonl を byte 単位で比べ、herdr の呼び出しを固定する | M06 は 8b-1 の `TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse` でも赤 |

AC ごとには、AC1 は starter の fold の行、AC2 は check-in の行、AC3 は D3 の表と下の差し込みの行、AC4 は見張りの行、AC5 は起こす手順の行、AC6 は start の片付けの行、AC7 は attach と表示の行に当たる。AC8(文書)のうちテストが固定するのは help の文(`TestOrgDirectorStart_HelpStatesTheCleanup`、`TestOrgDirectorAttach_HelpStatesTheChecksInTheRefusal`、`TestOrgWatch_HelpDescribesTheDirectorCheck`)と、run-test.sh の中の `tests/test-check-skill-sync.sh` などで、中身と drift は /verify の担当。AC9 は下の「隔離」。

## Edge cases

| 計画の edge case | テスト | 結果 |
| --- | --- | --- |
| 確かめと start が同じ登録を同時に見る(片付けのあとの印は id が違うので読まれない) | 読む側の照合 `TestDirectorAliveFor`(別の id、同じ id で別の attached_at)と、`TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector`(別の id や attached_at の印は not checked yet) | PASS。見張りと start を同時に走らせるテストはなく、照合の側で見ている |
| 2 つの start の重なり(後の start の読みと ps のあいだに、先の start が `launched` を書いて終わる) | `TestStartDirector_AStartThatFinishedDuringTheCheckIsNotCleanedUp`(herdr と launch_cmd。signal、close、新しい登録がない)、`TestCheckDirector_LaunchedAfterTheReadIsNotReadAsStartEnded` | PASS。M01、M24 が赤 |
| 見張りの確かめの途中で `launched` が書かれる | `TestRunWatchAll_ALaunchedThatLandsDuringTheCheckIsNotMarkedStartEnded` | PASS。M01、M24 が赤 |
| 片付けのあいだに別の start が登録を立てる | `TestStartDirector_TheCleanupActsOnTheRegistrationTheCheckRead`(close は版 A の pane と workspace だけ)、`_ARegistrationThatChangesBeforeTheRecordIsNotRecordedStopped`、`TestMarkStoppedAtVersion_RefusesAVersionThatChanged` | PASS。M03、M18、M23 が赤 |
| 印が壊れている、印の id が古い | `TestReadDirectorAlive_MissingMalformedAndAsRead`、`TestDirectorAliveFor`、`TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector`(parse できない印、印の場所が directory)、`TestOrgDirectorAttach_RefusalAddsTheWatchsMark`(parse できない印は何も足さない) | PASS |
| 見張りが止まっていて印が古い(時刻を出す) | `TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector` と `TestOrgDirectorAttach_RefusalAddsTheWatchsMark` が `checked by the watch at <時刻>` と `the watch checked d1 at <時刻>` を固定する | PASS。古さで文を変える仕組みは計画にない(D8) |
| `agent get` が成功するのに pane の label が違う(alive のまま) | `TestCheckDirector_HerdrRunningRows` の agent_found の行は herdr の呼び出しを `agent_get` だけに固定するので、pane の label は読まれない | PASS。label の違う pane を置いた agent_found の行はなく、呼び出しの固定で間接に見ている |

## Mutation

31 個すべてが赤になった。表は evidence の 5 節にある。依頼された 5 つの結果は次のとおり。

| 依頼 | mutant | 赤にしたテスト |
| --- | --- | --- |
| 確かめの版の結びつき(ロックの下の読み直しを消す) | M01 | `TestCheckDirector_LaunchedAfterTheReadIsNotReadAsStartEnded`、`_ChangedOnBothTriesIsUnknown`、`_ReplacedAfterTheReadIsNothingToCheck`、`_UnreadableDirectorFileIsAnError`、`TestStartDirector_AStartThatFinishedDuringTheCheckIsNotCleanedUp`、`TestRunWatchAll_ALaunchedThatLandsDuringTheCheckIsNotMarkedStartEnded`、`_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn` の 7 本 |
| start の片付けを版 A でなく読み直した登録に打つ | M03 | `TestStartDirector_TheCleanupActsOnTheRegistrationTheCheckRead` |
| 見張りのサイクルで確かめを起こす手順のあとへ | M04 | `TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse`、`_TheDirectorCheckWritesOnlyItsMark`、`_ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn` |
| check-in が終わった starter を受け付ける | M07 | `TestDirectorMarkCmdLaunched_RefusesACheckInOnceTheStartThatRecordedItHasEnded` ほか 4 本 |
| 表示の印の照合を id だけに | M12(org)、C01(cli) | org は `TestDirectorAliveFor`、cli は `TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector`、`TestOrgDirectorAttach_RefusalAddsTheWatchsMark` |

- 依頼の 5 つのほかに、順の別の形(M05 期限の前、M06 パルスのあと)、check-in の別の形(M08〜M11)、照合の別の場所(M13 起こす手順)、D3 の行(M14〜M16、M25)、記録の版の比較(M18、M23、M26)、試行の回数(M24)、表示と断りの分岐(C02〜C05)も赤になった
- 1 本のテストだけで赤になったもの: M02(読み直しのロックを外す)は `TestRunWatchAll_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn` が開けない director.lock で確かめを失敗させる道でだけ捕まる。M26(版の `folded` を空にする)は `TestDirectorVersionOf` が field を直に見ることでだけ捕まる。どちらも下の Test gaps に書く
- どの 1 本だけの赤も、落ちた assertion を読んで、決めた挙動の違いであり時間切れや揺れではないことを確かめた

## 隔離(AC9)

- この機械には `herdr`、`gh`、`tmux`、`orca`(`/opt/homebrew/bin`)と `osascript`、`open`(`/usr/bin`)が入っている。同じ名前(ほかに `agmsg`、`codex`、`claude`、`curl`)の canary を、呼ばれたら記録して exit 1 する script として PATH の先頭に置き、新しいテスト 67 本を流した。67 本すべて PASS、canary の呼び出しは 0 だった
- 感度の確かめ: `setupOrgStubPATH` が stub の dir を PATH に足さない overlay で cli の新しいテストを流すと、4 本が落ち、canary は 5 回(すべて `herdr workspace create`)呼ばれた。stub を外れた呼び出しは canary に届き、HEAD では届かない
- ps は自分の pid と、テストが起こして待った子にだけ打つ。signal は `TestDirectorMarkCmdLaunched_ReadsTheStarterWithPSWhenNoReaderIsGiven` が自分の起こした `sleep` に送る `Kill` だけ(差分の `+` 行を grep した)
- 実行のあいだ、worktree に `.harness/state/org/` と `director-alive.json` は現れず、`.git/ralph/` と main の checkout の `.harness/state/org/` に変わったファイルはなかった(`find -newermt`。変わったのは hook の `edited-files.log` などだけ)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(org、`regate_test.go:771`、go1.25.8 の全体の回) | 期待した「ran past 300ms and was stopped」でなく、`git fetch` が `exit 128`(`Could not read from remote repository`)で返った | 知られた揺れ(tester の memory に 2026-10-18 から記録)。重い Go の実行を 2 つ並べたときに起き、原因は未確認(I/O とプロセス生成の取り合いと推測)。`internal/org/regate*.go` はこの差分で変わっていない | 直さない。同じ toolchain で単独 10/10 PASS、org パッケージ全体の再実行も PASS。この差分の外の tech-debt |

この差分によるテストの失敗はない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 8a と 8b-1 の attach、start、stop、check-in、escalate、期限、起こす手順 | PASS | 3 回の実行で org と cli の既存のテストがすべて PASS(上の 1 件を除き、それも再実行で PASS) |
| 見張りの起こす手順が期限の送りのあと、パルスの前(8b-1) | PASS | `TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse` が M06 で赤 |
| stop が版 A の比較を持たない記録(`MarkStoppedIfUnchanged`)を今どおり書く | PASS | `TestMarkStoppedAtVersion_UnchangedWritesWhatMarkStoppedIfUnchangedWrites` が同じ byte を書くことを固定する。`director_stop_test.go` は変わらず PASS |
| 揺れで知られた `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` | PASS | 走らせた 4 回すべて |

## Test gaps

- `director_start_cleanup.go:38`(start の確かめで director.jsonl が読めない)と `:96`(launch_cmd の片付けで、確かめが ended と読んだ process の終わりを片付けの ps が確かめられない)を通るテストがない。後者は偽の reader に、1 回目は gone、2 回目はエラーを返させれば書ける。どちらも記録を書かずに断る道で、LOW
- M02 を捕まえるのは、開けない director.lock で確かめを失敗させる道だけで、書き手が director.lock を持って書いている途中に確かめが読み直す、という並びのテストはない。LOW
- M26(`folded` の比較)は white-box の `TestDirectorVersionOf` でだけ捕まる。event の数を変えずに手で書き換えた director.jsonl を確かめと片付けがどう扱うかの挙動のテストはない。計画が「手で書き換えたときの第 2 の守り」と書く分で、INFO
- 見張りと start、2 つの start を本当に同時に走らせるテストはない。重なりは差し込みの関数で順を決めて確かめる作りで、決まった順は上の表のとおり赤と緑で確かめた
- Linux(procps)の `ps -o stat=,lstart=` は試していない。本物の ps を読むテスト(`_ReadsTheStarterWithPSWhenNoReaderIsGiven`、`_LaunchCmdRecordsItsOwnProcessAsTheStarter`、cli の `deadChildPID` を使うもの)は ps のない環境では落ちる。`deadChildPID` はテストの中で pid が使い回されないことを前提にしており、おそらく起きないが未確認
- launch_cmd の director と push の起こし口の飛ばし(D5)は、実機ではなくテストだけが見ている(verify report のとおり)
- insight event は、この task の指示(テストの report と evidence だけをコミットする)により追記していない

## Verdict

- Verdict: pass
- Pass: run-test.sh(shell 41 本と Go 8 パッケージ)、go1.25.8 の全体(1 件の知られた揺れのほかすべて PASS、その 1 件は単独 10/10、org パッケージの再実行も PASS)、`-race`(550 本と 5 本、DATA RACE 0)、新しいテスト 67 本、mutant 31 個すべてが赤、canary の呼び出し 0(感度の確かめつき)
- Fail: なし(この差分によるもの)
- Blocked: なし
