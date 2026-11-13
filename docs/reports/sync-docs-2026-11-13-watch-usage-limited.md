# sync-docs report: watch-usage-limited

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(承認の digest `f2971e9c4a70`)。issue #312 の 1 本目(#312a)
- Branch: `feat/watch-usage-limited`。run 開始時の HEAD は `19478d7b`、origin/main との merge-base は `4220ea7c`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1。この節から「Insight event」まで)。2 回目は末尾の「Cycle 2」の節。1 回目の記述のうち 357、462、488 行目の数字と言い方は、その節の表で訂正した
- 先行 report: self-review `8010c4ca`(更新 `b7faa2bd`。Merge: yes、未解決の LOW は L5 と N1)、verify `c191529c`(pass。LOW 6 件 V1〜V6、INFO 5 件 I1〜I5)、test `cfee1be1`(pass)
- 見た範囲: `git diff origin/main...HEAD`(この commit の前で 27 ファイル、+2,730/-106)。walkthrough は書いていない(`/pr` が決める)
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `f2971e9c4a70`、`git diff --numstat` は 5/4(箱 4 つと足した 1 行)。Go のコードとテストは変えていない

## Summary

verify の V1〜V6 をすべて扱った。V1〜V5 は文書の直し(skill 2 つの 4 つずつの写し、README、quality-gates の 2 つの写し、multi-org-director の仕様)、V6 は tech-debt の Trigger に当たった 9 行(V6 の 8 行と、V4 で当たる 118 行目)への持ち越しの note。self-review の L5 と N1、test report の穴は tech-debt の新しい行にした(485〜487 行目)。sync-docs の size check で、この branch の新しいテストファイル 1 つが 800 行を超えていたので、それも 1 行にした(488 行目)。verify の I1 は 452 行目の Impact に 1 文足して扱った。AGENTS.md、`docs/architecture/repo-map.md`、`agent-messaging.md` の 2 つの写し、`ralph org watch --help` は実装と合っていて、直した所は無い。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md` と、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | 件の処理の `recurred` の段落の括弧を絞った(V1)。4 つの写しは `cmp` で同じ |
| `.claude/skills/org/SKILL.md` と 3 つの写し | 長時間運用の項に `usage_limited` を足し、その項の行を折り直した(V2)。4 つの写しは `cmp` で同じ |
| `README.md` | 133 行目の `ralph org …` の行に `usage_limited` を足した(V3) |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | Watchdog pulse layer の行を直した。同じ 1 行を 2 つの写しで(V4) |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-6 に 2026-11-13 の決定を 1 項足した(V5)。114 行目と 117 行目の日付つきの注は書き換えていない |
| `docs/tech-debt/README.md` | 既存の 10 行(88、118、180、262、353、357、452、462、464、477)に追記(Trigger に note と、Related に計画の path。452 行目は Impact に 1 文)。新しい 4 行を末尾に足した(485〜488 行目)。下の「V6」「新しい tech-debt の行」 |
| `docs/plans/active/2026-11-13-watch-usage-limited.md` | `## Progress checklist` の `Implementation started`、`Review artifact created`、`Verification artifact created`、`Test artifact created` を `[x]` にし、sync-docs の 1 行を足した |
| `docs/reports/sync-docs-2026-11-13-watch-usage-limited.md` | この report |
| `docs/insights/events/2026-11-13-watch-usage-limited.jsonl` | `sync_docs` の行(cycle 1、pass)。1 行から 2 行 |

commit は 1 つ(`docs: sync docs for watch-usage-limited`)。

## V1: director skill の括弧

直す前の文は「ack か resolve のあとに止まった座席は新しい件になる」と、条件がいったん下りて再び立つ場合を分けずに書いていた。コードを読み直して確かめた。`recordUsageLimitRecurrence` は、条件が立っているあいだに座席が増えた(`grown`)ときは open の件にだけ `recurred` を足し、条件が下りてから再び立った(`grown=false`)ときは open の件にも acked の件にも足して、`note=` に止まった座席をすべて書く。新しい文は次のとおり。

```
条件が立っているあいだに増えた座席は、
件が open のときだけ `recurred` になり、ack か resolve のあとに止まった
座席は新しい件になる。条件がいったん下りてから再び立ったときは、
resolve されていない件(acked の件も)の `recurred` になり、`note=` に
そのとき止まった座席がすべて出る。
```

4 つの写しで同じ。折り返しの幅は 79 桁(全角を 2 と数える)以内。

## V2: org skill の長時間運用の項

「全 org の停滞/生存/スコープ変更の ALERT」のあとに「座席が利用上限で止まった org の条件 `usage_limited` の件」を足した。足したことで折り返しが 106 桁になった行が出たので、その項を折り直した(最長 75 桁)。コマンド表の `watch` の行と「見張りの ALERT の件」の項は、branch の中で直っていて、触っていない。

org skill の表の行の長さ(行全体の文字数、`git show` で測った)は、`watch` が origin/main で 3,233、HEAD で 3,378。`stop` は 1,208、`disband` は 1,219 のままで、tech-debt の F-8 の行が引く数字は動いていない。

