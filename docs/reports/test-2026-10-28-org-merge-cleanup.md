# Test report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(`## Test plan` と AC1〜AC6、AC3b)。issue #263
- Tester: tester サブエージェント(`/test`、cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD 44f13b7f)。Go は `internal/org` と `internal/cli`、ほかに skill・文書・template。`./scripts/run-test.sh` は changed scope のまま打ち、選ばれた pack は golang で、`go test ./...` が 9 パッケージを `(cached)` なしで打った。テストの関数はこの diff で 47 本増え、消えたものはない
- Evidence: `docs/evidence/test-2026-10-28-org-merge-cleanup.log`(gitignore。`run-test.sh` の全出力、変異の表と各回の結果、coverage の数字。wrapper 自身の写しは `docs/evidence/verify-2026-10-28-113631.log`)

この下の「Test execution」から「Test gaps」までは cycle 1(HEAD 44f13b7f)の記録で、行番号もその時点のもの。パイプラインの cycle 2(cross-review cycle 1 の指摘の直し、HEAD 9a6ae47c)の test は、「Test gaps」と「Verdict」の間の「cycle 2(HEAD 9a6ae47c)」の節にある。いまのブランチの verdict は「Verdict」の節の先頭の cycle 2 の行で、cycle 1 の verdict はその下の「cycle 1(HEAD 44f13b7f)」に残している。cycle 2 の変異の ID は `C2-M<n>`。

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 9 パッケージ | shell 45 本 OK(PASS の行 4,782)、Go 9 パッケージ ok | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.49.0 で、古い git は stub の行が見る) | 2,608 秒(20:36:30〜21:19:58 JST)。開始時の load 62.9、終了時 72.8、途中で 112 まで上がった |
| └ Go: `internal/org` | 1,252 本(トップレベル) | ok | 0 | 0 | 933.7 秒 |
| └ Go: `internal/cli` | 845 本(トップレベル) | ok | 0 | 0 | 672.6 秒 |
| └ Go: `cmd/ralph`、`internal/{config,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 1.1〜5.8 秒 |
| 基準の絞った回: `go test -count=1 -v -run 'Cleanup' ./internal/org/` | 38(サブテスト込み 100) | 38 | 0 | 0 | 104 秒 |
| 基準の絞った回: `go test -count=1 -v -run 'OrgCleanup\|CleanupSteps\|CleanupPrinter\|OrgMerge\|PrintMergeResult' ./internal/cli/` | 48(サブテスト込み 132) | 48 | 0 | 0 | 219 秒 |
| 基準の絞った回: `go test -count=1 -v -run '^TestMerge' ./internal/org/` | — | ok | 0 | 0 | 16 秒 |
| 変異 26 個(`go test -overlay`、下の Regression checks) | 26 | 25 個が落ちた、1 個は等価 | — | — | 1 個 19〜546 秒 |
| coverage の回(org と cli の 2 回、下の Coverage) | — | ok | 0 | 0 | 170 秒、177 秒 |

負荷は、同じマシンの別の session の負荷生成(`yes` 12 本)と 2 つの worktree の `run-verify.sh` だった。`internal/org` の 933.7 秒は前の狭い回(約 670 秒)より長いが、30 分の上限の内で、時間切れや負荷による失敗はなかった。既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も、この回は通った。

`run-test.sh` が `internal/org` と `internal/cli` を `(cached)` なしで打ったので、lead が挙げた `go test ./internal/org/ ./internal/cli/ -count=1` の打ち直しは要らなかった。`check-skill-sync.sh` と `check-sync.sh` は同期の検査なので `/verify` の範囲で、verify の report で通っている。この回では打っていない。

## Coverage

- Statement: 変更の中心のファイルを、絞った回の profile で測った。`internal/org/cleanup.go` 243/277(87.7%)、`internal/org/merge_run.go` 118/121(97.5%)、`internal/cli/org_cleanup.go` 92/99(92.9%)、`internal/cli/org_merge.go` 66/69(95.7%)
- Branch: 計測なし(Go の道具は statement だけ)
- Function: `cleanup.go` の 27 関数のうち 100% が 12、最低は `checkedOut` 71.4%。`org_cleanup.go` は `runCleanupSteps` 60.0% のほかは 80% 以上
- Notes: パッケージ全体の数字は、絞った回なので意味がない(org 43.5%、cli 10.0%)。打たれていないブロックのうち振る舞いに関わるものは Test gaps の 2〜5

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

`run-test.sh` と基準の回、coverage の回で失敗はなかった。

## Test plan とテストの対応

Unit tests:

| 計画の項目 | テスト |
| --- | --- |
| binding がない | `RefusedAtTheLedgerAndTheRecord/a branch no binding names`(`internal/org/cleanup_test.go:927`)、`a branch with gate records only`(`:929`) |
| org が走っている | `RefusedAtTheLedgerAndTheRecord/the org runs`(`:945`)、`the org runs another feature`(`:948`)、CLI の `TestOrgCleanup_RefusalsAreSkippedLines/the org runs`(`internal/cli/org_cleanup_test.go:218`) |
| 記録がない | `TestCleanupFeatureWorktree_RecordGoneRemovesNothing`(`internal/org/cleanup_test.go:721`、4 つの場面) |
| 記録の branch・path・kind が違う | `RefusedAtTheLedgerAndTheRecord/a record on another branch`(`:961`)、`a record at another path`(`:972`)、`a record of another kind`(`:980`)、`TestCleanupFeatureWorktree_RecordOfAnotherPathOrKindNamesTheChecks`(`:1030`) |
| worktree の中から打った | `TestCleanupFeatureWorktree_RefusedInTheWorktree/run in the worktree`(`:405`)、`run below the worktree`(`:409`) |
| 未コミット、未追跡、rebase の途中 | 同じテストの `an uncommitted change`(`:414`)、`untracked files`(`:419`)、`a rebase in progress`(`:426`)。計画にない `a merge in progress`(`:435`)もある |
| ブランチの先頭が違う | `the branch moved past the merged head`(`:444`)、`TestCleanupFeatureWorktree_MovedBranchKeptUnderItsTip`(`:573`)、`TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch`(`:513`) |
| detached HEAD でコミットを足した、別のブランチにいる | `a detached HEAD with a commit`(`:456`)、`another branch`(`:462`) |
| どれも何も消さず理由と次の手を返す | 上のすべてが `assertRefused`(`:140`)で、前後の snapshot が同じことと理由・Manual の全文を見る。`TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten`(`:1153`)は理由の中のコマンドを sh でそのまま打って次の cleanup が通ることを見る |
| 再開: worktree だけ消えた、記録だけ残った | `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`:635`。`:636`、`:643`、`:658` とブランチが動いた 2 つ) |
| 記録がないときは何も消さない | `RecordGoneRemovesNothing`(`:721`) |
| script の `cleanup --id org-<id> --force-branch` を 1 回だけ、`worktree_cleaned` を 1 行 | `assertCleaned`(`:186`)が回数と台帳の最後の行を見る。`TestCleanupFeatureWorktree_SquashMergedBranch`(`:286`)は script の呼び出しが lookup と cleanup の 2 つだけであることも見る |

