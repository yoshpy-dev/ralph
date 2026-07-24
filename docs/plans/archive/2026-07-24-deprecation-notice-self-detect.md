# deprecation-notice-self-detect

- Status: Draft
- Owner: Claude Code
- Date: 2026-07-24
- Related request: scripts/ralph の deprecation notice が scripts/ を PATH に載せた場合に自分自身を Go CLI と誤検出する
- Related issue: #134
- Type: fix
- Branch: fix/134/deprecation-notice-self-detect

## Objective

`scripts/ralph`(レガシーシェルCLI)末尾の deprecation notice が、`scripts/` を PATH に載せて `ralph` として起動した場合に `command -v ralph` が自分自身に解決されて誤発火する問題を修正する。notice は「PATH で解決される `ralph` がこのスクリプト自身ではない」場合のみ表示する。

## Scope

- `scripts/ralph` の deprecation notice 条件を `-ef`(同一 inode 判定)ベースの自己除外付きに変更
- `templates/base/scripts/ralph` に同一修正を適用(byte-identical 維持)
- `tests/test-ralph-deprecation-notice.sh` に回帰ケース2件を追加:
  1. 実リポジトリの `scripts/` を PATH 先頭に載せて `ralph status` として起動(自己解決)→ notice が出ない **かつ** スクリプトが正常起動している(source エラー等の早期失敗でないこと)を assert
  2. 別パスのダミー `ralph` バイナリが PATH にある状態で `./scripts/ralph` 起動 → notice が出る(既存ケース1で実質カバー済みだが、issue の受け入れ条件に沿って明示)

  注: issue AC5 記載の「一時ディレクトリに `scripts/ralph` をコピー」方式は採らない。`scripts/ralph` は起動時に同ディレクトリの `ralph-config.sh` / `ralph-common.sh` を source するため、単体コピーでは notice 判定に到達する前に失敗し false green になる(Codex plan advisory HIGH finding)。

## Non-goals

- レガシーシェル CLI の retirement 自体(docs/tech-debt/README.md で別途トラッキング)
- Go CLI (`cmd/ralph/`) 側の変更
- notice メッセージ文言の変更
- `command -v` が関数/alias を返すケースの対応(非対話シェルでは実質発生しない — issue 補足どおり)

## Assumptions

- `scripts/ralph` は `#!/usr/bin/env bash` で実行され、`[ file1 -ef file2 ]` が macOS の bash 3.2 / GNU bash の双方で利用可能
- `$0` は起動時のパス(相対でも可)を保持しており、`-ef` は inode 比較のためパス表記の差異に影響されない
- `scripts/ralph` と `templates/base/scripts/ralph` は byte-identical 運用(`cmp` で確認済み)

## Affected areas

- `scripts/ralph`(647–649行付近の deprecation notice ブロック)
- `templates/base/scripts/ralph`(同上、byte-identical)
- `tests/test-ralph-deprecation-notice.sh`(回帰ケース追加)

## Design decisions

Critical forks: None

- 実装方式は issue #134 の修正案どおり `test -ef` による自己 inode 比較を採用。`realpath`/`readlink -f` のプラットフォーム差異を回避でき、最小・堅牢。代替案(パス文字列比較、`readlink -f` 正規化)は macOS 互換性・symlink 対応で劣るため不採用。
- 回帰テストは issue AC5 の「単体コピー」方式ではなく実 `scripts/` ディレクトリを PATH に載せる方式を採用(Codex plan advisory HIGH finding への対応: 単体コピーは sibling source 失敗により false green になる)。

## Acceptance criteria

- [x] AC1: Go CLI 未導入 + `PATH="$PWD/scripts:$PATH" ralph <cmd>` 起動 → notice が stderr に出ない(テストケース5で検証)
- [x] AC2: Go CLI(ダミー)が別パスに導入済み + `./scripts/ralph <cmd>` 起動 → notice が stderr に出る(既存テストケース1で検証)
- [x] AC3: `RALPH_NO_DEPRECATION=1` → 条件によらず notice が出ない(既存テストケース3で検証)
- [x] AC4: `scripts/ralph` と `templates/base/scripts/ralph` が引き続き byte-identical(`cmp` exit 0 で確認)
- [x] AC5: `tests/test-ralph-deprecation-notice.sh` に自己検出回帰ケース(notice 不在 + 正常起動の両方を assert)が追加され、全ケース pass(7 passed, 0 failed)

