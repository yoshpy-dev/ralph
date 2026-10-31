# sync-docs report: org-followups

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(issue #270。承認の digest `6ef9fe7cb76c` は run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。plan の編集は `## Acceptance criteria` の印 8 つと `## Progress checklist` だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json`)。base は `origin/main` の `1cc84ffa`、branch は `feat/org-followups`。run 開始時の HEAD は `523b9091`(verify の I2 を受けた F4 `a4cab358` と test の report のあと)。文書の変更とこの report は 1 つの commit に入れる
- 先行 report: `docs/reports/self-review-2026-10-30-org-followups.md`(run 2、Merge: yes)、`docs/reports/verify-2026-10-30-org-followups.md`(pass。LOW の V1・V2 をこの run に回した。INFO の I1 は計画の承認し直しで、I2 は F4 で済んでいる)、`docs/reports/test-2026-10-30-org-followups.md`(pass。Test gaps 7 つ)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、`523b9091` の時点): 41 ファイル、+6499/-97。この run の分は 8 ファイル、+43/-41
- 制約: Go のファイルは変えていない。打ったのは `check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`、`go test ./internal/org/ -run Skill` と `/org` skill を読む org のテスト 56 本、`git diff --check`、U+FFFD の走査、`plan-visual.sh digest`、tech-debt の欄数・backtick・plan 参照の確認、HEAD の binary での `followup` の probe。`run-verify.sh` と `run-test.sh` は打っていない

## Summary

follow-up の動詞と節の文書(help、`/org`・`/director` skill、README、AGENTS.md、仕様の注)は S5 と self-review の直し(F2・F3)と F4 で済んでいて、今のコードと合っていた。残っていたずれと、依頼の項目を次のように直した。

