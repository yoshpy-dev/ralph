# sync-docs report: reseat-leader-progress

- Date: 2026-11-16(cycle 3。run 3、cycle 2 の最初の run、cycle 1 も同じ日)
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(承認の digest `de0cca0b6f78`)。issue #312 の #312c-2
- Branch: `feat/reseat-leader-progress`。run 開始時の HEAD は `36d80504`(cycle 3 の test report の commit。report の中の `HEAD:` の行の値ではない)。内容の commit は `d812689e`
- Pipeline run: cycle 3(`cycle-count.json` は `cycle` 3、`cap` 3。3 回目の cross-review の ACTION_REQUIRED 1 件を受けた回で、cap の最後)。新しい run を先頭に置き、run 3、cycle 2 の最初の run、cycle 1 は履歴として残した
- 先行 report: self-review cycle 3 は `eaa9bc71`(Merge yes、LOW 2、C3-1・C3-2)、verify cycle 3 は `c693d220`(partial-pass、V22〜V27)、test cycle 3 は `36d80504`(pass、`run-test.sh` が rc 0)。3 回目の cross-review の triage は `3d8b5584`(ACTION_REQUIRED 1、WORTH_CONSIDERING 1)。その直しは `04d3d986`、cycle 3 の self-review の直しは `e6f98624`、verify の V26 の直しは `91578db5`(どれも `leader.md` とそのテストの 2 ファイルだけ)
- 見た範囲: verify cycle 3 の「Documentation drift」の表、「tech-debt の Trigger と数」、V22〜V27、self-review cycle 3 の C3-1・C3-2、triage の 3 回目の節の finding 1・2、test report の Cycle 3 の節。`leader.md` の座席の確認の項(`:162-216`)と、`91578db5` の diff を読んだ。文書の各文と数は HEAD の `leader.md`、テスト、`wc -l`、`git diff --numstat` と突き合わせた
- 制約: `go test ./...` と `run-test.sh` は回していない(handoff の指示)。Go のコードとコメントは変えていない(handoff の指示)
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `d812689e` を含み、あとの report の commit は docs だけなので、docs の外の数は変わらない): `git diff --shortstat origin/main...HEAD` は 29 ファイル、+4,364/-54(計画、report、insights を含む。origin/main は fetch していない手元の `48a8a661`)。docs の外は 19 ファイル、+2,907/-40。docs の外でテストを除くと 15 ファイル、+896/-40(Go の本体 4 ファイルが +632/-2: `reseat_progress.go` 582、`feature.go` +27/-1、`reseat.go` +16、`org_reseat.go` +7/-1。`leader.md` が +154。skill の 8 つの写しが +108/-36、AGENTS.md と README が 1 行ずつ)。Go のテストは 4 ファイルで +2,011(`reseat_progress_test.go` 969、`prompts_leader_reseat_test.go` 627、`reseat_progress_start_test.go` 411、`org_reseat_test.go` +4)。walkthrough は書いていない

## Cycle 3 (cycle-count 3, cap 3)

### Summary

verify の V22(仕様の Status)、V23(行 520 の (e))、V24(新しい行 526)、V25(数と Trigger の注)、V27(計画の Progress)を直した。V26(規則の新しい分岐の縁)は、`91578db5` が規則を直したあとの形で行 520 の (e) に書いた。Go、`leader.md`、skill、README、AGENTS.md、`ralph org reseat --help`、org-runtime の仕様は変えていない(下の「確かめたこと」)。

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-11-12-org-reseat.md` | Status に 2026-11-16 の 4 回目の段落(見つけた返事は捨てない、済んだと確かめられる手だけ飛ばす、例は計画の置き場所だけ、時刻の例と直しの commit の例は外した、済んだかが分からない直しは重なっても頼む、`RESUME:` の件は tech-debt に回した、パイプラインの結果、実機を見ていないこと)。3 回目の段落の (2) に「この枝は 4 回目のパイプラインで外した(下の段落)」の 1 句。FR-9、FR-10、AC の印は変えていない |
| `docs/tech-debt/README.md` | 下の「tech-debt」(6 行を直し、1 行を足した) |
| `docs/plans/active/2026-11-15-reseat-leader-progress.md` | Progress の sub-bullet 2 つ(test cycle 3、この pass)と、3 つの箱の行に cycle 3 の一言。cycle 3 の self-review(`eaa9bc71`)とその直し、verify(`c693d220`)と V26 の直しの sub-bullet は先の run が足していたので、足し直していない。計画の本文と AC の箱は変えていない。digest は編集の前後とも `de0cca0b6f78` |
| `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`、`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` | この report と insight event |

commit は 2 つ。内容の commit `d812689e`(3 ファイル、+15/-10)と、この report と insights の commit。

### tech-debt

行番号は commit `d812689e` のもの。既存の行は先頭の語でも探せる。

| 行(先頭の語) | 何をしたか |
|----|----|
| 520「Known gaps of feat/reseat-leader-progress (#312c-2), about the rule that `internal/org/prompts/leader.md` gives a re-seated leader」 | (e) を現行の規則に合わせて 4 つのセルと Related で書き直した。`d85fc0ef` の枝(返事より後の時刻の TASK の行があれば返事を扱い直さない)を `04d3d986` が外し、`e6f98624` と `91578db5` が残った 2 つの例(直しの commit、時刻)を外した経緯を 1 文にし、旧 (1)(同じ秒)と旧 (2)(「どの座席でも後の TASK」)は枝と一緒に消えたと書いた。新しい (1) は、済んだと確かめられる手だけ飛ばすので、前の leader が頼んだ直しを再起動した leader がもう一度頼みうる(同じ直しの TASK が 2 回、報告側)。何が「済んだ」に数えられるかは leader の読みで、規則が挙げる例は計画の置き場所だけ(V26 (d))。新しい (2) は旧 (3)(push のあとの `headRefOid` の遅れで BLOCKED)で、変わらない。影響のセルは「同じ直しが 2 回、または人が見る BLOCKED。どちらも報告側」、Why deferred のセルは「知らせが消える穴を塞いだ代わりの値段で、cycle 3 が cap 3 の最後」、Trigger のセルは「直しを 2 回頼んだ報告か、push のあとの BLOCKED。節が TASK の答える返事を示すか、gate の項目が head を待って読み直すかを決める」。Related に verify の V23・V26 と、3 回目の triage の finding 1 |
| 522「The section 「agmsg の受信箱の運用」 of `internal/org/prompts/leader.md`...」 | Trigger の末尾に 2026-11-16(cycle 3)の注: `04d3d986`、`e6f98624`、`91578db5` が座席の確認の項目をまた書き換えたので当たったが、`leader.md` と 2 つのテストを変える直しなので払っていない。偽の文は今も「agmsg の受信箱の運用」の節にあり、`prompts_leader_inbox_test.go` が固定している。Related の verify に「cycle 3 でも当たった」 |
| 365「`internal/org/verbs.go` is 2277 lines」 | 持ち越しの注の `prompts_leader_reseat_test.go` を 615 から 627 に、数えた commit を `d85fc0ef` から `91578db5` に。615 は `d85fc0ef` の数として残し、cycle 3 の 3 つの直しが 627 にしたと書いた。ほかの 6 ファイルは cycle 3 が触っていない |
| 426「The role prompts grew with org-cross-review」 | 注の `leader.md` を 1,547(+156)から 1,545(+154)、テストを 615 から 627 に。数は `91578db5` の `wc -l`(`git diff --numstat 48a8a661..HEAD` は 154 行足して 0 行消す。1,391 + 154 = 1,545)。cycle 3 の 3 つの直しは座席の確認の項を書き換えて 2 行減らしたと書き、1,547 は `d85fc0ef` の値として残した |
| 514「`internal/org/reseat.go` (1,307 lines)」 | `prompts_leader_reseat_test.go` の 615 を 627 に。数の基準の文は、`reseat_progress.go` と `reseat_progress_test.go` は `d85fc0ef`、`prompts_leader_reseat_test.go` は `91578db5`(最後に Go を変えた commit)と書き直した。Trigger の `reseat.go`、`reseat_test.go`、`startFeatureInput` は cycle 3 が触っていないので、新しい持ち越しの注は足していない |
| 524「`internal/org/reseat_progress_test.go`, new in feat/reseat-leader-progress (#312c-2), is 969 lines」 | 「`prompts_leader_reseat_test.go` 615」を 627(`91578db5`、`d85fc0ef` で 615)に。Why deferred の「its third run, after `d85fc0ef`, is the last」は、3 回目の run の cross-review が cycle 3 を生んだので、「最後と見ていたが、cycle 3 の 3 つの直しは `leader.md` とテストだけで、このファイルを触っていない。cycle 3 が cap 3 の最後」に直した。handoff の指示になく、掃引で見つけた。ファイルは触られていないので、Trigger の注は足していない |
| 526(新)「The seat-check item of the re-seat rule in `internal/org/prompts/leader.md`...sends an implementer slice again with no `RESUME:` line」 | V24(3 回目の cross-review の finding 2、WORTH_CONSIDERING)。座席の確認が implementer のスライスを計画と「branch の commit」の行から送り直す 3 つの道(その座席の行が無い、TASK_ID が `不明`、答えのない入力待ちで commit の行がスライスの済みを示さない)に `RESUME:` が無く、reseat が commit の前に止めた implementer の未コミットの変更を、新しい implementer が前からあった変更と読んで BLOCKED を返す(`implementer.md`)。leader 自身が BLOCKED のあとに頼み直すときは `git status --porcelain` から `RESUME:` を並べる(`leader.md` の使命の項)。LOW(報告側で何も消えない)。直さない理由は、triage の線引き(知らせる側に倒れる穴は tech-debt と Known gaps)と、cycle 3 が最後で直すと `/self-review` からやり直しになること。Trigger は座席の確認の送り直しを次に触るとき、または reseat された implementer が自分の変更で BLOCKED を返した報告。直し方は、送り直す前に `git status --porcelain` の scope の中のパスを `RESUME:` に並べ、`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckItem` に固定する。Related に triage、verify の V24、計画 |

触った行(上の 7 行)は 5 つのセルで、列の数は 5。新しい `file:line` の引用は 0 件。計画の参照は `docs/plans/active/2026-11-15-reseat-leader-progress.md` の 1 つで、ある。バッククォートと `~~` の数、U+FFFD(0 件)も確かめた。

### 掃引

Trigger のセルに `leader.md`、`prompts_leader_reseat_test`、`seat-check`、`座席の確認`、`RESUME`、`history.sh`、`headRefOid`、`prompts_leader`、`RenderRolePrompt`、`feature-org`、`機能ごとの org` の語を持つ行を python で挙げた(列は ` | ` で分けた)。16 行で、`91578db5` までの 3 つの直しの hunk(`leader.md` は座席の確認の項の中の 3 回の書き換え、テスト)と突き合わせて 1 行ずつ判断した。

| 行(先頭の語) | 判断 |
|----|----|
| 520、522 | 当たる。上の「tech-debt」のとおり |
| 365、426 | 365 の Trigger は「leader template tests」で当たる(注は既にあり、数を直した)。426 の Trigger は「節を足すとき」で、cycle 3 は節を足していない(数と経緯だけ直した) |
| 514 | 数だけ。Trigger は `reseat.go` などで、触っていない |
| 519 | Trigger は `reseat_progress.go` と gate の行を読む規則。gate の項目は 1 行も変わらない(`leader.md` の hunk は座席の確認の項の中だけ) |
| 188(a)、200(g)、377、384、427、508、197(取り消し線)、379(取り消し線) | `leader.md` の使命の項 2 や手順 1・3・4・5・14、`### cross-review` の小節、hook の bullet を名指す。cycle 3 の hunk は座席の確認の項だけで、当たらない。200 の (g) は払い済み(f1b0352e)で、この branch の先の cycle も注を足していない。188 の (a) は「leader prompt」を名指すが、使命の項 2 と手順 3・5・14 の `--driver` の渡し方は触っていない |
| 195、513 | `feature-org` と `RenderRolePrompt` の語の一致。Trigger の本文は `ralph org status`、`reseatNoActiveSeatErr` などで、cycle 3 は触っていない |

