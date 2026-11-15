# Verify report: reseat-leader-progress

- Date: 2026-11-16(cycle 2)。cycle 1 は 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)。`- Approved:` の digest は `sha256:de0cca0b6f78`(計算し直していない。承認の検査は `/implement` が済ませている)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン **2 回目**(`cycle-count.json` は `cycle: 2`。`max_pipeline_cycles` は 2 で、この回が最後)。新しい run を先頭に置き、cycle 1 は末尾に短い履歴にした
- Scope: HEAD `f6be8593`、`git diff main...HEAD`(base `48a8a661`、29 ファイル、+3267/-54)。cycle 1 の verify(`e483b52a` までの打ち直し)のあとに入ったのは `77a45022`(sync-docs)、`ec10ba6f`(cross-review の直し)、`b8a9bd38`(self-review cycle 2 の直し)と、記録の commit。`reseat_progress.go` は 412 行から 550 行、`leader.md` は 1,480 行から 1,516 行になった
- Evidence: `docs/evidence/verify-2026-11-16-reseat-leader-progress-cycle2.log`(`run-static-verify.sh` の出力そのものと、`git diff --check`・U+FFFD・試し merge・skill の写しの `cmp`・行数・ずれの箇所の grep。gitignore の対象で commit に入らない)。`run-static-verify.sh` 自身が書いた `docs/evidence/verify-2026-11-15-171019.log` も同じ内容
- Insight event: `./scripts/insights-append.sh --phase verify --verdict pass --cycle auto`(cycle 2、LOW 5)を 1 行足した(この report と同じ commit)
- **Verdict(cycle 2): partial-pass**。静的検査は通った(rc=0、golangci-lint 0 issues)。AC1〜AC6・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は満たす。AC8 は、cross-review の直しが節に足した 2 行を、文書の 14 か所の列挙が持たないので一部。AC7 は `/test` の担当で、cycle 1 の test report は `ec10ba6f` より前

## Cycle 2

### Spec compliance(cycle 1 から変わった点)

