# Verify report: org-watch-stop-failed

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-watch-stop-failed.md
- Verifier: verifier subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC5)、静的解析、文書のずれ。対象は `git diff da4dccb0...HEAD`(HEAD 1b609add、6 ファイル、+330/-10)。コードの変更は 2765f002(`internal/org/watch.go`、`internal/org/watch_test.go`、`docs/tech-debt/README.md`)と 3b219fc7(doc comment と tech-debt の HTML コメントの直し)の 2 つ。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-10-08-org-watch-stop-failed.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-08-041059.log`

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `53d656a5e19f` を返し、plan の `- Approved: 2026-10-08 sha256:53d656a5e19f` と一致した。

直す対象: `docs/reports/cross-review-triage-org-stop-all.md` の cycle 2 の ACTION_REQUIRED #1(`:26`)は、閉じられなかった stop が `stop_failed` だけを書き、`leaderActivityEventCount` がそれを数えない点を挙げている。この PR で `internal/org/watch.go:842` の `case` に `EventStopFailed` が入り、下の AC1〜AC3 のテストが足された。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `stop_failed` を 1 件と数え、`reason=watchdog_` つきは数えない | Met | `watch.go:842` の `case` に `EventStopFailed` があり、`:843` の `reason=watchdog_` の判定と `:834` の orgID の絞り込みをほかの種類と共有する。単体テスト `TestLeaderActivityEventCount_StopFailed`(`watch_test.go:1471`)の 3 件(`:1480-1482`)が、その org の分は数える、`reason=watchdog_` つきは数えない、別 org の分は数えない、の 3 点を固定する |
| AC2: 警告のあとの `stop_failed` で警告が消え、人に上げない | Met(テストの中身とコードで確認。実行は /test) | `TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert`(`watch_test.go:1380`)は `evaluateCycle` を 2 回回す。間の `stop_failed` は、`PaneGetErr`(`:39-42`)で確認の get を失敗させた本物の `Stop` が書き、setup でちょうど 1 件あることを assert する。cycle 2 のあとで escalation がなく pending が 0 であることを見る(`:1412-1415`)。leader の `AgentGet` は既定の `"ok"` を返し続け(`:142`)、agmsg の history は `HistorySeq` が空なので `""` のまま(`:250-260`)。このため、警告を消せるのは台帳の数え直し(`watch.go:998`)だけになる。`EventStopFailed` を `case` から外すとこのテストが落ちることは、コードからそう読めるが、mutation は再現していない |
| AC3: 別 org の `stop_failed` では消えない | Met(同上) | `TestWatch_Deadman_CrossOrgStopFailedEvent_DoesNotClearPendingAlert`(`watch_test.go:1423`)は、org-b の `Stop` が `stop_failed` を書いたことを assert してから、org-a の警告が 1 件 escalate されることを見る(`:1461-1464`) |
| AC4: doc comment の (b) の一覧に `stop_failed`、互換の除外が要らない理由、tech-debt の行が解決済み | Met(理由の書き方が AC の括弧書きと違う。V-3) | 一覧は `watch.go:784`。書き手と、watchdog が書かないことは `:790-796`。除外が `case` を共有するので `stop_failed` にも掛かることは `:816-818`。互換の guard が要らない理由(v5.1.0 までは書かない)と、残る窓は `:818-825`。tech-debt は `docs/tech-debt/README.md:167`(HTML コメント)と `:168`(取り消し線の行)で、既存の RESOLVED の行(`:16-17`、`:25-26` など)と同じ形 |
| AC5: 今ある watch のテストがすべて通る | 静的には合う。実行は /test | `fakeWatchHerdr.PaneGet` は `PaneGetErr` が nil のとき、変更前と同じ not-found を返す(`watch_test.go:90-97`)。`PaneGetErr` を設定するのは新しい 2 本だけ(`:1400`、`:1449`)。`go vet ./...` はテストファイルを含めて通った |

書き手の確認: `EventStopFailed` を書くのは `Stop` だけ(`internal/org/verbs.go:918-921`)。`Stop` の呼び出し元は `internal/cli/org.go:768`(`stop`)、`verbs.go:1716`(`Disband`)、`:1754`(`disbandOwnLast`)、`internal/org/verbs_all.go:138` と `:145`(`StopAll`)。`DisbandAll` は `Disband` を通る(`verbs_all.go:279`、`:286`)。`watch.go` は `Stop` を呼ばない。doc comment の「`stop` と `disband`(と `--all`)が書き、watchdog は書かない」は合っている。

配布状況の確認: `git grep -n 'stop_failed\|StopFailed' v5.1.0 -- internal` は 0 件。`git ls-remote --tags origin` の最新は `v5.1.0` で、origin の main は da4dccb0(#208 のマージ)。#208 を含む release はまだない。手元の `ralph` は Homebrew の 5.1.0。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed) | rc 0 | 言語は golang だけが選ばれた |
| shellcheck、`sh -n`(hooks 20 本)、`jq -e`(settings.json 2 本) | OK | この PR は shell を変えていない |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0 |
| `check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | OK | 13 skill が一致 |
| tech-debt README plan references | OK | `:168` が指す `docs/plans/active/2026-10-08-org-watch-stop-failed.md` は今あるので通る。`/pr` の `scripts/archive-plan.sh` が `archive/` に書き換える(`.claude/skills/pr/SKILL.md:54`) |
| gofmt、go vet、golangci-lint、staticcheck | `gofmt: ok`、出力なし、`0 issues.`、出力なし | 4 つとも成功 |
| `secret-scan-branch.sh`(runner の中) | clean | da4dccb0..1b609add |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 数える種類の一覧(`git grep -n 'spawned, spawn_started, stopped' -- internal docs .claude templates .agents .codex`) | Yes | 当たるのは `watch.go:784`(`stop_failed` 入り)と、plan の Verify plan の行(`:70`)だけ |
| deadman が数えるイベントを書いた skill・rule・spec | 該当なし | `git grep -i deadman` で当たるのは、`docs/quality/quality-gates.md:78` とその templates の写し、director の spec(`docs/specs/2026-10-07-org-multi-org-director.md:18`、`:55-56`、`:169`)、`ralph.toml` と `ralph-config.sh` の `deadman_minutes` だけ。どれも数える種類を書かない。`ralph org watch --help`(`internal/cli/org.go:1191-1199`)も書かない。`/org` skill の `stop` の行(`.claude/skills/org/SKILL.md:167`)は `stop_failed` を書くが、watchdog との関係には触れない |
| tech-debt の解決済みの行(`README.md:167-168`) | Yes | 既存の RESOLVED の形どおり。HTML コメントに書いた窓の条件は `watch.go:821-825` と同じ |
| tech-debt の Test gaps の行(`README.md:169`)の (c) | No(V-2) | 下を参照 |
| plan の前提の 3 つ目とリスク | No(V-1) | 下を参照。承認の digest の範囲なので本文は直さず、PR 本文に書く |
| plan の進捗のチェックボックス | 遅れている(V-4) | 下を参照 |

