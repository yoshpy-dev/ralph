# ralph upgrade: detect local edits and show unified diff

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-22
- Related request: `ralph upgrade` がテンプレート未変更のローカル編集を検知せず、また diff 表示が hash のみで実差分が見えない
- Related issue: N/A
- Branch: feat/upgrade-detect-local-edits

## Objective

`ralph upgrade` が「テンプレート未変更 + ローカル編集あり」のファイルを conflict として検知し、`[o]verwrite / [s]kip / [d]iff` を提示できるようにする。`[d]iff` 選択時は hash ではなく unified diff で実内容差分を表示する。

## Scope

- `internal/upgrade/diff.go` の判定ロジック拡張（テンプレート未変更でも disk drift があれば `ActionConflict`）
- `internal/upgrade/` に unified diff 生成ユーティリティを追加
- `internal/cli/upgrade.go` の `resolveConflict` を書き換え、実 diff を表示
- `skip` の収束契約を追加: `skip` を選んだエントリは manifest に `Managed=false` + `Hash=diskHash` を書き込み、以降 silent skip にする（prompt storm 回避）
- `Managed=false` のエントリを `ComputeDiffsWithManifest` で尊重（「user-accepted local variant」として扱い、auto-update もプロンプトも抑制）
- `resolveConflict` の I/O を DI 可能に切り出し、対話パスをテスト可能にする
- 既存テストの更新 + 新規ケース追加（`internal/upgrade/diff_test.go`, `internal/cli/cli_test.go`）

## Non-goals

- 三者マージ（template / manifest / local）の自動合成は行わない。ユーザー選択のみ。
- インタラクティブ UI の刷新（TUI 組み込みなど）は対象外。
- `ActionAdd` 経路（manifest に無いローカル差異）の既存挙動は変更しない。
- カラー表示は任意。端末判定が不要な範囲に留める。

## Assumptions

- 依存追加は避け、unified diff は自前実装（LCS ベースの最小限のもの）でよい。
- 既存の `packs/languages/...` 名前空間処理は変更不要（base と pack 両方で同じ `ComputeDiffsWithManifest` を使っているため、diff.go の修正で両方カバーされる）。
- ハッシュが空の heal パス（既存の特殊処理）は挙動を壊さない。
- `ManifestFile.Managed` は現在 schema 定義のみで upgrade ロジックから未使用のため、意味を新規付与しても既存ユーザの manifest を壊さない（既存エントリは全て `Managed=true`）。
- `Managed=false` エントリを silent skip にする挙動は、将来 `--resync` / `--adopt` 相当のエスケープハッチで上書き可能（本プランではスコープ外）。

## Affected areas

- `internal/upgrade/diff.go` —
  - テンプレート未変更分岐（L156 周辺）を `diskHash == mf.Hash` で条件分岐
  - `Managed=false` エントリは `ActionSkip`（NewHash/DiskHash だけ埋めて manifest を維持、プロンプトなし）
- `internal/upgrade/unified_diff.go`（新規）— LCS ベースの簡易 unified diff
- `internal/upgrade/unified_diff_test.go`（新規）
- `internal/upgrade/diff_test.go` — 新規ケースを追加、既存 `TestComputeDiffs_Skip_PreservesHash` は維持
- `internal/cli/upgrade.go` —
  - `resolveConflict` の diff 表示を unified diff に差し替え
  - prompt I/O を DI 可能に（stdin reader / stdout writer / prompt 関数を引数化）
  - `skip` 選択で `manifest.Files[d.Path] = {Hash: diskHash, Managed: false}` を書き込み（prompt storm 回避）
  - disk read 失敗時は警告ログ + hash-only フォールバック
- `internal/scaffold/manifest.go` — `Managed=false` を書けるヘルパー（既存 `SetFile` を壊さず、`SetFileUnmanaged(path, hash)` など最小追加）
- `internal/cli/cli_test.go` — 対話パスの回帰ケース追加
- 関連ドキュメント: 本プランファイル、README / docs 内 upgrade 挙動記述は `/sync-docs` 段で追従

## Acceptance criteria

