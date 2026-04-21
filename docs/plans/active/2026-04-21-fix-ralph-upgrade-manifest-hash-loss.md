# fix-ralph-upgrade-manifest-hash-loss

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-21
- Related request: `ralph upgrade` で AGENTS.md / CLAUDE.md 以外にも多数ファイルが `modified locally` と誤検出され、さらに pack ファイルが毎回 `removed` と `new file` に二重表示される不具合
- Related issue: N/A
- Branch: fix/ralph-upgrade-manifest-hash-loss

## Objective

`ralph upgrade` が同一バージョン間（例: 3.1.0 → 3.1.0）で安定する状態に戻す。以下 3 点を満たす。
1. 未編集ファイルが次回以降の upgrade で `ActionConflict` として扱われないよう、`ActionSkip` 時にマニフェストハッシュを正しく保持する。
2. pack 内のファイルが `removed from template` と `new file` に同時分類されない diff 判定に修正する。
3. 既に壊れたマニフェスト（`hash = ''`）を持つ既存プロジェクトも、同一バージョン間 upgrade 1 回で自動回復する（強制上書き不要）。

## Scope

- `internal/upgrade/diff.go`: `ActionSkip` に `NewHash` を設定 / 空ハッシュの heal ロジック / `ComputeDiffs` の API を `*scaffold.Manifest` を受け取る形へ拡張。
- `internal/cli/upgrade.go`: base と pack それぞれに対し、マニフェストのサブセットを作って diff 関数へ渡す。pack diff が失敗したときに旧マニフェストの該当エントリを保存して上書き喪失を防ぐ。
- `internal/upgrade/diff_test.go`: 回帰防止テスト群。
- `internal/cli/cli_test.go`: 統合テスト（同一バージョン init → upgrade×2、壊れた manifest の自動回復、pack diff 失敗時の旧エントリ保存）。
- `go build ./...`、`go test ./...`、`./scripts/run-verify.sh` が緑。

## Non-goals

- `upgrade` のインタラクティブ UX 変更（diff 表示の強化、カラー化など）。
- manifest フォーマットの変更（既存キー `hash`, `managed` は維持）。
- pack のディスクレイアウト変更。
- 新たな pack アンインストール機能の追加。
- CLI フラグ追加（`--repair` 等）。heal は無条件の「読み取り時補正」として実装する。

## Assumptions

- 現行 manifest は pack ファイルを `packs/languages/<pack>/<rel>` 形式で保存している（`internal/cli/init.go:157-161`）。
- `ComputeDiffsNoRemovals` の既存呼び出しは本ファイル（`upgrade.go`）内 1 箇所のみ。
- `scaffold.Manifest.Files` は `map[string]ManifestFile`。キー上の prefix フィルタで base / pack を判別できる。
- 空文字ハッシュ (`hash = ''`) は正規の sha256 文字列と衝突しない（sha256 は常に `sha256:<64 hex>`）。

## Affected areas

- `internal/upgrade/diff.go`
- `internal/upgrade/diff_test.go`
- `internal/cli/upgrade.go`
- `internal/cli/cli_test.go`

## Acceptance criteria

- [ ] `ralph init` 直後に同一バージョンで `ralph upgrade` を 2 回連続実行しても、base / pack どちらのファイルも `modified locally` / `removed from template` / `new file` として表示されない（すべて内部 ActionSkip）。
- [ ] `ralph upgrade` 実行後、`.ralph/manifest.toml` の base ファイルエントリが空文字ハッシュ (`hash = ''`) を持たない。
- [ ] `hash = ''` を含む破損マニフェストに対し、同一バージョンで `runUpgrade` を 1 回呼び出すと、ディスク内容がテンプレートと一致するファイルは強制確認なしで `ActionSkip` 扱いとなりハッシュが復旧する（ユーザが実際に編集したファイルだけが conflict として残る）。
- [ ] pack ファイルが同一 upgrade 中で `removed from template` と `new file` の両方に現れない。
- [ ] 旧マニフェストに存在する pack の `scaffold.PackFS` や diff 計算が失敗した場合、そのエントリが新マニフェストで失われない（旧エントリが保持される）。
- [ ] 既存テスト `TestComputeDiffs_AutoUpdate` / `_Conflict` / `_AddNewFile` / `_RemoveFile` が引き続き緑。
- [ ] 新規テスト: (a) ActionSkip で `NewHash` が非空、(b) 名前空間付き manifest + pack FS で pack ファイルが二重分類されない、(c) 空ハッシュエントリ + ディスク一致で `ActionSkip` に heal される、(d) pack diff 失敗時に旧エントリが保持される。
- [ ] `go test ./...` 緑、`./scripts/run-verify.sh` 緑。