Integration tests:

| 計画の項目 | テスト |
| --- | --- |
| 実 script と bare origin で squash merge とリモートのブランチの削除のあとに片付く | `TestCleanupFeatureWorktree_SquashMergedBranch`(`internal/org/cleanup_test.go:286`。fixture が `git branch -d` で止まる状態であることを先に確かめる)、`TestOrgCleanup_CleansUpABranchMergedOnGitHub`(`internal/cli/org_cleanup_test.go:109`) |
| CLI の `ralph org merge` で `CLEANUP: done` と exit 0、未追跡のファイルで `CLEANUP: skipped` と exit 0、何も消えない | `TestOrgMerge_CleansUpTheFeatureWorktree/cleaned up`(`internal/cli/org_merge_test.go:615`)、`/an untracked file`(`:651`) |
| `ralph org cleanup` の片付く道、gh がない道、`--branch` が merged でない道 | `TestOrgCleanup_CleansUpABranchMergedOnGitHub`(`internal/cli/org_cleanup_test.go:109`)、`TestOrgCleanup_WithoutGHRemovesNothing`(`:167`)、`TestOrgCleanup_BranchWhosePRIsNotMergedIsRefused`(`:193`)、`TestCleanupSteps_PicksTheMergedBranches`(`:491`) |

Regression tests:

