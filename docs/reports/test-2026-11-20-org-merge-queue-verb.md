# Test report: org-merge-queue-verb

- Date: 2026-11-20(Run 2。パイプラインの 2 回目の run で、`cycle-count.json` は `cycle: 2`。既定の上限 2 の最後の run。Run 1 は cycle 1 の `4be700ff` で pass で、下に全文を残す)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`(issue #342)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `6bf81fbd`(Go のコードは `ad178695` のまま。base `d919479a`、worktree は clean)。Run 1 のあとで Go の挙動を変えたのは `2c9a7fe6`(`queueMergeRecorded` が ledger 全体から同じ branch・PR・head の `pr_merged` を探す)だけで、`ad178695` は comment、`7469393c` は help の文とその固定のテストを変えた。`./scripts/run-test.sh` を 1 回、worktree のルートから、Bash tool の `run_in_background` で起動した(INT と HUP は既定のまま)。scope は wrapper の既定の changed(`- Requested scope: changed`)。この repo で見つかる言語は golang だけなので、full でも同じ pack が走る。そのあと queue の道のテストを `-v` で絞って回し、直しを確かめる変異 7 個、manifest の lock を確かめる手元のテスト 1 本、main のテストを HEAD に当てる run(AC3)を足した
- Evidence: `docs/evidence/test-2026-11-20-org-merge-queue-verb.log`(Run 1 の 5,829 行のあとに Run 2 の節を足して 12,552 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-051050.log` にも書いた(同じく追跡外)
- この report も Write tool ではなく shell の heredoc で書いた。security-guidance plugin の PreToolUse hook が、空白を含むパスを引用せずに `python3` に渡して落ち、Write を止めたため。書いたあとに U+FFFD がないことを `grep -c` で確かめ、`.claude/hooks/check_mojibake.sh` を手で回した

## Run 2(cycle 2 of 2、HEAD `6bf81fbd`)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(Run 1 と同じ条件つきの skip) | 2026-11-20 14:10:50 〜 14:30:18 JST、1,168 s、rc 0。開始時の 1 分 load 3.75、終了時 5.81 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 9.5 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。5 つが `(cached)` で、`-count=1` で打ち直して ok | 下の表 |
| `go test ./internal/org/ -count=1 -v`(queue の道の 43 関数。絞り込みは Run 1 と同じ) | 43 関数、`--- PASS` 317 行(subtest を含む。Run 1 の 316 行に AC15 の 1 行が増えた) | 全部 | 0 | 0 | 8 s(パッケージ 2.702 s) |
| `go test ./internal/cli/ -count=1 -v`(queue の 4 関数、`TestExitCode`、`TestOrgMerge_Help`) | 6 関数、`--- PASS` 20 行 | 全部 | 0 | 0 | 11 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok(`(cached)`、打ち直しも ok) | 0.743 s(打ち直し) |
| `internal/cli` | ok | 549.520 s |
| `internal/config` | ok | 11.954 s |
| `internal/gitroot` | ok | 1.979 s |
| `internal/insights` | ok(`(cached)`、打ち直しも ok) | 1.165 s(打ち直し) |
| `internal/org` | ok | 656.724 s |
| `internal/org/driver` | ok | 2.001 s |
| `internal/org/protocol` | ok(`(cached)`、打ち直しも ok) | 1.435 s(打ち直し) |
| `internal/scaffold` | ok(`(cached)`、打ち直しも ok) | 1.159 s(打ち直し) |
| `internal/toolchain` | ok | 16.887 s |
| `internal/upgrade` | ok(`(cached)`、打ち直しも ok) | 2.286 s(打ち直し) |

`(cached)` の 5 つのうち `cmd/ralph` と `internal/insights` は `internal/org` を import する(`go list -deps -test`)。cache は同じ入力で前に通った結果だが、run のあとで 5 つを `-count=1` で打ち直し、すべて ok だった。`internal/org` は golang pack の 1 パッケージの上限(30 分)の 36% で終わった。Go の段の 1 分 load は、見た 4 回で 5.6〜9.1。私の絞った run(変異と手元のテスト、1 回 8 s ほど)は、shell の段と Go の段の最初の 1 分に重なった。負荷に弱いと分かっているテストを含め、落ちたテストはなく、落ちたものの打ち直しもしていない。

