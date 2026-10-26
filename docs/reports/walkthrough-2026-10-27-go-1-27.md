# Walkthrough: go-1-27(go.mod の Go を 1.27.3 に上げ、gate・merge・regate の signal の取り消しを `signal.NotifyContext` にそろえる)

- Date: 2026-10-27(JST。UTC では 2026-10-26)
- Plan: docs/plans/active/2026-10-27-go-1-27.md(/pr で archive に移す)
- PR: https://github.com/yoshpy-dev/ralph/pull/260
- Branch: build/go-1-27(この walkthrough の前の HEAD 04443933。24 commits、19 files、+1407/-166)
- Base: 5ef14b1a(origin/main、「Merge pull request #259」)
- 読む範囲: Go と go.mod は 10 files で +337/-164。残りの追加 1070 行のうち 1067 行は計画(146)、report 5 本(446)、evidence(467)、insight event(8)で、3 行が tech-debt
- 後工程は cycle 1/2 のまま。`/test` の 1 回目の穴(G1、G2)を受けて S4、S5 を足し、`/self-review` から 2 周した。cross-review の指摘は 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → 版(S1) → signal の置き換え(S2) → tech-debt(S3) → test の穴(S4、S5) → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 依頼「なぜ2つのGoのバージョンがあるのですが、どちらかに統一させたい」(2026-10-26)に応える。go.mod は 1.25.8、手元は 1.27.3 で、CI と release は go.mod の版を入れる。PR #225 の CI はこのずれで落ちた
- `go` の行を 1.27.3 にし(`toolchain` の行はなし)、要らなくなった `gateSignalContext` を消して、3 つの command を `signal.NotifyContext` に替える。`signalCauseContext` は director の wait だけが使うので移す。手元の Go が go.mod とずれると落ちるテストを足す。tech-debt (j)(k)(l)(n) を払い、(m) は持ち越す
- 動きは変えない意図。1.27.3 の `NotifyContext` は、最初の signal で原因 `<signal> signal received` を付けて取り消し、`stop` まで後の signal も受ける(verify が `signal.go:293-325` を読んだ)
- release のノートに「source から build するには Go 1.27.3 以上が要る」と書く(release はユーザーが切る)。戻し方は revert で、`go` の行だけ戻すと Go 1.25 で原因の文が付かないので S2 も戻す

## 2. 読む順(ファイルごと)

commit は S1 97b78594、S2 07221fd9、S3 a6601083、S4 3afc1a81、S5 bceca742。ほかは計画、report、event。行番号は HEAD 04443933 のもの。

1. `go.mod:3` が `go 1.27.3`(97b78594)。`toolchain` の行はなく、`go mod tidy -diff` は空
2. `cmd/ralph/toolchain_test.go`(新規 107 行、97b78594)
   - `checkGoModToolchain`(`:24`)は `toolchain` の行(なければ `go` の行)の版を、`runtime.Version()` の最初の空白までと比べる(`:41-45`)。違えば、両方の版と直し方 2 つを言う(`:48-49`)
   - 表の `TestCheckGoModToolchain`(`:52`、8 行)と、go.mod を読む実物の `TestTheTestsRunOnTheGoThatGoModNames`(`:99`)。scaffold 先には配られない
3. `internal/org/prompts_leader_inbox_test.go:366`: errcheck が指した `<-done` を `err := <-done` にし、Fatalf に出す(`:367-368`)
4. `internal/cli/org_gate.go:102`、`org_merge.go:197`、`org_regate.go:146`(07221fd9)
   - 3 つの `signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)`。引数は前と同じ。gate のコメント(`org_gate.go:94-101`)は原因の文と「2 発目の Ctrl-C で終わらない」を言い、merge(`:194-196`)と regate(`:143-145`)は「later ones are caught until stop runs」とだけ言う。`org_merge.go:29`、`org_regate.go:24` の doc comment も直した
5. `internal/cli/org_wait_director.go`(+59)
   - `signalCauseContext` は `:129-166`、呼ぶのは `:66` だけ。本文は動かしただけで、origin/main の `org_gate.go:147` から始まる 38 行を `awk` で取り出し、HEAD の `:129` から始まる 38 行と `cmp` したら同一だった(この walkthrough を書く前に打った)
   - 変わったのは doc comment(`:111-128`。`gateSignalContext` に寄りかからない文にした)と import(`errors`、`sync`)だけ
6. `internal/cli/org_wait_director_test.go`(+154)
   - 移した 2 本(元は origin/main の `org_gate_test.go:729`、`:763`)。`...ParentCauseAndStop`(`:572`)は `received` が nil の確かめを足し、S4 の段(`:590-609`)は呼ぶ前に終わっていた親で `stop` が 5 秒以内に返ることを見る。`...CancelsOnTheFirstSignalAndKeepsLaterOnesCaught`(`:634`)は (k) の名前
   - 新しい `...StopUnregistersTheSignals`(`:684`)は (l)。子が `stop` のあとに自分へ SIGHUP を送り、SIGHUP で終わることを見る。親は先に SIGHUP を自分の channel に登録する(`:701`)。nohup 起動だと子が無視を引き継いで偽って落ちるため
7. `internal/cli/org_gate_test.go` は、`:721` の `}` と `:723` のコメントのあいだで旧テスト 2 本(77 行)を消した。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(`:655`)は残る。`org_merge_test.go:690` はコメントの名前を替えただけ
8. tech-debt: `docs/tech-debt/README.md:231` は Evidence 欄の `gateSignalContext` を `newOrgGateCmd` に直した。`:235` は (j)(k)(l)(n) を閉じ、(m) に持ち越しの注を足した。`:236` は新しい行(G1、下の 5)。どれも 5 欄のまま
9. 記録: 計画 146 行、report 5 本 446 行(self-review 91、verify 124、test 134、sync-docs 59、triage 38)、evidence 467 行、event 8 行(`docs/insights/events/2026-10-26-go-1-27.jsonl`。どれも cycle 1、pass。10-26 は UTC の日付)

## 3. 設計の判断(計画の D1〜D7)

- D1 `go` の行は 1.27.3、`toolchain` の行は足さない: 1.27.0 では CI と手元の patch がずれる。`go` の行は最低の版なので、手元が上がったときのずれは D7 で止める
- D2 (j) を払う: Trigger「`go` の行が 1.26 以上」を満たす。残すと「go.mod が 1.25.8 を許すので使わない」という理由が嘘になる
- D3 `signalCauseContext` を移す: gate が使わなくなると、使うのは director の wait だけ。中身は変えない(上の 2 の 5)
- D4 (k)(l) を払い、(m) は持ち越す: (k)(l) の Trigger はこの PR が当たる。(m) の Trigger は channel の編集で、この PR は channel を変えない
- D5(superseded)`_ = <-done` ではなく `err := <-done`: 計画の `_ = <-done` は staticcheck の S1005 に、裸の `<-done` は errcheck に当たる
- D6 `internal/org/split_test.go:541-542` の fixture は直さない: 分割計画の本文の例の文字列で、動きに関わらない
- D7 版のずれは repo のテストで止める(Codex の計画助言): 検証 script に `GOTOOLCHAIN` を足す案は、script が scaffold 先にも配られるので採らない。`go env -w` は利用者の go.env を書き換えるので採らない
- 計画からの逸れは 4 つ(計画の Progress): D5 の形、S1 の件名を `ci:` にしたこと(commit-msg hook が `build:` を受けない。`ci:` も changelog に出る)、(l) の親の SIGHUP の登録、S4 と S5

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | `go.mod:3`。`go mod tidy -diff` が空で、go.mod と go.sum の sha256 が前後で同じ(verify 1、2 回目) |
| AC2 | `GOTOOLCHAIN` なしの go1.27.3 で `./scripts/run-static-verify.sh` が rc 0。golangci-lint 2.15.0 は `0 issues.`、staticcheck 2026.2.2 は出力なし(verify 1、2 回目) |
| AC2b | 表 8 行と実物のテストが `-count=3` で通った(test E2)。mutant 2 つは `go test -overlay` で、表の 5 行と 2 行が落ち、対照は通った(計画の Progress、verify 2 回目) |
| AC3 | `git grep -n gateSignalContext -- '*.go'` が空(この walkthrough を書く前に打った)。gate・merge・regate の割り込みのテストが `-count=3` で通った(test E1) |
| AC4 | 本文は `cmp` で同一。テスト 3 本は `:572`、`:634`、`:684`。mutant 4 つは指定のテストで 3/3 落ち、対照は通った(計画の Progress)。S4 の `close(ended)` を消す mutant は 5 秒の区切りの文で落ちた(test 2 回目) |
| AC5 | `*.go` に残るのは `split_test.go:541-542` の fixture と `gate_runverify_test.go:58` の scratch module の `go 1.25` だけ(verify 1 回目) |
| AC6 | 231、235、236 行が 5 欄で、(j)(k)(l)(n) が閉じ、(m) に注がある(verify 2 回目) |
| AC7 | 一部。Go のテストは `./...` 1 本ではなくパッケージごとに打ち、通った(test 1 回目: `internal/org` 1,197 本、`internal/cli` 827 本、shell 44 本)。`./scripts/run-verify.sh` の rc 0(8ac89c46、914 秒)は PR 本文だけにある。PR の CI は結果待ちで、計画の AC7 は `[ ]` のまま |

## 5. 確かめていないこと

- G1(tech-debt `:236`): gate・merge・regate が頼る「最初の signal のあとも `stop` まで登録が残る」ことを、repo のテストは固定していない。今は標準ライブラリの文書に任せている。in-process のテストに見えないのは、`signalSelf`(`org_gate_test.go:633-646`)が signal をテストの channel にも登録するため。test の probe 3 は gate に「最初の signal で登録を外す」mutant を入れ、23 本が生き残った。使い捨ての子プロセスのテストでは 3/3 で差が出た。merge と regate には同じ mutant を入れていない
- R1: CI の `actions/setup-go` が `go 1.27.3` を入れられるかは、PR #260 の CI でしか確かめられない。結果はまだない
- R6: repo は golangci-lint と staticcheck の版を固定しない。確かめたのは golangci-lint 2.15.0 と staticcheck 2026.2.2 だけで、v0.7.0 の staticcheck は 1.27.3 の標準ライブラリを読めず落ちた。古い道具が PATH にある機械では静的検査が落ち、Go が go.mod と違う機械では `toolchain_test.go` が落ちる。ほかの機械では打っていない
- 実機の確かめは要らないと判断した。動きを変える意図がなく、signal の道はテストと mutant だけで見た。本物の `ralph org gate`・`merge`・`regate` に Ctrl-C を送ってはいない
- self-review の LOW 3 件(gate のコメントの括弧の読み、`org_wait_director_test.go` が 866 行、`devel` の Go での失敗の文)は直していない。判定は Merge OK。`-race` は打っていない。Linux は CI が見る
