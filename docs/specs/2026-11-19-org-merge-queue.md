# `ralph org merge` を merge queue のある既定のブランチで使えるようにする

- Date: 2026-11-19
- Issue: #342
- Status: Approved(2026-11-19、系列の指示で推す案を自律に承認した。consult の判定「直してから進める」の 3 点を入れたあと。人が決めたのは、fake のテストで実装し実地は release のあと下流で確かめる、の 1 点だけ)
- 実装の担当: レーン B(#340 の merge のあと、新しい main の上で /plan と実装)

## Summary

既定のブランチに GitHub の merge queue があると、`ralph org merge` は今は断る。これを、PR を queue に入れて MERGED になるまで待ち、`pr_merged` を記録する形に変える。queue があるときは、merge のたびの `ralph org regate` を要らなくし、並列の feature org の PR を queue の中で並べて merge できるようにする。

## Background and problem

### Current state

- `ralph org merge` の step 11(保護の確認、`CheckMergeEnforcement`、`internal/org/mergeenforce.go:124-157`)は、GraphQL の `repository.mergeQueue(branch:)` が null でないか、rules に `merge_queue` があれば断る。文は「ralph org merge does not merge into a branch with a merge queue, so merge the PR through the queue yourself」(`:231-232`、`:136-137`)。queue の値が読めないときも断る(`:208-229`)
- step 12(`readMethod`、`internal/org/merge.go:698-716`)は `viewerDefaultMergeMethod` が `MERGE_QUEUE` だと断る
- merge のあとの read-back(`internal/org/merge_run.go:421-449`)は 1 回だけ読み、OPEN のままの PR を「merge されなかった」として失敗にする。`autoMergeRequest` があれば `--disable-auto` で外す
- merge-order(`internal/org/mergeorder.go:1289-1322`)は、gate の記録の base が origin の既定のブランチの先端と違えば `rebase` にする。1 本 merge すると残りはすべて `rebase` になり、`ralph org regate` で取り込み直してから CI を待つ。N 本の PR は N 回、直列に CI を待つ
- doctor の Merge guarantee(`internal/org/mergeguarantee.go:212-249` の `decide`)は、rules の `merge_queue` と必須の検査が 1 つ以上あれば、すでに `github_protection` と判定する。断っているのは merge の動詞だけ
- これは意図した対象外だった: `docs/plans/archive/2026-10-17-org-auto-merge.md` の Non-goals 40 行「merge queue のある branch への merge(動詞は断る)」、Assumptions 53〜56 行、D6(前の版の「QUEUED: で終了コード 0」は捨てた)、D11(queue があれば断る)

### Desired state

- 既定のブランチに queue と必須の検査がある repo で、`ralph org merge --branch <b>` が今までの確認(step 1〜10、13)を通したあと、PR を queue に入れ、MERGED になるまで待ち、`pr_merged` を記録する
- queue から外されたら(CI の失敗、時間切れ、人の手、保護の失敗など)、記録せずに理由を出して終了コード 1
- 待つ予算が尽きても MERGED にならなければ、記録せず、PR は queue に残したまま、決まった終了コードで返す。同じ動詞を打ち直すと、queue に入れ直さずに待つところから続ける
- queue のある既定のブランチでは、merge-order と merge が、base の古い gate の記録の PR も merge してよいと読む(regate が要らない)。保証の拠り所は queue の必須の検査になる
- queue のない repo の挙動は変わらない

## Requirements

### Functional requirements

- [ ] FR-1 queue の判定: merge の step 11 は、既定のブランチの queue の有無を今までどおり GraphQL の `mergeQueue` と rules の `merge_queue` で読む。どちらかが queue を示せば「queue の道」に入る。どちらかが読めなければ今までどおり断る(知らない状態は断る側)
- [ ] FR-2 queue の道の保護の確認: 必須の status check が 1 つ以上あること(rules の `required_status_checks` か classic の保護)を求める。1 つもなければ断る(queue が検査なしで merge するので、merge の保証がない)。今の「up-to-date の ruleset を迂回できない」の確認は、queue の道では求めない(queue が最新の base の上で検査する)。merge を打つ本人が queue を迂回できるかは確かめない(ralph は `--admin` を使わず、queue を通すので迂回は起きない)
- [ ] FR-3 merge 方法: queue の道では step 12(`viewerDefaultMergeMethod`)を飛ばす。方法は queue の設定が決める。記録には GraphQL の `MergeQueueConfiguration.mergeMethod` が読めればその値、読めなければ `queue` と書く
- [ ] FR-4 queue に入れる: `gh api graphql` の `enqueuePullRequest(input: {pullRequestId, expectedHeadOid})` で入れる。`expectedHeadOid` は gate の head。PR の node ID は `gh pr view --json id` で読む。`gh pr merge` は使わない(queue のある branch では内部で auto-merge を有効にし、即時に入ったのか予約されたのかを出力から見分けられないため)。`--admin` は使わない
- [ ] FR-5 入っていたら入れ直さない: 入れる前に PR の `isInMergeQueue` と `mergeQueueEntry` を読み、すでに queue にあり、PR の head が gate の head なら、入れずに FR-6 の待ちから続ける
- [ ] FR-5b 外されたあとは黙って入れ直さない: queue になく、その branch の最新の gate の記録(`gate_passed`、`gate_ci_passed`、受け入れた `gate_accepted`)より後の `RemovedFromMergeQueueEvent` が PR にあれば、入れずに断る(終了コード 1、外された理由と時刻を出す)。同じ gate の結果で一度試して外されたので、新しい gate(push のあとの gate か regate)を通すか、わざと入れ直すときの明示の flag(案: `--requeue`)を求める。CI の揺れや人の手で外したときのための flag で、director は人に頼まれたときだけ付ける
- [ ] FR-6 MERGED までの待ち: queue に入れたあと、PR の `state`、`headRefOid`、`baseRefName`、`mergeCommit`、`isInMergeQueue`、`mergeQueueEntry { state position }` と、timeline の `RemovedFromMergeQueueEvent`(入れた時刻より後のもの)を一定の間隔で読む。間隔は `--wait-checks` と同じ 30 s。待つあいだ `regate.lock` を持たない(ほかの branch の merge が並べて queue に入れられるように)
- [ ] FR-7 予算: 待ちは `--wait-checks` の予算の残りで区切る(1 つの予算で、検査の待ちと queue の待ちの両方を区切る)。`--wait-checks` が 0(既定)なら、queue に入れたところで返す
- [ ] FR-8 MERGED になった: `headRefOid` が gate の head で、`baseRefName` が既定のブランチなら、今までどおり `pr_merged` を記録する。details に `via=merge_queue` を足し、`merge_commit` は PR の `mergeCommit.oid`(queue が既定のブランチに入れたコミット)。そのあとの worktree の片付け(`mergeCleanup`)は今までどおり。head か base が違えば記録せずに終了コード 1(今の read-back と同じ)
- [ ] FR-9 queue から外された: MERGED でなく、`isInMergeQueue` が false で、入れたあとの `RemovedFromMergeQueueEvent` があれば、記録せず、その `reason`(自由文字列)と actor を出して終了コード 1。外された理由が読めなくても、queue になく MERGED でもなければ同じ扱い
- [ ] FR-10 予算切れ: 予算が尽きても queue にあるなら、記録せず、PR は queue に残し(dequeue しない)、「queue に入っている、打ち直せば待ちを続ける、GitHub で merge されたら `ralph org cleanup` が片付ける」と出して、決まった終了コード(1 とも 0 とも違う値、案は 7)で返す
- [ ] FR-11 読めない状態: 待ちのあいだ GraphQL が読めないときは予算の中で読み直す。予算の終わりに状態が分からなければ、記録せず、「queue にあるかもしれない」と出して終了コード 1
- [ ] FR-12 regate を省く: queue のある既定のブランチでは、merge-order は、gate を通った(または受け入れられた)記録の base が既定のブランチの先端と違っても `rebase` にせず `ready`(または `accepted`)と読み、理由に queue を書く。merge の step 7 も同じ読みにする。step 13(`checkOriginUnmoved`、`merge.go:725-735`)は今 origin の既定のブランチの先端が gate の base から動いていれば断るので、queue の道では base の確かめを外し、branch の先端が gate の head であることの確かめだけを残す。queue の有無は live に読む(merge-order が gh で PR を読むのと同じ道)。merge は queue の有無を 1 回だけ読み、step 7、11、13 で同じ結果を使う(step ごとに読み直して結果が食い違うことを避ける)。読めなければ queue なしとして今までどおり `rebase` にする(merge は step 11 で断る)
- [ ] FR-13 director: `/director` skill の「merge と auto_merge」に、queue のある repo の 2 段の手順を書く。1 段目で merge-order の `ready` と `accepted` の branch をすべて `ralph org merge --branch <b> --wait-checks 0` で queue に入れ、FR-10 の終了コード(案 7)を集める。2 段目で 1 本ずつ予算付き(`--wait-checks 60m` など)で打ち直し、FR-5 の待ちから MERGED を待つ。regate の輪は回さない。今の skill は終了コード 1 を「人に頼まれないまま打ち直さない」としている(`director/SKILL.md:1723`)ので、それと対にして、FR-10 の終了コードは「打ち直してよい」と書く。FR-5b で断られたら人に伝え、`--requeue` は人に頼まれたときだけ付ける。今の「merge は 1 本ずつ background で打ち、終わるまでほかの merge を打たない」(`:1698`、`:1779`)は、queue のない repo の手順として残す。`/org` skill の「merge してよい順」と、doctor の merge guarantee との関係の文(queue は doctor では pass、merge では断る、の部分)を直す

### Non-functional requirements

- [ ] NFR-1 queue のない repo の挙動、出力、終了コードは変わらない(今のテストを変えずに通す)
- [ ] NFR-2 GitHub の挙動は文書と公開 repo の実データで裏を取り、取れない点は「確かめていない前提」として計画と tech-debt に書く。実地の確かめは release のあと下流の会社の repo でする(ユーザーの決定 2)
- [ ] NFR-3 分からない状態では `pr_merged` を記録しない(黙って merge 済みと記録する向きの穴を作らない)
- [ ] NFR-4 gh の呼び出しは今までどおり `--hostname <解いた host>`(#340)と `-R`、`GH_PROMPT_DISABLED=1`、待ちのあとの呼び出しは `runDetached`

## Acceptance criteria

- [ ] AC1 Given queue と必須の検査がある既定のブランチ(fake の gh), when `ralph org merge --branch <b> --wait-checks 30m`, then `enqueuePullRequest` を gate の head で 1 回打ち、MERGED を読んだあと `pr_merged`(`via=merge_queue`、`merge_commit` は `mergeCommit.oid`)を記録して終了コード 0
- [ ] AC2 Given 入れたあとに `RemovedFromMergeQueueEvent`(reason `failed_checks`), then 記録せず、reason を出して終了コード 1
- [ ] AC3 Given queue のない repo, then 今の merge のテストがすべて変わらずに通る
- [ ] AC4 Given `--wait-checks 0`、または予算の中で MERGED にならない, then 記録せず、queue に残したまま FR-10 の終了コードで返す
- [ ] AC5 Given すでに queue にある PR(head は gate の head), when 打ち直す, then `enqueuePullRequest` を打たずに待ちから続ける
- [ ] AC6 Given queue があり必須の検査がない, then 断る(終了コード 1、記録なし)
- [ ] AC7 Given queue があり、gate の記録の base が既定のブランチの先端より古い, then merge-order は `ready` と出し、merge は regate なしで step 7 と step 13 を通って queue に入れる。branch の先端が gate の head と違えば今までどおり断る。queue が読めない repo では今までどおり `rebase`
- [ ] AC10 Given 最新の gate の記録より後に `RemovedFromMergeQueueEvent` がある PR, when `--requeue` なしで打ち直す, then 入れずに断る(終了コード 1)。`--requeue` があれば入れる
- [ ] AC11 Given queue のある repo, when director の skill の 2 段の手順で 3 本の ready を扱う, then 1 段目で 3 本とも queue に入る(fake の gh で `enqueuePullRequest` が 3 回打たれ、どれも FR-10 の終了コードで返る)
- [ ] AC8 Given MERGED だが `headRefOid` が gate の head と違う, then 記録せずに終了コード 1
- [ ] AC9 `/director` と `/org` の skill の 4 写し、README の `ralph org merge` の説明、`ralph org merge --help` が、queue の道と FR-10 の終了コードを書く

## User stories

1. As a director with `auto_merge = true`, I want to hand every ready feature PR to the merge queue at once, so that their CI runs in parallel instead of one regate and one CI wait per PR.
2. As a human running `ralph org merge` by hand, I want it to enqueue and return at once when I give no wait budget, so that I can watch the queue on GitHub and run `ralph org cleanup` after it merges.

## Constraints

### In scope

- `internal/org/mergeenforce.go`(queue の道の判定と必須の検査)、`merge.go`(step 7・11・12)、`merge_run.go`(enqueue、待ち、記録)、`merge_wait.go`(予算の共有)、`mergeorder.go`(FR-12 の分類)、GraphQL の読みの新しいファイル(案: `merge_queue.go`)
- `internal/cli/org_merge.go` の help と終了コード
- `/director` と `/org` の skill の 4 写し、README、`docs/specs/2026-10-07-org-multi-org-director.md` の該当の決定に追記

- queue の道の merge は `[org.gate].regate_checks` を読まない。merge の拠り所は、queue が最新の base の上で走らせる必須の検査になる(doctor の `github_protection` と同じ拠り所)
- 下流の CI は `merge_group` のイベントでも必須の検査を走らせる必要がある。走らなければ queue は検査の報告を待ち、`check_response_timeout` で PR を外す(FR-9 で終了コード 1)。PR の側に検査が 1 つもなければ、今までどおり step 10 の 0 件で待つ。README と `/org` skill に書く

### Out of scope

- queue から外す操作(`dequeuePullRequest`)。ralph は queue から外さない
- `--admin` での迂回、queue を飛ばす merge
- queue の設定(ruleset の `merge_queue` の parameter)を ralph が変えること
- doctor の判定の変更(すでに queue と必須の検査で `github_protection`)。classic の保護の queue を doctor が読まない点はそのまま
- 実地の確かめ(release のあと下流で)

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `ralph org merge` | queue のある repo で断らずに queue に入れて待つ。新しい終了コード | High |
| `ralph org merge-order` | queue のある repo で base の古い記録を `ready` と読む | Medium |
| director の auto_merge の手順 | queue のある repo で regate の輪を回さない | Medium |
| queue のない repo | 変わらない | None |

## Dependencies

- #340(レーン B、`ssh -G` で解いた host)が先に main に入ること。`CheckMergeEnforcement` は解いた host を受け取る
- gh 2.103 以降(`gh api graphql`、`gh pr view --json id`)

## Research findings

### Codebase analysis

- step の並び: `checkMerge`(`internal/org/merge.go:203-228`)の 13 step。queue の判定は step 11、方法は step 12、ready の要求は step 7(`:296-300`)
- merge の argv は `gh pr merge <n> <method> --match-head-commit <head> -R <repo>`(`merge_run.go:345-346`)、60 s、1 回。read-back は `state,headRefOid,baseRefName,mergeCommit,autoMergeRequest` を 1 回(`:43-46`、`:420`)
- 待ちは `merge_wait.go`: 最大 6 h、30 s ごと、待つあいだ `regate.lock` を離す(`merge_run.go:247-248`)
- `pr_merged` の details: `branch pr head method merge_commit auto_merge`(と `gate=accepted`)(`merge_run.go:567-584`)
- 終了コードは今 0 と 1 だけ(`internal/cli/org_merge.go:316-366`)
- レーン B が #340 で確かめた点: `CheckMergeEnforcement` は解いた host を merge.go から受け取る。merge 以降の gh は `runDetached`

### Best practices

- `gh pr merge` は queue のある branch では GraphQL の `enablePullRequestAutoMerge` を呼ぶ(cli/cli の `pkg/cmd/pr/merge/merge.go`)。検査が通っていれば queue に入り、終わっていなければ auto-merge が有効になる。非 TTY では何も出さずに exit 0 で、どちらになったか分からない
- GitHub Docs は、auto-merge を使わず今入れられるときだけ入れるには `enqueuePullRequest` を使うと書く。入力は `pullRequestId`、`jump`、`expectedHeadOid`
- queue 経由で merge された PR(github/docs と NixOS/nixpkgs の 4 件を実測): `state` は MERGED、`headRefOid` は PR の head のまま、`mergeCommit.oid` は base に入ったコミット、`mergedBy` は queue に入れた人
- timeline の `RemovedFromMergeQueueEvent` は `reason`(自由文字列。実測では `merged`、`failed_checks`、`manual`)と `beforeCommit` を持つ。`MergedEvent` との順は決まっていない
- `gh pr view --json` に queue の欄はない(gh 2.103.3 で `isInMergeQueue` は Unknown JSON field)。`gh api graphql` で読む

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| GraphQL の `enqueuePullRequest` で入れる | 入ったか断られたかがはっきりする。auto-merge の予約が残らない | `gh api graphql` の呼び出しを 1 つ足す | ○ |
| `gh pr merge`(queue のある branch) | 今の argv に近い | 即時か予約かを見分けられない。予約が残ると今の `--disable-auto` の掃除と混ざる | |
| REST の `merge-async`(`merge_action: merge_queue`) | REST だけで済む | 2026-10-01 の GA で新しい。権限で断られた報告がある | |
| queue に入れたら待たずに 0 で返す | 単純 | 0 が「記録した merge」の意味を失う(D6 で捨てた形) | |
| 予算切れで dequeue する | queue に残さない | 権限が未確認。進んでいた queue を壊す | |

## Security considerations

- GraphQL の query は固定の文字列にし、変数(owner、name、PR の node ID、head の SHA)は `-f`/`-F` で渡す。node ID と SHA は形を確かめてから渡す
- `--admin` を使わないので、queue と保護を迂回しない
- 外された理由(`reason`)は GitHub から来た自由文字列なので、出すときに制御文字を落とす

## Open questions

- 終了コードの値(案 7)。ほかの `ralph org` の動詞の値(regate の 6、wait の 4・5)とぶつからない値を計画で決める
- `expectedHeadOid` を GitHub が enqueue のときに確かめるか(文書がない)。ralph は入れる前の step 9 で head を確かめ、MERGED のあとに `headRefOid` を確かめるので、確かめなくても記録は誤らない
- `MergeQueueEntry.state` の実際の値の遷移(QUEUED、AWAITING_CHECKS、MERGEABLE、UNMERGEABLE、LOCKED。公開 repo では queue が空で観測できなかった)。ralph は state で判断せず、MERGED か、queue にないかで判断する
- REBASE 方式の queue の `mergeCommit` の形(未観測)。記録は `mergeCommit.oid` をそのまま使う
- `enqueuePullRequest` に要る権限(merge の権限と同じはず、未確認)
- 1 本目と 2 本目の PR のあいだ(1 本目だけが入った main)では、queue のある repo の merge は動くが、2 本目の PR が入るまで merge-order は base の古い branch を `rebase` と出す。director はその間、今までどおり regate の輪で回す

## Decisions(系列の指示で自律に選んだもの)

- D1 queue への入れ方は GraphQL の `enqueuePullRequest`。`gh pr merge` は queue のある branch で即時か予約かを出力から見分けられず、残った予約が今の `--disable-auto` の掃除と混ざるため
- D2 queue のある repo では regate を省き、設定では切れない。issue が「保証の拠り所は queue の必須の検査、doctor の github_protection とそろえる」と方向を決めている。ローカルの gate を最新の base で打ち直したい repo は、queue を使わなければ今の手順のまま
- D3 予算は `--wait-checks` 1 つ。0 なら入れたところで FR-10 の終了コードで返す。director の 2 段の手順(FR-13)はこの形に頼る
- D4 外されたあとの入れ直しは、最新の gate の記録より後に外されていれば断る(FR-5b)。同じ gate の結果での入れ直しは `--requeue` を求める
- D5 PR は 2 本に分ける。1 本目で merge の動詞の queue の道(FR-1〜FR-11、FR-5b、AC1〜AC6、AC8、AC10)、2 本目で regate を省く読みと director(FR-12、FR-13、AC7、AC9、AC11)。1 本目だけでも、queue のある repo で 1 本ずつは merge でき(残りは regate の輪)、価値が出る

## References

- issue #342
- `docs/plans/archive/2026-10-17-org-auto-merge.md`(Non-goals 40 行、Assumptions 53〜56 行、D6、D11)
- https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/merging-a-pull-request-with-a-merge-queue
- https://docs.github.com/en/graphql/reference/pulls(`enqueuePullRequest`、`dequeuePullRequest`、`MergeQueueEntry`)
- https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue
- https://github.com/cli/cli/blob/trunk/pkg/cmd/pr/merge/merge.go
- https://github.com/github/docs/blob/main/data/reusables/pull_requests/merge-queue-removal-reasons.md
- https://github.com/github/docs/blob/main/data/reusables/gated-features/merge-queue.md
