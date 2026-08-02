# Walkthrough: org-runtime-mechanism (PR①)

- Date: 2026-08-02
- Plan: docs/plans/archive/2026-08-01-org-runtime-mechanism.md(アーカイブ後)
- Spec: docs/specs/2026-08-01-org-runtime.md
- Diff: 41 files, +6341/-5(main...HEAD)

## 読む順序(推奨)

1. **docs/specs/2026-08-01-org-runtime.md** — 全体像。本 PR は 5 段 PR 系列の①(機構層)のみ。
2. **docs/plans/active(→archive)/2026-08-01-org-runtime-mechanism.md** — PR① のスコープ・AC・設計判断(Codex advisory 5 件の反映を含む)。
3. **internal/config/config.go** — `[org]` エンベロープ(model_pool / driver_pool / roles / max_seats / budget)。既存の allowlist バリデーション流儀に追従。3 面ロックステップ(config.go / templates/base/ralph.toml / scripts/ralph-config.sh)は `defaults_sync_test.go` が執行。
4. **internal/org/**(コア、外部プロセスなし)
   - `seat.go` — saga 状態(state イベント vs 非 state イベントの区別が肝)
   - `manifest.go` — org_id 名前空間付き JSONL ストア。Roster 導出は「最新 state イベント」ベース。dry-run は実座席と別キーで完全分離
   - `envelope.go` — `ValidateSpawnEnvelope`(無状態)+ `ValidateSpawnCapacity`(容量)の分割。分割理由は cycle-2 self-review MEDIUM 1(副作用順序)
   - `receipts.go` — 三値 receipts(commanded / reported_effective / honored: true|false|unknown)
   - `spawn.go` — spawn saga 本体。順序: 冪等 early return → 無状態検証 → stale 補償 → 容量検証 → saga 副作用。この順序は cross-review ACTION_REQUIRED #1 と cycle-2 self-review MEDIUM 1 の 2 回の修正を経た最終形
   - `verbs.go` — send / wait / read / stop / status / disband
5. **internal/org/driver/** — herdr / agmsg の exec アダプタ。`Runner` インターフェースでスタブ注入可能。agmsg のフラグ形状は未確認仮定として `agmsg.go` に一元化(実 CLI 初回利用前に要検証)
6. **internal/cli/org.go** — cobra 配線(薄い)。`--dry-run` / `--org-id` / `--state-dir`
7. **internal/cli/doctor.go** — `info` ステータス新設(exit code 非影響)+ `--probe-models`
8. **テスト** — スタブバイナリ(PATH 注入)+ failure-injection が `internal/cli/org_test.go`、saga 単体が `internal/org/spawn_test.go`

## 設計上の要点

- **純追加**: 既存フロー(/work・Ralph Loop)のコードパスに分岐ゼロ。`ralph org` を呼ばない限り不可視(AC-9)。
- **監査の正本は manifest**: 全動詞が `.harness/state/org/manifest.jsonl` に自動追記。LLM 全停止でも `ralph org status` が状態説明可能(AC-4)。
- **spawn は saga**: `spawn_started` → `spawn_step`(外部 ID 永続化)→ `spawned` / `spawn_failed` + 補償記録(AC-10)。
- **herdr エージェント名は org_id 名前空間付き**(`<org_id>-<seat_id>`)— self-review HIGH の修正。

## レビュー往復の履歴(品質ゲート)

| Cycle | ゲート | 結果 |
|---|---|---|
| 1 | self-review | HIGH 1(herdr 名前空間)→ 9bfe07e で修正 |
| 1 | verify / test | PASS / PASS(181 テスト) |
| 1 | cross-review (Codex) | ACTION_REQUIRED 1(冪等順序)→ 4dcfc03 で修正 |
| 2 | self-review | MEDIUM 2(修正が生んだ副作用順序退行ほか)→ e6a162c で修正 |
| 2 | verify / test | PASS / PASS(222 テスト、-race クリーン) |
| 2 | cross-review (Codex) | ACTION_REQUIRED 0。残 1 件は既知 TOCTOU(Known gaps) |

## Known gaps(意図的繰延)

- 並行 spawn の TOCTOU(read→validate→append 非原子)— docs/tech-debt 登録済み、Lead が並行 spawn する PR③ までに flock 等で対応。
- `Verbs.Send/Wait/Read` のテストカバレッジ 0% — PR②(座席化)で実挙動と合わせて閉じる。
- agmsg CLI フラグ形状の実機検証 — PR② の実バイナリ統合時。
