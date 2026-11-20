# Self-review report: org-merge-queue-verb

- Date: 2026-11-20(Run 2)。Run 1 も 2026-11-20(下に全文を残す)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`(issue #342、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の 1 本目)
- Branch: `feat/org-merge-queue-verb`、HEAD `2c9a7fe6`、base は `main` の `d919479a`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2、既定の上限 2 の最後の実行)
- Scope: `git diff main...HEAD`(45 ファイル +5688 −305)の diff の質だけ。中心は cross-review の ACTION_REQUIRED の直し `2c9a7fe6`(`queueMergeRecorded` が ledger 全体から探す)と、Run 1 の HEAD `49a88cac` からの差(`435ecfe9` の L9、L10 の直し、`7469393c` の sync-docs)。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)
- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 4 件: Run 1 から残る L8 と、新しい L12 から L14)

## Run 2(cycle 2 of 2、HEAD `2c9a7fe6`)

### Evidence reviewed

- `2c9a7fe6` の差分すべて(5 ファイル: `merge_queue_wait.go`、`merge_run.go`、`merge_queue_wait_test.go`、`internal/cli/org_merge_queue_test.go`、計画)と、Run 1 の HEAD からの差 `git diff 49a88cac..HEAD`(Go、テスト、register、spec、skill、README)
- 枝全体の Go のコードを、Run 1 が通して読んでいない部分まで今回の目で読み直した: `merge_queue_wait.go` 全 385 行、`merge_queue.go` の 296〜489 行、`merge.go`、`merge_run.go`、`mergeenforce.go` の差分すべて、`cleanup.go`、`lockfile.go`、`regate.go`、`seat.go`、`spawn.go`、`merge_wait.go`、`exitcode.go`、`cmd/ralph/main.go` の差分、`internal/cli/org_merge.go` の 150〜300 行(help)と 360〜569 行。`merge_queue_run.go` と `merge_queue.go` の 1〜295 行は Run 1 が通して読んだもので、今回は `2c9a7fe6` 以降の差(`gateWhen` の 1 語)と `requeueNext` の周りだけ読んだ
- `2c9a7fe6` の影響範囲の grep: `after the branch's latest gate record`(D7 の意味で残る文はない。残るのは FR-5b の removal の話だけ)、`gate.pos` と `collectMergeTargets`(`mergeorder.go` が使うので dead にならない)、`slices`、`strconv`、`strings`(`merge_queue_wait.go` の 118、341、348 行などで使われ、import は余らない)
- Run 1 の L9、L10 の直しの網羅: `up to 5m0s`、`up to %s`、`up to that much`、`counts as after it` を Go、spec、skill、README で grep。help の Long の短い行は awk で探し、段落の終わりの行しかないことを確かめた
- skill の `org` と `director` を `.claude`、`.agents`、`templates/base` の 4 か所で `cmp`: すべて同じ。「打ち直せば待ちを続ける」の 3 文(`org` の 396 行と 2001 行、`director` の 1738 行)はすべて「origin の既定のブランチが gate のときのままなら」の条件つき
- register: `docs/tech-debt/README.md` の 536〜538 行(merge queue の 3 行)を `2c9a7fe6` と突き合わせた
- サイズ(今、`wc -l`): `merge.go` 826、`merge_test.go` 833、`merge_queue_test.go` 807、`merge_queue_wait_test.go` 874、`merge_queue.go` 489、`merge_queue_run.go` 333、`merge_queue_wait.go` 385、`merge_queue_run_test.go` 647、`merge_queue_path_test.go` 153、`internal/cli/org_merge_queue_test.go` 348

