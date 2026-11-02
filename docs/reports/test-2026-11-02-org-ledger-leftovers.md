# Test report: org-ledger-leftovers

- Date: 2026-11-02(run-test.sh は 13:24:08 から 14:04:01 JST)
- Plan: `docs/plans/active/2026-11-02-org-ledger-leftovers.md`(`## Test plan`、AC2〜AC7、AC8 のうち run-test.sh)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD 6c400e87、merge-base は origin/main c8c8c0f1。実行中に branch は動かなかった)。Go は `internal/org` と `internal/cli`。`./scripts/run-test.sh` は changed scope のまま打ち、Language scope は changed(changed_languages)で golang pack が選ばれた。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence:
  - `docs/evidence/test-2026-11-02-org-ledger-leftovers-real-binary.md`(コミットする。build した binary で AC3 を確かめたスクリプトと出力)
  - `docs/evidence/test-2026-11-02-org-ledger-leftovers.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。`run-test.sh` の全出力 5,618 行、名指しのテストの `-v` の出力、空白を含む一時ディレクトリでの 4 回の出力、coverage の 2 回と変更行の集計)
  - wrapper 自身のログは `docs/evidence/verify-2026-11-02-042410.log`(同じく対象外。名前の日付は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | shell 2 行、Go 2 本(下の注) | 2,393 秒。load は開始時 21.90 / 11.54 / 10.22、終了時 10.46 / 8.49 / 8.06 |
| └ Go: `internal/org` | — | ok | 0 | 1(Linux だけのテスト) | 1,250.3 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 1(Linux だけのテスト) | 917.3 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/scaffold`、`internal/upgrade` | — | ok | 0 | 0 | 13.6 秒、4.5 秒、2.4 秒、3.4 秒 |
| └ Go: `internal/insights`、`internal/org/driver`、`internal/org/protocol`、`cmd/ralph` | — | ok | 0 | 0 | 1.6 秒、2.6 秒、1.6 秒、1.0 秒 |
| 名指しのテスト、`internal/org`(`go test -count=1 -v`、下の「Test plan とテストの対応」) | 親 20、サブテスト 74 | 親 19、サブテスト 74 | 0 | 1 | 48 秒 |
| 名指しのテスト、`internal/cli`(同上) | 親 12、サブテスト 17 | 親 11、サブテスト 17 | 0 | 1 | 79 秒 |
| 空白を含む一時ディレクトリ、HEAD:`go test -count=1 -v -run 'Cleanup' ./internal/org/` | 親 50 | 41 | 9(サブテスト 16) | 0 | 161 秒、rc 1 |
| 空白を含む一時ディレクトリ、HEAD:`go test -count=1 -v -run 'OrgCleanup' ./internal/cli/` | 親 25 | 24 | 1 | 0 | 253 秒、rc 1 |
| 同、base c8c8c0f1(`git archive` の写し):`-run 'Cleanup' ./internal/org/` | 親 48 | 39 | 9(サブテスト 16) | 0 | 118 秒、rc 1 |
| 同、base c8c8c0f1:`internal/cli` の 2 本(`TestOrgCleanup_LeavesOutCleanedAndReboundBranches`、`TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead`) | 2 | 0 | 2 | 0 | 25 秒、rc 1 |
| coverage の回、`internal/org`(`-run 'ShellWord\|HintStateDir\|VerbCommand\|Cleanup\|AbsPath\|CheckMerge\|TestMerge_'`) | — | ok | 0 | — | 292 秒 |
| coverage の回、`internal/cli`(`-run 'ShellWord\|HintStateDir\|OrgCleanup\|OrgMerge_\|PrintMergeResult\|RetiredStateDirEnv\|Legacy\|OrgRead\|RetryStateDir'`) | — | ok | 0 | — | 476 秒 |
| build した binary での AC3 の確かめ | 3 ケース + 2 項目 | 5 | 0 | 0 | 下の「AC3: 実物の確かめ」 |

