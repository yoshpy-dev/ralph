# sync-docs report: org-start-issue

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-start-issue.md`(承認の digest `cbeb26942ff9`。この run の前後とも同じ)。issue #365
- Branch: `feat/org-start-issue`。run 開始時の HEAD は `701421da`(origin/main は `ccdf6293`、81530fb0 で取り込み済み)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review `70fd0fc2`(Merge: yes、MEDIUM 3、LOW 6。M1・M2・L2 の読み位置・L6 は `52e6cfd3` で直った)、verify `8b6bb6a3`(pass、HEAD 52e6cfd3)、test `701421da`(pass、`run-test.sh` rc 0、変異 46 個がすべて落ちた)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。skill は触っていないので `sync-skills.sh` は要らなかった
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff ccdf6293...HEAD` は 42 ファイル、+2,786/-182。Go は全体で +1,969/-138、テスト以外の Go が +355/-110。walkthrough を書くかどうかは `/pr` が決める

## Summary

台帳 `docs/tech-debt/README.md` に、この PR が Trigger を満たす 27 行へ日付つきの引き継ぎの注を足した(依頼の 25 行と、照合で見つけた 2 行)。482 の (a) は `52e6cfd3` で払ったので、取り消し線で閉じて、(b)(c)(e) を引き継いだ。新しい行は 3 つ。計画の本文は digest が変わるので直さず、self-review L1 は PR の本文の Known gaps に回す。仕様 `2026-10-07-org-multi-org-director.md` の FR-4 の項に、古い binary の補償で `issue=` が落ちることを 1 文足した。README、AGENTS.md、2 つの仕様、`/org` と `/director` の skill(各 4 写し)、leader の雛形は、S3 が書いたものと食い違わなかったので直していない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/tech-debt/README.md` | 27 行の Trigger のセルの末尾に `2026-11-22: feat/org-start-issue (#365) hit ...` の注、Related のセルの末尾に `docs/plans/active/2026-11-22-org-start-issue.md`。482 は (a) を `~~...~~ Closed in ... (52e6cfd3, 2026-11-22)` と `~~...~~ (closed)` で閉じた。新しい行 3 つ(555〜557 行目)。554 行から 557 行 | この report の commit |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-4 の 2026-11-22 の項(issue #365、98 行目)に、古い binary が自己終了の補償で結びつきを書き戻すと `issue=` が落ち、reseat の leader は issue の行を持たず、director の確かめも拾えないことを 1 文足した | この report の commit |
| `docs/plans/active/2026-11-22-org-start-issue.md` | Progress の `Test artifact created` の次に `Sync-docs artifact created` の行と、Known gaps の子の項目。本文は変えていない。digest は前後とも `cbeb26942ff9` | この report の commit |
| `internal/cli/org_start_issue_test.go` | コメントが引く `docs/plans/active/2026-11-22-org-start-issue.md` を `2026-11-22-org-start-issue` に直した(コメント 1 行のみ)。`archive-plan.sh` は台帳しか書き換えないので、`/pr` のあとで宙に浮く | この report の commit |
| `docs/reports/sync-docs-2026-11-22-org-start-issue.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-org-start-issue.jsonl` | `sync_docs` の行(cycle 1、pass)。1 行から 2 行(UTC の日付が 11-22 なので、test の行と同じファイル。`git status --short` は `M` だけで、迷子のファイルはない) | この report の commit |

コード(Go)に触れたのは、上のテストファイルのコメント 1 行だけ。`gofmt -l` と `go vet ./internal/cli/` は通った。パイプラインの再実行になるかどうかは lead が決める(#363 の sync-docs も同じ種類のコメントの直しを入れた)。

## 依頼の項目

### 1. 台帳の Trigger の注

行番号は依頼どおり 52e6cfd3 の値で、台帳は PR の diff に hunk がなく、HEAD でも同じ行にあった。最初の語で行を見つけた。注の形は「`2026-11-22: feat/org-start-issue (#365) hit <変えたもの>, but did not pay it: <理由>. Carried over to the next change that meets this trigger`」で、Related に計画を `docs/plans/active/...` の path で足した(`/pr` の archive が書き換える。`verify.local.sh` の `check_tech_debt_plan_refs` の同じ検査を単独で回して、存在しない参照は 0 件)。

| 行(最初の語) | 当たった項目 | この PR が触ったもの | 扱い |
|------|------|------|------|
| 182 `Two documentation findings of org-stop-all` | (a) | `/org` skill のコマンド表の `start`・`status`・`merge-order` の行(全体の長さ 2,516→2,841、1,047→1,168、5,761→5,861 文字) | 繰り越し。`stop`・`disband` の行は触っていない |
| 190 `What a leader is told about its reservation` | (a) | leader の prompt(手順 14。`leader.md` 1,668→1,685 行) | 繰り越し |
| 194 `Test gaps left by the org-limits-reserve cycle 2 fix` | (c) | `ralph org status` の `Long` | その 1 文だけ払った(`TestOrgStartIssueHelp_DescribesTheFlag` が `status --help` を読む)。残りは繰り越し |
| 197 `A ralph of another version on the pane's PATH` | `newOrgStartCmd` の `RunE` | `--issue` の flag と `StartFeature` への受け渡し | 繰り越し。版の食い違いが新しい `issue=` にも及ぶことを書いた |
| 202 `Small findings of org-feature-worktree` | (g) | `leader.md` の機能ごとの org の節(手順 14) | 払うものなし((g) は済み。新しい固定は `squashSpace` で折り返しに強い) |
| 203 `Test gaps left by org-feature-worktree` | (e)(f) | `runOrgStartPlan`、`StartFeature`(`readStartFeature` の issue の形の検査) | 繰り越し。coverage は `runOrgStartPlan` 96.0%、`newOrgStartCmd` 97.9% |
| 225 `Small leftovers of org-plan-approval` | (c) | `featureBindingFromTokens` に `issue` の鍵 | 繰り越し |
| 252 `Small leftovers of the org-merge-order PR` | 標準の Trigger | `mergeorder.go`(`MergeOrderBinding.Issue`、`collectMergeTargets`)、`org_merge_order.go`(`Long`) | 繰り越し |
| 311 `internal/org/mergeguarantee.go (877 lines` | `mergeorder.go` の次の変更 | `mergeorder.go` 1,512→1,519 行 | 繰り越し。**依頼になく、照合で見つけた行** |
| 370 `internal/org/verbs.go is 2277 lines` | leader の雛形のテスト | 新しい `prompts_leader_issue_test.go`(119 行)、`director_skill_sync_test.go` 1,318→1,320 | 繰り越し。どのファイルも各自の数 +300 を超えない |
| 382 `RC4` | feature-org の手順 | 手順 14 | 繰り越し。分割計画の例(`### 分割計画`)は触っていない |
| 389 `After step 14 of the leader template` | 手順 14 | `Closes` の文 | 繰り越し |
| 395 `Four test files went past the 800-line guide` | 4 ファイルのどれか | `director_skill_sync_test.go` | 繰り越し。新しい固定は新ファイル `skill_issue_test.go` |
| 396 `Two windows the merge cleanup of #263` | (b) | `StartFeature` の検査(issue の形) | 繰り越し |
| 401 `resolvedOrClean` | 呼び手 | `feature.go` 899→925 行、`feature_test.go` | 繰り越し。`resolvedOrClean` の呼びは編集していない |
| 423 `internal/org/mergeorder.go grew from 1291 lines` | `mergeorder.go` の次の変更 | 同じ | 繰り越し。**依頼になく、照合で見つけた行** |
| 424 `Ten test gaps of org-followups` | (10) | `merge-order --json` の `binding.issue` | 繰り越し(`gh_available` は変えていない) |
| 432 `Seven test gaps of org-cross-review` | (6) | `ralph org status` の JSON の `feature.issue` | 繰り越し |
| 480 `Two gaps of the feature digest` | (b) | `featureBindingFromTokens` の鍵 | 繰り越し。古い binary の補償で `issue=` が落ちることを書いた |
| 481 `Test gaps of fix/split-feature-digest` | (a)(b)(d) | `featureDigestRefusal`、`featureLeaderTask`、skill の再開の段落 | (d) は `--issue` の文だけ払った(4 写しで固定)。ほかは繰り越し |
| 482 `Five small findings of fix/split-feature-digest` | (a)(b)(c)(e) | `sameFeature` ほか | (a) は払った(下)。(b)(c)(e) は繰り越し |
| 519 `Fourteen small findings of docs/spec-org-reseat` | (d)(k) | `PlanReseat`、`reseatNoActiveSeatErr` | 繰り越し |
| 520 `internal/org/reseat.go (1,307 lines)` | `reseat.go`、`feature.go` | 1,938→1,948 行、899→925 行 | 繰り越し |
| 521 `Test gaps of docs/spec-org-reseat` | (h) | `/org` skill の reseat の小節(4 写し) | 繰り越し |
| 547 `ralph status and ralph org status` | 標準の Trigger | `printStatusTable`、`orgFeatureJSON`(`issue:` の行と鍵) | 繰り越し。`shared` を足すときの形が 2 通りに分かれたことを書いた |
| 548 `internal/org/reserve.go is 822 lines` | 2 つの条件 | `reserve.go` 822→**964** 行(`wc -l`、最後の Go commit 52e6cfd3 で測り直した。self-review の案の 936 は d486f4b5 の値で、verify の 964 と同じ) | 繰り越し。移す関数の組に新しい関数が入ることを書いた |
| 549 `Known gaps of feat/split-shared-paths` | (b)(d) | `featureBindingFromTokens` の `issue` の鍵 | 繰り越し。読めない `issue=` を director の確かめが拾えないことを書いた |

482 の (a): self-review の M1 が三つ目の写しとして挙げた `featureIssueRefusal` を、`52e6cfd3` が `sameFeatureExceptDigestsAndIssue` で払った。行が提案した名前は `sameFeatureExceptDigests` だったが、issue は別に比べるので名前に入れた。first cell と Impact の (a) を `~~...~~` で閉じて、Trigger の注に払ったことを書いた。(e) は払っていない(`sameFeature` と `reservationDecision` の comment を 1 規則 1 文に書き直していない)。

### 2. 計画

self-review L1(Risks の 4 つ目の「director の merge のあとの確かめで拾う」が成り立たない)は計画の本文の文なので、直していない(本文を直すと承認の digest が変わる。前後とも `cbeb26942ff9`)。PR の本文の Known gaps に入れてほしい(下)。同じ内容を、仕様 FR-4 の項と計画の Progress の子の項目にも書いた。Progress は `Test artifact created` の次に `Sync-docs artifact created` の行を足してチェックした。AC のチェックボックスは付けていない(verify V6 のとおり orchestrator が決める)。

### 3. 文書のずれ

| 文書 | 結果 | メモ |
|------|------|------|
| `AGENTS.md` の `internal/org/` の行 | 直さない | `feature.go` の括弧に `--issue` が入っている。merge-order の JSON の鍵は元から列挙していない |
| `README.md` | 直さない | 280 行目の段落に `--issue`、status の行、merge-order の `binding`、reseat が入っている |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 1 文足した | 古い binary の補償で `issue=` が落ちること。90 行目の結びつきの項目の列挙は、日付つきの追記の流儀(96・98 行目)で、上の文は書き換えない |
| `docs/specs/2026-11-12-org-reseat.md` | 直さない | FR-6 の 2026-11-22 の追記が済んでいる |
| `docs/specs/2026-08-01-org-runtime.md` | 直さない | status の行も merge-order の JSON の鍵も列挙していない |
| `/org` と `/director` の skill、`templates/base` の写し | 直さない | `check-sync.sh` は PASS、`check-skill-sync.sh` は 14 skill そろっている。`/director` の確かめには L3 の 3 点(gh の失敗、直後の読み直し、引き継ぎの範囲)が `52e6cfd3` で入っている |
| `internal/org/prompts/leader.md` | 直さない | 変更は手順 14 の 1 hunk だけ。ほかに task の行を列挙する所はない |
| `.claude/rules/ralph/agent-messaging.md`、`model-routing.md`、`docs/quality/quality-gates.md`、`docs/recipes/` | 直さない | `start --plan` を名指すが、`--issue` や task の行の一覧を書いていない |

### 4. 検査

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: 14 skill そろっている
- `go test -count=1 ./internal/org/ -run 'Skill|DocsMatch|Issue'`: 110 PASS、0 FAIL、0 SKIP
- `go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill|Issue|Help'`: 79 PASS、0 FAIL、0 SKIP
- 台帳: 変えた 27 行は、5 セル(`(?<!\\) \| ` で割る)のまま、セルごとのバッククォートと `~~` が偶数、新しい文に `file:line` なし。`git show HEAD:` と比べて、変わった既存行は 27 行だけで、元の Trigger と Related の文はそのまま残っている。mojibake の検査(`check_mojibake.sh`、JSON の入力)は rc 0、U+FFFD なし
- 計画の digest: `cbeb26942ff9`(前後とも)

## 台帳に照合で当てた行と、当てなかった行

handoff と verify の 25 行に加えて、台帳の全行の Trigger のセル(取り消し線の文と日付つきの注を除く)を、diff の Go のファイルと関数の名前(`reserve.go`、`feature.go`、`reseat.go`、`mergeorder.go`、`org_merge_order.go`、`cli/org.go`、`leader.md`、`featureBindingFromTokens`、`sameFeature`、`printStatusTable`、`orgFeatureJSON`、skill の節の名前など)で探した。見つけたのは 311 と 423(`mergeorder.go` の次の変更)の 2 行で、self-review も verify も挙げていなかった。当てなかった候補を、Trigger を読んだうえで理由つきで残す。

| 行(最初の語) | Trigger | 当てない理由 |
|------|------|------|
| 431 `The role prompts grew with org-cross-review` | `leader.md` に節を足す変更 | 手順 14 の 1 hunk の中の文で、節も段落も足していない |
| 426 `internal/org/skill_split_questions_test.go is over 800 lines`、427、274 (b) | このファイルにテストを足す、節の読み手を変える、`director_skill_sync_test.go` にテストを足す | 既存のテストの断片とコメントを直しただけで、テストも読み手も足していない |
| 247、248、249 | stage 8a の director が merge-order を何度も読む | 8a-1 の注のとおり director は報告のたびに merge-order を読む。この PR の `--all --json` の読みは merge の報告と引き継ぎ 1 回ずつで、ループではない。報告に基づく Trigger(遅い、`unreadable`、別の host の PR)は報告がない |
| 259 | `mergeorder.go` の `MergeOrder` の変更 | `MergeOrder` 関数は変えていない(構造体と `collectMergeTargets` を変えた) |
| 430 | `featureLeaderTask` の `[reviewer]` の 2 行の変更 | その 2 行は変えていない。issue の行は別の所 |
| 150、191、193、201、205、222、241、267、299、309、317、318、385、502、513 | 名指す関数・節(`resolveOrgConfig`、`spawnCapacityErr`、`idempotentRespawn`、`otherRecordFix`、受信箱の節、`gateToolsProbe` など) | その関数も節も変えていない |
| 202 (i)(k)、203 (b)(c)(j)(k) | `maxFeatureOrgIDLen`、`split.go` の comment、`ensureFailureErr` など | 変えていない |
| 251、254、470、512 | 閉じた行 | 取り消し線の行 |

## handoff と実際の違い

- 548 の行数は 964(依頼は「936 ではなく測り直す」、verify も 964)。`wc -l` を最後の Go commit 52e6cfd3 で測り、origin/main ccdf6293 の 822 と並べた
- self-review の「Tech debt identified」の 2 行のうち、1 行目(L3: gh の失敗、merge 直後の遅れ、引き継ぎの範囲)は `52e6cfd3` が director skill に入れたので、行にしていない(skill に 3 点とも書いてあり、`skill_issue_test.go` が固定する)。2 行目(L2)は読み位置の半分が `52e6cfd3` で直ったので、残りの半分(`- issue:` の行と計画の `Related issue:` が別の番号のとき。verify V5)だけを新しい行にした(555 行目)
- self-review L4(`featureLeaderTask` の文字列引数が 5 つ並ぶ)は、self-review が「今回は許容」とし、台帳に回していないので、行にしていない。要るなら言ってほしい
- verify V4(#364 は `binding.issue` を数で比べる)は台帳ではなく #364 の実装に回す項目なので、行にしていない
- verify の Trigger の候補を出す script の提案は、台帳に既存の行がなかったので新しい行にした(557 行目)。落としてよい

## 新しい台帳の行

| 行 | 内容 |
|------|------|
| 555 | 手順 14 が、task の `- issue:` の行と計画の `Related issue:` が別の番号のときにどちらを閉じるかを言わない(self-review L2 の後半、verify V5) |
| 556 | この PR のテストの穴: (1) gate と regate の `sameFeature` の比べで issue だけが変わる場合を固定するテストがない(変異 M1 が 4 つのテストで残る)、(2) 本物の GitHub で `Closes` が issue を閉じること、director の確かめを本物の gh で回したことがない、(3) 本物の herdr での `start --plan --issue` と古い binary を混ぜた台帳 |
| 557 | Trigger の照合を手で行い、各段が前の段の見落としを拾う(この PR は 7 行、18 行、2 行)。候補を出す script の提案 |

## PR の本文の Known gaps に入れるもの

1. 計画の Risks の 4 つ目(古い binary が結びつきを書き戻して `issue=` が落ちる)が言う「director の merge のあとの確かめで拾う」は成り立たない。落ちた記録は `binding.issue` も落ちるので、確かめるものがない(self-review L1)。版が混ざる期間に限る。計画の本文は digest が変わるので直さず、仕様 FR-4 の項に書いた
2. gate と regate の `sameFeature` の比べで issue だけが変わる場合を固定するテストがない(test の Test gaps 1、verify V3、台帳 556 の (1))。古い binary が書き戻した記録のとき、gate は「it is now bound to ...」と言って記録せず、打ち直せば通る(止まる側に倒れる)
3. 本物の GitHub で `Closes` が issue を閉じること、本物の herdr での `start --plan --issue`、古い binary と混ぜた台帳は未確認(台帳 556 の (2)(3))
4. 手順 14 は、`- issue:` の行と計画の `Related issue:` が別の番号のときを言わない(台帳 555)

## Insight event

`./scripts/insights-append.sh --slug org-start-issue --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行(cycle 1、pass)を足した。rc 0、`git status --short` は既存の jsonl の `M` だけ。