仕様 FR-9・FR-10・AC-13・AC-14 と計画の AC1〜AC9 を、本体とテストの file:line で読んだ。テストは実行していない(`/test` の担当)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 節に承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met | cycle 1 の 5 行に加え、`reseat_progress.go:304-313`(座席ごとの最後の TASK)、`:328-342`(上げた RESULT)、`:324-325` の QUESTION の `PLAN` 付き。テスト: `TestReseatProgress_ASectionHoldsAllPartsInOrder`、`..._TheLastTaskPerSeatComesFromTheSentEvents`(9 ケース)、`..._TheResultsTheLeaderRaisedAreListedInEveryState`、`..._AQuestionLineShowsItsPlanHeader` |
| AC2 `start --plan` の立て直しも同じ節。経過の無い org に節なし。disband のあとの start は commit だけ | Met(結合は部分、V7 のまま) | 空の規則: `add`(`:280-283`)が TASK の行と RESULT の行でも節を作るので、TASK か RESULT だけの org にも節が入る(計画 D6 と AC2 の括弧書きに TASK・RESULT は無い。V17)。`TestReseatProgress_AnOrgWithNoProgressHasNoSection` は、ほかの org の TASK と RESULT、dry run の TASK、TASK でない typed message、resolved の QUESTION、読み取りの失敗だけでは節がないことを固定する。`StartFeature` を通した disband→start→commit だけの節のテストは無い(tech-debt 行 521 (a) に記録済み) |
| AC3 今の命の外は載らない | Met | Tasks は `events[start:]`(`:156`。最後の本物の `disbanded` の次から)、Results は `disbanded` の時刻より後の件(`:191-206`。`time.Parse` で比べ、同じ秒は前の側、読めない時刻は注つきで残す)。テスト: `..._TheLastTaskPerSeat...` の「a TASK before the last disband is not used, one after it is」、`..._TheResultsTheLeaderRaised...`(disband 前の RESULT、ほかの org の RESULT、resolved の QUESTION が入らない。読めない時刻は注) |
| AC4 commit は新しい順に 20 件と総数。git が読めないときは 1 行 | Met(変更なし) | `readReseatProgressCommits` と `reseatProgressCommitsIn`(`:515-550`)は `e483b52a` から変わらない(`git diff e483b52a..HEAD` の hunk がこの範囲に無い) |
| AC5 値は 1 行にして 120 字で切り、本文は fence の中 | Met | 新しい行の値は `value()` か `reseatProgressText` を通る。RESULT の 1 行目は 120 字、HEAD と PLAN は 400 字(`:339-340`、`:436`)、PLAN と HEAD の値は `strings.Fields` で空白を 1 つにまとめる(`:420`)。新しい 2 行の頭は `座席ごとの最後の TASK:` `上げた RESULT:` と `- `。`reseatSectionLabels`(`prompts_leader_reseat_test.go:153-173`)が、`- ` で始まらない行は全部ラベルと `:` で始まると固定する |
| AC6 `leader.md` に規則があり、役割の prompt のテストが文を固定する(仕様 AC-14) | Met | 導入(`leader.md:150-161`)と 5 項目(座席の確認 `:162`、計画 `:204`、gate `:238`、受信箱 `:261`、座席 `:265`)。規則は `次の順で進めます。`(`:275`)の前。テストは 9 本のまま(`prompts_leader_reseat_test.go`)で、`b8a9bd38` が入力待ちの出口の文を足し、最後の TASK の行を `slice-1 (<時刻>)` に合わせた |
| AC9 gate が通ったあと全座席を reseat した場合 | Met(広げてある) | gate の項目(`:238-260`)は、座席が `(active)` になるのを待ち(10 分で BLOCKED)、手順 13 で止め、HEAD と `git status --porcelain` を確かめ、`gh pr list --head <ブランチ> --state all --json number,state,url` で状態を読む。`OPEN` か `MERGED` があれば PR は作らず、「上げた RESULT」に TASK_ID が org_id で HEAD が gate の head と同じ件がなければ手順 14 の RESULT を上げて手順 15。`CLOSED` だけなら BLOCKED。なければ手順 14。テスト: `TestRenderRolePrompt_Leader_ReseatRuleGateItem` |
| AC7 既存のテストが通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | 既存のテストファイルで変わったのは `internal/cli/org_reseat_test.go`(+3、`--help` の pin、`77a45022`)だけ。`featureLeaderTask` と `feature.go` は `e483b52a` から変わらない。cycle 1 の test report(`fc8f236a`、pass)は `ec10ba6f` と `b8a9bd38` より前で、`reseat_progress_test.go`(589 から 869 行)と `prompts_leader_reseat_test.go`(438 から 516 行)はそのあとで変わった。`/test` の打ち直しが要る |
| AC8 仕様の印、AC-12 は `[ ]`、AGENTS.md と skill の説明が節に合う | 一部 | 仕様の印と AC-12 は cycle 1 のまま。AGENTS.md、README、skill、`--help` の列挙は、cycle 1 の sync-docs が書いた時点の 6 項目(承認、gate、受信箱、commit、起動予定の座席、前の reseat)で、`ec10ba6f` が足した 2 行を持たない(V11) |
| 仕様 FR-9 の 5 項目 | Met | 5 項目はすべて持つ。仕様に無い 2 行(座席ごとの最後の TASK、上げた RESULT)と QUESTION の `PLAN` は、cross-review の finding 1・2・4 の直しで足した。仕様の FR-9 の列挙は書き足されていない(V11) |
| 仕様 FR-10 の 4 文 | Met | 規則の 5 項目(座席の確認、計画、gate、受信箱、座席)が、承認済みの計画を書き直さない、gate が今の HEAD で pass なら PR の手順へ、受信箱は答えを待つ、立て直された座席の TASK を送り直す、に対応する。gate の項目の PR の状態の分岐と RESULT の上げ直しは、FR-10 の 1 文の中身の細かい点で、仕様の文と矛盾しない |

### 規則が読む値と節が出す値

`leader.md` の規則が「行」「値」として読むものを 1 つずつ、節を作る側と突き合わせた。