起動の仕方: `run-test.sh` は Bash tool の 600 秒の上限を超えるので、`run_in_background` で runner を起動し、log と開始・終了・rc の meta を scratchpad に書かせた。待つあいだに項目 2〜4 を進め、同じ手番の中で wrapper の pid を `caffeinate -w` で待って meta を読んだ。完了の通知もそのあとで届いた。

所要時間について: 13:27 ごろから 13:52 ごろまで、名指しのテスト、空白の回、base の回、最初の coverage の回を run-test.sh と同時に打った。`internal/org` の 1,250 秒と `internal/cli` の 917 秒は、standard-config-gaps の回(load 3〜4 で 481 秒と 333 秒)より長い。30 分の時間切れ(1,800 秒)には届かず、既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も落ちなかったので、run-test.sh は打ち直していない。10 パッケージのどれも `(cached)` ではなかった。

最初の coverage の回(`internal/org`、`-run 'ShellWord|HintStateDir|VerbCommand|Cleanup|AbsPath'`)は、自分で付けた `-timeout 9m` に当たって `panic: test timed out after 9m0s` で止まった。そのとき走っていたのは `TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead`(42 秒目)で、何分も止まった goroutine はなかった。run-test.sh の Go の段と重なって遅くなっただけで、テストの失敗ではない。run-test.sh が終わってから、範囲を広げて打ち直した回(上の表)は 292 秒で ok だった。

Skipped について:

