# gate-lock-holder-log

- Status: Approved
- Approved: 2026-11-21 sha256:b5dcd0e88350
- Owner: Claude Code
- Date: 2026-11-21
- Related request: issue #357(待った側の gate の log が、持ち主の名前でなく「marker がまだない gate」を書くことがある)
- Related issue: 357
- Type: fix
- Branch: fix/gate-lock-holder-log

## Objective

gate.lock を待った gate が log の見出しに書く持ち主を、前の gate の名前にする。前の gate が marker を空にしてから lock を外すまでの隙間を待つ側の最後の poll が読んでも、`a gate whose marker is not written yet` にならないようにする。これで `TestGate_TwoGatesRunTheirScriptsOneAtATime` が release の隙間で落ちることはなくなる。待つ側が前の gate の scripts のあいだ(約 0.6 秒、poll は 20 ms ごと)に一度も名前を読めなかったときは、このテストは今でも落ちうる(待つ側がその間ずっと止められた場合だけで、この PR では扱わない)。

## Scope

- `acquireGateLock`(`internal/org/gate_lock.go`)の待ちの callback: 空の marker(`gateLockHolder` が `a gate whose marker is not written yet` を返す形)を読んだとき、それまでに名前を読めた持ち主があれば、その持ち主を残す。待つ側の stderr の行と log の見出し(`gateLockWait.holder`)の両方がこの持ち主を使う
- 名前を読めた持ち主がまだいないとき(lock を取った直後の gate を最初に読んだときなど)は、今のとおり `a gate whose marker is not written yet` を使う
- marker を読めないとき(`a gate whose marker cannot be read (...)`)は、今のとおりその文で置き換える。release の隙間ではないので
- 隙間を確実に作る決定的なテストを足す。時間に頼らず、poll ごとに読んだ marker の文をテストが受け取れる差し込み口(本番では nil の package 変数の関数など)で同期する。順は、持ち主が名前つきの marker で lock を持つ → 待つ側がその名前を読んだのを受け取る → marker を空にする → 待つ側が空の marker を読んだのを受け取る → lock を外す。待った側の log の見出しと stderr が、その持ち主の名前を言うことを確かめる。待ち時間は失敗を区切るためだけに使う

## Non-goals

- `gateLock.release` の順(marker を空にしてから lock を外す)は変えない。次の gate が前の gate の marker を自分の持ち主として読まないための順(`release` の doc)
- gate.lock の待ちの仕組み(`waitHeldFileLock`、poll の間隔、知らせの間隔)は変えない
- regate の `setOrgID` など、marker を書き直す道は変えない

## Assumptions

- release の隙間で読む空の marker は、持ち主が変わったことを意味しない。新しい gate が lock を取って marker を書く前の隙間も同じ文になるが、その間に前の持ち主の名前を出し続けても、次の poll で新しい marker が読めれば名前が替わる(数十 ms の遅れ)
- 2026-11-12 の gate-lock-accept の self-review が同じ窓を書き、「印が空のときは前に読めた保持者を残す」直し方を挙げていた(`docs/reports/self-review-2026-11-12-gate-lock-accept.md:53`)

## Affected areas

- `internal/org/gate_lock.go`(`acquireGateLock` の待ちの callback、必要なら空の marker の文を定数にする)
- `internal/org/gate_lock_test.go`(決定的なテストを足す)

## Visual review

None (1 つの関数の中の小さな直しで、部品の間のやり取り、状態の移り変わり、データの形のどれも変えない)

## Design decisions

Critical forks: None

- stderr の行も log と同じ持ち主を使う。空の marker で持ち主を替えないので、release の隙間で「marker がまだない gate」の行が 1 本増えることもなくなる。新しい gate が marker を書く前の数十 ms は前の持ち主の名前のままになるが、次の poll で替わる

## Acceptance criteria

- [ ] AC1: 名前を読めた持ち主のあとに空の marker を読み、そのあと lock を取った gate は、log の見出し(`# waited <d> for gate.lock: <holder>`)と `gateLockWait.holder` に、その持ち主の名前を書く(決定的なテストで確かめる)
- [ ] AC2: 待ちのあいだ一度も名前を読めなかった gate は、今のとおり `a gate whose marker is not written yet` を書く
- [ ] AC3: 読めない marker は今のとおり `a gate whose marker cannot be read (...)` で持ち主を置き換える
- [ ] AC4: 持ち主が名前つきの marker から別の名前つきの marker に替わったときは、今のとおり替わった行を出す(`TestAcquireGateLock_SaysSoWhenTheHolderChanges` が変わらず通る)
- [ ] AC5: `TestGate_TwoGatesRunTheirScriptsOneAtATime` と gate.lock の既存のテストが変わらず通る。新しいテストは、直しを外すと落ちる

## Implementation outline

1. S1 持ち主の残し方とテスト(AC1〜AC5) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'GateLock|TwoGates|AcquireGateLock'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)
- Spec compliance criteria to confirm: AC1〜AC5 と issue #357 の「直し方の案」
- Documentation drift to check: `acquireGateLock` と `gateLockWait` と `gateLockHolder` の doc、AGENTS.md の gate.lock の説明(「stderr lines name the holder」)、gate.lock の仕様(docs/specs の gate.lock の項)が、最後の poll で読んだ持ち主と書いていないか
- Evidence to capture: verify の report

## Test plan

- Unit tests: 新しい決定的なテスト(差し込み口で同期して、名前つき → 空 → lock を取る、名前を残す)、空のまま取る(AC2)、読めない marker(AC3)
- Integration tests: `TestGate_TwoGatesRunTheirScriptsOneAtATime`
- Regression tests: gate.lock の既存のテストすべて
- Edge cases: 名前 A → 空 → 名前 B(B に替わる)、空 → 名前 A(A に替わる)
- Evidence to capture: `./scripts/run-test.sh` の log、直しを外す変異で新しいテストが落ちること

## Risks and mitigations

- 新しい gate が lock を取った直後(marker を書く前)の数十 ms、前の持ち主の名前を出し続ける。次の poll で替わるので、待つ側の知らせの間違いは一瞬に限られる
- テストが時間に頼ると、待つ側が隙間のあいだ一度も poll しないとき、直しを外しても通ってしまう(codex の計画助言): 差し込み口で「名前を読んだ」「空を読んだ」を受け取ってから次へ進み、時間は失敗を区切るためだけに使う。差し込み口は本番では nil で、呼ぶ側の動きを変えない
- `TestGate_TwoGatesRunTheirScriptsOneAtATime` は、待つ側が名前を一度も読めなかったときに今も落ちうる(Objective の注)。この PR では同期させない

## Rollout or rollback notes

この PR を revert すれば、最後の poll で読んだ持ち主を書く今の動きに戻る。台帳の形は変えない。

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件: [MEDIUM] 100 ms の隙間と 10 ms の poll に頼るテストは、待つ側が隙間のあいだ一度も poll しなければ直しを外しても通る。2 つの gate のテストも名前を読んだことを保証しない → 計画に入れた(差し込み口で「名前を読んだ」「空を読んだ」を受け取って同期する、Objective の言い切りを狭めた)。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-21、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
