# Self-review report: gate-lock-holder-log

- Date: 2026-11-21
- Plan: `docs/plans/active/2026-11-21-gate-lock-holder-log.md`(issue #357)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `4893e314`、base `origin/main` 09d27450)。コードの変更は `a29daaaf`(`internal/org/gate_lock.go` と `internal/org/gate_lock_test.go`)、ほかに計画ファイルと進捗記録のコミット。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

- `git diff origin/main...HEAD` の全部(`gate_lock.go` と `gate_lock_test.go` の diff、計画ファイル)
- `internal/org/gate_lock.go` 全体(1〜244 行)と `internal/org/gate_lock_test.go` の 1〜500 行、呼び出し側の `waitHeldFileLock`(`internal/org/lockfile.go:243-268`)
- `gateLockHolder`、`gateLockPolled`、`gateLockNotWritten`、`not written yet` の参照の洗い出し(`internal/` 全体の `*.go`、`*.md`、`*.sh`)。非テストの参照は `gate_lock.go` の中だけで、`gateLockHolder` の呼び出し元は `acquireGateLock` の callback の 1 か所
- 兄弟の差し込み口の書き方: `spawn.go:3086` の `afterStaleCompensation`、`watch.go:901` の `afterStopRequestTaken`、`watch_recurrence.go:150` の `beforeAppendRecurred`(どれも `var x = func() {}`)、`cleanup.go:94` と `director.go:426,430` の構造体の field(nil なら呼ばない、`cleanup.go:683`)
- `t.Parallel` を使うのは `prompts_*_test.go` と `split_test.go` だけで、gate.lock を触らない
- 狭い実行 1 回: `go test -race -count=5 ./internal/org/ -run 'TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty|TestAcquireGateLock_SaysSoWhenTheHolderChanges|TestAcquireGateLock_SaysAMarkerIsNotWrittenYet|TestGateLockHolder'` が ok

確かめたこと(keep 規則の全順):

| 直前の `named` | 読んだ marker | callback の結果 | 取れる行・見出し |
| --- | --- | --- | --- |
| false(読めていない) | 空 | `notWritten` に置く(`named` は false のまま) | `not written yet`(AC2) |
| true | 空 | 置き換えない | 前の名前のまま(AC1) |
| true | 読めない | 置き換える、`named` は false | `cannot be read`(AC3) |
| false(読めない) | 空 | `notWritten` に置く | `not written yet`。計画の決定どおり |
| true | 別の名前 | 置き換える、`named` は true | 替わった行を 1 本(AC4) |
| false | 名前 | 置き換える、`named` は true | 名前の行 |

`seen != gateLockNotWritten || !named` は `!(seen == gateLockNotWritten && named)` と同じで、表のとおり。`gateLockNotWritten` と同じ文を名前つきの marker が作ることはない(名前つきの文は `, started ... by pid N` が必ず付く)ので、文字列の比較で空を取り違えない。callback は `waitHeldFileLock` を呼んだ goroutine の上で 1 回ずつ順に呼ばれる(`lockfile.go:258-260`)ので、`w`、`named`、`said`、`saidAt` に競合はない。

差し込み口 `gateLockPolled` の安全性:

- 本番は nil で、呼び出しは nil を確かめてから(`gate_lock.go:123`)。
- テストの代入(`watchGateLockPolls`)は `startAcquireGateLock` の `go` 文より前、cleanup の復元は `<-done` を受けたあと(`waitGateLockOutcome`)なので、通る道では happens-before が成り立つ。`-race -count=5` で報告なし。
- 落ちる道(`waitFor` の `t.Fatalf` など `waitGateLockOutcome` の前で止まる)では、cleanup が LIFO で `release` → `gateLockPolled` の復元の順に動くが、待つ側の goroutine を待たない。そのとき復元と待つ側の読みが競合しうる。既存の `shortenGateLockTimes` が `gateLockPoll` に対して持つ形と同じで、すでに失敗している実行の中だけなので、指摘にしない。

新しいテストの順の頑健さ: 各 step は「前の marker を待つ側が読んだ」あとにだけ書く。`writeGateFile` は `os.WriteFile`(truncate してから書く)なので、書き込みの途中に空を読まれうるが、6 つの形のどれでも、空を途中で読まれた場合の結果は変わらない(途中の空は、前の持ち主が名前つきなら残され、空なら同じ文のままで、読めない marker の上に非空を書く形がない)。step ごとの `read` はひとつ前と必ず違うので、古い poll を新しい読みと取り違えない。形を足すときの落とし穴にはなる(下の LOW 3 で触れる)。

既存のテストの期待は今も意味どおり: `TestAcquireGateLock_SaysAMarkerIsNotWrittenYet`(名前のない持ち主のまま待つ形で、文は変わらない)、`TestAcquireGateLock_SaysSoWhenTheHolderChanges`(名前 → 別の名前)、`TestAcquireGateLock_WaitsForTheHolderAndSaysWhoItIs`、`TestGateLockHolder`(表に `named` の期待を足した。`strings.HasPrefix(tc.want, "a gate whose marker ")` で導く)、`TestGate_TwoGatesRunTheirScriptsOneAtATime`(`gate_lock_gate_test.go:218` の期待は同じで、直しで落ちにくくなる側)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | naming | 差し込み口 `gateLockPolled` の名前が隣の `gateLockPoll`(poll の間隔の `time.Duration`)と 2 文字違いで、`grep gateLockPoll` が両方に当たる。過去形の名前は「poll した」という flag にも読める。同じ package の差し込み口は `before*` / `after*` の名前(`afterStaleCompensation`、`afterStopRequestTaken`、`beforeAppendRecurred`、`cleanup.go` の `beforeScript`)で、「どの動きの前後か」が名前に出ている | `internal/org/gate_lock.go:35-46`(`gateLockPoll` は 38 行、`gateLockPolled` は 45 行)、`spawn.go:3086`、`watch.go:901`、`watch_recurrence.go:150`、`cleanup.go:94` | `afterGateLockRead` のように、読んだあとに呼ぶと名前で分かる形にする。nil を確かめる書き方は `cleanup.go:683` と同じで、そのままでよい |
| LOW | readability | 持ち主を残す条件 `seen != gateLockNotWritten \|\| !named` は「空で、かつ名前つきの持ち主があるとき残す」の否定で、読み手が反転して読む。`gateLockHolder` の `named` は空と読めないの両方を false にするので、callback は「空」を文の比較で取り直している(`named` があるのに `gateLockNotWritten` との文字列比較も要る)。条件の行に注釈がなく、規則の説明は `acquireGateLock` の doc の 1 段落(20 行のうち約 10 行)と、`gateLockWait` と `gateLockHolder` の doc にも重なって散っている | `internal/org/gate_lock.go:94-113`(doc)、`:122-128`(callback)、`:71-74`、`:198-207` | 条件の行に「空の marker は持ち主が替わったことではない(release と書き直しの隙間)。名前を持っているときは残す」と 1 行の注釈を置き、規則の細かい説明はそちらに移して、`acquireGateLock` の doc は 1〜2 文にする。`gateLockHolder` が `empty` か種類を返す形にすれば文字列の比較も要らなくなるが、今の大きさでは注釈だけで足りる |
| LOW | readability | 新しいテストの本体に、待ちの行から持ち主を取り出す 8 行がある(`CutPrefix` と `Cut` の 2 段)。待ちの行の形は `gateLockWaitLine`(`gate_lock_test.go:130-133`)がすでに持っていて、文言が変わると 2 か所を直すことになる。あわせて `waitFor` の doc「one of the tries after the first from」は、`from` が添字(`p.seen[from:]`)なのか個数なのか、読んだだけでは決まらない | `internal/org/gate_lock_test.go:347-349`(doc)、`:432-439`(取り出し) | 取り出しを `waitingLineHolders(stderr, path) []string` に出し、`gateLockWaitLine` の隣に置く。doc は「the tries from index `from` on」の形にする。step の `read` がひとつ前と同じになる形を足すと `waitFor` が古い読みを拾うので、その旨を `step` の doc に 1 行足すとよい |

## Positive notes

- 直しは持ち主の決め方の 1 点に絞られ、`release` の順(空にしてから外す)、`waitHeldFileLock`、poll の間隔、`setOrgID` は触っていない。計画の Non-goals どおり
- `gateLockHolder` の戻り値の `named` は「名前を読めた」だけを意味し、doc が空と読めないの両方を除くと言い切っている。呼び出し元は 1 か所のみで、戻り値を増やした影響は `TestGateLockHolder` と `TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled` の 2 テストに閉じている
- `gateLockNotWritten` を定数にしたが、テストは文をリテラルで持ち直している(`gate_lock_test.go:380`、`:476`)ので、文言を変えるとテストが落ちる。定数が自分自身と比べるだけの空の検査にならない
- 新しいテストは時間でなく poll の読みで同期し(`waitFor`)、時間は 10 秒の失敗の区切りにだけ使っている。6 つの形が表で並び、AC1〜AC4 の順の違い(名前→空、空だけ、名前→読めない、読めない→空、名前→空→別の名前、空→名前)を 1 つずつ取れている
- 差し込み口は nil を確かめてから呼び、呼ぶ位置は規則の前(読んだ文をそのまま渡す)なので、規則が差し込み口の戻りに左右されない
- `acquireGateLock` の doc が「空のまま lock を取った gate は、marker を書くまで前の名前のまま」という既知の遅れを書いている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_なし。上の LOW 3 件は実装側がこの run で直せる大きさで、先送りの行は足していない。直さずに最終 cycle を過ぎる場合は、`docs/tech-debt/README.md` に LOW 2(条件の注釈)と LOW 1(名前)の行を足す。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3)
- Follow-ups:
  - 直すなら安い順に LOW 2(条件の行の注釈と doc の縮め)、LOW 1(`gateLockPolled` の名前)、LOW 3(テストの取り出しの helper 化と `waitFor` の doc)
  - 直した場合も記録だけの修正ではないので(コードに触れる)、`/verify` 以降の全体の再実行が要る
  - 計画の Verify plan にある文書の食い違い(AGENTS.md と gate.lock の仕様が「最後の poll で読んだ持ち主」と書いていないか)は `/verify` と `/sync-docs` の担当で、この review では見ていない