- shell の 2 行は、どちらもこの diff と関係のない手元の環境による。`tests/test-plan-visual.sh` の「browser detection (a browser in /Applications would start)」は、`/Applications` にブラウザがある機械で本物のブラウザを起こさないための SKIP。`tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」は、手元の git が 2.49.0 で、古い git は stub の行が見ている
- Go の 2 本は、この branch が足した Linux だけのテスト `TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`(`internal/org/abs_path_getwd_test.go:24`)と `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`(`internal/cli/org_retired_state_dir_env_abs_test.go:105`)。どちらも darwin では「os.Getwd does not fail for a deleted working directory on darwin」で SKIP する(計画の D7)。CI の ubuntu-latest が打つ。計画の Progress には、implementer が docker の linux/arm64 で 2 本とも PASS させた記録がある

## Coverage

- Statement: 絞った回の profile なので、パッケージ全体の数字(`internal/org` 35.0%、`internal/cli` 47.2%)には意味がない。代わりに、この diff が足した非テストの行のうち statement を含む行が、絞った回で実行されたかを数えた(`git diff -U0 origin/main...HEAD` の追加行と cover profile のブロックの突き合わせ。S1 で移した `cleanup_text.go` と F1 で移した `shellword.go` は全行を追加行として数えた)
  - `internal/org`: 197 行のうち 193 行。実行されなかった 4 行は `cleanup_text.go:108-109`(`pathWithin` の `return false`。S1 で移しただけの main からある分岐)と `regate_ci.go:120`、`:132`(`cleanupShellWord` を `ShellWord` に替えた 2 行。絞った回は regate のテストを含まない)
  - `internal/cli`: 29 行のうち 29 行
- Branch: 計測なし(Go の道具は statement だけ)
- Function: `shellword.go` の `ShellWord`、`VerbCommand`、`shellQuote` は 100%。`cleanup_text.go` の 16 関数は `pathWithin` の 92.9% のほかすべて 100%。`cleanup.go` の `checkCleanupParams` 100%。merge 側で変えた関数(`checkMerge`、`checkMergeBranch`、`checkMergeAfterChecks`、`again`、`verbCommand`、`runningRefusal`、`viewPR`、`checkOriginUnmoved`、`mergeAndRecord`、`otherBaseMerged`、`mergedElsewhere`、`recordMerge`、`newMergeWait`、`waited`、`waitedOut`、`enforceUnread`、`readEnforcementRepository`)は 100%、`mergeGHSaid` 91.7%、`CheckMergeEnforcement` 83.3%。CLI の `cleanupRetryCommand`、`orgVerbHint`、`cleanupNotInLedger`、`orgReadCommandHint`、`guardLegacyOrgStateDir`、`checkRetiredOrgStateDirEnv`、`resolveOrgLedger`、`printMergeResult` は 100%、`newOrgMergeCmd` 91.2%、`newOrgCleanupCmd` 90.9%
- Notes: 関数の数字のうち 100% でないものは、どれもこの diff が変えていない分岐を含む(変更行は上の数え方で 193/197 と 29/29)。`regate_ci.go` の 2 行が run-test.sh の全体の回で実行されたかは測っていない

## Failure analysis

`run-test.sh`、名指しのテスト、coverage の 2 回、実物の確かめには失敗がない。

失敗したのは、項目 3 の空白を含む一時ディレクトリの回だけ。`GOTMPDIR` と `TMPDIR` をどちらも `<scratchpad>/test-gamma/tmp with space` に向けて打った(Go 1.27 の `t.TempDir` は `GOTMPDIR` を先に読む)。HEAD で `internal/org` の 9 本(サブテスト 16)と `internal/cli` の 1 本が落ち、base c8c8c0f1 でも同じ 9 本・16 サブテストと cli の 1 本が落ちる。この branch の変更による失敗ではない。どれもテストの側が期待のコマンドを quote なしで組むためで、コードの出力は `ShellWord` で正しく quote されている(失敗の出力の `git -C '<一時ディレクトリ>'` を見た)。直していない。/sync-docs で tech-debt に書く材料として、各テストの最初の失敗の行を挙げる(`<T>` は空白を含む一時ディレクトリの下の、テストごとのディレクトリ)。

| Test | Error(最初の行) | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestCleanupFeatureWorktree_RefusedInTheWorktree`(`internal/org/cleanup_test.go:469`)の 3 サブテスト: the branch moved past the merged head、the branch moved, another branch checked out、the local branch gone, the worktree detached | `cleanup_test.go:573: refusal "… look at them (git -C '<T>' log …..feat/auth) …" does not say …`。detached の行は `Manual = "cd '<T>' && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch", want "cd <T> && …"` も出す | 期待の文の `git -C " + fx.root`、`cd`、`fx.scriptCmd()`(`cleanup_test.go:103-105`)が quote なし | 期待を `ShellWord(fx.root)` などで組む(S5 が cli の HeadNotHere で行った形)。空白のサブテストを足す |
| `TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch`(`:761`)の 2 サブテスト: the PR's number known、not known | `cleanup_test.go:789: refusal "… fetch the head (git -C '<T>' fetch origin …"`、続いて `cleanup_test.go:797: git -C <T> fetch origin refs/pull/7/head: exit status 128` | 同上。797 行は、テストが自分で組んだ quote なしのコマンドを `sh -c` で打ち、shell が空白で分けた | 同上。打つコマンドも `ShellWord` で組む |
| `TestCleanupFeatureWorktree_MovedBranchKeptUnderItsTip`(`:821`) | `cleanup_test.go:844: refusal "… (git -C '<T>' log …..feat/auth) …"` | 同上 | 同上 |
| `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`:883`)の 2 サブテスト | `cleanup_test.go:944: refusal "… (git -C '<T>' log …"`、続いて `:951: git -C <T> branch feat/auth-unmerged-… feat/auth: exit status 128` | 同上(期待と、`sh -c` で打つコマンド) | 同上 |
| `TestCleanupFeatureWorktree_GoneWorktreeRegisteredOffItsBranchRemovesNothing`(`:983`)の 4 サブテスト | `cleanup_test.go:1046: refusal "the directory of the feature worktree <T>/.claude/worktrees/org-auth is gone …"` と `Manual = "git -C '<T>' worktree remove '<T>/.claude/worktrees/org-auth' && …"`。続いて `:1065` か `:1077: exit status 128` | 同上。1077 行は `registrationReflog` が `sh -c` で打つ `fx.registrationCommonCmd()`(`:276-277`、`"git -C " + fx.root`)が quote なし | 同上。`registrationCommonCmd` も `ShellWord` で組む |
| `TestCleanupFeatureWorktree_RecordGoneRemovesNothing`(`:1116`)の 3 サブテスト | `cleanup_test.go:1169: Manual = "git -C '<T>' worktree remove '<T>/.claude/worktrees/org-auth' && git -C '<T>' branch -D feat/auth", want "git -C <T> worktree remove <T>/.claude/worktrees/org-auth && git -C <T> branch -D feat/auth"` | 同上 | 同上 |
| `TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead`(`:1186`)の 2 サブテスト | `cleanup_test.go:1217: refusal "the worktree record org-auth … does not exist …"`、続いて `:1225: git -C <T> branch feat/auth-unmerged-… …: exit status 128` | 同上 | 同上 |
| `TestCleanupFeatureWorktree_ScriptFailureRecordsNothing`(`:1247`) | `cleanup_test.go:1255: cleanup = {… Manual:cd '<T>' && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch …}` | `res.Manual != fx.scriptCmd()` の比べ方で、`scriptCmd` が quote なし | `scriptCmd` を `ShellWord` で組む |
| `TestCleanupFeatureWorktree_FakeScriptFailure`(`:1640`) | `cleanup_test.go:1652: cleanup = {… Manual:cd '<T>' && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch …}` | 同上 | 同上 |
| `TestOrgCleanup_LeavesOutCleanedAndReboundBranches`(`internal/cli/org_cleanup_test.go:479`) | `org_cleanup_test.go:514: cleanup --branch feat/old: … stdout "SKIPPED: feat/old … (git -C '<T>/work' log …) …\nMANUAL: git -C '<T>/work' branch -D feat/old\n"; want … "\nMANUAL: git -C <T>/work branch -D feat/old\n"` | `wantManual`(`org_cleanup_test.go:512`)が `"\nMANUAL: git -C " + fx.work` で quote なし | 1 行を `org.ShellWord(fx.work)` で組む |

`internal/org/cleanup_test.go` は tech-debt 386 行の 4 つのファイルの 1 つ(1,817 行)で、計画の D6 はこのファイルにテストを足さない。quote なしの `git -C " + fx.root`・`" + fx.worktree`・`"cd " + fx.root` を含む行は 18 行ある。386 行の分割と一緒に直すかどうかは lead と /sync-docs が決める。計画の「cli の側は通る(AC6)」は、AC6 が名指す `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` については成り立つ(空白の回でも 2 サブテストとも PASS)。`-run 'OrgCleanup'` の全体では、上の cli の 1 本が残る。

## Test plan とテストの対応

この branch が足したテスト 19 本(うち 1 本は `TestCleanupShellWord` の改名)は、項目 2 の名指しの回ですべて打った。17 本が PASS、Linux だけの 2 本が darwin で SKIP。

| 計画の項目 | テスト | 結果 |
| --- | --- | --- |
| S2・AC2: 6 つの値で CLI と org の層が同じ形 | `TestShellWord_QuotesOnlyWhatAShellReads`、`TestShellWord_OrgRetryCommandWritesTheBranchAndTheStateDirWithIt`、`TestShellWord_QuotesWhatAShellReadsInAPath`(org)、`TestShellWord_PrintedCommandsSpellAValueLikeTheOrgLayer`(cli、a plus・a colon・an at sign・a comma・a space・a single quote の 6 サブテスト) | PASS |
| S3・AC3: 文の組ごと | `TestVerbCommand_AddsTheHintStateDirOnlyWhenGiven`、`TestCleanupFeatureWorktree_HintStateDirReachesMergeOrderWithNoBranch`、`TestCheckMerge_HintStateDirReachesTheOtherVerbs`、`TestCheckMergeEnforcement_HintStateDirReachesMergeOrder`、`TestMerge_HintStateDirReachesMergeOrderOnceGHPRMergeRan`、`TestMerge_HintStateDirReachesStatusAndDisbandOnceTheWaitIsUsedUp`(org)、`TestOrgMerge_HintStateDirInTheRefusalOfAMissingBranch`、`TestPrintMergeResult_HintStateDirInTheNoOutcomeError`、`TestOrgCleanup_HintStateDirInTheRefusalOfABranchNotInTheLedger`(cli) | PASS |
| S3・AC3: 2 つの台帳で打ち直す | `TestOrgCleanup_HintStateDirMergeOrderReplaysOnTheLedgerNamedFirst`(サブテスト the personal file names the ledger を含む) | PASS |
| S3 で `-run HintStateDir` に当たる既存のテスト | `TestEscalate_HintStateDirReachesTheBannerAndTheRecoveryCommands`、`TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir`、`TestOrgSpawn_StopHints_RepeatInboxHintStateDir`、`TestWatchGateError_TheCommandsRepeatHintStateDir`、`TestFeatureLeaderTask_StateDirIsOneShellWord`、`TestFeatureLeaderTask_ScopeIsOneShellWord` | PASS |
| S4・AC4: help と、`--branch` なしの 2 つの場合 | `TestOrgCleanup_Help`、`TestOrgCleanup_RefusalsAreSkippedLines`(an untracked file・regate.lock held・the org runs)、`TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`(another owner/name・o/r on another host)、`TestOrgCleanup_WithoutGHRemovesNothing` | PASS |
| S4・AC5: 「remove what is left by hand」 | `TestCleanupFeatureWorktree_RecordGoneRemovesNothing`、`TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead` | PASS |
| S5・AC6: HeadNotHere の quote と空白 | `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead`(the temp dir・a temp dir with a space) | PASS(名指しの回と、空白の一時ディレクトリの回の両方) |
| S5・AC7: `st.notDir`、`git worktree remove` の失敗、`ok == false` | `TestCleanupFeatureWorktree_NotADirectoryAtTheWorktreesPathRemovesNothing`、`TestCleanupFeatureWorktree_FailedRemovalOfAMissingWorktreesRegistrationRemovesNothing`(org)、`TestOrgRetiredStateDirEnv_RefusalOfALedgerElsewhereNamesTheWorkingDirectorysLocalFile`(cli、git-main-worktree・git-toplevel・cwd) | PASS(org の 2 本は空白の一時ディレクトリの回でも PASS) |
| S5・AC7: Linux だけの 2 本 | `TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`、`TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved` | darwin で SKIP。CI の ubuntu-latest が打つ |
| Integration、Regression | `./scripts/run-test.sh` 全体 | rc 0 |

## AC3: 実物の確かめ

スクリプトと出力は `docs/evidence/test-2026-11-02-org-ledger-leftovers-real-binary.md` にある。HEAD 6c400e87 から build した ralph を、scratchpad の中の使い捨ての repository(bare の origin つき)で `env -i` のもとで打った。cleanup は origin の URL から GitHub の owner/repo を読めないと台帳を読む前に断るので、origin の URL を `git@github.com:o/r.git` にし、repo の `core.sshCommand` を、bare の origin の upload-pack を返す偽の ssh にした。gh は呼ばれた引数を記録して `gh pr list` に `[]` を返す偽物で、本物の gh とネットワークには届いていない。

| ケース | 結果 |
| --- | --- |
| (a) `ralph org cleanup --branch feat/none --state-dir '<S>/my ledger'` | rc 1。拒否の文は `ralph org merge-order --all --state-dir '<S>/my ledger' lists the ledger's branches` で終わる。そのコマンドを取り出して `sh -c` でそのまま打つと(PATH の ralph は build したもの)rc 0 で、名指した台帳の feat/a を列挙した。時刻を伏せると、その台帳を直接読んだ一覧と一致し、既定の台帳(`.harness/state/org`、feat/default)の一覧とは違う |
| (b) 同じものを `--state-dir` なしで | rc 1。文は `ralph org merge-order --all lists the ledger's branches` で、`--state-dir` は 0 件。写して打つと既定の台帳の一覧と一致 |
| (c) `ralph org merge` を `--branch` なしで | `--state-dir '<S>/my ledger'` では `(ralph org merge-order --state-dir '<S>/my ledger' lists the ledger's branches)`、quote の要らないパスでは quote なし、`--state-dir` なしでは `(ralph org merge-order lists the ledger's branches)`。3 つとも rc 1 |
| 台帳 | 2 つの manifest の sha256 は前後で同じ |
| 偽物の呼ばれ方 | 偽の gh 6 回、偽の ssh の upload-pack 7 回(準備の 1 回を含む)。merge の 3 回はどちらも呼んでいない |

