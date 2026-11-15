# Verify report: reseat-leader-progress

- Date: 2026-11-16(cycle 4。run 5 の cross-review の ACTION_REQUIRED 2 件の直し `4096ae88` と、cycle 4 の self-review の直し `70b1511c` のあと)。run 5 の verify の直し `8a6676e4` も、この run が初めて読む。run 5 以前の run も同じ日、cycle 1 は 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)。`./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` を打ち直し、出力 `de0cca0b6f78` が `- Approved:` の `sha256:de0cca0b6f78` と同じことを確かめた
- Verifier: verifier サブエージェント(`/verify`)。**Cycle 4**(`cycle-count.json` は `cycle: 4`、`cap: 4`。cap の最後の回で、`--cycle auto` が 4 を出した)。新しい run を先頭に置き、run 5 以前は履歴として残した
- Scope(cycle 4): HEAD `34a17128`、`git diff main...HEAD`(base `48a8a661`、main と origin/main の先頭、29 ファイル、+5161/-54)。run 5 の verify(`892703bc`)のあとに 12 commit。コードに当たるのは 3 commit、4 ファイル(`git diff 892703bc..HEAD -- internal` は +160/-29): `8a6676e4`(run 5 の verify の V32 (a) と V33)、`4096ae88`(run 5 の cross-review の 2 件)、`70b1511c`(cycle 4 の self-review の C4-1〜C4-4)。ファイルは `internal/org/reseat_progress.go`、`internal/org/reseat_progress_start_test.go`、`internal/org/prompts/leader.md`、`internal/org/prompts_leader_reseat_test.go`。Go の本体で変わったのは、コミットの範囲を読む `reseatProgressCommitsIn` と、2 つの doc だけ。skill、README、AGENTS.md、`ralph org reseat --help` は触っていない
- Evidence: `docs/evidence/verify-2026-11-16-reseat-leader-progress-cycle4.log`(`run-static-verify.sh` の出力そのものと、`git diff --check`・計画の digest・試し merge・コミット範囲の git の probe・固定の文の python による確かめ・行数。gitignore の対象で commit に入らない)。`run-static-verify.sh` 自身が書いた `docs/evidence/verify-2026-11-15-233431.log` も同じ内容
- Insight event: `./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase verify --verdict pass --source skill --cycle auto --low 4`(cycle 4、LOW 4)を 1 行足した(この report と同じ commit)。events のファイル名は UTC の日付(`2026-11-15-...jsonl`)で、report の日付(JST の 2026-11-16)と違う。00:00Z の前(23:43Z)に足したので、同じファイルに入った
- **Verdict(cycle 4): partial-pass**。静的検査は通った(rc=0、golangci-lint 0 issues)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は、`8a6676e4`・`4096ae88`・`70b1511c` を含めて読んだ範囲で満たす。run 5 の cross-review の 2 件は、コード、規則の文、テストの 3 つで閉じた。cycle 4 の self-review の LOW 4 件も閉じた(コメントの折り返しが一部残る。V40)。AC7 は `/test` の担当で、最後の test(`3998e20a`)のあとに 2 つのテストファイルが変わった。fail にする指摘はない。残りは `/sync-docs` の文書と register(V35〜V38)と、規則に書かれていない縁と記録の遅れ(INFO、V39〜V41)
- 以下は run 5 の verify を書いたときの見出し(履歴)
  - Date: 2026-11-16(run 5、cycle 3 の cross-review の ACTION_REQUIRED 2 件の直し `5ab60578` と、run 5 の self-review の直し `262b9df7` のあと)。cycle 3 の verify、run 3、cycle 2 の最初の run も同じ日、cycle 1 は 2026-11-15
  - Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)。`./scripts/plan-visual.sh digest <plan>` を打ち直し、`de0cca0b6f78` が `- Approved:` の `sha256:de0cca0b6f78` と同じことを確かめた
  - Verifier: verifier サブエージェント(`/verify`)。**Run 5**(`cycle-count.json` は `cycle: 3`、`cap: 4`。cycle 3 の cross-review の ACTION_REQUIRED 2 件を受けて cap を 3 から 4 に上げ、`cycle` は 3 のまま)。新しい run を先頭に置き、前の run(cycle 3、run 3、cycle 2 の最初の run)と cycle 1 は履歴として残した
  - Scope(run 5): HEAD `28447a1d`、`git diff main...HEAD`(base `48a8a661`、main と origin/main の先頭、29 ファイル、+4658/-54)。cycle 3 の verify(`c693d220`)のあとに 11 commit(`git log --oneline c693d220..HEAD`)。コードで変わったのは `internal/org/prompts/leader.md` とそのテスト `internal/org/prompts_leader_reseat_test.go` だけで、commit は 3 つ。依頼書が挙げた `5ab60578`(cycle 3 の cross-review の直し)と `262b9df7`(run 5 の self-review の C5-1〜C5-4 の直し)のほかに、`91578db5`(cycle 3 の verify の V26 の直し)も範囲に入る。`91578db5` は cycle 3 の test(`36d80504`)と 4 回目の sync-docs(`d812689e`、`cef100ae`)を通っているが、verify はこの run が初めてなので、一緒に読んだ
  - Evidence: `docs/evidence/verify-2026-11-16-reseat-leader-progress-run5.log`(`run-static-verify.sh` の出力そのものと、`git diff --check`・U+FFFD・試し merge・計画の digest・docs 以外の変更ファイル・`leader.md` の hunk・行数・固定の文の python による確かめ。gitignore の対象で commit に入らない)。`run-static-verify.sh` 自身が書いた `docs/evidence/verify-2026-11-15-220258.log` も同じ内容
  - Insight event: `./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase verify --verdict pass --source skill --cycle auto --low 4`(cycle 3、LOW 4)を 1 行足した(この report と同じ commit)
  - **Verdict(run 5): partial-pass**。静的検査は通った(rc=0、golangci-lint 0 issues)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は、`91578db5`・`5ab60578`・`262b9df7` を含めて読んだ範囲で満たす。cycle 3 の cross-review の 2 件は、規則の文とテストの両方で閉じた。AC2 は `StartFeature` を通した commit だけの節の結合が無く一部(行 521 (a) のまま)。AC7 は `/test` の担当で、最後の test(`ee9e41dc`)は 2 つの直しより前。fail にする指摘はない。残りは `/sync-docs` の文書と tech-debt(V28〜V31)で、規則の新しい確かめに書かれていない縁がある(V32、INFO)
- 以下は cycle 3 の verify を書いたときの見出し(履歴)
  - Date: 2026-11-16(cycle 3、3 回目の cross-review のあと)。run 3(cycle 2)と cycle 2 の最初の run も同じ日、cycle 1 は 2026-11-15
  - Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)。`- Approved:` の digest は `sha256:de0cca0b6f78`(計算し直していない。承認の検査は `/implement` が済ませている)
  - Verifier: verifier サブエージェント(`/verify`)。**Cycle 3**(`cycle-count.json` は `cycle: 3`、`cap: 3`。3 回目の cross-review の ACTION_REQUIRED 1 件と cycle 3 の self-review の LOW 2 件を受けた回で、cap の最後)。新しい run を先頭に置き、前の run(run 3、cycle 2 の最初の run)と cycle 1 は履歴として残した
  - Scope(cycle 3): HEAD `9235c95c`、`git diff main...HEAD`(base `48a8a661`、29 ファイル、+4226/-54)。前の verify(run 3、`c49a6bc5`)のあとに入ったのは、3 回目の test(`601b4643`)と sync-docs(`0d443ebb`、`8eeb7f69`)、3 回目の cross-review の triage(`3d8b5584`)、その直し(`04d3d986`)、cycle 3 の self-review(`eaa9bc71`)とその直し(`e6f98624`)、計画の記録(`b79f300e`、`9235c95c`)。コードで変わったのは `internal/org/prompts/leader.md` とそのテスト `internal/org/prompts_leader_reseat_test.go` だけ(`git diff c49a6bc5..HEAD --name-only` の docs 以外はこの 2 ファイル)
  - Evidence: `docs/evidence/verify-2026-11-16-reseat-leader-progress-cycle3.log`(`run-static-verify.sh` の出力そのものと、`git diff --check`・U+FFFD・試し merge・skill の写しの `cmp`・docs 以外の変更ファイル・`leader.md` の hunk・行数。gitignore の対象で commit に入らない)。`run-static-verify.sh` 自身が書いた `docs/evidence/verify-2026-11-15-203450.log` も同じ内容
  - Insight event: `./scripts/insights-append.sh --phase verify --verdict pass --cycle auto`(cycle 3、LOW 4)を 1 行足した(この report と同じ commit)
  - **Verdict(cycle 3): partial-pass**。静的検査は通った(rc=0、golangci-lint 0 issues)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は、`04d3d986` と `e6f98624` を含めて読んだ範囲で満たす。AC2 は `StartFeature` を通した commit だけの節の結合が無く一部(行 521 (a) のまま)。AC7 は `/test` の担当で、最後の test(`601b4643`)は 2 つの直しより前。fail にする指摘はない。残りは `/sync-docs` の文書と tech-debt(V22〜V25)で、規則の新しい分岐に、書かれていない縁がある(V26、INFO)

## Cycle 4 (cap 4、最後の回): HEAD `34a17128`

run 5 の verify(`892703bc`)のあとのコードの変更を読んだ。`8a6676e4` は run 5 の verify の V32 (a) と V33 の直しで、QUESTION の確かめが「読んだうちでいちばん古い行の時刻が座席の行の時刻よりあとなら数を増やして読み直す」文と、`NeverDiscardsAReply` の doc の 1 句。`4096ae88` は run 5 の cross-review の 2 件の直しで、(1) reviewer が gate か cross-review の TASK を打たなかった・誰も呼ばなかったと言う QUESTION を、DECISION だけで答えずに、手順 12 と「cross-review」の小節の reviewer の QUESTION の項(TASK を直して新しい TASK_ID で送り直す)へ回す。(2) commit の範囲を `<ref>..HEAD` から `HEAD --not <origin の既定のブランチ> <手元の既定のブランチ> --` に変え、計画の項の `git log` も同じ範囲にした。`70b1511c` は cycle 4 の self-review の C4-1(改訂の並びの終わりに `--`)、C4-2・C4-3(doc の範囲、折り返し、呼び名)、C4-4(「上限」の言い方)の直し。`leader.md` の hunk は 2 つだけ(座席の確認の項の QUESTION の経路と、計画の項の `git log`)で、gate・受信箱・座席の項は run 5 と同じ。テストは実行していない(`/test` の担当)。

固定の文は、`assertPhraseOrder` と同じ確かめ(空白を全部捨てた部分文字列が順に出るか)を `python3 -I` で、`prompts_leader_reseat_test.go` の座席の確認(4 テスト)・計画・gate・受信箱・座席の各項の順序つきの句と禁止の句に当てて見た。HEAD の `leader.md` は 118 句がすべて通った。`8a6676e4` の `leader.md` は、routing の句(`:447`)と計画の項の `git log` の句(`:542`)で落ちる。HEAD の `leader.md` から「それ以外の」だけ消した版も、`:449` の句で落ちる。

### run 5 の指摘と cycle 4 の self-review が閉じたか

| 指摘 | コードと規則の文 | 固定するテスト | 判定 |
| --- | --- | --- | --- |
| run 5 の cross-review 1 [P2] reviewer の QUESTION を DECISION だけで答え、TASK を送り直す項を飛ばす | `leader.md:220-228`: 見つけた QUESTION が、TASK の本文の形が違うので打たなかった・誰も呼ばなかったと言う reviewer のものなら、TASK_ID が `gate-` で始まる TASK は手順 12 の reviewer の QUESTION の項(`:759-763`)、`xreview-` で始まる TASK は「cross-review」の小節の項(`:1265-1271`)のとおり、TASK を直して新しい TASK_ID で送り直す。何が違ったか分からないときの BLOCKED と、cross-review の送り直しの上限もその項のとおり。それ以外の QUESTION は DECISION で答え、TASK は送らない。どちらの TASK かは、QUESTION の TASK_ID か座席の行の TASK_ID で見る | `TestRenderRolePrompt_Leader_ReseatRuleSeatCheckReadsOnlyThisTasksReplies`(`prompts_leader_reseat_test.go:430`)の順序つきの句に、`:447`(reviewer の QUESTION なら TASK を直して送り直す)、`:448`(BLOCKED と上限、どちらの TASK かの見方)、`:449`(それ以外は DECISION)。同じテストの `:474-481` が、回し先の 2 つの項(手順 12 の「reviewer の QUESTION(本文のコマンドが gate の TASK の 2 つと違うので打たなかった)」と、小節の「本文の行についての reviewer の QUESTION(TASK の行が欠けているか形が違うので、誰も呼ばなかった)」)が、TASK を直して送り直す文であることを見る | 閉じた。回し先の項は実在し、TASK を直して新しい TASK_ID で送り直す文を持つ(`leader.md` を読んで確かめた) |
| run 5 の cross-review 2 [P2] 手元の既定のブランチが origin より進んでいるとき、継承した commit を数える | `reseat_progress.go:570-603`(`reseatProgressCommitsIn`): `HEAD --not <base.ref>` に、origin と手元の既定のブランチのうち在る ref(`gateRefExists` で確かめる)を足し、末尾に `--`(`:575-588`)。`log` と `rev-list --count` が同じ `span` を写して使う(`:589`、`:594`)。`leader.md:251-257`: 計画の項の `git log` が同じ範囲で、手元だけにある commit を除く理由と、「branch の commit」の行も同じと書く | `TestReseatProgress_CommitsLeaveOutTheLocalDefaultBranchAheadOfOrigin`(`reseat_progress_start_test.go:84`): origin の main に 1 commit、手元の main にもう 1 commit、手元の main から branch を作って 2 commit の repo(`progressStartRepoAheadOfOrigin`、`:53`。手元の main が 1 commit 進んでいることを先に `Fatalf` で確かめる)で、total 2 と branch の 2 行だけを求める。commit のない branch は 0 件を求める(`:109-114`)。`TestReseatProgress_StartFeatureAddsNothingForAnOrgWithNoProgress` の新しい行(`:262`)は、その repo で節が入らないことを見る。`TestRenderRolePrompt_Leader_ReseatRulePlanItem`(`prompts_leader_reseat_test.go:534`)の `:542-544` が、新しい `git log` と 2 つの説明を固定する | 閉じた。git の probe(下)で、新しい範囲は 2、旧 `origin/main..HEAD` は 3。計画の項の `git log` も、新しい形は branch の計画だけ、旧は手元の main にだけある別の機能の計画も出す |
| cycle 4 の self-review C4-1 [LOW] 改訂の並びに `--` がない | `reseat_progress.go:588`(`span = append(span, "--")`) | `reseat_progress_start_test.go:100-107`: 根に `HEAD` というファイルを置いても、total 2 と 2 行を求める | 閉じた。probe: `HEAD` というファイルのある repo で、`--` なしは `fatal: ambiguous argument 'HEAD': both revision and filename`、`--` つきは通る |
| C4-2 [LOW] `reseatProgressSupplied` の doc が古い範囲 | `reseat_progress.go:115-124` が「the branch's own commits, those the default branch does not have」と言う | コメント | 閉じた |
| C4-3 [LOW] doc の折り返しと呼び名 | `:546` が「the run 5 cross-review」で、2 つのテストの呼び名と同じ。`:536-553` を組み直した | コメント | 呼び名は閉じた。折り返しは一部残る(V40) |
| C4-4 [LOW] gate への QUESTION の送り直しに「上限」を言う | `leader.md:225-226`: 「何が違ったのか分からないときの BLOCKED と、cross-review の TASK の送り直しの上限も、その項のとおりです」。手順 12 の項には BLOCKED(`:762-763`)、cross-review の項には BLOCKED と `-r3` の上限(`:1268-1271`)がある | `prompts_leader_reseat_test.go:448` | 閉じた |
| run 5 の verify V32 (a) | `leader.md:213-218`: 読んだうちでいちばん古い行の時刻が座席の行の時刻よりあとなら、数を増やして読み直す。`不明` の行はここで初めて読む | `prompts_leader_reseat_test.go:443`、`:444` | 閉じた |
| run 5 の verify V33 | `prompts_leader_reseat_test.go:374-380`: `NeverDiscardsAReply` の doc が時刻の条件を言う | コメント | 閉じた |

