# Review: doctor-escape (t-2)

- Plan: docs/plans/active/2026-10-20-doctor-escape.md
- HEAD: ce22ea105a43d94cb43c1cc2715684b152a4614d
- BASE: f0668004015323a2366ac7983a50e2d27017dfa2 (merge-base with origin/main)
- Reviewer: reviewer seat (org doctor-escape)

## Gate

| コマンド | 結果 |
|---|---|
| `./scripts/run-static-verify.sh` | pass(gofmt ok、lint 0 issues、secret scan clean。staticcheck は未インストールのため skip。docs/evidence/verify-2026-10-20-044519.log) |
| `go test ./internal/cli/` | ok(650.1s) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/` | ok(654.9s) |

GATE: pass

## 受け入れ条件

1. 台帳のパスの制御文字が Detail にそのまま出ない: 満たす。`internal/cli/doctor_merge_guarantee.go:90` で Detail 全体を `org.PrintableInboxText(..., false)` に通している。
2. それを確かめるテストがある: 満たす。`TestMergeGuaranteeResult_EscapesControlCharacters`(`internal/cli/doctor_merge_guarantee_test.go:336`)が ESC と改行を検査する。
3. 制御文字のない Detail は変わらない: 満たす。`PrintableInboxText`(`internal/org/inbox.go:168`)は制御文字以外をそのまま書く。既存の `TestMergeGuaranteeResult_EverySaveOutcome` も期待値を変えずに通っている。
4. `go test ./internal/cli/` が通る(go1.25.8 でも): 満たす(上の表)。

## 所見

- LOW: 計画のエッジケース「escape 済みの gh/git のエラー文が二重に escape されないか」を確かめるテストがない。コードを読む限り、`PrintableInboxText` はバックスラッシュを escape しないので、escape 済みの `\x1b` はそのまま残り、二重 escape は起きない。ただし、この性質はテストで固定されていない。Evidence: internal/org/inbox.go:168-182、ce22ea10。
- 差分は 1 行の変更とテスト 1 本だけで、命名、エラー処理、スコープに問題はない。

## 判定

Merge: 可(CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 1)
