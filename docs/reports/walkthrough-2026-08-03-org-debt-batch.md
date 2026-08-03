# Walkthrough — org-debt-batch (PR #141)

- Date: 2026-08-03
- Branch: chore/org-debt-batch(base: main / merge-base fde0e84)
- 規模: 33 files changed, +2,374 / −308
- プラン: docs/plans/archive/2026-08-03-org-debt-batch.md

## 読む順番(レビュー動線)

### 1. path 導出の一元化(Slice 1 — 最小だが最重要)

- `internal/org/manifest.go` / `receipts.go` — root 相対コンストラクタ(`NewManifestStore` 系)を削除し、`ManifestPathIn(stateDir)` / `ReceiptsPathIn(stateDir)` を唯一の導出点として新設。doc コメントに PR⑤ AR-1(state-dir 二重連結で `ralph status` が全マニフェストを見失ったバグ)の再発防止意図を記載
- `internal/cli/org.go` / `status.go` — 書き込み・読み取り両経路が `org.ManifestPathIn` を直接呼ぶ。中間ヘルパー `orgManifestPath` は削除
- テスト群 — fixture の `filepath.Join(dir, "manifest.jsonl")` を新ヘルパー経由に置換し、導出のドリフトをテストでも封じた

### 2. insights の org 再指向(Slice 3 — 実質新規コードの中心)

- `internal/insights/receipts.go` — 旧 pipeline スキーマ reader を削除、`Receipt = org.Receipt` alias + `org.NewReceiptStoreAtPath().Read()` 委譲
- `internal/insights/aggregate.go` — `AggregateReceipts`: `org_id` × `seat_id` group、tri-state honored 件数、rate = true/(true+false)(unknown は分母除外、全 unknown は `rate=n/a`)
- `internal/cli/insights.go` — 既定パスを `org.ResolveOrgStateDir` + `org.ReceiptsPathIn` で解決。出力契約はプラン AC-3 に明文化されており、テキスト例・JSON スキーマともテストで固定(空でも同一スキーマ)

### 3. watchdog 4 件(Slice 4 — 挙動変更を含む)

- `internal/org/watch.go` — (b) scope-change ALERT の `git status --porcelain` を 20 行 + 1600 バイト予算で有界化、protocol fallback に `SEAT:` ヘッダ / (c) `checkDeadman` の到達不能ガード削除 + `Escalated` を alertID 埋め込みタイムスタンプで直近 100 件に prune / (d) zero-active-seats ガード発火時に `Active`/`FirstTS` クリア(resume 後再アラート)
- `internal/cli/org.go` / `status.go` — (a) state-dir 解決 source(flag/env/git-toplevel/cwd)をバナーと status に表示、JSON に `state_dir_source`
- (d) の回帰テストは「遮断失敗で `Active=true, Cutoff=false` を作る」手法を採用 — 成功遮断は `Cutoff=true` になり別のラチェットが効くため(→ Known gap WC-1 参照)

### 4. ガード完全化と upgrade スモーク(Slices 2, 5)

- `tests/test-no-loop-references.sh` — 9 knob 完全形 PATTERN、`xreview-helpers.sh` のファイル全体除外を廃止
- `internal/cli/upgrade_retired_loop_artifacts_test.go` — 実 embedded テンプレートに対する E2E: loop 時代成果物入り manifest → `ralph upgrade` → removed-from-template 報告 + manifest から消滅 + **ディスクのファイルは残る**(remove は通知のみの安全設計 — 実契約を明文固定)

### 5. 台帳(tech-debt README)

- RESOLVED 化 4 row: C2-3 / C2-5 / C2-6 / watchdog deferred LOW 4 件
- 新規 2 row: Cutoff ラチェット(下記 Known gap)、cycle-1 deferred LOW 7 件バッチ

## Known gap(WC-1 / codex cycle-2 P2)

成功した total-budget 遮断の後に resume した org では、`evaluateTotalBudget` の `Cutoff` early-return が ALERT より先に効くため、ALERT も Stop も発生しない。Slice 4 の re-alert fix は遮断失敗時(`Cutoff=false`)のみ有効。Cutoff セマンティクス(clear するか resume-epoch スコープにするか)は専用の設計判断としてスコープ外 — tech-debt row にトリガー付きで登録済み。

## 品質エビデンス

| 項目 | 結果 |
|---|---|
| self-review | cycle1: 0C/0H/3M/6L(M 全修正)/ cycle2: 0C/0H/2M/4L(doc 修正) |
| verify | cycle1/2 とも PASS |
| test | 555/555 shell + Go 8/8、カバレッジ 83.5%(org 89.1%) |
| cross-review(codex) | cycle1: P3 1 件 fix / cycle2: P2 1 件 Known gap 記録 |
| evidence | docs/evidence/org-debt-batch-2026-08-03.txt(AC-1 grep、insights 実レンダー、upgrade スモーク) |
