# Plan: org runtime 残債バッチのクローズ

- Date: 2026-08-03
- Type: chore
- Branch: chore/org-debt-batch
- Related issue: N/A
- Canonical ref: docs/tech-debt/README.md の org runtime 系 5 row(C2-3 / C2-5 / C2-6 / watchdog deferred LOW 4 件 / upgrade 削除経路スモーク)

## Objective

PR⑤(#140)完了時点で tech-debt に登録した org runtime 系の残債 5 項目を 1 バッチでクローズし、該当 tech-debt row を RESOLVED にする。

## Scope

1. **C2-3 — footgun API 削除**: `org.NewManifestStore`/`ManifestRelPath`、`org.NewReceiptStore`/`ReceiptsRelPath`(呼び出し元ゼロの root 相対コンストラクタ)を削除。path 導出を `internal/org` に一元化(`org.ManifestPathIn(stateDir)` / `org.ReceiptsPathIn(stateDir)` を新設し、`internal/cli/org.go` の `orgManifestPath` はそれへ委譲または削除)。テストの `NewManifestStoreAtPath(filepath.Join(dir, "manifest.jsonl"))` パターンは新ヘルパー経由に置換。
2. **C2-5 — ガードテスト完全化**: `tests/test-no-loop-references.sh` の `PATTERN` を 8 knob 完全形 `RALPH_(FORCE|IMPLEMENT|SELF_REVIEW|VERIFY|TEST|SYNC_DOCS|PR|PROBE|ESCALATION)_MODEL` に拡張(9 トークン: FORCE 含む)。`EXCLUDE_REGEX` の `xreview-helpers.sh`(+ミラー)全体除外を provenance コメント行だけが通る形に絞る(ファイル全体除外をやめ、ヒット行の内容で判定するか、provenance 行の書き換えで除外自体を不要化 — 後者優先)。
3. **C2-6 — insights receipts の org 再指向**: `ralph insights --receipts` の既定パスを org runtime の receipts(`<state-dir>/model-receipts.jsonl`、`org.ResolveOrgStateDir` で解決)に変更し、org 受け皿スキーマ(`commanded_model` / `reported_effective_model` / `honored` tri-state 文字列)を読む reader を `internal/insights` に追加。旧 `.harness/state/pipeline/` スキーマの reader と flag help / doc コメントを整理(旧パス明示指定時の後方互換は不要 — writer は退役済み。旧スキーマ reader は削除)。
4. **watchdog deferred LOW 4 件**(`internal/org/watch.go` / `internal/cli/org.go`):
   a. `ResolveOrgStateDir` の `source` を `ralph org watch` 起動バナーと `ralph status`(state-dir 行)に表示。doctor は org state チェック行があれば併記、なければ対象外と明記。
   b. scope-change ALERT: `git status --porcelain` 出力を有界(先頭 20 行 + `… N more`)に truncate してからボディ構築。protocol validation fallback 経路にも `SEAT:` ヘッダを含め、Lead が対象座席を特定できるようにする。
   c. `checkDeadman` の到達不能な `status.Escalated[alertID]` ガードを削除(コメントで理由を残す)。`Escalated` マップにエントリ上限(例: 直近 100 件で prune)を導入し `watch-status-<org>.json` の無限成長を止める。
   d. zero-active-seats ガード(`evaluateTotalBudget`)発火時に `Conditions[key].Active` をクリアし、resume 後の同一条件で再アラートされるようにする。既存回帰テストの期待値を新仕様に更新し、resume→再スポーン→再遮断で ALERT が出るテストを追加。
5. **upgrade 削除経路スモークの自動化**: 旧 loop 成果物(`scripts/ralph-orchestrator.sh` 等)を持つ模擬プロジェクトを組み立て、`ralph upgrade` が manifest 差分で削除することをアサートするテストを追加(`internal/upgrade` の Go テスト or `tests/test-upgrade-remove-loop.sh`)。既存の remove-path テスト資産があれば拡張で済ます。**RESOLVED 化は自動テスト green が条件**(AC-5)。自動化が過大なら手動スモーク evidence を残した上で row は「手動確認済み・自動化未了」に更新し、クローズしない。

## Non-goals

- org runtime の新機能(verbs 追加、watcher 強化)
- `ralph insights` のイベント集計側(`docs/insights/events/`)のスキーマ変更
- org 以外の長期債(init/upgrade トランザクション安全性、mojibake フック、terraform 系)
- ガードテストの除外対象である歴史的文書の書き換え

## Assumptions

- 旧 pipeline receipts(`.harness/state/pipeline/model-receipts.jsonl`)の writer は存在しないため、旧スキーマ読み取りの後方互換は不要(スキーマ判断は Design decisions 参照)。
- `Escalated` prune の上限値は 100 で十分(1 org の ALERT 総数がこれを超える運用は watch の再起動で常に新 status ファイルになる)。
- watchdog 4 件はいずれも数行〜数十行の局所修正で、`internal/org` の既存テストパターン(fake driver)で検証可能。

## Affected files

- `internal/org/manifest.go` / `receipts.go`(+ 対応テスト、`ManifestPathIn`/`ReceiptsPathIn` 新設)
- `internal/cli/org.go`(`orgManifestPath` 委譲、watch バナー source 表示)、`internal/cli/status.go`(state-dir source 表示)
- `internal/cli/insights.go`、`internal/insights/receipts.go`(+テスト)
- `internal/org/watch.go`(+ `watch_test.go`)
- `tests/test-no-loop-references.sh`、`scripts/xreview-helpers.sh`(+ templates ミラー、provenance 行の書き換えのみ)
- `internal/upgrade` or `tests/`(削除経路スモーク)
- `docs/tech-debt/README.md`(5 row の RESOLVED 化)、`.claude/rules/` 影響箇所

## Acceptance criteria

- AC-1: `grep -rn 'NewManifestStore\b\|ManifestRelPath\|NewReceiptStore\b\|ReceiptsRelPath' --include='*.go'` の非テストヒットがゼロ(`...AtPath` と新 `*PathIn` のみ残る)。path 導出は `internal/org` の 1 箇所。
- AC-2: `tests/test-no-loop-references.sh` の `PATTERN` が 9 knob トークンを含み、`xreview-helpers.sh` のファイル全体除外が消えた状態で PASS。
- AC-3: `ralph insights` が org receipts を既定で読み、下記の出力契約どおりに表示する(単体テストで契約を固定 + 実ファイル手動確認 evidence)。旧パスの参照が help/doc から消えている。
  - **集計セマンティクス**: `org_id` × `seat_id` で group。`honored` は tri-state のまま件数集計(true/false/unknown の 3 カラム)。honored-rate は `true / (true+false)`(**unknown は分母から除外**し、`unknown N` を併記)。receipts ゼロ件時は「no org receipts found (path)」の 1 行。
  - **テキスト出力例**: `ORG demo  SEAT lead  commanded=opus  honored: true=3 false=1 unknown=2  rate=75% (unknown 2 excluded)`
  - **JSON**: `receipts` セクションを `{"path": ..., "orgs": [{"org_id": ..., "seats": [{"seat_id", "commanded_models", "honored_true", "honored_false", "honored_unknown"}]}], "skipped_lines": N}` とし、空でも同一スキーマ(status --json と同じ規約)。旧 per-phase honored-rate 表示(Routing セクション)はイベント由来のまま無変更。
- AC-4: watchdog 4 件それぞれに回帰テストがあり green(a: banner/status に source 文字列、b: truncate と fallback SEAT ヘッダ、c: Escalated prune、d: resume 後再アラート)。
- AC-5: upgrade 削除経路の**自動テストが green であることが tech-debt row RESOLVED の必須条件**。自動化が過大と判明した場合は row を RESOLVED にせず「手動確認済み・自動化未了(日付+evidence パス)」へ更新し、手動 evidence を `docs/evidence/` に残す($HOME→~ 赤入れ規約適用)。この場合 AC-5 は「row を正直な状態に更新した」ことで満たす。
- AC-6: 対象 tech-debt 5 row が RESOLVED(取り消し線+日付+PR 参照)。
- AC-7: 最終ゲートは `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`(static+test を包含)green + `go test ./... -count=1` green。`run-static-verify.sh` 単体は /verify フェーズの部分ゲートであり AC の対象外。

## Implementation outline

1. **Slice 1 — C2-3 API 一元化**(manifest/receipts path 導出 + cli 委譲 + テスト置換)
2. **Slice 2 — C2-5 ガード完全化**(シェル面のみ)
3. **Slice 3 — C2-6 insights 再指向**(reader + CLI 既定パス + 出力契約テスト)
4. **Slice 4 — watchdog 4 件**(watch.go 局所修正 + テスト)
5. **Slice 5 — upgrade 削除経路スモーク + tech-debt RESOLVED 化 + doc 追従**

各スライスは implementer サブエージェントに委譲し、1 スライス = 1 コミット。

## Design decisions

- **C2-6 のスキーマ方針(critical fork、2026-08-03 ユーザー決定)**: **org receipts に再指向**。既定パスを org state-dir の `model-receipts.jsonl` に変更し、tri-state `honored`(true/false/unknown)を読む reader を追加。旧 pipeline スキーマ reader は削除(writer 退役済みのため後方互換不要)。理由: insights を生きた診断コマンドに戻す。却下案: historical 格下げ(org receipts の集計手段が無いまま)、セクション削除(受領証確認の CLI 手段が消える)。
- Critical forks(その他): None — watchdog の prune 方式(上限 100)や truncate 行数(20)は可逆なデフォルト採用。

## Verify plan

- `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`(check-sync / check-skill-sync / check-pipeline-sync / gofmt / vet / staticcheck / golangci-lint)
- AC-1 の grep をそのまま実行して evidence 化
- doc ドリフト: `.claude/rules/model-routing.md`(receipts 記述)、`docs/insights/README.md`、`/org` skill の該当記述

## Test plan

- 単体: `internal/org`(path ヘルパー、watch 4 件)、`internal/insights`(org receipts reader、corrupt 行、missing file)、`internal/cli`(insights 既定パス、status/watch source 表示)
- シェル: `tests/test-no-loop-references.sh`(拡張後 PASS)、`tests/test-xreview-helpers.sh`(既存 29 維持)
- 回帰: `go test ./... -count=1` 全 pkg、シェル全スイート
- エッジ: org receipts 空/欠損時の insights 表示、Escalated ちょうど上限、zero-active-seats→respawn→再遮断

## Risk register

| リスク | 影響 | 緩和 |
|---|---|---|
| C2-3 でテストの path 前提が広く崩れる | ビルド/テスト赤 | 置換は機械的、`*PathIn` に寄せて grep で残骸確認 |
| C2-6 org receipts スキーマの集計解釈誤り | insights の誤表示 | tri-state をそのまま表示し bool に潰さない。テストで 3 値網羅 |
| watchdog d の仕様変更が既存回帰テストの意図と衝突 | 挙動の意図せぬ変化 | 既存テスト名の意図(silent が仕様だったか)をコミット履歴で確認してから変更 |
| upgrade スモーク自動化が過大化 | スコープ膨張 | 半日以内に自動化の目処が立たなければ手動 evidence にフォールバック(AC-5 は両対応) |

## Rollout / rollback

- 全変更は 1 PR だが、**1 スライス = 1 コミット = 独立 revert 可能**を部分失敗時の境界とする(C2-3/C2-5/C2-6/watchdog/スモークは互いに依存しない)。マージ後に特定領域だけ問題が出た場合は該当スライスのコミットのみ revert する。
- 受け皿変更(insights 既定パス)は CLI ローカル診断のみで外部影響なし。状態ファイルのマイグレーションなし。

## Evidence targets

- AC-1 grep 出力、insights 実行スクリーンショット相当のテキスト、upgrade スモーク出力(自動 or 手動、赤入れ規約適用)→ `docs/evidence/org-debt-batch-2026-08-03.txt`

## Progress checklist

- [x] Slice 1 — C2-3 API 一元化(3b91875 + 0e510a7)
- [x] Slice 2 — C2-5 ガード完全化(57effc3)
- [x] Slice 3 — C2-6 insights 再指向(e00e301 + 4ddd8e7)
- [x] Slice 4 — watchdog 4 件(ab4e9da)
- [x] Slice 5 — upgrade スモーク + tech-debt RESOLVED + doc 追従
- [ ] Self-review artifact created
- [ ] Verify artifact created
- [ ] Test artifact created
- [ ] Sync-docs artifact created

## Readiness checklist

- [x] Task worktree: `.claude/worktrees/org-debt-batch`(branch `chore/org-debt-batch`、base fde0e84)
- [x] C2-6 スキーマ方針のユーザー確認(org receipts 再指向)
- [x] Codex plan advisory(4 findings すべて反映済み: AC-3 出力契約明文化 / AC-5 自動化必須化 / スライス分割+revert 境界 / 検証ゲート統一)
