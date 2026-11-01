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

- [x] AC1: director の skill の「分割計画と機能ごとの org」に、分割計画を書く前の
  質問の段がある。`/spec` の手順 1〜3(聞かずに依頼を理解する、推奨つきの決定木の
  質問、残った曖昧さの確認)を、AskUserQuestion(1 回に 4 問まで、推奨の案を
  先頭に)と Codex の番号の選択肢で行い、聞く前にコードと文書を調べることを
  書く。`/spec` の手順 6〜8(出力先・worktree・PR)は取り込まない。手順 4〜5
  (下書きと人への提示)は、分割計画の執筆と図解ページの承認が受け持つと書く。
  機能をまたぐ分かれ目をこの段で決め、機能の内側の分かれ目は leader の `FORK`
  に任せることを書く
- [x] AC2: 分割計画の承認を AskUserQuestion(Approve / Needs changes。Codex は
  番号の選択肢)で聞き、Needs changes なら直して図解ページを作り直して聞き直し、
  Approve のあとにだけ digest の 2 行を書く
- [x] AC3: 機能ごとの出力先(issue に起票 / 今すぐ着手)を推奨つきの選択肢で
  聞く。走っている org の `reserved:` の行と重なる機能は「issue に起票」を
  推奨する。start が予約の重なりで断ったら、「issue に起票」の推奨で聞き直す
- [x] AC4: 起票の手順(本文のファイル、`gh issue create --body-file`、本文と note の
  印の行 `ralph-split: <id>/<slug> after: <ブランチ>,...`、重なる org のブランチ
  ごとの `ralph org followup add`、記録の前の `followup list` で重ねないこと、
  `feature:` の行のない org のとき、重ならない機能を人が起票に選んだとき)がある。
  新しい director の引き継ぎで、印の行の issue の follow-up の欠けを記録し直す
  手順がある
- [x] AC5: 「merge を待つ issue」で、着手できる follow-up の issue に着手して
  よいかを AskUserQuestion で聞く(問いの文は「<ブランチ> の実装が終わりました。
  コンフリクト回避のために起票した #<番号> に着手してよいですか」、選択肢は
  着手する / 後に回す / 要らない)。印の行の `after:` のどのブランチにも同じ
  issue の `ready` の follow-up があるか、記録の前に merge 済みのときだけ聞き、
  欠けた follow-up は先に記録する。着手するなら機能 1 つの分割計画を書いて承認を取り start し、
  それ以外は done に記録する。merge の報告と進捗の報告のたびに確かめる
- [x] AC6: headless の director(herdr の pane の claude)で、AskUserQuestion が
  出るか、pane から答えられるか、見張りの push の 1 行と Enter が開いた問いに
  何をするかを実機で確かめた証跡が `docs/evidence/` にあり、skill がその結果に
  合わせた聞き方を書く。codex の director は番号の選択肢で聞く
- [x] AC7: 「予約外の文書の follow-up」の 1 が、起票した機能(印の行のある issue の
  機能。follow-up の有無によらない)を数えない
