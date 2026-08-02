# org-runtime-watchdog

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-02
- Related request: docs/specs/2026-08-01-org-runtime.md(PR 系列④: Watchdog 二層、FR-8)
- Related issue: N/A
- Type: feat
- Branch: feat/org-runtime-watchdog

## Objective

org runtime PR④: Watchdog 二層を実装する。パルス層(決定論タイマー `ralph org watch`: heartbeat / 生存 / budget 自動遮断 / スコープ変更検知)とウォッチャー層(パルス層トリガーのオンデマンド `claude -p` による意味判定)、Lead 宛通知+デッドマン条項(人間エスカレーション)。あわせて PR④ 送りの tech-debt(state-dir の cwd 相対解決、codex 権限 fail-closed の実機検証、evidence 赤入れ規約、batchable LOW 群)をクローズする。

## Scope

- **パルス層 `ralph org watch --org-id <id>`**(Go 常駐プロセス、既定 30 秒間隔):
  - 監視条件(すべて決定論): (a) heartbeat 途絶 = 座席の最終 manifest イベント時刻と herdr `state_change_seq` の双方が `stall_minutes` 停滞、(b) プロセス生存 = herdr agent get 失敗/pane 消失、(c) **budget wall-clock** = `spawned` ts + `[org.budget] seat_wall_clock_minutes` 超過(org 全体は total)、(d) スコープ変更検知 = 座席 cwd の `git status --porcelain` の変化を宣言 scope とともに通知(自由文 scope のため自動遮断はせず通知まで)。
  - アクション(権限は 3 つのみ): **遮断(PAUSE)** = ハードリミット(budget)超過時のみ、既存 `Stop` 経由で自動実行し Details に `watchdog budget cutoff` を記録(判断を経由しない自動遮断)。**通知** = Lead 宛に typed `ALERT`(新 TYPE)を agmsg send。**REPLAN 要求** = ALERT の本文で表現(watchdog は裁定しない)。
  - watch 自身の可観測性: `.harness/state/org/watch-status.json` に heartbeat を書き、`org status` から見える。
- **typed protocol への `ALERT` 追加**: `internal/org/protocol`(TASK_ID 不要)+ `.claude/rules/agent-messaging.md` の表+ミラー同期。watchdog → lead の通知専用。
- **`[org.watchdog]` 設定**(ロックステップ 3 面 + defaults_sync_test): `interval_seconds`(30)/ `stall_minutes`(15)/ `watcher_enabled`(true)/ `watcher_model`("haiku")。デッドマンは既存 `[org] deadman_minutes`(10)を執行に昇格。
- **ウォッチャー層(オンデマンド、ユーザー確定)**: パルス層が意味判定トリガー(ハードリミット未満の停滞・blocked 長期化・スコープ変更通知)を検知した時のみ、`claude -p --model <watcher_model>` を 1 回起動。入力は該当座席の pane 末尾(herdr pane read)+ 直近 manifest イベント+役割規約。出力は判定 JSON(`verdict: normal|circular|role_violation|fake_progress`, `reason`)で、異常判定時のみ Lead へ ALERT。receipts に記録(watchdog phase)。spec FR-8 の「常駐座席」記述はこの決定に合わせ /sync-docs で改訂。
- **デッドマン条項**: パルス層が Lead へ ALERT 送信後、`deadman_minutes` 以内に Lead の活動(新規 manifest イベント / lead pane の state_change_seq 前進 / agmsg 送信)が観測されない場合、または異常主体が Lead 自身の場合、人間へエスカレーション: `.harness/state/org/escalations.jsonl` 追記+stderr バナー+macOS では `osascript` 通知(best-effort)。
- **state-dir の cwd 相対解決**(tech-debt 発動条件成立): 全 org 動詞の state-dir 既定を `--state-dir` フラグ > `RALPH_ORG_STATE_DIR` env > **git toplevel の `.harness/state/org`** > cwd フォールバックに変更。lead/operator の cwd 分裂問題を構造的に解消(/org skill の同一 cwd 規約は「同一リポジトリ内なら不要」に緩和)。
- **codex 権限 fail-closed の実機検証**: codex CLI の対話モード sandbox/approval フラグを実機プローブし、検証できたモードのみマッピング解除(想定: autonomous → `--sandbox workspace-write` 系)。検証不能なら fail-closed 維持+プローブ結果を tech-debt に記録。
- **evidence 赤入れ規約**: evidence `.txt` 恒久化時に `$HOME` → `~` を置換する規約を `docs/quality/definition-of-done.md` に 1 行追加し、本 PR の evidence から適用(tech-debt クローズ)。
- **batchable LOW 群のクローズ**: `checkCapacityAndStart` シグネチャ整理 / `newOrgRuntime` シグネチャ(cfg を Org.Config から取得)/ `/org` skill frontmatter description の英語化(他 skill と統一)。permission enum 配置は現状(コメントで相互参照済み)を最終とし row をクローズ。
- **実機スモーク**(手動・証拠必須): (1) 短縮 budget(1 分)座席で watch が自動遮断し ALERT が lead に届く、(2) 短縮 deadman で escalations.jsonl+通知が発火、(3) ウォッチャー 1 回のオンデマンド判定(haiku)が実行され receipts に記録、(4) state-dir 解決により worktree/リポジトリルートの分裂が起きない。証拠 `docs/evidence/org-watchdog-smoke-*.txt`($HOME 赤入れ適用)。

