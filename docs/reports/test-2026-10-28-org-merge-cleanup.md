# Test report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(`## Test plan` と AC1〜AC6、AC3b)。issue #263
- Tester: tester サブエージェント(`/test`、cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD 44f13b7f)。Go は `internal/org` と `internal/cli`、ほかに skill・文書・template。`./scripts/run-test.sh` は changed scope のまま打ち、選ばれた pack は golang で、`go test ./...` が 9 パッケージを `(cached)` なしで打った。テストの関数はこの diff で 47 本増え、消えたものはない
- Evidence: `docs/evidence/test-2026-10-28-org-merge-cleanup.log`(gitignore。`run-test.sh` の全出力、変異の表と各回の結果、coverage の数字。wrapper 自身の写しは `docs/evidence/verify-2026-10-28-113631.log`)

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

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 45 本 OK、Go 9 パッケージ ok、`internal/org` と `internal/cli` は `(cached)` なし)。計画の Test plan の項目すべてに対応するテストがあり、「`regate.lock` を別のプロセスが持つ」だけは同じプロセスの別の fd で代えている。変異 26 個のうち 25 個が落ち、残った 1 個は等価
- Fail: なし
- Blocked: なし
