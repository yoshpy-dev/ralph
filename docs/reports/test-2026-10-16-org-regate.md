# Test report: org-regate

- Date: 2026-10-17(計画の日付は 2026-10-16)
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認の digest は sha256:3c417c48763c)
- Tester: tester subagent (Claude)。パイプライン 3 回目(cycle-count.json は 2 のまま。上限を 3 に上げた回し直しを cycle 3 と呼ぶ)
- Scope: `git diff 670ec219...HEAD`(HEAD 22660dd5)。2 回目の報告(72299c71、HEAD f2f65a30)のあとに入ったのは、cross-review の cycle 2 の直し(556f73fb の `regateIsOwnMerge`、4129ae22 の push の前に読んだ値からだけ動かす追跡の ref と test hook の `afterPush` と `beforeTrackingUpdate`)、ファイルの移動(0e08999f の `internal/org/regate_inway.go`、14252f06 の `internal/org/gate_record.go`)、件名の確かめ(0844ad2b の `regateMergeSubject` と `--no-show-signature`)、注と help の直し(82cf0d39、ba84bccc、62f8e31b、f850de13)、テスト(f630e216、556f73fb と 4129ae22 と 0844ad2b のテスト、3989d914、3a452a71)。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。コードとテストは変えていない
- Evidence: `docs/evidence/test-2026-10-16-org-regate.log` の末尾の「pipeline cycle 3」の節(C3-1〜C3-9。7 つのコマンドの生の出力、regate のファイルと mergeorder.go の関数ごとと文ごとのカバレッジ、覆われていないブロックとその行、足したテストの突き合わせ、AC12 の grep、大文字小文字を区別する FS での実行、3 回の繰り返し、テストのあとの worktree。`~` は `$HOME`、`$SCRATCH` は session の scratchpad、`$TMPDIR` は macOS の一時ディレクトリ)。wrapper 自身のログは `docs/evidence/verify-2026-10-17-000646.log`。どちらも gitignore の対象

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は changed、言語は golang) | shell 41 本 + Go 8 パッケージ | 全部 | 0 | shell の 1 件(下の注) | 813 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,283 | 41 本とも OK | 0(`FAIL` の字は PASS の行の中にだけある) | 1 件 | 上に含む |
| └ `go test ./...`(go1.26.0) | 8 パッケージ | 8 | 0 | - | cli 241.8 s、org 324.8 s、config 0.7 s。insights・driver・protocol・scaffold・upgrade は `(cached)` |
| `go test -race ./internal/org/ ./internal/cli/ -count=1 -v`(go1.26.0、絞らずに全体) | 1,662 本(org 919、cli 743)、subtest を入れて 4,388 | 4,387 | 0 | 1(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、この変更とは関係がない) | 410 s(org 401.3 s、cli 296.5 s)。`DATA RACE` はなし。wrapper の shell の部分と Go の部分の始めに重なった |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -v` | 1,662 本、4,388 | 4,387 | 0 | 1(同じ) | 370 s(org 366.4 s、cli 303.3 s)。結果の行(時間を除いて並べたもの)は `-race` の実行と 4,388 行とも一致 |
| `go test -count=1 -coverprofile ./internal/org/ ./internal/cli/`(go1.26.0) | 2 パッケージ | 2 | 0 | - | 398 s(org 393.4 s、cli 309.1 s)。go1.25.8 の実行と重なった |
| `TMPDIR` を大文字小文字を区別する APFS のイメージに置き、`go test ./internal/org/ ./internal/cli/ -count=1 -v` | 1,662 本、4,388 | 4,381 | 0 | 7(大文字小文字の 6 行と、上の Darwin の 1 件) | 343 s(org 341.7 s、cli 252.2 s)。単独で走らせた |
| cycle 3 の D4/AC7 のテスト 6 本を `-count=3 -v` | 18 本、subtest を入れて 57 | 57 | 0 | 0 | 119 s。単独で走らせた |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の case を飛ばし、偽の古い git の case が代わりに見る。この変更とは関係がない
- 足したテストは 75 本(`internal/org/regate_merge_test.go` 29、`regate_test.go` 21、`internal/cli/org_regate_test.go` 11、`spawn_test.go` 3、`feature_test.go` 1、`mergeorder_test.go` 8、`gate_test.go` 2)。2 回目の 70 本に cycle 3 の 5 本が加わった。ほかに、元からあるテストの subtest が 3 行増えた(`TestRegate_AFailedPushIsNotRetriedPastWhatChanged` の 3 行目、`TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay` の 10 個と 11 個の行)。`TestOrgRegate_Help` は help の文を 1 つ足し、1 つ書き換えた。75 本の名前と `-race` の実行の `--- PASS` の行を `comm` で突き合わせ、欠けは 0(evidence の C3-5)
- 署名のテスト `TestRegate_RetriesThePushOfItsSignedMergeCommitWithShowSignatureOn` は、`/usr/bin/ssh-keygen`(OpenSSH_10.2p1)があるので、この機械では SKIP せずに走った。`-race`、go1.25.8、大文字小文字を区別する FS、3 回の繰り返しのどれでも PASS(`-v` でない wrapper とカバレッジの実行では、パッケージが ok だったことしか分からない)
- 大文字小文字の行: `regateSkipOnACaseSensitiveFS` が決める 6 行は、大文字小文字を区別しないこの機械の `$TMPDIR` では wrapper、`-race`、go1.25.8、カバレッジ、繰り返しのどれでも走って通った。区別する FS では 6 行とも SKIP になり、ほかはすべて PASS した。今回は regate のテストだけでなく、`internal/org` と `internal/cli` の全体をその FS で走らせた(2 回目は regate のテスト 6 本だけ)。イメージは `hdiutil` で scratchpad の下に作り、終わったあと detach して消した(evidence の C3-7)
- 既知の flaky の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org/director_cmd_test.go:299`)は、`internal/org` を丸ごと走らせた 5 回(wrapper、`-race`、go1.25.8、カバレッジ、大文字小文字を区別する FS)のどれでも落ちなかった。打ち直しはしていない
- テストのあとの worktree: `git status --porcelain` は空。ignore された中で変わったのは、wrapper の evidence の世代管理が古い `verify-2026-10-16-185915.log` を消して `verify-2026-10-17-000646.log` を書いたことだけ(evidence の C3-9)。main のチェックアウトの `GIT_OPTIONAL_LOCKS=0 git status --porcelain` も空