## Non-goals

- 旧系撤去・ドキュメント全面改稿(PR⑤)。
- LLM ウォッチャーによる裁定・停止(権限は判定と ALERT まで。遮断はパルス層のハードリミットのみ)。
- トークン/コスト計測ベースの budget(wall-clock のみ。三値 receipts は継続)。
- スコープ違反の自動遮断(自由文 scope のため通知+ウォッチャー判定まで。glob 化は将来)。
- 分散(複数ホスト)監視。

## Assumptions

- herdr `agent get` の `state_change_seq` は状態遷移で単調増加する(PR②③スモークで観測済み)。
- watcher の `claude -p` は非対話で JSON を返せる(--output-format json は PR① の run_agent で実績あり)。
- デッドマンの「Lead の活動」は manifest/herdr/agmsg の 3 情報源の OR で判定(herdr 表示だけに依存しない — 元設計の合議原則)。
- `osascript` 不在環境では stderr+escalations.jsonl のみ(テストはファイル出力を正とする)。

## Affected areas

- `internal/org/`: watch.go(新規: パルス層ループ・条件・遮断・デッドマン)、watcher.go(新規: オンデマンド判定の起動と JSON 解釈)、protocol/(ALERT)、spawn.go/verbs.go(state-dir 解決の共有化があれば)、statedir.go(新規)
- `internal/cli/`: org.go(`watch` サブコマンド、state-dir 解決差し替え)、doctor.go(必要なら)
- `internal/config/`(`[org.watchdog]`、ロックステップ、defaults_sync_test)
- `.claude/rules/agent-messaging.md`(ALERT)+ミラー、`.claude/skills/org/SKILL.md`(watch 動詞・同一 cwd 規約の緩和)+ 4 面ミラー
- `scripts/ralph-config.sh` + templates ミラー、`templates/base/ralph.toml`
- `docs/quality/definition-of-done.md`(赤入れ規約 1 行)、`docs/tech-debt/README.md`(該当行クローズ)
- 触らない: `scripts/ralph-orchestrator.sh` / `ralph-pipeline.sh` / `internal/ui` / `internal/state`

## Design decisions

