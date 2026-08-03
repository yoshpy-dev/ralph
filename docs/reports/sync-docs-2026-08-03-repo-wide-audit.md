# Sync-docs report — リポジトリ全体ドリフト監査(standalone)

- Date: 2026-08-03
- Scope: main @ 32b523b(PR #141 マージ後)の全域スイープ(/sync-docs skill のチェックリスト 15 項目)
- 実施: doc-maintainer サブエージェントによる読み取り監査 → 本ブランチで修正

## 検出と修正

| # | 深刻度 | 対象 | 内容 | 修正 |
|---|---|---|---|---|
| 1 | HIGH | `internal/cli/root.go` | `ralph --help` が「autonomous pipeline CLI / runs autonomous development pipelines」と退役済み Ralph Loop を説明 | 「org-runtime CLI / coordinates autonomous multi-seat org-runtime execution (ralph org)」に更新 |
| 2 | MEDIUM | `internal/cli/init.go` | `ralph init --help` の「pipeline settings」(`[pipeline]` config は PR #140 で削除済み) | 「org-runtime config」に更新 |
| 3 | LOW | `docs/insights/README.md`(+templates ミラー) | Receipts セクション(PR #141)への言及ゼロ — AGENTS.md がスキーマ正本として指す文書で機能が見えない | 参照節を追記(正本は `--help` と model-routing.md である旨明記) |
| 4 | MEDIUM(付随) | `tests/test-no-loop-references.sh` | `grep -r` が gitignored の `.claude/agent-memory/` 等ローカル状態を拾い、main チェックアウトで誤 FAIL | `git grep`(tracked ファイルのみ)に変更 — CI が見得ない偽陽性を構造的に排除 |

## クリーン確認(ドリフトなし)

README.md / AGENTS.md / CLAUDE.md / `.claude/rules/` 全 14 ファイル / `.claude/skills/` 全 13 スキル / `.claude/agents/` モデルティア / docs/architecture/repo-map.md / docs/quality/ 両ファイル / docs/recipes/ 4 ファイル / docs/tech-debt/ の open row 引用(3 件抽査) / `.claude/settings.json` フック参照 / check-sync・check-skill-sync・check-pipeline-sync 全 green / `ralph org --help` と /org skill の整合。

## 検証

- `./scripts/run-verify.sh` → All verifiers passed(evidence: docs/evidence/verify-2026-08-03-041721.log、gitignored)
- `bash tests/test-no-loop-references.sh` → PASS(main チェックアウト相当の環境でも誤 FAIL しないことを確認)
- `./scripts/check-sync.sh` → PASS(insights README ミラー同期済み)
