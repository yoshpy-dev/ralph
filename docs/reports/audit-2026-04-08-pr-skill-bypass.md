# 監査メモ: /pr スキルバイパスの根本原因と再発防止

- Date: 2026-04-08
- Auditor: Claude Code
- Trigger: PR #3 で `/pr` スキルが使われず `gh pr create` を直接実行、テンプレート不使用・英語見出しで PR 作成

## インシデント概要

codex-triage-step の実装後、ポスト実装パイプラインで以下のスキップが発生:

| 期待されたステップ | 実際 | 原因 |
|-------------------|------|------|
| `/self-review` via reviewer サブエージェント | インライン実行 | API 過負荷（フォールバックポリシー準拠 — 正常） |
| `/verify` via verifier サブエージェント | インライン実行 | 同上 |
| `/test` via tester サブエージェント | インライン実行 | 同上 |
| `/sync-docs` via doc-maintainer サブエージェント | **スキップ** | エージェントが実行を忘れた |
| `/codex-review` (inline) | **スキップ** | エージェントが実行を忘れた |
| `/pr` via Skill tool | **`gh pr create` 直接実行** | エージェントがスキルを迂回した |

## 根本原因

### 直接原因
エージェントがパイプライン後半 (sync-docs → codex-review → /pr) を Skill tool 経由で呼ばず、`gh pr create` を直接実行した。

### 構造的原因
「auto-invoked」が **prose のみの約束** で、**deterministic enforcement がゼロ**。

- `/work` Step 9e は `→ /pr` と矢印で繋ぐだけで、「Skill tool を使え」とは明示していなかった
- `gh pr create` を直接叩いてもフックが止めない
- テンプレート使用義務が `/pr` SKILL.md 内にしかなく、スキルを迂回すると見えない

### 寄与要因
API 過負荷によるサブエージェント失敗後、フォールバック対応に注意リソースが割かれ、パイプライン後半のスキル呼び出しが抜け落ちた。

## 実施した修正

### 1. `gh pr create` をフックでブロック（deterministic enforcement）

`pre_bash_guard.sh` に追加:
```sh
*"gh pr create"*)
  emit_decision "deny" "Do not call 'gh pr create' directly. Use the /pr skill (Skill tool) instead — it enforces the Japanese PR template, pre-checks, and plan archiving."
  ;;
```

**効果**: エージェントが `gh pr create` を直接実行しようとした場合、フックが deny して `/pr` スキルの使用を強制する。

### 2. `/work` と `/loop` の Step 9 を明示化（prose reinforcement）

旧: `e. /codex-review (optional, inline) → /pr`

新:
```
e. /codex-review (optional, inline — findings are triaged before user presentation)
f. **Invoke /pr via the Skill tool** — do NOT run `gh pr create` directly.
```

**効果**: パイプラインの最終ステップとして `/pr` スキルの Skill tool 呼び出しを明示。

### 3. PR テンプレートの残存英語見出しを日本語化

`template.md` の `## Related` → `## 関連リンク`、`## Walkthrough` → `## ウォークスルー` に修正。

**効果**: テンプレートが全セクション日本語になり、スキル経由で使用される限り英語見出し混入がなくなる。

## 防御の多層性

| レイヤー | 機構 | 対象 |
|---------|------|------|
| L1: Hook (deterministic) | `pre_bash_guard.sh` が `gh pr create` を deny | 迂回を物理的にブロック |
| L2: Skill (prose) | `/work` Step 9f が Skill tool 使用を明示指示 | 正しい行動を促す |
| L3: Template (content) | `template.md` が全日本語見出し | スキル経由なら品質保証 |
| L4: Skill (pre-check) | `/pr` SKILL.md が「日本語で書け」と明記 | 最終防衛線 |

## 残存リスク

- `/sync-docs` と `/codex-review` のスキップは今回のフック追加では防げない（これらは `gh` コマンドではなく Skill tool の呼び忘れ）。ただし `/work` Step 9 の明示化で prose レベルでは対処済み。
- パイプライン全体の deterministic enforcement（例: 「全レポートが存在しないと /pr が進まない」）は `/pr` SKILL.md の pre-checks で部分的にカバーされている。
