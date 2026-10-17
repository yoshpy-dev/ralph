# Test report: org-auto-merge

- Date: 2026-10-18(JST 00:15〜00:55。計画の日付は 2026-10-17)
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md(承認の digest は sha256:abefab9fdedd)
- Tester: tester subagent (Claude)。パイプラインの 1 回目(cycle 1)
- Scope: `git diff 15d22b1a...HEAD`(HEAD c0ad3f37)。/verify が見た b82f9dee から HEAD までに増えたのは verify の報告だけで、コードとテストは /verify のときと同じ。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。コード、テスト、文書は変えていない。追加の確かめ(probe と mutant)は `go test -overlay` で scratchpad のファイルを重ね、worktree のファイルには書いていない。環境は macOS(arm64、12 コア)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0
- Evidence: `docs/evidence/test-2026-10-17-org-auto-merge.log`(E1〜E8。wrapper の集計、go1.25.8 の結果と足したテスト 58 本の突き合わせ、待ちのテストの繰り返し、4 ファイルの関数ごとのカバレッジと覆われていないブロック、probe のソースと出力、mutant の diff と結果、flaky の確かめ、テストのあとの worktree。`~` は `$HOME`、`$SCRATCH` は session の scratchpad、`$TMPDIR` は macOS の一時ディレクトリ)。wrapper 自身のログは `docs/evidence/verify-2026-10-17-151510.log`。どちらも gitignore の対象
- テストは本物の gh、herdr、osascript、端末の app、ユーザーの tmux、ネットワークに触れない。org の merge のテストは偽の `MergeOrderExec`、CLI のテストは PATH を絞った偽の gh(`mergeCLIFakeGH`)と手元の bare repo を使う(`internal/cli/org_merge_test.go:17-27` の注とコードで確かめてから走らせた)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は changed。`scripts/ralph-config.sh` が分類できず全体に fallback、言語は golang) | shell 41 本 + Go 10 パッケージ | 全部(rc=0) | 0 | shell の 1 件(下の注) | 1,266 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,283 | 41 本とも OK | 0 | 1 | 上に含む |
| └ `go test ./...`(go1.26.0) | 10 パッケージ(テストのあるもの 8) | 8 | 0 | - | org 329.2 s、cli 269.8 s、config 0.6 s。insights・driver・protocol・scaffold・upgrade は `(cached)` |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -count=1 -timeout 30m -v` | 961 本、subtest を入れて 3,211 | 3,210 | 0 | 1(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、この変更とは関係がない) | 361 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -count=1 -timeout 30m -v` | 758 本、1,516 | 1,516 | 0 | 0 | 356 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/config/ -count=1 -timeout 30m -v` | 46 本、121 | 121 | 0 | 0 | 2 s(rc=0) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/insights/ -count=1` | 1 パッケージ | ok | 0 | - | 2 s(rc=0)。config と org を import するので、wrapper で `(cached)` だった分を打ち直した |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'WaitChecks\|MergeWait' -count=20` | 9 本 × 20 = 180 | 180 | 0 | 0 | 25 s(rc=0)。上の org と cli の go1.25.8 の実行と重ねた(load average 3.3〜4.7) |
| 同じものを `-race -count=5` | 9 本 × 5 = 45 | 45 | 0 | 0 | 22 s(rc=0)。`DATA RACE` はなし。同じく重ねた |
| 上の 2 つを、12 コアに 16 本の busy loop を足して打ち直し | 180 と 45 | 180 と 45 | 0 | 0 | 26 s と 9 s(rc=0)。load average 10.7 から 28.1 |
| カバレッジ: `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'Merge\|Enforce\|WaitChecks\|MergeWait\|PRLookup\|MergeOrder' -coverprofile=$SCRATCH/cov.*/cover.out -count=1` | 1 パッケージ | ok | 0 | - | 122 s(rc=0) |
| probe 2 本(`-overlay`、HEAD) | `TestProbe_ZeroChecksUntilTheTimeIsUp`(org、2 つの場合)、`TestProbe_NoChecksSentenceThroughTheRealExec`(cli、4 つの subtest) | 2 本とも PASS | 0 | 0 | 1 s と 12 s |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の場合を飛ばし、偽の古い git の場合が代わりに見る。この変更とは関係がない
- 足したテスト関数 58 本(`git diff 15d22b1a...HEAD` の `+func Test`)を、go1.25.8 の org・cli・config の `--- PASS` の行と `comm` で突き合わせた。欠けは 0(evidence の E2)
- 待ちの実時間のテスト `TestMerge_WaitChecksWithTheRealClock`(`internal/org/merge_wait_test.go:559`)の 2 つの subtest は、負荷なし・ほかの package と重ねた実行・busy loop を足した実行の、合わせて 50 回の通常の実行と 10 回の `-race` のどれでも通った。「the time is used up」は 0.06〜0.07 s、「ctx done while it sleeps」は 0.05 s で、10 s の上限から遠い。ほかの 8 本は偽の時計(`mwClock`)で動くので負荷の影響を受けない
- 並べた実行の 1 回で、この変更と関係のない `TestRegate_FetchFailures` の 1 行が落ちた。下の「Failure analysis」に書いた

## Edge case とテストの対応

計画の Test plan の edge case と `--wait-checks`(S7)の場合。行番号は HEAD c0ad3f37 のもの。

| 場合 | テスト(file:line) | 結果 |
| --- | --- | --- |
| gh が `no checks reported on the 'feat/a' branch` で終了コード 1(stdout なし)を 0 件と読む | `TestCheckMerge_ReadsTheChecks` の「exit 1 with gh's sentence of no checks for the branch」(`internal/org/merge_test.go:576`)。待ちに入ることは `TestMerge_WaitChecksWaitsOutPendingOrNoChecks` の「none as gh's sentence, then pending, then passed」(`merge_wait_test.go:207`) | 通った |
| 別のブランチの名前の文は gh の失敗 | 「…for another branch」(`feat/a2`、`merge_test.go:578`。`'feat/a'` を前に含む名前でも合わないことを見る)、`TestMerge_WaitChecksDoesNotWaitOutOtherRefusals` の「gh's sentence for another branch」(`merge_wait_test.go:463`) | 通った |
| ほかの終了コード 1 は gh の失敗 | 「exit 1 with another error」(HTTP 502、`merge_test.go:581`)、「gh pr checks fails」(`merge_wait_test.go:462`)。終了コード 0 で同じ文を stdout に出す場合も失敗(`merge_test.go:583`) | 通った |
| 上の 3 つを本物の exec で(err に stderr が入る形) | repo のテストは部品ごと: err の形 `exit status N: <stderr>` は `TestMergeOrder_RealExecRunsInDirWithEnvAndStderr`(`internal/org/mergeorder_test.go:2620`)、読み方は上の表の行。CLI を通すテストはない(下の Test gaps の 3)。probe `TestProbe_NoChecksSentenceThroughTheRealExec` が、偽の gh に文を stderr へ出させて終了コード 1 で 4 つの場合を打ち、`--wait-checks 1ms` のときは「has no checks reported yet」で待った | probe は通った |
| `[]` の 0 件、終了コード 8 で JSON | 「no checks」(`merge_test.go:571`)、「exit 8 with a pending check on stdout」(`:572`)、「a non-zero exit with checks that pass on stdout reads them」 | 通った |
| pass だけ、pass と skipping、skipping だけ、pending を 1 つ | `merge_test.go:564`、`:565`、`:566`、`:569` | 通った |
| pending と fail の混在は待たない | 種類が NotPassed になること: 「pending and fail」(`merge_test.go:597`)。待たないこと: 「pending beside a failed check」(`merge_wait_test.go:459`)。fail、cancel、skipping だけ、知らない bucket も待たない(`:455`〜`:458`) | 通った |
| `--wait-checks` 0 と省略 | 「no --wait-checks, checks pending」(`merge_wait_test.go:471`、待ちの行も眠りもない)、`MergeWaitChecksRefusal(0)` が空(`:528`) | 通った |
| 6h ちょうど | `merge_wait_test.go:528` と `:533-537`(6h で merge する)、CLI の「6h is taken」(`internal/cli/org_merge_test.go:520`) | 通った |
| 6h+1ns、7h | `merge_wait_test.go:505`、`:506`。CLI は `6h1m`(`org_merge_test.go:496`)。どれも git も gh も打たず、`regate.lock` を作らない | 通った |
| 負の値 | `-1ns`、`-1s`(`merge_wait_test.go:503-504`)、CLI の `-1s`(`org_merge_test.go:495`、state dir も作らない) | 通った |
| 待つ間のロックの解放 | 偽の時計の眠りのたびに `regate.lock` を取れることを `mwFixture.check` が全部の待ちのテストで数える(`merge_wait_test.go:163-165`)。待つ間に regate が印を残しても merge の前に空にする、待ちの終わりにロックが持たれていれば断る(`TestMerge_WaitChecksReleasesRegateLock`、`:384`) | 通った |
| 割り込み | 眠る間と読む間(`TestMerge_WaitChecksInterrupted`、`merge_wait_test.go:350`)、本物のタイマーで 1 時間の眠りをすぐ終える(`:578`)。どれも `gh pr merge` を打たず、記録しない | 通った |
| 予算切れ | 最後の眠りを残りに縮めて断りに待った時間を添える、2 回の待ちにまたがる(`TestMerge_WaitChecksBudget`、`merge_wait_test.go:312`)、本物のタイマー(`:560`)、CLI(`org_merge_test.go:527`)。どれも pending の場合だけで、0 件のまま切れる場合のテストはない(下の Test gaps の 1) | 通った |
| 待ちの終わり方: fail が出る、gh が落ちる、待つ間に head が動く、確かめ直しでまた pending | `TestMerge_WaitChecksEndsAndChecksAgain`(`merge_wait_test.go:243`、5 つの行) | 通った |
| merge の直前に既定のブランチかブランチが動く | `TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の「origin's default branch moved」(`merge_test.go:501`)、「the branch moved」(`:503`) ほか 3 行 | 通った |
| PR の head が gate の head と違う、予約がある、queue がある、方式が空 | `merge_test.go:460`、`:470`、`:481`、`:490`(方式は `MERGE_QUEUE`、小文字、field なしも) | 通った |
| 同じ owner/repo で host が違う PR | `merge_test.go:414`(gh pr list の url)、`:468`(gh pr view の url) | 通った |
| upstream の remote がある checkout | `TestCheckMerge_PassesWithEverythingTheMergeNeeds`(`merge_test.go:668`、`fx.exec.upstream = true` で `-R` のない gh を落とす) | 通った |
| 読み直しが落ちる、予約を外す gh が落ちる | `TestMerge_NotMergedRecordsNothing` の「the read-back fails」(`internal/org/merge_run_test.go:327`)、「a request that cannot be removed」(`:320`) | 通った |
| 記録が落ちる | `TestMerge_RecordThatFailsAfterTheMergeIsAnError`(`merge_run_test.go:431`)、`TestOrgMerge_RecordThatFailsPrintsTheLinesAndExitsOne`(`org_merge_test.go:392`) | 通った |
| merge の呼び出しの途中の中断 | `TestMerge_InterruptedOnceGHPRMergeStarted`(`merge_run_test.go:461`)、その前の中断は `:545` | 通った |
| `CheckMergeEnforcement` の表 | `TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn`(`internal/org/mergeenforce_test.go:145`。never・always・pull_requests_only・exempt・null、classic の非 admin と admin と enforce_admins、strict が読めない、queue が GraphQL だけ・rules だけ・読めない) | 通った |
| `pr_lookup` の 5 つの値 | `TestMergeOrder_PRLookupTellsWhatGHGaveForEachBranch`(`mergeorder_test.go:1944`)、`…IsSkippedForEveryBranchWhenNoGHAnswerIsUsed`(`:2016`)、`TestOrgMergeOrder_JSONEntriesTellWhatGHsPRLookupGave`(`internal/cli/org_merge_order_test.go:880`) | 通った |
| AC2 の設定 | `TestLoad_OrgDirectorAutoMergeWrongType`(`internal/config/config_test.go:969`)、`TestDefaultsLockStep`(`defaults_sync_test.go:92`、179 行で 3 面)、`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml`(`org_merge_test.go:562`)、`TestStatusCmd_DirectorLineShowsAutoMerge`(`status_test.go:823`)、`TestOrgDirector_StatusShowsAutoMerge`(`org_director_test.go:569`) | 通った |
| AC9、AC10 | `TestPRMerged_IsNotASeatAndDoesNotRun`(`merge_run_test.go:692`)、`TestMerge_SharesRegateLockWithRegate`(`:754`) | 通った |
| AC11 の skill の文 | `TestDirectorSkill_PinsTheAutoMergeSteps`(`internal/org/director_skill_sync_test.go`) | 通った |