| 計画の項目 | テスト |
| --- | --- |
| org の merge のテストすべて(opt-in なしでは今までのコマンドだけ) | `TestMerge_RunsTheCleanupOnlyAfterARecordedMerge/no repo root`(`internal/org/merge_run_test.go:926`)と、前からある `^TestMerge` のすべて。変異 M17(opt-in を外す)で、base からある merge のテストが 9 本落ちた |
| `mrFixture` の `-d`・`--delete-branch` の検査 | `mrFixture.merge`(`internal/org/merge_fake_test.go:460`)が merge のテストの呼び出しごとに見る。片付けの道の `TestMerge_CleansUpTheFeatureWorktreeOnceRecorded`(`merge_run_test.go:813`)も同じ fixture を通る |
| regate と spawn の `regate.lock` のテスト | `TestRegate_RefusedWhileAnotherRegateHoldsTheLock`(`internal/org/regate_test.go:642`)、`TestRegate_MarksTheLockWithTheOrgItWorksOn`(`:1259`)、`TestOrgSpawn_RefusedWhileRegateRunsOnTheOrg`(`internal/org/spawn_test.go:5887`)、`TestOrgSpawn_RegateLockStates`(`:5919`)、`TestMerge_SharesRegateLockWithRegate`(`merge_run_test.go:619`)、`TestCleanupFeatureWorktree_MarkerRefusesSpawnOfTheOrg`(`cleanup_test.go:826`) |
| `TestStartFeature` | `internal/org/feature_test.go` の `TestStartFeature_*` 26 本。片付けの間の断りは `TestStartFeature_RefusedWhileRegateRunsOnTheOrg`(`:1532`) |

Edge cases:

| 計画の項目 | テスト |
| --- | --- |
| 同じ org_id を別の機能で立て直した記録 | `RefusedAtTheLedgerAndTheRecord/a record on another branch`(`cleanup_test.go:961`)、`the org runs another feature`(`:948`)、`TestOrgCleanup_LeavesOutCleanedAndReboundBranches`(`internal/cli/org_cleanup_test.go:326`)、`TestMergeOrder_BindingBoundToNamesTheOrgsLaterBranch`(`internal/org/mergeorder_test.go:1177`) |
| 記録はあるが worktree のディレクトリがない | `ResumesWhatAnEarlierCleanupLeft/the directory deleted, its registration left`(`cleanup_test.go:643`。もう 1 つのなくなった worktree の登録が残ることも見る) |
| gitignore されたファイルだけがある worktree | `TestCleanupFeatureWorktree_IgnoredFilesGoWithTheWorktree`(`cleanup_test.go:311`) |
| `regate.lock` を別のプロセスが持っている | 一部。`TestCleanupFeatureWorktree_RefusedWhileRegateLockIsHeld`(`cleanup_test.go:1071`)と CLI の `RefusalsAreSkippedLines/regate.lock held` は、同じプロセスの中で別に開いた fd で flock を持つ。flock は開いたファイルごとに効くので別のプロセスと同じ扱いになるが、別のプロセスが持つ形のテストはない(Test gaps の 1) |

AC5 の文の固定は `TestOrgSkill_PinsTheCleanupAfterAMerge`(`internal/org/org_skill_send_sync_test.go:240`)、`TestDirectorSkill_PinsTheCleanupAfterAMerge`(`internal/org/director_skill_sync_test.go:928`)、leader の雛形の `TestRenderRolePrompt_Leader_ReportThenDisbandAsLastCommand`(`internal/org/prompts_test.go:444`)と `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:493`)、help の `TestOrgCleanup_Help`(`internal/cli/org_cleanup_test.go:700`)、`TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge`(`:759`)、`TestOrgMerge_Help`(`internal/cli/org_merge_test.go:1021`)、`TestOrgRegate_Help`(`internal/cli/org_regate_test.go:723`)、regate の案内の `TestRegateCheckWorktree_QuotesTheBranchInTheCleanupCommand`(`internal/org/regate_test.go:588`)が受け持つ。どれも `run-test.sh` の中で通った。

## Regression checks