- [ ] テンプレート未変更 + ローカル編集ありで `ralph upgrade` を実行すると conflict プロンプトが出る（`ActionConflict`）
- [ ] テンプレート未変更 + ローカル編集なしは依然 `ActionSkip`（manifest 空ハッシュの heal も含めて回帰なし）
- [ ] テンプレート変更 + ローカル編集ありも従来通り `ActionConflict`
- [ ] `[d]iff` を選択すると `--- local` / `+++ template (version)` の unified diff が出力され、変更行に `-` / `+`、周辺行に空白プレフィクスが付く（local → template の方向で、`-` はローカル側の行、`+` はテンプレート側の行）
- [ ] `[d]iff` 表示後に `[o]verwrite / [s]kip / [d]iff` を再度プロンプトし、`d` の連打でループに入らない（2 度目の `d` 入力は無視 or 再表示、入力 EOF で skip）
- [ ] disk 読み取り失敗時は警告を出し、hash サマリ付きで選択肢継続（abort しない）
- [ ] `overwrite` 選択でローカルがテンプレート内容に一致し、manifest が `{Hash: newHash, Managed: true}` に更新される
- [ ] `skip` 選択でローカルが維持され、manifest エントリが `{Hash: diskHash, Managed: false}` に更新される（次回 upgrade で silent skip 収束）
- [ ] `Managed=false` のエントリは `ComputeDiffsWithManifest` で `ActionSkip` になり、プロンプトも auto-update も発生しない
- [ ] `ralph upgrade --force` は従来通り全て上書き（`Managed` は true に戻る）
- [ ] `./scripts/run-verify.sh` と `go test ./...` が緑

## Implementation outline

1. `internal/upgrade/unified_diff.go` を追加し、`UnifiedDiff(oldText, newText []byte, oldLabel, newLabel string) string` を実装（LCS → hunk 化、context 3 行）。副作用なし、pure func。
2. `internal/upgrade/unified_diff_test.go` を追加（同一入力 / 追加のみ / 削除のみ / 置換 / 空ファイル / 末尾改行差異）。
3. `internal/upgrade/diff.go` の分岐を修正:
   - `!mf.Managed` のエントリは `ActionSkip`（`OldHash=mf.Hash`, `NewHash=newHash`, `DiskHash=diskHash` を埋めるが、manifest はそのまま保持）
   - `newHash == mf.Hash` かつ `diskHash == mf.Hash` → `ActionSkip`（既存踏襲）
   - `newHash == mf.Hash` かつ `diskHash != mf.Hash` → `ActionConflict`（`OldHash=mf.Hash`, `NewHash=newHash`, `NewContent=content`, `DiskHash=diskHash`）
4. `internal/upgrade/diff_test.go` に以下を追加:
   - テンプレート未変更 + ローカル編集 → Conflict
   - `Managed=false` エントリ → Skip（プロンプトも auto-update も抑制）
5. `internal/scaffold/manifest.go` に `SetFileUnmanaged(path, hash string)` を追加（`Managed=false` を明示的に書ける）。既存 `SetFile` は非変更。
6. `internal/cli/upgrade.go` の `resolveConflict`:
   - シグネチャを `(d FileDiff, absDir, version string, in io.Reader, out io.Writer) string` に変更（DI）
   - `[d]iff` 選択時、`os.ReadFile(filepath.Join(absDir, d.Path))` で disk を読み、`upgrade.UnifiedDiff(d.NewContent, localBytes, fmt.Sprintf("template (%s)", version), "local")` を出力
   - disk 読取失敗時は警告 + hash-only 表示にフォールバック、選択肢継続
   - hash サマリ（1 行）は障害調査用途で末尾に残す
   - diff 表示後も `[o]verwrite / [s]kip / [d]iff` を再表示（d の連打でも再描画のみ、ループ shadow なし）
7. `runUpgrade` の conflict 処理:
   - `overwrite` → 既存通り `manifest.SetFile(path, newHash)` で `Managed=true`
   - `skip` → `manifest.SetFileUnmanaged(path, diskHash)` で収束。skip 時のログに「future `ralph upgrade` で silent skip になる」旨を追記
8. `force` パス: 既存通り overwrite（`Managed=true`）。
9. `ActionAdd` / `ActionRemove`: 非変更。
10. `./scripts/run-verify.sh` を通し、evidence は後段 `/test` で保存。

## Verify plan

- Static analysis checks: `go vet ./...`, `gofmt -l`, `./scripts/run-verify.sh`
- Spec compliance criteria to confirm:
  - 受入基準の各項目が `diff_test.go` / `unified_diff_test.go` / `cli_test.go` のいずれかで検証されている
  - `.claude/rules/testing.md` に沿い「ローカル編集 → conflict」の edge case が追加されている
- Documentation drift to check: `README.md` / `docs/` 内の upgrade 挙動記述。実差分が発生したら `/sync-docs` 段で更新。
- Evidence to capture: `go test ./internal/upgrade/... -v` の出力、`ralph upgrade` 手動トレース（テンプレ未変更 + ローカル編集シナリオ）

## Test plan

