# Verify report: director-split-questions

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-director-split-questions.md`(issue #271、AC1〜AC8、D1〜D6、Progress の S1〜S7 と、D3・AC4・AC7 の本文が印の行の読み取りの確かめを書いていないことの記録)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff 4fef6b8c...HEAD`(base は origin/main 4fef6b8c、HEAD 425e4879、16 ファイル、+2679/−103)。spec の準拠、静的解析、文書のずれを見た。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は読み取りだけにした: skill の正規表現を python3 -I で当てたこと、skill の `gh issue list` のコマンド 2 つと `gh api user` を打ったこと(どちらも GitHub への読み取り)。main のチェックアウトには触れていない。
- Evidence: 全出力は `docs/evidence/verify-2026-11-01-director-split-questions.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-31-220523.log`(同じく対象外)。probe の出力はこの report の Observational checks に写した。
- load average: static の後で 9.67(`uptime`)。static は 56 秒で終わり、timeout に当たっていない。

## Spec compliance

行番号は `.claude/skills/director/SKILL.md` のもの。4 つの写しは `cmp` で同じバイト列だった(sha256 `0f21e13d…`)。`/org` skill の写し 4 つも同じだった。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 分割計画の前の質問の段(`/spec` の手順 1〜3、AskUserQuestion で 1 回 4 問まで、推奨の案を先頭、Codex は番号の選択肢、聞く前に調べる、手順 4〜5 は分割計画の執筆と図解の承認が受け持つ、手順 6〜8 は取り込まない、機能をまたぐ分かれ目をこの段で決め、内側は leader の `FORK`) | 満たす | `### 分割計画の前の質問`(`:501-531`)。1 は聞かずに依頼・AGENTS.md・`.claude/rules/`・コードを読む。2 は上流から順に、4 問まで(`/spec` の 5 問との違いの理由つき)、選択肢 2〜4、推奨を先頭にして「(推奨)」、コード・文書・Web で分かることは聞かない。3 は残った曖昧さだけ。`:521-525` に機能をまたぐ分かれ目の 4 つ(境目、`- Reserve:`、driver と model、順)と内側の `FORK`。`:527-530` に手順 4〜5 の受け持ちと 6〜8 の不採用。`/spec` の手順の名前は `.claude/skills/spec/SKILL.md:31-83` と合う。テストは `TestDirectorSkill_AsksTheSpecQuestionsBeforeWritingASplitPlan`(存在を確かめた。実行は /test) |
| AC2 分割計画の承認を Approve / Needs changes で聞き、Needs changes なら直して図解を作り直して聞き直し、Approve のあとにだけ digest の 2 行を書く | 満たす | `### 分割計画の承認`(`:532-552`)。2 が問いと 2 つの選択肢(Codex と push の director は番号の選択肢)、3(`:545`)が直しと図の作り直しと聞き直し、4(`:547`)が「Approve と答えたときだけ」2 行を書く。テストは `TestDirectorSkill_AsksApproveOrNeedsChangesBeforeTheSplitPlanDigest` |
| AC3 機能ごとの出力先を推奨つきで聞き、`reserved:` と重なる機能は「issue に起票」を推奨、start が重なりで断ったら同じ推奨で聞き直す | 満たす | `### 機能ごとの出力先`(`:553-586`)。1 で `ralph status` の `reserved:` の行と `- Reserve:` を `/org` skill の予約の規則で比べる。2 で 1 機能 1 問、重なれば「issue に起票」を推奨して先頭に置き、org_id とパスを説明に書く。`- Depends on:` で起票した機能に頼る機能も起票を推奨する(S3 の足し)。`:582` が start の断りのあとの聞き直し。`reserved:` の行は `internal/cli/status_followup.go:107` が出す形と合う。テストは `TestDirectorSkill_RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg` |
| AC4 起票の手順と、新しい director の引き継ぎでの記録し直し | 満たす | `### 起票と記録`(`:587-662`)。1 は重なる org ごとに `feature:` の行のブランチを読み、2 つ以上なら全部、`feature:` の行がない org と重ならない機能の起票は待つブランチなし(人に言う)。2 は `.harness/state/` の下のファイルに本文と行頭の印の行(`after: none` を含む)。3 は `gh issue create --title '<題>' --body-file <ファイル>`(`:615`)。4 は `followup list` で重ねない、merge 済みは記録しない、note は印の行、値は単一引用符(`:617-630`)。5 は途中で止まったときの記録し直し。引き継ぎは `:1276-1290`(`gh issue list --state open … --jq` で印の行の issue を探し、`followup list --all` にない待つブランチを記録する)。`feature:` の行の形は `internal/cli/org.go:1908` と合い、`.harness/state/` は root と `templates/base/` の `.gitignore:50` で無視される。計画の本文にない読み取りの確かめ(正規表現 `:641`、ちょうど 1 行 `:649`、書いた人が `gh api user` と同じ `:651`、note は組み直す)は S5 で足したもので、Progress に記録がある。テストは `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`、`TestDirectorSkill_UsesOnlyAMarkLineOfTheShapeFromAnIssueTheDirectorFiled`、`TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` |
| AC5 merge のあと、着手してよいかを決まった文と 3 つの選択肢で聞く。印の行の `after:` のどのブランチにも `ready` の follow-up があるか記録の前に merge 済みのときだけ聞き、欠けは先に記録する。着手なら機能 1 つの分割計画を書いて承認を取り start、それ以外は done。merge と進捗の報告のたびに確かめる | 満たす | 「merge を待つ issue」の 2(`:1300-1335`)。問いの文(`:1329`)は AC5 と issue #271 の文と同じで、選択肢は「着手する」「後に回す」「要らない」。待ちの判定、欠けの先の記録、印の行のない issue の扱い、推奨(重なれば「後に回す」)、人が会話にいないときは待つ。確かめる時機は 2 の冒頭の「merge を人に報告するとき(「進捗の報告」と「merge と auto_merge」)」。3(`:1336-1362`)はすべての open の follow-up を done にし、着手なら分割計画を書き直して(`:1341`)承認と start、start の断りで人が待つと答えたら `gh issue edit`(`:1353`)で `after:` を書き換えて記録し直す。テストは `TestDirectorSkill_AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`、`TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting`、更新された `TestDirectorSkill_RecordsAFollowupReadsItAtMergeReportsAndMarksItDone` |
| AC6 herdr の pane の claude で AskUserQuestion が出るか、pane から答えられるか、push の 1 行と Enter が開いた問いに何をするかの証跡があり、skill がその結果に合わせて聞き方を書く。codex は番号の選択肢 | 満たす | 証跡 `docs/evidence/director-ask-herdr-2026-11-01.md`(`## 結果` は `:42`)。T1・T2 で出て pane から答えられ、T3〜T5 で 1 行と Enter が開いた問いを閉じた(カーソルの下の選択肢を選ぶ、自由記述の欄なら 1 行が答えになる、選択肢の上では 1 行が消える)。D6 の 2 つ目の枝。skill は `### 人への聞き方`(`:476-500`)で、wake に push がある director は番号の選択肢、pull だけなら AskUserQuestion(headless の claude も)、Codex は wake によらず番号の選択肢と書く。計画の Scope にある既存の「計画の承認」の `FORK` の問いもそろえた(`:960`、`:969`、`:977`)。`wake:` の行は `internal/cli/org_director_report.go:314` の `strings.Join(reg.Wake, ",")` と合う。テストは `TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes`、`TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes` |
| AC7 「予約外の文書の follow-up」の 1 が、印の行のある issue の機能を follow-up の有無によらず数えない | 満たす | `:1380-1394`。`gh issue list --state all … --jq` で探し、印の行を確かめてから、分割計画の id と slug で機能を外す。2〜6 でも読まない。テストは `TestDirectorSkill_DocFollowupGatheringDoesNotWaitForAFiledFeature` |
| AC8 新しい固定のテストと既存のテスト(`go test ./internal/org/`)、`check-skill-sync.sh`、`check-sync.sh` が通る | 一部(静的の部分は満たす) | `./scripts/run-static-verify.sh` rc 0。その中の check-skill-sync(14 本)と check-sync(DRIFTED 0)が通った。gofmt、go vet、golangci-lint、staticcheck も通り、テストのファイルは vet で型の検査を受けた。`go test` は /test の範囲で、ここでは打っていない |

issue #271 の受け入れ条件 6 つ:

| issue の受け入れ条件 | Status | 根拠 |
| --- | --- | --- |
| 1. 分割計画の前の質問の手順(推奨つき、AskUserQuestion) | 満たす | AC1 |
| 2. 分割計画の承認と、機能ごとの出力先の選択を AskUserQuestion で聞く | 満たす | AC2、AC3。push の director と Codex は番号の選択肢(issue の「出せなければ番号の選択肢に落とす」を、出せるが push が干渉するという証跡に合わせて広げた) |
| 3. 走っている org と予約が重なる機能で「issue に起票」が推奨され、follow-up に記録される | 満たす | AC3、AC4。重なる org に `feature:` の行がなければ待つブランチがないので記録せず、人に言う(D2) |
| 4. 先行する PR の merge のあと、着手してよいかを AskUserQuestion で聞く | 満たす | AC5 |
| 5. headless の director で選択肢の聞き方を確かめた結果が記録されている | 満たす | AC6 の証跡 |
| 6. skill の同期の検査が通る | 一部 | check-skill-sync と check-sync は通った。`director_skill_sync_test.go` などの Go のテストは /test |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、56 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の parse、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..425e4879` で clean。打ち直しはしていない |
| `cmp` で `/director` と `/org` の 4 つの写し | 同じ | `.claude`、`.agents`、`templates/base/` の 2 つ |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ |
| `git diff --check 4fef6b8c...HEAD`、差分の U+FFFD | どちらも空 | |
| `go version` | go1.27.3 | go.mod と同じ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `git grep -n "人と話して仕事を機能に分け"` | yes | 当たるのは計画の Verify plan の行(`docs/plans/active/2026-11-01-director-split-questions.md:209`)だけ。skill の旧手順の文は残っていない |
| `docs/specs/2026-10-07-org-multi-org-director.md` の分割計画の下書きの記述(`:63`、`:143`) | yes | 「人と話して分割計画を書き、承認を取り」は新しい手順と食い違わない。dated の段落は書き換えていない |
| 同じ spec の FR-9 の #270 の段落(`:153`) | no(V1) | 下の Findings |
| README の director の段落(`:278`)と `followup` の説明(`:131`) | yes | 「drafts split plans with the human」と「an issue the director and the human put off」は今も正しい。push の 1 行の段落(`:280`)は I3 |
| AGENTS.md と `.claude/rules/ralph/ralph-workflow.md` の director の説明 | yes | 登録、受信箱、進捗の報告の粒度で書かれ、変わる文はない |
| director の役割の雛形 `internal/org/prompts/director.md` | yes | 計画の Non-goals どおり変えていない。「分割計画を書くこと、repo を読むこと、ralph と gh の操作だけ」(`:56`)は、本文のファイルを `.harness/state/` に書いて `gh issue create --body-file` に渡す手順と食い違わない(図解ページを書くのと同じ扱い) |
| `docs/tech-debt/README.md` | 一部(V2、V3) | 足した 1 行(push の 1 行が開いた問いを閉じる)は証跡と skill に合う。Trigger が発火した既存の行は下の Findings |
| 計画の Acceptance criteria の checkbox | 遅れ(I2) | AC1〜AC8 は `[ ]` のまま |

## Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | doc drift(spec) | spec の FR-9 の「2026-10-30 に決めたこと(issue #270)」(`docs/specs/2026-10-07-org-multi-org-director.md:153`)は「着手できる follow-up を見張りが受信箱の件にして人に送ることは #271 に送った」で終わる。#271 は見張りを変えず(Go のコードは Non-goals)、director が merge と進捗の報告のたびに `ralph status` と merge-order を読んで人に聞く形にした。spec には #271 の dated の段落がなく、読む人は見張りが件を送ると取りうる。この系列は #264・#265・#267・#268・#270・#276 で FR-9 などに dated の段落を足している | /sync-docs で FR-9 に「2026-11-01 に決めたこと(issue #271)」の段落を足す(質問の段、承認と出力先の問い、起票と印の行、ブランチごとの follow-up、報告のたびの着手の問い、見張りは件を送らないこと、push の 1 行と開いた問いの証跡)。#270 の段落は書き換えない |
| V2 | LOW | doc drift(tech-debt) | `docs/tech-debt/README.md:412` の Trigger「#271 (the watch sending ready follow-ups to the inbox) or any change that acts on status's `ready` without a human」と、`:415` の (7) の Trigger「#271 (the watch sending ready follow-ups to the inbox), or a report that real gh answers otherwise than the fake」は、この PR を番号で名指す。#271 は見張りに件を送らせず、director は `ready` を読んでも人に聞いてから動くので、どちらの行の仕事(`mergedAt` の記録、本物の gh の確かめ)もまだ要らない。ただし行は今のままでは #271 の中身を取り違えて書き、持ち越しの記録もない | /sync-docs で 2 行に「2026-11-01: feat/director-split-questions (#271) は … を見張りではなく director の問いにしたので払わない」という持ち越しの文を足し、Trigger の「#271 (…)」を「a change that has the watch send ready follow-ups to the inbox」などに直す |
| V3 | LOW | doc drift(tech-debt) | Trigger が発火した 2 行に持ち越しの記録がない。(a) `:259`「The next change to the inbox section (`受信箱の処理`) of the `/director` skill」: この PR は `### 計画の承認` の 1 と 2 を直し、push の項を足した(4 つの写し)。行の (1)〜(3) が見る wait の段落と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は変えていない。(b) `:386`「The next change to any of the four files」: `internal/org/director_skill_sync_test.go` が 1171 行から 1202 行になった(`directorSkillSection` と `directorSkillApprovalForkSteps` の断片) | /sync-docs で 2 行に「carried over by feat/director-split-questions」の文を足す(何を変え、なぜ払わないか。(b) は移動だけの分割が計画の Scope の外であること) |
| I1 | INFO | 行数 | 新しい `internal/org/skill_split_questions_test.go` は 814 行で、code review の 800 行の目安を超える。`director_skill_sync_test.go` は 1202 行(V3 の (b) の行が扱う)。新しいファイルの分は tech-debt にない | /sync-docs か PR 本文で記録するかを決める(任意) |
| I2 | INFO | 計画 | 計画の AC1〜AC8 の checkbox が `[ ]` のまま。また D3・AC4・AC7 の本文は読み取りの確かめを書いていないが、digest を変えないため Progress に記録してあり、意図どおり | /sync-docs か /pr で checkbox を `[x]` にする(checkbox の印は digest の外)。読み取りの確かめのことは PR 本文に書く |
| I3 | INFO | README | README の push の 1 行の段落(`:280`)は、1 行と Enter が開いた AskUserQuestion を閉じることを書いていない。skill と tech-debt の新しい行が扱っており、`wake_cmd` で端末に打つ設定でも同じことが起こりうる | 任意で、README のその段落に 1 文と証跡へのポインタを足す |
| I4 | INFO | skill の正規表現 | 印の行の正規表現は `after: none,feat/a` にも当たる(`none` をブランチの名前として読む)。director は自分で書く行に `none` と並べてブランチを書かないので、手で書き換えた本文でしか起きない。また note は 500 字までで、待つブランチがとても多いと印の行がそれを超えうる(固定の部分は最長 106 字)が、超えたときの手順はない。`ralph org followup add` がエラーで断るので、黙って欠けることはない | なし(記録だけ) |

