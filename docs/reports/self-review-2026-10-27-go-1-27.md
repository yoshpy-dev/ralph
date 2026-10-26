# Self-review report: go-1-27

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-go-1-27.md`(D1〜D7、AC1〜AC7、Progress checklist の逸れ)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD`(HEAD 8ac89c46、origin/main 5ef14b1a、12 ファイル、+462/-166、6 コミット)。診断の対象は diff の品質だけで、テスト、静的検査、spec 適合、文書のずれは見ていない(`/verify`・`/test`・`/sync-docs` の範囲)。

## Evidence reviewed

- `git diff origin/main...HEAD` の Go 側(`go.mod`、`cmd/ralph/toolchain_test.go`、`internal/cli/org_gate.go`、`org_merge.go`、`org_regate.go`、`org_wait_director.go` と、それぞれのテスト、`internal/org/prompts_leader_inbox_test.go`)と `docs/tech-debt/README.md` を全文読んだ。計画は Scope、Non-goals、Design decisions、Progress checklist を読んだ。
- 依頼された 4 点
  - コメントの正しさ: Go 1.27.3 の `$(go env GOROOT)/src/os/signal/signal.go:293-325` と照らした。`NotifyContext` は最初の signal で原因 `<signal> signal received` を付けて取り消し、`stop` は `cancel(nil)` と `Stop(ch)` を呼び、登録は `stop` まで残る。gate・merge・regate のコメント(「原因が signal を名指す」「後の signal も `stop` まで捕まえる」)と、移した `signalCauseContext` の doc comment(「`NotifyContext` と同じ原因」「`stop` は取り消し、goroutine を待ち、登録を外す」「`NotifyContext` は届いた signal を返さない」)はこれと合う。`org_gate.go` の旧版と移した先の関数本体は、`awk` で取り出して `diff` が空(中身は変わっていない。D3 のとおり)。`org_gate.go` の `errors`・`os/signal`・`syscall` の import は残り、`sync` だけが消えている。
  - 親側の `signal.Notify(caught, SIGHUP)`(`internal/cli/org_wait_director_test.go:679-680`): 子は親が無視した signal を無視のまま引き継ぎ、親が handler を持つ signal は既定に戻して起動する。Go の `signal.Notify` は無視されて始まった SIGHUP・SIGINT にも handler を入れる。コメントの 2 文はどちらも合う。子の側は `stop` のあと `signal.Stop` で handler が外れ、Go runtime は SIGHUP を `_SigKill` で扱うので、子は SIGHUP で終わる。親の失敗の道(`err == nil`、SIGHUP 以外の終わり方、30 秒の timeout で SIGKILL)はどれも別の `Fatalf` に入り、出力を添える。`defer signal.Stop(caught)` で親の登録は戻る。`defer stop()` と明示の `stop()` の二重呼びは、`cancel(nil)`、閉じた `ended` からの受信、`signal.Stop` がどれも 2 回目に害がないので問題ない。
  - `cmd/ralph/toolchain_test.go` の文と解析: `strings.Fields` なので tab、行末コメント(`go 1.27.3 // x`)、CRLF を読む。`//` で始まる行は 1 語目が `//` なので飛ばす。`toolchain` の行が先、なければ `go` の行を `"go"+` 付きで比べるのは、`setup-go` の `go-version-file` と `.github/workflows/verify.yml:15`・`release.yml:21` の使い方と合う。実験の接尾辞(`X:...`)は最初の空白までで切る。落ちたときの文は、走っている版、読んだ行の名前、go.mod の版、直し方 2 つを言い、表のテストが 3 種類の断片(走っている版、行の名前、`GOTOOLCHAIN=`)を固定する。実物のテストは `../../go.mod` を読み、`go test` は package の dir で動くのでコメントのとおり。
  - tech-debt の 2 行: 231 行と 235 行を `origin/main` と HEAD で比べ、`|` の数はどちらも 6(5 欄)のまま。消し線 `~~` は 235 行で 30(15 組)で、組は対になっている。(j)(k)(l)(n) は消し線のあとに `Closed in build/go-1-27 (commit …)` が続き、(m) は消し線なしで持ち越しの注がある。Impact・Why deferred・Trigger の欄の (j)〜(n) も同じ形に直っていて、読み順は「消し線の元の文、閉じた旨」で一貫する。Evidence 欄には `signalCauseContext` の今の場所、`cmd/ralph/toolchain_test.go`、コミット 97b78594・07221fd9、計画が足されている。閉じた旨の文が名指すテストの名前と場所は、実物と一致する。