| 規則が読む値 | 規則の位置 | 節が出す値 | 結果 |
| --- | --- | --- | --- |
| 「計画の承認」の行のパスと `sha256:<digest>`(空白で区切った 1 語) | `leader.md:204-` | `計画の承認: <パス> sha256:<digest> (承認 <時刻>)`(`reseat_progress.go:287`) | 合う。`TestReseatProgress_TheApprovalLineGivesTheDigestAsOneWord` が 1 語を固定する |
| 「最後の gate」の `pass`、`head`、`base` | `:162-`、`:238-` | `最後の gate: pass(<event>)head <sha> base <sha> branch ...`(`:294`) | 合う。括弧が語に付く点は tech-debt 行 519 (d) |
| 「座席ごとの最後の TASK」のその座席の行の TASK_ID | `:182-` | `- <座席>: <TASK_ID> (<時刻>)`(`:311`)。TASK_ID が無いときは `-` | 合う。`b8a9bd38` で TASK_ID と時刻の括弧のあいだに空白を入れ、1 語で切れる。`-` の扱いは規則に書かれていない(V16) |
| 「片付いていない受信箱の件」の QUESTION の行の終わりの `PLAN <パス> sha256:<digest>`、open か acked | `:222-` | `- <id> QUESTION <TASK_ID> <state>: <1 行目> / PLAN <パス> sha256:<digest>`(`:324-325`、`:436`)。`PLAN:` の無い QUESTION は付かない | 合う。手順 3 の QUESTION は `PLAN:` のヘッダを持つ(`leader.md:336`、例 `:351`) |
| 「上げた RESULT」の TASK_ID(org_id)と `HEAD`、gate の head との比較 | `:238-` | `- <id> <state> <TASK_ID> HEAD <head>: <1 行目>`(`:339`)。`HEAD` は RESULT の本文のヘッダ | 合う。手順 14 の RESULT は `TASK_ID: {{ORG_ID}}` と `HEAD:` を持つ(`leader.md:756-762` の本文、例は `:772-776`)。open か acked の RESULT は受信箱の一覧と RESULT の一覧の両方に出る(`..._ListedInEveryState` が固定) |
| 「branch の commit」の行 | `:182-`、`:204-` | `branch の commit: N 件(新しい順に M 件)`と `- <hash> <件名>`(`:349-357`)。読めなかったときは同じ見出しで理由 1 行(`:345`) | 合う。読めなかったときの行も同じ見出しである点は V16 |
| 「今回の立て直し」の起動予定の座席 | `:238-`、`:265-` | `今回の立て直し: reseat=<id>、leader のあとに起動する予定の座席: <座席|なし(leader だけ)>`(`:368`) | 合う |
| 手順 14 に渡す gate の `HEAD:` と `BASE:` | `:257-` | gate の行の `head` と `base` | 合う |
| 節の外: `ralph org status` の `(active)`、`history.sh`、手順番号 | `:162-`、`:182-` | cycle 1 で実在を確かめた(`internal/cli/org.go:2158`、`leader.md:1408`)。手順番号は `TestRenderRolePrompt_Leader_ReseatRuleStepsAreTheStepsItMeans` が固定する | 合う |

機械的な固定: `TestRenderRolePrompt_Leader_ReseatRuleNamesTheLinesTheSectionHas`(`prompts_leader_reseat_test.go:183-217`)が、全部の部分を持つ節を作り、ラベルが 1 つでも変われば落ちる。規則が名指す 7 ラベル(「前の立て直し」以外のすべて)が節に在ることと、規則が引く値(digest、gate の `pass`・head・base、`slice-1 (<時刻>)`、`PLAN <パス> sha256:<digest>`、`HEAD <sha>`、起動予定の座席)が節に印字されることを見る。

### 今の命の境目と空の規則(新しい 2 項目)