### mutant で確かめたこと

待ちの判定の 4 か所を、scratchpad の写しを `-overlay` で重ねて変えた(diff と結果は evidence の E6)。

| mutant | 変えたところ | 既存のテスト | probe |
| --- | --- | --- | --- |
| A | `internal/org/merge_run.go:187` の `kind.waits()` を `kind == mergeChecksPending` に(0 件のまま時間が切れたとき、待った時間の句を落とす) | 生き残る(org の全体と cli の `OrgMerge\|PrintMergeResult`。org の全体の 1 回は `TestRegate_FetchFailures` で落ちたが、その行は mutant を入れても単独で 3/3 通るので、mutant とは関係がない) | 落とす |
| B | `internal/org/merge_wait.go:156` の poll の `kind.waits()` を `kind == mergeChecksPending` に(待つ間の 0 件の読みで待ちを終える) | 生き残る(同じ範囲) | 落とす |
| C | `internal/org/merge.go:613` の文の比べ方から名指しのブランチを外す | 落ちる(`TestCheckMerge_ReadsTheChecks`、`TestMerge_WaitChecksDoesNotWaitOutOtherRefusals`) | - |
| E | `internal/org/merge_wait.go:96` の `left()` から `w.budget > 0 &&` を外す | 落ちる(`TestMerge_WaitChecksDoesNotWaitOutOtherRefusals`) | - |