この掃引の語に当たらない行のうち、verify が見た 521・523・525(Go の本体を触っていない)と 524(Trigger は `reseat_progress_test.go`、触っていない。数だけ直した)も、当たらない。

この pass 自身が編集するファイル(`docs/specs/2026-11-12-org-reseat.md`、`docs/tech-debt/README.md`、計画)を名指す Trigger は、cycle 2 と run 3 の掃引で見ており、新しい Trigger は足していない。

### 直さなかったこと

- 行 522 の偽の文、行 524 のファイル分割、行 525 の名前、行 526 の `RESUME:`: Go か `leader.md` を変えるので、cap の最後の回では払わず、行に書いた
- FR-9、FR-10、AC の印: handoff の指示で変えていない。4 回目の段落は FR の文を変えないと書いた
- README、AGENTS.md、`/org` と `/director` の skill(各 4 つの写し)、`ralph org reseat --help` とその pin、`docs/specs/2026-08-01-org-runtime.md` の改訂 (a): 変えていない。節の 8 項目を列挙するだけで、座席の確認が見つけた返事をどう扱うかを述べない(下の「確かめたこと」)
- 計画の Progress の並び順: 3 回目の cross-review と cycle 3 の self-review・verify の sub-bullet(`b79f300e` などが足した行)は、先の run の途中に入っている。順に並べ直すと既存の行を動かすので、handoff の「足りない行を足す」の範囲に収めて、cycle 3 の行をその直後に足した
- 行 522・523・524・525 の既存の注にある「the third run is the last under the cap of 3」: 当時の事実を書いた日付つきの注なので残した。新しい注に cycle 3 が最後と書いた(行 524 の Why deferred だけは、今を言う文だったので直した)

### `/pr` に渡す文

Known gaps に載せる 4 件。V16 と V17 は run 3 の report と同じ、(e) は書き直した形、`RESUME:` の行は新しい。

- V16(INFO)(a)(b): `leader.md` の規則に書かれていない縁。(a) 座席の行の TASK_ID が `-`(`--raw` の TASK で `TASK_ID:` が無い)のとき、同じ TASK_ID を履歴で探して見つからず、送り直しに進む。(b)「branch の commit」の行は読めなかったときも同じ見出しで出るので、計画項目の「commit の行があって」を満たしてしまう(計画項目は `git log --name-only` で自分でも調べるので害は小さい)。(c) は `d85fc0ef` の枝と一緒に消えた
- V17(INFO): 計画の D6 と AC2 の括弧書きは、節ができる条件に座席へ送った TASK と上げた RESULT を含まない。実装は両方で節を作る(`reseatProgressSection` の `add`)。計画の本文は変えない記録の書き方
- 行 520 の (e)(V21 の書き直し、V26): 座席の確認は見つけた返事を捨てず、済んだと確かめられる手だけ飛ばす。そのため前の leader が頼んだ直しを、再起動した leader がもう一度頼みうる(同じ直しの TASK が 2 回、報告側)。gate の項目で push のあとに GitHub が `headRefOid` を返すのが遅れると BLOCKED になる(人に渡る向き)。どちらも実機の通し(行 520 の (d)、(e))で見る
- 行 526(V24、3 回目の cross-review の finding 2、WORTH_CONSIDERING): 座席の確認が implementer のスライスを送り直すとき `RESUME:` の行が無い。reseat が commit の前に止めた implementer の未コミットの変更を、新しい implementer が前からあった変更と読んで BLOCKED を返す。何も消えず、報告側。直し方は、送り直す前に `git status --porcelain` の scope の中のパスを `RESUME:` に並べること

### 前の run の記述の訂正

| run 3 の記述 | cycle 3 の事実 |
|----|----|
| 行 365・426・514・524 の数: `prompts_leader_reseat_test.go` 615、`leader.md` 1,547(+156) | 627、1,545(+154)(`91578db5`)。行 426 の「最後に変えた commit」は `d85fc0ef` から `91578db5` |
| 行 520 の (e): 3 つの縁((1) 同じ秒の返事と TASK、(2) 「どの座席でも後の TASK が 1 つあれば扱った」の限界、(3) `headRefOid` の遅れ) | (1) と (2) は `04d3d986` が枝を外したので消えた。新しい規則の縁(同じ直しを 2 回頼みうる)と、旧 (3) の 2 つ |
| 行 524 の Why deferred: 「its third run, after `d85fc0ef`, is the last」 | cycle 3 がある。cycle 3 が cap 3 の最後で、このファイルは触っていない |
| 仕様の Status 3 回目の段落の (2): 返事より後の時刻の行が節に 1 つでもあれば扱い直さない | 4 回目で外した。(2) に 1 句と、4 回目の段落 |
| run 3 の `/pr` に渡す文: V21 の (a)(b)(c) | (a)(b) は枝と一緒に消えた。(c) は行 520 の (e) の (2) として残る。V26 の (a)〜(d) のうち、(a)(b)(c) は `91578db5` が時刻の例を外したので消え、(d) は「何が済んだと数えられるかは leader の読み」として (e) の (1) に入れた |
| 差分の大きさ: 29 ファイル、+3,995/-54。docs の外は 19 ファイル、+2,897/-40。テストを除くと 15 ファイル、+898/-40 | 29 ファイル、+4,364/-54。+2,907/-40。15 ファイル、+896/-40(`leader.md` が +156 から +154、`prompts_leader_reseat_test.go` が 615 から 627) |

### 確かめたこと(文書が実装と合っていた所)

- 仕様の Status の 4 回目の段落の 4 点は `leader.md`(`:162-216`)と合う: 見つけた返事は「前の leader が扱ったかどうかにかかわらず、今届いたものとして扱います」「返事を捨てることはしません」(`:196-199`)。応じる手ごとに済んでいないかを確かめ、「済んだと言えるのは、確かめられるものだけです」の例は計画の置き場所だけ(`:199-202`)。済んだかが分からない手は「重なってもかまわないので頼みます」(`:202-203`)。時刻の比べと直しの commit の例は本文に無い(`git grep` で `あとの時刻`、`直しの commit`、`時刻を比べ` は `leader.md` に 0 件)
- `91578db5` の diff は `leader.md` が 9 行(+4/-5)の 1 か所で、座席の確認の項の中だけ。gate の項目は動いていない
- README の `ralph org reseat` の段落、AGENTS.md の `reseat_progress.go` の項、`/org` skill の「座席の立て直し」の段落、`/director` skill の「座席を立て直す」の段落、`ralph org reseat --help` の節の列挙、`docs/specs/2026-08-01-org-runtime.md` の改訂 (a)は、節の 8 項目の列挙で、座席の確認の返事の扱いを述べない。`git grep` で `扱い直さ`、`あとの時刻`、`後の時刻`、`later TASK`、`扱い済み`、`直しの commit`、`時刻を比べ` を README、AGENTS.md、skill(`.claude`、`.agents`、`templates/base`)、`internal/cli`、`internal/org/prompts`、org-runtime の仕様、`docs/quality`、`docs/architecture`、`docs/recipes` で探し、当たったのは `/org` skill の `inbox` の行(「最後の時刻」)だけで、座席の確認とは無関係
- `git grep` で `headRefOid`、`座席の確認`、`seat check`、`seat-check` を同じ範囲で探した。`headRefOid` は `/org` skill の `cleanup` の行と `internal/cli/org_cleanup.go` の別の話。`座席の確認` は org-runtime の仕様の `:125` が、節があるときの 5 項目の名前(座席の確認、計画、gate、受信箱、座席)として挙げるだけで、返事の扱いを述べていない。`seat check` と `seat-check` は 0 件
- `prompts_leader_reseat_test.go` は 627 行、`leader.md` は 1,545 行(`wc -l`)。`reseat_progress.go` 582、`reseat_progress_test.go` 969、`reseat_progress_start_test.go` 411 は `c49a6bc5` から変わっていない(`git diff --stat c49a6bc5..HEAD -- . ':!docs'` は `leader.md` とそのテストの 2 ファイル)

### Size check

この branch が新しく作った Go ファイルのうち、800 行を超えたのは `reseat_progress_test.go` の 969 行だけで、行 524 と行 514 が持つ。ほかの新しいファイルは 800 行以内: `reseat_progress.go` 582、`reseat_progress_start_test.go` 411、`prompts_leader_reseat_test.go` 627。この branch が変えた既存の Go ファイルは、変える前から 800 行を超えていて、行 514 が持つ: `reseat.go` 1,553、`org_reseat_test.go` 927、`feature.go` 874。cycle 3 はこの 3 つを変えていない。`org_reseat.go` 527 は 800 行以内。`leader.md` 1,545 は Go でなく、行 426 が持つ。

### Verification

- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` は編集の前後とも `de0cca0b6f78`(前は `git show 36d80504:<計画>` を scratch に出して数えた)
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(rc 0)、`./scripts/check-skill-sync.sh` は 14 skill で ok(rc 0)
- `go build ./...` は成功(rc 0)、`go vet ./internal/org/ ./internal/cli/` は出力なし(rc 0)、`gofmt -l internal/` は出力なし、`git diff --check` は rc 0
- 変えた 3 ファイルの U+FFFD は 0 件
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)。natural-japanese の lint は hook が書き込みごとに回し、仕様と計画と tech-debt への指摘は表示されなかった

### Insight event

```
./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` に足した行は cycle 3、pass、ts は 2026-11-15T21:20:37Z(UTC の日付が 2026-11-15 なので、既存の file に足された。日本時間では 2026-11-16 06:20)。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。

### 残っていること(lead へ)

- `/pr` の Known gaps に V16 (a)(b)、V17、行 520 の (e)、行 526 の `RESUME:`(上の「`/pr` に渡す文」)
- 払っていないもの: 行 519〜526 の Trigger の注と、行 525(R3-2)、行 526 は開いたまま。直すには Go か `leader.md` を変えるので、`/self-review` からやり直しになり、cap 3 に届いている
- 節を読んだ leader の実機での動き(返事の扱い、PR の head、`不明` の分岐を含む)は見ていない(行 520 の (d)、(e)、仕様 AC-12)
- AC1〜AC9 の箱は `[ ]` のまま(`/pr` の前に付ける)。計画の Progress は、3 回目の cross-review から cycle 3 の行までが先の run の途中に入っていて並びが時系列でない(上の「直さなかったこと」)
- cross-review の次の run が ACTION_REQUIRED を出したら、cap を上げて回し直すかは lead の判断(ユーザーの規則は上限に届いたら上げて回し直す)

---

## Run 3(履歴)

下は run 3(2026-11-16、run 開始時の HEAD `601b4643`、内容の commit `0d443ebb`)の report で、そのまま残した。数と記述のうち cycle 3 で変わったものは、上の「前の run の記述の訂正」にある。

- Date: 2026-11-16(run 3。cycle 2 の最初の run と cycle 1 も同じ日)
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(承認の digest `de0cca0b6f78`)。issue #312 の #312c-2
- Branch: `feat/reseat-leader-progress`。run 開始時の HEAD は `601b4643`(run 3 の test report の commit。report の中の `HEAD:` の行の値ではない)。内容の commit は `0d443ebb`
- Pipeline run: 3 回目(`cycle-count.json` は `cycle` 2、`cap` 3。cycle 2 の cross-review の ACTION_REQUIRED 3 件を受けて cap を 2 から 3 に上げた回で、cap の最後)。新しい run を先頭に置き、cycle 2 の最初の run と cycle 1 は履歴として残した
- 先行 report: self-review run 3 は `9ea00d76`(Merge yes、LOW 2、R3-1・R3-2)、verify run 3 は `c49a6bc5`(partial-pass、V18〜V21)、test run 3 は `601b4643`(pass)。cycle 2 の cross-review の triage は `51119734`(ACTION_REQUIRED 3 件)、その直しは `d85fc0ef`(4 ファイル、+307/-46)
- 見た範囲: verify run 3 の「Documentation drift」の表と「tech-debt の Trigger と数」、V18〜V21、self-review run 3 の R3-1・R3-2。`d85fc0ef` の diff は 4 ファイルとも読んだ。文書の各文と数は HEAD の `reseat_progress.go`、`leader.md`、`wc -l` と突き合わせた
- 制約: `go test ./...` と `run-test.sh` は回していない(handoff の指示)。Go のコードとコメントは変えていない(handoff の指示)
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `0d443ebb` を含み、あとの report の commit は docs だけなので、docs の外の数は変わらない): `git diff --shortstat origin/main...HEAD` は 29 ファイル、+3,995/-54(計画、report、insights を含む。origin/main は fetch していない手元の `48a8a661`)。docs の外は 19 ファイル、+2,897/-40。docs の外でテストを除くと 15 ファイル、+898/-40(Go の本体 4 ファイルが +632/-2: `reseat_progress.go` 582、`feature.go` +27/-1、`reseat.go` +16、`org_reseat.go` +7/-1。`leader.md` が +156。skill の 8 つの写しが +108/-36、AGENTS.md と README が 1 行ずつ)。Go のテストは 4 ファイルで +1,999(`reseat_progress_test.go` 969、`prompts_leader_reseat_test.go` 615、`reseat_progress_start_test.go` 411、`org_reseat_test.go` +4)。walkthrough は書いていない

## Run 3 (cycle-count 2, cap 3)

### Summary

verify の V18(tech-debt の数と Trigger)、V19(仕様の Status と FR-9)、V20(計画の Progress)を直した。V21(規則に書かれていない縁 3 つ)は行 520 の (e) にし、`/pr` の Known gaps にも回す。self-review の R3-1 は V18 と同じ直しで済み、R3-2 は新しい行 525 にした。Go、skill、README、AGENTS.md は変えていない(下の「確かめたこと」)。

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 下の「tech-debt」(9 行を直し、1 行を足した) |
| `docs/specs/2026-11-12-org-reseat.md` | Status に 2026-11-16 の 3 回目の段落(PR の head の比べ、扱い済みの返事の読み飛ばし、`不明` の行、FR-10 は要約で文は変えないこと、パイプラインの結果と実機を見ていないこと)。FR-9 の 2026-11-16 追記の中に、`不明` の 1 句。FR-10 は gate の PR の分岐の細かさを書いていないので、変えていない |
| `docs/plans/active/2026-11-15-reseat-leader-progress.md` | Progress の sub-bullet 6 つ(cross-review と cap の引き上げ、`d85fc0ef`、self-review run 3、verify run 3、test run 3、この pass)と、3 つの箱の行に run 3 の一言。計画の本文と AC の箱は変えていない。digest は編集の前後とも `de0cca0b6f78` |
| `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`、`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` | この report と insight event |

commit は 2 つ。内容の commit `0d443ebb`(3 ファイル、+21/-12)と、この report と insights の commit。

### tech-debt

行番号は commit `0d443ebb` のもの。既存の行は先頭の語でも探せる。

| 行(先頭の語) | 何をしたか |
|----|----|
| 519「Small findings of feat/reseat-leader-progress (#312c-2), the first run of its self-review」 | (a) の「103 lines」を 107 行に直した(`reseatProgressSection` は `reseat_progress.go` の 305〜411 行、`d85fc0ef` の不明の行の分岐が 4 行足した。`funcs.py` で数え直した)。Trigger の末尾に、`d85fc0ef` が `reseatProgressSentTask`、不明の行、gate の行を読む規則に当たったが、cycle 2 の注と同じ数と状態の直しで、(a) から (d) は払っておらず、(d) は gate の行だけが残ると書いた。Related に R3-1 と V18 |
| 520「Known gaps of feat/reseat-leader-progress (#312c-2), about the rule that `internal/org/prompts/leader.md` gives a re-seated leader」 | (b) の RESOLVED のあと(取り消し線の外)に、`d85fc0ef` が (a)(b) の読みを変えた 1 句(PR の `headRefOid` を gate の行の `head` と比べる。head が同じ OPEN か MERGED なら PR は作らない、head が違う OPEN は push と読み直し、head が違う MERGED と CLOSED だけは BLOCKED)。新しい (e) を 4 つのセルに足した(V21 の 3 点: 同じ秒の返事と TASK、別の座席への TASK を前の leader が返事を読む前に送った場合、push のあとの `headRefOid` の遅れ)。(e) の Trigger は (d) の実機の通しと、返事の二重処理・取りこぼし・push が通ったあとの BLOCKED の報告。Related に V18・V21 と、cycle 2 の triage の finding 1 と 2(`git show 51119734:...`) |
| 522「The section 「agmsg の受信箱の運用」 of `internal/org/prompts/leader.md`...」 | Trigger の末尾に 2026-11-16 の注: `d85fc0ef` が座席の確認(不明の分岐と返事の時刻の比べ)を書き換えたので当たったが、`leader.md` と 2 つのテストを変える直しなので払っていない。偽の文は今も「agmsg の受信箱の運用」の節にあり(`leader.md` 1441 行目付近)、`prompts_leader_inbox_test.go` が固定している。座席の確認の項目は今も正しいことを言う。Related の triage の「(finding 1)」は cycle 1 の triage の finding 1 だった(`git show 0d660e77:...`)。`51119734` がこのファイルを cycle 2 の triage に書き換えたので、cycle 1 の commit に固定した。handoff の指示にはなく、掃引で見つけた。Related の verify に「run 3 で Trigger に当たった」 |
| 523「`TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites`...」 | Trigger の末尾に 2026-11-16 の注: `d85fc0ef` が `reseatProgressSentTask` の戻り値を 3 つにした(TASK_ID、TASK かどうか、TASK_ID が不明か)ので当たったが、定数を共有すると `verbs.go` と `send_fallback.go` を変え、計画の Scope にもなく、cap の最後の回なので払っていない。切れ目の印は `TestReseatProgress_TheTruncatedMarkerIsTheOneTruncateForDetailsWrites` が本物の `truncateForDetails` につないでいる。`raw=true ` と `submit_unconfirmed=true ` は文字列のまま。Related の verify に「run 3 で Trigger に当たった」 |
| 524「`internal/org/reseat_progress_test.go`, new in feat/reseat-leader-progress (#312c-2), is 869 lines」 | 数を直した: 869 行を 969 行(`wc -l` は `d85fc0ef`、最後に変えた commit。`b8a9bd38` で 869、`d85fc0ef` が 100 足した)、「約 260 行」を約 360 行(`progressSentEvent` から終わりまで 361 行)、`reseat_progress.go` 551 を 582、`prompts_leader_reseat_test.go` 516 を 615。分割後の約 610 行は変わらない(分割後の元のファイルは 604〜608 行)。Why deferred の「cap of 2」は、V15 を見つけた時点の文として残し、cap を 3 に上げたことと、3 回目の run が最後であることを足した。Trigger は「`ec10ba6f` と `d85fc0ef` が足した」に直し、`d85fc0ef` が当たった払っていない注を足した。Related に V18 と、この report の Run 3 の size check |
| 365「`internal/org/verbs.go` is 2277 lines」 | 持ち越しの注の `prompts_leader_reseat_test.go` を 516 から 615 に、数えた commit を `b8a9bd38` から `d85fc0ef` に。ほかの 6 ファイル(`verbs.go` など)は `d85fc0ef` が触っていないので数は同じで、そう書いた |
| 426「The role prompts grew with org-cross-review」 | 注の `leader.md` を 1,516(+125)から 1,547(+156)、テストを 516 から 615 に。数は `d85fc0ef`(両ファイルを最後に変えた commit)の `wc -l`。1,391 + 156 = 1,547 |
| 514「`internal/org/reseat.go` (1,307 lines)」 | 2026-11-16 の注の数を直した: `reseat_progress.go` 551 → 582、`reseat_progress_test.go` 869 → 969、`prompts_leader_reseat_test.go` 516 → 615。数の基準の文は、この 3 ファイルだけ `d85fc0ef`(最後に Go を変えた commit)で、`reseat.go`、`feature.go`、`org_reseat_test.go`、`org_reseat.go` は cycle 2 の sync-docs の数のまま、と書き直した。`d85fc0ef` はこの行の Trigger の `reseat.go`、`reseat_test.go`、`startFeatureInput` を触っていないので、新しい持ち越しの注は足していない |
| 525(新)「`reseatProgressUnknownTaskID` and the field `TaskIDUnknown`...」 | R3-2。定数(語 `不明`)と bool の名前が語順だけ違うこと、定数の doc が「the marker」と呼び、数行上の `reseatProgressTruncatedMarker` も「the marker」であること、`TestReseatProgress_TheTruncatedMarkerIsTheOneTruncateForDetailsWrites` が名前に出ない `不明` の定数と行の形も固定すること。直し方は定数を `reseatProgressUnknownTaskIDWord` にして doc を「the word」にし、テストを 2 つに分けるか名前を直す。Trigger は最後の TASK の項目(`reseatProgressTask`、`reseatProgressSentTask`、不明の行)を次に触るとき |

触った行(上の 9 行)は 5 つのセルで、バッククォートと `~~` はセルごとに偶数。新しい `file:line` の引用は 0 件(行 522 の `leader.md` 1441 行目付近はこの report だけで、行には書いていない)。計画の参照は 132 件で、無い参照は 0 件(`check_tech_debt_plan_refs` と同じ式)。HEAD との差は、行 519〜524 と 365・426・514 が挿入と数の置き換えだけ、行 525 が追加だけ(`tdcheck.py`。末尾の空行が行 525 に見えるので、列の数の比較で 1 件の「悪い」が出るが、これは新しい行を足したための見かけ)。

### 掃引

`my_sweep.py` で `d85fc0ef` の diff(`leader.md`、`reseat_progress.go`、テスト 2 つ)を、開いている行の Trigger(取り消し線と日付つきの注の前)と突き合わせた。候補は 32 行(出力 64 行)で、handoff が挙げた 519・520・522・523・524 は外してある。514 は数だけ直す行で、下の表には入れていない。32 行を 1 つずつ読んだ。

| 行(先頭の語) | 判断 |
|----|----|
| 377、384、427、508、197 | `leader.md` の手順 1・4・14、`### cross-review` の小節、hook の bullet を名指す。`d85fc0ef` の `leader.md` の hunk は 2 つで、どちらも「機能ごとの org」の導入の規則の中(座席の確認、gate の項目)。手順の本文は変えていない。cycle 2 と同じ判断(hunk を読んだ) |
| 200(f)(g)(j) | `leader.md` の編集で当たる (f)(g)(j) は払い済みで、開いている項目の Trigger は `StartFeatureResult` などを名指す |
| 239(o)(p)(q) | 手順 9・10 と `TestRenderRolePrompt_Leader_FeatureOrgGate` の編集を待つ。`d85fc0ef` は触っていない |
| 521(a) | Trigger は `startProgressSection` の呼び出しと `lastDisbandedIndexes`。`d85fc0ef` の `readReseatProgress` の hunk は `lastTask` の代入の 1 行だけで、どちらでもない |
| 426 | 文字どおりの Trigger は「`leader.md` に節を足すとき」で、`d85fc0ef` は節を足していない。数だけ直し、cycle 2 の持ち越しの注は変えていない |
| 110、138、192、218、248、250、358、366、374、376、392、398、457〜459、483、495、497、500、504、513、515 | `Fatalf`、`unknown`、`internal/org`、`docs/reports`、`/self-review` などの語の一致で、Trigger の本文は `d85fc0ef` が触らない関数とファイルを名指す(515 は `PlanReseat` など) |