### Run 1 の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| L2(残り) | 直った(文書の側) | skill の 3 文と README の 1 文が条件つきになった。queue にある PR を regate した道は README(296 行)が 1 回書き(local の regate は push の前に記録するので、次の merge は FR-5b で断り、そのとき `--requeue` が次の手)、register の前提の行(536 行)にも入った。Go の `requeueNext` の文は「直しを push して regate、揺れた検査のあとは `--requeue`」のままで、regate の push で外された道にはあてはまらないが、その道は README と register が引き受けている |
| L8 | 残す(意図して) | `readMergeQueue` と `mergeQueueGraphQLErrors` の `errors: []` の扱いの違い。計画が NFR-1 を理由に変えないと決めたもので、2 本目で `mergeenforce.go` を変えるときに寄せる |
| L9 | 一部残る(L12) | `435ecfe9` が `gateWhen` を「less than %s」にし、3 つのテストの文字列を合わせた。help、`--requeue` の usage、README は元から「less than 5 minutes」。同じ規則を言う 3 つ目の文が残った(L12) |
| L10 | 直った | help の Long を詰め直し、`exits 0. When --wait-checks leaves no time once the PR is in the queue` が 1 行にまとまった。120〜300 行に短い行は段落の終わりしかない |
| L11 | 直った(L14 が残る) | register の 800 行の行は、改名されたテストの名前、`435ecfe9` に結びついた行数、`5e409db0` が trigger の文字を満たしたが払っていないという注記を持つ。`2c9a7fe6` が同じ trigger をもう一度満たした(L14) |

### Findings

CRITICAL、HIGH、MEDIUM はない。`2c9a7fe6` の変更(ledger 全体から同じ branch・PR・head の `pr_merged` を探す)は、記録、lock、escape の正しさに関わる誤りを持たない。新しい指摘は LOW 3 件で、どれも文と記録の側にある。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | L12(新、L9 の残り)。`mergeQueueRemovedAfterGate` の doc が、1 つの doc の中で境目を 2 通りに言っている。1 文目は「removedAt is strictly after gateAt minus mergeQueueClockSkew」、2 文目は「A removal up to that much before the gate counts as after it」。コードは `removedAt.After(gateAt.Add(-skew))` で、ちょうど 5m0s 前は数えず、定数 `mergeQueueClockSkew` の doc(`:51`)も「less than this before the gate」と書く。英語の「up to」は 5m0s を含むので、2 文目だけが偽になる。`435ecfe9` は `gateWhen` の文と 3 つのテストの文字列だけを直し、同じ規則を言うこの文を見落とした | `internal/org/merge_queue.go:452-457`(2 文目は `:455-456`)。コードは `:458-460`。定数の doc は `:49-53`。直した側は `merge_queue_run.go:216`、テストの doc は `merge_queue_test.go:711-716`(「one at the bound or further before does not」) | `:455-456` を「A removal less than that much before the gate counts as after it」に直す。コードとテストは変えない |
| LOW | readability | L13(新、`2c9a7fe6` で入った)。直したとき doc の行を詰め直さなかった。`queueMergeRecorded` の doc の 5 行目 `// (cross-review of plan ...). Its Details are read as FollowupRecordedMerge reads them, branch= from the first` が 139 文字で、3 つのファイル(`merge_queue.go`、`merge_queue_run.go`、`merge_queue_wait.go`)の 100 文字を超える doc の行はこの 1 行だけ。ほかの doc の行は 82 文字以下。`recordQueueMerge` の doc の 5 行目 `// appends nothing and sets Note.` は 33 文字で、次の行が `Either way it sets Recorded.` の新しい文から始まる。Run 1 の L10 と同じ形(文を足したが、行を詰め直さなかった) | `internal/org/merge_queue_wait.go:337`(139 文字)と `:304`(33 文字)。`git show 2c9a7fe6 -- internal/org/merge_queue_wait.go` で、足された doc の行はこの 1 行だけが長い | 2 つの doc を 80 文字前後で詰め直す。`:337` の「(cross-review of plan ...)」は次の行へ送る。コードは変わらない |
| LOW | maintainability | L14(新)。register の 2 つの行の trigger が `2c9a7fe6` で文字どおりに満たされたが、行に書かれていない。(1) 537 行(800 行の目安)の trigger「The next change to either test file」: `merge_queue_wait_test.go` が 854 行から 874 行になった(AC15 のテストの書き直し)。行の carry-over の注記は `5e409db0` のことだけを言う。(2) 538 行(G1 から G4)の trigger「The next change to `recordQueueMerge`, ...」: `recordQueueMerge` の Note の文と doc が変わった。G1(`recordQueueMerge` の記録の失敗を読むテスト)は足されていない。この直しが pipeline を回し直すので、seam を作って G1 を足す機会でもあった。どちらも、行の pin(`435ecfe9` の行数)が古くなったわけではなく、trigger を読む人に「もう満たされて払われなかった」ことが伝わらない点が問題。Run 1 の L11 の(2)と同じ形 | `docs/tech-debt/README.md:537`(`Carried over to the next change that meets this trigger` の前の注記は `5e409db0` だけ)と `:538`(trigger 欄)。`git show 2c9a7fe6 --stat` に `merge_queue_wait_test.go` と `merge_queue_wait.go` | 537 行に「2c9a7fe6(cross-review の直し)も同じ test file を変えたが、移しは直しの Scope の外」を足し、538 行に「`recordQueueMerge` の Note の文を変えた 2c9a7fe6 が trigger を満たしたが G1 は足していない」を足す。または trigger を「この PR の merge のあとの次の変更」と言い直す |

