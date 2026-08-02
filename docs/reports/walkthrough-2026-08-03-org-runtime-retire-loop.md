# Walkthrough — org-runtime-retire-loop (PR #140)

- Date: 2026-08-03
- Branch: refactor/org-runtime-retire-loop(base: main / merge-base d074838)
- 規模: 228 files changed, +3,177 / −30,763
- スペック: docs/specs/2026-08-01-org-runtime.md(PR 系列⑤ / FR-11)
- プラン: docs/plans/archive/2026-08-03-org-runtime-retire-loop.md

## 読む順番(レビュー動線)

削除が体積の 9 割を占めるため、「何を残したか」→「何を書き換えたか」→「何を消したか」の順で読むのが最短。

### 1. 生存境界の確認(このPRの最重要判断)

退役スコープは **Ralph Loop 自律実行システムのみ**。開発ハーネス(/spec /plan /work /self-review /verify /test /sync-docs /cross-review /pr)は存続する。境界の定義は:

- `AGENTS.md` — Primary loop を「開発ハーネス + org runtime」の 2 本柱に再編。repo map から loop 系エントリを削除
- `.claude/rules/post-implementation-pipeline.md` / `subagent-policy.md` / `model-routing.md` — Loop 節・per-phase ルーティング表を削除し標準フローのみに
- `.claude/skills/plan/SKILL.md` / `work/SKILL.md` — フロー選択(標準 vs Ralph Loop)を削除。ディレクトリ型プラン検出時は退役エラーを案内

### 2. 唯一の実質新規コード: `ralph status` の org 化

- `internal/cli/status.go` — 旧 TUI 向けスライスリーダー依存を捨て、org manifest(`manifest.jsonl`)から全 org の roster を導出。`--org-id`/`--state-dir`/`--json`、watch ハートビート表示
- 重要修正(cross-review AR-1): state-dir 二重連結バグ。書き込み側(`newOrgRuntimeAt`)と読み取り側で `orgManifestPath()`(`internal/cli/org.go`)を共有し、実書き込み経路の回帰テスト `TestStatusCmd_SeesSeatWrittenByRealOrgSpawn` で再発を封じた
- 重要修正(self-review C2-1): dry-run 座席は行表示するが `active N/M` と JSON の `active_count`/`total_count` からは除外(spawn の max_seats ゲートと同じ `RosterOptions{}` 集計)

### 3. cross-review ヘルパー抽出

- `scripts/xreview-helpers.sh`(+ templates ミラー)— 退役した ralph-cli-driver.sh から `detect_base_branch` / `pick_reviewer` / `count_triage_findings` を救出。`pick_reviewer` は大文字ドライバ値を小文字正規化(cross-review AR-2)
- `.claude/skills/cross-review/SKILL.md`(4 面ミラー)— ヘルパーを実際に source して使う手順に配線
- `tests/test-xreview-helpers.sh` — 29 ケース

### 4. 恒久ガード

- `tests/test-no-loop-references.sh` — 退役面トークン(ralph-orchestrator / ralph-pipeline / RALPH_LOOP_DRIVER / ralph-cli-driver / ralph-loop.sh / ralph-status-helpers / new-ralph-plan / build-tui / ralph-tui / per-phase RALPH_*_MODEL / checkStaleOrchestratorState)への生参照ゼロを保証。歴史的文書(docs/plans/archive、docs/specs、docs/reports、docs/insights、docs/tech-debt)は除外
- `scripts/verify.local.sh` の `tests/test-*.sh` glob が自動収集するため CI 配線は追加不要

### 5. 削除本体(±0 で読める)

- `scripts/`: ralph-orchestrator.sh、ralph-pipeline.sh、ralph-loop-init.sh、ralph-loop.sh、ralph-status-helpers.sh、ralph-cli-driver.sh、build-tui.sh、new-ralph-plan.sh、旧 shell CLI `ralph`
- Go: `cmd/ralph-tui`、`internal/ui`、`internal/state`、`internal/action`、`internal/watcher`、cli run/retry/abort。`internal/config` から Loop/Pipeline/PhaseModel セクション削除
- スキル・テンプレート: `/loop`(4 面)、adversarial-claude プロンプト、ralph-loop-manifest/slice プランテンプレート、loop レシピ
- テスト: loop 系シェルスイート約 14 本

### 6. 付随クローズ(PR④ known gaps)

- `internal/org/watch.go` — デッドマンのプローブ復旧誤クリア: ALERT 時ベースラインが有効な場合のみプローブソースを採用(`pending.LeadAgentGet != ""` 等のゲート)
- watchdog join ラチェット: `ensureWatchdogJoined` が Join 成功時のみ `WatchdogJoined` を立てる

## 品質エビデンス

| 項目 | 結果 |
|---|---|
| self-review | cycle1: 0C/1H/8M/7L → fix / cycle2: 0C/0H(C2-1/C2-2 fix 済み) |
| verify | cycle1/2 とも PASS(AC-1〜AC-8) |
| test | cycle2: シェル 555/555 + Go 8/8 pkg、フレーク隔離再実行で安定確認 |
| cross-review(codex) | cycle1: AR 2 件 → fix / cycle2: 指摘ゼロ |
| スキャフォールドスモーク | docs/evidence/org-retire-loop-smoke-2026-08-03.txt(loop 成果物なし・org 成果物あり) |

## Known gaps(tech-debt 登録済み)

- C2-3: `org.NewManifestStore`/`ManifestRelPath` が呼び出し元ゼロのまま公開(二重連結 footgun の残置)
- C2-5: ガードテストの `RALPH_*_MODEL` トークンカバレッジ残(8 knob 中 5)
- C2-6: `ralph insights --receipts` 既定パスが退役 writer 由来
- 既存プロジェクトへの `ralph upgrade` 削除経路スモークは手動確認事項
