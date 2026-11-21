# Test report: gate-lock-holder-log

- Date: 2026-11-21(cycle 1。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-gate-lock-holder-log.md`(issue #357、AC1〜AC5)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `a4c475e5`、worktree は clean。`./scripts/run-test.sh` は 1 回だけ実行した。この run は、アプリの再起動で途切れた前の tester が 23:22:36 JST(HEAD `a4c475e5` の commit の 96 秒後)に worktree のルートから起動したもので、再起動のあとも動き続けていた。2 本目の全体の run を重ねないよう、新しく起動せずにこの run の終わりを待って結果を使った。起動の形は perl の fork と setsid による切り離しで、親は pid 1、出力はファイルに向いている。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと、AC に絞った `-v` の run を 3 本、`go test -overlay` による変異を 6 個(依頼の 2 種と対照 1 個、追加 3 個)、絞った run の coverage を取った。追跡しているファイルは何も変えていない(変異の前後で `git status --porcelain --ignored internal/` は同じ)
- Evidence: `docs/evidence/test-2026-11-21-gate-lock-holder-log.log`(6,937 行。run-test.sh の出力 5,829 行のあとに、run の開始と終了の時刻、30 秒ごとの load、AC の run の出力、変異ごとの diff と出力、coverage を足した。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-21-142236.log` にも書いた(同じく追跡外)
- 環境: go1.27.3 darwin/arm64、git 2.49.0、12 コア

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-21 23:22:36 〜 23:45:39 JST、1,383 s、rc 0(`==> All verifiers passed.`)。30 秒ごとの 1 分 load は 4.93〜23.21 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL: 0` の要約行 29 本、`N passed, 0 failed` の要約行 14 本) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0)。前の cycle と同じ 2 件 | 約 9 分(23:22:36〜23:31:49) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 830 s(23:31:49〜23:45:39)、下の表 |
| AC run 1: `go test -count=1 -v ./internal/org/ -run 'GateLock\|TwoGates\|AcquireGateLock'` | 20 関数、subtest 20 | 40(関数 20、subtest 20) | 0 | 0 | 19 s(パッケージ 17.629 s)、1 分 load 11.1 |
| AC run 2: 新しいテスト `TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty` を `-count=50 -v` | 50 回 × 6 subtest | 関数 50/50、subtest 300/300(6 つの形がそれぞれ 50 回) | 0 | 0 | 9 s(パッケージ 8.198 s)、1 分 load 11.2〜13.6 |
| AC run 3: `TestGate_TwoGatesRunTheirScriptsOneAtATime` を `-count=10 -v` | 10 | 10 | 0 | 0 | 29 s。1 回 2.46〜3.58 s、1 分 load 11.8〜13.6 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.784 s |
| `internal/cli` | ok | 787.636 s |
| `internal/config` | ok | 12.435 s |
| `internal/gitroot` | ok | 3.608 s |
| `internal/insights` | ok | 1.566 s |
| `internal/org` | ok | 813.757 s |
| `internal/org/driver` | ok | 3.603 s |
| `internal/org/protocol` | ok | 3.850 s |
| `internal/scaffold` | ok | 2.975 s |
| `internal/toolchain` | ok | 19.182 s |
| `internal/upgrade` | ok | 4.179 s |

`internal/org` は golang pack の 1 パッケージの上限(30 分)の 45% で終わった。run-test.sh には、ほかの run が 2 度重なった。1 度目は local verifier の段(23:23〜23:28 ごろ)で、前の tester の AC の run と `gate_lock.go` の変異の run。2 度目は Go の段(23:34〜23:41)で、この report の AC の run と変異の run。落ちたテストはなく、打ち直したテストもない。

### 計画の AC とテスト

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty` の「a named holder, then an empty marker」 | 名前つきの marker のあとに空の marker を読んでから lock を取った gate は、`gateLockWait.holder` と log の見出し `# waited <d> for gate.lock: <holder>` に前の持ち主の名前を書く。stderr の待ちの行も、その名前の 1 行だけ。差し込み口 `afterGateLockRead` で「名前を読んだ」「空を読んだ」を受け取ってから次へ進むので、poll の間隔に頼らない |
| AC2 | 同じテストの「an empty marker only」、既存の `TestAcquireGateLock_SaysAMarkerIsNotWrittenYet` | 一度も名前を読めなかった gate は `a gate whose marker is not written yet` を書く |
| AC3 | 同じテストの「a named holder, then a marker that cannot be read」「a marker that cannot be read, then an empty marker」、`TestGateLockHolder` | 読めない marker は名前つきの持ち主を置き換える。そのあとの空の marker は `not written yet` に置き換わる。`gateLockHolder` は空・改行だけ・読めない・ファイルがない、のどれにも `named=false` を返す |
| AC4 | 既存の `TestAcquireGateLock_SaysSoWhenTheHolderChanges`、同じテストの「a named holder, an empty marker, then another named holder」「an empty marker, then a named holder」 | 名前から別の名前への替わりは、今のとおり替わった行を出す。間に空の marker があっても `not written yet` の行は出ない |
| AC5 | AC run 1 の 20 関数、run-test.sh の `internal/org`、AC run 3、変異 M1 と M2 | 既存の gate.lock のテストがすべて通る。新しいテストは直しを外すと落ちる(M1)。release の隙間を広げても 2 つの gate のテストが通る(M2) |

### 変異(`go test -overlay`、追跡ファイルは変えていない)

| # | 変異(`internal/org/gate_lock.go`) | run | 結果 |
| --- | --- | --- | --- |
| M1(依頼 1) | 残す条件を常に置き換えに: `if seen != gateLockNotWritten \|\| !named \|\| true {` | `-run 'GateLock\|TwoGates\|AcquireGateLock' -count=1`、新しいテストだけを `-count=5` | 検出。新しいテストの AC1 の 2 行だけが落ちた: 「a named holder, then an empty marker」(`gate_lock_test.go:448`、`wait = {… holder:a gate whose marker is not written yet}`)と「a named holder, an empty marker, then another named holder」(`gate_lock_test.go:454`、待ちの行が `[org-b, not written yet, org-c]`)。`-count=5` では 2 行が 5/5 で落ち、残りの 4 行は 5/5 で通った。ほかの 19 関数は通った |
| M2(依頼 2) | `gateLock.release` で marker を空にしたあと、lock を外す前に `time.Sleep(200 * time.Millisecond)` を入れる(直しは残す) | `TestGate_TwoGatesRunTheirScriptsOneAtATime -count=5` | 5/5 通った(1 回 3.50〜4.98 s)。直しがあれば、隙間を 200 ms に広げても待った側の log は持ち主の名前を書く |
| M2 の対照 | M1 と M2 を重ねる(直しを外し、隙間を 200 ms に広げる) | 同じ `-count=5` | 5/5 落ちた。5 回とも `gate_lock_gate_test.go:219: log of the gate that waited has no wait for the other gate`、log の見出しは `a gate whose marker is not written yet`。#357 と、#353 の tester が直す前に見た 5/5 と同じ文。M2 が通るのは直しのおかげだと分かる |
| M3(追加) | 名前がないときの置き換えを外す: `\|\| (!named && false)` | `-run '^(TestAcquireGateLock_\|TestGateLockHolder$)'` | 検出。既存の `SaysAMarkerIsNotWrittenYet` と、新しいテストの 3 行(「an empty marker only」「a marker that cannot be read, then an empty marker」「an empty marker, then a named holder」) |
| M4(追加) | 読んだものを常に名前つきと記録する: `named = seen, seenNamed \|\| true` | 同上 | 検出。新しいテストの「a marker that cannot be read, then an empty marker」(読めない文を名前として残し、`not written yet` に替わらない) |
| M5(追加) | `gateLockHolder` が空の marker に `named=true` を返す | 同上 | 検出。`TestGateLockHolder/empty` と `/a_line_break_only` |

M3 と M4 の最初の形(`|| !named` を消す、`seenNamed` を `true` にする)は変数が使われなくなってビルドが落ちたので、`&& false` と `|| true` の形で作り直した。ビルドの失敗のまま数えた変異はない。

## Coverage

- Statement: 変更行の coverage は、`internal/org/gate_lock.go` の追加行のうち、絞った run(`go test -count=1 -coverprofile … ./internal/org/ -run 'GateLock|TwoGates|AcquireGateLock'`)の profile の文の block に入る 13 行すべてが実行された。関数ごとには `gateLockHolder` 100%、`acquireGateLock` 92.9%、`release` 75.0%(残りは書き込みの失敗の道で、変更行ではない)
- Branch: 計測していない(Go の標準の道具にない)
- Function: 絞った run なので、パッケージ全体の数字は出していない(パッケージの 8.8%)
- Notes: 残す条件の 4 つの組み合わせ(名前つき・空・読めない × 前に名前があるか)は、新しいテストの 6 行と M1・M3・M4 の変異で、どの項を外しても落ちることを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #357: 待った gate の log が、前の gate の release の隙間に読んだ空の marker から `a gate whose marker is not written yet` を書く | 直った | M2 の対照(直しなし + 200 ms の隙間)は 5/5 で落ち、M2(直しあり + 同じ隙間)は 5/5 で通る。AC run 3 は 10/10、run-test.sh の `internal/org` も ok |
| 名前から別の名前への替わりの行(AC4) | 変わらず通る | `TestAcquireGateLock_SaysSoWhenTheHolderChanges` が AC run 1 と run-test.sh で通った |
| `gateLockHolder` の戻り値が `(holder, named)` になった。既存の呼び出し | 変わらず通る | `TestGateLockHolder`(期待する文は変えず、`named` の確かめを足しただけ)と `TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled` が通った |
| regate の gate.lock の待ち | 変わらず通る | `TestRegate_WaitsForGateLockWithRegateLockFree`、`MarksGateLockWithTheBranchAndThenItsOrg`、`ARefusalOrAnInterruptReleasesGateLock`、`ACIRegateTakesNoGateLock` が AC run 1 で通った |

## Test gaps

1. `TestGate_TwoGatesRunTheirScriptsOneAtATime` は、待つ側が前の gate の scripts のあいだ(約 0.6 秒)に一度も名前つきの marker を読めなかったとき、今も落ちうる。計画の Objective と Risks が扱わないと決めた道で、この run では起きなかった(AC run 3 は 10/10、M2 は 5/5、run-test.sh も ok)。起きたときの文は #357 と同じになるので、見分けるには待つ側の stderr の行を見る(名前の行が 1 本もなければこの道)
2. `writeGateLockContent` が marker を書き直す途中の空(regate の `setOrgID`)は、doc では同じ規則で前の名前を残すと書いているが、本物の regate がその間に読まれる場合のテストはない。読む側の動きは release の隙間と同じ callback を通り、新しいテストの「a named holder, then an empty marker」がその規則を固定しているので、重さは LOW
3. 新しい gate が lock を取ってから marker を書くまでのあいだ、待つ側は前の持ち主の名前を出し続ける(計画の Risks の 1 つ目、次の poll まで)。「a named holder, an empty marker, then another named holder」が、そのあとに新しい名前へ替わることを固定している。遅れの長さそのものは測っていない

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,383 s、`(cached)` なし)、AC run 1(20 関数、subtest 20)、新しいテストの `-count=50`(300/300)、`TestGate_TwoGatesRunTheirScriptsOneAtATime` の `-count=10`(10/10)、AC1〜AC5 のテスト、依頼の変異 M1 を検出、M2 は直しで 5/5 通り対照は 5/5 落ちる、追加の変異 M3〜M5 も検出
- Fail: 0
- Blocked: なし