- [x] AC8: 新しい固定のテストと既存のテスト(`go test ./internal/org/`)、
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
  - S3 3f0b69fc(impl-271-s3): 「merge を待つ issue」の引き継ぎ(`gh issue list --state open --limit 1000 --json number,body --jq` で行頭の印の行の issue を探し、`followup list --all` にない待つブランチを記録し直す。done の件は数えて人の決定を覆さない)、2 の待ちの判定と着手の問い(推奨は着手する、予約が今の org と重なれば後に回す)、3 の done と書き直しと承認と start、「計画の承認」の `FORK` の問いの push の決まり、「予約外の文書の follow-up」の 1 で印の行の issue の機能を数えない、`/org` skill の 1 文、`- Depends on:` で起票した機能に頼る機能の起票の推奨(推奨は出力先の 2、`after: none` と頼る先の issue の番号は起票と記録の 1・2)。handoff からのずれ: 印の行のない issue の扱いを足した(open の follow-up がすべて ready で聞き、問いは「後に回した」)、着手で start が重なりで断ったら同じ issue を起票し直さず人に聞く、人が会話にいないときは問いを待ち follow-up を open のまま残す、印の行の正規表現を Go の regexp で確かめるチェック。検証 rc=0(`go test -run Skill` ok、check-skill-sync、check-sync、purity)、変異 29 個がすべて狙ったテストで落ちた、strict scan clean。自己点検: 誤字を直した(テストの doc comment)、ほか clean。consult(S3 の implementer が聞いた)の注意 1 点が入っていない: 着手で follow-up を done にしたあと start が重なりで断られ、人が待つと答えると待つ記録がない → S4 で足す
  - S4 379fda6a(impl-271-s4、sonnet): 「merge を待つ issue」の 3 に、着手のあと start が重なりで断られ人が待つと答えたら、相手の org の `feature:` のブランチを待つブランチにし、issue の印の行の `after:` を書き換え(`gh issue edit <番号> --body-file <ファイル>`)、「起票と記録」の 4 のとおり記録し直す(done は戻さない、`feature:` の行がなければ人に言う)文を足し、`TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` に断片 3 つ。検証 rc=0、変異 3 つが落ちた、strict scan clean、自己点検 5 項目 clean
