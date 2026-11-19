# Verify report: approve-resolve-decision

- Date: 2026-11-20(JST。UTC では 2026-11-19。cycle 1 の 2 回目の verify、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)。`- Approved:` の digest `37ad5d8a83d5` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 32274500...HEAD`(HEAD `a8e0e72f`、27 ファイル、+2031 −54)。merge-base は origin/main の `32274500` そのもの。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-19-approve-resolve-decision.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。1 回目の出力の後ろに「Run 2」の節を足した(前の中身はバイト列のまま残してある)。`run-verify.sh` 自身のログは `docs/evidence/verify-2026-11-19-160317.log`

## 1 回目の失敗とその直し

1 回目(HEAD `f8bcd876`、報告の commit `245f9336`)は fail だった。`./scripts/run-static-verify.sh` の golangci-lint と staticcheck が、`internal/org/approve_resolve.go:269` の断りの文に ST1005(エラー文が句読点で終わる)を出した。ほかの項目と spec の適合は通っていた。

- `f3c8e4b2` が断りの文の最後のピリオドを取り、同じ文を固める 2 つのテスト(`internal/org/approve_resolve_test.go:118`、`internal/cli/org_inbox_resolve_test.go:74`)も同じように直した。3 ファイル、+3 −3。差分はこの 1 文字だけ
- `a8e0e72f` は origin/main `32274500`(#341、#348、#349)を merge した。conflict は org skill の 4 写しの verb 表の 1 か所で、この branch の `approve` の行と main の `gate` の行を取り、`.agents` の 2 つは作り直した(下の「merge の確かめ」)

## Spec compliance

1 回目の binary の観察(`f8bcd876` から build した `ralph`、手で作った台帳、偽の herdr)は今も有効と判断した。`f8bcd876` から HEAD までの間に、この branch のファイルで変わったのは `f3c8e4b2` の 3 行と、merge の 6 ファイル(org skill の 4 写しの `gate` の行、README と `docs/tech-debt/README.md` の main 側の変更)だけで、Go のコードは断りの文のピリオド以外は同じ。main 側の Go の変更(`internal/org/gate*.go`、`internal/cli/org_gate*.go`、`internal/config/config.go`)はこの branch のファイルと重ならない。

HEAD から build し直した binary で、1 回目の台帳の写しを使って断りを 1 度だけ打ち直した(`$TMPDIR/verify2-ard/`)。開いている計画の QUESTION `e6` を approve し、DECISION を送らずに resolve すると rc=1、stdout は空、件は `state: open` のまま。stderr の並び(send の打ち方、resolve の打ち方、最後に `--without-decision` を言葉で述べる文)は 1 回目と同じで、最後の行は `... reads the note as the approval` と改行で終わる。1 回目は `approval.` と改行だった。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: 承認を記録した件を、DECISION の `sent` がないまま resolve すると終了コード 1 で断り、件は開いたまま。エラーに `ApprovalDecisionCommand` と `--without-decision` が出る | Met | 1 回目の観察(rc=1、`inbox.jsonl` と `manifest.jsonl` は前後で同じ、`state: open`)と、上の HEAD での打ち直し。どの行も `--without-decision` 付きのコマンドではない。コード: `ResolveInboxItem`(`internal/org/approve_resolve.go:91`)、断りの文は `approvalDecisionNotSentError`(`:268`) |
| AC-2: 承認のあとに DECISION を送れば今までどおり通る | Met | 1 回目の観察: 偽の herdr を通した本物の `ralph org send` のあと、resolve が rc=0、stdout `resolved e1` |
| AC-3: `--without-decision` で通り、stderr に確かめを求める 1 行が出る | Met | 1 回目の観察: org b の `e2` が rc=0、stderr は `note: resolved e2 with --without-decision: ...` の 1 行 |
| AC-4: 数えない `sent` 8 種、160 バイトと 180 バイトのパス、`submit_unconfirmed` | Met(ソースと binary の 4 種) | 1 回目の観察: dry run、同じ `PLAN:` の直しの依頼、`text_sha256` のない古い形の `sent` は断られ、`submit_unconfirmed=true` の `sent` は通って `ralph org read` の行が出た。残りの 4 種は `checkApprovalDecision`(`:202`)で読んだ。テスト `TestResolveInboxItem_SentsThatAreNotCounted` と `..._LongPlanPathsAreToldApartByTheWholeText` がソースにある |
| AC-9: 本物の `sent` が本文全体の sha256 を `text_sha256` に持つ | Met | 1 回目の観察: 2 つの本物の send の `sent` の値は `--text` の sha256 と一致し、dry run の `sent` には欄がない。書き込みは `verbs.go:680` の 1 か所だけ(1 回目の報告の `:679` は 1 行ずれていた。`f8bcd876` でも 680 行) |
| AC-10: 検査と書き込みを manifest の lock の中で行う。両方の順のテスト | Met(ソース)、lock を取ることは観察 | 1 回目の観察: `manifest.lock` を持ったまま、計画の QUESTION の resolve は 5.06 秒待って timeout、件は open のまま。計画の QUESTION でない件は 0.04 秒で resolve した。inbox → manifest の順で取る経路はない。両方の順のテストはソースにある |
| AC-5: `PLAN:` のない件、承認のない計画の件、前の命の承認しかない件は検査なしで resolve できる | Met | 1 回目の観察(`e4`、`e5` が rc=0)と、`checkApprovalDecision` が最後の `disbanded` の次から読むこと |
| AC-6: leader の prompt が、起こされたら `inbox show` で件を読み、resolved で note が `approved <同じ PLAN>` なら進み、open か acked なら待ち続ける | Met(ソース) | `internal/org/prompts/leader.md` の手順 3 の終わり(464-484 行)、手順 4(490 行)、立て直しの受信箱の項(348 行)。`TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem` が固める |
| AC-7: director skill の 4 写し、approve と resolve の help が断りと `--without-decision` を書く。`check-skill-sync.sh` が通る | Met | director skill と org skill の 4 写しは HEAD でも `cmp` で同じ。help の文は 1 回目に binary で確かめ、その後変わっていない。`check-skill-sync: 14 skill(s) in lock-step` |
| AC-8: 送らずに閉じる場合と送ってから閉じる場合を `internal/org` と CLI のテストで確かめる | Met(ソース) | `internal/org/approve_resolve_test.go` と `internal/cli/org_inbox_resolve_test.go`。2 つの断りのテストは `f3c8e4b2` でピリオドのない文に合わせてある。通るかは `/test` |

### issue #345 の受け入れ条件

| 受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| approve の後に DECISION を送らず resolve を打つと、明示の flag が無い限り断られる | Met | AC-1、AC-3 |
| 承認を待つ leader は、DECISION が届かなくても止まり続けない(台帳で進むか、director に知らせが届く) | Met(設計とソース)、実地は未確認 | 件が開いたままになり期限にかかる(AC-1)。起こされた leader は resolve の note で承認を読む(AC-6)。approve より前に resolve の期限が送り済みの件の穴は、register の 261 行目(merge で 260 行目から 1 行ずれた)が持つ |
| 送らずに閉じる場合と、送ってから閉じる場合の両方を試験で確かめる | Met(ソース) | AC-8 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | pass(rc=0) | `Language scope: changed (changed_languages)`。gofmt ok、`go vet ./...`、golangci-lint `0 issues.`、staticcheck は出力なし。1 回目の ST1005 は消えた。shellcheck と `sh -n`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照も通った。branch の secret scan は `32274500..a8e0e72f` が clean。golangci-lint の lock の競合は出なかった |
| `git diff --check 32274500...HEAD` | pass | 空白の誤りなし |

merge で入った #341 は `run-verify.sh` と `detect-changed-languages.sh` の base の読み方を変えた。この run は base を origin/main(`32274500`)として読み、範囲は上のとおり。

## merge の確かめ

- origin/main は今も `32274500` で、HEAD の祖先。今の origin/main との conflict はない
- 3-way の行の多重集合(base `e6d5d0ed`、ours `f3c8e4b2`、theirs `32274500`、merged `a8e0e72f`)を、org skill の 4 写し、README、`docs/tech-debt/README.md`、2 つの `quality-gates.md` で数えた。どれも「ours + theirs − base = merged」で、落ちた行も余分な行もない
- org skill の verb 表: HEAD の `approve` の行(391 行)は `f3c8e4b2` のものと同じで、`gate` の行(392 行)は `32274500` のものと同じ。4 写しは `cmp` で同じ
- main 側が足した行を `#345`、`without-decision`、`approve`、`inbox resolve`、`DECISION` で grep した。この branch の振る舞いを「まだない」と書く文はない。main の新しい仕様 `docs/specs/2026-11-19-org-merge-queue.md` は approve にも resolve にも触れない

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org approve --help`、`ralph org inbox resolve --help` | Yes | 1 回目に binary で確かめた。その後 help の文は変わっていない。help のテストの「reads that note as the approval.」は help の文で、エラー文ではないので ST1005 にかからない |
| director skill の 4 写し | Yes | 断りの並び(send、resolve、最後に言葉で flag)は code と一致する。断りの最後の文をそのまま引く文書はない(`git grep` で Go の 3 か所だけ) |
| org skill の 4 写し | 一部(L1) | `approve` の行、「受信箱」の項(566-572 行)、「機能の計画の承認」(1195-1209 行)は断りと flag を書く。verb 表の `inbox` の行(387 行)は今も「`ack` / `resolve` / `notify` は知らない ID と resolved の件で、終了コード 1 になる」と、終了コード 1 の場合を閉じた形で挙げ、新しい断りがない |
| leader の prompt | Yes | AC-6 の行 |
| README 133 行と 290 行 | Yes | merge は README の別の 2 行(main の #341 の変更)だけを変えた |
| `docs/specs/2026-10-07-org-multi-org-director.md` 156 行 | 一部(L2) | `--without-decision` の使いどころを「DECISION を送らずにわざと閉じるとき」とだけ言い、前の版の ralph で送ったときを挙げない。ほかの面は 2 つの場合を挙げる |
| `docs/tech-debt/README.md` 261 行 | Yes | #345 の行。計画のリンクは `docs/plans/active/` で、`/pr` の `archive-plan.sh` が書き換える |
| `docs/quality/quality-gates.md` の「Feature plan approval」の行(root と template) | INFO | merge の前と同じ。断りと `--without-decision` を足すかは `/sync-docs` の判断 |
| 計画の AC の印と Progress | INFO | AC の `[ ]` と「Verification artifact created」は orchestrator か `/sync-docs` が付ける。Progress には `f3c8e4b2` と merge `a8e0e72f` の記録がまだない |

### register の trigger の洗い出し

行番号は HEAD のもの。main の #341 が行を 1 つ足したので、1 回目の報告の 241 行目以降の番号は 1 つずつ増えた。この branch の carry-over の文を持つ行はまだない(`approve-resolve-decision` を含む行は 261 行目だけ)。

L3(1 回目からのもの): 190 行 (a) の leader の prompt、241 行(1 回目の 240 行)の `internal/org/lockfile.go`(main の #341 がこの行に別の注記を足したが、この branch の carry-over はない)、265 行の `受信箱の処理` の節、368 行の `Send` と leader の template のテスト、380 行の `leader.md` の feature-org の手順、468 行の `/director` skill。383 行の `/director` skill の send と確かめの手順と、209 行 (a) の `withFileLock` は判断が要る。

L4: 520 行(1 回目の 519 行)。`ApprovalDecisionCommand` が `--config` を付けない穴の行で、呼び手の一覧に、resolve の断りの文(`approvalDecisionNotSentError`)と `printResolveNotes` の `ralph org read` が入っていない。

L5(この run で見つけたもの): 1 回目の洗い出しが拾わなかった行が 2 つある。どちらも merge とも `f3c8e4b2` とも関係なく、1 回目の HEAD でも当たっていた。

- 182 行 (a)「The next edit to the `/org` skill's command table」: この branch は verb 表の `approve` の行を書き足した。表の行全体で 1,384 字から 1,681 字(`32274500` と HEAD で数えた)になり、(a) が名指す `stop`(1,208 字)と `disband`(1,219 字)より長くなった。この 2 つの行は変わっていない。前の PR と同じ形の carry-over の 1 文が要る
- 228 行 (b)「the next change to `Send`」: `Send` は `sent` に `TextSHA256` を書くようになり、doc に段落が 1 つ増えた(`verbs.go:400-404`)。(b) の「`The watch gate is` のあとの短い行」は直していない

INFO(判断が要るもの): 202 行の (g)「the feature-org section of `leader.md` is edited」は、この branch の手順 3、4 と立て直しの受信箱の項で当たる。ただし (g) はもう閉じていて、前の PR は「当たったが払うものがない」と注記した。365 行の「the roster (`internal/org/manifest.go`)」は、この branch が `ManifestEvent` に欄を足しただけで roster の読み方を変えていないので、当たらないと見た。

## Observational checks

- HEAD の binary での断りの打ち直しの全出力は evidence の「Run 2」の節。台帳の写しと binary は `$TMPDIR/verify2-ard/` にあり、repo には何も書いていない
- 1 回目の観察(approve の出力、send の paste の印、lock の待ち)の全出力は evidence の前半にある

## Coverage gaps

- テストは走らせていない。`f3c8e4b2` が直した 2 つのテストの文が実際の断りと合うかも、AC-4 の 8 種、AC-5 の前の命、AC-6 と AC-7 の文を固めるテスト、AC-8、AC-10 の両方の順も、通るかは `/test` が確かめる
- 1 回目の観察は `f8bcd876` の binary で打った。HEAD の binary で打ち直したのは AC-1 の断りだけで、AC-2 から AC-10 の観察はコードの差分(ピリオド 1 文字)から有効と判断した
- 本物の herdr と本物の leader での流れ(承認を待つ leader が起こされ、`inbox show` で note を読んで進む)は打っていない
- 決まった検査が足りない点: ST1005 のような lint の指摘は、手元に golangci-lint か staticcheck がないと分からず、CI の `verify.yml` は入れていない。CI の verify に staticcheck を足すのが最小の検査になる(この計画の範囲の外)
- 1 回目の verify の insight event は `"low":0` で記録された。1 回目の報告は L1 から L4 にあたる項目を `/sync-docs` に渡していたので、この数は合っていない。jsonl は追記だけなので、その行はそのまま残した

## Verdict

- Verdict: **pass**。`./scripts/run-static-verify.sh` は rc=0 で、1 回目の ST1005 は `f3c8e4b2` で消えた。spec の適合は 1 回目と同じく AC-1 から AC-10 がすべて Met で、merge は振る舞いを変えていない
- Verified: 断りの文がピリオド 1 文字だけを失ったこと(ソースの差分と HEAD の binary での打ち直し)。merge の解き方(この branch の `approve` の行と main の `gate` の行、4 写しが同じ、8 ファイルの 3-way の行の数)。静的検査の全項目。承認の digest
- Partially verified: AC-4 の残りの種類、AC-6、AC-8、AC-10 の両方の順(ソースとテストのソースだけ)。issue #345 の 2 つ目の受け入れ条件(設計とソース。実地は未確認)
- Not verified: テストの実行(`/test`)。本物の herdr と leader での起こされ方。CI で golangci-lint と staticcheck が走るか
- `/sync-docs` に渡すもの(LOW 5 件): L1 org skill の verb 表の `inbox` の行、L2 仕様 156 行の `--without-decision` の使いどころ、L3 register の trigger の行(190、241、265、368、380、468。383 と 209 は判断)の carry-over、L4 520 行の呼び手、L5 この run で見つけた 182 行 (a) と 228 行 (b) の carry-over。INFO として 202 行と 365 行の判断、quality-gates の行、計画の AC の印と Progress
- Insight event: `--phase verify --verdict pass --cycle auto --low 5` で 1 行を足した(1 回目の fail の行のあとに、同じ cycle 1 の pass の行が続く)