新しいテストが、確かめ 0〜6、merge の opt-in、CLI の選び方と終了コードを本当に見ているかを、変異で確かめた。worktree のファイルは変えず、1 か所だけの文字列の置き換えを `go test -overlay` で渡した。org 層は `-run 'Cleanup'`(merge の 3 つは `-run '^TestMerge'`)、CLI は上の基準の回と同じ絞り方で打った。前後の `git status --porcelain --ignored` は同じだった。表の件数は `--- FAIL` の行の数で、サブテストを含む。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| M01 確かめ 5(ブランチの先頭が merge された head)を飛ばす | 落ちた | `RefusedInTheWorktree` の 3 場面、`MovedBranchKeptUnderItsTip`、`HeadNotHereNamesItsFetch` ほか |
| M02 確かめ 4 が未コミットと未追跡を見ない | 落ちた | `an uncommitted change`、`untracked files`、`TestMerge_RefusedCleanupKeepsTheMerge/an untracked file` ほか |
| M03 `--untracked-files=all` を `no` に | 落ちた | `untracked files`、`TestMerge_RefusedCleanupKeepsTheMerge/an untracked file` ほか |
| M04 走っている org を通す | 落ちた | `the org runs`、`TestMerge_RefusedCleanupKeepsTheMerge/the org spawned again during gh pr merge` |
| M05、M06 worktree の下から、worktree そのものから打ったのを見ない | 落ちた | `run below the worktree`、`run in the worktree` |
| M07〜M09 記録の kind・path・branch を比べない | 落ちた | `a record of another kind`、`a record at another path`、`a record on another branch`、`RecordOfAnotherPathOrKindNamesTheChecks`、`TestMerge_CleanupOutcomesWithFakeWorktrees/a record on another branch` |
| M10 ディレクトリが消えた worktree の登録を `git worktree remove` で外さない | 落ちた | `ResumesWhatAnEarlierCleanupLeft` の登録の残る 2 場面 |
| M11 `worktree_cleaned` を書かない | 落ちた | 14 件(`SquashMergedBranch`、`IgnoredFilesGoWithTheWorktree`、再開の各場面ほか) |
| M12 `regate.lock` の marker に別の org_id を書く | 落ちた | `MarkerRefusesSpawnOfTheOrg`、`KeepsTheCallersLock` ほか 9 件 |
| M13 detached HEAD の断りを消す | 落ちた(文で) | `a detached HEAD with a commit`。この変異でも次の「別のブランチ」の確かめが断るので、落ちたのは理由の文の違いによる。消さないことは変わらない |
| M14 rebase・merge の途中を見ない | 落ちた | `a rebase in progress`、`a merge in progress` |
| M15 worktree があるときの確かめ 3〜6 をすべて飛ばす | 落ちた | 27 件 |
| M16 merge が記録されていなくても片付ける(`!res.Recorded` を `!res.Merged` に) | 残った(等価) | `mergeAndRecord`(`internal/org/merge_run.go:412`)は `Merged` で `Recorded` でないとき必ず err を返し、片付けを呼ぶ `switch` は先に `err != nil` を見る。外から見て違いが出ない |
| M17 `RepoRoot` がなくても片付けを呼ぶ(opt-in を外す) | 落ちた | 29 件。トップレベルの 10 本のうち 9 本は base からあるテスト(`TestMerge_MergesWithTheMethodAndRecordsPRMerged`、`TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch` など)で、残りの 1 本は新しい `TestMerge_RunsTheCleanupOnlyAfterARecordedMerge` |
| M18 中断された merge でも片付けを呼ぶ | 落ちた | `TestMerge_RunsTheCleanupOnlyAfterARecordedMerge/interrupted once gh pr merge started` |
| M19 `--branch` なしで merged でない PR も選ぶ | 落ちた | `TestCleanupSteps_PicksTheMergedBranches`、`TestOrgCleanup_BranchWhosePRIsNotMergedIsRefused` |
| M20 `--branch` なしで結びつきの古いブランチも選ぶ(`BoundTo` を見ない) | 落ちた | `TestCleanupSteps_PicksTheMergedBranches`、`TestOrgCleanup_LeavesOutCleanedAndReboundBranches`。この回は 2 件が落ちたあと、load 約 100 で `-timeout 9m` に達した |
| M21 `--branch` で merged でない PR を通す | 落ちた | `PicksTheMergedBranches/an open PR`、`/a closed PR`、`BranchWhosePRIsNotMergedIsRefused` |
| M22 gh がなくても進む | 落ちた | `PicksTheMergedBranches/gh not found` の 2 つ、`TestOrgCleanup_WithoutGHRemovesNothing` |
| M23 1 本残っても exit 0 | 落ちた | `TestOrgCleanup_RefusalsAreSkippedLines` の 3 場面、`TestCleanupPrinter_EachOutcome` ほか 12 件 |
| M24 片付けが断られたら merge が exit 1 | 落ちた | `TestOrgMerge_CleansUpTheFeatureWorktree/an untracked file`、`TestPrintMergeResult_EachOutcome` ほか 17 件 |
| M25 merge の CLI が repo root を渡さない | 落ちた | `TestOrgMerge_CleansUpTheFeatureWorktree/cleaned up` ほか 11 件 |
| M26 `NothingLeft` を SKIPPED として出す | 落ちた | `TestOrgCleanup_CleansUpABranchMergedOnGitHub/again, with nothing left`、`LeavesOutCleanedAndReboundBranches` ほか |

M20 の回だけ、2 件が落ちたあとに時間切れになった。M21〜M26 は `-timeout 25m -skip OrgMergeOrder` で打ち直し、どれも時間切れにならなかった。

## Test gaps

