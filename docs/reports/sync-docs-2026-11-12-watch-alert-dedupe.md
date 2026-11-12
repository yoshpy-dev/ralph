# sync-docs report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(承認の digest `e5a18ef9477a`)。issue #307 の前半
- Branch: `fix/watch-alert-dedupe`。base は origin/main `028a8f6d`、run 開始時の HEAD は `a33eb504`(run 4 の test の report のコミット)
- Pipeline run: run 4 (cycle-count 3)。`.harness/state/standard-pipeline/cycle-count.json` は `cycle` 3、`cap` 3 のままで、cap を 3 に上げたとき cycle を進めなかったため、run の数と cycle の数が 1 ずれる。lead の指示どおり insight event は `--cycle 4` で足した。cap 3 が許す最後の run
- 先行 report(run 4): self-review `abf95ceb`(Merge: yes、LOW 3 件 C4-1〜C4-3、止まる窓の一覧 W1〜W9)、verify `b305064a`(pass。LOW 3 件 V18〜V20、INFO 5 件 I15〜I19)、test `a33eb504`(pass。Test gaps 14〜16 は INFO)
- 制約: 計画の本文は変えていない。変えたのは `## Progress checklist` だけ(S12 の行の 2 か所を直し、run 4 の 4 行を足した)。`./scripts/plan-visual.sh digest` は編集の前後とも `e5a18ef9477a` で、`- Approved:` の行と同じ
- コードは変えていない。変えたのは Markdown だけ
- この report は run 4 で上書きした。cycle 3 までの run の内容は末尾に短く残した

## Summary

verify の V18〜V20 と、任意の I15、I17 を処理した。tech-debt の既知の穴の行(462 行)は「Eleven gaps」になり、止まる窓の一覧(W1〜W9、書き込みの境目、再起動のあとの向き、受け持ち)を (9) に、判定の重複(C4-1)を (10) に、comment のずれ(C4-2)を (11) に足した。S12 の周期の初めの reconcile は (9) の中に書いた。C4-3(運んだ回数のあとで止まると回数が 1 多い)は (9) の W3 と W4 にある。run 4 の引き継ぎの文を足し、「the pipeline's last run」を run の数で言い直した。357 行と 303 行には S12 の当たりを足し、357 行の行数は最後の Go の commit `14bc17a3` のものに直した。仕様の FR-6 の注には、周期の初めに受信箱がすでに持つ pending を片付けることを 1 文足した。計画の Progress は、S12 の行の 2 つの言い過ぎを直し、run 4 の 4 行を足した。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/tech-debt/README.md` | 3 行の変更(303、357、462)。下の「tech-debt」 | `abc0e4d4` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | (I15)116 行の FR-6 の 2026-11-12 の注の 1 行だけ。「状態ファイルに続きの記録が無いとき…`recurred` を足す。」のあとに、見張りが件を記録したあと状態ファイルを保存する前に止まっても、次の見張りは周期の初め(座席を見る前)に、受信箱がすでに持つ pending を escalate の再送の判定に通して消すので、件が resolved になったあとの発火は新しい ALERT になる、という 1 文を足した | `abc0e4d4` |
| `docs/plans/active/2026-11-12-watch-alert-dedupe.md` | (V20)Progress だけ。S12 の行の 2 か所を直し(下の項 3)、その後ろに run 4 の self-review(`abf95ceb`)、verify(`b305064a`)、test(`a33eb504`)、この sync-docs の 4 行を足した。「PR created」の箱は元の位置 | この report のコミット |
| `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` | この report | この report のコミット |
| `docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` | `sync_docs` の行(cycle 4、pass)を `insights-append.sh --slug watch-alert-dedupe --cycle 4` で足した。ファイル名の日付は UTC(この run は 2026-11-12 04:23 UTC)。15 行から 16 行になった | この report のコミット |

## 依頼の項目

