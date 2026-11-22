# issue-depends-wait

- Status: Approved
- Approved: 2026-11-22 sha256:7b44cea2d6bc
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #364(issue に起票した機能の待ちが予約の重なりだけで決まり、Depends on の依存を待てない)
- Related issue: 364
- Type: feat
- Branch: feat/issue-depends-wait

## Objective

分割計画の機能を issue に起票するとき、印の行の `after:`(merge を待つブランチ)を、走っている org の予約との重なりに加えて `- Depends on:` の頼る先からも決める。頼る先がまだ立っていない機能(issue に起票した機能を含む)でも、そのブランチを前もって待てるようにし、頼る先が merge されたら待つ側が「着手できる」と出るようにする。director が印の行を手で書き換える場面をなくす。

## Scope

- director skill(`.claude/skills/director/SKILL.md` と 3 写し)の「起票と記録」の 1(待つブランチを決める): 予約の重なりの org のブランチに加えて、機能の `- Depends on:` の頼る先の機能ごとに、次のブランチを待つ。
  - 頼る先の機能の org が走っていれば、その org の `feature:` の行のブランチ(今と同じ読み方)
  - 走っていなければ、分割計画の頼る先の機能の `<Type>/<slug>`(`ralph org start --plan` が機能に作るブランチの名前)。起票した機能、まだ立てていない機能、どちらも同じ。待たないのは、頼る先の機能が merge 済みと確かめられたときだけ: `ralph org merge-order --all --json` でそのブランチが `merged` で、かつそのブランチの結びつき(`binding`)が次のどちらかのとき。(a) `split` がこの分割計画の id で、`feature` が頼る先の slug。(b) `feature` が頼る先の slug で、`issue`(#365 の `ralph org start --plan --issue` が記録する番号)が、頼る先の機能を起票した issue の番号(待つ側の本文の「頼られる issue: #<番号>」の行の番号)と同じ(頼る先を issue から着手すると、director は機能 1 つの分割計画を別の分割計画として書き直すので、`split` は元の id と違う)。ブランチの名前と `merged` だけでは数えない(過去の別の分割計画の同じ名前のブランチが merge 済みのことがあるため。codex の計画助言の指摘)。`feature` だけが頼る先の slug と同じで (a) も (b) も当たらないときは、黙って待たず、結びつきの `split` と `issue` と、この分割計画の id を人に伝えて、頼る先の機能の merge かを聞く。結びつきがない、または `feature` が違うときは、待つブランチに入れて follow-up を記録する(follow-up は記録より後の merge だけを数えるので、前の同じ名前の PR では `ready` にならない)
  - 重なる org のブランチと合わせ、同じブランチは 1 つにする
- 同じ節の 4(follow-up の記録): `ralph org followup add` はまだないブランチでも記録でき、merge-order は PR のないブランチの follow-up を `waiting` と出す(`internal/org/followup.go` の `validateFollowupBranch`、`internal/org/mergeorder_followup.go`)ことを書き、頼る先のまだないブランチも記録する
- 「merge を待つ issue」の 3(issue の機能に着手するとき): 機能 1 つの分割計画を書き直すときは、元の分割計画の機能の Type と slug を変えない(待つ側の issue の `after:` と follow-up がそのブランチを待っているため)。どうしても変えるとき(前の start が残した worktree の記録とぶつかるなど)は、start のあとに、古いブランチを待つ issue(`ralph org followup list` の AFTER_BRANCH と、印の行の `after:`)を探し、今ある書き換えの手順(印の行の `after:` を `gh issue edit` で書き換え、新しいブランチで follow-up を足す)で新しいブランチに移す、と書く
- 引き継ぎ(「merge を待つ issue」の冒頭)と「merge を待つ issue」の 2(着手を聞く前の判定): `after:` に、まだ PR も org もないブランチがあり得ることを書く(頼る先の機能がまだ立っていない)。そのブランチの follow-up は `waiting` のまま待つ。follow-up のない `after:` のブランチを「記録の前に merge 済み」と数える今の判定は、そのブランチが印の行の分割計画(`ralph-split: <分割計画の id>/<slug>` の id)の機能の `<Type>/<slug>` に当たるときは、上と同じく merge-order の JSON の結びつきが上の (a) か (b) に当たるときだけにし、`feature` だけが合うときは人に聞き、そうでなければ follow-up を記録して待つ
- 「起票と記録」の 2(issue の本文): 機能の `- Reserve:` と `- Depends on:` に加えて、機能の `- Type:`(分割計画になければ `feat`。`internal/org/split.go` の既定)を書く。あとで機能 1 つの分割計画を書き直すとき、本文から同じ Type を読めるようにするため(consult の指摘)
- 人に起票を伝えるとき、どのブランチを何のために待つか(重なり、依存)を言う
- skill の文を固定するテスト(`internal/org/skill_*_test.go` の「起票と記録」と「merge を待つ issue」を読むもの)を合わせる
- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` に 2026-11-22 の項目(起票の待ちに依存を足した)

## Non-goals

- follow-up を機能の slug で待つ形(`ralph org followup add` に slug の欄を足す)。ブランチの名前は分割計画の Type と slug で決まるので、ブランチで待てば足りる
- `ralph org start --plan` が依存の順を強制すること
- 依存の頼る先の機能が分割計画から消えたときに、待つ側の follow-up を自動で片付けること(人に伝える)

## Assumptions

- `ralph org start --plan` は機能のブランチを `<Type>/<slug>` で作る(`internal/org/feature.go` の `startFeatureInput`)。director が機能 1 つの分割計画を書き直しても、Type と slug を変えなければ同じブランチになる
- `ralph org followup add --after-branch` は、ブランチがまだなくても形だけを確かめて記録する。merge-order は PR のないブランチを `waiting` と出し、`ralph status` は台帳の `pr_merged` だけを見る
- #365(`ralph org start --plan --issue`)が先に merge される。同じ「merge を待つ issue」の 3 を触り、merge-order の JSON の `binding.issue` を足すので、その main を取り込んでから実装する
- 受け入れ条件 3(印の行を手で書き換えなくてよい)の例外として、頼る先の機能を書き直すときに slug を変える場合(前の start が残した worktree の記録とぶつかるなど、人が関わる経路)は、今ある書き換えの手順が残る。PR の Known gaps に書く

## Affected areas

- `.claude/skills/director/SKILL.md` と `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` の写し
- skill の文を固定するテスト(`internal/org/skill_*_test.go`)
- `docs/specs/2026-10-07-org-multi-org-director.md`
- 必要なら `ralph org followup add --help` の文(まだないブランチでも記録できる)

## Visual review

None (director skill の手順の文と、それを固定するテストだけを変える。部品の間のやり取りの順や、記録や状態の移り変わりは変えない)

## Design decisions

Critical forks: None

- 待つ単位はブランチのままにする。機能のブランチは分割計画の Type と slug で決まるので、まだ立っていない機能のブランチも前もって名指せる。follow-up に slug の欄を足す案は、記録の形と merge-order と `ralph status` の読みを変えるので採らない
- 書き直した分割計画で Type と slug を変えないことを手順にする。変えたときだけ、今ある書き換えの手順で待つ側を移す

## Acceptance criteria

- [x] AC1: director skill の「起票と記録」の 1 が、予約の重なりの org のブランチに加えて、`- Depends on:` の頼る先ごとに、走っている org のブランチか、走っていなければ分割計画の `<Type>/<slug>` を待つと言う(merge 済みは待たない)。テストで固定する
- [x] AC2: 同じ節の 4 と引き継ぎが、まだないブランチの follow-up を記録し、`waiting` のまま待つと言う
- [x] AC3: 「merge を待つ issue」の 3 が、書き直す分割計画で Type と slug を変えないこと、変えたときに古いブランチを待つ issue を新しいブランチに移すことを言う
- [x] AC4: issue #364 の例(api は locks に、jobs-close は dataform と jobs-ingest に頼り、頼る先も issue に起票した)で、待つ側の印の行の `after:` に頼る先の `<Type>/<slug>` が入り、頼る先の PR が merge されると待つ側の follow-up が `ready` になる流れを、skill の手順の文で追える(テストの fixture か、skill の例の文で示す)
- [x] AC5: `./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る。director skill の 4 写しが同じ
- [x] AC6: 「過去の別の分割計画の同じ名前のブランチは merge 済みで、今回の頼る先の機能はまだ立っていない」場合に、待つ側の `after:` がそのブランチを持ち続け、着手できると数えないことを、skill の手順の文が言う(merge-order の JSON の `binding` の `split`、`feature`、`issue` で確かめる手順。(a) か (b) に当たるときだけ merge 済みと数え、`feature` だけが合うときは人に聞く)。頼る先を issue から着手して merge した場合((b))は merge 済みと数えることも言う。テストで固定する。merge-order の JSON が `binding` の `split`、`feature`、`issue` を出すことを既存のテストで確かめる(`issue` は #365 で足す)
- [x] AC7: 「起票と記録」の 2 の issue の本文に `- Type:` を書くことを skill が言う(テストで固定)。「merge を待つ issue」の 3 は、書き直す分割計画の機能の Type を本文の `- Type:` から読む

## Implementation outline

1. S1 director skill と、それを固定するテストと仕様(AC1〜AC5) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill|DirectorSkill|DocsMatch|Followup' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Followup|Help|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC5。`followup add` がまだないブランチを受けること、merge-order が PR のないブランチを `waiting` と出すことをコードで確かめる
- Documentation drift to check: director skill の 4 写し、org skill の「merge を待つ issue」、仕様、`ralph org followup add --help`
- Evidence to capture: skill の該当の節の文(テストの出力)

## Test plan

- Unit tests: skill の文の固定(「起票と記録」の 1 と 4、「merge を待つ issue」の冒頭と 3)
- Integration tests: なし(コードの振る舞いは変えない)。merge-order が PR のないブランチの follow-up を `waiting` と出す既存のテストがあることを確かめ、なければ足す
- Regression tests: 予約の重なりだけで決まる今の待ちの文が残ること
- Edge cases: 頼る先がすでに merge 済み、頼る先が走っている org、頼る先が起票した機能、頼る先が 2 つ、重なりと依存が同じブランチを指す
- Evidence to capture: `go test` の出力、`./scripts/run-test.sh` の rc

## Risks and mitigations

- 頼る先の機能が別の分割計画で別の Type か slug で立てられると、待つ側の follow-up がいつまでも `waiting` のまま: 3 の手順で、書き直すときは Type と slug を変えない、変えたら待つ側を移す、と書く。director は進捗の報告のたびに `waiting` の follow-up を読む
- 頼る先の機能が人の判断で取りやめになると、待つ側が待ち続ける: 引き継ぎと進捗の報告で、PR も org もないブランチを待つ follow-up を人に伝える
- #365 と同じ節を触る: #365 の merge のあとに main を取り込んでから実装する

## Rollout or rollback notes

- skill の文だけの変更で、新しい binary は要らない。revert すると今の待ち方(予約の重なりだけ)に戻る
- release ノート: issue に起票した機能は、予約の重なりに加えて `- Depends on:` の頼る先のブランチも待つ。頼る先がまだ立っていなくても、分割計画の Type と slug のブランチを待つ

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件、計画に入れた。[HIGH] 1: 頼る先がまだ立っていなくても、過去の別の分割計画の同じ名前のブランチが merge 済みだと、merge-order の `merged` だけで待ちから外し、着手を早く聞くか、`after: none` で着手を聞かなくなる → 頼る先を merge 済みと数えるのは、merge-order の JSON の結びつきがこの分割計画のその機能を名指すときだけにした。引き継ぎと着手前の判定も同じ(Scope、AC6)。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-22、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない。consult(consult-plan-depends)は「直してから進める」で、2 点を入れてから承認した: 頼る先を issue から着手したときは結びつきの `split` が元の id と違うので、#365 の `binding.issue` と待つ側の「頼られる issue」の番号で確かめる((b))、`feature` だけが合うときは人に聞く。起票の本文に `- Type:` を書く(AC7)
- [x] Branch created
- [x] Implementation started
  - 4a5a57d1 で main(#365、9979bde6)を取り込んだ
  - S1 b2ae60a2: director skill の 4 写し(「起票と記録」の 1〜5 と #364 の例、「merge を待つ issue」の引き継ぎ・2・3)、org skill の 4 写し(merge 済みのブランチを follow-up にしない規則に依存の例外を 1 文)、仕様 FR-9 の 2026-11-22 の項目、`internal/org/skill_issue_depends_test.go`(6 本)、`skill_split_questions_test.go` の固定 5 か所。handoff からのずれ: 4 の重複の判定を NOTE の列まで見る(移し替えで古い印の行の件が残るため)、「記録の直前に merge 済み」にも (a)(b) の確かめを掛ける、着手の問いの理由を依存のときは「頼る先の機能を待つために」と言う。`followup add --help` は直す必要がなかった。merge-order の JSON の `binding` と PR のないブランチの `waiting` は既存のテストが固定していた。機械の自己チェックは 5 つとも clean
  - そのあと main(#360、2a069b50)を取り込んだ(衝突なし、slice の確かめのコマンドは通った)
- [x] Review artifact created
  - `docs/reports/self-review-2026-11-22-issue-depends-wait.md`(cycle 1): CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6、Merge: yes。M1: 「`feature` だけが合うときは人に聞く」は ralph が作るブランチでは常にこの分岐に入り(ブランチは `<Type>/<slug>`)、報告のたびに繰り返し、問いの形と答えの行き先がない。残りは文の言い直しの散在(L1)、例の読みにくさ(L2)、org skill の言い切り(L3)、移す手順の選びすぎ(L4)、引き継ぎの知らせ(L5)、テストの重複(L6)
  - 同じ report の Run 2(cycle 2、HEAD ff59a0e1): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4、Merge: yes。ff59a0e1(cross-review の 2 件の直し)の文同士に食い違いと繰り返しはなく、同じ秒の扱いは `ready` の比べ方と同じ向き。LOW は C2-L1(`/org` skill の 1 文に (c) がない)、C2-L2(時刻を読めないときの回復と伝える回数の限り)、C2-L3(引き継ぎの例外がそれの直す規則と離れている)、C2-L4(テストの重なる確かめとファイルの長さ)。cycle 2 は上限の最後の周なので直さず、台帳の行は report の Tech debt に案として書いた(台帳には未追記)
- [x] Verification artifact created
  - `docs/reports/verify-2026-11-22-issue-depends-wait.md`(cycle 1、HEAD 03ced266): pass。AC1〜AC7 は Met、静的検証は rc=0、4 写しは同じ。03ced266 で self-review の M1・L1・L2・L3・L5 が直った。M1 の直しで、`feature` だけが合うときの問いは起票の 1 で 1 回になり、follow-up のあるブランチは引き継ぎと 2 で聞かない(Scope の 4 つ目の項目からのずれ)。所見は LOW 2(V1・V2: 仕様の 2026-11-22 の項目が 03ced266 より前の文のまま。`/sync-docs` へ)、INFO 4
- [x] Test artifact created
  - `docs/reports/test-2026-11-22-issue-depends-wait.md`(cycle 1、HEAD 95d755b2): pass。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 0(1,233 s、46 本の shell のテストと 11 の Go パッケージ、`(cached)` 0)。足した 7 関数と固定を変えた 4 関数は `-v` で PASS、SKIP 0。skill の文の変異 8 個と merge-order の overlay の変異 2 個はすべて落ちた。残る穴は、director の実際の動きを追う run がないことと、別の言い方の問いを足しても落ちないこと
- [x] Sync-docs artifact created
  - `docs/reports/sync-docs-2026-11-22-issue-depends-wait.md`(cycle 1、1 コミット)。仕様 FR-9 の 2026-11-22 の項を `03ced266` の決めごとに合わせた(verify V1・V2: 問いは起票で 1 回、答えは `after:` に残り、follow-up のあるブランチは引き継ぎでも聞き直さず、聞くのは follow-up のないブランチだけ。書き換えは 2 つの場合)。台帳に新しい行 2 つ(561・562 行目。561 は self-review L4 と verify V3、562 は self-review L6 と test の Test gaps 2)。台帳で Trigger を満たす行は 0(verify の結論を、Trigger の全行の照合で確かめた)。README と AGENTS.md は起票の待ちを書いていないので直していない。`followup add --help` と `/org` の表は一般の規則のままにした(verify V5)。AC1〜AC7 に印を付けた(digest は前後とも `7b44cea2d6bc`)
- [ ] PR created