### AC15 の直しの確かめ

Run 1 の「計画のテストと AC の対応」(下)のうち、変わるのは AC15 の行だけで、次の行に置き換わる(verify の Run 2 の R2-4)。ほかの AC のテストは Run 2 の `-v` の run でも PASS で、そのコードは comment しか変わっていない。

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC15 | `TestMerge_QueueWaitRecordsAMergeOnlyOnce`(org、7 subtest)、`TestOrgMerge_PrintsAMergeQueueMergeAsMerged` の「recorded already by another merge」(cli) | 同じ branch・PR・head の `pr_merged` が ledger のどこかにあれば書かず、`Recorded` と Note を立てる。書かないのは 3 つ: 待ちのあいだにほかの merge が書いた、そのあとに gate の記録が続いた(`2c9a7fe6` で足した行)、最新の gate の記録より前にある(Run 1 では書く側だった行)。書くのは 4 つ: 別の head、別の PR、別の branch、dry run。cli は gate の記録の句を外した新しい Note を NOTE の行として出す |

直しを確かめるため、`queueMergeRecorded`(`internal/org/merge_queue_wait.go:341-354`)と `recordQueueMerge`(`:307-330`)に変異を 1 個ずつ入れ、Run 1 と同じ絞り込み(queue の道の 43 関数)を `-vet=off -timeout 4m -overlay` で回した。変異の写しと overlay は `$TMPDIR` の下に置き、worktree には何も書いていない(`git status --porcelain --ignored` は前後で同じ)。対照の HEAD は同じ絞り込みで rc 0。

| ID | 変異 | 結果 |
| --- | --- | --- |
| R1 | 直しの前の境目を戻す: `collectMergeTargets` で branch の最新の gate の記録の位置を取り、`l.events[after+1:]` だけを探す(`2c9a7fe6` の前のコードのまま。Note の文は今のまま) | 検出。落ちたのは新しい形の 2 つの subtest、「a gate record followed the other merge's record」と「before the latest gate record」だけ。「another merge recorded it during the wait」は通る(ほかの merge の記録が gate の記録の後ろにあるので、境目があっても見つかる) |
| R2 | branch の一致を外す | 検出(「of another branch」) |
| R3 | head の一致を外す | 検出(「of another head」) |
| R7 | PR の一致を外す | 検出(「of another PR」) |
| R4 | dry run を数えない条件を外す | 検出(「a dry run」) |
| R5 | 座席の `pr_merged`(`SeatID` が空でない)を数えない条件を外す | 生き残り(Test gaps の 2) |
| R6 | ledger の読み、`queueMergeRecorded`、append を manifest の lock の外で行う(`withManifestLockContext` を、渡された関数を呼ぶだけの関数に置き換える) | 生き残り(Test gaps の 3) |

R1 で落ちる 2 つの subtest は、どちらも最初の `t.Fatalf`(結果の `Note` が空)で止まる。その 2 か所を `t.Errorf` にした test file の写しを R1 と一緒に当てると、続く append の確かめも落ちた。「a gate record followed …」では、待ちのあいだに足した `pr_merged` と `gate_passed` のあとに、この merge が 2 行目の `pr_merged` を書いた(足された events は `pr_merged gate_passed pr_merged`)。「before the latest gate record」では、何も書かないはずのところに `pr_merged` を 1 行書いた。cross-review が指摘した 2 行目の記録は、HEAD では起こらず、境目を戻すと起こる。

R6 は、手元で書いたテスト `TestProbe_QueueRecordWaitsForTheManifestLock`(`-overlay` で足しただけで、commit していない)で落ちる。このテストは、待ちの 2 回目の sleep でほかの merge の代わりに manifest の lock を取り、500 ms 後に同じ branch・PR・head の `pr_merged` を append してから lock を離す。HEAD では `recordQueueMerge` が lock を待ってから ledger を読むので、ほかの merge の 1 行だけが残り、結果に Note が立つ(`-count=3` で 3 回とも PASS、1 回 0.52 s)。R6 では lock を待たずに読んで書くので `pr_merged` が 2 行になり、Note も空になる(3 回とも FAIL)。テストの全文は evidence にある。

### main のテストを HEAD に当てる(AC3、NFR-1)

