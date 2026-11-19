# Verify report: approve-resolve-decision

- Date: 2026-11-20(JST。UTC では 2026-11-19。cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)。`- Approved:` の digest `37ad5d8a83d5` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff e6d5d0ed...HEAD`(HEAD `f8bcd876`、26 ファイル、+1929 −54)。self-review の 2 回目のあとの直し `ae5b028f`(N1〜N3)と `f8bcd876` を含む。テストは走らせていない(`/test` の仕事。別のセッションが同じ機械で `run-test.sh` を打っている)
- Evidence: `docs/evidence/verify-2026-11-19-approve-resolve-decision.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。静的検査の出力、試しの merge、binary の観察、2 つの help をこの順に入れた

## Spec compliance

binary の観察は、HEAD から build した `ralph` を、`$TMPDIR` の下に手で作った台帳(`scope_reserved` と leader の `spawned` を 3 つの org に 1 組ずつ)と偽の herdr(呼び出しを記録し、`agent wait` に `idle` を返すだけの sh)で打った。受信箱の件は `escalated` の行を手で書いた(`ralph org escalate` は人に知らせるので打たない)。approve と send は本物を打った。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: 承認を記録した件を、DECISION の `sent` がないまま resolve すると終了コード 1 で断り、件は開いたまま。エラーに `ApprovalDecisionCommand` と `--without-decision` が出る | Met | 観察: approve のあとの `inbox resolve e1` が rc=1。stdout は空、`inbox.jsonl` と `manifest.jsonl` は前後でバイト列が同じ、`inbox show` は `state: open`。エラーに `--state-dir` 付きの send の打ち方、続けて approve と同じ resolve、最後の行に `--without-decision` を言葉で述べる文が出る。どの行も `--without-decision` 付きのコマンドではない。コード: `ResolveInboxItem`(`internal/org/approve_resolve.go:91`)、断りの文は `approvalDecisionNotSentError`(`:268`) |
| AC-2: 承認のあとに DECISION を送れば今までどおり通る | Met | 観察: 偽の herdr を通した本物の `ralph org send`(approve が出した `--text`)のあと、resolve が rc=0、stdout `resolved e1`、stderr は空。`sent` は手で書いていない |
| AC-3: `--without-decision` で通り、stderr に確かめを求める 1 行が出る | Met | 観察: 何も送っていない org b の `e2` に `--without-decision` を付けると rc=0。stderr は `note: resolved e2 with --without-decision: no approval DECISION of plan ... recorded by this ralph was found for the leader of org b (...), so make sure the leader was told that the plan is approved` の 1 行。件の note は approve のもの |
| AC-4: 数えない `sent` 8 種、160 バイトと 180 バイトのパス、`submit_unconfirmed` | Met(ソースと binary の 4 種) | 観察: dry run、同じ `PLAN:` の直しの依頼(本物の send)、`text_sha256` のない古い形の `sent`(手で追記)はどれも断られ、`submit_unconfirmed=true` の `sent`(手で追記、hash は正しい)は通り、stderr に `ralph org read --org-id c --seat leader --state-dir ...` が出た。残りの 4 種(承認より前、leader 以外、別の org、別の TYPE)と別の計画は `checkApprovalDecision`(`:202`)で読んだ。最後の disband の後から数え、`wants` は承認を読んだあとにだけ増える。テスト `TestResolveInboxItem_SentsThatAreNotCounted`(8 種)と `..._LongPlanPathsAreToldApartByTheWholeText`(160、180)がソースにある |
| AC-9: 本物の `sent` が本文全体の sha256 を `text_sha256` に持つ | Met | 観察: 2 つの本物の send の `sent` はどちらも `text_sha256` を持ち、値は `--text` の sha256 と一致した(`7cec3f19...` は承認の DECISION、もう 1 つは直しの依頼の文)。偽の herdr の記録では `pane send-text` に paste の印が付いているので、hash は印を除いた文のもの。dry run の `sent` には欄がない。書き込みは `verbs.go:679` の 1 か所だけで、dry run(`:492`)は書かない |
| AC-10: 検査と書き込みを manifest の lock の中で行う。両方の順のテスト | Met(ソース)、lock を取ることは観察 | 観察: `manifest.lock` を python の `flock` で持ったまま、計画の QUESTION の resolve は 5.06 秒待って `acquire manifest lock ... timed out after 5s` で終わり、件は open のまま。計画の QUESTION でない件は 0.04 秒で resolve した。lock の順: `InboxStore` は manifest を持たず(`inbox.go:240`)、inbox の lock の中の 3 つの closure(`:296`、`:371`、`:529`)は inbox だけを読み書きするので、inbox → manifest の順で取る経路はない。両方の順のテスト `TestResolveInboxItem_ApproveWaitsForAResolveThatHoldsTheLock` と `..._ResolveWaitsForAnApproveThatHoldsTheLock` はソースにある |
| AC-5: `PLAN:` のない件、承認のない計画の件、前の命の承認しかない件は検査なしで resolve できる | Met | 観察: `PLAN:` のない QUESTION(`e4`)と承認していない計画の QUESTION(`e5`)はどちらも rc=0。前の命は `checkApprovalDecision` が最後の `disbanded` の次から読むことと、`TestResolveInboxItem_ItemsWithNoApprovalResolveAsBefore` の 5 件で確かめた |
| AC-6: leader の prompt が、起こされたら `inbox show` で件を読み、resolved で note が `approved <同じ PLAN>` なら進み、open か acked なら待ち続ける。prompt のテストが固める | Met(ソース) | `internal/org/prompts/leader.md` の手順 3 の終わり(464-484 行)、手順 4(490 行)、立て直しの受信箱の項(348 行)。`TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem` が 3 か所の文を順に固め、`approveNote`、`reseatProgressPlanSuffix`、`ApprovalResolveCommand` の形も確かめる |
| AC-7: director skill の 4 写し、approve と resolve の help が断りと `--without-decision` を書く。`check-skill-sync.sh` が通る | Met | 4 写しは `cmp` で同じ。`### 計画の承認` の 3 と 4(`.claude/skills/director/SKILL.md:1221-1245`)。build した binary の `ralph org approve --help` と `ralph org inbox resolve --help` に、断り、`--without-decision`、note を変えないことが出る(evidence の末尾)。`check-skill-sync: 14 skill(s) in lock-step` |
| AC-8: 送らずに閉じる場合と送ってから閉じる場合を `internal/org` と CLI のテストで確かめる | Met(ソース) | `internal/org/approve_resolve_test.go`(`RefusedUntilTheApprovalDecisionIsSent`、`WithoutDecision...`)と `internal/cli/org_inbox_resolve_test.go`(`ApprovedItemNeedsTheDecisionSent`、`WithoutDecisionSaysNoDecisionWasFound`)。通るかは `/test` |

