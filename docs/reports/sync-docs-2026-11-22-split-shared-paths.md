# sync-docs report: split-shared-paths

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(承認の digest `9046d5c8297e`。この run の前後とも同じ)。issue #363
- Branch: `feat/split-shared-paths`。run 開始時の HEAD は `84bdf572`(test report の commit)。base は main の `9cd63d60`。origin/main はその後 `10e9322d`(PR #368)まで進んでいる
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `4a4be9de`(Merge: yes、LOW 8 件。直しは `143deb8f`)、verify `77a6961a`(pass。V1〜V5 は `9535c7a7` で直し済み、V6〜V8 とセクション「直していないもの」の INFO が /sync-docs へ)、test `84bdf572`(pass、`run-test.sh` rc 0、変異 11 個がすべて落ちた)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff 9cd63d60...HEAD`
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go のファイルに触れたのは comment 7 か所だけで、行数は変わらない(`reserve.go` は前後とも 822 行)。計画の本文は Progress と AC の箱以外を変えておらず、`./scripts/plan-visual.sh digest` は編集の前後とも `9046d5c8297e`
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff 9cd63d60...HEAD` は 42 ファイル、+3,233/-339。Go は全体で +2,113/-164、テスト以外の Go が +418/-143(`internal/org/prompts/leader.md` の +8 は含まない)。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify が /sync-docs に回した V6・V7・V8 と INFO 3 件を片づけた。V6 と V7 は org skill の 2 か所(各 4 写し)、V8 は tech-debt の引き継ぎの注。注は、verify が挙げた 12 行と判断を求められた 1 行のほかに、自分の sweep で 4 行(88、202、519、521)が見つかり、17 行に足した。新しい行は 3 つ。そのほか、Go の comment 7 か所が計画を `docs/plans/active/...` の path で引いていて、`/pr` が計画を archive へ動かすと切れるので、slug の引用に直した(動きは変えていない)。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `.claude/skills/org/SKILL.md` と、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | V6: コマンド表の `spawn` の行の `--reserve` の括弧に「走っている機能の org の共有のパスと重なっても拒否」を足した(行は 2,998 → 3,026 文字)。V7: 分割計画の例の座席の決まりを「予約と共有のパスの外のファイルは編集しない」にした | `5cdc7150` |
| `docs/tech-debt/README.md` | 17 行の Trigger 末尾に 2026-11-22 の引き継ぎの注、Related 末尾に `docs/plans/active/2026-11-22-split-shared-paths.md` を足した。新しい行 3 つを末尾に足した(547〜549 行目)。既存の行の Debt、Impact、Why deferred の列は変えていない | `5cdc7150` |
| `internal/org/reserve.go`、`feature_shared_task_test.go`、`feature_shared_test.go`、`prompts_leader_shared_test.go`、`reseat_shared_test.go`、`reserve_shared_test.go`、`watch_scope_shared_test.go` | comment の `docs/plans/active/2026-11-22-split-shared-paths.md` を `2026-11-22-split-shared-paths` にした(各 1 か所、行数は同じ)。`gofmt -l` は空、`go vet ./internal/org/` は出力なし | `5cdc7150` |
| `docs/plans/active/2026-11-22-split-shared-paths.md` | AC1〜AC8 の箱を `[x]` に、`Review artifact created` を `[x]` にし、Progress に sync-docs の 1 行と、その下に release ノートの候補と `/pr` の前の注意を足した | この report の commit |
| `docs/reports/sync-docs-2026-11-22-split-shared-paths.md` | この report | この report の commit |
| `docs/insights/events/2026-11-21-split-shared-paths.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行(UTC の日付が 11-21 のままなので、同じ日付のファイル) | この report の commit |

## 依頼の項目

### V6: org skill のコマンド表の `spawn` の行

`--reserve` の括弧が「走っている他の org の予約と重なれば拒否」のままだった。正本の節「担当範囲の予約」は直っていたので、表の 1 句だけ足した。4 写しは `cmp` で同じ。この文を固定するテストはなく、`skill_shared_paths_test.go` の期待も変わらない。

### V7: 分割計画の例の前文と RC4 の行

例のヘッダに `- Shared: go.mod, go.sum` を宣言しながら、前文の座席の決まりが「予約の外のファイルは編集しない」で、字面どおりには宣言した共有のパスを座席に編集させなかった。「予約と共有のパスの外のファイルは編集しない」にした。この行を読むテストは `TestSkills_SharedLineExamplesParse` だけで、例の読み込みには影響しない(下の確認で通った)。

RC4 の行(「RC4 (org recheck 2026-10-24)」)は、前文の同じ規則が leader の雛形の手順と食い違うことを挙げている。直したのは共有のパスの分だけで、行が挙げる食い違いの本体(機能の計画、reviewer の report、org の report を予約の外に書かせること)は残る。そのため行は閉じず、Trigger に「共有のパスだけ名指す変更で、(1) の本体と (2) の step 4 は払っていない」と注を足した。

### V8: tech-debt の Trigger

次の「Trigger を引いた行」の節に、17 行と、見たが引かなかった候補を書く。注の形は 2026-10-26 以降の行(RC4、`resolvedOrClean`)に合わせた(`<元の Trigger>. 2026-11-22: <branch> hit this trigger with <編集>, but did not pay it: <理由>. Carried over to the next change that meets this trigger`、Related に計画を `docs/plans/active/...` で足す。`archive-plan.sh` が `/pr` で `archive/` に書き換える)。

### INFO 3 件

| INFO | 扱い |
|------|------|
| `reserve.go` が 822 行(base 686) | tech-debt に新しい行を足した(548 行目)。`FeatureBinding` 一式を `feature_binding.go` へ、パスの規則を `reserve_paths.go` へ、という move-only の split を Trigger にした。この大きさを挙げる既存の行はなかった。verify の「821 行」は `9535c7a7` の comment 1 行を足す前の数 |
| 既存の分割計画がヘッダより後の `- Shared:` で断られる | 計画の Progress に release ノートの候補として書いた(Rollout の節は revert の向きだけを書いているので、そこは変えていない) |
| `ralph status` と `ralph org status` が共有のパスを出さない | tech-debt に新しい行を足した(547 行目)。既存の行「`ralph org status` shows neither the approval mode ...」の Trigger は「stage 7b、または approval の状態を manifest で読んだ人」で、共有のパスは当たらないので、新しい行にして Related から指した。self-review が書いた行の案を英語にして使った |

test report の Test gaps 2〜4 と self-review の LOW 6 は、新しい行(549 行目、「Known gaps of feat/split-shared-paths (#363)」)の (a)〜(d) にした。Test gap 1 は既存の「Two gaps of the feature digest of a restarted feature org」の (b) と同じなので、その行の注に `shared=` の部分を足した。Test gap 5(main を取り込んだら run-test を回し直す)は `/pr` の前の注意で、計画の Progress に書いた。

## Trigger を引いた行(17 行)

行は最初の語で示す(行番号は動く)。どれも払っていない。521 の (h) だけ、書いた文の 1 つをテストで固定している(ただし文字の固定で、help やコードとの突き合わせではない)。

| 行 | 引いた Trigger | 編集 | 払わなかった理由 |
|----|---------------|------|------------------|
| 「6 batchable LOW findings from org-debt-batch」 | (3) `watch.go` を触る | `evaluateSeat` の呼び出しと comment 2 か所。2,246 → 2,249 行 | (3) の `SEAT: ` header は変えていない。sweep で見つけた(verify の一覧にない) |
| 「Two documentation findings of org-stop-all」 | (a) コマンド表の編集 | `watch` の行(3,378 → 3,410 文字、S4 の `eb43b1b7`)と `spawn` の行(この pass、2,998 → 3,026 文字) | `stop` と `disband` の 1,208 と 1,219 文字は変わらず、4 写しの表の組み替えは範囲外 |
| 「What a leader is told about its reservation and the org-wide limits」 | (a) leader の prompt | `leader.md` 1,626 → 1,634 行(段落 1 つ) | `RolePromptVars` と `{{SCOPE}}` の文言は同じ |
| 「Small findings of org-feature-worktree; none changes behavior」 | (g)、(f)、(j)(struck 済み)と (k) | `leader.md` の機能の org の段落。`split.go` の header comment と `parseSplitPlan` の doc | (g)(f)(j) は直し済みで払うものなし。(k) は後述。sweep で見つけた |
| 「Test gaps left by org-feature-worktree (the test report's T-1 to T-7」 | (e)、(f) | `StartFeature` の doc、`startFeatureLeaderParams`、`featureLeaderTask`。`feature.go` 874 → 898 行 | 動かない枝の測り直しは範囲外 |
| 「Small leftovers of org-plan-approval」 | (c) | `featureBindingFromTokens` の鍵 `shared` | 補償で読めない binding を書き戻すテストは足していない |
| 「`internal/org/verbs.go` is 2277 lines」 | leader の雛形のテスト | `prompts_leader_shared_test.go`(新しい、64 行)。名指しの 5 ファイルは未変更 | move-only の split は範囲外 |
| 「RC4 (org recheck 2026-10-24)」 | skill の例と leader の雛形 | 例の規則を共有のパスの分だけ直し(V7)、`leader.md` に段落 | 行が挙げる食い違いの本体と step 4 は払っていない |
| 「`resolvedOrClean` in `internal/org/statedir.go`」 | 呼び手の編集 | `feature.go`、`split.go`(呼び出しは未変更) | 呼び出しの置き換えは範囲外 |
| 「The role prompts grew with org-cross-review」 | 節を足す変更(判断) | 見出しなしの段落 1 つ(+8 行) | 2026-11-16 の注が「機能ごとの org」の前書きへの段落を数えているので、それに合わせて数えた。見出しは足していない |
| 「Eleven gaps that fix/watch-alert-dedupe」 | `scope_change` のコード、(5) | `watch.go`、`watch_scope.go`、`evaluateReservedScope` の doc | (5) の文は足していない。(6) の関数は未変更 |
| 「Two gaps of the feature digest of a restarted feature org」 | (b) | `featureBindingFromTokens` の鍵。古い binary が書き戻すと `shared=` も落ちる、を (b) の確かめに足した | 古い binary での実行はしていない |
| 「Test gaps of fix/split-feature-digest」 | (a)、(b)、(d) | `featureDigestRefusal`、`featureLeaderTask` のタスク、skill の立て直しの段落 | 表の行は未変更(新しい枝は `TestStartFeature_SharedChangedRefused` が走らせる)。(d) の段落は 4 写しで固定していない |
| 「Five small findings of fix/split-feature-digest」 | (a)(e)、(b)(c) | `sameFeature` の doc、`featureDigestRefusal`、`reservationDecision`(`reservationOverlapErr` へ)、`validateFeatureBinding`、`String()` | 範囲外。`featureDigestPattern` は `split_test.go:790` のまま |
| 「Fourteen small findings of docs/spec-org-reseat」 | (d) | `PlanReseat` の `scope` の 1 文と comment | 先頭の index と 2 つの組み立ての分割は範囲外。sweep で見つけた |
| 「`internal/org/reseat.go` (1,307 lines)」 | `reseat.go`、`reseat_test.go` | 1,933 → 1,938、2,004 → 2,005 行。テストは新しい `reseat_shared_test.go`(141 行) | `PlanReseat` の move は範囲外 |
| 「Test gaps of docs/spec-org-reseat (#312c, the first PR)」 | (h) | org skill の reseat の節の step 4(4 写し) | step 4 の文は `skill_shared_paths_test.go` が読むが、help との突き合わせではない。sweep で見つけた |

17 行の内訳は、verify の Coverage gaps が挙げた 12 行、判断を求められた 1 行(「The role prompts grew with org-cross-review」)、自分の sweep で見つけた 4 行(「6 batchable LOW findings」「Small findings of org-feature-worktree」「Fourteen small findings of docs/spec-org-reseat」「Test gaps of docs/spec-org-reseat (#312c, the first PR)」)。

### 見たが引かなかった候補(理由)

| 行 | 候補の Trigger | 引かなかった理由 |
|----|---------------|-----------------|
| 「A `ralph` of another version on the pane's `PATH`」 | `newOrgStartCmd` の `RunE` | 変えたのは `Long` と `orgReserveFlagUsage` の文だけで、`RunE` は未変更 |
| 「Test gaps left by the org-limits-reserve cycle 2 fix」 | (c) `ralph org status` の `Long`、`--config` の説明 | `ralph org start` の `Long` を変えた。`ralph org status` の `Long` は未変更 |
| 「Seven test gaps of org-cross-review」 | (6) `ralph org status` の JSON | `orgFeatureJSON` の doc comment だけで、JSON の形は変わらない(境界例。comment だけの編集を数える行の流儀なら引く) |
| 「`internal/org/watch.go` is 1848 lines」 | 見張りの escalation とそのテスト | 変えたのは scope_change の評価で、`escalateToInbox` などと `watch_test.go` は未変更。2026-11-12 の注も `watch_scope.go` だけの修正は数えていない |
| 「A seat in the set of the org condition `usage_limited`」 | `evaluateSeat` の画面の読み | `evaluateSeat` の予約の評価だけで、画面の読みは未変更 |
| 「After step 14 of the leader template」 | step 14 と「org の終わり方」の 6 | 変えたのは「org の終わり方」の 1 で、`leader.md` の step 14 は未変更 |
| 「Three test gaps of the org-director-attach PR」ほか(director skill の行 385、386、426、427、477) | `受信箱の処理`、send の確かめ、`skill_split_questions_test.go` など | director skill で変えたのは「分割計画の前の質問」「分割計画の承認」「機能ごとの出力先」「起票と記録」「共有のパス」で、名指しの節は未変更 |
| 「Small leftovers of standard-config-gaps」 | 4 つのテストの `commitRalphTomls` | 4 つとも未変更(`org_feature_test.go` は help のテストの comment と 1 行) |
| 「`ralph org status` shows neither the approval mode」 | stage 7b、または approval の状態を manifest で読んだ人 | 当たらない。Related から新しい行を指す |
| 「Nothing in ralph checks that a feature org's leader follows its split plan's preamble」 | `## Features` より後の規則 | 断ったのは `- Shared:` の行だけで、`## ` の節への warn は足していない |
| 「Small leftovers of the org-merge-order PR」ほか(252、259、473) | README の文 | Trigger の中の README は別の文(merge-order、merge guarantee、scaffold version)で、今回の README の編集は分割計画と見張りの段落 |

## 通常の確認

| 場所 | 確かめたこと | 結果 |
|------|-------------|------|
| `.claude/skills/org/SKILL.md`、4 写し | `cmp` で 4 写しが同じ。`./scripts/check-skill-sync.sh` は 14 skill が lock-step。`./scripts/check-sync.sh` は `PASS: all files in sync.` | 通った |
| skill の文を固定するテスト | `go test ./internal/org/ -run 'Skill\|DocsMatch\|Director\|Split\|SharedPaths\|SharedLine'` と `go test ./internal/cli/ -run 'Agents\|Readme\|Docs\|Skill'` | どちらも `ok`(105.4 s と 1.9 s) |
| Go の comment の編集 | `gofmt -l internal/org` は空、`go vet ./internal/org/` は出力なし、`git grep 'plans/active/2026-11-22-split-shared-paths' -- '*.go'` は 0 件 | 通った |
| README、AGENTS.md、2 つの仕様、`worktrees.md`、`ralph org start --help` | verify が実装と合うとした。今回の変更(skill の 2 か所と comment)は、これらの文に触れない | 変更なし |
| `docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`、`.claude/rules/` | `scope_change`、`Shared`、`共有` で `git grep` した。当たりは `quality-gates.md` の別の意味の "Shared or ambiguous changes" だけ。新しいファイルはテストだけで、repo-map の対象ではない | 変更なし |
| branch の secret scan | `./scripts/secret-scan-branch.sh --strict` は `scanned 9cd63d60..5cdc7150 against origin/main: clean` | 通った |
| tech-debt の編集 | `git show HEAD:` と行ごとに突き合わせた(`checkrows.py`、scratch)。変わった行は 17 行と末尾の 3 行だけで、5 つの列と `~~` の数が偶数のまま、元の Trigger の列が接頭辞として残る | 通った |

## Left open

- 「Small findings of org-feature-worktree」の (k): この PR は `split.go` の header comment と `parseSplitPlan` の doc を直した。(k) が求める 3 つの言い方(feature の中では本文、header では `- Status:` と `- Approved:` が効く、header より後では断る)に、書いた文が重なる。計画の Scope は `- Shared:` なので払ったとは書かず、注に「次の編集で (k) と突き合わせて、3 つとも言っていれば strike する」と書いた。strike するかどうかは lead の判断で、しなくてよい
- 「The role prompts grew with org-cross-review」: 段落の追加を数えるかは判断。2026-11-16 の前例に合わせて数えた。数えないなら注を削れる
- 「Seven test gaps of org-cross-review」(6): 引かなかったが境界例(上の表)
- `/pr` の前: main を取り込む。`git merge-tree` の予想で衝突するのは `docs/tech-debt/README.md` だけ(末尾に行の追記が両側にある)。取り込んだら `check-sync.sh`、`check-skill-sync.sh`、`./scripts/run-test.sh` を回し直す(`feature.go` は別のレーンも触る。test report の Test gap 5)
- release ノートの候補は計画の Progress に書いた(既存の分割計画のヘッダより後の `- Shared:` が読み込みで断られる)

## Not verified

- `./scripts/run-test.sh` と `./scripts/run-static-verify.sh` は回していない。回したのは上の表のものだけ。この run が変えたのは skill の 2 か所、tech-debt、計画の Progress と箱、Go の comment 7 か所で、動きは変えていない
- 新しい tech-debt の行の主張のうち、「Known gaps」の (a)(c)(d) は test report と self-review の記述に拠り、再確認していない(各 gap の元は Related に書いた)。(b) の `String()` と `approvalModeUnreadable` の対比は `reserve.go` を読んで確かめた

## Insight event

`./scripts/insights-append.sh --slug split-shared-paths --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行を足した(`cycle` 1、pass)。このレポートと同じコミットに入れた。
