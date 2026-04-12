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
| `scripts/ralph:294` の abort ワークツリーリストが pipe-subshell 変数スコープバグを残存（依存関係チェック・統合マージチェックは修正済み） | LOW: abort 時のワークツリーリストのみ残存 | 依存関係と統合マージは修正済み。abort 時の影響は監査ログの精度のみ | scripts/ralph の abort コマンドをリファクタリングするとき | docs/reports/self-review-2026-04-09-ralph-loop-v2.md |
| `ralph-pipeline.sh` の CRITICAL self-review 発見を無視するポリシー (line 421: "Don't stop — let verify and test catch real issues") が AGENTS.md および subagent-policy.md の契約と矛盾する。意図的な逸脱だが計画に記載がない。 | MEDIUM: セキュリティや正確性の問題でパイプラインが継続する可能性 | パイプライン自律性を優先; CRITICAL発見すべてで停止するのは過剰保守的と判断 | 実運用でCRITICAL発見クラスが明確になったとき、またはセキュリティインシデント発生時 | docs/reports/self-review-2026-04-09-ralph-loop-v2.md |
| PR prompt at `ralph-pipeline.sh:670` contains a hardcoded example URL `echo "https://github.com/..." > .harness/state/pipeline/.pr-url`. An agent may copy the example literally instead of writing the actual URL. A safer instruction would use `gh pr view --json url --jq '.url' > .harness/state/pipeline/.pr-url`. | LOW: Layer 1 (`gh pr list`) and Layer 3 (log grep) provide fallback detection, so the sidecar file failing does not block PR URL detection. Impact is cosmetic. | Deferred — MEDIUM finding from r2 self-review; functional impact is low given 3-layer detection. | Next revision of the PR prompt or `ralph-pipeline.sh`. | docs/reports/self-review-2026-04-09-pipeline-robustness-r2.md |
| `ralph-orchestrator.sh` の `wait_for_slice()` 関数が定義されているが一度も呼び出されていない。スライス待機は `sleep 5` ポーリングループで代替されている。 | LOW: 機能的影響なし。読者が誤ってブロッキング待機が実装されていると解釈するリスクのみ。 | 現在のポーリング方式は機能している。ブロッキング方式への移行は今回のスコープ外。 | `ralph-orchestrator.sh` を次回リファクタリングするとき | docs/reports/self-review-2026-04-10-ralph-loop-v2.md |
| `ralph-orchestrator.sh` の `integration_merge()` が `_conflicts` カウンタを定義・インクリメントするが、値はデバッグログのみに使われ、呼び出し元に返されない。 | LOW: コードの見通しを下げるだけで機能には影響しない。 | 軽微。今回のスコープ外。 | `integration_merge()` をリファクタリングするとき | docs/reports/self-review-2026-04-10-ralph-loop-v2.md |
| `ralph-orchestrator.sh:341` が `ralph-loop-init.sh --pipeline` を呼び出すが、`--pipeline` フラグは wip commit b8bd602 で削除済み。`exit 1` が `\|\| true` で抑制され、`ralph-pipeline.sh` の fallback path 2 (`.claude/skills/loop/prompts/pipeline-inner.md`) が代替する。機能的影響はないがスライスログにエラーが混入する。 | LOW: 機能的影響なし。スライスログのノイズのみ。 | pre-existing regression から wip commit; pipeline-quality-parity のスコープ外 | `ralph-loop-init.sh` または `ralph-orchestrator.sh` の次回リファクタリング時 | docs/reports/self-review-2026-04-10-pipeline-quality-parity.md |
