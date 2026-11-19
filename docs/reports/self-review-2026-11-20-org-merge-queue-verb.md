# Self-review report: org-merge-queue-verb

- Date: 2026-11-20(cycle 1、self-review の直しのあとの 2 回目の実行)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`
- Branch: `feat/org-merge-queue-verb`、HEAD `49a88cac`、base は `main` の `d919479a`(issue #342、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の 1 本目)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff d919479a...HEAD` の 40 ファイル(+5199 −275)の diff の質だけ。前回の report(`d28f6532`、HEAD `d53363fa` で LOW 8)のあとに入った `5e409db0`(L1〜L7 の直し、12 ファイル +188 −163)を中心に見た。`49a88cac` は計画の記録だけなので、Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

- `5e409db0` の差分すべて(`git show 5e409db0`)。コードは `merge_queue.go`(query、`mergeQueuePR`、`mergeQueueTimeline`、`latestMergeQueueRemovalAfter`)、`merge_queue_run.go`(`requeueNext`、`queueNext`)、`merge_queue_wait.go`(`queueLeft`)、`internal/cli/org_merge.go`(help、`ErrMergeQueued`、`printMergeLines`)、doc だけの `merge.go`(`mergeCheck`)、`seat.go`(`EventPRMerged`)、`internal/cli/exitcode.go`、`cmd/ralph/main.go`
- 通して読んだファイル: `merge_queue_run.go` 全 333 行、`merge_queue_wait.go` 全 390 行、`merge_queue.go` の 1〜310 行と 425〜480 行(query、型、検査、parse、removal の helper)
- 周辺(diff の外): regate の結果の doc(`regate.go:30-110`。`RegateOutcomeUpToDate` は「HEAD が既定のブランチの先端を含み、最新の gate の記録がその先端の上の HEAD の pass」のときだけで、push のあとの HEAD なら `RegateOutcomeGated` になる)、`mergeGateTime`(`merge_queue_run.go:36-52`)、step 8 の `m.gateAt` の書き込み(`merge.go:373`)と step 11 の読み(`:422`)、`printMergeResult` の Failed の枝(`org_merge.go:462-481`)
- テスト(読んだだけで、走らせていない): `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond` の表、`TestMergeQueue_RemovalAfterGateAllowsForClockSkew` の境界の表(`merge_queue_test.go:714-740`)、`TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue` の skew の行(`merge_queue_run_test.go:393-420`)、`TestOrgMerge_PrintsAMergeQueueMergeAsMerged` の NOTE の 4 行、`TestOrgMerge_HelpDescribesTheMergeQueuePath`
- 前回の指摘が名指した場所を 1 つずつ grep した: 「keep waiting」「待ちを続ける」(Go、skill、README)、`\.Added\b`、`ADDED_TO_MERGE_QUEUE_EVENT`、`AddedToMergeQueueEvent`、`fix what made the queue`、`get a new gate record`、`LatestRemovalAfterIsStrictlyAfter`
- register: `docs/tech-debt/README.md` の merge queue の 2 行(GitHub で確かめていない前提の行、800 行の目安の行 `:537`)を差分と突き合わせた
- サイズ(今): `merge.go` 826、`merge_queue_test.go` 807、`merge_queue_wait_test.go` 854、`merge_queue.go` 489、`merge_queue_run.go` 333、`merge_queue_run_test.go` 647、`internal/cli/org_merge.go` 569、`internal/cli/org_merge_queue_test.go` 348

## Findings

番号は前回の L1〜L8 を引き継ぎ、新しい指摘は L9 から振る。CRITICAL、HIGH、MEDIUM はない。残っている指摘は LOW 5 件(L2 の残り、L8、L9、L10、L11)。

### 前回の指摘の状態

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