## Implementation outline

1. `scripts/ralph` の deprecation notice ブロックを issue の修正案に従い書き換え:
   ```bash
   if [ -z "${RALPH_NO_DEPRECATION:-}" ]; then
     _ralph_resolved="$(command -v ralph 2>/dev/null || true)"
     if [ -n "$_ralph_resolved" ] && ! [ "$_ralph_resolved" -ef "$0" ]; then
       printf '...notice...' >&2
     fi
   fi
   ```
2. 同一内容を `templates/base/scripts/ralph` にコピー(`cp` で byte-identical を保証)
3. `tests/test-ralph-deprecation-notice.sh` に回帰ケースを追加:
   - 実リポジトリの `scripts/` を PATH 先頭に載せ(`PATH="$PROJECT_ROOT/scripts:/usr/bin:/bin"`)、コマンド名 `ralph` で `ralph status --no-color` を起動 → stderr に `this shell entrypoint is legacy` が含まれない **かつ** 早期失敗でないこと(source エラーが stderr に無い、または status 出力に到達している)を assert
   - (既存ケース1がダミーバイナリ経由の notice 表示を検証済み — ラベルを確認し必要なら明確化)
4. テスト実行 + `./scripts/run-verify.sh` で検証

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`(shellcheck 含む静的検証)
- Spec compliance criteria to confirm: AC1–AC5 が issue #134 の受け入れ条件と一致
- Documentation drift to check: AGENTS.md の `scripts/` 記述(「prints deprecation notice when Go `ralph` binary is on PATH」)— 挙動の本質は変わらないため更新不要見込みだが /sync-docs で確認
- Evidence to capture: `cmp scripts/ralph templates/base/scripts/ralph` の結果、テスト実行ログ

## Test plan

- Unit tests: `tests/test-ralph-deprecation-notice.sh`(既存4ケース + 追加ケース)
- Integration tests: なし(シェル単体で完結)
- Regression tests: 既存の deprecation notice テスト4ケースが引き続き pass すること
- Edge cases:
  - PATH 先頭に scripts/ を載せて `ralph` 名で起動(自己解決)→ notice なし
  - symlink 経由で自分自身に解決されるケース(`-ef` が inode 比較のためカバー)
  - `RALPH_NO_DEPRECATION=1` の抑止が引き続き機能
- Evidence to capture: テスト出力(pass/fail カウント)

## Risks and mitigations

- **リスク**: `-ef` 演算子が想定外の環境(非bash sh)で未サポート → **緩和**: スクリプトは `#!/usr/bin/env bash` 固定であり、テストは実際の実行で検証する
- **リスク**: `$0` が期待と異なる値になる起動形態(source 読み込み等)→ **緩和**: `scripts/ralph` は実行専用エントリポイントで source される設計ではない
- **リスク**: テンプレート側の反映漏れで byte-identical が崩れる → **緩和**: `cp` で反映し `cmp` を evidence として記録(AC4)

## Rollout or rollback notes

- 単一コミットの小変更。問題があれば revert 一発で戻せる
- scaffold 済み下流プロジェクト(例: koalive)へは次回 `ralph upgrade` で配布される

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/134/deprecation-notice-self-detect)
- [x] Implementation started (implementer subagent, commit 46d8ff0)
- [x] Review artifact created (docs/reports/self-review-deprecation-notice-self-detect.md)
- [x] Verification artifact created (docs/reports/verify-deprecation-notice-self-detect.md)
- [x] Test artifact created (docs/reports/test-deprecation-notice-self-detect.md; 7 passed, 0 failed)
- [ ] PR created
