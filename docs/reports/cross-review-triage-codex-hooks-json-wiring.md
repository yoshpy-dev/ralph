# Cross-review triage report: codex-hooks-json-wiring

- Date: 2026-08-20
- Plan: docs/plans/active/2026-08-20-codex-hooks-json-wiring.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-20-codex-hooks-json-wiring.md
- Self-review report: docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md(C3: HIGH 1 + MEDIUM 4 + LOW 4 → 3dd3a64/4e4e2a6 で全修正)
- Verify report: docs/reports/verify-2026-08-20-codex-hooks-json-wiring.md(Cycle 3: pass)
- Cycle-2 の AR 2 件は 4d8220c(+C3-H1 の干渉修正 3dd3a64)で修正済み・cycle-3 の verify/test 確認済み。本サイクルの 2 件は doctor の warn 精度(誤検知/見逃し)の指摘で、ゲートの fail-open ではない。triager がコードと公式スキーマ仕様の両面で評価した。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] config.toml の併存警告が `raw["hooks"]` の存在だけで発火するため、Codex 自身の `[hooks.state...]`(trust メタデータ)がプロジェクト config に書かれた場合にも「削除せよ」と誤案内する | Real=debatable: hooks.state は実測上ユーザレベル `~/.codex/config.toml` に記録され(2026-08-19/20 検証)、プロジェクト config での発生は推測的。発生しても warn のみで実害は誤案内に留まる。修正は hooks 直下の子キーから `state` を除外する数行 | internal/cli/doctor.go:322-323 |
| 2 | [P2] hooks.json スキーマ検証が緩く、未知トップレベルキーや matcher の型不正(非文字列)を pass にする | Real=部分的: matcher 型検査の欠落は正当な指摘(cheap)。一方「未知キーの拒否」は公式スキーマが `description` 等のトップレベルキーを許容し、ハンドラフィールド(timeout/async 等)の将来拡張もあるため、実装すると upstream 進化に対し brittle な false-positive 源になる。warn レベルの環境チェックという位置づけでは現行の「既知の壊れ形 4 種+ルーティング検査」が釣り合いと判断 | internal/cli/doctor.go:369-374 |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

cap は cycle-2 後に操作者承認で 2→3 に引き上げ済みで、本サイクル(3/3)で再到達。WORTH_CONSIDERING 2 件の扱い(cap 再引き上げ / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-2 record (superseded)

Cycle-2 のトリアージ(AR#1: サブディレクトリ起動での dispatcher 空回り、AR#2: apply_patch ペイロード不適合 — ACTION_REQUIRED=2)は 4d8220c で修正され(self-review C3-H1 の干渉も 3dd3a64 で解消)、cycle-3 の verify(2a6b15a)・test(6a6cead)で確認済み。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: features.hooks 非 boolean の無言スキップ — ACTION_REQUIRED=1)は d1df46f で修正され、cycle-2 の verify(1b5cfa2)・test(c288a81)で確認済み。