### 残っている指摘と新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L2(残り)。前回の L2 の後半、queue にある PR を regate した道が手当てされていない。既定のブランチが動いて step 7 が断ったあと、人が regate を打つと、local の regate は記録してから push する。push で PR が queue から外されると(GitHub がそうするかは register の前提の行どおり確かめていない)、外された時刻が gate の記録より後になり、次の merge は FR-5b で断る。そのときの次の手 `requeueNext` は「直しを push して regate」か「揺れた検査のあとは `--requeue`」で、どちらもこの場合に合わない。regate を打ち直しても `RegateOutcomeUpToDate` で何も記録せず、同じ断りに戻る。正しい手は `--requeue`(queue はこの gate の head をまだ試していない)だが、文からは読み取りにくい。register の前提の行にもこの道は書かれていない | `merge_queue_run.go:197-201`(`requeueNext`)、`:181-186`(FR-5b の断り)。regate の順は `regate.go:63-66`(前回の report の証拠)。register の行 `docs/tech-debt/README.md` の「The merge queue path of `ralph org merge` (the first PR of #342) was built on a fake gh」 | `requeueNext` の `--requeue` の条件を「揺れた検査のあと、または regate の push で外されたときのように、queue がこの gate の head を試していないとき」に広げるか、register の前提の行に「queue にある PR を regate すると push で外され、次の merge は FR-5b で断る。そのときは `--requeue`」を 1 文足す。skill の 3 文(`.claude/skills/director/SKILL.md:1743`、`.claude/skills/org/SKILL.md:396`、`:2002`、それぞれ 4 写し)と README(`README.md:296`)の無条件の「打ち直せば待ちを続ける」は、計画の S6 のとおり `/sync-docs` で直す |
| LOW | exception-handling | L8(前回のまま、意図して残す)。step 11 の `readMergeQueue` は `"errors": []` を errors ありとして断り、`mergeQueueGraphQLErrors` は空の list を errors なしとして読む | `internal/org/mergeenforce.go:271`、`internal/org/merge_queue.go:209-223` | 計画の判断(NFR-1)どおり残す。2 本目で `mergeenforce.go` を変えるときに寄せる |
| LOW | maintainability | L9(新)。FR-5b の断りの文と、コードと help が、skew の境目で食い違う。`gateWhen` は「a removal up to 5m0s before it counts as after it」と書くが、`mergeQueueRemovedAfterGate` は gate の時刻の 5 分前より厳密に後だけを数え、ちょうど 5m0s 前の removal は数えない。テストもそれを固定している(「5m before the gate, the edge of the skew」は数えない)。help、`--requeue` の usage、README は「less than 5 minutes before it」で、コードと合う。英語の「up to 5m0s」はふつう 5m0s を含むので、同じ verb の 2 つの文が同じ規則を別の境目で言っている。秒の粒度の時刻なので、ちょうど 5m0s 前は実際に起こりうる。前回の差分(`12ace9df`/`b6c0ba3b`)からあったもので、L3 で FR-9 の比べ方を「同じ秒を含む」にしたため、隣の FR-5b の文の境目が目立つようになった | 文: `merge_queue_run.go:216-217`。比べ方: `merge_queue.go:454-460`。テスト: `merge_queue_run_test.go:401`、`:413`、`merge_queue_test.go:714-735`。help: `internal/cli/org_merge.go:180`、`:355` | `gateWhen` を「a removal less than %s before it counts as after it」にし、それを固定する 3 つのテストの文字列(`merge_queue_run_test.go:401`、`:472`、`internal/cli/org_merge_queue_test.go:239`)を合わせる。コードは変えない |
| LOW | readability | L10(新、`5e409db0` で入った)。help の MERGED の文に NOTE の説明を足したとき、行を詰め直さなかったので、`"exits 0. When\n"` という 13 文字の行が 70〜75 文字の行の間に残った。cobra は Long をそのまま出すので、`ralph org merge --help` にこの短い行がそのまま出る | `internal/cli/org_merge.go:154-159` | 「exits 0. When」を前の行の続きか次の行の頭に寄せて詰め直す。help のテストは出力を `strings.Fields` で空白 1 つにそろえて比べる(`internal/cli/org_merge_queue_test.go:307`)ので、期待の文字列は変わらない |
| LOW | maintainability | L11(新、`5e409db0` で起きた)。register の 800 行の目安の行が古くなった。(1) 払う手順が名指すテスト `TestMergeQueue_LatestRemovalAfterIsStrictlyAfter` は `5e409db0` で `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond` に改名され、repo のどこにもない。(2) 行の trigger「The next change to either test file」(`merge_queue_test.go` と `merge_queue_wait_test.go` の移し)は、同じ branch の `5e409db0` が両方を変えたことで文字どおりには満たされたが、払われていない。この PR の中の直しを数えない意図なら、その旨が行にない。(3) 行の行数(806、859、`merge_queue.go` 501、`merge_queue_run.go` 326、`merge_queue_run_test.go` 646、`org_merge_queue_test.go` 332)は今 807、854、489、333、647、348 で、`merge.go` の数と違い commit に結びついていない | `docs/tech-debt/README.md:537`。`git show 5e409db0 --stat`(両方のテストファイルを含む)。改名は `merge_queue_test.go:670-676` | テスト名を新しい名前に直し、行数に「at 5e409db0」のように commit を添え、trigger を「この PR の merge のあとに両方のテストファイルを変える次の変更」と書く。この PR の中で払うなら、行が書く移しを move-only でする |

