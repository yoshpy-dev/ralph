# Walkthrough: codex-hooks-json-wiring(Codex 配布 hooks 配線の hooks.json 移行)

- Date: 2026-08-21
- Branch: fix/codex-hooks-json-wiring
- Diff: 41 files, +3,009 / -500(コミット 36 件)
- Plan: docs/plans/archive/2026-08-20-codex-hooks-json-wiring.md(PR 時にアーカイブ)
- Canonical ref: docs/tech-debt/README.md の配布配線ギャップ行(#148)+ 2026-08-19〜21 の実機 live-fire 検証

## 背景

overlay-scaffold-v2 Phase 5 が ship したインライン TOML `[[hooks.*]]` 配線は、codex-cli 0.147.0 の実機で実行されないことが判明した(#148 で記録)。本 PR は配線を実機で発火が実証済みの `hooks.json` 表現へ移行し、v5.0.0 リリースの前提を満たす。

## 読む順番(レビュー導線)

1. **`.codex/hooks.json`(新規、root/template byte-identical)** — 配布配線の本体。matcher `Edit|Write|MultiEdit|apply_patch`(公式仕様: matcher はツール名への正規表現、ファイル編集の実ツール名は `apply_patch`)、command は `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse`(hook の cwd はセッション cwd のため、git root へ cd してから dispatcher を起動 — dispatcher の cwd 相対 `.d` 解決と整合)。`.codex/config.toml` は hooks エントリを撤去し参照コメントのみ。
2. **hook スクリプトの apply_patch 対応** — Codex の編集ペイロードは `tool_input.file_path` を持たず、パッチ本文が `tool_input.command` に載る。`post_edit_verify.sh` / `check_mojibake.sh`(+twins)は envelope 行(`*** Add/Update/Delete File:` / `*** Move to:`)から対象パスを導出し、ペイロードの `cwd`(セッション cwd)で相対パスを解決してから処理する(git-root 実行との干渉を解消)。edited-files.log は root 相対で記録。
3. **doctor の反転(`checkCodexEffectiveConfig`)** — hooks.json を source of truth として検査: 存在/JSON 妥当性/公式スキーマ形(4 種の壊れ形をネガティブテストで固定)/dispatcher ルーティング/直接スクリプト参照の警告/`[features].hooks` の明示 false・非 boolean 警告/config.toml への hooks 再導入の併存警告。すべて warn レベル(環境チェック、--strict 非対象)。
4. **テスト面** — `tests/test-hook-wiring.sh`(30 ケース: hooks.json ルーティング+cd 前置ゲート+空白耐性 TOML テーブル検出の自己テスト)、`tests/test-post-edit-verify.sh`(新規 24 ケース)、`tests/test-check-mojibake.sh`(+4 ケース: apply_patch fixture、相対パス回帰)。
5. **live-fire 証跡(docs/evidence/)** — slice-1(git-root 形の発火+ペイロード捕捉)、AC-2(b)(fresh `ralph init` fixture でも bypass 付きで発火 — P5 期の不発は TOML 表現が原因だったことの裏付け)、cycle-3(サブディレクトリ起動での発火+apply_patch 由来の edited-files.log 記録)。
6. **trust UX ドキュメント** — `.codex/README.md` / `.codex/hooks/README.md` / recipes: hooks は per-command-hash の対話承認が必要で、未承認の `codex exec` は無警告 skip する事実を明記。

## 品質ゲートの履歴

パイプライン 3 サイクル(cap 2→3、引き上げは操作者承認):

| cycle | self-review | verify | test | cross-review(codex) |
|---|---|---|---|---|
| 1 | **HIGH 1**(hooks/README.md の旧配線説明残存)+ MEDIUM 5 + LOW 5 → 修正 | PASS | PASS(621 shell + Go 8/8) | P2 1 件(features.hooks 非 boolean の無言スキップ)→ 修正 |
| 2 | MEDIUM 2 + LOW 5 → 修正 | PASS | PASS | **P2 2 件**(サブディレクトリ起動で dispatcher 空回り / apply_patch ペイロード不適合による実質 no-op)→ cap 引き上げ後修正+live-fire 再実証 |
| 3 | **HIGH 1**(git-root 実行と envelope 相対パスの干渉 — session-cwd 解決で修正)+ MEDIUM 4 + LOW 4 → 修正 | PASS | PASS(653 shell + Go 8/8) | P2 2 件(doctor warn 精度)→ **Known gap 化(操作者承認)** |

## Known gap(操作者承認済み)

doctor の warn 精度 2 件: (1) 併存警告が `[hooks.state...]` メタデータに誤発火しうる(プロジェクト config での発生は未観測)、(2) matcher の型不正(非文字列)を検出しない。いずれも warn レベルで --strict ゲートの fail-open ではない。詳細: docs/tech-debt/README.md 最終行、triage レポート Cycle 3。

## リリース前提

本 PR マージ後、`/release` で v5.0.0(overlay-scaffold-v2 系列 + 本修正)を発行する。
