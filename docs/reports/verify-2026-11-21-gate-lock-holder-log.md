# Verify report: gate-lock-holder-log

- Date: 2026-11-21
- Plan: `docs/plans/active/2026-11-21-gate-lock-holder-log.md`(issue #357)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(base `origin/main` 09d27450 = merge-base、HEAD 4f7e6e56)。コードの変更は a29daaaf(直しと新しいテスト)と 4f7e6e56(self-review の LOW 1〜3 の直し。差し込み口の名前を `afterGateLockRead` に変え、持ち主の決まりを `acquireGateLock` の doc の 1 か所に書き、テストの取り出しを `gateLockWaitLineHolders` に出した)。静的検査と仕様の確かめは 4f7e6e56 を含む HEAD で行った。テストは打っていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-21-gate-lock-holder-log.log`(gitignore の対象。wrapper 自身の log は `docs/evidence/verify-2026-11-21-141552.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 名前を読めた持ち主のあとに空の marker を読み、そのあと lock を取った gate は、log の見出しと `gateLockWait.holder` にその持ち主の名前を書く | met(静的) | `internal/org/gate_lock.go:124-131`: `seen != gateLockNotWritten \|\| !named` のときだけ `w.holder` を替えるので、名前つきの持ち主のあとの空の marker は持ち主を替えない。log の見出しは `gateLockWait.logLines`(`:82-87`)が `w.holder` から作り、`Gate`(`gate.go:201,224`)と local の `Regate`(`regate.go:539,548`)の両方がこの値を渡す。新しいテスト `TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty`(`gate_lock_test.go:399`)の形「a named holder, then an empty marker」が `wait.holder`、`logLines()`、待ちの行の 3 つを確かめる。同期は差し込み口 `afterGateLockRead`(`gate_lock.go:42-46`、本番は nil)で取り、時間は 10 秒の失敗の区切りにだけ使う |
| AC2: 一度も名前を読めなかった gate は `a gate whose marker is not written yet` を書く | met(静的) | `named` の初期値は false(`gate_lock.go:120`)なので、空の marker は持ち主を `gateLockNotWritten` にする。新しいテストの形「an empty marker only」と、変わっていない `TestAcquireGateLock_SaysAMarkerIsNotWrittenYet`(`gate_lock_test.go:322`)が確かめる |
| AC3: 読めない marker は今のとおり `a gate whose marker cannot be read (...)` で持ち主を置き換える | met(静的) | `gateLockHolder` は読めない 3 つの道で `named=false` を返し(`gate_lock.go:213-225`)、文は `gateLockNotWritten` と違うので条件の左側で必ず置き換わる。新しいテストの形「a named holder, then a marker that cannot be read」と「a marker that cannot be read, then an empty marker」が確かめる。`TestGateLockHolder`(`:468`)は `named` の期待も足した |
| AC4: 名前つきから別の名前つきに替わったときは、今のとおり替わった行を出す | met(静的) | 名前つきの marker は常に置き換わり、`said` との比較(`gate_lock.go:132-135`)で替わった行が 1 本出る。`TestAcquireGateLock_SaysSoWhenTheHolderChanges`(`gate_lock_test.go:230`)は diff で変わっていない。新しいテストの形「a named holder, an empty marker, then another named holder」は待ちの行が `[b, c]` の 2 本だけ(空の読みで行が増えない)ことを確かめる |
| AC5: `TestGate_TwoGatesRunTheirScriptsOneAtATime` と既存の gate.lock のテストが変わらず通る。新しいテストは直しを外すと落ちる | partially met(静的に確かめた分) | `gate_lock_gate_test.go` は diff に入っていない(`git diff origin/main...HEAD --stat` が空)。既存のテストの変更は、`gateLockHolder` の戻り値が 2 つになったことへの合わせ(`TestGateLockHolder` に `named` の期待を足す、`TestGate_AScriptKeepsTheLockAfterItsRalphIsKilled` で 2 つめを捨てる)だけで、期待を弱めていない。直しを外す 2 つの変異は読みで追った: 「空でも常に置き換える」は形 1 で `holder` が not written yet になり、形 5 で待ちの行が `[b, notWritten, c]` になって落ちる。「空では置き換えない」は形 2 で持ち主が空文字のままになり、形 4 で `unreadable` が残って落ちる。テストを通すことと変異を実際に打つことは `/test` の担当で、ここでは確かめていない(計画の Progress の S1 は implementer が変異 2 種で落ちたと記録) |
| issue #357 の「直し方の案」 | met | 案 1「空の marker を読んだときに、それまでに見た空でない持ち主を上書きしない」を採った。案の「空でない持ち主」は、名前を読めた持ち主に狭めてある(読めない marker のあとの空は not written yet)。計画の Progress S1 に決めたこととして書かれ、AC3 と合う。`gateLock.release` の順、`waitHeldFileLock`、poll の間隔、`setOrgID` は変わっていない(計画の Non-goals どおり) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope) | pass(`rc=0`、`All verifiers passed.`) | `Language scope: changed (changed_languages)`、`Language packs selected: golang`。local verifier(shellcheck、`sh -n`、check-sync `DRIFTED: 0`、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README plan references)はすべて OK |
| gofmt(golang verifier の中) | pass | `gofmt: ok` |
| `go vet ./...`(同) | pass | 出力なし |
| `golangci-lint run ./...`(同) | pass | `0 issues.`。ロックの取り合いの揺れは出なかった |
| `staticcheck ./...`(同) | pass | 出力なし(skip の文も出ていないので走った) |
| branch secret scan(同) | pass | `scanned 09d27450..4f7e6e56 against origin/main: clean` |
| コミットの帰属の行 | なし | `git log origin/main..HEAD` の本文に `Co-Authored-By` と `Generated with` はない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `acquireGateLock` の doc(`gate_lock.go:97-115`) | yes | 決まり(最後の読みを使う、空の marker は名前つきの持ち主を替えない、名前がなければ not written yet、読めない marker は常に置き換える、次の gate が marker を書くまでは前の名前のまま)がコードの条件と一致する。空になる場面として挙げた `gateLock.release`(`:181`)と `writeGateLockContent`(`:191`、truncate してから書く)も実際の動きと合う |
| `gateLockWait` の doc(`:72-74`)と `gateLockHolder` の doc(`:201-209`) | yes | どちらも `acquireGateLock` の doc の決まりを指すだけで、「最後の poll で読んだ持ち主」とは書いていない。`named` の説明(空でも読めなくもないときだけ true)は戻り値と一致する |
| `afterGateLockRead` と `gateLockNotWritten` の doc(`:42-50`) | yes | 呼ばれる位置(読んだあと、持ち主を選ぶ前)と nil の扱いがコードと一致する |
| ファイル冒頭の gate.lock の説明(`gate_lock.go:18-30`) | yes | 「stderr に誰が持つかを言う」「release で marker を空にする」のままで、食い違いはない |
| AGENTS.md の `internal/org/` の gate.lock の説明 | yes | 「a wait with no time limit whose stderr lines name the holder」のまま正しい |
| README.md:292 と `ralph org gate --help`(`internal/cli/org_gate.go:57-64,91-92`)と `ralph org regate --help` | yes | 「待ちの始め、持ち主が替わったとき、5 分ごとに行が出る」「log の見出しが待った時間と持ち主を言う」のどちらも、直しのあとの動きと合う(release の隙間の行が 1 本減るだけで、行の出る場面の言い方は変わらない) |
| `docs/specs/2026-10-07-org-multi-org-director.md:140`(2026-11-12 の追記) | yes | 「待つ間は stderr に、ロックを持つ gate を出す」だけで、log の見出しの持ち主の決め方は書いていない。`docs/specs/` に「最後の poll で読んだ持ち主」と書いた箇所はない |
| `/director` skill(4 つの写し、1017-1022 行と 1350 行)、`/org` skill(1975 行、2407 行)、`internal/org/prompts/reviewer.md:82-85` | yes | 待ちの行の出る場面と `# waited ... for gate.lock` の行を読むことだけを書き、持ち主の決め方には触れていない |
| 計画 `docs/plans/active/2026-11-21-gate-lock-holder-log.md` の Progress checklist | no(LOW、V1) | 下の Findings を参照 |
| `docs/tech-debt/README.md` | yes(Trigger を引いた行なし) | 下の「Tech-debt の Trigger の確かめ」を参照 |

### Findings

| ID | Severity | File:line | Finding | For /sync-docs |
| --- | --- | --- | --- | --- |
| V1 | LOW | `docs/plans/active/2026-11-21-gate-lock-holder-log.md:96-97` | Progress の S1 の行が差し込み口を `gateLockPolled` と書いている。4f7e6e56 で `afterGateLockRead` に名前が変わったが、Progress にはこの直しのコミットの行がない。97 行の「Review artifact created」も、401b7855 で self-review の report をコミット済みなのに `[ ]` のまま | S1 の行の `gateLockPolled` を残すなら「(4f7e6e56 で `afterGateLockRead` に改名)」と書き足し、self-review の LOW 1〜3 を 4f7e6e56 で直したことを 1 行で記録する。「Review artifact created」と「Verification artifact created」に印を付ける |
| I1 | INFO | `docs/reports/self-review-2026-11-12-gate-lock-accept.md:53` | 2026-11-12 の self-review の「参考(指摘にはしない)」の 1 項目目(最後の poll が release の隙間に当たると見出しが not written yet になる、「印が空のときは前に読めた保持者を残す」)を、この PR が直した。この項目は register の行になっていないので、閉じる行はない。過去の report は書き換えない | 要らない。PR の本文で触れるなら「#357 と 11-12 の self-review の参考の項目を直した」と書ける |
| I2 | INFO | `internal/org/gate_lock_test.go` 494 → 643 行、`internal/org/gate_lock.go` 215 → 245 行 | どちらも 800 行の目安の内。この 2 つを名指す register の行もない | 要らない |

### Tech-debt の Trigger の確かめ

`docs/tech-debt/README.md` の各行を `re.split(r"(?<!\\) \| ", row)` で分け、4 つめの列(Trigger)を `gate_lock`、`gateLock`、`gate.lock`、`writeGateLockContent`、`TwoGates`、`Gate`、「next change to the gate」で探した。当たった行を 1 つずつ読み、どれも引いていないと判断した。

- 474(`gate.lock` has no wait limit and gives no order): Trigger は「持ち主を止める必要があった人、順を外れて走った gate の報告、要らない gate を待った regate」。#357 は待った相手の名前の記録の誤りで、どれにも当たらない
- 475(SIGKILL のあとに `setsid` の子が lock を持ち続ける): Trigger は「終わった pid を名指し、自分では終わらない待ち」。当たらない
- 476〜478、486(gate-lock-accept の残り): accept、load average、`waitGateMaxLoad`・`gateRunLoad.finish`・`newOrgRegateCmd` の変更が Trigger で、この PR はどれも触らない
- 161(FR-7 の品質ゲートが prompt の指示に頼る): Trigger は「org の品質パイプラインの次の変更」。#311 は記録と出力と skill を足して当たりと書いたが、この PR は待ちの行と log の見出しの持ち主の名前を、release の隙間で正すだけで、gate の判定、スクリプト、手順、記録の鍵を変えないので、当たらないと判断した
- 241、372(`gate.go`、`gate_run.go`、`lockfile.go` の次の変更)、403 (a)(`gateScriptEnv`、`Regate`、`runGateScripts`)、298・410(`regate*.go`)、299(gate の記録の鍵): この PR はこれらのファイルと関数を変えていない(`git diff origin/main...HEAD --stat` は `gate_lock.go` と `gate_lock_test.go` だけ)
- 88(org-debt-batch の 6 件、「対応するファイルに触れたとき」): 項目に `gate_lock*.go` はない

## Observational checks

- 待つ側の stderr の行は、持ち主が `w.holder == said` のままなら出ない(`gate_lock.go:132-134`)。直しのあとは、空の marker の読みが名前つきの持ち主を替えないので、release の隙間で「not written yet」の行が 1 本増えることもない。新しいテストの `said` の列が、行の数まで確かめている
- 新しい gate が lock を取ってから marker を書くまでの間(数十 ms)、待つ側は前の持ち主の名前を出し続ける。計画の Risks と `acquireGateLock` の doc(「Until a gate that took the lock writes its marker, the name stays the one before it」)の両方に書いてある
- テストの `writeGateFile` は truncate してから書くので、書き込みの途中で空が読まれうる。6 つの形のどれでも途中の空は結果を変えない(self-review と同じ結論を読みで確かめた)
- `TestGate_TwoGatesRunTheirScriptsOneAtATime` の 2 つの gate はどちらも `org-a` を gate する(`gate_lock_gate_test.go:218` の期待は、どちらが先に走っても `org-a` で正しい)

## Coverage gaps

- テストは 1 本も打っていない(`/test` の担当)。AC1〜AC5 の「met」は読みと静的検査による
- AC5 の「直しを外すと新しいテストが落ちる」は、2 つの変異を読みで追っただけで、実際には打っていない。`/test` で `-overlay` などの変異を 1 つ打てば確かめられる
- 計画の Objective のとおり、待つ側が前の gate のスクリプトの間に一度も名前を読めなかったときは、`TestGate_TwoGatesRunTheirScriptsOneAtATime` は今も落ちうる。この PR では扱わない(計画の Risks)
- 決定的な検査の不足はない。新しいテストが差し込み口で同期しているので、追加の verifier は提案しない

## Verdict

- Verified: AC1〜AC4 と issue #357 の直し方の案(静的、コードとテストの読み)。`./scripts/run-static-verify.sh` が rc=0(gofmt、go vet、golangci-lint 0 issues、staticcheck、secret scan clean)。doc comment、AGENTS.md、README、CLI の help、spec、skill、reviewer の prompt に食い違いはない。tech-debt の Trigger を引いた行はない
- Partially verified: AC5(テストの期待を弱めていないこと、`gate_lock_gate_test.go` が変わっていないこと、変異で落ちることは読みで確かめた。実際に通ること、落ちることは `/test`)
- Not verified: テストの実行、変異の実行
- Findings: LOW 1 件(V1、計画の Progress のずれ)、INFO 2 件
- Verdict: pass