1. `regate.lock` を別のプロセスが持つ形のテストがない。今のテストは同じプロセスの別の fd で flock を持つ。flock は開いたファイルごとなので振る舞いは同じと読む。未確認です
2. 片付けのあとに `worktree_cleaned` を書けなかったとき(`internal/org/cleanup.go:765-768`)を org 層で起こすテストがない。CLI の行の形(`CLEANED: ... (<error>)`、`CLEANUP: done (<error>)`)は printer の表のテスト(`TestCleanupPrinter_EachOutcome`、`TestPrintMergeResult_EachOutcome`)が見る
3. ディレクトリの消えた worktree の登録を外す `git worktree remove` が失敗したとき(`cleanup.go:731-736`、Failed と Manual)を打つテストがない
4. `runCleanupSteps`(`internal/cli/org_cleanup.go:255`)の、PR の読めないブランチの SKIPPED の行と、ブランチの間で中断されたときの道は、端から端では打たれていない。行の形は printer のテストが、そのブランチを選ぶことは `TestCleanupSteps_PicksTheMergedBranches` が見る
5. 確かめ 6 の「ブランチは checkout されているが HEAD が merge された head でない」(`cleanup.go:707-709`)は打たれていない。確かめ 5 を通ったあとは HEAD がブランチの先頭と同じなので、ふつうの状態では起きない守りの道と読む
6. merge の側の CLI のテストはリモートのブランチを消さない(verify の report も同じ指摘)。squash merge と削除のあとの片付けは cleanup の側のテストが見て、merge の側も同じ `cleanupRun.remove` と `--force-branch` を通る
7. 実機の director と leader が新しい手順 15 と skill の文のとおりに動くことは確かめていない

1〜6 は、どれも確かめ 0〜6 と「exit 0 は記録した merge」の判定を変えない道で、変異で確かめた主な判定はすべてテストに固定されている。

## cycle 2(HEAD 9a6ae47c)

- Date: 2026-10-29
- Tester: tester サブエージェント(`/test`、cycle 2)
- Scope: cycle 2 でコードが変わったのは 3 コミット。8e46db0f(S-X1: cleanup の PR の読み方を origin の repository に固定する。`internal/org/mergeorigin.go`、`MergeOrderParams.Origin`、CLI の origin の断り)、d3452a9d(S-X2: ディレクトリの消えた worktree の登録が binding のブランチにいなければ断る `checkRegistration`)、54fa6351(C2-F: `registrationAtRisk` を `checkRegistration` と `recordGone` が共有し、`foreignPRWhy` が origin を確かめるよう言う)。テストの関数は `732529d9..HEAD` で 10 本増え(`internal/org/cleanup_test.go` 2、`internal/org/merge_test.go` 2、`internal/org/mergeorder_test.go` 3、`internal/cli/org_cleanup_test.go` 3)、消えたものはない。ほかは文書と report。`./scripts/run-test.sh` は changed scope のまま打ち、選ばれた pack は golang だった
- Evidence: `docs/evidence/test-2026-10-28-org-merge-cleanup-cycle2.log`(gitignore、5,599 行。`run-test.sh` の全出力、基準の絞った回の PASS の行、変異 13 個の差分と各回の FAIL の行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-29-120824.log`。cycle 1 の証拠のログは書き換えていない

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 9 パッケージ | shell 45 本 OK(PASS を含む行 4,796)、Go 9 パッケージ ok | 0 | 1(cycle 1 と同じ `test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行) | 1,782 秒(21:08:24〜21:38:06 JST)。開始時の load 14.0、終了時 13.9、途中で 51 まで上がった |
| └ Go: `internal/org` | 1,260 本(トップレベル、HEAD の `go test -list`) | ok | 0 | 0 | 959.0 秒 |
| └ Go: `internal/cli` | 878 本(同。cycle 1 の 845 本との差には main の merge の分も入る) | ok | 0 | 0 | 790.5 秒 |
| └ Go: `cmd/ralph`、`internal/{config,insights,org/driver,scaffold}` | — | ok | 0 | 0 | 0.8〜3.7 秒 |
| └ Go: `internal/{org/protocol,upgrade}` | — | ok(`(cached)`) | 0 | 0 | — |
| 基準の絞った回: `go test -count=1 -v -run 'Cleanup\|MergeOrder\|MergeOrigin\|^TestMerge' ./internal/org/` | 158(サブテスト込み 640) | 158 | 0 | 0 | 185 秒 |
| 基準の絞った回: `go test -count=1 -v -run 'OrgCleanup\|CleanupSteps\|OrgMerge\|PrintMergeResult' ./internal/cli/` | 50(サブテスト込み 133) | 50 | 0 | 0 | 282 秒 |
| 変異 13 個(`go test -overlay`、下の Regression checks) | 13 | 13 個とも落ちた | — | — | 1 個 28〜299 秒 |
| coverage の回: `go test -count=1 -coverprofile -run 'Cleanup\|MergeOrder\|MergeOrigin\|^TestMerge\|CheckMerge' ./internal/org/` | — | ok | 0 | 0 | 286 秒(終わったときの load 158) |

