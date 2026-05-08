---
name: sync-docs
description: Sync plans, docs, and instruction files after behavior, commands, contracts, or workflows change. Also covers harness-internal consistency after skill, hook, rule, or script changes. Invoked as a delegated subagent task via Task(subagent_type="doc-maintainer") in the post-implementation pipeline, after /test and before /cross-review.
---
Use this skill when implementation or harness structure changed enough that documentation may have drifted.

## Product-level sync

Update, as needed:
- active plan progress
- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.claude/rules/`
- `docs/quality/`
- `docs/reports/` links or references

Keep `AGENTS.md` short and stable. If a new rule is path- or topic-specific, put it in `.claude/rules/` instead.

## Harness-internal sync

When skills, hooks, rules, scripts, or language packs changed, also check:

- **Skills added/removed/renamed**: Does `AGENTS.md` Repo map still reflect the skill set? Does `README.md` list the current operating loop?
- **Hooks added/removed**: Does `.claude/settings.json` reference the correct hook scripts? Are removed hooks cleaned out?
- **Rules added/removed**: Does `.claude/rules/` match the languages and topics actually in the project? Are `paths:` globs still accurate?
- **Language packs added/removed**: Does `scripts/detect-languages.sh` detect the language? Is there a matching `.claude/rules/<lang>.md`? Does `packs/languages/<lang>/verify.sh` run a real verifier (not the placeholder)?
- **Scripts added/removed**: Does `README.md` Quick Start still reference valid scripts? Does `docs/architecture/repo-map.md` list the current scripts?
- **Quality gates changed**: Does `docs/quality/definition-of-done.md` match the actual completion workflow in `/work`? Does `docs/quality/quality-gates.md` list verifiers that actually exist?
- **PR skill consistency**: Does `/pr` SKILL.md pre-checks align with `/self-review`, `/verify`, and `/test` output? Does the PR template match the current plan template fields? Does `AGENTS.md` primary loop include the PR step?

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
