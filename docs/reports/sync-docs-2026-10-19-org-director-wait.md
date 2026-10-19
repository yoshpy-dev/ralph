# sync-docs report: org-director-wait

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait.md`(8b-2b。承認の digest `7d6739ec79ce` はこの run の前後で同じ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `e38b89a9`、branch は `feat/org-director-wait`。run 開始時の HEAD は `b7acfbad`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-19-org-director-wait.md`、`docs/reports/verify-2026-10-19-org-director-wait.md`(pass。LOW 2、INFO 4)、`docs/reports/test-2026-10-19-org-director-wait.md`(pass。LOW 3、INFO 4 の穴)。実機の確認は `docs/evidence/director-wait-livefire-2026-10-19.md`、実験は `docs/evidence/director-wait-session-end-2026-10-19.md`
- この run が変えたのは、README の 1 文、recipe `director-wake.md` の 2 か所(root と template の写しで同じ)、tech-debt の 3 行(253、279 行を書き足し、308 行を足した)、計画の Progress checklist の 1 項目への注、この report。計画の本文、先行 3 report、insight event、コードには触っていない

## Summary

`e38b89a9...b7acfbad` の差分(48 file)のうち文書にあたる面を、実装と、この worktree でビルドした binary の help に突き合わせた。突き合わせた面は、README の動詞の表と `ralph status` の行、director の 2 つの段落、AGENTS.md の `internal/org/` の行、仕様 8b-2b の注(90 行)、recipe 2 つとその写し、`/director` と `/org` の skill の 8 面、雛形 `internal/org/prompts/director.md`、help(`org wait`、`org director status`)、tech-debt の 9 行、`docs/architecture/repo-map.md`、`docs/quality/`、`cmd/ralph/main.go` の終了コードのコメント。

ずれは 2 点で、どちらも先行 report が渡してきたものだった。直した。

1. README と recipe が、pull の director の wait を `ralph org wait --inbox` と書いていた(verify V5)。この段で skill と雛形は `--director <id>` を付けるようになり、README の 258 行は付けた形を書いている。director が打つコマンドを言う 3 か所(README の `[org.director].wake` の段落、recipe `director-wake.md` の冒頭と「When to stay on pull」)を、付けた形に直した。
2. tech-debt の 253 行は、きっかけ(`/director` skill の受信箱の処理か `WaitInbox` の次の変更)をこの段が満たしたのに、持ち越したと書いていなかった(verify V1)。279 行も、きっかけ(次にそのテストが落ちたとき)が `/test` の run で起きていたが、行は 1 回目の記録のままだった。2 行に書き足した。

そのほか、test report の残りの穴(G1〜G5、G7)を 1 行にまとめて 308 行に足した。

## Changes made

