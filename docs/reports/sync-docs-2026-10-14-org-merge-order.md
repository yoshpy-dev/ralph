# sync-docs report: org-merge-order

- Date: 2026-10-14(JST)
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md`(承認済み。`./scripts/plan-visual.sh digest` は `c0b1e9230999` で、`- Approved:` の行と同じ。この run が変えたのは checkbox の印だけ)
- Pipeline cycle: 1(`--cycle auto`。`.harness/state/standard-pipeline/cycle-count.json` は 1)。差分の base は `ac34ccf3`、branch は `feat/org-merge-order`。run 開始時の HEAD は `e20f2616`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-14-org-merge-order.md`(Merge: yes、CRITICAL・HIGH・MEDIUM 0、LOW 8)、`docs/reports/verify-2026-10-14-org-merge-order.md`(pass。文書のずれ D1 と D2)、`docs/reports/test-2026-10-14-org-merge-order.md`(pass、失敗 0)
- 項目の番号: D1・D2 は verify の「Documentation drift」の表の番号
- 内容の commit: この report、直した文書、計画の checkbox を含む 1 つの commit(`docs: sync docs for org-merge-order`)。insight event の `docs/insights/events/2026-10-13-org-merge-order.jsonl` は、指示どおりコミットに入れていない(未追跡のまま)

## Summary

verify が挙げた D1(README の分類の順)と D2(tech-debt の Trigger)を文書だけで直した。動詞の表(`README.md:124`)、AGENTS.md の repo map、仕様 FR-8 の注、`/org` skill の 4 面、`docs/architecture/repo-map.md`、`.claude/rules/ralph/` は、コードと合っていたので変えていない。

そのほかに、見張りなしで動く動詞を挙げる文書を掃いて、1 件を足した。`docs/quality/quality-gates.md` の「Cross-org watch gate」の行(と template の写し)が `merge-order` を挙げていなかった。同じ文が 7a の `approve` も挙げていなかったので、一緒に足した(skill の一覧は両方を挙げ、どちらも見張りの関門を呼ばない)。

計画は、依頼どおり checkbox の印(AC1 から AC11、Review・Verification・Test の artifact)だけを変えた。Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形、skill、`scripts/`、AGENTS.md、仕様、ほかの報告は触っていない。

## Changes made

