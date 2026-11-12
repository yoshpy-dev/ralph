# Walkthrough: send-screen-fallback(#308)

- Date: 2026-11-13
- Plan: docs/plans/archive/2026-11-12-send-screen-fallback.md
- Branch: fix/send-screen-fallback
- 差分(origin/main 0936aaea から): 78 ファイル、約 +7,000 / -200 行。このうち testdata の fixture(実機の画面 37 本と合成 2 本)、テスト、報告書が大半を占める。本体のコードは internal/org の 5 ファイルと internal/cli/org.go

## 読む順番

1. 計画の Scope、Assumptions、D1〜D9。S1 のあとで前提を 1 つ直した(手番の途中に打つと、その手番の作業が変わる)
2. `docs/evidence/org-send-screen-probe-2026-11-12.md`。実機の画面と herdr の状態の記録。とくに 2〜5 節(動作中と入力待ちの画面)、7〜9 節(herdr の状態の決め方、#308 の症状は再現しなかったこと)、10 節(複数行の文を入力欄に打ったときの見え方、claude の頭の欠け)
3. `internal/org/send_screen.go`。画面の判定
4. `internal/org/send_fallback.go`。send の待ちと画面の確かめ
5. `internal/org/verbs.go` の `Send` の doc コメントと本体の差分
6. `internal/org/status_live.go` と `internal/cli/org.go` の `status --live`
7. `internal/org/resume_screens.go` の差分(codex 0.166 の footer と、入力欄の見つけ方の分割)

## 各ファイルの要点

### internal/org/send_screen.go(新規)

- `ClassifySendScreen(driver, screen)` は、画面が入力待ちか(idle)、動作中か(busy)、選択の画面か(choice)、判断できないか(unknown)を返す。idle になるのは、入力欄が空(placeholder は空とみなす)、動作中の行が無い、選択の画面でない、手番が終わった肯定の証拠がある、のすべてがそろうときだけ。肯定の証拠は、手番の終わりの行(claude の `✻ Cooked for 3s · done …`、codex の `Worked for …`)が最後のプロンプトの塊のあとにあること、または起動直後の画面(バナーがあり、送った文の塊が無い)であること
- `sendInputHolds(driver, screen, typed)` は Enter の前の確かめ。文の頭が入力欄に見える、入力欄の文が打った文の終わりと一致する(codex は `↑` の印があるときだけ)、claude の貼り付けの placeholder と文の一部だけでできている、のどれか。印の行が番号付きの手順で始まる入力欄は、その文が打った文の一部で、入力欄の形(claude の罫線、codex の footer)があるときだけ受け入れる(`sendTypedInputBox`。cross-review cycle 1 の直し)
- `SendScreenSubmitted(driver, screen, typed)` は Enter のあとの確かめ。入力欄が空になったか動作中の行が出たときだけ true

### internal/org/send_fallback.go(新規)

- herdr の 1 回目の待ちは `--timeout-ms` から確かめの分(`min(10s, timeout/3)`)を除いた時間。確かめの分が「読みの間隔 2 つ + Enter の前の待ち」に届かなければ、今までどおり herdr を全時間待つ(既定値で `--timeout-ms` 20.25 秒未満)
- 1 回目の待ちが時間切れのときだけ、herdr の状態を聞き、画面を 3 秒あけて 2 回読む。2 回とも idle で同じなら画面の道。そうでなければ残りの時間で herdr の待ちに戻る(AC11)
- 確かめ全体は確かめの分の半分で区切り、herdr の呼び出しは 1 回ずつ区切る(self-review L1 の直し)
- 画面の道では、ロックの下で herdr を待つ代わりに画面を読み直し、Enter の前に `sendInputHolds`、Enter のあとに `SendScreenSubmitted` で確かめる
- エラーの文は、今の文の頭と `sendWaitDeadlineHint` を残し、herdr の状態、画面の判定と理由と決めた行、次の手(`ralph org read`、`ralph org status --live`、`herdr agent explain`)を足す

### internal/org/verbs.go

- `Send` の待ちを `sendWaitForSeat` に、ロックの下の待ちを `sendWaitUnderLock` に切り出した。herdr の道のエラーの文と `sent` の Details は今と同じ
- `typeAndSubmit` に確かめの関数を渡す引数を足した(nil で今の `confirmSubmitted`。resume は nil を渡す)
- `SendParams.NoScreenFallback` と `HintStateDir`

### internal/org/status_live.go(新規)と internal/cli/org.go

- `ralph org status --live` は、動いている座席ごとに herdr の状態と画面の判定を出し、herdr が working で画面が idle の座席に `mismatch:` の行を付ける。herdr のサーバーは起動しない。`--live` なしの表と JSON はバイト単位で今と同じ
- `ralph org send --no-screen-fallback`、help の文

### internal/org/resume_screens.go

- `resumeCodexFooterLine` が codex 0.166.0 の footer(`… · Context 7% used · main · …`)に合うようにした。これで resume も codex 0.166 の入力欄を見つけられる
- `findResumeInputBox` を 3 つの段(印の行を探す、番号付きの行の判定、入力欄の形)に分けた。resume の判定の順と返り値は変わらない(全 fixture で overlay を使って確かめた。変異 R1 で固定)

## 確かめていないこと

- 実機の herdr が working と言い続ける座席への送信(#308 の症状は 2026-11-12 の確かめで再現しなかった)。テストは偽の herdr と実機の画面の fixture で行った
- 既定の `--timeout-ms` 30000 での動き
- 番号付きの手順の入力欄の fixture 2 本は合成
- 送る文に入力欄の印(`❯`/`›`)で始まる行がある場合(cross-review cycle 2 の指摘、#322)
- claude に 1 KB 前後の文を打つと頭が消える件(#319)
