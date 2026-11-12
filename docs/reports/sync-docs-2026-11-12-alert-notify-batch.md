# sync-docs report: alert-notify-batch

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-alert-notify-batch.md`(承認の digest `a805c3b2b4b4`)。issue #307 の後半
- Branch: `fix/alert-notify-batch`。base は origin/main `94e61c38`、run 開始時の HEAD は `66bc9bd1`(test の report のコミット)
- Pipeline run: 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は `cycle` 1)
- 先行 report: self-review `59961e6a`(2 回目の run。Merge: yes、指摘なし)、verify `34becddd`(pass。LOW 3 件 V1〜V3、INFO 8 件 V4〜V11)、test `66bc9bd1`(pass。Test gaps 4 件)
- 制約: 計画の本文は変えていない。変えたのは AC のチェックボックスと `## Progress checklist` だけ。`./scripts/plan-visual.sh digest` は編集の前後とも `a805c3b2b4b4` で、`- Approved:` の行と同じ
- コードを変えた: `internal/cli/org_watch.go` の help の文字列、`internal/cli/org_watch_test.go` の断片 1 つ、`internal/config/config.go` の doc comment(V4)。挙動は変わらない。再走の要否は lead が決める

## Summary

verify の V1〜V6、V8、V9 を処理した。V7、V10、V11 は変えない。tech-debt は、この branch が Trigger を引いて払わなかった 11 行(依頼の 10 行と、自分の編集で引いた 118 行)に引き継ぎの注を足し、「Eleven gaps」の行(462 行)には D4 の新しい窓 W10 を (9) に足した。(4) は RESOLVED のまま。テストの穴 3 件は新しい行(463 行)にした。V4 の 1 句は help、`alert_notify` の config doc とテンプレートのコメント、director skill(4 つの写し)に足した。仕様の注、quality-gates(root と template)、README にも 1 句ずつ足した。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `internal/cli/org_watch.go`、`internal/cli/org_watch_test.go`、`internal/config/config.go`、`templates/base/ralph.toml` | V4。段が 1 件だけを送るときは、その件の文のままであることを 1 句ずつ。help の断片は `TestOrgWatch_HelpDescribesTheAlertNotifyBatch` に足した | `f1200678` |
| `.claude/skills/director/SKILL.md` と、その `.agents/` と `templates/base/` の 3 つの写し | V4。director skill の見張りの通知の段に「(1 件だけの段は、その件の文のまま)」。`sync-skills.sh` で `.agents` を作り、`templates/base` の 2 つは `cp` | `f1200678` |
| `docs/tech-debt/README.md` | 12 行の変更(118、258、262、303、319、355、356、357、381、415、462 と、新しい 463 行)。下の「tech-debt」 | `b404b929` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | V5。FR-6 の 2026-11-12(#307 の後半)の注の 1 文 | `b404b929` |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` | V6。Watchdog pulse layer の行に 1 句(同じ文) | `b404b929` |
| `README.md` | 2 か所。コマンド表の `wait --inbox` に `--types`、watch の段落に 1 段ごとの通知と `alert_notify` | `b404b929` |
| `docs/plans/active/2026-11-12-alert-notify-batch.md` | V8。AC1〜AC10、AC1b、AC1c の箱を `[x]`。Progress は verify、test、sync-docs の 3 行を足し、artifact の 3 つの箱を `[x]`。「PR created」は `[ ]` のまま | この report のコミット |
| `docs/reports/sync-docs-2026-11-12-alert-notify-batch.md` | この report | この report のコミット |
| `docs/insights/events/2026-11-12-alert-notify-batch.jsonl` | `sync_docs` の行(cycle 1、pass)。ファイル名の日付は UTC(この run は 2026-11-12 08:32 UTC) | この report のコミット |

## 依頼の項目

| 項 | 結果 |
|----|------|
| 1. V1(462 行) | Trigger の末尾に 2026-11-12 の注を足した。行の Trigger と、(9)〜(11) の名前での当たり((9)(10) は `escalateToInbox` の引数と「recorded」の枝の呼び出し、`reconcilePendingAlertsWithInbox` の nil の batch、(11) は `escalateAlert` の doc)を書き、(4) を RESOLVED のまま、(1)〜(3) と (5)〜(11) を払っていないと書いた。(5)(6)(8) は当たっていない(`watch_scope.go`、`leaderActedSinceAlert`、`foldIntoPendingAlert` は触っていない)。W10 は (9) に足した(下)。Debt、Impact、Why deferred、Trigger の (9)、Related に同じ窓の言い方を足した。見出しは「Eleven gaps」のまま(窓は (9) の中の項目で、新しい項目ではない) |
| 2. V2(319 行)と V3 | 319 行は、Trigger の 2 つめの条件(`director_wait.go` が 800 を越える)を満たした。799 → 821 行と、`director_wait_test.go` の 687 → 824 行を、origin/main `94e61c38` と `1f2197dc` で数えて書き、払っていない理由(move-only の分割は Scope の外、行のほかのファイルは触っていない)と、Debt の列の数が古くなったことを書いた。V3 の 357、356、355、262、258、381、303、415 の 8 行にも同じ形の注。357 の行数は `1f2197dc` のもの(`watch.go` 2139、`watch_test.go` 4463、`watch_all_test.go` 2613、`watch_recurrence.go` 275、`watch_scope.go` 467) |
| 3. テストの穴 | 463 行を新しい行にした(別の PR の穴なので、462 行の項目にはしなかった。262 行などの前例と同じ形)。(1) AC1b の director の半分(M1、M2 が org 全体で生き残る。probe の中身)、(2) `off` での周期の初めの再送(nil の batch)、(3) 1 周期に掃き出しと deadman の段を通すテストがない。Trigger は穴ごと、Related に test と verify の report。Test gaps 4(AC9 の skill の文)は行にしていない(下) |
| 4. V4 | 「1 件だけの段はその件の文のまま」を、help(`TestOrgWatch_HelpDescribesTheAlertNotifyBatch` に断片を足した)、config の doc、テンプレート、director skill の 4 つの写しに 1 句ずつ。org skill と仕様の注は元から書き分けているので変えていない。`go test ./internal/cli/ -run Watch`(ok、58.9 秒)、`go test ./internal/config/`(ok)、`go test ./internal/org/ -run Skill`(ok) |
| 5. V5 | 仕様の注に「記録と同時に送る見張り(director が無効、`--once`、止める頼みが出ている)が立て直したときは、deadman の段の件は周期の初めの突き合わせがそれより早くその場で送る」を足した。計画の D4 と AC5 は本文を変えず、Progress の sync-docs の行に記録した |
| 6. V6 | quality-gates の 83 行(template は 82 行)の最後の列に「the ALERT items of one deadline sweep or of one org's deadman step share one notification, none with `[org.watchdog].alert_notify = "off"`」。2 つの写しは同じ文。この編集は 118 行の Trigger を引いたので、118 行にも注を足した(下) |
| 7. V8 | 12 個の AC の箱を `[x]` にした(`git diff --numstat` は 18 追加、15 削除。15 は 12 + artifact の 3、追加の 3 は Progress の 3 行)。digest は変わらない |
| 8. V11 | 変えない(PR 本文で D5 を書く) |

V9(462 行の Related が AC1〜AC6 を挙げない)も直した。「AC1 to AC10, which close (4); D4 is the window of W10」にし、verify と test の report を足した。V7(AGENTS.md は org のファイルを全部は挙げない)と V10(help の `While an ALERT for the same org, seat, and` の行が短い。見た目だけ)は変えていない。

## tech-debt

行は先頭の語で見つけた。`git show HEAD:` との比較で、変わった行は 118、258、262、303、319、355、356、357、381、415、462 と、新しい 463 行だけ(行数は 463 から 464)。どの行も列の数は 5 で、列ごとのバッククォートと `~~` が偶数、新しい `file:line` が 0、新しい `\|` が 0 であることを `tdcheck.py` で確かめた。plan 参照は 116 件で、無い参照が 0 件。

### W10(D4 の窓)

まとまりの送りに入れた ALERT の件は、行と banner が先に出て、通知と `notified` は段の終わりの flush で出る。flush の前に見張りが止まると、件は通知も `notified` も持たない。次の見張りの動きは段で分かれる。掃き出しの件はすでに期限を過ぎているので、次の見張りの最初の掃き出しが送る。deadman の段の件は状態ファイルが保存されていないので、ack の期限に送られるか、記録と同時に送る見張りなら周期の初めの突き合わせがその場で送る(V5)。どちらも行と banner が 2 回、通知は遅れ、失われない(知らせる側)。flush の通知と `notified` の追記の間で止まるのは W6。テストは `TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline` が flush されない件の ack の期限の送りを固定するだけで、見張りを止めて立て直すテストはない。閉じるには、通知より先に `notified` を書くか(止まる境目が黙る側に移る)、まとめをやめるしかないので、計画のとおり受け入れた。

### 掃き出し(この branch の名前で Trigger を読み直した)

Trigger の欄を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、全体が `~~` の行は飛ばす)、変えた関数とファイルの名前(`SweepInboxDeadlines`、`sendInboxItemToHuman`、`escalateAlert`、`escalateToInbox`、`WaitInbox`、`openInboxItems`、`validateLoaded`、`deadline.go`、`director_wait.go`、`config.go`、`watch_test.go`、`SKILL.md`、`quality-gates.md` ほか)で当たりを出し、1 行ずつ Trigger を読んだ。

