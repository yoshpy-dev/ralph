# Walkthrough: spec-auto-invoke

- Date: 2026-07-14
- Plan: docs/plans/archive/2026-07-14-spec-auto-invoke.md
- Branch: feat/spec-auto-invoke
- Total diff: 20 files / +575 −20(うち実装・テスト・ドキュメント本体は約 70 行。残りはプラン+パイプラインレポート成果物)

## 変更の読み方(推奨順)

1. **トリガーポリシー本体** — `.claude/skills/spec/SKILL.md`
   frontmatter から `disable-model-invocation: true` を削除し、description の
   「Manual trigger only.」を自動起動条件(肯定: /plan には曖昧すぎるリポジトリ変更依頼
   / 否定: レビュー・Q&A・既存プラン実行・trivial fix・他スキル明示指定)に置換。
   本文は無変更。
2. **テンプレート同一コピー** — `templates/base/.claude/skills/spec/SKILL.md`
   ルートとバイト同一(`check-sync.sh` の強制)。
3. **Codex ミラー再生成** — `.agents/skills/spec/SKILL.md`、
   `templates/base/.agents/skills/spec/SKILL.md`(description 同期)。
   `agents/openai.yaml`(`allow_implicit_invocation: false`)は両側とも削除
   (`sync-skills.sh` の flag 撤去時クリーンアップ経路)。
4. **回帰テスト** — `tests/test-sync-skills.sh` Case G:
   flag 削除→再同期で `openai.yaml` が削除され、空の `agents/` も除去されること。
5. **ドキュメント整合(6ファイル)** — `CLAUDE.md`(手動トリガーは `/release` のみに)、
   `AGENTS.md` / `templates/base/AGENTS.md`(Primary loop: Spec (auto, optional))、
   `README.md`(ループ内は全ステップ auto)、`docs/architecture/repo-map.md`、
   `templates/base/CLAUDE.md`(scaffold は手動トリガースキルなし)。
6. **成果物** — プラン、self-review / verify / test / sync-docs / cross-review triage
   レポート、insight イベント。

## 検証サマリ

- `./scripts/run-verify.sh` PASS(check-sync / check-skill-sync / shellcheck / gofmt / staticcheck / go test)
- テンプレート側 `check-skill-sync.sh` PASS(12 skills in lock-step)
- `./scripts/run-test.sh` PASS(シェル37スイート + Go 10パッケージ、Case G 4/4)
- Codex cross-review: フィンディングなし