### Positive notes

- `2c9a7fe6` の直しは小さく、誤りの根を直している。PR の merge は 1 回なので、境目(最新の gate の記録)をなくして ledger 全体を見る形は、境目を持ち続ける案より単純で、`collectMergeTargets` への依存が `queueMergeRecorded` から消えた。`ev.Event`、`ev.DryRun`、`ev.SeatID` の 3 つの除外と、branch・head・pr の 3 つの一致は前のまま
- 文の側が一緒に直っている: `queueMergeRecorded` と `recordQueueMerge` の doc、`MergeResult.Recorded` の doc、`Note` の文(「after the branch's latest gate record」を外した)、`internal/cli/org_merge_queue_test.go` の期待、計画の Scope と D7。help(`org_merge.go:155-157`)は元から ledger のどこにあるかを言わず、そのまま合う。D7 の意味で古い文は grep に残らない
- AC15 のテストは、新しい方向(gate の記録が他の merge の記録のあとに続く)を足し、旧い負の行「最新の gate の記録より前」を消さずに、記録済みの期待へ移した。残る負の行(別の head、PR、branch、dry run)は `mxReadyLedger()` の上に置かれ、gate の記録との位置関係に依存しない
- `435ecfe9` の help の詰め直しは L10 を直し、新しい文(「On the merge queue path, merge releases regate.lock when the wait for the queue starts ...」)の行も詰め方がそろっている。`--wait-checks` の usage と help のテストの期待も一緒に更新された
- Run 1 で気づいた `requeueNext` の doc(push のない regate を挙げない理由)は変わらず、README と register がその残りの道を 1 回ずつ書いた。skill の 3 文は 4 か所の写しが `cmp` で同じ

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (新しい行は足さない) | — | L12 と L13 は Go の comment の直しで、この PR の中で直せる。L14 は既存の 2 つの行への 1 文ずつ。L8 は計画が NFR-1 を理由に変えないと決めたもので、先送りではない | — | この report |

### Coverage gaps

- テスト、静的解析、`check-sync.sh`、コンパイルは走らせていない(`/verify` と `/test` の担当)。`2c9a7fe6` の import の使われ方は目で確かめただけ
- `merge_queue_run.go` の 1〜175 行と 212〜333 行、`merge_queue.go` の 1〜295 行、テストファイルの大半は、Run 1 が読んだ内容から変わっていないので今回は読み直していない(変わった行だけ読んだ)
- GitHub の実際の挙動(head の変わった PR を queue から外すか、`enqueuedAt` と timeline の `createdAt` の関係、時刻が秒までか)は確かめていない。register の前提の行(536 行)が扱う
- この report は 2 回目で、パイプラインの上限(既定の 2)の最後の実行。L12 と L13 は Go の comment、L14 は `docs/tech-debt/README.md` で、記録だけの直しの例外(`docs/reports/` と insight event のみ)の外にあるので、直すには全 pipeline の回し直し(上限を上げる)が要る。上げずに `/pr` へ進むなら、3 件は既知の穴として PR の本文に書く

### Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 4 件で、記録、lock、escape の正しさに関わるものはない)
- Follow-ups: L12 は doc の 1 語(`merge_queue.go:455-456`)。L13 は doc の 2 段落の詰め直し(`merge_queue_wait.go:304-305`、`:337`)。L14 は register の 2 行への 1 文ずつ。L8 は 2 本目まで残す。3 件は 1 つの小さい commit で足りるが、上限のため回し直しの要否は operator が決める

## Run 1(cycle 1、HEAD `d53363fa` と `49a88cac`)

下は Run 1 の report の全文で、見出しを 1 段下げてある。1 回目の report(HEAD `d53363fa`)は LOW 8 件(L1〜L8)で、直しの `5e409db0` のあと、この 2 回目の実行(HEAD `49a88cac`)は LOW 5 件(L2 の残り、L8、L9、L10、L11)だった。

- Date: 2026-11-20(cycle 1、self-review の直しのあとの 2 回目の実行)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`
- Branch: `feat/org-merge-queue-verb`、HEAD `49a88cac`、base は `main` の `d919479a`(issue #342、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の 1 本目)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff d919479a...HEAD` の 40 ファイル(+5199 −275)の diff の質だけ。前回の report(`d28f6532`、HEAD `d53363fa` で LOW 8)のあとに入った `5e409db0`(L1〜L7 の直し、12 ファイル +188 −163)を中心に見た。`49a88cac` は計画の記録だけなので、Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)

