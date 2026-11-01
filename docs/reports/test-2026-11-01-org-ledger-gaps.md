# Test report: org-ledger-gaps

- Date: 2026-11-01(run-test.sh は 2026-11-01 23:36 から 2026-11-02 00:00 JST)
- Plan: `docs/plans/active/2026-11-01-org-ledger-gaps.md`(`## Test plan`、AC1〜AC10、AC11 のうち run-test.sh)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。前の tester はセッションの再起動で途中で切れ、repository に何も書いていない。この回は最初からやり直した)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD 81deeebf。origin/main は動いていない)。Go は `internal/config`、`internal/org`、`internal/org/driver`、`internal/cli`。`./scripts/run-test.sh` は changed scope のまま打った(選ばれた pack は golang)。test mode なので `tests/test-*.sh` の 45 本もすべて走った。テストの関数はこの diff で 30 本増えた。ほかに 10 本が `internal/config/local_test.go` から別のファイルへ移った(名前は前後で同じ、消えたテストはない)
- Evidence:
  - `docs/evidence/test-2026-11-01-org-ledger-gaps-real-binary.md`(commit する。PR #290 の test gaps 3・4 を build した binary で確かめたコマンドと出力)
  - `docs/evidence/test-2026-11-01-org-ledger-gaps.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。`run-test.sh` の全出力 5,618 行、`(cached)` だったパッケージの打ち直し、`go test -list` の本数、変更した関数の coverage、変異 13 個の定義と FAIL の行)
  - wrapper 自身のログは `docs/evidence/verify-2026-11-01-143609.log`(同じく対象外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 1,481 秒(23:36:08〜00:00:49 JST)。load は開始時 5.12 / 5.13 / 13.65、終了時 5.13 / 6.05 / 7.47 |
| └ Go: `internal/org` | 1,408 本(トップレベル、`go test -list`) | ok | 0 | 0 | 797.2 秒 |
| └ Go: `internal/cli` | 975 本(同) | ok | 0 | 0 | 665.0 秒 |
| └ Go: `internal/config` | 107 本 | ok | 0 | 0 | 9.9 秒 |
| └ Go: `internal/org/driver`、`internal/gitroot` | 75 本、6 本 | ok | 0 | 0 | 1.2 秒、2.7 秒 |
| └ Go: `cmd/ralph`、`internal/{insights,org/protocol,scaffold,upgrade}` | — | ok `(cached)` | 0 | 0 | — |
| 打ち直し: `go test -count=1 ./cmd/ralph ./internal/insights ./internal/org/protocol ./internal/scaffold ./internal/upgrade` | 5 パッケージ | ok | 0 | 0 | 0.6〜2.5 秒 |
| coverage の回: `go test -count=1 -coverprofile ./internal/config/ ./internal/org/driver/` | 107 + 75 | ok | 0 | 0 | 15.2 秒 + 1.4 秒 |
| coverage の回: `go test -count=1 -coverprofile -run 'ResolveOrgStateDir\|FeatureRepoRoot\|LedgerMainWorktreeRoot\|LedgerLocalConfigPath\|MustAbs\|Cleanup\|Merge_\|MergeRun\|StateDir' ./internal/org/` | — | ok | 0 | 0 | 183.9 秒 |
| coverage の回: `go test -count=1 -coverprofile -run 'OrgCleanup\|OrgMerge\|CleanupRetryCommand' ./internal/cli/` | — | ok | 0 | 0 | 310.6 秒 |
| coverage の回: `go test -count=1 -coverprofile -run 'RetiredStateDirEnv\|RetiredOrgStateDir\|CheckRetiredEnv\|RunDoctorOpts_LoadError\|DoctorHelp\|ConfigGet\|AgmsgHome\|OrgEscalate\|OrgLedger' ./internal/cli/` | — | ok | 0 | 0 | 41.7 秒 |
| 変異 13 個(`go test -overlay`、下の Regression checks) | 13 | 13 個とも落ちた | — | — | 1 個 1〜25 秒 |
| 実物の binary(PR #290 の test gaps 3・4) | 2 | 2 | 0 | 0 | 下の「PR #290 の test gaps 3・4」 |

起動の仕方: `run-test.sh` は Bash tool の 600 秒の上限を超えるので、lead の指示どおり Bash tool の `run_in_background` で起動した。小さな runner が log と開始・終了・rc の meta を書いた。待つあいだに gap 3・4 を進め、終わりの通知を受けてから meta を読んだ。

Skipped の 2 行は、どちらもこの diff と関係のない手元の環境による。

- `tests/test-plan-visual.sh` の「browser detection (a browser in /Applications would start)」。`/Applications` にブラウザがある機械で、本物のブラウザを起こさないための SKIP
- `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git は 2.49.0 で、古い git は stub の行が見ている

signal を使う shell のテストは SKIP しなかった。

`run-test.sh` の `go test ./...` は、`internal/cli`・`internal/config`・`internal/gitroot`・`internal/org`・`internal/org/driver` を `(cached)` なしで打った。残る 5 パッケージは `(cached)` だった。そのうち `cmd/ralph` と `internal/insights` は変更したパッケージに依存する(`go list -deps`)ので、5 つとも `-count=1` で打ち直し、すべて ok だった。30 分の時間切れにはならなかった。既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` もこの回は通ったので、打ち直しはしていない。

## Coverage

- Statement: `internal/config` はパッケージ全体で 97.3%、`internal/org/driver` は 95.6%。org と cli は `-run` で絞った回の profile なので、パッケージ全体の数字(org 37.4%、cli 34.9%)には意味がない。変更した行を含む関数は次のとおり(cli は 2 回の profile をブロックごとに合わせた)。
  - config: `IsAbsOrHomePath` 100%、`ExpandHomePath` 100%、`LocalOrgStateDir` 100%、`LocalOrgStateDirAt` 100%、`readLocalFile` 100%、`checkLocalAgmsgHome` 85.7%
  - driver: `expandTilde` 100%、`ResolveAgmsgHome` 100%
  - org: `ResolveOrgStateDir` 100%、`defaultOrgStateDirBase` 100%、`LedgerLocalConfigPath` 100%、`ledgerBaseDir` 100%、`mustAbs` 100%、`MainWorktreeRoot` 100%、`orgRetryCommand` 100%、`cleanupAgain` 100%、`cleanupCommand` 100%、`ledgerMoveStep` 100%、`checkCleanupParams` 100%、`mergeCheck.again` 100%、`mergeCleanup` 100%、`mergeCleanupInterrupted` 100%、`mergeLocked` 92.6%、`CleanupFeatureWorktree` 86.7%、`cleanupFeatureWorktreeLocked` 77.8%、`remove` 76.2%
  - cli: `resolveOrgLedger` 100%、`cleanupRetryCommand` 100%、`cleanupOriginRefusal` 100%、`cleanupReadOrigin` 100%、`mergeCleanupRoot` 100%、`agmsgHomeIsRelative` 100%、`wrapHelpLines` 100%、`newConfigGetCmd` 100%、`newOrgEscalateCmd` 96.0%、`orgWatchChild` 94.4%、`checkRetiredOrgStateDirEnv` 94.1%、`runCleanupSteps` 92.3%、`cleanupSteps` 88.2%、`runDoctorStateDir` 88.2%(足した `cfg = config.Default()` のブロックは通っている)、`cleanupNamedStep` 72.7%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes: 変更した行を含むブロックのうち、絞った回で打たれなかったのは次の 3 つだけ。
  - `internal/cli/org_legacy_ledger.go:153-155`: `org.LedgerLocalConfigPath` が false を返したときに `config.LocalPath("ralph.toml")` で求め直す道。既定の層の source なら台帳のパスは必ず `.harness/state/org` で終わるので、通常は来ない守り(self-review の L7)
  - `internal/org/cleanup.go:457-459`: feature worktree のパスにディレクトリでない物があるときの拒否。main からある分岐で、この diff は再実行の文を `cleanupAgain(c.p)` で作るようにしただけ。cleanup のテストにこの場合の fixture はない(`notDir` も「is not a directory」も cleanup のテストに出てこない)
  - `internal/org/cleanup.go:1003-1009`: 登録の残った feature worktree で `git worktree remove` が失敗したときの文。これも main からある分岐
  - どれも `cleanupAgain` と `orgRetryCommand` そのものは 100% で、`--state-dir` を付ける規則は別の分岐のテストが見ている。`cmd/ralph` などの数字は測っていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

`run-test.sh`、`(cached)` の打ち直し、coverage の 4 回のどれにも失敗はなかった。

## Test plan とテストの対応

| 計画の項目 | テスト(この diff で増えたもの) |
| --- | --- |
| AC1: cleanup と merge の再実行のコマンドに `--state-dir`、`--branch` の有無を保つ、台帳を動かす案内は例外 | `TestCleanupRetryCommand_QuotesEachWordAndAddsTheStateDirOnlyWhenGiven`(`internal/org/cleanup_retry_state_dir_test.go:28`、`;`・`$(x)` の branch、空白と `'` の台帳)、`TestCleanupFeatureWorktree_RefusalsBeforeGitRepeatTheStateDir`・`_WorktreeRefusalsRepeatTheStateDir`・`_LedgerMoveStepNamesNoOldStateDir`、`TestMerge_RetryCommandsRepeatTheStateDir`・`TestMerge_CleanupRepeatsTheStateDir`、`TestOrgCleanup_RefusalsAndInterruptionsNameTheStateDir`・`_RetryCommandKeepsTheBranchAndTheStateDir`・`_NoMainWorktreeNamesTheBranchAndTheStateDir`、`TestOrgMerge_CleanupRootNamesTheStateDir`・`_RefusalQuotesALedgerDirectoryWithASpace` |
| AC1: 2 つの台帳で出力のコマンドを打ち直す | `TestOrgCleanup_RetryCommandReplaysOnTheLedgerNamedFirst`、`TestOrgMerge_RetryCommandsReplayOnTheLedgerNamedFirst` |
| AC2: 判定と展開を 1 つずつ、6 つの値の表、`expandTilde` の home のエラー | `TestHomePath_ChecksAndExpansionsAgree`(`~/x`、`~`、`~user/x`、相対、絶対、空)、`TestExpandHomePath_UserHomeDirFails`、`TestExpandTilde_AgreesWithConfigExpandHomePath`、`TestExpandTilde_KeepsTheValueWhenUserHomeDirFails`、`TestAgmsgHomeIsRelative_ComplementsConfigIsAbsOrHomePath` |
| AC3: doctor は Load のエラーで `Default()` | `TestRunDoctorOpts_LoadErrorUsesDefaults` |
| AC4: 断る条件の定数と 4 つの場合の表 | `TestRetiredOrgStateDirRefusalCondition` |
| AC5、AC6: help | `TestDoctorHelp_StrictFlagHasNoValueName`、`TestConfigGet_HelpNamesTheLocalOnlyKeysOfExit3`、`TestConfigGet_WrapHelpLines` |
| AC7: git の探索は 1 回、拒否は git を打たない | `TestResolveOrgStateDir_LooksForItsDirectoryOnce`、`TestLedgerLocalConfigPath`、`TestLocalOrgStateDirAt_ReadsTheGivenFileWithoutGit`、`TestOrgRetiredStateDirEnv_RefusalNamesThePersonalFileWithoutGit` |
| AC8: `local_test.go` を分ける | 移した 10 本が同じ名前で `local_only_keys_test.go` と `local_state_dir_test.go` にあり、run-test.sh の中で通った |
| AC9: bare repository の linked worktree、`filepath.Abs` の失敗 | `TestFeatureRepoRoot_LinkedWorktreeOfBareRepositoryHasNoMainWorktree`、`TestMustAbs_KeepsThePathAsGivenWhenAbsFails`、`TestOrgRetiredStateDirEnv_KeepsARelativeValueAsGivenWhenAbsFails` |
| AC10: 拒否の頭と escalate | 全文を固定した既存のテスト 4 本(`TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger` ほか)と `TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman` |
| Edge cases: `RALPH_LOCAL_CONFIG=off` | `TestLocalOrgStateDir_LocalConfigEnv`(`internal/config/local_state_dir_test.go:156`、S1 で移したもの) |
| 回帰: 既存の org と cli のテスト | `run-test.sh` の中で org 1,408 本、cli 975 本がすべて通った |

## Regression checks

implementer は各スライスで変異を確かめている(計画の Progress: S2 5 個、S6 5 個、S7 9 個、S8 8 個、S9 2 個)。この回では、`cleanupRetryCommand` を呼ぶ CLI の層の 6 か所(helper そのものは 100% でも、呼ぶ所ごとに値を落とす変異は別に生き残りうる)と、計画の中心の判定を選び、独立に確かめ直した。worktree のファイルは変えず、1 行の文字列の置き換えを `go test -overlay` で渡し、関係するテストに `-run` で絞った。どの変異も build は通った。前後の `git status --porcelain --ignored` は同じだった。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| c1 main worktree がないときの文が `--state-dir` を落とす(`internal/cli/org_cleanup.go:166`、`cleanupRetryCommand(branch, "")`) | 落ちた | `TestOrgCleanup_NoMainWorktreeNamesTheBranchAndTheStateDir` |
| c2 origin が読めないときの文が落とす(`:231`) | 落ちた | `TestOrgCleanup_RefusalsAndInterruptionsNameTheStateDir` の every_merged_branch と --branch |
| c3 `cleanupSteps` の拒否の文が落とす(`:277`) | 落ちた | 同上の 2 件 |
| c4 `cleanupNamedStep` の「merge されたか分からない」の文が落とす(`:330`) | 落ちた | 同上の --branch |
| c5a `runCleanupSteps` の中断の文が `--state-dir` を落とす(`:381`) | 落ちた | 同上の 2 件 |
| c5b `runCleanupSteps` の中断の文が `--branch` を落とす(`:381`、`cleanupRetryCommand("", hintStateDir)`。plan advisory 1) | 落ちた | 同上の --branch |
| c6 merge の中の cleanup の main worktree がない文が落とす(`internal/cli/org_merge.go:291`) | 落ちた | `TestOrgMerge_CleanupRootNamesTheStateDir` |
| m-doctor Load のエラーで `Default()` を使わない(`internal/cli/doctor.go:111` を `_ = config.Default()` に、AC3) | 落ちた | `TestRunDoctorOpts_LoadErrorUsesDefaults` |
| m-expand `expandTilde` が home のエラーで空を返す(`internal/org/driver/agmsg.go:180`、AC2) | 落ちた | `TestExpandTilde_KeepsTheValueWhenUserHomeDirFails` |
| m-sentinel 拒否を `errOrgLedgerUnresolved` で包む(`internal/cli/org_legacy_ledger.go:107`、AC10) | 落ちた | `TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger`、`_EscalateStillReachesTheHuman`、`_NamingTheLegacyLedgerRefusesStop`、`TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger` |
| m-escalate-refused escalate が拒否の sentinel を記録しない道へ送らない(`internal/cli/org_inbox.go:102` の 2 つ目の `errors.Is` を `false` に) | 落ちた | `TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman` |
| m-escalate-unresolved escalate が解決できない台帳の sentinel を送らない(同じ行の 1 つ目を `false` に) | 落ちた | `TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman` の not_TOML と relative_state_dir |
| m-once `ResolveOrgStateDir` が個人用のファイルを `config.LocalOrgStateDir("")` で読み、探索をもう一度する(`internal/org/statedir.go:80`、AC7) | 落ちた | `TestResolveOrgStateDir_LooksForItsDirectoryOnce` |

m-sentinel は文の頭と sentinel を一緒に変えるので、escalate のテストは文の違いで落ちた。sentinel の扱いだけを変える 2 つ(m-escalate-*)も別に落ちることを確かめた。

## PR #290 の test gaps 3・4(実物の binary)

HEAD から build した binary を、worktree の外の一時の git repository で打った。コマンドと出力は `docs/evidence/test-2026-11-01-org-ledger-gaps-real-binary.md` にある。

| Gap | 結果 |
| --- | --- |
| 3a 実物の `ralph.local.toml` の `state_dir = "~/.ralph-probe-2cd83524/ledger"` | `ralph status` が `state-dir=/Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger (source: config)` を示した。読むだけで、HOME の下には何も作られなかった |
| 3b `RALPH_ORG_STATE_DIR` が別の場所を指し、個人用のファイルが台帳を決めている | `ralph status` も `ralph org spawn` も注意を 1 行出して進んだ。spawn はそのあと watch gate に断られ(見張りがない)、どこにも何も書かれなかった |
| 3c 個人用のファイルを消した | `ralph org spawn` は「ralph org spawn: refusing to change the org ledger: RALPH_ORG_STATE_DIR names …」で exit 1。`ralph.local.toml` のパスは台帳から作られ repository の root を指した。repository に `.harness` はできず、env の指す場所も空のまま |
| 4 `--state-dir` 付きの `ralph org watch --all --detach`(env あり) | 子の watch.log に `RALPH_ORG_STATE_DIR` の注意がちょうど 1 行(`grep -c` で 1)。`ralph org watch --stop --state-dir …` で止まり、`pgrep -fl 'ralph org watch'` は前後とも 0 件 |

作ったパス(`$HOME/.ralph-probe-2cd83524`、一時の repository、env 用と watch 用の一時の台帳、build した binary)は、確かめが終わってすぐ、この report を書く前に消した。消えたことは evidence の「後片付け」にある。

## Test gaps

1. `internal/org/cleanup.go:457-459`(feature worktree のパスにディレクトリでない物があるときの拒否)と `:1003-1009`(`git worktree remove` の失敗)は、この diff で再実行の文が `--state-dir` を持つようになったが、どちらの分岐もテストが打たない。main からある穴で、文を作る `cleanupAgain` は他の分岐のテストで 100%。ディレクトリの代わりにファイルを置く fixture 1 つで前者は閉じられる
2. `internal/cli/org_legacy_ledger.go:153-155`(`LedgerLocalConfigPath` が false のときの求め直し)は打たれていない。既定の層の source では来ない守り(self-review の L7)
3. `os.Getwd` の失敗による `filepath.Abs` の失敗は、テストではパッケージの変数の差し替えで起こしている(D7)。Linux で cwd を消したときの本物の失敗は、この機械(darwin)では確かめられない
4. 実物の binary の確かめは main checkout の root からだけ。linked worktree やサブディレクトリから打った場合はテスト(`TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree` など)だけが見ている。`ralph org start` は見張りを起こして残すので、実物では打っていない
5. 計画が範囲の外に置いた案内(merge と cleanup が出す `ralph org merge-order`、`merge.go` の `ralph org inbox`、`merge_wait.go` の `ralph org status --org-id`・`ralph org disband --org-id`)は `--state-dir` を持たず、そのことを見るテストもない。計画の Progress のとおり /sync-docs で tech-debt に書く項目

どれも計画の判定(再実行のコマンドの `--state-dir` と `--branch`、`~/` の判定と展開、doctor の既定値、`RALPH_ORG_STATE_DIR` の拒否と頭、git の探索の回数)を変えない道で、その判定は上の変異で落ちることを確かめた。

## Verdict

- Pass: run-test.sh rc 0(shell 45 本、Go 10 パッケージ)、`(cached)` の 5 パッケージの打ち直し、coverage の 4 回、変異 13 個すべて落ちた、PR #290 の test gaps 3・4 を実物の binary で確かめた
- Fail: 0
- Blocked: 0
- Verdict: pass。/sync-docs へ進んでよい
