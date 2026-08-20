# Cross-review triage report: codex-hooks-json-wiring

- Date: 2026-08-20
- Plan: docs/plans/active/2026-08-20-codex-hooks-json-wiring.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-20-codex-hooks-json-wiring.md
- Self-review report: docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md(C2: MEDIUM 2 + LOW 5 → bced11a/67f56d5 で全修正)
- Verify report: docs/reports/verify-2026-08-20-codex-hooks-json-wiring.md(Cycle 2: pass)
- Cycle-1 の AR 1 件(features.hooks 非 boolean)は d1df46f で修正済み・cycle-2 の verify/test が確認済み。本サイクルの 2 件は reviewer が hook スクリプトを実行して実測した機能面の指摘で、triager もコードで裏取りした。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Codex がサブディレクトリから起動されると hook の cwd もそこになり、command は git root で dispatcher を解決するが `ralph-dispatch.sh` 自身は cwd 相対で `.d` 層を探すため、発火しても 0 スクリプトで無音スキップ | 実問題(dispatcher の `.d` 解決が cwd 相対であることをコードで確認)。Slice 1 の live-fire は root 起動のため未検出。修正は command 文字列への git-root cd 前置(両コピー)+サブディレクトリ起動の live-fire で固定 | .codex/hooks.json + templates twin、tests/test-hook-wiring.sh の期待文字列 |
| 2 | [P2] apply_patch のペイロードはパッチ本文を `tool_input.command` に載せ、共有スクリプト(post_edit_verify.sh / check_mojibake.sh)は `tool_input.file_path` しか抽出しないため、Codex 編集では発火しても実質 no-op(reviewer が実測: rc=0・記録なし) | 実問題。ただし旧配線(直接呼び出し)でも同一ペイロードで同じ no-op だった既存ギャップの顕在化であり、リグレッションではない。修正はパッチ本文から対象パス行を導出する抽出拡張+fixture テスト — 本 PR の「Codex hooks を機能させる」目的の実質部分 | .claude/hooks/post_edit_verify.sh、check_mojibake.sh(+lib_json.sh)、templates twins、tests |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2` の cap に到達(本サイクルが 2/2)。ACTION_REQUIRED 2 件の扱い(cap 引き上げ再実行 / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: features.hooks 非 boolean の無言スキップ — ACTION_REQUIRED=1)は d1df46f で修正され、cycle-2 の verify(1b5cfa2)・test(c288a81)で確認済み。
