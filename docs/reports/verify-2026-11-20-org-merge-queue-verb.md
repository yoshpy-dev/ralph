# Verify report: org-merge-queue-verb

- Date: 2026-11-20(Run 2。パイプラインの 2 回目の run で、`cycle-count.json` は `cycle: 2`。既定の上限 2 の最後の run。Run 1 は cycle 1 の `d7ffc12a` で pass で、下に全文を残す)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`。`./scripts/plan-visual.sh digest` の値 `d584e5cd388b` は、計画の `- Approved:` の行と一致する(`2c9a7fe6` が D7 と Scope の文を直して取り直した値。Run 1 のときは `aef9c99a6ea5`)
- Spec: `docs/specs/2026-11-19-org-merge-queue.md`(2 本のうち 1 本目: FR-1〜FR-11、FR-5b、NFR-1〜NFR-4。FR-12、FR-13、AC7、AC9 の skill の部分、AC11 は 2 本目で、計画の Non-goals に入る)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `d919479a`、HEAD `ad178695`、45 ファイル、+5760 −305)。手元の `main` と `origin/main` はどちらも `d919479a`(fetch はしていない)。branch 全体の spec 適合と、Run 1 のあとの変更の文書のずれを見た。Run 1 のあとで Go の挙動を変えたのは `2c9a7fe6`(`queueMergeRecorded`)だけで、ほかは comment、help の文、テストの期待、文書の変更
- Evidence: `docs/evidence/verify-2026-11-20-org-merge-queue-verb.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。Run 1 の内容のあとに Run 2 の節を追記した。静的検査の出力、HEAD の binary の help、行数、register の Trigger の走査が入っている。ラッパー自身のログは `docs/evidence/verify-2026-11-20-045933.log`

## Run 2(cycle 2 of 2、HEAD `ad178695`)

### Run 1 のあとの commit

| Commit | 中身 | verify で見たこと |
| --- | --- | --- |
| `9800f0a9`、`39a52510` | test の report と event(cycle 1、pass) | report の AC15 の行は `2c9a7fe6` の前の形(R2-4) |
| `7469393c`、`5d35d67c` | sync-docs: Run 1 の D1〜D8 の直しと、test の G1〜G4 の register の行(538 行) | 下の「Run 1 の D の行き先」 |
| `193463e6` | cross-review の triage(ACTION_REQUIRED 1 件) | 直しの範囲 |
| `2c9a7fe6` | `queueMergeRecorded` が ledger 全体から同じ branch・PR・head の `pr_merged` を探す。Note の文、AC15 のテスト、cli の fixture、計画の D7 と Scope も直した | AC15、文書、Trigger |
| `f4f99555`、`e76960cf` | run 2 の self-review(LOW 4 件)と event | L12〜L14 の扱い |
| `ad178695` | L12(`mergeQueueRemovedAfterGate` の doc を「less than that much」に)と L13(doc の行の詰め直し)、計画の Progress | comment だけ。L14 は `/sync-docs` へ |

`git diff 435ecfe9 HEAD -- '*.go' ':!*_test.go'` から comment と文字列の行を除くと、残るのは `recordQueueMerge` の Note の `Sprintf` の行と、`queueMergeRecorded` から `collectMergeTargets` の境目を消した行だけだった。help の変更は `internal/cli/org_merge.go` の文字列(Run 1 の D6 の直し)だけ。

### Spec compliance

