# Test report: org-stop-all

- Date: 2026-10-07(JST。実行の記録は UTC の 2026-10-07 13:51〜14:54)
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1
- Scope: branch feat/org-stop-all の HEAD de154798 と origin/main(f423f230)の差分(29 ファイル、+5832/-165)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC15 のうちテストで確かめられる部分を見た。AC9 の文書の部分は /verify の担当で、ここでは leader の雛形の順序だけをテストにした
- Added tests: f63ae025(`test: close org-stop-all coverage gaps found by mutation`、テストファイル 3 本、+134 行)。本番のコードは変えていない
- Evidence: `docs/evidence/test-2026-10-07-org-stop-all.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-07-135142.log`(HEAD)と `docs/evidence/verify-2026-10-07-144820.log`(テスト追加後)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(既定の changed、HEAD de154798) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 673 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(HEAD) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 103 s、rc 0 |
| `go test ./internal/cli/ ./internal/org/ ./internal/org/driver/ -count=1 -v -coverpkg=./internal/org/...,./internal/cli/`(HEAD、パッケージをまたいだ coverage) | 3 パッケージ | 3 | 0 | 0 | rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1`(HEAD) | 4 パッケージ | 4 | 0 | 0 | 109 s、rc 0。DATA RACE の報告なし |
| 差分で増えた 69 本(org 50、driver 6、cli 13)を `-count=3 -run '^(...)$'` | top-level 207 回、subtest 411 回 | すべて | 0 | 0 | 91 s |
| 期限の 16 本を `-count=10`(`Unanswered\|TimesOut\|Timeout\|WaitDelay\|GrandchildHolding`)、同時に `internal/cli` 全体を裏で実行 | 160 回 | 160 | 0 | 0 | 前面 rc 0、裏 rc 0 |
| 追加したテストを `-race -count=3` | org の top-level 12 回、cli の 3 回と subtest 30 回 | すべて | 0 | 0 | rc 0 |
| `go test ./... -count=1 -coverprofile`(テスト追加後) | 8 パッケージ | 8 | 0 | 0 | rc 0 |
| `./scripts/run-test.sh`(テスト追加後) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 374 s、rc 0 |
| mutation(私が作った 82 件。scratchpad の `git archive` の写し 3 つで並行に実行し、worktree には触れていない) | ビルドが通った 81 件 | HEAD のテストで red 72 件、追加後に 80 件 | 生き残り HEAD で 9 件、追加後に 1 件 | - | 最初の 78 件は 3 本並行で約 11 分、追加の確認を合わせて約 20 分 |

- `run-test.sh` は 2 回とも `Language scope: changed (changed_languages)`、`Language packs selected: golang` で走った。shell の 39 本は `scripts/verify.local.sh` から走る。件数の数え方は前回と同じで、各 suite の区間で `PASS` / `FAIL` で始まる行を数え、`PASS: 47 / 47` のような集計行は除いた。2,047 件は前回(org-state-dir-common)と同じで、この差分は shell のテストを変えていない。
- `run-test.sh` の中の `go test ./...` は、1 回目は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(2 行目)。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回の実行のどちらでも出なかった(123 / 123)。単独の再実行はしていない。
- top-level の Test 関数の数(`go test -list`、テスト追加後): `internal/org` 357、`internal/org/driver` 48、`internal/cli` 478。
- 実行のあと、main のチェックアウトと worktree のどちらにも `.harness/state/org` はなく、main の `git status --porcelain` は 0 行だった。テストが共通の台帳に書くことはなかった。
- 追加したテストのコンパイル確認として、`gofmt -l internal/` と `go vet ./internal/org/ ./internal/cli/` を 1 回ずつ流した(どちらも出力なし)。判定には使っていない。静的解析の判定は /verify のレポートのとおり。

## AC とテストの対応

| AC | 主なテスト | 結果 |
| --- | --- | --- |
| AC1 C-c → close、見つからなければ閉じ済み | `TestOrgStop_CallOrder_CtrlCThenCloseThenLeaveThenStopped`、`_PaneNotFound_CountsAsClosed`、`_PaneGoneAtCheck_RecordsAlreadyClosed`、既存の `_ExistingSeat_RecordsPaneAndLeaveOutcomes`(期待値が `pane=ok` から `ctrl_c=ok` と `pane=closed` に変わった) | pass。M01・M09・M18 が red |
| AC2 閉じられなければ `stop_failed`、打ち直しで `stopped` | `TestOrgStop_CloseFails_RecordsStopFailedThenRetryStops`、CLI の `TestOrgStopAll_OneCloseFails_ExitsOneThenRerunStopsIt` | pass。M11・M12・M13 が red |
| AC3 disband は座席 → workspace → `disbanded`、失敗なら `disbanded` なし | `TestOrgDisband_ClosesSeatPanesThenWorkspace`、`_SeatCloseFails_LeavesWorkspaceAndRetries`、`_WorkspaceCloseFails_NoDisbandedThenRetry`、`_WorkspaceNotFound_CountsAsClosed`、追加した `_WorkspaceGoneAtCheck_CountsAsClosed`、CLI の `TestOrgDisband_OneOrgSeatFails_NoDisbandedLine` | pass。M26・M27・M34・M37・M72 が red。M79 は HEAD で生き残り、追加したテストで red |
| AC4 `stop --all` | `TestOrgStopAll_StopsEveryActiveSeatAcrossOrgsInOrder`、`_OneSeatFails_OthersStoppedThenRetryStopsIt`、CLI の `TestOrgStopAll_StopsEveryActiveSeatAcrossOrgs` | pass。M46・M73 が red |
| AC5 `disband --all`、失敗した org の再対象 | `TestOrgsToDisband_Definition`、`TestOrgDisbandAll_FailedOrgNotDisbandedThenRetried`、CLI の `TestOrgDisbandAll_OneOrgFails_ExitsOneAndNextRunRetriesIt` | pass。M47・M50〜M53 が red |
| AC6 同時指定のエラー | `TestOrgStopDisbandAll_ConflictingFlags_RejectedWithoutChanges`(5 例) | pass。M67・M68 が red |
| AC7 `--all --dry-run` | `TestOrgStopAll_DryRun_NoDriverCallsDryRunRecordsOnly`、`TestOrgDisbandAll_DryRun_NoDriverCallsDryRunRecordsOnly`、CLI の `TestOrgStopDisbandAll_DryRun_RecordsWithoutDriverCalls` | pass。M21・M39・M44 が red |
| AC8 記録していない id は閉じない | `TestOrgStop_ClosesOnlyTheSeatsRecordedPane`、`TestOrgDisband_ClosesOnlyThisOrgsRecordedIDs` | pass。M23・M25 が red |
| AC9 leader の雛形の締めの順 | 追加した `TestRenderRolePrompt_Leader_ReportThenDisbandAsLastCommand` | pass。M78(report と disband の手順を入れ替える)は HEAD で生き残り、追加したテストで red。文書のほかの部分は /verify の担当 |
| AC10 `--force` | `TestOrgStop_Force_CloseFails_RecordsStoppedForced`、`TestOrgDisband_Force_RecordsPastCloseFailures`、CLI の `TestOrgStopAndDisband_Force_FailuresAreWarningsExitZero` | pass。M14・M36・M38・M45・M54・M71 が red |
| AC11 呼び出しごとの期限 | `TestOrgStop_UnansweredHerdr_EachCallTimesOut`、`_UnansweredLeave_`、`TestOrgStopDisband_UnansweredGet_LeavesItAlone`、`TestOrgStopAll_UnansweredHerdrForOneOrg_OthersStillStopped`、`TestOrgDisbandAll_UnansweredHerdrForOneOrg_OthersStillDisbanded` | pass。負荷をかけた `-count=10` でも 160 / 160。M10(期限なし)は 5 分の test timeout で red、M19 が red |
| AC12 自分の pane と workspace は最後 | CLI の `TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput`、org の `TestOrgDisband_OwnWorkspace_ClosedLastByCaller` ほか、追加した `TestOrgStopDisband_OwnRecordFails_NotHandedBack` | pass。M15・M16・M28〜M32・M42・M43・M48・M49・M55・M69・M70・M76・M77b が red。M20・M81 は HEAD で生き残り、追加したテストで red |
| AC13 disband のあとの spawn は新しい workspace | `TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace`、`TestOpenOrgWorkspaces` | pass。M35・M40・M41 が red |
| AC14 label が違えば閉じない | `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose`(追加後は 6 例 × `Force` の有無)、`TestOrgDisband_WorkspaceNotConfirmed_LeftOpen`、`TestOrgStopDisband_OwnIDNotConfirmed_NotDeferredNotClosed`、`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose` | pass。M02〜M06・M08・M17・M22・M24・M33 が red。M07・M80 は HEAD で生き残り、追加した 2 例で red |
| AC15 `ErrWaitDelay` を成功にする | `TestExecRunner_Run_SuccessNotFailedByGrandchildHoldingPipes`、`_TimeoutNotHeldByGrandchild` | pass。M56・M58 が red。M57 は生き残り(下の Test gaps の 1) |

## 依頼にあった確認点

- `internal/org/verbs.go:704-705`(`confirmSeatPane` の `wsGone`): HEAD では通るテストがなかった。M07(この枝を「確認できた」に変える)は org と cli の全テストで生き残った。pane が見つかって tab の label も合うのに、pane の workspace を herdr が知らない場合に、C-c と close が送られる書き換えである。`TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` に例を足し、M07 が red になることを確かめた。
- `internal/org/verbs.go:1008`(`CloseDeferredSelfPane` の `no seat recorded on it`): HEAD で通っている。`TestOrgCloseDeferredSelfPane_Outcomes/no_seat_recorded_on_it:_left_open` が持ち、S7(05977322)で入った。M23(記録のない pane も閉じる)は red。self-review の「この枝を通るテストはない」は、この行については当たらない。
- verify V-1(古い台帳の判定の表に `--all` がない): M74・M75(`--all` の経路だけ判定を「読むだけ」にする)が生き残り、`internal/cli/org.go:742-744` と `:1012-1014` は coverage でも 0 回だった。表に `stop_all`、`stop_all_dry_run`、`disband_all` の 3 行を足し、M74・M75 が red になることを確かめた。
- verify V-3(leader の雛形の締めの順がテストで固定されていない): M78 が生き残った。テストを足して red にした。
- 依頼にあった mutation は次のとおりで、どれも red。持ち主の確認を外す(M01)、get の失敗を「閉じ済み」にする(pane の get は M02、workspace の get は M03。tab の get は M80 で、HEAD では生き残ったので例を足した)、座席が失敗しても disband が workspace を閉じる(M26)、`stop_failed` を書かない(`stopped` を書く M11、何も書かない M12)、自分の pane を先に閉じる(Stop の中ですぐ閉じる M15、確認の前に自分と判定する M16、disband と stop --all で最後に回さない M28・M42、CLI が出力の前に閉じる M69・M70)。

## Mutation

scratchpad に `git archive HEAD` で写しを作り、1 件ずつ書き換えて対象のパッケージのテストを流し、写しの元のファイルで戻した。worktree の本番のコードには触れていない。並行の 3 本で流したので、期限のテストで落ちた mutant(M03、M10、M19)はログで落ちた理由を見た。どれも負荷ではなく書き換えそのものが理由だった(M19 は「C-c の失敗で close を飛ばした」、M03 は「期限切れの get で disband が通った」、M10 は応答しない fake で 5 分止まった)。

`internal/org` の書き換えは org のテストを先に流し、生き残ったときだけ cli のテストを流した。cli のテストは時間を縮めるため `-run 'TestOrgStop|TestOrgDisband|TestOrgLegacyLedger|TestWithPrefixOnce|TestOrgCleanupVerbs'` で流し、HEAD で生き残った M07 と M20 は cli の全テスト(`-run` なし)でも生き残ることを確かめた。

| 対象 | red になった mutation | HEAD での生き残り |
| --- | --- | --- |
| 持ち主の確認(`confirmSeatPane`、`confirmOrgWorkspace`) | M01〜M06、M08、M09 | M07(pane の workspace が消えた枝)、M80(tab の get の失敗を「閉じ済み」にする) |
| 期限(`callWithTimeout`) | M10 | なし |
| `Stop` / `stopSeatPane` | M11〜M19、M21 | M20(`stopped` の記録に失敗しても自分の pane を返す) |
| 後回しの close(`CloseDeferredSelf*`) | M22〜M25 | なし |
| `Disband` / `closeOrgWorkspace` / `DisbandResult` | M26〜M39 | M79(確認の段で消えていた workspace を失敗にする)、M81(記録に失敗しても自分の workspace を返す) |
| `resolveWorkspace` / `openOrgWorkspaces`(AC13) | M40、M41 | なし |
| `StopAll` / `DisbandAll` / `orgsToDisband` | M42〜M55 | なし |
| driver(`ExecRunner`、herdr の close と get) | M56、M58〜M66 | M57(`ctx.Err() == nil` の条件を外す) |
| CLI | M67〜M73、M76、M77b | M74、M75(`--all` の経路の古い台帳の判定) |
| leader の雛形 | - | M78 |

- M77 は最初の書き方でビルドが通らなかった(使わない変数)。コンパイルが通る形に直して M77b として流した。件数はビルドが通った 81 件。
- HEAD で生き残った 9 件のうち M57 以外の 8 件は、f63ae025 のテストで red になった。追加のテストは既存のテストを変えずに例と関数を足しただけなので、HEAD で red だった 72 件が生き残りに戻ることはない。追加後に流し直したのはこの 8 件だけ。
- 作業の途中で私の手順の誤りが 2 件あった。1 件目は、zsh が `$ids` を単語に分けず、最初の起動で mutant が 1 件も流れなかったこと(`${=...}` で流し直した)。2 件目は、M74・M75 の再確認を追加前のテストの写しで流してしまったこと。この結果は捨て、追加後のテストの写しでの結果(red)を採った。

## 追加したテスト(f63ae025)

| テスト | 閉じた穴 | red / green の確認 |
| --- | --- | --- |
| `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` に 2 例(`workspace gone although the pane was found`、`tab get fails`)× `Force` の有無 | `verbs.go:704-705`(`wsGone`)と `:694-695`(tab の get の失敗)。AC14 と AC11 の「確認できなければ閉じない」 | M07・M80 が red、HEAD のコードで pass |
| `TestOrgStopDisband_OwnRecordFails_NotHandedBack`(stop の自分の pane、disband の自分の workspace) | `Stop` の `selfPane && err == nil`(`:935`)と `closeOrgWorkspace` の記録の失敗(`:1651-1654`)。AC12 の「記録を済ませてから閉じる」。台帳を読み取り専用にして記録を失敗させる。root では skip | M20・M81 が red、HEAD のコードで pass |
| `TestOrgDisband_WorkspaceGoneAtCheck_CountsAsClosed` | `closeOrgWorkspace` の `case gone`(`:1627-1628`)。本物の herdr では、消えた workspace は close より前の `workspace get` で `workspace_not_found` になるので、こちらが「見つからなければ閉じ済み」の普段の経路になる。既存の `_WorkspaceNotFound_CountsAsClosed` は close の段の not-found だけを見ていた | M79 が red、HEAD のコードで pass |
| `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` に 3 行 | verify V-1 | M74・M75 が red |
| `TestRenderRolePrompt_Leader_ReportThenDisbandAsLastCommand` | verify V-3。`## ミッション` と `## 運用規律` のそれぞれで report が disband より前にあり、disband の項目に「最後のコマンド」があること | M78 が red |