- 座席ごとの最後の TASK: `events[start:]` で読むので、最後の本物の `disbanded` より前の TASK は載らない。dry run、ほかの org、leader 宛て、TASK でない typed message は数えない。`sent` の Details の前に付く語(`raw=true `、`via=screen herdr_state=<state> `、`submit_unconfirmed=true `)は `=` を含む語として落とす。`Send` の `sent` は `OrgID` と `SeatID`(宛先)を持つ(`verbs.go:484`、`:671`)
- 上げた RESULT: 受信箱の件の `EscalatedAt` を `disbanded` の時刻と比べる。受信箱の件と同じ規則(同じ秒は前の側、読めない時刻は注つきで残す)。5 件を超えると古いものを `ほか N 件(古いもの)` と数える
- 空の規則: TASK か RESULT だけの org にも節が入る。受信箱か git の読み取りの失敗は引き続き節を作らない(`:373-375`、`TestReseatProgress_AnOrgWithNoProgressHasNoSection`)

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **PASS**(rc=0、`==> All verifiers passed.`) | gofmt ok、`golangci-lint` は `0 issues.`(cycle 1 の V1 は再発しない)、`go vet` は出力なし。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 つの settings、Codex hook の 3 つの検査、`check-sync.sh`(DRIFTED=0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の計画の参照 |
| branch の secret scan | PASS | `48a8a661..f6be8593` clean |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイルの U+FFFD | 0 件 | `git diff main...HEAD --name-only` の追加・変更のファイルすべて |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし | `origin/main` は `main` と同じ `48a8a661`(fetch していない手元の ref) |
| skill の写しの `cmp` | 同じ | `/org` と `/director` の `.claude` と、`.agents`、`templates/base/.claude`、`templates/base/.agents` |

### Documentation drift(`/sync-docs` に渡す一覧)

cycle 1 の drift(V2〜V6、V9、V10)は `77a45022` で閉じた(README の reseat の段落、`/org` と `/director` の skill、`docs/specs/2026-08-01-org-runtime.md` の改訂、AGENTS.md、仕様の影響の表、tech-debt の行 519〜521 と持ち越しの注)。以下は `ec10ba6f` と `b8a9bd38` が新しく作ったずれと、self-review cycle 2 の引き継ぎの確認。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:303` の「A leader it starts gets, at the end of its task, a section 立て直しの経過 ...」 | No(V11) | 列挙が 6 項目(承認、gate、受信箱、新しい 20 件の commit と総数、起動予定の座席、前の reseat)。節は「座席ごとの最後の TASK」と「上げた RESULT」も持つ。self-review の引き継ぎのとおり |
| `AGENTS.md:92` の `internal/org/` の行、`reseat_progress.go` の説明 | No(V11) | 同じ 6 項目の列挙(「the latest plan approval, the last gate record, the open and acked inbox items, the branch's commits, this reseat's seats, and the previous reseat」)。self-review の引き継ぎのとおり |
| `docs/specs/2026-11-12-org-reseat.md:80`(FR-9) | No(V11) | 列挙に 2 行が無い。`[x]` の FR の行なので、列挙に足すか、日付つきの印を足すかは `/sync-docs` が決める(`:141` の影響の表は `StartFeature` に直してあり合う) |
| `docs/specs/2026-08-01-org-runtime.md:125`(2026-11-15 改訂 (a)) | No(V11) | 同じ列挙。この仕様は本文を書き換えず改訂の節を日付ごとに足す作り(`:85`)なので、(a) に足すか 2026-11-16 の改訂を足す。`5 項目の規則` の数と中身は合う |
| `/org` skill の 4 つの写し(`.claude`、`.agents`、`templates/base/.claude`、`templates/base/.agents`)の `:1836-1842` | No(V11) | 「計画の承認、最後の gate の記録、片付いていない受信箱の件、branch の commit、今回の reseat の id と起動する座席、前の reseat」の列挙。4 つは `cmp` で同じなので、1 つ直して `scripts/sync-skills.sh` と `check-skill-sync.sh` に通す |
| `/director` skill の 4 つの写し、`:1565-1567` | No(V11) | 「節(最新の計画の承認、最後の gate、片付いていない受信箱の件、branch の commit、今回の立て直しの座席、前の reseat)」の列挙。同じ 4 パス |
| `ralph org reseat --help`(`internal/cli/org_reseat.go:104-109`)と、その pin(`internal/cli/org_reseat_test.go:873-875`) | No(V11) | help の文は 6 項目を閉じた列挙で書き、test が逐語で固定する。直すなら 2 つを一緒に直す。`ralph org start --help` は節に触れず、変更なしでよい |
| `internal/org/reseat_progress.go:257-258` の doc | No(V12) | 「`- <seat>: <TASK_ID>(<time>)`」。`b8a9bd38` が出力を `<TASK_ID> (<time>)` に変えた(`:311`)。コメントだけ古い |
| `internal/org/feature.go:226-228` の `StartFeature` の doc | No(V12) | 「when the org has progress in its current life (an approval, a gate record, an open inbox item, commits on the branch, a reseat)」。送った TASK と RESULT の件だけでも節が入る(`reseat_progress.go:280-283`) |
| `internal/org/prompts/leader.md:1410`(「agmsg の受信箱の運用」)と `internal/org/prompts_leader_inbox_test.go:107` | No(V13) | 「あなたが送った TASK も 50 件に入るので」。`Send` は agmsg に何も投げない(`verbs.go` で `o.Agmsg.` を呼ぶのは `:1479` の `Leave` だけ)。新しい座席の確認(`leader.md:182-184`)は「送った TASK は台帳に残り、agmsg の履歴には残りません」と書き、逆を言う。main にあった文 |
| `docs/tech-debt/README.md` | No(V14、V15) | 下に分けて書く |
| 計画 `docs/plans/active/2026-11-15-reseat-leader-progress.md` の D6 と AC2 の括弧書き、AC の checkbox | 不一致(INFO、V17) | 節ができる条件に、送った TASK と上げた RESULT が入っていない。AC1〜AC9 の checkbox は全部 `[ ]` のまま(`/pr` の前に付ける。drift ではない) |
| `ralph org start --help`(`internal/cli/org.go:1115-1124`)、`/org` skill の `start` の行と「機能ごとの org」の leader のタスクの説明 | Yes(INFO) | 節に触れない。cycle 1 と同じで、1 文足すのは任意 |

### tech-debt の Trigger と数(`/sync-docs` が直す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を読んだ。

- 行 520 の (a)(`--state all` が閉じた PR を数える)、(b)(PR のあと RESULT の前に止まる)、(c)(計画助言が済んだか分からない)は `ec10ba6f` が直した(gate の項目の `OPEN` `MERGED` `CLOSED` の分岐、「上げた RESULT」の確認、QUESTION の `PLAN` と `## Design decisions` の要約)。行はまだ 3 つを未解決として書く。(d)(実機の確認)は残る
- 行 519 (a) の「86 lines」は、`reseatProgressSection` が 103 行になって数が違う(`:277-379`)。(d) のゲートの行の括弧は変わらず残る。行 519 の Trigger(`reseat_progress.go` か gate の行を読む規則を触る)に、この PR 自身の `ec10ba6f` が当たる(同じ PR の行なので、持ち越しではなく数と状態の直し)
- 数が古い行: 行 365(`prompts_leader_reseat_test.go` 438 は 516 行)、行 426(`leader.md` 1,480 / +89 は 1,516 / +125)、行 514(`reseat_progress.go` 412 は 550、`reseat_progress_test.go` 589 は 869、`prompts_leader_reseat_test.go` 438 は 516。`reseat_progress_start_test.go` 411 は変わらない)。数は `wc -l` を `e483b52a` と HEAD で比べた。行 514 の「all under the guide」は、`reseat_progress_test.go` が 800 行を超えたので偽になる(V15)
- 行 465(`/director` skill の次の変更): `77a45022` が `/director` skill の 4 つの写しを直し、`/sync-docs` の cycle 2 がもう一度直す。字義どおり当たるが、cycle 1 の持ち越しの注は 514、201、365、426、396、513、515 の 7 行で、行 465 は無い(行 465 には、`feat/gate-lock-accept` が `/director` skill を直して `hit this trigger, but did not pay it` と書いた前例がある)
- 行 201、396(`feature.go`)、行 513、515(help と skill の段落)、行 514 の `reseat.go` と `feature.go` の数(1,553 と 873)は、cycle 2 の fix commit が `reseat.go` と `feature.go` に触れていない(`git diff e483b52a..HEAD` に両方とも無い)ので cycle 1 のまま。`/sync-docs` が `--help` と skill を直すと、行 513 (f)(j)(m)(n) と行 515 (h)(n) にもう一度当たる
- 行 377(`leader.md` の機能ごとの org の手順)と 384(手順 14)、行 508(hook の bullet): 規則は手順の前に置いただけで、手順 4・14 の本文は変えていない。当たらない
- register に足す新しい項: (1) V13(`leader.md:1410` と座席の確認の逆)と、self-review cycle 2 の C2-4(`TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites` の語が `Send` の文字列に結びつかない。`verbs.go:479` と `:666` と `send_fallback.go:60-65`)。Trigger は「次に `leader.md` の座席の確認か『agmsg の受信箱の運用』の節、`reseat_progress.go` の Tasks の行、`sent` の Details を作る `Send` を触るとき」。(2) V15(新しいファイルが 800 行を超えた)。(3) 実機の限りは行 520 (d) が持つ

### Observational checks

- `leader.md` の gate の項目が手順 14 の RESULT を上げる分岐は、手順 14 の RESULT の書き方(`TASK_ID: {{ORG_ID}}`、`HEAD:`、`BASE:`、`DOC_FOLLOWUP:` か `SYNC_DOCS:`、`CROSS_REVIEW:`、`EVIDENCE:`)と矛盾しない。控えが残っていない `SYNC_DOCS:` と cross-review の値は「branch の report から読み、読めなければ `立て直しのため控えが残っていない`」で、手順 9 の `SYNC_DOCS: skipped` を控える文(`TestRenderRolePrompt_Leader_ReseatRuleStepsAreTheStepsItMeans` が固定)と合う
- 座席の確認の入力待ちの分岐は、`TASK_ID` の頭で行き先を分ける(`gate-` は手順 12、`xreview-` は「cross-review」の小節、それ以外は手順 8)。RESULT がない出口も、implementer のスライスの TASK は commit の行で、reviewer のレビューの TASK と `gate-`・`xreview-` の TASK は新しい TASK_ID で頼み直す形に直った(`b8a9bd38`、self-review C2-2)
- 計画項目の計画助言の判定(per-plan は QUESTION の `PLAN` の digest、split は `## Design decisions` の要約)は、「計画助言」の小節が `ADVISORY:` の行と `## Design decisions` に印を残すと書くことと合う

## Coverage gaps

- テストの実行結果。この report は本体とテストを読んだだけで、通ることは確かめていない。`reseat_progress_test.go`、`prompts_leader_reseat_test.go`、`org_reseat_test.go` は cycle 1 の test report のあとで変わった(`/test` の打ち直し)
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 (d))
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))
- `golangci-lint` が CI で動くか。`.github/workflows/verify.yml` に導入の step があるかは、cycle 1 で未確認とした点のまま(今回は手元で `0 issues`)