テストは走らせていない(`/test` の担当)。コードを読み、各 AC にテストがあることまでを確かめた。下の Run 1 の表は HEAD `435ecfe9` での判定で、`2c9a7fe6` が変えた AC15 のほかは、その道のコードは comment しか変わっていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC15: lock を離して待つ 2 つの merge が同じ MERGED を見ても、`pr_merged` は 1 行だけ | Met | `queueMergeRecorded`(`internal/org/merge_queue_wait.go:341-354`)は ledger の events 全体を見て、org-level で dry run でない `pr_merged` のうち、`branch=`(先頭の token)、`head=`、`pr=` が合うものがあれば true を返す。前の「最新の gate の記録より後」の境目(`collectMergeTargets` の `gate.pos`)はなくなった。`recordQueueMerge`(`:307-330`)は manifest の lock の中で ledger を読んでこれを呼び、true なら書かずに `Recorded` と Note を立てる。`TestMerge_QueueWaitRecordsAMergeOnlyOnce`(`internal/org/merge_queue_wait_test.go:607`)は 7 つの subtest を持つ。書かないのは 3 つ(待ちの間にほかの merge が書いた、そのあとに gate の記録が続いた、最新の gate の記録より前にある)、書くのは 4 つ(別の head、別の PR、別の branch、dry run)。境目を戻すと、書かない側の 2 つ目と 3 つ目が落ちる(読んで確かめた) |
| AC1、AC2、AC4、AC5、AC8、AC10、AC12、AC13、AC16 | Met | Run 1 のとおり。`435ecfe9` のあとにこれらの道で変わったのは `mergeQueueRemovedAfterGate` の doc の 1 語(`ad178695`、L12)だけで、新しい doc はコード `removedAt.After(gateAt.Add(-mergeQueueClockSkew))`(`internal/org/merge_queue.go:458-460`)と定数の doc(`:49-53`)に合う。PR の読みの query と mutation の文字列は変わっていないので、Run 1 の schema の probe の結果はそのまま当たる |
| AC3: queue のない repo では今の merge のテストがすべて変わらずに通る | Met(注つき) | Run 1 のとおり。`recordMerge` と queue のない道は `2c9a7fe6` でも変わらない(D7 は queue の道だけ)。sync-docs が `TestOrgMerge_Help` の固定の文字列 1 つ(lock の文)を延ばしたのは help の文の変更で、挙動の変更ではない |
| AC6 | Met | Run 1 のとおり。`mergeenforce.go` は Run 1 のあと変わっていない |
| AC14: README と `--help` が queue の道、`--requeue`、終了コード 7 を書き、tech-debt に確かめていない前提と実地の確かめの予定を書く | Met | Run 1 の注(D1)は `7469393c` で直った。HEAD の binary の help は、queue の待ちで lock を外すこと(231 行)、ほかの merge が記録済みのときの NOTE(103-104 行)、終了コード 7 と打ち直しの条件(114 行)、`--requeue` の段落(127 行)と flag の文を出す |

仕様の要件(FR-1〜FR-11、FR-5b、NFR-1〜NFR-4)は Run 1 の表のとおり満たす。FR-8 の `pr_merged` は、`2c9a7fe6` によって、待ちの間に gate か regate が gate の記録を足しても 1 行だけになった。NFR-3(分からない状態では記録しない)は変わらない。この直しで増えたのは記録しない枝だけで、記録する枝は増えていない。

