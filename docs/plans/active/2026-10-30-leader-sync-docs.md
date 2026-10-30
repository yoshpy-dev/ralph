# leader-sync-docs

- Status: Approved
- Approved: 2026-10-30 sha256:8655b404a42a
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
- D2 コミットするのは leader: doc-maintainer は文書を直して sync-docs の report を書くだけで、コミットしない。leader は `git status --porcelain` で変わったパスを読む。予約の外のファイルか機能の計画が 1 つでも変わっていれば、report のほかの sync-docs の変更をすべて戻し(追跡しているファイルは `git restore`、新しいファイルは消す)、戻したものをすべて follow-up に足す。一部だけを戻すと、写し(skill の写しなど)の片方だけが変わり、`check-skill-sync.sh` のような検査が gate で落ち、implementer も予約の外を直せずに進めなくなるため(codex の計画助言 HIGH)。計画を戻すのは、per-plan では承認の検査が断るため。残った変更と report を 1 つのコミットにする。手順 9 で reviewer のレポートを leader がコミットするのと同じ形
- D2b doc-maintainer に渡す内容: 機能の計画のパス、比べる base(`origin/<既定のブランチ>`)、予約したパス、予約の中のファイルだけを直すこと、repo の検査がそろえる写しの組(skill の写しなど)は組のすべてが予約の中のときだけ直し、1 つでも外なら直さずに follow-up に書くこと、直したあと repo にある写しの検査(`scripts/check-skill-sync.sh`・`scripts/check-sync.sh` があれば)を打ち、落ちる自分の変更を戻して follow-up に回すこと、report を `docs/reports/sync-docs-<日付>-<機能の slug>.md` に書き、予約の外のずれを `## 予約外の文書の follow-up` の節に書くこと、insight event を付けないこと、コミットしないこと、計画を変えないこと、`ralph org` のコマンドを打たないこと(consult)
- D3 follow-up の置き場: doc-maintainer は予約の外のずれを sync-docs の report の `## 予約外の文書の follow-up` の節に書く。report は PR に入るので、leader が RESULT と PR の本文を書くときの元になり、あとから director も読める。RESULT には 1 件 1 行の `DOC_FOLLOWUP: <パス>: <ずれ>`(なければ `DOC_FOLLOWUP: なし`)を書き、2,000 字に収まらなければ件数と PR の本文を指す 1 行にする。PR の本文には `## 予約外の文書の follow-up` の節を置く
- D4 director の集め方: RESULT の件を resolve するとき、`DOC_FOLLOWUP:` の行を人に報告し、`--note` には件数と PR の URL だけを 1 行で書く(`ValidateInboxNote` は 1 行・500 字までなので、follow-up そのものは入れない。中身は RESULT の件と PR の本文に残る。codex の計画助言 MEDIUM)。分割計画の全機能が merge されたら(`ralph org merge-order --all` で全機能が `merged`)、`ralph org inbox --all` の RESULT の件と各 PR の本文から follow-up を集めて重複をまとめ、文書の機能 1 つの分割計画の案を人に見せる。人が承認したら、今までの「分割計画と機能ごとの org」の手順で org を立てる。follow-up がなければ何もしない
- D5 doc-maintainer がないとき: 飛ばして、RESULT に `SYNC_DOCS: skipped (<理由>)`、PR の本文の follow-up の節にも同じ理由を書く。leader が自分で文書を直すことはしない(leader のコンテキストを使わないという issue の決め)
- Critical forks: None(D1〜D5 は、雛形と issue の決めと今の手順の形と今のテストから決まる。consult は D1 の置き場所を手順 9 の末尾に直したうえで、ほかを妥当とした)

## Acceptance criteria

- [ ] AC1 leader の雛形の手順 9 の末尾(reviewer のレポートのコミット、fetch と rebase のあと)に sync-docs があり、手順 10 の org の report と計画の archive の前に入る。文書のコミットは手順 12 の gate の前に入る。手順 10 の 1 行目は今のまま `` `ralph org report`` で始まる
- [ ] AC2 leader は sync-docs を doc-maintainer の subagent に渡す(Claude Code は `Task(subagent_type="doc-maintainer")`、Codex は `.codex/agents/doc-maintainer.toml`)。渡す内容に D2b の項目(計画のパス、base、予約したパス、予約の中だけを直すこと、写しの組の扱い、写しの検査、report のパスと follow-up の節、insight event を付けないこと、コミットしないこと、計画を変えないこと、`ralph org` を打たないこと)がある。leader は変わったパスを確かめ、予約の外か計画の変更が 1 つでもあれば report のほかの変更をすべて戻して follow-up に足し、残りと report を 1 つのコミットにする。doc-maintainer の定義がなければ飛ばし、そのことを記録する
- [ ] AC8 写しの組の片方だけが予約の中にあるときは、doc-maintainer はどちらも直さずに follow-up に書き、leader は予約の外の変更を見つけたら一部ではなくすべてを戻す、と雛形に書いてある(gate の写しの検査を、予約の外を直せない implementer に回さないため)
- [ ] AC3 `/org` skill の「座席内 fan-out」が leader の subagent を認め、「org の終わり方」に sync-docs の段がある
- [ ] AC4 leader の雛形の RESULT の例に `DOC_FOLLOWUP:` の行があり、PR の本文に「予約外の文書の follow-up」の節を置くことが書いてある。`/org` skill の「org の終わり方」にも同じ欄がある
- [ ] AC5 `/director` skill に、RESULT の件の follow-up を拾うこと(resolve の `--note` は件数と PR の URL の 1 行、500 字まで)と、全機能の merge のあとに follow-up を集めて文書の機能の分割計画を人と書き、承認のあと org を立てる手順がある
- [ ] AC6 文書の変更で gate が落ちたとき(`check-sync.sh` など)は、今の `GATE: fail` のやり直しの手順に乗り、やり直しで手順 9 の末尾の sync-docs がもう一度走ることが書いてある
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

- doc-maintainer が予約の外を直す: leader が `git status --porcelain` で確かめ、1 つでもあれば report のほかの変更をすべて戻して follow-up に足す(D2)。予約の中かの読み方は ralph の予約と同じにする
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
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