## Findings

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V11 | LOW | doc | 節の中身を列挙する 14 か所(README:303、AGENTS.md:92、reseat 仕様 FR-9 の `:80`、org-runtime 仕様の `:125`、`/org` skill の 4 つ、`/director` skill の 4 つ、`ralph org reseat --help` と test の pin)が、`ec10ba6f` が足した 2 行(座席ごとの最後の TASK、上げた RESULT)を持たない。`77a45022` の時点では正しく、あとから古くなった。AC8 の「説明が節に合う」が一部になる | `/sync-docs`(`--help` と pin は一緒に) |
| V12 | LOW | comment | `reseat_progress.go:257-258` の doc は最後の TASK の行を `<TASK_ID>(<time>)` と書き、出力は `<TASK_ID> (<time>)`(`b8a9bd38`)。`feature.go:226-228` の doc は、節ができる場合に送った TASK と RESULT の件を挙げない | `/sync-docs` か tech-debt。コメントだけの変更でも Go のファイルなので、直すかどうかは lead が決める |
| V13 | LOW | prompt | `leader.md:1410` の「あなたが送った TASK も 50 件に入る」は `Send` の挙動と合わず、同じファイルの座席の確認(`:182-184`)と逆を言う。`prompts_leader_inbox_test.go:107` がこの文を固定する。main にあった文を、座席の確認の直しが矛盾にした(self-review C2-1) | 直すなら `leader.md` と 2 つのテストで、`/self-review` からやり直し。cap に達したので、行を register に足して進めるのが現実的 |
| V14 | LOW | tech-debt | 行 519 (a) の「86 lines」(103 が正)、行 520 (a)〜(c)(`ec10ba6f` が直した)、行 365・426・514 の行数(上の「tech-debt の Trigger と数」)が古い。行 465 の持ち越しの注が無い。V13 と C2-4 の行が無い | `/sync-docs` |
| V15 | LOW | size | `internal/org/reseat_progress_test.go` は 869 行(`e483b52a` で 589)で、新しいファイルが 800 行の目安を超えた。行 514 の「all under the guide」が偽になる | register に 1 行(move-only の分割は `/self-review` からやり直しなので払わない)。`/sync-docs` |
| V16 | INFO | rule | 規則に書かれていない縁が 3 つ。(a) 座席の行の TASK_ID が `-`(`--raw` の TASK で `TASK_ID:` が無い)のときの扱い。実際は同じ TASK_ID を履歴で探して見つからず、送り直しに進む。(b) 「branch の commit」の行は、読めなかったときも同じ見出しで出るので、計画項目の「commit の行があって」を満たしてしまう。そのとき計画項目は `git log --name-only` で自分でも調べるので、害は小さい。(c) 座席ごとの最後の TASK に RESULT が見つかっても、その RESULT を前の leader が処理して次の座席へ TASK を送ったあとかは、時刻を比べないと分からない | 実機の通し(行 520 (d))で見る。今は直さない |
| V17 | INFO | plan | 計画の D6 と AC2 の括弧書きは、節ができる条件に送った TASK と上げた RESULT を含まない。実装は両方で節を作る(`:280-283`)。計画の本文は変えない記録の書き方(cycle 1 の V8 と同じ) | `/pr` の Known gaps か、行 521 に 1 文 |