この pass 自身が編集するファイル(`docs/specs/2026-11-12-org-reseat.md`、`docs/tech-debt/README.md`、計画)を名指す Trigger も探した(`trig_files.py`)。行 177 は別の仕様、行 371 は 5 つの recipe の編集で、当たらない。

### 直さなかったこと

- FR-10: gate の分岐の細かさ(PR の head の分岐)を書いていないので、足していない。Status の段落に、FR-10 の「PR の手順に進む」が分岐を含む要約で、文は変えないと書いた(V19 の INFO)
- README、AGENTS.md、`/org` と `/director` の skill(各 4 つの写し)、`ralph org reseat --help` とその pin、`docs/specs/2026-08-01-org-runtime.md` の改訂 (a)、Go のコメント: 変えていない。下の「確かめたこと」
- 行 522 の偽の文、行 523 の定数の共有、行 524 のファイル分割、行 525 の名前: どれも Go か `leader.md` を変えるので、cap の最後の run では払わず、行に書いた
- 計画の `## Deviation notes`: handoff が Progress だけと指示したので足していない。cap の引き上げは Progress に書いた
- AC の箱(`[ ]`)と D6・AC2 の括弧書き(V17): 変えていない(`/pr` の前に付ける。計画の本文は書き換えない)

### `/pr` に渡す文

Known gaps に載せる 3 件(V16 と V17 は cycle 2 の report と同じ、V21 が新しい)。

- V16(INFO): `leader.md` の規則に書かれていない縁が 3 つある。(a) 座席の行の TASK_ID が `-`(`--raw` の TASK で `TASK_ID:` が無い)のとき、同じ TASK_ID を履歴で探して見つからず、送り直しに進む。(b)「branch の commit」の行は読めなかったときも同じ見出しで出るので、計画項目の「commit の行があって」を満たしてしまう(計画項目は `git log --name-only` で自分でも調べるので害は小さい)。(c) 座席ごとの最後の TASK に RESULT が見つかっても、前の leader が処理して次の座席へ TASK を送ったあとかは、時刻を比べないと分からない(cycle 2 の report の文)。(c) は `d85fc0ef` の返事の時刻の比べで扱われ、残った縁は V21 に移った。verify run 3 は「V16 の 3 つは変わらない」と書いたが、(c) の元の文は cross-review の指摘 2 の直しで古くなったので、(a) と (b) だけを載せ、(c) は V21 に含めるのがよい
- V17(INFO): 計画の D6 と AC2 の括弧書きは、節ができる条件に座席へ送った TASK と上げた RESULT を含まない。実装は両方で節を作る(`reseatProgressSection` の `add`)。計画の本文は変えない記録の書き方
- V21(INFO): `d85fc0ef` が足した規則の分岐に、書かれていない縁が 3 つある。(a) 返事と同じ秒の TASK の行は「あと」に数えず、扱い済みの返事を新しく届いたものと読む。(b)「あとの行が 1 つでもあれば扱い済み」は、前の leader が別の座席への TASK を返事を読む前に送っていた場合を区別できず、失敗の返事の扱いが落ちうる。(c) gate の項目で push のあとに GitHub が `headRefOid` を返すのが遅れると BLOCKED になる(人に渡る向き)。実機の通し(tech-debt 行 520 の (d)、(e))で見る

### 前の run の記述の訂正

| cycle 2 の記述 | run 3 の事実 |
|----|----|
| 行 519 (a) の 103 行 | 107 行 |
| 行 514 の数: `reseat_progress.go` 551、`reseat_progress_test.go` 869、`prompts_leader_reseat_test.go` 516 | 582、969、615(`d85fc0ef`) |
| 行 365 と 426 の数: `prompts_leader_reseat_test.go` 516、`leader.md` 1,516(+125) | 615、1,547(+156) |
| 行 524: 869 行、約 260 行、「cap of 2」 | 969 行、約 360 行、cap 3 の最後の回。約 610 行(分割後)は変わらない |
| 行 520 (a)(b) の RESOLVED(`ec10ba6f` の読み) | `d85fc0ef` で PR の head を比べる形に変わった。1 句を足した |
| 行 522 の Related「(finding 1)」(triage) | cycle 1 の triage の finding 1。`51119734` で triage が書き換わったので、`0d660e77` に固定した |
| 差分の大きさ: 29 ファイル、+3,342/-54。docs の外でテストを除くと 15 ファイル、+836/-40 | 29 ファイル、+3,995/-54。15 ファイル、+898/-40 |

### 確かめたこと(文書が実装と合っていた所)

- 仕様の Status の 3 点は `leader.md` の文と合う: gate の項目は 5 つの場合(head が同じ OPEN か MERGED、head が違う OPEN、head が違う MERGED、CLOSED だけ、PR なし)で、head が違う OPEN は手順 14 の `wip:` の確かめと push のあと一覧を読み直す。座席の確認は、返事の時刻より後の行が 1 つでもあれば扱い済み。不明の行の形は `reseat_progress.go` の `Sprintf`(`- %s: %s (記録が切れている、%s)`)と `leader.md` と仕様で同じ
- `sendDetailsMaxLen` は 200(`verbs.go`)で、仕様の「200 byte」と合う
- README の `ralph org reseat` の段落、AGENTS.md の `reseat_progress.go` の項、`/org` skill の「座席の立て直し」の段落、`/director` skill の「座席を立て直す」の段落、`ralph org reseat --help` の節の列挙、`docs/specs/2026-08-01-org-runtime.md` の改訂 (a)(いずれも節の 8 項目の列挙。`d85fc0ef` は 8 項目を変えていない)は、PR の head、扱い済みの返事、`不明` の行を述べていない。「gate が今の HEAD で pass していれば PR の手順へ」は今も正しい
- `git grep` で、`記録が切れている`、`TaskIDUnknown`、`不明 (`、`state all` を、`leader.md`・Go・tech-debt・report・仕様・計画の外で探した。skill の `state all` は `merge-order` と `cleanup` の行の別の話で、`headRefOid` も同じ
- `reseatProgressSection` は 107 行(`funcs.py`)、`readReseatProgress` は 73 行、`reseatProgressSentTask` は 19 行
- `reseat_progress_test.go` の `progressSentEvent` から終わりまでは 361 行(969 − 608)

### Size check

この branch が新しく作った Go ファイルのうち、800 行を超えたのは `reseat_progress_test.go` の 969 行だけで、行 524 と行 514 が持つ。ほかの新しいファイルは 800 行以内: `reseat_progress.go` 582、`reseat_progress_start_test.go` 411、`prompts_leader_reseat_test.go` 615。この branch が変えた既存の Go ファイルは、変える前から 800 行を超えていて、行 514 が持つ: `reseat.go` 1,553、`org_reseat_test.go` 927、`feature.go` 874。`d85fc0ef` はこの 3 つを変えていない(`git diff --stat d85fc0ef^ d85fc0ef` は 4 ファイル)。`org_reseat.go` 527 は 800 行以内。`leader.md` 1,547 は Go でなく、行 426 が持つ。

### Verification

- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` は編集の前後とも `de0cca0b6f78`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(rc 0)、`./scripts/check-skill-sync.sh` は 14 skill で ok(rc 0)
- `go build ./...` は成功、`go vet ./internal/org/ ./internal/cli/` は出力なし、`gofmt -l internal/` は出力なし、`git diff --check` は rc 0
- tech-debt: 上の「tech-debt」の末尾のとおり
- 変えた 3 ファイルの U+FFFD は 0 件
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)。natural-japanese の lint は hook が書き込みごとに回し、仕様と計画と tech-debt への指摘は出なかった(scratch の `.txt` に和欧文の半角空白の指摘が出たが、リポジトリの文書が同じ書き方なので残した)

### Insight event

```
./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` に足した行は cycle 2、pass、ts は 2026-11-15T19:59:33Z(UTC の日付が 2026-11-15 なので、既存の file に足された。日本時間では 2026-11-16 04:59)。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。

### 残っていること(lead へ)

- `/pr` の Known gaps に V16、V17、V21(上の「`/pr` に渡す文」)
- 払っていないもの: 行 519〜524 の Trigger の注と、行 525(R3-2)は開いたまま。直すには Go か `leader.md` を変えるので、`/self-review` からやり直しになり、cap 3 に届いている
- 節を読んだ leader の実機での動き(PR の head、扱い済みの返事、`不明` の分岐を含む)は見ていない(行 520 の (d)、(e)、仕様 AC-12)
- AC1〜AC9 の箱は `[ ]` のまま(`/pr` の前に付ける)
- cycle 2 の report が lead に聞いた行 465 の持ち越しの注は、この run の handoff に無く、変えていない

---

## Cycle 2(履歴)

下は cycle 2 の最初の run(2026-11-16、run 開始時の HEAD `043dbe59`、内容の commit `5aef79a4`)の report で、そのまま残した。数のうち run 3 で変わったものは、上の「前の run の記述の訂正」にある。

# sync-docs report: reseat-leader-progress

- Date: 2026-11-16(cycle 2。cycle 1 も同じ日)
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(承認の digest `de0cca0b6f78`)。issue #312 の #312c-2
- Branch: `feat/reseat-leader-progress`。run 開始時の HEAD は `043dbe59`(cycle 2 の test report の commit。report の中の `HEAD:` の行の値ではない)。内容の commit は `5aef79a4`
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2。`max_pipeline_cycles` は 2 で、この回が最後)。新しい run を先頭に置き、cycle 1 は末尾に履歴として残した
- 先行 report: self-review cycle 2 は `a02c5b5c`(Merge yes、LOW 5、C2-1〜C2-5。C2-2 と C2-3 は `b8a9bd38` で直し、計画への記録は `f6be8593`)、verify cycle 2 は `46b7993c`(partial-pass、V11〜V17)、test cycle 2 は `043dbe59`(pass)
- 見た範囲: verify cycle 2 の「Documentation drift」の表と「tech-debt の Trigger と数」、V11〜V17、self-review cycle 2 の C2-1〜C2-5。文書の各文と数は HEAD のコード(`reseat_progress.go`、`feature.go`、`leader.md`、`org_reseat.go`)と `wc -l` に突き合わせた
- 制約: `go test ./...` と `run-test.sh` は回していない(handoff の指示)。Go は 4 ファイルを変えたので、`go test ./internal/cli/ -run 'Reseat' -count=1` を回した(`ok`、33.7 秒)。ほかに `go test ./internal/org/ -run 'Skill' -count=1`(`ok`)
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `5aef79a4` を含み、あとの report の commit は docs だけなので、docs の外の数は変わらない): `git diff --shortstat origin/main...HEAD` は 29 ファイル、+3,342/-54(計画、report、insights を含む。origin/main は fetch していない手元の `48a8a661`)。docs の外は 19 ファイル、+2,636/-40。docs の外でテストを除くと 15 ファイル、+836/-40(Go の本体 4 ファイルが +601/-2: `reseat_progress.go` 551、`feature.go` +27/-1、`reseat.go` +16、`org_reseat.go` +7/-1。`leader.md` が +125。skill の 8 つの写しが +108/-36、AGENTS.md と README が 1 行ずつ)。Go のテストは 4 ファイルで +1,800。walkthrough は書いていない

## Cycle 2

### Summary

verify の V11(節の中身を列挙する 14 か所)と V12(Go のコメント 2 か所)を直し、V14 の tech-debt の数と状態を直した。V13 と C2-4 は新しい行にした。V15 は大きさの行にした。V16 と V17 は `/pr` の Known gaps に回す(下の「`/pr` に渡す文」)。行 465 の持ち越しの注は足していない(下の「直さなかったこと」)。

### Changes made

| File | Change |
|------|--------|
| `README.md`、`AGENTS.md` | 節の中身の列挙に「座席ごとの最後の TASK」(the last TASK sent to each seat)と「leader が上げた RESULT」(the RESULT items the leader raised)を足した。節の順(承認、gate、座席ごとの最後の TASK、受信箱、上げた RESULT、commit、今回の reseat、前の reseat)に並べた |
| `docs/specs/2026-11-12-org-reseat.md` | FR-9 の文は変えず、その行の終わりに「2026-11-16 追記」を足した(2 項目、QUESTION の行の `PLAN`、TASK か RESULT だけの org にも節が入ること)。Status に 2026-11-16 の段落。下の「FR-9 の扱い」 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-15 改訂(feat/reseat-leader-progress)の (a) の中身の列挙に 2 項目と QUESTION の `PLAN:` を足した。下の「org-runtime 仕様の扱い」 |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「座席の立て直し」節の最後の段落の列挙に 2 項目。1 つ直して `sync-skills.sh` を回し、`templates/base/` の 2 つは `cp` した。足した行の幅は 73 桁以内 |
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | 「座席を立て直す」節の最後の段落の列挙に 2 項目(同じ手順、76 桁以内) |
| `internal/cli/org_reseat.go`、`internal/cli/org_reseat_test.go` | `reseat --help` の節の文に 2 項目(「the last TASK sent to each seat」「the RESULT items the leader raised」)と、その pin を一緒に直した |
| `internal/org/reseat_progress.go`、`internal/org/feature.go` | コメントだけ。V12 の下 |
| `docs/tech-debt/README.md` | 下の「tech-debt」 |
| `docs/plans/active/2026-11-15-reseat-leader-progress.md` | Progress の sub-bullet 3 つ(verify cycle 2、test cycle 2、この pass)と、3 つの箱の行に cycle 2 の一言。計画の本文と AC の箱は変えていない。digest は編集の前後とも `de0cca0b6f78` |
| `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`、`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` | この report と insight event |

commit は 2 つ。内容の commit `5aef79a4`(18 ファイル、+71/-51)と、この report と insights の commit。

### V11: 節の中身を列挙する 14 か所

README:303、AGENTS.md:92、reseat 仕様 FR-9、org-runtime 仕様の改訂 (a)、`/org` skill の 4 つの写し、`/director` skill の 4 つの写し、`ralph org reseat --help` とその pin で 14 か所。節は `reseatProgressSection` の順に 8 つの部分(`reseat_progress.go:288-372`)を持つので、列挙も同じ順にした。help の文は pin と一緒に直し、`go test ./internal/cli/ -run 'Reseat' -count=1` が通った。`ralph org start --help` は節に触れないので変えていない(cycle 1 と同じ)。

足した 2 項目の事実は、`reseat_progress.go` から取った。座席ごとの最後の TASK は今の命の `sent` の記録から(dry run、leader 宛て、TASK でない typed message は数えない)、上げた RESULT は今の命の RESULT の件を全部の状態で、新しい 5 件。QUESTION の件の行は、`PLAN:` のヘッダがあれば終わりに `PLAN <パス> sha256:<digest>` を付ける。

### FR-9 の扱い

FR-9 は `[x]` の要件の行なので、原文は変えず、行の終わりに日付つきの追記を足す形にした(列挙を書き換えると、cross-review の直しの前に承認された要件が見えなくなる)。Status には、2026-11-13 の 2 回目のパイプラインの段落と同じ型で、2026-11-16 の段落を足した(節の 2 行と `PLAN`、`leader.md` の規則が TASK_ID で突き合わせ、PR の状態と計画助言を確かめるようになったこと、FR-9 の文と FR-10 の 4 文と印は変えていないこと)。Summary の「経過(承認した計画、最後の gate、片付いていない受信箱の件、branch の commit)」は仕様を書いた時点の意図なので変えていない。AC-13 は「それぞれが載る」と書き、今も正しい。

### org-runtime 仕様の扱い

この仕様は本文を書き換えず、改訂の節を日付ごとに足す作りだが、2026-11-15 改訂(feat/reseat-leader-progress)はこの branch が cycle 1 で書いたもので、まだ main に無い。別の改訂の節を足すより、この改訂の (a) の列挙を直す形にした。見出しの日付は実装の日の 2026-11-15 のまま。`leader.md` の規則の説明(5 項目の名前)は、数も中身も合っているので触っていない(verify の「Documentation drift」)。

### V12: Go のコメント(コードの変更なし)

- `internal/org/reseat_progress.go` の `reseatProgressSection` の doc: 最後の TASK の行の書式を `- <seat>: <TASK_ID> (<time>)`(TASK_ID が無いときは `-`)に直し、承認の行と最後の TASK の行が括弧の前に空白を置いて、空白で切ると `sha256:<digest>` か TASK_ID が 1 語になることを書いた
- `internal/org/feature.go` の `StartFeature` の doc: 節が入る条件に、座席へ送った TASK、受信箱の open か acked の件、上げた RESULT の件を足した

`gofmt -l internal/` は出力なし、`go vet ./internal/cli/ ./internal/org/` は出力なし。コメントだけなので、`go build ./...` と `go test ./internal/cli/ -run 'Reseat'` のほか、挙動を確かめるものは回していない。

### tech-debt

行番号は commit `5aef79a4` のもの。既存の行は先頭の語でも探せる。

| 行(先頭の語) | 何をしたか |
|----|----|
| 519「Small findings of feat/reseat-leader-progress (#312c-2), the first run of its self-review」 | (a) の「86 lines」を 103 行に直した(`reseatProgressSection` は `reseat_progress.go:278-380`。HEAD でも作業ツリーでも 103)。(d) は、最後の TASK の行が同じ括弧の型で、`b8a9bd38` が直したので、gate の行だけが残ると書いた。Trigger の末尾に、`ec10ba6f`・`b8a9bd38` とこの pass のコメントが当たったが、持ち越しではなく数と状態の直しだと書いた。Related に V14 |
| 520「Known gaps of feat/reseat-leader-progress (#312c-2), about the rule that `leader.md` gives a re-seated leader」 | (a) から (c) を、4 つのセルで取り消し線にして、`RESOLVED 2026-11-16 in feat/reseat-leader-progress, ec10ba6f` と、直した中身とテスト名を書いた。Trigger は `Done in feat/reseat-leader-progress (commit ec10ba6f)`。(d)(実機の確認)は開いたまま。Related に V14 |
| 365「`internal/org/verbs.go` is 2277 lines」 | 持ち越しの注の `prompts_leader_reseat_test.go` を 438 から 516 に。ほかの 6 ファイルは触れておらず、数は cycle 1 のまま |
| 426「The role prompts grew with org-cross-review」 | 注の `leader.md` を 1,480(+89)から 1,516(+125)、テストを 438 から 516 に。数は `b8a9bd38`(両ファイルを最後に変えた commit)の `wc -l` |
| 514「`internal/org/reseat.go` (1,307 lines)」 | 数を直した: `org_reseat_test.go` 923 → 927、`org_reseat.go` 521 → 527、`feature.go` 848 → 874、`reseat_progress.go` 412 → 551、`reseat_progress_test.go` 589 → 869、`prompts_leader_reseat_test.go` 438 → 516。「all under the guide」を、`reseat_progress_test.go` を除いて偽でない文に直し、新しい大きさの行を指す。数の基準を、cycle 2 の内容の commit に書き換えた |
| 201「Test gaps left by org-feature-worktree」(e)、(f) | cycle 2 の注。`StartFeature` の doc を直したのでコメントだけの編集で、(e)(f) にもう一度当たる(`feature.go` 873 → 874)。払っていない |
| 396「`resolvedOrClean` in `internal/org/statedir.go`」 | 同じ注。呼び出し側 `feature.go` のコメントだけの編集で、`resolvedOrClean` の呼び出しは変えていない |
| 513「Fourteen small findings of docs/spec-org-reseat」(f)、(j)、(m)、(n) | cycle 2 の注。help の文に 2 項目を足して折り返しを変えたので、4 項目にもう一度当たる。どれも払っていない(文は `ralph.toml` の段落のまま) |
| 515「Test gaps of docs/spec-org-reseat」(h)、(n) | cycle 2 の注。`/org` と `/director` の skill の同じ段落、help の文と pin を直したのでもう一度当たる。skill と help を突き合わせる test は書いていない |
| 522(新)「The section 「agmsg の受信箱の運用」 of `internal/org/prompts/leader.md` says of `history.sh`...」 | V13 と C2-1。`Org.Send` は agmsg に何も投げない(`verbs.go` で `o.Agmsg.` を呼ぶのは `Leave` だけ)ので、「あなたが送った TASK も 50 件に入る」は偽で、座席の確認の項目が逆を言い、そちらが正しい。2 つのテストの名前と、直し方(節の文、`TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` の pin、`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckItem` の禁止語を `leader.md` 全体に)を書いた。前からある文なので「この branch が作った」とは書いていない |
| 523(新)「`TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites` ...」 | C2-4。`raw=true ` と `submit_unconfirmed=true ` を文字列で写し、`Send` に共有の定数が無いこと、語が増えると TASK が節から黙って落ちること(座席の確認は「行が無い」と読んで送り直す)。直し方は 2 通り(定数を共有する、または test が `Org.Send` と fake herdr で `sent` を作る) |
| 524(新)「`internal/org/reseat_progress_test.go`, new in feat/reseat-leader-progress (#312c-2), is 869 lines」 | V15。`ec10ba6f` が 589 から 869 にした(+280)。`progressSentEvent` から終わりまでの約 260 行を `reseat_progress_tasks_test.go` に移すと約 610 行になる、という直し方 |

触った行(上の 12 行)はどれも 5 つのセルで、バッククォートと `~~` はセルごとに偶数。計画の参照は `docs/plans/active/2026-11-15-reseat-leader-progress.md` と `docs/plans/archive/` の既存の名前だけで、`check_tech_debt_plan_refs` と同じ式で、無い参照は 0 件。

### 掃引

Trigger の列を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、取り消し線の部分と、日付つきの note より後ろを外す)、cycle 2 の変更(`reseat_progress.go`、`leader.md` の規則の 3 つの項目、`StartFeature`、help、2 つの skill、README、AGENTS.md、2 つの仕様)の語 27 個で候補を出した(この report を書いた時点の file で 107 行。5 つのセルを持つ 416 行のうち)。ほとんどは「sent」「inbox」「Send」の語の一致で、原文の Trigger を読んで判断した。

当たった行は上の表のとおり。当たらないとした行。

| 行(先頭の語) | 判断 |
|----|----|
| 521「Test gaps of feat/reseat-leader-progress」(a)、(b) | (a) の Trigger は `StartFeature` の `startProgressSection` の呼び出しと `lastDisbandedIndexes`。どちらも編集していない(`ec10ba6f` の hunk に `start := disbanded + 1` は文脈の行として出るだけ)。(b) の「ほかに何も持たない start」は、座席へ送った TASK か上げた RESULT だけがある org にも節が入るようになったが、文は今も正しい |
| 180(a)、177、244〜252、262、355、380、381、472 | `/org` skill の command table と、`/director` skill のほかの節の編集を待つ。この pass が編集したのは、2 つの skill の reseat の段落の列挙だけ |
| 377、384、427、508、197 | `leader.md` の手順 1・4・14 などを名指す。cycle 2 の fix は規則の 3 つの項目を直し、手順の本文は変えていない(verify の判断のとおり) |
| 200(g)、516、517、371、401、468 | 取り消し線の項目、`Disband` や `takeReseatLock`、README の別の段落を指す |

### 直さなかったこと

- 行 465(`/director` skill の次の変更、scaffold version note)に持ち越しの注: handoff と verify は足すよう求めたが、足していない。この行は 1 つ目のセルが `~~...~~ (RESOLVED 2026-11-13 in docs/director-notice-handling)` で取り消されており、`/director` skill は今、`.ralph/manifest.toml` の版の note と `ralph upgrade` を打たないことを書いている(`git grep -n -i -e scaffold -e 'manifest.toml' -e 'ralph upgrade' -- .claude/skills/director/SKILL.md` は 7 行)。「did not pay it」「the skill still says nothing of the scaffold version note」と書くと偽になる。Trigger のセルの 2026-11-13 から 2026-11-15 の持ち越しの注(docs/spec-org-reseat の 3 つ)は、解決が main に入る前に書かれた文で、今は古い。取り消し線も `Done in` もセル 4 に付いていないが、解決済みの行に手を入れるのは範囲の外とし、足していない。lead が決めてほしい(下の「残っていること」)
- V13 の `leader.md` の文と 2 つのテスト: prompt とテストの変更で、`/self-review` からのやり直しになる。cap の 2 回に達したので、行 522 にした(verify も同じ判断)
- C2-4 の定数の共有: `verbs.go` と `send_fallback.go` の変更で、計画の Scope の外。行 523
- V15 の test ファイルの分割: move-only でも `/self-review` からやり直し。行 524
- V16 と V17: `/pr` の Known gaps に回す(handoff のとおり)。V17 の 1 文を行 521 に足す案は、handoff が `/pr` へ回すとしたので採っていない
- `ralph org start --help`、`/org` skill の command table の `reseat` と `start` の行: cycle 1 と同じで触っていない。行 180 (a) が cell を伸ばさないことを求める
- 計画の本文と AC の箱: 変えていない。D6 と AC2 の括弧書きが TASK と RESULT を含まない件(V17)は、計画の本文を書き換えない記録の書き方

### `/pr` に渡す文

Known gaps に載せる 2 件。

- V16(INFO): `leader.md` の規則に書かれていない縁が 3 つある。(a) 座席の行の TASK_ID が `-`(`--raw` の TASK で `TASK_ID:` が無い)のとき。実際は同じ TASK_ID を履歴で探して見つからず、送り直しに進む。(b)「branch の commit」の行は読めなかったときも同じ見出しで出るので、計画項目の「commit の行があって」を満たしてしまう。計画項目は `git log --name-only` で自分でも調べるので害は小さい。(c) 座席ごとの最後の TASK に RESULT が見つかっても、その RESULT を前の leader が処理して次の座席へ TASK を送ったあとかは、時刻を比べないと分からない。実機の通し(tech-debt 行 520 (d))で見る
- V17(INFO): 計画の D6 と AC2 の括弧書きは、節ができる条件に座席へ送った TASK と上げた RESULT を含まない。実装は両方で節を作る(`reseatProgressSection` の `add`)。計画の本文は変えない記録の書き方(cycle 1 の V8 と同じ)

### cycle 1 の記述の訂正

| cycle 1 の記述 | cycle 2 の事実 |
|----|----|
| 節の中身の列挙は 6 項目(README、AGENTS.md、skill、help の文) | 8 項目に直した。cycle 1 の列挙は `77a45022` の時点では正しく、`ec10ba6f` が 2 項目を足してから古くなった |
| 行 514 の数: `reseat_progress.go` 412、`reseat_progress_test.go` 589、`prompts_leader_reseat_test.go` 438、`org_reseat_test.go` 926、`org_reseat.go` 526、`feature.go` 873 | 551、869、516、927、527、874。「all under the guide」は `reseat_progress_test.go` が 800 行を超えたので偽 |
| 行 365 と 426 の数: `prompts_leader_reseat_test.go` 438、`leader.md` 1,480(+89) | 516、1,516(+125) |
| 行 519 (a) の 86 行 | 103 行 |
| 「新しい 3 行(519〜521)」 | 変わらない。cycle 2 の新しい行は 522〜524 |
| 差分の大きさ: 27 ファイル、+2,558/-54。docs の外でテストを除くと 15 ファイル、+651/-40 | 29 ファイル、+3,342/-54。15 ファイル、+836/-40 |

### 確かめたこと(文書が実装と合っていた所)

- `reseat_progress.go:288-372` の部分の順(承認、gate、最後の TASK、受信箱、上げた RESULT、commit、今回の reseat、前の reseat)は、列挙の順と同じ
- 新しい 2 項目の説明は、`readReseatProgress` の Tasks と Results、`reseatProgressSentTask`(前置きをはがして TYPE が TASK のもの)と合う
- `leader.md` の gate の項目(`OPEN` か `MERGED` なら PR を作らず、`CLOSED` だけなら BLOCKED)と計画の項目(per-plan は QUESTION の `PLAN`、split は `## Design decisions`)は、仕様の Status の段落の言い方と合う
- `docs/quality/`、`docs/architecture/repo-map.md`、`docs/recipes/`、`.claude/rules/ralph/` に、節の中身を列挙した所は無い(`git grep` で確かめた)

