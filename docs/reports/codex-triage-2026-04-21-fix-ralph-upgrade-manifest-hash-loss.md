# Codex triage report: fix-ralph-upgrade-manifest-hash-loss

- Date: 2026-04-21
- Plan: `docs/plans/archive/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Total Codex findings: 2 (both P2)
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

> 注記: 初回 `/codex-review` は usage limit により中断。リセット後のリトライで本レポートを生成。

## Triage context

- Active plan: `docs/plans/archive/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Self-review report: `docs/reports/self-review-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Verify report: `docs/reports/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Implementation context summary: base と pack の diff sweep を `splitManifestForBase` / `splitManifestForPack` でマニフェスト subset 化して分離。pack 側は `ComputeDiffsWithManifest(..., false)` で removal sweep を無効化。`preservePackEntries` で PackFS / diff 失敗時に旧エントリを保持。

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | [P2] Restore removal detection for files deleted from an existing pack. `internal/cli/upgrade.go:122-124` の `ComputeDiffsWithManifest(packManifest, packDir, packFS, false)` は pack FS にもう存在しないマニフェストエントリを `ActionRemove` として報告しない。base sweep は `splitManifestForBase` で `packs/languages/*` を除外済みなので、結果として "pack ファイルがテンプレートから削除された" 通知パス（以前は base sweep 経由で発火していた副作用的動作）が完全に消失。新マニフェスト再構築で該当エントリが静かに脱落、ディスク上のファイルは残る。| リグレッション。user が fix した元のバグ (pack を毎回 Remove と誤判定) は base sweep 側で既に防げているので、pack 側で `checkRemovals=true` にしても二重分類は起きない（base 側は pack prefix を完全に除外、pack 側はファイルが双方に同時に存在しないのでAdd と Remove が同一 path に発火しない）。Axis1=Yes (real regression), Axis2=Yes (small diff, high signal value)。 | `internal/cli/upgrade.go:124` |
| 2 | [P2] Don't preserve packs that disappeared from the release. `upgrade.go:116-119` の `PackFS` エラーは `unknown language pack`（新リリースで削除/改名された正常ケース）にも発火。結果として削除された pack のエントリが永遠に残り、以降の upgrade で毎回 "Warning: pack X" を出し続け、ユーザに「この pack は消えました」という signal を渡せない。| 真の regression。`preservePackEntries` は "transient エラー" 用として plan で設計したが、`scaffold.PackFS` の error 種別を区別しておらず、永続的な削除ケースにも発火。修正方針: `scaffold.AvailablePacks()` で事前チェックし、現リリースに存在しない pack は preserve 対象外（drop + `Meta.Packs` からも除外）。Axis1=Yes, Axis2=Yes。| `internal/cli/upgrade.go:114-128` |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|

（なし）

## DISMISSED

| # | Codex finding | Dismissal reason | Category |
|---|---------------|------------------|----------|

（なし）

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

---

## Round 2 (post-d16cb4d)

- Codex findings: 2 (both P2)
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

### ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 3 | [P2] Avoid repeating pack-removal notices on every upgrade — `internal/cli/upgrade.go:143-147`. `checkRemovals=true` にしたことで削除された pack ファイルが `ActionRemove` で通知されるが、現在の ActionRemove 分岐は `manifest.SetFile(d.Path, d.OldHash)` でエントリを保持するため、次回 upgrade でも同じファイルが再通知され続ける。idempotency 契約を破る。| 真の regression（pre-existing バグを私の pack-removal 復活で顕在化）。base ファイルにも同じロジックが効いているため、過去も "removed from template" が永続警告として出続けていた可能性が高い。修正方針: ActionRemove で manifest エントリをドロップする（"review and delete manually" とユーザに伝えた通り、次回以降 manifest からも外す）。Axis1=Yes, Axis2=Yes。| `internal/cli/upgrade.go:225-229`, `internal/upgrade/diff.go:171-183` |

### WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 4 | [P2] Normalize pack manifest keys in tests for Windows — `internal/cli/cli_test.go:181-182`. テストが `"packs/languages/golang/README.md"` のようにスラッシュで直書きしているが、`executeInit` は `filepath.Join` で manifest キーを作るので Windows では `\` 区切りになり lookup が失敗する。| 実装側は `splitManifestForPack` が `filepath.ToSlash` で正規化済みで正しく動作する。テストの portability だけの問題。CI が Linux/macOS しか回っていない現状で regression 相当ではなく WORTH_CONSIDERING。ただし `filepath.Join` へ書き換えるのは 5 行・低リスクなので同時に直しても良い。Axis1=Debatable, Axis2=Debatable。| `internal/cli/cli_test.go:182, 265, 269, 282, 298, 319, 323` |