負荷は、同じマシンの別の session の Xcode のビルドとシミュレータ(`swift-frontend` など)だった。変異と基準の回は、`run-test.sh` の間に 1 本ずつ順に打った。時間切れや負荷による失敗はなく、lead が挙げた既知の flaky な 2 本(`TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent/an_attach_--replace`、`TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge/while_a_script_runs`)も、`run-test.sh` の中で通った。`internal/org/protocol` と `internal/upgrade` の `(cached)` は、`go list -deps -test` でどちらも `internal/org` と `internal/cli` に依らないことを確かめた。

### Coverage(cycle 2)

- Statement: coverage の回の profile で、`internal/org/cleanup.go` 277/311(89.1%。cycle 1 は 243/277)、`internal/org/mergeorigin.go` 22/23(95.7%)
- Function: cycle 2 で足した、または変えた関数のうち、`checkRegistration`、`registrationAtRisk`、`recordGone`、`keptBranch`、`lookupMergePRs`、`MergeOrigin.Arg`・`hasPR`・`foreignPRWhy` が 100%、`run` 93.5%、`readState` 87.5%、`ReadMergeOrigin` 93.3%
- Branch: 計測なし(Go の道具は statement だけ)
- Notes: 打たれていないブロックのうち cycle 2 の変更に入るのは、`ReadMergeOrigin` の `run == nil`(`internal/org/mergeorigin.go:91-92`)だけ。絞った org の回は偽の exec を渡すためで、nil を渡す CLI の道は `TestOrgCleanup_OriginThatNamesNoGitHubRepositoryIsRefused` などが実 git で打つ。`run` と `readState` の残りは cycle 1 からある道(パスがディレクトリでない、`os.Stat` や `rev-parse` の失敗)。`internal/cli` の profile は、org の coverage の回が終わったとき load が 150 を越えていたので取らなかった。CLI の新しい関数 `cleanupReadOrigin`・`cleanupOriginRefusal`・`cleanupLookupWhy` は、下の変異 C2-M12・C2-M13 と `TestOrgCleanup_OriginRefusalForEachProblem`(問題の値 6 つ)が見る

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

`run-test.sh`、基準の回、coverage の回で失敗はなく、打ち直しもしていない。

### cycle 2 の変更とテストの対応

S-X1(PR の読み方を origin に固定):

| 変更 | テスト |
| --- | --- |
| origin の URL の読み方: https、scp の形、ssh.github.com、大文字の GHE は読む。ローカルパス、file URL、owner なし、`-` で始まる host・owner・repo は repository なし | `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`(`internal/org/merge_test.go:636`、10 行) |
| get-url の失敗、時間切れ、中断、timeout 0 と負 | `TestMergeOrigin_GetURLThatFailsTimesOutOrIsInterrupted`(`:675`、5 つ) |
| `Origin` があると gh に `-R github.com/o/r`、get-url を打たない、大文字の url の PR も使う | `TestMergeOrder_PinnedOriginReadsThePRsInItsRepository`(`internal/org/mergeorder_test.go:2224`) |
| 別の owner、別の repository 名、別の host、http、repository を名指さない url の PR で、全ブランチが PR なしの skipped になり、同じ `PRLookupWhy` と note を持つ。gh は打ち直さない | `TestMergeOrder_PinnedOriginLeavesEveryPROutForAPROfAnotherRepository`(`:2264`、5 つ) |
| `PRLookupWhy`: failed は gh の誤り、found・none・limit_forks は空、gh がないとき、origin の URL と違う repository のとき。JSON には出ない | `TestMergeOrder_PRLookupWhySaysWhyGHsAnswerWasNotUsed`(`:2310`、3 つ) |
| CLI: 別の owner/name、別の host の MERGED の PR は SKIPPED と exit 1、`--branch` も同じ理由で断る。gh の呼び出しはどれも `-R` 付き | `TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`(`internal/cli/org_cleanup_test.go:232`、2 つ) |
| CLI: origin がローカルパス、file URL、origin がないとき、gh を打たずに断る(`--branch` ありとなし) | `TestOrgCleanup_OriginThatNamesNoGitHubRepositoryIsRefused`(`:279`、3 つ) |
| CLI: 問題ごとの断りの文 | `TestOrgCleanup_OriginRefusalForEachProblem`(`:751`、6 つ) |
| merge の `readOrigin` を `ReadMergeOrigin` に置き換えても、merge の断りが変わらない | 前からある `TestCheckMerge_*` と `^TestMerge` のテスト(coverage の回と基準の回で通過) |