### issue #345 の受け入れ条件

| 受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| approve の後に DECISION を送らず resolve を打つと、明示の flag が無い限り断られる | Met | AC-1、AC-3 |
| 承認を待つ leader は、DECISION が届かなくても止まり続けない(台帳で進むか、director に知らせが届く) | Met(設計とソース)、実地は未確認 | 件が開いたままになり ack と resolve の期限にかかる(AC-1)。起こされた leader は resolve の note で承認を読む(AC-6)。approve より前に resolve の期限が送り済みの件は期限にかからない穴が残り、register の 260 行目が 2026-11-19 の追記で持つ。本物の leader が起こされて note を読む流れは打っていない |
| 送らずに閉じる場合と、送ってから閉じる場合の両方を試験で確かめる | Met(ソース) | AC-8 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | **FAIL**(rc=1) | golangci-lint と staticcheck が同じ 1 件を出した: `internal/org/approve_resolve.go:269:9: ST1005: error strings should not end with punctuation or newlines`。`ae5b028f`(N3 の直し)が断りの文の最後を `... reads the note as the approval.` にしたことで入った。`47257bf2` と `42524d6e` の文はピリオドで終わっていない。repo のほかの場所の指摘は 0 件 |
| 同じ run の残りの項目 | pass | shellcheck と `sh -n`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、gofmt、`go vet ./...`、branch の secret scan(`e6d5d0ed..f8bcd876` clean) |
| `git diff --check e6d5d0ed...HEAD` | pass | 空白の誤りなし |

