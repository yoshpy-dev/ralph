# Pipeline robustness improvements

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-09
- Related request: PR #5 監査で発見された実行時リスク 3件 + 軽微な不整合 2件の改善
- Related issue: N/A
- Branch: feat/ralph-loop-v2 (追加コミット)
- Codex advisory: 6件 → 全件反映済み

## Objective

`ralph-pipeline.sh` の自律実行信頼性を向上させる。`claude -p` 出力の脆弱な grep パース（R1, R2, R3）を構造化 JSON 解析に置き換え、多層防御パターンを導入する。軽微な不整合（M1, M2）も合わせて修正する。

## Scope

- `scripts/ralph-pipeline.sh` の `run_claude()` 関数を `--output-format json` + jq 解析に移行（テキストフォールバック付き）
- Session ID 抽出を JSON `.session_id` フィールドから取得に変更
- COMPLETE/ABORT シグナル検出をサイドカーファイル方式に変更（agent がファイルに書く + JSON result のマーカー検出をフォールバック）。両方のシグナルに対応。
- PR URL 検出を `gh pr list --head <branch> --state open --json url --jq '.[0].url'` による外部検証に変更
- サイドカーファイルのライフサイクル管理（各サイクル開始時にクリア）
- Preflight probe に `--output-format json` サポート検証を追加
- `ralph-pipeline.sh --help` の exit code を 0 に修正
- `pipeline-review.md` のレポート出力先指示を統一

## Non-goals

- `ralph-orchestrator.sh` の変更（R1-R3 はすべて `ralph-pipeline.sh` に局所化）
- `ralph-loop.sh`（標準モード）の変更
- `claude -p` の `--json-schema` による構造化出力強制（プロンプトの柔軟性を損なうため）
- 新しいテストフレームワークの導入
- abort の pipe-subshell 残存バグ修正（既に tech-debt に記録済み、影響が監査ログ精度のみ）

## Assumptions

- `claude -p --output-format json` は `{ "result": "...", "session_id": "...", ... }` 形式の JSON を返す
- `gh pr list --head <branch> --state open --json url` で PR の存在と URL を取得可能
- `jq` は preflight probe で検証済み（必須依存として扱う）
- プロンプトテンプレート内の COMPLETE/ABORT マーカー指示は維持（フォールバック用）
- `claude -p` の stdout に非 JSON テキスト（警告、進捗等）が混在する可能性がある

## Affected areas

| ファイル | 変更種別 |
|---------|---------|
| `scripts/ralph-pipeline.sh` | 修正 (run_claude, preflight, session_id 抽出, signal 検出, PR URL 検出, sidecar lifecycle, usage exit code) |
| `.claude/skills/loop/prompts/pipeline-inner.md` | 修正 (サイドカーファイル出力指示の追加 — COMPLETE と ABORT 両方) |
| `.claude/skills/loop/prompts/pipeline-outer.md` | 修正 (サイドカーファイル出力指示の追加 — PR URL) |
| `.claude/skills/loop/prompts/pipeline-review.md` | 修正 (レポート出力先の統一) |
| `docs/quality/definition-of-done.md` | 修正 (レポート出力先の明確化) |

## Acceptance criteria

- [ ] AC1: `run_claude()` が `--output-format json` を使用し、JSON 全体を `${_log_file}.json` に保存。jq で `.result` を `${_log_file}` に抽出する。JSON パースに失敗した場合は raw 出力を `${_log_file}` にフォールバック保存し、警告ログを出力する。
- [ ] AC2: Session ID が JSON の `.session_id` フィールドから jq で抽出される（grep フォールバックなし）
- [ ] AC3: COMPLETE **および** ABORT 検出が 2層構造: (1) サイドカーファイル `.harness/state/pipeline/.agent-signal`（値は `COMPLETE` または `ABORT`）、(2) JSON result テキスト内のマーカー grep
- [ ] AC4: サイドカーファイル (`.agent-signal`, `.pr-url`) が各 Inner Loop サイクル開始時にクリアされる（stale 値の防止）
- [ ] AC5: PR URL 検出が `gh pr list --head <branch> --state open --json url --jq '.[0].url'` を最優先とし、サイドカーファイル (`.pr-url`) を第2層、agent 出力 grep を第3層とする
- [ ] AC6: Preflight probe に `claude -p --output-format json` サポートの検証ステップが追加される。非対応の場合は `--output-format text` にフォールバックし、警告ログを出力する
- [ ] AC7: `ralph-pipeline.sh --help` が exit 0 を返す
- [ ] AC8: `pipeline-review.md` のレポート出力先が `.harness/state/pipeline/` に統一される
- [ ] AC9: `definition-of-done.md` がパイプラインモードのレポート出力先を明確に記述する
- [ ] AC10: 既存の dry-run テスト (`--preflight --dry-run`, `--dry-run --max-iterations 3`) が引き続き PASS する
- [ ] AC11: `sh -n` 構文チェックが全スクリプトで PASS する

