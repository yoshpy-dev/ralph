# sync-docs report: gate-lock-holder-log

- Date: 2026-11-21
- Plan: `docs/plans/active/2026-11-21-gate-lock-holder-log.md`(承認の digest `b5dcd0e88350`。この run の前後とも同じ)。issue #357
- Branch: `fix/gate-lock-holder-log`。run 開始時の HEAD は `29e691c1`(test report の commit)。origin/main `09d27450` は merge-base と同じ
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `401b7855`(Merge: yes、LOW 3 件。直しは `4f7e6e56`)、verify `a4c475e5`(pass、LOW 1 件 V1、INFO 2 件 I1・I2)、test `29e691c1`(pass、`run-test.sh` rc 0、test gaps 3 件)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: この task の変更 `git diff origin/main...HEAD`。コードは `internal/org/gate_lock.go` と `internal/org/gate_lock_test.go` の 2 ファイルだけ(`a29daaaf` の直しとテスト、`4f7e6e56` の self-review の直し)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go のファイルは変えていない。計画の本文は `## Progress checklist` と AC の箱の `[x]` 以外を変えておらず、`./scripts/plan-visual.sh digest` は編集の前後とも `b5dcd0e88350`
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 8 ファイル、+555/-20(計画、report 3 本、insights、tech-debt を含む)。Go は 2 ファイルで +199/-20(`gate_lock.go` +44/-14、`gate_lock_test.go` +155/-6)。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify の V1(計画の Progress のずれ)を直し、test report の穴 3 件を tech-debt の 1 行に残した。通常の drift の確認(AGENTS.md、README、gate.lock の仕様、`ralph org gate --help`)は、どれも変更が要らなかった。この PR は待つ側が log の見出しと stderr に書く持ち主の決め方を 1 点直すだけで、行の出る場面(待ちの始め、持ち主が替わったとき、5 分ごと)は変えていない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/plans/active/2026-11-21-gate-lock-holder-log.md` | Progress の S1 に 4f7e6e56 の行を足し(`gateLockPolled` を `afterGateLockRead` に改名、self-review の LOW 1〜3 を直した)、Review・Verification・Test の artifact の箱を `[x]` にして各 report の path を添えた。AC1〜AC5 の箱を `[x]` にした。`git diff --numstat` は 9/8(行の数は箱の置き換えと足した 1 行) | `07d77657` |
| `docs/tech-debt/README.md` | 新しい 1 行を末尾に足した(546 行目、`Known gaps of fix/gate-lock-holder-log (#357)`)。既存の行は変えていない(`git diff --numstat` は 1/0) | `07d77657` |
| `docs/tech-debt/README.md` | その新しい行の Why deferred の列に、「計画は理由を書いていないので、この列の残りは計画の読み」の 1 文を足した(下の 2 の Why deferred の項) | この report の commit |
| `docs/plans/active/2026-11-21-gate-lock-holder-log.md` | Progress に sync-docs の 1 行(この report を指す) | この report の commit |
| `docs/reports/sync-docs-2026-11-21-gate-lock-holder-log.md` | この report | この report の commit |
| `docs/insights/events/2026-11-21-gate-lock-holder-log.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 | この report の commit |

## 依頼の項目

### 1. verify V1: 計画の Progress

- S1 の行には、4f7e6e56 の行を子の項として足した。S1 の本文の `gateLockPolled` は書き換えず、「上の `gateLockPolled` は改名前の名前」と添えた(S1 は a29daaaf の記録で、その時点の名前が正しい)。テストの helper の改名(`gateLockReads`、`watchGateLockReads`)、持ち主を残す条件の注釈と決まりの 1 か所化、`gateLockWaitLineHolders`、`waitFor` の doc は、4f7e6e56 の commit message と `git show --stat` で確かめた
- 箱: `Review artifact created`、`Verification artifact created`、`Test artifact created` を `[x]` にした。それぞれ report の path と verdict を 1 行で添えた。`PR created` は `[ ]` のまま
- AC1〜AC5: test report の表で、AC1 は新しいテストの「a named holder, then an empty marker」、AC2 は「an empty marker only」と既存の `SaysAMarkerIsNotWrittenYet`、AC3 は「a named holder, then a marker that cannot be read」など、AC4 は既存の `SaysSoWhenTheHolderChanges` と新しいテストの 2 形、AC5 は AC run 1 の 20 関数と `run-test.sh` と変異 M1 の検出。5 つとも通っているので、5 つとも `[x]` にした
- digest: 編集の前後とも `./scripts/plan-visual.sh digest` が `b5dcd0e88350`。`## Progress checklist` と AC の箱のほかは触っていない(`git diff -U0` の `-` の行は AC の 5 行と Progress の 3 行だけ)

### 2. test report の穴: tech-debt の行を足した

足すと決めた。register の目的は「チャットの履歴に消えさせない」ことで、穴 1 は計画の Objective と Risks にだけ書かれていて、計画は `/pr` で archive へ移る。「#357 を閉じる PR のあとに、同じ文で落ちる flake が残りうる」ことは、flake を見た人が探す場所(register と、落ちたテストの名前)に要る。穴 2 と 3 は同じ関数の周りの穴で、別の行にすると同じ Trigger が 2 回並ぶので、1 つの行の (b)・(c) にまとめた。

| 項目 | 書いたこと |
|------|-----------|
| (a)(test の穴 1) | `TestGate_TwoGatesRunTheirScriptsOneAtATime` は、待つ側が前の gate の scripts のあいだ(`sleep 0.3` の 2 本、約 0.6 秒、poll は 20 ms ごと)に一度も名前を読めなかったとき、今も #357 と同じ文で落ちうる。直しが閉じたのは最後の読みが `release` の隙間に当たる場合だけ。この run では起きなかった(10/10、隙間を 200 ms にして 5/5、`run-test.sh` rc 0)。見分け方は、待つ側の stderr に前の gate を名指す行が 1 本もないこと |
| (b)(test の穴 2) | regate が marker を書き直す途中(`gateLock.setOrgID` は `writeGateLockContent` を通り、truncate してから書く)の空の読みを、本物の regate で起こすテストはない。読む側は release の隙間と同じ callback で、新しいテストの「名前 → 空」が規則を固定している |
| (c)(test の穴 3) | 新しい gate が lock を取ってから marker を書くまで、待つ側は前の名前を出し続ける。計画の Risks の「数十 ms」は測っていない。「名前 → 空 → 別の名前」の形は、新しい marker を読んだら名前が替わることだけを固定している |

- Impact は LOW。(a) は CI のテストが落ちるだけで、コマンドの動きは変わらない(1 本の gate の log の見出しが持ち主を言わないだけ)。(b)(c) は待つ側の行の持ち主が 1 poll(本番で 1 秒)だけ前の名前になる
- Why deferred は、計画の Objective と Risks の言い方(「この PR では同期させない」)を引いたうえで、計画が理由を書いていないことを明記した(初めに足した文は、理由が計画のものに読めたので、この report の commit で「give no reason; the rest of this cell is a reading of it」の 1 文を足した)。列の残りは読みで、「2 つの gate のテストは本物の scripts を走らせるので、待つ側が名前を読むまで前の gate の scripts を止める仕組みが要る。それは計画の Scope の外の、このテストの変更」と書いてある
- Trigger は symbol で書いた(行番号は使っていない)。「`TestGate_TwoGatesRunTheirScriptsOneAtATime` が `a gate whose marker is not written yet` で落ち、待つ側の stderr が gate を名指さない」と、「`acquireGateLock`、`gateLockHolder`、`writeGateLockContent` の次の変更」の 2 つ
- Related は計画を `docs/plans/active/...` で書いた(`/pr` の `archive-plan.sh` が書き換える。`verify.local.sh` の plan 参照の確かめは path の存在を見るので、先に `archive/` とは書かない)。test report、`gate_lock.go`、`gate_lock_gate_test.go`、`gate_lock_test.go` を足した。存在は `test -f` で確かめた

#### 既存の行の Trigger の掃引

verify が列で引いた結果に、自分でも `gate_lock`、`gateLock`、`acquireGateLock`、`afterGateLockRead`、`gate.lock`、`TwoGates`、`waitHeldFileLock`、`writeGateLockContent`、`setOrgID`、`next change to the gate` を 4 つ目の列(Trigger)と行全体の両方で探した。Trigger の列に当たったのは 161、241、298、299、372、395、401、403、410、474、526、539 で、Trigger の文を 1 つずつ読んだ。

この PR が変えたのは `gate_lock.go` と `gate_lock_test.go` だけで、当たった行の語は、Trigger の元の文ではなく、過去の PR(#309、#263、#342 など)が足した持ち越しの note の中の `gate.lock` が大半だった。各行の Trigger の元の文(冒頭)は次のとおり。

- 161: org の品質パイプラインの次の変更。この PR は gate の判定、scripts、手順、記録の鍵を変えず、待つ側の行と log の見出しの持ち主の名前を release の隙間で正すだけ(verify と同じ判断)
- 241、372: `gate.go`、`gate_run.go`、`lockfile.go` の次の変更。この PR は 3 つとも変えていない
- 298: `regate_test.go` と `regate_merge_test.go` の次の変更。395: `org_merge_test.go` など 4 ファイルの次の変更。401: `resolvedOrClean` の caller か `statedir.go` の次の編集。403: `gateScriptEnv`、`Regate`、`runGateScripts` の次の変更(`setupOrgStubPATH` を含む(b)も)。410: `regate.go`、`regate_ci.go`、`org_regate_test.go` の次の変更。539: `recordQueueMerge`、`queueMergeRecorded`、`queueMergeCleanup`、`enqueue`、`gateWhen` の次の変更。どれも、この PR が触っていないファイルと関数
- 299: gate の記録の鍵の次の変更。526: leader の rule の gate の項の次の変更。どちらも触っていない
- 474: 持ち主を止める必要があった人、順を外れて走った gate、要らない gate を待った regate。#357 は待った相手の名前の記録の誤りで、どれにも当たらない
- 305、396、429、475、499 は Trigger の列に語が出ない(行のほかの列だけ)。475 の Trigger は「終わった pid を名指し、自分では終わらない待ち」で、当たらない

当たった行はない。verify の結論と同じで、新しい持ち越しの note は足していない。

### 3. 通常の確認(変更なし)

| 場所 | 確かめたこと | 結果 |
|------|-------------|------|
| `AGENTS.md` の `internal/org/` の gate.lock の説明 | 「a wait with no time limit whose stderr lines name the holder, Gate's re-check ..., and the lock's file passed to the scripts」 | 変更なし。持ち主の決め方(最後の読み、空の marker の扱い)は書いていない |
| `README.md`(292 行目ほか) | 「stderr names the holder when the wait starts, when the holder changes, and every 5 minutes」 | 変更なし。行の出る場面は直しのあとも同じ(空の marker の読みで行が 1 本増える場面がなくなっただけ)。log の見出しの持ち主の決め方には触れていない |
| gate.lock の仕様(`docs/specs/2026-10-07-org-multi-org-director.md` の 2026-11-12 の追記) | 「待つ間は stderr に、ロックを持つ gate を出す」 | 変更なし。`docs/specs/` に「最後の poll で読んだ持ち主」と書いた箇所はない(`git grep` で `last poll`、`最後の poll`、`marker is not written` を探し、spec には当たらなかった) |
| `ralph org gate --help`(`internal/cli/org_gate.go`)と `ralph org regate --help` | 「names the holder ... when the wait starts, when the holder changes, and every 5 minutes」、「The log's header says how long gate waited for gate.lock and who held it」 | 変更なし。どちらも直しのあとの動きと合う |
| `docs/quality/quality-gates.md` とその template の写し | gate.lock の 1 文(「wait for it with no time limit」) | 変更なし。持ち主の決め方は書いていない |
| `/director`・`/org` skill、`internal/org/prompts/reviewer.md`、`agent-messaging.md` | verify が読んで「待ちの行の出る場面と `# waited ... for gate.lock` の行を読むことだけを書き、決め方には触れない」とした | 変更なし。skill を変えていないので `sync-skills.sh` は回していない |
| コードの doc comment | verify が `acquireGateLock`、`gateLockWait`、`gateLockHolder`、`afterGateLockRead`、`gateLockNotWritten`、ファイル冒頭の説明を、コードの条件と照らして一致とした | 変更なし(コードは `sync-docs` の担当外) |

verify の INFO 2 件も変更は要らない。I1(2026-11-12 の self-review の「参考」の項目をこの PR が直した)は register の行ではなく、過去の report は書き換えない。I2(`gate_lock_test.go` 643 行、`gate_lock.go` 245 行)は 800 行の目安の内で、この 2 ファイルを名指す register の行もない。

## 変えなかったもの

- Go のコードとテスト。`README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/`、`docs/quality/`、spec、skill、`templates/`
- `docs/reports/` の self-review、verify、test の report(verify の V1 は計画の Progress の直しで、report の verdict の行は動かさない)
- tech-debt の既存の行(Trigger に当たる行がなかった)

## Not verified

- `./scripts/run-test.sh` と `./scripts/run-static-verify.sh` は回していない。この run が変えたのは計画の Progress と箱、tech-debt の 1 行、この report と insights の行だけで、コードに触れていない
- tech-debt の行の Why deferred の「本物の scripts を走らせるテストに、待つ側が名前を読むまで前の gate の scripts を止める仕組みを足すのは Scope の外」は、計画の「この PR では同期させない」からの読みで、計画自身は理由を書いていない