計画の文は `2c9a7fe6` と合う。Scope(29 行)と D7(77 行)は「ledger 全体から探す」と書く。「最新の gate の記録より後」が残るのは D4、AC10、Progress の経緯の文で、どれも FR-5b か直しの説明になっている。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | Pass(rc=0) | shellcheck と `sh -n`(hook と verify のスクリプト)、guard の awk、`jq -e`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の verifier(選ばれた言語は golang だけ。`gofmt` は ok、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`d919479a..ad178695` は clean) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-org-merge-queue-verb.md` | `d584e5cd388b` | 計画の `- Approved:` と一致 |
| `go build -o $TMPDIR/omqv2/ralph ./cmd/ralph` と `org merge --help` | rc=0 | 下の Observational checks |
| comment の行の長さ(`merge_queue.go`、`merge_queue_run.go`、`merge_queue_wait.go`、`merge_run.go`) | 100 文字を超える comment の行はない | L13 の直しの確かめ |

### Documentation drift

`2c9a7fe6` の文の側は揃っている。help の「one when another merge that waited for the same queue had recorded it already」、README の 296 行の「records `pr_merged` once」、AGENTS.md の `internal/org/` の行の「the one `pr_merged` with `via=merge_queue`」は、もともと境目を書いていないので直す必要がない。repo 全体で `latest gate record` を grep した結果、D7 の意味で残る文はない。残りは FR-5b、merge-order、regate、gate accept の文と、AC15 のテストの subtest の名前だけだった。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| R2-1(self-review の L14): register の 537 行と 538 行 | No(LOW、既知、`/sync-docs` へ) | 537 行の Trigger「The next change to either test file」に、`2c9a7fe6` の `merge_queue_wait_test.go` の書き直し(854 → 874 行)が当たるが、行の注は `5e409db0` のことしか書いていない。行の数(`merge_queue_wait.go` 390、`merge_queue_wait_test.go` 854)は「every count of this row is `wc -l` at 435ecfe9」と commit に結んであるので偽ではない(HEAD では 386 と 874)。538 行の Trigger「The next change to `recordQueueMerge`」には、`2c9a7fe6` の Note の文と doc の変更(`ad178695` の doc の詰め直しも)が当たる。この行にも注がなく、G1 のテストも足されていない |
| R2-2: 仕様のチェックボックス | No(LOW、新、Run 1 は見落とした) | `docs/specs/2026-11-19-org-merge-queue.md` の FR、NFR、AC はすべて `[ ]` のまま。2 本に分けたほかの仕様では、1 本目の PR が自分の分に印を付けている(`a40c646f` は `docs/specs/2026-11-12-org-reseat.md` の FR-1〜FR-4 ほかに、`244c5144` は org-herdr-autostart-panes の part A に付けた)。この PR が満たすのは FR-1〜FR-11、FR-5b、NFR-1〜NFR-4、AC1〜AC6、AC8、AC10 で、FR-12、FR-13、AC7、AC11 は 2 本目。AC9 は README と help の部分がこの PR、skill の部分が 2 本目なので、印を付けるなら注が要る |
| R2-3: register の 502 行の 2026-11-20 の注 | INFO | Trigger「a gate, regate, or merge test of `internal/cli`」には、`2c9a7fe6` による `TestOrgMerge_PrintsAMergeQueueMergeAsMerged` の期待の変更(`internal/cli/org_merge_queue_test.go`、348 行のまま)も当たる。ただし注はこのファイルをこの PR の分として名指していて、「gate も probe も走らせない」は今も正しいので、直さなくてよい。R2-1 の注を書くときに `2c9a7fe6` を並べてもよい |
| R2-4: test の report の AC15 の行 | INFO(`/test` へ) | `docs/reports/test-2026-11-20-org-merge-queue-verb.md` の 58 行は「6 subtest」と「gate より前 … は数えない」を書く。`2c9a7fe6` の前の形で、今は 7 subtest になり、gate の記録より前の `pr_merged` も記録済みとして数える。`/test` の run 2 が report を書き直せば直る |
| R2-5: この report の Run 1 の AC15 の行 | INFO | 下の Run 1 の AC15 の行は境目のある形を書く。この Run 2 の AC15 の行に置き換わる |
| 計画の Scope、D7、Progress | Yes | Spec compliance の最後の段落 |
| Go の doc comment(`queueMergeRecorded`、`recordQueueMerge`、`MergeResult.Recorded`、`mergeQueueRemovedAfterGate`) | Yes | ledger 全体を見ることと、5 分の幅が「less than」であることを書く。`mergeQueueRemovedAfterGate` の doc は、定数の doc(`merge_queue.go:49-53`)と `gateWhen` の文に揃った |
| skill の 4 写し | Yes | `check-skill-sync.sh` と `check-sync.sh` が通った。`2c9a7fe6` と `ad178695` は skill を変えていない |

Run 1 の D の行き先(`7469393c` で直った。HEAD で確かめた):

| ID | 中身 | HEAD で |
| --- | --- | --- |
| D1 | README と skill の「打ち直せば待ちを続ける」に条件がない、regate した PR の道が書かれていない | 解消。README の 296 行、org skill の `merge` の行(396 行)と 2002-2004 行、director skill の 1742-1746 行が「origin の既定のブランチが gate のときのまま」の条件を持ち、4 写しは同じ。regate した PR の道は README の 296 行と register の 536 行が書く。`requeueNext` の文はこの道を書かないが、run 2 の self-review はこれを README と register が引き受けたと判断した(L2 の行) |
| D2 | register の 537 行のテストの名前と行数 | 解消。行数は `435ecfe9` に結んだ。そのあとのずれは R2-1 |
| D3 | register の 395 行と 502 行の固定の文字列の数と行数 | 解消(three fragments、1250 と 1251、348) |
| D4 | register の 182 行(`/org` skill の命令の表)の持ち越しの注 | 解消。注の数(`merge` の行は 5,285 → 5,939 文字、`stop` は 1,208、`disband` は 1,219)は HEAD の表の行の長さと一致した |
| D5 | register の 534 行の `merge_test.go` の行数 | 解消(833) |
| D6 | merge の間の `regate.lock` の文 | 解消。help(`internal/cli/org_merge.go` の 283-288 行)、org skill の 2150-2154 行、`TestOrgMerge_Help` の固定の文字列が直った。コード(`mergeLocked` の `defer lock.release()`、`mergeThroughQueue`、`queueMergeCleanup`)と合う |
| D7 | 仕様の Open questions の終了コード | 解消(「決まった(2026-11-20): 7」) |
| D8 | 計画の受け入れ条件のチェックボックス | 解消(すべて `[x]`) |

register の Trigger の走査では、各行の 4 つ目の欄を python で ` | ` で分けて読み、Run 1 のあとに変わったファイル、関数、skill の節の名前で探した。

- 当たって注があるもの: 182 行(sync-docs による `merge` の行の書き換え)、395 行(sync-docs による `org_merge_test.go` の変更、1251 行)
- 当たって注がないもの: 537 行と 538 行(R2-1)。502 行は、注が同じ PR の分としてそのファイルを名指している(R2-3)
- 当たらないと判断したもの: 386 行(「the exit-code part of the `/director` skill」は wait の終了コード 4 と 5 の部分で、今回の書き換えは「merge と auto_merge」の節にある)、267 行と 385 行(受信箱と send の確かめの節は変わっていない)、252 行(`queueMergeRecorded` は `collectMergeTargets` を呼ばなくなったが、`mergeorder.go` は変わっていない)、424 行(`recordedMergeHead` は呼ばれるだけで変わっていない)、213 行、464 行、508 行(`PrintableInboxText` も呼ばれるだけ)、161 行、382 行、406 行、513 行(該当の節は変わっていない)

### Observational checks

HEAD(`ad178695`)から作った binary の `ralph org merge --help` を grep した。queue の待ちの間は `regate.lock` を外し、片付けのときだけ取り直すという文(231 行)、ほかの merge が記録済みのときの NOTE の文(103-104 行)、終了コード 7 と打ち直しの条件(114 行)、`--requeue` の段落(127 行)と flag の文(251 行)が出る。

GitHub の GraphQL の schema の probe は打ち直していない。2 つの query の文字列は `435ecfe9` から変わっていない(`merge_queue.go` の変更は doc の 2 行だけ)ので、Run 1 の結果がそのまま当たる。

### Coverage gaps

- テストは走らせていない。AC15 の 7 つの subtest があること、境目を戻すと落ちる形になっていることは、読んで確かめた。通るかは `/test` が確かめる
- `queueMergeRecorded` は座席の `pr_merged`(`ev.SeatID != ""`)を数えないが、それを確かめる subtest はない(Run 1 からある穴。dry run の除外には subtest がある)
- `enqueuePullRequest` の mutation、GitHub の queue の実際の挙動、GHES の schema は確かめていない(Run 1 と同じ。register の 536 行)
- 2 つの merge が同時に待つ AC15 は、テストの順序づけた fake で見ただけで、別のプロセスでの競合は見ていない

## Verdict

- Verdict: pass
- Verified: 静的検査はすべて通った(gofmt、go vet、golangci-lint、staticcheck、同期の検査、計画の参照、secret scan)。計画の 13 の受け入れ条件と、仕様の FR-1〜FR-11、FR-5b、NFR-1〜NFR-4 はコードの上で満たされ、各条件にテストがある。cross-review の ACTION_REQUIRED は `2c9a7fe6` で閉じた。queue の道の記録の重複判定は ledger 全体を見るので、待ちの間に gate か regate が gate の記録を足しても `pr_merged` は 1 行だけになり、そのテストの行もある。self-review の run 2 の L12 と L13 は `ad178695` で、Run 1 の D1〜D8 は `7469393c` で直った
- Partially verified: 文書。R2-1(self-review の L14、register の 537 行と 538 行)と R2-2(仕様のチェックボックス)が `/sync-docs` に残る。どちらも LOW で、verdict は変えない。R2-3〜R2-5 は INFO
- Not verified: テストの合否(`/test`)、mutation の schema と GitHub の queue の実際の挙動、GHES の schema、別プロセスでの記録の競合
- Insight event: この report の commit のあとに `./scripts/insights-append.sh --slug org-merge-queue-verb --flow standard --phase verify --verdict pass --cycle auto --source skill --low 2` を打ち、events のファイルだけを別の commit にする。`--low 2` は R2-1 と R2-2 の数

## Run 1(cycle 1、HEAD `435ecfe9`)

下は Run 1 の report の全文で、見出しを 1 段下げた。行番号は `435ecfe9` のもの。AC15 の行は `2c9a7fe6` の前の境目を書いていて、Run 2 の AC15 の行に置き換わる(R2-5)。D1〜D8 の行き先は Run 2 の Documentation drift にある。

- Date: 2026-11-20(パイプラインの 1 回目の run。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`。`./scripts/plan-visual.sh digest` の値 `aef9c99a6ea5` は、計画の `- Approved:` の行と一致する
- Spec: `docs/specs/2026-11-19-org-merge-queue.md`(2 本のうち 1 本目: FR-1〜FR-11、FR-5b、NFR-1〜NFR-4)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `d919479a`、HEAD `435ecfe9`、40 ファイル、+5213 −278)。手元の `main` と `origin/main` はどちらも `d919479a` で、branch は main の先端の上にある(fetch はしていない)。self-review の 2 回目(`bbbf64e7`)のあとの `435ecfe9` は、L9(FR-5b の断りの文を「less than 5m0s before it」に)と L10(help の詰め直し)を直し、`--wait-checks` の flag の文に merge queue を足した
- Evidence: `docs/evidence/verify-2026-11-20-org-merge-queue-verb.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。静的検査の出力、GitHub の GraphQL の schema の読みだけの probe、HEAD から作った binary の `ralph org merge --help` が入っている。ラッパー自身のログは `docs/evidence/verify-2026-11-19-233220.log`

### Spec compliance

テストは走らせていない(`/test` の担当)。判定はコードを読んだ結果で、各 AC に当たるテストがあることまでを確かめた。テストが通るかは `/test` が確かめる。

#### 計画の受け入れ条件

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

#### 仕様の要件(1 本目の分)

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

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | Pass(rc=0) | shellcheck と `sh -n`(hook と verify のスクリプト)、`check-sync.sh`(DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の verifier(`gofmt -l` は空、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`d919479a..435ecfe9` は clean)。言語は golang だけが選ばれた |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-org-merge-queue-verb.md` | `aef9c99a6ea5` | 計画の `- Approved:` と一致 |
| `go build -o $TMPDIR/ralph-omq ./cmd/ralph` と `org merge --help` | rc=0 | help の文は下の Observational checks |