- 消した名前の残り: `git grep -n gateSignalContext` の `*.go` は 0 件。`docs/evidence`、`docs/reports`、`docs/plans` の過去の記録と、235 行の閉じた旨の文だけが名前を持つ。`1.25.8` は `internal/org/split_test.go:541-542` の fixture(D6、直さない)だけ。
- 確かめていない: テストと静的検査の実行、mutant の再現、CI の `setup-go` が 1.27.3 を入れること。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | gate のコメントの括弧 `("<signal> signal received", signal.NotifyContext)` が、原因の文と出どころを並べて、「names it (A, B)」のように読める。merge・regate のコメントは `(signal.NotifyContext)` だけで、文の出どころは gate の 1 か所に書かれる | `internal/cli/org_gate.go:95-96` | `names it (signal.NotifyContext's "<signal> signal received")` のように、文が NotifyContext のものだと一続きで言う。直さなくても誤りではない |
| LOW | maintainability | `internal/cli/org_wait_director_test.go` が 712 行から 844 行になり、files < 800 行の目安(`~/.claude/rules/code-review.md`)を超えた。足した 3 つのテストと 2 つの env 定数(564-693 行。3 つのテストは 571・612・662 行)は signalCauseContext だけを扱い、director の wait のテストとは別のまとまり。移す前の `org_gate_test.go` も 836 行だったので前例はある | `internal/cli/org_wait_director_test.go`(`wc -l` 844)、`:571`、`:612`、`:662` | 次にこのファイルを編集するとき、signal の 3 つを `org_wait_director_signal_test.go` に分ける。この PR では必須でない |
| LOW | readability | 落ちたときの文の「raise that line to the running version in a PR」は、走っている Go が `devel go1.28-...` のときは読みどおりにできない(`devel` の版には上げられない)。表のテストはその文面をそのまま固定している。2 つ目の直し方(`GOTOOLCHAIN=go<go.mod の版>`)で足りるので、迷う人は少ない | `cmd/ralph/toolchain_test.go:48-49`、`:72-73` | `devel` のときだけ 2 つ目の直し方を先に言うか、「a release version」と言う。手元の Go が release 版なら当たらない |

## Positive notes

- 置き換えた 3 か所(gate・merge・regate)は、引数、`defer stop()`、`parent` の扱いが同じ形で、`org_regate.go` の doc comment の折り返し直し(3 行)以外に余計な変更がない。
- errcheck の直し(`prompts_leader_inbox_test.go:366`)は、`_ = <-done` が staticcheck S1005 に当たるという逸れを計画に記録し、受けた error を失敗の文に出す形にした。`err` は case の中の宣言で、外の `err` を隠さない。
- 移した 2 つのテストは、旧版の「`ParentCauseAndStop`」に `received` が nil であることの確かめを足し、「`KeepsLaterSignalsCaught`」は名前を前半・後半の両方を言うものにした。(l) のテストは親側の `signal.Notify` で nohup 起動の偽の失敗を防ぎ、その理由をコメントが言う。
- `toolchain_test.go` の比べる関数は文字列だけを受け、本番コードに足したものがない。`go` 行だけ、`toolchain` 行が勝つ、`toolchain` 行が一致して `go` 行が古い、どちらもない、実験の接尾辞、`devel` の 8 通りを表で固める。
- 秘密、デバッグ出力、残った TODO、コメントアウトされたコードはない。入力は go.mod と自分のテストの定数だけで、外から来る値を扱う新しい道はない。

## Known gaps (diff 品質の範囲の外)

- gate・merge・regate の「後の signal は `stop` まで捕まえる」は `signal.NotifyContext` の文書化された保証(上の `signal.go` の doc)に頼っていて、この repo のテストは gate の道では固定しない(旧 `TestOrgGate_SignalContextKeepsLaterSignalsCaught` は `signalCauseContext` に移した)。テストの網羅は `/test` で見る。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(上の LOW 3 件は任意の直しで、繰り越しの記録には当たらない) | - | - | - | - |

_(行を足していないので `docs/tech-debt/` への追記はない。)_

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3)
- Follow-ups: 任意。LOW の 1 件目と 3 件目は文言の直し、2 件目は次にそのテストファイルを編集するときに分ける。直しはどれも `.go` の編集なので、直す場合は `/self-review` から回し直す。

## Run 2

- Date: 2026-10-27
- Reviewer: reviewer サブエージェント(`/self-review`)。post-implementation pipeline の 2 回目で、cycle は 1 のまま。cross-review ではなく、`/test` が出したテストの穴(G1・G2)を塞ぐ追加(S4・S5)で回し直した。
- Scope: run 1 以降の `git diff f6725ebf..HEAD -- internal/ docs/tech-debt/` を精読し、`git diff origin/main...HEAD`(HEAD 7ddf3c36、17 ファイル、+994/-166)の全体を文脈として読み直した。run 1 以降の Go の変更は 3afc1a81 の `internal/cli/org_wait_director_test.go` だけで、`cmd/` と `internal/cli/org_gate.go`・`org_merge.go`・`org_regate.go` には触れていない。診断の対象は diff の品質だけで、テスト、静的検査、spec 適合、文書のずれは見ていない。