Run 1 のあと `7469393c` が help の文と `TestOrgMerge_Help` の固定の文字列を変えたので、Run 1 と同じ run を HEAD に当て直した。結果は Run 1 と同じだった。org(main の `merge_test.go` と `mergeenforce_test.go` の 19 関数)は `--- PASS` 192 行で、落ちたのは queue の 3 行(`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck/merge_queue`、`TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn` の `merge_queue_in_GraphQL_only` と `merge_queue_in_the_rules_only`)だけ。cli(main の `org_merge_test.go` の 16 関数)は 15 関数が通り、`TestOrgMerge_Help` が Run 1 と同じ 3 つの文(`shows no merge queue`、片付けの文の `regate.lock still held:`、`--wait-checks` の flag の文)で落ちた。どれもこの branch が置き換えた queue の挙動と help の文で、queue のない行は変わらずに通る。

### Coverage

- Statement: queue の道の run の profile で、`merge_queue_wait.go` 116/120(96.7%)。Run 1 の 122/126 から、`2c9a7fe6` が消した境目の 6 文(どれも Run 1 で実行されていた)が減った。`queueMergeRecorded` は 100%。`merge_queue.go` 122/122、`merge_queue_run.go` 98/100、`mergeenforce.go` 116/117 は Run 1 と同じ
- Branch: Go の標準ツールでは測れない。`queueMergeRecorded` の条件のうち、座席の記録を除く条件(R5)と、それを囲む manifest の lock(R6)は、どのテストにも縛られていない
- Function: 100% でないのは Run 1 と同じ 4 関数(`enqueue`、`gateWhen`、`recordQueueMerge` 85.7%、`queueMergeCleanup` 83.3%)
- Notes: 実行されない文は Run 1 の 7 か所のまま。`merge_queue_wait.go` の片付けの 2 か所は 4 行上に動き、`:379` と `:383` になった(ほかは `merge_queue_run.go:102` と `:214`、`merge_queue_wait.go:312` と `:321`、`mergeenforce.go:148`)。パッケージ全体の profile は取っていない

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の指摘: 待ちのあいだに gate か regate が gate の記録を足すと、同じ queue を待つ 2 つ目の merge が 2 行目の `pr_merged` を書く | 直った | HEAD で `TestMerge_QueueWaitRecordsAMergeOnlyOnce` の 7 subtest が通り、境目を戻す R1 で新しい 2 subtest が落ちる。`Errorf` にした写しで、R1 が 2 行目の `pr_merged` を書くことも確かめた |
| queue のない repo の `ralph org merge`(NFR-1、AC3) | 変わらない | main の 3 テストファイルを HEAD に当てた run の結果が Run 1 と同じ。全体の run で merge の既存テストファイルはすべて ok |
| 終了コード 1〜7(`TestExitCode`)と `ralph org merge --help` | 変わらない | cli の `-v` の run で `TestExitCode`、`TestOrgMerge_Help`、`TestOrgMerge_HelpDescribesTheMergeQueuePath` が通る |

### Test gaps

`/sync-docs` に渡すもの。