- [x] Review artifact created(self-review Run 1 28d34ed2: CRITICAL 0、HIGH 0、MEDIUM 1、LOW 8、Merge: yes。M1 は直す: issue の本文から読んだ印の行とブランチを shell のコマンドに入れる(公開の repo で誰でも書ける)。LOW も 8 件とも実害があり安いので直す)
  - S5 c72901e2(impl-271-s5): M1 は「起票と記録」の「印の行の形と読み方」に正規表現(id は `splitIDPattern`、slug は `identifierPattern` を 20 字で切った形、ブランチは `[A-Za-z0-9._][A-Za-z0-9._/-]*`)と、本文から読むときの 2 条件(印の行がちょうど 1 行で形に当たる、書いた人が `gh api user --jq .login` と同じ)を書き、そろわない issue は使わず番号と理由を人に言う、note は確かめた値から組み直す、値は単一引用符で囲む、「してはいけないこと」に 1 項。L1(待つブランチがすべて merge 済みならすぐ着手を聞く)、L2(頼る機能/頼られる機能)、L3(本文を `gh issue view` で読み `after:` だけを置き換える)、L4(起票した機能は文書の follow-up の 2〜6 でも読まない、7 の参照先)、L5(`push,pull` も)、L6(tech-debt に 1 行)、L7(2 の文の順)、L8(helper を移し、否定の固定を skill 全体に)。検証 rc=0、変異 16 個がすべて落ちた、strict scan clean。consult(S5 の implementer が聞いた)の注意: director の gh のアカウントが変わると自分の issue も「ほかの人の issue」になる → PR の Known gaps。書き換えで確かめに通らないときは「印の行の形と読み方」の「使わず人に言う」で覆われている
  - self-review Run 2 89b50e8f: CRITICAL・HIGH・MEDIUM 0、LOW 2(R2-1 起票の本文のほかの行の行頭と行末の CR、R2-2 小さな文の直し 5 つ)、Merge: yes。M1 と L1〜L8 は直っていた
  - S6 ec988003(impl-271-s6、sonnet): R2-1(ほかの行の行頭を `ralph-split: ` にしない、行末の CR は取ってから確かめる)、R2-2 の (a)〜(e)、S5 の consult の注意 2(3 の書き換えで確かめに通らなければ書き換えず記録もせず人に言う)。検証 rc=0、変異 7 つが落ちた、strict scan clean、自己点検 5 項目 clean
  - self-review Run 3 5b65f5a1(S6 の差分だけ): CRITICAL・HIGH・MEDIUM 0、LOW 3(R3-1 `/org` の文の読み違え、R3-2 テストの comment の折り返し、R3-3 書き換えのあとの note の印の行が古い `after:` のまま)、Merge: yes
  - S7 a14d7169(impl-271-s7、sonnet): R3-1〜R3-3 を直した。検証 rc=0、変異 2 つが落ちた、strict scan clean、自己点検 5 項目 clean。S7 のあとは self-review を回し直さない: 直すたびに LOW が出る流れで、S7 は文 2 か所と comment の折り返しだけ。S7 の差分は cross-review がブランチ全体として見る(PR の本文に書く)。Run 3 の注意「確かめに通らないとき、人が待つと答えた約束が台帳に残らない」は PR の Known gaps に書く
  - cross-review cycle 1(codex、101e7552 の triage): ACTION_REQUIRED 2(#1 引き継ぎで待つブランチがすべて merge 済みだと issue が浮かばない、#2 start の断りは最初の重なりだけを出すので `after:` がほかのブランチを落とす)、WORTH_CONSIDERING 1(#3 `gh issue list --limit 1000` が切れると印の行がないと決める)。ユーザーの指示どおり聞かずに 3 件とも直し、cycle-count を 2 にした。手順 0 は PATH の ralph 6.0.0 に `config` がないので `go run ./cmd/ralph config get` で true を読み直した(triage report に記録)
  - S9 9248d73b(impl-271-s9): #1 引き継ぎですべて merge 済みで follow-up が 1 件もなければすぐ着手を聞く、#2 待つと答えたら `ralph status` を読み直して重なる org を全部数え全部を `after:` と記録に、#3 `--jq` の 1 行目に件数を出し、上限に届いたら切れたとみなす(gh 2.102.0 で形を確かめた)。検証 rc=0、変異 13 個が落ちた
  - S10 e95f19e3(impl-271-s10、sonnet): S9 が consult の返事の前に入れた「もう重ならなければ、書き換えずに start を打ち直す」を「書き換えず、そのことを人に伝えて start してよいかを聞く」に替えた(人に聞かないまま着手しない決まりとぶつかるため)。検証 rc=0、変異 1 つが落ちた
  - self-review cycle 2 25731fdc(S9・S10 の差分): CRITICAL・HIGH 0、MEDIUM 1(C2-1 follow-up のない起票に人が答えても台帳に残らず、引き継ぎがまた聞く)、LOW 5、Merge: yes。cross-review cycle 1 の #1〜#3 は直っていた
  - S11 80cb8b6b(impl-271-s11): C2-1 は「merge を待つ issue」の 3 に、follow-up のない issue に人が答えたら `after:` のブランチの 1 つで `followup add` を打ちすぐ `followup done` を打つ段落を足し、引き継ぎと「起票と記録」の 4 から指す(答える前は記録しない)。`followup add` が merge 済みのブランチを断らないことをコードと scratch の台帳で確かめた。C2-2〜C2-6 も直した。検証 rc=0、変異 9 個が落ちた、strict scan clean。consult(S11 の implementer が聞いた)の注意「add のあと done の前に止まると、merge 済みのブランチを待つ open の件が黙って残る」は確率が低いので PR の Known gaps に書く(LOW の直しが続く流れを止める)
  - cross-review cycle 2(codex、6f62affb の triage、Cycle 2/2 cap reached): ACTION_REQUIRED 1(#1 起票した機能に頼る機能が、頼る先が未実装でも merge のあとに「着手する」を推奨される)、WORTH_CONSIDERING 2(#2 書き換えのあと中断すると古い done の件で問いが出ない、#3 done を打つ途中で中断すると戻れない)。ユーザーの指示どおり上限を 3 に上げ(`cycle-count.json` に `cap: 3`、`cycle` は 2 のまま。ユーザーの「export RALPH_STANDARD_MAX_PIPELINE_CYCLES=3」は #276 で env が撤去されたのでこの書き込みに読み替えた)、3 件とも直して全 pipeline を回し直す(3 回目の run)
  - S12 e775792d(impl-271-s12): #1 2 の推奨に `- Depends on:` の頼られる issue が OPEN なら後に回す(番号は数字だけを使う)、#2 引き継ぎで数える follow-up を今の印の行の `after:` のブランチの件に限る、#3 done と open が混ざる issue は DONE_NOTE を答えとして残りを done にする(着手なら org が走っているかを確かめ、なければ人に聞く)。検証 rc=0、変異 7 つが落ちた、strict scan clean、自己点検 5 項目 clean
  - self-review 3 回目の run 10ea72c6(S12 の差分): CRITICAL・HIGH 0、MEDIUM 1(C3-1 頼られる issue の番号を書く形が決まっていない)、LOW 5、Merge: yes。C3-2(open は未実装を表さない)と C3-5(すべて done にしたあと start の前に director が替わる窓)は設計が要るので PR の Known gaps に書く
  - S13 8a23af28(impl-271-s13): C3-1 は「起票と記録」の 2 に `頼られる issue: #<番号>` の行を書き、「merge を待つ issue」の 2 は `^頼られる issue: #[0-9]+$` に当たる行の番号だけを使う(行末の CR は取る)。C3-3 は「今の `after:` のブランチの follow-up が 1 件もないとき」、C3-4 は DONE_NOTE を単一引用符で渡し `'` を含めば自分の言葉で書く、C3-6 は tech-debt の行数を「over 800 lines」に。検証 rc=0、変異 11 個が落ちた、strict scan clean、自己点検 5 項目 clean。3 の例 `--note <決めたこと>` の引用符なしは #270 からの文で `skill_followup_test.go` が固定しているので残した(プレースホルダの文)
  - cross-review 3 回目の run(codex、3edfa501 の triage、Cycle 2/3): ACTION_REQUIRED 1(#1 最初の start の断りのあと `ralph status` を読み直さず、聞いているあいだに立った org の重なりを落とす)、WORTH_CONSIDERING 1(#2 add のあと done の前に止まると承認した着手が黙って失われる)。cap 3 に届いていないので、指示どおり聞かずに直し、cycle を 3 にした(4 回目の run)
  - S14 09c9f34d(impl-271-s14): #1 は「起票と記録」の 1 の先頭で `ralph status` を打ち直して重なる org を全部数え直す、出力先の 3 からも指す。#2 は引き継ぎに「すべて open・すべて merged・どれも waiting」の規則を足した。検証 rc=0、変異 5 つが落ちた
  - S15 ac5a0578(impl-271-s15): S14 の #2 は consult の指摘(A は ready、B は記録の直前に merge されて waiting の混在に当たらない。「起票と記録」の 4 で読んでから add までの merge も同じ形)を受け、ブランチごとの規則にした: 2 の判定で、open の follow-up の STATE が `waiting` でも merge-order で `merged` なら記録の直前に merge 済みとして待ち終えたと数え、2 を始める条件にもこの件を足し、引き継ぎと 3 はこの規則を指す。検証 rc=0、変異 9 個が落ちた、strict scan clean、自己点検 5 項目(誤字を直した、ほか clean)
  - cross-review 4 回目の run(codex、9b1175c3 の triage、Cycle 3/3 cap reached): ACTION_REQUIRED 1(#1 ブランチ名の使い回しで merge-order が前の PR を `merged` と出し、今の待ちを済んだと数える)、WORTH_CONSIDERING 2(#2 確かめのあとの予約の取り合いで start が残した worktree が書き直した分割計画の start を止める、#3 fork の checkout で gh の issue のコマンドが upstream を指す)。ユーザーの指示どおり上限を 4 に上げ(`cap: 4`、`cycle` は 3 のまま)、3 件とも直して 5 回目の run を回す。指示は 4 までなので、5 回目の cross-review で残った所見は PR の Known gaps に書く
  - S16 a7e7e4bd(impl-271-s16): #1 は「起票と記録」の 4 に、そのブランチを `feature:` の行に持つ org が走っていれば `merged` でも merge 済みと数えない、を置き、引き継ぎと「merge を待つ issue」の 2 から指す。#2 は書き直しの前に `./scripts/ralph-worktree.sh state-path org-<slug>` の記録を確かめ、あれば slug を変えるか人に聞く(cleanup は打たない)。#3 は gh の issue のコマンド 11 か所に `--repo <owner>/<repo>`、決め方(origin の URL の sed)を「起票と記録」の末尾に。検証 rc=0、変異 10 個が落ちた、`--repo` の view と list は読み取りで実際に打った
  - S17 cfbfe147(impl-271-s17、sonnet): S16 の consult の指摘(worktree が残るのは確かめのあとの予約の取り合いと、PR が merge されずに閉じたとき)で 3 の括弧の文を直し、「予約外の文書の follow-up」の 2 の `gh pr view` にも `--repo` を付けた。検証 rc=0。「merge を待つ issue」の 1(#270 からの文)の使い回しの穴は PR の Known gaps
  - self-review 5 回目の run ca4e87d3: LOW 5(C5-1 「merge を待つ issue」の 1 の使い回しの穴、C5-2 worktree が残る場合の書き方、C5-3 origin の URL の sed が host を落とす、C5-4 sed を打つテストがない、C5-5 comment)、Merge: yes。verify 5 回目の run 0b02aecf: pass、INFO 4、C5-3 を probe で再現(ローカルのパスや `file://` の origin で sed が `foo/bar` を出し、github.com の無関係な repository に issue を立てうる)。外に書き込む操作なので C5-3 だけ直す
  - S18 172b713c(impl-271-s18): `<owner>/<repo>` を決める前に origin の URL が github.com の repository を名指す形(https・ssh・`git@github.com:`)かを `grep -Eq` で確かめ、当たらなければ gh の issue と pr のコマンドを打たず人に言う(GHES は対象外)。`ralph org merge` と `ralph org cleanup` もローカルの origin を断る(merge-order は照合を飛ばすだけ)。検証 rc=0、判定を当たる URL 11 個と当たらない URL 12 個で Go の regexp と実際のシェルで確かめた、変異 3 つが落ちた、strict scan clean、自己点検 5 項目 clean。S18 の差分だけ self-review と verify を回し直してから /test に進む
  - cross-review 5 回目の run(codex、23ab59d7 の triage、Cycle 3/4): ACTION_REQUIRED 2(#1 `GH_HOST` が Enterprise の host だと `--repo <owner>/<repo>` が別の host を指す、#2 印の行のない普通の issue で、着手のあとの断りで待つと答えると記録できない)、WORTH_CONSIDERING 1(#3 書き換えた待ちに前と同じブランチがあると古い done の件で新しい待ちを片付ける)。cycle 3 < cap 4 なので指示どおり直して cycle を 4 にした(6 回目の run)。4 回目の triage の「5 回目で最後」は私の読み替えで、consult に確かめて訂正した。6 回目の cross-review(4/4)で残った所見は Known gaps
  - S20 d8dfb9b9(impl-271-s20): #1 は gh の 13 か所を `--repo github.com/<owner>/<repo>`、書いた人の確かめを `gh api user --hostname github.com`(`GH_HOST=ghe.example.invalid` でも通り、host なしの `--repo` だけ落ちることを読み取りで確かめた)。#2 は印の行が確かめに通らない issue は本文を書き換えず `feature:` のブランチごとに `followup add` だけ。#3 は引き継ぎで数える件を NOTE が今の印の行と同じ件にし、近くの 5 か所もそろえた。検証 rc=0、変異 12 個が落ちた、strict scan clean、自己点検 5 項目 clean
  - 計画の本文とのずれ(digest を変えないためここに書く): D3・AC4・AC7 は、本文から印の行を読むときの確かめ(印の行の正規表現、ちょうど 1 行、書いた人が director の gh のアカウント、note は確かめた値から組み直す)を書いていない。self-review M1 の直し(S5)で足した。AC4・AC7 はこの確かめを含めて読む
- [x] Verification artifact created(`docs/reports/verify-2026-11-01-director-split-questions.md`、pass、cycle 1。AC1〜AC7 を満たし、AC8 は静的の部分。所見は LOW 3(V1〜V3、文書のずれ)と INFO 4)
- [x] Test artifact created(`docs/reports/test-2026-11-01-director-split-questions.md`、pass、cycle 1。`run-test.sh` rc 0。変異 20 個中 17 個が落ち、残った 3 個はテストの穴 1・2)
  - S8 8bc6ae84(impl-271-s8): テストの穴 1・2 を埋めた。`skill_split_questions_test.go` の当たってはいけない行の表に shell に届く文字を 1 つだけ持つ行を 14 行、`director_skill_sync_test.go` に `TestDirectorSkillSection` の 6 行の表。変えたのはテストの 2 ファイルだけで、skill の文は変えていない。残った変異 3 つは足したテストで落ちた(test report の Re-run)
  - sync-docs(`docs/reports/sync-docs-2026-11-01-director-split-questions.md`): V1(spec の FR-9 に 2026-11-01 の段落)、V2(tech-debt の status の ready の行と test gaps の (7) の Trigger を振る舞いで書き直した)、V3(`/director` skill の受信箱の処理と `director_skill_sync_test.go` を Trigger に持つ行に持ち越しの注。自分で Trigger を掃いて見つけた 3 行にも注。recipe の `docs/evidence/` の引用の行は `director-wake.md` の分を払った)、I1(`skill_split_questions_test.go` の行を足した)、I3(README と `docs/recipes/director-wake.md` に、push の 1 行が開いた AskUserQuestion を閉じることを足した)。skill と Go は変えていない。AC の checkbox は付けていない(lead か /pr が付ける)
  - verify cycle 2 d4fa10f3(`docs/reports/verify-2026-11-01-director-split-questions.md` の `## Cycle 2`): pass。AC4・AC5・AC7 は cross-review cycle 1 の #1〜#3 と self-review cycle 2 の C2-1〜C2-6 の直しを満たし、AC8 は静的の部分。所見は LOW 2(C2-V1 `/org` skill の例外の 1 文、C2-V2 tech-debt の行数)と INFO 4
  - test cycle 2 3b388a55(`docs/reports/test-2026-11-01-director-split-questions.md` の `## Cycle 2`): pass。`run-test.sh` rc 0(1,447 秒)、`(cached)` だった 8 パッケージもキャッシュなしで ok。S9〜S11 の断片への変異は 16 個中 15 個が落ちた(残った C15 は断片の方式に元からある性質)。テストの穴 1(`--limit` の値がテストの helper と断片に別々に書かれ、片方だけ変えても通る)は LOW で、test の report に残る
  - sync-docs cycle 2(`docs/reports/sync-docs-2026-11-01-director-split-questions.md` の `## Cycle 2`): C2-V1(`/org` skill の「merge を待つ issue」に、director が起票した issue への答えを残す例外の 1 文。4 つの写しを作り直した)、C2-V2(tech-debt の `skill_split_questions_test.go` の行数)、テストの穴 1(tech-debt の新しい行)、spec の FR-9 の #271 の段落に S9〜S11 の振る舞い。AC の checkbox は付けていない(lead か /pr が付ける)
  - verify cycle 3 7f4046a0(`docs/reports/verify-2026-11-01-director-split-questions.md` の `## Cycle 3`): pass。AC4・AC5 は cross-review cycle 2 の #1〜#3 と self-review 3 回目の run の C3-1・C3-3・C3-4・C3-6 の直しを満たし、AC8 は静的の部分。`/director` と `/org` の 4 つの写しは同じ、計画の digest も合う。所見は INFO 4(C3-I1 spec の (5)、C3-I2 `DONE_NOTE` が `-` のとき、C3-I3 ブランチ名の使い回し、C3-I4 insight event に self-review の行がない)で、LOW 以上はない
  - test cycle 3 b5701eca(`docs/reports/test-2026-11-01-director-split-questions.md` の `## Cycle 3`): pass。`run-test.sh` は rc 1(2,047 秒)で、落ちたのは負荷による時間切れの `TestRunWatcher_TimeoutIndependentOfSmallInterval` 1 本だけ。`internal/org` をキャッシュなしで打ち直して ok、単独でも 20/20、`(cached)` だった 8 パッケージもキャッシュなしで ok なので Fail に数えない。変異は 29 個中 27 個が落ち、残った D21・E7 はテストの穴 1・2
  - sync-docs cycle 3(`docs/reports/sync-docs-2026-11-01-director-split-questions.md` の `## Cycle 3`): C3-I1(spec の FR-9 の #271 の段落に S12・S13 の振る舞い)、テストの穴 2(tech-debt の `--limit` の行に、正規表現の表にタブの行がないこと)、`TestRunWatcher_TimeoutIndependentOfSmallInterval` の時間切れの tech-debt の新しい行、insight event に `self_review` cycle 3 と `sync_docs` cycle 3。`/director` skill と Go は変えていない。AC の checkbox は付けていない(lead か /pr が付ける)
  - self-review 4 回目の run 929a3b71(S14・S15 の差分。`docs/reports/self-review-2026-11-01-director-split-questions.md` の `## Cycle 4`): CRITICAL・HIGH・MEDIUM 0、LOW 5(C4-1〜C4-5)、Merge: yes。cross-review 3 回目の run の #1・#2 は直っていた。C4-1(名前の使い回しで古い merge を待ち終えたと数える道)と C4-3(数え直しが 0 件のときの枝がない)は PR の Known gaps に書く。C4-2(「直前」の言い方)、C4-4(テストの comment の `cycle 3`)、C4-5(折り返し)は直さない
  - verify cycle 4 b05d6af6(`docs/reports/verify-2026-11-01-director-split-questions.md` の `## Cycle 4`): pass。AC3〜AC5 は S14・S15 の直しを満たし、AC8 は静的の部分。`/director` と `/org` の 4 つの写しは同じ、計画の digest も合う。所見は INFO 3(C4-I1 spec の (4)・(5)、C4-I2 `/org` skill の答えの記録の例外、C4-I3 insight event に self-review の行がない)で、LOW 以上はない
  - test cycle 4 960c88c3(`docs/reports/test-2026-11-01-director-split-questions.md` の `## Cycle 4`): pass。`run-test.sh` rc 0(1,625 秒)、`(cached)` だった 8 パッケージもキャッシュなしで ok。変異は 20 個中 17 個が落ち、残った G16・G20・G21 は見込みどおりで、テストの穴 1・2
  - sync-docs cycle 4(`docs/reports/sync-docs-2026-11-01-director-split-questions.md` の `## Cycle 4`): C4-I1(spec の FR-9 の #271 の段落の (4) に起票の前の数え直し、(5) に open の件が `waiting` でも merge-order で `merged` のブランチは待ち終えたと数えること)、C4-I2(`/org` skill の答えの記録の例外に、add と done のあいだで止まった件は director が「merge を待つ issue」の 2 で拾う、の 1 文。4 つの写しを作り直した)、C4-I3(insight event に `self_review` cycle 4 と `sync_docs` cycle 4)。`/director` skill と Go は変えていない。AC の checkbox は付けていない(lead か /pr が付ける)
  - self-review 5 回目の run(`docs/reports/self-review-2026-11-01-director-split-questions.md` の `## Cycle 5` ca4e87d3 と `## Cycle 5 (S18)` 646aff7a): どちらも Merge: yes。S16・S17 の差分は LOW 5(C5-1〜C5-5)、S18 の差分は LOW 4(C5S-1 断る理由の文、C5S-2 token 入りの URL を人に言う、C5S-3 `GH_HOST`、C5S-4 断る側の表)。C5-3 は github.com の origin では直っていた
  - verify 5 回目の run(`docs/reports/verify-2026-11-01-director-split-questions.md` の `## Cycle 5` 0b02aecf と `## Cycle 5 (S18)` f86feec0): どちらも pass。S16・S17 の節は INFO 4(C5-I1〜C5-I4)で、cross-review 4 回目の run の #1〜#3 が直ったことを確かめた。S18 の節は LOW 1(C5S-V1 tech-debt の Trigger に S18 のテストの追加が当たった)と INFO 1(C5S-V2 判定が断ったあとの次の手)
  - test 5 回目の run(`docs/reports/test-2026-11-01-director-split-questions.md` の `## Cycle 5` 30e19d17): pass。`run-test.sh` rc 0(1,234 秒)、`(cached)` だった 8 パッケージもキャッシュなしで ok。変異は 28 個中 23 個が落ち、残った H9・H11・H13・H19・H28 はテストの穴 1〜3。S19 e633b89c(impl-271-s19): テストの穴 1 を埋めた。`skill_split_questions_test.go` の origin の URL の表に断るべき行を 3 つ足し、H9・H11・H13 は足した行で落ちた(H19・H28 は残る)。変えたのはテストの 1 ファイルだけで、skill の文は変えていない。`run-test.sh` は打ち直していない(`## Re-run(S19 のあと、5 回目の run)` bdebf8e4)
  - sync-docs cycle 5(`docs/reports/sync-docs-2026-11-01-director-split-questions.md` の `## Cycle 5`): C5S-V1(tech-debt の `skill_split_questions_test.go` の行とその `--limit` の行に、S18 のテストの追加が Trigger に当たったが払っていない持ち越しの注。ファイルの行数の行は `skill_doc_followup_test.go` の 891 を 898 に直した)、C5-I2(spec の FR-9 の #271 の段落に S16〜S18 の振る舞い: gh の issue と pr のコマンドの `--repo`、origin が github.com の repository を名指さないときは打たないこと、ブランチ名の使い回しのただし書き、書き直しの前の worktree の記録の確かめ)、C5-I4(insight event に `self_review` cycle 5 と `sync_docs` cycle 5)。`/director` skill と Go は変えていない。AC の checkbox は付けていない(lead か /pr が付ける)
  - self-review 6 回目の run(`docs/reports/self-review-2026-11-01-director-split-questions.md` の `## Cycle 6` 942b2d7e): Merge: yes。LOW 2(C6-1 S20 が足した test の comment の置き去りの短い行と cite の数え方、C6-2 「印の行の形と読み方」の結びの「記録もしない」の括弧が S20 の `followup add` を指さない)。cross-review 5 回目の run の #1〜#3 は直っている
  - verify 6 回目の run(`docs/reports/verify-2026-11-01-director-split-questions.md` の `## Cycle 6` 8a1915e3): pass。LOW 1(C6-V1 spec FR-9 の #271 の段落が S20 の前のまま)、INFO 2(C6-I1 印の行の一般の規則の括弧が 3 の道を挙げない、C6-I2 書き換えた `after:` が前と並びまで同じときの例外)。`GH_HOST=ghe.example.invalid` のまま、host つきの読み取りの 4 つの形は通り、host なしの 2 つの形は落ちた
  - test 6 回目の run(`docs/reports/test-2026-11-01-director-split-questions.md` の `## Cycle 6` 67672efa): pass。`run-test.sh` rc 0(1,495 秒)、`(cached)` だった 5 パッケージもキャッシュなしで ok。変異は 15 個中 14 個が落ち、残った I15(`--repo` のない gh のコマンドを足しても通る)はテストの穴 1。最初の tester は session の再起動で止まり、立て直した
  - sync-docs cycle 6(`docs/reports/sync-docs-2026-11-01-director-split-questions.md` の `## Cycle 6`): C6-V1(spec の FR-9 の #271 の段落の 3 か所: `--repo github.com/<owner>/<repo>` と書いた人の確かめの `--hostname github.com`、引き継ぎで数える follow-up を「`NOTE` の列が今の印の行と同じ件」に、印の行の確かめに通らない issue は本文を書き換えず `followup add` だけを打つこと)、insight event に `self_review` cycle 6 と `sync_docs` cycle 6。`/director` skill と Go は変えていない。tech-debt の Trigger はこの run で発火した行なし。AC の checkbox は付けていない(lead か /pr が付ける)
- [x] PR created(#295)
