# sync-docs report: director-wait-types-note

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-director-wait-types-note.md`(承認の digest `574fe7322c7d`。この run の前後とも同じ)。issue #374
- Branch: `fix/director-wait-types-note`。run 開始時の HEAD は `0db2e56e`(origin/main の `64c3bfc5` を取り込んだ merge)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review `b6759cab`(Merge: yes、MEDIUM 2、LOW 6。M1・M2・L1〜L6 は `7e3448b7` で直った)、verify `ea088c24`(pass、HEAD 7e3448b7)、test `2e99c43a`(pass、`run-test.sh` rc 0、変異 8 個が落ち、穴を測る 2 個は落ちない)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。回したのは `check-sync.sh`、`check-skill-sync.sh`、`gofmt`、`go vet ./internal/org/`、狭い Go テスト(下)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 17 ファイル、+1,471/-43。Go は全体で +864/-23、テスト以外の Go が +93/-9。walkthrough を書くかどうかは `/pr` が決める

## Summary

verify の一覧 4 件(V1〜V4)と、test report の Test gap 1 を片づけた。V1 は `/director` skill の打ち直しの文 4 つ(終了コード 0 の項、終了コード 4 の項、Monitor の期限切れの段落の 2 文)に「ほかの wait がまだ走っているあいだは打たない」を足し、4 写しをそろえて、足した文を `directorSkillWaitOneAtATimeSteps` の fragment で固定した。V2 は台帳の行 267・386・324 に日付つきの繰り越しの注を足し、V3 で `/org` skill の `wait` の行に 1 文足したので、行 182 (a) にも注を足した。Test gap 1 は台帳に新しい行 1 つ(569 行目)にした。V4 は計画の Progress に `7e3448b7` の行と `Sync-docs artifact created` の行を足し、AC1〜AC4 に印を付けた。計画の本文は変えていないので digest は `574fe7322c7d` のまま。仕様、README、AGENTS.md、`ralph org wait --help` は verify が確かめたとおりで、直していない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | `### wait で待つ` の 3 か所に条件を足した。(1) Claude Code の箇条の終了コード 0(「件を処理し、また待つ。ほかの wait がまだ走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)」)、(2) 終了コード 4 の項(「wait を打ち直す。ほかの wait がまだ走っているあいだは打ち直さない(…)」)、(3) Monitor の期限切れの段落の 2 文(最初の「wait を打ち直し」の次に「ただし、ほかの wait がまだ走っているあいだは打ち直さない(…)」、あとの「そこで同じように打ち直し」の次に「(ほかの wait がまだ走っているあいだは打たない)」)。`director_skill_sync_test.go` が固定する `あとからその Monitor の知らせが届いたら、そこで同じように打ち直し` はそのまま残した。`.claude` を 2 か所に書き、`.agents` は `sync-skills.sh` で作った(root と template の 2 回) | この report の commit |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | コマンド表の `wait` の行の `--types` の説明の次に 1 文: `--types` 付きの wait が時間切れになるとき、数えなかった種類の open の件があれば、時間切れの文の後ろにその ID と TYPE が付く(5 件まで、残りは数だけ)。行の長さは 1,090 から 1,183 文字(python で数えた) | この report の commit |
| `internal/org/director_skill_wait_types_note_test.go` | `directorSkillWaitOneAtATimeSteps` に fragment を 4 つ足した(23 から 27)。順は skill の順で、終了コード 0 の項、終了コード 4 の項、Monitor の期限切れの 2 文。変数の doc に 4 つの規則を足した。313 行から 323 行。`director_skill_sync_test.go` は触っていない(origin/main と同じ 1,320 行のまま) | この report の commit |
| `docs/tech-debt/README.md` | 行 267・386・324・182 の Trigger のセルの末尾に `2026-11-22: fix/director-wait-types-note (#374) hit ...` の注、Related のセルの末尾に `docs/plans/active/2026-11-22-director-wait-types-note.md`。新しい行 1 つ(569 行目、Test gap 1)。新しい行は大きさの行ではない(依頼のとおり) | この report の commit |
| `docs/plans/active/2026-11-22-director-wait-types-note.md` | Progress の `Implementation started` の下に `7e3448b7` の子の項目、`Test artifact created` の次に `Sync-docs artifact created`。`## Acceptance criteria` の AC1〜AC4 に `[x]`。本文は変えていない。digest は前後とも `574fe7322c7d` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-director-wait-types-note.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-director-wait-types-note.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行。`git status --short` は `M` だけで、迷子のファイルはない | この report の commit |

コード(Go)に触れたのは上のテストファイル 1 つ(fragment と doc)で、skill の文も変えた。どちらも記録だけの直しではないので、`post-implementation-pipeline.md` の例外には当たらない。sync-docs の次は cross-review なので、self-review・verify・test を回し直すかは lead が決める。

