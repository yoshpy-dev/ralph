# spec-skill

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-12
- Related request: 抽象的な指示を詳細仕様に落とす /spec スキルの新規作成
- Related issue: N/A
- Branch: feat/spec-skill

## Objective

抽象的・曖昧なプロンプトを受け取り、コードベース調査・Web検索・ユーザー対話を通じて、実装可能な詳細仕様書（spec）に変換する新規スキル `/spec` を作成する。

## Scope

- `.claude/skills/spec/SKILL.md` — スキル定義（allowed-tools 明記）
- `.claude/skills/spec/template.md` — 仕様書テンプレート
- `CLAUDE.md` — `/spec` の位置づけ追記
- `AGENTS.md` — Primary loop への `/spec` 追加

## Non-goals

- `/plan` スキルの変更・統合（役割分離を維持）
- 新規エージェント定義（`/spec` はインライン実行）
- CI/CD への組み込み
- `docs/specs/` ディレクトリのサンプルファイル作成

## Role separation: /spec vs /plan

| 観点 | /spec | /plan |
|------|-------|-------|
| 入力 | 曖昧なアイデア・抽象的な指示 | 明確な仕様・定義されたタスク |
| 焦点 | **何を作るか**（要件・制約・仕様） | **どう作るか**（実装戦略・ファイル構成） |
| 出力 | 仕様書（docs/specs/*.md）/ GitHub issue | 実装計画（docs/plans/active/*.md） |
| 調査 | コードベース探索・Web検索・ベストプラクティス | 影響ファイル特定・リスク分析 |
| ユーザー対話 | 積極的に質問（要件明確化） | フロー選択（標準/Ralph） |

## Assumptions

- `/spec` は手動トリガーのみ（`disable-model-invocation: true`）
- 仕様書は `docs/specs/` に保存
- `/spec` 完了後、ユーザーの選択で `/plan` に遷移可能
- GitHub issue 作成は `gh issue create` CLI 経由（失敗時はファイル保存にフォールバック）

## Affected areas

- `.claude/skills/spec/` (新規)
- `CLAUDE.md` (追記)
- `AGENTS.md` (追記)

## Acceptance criteria

- [ ] `.claude/skills/spec/SKILL.md` が以下の frontmatter フィールドを持つ: `name: spec`, `description`, `disable-model-invocation: true`, `allowed-tools` (Task, Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion, WebSearch, WebFetch, Skill)
- [ ] `.claude/skills/spec/template.md` が以下のセクションを含む: Overview, Background, Requirements (functional/non-functional), Constraints, User stories, Dependencies, Research findings, Open questions, References
- [ ] SKILL.md のステップに Explore エージェント呼び出し（`Task(subagent_type="Explore")`）が含まれる
- [ ] SKILL.md のステップに WebSearch / WebFetch の使用が含まれる
- [ ] SKILL.md のステップに AskUserQuestion による要件明確化ループが含まれる
- [ ] SKILL.md の最終ステップに 4 択の AskUserQuestion が含まれる（issue のみ / file のみ / file+issue / file+plan）
- [ ] issue 作成パスが `gh issue create --title ... --body ...` で定義されている
- [ ] issue 作成失敗時のフォールバック（ファイル保存 + 警告メッセージ）が定義されている
- [ ] plan 遷移パスが `Skill(skill="plan")` で定義されている
- [ ] `CLAUDE.md` の Default behavior セクションに `/spec` の説明が 1-2 行で追記されている
- [ ] `AGENTS.md` の Primary loop に `/spec` がステップ 1.5 として追加されている

## Implementation outline

1. `.claude/skills/spec/template.md` を作成（仕様書テンプレート）
2. `.claude/skills/spec/SKILL.md` を作成（スキル定義 — allowed-tools 明記、gh CLI パス定義、フォールバック定義）
3. `CLAUDE.md` に `/spec` の位置づけを追記（Default behavior セクション）
4. `AGENTS.md` の Primary loop に `/spec` を追加

## Verify plan

- Static analysis checks: SKILL.md の frontmatter に `name`, `description`, `disable-model-invocation`, `allowed-tools` が存在すること（grep で確認）
- Spec compliance criteria to confirm:
  - template.md のセクション見出しが acceptance criteria のリストと一致
  - SKILL.md に 4 択 AskUserQuestion のオプションテキストが含まれる
  - SKILL.md に `gh issue create` コマンドが含まれる
  - SKILL.md に `Skill(skill="plan")` 遷移が含まれる
- Documentation drift to check: CLAUDE.md と AGENTS.md が `/spec` を正しく参照
- Evidence to capture: `grep -c` 結果

## Test plan

- Unit tests: N/A（設定ファイルのみ）
- Integration tests: `/spec` を手動実行して仕様書生成を確認（手動検証）
- Regression tests: 既存 `/plan` スキルが影響を受けないことを確認（diff で変更なし）
- Edge cases: 引数なしで `/spec` を実行した場合の挙動
- Evidence to capture: 手動実行時の出力サンプル

## Risks and mitigations

| リスク | 影響 | 対策 |
|--------|------|------|
| /spec と /plan の境界が曖昧になる | ユーザー混乱 | 役割分離テーブルを SKILL.md に明記 |
| AskUserQuestion の過剰使用 | ボトルネック化 | anti-bottleneck ルールを適用 |
| gh CLI 未認証でissue作成失敗 | 仕様書が保存されない | ファイル保存を常にフォールバックとして実行 |
| allowed-tools に不足がある | スキル実行時にツールブロック | 全必要ツールを frontmatter に列挙 |

## Rollout or rollback notes

- 新規ファイルの追加のみ。既存スキルへの変更は CLAUDE.md/AGENTS.md の追記のみ。
- ロールバック: `.claude/skills/spec/` 削除 + CLAUDE.md/AGENTS.md の追記部分を revert

## Open questions

（なし — ユーザーからの回答で全て解決済み）

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] Sync docs completed
- [ ] PR created