### Evidence reviewed (run 2)

- 3afc1a81(`internal/cli/org_wait_director_test.go:590-609` の新しい段)を `signalCauseContext`(`internal/cli/org_wait_director.go:129-166`)と照らした。親が呼ぶ前に終わっていると `ctx.Err() != nil` の枝(152-154 行)で goroutine を起こさず `ended` を閉じ、`stop` は `cancel(nil)`、閉じた `ended` からの受信、`signal.Stop` の順で返る。段の期待(`ctx.Err()` が `context.Canceled`、`context.Cause` が親の原因、`stop` が 5 秒以内に返る、`received()` が nil)はこの枝の挙動と合う。`defer stop()`(577 行)は最初の `stop` に束縛済みで、593 行の再代入で変わらない。`cancelEnded` は呼んでいて、`go vet` の lostcancel には当たらない形。`stop` が返らない場合は goroutine を 1 つ残して `t.Fatal` が文を出して終わる。
- 564-571 行の doc comment は「後で終わる親」「先に終わっていた親」「`stop`」の 3 つを言い、`received` が nil になるのも 3 つで、段の数と合う。
- bceca742(`docs/tech-debt/README.md:236`)は `|` が 6 個(5 欄)で 235 行と同じ形。名指しの `newOrgGateCmd`・`newOrgMergeCmd`・`newOrgRegateCmd` は `org_gate.go:24`・`org_merge.go:35`・`org_regate.go:28` にある。「gate の呼び出しのコメントは 2 発目の Ctrl-C を言い、merge と regate のコメントは `stop` まで後の signal を捕まえるとだけ言う」は `org_gate.go:99-101` と、`org_merge.go:194-196`・`org_regate.go:143-145` のとおり。「the SIGKILL row above」は 231 行。Evidence 欄の `docs/reports/test-2026-10-27-go-1-27.md`(probe 3 と G1)と `docs/evidence/test-2026-10-27-go-1-27.txt`(254 行の Probe 3: 23 本が通る行、使い捨てのテストが 3 回通る行)は実在し、本文の「23 tests」と合う。
- 計画の Progress 行(S4 は 3afc1a81、S5 は bceca742、「235 行の次に 1 行」)は実物と一致する。
- 確かめていない: テストと静的検査の実行、`close(ended)` を消す mutant の再現、probe 3 の merge・regate 側(行自身が「未確認」と書く)。

### Findings (run 2)

新しい指摘はない。run 1 の LOW 3 件は次のとおり。

| Run 1 の LOW | 状態 | 根拠 |
| --- | --- | --- |
| 1: gate のコメントの括弧の読み(`org_gate.go:95-96`) | 未修正、変化なし | 新しいコミットは `org_gate.go` に触れていない。95-96 行は run 1 のまま |
| 2: `org_wait_director_test.go` が 800 行を超える | 未修正、超過が広がった | 844 行から 866 行になった(`wc -l`)。S4 の段(590-609 行)が `TestOrgWaitDirector_SignalCauseContextParentCauseAndStop`(572 行)に入ったため。signal の 3 つのテストは 572・634・684 行、env 定数は 624・678 行で、まとまりは変わらない。重さは LOW のまま(分ける案は run 1 のとおり) |
| 3: 落ちたときの文が `devel` の Go で読みどおりにできない(`cmd/ralph/toolchain_test.go:48-49`、`:72-73`) | 未修正、変化なし | `cmd/` は run 1 以降の diff にない |

### Known gaps (run 2)

- run 1 の Known gap(gate・merge・regate の「後の signal は `stop` まで捕まえる」を repo のテストが固定していない)は、bceca742 で `docs/tech-debt/README.md:236` に行として記録された。上の Evidence のとおり、行の記述に実物との食い違いは見つからなかった。テストを足すかどうかは `/test` とトリアージの判断。

### Tech debt identified (run 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 今回の run で足す行はない。run 1 の「なし」は、`docs/tech-debt/README.md:236`(bceca742、`/test` の G1 を実装側が記録)で置き換わっている | - | - | - | - |

_(この run では `docs/tech-debt/` に追記していない。)_

### Recommendation (run 2, current)

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。run 2 の新しい指摘は 0 で、LOW 3 は run 1 の 3 件が未修正のまま残っている)
- Follow-ups: 任意。内容は run 1 の Follow-ups のまま。LOW の 2 件目は、次にこのファイルへ手を入れる人が signal の 3 つのテストを `org_wait_director_signal_test.go` に分ける形が収まりがよい。