## 渡された一覧の行き先

### V1: 打ち直しの文 3 か所

verify は行 1039・1078・1121・1125 を挙げ、行番号は今の HEAD でも同じだった。条件の言い回しは、M1 で入れた 3 か所と同じ「ほかの wait がまだ走っているあいだは打たない」(終了コード 4 の項と、Monitor の最初の文は「打ち直さない」)にそろえた。Monitor の段落の最初の「wait を打ち直し」は文の途中なので、文を足す形にし(「ただし、…打ち直さない」)、「そこで同じように打ち直し」は「同じように」が前の文の条件を引き継ぐが、局所の 1 文で動く director を考えて括弧で明示した。fragment は `directorSkillWaitOneAtATimeSteps` の中で skill の順に置いた(終了コード 0 の項は Claude Code の箇条の時間切れの fragment の前、終了コード 4 の項は Codex の箇条の次、Monitor の 2 つは終了コード 1 の項の次で「wait は 1 本ずつ張る」の前)。

足した文ごとに、skill の部分を空白と改行を外した形にして条件の句を抜く確かめを python で回した(scratch、追跡ファイルは触っていない)。4 つの fragment とも、今の skill にはあり、句を抜いた形にはなかった。固定されたままの `あとからその Monitor の知らせが届いたら、そこで同じように打ち直し` も今の skill にある。これは Go の変異ではない(テストは追跡ファイルを読むので、変異の写しを作る道を取っていない)。

### V2: 台帳

行番号は verify の値(267・386・324)が HEAD でも同じで、最初の語でも見つけた。注の形は「`2026-11-22: fix/director-wait-types-note (#374) hit <変えたもの>, but did not pay it: <理由>. Carried over to the next change that meets this trigger`」で、Related の末尾に計画を `docs/plans/active/...` の path で足した(`/pr` の archive が書き換える)。`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検査を単独で回し、存在しない参照は 0 件。

| 行(最初の語) | 当たった Trigger | この PR が触ったもの | 扱い |
|------|------|------|------|
| 267 `Three test gaps of the org-director-attach PR` | `受信箱の処理` か `WaitInbox` の次の変更 | `### wait で待つ`(4 写し)と、`WaitInbox` の時間切れ(`inboxWaitTimeoutNote`、`readInboxWaitOtherTypesText`) | 繰り越し。`director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は触っていない。(1)の `t.Skipf` は新しいファイルの 2 つのテストに増えたと書いた。(2)(3)は書き足していない |
| 386 `RC9` | wait の loop か終了コード、または skill の終了コードの部分 | `loop` の時間切れの枝、skill の終了コード 0・4・1 の項 | 繰り越し。`director.jsonl` の読み直し、skill の 4 の説明、テストは前のまま |
| 324 `internal/org/director_start.go is 1203 lines` | `director_wait.go` が 800 を超えたままの次の変更 | `director_wait.go` 821→833 行(`64c3bfc5` と `7e3448b7` で数えた)、`director_wait_test.go` 824→829(`64c3bfc5` と `8a258cdc`) | 繰り越し。新しい大きさの行は起こさない(依頼どおり)。1 つ目のセルの「799 と 687」は 833 と 829 になったと書いた。新しいテストは別ファイル(434 行と 323 行)で、どちらも 800 未満 |
| 182 `Two documentation findings of org-stop-all` | (a) `/org` skill のコマンド表の次の編集 | `wait` の行(1,090→1,183 文字、4 写し) | 繰り越し。`stop`・`disband` の行は触っておらず(1,208 と 1,219 文字)、`wait` の行は今も両方より短いと書いた。**自分の編集で当たった行**(V3 を直したため) |

### V3: `/org` の `wait` の行

直した(上の表の 182)。足した 1 文は help の「A wait with --types that times out names the open items of other TYPEs it did not count (their IDs and TYPEs) after its timeout text」と、5 件で切る決まり(`inboxWaitOtherTypesShown` の 5、残りは `and <n> more`)に合わせた。`--director` の印の `detail` のことは書いていない(`--json` の `wait.detail` で `/director` skill が言う。`/org` の行は leader と人向けで、help の同じ 1 文の後半)。この行を固定するテストはない(`ack の期限に見張りが人へ送る` を引く Go のテスト、script は 0 件)。

### V4: 計画の Progress

