# Walkthrough: org-runtime-lead (PR③)

- Date: 2026-08-02
- Plan: docs/plans/archive/2026-08-02-org-runtime-lead.md(アーカイブ後)
- Spec: docs/specs/2026-08-01-org-runtime.md(PR 系列③)
- Diff: 44 files, +4,992/-202(main...HEAD)

## この PR の性格

PR①(機構)・PR②(座席実機統合)の上に「**Lead が組織を自律編成できる**」を載せる段。座席の権限問題(PR② で座席が権限ダイアログで blocked)をエンベロープ設定で解消し、headless Lead(`ralph org start`)と Lead の操作マニュアル(`/org` skill)を追加、繰延 tech-debt 6 件をクローズした。実機スモークで「autonomous 座席が typed TASK の bash をダイアログなしで実行し protocol 準拠 RESULT を返す」「headless lead が 50 秒で編成 E2E(spawn→send→status→report→disband)を完遂する」ことを証明済み。

## 読む順序(推奨)

1. **docs/plans/archive/2026-08-02-org-runtime-lead.md** — スコープ・Codex advisory 4 件の反映(autonomous 最小制御ゲート、codex fail-closed 等)・実機発見 3 件+cycle-2 修正の全記録。
2. **internal/config/config.go + internal/org/permissions.go** — `[org.permissions]`(driver 非依存 mode: autonomous/edits/guarded、役割別上書き、既定 autonomous)と driver ネイティブフラグ変換(claude: `--permission-mode`)。**codex は guarded 以外を fail-closed で拒否**(実機検証まで「サイレント no-op」を構造的に排除)。
3. **internal/org/spawn.go** — 最重要ファイル。順序は「識別子/結合長 → [ロック] 冪等 return → 無状態エンベロープ → permission → **AC-2b scope ゲート**(autonomous は --scope 必須 fail-closed、`--allow-unscoped` で明示解除・記録)→ [ロック解放] stale 補償 → [再ロック] 冪等再チェック+容量 → saga」。flock による TOCTOU 直列化(tech-debt #4)、announce 失敗時の Leave 補償(#1)、`LeadIdentity` 定数(#2)、`herdr_agent_name` 永続化(#3)、dry-run エラー伝播(#6)。lead 自己 spawn(`org start`)は lead join を座席 join と統合し HELLO を省略。
4. **internal/org/prompts/lead.md + internal/cli/org.go start** — headless Lead は「役割 lead の座席」という糖衣(機構の対称性維持。PR④ Watchdog も通常座席として監視可能)。
5. **internal/org/report.go + `org report`** — manifest+receipts から編成履歴(roster/permission 列、タイムライン、三値 receipts、残留)を `docs/reports/org-manifest-*.md` に成果物化(FR-9 後半)。
6. **.claude/skills/org/SKILL.md**(4 面ミラー)— Lead の正準マニュアル: 動詞リファレンス、編成パターン(Solo/Leaded/Parallel)、2 運用経路、前提(bypass 初回承諾・同一 cwd 規約)。
7. **docs/evidence/org-lead-smoke-2026-08-02.txt** — 実機証拠。

## 実機発見(すべて plan の deviations に記録)

| # | 発見 | 対応 |
|---|---|---|
| 1 | `bypassPermissions` はマシンごと初回 1 回の承諾ダイアログ | 自動承諾はしない。/org skill の前提に明記 |
| 2 | herdr は入力待ち対話エージェントを `done` と報告(`idle` でない) | send/wait の待機を `idle,done` に(d35f157) |
| 3 | state-dir は cwd 相対 → lead と operator の cwd 分裂で manifest 分裂 | 同一 cwd 規約を /org skill に明記、恒久対応は tech-debt(PR④) |

## 品質ゲート履歴

| Cycle | ゲート | 結果 |
|---|---|---|
| 1 | self-review | HIGH 1(wait の idle 待ち残り)+ MEDIUM 4 → 9a22942 で修正 |
| 1 | verify / test | PASS / PASS(978 shell + 345 Go、org 90.0%) |
| 1 | cross-review (Codex) | ACTION_REQUIRED 1(scope ゲートが冪等より前)→ de4de50 で修正 |
| 2 | self-review | MEDIUM 2(dry-run 順序不一致・Phase2 冪等再チェック)→ 69be944 で修正 |
| 2 | verify / test | PASS / PASS(org 90.3%) |
| 2 | cross-review (Codex) | **指摘なし(クリーン)** |

## Known gaps(tech-debt 登録済み)

- state-dir の cwd 相対(同一 cwd 規約で回避中、PR④ で設計)/ evidence 赤入れ規約 / /org skill の説明文ほか batchable LOW 群 / codex 権限 fail-closed の解除条件(実機検証待ち)。