- **フロー**: Standard flow (/work)(ユーザー確定)。
- **ウォッチャーはオンデマンド `claude -p`**(ユーザー確定、spec FR-8 の「常駐」から意図的逸脱 — /sync-docs で spec 追従): 常駐コストゼロ・判定器ハングは次回起動で回復・「発火時のみ LLM 補助」の元設計と整合。
- **遮断は既存 Stop 経由**: 新しい kill 経路を作らない(元設計の「PAUSE は既存 abort 経路」原則)。
- **デッドマンは 3 情報源の OR**: herdr 表示・manifest・agmsg の合議(単一ソース誤判定の回避)。
- **Codex advisory 反映(5 件)**: ① watch の冪等性(条件ごとの dedupe キーを watch-status.json に永続化し、同一条件は回復まで 1 遮断/1 ALERT/1 エスカレーション。複数 interval のテスト必須)、② total budget の AC 化(org_start = 最初の非 dry-run `spawned`。超過時は全 active 座席を遮断し org レベルの記録)、③ ウォッチャーはパルスを塞がない(タイムアウト < interval・非同期起動・ハング/不正 JSON は `watcher_error` receipt。ハング時でも budget 遮断が発火するテスト)、④ `StopParams` に `Reason` を追加し watchdog 起因(条件種別・閾値・観測値)を stopped の Details に永続化、⑤ state-dir 解決は `Flags().Changed` +空既定+解決元を返す resolver で実装(flag 明示/env/git サブディレクトリ/git ルート/非 git の 5 ケーステスト)。
- Critical forks: フロー・ウォッチャー形態の 2 点を解決済み。

## Acceptance criteria

