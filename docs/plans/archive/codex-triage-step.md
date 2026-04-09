# codex-review トリアージステップの導入

- Status: In Progress
- Owner: Claude Code
- Date: 2026-04-08
- Related request: codex-review のレビューを Claude Code がトリアージして余計なノイズを除去する
- Related issue: N/A
- Branch: feat/codex-triage-step

## Context

Codex のコードレビューが粒度が細かすぎ、偽陽性やスタイル上の好みなど本質的でない指摘が混在している。現状の `/codex-review` は全指摘をそのままユーザーに提示し「修正する / PR 作成する」の二択を迫るため、ユーザーの判断負荷が高い。

ベストプラクティス調査の結果、以下の業界パターンが有効:
- **Semgrep**: 別軸の判定チェーン（TP/FP 分離）+ 保守的フィルタリング（96% 一致率）
- **Datadog**: LLM によるコンテキスト考慮型分類 + 信頼度ベース表示
- **Zylos Research**: 多段モデルレビューでのコーディネーター（トリアージ役）分離

これらを踏まえ、実装コンテキストを持つ Claude Code がトリアージ役となり、Codex 指摘を分類・フィルタリングしてからユーザーに提示する。

## Objective

`/codex-review` に「トリアージステップ」を追加し、Claude Code が実装コンテキスト（プラン、self-review 結果、設計意図）を用いて Codex 指摘を 3 段階に分類してからユーザーに提示する。

## Scope

1. `/codex-review` SKILL.md にトリアージロジックを追加
2. トリアージレポートテンプレートを新規作成
3. `subagent-policy.md` にトリアージのインライン実行根拠を文書化
4. `/work` と `/loop` のフロー参照を微修正

## Non-goals

- Codex の呼び出しコマンド (`codex exec review`) の変更
- 新しいサブエージェント定義の作成（トリアージはインライン実行）
- self-review の SKILL.md や reviewer エージェントの変更
- `/plan` スキルの Codex plan advisory（Step 6.5）の変更
- 既存の severity 体系（CRITICAL/HIGH/MEDIUM/LOW）の変更

## Assumptions

- Codex が構造化された指摘（severity + affected files + recommendation）を返す
- self-review レポートがトリアージ時点で `docs/reports/` に存在する
- トリアージはメインコンテキスト内で完結する（サブエージェント不要）

## Affected areas

| ファイル | 変更内容 |
|---------|---------|
| `.claude/skills/codex-review/SKILL.md` | トリアージステップ追加、AskUserQuestion ロジック改修 |
| `docs/reports/templates/codex-triage-report.md` | 新規作成 |
| `.claude/rules/subagent-policy.md` | Codex triage セクション追加 |
| `.claude/skills/work/SKILL.md` | Step 9e の参照微修正 |
| `.claude/skills/loop/SKILL.md` | Step 3e の参照微修正 |

## Design decisions

### トリアージ分類: 3 段階

| 分類 | 意味 | ユーザー提示 |
|------|------|-------------|
| `ACTION_REQUIRED` | 実際の問題（セキュリティ、正確性、データ損失） | 最初に表示、修正を推奨 |
| `WORTH_CONSIDERING` | 妥当だが文脈上議論の余地あり（設計判断、エッジケース） | 次に表示、判断を委ねる |
| `DISMISSED` | 偽陽性、既対処済み、スタイル好み、スコープ外 | レポートに記録（透明性）、ユーザーには件数のみ |

4 段階は不採用: self-review が既に CRITICAL/HIGH/MEDIUM/LOW を使っており、Codex トリアージは意図的に簡素化して判断コストを下げる。

### トリアージの実行場所: インライン（メインコンテキスト）

理由: トリアージの価値は「なぜそのコードにしたか」という実装コンテキストに依存。サブエージェントにはこのコンテキストがなく、偽陽性の適切な判別ができない。

### 2 軸評価（Semgrep パターン準拠）

各指摘を 2 つの独立した軸で評価:
1. **Axis 1**: これは実際の問題か？（正確性、セキュリティ、信頼性）
2. **Axis 2**: 実装コンテキスト上、修正する価値があるか？（プランの非ゴール、既対処、コスト対効果）

### 保守的原則

不確実な場合は上位に分類: DISMISSED → WORTH_CONSIDERING → ACTION_REQUIRED。指摘を暗黙的に削除しない。

## Acceptance criteria

- [ ] `/codex-review` SKILL.md に Step 3 (Triage) と Step 4 (Write triage report) が追加されている
- [ ] トリアージは 2 軸評価（real issue? + worth fixing?）で各指摘を分類する
- [ ] 分類結果に応じて AskUserQuestion の選択肢が分岐する（ACTION_REQUIRED あり/なし/全 DISMISSED）
- [ ] DISMISSED 指摘も理由カテゴリ付きでトリアージレポートに記録される（透明性）
- [ ] `docs/reports/templates/codex-triage-report.md` が作成されている
- [ ] `subagent-policy.md` に Codex triage のインライン実行根拠が文書化されている
- [ ] `/work` と `/loop` の参照が更新されている
- [ ] 保守的原則（不確実時は上位に分類）が SKILL.md に明記されている

## Verify plan

- Static analysis checks: N/A（Markdown ファイルのみ）
- Spec compliance criteria to confirm:
  - SKILL.md のステップ番号が連番で矛盾がないか
  - subagent-policy.md のインライン記述と SKILL.md が整合するか
  - /work と /loop の参照が SKILL.md の実際の動作と一致するか
- Documentation drift to check:
  - CLAUDE.md、AGENTS.md の codex-review 言及が新しいフローと矛盾しないか
- Evidence to capture: 変更後の全ファイルの diff

## Test plan

- Unit tests: N/A（ワークフロー定義の変更のみ）
- Integration tests: N/A
- Regression tests: 既存の codex-review 動作（Codex 不可時のスキップ、diff なし時のスキップ）が維持されていることを SKILL.md 上で確認
- Edge cases:
  - Codex 指摘が 0 件の場合のフロー
  - 全指摘が DISMISSED の場合のフロー
  - ACTION_REQUIRED のみの場合のフロー
  - Codex が非構造化出力を返した場合のフォールバック
- Evidence to capture: SKILL.md の diff

## Risks and mitigations

| リスク | 軽減策 |
|--------|--------|
| トリアージが実際の問題を DISMISSED に分類 | 保守的原則の明記 + DISMISSED に dismissal reason 必須 |
| コンテキストトークン消費 | トリアージ理由を各指摘 1-2 文に限定、レポートはファイル出力 |
| Codex が非構造化出力を返す | トリアージスキップ → 現行動作（全指摘提示）にフォールバック |

## Rollout or rollback notes

- ロールアウト: SKILL.md の上書きで即時有効
- ロールバック: git revert で旧 SKILL.md に復元可能
- 影響範囲: `/codex-review` を使う全フロー（/work、/loop）

## Open questions

なし

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
