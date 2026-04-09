# Harness audit memo — 2026-04-09 (PR #5 最終確認)

Trigger: PR #5 (`feat/ralph-loop-v2`) の自律実装可能性評価。

## 結論

**PR #5 は概ね自律実行可能な状態。ただし、実行時リスク 3件と軽微な不整合 2件あり。**

マージをブロックするものはないが、初回実運用は dry-run + preflight を先行させるべき。

---

## 強み

| # | 項目 | 評価 |
|---|------|------|
| 1 | **シェルスクリプト構文**: 3ファイルすべて `sh -n` PASS | GOOD |
| 2 | **Preflight probe**: `ralph-pipeline.sh --preflight --dry-run` 正常動作 | GOOD |
| 3 | **CLI ヘルプ**: `ralph --help` exit 0、`ralph status` 正常動作 | GOOD |
| 4 | **テスト**: 13/13 PASS (2ラウンド) | GOOD |
| 5 | **Codex 4件 ACTION_REQUIRED**: 全件修正・再検証済み | GOOD |
| 6 | **POSIX sh 準拠**: bash 依存なし、`set -eu` 環境で堅牢 | GOOD |
| 7 | **Pipe-subshell 修正**: 変数スコープバグ全箇所を temp-file パターンに置換済み | GOOD |
| 8 | **Post-implementation pipeline**: single source of truth (`post-implementation-pipeline.md`) 新設でスキップバグ再発防止 | GOOD |
| 9 | **Quality docs 整合**: `definition-of-done.md` と `quality-gates.md` がパイプラインモードを反映 | GOOD |
| 10 | **安全弁**: MAX_ITERATIONS=20, MAX_INNER_CYCLES=10, stuck detection, repair limit, ABORT signal | GOOD |
| 11 | **状態管理**: checkpoint.json + execution-events.jsonl + audit log | GOOD |
| 12 | **アボート**: `ralph abort` でプロセス停止・状態アーカイブ・ワークツリー除去・監査ログ出力 | GOOD |

## 実行時リスク (自律実行への障壁)

| # | リスク | 深刻度 | 詳細 | 緩和策 |
|---|--------|--------|------|--------|
| R1 | **`claude -p` 出力形式依存** | HIGH | `session_id=` パターンの grep (`scripts/ralph-pipeline.sh:369`) で session ID を抽出。`claude -p --output-format text` の出力に `session_id=` が含まれる保証がない。 | `--resume` は session ID がなければスキップされる設計なので、機能劣化にとどまる。セッション継続が必要なら手動で `--resume` に session ID を渡す運用で回避可能。 |
| R2 | **COMPLETE/ABORT シグナル検出** | MEDIUM | `<promise>COMPLETE</promise>` という非標準タグを agent 出力から grep で検出 (`scripts/ralph-pipeline.sh:380-388`)。`claude -p` の agent が確実にこのタグを出力する保証がない。 | COMPLETE 未検出の場合は MAX_ITERATIONS まで実行が継続するだけ（安全側に倒れる）。ABORT 未検出も同様。最悪ケースはコスト増加のみ。 |
| R3 | **Outer Loop の PR 作成** | MEDIUM | `claude -p` に PR 作成を委任。PR URL の検出は `grep -oE 'https://github\.com/...'` で行う (`scripts/ralph-pipeline.sh:598`)。agent が URL をそのまま出力しない場合、checkpoint に `pr_created: false` が記録される。 | PR が作成されなかった場合、ユーザーが `ralph status` で確認し手動で `/pr` を実行可能。ブロッカーではない。 |

## 軽微な不整合

| # | 問題 | 深刻度 | 推奨対応 |
|---|------|--------|---------|
| M1 | `ralph-pipeline.sh --help` が exit 1 | LOW | `ralph --help` は exit 0 に修正済みだが、pipeline.sh の `usage()` は `exit 1` のまま。一貫性のため `exit 0` に変更推奨。 |
| M2 | `pipeline-review.md` のレポート出力先が `docs/reports/self-review-<date>-pipeline.md` と記載されているが、`definition-of-done.md` では「Reports live in `.harness/state/pipeline/`」 | LOW | パイプラインモードでは agent が docs/reports/ に書くのか .harness/state/pipeline/ に書くのかが曖昧。統一推奨。 |