### V-1: plan の前提とリスクが、残る窓を「起きない」と書いたまま

plan の前提の 3 つ目(`:33`)は「この PR を #208 と同じ release に入れれば、このずれは起きない」と書く。リスク(`:85`)は、残るのは main から自分でビルドしたバイナリを #208 とこの PR の間に使って警告を保存した場合だけ、と書く。一方、進捗の記録(`:106`)と doc comment(`watch.go:821-825`)は別の窓を 1 つ書いている。v5.1.0 の `ralph org watch` を動かしたままバイナリを更新し、新しい `ralph org stop` が `stop_failed` を書いたあとで古い watch が警告を保存すると、その基準に `stop_failed` が入らない。警告が残ったまま watch を新しいバイナリで立て直すと、数え直しが 1 大きくなり、新しい活動がないのに警告が消える。

この窓は、#208 とこの PR を同じ release に入れても残る。動き続けている watch のプロセスは古いコードのままで、`stop` は更新後のバイナリで走るからである。前提の 3 つ目の「起きない」は、この場合には当たらない。進捗の記録と doc comment の記述は正しい。plan の本文は承認の digest の範囲なので、ここでは直さない。

判断: PR 本文の既知の穴に次の 2 つを書く必要がある。plan のリスク(`:85`)と進捗(`:106`)も、PR 本文に書くと決めている。

