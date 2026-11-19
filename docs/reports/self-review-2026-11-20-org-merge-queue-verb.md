# Self-review report: org-merge-queue-verb

- Date: 2026-11-20(cycle 1)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`
- Branch: `feat/org-merge-queue-verb`、HEAD `d53363fa`、base は `main` の `d919479a`(issue #342、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の 1 本目)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff d919479a...HEAD` の 38 ファイル(+5110 −276)の diff の質だけ。コードは `7a8a1e2b`(`merge_queue.go`)、`12ace9df`(step 11 の queue の道)、`b6c0ba3b`(`merge_queue_run.go`)、`1e706f0f`(`merge_queue_wait.go`)、`56fc4cab`(CLI の `--requeue` と終了コード 7)、`a9031034`(doc comment)。文書は `84184afd`。計画の記録の commit は Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify` と `/test` の担当)

## Evidence reviewed

- 新しいコード: `internal/org/merge_queue.go` 全 501 行、`merge_queue_run.go` 全 326 行、`merge_queue_wait.go` 全 390 行
- 変えたコード: `merge.go`(step 8 の `m.gateAt`、step 11・12、`mergeReady` と `mergeCheck` の欄)、`merge_run.go`(`mergeLocked` の queue の分岐、`recordMerge` を `mergeRecordDetails` と `mergeRecordErr` に分けた部分、`otherHeadClause` と `otherBaseClause`、`mergeInterruptedThen`)、`mergeenforce.go`(`enforce`、`readMergeQueue`、`enforceQueue`)、`merge_wait.go`(`poll` と `line` は diff の外で読んだ)、`internal/cli/org_merge.go`(help、`printMergeResult`、`printMergeQueuedLines`、`mergeQueuedError`)、`internal/cli/exitcode.go`
- doc comment だけの変更: `seat.go`(`EventPRMerged`)、`lockfile.go`、`regate.go`、`spawn.go`、`cleanup.go`
- 周辺(diff の外): `PrintableInboxText`(`inbox.go:208`、`\` は残すので 2 回かけても同じ)、`recordedMergeBranch` と `recordedMergeHead`(`followup.go:534-555`)、`mergeGateRecord.pos`(`mergeorder.go:465`、`l.events` の添字)、`orgRetryCommand` と `VerbCommand`、regate の `RegateOutcomeUpToDate`(`regate.go:46-48`、`:89-91`)と local の regate が記録してから push する順(`regate.go:63-66`)
- テスト(読んだだけで、走らせていない): `merge_queue_test.go` の strict の比べ方の表(`TestMergeQueue_LatestRemovalAfterIsStrictlyAfter`)、`merge_queue_wait_test.go` の `TestMerge_QueueWaitRecordDetails` と読み失敗の表、`merge_queue_run_test.go` の割り込みの表と gh の時間切れ、`internal/cli/org_merge_queue_test.go` の `--requeue` の表
- register: 変えた 9 行はどれも 5 つの cell(`(?<!\\)\|` で分けて数えた)。新しい 2 行(GitHub の確かめていない前提、800 行の目安)を読んだ
- サイズ: `merge.go` 779 → 822、`merge_run.go` 600 → 747、`internal/cli/org_merge.go` 428 → 558、`mergeenforce_test.go` 480 → 618。800 行を超えた 3 本(`merge.go`、`merge_queue_test.go` 806、`merge_queue_wait_test.go` 859)は register の新しい行が名指ししている

## Findings

指摘は L1 から振る。どれも LOW で、CRITICAL、HIGH、MEDIUM はない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1. FR-5b の断りと FR-9 の Failed が出す次の手(`requeueNext`)は、新しい gate の記録の取り方として `ralph org regate --branch <b>` を最初に挙げる。ところが regate は、HEAD が origin の既定のブランチの先端を含み、最新の gate の記録がその先端の上の HEAD の `gate_passed` なら、何も走らせず何も記録しない(`RegateOutcomeUpToDate`)。queue から外されたあと既定のブランチが動いていなければ(checks の揺れで外された場合など)、書かれた手のとおり regate を打っても記録は増えず、merge を打ち直すと同じ断りに戻る。FR-9 の文は「外された原因を直してから、新しい gate か、同じ gate のまま `--requeue`」と続くので、直す手(push)と同じ gate のままの入れ直しが 1 文に並んでいる | `internal/org/merge_queue_run.go:193-197`(`requeueNext`)、それを使う `:183-185` と `merge_queue_wait.go:247-248`。`internal/org/regate.go:46-48` と `:89-91`(「Nothing was run, read, recorded, or pushed」) | regate を「origin の既定のブランチが gate のあと動いたとき」に限って挙げるか、新しい gate は「push のあとの gate」を先に書く。FR-9 の文は「原因を直して push し、その gate を通す」と「揺れなら `--requeue`」の 2 つに分ける |
| LOW | maintainability | L2. 終了コード 7 の「打ち直せば待ちを続ける」は、1 本目では origin の既定のブランチが gate のあと動いていないときだけ当たる。動いていれば step 7 が branch を ready でも accepted でもないと断り(step 13 も同じ)、queue の読みまで行かない。動かないことは queue が前の PR を merge すると崩れる。仕様 D5 はこの間の regate の輪を受け入れている(FR-12 は 2 本目)が、その輪で queue にある PR を regate すると、local の regate は記録してから push するので、push で外された時刻(GitHub が head の変わった PR を queue から外す前提。この repo では確かめていない)が gate の記録より後になり、次の merge は FR-5b で `--requeue` を求める。どちらも記録を誤らせはせず、終了コード 1 で人に渡る | 約束する文: `merge_queue_run.go:300-304`(`queueNext`)、`internal/cli/org_merge.go:164`、`:273`、`internal/cli/exitcode.go:28-31`、`.claude/skills/director/SKILL.md:1742-1743`(4 写し)。step 7 の断り `internal/org/merge.go:356`、step 13 `:799`。仕様 `docs/specs/2026-11-19-org-merge-queue.md:47`(FR-12)、`:159`(D5)。regate の順 `internal/org/regate.go:63-66` | 2 本目が同じ release に入らないなら、これらの文に「origin の既定のブランチが動いていなければ」を添える。register の GitHub の前提の行に、queue にある PR の regate が FR-5b に当たる道を 1 文足す |
| LOW | null-safety | L3. FR-9 の「入れたあとに外された」は、外された時刻が entry の `enqueuedAt` より厳密に後のものだけを数える。GitHub の DateTime は秒までなので、入れたのと同じ秒に外された(すぐに外された)removal は落ち、Failed は理由と actor の代わりに「外されたことは読めなかった、PR を確かめて」と出す。記録しない向きなので害は文だけ。同じ秒の removal は、前の entry のものより今の entry のものである見込みが高い | `internal/org/merge_queue.go:455-462`(`r.At.After(after)`)、`merge_queue_wait.go:236-252`(`queueLeft`)。`merge_queue_test.go:695` が「同じ時刻なら見つからない」を固定している | `!r.At.Before(after)`(同じ時刻を含む)にし、テストの 695 行の期待を変える。厳密にしておくなら、doc comment に秒の粒度で同じ秒を落とすことを書く |
| LOW | maintainability | L4. `mergeQueuePR.Added` は、query で `ADDED_TO_MERGE_QUEUE_EVENT` を頼み、`createdAt` を RFC 3339 として厳しく読み(読めなければ read 全体が失敗)、テストが query の文字列ごと固定しているが、本番のコードはどこも読まない。読めない `createdAt` が 1 つあると、使わない値のために、入れる前の読みは断りに、待ちの最後の読みは FR-11 の終了コード 1 になる。`last: 50` の枠も add と remove で分け合うので、FR-5b が見る removal の数が減る | `internal/org/merge_queue.go:36-37`(query)、`:90-93`(欄)、`:329-334`(読み)。`grep -rn '\.Added\b' --include='*.go' internal` で本番の読み手は 0 件、`merge_queue_test.go:123` と `:263` だけ | 使う予定(entry がないときの `enqueuedAt` の代わりなど)がなければ、query から `ADDED_TO_MERGE_QUEUE_EVENT` を外し、欄と枝も消す。残すなら、doc comment に誰も読まないことと理由を書く |
| LOW | maintainability | L5. `MergeResult.QueueNote` の doc は「wait がどう終わっても残る」と言い、`TestMerge_QueueWaitRecordDetails` は MERGED の結果に残ることを確かめる。ところが表示するのは `QUEUED:` の行(`printMergeQueuedLines`)だけで、MERGED の行(`printMergeLines`)と Failed は出さない。`--requeue` で外されたあとに入れ直した事実(外された時刻、actor、理由)は、merge まで進むと出力にも台帳にも残らない | `internal/org/merge_run.go:141-146`(doc)、`internal/cli/org_merge.go:477-490`(`printMergeLines` は `res.Note` だけ)、`:507-508`。`merge_queue_wait_test.go:264`、`:299` | MERGED の行にも `NOTE:` として出す(`res.Note` と並べる)か、`pr_merged` の Details に `requeue=true` のような印を足す。どちらもしないなら doc を「`QUEUED:` の行にだけ出る」に直す |
| LOW | readability | L6. `mergeCheck` の doc は「checkMerge の実行中の MergeParams に既定値を埋めたもの」のままだが、`gateAt` が step 8(`checkMergeBranch`)の出力を step 11 に運ぶ欄として加わった。設定だけだった構造体に、手順の間の状態が 1 つ入り、step 8 の関数がそれを書く副作用を持つ。欄の doc は「gate の記録の時刻」と言うが、受け入れのほうが遅ければその時刻になる(`mergeGateTime`) | `internal/org/merge.go:206-228`、書く場所 `:369`、読む場所 `:418`。計画の S3a のずれの記録(既存のテストが `mergeReady` を `==` で比べるため) | 構造体の doc に「step 8 が `gateAt` を書き、step 11 が読む」を足し、欄の doc を「gate の記録か受け入れの、遅いほうの時刻」に直す |
| LOW | maintainability | L7. `EventPRMerged` の doc(Details の契約)は、queue の道の記録を「ralph org merge が enqueuePullRequest で入れた PR を queue が merge した」と書く。FR-5 の道では、すでに queue にあった PR(人や別の道具が入れたものも含む)を入れずに待ち、merge されれば同じ `via=merge_queue` を記録する | `internal/org/seat.go:147-152`、`merge_queue_run.go:89-92`(入れずに `queued(res, false, ...)`) | 「ralph org merge が入れたか、gate の head で queue にあるのを見つけた PR」にする |
| LOW | exception-handling | L8. 同じ verb の 2 つの GraphQL の読みが `errors` の扱いで食い違う。step 11 の `readMergeQueue` は `"errors": []` を errors ありとして断り、新しい `mergeQueueGraphQLErrors` は空の list を errors なしとして読む。GitHub が空の list を返すことは見込みにくく実害はないが、この diff は `readMergeQueue` を書き換えていて、新しい helper に寄せる機会だった | `internal/org/mergeenforce.go:271`(`len(errs) > 0 && string(errs) != "null"`)、`internal/org/merge_queue.go:216-226` | `readMergeQueue` も `mergeQueueGraphQLErrors` で判定する(文は `enforceUnread` で包むまま) |

## Positive notes

- GraphQL の 3 つの文字列(`mergeQueuePRQuery`、`mergeQueueEnqueueMutation`、`mergeQueueQuery`)は固定で、値はすべて gh の field で渡す。`-F` は `mergeQueuePRCheck` が 1〜`MaxInt32` に絞った番号だけで、`@` や `{owner}` に読まれる余地がない。node ID は文字の種類と長さ、head は小文字の 40 か 64 桁を argv を作る前に確かめ、通らなければ `errMergeQueueInput` で「何も送っていない」と「入ったか分からない」を分けている
- `pr_merged` を書く道は `queueDecided` の MERGED の枝だけで、head と base を step 9 と同じ欄、同じ完全一致で比べる。D7 の確かめは manifest の lock の中で台帳を読み直し、branch・PR・head と最新の gate の記録の位置で見る。`mergeGateRecord.pos` は `l.events` の添字なので `l.events[after+1:]` はずれない
- lock の扱いは計画の D1 のとおり。`mergeLocked` が返る時点で `defer` が `regate.lock` を離し、待ちの最初の行の「with regate.lock released」は事実と合う。片付けの前に取り直して marker を `Truncate(0)` で空にし、取れなければ片付けを飛ばして `ralph org cleanup` を名指す
- 割り込みは 3 つの時点で別の文になる。enqueue を送る前は結果を空にして「入れていない」、送ったあとは「入っているか入っているかもしれない」、待ちの間は「何も記録していない、queue に残っているかもしれない」。MERGED を読んだあとの割り込みは `WithoutCancel` で記録を済ませ、片付けだけを飛ばす
- GitHub から来る文字(state、base、reason、actor、gh の stderr)は、どの道でも最後に `PrintableInboxText` を通る。待ちの行は `mergeWait.line` が escape し、確かめの失敗の文は `mergeQueueShown` で引用して 64 バイトで切る
- 読めない値は断る向きに倒している。gate の ts が読めなければ `gateAt` は 0 で、すべての removal を数える。`timelineItems.nodes` がない答えは FR-5b の見落としにつながるので読み失敗にする。mutation の答えに entry がなければ記録せず終了コード 1
- queue のない道は文が変わらない。`recordMerge` の Details と失敗の文、`otherHeadMerged` と `otherBaseMerged`、`mergeInterruptedAfterStart` は、helper に分けたあとも同じ文字列を作る。`mergeQueueQuery` に足した `configuration { mergeMethod }` は `mergeQueue` が null の repo では解決されない
- `MergeResult` の各欄の doc が、queue の道で何がいつ入るかを書き分けている。CLI の `printMergeResult` は、割り込みで `Queued` が残っていても `err` を先に返し、終了コード 7 を出さない(D6)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (新しい行は足さない) | — | L1〜L8 はどれも小さな文やコードの直しで、この PR の中で直せる。L2 の regate と FR-5b の道は、register の既存の行(merge queue の道の確かめていない前提)に 1 文足せば足りる | — | この report |

## Coverage gaps

- テスト、静的解析、`check-sync.sh` は走らせていない(`/verify` と `/test` の担当)。計画の Progress によると、各スライスの検証コマンドは exit 0
- GitHub の実際の挙動(`enqueuePullRequest` の権限、`expectedHeadOid`、removal と merge の順、head の変わった PR を queue から外すかどうか)はこの review でも確かめていない。register の新しい行が扱う
- FR-5b は時刻だけを比べ、外された entry がどの head のものだったかは見ない。L2 の道はその帰結の 1 つで、仕様の FR-5b の言い方どおり。2 本目(FR-12、FR-13)の設計で扱う話として残す
- doc comment と skill の文のずれは対象外。気づいた点は L5〜L7 に入れた

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 8 件で、記録や lock の正しさに関わるものはない)
- Follow-ups: L1 と L2 は次の手の文の直しで、1 commit にまとまる(L2 は register の 1 文も)。L3 は比較 1 か所とテストの期待 1 行。L4 は query と欄の削除か doc の 1 文。L5〜L7 は doc か表示の小さな直し。L8 は `readMergeQueue` の 1 行を helper に寄せる
