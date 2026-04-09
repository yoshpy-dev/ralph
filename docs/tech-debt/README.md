# Tech debt

Record debt that should not disappear into chat history.

Recommended fields:
- debt item
- impact
- why it was deferred
- trigger for paying it down
- related plan or report

## Entries

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| ~~CLAUDE.md line 14 の "proceed through /self-review, /verify, /test" がsubagent委譲を明示していない。line 21 の新ポリシーと表面上矛盾する。~~ **[RESOLVED: feat/ralph-loop-v2 で解消。line 13-15 がパイプライン/標準モードを明示的に区別するよう更新済み]** | ~~新規読者が line 14 と line 21 を別フローと解釈するリスク~~ | ~~今回のスコープはline 21のみ変更。line 14の修正は計画の非ゴール~~ | ~~CLAUDE.md 次回編集時、または混乱報告が発生したとき~~ | docs/reports/self-review-2026-04-08-subagent-trigger-policy.md |
| ~~`ralph-orchestrator.sh` の pipe-subshell 変数スコープバグ 3箇所~~ **[PARTIALLY RESOLVED: 依存関係チェック、統合マージチェックは temp file ベースに修正済み。scripts/ralph:294 の abort ワークツリーリストは未修正]** | ~~HIGH~~ LOW: abort 時のワークツリーリストのみ残存 | 依存関係と統合マージは修正済み。abort 時の影響は監査ログの精度のみ | scripts/ralph の abort コマンドをリファクタリングするとき | docs/reports/self-review-2026-04-09-ralph-loop-v2.md |
| `ralph-pipeline.sh` の CRITICAL self-review 発見を無視するポリシー (line 421: "Don't stop — let verify and test catch real issues") が AGENTS.md および subagent-policy.md の契約と矛盾する。意図的な逸脱だが計画に記載がない。 | MEDIUM: セキュリティや正確性の問題でパイプラインが継続する可能性 | パイプライン自律性を優先; CRITICAL発見すべてで停止するのは過剰保守的と判断 | 実運用でCRITICAL発見クラスが明確になったとき、またはセキュリティインシデント発生時 | docs/reports/self-review-2026-04-09-ralph-loop-v2.md |
| ~~`ckpt_update()` が生のjqフィルタ式を受け取る汎用インターフェースのため、外部値（`_pr_url`, `_new_session`）が `--arg` なしで文字列連結で埋め込まれている。URLに `"` や `\` が含まれるとjqフィルタが壊れる（HIGH: security）。~~ **[RESOLVED: feat/ralph-loop-v2 pipeline-robustness で解消。`_new_session`（line 423）と `_pr_url`（line 700）の両箇所で `ckpt_update --arg` を使用した安全なJSON更新に修正済み]** | ~~不正なJSON書き込みまたはjq injection。github.com URLのみのため現実的リスクは低いが構造的に脆弱。~~ | ~~`ckpt_update` インターフェース変更は今回スコープ外の大きなリファクタリング。~~ | ~~`ralph-pipeline.sh` の次回リファクタリング時、またはURLソースが外部エンティティに拡張されるとき~~ | docs/reports/self-review-2026-04-09-pipeline-robustness.md |
| `run_inner_loop()` での phase更新後に旧phaseを読む順序バグ（`ralph-pipeline.sh:349-350`）。`phase_transitions` の `from` フィールドが常に "inner" を記録し、デバッグ情報の精度が低い。前回レビューと同じパターンが本ファイルでも再現。 | LOW: パイプライン動作には影響しない。監査ログの精度のみ。 | 機能的影響なし。今回の修正スコープ外。 | `ckpt_transition` の呼び出しパターンをリファクタリングするとき | docs/reports/self-review-2026-04-09-pipeline-robustness.md |
