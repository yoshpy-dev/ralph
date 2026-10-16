# Test report: org-regate

- Date: 2026-10-17(計画の日付は 2026-10-16)
- Plan: docs/plans/active/2026-10-16-org-regate.md
- Tester: tester subagent (Claude)。パイプライン 2 回目
- Scope: `git diff 670ec219...HEAD`(HEAD f2f65a30)。1 回目の報告(6e67e718、HEAD f8638904)のあとに入ったのは、既定のブランチが HEAD の子孫の場合のテスト(5e4ce6cd)、merge の前のパスの確かめ(D3、85558f2c、ca840a45、c621ccc9)、読めない gate の記録を org のすべてのブランチに付ける直し(D2、f793924d)、help と注の直し。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。コードとテストは変えていない
- Evidence: `docs/evidence/test-2026-10-16-org-regate.log`(4 つのコマンドの生の出力、regate のファイルと mergeorder.go の関数ごとと文ごとのカバレッジ、覆われていないブロックとその行、足したテストの突き合わせ、AC12 の grep、大文字小文字を区別する FS での実行、HEAD の binary の probe と 1 回目の binary の対照、テストのあとの worktree。`~` は `$HOME`、`$SCRATCH` は session の scratchpad、`$TMPDIR` は macOS の一時ディレクトリ)。wrapper 自身のログは `docs/evidence/verify-2026-10-16-200358.log`。どちらも gitignore の対象

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は changed、言語は golang) | shell 41 本 + Go 8 パッケージ | 全部 | 0 | shell の 1 件(下の注) | 716 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,283 | 41 本とも OK | 0(`FAIL:` で始まる集計はすべて 0) | 1 件 | 上に含む |
| └ `go test ./...`(go1.26.0) | 8 パッケージ | 8 | 0 | - | cli 234.1 s、org 275.6 s、config 2.4 s、insights 1.9 s。driver・protocol・scaffold・upgrade は `(cached)` |
| `go test -race ./internal/org/ ./internal/cli/ -run 'Regate\|Gate\|MergeOrder\|Spawn\|StartFeature' -count=1 -v`(go1.26.0) | 383 本(org 292、cli 91)、subtest を入れて 1,054 | 1,054 | 0 | 0 | 274 s(org 271.1 s、cli 94.5 s)。`DATA RACE` はなし。wrapper の shell の部分と同時に走らせた |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1` | 8 パッケージ | 8 | 0 | - | 275 s(cli 233.6 s、org 271.1 s。8 パッケージとも uncached) |
| `go test -count=1 -coverprofile ./internal/org/ ./internal/cli/`(go1.26.0) | 2 パッケージ | 2 | 0 | - | 312 s(org 308.7 s、cli 244.0 s) |
| 手で足した確かめ: `TMPDIR` を大文字小文字を区別する APFS のイメージに置き、cycle 2 で足した `TestRegate_` 6 本を `-v` で走らせた | 6 本、subtest を入れて 26 | 20 | 0 | 6(下の注) | 42 s |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の case を飛ばし、偽の古い git の case が代わりに見る。この変更とは関係がない
- 足したテストは 70 本(`internal/org/regate_test.go` 21、`regate_merge_test.go` 24、`internal/cli/org_regate_test.go` 11、`spawn_test.go` 3、`feature_test.go` 1、`mergeorder_test.go` 8、`gate_test.go` 2)。1 回目の 61 本に、5e4ce6cd の 1 本、D3 の 5 本、D2 で名前と期待を変えた merge-order の 3 本が加わった。名前の一覧と `-race` の実行の `--- PASS` の行を `comm` で突き合わせ、欠けは 0(evidence の 5 節)
- 大文字小文字の行: `TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay` の 3 行(`…spelled_in_another_case`)と `TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay` の 3 行(`…renames_only_in_case` 2 行と `…turns_into_a_directory_spelled_in_another_case`)は、`regateSkipOnACaseSensitiveFS`(`regate_merge_test.go:1239`)が `t.TempDir()` で `a` を作り `A` を lstat して決める。この機械の `$TMPDIR` は大文字小文字を区別しない APFS なので、wrapper、`-race`、go1.25.8、カバレッジのどの実行でも 6 行とも走って通り、SKIP は 0 だった。区別する FS では 6 行とも SKIP になることを、`hdiutil` で作った Case-sensitive APFS のイメージに `TMPDIR` を置いて確かめた(SKIP 6、ほかの 20 件は PASS。evidence の 7 節)。CI は `ubuntu-latest` で区別する FS なので、CI ではこの 6 行は走らない
- 既知の flaky の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は、`internal/org` を丸ごと走らせた 3 回(wrapper、go1.25.8、カバレッジ)のどれでも落ちなかった。打ち直しはしていない
- テストのあとの worktree: `git status --porcelain` は空。ignore された中で変わったのは、wrapper の evidence の世代管理が古い `verify-2026-10-16-121647.log` を消して `verify-2026-10-16-200358.log` を書いたことだけで、テストが残した `.harness/state/org` や `gates/` はない(evidence の 9 節)。main のチェックアウトの `git status --porcelain` も空

## AC とテストの対応

| AC | テスト(file:line) |
| --- | --- |
| AC1 断る | `TestRegate_RefusedBeforeTheFetch`(`internal/org/regate_test.go:391`、19 の case。記録、worktree、origin、スクリプトの呼び出し、`regate.lock` の解放を毎回見る)、`TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns`(`:585`)、`TestRegate_RefusedWhileAnotherRegateHoldsTheLock`(`:618`、D7 の 1 秒の打ち直しを経過時間で見る)、`TestOrgRegate_RefusalPrintsNothingAndChangesNothing`(`internal/cli/org_regate_test.go:326`)、`TestOrgRegate_OrgIDRefusedBranchRequiredNoArguments`(`:372`)、`TestOrgRegate_LegacyLedgerGuard`(`:669`) |
| AC1 の D3(merge が書くパスに ignored のもの) | `TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay`(`internal/org/regate_merge_test.go:1290`、12 の case。tip が足すパス、HEAD が消して tip が変えたパス、ディレクトリになるパスのファイル、ファイルになるパスのディレクトリ、symlink、11 個目からの「and 2 more」、両側の directory rename の移った先、file/directory の衝突の `d~HEAD`、綴りだけ違う 3 行。断りの全文、記録と worktree と origin、残したファイルの中身、`MERGE_HEAD` がないことを見る)、`TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay`(`:1501`、断らない 6 の case)、`TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch`(`:1473`)、`TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse`(`:1581`)、`TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree`(`:1620`、2.38 より前の git の代わりの道と、落ちる merge-tree)。CLI の終了コード 1 と stderr は HEAD の binary の probe で確かめた(下の「Regression checks」) |
| AC2 fetch と HEAD | `TestRegate_FetchFailures`(`regate_test.go:712`、origin がない、origin にブランチがない、既定のブランチがない、120 秒の上限)、`TestRegate_AFetchEndedByASignalRecordsNothing`(`:781`)、`TestRegate_RefusesAHeadThatIsNotOriginsTip`(`:807`)、`TestRegateRemoteEnv`(`:863`)、`TestRegate_FetchRunsWithNoTerminalPrompt`(`:914`)。push の `GIT_TERMINAL_PROMPT=0` は `TestRegate_MergesTheTipInAndPushes`(`regate_merge_test.go:31`) |
| AC3 何もしない道と壊れた記録 | `TestRegate_UpToDateDoesNothing`(`regate_test.go:946`)、`TestRegate_GatesWhenNoRecordLetsItStop`(`:995`。壊れた Details、壊れた JSON 行、org_id を別のブランチに結び直したあとの読めない記録(D2)の 3 case で、取り直したあと merge-order が `ready`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`) |
| AC4 取り込みの道 | `TestRegate_MergesTheTipInAndPushes`(`regate_merge_test.go:31`)、`TestRegate_PointsTheRemoteTrackingRefAtThePush`(`:111`)、`TestRegate_MergesADefaultBranchTipThatAlreadyContainsTheBranch`(`:132`、既定のブランチが HEAD の子孫。親が前の HEAD と tip、木が tip と同じ merge commit、`via=regate` の `gate_passed`、push、merge-order の `ready`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`、stdout の 6 行と merge-order の `ready`) |
| AC5 merge commit を作れない | `TestRegate_RefusedWhenTheMergeCommitCannotBeMade`(`regate_merge_test.go:210`、衝突 2 つ、pre-merge-commit の hook、signal、名前がない)、`TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack`(`:312`)、`TestRegate_AMergeEndedAfterItsCommitNamesResetKeep`(`:360`)、`TestRegate_AMergeCommitItCannotReadNamesResetKeep`(`:411`)、`TestRegate_AnotherCommitAfterTheMergeNamesResetKeep`(`:491`)。共通の祖先がない tip を git merge が断る場合は `TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse`(`:1581`) |
| AC6 スクリプトの失敗と中断 | `TestRegate_AScriptFailureAfterTheMergeUndoesIt`(`regate_merge_test.go:550`)、`TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork`(`:605`、`head_moved`、`tree_changed`、`reset --keep` の失敗)、`TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge`(`:701`)、`TestOrgRegate_FailureAfterAMergeUndoesItAndExits1`(`org_regate_test.go:222`)、`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing`(`:506`、SIGINT・SIGTERM・SIGHUP)、`TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits`(`:594`)、`TestRegate_Interrupted`(`regate_test.go:1373`) |
| AC7 push の失敗と打ち直し | `TestRegate_APushThatFailsKeepsTheRecordAndTheMerge`(`regate_merge_test.go:900`)、`TestRegate_APushThatLandedWhoseAnswerWasLost`(`:955`)、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged`(`:980`)、`TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge`(`:1045`)、`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`(`:1083`)、`TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt`(`:1122`)、`TestRegate_AnUnreadableLatestRecordGivesNoPushOnlyAndNoReset`(`:1160`)、`TestOrgRegate_AFailedPushIsPushedOnlyNextTime`(`org_regate_test.go:266`) |
| AC8 結びつきが変わった | `TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan`(`regate_test.go:1176`)、`TestRegate_ARefusedRecordAfterTheMergeUndoesIt`(`regate_merge_test.go:853`) |
| AC9 gate だけの道 | `TestRegate_GatesWhenNoRecordLetsItStop`(`regate_test.go:995`)、`TestRegate_GateOnlyRecordsAGatePassedViaRegate`(`:1066`)、`TestRegate_RecordsWithTheOrgOfTheLatestBinding`(`:1109`)、`TestRegate_GateOnlyFailuresRecordGateFailed`(`:1125`)、`TestOrgRegate_GateOnlyPassPrintsNoMergeAndNoPush`(`org_regate_test.go:188`) |
| AC10 merge-order の理由の欄 | `internal/org/mergeorder_test.go` の `TestMergeOrder_DetailsNameRegateForABranchWhoseOrgIsNotRunning`(`:558`)、`…DetailsOfARunningOrgOrNoBindingAreUnchanged`(`:630`)、`…DetailsOfABranchWhoseOrgRunsAnotherFeature`(`:713`)、`…SeveralReadyNoteNamesRegateForADisbandedOrg`(`:800`)、`…GateViaIsInTheEntryAndItsJSON`(`:829`)。`TestMergeOrder_LatestGateRecordWins`(`:861`)の壊れた行の注は全文一致で比べる |
| D2 読めない記録を org のすべてのブランチに付ける | `TestMergeOrder_UnreadableRecordAttachesToEveryBranchOfItsOrg`(`mergeorder_test.go:982`、2 case。どちらのブランチも `unreadable` で、Detail が「同じ org のほかのブランチのものかもしれない」とその org にブランチが 2 本あったことを言う全文を比べる)、`…UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranches`(`:1036`)、`…UnreadableRecordAttachesToTheBoundAndTheGatedBranch`(`:1073`)、`…UnreadableRecordAttachesToTheBindingBeforeIt`(`:943`、ブランチが 1 つなら前の規則と同じ分類) |
| AC13 Spawn との排他 | `TestOrgSpawn_RefusedWhileRegateRunsOnTheOrg`(`internal/org/spawn_test.go:4584`、最初の確かめのあとと記録の直後の hook)、`TestOrgSpawn_RegateLockStates`(`:4616`)、`TestOrgSpawn_RegateLockThatCannotBeCheckedRefuses`(`:4663`)、`TestStartFeature_RefusedWhileRegateRunsOnTheOrg`(`internal/org/feature_test.go:1246`)、regate の側は `TestRegate_MarksTheLockWithTheOrgItWorksOn`(`regate_test.go:1222`)、`TestRegate_EmptiesAnEarlierMarkerBeforeTheManifestLock`(`:1267`)、`TestRegate_ARefusedFirstCheckLeavesNoMarker`(`:1310`)、`TestReadRegateMarker`(`:1344`) |
| D6 の `via=regate` と共有の helper | `TestGateRecordDetails_Via`(`internal/org/gate_test.go:1463`)、`TestGateGitHelpers_ErrorsStartWithTheCallersPrefix`(`:773`)、`TestRegate_ErrorsOfTheSharedHelpersSayRegate`(`regate_test.go:1470`)、`TestGateWording_RefusalsThatAskForACommit`(`:654`) |
| AC11 文書 | 自動のテストは CLI の help だけ(`TestOrgRegate_Help`、`org_regate_test.go:723`)。skill の 8 面の一致は `check-skill-sync.sh`(/verify の受け持ち)で、skill の文の中身を固定するテストはない |
| AC12 テストの隔離と 2 つの版 | テストの性質なので、下の「AC12 の確かめ」で見た |

### AC12 の確かめ

- 1 回目のあとに変わったテストファイル(`regate_merge_test.go`、`regate_test.go`、`mergeorder_test.go`)の足した行に、URL、`ssh://`、`git@`、`net.Dial`、`http.`、`osascript`、`tmux`、端末のアプリの名前はない。`herdr` の字は記録の Details の `herdr_pane=unset` の 1 行だけ。signal を送る行(`Kill`、`Signal`、`SIGKILL`、`SIGTERM`、`kill`)は 0(evidence の 6 節)
- origin はどれも `t.TempDir()` の下の bare repository(`newRegateFixture`、`regate_test.go:52`)。fixture は `isolateGitEnv` で global と system の git の設定を外し、自動の maintenance と gc を止める。`excludeLocally` が書く `.git/info/exclude` は fixture の main のチェックアウトのもの。`TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse` の `commit-tree` は fixture の bare origin の中で `-c user.name=t -c user.email=t@example.com` を付けて打つ
- `TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree` の PATH の wrapper(`regateWrapGit`、`regate_merge_test.go:1599`)は、`t.TempDir()` の下の `bin/git` を `t.Setenv("PATH", …)` で PATH の先頭に置く。`t.Setenv` は subtest の終わりに PATH を戻し、`t.Parallel` とは併用できないので、ほかのテストに漏れない。wrapper は引数に `merge-tree` があるときだけ引数を記録して与えられた shell の断片を走らせ、それ以外は wrap の前に `exec.LookPath` で見つけた本物の git に `exec` する。git のほかは何も包まない
- 足した行の `exec.Command` は、fixture の worktree で `git merge-base --is-ancestor` を打つ前提の確かめの 1 行だけ。regate のコードが `exec` で動かすのは `git` だけで(`regate_git.go:177`)、ほかはテストの stub の gate スクリプト(`gate_run.go:186`)
- go1.26.0(wrapper、`-race`、カバレッジ)と `GOTOOLCHAIN=go1.25.8`(全体を uncached で)の両方で通った

## Coverage

- Statement: `internal/org` 94.1%、`internal/cli` 88.1%(go1.26.0、`-count=1`)。1 回目は org 94.2%、cli 88.1%
- regate のファイルの文の割合(cover.out から数えた): `internal/org/regate.go` 89.4%(185/207)、`internal/org/regate_merge.go` 84.8%(245/289。1 回目は 84.3%、167/198)、`internal/org/regate_git.go` 84.5%(60/71)、`internal/cli/org_regate.go` 98.1%(53/54)。`internal/org/mergeorder.go` は 99.7%(369/370)
- Branch: Go の標準のツールは分岐のカバレッジを出さない。regate のファイルの覆われていないブロック 74 個(1 回目は 61 個。増えた 13 個は D3 の 4 関数のもの)を evidence の 4c〜4e 節に行ごとに書いた
- Function: cycle 2 で足した関数は `regateCheckInTheWay` 87.0%、`regateMergeWrites` 80.6%、`regatePathInTheWay` 84.2%、`regateSpellingOnDisk` 92.9%。変わった `regateMerged` は 94.6%(1 回目 93.9%)。ほかの 100% でない関数は 1 回目と同じ値で、`Regate` 89.6%、`regateFirstCheck` 93.8%、`writeRegateMarker` 72.7%、`regateCheckWorktree` 87.5%、`regateCheckInProgress` 81.8%、`regateGate` 92.0%、`recordRegate` 85.2%、`regateFetch` 87.5%、`regateCheckOriginHead` 76.9%、`regateIsAncestor` 66.7%、`regateBoundedGit` 84.0%、`regateHeadNotOnOrigin` 78.6%、`regateParents` 85.7%、`regateCheckIdentity` 71.4%、`regateMergeTip` 91.2%、`regateAbortLeftMerge` 72.7%、`regateUndoMerge` 69.6%、`regateUndoneText` 66.7%、`regatePush` 76.5%、`regateSetTrackingRef` 80.0%、`newOrgRegateCmd` 94.4%。regate の 46 関数のうちほかの 20 関数は 100%
- mergeorder.go で変わった関数(`collectMergeTargets`、`classifyMergeTarget`、`classifyGatePassed`、`mergeRegateBranches`、`regates`、`namesRegate`、`mergeRegateNext`、`mergeRegateFailedDetail`、`mergeLedgerNotes`、`mergeEntryFor`、`mergeOrderListNotes` ほか)は全部 100%。`MergeOrder` の 96.7% は変わっていない `run = realMergeOrderExec`(`mergeorder.go:250`)だけで、テストはいつも Exec を差し込む
- Notes: D3 の 4 関数の覆われていないブロックは、git が落ちる、上限を超える、読めないときの error の返しと、ループのあとの `return "", nil`(`regate_merge.go:416`)。416 行はループの最後の要素が必ず return するので届かない。振る舞いとして意味のあるものは「Test gaps」に書いた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の P1(merge が ignored のファイルを黙って上書きする) | 直った | HEAD の binary とローカルの bare origin で、origin の main が `git add -f` で `docs/evidence/run.log` を追跡し始め、worktree に ignore された同じ名前のファイルがある場面を打った。HEAD f2f65a30 は終了コード 1、stdout は空、stderr に `docs/evidence/run.log` を名指す断り、記録は増えず、`MERGE_HEAD` もなく、ファイルの中身はそのまま。退かして打ち直すと merge、gate、push が通り、merge-order は `ready`。同じ場面を 1 回目の HEAD(6e67e718 を `git archive` して build)で打つと、終了コード 0 で merge と push をし、ファイルの中身は main のものに書き換わった(evidence の 8 節) |
| cross-review の P2(読めない gate の記録が立て直した org_id の 1 本のブランチにだけ付く) | 直った | `TestMergeOrder_UnreadableRecordAttachesToEveryBranchOfItsOrg` ほか D2 の 3 本と、`TestRegate_GatesWhenNoRecordLetsItStop` の D2 の case |
| 1 回目の Test gap 1(既定のブランチが HEAD の子孫の場合のテストがない) | テストができた | `TestRegate_MergesADefaultBranchTipThatAlreadyContainsTheBranch`(`regate_merge_test.go:132`)が、1 回目の probe で見た結果(親が前の HEAD と tip、木が tip と同じ merge commit、`via=regate` の記録、push、merge-order の `ready`)を固定する |
| `ralph org gate` の既存のテスト(記録の形は regate の記録にだけ `via` が付く) | pass | `-race` の `Gate` に当たるテストと 2 回の全体の実行。1 回目のあと `gate_test.go` は変わっていない |
| merge-order の既存のテスト | pass。3 本は D2 で期待を変えた | main にあった 3 本を、D2 の新しい規則に合わせて名前と期待を変えた。`…AttachesToTheLatestOfTwoEarlierBindings` は `…AttachesToEveryBranchOfItsOrg` に、`…WithNoBindingAttachesToItsOrgsGatedBranch` は `…Branches` に(`feat/e` が `ready` から `unreadable` に)、`…PrefersTheBindingToAGatedBranch` は `…AttachesToTheBoundAndTheGatedBranch` に(`feat/x` が `ready` から `unreadable` に)。計画の D2(cross-review の P2)の意図した変更で、どのブランチも古い成功の記録に戻らない側への変更なので、弱めてはいない。走っている org の理由の欄は `TestMergeOrder_DetailsOfARunningOrgOrNoBindingAreUnchanged` が全文で比べ、通った |
| Spawn と start の既存のテスト | pass | 1 回目のあと `spawn_test.go` と `feature_test.go` は変わっていない。`-race` の `Spawn` と `StartFeature` に当たるテストが全部通った |
| 既知の flaky(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | 落ちなかった | `internal/org` の全体を 3 回走らせて 3 回とも ok |

## Test gaps

1. fetch か push が走っている間に ctx が終わる場合(`regate_git.go:197` と `regate_merge.go:708`)を通すテストがない。push の途中の中断は、pre-push の hook が git を殺す case(`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`)が同じ `regatePushInterrupted(during=true)` の文を通すので、出力は固定されている。端末の Ctrl-C は git にも ralph にも届くので、どちらの枝に入るかは早さで決まる
2. 120 秒の上限は fetch だけがテストで縮めて通す(`TestRegate_FetchFailures` の「the fetch runs past the bound」)。push の上限(`regate_merge.go:714`、`PushFailed` に「origin may not have it」)、merge の上限(`:495`、`:510`)、`git var` の上限(`:224`)、`reset --keep` の上限(`:656`)、cycle 2 で足した `git merge-tree` の上限(`:322`)は通らない。どれも fetch と同じく `regateGitTimeout` を縮めれば打てる
3. `git merge --abort` そのものが失敗する場合(signal、走らない、exit が 0 でない)と、abort のあとに HEAD か作業ツリーが読めない場合(`regateAbortLeftMerge`、`regate_merge.go:559-587`)を通すテストがない。AC5 の「HEAD と作業ツリーが前のままであることを確かめ」の確かめが失敗した側の文は固定されていない
4. `regateUndoMerge` で HEAD か作業ツリーが読めない、`reset --keep` が signal で止まる、走らない、上限を超える場合(`regate_merge.go:631-658`)は通らない。`reset --keep` が exit 1 で落ちる場合は通る
5. D3 の確かめの error の道は、`--write-tree` を知っている merge-tree が exit 128 で落ちる場合だけを通す。merge-base が 0 と 1 のほかで終わる(`regate_merge.go:262`)、merge-tree が走らない・上限を超える・1 行目に木がない(`:320-330`)、代わりの道の `merge-tree -h` と 2 つの diff が落ちる(`:336-350`)、lstat が「ない」のほかの誤りを返す(`:392`)、ディレクトリが読めない(`:398`、`:431`)場合の文は固定されていない。どれも何も merge も記録もせずに error で返す
6. 2.38 より前の git の代わりの道(2 つの diff の積)は、PATH の wrapper が `merge-tree` を `--write-tree` を名指さない usage と exit 129 に差し替えた場合だけで通り、本物の古い git では走らせていない。通る case も「tip が足すパスに ignored のファイル」の 1 つだけで、代わりの道が directory rename の移った先を見落とすこと(`regateMergeWrites` の注に書かれた割り切り)を示すテストはない。CI の ubuntu の git も `--write-tree` を持つので、CI でも代わりの道は wrapper だけで通る
7. 大文字小文字の 6 行(断る 3 行、断らない 3 行)は、大文字小文字を区別しない FS でだけ走る。この機械では走って通った。区別する FS では SKIP になり(Case-sensitive APFS で確かめた)、CI の `ubuntu-latest` でも SKIP になるので、c621ccc9 の直しを CI で見るテストはない。Linux そのものでは走らせていない。git の外で worktree のパスの綴りだけを変えた場合(verify の V2-5、tech-debt に回す)と、Unicode の正規化の形だけが違う名前(`regateSpellingOnDisk` の注)のテストもない
8. 断りが名指すパスの数の境目は、12 個(`regateInTheWayListed` の 10 と 2 個)の case だけが見る。ちょうど 10 個(「and N more」が付かない)と 11 個(「and 1 more」)の case はないので、`more > 0` を `more >= 0` や `more > 1` に変える off-by-one は 12 個の case では落ちない。コードを読んでの判断で、mutant は走らせていない
9. AC11 の skill の文の中身(`/org` と `/director` の regate の手順)を固定するテストはない。4 面の一致は /verify の `check-skill-sync.sh` が見る
10. 端末の本物の Ctrl-C、ssh の origin、本物の GitHub への push は確かめていない。テストは macOS の go1.26.0 と go1.25.8 で、Linux では走らせていない
11. 計画の Evidence に挙がる「mutant の赤」は、この /test では作っていない。P1 については、1 回目の binary を対照にした probe が、HEAD の確かめがないと上書きが起きることを示した

## Verdict

- Pass: pass。`./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、`-race` の絞った実行(1,054 件)、`GOTOOLCHAIN=go1.25.8` の全体の実行、カバレッジの実行がどれも exit 0 で、失敗は 0。足したテスト 70 本が全部通り、AC1〜AC10 と AC13、D2 と D3 の直しにそれぞれテストがある。大文字小文字の 6 行はこの機械では走って通り、区別する FS では SKIP になる
- Fail: なし
- Blocked: なし。AC11 は文書の AC で、自動のテストは CLI の help だけ。上の Test gaps は、どれも PR を止めるものではない