| File | Change |
|------|--------|
| `README.md`(`[org.director].wake` の段落) | 「`["pull"]` (the default) has it wait with `ralph org wait --inbox`」を `--director <id>` を付けた形にした。動詞の表(124 行)、`ralph status` の行(125 行)、director の段落(258 行)は元から `--director` を書いていて、変えていない |
| `docs/recipes/director-wake.md`(冒頭の段落、「When to stay on pull」の段落) | pull の director の wait を `ralph org wait --inbox --director <id>` にした。2 文とも折り返しを 1 行足して 72 桁に収めた。289 行の「In its first turn codex did not start `ralph org wait --inbox`」は、古い skill が言っていたコマンドと、その実機の観察を書いたものなので変えていない |
| `templates/base/docs/recipes/director-wake.md` | 上の recipe と同じ 2 か所。`cmp` で root と同一バイト |
| `docs/tech-debt/README.md`(253 行) | きっかけの欄に、8b-2b がそのきっかけを満たして (1)〜(3) を払わず持ち越したことを足した。持ち越した理由は、受信箱の処理の wait の段落を書き換えて(Codex の打ち直しの箇条が `--director d1` を付けるようになった)`WaitInbox` の読みを `openInboxItems` に出したが、skill のテストには触れず、この計画の範囲が director の wait の印と見張りの停止だったこと。director の wait の時間切れの文は `TestOrgWaitDirector_TimesOutWithThePhrase` と `TestWaitInboxForDirector_TimesOutWithThePhraseAfterANote` が固定するが、`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は `WaitInbox` の誤りだけを読むこと、(2) は今も成り立つこと(4 面とも 1 回)も書いた。きっかけは残した。Related に計画と verify report(V1)を足した |
| `docs/tech-debt/README.md`(279 行) | 項目の欄に 2 回目の記録(org-director-wait の `/test` の `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -v` で `process=ended after SIGTERM`。直後の load average は 1 分で 36、5 分で 45。単独で 10 回中 10 回 PASS、`internal/org` の再実行も PASS)を足した。先送りの理由の欄に、この計画も `director_cmd_test.go` と `director_stop*.go` を変えないので、2 回目は持ち越したことを足した。きっかけの欄は元の文のままで、「2 回目は持ち越し済み」を括弧で足した。Related に計画、test report の Failure analysis、evidence を足した |
| `docs/tech-debt/README.md`(308 行、新規) | test report の残りの穴を 1 行にまとめた。G1(O02、O35: 書いたあとに印が消えた wait。`save` と `sayLost` の印がない道)、G2(O31: `director-wait.lock` を取れない `begin`)、G3(O28: 開始の settle の途中で ctx が終わったときに `note:` を出さないこと)、G4(O32、INFO: `end` の `detail` の escape)、test finding 2 = G5(INFO: help が言う 5 秒の `directorWaitWatchProbeInterval` を固定するテストがない)、test finding 3 = G7(INFO: `reraiseSignal` のコメントが要る条件より強い。C07 は等価)。G1〜G4 は証拠として evidence の probe P1〜P4(HEAD で `-count=10` と `-race -count=3` が PASS)を指す。きっかけは「`director_wait.go` の begin、save、持ち主の次の変更、または wait の CLI の次の変更」。参照は関数名とテスト名で書き、file:line は使っていない。欄は 5、`\|` なし |
| `docs/plans/active/2026-10-19-org-director-wait.md`(Progress checklist) | 「Implementation started」の下に、この run の直しの注を 1 項目足した。本文(Progress checklist より上)は変えていない。digest は前後で `7d6739ec79ce` |
| `docs/reports/sync-docs-2026-10-19-org-director-wait.md` | 新規 |

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `README.md`(動詞の表 124 行、`ralph status` の行 125 行) | 合っている | 124 行は `wait --inbox --director <id>` の書く印、終了コード 4 と 5 を言う。help の文と同じ。125 行の `wait:`、`director.wait` の言及は `status` の help に合う |
| `README.md`(director の段落 258 行) | 合っている | 「登録はセッションが動いていることを表さない」のあとに印の話が続き、印は wait が書いたことでありセッションの生死ではないと言う。4 のとき director が事実と `watch --all --detach` を人に伝え、自分では起動し直さないこと、5 のとき director として動くのをやめることは、skill と雛形に合う |
| `README.md`(`Exit codes:` の 146 行) | 変更不要 | `ralph upgrade` の節の終了コード(0、1、3)で、`org wait --director` の 4 と 5 は別の動詞。`cmd/ralph/main.go` のコメントが 3、4、5 を分けて書いている |
| `AGENTS.md`(`internal/org/` の行 90 行) | 合っている | 「the director's wait (`director_wait.go`...)」の句は、印の field、`director-wait.lock`、表示、終了コード 4 と 5 が実装に合う |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(8b-2b の注 90 行) | 合っている | D1〜D7 の決めたことと、実装の「30 秒ごとの読み直しの前に登録を読む」(verify V3 の言い方)が合う。93 行と 213 行の `ralph org wait --inbox`(FR-5 の要件の文と codex の pull の注)は仕様の要件の書き方なので変えていない |
| recipe `director-wake.md` と template の写し | 1 点直した | 上の Changes made。`cmp` は一致。`director-launch-cmd.md` は 188 行に「the director waits on the inbox itself」とだけあり、コマンドを書かない |
| `/director` skill(4 面)、`/org` skill(4 面) | 合っている | `cmp` で 3 つの写しとも同一バイト。director の「受信箱の処理」の wait の段落は `--director d1`、終了コード 4、5、2 種の `note:`、古い ralph の 1 を書き分ける。Codex の箇条は `--director d1 --timeout-ms 300000`。「してはいけないこと」は昇格の手順 2 の起動と、director が自分で見張りを起動し直すことを分けて禁じる |
| `internal/org/prompts/director.md` | 合っている | wait に `--director {{ID}} {{ORG_FLAGS}}` と終了コード 5 の扱い(止まる、detach は打たない)。`prompts_test.go` が描画を固定する |
| help(binary を scratch にビルドして読んだもの) | 合っている | `org wait --help` は `--director` の節に 30 秒の読み直し、30 秒ごとの印、5 秒の見張りの確かめ、終了コード 4 と 5、`note:`、signal と nohup を書き、README と skill の数字に合う。`org director status --help` は `wait:` の行(60 秒と 30 秒、`end` の 6 値)と `--json` の 8 キーを書く。`org director --help`(グループ)に wait の記述はなく、「Nor does it mean a promoted session is still running; the cross-org watch checks a headless director each cycle (see status)」は今も誤りではない(下の Left open)。`ralph status --help` は `wait:` の行と `director.wait` を書く(README と合う) |
| `cmd/ralph/main.go` のコメント | 合っている | 3(upgrade の drift)、4(`ErrDirectorWaitNoWatch`)、5(`ErrDirectorWaitNotCurrent`)、ほかは 1 |
| tech-debt(209、231 の (j)、247、248、250、306、307 行) | 合っている | 306 行の数(743、683、590、342、505、1203、857)は `wc -l` と合う。247 と 248 行は、この段で入った印と残り(B、8b-2c)を書き分ける。ほかは verify report が確かめた内容から変わっていない |
| tech-debt(253、279、308 行) | 直した(上の Changes made) | 3 行とも欄は 5、`\|` は行頭の区切りを除いて 6 個、U+FFFD なし。参照する path は全部実在する。`docs/plans/active/...` の参照も `verify.local.sh` の `check_tech_debt_plan_refs` と同じ検査で欠けなし |
| `docs/architecture/repo-map.md`(37、48 行) | 変更不要 | recipe の一覧と `.claude/skills/director/` の行は、wait の印を書かない粒度。新しい file は `internal/` の下だけ |
| `docs/quality/quality-gates.md`(80 行の watch gate)、`definition-of-done.md` | 変更不要 | `ralph org wait --director` は watch gate が断る動詞に入らない(見張りがなければ `note:` を出して待つ)。新しい門は増えていない |
| evidence(`director-wait-livefire-2026-10-19.md`) | 合っている | verify V2(テストの path)と V6(あとの Go の変更の言い方)は 45f5d586 までに直っている。冒頭と「Not checked here」を読んで確かめた |
| 計画の Progress checklist | 合っている | verify V4(Review artifact created)は `[x]` になっている。D1 の言い方と実装の差(V3)は、checklist に既に書いてある |
| `$HOME` の絶対パス、U+FFFD、`git diff --check` | なし | 追加行に `/Users/` はない。`git diff --check` は終了コード 0 |

## Plan drift notes

計画の本文(Progress checklist より上)は承認の digest の下にあるので、書き換えていない。Progress checklist には、この run の直しの注を 1 項目足した。次を orchestrator が見る材料として残す。

- 計画の D1 は「印を書き直すたびに登録を読み直し」と書くが、実装は 30 秒ごとの読み直し(`refresh`)だけである(verify V3)。help、skill、仕様の注、AC4 はこの動きで書いてあり、checklist にも書いてある。本文は直していない
- 計画の D2 の `directorWaitMarkInterval` は、実装では export した `DirectorWaitMarkInterval` 1 つになった(checklist の self-review の項目に書いてある)
- 計画の Verify plan の「Documentation drift to check」は、README と recipe の pull の wait の言い方を挙げていない。この run で直した
- tech-debt の行番号は、この段で次のように動いた。209、247、250 は書き換え、248 と 306、307 は新規、253 と 279 は書き足し、308 は新規。計画の本文の「247 行」「209 行」は承認時の番号である

## Left open

- test report の G6(INFO)は 308 行に書いていない。通らない block のうち、2 件以上の open の件の `detail`(`<n> open inbox items`)、回の settle の途中の ctx の終わり、running・stopping・stopped 以外の見張りの印、最後の受け皿の文で、どれも文か、すぐ終わる道である。orchestrator の渡した一覧に入っていなかったので足していない。足すかは orchestrator が決める
- `ralph org director --help`(グループ)の段落は、昇格したセッションについて「動いていることを表さない」までを言い、`status` が出す wait の印には触れていない。`status` と `ralph status` の help、README、skill は印を書いていて、グループの help だけが 8b-2a の言い方のままである。誤りではないので変えていない(変えるには Go の help の文とその pin `TestOrgDirectorHelp_DescribesTheRegistration` の変更が要り、この run の範囲を超える)。足すかは次に director のグループの help を触るときに決める
- 実機で打っていない面(verify と test の report のとおり): Monitor で打った wait、codex の繰り返しで打った wait、open の件が 4 より先になる場合(テストだけ)、nohup で始めた wait(テストだけ)、Linux での signal の継承。文書はこれらを実装とテストから書いていて、実機の出力には合わせていない
- 8b-2c(wait の持ち主の終わりで止まること)は未着手で、tech-debt の 248 行が記録している
- 279 行の揺れは、この plan の範囲の外なので直していない。きっかけは 3 回目に落ちたときか `director_cmd_test.go` の次の変更のまま
- release のノート(新しい skill には新しい binary が要る)は計画の checklist のとおり PR の本文と memory で扱う。この run では書いていない

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていないので `sync-skills.sh` は走らせていない |
| `./scripts/check-sync.sh` | 終了コード 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `cmp docs/recipes/director-wake.md templates/base/docs/recipes/director-wake.md` | 同一バイト |
| `cmp` の `director` と `org` の skill(`.claude` と、`.agents`・`templates/base/.claude`・`templates/base/.agents`。2 skill で 6 組) | どれも同一バイト |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-19-org-director-wait.md` | 編集の前もあとも `7d6739ec79ce`(計画の `- Approved:` の行と同じ) |
| `git diff --check` | 出力なし、終了コード 0 |
| tech-debt の 253、279、308 行の欄の数 | 5(`\|` は 6 個)。U+FFFD なし。row 277 の `\|` が 10 個なのは HEAD にある元の行で、この run の変更ではない |
| tech-debt の plan 参照(`check_tech_debt_plan_refs` と同じ grep と `-e`) | 欠けなし |
| 253、279、308 行が名指す path の実在 | すべて実在 |
| binary を scratch にビルドして `org wait --help`、`org director status --help`、`ralph status --help` を読む | README、skill、仕様の数字と合う(上の表) |

