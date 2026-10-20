# doctor Merge guarantee Detail escape

- Status: Draft
- Approved: TBD
- Owner: Claude Code (org leader doctor-escape)
- Date: 2026-10-20
- Related request: split plan trial, feature doctor-escape
- Related issue: tech-debt "Small leftovers of the org-merge-guarantee PR" (h)
- Type: fix
- Branch: fix/doctor-escape

## Objective

`ralph doctor` の `Merge guarantee` の Detail には、キャッシュの書き込みと削除のエラー、台帳のディレクトリのパス、origin の host と owner/repo が escape されずに入る。`mergeGuaranteeResult` の最後で Detail 全体を `org.PrintableInboxText(..., false)` に 1 回通し、制御文字が端末にそのまま出ないようにする。

## Scope

- `internal/cli/doctor_merge_guarantee.go` の `mergeGuaranteeResult`
- `internal/cli/doctor_merge_guarantee_test.go` へのテストの追加

## Non-goals

- `org.PrintableInboxText` 自体の変更
- doctor のほかの検査の escape

## Assumptions

- `org.PrintableInboxText` は制御文字のない文字列をそのまま返す

## Affected areas

- internal/cli/doctor_merge_guarantee.go
- internal/cli/doctor_merge_guarantee_test.go

## Visual review

.harness/state/plan-visual/doctor-escape.html

## Design decisions

Critical forks: None

## Acceptance criteria

- [ ] 台帳のパスに制御文字を含むとき、その文字がそのまま Detail に出ない
- [ ] その場合を確かめるテストが `internal/cli/doctor_merge_guarantee_test.go` にある
- [ ] 制御文字のない Detail の出力は変わらない
- [ ] `go test ./internal/cli/` が通る(`GOTOOLCHAIN=go1.25.8` でも)

## Implementation outline

1. `mergeGuaranteeResult` の最後を `r.Detail = org.PrintableInboxText(c.Kind + ": " + c.Detail + "; " + saved, false)` にする(改行とタブも escape する。Detail は 1 行)
2. NoLedger の Path に ESC と改行を入れたテストを足し、既存のテストの期待値が変わらないことを確かめる

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: 受け入れ条件の 4 項目
- Documentation drift to check: なし
- Evidence to capture: reviewer のレポート

## Test plan

- Unit tests: `go test ./internal/cli/ -run MergeGuarantee`
- Regression tests: `go test ./internal/cli/` と `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/`
- Edge cases: ESC、改行、すでに escape 済みの gh/git のエラー文が二重に escape されないか
- Evidence to capture: テストの出力

## Risks and mitigations

- escape 済みの gh/git のエラー文が二重に escape される: テストで確かめ、起きるなら escape 前の値を使う

## Rollout or rollback notes

1 コミットの revert で戻せる。

## Open questions

なし

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [ ] Implementation started
- [ ] Review artifact created
- [ ] PR created
