# director-split-questions

- Status: Approved
- Approved: 2026-11-01 sha256:c3e06a2ab308
- Owner: Claude Code
- Date: 2026-11-01
- Related request: https://github.com/yoshpy-dev/ralph/issues/271
- Related issue: 271
- Type: feat
- Branch: feat/director-split-questions

## Objective

director が分割計画を書く前に、`/spec` の手順 1〜3 と同じ深さで人と要件を詰め、
分割計画の承認と機能ごとの出力先(issue に起票か、今すぐ着手か)を選択肢で聞く
ようにする。走っている org と予約が重なる機能は issue に起票して follow-up に
記録し、先行する PR の merge のあとに着手してよいかを選択肢で聞く。

## Scope

- `/director` skill の「分割計画と機能ごとの org」に、分割計画の前の質問の段
  (`/spec` の手順 1〜3 の取り込み)を足す。機能をまたぐ分かれ目(機能の境目、
  共有のファイルをどの機能が予約するか、座席の driver と model、機能の順)を
  この段で人と決める。機能の内側の分かれ目は leader の `FORK` の行(#267)に
  任せる
- 分割計画の承認を選択肢(Approve / Needs changes)で聞く
- 機能ごとの出力先を、推奨つきの選択肢(issue に起票 / 今すぐ着手)で聞く。
  機能の `- Reserve:` が、`ralph status` の `reserved:` の行に出る走っている org
  の予約と重なれば「issue に起票」を推奨する。start が予約の重なりで断った
  ときも、その機能を「issue に起票」の推奨で聞き直す
- 起票の手順: issue の本文(印の行 `ralph-split: <id>/<slug> after: ...` を含む)を
  `.harness/state/` の下のファイルに書いて `gh issue create --body-file` で起票し、
  重なる org のブランチごとに `ralph org followup add` で記録する。記録が途中で
  止まったときの記録し直しと、新しい director の引き継ぎでの欠けの確かめも書く
- 「merge を待つ issue」の 2 で、着手できる follow-up の issue に着手してよいかを
  選択肢で聞く(着手する / 後に回す / 要らない)。着手するなら新しい分割計画を
  書いて承認を取り、start する
- 「予約外の文書の follow-up」の 1 で、起票した機能を「まだ merge されていない
  機能」に数えない
- headless の director(herdr の pane の claude)で AskUserQuestion が出せるかと、
  見張りの push の 1 行が開いた問いに何をするかを実機で確かめ、証跡に残し、
  その結果に合わせて skill の聞き方の決まりを書く。codex の director は番号の
  選択肢で聞く。結果が「出せない」か「push の 1 行が開いた問いに答えてしまう」
  なら、既存の「計画の承認」(#267 の `FORK` の問い)の AskUserQuestion の文も
  同じ決まりにそろえる
- `/org` skill の「merge を待つ issue」に、director が起票と記録をすることを
  指す 1 文を足す
- 上の文を固定する Go のテストと、変わる既存の固定の更新

## Non-goals

- ralph の Go のコード(新しい動詞、`ralph status` の出力、start の判定)を
  変えること。重なりの判定は director が `ralph status` を読んで行い、
  `ralph org start --plan` の断りが最後の守りのまま
- 見張りの push の 1 行が、開いた問いに入らないようにする直し(実機で問題が
  見つかれば tech-debt に記録する)
- headless の director の役割の雛形(`internal/org/prompts/director.md`)の
  変更。雛形は skill に従うと書いており、聞き方の決まりは skill に書く
- `/spec` skill と leader の雛形の変更
- 実機の director で分割計画から start までを通すこと
- #270 の残り(片付けで `mergedAt` を記録する直し)

## Assumptions

- AskUserQuestion は 1 回に 4 問まで、1 問に 2〜4 の選択肢で、「その他」は
  ツールが足す。`/spec` は 5 問ずつと書くが、director はツールの上限の 4 問に
  そろえる
- headless の claude の director は `-p` ではなく対話の claude として pane で
  動く(`internal/org/director_start.go` の起動の引数は permission の flag、
  `--model`、雛形のポインタ)。そのため AskUserQuestion の画面が pane に出る
  見込みだが、実機で確かめる
- codex には AskUserQuestion がなく、どの skill も番号の選択肢で聞くと
  書いている(skill の CLI の表)
- `ralph status` は走っている org ごとに `reserved:` の行を出す(#270)。
  機能のブランチは `ralph org status --org-id <org>` の `feature:` の行に出る
- director は gh を打ってよい(「してはいけないこと」は実装を禁じるが、ralph と
  gh の操作は許す)

## Affected areas

- `.claude/skills/director/SKILL.md` と `templates/base/.claude/skills/director/SKILL.md`
  (`.agents` の 2 つの写しは `scripts/sync-skills.sh` で作る)
- `.claude/skills/org/SKILL.md` と `templates/base/.claude/skills/org/SKILL.md`
  (同じく写しを作る)
- `internal/org/` の skill の文を固定するテスト(新しいファイル
  `internal/org/skill_split_questions_test.go` と、
  `skill_followup_test.go`・`skill_doc_followup_test.go` の固定の更新)
- `docs/evidence/director-ask-herdr-2026-11-01.md`(実機の確かめ)
- 必要なら `docs/tech-debt/README.md`(push の 1 行と開いた問い)

## Visual review

- Page: `.harness/state/plan-visual/director-split-questions.html`(かんたんな図 2 枚
  `overview`・`easy-wait`、折りたたみの中に `detail-overview`・`detail-flow`・
  `detail-livecheck`)
- Lint: `./scripts/plan-visual.sh lint` rc=0
- Self-check: 全体、2 枚のかんたんな図、3 枚の詳しい図を 1 枚ずつ撮って読んだ。
  詳細 3 の 2 つ目の箱の文のはみ出しと、詳細 2 の 5 の文が director の線に
  かかるのを直し、撮り直して読み直した

## Design decisions

- D1 重なりの判定: director が `ralph status` の `reserved:` の行と機能の
  `- Reserve:` を、`/org` skill の「担当範囲の予約」のパスの規則で比べる。
  判定の動詞は足さない。判定を誤って「今すぐ着手」を推奨しても、
  `ralph org start --plan` が重なりで断り、エラーが相手の org とパスを出すので、
  そこで「issue に起票」の推奨で聞き直す
- D2 待つブランチ: 重なる org の機能のブランチ(`ralph org status --org-id <org>`
  の `feature:` の行)を `--after-branch` に渡す。重なる org が 2 つ以上なら
  ブランチごとに 1 件記録する。待つブランチの全部を印の行(D3)に書き、
  着手を聞くのは、印の行のどのブランチにも同じ issue の `ready` の follow-up が
  あるときだけにする。印の行のブランチに follow-up がなければ(記録の途中で
  止まった)、先に記録してから待つ。そのブランチの PR がもう merge されていれば
  (`ralph org merge-order --all` の `merged`)記録せず、待ち終えたものとして
  数える。記録の前に `ralph org followup list` を読み、同じ issue とブランチの
  open の件があれば足さない(打ち直しで 2 件にしない)。重なる org に `feature:`
  の行がない(`start <task>`、昇格した leader、`--reserve` だけの org)なら待つ
  ブランチがないので記録せず、そのことを人に言い、どうするかを聞く
- D3 起票した機能の印: issue の本文と follow-up の note に、同じ印の行
  `ralph-split: <分割計画の id>/<slug> after: <ブランチ>,<ブランチ>` を書く
  (待つブランチがなければ `after: none`)。分割計画は承認の digest があるので
  書き換えない。印の行は issue にあるので、follow-up のない起票(重ならない機能を
  人が起票に選んだとき、待つブランチのない org と重なるとき)にも残る。
  「予約外の文書の follow-up」の 1 は、印の行のある issue(`gh issue list --state all`
  の本文の検索で探し、本文の印の行を読んで確かめる)の機能を数えない。新しい
  director の引き継ぎ(「merge を待つ issue」の冒頭)では、`followup list` に加えて
  open の印の行の issue を探し、`after:` のブランチの follow-up が欠けていれば
  D2 のとおり記録する(起票のあと記録の前に止まったとき)
- D4 後に回すときの記録: 既存の「merge を待つ issue」の 3 のとおり、
  `ralph org followup done <id> --note <決めたこと>` を打つ。GitHub の issue は
  open のまま残り、それが後に回した記録になる。open のままにすると merge の
  報告のたびに同じ問いが出るため
- D5 着手するときの分割計画: 元の分割計画の機能から start せず、issue の本文と
  今の既定のブランチから機能 1 つの分割計画を書き直し、承認を取ってから start
  する(issue の「着手なら分割計画を書いて start」)。元の計画のあとで既定の
  ブランチが進み、予約や本文が古くなっているため
- D6 headless の聞き方: S1 の実機の結果で決める。AskUserQuestion が pane に
  出て pane から答えられれば、headless の claude の director も AskUserQuestion
  で聞く。見張りの push の 1 行と Enter が開いた問いに答えてしまうなら、登録の
  wake に push がある director は AskUserQuestion を使わず、番号の選択肢を
  文で書いて聞く(既定の wake は pull)。出せなければ、headless の director は
  番号の選択肢で聞く
- Critical forks: なし(D1〜D6 は repo の決まりと issue の文から選んだ。理由は
  上と Progress checklist)

## Acceptance criteria

- [ ] AC1: director の skill の「分割計画と機能ごとの org」に、分割計画を書く前の
  質問の段がある。`/spec` の手順 1〜3(聞かずに依頼を理解する、推奨つきの決定木の
  質問、残った曖昧さの確認)を、AskUserQuestion(1 回に 4 問まで、推奨の案を
  先頭に)と Codex の番号の選択肢で行い、聞く前にコードと文書を調べることを
  書く。`/spec` の手順 6〜8(出力先・worktree・PR)は取り込まない。手順 4〜5
  (下書きと人への提示)は、分割計画の執筆と図解ページの承認が受け持つと書く。
  機能をまたぐ分かれ目をこの段で決め、機能の内側の分かれ目は leader の `FORK`
  に任せることを書く
- [ ] AC2: 分割計画の承認を AskUserQuestion(Approve / Needs changes。Codex は
  番号の選択肢)で聞き、Needs changes なら直して図解ページを作り直して聞き直し、
  Approve のあとにだけ digest の 2 行を書く
- [ ] AC3: 機能ごとの出力先(issue に起票 / 今すぐ着手)を推奨つきの選択肢で
  聞く。走っている org の `reserved:` の行と重なる機能は「issue に起票」を
  推奨する。start が予約の重なりで断ったら、「issue に起票」の推奨で聞き直す
- [ ] AC4: 起票の手順(本文のファイル、`gh issue create --body-file`、本文と note の
  印の行 `ralph-split: <id>/<slug> after: <ブランチ>,...`、重なる org のブランチ
  ごとの `ralph org followup add`、記録の前の `followup list` で重ねないこと、
  `feature:` の行のない org のとき、重ならない機能を人が起票に選んだとき)がある。
  新しい director の引き継ぎで、印の行の issue の follow-up の欠けを記録し直す
  手順がある
- [ ] AC5: 「merge を待つ issue」で、着手できる follow-up の issue に着手して
  よいかを AskUserQuestion で聞く(問いの文は「<ブランチ> の実装が終わりました。
  コンフリクト回避のために起票した #<番号> に着手してよいですか」、選択肢は
  着手する / 後に回す / 要らない)。印の行の `after:` のどのブランチにも同じ
  issue の `ready` の follow-up があるか、記録の前に merge 済みのときだけ聞き、
  欠けた follow-up は先に記録する。着手するなら機能 1 つの分割計画を書いて承認を取り start し、
  それ以外は done に記録する。merge の報告と進捗の報告のたびに確かめる
- [ ] AC6: headless の director(herdr の pane の claude)で、AskUserQuestion が
  出るか、pane から答えられるか、見張りの push の 1 行と Enter が開いた問いに
  何をするかを実機で確かめた証跡が `docs/evidence/` にあり、skill がその結果に
  合わせた聞き方を書く。codex の director は番号の選択肢で聞く
- [ ] AC7: 「予約外の文書の follow-up」の 1 が、起票した機能(印の行のある issue の
  機能。follow-up の有無によらない)を数えない
- [ ] AC8: 新しい固定のテストと既存のテスト(`go test ./internal/org/`)、
  `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` が通る

## Implementation outline

1. S1: 実機の確かめ(AC6 の証跡)。分離した herdr の server の pane で、
   scratch の project(git が追わない一時のディレクトリ)を cwd にし、headless の
   director と同じ permission の flag(`--permission-mode bypassPermissions --model <m>`)
   に、道具を絞る `--tools AskUserQuestion --strict-mcp-config` を足した対話の
   claude に AskUserQuestion を出させ、pane を読み、pane から答える。main の
   チェックアウトでは動かさない。
   もう一度問いを出させ、`director_wake.go` と同じ打ち方(`herdr pane send-text`
   で wake の 1 行、750 ms 待って `herdr pane send-keys Enter`)を送り、何が
   起きるかを読む。証跡を `docs/evidence/director-ask-herdr-2026-11-01.md` に書く
   — verify: `test -s docs/evidence/director-ask-herdr-2026-11-01.md && grep -q '^## 結果' docs/evidence/director-ask-herdr-2026-11-01.md`
2. S2: director の skill の分割計画の前の質問、承認の問い、出力先の問い、起票と
   記録(AC1〜AC4、AC6 の skill の文)。4 つの写しを作り、固定のテストを足す
   — verify: `./scripts/sync-skills.sh && CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill'`
3. S3: merge のあとの着手の問い、文書の follow-up の除外、`/org` skill の 1 文
   (AC5、AC7)。S1 の結果が D6 の 2 つ目か 3 つ目の枝なら、「計画の承認」の
   AskUserQuestion の文も同じ決まりにそろえる。既存の固定を更新する
   — verify: S2 と同じ

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、
  golangci-lint、check-sync、check-skill-sync、purity を含む)
- Spec compliance criteria to confirm: AC1〜AC8 と issue #271 の受け入れ条件 6 つ
- Documentation drift to check: `git grep -n "人と話して仕事を機能に分け"` で古い
  手順の文が残っていないこと。`docs/specs/2026-10-07-org-multi-org-director.md` の
  分割計画の下書きの記述と食い違わないこと(dated の段落は書き換えない)。
  README と AGENTS.md の director の説明
- Evidence to capture: `run-static-verify.sh` の出力、S1 の証跡

## Test plan

- Unit tests(skill の文、`checkDirectorSkillPlaces` の形):
  - 質問の段: `/spec` の手順 1〜3、AskUserQuestion、4 問まで、推奨の案を先頭、
    Codex の番号の選択肢、聞く前に調べる、手順 4〜8 を取り込まない、機能を
    またぐ分かれ目の 4 つ、内側は leader の `FORK`
  - 承認の問い: Approve / Needs changes、直して聞き直す、承認のあとにだけ 2 行
  - 出力先の問い: 2 つの選択肢、重なりなら issue の推奨、start の断りで聞き直す
  - 起票: `gh issue create --body-file`、`ralph org followup add`、note の形、
    ブランチごとに 1 件、`feature:` の行がない org
  - 着手の問い: 問いの文、3 つの選択肢、すべて `ready` になってから、着手なら
    分割計画の書き直しと承認、それ以外は done
  - 文書の follow-up の 1: 起票した機能を数えない
  - headless の聞き方(S1 の結果の文)
  - 節の見出しの順
- Integration tests: なし(skill の文の変更)
- Regression tests: `TestDirectorSkill_*`、`TestOrgSkill_*`、
  `skill_doc_followup_test.go` の既存の固定、`check-skill-sync.sh`、`check-sync.sh`
- Edge cases: 重なる org が 2 つ以上、重なる org に `feature:` の行がない、
  start が重なりで断る、同じ issue の follow-up の一部だけが `ready`、着手を聞く
  ときに新しい org と重なっている(後に回すを推奨)、人が会話にいない(問いは
  人が戻るまで待つ)。記録の途中で止まった(起票のあと記録の前、1 件目のあと
  2 件目の前)、止まっているあいだに待つブランチが merge された、打ち直しで同じ
  follow-up を足しかける、重ならない機能を人が起票に選んだ(`after: none`)。
  どれも skill の文の固定で、手順の文があることを確かめる
- Evidence to capture: `go test` の出力、`run-test.sh` の結果、tester の変異の結果

## Risks and mitigations

- director が重なりを見落とし「今すぐ着手」を推奨する: start が重なりで断るので
  org は立たない。断りを受けて「issue に起票」の推奨で聞き直す(D1)
- 見張りの push の 1 行と Enter が、開いた AskUserQuestion の推奨の案を選んで
  しまう: S1 で確かめ、そうなら push のある director は番号の選択肢を文で書く
  (D6)。直しは tech-debt に書く
- 起票した機能が merge-order に出ないため、文書の follow-up を集めない: note の
  印で数えない(D3、AC7)
- 同じ issue に重なる org が複数あり、1 つの merge で着手を聞いてしまう: 印の行の
  `after:` のどのブランチにも `ready` の follow-up があるときだけ聞く(D2)
- 記録の途中で止まり、記録できた分だけで「すべて ready」に見える(計画助言 HIGH):
  待つブランチの全部を印の行に書き、聞く前に欠けを記録し直す。起票のあと記録の
  前に止まった分は、新しい director の引き継ぎで印の行の issue から見つける(D2、D3)
- 重ならない機能を人が起票に選ぶと follow-up がなく、文書の follow-up の集め始めが
  止まる(計画助言 MEDIUM): 印の行を issue の本文に書き、集め始めの判定は issue の
  印の行で行う(D3)。GitHub の検索は本文の語を分けて当てるので、見つけた issue の
  本文の印の行を読んで確かめる
- skill が長くなり(今 1,231 行)、director の文脈を食う: 新しい文は手順と条件に
  絞り、`/spec` の手順は写さずに指す
- director が人の答えなしに起票する: 起票は人が「issue に起票」を選んだときだけ
  にし、選択肢の答えを人の同意として扱う

## Rollout or rollback notes

- skill は、新しい skill を読んだ director から効く。ralph の binary は変えない
- 戻すときはこの PR を revert する。台帳と設定の形は変えない
- release のノート: director が分割計画の前に推奨つきの選択肢で要件を詰め、承認と
  機能ごとの出力先を選択肢で聞く。走っている org と重なる機能は issue に起票して
  follow-up に記録し、merge のあとに着手を聞く

## Open questions

- なし(S1 の結果で D6 の枝を選ぶ)

## Progress checklist

- [x] Plan reviewed(consult-plan-splitq: 判定「直してから進める」。AC1 を issue の「手順 6〜8 は取り込まない」にそろえ 4〜5 は分割計画と図解の承認が受け持つと書いた。S1 は scratch の project で `--tools AskUserQuestion --strict-mcp-config` に絞り、main のチェックアウトでは動かさない。S1 の結果で既存の「計画の承認」の問いもそろえる枝を Scope と S3 に足した。D1〜D6 とスライスはそのまま。plan advisory: ran with codex(PATH の ralph 6.0.0 は `config` の動詞を持たず手順 11.a では「古い ralph」として飛ばすところ、この worktree の `go run ./cmd/ralph config get standard.plan_advisory` で true を読み直して回した。reviewer の model と effort は `ralph_setting` の既定 gpt-6-astra / xhigh。呼ぶ前後の git の状態は同じ)。所見 2 件をユーザーの指示どおり聞かずに反映した: [HIGH] ブランチごとの記録が途中で止まると、記録できた分だけで「すべて ready」に見える → D2 と D3 を直し、待つブランチの全部を issue の本文と note の印の行に書き、聞く前に欠けを記録し直し、引き継ぎで印の行の issue を確かめる。AC4・AC5 を直した。[MEDIUM] follow-up のない起票(重ならない機能、ブランチのない org)では文書の follow-up の集め始めが止まる → 印の行を issue の本文に置き、集め始めの判定を issue の印の行で行う。AC7 を直した)
- [x] Plan approved(自動承認(ユーザーの指示)。2026-11-01、digest sha256:c3e06a2ab308)
- [x] Branch created(ブランチは worktree の記録の feat/director-split-questions。`branch-name.sh from-plan` は issue 番号つきの feat/271/director-split-questions を出すが、どちらも正しい形なので記録のほうを使う。#267 と同じ)
- [x] Implementation started
  - S1 667e4d29(impl-271-s1): 証跡 `docs/evidence/director-ask-herdr-2026-11-01.md`。問いの画面は herdr の pane に出て、pane から答えられた(T1・T2)。見張りの push と同じ打ち方の 1 行と Enter は、開いた問いを閉じた。カーソルの下の選択肢(出した直後は推奨の 1 番)が選ばれ、自由記述の欄なら 1 行が答えになり、選択肢の上では 1 行は消えて director に届かない(T3〜T5、T5 は足した試し)。D6 の 2 つ目の枝: wake に push がある director は番号の選択肢を文で書く。pull だけなら AskUserQuestion。handoff からのずれ: herdr の server を `env -i` で立てた(この session の env を pane に渡さないため)、日本語の prompt を起動スクリプトで渡した、T4・T5 を足した。node が PATH になく、ユーザーの node の hook が pane の中で失敗した(非ブロック、結果に影響なし)。検証 `test -s … && grep -q '^## 結果' …` rc=0、strict scan clean、push 済み。自己点検: 誤字と写し間違いを直した(T2 の問いの番号、T3 の Tip の行)、ほか clean
  - S2 b0f0a8ad(impl-271-s2): 「## 分割計画と機能ごとの org」を冒頭の段落と `### 人への聞き方`・`### 分割計画の前の質問`・`### 分割計画の承認`・`### 機能ごとの出力先`・`### 起票と記録` にした。旧 1〜3 は承認の節、旧 4 は出力先の 3 へ。見出しを足すと `directorSkillSubsection` が最初の `###` で止まるので、`## ` の範囲を返す `directorSkillSection` を足し、`skill_doc_followup_test.go` の固定の範囲だけを差し替えた(断片は変えていない)。handoff にない足し: 番号の選択肢の返事を待つあいだの見張りの 1 行は人の答えではない、証跡のパスに「ralph の repository の」、承認の問いは推奨を付けず Approve を先頭(/plan と同じ)。検証 rc=0(sync-skills、check-skill-sync、check-sync、gofmt、vet、`go test -run Skill` PASS 69)、変異 19 個がすべて狙ったテストで落ちた(コミットしない写しで)、strict scan clean。自己点検 5 項目 clean。気づき: 起票した機能に `- Depends on:` で頼る機能の推奨の規則がない → S3 で足す
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