## AC とテストの対応

| AC | テスト(file:line) |
| --- | --- |
| AC1 断る | `TestRegate_RefusedBeforeTheFetch`(`internal/org/regate_test.go:391`)、`TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns`(`:585`)、`TestRegate_RefusedWhileAnotherRegateHoldsTheLock`(`:618`)、`TestOrgRegate_RefusalPrintsNothingAndChangesNothing`(`internal/cli/org_regate_test.go:326`)、`TestOrgRegate_OrgIDRefusedBranchRequiredNoArguments`(`:372`)、`TestOrgRegate_LegacyLedgerGuard`(`:669`) |
| AC1 の D3(merge が書くパスに ignored のもの) | `TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay`(`internal/org/regate_merge_test.go:1589`、14 の case。f630e216 で、断りが並べる数ちょうどの 10 個と 1 つ多い 11 個の行が加わった)、`TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay`(`:1825`)、`TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch`(`:1797`)、`TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse`(`:1905`)、`TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree`(`:1944`)。確かめる関数は 0e08999f で `internal/org/regate_inway.go` に移り、中身は変わっていない |
| AC2 fetch と HEAD | `TestRegate_FetchFailures`(`regate_test.go:712`)、`TestRegate_AFetchEndedByASignalRecordsNothing`(`:781`)、`TestRegate_RefusesAHeadThatIsNotOriginsTip`(`:807`)、`TestRegateRemoteEnv`(`:863`)、`TestRegate_FetchRunsWithNoTerminalPrompt`(`:914`) |
| AC3 何もしない道と壊れた記録 | `TestRegate_UpToDateDoesNothing`(`regate_test.go:946`)、`TestRegate_GatesWhenNoRecordLetsItStop`(`:995`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`) |
| AC4 取り込みの道 | `TestRegate_MergesTheTipInAndPushes`(`regate_merge_test.go:31`)、`TestRegate_PointsTheRemoteTrackingRefAtThePush`(`:111`)、`TestRegate_MergesADefaultBranchTipThatAlreadyContainsTheBranch`(`:201`)、`TestOrgRegate_MergesPushesAndThenIsUpToDate`(`org_regate_test.go:120`) |
| AC5 merge commit を作れない | `TestRegate_RefusedWhenTheMergeCommitCannotBeMade`(`regate_merge_test.go:279`)、`TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack`(`:381`)、`TestRegate_AMergeEndedAfterItsCommitNamesResetKeep`(`:429`)、`TestRegate_AMergeCommitItCannotReadNamesResetKeep`(`:480`)、`TestRegate_AnotherCommitAfterTheMergeNamesResetKeep`(`:560`) |
| AC6 スクリプトの失敗と中断 | `TestRegate_AScriptFailureAfterTheMergeUndoesIt`(`regate_merge_test.go:619`)、`TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork`(`:674`)、`TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge`(`:770`)、`TestOrgRegate_FailureAfterAMergeUndoesItAndExits1`(`org_regate_test.go:222`)、`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing`(`:506`)、`TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits`(`:594`)、`TestRegate_Interrupted`(`regate_test.go:1373`) |
| AC7 push の失敗と打ち直し(前からの行) | `TestRegate_APushThatFailsKeepsTheRecordAndTheMerge`(`regate_merge_test.go:969`)、`TestRegate_APushThatLandedWhoseAnswerWasLost`(`:1024`)、`TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge`(`:1134`)、`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`(`:1172`)、`TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt`(`:1211`)、`TestRegate_AnUnreadableLatestRecordGivesNoPushOnlyAndNoReset`(`:1459`)、`TestOrgRegate_AFailedPushIsPushedOnlyNextTime`(`org_regate_test.go:266`) |
| AC7 と D4(cycle 3 で書き直した部分) | 下の「AC7 と D4 の cycle 3 の場合」の表 |
| AC8 結びつきが変わった | `TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan`(`regate_test.go:1176`)、`TestRegate_ARefusedRecordAfterTheMergeUndoesIt`(`regate_merge_test.go:922`) |
| AC9 gate だけの道 | `TestRegate_GatesWhenNoRecordLetsItStop`(`regate_test.go:995`)、`TestRegate_GateOnlyRecordsAGatePassedViaRegate`(`:1066`)、`TestRegate_RecordsWithTheOrgOfTheLatestBinding`(`:1109`)、`TestRegate_GateOnlyFailuresRecordGateFailed`(`:1125`)、`TestOrgRegate_GateOnlyPassPrintsNoMergeAndNoPush`(`org_regate_test.go:188`) |
| AC10 merge-order の理由の欄 | `internal/org/mergeorder_test.go` の `TestMergeOrder_DetailsNameRegateForABranchWhoseOrgIsNotRunning`(`:558`)、`…DetailsOfARunningOrgOrNoBindingAreUnchanged`(`:630`)、`…DetailsOfABranchWhoseOrgRunsAnotherFeature`(`:713`)、`…SeveralReadyNoteNamesRegateForADisbandedOrg`(`:800`)、`…GateViaIsInTheEntryAndItsJSON`(`:829`)、`…LatestGateRecordWins`(`:861`) |
| D2 読めない記録を org のすべてのブランチに付ける | `TestMergeOrder_UnreadableRecordAttachesToEveryBranchOfItsOrg`(`mergeorder_test.go:982`)、`…UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranches`(`:1036`)、`…UnreadableRecordAttachesToTheBoundAndTheGatedBranch`(`:1073`)、`…UnreadableRecordAttachesToTheBindingBeforeIt`(`:943`) |
| AC13 Spawn との排他 | `TestOrgSpawn_RefusedWhileRegateRunsOnTheOrg`(`internal/org/spawn_test.go:4584`)、`TestOrgSpawn_RegateLockStates`(`:4616`)、`TestOrgSpawn_RegateLockThatCannotBeCheckedRefuses`(`:4663`)、`TestStartFeature_RefusedWhileRegateRunsOnTheOrg`(`internal/org/feature_test.go:1246`)、`TestRegate_MarksTheLockWithTheOrgItWorksOn`(`regate_test.go:1222`)、`TestRegate_EmptiesAnEarlierMarkerBeforeTheManifestLock`(`:1267`)、`TestRegate_ARefusedFirstCheckLeavesNoMarker`(`:1310`)、`TestReadRegateMarker`(`:1344`) |
| D6 の `via=regate` と共有の helper | `TestGateRecordDetails_Via`(`internal/org/gate_test.go:1463`。記録の型は 14252f06 で `internal/org/gate_record.go` に移り、中身は変わっていない)、`TestGateGitHelpers_ErrorsStartWithTheCallersPrefix`(`:773`)、`TestRegate_ErrorsOfTheSharedHelpersSayRegate`(`regate_test.go:1470`)、`TestGateWording_RefusalsThatAskForACommit`(`:654`) |
| AC11 文書 | 自動のテストは CLI の help だけ(`TestOrgRegate_Help`、`org_regate_test.go:723`。3a452a71 で、hook が件名を変えたときの断りの文と、spawn と start の断りの書き直した文を押さえる)。skill の文の中身を固定するテストはない |
| AC12 テストの隔離と 2 つの版 | テストの性質なので、下の「AC12 の確かめ」で見た |

### AC7 と D4 の cycle 3 の場合

どれも `-race`(go1.26.0)、go1.25.8、大文字小文字を区別する FS、`-count=3` で PASS した(evidence の C3-2、C3-3、C3-7、C3-8)。

| 場合 | テスト(file:line) | 見ていること |
| --- | --- | --- |
| gate だけの道で記録した head は、origin がそこを離れたあと push し直さない | `TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`(`regate_merge_test.go:1253`、3 行。ブランチ自身の commit の pass と failure、別のブランチの merge commit の pass) | origin の `feat/auth` を head の第 1 親に戻したあとの regate が断り、何も push せず、何も記録せず、スクリプトを走らせず、`reset --keep` の手を出さない |
| 落ちた push は、変わったものを越えて打ち直さない | `TestRegate_AFailedPushIsNotRetriedPastWhatChanged`(`:1051`、3 行。fetch と push の間に誰かが push した、origin の main が進んだ、origin の `feat/auth` が merge commit の第 1 親より下に戻された。3 行目が 556f73fb で加わった) | 1 行目と 3 行目は断るだけで手を出さない。2 行目は `reset --keep` の手を全文で比べ、手に従って reset したあとの regate が新しく merge し、push し、merge-order が `ready` になる |
| push のあと何かが動かした追跡の ref は残す | `TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush`(`:135`、3 行) | push の直後(`afterPush`)の fetch では update-ref まで進まない(`beforeTrackingUpdate` が呼ばれたら Errorf)。`remote.origin.fetch` のない repo で update-ref の直前(`beforeTrackingUpdate`)に fetch が動かすと、CAS が落ち、読み直して動いていたので誤りにしない。どちらも ref は fetch が持ってきた値のまま。ref が push の前の値のまま lock ファイルで update-ref が落ちると誤りになり、その文の頭と尾を比べる |
| 人の merge commit を regate 自身のものと取り違えない | `TestRegate_APersonsMergeOfTheDefaultBranchIsNotTakenForRegates`(`:1297`、4 行。`git merge origin/main` と GitHub の Update branch の件名、それぞれ pass と failure) | 親が regate の merge と同じであることと件名を前提として確かめたうえで、origin を第 1 親に戻したあとの regate が、origin の main がそのままのときも進んだあとも断り、push も記録も手も出さない |
| hook が件名を変えた regate の merge commit は断る | `TestRegate_AMergeCommitWhoseSubjectAHookChangedIsNotPushedOnly`(`:1364`) | prepare-commit-msg の hook が件名の頭に `[checked] ` を足した merge commit の push が落ちたあと、打ち直しが push だけの道に進まず断り、`reset --keep` の手を出さない |
| 署名した merge commit を `log.showSignature=true` のもとで打ち直す | `TestRegate_RetriesThePushOfItsSignedMergeCommitWithShowSignatureOn`(`:1398`) | テストの中で作った ssh の鍵で署名し、`git log --format=%s` が署名の確かめの行のあとに件名を出すことを前提として確かめたうえで、打ち直しが `PUSH_ONLY` で、スクリプトを走らせず、記録を増やさず、origin が署名した merge commit になる。この機械では SKIP せずに走った |

### AC12 の確かめ

- 2 回目のあとに変わったテストファイルは `internal/org/regate_merge_test.go` と `internal/cli/org_regate_test.go` の 2 つ。足した行に、URL、`ssh://`、`git@`、`net.Dial`、`http.`、`osascript`、`tmux`、`herdr`、端末のアプリの名前、signal を送る行(`Kill`、`Signal`、`SIGTERM`、`kill`)はない(evidence の C3-6)
- 足した行のうちプロセスを動かす 30 行のうち 28 行は、fixture の `fx.origin`、`fx.worktree`、`fx.main`(どれも `t.TempDir()` の下)で git を打つ。残りの 2 行は署名のテストの `exec.LookPath("ssh-keygen")` と、`t.TempDir()` の下に鍵を作る `ssh-keygen -q -t ed25519 -N ""`。そのテストは `SSH_AUTH_SOCK` を `t.Setenv` で戻せる形にしてから外すので、ssh-agent には問い合わせない。署名の設定は fixture の main のチェックアウトの config に置き、`isolateGitEnv` が global と system の git の設定を外す
- 新しい test hook(`regateTestHooks.afterPush`、`beforeTrackingUpdate`)の中で動くのは、fixture の bare origin への push と、fixture の worktree での fetch だけ
- go1.26.0(wrapper、`-race`、カバレッジ、大文字小文字を区別する FS)と `GOTOOLCHAIN=go1.25.8`(org と cli を uncached で)の両方で通った

## Coverage

- Statement: `internal/org` 94.1%、`internal/cli` 88.1%(go1.26.0、`-count=1`)。2 回目と同じ
- regate のファイルの文の割合(cover.out から数えた): `internal/org/regate.go` 89.4%(185/207、2 回目と同じ)、`internal/org/regate_merge.go` 85.0%(193/227)、`internal/org/regate_inway.go` 85.1%(74/87)、`internal/org/regate_git.go` 84.5%(60/71、2 回目と同じ)、`internal/org/gate_record.go` 100.0%(32/32)、`internal/cli/org_regate.go` 98.1%(53/54)。2 回目の `regate_merge.go` は 84.8%(245/289)で、D3 の関数が `regate_inway.go` に移ったので、2 つを足した 85.0%(267/314)と比べる。`internal/org/mergeorder.go` は 99.7%(369/370)
- Branch: Go の標準のツールは分岐のカバレッジを出さない。regate のファイルの覆われていないブロックは 77 個(2 回目は 74 個)、mergeorder.go は 1 個で、evidence の C3-4b〜C3-4e 節に行ごとに書いた
- Function(D4 と AC7 の関数): `regateHeadNotOnOrigin` 83.3%、`regateResetHint` 100%、`regateIsOwnMerge` 85.7%、`regateMergeSubject` 100%、`regateParents` 85.7%、`passedOn` 100%、`regatePush` 73.9%(2 回目は 76.5%)、`regatePushInterrupted` 100%、`regateSetTrackingRef` 92.3%(2 回目は 80.0%)、`regateTrackingRef` 75.0%(新しい関数)、`regateIsAncestor` 66.7%、`regateCheckOriginHead` 76.9%。ほかの 100% でない関数は 2 回目と同じ値(`Regate` 89.6%、`regateFirstCheck` 93.8%、`writeRegateMarker` 72.7%、`regateCheckWorktree` 87.5%、`regateCheckInProgress` 81.8%、`regateGate` 92.0%、`recordRegate` 85.2%、`regateFetch` 87.5%、`regateBoundedGit` 84.0%、`regateCheckIdentity` 71.4%、`regateMergeTip` 91.2%、`regateAbortLeftMerge` 72.7%、`regateUndoMerge` 69.6%、`regateUndoneText` 66.7%、`regateMerged` 94.6%、`newOrgRegateCmd` 94.4%、D3 の 4 関数 87.0%・80.6%・84.2%・92.9%)。`gate_record.go` の 4 関数と mergeorder.go の関数は `MergeOrder`(96.7%)のほかは全部 100%
- D4 と AC7 で覆われていないブロック:
  - push だけの打ち直し: `regateIsOwnMerge` か `regateIsAncestor` が誤りを返したとき(`internal/org/regate_merge.go:84-86`)と、その先の断りの道で `regateCheckOriginHead` と `regateResetHint` が誤りを返したとき(`:93-95`、`:97-99`)
  - 自身の merge commit の確かめ: 件名の比べは両側とも通る(合う側は署名のテストと `TestOrgRegate_AFailedPushIsPushedOnlyNextTime`、合わない側は hook と人の merge のテスト)。件名を読む `git log -1 --no-show-signature` が誤りを返す枝(`:154-156`)は通らない。親を読む誤りは、親が合わない場合と同じブロック(`:150-152`)にあるので、行のカバレッジでは区別できない
  - 追跡の ref: push の前の読みの失敗(`:557-558` は ctx が終わった、`:559-561` は git が走らない)、push のあとの読みの失敗(`:615-617`)、`regateTrackingRef` の誤り(`:639-641`)は通らない。CAS が落ちたあとの読み直しは、動いていた側(`TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush` の 2 行目、誤りにしない)と動いていない側(3 行目、誤り)の両方が通る。読み直し自身の失敗は誤りの側と同じブロックに入る
  - push そのもの: ctx が push の途中で終わる(`:566-567`)、git が走らない(`:570-571`)、上限を超える(`:572-575`)。2 回目からの穴
- Notes: `regateTrackingRef` と `gateGit` は、ctx が終わったとき、git が起動できないとき、signal で終わったときだけ誤りを返す(0 でない exit は誤りにせず、`rev-parse --verify --quiet` の exit 1 は「ref がない」になる)。上の読みの失敗の枝は、D3 のテストが使う PATH の wrapper(`regateWrapGit`)と同じやり方で、特定の引数の git だけを signal で終わらせれば通せる。`regateCheckOriginHead` の `t.head == originHead`(`internal/org/regate_git.go:83-85`)は、呼ぶ側が先に同じ比べをするので届かない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の cycle 2 の 1 件目(計画の P2-1。gate だけの regate も `via=regate` を書くので、1 つ親の commit に `reset --keep` の手が出て、origin を戻したあとに push し直す) | 直った | `TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt` の 3 行と、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged` の 3 行目。計画によれば、どれも直す前に赤を取ったと implementer が確かめた(この /test では直す前の版を走らせていない) |
| cross-review の cycle 2 の 2 件目(計画の P2-2。push のあとに読んだ値から追跡の ref を動かすので、間の fetch を巻き戻す) | 直った | `TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush` の 3 行。`afterPush` の行は update-ref まで進まないことを、`beforeTrackingUpdate` の行は CAS が落ちたあとの読み直しを見る |
| verify の cycle 3 の V3-1(衝突の断りに従って人が作る merge commit を、親だけで regate 自身のものと取り違える) | 直った | `TestRegate_APersonsMergeOfTheDefaultBranchIsNotTakenForRegates` の 4 行。hook が件名を変えた regate の merge commit は断る側に倒れる(`TestRegate_AMergeCommitWhoseSubjectAHookChangedIsNotPushedOnly`) |
| verify の cycle 3 の V3-7(`--no-show-signature` を押さえるテストがない) | テストができ、この機械で走って通った | `TestRegate_RetriesThePushOfItsSignedMergeCommitWithShowSignatureOn`(3989d914)。flag を外すと落ちることは計画と verify の F2 の対照に書かれている。この /test では変異を走らせていない |
| verify の cycle 3 の V3-6(help が hook の件名の場合を書かない) | 直り、テストが文を押さえる | `TestOrgRegate_Help`(3a452a71)が PASS |
| 2 回目の Test gap 8(断りが並べるパスの数の境目) | テストができた | f630e216 の「as many paths in the way as the refusal lists」と「one path in the way more than the refusal lists」(`regate_merge_test.go:1651`、`:1655`)が PASS |
| 前からの push だけの打ち直し(leader の記録、regate の記録) | pass | `TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt`、`TestOrgRegate_AFailedPushIsPushedOnlyNextTime`、`TestRegate_APushThatFailsKeepsTheRecordAndTheMerge` が `regateIsOwnMerge` を通る形で PASS |
| ファイルの移動(0e08999f、14252f06)で振る舞いが変わらない | pass | D3 のテストと gate の記録のテストが全部通った。`ralph org gate` の既存のテストは `-race` と go1.25.8 の全体の実行で通った |
| merge-order、Spawn、start の既存のテスト | pass | 2 回目のあと `mergeorder_test.go`、`spawn_test.go`、`feature_test.go` は変わっていない。`-race` と go1.25.8 の全体の実行で通った |
| 既知の flaky(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | 落ちなかった | `internal/org` の全体を 5 回走らせて 5 回とも ok |

## Test gaps

1. D4 と AC7 の誤りの枝: 件名を読む git の誤り(`internal/org/regate_merge.go:154-156`)、それを受ける push だけの道と断りの道(`:84-86`、`:93-95`、`:97-99`)、追跡の ref の push の前と後の読みの失敗(`:557-561`、`:615-617`、`:639-641`)を通すテストはない。どれも ctx が終わるか git が起動できないか signal で終わるときだけの枝で、出る文は固定されていない。D3 のテストの PATH の wrapper のやり方で、引数を見て git を signal で終わらせれば通せる
2. C3-5(push だけの道は、push が届いたあとに origin が merge の第 1 親に戻された場合と、push の失敗を見分けない)は tech-debt に回した割り切りで、どちらの振る舞いもテストは押さえていない。leader の記録(`via` なし)で origin を戻したあとに push し直す件(tech-debt)も、`TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt` の 2 行は origin を戻さないので押さえていない
3. 件名の確かめは、git merge と GitHub の Update branch の件名を手元の git で作って確かめた。本物の GitHub の Update branch の件名は確かめていない(verify と同じ)。署名は ssh の鍵ファイルの場合だけで、gpg、x509、ssh-agent の鍵は試していない。署名のテストが SKIP するのは `ssh-keygen` がないときだけで、ssh の署名を持たない 2.34 より前の git では SKIP せずに落ちる。CI の `ubuntu-latest` には `ssh-keygen` と 2.34 より新しい git があるので、おそらく走る。未確認です
4. fetch か push が走っている間に ctx が終わる場合(`internal/org/regate_git.go:197-198`、`regate_merge.go:566-567`)を通すテストはない。2 回目からの穴。push の途中の中断の文は、pre-push の hook が git を殺す case(`TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge`)が同じ `regatePushInterrupted(during=true)` を通すので固定されている
5. 120 秒の上限は fetch だけがテストで縮めて通す。push(`regate_merge.go:572-575`)、`git var`(`:280-282`)、merge(`:343-344`、`:358-361`)、`reset --keep`(`:504-506`)、`git merge-tree`(`internal/org/regate_inway.go:106-108`)の上限は通らない。2 回目からの穴
6. `git merge --abort` そのものの失敗と、abort のあとに HEAD か作業ツリーが読めない場合(`regate_merge.go:408-435`)、`regateUndoMerge` で HEAD か作業ツリーが読めない、`reset --keep` が signal で止まる、走らない場合(`:479-503`)は通らない。2 回目からの穴
7. D3 の確かめの誤りの道(`regate_inway.go:42-59`、`:104-134`、`:176-184`、`:215-217`)と、2.38 より前の git の代わりの道を本物の古い git で走らせること。2 回目からの穴
8. 大文字小文字の 6 行は、大文字小文字を区別しない FS でだけ走る。この機械では走って通り、区別する FS では SKIP になった。CI の `ubuntu-latest` でも SKIP になるので、c621ccc9 の直しを CI で見るテストはない。Linux そのものでは走らせていない
9. AC11 の skill の文の中身(`/org` と `/director` の regate の手順)を固定するテストはない
10. 端末の本物の Ctrl-C、ssh の origin、本物の GitHub への push は確かめていない。テストは macOS の go1.26.0 と go1.25.8 で、Linux では走らせていない
11. 計画の Evidence に挙がる「mutant の赤」は、この /test では作っていない。cycle 3 の直しの赤は、計画の Progress に implementer が直す前に取ったと書いたもの(556f73fb、4129ae22、0844ad2b)と、verify の F2 の変異の対照による
12. この PR の外(このブランチでは変わっていない `internal/org/watch_test.go`): `newTestWatchRun`(`internal/org/watch_test.go:332`)は `os.MkdirTemp("", "watch-status-*")` を作って消さないので、`internal/org` を走らせるたびに `$TMPDIR` に約 29 個のディレクトリが残る。今回の 4 回の実行で 116 個増え、`$TMPDIR` には 39,931 個あった。テストの判定には響かない。`t.TempDir()` に変える 1 行を tech-debt に足すことを勧める。消すのはこの task の worktree の外なので、残してある

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、`-race` の org と cli の全体(4,388 件)、`GOTOOLCHAIN=go1.25.8` の org と cli の全体(4,388 件、`-race` の実行と結果の行が一致)、カバレッジの実行、大文字小文字を区別する FS での org と cli の全体、cycle 3 のテストの 3 回の繰り返しが、どれも exit 0 で失敗は 0。足したテスト 75 本が全部通り、AC7 と D4 の書き直した 6 つの場合にそれぞれテストがあり、署名のテストはこの機械で走って通った
- Fail: なし
- Blocked: なし。AC11 は文書の AC で、自動のテストは CLI の help だけ。上の Test gaps は、どれも PR を止めるものではない
