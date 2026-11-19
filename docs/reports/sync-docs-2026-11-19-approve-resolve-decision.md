# sync-docs report: approve-resolve-decision

- Date: 2026-11-20(JST。UTC では 2026-11-19)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)。承認の digest `37ad5d8a83d5` は、`./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない(AC の箱 10 個を `[x]` にしただけ)
- Branch: `fix/approve-resolve-decision`。run 開始時の HEAD は `223c82f0`(origin/main `32274500` を `a8e0e72f` で取り込み済み。origin/main は今も `32274500`)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は `d35730a7`(2 回目。LOW 4 件、L4 と org skill の表と仕様の追記が `/sync-docs` 行き)、verify は `b7a9552e`(2 回目。pass。LOW 5 件 L1〜L5 が `/sync-docs` 行き)、test は `e42ee962`(pass。G1〜G4 は `4c4edcd7` でテストを足して閉じた)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この pass の編集と report を足す前): `git diff --shortstat origin/main...HEAD` は 28 ファイル、+2,318/-54。Go は 11 ファイルで +1,658/-26。そのうちテストでない Go は 7 ファイルで +402/-19
- 制約: Go のコードとテストは触っていない。`./scripts/run-test.sh` と `go test` は回していない(下の「確認」)

## Summary

handoff の 1 から 6 を片付けた。docs の編集は 10 ファイル。

- org skill の verb 表の `inbox` の行に、resolve の断りと `--without-decision` を足した(L1)。4 写しは同じ内容にそろえた
- 仕様 156 行目の `--without-decision` の使いどころを、別の方法で知らせたときと、前の版の ralph で送ったときの 2 つにした(L2)
- tech-debt は 14 行に note や項目を足し、新しい行を 1 つ入れた。verify の L3、L4、L5 が挙げた行に加えて、verify が挙げなかった 118(この pass 自身の編集で当たった)と、verify が INFO として判断を残した 202、365 を足した。468 は足さなかった(理由は下の「tech-debt」)
- self-review の L4(承認の拾い方と `sentSubmitUnconfirmed` の写し)は、新しい行 1 つと、既存の 526 行目の拡張に分けて register に入れた
- quality-gates の「Feature plan approval」の行(root と template)と、`AGENTS.md` の `internal/org/` の行に 1 文ずつ足した

verify は 182 行 (a) で「`approve` の行が `stop` と `disband` より長くなった」と書いたが、数え直すと違った。`approve` の行は base で 1,384 字、`inbox` の行は 1,429 字で、`stop`(1,208)と `disband`(1,219)より最初から長い。note にはこの数を書いた。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | verb 表の `inbox` の行(387 行目)の、終了コード 1 を言う文のあとに 3 文。`resolve` は `approve` が承認を記録した計画の QUESTION を、承認の DECISION を leader に送るまで、何も書かずに終了コード 1 で断ること(エラーに send の打ち方が出る。dry run の send、承認より前の send、前の版の ralph の `sent` は数えない)。`--without-decision` は、数えられる DECISION のないまま件をわざと閉じるとき(別の方法で伝えたあと、または前の版の ralph で送ったとき)だけ使い、note は approve が出したままにすること。`.claude` を編集し、`./scripts/sync-skills.sh` で root の `.agents` を、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で template の `.agents` を作り、template の `.claude` は `cp` で写した(sync-skills.sh は template の `.claude` を作らない)。4 つの `shasum` は同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 156 行目(2026-11-19 の追記)の括弧の中。`--without-decision` の使いどころを、別の方法で leader に伝えたあと、または前の版の ralph で DECISION を送ったときの 2 つにした。後者は `sent` に `text_sha256` がないので数えない、は残した |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | 「Feature plan approval」の行の最後の文。「DECISION を送らなかった件は inbox の期限のもとに残る」の理由として、resolve が承認を記録した件を、承認の後に記録された本物の send(本文全体の `text_sha256` つき)がないうちは断ること、`--without-decision`、前の版の ralph の send は数えないことを足した |
| `AGENTS.md` | `internal/org/` の行の、`approval.go` の説明の括弧の中に `approve_resolve.go` の 1 節。manifest の lock の下で、承認の DECISION が leader に送られたと記録されるまで resolve を断る、`--without-decision` で例外 |
| `docs/plans/active/2026-11-19-approve-resolve-decision.md` | AC-1 から AC-10 の箱を `[x]` に。`git diff --numstat` は 10/10 で、変わったのはこの 10 行だけ。digest は `37ad5d8a83d5` のまま。Progress の「PR created」は `[ ]` のまま(`/pr` が付ける) |
| `docs/tech-debt/README.md` | 14 行の直しと新しい 1 行(下の「tech-debt」)。`git diff --numstat` は 15/14 |
| `docs/reports/sync-docs-2026-11-19-approve-resolve-decision.md` | この report |
| `docs/insights/events/2026-11-19-approve-resolve-decision.jsonl` | sync_docs の 1 行 |

## handoff の項目の処理

| 項目 | 結果 |
|---|---|
| 1. org skill の `inbox` の行(verify L1) | 上のとおり。4 写し |
| 2. 仕様 156 行目(verify L2) | 上のとおり |
| 3. tech-debt の carry-over(verify L3、L5) | 下の表のとおり。handoff の番号(190、240、264、367、379、467、382/209、519/520)は最初の verify のもので、merge のあとは 1 つずつ後ろにずれる。内容で探し直した |
| 4. self-review L4 | 新しい行(527 行目)と、526 行目への足し込み |
| 5. quality-gates | 足した。row が approve と resolve を述べていたので、2 写しとも |
| 6. そのほか | `AGENTS.md` の 1 文、plan の AC の箱、確認の項。README、`agent-messaging.md` などは変えない(下の「他に確かめた文書」) |

## tech-debt

行番号は、この pass の編集前の HEAD(`223c82f0`)のもの。526 行目のあとに新しい行を 1 つ入れたので、編集後は 527 行目以降が 1 つずつ後ろにずれる。note は Trigger のセルの末尾に足し、Related に計画(`docs/plans/active/…`。`archive/` は `/pr` の `archive-plan.sh` が書き換えるので先には書かない)を足した。

| 行 | 項目 | 判断と内容 |
|---|---|---|
| 118 | template の quality-gates の「Must pass in CI」 | この pass 自身の編集が当たった(own-edit hit)。template の quality-gates の「Feature plan approval」の行を触った。「Must pass in CI」の節と verify chain(`verify.yml`)は変えていないので払っていない。verify も handoff も挙げていない |
| 182 (a) | org skill の command table | verify L5。`approve` の行は 1,384 字から 1,681 字、`inbox` の行(この pass)は 1,429 字から 1,744 字(origin/main `32274500`、HEAD `223c82f0`、working tree で数えた)。`stop`(1,208)と `disband`(1,219)は変わらず、2 つの行は最初から両方より長い。table の整理は計画の Scope の外 |
| 190 (a) | leader の prompt | verify L3。`leader.md` の手順 3(起こされたときの読み方の 1 段落)、手順 4(1 つの句)、立て直しの inbox の項(1 文)。1,600 行から 1,626 行(origin/main と HEAD で `wc -l`)。`RolePromptVars` と `{{SCOPE}}` は変えていないので払っていない。(b) は当たっていない |
| 202 | feature-org の節の `leader.md`(g、done の f、j) | verify の INFO。(g) は f1b0352e、(f)(j) は b0fc4c59 で済み。前の PR と同じ形で、当たったが払うものはない、と note。新しい pin(`TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem`)は `assertPhrasesInOrder`(`squashSpace` を通す)で読むので、折り返しを直しても落ちない |
| 209 (a) | `withFileLock` | verify の「判断が要る」。`withFileLock` の doc(`lockfile.go`)だけを変えたが、Trigger の「the next change to `withFileLock`」は comment だけの変更も数える決まり(row 88 の前例)なので当たりとした。この branch の lock のテストは同じ process の goroutine で lock を持つ(`waitForLockWaiter`)ので、(a) が求める子 process のテストは足していない |
| 228 (b) | `Send` | verify L5。`Send` の doc に `text_sha256` の段落、本体の `sent` の append に `TextSHA256` を足した。(b) の 7 回目の当たり。`approval_test.go` の `approval gets` の短い行は直していない。(c) は当たっていない |
| 241 | gate の small findings(`lockfile.go` の次の変更) | verify L3。`lockfile.go` の `withFileLock` の doc に lock の順(manifest、次に inbox)を足した。(g) が名指す doc そのものだが、「the flock mechanism behind withManifestLock」と「each of the five」は残っているので、(g) は触れたが払っていない。(g) を書き直す人は新しい lock の順の文を残す、と書いた。(a)〜(d)、(f)〜(i) は触っていない |
| 265 | `/director` skill の inbox の節 | verify L3。「計画の承認」の 3 と 4(4 写し)。`director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は触っていない。新しい文は `TestDirectorSkill_ApprovalResolveWaitsForTheDecision`(`prompts_leader_plan_test.go`)が固める。(1)〜(3) は開いたまま |
| 365 | roster(`manifest.go`) | verify は「当たらない」とした行。前例(2026-11-05 の `ModelCLIDefault`)が、`ManifestEvent` に欄を足したことを当たりとして数えているので、`TextSHA256` も同じ形で note にした。`Roster` と seat の表は変えていない |
| 368 | `Send` と leader の template のテスト、800 行 | verify L3。`verbs.go` 2639 → 2655(+16)、`prompts_leader_plan_test.go` 1302 → 1476(+174。+300 の下)、`leader.md` 1600 → 1626。ほかの 5 ファイルは変わっていない。最後の Go の commit `4c4edcd7` は新しいテストファイルだけを変えた。新しい `internal/org/approve_resolve_test.go` は 866 行で 800 行を 66 行超えるが、どの行もこのファイルを持っていない(note にだけ書いた。下の「直さなかったこと」) |
| 380 | `leader.md` の feature-org の手順(RC4) | verify L3。手順 3 の段落と手順 4 の句。RC4 は Scope の外で、plan の 2 行を書き換える文は道具を名指さないまま |
| 383 | `/director` skill の send と確かめ | verify の「判断が要る」。手順 4 は send の確かめ(DECISION の `approved` の行が送られた入力として出ていて、入力欄に残っていない)を持つ。確かめの文は変えていないが、その手順を編集したので当たりとした。ほかの兆候の名指しと `ralph org read` は変えていない |
| 520 | `ApprovalDecisionCommand` の呼び手が `--config` を付けない | verify L4。新しい呼び手が 2 つある。resolve の断り(`approvalDecisionNotSentError`、`ApprovalDecisionCommand`)と、`printResolveNotes` の `ralph org read`(`orgVerbHint`)。どちらも `--state-dir` だけ。断りの resolve(`ApprovalResolveCommand`)も同じ形で、`ralph org inbox resolve` も `newOrgRuntimeAt` で `--config` を読む。Debt のセルの末尾に足し、Trigger に「この 2 つの次の変更も当たる」を足し、Related にコードの参照を足した |
| 526(と新しい 527) | `sent` の Details の語の読み手 | self-review L4 の (b)。526 行目は、3 つ目の読み手 `sentSubmitUnconfirmed`(`approve_resolve.go`)を Debt と Trigger と Related に足した。この branch は `Send` に `text_sha256` の欄を足したが、Details の語は変えていないので、Trigger の前半は当たっていない |
| 527(新) | `checkApprovalDecision` の承認の拾い方 | self-review L4 の (a)。`currentPlanApprovals` の条件(dry run でない、org 全体の `plan_approved`、最後の `disbanded` のあと、`planApprovalFromDetails` が読める)の手書きの写し。(b) の `sentSubmitUnconfirmed` にも触れ、526 行目を指す。5 つのセルをすべて書いた。末尾ではなく 526 の次に置いた(ほかの PR が末尾に足す行と衝突しないため) |
| 261 | #345 の行 | この branch が書いた note が既にある。変えない |