直し方: 断りの文の最後のピリオドを取る(または文の最後を句読点でない語で終える)。同じ文を固めているテストが 2 つある(`internal/org/approve_resolve_test.go:118`、`internal/cli/org_inbox_resolve_test.go:74`)ので、3 か所を一緒に変える。最後の行が `--without-decision` を述べる文でコマンドでないこと(テストの確かめ)は変わらない。CI の `verify.yml` は golangci-lint も staticcheck も入れないので、CI ではおそらく通る。未確認です。

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org approve --help` | Yes | 「resolve は DECISION を送るまで断る(または `--without-decision`)」と、resolve の期限が送り済みの件は `ralph org inbox` を読むしかないことを言う |
| `ralph org inbox resolve --help` と `--without-decision` の flag の説明 | Yes | 断り、数えない古い `sent`、`submit_unconfirmed` の stderr、note を変えないこと。binary の出力で確かめた |
| director skill の 4 写し | Yes | `cmp` で同じ。断りの文の並び(send、resolve、最後に言葉で flag)と code が一致する |
| org skill の 4 写し | 一部 | `cmp` で同じ。`approve` の行(391 行)、「受信箱」の項(566-572 行)、「機能の計画の承認」(1195-1209 行)は断りと flag を書く。ただし verb 表の `inbox` の行(387 行)は今も「`ack` / `resolve` / `notify` は知らない ID と resolved の件で、終了コード 1 になる」と、終了コード 1 の場合を閉じた形で挙げ、新しい断りがない(self-review が `/sync-docs` に渡した項目。LOW) |
| leader の prompt | Yes | AC-6 の行 |
| README 133 行と 290 行 | Yes | どちらも resolve の断りと `--without-decision` を書く。290 行の「stays under the inbox deadlines」は前からある言い方で、期限の送り済みの穴は register の 260 行目が持つ |
| `docs/specs/2026-10-07-org-multi-org-director.md` 156 行 | 一部 | 日付つきの追記が D7 の形を保つことと resolve の断りを書く。`--without-decision` の使いどころを「DECISION を送らずにわざと閉じるとき」とだけ言い、前の版の ralph で送ったときを使いどころとして挙げない。ほかの面は 2 つの場合を挙げる(self-review が `/sync-docs` に渡した項目。LOW) |
| `docs/tech-debt/README.md` 260 行 | Yes | #345 が trigger の前半に当たったこと、期限は変えていないので穴は残ることを書く。計画のリンクは `docs/plans/active/` で、`/pr` の `archive-plan.sh` が書き換える |
| `docs/quality/quality-gates.md` の「Feature plan approval」の行(root と template) | 一部(INFO) | 「DECISION を送らなかった件は inbox の期限にかかる」は、resolve の断りで前より正しくなった。断りと `--without-decision` は書いていない。足すかは `/sync-docs` の判断 |
| 計画の AC の印と Progress | INFO | AC の `[ ]` と「Verification artifact created」は orchestrator か `/sync-docs` が付ける |

### register の trigger の洗い出し

register の Trigger の欄を python で列に分けて読み、この差分が触れるものに当たる行を挙げる。どれにも fix/approve-resolve-decision の carry-over の文はない。払うかは計画の範囲の外なので、`/sync-docs` に「carried over by fix/approve-resolve-decision」の 1 文を頼む(LOW)。

- 190 行 (a)「the next change to ... the leader prompt」: `leader.md` の手順 3、4 と立て直しの受信箱の項を変えた
- 240 行「the next change to ... `internal/org/lockfile.go`」と (h) の `withFileLock` 側: `withFileLock` の doc を変えた(コードは同じ)
- 264 行「the inbox section (`受信箱の処理`) of the `/director` skill」: `### 計画の承認` はこの `##` の下にある
- 367 行「the next change to `Send` or to the leader template tests」: `Send` が `TextSHA256` を書くようになり、`prompts_leader_plan_test.go` は 1302 行から 1476 行(+174)。`verbs.go` は 2639 行から 2655 行
- 379 行「the feature-org steps of `internal/org/prompts/leader.md`」: 手順 3 と 4
- 467 行「the next change to the `/director` skill」
- 382 行「the send-and-check steps of the `/director` skill」: 手順 4 の send と確かめの文の後ろに resolve の断りを足した。確かめの文そのものは変えていないので、当たるかは判断が要る
- 209 行 (a)「the next change to `withFileLock` or to the lock of `InboxStore`」: doc の変更だけ。inbox の lock を manifest の lock の中で取る入れ子が新しくできたが、`InboxStore` の lock は変えていない。判断が要る
- 519 行(trigger には当たらない): `ApprovalDecisionCommand` が `--config` を付けない穴の、新しい呼び手として resolve の断りの文(`approvalDecisionNotSentError`)と `printResolveNotes` の `ralph org read` が増えた。行の説明の呼び手の一覧に足すとよい
- 525 行: `sent` の Details は変えていないので当たらない。self-review の L4 は `sentSubmitUnconfirmed` をこの行の読み手に足すよう勧めている(follow-up)