## V3: README の `ralph org …` の行

「stall, liveness, scope-change, and usage_limited ALERTs to its leader」と書き、「an `ALERT` item in the inbox for an ALERT the leader did not act on, one about the leader, or a usage_limited one, which it records in the same cycle」とした。280 行目は branch の中で直っている。

## V4: quality-gates の Watchdog pulse layer の行

root の 83 行目と雛形の 82 行目は同じ文で、2 か所を直した。ALERT の種類に `usage_limited` を足し、escalation の説明に「usage_limited の件はその周期のうちに記録される」を足した。もう 1 か所は stall を出さない条件の文で、上限の画面の座席は stall を立てず org に 1 件の usage_limited になること、そこに途中で加わった座席は、件が open なら `recurred` の note、acked か resolve 済みなら新しい件になることを足した。雛形の写しに触れたので、tech-debt の 118 行目の Trigger に当たる(V6)。

## V5: multi-org-director の仕様

FR-6 の 2026-11-12 の注(#307 の後半)のあとに、2026-11-13 の決定を 1 項足した(#312a)。見張りの escalation の理由に `usage_limit` が加わること、件の `SEAT` が leader で `SEATS` に座席が並ぶこと、上の 2026-10-23 の注の「件を書かず今までの経路に落ちる条件」に usage_limited の座席が 0 のときと座席・driver が識別子の形でないときが加わること(`checkWatchEscalationValues` が `checkUsageLimitedSeats` を呼ぶため)、2026-11-12 の注の「続きは新しい件を作らず recurred」が usage_limited では、条件が立っているあいだに座席が増える場合に当たらないこと、を書いた。詳細は org-runtime 仕様の 2026-11-13 改訂 (a)〜(c) を指す。114 行目と 117 行目は日付つきの記録なので、書き換えていない。

## V6: tech-debt の Trigger

note は Trigger セルの末尾に足し(`. 2026-11-13: feat/watch-usage-limited (#312a) hit ...`)、Related の末尾に `docs/plans/active/2026-11-13-watch-usage-limited.md` を足した(`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない)。行の読み方は、Trigger の本文と、これまでの note の形に合わせた。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 88「6 batchable LOW findings」 | `watch.go`。(1) `watchStatusFile` にフィールドを足し、gofmt が `Escalated` の行をそろえた。(3) `sendAlert` と `watchEscalationBody` を変えた | (1) と (3) に当たり、払っていない。`status.Escalated` は書かれ、消され、誰にも読まれないまま。(3) の `SEAT: ` の見出しは変わらない。`watch.go` は 2,146 行から 2,246 行(merge-base `4220ea7c` と `e25def99` で `wc -l`) |
| 118「templates/base/docs/quality/quality-gates.md」 | この sync-docs の編集(V4)。雛形の写し | Trigger の言う「次のタッチ」に当たる。「Must pass in CI」の節と雛形の verify の連なり(`check-coverage.sh` も `check-pipeline-sync.sh` も呼ばない)は変えていない。Scope の外。持ち越し。Related は日本語の読点の形に合わせた |
| 180「The `/org` skill's command table」(a) | `watch` の行(行全体が 3,233 字から 3,378 字) | (a) に当たり、払っていない。`stop` と `disband` は変わらない(1,208 字と 1,219 字)。表の作り直しは写しが 4 つの別の変更で、計画の Scope の外。`watch` の行は元から 2 つの cell より長い |
| 262「Test gaps left by org-director-attach」 | director skill の `### 件の処理`(受信箱の処理の節)。写し 4 つ | (1)〜(3) は開いたまま。`director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は触っておらず、`WaitInbox` と受信箱の期限の道も変わらない |
| 353「`sendAlert`」 | `sendAlert` が pending を返すようにした | 送り方と時間切れは変えていない。agmsg の `send.sh` が詰まる理由は、負荷をかけた実行が要るので Scope の外。持ち越し |
| 357「The next change to the watch's escalation」 | `escalateAlert`、`escalateToInbox`、`checkWatchEscalationValues`、`watchEscalationBody`、`escalateWithoutInbox`、新しい `watchEscalationReason` | 新しいコードは `watch_usage_limit.go`(461 行)、そのテスト `watch_usage_limit_test.go`(936)、`usage_limit_screen.go`(96)、`usage_limit_screen_test.go`(402)に置いた。最後の Go の commit `e25def99` で `watch.go` は 2,246 行(merge-base で 2,146)、`watch_test.go` 4,463、`watch_all_test.go` 2,613、`watch_recurrence.go` 275、`watch_scope.go` 467 は merge-base と同じ。移す変更は Scope の外。持ち越し |
| 462「The next change to the `scope_change`, `stall`, or recurrence code」 | `evaluateSeat` の stall の道、`sendAlert`、`escalateToInbox`、`escalateWithoutInbox` | 行の Trigger と (9)、(10) に当たり、(1)〜(3)、(5)〜(10) は払っていない。(5)、(6)、(8) の Trigger は当たっていない(`watch_scope.go`、`watch_recurrence.go`、`leaderActedSinceAlert`、`foldIntoPendingAlert` に触れていない)。(11) も当たっていない(`evaluateCycle` の doc に条件の段階を足したが、`escalateAlert` の doc と `evaluateCycle` の中のコメントは元のまま。`git diff` で「records nothing new」が増減していないことを見た) |
| 464「The `/director` skill and the scaffold version note」 | director skill の写し 4 つ | director が scaffold の版の知らせを人に伝えるか、`ralph upgrade` を実行するかは製品の判断で、計画の Scope の外。skill は今も `ralph upgrade`・`manifest.toml`・scaffold のどれも書いていない(`git grep -c -i` が 0)。prompt と 4 つの動詞の `--help` は変えていない。持ち越し |
| 477「A hook that was changed after codex approved it」 | 見張りの ALERT の条件を足した(`usage_limited`)。`resume_screens.go` に `UsageLimit` の印を足した | 「Hooks need review」の dialog の条件は足していない(Scope は利用上限)。見張りが stall の座席の画面を読むようになったので(`readUsageLimitScreen`)、その条件は同じ読みに載せられる、と書いた。持ち越し |

掃引で当たった行(verify も handoff も挙げていない)は、452 行目の Impact 1 文(下の I1)のほかにない。Trigger を読んで当たらないと判断した行は次のとおり。

| 行(先頭の語) | 理由 |
|----|----|
| 177「`ralph org stop --all` and `disband --all`」 | 見張りの `--all` の話ではなく、`stop --all` の時間と `disband` の競合。この branch は触れていない |
| 450、451 | 「phase 2: 見張りか `ralph status` が `Waiting` の文で画面を読む」が Trigger。この branch が見張りに読ませる文は `UsageLimit` の印だけで、`Waiting` の文は読まず、表示も足していない |
| 461 | `resume_screens.go` の Trigger は `resumeCodexFooterLine` と `empty` の placeholder の規則。この branch は表の型に `UsageLimit` を足しただけで、どちらにも触れていない |
| 371、401 | Trigger は recipes と `checkRetiredOrgStateDirEnv`。この branch が README で触れたのは `ralph org …` の行だけ |

## 新しい tech-debt の行(485〜488 行目)

| 行 | 中身 | 出どころ |
|----|----|----|
| 485 | 集合にいた座席の画面の読みが 1 回失敗すると stall になって集合を外れ、次に読めた周期で新しい座席と数えられる。件が open なら `recurred`、acked なら新しい件 | self-review L5(計画の AC4、AC6 のとおり。未解決) |
| 486 | 画面が読めなかった理由は leader への stall の ALERT(agmsg)の最後の 1 行にだけ載り、受信箱の件の本文と watch の stderr には出ない | self-review N1(未解決) |
| 487 | test report の Test gaps 1〜6 をまとめた 1 行。(a) leader 以外だけが上限の場合のテストが無い(overlay の probe で通ることは確かめた)、(b) leader が動いても上限でもないとき、leader の stall と usage_limited の 2 件ができる(製品の判断)、(c) 通らない枝 5 つ、(d) 変異 M2・M3 の guard が 1 行だけで固まっていることと、`watchEscalationReason` の順が本文で見えないこと | test report の Test gaps 1〜6 |
| 488 | `internal/org/watch_usage_limit_test.go` が 936 行(`e25def99` で `wc -l`)。先頭の約 245 行は fixture と helper で、`internal/org/watch_usage_limit_fixture_test.go` に移すと約 690 行になる | この sync-docs の size check。verify の `wc -l` は本体の 3 ファイルだけを数えていた |

test report の Test gaps 7(self-review の L5 と N1 と同じもの)は 485 と 486 に入れ、8(上限の 3 つの文の実機の画面)は 452 行目が持つので足していない。9(AC9 の文面をテストで固定していない)は verify が文書を読んで確かめた範囲で、行にはしていない。

## I1〜I5

- I1: 452 行目の Impact に 1 文足した。この branch から見張りの `usage_limited` も `resume_screens.go` の同じ印を読み、実機の画面の文が違えばその座席は今までどおり stall の ALERT になる(安全側)
- I2: 扱わない。leader の role prompt が `usage_limited` の ALERT を受けたときの扱いを書いていないのは、計画の文書の範囲外で、ALERT の本文が org の条件だと言っている
- I3: 扱わない。director skill の「座席を立て直す動詞は issue #312 で扱っていて、この版にはない」は、#312c(origin/docs/spec-org-reseat)が先に入れば古くなる。後から merge する側が直す。衝突するのは AGENTS.md と `docs/specs/2026-08-01-org-runtime.md`(verify の試しの merge)
- I4: `Implementation started` などの箱は `[x]` にした。AC の箱は `[ ]` のまま(本文は変えない)
- I5: self-review の insights 行は最初の run の数(low 7)のまま。`/pr` の前に合わせるなら、記録だけの修正の条件に当たる(`docs/reports/` の report と insights の jsonl だけの commit)

self-review の N2(計画の D4 が report の指摘 ID を指す)は `c8681f0d` で直っている。self-review の report の N2 の行は、その前の版のまま未解決と書いている。

## 確かめたこと(文書が実装と合っていた所)

- `AGENTS.md` の `internal/org/` の行: branch の中で `watch_usage_limit.go` と `usage_limit_screen.go` の説明が入っていて、`git diff origin/main...HEAD` の関数名(`UsageLimitScreen`、`usage_limited`、`REASON usage_limit`)と合う
- `docs/architecture/repo-map.md`: org の行は `.harness/state/org/` の 1 行だけで、watch の条件を並べていない。足す所は無い
- `.claude/rules/ralph/agent-messaging.md` と雛形の写し: branch の中で直済み。`usage_limited` の節の挙動(`SEAT` が leader、`SEATS` の見出し、open の件だけ `recurred`、acked か resolve 済みは新しい件、下りてから再び立つときは前の規則)は `recordUsageLimitRecurrence` と合う。`check-sync.sh` で 2 つは同じ
- `ralph org watch --help`(`internal/cli/org_watch.go`): `usage_limited` の ALERT、`SEATS` の見出し、同じ周期で件にすること、open の件だけ `recurrence` になること。コードと合う
- `docs/specs/2026-08-01-org-runtime.md` の 2026-11-13 改訂 (a)〜(d): コードと合う(verify が照合済み)
- `.claude/rules/ralph/model-routing.md`、`docs/quality/definition-of-done.md`、`.claude/rules/ralph/post-implementation-pipeline.md`: この branch の変更に関わらない

## 残っていること(lead へ)

- コードの動きとテストは変えていない。パイプラインの再走は要らない
- 払っていないもの: 持ち越しの note を足した 9 行の Trigger と、新しい 4 行の中身。L5 と N1 はコードの変更で、直すなら `/self-review` からの再走になる。直さずに進めるなら、PR 本文の Known gaps に 485、486 行目の 1 行ずつを書く
- 487 行目の (b)(leader が上限の座席を待つとき、leader の stall と usage_limited の 2 件ができる)は製品の判断で、決まっていない。(a) の test は probe が通ったので、足すだけで済む
- 実機の画面は、claude の session、Opus、Sonnet と codex の 2 つの上限の文を見ていない(452 行目)。見張りがこの文で usage_limited を立てることは fixture の上でしか確かめていない
- 同じファイル(`docs/tech-debt/README.md`)を別のセッションも編集している。この commit は既存の 10 行への挿入と末尾の 4 行の追加で、既存の行は消していない。merge の衝突は末尾の追加行と、88・118・180・262・353・357・462・464・477 行目の Trigger と Related の末尾で起きうる
- 下流への注意: org skill と director skill の写し(`templates/base/` 側)は `ralph upgrade` で下流に配られる。新しい skill の文は新しい binary を前提にする(release ノートに書く)

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(scratchpad の `tdcheck2.py`)で、変わった行は 88、118、180、262、353、357、452、462、464、477、末尾の 4 行(485〜488)。変わった行も新しい行も列は 5、バッククォートは偶数、行は `| ` で始まり ` |` で終わる。変わった既存の行は、変わった列がどれも元の文字列の末尾への追記
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` が 122 件で、無い参照は 0 件。新しい行の `.go:NNN` の形は 0 件。U+FFFD は 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-watch-usage-limited.md` は編集の前後とも `f2971e9c4a70`
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`(rc 0)。`./scripts/check-sync.sh`: `PASS: all files in sync.`(rc 0)。`git diff --check`: rc 0
- director と org の skill の 4 つの写しは、それぞれ `cmp` で同じ(8 組)
- `go test ./internal/org -run 'Skill|Director|ReportNames|SendDefaults|Prompt' -count=1`: ok(315 秒)
- `go test ./internal/cli -run 'TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs|Skill|Docs' -count=1`: ok(0.5 秒)。`internal/cli` で、この commit が触った文書(README と org skill の 2 つの写し)を読むテストはこれだけ(`SKILL.md`・`README.md`・`quality-gates.md` を読む `_test.go` を `grep` で洗った)
- `go test ./internal/scaffold ./internal/upgrade -count=1`: ok(雛形の写しを読む側)
- 通らなかったもの: `go test ./internal/cli -run 'Skill|Docs|Org' -count=1` は既定の 10 分の時間切れで止まった(`TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` の実行中。失敗したテストは 0 件)。`Org` の語が `internal/cli` の org のテスト全部に当たる広さで、ほかのセッションの `go test ./...` が同じ機械で回っていた。この run は文書の編集で結果が変わるテストを含まないので、狭めた上の run に置き換えた。`Org` の全体は走り切っていない(未確認)
- 実行していないもの: `run-verify.sh` の全体(コードの動きを変えていない。verify と test が走らせ済み)

## Insight event

```
./scripts/insights-append.sh --slug watch-usage-limited --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

`docs/insights/events/2026-11-13-watch-usage-limited.jsonl` が 1 行から 2 行になった(`ts` は `2026-11-13T02:33:42Z`、`phase` は `sync_docs`、`cycle` は 1、`verdict` は `pass`)。`git status --short` に `<date>-<date>-` 形の余分なファイルは出ていない。

---

## Cycle 2

- Date: 2026-11-13
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2)。run 開始時の HEAD は `898c0990`。cycle 1 の cross-review の ACTION_REQUIRED の直し(`34e5bd86`、`193b044a`)のあとの通しの再実行で、branch は origin/main を `9f95c51a` で取り込んだ。merge-base は `4220ea7c` から `bb416c48` に動いた
- 入力: verify cycle 2(report の commit は `fb273771`、HEAD `a7321fa9` を見た。pass。LOW V7〜V9、INFO I7〜I12)、self-review cycle 2(`bce90b38`。Merge: yes。LOW N3、N4)、test cycle 2(`78fabaf8`、HEAD `9f95c51a` を見た。pass。Test gaps 1〜10 で、3 と 5 を書き換え、10 を足した)
- 見た範囲: `git diff origin/main...HEAD`(この commit の前で 37 ファイル、+3,431/-152)。cycle 1 の sync-docs `86f7c1f8` のあとに Go を変えたのは `34e5bd86`(`inbox.go`、`inbox_test.go`、`watch_usage_limit.go`、新しい `watch_usage_limit_race_test.go`)と `193b044a`(`watch_recurrence.go` のコメント)だけ。`git diff 193b044a HEAD -- internal/org` は空
- 制約: Go のコード、テスト、skill、rule は変えていない(V7 の仕様の 1 文を除く)。計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` の 1 行の追加だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `f2971e9c4a70`、`git diff --numstat` は 1/0

### Summary

V7 は仕様の 1 文。V8 は 3 行(211、485、487)への持ち越しの note、V9 は cycle 1 の note の数字と言い方の訂正(357、462、488)。I9 は N3 と N4 を 1 行(492 行目)に、I10 は `inbox_test.go` の大きさを 1 行(493 行目)にした。test report の Test gaps は 487 行目の (a)(c)(d) を cycle 2 の言い方に書き直し、新しく行にするものは無かった(10 は 492 行目と 487 行目の (d) が持つ)。

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | 83 行目、2026-11-13 改訂 (c) の「recurred を足すあいだに件が resolve されたときも新しい件にする」を「resolve か ack されたときも」にし、「(open かどうかは、足すときに inbox のロックの中でも確かめる)」を足した(V7) |
| `docs/tech-debt/README.md` | 既存の 6 行(211、357、462、485、487、488)を直し、末尾に 2 行(492、493)を足した。下の V8、V9、I9、I10 |
| `docs/plans/active/2026-11-13-watch-usage-limited.md` | `## Progress checklist` に sync-docs cycle 2 の 1 行 |
| `docs/reports/sync-docs-2026-11-13-watch-usage-limited.md` | この節と、冒頭の Pipeline run の行 |
| `docs/insights/events/2026-11-13-watch-usage-limited.jsonl` | `sync_docs` の行(cycle 2、pass) |

commit は 1 つ(`docs: sync docs for cycle 2 of watch-usage-limited`)。

### V7: 仕様の (c)

直した文は 83 行目の 1 文だけ。同じ「足すあいだ」の言い方を `README.md`、`AGENTS.md`、`docs/specs`、`docs/quality`、`docs/recipes`、`.claude`、`templates` で探して(`git grep -F -f` で 7 つの言い回し。`docs/plans`、`docs/reports`、`docs/evidence` は除いた)、当たったのは仕様の 83 行目と、別の話の語が一致した tech-debt の 1 行(org-state-dir-local の行)だけだった。計画の Scope と AC4 にも同じ言い方がある(verify の I7。承認の digest の中なので変えない。Progress が記録している)。`agent-messaging.md` と `templates/base` の写しは「open のときは recurred、acked か resolved のときは新しい件」と時点を分けずに書いていて、狭い言い方ではないので変えていない。multi-org-director の仕様の FR-6(119 行目)も「open のときだけ recurred、acked か resolve 済みなら新しい件」で、同じ。

### V8: cycle 2 の直し(`34e5bd86`)が当たった Trigger

note は Trigger セルの末尾に `2026-11-13, ...: feat/watch-usage-limited (#312a) hit this trigger with ..., but did not pay it: ... Carried over to the next change that meets this trigger` の形で足した(同じ行の 2026-11-12 の note の形)。

| 行 | 当たった所 | 書いたこと |
|----|----|----|
| 211 (a) | `inbox_test.go` に `TestInboxAppendRecurredIfOpen_*` の 3 本(1,517 行から 1,655 行。`e25def99` と `34e5bd86` で `wc -l`) | `TestPrintableInboxText` は `escalate_test.go` のまま。テストを別のファイルへ移すのは Scope の外。Related に計画を足した |
| 485 (L5) | `recordUsageLimitRecurrence`(座席が増えたときは `AppendRecurredIfOpen` で足す) | 直しは「読んだ件と append のあいだ」の窓で、画面の読みの失敗には触れていないので L5 は払っていない。Related には cycle 1 で計画が入っている |
| 487 (Test gaps) | `watch_usage_limit.go` と、新しい test file `watch_usage_limit_race_test.go` | (a) は足していない。(c) の `recur` の失敗の枝は直しが変えた枝で、まだテストがない。(d) の 2 行目も足していない。行の本文は下のとおり書き直した |

487 行目の本文は、test report cycle 2 が見つけたことに合わせた。verify の推定は使っていない。

- 見出し: 「Test gaps 1 to 6」に「その cycle 2 の run が 3 と 5 を書き換え、10 を足した」を足した
- (a): 「probe は cycle 1 で回し、cycle 2 では回し直していない」を足した(test report の Test gaps 1)
- (c): 「`AppendRecurred` が `ErrInboxResolved` 以外で失敗する枝」を、「`recur`(`AppendRecurred`、座席が増えたときは `AppendRecurredIfOpen`)が `ErrInboxResolved` と `ErrInboxAcked` 以外で失敗する枝で、stderr の文をテストが固定していない」にした。行番号は書かず、名前で引いた(verify の `watch_usage_limit.go:269-272` の代わり)
- (d): 「M2 と M3 の guard は 1 行(acked の件)」を、test report の変異の節のとおりに書き直した。M6(座席が増えた条件を `AppendRecurred` で足す、つまり直しを外す)を落とすのは競合のテスト 1 本だけ。M3 は競合のテストと「e1 acked」の行が落とす。M2(読んだ時点の状態の確かめ)は結果の変わらない変異で、落とすテストはない。`appendRecurred` の `openOnly` の行(`inbox.go`)には変異を当てていない。条件が立っているあいだに座席が 2 回続けて増える 2 行目が足りない。verify は M3 を競合のテストが落とすと読みで推定したが、test report は走らせて確かめ、落とす assert が seam の回数と件の結果の 2 つだと見つけた(診断の overlay)。行には、M3 を競合のテストと「e1 acked」の行が落とす、とだけ書いた。assert の内訳は test report に任せた
- Related: 「(Test gaps 1 to 6, and 10 for (d))」

私の掃引(Trigger セルから取り消し線の部分を除き、26 の語で探した。語は `inbox.go`、`AppendRecurred`、`ErrInboxResolved`、`ErrInboxAcked`、`InboxStore`、`beforeAppendRecurred`、`watch_recurrence.go`、`watch_usage_limit`、`recordUsageLimitRecurrence`、`2026-08-01-org-runtime` など)で 22 行が語に当たった。Trigger を読んで当たったのは 211、485、487 と、コメントだけの変更が当たる 462。357 と 488 は当たり直さず、cycle 1 の note の訂正だけ(V9)。当たらないと判断した行は次のとおり。

| 行 | 理由 |
|----|----|
| 207 | Trigger は `withFileLock` か `InboxStore` の lock の次の変更。`34e5bd86` は lock を使う新しい method `appendRecurred` を足しただけで、`withFileLock` も lock の作りも変えていない。新しいロックのテスト(`TestInboxAppendRecurredIfOpen_ChecksTheStateUnderTheInboxLock`)は同じ process の goroutine で、(a) の child process の flock の形ではない |
| 489 | Trigger の `usageLimitRecurNote` と `fitUsageLimitedSeats` は、cycle 2 の hunk が触れた関数に入っていない(`git diff e25def99 193b044a -- internal/org/watch_usage_limit.go` で、変わったのは `recordUsageLimitRecurrence` の doc と本体だけ) |
| 486 | `usageLimitUnreadNote`、`readUsageLimitScreen`、`watchEscalationBody` の stall の枝は変わっていない |
| 353、355、258、262、180、118 | cycle 2 が編集した所は `sendAlert`、`InboxDeadlineDue`、director skill、`/org` skill の表、`templates/base` の quality-gates のどれでもない。この pass も skill と quality-gates は編集していない |
| 365 | `prompts_leader_inbox_test.go` の語に `inbox_test.go` が部分一致しただけ |
| 177、198、206、342、371、376、490 | `docs/specs`、`reports/`、`InboxStore` などの語が別の話で出ているだけ |

この pass 自身が編集した `docs/specs/2026-08-01-org-runtime.md` と `docs/tech-debt/README.md` を Trigger に挙げる行は無かった。

### V9: cycle 1 の note の訂正

cycle 1 の記述(この report の先の節と、357、462、488 行目の note)を、`wc -l` と `git log` で測り直して直した。数字はすべて `193b044a` の木(HEAD と `internal/org` は同じ)のもの。

| 行 | cycle 1 の記述 | 訂正 |
|----|----|----|
| 357 | `watch_usage_limit.go`(461 行)。「`e25def99`、branch の最後の Go の commit」。`watch_recurrence.go` 275 は merge-base と同じ | `watch_usage_limit.go` は 470 行。branch の `internal/org` の Go を最後に変えた commit は `193b044a`。`watch_recurrence.go` は 277 行(`4220ea7c` で 275。`193b044a` が `beforeAppendRecurred` のコメントを変えた)。`watch.go` 2,246(`4220ea7c` で 2,146)、`watch_test.go` 4,463、`watch_all_test.go` 2,613、`watch_scope.go` 467 は `4220ea7c` と同じ。`watch_usage_limit_race_test.go`(63 行)を新しいコードの一覧に足した。cycle 2 の直しは escalation のコードにも `watch_test.go` にも触れていないので、この行の Trigger には当たり直していない。merge-base は `9f95c51a` で動いたので、「merge-base」の語をやめて `4220ea7c` と書いた |
| 462 | 「`watch_recurrence.go` … は触れていない」(5)(6)(8) の Trigger | 「in the first run」を足し、cycle 2 の文を足した。`193b044a` は `beforeAppendRecurred` のコメントだけを変えた(275 行から 277 行) |
| 488 | 「`wc -l` at `e25def99`、branch の最後の Go の commit」 | `193b044a`。936 行は `e25def99` 以降同じ。cycle 2 の直しのテストは新しい 63 行の file に置かれ、この file は増えていない、と 1 文足した |

462 行目の判断: コメントだけの変更は、Trigger が名指しするファイルへの変更として数える。この register に前例がある(201、254、418 行目の note は、コメントだけの編集を「hit this trigger ... but did not pay it」として数えている)。`watch_recurrence.go` は行の Trigger が名指しするファイルなので、行の Trigger には当たる。(1)〜(11) の Trigger は関数名で書かれていて(`evaluateReservedScope` の doc、`scopeRecordsAtBase` など、`leaderActedSinceAlert`、`foldIntoPendingAlert`、`recordRecurrence`、`reconcilePendingAlertsWithInbox`、`escalateAlert` の doc)、コメントがあるのはテスト用の seam `beforeAppendRecurred` で、どれにも当たらない。note は「当たったが払っていない」と書き、持ち越しにした。

### I9: self-review cycle 2 の N3 と N4(492 行目)

self-review の「Tech debt identified」が行を求めていたので、1 行にした(N3 を (a)、N4 を (b))。列は Debt、Impact、Why deferred、Trigger、Related の 5 つ。self-review の行から変えた所は次のとおり。

- 書き写さなかった所: self-review の行は `\|\|` を含んでいた。表の区切りと混ざるので、「`if !ok` の check の `grown && item.State != InboxStateOpen` の節」と言い換えた。「パイプラインの上限(cap 2)」の文も書いていない(上限は動かしうる)
- N3: 「M2 は結果の変わらない変異」と「doc が fast path と呼ぶ節で、cycle 2 の run は残した」(計画の Progress の cycle 2 self-review の行)を書いた。487 行目の (c)(d) の書き直しは、この pass で済んでいる
- N4: 判断は「債務としては弱い条件つき」。cycle 2 の cross-review の triage は `docs/reports/cross-review-triage-watch-usage-limited.md` に足し、番号を 3 から続けて振る(handoff と計画の Progress)。489 行目の「WORTH_CONSIDERING 2」と競合のテストのコメントの「finding 1」は外れない。そこで行の本文は「番号が外れるのは、triage を 1 から書き直す run があったとき」とし、Trigger に「triage を 1 から書き直す `/cross-review`、または 489 行目かそのコメントの次の編集」を置いた。489 行目と競合のテストのコメントは変えていない(コメントはコードの編集になる)。cross-review の skill は、triage を足すか書き直すかを書いていない(`git grep -i` で確認)ので、足す動きは handoff の説明に頼っている。cross-review が実際に足したあとに、番号が合っていることを見る

### I10: `inbox_test.go` の大きさ(493 行目)

行を足した。理由は、`inbox_test.go` が branch の前から 800 行を超えていた(1,517 行)うえで、この branch が 138 行を足したこと。同じ作りの行が register にあり(357、365、488 行目)、211 行目の note は数を書くだけで追う行ではなかった。内容は短く: 1,655 行(`34e5bd86` の `wc -l`)、`recurred` のテストと fixture が約 420 行(`alertEscalatedLine` から `TestInboxAppendRecurredIfOpen_ChecksTheStateUnderTheInboxLock` まで。`alertEscalatedLine`、`recurredLine`、`escalateTestAlert` はこの範囲の外から使われていない。`grep` で確かめた)、その block を `inbox_recurred_test.go` に移すと約 1,240 行で、それでも 800 行を超える。211 行目の (a) は `TestPrintableInboxText` をこの file へ移せと言うので、分けたあとは大きくならない側へ置く、と Trigger に書いた。

branch の diff で 800 行を超える file はほかに、skill の写し(org 2,336 行、director 2,042 行)、`watch.go`(2,246 行。357 行目)、`watch_usage_limit_test.go`(936 行。488 行目)、`internal/cli/org_watch.go`(883 行)。`org_watch.go` は origin/main で 875 行あり、この branch は help の文で 8 行足しただけなので、行にしていない(判断は lead の手元に残す)。skill の写しは cycle 1 の時点と同じで、足していない。

### test report の Test gaps と行の対応

| Test gaps(cycle 2) | 行 |
|----|----|
| 1(leader 以外だけが上限。probe は cycle 1 のまま) | 487 の (a)(b) |
| 2(pending の分岐) | 487 の (c) |
| 3(cycle 2 で書き換え。受信箱が読めない枝と `recur` の失敗の枝) | 487 の (c)(書き直した) |
| 4(`withUsageLimitedLines`、`cutUsageLimitText`) | 487 の (c) |
| 5(cycle 2 で書き換え。M3 は 2 本、直し自体の M6 は競合のテスト 1 本、`openOnly` の行は変異なし) | 487 の (d)(書き直した) |
| 6(`watchEscalationReason` の順) | 487 の (d) |
| 7(L5 と N1) | 485、486 |
| 8(実機の画面) | 452 |
| 9(AC9 の文面) | 行にしない(cycle 1 と同じ判断。verify が文書を読んで確かめた) |
| 10(cycle 2 で足した。M2 は結果の変わらない変異) | 492 の (a)、487 の (d) |

### 確かめたこと(合っていた所)

- `agent-messaging.md` と `templates/base` の写し、README、AGENTS.md の `internal/org/` の行、quality-gates、org と director の skill、help: cycle 1 の確認から変わらず、実装と合う。`AppendRecurredIfOpen`、`ErrInboxAcked`、`beforeAppendRecurred` の名前は、`internal/org` と `docs/tech-debt/README.md` 以外の文書に 0 件(`git grep -c -F -f`)なので、足す所は無い(verify の I8 と同じ)
- 486 行目と 489 行目は、この pass で変えていない(`tdcheck2.py` の変わった行に出ない)

### 残っていること(lead へ)

- コードとテストは変えていない。この commit は仕様の 1 文、tech-debt の register、計画の Progress の 1 行、この report と insights の行だけで、パイプラインの再走は要らない
- 払っていないもの: 持ち越しの note を足した 485、487 行目の中身(L5、Test gaps)、新しい 492 行目の N3(コードの形)と N4(triage の番号)。PR 本文の Known gaps には、485(L5)、486(N1)、489(WORTH_CONSIDERING 2)、492(N3、N4)の 1 行ずつを並べる
- 487 行目の (b)(leader が上限の座席を待つとき、leader の stall と usage_limited の 2 件ができる)は製品の判断で、cycle 1 から決まっていない
- cycle 2 の cross-review が triage を 1 から書き直したら、489 行目と 492 行目と競合のテストのコメントの番号が外れる。足す形なら外れない
- origin/main は `bb416c48` まで進み、merge-base と同じ。`docs/tech-debt/README.md` は両側が末尾に行を足すので、この branch の 492、493 行目が次の merge で衝突しうる(verify の I11 と同じ)
- 実機の上限の画面は見ていない(452 行目)
- 下流への注意は cycle 1 のまま(org skill と director skill の写しは `ralph upgrade` で配られる。この pass は skill を変えていない)

### Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck2.py`)で、変わった行は 211、357、462、485、487、488 と、末尾の追加 2 行(492、493)。どの行も列は 5、バッククォートは偶数、`~~` は偶数、行は `| ` で始まり ` |` で終わる。211 と 485 は列の末尾への追記だけ。357、462、487、488 は途中の編集で、`difflib` の置き換えを読んで、意図した文字列(`461`→`470`、`e25def99`→`193b044a`、「Test gaps 1 to 6」の言い直しなど)だけが置き換わっていることを確かめた
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` は一意に 124 件で、無い参照は 0 件(cycle 1 は 122 件)。`verify.local.sh` の `check_tech_debt_plan_refs` と同じ `grep` と `sed` を使った。変わった行と新しい行の `.go:NNN` の形は 0 件。U+FFFD は README に 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-watch-usage-limited.md` は編集の前後とも `f2971e9c4a70`。`git diff --numstat` は計画 1/0、仕様 1/1、tech-debt 8/6(既存の 6 行の編集と 2 行の追加)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`(rc 0)。`./scripts/check-sync.sh`: `PASS: all files in sync.`(rc 0)。`git diff --check`: rc 0
- Go のテストは回していない。この commit は Go のコード、テスト、skill、rule を変えていない。`*_test.go` で `docs/specs`、`docs/tech-debt` を読むものは無い(`git grep` で出たのはコメントと失敗メッセージの文字だけ)ので、文書の編集で結果が変わるテストが無い。cycle 2 の test report の通りが、コードの状態の確認になる
- 実行していないもの: `run-verify.sh` の全体(コードを変えていない)

### Insight event

```
./scripts/insights-append.sh --slug watch-usage-limited --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

rc 0。`docs/insights/events/2026-11-13-watch-usage-limited.jsonl` が 6 行から 7 行になった(`ts` は `2026-11-13T04:46:19Z`、`phase` は `sync_docs`、`cycle` は 2、`verdict` は `pass`)。UTC の日付が cycle 1 の run と同じ 2026-11-13 なので、新しい file はできていない。`git status --short` は既存の 4 つの file の `M` だけで、`<date>-<date>-` 形の余分なファイルは出ていない。