### Spec compliance(run 5 から変わった点)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 節に承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met(変更なし) | 節を作る関数(`reseatProgressSection`、`readReseatProgress`)は run 5 と同じ。変わったのは commit の行に載せる commit の範囲だけ(AC2 の行) |
| AC2 経過の無い org に節が入らない。disband のあとの start で commit だけの節 | Met。言葉の意味が変わった | 「branch の commit が 0 件」は、origin の既定のブランチにも手元の既定のブランチにもない commit が 0 件、の意味になった。feature の worktree は手元の既定のブランチから作られるので、手元が origin より進んでいても、最初の start に節は入らない(`..._StartFeatureAddsNothingForAnOrgWithNoProgress` の新しい行、`:262`)。disband のあとの start で commit だけの節が入る道(R8)は、branch 自身の commit があれば今までどおり。`StartFeature` を通した結合は行 521 (a) のまま |
| AC3 今の命の外は載らない | Met(変更なし) | |
| AC4 commit は最初の 20 件と総数。git が読めないときは理由が 1 行 | Met。言葉の意味が変わった | 20 件と総数は、branch 自身の commit のもの(`--max-count` と `rev-list --count` が同じ `span`)。読めないときの 1 行は変わらず、根に `HEAD` というファイルのある worktree でも読める |
| AC5 値は 1 行にして切る、fence の中 | Met(変更なし) | `reseatProgressText` と fence は変わらない |
| AC6 `leader.md` に規則があり、テストが文を固定する(仕様 AC-14) | Met | D7 の 4 項目の計画・gate・受信箱・座席の項のうち、計画の項は `git log` の範囲が変わった(「承認が消えていても書き直さない」の判定に使う commit が、branch 自身のものだけになった)。gate・受信箱・座席の項は run 5 と同じ。座席の確認の項は QUESTION の経路が変わった(上の表)。固定の句 118 が順に出る |
| AC7 既存のテストが通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | 最後の test(`3998e20a`、run 5)のあとで `prompts_leader_reseat_test.go`(+29/-6)と `reseat_progress_start_test.go`(+75)が変わり、`reseat_progress.go` の本体(+37/-16)が変わった。`featureLeaderTask` と `feature.go` は変えていない。`/test` の打ち直しが要る |
| AC8 仕様の印、AC-12 は `[ ]`、AGENTS.md と skill の説明が節に合う | Met(Status と FR-9 の範囲の記録は V35) | 印と AC-12 の `[ ]` は変わらない(`docs/specs/2026-11-12-org-reseat.md:88`、`:89`、`:114`、`:118`、`:119`)。AGENTS.md、README、skill、`--help`、org-runtime 仕様は commit の範囲を述べない(下の drift の表) |
| AC9 gate が通ったあと全座席を reseat した場合 | Met(変更なし) | gate の項は `892703bc` と同じ(`leader.md` の hunk は 2 つで、どちらも gate の項の外) |
| 仕様 FR-9 の 5 項目 | Met | 節の 5 項目は変わらない。FR-9 の「origin の既定のブランチから HEAD まで」は V35 |
| 仕様 FR-10 の 4 文 | Met | 4 文の項は文字まで同じ。計画の項の `git log` の範囲だけ変わった |
| 仕様 AC-13、AC-14 | Met | |

計画の本文(承認 digest が覆う D4・AC2・AC4・D6・R8)は、commit の範囲を「gate と同じ base から HEAD まで」「0 件」と書く。`4096ae88` のあとの意味は上の表のとおりで、計画の本文は変えない(Progress の `:168` が変更を記録している。INFO、V41)。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **PASS**(rc=0、`==> All verifiers passed.`) | gofmt ok、`golangci-lint` は `0 issues.`。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 つの settings、Codex hook の 3 つの検査、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`(6 つの文書が pipeline の全ステップを参照)、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`(PASS)、tech-debt の計画の参照 |
| branch の secret scan | PASS | `48a8a661..34a17128` clean |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイルの U+FFFD | 0 件 | `git diff main...HEAD --name-only --diff-filter=AM` の 29 ファイル |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし(rc=0) | tree `816abece411f`。`main` と `origin/main` は `48a8a661`(fetch していない手元の ref) |
| 計画の digest | 一致 | `./scripts/plan-visual.sh digest <plan>` が `de0cca0b6f78` |
| skill の写し | 変更なし | 3 commit は skill を触っていない(`git diff 892703bc..HEAD --name-only` に skill が無い)ので、run 5 の `cmp`(8 組が同じ)のまま |
| `ralph org reseat --help` | 変更なし | `internal/cli` は触っていない(build していない) |
| `git status --porcelain`(worktree、report を書く前) | 空 | run-static-verify.sh の log は gitignore の対象 |

### Documentation drift(`/sync-docs` に渡す一覧。直していない)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-11-12-org-reseat.md:88`(FR-9) | No(V35) | 「branch の commit(origin の既定のブランチから HEAD まで、1 行ずつ最大 20 件と総数)」は、`4096ae88` のあと偽。範囲は origin の既定のブランチにも手元の既定のブランチにもない commit。FR-9 は日付つきの追記で補う作りで、この範囲の追記が無い(self-review の cycle 4 の引き継ぎと同じ) |
| 同 Status(`:7`〜`:23`) | No(V35) | 6 回目のパイプラインの段落が無い。書く中身: run 5 の cross-review の 2 件(reviewer の QUESTION の経路、commit の範囲)、`4096ae88`、cap 4 の最後の回(`cycle-count.json` は 4/4)、cycle 4 の self-review の C4-1〜C4-4 と `70b1511c`。`:23`(5 回目の段落)の (2) は「あれば DECISION で答えて TASK は送らない」と言い切るが、reviewer が打たなかった・呼ばなかったと言う gate と xreview の TASK の QUESTION は、今は TASK を直して送り直す |
| 同 FR-10(`:89`)、AC-12(`:114`)、AC-13(`:118`)、AC-14(`:119`) | Yes | 4 文の要約。範囲を述べない |
| `docs/tech-debt/README.md` 行 520 の (e) | No(V36) | 下に分けて書く |
| 行 520・522 の cross-review triage の位置 | No(V37) | 下に分けて書く |
| 行 365・426・514・524 の数、行 519・522・526 の Trigger の注 | No(V38) | 下に分けて書く |
| 行 521・523・525 | Yes | Trigger に当たらない(521: `startProgressSection` と `lastDisbandedIndexes` を触っていない。523: `Send` と `reseatProgressSentTask` を触っていない。525: 最後の TASK の項目を触っていない)。524 の Trigger は `reseat_progress_test.go` で、触っていない(数だけ古い) |
| `AGENTS.md:92`(`internal/org/` の行) | Yes | `reseat_progress.go` の項目の列挙と「the branch's commits」まで。範囲を述べない |
| `README.md:303` | Yes | 「the newest 20 commits of the branch and their total」。範囲を述べない |
| `/org` skill(`.claude/skills/org/SKILL.md:1832-1846` と 3 つの写し)、`/director` skill(`:1565-1568` と 3 つの写し) | Yes | 「branch の commit(新しい順に 20 件と総数)」「branch の commit」。範囲を述べない。`/org` の `:664` の「origin の既定のブランチにある commit も数えない」は watch の scope_change の文で、経過の節とは別 |
| `ralph org reseat --help`(`internal/cli/org_reseat.go:104-109`) | Yes | 「the branch's commits」。範囲を述べない |
| `docs/specs/2026-08-01-org-runtime.md:125` | Yes | 「branch の commit(新しい順に 20 件と総数)」。範囲を述べない |
| `docs/recipes`、`docs/quality` | Yes | 範囲の文も、経過の節の commit の語も無い(grep) |
| Go のコメント | 1 点(INFO、V40) | `reseat_progress.go` の折り返しの残り。範囲の説明は、`:115-124`、`:536-553`、`reseat_progress_start_test.go:44-52`、`:77-83` が、コードと合う |
| `leader.md` 自身 | Yes(INFO、V39 (c)) | 手順 9・12・14 の `origin/<既定のブランチ>..HEAD`(`:592`、`:719`、`:768`)は前からある形で、この差分の外 |
| 計画 `docs/plans/active/2026-11-15-reseat-leader-progress.md` の Progress checklist | 8 項目は記録済み(INFO、V41) | run 5 の verify(`:163`)、`8a6676e4`(`:164`)、run 5 の test(`:165`)、run 5 の sync-docs(`:166`)、cross-review(`:167`、`5528e51c`)、`4096ae88`(`:168`)、cycle 4 の self-review(`:169`、`f4edc533`)、`70b1511c`(`:170`)。三つの artifact の行(`:184`〜`:186`)は run 5 で止まる |

### tech-debt の Trigger と数(`/sync-docs` が直す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を python で読んだ。`leader.md`、`prompts_leader_reseat_test`、`reseat_progress`、`seat-check`、`座席の確認`、`prompts_leader_inbox_test`、`resolveGateBase`、`gate.go`、`ralph-worktree`、`default branch`、`QUESTION` を挙げる Trigger は 29 行(行 188・197・199・200・201・222・224・236・238・262・292・313・365・367・377・382・384・405・426・427・462・507・508・514・519・520・522・524・526。うち 197・292・507 は取り消し線つきの RESOLVED)。中身を読んで、当たるものを選んだ。

- 行 519(Trigger: 「The next change to `reseat_progress.go`」): `4096ae88` と `70b1511c` が `reseat_progress.go` を変えたので当たる。払っていない((a)〜(d) のどれも直していない。(a) の 107 行は `reseatProgressSection` が変わらないので同じ)。「hit again, did not pay」の 1 句
- 行 522(Trigger: 座席の確認の項、`prompts_leader_inbox_test.go`): `4096ae88` が項の QUESTION の経路を書き換え、`70b1511c` が文言を直したので、4 回目の当たり(3 回目の注、cycle 3 の注、run 5 の注の続き)。払っていない(`leader.md` の受信箱の節の「あなたが送った TASK も 50 件に入るので」(`:1467`)と `prompts_leader_inbox_test.go:107` の固定は前のまま)。座席の確認の項は「sent TASK が agmsg の履歴に残らない」と今も正しく言う
- 行 526(Trigger: 座席の確認の項の、implementer のスライスの送り直し): `4096ae88` の変更は「QUESTION もなければ、TASK の種類で決めます。」より前の枝(QUESTION が見つかったとき)なので、run 5 の注(送り直しの文の頭と前の枝を書き換えたので当たる)と同じ読みなら当たる。`RESUME:` の行は足していない(座席の確認の項に `RESUME` が無いことを grep で確かめた)ので払っていない
- 行 365(Trigger: 「`Send` or the leader template tests」)と 426(Trigger: 節を足すとき): 前の run が数だけを直した扱いに合わせる。426 は見出しを足していないので当たらない
- 行 365(cell 4 の 2 か所)・426(cell 4 の 2 か所)・514(cell 4 の 3 か所)・524(cell 1 の 3 か所): 数が `8a6676e4` の時点。`wc -l` は HEAD で、4 ファイルとも最後に変えた commit は `70b1511c`: `reseat_progress.go` 582 は 603、`reseat_progress_start_test.go` 411 は 486、`prompts_leader_reseat_test.go` 686 は 706(行 365 の「685, 683, and 686 after each」の並びに `4096ae88` の 706 を足す)、`leader.md` 1,563(+172)は 1,573(+182、base `48a8a661` は 1,391 で `git diff --numstat` は +182 -0)。どれも 800 行の内。`reseat_progress_test.go` の 969(`d85fc0ef`)は変わらない
- 当たらない行: 521(`startProgressSection` と `lastDisbandedIndexes`)、523(`Send` の Details と `reseatProgressSentTask`)、525(最後の TASK の項目)、524 の Trigger(`reseat_progress_test.go`)、188・200・377・384・427・508(hunk は座席の確認の項と計画の項の中だけで、手順 4・14 の本文、split の例、`### cross-review` の小節、feature-org の見出しに当たらない。`leader.md:1265-1271` は参照だけで、書き換えていない)、514 の Trigger(`reseat.go`、`feature.go` を触っていない)
- register に足す新しい行: 無し。V39 の縁は行 520 の (e) に書く。V39 (b)(c) を別の行にするかは `/sync-docs` が決める

### Observational checks

git の probe(scratch の repo、`git` だけ。テストは回していない。出力は evidence の log):