## Coverage

- Statement(HEAD → テスト追加後): `internal/cli` 85.2% → 85.3%、`internal/org` 92.0% → 92.4%、`internal/org/driver` 93.1%(変わらず)。前回の snapshot(org-state-dir-common)はそれぞれ 84.8%・91.2%・92.0%。ほかの 5 パッケージは前回と同じ(`config` 92.3%、`insights` 86.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Branch: Go の標準ツールでは取れない
- Function(テスト追加後、各パッケージ自身のテストでの値。括弧は HEAD の値が違うもの):
  - `internal/org/verbs.go`: `Stop` 95.8%、`stopSeatPane` 100%、`confirmSeatPane` 100%(90.9%)、`confirmOrgWorkspace` 100%、`callWithTimeout` 100%、`Disband` 93.3%、`disbandOwnLast` 100%、`closeOrgWorkspace` 100%(88.0%)、`recordWorkspaceClosed` 100%(66.7%)、`appendDisbanded` 50.0%、`CloseDeferredSelfPane` 93.8%、`CloseDeferredSelfWorkspace` 93.8%、`lastSeatOnPane` / `lastOrgOfWorkspace` 100%、`DisbandResult` の `recordStop` / `failSeat` / `failWorkspace` / `forcedNote` 100%
  - `internal/org/verbs_all.go`: `StopAll`、`DisbandAll`、`orgsToDisband`、`splitOwnOrgs`、`holdsCaller`、`add`、`leaveOwnOrg`、`failSeat` は 100%、`StopAllResult.recordStop` 87.5%(`ModelReceipts` に足す行は cli の `TestOrgStopAll_CLI_CodexModelMismatch_PrintsWarning` だけが通る)
  - `internal/org/spawn.go`: `resolveWorkspace` 87.5%、`openOrgWorkspaces` 90.9%
  - `internal/org/driver`: `ExecRunner.Run` 93.8%、`PaneClose` / `WorkspaceClose` / `checkHerdrCloseResult` / `wrapHerdrRunError` / `herdrGetResult` / `NotFound` / `IsNotFound` 100%、`PaneGet` / `TabGet` / `WorkspaceGet` 90.0%
  - `internal/cli/org.go`: `newOrgStopCmd` 96.8%(93.5%)、`newOrgDisbandCmd` 100%(95.7%)、`rejectFlagsWithAll` / `withPrefixOnce` / `printSeatFailure` / `printWorkspaceFailure` / `printStopAllResult` / `closeDeferredSelf` / `printDisbandResult` 100%、`printDisbandAllResult` 95.8%、`printOtherErrs` 75.0%
- Notes: org・driver・cli の 3 パッケージのテストを合わせても通らない行は、台帳の読み込みの失敗(`verbs.go:838`、`:1003`、`:1045`、`:1542`)、`stop_failed` と `disbanded` と `org_workspace_created` の追記の失敗(`:929`、`:1683`、`spawn.go:1716`)、herdr の get の result が JSON として読めない場合(`herdr.go:395`、`:418`、`:442`)、stderr のない非 0 終了(`driver.go:70`)、CLI の `printOtherErrs` が別のエラーを出す行(`org.go:832`)と、`disband --all` が台帳を読めない場合(`:1103`)、`requireSeatIdentifier` の失敗(`:752`)、`getenv` の `os.Getenv` 側(`verbs.go:642`。テストは必ず `o.Getenv` を差し替える)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の `stop` / `disband` のテストの意図(plan の Test plan が求める確認) | 弱めていない | 差分の test ファイルの削除は 21 行で、herdr の stub の 2 行(状態を持つ stub に置き換え)、fake のメソッドの引数名、assert の期待値の書き換えだけだった。`pane=ok` は `ctrl_c=ok` と `pane=closed` の 2 つに分かれ、「pane_id も agmsg_team もない座席に driver を呼ばない」は close の呼び出しも数えるようになった。`TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears` は seat-2 に別の pane を与えた(同じ pane だと fake が seat-1 の tab の label を書き換え、持ち主の確認が seat-1 を拒む) |
| spawn → disband → 同じ org_id で spawn(AC13) | pass | `TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace`。M35・M40・M41 が red |
| 古い台帳の判定(1 段目)が書き換えの動詞を止める | pass、`--all` の 3 つも固定した | `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` の 10 行 |
| watch の deadman と見張りのテスト(`fakeWatchHerdr` に get と close が増えた) | pass | `internal/org` 全体が ok |
| テストが main のチェックアウトの台帳に書かない | pass | 全実行のあと main と worktree に `.harness/state/org` はない |

## Test gaps

1. M57(`ExecRunner.Run` の `errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil` から `ctx.Err() == nil` を外す)が生き残った。この 2 つが同時に成り立つのは、コマンドが期限ちょうどに exit 0 で終わり、孫がパイプを握っている場合だけで、決まった結果を出すテストを書くのは難しい。外しても、期限を過ぎたのに途中の出力で成功になる、という差しか出ない。テストは足していない
2. 失敗の経路のうち、台帳の読み込みの失敗と、`stop_failed` / `disbanded` / `org_workspace_created` の追記の失敗(上の Notes の行)は、テストがない。どれも本番のコードはエラーを返す形になっている。足すなら、追加した `TestOrgStopDisband_OwnRecordFails_NotHandedBack` と同じく台帳を読み取り専用にする形で書ける
3. 本物の herdr は自動のテストで動かしていない。org は Go の fake、cli は sh の stub で、`pane close` / `workspace close` の stub は exit 0 で空の出力を返し、状態のファイルを消さない(close のあとに get しても見つかる)。本物の挙動は実装者の `docs/evidence/herdr-pane-close-2026-10-07.md` に頼っている
4. 別の herdr セッションから打った `stop` で本物の pane が `pane_not_found` と読まれる場合(self-review の F-1 の残余 (c))、`Disband` が台帳を読んだあとに別のプロセスが spawn した座席(F-4)、`--all` 全体の所要時間(F-3)は、テストがない。どれも self-review が tech-debt に送る予定のもの
5. `fakeWatchHerdr` は get をすべて not-found で返すので、watch のテストを通る `Stop` は C-c も close も送らない。watch の側で close の経路を見るテストはない(watch のテストが見る対象ではない)
6. spawn の補償の `compensateStale`(`spawn.go:1811`)が台帳の pane id に確認なしで C-c を送る既存の経路は、この PR の Non-goals で、テストもない

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ ok)。`-count=1` の Go 全体、`-race`、新しいテストの `-count=3`、負荷をかけた期限のテストの `-count=10` もすべて pass。flake は出なかった。mutation は 81 件中、HEAD のテストで 72 件、テスト追加後に 80 件が red になった。生き残りは M57 の 1 件で、Test gaps の 1 に書いた
- Fail: なし
- Blocked: なし