| 項 | 結果 |
|----|------|
| 1. V18(462 行) | 見出しを「Eleven gaps」にした(「Eight gaps」と呼ぶ report は当時の名前のまま)。(9) は止まる窓の W1〜W9。W1 `sendAlert` のあと(二重、この PR より前から)、W2 件を書いて運ぶ回数を書く前((2))、W3 運んだ回数のあと、W4 `escalateAsRecurrence` のあと(どちらも回数が 1 多い、C4-3)、W5 `recordRecurrence` のあと((7))、W6 件と通知のあと `notified` の前(二重、D4 の replay)、W7 `escalateWithoutInbox` のあと(二重、D5)、W8 pending を消すだけの境目(対応不要)、W9 周期の初めの読みだけが失敗(黙る向き。黙るのはこれだけで、極小)。受け持ちは、S12 のテスト、(2)、(7)、2026-10-23 の D4 と D5、「テストなし」で書き分けた。S12 の reconcile(`reconcilePendingAlertsWithInbox` を `evaluateCycle` が座席の前に呼ぶ)は、W2〜W4 の黙る向きを閉じたことと、受信箱が持たない pending と運ぶ回数は前のままであることを書いた。(10) は C4-1、(11) は C4-2 の 2 か所。Impact と Why deferred の欄にも (9)〜(11) を足した。Trigger の欄は (9)〜(11) の直し方と、run 4 の引き継ぎの文(S12 が行の Trigger と (8) の Trigger に名前で当たったこと、(5) と (6) には当たっていないこと、(1)〜(3) と (5)〜(8) は払っていないこと)を、「Carried over」の前に入れた。Related は S12、`evaluateCycle`、`escalateAlert`、`reconcilePendingAlertsWithInbox`、self-review の `abf95ceb`、verify の `b305064a`、test の `a33eb504`(Test gaps 14〜16)を足した。(8) の Why deferred の「cycle 3 of 3」の括弧は、4 回目の run を足す形に直し、Trigger の cycle 3 の文の「after the pipeline's last run」は「after the pipeline's third run」にした。(6) の「after the pipeline's last run」(cycle 1 の一般則)と cycle 2 の文のもの(同じ文が cap 2 を使い切ったことを説明している)は、そのままにした |
| 2. V19(357 行) | S12 の当たりを 1 文足した(`escalateAlert` の 2 つめの呼び出し元、doc の変更、テスト 4 本)。「At e90c85da」を「At 14bc17a3」にし、`watch.go` 2090、`watch_test.go` 4463、`watch_recurrence.go` 273、`watch_scope.go` 467 に直した。4 つとも `git show 14bc17a3:<file> \| wc -l` と HEAD の `wc -l` が同じ。`watch_scope.go` は `e90c85da` から変わっていない(`git diff --stat e90c85da 14bc17a3` は 3 ファイル、`watch_scope.go` を含まない) |
| 3. V20(計画の Progress) | 上の表のとおり。S12 の行の直し 2 か所: 「受信箱が読めないときは何もしない(知らせる側)」に W9 の狭い黙る向きを加えた。「消す pending が持っていた回数は失われる」は、失われるのは W2 だけで、W3 と W4 では運んだ回数は残り件の回数が 1 多くなる(C4-3)に直した。足した 4 行は、時の順で S12 の行のすぐ後ろ。`git diff -U0` の hunk は 177 行の 1 行変更と 4 行の追加の 1 つだけで、digest は変わらない |
| 4. I15(任意) | 足した。表のとおり、仕様の FR-6 の注の 1 文。2026-10-23 の注の「同じ alert_id の件がもう受信箱にあれば…」は S12 のあとも正しいので、そのままにした |
| 5. I17(任意)、I18、I19 | I17 は 303 行に足した(4 テスト、275 行、`watch_test.go` 4463 行、`newTestWatchRun(` を含む行は 30 のまま、新しいテストは `newWatchInboxFixture` を使う)。確かめ: `git diff e90c85da 14bc17a3 -- internal/org/watch_test.go` の追加行に `newTestWatchRun(` は 0、`newWatchInboxFixture(` は 4。I18(手で書き換えた key `""` の pending は周期の初めに受信箱の外へ escalate される)と I19(triage の rationale)は記録のまま。I18 は手の書き換えでしか起きず、向きは知らせる側。I19 は /cross-review の記録で、変えない |

