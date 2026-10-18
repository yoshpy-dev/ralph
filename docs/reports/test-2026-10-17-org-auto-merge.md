# Test report: org-auto-merge

- Date: 2026-10-18(JST 08:22〜09:15。計画の日付は 2026-10-17)
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md
- Tester: tester subagent (Claude)。パイプラインの 3 回目(cycle 3)。上限を `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げた回し直しで、`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま。cycle 2 の報告はこのファイルの前の版で、git の c7dfe662 に残っている(HEAD 56003422 を見て PASS、LOW 1 件)。その T1 は 425a8543 で直っている(下の Regression checks)
- Scope: `git diff 15d22b1a...HEAD`(HEAD 2f975743)。テストは全体を走らせ、場合と mutant は cycle 2 からの差分 `git diff 56003422..HEAD` に絞った。増えたのは、425a8543(`TestMerge_WaitChecksChecksThenOrgShareTheBudget`)、56efae4a(文書と help。`TestOrgMerge_Help` の断片が変わった)、S10 の 590214fb(`gh pr merge` のあとの読み直しで `baseRefName` も読み、gate の head で origin の既定のブランチへ MERGED のときだけ `pr_merged` を記録する。`otherBaseMerged`。テストは新しい `internal/org/merge_readback_test.go`)、S11a の f6e5208e(動きは変えない。`mergeOutcomeFields`、`mergedElsewhere`、base の Failed に merge-order の注、`mergeWait.wait` の待つ kind を名指しの case にして `default:` はエラー、`TestMergeWait_EveryKindThatWaitsHasACase`、`mrNext` と `mrMergedAfter` を `merge_fake_test.go` へ、`addInSleeps`)、S11b の dd1d4867 と 89829dbf(文書とコメントだけ)、verify cycle 3 の af36aaac と計画の記録。振る舞いのテストだけを走らせた。静的解析は /verify の受け持ちなので走らせていない。この報告のほかにコード、テスト、文書は変えていない。probe と mutant は `go test -overlay` で scratchpad のファイルを重ね、worktree のファイルには書いていない。環境は macOS(arm64、12 コア)、go1.25.8(`GOTOOLCHAIN=go1.25.8`、wrapper にも渡した)、git 2.49.0
- Evidence: session の scratchpad(`$SCRATCH/test-c3/`、commit しない)に、各コマンドの出力、mutant の diff と結果(`mut/<id>/diff.txt`、`org.out`、`cli.out`、`orgall.out`)、mutant を作るスクリプト(`mkmut.pl`。置き換える文字列がファイルに 1 回だけ現れることを確かめてから置き換える)、probe のソースと出力(`probe/`)、カバレッジの profile、テストの前後の `git status --porcelain --ignored` を置いた。wrapper 自身のログは `docs/evidence/verify-2026-10-17-232230.log`(gitignore の対象)。`$SCRATCH` は session の scratchpad
- テストは本物の gh、herdr、osascript、端末の app、ユーザーの tmux、ネットワークに触れない。org の merge のテストは偽の `MergeOrderExec`(`mxExec` と `mrExec`、`internal/org/merge_fake_test.go`)、CLI のテストは偽の gh(`mergeCLIFakeGH`、`internal/cli/org_merge_test.go:37`)と手元の bare repo を使い、PATH を fixture の bin だけに置き換える(`internal/cli/org_merge_order_test.go:159`)。gh、herdr、osascript、tmux、agmsg の名前で呼ばれた回数を数える canary を PATH の先頭に置いて org の絞った実行を打ち、呼ばれた回数は 0 だった(canary は `sh -c 'gh …; tmux ls'` で 2 回数えることを先に確かめた)。CLI のテストは PATH を置き換えるので canary には届かない。そのため cycle 2 と違って CLI では canary を回していない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOTOOLCHAIN=go1.25.8 ./scripts/run-test.sh`(scope は changed。`scripts/ralph-config.sh` が分類できず全体に fallback、言語は golang) | shell 41 本 + Go 10 パッケージ | 全部(rc=0) | 0 | shell の 1 件(下の注) | 1,022 s |
| └ shell の suite(local verifier、`tests/*.sh`) | 41 本、PASS の行 4,283(うち `PASS: 29` の形の集計が 25 行、残る 4,258 行が個々の確かめ) | 41 本とも OK | 0 | 1 | 上に含む。suite ごとの PASS・FAIL・SKIP の数は、cycle 2 のログ `docs/evidence/verify-2026-10-17-194935.log` を同じ awk で数えたものと同じ |
| └ `go test ./...`(go1.25.8) | 10 パッケージ(テストのあるもの 8) | 8 | 0 | - | org 446.5 s、cli 338.5 s、config 1.1 s、insights 2.0 s、driver 2.6 s、protocol 1.7 s、scaffold 3.0 s、upgrade 3.2 s。toolchain が cycle 2 の wrapper(go1.26.0)と違うので `(cached)` はない |
| カバレッジ: `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -coverprofile=$SCRATCH/test-c3/cover.out -count=1 -timeout 30m -v` | 972 本、subtest を入れて 3,257 | 3,257 | 0 | 1 | 502 s(rc=0)。package 全体で 94.5%。SKIP は Darwin 以外の場合を見る `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -count=1 -v -timeout 30m` | 758 本、1,519 | 1,519 | 0 | 0 | 530 s(rc=0)。cycle 2 は cli の全体を go1.26.0 でだけ打った。今回は wrapper と合わせて go1.25.8 で 2 回 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/config/ -count=1 -v` | 46 本、121 | 121 | 0 | 0 | 1 s(rc=0) |
| 足した・変えた merge のテスト 11 本(下の注)を `-count=20` | 11 本 × 20 = 220 | 220 | 0 | 0 | 11 s(rc=0) |
| 同じものを `-race -count=5` | 11 本 × 5 = 55 | 55 | 0 | 0 | 14 s(rc=0)。`DATA RACE` はなし |
| `GOTOOLCHAIN=go1.25.8 go test -race ./internal/org/ -run 'Merge\|WaitChecks' -count=1 -v` | 145 本、684 | 684 | 0 | 0 | 283 s(rc=0)。`DATA RACE` はなし |
| canary を PATH の先頭に置いて org の絞った実行(`-run 'CheckMerge\|Merge_\|MergeWait\|WaitChecks\|Enforce\|DirectorSkill\|MergeOrder\|PRMerged'`) | 1 パッケージ | ok | 0 | - | 9 s(rc=0)。canary が呼ばれた回数は 0 |
| mutant 23 個(`-overlay`、下の表) | - | - | - | - | org の絞った実行は 1 個あたり 13〜15 s(W2 だけ 4 分の timeout まで)、CLI の絞った実行(`OrgMerge\|PrintMergeResult`)は 1 個あたり 90〜130 s。生き残った O1 と B14 は org の全体(575 s と 464 s、どちらも rc=0)と CLI でも打った |
| probe 5 本(`-overlay`、HEAD) | org 4 本(subtest 9)、CLI 1 本(subtest 2) | 全部 PASS | 0 | 0 | 1 s と 9 s(rc=0) |

- shell の SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git が 2.49.0 なので本物の古い git の場合を飛ばし、偽の古い git の場合が代わりに見る。cycle 1、cycle 2 と同じで、この変更とは関係がない
- 足したテスト関数 69 本(`git diff 15d22b1a...HEAD` の `+func Test`。cycle 2 は 65 本。増えたのは `TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch`、`TestMerge_InterruptedThenMergedIntoAnotherBase`、`TestMerge_WaitChecksChecksThenOrgShareTheBudget`、`TestMergeWait_EveryKindThatWaitsHasACase` の 4 本)を、go1.25.8 の org・cli・config の `-v` の実行の `--- PASS` の行と `comm` で突き合わせた。欠けは 0
- 繰り返しの 11 本は、上の 4 本と、S10 と S11a が期待する文や fixture を変えた 7 本(`TestMerge_RecordsOnlyAMergeOfTheGatesHead`、`TestMerge_InterruptedOnceGHPRMergeStarted`、`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain`、`TestMerge_WaitChecksOrgAndChecksShareTheBudget`、`TestMerge_MergesWithTheMethodAndRecordsPRMerged`、`TestMerge_NotMergedRecordsNothing`、`TestMerge_ReadsThePRBackAfterAFailedMerge`)
- org の全体は同時に 2 つまでにした(wrapper とカバレッジの実行が重なり、O1 と B14 の全体の実行はそれぞれ 1 つだけで、cli の実行と重なった)
- 既知の揺れ `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(`docs/tech-debt/README.md`)は、wrapper、カバレッジ、O1 と B14 の全体の実行のどれでも通った。ほかの既知の揺れ(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)も落ちなかった

## Edge case とテストの対応

依頼の場合と、計画の Test plan の S10 の行(`docs/plans/active/2026-10-17-org-auto-merge.md:148`)。行番号は HEAD 2f975743 のもの。cycle 1 と cycle 2 の表の行(検査の待ち、org の待ち、`pr_lookup`、classic の保護、別の head ほか)は、テストの名前を変えずに残り、上の実行で全部通った(cycle 2 の報告 c7dfe662 を参照)。

| 場合 | テスト(file:line) | 結果 |
| --- | --- | --- |
| 読み直しが gate の head で MERGED、`baseRefName` が `release` | `TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch` の「gh pr merge exited 0, merged into another base」と「gh pr merge failed, merged into another base」(`internal/org/merge_readback_test.go:41`、`:43`)。Failed の全文を比べ、Merged は立たず、記録しない。gh の失敗の末尾も付く | 通った |
| 同じく `baseRefName` が `""` | 「MERGED with an empty base」(`merge_readback_test.go:45`)。「with no base, so whether it merged into main is unknown」と「may have got nothing」の文 | 通った |
| 同じく `baseRefName` の field がない | 「MERGED with no baseRefName」(`merge_readback_test.go:47`) | 通った |
| 同じく `baseRefName` が JSON の null、`refs/heads/main`、`Main`、文字列でない | テストはない。probe `TestProbeC3_ReadBackBaseEdges` で、null は field がない場合と同じ文、`refs/heads/main` と `Main` は別の base の文(閉じる側に倒れる)、文字列でないものは読み直しの JSON が読めない場合の文(`unknown`)になり、どれも記録しないことを確かめた | probe は通った |
| head と base の両方が違う(head の文が勝つ) | 「another head into another base names the head」「another head with no base names the head」(`merge_readback_test.go:52`、`:54`)。mutant B3(case の順を入れ替える)を落とす | 通った |
| `gh pr merge` が始まったあとの中断と、別の base への MERGED の読み直し | `TestMerge_InterruptedThenMergedIntoAnotherBase`(`merge_readback_test.go:92`)。中断の誤りのあとに base の Failed が続き、記録せず、`regate.lock` を放す。base がない場合と、別の head かつ別の base の場合は probe `TestProbeC3_InterruptedThenOtherBaseEdges` で確かめた | 通った |
| 読み直しの `gh pr view` が求める field がちょうど `state,headRefOid,baseRefName,mergeCommit,autoMergeRequest` | `TestMerge_MergesWithTheMethodAndRecordsPRMerged` が argv を文字どおり比べる(`internal/org/merge_run_test.go:57`)。CLI の偽の gh は同じ文字列のときだけ読み直しに答える(`internal/cli/org_merge_test.go:41`)。偽の `mrExec` は本番の定数 `mergeReadBackFields` で照合するので、org の側で文字列を pin するのは `:57` だけ。mutant B4(baseRefName を外す)と B5(同じ field の並べ替え)を落とす | 通った |
| 確かめるコマンドとして示す `--json` の field にも `baseRefName` | `TestMerge_NotMergedRecordsNothing` の `unknown`(`merge_run_test.go:104`)、`mrCheckIt`(`internal/org/merge_fake_test.go:333`)を使う行。mutant B6 を落とす | 通った |
| CLI は別の base で `MERGED:` の行を出さず exit 1 | `TestOrgMerge_NotMergedExitsOneWithNothingOnStdout` の「the PR merged into another base」(`internal/cli/org_merge_test.go:381`。err があり、stdout が空、台帳が変わらず、`gh pr merge` は 1 回)。`main` は err を exit 1 にする(`cmd/ralph/main.go:31`)。base がない場合と空の場合は probe `TestProbeC3_CLIMergedWithNoBase` で確かめた。mutant B1 と B10(別の base で Merged を立てる)を CLI でも落とす | 通った |
| `mergeRefusalKind.waits()` が true の kind に、どれも `mergeWait.wait` の case がある | `TestMergeWait_EveryKindThatWaitsHasACase`(`internal/org/merge_wait_test.go:693`)。kind を `mergeRefusalChecksUnread + 8` の手前まで回し、予算 0 の `wait` が誤りなく 2 行を書くことを見る。依頼の mutant W1(`waits()` に `\|\| k == mergeRefusalChecksUnread`)で落ちる | 通った |
| 検査の待ちのあとに org の待ちが続き、1 つの予算を分け合う(cycle 2 の T1) | `TestMerge_WaitChecksChecksThenOrgShareTheBudget`(`merge_wait_test.go:548`)。断りが「for the checks and for org_id "org-a" to be disbanded」の順で 2 つを書く。cycle 2 で生き残った n1 に当たる N1 を落とす | 通った |
| 既定のブランチが `main` でない repo で、既定のブランチへの merge を記録し、`main` への merge を記録しない | テストはない(merge のテストの fixture は既定のブランチがいつも `main`)。probe `TestProbeC3_DefaultBranchOtherThanMain`(既定のブランチ `trunk`)で確かめた。比べる相手を `"main"` と書き込む mutant B14 が生き残る。finding の T1 | probe は通った |
| 別の base に merge された PR を、merge-order が `merged` と並べる(base の Failed の注が言うこと) | テストはない(merge-order のテストの MERGED の PR は base がいつも `main`、`internal/org/mergeorder_test.go:1237` ほか)。probe `TestProbeC3_MergeOrderListsAMergeIntoAnotherBaseAsMerged` で、今は `merged` と並ぶことを確かめた。MERGED でも base が既定のブランチでなければ `merged` にしない mutant O1 が生き残る。finding の T2 | probe は通った |

### mutant で確かめたこと

scratchpad の写しを `-overlay` で重ね、org を `CheckMerge|Merge_|MergeWait|WaitChecks|Enforce|MergeOrder|PRMerged|DirectorSkill` で打った。CLI に出る違いがあるものは cli も `OrgMerge|PrintMergeResult` で打った。生き残ったものは org の全体と cli で打ち直した(diff と出力は `$SCRATCH/test-c3/mut/<id>/`)。mutant を入れない同じ実行(BASE)は org、cli とも通った。mutant と probe をすべて打ったあとの `git status --porcelain` は、この報告のほかは空だった。

| mutant | 変えたところ | 結果(落ちたテスト) |
| --- | --- | --- |
| B1(依頼) | `internal/org/merge_run.go:357` の base の case を消す(gate の head ならどの base でも記録する) | 落ちる。`TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch` の 5 行、`TestMerge_InterruptedThenMergedIntoAnotherBase`。CLI の `org_merge_test.go:381` も落ちる |
| B2 | 同じ行に `back.BaseRefName != "" &&` を足す(空の base は記録する) | 落ちる。「MERGED with an empty base」「MERGED with no baseRefName」 |
| B3 | head の case と base の case の順を入れ替える | 落ちる。「another head into another base names the head」「another head with no base names the head」、`TestMerge_RecordsOnlyAMergeOfTheGatesHead` の「MERGED with no headRefOid」 |
| B4(依頼) | `merge_run.go:41` の読み直しの field から `baseRefName` を外す | 落ちる。`TestMerge_MergesWithTheMethodAndRecordsPRMerged` の 4 行、`TestMerge_NotMergedRecordsNothing` の 3 行。CLI では 5 本の 8 行(偽の gh が読み直しに答えない) |
| B5 | 読み直しの field を同じ集合のまま並べ替える | 落ちる。B4 と同じ 7 行 |
| B6 | `merge_run.go:38` の `mergeOutcomeFields` から `baseRefName` を外す(読み直しは変えない) | 落ちる。6 本の 18 行 |
| B7 | `otherBaseMerged` の `if base == ""` を `if false` に | 落ちる。空と field なしの 2 行 |
| B8 | base がないときの「may have got」を「got」に | 落ちる。同じ 2 行 |
| B9 | `mergedElsewhere` が `listed`(merge-order の注)を書かない | 落ちる。5 行と中断のテスト |
| B10 | 別の base で `res.Merged` を立てる | 落ちる。5 行と中断のテスト。CLI の `org_merge_test.go:405`(stdout に `MERGED:` の行が出る) |
| B11 | base の case を `a.err == nil` のときだけにする | 落ちる。「gh pr merge failed, merged into another base」「MERGED with no baseRefName」と中断のテスト |
| B12 | `otherHeadMerged` にも merge-order の注を付ける | 落ちる。3 本の 8 行 |
| B13 | base があるときも「may have got」にする | 落ちる。3 行と中断のテスト |
| B14 | base の case の比べる相手を `run.r.defaultBranch` でなく `"main"` にする | 生き残る(org の全体、cli の `OrgMerge\|PrintMergeResult`)。probe `TestProbeC3_DefaultBranchOtherThanMain` が 2 行とも落とす。finding の T1 |
| W1(依頼) | `internal/org/merge.go:626` の `waits()` に `\|\| k == mergeRefusalChecksUnread` を足す | 落ちる。`TestMergeWait_EveryKindThatWaitsHasACase`(kind 5 に case がない)、`TestMerge_WaitChecksDoesNotWaitOutOtherRefusals` の 3 行(gh pr checks が読めない場合)と `TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` の 1 行 |
| W2 | W1 に加え、`internal/org/merge_wait.go:183` の `default:` が誤りを返さず `nil` を返す | 落ちる。`TestMergeWait_EveryKindThatWaitsHasACase` だけで打つと「0 lines」で落ちる。絞った実行では `TestMerge_WaitChecksDoesNotWaitOutOtherRefusals` の「gh pr checks fails」が眠らずに回り続け、4 分の timeout まで終わらない(下の注) |
| W3 | `waits()` に宣言のない kind(`mergeRefusalChecksUnread+1`)を足す | 落ちる。`TestMergeWait_EveryKindThatWaitsHasACase` だけが落とす(kind 6) |
| W4 | `merge_wait.go:172` の case から `mergeRefusalChecksZero` を外す | 落ちる。`TestMerge_WaitChecksWaitsOutPendingOrNoChecks` と `TestMerge_WaitChecksBudget` の 0 件の 4 行、`TestMergeWait_EveryKindThatWaitsHasACase` |
| W5 | `waits()` に `mergeRefusalOther` を足す | 落ちる。5 本の 60 行 |
| W6 | `waits()` に `mergeRefusalChecksNotPassed` を足す | 落ちる。4 本の 7 行 |
| N1(cycle 2 の n1) | `merge_wait.go:126` の `waited()` が待った物を並べ替えた順に書く | 落ちる。`TestMerge_WaitChecksChecksThenOrgShareTheBudget` |
| L1(cycle 2 の l) | `merge_wait.go:258` で台帳に org の `disbanded` が 1 つでもあれば待ちを終える(起き直しを見ない) | 落ちる。`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` の 2 行(S11a の `addInSleeps` に移したあとも落とす)、`TestMerge_WaitChecksWaitsForTheOrgToBeDisbanded` の 1 行、`TestMerge_WaitChecksChecksThenOrgShareTheBudget` |
| O1 | `internal/org/mergeorder.go:990` で、MERGED の PR でも base が既定のブランチでなければ `merged` にしない | 生き残る(org の全体、cli の `OrgMerge\|PrintMergeResult` と `MergeOrder\|Status\|Director`)。probe `TestProbeC3_MergeOrderListsAMergeIntoAnotherBaseAsMerged` が落とす(`wrong_base` になる)。finding の T2 |

依頼の 4 つ(base の case を消す B1、読み直しの field を変える B4、`waits()` に Unread を足す W1、CLI の `MERGED:` の行)はどれも 1 本以上のテストを落とした。cycle 2 で生き残った n1 に当たる N1 は、425a8543 で足したテストが落とす。

W2 は 2 か所を同時に変えた mutant で、`waits()` が true なのに `wait` に case がなく、`default:` も黙って `nil` を返すと、`Merge` は眠らずに確かめと待ちを繰り返す(fake の時計は眠らないと進まない)。S11a の `default:` の誤りがこの形を防いでいて、`TestMergeWait_EveryKindThatWaitsHasACase` はその 1 本だけでも落とす。

probe が HEAD 以外を見分けることも確かめた。B2 と B7 は `TestProbeC3_ReadBackBaseEdges` の null の行と `TestProbeC3_InterruptedThenOtherBaseEdges` の「no base」を、B3 は「another head into another base」を落とす。

## Coverage

- Statement: 依頼の 3 ファイルで 405/409 = 99.0%(org の全体の実行。`merge.go` 209/209、`merge_run.go` 110/113、`merge_wait.go` 86/87)。`mergeenforce.go` 90/91 を入れた 4 ファイルで 495/500 = 99.0%。org の package 全体は 94.5%
- Branch: Go の標準のツールでは取れない
- Function: S10 と S11a の関数はどれも 100%。`otherBaseMerged`(`merge_run.go:432`)、`mergedElsewhere`(`:450`)、`otherHeadMerged`(`:408`)、`mergeAndRecord`(`:347`)、`readBack`(`:331`)、`waits`(`merge.go:625`)、`waited`(`merge_wait.go:121`)、`orgWaitEnd`(`:253`)。4 ファイルで 100% でないのは `mergeLocked`(`merge_run.go:186`)91.7%、`mergeInterruptedAfterStart`(`:512`)80.0%、`wait`(`merge_wait.go:153`)96.8%、`CheckMergeEnforcement`(`mergeenforce.go:100`)91.7% の 4 つで、80% を下回るものはない
- Notes: 覆われていないブロックは 5 つ。cycle 2 と同じ 4 つ(行は動いた)は 1 文の異常の道で、`merge_run.go:194-196`(`regate.lock` が「持たれている」以外の理由で取れない)、`:203-205`(印を空にする `Truncate` が落ちる)、`:516-517`(`gh pr merge` が始まったあとの中断と記録の err が重なる)、`mergeenforce.go:111-113`(`Exec` が nil のときの既定の `realMergeOrderExec`)。増えた 1 つは S11a の `wait` の `default:`(`merge_wait.go:183-185`)で、注のとおり届かない道。`TestMergeWait_EveryKindThatWaitsHasACase` は待つ kind だけを渡すので通らず、`waits()` と case をずらす mutant(W1、W3〜W6)でだけ通る

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

wrapper、go1.25.8 の各実行、`-race`、繰り返し、カバレッジ、canary、probe のどれにも FAIL はない。既知の揺れも出なかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 2 の T1(検査の待ちのあとに org の待ちが来る順のテストがない) | 直った。`TestMerge_WaitChecksChecksThenOrgShareTheBudget`(`merge_wait_test.go:548`)が n1 に当たる N1 を落とす | mutant の表 |
| cross-review cycle 2 の 1 件(9 段の確かめのあとに PR の base が付け替えられると、gate の head のまま別のブランチへの merge を記録した) | 直った。S10 のテストが base の case を消す B1、空を記録する B2、`a.err` のときだけ外す B11 を落とし、CLI の `MERGED:` の行を出す B10 を CLI でも落とす | 上の 2 つの表 |
| S11a の動きを変えない整理(`mergeOutcomeFields`、`mergedElsewhere`、`addInSleeps`、`mrNext` の移動) | 移動の前からあるテストは全部通り、期待する文も変わっていない(`--json` の field に `baseRefName` が増えたほか)。cycle 2 の mutant l に当たる L1 は、`addInSleeps` に移した `TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` がまだ落とす | Test execution、mutant の表 |
| cycle 1、cycle 2 の mutant で落ちたもの | 今回打ち直した N1、L1 のほか、cycle 2 の表の a〜m のテストはどれも通った(テストの名前は変わっていない) | Test execution |
| 既知の揺れ | どの実行でも落ちなかった | Test execution の注 |
| テストのあとの worktree | `git status --porcelain` はテストの前と同じで空。ignore されたもので変わったのは、wrapper のログ `docs/evidence/verify-2026-10-17-232230.log` が増え、wrapper の evidence の保持数の整理で一番古い `docs/evidence/verify-2026-10-17-133504.log` が消えたことと、`.harness/cache/` が 322 個増えたこと。main のチェックアウトの `GIT_OPTIONAL_LOCKS=0 git status --porcelain` は空 | `$SCRATCH/test-c3/status-*.txt` |

## Test gaps

1. merge のテストの fixture は、org でも CLI でも origin の既定のブランチがいつも `main`。base を比べる相手を `"main"` と書き込む mutant B14 が生き残る。finding の T1
2. 別の base に merge された PR を merge-order が `merged` と並べることは、`otherBaseMerged` の Failed が書く(`merge_run.go:440`)うえ、`docs/tech-debt/README.md` の新しい行が既知の限界として書くが、テストは pin していない。merge-order の分類を変える mutant O1 が生き残る。finding の T2
3. 読み直しの `baseRefName` が JSON の null、`refs/heads/main`、`Main`、文字列でない場合のテストはない。probe で、どれも記録しない側に倒れることを確かめた。今の gh は短い名前の文字列を返すので、finding にしない
4. 中断のあとの読み直しで base がない場合と、別の head かつ別の base の場合は、中断のない場合の行と中断の 1 行(`merge_readback_test.go:92`)の組み合わせで足りるので、finding にしない(probe で確かめた)
5. CLI を通す base がない場合の行はない。org の層の 2 行と CLI の別の base の行があるので、finding にしない(probe で確かめた)
6. 覆われていない異常の道 4 つと届かない `default:`(上の Coverage)。finding にしない
7. 本物の GitHub の `gh pr view --json baseRefName` の答え、付け替えた PR の merge、`enforce_admins` の endpoint の答え、queue のある repo は、計画の Risks のとおりテストでは見ていない(偽の gh の振る舞いでだけ見る)

## Findings

| ID | Severity | File:line | Summary |
| --- | --- | --- | --- |
| T1 | LOW | `internal/org/merge_readback_test.go:27`(`TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch`) | 読み直しの base のテストは、fixture の既定のブランチがいつも `main` なので、`merge_run.go:357` の比べる相手が `run.r.defaultBranch` か `"main"` の書き込みかを見分けない。mutant B14 は org の全体と CLI の merge のテストで生き残る。既定のブランチを `trunk` にした行(ls-remote、PR の base、`gh api repos/o/r` の `default_branch`、`rules/branches/trunk` を trunk にする)で、trunk への merge を記録し `main` への merge を記録しないことを足すと落とせる(probe `TestProbeC3_DefaultBranchOtherThanMain` が HEAD で通り、B14 で 2 行とも落ちる)。今のコードは正しい |
| T2 | LOW | `internal/org/merge_run.go:440`(`otherBaseMerged` の Failed の注)、`internal/org/mergeorder.go:990`(`classifyMergeTarget`) | 別の base への merge の Failed は「merge-order はこのブランチを merged と並べる」と書くが、merge-order のテストの MERGED の PR は base がいつも `main` で、そのことを pin していない。MERGED でも base が既定のブランチでなければ `merged` にしない mutant O1 は、org の全体と cli の merge、merge-order、status のテストで生き残る。tech-debt の行を直すときに Failed の文が古くなっても、どのテストも落ちない。merge-order のテストに別の base の MERGED の行を 1 つ足すと落とせる(probe `TestProbeC3_MergeOrderListsAMergeIntoAnotherBaseAsMerged`)。今の文は今の動きと合っている |

情報(finding にしない):

- probe のとおり、読み直しの `baseRefName` が null、`refs/heads/main`、`Main`、文字列でないときは、どれも記録しない側に倒れる
- W2 のとおり、待つ kind に case がないまま `default:` が黙ると `Merge` は眠らずに回り続ける。今は `default:` の誤りと `TestMergeWait_EveryKindThatWaitsHasACase` がそれを防いでいる
- CLI のテストは PATH を fixture の bin に置き換えるので、PATH の先頭の canary は CLI では意味がない。偽の gh の外へ出ないことは PATH の置き換えが保証する

## Verdict: PASS

- Verdict: pass
- Pass: `GOTOOLCHAIN=go1.25.8 ./scripts/run-test.sh`(shell 41 本と Go 8 パッケージ)、go1.25.8 の org の全体(カバレッジ)、cli の全体、config、`-race` の実行、足した・変えたテストの `-count=20` と `-race -count=5`、canary の実行が、どれも exit 0 で FAIL は 0。足したテスト関数 69 本が全部通った。依頼の mutant(base の case を消す、読み直しの field を変える、`waits()` に Unread を足す、CLI の `MERGED:` の行)はどれも 1 本以上のテストを落とし、cycle 2 の T1 は直った。依頼の場合はどれも名前のあるテストか probe で確かめた。3 ファイルの statement は 99.0% で、80% を下回る関数はない
- Fail: なし
- Blocked: なし。T1 と T2 は LOW で、PR を止めない
- insight event に足す行(orchestrator が書く): phase `test`、cycle 3、verdict `pass`、findings は critical 0、high 0、medium 0、low 2