| File | D | Change |
|------|---|--------|
| `README.md:262`(`ralph org merge-order` の段落の 3 文目) | D1 | 「Each branch gets the first class that applies, and the list is in this order: ...」は並びの順だけを挙げ、「最初に当てはまる」をどの順で調べるかを言わなかった。並びの順で読むと判定と食い違う(push していない gate なしの枝は `no_gate`、並びでは `no_remote` が先)。「checked in this order: `merged` or `closed`, `unreadable`, `no_gate`, `failed`, `no_remote`, `regate`, `wrong_base`, `rebase`, and `ready` for a branch none of those fits」と判定の順を先に言い、次の文で「The list is in a different order: `ready`, `rebase`, `wrong_base`, `regate`, `unreadable`, `failed`, `no_remote`, `no_gate`, and with `--all` `merged` and `closed`, which it leaves out otherwise」と並びの順を分けた。分類ごとの括弧の説明(`unreadable`、`no_remote`、`regate`、`wrong_base`、`rebase`、`merged`・`closed` が `gh` でだけ分かること)は、最初に名前が出る判定の順の文に移した。判定の順は `classifyMergeTarget` と `classifyGatePassed`(`internal/org/mergeorder.go`)、並びの順は `mergeClassOrder` から取り、skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`)と仕様 FR-8 の 7c-1 の注と同じ順になる。段落のほかの文は変えていない |
| `docs/tech-debt/README.md:231`(「Small findings of org-gate cycle 2」の行の Trigger の列) | D2 | Trigger の「and stage 7c, which reuses these files.」のあとに、この trigger が 7c-1(`feat/org-merge-order`)で発火し、払われずに持ち越されたことを足した。`internal/org/mergeorder.go` は `gate.go` から `gateRecord`・`gateRecordFromDetails`・`gitObjectPattern` を読むだけで、`gate.go`・`gate_run.go`・`lockfile.go` は 1 つも変えず(スクリプトを走らせず、ロックも取らない)、(a) から (i) のどれにも触れていない。次にそれらのファイルを変えるときは今も trigger になる。行は消していない。5 列、変わったのは Trigger の列だけ(HEAD と比べて変わった行は 231 の 1 行だけで、ほかの列は 1 字も変わらない) |
| `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79`(「Cross-org watch gate」の行の末尾の文) | 掃引 | 「`stop`, `disband`, `escalate`, `gate`, and the inbox verbs work with no watch」を「`stop`, `disband`, `escalate`, `approve`, `gate`, `merge-order`, and the inbox verbs work with no watch」にした(両面とも同じ文)。`merge-order` は見張りの関門も承認の検査もかけない(AC9。`checkWatchGate` を呼ぶのは start・spawn・send だけ)。`approve` は 7a から抜けていた(`org_approve.go` が関門を呼ばないことは verify が確かめている) |
| `docs/plans/active/2026-10-14-org-merge-order.md`(`## Acceptance criteria` と `## Progress checklist`) | - | AC1 から AC11 の 11 個と、`Review artifact created`・`Verification artifact created`・`Test artifact created` の 3 個を `- [x]` にした(`git diff --numstat` が 14 / 14)。verify の「Spec compliance」の表は AC1 から AC9 と AC11 を「満たす」とし、AC10 は README の 1 文(D1)だけがずれていた。D1 をこの run で直したので AC10 も付けた。`PR created` は /pr の担当なので付けていない。ほかの文は 1 字も変えていない |
| `docs/insights/events/2026-10-13-org-merge-order.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-merge-order`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記で、コミットには入れない |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。4 面の `shasum -a 256` は 1 種類。動詞の表の `merge-order` の行(`:183`)の分類の順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)と並びの順は、コードの `classifyMergeTarget`・`classifyGatePassed`・`mergeClassOrder` と合い、README の直した文とも同じ。見張りなしで使える動詞の一覧(`:1097` から `:1103`)は `approve`・`gate`・`merge-order` を挙げる |
| `README.md:124`(動詞の表) | 変更なし。`merge-order` が動詞の列と 1 文の説明にある |
| `AGENTS.md:90`(repo map の `internal/org/` の行) | 変更なし。`mergeorder.go` を挙げる。`internal/cli/` の行は `org` でまとめており、新しいファイル名を挙げない既存の書き方と同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の「7c-1 で決めたこと」の注) | 変更なし。分類の順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)、`ready` の定義、7c-2 と 8a への送りが実装と合う。FR-7 と FR-8 の checkbox は 7c-2 と 8a が残るので未完のまま |
| `docs/quality/quality-gates.md` と template の写し(「Cross-org watch gate」以外の行) | 変更なし。merge-order は gate の記録を読むだけで、新しい検査や拒否を足さない。「Quality pipeline gate」の行は `ralph org gate` の拒否を書き、merge-order を述べない |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md` | 変更なし。definition-of-done の org の節は仕様と rule を指すだけ。repo-map は `.harness/state/org/` の要約で、merge-order は台帳に何も書かず、動詞も script も足さない |
| `.claude/rules/ralph/`(`agent-messaging.md`、`post-implementation-pipeline.md`、`subagent-policy.md`、`model-routing.md`、`git-commit-strategy.md`)と template の写し | 変更なし。merge-order は agmsg のメッセージを送らず、受信箱も読まない(B2。verify が `mergeorder.go` を grep して確かめた)ので、TYPE の表にも承認の規則にも影響しない |
| `docs/recipes/`(`codex-seat-permissions.md` とその template の写し、`worktrees.md`) | 変更なし。reviewer と leader の雛形は触っておらず、recipe の TASK の前提は変わらない |
| `docs/tech-debt/README.md` の追加 4 行(`:233` から `:236`)と変更 1 行(`:226`) | 変更なし。挙げた名前(`gateOrgLife`、`StartFeature`、`classifyGatePassed`、`listOriginBranches`、`realMergeOrderExec`、`lookupMergePRs`、`readMergeLedger`、`readJSONLines` など)は verify が今のコードで確かめた。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き直す。`run-static-verify.sh` の `tech-debt README plan references` は OK |
| `docs/tech-debt/README.md:229`(gate が feature ブランチのスクリプトを走らせる行)の Trigger「stage 7c's merge order wanting a stronger guarantee」 | 発火していない。merge-order は gate の記録だけを読み、保証を強めていない(verify の情報の項目と同じ判断)。変更なし |
| `internal/org/gate.go:27`・`:164` のコメント(「Stage 7c's merge order reads that record」) | 合う。7c-1 が読むので今も正しい。触らない |
| `ralph org --help`、`ralph org merge-order --help`(`internal/cli/org.go`、`org_merge_order.go`) | 変更なし。verify が binary で確かめた。文字列は変えていない |
| `docs/reports/`、`docs/evidence/` | 編集していない(この report を除く)。`docs/evidence/` の org-merge-order の 5 ファイルで、`grep -c '/Users/[a-z]'` はどれも 0 件(`docs/quality/definition-of-done.md` が求める `$HOME` の `~` への置き換えは済んでいる) |

## Left open

- self-review の L1 から L8 と、test の Test gaps(M05、C12、M03、C06・C07、M09。どれも 1 行の probe が用意されている)は、コードとテストと skill に触れる直しなので、この run では直していない。直すと `/self-review` から回し直しになる(cycle 2)。self-review は、直さずに進めるなら L2(PR の lookup の失敗が entry の側で区別できない)と L4(外のコマンドの実行関数の 4 つ目の写し)を 1 行にまとめて tech-debt に足すよう勧めている。cycle 2 で直すかどうかが決まっていないので、この run では tech-debt に足していない。cross-review のあとも直さずに `/pr` へ進むなら、足すか、PR の本文に known gaps として書く(orchestrator の判断)
- verify の Coverage gaps のうち、ls-remote の 30 秒の timeout と ssh の passphrase は未確認のまま(tech-debt の `:234` に記録済み)。本物の GitHub と `gh` の確認は、test の real-origin の evidence で済んでいる
- release のノート(計画の Rollout が言う、新しいサブコマンド `ralph org merge-order`(`--json`、`--all`)、`git ls-remote` でリモートを読むこと、`gh` があれば merge 済みと閉じた PR を外すこと)と PR の本文は、この run の範囲に含めていない(`/pr`)
- 計画の `PR created` は `/pr` が付ける

## Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: `IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`(quality-gates の root と template を同じ文で変えたあと)
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 13 skill(s) in lock-step`。skill は触っておらず、4 面の `shasum -a 256` は 1 種類
- `./scripts/check-template-purity.sh`: `PASS: no meta-repo-specific references found in templates.`(`templates/base/docs/quality/quality-gates.md` を変えたので打った)
- `./scripts/run-static-verify.sh`(changed の範囲): rc 0、`==> All verifiers passed.`。`check-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`tech-debt README plan references`、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned ac34ccf3..e20f2616 against origin/main: clean`。wrapper が書いたログは gitignore の対象で、コミットしていない
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md`: `c0b1e9230999`(checkbox の印を付けたあとも、`- Approved:` の行と同じ)
- 計画の差分: `git diff --numstat` が 14 / 14 で、変わった行はすべて `- [ ]` から `- [x]` への 1 字だけ
- tech-debt の行: HEAD と比べて変わったのは `:231` の 1 行だけ。5 列のまま、変わった列は Trigger だけ、列ごとの backtick は偶数、`file:line` の形は 0 件、U+FFFD は 0 件、計画のパスは `docs/plans/archive/2026-10-13-org-gate.md` の 1 つのまま(Python。スクリプトは scratchpad)
- `git diff --check`: 出力なし
- `./scripts/insights-append.sh --slug org-merge-order ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-merge-order.jsonl` が未追跡のまま、ほかに新しい `??` のファイルはない

## Cycle 2

- Date: 2026-10-14(JST)
- Pipeline cycle: 2(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。差分の base は `ac34ccf3`、branch は `feat/org-merge-order`。run 開始時の HEAD は `d0e6b600`(run 開始時の値で、この cycle の commit の SHA ではない)
- 先行 report の `Cycle 2`: self-review(C2-1 から C2-6、LOW 6、Merge: yes)、verify(pass、LOW 2 件の D3 と D4)、test(pass、生き残った mutant 15 件のうち 10 件に、HEAD で通る probe の行がある)
- 項目の番号: C2-n は self-review の `Cycle 2` の所見の表、D3・D4 は verify の `Cycle 2` の「文書のずれ(cycle 2)」の表、L3・L5・L6 は self-review の cycle 1 の所見の表、S04・O03 などは test の `Cycle 2` の「Mutation(cycle 2)」の番号
- 内容の commit: この節、直した文書とコメント、tech-debt の 1 行を含む 1 つの commit(`docs: sync docs for org-merge-order (cycle 2)`)。指示どおり、`docs/insights/` と `docs/reports/cross-review-triage-org-merge-order.md` には触れていない。sync_docs の cycle 2 の insight event は足していないので、orchestrator が足す

### Summary

verify の D3(gh の PR の突き合わせを例外なしに言い切る 4 か所)と D4(`internal/org/seat.go` の package doc が `git remote get-url origin` を挙げない)を直した。`internal/cli/org_merge_order_test.go` の先頭のコメントは cycle 2 のテストに合わせ、self-review と test の cycle 2 の小さい所見は tech-debt の 1 行にまとめた。コードの動きは変えていない。Go のファイルで変えたのは、help の文字列(`Long` の PR の段落、出力は変わる)とコメントだけで、`gofmt -l internal/` の出力は空。

D3 は self-review の C2-3 と同じ件で、self-review は「help の 1 句だけ直し、残りは known gaps に書く」案も挙げていた。cycle 2 は文書だけで 5 か所を直せる(skill は 4 面を機械的に合わせる)ので、5 か所とも直した。

### Changes made

| File | 項目 | Change |
|------|------|--------|
| `internal/cli/org_merge_order.go`(`Long` の PR の段落) | D3 | 「gh's PRs are used only when their repository is origin's ...」の文のあとに、「The check has three gaps」から始まる文を足した(5 行に折り返し)。(a) origin の URL に owner/repo がない(ローカルのパス、`file://`、`git+ssh://` のような読まないスキーム)ときは確かめず、gh の PR をそのまま使う。(b) host は比べない。(c) 確かめは gh が返した PR にだけ走るので、gh が読む repository にそのブランチの PR がなければ注記は出ない。help の読み手の手元には tech-debt の行がなく、skill と README もいまは tech-debt を指していないので、行への pointer は付けていない。折り返しは既存の行と同じ幅(最大 79 桁、タブ込み) |
| `.claude/skills/org/SKILL.md`(動詞の表の `merge-order` の行と、「merge してよい順」の節の `gh` の項目)と、その 3 つの写し | D3 | 表の行は、突き合わせを言う文のあとに「確かめない場合が 3 つある: ...」を足した。節の項目は、「注記で言う。」のあとに 5 行(「確かめない場合が 3 つある。」から「注記も出ない。」まで)を足した。節の項目の新しい行の表示幅は 69〜71 桁(最後の 1 行を除く)で、隣の行(68〜75 桁)に合わせた。`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り直し、`templates/base/.claude/skills/org/SKILL.md` と `templates/base/.agents/skills/org/SKILL.md` は `cp` で合わせた(`sync-skills.sh` は template を作り直さない)。4 面の `shasum -a 256` は 1 種類 |
| `README.md:262`(`ralph org merge-order` の段落) | D3 | 「a note says to run `gh repo set-default`.」のあとに、「The check has three gaps: ...」の 1 文を足した。段落のほかの文は変えていない |
| `internal/org/mergeorder.go`(先頭のコメント、`origin` の URL を読む件の段落) | D3 | 「The check has three gaps (`mergeOriginRepo.check`)」の 3 行を足し、例外の詳細は `mergeOriginRepo.check` の doc に任せた。コメントを組み直したので、続く 2 文の折り返しが変わっている |
| `internal/org/seat.go`(package doc) | D4 | 外のコマンドの一覧の `git ls-remote` in mergeorder.go を、「`git ls-remote`, plus `git remote get-url origin` once gh returned a PR, in mergeorder.go」にした |
| `internal/cli/org_merge_order_test.go`(先頭のコメント) | 項目 3 | 古かった 4 点を直した。(1) org 層のテストが受け持つ範囲に、repository の確かめ(C5)と gh の失敗のまとめを足した。(2) このファイルが確かめる出力に、gh が失敗したときの注記と、origin と別の repository の PR の注記を足した。(3) git の wrapper を「本物の git を exec する」とだけ書いていたが、`$MERGE_ORDER_ORIGIN_URL` があれば `git remote get-url origin` に答える(`mergeOrderGitWrapper`)ので、その旨を足した。(4) origin はこの変数がなければローカルのパスで、ローカルのパスは owner/repo を持たないので確かめが飛ばされ、gh の PR がそのまま使われることを段落に足した。コメントだけ |
| `internal/org/mergeorder_test.go`(先頭のコメント) | 掃引 | 偽の `MergeOrderExec` が「`git ls-remote` と gh の代わり」と書いていたが、`git remote get-url origin` の代わりもするので 1 句足した。handoff の項目 3 は CLI のテストだけだったが、同じ cycle 2 のずれなので同じ commit で直した。コメントだけ |
| `docs/tech-debt/README.md`(末尾に 1 行) | 項目 4 | org-merge-order の cycle 2 の小さい所見を 1 行にまとめた。5 列、英語。コード・文書の項目 (a) から (h)(C2-1、C2-2、C2-4、C2-5、C2-6、cycle 1 の L3・L5・L6)と、test の生き残った mutant のうち probe で落とせる 10 件を、テストの穴の項目 (i) から (n)(S04、O03 と O04、D02、L04 と L05、G04、P04 と P05 と O16)にした。C2-3 は同じ commit で直したので項目にせず、Why deferred に書いた。probe で落とせない 5 件(O06、O19、O20、O21、U02)は「本物の入力では動きが変わらない」ので項目にせず、Why deferred に名前だけ挙げた。名前は関数・test・定数で書き、`file:line` は 0 件 |
| この節 | - | 追記。cycle 1 の節は変えていない |

C2-1 は、行の (a) に「`TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` の「has no org to name」の部分が今の欠落を固めているので、直すときはその assertion を変える」と書いた。test report の mutant H01(self-review の推奨の直しを入れたもの)が同じ test で red になったことと合う。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md:124`(動詞の表)、`AGENTS.md:90`(repo map の `internal/org/` の行) | 変更なし。AGENTS.md は「one `git ls-remote`」と言い、今も正しい(`git remote get-url origin` は gh が PR を返したときだけで、ls-remote は 1 回のまま)。README の表は動詞と 1 文の説明だけで、gh の突き合わせは言わない |
| `docs/quality/quality-gates.md` と template の写し | 変更なし。「Cross-org watch gate」の行は cycle 1 で `merge-order` を足した。cycle 2 は新しい検査も拒否も足さない |
| `docs/tech-debt/README.md` の既存の 2 行(`mergeOriginRepo.check` の行と、`--json` の `pr` の行) | 変更なし。前者は D3 の 3 つの例外((a) owner/repo のない origin、(b) host、(c) PR のない別の repository)をもう持っていて、直した 4 か所の言い方と食い違わない。後者の名前は今のコードにある |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の「7c-1 で決めたこと」の注) | 変更なし。verify が cycle 2 の情報で「origin の repository との突き合わせは書かないが、食い違う記述はない」と確かめている。突き合わせの決定は計画の Progress にある |
| 計画 `docs/plans/active/2026-10-14-org-merge-order.md` | 変更なし(本文も `## Progress checklist` も)。`./scripts/plan-visual.sh digest` は `c0b1e9230999` で、`- Approved:` の行と同じ。cycle 2 の逸脱は Progress の cycle 2 の項目に書いてある |
| `.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md` | 変更なし。cycle 2 は動詞、script、hook、rule を足していない |
| `ralph org merge-order --help` | 変わった(D3 の段落の追加)。テスト(`-run 'MergeOrder\|LegacyLedger\|Help'`)は help の文字列の内容を固めておらず、通った |
| `docs/reports/`(この report を除く)、`docs/evidence/` | 編集していない |