S-X2 と C2-F(ディレクトリの消えた登録の確かめ):

| 変更 | テスト |
| --- | --- |
| ディレクトリの消えた登録が、detached(PR が merge していないコミット)、別のブランチ、消えた binding のブランチ(HEAD がゼロ)、読めない HEAD(ゼロ)のどれかなら何も消さない。理由は登録の HEAD と、退避のコマンドか reflog の場所を出す。Manual をそのまま打つと登録・ブランチ・記録が消え、別のブランチは残る | `TestCleanupFeatureWorktree_GoneWorktreeRegisteredOffItsBranchRemovesNothing`(`internal/org/cleanup_test.go:805`、4 つ) |
| 記録がなく、登録が detached で、ディレクトリが残る場合と消えた場合: 同じ文で HEAD と退避を出す。退避と Manual をそのまま打つと、コミットは退避のブランチに残る | `TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead`(`:1008`、2 つ) |
| 記録がなく、登録だけが残り、ブランチが消えた(HEAD がゼロ) | `TestCleanupFeatureWorktree_RecordGoneRemovesNothing/only the registration left`(`:959`) |
| 再開の道は変わらない(worktree を消した、ディレクトリを消して登録が binding のブランチに残る、記録だけ、ブランチが動いた 2 つ) | `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`:705`、5 つ。どれも基準の回で通過) |

lead が挙げた端の場面:

- ゼロの HEAD: `GoneWorktreeRegisteredOffItsBranchRemovesNothing` の `the binding's branch, deleted` と `a HEAD git cannot read`(どちらも登録の HEAD は 0 が 40 個)、`RecordGoneRemovesNothing/only the registration left`
- 別のブランチ: `GoneWorktreeRegisteredOffItsBranchRemovesNothing/another branch`(Manual のあとも `other` が登録の HEAD に残ることを見る)
- ディレクトリが残る場合と消えた場合: `RecordGoneNamesADetachedRegistrationsHead` の 2 つ。ディレクトリがあり記録もあるときは cycle 1 からある確かめ 6 の道(`RefusedInTheWorktree/a detached HEAD with a commit`、`another branch`)
- GitHub の repository を名指さない origin: `OriginThatNamesNoGitHubRepositoryIsRefused` の 3 つ、`ReadsOriginsRepositoryOrSaysWhyNot` の repository なしの 6 行
- 別の host や owner の PR: `MergedPROfAnotherRepositoryRemovesNothing` の 2 つ、`PinnedOriginLeavesEveryPROutForAPROfAnotherRepository` の 5 つ。merge の側は `TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の別の host と別の owner/repo の 4 行

### Regression checks(cycle 2)

cycle 2 の判定をテストが本当に見ているかを、cycle 1 と同じやり方の変異で確かめた。worktree のファイルは変えず、1 か所だけの文字列の置き換えを `go test -overlay` で渡した。`cleanup.go` の変異は `-run 'Cleanup' ./internal/org/`、`mergeorigin.go` と `mergeorder.go` の変異は `-run 'MergeOrder|MergeOrigin|^TestMerge' ./internal/org/`、CLI の変異は `-run 'OrgCleanup|CleanupSteps' ./internal/cli/` で打った。C2-M8 と C2-M9 は、merge の側と CLI の側でも落ちることを見るため、`-run 'CheckMerge' ./internal/org/` と CLI の `-run 'TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing'` でも打った。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じで、`git status --porcelain` は空のまま。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| C2-M1 `run` が `checkRegistration` を通さない(ディレクトリの消えた登録をそのまま外す) | 落ちた | `GoneWorktreeRegisteredOffItsBranchRemovesNothing` の 4 つすべて |
| C2-M2 `recordGone` が `registrationAtRisk` の文を足さない(`why != "" && false`) | 落ちた | `RecordGoneNamesADetachedRegistrationsHead` の 2 つ、`RecordGoneRemovesNothing/only the registration left` |
| C2-M3 `readState` が登録の `detached` を記録しない | 落ちた(文で) | `GoneWorktreeRegisteredOffItsBranchRemovesNothing/a detached HEAD with a commit the PR did not merge`、`RecordGoneNamesADetachedRegistrationsHead` の 2 つ。この変異でも登録は「読めるブランチがない」として断られ、退避のコマンドも同じなので、落ちたのは理由の文の違いによる |
| C2-M4 `registrationAtRisk` が binding のブランチの消えた登録を通す | 落ちた | `GoneWorktreeRegisteredOffItsBranchRemovesNothing/the binding's branch, deleted`、`RecordGoneRemovesNothing/only the registration left` |
| C2-M5 ゼロの HEAD をコミットとして扱い、退避のコマンドを出す | 落ちた | `the binding's branch, deleted`、`a HEAD git cannot read`、`only the registration left` |
| C2-M6 別のブランチの登録にも退避のコマンドを出す | 落ちた | `GoneWorktreeRegisteredOffItsBranchRemovesNothing/another branch` |
| C2-M7 `readState` の「この worktree の登録」の印が次の登録に持ち越される(`ours = ours \|\| samePath(...)`) | 落ちた | `ResumesWhatAnEarlierCleanupLeft/the directory deleted, its registration left` だけ。あとに並ぶ別のなくなった worktree の `branch other` を読み、別のブランチの登録として断った |
| C2-M8 `MergeOrigin.hasPR` が host を比べない | 落ちた | `PinnedOriginLeavesEveryPROutForAPROfAnotherRepository/o/r on another host`、`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の別の host の 2 行(merge の `isOriginsPR` も同じ関数を使う)、CLI の `MergedPROfAnotherRepositoryRemovesNothing/o/r on another host` |
| C2-M9 `hasPR` が owner/repo を比べない | 落ちた | `PinnedOriginLeavesEveryPROutForAPROfAnotherRepository/another owner`、`/another repository name`、`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の別の owner/repo の 2 行、CLI の `/another owner/name` |
| C2-M10 `Origin` があっても gh に `-R` を渡さない | 落ちた | `TestMergeOrder_PinnedOriginReadsThePRsInItsRepository` |
| C2-M11 `Origin` があるとき、別の repository の PR を使う(`foreignPRWhy` を見ない) | 落ちた | `PinnedOriginLeavesEveryPROutForAPROfAnotherRepository` の 5 つすべて |
| C2-M12 CLI が `Origin` を merge-order に渡さない | 落ちた | `TestOrgCleanup_CleansUpABranchMergedOnGitHub`(gh の呼び出しに `-R` がない)、`MergedPROfAnotherRepositoryRemovesNothing` の 2 つ |
| C2-M13 CLI が GitHub の repository を名指さない origin を通す | 落ちた | `OriginThatNamesNoGitHubRepositoryIsRefused` のローカルパスと file URL、`OriginRefusalForEachProblem/no repository` |