| 行(先頭の語) | 判断 |
|----|----|
| 462、357、356、355、262、258、381、303、319、415 | 当たった。払っていない。注を足した(依頼の 10 行) |
| 118(template の quality-gates の「次のタッチ」) | 依頼に無かったが当たる。この sync-docs が template の quality-gates に 1 句足したため。「Must pass in CI」と verify chain は触っていない(`check-coverage.sh` と `check-pipeline-sync.sh` は template の workflow と `run-verify.sh` に無い)。注を足した |
| 322(`director_wait.go` の文) | 当たったが、行の求める「触った block ごとのテスト」を満たす。新しい `directorWaitTimeoutDetail` に `TestDirectorWaitTimeoutDetail` がある。注なし |
| 358(`sendInboxItemToHuman` の「6 つめの string の引数か 5 つめの呼び出し元」) | 当たらない。7 つめの引数 `batch` は string でなく、本番の呼び出し元は 4 か所のまま。Debt の列の「string 5 つと banner の `io.Writer`」には `batch` が足りないが、Trigger は引いていない |
| 453(`[org]` の検査) | 当たらない。足した検査は `validateLoaded` の `[org.watchdog]` の塊で、`api_wait_minutes` の `[org]` の塊ではない |
| 117、210、211 (a)、353、380、401 | 当たらない。117 は restructure でない、210 は `evaluateCycle` の打ち切り方でない(flush は 1 段に 1 回の osascript で、herdr と git の呼び出しは増えない)、211 (a) は `TestPrintableInboxText` と `inbox_test.go` を編集していない、353 は `sendAlert` を触っていない、380 と 401 は skill の別の項と README の別の段落 |
| 134、206、209、213、215、335、346、370、376、409 | `~~` で閉じた Trigger |
| 88、148、150、151、190、191、275、382、400、404、414、425、429、431 | 語を拾っただけ(`ralph.toml`、`defaults_sync_test.go` が別のファイル名の中、`config_test.go`)。Trigger は別の変更を指す |
| 177、195、201、239、427 | 語を拾っただけ(`org.go`、仕様の名前)。diff の `org.go` は `newOrgWaitCmd` と `runOrgWaitInbox` だけで、Trigger の関数(`newOrgStartCmd`、`runOrgStartPlan`、`signalCauseContext`)は編集していない |

