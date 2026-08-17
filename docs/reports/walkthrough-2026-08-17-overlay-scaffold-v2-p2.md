# Walkthrough: overlay-scaffold-v2 Phase 2(テンプレート v2 再編 + init + root 移行)

- Date: 2026-08-18
- Branch: feat/overlay-scaffold-v2-p2
- Diff: 136 files, +4,614 / -379(コミット 37 件)
- Plan: docs/plans/archive/2026-08-17-overlay-scaffold-v2-p2.md(PR 時にアーカイブ)
- Spec: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 1 = PR #143)

## 読む順番(レビュー導線)

1. **templates/base/ の v2 レイアウト**(slice 1–2)— ralph ガイダンスを `.claude/rules/ralph/` へ移設(`ralph-workflow.md` 新設、CLAUDE.md は最小シード化)。`AGENTS.md` は「ユーザ所有スケルトン + managed block」で、block 生成元は `.ralph/core/AGENTS.core.md`(56 行、meta 記述なし)。`.gitignore` は `#` 形式 managed block(スペック FR-5 に追補済み)。
2. **hooks dispatcher**(`templates/base/.claude/hooks/ralph-dispatch.sh`)— settings.json の各イベントは dispatcher 1 エントリのみ。実装は現位置維持、`.d/` は `exec` 薄シム。JSON マージ意味論(decision 先勝ち・additionalContext 集約・単一出力はバイト透過)、シグナル処理(INT/TERM/HUP で子 hook を kill → cleanup → exit、EXIT trap 併用)。テスト: `tests/test-ralph-dispatch.sh` 26 ケース(実ペイロード、SIGTERM 実効性の正負プローブ済み)。
3. **`.ralph/local/` drop-in**(L3 overlay)— `hooks/<event>.d/`・`verify.d/`・`test.d/`。`run-verify.sh` が HARNESS_VERIFY_MODE に応じて実行。
4. **`ralph init` v2**(`internal/cli/init.go`、slice 3)— manifest v3(`layout="v2"` + 全エントリ owner: block/seed/core、`.ralph/local/**` は seed)。既存 `AGENTS.md`/`.gitignore` へは block 追記のみ(block 外バイト保全、malformed/symlink/非 regular は warn + 据え置き)。v2 既存プロジェクトへの再 init は no-op。
5. **パック追従 + 過渡ガード**(slice 4)— pack rule は `.claude/rules/ralph/<lang>.md` へ。旧 upgrade エンジンは `layout="v2"` 検出で書き込みゼロの fail-closed(AC-10)。`ralph pack add` も v2 時に owner 付与。
6. **root メタリポジトリ同時移行**(slice 5)— テンプレートと同構造へ。`check-sync.sh` は AGENTS.md を block-aware 比較(block 内容 = AGENTS.core.md 一致、block 外は所有者自由)に変更し、KNOWN_DIFFS から AGENTS.md を解除(Phase 1 tech-debt 回収)。`tests/test-hook-wiring.sh` 新設(Claude/Codex 両 hook コマンドの実在検査)。
7. **doctor 追従** — hooks integrity がコマンド文字列の実行ファイルトークンのみを stat(dispatcher の「パス + 引数」形式対応)。
8. **封じ込めガード** — dangling symlink 対策を RenderFS(`os.Lstat`)と `renderMappedFile` の両層に適用。

## 品質ゲートの履歴

パイプライン 4 サイクル(cap 2→3→4 と 2 回引き上げ、いずれもユーザ承認):

| cycle | self-review | verify | test | cross-review(codex) |
|---|---|---|---|---|
| 1 | HIGH 1 + MEDIUM 6 → 修正 | PASS | PASS(593 shell) | AR 2 件 → 修正 |
| 2 | MEDIUM 5 → 修正 | PASS | PASS(595) | AR 2 件(doctor 偽 fail / symlink)→ 修正 |
| 3 | MEDIUM 2 → 修正 | PASS | PASS(596) | AR 2 件 + WC 1 件(子プロセス kill / mapped Lstat / insight 誤記)→ 修正 |
| 4 | MEDIUM 1(doc)→ 修正 | PASS | PASS(599 shell + Go 8/8) | **指摘ゼロ(収束)** |

計 7 件の cross-review 指摘を全て解消。最終カバレッジ: internal/cli 79.0% / scaffold 78.9% / upgrade 90.0%。

## Known gaps(docs/tech-debt/README.md 追跡)

- Codex 側 hook surface は dispatcher/.d 未適用(Claude Code 側のみ。Phase 5 のパリティ作業)
- `check-sync.sh` の block-aware DRIFTED 分岐に隔離 fixture テストなし
- `pre_bash_guard.sh` が実ペイロードの `.tool_input.command` でなく `.command` を読む既存バグ(本 PR 由来ではない)
- 旧 upgrade エンジンはパック rule 移設を remove+add と扱う(rename 検出は Phase 4 の移行分類器)

## 過渡状態の注記

Phase 2 時点では「init は v2 を生成、upgrade は v2 に fail-closed」— v2 の upgrade 配線は Phase 3。リリースタグは系列完了まで切らないため下流には未到達。