## tech-debt

行は先頭の語で見つけた(行番号は `a33eb504` のもの)。`git show HEAD:` との比較で、変わった行は 303、357、462 の 3 行だけ(行数は 463 のまま)。3 行とも列の数は 5 で、列ごとのバッククォートと `~~` が偶数、新しい `file:line` が 0、U+2028・U+2029・U+FFFD が 0 件であることをスクリプトで確かめた。462 行は 11,298 バイトから 19,993 バイトになった(Debt の列は 3,783 から 8,241)。tech-debt の plan 参照は、無い参照が 0 件(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ式を直接打った)。

### 掃き出し(S12 の名前で Trigger を読み直した)

Trigger の欄を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、全体が `~~` の行は飛ばす)、S12 が触った名前(`watch.go`、`watch_test.go`、`watch_recurrence`、`evaluateCycle`、`escalateAlert`、`escalateToInbox`、`reconcilePendingAlertsWithInbox`、`foldIntoPendingAlert`、`checkDeadman`、`PendingAlerts`、`status file`、`recurrence`、`deadman`、`escalation`、`newWatchInboxFixture`、`escalateRecordedItem`、`inbox_test.go`)と、この sync-docs が編集するファイル(tech-debt、仕様、計画)の名前で照合した。hunk は `git diff e90c85da 14bc17a3 -- internal/org/watch.go internal/org/watch_recurrence.go` を読んだ(`watch.go` は `evaluateCycle` の doc と呼び出し 1 行、`escalateAlert` の doc、`watch_recurrence.go` は新しい関数と `foldIntoPendingAlert` の doc 3 行)。

| 行(先頭の語) | 判断 |
|----|----|
| 462(Eleven gaps) | 行全体の Trigger に当たり、(8) の Trigger に名前で当たった。項 1 |
| 357(`watch.go` と `watch_test.go` の大きさ) | 当たった(escalation のコードとテストの次の変更)。払っていない。項 2 |
| 303(`newTestWatchRun`) | 当たった(`watch_test.go` の次の変更)。払っていない。項 5 |
| 210(止める頼みが周期の終わりまで待つ) | 当たらない。Trigger は `evaluateCycle` の呼び出しの打ち切り方の変更。reconcile は herdr、agmsg、git を打たず、受信箱のファイルを 1 回読むだけで、replay の人への送りは `checkDeadman` の中と同じ道を通る。Debt の git の数は変わらない |
| 457(`ralph org resume` の変異) | 当たらない。(f) の Trigger は `leaderActivityEventCount` で、S12 は触っていない。`foldIntoPendingAlert` の doc の 3 行は注の中で名前が出るだけ |
| 353(`sendAlert`)、355 と 258(`InboxDeadlineDue`、`SweepInboxDeadlines`)、262(`SweepInboxDeadlines`、`RunWatchAll`) | 当たらない。S12 の hunk にその関数は無い |
| 179、211(a)、180、118 | 当たらない。179 は `Stop` などの変更、211 (a) は `inbox_test.go` の編集、180 と 118 は `/org` skill の表と template の quality-gates の編集が Trigger で、この run はどれも編集していない |
| 133、335、346、370 | `~~` で閉じた Trigger(または RESOLVED) |
| 24、100、177、338、365、371、401 | 語を拾っただけ(`recurrence`、`tech-debt`、仕様の名前、`deadman`、`inbox_test.go`、`README.md`)。Trigger は別の変更を指す |

