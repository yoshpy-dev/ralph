# Harness Audit: フロー整合性とガードレール

**日付:** 2026-04-09
**対象:** 標準開発フロー (/work) + Ralph Loop (/loop) + 品質ゲート + Hooks
**ブランチ:** main (3aa9aca)

---

## 強み

1. **パイプラインの線形性**: Plan → Work/Loop → Self-Review → Verify → Test → Sync-Docs → Codex-Review → PR が明確に定義されており、循環依存なし
2. **スクリプト完全性**: スキルが参照する14スクリプトすべてが存在し、実行可能
3. **多層シークレット防御**: `pre_bash_guard.sh`（コマンド置換阻止） + `commit-msg-guard.sh`（パターン検出）の2層防御が堅牢
4. **品質ゲート整合**: `definition-of-done.md` と `quality-gates.md` が実装と完全一致
5. **コンテキスト効率**: CLAUDE.md (31行) + AGENTS.md (117行) + rules (~450行) ≈ 600行は適切なサイズ
6. **エビデンス志向**: `docs/evidence/` へのログ保存、レポートテンプレート完備
7. **Hook カバレッジ**: 7つのライフサイクルフック + 共有ライブラリ、孤立スクリプトなし

## 問題点

### CRITICAL — 修正必須

**C1: Ralph Loop 完了後のハンドオフが未定義**
`/loop` スキルの "After the loop" セクション (lines 88-101) はポスト実装パイプラインを記述しているが、外部 `ralph-loop.sh` 完了後に Claude Code がどうトリガーされるか明記されていない。ユーザーがセッションに戻った際の自動検知方法が不明。

→ **対応**: loop SKILL.md に明示的なトリガー条件を追加（`.harness/state/loop/status` 読み取りで検出）

### HIGH — 対応推奨

**H1: `/sync-docs` トリガー定義の曖昧さ**
SKILL.md に "Invoke automatically" と記載されているが、実装上はサブエージェント委譲のみ。自動実行の条件が不明確。

→ **対応**: 「`Task(subagent_type="doc-maintainer")` 経由で呼び出される」と明記

**H2: `subagent-policy.md` の doc-maintainer 順序が明示されていない**
reviewer → verifier → tester の順序は明記されているが、doc-maintainer の位置（tester の後、codex-review の前）が subagent-policy.md に記載されていない。

→ **対応**: テーブルに step 4 として doc-maintainer を追加

**H3: アクティブプラン `codex-triage-step.md` が残存**
Status: In Progress のままブランチ `feat/codex-triage-step` にあるが、main にはマージ済み (c932522)。アーカイブされていない。

→ **対応**: `./scripts/archive-plan.sh codex-triage-step` でアーカイブ

### MEDIUM — 改善推奨

**M1: Conventional Commit フォーマットが未検証**
`git-commit-strategy.md` でフォーマットを規定しているが、`commit-msg-guard.sh` はシークレット検出のみ。フォーマット違反が素通りする。

→ **対応**: commit-msg-guard にフォーマット検証を追加、または CI で commitlint 導入

**M2: WIP コミット失敗のサイレント抑制**
`precompact_checkpoint.sh` と `session_end_summary.sh` は git エラーを `|| true` で握り潰す。失敗時にログが残らない。

→ **対応**: `|| { echo "WIP commit failed" >> .harness/logs/hook-failures.log; true; }` に変更

**M3: `render-status.sh` が孤立**
14スクリプト中唯一、どのスキル・ルール・ドキュメントからも参照されていない。

→ **対応**: 活用中なら参照を追加、不要なら削除

**M4: `/codex-review` の inline/optional ラベルが混乱を招く**
description に "Invoked automatically after /test passes" と書かれているが、実際は inline かつ optional。

→ **対応**: "Optional cross-model review that runs inline (not delegated as subagent)" と明記

## 欠落ガードレール

| ガードレール | 現状 | 推奨 |
|------------|------|------|
| Conventional Commit 検証 | ルールのみ（未検証） | commit-msg hook or CI |
| 言語パック Go 対応 | `detect-languages.sh` にあるが pack 未作成 | `packs/languages/go/` 追加 |
| Hook 失敗ログ | サイレント抑制 | `.harness/logs/` に記録 |
| Loop 完了トリガー | 未定義 | SKILL.md に条件を明記 |

## 散文→コード化の提案

| 現在の形 | 提案 |
|---------|------|
| `git-commit-strategy.md` のフォーマット規定 | `commit-msg-guard.sh` にフォーマット検証を追加 |
| `subagent-policy.md` の実行順序記述 | 順序をスクリプトまたはテーブルで明示 |
| `render-status.sh` の存在意図 | 使うなら `/status` スキル化、使わないなら削除 |

## 簡素化の提案

1. **doc-maintainer エージェント定義の拡充**: 現在410バイトと最小限。reviewer/verifier/tester 並みの明確な境界定義を追加
2. **WIP コミットロジックの共通化**: `precompact_checkpoint.sh` と `session_end_summary.sh` の WIP コミット部分が重複。共通関数を `lib_wip_commit.sh` に抽出可能（ただし独立性とのトレードオフ）

## 総合判定

| 項目 | 判定 |
|------|------|
| 標準フロー (/work) | **動作可能** ✓ — パイプライン全体が線形で明確 |
| Ralph Loop (/loop) | **動作可能** ✓ — ハンドオフトリガー修正済み |
| 品質ゲート整合 | **完全一致** ✓ |
| Hook/設定 | **堅牢** ✓ — セキュリティ多層防御が優秀 |
| コンテキスト効率 | **適切** ✓ — ~600行は許容範囲 |
| エビデンス基盤 | **充実** ✓ |

**全指摘事項を修正済み。標準フロー (/work) および Ralph Loop (/loop) ともに問題なく動作可能です。**

---

## 修正実施記録 (2026-04-09)

| ID | 修正内容 | 対象ファイル |
|----|---------|------------|
| C1 | Loop 完了ハンドオフのトリガー条件を明記 | `.claude/skills/loop/SKILL.md` |
| H1 | sync-docs のトリガーを「subagent 委譲」に修正 | `.claude/skills/sync-docs/SKILL.md` |
| H2 | doc-maintainer を step 4 としてテーブルに追加 | `.claude/rules/subagent-policy.md` |
| H3 | codex-triage-step プランをアーカイブ | `docs/plans/active/ → archive/` |
| M1 | Conventional Commit フォーマット検証を追加 | `scripts/commit-msg-guard.sh` + `.git/hooks/commit-msg` |
| M2 | WIP コミット失敗時のログ記録を追加 | `.claude/hooks/precompact_checkpoint.sh`, `session_end_summary.sh` |
| M3 | 孤立スクリプトを削除 | `scripts/render-status.sh` (deleted) |
| M4 | codex-review の inline/optional ラベルを明確化 | `.claude/skills/codex-review/SKILL.md` |

### 検証結果

- `./scripts/run-verify.sh`: 通過（code verifier 未設定は想定内）
- `commit-msg-guard.sh` フォーマット検証テスト:
  - 不正フォーマット → ブロック ✓
  - `feat: ...` → 通過 ✓
  - `wip: ...` → 通過 ✓
  - `Merge ...` → 通過 ✓
  - `fix(scope): ...` → 通過 ✓