`./scripts/verify.local.sh`、`./scripts/run-static-verify.sh`、`go test` は走らせていない。この run はコードを変えておらず、変えたのは markdown 4 file と、その tech-debt の plan 参照の検査を手で同じ grep で打った分だけである。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 上の Left open の実機の面、Linux、`go test` と `run-static-verify.sh`(コードの変更なし)

## Insight event

この run では追加していない(orchestrator が追加する)。

## Cycle 2

- Date: 2026-10-19
- Pipeline cycle: 2。範囲は `8a33364b..28eec963`(28eec963 は run 開始時の HEAD で、この report を含む commit の SHA ではない)。base は `e38b89a9`
- 渡された指摘: verify cycle 2 の C2-1(LOW、実機の evidence の冒頭)と C2-3(INFO、仕様の注)、test cycle 2 の G8〜G11(tech-debt 308 行)、cycle 2 が起こしうる文書のずれの確認
- cycle 2 のコードの変更(`git diff 8a33364b..HEAD`): 38cac6f1(`begin` がロックの下で `director.jsonl` を読み直す)、b3b692e1 と 99144e8f(テスト、`DirectorWaitWatchProbeInterval` の export、`reraiseSignal` のコメント)、6086af65(`begin` を `decideStart` と `noteStartWatch` に分けた)。`.claude`、`.agents`、`templates`、`README.md`、`AGENTS.md`、`docs/recipes`、`docs/quality`、`docs/architecture` は、この差分に入っていない
- この run が変えたのは、実機の evidence の冒頭、仕様の 8b-2b の注の 1 句、tech-debt の 306 行と 308 行、計画の Progress checklist の 1 項目、この report、insight event。コード、計画の本文、先行の report(cycle 1 の sync-docs と、cycle 2 の self-review、verify、test)には触っていない