## 確かめたこと

- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 本が lock-step、`./scripts/check-template-purity.sh` は PASS。skill の 4 つの写し(director)は `cp` と `sync-skills.sh` のあとの `check-sync.sh` で一致を確かめた。`f1200678` の前と `b404b929` の前に打ち、この report のコミットの前にも打ち直した(結果は同じ)
- `gofmt -l internal/cli internal/config` は空、`go vet ./internal/cli/ ./internal/config/` は ok
- 計画の digest は 3 回(編集の前、AC の箱のあと、Progress のあと)とも `a805c3b2b4b4`
- 変えたファイルの追加行に U+2028・U+2029・U+FFFD・U+200B は 0 件(`f1200678` と `b404b929` の diff を `LC_ALL=C grep -c` で数えた)

## 残っていること(lead へ)

- `f1200678` はコードの変更(help の文字列、doc comment、テストの断片)を含む。挙動は変わらず、help の断片は `TestOrgWatch_HelpDescribesTheAlertNotifyBatch` が固定する。cap 2 のうち 1 回目の run の中なので、再走の要否は lead が決める
- 払っていないもの: 462 行の (1)〜(3)、(5)〜(11)、463 行の 3 つの穴。どれもコードかテストの変更
- `/pr` の Known gaps に写すもの: W10(行と banner が 2 回、通知が遅れる)、463 行の 3 つ、AC9 の skill の文を固定するテストがないこと(Test gaps 4。文書の範囲として tech-debt にしていない)
- PR 本文に書くもの: D5(director の待ちが既定のままの理由)、V5 の言い足り(計画の D4、AC5 は本文を変えていない)
- 358 行の Debt の列は `batch` を数えていない(Trigger は引いていない)。次に 358 行を触る人が直せばよい

## Insight event

```
./scripts/insights-append.sh --slug alert-notify-batch --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-alert-notify-batch.jsonl` が 4 行から 5 行になり、足した行は cycle 1、pass、ts は 2026-11-12T08:32:24Z。ファイルの日付は UTC で、`date -u` は 2026-11-12 だった。`git status --short` は既存の jsonl と計画の `M`、この report の `??` だけで、迷子のファイルはない。