### Size check

verify の V15 が挙げた `internal/org/reseat_progress_test.go` は 869 行(`wc -l`)。この branch が新しく作った Go ファイルのうち、800 行を超えたのはこれだけで、行 524 にした。ほかの新しいファイルは 800 行以内: `reseat_progress.go` 551、`reseat_progress_start_test.go` 411、`prompts_leader_reseat_test.go` 516。この branch が変えた既存の Go ファイルは、変える前から 800 行を超えていて、行 514 が持つ: `reseat.go` 1,553、`org_reseat_test.go` 927(923 から)、`feature.go` 874(848 から)。`org_reseat.go` 527 は 800 行以内。`leader.md` 1,516 は Go でなく、行 426 が持つ。

### Verification

- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` は編集の前後とも `de0cca0b6f78`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`./scripts/check-skill-sync.sh` は 14 skill で ok
- `go build ./...` は成功、`go vet ./internal/cli/ ./internal/org/` は出力なし、`gofmt -l internal/` は出力なし、`git diff --check` は rc 0
- `go test ./internal/cli/ -run 'Reseat' -count=1`: `ok`(コメントと help と pin を直した直後に 25.7 秒、commit 前に 33.7 秒)。`go test ./internal/org/ -run 'Skill' -count=1`: `ok`
- tech-debt: 触った 12 行は 5 つのセル、バッククォートと `~~` はセルごとに偶数。無い計画の参照は 0 件
- 変えたファイル 18 本の U+FFFD は 0 件
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)、`/natural-japanese` の lint(hook は書き込みごとに回り、指摘は出なかった)

### Insight event

```
./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` に足した行は cycle 2、pass、ts は 2026-11-15T18:13:11Z(UTC の日付が 2026-11-15 なので、既存の file に足された。日本時間では 2026-11-16 03:13)。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。

### 残っていること(lead へ)

- 決めてほしい点 1(行 465): handoff の「持ち越しの注を足す」は、行が RESOLVED なので足していない。行の Trigger のセルの 2026-11-13 から 11-15 の古い注(「the skill still says nothing」)を直すか、セル 4 に `Done in` を足すかは、この PR の範囲の外とした。足すなら 1 回の Edit
- 決めてほしい点 2(Go の変更): この pass は Go を 4 か所変えた(コメント 2 か所、help の文とその pin)。cycle 2 の `/test`(`043dbe59`)のあとの変更で、フルの `run-test.sh` は通していない。`go test ./internal/cli/ -run 'Reseat'` は通った。コメントは挙動を変えず、help の文は pin と一緒に通っている。cap に達しているので、`/pr` に進む前に `/test` を打ち直すかは lead が決める
- `/pr` の Known gaps に V16 と V17(上の「`/pr` に渡す文」)
- 払っていないもの: 持ち越しの注を足した行 201、396、513、515 は、どれも開いたまま。行 522〜524 も開いている
- 節を読んだ leader の実機での動きは見ていない(行 520 (d)、仕様 AC-12)

---

## Cycle 1(履歴)

下は cycle 1(2026-11-16、run 開始時の HEAD `fc8f236a`、内容の commit `77a45022`)の report で、そのまま残した。数のうち cycle 2 で変わったものは、上の「cycle 1 の記述の訂正」にある。

# sync-docs report: reseat-leader-progress

- Date: 2026-11-16
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(承認の digest `de0cca0b6f78`)。issue #312 の #312c-2
- Branch: `feat/reseat-leader-progress`。base は origin/main `48a8a661`(#335 の PR。`e3914c93` で取り込み済み)。run 開始時の HEAD は `fc8f236a`(test report の commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は 3 回の run(`64c7f828`、`c9ce0f77`、`f57e37f4`。最後は Merge yes)、verify は `12a76fc7` が fail(V1)で、`e483b52a` で直して `1a1d8981` で打ち直し、partial-pass、test は `fc8f236a` が pass。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD` と、3 つの report の V2〜V10、Documentation drift、register への引き継ぎ。文書の各文は HEAD のコード(`reseat_progress.go`、`feature.go`、`reseat.go`、`leader.md`、`org_reseat.go`)と突き合わせた
- 制約: `go test ./...` と `run-test.sh` は回していない(handoff の指示)。Go のファイルは `ralph org reseat --help` の 1 文とその pin の 2 か所だけ変えたので、`go test ./internal/cli/ -run 'Reseat' -count=1` を回した(`ok`、26.1 秒)。ほかに `go test ./internal/org/ -run 'Skill'`(`ok`)と `go test ./internal/config/`(`ok`)
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `77a45022` を含み、あとの report の commit は docs だけなので、docs の外の数は変わらない): `git diff --shortstat origin/main...HEAD` は 27 ファイル、+2,558/-54(計画、report、insights を含む)。docs の外は 19 ファイル、+2,092/-40。docs の外でテストを除くと 15 ファイル、+651/-40(Go の本体 4 ファイルが +460/-2: `reseat_progress.go` 412、`feature.go` +26/-1、`reseat.go` +16、`org_reseat.go` +6/-1。`leader.md` が +89。残りは skill の 8 つの写しと AGENTS.md、README)。Go のテストは 4 ファイルで +1,441。walkthrough は書いていない