### Evidence reviewed

- `5e409db0` の差分すべて(`git show 5e409db0`)。コードは `merge_queue.go`(query、`mergeQueuePR`、`mergeQueueTimeline`、`latestMergeQueueRemovalAfter`)、`merge_queue_run.go`(`requeueNext`、`queueNext`)、`merge_queue_wait.go`(`queueLeft`)、`internal/cli/org_merge.go`(help、`ErrMergeQueued`、`printMergeLines`)、doc だけの `merge.go`(`mergeCheck`)、`seat.go`(`EventPRMerged`)、`internal/cli/exitcode.go`、`cmd/ralph/main.go`
- 通して読んだファイル: `merge_queue_run.go` 全 333 行、`merge_queue_wait.go` 全 390 行、`merge_queue.go` の 1〜310 行と 425〜480 行(query、型、検査、parse、removal の helper)
- 周辺(diff の外): regate の結果の doc(`regate.go:30-110`。`RegateOutcomeUpToDate` は「HEAD が既定のブランチの先端を含み、最新の gate の記録がその先端の上の HEAD の pass」のときだけで、push のあとの HEAD なら `RegateOutcomeGated` になる)、`mergeGateTime`(`merge_queue_run.go:36-52`)、step 8 の `m.gateAt` の書き込み(`merge.go:373`)と step 11 の読み(`:422`)、`printMergeResult` の Failed の枝(`org_merge.go:462-481`)
- テスト(読んだだけで、走らせていない): `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond` の表、`TestMergeQueue_RemovalAfterGateAllowsForClockSkew` の境界の表(`merge_queue_test.go:714-740`)、`TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue` の skew の行(`merge_queue_run_test.go:393-420`)、`TestOrgMerge_PrintsAMergeQueueMergeAsMerged` の NOTE の 4 行、`TestOrgMerge_HelpDescribesTheMergeQueuePath`
- 前回の指摘が名指した場所を 1 つずつ grep した: 「keep waiting」「待ちを続ける」(Go、skill、README)、`\.Added\b`、`ADDED_TO_MERGE_QUEUE_EVENT`、`AddedToMergeQueueEvent`、`fix what made the queue`、`get a new gate record`、`LatestRemovalAfterIsStrictlyAfter`
- register: `docs/tech-debt/README.md` の merge queue の 2 行(GitHub で確かめていない前提の行、800 行の目安の行 `:537`)を差分と突き合わせた
- サイズ(今): `merge.go` 826、`merge_queue_test.go` 807、`merge_queue_wait_test.go` 854、`merge_queue.go` 489、`merge_queue_run.go` 333、`merge_queue_run_test.go` 647、`internal/cli/org_merge.go` 569、`internal/cli/org_merge_queue_test.go` 348