## Positive notes

- `5e409db0` は前回の各指摘が名指した場所を Go の側ではすべて直した。L2 の 6 か所は同じ条件の文になり、L4 は query、型、parse、テストの補助関数(`mqAdded`、`mqPRText` の `added=`)まで一緒に消している。テストの期待の文字列は `mqrRequeueNext` という定数 1 つにまとまり、4 つのテストが同じ文を写す形はなくなった
- L3 の直しは境目のテストを両側に置いた(同じ秒は数える、1 秒後の `after` では数えない)。FR-9 の wait の表にも同じ秒の行がある
- L5 は MERGED の 4 通り(queue にすでにあった、`--requeue` で入れ直した、制御文字を含む、別の merge が記録済み)をテストに足し、2 つの NOTE の行の順も固定した
- `requeueNext` の doc は、push のない regate を挙げない理由を、step 7 が既定のブランチの位置を要ることと `RegateOutcomeUpToDate` に結びつけて書いている。理由が doc にあるので、次に文を変える人が regate を戻してしまう心配が少ない
- 前回の Positive notes に書いた記録、lock、escape、割り込みの扱いは `5e409db0` で変わっていない(`recordQueueMerge`、`queueMergeCleanup`、`queueInterruptedBefore`/`After` の差分はない)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (新しい行は足さない) | — | L2 の残りは既存の前提の行への 1 文で足り、L11 は既存の 800 行の行の直しで足りる。L9、L10 は文の直しで、この PR の中で直せる。L8 は計画が NFR-1 を理由に変えないと決めたもので、先送りではない | — | この report |

## Coverage gaps

- テスト、静的解析、`check-sync.sh` は走らせていない(`/verify` と `/test` の担当)。計画の Progress によると、S6 の検証コマンドは exit 0
- skill と README の「打ち直せば待ちを続ける」が無条件のままであることは L2 に書いたが、文書のずれの検査は `/sync-docs` の担当なので、ほかの文書は探していない
- push と regate で新しい gate を取っても、その記録が removal から 5 分以内なら、FR-5b は skew のためまだ断る。断りの文は skew と時刻を書くので人は読み取れる。指摘にはしなかった
- GitHub の実際の挙動(head の変わった PR を queue から外すか、`enqueuedAt` と timeline の `createdAt` の関係、時刻が秒までか)は確かめていない。register の前提の行が扱う

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。残る LOW は 5 件で、記録、lock、escape の正しさに関わるものはない)
- Follow-ups: L9 は `gateWhen` の 1 語とテストの文字列 3 か所。L10 は help の 2 行の詰め直し。L11 は register の 1 行の直し(テスト名、行数の commit、trigger の言い方)。L2 の残りは `requeueNext` の条件を広げるか register に 1 文足すかで、skill と README は `/sync-docs` に渡す。L8 は 2 本目まで残す