cycle 1 の V7(結合のテスト)と V8(計画の本文との差)は、行 521 と行 520 に記録済みで変わらない。

## Verdict

- Verified: 静的検査(`run-static-verify.sh` rc=0、golangci-lint 0 issues、`git diff --check`、U+FFFD 0 件、`origin/main` との試し merge、skill の写しの一致)。AC1・AC3・AC4・AC5・AC6・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応。`leader.md` の規則が読む値のすべてを節が出すこと(規則が名指す 7 ラベルは test が機械的に固定)。新しい 2 項目が今の命の境目と空の規則を守ること
- Partially verified: AC2(`StartFeature` を通した commit だけの節の結合、行 521 (a))、AC8(仕様の印は済み。列挙の文書 14 か所が節の 2 行を持たない、V11)
- Not verified: AC7(既存のテストが通ること、`/test` の打ち直し)、節を読んだ leader の実機での動き、`golangci-lint` が CI で動くか
- Verdict: **partial-pass**。静的検査は通り、仕様の AC は読んだ範囲ですべて満たす。残りは `/sync-docs` の文書の直し(V11、V12、V14、V15)と register の行(V13、V15)で、fail にする指摘はない。insight event の `verdict` は `pass`

## Cycle 1(履歴): fail から partial-pass

HEAD `9748a702` と、V1 の直し `e483b52a` のあと。報告書の全文は `git show 1a1d8981:docs/reports/verify-2026-11-15-reseat-leader-progress.md`。

