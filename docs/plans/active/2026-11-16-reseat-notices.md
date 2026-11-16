# reseat-notices

- Status: Approved
- Approved: 2026-11-16 sha256:2aa9b9bde56c
- Owner: Claude Code
- Date: 2026-11-16
- Related request: v8.2.0 の release のあとに見つけた漏れ。director skill で雛形の版の行を出す動詞の一覧から `ralph org reseat` が抜けている。調べると、reseat は codex の座席を立て直すときに codex の hook の承認の行も出していなかった。ユーザーが「両方を 1 本で」直すと決めた(2026-11-16)
- Related issue: N/A
- Type: fix
- Branch: fix/reseat-notices

## Objective

人に向けた 2 つの `note:` の行について、`ralph org reseat` を `ralph org start` と同じ扱いにそろえる。

- codex の hook の承認の行(#310): spawn・start・director start は codex の agent を立てる前に出すが、reseat は出さない。reseat は `--driver codex` で座席を codex に替えて立て直せるので、承認されていない hook のまま codex の座席が知らせなく動きうる。reseat も、立てる codex の座席ごとに同じ行を出す
- 雛形の版の行(#313): reseat はすでに出している(`internal/cli/org_reseat.go:215`)。それを並べる文書の一覧に reseat が無いので足す

## Scope

- `internal/org/reseat.go`: `ReseatParams` に、計画ができたあと、どの座席も止める前に 1 度だけ呼ぶ関数(仮名 `OnPlan func(ReseatPlan)`)を足す。`ReseatParams` にすでにある関数のフィールド(`WatchProbe`、`HerdrServerProbe`)と同じ形にする。nil なら何もしない。呼ぶ場所は 2 つある。dry run では、`Reseat` が `PlanReseat` のすぐあとで返す前(`reseat.go:1111-1114`)。本物の reseat では、ロックの中で `PlanReseat` が通り、印を書いたあと、`stopAll` の前
- `internal/cli/org_reseat.go`: その関数で、`plan.Start` の座席ごとに `printCodexHookTrustNotice(stderr, seat.Driver, "org reseat", seat.SeatID, <座席が動くディレクトリ>)` を呼ぶ。座席が動くディレクトリは機能の worktree(`plan.Binding.Worktree`)。claude の座席では関数が何もしない。reseat の `--help` の末尾に `orgCodexHookTrustNoticeHelp` を足す
- コメント: `printCodexHookTrustNotice`、`orgCodexHookTrustNoticeHelp`、`orgScaffoldVersionNoticeHelp` の、呼ぶ動詞を並べたコメントに reseat を足す
- `/director` skill(`.claude/skills/director/SKILL.md` と `templates/base/.claude/skills/director/SKILL.md`。`.agents` の 2 写しは `scripts/sync-skills.sh` で作る)の「人に向けた知らせの行」
  - 雛形の版の行を出す動詞に `ralph org reseat` を足し、「`ralph org` の 3 つの動詞」を 4 つにする
  - hook の行を出す動詞に `ralph org reseat` を足す
  - 「`ralph org start` が出した行を見たら、行をそのまま人に伝える」を reseat にも当てる(reseat を打つのは director か人で、leader ではない)
- `internal/org/prompts/director.md` の 62 行: 「`ralph status` か機能ごとの org の start」に reseat を足す
- `docs/specs/2026-10-07-org-multi-org-director.md` の #318 の項の動詞の一覧、`.codex/README.md` と `templates/base/.codex/README.md` の hook の段落の動詞の一覧に reseat を足す
- テスト: reseat が codex の座席に hook の行を出すこと、claude の座席には出さないこと、reseat の `--help` が段落を持つこと。`internal/org/prompts_notice_test.go` の skill の断片に reseat を入れる

## Non-goals

- 行の文面と、`printCodexHookTrustNotice`・`printScaffoldVersionNotice` の中身の変更
- `ralph org resume` など、座席を新しく立てない動詞(hook を読み直さない)
- leader の role prompt(leader は reseat を打たない。`internal/org/prompts/leader.md:354`)
- 次の release の注記(`internal/upgrade/notes/<版>.md`)。`/release` の段 4b で書く
- 見張りや受信箱の動き

## Assumptions

- reseat が立てる座席は、どれも機能の worktree(`plan.Binding.Worktree`)で動く。leader も `StartFeature` から同じ worktree で立つ。`codexHookTrustFor` は linked worktree を main のチェックアウトに読み替える(`codexHookCheckoutOf`)ので、start --plan が main worktree のルートで確かめるのと同じ hooks.json を読む
- `printCodexHookTrustNotice` は速い(hooks.json と codex の設定を読むだけ)。reseat のロックの中で呼んでも、ほかの reseat を長く待たせない
- reseat の座席の `Driver` は計画の時点で決まっている(記録された driver か `--driver`)

## Affected areas

- `internal/org/reseat.go`、`internal/cli/org_reseat.go`、`internal/cli/codex_hook_trust_notice.go`(コメント)、`internal/cli/scaffold_version_notice.go`(コメント)
- `internal/org/prompts/director.md`
- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`
- `.codex/README.md`、`templates/base/.codex/README.md`
- `docs/specs/2026-10-07-org-multi-org-director.md`
- テスト: `internal/cli/codex_hook_trust_notice_test.go`(または `internal/cli/org_reseat_test.go`)、`internal/org/reseat_test.go`、`internal/org/prompts_notice_test.go`

## Visual review

- ページ: `.harness/state/plan-visual/reseat-notices.html`。かんたんな図 2 枚(`overview`、`easy-docs`)と、詳しい図 2 枚(`detail-overview`、`detail-order`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 4 枚の図を `shot` で撮って読んだ。ファイルの地図で直したのは 2 点。コメントを直すファイルを「変える」の色にして S2 の印を付けたことと、文書の箱の「ほか 1 つ」を「ほか 2 つ」に直したこと。撮り直して、どれもきれいに出ることを確かめた

## Design decisions

Critical forks: None。

- D1(hook の行を出す時): reseat の計画ができたあと、どの座席も止める前に、立てる codex の座席ごとに 1 行ずつ出す。spawn と start が agent を立てる前に出すのと同じ順にするためで、reseat が途中で切れても行は人に届く。退けた案は、reseat が終わってから `res.Plan` で出す形。internal/org を変えずに済むが、行が止める・立てるの後になり、切れた reseat では順が start と食い違う
- D2(仕組み): `ReseatParams` に関数を 1 つ足し、internal/org から internal/cli へ計画を渡す。internal/org は internal/cli を import できない(逆向きの依存)ので、CLI が関数を渡す。dry run でも呼ぶ(`ralph org spawn --dry-run` も行を出す)
- D3(座席が動くディレクトリ): 機能の worktree(`plan.Binding.Worktree`)を渡す。spawn は座席の cwd を渡し、start --plan は worktree がまだ無いので main worktree のルートを渡している。reseat では worktree があるので、座席が実際に動く場所を渡す
- D4(文書): 一覧に reseat を足すだけで、director の動きは変えない。reseat の行は director か人が打った reseat から直接見えるので、start の行と同じく「そのまま人に伝える」に入れる

## Acceptance criteria

- [ ] AC1: 本物の `ralph org reseat` は、立てる座席のうち driver が codex のものごとに、`printCodexHookTrustNotice` の行(動詞 `org reseat`、座席 id)を stderr に出す。出すのはどの座席も止める前。claude の座席には出さない。計画の前に断られた reseat は出さない
- [ ] AC2: `ralph org reseat --dry-run` も、AC1 と同じ行を出す
- [ ] AC3: `ralph org reseat --help` が `orgCodexHookTrustNoticeHelp` と `orgScaffoldVersionNoticeHelp` の両方の段落を持つ。3 つのコメントが reseat を挙げる
- [ ] AC4: `/director` skill の 4 写しの「人に向けた知らせの行」が、2 つの行の動詞に `ralph org reseat` を挙げる。「`ralph org` の 4 つの動詞」になり、reseat の hook の行も「そのまま人に伝える」に入る。`internal/org/prompts_notice_test.go` がこの断片を確かめる
- [ ] AC5: `internal/org/prompts/director.md` が、2 つの行を出す動詞に reseat を含める。テストが確かめる
- [ ] AC6: multi-org-director の仕様の #318 の項と、`.codex/README.md` の 2 写しが reseat を挙げる。`scripts/check-sync.sh` と `scripts/check-skill-sync.sh` が通る
- [ ] AC7: `OnPlan` を nil にした reseat は今までどおり動く(既存の reseat のテストが通る)

## Implementation outline

1. S1(internal/org): `ReseatParams` に `OnPlan func(ReseatPlan)` を足し、`Reseat` が計画のあと、止める前に 1 度呼ぶ(dry run でも)。nil なら呼ばない。呼ばれる時と回数を確かめるテストを足す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Reseat' ./internal/org/`
2. S2(internal/cli): `runOrgReseat` が `OnPlan` で、立てる座席ごとに `printCodexHookTrustNotice` を呼ぶ。reseat の `--help` に段落を足し、3 つのコメントを直す。codex の座席では行が出ること、claude の座席と dry run の扱い、`--help` を確かめるテストを足す — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'CodexHookTrust|Reseat|ScaffoldVersion' ./internal/cli/`
3. S3(文): director skill の 2 写しと `scripts/sync-skills.sh` で作る `.agents` の 2 写し、`internal/org/prompts/director.md`、仕様の #318 の項、`.codex/README.md` の 2 写しを直し、`internal/org/prompts_notice_test.go` の断片を合わせる — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Notice|DirectorSkill|RenderRolePrompt_Director' ./internal/org/ ./internal/cli/`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync)
- Spec compliance criteria to confirm: AC1〜AC7。AC1 は行が止める前に出ることを、テストの順の記録か stderr と台帳の event の順で確かめる
- Documentation drift to check: 2 つの行を出す動詞を並べる文がほかにないか(`git grep 'org director start'` で、雛形・hook・note の語を含む行)。AGENTS.md の internal/org の行に reseat の知らせを書くべきか
- Evidence to capture: verify のログ、`git grep` の結果

## Test plan

- Unit tests: S1 の `OnPlan` が 1 度呼ばれること。dry run でも呼ばれ、本物では `stopAll` の前に呼ばれること(`reseatTestHooks.beforeStopCheck` との順で確かめる)。nil で何も変わらないこと。計画の前に断られたら呼ばれないこと
- Integration tests: S2 の CLI のテストで、codex の座席を持つ org を reseat して stderr に行が 1 つ出ること、claude の座席だけなら出ないこと、`--driver codex` で claude の座席を codex に替えると行が出ること
- Regression tests: 既存の reseat、hook の行、雛形の版の行のテストがすべて通ること。`./scripts/run-test.sh`
- Edge cases: codex の座席が複数のときは座席ごとに 1 行ずつ出ること。worktree の hooks.json が main のチェックアウトに読み替えられること(start --plan のテストと同じ組み立て)
- Evidence to capture: run-test.sh のログ、変異(`OnPlan` を呼ばない、driver の判定を外す)が落ちること

## Risks and mitigations

- `OnPlan` が reseat のロックの中で呼ばれ、遅いとほかの reseat を待たせる: printCodexHookTrustNotice は小さなファイルを読むだけ。関数は計画を読むだけにし、ralph の動詞を打たない、とコメントに書く
- 行を止める前に出すので、そのあと断られた reseat(座席の記録が変わった、など)でも行が残る: start も断られる前に行を出しているので同じ扱い。行は「この座席を codex で立てるなら hook が動かない」と言うだけで、立てたとは言わない
- skill の文の変更が prompts_notice_test.go の断片の順を崩す: S3 で断片を一緒に直す
- 計画の前に断られた reseat は行を出さない。start --plan は driver を決めたあとすぐ行を出すので、そのあと StartFeature に断られても行は残る。reseat が行を出さないのはこの点で start と違う。ただ、行は「codex seat <座席>」と座席を名指すので、座席が決まる計画の前には出せない。違いは help のコメントに書く

## Rollout or rollback notes

- skill の変更は `ralph upgrade` で届き、reseat と director の prompt の変更は新しい binary が要る。次の release の注記に書く
- 戻すときは PR を revert する。`OnPlan` は nil なら何もしないので、片方だけ戻しても壊れない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: no findings (codex)。consult(consult-plan-reseatnotices)の判定は「進めてよい」。言われた 3 点を入れた: `OnPlan` を呼ぶ 2 つの場所を Scope に書く、dry run と止める前の順を Unit tests に足す、計画の前に断られたときの start との違いを Risks に書く
- [x] Plan approved
- [x] Branch created
- [x] Implementation started — S1 は b21c75c7(`ReseatParams.OnPlan`、テスト 2 つ・8 ケース、変異 7 つがすべて落ちた。印の書き込みの失敗で呼ばない経路は、失敗させる差し込み口がなくテストしていない)。S2 は 72cc7d72(`printOrgReseatCodexHookTrustNotices`、テスト 4 つ、help のテストに reseat、変異 5 つがすべて落ちた。止める前に出ることは、行を書いた時点の manifest が reseat 前と同じことで確かめた)。S2 が見つけた `TestScaffoldVersionHelp_EachVerbDescribesTheNote` に reseat が無い件は、S3 に足した(`internal/cli/scaffold_version_notice_cli_test.go`)。S3 は 0d7dfe1f(director skill の 4 写し、director.md、`prompts_notice_test.go`、仕様の #318 の項に日付つきの 1 文、`.codex/README.md` の 2 写し、`TestScaffoldVersionHelp_EachVerbDescribesTheNote`)。
- 計画からのずれ(S4、f18738bd): S3 の検索で、この変更が開く穴が見つかった。director skill の「座席を立て直す」は dry run の reseat のあとに本当の reseat を打たせるが、どちらも同じ座席に同じ hook の行を出す。この節は知らせの行の小節を指しておらず、hook の行には「1 度だけ伝える」の決まりもない。そこで S4 で次の 3 つを足した。1 つ目は、手順 2 から小節を指す文。2 つ目は、手順 4 と小節の reseat の項に置く「dry run のあとで伝えた座席の行は、もう 1 度は伝えない」。3 つ目は、それを確かめる `TestDirectorSkill_ReseatPointsAtTheHumanNotes`。あわせて、tech-debt の 478 行(承認のあとで変わった hook に起動の知らせが黙る穴)の動詞に reseat を足した。どれも AC4 の「reseat の行を start の行と同じく扱う」を満たすための直しで、計画の Scope に挙げた director skill の小節の外にも及んだ。変異 5 つはすべて落ちた
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
