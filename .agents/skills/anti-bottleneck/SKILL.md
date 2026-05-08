---
name: anti-bottleneck
description: Load this skill BEFORE asking the user for confirmation, approval, next steps, or choices that can be resolved through verification, repo context, or reasonable defaults. Also load when you are unsure how to proceed and might otherwise stop early.
user-invocable: false
---
Human attention is the scarcest resource in the harness.

Before asking the user anything, check whether you can:
- inspect the codebase
- inspect plans or docs
- run tests or verification scripts
- gather logs or screenshots
- use a focused subagent
- choose a reasonable default and document it

Escalate only for:
- irreversible destructive actions
- external approvals that are genuinely required
- missing credentials or secrets
- product or design judgments that cannot be grounded in the repo or evidence

When stuck:
1. reduce scope
2. gather evidence
3. update the plan
4. try a different verifier or reviewer
5. present the best grounded answer you can

## CLI 別実行ガイダンス

このスキルは Claude Code と Codex の両方で動作する。実行モードは AGENTS.md と
`.codex/AGENTS.override.md` の規約に従う。

| 観点 | Claude Code | Codex |
|------|-------------|-------|
| Skill 起動 | `/skill-name` slash command | `$skill-name` mention または `/skills` メニュー (`/skill-name` 形式は built-in 衝突のため使わない) |
| Skill 本体パス | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| サブエージェント | `Task(subagent_type=...)` で並列呼び出し可 | 順次 inline 実行 — 単一 agent 内で連続実行 |
| 構造化対話 | `AskUserQuestion` | 番号付き選択肢を stdout に出して数字を待機 |
| 成果物 | `docs/reports/`, `docs/plans/`, `docs/specs/` 共通 | 同左 (CLI 非依存) |

drift check (`./scripts/check-skill-sync.sh`) は両側の本文と起動メタデータを
照合する — 片側だけ編集すると CI で fail する。