gh やネットワークが要って打てなかったケースはない。

## Regression checks

計画の Progress のとおり、S3 と S5 の implementer は `go test -overlay` の変異で新しいテストが分岐を見ていることを確かめている(2 つの台帳の打ち直しは hint を無視する写しで落ち、`st.notDir`、`git worktree remove` の失敗、`ok == false` のテストは分岐を壊した写しで落ちた)。この回では変異を打っていない(任意)。代わりに、空白の一時ディレクトリで base と HEAD を比べた。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 420(c): `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` が空白を含む一時ディレクトリで落ちる | 直った | base c8c8c0f1 の写しでは空白の回で落ちた(`org_cleanup_test.go:424` と `:434`、`git -C <T> … exit status 128`)。HEAD では、名指しの回でも空白の回でも、2 サブテストとも PASS |
| 空白の一時ディレクトリで落ちる `internal/org` の既存のテスト | 変わらない(この branch は増やしも減らしもしていない) | base と HEAD で、落ちる親 9 本とサブテスト 16 本の名前が同じ。HEAD の親の数が 2 多い(48 → 50)のは、この branch が足した `cleanup_run_test.go` の 2 本で、どちらも空白の回で PASS |
| 空白の一時ディレクトリで落ちる `internal/cli` の既存のテスト | 変わらない | `TestOrgCleanup_LeavesOutCleanedAndReboundBranches` は base でも同じ形で落ちる(`org_cleanup_test.go:489` の比べ方。HEAD では S1 の移動で `:514`) |