## Implementation outline

1. **diff.go: API 拡張**  
   現状 `ComputeDiffs(manifestPath, targetDir, newFS)` は内部で manifest を読んで走査と removal sweep を両方行うため、呼び出し側から manifest のサブセットを渡せない。以下を追加する。
   - `ComputeDiffsWithManifest(m *scaffold.Manifest, targetDir string, newFS fs.FS, checkRemovals bool) ([]FileDiff, error)` を新設。
   - 既存 `ComputeDiffs` / `ComputeDiffsNoRemovals` は `scaffold.ReadManifest` の結果を渡すラッパへ書き換える（シグネチャ互換を維持）。
2. **diff.go: ActionSkip に NewHash**  
   `if newHash == mf.Hash` ブランチで `NewHash: newHash`、`OldHash: mf.Hash` を詰める。
3. **diff.go: 空ハッシュの heal**  
   `mf.Hash == ""`（破損マニフェスト）の場合、
   - `diskHash == newHash` なら `ActionSkip` + `NewHash: newHash`（ディスクがテンプレートと一致 → 無風で hash 復旧）。
   - `diskHash != newHash` なら `ActionConflict`（ユーザ編集の可能性、既存の挙動と同じ）。
   通常パス（`mf.Hash != ""`）の分岐ロジックは変更しない。
4. **upgrade.go: manifest を base/pack でサブセット化**  
   - `oldManifest` を pre-read（既存コードにある）。
   - base 用に `baseManifest` を生成: `packs/languages/` で始まらないキーだけを含む。
   - pack ごとに `packManifest` を生成: `packs/languages/<pack>/` で始まるキーを prefix 剥がしたもの。
   - それぞれ `ComputeDiffsWithManifest` を呼ぶ。pack 側は `checkRemovals=false`。
   - pack diff 結果の `Path` を `packs/languages/<pack>/` で名前空間付けして集約。
5. **upgrade.go: 部分失敗時の manifest 保護**  
   pack FS のロード失敗 / diff 計算失敗時、対応する pack の旧 manifest エントリを新 manifest に直接コピーしておく（警告のみでスキップしない）。これにより pack が一時的に壊れても追跡が失われない。
6. **テスト追加**  
   - `diff_test.go`:
     - `TestComputeDiffs_Skip_PreservesHash`
     - `TestComputeDiffsWithManifest_PackPrefixedKeys`（名前空間付き manifest + pack FS で ActionSkip を返す）
     - `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate`
     - `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers`
   - `cli_test.go`:
     - `TestRunUpgrade_SameVersionIsIdempotent`（init → upgrade → upgrade で 2 回目の conflict 0 件、manifest に空ハッシュなし）
     - `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure`（スタブ経由、unknown pack 名を installedPacks に含めるなどで failure 発生）
7. `./scripts/run-verify.sh` 実行。
8. `docs/specs/2026-04-16-ralph-cli-tool.md` と `docs/recipes/*` に upgrade の冪等性 / heal 挙動を追記（drift 確認の上）。

## Verify plan

