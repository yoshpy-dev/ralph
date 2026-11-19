# org-merge-queue-verb

- Status: Approved
- Approved: 2026-11-20 sha256:7f9b573d4114
- Owner: Claude Code
- Date: 2026-11-20
- Related request: 既定のブランチに merge queue があると `ralph org merge` が断り、director の auto_merge と merge queue を両立できない(#342)。仕様 `docs/specs/2026-11-19-org-merge-queue.md`(A が #348 で main に入れた、Approved)の D5 の 1 本目。issue 系列 #340〜#344 のレーン B
- Related issue: #342
- Type: feat
- Branch: feat/org-merge-queue-verb

## Objective

既定のブランチに GitHub の merge queue と必須の検査がある repo で、`ralph org merge --branch <b>` が今までの確認(step 1〜10、13)を通したあと、PR を GraphQL の `enqueuePullRequest` で queue に入れ、`--wait-checks` の予算の中で MERGED を待ち、`pr_merged`(`via=merge_queue`)を記録する。外されたら記録せずに終了コード 1、予算切れなら queue に残して終了コード 7 で返す。queue のない repo の挙動は変えない。仕様の FR-1〜FR-11、FR-5b、AC1〜AC6、AC8、AC10 を満たす(FR-12、FR-13、AC7、AC9 の skill の部分、AC11 は 2 本目)。

## Scope

- 新しいファイル `internal/org/merge_queue.go`: queue の読みと入れ方
  - PR の queue の状態を 1 つの GraphQL の query で読む(`repository(owner, name).pullRequest(number)` の `id`、`state`、`headRefOid`、`baseRefName`、`mergeCommit { oid }`、`isInMergeQueue`、`mergeQueueEntry { state position enqueuedAt }`、`timelineItems(last: N, itemTypes: [ADDED_TO_MERGE_QUEUE_EVENT, REMOVED_FROM_MERGE_QUEUE_EVENT])` の `createdAt`、`reason`、`actor { login }`)
  - `enqueuePullRequest(input: {pullRequestId, expectedHeadOid})` の mutation。返る `mergeQueueEntry { enqueuedAt position }` を読む
  - query と mutation は固定の文字列。文字列の変数(owner、name、node ID、SHA)は `-f` で渡し、PR の番号は GraphQL の型が `Int!` なので `-F number=<n>` で渡す(`-f` は文字列で送るので型エラーになる。計画助言の指摘)。PR の node ID と SHA は形を確かめてから渡す。`reason` と actor は出す前に制御文字を落とす(`PrintableInboxText`)
- `internal/org/mergeenforce.go`(step 11): GraphQL の `mergeQueue` か rules の `merge_queue` が queue を示せば「queue の道」に入り、断らずに `MergeEnforcement` に queue であることと queue の設定の `mergeMethod`(読めなければ空)を返す(FR-1、FR-3)。queue の道では必須の status check(rules の `required_status_checks` か classic の保護)が 1 つ以上あることを求め、なければ断る(FR-2、AC6)。up-to-date の ruleset の迂回の確認は queue の道では求めない。queue の有無が読めなければ今までどおり断る。mergeQueue の query に `configuration { mergeMethod }` を足す
- `internal/org/merge.go`: step 12(`readMethod`)を queue の道では飛ばし、方法は queue の設定の値を小文字にしたもの、読めなければ `queue` にする(FR-3)。`mergeReady` に queue の道であることを持たせる。`MergeParams` に `Requeue bool` を足す
- `internal/org/merge_run.go`: queue の道の merge
  - 入れる前に PR の queue の状態を読む。queue にあり head が gate の head なら入れずに待つ(FR-5、AC5)。queue になく、その branch の最新の gate の記録(`gate_passed`、`gate_ci_passed`、受け入れた `gate_accepted`)の時刻より後の `RemovedFromMergeQueueEvent` があれば、`Requeue` なしでは断る(FR-5b、AC10)
  - `enqueuePullRequest` を gate の head で 1 回打つ(FR-4)。`gh pr merge` と `--admin` は使わない
  - 入れたあとは `regate.lock` を離し、`--wait-checks` の予算の残りで 30 秒ごとに読み直す(FR-6、FR-7)。予算は今の `mergeWait` と共有し、検査の待ちと queue の待ちを 1 つの予算で区切る
  - MERGED で head が gate の head、base が既定のブランチなら `pr_merged` を記録する。details に `via=merge_queue` を足し、`merge_commit` は `mergeCommit.oid`(FR-8、AC1)。head か base が違えば記録せずに終了コード 1(AC8)。記録は manifest の lock だけで行い(今の `recordMerge` と同じ)、そのあとの worktree の片付けは `regate.lock` を取り直してから行う
  - queue の道の記録は二重にしない(D7): manifest の lock の中で、同じ branch・PR・head の `pr_merged` が最新の gate の記録より後にすでにあれば書かず、記録済みとして扱う。lock を離して待つ 2 つの merge が同じ merge を見ると、今の `recordMerge` は 2 行書き、follow-up の「merge のあとに記録した」の判定(`FollowupRecordedMerge`)を誤らせる(計画助言の指摘)
  - queue になく MERGED でもなければ、入れた時刻より後の `RemovedFromMergeQueueEvent` の `reason` と actor を出して終了コード 1(FR-9、AC2)。理由が読めなくても同じ扱い
  - 予算が 0 か尽きて、まだ queue にあれば、記録せず、PR は queue に残し(dequeue しない)、終了コード 7 で返す(FR-10、AC4)
  - 予算の終わりに状態が読めなければ、記録せず「queue にあるかもしれない」と出して終了コード 1(FR-11)。待ちのあいだの読み失敗は予算の中で読み直す
  - 待ちのあいだの割り込み(Ctrl-C)は、queue に残したまま、記録せずに割り込みのエラーで返す
- `internal/cli/org_merge.go` と `internal/cli/exitcode.go`: `--requeue` の flag、終了コード 7(`ErrMergeQueued` のような sentinel)、queue の道の出力の行(`QUEUED:` など)、help
- テスト(fake の gh): 上の各道。queue のない repo の今の merge のテストは変えずに通す(AC3、NFR-1)
- 文書: README の `ralph org merge` の説明、仕様 `docs/specs/2026-11-19-org-merge-queue.md` の FR-4 の node ID の取り方(D3)、`docs/specs/2026-10-07-org-multi-org-director.md` の 8a-3a の決定への追記、tech-debt(確かめていない GitHub の前提と、実地の確かめを release のあと下流でするというユーザーの決定)、AGENTS.md の internal/org の行に `merge_queue.go`

## Non-goals

- FR-12(merge-order と merge の step 7・13 で regate を省く読み)、FR-13(`/director` の 2 段の手順、`/org` skill)、AC7、AC9 の skill の部分、AC11。2 本目の PR(仕様の D5)
- queue から外す操作(`dequeuePullRequest`)、`--admin` での迂回、queue の設定を ralph が変えること
- doctor の判定の変更
- 実地の確かめ(ユーザーの決定 2: release のあと下流の会社の repo で確かめる。この repo の持ち主は個人アカウントで merge queue を使えない)

## Assumptions

仕様の Research findings で裏を取ったもの(GitHub Docs と公開 repo の実データ)と、取れていないものを分ける。取れていないものは tech-debt に書く(NFR-2)。

- 裏を取った: `enqueuePullRequest` の入力は `pullRequestId`、`expectedHeadOid`、`jump`。queue 経由で merge された PR は `state` が MERGED、`headRefOid` が PR の head のまま、`mergeCommit.oid` が base に入ったコミット。`RemovedFromMergeQueueEvent` は `reason`(自由文字列)を持つ。`gh pr view --json` に queue の欄はない(gh 2.103.3)
- 確かめていない前提: `enqueuePullRequest` は auto-merge の予約を作らない(GitHub Docs は「auto-merge を有効にせず入れる」と書く。打ち直したときに step 9 の auto-merge の断りに当たらない見込み)、`MergeQueueEntry.state` の値の遷移(ralph は state で判断しない)、GitHub が `expectedHeadOid` を enqueue のときに確かめるか(ralph は step 9 と MERGED のあとに head を確かめる)、`enqueuePullRequest` に要る権限、REBASE 方式の queue の `mergeCommit` の形、`RemovedFromMergeQueueEvent` と `MergedEvent` の順、`mergeQueueEntry.enqueuedAt` と timeline の時刻の関係
- gate の記録の時刻(ledger の ts、ralph の時計)と GitHub の `createdAt`(GitHub の時計)を FR-5b で比べる。時計のずれは数秒の範囲とみる
- `recordMerge` は manifest の lock だけで書く(今もそう)。`regate.lock` は worktree の片付けにだけ要る

## Affected areas

- `internal/org/merge_queue.go`(新規)、`internal/org/mergeenforce.go`、`internal/org/merge.go`、`internal/org/merge_run.go`、`internal/org/merge_wait.go`(予算の共有)
- `internal/cli/org_merge.go`、`internal/cli/exitcode.go`
- テスト: `internal/org/merge_queue_test.go`(新規)ほか、`internal/cli/org_merge_test.go` か新しい test ファイル
- 文書: README.md、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`、AGENTS.md
- 共有ファイル(系列の連携): README.md、AGENTS.md、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-merge-queue-verb.html`。かんたんな図 2 枚(`overview`、`easy-wait`)と、詳しい図 3 枚(`detail-overview`、`detail-sequence`、`detail-outcomes`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 5 枚の図を `shot` で撮って読んだ。順序の図で、手順の番号の「7 へ」と終了コードの 7 が紛らわしかったので「手順 7 へ」「終了コード 7」と書き分け、撮り直して確かめた
- 計画助言の指摘を入れたあと、順序の図(手順 7 に「まだなければ」)と終わり方の一覧(5 分の幅)を直し、lint(exit 0)と撮り直しをした

## Design decisions

Critical forks: 6 件。consult(consult-plan-mergequeue、判定「直してから進める」、直す点 1 つを入れた)に聞き、自律で選んだ。

- D1 待ちのあとの記録と片付けの lock: 入れたあと `regate.lock` を離して待ち(仕様 FR-6)、MERGED を読んだら manifest の lock だけで `pr_merged` を記録する(今の `recordMerge` と同じ)。片付けの前に `regate.lock` を取り直し、`mergeLocked` と同じく marker を空にしてから `cleanupFeatureWorktreeLocked` を呼ぶ。取れなければ片付けを飛ばし、`ralph org cleanup` を名指す。cleanup の最初の確かめ(`cleanupFirstCheck`)は binding と running だけを見て、regate は PR の状態を live に読むので、`pr_merged` を `regate.lock` なしで書いても衝突しない(consult が確かめた)。待つあいだも lock を持つ案は FR-6 に反し、ほかの branch の merge を止める
- D2 Merge の構造: `mergeLocked` が「queue の待ち」を表す値を返し、`Merge` が lock の外で queue を待ち、終わったら記録と片付けをする新しい関数を呼ぶ。今の `mergeWaitFor` に乗せて待ちのあと `mergeLocked` を最初から呼び直す形は、PR が MERGED になると step 9 が断るので合わない。予算は今の `mergeWait` を共有する(最初の待ちの開始から数える)
- D3 PR の node ID: queue の状態を読む 1 つの GraphQL の query で `id` も読む。呼び出しが 1 つ減り、id と状態が同じ答えから来る。仕様 FR-4 は `gh pr view --json id` と書いているので、仕様の文も同じ PR で直す(計画からのずれではなく、仕様の文の更新)
- D4 外されたことの判定の時刻: FR-9(入れたあとに外された)は、mutation が返す `mergeQueueEntry.enqueuedAt`(GitHub の時計)より後の `RemovedFromMergeQueueEvent` で見る(GitHub の時計どうし)。FR-5 で入れずに待つときは、読んだ entry の `enqueuedAt` を使う。FR-5b(最新の gate の記録より後に外された)は ledger の ts(ralph の時計)と GitHub の `createdAt` を比べるしかない。計画助言の指摘を受けて、時計のずれで「gate より前に外された」と読み違えて `--requeue` なしに入れ直す向きを塞ぐ: gate の ts から `mergeQueueClockSkew`(5 分)を引いた時刻より後に外されていれば、gate より後とみなして断る。迷う側は断る向きで、正しい gate のあとでも断られたら `--requeue` で抜けられる。外されてから 5 分以内に取り直した gate の PR は `--requeue` が要る
- D7 記録を二重にしない: queue の道の `pr_merged` は、manifest の lock の中で同じ branch・PR・head の `pr_merged` を探し、最新の gate の記録より後にあれば書かない(記録済みとして返し、出力にそう書く)。queue のない道の `recordMerge` は変えない(`regate.lock` の中で gh pr merge を 1 回打つので二重にならない。NFR-1)
- D5 予算 0: `enqueuePullRequest` の答えの `mergeQueueEntry` を queue に入った証拠とし、読み直さずに終了コード 7 で返す(仕様 FR-7「入れたところで返す」、consult の指摘)。FR-5 で入れずに続けるときの予算 0 も、直前に読んだ状態で 7。mutation の答えに `mergeQueueEntry` がなければ(errors、null)、入ったか分からないので記録せず終了コード 1
- D6 終了コード 7 は cli 側の sentinel(`ErrMergeQueued`)で、`MergeResult` に queue に残ったことを示す欄を足す。`--requeue` は `MergeParams.Requeue`。待ちのあいだの Ctrl-C は割り込みのエラー(終了コード 1)で、文に「queue に残っている、打ち直せば待ちを続ける」を終了コード 7 と同じ言葉で書く
- FR-2 の「必須の検査 1 つ以上」は、`readMergeRules` の `rules.checks`(strict でない `required_status_checks` の規則の検査も数える)と classic の保護の必須の検査の数で見る。拡張は要らない(コードで確かめた)

## Acceptance criteria

- [ ] AC1: queue と必須の検査がある既定のブランチ(fake の gh)で `--wait-checks 30m` のとき、`enqueuePullRequest` を gate の head で 1 回打ち、MERGED を読んだあと `pr_merged`(`via=merge_queue`、`merge_commit` は `mergeCommit.oid`)を記録して終了コード 0
- [ ] AC2: 入れたあとに `RemovedFromMergeQueueEvent`(reason `failed_checks`)が出たら、記録せず reason を出して終了コード 1
- [ ] AC3: queue のない repo では今の merge のテストがすべて変わらずに通る
- [ ] AC4: `--wait-checks 0`、または予算の中で MERGED にならなければ、記録せず、queue に残したまま終了コード 7。予算 0 は `enqueuePullRequest` の答え(入れずに続けるときは直前の読み)で決め、読み直さない(D5)
- [ ] AC5: すでに queue にある PR(head は gate の head)を打ち直すと、`enqueuePullRequest` を打たずに待ちから続ける
- [ ] AC6: queue があり必須の検査がなければ断る(終了コード 1、記録なし)
- [ ] AC8: MERGED だが `headRefOid` が gate の head と違えば記録せずに終了コード 1。base が既定のブランチと違っても同じ
- [ ] AC10: 最新の gate の記録より後に `RemovedFromMergeQueueEvent` がある PR は、`--requeue` なしでは入れずに断る(終了コード 1)。`--requeue` があれば入れる
- [ ] AC12: 待ちの終わりに queue の状態が読めなければ、記録せず「queue にあるかもしれない」と出して終了コード 1(FR-11)。待ちのあいだの読み失敗は予算の中で読み直す
- [ ] AC13: queue の道の gh は `--hostname <解いた host>` と `GH_PROMPT_DISABLED=1` で打ち、enqueue のあとの呼び出しは `runDetached` を通る。GraphQL の文字列の変数は `-f`、PR の番号は `-F` で渡し(query の変数の宣言 `$number: Int!` と送る引数をテストで確かめる)、形の確かめに通らない node ID と SHA は渡さない
- [ ] AC15: lock を離して待つ 2 つの merge が同じ MERGED を見ても、`pr_merged` は 1 行だけ書かれる(2 つ目は書かずに記録済みとして返す)。テストで確かめる
- [ ] AC16: FR-5b の時刻の比べ方は、外された時刻が gate の ts より前・同じ・後のそれぞれと、時計のずれの幅(5 分)の内と外をテストで確かめる。幅の内なら断る
- [ ] AC14: README の `ralph org merge` の説明と `ralph org merge --help` が、queue の道、`--requeue`、終了コード 7 を書く。tech-debt に確かめていない前提と実地の確かめの予定を書く

## Implementation outline

1. S1 queue の読みと入れ方: `internal/org/merge_queue.go` に PR の queue の状態の query と `enqueuePullRequest` の mutation の引数づくりと答えの読みを足す(純粋な関数と、`MergeOrderExec` で打つ薄い関数)。node ID と SHA の形の確かめ、`reason` と actor の制御文字の扱い。表のテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'MergeQueue'`
2. S2 step 11・12 の queue の道: `CheckMergeEnforcement` が queue を示すとき断らずに queue の道と queue の設定の `mergeMethod` を返し、必須の検査がなければ断る。mergeQueue の query に `configuration { mergeMethod }` を足す。`readMethod` を queue の道で飛ばす。`mergeReady` に queue の道を持たせる。今の merge と enforcement のテストを変えずに通し、queue の道のテスト(AC6)を足す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Merge|Enforce'`
3. S3a 入れるまで: merge_run の queue の道で、入れる前の読み(FR-5 の入っていれば入れない、FR-5b の外されたあとは `Requeue` なしで断る)、`enqueuePullRequest`、予算 0 の終わり(D5)。`MergeParams.Requeue` と `MergeResult` の queue の欄。FR-5b の時計のずれの幅(D4)。AC4(予算 0)、AC5、AC10、AC16 のテスト — verify: 同上の `-run 'Merge'`
4. S3b 待ちと記録: lock を離した待ち(予算の共有、30 秒、読み失敗の読み直し)、MERGED の確かめと `pr_merged`(`via=merge_queue`、二重にしない D7)、外されたとき、予算切れ、読めないとき、割り込み、lock を取り直しての片付け(D1、D2)。AC1、AC2、AC4(予算切れ)、AC8、AC12、AC13、AC15 のテスト — verify: 同上の `-run 'Merge'`
5. S4 CLI: `--requeue`、終了コード 7(`ErrMergeQueued`、exitcode.go)、queue の道の出力の行、help。CLI のテスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgMerge|ExitCode'`
6. S5 文書: README の `ralph org merge` の説明、仕様の FR-4 の node ID の文(D3)、`docs/specs/2026-10-07-org-multi-org-director.md` の 8a-3a の決定への追記、tech-debt(確かめていない前提、実地の確かめは release のあと下流)、AGENTS.md の internal/org の行に `merge_queue.go` — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: 仕様の FR-1〜FR-11、FR-5b、NFR-1〜NFR-4、AC1〜AC6、AC8、AC10 と、この計画の AC12〜AC14
- Documentation drift to check: README の merge の段落、`ralph org merge --help`、仕様と 8a-3a の決定、tech-debt、AGENTS.md
- Evidence to capture: verify の report

## Test plan

- Unit tests: GraphQL の query と mutation の引数、答えの読み(MERGED、queue にある、外された、errors、null、形の悪い node ID と SHA)、`reason` の制御文字、step 11 の queue の道(GraphQL と rules のどちらで queue を見ても、必須の検査なしで断る、mergeMethod の読み)
- Integration tests: `Org.Merge` を fake の gh で通す。AC1、AC2、AC4、AC5、AC8、AC10、AC12、割り込み、待ちのあいだの lock、片付け
- Regression tests: 今の merge のテスト(queue のない repo)を変えずに通す
- Edge cases: 予算 0、予算の途中で外される、読み失敗が続く、queue にあるが head が違う、MERGED の base が違う、`mergeCommit` がない
- Evidence to capture: test の report

## Risks and mitigations

- 実地で確かめられない(この repo は merge queue を使えない)。fake のテストで実装し、GitHub の挙動の裏を文書と公開 repo の実データで取り、取れない前提を tech-debt に書く。release のあと下流で確かめる(ユーザーの決定 2)
- 分からない状態で `pr_merged` を記録すると、merge されていない PR を merge 済みと記録する。MERGED を読み、head と base が合うときだけ記録する(NFR-3)
- 予算切れで queue に残した PR が、あとで GitHub で merge される。ralph はその merge を記録しない。終了コード 7 の文で「打ち直せば待ちを続ける、merge されたら `ralph org cleanup` が片付ける」と言う
- 待ちのあいだ `regate.lock` を離すので、同じ branch の regate や cleanup が走りうる。片付けの前に lock を取り直し、取れなければ片付けを飛ばして `ralph org cleanup` を名指す
- FR-5b の時刻の比べ方は ralph の時計と GitHub の時計をまたぐ。5 分の幅を取って断る向きにしたので、ずれが 5 分を超えると `--requeue` なしに入れ直す向きの読み違いが残る(D4)。幅の内の正しい gate のあとは `--requeue` で抜けられる
- lock を離して待つ 2 つの merge が同じ merge を記録する。queue の道の記録は manifest の lock の中で確かめてから書く(D7)
- GraphQL の型の誤り(番号を文字列で送るなど)は fake のテストでは見えない。query の変数の宣言と送る引数の型(`-f` か `-F`)をテストで固定する(AC13)。実地は release のあと下流で確かめる

## Rollout or rollback notes

- 次の release に入る。queue のない repo の挙動は変わらない。release ノートの候補は PR に書く
- 戻すときは revert する

## Open questions

- なし(分かれ目は Design decisions に書く)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 3 件、すべて取り入れた: [HIGH] GraphQL の `number` は `Int!` なので `-f` では型エラーになる → 番号は `-F`、引数の型をテストで固定(Scope、AC13、Risks)。[MEDIUM] FR-5b の時計のずれで `--requeue` なしの入れ直しを許す → 5 分の幅を取って断る向き(D4、AC16、Risks)。[MEDIUM] lock を離した 2 つの待ちが同じ merge を 2 回記録する → queue の道の記録は manifest の lock の中で確かめる(Scope、D7、AC15、Risks)
- [x] Plan approved — ユーザーの指示(「計画は推す案で自動承認する」)により、推す案で自動承認した
- [x] Branch created — `feat/org-merge-queue-verb`(worktree `.claude/worktrees/org-merge-queue-verb`)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
