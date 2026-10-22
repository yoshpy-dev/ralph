# seat-wip-checkpoint

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-22
- Related request: 試行の所見を直す系列の 4 本目の 3 つ目「座席の環境 4c」(ユーザーの依頼「2. 修正可能なものは修正してください」、2026-10-21)
- Related issue: N/A
- Type: fix
- Branch: fix/seat-wip-checkpoint

## Objective

ralph が作る herdr の pane で動く Claude Code のセッションが、終わるときと context を compact するときに `wip:` のコミットを作らないようにする。試行(stage 9)の所見 F11 と tech-debt の 191 行を閉じる。あわせて、4b(#249)の sync-docs が残した leader の雛形の欠け(spawn の失敗の扱い)を埋め、記録に残る「fix/seat-spawn-close」を「#249」に直す。

## Scope

- 2 つの hook(`.claude/hooks/session_end_summary.sh` と `.claude/hooks/precompact_checkpoint.sh`、`templates/base/.claude/hooks/` の写し)が、環境の `RALPH_WIP_CHECKPOINT` がちょうど `off` のとき WIP コミットだけを飛ばす。summary と checkpoint の md は今までどおり書き、コミットを飛ばしたことを 1 行書く
- ralph が作る pane(座席の leader・implementer・reviewer と、herdr の headless の director)の `herdr tab create` に、`--env AGMSG_SELF_NAME=off` に加えて `--env RALPH_WIP_CHECKPOINT=off` を付ける。pane に足す値の一覧は 1 か所に置き、launch_cmd の起動スクリプトもその一覧を export する
- `--env` のない herdr の warning の文と `Herdr.PaneEnv` の説明を、2 つの値それぞれの帰結を書く形に直す(G3)
- leader の雛形(`internal/org/prompts/leader.md`)の番号のない「運用規律」の節に、spawn の失敗の扱いを 1 項足す
- 文書: `.claude/rules/ralph/git-commit-strategy.md`(と写し)、`/org` と `/director` の skill(と写し)、`docs/recipes/director-launch-cmd.md`(と写し)のうち pane の環境を書いた箇所
- 実機の確かめ: 分離した herdr server で本物の claude を使い、hook が pane の環境の値を見ることを確かめる
- 記録: tech-debt の F11 の行(326 行)と 191 行、試行の evidence の F11 の文への訂正の注、根拠の表と仕様の F11 の行、「fix/seat-spawn-close」を「#249」に(仕様、根拠の表、tech-debt)、PR 一覧を 32 本に

## Non-goals

- tech-debt 192 行(`start <task>` の leader と昇格した leader が /plan の task worktree に org report を残し、/pr の cleanup が断る)。座席の hook とは別の話で、今回は変えない
- 人が自分で打つ対話セッションの WIP コミットの振る舞い。`RALPH_WIP_CHECKPOINT` を設定しない限り今までどおり
- Codex の SessionEnd・PreCompact の配線(tech-debt 127 行で配線しないと決めてある)
- `--env` のない herdr で spawn を断ること。4a と同じく warning を出して進む
- 試行の evidence(`docs/evidence/org-trial-2026-10-20.md`)の本文の書き換え。記録なので注を足すだけにする
- 古い binary が作った pane(`RALPH_WIP_CHECKPOINT` を持たない)への手当て

## Assumptions

- Claude Code の hook は、Claude Code のプロセスの環境を受け継ぐ。4a の evidence の (e) で、`--env` 付きの pane で起動した claude の Bash の子が値を見たことは確かめてある。hook も同じ子のはずだが未確認で、S3 の実機の確かめで見る
- `ralph org stop` の C-c と `herdr pane close` で、座席の SessionEnd の hook が走る。試行の時刻(WIP コミット 05:56:51Z が implementer の stop 05:56:46Z と reviewer の stop 05:56:52Z のあいだ)からの推定で、S3 で確かめる
- herdr 0.7.5 の `tab create --env` は複数回付けられる(4a の (a) は 1 つの値で確かめた。S3 で 2 つ付けて確かめる)

## Affected areas

- `.claude/hooks/session_end_summary.sh`、`.claude/hooks/precompact_checkpoint.sh`、`templates/base/.claude/hooks/` の同じ 2 つ
- `tests/test-wip-checkpoint-hooks.sh`(新規)
- `internal/org/driver/herdr.go`(`PaneEnv` の説明、`warnNoPaneEnv`)、`internal/org/driver/agmsg.go` か新しい小さなファイル(pane に足す値の一覧)
- `internal/cli/org.go`(本番の組み立て、356 行)
- `internal/org/director_cmd.go`(起動スクリプトの export)
- テスト: `internal/org/driver/herdr_test.go`、`internal/cli/org_runtime_env_test.go`、`internal/cli/org_test.go`、`internal/cli/org_director_headless_test.go`、`internal/org/director_cmd_test.go` ほか、`--env AGMSG_SELF_NAME=off` を固めているもの
- `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- `.claude/rules/ralph/git-commit-strategy.md` と `templates/base/.claude/rules/ralph/git-commit-strategy.md`
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- `docs/recipes/director-launch-cmd.md` と `templates/base/docs/recipes/director-launch-cmd.md`
- `docs/tech-debt/README.md`、`docs/evidence/org-trial-2026-10-20.md`(注だけ)、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/specs/2026-10-07-org-multi-org-director.md`
- `docs/evidence/seat-wip-checkpoint-2026-10-22.md`(新規)

## Visual review

- Page: `.harness/state/plan-visual/seat-wip-checkpoint.html`
- Lint: `./scripts/plan-visual.sh lint` rc 0
- Self-check: 全体、易しい図 3 枚(`overview`、`easy-yours`、`easy-spawn`)、折りたたみの中の 3 枚(`detail-overview`、`detail-sequence`、`detail-hook`)を撮って読んだ。詳しい図 1 の S3・S5 の印が箱の縁に重なっていたので「記録」の見出しの横へ移し、撮り直して崩れがないことを確かめた

## Design decisions

- D1 止め方: 座席であることを hook に知らせる印(例 `RALPH_ORG_SEAT`)ではなく、WIP コミットだけを止める値 `RALPH_WIP_CHECKPOINT=off` を使う。用途が名前で分かり、grep でき、hook は org のことを知らずに済み、人も対話セッションで同じ値を設定して止められる。`off` 以外の値(空、`on`、`OFF` など)は今までどおりコミットする
- D2 効かせる範囲: ralph が作る pane のすべて(leader を含む座席と、herdr の headless の director)。leader も座席の pane で動き(`spawn.go` の `TabCreate`)、機能の worktree を implementer と reviewer と分け合うので、どの座席のセッション終わりも同じ worktree をコミットしうる。headless の director は main worktree のルートで動き、ふつう main なので hook はもともと飛ばすが、揃える。launch_cmd で立てた director は起動スクリプトが同じ一覧を export する
- D3 一覧の置き場所: pane に足す値は 1 つの一覧に置き、本番の組み立て(`driver.NewHerdr`)と起動スクリプトの export の両方がそれを読む。warning はその一覧の値ごとに帰結を書く
- D4 hook の文: summary と checkpoint の md の書き方は変えず、値が `off` のときに `- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)` の行を足し、`git add` も `git commit` も打たない。main・master・unknown のときの振る舞いは変えない
- D5 leader の雛形: F11 のためには手順を変えない。現行の手順 9(gate の前の `git status --porcelain`)と手順 14(HEAD が gate の HEAD と同じか)が、座席の WIP コミットが push に乗るのを既に止めている。追加は 4b の follow-up の 1 項だけで、番号のない「運用規律」の節に置く(手順の位置を固めるテストをずらさない。consult の指摘)
- D6 実機の確かめの比べ方: ralph の spawn を通さず、分離した herdr で `herdr tab create --env` を直接打って本物の claude を起動する(4a の (e) と同じ)。ralph が 2 つの `--env` を付けることは Go のテストで固める。一時の repo には `.claude/hooks/` を丸ごと(dispatcher、`.d`、本体)写し、settings.json と揃える(consult の指摘: 本体だけだと dispatcher を通らず、hook が走らないと読み違える)
- Critical forks: None(D1〜D6 は consult と相談して決めた。どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [ ] AC1: 2 つの hook(root と写しは byte 同じ)は、feature ブランチで作業ツリーが汚れていて `RALPH_WIP_CHECKPOINT=off` のとき、コミットを作らず(HEAD が動かず、index も変わらない)、md に `- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)` を書く。値がないとき、`on`・`OFF`・空のときは今までどおり `wip: checkpoint before session end` / `... context compaction` をコミットする。main では値によらずコミットしない
- [ ] AC2: 本番の組み立ての `Herdr.PaneEnv` は `AGMSG_SELF_NAME=off` と `RALPH_WIP_CHECKPOINT=off` をこの順で持ち、`--env` のある herdr では、座席の pane(spawn と start)と herdr の headless の director の pane の `herdr tab create` に `--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off` が付く。`--env` の確かめは今までどおり 1 回だけ
- [ ] AC3: launch_cmd の起動スクリプトは、check-in とエージェントの exec の前に、一覧のすべての値(`AGMSG_SELF_NAME=off` と `RALPH_WIP_CHECKPOINT=off`)を export する
- [ ] AC4: `--env` のない herdr(help に `--env` がない、help が失敗する)では、どちらの値も付けずに pane を作り、spawn と start は今までどおり成功し、stderr に warning が 1 行出る。その行は 2 つの値を挙げ、それぞれの帰結(agmsg が pane の agent の名前を変えうること、pane の中の Claude Code のセッションの hook が feature ブランチで WIP コミットを作りうること)を書く。`Herdr.PaneEnv` と `warnNoPaneEnv` の説明が「1 つの値」と書かない
- [ ] AC5: leader の雛形の「運用規律」の節に、spawn の失敗の扱いの 1 項がある。中身: spawn が終了コード 1 で `spawn failed:` を出したらエラーの文を読む。`seat left active` と `ralph org stop` が出ていれば、出ている `ralph org stop` を打ってから打ち直す。`--timeout-ms` の期限が切れたと出ていれば、大きい `--timeout-ms` で打ち直す。同じ座席で 2 回続けて失敗したら、打ち直さずに BLOCKED を上げる。spawn に失敗した座席の HELLO が受信箱に届いていても、その座席は動いていない。`internal/org/prompts_test.go` がこの項を固め、手順の位置を固める既存のテストは変えずに通る
- [ ] AC6: 分離した herdr server での実機の確かめを `docs/evidence/seat-wip-checkpoint-2026-10-22.md` に書く。(a) `herdr tab create --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off` の pane で、両方の値が pane の shell に届く。(b) その pane で起動した本物の claude を C-c と `herdr pane close` で終わらせると、SessionEnd の hook が走り(summary の md が書かれる)、コミットは増えず、md に skipped の行がある。(c) 値を付けない pane の同じ手順では WIP コミットが 1 つ増える(対照)。(d) PreCompact: `/compact` を送り、同じく比べる。起こせなければ、そのことと理由を書く
- [ ] AC7: 文書を合わせる。`git-commit-strategy.md`(と写し)の End-of-Session の節に、`RALPH_WIP_CHECKPOINT=off` のときは飛ばすことと、ralph がそれを自分の作る pane に付けることを書く。`/org`・`/director` の skill と `director-launch-cmd.md`(と写し)のうち `AGMSG_SELF_NAME=off` を pane や起動スクリプトの環境として書いた箇所に、2 つ目の値を足す
- [ ] AC8: 記録を合わせる。tech-debt の F11 の行(326 行)と 191 行は、AC6 の (b) と (c) が通ったときだけ RESOLVED にする。326 行の「a leader's session ends ... after the gate, the push, and the PR」を、試行の時刻から読めること(止められた座席のセッション)に直す。RESOLVED の注には、現行の leader の手順 9 と 14 が座席の WIP コミットの push を既に止めていたことと、hook を止めると `git add -A` による巻き込みもなくなることを書く。試行の evidence の F11 の文(806〜808 行)には訂正の注を足す(「after the PR」は時刻と合わない)。根拠の表の F11 の行の「直す PR」と、仕様の Status と PR 一覧(32 本)を合わせる。仕様、根拠の表、tech-debt に残る「fix/seat-spawn-close」を「#249」に直す
- [ ] AC9: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る

## Implementation outline

1. S1(implementer): D1・D4。2 つの hook と写し、`tests/test-wip-checkpoint-hooks.sh`(AC1)
2. S2(implementer): D2・D3。pane に足す値の一覧、本番の組み立て、起動スクリプトの export、warning の文と説明、それを固めるテストの更新(AC2、AC3、AC4)
3. S3(orchestrator が打つ): 実機の確かめ(AC6)。S1 の hook を写した一時の repo で打つ。hook が値を見ないと分かったら、D1 を見直して計画に戻る
4. S4(implementer): leader の雛形の 1 項とテスト(AC5)、文書(AC7)
5. S5(implementer、sonnet): 記録(AC8)。S3 の結果で RESOLVED にするかを決める

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、shellcheck、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC9。仕様の印の数(`[x]` 26 / `[ ]` 6、根拠の表は 25 / 6)は、F11 が挙げる FR-4 と FR-7 がどちらも `[x]` なので変わらないはず。変わるならその理由を書く
- Documentation drift to check: `AGMSG_SELF_NAME` を pane の環境として書いた文書(`/org`・`/director` の skill、`director-launch-cmd.md`、README、仕様)に 2 つ目の値の書き漏れがないか。`git-commit-strategy.md` の Org Runtime Commits の節と End-of-Session の節が食い違わないか
- Evidence to capture: run-verify の rc と時間、evidence の文書

## Test plan

- Unit tests: `tests/test-wip-checkpoint-hooks.sh` が、一時の git repo(feature ブランチ、汚れたファイル)で 2 つの hook を、値なし・`off`・`on`・`OFF`・空で打ち、コミットの有無と md の行を見る。main でコミットしないことも見る。Go: 一覧の中身と順、`TabCreate` が 2 つの `--env` を付ける、warning が 2 つの値と帰結を書く、起動スクリプトが 2 つを export する、leader の雛形の項
- Integration tests: `internal/cli` の本番の組み立てで `PaneEnv` が 2 つの値を持つこと、spawn と director start の stub の herdr が 2 つの `--env` を受けること
- Regression tests: 今の driver・spawn・director・prompts のテストが、期待の文を 2 つ目の値に合わせる以外は変えずに通る。手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は変えない
- Edge cases: 値が `OFF`(大文字)や空のときはコミットする。作業ツリーがきれいなら値によらず何もしない。`--env` の確かめが失敗したときの warning
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で、`bash tests/test-wip-checkpoint-hooks.sh`

## Risks and mitigations

- hook が pane の環境を見ない(Claude Code が hook に環境を渡さない、または dispatcher が消す): S3 で先に確かめ、見なければ D1 を見直す(たとえば hook が `HERDR_PANE_ID` と ralph の台帳を読む形)。`ralph-dispatch.sh` が環境を消さないことは S3 の前に読んで確かめる
- 古い herdr(`--env` なし)では値が届かず、WIP コミットが戻る: warning の 1 行で知らせる。現行の leader の手順 9 と 14 が push への混入を止める
- 座席が残した未コミットの変更が、コミットされずに作業ツリーに残る: これは意図した形で、leader の手順 9 の `git status --porcelain` が gate の前に拾う。gate のあとに残ったものは cleanup が断るので、人が気づく
- 実機の確かめで、claude のフォルダ信頼の項目が `~/.claude.json` に 1 つ増え、session の記録が `~/.claude/projects/` に残る(4a と同じ)。evidence に書く。ユーザーのグローバルの hook も一時の repo で走る
- 文書の写しの書き漏れ: check-sync と check-skill-sync が落とす

## Rollout or rollback notes

- 効くのは次の release の binary と、`ralph upgrade` で入る hook から。hook と binary のどちらかだけが新しいときは、今までどおり WIP コミットを作る(値を付けない binary、値を読まない hook)
- 戻すときはこの PR を revert する。値を読まない hook に戻れば、pane の環境に値が残っていても振る舞いは今までどおり

## Open questions

- PreCompact を `/compact` で起こせるか(空に近い会話で compact が断られるか)。S3 で見る

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