`7e3448b7` の子の項目を `Implementation started` の S2 の次に足した。改名は `inboxWaitOtherTypes` を `readInboxWaitOtherTypesText` に、`TestDirectorSkillStopsAWaitRe` を `TestDirectorSkillStopsAWait` に。ピンの移動(`director_skill_sync_test.go` から `director_skill_wait_types_note_test.go`)と M1・L1 の skill の直しも書いた。S1 の行の古い名前は、verify の言うとおり記録として残し、子の項目の末尾で「改名の前の名前」と言った。self-review の report は 61dd8ced の記録なので書き換えていない。`Sync-docs artifact created` の行は `Test artifact created` の次に足した(この計画の Progress にはその行がなかった)。AC1〜AC4 は verify の表(met)と test の report(Pass、重なりのテストは `-race -count=5` で 10/10)で確かめられているので、`[x]` にした。AC3 の V1 は今回の直しで閉じた。

### test report の Test gap 1

この branch の台帳には行がなかったので、短い新しい行を 1 つ足した(569 行目)。`directorSkillStopsAWait` の字句の検査が言い換え(P1: `古いほうの Monitor に TaskStop をかけて終える。`、P2: `古いほうは TaskStop で片づける。`)を見つけないこと、LOW、今の skill にそう書いた文はないこと、Trigger は重なりの段落の次の変更か、止める指示が review をすり抜けたとき、を書いた。Test gap 2〜4 は台帳に書いていない(依頼は 1 だけ。下の Known gaps)。

## 掃き出し

verify が挙げたのは行 267・386・324・327(INFO)で、当たらないと言ったのは行 395・274 (b)・426・427・261・262・182(V3 を直す前)。私の掃き出しは Trigger のセル(3 番目、取り消し線の文を除く)を割って、この branch の変更に当たる語で探した。1 回目は語を広くして(`skill`、`help`、`loop` を含めて)125 行が当たり、2 回目は固有の名前だけにして(`escalate.go`、`director_wait`、`WaitInbox`、`openInboxItems`、`inboxWaitTimeoutPhrase`、`wait で待つ`、`受信箱の処理`、`--types`、`ralph org wait`、`internal/cli/org.go`、`org_inbox_test`、`directorWaitTimeoutDetail`、`director_skill_sync_test`、`SKILL.md`、`/director` skill)、最初の日付つきの注より前の本文で 26 行になった。Debt のセルだけが変更したファイルを挙げる行(Trigger が「any of」「these files」の形)も別に探し、2 行(181、395)だった。

新しく見つかった行はない(267・386・324 は verify が挙げた行、182 は自分の編集で当たった行)。当たらないと判断した行は次の表のとおり。

| 行 | 理由 |
|------|------|
| 327 `Untested sentence paths of the org-director-wait PR` | Trigger が名指す関数は `directorWaitItemsDetail`、`probeAgain`、`directorWaitWatchFacts`、`directorWaitWatchState`。`director_wait.go` の diff の 3 つ目の hunk の見出しは `func directorWaitItemsDetail` だが、見出しは hunk の前の関数を指す。hunk が書き換えるのは `directorWaitTimeoutDetail` の doc と本体で、4 つの関数はどれも変わっていない。verify も INFO |
| 395 `Four test files went past the 800-line guide` と 274 (b) | `director_skill_sync_test.go` の `origin/main` との差分は空(b0ff053d が足した 215 行を `7e3448b7` が新しいファイルへ移した)。固定は新しいファイルに入れた。行 274 (b) の Trigger は「`director_skill_sync_test.go` に足す次のテスト」で、足していない |
| 181、197、203、432、194、88、150 | `internal/cli/org.go` や `--help` を挙げる Trigger は、`Stop`・`Disband`(181)、`newOrgStartCmd`(197)、`runOrgStartPlan`(203)、`ralph org status` の JSON(432)と help(194)、`status.go`(88)、`resolveOrgConfig`・`withMainWorktreeOrgLimits`・`MainWorktreeRoot`・`ResolveOrgStateDir`(150)を名指す。この PR が `org.go` で変えたのは `ralph org wait` の `--types` の説明 1 文だけで、どれも変えていない |
| 213、464、508、220 | `TestPrintableInboxText`、`PrintableInboxText`、`escalate_test.go` の Trigger。`escalate_test.go` は origin/main と同じ 1,745 行で、`PrintableInboxText` は呼ぶだけで編集していない |
| 362 | Trigger は watch の escalation(`watch_escalate.go` へ移す)で、`escalate.go`(`WaitInbox` の側)は別 |
| 201・247・248・249・251・252・254・256・257・264・385・426・470・512・521・537・561・562・566 | `/director` skill、8a-1、8b-2b などを挙げるが、Trigger の中身は別の節か別の関数(merge-order、`Attach`、送ったあとの確かめ、`skill_split_questions_test.go`、struck の行、`merge を待つ issue` など)。この PR は `### wait で待つ` と終了コードの部分だけを変えた |