### Summary

ずれは 4 点で、どれも記録の文だった。直した。help、`/director` skill(4 面)、README、recipe(2 面)、AGENTS.md、`agent-messaging.md` は cycle 2 の動きと合っていて、変えていない。

1. 実機の evidence の冒頭は、実機に使った fa95a3d8 より後のコミットは wait の動きを変えないと書いていた。38cac6f1 は、最初の確かめとロックのあいだに登録が変わった wait を、印を書かずに 5 で返すようにしたので、その文が成り立たなくなった(verify C2-1)。
2. 仕様 8b-2b の注は、始める前の確かめしか言わず、最初の印を書く前のロックの下の読み直しに触れていなかった(verify C2-3)。
3. tech-debt の 308 行は、cycle 2 の `/test` が出した G8〜G11 を 99144e8f が払ったことを言っていなかった。
4. tech-debt の 306 行の行数が 99144e8f で古くなっていた(`director_wait_watch_test.go` 630 が 640 に、`director_wait_owner_test.go` 235 が 388 に)。

### Changes made

| File | Change |
|------|--------|
| `docs/evidence/director-wait-livefire-2026-10-19.md`(冒頭) | 第 1 段落を「The later commits, except 38cac6f1 below, ...」に直し、段落を 1 つ足した。足した段落は事実だけを言う: 38cac6f1 は 1 つの道で wait の動きを変えた(`begin` がロックの下で `director.jsonl` を読み直し、今の登録でなければ印を書かずに 5 で返る。前は、その wait が印を書き、あとの読み直しが 5 で終わらせるまで待った)。(a)〜(h) はこの道を通らない((e) の 5 は始まった wait の 30 秒の読み直しから、断られた wait は最初の確かめで落ちた)。38cac6f1 以降の binary で打ち直していない。そのあとの 3 コミット(b3b692e1、6086af65、99144e8f)は動きを変えない。実機の出力、表、script は書き換えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(8b-2b の注、90 行) | 「始める前にそうでなければ印を書かずに終了コード 5 で返り」のあとに 1 句「(最初の印を書く前にも `director-wait.lock` の下で読み直し、そのあいだに登録が変わっていても同じ)」を足した。注は 5 を返して印を書かない場面を決めたこととして書く場所で、その場面が 1 つ増えたため。同じ id で `attached_at` が変わった場合、読めない場合、lock の file が残りうることは計画の D2 に書いてあるので、注には足していない |
| `docs/tech-debt/README.md`(308 行) | 払った穴の一覧に 1 文足した: cycle 2 の `/test`(mutant 30 個、生き残り 7 個)が G8〜G11 を出し、99144e8f が probe P5〜P8 をテストにした。G8 は `TestWaitInboxForDirector_ASameIDAttachedAgainBeforeTheStartTakesTheLock`、G9 は `..._AStartWhoseWriteFailsSaysSoAtOnce`、G10 は `..._TheStartsMarkHasSeenAt`、G11 は `TestDirectorWait_ZeroIntervalsTakeTheConstants`。Related の `(Test gaps G1 to G7)` に `; Cycle 2, Test gaps G8 to G11` を足した。残る項目は G6 だけで、セルは 5、`\|` なし |
| `docs/tech-debt/README.md`(306 行) | cycle 2 の文の行数を `wc -l` に合わせた(`director_wait_watch_test.go` 640、`director_wait_owner_test.go` 388)。「the tests of b3b692e1」を「the tests of b3b692e1 and 99144e8f」にした。ほかの数(`director_wait.go` 799、`director_wait_test.go` 687、1203、857、804、852、782、342、505)は今の行数と合う。セルは 5 |
| `docs/plans/active/2026-10-19-org-director-wait.md`(Progress checklist) | 「Implementation started」の下に、この run の直しの注を 1 項目足した。本文(Progress checklist より上)は変えていない。digest は `e67985a2f2d7` のまま |
| `docs/insights/events/2026-10-19-org-director-wait.jsonl` | `sync_docs`、cycle 2、`complete` の 1 行を `insights-append.sh` で足した |
| `docs/reports/sync-docs-2026-10-19-org-director-wait.md` | この節を足した |

### Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `ralph org wait --help`(`go run ./cmd/ralph org wait --help`) | 合っている | 「It waits only while <id> is the current registration and not stopped: otherwise it exits 5 without waiting」は、ロックの下の読み直しで断られる場合にも当たる(待たずに 5 で返る)。30 s の読み直しと印、5 s の見張りの確かめ、4 と 5、`note:`、signal の文は cycle 1 のまま。30 s と 5 s は `TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer` が `org.DirectorWaitMarkInterval` と `org.DirectorWaitWatchProbeInterval` に結ぶ |
| 終了コード(0、1、4、5) | 合っている | `internal/cli/exitcode.go` は 3(upgrade)、4(`ErrDirectorWaitNoWatch`)、5(`ErrDirectorWaitNotCurrent`)、ほかは 1 のまま。cycle 2 は終了コードを足していない。ロックの下で `director.jsonl` を読めないときは prefix 付きの plain error で 1 になり、skill の「0、4、5 と時間切れのほかは失敗」に入る |
| 旧名 `directorWaitWatchProbeInterval`、`directorWaitMarkInterval` | 残っていない | `git grep`(`docs/reports` と `docs/evidence` を除く)で当たるのは、計画の Progress の 169 行(D2 の名前が変わった記録)だけ。コード、help、計画の D2 と D3、tech-debt にはない |
| `/director` skill(`.claude`、`.agents`、`templates/base/.claude`、`templates/base/.agents`) | 合っている | 4 面は `cmp` で同一バイト。「始める前にそうでなければ待たずに終了コード 5 で返る」は新しい道にも当たる(待たずに 5 で、director として動くのをやめる扱いは同じ)。30 秒の読み直し、5 秒の見張りの確かめ、0、4、5 の扱いは実装と合う。変えていない |
| `README.md`(124、258、260 行) | 合っている | `wait --inbox --director <id>`、「waits only while `<id>` is the current registration」、4 と 5、印の説明は cycle 2 の動きと食い違わない |
| `docs/recipes/director-wake.md` と template の写し | 合っている | `cmp` で同一バイト。7 行と 271 行は `--director <id>` を書く |
| `AGENTS.md`(`internal/org/` の行) | 合っている | 「ends with exit 4 ... and 5 when its registration is not current」は実装に合う。file 名 `director_wait.go` は変わらない |
| `.claude/rules/ralph/agent-messaging.md` | 変更不要 | `wait` の記述がない(TYPE の表、星形の topology、2,000 字の上限、escalate の話) |
| `docs/architecture/repo-map.md`、`docs/quality/` | 変更不要 | cycle 2 の差分に入っていない。新しい file も script も門も足していない |
| 実機の evidence | 直した | 上の Changes made |
| 仕様 8b-2b の注 | 直した | 上の Changes made。ほかの文(印の field、`end` の 6 値、5 秒と 30 秒、終了コード 4 と 5)は実装と合う |
| tech-debt(306、308 行) | 直した | 上の Changes made。plan の参照は `docs/plans/active/2026-10-19-org-director-wait.md` で、実在する |
| `$HOME` の絶対パス、U+FFFD、`git diff --check` | なし | 変えた 4 file に `/Users/` も U+FFFD もない。`git diff --check` は終了コード 0 |

