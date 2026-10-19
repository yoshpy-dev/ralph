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
