# Test report: org-auto-merge

- Date: 2026-10-18(JST 04:49〜05:25。計画の日付は 2026-10-17)
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md
- Tester: tester subagent (Claude)。パイプラインの 2 回目(cycle 2)。cycle 1 の報告は 6e43569a(PASS、LOW 2 件)。その T1 と T2 は 29afd327 で直っている(下の Regression checks)
- Scope: `git diff 15d22b1a...HEAD`(HEAD 56003422)。cycle 1 から増えたのは S8(9506ec64 classic の保護で誰にでも `enforce_admins` を求める、b2f0b534 読み直しの `headRefOid` が gate の head のときだけ記録する、320b4d75 `--wait-checks` で org の disband も待つ)と S9(e5831d14、7d0d08dd。動きは変えず、偽の gh と git と fixture を `internal/org/merge_fake_test.go` に移し、`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` を足した)。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。コード、テスト、文書は変えていない。probe と mutant は `go test -overlay` で scratchpad のファイルを重ね、worktree のファイルには書いていない。環境は macOS(arm64、12 コア)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0
- Evidence: session の scratchpad(`$SCRATCH/test-c2/`、commit しない)に、各コマンドの出力、mutant の diff と結果(`mut/<id>/diff.txt`、`focus.out`、`full.out`)、probe のソースと出力(`probe/`)、カバレッジの profile、テストの前後の `git status --porcelain --ignored` を置いた。wrapper 自身のログは `docs/evidence/verify-2026-10-17-194935.log`(gitignore の対象)。`$SCRATCH` は session の scratchpad
- テストは本物の gh、herdr、osascript、端末の app、ユーザーの tmux、ネットワークに触れない。org の merge のテストは偽の `MergeOrderExec`(`mxExec`、`internal/org/merge_fake_test.go:60`)、CLI のテストは PATH を絞った偽の gh(`mergeCLIFakeGH`、`internal/cli/org_merge_test.go:37`)と手元の bare repo を使う。gh、herdr、osascript、tmux、agmsg の名前で呼ばれた回数を数える canary を PATH の先頭に置いて org の絞った実行と cli の `OrgMerge|PrintMergeResult` を打ち、呼ばれた回数は 0 だった(canary は `sh -c 'gh …; tmux ls'` で 2 回数えることを先に確かめた)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は changed。`scripts/ralph-config.sh` が分類できず全体に fallback、言語は golang) | shell 41 本 + Go 10 パッケージ | 全部(rc=0) | 0 | shell の 1 件(下の注) | 1,104 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,283 | 41 本とも OK | 0 | 1 | 上に含む |
| └ `go test ./...`(go1.26.0) | 10 パッケージ(テストのあるもの 8) | 8 | 0 | - | org 438.9 s、cli 375.3 s、config 0.5 s。insights・driver・protocol・scaffold・upgrade は `(cached)` |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'CheckMerge\|Merge_\|MergeWait\|WaitChecks\|Enforce\|DirectorSkill\|MergeOrder' -count=1 -timeout 30m -v` | 91 本、subtest を入れて 483 | 483 | 0 | 0 | 10 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -run 'OrgMerge\|OrgRegate\|Status\|OrgDirector' -count=1 -timeout 30m`(`-v` を足して数えた) | 160 本、373 | 373 | 0 | 0 | 215 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test -race ./internal/org/ -run 'Merge\|WaitChecks\|Regate' -count=1 -timeout 30m`(`-v` を足して数えた) | 172 本、783 | 783 | 0 | 0 | 426 s(rc=0)。`DATA RACE` はなし。load average 6.2〜6.6 |
| カバレッジ: `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -coverprofile=$SCRATCH/test-c2/cover.out -count=1 -timeout 30m` | org の全体 | ok | 0 | - | 449 s(rc=0)。package 全体で 94.5% |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/config/ -count=1 -v` | 46 本、121 | 121 | 0 | 0 | 2 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/insights/ -count=1` | 1 パッケージ | ok | 0 | - | 2 s(rc=0)。config と org を import するので、wrapper で `(cached)` だった分を打ち直した。driver・protocol・scaffold・upgrade は `go list -deps` で org・config・cli を import しないことを確かめた |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -run TestPrintMergeResult_EachOutcome -count=1 -v` | 1 本、12 | 12 | 0 | 0 | 1 s(rc=0)。cli の絞った実行の名前に合わないので別に打った |
| 待ちと S8 のテスト(`WaitChecks\|MergeWait\|RecordsOnlyAMergeOfTheGatesHead\|ClassicNeedsEnforceAdmins`)を `-count=20` | 15 本 × 20 = 300 | 300 | 0 | 0 | 26 s(rc=0) |
| 同じものを `-race -count=5` | 15 本 × 5 = 75 | 75 | 0 | 0 | 9 s(rc=0)。`DATA RACE` はなし |
| canary を PATH の先頭に置いて、org の絞った実行(上の `-run` に `PRMerged` を足したもの)と cli の `OrgMerge\|PrintMergeResult` | 2 パッケージ | ok | 0 | - | 10 s と 101 s(rc=0)。canary が呼ばれた回数は 0 |
| mutant 16 個(`-overlay`、下の表) | - | - | - | - | 1 個あたり 12〜14 s。生き残った n1 だけ org の全体を 439 s |
| probe 2 本(`-overlay`、HEAD) | `TestProbeC2_ChecksWaitThenOrgWaitShareTheBudget`、`TestProbeC2_GateOnlyBranchWaitsUntilTheBudget` | 2 本とも PASS | 0 | 0 | 5 s(rc=0) |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の場合を飛ばし、偽の古い git の場合が代わりに見る。cycle 1 と同じで、この変更とは関係がない
- 足したテスト関数 65 本(`git diff 15d22b1a...HEAD` の `+func Test`。cycle 1 は 58 本)を、go1.25.8 の org・cli・config の `-v` の実行の `--- PASS` の行と `comm` で突き合わせた。欠けは 0
- wrapper の Go は go1.26.0 で走った(`GOTOOLCHAIN` を渡していない)。go1.25.8 では org の全体(カバレッジの実行)、config、insights、cli の絞った実行と `TestPrintMergeResult_EachOutcome` を打ったが、cli の全体は go1.25.8 で打っていない
- 既知の揺れ `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(`docs/tech-debt/README.md`)は、wrapper、カバレッジ、`-race` のどの実行でも通った。ほかの既知の揺れ(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval` ほか)も落ちなかった