### Left open

- test report の cycle 2 の節の「残る穴: G8、G9、G10(LOW)、G11、G6(INFO)」は、99144e8f より前の記録である。`/test` が verdict を持つ report なので書き換えていない。払ったことは計画の Progress と tech-debt の 308 行にある
- 実機の確認は fa95a3d8 の binary のままで、38cac6f1 以降の binary で (a)〜(h) を打ち直していない。evidence の冒頭にそう書いた。打ち直すかは orchestrator が決める(変わったのは、(a)〜(h) が通らない道だけである)
- self-review cycle 2 の I2(ロックを取れない開始と `attach --replace` が重なる場合)は、self-review が上げないと決めたとおり、tech-debt にも書いていない。`begin` のコメントは、ロックを取れないときは最初の `refresh` が登録を読み直すと言っている
- `ralph org director --help`(グループ)の段落が 8a-2 の言い方のままであることは、cycle 1 の report のとおり。cycle 2 の差分は触れていない
- 実機で打っていない面(Monitor で打った wait、codex の繰り返しで打った wait、nohup で始めた wait、Linux での signal の継承)は、cycle 1 の report と verify と test の report のとおり

### Verification

| Command | Result |
|---------|--------|
| `go run ./cmd/ralph org wait --help` | 上の表のとおり、help の文が cycle 2 の動きと合う |
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていないので `sync-skills.sh` は走らせていない |
| `./scripts/check-sync.sh` | 終了コード 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `cmp` の `director` skill の 4 面と recipe の 2 面 | どれも同一バイト |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-19-org-director-wait.md` | 編集のあとも `e67985a2f2d7`(計画の `- Approved:` の行と同じ) |
| tech-debt の 306、308 行の欄の数と plan の参照 | セルは 5(`\|` は行頭と行末を含めて 6 個)。U+FFFD なし。参照する `docs/` と `internal/` の path は実在し、`docs/plans/` の参照に欠けなし |
| `git diff --check` | 出力なし、終了コード 0 |

`./scripts/verify.local.sh`、`./scripts/run-static-verify.sh`、`go test` は走らせていない。この run はコードを変えておらず、変えたのは markdown 5 file と insight event だけである。

### Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 38cac6f1 以降の binary での実機の確認、上の Left open の実機の面、Linux、`go test` と `run-static-verify.sh`(コードの変更なし)

### Insight event

`./scripts/insights-append.sh --slug org-director-wait --flow standard --phase sync_docs --verdict complete --source skill --cycle 2` で `docs/insights/events/2026-10-19-org-director-wait.jsonl` に 1 行足した(`ts` 2026-10-19T10:11:48Z)。
