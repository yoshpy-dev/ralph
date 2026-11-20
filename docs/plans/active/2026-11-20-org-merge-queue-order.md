# org-merge-queue-order

- Status: Approved
- Approved: 2026-11-20 sha256:357148b1a947
- Owner: Claude Code
- Date: 2026-11-20
- Related request: 既定のブランチに merge queue がある repo で、merge-order と merge が gate の base の古いブランチを `rebase` と読み、director が merge のたびに regate の輪を回す(#342)。仕様 `docs/specs/2026-11-19-org-merge-queue.md` の D5 の 2 本目。1 本目(merge の動詞の queue の道)は PR #352(merge commit 1d05c9ac)で main に入った。issue 系列 #340〜#344 のレーン B
- Related issue: 342
- Type: feat
- Branch: feat/org-merge-queue-order

## Objective

既定のブランチに merge queue がある repo で、gate を通った(または受け入れられた)ブランチを、gate のあとに既定のブランチが進んでも `rebase` にせず `ready`(`accepted`)と読み、`ralph org merge` が regate なしで queue に入れられるようにする(FR-12)。director の skill に、queue のある repo では ready のブランチをまとめて queue に入れてから 1 本ずつ待つ 2 段の手順を書く(FR-13)。仕様の FR-12、FR-13、AC7、AC9 の skill の部分、AC11 を満たし、#342 を閉じる。

## Scope

- queue の有無の読み(共通): 1 本目の `readMergeQueue`(`internal/org/mergeenforce.go`、GraphQL の `mergeQueue(branch)`)を、`CheckMergeEnforcement` の外からも呼べる形にする。結果は「ある(方法つき)」「ない」「読めない(理由つき)」の 3 つ。`MergeEnforcementParams` に読み済みの結果を渡す欄を足し、渡されたら step 11 は読み直さない(FR-12「merge は queue の有無を 1 回だけ読む」)
- 分類(`classifyGatePassed`、`internal/org/mergeorder.go`): 分類に queue の状態を渡す。queue が「ある」とき、gate の記録の base_ref が今の既定のブランチを名指し、base が既定のブランチの先端と違うだけなら、`rebase` にせず `ready`(受け入れのあるブランチは `accepted`)とし、理由に queue が最新の既定のブランチの上で検査してから merge することを書く。base_ref が別のブランチを名指すとき、先端が gate の head と違うとき(`regate`)、PR の統合先が違うとき(`wrong_base`)は今までどおり。queue が「ない」「読めない」なら今までどおり `rebase`
- `ralph org merge-order`(FR-12): 対象のブランチがあり ls-remote が通ったら、origin の repository(`ReadMergeOrigin`、#340 の `ssh -G` の解決つき。merge-order の CLI は今 `Origin` を渡していないので、MergeOrder の中で読む)と既定のブランチの queue を 1 回読み、分類に使う。origin が読めないときも queue は「読めない」とする。結果を `MergeOrderResult` の新しい欄(JSON `merge_queue`)に入れ、テキストでは queue が「ある」か「読めない」ときだけ `merge queue:` の行を `merge guarantee:` の行のあとに出す(「ない」ときは今の出力のまま)。「読めない」ときは注記で、base の古いブランチを `rebase` と出したことを言う。queue を読むのは `merge-order` の動詞だけで、`ralph org cleanup` が中で使う merge-order は読まない(`MergeOrderParams` の新しい欄で選ぶ)
- merge-order の注記(`mergeOrderListNotes`): ready と accepted が 2 本以上あるときの「1 本 merge すると他は rebase になる」の注記は、queue が「ある」ときは出さず、queue の 2 段の手順(まとめて queue に入れてから待つ)を言う注記にする(consult の指摘)
- `ralph org merge`(FR-12): step 7 は、queue なしで分類した結果が queue ありなら変わるときだけ(base が動いただけで `rebase` のとき)queue を読み、その結果で分類する。これを名前のある関数にし、`checkMerge` の doc の step 7 に書く(隠れた制御フローにしない)。読んだ結果は step 11 に渡し、step 11 はそれを使う(まだ読んでいなければ今までどおり step 11 で読む)。どちらでも queue の GraphQL は 1 回だけ。step 7 で queue が読めず `rebase` として断るときは、断りに読めなかった理由を添える(添えないと、gh の認証切れなどが regate を勧める断りに化ける。consult の指摘)。step 13(`checkOriginUnmoved`)は queue の道(`r.queue`)では既定のブランチの先端が gate の base であることを確かめず、既定のブランチの名前とブランチの先端が gate の head であることは今までどおり確かめる
- `ralph org merge` の `requeueNext` の doc comment(「既定のブランチは gate のときのまま」)を FR-12 に合わせて直す
- テスト(fake の gh): 上の各道、AC11(director の 2 段の手順を cli の層でなぞる)
- 文書(FR-13、AC9 の skill の部分): `/director` skill の「merge と auto_merge」に queue のある repo の 2 段の手順(1 段目の `--wait-checks 0` では、org がまだ走っている断り(step 8)と、検査が pending か 0 件の断り(step 10)が待たずに終了コード 1 になるので、その 2 つは止めずに 2 段目へ回す。2 段目で待つ間に queue がほかの PR を merge すると、そのブランチの merge は step 7 で `merged` と分類されて断られるので、`ralph org cleanup --branch <ブランチ>` で片付ける)、`/org` skill の「merge してよい順」と動詞リファレンスの `merge-order` の行、README の `ralph org merge-order` の説明、4 写し(`scripts/sync-skills.sh`)。仕様の FR-12、FR-13、AC7、AC9、AC11 の印、`docs/specs/2026-10-07-org-multi-org-director.md` の該当の決定への追記、tech-debt の関係する行

## Non-goals

- 1 本目の範囲(enqueue、待ち、記録、終了コード 7、`--requeue`)の挙動を変えること。step 13 と step 7 の queue の扱いだけを変える
- ruleset の `merge_queue` の rule だけがあり GraphQL の `mergeQueue` が null の repo で、step 7 を緩めること(step 11 は今までどおり rule でも queue の道に入るが、step 7 と merge-order は GraphQL の答えだけで決める)
- `ralph org regate` と doctor の挙動の変更
- 本物の GitHub での実走(ユーザーの決定 2: release のあと下流で確かめる)

## Assumptions

- queue は PR を最新の既定のブランチの上で必須の検査にかけてから merge するので、gate の base が古くても merge の時点の保証は queue が持つ(仕様 Constraints、doctor の `github_protection` と同じ拠り所)
- director は merge-order の `merge queue:` の行で queue のある repo かを知る。前の版の ralph はその行を出さないので、行がなければ今の 1 本ずつの手順になり、前の版との組み合わせでも誤らない
- GraphQL の `mergeQueue(branch)` が null なら queue はない(1 本目の `readMergeQueue` と同じ読み)

## Affected areas

- `internal/org/mergeenforce.go`(queue の読みを外から呼べる形、読み済みの結果の受け取り)
- `internal/org/mergeorder.go`(`mergeRemote` と `classifyGatePassed`、`MergeOrderParams`、`MergeOrderResult`)、新しいファイル `internal/org/mergeorder_queue.go`(merge-order の queue の読みと注記。`mergeorder.go` は 1447 行で目安を超えているので足さない)
- `internal/org/merge.go`(step 7 の読み、step 11 への受け渡し、step 13)、`internal/org/merge_queue_run.go`(doc comment)
- `internal/cli/org_merge_order.go`(読む欄を立てる、`merge queue:` の行、help)
- テスト: `internal/org/mergeorder_queue_test.go`(新規)、`internal/org/merge_queue_order_test.go`(新規、merge の step 7・13 と AC4 の 2 段の手順)、`internal/cli/org_merge_order_test.go`、既存の fake(`moExec` に GraphQL の答えを足す)
- 文書: `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と 3 写し、README、仕様 2 つ、tech-debt

## Visual review

- ページ: `.harness/state/plan-visual/org-merge-queue-order.html`。かんたんな図 2 枚(`overview`、`easy-director`)と、詳しい図 3 枚(`detail-overview`、`detail-merge`、`detail-classify`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 5 枚の図を `shot` で撮って読んだ。重なり、はみ出し、線が文字を通る所はない

## Design decisions

Critical forks: 3 件。consult(consult-plan-queueorder、判定「直してから進める」。3 件とも推す案でよいとし、Scope と AC の穴 3 つを指摘した。すべて入れた)に聞き、自律で選んだ。

- D1 merge-order が queue を読むとき: `merge-order` の動詞のときだけ、対象のブランチがあり ls-remote が通れば毎回 1 回読み、queue が「ある」か「読めない」ときだけ `merge queue:` の行を出す。director は 1 段目に入る前に queue の有無を知る必要があり、base の動いていない ready しかないとき(1 段目の典型)にも分かる。前の版の ralph は行を出さないので、行がなければ 1 本ずつの手順になり、組み合わせでも誤らない。退けた案: 分類が queue で変わるときだけ読む(director が 1 段目を選べない)、「ない」ときも行を出す(既存の出力のテストがほぼすべて変わる)
- D2 merge が queue を 1 回だけ読む形: step 7 で、queue「なし」と「あり」で分類が変わるとき(base が動いただけで `rebase`)だけ読み、結果を step 11 に渡す。読まなかったときは今までどおり step 11 で読む。分類の関数は queue の状態を受け取る純粋な関数のまま 2 回呼び、読むかどうかは名前のある関数が決め、`checkMerge` の doc に書く。queue のない repo で base の動いていない merge の呼び出しの並び(`mxAll`、`after(n)` の番号)が変わらない。退けた案: step 7 の前に必ず読む(既存の merge のテストが広く揺れ、queue のない repo でも呼び出しの順が変わる)
- D3 step 13 の緩め方: queue の道では既定のブランチの先端が gate の base であることだけを確かめない。既定のブランチの名前(queue を読んだブランチ)と、ブランチの先端が gate の head であることは今までどおり確かめる
- D4 GraphQL と ruleset の非対称: step 7 と merge-order は GraphQL の `mergeQueue` だけで queue の有無を決め、ruleset の `merge_queue` の rule だけの repo では base の動きを許さない(step 11 は今までどおり rule でも queue の道に入る)。GraphQL の `mergeQueue` は保護ブランチの queue も見る(`internal/org/mergeenforce.go` の `CheckMergeEnforcement` の doc、step 2)ので、rule だけがあり GraphQL が null の repo はまれで、そのときは安全側(`rebase`)に倒れる
- D5 2 段目で queue が先に merge した PR: merge の step 7 は PR が MERGED なら `merged` と分類して断る(今の挙動)。queue が merge したブランチは、1 本目の skill の文のとおり `ralph org cleanup --branch <ブランチ>` で片付ける(台帳の `pr_merged` は書かれず、`worktree_cleaned` が残る)。merge がそれを `pr_merged` として記録する形は範囲の外にし、tech-debt に書く

## Acceptance criteria

- [x] AC1(仕様 AC7 の merge-order): queue が「ある」既定のブランチで、gate の記録の base が既定のブランチの先端より古いブランチを、merge-order は `ready`(受け入れたブランチは `accepted`)と出し、理由に queue を書く。queue が「ない」なら今までどおり `rebase`。「読めない」なら `rebase` と注記。base_ref が別のブランチを名指す記録は queue があっても `rebase`
- [x] AC2(仕様 AC7 の merge): queue のある既定のブランチで base の古いブランチに `ralph org merge` を打つと、regate なしで step 7 と step 13 を通って queue に入れる。ブランチの先端が gate の head と違えば今までどおり断る(step 7 の `regate`、step 13)。queue の GraphQL は 1 回だけ打たれる。step 7 で queue が読めず `rebase` として断るときは、断りに読めなかった理由が入る
- [x] AC3: queue のない repo で、base の動いていないブランチの merge の gh・git の呼び出しの並びは今と同じ(既存の merge のテストが変わらずに通る)。base の動いたブランチは今までどおり step 7 で `rebase` として断る(呼び出しは queue の読みが 1 つ増えるだけ)
- [x] AC4(仕様 AC11): queue のある repo で ready の 3 本に、director の 1 段目のとおり `--wait-checks 0` で merge を打つと、`enqueuePullRequest` が 3 回(1 本に 1 回)打たれ、どれも PR を queue に残した結果(`MergeResult.Queued`。CLI はこれを終了コード 7 にする。既存の `TestOrgMerge_PrintsAQueuedResultAndExitsSeven`)で返る。2 段目として、1 本目が merge されて既定のブランチが進んだあと 2 本目に予算を付けて打つと、step 7 と 13 を通り、queue にあるので入れずに待ち、MERGED を記録する。後回しの打ち直しとして、A の検査が pending のまま B の検査が通ったとき、A の `--wait-checks 0` は何も入れずに断り、B は queue に入り、そのあと A の検査が通ってから打ち直すと A が入る(`enqueuePullRequest` は B、A の順。計画助言の指摘)。3 本のブランチの fake が要るので org の層(Go の fake の gh)で書く
- [x] AC5: `ralph org merge-order` は queue が「ある」か「読めない」ときだけ `merge queue:` の行を出し、`--json` には `merge_queue` が出る。origin が読めないときは「読めない」。対象のブランチがないときと、`ralph org cleanup` の中では queue を読まない。queue が「ある」とき、ready と accepted が 2 本以上でも「1 本 merge すると他は rebase になる」の注記は出ず、queue の 2 段の手順を言う注記が出る
- [x] AC6(仕様 FR-13、AC9 の skill の部分): `/director` skill の「merge と auto_merge」が、`merge queue:` の行が queue を示すときの 2 段の手順を書き、queue のない repo の 1 本ずつの手順を残す。2 段の手順: 1 段目は ready と accepted をすべて `--wait-checks 0` で入れて終了コード 7 を集める。org がまだ走っている断りと、検査が pending か 0 件の断り(どちらも `--wait-checks 0` では待たずに終了コード 1)は止めずに後回しにし、後回しのブランチが残る間は受信箱の wait を 5 分で区切り、wait が戻るたびに merge-order を打ち直して後回しのブランチに `--wait-checks 0` を打ち直す(入れられるようになったブランチから順に queue に入る。ほかのブランチの待ちに止められない。計画助言の指摘)。最初に後回しにしてから 60 分たっても入らないブランチは、ほかの断りと同じく人に伝える。後回しのブランチがなくなってから 2 段目に入り、1 本ずつ予算付き(`--wait-checks 60m`)で打ち直す。2 段目の 1 本が終わるたびに merge-order を打ち直し、新しく ready になったブランチがあれば先に 1 段目を打つ。queue が先に merge したブランチ(merge が `merged` と分類して断る)は `ralph org cleanup --branch <ブランチ>` で片付ける。regate の輪は回さない。終了コード 7 は打ち直してよい。FR-5b の断りは人に伝え、`--requeue` は人に頼まれたときだけ。`merge queue: unknown` の行なら 1 本ずつの手順で進める。`/org` skill の「merge してよい順」と `merge-order` の行が、queue のある repo では base の古いブランチも `ready` になることと `merge queue:` の行を書く。4 写しが一致する(`scripts/check-skill-sync.sh`)
- [x] AC7: step 13 は queue の道で、既定のブランチの先端が gate の base から動いていても断らない。既定のブランチの名前が変わったとき、ブランチの先端が gate の head から動いたときは今までどおり断る

## Implementation outline

1. S1 queue の読みと分類(org): `readMergeQueue` を外から呼べる関数にし、`MergeEnforcementParams` に読み済みの結果の欄を足す。`mergeRemote` に queue の状態を持たせ、`classifyGatePassed` が queue のある既定のブランチで base の動きを許す。AC1 の分類の部分、AC7 の前提 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'MergeOrder|Enforce|Classify'`
2. S2 merge-order の queue の読み(org): `mergeorder_queue.go`、`MergeOrderParams` の欄、`MergeOrderResult.MergeQueue`、読めないときの注記、cleanup は読まない。`moExec` に GraphQL の答え。AC1、AC5 の org の部分 — verify: 同上の `-run 'MergeOrder|Cleanup'`
3. S3 merge の step 7・11・13(org): 1 回だけの読み、step 7 の読めなかった理由、step 13 の queue の道、`requeueNext` の doc。AC2、AC3、AC4、AC7 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Merge'`
4. S4 CLI(cli): merge-order が読む欄を立てる、`merge queue:` の行、`--json`、help。AC5 の CLI の部分 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgMerge'`
5. S5 文書: director と org の skill(`scripts/sync-skills.sh` で写す)、README、仕様 2 つ、tech-debt。AC6 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill|Director'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、同期の検査)
- Spec compliance criteria to confirm: 仕様の FR-12、FR-13、AC7、AC9、AC11。1 本目の FR-1〜FR-11 の挙動が変わっていないこと
- Documentation drift to check: README の merge-order と merge の説明、org と director の skill の 4 写し、help、仕様の印、AGENTS.md の repo map(新しいファイル)
- Evidence to capture: verify の report と log

## Test plan

- Unit tests: 分類の表(queue ある・ない・読めない × base が同じ・動いた・別のブランチ × ready・accepted・regate・wrong_base)。merge-order の queue の読みの各結果と注記、出力の行
- Integration tests: merge の step 7 と 13 を fake の gh で通す(base の動いた queue のある repo、先端が動いたブランチ、既定のブランチの改名)。AC4 の cli の 2 段のテスト
- Regression tests: queue のない repo の merge の呼び出しの並び(`mxAll`)と既存の merge-order のテストが変わらない。1 本目の queue の道のテストが通る
- Edge cases: queue を読めない(gh の失敗、errors、null でない壊れた答え)、origin を読めない(merge-order)、受け入れたブランチ、queue のある repo で gate の base_ref が別のブランチ
- Evidence to capture: test の report と log、変異(queue の扱いを外すと落ちること)

## Risks and mitigations

- queue のある repo で base の古いブランチを `ready` にすると、queue が検査に落としたときに外される。外されたら 1 本目の FR-9 のとおり記録せず終了コード 1 で人に伝わるので、黙って誤った merge にはならない
- merge-order の gh の呼び出しが 1 つ(と origin の読み)増える。対象のブランチがあるときだけ、1 回だけ読む
- テストの fake の呼び出しの並びが変わると既存のテストが大きく揺れる。merge は base が動いたときだけ step 7 で読むので、base の動いていない既存のテストの並びは変わらない(AC3)
- director の手順は文書なので、実地の確かめは release のあと下流で(ユーザーの決定 2)
- 後回しのブランチを予算付きの merge で 1 本ずつ待つと、先に検査が通ったほかのブランチが、前のブランチの待ちが終わるまで queue に入れない(計画助言の指摘)。後回しのブランチは `--wait-checks 0` を wait の区切りごとに打ち直す形にした。予算付きの merge を並べて打つ形は、`regate.lock` を 1 秒しか待たない(`regateLockWait`)ので、確かめの時間が重なると断りになり、採らない
- 2 段目で queue が先に merge した PR は `pr_merged` が書かれず、cleanup の `worktree_cleaned` だけが残る(D5)。follow-up の「merge のあと」の判定は、status が cleanup を数え、merge-order が gh で読むので、どちらも merge を見落とさない。merge が記録する形は tech-debt に書く

## Rollout or rollback notes

- revert すれば 1 本目だけが入った状態に戻る(queue のある repo でも base の古いブランチは `rebase`、director は regate の輪で回す)
- release ノート: queue のある repo では merge-order が base の古いブランチも `ready` と出し、`merge queue:` の行が出る。director は 2 段の手順で merge する

## Open questions

- なし(分かれ目は Design decisions)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 1 件、取り入れた: [MEDIUM] 後回しのブランチを 2 段目で 1 本ずつ予算付きで待つと、先に検査が通ったほかのブランチが前の待ちの終わりまで queue に入れない → 後回しのブランチは受信箱の wait を 5 分で区切って `--wait-checks 0` を打ち直す形にし(AC6、Risks)、A が pending のまま B が先に入るテストを AC4 に足した。予算付きの merge を並べて打つ形は `regate.lock` の 1 秒の待ちでぶつかるので採らなかった
- [x] Plan approved — ユーザーの指示(「計画は推す案で自動承認する」)により、推す案で自動承認した
- [x] Branch created — `feat/org-merge-queue-order`(worktree `.claude/worktrees/org-merge-queue-order`)
- [x] Implementation started
- [x] S1(4d96a0a6): `mergeQueueLookup`(状態 `""`・`absent`・`present`・`unknown`、`mergeQueueRead` は 1 本目の型と名前がぶつかるので改名)、`lookupMergeQueue`(`CheckMergeEnforcement` と同じ引数、共通の `newMergeEnforcementRun`)、`MergeEnforcementParams` の非公開の欄 `queue`、`classifyGatePassed` と `classifyAccepted` の queue の文、新しい `mergeorder_queue.go`(理由文の 2 関数)。検証コマンドとパッケージ全体(750 秒)が通り、変異 8 個すべて検出。自己点検 5 項目 clean。S3 への申し送り: 読み済みの結果は `DefaultBranch` を確かめないので同じブランチで読む、step 7 の読みも `checkMergeAfterChecks` の gh の見つからなさを拾う包んだ exec で呼ぶ
- [x] S2(5782fd38): `MergeOrderParams.ReadMergeQueue`、`MergeOrderResult.MergeQueue`(JSON `merge_queue`、`state`・`method`・`why`)、公開の定数 `MergeOrderQueuePresent`・`Absent`・`Unknown`、読む位置は `lookupMergePRs` のあと `orderMergeEntries` の前、時間の区切りは `p.GHTimeout`(既定 15 秒)、origin は `ReadMergeOrigin`(`p.Origin` があればそれ)、読めないときの注記、queue があるときの 2 段の手順の注記。パッケージ全体(734 秒)が通り、変異 12 個すべて検出。自己点検は誤字 2 か所を直して clean
- [x] S3(285e3827): `merge_queue_order.go`(新規。step 7 の `classify`、`readQueueIfOnlyBaseMoved`、`queueReadFor`、gh の見つからなさを拾う `runNotingGHNotFound` を step 7 と 11 で共用)、step 11 に読み済みを渡す、step 13 は queue の道で base を見ない。変異 10 個すべて検出。実装の注(計画の本文は変えていない): (1) FR-12 で queue のある repo は既定のブランチが進んでも打ち直しが続くので、1 本目の `queueNext` の文から「while origin's default branch is where the gate left it,」を外し、`queueNext` と `requeueNext` の doc を直した(固定の `mqrNext` も)。CLI の help と README の同じ条件は S4・S5 で直す。(2) AC3 のとおり base の動いた queue のない repo では step 7 の前に GraphQL の読みが 1 つ増えるので、`merge_test.go` の "not ready: rebase" の行(呼び出しを `mxAll[:3]` に固定)を外し、同じ断りと呼び出しの並びを `TestCheckMerge_NoQueueKeepsTodaysCommands/base_moved` で確かめる形にした(handoff の範囲を広げて implementer が直した)。`merge.go` は 843 行
- [x] S4(8d3baa66): merge-order が `ReadMergeQueue` を立て、`merge queue:` の行(`mergeQueueLine`、present は `origin/<def> has one (method <小文字>): a branch whose gate base origin/<def> moved past stays ready or accepted, and ralph org merge adds its PR to the queue`、unknown は `unknown (<why>)`)、merge-order と merge の help(終了コード 7 の段落の条件を外し、step 13 は queue の道で base を見ない)。パッケージ全体(675 秒)が通り、変異 5 個すべて検出。実装の注: テストの fixture の origin はローカルのパスなので queue は unknown になり、既存の merge-order の CLI テスト 7 本の期待出力に `merge queue: unknown (...)` の行と注記が増えた(fixture の origin を GitHub の URL に替えると PR の照合で広く落ちるため)。製品でも、origin が GitHub の repository を名指さない repo では merge-order が毎回 unknown の行と注記を出す(S2 の決めたとおり)。`internal/cli/exitcode.go` のコメントに残る古い条件は S5 で直す
- [x] S5(6bcdff45): director skill の「merge と auto_merge」に queue のある repo の 2 段の手順(`merge queue:` の行で選ぶ、1 段目の後回しの 2 つの断りの見分け方、wait を 5 分で区切る打ち直し、60 分で人に伝える、2 段目、queue が先に merge したブランチの cleanup、regate を名指すブランチ)、終了コード 1 と 7 の箇条、org skill の `merge-order` と `merge` の行と「merge してよい順」、README、exitcode.go のコメント、仕様の印(FR-12、FR-13、AC7、AC9、AC11)、2026-10-07 の仕様の注、tech-debt(D5 の行、`mergeorder.go` と `merge.go` の行の持ち越し)、AGENTS.md。implementer が足したもの(AC6 の外): 2 段目で続けて 2 回終了コード 7 なら人に伝えて止める、push だけの wake の director も後回しの間は wait を打つ。検証コマンド通過、写し 4 つ一致
- [x] inline の直し(S5 のあと、S5 の implementer の申し送り): director skill の「merge のあとの取り直し」の冒頭と手順 2 の括弧、受け入れの節の「ほかを先に merge すると rebase」に「merge queue のない repo では」を付けた。`cmd/ralph/main.go` の終了コード 7 のコメントから古い条件を外した。検証: check-skill-sync、check-sync、`go test ./internal/org/ -run 'Skill|Director'`、`go vet ./cmd/ralph/`、`go test ./cmd/ralph/` が通過。自己点検 5 項目 clean(行の折り返しを 2 か所直した)。持ち越し: tech-debt の org-merge-order の残りの行(Trigger が `mergeorder.go` の変更で当たる)と `mergeorder_queue_test.go` の 807 行は /sync-docs で扱う
- [x] Self-review run 1(1ce09b06、27723e15): CRITICAL・HIGH なし、MEDIUM 1、LOW 8、Merge: yes
- [x] fbdb30b1(self-review の直し、implementer): M1(1 段目で検査の列が `and <N> more` で切れても、見えている検査がどれも pending なら後回しにする。隠れた失敗は pending が終わると列に出て止まる、と skill に書いた。self-review の勧めの「止める」は、検査の多い PR が 1 段目で毎回止まるので採らなかった)、L1・L2(tech-debt の注)、L3(手順の番号に手順の名前を付けた)、L4(折り返し)、L5(merge-order の queue の読みが ssh の別名のヒントを付ける、テスト 4 ケース)、L7(doc)。L6(origin の読めない文の写し)と L8(テストの表の書き方)は /sync-docs で tech-debt に書く。inline で `mergeSSHAliasHint` の doc の使い手に merge-order を足した
- [x] Verify run 1(ec97fdd8、8bec7f39): pass。/sync-docs に LOW 10・INFO 3(NFR-1 の印の注、tech-debt の Trigger の当たった行、org skill の 2121 行の条件、読めない理由の列、AGENTS.md の行を組む場所、register の行数、L6・L8・M1 の根の行、FR-7 の注、AC の印)
- [x] d729352e(verify の勧め、implementer): `internal/org/director_skill_queue_test.go`。director の 2 段の手順が見分けに使う英語の断片 14 個と Go の値 5 つを、それを出す Go の文字列リテラルに結びつける(skill の 4 写し)。ずれはなかった。Go の文を言い換えた写しの repo で落ちることを確かめた
- [x] Test run 1(9726617e、b3b4b6ca): pass。`./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0)、新しいテストと変えたテストの `-v` run、main のテストを HEAD に当てた run(落ちたのは期待した差分だけ)、変異 34 個すべて検出。/sync-docs に LOW 1・INFO 4(test report の Test gaps 1〜5)
- [x] Sync-docs run 1(001fd717、f45cb832、cfed6c98、f2e01fa9、d562d672、06b74e8f、e6c81e40、28b8a789): verify の D1〜D13、self-review の L6・L8、test の Test gaps 1〜5 を扱った。仕様 `2026-11-19-org-merge-queue.md` の NFR-1 に 2 本目の印、`2026-10-07-org-multi-org-director.md` の FR-7 に追記、`/org` skill の regate の手順 3 に「merge queue のない repo では」(4 写し)、merge-order の help・README・`/org` skill の「読めない」の理由に `git remote get-url origin` の失敗と時間切れ、「queue を読むのは 1 回」を確かめの 1 回の実行ごとに、AGENTS.md に `mergeQueueLine`。tech-debt は、Trigger に当たった 182(a)・247・248・259・317・318・502・535(h) に 2026-11-20 の持ち越しの注、311 と 423 の行数を `fbdb30b1` のあとの値に直し、536 に本物の GitHub で確かめていない読みと手順を足し、L6・L8 と Test gaps 1〜3 を 1 行にした。help の文字列を直したので `go test ./internal/cli/ -run OrgMergeOrder` が通過(その文は pin されていない)。M1 の根(skill が stderr の文で断りの種類を見分ける)は `internal/org/director_skill_queue_test.go`(d729352e)が固定するので、行は足さない。計画の AC1〜AC7 を `[x]` にした(digest は変わらない)。report は `docs/reports/sync-docs-2026-11-20-org-merge-queue-order.md`
- [x] Review artifact created — `docs/reports/self-review-2026-11-20-org-merge-queue-order.md`
- [x] Verification artifact created — `docs/reports/verify-2026-11-20-org-merge-queue-order.md`
- [x] Test artifact created — `docs/reports/test-2026-11-20-org-merge-queue-order.md`
- [ ] PR created
