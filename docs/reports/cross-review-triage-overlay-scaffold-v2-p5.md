# Cross-review triage report: overlay-scaffold-v2-p5 (Phase 5)

- Date: 2026-08-19
- Plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 4
- After triage: ACTION_REQUIRED=4, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Self-review report: docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md(C1: 13 件 → 7cf4f47 で全修正。C2: 7 件 → 5 修正 + 2 deferral)
- Verify report: docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md(Cycle 2: pass)
- Cycle-1 の AR 2 件(planning エラーの strict 化 / snapshot 存在検査)は e01939e で修正済みで、cycle-2 の verify/test が契約一致を確認済み。本サイクルの 4 件は同じ「fail-open」クラスの新規残余 3 経路+purity ガードのパス走査欠落で、triager が全サイトのコードを直接読んで確認した(#4 は reviewer が fixture で素通りを実測)。3 サイクル連続で同クラスが出ていることから、修正時は warn 返却経路の全数監査を含めるべき。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `.claude/settings.json` が不正 JSON だと `MergeOwnedSettings` がエラーを返すが warn 固定で、`--strict` でも exit 0 — FR-9(c) が検証不能なのにゲートが通る | 実問題(コード確認済み: doctor.go:889-892 の warn 固定)。AR#1(cycle 1)と同クラスの meta-failure。修正: strict-eligible 化+回帰テスト(不正 JSON fixture) | internal/cli/doctor.go:889-892 |
| 2 | [P2] block 面(AGENTS.md / .gitignore)が存在するが読めない場合、`applyBlockUpdatesV2` のエラーが warn 固定 — FR-9(b) の検証不能が素通し | 実問題(コード確認済み: doctor.go:810-813)。同クラス。修正: strict-eligible 化+回帰テスト(読めない block 面 fixture) | internal/cli/doctor.go:810-813 |
| 3 | [P2] `ralph status` で manifest が存在するが corrupt/読めない場合に `nil, nil`(=非 ralph プロジェクトと同じ扱い)を返し、scaffold セクション自体が消える — 壊れたプロジェクトが隠れる | 実問題(コード確認済み: status.go:487-489)。M7 の degrade 設計は「エラーを表示して続行」であり「無かったことにする」ではない。修正: fs.ErrNotExist のみ nil、それ以外は scaffoldStatus.Err で unavailable 表示 | internal/cli/status.go:487-489 |
| 4 | [P2] purity ガードは内容 grep のみで、パスベースのリーク(例: `templates/base/.claude/skills/release/` に本文が無害な SKILL.md)を検出できない — maintainer 専用 skill の同梱を防げない | 実問題(reviewer が fixture で rc=0 を実測)。AC-11 の意図(メタリポ固有要素の混入検出)にパス次元が欠けている。修正: パスパターン走査(find/glob ベース)を追加し、`skills/release` 等のパス禁止則+fixture テスト | scripts/check-template-purity.sh, tests/test-template-purity.sh |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2` の cap に到達(本サイクルが 2/2)。ACTION_REQUIRED 4 件の扱い(cap 引き上げ再実行 / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: ownership planning エラーの warn 固定、AR#2: settings snapshot の (e) 除外 — ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0)は e01939e で修正され、cycle-2 の verify(c805cf5)・test(5312a55)で契約一致・回帰テスト green を確認済み。
