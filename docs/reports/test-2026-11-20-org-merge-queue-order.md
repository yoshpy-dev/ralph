# Test report: org-merge-queue-order

- Date: 2026-11-20(cycle 1。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-order.md`(issue #342 の 2 本目、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の FR-12、FR-13、AC7、AC11)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b9ec3ae9`(base `1d05c9ac` = main、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから起動した。Bash tool の前景の上限 600 s を超えるので、perl の fork と setsid で切り離し、INT・HUP・QUIT・TERM を既定に戻し、標準入出力を `/dev/null` とログにつないだ。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと、新しいテストと変えたテストを `-v` で回し、main のテストを HEAD のコードに当てる run(AC3)、変異 34 個、skill の写しと Go の文の変異 2 個、絞った coverage を足した
- Evidence: `docs/evidence/test-2026-11-20-org-merge-queue-order.log`(10,531 行。run-test.sh の出力 5,829 行のあとに、`-v` の run、AC3 の run、変異の生成スクリプトと結果、coverage を足した。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-100338.log` にも書いた(同じく追跡外)
- 環境: go1.27.3 darwin/arm64、git 2.49.0、12 コア

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-20 19:03:37 〜 19:25:01 JST、1,284 s、rc 0。開始時の 1 分 load 4.83、終了時 6.87 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL: 0` の要約行 29 本、`N passed, 0 failed` の要約行の合計 773) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 9.5〜10 分(Go の段の最長の `internal/org` 694 s から逆算した概算) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 下の表 |
| `go test ./internal/org/ -count=1 -v -run 'Queue\|DirectorSkill'` | 107 関数、`=== RUN` 467 行(subtest を含む) | 467 | 0 | 0(`TestDirectorSkill*` の skip もない) | パッケージ 2.204 s |
| `go test ./internal/cli/ -count=1 -v -run 'OrgMergeOrder\|OrgMerge_'` | 53 関数、`=== RUN` 106 行 | 106 | 0 | 0 | 109.560 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.726 s |
| `internal/cli` | ok | 593.346 s |
| `internal/config` | ok | 13.900 s |
| `internal/gitroot` | ok | 3.926 s |
| `internal/insights` | ok | 1.258 s |
| `internal/org` | ok | 694.091 s |
| `internal/org/driver` | ok | 2.304 s |
| `internal/org/protocol` | ok | 1.027 s |
| `internal/scaffold` | ok | 1.466 s |
| `internal/toolchain` | ok | 17.956 s |
| `internal/upgrade` | ok | 2.919 s |

`internal/org` は golang pack の 1 パッケージの上限(30 分)の 39% で終わった。Go の段(19:13 ごろから)には、私の絞った org の run が重なった: 絞り込みの時間を測る run 1 回(235 s)と、変異 28 個を 1 個ずつ(1 個 19〜25 s)。その間に記録した 1 分 load は 3.9〜12.7。落ちたテストはなく、打ち直したテストもない。

`-v` の org の run は、`mergeorder_queue_test.go`、`merge_queue_order_test.go`、`director_skill_queue_test.go` の全関数と、queue の道の既存のテスト(`merge_queue_run_test.go` の `queueNext` の固定 `mqrNext` を使う 3 関数と、`merge_queue_wait_test.go` で同じ固定を使う関数)を含む。cli の run は `org_merge_order_test.go` と `org_merge_queue_test.go` の全関数と、`org_merge_test.go` の `TestOrgMerge_*` を含む。

### 計画の AC とテスト

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestClassifyMergeTarget_MergeQueueKeepsABranchWhoseBaseMovedReady`(64 subtest: 8 つの場面 × 受け入れの有無 × queue の読み 4 つ)、`TestMergeOrder_MergeQueueKeepsMovedBranchesReadyAndChangesTheNote`、`TestMergeOrder_NoMergeQueueKeepsTodaysListAndNotes`、`TestMergeOrder_MergeQueueThatCannotBeReadIsUnknown`(5 subtest) | queue があれば base の動いたブランチは `ready`(`accepted`)で、理由に queue の文が入る。ない・読んでいないなら今の `rebase`、読めないなら `rebase` に読めなかった理由の句が付く。base_ref が別のブランチ、先端が gate の head でない、PR の統合先が違う、の 3 つは queue があっても変わらない |
| AC2 | `TestCheckMerge_QueueKeepsABranchWhoseBaseMovedReady`(3)、`TestMerge_QueueEnqueuesABranchWhoseBaseMovedWithoutARegate`、`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase`(6)、`TestCheckMerge_QueueReadAtStep7ThatFails`(4)、`TestCheckMerge_QueueReadAtStep7Interrupted`、`TestMerge_QueueIsReadAgainInEachRunOfTheChecks`、`TestMergeEnforcement_TakesAQueueReadAlreadyMade`(7)、`TestMergeQueueLookup_OneReadServesTheEnforcementCheck` | base の動いたブランチを regate なしで queue に入れる。GraphQL の `mergeQueue` は checks の 1 回の run で 1 回だけ(step 7 で読めば step 11 は読まない)。step 7 で読めないと `rebase` の断りの末尾に理由が付き、gh が見つからなければ gh の断りになる。wait のあとの run は読み直す |
| AC3 | `TestCheckMerge_NoQueueKeepsTodaysCommands`(3)と、次の節の「main のテストを HEAD に当てる」 | queue のない repo で base の動いていないブランチの呼び出しの並びは `mxAll` と `mrMerged` のまま。base の動いたブランチは GraphQL の読みが 1 つ増え、今の文の `rebase` で断る |
| AC4 | `TestMerge_QueueTwoStagesForThreeReadyBranches`、`TestMerge_QueueDeferredBranchGoesInOnceItsChecksPass`、終了コード 7 は既存の `TestOrgMerge_PrintsAQueuedResultAndExitsSeven` | 1 段目で `enqueuePullRequest` が 3 本に 1 回ずつ、どれも `Queued`。2 段目は base が動いたあとも step 7 と 13 を通り、queue にある PR を入れ直さずに待って `pr_merged`(`via=merge_queue`)を記録する。後回しの A は何も入れずに断られ、B のあとで A が入る(`enqueuePullRequest` は B、A の順) |
| AC5 | `TestMergeOrder_ReadsNoMergeQueueUnlessAskedAndOnlyWithATarget`(3)、`TestMergeOrder_MergeQueueReadsOriginsRepository`(4)、`TestMergeOrder_MergeQueueReadNamesTheSSHKeyFixForAnUnresolvedHost`、`TestMergeOrder_InterruptedWhileTheMergeQueueIsReadIsAnError`、cli の `TestOrgMergeOrder_MergeQueueLineFollowsTheGuaranteeLine`、`TestOrgMergeOrder_MergeQueueLineLeavesOutNoQueueAndEscapesItsValues`、`TestOrgMergeOrder_HelpNamesTheMergeQueue`、cleanup の cli テスト(変異 E3 の行) | `merge queue:` の行は queue が「ある」か「読めない」ときだけ出て、`--json` に `merge_queue` が出る。対象のブランチがないときと `ralph org cleanup` の中では読まない。queue があれば 2 本以上の ready の注記が 2 段の手順の注記に替わる |
| AC6 | `TestDirectorSkill_PinsTheMergeQueueTexts`、`TestDirectorSkillQueueProblems`、`TestDirectorSkillGoPrinted`、`TestDirectorSkillParagraphAndList` | skill の 4 写しの 2 段の手順が見分けに使う英語の断片 14 個と Go の値 5 つが、それを出す Go の文字列リテラルと一致する。4 写しの一致そのもの(`scripts/check-skill-sync.sh`)は verify の範囲 |
| AC7 | `TestCheckMerge_QueueKeepsABranchWhoseBaseMovedReady` の「main moved only before step 13」、`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase` の step 13 の 4 行 | step 13 は queue の道で base の動きを通し、既定のブランチの改名、ブランチの先端の移動、ブランチの消失は今までどおり断る |

## main のテストを HEAD に当てる(AC3)

この branch は既存のテストファイルを 5 つ書き換えた(org の `merge_test.go`、`mergeorder_test.go`、`merge_queue_run_test.go`、cli の `org_merge_order_test.go`、`org_merge_queue_test.go`)。書き換えが期待した差分の外に及んでいないことを、main(`1d05c9ac`)の 5 ファイルを `go test -overlay` で HEAD のコードに当てて確かめた。org では、main の helper と合わない新しいテストファイル 3 つ(`mergeorder_queue_test.go`、`merge_queue_order_test.go`、`director_skill_queue_test.go`)を overlay で消し、main の 3 ファイルにある関数だけを `-run '^(…)$'` で回した。

| run | 関数 | 結果 | 落ちたもの |
| --- | --- | --- | --- |
| org: main の 3 ファイル | 86 | `=== RUN` 453 行、PASS 422、rc 1 | 8 関数(31 行)。`merge_queue_run_test.go` の queue の道の 7 関数と、`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck/not_ready:_rebase` |
| org: 同じ run で `mqrNext` だけを HEAD の文にした写し | 86 | PASS 451、rc 1 | `not_ready:_rebase` の行とその親だけ。断りの文は同じで、違うのは呼び出しの並びに `gh api graphql` が 1 つ増えたこと(`commands = [… "gh pr list" "gh api graphql"]`) |
| cli: main の 2 ファイル | 25 | `=== RUN` 47 行、PASS 26、rc 1 | merge-order の 15 関数は、main の `parseMergeOrderText` が新しい `merge queue: unknown (…)` の行を知らずに `t.Fatalf` で止まる。`TestOrgMerge_HelpDescribesTheMergeQueuePath` は置き換えた help の 2 文 |
| cli: main の 2 ファイルを 4 か所だけ合わせた写し | 25 | PASS 46、rc 1 | `TestOrgMerge_HelpDescribesTheMergeQueuePath` の help の 2 文(「run merge again to keep waiting while origin's default branch is where the gate left it …」と「… (run merge again to keep waiting while origin's default branch has not moved).」)だけ |

cli の 4 行目で合わせたのは、期待した差分そのものの 4 つ: 出力の `merge queue:` の行を読み飛ばす、`the merge queue of origin/` で始まる注記を外す、`--json` の `merge_queue` とその注記を外す、fake gh のログから queue の読み(`mergeQueue(branch: $branch)` を含む行)を外す。最初はログから `graphql` で終わる行をすべて外したところ、`enqueuePullRequest` の呼び出しまで消えて `TestOrgMerge_RequeueReachesTheMergeQueuePath` が落ちたので、queue の読みの行だけを外す形に直した(写しの差分は evidence にある)。fixture の origin はローカルのパスなので、合わせる前の run で出た `unknown` の理由は、ローカルの origin(GitHub の repository がない)か、main の fake gh が GraphQL に `[]` を返したこと(JSON として読めない)か、gh がないこと、のどれか。

結果は依頼の期待と一致した: `merge_test.go` の「not ready: rebase」の行(新しいテストに移った)、ローカルの origin で merge-order の出力に増える `merge queue: unknown` の行と注記、help の文、`queueNext` の文。ほかの差分はなかった。main の `mergeorder_test.go` の 60 関数は、そのまま HEAD でもすべて通った。

## 変異の確かめ

依頼の 5 つの振る舞いと、merge-order の CLI の行に変異を入れた。写しと overlay は `$TMPDIR` の下に置き、生成スクリプトは文字列の置き換えが 1 回だけ当たることを確かめる(evidence に全文)。org の変異は `-run '^Test(CheckMerge|Merge|ClassifyMergeTarget|DirectorSkill)'`(HEAD で 19 s、ok)、cli の変異は `-run 'OrgMergeOrder|OrgMerge_|OrgCleanup'`(HEAD で 139 s、ok)を `-vet=off -timeout 6m -overlay` で 1 個ずつ回した。build の失敗は 0 個。worktree は前後で `git status --porcelain --ignored` が同じ(go-build の cache と run-verify.sh の evidence を除く)。

| ID | 場所 | 変異 | 結果(落ちたテスト) |
| --- | --- | --- | --- |
| A1 | step 7 の読み | `class != MergeClassRebase` の早い戻りを外す(どの class でも読む) | 検出。queue のない repo の呼び出しの並び(`TestCheckMerge_NoQueueKeepsTodaysCommands`)をはじめ 56 関数 |
| A2 | step 7 の読み | queue ありで分類し直した class が ready・accepted でないときの戻りを外す(base_ref が別のブランチでも読む) | 検出(`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase`) |
| A3 | step 7 の読み | 一度も読まない | 検出(新しい 9 関数) |
| A4 | step 7 の分類 | 読んだあとに分類し直さない | 検出(7 関数) |
| A5 | step 7 の読み | gh が見つからないときの断りを外す | 検出(`TestCheckMerge_QueueReadAtStep7ThatFails`) |
| B1 | step 11 の受け取り | `queueReadFor` が常に「読んでいない」を返す | 検出(1 回だけの読みを数える 7 関数) |
| B2 | step 11 の受け取り | `queueReadFor` のブランチの比較を外す | 検出。ただし直接の単体テスト `TestMergeCheck_QueueReadForTakesOnlyTheBranchStep7Read` だけ(Test gaps の 2) |
| B3 | `enforce` | 渡された読みを無視して必ず読む | 検出(8 関数) |
| B4 | `enforce` | 読めなかった読みを ok でないとしない | 検出(6 関数) |
| B5 | `checkMergeAfterChecks` | `queue:` に常にゼロ値を渡す | 検出(6 関数) |
| B6 | step 7 の読み | 読んだブランチ名を記録しない(`queueOf` を空に) | 検出(6 関数) |
| C1 | step 13 | queue の道でも base の動きで断る | 検出(6 関数) |
| C2 | step 13 | queue の道では既定のブランチの改名を見ない | 検出(`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase`) |
| C3 | step 13 | queue の道ではブランチの消失を見ない | 検出(同上) |
| C4 | step 13 | queue の道ではブランチの先端の移動を見ない | 検出(同上) |
| D1 | 分類 | 読めない queue も ready にする | 検出(4 関数) |
| D2 | 分類 | queue がない(absent)のに ready にする | 検出(3 関数) |
| D3 | 分類 | queue があれば base_ref が別のブランチでも ready にする | 検出(2 関数) |
| D4 | `classifyAccepted` | base が先端と同じでも queue の文を付ける | 検出(`TestClassifyMergeTarget_MergeQueueKeepsABranchWhoseBaseMovedReady`) |
| D5 | `classifyAccepted` | queue の文を付けない | 検出(2 関数) |
| D6 | `mergeQueueUnreadText` | 読めなかった理由の句を付けない | 検出(4 関数) |
| D7 | `mergeOrderListNotes` | `present()` の代わりに「読んだ」だけで 2 段の手順の注記にする | 検出(`TestMergeOrder_NoMergeQueueKeepsTodaysListAndNotes`) |
| D8 | `present()` | state を見ない | 検出(同上) |
| D9 | `mergeOrderQueueNotes` | 読めなかった注記を出さない | 検出(2 関数) |
| D10 | 分類 | base が先端と同じ場合の枝を外す | 検出(104 関数) |
| D11 | `mergeOrderListNotes` | 読めない queue でも 2 段の手順の注記にする | org の絞り込みでは生き残り、cli の `TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes` で検出(Test gaps の 1) |
| E1 | `MergeOrder` | `ReadMergeQueue` によらず読む | 検出(`TestMergeOrder_ReadsNoMergeQueueUnlessAskedAndOnlyWithATarget` など 16 関数) |
| E2 | cli の merge-order | `ReadMergeQueue: false` | 検出(8 関数) |
| E3 | cli の cleanup | cleanup の `MergeOrder` に `ReadMergeQueue: true` を足す | 検出(`TestOrgCleanup_CleansUpABranchMergedOnGitHub` など 3 関数) |
| E4 | `readMergeOrderQueue` | `p.Origin` を使わずに必ず origin を読む | 検出(`TestMergeOrder_MergeQueueReadsOriginsRepository`) |
| E5 | `readMergeOrderQueue` | ssh の別名のヒントを付けない | 検出(`TestMergeOrder_MergeQueueReadNamesTheSSHKeyFixForAnUnresolvedHost`) |
| G2 | `readMergeOrderQueue` | queue の読みに merge-order の `GHTimeout` を渡さない(lookup の既定 20 s になる) | 検出(2 関数) |
| F1 | cli の `mergeQueueLine` | 読めないときの行を出さない | 検出(3 関数) |
| F2 | cli の `mergeQueueLine` | queue がないときも行を出す | 検出(3 関数) |

34 個すべてを検出した(D11 は cli のテストで)。

`director_skill_queue_test.go` は skill と Go のファイルをディスクから読むので、overlay は効かない。`git archive HEAD` の写しで 2 個を確かめた。K1(`templates/base/.agents/skills/director/SKILL.md` の `` `and <N> more` `` を `` `and <N> others` `` に)は `TestDirectorSkill_PinsTheMergeQueueTexts` がその写しの名前を挙げて落ちた。K2(`internal/cli/org_merge_order.go` の `has one%s: a branch` を `has a queue%s: a branch` に)は同じテストと `TestDirectorSkillQueueProblems` が落ちた。写しを変えない対照は 2 関数とも PASS で、skip はなかった。

## Coverage

- Statement: 絞った run(org は `^Test(CheckMerge|Merge|ClassifyMergeTarget|DirectorSkill)`、cli は `OrgMergeOrder|OrgMerge_|OrgCleanup|ExitCode`)の profile で、この branch が足した行のうち文にあたる行は、org 141/143(98.6%)、cli 76/76。新しい関数は、`mergeOrderQueueResult`(66.7%)を除いて 100%
- Branch: Go の標準ツールでは測れない。変異の結果を代わりに置く
- Function: org で 100% でないのは `mergeOrderQueueResult` と、既存の `newMergeEnforcementRun`(90.0%、`Exec` が nil のときに本物の exec を使う枝。テストはどれも `Exec` を渡す)
- Notes: 実行されない足した行は `internal/org/mergeorder_queue.go:58-59`(`mergeOrderQueueResult` の「読んでいない」で nil を返す枝)だけ。`MergeOrder` は読むときにしか呼ばず、読みは「読んでいない」を返さないので、`MergeOrder` からは届かない。パッケージ全体の profile は取っていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #342: queue のある repo で、gate の base の古いブランチを merge-order と merge が `rebase` と読み、director が merge のたびに regate の輪を回す | 直った | AC1、AC2、AC4 のテストが通り、変異 A3・A4・C1・D5 などで落ちる |
| queue のない repo の `ralph org merge` と merge-order(AC3) | 変わらない | main のテストを HEAD に当てた run で、落ちたのは期待した差分だけ。全体の run で merge と merge-order の既存のテストファイルはすべて ok |
| 1 本目の queue の道(enqueue、待ち、記録、終了コード 7、`--requeue`) | 変わらない | `-v` の run で `merge_queue_run_test.go`、`merge_queue_wait_test.go` の queue の道のテストと cli の `TestOrgMerge_PrintsAQueuedResultAndExitsSeven`、`TestOrgMerge_RequeueReachesTheMergeQueuePath` が通る。変わったのは `queueNext` の文の条件だけ |

## Test gaps

`/sync-docs` に渡すもの。どれも verdict は変えない。

1. (LOW)読めない queue と 2 本以上の ready の組み合わせで、「1 本 merge すると他は rebase になる」の注記が残ることを、org の層のテストは縛っていない。D11(読めない queue でも 2 段の手順の注記にする)は org の絞り込みで生き残り、cli の `TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes` だけが落とした。このテストが縛るのは、fixture の origin がローカルのパスで queue が unknown になるからで、fixture の origin が変わると縛りが外れる。足すなら `TestMergeOrder_MergeQueueThatCannotBeReadIsUnknown` に ready の 2 本の行を 1 つ足し、注記が今の文のままであることを見る
2. (INFO)`queueReadFor` のブランチの比較(`m.queueOf != def`)は、直接の単体テストだけが縛る(B2)。`checkMerge` の 1 回の run では、step 7 と step 11 が同じ ls-remote の既定のブランチを使う(`merge.go:385` の `r.defaultBranch = remote.defaultBranch`)ので、比較が偽になる道がない。守りのための比較で、直す必要はない
3. (INFO)`mergeOrderQueueResult` の nil を返す枝(`internal/org/mergeorder_queue.go:58-59`)は `MergeOrder` から届かず、実行されない。守りのための枝
4. (INFO)本物の GitHub での確かめは計画の Non-goals(ユーザーの決定 2)。register(`docs/tech-debt/README.md`)の「The merge queue path of `ralph org merge` … was built on a fake gh」の行は 1 本目の AC1 と AC2 を挙げている。この PR が足した fake gh だけの読み(merge-order の GraphQL の読み、base の動いた step 7 と 13、director の 2 段の手順)も同じく実地の証拠がないので、その行に足すかを `/sync-docs` で決める
5. (INFO)director の 2 段の手順(wait を 5 分で区切る打ち直し、60 分で人に伝える、2 回続けて終了コード 7 なら止める)は skill の文で、テストが縛るのは、手順が見分けに使う断片と Go の文の一致(AC6)と、ralph の側の振る舞いを org の層でなぞる AC4 まで。director が手順のとおりに動くかは、release のあと下流で確かめる

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,284 s、`(cached)` なし)、新しいテストと変えたテストの `-v` run(org 107 関数、cli 53 関数)、AC1〜AC7 のテスト、main のテストを HEAD に当てた run(落ちたのは期待した差分だけ)、変異 34 個すべてを検出(D11 は cli で)、skill の写しと Go の文の変異 2 個を検出
- Fail: 0
- Blocked: なし