### Left open

- tech-debt の新しい行の (a) から (n) は直していない。cycle 2 は上限の最後で、直すと 3 回目の pipeline が要る。唯一読む人に当たるのは (a)(C2-1)で、次の手は 1 つに決まる(`--org-id` を付けて打ち直す)
- この commit は Go のファイル 5 つ(`org_merge_order.go`、`org_merge_order_test.go`、`mergeorder.go`、`mergeorder_test.go`、`seat.go`)を触る。変えたのは help の文字列 1 段落とコメントだけで、動きは変えていない。pipeline を回し直すかどうかは lead の判断
- sync_docs の cycle 2 の insight event(`./scripts/insights-append.sh --slug org-merge-order --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`)は、指示で `docs/insights/` に触れないので足していない
- release のノートと PR の本文は `/pr` の担当。PR の本文の known gaps には、tech-debt の新しい行と、`mergeOriginRepo.check` の行(3 つの例外は help・skill・README に書いた)を挙げるとよい
- 計画の `PR created` は `/pr` が付ける

### Verification

commit の前に、worktree のルートで打った。

- `gofmt -l internal/`: 出力なし(`gofmt -w` は使っていない)
- `go test ./internal/org/ ./internal/cli/ -run 'MergeOrder|LegacyLedger|Help' -count=1`(go1.26.0): `ok internal/org 4.198s`、`ok internal/cli 56.194s`
- `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -run 'MergeOrder|LegacyLedger|Help' -count=1`: `ok internal/org 5.415s`、`ok internal/cli 48.899s`
- `./scripts/run-static-verify.sh`(changed の範囲): rc 0、`==> All verifiers passed.`。`check-sync.sh`(`IDENTICAL: 167`、`DRIFTED: 0`、`PASS: all files in sync.`)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(`13 skill(s) in lock-step`)、`check-template-purity.sh`(`PASS`)、`tech-debt README plan references`(OK)、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned ac34ccf3..d0e6b600 against origin/main: clean`。wrapper が書いたログ(`docs/evidence/verify-2026-10-14-001031.log`)は gitignore の対象で、コミットしていない
- `./scripts/check-sync.sh`: `IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS`
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 13 skill(s) in lock-step`。4 面の `shasum -a 256` は 1 種類
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md`: `c0b1e9230999`(計画は触っていない)
- tech-debt の行: HEAD と比べて、既存の 239 行は 1 字も変わらず、末尾に 1 行だけ増えた。5 列(`|` で分けると 7 片)、列ごとの backtick は偶数、`file:line` の形は 0 件、U+FFFD は 0 件、計画のパスは `docs/plans/active/2026-10-14-org-merge-order.md` の 1 つ(Python。スクリプトは scratchpad)
- `git diff --check`: 出力なし