B が入ると、CI のない repo で待つ間、0 件を読むたびに「a check did not pass」で待ちを終え、確かめを最初から打ち直す(30 秒ごとに git 2 本と gh 3 本)。AC13 の「30 秒ごとに `gh pr checks` だけを読む」に反するが、既存のテストはこれを見ない。HEAD のコードは正しい(probe が HEAD で通る)。

## Coverage

- Statement: 4 ファイル(`merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go`)で 455/459 = 99.1%(上の `-run` で絞った実行。package 全体の数字 30.3% は絞った実行のもの)
- Branch: Go の標準のツールでは取れない
- Function: 61 関数のうち 80% を下回るものは 0。100% でないのは `mergeInterruptedAfterStart`(`merge_run.go:422`)80.0%、`mergeLocked`(`:160`)91.7%、`CheckMergeEnforcement`(`mergeenforce.go:100`)91.7% の 3 つ
- Notes: 覆われていないブロックは 4 つで、どれも 1 文の異常の道。`merge_run.go:168-170`(`regate.lock` が「持たれている」以外の理由で取れない)、`:177-179`(印を空にする `Truncate` が落ちる)、`:426-427`(`gh pr merge` が始まったあとの中断と、記録などの err が重なる)、`mergeenforce.go:111-113`(`run` が nil のときの既定の `realMergeOrderExec`。動詞からは必ず渡される)。どれも断るか err を返すだけの道で、テストを足さなくても PR を止めない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(`internal/org/regate_test.go:750`) | mutant A の overlay で org の全体を走らせた 1 回で、fetch が 300 ms の上限で止められず、`exit 128`(`fatal: Could not read from remote repository.`)で返った | 負荷による揺れと見る(原因は未確認)。org の全体の実行を 3 本(カバレッジ、mutant A、mutant B)並べていた。この diff は `regate.go` の断りの文を「regate or merge」に変えただけで、fetch の道(`regate_git.go`)には触れていない。HEAD で単独 10/10、16 本の busy loop を足して 5/5、mutant A を入れて単独 3/3 通った。wrapper と go1.25.8 の org の全体でも通った | この PR では直さない。同じ形で落ちたら、単独で打ち直してから regression かを決める |

