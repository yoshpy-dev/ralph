# Test report: org-watch-stop-failed

- Date: 2026-10-08(JST。実行の記録は UTC の 2026-10-08 04:19〜04:56)
- Plan: docs/plans/active/2026-10-08-org-watch-stop-failed.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch fix/org-watch-stop-failed と base da4dccb0 の差分。コードの変更は 2765f002(`leaderActivityEventCount` の `case` に `EventStopFailed`、`fakeWatchHerdr.PaneGetErr`、テスト 3 本)と 3b219fc7(doc comment だけ)。テスト開始時の HEAD は 841b0585。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC3、AC5 を見た。AC4 は文言の基準なので /verify の担当
- Evidence: `docs/evidence/test-2026-10-08-org-watch-stop-failed.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-041947.log`(テスト追加前)と `docs/evidence/verify-2026-10-08-043845.log`(追加後)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加前、841b0585) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 約 11 分、rc 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加後、14ac4611 と同じ内容) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 963 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(テスト追加前) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 179 s、rc 0 |
| `go test -race ./internal/org/... -count=1`(追加前と追加後に 1 回ずつ) | 3 パッケージ | 3 | 0 | 0 | 21 s 前後、rc 0 |
| 新しいテストを `-count=5`(追加前は依頼の 3 本) | top-level 15、subtest 15 | 30 | 0 | 0 | 0.3 s、rc 0 |
| 新しいテストを `-count=5`(追加後は私の 1 本を足した 4 本) | top-level 20、subtest 15 | 35 | 0 | 0 | rc 0 |
| `go test ./internal/org/ -run TestWatch -count=3 -v`(追加前) | top-level 81(27 本 × 3)、subtest 6 | 87 | 0 | 0 | 0.5 s、rc 0 |
| `go test ./internal/org/ -run TestWatch -count=3 -v`(追加後) | top-level 84(28 本 × 3)、subtest 6 | 90 | 0 | 0 | rc 0 |
| `go test ./internal/org/ -count=1 -coverprofile`(追加後) | 1 パッケージ | ok | 0 | 0 | 16 s、rc 0 |
| mutation(9 件。`go test -overlay` で差し替え、worktree には触れていない) | 9 | red 9 件(テスト追加前は 8 件) | 生き残り 0 件(追加前は 1 件) | - | 測っていない |

- 2 回の `run-test.sh` は、どちらも `Requested scope: full`、`Language scope: full`、`Language packs selected: golang` で走った。shell の 39 本は `scripts/verify.local.sh` から走る。
- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(3 行目)。`internal/cli` は 176 s かかった。前回の記録(70〜100 s)より長いのは、同じ機械で別の session が動いていたためとみられる(2 回目の `run-test.sh` の終了時の load average は 7.95)。
- shell の件数の数え方: 各 suite の区間で `PASS` / `FAIL` で始まる行(先頭の空白は許す)を数え、`PASS: 29` や `PASS: 47 / 47` のような集計行は除いた。合計 2,047 件は #206 と org-state-dir-common の test レポートの件数と同じで、この差分は shell のテストを変えていない。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回とも出なかった(123 / 123)。失敗したテストがないので、単独の再実行はしていない。

## Coverage