## Observational checks

- 印の行の正規表現(skill の `:641` の行を取り出して python3 -I の `re` で当てた。Go の RE2 と同じ結果になる書き方だけを使っている): 当たったのは `ralph-split: 2026-11-01-auth/token after: feat/a`、`… after: feat/a,fix/b.c_d`、`… after: none`、slug 20 字、`after: none,feat/a`(I4)。当たらなかったのは行末の CR、`after: -rf`、`after: /etc`、単一引用符を含む値、空白を含む値、`$(id)`、slug 21 字、空の `after:`。shell に入る値の形(先頭の `-` と `/`、引用符、空白、`$`)はどれも外れた。
- skill の引き継ぎのコマンド(`gh issue list --state open --limit 1000 --json number,author,body --jq '… test("(?m)^ralph-split: ") …'`)は rc 0 で何も出さなかった(印の行の issue はまだない)。同じ形で正規表現を `(?m)^## ` に替えた対照(`--state all --limit 5`)は 5 件(#273〜#277、作者 yoshpy-dev)を出したので、gh の jq が `(?m)` の行頭を読み、`.author.login` が取れることを確かめた。`gh api user --jq .login` は `yoshpy-dev`。
- skill が名指す CLI の出力の形をコードで確かめた: `feature: %s/%s branch %s worktree %s`(`internal/cli/org.go:1908`)、`  reserved: %s`(`internal/cli/status_followup.go:107`)、`wake` は登録の値を `,` でつないだもの(`internal/cli/org_director_report.go:314`)、note の上限 `InboxNoteMaxRunes = 500`(`internal/org/inbox.go:76`)、after-branch は先頭の `-` と空白と制御文字を断る(`internal/org/followup.go:239-253`)、分割計画の id は `^[a-z0-9][a-z0-9-]{0,63}$`(`internal/org/split.go:63`)、slug は `identifierPattern` で 20 字まで(`internal/org/split.go:498-504`)。skill の正規表現はこれらと合う。

## Coverage gaps

- `go test ./internal/org/` と変異の再実行は /test の担当で、打っていない。S2〜S7 の implementer が記録した変異(19、29、3、16、7、2 個)は、計画の Progress を読んだだけで確かめていない。
- skill の手順を実機の director で通すこと(分割計画から start、起票、merge のあとの問い)は計画の Non-goals で、確かめていない。
- 証跡の「確かめていないこと」(codex の director、`ralph org director start` と本物の見張りの wake、`wake_cmd`、2〜4 問の AskUserQuestion、Esc、ほかの版と大きさ)は、そのまま残る。
- `gh issue create`、`gh issue edit`、`ralph org followup add` は書き込みなので打っていない。

## Verdict

pass

- Verified: AC1〜AC7 を skill の文と証跡で確かめた。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。skill の `gh` の読み取りのコマンドは書いたとおりに動いた。
- Partially verified: AC8 と issue の受け入れ条件 6 は、静的の部分だけ。
- Not verified: Go のテストの実行(/test)、実機の director での通し(Non-goals)。
- 所見: LOW 3 件(V1〜V3、どれも文書のずれ)、INFO 4 件。どれも merge を止めない。
- /sync-docs に回すもの: V1(spec の FR-9 に #271 の dated の段落)、V2(tech-debt の 412 行と 415 行の (7) の持ち越しと Trigger の文)、V3(tech-debt の 259 行と 386 行の持ち越し)。任意: I1〜I3。
- Insight event: report を書いたあとに `./scripts/insights-append.sh --slug director-split-questions --flow standard --phase verify --verdict pass --cycle auto --source skill --low 3` を打つ。UTC の日付で `docs/insights/events/2026-10-31-director-split-questions.jsonl` に入る(この slug の events のファイルはまだない)。

## Cycle 2

- Date: 2026-11-01
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。`.harness/state/standard-pipeline/cycle-count.json` は 2)
- Scope: cycle 2 の差分は `git diff 101e7552..HEAD`(S9 9248d73b、S10 e95f19e3、S11 80cb8b6b と、計画と self-review の記録。8 ファイル、+366/−94)。AC1〜AC8 と issue #271 の受け入れ条件は、ブランチ全体 `git diff 4fef6b8c...HEAD`(HEAD 5ca9949d、25 ファイル、+3434/−117)で確かめ直した。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は読み取りだけにした: skill の `gh issue list` のコマンド 2 つと対照 1 つ、`gh issue view 271`。main のチェックアウトには触れていない。
- Evidence: `./scripts/run-static-verify.sh` の全出力は `docs/evidence/verify-2026-11-01-director-split-questions-c2.log`、run-verify 自身のログは `docs/evidence/verify-2026-10-31-235732.log`(どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。
- load average: static の前が 5.75、後が 6.14(1 分値)。static は 31.9 秒で終わり、timeout に当たっていない。

### Spec compliance

行番号は HEAD の `.claude/skills/director/SKILL.md` のもの。4 つの写しは `cmp` で同じバイト列だった(sha256 `ebde6129…`)。`/org` skill の 4 つの写しも同じだった。

| Acceptance criterion | Status | cycle 2 で確かめたこと |
| --- | --- | --- |
| AC1 分割計画の前の質問の段 | 満たす(変化なし) | cycle 2 は `### 分割計画の前の質問`(`:501-531`)に触れていない |
| AC2 分割計画の承認 | 満たす(変化なし) | `### 分割計画の承認`(`:532-552`)に触れていない |
| AC3 機能ごとの出力先と、start の断りのあとの聞き直し | 満たす(変化なし) | `### 機能ごとの出力先`(`:553-586`)に触れていない。断りのあとの聞き直し(`:582-583`)は cycle 1 のまま(C2-I2) |
| AC4 起票の手順と、引き継ぎでの記録し直し | 満たす | 「起票と記録」の 4 に、待つブランチがすべて merge 済みで記録せずに聞いたとき、人の答えを「merge を待つ issue」の 3 のとおり台帳に残す 1 文が入った(`:625`)。引き継ぎ(`:1277-1304`)は、`--jq` の 1 行目に絞り込む前の件数を出し(`:1284`)、1 行目が `--limit` に届いたら一覧が切れたとみなして、`--limit` を上げて打ち直すか人に言う(`:1287-1290`、cross-review cycle 1 #3)。`after:` のブランチの PR がすべて `merged` で、その issue の follow-up が `followup list --all` に 1 件もなければ、記録せずに番号を伝えてすぐ着手を聞く。一部だけ merge 済みなら、merge されていないものだけを記録する(`:1296-1303`、#1)。テストは `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` と `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` に断片が足された |
| AC5 merge のあとの着手の問いと、答えの記録 | 満たす | 「merge を待つ issue」の 3(`:1350-1395`)。人が答えたあと、follow-up が 1 件もない印の行の issue には、`after:` のブランチの 1 つで `followup add` を打ち、すぐ `followup done` にする(`:1353-1366`、self-review C2-1)。答える前は記録しない。着手で start が重なりで断られ、人が待つと答えたら、`ralph status` を読み直して重なる org を全部数え、それぞれの `feature:` のブランチを全部 `after:` に書き、全部に記録する(`:1373-1390`、#2)。もう重なる org がなければ書き換えず、start してよいかを人に聞く(`:1376-1378`、S10)ので、「人に聞かないまま issue に着手しない」(`:1395`)とぶつからない。問いの文と 3 つの選択肢(`:1336-1343`)は cycle 1 のまま。テストは `TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` |
| AC6 headless の聞き方の証跡と skill の決まり | 満たす(変化なし) | `### 人への聞き方`(`:476-500`)、証跡、`### 計画の承認` に触れていない |
| AC7 「予約外の文書の follow-up」の 1 が起票した機能を数えない | 満たす | 1(`:1407-1431`)のコマンドを引き継ぎと同じ件数の形にし(`:1425`)、同じ読み方の文を足した(`:1428-1431`)。起票した機能を数えない規則は cycle 1 のまま。テストは `TestDirectorSkill_DocFollowupGatheringDoesNotWaitForAFiledFeature` |
| AC8 テスト、check-skill-sync、check-sync | 一部(静的の部分は満たす) | `run-static-verify.sh` rc 0。その中の check-skill-sync(14 本)と check-sync(DRIFTED 0)が通った。`go test` は /test |

S9〜S11 の文がコードについて言うことを、コードで確かめた。

- 「start のエラーは最初に重なった org だけを出す」(`:1373-1374`): `reservationDecision`(`internal/org/reserve.go:567-579`)は予約を持つほかの org を org_id の順に並べ、最初に重なった org で `return` する。文のとおり。
- 「`followup add` は gh を読まないので、merge 済みのブランチでも記録できる」(`:1356-1357`): `newOrgFollowupAddCmd`(`internal/cli/org_followup.go:84-107`)は `ValidateFollowupAdd` と `FollowupStore.Add` だけを呼び、`internal/org/followup.go` に `exec.` の呼び出しはない。ブランチの検査は、空でない、空白と制御文字がない、先頭が `-` でない、の 3 つだけ。文のとおり。書き込みになるので実際には打っていない。
- `--jq` の件数の行: 下の Observational checks。

issue #271 の受け入れ条件 6 つ(本文は 2026-10-27 から変わっていない):

| issue の受け入れ条件 | Status | 根拠 |
| --- | --- | --- |
| 1. 分割計画の前の質問の手順 | 満たす | AC1 |
| 2. 分割計画の承認と、機能ごとの出力先の選択を聞く | 満たす | AC2、AC3。push の director と Codex が番号の選択肢で聞くのは cycle 1 のまま |
| 3. 重なる機能で「issue に起票」を推奨し、follow-up に記録する | 満たす | AC3、AC4。cycle 2 で、着手のあとに待つと答えたときも重なる org を全部数えるようになった |
| 4. 先行する PR の merge のあと、着手してよいかを聞く | 満たす | AC5。cycle 2 で、引き継ぎのときにすべて merge 済みの issue も聞くようになり、その答えも台帳に残る(issue の「待つなら記録を残す」) |
| 5. headless の director での聞き方の結果の記録 | 満たす | AC6 の証跡(変化なし) |
| 6. skill の同期の検査が通る | 一部 | check-skill-sync と check-sync は通った。Go のテストは /test |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、31.9 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の parse、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..5ca9949d` で clean。打ち直しはしていない |
| `cmp` で `/director` と `/org` の 4 つの写し | 同じ | `.claude`、`.agents`、`templates/base/` の 2 つ |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ。cycle 2 は Progress だけを変えた |
| `git diff --check 4fef6b8c...HEAD`、差分の U+FFFD | どちらも空 | |
| `skill_split_questions_test.go` の `func Test` の数 | 101e7552 と HEAD でどちらも 14 | cycle 2 はテスト関数を足さず、既存のテストに断片を足した。節を読む関数(`directorSkillIntro`、`directorFollowupStep`、`directorSplitSubsection`、`directorSplitStep`)は変わっていない |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の「merge を待つ issue」の「PR がもう merge されたブランチは follow-up にしない」(`.claude/skills/org/SKILL.md:1652`、ほかの 3 つの写しも `:1652`) | no(C2-V1) | 下の Findings |
| spec の FR-9 の「2026-11-01 に決めたこと(issue #271)」(`docs/specs/2026-10-07-org-multi-org-director.md:154`) | yes(C2-I1) | (5) は「どのブランチにも `ready` の follow-up があれば…聞く」と、聞く条件の 1 つを書く要約で、「そのときだけ」とは書いていない。引き継ぎで merge 済みの issue を聞くこと、答えを add と done で残すこと、重なる org の数え直し、件数の行は書いていないが、どれも要約と食い違わない |
| README の `followup` の説明(`:131`)と director の段落(`:278`) | yes | 「`followup done <id>` marks one done once the human was asked」は、答えのあとに add と done を打つ S11 の手順と合う。cycle 2 で変わる文はない |
| `docs/recipes/director-wake.md` とその写し | yes | S9〜S11 は push の 1 行と開いた問いの扱いに触れていない。2 つの写しは `cmp` で同じ |
| `docs/tech-debt/README.md:417`(`skill_split_questions_test.go` の行) | yes | 886 行と書き、`wc -l` も 886(self-review C2-6 の直し) |
| `docs/tech-debt/README.md:362`(`verbs.go` などの大きいファイルの行)の持ち越しの注 | no(C2-V2) | 下の Findings |
| tech-debt の Trigger | yes | cycle 2 の差分が触った面(「起票と記録」の 4、「merge を待つ issue」、「予約外の文書の follow-up」の 1、`skill_split_questions_test.go`)で Trigger の列を掃いた。`:417` の Trigger(テストを足す、節を読む関数を変える)は上の表のとおり発火していない。`:259`(受信箱の処理)、`:376`(send のあとの確かめ)、`:377`(wait の終了コード)、`:177`(`/org` の command の表)は、cycle 2 が触っていない節を名指す |
| 計画の Acceptance criteria の checkbox | 遅れ(cycle 1 の I2 のまま) | AC1〜AC8 は `[ ]` のまま |

### Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| C2-V1 | LOW | doc drift(`/org` skill) | `/org` skill の「merge を待つ issue」(`.claude/skills/org/SKILL.md:1652`、4 つの写し)は「PR がもう merge されたブランチは follow-up にしない。記録より前の merge は数えないので `ready` にならない。issue はすぐ人に聞く。」と、例外なしに書く。S11 は `/director` skill の「merge を待つ issue」の 3(`:1353-1366`)に、すべて merge 済みの起票した issue に人が答えたら、merge 済みのブランチで `followup add` を打ってすぐ `followup done` にする手順を足した。待つための件ではなく答えの記録で、`/director` の側は理由も書いているが、`/org` skill だけを読む人(leader、人)には、merge 済みのブランチの done の件が決まりに反して見える。`/org` の文は `internal/org/skill_followup_test.go:200` が断片で固定している | /sync-docs で、`/org` skill のその項に、例外の 1 文を足す(例:「director が、すべて merge 済みの起票した issue への人の答えを残すために足し、すぐ done にする件は別(`/director` skill の「merge を待つ issue」の 3)」)。固定の断片は変えずに済む。4 つの写しは `scripts/sync-skills.sh` で作る |
| C2-V2 | LOW | doc drift(tech-debt) | `docs/tech-debt/README.md:362` の 2026-11-01 の持ち越しの注は、`internal/org/skill_split_questions_test.go` を「832 lines; it has a row of its own below」と書く。HEAD では 886 行で、その行(`:417`)は self-review C2-6 の直しで 886 に直ったが、`:362` の数は直っていない。self-review C2-6 と同じ種類のずれ | /sync-docs で `:362` の数を 886 にするか、数を消して `:417` の行を指すだけにする |
| C2-I1 | INFO | spec の要約 | spec の FR-9 の #271 の段落(`:154`)の (5) は、引き継ぎで merge 済みの issue をすぐ聞くこと、答えを add と done で残すこと、重なる org の数え直しを書いていない。(5) は聞く条件の 1 つを書く要約で、ほかの経路を否定しない。食い違いではない | なし(記録だけ) |
| C2-I2 | INFO | 同じ種類の経路 | cross-review cycle 1 #2 の直し(重なる org を全部数え直す)は「merge を待つ issue」の 3 の経路だけに入った。最初の経路(「機能ごとの出力先」の 3 で start が重なりで断り、「issue に起票」で聞き直す)は、「起票と記録」の 1(`:592-601`)が「重なる org ごとに」と書くだけで、断りのあとに `ralph status` を読み直すとは書いていない。1 の比べ方で重なりを見落としたか、比べたあとに新しい org が立ったときに限り、エラーが出す 1 つの org だけを待つ恐れがある。人が推奨に逆らって「今すぐ着手」を選んだときは、1 で数えた org を使うので起きない | 任意で、「機能ごとの出力先」の 3 の聞き直しの前に「1 をやり直す」と足す。直さないなら PR の Known gaps に書く |
| C2-I3 | INFO | CLI の help | `ralph org followup add` の help(`internal/cli/org_followup.go:74-75`)は「a branch whose PR is already merged needs no follow-up: ask the human about the issue now instead」と書く。director は先に人に聞き、答えのあとに記録のための件を足すので、待つための件としては help の言うとおりのまま。Go のコードを変えることは計画の Non-goals | なし(記録だけ) |
| C2-I4 | INFO | insight event | events のファイルに self-review の cycle 2 の行がない(commit 25731fdc は report だけを足した)。この verify の行は、lead の指示どおり `2026-10-31-director-split-questions.jsonl` に足した。打った時刻が UTC で 2026-11-01 になり、script は `2026-11-01-…jsonl` を作るため、`--events-dir` に scratch を渡して作った行をそのまま写した | self-review の cycle 2 の行を足すかを lead が決める |

### Observational checks

- skill のコマンドを、書いてあるとおりに `gh` 2.102.0 で打った(読み取りだけ)。`--state open` は `3` の 1 行、`--state all` は `68` の 1 行を出した(印の行のある issue はまだない)。対照として、`--state all` のコマンドの `--limit 1000` を `--limit 5` に、`(?m)^ralph-split: ` を `(?m)^## ` に替えると、1 行目が `5`、2 行目から `277	yoshpy-dev` から `273	yoshpy-dev` の 5 行を出した。一覧が切れると 1 行目が `--limit` の値と同じになり、番号と書いた人は 2 行目から出ることを確かめた。skill の文(`:1287-1290`、`:1428-1431`)と合う。
- self-review cycle 2 の所見の直しをコミットと突き合わせた。C2-1 は 3(`:1353-1366`)、「起票と記録」の 4(`:625`)、引き継ぎ(`:1300-1301`)。C2-2 はテストの comment の 5 か所が `cross-review cycle 1 #<n>` で、cycle のない `cross-review #` は残っていない。C2-3 は引き継ぎの 2 か所が「「merge を待つ issue」の 2」と節の名前を持つ(`:1299-1300`、`:1303`)。C2-4 は 2 か所が同じ読み方の文を持ち、テストは定数 `directorSplitMarkIssuesOutput` で 2 か所を固定する。C2-6 は上の Documentation drift。
- `gh issue view 271`: OPEN、updatedAt 2026-10-27T07:15:09Z。受け入れ条件 6 つは cycle 1 の表と同じ。

### Coverage gaps

- `go test ./internal/org/` と変異の再実行は /test の担当で、打っていない。S9〜S11 の implementer が記録した変異(13、1、9 個)は、計画の Progress を読んだだけで確かめていない。
- `followup add` を merge 済みのブランチで打ち、すぐ `followup done` を打つ手順は、書き込みなので打っていない。S11 の scratch の台帳での確かめは Progress の記録だけを読んだ。
- issue が 1,000 本を超える repository で `--limit` に届いたときの実際の出力は確かめていない(対照は `--limit 5` で代えた)。
- 実機の director での通し(Non-goals)と、PR の Known gaps に回すもの(add と done のあいだで止まったとき、director の gh のアカウントが変わったとき)は残る。

### Verdict

pass

- Verified: cycle 2 の S9〜S11 で、AC4・AC5・AC7 の文が cross-review cycle 1 の #1〜#3 と self-review cycle 2 の C2-1〜C2-6 を直したこと、AC1〜AC3 と AC6 の節が変わっていないこと、S9 と S11 がコードについて言うこと(start のエラーが最初の org だけを出す、`followup add` は gh を読まない)を確かめた。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。`--jq` の件数の行は書いたとおりに動いた。
- Partially verified: AC8 と issue の受け入れ条件 6 は、静的の部分だけ。
- Not verified: Go のテストの実行(/test)、`followup add` と `done` の書き込み、実機の director での通し(Non-goals)。
- 所見: LOW 2 件(C2-V1、C2-V2、どちらも文書のずれ)、INFO 4 件。どれも merge を止めない。
- /sync-docs に回すもの: C2-V1(`/org` skill の例外の 1 文)、C2-V2(tech-debt の `:362` の行数)。任意: C2-I2。
- Insight event: `./scripts/insights-append.sh --slug director-split-questions --flow standard --phase verify --verdict pass --cycle auto --source skill --low 2 --events-dir <scratch>` で作った行を、`docs/insights/events/2026-10-31-director-split-questions.jsonl` の末尾に写した(C2-I4)。

## Cycle 3

- Date: 2026-11-01
- Verifier: verifier サブエージェント(`/verify`、パイプラインの 3 回目の run。cross-review cycle 2 で上限を 3 に上げた。`.harness/state/standard-pipeline/cycle-count.json` は `cycle` 2、`cap` 3)
- Scope: この run の差分は `git diff 6f62affb..HEAD`(S12 e775792d、S13 8a23af28 と、計画と self-review の記録。8 ファイル、+374/−94)。AC1〜AC8 と issue #271 の受け入れ条件は、ブランチ全体 `git diff 4fef6b8c...HEAD`(HEAD 6f919007、25 ファイル、+3971/−121)で確かめ直した。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は読み取りだけにした(`gh issue view 271`)。main のチェックアウトには触れていない。
- Evidence: `./scripts/run-static-verify.sh` の全出力は `docs/evidence/verify-2026-11-01-director-split-questions-c3.log`、run-verify 自身のログは `docs/evidence/verify-2026-11-01-014338.log`(どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。
- load average: static の前が 5.46、後が 12.06(1 分値)。static は 54 秒で終わり、timeout に当たっていない。

### Spec compliance

行番号は HEAD の `.claude/skills/director/SKILL.md` のもの。`/director` と `/org` の 4 つの写しは、それぞれ `cmp` で同じバイト列だった(sha256 は `/director` が `d838b826…`、`/org` が `5af698fd…`)。この run の差分が触ったのは、「起票と記録」の 2(`:602-610`)、「merge を待つ issue」の引き継ぎ(`:1293-1317`)、2(`:1358-1374`)、3(`:1375-1388`)だけだった。

| Acceptance criterion | Status | この run で確かめたこと |
| --- | --- | --- |
| AC1 分割計画の前の質問の段 | 満たす(変化なし) | `### 分割計画の前の質問`(`:501-531`)に触れていない |
| AC2 分割計画の承認 | 満たす(変化なし) | `### 分割計画の承認`(`:532-552`)に触れていない |
| AC3 機能ごとの出力先と、start の断りのあとの聞き直し | 満たす(変化なし) | `### 機能ごとの出力先`(`:553-586`)に触れていない |
| AC4 起票の手順と、引き継ぎでの記録し直し | 満たす | 「起票と記録」の 2 は、頼られる機能を先に起票したとき、その番号を `頼られる issue: #<番号>` の 1 行で行頭から書く(2 つ以上なら 1 つ 1 行。`:605-607`、self-review C3-1)。`- Depends on:` の行は slug の並びのままで、`/org` skill の定義(`.claude/skills/org/SKILL.md:608`)と合う。引き継ぎは、数える follow-up を `followup list --all` の `AFTER_BRANCH` が今の印の行の `after:` のブランチの件に限り、前の待ちの done の件を数えない(`:1293-1296`、cross-review cycle 2 #2)。今の `after:` のブランチで done と open が混ざる issue は、done の件の `DONE_NOTE` を人の答えとして、残りの open の件を同じ note で done にする(`:1308-1317`、#3)。テストは `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`(行の形を断片と、空白を無視しない `strings.Contains` で固定)と `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` |
| AC5 merge のあとの着手の問いと、答えの記録 | 満たす | 2 の推奨に、本文の頼られる issue の行の issue のどれかが `OPEN` なら「後に回す」を推奨にして先頭に置き、open の頼られる issue を全部説明に書く、が入った(`:1361-1371`、#1)。番号は、行全体が `^頼られる issue: #[0-9]+$` に当たる行(行末の CR は取る)の数字だけを使い、`- Depends on:` の行からは拾わない。3 の「follow-up が 1 件もないとき」は「今の `after:` のブランチの follow-up が 1 件もないとき」になり(`:1379-1380`、C3-3)、引き継ぎの数え方と同じになった。問いの文と 3 つの選択肢(`:1354-1356`)は変わっていない。テストは `TestDirectorSkill_AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`(正規表現を Go の `regexp` で 12 行に当てる)と `TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` |
| AC6 headless の聞き方の証跡と skill の決まり | 満たす(変化なし) | `### 人への聞き方`(`:476-500`)、証跡、`### 計画の承認`(`:950-1011`)に触れていない |
| AC7 「予約外の文書の follow-up」の 1 が起票した機能を数えない | 満たす(変化なし) | `### 予約外の文書の follow-up`(`:1424-`)と「印の行の形と読み方」(`:637-663`)に触れていない |
| AC8 テスト、check-skill-sync、check-sync | 一部(静的の部分は満たす) | `run-static-verify.sh` rc 0。その中の check-skill-sync(14 本)と check-sync(DRIFTED 0)が通った。`go test` は /test |

cross-review cycle 2 の triage と self-review の 3 回目の run の所見は、次のとおり。

| 所見 | 状態 | 根拠 |
| --- | --- | --- |
| cross-review cycle 2 #1(頼る先が未実装でも着手を推奨する) | 直った | 上の AC5。S12 が入れた「数字だけの値」は、S13 で行の形と正規表現になった |
| cross-review cycle 2 #2(書き換えのあとの古い done の件) | 直った | 上の AC4 |
| cross-review cycle 2 #3(done を打つ途中で止まる) | 直った | 上の AC4。答えが「着手する」なら `ralph status` で org を確かめ、走っていなければ人に聞く(`:1314-1316`)ので、「人に聞かないまま issue に着手しない」(`:1422`)とぶつからない |
| self-review C3-1(頼られる issue の番号の形) | 直った | `:605-607` と `:1365-1371` |
| self-review C3-3(3 の文言) | 直った | `:1379-1380` |
| self-review C3-4(台帳から読んだ note) | 直った | 単一引用符で囲み、`'` を含めば写さずに自分の言葉で書く(`:1311-1314`) |
| self-review C3-6(tech-debt の行数) | 直った | `docs/tech-debt/README.md:362` と `:417` が行数を書かず「over 800 lines」と書く。`wc -l` は 995 行で、文は正しい |
| self-review C3-2、C3-5 | 直していない | 計画の Progress(`:300`)が PR の Known gaps に回すと決めている |

S12・S13 の文がコードについて言うことを、コードで確かめた。

- 「`followup list --all` の `AFTER_BRANCH` の列」と「done の件の `DONE_NOTE`」: `printFollowupTable`(`internal/cli/org_followup.go:258-281`)の見出しは `ID STATE ISSUE AFTER_BRANCH ADDED_AT NOTE` で、`--all` のときだけ `DONE_AT DONE_NOTE` が続く。どちらの列名も実在する。
- `DONE_NOTE` をそのまま `followup done --note` に渡すこと: note は 1 行で、制御文字を含めない(`validateFollowupNote`、`internal/org/followup.go:269-286`)。`followup list` のエスケープが掛かるのは制御文字なので、ralph が書いた note なら表示と中身は同じになる。`'` だけが単一引用符を壊し、skill はその場合を書いている。
- 「機能の org が走っているか」の手がかり: `ralph org start --plan` の org_id は既定で機能の slug(`internal/org/feature.go:95`)。`ralph status` は `feature:` の行を出さないので、`--org-id` を変えた org は slug では見つからない。self-review C3-5 の (c) と同じで、Known gaps に回る。
- 3 の例 `--note <決めたこと>` の引用符なし: main(4fef6b8c)の skill の `:1117` に同じ文があり、`internal/org/skill_followup_test.go:274` が固定している。S13 の記録のとおり、この PR が持ち込んだ文ではない。

issue #271 の受け入れ条件 6 つ(OPEN、updatedAt 2026-10-27T07:15:09Z で、本文は変わっていない):

| issue の受け入れ条件 | Status | 根拠 |
| --- | --- | --- |
| 1. 分割計画の前の質問の手順 | 満たす | AC1 |
| 2. 分割計画の承認と、機能ごとの出力先の選択を聞く | 満たす | AC2、AC3 |
| 3. 重なる機能で「issue に起票」を推奨し、follow-up に記録する | 満たす | AC3、AC4 |
| 4. 先行する PR の merge のあと、着手してよいかを聞く | 満たす | AC5。この run で、頼る先が未実装なら「後に回す」を推奨するようになり、中断のあとの引き継ぎも答えを失わなくなった |
| 5. headless の director での聞き方の結果の記録 | 満たす | AC6(変化なし) |
| 6. skill の同期の検査が通る | 一部 | check-skill-sync と check-sync は通った。Go のテストは /test |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、54 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..6f919007` で clean。打ち直しはしていない |
| `cmp` で `/director` と `/org` の 4 つの写し、recipe の 2 つの写し | 同じ | `.claude`、`.agents`、`templates/base/` の 2 つ。`docs/recipes/director-wake.md` と `templates/base/` の写し |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ。この run は Progress だけを変えた |
| `git diff --check 4fef6b8c...HEAD`、差分の U+FFFD | どちらも空 | |
| `skill_split_questions_test.go` の `func Test` の数と行数 | 6f62affb と HEAD でどちらも 14 本、886 行から 995 行 | S12・S13 はテスト関数を足さず、既存のテストに断片と、`regexp` と `strings.Contains` の確かめを足した。節を読む関数(`directorSkillIntro`、`directorFollowupStep`、`directorSplitSubsection`、`directorSplitStep`)は変わっていない |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の「merge を待つ issue」(`.claude/skills/org/SKILL.md:1605-1665`)と command の表の `followup list`(`:255`) | yes | `followup done` は「人に聞いたら」打つもので、混ざった件の片付けも人の答えを写すので合う。`DONE_NOTE` の列と、制御文字のエスケープの説明もコードと合う。C2-V1 の例外の 1 文(`:1652-1656`)は、「今の `after:` のブランチの follow-up が 1 件もないとき」に狭めた 3 と食い違わない |
| `/org` skill の `- Depends on:` の定義(`:608`) | yes | S13 で、番号は `- Depends on:` の括弧ではなく issue の本文の別の行に書くことになった。`- Depends on:` は slug の並びのままで、S12 の文(括弧の中の番号)が持ち込みかけた食い違いは消えた |
| spec の FR-9 の「2026-11-01 に決めたこと(issue #271)」(`docs/specs/2026-10-07-org-multi-org-director.md:154`) | yes(C3-I1) | 食い違いはない。(5) は、頼られる issue の行による推奨、今の `after:` だけを数えること、done と open が混ざる issue の片付けを書いていない。(5) は着手の問いの推奨をもともと書かず(予約の重なりで「後に回す」も書いていない)、引き継ぎの「follow-up が 1 件もない issue も聞く」は、今の `after:` に絞った skill の条件の一部を言う文で、「そのときだけ」とは言っていない |
| README の `followup` の説明(`:131`)と director の段落(`:278`) | yes | 「`followup done <id>` marks one done once the human was asked」は、混ざった件を人の答えで閉じる手順と合う。この run で変わる文はない |
| `docs/recipes/director-wake.md` とその写し | yes | S12・S13 は push の 1 行と開いた問いに触れていない |
| `git grep` で S12 の古い文(「その issue の番号も」「数字だけの値」) | 残っていない | reports と plans を除いて 0 件 |
| `docs/tech-debt/README.md:362` と `:417` | yes | C2-V2 と C3-6 の直しで、どちらも「over 800 lines」 |
| tech-debt の Trigger | yes | Trigger の列(4 列目)を、この run が触った面(`/director` skill、`skill_split_questions_test.go`、`followup done`、`- Depends on:`)の語で掃いた。当たった行(`:174`〜`:249`、`:259`、`:347`、`:362`、`:376`、`:377`、`:386`)は、受信箱の処理、send のあとの確かめ、wait の終了コード、merge-order、`director_skill_sync_test.go` など、この run が触っていない面を名指す。`:417` の Trigger(テストを足す、節を読む関数を変える)は、cycle 2 と同じ読みで、テスト関数が増えず読む関数も変わらないので発火していない。`--limit` の行(`:418`)も、`--limit` が変わっていないので発火していない |
| 計画の Acceptance criteria の checkbox | 遅れ(cycle 1 の I2 のまま) | AC1〜AC8 は `[ ]` のまま |

### Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| C3-I1 | INFO | spec の要約 | 上の Documentation drift のとおり、spec の FR-9 の #271 の段落の (5) は S12・S13 の 3 つの振る舞いを書いていない。食い違いではない | 任意で、/sync-docs が (5) に 1 文ずつ足す(cycle 2 の sync-docs が S9〜S11 を足したのと同じ粒度) |
| C3-I2 | INFO | 混ざった件の片付け | `followup done` の `--note` は任意(`internal/cli/org_followup.go:174-176`)で、note なしの done の `DONE_NOTE` は `-` と出る。director 自身の 3 はいつも `--note` を付けるので、`-` になるのは人や `/org` skill の読み手が手で done にしたときだけ。そのとき「`DONE_NOTE` を人の答えとして読む」の答えがない。skill は `'` を含む場合だけを書いている | 任意で、「`DONE_NOTE` が `-` なら答えが残っていないので、人に聞く」と足す。直さないなら記録だけ |
| C3-I3 | INFO | 数え方 | 「今の `after:` のブランチの件」はブランチの名前だけで数える。3 の書き換えで新しく待つ相手の org が、前に待って done にしたブランチと同じ名前を持つとき(同じ slug の機能を merge のあとにもう一度立てたとき。`/org` skill が言う、名前の使い回し)、古い done の件と新しい open の件が「混ざっている」に当たり、引き継ぎが新しい待ちを古い答え「着手する」で閉じる。そのあとの「org が走っているか」の確かめで人に聞くので、人に聞かずに着手はしないが、待つ記録は消える | 記録だけ。起きるのは名前の使い回しと書き換えと中断が重なるときに限る |
| C3-I4 | INFO | insight event | events のファイルに self-review の 3 回目の run の行がない(commit 10ea72c6 は report だけを足した。cycle 2 の C2-I4 と同じ形)。この verify の行は、lead の指示どおり `--cycle 3` で作り、`2026-10-31-director-split-questions.jsonl` に足した | self-review の 3 回目の行を足すかを lead が決める |

LOW 以上の所見はない。

### Observational checks

- `gh issue view 271`: OPEN、updatedAt 2026-10-27T07:15:09Z。受け入れ条件 6 つは cycle 1 の表と同じ。
- HEAD の tree で、S12 の古い文(「先に起票したその issue の番号も」「その issue の番号も」「数字だけの値」)を reports と plans の外で `git grep` し、0 件だった。
- `頼られる issue` の正規表現が当たる行と当たらない行は、テストの 12 行の表(`internal/org/skill_split_questions_test.go` の `TestDirectorSkill_AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`)で読んだ。`#$(touch pwned)`、`;`、全角の数字、行末の CR を断る行がある。実行は /test。

### Coverage gaps

- `go test ./internal/org/` と変異の再実行は /test の担当で、打っていない。S12・S13 の implementer が記録した変異(7 つと 11 個)は、計画の Progress を読んだだけで確かめていない。
- `gh issue view <番号> --json state --jq .state` と `followup done --note '<DONE_NOTE>'` は、実際の issue と台帳では打っていない(書き込みか、頼られる issue のある本文がまだないため)。
- 実機の director での通し(Non-goals)と、PR の Known gaps に回すもの(self-review C3-2、C3-5、cycle 2 までの分)は残る。

### Verdict

pass

- Verified: S12・S13 で、AC4・AC5 の文が cross-review cycle 2 の #1〜#3 と self-review の 3 回目の run の C3-1・C3-3・C3-4・C3-6 を直したこと、AC1〜AC3・AC6・AC7 の節が変わっていないこと、S12・S13 がコードについて言うこと(`AFTER_BRANCH` と `DONE_NOTE` の列、note に制御文字が入らないこと)を確かめた。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。README、spec、recipe、`/org` skill に食い違いはない。
- Partially verified: AC8 と issue の受け入れ条件 6 は、静的の部分だけ。
- Not verified: Go のテストの実行(/test)、`gh issue view` の state と `followup done` の書き込み、実機の director での通し(Non-goals)。
- 所見: INFO 4 件(C3-I1〜C3-I4)。LOW 以上はない。どれも merge を止めない。
- /sync-docs に回すもの: なし。任意: C3-I1(spec の (5))、C3-I2(skill の 1 文。skill の文を変えるので、記録だけの直しの例外には入らない)。
- Insight event: `./scripts/insights-append.sh --slug director-split-questions --flow standard --phase verify --verdict pass --cycle 3 --source skill --events-dir <scratch>` で作った行を、`docs/insights/events/2026-10-31-director-split-questions.jsonl` の末尾に写した。

## Cycle 4

- Date: 2026-11-01
- Verifier: verifier サブエージェント(`/verify`、パイプラインの 4 回目の run。`.harness/state/standard-pipeline/cycle-count.json` は `cycle` 3、`cap` 3 で、上限の最後の run)
- Scope: この run の差分は `git diff 3edfa501..HEAD`(S14 09c9f34d、S15 ac5a0578 と、計画と self-review の記録。7 ファイル、+242/−62)。AC1〜AC8 と issue #271 の受け入れ条件は、ブランチ全体 `git diff 4fef6b8c...HEAD`(HEAD 929a3b71、25 ファイル、+4429/−121)で確かめ直した。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は読み取りだけにした(`gh issue view 271`)。main のチェックアウトには触れていない。
- Evidence: `./scripts/run-static-verify.sh` の全出力は `docs/evidence/verify-2026-11-01-director-split-questions-c4.log`、run-verify 自身のログは `docs/evidence/verify-2026-11-01-034847.log`(どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。
- static は 03:48:47Z に始まり、約 30 秒で終わった。終わったあとの load average は 6.32(1 分値)。

### Spec compliance

行番号は HEAD の `.claude/skills/director/SKILL.md` のもの。`/director` と `/org` の 4 つの写しは、それぞれ `cmp` で同じバイト列だった(sha256 の先頭は `/director` が `28e79f781d43`、`/org` が `5af698fd6a7c` で、`/org` は cycle 3 から変わっていない)。この run の差分が触ったのは、「機能ごとの出力先」の 3(`:582-585`)、「起票と記録」の 1(`:594-598`)、「merge を待つ issue」の引き継ぎ(`:1322-1325`)、2(`:1352-1365`)、3(`:1406-1408`)だけだった。S14 が引き継ぎに足した「すべて open で、どのブランチも merged、STATE はどれも waiting」の規則は S15 でブランチごとの規則に置き換わり、HEAD には残っていない。

| Acceptance criterion | Status | この run で確かめたこと |
| --- | --- | --- |
| AC1 分割計画の前の質問の段 | 満たす(変化なし) | `### 分割計画の前の質問`(`:501-531`)に触れていない |
| AC2 分割計画の承認 | 満たす(変化なし) | `### 分割計画の承認`(`:532-552`)に触れていない |
| AC3 機能ごとの出力先と、start の断りのあとの聞き直し | 満たす | 断られた機能を「issue に起票」の推奨で聞き直すことは変わらない。聞き直す前に「起票と記録」の 1 のとおり重なる org を全部数え直し、説明に全部書く文が足された(`:582-585`、cross-review の 3 回目の run の #1)。start のエラーが最初に重なった 1 つだけを出すことは、`reservationDecision` が最初の重なりで返す(`internal/org/reserve.go:576`)ことで確かめた |
| AC4 起票の手順と、引き継ぎでの記録し直し | 満たす | 「起票と記録」の 1 は、先に `ralph status` を打ち直し、`reserved:` を `- Reserve:` と「機能ごとの出力先」の 1 の規則で比べて、重なる org を全部数え直す(`:594-598`)。材料と比べ方は「merge を待つ issue」の 3 の数え直し(`:1420-1425`)と同じ。引き継ぎは、今の `after:` のブランチの follow-up がすべて open の issue も 2 の判定で確かめ、そろえば 2 の問いで聞き、答えのあと 3 のとおり done にする(`:1322-1325`、#2)。答えを聞く前に done にしない |
| AC5 merge のあとの着手の問いと、答えの記録 | 満たす | 2 は、open の follow-up の STATE が `waiting` でも、そのブランチが `merge-order --all` で `merged` なら「記録の直前に merge 済み」として待ち終えたと数える(`:1355-1362`)。その件のある issue は、`ready` の件がなくても報告のたびに確かめる(`:1352-1355`)。印の行のない issue も同じ(`:1363-1365`)。この件の merge は時刻で見れば記録より前の merge なので、AC5 の「記録の前に merge 済みのときだけ聞き」の範囲に入る(AC5 は follow-up がないことを条件にしていない)。問いの文と 3 つの選択肢(`:1367-1374`)は変わっていない。3 は、add のあと done の前に止まった件を 2 が拾い、もう一度聞いてから done にすると書く(`:1406-1408`) |
| AC6 headless の聞き方の証跡と skill の決まり | 満たす(変化なし) | `### 人への聞き方`(`:476-500`)、証跡、`### 計画の承認`(`:956-`)に触れていない |
| AC7 「予約外の文書の follow-up」の 1 が起票した機能を数えない | 満たす(変化なし) | `### 予約外の文書の follow-up`(`:1444-`)に触れていない |
| AC8 テスト、check-skill-sync、check-sync | 一部(静的の部分は満たす) | `run-static-verify.sh` rc 0。その中の check-skill-sync(14 本)と check-sync(DRIFTED 0)が通った。`go test` は /test |

cross-review の 3 回目の run(triage は 3edfa501 の版)と、self-review の 4 回目の run の所見は、次のとおり。

| 所見 | 状態 | 根拠 |
| --- | --- | --- |
| cross-review の 3 回目の run の #1(最初の start の断りのあと、重なる org を数え直さない) | 直った | 上の AC3、AC4 |
| cross-review の 3 回目の run の #2(add のあと done の前に止まると、着手の答えが失われる) | 直った | 上の AC4、AC5。止まった件では、add を打ったブランチは open の `waiting`、打たなかったブランチは follow-up なしの `merged` になり、2 の判定で全部そろう |
| self-review の Cycle 4 の C4-1〜C4-5(LOW) | 直していない | self-review が、直すか Known gaps に回すかをリードに回している。C4-1 の道は下で確かめた |

S15 の文がコードとほかの手順について言うことを確かめた。

- 「記録のときは merge されていないことを確かめてから記録する」: director が follow-up を足す場所は 5 つある。「起票と記録」の 4(`:624-628`。`merge-order --all` で `merged` なら記録しない)、引き継ぎ(`:1303-1306`。同じ 4 を使う)、「merge を待つ issue」の 1(`:1332-1333`)、3 の書き換え(`:1435-1437`。同じ 4 を使う)、3 の答えの記録(`:1397-1402`。merge 済みのブランチに足してすぐ done)。答えの記録のほかは、どれも merge を確かめてから足す。S15 が挙げる 2 つの原因(確かめたあと記録の前の merge、答えの記録の途中の停止)は、この 5 か所と合う。
- self-review C4-1 の 3 つ目の原因(ブランチ名の使い回し)は、director が自分で足す follow-up からは生じない。merge-order の entry(`pickMergePR`、`internal/org/mergeorder.go:1000`)と follow-up の判定の top(`pickFollowupPRs`、`internal/org/mergeorder_followup.go:344`)は、どちらも番号が最大の fork でない PR を採る。古い PR が merge 済みで新しい PR がまだない名前は entry が `merged` になり、上の 4 か所は記録しない。生じるのは、`/org` skill の「PR がもう merge されたブランチは follow-up にしない」(`.claude/skills/org/SKILL.md:1652`)に反して手で足した件だけで、C4-1 の LOW の評価に同意する。
- 「merge-order で `merged`」: `merged` の分類は `--all` のときだけ出る(`/org` skill の command の表、`:259`)。2 の新しい確かめは `ralph org merge-order --all` を名指しているので合う。gh がなければ merge-order は merged を見分けないので、この数え方は働かず、件は `waiting` のまま残る。聞かない側に倒れるので、「人に聞かないまま issue に着手しない」とはぶつからない。
- コードの DETAIL: 記録より前に merge された PR の follow-up の DETAIL は、新しい PR の merge を待つこと、来ないなら今 issue について人に聞いて `followup done` を打つことを言う(`internal/org/mergeorder_followup.go:330-334`)。S15 の「2 の問いで聞き、答えのあと done」は、この指示と同じ向き。

issue #271 の受け入れ条件 6 つ(OPEN、updatedAt 2026-10-27T07:15:09Z で、本文は変わっていない):

| issue の受け入れ条件 | Status | 根拠 |
| --- | --- | --- |
| 1. 分割計画の前の質問の手順 | 満たす | AC1 |
| 2. 分割計画の承認と、機能ごとの出力先の選択を聞く | 満たす | AC2、AC3 |
| 3. 重なる機能で「issue に起票」を推奨し、follow-up に記録する | 満たす | AC3、AC4。この run で、記録の前に重なる org を数え直すようになった |
| 4. 先行する PR の merge のあと、着手してよいかを聞く | 満たす | AC5。この run で、確かめたあと記録の前に merge されたブランチと、答えの記録が途中で止まった issue でも聞くようになった |
| 5. headless の director での聞き方の結果の記録 | 満たす | AC6(変化なし) |
| 6. skill の同期の検査が通る | 一部 | check-skill-sync と check-sync は通った。Go のテストは /test |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、約 30 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..929a3b71` で clean。打ち直しはしていない |
| `cmp` で `/director` と `/org` の 4 つの写し | 同じ | `.claude`、`.agents`、`templates/base/` の 2 つ |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ。この run は Progress だけを変えた |
| `git diff --check 4fef6b8c...HEAD`、差分の U+FFFD | どちらも空 | |
| `skill_split_questions_test.go` の `func Test` の数と行数 | 3edfa501 と HEAD でどちらも 14 本、995 行から 1037 行 | S14・S15 はテスト関数を足していない。断片を更新し、既存のテストの中に、2 に古い文(「waiting のブランチが残っていれば聞かない」)が残っていないことの否定の確かめを 1 つ足した。節を読む関数は変わっていない |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の「merge を待つ issue」の `ready` の決め方(`.claude/skills/org/SKILL.md:1628-1651`) | yes | 「記録より前の merge は数えない」は、`ralph status` と merge-order が出す STATE の `ready` の定義。S15 は STATE もその定義も変えず、director の側で「待ち終えた」と数える。`ready` にならない件をすぐ人に聞く向きは、`:1652-1656`(merge 済みなら記録せずすぐ聞く)とも、上のコードの DETAIL とも同じ。名前の使い回しを数えない理由(`:1629-1630`)は、上の到達の道のとおり、director の記録の手順が先に merge を確かめることで守られる |
| `/org` skill の答えの記録の例外(`:1652-1656`) | yes(C4-I2) | add と done のあいだで止まった件のことは書いていない。「待つための件ではなく答えの記録」は止まった件にも当てはまるので、食い違いではない |
| `/org` skill の command の表の `followup add`(`:254`)と `merge-order`(`:259`) | yes | `merged` が `--all` のときだけ出ること、merge 済みのブランチは follow-up にしないことと合う |
| spec の FR-9 の「2026-11-01 に決めたこと(issue #271)」(`docs/specs/2026-10-07-org-multi-org-director.md:154`) | yes(C4-I1) | (3)〜(5) は、S14 の起票の前の数え直しと、S15 の数え方を書いていない。(5) の「どのブランチにも ready の follow-up があれば…聞く」は聞く場合を挙げる文で、「そのときだけ」とは言っていない。(5) の終わりの、着手のあと start が断ったときに `ralph status` を読み直して重なる org を全部数える文は、S14 と同じ材料の数え直しで合う |
| README の `followup` の説明(`:131`)と director の段落(`:278`) | yes | 「followup done marks one done once the human was asked」は、S15 の「答えのあと done」と合う。README は起票の手順と数え方を書いていない |
| `docs/recipes/` とその写し | yes | `ralph-split`、`followup`、`reserved:` は recipe に出てこない(`grep` で 0 件)。`director-wake.md` の push の 1 行と開いた問いに、S14・S15 は触れていない |
| `git grep` で S14 の規則と S15 の前の文 | 残っていない | reports と plans の外で、S14 の「すべて merged」「どれも waiting」は 0 件。「waiting のブランチが残っていれば聞かない」は、テストの否定の確かめ(`internal/org/skill_split_questions_test.go:808`)にだけある |
| tech-debt の Trigger | yes | `:417` の Trigger(このファイルへのテストの追加、節を読む関数の変更)は、テスト関数が 14 本のままで読む関数も変わらないので、cycle 2・3 と同じ読みで発火していない(既存のテストの中の確かめの追加は数えていない)。1037 行なので `:417` の「over 800 lines」は正しい。`:386` の 4 ファイルに、この run は触れていない |
| 計画の Acceptance criteria の checkbox | 遅れ(cycle 1 の I2 のまま) | AC1〜AC8 は `[ ]` のまま |

### Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| C4-I1 | INFO | spec の要約 | 上の Documentation drift のとおり、spec の FR-9 の #271 の段落は S14・S15 の振る舞いを書いていない。食い違いではない | 任意で、/sync-docs が (4) に起票の前の数え直しを、(5) に「open の件が `waiting` のまま merge-order で `merged` のブランチも待ち終えたと数え、答えの記録が途中で止まった issue は聞き直す」を 1 文ずつ足す |
| C4-I2 | INFO | `/org` skill | `/org` skill の答えの記録の例外(`:1652-1656`)は、add と done のあいだで止まると、その件が open の `waiting` として `follow-ups:` に残ることを書いていない。その件の DETAIL が次の手(人に聞いて done)を言うので、`/org` の読み手も迷わない | 任意で 1 文。直さないなら記録だけ |
| C4-I3 | INFO | insight event | events のファイルに self-review の 4 回目の run の行がない(929a3b71 は report だけを足した。C2-I4、C3-I4 と同じ形)。この verify の行は、lead の指示どおり `--cycle 4` で作り、末尾に写した | self-review の 4 回目の行を足すかを lead が決める |

LOW 以上の所見はない。self-review の Cycle 4 の C4-1〜C4-5(LOW)は、この report の所見には数えていない。

### Observational checks

- `gh issue view 271`: OPEN、updatedAt 2026-10-27T07:15:09Z。
- `reservationDecision`(`internal/org/reserve.go:539`)を読み、走っている org を順に見て最初の重なりでエラーを返すことを確かめた。S14 の「start のエラーは最初に重なった org だけを出す」と合う。
- 記録より前に merge された PR の follow-up の STATE と DETAIL を、`internal/org/mergeorder_followup.go:330-334` で読んだ。STATE は `waiting` で、DETAIL は人に聞いて done にする道を言う。

### Coverage gaps

- `go test ./internal/org/` と変異の再実行は /test の担当で、打っていない。S14・S15 の implementer が記録した変異(5 つと 9 個)は、計画の Progress を読んだだけで確かめていない。
- 確かめたあと記録の前の merge と、答えの記録の途中の停止は、実際の台帳と GitHub では起こしていない(書き込みが要るため)。コードと skill の読みだけで確かめた。
- 実機の director での通し(Non-goals)と、PR の Known gaps に回すもの(self-review の C3-2、C3-5、C4-1〜C4-5 のうち直さないもの)は残る。

### Verdict

pass

- Verified: S14・S15 で、cross-review の 3 回目の run の #1・#2 が直ったこと、AC3〜AC5 の文がそれに合わせて変わり、AC1・AC2・AC6・AC7 の節が変わっていないことを確かめた。S15 の「記録のときは merge されていないことを確かめてから記録する」は、director が follow-up を足す 5 か所と合い、名前の使い回しの道は director の記録からは生じない。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。`/org` skill の「記録より前の merge は数えない」、README、spec、recipe に食い違いはない。
- Partially verified: AC8 と issue の受け入れ条件 6 は、静的の部分だけ。
- Not verified: Go のテストの実行(/test)、台帳と GitHub での停止と競合の再現、実機の director での通し(Non-goals)。
- 所見: INFO 3 件(C4-I1〜C4-I3)。LOW 以上はない。どれも merge を止めない。
- /sync-docs に回すもの: なし。任意: C4-I1(spec の (4)・(5))、C4-I2(`/org` skill の 1 文)。
- Insight event: `./scripts/insights-append.sh --slug director-split-questions --flow standard --phase verify --verdict pass --cycle 4 --source skill --events-dir <scratch>` で作った行を、`docs/insights/events/2026-10-31-director-split-questions.jsonl` の末尾に写した。

## Cycle 5

- Date: 2026-11-01
- Verifier: verifier サブエージェント(`/verify`、パイプラインの 5 回目の run で、最後の run。`.harness/state/standard-pipeline/cycle-count.json` は `cycle` 3、`cap` 4。insight event の cycle はリードの指示で 5)
- Scope: この run の差分は `git diff 9b1175c3..HEAD`(S16 a7e7e4bd、S17 cfbfe147 と、計画と self-review の記録。8 ファイル、+462/−179)。AC1〜AC8 と issue #271 の受け入れ条件は、ブランチ全体 `git diff 4fef6b8c...HEAD`(HEAD ca4e87d3、25 ファイル、+4978/−128)で確かめ直した。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は読み取りだけにした(`gh issue list`・`gh issue view 271`・`gh pr view 294` を `--repo yoshpy-dev/ralph` で、skill の `sed` を手元の文字列に)。main のチェックアウトには触れていない。
- Evidence: `./scripts/run-static-verify.sh` の全出力は `docs/evidence/verify-2026-11-01-director-split-questions-c5.log`、run-verify 自身のログは `docs/evidence/verify-2026-11-01-052854.log`(どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。
- static は 05:28:54Z に始まり、約 31 秒で終わった。終わったあとの load average は 7.79(1 分値)。

### Spec compliance

行番号は HEAD の `.claude/skills/director/SKILL.md` のもの。`/director` と `/org` の 4 つの写しは、それぞれ `cmp` で同じバイト列だった(sha256 の先頭は `/director` が `f2fa1d5b8f51`、`/org` が `9ea5050fccd6`。`/org` は sync-docs の cycle 4(844c0061)で変わり、この run の差分には入っていない)。この run が触ったのは、「分割計画の前の質問」の 1(`:507-510`)、「起票と記録」の 3・4 と末尾の `--repo` の決まり(`:623-641`、`:668-688`)、「merge を待つ issue」の引き継ぎ(`:1305-1323`)、2(`:1371-1404`)、3(`:1438-1468`)、「予約外の文書の follow-up」の 1・2(`:1500-1533`)。

| Acceptance criterion | Status | この run で確かめたこと |
| --- | --- | --- |
| AC1 分割計画の前の質問の段 | 満たす | 1 の `gh issue view` に `--repo` が付いただけで、手順 1〜3、4 問まで、推奨を先頭、Codex の番号の選択肢、手順 4〜8 の扱い、機能をまたぐ分かれ目と `FORK` の文は変わっていない |
| AC2 分割計画の承認 | 満たす(変化なし) | `### 分割計画の承認`(`:533-553`)に触れていない |
| AC3 機能ごとの出力先と、start の断りのあとの聞き直し | 満たす(変化なし) | `### 機能ごとの出力先`(`:554-589`)に触れていない |
| AC4 起票の手順と、引き継ぎでの記録し直し | 満たす | 3 の `gh issue create` に `--repo` が付いた(`:623-624`)。4 に、そのブランチを `feature:` の行に持つ org が走っていれば `merged` でも merge 済みと数えない、のただし書きが入った(`:629-634`)。数えないブランチは記録に回る。引き継ぎも同じただし書きを指す(`:1321-1323`)。本文の印の行の読み方(ちょうど 1 行、書いた人、組み直し)は変わっていない |
| AC5 merge のあとの着手の問いと、答えの記録 | 満たす | 2 の「記録の直前に merge 済み」の数え方にただし書きが入った(`:1383-1385`)。問いの文と 3 つの選択肢は変わっていない。3 で着手するとき、書き直す前に `./scripts/ralph-worktree.sh state-path org-<slug>` の記録を確かめ、あれば slug を変えるか人に聞き、`cleanup` は人に頼まれないまま打たない(`:1438-1449`) |
| AC6 headless の聞き方の証跡と skill の決まり | 満たす(変化なし) | `### 人への聞き方`(`:476-500`)、証跡、`### 計画の承認`(`:975-`)に触れていない |
| AC7 「予約外の文書の follow-up」の 1 が起票した機能を数えない | 満たす | 1 の `gh issue view` と `gh issue list` に `--repo` が付いただけで、印の行の issue の機能を数えない文は変わっていない(`:1497-1510`) |
| AC8 テスト、check-skill-sync、check-sync | 一部(静的の部分は満たす) | `run-static-verify.sh` rc 0。その中の check-skill-sync(14 本)と check-sync(DRIFTED 0)が通った。`go test` は /test |

cross-review の 4 回目の run(triage は 9b1175c3 の版)の所見は、次のとおり。

| 所見 | 状態 | 根拠 |
| --- | --- | --- |
| #1 ブランチ名の使い回しで、merge-order が前の PR を `merged` と出し、今の待ちを済んだと数える | 直った(「merge を待つ issue」の 1 を除く) | `classifyMergeTarget`(`internal/org/mergeorder.go:1068-1076`)は PR の state だけで `merged` を返し、`pickMergePR`(`:1000`)は番号が最大の fork でない PR を採る。新しい org が PR を立てる前は、前の PR の `merged` が出ることを確かめた。ただし書きは 4・引き継ぎ・2 の 3 か所で同じ向きに効く。`ralph org status` は `feature: <split>/<feature> branch <ブランチ> ...` の行を出す(`internal/cli/org.go:1908`)ので、読み方も合う。「merge を待つ issue」の 1(`:1353`)は自分ではただし書きを指さない。計画の S17 の行と self-review の C5-1 のとおり、PR の Known gaps に回る |
| #2 確かめのあとの予約の取り合いで、start が残した worktree が書き直した分割計画の start を止める | 直った | `featureWorktreeID` は `"org-" + orgID`(`internal/org/feature.go:162-164`)、org_id の既定は slug、`state-path <task-id>` は記録の有無によらずパスを出す(`scripts/ralph-worktree.sh:13`、`:66-73`)。別の分割計画の記録は `canonical_ref`(`split:<パス>#<slug>`、`feature.go:178`)の違いで断られる(`feature.go:558`、`:596`)。案内は `/org` skill の「start が拒否するもの」(`.claude/skills/org/SKILL.md:741-`)の slug を変える枝と同じ |
| #3 fork の checkout で gh の issue のコマンドが upstream を指す | 直った | `gh issue` のコマンドは 11 か所すべてに `--repo <owner>/<repo>` が付く(`:508`、`:623`、`:669`、`:1305`、`:1310`、`:1371`、`:1404`、`:1462`、`:1468`、`:1500`、`:1508`)。`gh pr view` も付く(`:1529`)。`--repo` のない `gh` は `gh api user`、禁じる側の `gh pr merge`、ralph の振る舞いを述べる `gh pr checks` だけ。`gh issue list --help` と `gh pr view --help` は `-R, --repo [HOST/]OWNER/REPO` を出す。skill の `sed` はこの repo の origin で `yoshpy-dev/ralph` を出し、その値で `gh issue list`(引き継ぎの形)・`gh issue view 271`・`gh pr view 294` が rc 0 で読めた |

S16・S17 がコードとほかの手順について言うことを確かめた。

- `sed` は、`https://`(`.git` と末尾の `/` の有無、`user@`)、`ssh://`(port つきを含む)、`git@<host>:`、`git+ssh://` の 11 通りで `o/r` を出した。URL でない origin では、`/Users/me/src/foo/bar` も `file:///srv/git/foo/bar.git` も `foo/bar` を出す。ralph の `merge` と `cleanup` は、こうした origin を断り、host つきの `<host>/<owner>/<repo>` に固定する(`MergeOrigin.Arg`、`internal/org/mergeorigin.go:28-33`)。skill は host を落とし、URL でない origin で止まる文がない。self-review の C5-3(LOW)と同じ所見で、評価に同意する。付け足すと、`gh issue create` は外に出る書き込みなので、`foo/bar` が github.com にある他人の repository で issue を受け付けると、そこに issue が立つ。起きるのは origin がローカルのパスか `file://` の repo で director が起票するときだけ。
- 3 の括弧の文は、worktree が残る場合を 2 つに限る(`:1439-1443`)。self-review の C5-2 のとおり、worktree を作ったあとの spawn の失敗は理由によらず残す(`feature.go:271-274`)。加えて、予約の取り合いの窓は「worktree を作るまで」ではなく、spawn が manifest のロックの下で確かめ直すまで(`feature.go` の冒頭の 2 と 6)。別の分割計画の同じ slug の org が残した worktree(走っている、PR の merge を待つ、merge のあと片付けの前)も同じ記録に当たる。確かめそのものは「ファイルがあれば」で、原因によらず働くので、手順は誤らない。
- 「merge-order は origin で PR を読む」(`:1531`)は、正確には「gh の既定の repository で引き、URL が origin の owner/repo の PR だけを使う」(`mergeorder.go:35-38`、`/org` skill の command の表 `:259`)。PR の列に出る番号は origin の PR なので、`--repo` を付ける結論は同じ。gh の既定が upstream の fork の checkout では、その PR は使われず列が空になる。

前の run の report について: Cycle 4 の「名前の使い回しの道は director の記録からは生じない」は誤りだった。director は記録の前に merge-order で merge を確かめるが、新しい org がまだ PR を立てていないあいだは、その確かめ自体が前の PR の `merged` を返す。cross-review の 4 回目の run の #1 がこの道で、S16 が直した。

issue #271 の受け入れ条件 6 つ(OPEN、updatedAt 2026-10-27T07:15:09Z で、本文は変わっていない):

| issue の受け入れ条件 | Status | 根拠 |
| --- | --- | --- |
| 1. 分割計画の前の質問の手順 | 満たす | AC1 |
| 2. 分割計画の承認と、機能ごとの出力先の選択を聞く | 満たす | AC2、AC3 |
| 3. 重なる機能で「issue に起票」を推奨し、follow-up に記録する | 満たす | AC3、AC4。この run で、起票が origin の repository に固定され、使い回しの名前の前の PR で記録を飛ばさなくなった |
| 4. 先行する PR の merge のあと、着手してよいかを聞く | 満たす | AC5。この run で、使い回しの名前で待ちを済んだと数えなくなり、着手の start を残った worktree が止める場合の案内が入った |
| 5. headless の director での聞き方の結果の記録 | 満たす | AC6(変化なし) |
| 6. skill の同期の検査が通る | 一部 | check-skill-sync と check-sync は通った。Go のテストは /test |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、約 31 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、guard の awk、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..ca4e87d3` で clean。打ち直しはしていない |
| `cmp` で `/director` と `/org` の 4 つの写し | 同じ | `.claude`、`.agents`、`templates/base/` の 2 つ |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ。この run は Progress だけを変えた |
| `git diff --check 4fef6b8c...HEAD`、差分の U+FFFD | どちらも空 | |
| `skill_split_questions_test.go` の `func Test` の数と行数 | 9b1175c3 と HEAD でどちらも 14 本、1037 行から 1092 行 | S16・S17 はテスト関数を足していない。定数 2 つ(`directorSplitRepoFlag`、`directorSplitOriginRepoCmd`)を足し、helper `directorSplitMarkIssuesCmd` の文字列と既存のテストの断片を更新した。節を読む関数は変わっていない |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の「start が拒否するもの」(`.claude/skills/org/SKILL.md:741-`) | yes | 記録 `org-<org_id>` の `canonical_ref` が違えば断る、org_id が slug なら slug を変えて承認し直す、要らないときに限り `cleanup --id org-<org_id>`。director の 3 の文はこの枝を指し、`cleanup` を人に頼まれないまま打たないので、`/org` の「要らないときに限り」と合う |
| `/org` skill の「merge を待つ issue」(`:1605-`) | yes | follow-up の `ready` は「記録より後の merge」で決まり、使い回しの名前の前の merge は数えないと書く。S16 のただし書きは merge-order の entry の `merged` についての director 側の数え方で、STATE の定義を変えない。「PR がもう merge されたブランチは follow-up にしない」とも食い違わない(使い回しのときに merge 済みなのは前の PR で、待つ PR ではない) |
| `/org` skill の `merge-order` の行(`:259`) | yes(C5-I1) | gh の既定の repository で引き、origin の PR だけを使う。director の「origin で PR を読む」は言い方が粗いが、結論は同じ |
| spec の FR-9 の「2026-11-01 に決めたこと(issue #271)」(`docs/specs/2026-10-07-org-multi-org-director.md:154`) | yes(C5-I2) | `--repo`、使い回しの名前のただし書き、`state-path` の確かめを書いていない。(4) の「起票は `gh issue create --body-file` で行い」と (5) の `gh issue view <番号> --json state` は `--repo` を省いた要約で、「付けない」とは言っていない。(5) の「`waiting` でも `merged` のブランチは…待ち終えたと数える」は、ただし書きを省いた要約 |
| README の director の段落(`:278`)と `followup` の説明 | yes | 起票の手順、`--repo`、worktree の記録を書いていない。`ralph-split`・使い回し・`--repo` は README と recipe に出てこない(`git grep` で 0 件) |
| `docs/recipes/` とその写し | yes | S16・S17 に関わる記述がない。`state-path` は skill の 4 つの写しと `tests/test-ralph-worktree.sh` のほかに出てこない |
| tech-debt の Trigger | yes | `:417`(このファイルへのテストの追加、節を読む関数の変更)は、テスト関数が 14 本のままで節を読む関数も変わらないので、cycle 2〜4 と同じ読みで発火していない。`:418`(どちらかの `--limit` の変更、このファイルへのテストの追加)は、`--repo` を足しただけで `--limit` は 1000 のままなので発火していない。`:386` の 4 ファイルに、この run は触れていない |
| 計画の Acceptance criteria の checkbox | 遅れ(cycle 1 の I2 のまま) | AC1〜AC8 は `[ ]` のまま |

### Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| C5-I1 | INFO | skill の言い方 | 「merge-order は origin で PR を読む」(`:1531`)と、同じ意味のテストの comment(`skill_doc_followup_test.go` の `TestDirectorSkill_DocFollowupCommandsTakeThePRFromMergeOrder` の doc comment)は、merge-order の実際の読み方(gh の既定で引き、origin の PR だけを使う)より粗い。`--repo` を付ける結論は同じ | 直さなくてよい。直すなら「merge-order は origin の PR だけを出すので」 |
| C5-I2 | INFO | spec の要約 | spec の FR-9 の #271 の段落は、S16・S17 の振る舞い(`--repo`、使い回しの名前のただし書き、書き直しの前の `state-path` の確かめ)を書いていない。食い違いではない | 任意で /sync-docs が (4)・(5) に 1 文ずつ |
| C5-I3 | INFO | skill の理由の文 | self-review の C5-2 に足す点: 3 の括弧の文の予約の取り合いの窓は「worktree を作るまで」より広く(spawn のロックまで)、別の分割計画の同じ slug の org の worktree も同じ記録に当たる。確かめは原因によらず働く | C5-2 と一緒に PR の Known gaps に書くか、括弧の文を「残ることがある(例: …)」にする |
| C5-I4 | INFO | insight event | events のファイルに self-review の 5 回目の run の行がない(ca4e87d3 は report だけを足した。C2-I4〜C4-I3 と同じ形)。この verify の行は、リードの指示どおり `--cycle 5` で作り、末尾に写した | self-review の 5 回目の行を足すかをリードが決める |

LOW 以上の所見はない。self-review の Cycle 5 の C5-1〜C5-5(LOW)は、この report の所見には数えていない。そのうち C5-3(host を落とし、URL でない origin を読み違える)は上の probe で同じ結果を得た。

### Observational checks

- `gh issue view --repo yoshpy-dev/ralph 271 --json state --jq .state`: `OPEN`。updatedAt 2026-10-27T07:15:09Z。
- 引き継ぎの `gh issue list --repo yoshpy-dev/ralph --state open --limit 1000 ...` は rc 0 で、1 行目の件数 `3` だけを出した(印の行の issue はない)。`gh pr view 294 --repo yoshpy-dev/ralph --json state --jq .state` は `MERGED`。
- `StartFeature` の順(`internal/org/feature.go` の冒頭の comment の 1〜6)を読み、予約は 2 でロックなしに、6 で manifest のロックの下で確かめ、worktree は 4 で作ることを確かめた。

### Coverage gaps

- `go test ./internal/org/` と変異の再実行は /test の担当で、打っていない。S16 の implementer が記録した変異 10 個は、計画の Progress を読んだだけで確かめていない。
- fork の checkout、GHES の origin、ローカルのパスの origin で `--repo` つきの gh を打つことはしていない(書き込みか、別の環境が要るため)。`sed` の出力と gh の help の読みだけで確かめた。
- 名前を使い回したブランチで、新しい org の PR の前に merge-order が `merged` を出すことは、コードの読みだけで確かめた。
- 実機の director での通し(Non-goals)と、PR の Known gaps に回すもの(self-review の C5-1〜C5-5、C4-2〜C4-5 のうち直さないもの)は残る。

### Verdict

pass

- Verified: S16・S17 で、cross-review の 4 回目の run の #1〜#3 が直ったこと(#1 は「merge を待つ issue」の 1 を除く)、AC1・AC4・AC5・AC7 の文がそれに合わせて変わり、AC2・AC3・AC6 の節が変わっていないことを確かめた。S16・S17 がコードについて言うこと(merge-order の `merged` の決め方、`org-<org_id>` の記録と `canonical_ref` の断り、`state-path` の使い方、`feature:` の行)はコードと合う。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。`/org` skill、README、spec、recipe に食い違いはない。
- Partially verified: AC8 と issue の受け入れ条件 6 は、静的の部分だけ。
- Not verified: Go のテストの実行(/test)、fork・GHES・ローカルのパスの origin での gh、名前の使い回しの実際の再現、実機の director での通し(Non-goals)。
- 所見: INFO 4 件(C5-I1〜C5-I4)。LOW 以上はない。どれも merge を止めない。
- /sync-docs に回すもの: なし。任意: C5-I2(spec の (4)・(5))。
- Insight event: `./scripts/insights-append.sh --slug director-split-questions --flow standard --phase verify --verdict pass --cycle 5 --source skill --events-dir <scratch>` で作った行を、`docs/insights/events/2026-10-31-director-split-questions.jsonl` の末尾に写した。

## Cycle 5 (S18)

- Date: 2026-11-01
- Verifier: verifier サブエージェント(`/verify`、パイプラインの 5 回目の run の追い分。上限は 4 のまま)
- Scope: `git show 172b713c`(S18。skill の 4 つの写しと `internal/org/skill_split_questions_test.go`、5 ファイル、+157/−10)と、その記録 fae04e46(計画の Progress に 2 行)。見たのは、AC4 の起票の手順と self-review C5-3 の直しとして足りるか、skill の「`ralph org merge` と `ralph org cleanup` もローカルの origin を断る」の文がコードと合うか、文書のずれ。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。probe は手元だけ(scratch の git repo の origin を付け替えて skill の 2 行を打つ、BSD grep、`pre_bash_guard.sh`)。GitHub には何も打っていない。main のチェックアウトには触れていない。insight event はリードの指示で足さない。
- Evidence: `./scripts/run-static-verify.sh` の全出力は `docs/evidence/verify-2026-11-01-director-split-questions-c5-s18.log`、run-verify 自身のログは `docs/evidence/verify-2026-11-01-054837.log`(どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。
- static は 05:48:37Z に始まり、約 32 秒で終わった。終わったあとの load average は 5.47(1 分値)。
- 同じ worktree の reviewer は、この verify の途中で `## Cycle 5 (S18)` の self-review(646aff7a、C5S-1〜C5S-4)をコミットした。下の所見はそれを読んでから書いた。

### Spec compliance

行番号は HEAD の `.claude/skills/director/SKILL.md` のもの。4 つの写しは `cmp` で同じバイト列(sha256 の先頭 `5c6b4361cd34`)。

| 項目 | Status | 確かめたこと |
| --- | --- | --- |
| AC4 起票の手順と、引き継ぎでの記録し直し | 満たす | 手順 1〜5(`:595-647`)と印の行の読み方は変わっていない。変わったのは `--repo` の決め方(`:685-704`)で、`<owner>/<repo>` を決める前に origin の URL を github.com の 3 つの形で確かめ(`:691`)、当たらなければ gh の issue と pr のコマンドを打たず、起票もせず、人に言う。当たらない checkout では起票の手順が使えなくなるが、止まる向きで、手順そのものはある |
| self-review C5-3 (b) URL でない origin で sed が `foo/bar` を出す | 直った | scratch の repo の origin を付け替えて skill の 2 行を打った。`/Users/x/foo/bar`、`../bar`、`file:///srv/git/foo/bar.git` は判定が断り、sed まで進まない |
| self-review C5-3 (a) host を落とす | github.com に絞って直った | gitlab.com・`github.com.example.com`・`github.com.emu` の別名は判定が断る。判定が通る URL では、sed の出力は github.com の owner/repo そのもの(4 通りで確かめた)。残るのは `GH_HOST` が github.com 以外のとき(self-review C5S-3 と同じ所見で、評価に同意する) |
| 「`ralph org merge` と `ralph org cleanup` も、origin がローカルのパスや `file://` のときは断る」(`:698-699`) | コードと合う | どちらも `ReadMergeOrigin`(`internal/org/mergeorigin.go:89-112`)で origin を読み、`mergeOriginFromURL`(`internal/org/mergeorder.go:892`)はスキームのない、`:` のない、または `:` の前に `/` がある値(ローカルのパス)と、host のない `file://` を `MergeOriginNoRepository` にする。merge は `readOrigin` で断り(`internal/org/merge.go:378-381`、「a local path, file://, …」)、cleanup は台帳を読む前に断る(`internal/cli/org_cleanup.go:163`、`:223-226`)。表のテストは `merge_test.go:682-683` の 2 行。文は「ほかの host」を断るとは言っておらず、ralph は gitlab.com などを断らないので、それとも食い違わない。計画の S18 の行の「merge-order は照合を飛ばすだけ」も `mergeOriginRepo.check`(`mergeorder.go:823-850`)と合う |
| テストの表(23 通り)と grep -E の読み | 一致 | 表の 23 行をそのまま BSD grep 2.6.0 の `grep -Eq` に通し、23 行とも Go の `regexp` の期待と同じ結果だった。`TestDirectorSkill_ChecksOriginNamesAGitHubRepositoryBeforeReadingOwnerRepo`(`skill_split_questions_test.go:678`)は 2 行を節の中の 1 行として固定し、判定の文の断片も `:583` で固定する |
| guard | deny されない | 判定と sed の 2 行を、worktree の root と `templates/base/` の `pre_bash_guard.sh` に default と bypassPermissions で通し、8 回とも出力なし。対照の `git commit` に `--no-verify` を付けた 1 行は deny になった |

当たってほしいが判定が断る形(`https://<user>:<token>@github.com/...`、`ssh://git@github.com:22/...`、`ssh://git@ssh.github.com:443/...`、`https://GitHub.com/...`、`ssh://github.com/...`、`https://github.com/o/r//`)は、self-review C5S-1 の probe と同じ結果だった。ralph の merge は `ssh.github.com` を github.com に読み替える(`internal/org/mergeguarantee.go:276-281`)ので、同じ origin で merge が通り起票だけが止まる組がある。止まる向きの違いで、C5S-1 の評価(LOW)に同意する。表の外で判定が通る `https://github.com/../..` と `https://github.com/o/.git` は、それぞれ `../..` と `https://github.com/o/` を出す。GitHub にそういう repository はなく、gh が断るだけ。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、約 32 秒) | scope は changed で、Go の pack が選ばれた。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、guard の awk、settings.json の jq、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..fae04e46` で clean。打ち直しはしていない |
| `cmp` で `/director` の 4 つの写し | 同じ | |
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308` | `- Approved:` の digest と同じ。fae04e46 は Progress だけを変えた |
| `git diff --check 0b02aecf..HEAD`、差分の U+FFFD | どちらも空 | |
| `skill_split_questions_test.go` の `func Test` の数と行数 | 0b02aecf の 14 本・1092 行から、HEAD で 15 本・1175 行 | S18 がテスト関数を 1 本足した。下の tech-debt の Trigger に当たる |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の `merge` と `cleanup` の行(`.claude/skills/org/SKILL.md:261-262`) | yes | merge は「ローカルのパスなど、読めない origin は断る」、cleanup は「origin が GitHub の repository を名指さないか(ローカルのパスなど)…台帳を読む前に、何も消さず断る」。director の文と合う |
| spec の FR-9 の #271 の段落 | yes | `--repo` も判定も書いていない要約で、食い違いはない(前の節の C5-I2 と同じ読み) |
| README、`docs/recipes/` | yes | director の起票の `--repo` と判定に触れる記述はない |
| tech-debt の Trigger(`docs/tech-debt/README.md:417`、`:418`) | no(C5S-V1) | どちらの行も Trigger に「このファイルへの次のテストの追加」(`:417` は "The next test added to this file"、`:418` は "the next test added to that file")を持つ。S18 は `skill_split_questions_test.go` にテスト関数を 1 本足したので、両方の Trigger に当たる。行は払われておらず(mark-line のテストと merge-wait のテストは同じファイルのまま、`--limit` の定数もない)、持ち越しの注もない。ほかの行の Trigger(第 4 列を python で読んだ。director・origin・skill を含む行)には当たらない |
| 計画の Acceptance criteria の checkbox | 遅れ(cycle 1 の I2 のまま) | AC1〜AC8 は `[ ]` のまま |

### Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| C5S-V1 | LOW | tech-debt の Trigger | `docs/tech-debt/README.md:417` と `:418` の Trigger(このファイルへの次のテストの追加)に S18 が当たったが、払っても持ち越しの注を書いてもいない。前の verify の節は「テスト関数が 14 本のままなので発火していない」と読んでおり、同じ読みでは今は発火している | /sync-docs が両方の行に「2026-11-01: S18(director-split-questions)がテストを 1 本足して当たったが払っていない。理由: 上限の最後の run で、移すだけの分割は skill の文の review と混ざる」の形で持ち越しの注を書く。払うなら別の slice |
| C5S-V2 | INFO | skill の流れ | 判定が断ったあとの次の手を skill が言わない。「機能ごとの出力先」で人が「issue に起票」を選んだ機能は、起票されないまま残る。self-review C5S-1 の後半(「director も次に何をするか決まらない」)と同じ所見で、評価に同意する | C5S-1 と一緒に直すか、PR の Known gaps に書く |

self-review の C5S-1〜C5S-4(LOW)はこの report の所見には数えていない。そのうち C5S-1(断る形と理由の文)と C5S-3(`GH_HOST`)は、上の probe とコードの読みで同じ結果を得た。C5S-2(断るときに token 入りの URL をそのまま人に言う)にも同意する。判定が断る形に `https://<user>:<token>@github.com/...` が入るので、S18 で新しくできた道である。

### Observational checks

- この repo の origin `ssh://git@github.com/yoshpy-dev/ralph.git` は判定に当たり(rc 0)、sed は `yoshpy-dev/ralph` を出した。
- scratch の repo で origin を 22 通りに付け替え、skill の 2 行を `/bin/sh` で打った。当たったのは github.com の 3 つの形の 4 通りと、上の `../..`・`.git` の 2 通りと、次の 1 通り。改行を含む origin(`git remote add` が受け付けた)では、grep が 1 行目で当たり、sed が 2 行を出した。手元の git の設定を書ける人しか作れない値なので、所見にはしていない。

### Coverage gaps

- `go test ./internal/org/` と、計画の S18 の行にある変異 3 つが落ちたことは、/test の担当で、確かめていない。
- GNU grep では打っていない(self-review が `ggrep` で 25 通りを確かめた記録を読んだだけ)。
- 判定が断る origin で、実機の director が人に何を言うか、`GH_HOST` が GHES の環境での起票は、確かめていない(書き込みか、別の環境が要るため)。

### Verdict

pass

- Verified: S18 は self-review C5-3 を、origin が github.com の repository を名指す checkout について直した。ローカルのパスと `file://` の origin では sed まで進まず、起票もしない。AC4 の起票の手順は変わらずにあり、判定はその前に止まる向きにだけ働く。「`ralph org merge` と `ralph org cleanup` もローカルの origin を断る」の文は `ReadMergeOrigin` と両方の断りに合う。テストの表 23 行は BSD grep でも同じ判定になる。静的解析(gofmt、vet、golangci-lint、staticcheck、check-sync、check-skill-sync、purity、secret scan)はすべて通った。4 つの写しは同じで、計画の digest も合う。
- Partially verified: AC8 は静的の部分だけ。
- Not verified: Go のテストと変異(/test)、GNU grep、実機の director が判定を打つこと、`GH_HOST` の場合。
- 所見: LOW 1 件(C5S-V1)、INFO 1 件(C5S-V2)。どちらも merge を止めない。
- /sync-docs に回すもの: C5S-V1(`docs/tech-debt/README.md:417` と `:418` の持ち越しの注)。
- Insight event: リードの指示で足していない。