- [x] AC-1: `[org.watchdog]` が 3 面ロックステップで追加され defaults_sync_test が drift を検出する。Evidence: `internal/config/config.go`(`OrgWatchdogConfig`/`Default()`/`Load()`)、`templates/base/ralph.toml:122-126`、`scripts/ralph-config.sh:101-110`+テンプレミラー、`internal/config/defaults_sync_test.go`; verify report AC-1 (Met).
- [x] AC-2: protocol に `ALERT`(TASK_ID 不要)が追加され、rule doc の表+ミラーが同期(check-sync 系 green)。Evidence: `internal/org/protocol/protocol.go`(`TypeAlert`)、`.claude/rules/agent-messaging.md`+3 ミラー(`cmp` byte-identical); verify report AC-2 (Met).
- [x] AC-3: `ralph org watch` が interval ごとに条件評価し、budget 超過座席を Stop 経由で自動遮断(`StopParams.Reason` で条件種別・閾値・観測値を Details に永続化)、Lead へ ALERT を送る(スタブで検証)。Evidence: `internal/org/watch.go`(`evaluateSeatBudget`); test: `TestWatch_SeatBudgetCutoff_AtBoundary_NotBeforeThenCutoffThenDeduped`, `TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds`.
- [x] AC-3b(total budget): org_start(最初の非 dry-run `spawned`)から `total_wall_clock_minutes` 超過で全 active 座席を遮断し、org レベルの記録を残す(複数座席テスト)。Evidence: `evaluateTotalBudget`; test: `TestWatch_TotalBudgetCutoff_CutsAllActiveSeats_OneOrgLevelAlert`.
- [x] AC-3c(冪等性): 同一条件は回復まで 1 遮断/1 ALERT/1 エスカレーションに dedupe される(handled 状態を watch-status.json に永続化。2 interval 以上のテストで検証)。Evidence: `watchConditionRecord.Cutoff`/`Active`; test: boundary test above + `TestWatch_Stall_AlertsNoCutoff_RecoversAndRefires`.
- [x] AC-4: heartbeat 停滞・pane 消失・スコープ変更が ALERT として Lead に通知される(遮断はされない)(スタブ)。Evidence: `evaluateSeat` (c)/(d)/(e); test: `TestWatch_Stall_*`, `TestWatch_Liveness_AlertOnAgentGetError`, `TestWatch_ScopeChange_AlertCarriesScopeText_NoCutoff`, `TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents`.
- [x] AC-5: デッドマン: ALERT 後 `deadman_minutes` 無応答(3 情報源 OR)または Lead 自身の異常で `escalations.jsonl` 追記+stderr(+darwin 通知 best-effort)(短縮値でテスト)。Evidence: `checkDeadman`, `realEscalate`(`osascript`, `%q` 引用); test: `TestWatch_Deadman_NoActivity_EscalatesOnceAfterTimeout`, `TestWatch_Deadman_LeadActivity_PreventsEscalation`, `TestWatch_Deadman_LeadIsAnomalySubject_EscalatesImmediately`. **既知の残課題**: `leadActivityEventCount` は watchdog 自身のイベントのみ除外し、org/seat スコープには絞っていない(verify report AC-5 caveat、test report Test gaps)。tech-debt に記録(下記)。
- [x] AC-6: ウォッチャーがトリガー時のみ 1 回起動し(タイムアウト < interval・非同期でパルスを塞がない)、判定 JSON を解釈、異常時のみ ALERT、receipts に watchdog phase で記録。ハング/不正 JSON は `watcher_error` receipt となり、**ウォッチャーがハングしても budget 遮断は発火する**(ハングスタブテスト)。Evidence: `internal/org/watcher.go`(`newWatchdogHooks`、`watcherInvokeTimeout` 固定 60 秒); test: `TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt`, `TestRunWatcher_TimeoutIndependentOfSmallInterval`, `TestNewWatchdogHooks_Dispatch_NeverBlocksCaller`, `TestNewWatchdogHooks_SingleFlight_SecondTriggerSkippedWhileBusy`.
- [x] AC-7: state-dir 既定が flag(`Flags().Changed` で明示検出・空既定)> env > git toplevel > cwd で解決され(resolver は解決元も返す)、リポジトリ内の異なる cwd から同一 manifest を読める(明示相対 flag / env / git サブディレクトリ / git ルート / 非 git の 5 ケーステスト+スモーク)。tech-debt 行クローズ。Evidence: `internal/org/statedir.go`(`ResolveOrgStateDir`); test: `TestResolveOrgStateDir_ExplicitFlagWins`, `_EnvWinsOverGitAndCwd`, `_GitSubdirResolvesToToplevel`, `_GitRoot`, `_NonGitCwdFallsBackToCwd`; tech-debt row struck through (`docs/tech-debt/README.md` line 64-65, `RESOLVED 2026-08-02`). スモーク上のクロス cwd 実機確認は `docs/evidence/org-watchdog-smoke-2026-08-02.txt` の cross-cwd addendum(commit `6a09e64`)で補完。
- [x] AC-8: codex 対話フラグの実機プローブ結果が記録され、検証できた場合のみ autonomous マッピング解除(できなければ fail-closed 維持を明記)。Evidence: `docs/evidence/org-watchdog-smoke-2026-08-02.txt`(`-s/--sandbox`・`-a/--ask-for-approval` 確認)、`internal/org/permissions.go`(`codex_verified` gate, 既定 false); test: `TestPermissionArgsForDriver_Codex_FailClosed`, `_VerifiedUnlocksMapping`, `TestPermissionArgsForDriver_Codex_UnknownMode_DistinctError`.
- [x] AC-9: 赤入れ規約が definition-of-done に追記され、本 PR の evidence に適用。batchable LOW 群クローズ(signature 2 件+description 英語化)。Evidence: `docs/quality/definition-of-done.md`(`$HOME → ~` 行)、`docs/evidence/org-watchdog-smoke-2026-08-02.txt`(redaction header)、`checkCapacityAndStart`/`newOrgRuntime` シグネチャ整理(`internal/org/spawn.go`/`internal/cli/org.go`)、`/org` skill description 英語化(4 ミラー、`check-skill-sync.sh` green); tech-debt row struck through(`docs/tech-debt/README.md` line 66-69)。
- [x] AC-10(実機スモーク): 短縮 budget での自動遮断+ALERT 受信、短縮 deadman でのエスカレーション発火、ウォッチャー実判定 1 回、state-dir 解決の実機確認。証拠 `docs/evidence/org-watchdog-smoke-2026-08-02.txt`(cross-cwd addendum commit `6a09e64` を含む)。全 4 項目が同ファイルで実機確認済み。
- [x] AC-11: `go test ./...` / `./scripts/run-verify.sh` green。既存フロー非干渉。Evidence: verify report(`./scripts/run-static-verify.sh` PASS)、test report(`./scripts/run-test.sh` 30 shell suites + `go test ./...` 13 packages + `-race` targeted run, 全 green)。

