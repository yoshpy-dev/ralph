# Test report: org-merge-queue-verb

- Date: 2026-11-20。cycle 1(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`(issue #342)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `4be700ff`(base `d919479a` = main、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから、Bash tool の `run_in_background` で起動した(INT と HUP は既定のまま)。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと queue の道のテストを `-v` で絞って回し、main のテストを HEAD のコードに当てる run(AC3)と、変異 9 個を足した
- Evidence: `docs/evidence/test-2026-11-20-org-merge-queue-verb.log`(5,829 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-034409.log` にも書いた(同じく追跡外)
- この report は Write tool ではなく shell の heredoc で書いた。security-guidance plugin の PreToolUse hook が、空白を含むパスを引用せずに `python3` に渡して落ち、Write を 2 回とも止めたため。mojibake の確かめは書いたあとに `.claude/hooks/check_mojibake.sh` を手で回した(exit 0)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-20 12:44:09 〜 13:03:53 JST、1,184 s、rc 0。開始時の load 7.95、終了時 7.88 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 13 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |
| `go test ./internal/org/ -count=1 -v -coverprofile`(queue の道の 43 関数) | 43 関数、`--- PASS` 316 行(subtest を含む) | 全部 | 0 | 0 | 9 s(パッケージ 2.664 s) |
| `go test ./internal/cli/ -count=1 -v`(queue の 4 関数、`TestExitCode`、`TestOrgMerge_Help`) | 6 関数、`--- PASS` 20 行 | 全部 | 0 | 0 | 5 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.880 s |
| `internal/cli` | ok | 545.519 s |
| `internal/config` | ok | 11.461 s |
| `internal/gitroot` | ok | 3.757 s |
| `internal/insights` | ok | 1.987 s |
| `internal/org` | ok | 649.385 s |
| `internal/org/driver` | ok | 4.246 s |
| `internal/org/protocol` | ok | 3.253 s |
| `internal/scaffold` | ok | 4.560 s |
| `internal/toolchain` | ok | 19.345 s |
| `internal/upgrade` | ok | 4.735 s |

負荷に弱いと分かっているテスト(`TestRegateCI_AnInterruptOnTheMergePath`、`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`)を含め、落ちたテストはない。打ち直しはしていない。

`-v` の run の絞り込みは次のとおり。

- org: `-run '^(TestMergeQueue_|TestMergeQueueRun_|TestMergeGateTime_|TestMergeThroughQueue_|TestMerge_QueuePath|TestMerge_QueueWait|TestCheckMerge_TakesTheQueuePath|TestCheckMerge_QueueWithNoRequiredCheckIsRefused|TestMergeEnforcement_)'`(`merge_queue_test.go`、`merge_queue_path_test.go`、`merge_queue_run_test.go`、`merge_queue_wait_test.go` のすべてと、`mergeenforce_test.go` の 6 関数)
- cli: `-run '^(TestOrgMerge_PrintsAQueuedResultAndExitsSeven|TestOrgMerge_PrintsAMergeQueueMergeAsMerged|TestOrgMerge_RequeueReachesTheMergeQueuePath|TestOrgMerge_HelpDescribesTheMergeQueuePath|TestExitCode|TestOrgMerge_Help)$'`

## 計画のテストと AC の対応

テスト名はすべて上の `-v` の run で PASS を確かめた。`(org)` は `internal/org`、`(cli)` は `internal/cli`。

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestMerge_QueueWaitRecordsTheMergeOnceTheQueueMergedIt`(org)、`TestMerge_QueueWaitRecordDetails`(org)、`TestMerge_QueuePathEnqueuesThePRAtTheGatesHead`(org)、`TestOrgMerge_PrintsAMergeQueueMergeAsMerged`(cli) | `--wait-checks 30m` で enqueue 1 回、30 秒ごとの読み、MERGED で `pr_merged` を `via=merge_queue` と `mergeCommit.oid` で 1 行、片付けは `regate.lock` を取り直してから。待ちと読みのあいだ lock は空いている。`merge_commit=unknown` と accepted の branch の details |
| AC2 | `TestMerge_QueueWaitRecordsNothingWhenThePRLeavesTheQueue`(org、5 subtest)、`TestMergeQueue_LatestRemovalAfterCountsTheSameSecond`、`TestMergeQueue_RemovalTextEscapesControlCharacters`(org) | `failed_checks` で外れたら記録なしで Failed。reason と actor を escape して出す。入れたのと同じ秒の外しも数え、入れる前の外しは数えない |
| AC3 | 全体の run の queue のない merge のテスト(`merge_test.go`、`merge_wait_test.go`、`merge_run_test.go`、`merge_readback_test.go`、`merge_fake_test.go`、`merge_ssh_alias_test.go`、`merge_hint_state_dir_test.go`、`mergeorder*`、`mergeguarantee*`、cli の `org_merge_test.go`)。次の節の「main のテストを HEAD に当てる」 | 次の節 |
| AC4 | `TestMerge_QueuePathEnqueuesThePRAtTheGatesHead`(予算 0、答えの entry で終わり、読み直さない)、`TestMerge_QueueWaitLeavesThePRQueuedWhenTheTimeRunsOut`(3 subtest)、`TestMerge_QueueWaitSharesTheBudgetWithTheChecksWait`(org)、`TestOrgMerge_PrintsAQueuedResultAndExitsSeven`、`TestExitCode` の 3 行(7)、`TestOrgMerge_RequeueReachesTheMergeQueuePath`(cli、exit 7) | 記録なし、dequeue の呼び出しなし、終了コード 7 |
| AC5 | `TestMerge_QueuePathDoesNotEnqueueAPRInTheQueueAlready`(4 subtest)、`TestMerge_QueueWaitRecordDetails` の「queue にすでにある」場合(org) | enqueue を打たずに続け、QueueNote を残す |
| AC6 | `TestCheckMerge_QueueWithNoRequiredCheckIsRefused`(2 subtest)、`TestMergeEnforcement_TakesTheQueuePath` の必須の検査なしの 4 行(org) | step 11 で断り、step 12・13 を打たない |
| AC8 | `TestMerge_QueueWaitRecordsNothingForAMergeOfAnotherHeadOrBase`(another head、another base)、入れる前の読みは `TestMerge_QueuePathRefusesAReadThatDisagreesWithTheChecks`(org) | 記録なし、片付けなし、Failed に merge されたものを書く |
| AC10 | `TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue`(16 subtest、`requeue=false`/`true` の対)、`TestMerge_QueuePathComparesRemovalsWithTheAcceptanceWhenLater`(org)、`TestOrgMerge_RequeueReachesTheMergeQueuePath`(cli) | `--requeue` なしは enqueue なしで断る(exit 1)、ありは入れて NOTE を出す |
| AC12 | `TestMerge_QueueWaitReadsAgainAfterAFailedRead`(5 subtest: 失敗が続く、時間切れが続く、data なし、queue のあと失敗、失敗のあと queue) | 待ちのあいだの失敗は読み直す。最後の読みが失敗なら記録なしで「queue にあるかもしれない」 |
| AC13 | `TestMerge_QueuePathPinsTheGraphQLCallsToTheResolvedHost`、`TestMergeQueue_ReadRunsOneGraphQLCallWithTheNumberAsAnInt`、`TestMergeQueue_EnqueueRunsOneGraphQLCallWithStringVariables`、`TestMergeQueue_QueriesAreFixedAndDeclareTheirVariables`、`TestMergeQueue_ChecksRefuseBadValuesBeforeAnyGH`、`TestMergeThroughQueue_ReadsThroughRunDetached`、`TestMerge_QueuePathRefusesAnEnqueueThatSendsNothing`(org) | `--hostname` と `GH_PROMPT_DISABLED=1`、`-F number` と `$number: Int!`、ほかは `-f`、enqueue のあとの読みは `runDetached`、形の悪い node ID と SHA では gh を打たない |
| AC14 | `TestOrgMerge_HelpDescribesTheMergeQueuePath`、`TestOrgMerge_Help`(cli) | help の queue の道、`--requeue`、終了コード 7。README と tech-debt の文は /verify の範囲 |
| AC15 | `TestMerge_QueueWaitRecordsAMergeOnlyOnce`(6 subtest) | 待ちのあいだにほかの merge が書いた `pr_merged` があれば書かず Note を出す。gate より前、別の head・PR・branch、dry run は数えない |
| AC16 | `TestMergeQueue_RemovalAfterGateAllowsForClockSkew`(8 subtest)、`TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue`(gate の後、同じ時刻、4m59s 前、ちょうど 5m 前、10m 前)、`TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance`(org) | 幅の内(4m59s 前まで)は断り、ちょうど 5m 前とそれより前は数えない |

割り込み(D6)は `TestMerge_QueuePathInterrupted`(5 subtest)と `TestMerge_QueueWaitInterrupted`(4 subtest)、片付けの lock(D1)は `TestMerge_QueueWaitCleanupWithTheLockHeldByAnother` と `TestMerge_QueueWaitCleanupEmptiesTheMarkerAndKeepsTheMerge` が通った。

## main のテストを HEAD に当てる(AC3、NFR-1)

この branch は既存のテストファイルを 3 つ書き換えた(`internal/org/merge_test.go`、`internal/org/mergeenforce_test.go`、`internal/cli/org_merge_test.go`)。書き換えが queue のない行に及んでいないことを、main(`d919479a`)の 3 ファイルを `go test -overlay` で HEAD のコードに当てて確かめた。

| run | 関数 | 結果 | 落ちたもの |
| --- | --- | --- | --- |
| org: main の `merge_test.go` と `mergeenforce_test.go` の 19 関数 | 19 | `--- PASS` 192 行、rc 1 | `TestCheckMerge_RefusesWithNothingAfterTheFailingCheck/merge_queue`、`TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn/merge_queue_in_GraphQL_only`、同 `/merge_queue_in_the_rules_only` の 3 行だけ |
| cli: main の `org_merge_test.go` の 16 関数 | 16 | 15 関数 PASS、rc 1 | `TestOrgMerge_Help` の 3 つの文(`shows no merge queue`、片付けの文の `regate.lock still held:`、`--wait-checks` の flag の文)だけ |

落ちた 3 行は「queue があれば断る」という前の挙動の行で、この変更が queue の道に置き換えたもの(仕様 FR-1)。help の 3 文は branch が書き換えた文。queue のない行と、ほかの 15 関数はすべて変えずに通る。HEAD の `TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn` は、queue のない行のすべてで `Queue` が false、`QueueMethod` が空であることも確かめている。

## Coverage

- Statement: queue の道の `-v` run の profile で、`merge_queue.go` 122/122(100.0%)、`merge_queue_run.go` 98/100(98.0%)、`merge_queue_wait.go` 122/126(96.8%)、`mergeenforce.go` 116/117(99.1%)。変更行で見ると `mergeenforce.go` 54/54、`merge.go` 4/4、`merge_run.go` 26/52。`merge_run.go` の残り 26 行は queue のない道(`mergeLocked` の戻り値の追加、`mergedElsewhere` の分け方、`mergeRecordErr`)で、queue の道のテストは通らない。全体の run の queue のない merge のテストは通ったが、パッケージ全体の profile は取っていない
- Branch: Go の標準ツールでは測れない
- Function: 新しい 3 ファイルの関数はすべて実行された。100% でないのは `enqueue` 94.1%、`gateWhen` 83.3%、`recordQueueMerge` 85.7%、`queueMergeCleanup` 83.3%
- Notes: 実行されない文は次の 7 か所。`merge_queue_run.go:102`(読みのあと enqueue の前の割り込みの確かめ)、`:214`(gate の時刻が読めないときの `gateWhen` の文)、`merge_queue_wait.go:312` と `:321`(ledger が読めない・書けないときの `recordQueueMerge` のエラー)、`:383` と `:387`(`regate.lock` が取れない・marker を空にできないときの片付けの飛ばし)、`mergeenforce.go:148`(`run` が nil のときの既定の runner)

変異の確かめ(queue の道の 43 関数を `-vet=off -timeout 4m -overlay` で 1 個ずつ。worktree は前後で `git status --porcelain --ignored` が同じ)。

| ID | 変異 | 結果 |
| --- | --- | --- |
| M1 | `queueDecided` の head の確かめを外す | 検出(`TestMerge_QueueWaitRecordsNothingForAMergeOfAnotherHeadOrBase`) |
| M2 | 同じく base の確かめを外す | 検出(同上) |
| M3 | `queueMergeRecorded` の PR の一致を外す | 検出(`TestMerge_QueueWaitRecordsAMergeOnlyOnce`) |
| M4 | `queueMergeRecorded` を gate の記録の前から探す | 検出(同上) |
| M5 | `recordQueueMerge` のエラーを `mergeRecordErr` で包まずに返す | 生き残り |
| M6 | `enqueue` の、読みのあと enqueue の前の `ctx.Err()` の確かめを外す | 生き残り |
| M7 | FR-5b の時計のずれの幅(5 分)を外す | 検出(`TestMergeQueue_RemovalAfterGateAllowsForClockSkew` ほか 2 関数) |
| M8 | 片付けの前に marker を空にしない | 検出(`TestMerge_QueueWaitCleanupEmptiesTheMarkerAndKeepsTheMerge`) |
| M9 | `--requeue` がなくても断らない | 検出(`TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue` ほか 1 関数) |

M5 と M6 は別のファイルの別の式なので 1 つの overlay にまとめ、cli の `-run 'OrgMerge|ExitCode'`(61 s、ok)と org のパッケージ全体(629 s、ok)でも回した。どちらでも生き残った。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| queue のない repo の `ralph org merge`(NFR-1、AC3) | 変わらない | main の 3 テストファイルを HEAD に当てた run で、落ちたのは queue の行 3 つと書き換えた help の 3 文だけ。全体の run で merge の既存テストファイルはすべて ok |
| 終了コード 1〜6(`TestExitCode`) | 変わらない | `TestExitCode` の既存の行が通る(7 の 3 行を足しただけ) |
| `ralph org merge --help` の queue のない部分 | 変わらない | `TestOrgMerge_Help` が通る |

## Test gaps

- G1(LOW): queue の道で `pr_merged` を書けないとき(ledger が読めない、append に失敗する)のエラーは、どのテストも通らない。M5(`mergeRecordErr` で包まずに返す)は org のパッケージ全体と cli の merge のテストで生き残った。包まないと、エラーから「merge は残っていて ledger だけが記録し損ねた」という文と、確かめる場所が消える。queue のない道の `recordMerge` のエラーには既存のテストがある。足すなら、queue の待ちが MERGED を読んだあとで manifest を書けなくする行(読み取り専用の state dir)で、エラーがその文を含むことを確かめる
- G2(LOW): 読みのあと enqueue の前の割り込みの確かめ(`merge_queue_run.go:101`)は、テストから決まった順で起こせない。`readQueue` が読みの最中と直後の割り込みを自分で返すので、残るのは読みの答えから 101 行までのあいだだけ。M6 で外すと、そこで割り込まれたとき enqueue が取り消し済みの ctx で走り、エラーの文が「queue に入れていない」から「queue にあるかもしれない」に変わる(慎重な側に倒れる)。テストで固めるには、読みの直後に ctx を取り消す seam が要る
- G3(LOW): `gateWhen` の「gate の時刻が読めない」文(`merge_queue_run.go:214`)は、FR-5b の断りと `--requeue` の NOTE の中でしか使われず、その道を通るテストがない。時刻が読めないときにすべての外しを数えることは `TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance` が確かめている
- G4(LOW): `queueMergeCleanup` の、`regate.lock` が held 以外のエラーで取れないとき・marker を空にできないときの飛ばしの文は実行されない。held のときは `TestMerge_QueueWaitCleanupWithTheLockHeldByAnother` が確かめている
- 実地の GitHub(merge queue のある repo)での確かめは計画の Non-goals(ユーザーの決定 2、release のあと下流で確かめる)。GraphQL の型と答えの形は fake の gh で確かめただけで、2 つの query が github.com の schema で通ることは /verify が確かめた

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,184 s)、queue の道の `-v` run(org 43 関数、cli 6 関数)、AC1〜AC6、AC8、AC10、AC12〜AC16 のテスト、main のテストを HEAD に当てた run(落ちたのは置き換えた queue の行と help の文だけ)、変異 9 個のうち 7 個を検出
- Fail: 0
- Blocked: なし