| ID | Severity | 内容 | cycle 2 の状態 |
| --- | --- | --- | --- |
| V1 | MEDIUM | `prompts_leader_reseat_test.go:94` の staticcheck QF1001 | 閉じた(`e483b52a`)。cycle 2 の `golangci-lint` は 0 issues |
| V2〜V4 | MEDIUM | README の reseat の段落と `/org`・`/director` skill が「reseat は経過を leader に渡さない」と逆を言う | 閉じた(`77a45022`)。列挙の古さは V11 |
| V5、V6、V9 | LOW | org-runtime 仕様の (d)、AGENTS.md、reseat 仕様の影響の表 | 閉じた(`77a45022`)。列挙の古さは V11 |
| V7 | LOW | `StartFeature` を通した disband→start→commit だけの節のテストが無い | 行 521 (a) に記録 |
| V8 | LOW | 計画 D7 の `gh pr list --head` に `--state all` を足した差と、AC4 の読み取りの失敗の行 | 行 520 と 521 (b) に記録。`--state all` は `ec10ba6f` で `--json number,state,url` の状態の分岐になった |
| V10 | LOW | tech-debt の持ち越しと新しい項 | 閉じた(`77a45022`、行 519〜521 と持ち越しの注)。数の古さは V14 |

cycle 1 の Verdict: partial-pass(V1 は直し、残りは `/sync-docs` と記録)。