- C: 手元の main が origin の main より 1 commit 進み、手元の main から branch を作って 2 commit。新しい範囲(`HEAD --not origin/main main --`)は 2、旧(`origin/main..HEAD`)は 3
- D: 根に `HEAD` というファイル。`--` つきは通り、`--` なしは `fatal: ambiguous argument 'HEAD': both revision and filename`
- E: 手元の main を branch の先端まで進めた(手元の ff の merge、origin には push していない)。新しい範囲は 0、旧は 5(V39 (b))
- F: origin の ref が無く、base が手元の main。範囲は `HEAD --not refs/heads/main --` で通る(存在しない ref は足さない)
- 計画の項の `git log --name-only --format= HEAD --not origin/main main -- docs/plans/active/`: branch の計画だけ。旧の `origin/main..HEAD` は、手元の main にだけある別の機能の計画も出す

### Coverage gaps

- テストの実行結果。この run は本体とテストを読み、固定の文を python で当て、範囲を git の probe で見ただけで、テストが通ることは確かめていない。`prompts_leader_reseat_test.go` と `reseat_progress_start_test.go` は最後の test report(`3998e20a`)のあとで変わった(`/test` の打ち直し)。implementer と self-review が計画に書いた `-overlay` の変異(旧 `reseat_progress.go`、旧 `leader.md`)も、この run では再実行していない。python の port は旧 `leader.md` で同じ句が落ちることを見たが、Go のテストの失敗そのものではない
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 の (d))。reviewer の QUESTION の経路(新しい項)は、文と回し先の項を読んだだけ
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))と、手元の main が origin より進んだ repo での `StartFeature` 全体(範囲の関数と、節が入らない表の 1 行までは見た)
- 手元の main が branch の先端まで進んだ repo(V39 (b))は、probe だけで、`StartFeature` を通していない

### Findings(cycle 4)

ID は run 5 の V34 から続ける。

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V35 | LOW | doc | `docs/specs/2026-11-12-org-reseat.md:88`(FR-9)は「branch の commit(origin の既定のブランチから HEAD まで、…)」と書くが、`4096ae88` のあと、範囲は origin の既定のブランチにも手元の既定のブランチにもない commit。FR-9 は日付つきの追記で補う作りで、この範囲の追記が無い。Status(`:7`〜`:23`)に 6 回目のパイプラインの段落が無い。`:23`(5 回目の段落)の (2) は、見つけた QUESTION に「DECISION で答えて TASK は送らない」と言い切るが、reviewer が gate か xreview の TASK を打たなかった・呼ばなかったと言う QUESTION は、`4096ae88` のあと TASK を直して送り直す。cycle 4 の self-review の引き継ぎは FR-9 の範囲だけを挙げた。Status の 2 点はこの run が足す | `/sync-docs`(FR-9 に日付つきの追記 1 句、Status に 6 回目の段落、`:23` の (2) に「6 回目で、reviewer の gate と xreview の QUESTION は例外にした」の 1 句) |
| V36 | LOW | tech-debt | 行 520 の (e) が `4096ae88` と `70b1511c` のあとの規則と合わない。(1) 「the leader looks, before it sends anything, for that seat's QUESTION since that time, answers it with a DECISION, and sends no TASK」は、reviewer の gate と xreview の TASK への QUESTION の例外(TASK を直して新しい TASK_ID で送り直す)を言わない。(2) 導入の「Six edges」は、列挙が (1)〜(7)、締めが「All seven」で、`6bebbd66` が (7) を足したとき導入を直さなかった。cell 2、cell 3、cell 4 の「(3) to (6)」にも (7) が入っていない。(3) 「The cap went from 3 to 4 at that cross-review and run 5 used it」のあと、run 5 の cross-review の 2 件で cycle が 4 になり、cap 4 を使い切った(`cycle-count.json` は 4/4)ことが無い。(4) 新しい縁(V39 (a)(b))がどこにも無い。(5) cell 5 の triage の参照に、run 5 の triage の finding 1・2(`4096ae88` が直した)が無い | `/sync-docs`(例外の 1 句、「Six」を seven に、(7) を (3) to (6) の参照へ、cap の経緯、V39 の縁、triage の参照) |
| V37 | LOW | tech-debt | `5528e51c`(run 5 の cross-review の triage)が `docs/reports/cross-review-triage-reseat-leader-progress.md` の先頭を run 5 の triage に置き換え、cycle 3 の triage を `## Cycle 3 history`(`:40`)に下げたので、先頭を指す位置が古い。行 520 の cell 5「findings 1 and 2 of the cycle 3 triage, at the top of the file」は、今は `## Cycle 3 history` の中。行 522 の cell 5「the top of the file is the cycle 3 triage」は、先頭が run 5 の triage。`## Third-run history`(`:67`)と `## Cycle 2 history`(`:95`)を指す文は今も合う | `/sync-docs`(2 行の位置) |
| V38 | LOW | tech-debt | 行 365・426・514・524 の数が `8a6676e4` の時点で、HEAD と合わない(`reseat_progress.go` 603、`reseat_progress_start_test.go` 486、`prompts_leader_reseat_test.go` 706、`leader.md` 1,573 で +182。最後に変えた commit は 4 ファイルとも `70b1511c`)。行 519(`reseat_progress.go` の変更)、行 522(座席の確認の項の書き換え)の Trigger に `4096ae88` と `70b1511c` が当たる(払っていない)。行 526 の Trigger も、run 5 と同じ読みなら当たる | `/sync-docs`(数、行 519・522・526 の注 1 句ずつ) |
| V39 | INFO | rule | 新しい経路と範囲に、書かれていない縁がある。(a) reviewer の QUESTION の行き先は、QUESTION の TASK_ID か座席の行の TASK_ID で決める(`leader.md:226-227`)。行が `不明` で QUESTION にも TASK_ID が無いと、決め手が無い。gate の TASK の TASK_ID は `gate-<n>` で、台帳の記録が切れるのは raw の TASK か長い ID のときだけと cycle 2 の run 3 で読んだ(この run は読み直していない)ので、届きにくい。届けば、leader が「それ以外」の DECISION だけで答え、reviewer が TASK を待つ止まり方(finding 1 の元の形)に戻りうる。(b) commit の行は、手元の既定のブランチにある commit を、branch 自身のものでも除く。probe の E: 手元の main を branch の先端まで進めた repo では範囲が 0(旧は 5)。org の流れでは merge が GitHub の PR で手元の既定のブランチは進まないので届きにくいが、届けば「branch の commit」の行が消え、implementer のスライスが済んだと読めず、同じスライスの TASK が重なる(余分な TASK、通知は消えない)。(c) 手順 9・12・14 の `origin/<既定のブランチ>..HEAD`(`leader.md:592`、`:719`、`:768`)は、手元の継承 commit を数える形のまま。`wip:` の確かめと sync-docs の前の commit の確かめで、継承 commit が `wip:` や sync-docs の件名を持つとは考えにくく、向きは余分な BLOCKED(cycle 4 の self-review の「指摘にしないもの」と同じ判断) | 実機の通し(行 520 (d))で見る。直すなら `leader.md` を変え、cap 4 を使い切っているので、上げるかは lead の判断。直さないなら行 520 の (e) に書く(V36) |
| V40 | INFO | comment | `reseat_progress.go` の doc の折り返しが一部残る。`:120` が 83 桁(前後の行は 70〜79 桁)、`:550` が 43 桁で段落の途中に短い行、`:553` が `// one.` の 1 語。C4-3 (a) は `:536-553` を組み直したが、短い行と孤立した語が残った(gofmt はコメントを折り返さない) | 次にこのファイルを触るとき。判定は変わらない |
| V41 | INFO | plan | 計画の本文(digest が覆う D4・AC2・AC4・D6・R8)は commit の範囲を「gate と同じ base から HEAD まで」「0 件」と書き、`4096ae88` のあとの意味と違う。Progress の `:168` が変更を記録していて、本文は変えない。Progress の「Review artifact created」(`:184`)は run 5 の self-review(`c88c0c26`)で止まり cycle 4(`f4edc533`)が無い。「Verification artifact created」(`:185`)は run 5 の verify(`892703bc`)まで、「Test artifact created」(`:186`)は run 5 の test(`3998e20a`)まで。`:166` と `/pr` の Known gaps に載せる候補の「行 520 の (e)(縁 6 つ)」は、列挙が 7 つなので 1 つ古い。V16 (a)(b)、V17、行 520 の (e)、`RESUME:` の行(行 526)の Known gaps は変わらず、V39 (a)〜(c) を行 520 の (e) に足す | lead(`/pr` の前) |

V35〜V38 は文書と register の直しで、コードも `leader.md` も触らないので、pipeline の再実行は要らない。V39 を直す場合だけ `leader.md` を変える。

## Verdict

- Verified: 静的検査(`run-static-verify.sh` rc=0、golangci-lint 0 issues、`git diff --check`、U+FFFD 0 件、`main` との試し merge、branch の secret scan、計画の digest)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応(`8a6676e4`、`4096ae88`、`70b1511c` を含む)。run 5 の cross-review の 2 件が、コード(`reseat_progress.go:570-603`)、規則の文(`leader.md:220-228`、`:251-257`)、テストの 3 つで閉じたこと。cycle 4 の self-review の LOW 4 件が閉じたこと(折り返しの一部を除く)。固定の文 118 句が順に出て、旧 `leader.md` では新しい句で落ちること(python による同じ確かめ)。git の probe で、新しい範囲が手元の既定のブランチの先行 commit を除き、`HEAD` というファイルで止まらないこと
- Partially verified: AC2(`StartFeature` を通した commit だけの節の結合、行 521 (a)。範囲の関数と、節が入らない表の 1 行までは読んだ)
- Not verified: AC7(既存のテストが通ること。2 つのテストファイルと `reseat_progress.go` が最後の test のあとで変わった。`/test` の打ち直し)、節を読んだ leader の実機での動き(reviewer の QUESTION の新しい経路を含む)
- Verdict: **partial-pass**。静的検査は通り、仕様の AC は読んだ範囲ですべて満たす。残りは `/sync-docs` の文書と register(V35〜V38)で、fail にする指摘はない。insight event の `verdict` は `pass`、LOW 4

## Run 5 (cycle-count 3, cap 4): HEAD `28447a1d`

cycle 3 の verify(`c693d220`)のあとに入ったコードは、`leader.md` の「座席の確認」の項と、そのテストだけ。3 つの commit を読んだ。`91578db5` は cycle 3 の verify の V26 の直しで、済んだと確かめられる手だけ飛ばし、済んだかが分からない手は重なっても頼む(例は計画の置き場所だけ)。`5ab60578` は cycle 3 の cross-review の 2 件(前の命の返事、既読にされた QUESTION)の直し。`262b9df7` は run 5 の self-review の LOW 4 件(C5-1 数える返事の名指し、C5-2 QUESTION の確かめの範囲とコマンドの場所、C5-3「命」を life に、C5-4 テストの doc と重複)の直し。Go の本体(`reseat_progress.go`、`feature.go`、`reseat.go`)は `c693d220` から 1 行も変わっていない。

`leader.md` の hunk は新しい版の `:188-195`、`:198-202`、`:207-223` の 3 つで、すべて座席の確認の項(`:162-232`)の中。`git show c693d220:` と `git show HEAD:` の 2 版を python で比べ、項の外と、計画(34 行)・gate(39 行)・受信箱(4 行)・座席(10 行)の 4 項が文字まで同じことを確かめた。テストは実行していない(`/test` の担当)。固定の文は、`assertPhraseOrder` と `containsPhrase` と同じ確かめ(空白を全部捨てた部分文字列が順に出るか)を `python3 -I` で座席の確認の項に当てて見た(下)。

### cycle 3 の cross-review の 2 件が閉じたか

| cycle 3 の finding | 規則の文(`leader.md`) | 固定するテスト(`prompts_leader_reseat_test.go`) | 判定 |
| --- | --- | --- | --- |
| 1 [P2] 前の命の返事を、今の TASK への返事と取る。agmsg の team は disband のあとも履歴を残すので、TASK_ID が同じ前の命の RESULT が、済んでいない仕事を済みと読ませる(知らせが消える向き) | `:197-202`: 履歴の RESULT か BLOCKED のうち数えるのは、行の時刻(`[<時刻>]`)が座席の行の時刻と同じかそれよりあとのものだけ(どちらも UTC)。それより前は「disband より前の life など、同じ TASK_ID で前に送った TASK への返事」で、ないものとして扱う。数える返事は前の leader が扱ったかにかかわらず今届いたものとして扱い(`:202-205`)、捨てない(`:205`)。同じ秒の返事は数える。時刻の出どころ: 座席の行は台帳の `sent` の `ts`(UTC の RFC3339、`reseat_progress.go:343`、変更なし)、履歴の行は `history.sh` の表示行 `[$ts]` で、`ts` は storage ドライバの `date -u +%Y-%m-%dT%H:%M:%SZ`(agmsg の `drivers/storage/jsonl.sh:32`、`sqlite.sh:21` を読んだ) | `..._ReseatRuleSeatCheckReadsOnlyThisTasksReplies`(`:425`)の 17 句の順に「数えるのは…同じかそれよりあとのものだけ」「それより前のものは…ないものとして扱います」「数える返事があれば…今届いたものとして扱います」。`..._ReseatRuleSeatCheckItem`(`:289`)の 28 句の順にも同じ 3 句。`..._ReseatRuleSeatCheckNeverDiscardsAReply`(`:379`)は返事を捨てないことと、節の最後の TASK の行が UTC の時刻で終わること(`reseatTimeShape` `:365-367`、検査 `:403-412`)。外した枝の語 4 つ(`返事よりあとの時刻の行が 1 つでもあれば` など)が出ないこと(`:392-401`) | 閉じた |
| 2 [P2] 前の leader の見張りが既読にした座席の QUESTION を見ずに、TASK を送り直す | `:210-220`: 数える返事がなければ、何かを送る前に QUESTION を確かめる(数える返事があるときはしない)。確かめは、その座席の QUESTION のうち時刻が行の時刻と同じかそれよりあとのものを `history.sh {{TEAM}} leader 50` で探し(QUESTION には TASK_ID がないことがあり、既読の QUESTION も履歴に残る)、あれば、ミッションの 5 のとおり DECISION で答え、TASK は送らずに RESULT か BLOCKED を待つ。`不明` の行も、送る前にこの確かめだけをする(`:186-191`、返事は探さない)。なければ、`不明` の行は座席の役割で(`:191-194`)、それ以外は TASK の種類で決める(`:221-228`)。既読の QUESTION が履歴に残る根拠は、同じファイルの受信箱の節(`:1450-1454`: `inbox.sh --quiet` は既読にし、`history.sh` は既読の印を変えない)と `history.sh` の表示行(○ 既読と ● 未読の両方を出す) | `..._ReseatRuleSeatCheckReadsOnlyThisTasksReplies` の 17 句の順に「送る前に、下の QUESTION の確かめだけを、この行の時刻でします」「このときは、何かを送る前に QUESTION を確かめます(数える返事があるときは、この確かめはしません)」「QUESTION の確かめでは…」「ミッションの 5 のとおりその QUESTION に DECISION で答え、TASK は送らずに…待ちます」「QUESTION もなければ、TASK の種類で決めます」。ミッションの 5 の文(`:458`)。`..._ReseatRuleSeatCheckUnknownTaskID`(`:476`)の 7 句の順(`不明` の枝で、履歴で返事を探さず、QUESTION の確かめだけをして、役割で決める)。古い語 3 つ(`履歴は読まず` など)が出ないこと(`:450`) | 閉じた(`不明` の行を含む)。確かめに書かれていない縁がある(V32) |

