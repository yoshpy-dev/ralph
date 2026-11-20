# Verify report: org-merge-queue-order

- Date: 2026-11-20(Run 1。パイプラインの 1 回目の run で、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-order.md`。`./scripts/plan-visual.sh digest` の値 `357148b1a947` は、計画の `- Approved:` の行と一致する
- Spec: `docs/specs/2026-11-19-org-merge-queue.md`(#342 の 2 本のうち 2 本目: FR-12、FR-13、AC7、AC9 の skill の部分、AC11)。1 本目(PR #352)の FR-1〜FR-11 の挙動が、FR-12 が外す打ち直しの条件のほかは変わっていないことも見た
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `1d05c9ac`、HEAD `959246e5`、34 ファイル、+3306 −323)。branch 全体の spec 適合と文書のずれを見た。self-review の直し `fbdb30b1` と `a019591f` を含む。手元の `main` と `origin/main` は `a672b72a`(merge base のあとに #355 の仕様の文書 1 つ)で、`git merge-tree --write-tree --name-only main HEAD` は衝突なし
- Evidence: `docs/evidence/verify-2026-11-20-org-merge-queue-order.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。静的検査の出力、skill の 4 写しの `cmp`、行数、仕様のチェックボックスの数が入っている。ラッパー自身のログは `docs/evidence/verify-2026-11-20-093238.log`

## Spec compliance

テストは走らせていない(`/test` の担当)。コードを読み、各 AC にテストがあることまでを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(仕様 AC7 の merge-order): queue がある既定のブランチで base の古いブランチは `ready`(受け入れたブランチは `accepted`)、理由に queue。ない、読めないなら `rebase`(読めないときは注記)。base_ref が別のブランチなら queue があっても `rebase` | Met | `classifyGatePassed`(`internal/org/mergeorder.go:1352-1392`)は、`!named \|\| name != def` を先に見て `rebase` にし、そのあと `r.Base == defaultHead` なら ready、`queue.state == mergeQueuePresent` なら `mergeQueueMovedText` つきの ready、それ以外は今までの `rebase` に `mergeQueueUnreadText` を足す。先端の違い(`regate`)と PR の統合先(`wrong_base`)はその前の switch のまま。`classifyAccepted`(`:1242-1257`)は ready を accepted にし、queue で base が動いたときは理由の文を替える。読めないときの注記は `mergeOrderQueueNotes`(`internal/org/mergeorder_queue.go:115-121`)。テストは `TestClassifyMergeTarget_MergeQueueKeepsABranchWhoseBaseMovedReady`、`TestMergeOrder_MergeQueueKeepsMovedBranchesReadyAndChangesTheNote`、`TestMergeOrder_NoMergeQueueKeepsTodaysListAndNotes`、`TestMergeOrder_MergeQueueThatCannotBeReadIsUnknown`(`internal/org/mergeorder_queue_test.go`) |
| AC2(仕様 AC7 の merge): regate なしで step 7 と 13 を通って queue に入れる。先端が動けば断る。queue の GraphQL は 1 回。step 7 で読めないときの断りに理由が入る | Met | step 7 は `classify`(`internal/org/merge_queue_order.go:32-46`)。分類が `rebase` で、queue を仮に「ある」として分類し直すと ready か accepted になるときだけ `readQueueIfOnlyBaseMoved`(`:60-84`)が `lookupMergeQueue` で読み、結果を `m.queue`/`m.queueOf` に置く。step 11 は `queueReadFor(r.defaultBranch)`(`:91-96`)で同じブランチの読みだけを `CheckMergeEnforcement` に渡し、`enforce`(`internal/org/mergeenforce.go:221-231`)は渡された読みがあれば GraphQL を打たない。読めないときは `classify` が `mergeQueueUnreadText` のついた `rebase` の断りを返す。step 7 の gh の見つからなさは `runNotingGHNotFound` で拾い、`ghMissing` で断る。テストは `TestCheckMerge_QueueKeepsABranchWhoseBaseMovedReady`、`TestMerge_QueueEnqueuesABranchWhoseBaseMovedWithoutARegate`、`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase`、`TestCheckMerge_QueueReadAtStep7ThatFails`、`TestMergeQueueLookup_OneReadServesTheEnforcementCheck`。「1 回」は checks の 1 回の実行ごと(`--wait-checks` の待ちのあとの実行は読み直す、`TestMerge_QueueIsReadAgainInEachRunOfTheChecks`、下の D6) |
| AC3: queue のない repo で base の動いていないブランチの呼び出しの並びは今と同じ。base の動いたブランチは step 7 で `rebase`(読みが 1 つ増える) | Met(注つき) | `readQueueIfOnlyBaseMoved` は分類が `rebase` でなければ何も読まないので、base の動いていないブランチの並びは step 11 の読みだけで前と同じ。`TestCheckMerge_NoQueueKeepsTodaysCommands`(`internal/org/merge_queue_order_test.go:202`)が `mxAll` と `mrMerged` を固定する。`merge_test.go` の "not ready: rebase" の行は、AC3 が分けた base の動いた場合で、同じ断りを `TestCheckMerge_NoQueueKeepsTodaysCommands/base_moved` に移した(計画の Progress の S3 の注 (2) に記録あり) |
| AC4(仕様 AC11): ready の 3 本に `--wait-checks 0` で `enqueuePullRequest` が 3 回、どれも `MergeResult.Queued`。2 段目は入れ直さずに MERGED を記録。後回しの A は B のあとに入る | Met | `TestMerge_QueueTwoStagesForThreeReadyBranches`(`internal/org/merge_queue_order_test.go:601`)は 3 本の `Queued`(位置 1〜3)、`enqueued` が a、b、c、各実行で queue の読みが 1 回、2 段目の feat/b は読み 1 回、enqueue なし、`via=merge_queue` の `pr_merged` 1 行を確かめる。`TestMerge_QueueDeferredBranchGoesInOnceItsChecksPass`(`:651`)は A の pending の断り(読みなし、enqueue なし)、B の enqueue、A の打ち直しの enqueue で、順が b、a。CLI の終了コード 7 は既存の `TestOrgMerge_PrintsAQueuedResultAndExitsSeven`(`internal/cli/org_merge_queue_test.go:105`) |
| AC5: `merge queue:` の行は「ある」か「読めない」ときだけ、`--json` に `merge_queue`、origin が読めなければ「読めない」、対象のブランチがないときと cleanup の中では読まない、queue があれば注記を替える | Met | `MergeOrder`(`internal/org/mergeorder.go:402-408`)は `p.ReadMergeQueue` のときだけ、対象のブランチがあり ls-remote が通ったあと(`len(targets) == 0` は `:383` で先に返る)に読む。CLI の merge-order だけが立てる(`internal/cli/org_merge_order.go:246`)。`ralph org cleanup` は立てない(`internal/cli/org_cleanup.go:181`)。origin の get-url の失敗、時間切れ、repository のない URL は `mergeQueueUnknown`(`mergeorder_queue.go:83-94`)。行は `mergeQueueLine`(`org_merge_order.go:333-348`)が present と unknown のときだけ作る。JSON は `MergeQueue *MergeOrderQueue \`json:"merge_queue,omitempty"\``で、読んだときは absent も出る。注記は `mergeOrderListNotes`(`mergeorder.go:1438-1450`)が `res.MergeQueue.present()` で `mergeQueueSeveralReadyNote` に替える。テストは `TestMergeOrder_ReadsNoMergeQueueUnlessAskedAndOnlyWithATarget`、`TestMergeOrder_MergeQueueReadsOriginsRepository`、`TestOrgMergeOrder_MergeQueueLineFollowsTheGuaranteeLine`、`TestOrgMergeOrder_MergeQueueLineLeavesOutNoQueueAndEscapesItsValues` |
| AC6(仕様 FR-13、AC9 の skill の部分): director の 2 段の手順と org skill の文、4 写しの一致 | Met | `/director` skill の「merge と auto_merge」(`.claude/skills/director/SKILL.md:1693-1812`)は、`merge queue: origin/<既定のブランチ> has one` の行で手順を選び、行がないか `unknown` なら 1 本ずつの手順を残す。2 段の手順の 1〜6 が AC6 の各項(後回しの 2 つの断り、5 分の wait、60 分で人に伝える、2 段目の `--wait-checks 60m`、打ち直しのたびの merge-order、queue が merge したブランチの cleanup、終了コード 7 は打ち直してよい、FR-5b と `--requeue`)を書く。skill が引く stderr の文 6 つ(`which is running, so its leader may still push`、`now for another feature`、`reported no checks`、`are not pass or skipping:`、`and <N> more`、`as merged, not ready or accepted`)は Go の書式文字列(`internal/org/merge.go:623`、`:631`、`:766`、`:773`、`mergeorder.go:880`、`merge_queue_order.go:43`)にある。`/org` skill の `merge-order` と `merge` の行(394、396 行)と「merge してよい順」(1971-1988 行)は、base の古いブランチも `ready` になることと `merge queue:` の行を書く。4 写しは `cmp` で一致し、`check-skill-sync.sh` も通る。implementer が AC6 の外に足した 2 つ(2 段目で続けて 2 回 7 なら止める、push だけの wake の director も wait を打つ)は Progress の S5 に記録がある |
| AC7: step 13 は queue の道で base が動いても断らない。既定のブランチの名前が変わったとき、先端が動いたときは断る | Met | `checkOriginUnmoved`(`internal/org/merge.go:821-844`)の base の case に `&& !r.queue` を足しただけで、名前の違い、ブランチがない、先端の違いの case はそのまま。`TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase` が step 13 の 4 つの道(先端が動いた、step 13 の前だけ main が動いた、ブランチが消えた、既定のブランチが別)を持つ |

仕様の要件:

- FR-12: Met。step 7、11、13 の扱いは上の AC2、AC7 のとおり。step 13 は仕様の「branch の先端が gate の head であることの確かめだけを残す」より 1 つ多く、既定のブランチの名前も確かめる(計画の D3。安全側の差で INFO、下の D12)。ruleset の rule だけの queue で step 7 が base の動きを許さないのは計画の D4 で、仕様の印が書いている
- FR-13: Met(AC6)
- AC7、AC9、AC11: 上の AC1、AC2、AC4、AC6 のとおり。仕様のチェックボックスは 29 個すべて `[x]`(未チェック 0)で、状態の段落の「仕様の項目はすべて `[x]` になった」は正しい
- 1 本目の FR-1〜FR-11: 変わったのは打ち直しの条件の文(`queueNext`、`requeueNext` の doc、help、README、skill)と、queue の道の step 13 だけ。`queueNext` の文の変更は `merge_queue_run_test.go` の `mqrNext` が固定する。enqueue、待ち、記録、終了コード 7、`--requeue` のコードは触っていない
- NFR-1: 一部ずれた(下の D1)。queue を読めない repo の merge-order の出力が変わり、既存の CLI テスト 7 本の期待が変わった。計画の AC5 と D1 が選んだ形で、計画の Progress(S4)に記録があるが、仕様の NFR-1 の印には書かれていない

self-review の直し(`fbdb30b1`、`a019591f`)の突き合わせ: M1 は勧め(「`and <N> more` で切れたら止める」)と違う形で直した(名前の出た検査がどれも pending なら後回しにし、隠れた失敗は pending が終わると名前で出る、と書いた)。勧めと違う理由は Progress にある。L1(536 行の RESOLVED の文)、L2(252 行の注)、L3(手順の名前)、L4(折り返し)、L5(`readMergeOrderQueue` が `mergeSSHAliasHint` を足す、`mergeorder_queue.go:104-106`)、L7(`mergeOriginReadStep` の doc)はコードと文書で直っている。L6 と L8 は Progress のとおり `/sync-docs` に残っている(下の D10)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | Pass(rc=0) | shellcheck と `sh -n`(hook と verify のスクリプト)、guard の awk、`jq -e`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の verifier(選ばれた言語は golang だけ。`gofmt` は ok、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`1d05c9ac..959246e5` は clean) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-org-merge-queue-order.md` | `357148b1a947` | 計画の `- Approved:` と一致 |
| `cmp` で director と org の skill の 4 写し | 6 組すべて一致 | `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` |
| `go build` を `$TMPDIR` に出して `ralph org merge-order --help`、`ralph org merge --help` | どちらも rc=0 | 80 桁を超える行は flag の行だけ(前からある) |
| branch の commit の帰属の行と、diff の U+FFFD | どちらも 0 | |

## Documentation drift

README、AGENTS.md、2 つの help、2 つの skill の 4 写し、仕様 2 つは、FR-12 と FR-13 の挙動をおおむね正しく書いている。打ち直しの古い条件(`where the gate left it`、`has not moved`、`gate のときのまま`)は、条件つきで正しい `merge_queue_run.go:194` の `requeueNext` の doc と、`/org` skill の `merge` の行の step 13 の文(括弧で queue の道を断っている)のほかに残っていない。下は `/sync-docs` に渡すずれで、どれも AC の判定は変えない。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1: 仕様の NFR-1 の印 | No(LOW) | NFR-1 は「queue のない repo の挙動、出力、終了コードは変わらない(今のテストを変えずに通す)」で、1 本目の印だけがある。2 本目では、merge-order が queue を読めないとき(origin が GitHub の repository を名指さない、gh がない、落ちた、GHES に `mergeQueue` がない、など)に `merge queue: unknown (...)` の行と注記を毎回出す。fixture の origin がローカルのパスなので、既存の CLI テスト 7 本の期待出力にその行と注記が増え(Progress の S4)、`merge_test.go` の base の動いた行は新しいテストに移った(Progress の S3 (2))。merge-order は対象のブランチがあれば毎回 get-url(ssh の origin は `ssh -G` も)と GraphQL を 1 つずつ多く打つ。計画どおりの形なので、NFR-1 の下に 2 本目の印(何が変わり、どこに記録したか)を足す |
| D2: register の Trigger に当たったのに 2026-11-20 のこの branch の注がない行(確かなもの) | No(LOW) | 182 行の (a)「The next edit to the `/org` skill's command table」に、`merge-order` と `merge` の行の変更(4 写し)が当たる。259 行の「the next edit to ... `MergeOrder` in `internal/org/mergeorder.go`」に、`MergeOrder` の queue の読みと doc が当たる。この branch は (b) を広げた(merge-order の 1 回の実行が `git remote get-url origin` を `readMergeOrderQueue` の `ReadMergeOrigin` でもう 1 回読む)。(c) にも関わる(skill と README が、doctor の読まない classic の保護の queue を GraphQL の行が言うことがあると書いた)。317 行の「the next change to ... merge-order's classification」に、`classifyGatePassed` と `classifyAccepted` の queue の分岐が当たる。どの行にも、前の branch と同じ形の「当たったが払わなかった、持ち越す」の注を足す |
| D3: register の 248 行(と 247 行)の director の Trigger | No(LOW) | 248 行の Trigger は「the director part of the trigger moves to a director that reads merge-order in a loop」まで移っている。この branch の 2 段の手順の 2 は、後回しのブランチが残る間、5 分の wait が戻るたびに merge-order を打ち直す(最大 60 分)ので、その輪に当たる。輪の中で manifest の追記の途中を読んだ偽の `unreadable` が出ると、director はそのブランチを後回しから外して理由の欄に従う(disband 済みの org なら手順 6 で regate を打ちうる)。247 行(merge-order の gh の呼び出しの重さ)も同じ輪で、各実行に get-url、`ssh -G`、GraphQL が増えた。どちらにも注を足す(払うかは判断) |
| D4: register の Trigger に文言の上で当たる行(判断による) | No(LOW) | 318 行の「The next change to merge-order's classes」は、`ready` と `rebase` の定義(定数の doc と help の表)が変わったので当たると読める(新しい class はない)。502 行の「a gate, regate, or merge test of `internal/cli`」に、`internal/cli/org_merge_queue_test.go` の `TestOrgMerge_HelpDescribesTheMergeQueuePath` の変更が当たる(#340 と 1 本目は同じ種類の変更で注を書いた)。535 行の (h)「the next change to `mergeSSHAliasHint`」には、`a019591f` の doc の使い手の一覧と、新しい呼び出し元(`readMergeOrderQueue`)が当たる。(h) の中身(host の大文字小文字)は変わっていない |
| D5: `/org` skill の regate の手順の 3(2121 行、4 写し) | No(LOW) | 「それを merge すると、ほかの `ready` と `accepted` はまた `rebase` になるので、1 に戻る。」が条件なしで残る。同じ手順の 1(2087-2090 行)と、director skill の同じ文(1915-1916 行と 1969-1973 行)は「merge queue のない repo では」を付けた。queue のある repo でも、gate のあとに push されたブランチは regate の手順に来るので、そのときこの文は偽になる |
| D6: 「読めない」の理由の列 | No(LOW) | merge-order の help(`internal/cli/org_merge_order.go:205-207`「gh is missing or failed, its answer cannot be read, or origin's URL names no GitHub repository」)、README(「`gh` missing or failing, an answer it cannot read, an origin that names no GitHub repository」)、`/org` skill の `merge-order` の行(「`gh` がない、落ちた、答えが読めない、origin が GitHub の repository を名指さない」)は、`git remote get-url origin` が落ちたか時間切れになったとき(`readMergeOrderQueue` の `MergeOriginTimedOut`/`MergeOriginGetURLFailed`、`mergeorder_queue.go:86-88`)を挙げていない。閉じた列に見えるので 1 つ足す |
| D7: 「queue を読むのは 1 回だけ」の範囲 | No(LOW、任意) | README の `ralph org merge` の段落(「so the queue is read once」)と `/org` skill の `merge` の行(「queue を読むのは 1 回だけ」)は、checks の 1 回の実行ごとの話で、`--wait-checks` の待ちのあとの実行は読み直す(Go の doc は「one run of the checks reads the queue once at most」と書く)。「1 回の確かめで」と範囲を書くと誤読がない |
| D8: AGENTS.md の `mergeorder_queue.go` の説明 | No(LOW) | 「the texts of the `merge queue:` line and notes」とあるが、`merge queue:` の行を組むのは `internal/cli/org_merge_order.go` の `mergeQueueLine` で、`mergeorder_queue.go` が持つのは理由の欄の文(`mergeQueueMovedText`、`mergeQueueUnreadText`)と注記、`MergeOrderQueue` の型 |
| D9: register の 311 行と 423 行の行数 | No(LOW) | 「`internal/org/mergeorder_queue.go` (154 lines)」と、423 行の「`internal/org/mergeorder_queue_test.go` is 807 lines」は `7b3acbcd` の値で、self-review の直し `fbdb30b1`(L5)のあとの HEAD では 160 行と 862 行。`mergeorder.go` の行数は commit に結んであるが、この 2 つは結んでいない。Progress は 807 行の扱いを `/sync-docs` に渡している |
| D10: self-review の L6、L8 と、M1 の根の行 | No(LOW、予定どおり) | L6(「host と owner/repo がない」の節が 5 か所の写し、`mergeorder_queue.go:92` が 5 つ目)と L8(11 欄の位置埋めの表)は、Progress のとおり register の行がまだない。M1 の根(skill が stderr の文 6 つで断りの種類を見分け、Go の文との結びつきを固定するテストがない)も、self-review の「Tech debt identified」の行の候補で、今の register にない。下の Coverage gaps の検査を足すか、行にする |
| D11: `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 | INFO | FR-7 の「後から入る PR は、main に rebase してゲートを取り直してから出す」に、queue のある repo では取り直しを queue に任せる、という日付つきの注がない。この branch の注は FR-8 の 8a-3a の段落(144 行)にあり、merge-order、merge、director のことは書いている。FR-7 からそこを指す 1 文を足すかは任意 |
| D12: 仕様の FR-12 の step 13 の文 | INFO | 仕様は「branch の先端が gate の head であることの確かめだけを残す」で、実装は既定のブランチの名前とブランチの有無も確かめる(計画の D3)。安全側の差。FR-12 の印は D2 と D4 を挙げ、D3 は挙げていない |
| D13: 計画の AC のチェックボックス | INFO | AC1〜AC7 と Progress の「Verification artifact created」は `[ ]` のまま。いつもの遅れ |

## Observational checks

- HEAD の binary の `ralph org merge-order --help` は、class の表の `rebase` の行(「merge-order found no merge queue on it」)、merge queue の段落、`merge_queue` の JSON、終了コードの文を出す。`ralph org merge --help` は step 7 の queue の読み、step 13 の queue の道、終了コード 7 の条件なしの打ち直しを出す。どちらも rc=0
- 実際の GitHub の merge queue での確かめはしていない(仕様の NFR-2 と計画の Non-goals: release のあと下流で 1 回確かめる)

## Coverage gaps

- 決定的な検査がないもの: director skill の 2 段の手順は stderr の文 6 つと `merge queue: origin/<既定のブランチ> has one` で判断するが、それらが Go の文に今もあることを固定するテストがない。文言が変わると、director は人に伝える側か 1 本ずつの手順の側(regate の輪)に倒れるので害は小さいが、2 段の手順が黙って使われなくなる。足すと効く一番小さい検査: `internal/org/director_skill_sync_test.go` に、「merge と auto_merge」の節の 2 段の手順から backtick の英語の断片を抜き出し、各断片が `internal/org/` と `internal/cli/` のテスト以外の Go の文字列にあることを確かめるテストを 1 つ
- AC4 の 2 段の手順は org の層の fake で確かめ、CLI の終了コード 7 は既存のテストに任せている(計画どおり)。director の skill を実際に回したことはない
- GHES の GraphQL に `mergeQueue` があるかは見ていない。ない GHES では merge-order が毎回 `merge queue: unknown` を出す(D1、1 本目の NFR-1 の印と register の merge queue の行が step 11 について同じことを書いている)

## Verdict

- Verified: 計画の AC1〜AC7、仕様の FR-12、FR-13、AC7、AC9、AC11(コードとテストの存在を読んで)。1 本目の FR-1〜FR-11 の挙動が、打ち直しの条件のほかに変わっていないこと。静的検査がすべて通ること(gofmt、go vet、golangci-lint、staticcheck、同期の検査、secret scan)。skill の 4 写しの一致。計画の digest。help の表示
- Partially verified: 仕様の NFR-1(queue を読めない repo の merge-order の出力が変わる。計画どおりで、仕様の印に書かれていない。D1)。文書のずれ D1〜D10 は LOW で、`/sync-docs` に渡す
- Not verified: テストの実行(`/test` の担当)。本物の GitHub の merge queue での挙動(release のあと下流で)。director が 2 段の手順を実際に回すこと

判定: pass。
