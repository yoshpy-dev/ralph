# 9 段目の試行: 小さな escape の直し 2 件

- Status: Approved
- Approved: 2026-10-20 sha256:0ffd32661606

## 目的

機能ごとの org を 2 つ同時に走らせる試行(`docs/specs/2026-10-07-org-multi-org-director.md` の FR-12)。
どちらの機能も tech-debt の表の小さな項目を 1 つ直し、PR を 1 本出す。2 つの機能が触るファイルは重ならない。

座席の決まり(両方の機能に共通):

- implementer は `--driver claude --model opus`、reviewer は `--driver claude --model sonnet` で立てる
- `docs/tech-debt/README.md` は編集しない。項目を閉じる記録は、試行の記録の PR がまとめて書く
- 予約の外のファイルは編集しない

## Features

### report-escape

- Type: fix
- Reserve: internal/org/report.go, internal/org/report_test.go
- Depends on: none

`ralph org report` のイベントの表(`internal/org/report.go` の `| ts | seat_id | event | details |` の行を書くところ)は、各イベントの Details をそのまま Markdown の表の行に書く。そのため Details に `|` があると、その行の欄の数が崩れる(tech-debt の表の「Small leftovers of org-plan-approval」の行の (a))。Details の `|` を `\|` に escape して、行が 4 つの欄を保つようにする。

- [ ] Details に `|` を含むイベントの行が、escape されて 4 つの欄のまま書かれる
- [ ] その行を確かめるテストが `internal/org/report_test.go` にある
- [ ] Details に `|` を含まない行の出力は変わらない
- [ ] `go test ./internal/org/` が通る(`GOTOOLCHAIN=go1.25.8` でも)

### doctor-escape

- Type: fix
- Reserve: internal/cli/doctor_merge_guarantee.go, internal/cli/doctor_merge_guarantee_test.go
- Depends on: none

`ralph doctor` の `Merge guarantee` の検査は、Detail を組み立てた最後(`internal/cli/doctor_merge_guarantee.go` の `mergeGuaranteeResult`)で、gh と git のエラーの文のほかは escape していない。キャッシュの書き込みと削除のエラー、台帳のディレクトリのパス、origin の host と owner/repo が、そのまま Detail に入る(tech-debt の表の「Small leftovers of the org-merge-guarantee PR」の行の (h))。`mergeGuaranteeResult` の最後で Detail 全体を `org.PrintableInboxText` に 1 回通す。

- [ ] 台帳のパスに制御文字を含むとき、その文字がそのまま Detail に出ない
- [ ] その場合を確かめるテストが `internal/cli/doctor_merge_guarantee_test.go` にある
- [ ] 制御文字のない Detail の出力は変わらない
- [ ] `go test ./internal/cli/` が通る(`GOTOOLCHAIN=go1.25.8` でも)