関数名で書き、`file:line` の参照は足していない。

### 当たらなかったとした行

| 行 | 理由 |
|---|---|
| 468 | Debt が `~~...~~ (RESOLVED 2026-11-13 in docs/director-notice-handling)` で、skill には scaffold の版の note の文がもうある(`.claude/skills/director/SKILL.md` の 1632 行目あたり)。Trigger のセルは打ち消されておらず、2026-11-15 の note は「skill はまだ何も言わない」と書いたままで、今の skill と合わない。carry-over を足すと偽の文を重ねるので足していない。Trigger の打ち消しは別の branch の仕事なので直していない(lead に知らせる) |
| 387、429 | 387 は手順 14 の変更、429 は節を足す変更が Trigger。この branch は手順 3 と 4 を変えただけで、節は足していない |
| 525 | Trigger は `leader.md` の「agmsg の受信箱の運用」の節。この branch の hunk は `## 機能ごとの org`(118〜1334 行)の中だけ |
| 252 | Trigger は `mergeorder.go` などの編集。`manifest.go` は Debt の (f) に出るだけ(`ManifestStore.Read` の decode の写しで、decode は変えていない) |
| 428 | Trigger は `org_approve_test.go` の 4 つの特定のテスト。この branch が直したテストはその 4 つではない |
| 117、235 | 117 は template の restructure、235 (e) は `fetch-depth` の文。どちらも変えていない |
| 226、139、340、341、464、477、515 | `Send` の別の部分(seat の lookup、idle の待ち、承認の検査、`via=screen`、`sendTypedNotSubmittedMark`)や `checkResolvedMeanwhile` が Trigger。この branch は `Send` に `TextSHA256` を足しただけで、どれも変えていない |
| 522、523、529 | `reseat_progress.go` の変更や leader の re-seat の規則の項が Trigger。この branch は re-seat の規則の「受信箱」の項に 1 文足したが、Trigger が名指す gate の項、seat-check の項、`reseat_progress.go` は変えていない |

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `README.md` | 変えない。133 行目と 290 行目に、resolve の断りと `--without-decision` が既にある(slice 3 の `86f72048`)。verify も同じ判断 |
| `.claude/rules/ralph/agent-messaging.md`(2 写し) | 変えない。typed protocol と send の打ち方の契約で、`sent` の記録の欄は書いていない(`sent` は `raw=true` の記録の話だけ)。承認の resolve の話もない |
| `docs/specs/2026-08-01-org-runtime.md` | 変えない。`sent` の欄の一覧を持たない |
| `docs/architecture/repo-map.md` | 変えない。script の一覧で、新しい script はない |
| `docs/quality/definition-of-done.md`(2 写し) | 変えない |
| `docs/recipes/` | 変えない。approve と resolve を述べる recipe はない |
| org skill の「受信箱」の項と「機能の計画の承認」 | 変えない。slice 3 で書いた文(断り、`--without-decision`、note は変えない)が、code と、director skill、`--help` と合っている |
| hooks、rules、language pack | この branch は触っていない |