固定の文の確かめ(`python3 -I`、テストの実行ではない): `{{ORG_ID}}`、`{{TEAM}}`、`{{AGMSG_SCRIPTS}}` を描画値に置き換え、座席の確認の項に、`..._ReseatRuleSeatCheckItem` の 28 句、`..._NeverDiscardsAReply` の 6 句、`..._ReadsOnlyThisTasksReplies` の 17 句、`..._UnknownTaskID` の 7 句(計 58 句)が順に出て、否定の 11 句(`最後に送った TASK のあとに`、`あなたが送った TASK も 50 件に入る`、`TASK を持っていないので、計画の残りの`、`を送り直します`、`直しの commit があればその直しは頼まない`、`その直しは頼み直さない`、`返事よりあとの時刻の行が 1 つでもあれば`、`その返事は扱い直さず`、`履歴は読まず`、`返事ではありません。あれば、`、`history.sh` の前に「上の」を置いた句)が出ないことを確かめた。cycle 3 の verify では 38 句だった。implementer が計画に書いた変異(3 つの句を 1 つずつ消した `leader.md` を `go test -overlay` で当てて落ちた)は、この run では再実行していない。

### Spec compliance(cycle 3 の verify から変わった点)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 節に承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met(変更なし) | Go の本体が `c693d220` から変わっていない(docs 以外の変更は `leader.md` とそのテストの 2 ファイル) |
| AC2 `start --plan` の立て直しも同じ節。経過の無い org に節なし | Met(結合は部分、行 521 (a) のまま) | 同上 |
| AC3 今の命の外は載らない、AC4 commit は新しい順に 20 件と総数、AC5 値は 1 行にして切る | Met(変更なし) | `reseat_progress.go` と `reseat_progress_test.go` の差分が無い(582 行と 969 行のまま) |
| AC6 `leader.md` に規則があり、役割の prompt のテストが文を固定する(仕様 AC-14) | Met | 上の表。座席の確認の項が、返事の数え方(時刻の条件)と QUESTION の確かめ(`不明` の行を含む)を持ち、58 句の順で固定される。規則の D7 の 4 項目(承認済みの計画を書き直さない、gate が今の HEAD で pass なら TASK を送り直さず止めて PR の手順へ、片付いていない件は答えを待つ、立て直した座席は立ったことを確かめてから TASK を送り直す)は、計画・gate・受信箱・座席の項が文字まで同じ |
| AC7 既存のテストが通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | 最後の test(`ee9e41dc`、cycle 3、pass)のあとで `prompts_leader_reseat_test.go` が変わった(627 行から 683 行、`5ab60578` で新しいテスト `..._ReadsOnlyThisTasksReplies` を足し、`262b9df7` で 3 つの固定と doc を直した)。`featureLeaderTask` と `feature.go` は変えていない。`/test` の打ち直しが要る |
| AC8 仕様の印、AC-12 は `[ ]`、AGENTS.md と skill の説明が節に合う | Met(Status の記録は V28) | 印と AC-12 の `[ ]` は変わらない(`docs/specs/2026-11-12-org-reseat.md:86`、`:87`、`:112`、`:116`、`:117`)。README:303、AGENTS.md:92、`/org` skill(`:1836-1842`)と `/director` skill(`:1565-1568`)の各 4 つの写し(8 組が `cmp` で同じ)、`ralph org reseat --help`(`internal/cli/org_reseat.go:104-109`)、org-runtime 仕様の `:125` は、節の項目の列挙と「まず leader 以外の座席の状態を見て、次に計画、gate、受信箱、座席の順に」までで、返事の数え方も QUESTION の確かめも述べない。`扱い直さ`・`あとの時刻`・`後の時刻`・`later TASK`・`扱い済み`・`履歴は読まず`・`QUESTION の確かめ`・`数える返事`・`既読にした QUESTION`・`返事を捨て`・`disband より前` の grep は、README、AGENTS.md、skill(`.claude`、`.agents`、`templates/base`)、`internal/cli`、org-runtime 仕様、`docs/recipes`、`docs/quality` で、`/org` skill の `inbox` の行(`:373`、4 つの写し)だけ。座席の確認とは無関係 |
| AC9 gate が通ったあと全座席を reseat した場合 | Met(変更なし) | gate の項(`:267-305`、39 行)は `c693d220` と同じ。座席の確認の頭(`:165-168`)の「gate の項目に当たるときは確認の残りをせず、TASK を送らずに gate の項目のとおり止める」も、2 つの直しのあとに残る |
| 仕様 FR-9 の 5 項目 | Met(変更なし) | 節を作る側は変わらない。FR-9 の追記(`:86`)は節の行の中身を述べ、`不明` も書いてあり、本文と合う |
| 仕様 FR-10 の 4 文 | Met | 4 文(計画を書き直さない、gate が pass なら PR の手順、片付いていない件は答えを待つ、立て直した座席の TASK を送り直す)の項は文字まで同じ。座席の確認の新しい QUESTION の確かめは、座席が agmsg で leader に出した QUESTION に答える手で、受信箱の項の「受信箱の件(leader が `ralph org escalate` で上げたもの)は答えを待つ」とは別の対象。どちらも答えを待つ形で、矛盾しない |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **PASS**(rc=0、`==> All verifiers passed.`) | gofmt ok、`golangci-lint` は `0 issues.`。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 つの settings、Codex hook の 3 つの検査、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の計画の参照 |
| branch の secret scan | PASS | `48a8a661..28447a1d` clean |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイルの U+FFFD | 0 件 | `git diff main...HEAD --name-only --diff-filter=AM` |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし(rc=0) | tree `6abe04c2c0a7`。`main` と `origin/main` は `48a8a661`(fetch していない手元の ref) |
| 計画の digest | 一致 | `./scripts/plan-visual.sh digest <plan>` が `de0cca0b6f78` |
| skill の写しの `cmp` | 同じ(8 組。`/org` と `/director` の `.agents`、`templates/base/.claude`、`templates/base/.agents` を `.claude` と比べた) | 3 つの commit は skill を触っていない |
| `ralph org reseat --help` | 変更なし(build していない) | Go の本体が変わっていないので cycle 3 の確認のまま |
| `git status --porcelain`(worktree) | 空 | run-static-verify.sh の log は gitignore の対象 |

### Documentation drift(`/sync-docs` に渡す一覧。直していない)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-11-12-org-reseat.md:19`(Status の 3 回目の段落)の (3) | No(V28) | 「規則はその座席の履歴を読まず、TASK_ID を推し量らず…」の「履歴を読まず」は、`5ab60578` のあと偽。`不明` の行の枝も、QUESTION の確かめのために履歴を読む(返事は探さない。`leader.md:186-191`、`:213-217`)。同じ (3) の「TASK_ID を推し量らず、同じ TASK_ID でも送り直さず、新しい TASK_ID で送る」は今も合う |
| 同 `:21`(4 回目の段落)の (1) | No(V28) | 「見つけた返事は、前の leader が扱ったかどうかにかかわらず、今届いたものとして扱い、捨てない」は、数えるのが履歴の時刻が座席の行の時刻と同じかそれよりあとの返事だけで、前のものはないものとして扱う、という限りを言わない。(2)(3) と「FR-9 の文、FR-10 の 4 文、印は変えていない」は今も合う |
| 同 Status に run 5 の段落が無い | No(V28) | cycle 3 の cross-review の 2 件(前の命の返事を時刻で限ること、送る前の QUESTION の確かめ、`不明` の行を含む)、cap の 3 から 4 への引き上げ(`cycle` は 3 のまま)、`262b9df7` の言い直しを書く段落。Status は run ごとに 1 段落を足す作り(`:9`〜`:21`)。run 5 の self-review の引き継ぎ(`docs/reports/self-review-2026-11-15-reseat-leader-progress.md` の「`/sync-docs` への引き継ぎ」)と同じ |
| 同 FR-9 の追記(`:86`)、FR-10(`:87`)、AC-12(`:112`)、AC-13、AC-14 | Yes | 節の中身と 4 文の要約。時刻が規則の比べ方に使われることは、節の「TASK_ID と時刻」で既に言っている |
| `docs/tech-debt/README.md` 行 520 の (e) | No(V29) | 下に分けて書く |
| `docs/tech-debt/README.md` 行 520・522・526 の cross-review triage の位置 | No(V30) | 下に分けて書く |
| `docs/tech-debt/README.md` 行 365・426・514・524 の数、行 522 と 526 の Trigger の注 | No(V31) | 下に分けて書く |
| `README.md:303`、`AGENTS.md:92`、`/org` と `/director` skill の各 4 つの写し、`ralph org reseat --help` とその pin、`docs/specs/2026-08-01-org-runtime.md:125` | Yes | 節の項目の列挙と、規則の項の順までを述べる(AC8 の行) |
| `leader.md` 自身 | Yes | 座席の確認の項の中の言い換え(「命」を life に、QUESTION の確かめの範囲)は文の中で閉じている。受信箱の節の `:1455`「あなたが送った TASK も 50 件に入るので」(行 522)は前のまま |
| Go のコメント | 1 点(INFO、V33) | `prompts_leader_reseat_test.go:369-378` の `..._NeverDiscardsAReply` の doc は「履歴で見つけた返事は常に今届いたものとして扱う」と書き、時刻の条件を言わない(条件は次のテスト `:415-424` の doc が書く)。`..._UnknownTaskID` の doc は C5-4 で直り、`reseat_progress.go` のコメントは規則の比べ方に触れない |
| 計画 `docs/plans/active/2026-11-15-reseat-leader-progress.md` の Progress checklist | 4 項目は記録済み(INFO、V34) | cycle 3 の cross-review(`:159`、`780a7adf`)、`5ab60578`(`:160`)、run 5 の self-review(`:161`、`c88c0c26`)、`262b9df7`(`:162`)。「Review artifact created」(`:176`)は cycle 3 の `eaa9bc71` で止まり run 5 の self-review が無い。「Verification artifact created」(`:177`)は cycle 3 の `c693d220` まで、「Test artifact created」(`:178`)は cycle 3 の `36d80504` まで。この verify、run 5 の test と sync-docs、cross-review の項は後から。AC の checkbox は `[ ]` のまま(`/pr` の前に付ける。drift ではない) |
| 前の run の記録(test report の Cycle 3 の節、sync-docs report の Cycle 3 の節、cross-review triage の cycle 3 の節、self-review の run 5 の節) | 履歴 | 当時の規則を書く。test report は HEAD `ee9e41dc` の run(AC7 の行) |

### tech-debt の Trigger と数(`/sync-docs` が直す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を python で読んだ。`leader.md`、`prompts_leader_reseat_test.go`、`seat-check`、`座席の確認`、`prompts_leader_inbox_test` を挙げる Trigger は行 188・197・200・365・377・384・426・427・508・514・519・520・522・526 の 14 行(行 197 は取り消し線つきの RESOLVED)。

- 行 522(Trigger: 座席の確認の項、`prompts_leader_inbox_test.go`): `5ab60578` と `262b9df7` が座席の確認の項をもう一度書き換えたので当たる(3 回目の注と cycle 3 の注の続き)。払っていない(`leader.md:1455` の文と `prompts_leader_inbox_test.go` の固定は前のまま)ので、「hit again, did not pay」の 1 句。払わない理由は、直しが `leader.md` と 2 つのテストを変え、pipeline が `/self-review` から回り直すこと(cap は 4 に上がっている)
- 行 526(Trigger: 座席の確認の項の、implementer のスライスの送り直し): `5ab60578` と `262b9df7` は送り直しの文の頭(「QUESTION もなければ、TASK の種類で決めます。その TASK が implementer への…」`:221-228`)と、`不明` の行・行なしの枝の前の確かめを書き換えたので、当たると読める。`RESUME:` の行を足す直しは入っていない(項に `RESUME` が無いことを grep で確かめた)ので、「hit again, did not pay」の 1 句。第 1 欄(「送り直す TASK に leftover の path を書かない」)は今も合う。第 3 欄の「Cycle 3 is the last pipeline run under the cap of 3」は、cap が 4 に上がり run 5 が走ったので、読み手に古い。当たると読むかは `/sync-docs` が決める
- 行 365(`prompts_leader_reseat_test.go` 627 は 683 行、2 か所。「cycle 3 の直しが 627 にした」の列に `5ab60578` と `262b9df7` を足し、「`wc -l` at `91578db5`」は `262b9df7`)、行 426(`leader.md` 1,545(+154)は 1,561(+170)、「`wc -l` at `91578db5`, the last commit to change it」は `262b9df7`。同じ行の `prompts_leader_reseat_test.go` 627 は 683)、行 514(627 は 683。「the count of `prompts_leader_reseat_test.go`, which is from `91578db5`」は `262b9df7`)、行 524(627 は 683、「`wc -l` at `91578db5`」は `262b9df7`): 数だけ古い。`wc -l` は HEAD。どれも 800 行の目安の内で(683)、行 524 の `reseat_progress.go` 582、`reseat_progress_start_test.go` 411、`reseat_progress_test.go` 969 は変わらない。base `48a8a661` の `leader.md` は 1,391 行で、`git diff --numstat` は +170 -0
- 行 520 の (e): V29 のとおり。(a)(b)(c) の RESOLVED と (d) は変わらない
- 行 520・522・526 の第 5 欄の triage の位置: V30 のとおり
- 当たらない行: 519(`reseat_progress.go` も gate の行を読む規則も触っていない。gate の項は文字まで同じ)、521(`startProgressSection` を触っていない)、523・525(`reseatProgressSentTask` と最後の TASK の項目を触っていない)、524 の Trigger(`reseat_progress_test.go` を触っていない)、377・384・508・200・188・427(hunk は座席の確認の項の中だけで、手順 4・14 の本文、split の例、`### cross-review` の小節、feature-org の見出しに当たらない)、514 の Trigger(`reseat.go`、`feature.go` を触っていない)、426 の Trigger(節を足していない。数だけ)
- register に足す新しい行: 無し。run 5 の self-review の LOW 4 件(C5-1〜C5-4)は `262b9df7` で直った(self-review の「Tech debt identified」の行は不要)。V32 の縁は行 520 の (e) に書く。V29 の縁を行 520 に足すか、別の行にするかは `/sync-docs` が決める

