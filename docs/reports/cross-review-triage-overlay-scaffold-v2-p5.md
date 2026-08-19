# Cross-review triage report: overlay-scaffold-v2-p5 (Phase 5)

- Date: 2026-08-19
- Plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Self-review report: docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md(C3: MEDIUM 1 + LOW 7 → 6353a07 で処理済み)
- Verify report: docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md(Cycle 3: pass、fail-open クラス枯渇と結論)
- Cycle-2 の AR 4 件は 25b4f79(+fd136ae)で修正済みで、cycle-3 の verify/test が契約一致を確認済み。本サイクルの 2 件は新規クラス: (1) cycle-1 sync-docs が追加した drift 解消案内(b9a956f)の適用範囲漏れ、(2) Slice 4 が config.toml コメントに残したメタリポ履歴の FR-10 リーク。triager が両サイトを直接確認した。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 未追跡 drift(新テンプレート core パスと既存未追跡ファイルの衝突 — manifest エントリなしで plan.Drift に載る)に対しても案内文が `ralph eject`/`ralph adopt` を指すが、両コマンドは未追跡パスを拒否する設計(AC-3)のため、ユーザは exit 3 のまま解決経路がない | 実問題(確認済み: eject.go:73 / adopt.go:211-215 が未追跡を拒否、案内は upgrade_v2.go:229 で無条件)。案内文字列に未追跡ケースの分岐を追加する(ローカルファイルを退避または内容を手動統合してから再実行、の案内)。コマンド側の未追跡対応はスコープ拡大になるため文言側で解決 | internal/cli/upgrade_v2.go:229(および同文言の他 3 サイト)、internal/upgrade/report.go |
| 2 | [P3] `templates/base/.codex/config.toml` のコメントが「Phase 5 Slice 4」等のメタリポ開発履歴を下流に同梱 — FR-10 違反。purity ガードは一般語のため未検出 | 実問題(確認済み: 86-87 行付近)。コメントを一般化(調査結果の技術内容は保持、フェーズ/スライス/セッション参照を除去)し、root コピーと byte-identical 維持。purity ガードに "Slice " 等を足すかは過剰検出リスクがあるため任意 | templates/base/.codex/config.toml + .codex/config.toml |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

cap は cycle-2 後に操作者承認で 2→3 に引き上げ済みで、本サイクル(3/3)で再到達。ACTION_REQUIRED 2 件の扱い(cap 再引き上げ / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-2 record (superseded)

Cycle-2 のトリアージ(AR#1: settings 検査エラーの fail-open、AR#2: block 検査エラーの fail-open、AR#3: status の corrupt-manifest 隠蔽、AR#4: purity ガードのパス走査欠落 — ACTION_REQUIRED=4)は 25b4f79 + fd136ae で修正され、cycle-3 の verify(71236d8)・test(aba2eda)で契約一致・回帰テスト green を確認済み。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: ownership planning エラーの warn 固定、AR#2: settings snapshot の (e) 除外 — ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0)は e01939e で修正され、cycle-2 の verify(c805cf5)・test(5312a55)で契約一致・回帰テスト green を確認済み。