- 見張りなしで打てる動詞の一覧 2 か所に `followup` の動詞がなかった(依頼の外。下の「判断を残したもの」)。quality-gates の 2 つの写しと `/org` skill の 4 つの写しに足した。
- tech-debt: Trigger に当たった 6 行(V1)に持ち越しの注、`status.go` の行の行数(V2)、`mergeorder.go` の 2 つの行の相互の指し、test gaps 1〜7 の新しい 1 行。
- plan: AC1〜AC8 と artifact の 3 つに印、Progress に sync-docs の行を 1 つ。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(`/org` skill の command table の行、org-stop-all の 2 つの所見) | V1。(a) の Trigger の末尾に持ち越しの注。`followup` の 3 行(表の 1 行として 664・486・343 字)を足し、`status` の行を 405 から 623 字に、`merge-order` の行を 3,662 から 4,091 字に伸ばした。`stop` と `disband` の欄は変わっていない。Related に plan |
| 同(org-gate の cycle 2 の小さい所見) | V1。`lockfile.go` の (g) が名指す 3 つの doc を、この branch は書き直した(director wait の lock と follow-up の lock を一覧に足した)が、「withFileLock serves all five locks」と「the flock mechanism behind withManifestLock」は残った。`withFileLock` が仕えるのは 4 つで、manifest の lock は `withFileLockContext` を通る。(g) は開いたままと注に書いた。Related に plan |
| 同(merge-order の PR の小さい残り) | V1。`mergeorder.go`・`org_merge_order.go`・`TestMergeOrder_ResultJSONShape` を変えたので Trigger の前半に当たった。文字の付いた項目には触れていない。(g) は広がった: gh の上限 1000 を、merge-order の help、README、`/org` skill の follow-up の節(4 つの写し)、その pin、follow-up のテストの argv にも手で書いた。Related に plan |
| 同(merge-guarantee の小さい残り) | V1。`MergeOrder` を変えたので Trigger に当たった。(b) と (d) は広がった: follow-up の照合は自分の `mergeOriginRepo` を持つ(`checkFollowupPRs`)ので、follow-up のある merge-order の 1 回で `git remote get-url origin` が最大 3 回になる。Related に plan と `checkFollowupPRs` |
| 同(`mergeguarantee.go` と `mergeorder.go` の大きさの行) | V1。`mergeorder.go` を 1291 から 1322 行にしたが runner は動かしていない、と注。この branch の `mergeorder.go` の行(ranking か PR の引き方を外へ)を名指し、runner も同じ変更で動かせると書いた。Related に plan(Risks) |
| 同(`ralph status` の座席の表で `spawn_close_failed` が見えない行) | V1。座席の表を変えた(予約の `reserved:` の行、座席のない組では見出しを出さない)ので Trigger に当たった。座席の行、状態の列、roster は変えていない。Related に plan |
| 同(`status.go` が 800 行を超えた行、この branch の行) | V2。「901 to 959」を「901 to 963」に、`status_followup.go` を 187 から 189 行に。base `1cc84ffa` と `523b9091` で数えたと書いた |
| 同(`mergeorder.go` が 1322 行の行、この branch の行) | Trigger に、もう 1 つの大きさの行が同じ契機で runner(`realMergeOrderExec`、`realMergeDetachedExec`)の移動を求めていることを 1 文。行番号でなく、行の中身で名指した |
| 同(新しい行) | 依頼の項目 4。test の Test gaps 1〜7 を 1 行に。(1) Done の lock は確率でしか固定されていない(m3b は 20 回中 7 回)、(2) 書き込みの失敗、(3) `head=` のない Details、(4) origin の照合の途中の中断、(5) CLI の層の store の失敗、(6) status の CLI の行の 2 つの場合、(7) 実物の gh。Trigger は項目ごとに関数の名前で書いた |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(Cross-org watch gate の行) | 見張りなしで打てる動詞の一覧に「the `followup` verbs」を足した。2 つの写しのこの行は同じ文(ファイルは KNOWN_DIFF) |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`(見張りの節、見張りなしで使える動詞の段落) | 一覧に `followup` の動詞を足し、5 行を折り返し直した(表示幅 74〜78)。root と template の `.claude` は Edit で直し、`.agents` の 2 つは `sync-skills.sh` を 2 回(root と template)で作り直した。4 つは `cmp` で同じ |
| `docs/plans/active/2026-10-30-org-followups.md` | AC1〜AC8、Review・Verification・Test artifact created に印。Progress の Implementation started の下に sync-docs の行を 1 つ(F4 の SHA `a4cab358` もそこに書いた)。digest は前後で `6ef9fe7cb76c` |
| `docs/reports/sync-docs-2026-10-30-org-followups.md` | この report |
| `docs/insights/events/2026-10-31-org-followups.jsonl` | `sync_docs` の行(cycle 1、pass)を足した。ファイル名は UTC の日付 |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| 1 文書と branch の動きの一致 | 下の「Surfaces checked」。ずれは見張りなしで打てる動詞の一覧の 2 か所(直した)。follow-up の動詞、status と merge-order の節(SOURCE の列、JSON の object)、予約の行、D2・D5、M1 の片付けの例外、割れたときの規則は、help・skill・README・AGENTS.md・仕様の注で今のコードと合っていた |
| 2 V1 の 6 行 | 6 行とも持ち越しの注を足した。払える文書だけの直しはなかった(235 行の (g) は Go のコメント、246・251 行の項目と 177 行の (a) は Scope の外)。246 行の (g) と 251 行の (b)・(d) は、この branch で広がったことも書いた |
| 3 V2 の行数 | 直した。`wc -l` は `status.go` 963、`status_followup.go` 189(HEAD `523b9091`) |
| 4 test gaps 1〜7 | 1 行にまとめて足した(5 欄) |
| 5 plan の印 | AC1〜AC8 は verify の表(すべて「満たす」、AC4・AC6 のテストは /test)と test の report(`run-test.sh` rc 0、`TestStatusCmd_FollowupsRunNoGH` と skill の pin を含む)で確かめた。digest は `6ef9fe7cb76c` のまま |
| 6 大きさ | 下の「800 行の目安」 |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `ralph org --help`、`ralph org followup --help` | 合っている | HEAD の binary で読んだ。動詞の一覧に `followup`、`--org-id` を断る一覧にも `followup` |
| `ralph org merge-order --help` と `/org` skill の `merge-order` の行 | 合っている | 列は help・skill・コード(`printMergeOrderFollowups`)・pin(`mergeOrderFollowupHeader`)とも ID・ISSUE・AFTER_BRANCH・STATE・SOURCE・PR・DETAIL(F4 のあと) |
| `ralph status --help` と `/org` skill の `status` の行 | 合っている | 列は ID・ISSUE・AFTER_BRANCH・STATE・SOURCE・DETAIL(`printStatusFollowups`、`statusFollowupHeader`)。`reserved:` の行と `reservation` |
| 割れたときの規則(ready と waiting なら merge-order、`unknown` なら理由を添えて status) | 合っている | `git grep` で、英語の文は AGENTS.md・README・tech-debt・help 4 か所(followup 2、merge-order、status)、日本語の文は `/org`・`/director` skill の 4 つずつの写しと仕様の注にある。「merge-order の答えを使う」の文を `git grep` で全部読み、`unknown` の節のないものは、計画の Progress の F2 の記録(過去の出来事)だけだった |
| `/director` skill の「merge を待つ issue」と進捗・merge の報告 | 合っている | 3 手順と、merge の報告で follow-up の節を読むこと |
| README の動詞の表、`ralph status` の行、merge-order の段落 | 合っている | 表の動詞に `followup`。README は列を書いていないので F4 で古くならない |
| AGENTS.md の地図 | 合っている | `internal/org/` の行に `followup.go` と `mergeorder_followup.go`。`internal/cli/` の行は動詞を親の `org` で挙げるだけ |
| 仕様の注(FR-9、`docs/specs/2026-10-07-org-multi-org-director.md`) | 合っている | D2、D5、M1 と R2-1 の例外、#271 への送り |
| `docs/quality/quality-gates.md`(root と template) | 直した | 見張りなしで打てる動詞に `followup` がなかった |
| `/org` skill の見張りの節 | 直した | 同じ一覧 |
| `docs/architecture/repo-map.md`、`.claude/rules/ralph/`、CLAUDE.md、definition-of-done | 変更なし | `internal/org` のファイルの一覧も、動詞の一覧も持たない。follow-up はメッセージでないので `agent-messaging.md` は関係しない |
| `docs/tech-debt/README.md` の欄数・backtick・plan 参照 | 通った | 変えた 8 行と新しい 1 行は、すべて 5 欄で、各欄の backtick と `~~` が偶数。`file:line` の参照は足していない。`docs/plans/(active\|archive)/` の参照に存在しないパスはない(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ手順を手で打った) |
| U+FFFD、`git diff --check` | なし | 変えたファイルを走査 |

## 800 行の目安

| ファイル | base → HEAD | 台帳 |
|---------|-------------|------|
| `internal/cli/status.go` | 901 → 963 | ある(この branch の行。V2 で数を直した) |
| `internal/org/mergeorder.go` | 1291 → 1322 | ある(この branch の行。`mergeguarantee.go` の大きさの行にも持ち越しの注) |
| `internal/cli/org.go` | 2080 → 2084 | ない。前から 800 行を超えていて、この branch が変えたのは `ralph org` の help の 1 文、`--org-id` の usage、`followup` の登録の 1 行だけ。行は足さず、ここに書く |
| `internal/org/mergeorder_test.go` | 3000 → 3001 | ない。前から超えていて、`TestMergeOrder_ResultJSONShape` の key の行の置き換えと 1 行の追加だけ。行は足さず、ここに書く |
| `/org` skill(4 つの写し) | 1979 → 2040 | ない。Markdown で、800 行の目安はコードのファイルのもの |
| `/director` skill(4 つの写し) | 1196 → 1231 | ない。同上 |
| 新しいファイル | 最大は `internal/cli/status_followup_test.go` の 752 | すべて 800 行未満 |

前から 800 行を超えていて、この branch が数を増やしていないもの: `internal/cli/org_regate_test.go`(1273)、`internal/cli/org_director_test.go`(1165)。どちらも help の pin の 1 行を置き換えただけ。

## 判断を残したもの

- 見張りなしで打てる動詞の一覧(quality-gates の Cross-org watch gate の行、`/org` skill の見張りの節)に `followup` の動詞を足した。依頼の範囲は follow-up の文書だったが、この 2 か所は手で保つ一覧で、ないと `followup` が見張りを要るように読める。`followup` の CLI は watch gate を作らない(`newOrgFollowupStore` は台帳を解決して store を作るだけ)。HEAD の binary を、`git init` しただけの一時の repo と見張りのない台帳で打ち、`add`・`list`・`done` が rc 0 で、台帳に `followups.jsonl` と `followups.lock` だけができたことを見た。
- 246 行の (g) と 251 行の (b)・(d) は、払わずに「広がった」と書いた。直すのは merge-order の別の変更(上限の定数の公開、origin を 1 回読んで渡す)で、計画の Scope の外。
- 235 行の (g) の文言(「all five locks」)は Go のコメントなので、この run では直さない。直すと `.go` の変更になり、パイプラインの回し直しが要る。

## PR の Known gaps に回すもの

1. test gaps 1〜7(tech-debt の新しい行)。とくに (1) Done の lock は 1 回の実行では見逃しうる(m3b は 20 回中 7 回だけ落ちた)。
2. GitHub で記録より前に merge し、記録のあとに片付けた PR の follow-up は、status が `ready`、merge-order が `waiting` になる(tech-debt の M1 の行。文書は割れたときの規則を書く)。
3. 実物の gh と GitHub では確かめていない(test gaps の 7)。

## PR 本文の材料

- verify の I1: 計画の D5 は R2-1 の形に直して承認し直した(digest `6ef9fe7cb76c`)。I2: merge-order の follow-up の表に SOURCE の列を足した(F4 `a4cab358`)。
- release のノート(計画の Rollout): `ralph org followup` を足した。`ralph status` が予約と follow-up を、`ralph org merge-order` が着手できる follow-up を出す。

## Verification (この pass)

| コマンド | 結果 |
|---------|------|
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-template-purity.sh` | PASS |
| `go test -count=1 ./internal/org/ -run Skill` | ok |
| `/org` skill を読む org のテスト(`org_skill_send_sync_test.go`、`prompts_verification*_test.go` 3 つ、`report_names_sync_test.go`、`send_defaults_sync_test.go`、`skill_doc_followup_test.go`、`skill_followup_test.go` のトップレベルの 56 本) | `--- PASS` 56、`--- FAIL` 0 |
| `git diff --check`、U+FFFD の走査 | 空 |
| `./scripts/plan-visual.sh digest <plan>` | `6ef9fe7cb76c`(承認のときと同じ) |
| `git diff --numstat`(commit の前) | plan 12/11(印 11 と新しい行 1)、tech-debt 9/8、quality-gates 1/1 が 2 つ、skill 5/5 が 4 つ |
| tech-debt の欄数・backtick・`~~`・`file:line`・plan 参照 | 通った(上の表) |
| HEAD の binary で `ralph org followup add`・`list`・`done`(見張りなし) | rc 0 が 3 つ |

help の文と Go のテストが固定する文は変えていないので、`internal/cli` の pin は打っていない。

## Insight event

`./scripts/insights-append.sh --slug org-followups --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` は rc 0。`cycle` は 1。`docs/insights/events/2026-10-31-org-followups.jsonl` に 1 行が足され、ほかのファイルは増えなかった。