## Implementation outline

### Phase 1: `run_claude()` の JSON 出力移行 + Preflight 拡張

1. Preflight probe に `--output-format json` サポート検証を追加:
   - dry-run でない場合: `echo "Reply PROBE_OK" | claude -p --output-format json 2>/dev/null` を実行
   - 有効な JSON が返れば `json_output_supported=true`、そうでなければ `false` で警告
   - `false` の場合は従来の `--output-format text` にフォールバック（パイプラインは続行）
   - 結果を `preflight-probe.json` に記録

2. `run_claude()` を修正:
   - JSON モード対応の場合: `claude -p --output-format json < "$_prompt_file" > "${_log_file}.json" 2>"${_log_file}.stderr"`
   - **stdout/stderr を分離** して JSON パースの堅牢性を確保
   - `jq -r '.result // empty' "${_log_file}.json" > "${_log_file}"` で result を抽出
   - **jq 失敗時のフォールバック**: `jq` が失敗したら raw 出力を `${_log_file}` にコピーし、`log "Warning: JSON parse failed, using raw output"` を出力
   - `tee` はログファイル抽出後のテキストに対してのみ使用
   - テキストフォールバックモードの場合: 従来の `claude -p --output-format text | tee` を維持

3. Session ID 抽出を修正:
   - `jq -r '.session_id // empty' "${_impl_log}.json"` で取得
   - 空なら警告ログのみ（パイプラインは継続）

### Phase 2: シグナル検出の 2層化 + ライフサイクル管理

1. **サイドカーファイルのクリア**: Inner Loop サイクル開始時に `rm -f "${PIPELINE_DIR}/.agent-signal" "${PIPELINE_DIR}/.pr-url"` を実行（stale 値の防止）

2. プロンプトテンプレートにサイドカーファイル指示を追加:
   - `pipeline-inner.md`:
     - COMPLETE 時: `echo COMPLETE > .harness/state/pipeline/.agent-signal`
     - ABORT 時: `echo ABORT > .harness/state/pipeline/.agent-signal`
   - `pipeline-outer.md`:
     - PR 作成成功時: PR URL を `.harness/state/pipeline/.pr-url` に書き込む

3. `ralph-pipeline.sh` のシグナル検出ロジックを修正:
   - **COMPLETE 検出**:
     - Layer 1: `cat "${PIPELINE_DIR}/.agent-signal"` が `COMPLETE` を含むか
     - Layer 2: ログファイル（result テキスト）内の `<promise>COMPLETE</promise>` マーカー grep
   - **ABORT 検出**:
     - Layer 1: `cat "${PIPELINE_DIR}/.agent-signal"` が `ABORT` を含むか
     - Layer 2: ログファイル（result テキスト）内の `<promise>ABORT</promise>` マーカー grep
   - いずれかの層で検出されれば該当シグナル扱い

### Phase 3: PR URL 検出の外部検証化

1. Outer Loop の PR 検出ロジックを修正:
   - Layer 1: `gh pr list --head "$(git rev-parse --abbrev-ref HEAD)" --state open --json url --jq '.[0].url'` で外部検証
   - Layer 2: サイドカーファイル `${PIPELINE_DIR}/.pr-url` の内容を確認
   - Layer 3: agent 出力ログの URL grep (既存のフォールバック)
   - 各層で取得できた最初の URL を採用

### Phase 4: 軽微な不整合修正

1. `ralph-pipeline.sh` の `usage()`: `exit 1` → `exit 0`
2. `pipeline-review.md`: レポート出力先を `.harness/state/pipeline/` に統一
3. `definition-of-done.md`: パイプラインモードのレポート配置を明確化（「Inner Loop ログは `.harness/state/pipeline/`、最終レポートは `docs/reports/`」）

## Verify plan

- Static analysis checks:
  - `sh -n scripts/ralph-pipeline.sh` PASS
  - `sh -n scripts/ralph` PASS
  - `sh -n scripts/ralph-orchestrator.sh` PASS (変更なしの確認)
- Spec compliance criteria to confirm:
  - AC1-AC11 の全項目をコードパスで検証
  - `run_claude()` が `--output-format json` を使用している（フォールバック付き）
  - Session ID 抽出に grep が使われていない
  - PR URL 検出に `gh pr list` が使われている
  - サイドカーファイルがサイクル開始時にクリアされている
  - ABORT シグナルがサイドカーとマーカーの両方で検出可能
  - Preflight で JSON output サポートが検証されている
