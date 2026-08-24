# Repo-wide sync-docs drift audit(2026-08-24)

- 対象: main@b343040(overlay-scaffold-v2 #143〜#147、Codex hooks #148/#149、v5.0.0 リリース、#150 4 イベント配線がすべてマージ済みの時点)
- 前回の全体監査: PR #142(pre-v5)
- 実施: doc-maintainer subagent による読み取り専用監査 → 本 PR で修正

## 検出ドリフト(3 件、すべて本 PR で修正)

| # | 深刻度 | ファイル | ドキュメントの記載 | 実際のコード | 修正 |
|---|---|---|---|---|---|
| 1 | MEDIUM | `docs/quality/quality-gates.md`(root + templates/base 両方) | `check-pipeline-sync.sh` は「8 reference files」を検査 | `REFS` は 6 件(work / cross-review SKILL、subagent-policy、definition-of-done、README、AGENTS)。8→6 への縮小(cross-review リネーム、CLAUDE.md 除外)時に散文が未更新 | 「6 reference files」へ訂正 |
| 2 | MEDIUM | `docs/architecture/repo-map.md` | CI/drift チェックのスクリプト群に `check-template-purity.sh` が無い | P5 で追加され `verify.local.sh` 経由で実行中 | 一覧へ追加 |
| 3 | LOW | `README.md` | `ralph org` サブコマンド表が `spawn/send/wait/read/stop/status/disband`(7 件) | 実 CLI は `start` / `report` / `watch` を含む 10 件(`report` は同 README の 3 行後の例で使用) | 10 件へ訂正 |

## クリーン確認済み領域

AGENTS.md repo-map のディレクトリ/スクリプト一覧、`.claude/rules/ralph/*.md` の `paths:` glob、
model-routing.md と `.claude/agents/*.md` frontmatter の整合、subagent-policy.md と
`.codex/agents/*.toml` の整合、`.claude/settings.json` と hooks 実体の整合、
`.codex/hooks.json` 4 イベント集合と両 README の整合、definition-of-done と `/work`
completion gate の整合、`docs/recipes/codex-setup.md`、`docs/insights/README.md` と
`insights-append.sh` フラグ(完全一致)、README の install/quick-start と v5.0.0 CLI、
Ralph-Loop/pipeline 時代の残存参照なし。

drift ゲート実行結果: `check-sync.sh` / `check-skill-sync.sh` / `check-pipeline-sync.sh` /
`check-template-purity.sh` すべて pass(check-sync の ROOT_ONLY 1 件は監査セッション自身の
session-lock アーティファクトで、追跡対象外)。

## 再発傾向のメモ

`docs/architecture/repo-map.md` のスクリプト一覧と `docs/quality/quality-gates.md` の
チェック記述は、スクリプト追加・リネーム時に散文が追随しない再発ドリフト源
(前回 #142 監査でも同型)。`scripts/check-*.sh` に触れる系列を閉じる際はこの 2 ファイルを
明示的に突合すること。
