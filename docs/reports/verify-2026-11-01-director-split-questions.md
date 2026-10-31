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
