# Cross-review triage — org-runtime-retire-loop

- Date: 2026-08-03
- Driver: claude  Reviewer: codex
- Base: main (merge-base d074838), HEAD: a868fca
- Cycle: 1/2
- Summary: ACTION_REQUIRED 2 / WORTH_CONSIDERING 0 / DISMISSED 0

## ACTION_REQUIRED

### AR-1 (Codex P1) `ralph status` がマニフェストを読めない(state-dir 二重連結)

- 対象: `internal/cli/status.go:57`
- 指摘: `runStatus` は `org.NewManifestStore(stateDir)` を呼ぶが、`stateDir` は既に `org.ResolveOrgStateDir` が解決した最終ディレクトリ(例: `<git-toplevel>/.harness/state/org`)。`NewManifestStore(root)` は `root + ".harness/state/org/manifest.jsonl"` を連結するため、実効パスが `<state-dir>/.harness/state/org/manifest.jsonl` に二重化する。org runtime 側(`newOrgRuntimeAt`)は `NewManifestStoreAtPath(filepath.Join(resolvedStateDir, "manifest.jsonl"))` で書くので、**status は実運用のすべての状態で「no org runtime state found」を返す**。
- トリアージ: 実機再現済み(`ralph org spawn --state-dir $tmp --dry-run` → `$tmp/manifest.jsonl` 生成 → `ralph status --state-dir $tmp` が state not found)。既存の status テストは二重パス前提でフィクスチャを書いていたため検出できなかった。真正・修正必須。
- 修正方針: `org.NewManifestStoreAtPath(filepath.Join(stateDir, "manifest.jsonl"))` に変更し、テストのフィクスチャ配置を実書き込みパス(`<state-dir>/manifest.jsonl`)に合わせる。live-write との整合を固定する回帰テスト(spawn dry-run → status)を追加。

### AR-2 (Codex P2) `pick_reviewer` が大文字ドライバ値で同一 CLI を返す

- 対象: `scripts/xreview-helpers.sh:62-66`(+ templates ミラー)
- 指摘: `/cross-review` SKILL.md は `RALPH_PRIMARY_CLI` を case-insensitive と明記するが、`pick_reviewer` は `CODEX`/`Claude` 等をワイルドカード分岐に落とし `codex` を返す。`RALPH_PRIMARY_CLI=CODEX` のとき reviewer が driver と同一 CLI になり、クロスモデルレビューの前提が壊れる。
- トリアージ: 真正(ドキュメント契約との不一致)。修正は小文字正規化 1 行 + テスト追加で安価。
- 修正方針: `tr '[:upper:]' '[:lower:]'` で正規化してから case 分岐。`tests/test-xreview-helpers.sh` に大文字ケースを追加。

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

## 判断

Cycle 1/2(cap 未到達)。ユーザーの常任指示(自律実行・CI パス後マージ可)と本セッションで一貫した Fix 選択実績に基づき **Fix を選択**。修正後、フルパイプライン(/self-review → /verify → /test → /sync-docs → /cross-review)を cycle 2 として再実行する。

---

# Cycle 2(再レビュー)

- Date: 2026-08-03
- Driver: claude  Reviewer: codex
- HEAD: 7d4864e(AR-1/AR-2 修正 fea2ee6 + C2-1/C2-2 修正 47e2eee を含む)
- Cycle: 2/2(cap 到達)
- Summary: ACTION_REQUIRED 0 / WORTH_CONSIDERING 0 / DISMISSED 0

## 結果

Codex 再レビューの結論: 「I did not identify any discrete, actionable regressions in the changed code. The Go and shell verification paths pass (`go test ./...`, `go vet ./...`, and `./scripts/verify.local.sh`).」

指摘ゼロ(Case C)。AR-1(status のマニフェストパス二重連結)・AR-2(pick_reviewer の大文字正規化)の修正はレビュー内で再検証され、追加指摘なし。`/pr` に進む。