## Summary

verify の V2〜V6 と V9 は直した。V10(register)は、Trigger に当たった 7 行に持ち越しの注を足し、新しい 3 行を足した。V7 と V8 は新しい行(テストの穴)に、self-review の L5 と L3 の残りと S3 の implementer の 3 点は新しい行に入れた。`ralph org reseat --help` は、すでに leader のタスクが持つ物(`[reviewer]` の 4 つの値)を書いていたので、節の 1 文を足して pin した。`ralph org start --help` は触っていない(下の「help の扱い」)。

verify が `/org` skill の `start` の行と「機能ごとの org」の leader のタスクの説明にも 1 文足してよいとしていた件は足していない(下の「直さなかったこと」)。

## Changes made

| File | Change |
|------|--------|
| `README.md` | `ralph org reseat` の段落の最後の 2 文(「reseat does not tell the leader …」)を、立て直した leader が節を受け取り、`leader.md` の規則で続きから進むことに置き換えた。下の「V2」 |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「座席の立て直し」節の最後の段落(「reseat はまだ、それまでの経過を leader に渡さない」)。4 つの写しは同じ文。下の「V3」 |
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | 「座席を立て直す」節の最後の段落(「leader も立て直したときは、reseat はまだ経過を渡さないので…」)。4 つの写しは同じ文。下の「V4」 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-12 改訂 (d) の末尾に改訂の印、「2026-11-15 改訂(feat/reseat-leader-progress)」の項を足した。下の「V5」 |
| `AGENTS.md` | `internal/org/` の行に `reseat_progress.go` の一節。下の「V6」 |
| `docs/specs/2026-11-12-org-reseat.md` | Impact の表の 3 行目。下の「V9」 |
| `internal/cli/org_reseat.go`、`internal/cli/org_reseat_test.go` | `reseat --help` の `ralph.toml` の段落に節の 1 文(5 行)、`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` にその pin(1 項目)。下の「help の扱い」 |
| `docs/tech-debt/README.md` | 持ち越しの注 7 行(514、201、365、426、396、513、515)と、新しい 3 行(519〜521)。既存の字は消していない。下の「V10」 |
| `docs/plans/active/2026-11-15-reseat-leader-progress.md` | Progress の sub-bullet 3 つ(verify、test、sync-docs)と、3 つの箱(`Review artifact created`、`Verification artifact created`、`Test artifact created`)を `[x]` に。計画の本文と AC の箱は変えていない |
| `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md` | この report |
| `docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` | `sync_docs` の行(cycle 1、pass)。ts は 2026-11-15T15:57:19Z(UTC の日付なので、既存の file に足された) |

commit は 2 つ。内容の commit `77a45022`(上の表の README から計画の Progress まで。16 ファイル)と、この report と insights の commit。

## verify が挙げた drift

### V2: README

reseat の段落の終わりを、次の内容に直した。節の名前(立て直しの経過)、中身(最新の計画の承認、最後の gate の記録、open か acked の受信箱の件、branch の新しい順 20 件の commit と総数、今回の reseat の id と leader のあとに起動する座席、前の reseat)、`start --plan` の立て直しも同じ節を足すこと、経過のない org と、受信箱か git が読めなかっただけの場合は節がないこと、leader は `leader.md` の規則で手順 1 からではなく、座席の確認、計画、gate、受信箱、座席の順に進むこと(承認済みの計画は書き直さない、HEAD と同じ gate の pass は PR の手順へ)、DECISION は節が言えないことと、leader が動き続けているあいだに reseat が立てた座席のためのものであること。「see above」は、段落の前のほうにある DECISION の説明(leader が対象でないとき)を指す。

### V3: `/org` skill

「leader も立て直したときは(`start --plan` の立て直しも同じ)」で始め、節の足し場所(`StartFeature` が leader の起動の前)と中身、節がない場合、leader の規則(まず leader 以外の座席の状態、次に計画、gate、受信箱、座席)、director か人が経過を DECISION に書き写す必要はないこと、節の条件に当たらないとき(計画の digest が承認のときと違う、gate の head が今の HEAD と違う、など)は雛形の手順のとおりその手順から進むこと、を書いた。1 行の幅は 78 桁以内(隣の行の最大と同じ)。編集は `.claude/skills/org/SKILL.md` にだけ手でして、`./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`templates/base/` の 2 つは `cp` した。`check-skill-sync.sh` は 14 skill で ok。

### V4: `/director` skill

前提を直した: leader のタスクには節が入り、leader は `leader.md` の規則で続きから進むので、director が経過を DECISION に書き写す必要はない。leader 以外の座席だけを立て直したときに reseat が出す DECISION は、これまでどおり director が送る(同じ節の 6)。leader がそれでも済んだ手順をやり直し始めたら、人に伝え、DECISION で伝えるかを決めてもらう手順は残した。disband のあとの `start --plan` は承認が消えているので、承認を求め直すのは誤りではない、と括弧に足した(`leader.md` の計画の項目が、disband のあとの per-plan は手順 2・3 から進むと書く)。「DECISION を「答える順」の 1 と 2 のとおりに送る」の文は変えていない。

### V5: `docs/specs/2026-08-01-org-runtime.md`

(d) の末尾に `**(2026-11-15 改訂で変更: …)**` を足した(この仕様の前の印と同じ形)。新しい項は、前の最後の改訂の項(`2026-11-15 改訂(fix/send-codex-paste)`)の次、`### 運用ノート` の前。(a) は節の中身と足す場所、`reseat` と `start --plan` の両方、節がない場合、`leader.md` の 5 項目の規則。もう 1 項は「本文は書き換えない」で、(d) の印と、leader 以外の座席だけを立て直したときの DECISION がこの改訂の外であることを書いた。項の日付は 2026-11-15 とした(姉妹の仕様 `2026-11-12-org-reseat.md` の Status の段落と同じ日で、実装の日)。この run は 2026-11-16 の日本時間。

### V6: `AGENTS.md`

`internal/org/` の行の reseat の項(`reseat.go` … `one reseated event)`)のあと、`and the pane lock` の前に、`the progress section of a restarted leader's task (reseat_progress.go: …)` を足した。中身の 6 つ(最新の計画の承認、最後の gate の記録、open と acked の受信箱の件、branch の commit、今回の reseat の座席、前の reseat)、`StartFeature` が leader を起動する前に足すこと、reseat と `start --plan` の立て直しで同じこと、経過のない org では足さないこと。行のほかの字は変えていない。`check-sync.sh` は PASS(`internal/org/` の行は root だけの行で、`templates/base/AGENTS.md` には無い)。

### V9: `docs/specs/2026-11-12-org-reseat.md`

Impact の表の `feature.go` の `featureLeaderTask`(2 本目)を、`internal/org/reseat_progress.go` と `feature.go` の `StartFeature`(`featureLeaderTask` は変えず、`StartFeature` が節を足す)に直した。verify が V9 の (2) と (3) とした 3 点(`:7` と `:36` の日付つきの過去の記述、`:15` が引く `docs/plans/archive/...` は `/pr` のあとにだけ在る)は直していない。`:15` は 2026-11-15 の段落で、`/pr` が計画を archive へ移す名前は同じ(`archive-plan.sh` が `docs/plans/active/<name>` を `docs/plans/archive/<name>` に移す)。

### V10: tech-debt

Trigger に当たった行(行番号は commit `77a45022` のもの。verify の時点から動いていない)。どの行も、Trigger のセルの末尾に日付つきの note、Related のセルの末尾に `docs/plans/active/2026-11-15-reseat-leader-progress.md`(`/pr` の `archive-plan.sh` が書き換える。`archive/` とは書いていない)を足した。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 514「`internal/org/reseat.go` (1,307 lines)」 | `reseat.go` 1,537 → 1,553(+16)、`feature.go` 848 → 873(+25)。この run の help の pin で `org_reseat_test.go` 923 → 926 | 持ち越し。`startFeatureInput` とその 2 つの隣は編集していない。新しい 4 ファイル(412、589、411、438)は 800 行未満。`reseat_test.go` は 1,868 のまま。数は `e483b52a` の `wc -l`、`internal/cli` の 2 ファイル(526、926)は help の文のあと |
| 201「Test gaps left by org-feature-worktree」(e)、(f) | `StartFeature`(節の呼び出しと `StartFeatureParams.Reseat`) | 持ち越し。(e) の通らない枝と (f) の防御の枝は測り直していない。新しい呼び出しは 4 つの `TestReseatProgress_StartFeature*` が通す |
| 365「`internal/org/verbs.go` is 2277 lines」 | leader の役割の prompt の新しいテスト `prompts_leader_reseat_test.go`(438 行) | 持ち越し。ほかの 6 ファイル(`verbs.go` 2639、`verbs_test.go` 5893、`prompts_test.go` 2727、`spawn.go` 3109、`spawn_test.go` 6408、`director_skill_sync_test.go` 1318)は触れていない |
| 426「The role prompts grew with org-cross-review」 | `leader.md` 1,391 → 1,480(+89)。見出しは足していないが、「機能ごとの org」の導入に規則(段落と 5 項目)を足した | 持ち越し。字義どおりの「section」ではないことと、この行の直し方の対象に当たることを書いた |
| 396「`resolvedOrClean` in `internal/org/statedir.go`」 | `StartFeatureParams.Reseat` の新しい field と節の呼び出し(2026-11-05 の `DriverFromRoleEntry` の前例に従った) | 持ち越し。`resolvedOrClean` の呼び出しは編集していない |
| 513「Fourteen small findings of docs/spec-org-reseat」(f)、(j)、(m)、(n) | この run の help の 1 文と pin(`orgReseatHelp`、`TestOrgReseatHelp_…`) | 持ち越し。4 項目はどれも払っていない。新しい文は `ralph.toml` の段落にあり、4 項目が名指す所ではない |
| 515「Test gaps of docs/spec-org-reseat」(h)、(n) | (h) は `/org` skill の reseat の節と `/director` skill の「座席を立て直す」の編集。(n) は help の 1 文 | 持ち越し。(n) の文は 1 語ずつ pin したが、前の文は 1 つの断片ずつのまま。変異は回していない。command table の reseat の行は編集していない |