## 確かめたこと

| 確かめ | 結果 |
|------|------|
| `./scripts/check-sync.sh` | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、PASS |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `cmp`(director と org の skill、`.claude` 4 つと `.agents` 4 つ) | 4 写しが同じ(`.claude` の 2 つ、`.agents` の 2 つ、`.claude` と `.agents`) |
| `test -z "$(gofmt -l internal)"`、`go vet ./internal/org/` | どちらも問題なし |
| `go test -count=1 ./internal/org/ -run 'DirectorSkill\|Skill\|DocsMatch\|WaitInbox\|DirectorWait'` | ok 10.825 s |
| `go test -count=1 ./internal/cli/ -run 'OrgWait\|Help'` | ok 9.423 s |
| `go test -count=1 -v` で `TestDirectorSkill_RunsOneInboxWaitAtATime`、`..._ReadsTheUncountedItemsOfATypesTimeout`、`..._PinsTheClaudeCodeWaitRules`、`..._NamesTheInboxWaitTimeoutPhrase`、`TestDirectorSkillStopsAWait` | 5 つとも PASS(SKIP は出ていない) |
| 台帳の編集の確かめ(scratch の python、`git show HEAD:` と比べた) | 変わった行は 182・267・324・386 の 4 行と、追加の 1 行(569)。5 つのセル、セルごとのバッククォートは偶数、`~~` は偶数。4 行とも古いセルは新しいセルの前の部分(Related の末尾の ` |` だけ、足した path の前に動いた)。`file:line` の新しい参照は 0 件 |
| `docs/plans/active/` と `docs/plans/archive/` の参照 | 存在しない参照 0 件(`verify.local.sh` の検査と同じ式を単独で) |
| 計画の digest | `./scripts/plan-visual.sh digest` が前後とも `574fe7322c7d` |

回していないもの: `./scripts/run-test.sh`(依頼の指示)、`./scripts/run-static-verify.sh` の全体、`go test ./...`。skill の文を変えたのでピンのテストは狭く回した。`internal/org` と `internal/cli` の全体は回していない。

## 直さなかったもの

- V5(INFO、仕様の項目の置き場所)と V6(INFO、help と skill の `--director` の detail の文は条件を言わない)は、verify が「なし」としたので、そのまま。
- V7(AC の checkbox)は上のとおり印を付けた。V8(Progress の「変異 10 個」「28 個」の出力が repo にない)は /test への項目で、test report が変異を別に回している(8 個が落ち、穴を測る 2 個は落ちない)。implementer の 38 個の出力は repo にないままで、Progress の文は直していない。
- 行 327 の注(INFO、任意)は、Trigger の 4 つの関数が変わっていないので足していない。
- self-review の report、verify の report、test の report は書き換えていない。
- 行 324 の「`director_wait_test.go` の 824 行」の注は、`director_wait.go` と同じセルの注の中で数を言った。大きさの新しい行は起こしていない。

## PR 本文の Known gaps に載せるもの

1. 台帳の Test gap 1(行 569): 停止の指示の字句の検査は言い換えを見つけない(P1、P2)。今の skill にその文はない。LOW
2. test の Test gap 2: 句のための読みは wait の最後の読みのあとにもう 1 回読むので、そのあいだに外の件が resolve されると、句がないか resolve された件が載るかのどちらかになる。時間切れの扱いは変わらない。テストは手を入れる場所がなく、確かめていない
3. test の Test gap 3: ビルドした binary での `--director` の時間切れは回していない(CLI の層の probe で確かめた)。verify も binary で `--director` の時間切れを打っていない
4. issue #374 の原因(別の repo の台帳の e144・e145 の events と、director の Monitor のコマンド行の `--types`)は確かめていない。PR は `Refs #374`。ユーザーに `ralph org inbox show e144` の notified の reason(`inbox_ack_deadline` なら `--types` の道、すぐ人へなら見張りが受け付けていなかった道)と Monitor のコマンド行の `--types` を見てもらう(計画の Rollout)
5. この sync-docs の commit は skill の文(director と org、4 写しずつ)と Go のテスト(fragment と doc)を変えている。足した文の固定は python で確かめ、狭い Go テストは通ったが、verify と test は 7e3448b7 の木で回したもので、この commit の skill の文とテストは見ていない
6. `/org` の `wait` の行を固定するテストはない(足した 1 文は help と手で合わせた)

## Insight event

`./scripts/insights-append.sh --slug director-wait-types-note --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行(cycle 1、pass)を足した。`docs/insights/events/2026-11-22-director-wait-types-note.jsonl` は self_review、verify、test の 3 行があり、sync_docs で 4 行になった。`--slug` に日付は付けていない。
