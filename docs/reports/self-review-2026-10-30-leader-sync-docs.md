# Self-review report: leader-sync-docs

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-leader-sync-docs.md`(issue #268、D1〜D5、AC1〜AC8、スライス S1 2d480eb4・S2 4a437059・S3 44275a04)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 1e266f96、HEAD f9234b39、14 ファイル、+1519/-33)。leader の雛形 `internal/org/prompts/leader.md`、`/org` と `/director` の skill(写し 3 つずつ)、仕様の FR-4 の注、README の `ralph org gate` の段落、新しいテスト 2 本。雛形と skill は agent が従う指示なので、agent が読む順に読み、手順の番号、今の手順 9・10・12・14 と per-plan の手順 6 との食い違い、書いたとおりには動かないコマンド、読み違えうる文を見た。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1・M2(MEDIUM)と L1〜L6(LOW)。

## Evidence reviewed

- 計画の D1〜D5、AC1〜AC8、Progress checklist(S1 の implementer が挙げた `GATE: fail` の繰り返し、S3 の consult の申し送り)。
- `leader.md` の「機能ごとの org」の全体(110〜561 行)。追加の 3 段落(348〜396 行)を、手順 9 の前半(322〜346 行)、手順 10 の終わりの戻し方(407〜413 行)、手順 12 の上限(442〜445 行)と 3 つの枝(447〜494 行)、手順 14(496〜550 行)、ミッションの 5(77〜86 行)と突き合わせた。
- `/org` skill の「org の終わり方」の 1〜7(989〜1197 行)、「座席内 fan-out」(1503〜1524 行)、「完了条件」(1856〜1880 行)。`/director` skill の「件の処理」(643〜690 行)と新しい節「予約外の文書の follow-up」(1077〜1116 行)。
- 写し: `/org` と `/director` の 4 つの写しは `cmp` で一致した。
- 雛形が名指すものが今のコードと合うか: 予約の読み方(`internal/org/reserve.go:209-250` の `normalizeReservePath` と `reservePathsOverlap`。ディレクトリは `/` で終わり、`.` は repo 全体)、タスクの `- 機能:` と `- 予約したパス:` の行(`internal/org/feature.go:420`、`:423`)、機能のブランチ `<type>/<slug>`(`feature.go:209`)と `- Type:` の値に `docs` があること(`split.go:69-72`)、`ralph org inbox --all` が resolved を出し ORG と TYPE の列を持つこと(`internal/cli/org_inbox.go:127-131`、`:395`)、merge-order の ORG と PR の列(`internal/cli/org_merge_order.go:207-218`)、`ValidateInboxNote` の 1 行・500 字(`internal/org/inbox.go:401-416`)。
- doc-maintainer の定義(`.claude/agents/doc-maintainer.md`、`templates/base/` の写し、`.codex/agents/doc-maintainer.toml`)と `/sync-docs` skill。skill は計画の進み具合の更新と insight event を指示するが、雛形の渡す指示は両方を禁じている。定義の `memory: project` が書く `.claude/agent-memory/` は、root と `templates/base/` の `.gitignore`(44 行)が無視するので、`git status --porcelain` に出ず、全部を戻す判定を起こさない。
- probe(scratchpad の使い捨ての repo、確かめたあと消した): ステージした変更に `git restore <パス>` を打つと rc 0 で何も戻らず、そのあとのコミットに入る(L1)。
- テスト: `prompts_leader_syncdocs_test.go`(298 行、7 本)と `skill_doc_followup_test.go`(514 行、9 本と補助のテスト `TestSkillParagraphAfter` 1 本)。比べ方は `squashSpace`(`prompts_test.go:139-147`)と `skillOrderedFragmentProblems` で、空白と改行を除いて比べるので、折り返しを変えても通る。順は `assertPhrasesInOrder` が位置で見る。例の RESULT は `validateEscalation` に通し、`DOC_FOLLOWUP:` がヘッダでなく本文にあることも見る。
- 追加行の表示幅: 中央値は雛形 73、`/org` 74、`/director` 72。段落の途中の短い行は `leader.md:371` の 1 行だけで、所見にはしない。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1 sync-docs の変更で gate が落ちると、同じ `GATE: fail` が上限なしに繰り返されうる。やり直しでは手順 9 の終わりで sync-docs が新しい HEAD でもう一度走るが、doc-maintainer には前の gate が何で落ちたかを渡さない。doc-maintainer が自分で打つのは写しの検査だけで、gate が走らせる `run-static-verify.sh` と `run-test.sh` は打たない。文書の変更は写しの検査のほかでも gate を落とす(この repo なら、skill の文を実行時に読んで固定する Go のテスト)。implementer の直しが「sync-docs の変更を戻す」なら、次の sync-docs は同じずれを見つけて同じ変更を入れ直し、gate はまた同じ理由で落ちる。手順 12 の「2 回続けて通らなければ BLOCKED」は `HEAD:` の食い違い、`GATE: pass` の行のないファイル、同じ環境の `GATE: unrunnable` だけを数え、`GATE: fail` を数えない。1 回の往復は implementer・レビュー・sync-docs・数十分の gate で、座席は動いているので見張りの停滞の ALERT も出ない。 | `internal/org/prompts/leader.md:369-371`(doc-maintainer が打つ検査)、`:393-396`(やり直しで sync-docs が走り直す)、`:442-445`(上限の対象)、`:482-489`(`GATE: fail` の枝)、`:81-82`(ミッションの 5)。`.claude/skills/org/SKILL.md:1073-1076`、`:1139-1143`。`internal/org/skill_doc_followup_test.go:68`(SKILL.md を `os.ReadFile` で読む) | やり直しの sync-docs には、前の `GATE: fail` の `REASON:` と、`LOG:` が名指す sync-docs のコミットのファイルを渡し、そのファイルはもう直さずに follow-up に回すよう頼む。あわせて、sync-docs のコミットのファイルで落ちた `GATE: fail` が 2 回続いたら 3 回目の gate を頼まずに BLOCKED を上げる、と手順 12 の上限に足す。`/org` skill の 1 と 4 も同じに直し、2 つのテストファイルの固定を足す。 |
| MEDIUM | maintainability | M2 director が書く文書だけの機能の予約が、写しの組の一部しか覆わないことがある。leader の手順 9 は、写しの組の 1 つでも予約の外にあれば doc-maintainer に組のどれも直させず follow-up に書かせる(AC8)。follow-up の形は 1 件 1 行の「パスと、何が古いか」で、パスは 1 つ。director の手順 6 は `- Reserve:` に「一覧のファイル」を並べるので、`.claude/skills/x/SKILL.md` の 1 行だけなら、その 1 つだけを予約する。文書の org の implementer は scope の中のパスだけを `git add` するので残りの写しを直せず、gate の写しの検査(`check-skill-sync.sh`)が落ち、M1 の往復か BLOCKED になる。その org の sync-docs も同じ AC8 の決まりで組を直さず、同じ follow-up をまた書く。AC8 が生む follow-up にちょうど当たる形。 | `internal/org/prompts/leader.md:366-368`(組の扱い)、`:372-375`(1 件 1 行)、`:532`(`DOC_FOLLOWUP: <パス>: <ずれ>`)、`:267`(implementer は scope の中だけを `git add`)。`.claude/skills/director/SKILL.md:1101-1102`、`:1105-1108` | director の手順 6 に「写しの組のファイルの follow-up は、repo の写しの検査がそろえる組のファイル(`.claude/skills/<name>/` と `.agents/skills/<name>/`、`templates/base/` の写しなど)をすべて `- Reserve:` に入れる」と書く。または leader の渡す指示で、直さなかった組は写しを 1 つ 1 行ずつ follow-up に書くよう頼む。どちらかをテストで固定する。 |
| LOW | exception-handling | L1 全部を戻す手の `git restore <パス>` は、ステージされた変更を戻さない。`git restore` は index から作業ツリーへ戻すので、doc-maintainer が `git add` した変更には rc 0 のまま何もしない。渡す指示は「コミットしないこと」とだけ書き、ステージを禁じていない。そのあとの「残った変更と report を 1 つのコミット」は index をそのままコミットするので、予約の外のステージ済みの変更がコミットに入り、コミットのあとの `git status --porcelain` も空になる(新しいファイルなら ` D` が残って気づける)。reviewer は sync-docs の変更をレビューせず、gate は予約を見ないので、あとで止めるものがない。M1・M2 より起きにくいのは、doc-maintainer が予約の外を直し、しかもステージする、の 2 つが重なったときに限るため。 | `internal/org/prompts/leader.md:375-377`(禁じること)、`:381-383`(戻し方)、`:388-390`(コミット)。`.claude/skills/org/SKILL.md:1063-1065`。probe: `M  out.md` と `A  newout.md` に `git restore out.md`(rc 0)と `rm newout.md` を打ち、`git add in.md` のあとのコミットに `out.md` と `newout.md` が入った | 戻し方を `git restore --source=HEAD --staged --worktree -- <パス>`(追跡しているファイルとステージした新しいファイル)と、未追跡のファイルを消すことに分ける。渡す指示に「ステージしないこと」を足す。2 つのテストの固定(`prompts_leader_syncdocs_test.go:140`、`:209`、`skill_doc_followup_test.go:173-174`、`:340`)も合わせて直す。 |
| LOW | exception-handling | L2 「呼び出しが失敗した」ときの枝は、戻さずに手順 10 へ進む。すぐあとの「report を書かずに終わった」枝は、変更をすべて戻してから進む。Task が途中で落ちた、Codex の custom agent が途中で止まった、のように、ファイルを書いたあとで呼び出しが失敗したときは、leader はどちらの枝にも読める。前の枝を選ぶと、汚れた作業ツリーのまま手順 10 に進み、report のコミットに文書の変更が混ざるか、手順 12 の gate が汚れた作業ツリーとして断り、その枝が「誰が残したかわからない変更」として BLOCKED を上げる。 | `internal/org/prompts/leader.md:352-356`、`:357-361`、`:459-468`。`.claude/skills/org/SKILL.md:1037-1040`、`:1041-1045` | 2 つの枝を 1 つにする: 「定義がないときは飛ばす。呼び出しが失敗したか report を書かずに終わったときは、次の段落の戻し方で変更をすべて戻し、`git status --porcelain` が何も出さないことを確かめてから飛ばす」。テストの固定(`prompts_leader_syncdocs_test.go:163-164`、`:211`、`skill_doc_followup_test.go:131-132`、`:343`)も合わせる。 |
| LOW | maintainability | L3 全部を戻したときの記録が、コミットした report とも PR とも合わない。report は doc-maintainer が書いたままなので、戻した変更を「直した」と書いたままコミットされる。戻した変更の控えは leader のコンテキストにしかなく、手順 12 の gate(数十分)を挟んで手順 14 まで持ち越す。`HEAD:` と `BASE:` は gate の出力ファイルから読み直せるが、戻した変更は `git restore` のあとはどこにも残らないので、そのあいだに compaction や leader のやり直しがあると、PR の本文の節から落ちる。また「何が古いか」は戻す前の差分からしか読めないが、文は「戻し、控える」の順に書いている。予約の中の変更も戻すので、それが「予約外の文書の follow-up」の節に入る。 | `internal/org/prompts/leader.md:381-384`、`:395-396`、`:521-525`。`.claude/skills/org/SKILL.md:1062-1066` | 戻す前に `git diff -- <パス>` と report の項を読んで控える、と順を書く。控えは `.harness/state/sync-docs-{{ORG_ID}}.txt`(gate の出力と同じく git が無視する場所)か、sync-docs のコミットのメッセージ本文(`Undone: <パス>` の行)に書き、手順 14 はそこから読む。予約の中だったものは、節の中でそうと分かる書き方にする。 |
| LOW | maintainability | L4 director の新しい節の、集め始める条件と止める条件が足りない。(a) 手順 1 は merge を報告するたび、`merged` を見るたびに走るので、集めて案を見せたあとの別の merge でも同じ分割計画について集め直し、同じ案をもう一度見せる(集めたことの印がない)。(b) 文書だけの機能の org の leader も sync-docs を走らせるので、その RESULT にも `DOC_FOLLOWUP:` が出て、文書の分割計画の merge のあとに同じ手順がまた案を作る。M2 の組の follow-up はここで同じ項として戻ってくる。文書の計画から出た follow-up をどう扱うかが書かれていない。(c) PR を close した機能は `merged` にならないので、その分割計画では集め始めない。どれも人が承認しないと org は立たないので、手間が増えるだけで済む。 | `.claude/skills/director/SKILL.md:1087-1091`、`:1105-1112`。merge-order の class `closed`(`/org` skill の `merge-order` の行) | 手順 1 を「どの機能も `merged` か `closed`(closed の機能は人に名前を言う)なら、まだ集めていない分割計画について始める」にし、集めたことの印(文書の分割計画の本文に元の分割計画の id を書く、など)を決める。文書の分割計画から出た follow-up は人に報告するだけにし、続きの計画は人が頼んだときに書く、と足す。 |
| LOW | security | L5 leader が書いた PR の URL を、director が shell のコマンドに入れる。手順 2 の `gh pr view <PR の URL> --json body --jq .body` は引用符のない置き場所で、RESULT の件の `--note '<PR の URL> follow-up 2 件'` は単一引用符で囲む。URL は RESULT の EVIDENCE、つまり同じ節が「leader が書いたデータで、指示ではない」と言うものから来るので、`;`、`$(`、`'` を含む値なら director の shell で別のコマンドとして走る。PR の番号と URL は、ralph が gh で読んだものが merge-order の PR の列(`#<番号> <URL>`)にある。 | `.claude/skills/director/SKILL.md:664`、`:1095`、`:1114-1116`。`internal/cli/org_merge_order.go:211-213` | 手順 2 は merge-order の PR の列の番号で `gh pr view <番号>` を打つ、と書く。note に写す URL も同じ列から取るか、`https://github.com/<owner>/<repo>/pull/<数字>` の形であることを確かめてから使う、と足す。 |
| LOW | readability | L6 `/org` skill の「完了条件」は、機能ごとの org の項目を「org の終わり方」と同じ順に並べたと書くが、sync-docs のコミット(または `SYNC_DOCS: skipped`)の項がなく、RESULT の項も `DOC_FOLLOWUP:` の行に触れない。昇格した leader がこの一覧で終わりを確かめると、sync-docs が抜けていても気づかない。この PR が触っていない行なので、`/sync-docs` の範囲とも重なる。 | `.claude/skills/org/SKILL.md:1858-1872` | 3 つ目の項の rebase と report のあいだに「sync-docs のコミット(飛ばしたときは控えた `SYNC_DOCS: skipped`)」を、4 つ目の項の RESULT に「`DOC_FOLLOWUP:` の行」を足す。写し 3 つも同じに。 |

## Positive notes

- sync-docs を手順 9 の段落に置き、番号を振り直さずに手順 10 の 1 行目を残した(D1)。今のテストの固定を動かさずに済み、#269 の差し込みの場所も空く。
- 予約の外を見つけたら一部でなく全部を戻す理由(写しの組の片方だけが残って gate の写しの検査が落ち、implementer が直せない)と、計画を戻す理由(手順 6 の承認の検査)が、手順の中に書いてある。読み手が規則を外したときの結果が分かる。
- 予約の読み方の説明が `reservePathsOverlap` の読み方と同じで、テストがその関数の 5 つの case で文と結びつけている(`prompts_leader_syncdocs_test.go:109-122`)。
- `--note` を 1 行・500 字に収める判断(D4)は `ValidateInboxNote` と合い、テストが実際に note の例を通し、2 行の note が断られることも見る(`skill_doc_followup_test.go:423-428`)。
- テストは空白と改行を除いて比べ、順を位置で見る。雛形と skill が同じ名前(呼び方、定義と report のパス、節の名前、skip の行)を使うことを 1 本のテストで固定した(`skill_doc_followup_test.go:252-293`)。
- 件の本文と PR の本文をデータとして読む文を、新しい節にも置いた(`director/SKILL.md:1114-1116`)。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(この cycle では台帳に足していない。所見を直さずに残すなら、M1 と M2 は 1 行ずつ、LOW はまとめて 1 行にして `docs/tech-debt/README.md` に足す。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH は 0 件。M1 と M2 は、機能の org が先に進めなくなる形なので、PR の前に直すことを勧める)
- Follow-ups:
  - M1・M2 と L1・L2 は雛形と `/org`・`/director` skill の文の直しで、テストの固定も変わる。記録だけの直しの例外には入らないので、直したら `/self-review` から通し直す。
  - Codex の leader が herdr の pane の中で custom agent を呼べるかは、計画のとおり実機で確かめていない(known gap)。
  - 挙動を変える雛形と skill の変更だが、コミットは `feat:` 1 つと `docs:` で、release のノートの文は計画の Rollout にある。`/pr` で PR の本文に載せる。

## cycle 1 run 2(HEAD b95f8d8c)

- Date: 2026-10-30
- Scope: run 1 の report のコミット 117ab006 から HEAD b95f8d8c までの 5 コミット(c2370794 計画の書き換えと承認のし直し、dadd4daa F1、738e7bf8 F2、8556265e 仕様と README、b95f8d8c 計画の Progress)。14 ファイル、+1177/-398。run 1 の所見が直ったかと、直しで新しく入った問題を見た。見方は run 1 と同じ 6 項目。この節の所見の番号は R2-1〜R2-3。この節の `Merge:` の行が run 1 の行に代わる。

### Evidence reviewed(run 2)

- 計画の D2・D2c・D3・D4・AC6 の書き換えと Progress の F1・F2・8556265e の記録。digest は c8485493b418。
- `leader.md` の手順 9 の 3 段落(348〜423 行)、手順 12 の `GATE: fail` の枝(509〜518 行)、手順 14(548〜584 行)。`/org` skill の「org の終わり方」の 1・4・6(1032〜1225 行)と「完了条件」(1890〜1910 行)。`/director` skill の RESULT の箇条(655〜675 行)と「予約外の文書の follow-up」(1085〜1148 行)。
- 写し: `/org` と `/director` の 4 つの写しは `cmp` で一致した。
- probe(scratchpad の使い捨ての repo、確かめたあと消した):
  - `git log --format=%H --grep='^docs: sync docs for auth$'` は、件名が `docs: sync docs for auth-token` のコミットを拾わない。件名がその文のコミットは本文があっても拾う。件名が `fix: something` で、本文の 1 行がその文のコミットも拾う(R2-1)。
  - 同じコマンドは `grep.patternType` が `basic`・`extended`・`perl` と未設定のとき 1 件、`fixed` のとき 0 件を返した。`--basic-regexp` を付けると `fixed` でも 1 件になる(R2-1)。
  - `git show --name-only --format= <コミット>` は空行を出さず、ファイル名だけを出す。
  - `git restore --source=HEAD --staged --worktree -- <パス>` は `A `・`AM` を index と作業ツリーから消し、` M`・`M `・`MM`・` D`・`D ` を HEAD に戻した(rc 0)。`??` は rc 1。計画の F1 の記録と同じ。
- 機能の slug は `ValidateIdentifier`(`^[a-z][a-z0-9-]{0,29}$`、`internal/org/identifier.go:38`)を通るので、`--grep` の正規表現にも単一引用符にも特別な文字は入らない。
- 文書の分割計画の印 `元の分割計画: <id>` は、分割計画の読み手では機能の本文の行になる(`internal/org/split.go:530-536` の `read` は `- <key>:` の形でない行を本文にする)。digest もこの行を覆う(テスト `TestDirectorSkill_DocFollowupGatheringMarksAGatheredSplitPlanAndNeverGathersItAgain` が読み手と digest の両方を見る)。
- merge-order の PR の列は `#<番号> <URL>` で、gh が origin の repository の PR を返したときだけ埋まる(`internal/cli/org_merge_order.go:211-213`)。そのため列に番号があれば、同じディレクトリで打つ `gh pr view <番号>` は同じ repository を読む。`merged` は gh がないと分からないので、`merged` の行の PR の列は空にならない。
- テスト: `prompts_leader_syncdocs_test.go`(493 行)と `skill_doc_followup_test.go`(790 行)。新しい定数(コミットのメッセージ、2 つのコマンド、report の接頭辞、戻し方、`## 戻した変更`)を雛形と skill の両方の固定に使い、`TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate` が両方に同じ名前があることを見る。`TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezeFindsTheFixedMessage` は雛形の 2 つのコマンドを使い捨ての repo で走らせ、前方一致の slug(`a-b`)と origin の上のコミットを外すことを確かめる。消した文が戻らないことも、古い言い方の否定で固定した。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。

### run 1 の所見の状態

| ID | 状態 | 根拠 |
| --- | --- | --- |
| M1 | 直った | やり直しでは、これまでの sync-docs のコミットのファイルの和集合を git から引いて凍らせ(`leader.md:364-373`)、凍らせたファイルが変わったら全部を戻す(`:396-397`、`:410-412`)。手順 12 の `GATE: fail` の枝もそれを言う(`:516-518`)。和集合は回ごとに増えるだけなので、同じ変更の入れ直しも、別のファイルとのあいだの往復も、doc-maintainer が直せるファイルがなくなったところで止まる。`/org` skill は 1047〜1056 行。検索の確かさに R2-1 が残る。 |
| M2 | 直った | leader 側は、直さなかった写しの組の写しを 1 つ 1 行ずつ書かせる(`leader.md:380-384`)。director 側は、組のファイルをすべて `- Reserve:` に入れる(`director/SKILL.md:1127-1132`)。 |
| L1 | 直った | 戻し方は `git restore --source=HEAD --staged --worktree -- <パス>` と未追跡の削除(`leader.md:399-402`)、渡す指示に「ステージしないこと」(`:391`)。probe で 7 つの状態を確かめた。 |
| L2 | 直った | 呼び出しの失敗と report なしの終わりを 1 つの枝にし、report も含めて戻して `git status --porcelain` が空なのを確かめてから飛ばす(`leader.md:356-363`)。指し先の文に R2-3 が残る。 |
| L3 | 直った | 戻す前に `git diff HEAD -- <パス>` を読み(`leader.md:397-399`)、戻したものを report の `## 戻した変更` と follow-up の節に書き(`:402-406`)、手順 14 は最も新しい sync-docs のコミットの report を読み直す(`:551-557`)。飛ばしたときの記録は記憶のままで、R2-3 に書いた。 |
| L4 | 一部 | (b) 文書の分割計画の follow-up は報告だけにした(`director/SKILL.md:1107-1108`、`:1141-1143`)。(c) `merged` か `closed` で始める(`:1095-1101`)。(a) 集めて承認された分割計画は `splits/` の印で見分けられるが、人が要らないと言った分割計画と follow-up のなかった分割計画には残る記録がない(R2-2)。 |
| L5 | 直った | note の URL も `gh pr view` の番号も merge-order の PR の列から取り、列が `-` のときだけ形を確かめた本文の URL を使う(`director/SKILL.md:664-673`、`:1111-1115`)。 |
| L6 | 直った | 「完了条件」に sync-docs のコミットと `DOC_FOLLOWUP:` の行を足した(`org/SKILL.md:1897-1900`、`:1904-1907`)。 |

### Findings(run 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | R2-1 凍らせるファイルを探す `git log --grep='^docs: sync docs for <機能の slug>$'` は、利用者の git の設定と本文の行で結果が変わる。(1) `grep.patternType=fixed` の設定では `^` と `$` を文字として読むので 0 件になり、和集合が空のまま、凍らせずに sync-docs が走る。M1 の守りが黙って外れる。(2) `^` と `$` は行ごとに当たるので、本文の 1 行がその文のコミット(probe の `fix: something`)も拾う。そのファイルが凍り、それが最も新しければ、手順 14 が読み直す「そのコミットが入れた `docs/reports/sync-docs-` のファイル」がない。どちらも起きにくい(`fixed` を既定にする人は少なく、leader が本文にその文だけの行を書く手順はない)。テストの repo にはどちらの形もない。 | `internal/org/prompts/leader.md:366`、`:552-553`。`.claude/skills/org/SKILL.md:1049`。`internal/org/prompts_leader_syncdocs_test.go:274-334`。probe: `-c grep.patternType=fixed` で 0 件、`--basic-regexp` を足すと 1 件 | コマンドに `--basic-regexp` を足す(probe で `fixed` の設定より優先した)。本文の行まで外すなら、`git log --format='%H %s'` の出力から件名がメッセージと等しい行だけを残す形にする。テストに本文の行で当たるコミットを 1 つ足す。 |
| LOW | maintainability | R2-2 director が集め直さない分割計画のうち、印が残るのは集めて承認されたものだけ。印 `元の分割計画: <id>` は、人が「進める」と言ってから書く文書の分割計画の中にあるので、案を見せてから答えが来るまでのあいだにはまだない。「人が要らないと言った分割計画」と「follow-up がなかった分割計画」には、どこにも記録がない。director は merge を報告するたびに手順 1 を通るので、全機能が終わったそれらの分割計画について、そのたびに受信箱と PR を読み直す。compaction や headless の director の起動し直しのあとは、要らないと言われた案をもう一度人に見せうる。どれも人の承認がないと org は立たないので、手間が増えるだけで済む。 | `.claude/skills/director/SKILL.md:1102-1108`、`:1123-1124`、`:1136-1140` | 人の答えと「follow-up なし」を、`splits/` の分割計画として読まれない決まった置き場(台帳の下の別のファイルなど)に 1 行ずつ残し、手順 1 はそれも読む、と書く。置き場を足さないなら、手順 1 に「記録のない分割計画をもう一度集めたときは、前に人に見せたかを人に聞いてから案を見せる」と書く。 |
| LOW | readability | R2-3 呼び出しが失敗したときの枝(L2 の直し)は「次の段落の戻し方で sync-docs の変更を report も含めてすべて戻し」と書くが、次の段落の戻し方は「report のほかの sync-docs の変更をすべて戻す」で始まり、`git diff HEAD` を読んで控えることと、report に `## 戻した変更` の節を足すことで終わる。この枝では report も消すので、指し先の手順の範囲と最後の手が合わない。文のとおりに次の段落をなぞる leader は、report を残すか、消した report に節を書き足しうる。もう 1 つ、この枝の記録 `SYNC_DOCS: skipped (<理由>)` は記憶にしかない。前の回に sync-docs のコミットがあれば、手順 14 が git で引く最も新しいコミットは前の回のもので、git からは最後の回が飛ばされたと分からない。L3 の直しの理由(記憶でなく report から読む)が、この枝には届いていない。 | `internal/org/prompts/leader.md:356-363`、`:394-406`、`:419-423`、`:551-557`。`.claude/skills/org/SKILL.md:1040-1046` | 枝の中に戻し方を直接書く(「`git restore --source=HEAD --staged --worktree -- <パス>` と未追跡のファイルの削除で、report も含めて戻す。`## 戻した変更` は書かない」)。飛ばした記録は、gate の出力と同じく git が無視する `.harness/state/` の下のファイル(`sync-docs-{{ORG_ID}}.txt` など)に書き、手順 14 はそこから読む、とする。2 つのテストの `syncDocsNoReportSkip` の固定も合わせる。 |

### 所見にしなかったこと(run 2)

- 凍らせる和集合は、`GATE: fail` の原因が文書かどうかを見ずに広がる(D2c の「`LOG:` を読まない」)。そのため、やり直しで implementer のコードが前の回に直した文書をまた古くしても、その文書は直さずに follow-up になる。予約の中の文書が `## 予約外の文書の follow-up` の節に入ることもある。計画が決めた引き換えで、follow-up として PR と RESULT に残るので、所見にはしない。
- report を凍らせる一覧から外すので、gate が sync-docs の report そのもので落ちる形(markdown の lint など)は凍らせても止まらない。report は予約の外にあり implementer は直せないので、往復は implementer の BLOCKED で終わる。
- `gh pr view <番号>` は `-R` を付けないが、merge-order の PR の列は gh が origin の repository の PR を返したときだけ埋まるので、同じディレクトリで打つ限り同じ repository を読む。
- 新しいテストの git は、パッケージのほかの git のテスト(`runGit`、`gateCommit`)と同じく利用者の設定を引き継ぐ。R2-1 の `fixed` の設定の人の手元では、このテストが落ちて知らせる。

### Recommendation(run 2)

- Merge: yes(CRITICAL・HIGH・MEDIUM は 0 件。run 1 の 8 件のうち 7 件が直り、L4 は一部。新しい所見は LOW 3 件)
- Follow-ups:
  - R2-1〜R2-3 は雛形と skill の文の直しで、テストの固定も変わる。直すなら `/self-review` から通し直す。この cycle で直さずに残すなら、3 件をまとめて `docs/tech-debt/README.md` に 1 行足す(この run では台帳に書いていない)。
  - Codex の leader が pane の中で custom agent を呼べるかは、run 1 のとおり実機で確かめていない。
