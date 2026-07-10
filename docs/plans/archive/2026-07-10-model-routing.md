# model-routing

- Status: Approved
- Owner: Claude Code
- Date: 2026-07-10
- Related request: koalive で実装したモデルルーティング階層化（ai-dena/koalive PR #1046）の upstream 移植
- Related issue: N/A
- Type: chore
- Branch: chore/model-routing

## Objective

パイプラインサブエージェントと Ralph のモデル配分を「判断席 = opus、手続き席 =
sonnet」の階層に揃え、委譲基準を always-on ルールとして明文化する。ルート実装と
`templates/base/`（下流プロジェクトへの配布元）の両方に適用し、下流で同じ inherit
事故・フルモデルIDのステーレ化が再発しないようにする。

## Scope

1. `.claude/agents/` と `templates/base/.claude/agents/` の frontmatter:
   `verifier` / `tester` / `doc-maintainer` を `model: opus` → `model: sonnet`。
   `reviewer` は判断席として `model: opus` 維持
2. `.claude/rules/model-routing.md` と `templates/base/.claude/rules/model-routing.md`
   を新規作成（モデル階層表 + inherit 継承・`CLAUDE_CODE_SUBAGENT_MODEL`・
   キャッシュ失効・スケジュール実行・effort の落とし穴 + エイリアス使用方針）
3. `subagent-policy.md`（ルート + templates/base）に model-routing への参照1行
4. `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` の
   デフォルト: `claude-opus-4-7`→`opus`、`xhigh`→`high`（3値、両コピーは同一内容）
5. `templates/base/ralph.toml`: `model`/`effort`/`claude_reviewer_model` を同期
6. cross-review SKILL.md 4ミラー（ルート `.claude`/`.agents` +
   templates/base の両方）: `--model claude-opus-4-7` → `--model opus`
7. `docs/recipes/ralph-loop.md`（ルート + templates/base）のデフォルト表
8. `tests/test-ralph-config.sh` のデフォルト値アサーションを `opus`/`high` に更新
9. （実装中の追加）`internal/scaffold/embed_test.go` の
   `TestTemplateBaseRalphTomlHasLoopSection` が `claude_reviewer_model` の旧値を
   ピンしていたため期待値を `opus` に更新
10. （self-review HIGH 対応）`internal/config/config.go` の `Default()` も
    `opus`/`high`/`opus` に更新。`ralph run` が `cfg.Pipeline.Model` 等を
    `RALPH_*` 環境変数として export するため、Go 層を残すとシェル側の新デフォルトが
    実行時にマスクされる。`internal/config/config_test.go` と
    `internal/cli/doctor_loop_test.go` のフィクスチャも同期。ルート側
    model-routing.md に「シェル・toml・Go の3箇所を lock-step で更新」の注意を追記
    （テンプレート側コピーには Go 層が存在しないため追記しない。意図的な差分）

## Non-goals

- detect-changed-languages.sh の harness 分類追加（koalive 固有の detector 拡張。
  本リポジトリの detector は構造が異なり同問題が発生しない）
- `.codex/agents/`（GPT 系で動作、対象外）
- `docs/specs/` 内の歴史的記述の旧モデルID（仕様の記録として保持）

## Design decisions

koalive 側で AskUserQuestion により確定済みの決定をそのまま移植する:
- reviewer のみ opus 維持（セキュリティ判断席）、他3エージェントは sonnet
- モデル指定は安定エイリアス（`opus`/`sonnet`/`haiku`）に統一
  （フルIDのステーレ化と無効IDリスクを構造的に回避）
- effort は xhigh → high（公式が xhigh/max の収益逓減を明記）

Critical forks: None（上流での新規判断なし。決定済み内容の移植）

## Acceptance criteria

- [ ] ルートと templates/base の両方で verifier/tester/doc-maintainer が
      `model: sonnet`、reviewer が `model: opus`
- [ ] model-routing.md がルートと templates/base に存在（paths frontmatter なし）
- [ ] ralph-config.sh 実効値が `opus`/`high`/`opus`（両コピー md5 一致を維持）
- [ ] templates/base/ralph.toml が `opus`/`high`/`opus`
- [ ] `claude-opus-4-7`/`xhigh` が設定値として残存しない
      （docs/specs/・docs/plans/・docs/reports/ を除く）
- [ ] `./scripts/run-test.sh`（tests/ 一式）と `./scripts/run-verify.sh` が成功
- [ ] `./scripts/check-skill-sync.sh` が成功

## Verify / Test plan

- `./scripts/run-verify.sh`（静的検証、skill-sync 含む）
- `./scripts/run-test.sh`（test-ralph-config.sh のデフォルト値テスト更新を含む）
- grep による旧ID残存確認、`sh -c '. scripts/ralph-config.sh; echo ...'` の実効値証跡

## Risks and mitigations

- 下流テンプレート利用者のデフォルト挙動が変わる（opus-4-7/xhigh → opus/high）
  → README/recipes のデフォルト表を同期。環境変数上書きの優先順位は不変
- sonnet 化した verifier/tester の精度 → koalive 側で pipeline 全ステップ合格を確認済み。
  frontmatter 1行の revert で復旧可能

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