### Documentation drift

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

### Observational checks

GitHub の GraphQL の schema に、コードの 2 つの query を読みだけで通した(`gh api --hostname github.com graphql`、アカウント `yoshpy-dev`、2026-11-20)。query の文字列は Go の定数から python で取り出した。mutation(`enqueuePullRequest`)は送っていない。

- `mergeQueuePRQuery` を `-F number=351` で(この repo の PR #351): rc=0。答えは `id`(`PR_kwDO...`、`mergeQueueNodeIDPattern` に合う形)、`state: MERGED`、`headRefOid`、`baseRefName: main`、`mergeCommit.oid`、`isInMergeQueue: false`、`mergeQueueEntry: null`、`timelineItems.nodes: []`。`parseMergeQueuePR` が読む欄の名前と形に合う
- 同じ query を `-f number=351` で(負の対照): rc=1、`Variable $number of type Int! was provided invalid value`。計画助言の HIGH(番号を `-f` で送ると型エラー)と、`-F` にした直しの理由を本物で確かめた
- `mergeQueueQuery`(step 11、`configuration { mergeMethod }` つき)を `branch=main` で: rc=0、`{"data":{"repository":{"mergeQueue":null}}}`。queue のない repo で step 11 の query が schema で通る(NFR-1)

HEAD から作った binary の `ralph org merge --help` は、queue の道の段落、`--requeue` の段落(「less than 5 minutes before it」)、終了コード 7、`--wait-checks` と `--requeue` の flag の文を出す。L10 の短い行(`exits 0. When`)はなくなった。

### Coverage gaps

- テストは走らせていない。AC ごとのテストがあることは読んで確かめたが、通るかは `/test` が確かめる
- `enqueuePullRequest` の mutation は本物に送っていない(副作用のない確かめ方がない)。入力の型(`ID!`、`GitObjectID!`)と答えの `mergeQueueEntry { state position enqueuedAt }` は schema で確かめていない。register の 536 行のとおり、release のあと下流で確かめる
- GHES の schema は見ていない。`MergeQueue.configuration` がない版の GHES では、step 11 の query が検証で落ち、queue のない repo の merge も断るはず(推測で、未確認)。github.com では通る
- queue のある repo での実際の挙動(queue から外す時刻と timeline の関係、`enqueuedAt` の値、REBASE の `mergeCommit`、権限)は確かめていない。計画の Non-goals と register の 536 行のとおり
- 2 つの merge が同時に待つ AC15 は、テストの順序づけた fake で見るだけで、別のプロセスでの競合は見ていない

### Verdict

- Verdict: pass
- Verified: 静的検査はすべて通った(gofmt、go vet、golangci-lint、staticcheck、同期の検査、secret scan)。計画の 13 の受け入れ条件と、仕様の FR-1〜FR-11、FR-5b、NFR-1〜NFR-4 は、コードの上で満たされ、各条件にテストがある。記録は MERGED を gate の head と既定のブランチで読んだときだけで、manifest の lock の中で 1 回だけ書く。待ちの間 `regate.lock` を離し、片付けの前に取り直して marker を空にする。番号は `-F`、文字列は `-f`、node ID と SHA は形を確かめてから渡す。FR-5b の幅は 5 分で、境目の文とテストが揃っている。終了コード 7 は `ErrMergeQueued` から出る。2 つの読みの query は github.com の schema で通った
- Partially verified: AC3 と NFR-1 は、queue のない道のコードとテストの期待が変わらないことを読んで確かめたが、テストの実行は `/test`。help の固定の文字列 3 つの書き換えは計画に書かれていない(注)。文書は D1〜D6 のずれが残る(LOW 以下、`/sync-docs` へ)
- Not verified: mutation の schema と GitHub の queue の実際の挙動、GHES の schema、別プロセスでの記録の競合