`cap 2 of`、`cap of 2 runs`、`cycle 3 of 3` で 462 行を grep すると、残るのは cycle 2 の文の「the cross-review findings spent the cap of 2 runs」(事実の文)だけで、「cycle 3 of 3」は 0 件。

## 確かめたこと

- `./scripts/check-sync.sh` は `PASS: all files in sync.`(ROOT_ONLY 0、KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 本が lock-step。`abc0e4d4` の前に打った。skill、template、script は変えていない
- 計画の digest は 3 回(最初、tech-debt と仕様のコミットのあと、Progress を直したあと)とも `e5a18ef9477a`
- README、AGENTS.md、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその写し、`/org` と `/director` の skill、`ralph org watch --help`、agent-messaging は、周期の初めの reconcile を書いていない。どれも replay がいつ走るかを書かない要約で、S12 のあとも正しい(verify の I16)。直したものはない
- 変えたファイルに U+2028・U+2029・U+FFFD は 0 件(tech-debt はスクリプト、そのほかは hook の検査)

## 残っていること(lead へ)

- 払っていないもの: (1)〜(3)、(5)〜(8) の引き継ぎの項目と、新しい (9)〜(11)。run 4 は cap 3 が許す最後の run で、直すと comment とコードの変更になり、パイプラインの回し直しになる
- C4-1 と C4-2 を直す場合(判定を 1 つの関数に寄せる、comment 2 か所)は full re-run。直さないなら tech-debt の (10)、(11) が記録になる
- `/pr` の Known gaps に写すもの: W1・W6・W7(二重)、W3〜W5(回数が 1 多い)、W9(黙る向き、極小)、C4-1、C4-2。tech-debt の (9)〜(11) が出典
- `/director` skill の ALERT の件の項は、件に回数が付くことを書かない。#307b で直す(計画の Non-goals)
- 実機での件の数え直しと、止まる窓を実際に止めて立て直す確かめは、verify と test のとおり未実施のまま

## 過去の run の記録

cycle 1 の run(HEAD `3506f99a`、コミット `45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)は verify の V1〜V8、self-review の N5、test の M30・M50 を処理した。`/org` skill(4 つの写し)、README の 2 か所、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその template の写しに、今回の挙動を足した。tech-debt は既知の穴の行を 4 つから 6 つに増やし、Trigger に当たった 9 行に引き継ぎの注か数を足した。全文は `git show 2bebc1ac:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

cycle 2 の run(HEAD `fe4eff60`、コミット `0cc32e6d`、`74e6b1db`、report は `cd4a9268`)は verify の V9〜V14 と test の C24・C04 を処理した。仕様の FR-3 の注と tech-debt の RC3 の注に `--cc` と origin の既定のブランチの決め方を足し、既知の穴の行を「Eight gaps」にして (7) と (8) を足した。`/org` skill(4 つの写し)に、`recurred` の event に載った alert_id の escalation の打ち直しは何も書かず送らない、という 1 文を足した。全文は `git show cd4a9268:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

cycle 3 の run(HEAD `24ca3804`、コミット `baee33db`、report は `ad5beef6`)は verify の V15〜V17 と I11 を処理した。462 行の「cap 2 of 2」と「cap of 2 runs」を cap 3 に合わせて直し、S10 と S11 の当たりを 1 文足した。357 行の行数は最後の Go の commit `e90c85da` のものにした。仕様の FR-3 の注と RC3 の注に、commit のパスは `-c diff.relative=false` で root からの相対になることを足した。計画の Progress の cycle の行を時の順に並べ直した。全文は `git show ad5beef6:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

## Insight event

```
./scripts/insights-append.sh --slug watch-alert-dedupe --flow standard --phase sync_docs \
  --verdict pass --cycle 4 --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` が 15 行から 16 行になり、足した行は cycle 4。ファイルの日付は UTC(この run は 2026-11-12 04:23 UTC)。`git status --short` は既存の jsonl と計画の `M` だけで、迷子のファイルはない。
