# sync-docs report: org-spec-complete

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-spec-complete.md`(承認の digest `98781cfa6662` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `765bcc46`、branch は `docs/org-spec-complete`。run 開始時の HEAD は `85256177`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-21-org-spec-complete.md`(Merge yes)、`docs/reports/verify-2026-10-21-org-spec-complete.md`(pass。INFO V1〜V3)、`docs/reports/test-2026-10-21-org-spec-complete.md`(pass。INFO T1〜T3)
- この run が変えたのは、tech-debt の既存の 1 行、計画の Progress checklist の 1 項目、この report、insight event。コード、script、skill、rule、仕様、根拠の表、先行 3 report、計画の本文には触っていない

## Summary

`765bcc46...85256177` の差分は docs だけである。仕様に Status の節と印と PR 番号を足し、根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` を新しく置き、tech-debt に 1 行足した。この差分が起こす drift を、仕様の状態を述べる文書(README、AGENTS.md、`docs/architecture/repo-map.md`、`docs/quality/`、`docs/recipes/`、`.claude/rules/ralph/`、`/org` と `/director` の skill、`internal/org/prompts/`、2026-08-01 の仕様)から探した。「9 段目が進行中」「次の段」「未実施」と書く文書や、仕様の行番号を指す生きた文書は見つからなかった。

直した点は 2 つある。test report の T2(INFO)が提案した、doctor のテストの load による失敗を tech-debt の既存の行に足すこと。計画の Progress checklist の「Test artifact created」が `[ ]` のまま残っていたこと。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`("Two doctor tests" の行、311 行) | 欄は 5 つのまま。(1) 第 1 欄の末尾に、3 本目のテスト `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` が org-spec-complete の `/test` で同じ原因(`claude --version timed out after 5s`)で 1 回落ち、単独 5/5 と 2 回の打ち直し(go1.25.8、go1.26.0)で通ったことを足した。(2) 第 3 欄の「Seen once」を「Seen in two runs (2026-10-19, 2026-10-21)」にした。(3) 第 4 欄の trigger を「A second failure of either test」から「Another failure of any of the three tests」にした。(4) 第 5 欄に `internal/cli/doctor_codex_writable_root_test.go` と test report(Failure analysis)を足した。参照は関数名と test の名前で、行番号は使っていない |
| `docs/plans/active/2026-10-21-org-spec-complete.md`(Progress checklist) | 「Test artifact created」に印を付け、test report の結果(pass、`84ef6f59`、INFO T1〜T3)と doctor の flake の扱いを 1 行で書いた。本文(Progress checklist より上)は変えていない。digest は前後で `98781cfa6662` |
| `docs/reports/sync-docs-2026-10-21-org-spec-complete.md` | 新規 |
| `docs/insights/events/2026-10-21-org-spec-complete.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md`(`ralph org ...` の動詞の表と 254〜268 行の段落) | 変更なし | `director`、`merge`、`gate` などの動詞の動きを書くだけで、仕様の段の状態を言わない。`stage 9`、`9 段`、`FR-12`、`next stage`、`in progress` の grep は 0 件 |
| `AGENTS.md`、`docs/architecture/repo-map.md`(`docs/specs/`、`docs/evidence/` の行) | 変更なし | 新しい `docs/evidence/org-spec-evidence-2026-10-21.md` は前からある `docs/evidence/` の下の 1 file で、新しい種類の置き場を足さない。`docs/evidence/README.md` に file の一覧はない。短い地図のままにする |
| `docs/quality/quality-gates.md`、`docs/quality/definition-of-done.md` | 変更なし | gate の行(79〜82 行)は動詞と script の動きで、仕様の状態を持たない |
| `docs/recipes/`(`director-launch-cmd.md`、`director-wake.md`、`codex-seat-permissions.md`) | 変更なし | 仕様の file を指さず、日付つきの live-fire の evidence を指す。仕様の印や段の状態に依る文はない |
| `.claude/rules/ralph/`(`ralph-workflow.md` の `/director` の説明、`agent-messaging.md`) | 変更なし | 動きの説明で、仕様の段の状態を持たない |
| `/org` と `/director` の skill(4 つの写し)、`internal/org/prompts/` | 変更なし | `org-multi-org-director` を指す文がない(grep 0 件) |
| `docs/specs/2026-08-01-org-runtime.md`(29〜31 行の「後継の仕様」) | 変更なし | 「その仕様の各段が入った時点でそちらの記述が優先する」は、9 段がすべて入った今も正しい。この仕様の本文は書き換えないと同じ節が言う |
| Go のコードがこの仕様を引くコメント(14 file) | 変更なし | 引くのは仕様の名前と FR の番号で、行番号を持つものはない。印を付けても文は合う。test report の 1 も、仕様を読む test が 0 件であることを確かめた |
| 仕様の行番号を指す文書 | 生きた文書は 0 件 | `git grep -E 'org-multi-org-director\.md:[0-9]'` の 24 件(evidence 1、report 23)はどれも日付つきの履歴(例: `docs/reports/sync-docs-2026-10-11-org-cross-watch.md` の `:63`)で、書かれた日の行番号である。Status の節で行が下がったが、履歴は書き換えない |
| 仕様の印と数 | 合っている | `grep -c '^- \[x\]'` 相当は 20(根拠の表の 19 と、前から `[x]` の FR-12)、`[ ]` は 12(表の 12、Status の「12 項目」)。FR-12 の行は `[x]` のまま。Status の PR は 28 本で、根拠の表の「段ごとの PR」と同じ並び |
| 仕様と根拠の表が引く tech-debt の行 | 合っている | 引用 16 種(仕様と表で重なるものは 1 つと数える)は、どれも `docs/tech-debt/README.md` のちょうど 1 行の第 1 欄に含まれる(F8 の行の引用は取り消し線の `~~` を除いて。verify の V2 と同じ)。表は「最初のセルの先頭の数語」で指す。`ralph org report`'s event timeline だけは第 1 欄に含む行が 3 行あるが、先頭が一致するのは 324 行の 1 行で、先頭の数語で指す規則なら 1 行に決まる |
| 仕様と根拠の表が引く path | 合っている | backtick の中の path 250 か所のうち、存在しないのは 3 つだけで、どれも path ではない。`docs/org-spec-complete`(branch の名前)、`internal/auth/` と `internal/auth/token.go`(受け入れ条件 4 の例の文) |
| 根拠の表のテスト名 | 合っている | 66 個すべてが、このブランチの `git grep -n -E '^func <名前>\(' -- '*.go'` で 1 件ずつ見つかる(test report の 6 と同じ結果) |
| 根拠の表の「NFR 入れ方」の行 | 合っている | 表が数える 25 本の PR の計画は `docs/plans/archive/` にある。`2026-10-07` から `2026-10-20` の `org-*` の 24 本は、self-review・verify・test・sync-docs の 4 種の report がそろい、`secret-scan-test-tmp`(#240)にもそろう |
| `docs/tech-debt/README.md` の新しい行(327 行、ALERT) | 合っている | 欄は 5 つで 323〜326 行と同じ。引く関数は `sendAlert`、`SendWatchdogAlert`(`internal/org/watch.go`)と `callWithTimeout`、`defaultDriverCallTimeout = 10 * time.Second`(`internal/org/verbs.go`)で、いまのコードにある。`sendAlert` の本文は、送信の失敗を stderr に書いて進み、送れたかに関わらず `PendingAlerts` に入れる。行の「FR-3 stays `[ ]`」は仕様の FR-3 の印と合う。計画の path は `docs/plans/active/` の形(`/pr` の `archive-plan.sh` が書き換える。`scripts/verify.local.sh` の `check_tech_debt_plan_refs` が読む形) |
| tech-debt の stage 9 を語る行 | 合っている | 171、216〜217、246、297〜299、304、309、313〜325 行を読んだ。どれも試行が済んだことを前提に書くか、日付つきの履歴である。「9 段目はまだ」と言う行はない。216 行の「a live run of a feature org is the trial of stage 9」(Why 欄)と 171 行の「the stage-9 trial runs two feature orgs」(RESOLVED の注)は、試行の前の言い方がそのまま残る。同じ行が試行の結果も記録しているので、誤りではなく、直さない |
| 計画 | 合っている | AC1〜AC8 と AC6b は `f16e90f8` で印が付いている。Progress checklist は「Test artifact created」を直した。残る未了は「PR created」だけ。verify の V3(AC の checkbox の遅れ)は `f16e90f8` で解消済み |

## Plan drift notes

- 計画の Affected areas に `docs/tech-debt/README.md` はない。ALERT の行は self-review の M1・L1 の直しで足したもので、計画の Progress に記録がある。この run が足した doctor の行の編集は既存の行の広げ方で、新しい行ではない
- verify の V1(見込みの `[ ]` が 9 項目、実際は 12)は、計画の Progress の「verify の V1 の記録」の行が FR-4 と受け入れ条件 5 を足して、すでに記録している。V3(AC の checkbox の遅れ)も `f16e90f8` で解消済み。この run が足したのは「Test artifact created」の印だけで、本文は変えていない

## Left open

- T2(doctor のテストの flake): tester の提案どおり tech-debt の既存の行を広げた(上の変更)。行が挙げる直し(テストが probe の timeout を設定できるようにする)は doctor とテストのコードの変更なので、この PR では扱わない。行の trigger(次の失敗か `probeBinary` の変更)が拾う。この編集は依頼の範囲(この差分が起こす drift)の外にあるので、lead が不要と見るなら 311 行の 4 つの Edit(第 1・3・4・5 欄)だけを戻せる
- T1(根拠の表のテスト名の照合が suite に入っていない): 表は日付つきの記録で、系列の 2 本目以降が表を更新するときに手で照合する(test report の evidence の log は `docs/evidence/*.log` の gitignore の対象でコミットされず、PR からは読めない。名前の照合は `git grep -n -E '^func <名前>\(' -- '*.go'` で打てる。計画の Risks は、系列の後の PR が表を更新すると書く)。tech-debt には足していない(INFO で、系列の PR が表を更新する運用の中にある)
- 24 件の履歴(evidence 1、report 23)に残る仕様の行番号: 書かれた日の行番号で、読む人は日付から仕様の版を引ける。直さない
- `docs/specs/2026-10-07-org-multi-org-director.md` の References の「1 段目の計画」の行(298 行付近)は「merge 後は `docs/plans/archive/`」と書く。計画はすでに archive にある(`2026-10-07-org-state-dir-common.md`)。文は条件つきで間違っていない。計画の AC7(印・注・Status・Rollout の PR 番号の外は変えない)と、依頼の「仕様の印と判定に触らない」の範囲外なので直していない
- 仕様の印と判定、根拠の表の中身: この run の範囲外(依頼で触らない)。印の根拠の強さは self-review と verify が見た

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-21-org-spec-complete.md`(編集のあと) | `98781cfa6662`(承認の digest と同じ) |
| `./scripts/check-sync.sh`(編集のあと) | rc 0。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。skill は編集していない |
| tech-debt の編集した行(311 行)と ALERT の行(327 行) | 欄はどちらも 5 つ。引いた `docs/plans/archive/2026-10-19-org-director-wait.md`、`docs/plans/active/2026-10-21-org-spec-complete.md`、`internal/cli/doctor_codex_writable_root_test.go`、test report は存在する。`TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` は `internal/cli/doctor_codex_writable_root_test.go` に 1 件ある |
| 上の「Surfaces checked」の grep、path、テスト名、行の先頭の照合 | 結果は表のとおり。scratchpad の python で打ち、仕様と根拠の表と tech-debt の行は読み取りだけ |

`go test`、`run-static-verify.sh`、`run-verify.sh` は走らせていない。この run が変えたのは markdown 2 file と insight event と、この report で、コードも script も skill も変えていない。`scripts/verify.local.sh` の既定の実行は全 test を走らせるので回さず、計画の path の存在を手で確かめた。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Linux での実行(test report の T3 と同じ)。仕様の印が根拠として十分かの判断(self-review と verify の範囲)。履歴の記録 24 件の中身

## Insight event

`./scripts/insights-append.sh --slug org-spec-complete --flow standard --phase sync_docs --verdict pass --cycle 1 --source skill` で `docs/insights/events/2026-10-21-org-spec-complete.jsonl` に 1 行足した。phase の値は script の受け付ける `sync_docs` である(依頼の文面は `sync-docs`)。cycle は `cycle-count.json` の値と同じ 1 を明示した。