- Unit tests:
  - `TestComputeDiffs_LocalEditWithUnchangedTemplate` — 新規（テンプレ未変更 + ローカル編集 → Conflict）
  - `TestComputeDiffs_Unmanaged_IsSilentSkip` — 新規（`Managed=false` → Skip、プロンプト抑制）
  - `TestUnifiedDiff_*` — 追加 / 削除 / 置換 / 空 / 末尾改行差異 / 同一入力（空文字列）
- Integration tests（`internal/cli/cli_test.go`）:
  - `TestUpgrade_ForceOverwritesLocalEdit` — `--force` 経由の回帰確認
  - `TestUpgrade_InteractiveOverwrite_WritesManaged` — DI した入力に `o\n` を流し、manifest が `{Hash: newHash, Managed: true}` になることを確認
  - `TestUpgrade_InteractiveSkip_WritesUnmanaged` — 入力に `s\n` を流し、manifest が `{Hash: diskHash, Managed: false}` になることを確認
  - `TestUpgrade_InteractiveDiff_ShowsUnifiedDiff` — 入力に `d\ns\n` を流し、stdout に `--- template` / `+++ local` と `-`/`+` プレフィクス行が含まれること
  - `TestUpgrade_NextRunAfterSkip_IsSilent` — skip 後に再度 upgrade を実行してもプロンプトが発生しないこと（収束性）
  - `TestUpgrade_DiskReadFailure_FallsBackToHash` — diff 表示対象が一時的に欠落した場合のフォールバック
- Regression tests:
  - `TestComputeDiffs_Skip_PreservesHash`
  - `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate`
  - `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers`
  - `TestComputeDiffs_AddBecomesConflictWhenDiskDiffers`
  - `TestComputeDiffsWithManifest_PackPrefixedSubset`
- Edge cases: 改行コード差異（LF/CRLF）、末尾 newline の有無、空ファイル対空ファイル、EOF で入力が途切れた場合の扱い
- Evidence to capture: `go test ./... -count=1` 出力、対話パスの stdout スナップショット

## Risks and mitigations

- Risk: 「テンプレ未変更 + ローカル編集」状態のユーザで upgrade 毎に conflict プロンプトが連打される（prompt storm）
  → `skip` 時に `Managed=false` + `Hash=diskHash` を書き込み、以降 silent skip にする収束契約で解決。将来 `--resync` / `--adopt` で再度管理下に戻せる設計余地を残す。
- Risk: unified diff 自前実装のバグで誤表示
  → 置換 / 追加 / 削除 / 空 / 末尾改行の unit テストで網羅、決定論的出力を保証。
- Risk: `FileDiff` フィールド追加で既存呼び出しが壊れる
  → disk 読み直し方針でフィールド追加を回避。
- Risk: 対話パス新規 I/O（`os.ReadFile`）の失敗で upgrade が abort
  → disk 読取失敗時は警告 + hash-only 表示にフォールバック、選択肢継続で abort させない。`TestUpgrade_DiskReadFailure_FallsBackToHash` で検証。
- Risk: `Managed=false` の意味変更で既存 manifest が意図しない silent skip になる
  → 既存 manifest は全エントリ `Managed=true`（`SetFile` が常に true を書いてきた）ため影響なし。新規に `Managed=false` を書くのは `skip` resolution 経由に限定。
- Risk: Windows パス / CRLF
  → 既存の `filepath.ToSlash` 踏襲。unified diff は `\n` 区切り前提で CRLF もそのまま扱う（テストケース追加）。

## Rollout or rollback notes

- Rollout: 単一 PR。`--force` で影響一時回避可能。リリースノートで挙動変更（「ローカル編集検知 + skip で unmanaged 化」）と、将来の `--resync` 計画を明記。
- Rollback: revert 単発で挙動復帰。manifest schema は `Managed` 既存フィールドの意味付けのみ追加で schema 破壊変更なし。既存の `Managed=true` エントリは旧コードでも新コードでも同じ挙動。

## Open questions

- `[d]iff` の出力にカラー付与 → 初期スコープ外。プレーンで開始。
- hash サマリを残すか → 残す（障害調査時の一意識別に有用）。
- `--resync <path>` / `--adopt` のエスケープハッチ → スコープ外、後続 PR で。

## Progress checklist

- [x] Plan reviewed (Codex adversarial review incorporated: Managed=false convergence + DI + fallback)
- [x] Branch created (`feat/upgrade-detect-local-edits`)
- [x] Implementation started
- [x] Acceptance criteria met (unit + integration tests)
- [x] `./scripts/run-verify.sh` green
- [x] Review artifact created (`docs/reports/self-review-2026-04-22-upgrade-detect-local-edits.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-04-22-upgrade-detect-local-edits.md`)
- [x] Test artifact created (`docs/reports/test-2026-04-22-upgrade-detect-local-edits.md`)
- [ ] PR created