### Observational checks(規則に書かれていない縁)

- `91578db5` のあとでも、cycle 3 の V26 (a)(b)(d) は閉じている。例が計画の置き場所だけになり、直しの TASK は済んだか分からない手として重なっても頼む。V26 (c)(`不明` の行の時刻は `(記録が切れている、<時刻>)` の中にあるのに、規則は位置を言わない)は、`:190` の「この行の時刻」で今も同じ(V32 (e))
- 数える返事の条件は「行の時刻と同じかそれよりあと」なので、同じ秒の返事を落とさない。行の時刻は `sent` が Enter と送信の確認(`confirmSubmitted`、既定 3 秒で打ち切る)のあとに書くので、Enter より最大数秒あとになる。座席の最初の返事は 1 ターンのあとで、これより前には来ないと読むが、測っていない(V32 (d))
- QUESTION の確かめが答える DECISION は、台帳の `sent` に残るが、節は TYPE が TASK の `sent` だけを出す(`reseat_progress.go`)ので、立て直し後に前の leader が答えた QUESTION を見分けられない(V32 (c))

### Coverage gaps

- テストの実行結果。この run は本体とテストを読み、固定の文を python で当てただけで、通ることは確かめていない。`prompts_leader_reseat_test.go` は最後の test report(`ee9e41dc`)のあとで 2 回変わった(`/test` の打ち直し)。計画に書かれた implementer の `go test` の結果と変異(3 つの句を消した `leader.md` が新しいテストで落ちる)も、この run では再実行していない
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 の (d))。規則の新しい確かめ(数える返事、QUESTION の確かめ)は読んだだけ
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))
- `history.sh` の `ts` の書き手: 書式(`date -u +%Y-%m-%dT%H:%M:%SZ`)と `history.sh` の表示行は agmsg の checkout で読んだ。送信側の `at` を書く呼び出し(`send.sh` など)は読んでいない

### Findings(run 5)

ID は cycle 3 の V27 から続ける。

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V28 | LOW | doc | `docs/specs/2026-11-12-org-reseat.md:19`(Status の 3 回目の段落)の (3) は「規則はその座席の履歴を読まず」と書くが、`5ab60578` のあと、`不明` の行の枝も QUESTION の確かめのために履歴を読む(`leader.md:186-191`、`:213-217`)。`:21`(4 回目の段落)の (1) は、見つけた返事を今届いたものとして扱い、捨てないと書き、数えるのが時刻の条件に合う返事だけ(前のものはないものとして扱う、`leader.md:197-202`)という限りを言わない。run 5 の段落(cycle 3 の cross-review の 2 件、cap の 3 から 4 への引き上げ、`262b9df7`)が無い。self-review の run 5 が引き継いだ点と同じ | `/sync-docs`(5 回目の段落 1 つ。(3) への「5 回目で、不明の行も QUESTION のために読む」と (1) への「条件に合う返事だけ」の各 1 句は任意) |
| V29 | LOW | tech-debt | 行 520 の (e) が、`5ab60578` と `262b9df7` のあとの規則と合わない。(1) 「the item now takes a reply it finds for a seat's last TASK_ID as just arrived, whether or not the previous leader handled it」は、時刻が座席の行の時刻と同じかそれよりあとの返事に限るという条件を言わない(前のものは、ないものとして扱う)。影響の欄の (e) の「no failure is dropped, because a found reply is always handled」は、その条件のもとでだけ真。(2) 「Why deferred」の欄の (e) と「Cycle 3 is the last pipeline run under the cap of 3」は、cap が 4 に上がり run 5 が走ったことを言わない。(3) 新しい縁(V32 の (a)〜(e))がどこにも無い | `/sync-docs`(行 520 の (e) に、時刻の条件と QUESTION の確かめの 2 句を足し、V32 の縁と cap の経緯を書く) |
| V30 | LOW | tech-debt | `780a7adf`(cycle 3 の triage)が `docs/reports/cross-review-triage-reseat-leader-progress.md` の先頭を cycle 3 の triage に置き換え、前の triage を `## Third-run history`・`## Cycle 2 history`・`## Cycle 1 history` に下げたので、先頭を指す位置が古い。行 520 の「finding 1 of the third-run triage, at the top of the file」と行 526 の「finding 2 of the third-run triage, at the top of the file」は、今は `## Third-run history`(`:40`)の中。行 522 の「the file holds the cycle 2 triage now」は、cycle 2 の triage が `## Cycle 2 history`(`:68`)の中にあり、先頭は cycle 3 の triage。行 520 の「findings 1 and 2 of the cycle 2 triage, `git show 51119734:...`」は、同じ内容が `## Cycle 2 history` にもあるので今も辿れる | `/sync-docs`(3 行の位置を「`## Third-run history` の中」「`## Cycle 2 history` の中」に直す。cycle 3 の finding 1・2(`5ab60578` が直した)を行 520 の参照に足すかは任意) |
| V31 | LOW | tech-debt | 行 365(2 か所)・426(2 か所)・514(2 か所)・524 の数が `91578db5` の時点で、`262b9df7` の HEAD と合わない。`prompts_leader_reseat_test.go` は 627 行から 683 行、`leader.md` は 1,545(+154)から 1,561(+170)。行 522 の Trigger に `5ab60578` と `262b9df7` が当たる(払っていない)。行 526 の Trigger も当たると読める(上の節) | `/sync-docs`(数 7 か所と、行 522 の注 1 句、行 526 の注 1 句) |
| V32 | INFO | rule | 規則の新しい確かめに、書かれていない縁がある。(a) `不明` の行の枝の QUESTION の確かめは履歴を初めて読む(`:215-217`)のに、返事の探索の「足りなければ数を増やします」(`:197`)を引き継がず、50 件の窓に出てしまった QUESTION を見落とす。見落とすと新しい TASK_ID の TASK を送るので、座席は古い QUESTION を持ったまま新しい TASK を受ける(座席がもう一度聞く側に倒れる)。(b) 「座席の行がなければ」の枝(`:184-186`)は QUESTION の確かめをせず、TASK を送る。TASK を受けていない座席の QUESTION は想定しにくい(reviewer が承認の前に TASK なしで立つ場合が考えられるが、聞くことが無い)。(c) 前の leader がもう DECISION で答えた QUESTION も、確かめは見つけて答え直す。DECISION は agmsg の履歴にも節にも残らない(self-review run 5 の「指摘にしないもの」と同じ)。座席に DECISION がもう 1 通届く。(d) 数える返事の境は、agmsg の `at` と台帳の `ts`(Enter と `confirmSubmitted` のあとに書く)の比べで、行の時刻より前に着いた返事は数えない。座席の最初の返事は 1 ターンのあとで、数秒より前には来ないと読むが、測っていない。数えなければ、TASK を同じ TASK_ID で送り直す(座席が同じ返事を出し直す側に倒れる)。(e) 旧 V26 (c) のとおり、`不明` の行の時刻の位置を規則が言わない(`:190` の「この行の時刻」)。読み損ねると窓が広がるか狭まる。(a)〜(e) の向きはどれも、余分な手か聞き直しで、知らせが消える向きではない | 実機の通し(行 520 (d))で見る。直すなら `leader.md` を変えるので、pipeline が `/self-review` から回り直す(cap 4 は run 5 で使った。上げるかは lead の判断)。直さないなら行 520 の (e) に書く(V29) |
| V33 | INFO | comment | `prompts_leader_reseat_test.go:369-378` の `..._NeverDiscardsAReply` の doc は、「a RESULT or BLOCKED found in the history for the seat's last TASK_ID is always taken as just arrived」と書き、時刻の条件を言わない。その条件は次のテスト `:415-424` の doc に書かれ、同じファイルの中で読める | `/sync-docs` か次にそのテストを触るとき(doc の 1 句。判定は変わらない) |
| V34 | INFO | plan | 計画の Progress の 4 項目(cycle 3 の cross-review、`5ab60578`、run 5 の self-review、`262b9df7`)は `:159-162` に記録済み。「Review artifact created」(`:176`)に run 5 の self-review(`c88c0c26`)が、「Verification artifact created」(`:177`)に cycle 3 の verify(`c693d220`)のあとの run 5 の verify が、「Test artifact created」(`:178`)に cycle 3 のあとの run 5 の test が無い(後から足す)。V16・V17 の Known gaps は変わらず、V21(旧)と V26 の縁は V32 に差し替える | lead(`/pr` の前) |

V28〜V31 は文書と register の直しで、コードも `leader.md` も触らないので、pipeline の再実行は要らない。V32 を直す場合だけ `leader.md` を変える。

### Verdict(run 5)

- Verified: 静的検査(`run-static-verify.sh` rc=0、golangci-lint 0 issues、`git diff --check`、U+FFFD 0 件、`main` との試し merge、skill の写しの一致、branch の secret scan、計画の digest)。AC1・AC3〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応(`91578db5`、`5ab60578`、`262b9df7` を含む)。座席の確認の項の外と、計画・gate・受信箱・座席の 4 項が `c693d220` と文字まで同じこと。cycle 3 の cross-review の 2 件が、規則の文(`leader.md:197-202`、`:210-220`、`:186-191`)とテスト(`..._ReadsOnlyThisTasksReplies`、`..._SeatCheckItem`、`..._NeverDiscardsAReply`、`..._UnknownTaskID`)の両方で閉じたこと。固定の文 58 句が順に出て、外した枝の語が出ないこと(python による同じ確かめ)
- Partially verified: AC2(`StartFeature` を通した commit だけの節の結合、行 521 (a))
- Not verified: AC7(既存のテストが通ること。`prompts_leader_reseat_test.go` が最後の test のあとで変わった。`/test` の打ち直し)、節を読んだ leader の実機での動き(数える返事、QUESTION の確かめ、PR の head、`不明` の分岐を含む)、送信側の `at` の書き手
- Verdict: **partial-pass**。静的検査は通り、仕様の AC は読んだ範囲ですべて満たす。残りは `/sync-docs` の文書と register(V28〜V31)で、fail にする指摘はない。insight event の `verdict` は `pass`、LOW 4

## Cycle 3 (cycle-count 3, cap 3、最後の回): HEAD `9235c95c`

run 3 の verify(`c49a6bc5`)のあとの直し 2 つを読んだ。`04d3d986` は 3 回目の cross-review の ACTION_REQUIRED(座席の確認が、返事より後の時刻の TASK の行が 1 つでもあれば返事を捨てる枝。届いた BLOCKED が消えうる)の直しで、その枝を外し、見つけた返事は前の leader が扱ったかにかかわらず届いたものとして扱い、応じる手ごとに済んでいないかを確かめて、済んでいる手だけ繰り返さない形にした。`e6f98624` は cycle 3 の self-review の C3-1 と C3-2 の直しで、「「branch の commit」に直しの commit があれば」の例を外し、時刻の例に返事の時刻の出どころを戻して固定した。変わったのは `leader.md` の座席の確認の項(`:193-205`、hunk は 2 つ)とそのテストだけで、Go の本体と gate の項目は変わらない。テストは実行していない(`/test` の担当)。3 回目の cross-review の WORTH_CONSIDERING(finding 2、`RESUME:` が無い)は、triage のとおり tech-debt と PR の Known gaps に回す対象で、直しに入っていない。register にはまだ行が無い(V24)。

run 3 の V18〜V20 は 3 回目の sync-docs(`0d443ebb`)で閉じたことを確かめた(行 519 (a) は 107 行、行 522〜524 に `d85fc0ef` の持ち越しの注、行 525 が R3-2、仕様の Status の 3 回目の段落と FR-9 の `不明`、計画の Progress)。V21 は行 520 の (e) に書かれたが、`04d3d986` が枝を変えたので古い(V23)。