1. この PR は #208 と同じ release に入れる。#208 だけの release を切らない(plan の rollout、`:89`)
2. 残る窓と、その避け方。更新の前に `ralph org watch` を止めておけば起きない。起きた場合は、止まった leader の警告が人に上がらずに消える。起きるのは、古い watch が更新をまたいで動き続け、その間に新しい stop が `stop_failed` を書き、そのあと古い watch が警告を保存し、警告が残ったまま watch を立て直したときに限る

同じ窓には逆向きの影響もある。古い watch は新しい stop が書いた `stop_failed` を数えないので、herdr が応答しない間は、leader が動いていても警告を人に上げうる(この PR の前と同じ動き)。こちらは人に上げる側に倒れるので、2 つ目と同じ避け方で足りる。これはコードを読んで確かめただけで、テストはない。

### V-2: tech-debt の Test gaps の行の (c) は、見直すきっかけがこの PR で来ている

`docs/tech-debt/README.md:169` の (c) は、`fakeWatchHerdr` がどの `get` にも not-found を返すので、watch のテストから呼んだ `Stop` は閉じ済みの分岐を通り、C-c も close も送らない、と書く。見直すきっかけは「(c) The next change to the watch tests.」になっている。この PR は watch のテストを変え、`PaneGetErr` を足した。`PaneGetErr` を設定すると `Stop` は `stop_failed` の分岐を通るので、「どの `get` にも not-found」はもう正確ではない。C-c と close の経路を watch のテストから通さないことは変わらない。/sync-docs で (c) を直すことを勧める。`PaneGetErr` のことを足し、C-c と close はまだ通らないと書き、きっかけを新しくする。LOW。

### V-3: AC4 の括弧書きと doc comment の理由の書き方が違う

AC4 は理由を「#208 と同じ release に入る」と書く。3b219fc7(self-review F-2 の直し)のあとの doc comment は、release の順序をコードから外し、「ralph up to v5.1.0 never writes it」という事実で書いた(`watch.go:818-821`)。この理由は、v5.1.0 とこの PR を含む release の間に `stop_failed` を書く release がないときに成り立つ。その条件はコードには書かれていない。V-1 の 1 つ目を PR 本文に書けば足りるので、AC4 は満たしていると判断した。この書き換えは plan の進捗に記録されていない。記録は self-review の Re-check の節にある。

### V-4: plan の進捗のチェックボックス

`:102` の「Review artifact created」は未チェックだが、self-review のレポートは 290e7efb で作られている。「Verification artifact created」は、このレポートで満たされる。/sync-docs か /pr で直せる。

## Observational checks

- なし。この PR が変えるのは doc comment、`case` の 1 語、テスト、tech-debt の行だけで、herdr もバイナリも動かしていない

## Coverage gaps

- テストは実行していない(`/test` の担当)。AC2、AC3、AC5 はテストの中身を読んで判断した
- `EventStopFailed` を `case` から外すと 1 本目と 3 本目が落ちるという記録(tech-debt の `:167`、plan の `:106`)は、コードを読んで合うことだけを確かめた。mutation は再現していない
- V-1 の窓(古い watch と新しい stop の組み合わせ)にはテストがない。2 つのバイナリをまたぐので、単体テストでは作りにくい
- 本物の herdr では動かしていない

## Verdict

- Verdict: pass
- Verified: AC1(`watch.go:842` と単体テスト 3 件)。AC4(doc comment の `:784`、`:790-796`、`:816-825` と、tech-debt の `:167-168`)。cross-review の ACTION_REQUIRED #1 がこの変更で直ること。`stop_failed` を書くのが stop と disband の系統だけであること。v5.1.0 が `stop_failed` を書かず、#208 が未配布であること。plan の承認の digest が一致すること。`run-static-verify.sh` が rc 0 で終わること
- Partially verified: AC2、AC3、AC5 はテストの中身とコードで確かめ、実行は /test に任せる。文書は V-1(plan の前提とリスク。PR 本文の既知の穴に書く)、V-2(tech-debt の Test gaps の行の (c)。/sync-docs で直す)、V-3(AC4 の書き方。PR 本文で足りる)、V-4(進捗のチェックボックス)が残る。どれも merge を止めない
- Not verified: テストの実行、mutation、V-1 の窓の再現、本物の herdr での動作