新しい行(519〜521。行頭の語で探す)。

- 「Small findings of feat/reseat-leader-progress (#312c-2), the first run of its self-review: L5 and the remainder of L3」: (a) `reseatProgressSection` が 86 行、(b) `reseatProgressFirstBodyLine` と `inboxSummary` の二重、(c) `reseatProgressText` が呼ぶたびに `NewReplacer` を作ること、(d) gate の行の括弧が値に付くこと。(d) は handoff の `branch <名前>(<時刻>)` と `via <名前>(<時刻>)` に、コードを読んで見つけた `pass(gate_passed)head <sha>` を足した(`reseat_progress.go:220`)。規則が比べるのは `head` と `base` だけなので判断は変わらない
- 「Known gaps of feat/reseat-leader-progress (#312c-2), about the rule that `internal/org/prompts/leader.md` gives a re-seated leader」: (a) `--state all` は merge されずに閉じた PR も「ある」と読む、(b) PR を作ったあと RESULT を上げる前に止まった leader は手順 15 に進み RESULT が上がらない、(c) 「承認なし、コミット済みの計画」から手順 2 に進むとき計画助言が済んだか分からない(計画助言の結果の残る場所を読んで、Trigger に書いた: `.harness/state/plan-advisory-out.md`、split の計画の `## Design decisions`、per-plan の QUESTION の `ADVISORY:` の行)、(d) 節を読んだ leader の実機での動きは見ていない(仕様 AC-12 と同じ限り)
- 「Test gaps of feat/reseat-leader-progress (#312c-2), found by its verify report (V7, V8) and its test report (Test gaps 1 and 2)」: (a) `StartFeature` を通した「disband のあとの start で commit だけの節」のテストがない(部品は別々のテストが見る)、(b) 読み取りの失敗の行は節が別の理由で在るときだけ載る(計画の D6・AC7 のとおりの設計で、バグではない)と明記した

どの新しい行も 5 つのセルで、バッククォートと `~~` は列ごとに偶数。計画の参照は `docs/plans/active/2026-11-15-reseat-leader-progress.md` だけ。

## help の扱い

- `ralph org reseat --help`: 「A leader it starts gets the current values of the four [reviewer] keys in its task」と、すでに leader のタスクの中身を書いていたので、handoff の条件(すでに leader が受け取る物を書いているとき)に当たる。その段落に 1 文足した: 「A leader it starts also gets, at the end of its task, what the org holds for its current life, when it holds any: the latest plan approval, the last gate record, the open and acked inbox items, the branch's commits, the seats this reseat starts after the leader, and the previous reseat. Its role prompt tells it to go on from there.」。既存の 2 つの pin の断片(`[reviewer]` の文と、`The reseat, stop, spawn, …` の文)は、間に入れても壊れない。新しい文を `TestOrgReseatHelp_…` の項目に足した。`go run ./cmd/ralph org reseat --help` で段落の見た目を確かめた
- `ralph org start --help`(`internal/cli/org.go`): 触っていない。verify は reseat の help と同じ扱いで「間違いではなく不足」とし、handoff が reseat の help だけを指したので、start の help は handoff の範囲の外とした。start の help も leader のタスクの `[reviewer]` の値を書く(段落の終わりの「written in the leader's task as two lines」)ので、足すなら同じ種類の 1 文で足りる。start の help の pin の場所は、この run では調べていない

## 直さなかったこと

- `/org` skill の command table の `reseat` の行と `start` の行: 節への 1 文を足す案(verify の V3 の備考)は、いったん `reseat` の行に足したが、tech-debt の行 180 の (a)(「次に command table を編集するときは、各 cell を 1〜2 文に切る」)に当たり、cell を伸ばす向きなので戻した。節の説明は「座席の立て直し」節の最後の段落にあり、table の行は「座席の立て直し」節を指す
- `docs/specs/2026-11-12-org-reseat.md` の `:7` の「2 本目は未着手」と `:36` の「経過を持たない」: 日付の付いた過去の記述で、verify も直さないとしている
- Go のコメントが指す `docs/plans/archive/2026-11-15-reseat-leader-progress.md`: `/pr` が計画を移したあとにだけ在る名前。`archive-plan.sh` の archive 先の名前と同じことを確かめた(`docs/plans/active/` の名前をそのまま `archive/` に移す)
- 計画の本文と AC の箱: 変えていない。`./scripts/plan-visual.sh digest` は編集の前後とも `de0cca0b6f78`

## 掃引

Trigger 列を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、全体が取り消し線の行は外す。日付つきの note より前の部分を読む)、この branch が変えた file、関数、skill、doc の語 36 個で候補を出した(51 行)。1 回目の候補から、原文の Trigger を読んで判断した。

当たった行は上の表のとおり。当たらないとした行。

| 行(先頭の語) | 判断 |
|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の command table の編集に当たる。最終的に table は編集していない(上の「直さなかったこと」)ので、持ち越しの注は足していない |
| 177、244〜252、262、355、380、381、472 | `/director` skill の別の節(`stop --all` の説明、progress の報告、inbox の節、send の確かめ、wait の終了コード、accept)の編集を待つ。この branch が編集したのは「座席を立て直す」の最後の段落だけ |
| 200(g) | 取り消し線の項目(`f1b0352e` で閉じた)。`leader.md` の機能ごとの org の節を編集したが、(g) は終わっている |
| 377、384、367、372、427、508 | `leader.md` の機能ごとの org の手順 1・4・14、「運用規律」の hook の項、cross-review の小節、reviewer の手順を名指す。この branch は規則を「次の順で進めます。」の前に置いただけで、手順は変えていない |
| 195、203、204、199、391、431、432、425、475、476 | `StartFeature` や `featureLeaderTask` の別の部分(version skew、worktree の再利用、digest、herdr のゲートの順序、設定の読み)の Trigger。`featureLeaderTask` は変えていない |
| 516、517 | `Disband` と `takeReseatLock`、`ApprovalDecisionCommand` と `orgReadCommandHint`。どちらも変えていない |
| 371、401、468 | README を挙げるが、Trigger は recipe の引用、`[org].state_dir` の段落、scaffold のノートで、reseat の段落ではない |

## 確かめたこと(文書が実装と合っていた所)

- `reseat_progress.go`: 「新しい順に <n> 件」(`reseatProgressMaxCommits` は 20)、「古い順に <n> 件」と「ほか <n> 件(新しいもの)」の受信箱、読み取りの失敗は別の理由で節があるときだけ(`reseatProgressSection` の doc と `progress` の変数)、`今回の立て直し:` と `前の立て直し:` の 2 行。skill と README の言い方と合う
- `leader.md:150-237`: 導入と 5 項目(座席の確認、計画、gate、受信箱、座席)。skill の「まず leader 以外の座席の状態、次に計画、gate、受信箱、座席」は、この順
- `reseat.go` の `startAll` が `StartFeatureReseat{ID, Seats}` を渡し、`StartFeature`(`feature.go`)が leader の spawn の前に `startProgressSection` を呼ぶ。AGENTS.md の「before it starts the leader」と合う
- `ralph org reseat` の終わりの文(`internal/cli/org_reseat.go` の「reseat sends the leader nothing, and the seats it started have no TASK」)は leader が対象でないときの DECISION の注で、今も正しい(verify の「Yes」の行)
- `docs/quality/`、`docs/architecture/repo-map.md`、`docs/recipes/`、`.claude/rules/ralph/` に、reseat が leader に経過を渡さないと書いた所は無い(`git grep` で確かめた)

## 残っていること(lead へ)

- 決めてほしい点: この run は Go を 2 か所変えた(help の 1 文と、その pin の 1 項目。`77a45022` の `internal/cli/org_reseat.go` と `org_reseat_test.go`)。`/test` の report(`fc8f236a`)のあとの Go の変更で、フルの `run-test.sh` はこの 2 か所を通していない。`go test ./internal/cli/ -run 'Reseat'` は通った。パイプラインの規則では、Go の変更は `/self-review` からの回し直しになりうる。help の文を入れずに進めるなら、`77a45022` の 2 つの Go のファイルだけ戻せばよい(pin も一緒に消える。tech-debt の行 513 と 515 の note は help の文を前提に書いたので、戻すときは note の「added one sentence to `orgReseatHelp`」の段落も外す)
- 払っていないもの: 持ち越しの注を足した 7 行は、どれも「Carried over to the next change that meets this trigger」。新しい 3 行の項目もすべて開いている
- 節を読んだ leader の実機での動きは見ていない(新しい行の (d)、仕様 AC-12)
- `docs/tech-debt/README.md`、`AGENTS.md`、4 つの skill の写しは、ほかの session も編集しうる。この run の tech-debt の変更は新しい 3 行と 7 行への追記で、既存の字は消していない。merge の衝突は末尾(新しい 3 行)と、365・514 の Trigger の末尾で起きやすい

## Verification

- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` は編集の前後とも `de0cca0b6f78`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`./scripts/check-skill-sync.sh` は 14 skill で ok
- `go build ./...` は成功、`go vet ./internal/cli/ ./internal/org/` は出力なし、`gofmt -l internal/` は出力なし、`git diff --check` は rc 0
- tech-debt: `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で、無い計画の参照は 0 件。触った 10 行はどれも 5 つのセルで、バッククォートと `~~` は列ごとに偶数。`git diff --word-diff=porcelain` に出る消えた語は 5 つで、どれも語の後ろに句読点(`.`、`;`)を足した所
- `go test ./internal/cli/ -run 'Reseat' -count=1`: `ok  github.com/yoshpy-dev/ralph/internal/cli 26.120s`。`go test ./internal/org/ -run 'Skill' -count=1`: `ok`。`go test ./internal/config/ -count=1`: `ok`
- 変えたファイルの U+FFFD は 0 件
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`(handoff の指示)、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)、`/natural-japanese` の lint(hook は書き込みごとに回り、指摘は出なかった)

## Insight event

```
./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` に足した行は cycle 1、pass、ts は 2026-11-15T15:57:19Z(UTC の日付が 2026-11-15 なので、既存の file に足された)。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。