## Implementation outline

1. **Slice 1 — state-dir 解決 + LOW batch**: statedir.go(flag > env > git toplevel > cwd)+全動詞差し替え、checkCapacityAndStart/newOrgRuntime 整理、skill description 英語化(ミラー)、tech-debt 行更新。
2. **Slice 2 — ALERT + `[org.watchdog]` 設定**: protocol/rule doc/ミラー、config 3 面+tripwire。
3. **Slice 3 — パルス層 `org watch`**: 条件評価ループ、budget 遮断、ALERT 送信、watch-status.json、デッドマン(escalations.jsonl+通知)。スタブでの時間注入(fake clock)テスト。
4. **Slice 4 — ウォッチャー層 + codex プローブ**: オンデマンド claude -p 起動・JSON 解釈・receipts、codex 対話フラグの実機プローブと(検証時のみ)マッピング解除。
5. **Slice 5 — 実機スモーク + 証拠 + ドキュメント**: definition-of-done 追記、/org skill 追従(watch・cwd 規約緩和)、スモーク、evidence 赤入れ、AGENTS.md 最小追従。

## Verify plan

- Static: `./scripts/run-static-verify.sh`。
- Spec compliance: FR-8(二層・通知 Lead 宛・デッドマン・権限 3 種)、FR-9(遮断イベント記録)。ウォッチャー形態の spec 改訂を /sync-docs で実施。
- Doc drift: rule/skill 4 面ミラー、definition-of-done、tech-debt 行、spec FR-8。
- Evidence: スタブテスト出力、実機スモークログ(赤入れ済み)。

## Test plan

- Unit: 条件評価(fake clock で budget/stall 境界)、デッドマン 3 情報源 OR、ALERT 生成の protocol 適合、state-dir 解決優先順位、watcher JSON 解釈(正常/异常/破損)、escalations 追記。
- Integration: スタブ herdr/agmsg/claude で watch 1 サイクル(遮断→ALERT→デッドマン→エスカレーション)、state-dir クロス cwd。
- Regression: PR①〜③全テスト、ロックステップ、check-sync 系。
- Edge: budget ちょうど、watch 対象 org が空、watcher 無効設定、escalations 並行追記。

## Risks and mitigations

| リスク | 影響 | 緩和 |
|---|---|---|
| watch の誤遮断(時刻ずれ等) | 座席の不当停止 | 遮断は budget のみ・境界テスト・Details に根拠記録・遮断前に猶予 1 interval |
| デッドマン誤発火 | 不要な人間割込み | 3 情報源 OR+短縮値テストで検証 |
| state-dir 既定変更の互換 | 既存ユーザーの状態位置移動 | cwd=toplevel なら従来と同一パス。差が出るのはサブディレクトリ実行時のみで、旧位置が存在すれば警告表示 |
| codex フラグ形状の誤解 | 権限の誤適用 | 実機プローブ必須・検証不能なら fail-closed 維持 |

## Rollout or rollback notes

- 純追加系(watch を起動しない限り不可視)+ state-dir 既定の改善(上記互換注意)。ロールバックは revert。
- スライス別 green ゲート(PR③ と同じ規約)。

## Open questions

- watcher 判定 JSON のスキーマ詳細(実装時に確定、protocol とは独立)。
- 遮断猶予(1 interval)の要否 — 実装時判断。

## Implementation notes (deviations)