- Documentation drift to check:
  - `pipeline-inner.md`, `pipeline-outer.md`, `pipeline-review.md` のサイドカーファイル指示
  - `definition-of-done.md` のレポート配置記述
  - `quality-gates.md` に変更がないこと（今回は不要）
- Evidence to capture:
  - 修正前後の `run_claude()` 関数差分
  - dry-run テスト結果

## Test plan

- Unit tests:
  - `ralph-pipeline.sh --preflight --dry-run` が exit 0
  - `ralph-pipeline.sh --help` が exit 0（**AC7 検証**）
  - `ralph-pipeline.sh --dry-run --max-iterations 3` が全フェーズを通過
- Integration tests:
  - N/A（`claude -p` 実行はテスト環境で不可）
- Regression tests:
  - `ralph --help` が exit 0
  - `ralph status` が正常動作
  - `ralph-loop-init.sh --pipeline` が正常動作
  - `ralph-orchestrator.sh --dry-run` が正常動作（変更なしの確認）
- Edge cases:
  - `run_claude()` に不正な JSON（非 JSON 混在）が返された場合: raw フォールバック動作を確認
  - `.agent-signal` ファイルが存在しない場合: Layer 2 フォールバック動作を確認
  - `gh pr list` が空配列を返した場合: Layer 2/3 フォールバック動作を確認
  - stale `.agent-signal` が残っている場合: サイクル開始時のクリアで除去されることを確認
- Evidence to capture:
  - テスト実行ログ

## Risks and mitigations

| リスク | 影響 | 緩和策 |
|--------|------|--------|
| `claude -p --output-format json` の出力に非 JSON テキストが混在する | jq パース失敗 | stdout/stderr 分離 + jq 失敗時の raw フォールバック (AC1) |
| `--output-format json` が古い claude CLI でサポートされていない | パイプライン起動失敗 | Preflight で検証し、非対応なら `--output-format text` にフォールバック (AC6) |
| agent がサイドカーファイルに書き込まない | シグナル検出失敗 | Layer 2（マーカー grep）がフォールバック。安全側に倒れる |
| `gh pr list` が認証失敗する/複数 PR を返す | PR URL 外部検証失敗/誤検出 | `--state open` + `--jq '.[0].url'` で最新 open PR に限定。失敗時は Layer 2/3 にフォールバック |
| stale サイドカーファイルによる誤検出 | 前回の COMPLETE/PR URL を拾う | 各サイクル開始時にクリア (AC4) |

## Rollout or rollback notes

- `feat/ralph-loop-v2` ブランチ上での追加コミットとして実装
- ロールバック:
  1. コードリバート: `git revert <commit>` で元の動作に復帰
  2. 状態クリーンアップ: `rm -f .harness/state/pipeline/.agent-signal .harness/state/pipeline/.pr-url` でサイドカーファイルを除去
  3. 部分適用対策: プロンプトテンプレートの変更はコードリバートに含まれるため、追加クリーンアップ不要

## Codex advisory findings (2026-04-09)

| # | 深刻度 | 指摘 | 対応 |
|---|--------|------|------|
| 1 | HIGH | stale サイドカーファイルによる誤検出 | AC4 追加: サイクル開始時クリア |
| 2 | HIGH | JSON パース堅牢性の不足 | AC1 修正: stdout/stderr 分離 + raw フォールバック |
| 3 | MEDIUM | ABORT ハンドリングの欠如 | AC3 修正: ABORT もサイドカー+マーカーの2層対応 |
| 4 | MEDIUM | `gh pr list` の曖昧な結果 | AC5 修正: `--state open --jq '.[0].url'` で限定 |
| 5 | MEDIUM | 後方互換性管理の不足 | AC6 追加: Preflight で JSON output サポートを検証 |
| 6 | LOW | ロールバック注記の不足 | Rollout notes 修正: 状態クリーンアップ手順を追加 |

## Open questions

なし（Codex 指摘を含め全件対応済み）

## Progress checklist

- [x] Plan reviewed
- [x] Codex advisory reviewed (6件 → 全件反映)
- [x] Branch created (feat/ralph-loop-v2 — 既存ブランチに追加コミット)
- [x] Phase 1 implemented (run_claude JSON 移行 + Preflight 拡張)
- [x] Phase 2 implemented (シグナル検出 2層化 + ライフサイクル管理)
- [x] Phase 3 implemented (PR URL 外部検証)
- [x] Phase 4 implemented (軽微な不整合修正)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR updated