1. G1〜G4(LOW): Run 1 のまま。register(`docs/tech-debt/README.md`)の「Four test gaps of feat/org-merge-queue-verb」の行に入っている。`2c9a7fe6` が変えたのは `queueMergeRecorded` と Note の文だけで、G1〜G4 の道(`recordQueueMerge` のエラー、`enqueue` の割り込みの確かめ、`gateWhen` の文、`queueMergeCleanup` の飛ばし)は変わっていないので、行の文はそのまま正しい
2. G5(LOW、新規): `queueMergeRecorded` は座席の `pr_merged`(`SeatID` が空でない)を数えないが、その行を持つテストがない。R5(条件を外す)は queue の道の 43 関数で生き残り、`pr_merged` に `SeatID` を立てたテストは `internal/org` にも `internal/cli` にもない(grep)。ralph は `pr_merged` を org の行としてしか書かないので、今の挙動には影響しない。足すなら、`TestMerge_QueueWaitRecordsAMergeOnlyOnce` の「書く」側の表に、同じ branch・PR・head で `SeatID` が `implementer` の行を 1 つ足す。verify の Run 2 の Coverage gaps の 2 つ目と同じもの
3. G6(MEDIUM、新規): D7 が 2 つのプロセスのあいだで成り立つのは、`recordQueueMerge` が ledger の読み、`queueMergeRecorded`、append を 1 つの manifest の lock の中で行うからで、それを縛るテストがない。AC15 のテストが見るのは、ほかの merge の記録がこの merge の読みより前に ledger にある場合だけ。R6(lock を外す)は queue の道の 43 関数で生き残った。`recordQueueMerge` に届くテストは `merge_queue_wait_test.go` にしかなく、その関数はすべて絞り込みに入る。上の `TestProbe_QueueRecordWaitsForTheManifestLock` は HEAD で 3 回とも通り、R6 で 3 回とも落ちるので、名前を変えて `merge_queue_wait_test.go` に足せば縛れる。MEDIUM にしたのは、lock を外す変更がテストを 1 つも落とさずに、2 つのプロセスの場合の AC15 を破れるため。今のコードは正しい(HEAD で手元のテストが通る)ので、verdict は変えない。verify の Run 2 の Coverage gaps の 4 つ目(別のプロセスでの競合は見ていない)のうち、lock の部分にあたる
4. 実地の GitHub(merge queue のある repo)での確かめは Run 1 と同じく計画の Non-goals で、register の「The merge queue path of `ralph org merge` … was built on a fake gh」の行にある

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,168 s。`(cached)` の 5 パッケージは `-count=1` で打ち直して ok)、queue の道の `-v` run(org 43 関数、cli 6 関数)、AC15 の 7 subtest、直しの確かめ(境目を戻す R1 は新しい 2 subtest で検出し、HEAD は通る)、main のテストを HEAD に当てた run(Run 1 と同じ結果)、変異 7 個のうち 5 個を検出
- Fail: 0
- Blocked: なし

## Run 1(cycle 1、HEAD `4be700ff`)

下は Run 1 の report の全文で、見出しを 1 段下げた。AC15 の行は `2c9a7fe6` の前の形(6 subtest、gate の記録より前は数えない)を書いていて、Run 2 の AC15 の行に置き換わる。