## Observational checks

- binary の観察の全出力は evidence の「binary probe」の節。台帳と偽の herdr は `$TMPDIR/verify-ard.TGaAWI/` にあり、repo には何も書いていない
- approve の出力: 件ごとに send と resolve の打ち方を出し、`--state-dir` を繰り返す。resolve 済みの件の approve は `the item is already resolved` で断られた
- send は claude の座席に paste の印で包んで打ち、`agent wait --until working --until blocked` で submit を確かめた(偽の herdr の記録)

### 今の origin/main との試しの merge

`git fetch origin main` のあと `git merge-tree --write-tree --name-only origin/main HEAD`(origin/main は `32274500`、merge-base は `e6d5d0ed`)は **conflict** を返した(rc=1)。

- conflict は org skill の 4 写し(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 つ)の 1 か所ずつだけ。README と `docs/tech-debt/README.md` は自動で merge できた
- 原因は隣り合う表の行: main の `6a05139e`(#341 の sync-docs)が verb 表の `gate` の行に `GITHUB_BASE_REF` の 2 句を足し、この branch は直前の `approve` の行を変えた。base と比べると、main の `approve` の行は base と同じで、この branch の `gate` の行も base と同じ。直し方は機械的で、この branch の `approve` の行と main の `gate` の行を取る。4 写しの conflict の結果は同じ中身なので、`.claude/skills/org/SKILL.md` と `templates/base/.claude/skills/org/SKILL.md` を直して `scripts/sync-skills.sh` で `.agents` の 2 つを作ればよい

## Coverage gaps

- テストは走らせていない。AC-4 の 8 種、AC-5 の前の命、AC-6 と AC-7 の文を固めるテスト、AC-8、AC-10 の両方の順は、ソースで中身を読んだだけで、通るかは `/test` が確かめる
- 本物の herdr と本物の leader での流れ(承認を待つ leader が stall の ALERT か Monitor の張り直しで起こされ、`inbox show` で note を読んで進む)は打っていない
- 古い ralph が書いた `sent` は、Details に DECISION の全文を入れた行を手で書いて模した。古い binary は build していない
- 決まった検査が足りない点: 断りの文のような長いエラー文に ST1005 が当たることは、手元に golangci-lint か staticcheck がないと分からず、CI は入れていない。CI の verify に staticcheck を足すのが最小の検査になる(この計画の範囲の外)

## Verdict

- Verdict: **fail**。spec の適合と文書は通るが、`./scripts/run-static-verify.sh` が ST1005 の 1 件(`internal/org/approve_resolve.go:269`)で rc=1。断りの文の最後のピリオドを取り、同じ文を固める 2 つのテストを一緒に直して、`/self-review` から回し直す
- Verified: AC-1、AC-2、AC-3、AC-9、AC-5(前の命を除く)を binary で観察した。AC-4 の 4 種と `submit_unconfirmed`、AC-10 の「計画の QUESTION の resolve は manifest の lock を取り、ほかの件は取らない」も観察した。AC-7 の help と 4 写し、lock の順(inbox → manifest の経路がないこと)、承認の digest も確かめた。`run-static-verify.sh` の ST1005 以外の全項目は通った
- Partially verified: AC-4 の残りの種類、AC-6、AC-8、AC-10 の両方の順(ソースとテストのソースだけ)。issue #345 の 2 つ目の受け入れ条件(設計とソース。実地は未確認)
- Not verified: テストの実行(`/test`)。本物の herdr と leader での起こされ方。CI で golangci-lint と staticcheck が走るか
- `/sync-docs` に渡すもの: org skill の verb 表の `inbox` の行、仕様 156 行の `--without-decision` の使いどころ、register の trigger の行(190、240、264、367、379、467。382 と 209 は判断)の carry-over、519 行の呼び手、quality-gates の行(INFO)、計画の AC の印
- merge の前に: origin/main と org skill の 4 写しで conflict する(隣り合う行で、直し方は機械的)