- Static analysis checks: `go vet ./...`, `gofmt -l internal/`, `./scripts/run-verify.sh`。
- Spec compliance criteria to confirm: 本プランの Acceptance criteria すべて。
- Documentation drift to check:
  - `docs/specs/2026-04-16-ralph-cli-tool.md` の upgrade セクション（heal 動作 / pack 名前空間）。
  - `docs/recipes/*` のうち `ralph upgrade` に触れるファイル。
  - `AGENTS.md` / `CLAUDE.md`（挙動変更なのでマップ記述の齟齬を確認。通常は更新不要）。
- Evidence to capture:
  - `docs/reports/verify-<slug>.md`
  - `go test ./... -count=1 -run Diff|Upgrade` の stdout を reports に添付。

## Test plan

- Unit tests:
  - `TestComputeDiffs_Skip_PreservesHash` — manifest と newFS が同一ハッシュ、ディスクも同一 → `ActionSkip` かつ `NewHash == HashBytes(content)`。
  - `TestComputeDiffsWithManifest_PackPrefixedKeys` — pack FS ルート相対ファイル + 名前空間付き manifest で `ActionSkip`。`ActionAdd` にならないこと。
  - `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` — manifest hash が空、disk == newFS → `ActionSkip` + `NewHash` 復旧。
  - `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` — manifest hash が空、disk != newFS → `ActionConflict`（既存挙動維持）。
- Integration tests:
  - `TestRunUpgrade_SameVersionIsIdempotent` — `executeInit` → `runUpgrade` × 2 で、2 回目の `runUpgrade` 後も manifest 内にあるはずの base ファイルエントリが空ハッシュを持たない。stdin を閉じた非対話モードでも走る。
  - `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` — `installedPacks` に存在しない pack 名を含め、pack diff が失敗しても旧 manifest 側の正常 pack エントリが保持される。
- Regression tests: `TestComputeDiffs_AutoUpdate` / `_Conflict` / `_AddNewFile` / `_RemoveFile` / `TestRunUpgrade_AutoUpdate` 緑維持。
- Edge cases:
  - manifest 自体が無い（既存 error パスのまま）。
  - pack が途中で無効化された場合（手順 5 の保護パスがカバー）。
  - 空ハッシュで disk が newFS と一致するケース（heal）。
  - 空ハッシュで disk が newFS と異なるケース（conflict）。
- Evidence to capture: `docs/reports/test-<slug>.md`、`go test ./... -count=1` の出力を report に貼る。

## Risks and mitigations

- **R1: API 拡張（`ComputeDiffsWithManifest`）で既存呼び出し元を壊す**  
  → 既存シグネチャは薄いラッパで維持。`ComputeDiffsNoRemovals` は unit test で依存があるので変更しない。
- **R2: heal ロジックが意図せぬ上書きを招く**  
  → heal は「disk == newFS」の場合のみ `ActionSkip`（書き込みなし）。disk が違えば従来通り conflict で確認。書き込みを発生させないため安全。
- **R3: 部分失敗時の manifest 旧エントリ保持が古い情報を遅延リークさせる**  
  → 警告を stderr に出し、`docs/tech-debt/` に「将来 pack 明示的アンインストールフラグを入れる」メモを残す（spec change を伴う場合のみ）。
- **R4: 既存破損 manifest を持つリポでの大量 file-write**  
  → heal は ActionSkip でディスク書き込みなし、manifest の hash フィールドだけを書き換える。影響を最小化。

## Rollout or rollback notes

- ロールアウト: `/work` → post-implementation pipeline → `/pr` → CI 緑 → 人レビュー → main マージ。
- ロールバック: revert PR で復旧可能。heal ロジックを revert しても、(1) の新しい skip+NewHash は健全なので、revert 後も新規 init は壊れない。既存の壊れた manifest が再度壊れる可能性はあるが、データ破損ではなく UX バグ再燃の範囲。

## Open questions

- `ComputeDiffsNoRemovals` は現状唯一の pack diff 経路。今回 API が `ComputeDiffsWithManifest` に集約されるため、将来的にはこちらを deprecated にする価値があるか（今回は維持）。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] Sync-docs artifact created
- [ ] PR created
