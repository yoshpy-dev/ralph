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