## Test gaps

1. 空白を含む一時ディレクトリで、`internal/org/cleanup_test.go` の 9 本(サブテスト 16)と `internal/cli/org_cleanup_test.go` の `TestOrgCleanup_LeavesOutCleanedAndReboundBranches` が落ちる(上の Failure analysis)。どれも base からある、テストの側の期待の quote の問題で、この branch では直していない。/sync-docs で tech-debt に書く。org の側は 386 行のファイル(D6)
2. Linux だけの 2 本は darwin で SKIP した。結果は PR の CI(ubuntu-latest、amd64)で見る
3. `regate_ci.go:120`、`:132`(名前の置き換えだけの 2 行)は、絞った coverage の回では実行されていない。全体の回の coverage は測っていない
4. 実物の確かめは、CLI の層の 2 つの文(cleanup の「台帳にない branch」の拒否と、merge の「`--branch` がない」拒否)だけを binary で出した。org の層の 15 文(merge の gh pr merge のあとや wait の使い切りなど)は gh の PR を読む repo が要るので binary では出しておらず、テスト(`merge_hint_state_dir_test.go` など)だけが見ている
5. 変異はこの回では打っていない。Progress に記録された slice の変異に拠る
6. run-test.sh の `internal/org` は、同時に打った回の負荷で 1,250 秒かかり、30 分の上限の 69% に達した。負荷の高い機械で run-test.sh と別の Go テストを同時に打つと、上限に近づく

## Verdict

- Pass: run-test.sh rc 0(shell 45 本、Go 10 パッケージ、`(cached)` なし)、この branch が足したテスト 19 本のうち 17 本が PASS(2 本は darwin で SKIP する Linux だけのテスト)、名指しの既存のテストも PASS、coverage の 2 回は ok、AC3 の実物の確かめの 5 項目
- Fail: 0(gate の対象のテストで失敗はない。空白を含む一時ディレクトリの回で落ちた 10 本は、base でも同じく落ちる既存のテストの期待の問題で、上の Failure analysis と Test gaps 1 に書いた)
- Blocked: 0
- Verdict: pass。/sync-docs へ進んでよい。空白の回の 10 本は tech-debt に書く材料として /sync-docs に渡す
