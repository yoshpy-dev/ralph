# Walkthrough: codex-seat-permission-verification

- Date: 2026-09-18
- Plan: docs/plans/archive/2026-09-18-codex-seat-permission-verification.md(PR 作成時に active から移動)
- Issue: #155(Closes)
- Branch: docs/codex-seat-permission-verification(base: main @ e23c1f4)
- 差分規模: 22 files / +1675 −62(実体は evidence 1 本、recipe 1 本(2 コピー)、コメント更新 5 面、skill 4 面、エラー文言 2 行 + テスト 3 箇所。残りは plan と pipeline レポート 2 cycle 分)

## 読む順番

1. `docs/evidence/codex-seat-permissions-2026-09-18.md` — 実機検証の記録。環境と実効 codex config、判明した環境依存の問題 P1〜P5(shell alias の衝突、`ralph org send` の Enter、agmsg DB の writable root、`/tmp` は writable root、退役モデルの自動移行)、Run A / A2(autonomous = pass)、E1(edits、実 config 継承 = partial)、E2 / E2b(edits、最小 config = pass(機構)、承認プロンプトの実文言)、結論表、後始末。
2. `docs/recipes/codex-seat-permissions.md`(+ `templates/base/` に byte 同一)— 下流運用者向けの手順。前提(alias、agmsg writable root、`/tmp` 回避)、スクラッチ config、autonomous / edits の spawn と `--org-id` / `--config` / `--state-dir` を伴う後続コマンド、typed protocol の `task.txt` 例、観測項目、pass / partial / inconclusive の判定、opt-in の条件と再検証、後始末。
3. `templates/base/ralph.toml` / `scripts/ralph-config.sh`(+ template)/ `internal/config/config.go` / `internal/org/permissions.go` — `codex_verified` の位置づけを「未検証のための暫定制約(存在しない tech-debt 行を参照)」から「2026-09-18 に検証済み、既定 false は設計、マシンごとに recipe で opt-in、codex config を継承」に統一。`permissions.go` の非コメント差分は fail-closed エラー文言 2 行のみ(recipe を指す)。`permissions_test.go` の 3 箇所が追従。
4. `.claude/skills/org/SKILL.md`(+ 3 ミラー)— 前提節に shell alias の注意、permission 作法の codex 行を recipe 参照に。

## コミット単位

| SHA | 内容 |
|---|---|
| 7f299cc | evidence(実機検証の記録) |
| 0771073, 529b360 | recipe(初版)+ codex-setup See also + ralph.toml / permissions.go コメント + skill 4 面、template purity 修正 |
| 51df396, 7d07e2f | self-review cycle 1 の H1・M1〜M3・L1〜L7 と follow-up 4 点(config.go / permissions.go の古いコメント、recipe 英語化と edits 用 spawn、alias 注意の一般化、エラー文言) |
| c357cac | ralph-config.sh コメント |
| 33158e2 | cross-review AR-1: recipe の後続コマンドに `--org-id` / `--config` / `--state-dir` |
| f769f40 | self-review cycle 2 の LOW 4 点(`task.txt` 例、send / wait 分離、status の `--config`、折り返し) |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review レポート(cycle 1・2) |

## 設計判断(plan Design decisions より)

- 実機検証をこのセッションで行い(ユーザー決定)、`codex_verified` の既定は false のまま + 手順を recipe 化(ユーザー決定)。
- Codex plan advisory の 3 所見を採用: 否定テストの対象を writable root の外へ + 3 値判定、実効 approval policy / sandbox 設定の記録、実引数は manifest ではなく子プロセスから取得。
- cross-review cycle 1 の AR-1(recipe の後続コマンドのフラグ欠落)はユーザー判断で修正 + 全 pipeline 再実行(cycle 2/2)。cycle 2 は所見 0。
- 派生 issue 候補(PR 後に起票): shell alias の doctor 検出、`ralph org send` の Enter timing、agmsg DB writable root の自動付与 / doctor 検査、退役モデルの自動移行を receipts の `honored` で捕捉。

## レビューで特に見てほしい箇所

- evidence の判定(autonomous = pass、edits = pass(機構)/ E1 は partial)が recipe の判定基準と一致していること。
- recipe の各コマンドがそのまま実行できること(`--org-id` / `--config` / `--state-dir` の持ち回り、`task.txt` の typed protocol 形式)。
- `permissions.go` のロジックが不変であること(非コメント差分はエラー文言 2 行)。