### Findings

番号は前回の L1〜L8 を引き継ぎ、新しい指摘は L9 から振る。CRITICAL、HIGH、MEDIUM はない。残っている指摘は LOW 5 件(L2 の残り、L8、L9、L10、L11)。

#### 前回の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| L1 | 直った | `requeueNext`(`merge_queue_run.go:197-201`)は「feature worktree から branch に直しを push して `ralph org regate --branch <b>` で新しい head の gate を取る、揺れた検査のあとは `--requeue`」になった。push のあとの HEAD は既定のブランチの先端を含み、最新の gate の記録は前の head のものなので、regate は `RegateOutcomeGated` で記録する(`regate.go:46-58`)。push のない regate を挙げない理由も doc にある。FR-9 の文(`merge_queue_wait.go:246-248`)も同じ手を使い、前の「原因を直してから」と push の手が 1 文に重なる形はなくなった |
| L2 | 一部残る(下の表) | Go の 6 か所は条件を書いた: `queueNext`(`merge_queue_run.go:306-311`)、help の 2 か所(`org_merge.go:166-170`、`:277-278`)、`ErrMergeQueued`(`:393-394`)、`exitcode.go:31`、`main.go:42`。skill の 3 文と README の 1 文はまだ無条件だが、計画の S6 が `/sync-docs` に渡すと書いている。register に足す 1 文は入っていない |
| L3 | 直った | `latestMergeQueueRemovalAfter` は `!r.At.Before(after)`(`merge_queue.go:445`)。doc、`queueLeft` の doc、2 つのテストの期待(同じ秒は理由が出る、1 秒前は出ない)がそろっている。同じ秒の removal の文は「removed at T ... after it entered the queue at T」と読めるが、秒の粒度では偽ではない |
| L4 | 直った | query は `REMOVED_FROM_MERGE_QUEUE_EVENT` だけを頼み(`merge_queue.go:33-37`)、`Added` と枝は消えた。本番のコードとテストに `\.Added\b` は残っていない。テストは `absent` で add event がないことも固定し、返ってきた add event は「頼んでいない型」として読み失敗になる(`mergeQueueTimeline` の他の型と同じ扱い) |
| L5 | 直った | `printMergeLines` が `QueueNote` を `res.Note` の前に `NOTE:` として出す(`org_merge.go:494-496`)。escape は `QUEUED:` の側と同じで、`PrintableInboxText` は `\` を残すので 2 回かけても同じ。Failed の道は stderr の 1 文だけで `QueueNote` を出さないが、`MergeResult` の doc(「wait がどう終わっても欄は残る」)は欄の話で、表示の契約は `printMergeResult` の doc にある。ずれではない |
| L6 | 直った | `mergeCheck` の doc が「step 8 が書き、step 11 が読む」を言い、`gateAt` の doc は受け入れの時刻と ts が読めないときの 0 を書く。`mergeGateTime` と合う |
| L7 | 直った | `EventPRMerged` の doc に「or found in it at that head already (whoever added it)」が入った(`seat.go:147-152`) |
| L8 | 残す(意図して) | `readMergeQueue` の `errors: []` の扱いは変えていない。計画は queue のない repo の挙動を変えない NFR-1 を理由に挙げる。実害は見込みにくく、この判断でよい |

#### 残っている指摘と新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L2(残り)。前回の L2 の後半、queue にある PR を regate した道が手当てされていない。既定のブランチが動いて step 7 が断ったあと、人が regate を打つと、local の regate は記録してから push する。push で PR が queue から外されると(GitHub がそうするかは register の前提の行どおり確かめていない)、外された時刻が gate の記録より後になり、次の merge は FR-5b で断る。そのときの次の手 `requeueNext` は「直しを push して regate」か「揺れた検査のあとは `--requeue`」で、どちらもこの場合に合わない。regate を打ち直しても `RegateOutcomeUpToDate` で何も記録せず、同じ断りに戻る。正しい手は `--requeue`(queue はこの gate の head をまだ試していない)だが、文からは読み取りにくい。register の前提の行にもこの道は書かれていない | `merge_queue_run.go:197-201`(`requeueNext`)、`:181-186`(FR-5b の断り)。regate の順は `regate.go:63-66`(前回の report の証拠)。register の行 `docs/tech-debt/README.md` の「The merge queue path of `ralph org merge` (the first PR of #342) was built on a fake gh」 | `requeueNext` の `--requeue` の条件を「揺れた検査のあと、または regate の push で外されたときのように、queue がこの gate の head を試していないとき」に広げるか、register の前提の行に「queue にある PR を regate すると push で外され、次の merge は FR-5b で断る。そのときは `--requeue`」を 1 文足す。skill の 3 文(`.claude/skills/director/SKILL.md:1743`、`.claude/skills/org/SKILL.md:396`、`:2002`、それぞれ 4 写し)と README(`README.md:296`)の無条件の「打ち直せば待ちを続ける」は、計画の S6 のとおり `/sync-docs` で直す |
| LOW | exception-handling | L8(前回のまま、意図して残す)。step 11 の `readMergeQueue` は `"errors": []` を errors ありとして断り、`mergeQueueGraphQLErrors` は空の list を errors なしとして読む | `internal/org/mergeenforce.go:271`、`internal/org/merge_queue.go:209-223` | 計画の判断(NFR-1)どおり残す。2 本目で `mergeenforce.go` を変えるときに寄せる |
| LOW | maintainability | L9(新)。FR-5b の断りの文と、コードと help が、skew の境目で食い違う。`gateWhen` は「a removal up to 5m0s before it counts as after it」と書くが、`mergeQueueRemovedAfterGate` は gate の時刻の 5 分前より厳密に後だけを数え、ちょうど 5m0s 前の removal は数えない。テストもそれを固定している(「5m before the gate, the edge of the skew」は数えない)。help、`--requeue` の usage、README は「less than 5 minutes before it」で、コードと合う。英語の「up to 5m0s」はふつう 5m0s を含むので、同じ verb の 2 つの文が同じ規則を別の境目で言っている。秒の粒度の時刻なので、ちょうど 5m0s 前は実際に起こりうる。前回の差分(`12ace9df`/`b6c0ba3b`)からあったもので、L3 で FR-9 の比べ方を「同じ秒を含む」にしたため、隣の FR-5b の文の境目が目立つようになった | 文: `merge_queue_run.go:216-217`。比べ方: `merge_queue.go:454-460`。テスト: `merge_queue_run_test.go:401`、`:413`、`merge_queue_test.go:714-735`。help: `internal/cli/org_merge.go:180`、`:355` | `gateWhen` を「a removal less than %s before it counts as after it」にし、それを固定する 3 つのテストの文字列(`merge_queue_run_test.go:401`、`:472`、`internal/cli/org_merge_queue_test.go:239`)を合わせる。コードは変えない |
| LOW | readability | L10(新、`5e409db0` で入った)。help の MERGED の文に NOTE の説明を足したとき、行を詰め直さなかったので、`"exits 0. When\n"` という 13 文字の行が 70〜75 文字の行の間に残った。cobra は Long をそのまま出すので、`ralph org merge --help` にこの短い行がそのまま出る | `internal/cli/org_merge.go:154-159` | 「exits 0. When」を前の行の続きか次の行の頭に寄せて詰め直す。help のテストは出力を `strings.Fields` で空白 1 つにそろえて比べる(`internal/cli/org_merge_queue_test.go:307`)ので、期待の文字列は変わらない |
| LOW | maintainability | L11(新、`5e409db0` で起きた)。register の 800 行の目安の行が古くなった。(1) 払う手順が名指すテスト `TestMergeQueue_LatestRemovalAfterIsStrictlyAfter` は `5e409db0` で `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond` に改名され、repo のどこにもない。(2) 行の trigger「The next change to either test file」(`merge_queue_test.go` と `merge_queue_wait_test.go` の移し)は、同じ branch の `5e409db0` が両方を変えたことで文字どおりには満たされたが、払われていない。この PR の中の直しを数えない意図なら、その旨が行にない。(3) 行の行数(806、859、`merge_queue.go` 501、`merge_queue_run.go` 326、`merge_queue_run_test.go` 646、`org_merge_queue_test.go` 332)は今 807、854、489、333、647、348 で、`merge.go` の数と違い commit に結びついていない | `docs/tech-debt/README.md:537`。`git show 5e409db0 --stat`(両方のテストファイルを含む)。改名は `merge_queue_test.go:670-676` | テスト名を新しい名前に直し、行数に「at 5e409db0」のように commit を添え、trigger を「この PR の merge のあとに両方のテストファイルを変える次の変更」と書く。この PR の中で払うなら、行が書く移しを move-only でする |

### Positive notes

- `5e409db0` は前回の各指摘が名指した場所を Go の側ではすべて直した。L2 の 6 か所は同じ条件の文になり、L4 は query、型、parse、テストの補助関数(`mqAdded`、`mqPRText` の `added=`)まで一緒に消している。テストの期待の文字列は `mqrRequeueNext` という定数 1 つにまとまり、4 つのテストが同じ文を写す形はなくなった
- L3 の直しは境目のテストを両側に置いた(同じ秒は数える、1 秒後の `after` では数えない)。FR-9 の wait の表にも同じ秒の行がある
- L5 は MERGED の 4 通り(queue にすでにあった、`--requeue` で入れ直した、制御文字を含む、別の merge が記録済み)をテストに足し、2 つの NOTE の行の順も固定した
- `requeueNext` の doc は、push のない regate を挙げない理由を、step 7 が既定のブランチの位置を要ることと `RegateOutcomeUpToDate` に結びつけて書いている。理由が doc にあるので、次に文を変える人が regate を戻してしまう心配が少ない
- 前回の Positive notes に書いた記録、lock、escape、割り込みの扱いは `5e409db0` で変わっていない(`recordQueueMerge`、`queueMergeCleanup`、`queueInterruptedBefore`/`After` の差分はない)

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (新しい行は足さない) | — | L2 の残りは既存の前提の行への 1 文で足り、L11 は既存の 800 行の行の直しで足りる。L9、L10 は文の直しで、この PR の中で直せる。L8 は計画が NFR-1 を理由に変えないと決めたもので、先送りではない | — | この report |

### Coverage gaps

- テスト、静的解析、`check-sync.sh` は走らせていない(`/verify` と `/test` の担当)。計画の Progress によると、S6 の検証コマンドは exit 0
- skill と README の「打ち直せば待ちを続ける」が無条件のままであることは L2 に書いたが、文書のずれの検査は `/sync-docs` の担当なので、ほかの文書は探していない
- push と regate で新しい gate を取っても、その記録が removal から 5 分以内なら、FR-5b は skew のためまだ断る。断りの文は skew と時刻を書くので人は読み取れる。指摘にはしなかった
- GitHub の実際の挙動(head の変わった PR を queue から外すか、`enqueuedAt` と timeline の `createdAt` の関係、時刻が秒までか)は確かめていない。register の前提の行が扱う

### Recommendation

- Merge(Run 1 の時点): yes(CRITICAL、HIGH、MEDIUM なし。残る LOW は 5 件で、記録、lock、escape の正しさに関わるものはない)
- Follow-ups: L9 は `gateWhen` の 1 語とテストの文字列 3 か所。L10 は help の 2 行の詰め直し。L11 は register の 1 行の直し(テスト名、行数の commit、trigger の言い方)。L2 の残りは `requeueNext` の条件を広げるか register に 1 文足すかで、skill と README は `/sync-docs` に渡す。L8 は 2 本目まで残す
