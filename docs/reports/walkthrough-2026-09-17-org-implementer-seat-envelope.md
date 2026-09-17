# Walkthrough: org-implementer-seat-envelope

- Date: 2026-09-17
- Plan: docs/plans/archive/2026-09-16-org-implementer-seat-envelope.md
- Author: Claude Code (standard flow, implementer / reviewer / verifier / tester / doc-maintainer subagents)

## What changed

org runtime の座席機構を 3 点で見直した。

1. **budget 概念の撤去**: `[org.budget]`(seat / total wall-clock、max_fix_rounds)と watchdog の wall-clock 自動遮断を削除。stall / 生存 / スコープ外変更 / デッドマンの監視は不変。旧 `watch-status-<org_id>.json` に残る `seat_budget` / `total_budget` エントリは読み込み時に prune し、廃止済み制御の ALERT がデッドマン経路で人間へ誤エスカレーションしないようにした。
2. **model_pool 既定の刷新と `--model` 運用**: 既定プールを claude `fable` / `opus` / `sonnet` / `haiku` + codex `gpt-6-astra` / `gpt-5.6-sol` / `gpt-5.6-terra` / `gpt-5.6-luna` / `gpt-5.5` に変更(config 3 面ロックステップ)。`ralph doctor` に codex スラッグを `~/.codex/models_cache.json` と突き合わせる軽量 Check を追加(プロセス起動なし)。`ralph org spawn` は `--model` 必須をやめ、`start` と同じく「役割に許可されたプール先頭」へ stderr 警告付きでフォールバックする。運用ルールとして `--model` は必ず明示する。
3. **implementer 座席と座席内 fan-out**: 役割を lead / implementer / reviewer / qa の 4 種にし、implementer の役割雛形を新設。implementer / reviewer / qa の雛形に「座席内 fan-out」節を追加し、座席が自分のドライバのサブエージェントへ作業を分割してよいこと、子は org の座席ではなく lead へ送信しないことを明文化した。

cross-review(cycle 1)で見つかった後方互換の退行(`driver_pool` のみを上書きした ralph.toml が新既定の codex エントリで Load に失敗する)を cycle 2 で修正した。

## Key files to read first

- `internal/config/config.go` — `OrgBudgetConfig` 削除、`Default()` の新プール、`Load()` の `orgPoolKeysPresent` / `filterModelPoolByDrivers`(driver_pool のみ上書き時に継承プールを絞る)
- `internal/org/watch.go` — budget 評価関数と `Cutoff` ラチェットの削除、`pruneRetiredConditions` / `retiredConditionTypeFromAlertID`
- `internal/cli/doctor_codex_models.go` — `checkCodexModelSlugs`(pass / warn / info の 3 経路)
- `internal/cli/org.go` — `resolveModelOrWarn`(spawn / start 共通)、spawn の required ループから `--model` を除外
- `internal/org/envelope_summary.go` — `DefaultModelForDriverAndRole`(`[org.roles]` を尊重するフォールバック)
- `internal/org/prompts/implementer.md`(新設)、`reviewer.md` / `qa.md` の「座席内 fan-out」節、`lead.md` の委譲先と `--model` ルール
- `templates/base/ralph.toml`、`scripts/ralph-config.sh`(両面)、`.claude/skills/org/SKILL.md`(4 面ミラー)

## Main control flow

- **config.Load()**: `Default()` → `toml.Unmarshal` → `orgPoolKeysPresent` で `driver_pool` / `model_pool` の明示有無を probe → `driver_pool` のみ明示なら `filterModelPoolByDrivers(Default().Org.ModelPool, cfg.Org.DriverPool)`(空なら `driver_pool` を名指すエラー)→ 既存の `[org]` 検証(model_pool 非空、driver 所属、重複、roles 参照)。
- **ralph org spawn / start**: フラグ検証 → `newOrgRuntime` → `resolveModelOrWarn(cfg, driver, role, model, stderr)` → `--model` 省略時は `DefaultModelForDriverAndRole` で役割許可プール先頭を選び警告 1 行 → `Spawn`(`ValidateSpawnEnvelope` は従来通り)。
- **ralph org watch**(1 サイクル): status 読み込み → `pruneRetiredConditions` → 座席ごとに stall / 生存 / スコープ外変更 → デッドマン。budget 評価は存在しない。
- **ralph doctor**: Check 10 の envelope 要約の直後に Check 11「Org codex model slugs」。`CODEX_HOME` があればそこ、なければ `~/.codex/models_cache.json` を JSON デコードし、プール内 codex スラッグの有無で pass / warn、ファイル無し・codex エントリ無し・JSON 不正は info。

## Risky code paths

- `pruneRetiredConditions` は `watchPendingAlert.AlertID`(`<org>/<seat>/<condition>@<unixnano>`)を文字列パースして condition を復元する。ID 形式が変われば prune が効かなくなるため、`sendAlert` の形式と回帰テスト `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` を一緒に見る。
- `filterModelPoolByDrivers` は「文書側に `model_pool` が無い」場合だけ動く。明示 `model_pool` は従来通り厳格検証(`TestLoad_ExplicitModelPoolStillStrict`)。
- `--model` 省略時のフォールバックは claude で `fable`(最上位モデル)になる。運用ルール(`/org` skill、lead 雛形)で明示を必須化しているが、機構は fail-closed ではない。
- 既定プールの codex スラッグはモデル更新で陳腐化する。実装中に `gpt-6-astra` が一時的にローカルキャッシュから消え、doctor が warn を出した事例あり(翌日復帰)。Check はその検知が目的で、既定値の自動更新はしない。

## What a human reviewer should pay special attention to

- budget 撤去は破壊的変更。下流の `ralph.toml` に `[org.budget]` が残っていても Load は無視する(`TestLoad_IgnoresRetiredOrgBudgetTable`)が、`RALPH_ORG_*BUDGET*` / `RALPH_ORG_MAX_FIX_ROUNDS` を参照するローカルスクリプトがあれば未定義になる。
- 座席内 fan-out はスター型を壊さない前提(子は座席内部、agmsg identity を持たない)を雛形の文言だけで担保している。機構的な制御は追加していない。
- `DefaultModelForDriver`(役割非依存)は本番の呼び出し元がなくなったが、ユニットテスト付きの基本関数として残した。
- `StopParams.Reason` の本番プロデューサが budget 撤去でゼロになった(self-review MEDIUM-3、tech-debt 起票済み)。

## Known limitations

- claude エイリアスのオフライン検証手段はない(`ralph doctor --probe-models` の実起動プローブに委ねる)。
- codex 座席の autonomous / edits permission mode は引き続き未検証で fail-closed(`codex_verified = false`)。既定プールに codex モデルを入れても、autonomous 座席として起動するには先に実機検証が要る。
- 先送りした LOW 所見(cycle-1 8 件 + cycle-2 1 件、いずれもコスメティック)は `docs/tech-debt/README.md` に 1 行でまとめて起票。