## Always-on コンテキスト量

| カテゴリ | バイト数 |
|---------|---------|
| CLAUDE.md | 2,687 |
| AGENTS.md | 3,705 |
| `.claude/rules/` (全 10 ファイル) | 12,785 |
| `.claude/skills/loop/SKILL.md` | 8,089 |
| **合計** | **27,266** (~27KB) |

`loop/SKILL.md` が 8KB と大きいが、allowed-tools でスコープされているため常時ロードではない。`post-implementation-pipeline.md` (1.7KB) と `subagent-policy.md` (4.1KB) は常時ロード。合計は許容範囲。

## 欠けているガードレール

| # | ガードレール | 推奨 | 優先度 |
|---|-------------|------|--------|
| G1 | checkpoint.json スキーマ検証 | `jq` による必須フィールド検証を `ralph status` に追加 | LOW |
| G2 | パイプライン全ステップのレポート存在確認 | `/pr` pre-checks にレポート存在チェックを追加 | MEDIUM |
| G3 | `ralph-pipeline.sh` の統合テスト | mock `claude` コマンドを使った E2E テスト追加 | MEDIUM |
| G4 | `ralph abort --slice` 時の pipe-subshell 残存バグ | tech-debt に記録済み（`scripts/ralph:294`）。影響は監査ログ精度のみ | LOW |

## Tech debt 状況

| 項目 | 状態 |
|------|------|
| CLAUDE.md 矛盾 | RESOLVED (PR #5) |
| pipe-subshell バグ | PARTIALLY RESOLVED (abort のみ残存) |
| CRITICAL self-review 無視ポリシー | OPEN (意図的逸脱として記録済み) |

## 自律実行シナリオ評価

### 想定ワークフロー
```
/plan → /loop (pipeline mode) → ralph run → (自律: implement → review → verify → test → docs → codex → PR)
```

### 判定

| フェーズ | 自律可能か | 条件 |
|---------|-----------|------|
| Preflight | YES | `claude`, `jq`, `git` が PATH にあること |
| Inner Loop (implement) | YES (with caveats) | `claude -p` が CLAUDE.md と plan を読めること (preflight で検証) |
| Inner Loop (self-review) | YES | agent がレポートを書けること |
| Inner Loop (verify/test) | YES | `run-verify.sh` / `run-test.sh` が存在すること |
| Outer Loop (sync-docs) | YES | agent がドキュメント更新をコミットできること |
| Outer Loop (codex-review) | CONDITIONAL | codex CLI の有無。なければスキップ（正常動作） |
| Outer Loop (PR creation) | YES (with caveats) | `gh` CLI が認証済みであること。agent が PR URL を出力すること |
| Stuck/Abort handling | YES | 全パスが dry-run で検証済み |

### 推奨初回実行手順
```sh
# 1. Preflight のみ
scripts/ralph-pipeline.sh --preflight

# 2. Dry-run で全フローを確認
scripts/ralph run --dry-run

# 3. 制限付き実行
scripts/ralph run --max-iterations 5

# 4. 問題なければフル実行
scripts/ralph run
```

## 総合評価

PR #5 は **マージ可能**。コアロジック（Inner/Outer Loop、checkpoint、stuck detection、abort）は堅牢で、4件の Codex 指摘も修正済み。

自律実行における最大のリスクは `claude -p` の出力形式依存（R1-R3）だが、いずれも安全側にフォールバックする設計。初回実運用時は段階的実行（preflight → dry-run → bounded run）を推奨。

軽微な不整合（M1, M2）は次回修正で十分。

---

Auditor: Claude Code (audit-harness skill)
Date: 2026-04-09
