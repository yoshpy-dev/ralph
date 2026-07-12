# Walkthrough: loop-model-routing

- Date: 2026-07-11
- Plan: docs/plans/archive/2026-07-11-loop-model-routing.md
- Branch: feat/loop-model-routing (base: main @ 45e9060)
- Diff size: 28 files, +2415 / -55

## 読む順序(推奨)

1. `.claude/rules/model-routing.md` — 何が導入されたかの全体像(per-phase
   テーブル、precedence、エスカレーション、レシート)
2. `scripts/ralph-config.sh` — 9 個の新変数(`RALPH_IMPLEMENT_MODEL` ほか
   per-phase 8 変数 + `RALPH_FORCE_MODEL`)
3. `scripts/ralph-cli-driver.sh` — 3 つの新ヘルパー
4. `scripts/ralph-pipeline.sh` — 呼び出し箇所の配線
5. Go 層(`internal/config/config.go`, `internal/cli/run.go`)と
   `templates/base/ralph.toml`
6. テスト(`tests/test-model-routing.sh` が新規、他 2 本は拡張)

## 変更の核

### 1. ルーティング(scripts/ralph-cli-driver.sh)

- `resolve_phase_model <phase> [cycle]` — 純関数。
  `RALPH_FORCE_MODEL` > エスカレーション(implement かつ pass ≥ 2)>
  フェーズ変数 > `RALPH_MODEL` フォールバックの順で解決。
- `write_model_receipt <phase> <cycle> <requested_model> <reason>
  [driver_override]` — `.harness/state/pipeline/model-receipts.jsonl` に
  1 行 JSON を追記。`requested_model` / `effective_model` / `honored` を
  分離し、codex ドライバがモデル引数を無視する事実を監査証跡が
  偽装しない(codex → `effective_model="codex-default"`, `honored=false`)。
- `run_agent` に第 4 引数 `[model]` を追加(省略時は従来どおり
  `RALPH_MODEL`。既存呼び出しは無変更で互換)。

### 2. 配線(scripts/ralph-pipeline.sh)

- 6 フェーズ(implement / self_review / verify / test / sync_docs / pr)+
  2 つの capability probe + cross-review 2 箇所をルーティング。
- `run_inner_loop` は第 3 引数として **1-based のパス番号**
  (`$((_outer_cycle + 1))`)を受け取る。パス 1 = 初回実装(sonnet)、
  パス 2 = 最初の fix-and-revalidate(→ `RALPH_ESCALATION_MODEL`)。
  ※ cycle 1 の cross-review 指摘(オフバイワン)の修正結果。
- cross-review のレシートは `driver_override` で「実際にレビューした
  CLI」(常にループドライバの反対側)を記録。

### 3. 設定伝搬(Go / toml)

- `internal/config/config.go` — `PhaseModelConfig`(9 キー)。
  `[pipeline.phases]` 不在時はシェルと同一のデフォルトへバックフィル
  (`force` のみ空が有意なので除外)。
- `internal/cli/run.go` — `appendEnvIfMissing` で env > toml > default を
  保証。`RALPH_FORCE_MODEL` は toml 値が非空のときだけエクスポート
  (空文字が env に入ると「override なし」と区別できなくなるため)。
- `templates/base/ralph.toml` — `[pipeline.phases]` セクション追加。

### 4. ミラー / ドキュメント

- `templates/base/scripts/{ralph-config,ralph-cli-driver,ralph-pipeline}.sh`
  はルートとバイト同一(check-sync.sh / check-pipeline-sync.sh ゲート)。
- `model-routing.md` はルート版のみ Go 層の記述を持つ既存の
  KNOWN_DIFF ポリシーを踏襲。
- `docs/tech-debt/README.md` に codex ドライバの per-phase モデル非対応を
  known gap として記録。

## レビューで特に見てほしい点

- `resolve_phase_model` の優先順位(FORCE > escalation > phase > fallback)
  が意図どおりか — `tests/test-ralph-cli-driver.sh` Test 8-9 と
  `tests/test-model-routing.sh` Case 4 が仕様のリファレンス。
- 1-based パス番号の意味論(`run_inner_loop` の `${3:-1}`)。
- デフォルト変更のリスク許容: implement が opus → sonnet になる。
  ロールバックは `RALPH_FORCE_MODEL=opus` 一発。

## エビデンス

- Self-review: docs/reports/self-review-2026-07-11-loop-model-routing.md
  (cycle 1 + cycle 2 addendum、CRITICAL/HIGH/MEDIUM 0)
- Verify: docs/reports/verify-2026-07-11-loop-model-routing.md(PASS ×2)
- Test: docs/reports/test-2026-07-11-loop-model-routing.md
  (cycle 2: shell 791/791、Go 全 9 パッケージ ok)
- Cross-review: docs/reports/cross-review-triage-loop-model-routing.md
  (cycle 1: ACTION_REQUIRED 2 件 → 修正、cycle 2: 指摘なし)
