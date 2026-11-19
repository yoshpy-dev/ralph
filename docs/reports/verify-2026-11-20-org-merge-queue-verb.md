# Verify report: org-merge-queue-verb

- Date: 2026-11-20(パイプラインの 1 回目の run。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`。`./scripts/plan-visual.sh digest` の値 `aef9c99a6ea5` は、計画の `- Approved:` の行と一致する
- Spec: `docs/specs/2026-11-19-org-merge-queue.md`(2 本のうち 1 本目: FR-1〜FR-11、FR-5b、NFR-1〜NFR-4)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `d919479a`、HEAD `435ecfe9`、40 ファイル、+5213 −278)。手元の `main` と `origin/main` はどちらも `d919479a` で、branch は main の先端の上にある(fetch はしていない)。self-review の 2 回目(`bbbf64e7`)のあとの `435ecfe9` は、L9(FR-5b の断りの文を「less than 5m0s before it」に)と L10(help の詰め直し)を直し、`--wait-checks` の flag の文に merge queue を足した
- Evidence: `docs/evidence/verify-2026-11-20-org-merge-queue-verb.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。静的検査の出力、GitHub の GraphQL の schema の読みだけの probe、HEAD から作った binary の `ralph org merge --help` が入っている。ラッパー自身のログは `docs/evidence/verify-2026-11-19-233220.log`

## Spec compliance

テストは走らせていない(`/test` の担当)。判定はコードを読んだ結果で、各 AC に当たるテストがあることまでを確かめた。テストが通るかは `/test` が確かめる。

### 計画の受け入れ条件

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: queue と必須の検査がある既定のブランチで `--wait-checks 30m` のとき、`enqueuePullRequest` を gate の head で 1 回打ち、MERGED を読んだあと `pr_merged`(`via=merge_queue`、`merge_commit` は `mergeCommit.oid`)を記録して終了コード 0 | Met | enqueue は `sendEnqueue`(`internal/org/merge_queue_run.go:235-265`)が `enqueueMergeQueuePR` を `r.head` で 1 回呼ぶ。待ちは `waitForQueue`(`internal/org/merge_queue_wait.go:113-164`)、MERGED の判定は `queueDecided`(`:201-216`)で、head が gate の head、base が既定のブランチのときだけ `Merged` と `MergeCommit`(`pr.MergeCommit`)を立てる。記録は `recordQueueMerge`(`:307-330`)が `mergeRecordDetails(res) + " via=merge_queue"` を書く。テストは `TestMerge_QueueWaitRecordsTheMergeOnceTheQueueMergedIt`、`TestMerge_QueueWaitRecordDetails`、`TestMerge_QueuePathEnqueuesThePRAtTheGatesHead` |
| AC2: 入れたあとに `RemovedFromMergeQueueEvent`(reason `failed_checks`)が出たら、記録せず reason を出して終了コード 1 | Met | queue になく MERGED でもない読みは `queueLeft`(`merge_queue_wait.go:236-253`)の `Failed` になり、入れた時刻以後の最新の removal(`latestMergeQueueRemovalAfter`、`internal/org/merge_queue.go:443-450`)の時刻、actor、reason を `mergeQueueRemovalText`(`:479-489`)で出す。`Failed` は CLI で終了コード 1。テストは `TestMerge_QueueWaitRecordsNothingWhenThePRLeavesTheQueue` |
| AC3: queue のない repo では今の merge のテストがすべて変わらずに通る | Met(注つき) | queue のない道は `mergeLocked` の `if r.queue` の外(`internal/org/merge_run.go:368-385`)で、`mergeAndRecord` を前と同じ `MergeResult` で呼ぶ。`recordMerge` の Details は `mergeRecordDetails` に移しただけで文字列は同じ。`otherHeadMerged`、`otherBaseMerged`、`mergeInterruptedAfterStart`、`mergeRecordErr` も文は同じ。既存のテストで消えたのは、queue を断る期待の 3 行(`merge_test.go` の 1 行、`mergeenforce_test.go` の 2 行)だけで、queue の道のテストに移った。`TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn` には「queue の道に入らない」の確かめが足された。注: `TestOrgMerge_Help` の 3 つの固定の文字列は help の文が変わったので書き換えた(偽になった「shows no merge queue」を外し、cleanup の lock の括弧と `--wait-checks` の flag の文に queue を足した)。queue のない repo の挙動の確かめではないが、計画は S2 の 3 行の移しだけを書いている |
| AC4: `--wait-checks 0`、または予算の中で MERGED にならなければ、記録せず、queue に残したまま終了コード 7。予算 0 は enqueue の答え(入れずに続けるときは直前の読み)で決め、読み直さない | Met | 予算 0 では `mergeWait.left()`(`internal/org/merge_wait.go:124-126`)が偽になり、`mergeLocked` は enqueue の結果の `Queued` をそのまま返す(`merge_run.go:363-364`)。入れずに続ける道では `queued(res, false, pr.Entry)`(`merge_queue_run.go:89-93`)。予算切れは `queuedAtEnd`(`merge_queue_wait.go:274-281`)。`Queued` は `printMergeResult` で `mergeQueuedError` になり(`internal/cli/org_merge.go:473-475`)、`ExitCode` が 7 にする(`internal/cli/exitcode.go:50-51`)。queue から外す呼び出し(`dequeuePullRequest`)はどこにもない。テストは `TestMerge_QueueWaitLeavesThePRQueuedWhenTheTimeRunsOut`、`TestOrgMerge_PrintsAQueuedResultAndExitsSeven`、`TestExitCode` の 3 行 |
| AC5: すでに queue にある PR(head は gate の head)を打ち直すと、`enqueuePullRequest` を打たずに待ちから続ける | Met | `enqueue` は `pr.InQueue` なら `sendEnqueue` を呼ばずに返す(`merge_queue_run.go:89-93`)。queue にあって head が違えば断る(`queueReadDisagrees`、`:158-161`)。テストは `TestMerge_QueuePathDoesNotEnqueueAPRInTheQueueAlready` |
| AC6: queue があり必須の検査がなければ断る(終了コード 1、記録なし) | Met | `enforceQueue`(`internal/org/mergeenforce.go:312-338`)は rules の検査が 0 件なら classic の保護を読み、それも 0 件なら `OK` を立てずに返す。step 11 の断りになる。テストは `TestMergeEnforcement_TakesTheQueuePath`、`TestCheckMerge_QueueWithNoRequiredCheckIsRefused` |
| AC8: MERGED だが `headRefOid` が gate の head と違えば記録せずに終了コード 1。base が既定のブランチと違っても同じ | Met | `queueDecided` の最初の 2 つの枝(`merge_queue_wait.go:205-208`)が `queueMergedElsewhere` の `Failed` にし、記録しない。文は gh pr merge の道と同じ `otherHeadClause` と `otherBaseClause`(`merge_run.go:597-638`)。テストは `TestMerge_QueueWaitRecordsNothingForAMergeOfAnotherHeadOrBase` |
| AC10: 最新の gate の記録より後に `RemovedFromMergeQueueEvent` がある PR は、`--requeue` なしでは断る(終了コード 1)。`--requeue` があれば入れる | Met | `enqueue`(`merge_queue_run.go:94-100`)。gate の時刻は step 8 が `mergeGateTime`(`:36-52`、受け入れのほうが後ならその時刻)で `mergeCheck.gateAt` に置き(`internal/org/merge.go:373`)、queue の道でだけ `mergeReady.gateAt` に写す(`:421-422`)。`--requeue` は `MergeParams.Requeue` に渡る(`org_merge.go:341`、`:354`)。テストは `TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue`、`TestMerge_QueuePathComparesRemovalsWithTheAcceptanceWhenLater`、`TestOrgMerge_RequeueReachesTheMergeQueuePath` |
| AC12: 待ちの終わりに queue の状態が読めなければ、記録せず「queue にあるかもしれない」と出して終了コード 1。待ちのあいだの読み失敗は予算の中で読み直す | Met | 読み失敗は `last` に残して待ちを続け(`merge_queue_wait.go:138-142`)、予算の終わりの最後の読みが失敗なら `queueUnread`(`:260-266`)が「it may still be in the merge queue」を含む `Failed` にする。テストは `TestMerge_QueueWaitReadsAgainAfterAFailedRead` |
| AC13: queue の道の gh は `--hostname <解いた host>` と `GH_PROMPT_DISABLED=1`、enqueue のあとの呼び出しは `runDetached`。文字列の変数は `-f`、PR の番号は `-F`。形の確かめに通らない node ID と SHA は渡さない | Met | argv は `mergeQueuePRArgs` と `mergeQueueEnqueueArgs`(`merge_queue.go:196-207`)、環境は `:401` と `:427`、host は `r.apiHost`(#340 の解いた host)。待ちの読みは `run.m.runDetached`(`merge_queue_wait.go:175`)。node ID は `mergeQueueNodeIDPattern`(`merge_queue.go:66`)、head は `gitObjectPattern` で確かめる(`:165-177`)。テストは `TestMergeQueue_ReadRunsOneGraphQLCallWithTheNumberAsAnInt`(`$number: Int!` と `-F` を固定)、`TestMergeQueue_EnqueueRunsOneGraphQLCallWithStringVariables`、`TestMergeQueue_QueriesAreFixedAndDeclareTheirVariables`、`TestMergeQueue_ChecksRefuseBadValuesBeforeAnyGH`、`TestMerge_QueuePathPinsTheGraphQLCallsToTheResolvedHost`、`TestMergeThroughQueue_ReadsThroughRunDetached`。加えて、PR の読みの query を github.com の本物の schema に通した(Observational checks) |
| AC15: lock を離して待つ 2 つの merge が同じ MERGED を見ても、`pr_merged` は 1 行だけ | Met | `recordQueueMerge` は manifest の lock の中で台帳を読み、`queueMergeRecorded`(`merge_queue_wait.go:340-358`)が、最新の gate の記録より後に同じ branch・PR・head の `pr_merged` を見つけたら書かずに `Note` を立てる。gate の記録の位置は `collectMergeTargets` の `gate.pos`(台帳の index)。テストは `TestMerge_QueueWaitRecordsAMergeOnlyOnce` |
| AC16: FR-5b の時刻の比べ方を、gate の ts より前・同じ・後と、5 分の幅の内と外でテストする。幅の内なら断る | Met | `mergeQueueRemovedAfterGate`(`merge_queue.go:458-460`)は「gate の時刻 − 5 分」より厳密に後を数える。`TestMergeQueue_RemovalAfterGateAllowsForClockSkew` が 1 時間前、幅のすぐ外、幅ちょうど(数えない)、幅のすぐ内、1 秒前、同時、1 秒後、1 時間後の 8 行を持つ。`435ecfe9` で断りの文(`gateWhen`、`merge_queue_run.go:216`)も「less than 5m0s before it」になり、help(`--requeue` の段落と flag の文)、README と境目が揃った |
| AC14: README の `ralph org merge` の説明と `--help` が、queue の道、`--requeue`、終了コード 7 を書く。tech-debt に確かめていない前提と実地の確かめの予定を書く | Met(注つき) | README の 133 行(表)と 296 行(段落)、help(`org_merge.go:136-185`、`:276-279`、flag の文 `:351-356`)、register の 536 行。注: README と skill の「打ち直せば待ちを続ける」には条件がない(Documentation drift の D1) |

### 仕様の要件(1 本目の分)

| Requirement | Status | Evidence |
| --- | --- | --- |
| FR-1 queue の判定、読めなければ断る | Met | `enforce`(`mergeenforce.go:155-175`)は GraphQL の `mergeQueue` か rules の `merge_queue` で queue の道に入る。`readMergeQueue`(`:249-299`)の失敗と rules の読みの失敗は、今までどおり断る |
| FR-2 queue の道は必須の検査を 1 件以上求め、迂回は読まない | Met | `enforceQueue`(`:312-338`)。`enforceRulesets`、`readMergeClassicStrict`、`enforceClassic` は queue の道で呼ばれない |
| FR-3 step 12 を飛ばし、方法は queue の設定か `queue` | Met | `merge.go:421-422`(`cmp.Or(strings.ToLower(enf.QueueMethod), "queue")`)。設定の値は `mergeMethodFlags` の 3 つだけを通す(`mergeenforce.go:291-295`) |
| FR-4 `enqueuePullRequest` を gate の head で。node ID は状態の query から。`gh pr merge` と `--admin` は使わない | Met | `sendEnqueue` は `pr.ID`(読みの答え)と `r.head` を渡す。queue の道は `mergeAndRecord` に入らない(`merge_run.go:360-367`) |
| FR-5 queue にあれば入れ直さない | Met | AC5 と同じ |
| FR-5b 最新の gate の記録より後に外されていれば `--requeue` なしで断り、理由と時刻を出す | Met | AC10、AC16 と同じ。断りの文は `removedAfterGate`(`merge_queue_run.go:181-186`) |
| FR-6 30 秒ごとの読み、待つあいだ `regate.lock` を持たない | Met | `mergeLocked` の `defer lock.release()`(`merge_run.go:332`)が戻るときに lock を離し、そのあとで `Merge` が `mergeThroughQueue` を呼ぶ(`:293-294`)。間隔は `mergeWait.interval` |
| FR-7 予算は `--wait-checks` 1 つ、0 なら入れたところで返す | Met | `waitForQueue` は `w.started` が空のときだけ予算を始める(`merge_queue_wait.go:114-117`)。0 は AC4 と同じ |
| FR-8 MERGED で head と base が合えば `pr_merged`(`via=merge_queue`)、片付けは今までどおり | Met | AC1 と同じ。片付けは `queueMergeCleanup`(`merge_queue_wait.go:369-390`)が `regate.lock` を取り直し、marker を空にして(`:386`)`mergeCleanup` を呼ぶ。取れなければ `ralph org cleanup` を名指す |
| FR-9 外されたら記録せず reason と actor を出して 1 | Met | AC2 と同じ |
| FR-10 予算切れは queue に残し、終了コード 7 | Met | AC4 と同じ |
| FR-11 読み失敗は予算の中で読み直し、終わりに分からなければ 1 | Met | AC12 と同じ |
| NFR-1 queue のない repo は変わらない | Met(注つき) | AC3 と同じ。step 11 の query に `configuration { mergeMethod }` が足されたので、queue のない repo でも gh に渡す query の文字列は変わった。github.com の schema では通ること(`mergeQueue: null`)を probe で見た。GHES の schema は見ていない |
| NFR-2 確かめていない前提を tech-debt に | Met | register の 536 行、計画の Assumptions |
| NFR-3 分からない状態では記録しない | Met | 記録は `queueDecided` の MERGED の枝のあとだけ。enqueue の答えに entry がない、読めない、外された、別の head か base は、どれも記録しない |
| NFR-4 `--hostname`、`GH_PROMPT_DISABLED=1`、待ちのあとは `runDetached` | Met | AC13 と同じ。`-R` は `gh api graphql` には当たらない(owner と name は変数で渡す) |

仕様の Security considerations(固定の query、`-f`/`-F`、node ID と SHA の確かめ、reason の制御文字を落とす)も、上の AC13 と `mergeQueueRemovalText` で満たす。FR-12、FR-13、AC7、AC9 の skill の部分、AC11 は計画の Non-goals(2 本目)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | Pass(rc=0) | shellcheck と `sh -n`(hook と verify のスクリプト)、`check-sync.sh`(DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の verifier(`gofmt -l` は空、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`d919479a..435ecfe9` は clean)。言語は golang だけが選ばれた |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-org-merge-queue-verb.md` | `aef9c99a6ea5` | 計画の `- Approved:` と一致 |
| `go build -o $TMPDIR/ralph-omq ./cmd/ralph` と `org merge --help` | rc=0 | help の文は下の Observational checks |

## Documentation drift

D1 と D2 は self-review の L2 の残りと L11 で、計画が `/sync-docs` に渡すと決めたもの。D3〜D6 はこの verify で見つけた。どれも LOW 以下で、verdict は変えない。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1(L2 の残り): README と skill の「打ち直せば待ちを続ける」 | No(LOW) | Go の文(`queueNext`、help、`ErrMergeQueued`、`exitcode.go`、`main.go`)は「origin の既定のブランチが gate のときのままなら」の条件を書く。条件がないのは README の 296 行(「running merge again keeps waiting without enqueueing the PR twice」)、org skill の 396 行(「打ち直せば待ちを続ける」)と 2002 行(「打ち直せば待ちを続ける。」)、director skill の 1742-1743 行(「あとで `--wait-checks` を付けて打てば、queue に入れ直さずに待ちを続ける」)。skill はそれぞれ 4 写し。既定のブランチが動けば step 7 と 13 が queue を読む前に断る。もう 1 つ、queue にある PR を regate した道(push で PR が外され、次の merge は FR-5b で断る。そのときの手は `--requeue` だが、`requeueNext` の文と register の 536 行はこの道を書かない)も残っている |
| D2(L11): register の 537 行(800 行の目安) | No(LOW) | 払う手順が名指す `TestMergeQueue_LatestRemovalAfterIsStrictlyAfter` は `5e409db0` で `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond`(`internal/org/merge_queue_test.go:676`)に改名された。行数は HEAD で `merge.go` 826(行は 822)、`merge_queue_test.go` 807(806)、`merge_queue_wait_test.go` 854(859)、`merge_queue.go` 489(501)、`merge_queue_run.go` 333(326)、`merge_queue_run_test.go` 647(646)、`internal/cli/org_merge_queue_test.go` 348(332)。Trigger の「The next change to either test file」は同じ branch の `5e409db0` が文字どおりには満たしている |
| D3: register の 395 行と 502 行の持ち越しの注 | No(LOW) | どちらも「two fragments of `TestOrgMerge_Help`」と「1248 → 1249 lines」と書くが、`435ecfe9` が 3 つ目の固定の文字列(`--wait-checks` の flag の文、`internal/cli/org_merge_test.go:1201-1203`)を変え、ファイルは 1250 行になった。395 行は数を 56fc4cab に結びつけているが、fragment の数は今と合わない。502 行は数を commit に結びつけていない。どちらも `org_merge_queue_test.go` を 332 行と書く(今 348) |
| D4: register の 182 行(`/org` skill の命令の表) | No(LOW) | Trigger (a)「The next edit to the `/org` skill's command table」に、この PR の `merge` の行の書き換え(4 写し)が当たる。前の PR はこの trigger に当たるたびに持ち越しの注を書いてきたが、この PR の注はない |
| D5: register の 534 行(`merge_test.go` の行数) | No(INFO) | 行は 835 行と書く。この branch が queue の 1 行を消して 833 行になった。Trigger(in-place の直しより大きい変更)には当たらない |
| D6: merge の間の `regate.lock` の文 | No(LOW) | help の「While it runs, except while it waits, ralph org regate, another ralph org merge, and ralph org cleanup are refused on the ledger」(`internal/cli/org_merge.go:281-283`)、org skill の 396 行「merge の間は台帳の regate とほかの merge と `cleanup` が断られる(`--wait-checks` で待つ間は持たない)」と 2145-2147 行。queue の道では、待ちのあと `pr_merged` を manifest の lock だけで記録し(`recordQueueMerge`)、`regate.lock` は片付けのときだけ取り直す(`MergeParams.RepoRoot` が空なら取り直さない)。待ちが終わってから取り直すまでの間も regate、merge、cleanup は断られない。README の 296 行の queue の段落は「takes `regate.lock` again for the cleanup」と書いていて合う |
| D7: 仕様の Open questions の 1 つ目(終了コードの値、案 7) | No(INFO) | 計画の D6 が 7 に決め、コードも 7。この PR は仕様の FR-4 と Dependencies を直したが、この項は開いたまま |
| D8: 計画の受け入れ条件のチェックボックス | No(INFO) | すべて `[ ]`。Progress checklist は S1〜S6 と self-review まで更新済み |
| AGENTS.md の `internal/org/` の行 | Yes | `merge_queue.go`、`merge_queue_run.go`、`merge_queue_wait.go`、queue の道、終了コード 7 を書く |
| `docs/specs/2026-10-07-org-multi-org-director.md` の 8a-3a | Yes | 「merge queue のある既定のブランチを断ることは #342 の 1 本目で改めた」を足した。7c-2 の doctor の判定の文は変わらず、doctor は変えていない |
| `docs/specs/2026-11-19-org-merge-queue.md` | Yes | FR-4 の node ID の取り方(D3)と Dependencies を直した |
| `cmd/ralph/main.go`、`internal/cli/exitcode.go` のコメント | Yes | 終了コード 7 を足した。7 はほかの動詞の値(3〜6)とぶつからない |
| Go の doc comment(`lockfile.go`、`regate.go`、`spawn.go`、`cleanup.go`、`seat.go`、`merge.go`、`merge_wait.go`) | Yes | queue の道の lock の持ち方と `pr_merged` の Details(`via=merge_queue`、`method=queue`)を書く |
| register の 241、257、312、395、401、410、502 行の持ち越しの注 | 一部(D3) | 当たった trigger と払わなかった理由を書いている。数のずれは D3 |
| register の 536 行(確かめていない前提) | Yes | 計画の Assumptions と合う。下の probe で、2 つの query は github.com の schema で通ることを見た。行に書き足すなら `/sync-docs` の判断 |
| 計画の Progress の S6 の記録 | Yes | `5e409db0` の L1〜L7 と、`435ecfe9` の前までが書かれている。`435ecfe9` の分はまだ書かれていない(INFO) |

repo 全体で、前の断りの文(「does not merge into a branch with a merge queue」「shows no merge queue」「merge queue がなく」「merge queue がある」)を探した。残っているのは仕様の Current state(前の挙動の説明)、計画の Related request、help に戻らないことを確かめるテストだけで、どれも正しい。

## Observational checks

GitHub の GraphQL の schema に、コードの 2 つの query を読みだけで通した(`gh api --hostname github.com graphql`、アカウント `yoshpy-dev`、2026-11-20)。query の文字列は Go の定数から python で取り出した。mutation(`enqueuePullRequest`)は送っていない。

- `mergeQueuePRQuery` を `-F number=351` で(この repo の PR #351): rc=0。答えは `id`(`PR_kwDO...`、`mergeQueueNodeIDPattern` に合う形)、`state: MERGED`、`headRefOid`、`baseRefName: main`、`mergeCommit.oid`、`isInMergeQueue: false`、`mergeQueueEntry: null`、`timelineItems.nodes: []`。`parseMergeQueuePR` が読む欄の名前と形に合う
- 同じ query を `-f number=351` で(負の対照): rc=1、`Variable $number of type Int! was provided invalid value`。計画助言の HIGH(番号を `-f` で送ると型エラー)と、`-F` にした直しの理由を本物で確かめた
- `mergeQueueQuery`(step 11、`configuration { mergeMethod }` つき)を `branch=main` で: rc=0、`{"data":{"repository":{"mergeQueue":null}}}`。queue のない repo で step 11 の query が schema で通る(NFR-1)

HEAD から作った binary の `ralph org merge --help` は、queue の道の段落、`--requeue` の段落(「less than 5 minutes before it」)、終了コード 7、`--wait-checks` と `--requeue` の flag の文を出す。L10 の短い行(`exits 0. When`)はなくなった。

## Coverage gaps

- テストは走らせていない。AC ごとのテストがあることは読んで確かめたが、通るかは `/test` が確かめる
- `enqueuePullRequest` の mutation は本物に送っていない(副作用のない確かめ方がない)。入力の型(`ID!`、`GitObjectID!`)と答えの `mergeQueueEntry { state position enqueuedAt }` は schema で確かめていない。register の 536 行のとおり、release のあと下流で確かめる
- GHES の schema は見ていない。`MergeQueue.configuration` がない版の GHES では、step 11 の query が検証で落ち、queue のない repo の merge も断るはず(推測で、未確認)。github.com では通る
- queue のある repo での実際の挙動(queue から外す時刻と timeline の関係、`enqueuedAt` の値、REBASE の `mergeCommit`、権限)は確かめていない。計画の Non-goals と register の 536 行のとおり
- 2 つの merge が同時に待つ AC15 は、テストの順序づけた fake で見るだけで、別のプロセスでの競合は見ていない

## Verdict

- Verdict: pass
- Verified: 静的検査はすべて通った(gofmt、go vet、golangci-lint、staticcheck、同期の検査、secret scan)。計画の 13 の受け入れ条件と、仕様の FR-1〜FR-11、FR-5b、NFR-1〜NFR-4 は、コードの上で満たされ、各条件にテストがある。記録は MERGED を gate の head と既定のブランチで読んだときだけで、manifest の lock の中で 1 回だけ書く。待ちの間 `regate.lock` を離し、片付けの前に取り直して marker を空にする。番号は `-F`、文字列は `-f`、node ID と SHA は形を確かめてから渡す。FR-5b の幅は 5 分で、境目の文とテストが揃っている。終了コード 7 は `ErrMergeQueued` から出る。2 つの読みの query は github.com の schema で通った
- Partially verified: AC3 と NFR-1 は、queue のない道のコードとテストの期待が変わらないことを読んで確かめたが、テストの実行は `/test`。help の固定の文字列 3 つの書き換えは計画に書かれていない(注)。文書は D1〜D6 のずれが残る(LOW 以下、`/sync-docs` へ)
- Not verified: mutation の schema と GitHub の queue の実際の挙動、GHES の schema、別プロセスでの記録の競合