## Edge case とテストの対応

依頼の 5 つの場合と、計画の Test plan の S8 の行。行番号は HEAD 56003422 のもの。cycle 1 の表の行(検査の待ち、`pr_lookup`、AC2・AC9・AC10・AC11 ほか)は、テストの名前を変えずに移っただけで、上の実行で全部通った(cycle 1 の報告 6e43569a を参照)。

| 場合 | テスト(file:line) | 結果 |
| --- | --- | --- |
| org の待ちのあとに検査の待ちが続き、1 つの予算を分け合う | `TestMerge_WaitChecksOrgAndChecksShareTheBudget`(`internal/org/merge_wait_test.go:538`。70 s の予算で org を 30 s 待ち、検査を残りの 40 s 待つ。断りに「waited 1m10s for org_id "org-a" to be disbanded and for the checks」)。CLI はない | 通った |
| 上の逆の順(検査の待ちのあとに org の待ち) | テストはない。probe `TestProbeC2_ChecksWaitThenOrgWaitShareTheBudget`(検査を待つ間に org-a が同じ機能で起き直す)で、org の待ちが残りの 40 s を受け取り、断りが「for the checks and for org_id "org-a" to be disbanded」の順で 2 つを書くことを確かめた。下の finding の T1 | probe は通った |
| org が走ったまま時間が切れ、断りに `ralph org status --org-id` を添える | `TestMerge_WaitChecksForTheOrgEnds` の「the time is up」(`merge_wait_test.go:415`、断りの全文を比べる)、`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` の「the time is up」(`:516`)、CLI の `TestOrgMerge_WaitChecks` の「the time is up with the org still running」(`internal/cli/org_merge_test.go:577`、断りの先頭と末尾、stderr の 2 行、台帳が変わらないこと、gh は 2 回の確かめの `gh pr list` だけ) | 通った |
| 読み直しの `headRefOid` が空、または field がない | `TestMerge_RecordsOnlyAMergeOfTheGatesHead` の「MERGED with an empty head」(`internal/org/merge_run_test.go:253`)と「MERGED with no headRefOid」(`:255`)。どちらも Failed で Merged は立たず、記録しない | 通った |
| 読み直しの `headRefOid` が gate の head と違う | 同じテストの「gh pr merge refused, another head merged」(`merge_run_test.go:249`、`--match-head-commit` で落ちた場合)、「gh pr merge exited 0, another head merged」(`:251`)、制御文字の escape(`:258`)。中断と重なる場合は `TestMerge_InterruptedOnceGHPRMergeStarted` の「another head is merged」(`:343`)。CLI は `TestOrgMerge_NotMergedExitsOneWithNothingOnStdout` の「another head was merged」(`org_merge_test.go:371`、`MERGED:` の行を出さず exit 1) | 通った |
| classic の保護で `permissions.admin` が true、false、field なし、`permissions` なし | `TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller`(`internal/org/mergeenforce_test.go:311`)。呼び手 4 通り(`:317-320`)× `enforce_admins` の答え 4 通り(有効、無効、読めない、`enabled` なし、`:328-338`)の 16 の subtest。有効のときだけ ok で、gh の endpoint はどれも同じ 6 本(`enforce_admins` を毎回読む) | 通った |
| 待つ間に org が disband し、同じ org_id が同じブランチで同じ機能に起き直す | `TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain`(`merge_wait_test.go:484`)。1 回目の眠りで disband と結びつきを足すと待ちが続き、3 回目の眠りでまた disband すると merge して 1 回記録する(`:486`)。起き直したまま時間が切れると断る(`:516`) | 通った |
| `--wait-checks` で org の disband を待って merge する | `TestMerge_WaitChecksWaitsForTheOrgToBeDisbanded` の 1 回目と 3 回目の眠りの行(`merge_wait_test.go:377`、`:379`) | 通った |
| org を待つ間は git も gh も打たない | 上の org の待ちのテストはどれも `mwFixture.check`(`merge_fake_test.go:627`)で打ったコマンドの列を全部比べる。CLI は `org_merge_test.go:577` が gh の呼び出しを数える。mutant e1 と e2 で確かめた | 通った |
| org がほかの機能を走らせていれば待たない | 最初から: `TestMerge_WaitChecksDoesNotWaitOutOtherRefusals` の「the org runs another feature now」(`merge_wait_test.go:584`)、`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の「org runs another feature now」(`internal/org/merge_test.go:156`)。待つ間にそうなったら待ちを終える: `merge_wait_test.go:381` | 通った |
| `--wait-checks` なしでは今の断りのまま | `TestMerge_WaitChecksForTheOrgEnds` の「no --wait-checks」(`merge_wait_test.go:430`、眠りも行もない) | 通った |
| 待つ間に台帳が読めなくなる、待つ間の中断 | 「the ledger cannot be read」(`merge_wait_test.go:437`)、「interrupted while it sleeps」(`:460`) | 通った |
| 結びつきがなく gate の記録だけのブランチで、その org が別の機能を走らせている | テストは種類だけ(`TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits` の gate だけの行、`merge_test.go:257`)。probe `TestProbeC2_GateOnlyBranchWaitsUntilTheBudget` で、`docs/tech-debt/README.md:298` に書かれたとおり予算が切れるまで待ち、`ralph org status --org-id org-b` を添えて断り、何も記録しないことを確かめた。文書どおりの限界なので finding にしない | probe は通った |

### mutant で確かめたこと

scratchpad の写しを `-overlay` で重ね、org を `CheckMerge|Merge_|MergeWait|WaitChecks|Enforce|MergeOrder|PRMerged|DirectorSkill` で打った。落ちなかったものだけ org の全体で打ち直した(diff と出力は `$SCRATCH/test-c2/mut/<id>/`)。どの mutant のあとも `git status --porcelain` は、テストの前からあった `docs/insights/events/2026-10-17-org-auto-merge.jsonl` の 1 行(orchestrator が書いた verify cycle 2 の行で、この報告とは別)だけだった。

| mutant | 変えたところ | 結果(落ちたテスト) |
| --- | --- | --- |
| a1(依頼の a) | `internal/org/mergeenforce.go:299` の `enforce_admins` 無効の `return false` を `true` に | 落ちる。`TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller` の無効 × 呼び手 4 通り |
| a2(依頼の a) | `mergeenforce.go:295` の `case *body.Enabled:` を `case body.Enabled != nil:` に(無効も有効と読む) | 落ちる。同じ 4 つ |
| b1(依頼の b) | `internal/org/merge_run.go:339` の別の head の case を消す(MERGED ならどの head でも記録する) | 落ちる。`TestMerge_RecordsOnlyAMergeOfTheGatesHead` の 5 行、`TestMerge_InterruptedOnceGHPRMergeStarted` の「another head is merged」 |
| b2(依頼の b) | 同じ行に `back.HeadRefOid != "" &&` を足す(空の head は記録する) | 落ちる。「MERGED with an empty head」「MERGED with no headRefOid」 |
| c(依頼の c) | `internal/org/merge.go:617` の `waits()` から `mergeRefusalOrgRunning` を外す | 落ちる。org の待ちのテスト 5 本(`TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits` ほか)。CLI の `org_merge_test.go:577` も落ちる |
| d(依頼の d) | `internal/org/merge_wait.go:261` の「the org runs another feature now」を `""` に(その場合に待ちを終えない) | 落ちる。`merge_wait_test.go:381` の 1 行だけ(待ちが終わらず眠りが `mwMaxSleeps` を超える) |
| e1(依頼の e) | `merge_wait.go:170` の org の読みの前に `gh pr checks` を打つ | 落ちる。org の待ちのテスト 4 本。CLI の `org_merge_test.go:577` も落ちる(gh の呼び出しに `pr checks 7` が増える) |
| e2 | 同じ所で `git ls-remote origin` を打つ | 落ちる。同じ 4 本 |
| f | `merge.go:503` のほかの機能の断りの種類を `mergeRefusalOrgRunning` に | 落ちる。`merge_test.go:156`、`merge_wait_test.go:381`、`:584` |
| g | `merge_wait.go:136` の `waitedOut` で org の文を添えない | 落ちる。「the time is up」の 2 行(`:415`、`:516`) |
| h | `merge_wait.go:154` で待ちのたびに予算を最初から数え直す | 落ちる。`TestMerge_WaitChecksOrgAndChecksShareTheBudget`、`TestMerge_WaitChecksBudget` の「over two waits」、`TestMerge_WaitChecksEndsAndChecksAgain` の 2 行 |
| i | `merge.go:225` で待つ種類の断りでも `mergeReady` を空にする | 落ちる。12 本 |
| l | `merge_wait.go:255` で台帳に org の `disbanded` が 1 つでもあれば待ちを終える(起き直しを見ない) | 落ちる。`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` の 2 行と `merge_wait_test.go:381` |
| m1(cycle 1 の A) | `merge_run.go:199` の `kind.waits()` を `kind == mergeRefusalChecksPending` に | 落ちる。`TestMerge_WaitChecksBudget` の 0 件の 2 行(`merge_wait_test.go:201-202`)と org の「the time is up」の 2 行 |
| m2(cycle 1 の B) | `merge_wait.go:228` の待ちを続ける条件から `mergeRefusalChecksZero` を外す | 落ちる。`TestMerge_WaitChecksBudget` の 0 件の 2 行 |
| n1 | `merge_wait.go:126` の `waited()` が待った物を始めた順でなく並べ替えた順に書く | 生き残る(org の全体、cli の `TestOrgMerge_WaitChecks` と `TestPrintMergeResult`)。probe `TestProbeC2_ChecksWaitThenOrgWaitShareTheBudget` が落とす。finding の T1 |

依頼の 5 つ(a〜e)はどれも 1 本以上のテストを落とした。cycle 1 で生き残った A と B に当たる m1 と m2 は、29afd327 で足した 0 件の行が落とす。

## Coverage

- Statement: 4 ファイルで 487/491 = 99.2%(org の全体の実行。`merge.go` 209/209、`merge_wait.go` 86/86、`merge_run.go` 102/105、`mergeenforce.go` 90/91)。org の package 全体は 94.5%
- Branch: Go の標準のツールでは取れない
- Function: 依頼の関数はどれも 100%。`merge_wait.go` の 12 関数(`MergeWaitChecksRefusal`、`newMergeWait`、`left`、`waited`、`waitedOut`、`wait`、`poll`、`checksWaitEnd`、`orgWaitEnd`、`line`、`mergeWaitDuration`、`sleepContext`)、`mergeAndRecord`(`merge_run.go:331`)、`otherHeadMerged`(`:390`)、`enforceClassic`(`mergeenforce.go:281`)、`readEnforcementRepository`(`:167`)、`runningRefusal`(`merge.go:496`)、`checkMergeBranch`(`:236`)。4 ファイルの 65 関数で 100% でないのは `mergeLocked`(`merge_run.go:172`)91.7%、`mergeInterruptedAfterStart`(`:458`)80.0%、`CheckMergeEnforcement`(`mergeenforce.go:100`)91.7% の 3 つで、80% を下回るものはない
- Notes: 覆われていないブロックは cycle 1 と同じ 4 つ(行は動いた)で、どれも 1 文の異常の道。`merge_run.go:180-182`(`regate.lock` が「持たれている」以外の理由で取れない)、`:189-191`(印を空にする `Truncate` が落ちる)、`:462-463`(`gh pr merge` が始まったあとの中断と記録の err が重なる)、`mergeenforce.go:111-113`(`Exec` が nil のときの既定の `realMergeOrderExec`)。S8 と S9 が足した道は全部覆われている

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

wrapper、go1.25.8 の各実行、`-race`、繰り返し、カバレッジ、canary、probe のどれにも FAIL はない。probe P1 の最初の 1 回は、期待するコマンドの列に時間切れのあとの最後の確かめ(git 2 本と `gh pr list`)を書き忘れて落ちた。probe の誤りで、列を直して通った。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の T1(0 件のまま予算が切れる場合のテストがない) | 直った。`TestMerge_WaitChecksBudget` に `[]` と gh の文の 2 行(`merge_wait_test.go:201-202`)があり、cycle 1 の mutant A と B に当たる m1 と m2 を落とす | mutant の表 |
| cycle 1 の T2(`internal/config/config_test.go` の古い注) | 直った。`config_test.go:89-94` が `TestDefaultsLockStep` と `RALPH_ORG_DIRECTOR_AUTO_MERGE` の比べ方を書いている | ファイル |
| cross-review cycle 1 の 3 件(classic の保護、別の head の記録、走っている org の断り) | 直しを pin するテストがあり、mutant a1・a2・b1・b2・c・e1 を落とす | 上の 2 つの表 |
| S9b の fixture の移動で既存のテストの中身が変わらない | 移動の前の名前のテストが全部通り、足したテスト関数 65 本の欠けは 0 | Test execution |
| 既知の揺れ | どの実行でも落ちなかった | Test execution の注 |
| テストのあとの worktree | `git status --porcelain` はテストの前と同じ(前からあった insight event の 1 行だけ)。ignore されたもので変わったのは、wrapper のログ `docs/evidence/verify-2026-10-17-194935.log` が増え、wrapper の evidence の保持数の整理で一番古い `docs/evidence/verify-2026-10-17-105543.log` が消えたことと、`.harness/cache/` の 12 個。main のチェックアウトの `GIT_OPTIONAL_LOCKS=0 git status --porcelain` は空 | `$SCRATCH/test-c2/status-*.txt` |

## Test gaps

1. 検査の待ちのあとに org の待ちが来る順のテストがない。`TestMerge_WaitChecksOrgAndChecksShareTheBudget` は org が先の順だけを打つので、待った物を並べ替える mutant n1 が生き残る。finding の T1
2. 結びつきのない gate だけのブランチで、その org が別の機能を走らせている間の待ちのテストはない(probe P2 で確かめた)。`docs/tech-debt/README.md:298` に書かれた既知の限界で、断りの種類は `merge_test.go:257` が pin する。finding にしない
3. mutant d を落とすのは `merge_wait_test.go:381` の 1 行だけ。今は落ちるので finding にしない
4. CLI を通す空の `headRefOid` の行はない。org の層の 2 行(`merge_run_test.go:253`、`:255`)と CLI の別の head の行(`org_merge_test.go:371`)があるので、finding にしない
5. 覆われていない異常の道 4 つ(上の Coverage)。finding にしない
6. 本物の GitHub での `enforce_admins` の endpoint の答え(admin でない呼び手が読めるか)、queue のある repo、本物の `gh pr view` の `headRefOid` は、計画の Risks のとおりテストでは見ていない(偽の gh の振る舞いでだけ見る)

## Findings

| ID | Severity | File:line | Summary |
| --- | --- | --- | --- |
| T1 | LOW | `internal/org/merge_wait_test.go:538`(`TestMerge_WaitChecksOrgAndChecksShareTheBudget`) | 予算を分け合う待ちのテストが org の待ちを先にする順だけを打つ。`mergeWait` の注は待った物を「始めた順」に書くと言うが、`waited()`(`internal/org/merge_wait.go:126`)が並べ替えた順で書く mutant n1 は org の全体と CLI の待ちのテストで生き残る。検査を待つ間に org が起き直す行を足すと落とせる(probe `TestProbeC2_ChecksWaitThenOrgWaitShareTheBudget` が HEAD で通り、n1 で落ちる)。影響は時間切れの断りの文の語順だけで、merge するかどうかは変わらない |

情報(finding にしない):

- probe P2 のとおり、gate だけのブランチでは別の機能を走らせる org でも予算が切れるまで待つ。tech-debt の行のとおりで、何も merge も記録もしない
- wrapper の Go は go1.26.0 で、cli の全体は go1.25.8 で打っていない(Test execution の注)

## Verdict: PASS

- Verdict: pass
- Pass: `./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、go1.25.8 の org の絞った実行と全体(カバレッジ)、cli の絞った実行、config、insights、`-race` の実行、待ちと S8 のテストの `-count=20` と `-race -count=5`、canary の実行が、どれも exit 0 で FAIL は 0。足したテスト関数 65 本が全部通った。依頼の mutant a〜e はどれも 1 本以上のテストを落とし、cycle 1 の T1 は直った。依頼の 5 つの場合のうち 4 つに名前のあるテストがあり、残る逆の順の予算の分け合いは probe で確かめた(T1)。4 ファイルの関数で 80% を下回るものはない
- Fail: なし
- Blocked: なし。T1 は LOW で、PR を止めない
- insight event に足す行(orchestrator が書く): phase `test`、cycle 2、verdict `pass`、findings は critical 0、high 0、medium 0、low 1