### Test gaps(cycle 2)

1. 実際の gh が `-R <host>/<owner>/<repo>` を受けて、GHE の PR の url をテストの形で返すことは確かめていない。テストは偽の gh で見る(verify の cycle 2 と同じ)
2. ssh の別名の origin(`~/.ssh/config` の Host)で cleanup が全ブランチを SKIPPED にすることを打つテストはない。計画 `:216` の記録どおり PR の既知の穴
3. git 2.48 以降の `worktree.useRelativePaths=true` では gitdir が相対パスになり、C2-L1 の文の `<worktree>/.git` と文字では合わない。テストは絶対パスの既定だけを打つ(self-review cycle 2 run 2 の、所見にしていない穴)
4. C2-M7 を落とすのは `ResumesWhatAnEarlierCleanupLeft/the directory deleted, its registration left` の 1 つだけで、git が別の登録をこの worktree の登録のあとに並べることに依る。並びが逆の環境ではこの変異は残るかもしれない。未確認です
5. `internal/cli` の cycle 2 の coverage は測っていない(上の Coverage の Notes)
6. cycle 1 の Test gaps の 1〜7 は、cycle 2 の変更で変わっていない

どれも cycle 2 の判定(登録の確かめ 6、origin への固定、origin の断り)を変えない道で、変異で確かめた判定はすべてテストに固定されている。

## Verdict

- Verdict(cycle 2、HEAD 9a6ae47c): pass
- Pass: `./scripts/run-test.sh` rc 0(shell 45 本 OK、Go 9 パッケージ ok、`internal/org` と `internal/cli` は `(cached)` なし)。cycle 2 の新しいテスト 10 本と、前からある再開のテストが通った。変異 13 個はすべて落ちた
- Fail: 0
- Blocked: 0

### cycle 1(HEAD 44f13b7f)

- Pass: `./scripts/run-test.sh` rc 0(shell 45 本 OK、Go 9 パッケージ ok、`internal/org` と `internal/cli` は `(cached)` なし)。計画の Test plan の項目すべてに対応するテストがあり、「`regate.lock` を別のプロセスが持つ」だけは同じプロセスの別の fd で代えている。変異 26 個のうち 25 個が落ち、残った 1 個は等価
- Fail: なし
- Blocked: なし