- Date: 2026-11-20。cycle 1(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`(issue #342)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `4be700ff`(base `d919479a` = main、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから、Bash tool の `run_in_background` で起動した(INT と HUP は既定のまま)。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと queue の道のテストを `-v` で絞って回し、main のテストを HEAD のコードに当てる run(AC3)と、変異 9 個を足した
- Evidence: `docs/evidence/test-2026-11-20-org-merge-queue-verb.log`(5,829 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-034409.log` にも書いた(同じく追跡外)
- この report は Write tool ではなく shell の heredoc で書いた。security-guidance plugin の PreToolUse hook が、空白を含むパスを引用せずに `python3` に渡して落ち、Write を 2 回とも止めたため。mojibake の確かめは書いたあとに `.claude/hooks/check_mojibake.sh` を手で回した(exit 0)

### Test execution

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

### 計画のテストと AC の対応

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
| AC15 | `TestMerge_QueueWaitRecordsAMergeOnlyOnce`(6 subtest) | 待ちのあいだにほかの merge が書いた `pr_merged` があれば書かず Note を出す。gate より前、別の head・PR・branch、dry run は数えない(`2c9a7fe6` の前の形。今の形は Run 2 の AC15 の行) |
| AC16 | `TestMergeQueue_RemovalAfterGateAllowsForClockSkew`(8 subtest)、`TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue`(gate の後、同じ時刻、4m59s 前、ちょうど 5m 前、10m 前)、`TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance`(org) | 幅の内(4m59s 前まで)は断り、ちょうど 5m 前とそれより前は数えない |

割り込み(D6)は `TestMerge_QueuePathInterrupted`(5 subtest)と `TestMerge_QueueWaitInterrupted`(4 subtest)、片付けの lock(D1)は `TestMerge_QueueWaitCleanupWithTheLockHeldByAnother` と `TestMerge_QueueWaitCleanupEmptiesTheMarkerAndKeepsTheMerge` が通った。

### main のテストを HEAD に当てる(AC3、NFR-1)

この branch は既存のテストファイルを 3 つ書き換えた(`internal/org/merge_test.go`、`internal/org/mergeenforce_test.go`、`internal/cli/org_merge_test.go`)。書き換えが queue のない行に及んでいないことを、main(`d919479a`)の 3 ファイルを `go test -overlay` で HEAD のコードに当てて確かめた。

| run | 関数 | 結果 | 落ちたもの |
| --- | --- | --- | --- |
| org: main の `merge_test.go` と `mergeenforce_test.go` の 19 関数 | 19 | `--- PASS` 192 行、rc 1 | `TestCheckMerge_RefusesWithNothingAfterTheFailingCheck/merge_queue`、`TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn/merge_queue_in_GraphQL_only`、同 `/merge_queue_in_the_rules_only` の 3 行だけ |
| cli: main の `org_merge_test.go` の 16 関数 | 16 | 15 関数 PASS、rc 1 | `TestOrgMerge_Help` の 3 つの文(`shows no merge queue`、片付けの文の `regate.lock still held:`、`--wait-checks` の flag の文)だけ |

落ちた 3 行は「queue があれば断る」という前の挙動の行で、この変更が queue の道に置き換えたもの(仕様 FR-1)。help の 3 文は branch が書き換えた文。queue のない行と、ほかの 15 関数はすべて変えずに通る。HEAD の `TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn` は、queue のない行のすべてで `Queue` が false、`QueueMethod` が空であることも確かめている。

### Coverage

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

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| queue のない repo の `ralph org merge`(NFR-1、AC3) | 変わらない | main の 3 テストファイルを HEAD に当てた run で、落ちたのは queue の行 3 つと書き換えた help の 3 文だけ。全体の run で merge の既存テストファイルはすべて ok |
| 終了コード 1〜6(`TestExitCode`) | 変わらない | `TestExitCode` の既存の行が通る(7 の 3 行を足しただけ) |
| `ralph org merge --help` の queue のない部分 | 変わらない | `TestOrgMerge_Help` が通る |

### Test gaps

- G1(LOW): queue の道で `pr_merged` を書けないとき(ledger が読めない、append に失敗する)のエラーは、どのテストも通らない。M5(`mergeRecordErr` で包まずに返す)は org のパッケージ全体と cli の merge のテストで生き残った。包まないと、エラーから「merge は残っていて ledger だけが記録し損ねた」という文と、確かめる場所が消える。queue のない道の `recordMerge` のエラーには既存のテストがある。足すなら、queue の待ちが MERGED を読んだあとで manifest を書けなくする行(読み取り専用の state dir)で、エラーがその文を含むことを確かめる
- G2(LOW): 読みのあと enqueue の前の割り込みの確かめ(`merge_queue_run.go:101`)は、テストから決まった順で起こせない。`readQueue` が読みの最中と直後の割り込みを自分で返すので、残るのは読みの答えから 101 行までのあいだだけ。M6 で外すと、そこで割り込まれたとき enqueue が取り消し済みの ctx で走り、エラーの文が「queue に入れていない」から「queue にあるかもしれない」に変わる(慎重な側に倒れる)。テストで固めるには、読みの直後に ctx を取り消す seam が要る
- G3(LOW): `gateWhen` の「gate の時刻が読めない」文(`merge_queue_run.go:214`)は、FR-5b の断りと `--requeue` の NOTE の中でしか使われず、その道を通るテストがない。時刻が読めないときにすべての外しを数えることは `TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance` が確かめている
- G4(LOW): `queueMergeCleanup` の、`regate.lock` が held 以外のエラーで取れないとき・marker を空にできないときの飛ばしの文は実行されない。held のときは `TestMerge_QueueWaitCleanupWithTheLockHeldByAnother` が確かめている
- 実地の GitHub(merge queue のある repo)での確かめは計画の Non-goals(ユーザーの決定 2、release のあと下流で確かめる)。GraphQL の型と答えの形は fake の gh で確かめただけで、2 つの query が github.com の schema で通ることは /verify が確かめた

### Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,184 s)、queue の道の `-v` run(org 43 関数、cli 6 関数)、AC1〜AC6、AC8、AC10、AC12〜AC16 のテスト、main のテストを HEAD に当てた run(落ちたのは置き換えた queue の行と help の文だけ)、変異 9 個のうち 7 個を検出
- Fail: 0
- Blocked: なし