- Statement: `internal/org` 92.6%(テスト追加の前も後も同じ)。ほかは `internal/cli` 85.3%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org/driver` 93.1%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。どれも org-stop-all cycle 2 の値と同じ
- Function: `leaderActivityEventCount` 100.0%、`checkDeadman` 94.4%、`sendAlert` 87.5%
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: base(da4dccb0)の `watch.go` と `watch_test.go` を `-overlay` で差し替えて測っても、`internal/org` は 92.6%、`leaderActivityEventCount` は 100.0% だった。`case` の行は `spawned` などで修正前から通るので、line coverage はこの修正の有無を区別しない。区別は下の mutation で確かめた

## Mutation

`internal/org/watch.go`(M4 だけ `watch_test.go`)を scratchpad の写しに差し替え、`go test -overlay=<json> ./internal/org/ -run 'TestWatch|TestLeaderActivity' -count=1 -v` で流した。worktree のファイルは書き換えていない。結果は、私が足したテスト(下の Test gaps 1)を含めたものである。

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| M1 | `case` から `EventStopFailed` を外す(この PR を戻す) | red | `TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert`(`watch_test.go:1412`、escalation が 1 件出る)、`TestLeaderActivityEventCount_StopFailed` の「watched org counts」 |
| M2 | `reason=watchdog_` の判定を外す | red | `TestLeaderActivityEventCount_StopFailed` の「reason=watchdog_ does not count」と、既存の `LegacyWatchdogStopEvent`、`PersistedAlertBaseline_SurvivesLegacyWatchdogStop`、`SeatSentEvent_..._LegacyWatchdogStopDoesNot` |
| M3 | orgID の絞り込みを外す | red | `TestWatch_Deadman_CrossOrgStopFailedEvent_DoesNotClearPendingAlert`(`:1463`)、`TestLeaderActivityEventCount_StopFailed` の「another org」、既存の `CrossOrgActivity_DoesNotClearPendingAlert` |
| M4 | 偽の herdr が `PaneGetErr` を無視する(`watch_test.go` 側) | red | `stop_failed` を作る 3 本が、どれも自分の前提の assert で落ちる(`:1400`、`:1449`、`:1487`、「expected an error while herdr does not answer」)。前提が成り立たないまま通ることはない |
| M5 | `stop_failed` だけ `sent` と同じ枝で数え、`reason=watchdog_` の判定を通さない | red | `TestLeaderActivityEventCount_StopFailed` の「reason=watchdog_ does not count」 |
| M6 | `stop_failed` だけ orgID の絞り込みより前で数える | red | `CrossOrgStopFailedEvent`、`TestLeaderActivityEventCount_StopFailed` の 2 件 |
| M7 | `checkDeadman` の `>` を `>=` にする(既存のコードの確認) | red | deadman の既存テスト 10 本と、新しい `CrossOrgStopFailedEvent`、`StopFailedBeforeAlert` |
| M8 | `sendAlert` の基準値だけ `stop_failed` を数えない(数え直しは数える) | red(テスト追加前は生き残り) | `TestWatch_Deadman_StopFailedBeforeAlert_DoesNotClearPendingAlert`(`:1509`。基準と数え直しがずれるので、cycle 1 の中で警告が消える) |
| M9 | `stop_failed` を 2 件と数える | red | `TestLeaderActivityEventCount_StopFailed` の「watched org counts」。deadman のテストは差が正かどうかしか見ないので、これを捕まえるのは単体テストだけ |

- M1 では `CrossOrgStopFailedEvent` は通る。このテストが固定するのは orgID の絞り込みで、`stop_failed` を数えるかどうかではないので、設計どおりである(M3 と M6 で落ちる)。
- M8 は plan の Test plan の Edge cases(「警告を出す前からある `stop_failed` は基準値に入り、差し引きで消えること」)にあたる。PR の 3 本は、どれも警告のあとに `stop_failed` を書くので、M8 を区別しなかった。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #208 の後退: herdr が応答しない間の leader の stop(`stop_failed` だけが残る)が活動に数えられず、deadman が人に上げる(`docs/reports/cross-review-triage-org-stop-all.md` の cycle 2 の ACTION_REQUIRED #1) | 直っている | `TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert` が通る。この PR を戻す M1 でこのテストが red になる |
| 別の org の活動で警告が消えない(cross-review AR-1) | 保たれている | `CrossOrgActivity_DoesNotClearPendingAlert` と新しい `CrossOrgStopFailedEvent_DoesNotClearPendingAlert` が通る。M3、M6 で red |
| pre-#152 の watchdog の cutoff(`reason=watchdog_`)で警告が消えない | 保たれている | `LegacyWatchdogStopEvent`、`PersistedAlertBaseline_SurvivesLegacyWatchdogStop` が通る。M2 で red |
| 今ある watch のテスト(AC5) | すべて通る | `-run TestWatch -count=3` で 0 failed。`PaneGetErr` が nil のときの `PaneGet` は変更前と同じ not-found を返す |

## Test gaps

1. 追加したテスト: `TestWatch_Deadman_StopFailedBeforeAlert_DoesNotClearPendingAlert`(14ac4611、`internal/org/watch_test.go` だけ、+55 行)。plan の Test plan の Edge cases にあるのに、テストがなかった。AC1〜AC5 の穴ではなく、Test plan の穴である。M8 が生き残ったのでこれを足し、M8 が red になることを確かめた。不要なら、この commit だけを戻せば元に戻る。
2. 更新をまたぐ窓(verify の V-1。v5.1.0 の `ralph org watch` が動いたまま、新しい `ralph org stop` が `stop_failed` を書く場合)にはテストがない。2 つのバイナリをまたぐので、単体テストでは作れない。M8 は、この窓と同じ「基準と数え直しのずれ」を 1 つのバイナリの中で作ったものにあたる。
3. `stop_failed` を書く経路のうち、テストが通すのは `Stop` を直接呼ぶものだけで、`Disband`、`disbandOwnLast`、`StopAll` から書かれた `stop_failed` は watch のテストでは作っていない。`leaderActivityEventCount` はイベントの種類と orgID と Details しか見ないので、書き手が違っても数え方は同じはずだ。コードを読んで確かめただけである。
4. `PaneGetErr` が作るのは、確認の `get` が失敗する経路(`ctrl_c=skipped: pane not confirmed`)の `stop_failed` だけである。`PaneClose` が失敗する経路(`pane=close failed`)の `stop_failed` は watch のテストにない。これも数え方は同じなので、影響はないとみている。未確認。
5. 本物の herdr では動かしていない。
6. verify の V-2(tech-debt の Test gaps の行の (c)。`fakeWatchHerdr` がどの `get` にも not-found を返すという記述)は、追加したテストも `PaneGetErr` を使うので、/sync-docs で直すときに合わせて見る必要がある。

## Verdict

- Pass: pass。`run-test.sh`(full)は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`-race`、新しいテストの `-count=5`、`TestWatch` の `-count=3` も 0 failed。mutation は 9 件すべて red。AC1、AC2、AC3、AC5 をテストの実行で確かめた
- Fail: なし
- Blocked: なし。Test gaps 2〜5 は merge を止めない
