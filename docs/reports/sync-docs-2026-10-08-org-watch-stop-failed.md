# sync-docs report: org-watch-stop-failed

## Cycle 1

- Date: 2026-10-08
- Plan: `docs/plans/active/2026-10-08-org-watch-stop-failed.md`
- Pipeline cycle: 1 of 2。差分は merge base `da4dccb0` から branch HEAD `2d98e7be`(fix/org-watch-stop-failed)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-08-org-watch-stop-failed.md`(`290e7efb` で追加、`1b609add` で再確認を追記。Merge 可)、
  `docs/reports/verify-2026-10-08-org-watch-stop-failed.md`(`841b0585`。pass、V-1〜V-4)、
  `docs/reports/test-2026-10-08-org-watch-stop-failed.md`(`2d98e7be`。pass、mutation 9 件すべて red、Test gaps 1〜6)

## Summary

この差分で古くなった文書は 2 か所だった。tech-debt の Test gaps の行の (c)(verify の V-2)と、plan の進捗のチェックボックス 3 つ(V-4)である。deadman が数えるイベントを書いた skill、rule、spec、README、`ralph org watch --help` は見つからなかった。`stop_failed` を数える側の変更は、`watch.go` の doc comment と tech-debt の解決済みの行(167〜168 行目)に先に入っていて、verify が確認済みである。

tech-debt の既存の行 1 件(169 行目)を更新した。新しい行は足していない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 169 行目の Test gaps の行だけを変えた。(c) を「`fakeWatchHerdr` は `get` のどれにも not-found を返す。ただし、テストが `PaneGetErr` を設定すると `PaneGet` が失敗する」に直した。設定がなければ watch のテストから呼んだ `Stop` は閉じ済みの分岐を通り、設定があれば `Stop` は pane を確認できず `stop_failed` で終わる(`TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert`、`TestWatch_Deadman_CrossOrgStopFailedEvent_DoesNotClearPendingAlert`、`TestWatch_Deadman_StopFailedBeforeAlert_DoesNotClearPendingAlert`)。どちらの分岐も `confirmSeatPane` を通らないので、C-c も close も送らない、と書いた。きっかけ(trigger)の (c) は「watch のテストの次の変更」から、「C-c か close を assert する最初の watch のテスト、または `stopSeatPane` のその 2 つの手順の変更。fake が `PaneGet` に tab と workspace の id、`TabGet` に座席 id、`WorkspaceGet` に org_id を返すようにして `confirmSeatPane` を通す」に変えた。Related に、`PaneGetErr` を足した plan を足した。Impact の (c)(「watch のテストは `Stop` が送るものの変化を見られない」)と Why deferred は、今も当たるので変えていない。ほかの列と (a)(b)(d)(e)、ほかの行には触れていない |
| `docs/plans/active/2026-10-08-org-watch-stop-failed.md` | 進捗のチェックボックス(102〜104 行目)だけを変えた。Review / Verification / Test artifact created にチェックを付けた(V-4)。「PR created」は `/pr` が付けるのでそのまま。本文は触っていない。`./scripts/plan-visual.sh digest` は変更後も `53d656a5e19f`(plan の `- Approved:` 行と一致) |
| `docs/insights/events/2026-10-08-org-watch-stop-failed.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 1) |
| `docs/reports/sync-docs-2026-10-08-org-watch-stop-failed.md` | この report |

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep -n 'spawn_started' -- .claude docs README.md internal/cli templates`(`.agents`、`.codex`、`AGENTS.md`、`CLAUDE.md` も足した) | 数える種類の一覧を持つ現行の文書はなかった。当たったのは、履歴(`docs/plans/archive/`、`docs/reports/`、`docs/evidence/`)、この PR の plan 自身(18、70 行目)、org runtime spec の FR-9(監査証跡で、記録するイベントの名前を挙げるだけ)、tech-debt の RESOLVED 行(60、79 行目)、`internal/cli/org_test.go` の座席の台帳の fixture。一覧は `internal/org/watch.go` の doc comment にだけあり、`stop_failed` が入っている |
| `/org` skill の `watch` の説明(4 面: `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`) | 4 面とも同じ内容で、`watch` の行(170 行目)と並走の注意(324〜326 行目)は「停滞/生存/スコープ変更の ALERT・デッドマン人間エスカレーション」と書くだけ。活動として数えるイベントは書いていない。`stop` の行(167 行目)は `stop_failed` を書くが、watchdog との関係には触れない。変更なし |
| `ralph org watch --help`(`internal/cli/org.go:1191-1199`) | 「leader が `[org].deadman_minutes` 以内に応答しないとき deadman escalation」と書くだけで、数えるイベントを挙げない。変更なし |
| `docs/quality/quality-gates.md:78` と `templates/base/docs/quality/quality-gates.md:77` | 「Watchdog pulse layer」の 1 行だけ。変更なし |
| `docs/specs/2026-08-01-org-runtime.md`(FR-8、90、179 行目)、`docs/specs/2026-10-07-org-multi-org-director.md`(18、56 行目) | deadman の仕様と、leader の無応答が人に届く唯一の経路としての言及で、数える種類は書かない。変更なし |
| `README.md`、`AGENTS.md`、`CLAUDE.md`、`docs/architecture/`、`docs/recipes/`、`docs/quality/`、`.claude/rules/`、`.ralph/`、`templates/`(`git grep -n -i 'watch\.go\|watch_test\|fakeWatchHerdr\|PaneGetErr'`、`docs/reports` を除く) | 0 件。`watch.go` もそのテストも指していない。変更なし |
| tech-debt の `fakeWatchHerdr` を挙げる行 | 当たるのは 165 行目の (c)(F-7)と 169 行目の (c)。165 行目は `driver.NewHerdrError` が「テストが not-found の返答を作るためだけに export されている」と言い、`PaneGet` は `PaneGetErr` が nil のとき今も `NewHerdrError` で not-found を返す(`watch_test.go:90-97`)ので正しいまま。169 行目は上で更新した |
| tech-debt の 167〜168 行目(`leaderActivityEventCount` の解決済みの行) | verify が確認済み(HTML コメントの窓の条件は `watch.go:821-825` と同じ)。今回は変更していない |
| harness 内部の整合(skill、hook、rule、script、language pack) | この差分はどれも変えていない(変更は `internal/org/watch.go`、`internal/org/watch_test.go`、docs だけ)ので、Harness-internal sync の確認項目は該当なし |

## Found but left

- verify V-1: plan の前提の 3 つ目(33 行目)とリスク(85 行目)は、更新をまたぐ窓を「起きない」「残るのは手元ビルドだけ」と書いたままで、進捗の記録(106 行目)と `watch.go` の doc comment が書く窓と食い違う。本文は承認 digest の範囲なので直していない。`/pr` が PR 本文の既知の穴に書く。内容は 2 点: (1) この PR は #208 と同じ release に入れ、#208 だけの release を切らない。(2) v5.1.0 の `ralph org watch` が更新をまたいで動き続け、その間に新しい `stop` が `stop_failed` を書き、古い watch が警告を保存し、その警告が残ったまま watch を新しいバイナリで立て直すと、止まった leader の警告が人に上がらずに消える。更新の前に watch を止めておけば起きない。逆向きの影響として、古い watch は新しい `stop_failed` を数えないので、herdr が応答しない間は leader が動いていても警告を上げうる(この PR の前と同じ動き、人に上げる側に倒れる)
- verify V-3: AC4 の括弧書き(「#208 と同じ release に入る」)と、3b219fc7 のあとの doc comment(「ralph up to v5.1.0 never writes it」という事実)の書き方が違う。V-1 の (1) を PR 本文に書けば足りるので、文書は変えていない
- test report の Test gaps 3(`Disband`、`disbandOwnLast`、`StopAll` が書く `stop_failed` を watch のテストで作っていない)と 4(`PaneClose` が失敗する経路の `stop_failed` が watch のテストにない)は、tech-debt に足していない。`leaderActivityEventCount` はイベントの種類、orgID、Details しか見ないので、書き手や経路が違っても数え方は同じで、test report も影響なしとみている。足す場合は、169 行目の (c) と同じ行に入れるのが近い
- self-review の F-3(`fakeWatchHerdr.PaneGetErr` の代入が `mu` で守られていない)は、受け入れ済みの LOW のまま。tech-debt には足していない
- `docs/tech-debt/README.md` の既存の行は、169 行目のほかには触っていない

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/run-static-verify.sh` | rc 0。`check-sync.sh` PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh` OK、`check-skill-sync.sh` OK(13 skill)、`check-template-purity.sh` PASS、tech-debt README plan references OK、gofmt ok、golangci-lint `0 issues.`、branch secret scan clean(`da4dccb0..2d98e7be`) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-08-org-watch-stop-failed.md` | `53d656a5e19f`(plan の `- Approved:` 行と一致。チェックボックスの変更の前後で同じ) |
| `git grep -n 'spawn_started' -- .claude docs README.md internal/cli templates` ほか(上の表) | 数える種類の一覧を持つ現行の文書なし |

## Not verified

- 「deadman が数えるイベントを書いた文書はない」は、`spawn_started`、`deadman`、`stop_failed`、`watch.go` などで grep して当たった範囲の確認である。別の言い方で書かれた記述は拾えていない可能性がある。未確認です
- tech-debt 169 行目の (c) の記述(`PaneGetErr` を設定した `Stop` が `stop_failed` で終わること、C-c と close を送らないこと)は、`verbs.go` の `confirmSeatPane` と `stopSeatPane`、`watch_test.go` の fake を読んで書いた。テストは実行していない(test report が実行済み)
