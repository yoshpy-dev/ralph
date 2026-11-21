# Walkthrough: gate-lock-holder-log

- Date: 2026-11-21
- Plan: docs/plans/active/2026-11-21-gate-lock-holder-log.md
- Issue: #357
- Base: main 09d27450

差分は報告を含めて 10 ファイル、コードとテストは 2 ファイル(約 220 行)。下の順に読むと早い。

## 1. 何が起きていたか

同じ台帳の gate は gate.lock で 1 本ずつ走る。待つ側は 20 ms ごとに lock を試し、取れないたびに gate.lock の marker を読んで持ち主を記録する(`acquireGateLock` の待ちの callback、`internal/org/gate_lock.go:123-131`)。前の gate の `gateLock.release` は、marker を空にしてから lock を外す(次の gate が前の marker を自分の持ち主と読まないため)。待つ側の最後の poll がこの隙間に当たると、持ち主が `a gate whose marker is not written yet` に上書きされ、log の見出し(`gateLockWait.logLines`)がその文になっていた。2026-11-20 の通しのテストで `TestGate_TwoGatesRunTheirScriptsOneAtATime` が 1 回これで落ちた。

## 2. 直し方

- `gateLockHolder`(`gate_lock.go:210`)は `(holder, named)` を返す。named は、marker が空でも読めなくもなく、持ち主の名前を言えたときだけ true
- 待ちの callback(`gate_lock.go:123-131`)は、読んだ文が空の marker(定数 `gateLockNotWritten`)で、今の持ち主に名前があるときだけ置き換えない。名前つきの marker と読めない marker は今までどおり置き換える。stderr の行も log の見出しも、この持ち主を使う
- 規則は `acquireGateLock` の doc に 1 か所だけ書き、`gateLockWait` と `gateLockHolder` の doc はそれを指す

決めたこと: 名前 → 読めない → 空 の順では、最後は `not written yet` になる。読めない文は名前ではないので、空で置き換わる(計画の Progress の S1)。

## 3. 先に読むテスト

- `TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty`(`gate_lock_test.go:399`): 6 つの形(名前 → 空、空だけ、名前 → 読めない、名前 → 読めない → 空、名前 → 空 → 別の名前、空 → 名前)。時間に頼らず、差し込み口 `afterGateLockRead`(`gate_lock.go:42-46`、本番では nil)で「待つ側がこの marker を読んだ」を受け取ってから次へ進む。codex の計画助言の指摘(時間に頼ると直しを外しても通りうる)による形
- `gateLockWaitLineHolders`(`gate_lock_test.go:139`): 待ちの行から持ち主を取り出す helper。行の書式は `gateLockWaitLine` の 1 か所にある
- `TestGateLockHolder`(`gate_lock_test.go:468`): named の値も確かめる

## 4. 確かめたこと

- `./scripts/run-test.sh` rc 0。新しいテスト `-count=50` で 300/300、2 つの gate のテスト `-count=10` で 10/10
- release の隙間に 200 ms の sleep を入れても、2 つの gate のテストは 5/5 で通る。直しを外して同じ sleep を入れると 5/5 で落ち、#357 と同じ文になる(test report)

## 5. 残したもの

tech-debt の「Known gaps of fix/gate-lock-holder-log (#357)」の行(`docs/tech-debt/README.md:546`): 待つ側が前の gate のあいだに一度も名前を読めなければ、2 つの gate のテストは今も落ちうる。regate の `setOrgID` が marker を書き直す途中の空の読みを再現するテストはない。新しい gate が marker を書くまで前の名前を出す遅れは測っていない。
