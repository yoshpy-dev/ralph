# Test report: org-regate

- Date: 2026-10-17(計画の日付は 2026-10-16)
- Plan: docs/plans/active/2026-10-16-org-regate.md
- Tester: tester subagent (Claude)。パイプライン 1 回目
- Scope: `git diff 670ec219...HEAD`(HEAD f8638904)。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。コードとテストは変えていない
- Evidence: `docs/evidence/test-2026-10-16-org-regate.log`(4 つのコマンドの生の出力、新しいファイルの関数ごとと文ごとのカバレッジ、覆われていないブロックとその行、AC12 の grep、手で打った probe。`~` は `$HOME`、`$SCRATCH` は session の scratchpad)。wrapper 自身のログは `docs/evidence/verify-2026-10-16-161810.log`。どちらも gitignore の対象

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は changed、言語は golang) | shell 41 本 + Go 8 パッケージ | 全部 | 0 | shell の 1 件(下の注) | 765 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,258 | 41 本とも exit 0 | 0(`FAIL:` の行は 25 本の集計の `FAIL: 0` だけ) | 1 件 | 上に含む |
| └ `go test ./...`(go1.26.0) | 8 パッケージ | 8 | 0 | - | cli 270.9 s、org 272.9 s。insights・driver・protocol・scaffold・upgrade は `(cached)` |
| `go test -race ./internal/org/ ./internal/cli/ -run 'Regate\|Gate\|MergeOrder\|Spawn\|StartFeature' -count=1 -v`(go1.26.0) | 377 本(org 286、cli 91)、subtest を入れて 1,025 | 1,025 | 0 | 0 | 270 s(org 261.7 s、cli 108.9 s)。`DATA RACE` はなし |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1` | 8 パッケージ | 8 | 0 | - | 248 s(cli 237.3 s、org 244.6 s) |
| `go test -count=1 -cover -coverprofile ./internal/org/ ./internal/cli/`(go1.26.0) | 2 パッケージ | 2 | 0 | - | 305 s(wrapper の shell と同時に走らせたので遅い) |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の case を飛ばし、偽の古い git の case が代わりに見る。この変更とは関係がない
- 足したテスト 61 本(`internal/org/regate_test.go` 21、`regate_merge_test.go` 18、`internal/cli/org_regate_test.go` 11、`spawn_test.go` 3、`feature_test.go` 1、`mergeorder_test.go` 5、`gate_test.go` 2)は、`-race` の実行で全部 `--- PASS` になった(名前の一覧と `--- PASS` の行を `comm` で突き合わせ、欠けは 0)
- 既知の flaky の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は、`internal/org` を丸ごと走らせた 3 回(wrapper、カバレッジ、go1.25.8)のどれでも落ちなかった。打ち直しはしていない
- テストのあとの worktree: `git status --porcelain` は空。ignore された中にも、テストが残した `.harness/state/org` や `gates/` はない(evidence の 5 節)

## AC とテストの対応

| AC | テスト(file:line) |
| --- | --- |
| AC1 断る | `TestRegate_RefusedBeforeTheFetch`(`internal/org/regate_test.go:391`、19 の case。記録、worktree、origin、スクリプトの呼び出し、`regate.lock` の解放を毎回見る)、`TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns`(`:585`)、`TestRegate_RefusedWhileAnotherRegateHoldsTheLock`(`:618`、D7 の 1 秒の打ち直しを経過時間で見る)、`TestOrgRegate_RefusalPrintsNothingAndChangesNothing`(`internal/cli/org_regate_test.go:326`)、`TestOrgRegate_OrgIDRefusedBranchRequiredNoArguments`(`:372`)、`TestOrgRegate_LegacyLedgerGuard`(`:669`) |
| AC2 fetch と HEAD | `TestRegate_FetchFailures`(`regate_test.go:712`、origin がない、origin にブランチがない、既定のブランチがない、120 秒の上限)、`TestRegate_AFetchEndedByASignalRecordsNothing`(`:781`)、`TestRegate_RefusesAHeadThatIsNotOriginsTip`(`:807`)、`TestRegateRemoteEnv`(`:863`)、`TestRegate_FetchRunsWithNoTerminalPrompt`(`:914`)。push の `GIT_TERMINAL_PROMPT=0` は `TestRegate_MergesTheTipInAndPushes`(`internal/org/regate_merge_test.go:31`) |
| AC3 何もしない道と壊れた記録 | `TestRegate_UpToDateDoesNothing`(`regate_test.go:946`)、`TestRegate_GatesWhenNoRecordLetsItStop`(`:994`。壊れた Details と壊れた JSON 行の 2 case で、取り直したあと merge-order が `ready`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`) |
| AC4 取り込みの道 | `TestRegate_MergesTheTipInAndPushes`(`regate_merge_test.go:31`)、`TestRegate_PointsTheRemoteTrackingRefAtThePush`(`:111`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`、stdout の 6 行と merge-order の `ready`) |
| AC5 merge commit を作れない | `TestRegate_RefusedWhenTheMergeCommitCannotBeMade`(`regate_merge_test.go:153`、衝突 2 つ、pre-merge-commit の hook、signal、名前がない)、`TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack`(`:255`)、`TestRegate_AMergeEndedAfterItsCommitNamesResetKeep`(`:303`)、`TestRegate_AMergeCommitItCannotReadNamesResetKeep`(`:354`)、`TestRegate_AnotherCommitAfterTheMergeNamesResetKeep`(`:434`) |
| AC6 スクリプトの失敗と中断 | `TestRegate_AScriptFailureAfterTheMergeUndoesIt`(`regate_merge_test.go:493`)、`TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork`(`:548`、`head_moved`、`tree_changed`、`reset --keep` の失敗)、`TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge`(`:644`)、`TestOrgRegate_FailureAfterAMergeUndoesItAndExits1`(`org_regate_test.go:222`)、`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing`(`:506`、SIGINT・SIGTERM・SIGHUP)、`TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits`(`:594`)、`TestRegate_Interrupted`(`regate_test.go:1364`) |
| AC7 push の失敗と打ち直し | `TestRegate_APushThatFailsKeepsTheRecordAndTheMerge`(`regate_merge_test.go:843`)、`TestRegate_APushThatLandedWhoseAnswerWasLost`(`:898`)、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged`(`:923`)、`TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge`(`:988`)、`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`(`:1026`)、`TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt`(`:1065`)、`TestRegate_AnUnreadableLatestRecordGivesNoPushOnlyAndNoReset`(`:1103`)、`TestOrgRegate_AFailedPushIsPushedOnlyNextTime`(`org_regate_test.go:266`) |
| AC8 結びつきが変わった | `TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan`(`regate_test.go:1167`)、`TestRegate_ARefusedRecordAfterTheMergeUndoesIt`(`regate_merge_test.go:796`) |
| AC9 gate だけの道 | `TestRegate_GatesWhenNoRecordLetsItStop`(`regate_test.go:994`)、`TestRegate_GateOnlyRecordsAGatePassedViaRegate`(`:1057`)、`TestRegate_RecordsWithTheOrgOfTheLatestBinding`(`:1100`)、`TestRegate_GateOnlyFailuresRecordGateFailed`(`:1116`)、`TestOrgRegate_GateOnlyPassPrintsNoMergeAndNoPush`(`org_regate_test.go:188`) |
| AC10 merge-order の理由の欄 | `internal/org/mergeorder_test.go` の `TestMergeOrder_DetailsNameRegateForABranchWhoseOrgIsNotRunning`(`:558`)、`…DetailsOfARunningOrgOrNoBindingAreUnchanged`(`:630`)、`…DetailsOfABranchWhoseOrgRunsAnotherFeature`(`:713`)、`…SeveralReadyNoteNamesRegateForADisbandedOrg`(`:800`)、`…GateViaIsInTheEntryAndItsJSON`(`:829`)。`TestMergeOrder_LatestGateRecordWins`(`:861`)の壊れた行の注は、部分一致から全文一致に強められた |
| AC13 Spawn との排他 | `TestOrgSpawn_RefusedWhileRegateRunsOnTheOrg`(`internal/org/spawn_test.go:4584`、最初の確かめのあとと記録の直後の hook)、`TestOrgSpawn_RegateLockStates`(`:4616`)、`TestOrgSpawn_RegateLockThatCannotBeCheckedRefuses`(`:4663`)、`TestStartFeature_RefusedWhileRegateRunsOnTheOrg`(`internal/org/feature_test.go:1246`)、regate の側は `TestRegate_MarksTheLockWithTheOrgItWorksOn`(`regate_test.go:1213`)、`TestRegate_EmptiesAnEarlierMarkerBeforeTheManifestLock`(`:1258`)、`TestRegate_ARefusedFirstCheckLeavesNoMarker`(`:1301`)、`TestReadRegateMarker`(`:1335`) |
| D6 の `via=regate` と共有の helper | `TestGateRecordDetails_Via`(`internal/org/gate_test.go:1463`)、`TestGateGitHelpers_ErrorsStartWithTheCallersPrefix`(`:773`)、`TestRegate_ErrorsOfTheSharedHelpersSayRegate`(`regate_test.go:1461`)、`TestGateWording_RefusalsThatAskForACommit`(`:654`) |
| AC11 文書 | 自動のテストは CLI の help だけ(`TestOrgRegate_Help`、`org_regate_test.go:723`)。skill の 8 面の一致は `check-skill-sync.sh`(/verify の受け持ち)で、skill の文の中身を固定するテストはない |
| AC12 テストの隔離と 2 つの版 | テストの性質なので、下の「AC12 の確かめ」で見た |

### AC12 の確かめ

- 新しい 3 つのテストファイルに URL、`ssh://`、`git@`、`net.Dial`、`http.` はない。`github.com` は import の 1 行だけ。origin はどれも `t.TempDir()` の下の bare repository(`newRegateFixture`、`regate_test.go:52`、`newOrgRegateFixture`、`org_regate_test.go:46`)
- `herdr` の字は、記録の Details の `herdr_pane=unset` と、環境から `HERDR_PANE_ID` を外す行だけ。`osascript`、`tmux`、端末のアプリの名前はない。regate のコード(`regate*.go`、`org_regate.go`)が `exec` で動かすのは `git` だけで(`regate_git.go:177`)、ほかはテストの stub の gate スクリプト。CLI の merge-order は gh のない PATH で打ち(`org_regate_test.go:174`)、古い台帳のテストは stub の PATH で打つ(`:670`)。spawn と start のテストは偽の herdr と agmsg を使い、足した行に `exec.Command` はない
- signal を送る先: git の hook の `kill -KILL $PPID` は、regate がテストの中で起こした git(fetch、merge、push)を殺す。`signalSelf` はテストのプロセス自身に送る。`syscall.Kill(-pid, SIGKILL)`(`org_regate_test.go:535`)はテストが落ちたときの後始末だけで、相手はテストが起こした stub のスクリプトの pid。ほかのプロセスに signal を送る行はない
- git の設定は `isolateGitEnv` と `isolateFeatureGitEnv` で global と system を外し、fixture の repository では自動の maintenance と gc を止めている
- go1.26.0(wrapper とカバレッジの実行)と `GOTOOLCHAIN=go1.25.8`(全体を uncached で)の両方で通った

## Coverage

- Statement: `internal/org` 94.2%、`internal/cli` 88.1%(go1.26.0、`-count=1`)。前の計測(2026-10-16、org-director-launch-cmd)は org 94.7%、cli 88.0%
- 新しいファイルの文の割合(cover.out から数えた): `internal/org/regate.go` 89.4%(185/207)、`internal/org/regate_merge.go` 84.3%(167/198)、`internal/org/regate_git.go` 84.5%(60/71)、`internal/cli/org_regate.go` 98.1%(53/54)
- Branch: Go の標準のツールは分岐のカバレッジを出さない。覆われていないブロック 61 個を evidence の 4c 節に行ごとに書いた
- Function: 100% でない関数は次のとおり。`Regate` 89.6%、`regateFirstCheck` 93.8%、`writeRegateMarker` 72.7%、`regateCheckWorktree` 87.5%、`regateCheckInProgress` 81.8%、`regateGate` 92.0%、`recordRegate` 85.2%、`regateFetch` 87.5%、`regateCheckOriginHead` 76.9%、`regateIsAncestor` 66.7%、`regateBoundedGit` 84.0%、`regateHeadNotOnOrigin` 78.6%、`regateParents` 85.7%、`regateMerged` 93.9%、`regateCheckIdentity` 71.4%、`regateMergeTip` 91.2%、`regateAbortLeftMerge` 72.7%、`regateUndoMerge` 69.6%、`regateUndoneText` 66.7%、`regatePush` 76.5%、`regateSetTrackingRef` 80.0%、`newOrgRegateCmd` 94.4%。ほかの 20 関数は 100%
- Notes: 覆われていないブロックの多くは、git や manifest が読めないときの error の返し(`regateInterrupted(ctx, err, "")` の枝など)と、`regateMerged` の `default`(`regate_merge.go:185`、今の理由では来ないとコメントが書く)、CLI の `parent == nil`(`org_regate.go:128`)。振る舞いとして意味のあるものは「Test gaps」に書いた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `ralph org gate` の既存のテスト(記録の形は regate の記録にだけ `via` が付く) | pass | `-race` の `Gate` に当たるテストと 2 回の全体の実行。既存のテストの書き換えは `resolveGateBase` に接頭辞の引数を足した 2 行だけ(`gate_test.go:745`、`:763`) |
| merge-order の既存のテスト(走っている org の理由の欄は変わらない) | pass | `TestMergeOrder_DetailsOfARunningOrgOrNoBindingAreUnchanged` が今までの文を全文で比べる。既存のテストで変わったのは `TestMergeOrder_LatestGateRecordWins` の 1 か所で、部分一致から全文一致になり、弱めてはいない |
| Spawn と start の既存のテスト(regate が走っていなければ共有のロックはすぐ取れて放す) | pass | `spawn_test.go` と `feature_test.go` は足しただけで、消した行は 0。`-race` の `Spawn` と `StartFeature` に当たるテストが全部通った |
| 既知の flaky(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | 落ちなかった | `internal/org` の全体を 3 回走らせて 3 回とも ok |

## Test gaps

1. 既定のブランチが HEAD の子孫(機能のコミットがない)の場合を見る自動のテストがない。計画の Test plan の Edge cases に挙がっている。HEAD の binary とローカルの bare origin で打った(evidence の 6 節)。`--no-ff` の merge commit を作り(親は前の HEAD と取り込んだ先頭、木は origin の main と同じ)、gate を記録して push し、終了コード 0。merge-order は `ready`、打ち直すと `UP_TO_DATE:`。PR の差分は空になるが、D3 の手順どおりの結果で、断るべきだとは計画に書かれていない
2. fetch か push が走っている間に ctx が終わる場合(`regate_git.go:197` と `regate_merge.go:485`)を通すテストがない。push の途中の中断は、pre-push の hook が git を殺す case(`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`)が同じ `regatePushInterrupted(during=true)` の文を通すので、出力は固定されている。端末の Ctrl-C は git にも ralph にも届くので、どちらの枝に入るかは早さで決まる
3. 120 秒の上限は fetch だけがテストで縮めて通す(`TestRegate_FetchFailures` の「the fetch runs past the bound」)。push の上限(`regate_merge.go:491`、`PushFailed` に「origin may not have it」)、merge の上限(`:272`、`:287`)、`git var` の上限(`:210`)、`reset --keep` の上限(`:433`)は通らない。push の上限は fetch と同じく `regateGitTimeout` を縮めれば打てる
4. `git merge --abort` そのものが失敗する場合(signal、走らない、exit が 0 でない)と、abort のあとに HEAD か作業ツリーが読めない場合(`regateAbortLeftMerge`、`regate_merge.go:337-364`)を通すテストがない。AC5 の「HEAD と作業ツリーが前のままであることを確かめ」の確かめが失敗した側の文は固定されていない
5. `regateUndoMerge` で HEAD か作業ツリーが読めない、`reset --keep` が signal で止まる、走らない、上限を超える場合(`regate_merge.go:408-435`)は通らない。`reset --keep` が exit 1 で落ちる場合は通る
6. AC11 の skill の文の中身(`/org` と `/director` の regate の手順)を固定するテストはない。4 面の一致は /verify の `check-skill-sync.sh` が見る
7. 端末の本物の Ctrl-C、ssh の origin、本物の GitHub への push、Linux での実行は確かめていない(テストは macOS の go1.26.0 と go1.25.8)
8. 計画の Evidence に挙がる「mutant の赤」は、この /test では作っていない

## Verdict

- Pass: pass。`./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、`-race` の絞った実行(1,025 件)、`GOTOOLCHAIN=go1.25.8` の全体の実行がどれも exit 0 で、失敗は 0。足したテスト 61 本が全部通り、AC1〜AC10 と AC13 にそれぞれテストがある
- Fail: なし
- Blocked: なし。AC11 は文書の AC で、自動のテストは CLI の help だけ。上の Test gaps は、どれも PR を止めるものではない
