# Walkthrough: org-runtime-watchdog (PR④)

- Date: 2026-08-03
- Plan: docs/plans/archive/2026-08-02-org-runtime-watchdog.md(アーカイブ後)
- Spec: docs/specs/2026-08-01-org-runtime.md(FR-8)

## この PR の性格

org runtime の安全網。決定論パルス層(`ralph org watch`)+オンデマンド LLM ウォッチャーの二層 Watchdog を追加し、budget 自動遮断・Lead 宛 typed ALERT・デッドマン人間エスカレーションを実機で証明した。品質パイプラインは 3 サイクル(cap を 1 回引き上げ、デッドマン安全網の P1 を PR 内で修正)。

## 読む順序

1. **docs/plans/archive/2026-08-02-org-runtime-watchdog.md** — AC・Codex advisory 5 件・実機/レビュー起因の修正全履歴。
2. **internal/org/watch.go** — パルス層本体。条件評価(seat/total budget・stall・liveness・scope 変更)、遮断は既存 Stop 経由(`StopParams.Reason` で完全監査)、条件キー dedupe(org 別 `watch-status-<org>.json`)、デッドマン(3 情報源: org+lead 帰属の manifest イベント/lead pane seq/lead 発信 history 行数の単調比較)、`escalations.jsonl`+osascript。
3. **internal/org/watcher.go** — オンデマンド `claude -p` 判定(非同期 single-flight・固定 60 秒・三値 receipts・JSON verdict)。
4. **internal/org/statedir.go** — state-dir の flag > env > git toplevel > cwd 解決(lead/operator の cwd 分裂解消)。
5. **internal/org/protocol**(ALERT 追加)、`[org.watchdog]` 設定 3 面、permissions(codex_verified ゲート)。
6. **docs/evidence/org-watchdog-smoke-2026-08-02.txt** — 実機証拠(2 バグ発見→修正→完走、$HOME 赤入れ)。

## 実機・レビューが検出した主要バグ(全て修正済み)

| 発見 | 修正 |
|---|---|
| ALERT が座席動詞経由で lead 座席不在時に不達(実機) | 7101351(identity 直送) |
| watcher 10 秒タイムアウト不足(実機) | 7101351(固定 60 秒・async single-flight) |
| デッドマン活動判定の org 未スコープ(Codex) | 19c7630 |
| 非アクティブ org への total-budget 誤 ALERT(Codex) | 19c7630 |
| agmsg history の非 lead トラフィックでデッドマン取り逃がし(P1, Codex) | 764b01f |
| `sent` イベント帰属モデルの反転(self-review H3-1) | 8bbd55e(sent=lead-by-construction) |

## Known gaps(tech-debt 登録済み)

- プローブ復旧時のデッドマン誤クリア(限定窓)/ watchdog join の transient 失敗ラチェット — cycle-3 cap 到達につき記録繰延(PR⑤ or 初回実運用事象時)
- realEscalate の osascript 経路はスモークのみ / L2-6(遮断後 Active フラグ非クリア)ほか deferred LOW 群
