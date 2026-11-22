# sync-docs report: org-result-missing

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)。承認の digest `698af1d189e8` は、この run の前後とも同じ(`./scripts/plan-visual.sh digest` で確かめた)
- Branch: `fix/org-result-missing`。run 開始時の HEAD は `e976f232`(push 済み、clean)。base は origin/main `2a069b50`
- Pipeline run: cycle 1(`.harness/state/standard-pipeline/` が worktree にないので `--cycle auto` は 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review Run 1 `98392dd5` と Run 2 `92ea795c`(Merge: yes。直しは `f57cb92d` と `b0a9b28a`)、verify `46c5040b`(HEAD `b0a9b28a` で回した。pass、MEDIUM 1、LOW 5、INFO 5)、test `6a7b39ab`(HEAD `46c5040b` で回した。pass、`run-test.sh` rc 0)。test の Test gaps 1、2、4 は、そのあとの test だけの commit `e976f232` で閉じた(下の「e976f232 の追記」)。Test gaps 3 と 5 は残る
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。回したのは下の「確かめたこと」の narrow の `go test` と静的な確認だけ。一時ファイルは scratchpad に置いた。commit は 1 つで、push はしていない
- この commit は Go のファイルに触れる(`internal/cli/org_watch.go` の `--help` の文字列、`internal/org/prompts/leader.md`(embed される雛形)、テスト 3 本)。どれも文と pin の追加で、判定の論理は変えていない。記録だけの修正の例外(`post-implementation-pipeline.md`)には当たらないので、`/self-review` から回し直すかどうかは lead が決める
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff --shortstat origin/main` は 40 ファイル、+4,257/-152。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify が渡した 13 件と候補 2 行の判断、依頼の追加分(rule の失敗の道の 1 文、テスト 3 本の pin、テストファイルの大きさの行)をすべて片づけた。直した場所は、台帳 13 行と新しい 1 行、README、quality-gates(2 写し)、rule(2 写し)、`/org` と `/director` の skill(4 写しずつ)、仕様、recipe(2 写し)、`ralph org watch --help`、leader の雛形。

中心の食い違いは verify の V1 だった。`f57cb92d` で「`result_missing` は TASK ごとに数える。別の TASK の立ち上がりは新しい ALERT と件になる」に変わったのに、再発の一般の決まりを書く場所が例外を書いていなかった。`--help`、quality-gates、README、`/org` と `/director` の skill、仕様の (b) の 6 種類に、例外を 1 文ずつ足した。`--help` と skill には pin のテストも足した。

台帳は、verify の掃き出しに加えて、この run の自分の編集で Trigger に当たる行が 4 つあった(267、376、118、517)。verify の一覧にも、lead の依頼にもない行なので、下に分けて書く。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/tech-debt/README.md` | 既存 13 行(88、118、182、267、362、370、376、377、389、431、467、517、563)に注か直し。注を足した行の多くは、Related にも計画か report を足した。新しい 1 行(`watch_result_missing_test.go` の大きさ)。5 列、backtick の数が偶数、`file:line` の参照を足していないこと、`docs/plans/` の参照が存在することは `git show HEAD:` と比べて確かめた | この report の commit |
| `README.md` | `watch --all` の段落の再発の文に `result_missing` の例外、`RESULT` を `RESULT` or `REVIEW` に | この report の commit |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | Watchdog pulse layer の行: 条件の並びに `result_missing`、再発の例外、stall の保留の 1 句。2 写しは同じ文 | この report の commit |
| `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md` | `result_missing` の段落に、台帳か受信箱が読めないときと記録に失敗したときは、同じ TASK でも新しい ALERT と 2 つ目の件になる、の 1 文(V4)。2 写しは `cmp` で同じ | この report の commit |
| `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と、`.agents` と `templates/base` の写し(計 8 ファイル) | `/org`: `watch` の行に例外の句、「続いて立つ ALERT は件を増やさない」の項に例外の 1 文、`result_missing` の項に stall の保留の範囲(遡れなかった TASK は除く)と TASK ごとの再発と失敗の道。`/director`: `recurred` の段落に例外。`.claude` と `templates/base/.claude` は同じ Edit を 2 回、`.agents` は `sync-skills.sh` で作った。`check-skill-sync.sh` は 14 skill で通る | この report の commit |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-22 改訂の (b) に状態ファイルの 5 つの欄、TASK ごとの再発、状態ファイルを失ったときの探し直し、失敗の道を足した。(c) の保留の条件に「遡れなかった TASK は除く」 | この report の commit |
| `docs/recipes/isolated-herdr-server.md`、`templates/base/docs/recipes/isolated-herdr-server.md` | 「Optional: a private agmsg」の 3 か所(S1 で座席の雛形が写しのパスを渡すようになったこと、実地では確かめていないこと、確かめるなら 1 つの TASK から)。2 写しは `cmp` で同じ | この report の commit |
| `internal/cli/org_watch.go` | `ralph org watch --help` に、条件の並びへの `result_missing` と、`usage_limited` の段落のあとの 1 段落(stall の保留、TASK ごとの再発)。既存の pin の文は 1 字も変えていない(後ろに足した) | この report の commit |
| `internal/cli/org_watch_result_missing_test.go`(新規) | 上の help を読む pin 1 本。`org_watch_test.go` は 1,967 行で、足さずに新しいファイルにした | この report の commit |
| `internal/org/prompts/leader.md` | `result_missing` の項の「RESULT」を「RESULT(か REVIEW)」に。3 行を折り直しただけで、行数は 1,689 のまま | この report の commit |
| `internal/org/prompts_leader_result_missing_test.go` | leader の pin の断片を新しい言い方に合わせた(意図は弱めていない)。rule、`/org` の skill、`/director` の skill に足した文の pin を足した | この report の commit |
| `docs/plans/active/2026-11-22-org-result-missing.md` | Progress に Sync-docs の行を足し、AC1 から AC6 に `[x]`。本文は変えていない。digest は前後とも `698af1d189e8` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-org-result-missing.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-org-result-missing.jsonl` | `sync_docs` の行(cycle 1、pass)。UTC の日付は 11-22 で、既存の 4 行と同じファイル | この report の commit |

## verify の一覧(1 から 13 と、14 の候補の判断)

1. 台帳 467(確かめた): 注の「`escalateToInbox`、`recordRecurrence`、`watch_recurrence.go` は触れていない」は S2 と S2b の時点の話で、`f57cb92d` が `escalateToInbox` を、`b0a9b28a` が `recordRecurrence` の doc と `watch_recurrence.go` の冒頭を変えた(`git log` で、`watch_recurrence.go` を変えた commit は `b0a9b28a` だけ)。同じ cell に日付つきの注を足し、行の Trigger と (9) と (10)(#312a が `escalateToInbox` の同じ枝で当てた前例どおり)が当たり、払っていないことを書いた。(7) の直しの対象と Debt の文に `recordResultMissingRecurrence` を足した。行数は `watch.go` 2,249 → 2,328、`watch_recurrence.go` 277 → 288(`wc -l`、origin/main `2a069b50` と `b0a9b28a`)
2. 台帳 88(確かめた): この branch の注を直した(新しく足した注ではなく、同じ文を書き換えた)。欄は `IdleSince`、`ResultSeen` に `ResultOutOfReach`、`ResultMissingTask`、`ResultMissingAlert` を足し、行数は 2,249 → 2,328(`b0a9b28a` は watch.go を変えた最後の commit)
3. 台帳 362(新しく見つけた): Trigger に `f57cb92d` の `escalateToInbox` が当たる。注を足し、Related に計画を足した。`watch_result_missing_test.go` は 1,672 行で 800 行の目安を超えるのに、台帳に行数の記録がなかったので、新しい行を末尾に足した(`watch_result_missing.go` は 675 行で目安の内)。新しい行は 31 のテスト関数と 2 つの fixture を 4 つの塊(時計、履歴の読みと遡り、stall の保留、TASK ごとの再発)に分けて挙げ、Trigger は塊ごとにファイルを分けることにした
4. 台帳 563: (a) の Impact に、別の TASK の件が増えて 1 つの座席に TASK_ID のない件が並びうること、前の件の TASK は `ALERT_TS` の前にその座席へ送られた最後の TASK として台帳で探すこと。(b) に `b0a9b28a` の注(`raiseResultMissing` が状態を失ったときの探し直しで変わり Trigger に当たり、払っていない。探し直しで見つけた ALERT への再発は、watcher を起こさないまま)。(c) として、台帳か受信箱が読めない、`recurred` の追記に失敗した、で 2 つ目の件になる道を足し、状態を失ったときは `resultMissingAlertOfTask` で閉じたことを書いた。`recordResultMissingRecurrence` の読みで確かめて、stderr に 1 行出るのは台帳の読みの失敗と追記の失敗で、受信箱の読みの失敗は何も出さない。test の Test gaps 3 は (c) の Trigger に、Test gaps 5 は (d)(実機の確かめ)として足した。(a) の Trigger に `/org` の skill と仕様の (b) を足した(どちらも「件の本文に TASK_ID は載らない」を言う)。Related に self-review、verify、test の report を足した
5. `README.md`(確かめた): V1 と V7 の両方を直した
6. 台帳 182(新しく見つけた): `/org` の skill の `watch` の行は、origin/main で 3,410 文字、この run の前で 3,609、後で 3,673(python で行全体を数えた。4 写しとも同じ)。`stop` と `disband` は 1,208 と 1,219 のまま。注を足し、Related に計画を足した。skill の編集のあとで数え直したので、数は 3,673
7. 台帳 377(新しく見つけた): #360 の注の「`reviewer.md` still has no `{{AGMSG_SCRIPTS}}`」は main の上では正しく、この branch では誤りだったので、「on main then, …」と言い換えた。取り消し線の中の「see the 2026-11-22 note at the end of this cell」も「of fix/org-result-missing in this cell」にした(#360 の注が末尾に来たため)。この branch の注の「Still open: … then the recipe bullet」は、recipe を直したので書き換えた(残るのは私用の写しでの実地の run だけ)。Impact の末尾にも 1 句足した
8. leader の雛形の REVIEW(確かめた): 「その TASK の RESULT(か REVIEW)を送っていない」にした。テストの断片も同じ言い方にした。`assertPhrasesInOrder` は空白をすべて落として比べるので、折り直しは pin に効かない
9. 仕様の (c) と `/org` の stall の文(確かめた): (c) は「遡れなかった TASK を除く」、`/org` の `result_missing` の項は rule と同じ言い方にした。V1 の仕様の (b) の「立ち直しは `recurred`」も直した
10. recipe(確かめた): 3 か所を直した。S1 で雛形が写しのパスを渡すようになったこと、本物の run では確かめていないこと、確かめるときは 1 つの TASK から始めて leader の履歴で座席の RESULT を見ること、座席が雛形の手順でなく自分の agmsg の skill で送れば自分の agmsg に書かれること。2 写しは `cmp` で同じ
11. 同じ TASK の 2 つ目の件(確かめた): 状態を失った道は `b0a9b28a` で閉じている。読み書きの失敗の道は残るので、rule に 1 文(V4)、台帳 563 の (c)、`/org` の skill、仕様、`--help` の括弧に書いた。rule の文は pin した
12. `--help` と quality-gates(新しく見つけた): 条件の並びに `result_missing`、stall の保留、TASK ごとの再発を足した。`TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope` の pin の文は変えず、新しい段落を `usage_limited` の段落の後ろに置いた。新しい pin は新しいファイルのテスト 1 本。quality-gates は 2 写しとも同じ 3 か所
13. `/org` と `/director` の skill の再発の段落(新しく見つけた): `/org` は `watch` の行、「続いて立つ ALERT は件を増やさない」の項、`result_missing` の項。`/director` は `recurred` の段落(director が同じ座席の件を 2 つ見たとき、座席の画面と leader の ALERT で TASK を見分ける、と書いた)。4 写しずつ
14. 候補の 431 と 389: どちらも注を足した。431 は、見出しを足していなくても、`reviewer.md` と `implementer.md` の typed protocol の節に手順を、`leader.md` の運用規律に項を足した変更で、#312c-2 と #363 が同じ種類の追記に注を残した前例に合わせた。389 の Trigger は「止める規則(使命の 6 とディシプリンの節)」で、運用規律に足した項は「この ALERT を理由に座席を stop も spawn し直しもしない」と言う。止める規則そのものは変わっていないので、置き場所で当たったと書いて、払っていないことを明記した

## 掃き出しで見つけた行(verify の一覧にも依頼にもない)

私の編集で Trigger に当たった行を、Trigger の cell(打ち消し線を除く)を python で読んで探した。ファイル名と関数名の一致で拾ったので、名前を出さずに書かれた Trigger は拾えていない。

- 267: 「`/director` の skill の inbox section(`受信箱の処理`)の次の変更」。S3 の `### 件の処理` の編集と、この run の `recurred` の段落の編集の両方が当たる。注を足し、Related に計画を足した。`director_skill_sync_test.go` には触れておらず、`WaitInbox` と期限の道もそのままなので (1) から (3) は開いたまま
- 376: 「この 5 つの recipe のどれかの次の編集」。`isolated-herdr-server.md` を 2 写しとも編集した。`docs/evidence/` の引用は足しておらず、`docs/tech-debt/README.md` の引用は元からある 1 つを書き換えただけ。注を足し、Related に verify の V6 を足した
- 118: テンプレート側 `quality-gates.md` への次のタッチ。Watchdog の行を編集した。「Must pass in CI」の節にも verify の連鎖にも触れていない。注を足し、Related に verify の V2 を足した
- 517: 「`newOrgWatchCmd` の段落」。`usage_limited` の段落は変えず、条件の並びと、その後ろに足した段落を変えた。境目で当たったとして注を足した。足した pin は (d)(`usage_limited` の段落の語を読む)ではない
- 370: S3 の注の続きとして、leader の雛形のテストを再び編集したこと(253 → 275 行、`leader.md` は 1,689 行)を足した。`org_watch_test.go` は 1,967 行のままで、新しいテストを別のファイルに置いたので、サイズの行には当たらない

当たらなかった行: 202(`leader.md` の「機能ごとの org」の節。編集した運用規律の項はそこにない)、495(`usageLimitRecurNote` などのコードに触れていない)、360(`/org` の skill の「DECISION を送る」の項に触れていない)、385、386、512、513(`/director` の skill の別の節)。431 と 389 は、上の 14 番のとおり注を足した。

## e976f232 の追記

`e976f232`(`test: pin the reach-back boundary and a rise after the leader acted`)は test だけの commit で、`internal/org/watch_result_missing_test.go` に +92 行。test report の Test gaps 3 つを閉じた。

- Test gaps 1(遡れたとみなす境目、変異 Rc): `TestWatch_ResultMissing_AReadWhoseOldestLineIsInTheTasksSecondDoesNotReachBack`
- Test gaps 2(leader が応じた保留中の ALERT のあとの再発、変異 O6): `TestWatch_ResultMissing_ARiseAfterTheLeaderActedOnThePendingAlertIsANewAlert`
- Test gaps 4(時刻が読めない `sent` を飛ばす分岐): `TestResultMissing_FirstSentAfter`

implementer の報告では、`go test -count=1 ./internal/org/ -run 'ResultMissing|Watch'` が ok、変異 4 つがどれもテストの assert で落ちた。この run でも同じ絞り込みを回して ok だった(下の「確かめたこと」)。変異は回し直していない。この commit のあとの `./scripts/run-test.sh` は回していない。test report の Verdict(HEAD `46c5040b` で rc 0)は、この commit を含まない木の結果で、test だけの commit なので `internal/org` の本体のコードは `46c5040b` と同じ。test report の Test gaps 3 と 5 は台帳 563 の (c) と (d) に入れた。test report 自体は書き換えていない。

## 確かめたこと

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 skill で lock-step |
| `./scripts/check-template-purity.sh`、`./scripts/check-pipeline-sync.sh` | 通った |
| `test -z "$(gofmt -l internal)"`、`go vet ./internal/cli/ ./internal/org/` | どちらも通った |
| `git diff --check` | 空白の問題なし |
| `go test -count=1 ./internal/cli/ -run 'OrgWatch\|Help\|Agents\|Readme\|Docs\|Skill'` | ok(20.0 s) |
| `go test -count=1 ./internal/org/ -run 'Skill\|DocsMatch\|AgentMessaging\|RenderRolePrompt_Leader\|ResultMissing'` | ok(52.3 s) |
| `go test -count=1 ./internal/org/ -run 'ResultMissing\|Watch'` | ok(36.1 s) |
| 台帳の `docs/plans/` の参照 | `verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep で、存在しない参照なし |
| 変異 1 つ | `.claude/skills/org/SKILL.md` の「TASK ごとに数える」を「TASK ごとには数えない」に変えると、新しい pin(`TestOrgSkill_ResultMissingIsInTheWatchRowAndTheAlertItems`)が落ちた。元に戻して通った |

`./scripts/run-test.sh` と `./scripts/run-verify.sh` は回していない。`internal/cli` と `internal/org` の全体も回していないので、pin 以外のテストが今回の文の変更で落ちる可能性は、narrow の絞り込みの外に残る。

## 直さなかったもの

- V7 の残り: `/org` の skill の `watch` の行と `/director` の skill の `result_missing` の項は、まだ「RESULT」だけを言う。どちらも既存の pin の文に入っていて(`TestOrgSkill_...`、`TestDirectorSkill_...`)、INFO なので依頼の範囲(README と leader の雛形)にとどめた。`/org` の「受信箱」の項は元から「RESULT(か REVIEW)」と言っている
- V8: rule の座席の確かめの書き方が雛形より緩い。INFO。次に段落を直すとき
- V9: `internal/config/config.go` の `IntervalSeconds` と `StallMinutes` のコメントが `result_missing` を挙げない。INFO で Go のコメントなので、次にこの構造体に触るとき
- V11: 台帳 530 行目に、`sentDetailsText` を読み手の候補として添える。任意。530 の Trigger には当たっていないので書かなかった
- test report の Test gaps 6(`internal/cli` に `result_missing` のテストがない): この run で help の pin を足したので、help の範囲は閉じた。見張りの判定は `internal/org` の中で閉じたまま

## PR 本文の Known gaps に載せるもの

1. 同じ TASK でも、台帳か受信箱が読めないときと `recurred` の追記に失敗したときは、新しい ALERT と 2 つ目の件になる(黙る側には倒れない)。test がなく、台帳 563 の (c)。rule、skill、仕様、`--help` に書いた
2. 受信箱の `result_missing` の件に TASK_ID が載らない。別の TASK の件が増えたので、1 つの座席に TASK_ID のない件が並びうる。台帳 563 の (a)
3. 実機の確かめがない。座席の送信の手順(`SEND FAILED:`、`history.sh` の確かめ)も、見張りの `result_missing` も、fake の herdr と fake の history.sh で固定しただけ。私用の agmsg の写しで座席の RESULT が leader に届くかも未確認。台帳 563 の (d) と 377。#361 の原因は、agmsg の保存先に座席の sandbox から書けないという見込みのままで、確かめていない(`docs/evidence/org-result-missing-hook-probe-2026-11-22.md`。hook は止めていなかった)
4. 受信箱の件ごとに watcher が起きる(`watcher_enabled` が既定で on)。台帳 563 の (b)
5. claude の reviewer が gate を `run_in_background` で待つあいだ、`result_missing` が上がる。stall の代わりに出るので知らせの数は増えない(台帳の `run_in_background` の行)
6. `watch_result_missing_test.go` が 1,672 行で 800 行の目安を超える。台帳の末尾の行
7. 動いている org は座席を立て直すまで前の雛形のまま(見張りは新しい binary で動く)。計画の Rollout にある

## Insight event

`./scripts/insights-append.sh --slug org-result-missing --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を 1 回打った(rc 0、UTC 04:58:49)。UTC の日付が 11-22 のうちだったので、既存の `2026-11-22-org-result-missing.jsonl` の 5 行目になった。`git status --short` で見えるのは、そのファイルの `M` だけで、日付つきの slug で作られる余分なファイルはない。