ほかの FAIL はない(wrapper、go1.25.8 の 4 パッケージ、繰り返し、カバレッジ、probe)。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| merge-order の text の出力が変わらない | 既存のテストが通った(`TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes`、`internal/cli/org_merge_order_test.go:450`、ほか) | E1、E2 |
| doctor の Merge guarantee の判定とキャッシュが変わらない | 既存の `mergeguarantee_test.go` と doctor のテストが通った | E1、E2 |
| `pr_merged` が `RunningOrgs`、`orgsToDisband`、`max_orgs`、`leaderActivityEventCount`、Roster に入らない | `TestPRMerged_IsNotASeatAndDoesNotRun` が通った | E2 |
| regate と merge のロックの排他、merge の間の Spawn | `TestMerge_SharesRegateLockWithRegate`、regate と spawn の既存のテストが通った | E2 |
| 既知の flaky(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` ほか) | どの実行でも落ちなかった | E1、E2 |
| テストのあとの worktree | `git status --porcelain` は空(この報告を書く前)。ignore されたもので増えたのは、wrapper の `docs/evidence/verify-2026-10-17-151510.log`、`.harness/cache/go-build/` の 12 個、この報告の evidence だけ。main のチェックアウトの `GIT_OPTIONAL_LOCKS=0 git status --porcelain` も空 | E8 |

## Test gaps

1. 0 件のまま `--wait-checks` が切れる場合(計画の Risks の「CI のない repo では 0 件のまま時間が切れるまで待つ」)のテストがない。`TestMerge_WaitChecksBudget` と CLI の予算切れのテストは pending の場合だけを打つ。そのため上の mutant A と B が生き残る。`TestProbe_ZeroChecksUntilTheTimeIsUp`(evidence の E5 にソース)と同じ形で、`mwNone` と `mwNoneSentence` で 70 s の予算が切れる行を `TestMerge_WaitChecksBudget` に足せば、両方を落とせる(確かめた)。finding の T1
2. `internal/config/config_test.go:89-91` の注が古い。下の finding の T2
3. gh の 0 件の文を本物の exec と CLI を通して読むテストはない。部品(err の形と読み方)はそれぞれテストがあり、probe の 4 つの場合と /verify の binary の probe が通しで確かめた。足すかは任意で、finding にはしない
4. 覆われていない異常の道 4 つ(上の Coverage)。finding にはしない
5. 本物の GitHub での `gh pr checks` の 0 件の出力、queue のある repo、迂回できる ruleset は、計画の Risks のとおりテストでは見ていない(偽の gh の振る舞いでだけ見る)

## Findings

| ID | Severity | File:line | Summary |
| --- | --- | --- | --- |
| T1 | LOW | `internal/org/merge_wait_test.go:312`(`TestMerge_WaitChecksBudget`) | 0 件のまま予算が切れる場合のテストがなく、`merge_run.go:187` と `merge_wait.go:156` の `kind.waits()` を Pending だけにする mutant が生き残る。B は AC13 の「待つ間は `gh pr checks` だけを読む」を破る。0 件の 2 つの形で予算が切れる行を足す |
| T2 | LOW | `internal/config/config_test.go:89-91` | 注が「`auto_merge` は `scripts/ralph-config.sh` に写しがなく `TestDefaultsLockStep` が見ない」と書くが、S4(e7624d79)で `RALPH_ORG_DIRECTOR_AUTO_MERGE` が足され、`defaults_sync_test.go:179` が 3 面を比べている。注は S2(cddb608c)のまま。アサーションは今も正しい |

情報(finding にしない):

- `--wait-checks 1ms` の probe で、最後の断りは「waited 276ms for the checks (--wait-checks 1ms)」、待ちの終わりの行は「after 31ms」だった。`waited()`(`merge_wait.go:102`)は最初の待ちの始めから断りを作るまでを数え、待ちのあとの確かめ直しの時間も含む。doc の注と D13 のとおりで、本物の gh なら確かめ直しの数秒が予算に足されて見える
- `TestRegate_FetchFailures/the_fetch_runs_past_the_bound` の揺れ(上の Failure analysis)

## Verdict: PASS

- Verdict: pass
- Pass: `./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、`GOTOOLCHAIN=go1.25.8` の org・cli・config・insights、待ちのテストの `-count=20` と `-race -count=5`(負荷を足した打ち直しを含む)、カバレッジの実行が、どれも exit 0 で FAIL は 0。足したテスト関数 58 本が全部通った。計画の Test plan の edge case と、依頼の 8 つの場合には、0 件のまま予算が切れる場合(T1)と本物の exec を CLI まで通す場合(gap 3)を除いて、名前のあるテストがある。4 ファイルの関数で 80% を下回るものはない
- Fail: なし(並べた実行の 1 回の `TestRegate_FetchFailures` は、この変更と関係のない揺れで、単独と負荷をかけた打ち直しで通った)
- Blocked: なし。T1 と T2 は LOW で、PR を止めない
- insight event に足す行(orchestrator が書く): phase `test`、cycle 1、verdict `pass`、findings は critical 0、high 0、medium 0、low 2
