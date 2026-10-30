# leader-sync-docs

- Status: Approved
- Approved: 2026-10-30 sha256:94dc29701d29
- Owner: Claude Code
- Date: 2026-10-30
- Related request: org: leader が sync-docs を行い、予約外の文書のずれを文書の org にまとめる(https://github.com/yoshpy-dev/ralph/issues/268)
- Related issue: 268
- Type: feat
- Branch: feat/leader-sync-docs

## Objective

機能の org の leader が、reviewer のレビューが通ったあと、report と計画の archive と gate の前に、doc-maintainer の subagent に sync-docs を任せる。文書の変更は機能の予約したパスの中だけにし、予約の外で見つけた文書のずれは「予約外の文書の follow-up」として RESULT と PR の本文に書く。director は follow-up を集め、全機能の merge のあとに、文書だけの機能 1 つの分割計画を人と書いて org を立てる。

## Scope

- leader の雛形(`internal/org/prompts/leader.md`)の「機能ごとの org」: 手順 9 の末尾(fetch と rebase のあと)に sync-docs を置く(doc-maintainer への渡し方、変わったパスの確かめ、予約の外の変更の戻し方、コミット、doc-maintainer がないときの扱い)。手順 14 の RESULT の例と PR の本文に「予約外の文書の follow-up」を足す。gate が文書の変更で落ちたときのやり直しを書く
- `/org` skill: 「座席内 fan-out」に leader の subagent(sync-docs の doc-maintainer)を足す。「org の終わり方」に sync-docs の段と follow-up の欄を足す。写し 3 つ
- `/director` skill: RESULT の件から follow-up を拾うこと、全機能の merge のあとに follow-up を本文にした文書の機能の分割計画を人と書いて org を立てる手順。写し 3 つ
- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 に、2026-10-30 に決めたことの注
- 文を固定するテスト(新しいファイル)

## Non-goals

- reviewer 座席の cross-review(#269。sync-docs の前に入る段で、この計画は場所を空けるだけ)
- 手順の番号の振り直し(雛形の中で 34 か所、テストで 23 か所が番号を参照している)
- ralph の binary の動き(Go の本番コード)の変更。変わるのは埋め込みの雛形の文とテストだけ
- 標準フローの `/sync-docs` skill と doc-maintainer の定義の変更
- sync-docs の変更を reviewer にもう一度レビューさせること(issue で決めた順。文書の検査は gate が走らせる)
- 文書の org を director が人に聞かずに立てること(分割計画は人と書き、人の承認を受ける)

## Assumptions

- 機能の worktree には、scaffold が置く `.claude/agents/doc-maintainer.md`(Codex では `.codex/agents/doc-maintainer.toml`)と `.claude/skills/sync-docs/SKILL.md` がある。古い scaffold で doc-maintainer がなければ、leader は sync-docs を飛ばし、そのことを RESULT と PR の本文に書く
- leader の座席(Claude Code)は `Task` で subagent を呼べる。subagent は leader の permission mode を引き継ぐ。Codex の leader が herdr の pane の中で custom agent を呼べるかは実機で確かめていない(リスク参照)
- タスクの `- 予約したパス:` の行が予約の一覧で、予約の中かどうかは ralph の予約と同じ読み方(ファイルはそのパス、ディレクトリはその下、`.` は repo 全体)で決める
- per-plan の機能の計画は承認のあと先頭の 2 行と checkbox の印のほかを変えられない(雛形の手順 6)

## Affected areas

- 雛形: `internal/org/prompts/leader.md`(「機能ごとの org」の前文の「予約の外に書く手順のもの」に sync-docs の report、手順 9 の末尾、手順 12 の `GATE: fail`、手順 14 の RESULT の例と PR の本文)
- skill: `.claude/skills/org/SKILL.md`(「座席内 fan-out」「org の終わり方」)、`.claude/skills/director/SKILL.md`(「件の処理」の RESULT、follow-up を集めて文書の org を立てる節)と、それぞれの写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- 仕様: `docs/specs/2026-10-07-org-multi-org-director.md`(FR-4 の注)
- README: `README.md` の `ralph org gate` の段落(reviewer のレビューのあとに leader がすることの並び)
- テスト: 新しい `internal/org/prompts_leader_syncdocs_test.go`(雛形)、新しい `internal/org/skill_doc_followup_test.go`(2 つの skill)。今の `prompts_test.go` などは変えずに通す

## Visual review

- Page: `.harness/state/plan-visual/leader-sync-docs.html`(やさしい図 `overview`・`easy-followup`、折りたたみの中に `detail-overview`・`detail-finish-order`・`detail-handoff`・`detail-followup`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: full、overview(`--scale 2`)、ほかの図を 1 枚ずつ撮って読んだ。直したのは、やさしい図 2 枚の見出しの折り返しと、overview の下の縁の切れ(撮る高さ)

## Design decisions

- D1 sync-docs の位置: 手順 9 の末尾(reviewer のレポートのコミット、fetch と rebase のあと)に段落として置き、番号を振り直さない。org の report と archive(手順 10)の前、手順 12 の gate の前になる。gate の `GATE: fail` と、implementer の手が要る断りは、今までどおり手順 8 の往復と手順 9 に戻るので、やり直しのたびに sync-docs がもう一度走る。手順 10 の先頭には置かない: 今のテスト(`internal/org/prompts_test.go` の 884〜889・923・963・977 行)が、手順 10 の 1 行目に `` `ralph org report`` が続くことを固定している(consult)。番号の振り直しは、雛形とテストの参照を 50 か所以上動かし、#269 もこの雛形を触るので避けた
- D2 コミットするのは leader: doc-maintainer は文書を直して sync-docs の report を書くだけで、コミットしない。leader は呼ぶ前に `git status --porcelain --untracked-files=all` が何も出さないことを確かめ(出れば手順 9 の最初の段落の分け方で扱い、呼ばない)、終わったら同じコマンドで変わったパスを 1 ファイルずつ読む(既定の出し方では新しいディレクトリが 1 行にまとまり、予約と照らせないため。cross-review cycle 1)。予約の外のファイルか機能の計画が 1 つでも変わっていれば、report のほかの sync-docs の変更をすべて戻し(追跡しているファイルとステージした新しいファイルは `git restore --source=HEAD --staged --worktree -- <パス>`、未追跡のファイルは消す。doc-maintainer にはステージさせない。self-review L1)、戻したものをすべて follow-up に足す。一部だけを戻すと、写し(skill の写しなど)の片方だけが変わり、`check-skill-sync.sh` のような検査が gate で落ち、implementer も予約の外を直せずに進めなくなるため(codex の計画助言 HIGH)。計画を戻すのは、per-plan では承認の検査が断るため。すべて戻したときは、戻したパスと中身の要点を report の `## 戻した変更` の節と follow-up の節に足す(report が「直した」と書いたまま残らず、控えがファイルに残る。self-review L3)。残った変更と report を、決まったメッセージ `docs: sync docs for <機能の slug>` の 1 つのコミットにする。手順 9 で reviewer のレポートを leader がコミットするのと同じ形
- D2b doc-maintainer に渡す内容: 機能の計画のパス、比べる base(`origin/<既定のブランチ>`)、予約したパス、予約の中のファイルだけを直すこと、repo の検査がそろえる写しの組(skill の写しなど)は組のすべてが予約の中のときだけ直し、1 つでも外なら直さずに follow-up に書くこと、直したあと repo にある写しの検査(`scripts/check-skill-sync.sh`・`scripts/check-sync.sh` があれば)を打ち、落ちる自分の変更を戻して follow-up に回すこと、report を `docs/reports/sync-docs-<日付>-<機能の slug>.md` に書き、予約の外のずれを `## 予約外の文書の follow-up` の節に書くこと、insight event を付けないこと、コミットしないこと、計画を変えないこと、`ralph org` のコマンドを打たないこと(consult)
- D2c やり直しの sync-docs で凍らせるファイル: ブランチにすでに sync-docs のコミットがあれば(`git log --format='%H %s' origin/<既定のブランチ>..HEAD` の件名を `docs: sync docs for <機能の slug>` と完全一致で比べて引く。`--grep` は利用者の `grep.patternType` で結果が変わり、本文の行にも当たるので使わない。self-review R2-1)、それらのコミットが変えたファイルの和集合(`git show --name-only`)を doc-maintainer に渡し、そのファイルはもう直さず、残るずれは follow-up に書かせる。sync-docs の変更で落ちた `GATE: fail` を、sync-docs が同じ変更を入れ直して繰り返さないため(self-review M1)。直前の 1 回分だけでは、別のファイルとのあいだで往復しうるので和集合にする。一覧は leader の記憶でなく git から引くので、compaction と rebase にも耐える(consult)。LOG を読んで原因を推し量ることはしない
- D3 follow-up の置き場: doc-maintainer は予約の外のずれを sync-docs の report の `## 予約外の文書の follow-up` の節に書く。直さなかった写しの組は、組の写しを 1 つ 1 行ずつ書く(self-review M2)。report は PR に入るので、leader が RESULT と PR の本文を書くときの元になり、あとから director も読める。RESULT には 1 件 1 行の `DOC_FOLLOWUP: <パス>: <ずれ>`(なければ `DOC_FOLLOWUP: なし`)を書き、2,000 字に収まらなければ件数と PR の本文を指す 1 行にする。PR の本文には `## 予約外の文書の follow-up` の節を置く
- D4 director の集め方: RESULT の件を resolve するとき、`DOC_FOLLOWUP:` の行を人に報告し、`--note` には件数と PR の URL だけを 1 行で書く(`ValidateInboxNote` は 1 行・500 字までなので、follow-up そのものは入れない。中身は RESULT の件と PR の本文に残る。codex の計画助言 MEDIUM)。分割計画の全機能が merge されたら(`ralph org merge-order --all` で全機能が `merged`)、`ralph org inbox --all` の RESULT の件と各 PR の本文から follow-up を集めて重複をまとめ、文書の機能 1 つの分割計画の案を人に見せる。人が承認したら、今までの「分割計画と機能ごとの org」の手順で org を立てる。follow-up がなければ何もしない。self-review の直しで足したこと: 写しの組の follow-up は組のすべてのファイルを `- Reserve:` に入れる(M2)。follow-up の文書を読むテストや文を固定するテストと fixture も探して `- Reserve:` に入れ、人に言う(予約の外のテストを直せず文書の org が gate を通れないため。cross-review cycle 2)。集め始めるのは分割計画のどの機能も `merged` か `closed` になったとき(closed は人に名前を言う)で、文書の分割計画の本文に元の分割計画の id を書いて集めた印にし、印のある分割計画と、人が要らないと言った分割計画は集め直さない。文書の分割計画から出た follow-up は人に報告するだけにし、続きの計画は人が頼んだときに書く(L4)。PR は merge-order の PR の列の番号で読む(L5)
- D5 doc-maintainer がないとき: 飛ばして、RESULT に `SYNC_DOCS: skipped (<理由>)`、PR の本文の follow-up の節にも同じ理由を書く。leader が自分で文書を直すことはしない(leader のコンテキストを使わないという issue の決め)
- Critical forks: None(D1〜D5 は、雛形と issue の決めと今の手順の形と今のテストから決まる。consult は D1 の置き場所を手順 9 の末尾に直したうえで、ほかを妥当とした)

## Acceptance criteria

- [ ] AC1 leader の雛形の手順 9 の末尾(reviewer のレポートのコミット、fetch と rebase のあと)に sync-docs があり、手順 10 の org の report と計画の archive の前に入る。文書のコミットは手順 12 の gate の前に入る。手順 10 の 1 行目は今のまま `` `ralph org report`` で始まる
- [ ] AC2 leader は sync-docs を doc-maintainer の subagent に渡す(Claude Code は `Task(subagent_type="doc-maintainer")`、Codex は `.codex/agents/doc-maintainer.toml`)。渡す内容に D2b の項目(計画のパス、base、予約したパス、予約の中だけを直すこと、写しの組の扱い、写しの検査、report のパスと follow-up の節、insight event を付けないこと、コミットしないこと、計画を変えないこと、`ralph org` を打たないこと)がある。leader は変わったパスを確かめ、予約の外か計画の変更が 1 つでもあれば report のほかの変更をすべて戻して follow-up に足し、残りと report を 1 つのコミットにする。doc-maintainer の定義がなければ飛ばし、そのことを記録する
- [ ] AC8 写しの組の片方だけが予約の中にあるときは、doc-maintainer はどちらも直さずに follow-up に書き、leader は予約の外の変更を見つけたら一部ではなくすべてを戻す、と雛形に書いてある(gate の写しの検査を、予約の外を直せない implementer に回さないため)
- [ ] AC3 `/org` skill の「座席内 fan-out」が leader の subagent を認め、「org の終わり方」に sync-docs の段がある
- [ ] AC4 leader の雛形の RESULT の例に `DOC_FOLLOWUP:` の行があり、PR の本文に「予約外の文書の follow-up」の節を置くことが書いてある。`/org` skill の「org の終わり方」にも同じ欄がある
- [ ] AC5 `/director` skill に、RESULT の件の follow-up を拾うこと(resolve の `--note` は件数と PR の URL の 1 行、500 字まで)と、全機能の merge のあとに follow-up を集めて文書の機能の分割計画を人と書き、承認のあと org を立てる手順がある
- [ ] AC6 文書の変更で gate が落ちたとき(`check-sync.sh` など)は、今の `GATE: fail` のやり直しの手順に乗り、やり直しで手順 9 の末尾の sync-docs がもう一度走ることが書いてある。そのとき、ブランチのこれまでの sync-docs のコミットが変えたファイルの和集合を git から引いて凍らせ、doc-maintainer にもう直させないことが書いてある(D2c)
- [ ] AC7 新しい文を固定するテストがあり、今の雛形と skill のテスト(`prompts_test.go`、`prompts_verification_test.go`、`prompts_leader_plan_test.go`、`director_skill_sync_test.go`、`org_skill_send_sync_test.go`)が変えずに通る。`check-skill-sync.sh` と `check-sync.sh` が通る

## Implementation outline

1. S1 leader の雛形: 「機能ごとの org」の前文、手順 9 の末尾の sync-docs(D1・D2・D2b・D5、AC8)、手順 12 の `GATE: fail` の一文(AC6)、手順 14 の RESULT の例と PR の本文(D3)。固定するテストを新しい `internal/org/prompts_leader_syncdocs_test.go` に書く — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt'`
2. S2 `/org` skill: 「座席内 fan-out」と「org の終わり方」(AC3・AC4)、写し 3 つ。固定するテストを新しい `internal/org/skill_doc_followup_test.go` に書く — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill'`
3. S3 `/director` skill と仕様と README: 「件の処理」の RESULT と、follow-up を集めて文書の org を立てる節(D4、AC5)、写し 3 つ、仕様 FR-4 の注、README の `ralph org gate` の段落の並びに sync-docs。固定するテストを S2 のファイルに足す — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill' && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`git diff --check`
- Spec compliance criteria to confirm: AC1〜AC8。特に AC1 の順(手順 9 のレポートのコミットと rebase → sync-docs → 手順 10 の report と archive → 11 の secret scan → 12 の gate)と、AC2・AC8 の予約の外の戻し方、per-plan で計画を変えないこと
- Documentation drift to check: leader の手順を写している文書(`/org` skill の「org の終わり方」、仕様 FR-4・FR-7 の注、README の org の節)、手順の番号を参照している文(番号は変えないので、ずれないこと)
- Evidence to capture: verify の report、static verify の出力

## Test plan

- Unit tests: 雛形の描画(`RenderRolePrompt` の leader、機能ごとの org)に、手順 9 の末尾の sync-docs の文、doc-maintainer の Claude と Codex の呼び方、D2b の渡す項目、写しの組の扱い、予約の外か計画の変更があればすべて戻すこと、コミット、飛ばすときの記録、`DOC_FOLLOWUP:` の例、PR の本文の節、`GATE: fail` の一文があること。順(sync-docs の文が手順 9 の rebase の文のあと、手順 10 の `` `ralph org report`` の前、gate の TASK の前)を位置で確かめる
- Integration tests: skill の文の固定(`/org` の fan-out と終わり方、`/director` の RESULT と follow-up の節)、写しの一致(`check-skill-sync.sh`、`check-sync.sh`)
- Regression tests: 今の雛形と skill のテストが変えずに通る(番号を振り直さないことの確かめ)。手順 9 と 12 の今の文(レポートのコミット、fetch と rebase、gate の TASK)が残っている
- Edge cases: doc-maintainer の定義がない、予約が `.`(repo 全体)、follow-up が 2,000 字に収まらない、gate のやり直しで sync-docs が 2 回目に何も変えない
- Evidence to capture: test の report、新しいテストが変更の前の雛形と skill で落ちること(`go test -overlay`)

## Risks and mitigations

- doc-maintainer が予約の外を直す: leader が `git status --porcelain --untracked-files=all` で確かめ、1 つでもあれば report のほかの変更をすべて戻して follow-up に足す(D2)。予約の中かの読み方は ralph の予約と同じにする
- 写しの組の片方だけが予約の中にある(codex の計画助言 HIGH): doc-maintainer はどちらも直さず follow-up に書き、自分で写しの検査を打つ(D2b)。leader の戻し方も一部ではなくすべてなので、片方だけ変わった状態で gate に進まない(AC8)
- per-plan で sync-docs が計画を書き換え、そのあとの implementer への TASK が承認の検査で断られる: 渡す内容で計画を変えないよう書き、leader が計画の変更を戻す(D2)
- sync-docs の変更が reviewer のレビューを通らない: issue で決めた順。gate が `run-static-verify.sh` と `check-sync.sh` などを走らせ、落ちたら `GATE: fail` のやり直しに乗る(AC6)
- Codex の leader が pane の中で custom agent を呼べない(未確認): 呼べなければ D5 と同じく飛ばして記録する、と雛形に書く。実機の確かめは known gap として PR に書く
- follow-up が RESULT の 2,000 字に収まらない: 件数と PR の本文を指す 1 行にする(D3)
- #269 が同じ雛形と skill を触る: #269 はこの PR の merge のあとに始める(lane A の順)。sync-docs の前に cross-review を入れる場所は、手順 10 の先頭の前(手順 9 のあと)になる

## Rollout or rollback notes

- 雛形は binary に埋め込まれるので、新しい binary で立てた leader から効く。走っている org の leader には効かない
- skill は `ralph upgrade` で配られる。release のノートに「機能の org の leader が gate の前に sync-docs を doc-maintainer に任せ、予約外の文書のずれを RESULT の `DOC_FOLLOWUP:` と PR の本文に書く。director はそれを集めて文書の org を人と立てる」を書く
- 戻すときはこの PR を revert する。台帳の形と ralph の動きは変えないので、走っている org への影響はない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult の判定は「直してから進める」。直したのは、sync-docs を手順 10 の先頭ではなく手順 9 の末尾に置くこと(今のテストが手順 10 の 1 行目を固定している、D1)、doc-maintainer に insight event を付けさせず、report のパスと follow-up の節を名指すこと(D2b)。plan advisory: ran with codex。HIGH 1 件(写しの組の片方だけが予約の中にあると、予約の外だけを戻したとき写しの検査が gate で落ち、implementer も直せない)は、doc-maintainer が組の片方だけを直さないことと、leader が予約の外を見つけたら一部ではなくすべてを戻すことで反映した(D2・D2b・AC8)。MEDIUM 1 件(follow-up を `resolve --note` に入れると、1 行・500 字の制約で resolve が断られる)は、`--note` を件数と PR の URL の 1 行にすることで反映した(D4・AC5))
- [x] Plan approved(自動承認。ユーザーの「最後まで止まらずに」の決め。digest 8655b404a42a)
- [x] Branch created
- [x] Implementation started
  - S1 2d480eb4(leader の雛形): 前文に sync-docs の report(列挙は `prompts_test.go:549` が 1 つの語句で固定しているので、直後の 1 文で足した)、手順 9 の末尾の 2 段落(doc-maintainer の呼び方と渡す内容、変わったパスの確かめとすべて戻すこと、1 つのコミット、飛ばしたときの `SYNC_DOCS: skipped`、やり直しで sync-docs がもう一度走ること)、手順 12 の `GATE: fail` の 1 文、手順 14 の PR の本文の節と RESULT の `DOC_FOLLOWUP:`。計画から決めたこと: 飛ばしたときは RESULT に `DOC_FOLLOWUP:` を書かず `SYNC_DOCS: skipped` を書く(確かめていないのに「なし」と書かない)。戻した変更は report に足さず、`HEAD:`・`BASE:` と同じく手順 14 のために控え、やり直しのたびに最後の回のものに置き換える。2 回目の sync-docs が何も変えず report も同じならコミットしない。新しいテスト 6 本(`internal/org/prompts_leader_syncdocs_test.go`)は前の雛形で落ちる。検証: gofmt・vet、`-run 'RenderRolePrompt'` が通過。機械的な確かめは 5 項目とも clean
  - S1 の implementer が気づいた範囲外のこと: sync-docs が gate を落とす変更をやり直しのたびに入れ直すと `GATE: fail` が繰り返され、手順 12 の「2 回続けたら BLOCKED」は `GATE: fail` を数えない。doc-maintainer が写しの検査を自分で打つので写しの検査ではまず起きない。self-review で見てもらう
  - S3 への申し送り(consult): director は `DOC_FOLLOWUP:` の行だけでなく `SYNC_DOCS: skipped` の行も拾う
  - S2 4a437059(`/org` skill): 「座席内 fan-out」に leader の doc-maintainer の子を足し、「org の終わり方」の 1 の末尾に sync-docs の 2 段落、4 の `GATE: fail` に 1 文、6 に PR の本文の節と `DOC_FOLLOWUP:`。写し 3 つは Edit と `sync-skills.sh` で作った。新しいテスト 6 本(`internal/org/skill_doc_followup_test.go`)は HEAD の skill で落ち、うち 1 本は skill と雛形の名前(呼び方、定義と report のパス、節の名前)が同じことを確かめる。検証: check-skill-sync・check-sync・gofmt・vet、`-run 'Skill'` が通過。機械的な確かめは 4 項目 clean、誤字 1 件を直した
  - S3 44275a04(`/director` skill、仕様、README、consult の一言): 「件の処理」の RESULT の箇条(follow-up と skip の行を人に報告し、`--note` は PR の URL と件数の 1 行)と新しい節「### 予約外の文書の follow-up」(全機能の merge のあとに `inbox --all` と `gh pr view` で集め、文書の機能 1 つ(`- Type: docs`)の分割計画を人と書き、承認のあと今の手順で立てる)、仕様 FR-4 の「2026-10-30 に決めたこと」、README の gate の段落に sync-docs。計画から決めたこと: `DOC_FOLLOWUP:` も `SYNC_DOCS:` もない RESULT(古い ralph の leader)も確かめていない機能として扱い、その予約は人と決める。consult の一言として、report を書かずに終わった doc-maintainer は飛ばしたものとして扱う 1 文を雛形と `/org` skill に足した(共通の定数で両方のテストに固定)。skill のテストは SKILL.md を実行時に読むので `-overlay` では差し替わらず、`git archive HEAD` の写しで新しいテスト 4 本が落ちることを確かめた。検証: handoff の一式(check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check`)が通過。機械的な確かめは 5 項目とも clean
  - self-review で見てもらうこと(consult): 文書だけの機能の org の leader も sync-docs を走らせるので、その RESULT にも `DOC_FOLLOWUP:` が出うる
  - self-review cycle 1(report は 117ab006、reviewer は opus): CRITICAL・HIGH は 0、MEDIUM 2(M1 sync-docs の変更で落ちた `GATE: fail` が繰り返されうる、M2 文書の org の予約が写しの組の一部しか覆わない)、LOW 6、Merge yes。consult(判定: 直してから進める)と相談し、すべてを直す。M1 は凍らせる範囲を「これまでの sync-docs のコミットすべてのファイルの和集合を git から引く」にし、コミットのメッセージを固定する(consult)。計画の本文 D2・D2c・D3・D4・AC6 を書き換え、digest を取り直して承認を付け直した(8655b404a42a → c8485493b418、自動承認)。図解は `detail-finish-order` だけを直し、lint exit 0、撮って読んだ。直しは F1(雛形と `/org` skill: M1・M2 の leader 側・L1・L2・L3・L6)と F2(`/director` skill: M2 の director 側・L4・L5)の 2 スライス
  - F1 dadd4daa(M1・M2 の leader 側・L1・L2・L3・L6): sync-docs のコミットのメッセージを `docs: sync docs for <機能の slug>` に固定し、やり直しでは `git log --grep` と `git show --name-only` でこれまでの sync-docs のコミットのファイルの和集合を引いて凍らせる(report は除く。除かないと同じ日のやり直しで report を書けない)。直さなかった写しの組は写しを 1 つ 1 行ずつ follow-up に書く。戻し方は `git restore --source=HEAD --staged --worktree -- <パス>` と未追跡の削除で、doc-maintainer にはステージさせない(使い捨ての repo で、`A `・`AM` は消え、`M `・`MM`・` M`・`D `・` D` は HEAD に戻り、`??` は rc 1 になることを確かめた)。呼び出しの失敗と report なしの終わりを 1 つの枝にし、すべて戻してから飛ばす。すべて戻したときは report の `## 戻した変更` と follow-up に書き、手順 14 は最新の sync-docs のコミットの report を読み直す。`/org` skill の完了条件に sync-docs と `DOC_FOLLOWUP:` を足した。雛形のコマンドを使い捨ての repo で走らせるテストを足した。文を変えた既存の固定 12 本を合わせた。検証: check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check` が通過。機械的な確かめは 4 項目 clean、不要な折り返しの変更を 1 件戻した
  - consult(F1 のコミットの前): 凍らせたファイルを doc-maintainer が直したときの見張りがない。戻す条件に「凍らせたファイル」を足す(F2 に入れた)
  - F2 738e7bf8(M2 の director 側・L4・L5・凍らせたファイルの見張り): 戻す条件に「凍らせたファイル」を足した(雛形と `/org` skill)。director は写しの組の follow-up で組のすべてを予約に入れ、分割計画のどの機能も `merged` か `closed` になったら集め始め(closed は名前を言い、その follow-up は集めない)、文書の機能の本文の先頭行 `元の分割計画: <id>` を印にして、印のある分割計画、follow-up のなかった分割計画、人が要らないと言った分割計画は集め直さない。文書の分割計画から出た follow-up は報告だけ。PR は merge-order の PR の列の番号で読み、note の URL も同じ列から作る(列が `-` なら本文の URL の形を確かめ、違えば件数だけ)。implementer はテストの追記を shell の heredoc で書き、U+FFFD がないことを確かめてから Edit を入れて hook を通した。検証: handoff の一式が通過、新しいテスト 8 本は HEAD の写しで落ちる。機械的な確かめは 5 項目とも clean
  - 8556265e(inline、2 ファイルの文の直し): F2 の報告の範囲外の所見。仕様 FR-4 の注に凍らせたファイルと戻した変更の書き方、集め始めの `merged` か `closed` と印を足し、README の gate の段落を「merged or closed」と凍らせたファイルに合わせた。`check-sync.sh`・`git diff --check`・`-run 'Skill|RenderRolePrompt'` が通過。機械的な確かめは 5 項目とも clean
  - self-review cycle 1 run 2(report は c2417dc9、HEAD b95f8d8c): CRITICAL・HIGH・MEDIUM は 0、LOW 3(R2-1〜R2-3)、Merge yes。run 1 の 8 件のうち 7 件が直り、L4 は一部(R2-2)。R2-1(`git log --grep` が `grep.patternType=fixed` で 0 件になり、本文の行にも当たる)と R2-3(失敗の枝の戻し方の指し先が合わず、最後の回で飛ばしたとき手順 14 が前の回の report を読む)は F3 で直し、R2-2(人の「要らない」と「なかった」の記録が残らない)は台帳に入れる。#264 の前例にならい、run 3 は回さずに verify へ進む
  - F3 7f33188b(R2-1・R2-3): 前の sync-docs のコミットは `git log --format='%H %s' … | awk 'substr($0, index($0, " ") + 1) == "docs: sync docs for <機能の slug>" { print $1 }'` で件名を完全一致で比べて引く(タブ区切りの形は件名にタブがあると誤る、と probe で確かめた)。probe: 新しい形は `grep.patternType` が未設定・basic・extended・fixed・perl のどれでも、`log.showSignature=true` でも 2 件を新しい順に出し、古い形は本文の行も拾い fixed で 0 件だった(awk は macOS の `/usr/bin/awk` だけで確かめた)。失敗の枝に自分の戻し方(report も含めて戻し、`## 戻した変更` は書かない)を書き、手順 14 と skill の 6 は最後の回で飛ばしたとき report を読まず控えた行を使う。テストは本文の行のコミット、件名にタブのあるコミット、`grep.patternType` を basic と fixed の両方で回す形に足した。検証: handoff の一式が通過、変えた固定は前の版で落ちる。機械的な確かめは 5 項目とも clean
  - 計画の D2c に古い `--grep` のコマンドが残っていたので、件名の完全一致に書き換え、digest を取り直して承認を付け直した(c8485493b418 → ab78e5f0645b、自動承認。図解はコマンドを描いていないので変えない)
  - 台帳に 2 行: R2-2 と leader の `SYNC_DOCS: skipped` の控えが記憶にしかないこと、`git show --name-only` が `core.quotePath` と `log.showSignature` で変わりうること(未確認)と awk を BSD の awk だけで確かめたこと
  - origin/main(0351800d、lane B の #290 org-state-dir-local)を 0dfc49f0 で merge した。重なった 11 ファイルに衝突はなく、check-skill-sync・check-sync・build・vet・`-run 'Skill|RenderRolePrompt'` が通過
  - verify cycle 1(report は ad820f13、HEAD 0dfc49f0): fail。V1(HIGH)`/org` skill の template の写しが meta-repo にしかない `scripts/check-sync.sh` を名指し、`check-template-purity.sh` が落ちる(CI の verify も落ちる。スライスの検証コマンドに purity を入れていなかった)。V2(LOW)`/org` skill の受信箱の文が「PR の URL を `--note` に書いて resolve」のまま。AC1〜AC6・AC8 は met、AC7 は /test 待ち
  - F4 d8e73710(V1・V2): 雛形と `/org` skill の写しの検査を「`scripts/check-skill-sync.sh` など」にし(配布される検査だけを名指す)、雛形全体に `check-sync.sh` が出ないことの否定のテストを足した(`internal/org/prompts/` は purity の対象外だが binary で配られる)。`/org` skill の受信箱の文を `/director` の RESULT の項に合わせた(note は merge-order の PR の列の URL と件数の 1 行、飛ばしたときはそのこと)。検証: purity・check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check`・`run-static-verify.sh` が通過。機械的な確かめは 5 項目とも clean。consult(進めてよい): この meta-repo では doc-maintainer が `check-sync.sh` を打たないことがあるが、半分だけ変えれば gate で落ちて凍結と follow-up の道に乗る
  - cross-review cycle 1(triage は da707a0d、HEAD e5f90219): codex の所見 1 件(P2)を ACTION_REQUIRED にした。既定の `git status --porcelain` は新しい未追跡のディレクトリを `?? docs/new/` の 1 行にまとめるので、予約との照らし合わせがファイルごとにできない。`cycle-count.json` を 2 に上げ(上限 2)、直して pipeline を回し直す
  - F5 4d0f70e4(cross-review #1): sync-docs の段落の `git status` 4 か所を `git status --porcelain --untracked-files=all` にそろえ、未追跡のファイルは一覧に出たものを 1 つずつ消す(ディレクトリごとは消さない。ignore されたファイルを消さないため)。probe: 短い形は `?? docs/` の 1 行で予約の中の新しいファイルを「外」と読む、長い形は 1 行ずつ出す、ignore は出ない、`status.showUntrackedFiles=no` では短い形が何も出さない。新しいテストは雛形のコマンドを使い捨ての repo で走らせる。検証: purity・check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check`・`run-static-verify.sh` が通過。機械的な確かめは 5 項目とも clean
  - F5b aec1381f(consult の勧め): doc-maintainer を呼ぶ前に `git status --porcelain --untracked-files=all` が何も出さないことを確かめ、出れば手順 9 の最初の段落と同じく振り分けて呼ばない、の 2 文を雛形と `/org` skill に足した(手順 9 の最初の確かめは短い形で、`status.showUntrackedFiles=no` では他人の未追跡ファイルが見えないため。最初の段落は `prompts_test.go` が固定するので変えない)。検証は F5 と同じ一式が通過。機械的な確かめは 5 項目とも clean。範囲外: `skill_doc_followup_test.go` の 1 つのテストのコメントに F5 の前の書き方が残る(cycle 2 の直しで扱う)
  - 計画の D2 と Risks の `git status --porcelain` を、呼ぶ前の空の確かめと `--untracked-files=all` に書き換え、digest を取り直して承認を付け直した(ab78e5f0645b → 220fa1f3994f、自動承認。図解は `git status --porcelain` のノードがあるが、意味は同じなので変えない)
  - self-review cycle 2(report は 8bd985fb、HEAD ef4a15a7): CRITICAL・HIGH・MEDIUM は 0、LOW 2、Merge yes。P2 は直った。空のディレクトリは gate・cleanup・`git worktree remove`・写しの検査のどれにも掛からない。C2-1(F5 の前の書き方がテストのコメントと台帳の行に残る)と C2-2(呼ぶ前の確かめの振り分けが per-plan の `PLAN:` のヘッダを落とす)は直す。台帳の行は 94c10923(inline、1 ファイル、機械的な確かめは 5 項目とも clean)、残りは F6
  - F6 41468881(C2-1 のテストのコメント、C2-2): 呼ぶ前の確かめの implementer の項に「(per-plan では `PLAN:` のヘッダを付ける)」を足し(雛形と `/org` skill、共通の定数 `syncDocsLeftoverSort` を伸ばした)、テストのコメントを今の文に合わせた。検証: purity・check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check`・`run-static-verify.sh` が通過、新しい固定は前の版で落ちる。機械的な確かめは 5 項目とも clean。#264 の前例にならい、cycle 2 の self-review の run 2 は回さずに verify へ進む
  - cycle 2 の verify(b0845cd3、pass、LOW 1 は仕様 FR-4 の注の古い書き方)、test(147e84bc、pass、`run-test.sh` rc 0、変異 21 個がすべて落ちる)、sync-docs(822acf62、仕様 FR-4 の注を直した)
  - cross-review cycle 2(triage は 308409a7、HEAD 822acf62): codex の所見 1 件(P2)を ACTION_REQUIRED にした。director が書く文書だけの機能の予約に、follow-up の文書の文を固定するテストが入らず、文書の org が gate を通れない。上限 2 に届いたので、ユーザーの決めどおり上限を 3 に上げて回し直す(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3`、`cycle-count.json` は 2 のまま)
  - F7 44694f5b(cross-review cycle 2 #1): `/director` skill の手順 6 に、案を見せる前に follow-up の文書を読むテストや文を固定するテストと fixture を探して `- Reserve:` に入れ、足したものと理由を人に言う 2 文を足した(この repo のテストの名前は書かない。否定の検査で固定)。検証: purity・check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check`・`run-static-verify.sh` が通過、新しい固定は前の版の 4 つの写しすべてで落ちる。機械的な確かめは 5 項目とも clean。計画の D4 に同じことを書き、digest を取り直して承認を付け直した(220fa1f3994f → 94dc29701d29、自動承認。図解は予約の中身を描いていないので変えない)
  - self-review cycle 3(report は eea3eb84、HEAD ebc2b64a): CRITICAL・HIGH・MEDIUM は 0、LOW 1、Merge yes。C2-1・C2-2・cross-review cycle 2 の P2 は直った。C3-1(F7 の「一覧のファイルを読むか、…テストと fixture を探し」は「文書を読む、またはテストを探す」とも読める)は F8 で直す。仕様 FR-4 の注に予約のテストを足すことは cycle 3 の sync-docs に渡す
  - F8 b6a8060a(inline、C3-1): `/director` skill の 4 つの写しの文を「一覧のファイルを読むテストと、その文を固定するテストと fixture を探し」にし、テストの固定を合わせた(5 ファイル、各 1 行。写しは `sync-skills.sh` で作った)。検証: purity・check-skill-sync・check-sync・gofmt・vet・`-run 'Skill|RenderRolePrompt'`・`git diff --check` が通過。機械的な確かめは 5 項目とも clean
  - 2026-10-30: 計画の Progress を書く途中でディスクが満杯(Data ボリュームの空き 944MiB、ENOSPC)になった。Go の build cache(38G)のうち 24 時間使われていない項目(26.4GB)を消し、空きは 30GiB に戻った。ほかのセッションの一時ファイル(stoic-app の 22G)には触れていない
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [x] Sync-docs report created(`docs/reports/sync-docs-2026-10-30-leader-sync-docs.md`、cycle 1、run 開始時の HEAD は a5e28500)。直したのは `/org` skill の 2 か所(座席の責務の 7 と「タスク終了時は必ず」の、機能の org の締めの順に sync-docs を足した)とその写し 3 つだけ。AGENTS.md の地図、quality-gates.md とその template の写し、definition-of-done.md、recipes、仕様の FR-7 の注、agent-messaging.md、subagent-policy.md、model-routing.md は見て、変えていない。AC の checkbox と計画の本文は触っていない。digest は前後で ab78e5f0645b
- [x] Sync-docs cycle 2(report の末尾の「cycle 2」の節、run 開始時の HEAD は 147e84bc)。verify の V3 を直した: 仕様 FR-4 の注(`docs/specs/2026-10-07-org-multi-org-director.md:93`)の `git status --porcelain` を、呼ぶ前の確かめと `--untracked-files=all` の形にした。F5・F5b・F6 の影響を README、AGENTS.md、`/director` skill、`/org` skill の要約 2 か所、品質の文書、rules で見て、ほかに古い文はなかった。計画の本文と digest(220fa1f3994f)は触っていない
- [ ] PR created