### Spec compliance(run 3 から変わった点)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 節に承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met(変更なし) | `c49a6bc5` から Go の本体は変わっていない(docs 以外の変更は `leader.md` とそのテストの 2 ファイル)。節の 8 つの部分と `不明` の行の形は run 3 のまま |
| AC2 `start --plan` の立て直しも同じ節。経過の無い org に節なし | Met(結合は部分、行 521 (a) のまま) | Go の本体が変わっていないので run 3 と同じ |
| AC3 今の命の外は載らない、AC4 commit は新しい順に 20 件と総数、AC5 値は 1 行にして切る | Met(変更なし) | `reseat_progress.go` の差分が無い |
| AC6 `leader.md` に規則があり、役割の prompt のテストが文を固定する(仕様 AC-14) | Met | 座席の確認の返事の扱い(`leader.md:193-205`): 返事が見つかれば、前の leader が扱ったかにかかわらず今届いたものとして扱い、「返事を捨てることはしません」(`:196-199`)。応じる手ごとに先に済んでいないかを確かめ、済んでいる手だけ繰り返さない。例は 2 つで、計画がもう `docs/plans/active/` に戻っていれば archive の戻しはしない(`:201`)と、「座席ごとの最後の TASK」の、直しを頼む座席の行の時刻が返事の時刻(`history.sh` の行の `[` と `]` のあいだ)よりあとなら頼み直さない(`:202-205`)。時刻を比べて返事を捨てる枝(`d85fc0ef` の「返事よりあとの時刻の行が 1 つでもあれば扱い直さず」)と、「直しの commit があれば」の例(C3-1)は消えた。テスト 11 本のうち変わったのは 2 本。`..._ReseatRuleSeatCheckItem` の 1 句(`:310`)を新しい文に合わせ、`..._SkipsRepliesAlreadyHandled` を `..._NeverDiscardsAReply`(`prompts_leader_reseat_test.go:377`)に置き換えた。新しい固定は 6 句の順(返事は必ず扱う、返事を捨てない、済んでいる手だけ繰り返さない、例 (a)、例の時刻と出どころ、返事がなければ)、古い枝の 2 句と外した例の否定、最後の TASK の行が `(YYYY-MM-DDTHH:MM:SSZ)` で終わること(`e6f98624` が時刻の例の固定と、出どころの句を足した。C3-2 は閉じた)。固定は弱まっていない(古い枝の固定は枝と一緒に消え、戻らないことは否定で見る) |
| AC7 既存のテストが通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | 最後の test(`601b4643`、run 3、pass)は `04d3d986` と `e6f98624` より前。2 つの直しは `prompts_leader_reseat_test.go`(615 から 624 行)を変えた。`featureLeaderTask` と `feature.go` は変えていない。`/test` の打ち直しが要る |
| AC8 仕様の印、AC-12 は `[ ]`、AGENTS.md と skill の説明が節に合う | Met(Status の記録は V22) | README:303、AGENTS.md:92、`/org` skill(`:1836-1842`)と `/director` skill(`:1565-1568`)の各 4 つの写し、`ralph org reseat --help`(`internal/cli/org_reseat.go:104-109`)、org-runtime 仕様の `:125` は、節の 8 項目を列挙するだけで、返事の扱い、PR の head、`不明` を述べていない。`扱い直さ`・`あとの時刻`・`後の時刻`・`later TASK`・`扱い済み` の grep は、README、AGENTS.md、skill(`.claude` と `.agents`)、`internal/cli`、org-runtime 仕様で、`/org` skill の `inbox` の行(`:373`、「最後の時刻」に `後の時刻` が当たる。座席の確認とは無関係)だけ。skill の写し 8 組は `cmp` で同じ。仕様の Status の 3 回目の段落だけが外した枝を書く(V22) |
| AC9 gate が通ったあと全座席を reseat した場合 | Met(変更なし) | gate の項目(`leader.md:252-290`)は `c49a6bc5` から 1 行も変わらない。`git diff c49a6bc5..HEAD -U0 -- internal/org/prompts/leader.md` の hunk は 2 つで、どちらも `:196-205` の範囲 |
| 仕様 FR-9 の 5 項目 | Met(変更なし) | 節を作る側は変わらない。FR-9 の 2026-11-16 追記は `不明` を述べ(3 回目の sync-docs)、本文と合う |
| 仕様 FR-10 の 4 文 | Met | 4 文に対応する 5 項目(座席の確認、計画、gate、受信箱、座席)は変わらない。`04d3d986` は座席の確認が見つけた返事をどう扱うかだけを変え、「立て直された座席の今の TASK を送り直す」の入力待ち・返事なしの枝(`:205-215`)は同じ。返事を必ず扱う形は、FR-10 の「片付いていない受信箱の件は答えを待つ」とも、受信箱の項目の「どの件も `ralph org escalate` で上げ直しません」(`:291-294`)とも矛盾しない |

固定の文の確認(テストの実行ではない): `assertPhraseOrder` と `containsPhrase` は、空白を全部捨てた部分文字列の順の確かめなので、同じ確かめを `python3 -I` で `leader.md` の座席の確認の項に当てた(`{{ORG_ID}}`、`{{TEAM}}`、`{{AGMSG_SCRIPTS}}` は描画値に置換)。`..._ReseatRuleSeatCheckItem` の 26 句、`..._NeverDiscardsAReply` の 6 句、`..._UnknownTaskID` の 6 句が順に出て、否定の 3 句(`直しの commit があればその直しは頼まない`、`返事よりあとの時刻の行が 1 つでもあれば`、`その返事は扱い直さず`)は出ない。

### 規則が読む値と節が出す値(`04d3d986`・`e6f98624` の分)

| 規則が読む値 | 規則の位置 | 節が出す値 | 結果 |
| --- | --- | --- | --- |
| 返事の時刻(`history.sh` の行の `[` と `]` のあいだ、UTC の `YYYY-MM-DDTHH:MM:SSZ`)と、直しを頼む座席の行の時刻 | `leader.md:202-205` | `- <座席>: <TASK_ID> (<時刻>)`。時刻は台帳の `ts`(UTC の RFC3339、`reseat_progress.go:343`) | 合う。`reseatTimeShape` が行の終わりの形を固定する(`prompts_leader_reseat_test.go:363-365`、`:398-408`)。その doc の「項目が `history.sh` の行の時刻と同じ書き方と言う」は、`e6f98624` が出どころの句を項目に戻したので真(cycle 3 self-review の C3-2 の「コメントが偽」は閉じた)。`不明` の行は時刻が `(記録が切れている、<時刻>)` の中にあり、規則はその読み方を書かなくなった(V26 (c)) |
| 「計画がもう `docs/plans/active/` に戻っていれば」 | `leader.md:201` | 手順 10 の末尾(`:645-647`)と手順 12 の `GATE: fail`(`:722-724`)が、archive のコミットを作っていれば `git revert --no-edit` で `docs/plans/active/` に戻す | 合う。archive のコミットを作らない org は計画が最初から `active/` にあり、戻す手もない |
| 応じる手のうち escalate | `leader.md:199-201` | 受信箱の項目(`:291-294`)「どの件も `ralph org escalate` で上げ直しません」 | 矛盾なし(上げ済みの件は「済んでいる手」) |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **PASS**(rc=0、`==> All verifiers passed.`) | gofmt ok、`golangci-lint` は `0 issues.`、`go vet` は出力なし。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 つの settings、Codex hook の 3 つの検査、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の計画の参照 |
| branch の secret scan | PASS | `48a8a661..9235c95c` clean |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイルの U+FFFD | 0 件 | `git diff main...HEAD --name-only --diff-filter=AM` |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし(rc=0) | tree `173c89859cb3`。`main` は `48a8a661` のまま(fetch していない手元の ref) |
| skill の写しの `cmp` | 同じ(6 組。`/org` と `/director` の `.agents`、`templates/base/.claude`、`templates/base/.agents` を `.claude` と比べた) | 2 つの直しは skill を触っていない |
| `ralph org reseat --help` | 変更なし(build していない) | Go の本体が `c49a6bc5` から変わっていないので、run 3 の確認(8 項目の列挙)のまま |

### Documentation drift(`/sync-docs` に渡す一覧)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-11-12-org-reseat.md:19`(Status の 2026-11-16 の 3 回目の段落)の (2) | No(V22) | 「見つけた返事の時刻より後の時刻の行が節に 1 つでもあれば、…返事を扱い直さず、その座席を空いたものとして次へ進む」は、`04d3d986` が外した枝。同じ段落の (1)(PR の head)と (3)(`不明`)、FR-9 の追記、FR-10 は合う。Status は run ごとに 1 段落を足す作り(`:13`〜`:19`)なので、4 回目の段落を足す(見つけた返事は必ず扱う、済んでいる手だけ飛ばす、例は計画の置き場所と直しの TASK の時刻、直しの commit の例は外した、FR-9 と FR-10 と印は変えていない)。(2) に「4 回目で変えた」の 1 句を足すかは `/sync-docs` が決める |
| `docs/tech-debt/README.md` 行 520 の (e) | No(V23) | 下に分けて書く |
| `docs/tech-debt/README.md`: 3 回目の cross-review の finding 2(WORTH_CONSIDERING) | No(V24) | register に行が無い。triage は tech-debt と PR の Known gaps に回すと決めた |
| `docs/tech-debt/README.md` 行 365・426・514・524 の数、行 522 の Trigger の注 | No(V25) | 下に分けて書く |
| `README.md:303`、`AGENTS.md:92`、`/org` と `/director` skill の各 4 つの写し、`ralph org reseat --help` とその pin、`docs/specs/2026-08-01-org-runtime.md:125` | Yes | 節の 8 項目の列挙だけで、返事の扱いを述べない(AC8 の行) |
| `leader.md` 自身と Go のコメント | Yes | `leader.md:1440` の「あなたが送った TASK も 50 件に入るので」(行 522)は前のまま。テストの doc(`reseatTimeShape`、`..._NeverDiscardsAReply`)は今の項目に合う |
| 計画 `docs/plans/active/2026-11-15-reseat-leader-progress.md` の Progress checklist | 不一致(INFO、V27) | 先の行には 3 回目の cross-review、`04d3d986`、cycle 3 の self-review(`eaa9bc71`)、`e6f98624` が入っている。「Review artifact created」の行は run 3(`9ea00d76`)で止まり、cycle 3 の self-review が無い。この verify、cycle 3 の test と sync-docs、cross-review の項は後から。AC1〜AC9 の checkbox は `[ ]` のまま(`/pr` の前に付ける。drift ではない) |
| 前の run の記録(sync-docs report の Run 3 の節の `:76`・`:92`、test report の `:15`、cross-review triage の cycle 2 の節、self-review の run 3 の節) | 履歴 | 当時の規則を書く。次の run が書き直すか、履歴の節として残る |

### tech-debt の Trigger と数(`/sync-docs` が直す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を python で読んだ。`leader.md`、`prompts_leader_reseat_test.go`、`seat-check`、`座席の確認` を挙げる Trigger は行 188・197・200・365・377・384・426・427・508・514・519・520・522 の 13 行(行 197 は取り消し線つきの RESOLVED)。

- 行 522(Trigger: 座席の確認の項、`prompts_leader_inbox_test.go`): `04d3d986` と `e6f98624` がもう一度書き換えたので当たる(`d85fc0ef` の注の続き)。払っていない(`leader.md:1440` の文と `prompts_leader_inbox_test.go:107` は前のまま)ので、「hit again, did not pay」の 1 句。cycle 3 で `leader.md` を直すと cap を上げて `/self-review` から回し直すので、払わない理由は run 3 と同じ(`leader.md` と 2 つのテストの直し)
- 行 520 の (e): V23 のとおり。(a)(b) の RESOLVED と (d) は変わらない
- 行 365(`prompts_leader_reseat_test.go` 615 は 624 行)、行 426(`leader.md` 1,547 / +156 は 1,546 / +155。同じ行のもう 1 か所の `prompts_leader_reseat_test.go` 615 は 624。「`wc -l` at `d85fc0ef`, the last commit to change it」は `e6f98624`)、行 514(`prompts_leader_reseat_test.go` 615 は 624)、行 524(同じファイルの 615 は 624): 数だけ古い。`wc -l` は HEAD。どれも 800 行の目安の内で、行 524 の `reseat_progress_test.go` 969 と `reseat_progress.go` 582 は変わらない
- 当たらない行: 519(`reseat_progress.go` も gate の行を読む規則も触っていない。gate の項目は 1 行も変わらない)、521(`startProgressSection` を触っていない)、523・525(`reseatProgressSentTask` と最後の TASK の項目を触っていない)、524 の Trigger(`reseat_progress_test.go` を触っていない)、377・384・508・200・188・427(`leader.md` の hunk は座席の確認の項の中だけで、手順 4・14 の本文、split の例、`### cross-review` の小節、feature-org の見出しに当たらない。行 200 の (g) は済みの印つき)
- register に足す新しい行: V24 の 1 行(3 回目の cross-review の finding 2)

### Observational checks(規則に書かれていない縁)

- 座席の確認が、返事を必ず届いたものとして扱うので、`BLOCKED` が別の座席への TASK で消える経路は無い。外れたときの向きは余分な手(TASK や escalate の重複)で、cycle 2 の finding 2 の「繰り返し」は、手ごとの確かめで受けた(self-review cycle 3 の見立てと同じ)
- 手ごとの確かめの材料が書かれているのは 2 つの手(archive の戻し、直しの TASK)だけ。ほかの手(implementer の RESULT のあとの reviewer への TASK、`GATE: pass` のあとの手順 9)の確かめは leader の判断に任される。self-review cycle 3 が指摘にしなかった点で、V26 (d) に書く
- 例の (b) の材料は時刻だけで、返事と直しの TASK の結びつきを時刻の前後に頼る。V26 (a)(b)

### Coverage gaps

- テストの実行結果。この run は本体とテストを読み、固定の文を python で当てただけで、通ることは確かめていない。`prompts_leader_reseat_test.go` は最後の test report(`601b4643`)のあとで変わった(`/test` の打ち直し)
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 の (d))。規則の新しい分岐(返事を必ず扱う、手ごとの確かめ)は読んだだけ
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))
- `history.sh` の `ts` の書き手(メッセージの行)は読んでいない(run 3 と同じ)

### Findings(cycle 3)

