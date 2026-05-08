---
name: self-review
description: Self-review the diff for code quality before formal verification. Covers naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, and maintainability. Invoke automatically after /work completes or when significant code changes are staged.
---
Perform a self-review of the current diff and write a report to `docs/reports/`.

## Review scope — diff quality only

Focus exclusively on the diff itself. Do NOT evaluate spec compliance, test coverage, or documentation drift — those belong to `/verify` and `/test`.

Evaluate the diff for:
1. **Unnecessary changes** — unrelated modifications, formatting-only diffs, accidental includes
2. **Naming** — clarity, consistency with surrounding code, grep-ability
3. **Readability** — function length, nesting depth, comment quality
4. **Typos and copy-paste errors**
5. **Null safety and defensive checks** — missing guards at boundaries
6. **Debug code** — leftover console.log, print, TODO markers, commented-out code
7. **Secrets and credentials** — hardcoded keys, tokens, passwords
8. **Exception handling** — swallowed errors, generic catches, missing error paths
9. **Security** — injection risks, XSS, CSRF, unsafe deserialization, path traversal
10. **Maintainability** — tight coupling, hidden side effects, magic numbers

## Review method

1. Inspect the active plan and changed files via `git diff`.
2. Prefer evidence from the diff and repository contracts over intuition.
3. Record findings in a report using [template.md](template.md).
4. Separate blocking issues from follow-up suggestions.
5. If any finding represents deferred work, known shortcuts, or accumulated complexity, append it to `docs/tech-debt/README.md` or create a dedicated file in `docs/tech-debt/`.
6. If there are no findings, say what was checked and what evidence supports that conclusion.

## Output

- `docs/reports/self-review-<date>-<slug>.md`
- severity-tagged findings
- merge or no-merge recommendation
- tech-debt entries in `docs/tech-debt/` if deferred work was identified

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