- **実機スモーク発見 #1(ALERT 未配送)**: sendAlert が座席操作用 `Send` 動詞を使っており、lead が座席でない org(現セッション昇格型)では ALERT がサイレントに落ちていた(attempt 1 で agmsg 履歴 0 件)。identity レベルの `Agmsg.Send`(`SendWatchdogAlert` に集約)へ修正(7101351)。attempt 2 で watchdog→lead の ALERT 2 件配送を実機確認。
- **実機スモーク発見 #2(watcher タイムアウト不足)**: interval 由来の 10 秒では実 `claude -p` が完走しない。watcher は非同期 single-flight でパルスは構造的に保護済みのため、interval 非依存の固定 60 秒(テスト用 seam 変数)に変更(7101351)。attempt 2 で haiku が `verdict=normal` を実判定。
- **codex 権限プローブ(AC-8)**: 対話モードに `-s/--sandbox <MODE>` と `-a/--ask-for-approval <POLICY>` の存在を実機確認。マッピングは `[org.permissions] codex_verified = true` の明示解除に限り有効(既定 fail-closed 維持)。ライブ座席での最終検証は運用者の初回 codex 運用時。
- **スモーク成立(AC-10)**: 1 分 budget 座席が `observed=1m0s` ちょうどで自動遮断(Reason 完全監査)、stall→watcher 実判定、デッドマン発火(escalations.jsonl+stderr)、alert_id での dedupe、watch-status heartbeat、いずれも実機確認。証拠 docs/evidence/org-watchdog-smoke-2026-08-02.txt($HOME 赤入れ適用)。
- **Cycle 2(cross-review + self-review 追加修正)**: cross-review AR-1/AR-2(`19c7630`)で (a) `leadActivityEventCount` に `orgID` フィルタを追加(異なる org のイベントを lead activity から除外)、(b) `evaluateTotalBudget` に active seat が 0 件の org を total-budget cutoff/ALERT の対象から外すガードを追加(既に停止済みの org を再監視した際のスプリアス ALERT/デッドマン登録を防止)。続く cycle-2 self-review(`ccf506e`)が AR-1 の "fully closes this row" という closure note を過大評価と指摘(finding M2-1): 同一 org 内の無関係な座席自身のイベントは依然 lead activity としてカウントされていたため、`leadActivityEventCount` を `ev.SeatID == LeadIdentity` または watchdog 自身の遮断書き込みでない `stopped`/`disbanded` イベントのみを数えるよう座席帰属化。同 cycle で `newWatchdogHooks`(`internal/cli/org.go`)が `context.Background()` ではなく `cmd.Context()` を追跡 goroutine の `RunWatcher`/`SendWatchdogAlert` 呼び出しへ通す形に修正(finding M2-2): 常駐 `ralph org watch`(非 `--once`)モードで SIGINT が `watcherInvokeTimeout`(固定 60 秒)を待たずに即座に判定を打ち切れるようにした。両修正とも回帰テスト追加済み(`docs/reports/verify-2026-08-02-org-runtime-watchdog.md` cycle-2 節、`docs/reports/test-2026-08-02-org-runtime-watchdog.md` cycle-2 節参照)。`leadActivityEventCount` の tech-debt 行は両半分の修正完了により `docs/tech-debt/README.md` で fully closed とした。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/org-runtime-watchdog)
- [x] Implementation started
- [x] Implementation complete (Slices 1-5: f938413 / 274ca41 / ea75adc+611b14e / 63313a5+00378d1 / 7101351+smoke)
- [x] Review artifact created (`docs/reports/self-review-2026-08-02-org-runtime-watchdog.md`; H-1/H-2/M-1/M-2 fixed same cycle, M-3/M-5/M-6/L-7 fixed, M-4/L-2/L-5/L-6 deferred to tech-debt)
- [x] Verification artifact created (`docs/reports/verify-2026-08-02-org-runtime-watchdog.md`; PASS with two follow-up items — AC-5/M-4 partial fix, AC-10 state-dir smoke line, both closed by the cross-cwd addendum + tech-debt row below)
- [x] Test artifact created (`docs/reports/test-2026-08-02-org-runtime-watchdog.md`; Pass, 0 failures)
- [ ] PR created

## Readiness checklist

- [x] フロー確定(Standard /work)・ウォッチャー形態確定(オンデマンド claude -p、spec 逸脱は /sync-docs で追従)
- [x] PR③ までの実機制約を前提に織り込み(herdr state_change_seq / done 状態 / state-dir 分裂)
- [ ] Codex plan advisory(次ステップ)