ID は run 3 の V21 から続ける。

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V22 | LOW | doc | `docs/specs/2026-11-12-org-reseat.md:19`(Status の 3 回目の段落)の (2) は、`04d3d986` が外した枝(返事より後の時刻の行が 1 つでもあれば返事を扱い直さない)を、今の規則として読める形で書いている。run ごとに 1 段落を足す作りだが、4 回目の段落がまだ無い | `/sync-docs`(4 回目の段落 1 つ。(2) への「4 回目で変えた」の 1 句は任意) |
| V23 | LOW | tech-debt | 行 520 の (e) の (1)(同じ秒の返事と TASK)と (2)(「どの座席でも後の TASK が 1 つあれば扱った」の限界)は、`04d3d986` が外した枝の縁。同じ行の影響の欄の (e)(「扱っていない失敗が落ちる、扱った失敗が 2 回扱われる」の前半 2 つ)、「Why deferred」の欄の (e)(V21 が `d85fc0ef` のあとに見つけた、(1) と (2) を決めるのは実機)、Trigger の欄の (e)(送りの順を使うか)も、その枝を前提にする。新しい規則の縁は V26 | `/sync-docs`(行 520 の (e) を、(3)(GitHub の `headRefOid` の遅れ)と V26 の縁に書き直す) |
| V24 | LOW | tech-debt | 3 回目の cross-review の WORTH_CONSIDERING(finding 2)の行が register に無い。座席の確認が implementer のスライスを送り直す道(`leader.md:190-191` の `不明` の行、`:206-209` の同じ TASK_ID か次のスライス、`:214-215` の立っていない役割)は、計画と「branch の commit」の行から TASK を作り、`RESUME:` の行を持たない。leader が自分で BLOCKED のあとに頼み直すときは、`git status --porcelain` から `RESUME:` を並べる(`leader.md:55-72`)。reseat が implementer をコミットの前に止めると、未コミットの変更が残る(org の pane は WIP checkpoint を止めてある)ので、新しい implementer はそれを前からあった変更と読み、`RESUME:` が無いと止まって BLOCKED を返す(`leader.md:62-64`、`implementer.md:19-27`)。知らせる側に倒れる縁で、何も消えない | `/sync-docs`(新しい行 1 つ。Trigger は座席の確認の送り直しの道を次に触るとき、または、reseat された implementer が自分の変更で BLOCKED を返した報告。直し方は、送り直す前に `git status --porcelain` の scope の中のパスを `RESUME:` に並べる。PR の Known gaps にも) |
| V25 | LOW | tech-debt | 行 365・426・514・524 の `prompts_leader_reseat_test.go` の 615 行は 624 行(`wc -l` は HEAD)、行 426 の `leader.md` 1,547(+156)は 1,546(+155)で、`d85fc0ef` を「最後に変えた commit」とする句は `e6f98624`。行 522 の Trigger に `04d3d986` と `e6f98624` が当たる(払っていない) | `/sync-docs`(数 5 か所と、行 522 の注 1 句) |
| V26 | INFO | rule | 規則の新しい分岐に、書かれていない縁がある。(a) 返事の時刻(agmsg の `ts`)と直しを頼む座席の行の時刻(台帳の `ts`)は別の書き手の値で、同じ秒は「あと」に数えないので、直しを頼み直す(余分な TASK、報告側)。(b) 「直しを頼む座席の行」は最後の TASK の 1 行で、その TASK が返事の失敗の直しかを時刻の前後にだけ頼る。前の leader が返事を読む前に、その座席へ直しでない TASK を送っていると、直しを頼んだものと読み、直しを頼まない。返事は届いたものとして扱うので失敗は見えるが、leader は頼んでいない直しの RESULT を待ち、見張りの stall か deadman だけが知らせる。旧 V21 (b) の「どの座席でも」を「直しを頼む座席の行」に狭めた形で、起きるには、返事が届いてから読むまでのあいだに、直しを頼む座席宛の直しでない TASK が出る必要がある。(c) `不明` の行の時刻は `(記録が切れている、<時刻>)` の中で、規則はその位置を書かなくなった(旧規則は「括弧の中の `、` のあと」)。読み損ねると直しを頼み直す側に倒れる。(d) 材料が書かれた手は 2 つで、ほかの手の「済んでいない」は leader の判断に任される。外れると余分な TASK か escalate。(e) 旧 V21 (c)(push のあとに GitHub が `headRefOid` を返すのが遅れると BLOCKED)は変わらない | 実機の通し(行 520 (d))で見る。(b) は通知が消える向きの縁なので、cap をもう一度上げて `leader.md` を直すかは lead の判断(直すなら、例 (b) を「その返事のあとに、その直しとして送った TASK」と TASK_ID か本文で結べる材料に狭める)。直さないなら行 520 の (e) に書く |
| V27 | INFO | plan | 計画の Progress の「Review artifact created」の行が run 3(`9ea00d76`)で止まり、cycle 3 の self-review(`eaa9bc71`)が無い。この verify と、cycle 3 の test・sync-docs・cross-review の項が無い(後から足す)。V16・V17・V21 の Known gaps は、V21 を V26 に差し替える | lead(`/pr` の前) |

V22〜V25 は文書と register の直しで、コードも `leader.md` も触らないので、pipeline の再実行は要らない。V26 (b) を直す場合だけ `leader.md` を変える。

### Verdict(cycle 3)

- Verified: 静的検査(`run-static-verify.sh` rc=0、golangci-lint 0 issues、`git diff --check`、U+FFFD 0 件、`main` との試し merge、skill の写しの一致、branch の secret scan)。AC1・AC3〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応(`04d3d986` と `e6f98624` を含む)。gate の項目が 1 行も変わらないこと。固定の文 38 句が `leader.md` に順に出て、外した枝の 3 句が出ないこと(python による同じ確かめ)。規則が読む値(時刻、計画の置き場所、escalate の扱い)と、節が出す値の対応
- Partially verified: AC2(`StartFeature` を通した commit だけの節の結合、行 521 (a))
- Not verified: AC7(既存のテストが通ること。`/test` の打ち直し)、節を読んだ leader の実機での動き(返事を必ず扱う、手ごとの確かめ、PR の head、`不明` の分岐を含む)、`history.sh` の `ts` の書き手
- Verdict: **partial-pass**。静的検査は通り、仕様の AC は読んだ範囲ですべて満たす。残りは `/sync-docs` の文書と register(V22〜V25)で、fail にする指摘はない。insight event の `verdict` は `pass`

## Run 3(cycle-count 2, cap 3、履歴): HEAD `9ea00d76`

以下は run 3 の全文。cycle 3 の時点の状態は、上の Cycle 3 の節が優先する(V18〜V20 は `0d443ebb` で閉じ、V21 は V26 に差し替わった)。run 3 の見出しの箇条は次のとおり。

- Scope(run 3): HEAD `9ea00d76`、`git diff main...HEAD`(base `48a8a661`、29 ファイル、+3857/-54)。前の run(`46b7993c`、HEAD `f6be8593`)のあとに入ったのは `5aef79a4`(sync-docs の 2 回目)、`51119734`(cross-review の triage)、`d85fc0ef`(その直し。`leader.md`、`reseat_progress.go` とテスト 2 つ)と、記録の commit
- Evidence(run 3): `docs/evidence/verify-2026-11-16-reseat-leader-progress-run3.log`(`run-static-verify.sh` の出力そのものと、`git diff --check`・U+FFFD・試し merge・skill の写しの `cmp`・行数・`--help`・規則と節の行の形の grep)。`run-static-verify.sh` 自身が書いた `docs/evidence/verify-2026-11-15-190351.log` も同じ内容
- Insight event(run 3): `--phase verify --verdict pass --cycle auto`(cycle 2、LOW 2)
- Verdict(run 3): partial-pass。静的検査は通った(rc=0、golangci-lint 0 issues)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は、`d85fc0ef` を含めて読んだ範囲で満たす。AC2 は `StartFeature` を通した commit だけの節の結合が無く一部(行 521 (a) のまま)。AC7 は `/test` の担当で、cycle 2 の test report(`043dbe59`)は `d85fc0ef` より前。残りは `/sync-docs` の tech-debt の数と trigger の注(V18)と、reseat の仕様の Status の記録(V19)で、fail にする指摘はない

`d85fc0ef`(cycle 2 の cross-review の ACTION_REQUIRED 3 件の直し)と、そのあとの文書の変更を読んだ。変わったのは `internal/org/prompts/leader.md`(座席の確認と gate の 2 項目、`@@ -183`、`@@ -245` の 2 か所だけで、手順の本文は変わらない)、`internal/org/reseat_progress.go`(`reseatProgressSentTask`、`reseatProgressTask`、最後の TASK の行)と、そのテスト 2 つ。テストは実行していない(`/test` の担当)。

### Spec compliance(run 2 から変わった点)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 節に承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met | 8 つの部分は変わらない。最後の TASK の行が、TASK_ID を台帳の切れた記録から読めないとき `- <座席>: 不明 (記録が切れている、<時刻>)` になった(`reseat_progress.go:335-338`)。読む側 `reseatProgressSentTask`(`:252-270`)は、本文が切れ目の印 `...(truncated)` で終わり、かつ TASK_ID が空か印を含むときだけ不明にする。`truncateForDetails` は `verbs.go:950-958` で 200 byte で切り `...(truncated)` を足す。テスト: `TestReseatProgress_ACutTaskIDIsShownAsUnknown`(7 形)、`..._TheTruncatedMarkerIsTheOneTruncateForDetailsWrites` |
| AC2 `start --plan` の立て直しも同じ節。経過の無い org に節なし | Met(結合は部分、V7 のまま) | 不明の行も `add` を通るので、TASK の行と同じに節を作る(V17 と同じ扱い)。`StartFeature` を通した commit だけの節のテストは無いまま(行 521 (a)) |
| AC3 今の命の外は載らない | Met | `lastTask` を作る呼び出し(`:176`)は `events[start:]` の中のまま。`d85fc0ef` は呼び出しの戻り値が 3 つになっただけ |
| AC4 commit は新しい順に 20 件と総数。読めないときは 1 行 | Met(変更なし) | `d85fc0ef` の hunk は `reseat_progress.go` の 71・159・223・256・305 行目の近くだけで、commit を読む範囲に無い |
| AC5 値は 1 行にして切り、本文は fence の中 | Met | 不明の行の座席と時刻は `value()` を通り、語 `不明` は定数 `reseatProgressUnknownTaskID`。行は `- ` で始まるので `reseatSectionLabels` の規則(`- ` で始まらない行は全部ラベルで始まる)に当たる。`## ` で始まる行は増えない |
| AC6 `leader.md` に規則があり、役割の prompt のテストが文を固定する(仕様 AC-14) | Met | 座席の確認(`leader.md:162-218`)に、`不明` の行の扱い(`:186-193`)と、返事の時刻の比べ方(`:195-206`)が入った。gate の項目(`:253-291`)は PR の `headRefOid` で 5 つの場合に分かれる。テストは 11 本。新しい 2 本は `..._ReseatRuleSeatCheckSkipsRepliesAlreadyHandled`(`prompts_leader_reseat_test.go:376`)と `..._ReseatRuleSeatCheckUnknownTaskID`(`:409`)。`..._ReseatRuleGateItem`(`:501`)は PR の head の文を固定する。既存の固定は言い換えだけで、検査の対象を外していない(古い読みの `--json number,state,url`、「PR があれば作りません」が消えたことを見る否定は残る) |
| AC7 既存のテストが通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | cycle 2 の test report(`043dbe59`、pass)は `d85fc0ef` より前。`d85fc0ef` は `reseat_progress_test.go`(869 から 969 行)と `prompts_leader_reseat_test.go`(516 から 615 行)を変えた。`featureLeaderTask` と `feature.go` は変えていない。`/test` の打ち直しが要る |
| AC8 仕様の印、AC-12 は `[ ]`、AGENTS.md と skill の説明が節に合う | Met(Status の記録は V19) | 仕様 `docs/specs/2026-11-12-org-reseat.md` の FR-9(`:82`)、FR-10(`:83`)、AC-13、AC-14 は `[x]`、AC-12(`:108`)は `[ ]` で限りの文あり。README:303、AGENTS.md:92、`/org` skill(`:1836-1842`)と `/director` skill(`:1565-1568`)の各 4 つの写し、`ralph org reseat --help` の 8 項目の列挙は、`d85fc0ef` で変わった点(行の形、PR の head の比べ方、扱い済みの返事)を述べておらず、合っている |
| AC9 gate が通ったあと全座席を reseat した場合 | Met(広げてある) | gate の項目は、座席が `(active)` になるのを待ち、手順 13 で止め、HEAD と `git status --porcelain` を確かめ、`gh pr list --head <ブランチ> --state all --json number,state,url,headRefOid` を読む。(A)head が gate の head と同じ `OPEN` か `MERGED` があれば PR は作らず、「上げた RESULT」を見て手順 14 の RESULT か手順 15、(B)A がなく head が違う `OPEN` があれば、手順 14 の `wip:` の確かめと force なしの push のあと、同じ一覧を読み直し、head が合えば A と同じに、合わなければ BLOCKED、(C)A も B もなく `MERGED` があれば BLOCKED、(D)`CLOSED` だけなら BLOCKED、(E)PR がなければ手順 14。5 つは互いに素で、手順 14 には `wip:` の確かめ(`leader.md:743`)と `git push -u origin <ブランチ>`(`:757`)がある |
| 仕様 FR-9 の 5 項目 | Met | 5 項目に加えた 2 行(座席ごとの最後の TASK、上げた RESULT)と QUESTION の `PLAN` は run 2 のまま。TASK_ID が不明な行は FR-9 の 2026-11-16 追記に書かれていない(V19) |
| 仕様 FR-10 の 4 文 | Met | 4 文に対応する 5 項目は変わらない。gate の項目の PR の head の分岐(B と C)は、FR-10 の「gate が今の HEAD で pass なら PR の手順に進む」の中身の細かい点で、文と矛盾しない。C は PR の手順に進まず BLOCKED になる縁で、仕様の文は要約(INFO、V19 で仕様に書き足す候補) |

### 規則が読む値と節が出す値(`d85fc0ef` の分)

`d85fc0ef` が規則に足した「行」「値」を、節を作る側と突き合わせた。

| 規則が読む値 | 規則の位置 | 節が出す値 | 結果 |
| --- | --- | --- | --- |
| 「座席ごとの最後の TASK」の行の TASK_ID が `不明`、行の形 `- <座席>: 不明 (記録が切れている、<時刻>)` | `leader.md:186-188` | `fmt.Sprintf("- %s: %s (記録が切れている、%s)", seat, reseatProgressUnknownTaskID, ts)`(`reseat_progress.go:336`)。定数は `不明` | 合う。`..._ReseatRuleSeatCheckUnknownTaskID` が節の出力から行を作って規則の文と照らし、`..._ACutTaskIDIsShownAsUnknown` が 2 語(`不明` と括弧)を固定する |
| 不明の行の時刻は「括弧の中の `、` のあと」、不明でない行は括弧の中 | `leader.md:198-199` | 不明の行 `(記録が切れている、<時刻>)`、不明でない行 `(<時刻>)`(`:343`) | 合う |
| 返事の時刻(`history.sh` の `[` `]` のあいだ)と、節の行の時刻(台帳の `ts`、UTC の RFC3339)を同じ書き方として比べる | `leader.md:196-199` | `ManifestEvent.TS` は UTC RFC3339(`manifest.go:21`)。`history.sh` は `[$ts]` を出す(`history.sh:125`) | 台帳側は確かめた。agmsg 側の `ts` が `YYYY-MM-DDTHH:MM:SSZ`(sqlite の driver なら `%SZ`)であることは、self-review run 3 が driver を読んだ結果を引く。この install の `join.sh`・`leave.sh`・`key.sh` は同じ形で書くが、メッセージの `ts` の書き手は読んでいない |
| gate の `head`(`headRefOid` と比べる)、`base` | `leader.md:253-291` | `最後の gate: pass(<event>)head <sha> base <sha> ...`。head は `git rev-parse --verify HEAD^{commit}`(`gate.go:675`)の 40 桁で、`headRefOid` と同じ書き方 | 合う。値は `value()` の 400 字の内に収まる |
| 「上げた RESULT」の TASK_ID(org_id)と `HEAD` | `leader.md:266-268` | `- <id> <state> <TASK_ID> HEAD <head>: <1 行目>`(`:371`) | 合う(run 2 から変わらない) |
| 手順 14 の `wip:` の確かめと push | `leader.md:271-274` | 手順 14(`:743`、`:757`) | 合う。規則はコマンドを写さず手順 14 を指す |