## 直さなかったこと

- 新しいテストファイル `internal/org/approve_resolve_test.go`(866 行)が 800 行の目安を超えている。self-review、verify、test のどれも挙げていない。368 行目の note に書いたが、このファイルのための行は作っていない。行にするかどうかは lead の判断
- org skill の新しい文(`inbox` の行、受信箱の項、機能の計画の承認)を固めるテストはない。director skill は `TestDirectorSkill_ApprovalResolveWaitsForTheDecision` が固める。計画の AC-7 は director skill、approve と resolve の help だけを求めたので、足りなくはないが、org skill の文は code と別々に変わりうる。行は作っていない
- 468 行目の Trigger の打ち消し(上の表)
- self-review の N1 から N3 は `ae5b028f` で直っていて、この pass の対象ではない。verify の「実地は未確認」の点(本物の herdr と leader での起こされ方)は、doc で直せない

## 確認

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`。org skill の 4 写しの `shasum` は同じ
- 計画の digest: `37ad5d8a83d5`(変わっていない)。計画のファイルの `git diff --numstat` は 10/10 で、AC の 10 行だけ
- tech-debt の編集: HEAD と突き合わせ(`tdcheck.py`、scratch)。変わった行は 118、182、190、202、209、228、241、265、365、368、380、383、520、526 の 14 行と、新しい 527 行目だけ。どの行も 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし。ほかの行は 1 文字も変わらない
- 計画の参照: `docs/plans/active/2026-11-19-approve-resolve-decision.md` は実在する。`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索を手で回し、欠けた参照は 0 件。`verify.local.sh` そのものは回していない
- 回していないもの: `./scripts/run-test.sh`(handoff の決まり)、`go test`、`run-verify.sh`、`verify.local.sh`。この pass は Go のコードを変えていない

## Insight event

`./scripts/insights-append.sh --slug approve-resolve-decision --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-approve-resolve-decision.jsonl` に足した(`cycle` 1)。終了コードは 0、`git status --short` には既存の jsonl が `M` で出るだけで、余分なファイルは増えていない。