`d85fc0ef` の 3 つの直しと cross-review の finding の対応は、1(PR の head)が gate の項目、2(扱い済みの返事)が座席の確認の時刻の比べ、3(切れた TASK_ID)が節の不明の行と座席の確認の `不明` の分岐。どれも、規則が読む値は節が印字し、節が出す語は規則が名指す。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **PASS**(rc=0、`==> All verifiers passed.`) | gofmt ok、`golangci-lint` は `0 issues.`、`go vet` は出力なし。hook の shellcheck と `sh -n`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の計画の参照 |
| branch の secret scan | PASS | `48a8a661..9ea00d76` clean |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイルの U+FFFD | 0 件 | `git diff main...HEAD --name-only --diff-filter=AM` |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし(rc=0) | tree `da6db9ce72fe`。`main` は `48a8a661` のまま(fetch していない手元の ref) |
| skill の写しの `cmp` | 同じ(8 組) | `/org` と `/director` の `.claude`、`.agents`、`templates/base/.claude`、`templates/base/.agents` |
| `ralph org reseat --help`(HEAD を `go build` して実行) | 節の列挙が 8 項目 | 「the latest plan approval, the last gate record, the last TASK sent to each seat, the open and acked inbox items, the RESULT items the leader raised, the branch's commits, the seats this reseat starts after the leader, and the previous reseat」。`d85fc0ef` の 3 点は書かれていない。`ralph org start --help` は節に触れない(grep 0 件) |

### Documentation drift(`/sync-docs` に渡す一覧)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:303`、`AGENTS.md:92`、`/org` skill と `/director` skill の各 4 つの写し、`ralph org reseat --help` とその pin | Yes | 節の 8 項目の列挙は run 2 の sync-docs で揃った。`d85fc0ef` は列挙を変えず、行の形、PR の head の比べ方、扱い済みの返事の扱いを、これらの文書は述べていない(`不明 (記録が切れている` と、leader の規則の `gh pr list --head … headRefOid` は、`leader.md`・Go・tech-debt・report にだけ出る。skill の `headRefOid` は `ralph org cleanup` と `merge-order` の行の別の話) |
| `docs/specs/2026-08-01-org-runtime.md:125`(2026-11-15 改訂 (a)) | Yes | 8 項目の列挙と「5 項目の規則」。行の形や PR の分岐は述べていない |
| `docs/specs/2026-11-12-org-reseat.md` の Status(`:13`、`:15`、`:17`)、FR-9 の追記(`:82`)、FR-10(`:83`) | No(V19) | Status は run ごとの日付つき段落で、`:17` は 2 回目のパイプライン(2026-11-16)で終わる。3 回目の run(PR の `headRefOid` の比べ、扱い済みの返事の読み飛ばし、`不明` の TASK_ID)の段落が無い。FR-9 の追記は「TASK_ID と時刻」とだけ書き、`不明` を言わない。FR-10 は gate の PR の head の分岐を言わない(これは要約と縁の差で INFO) |
| `internal/org/reseat_progress.go` の doc(`:72-91`、`:236-251`、`:281-304`)、`internal/org/feature.go:225-228` | Yes | `d85fc0ef` が最後の TASK の行の形と `不明` を doc に書き、出力(`:336`、`:343`)と合う。run 2 の V12 は閉じている |
| `internal/org/prompts/leader.md:1441` の「あなたが送った TASK も 50 件に入るので」と `prompts_leader_inbox_test.go:107` | No(V13、行 522 のまま) | 直していない。`d85fc0ef` の座席の確認は、台帳の記録に残る TASK と agmsg の返事を分ける書き方で、この文と逆のまま。行 522 の Trigger に、この直しが当たる(下) |
| `docs/tech-debt/README.md` | No(V18) | 下に分けて書く |
| 計画 `docs/plans/active/2026-11-15-reseat-leader-progress.md` の Progress checklist | 不一致(INFO、V20) | cycle 2 の cross-review(`51119734`、ACTION_REQUIRED 3 件)、cap の 2 から 3 への引き上げ、`d85fc0ef`、self-review の run 3(`9ea00d76`)の項が無い。AC1〜AC9 の checkbox は全部 `[ ]` のまま(`/pr` の前に付ける。drift ではない)。D6 と AC2 の括弧書き(V17)も変わらない |

### tech-debt の Trigger と数(`/sync-docs` が直す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を python で読んだ。self-review run 3 が挙げた 519・520・523・524 に、522 が加わる。

- 行 519(Trigger: `reseat_progress.go` か gate の行を読む規則を触る): `d85fc0ef` が当たる(同じ PR の行なので、持ち越しではなく数と状態の直し)。(a) の「103 lines」は `reseatProgressSection` が 107 行(`:305-411`)。(d) は gate の行だけが残る(変わらない)
- 行 520: (a)(b) の取り消し線つきの RESOLVED は `ec10ba6f` の読み(`--json number,state,url`、「OPEN か MERGED の PR があれば PR は作らない」)をそのまま書く。今の規則は `headRefOid` を gate の head と比べ、OPEN で head が違えば push、MERGED で head が違えば BLOCKED、になった(`d85fc0ef`)。「`d85fc0ef` で PR の head の比べに変わった」の 1 句を足す。(d)(実機の確認)は残る。新しい未確認の縁は V21 の 3 点で、(e) として書き足せる
- 行 522(Trigger: 「`leader.md` の座席の確認の項に触るとき」を含む): `d85fc0ef` が座席の確認を書き換えたので当たる。支払い(`leader.md` の 1 文と 2 つのテストの直し)は `/self-review` からやり直しになるので、この run では払わず、「hit this trigger, did not pay」の注にする。偽の文は今 `leader.md:1441`
- 行 523(Trigger: `reseatProgressSentTask`): `d85fc0ef` が戻り値を 2 つから 3 つにし、doc を書き換えたので当たる。支払い(`Send` の語を定数にする)はしていない。持ち越しの注を足す
- 行 524(Trigger: このファイルを次に触るとき): `reseat_progress_test.go` は 969 行(`wc -l` は `d85fc0ef`、最後に変えた commit)、`progressSentEvent`(`:609`)から終わりまでは 361 行(「約 260 行」は `ec10ba6f` の分で、`d85fc0ef` が約 100 行足した)。分割後の元のファイルは 608 行で「約 610 行」は変わらない。同じ行の `reseat_progress.go` は 582、`prompts_leader_reseat_test.go` は 615 行。「cap of 2 runs」の記述は cap 3 の run 3 に直す。`d85fc0ef` が当たるので支払いなしの注を足す
- 行 365(`prompts_leader_reseat_test.go` 516 は 615 行)、行 426(`leader.md` 1,516 / +125 は 1,547 / +156)、行 514(`reseat_progress.go` 550 は 582、`reseat_progress_test.go` 869 は 969、`prompts_leader_reseat_test.go` 516 は 615。`reseat_progress_start_test.go` 411 と、`reseat.go` 1,553、`feature.go` 874、`org_reseat_test.go` 927、`org_reseat.go` 527 は変わらない): 数だけ古い。`wc -l` は `d85fc0ef` と HEAD で同じ
- 当たらない行: 521(`startProgressSection` と `lastDisbandedIndexes` を触っていない)、377・384・508(手順の本文を触っていない。`d85fc0ef` の `leader.md` の hunk は 2 つで、どちらも「機能ごとの org」の導入の規則の中)、465(RESOLVED の行)
- register に足す新しい行: self-review run 3 の R3-2(`reseatProgressUnknownTaskID` と `TaskIDUnknown` の名前、定数の doc の「the marker」、test の名前)。この branch の register にはまだ無い(`TaskIDUnknown` と `reseatProgressUnknownTaskID` の grep は 0 件。`R3-2` は行 388 と 460 に当たるが、別の branch の R3-2)

### Observational checks

- 座席の確認の時刻の比べは、返事の時刻と節のどの座席の行の時刻とも比べ、あとの行が 1 つでもあれば扱い済み(`leader.md:199-206`)。返事を受けた座席自身の最後の TASK はその返事より前に送られているので、自分の行だけで「扱い済み」にはならない。時刻が同じ秒の行は「あとの行」に数えない(V21 (a))
- `不明` の分岐は implementer と reviewer の両方に出口がある(`leader.md:190-193`)。implementer の「まだ済んでいない最初のスライス」は、全スライスが済んでいるときの行き先を書かない。不明でない行の出口も同じで、`d85fc0ef` が持ち込んだ欠陥ではない
- gate の項目の B は、push が non-fast-forward で失敗したときは手順 14 のとおり BLOCKED になる。誤って完了を報告する向きの穴は見つからなかった。push のあとの再読で GitHub 側の更新が遅れると BLOCKED になりうる(人に渡る側に倒れる)

## Coverage gaps

- テストの実行結果。この run は本体とテストを読んだだけで、通ることは確かめていない。`reseat_progress_test.go`、`prompts_leader_reseat_test.go` は cycle 2 の test report のあとで変わった(`/test` の打ち直し)
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 (d))。規則の新しい分岐(PR の head、扱い済みの返事、`不明`)は読んだだけ
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))
- `history.sh` の `ts` の書き手(メッセージの行)は、この run では読んでいない

## Findings

ID は run 2 の V17 から続ける。

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V18 | LOW | tech-debt | `d85fc0ef` で register の数と状態が古くなり、Trigger が 5 行に当たった。行 519 (a) の 103 行は 107、行 520 (a)(b) の RESOLVED は `ec10ba6f` の読みのまま、行 524 は 869/551/516 行と「約 260 行」と「cap of 2」、行 365・426・514 の数(上の「tech-debt の Trigger と数」)。Trigger は 519・520・522・523・524 に当たる。self-review の R3-1 と同じ。R3-2 の行が無い。self-review run 3 は 522 を挙げていない(座席の確認の項を触ったので、Trigger の文字どおり当たる) | `/sync-docs` |
| V19 | LOW | doc | `docs/specs/2026-11-12-org-reseat.md` の Status に 3 回目の run の段落が無い(PR の `headRefOid` の比べ、扱い済みの返事の読み飛ばし、切れた TASK_ID の `不明`)。FR-9 の追記は「TASK_ID と時刻」とだけ書き、`不明` を言わない。run 2 までは run ごとに 1 段落(`:13`、`:15`、`:17`)を足してきた | `/sync-docs`(Status の段落 1 つと FR-9 の追記に 1 句。FR-10 に gate の PR の head の分岐を書き足すかは任意) |
| V20 | INFO | plan | 計画の Progress checklist に cycle 2 の cross-review(`51119734`)、cap の引き上げ、`d85fc0ef`、self-review の run 3(`9ea00d76`)、この verify の項が無い。D6 と AC2 の括弧書き(V17)も変わらない | lead(`/pr` の前に、Progress の追記と、V16・V17・V21 の Known gaps) |
| V21 | INFO | rule | 規則に書かれていない縁が 3 つ増えた。(a) 返事の時刻と同じ秒の行は「あとの行」に数えず、扱い済みの返事を新しく届いたものと読む。(b) 「あとの行が 1 つでもあれば扱い済み」は目安で、前の leader が別の座席への TASK を、その返事を読む前に送っていた場合は区別できない。そのとき失敗の返事の扱いが落ちうる。(c) gate の項目の B は、push のあとに GitHub が `headRefOid` を返すのが遅れると BLOCKED になる(人に渡る向き)。V16 の 3 つは変わらない | 実機の通し(行 520 (d))で見る。今は直さず、行 520 の (e) に書く |

run 2 の V11〜V16 は、V13(行 522)と V16 を除いて `5aef79a4` までに閉じた。V12(コメント)、V11(列挙 14 か所)、V14(数)、V15(大きさの行)は直っている(上の Documentation drift と tech-debt の節)。V7・V8(cycle 1)は行 521 と行 520 に記録済みで変わらない。

### Verdict(run 3)

- Verified: 静的検査(`run-static-verify.sh` rc=0、golangci-lint 0 issues、`git diff --check`、U+FFFD 0 件、`main` との試し merge、skill の写し 8 組の一致、branch の secret scan)。AC1・AC3〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応(`d85fc0ef` を含む)。規則が読む値のすべてを節が印字し、節が出す語(`不明` と行の形)を規則が名指すこと。節の列挙 8 項目が、README、AGENTS.md、skill の 8 つの写し、`--help`、org-runtime 仕様で揃うこと
- Partially verified: AC2(`StartFeature` を通した commit だけの節の結合、行 521 (a))
- Not verified: AC7(既存のテストが通ること。`/test` の打ち直し)、節を読んだ leader の実機での動き(PR の head、扱い済みの返事、`不明` の分岐を含む)、`history.sh` の `ts` の書き手
- Verdict: **partial-pass**。静的検査は通り、仕様の AC は読んだ範囲ですべて満たす。残りは `/sync-docs` の tech-debt(V18)と仕様の Status(V19)で、fail にする指摘はない。insight event の `verdict` は `pass`

## Run 2(cycle 2 の最初の run、HEAD `f6be8593`、履歴)

以下は run 2 の全文。run 3 の時点の状態は、上の Run 3 の節が優先する(V11〜V17 の多くは `5aef79a4` までに閉じた)。

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

### Coverage gaps(run 2)

- テストの実行結果。この report は本体とテストを読んだだけで、通ることは確かめていない。`reseat_progress_test.go`、`prompts_leader_reseat_test.go`、`org_reseat_test.go` は cycle 1 の test report のあとで変わった(`/test` の打ち直し)
- 経過の節を受け取った leader が規則のとおり動くかの実機の通し(AC-12 の限りと同じ。tech-debt 行 520 (d))
- `StartFeature` を通した disband→start→commit だけの節(行 521 (a))
- `golangci-lint` が CI で動くか。`.github/workflows/verify.yml` に導入の step があるかは、cycle 1 で未確認とした点のまま(今回は手元で `0 issues`)

### Findings(run 2)

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

### Verdict(run 2)

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
