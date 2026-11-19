# doctor-alias-herdr-guard

- Status: Approved
- Approved: 2026-11-19 sha256:2c2d8e5932b3
- Owner: Claude Code
- Date: 2026-11-19
- Related request: 下流の issue #343。`~/.config/zsh/.zshrc` の codex と claude の alias を `[[ -n $HERDR_PANE_ID ]] || alias codex=...` の形にして herdr の pane では定義しないようにしたが、`ralph doctor` の「Shell aliases (codex/claude)」は warn のまま。#340〜#344 の系列のレーン A が受け持つ(2026-11-19、ユーザーの決定で計画は推す案で自動承認)
- Related issue: #343
- Type: fix
- Branch: fix/doctor-alias-herdr-guard

## Objective

herdr の pane の中では定義されない alias を、doctor が「座席に効かない」と読めるようにする。herdr は pane の対話シェルに `HERDR_PANE_ID` を入れる。この変数を条件にして、pane の中では定義されない形に書いた codex と claude の alias は、warn ではなく info にする。Detail では、その alias は herdr の pane では定義されないと伝える。ほかの条件の alias と、条件のない alias の扱いは変えない。

## Scope

- `internal/cli/doctor_shell_alias.go`
  - 文の分け方(`shellAliasStatements`)で、文の前にあった区切り(`;`、`&&`、`||`、`|`、`&` などの連なり)を残す。今の呼び出し(`parseAliasStatements`、`shellAliasValueTokens`)の結果は変えない
  - `HERDR_PANE_ID` の判定の文を見分ける。pane の中で真になる形(`-n`)と偽になる形(`-z`)の 2 種類。書き方は `[[ … ]]`、`[ … ]`、`test …` の 3 つ。変数は `$HERDR_PANE_ID` と `${HERDR_PANE_ID}` の 2 つで、引用符の有無は問わない
  - 1 行の中の形: `-n` の判定の直後に `||` で続く alias の文と、`-z` の判定の直後に `&&` で続く alias の文。判定の文は、その行の並びの頭(前に何もないか `;`、または `then`、`else`、`do`、`{` で始まる)にあるときだけ数える。物理行の初めの文を並びの頭とみなすのは、前の物理行が閉じて終わったときだけにする。前の物理行が `&&`、`||`、`|` か、引用符の外の行末のバックスラッシュで終わったとき、または引用符が閉じないまま終わったときは、次の行の初めの判定を数えない(`false &&` の次の行の `[[ -n $HERDR_PANE_ID ]] || alias …` は、pane の中でも alias を定義する。Codex の計画助言の指摘 1)
  - if の形: rc ファイルごとに if の入れ子を追う。`if <-z の判定>` の then の側と、`if <-n の判定>` の else の側にある alias は、pane の中では定義されない。1 行の `if …; then alias …; fi` も、複数行にまたがる if も同じ仕組みで読む。if を条件付きと読むのは、`if` と `then` の間が判定の 1 文だけで、その文が `;` か改行で閉じるときに限る(`&&`、`||`、`|`、行末のバックスラッシュで続く条件や、2 文以上の条件は、pane の中でも then か else の側が走りうる。consult の指摘)。`elif` が来た if、`HERDR_PANE_ID` の判定でない if、条件がこの形でない if は、その中の alias を条件付きとは読まない。ファイルの終わりで if が閉じていないか、対応する if のない `fi` があれば、そのファイルの if の形による判定をすべて取り消す(1 行の中の形の判定は残す)
  - `shellAliasDef` に、pane の中では定義されないことを示す欄を足す
  - `checkShellAliases`: 条件付きの finding は warn にしない。条件付きの finding だけなら info にし、Detail には別の文で「herdr の pane では定義されない」と書く。条件のない finding が 1 つでもあれば今までどおり warn で、warn の文には条件のない finding だけを並べる。codex の文の最後の案内(alias を外すか、alias を読まない rc で herdr を起動する)に、`HERDR_PANE_ID` の条件を付ける手を足す
- テスト: `internal/cli/doctor_shell_alias_test.go`
- 文書
  - `docs/recipes/codex-seat-permissions.md` と `templates/base/docs/recipes/codex-seat-permissions.md` の前提の「Shell aliases break every codex spawn」の項に、条件付きの alias を 3 つ目の回避策として足す。herdr 0.7.5 で `HERDR_PANE_ID` が入ることを確かめたと書き、doctor の severity の説明に「条件付きは info」を足す
  - `docs/recipes/isolated-herdr-server.md` と templates の写しの「Shell aliases」節に 1 文足す
  - org skill の 4 写し(`.claude/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`.agents` の 2 写しは `scripts/sync-skills.sh` で作る)の「shell alias に注意」に、同じ回避策と doctor の扱いを足す

## Non-goals

- fish の書き方(`set -q HERDR_PANE_ID; or alias …`、`if not set -q HERDR_PANE_ID … end`)。今までどおり warn
- `||` や `&&` のあとで改行して次の物理行に alias を書く形。scanner は物理行ごとに読むので、今までどおり warn
- `!` で打ち消した判定、`[[ -v HERDR_PANE_ID ]]`、`${HERDR_PANE_ID:-}` などの別の書き方。今までどおり warn
- `{ … }` で囲んだ alias(`[[ -n $HERDR_PANE_ID ]] || { alias codex=…; alias claude=…; }`)。中の 1 文目だけを数えると、同じ行で片方が info、片方が warn になって人を迷わせる。数えずに warn のままにし、recipe と skill には 1 つの alias を 1 行に書く形を示す
- rc の前の行で `HERDR_PANE_ID` を書き換える(`unset` や代入)形を見分けること。その rc では判定が pane の中でも偽になりうるが、doctor は読まない
- ralph が pane に渡すほかの変数(`AGMSG_SELF_NAME`、`RALPH_WIP_CHECKPOINT`)を条件にする形
- source 先のファイルを追うこと、関数の中の alias の扱い、heredoc の中身を読み分けること
- `ralph.toml` の役割のモデルを読んで severity を変えること(tech-debt の 136 行の項)

## Assumptions

- herdr は pane の対話シェルに `HERDR_PANE_ID` を入れる。下流で herdr 0.7.5 の `herdr workspace create` の pane が `wD:p1` を返したと報告があり、ralph 自身も `ralph org approve` の仕掛け(`internal/org/approve.go` の `approveTripwire`)でこの変数に頼っている。座席と、herdr の中の headless の director(`ralph org director start`)は、どちらも herdr の pane のシェルで起動するので、rc を読む時点で変数がある
- `[org.director].launch_cmd` で立てる director は、`#!/bin/sh` の起動スクリプトから agent を絶対パスで `exec` する(`internal/org/director_cmd.go` の 322〜337 行)。alias はもともと効かないので、`HERDR_PANE_ID` がないことは問題にならない
- herdr の pane の値には空白が入らない。そのため、引用符のない `[ -n $HERDR_PANE_ID ]` も pane の中では真、`[ -z $HERDR_PANE_ID ]` は偽になる。pane の外での真偽は、この Check の判定に関わらない

## Affected areas

- `internal/cli/doctor_shell_alias.go`、`internal/cli/doctor_shell_alias_test.go`
- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`
- `docs/recipes/isolated-herdr-server.md`、`templates/base/docs/recipes/isolated-herdr-server.md`
- `.claude/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`

## Visual review

- ページ: `.harness/state/plan-visual/doctor-alias-herdr-guard.html`。かんたんな図 2 枚(`overview`、`easy-forms`)と、詳しい図 3 枚(`detail-overview`、`detail-inline`、`detail-ifstate`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 5 枚の図を `shot` で撮って読んだ。直したのは 3 点。図 1 の「これから」の文字と人のアイコンの重なり、ファイルの地図で 2 本の矢印が S1・S2 のバッジに刺さっていたこと、承認の前にの欄で `-n`・`-z` の折り返しが崩れていたこと。撮り直して、どれもきれいに出ることを確かめた

## Design decisions

Critical forks: None。次の 4 点は、ユーザーの指示(系列の計画は推す案で自動承認)に従って自律で選んだ。

1. 複数行の if も扱う。issue の「やること」に `if [[ -z $HERDR_PANE_ID ]]; then ... fi` が挙がっており、2 つの alias を 1 つの if で囲む書き方はよくある。if の入れ子を追う仕組みは 1 行の if にも要るので、複数行に広げる手間は小さい。黙る向きの誤り(本当は効く alias を info にする)を避けるため、if の条件は判定の 1 文だけのものに限り、if の対応が崩れたファイルでは if による判定をすべて取り消す
2. 1 行の中の形は、判定の直後の 1 つの alias の文だけを数え、判定が並びの頭にあることを求める。`foo && [[ -n $HERDR_PANE_ID ]] || alias …` は、foo が偽なら pane の中でも alias が定義されるので、条件付きとは読まない。`|` と `&` は条件とは読まない
3. 見分ける判定の書き方は、`[[`、`[`、`test` と `-n`、`-z`、2 つの変数の書き方の組み合わせに限る。書き方を広げるほど読み違いが増えるので、recipe と skill には doctor が読む形を示す
4. 条件付きの finding の severity は info にする(pass にはしない)。herdr の版によって変数が入らないことがありうるので、Detail に「herdr の pane では定義されない」と書いて人の目に残す

## Acceptance criteria

- [ ] AC1: issue の 2 行(`[[ -n $HERDR_PANE_ID ]] || alias claude="claude --model opus …"` と `[[ -n $HERDR_PANE_ID ]] || alias codex="codex -m gpt-6-astra …"`)だけがある rc で、herdr があっても Shell aliases が warn にならず info になる。Detail は 2 つの alias が herdr の pane では定義されないと言う(fixture のテスト)
- [ ] AC2: 次の形も条件付きとして info になる。`[ -n "$HERDR_PANE_ID" ] || alias …`、`test -n "$HERDR_PANE_ID" || alias …`、`[[ -n ${HERDR_PANE_ID} ]] || alias …`、`[[ -z $HERDR_PANE_ID ]] && alias …`、1 行の `if [[ -z $HERDR_PANE_ID ]]; then alias …; fi`、複数行の `if [[ -z $HERDR_PANE_ID ]]; then` 〜 `fi` の then の側、複数行の `if [[ -n $HERDR_PANE_ID ]]; then` 〜 `else` 〜 `fi` の else の側、`foo; [[ -n $HERDR_PANE_ID ]] || alias …`(テスト)
- [ ] AC3: 次の形は今までどおり warn になる。条件のない alias、`command -v codex >/dev/null && alias …`、`if command -v codex …; then alias …; fi`、向きが逆の `[[ -n $HERDR_PANE_ID ]] && alias …` と `[[ -z $HERDR_PANE_ID ]] || alias …`、並びの頭にない判定(`foo && [[ -n $HERDR_PANE_ID ]] || alias …`)、前の物理行から続く判定(1 行目が `false &&` で 2 行目が `[[ -n $HERDR_PANE_ID ]] || alias …` の形と、1 行目が `false && \` の形)、パイプ(`[[ -n $HERDR_PANE_ID ]] | alias …`)、`-z` の if の else の側、`elif` の来た if の中、条件が判定 1 文でない if(`if [[ -n $HERDR_PANE_ID ]] && [[ -n $FOO ]]; then :; else alias …; fi`、`if [[ -z $HERDR_PANE_ID ]] || foo; then alias …; fi`、`if [[ -z $HERDR_PANE_ID ]]; foo; then alias …; fi`、1 行目が `if [[ -n $HERDR_PANE_ID ]] &&` で 2 行目が `foo; then` の複数行の if)、`{ … }` の中の alias(`[[ -n $HERDR_PANE_ID ]] || { alias codex=…; }`)、別の変数(`$HERDR_PANE`、`$OTHER_ID`)、閉じていない if と対応のない `fi` があるファイルの if の中(テスト)
- [ ] AC4: 条件付きの codex の alias と条件のない codex の alias が両方ある rc では warn になり、warn の文には条件のない方だけが並び、条件付きの方は別の文に出る(テスト)
- [ ] AC5: 条件付きの finding の Detail にも alias の値は出ない(`TestCheckShellAliases_AliasValueNeverAppearsInDetail` と同じ確かめを新しい文にも当てる)
- [ ] AC6: 今あるテストがすべて通る。テストの期待値を弱めない
- [ ] AC7: recipe(`codex-seat-permissions.md` と `isolated-herdr-server.md`)と org skill の 4 写しに、この書き方と doctor の扱いが載る。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る

## Implementation outline

1. S1 文の区切りと判定の読み: `shellAliasStatements` の内側で区切りの演算子を残す形を作り(今の関数はそれを包んで語だけを返す)、`HERDR_PANE_ID` の判定の分類と、1 行の中の形と if の入れ子の追跡を書く。`scanShellAliasFile` は if の状態をファイルごとに持ち、継続行を読むあいだは状態を確定させない。`shellAliasDef` に条件付きの欄を足す。AC2・AC3 の読みの部分を単体テストで確かめる — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ShellAlias'`
2. S2 severity と Detail: `checkShellAliases` で条件付きの finding を warn から外し、別の文を作る。codex の文の最後の案内に条件を付ける手を足す。AC1・AC4・AC5 を fixture のテストで確かめる — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ShellAlias'`
3. S3 文書: recipe の 2 つ(root と templates)と org skill の 2 つを書き、`scripts/sync-skills.sh` で `.agents` の写しを作る — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync)
- Spec compliance criteria to confirm: AC1〜AC7。issue #343 の受け入れ条件 3 つ(条件付きだけの rc で warn にならない、ほかの条件は warn のまま、recipe と skill の 4 面に載る)
- Documentation drift to check: `docs/recipes/codex-seat-permissions.md` の severity の説明と、org skill の「Shell aliases」の severity の説明が、コードの判定と合うこと。`internal/cli/doctor.go` の Check 8b のコメント
- Evidence to capture: 条件付きの rc での doctor の Shell aliases の行(一時の HOME と rc で `ralph doctor` を打った出力)

## Test plan

- Unit tests: `shellAliasStatements` が区切りを残しても今の語の列が変わらないこと。判定の分類(6 つの書き方 × 2 つの変数 × 引用符の有無、別の変数、`!` 付き)。if の入れ子(then の側、else の側、elif、入れ子の別の if、閉じていない if、対応のない fi)
- Integration tests: `checkShellAliases` を fixture の rc で打ち、status と Detail を確かめる(AC1〜AC5)
- Regression tests: 今の 72 本のテスト(`TestCheckShellAliases_AliasAfterAndAnd_Detected`、`TestCheckShellAliases_AliasInsideThenClause_Detected` を含む)
- Edge cases: 継続行(引用符の閉じていない alias の値)をまたぐ if、同じ行に判定と alias が 2 組ある行、条件付きで値に flag のない alias(harmless のまま)、claude の `--permission-mode` を足す条件付きの alias(warn にならない)、herdr がない環境(条件付きでも条件なしでも info)
- Evidence to capture: `go test ./internal/cli/ -run ShellAlias -v` の要約、`./scripts/run-test.sh` の結果

## Risks and mitigations

- 本当は座席に効く alias を info にしてしまう(黙る向きの誤り): 判定の書き方を語の並びで厳密に照らし、判定が並びの頭にあることと直後の 1 文であることを求める。scanner は未完の alias 文のほかは物理行を連結しないので、前の物理行が演算子か行末のバックスラッシュで終わったか、引用符が閉じていないかを持ち越し、そのときは次の行の初めを並びの頭と読まない。if の対応が崩れたファイルでは if の判定を取り消す。向きの逆、別の変数、パイプなどの否定のテストを並べる
- herdr の版によっては `HERDR_PANE_ID` を入れない: Detail と recipe に「herdr 0.7.5 で確かめた」と書き、severity は pass ではなく info に留める。ralph の approve の仕掛けも同じ変数に頼っているので、入らない herdr ではそちらも働かない
- heredoc や文字列の中の `if`/`fi` で入れ子を読み違える: 対応が崩れればファイルの if の判定を取り消す。崩れずに読み違える形は残りうるので、Known gaps に書く
- `shellAliasStatements` の変更で今の読みが変わる: 今の関数は語だけを返す包みとして残し、`TestShellAliasStatements` と `TestShellAliasValueTokens` を変えずに通す

## Rollout or rollback notes

- doctor の 1 つの Check の判定だけが変わる。設定と記録の形は変わらない。戻すときはこの PR を revert する
- 下流の skill と recipe の文面は `ralph upgrade` で届く。binary だけを更新した下流では、doctor は新しい判定になるが skill の説明は古いまま
- release ノートの候補: 「doctor の Shell aliases が、`HERDR_PANE_ID` を条件にして herdr の pane では定義されない alias を info にする」

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件: [HIGH] 物理行の初めを並びの頭とみなすと、前の行の `&&` や行末のバックスラッシュから続く判定で、pane の中でも定義される alias を info にしてしまう(codex が `HERDR_PANE_ID=wD:p1` の `zsh -f` で確かめた)。計画に入れた: Scope の 1 行の中の形で前の物理行の終わり方を持ち越す、AC3 に 2 つの反例を足す、Risks に書く(memory の決まりで、計画助言の修正提案は聞かずに入れる)。そのあと consult(consult-plan-alias343、判定「直してから進める」)の指摘 2 点も入れた: if の条件は `if` と `then` の間が判定の 1 文だけのときに限る(AC3 に 4 つの反例)、`{ … }` の中の alias は数えない(Non-goals と AC3)
- [x] Plan approved — 2026-11-19、系列の指示(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [x] Implementation started
  - S1(faef9b12、implementer): 確認コマンドが exit 0(orchestrator も `-count=1` で打ち直して ok)。handoff からのずれは 8 点で、どれも印を付けない向きに厳しくしたもの(条件の直後の `else`・`elif`・`fi` も対応の崩れ、前の行から続く文の keyword は読まない、`if … &` ⏎ `then` は閉じていない条件、`|&` や `;;` などの行末の演算子も閉じていない、空行とコメント行は前の行の状態を変えない、fish は拡張子で見分けて読まない、など)。zsh 5.9 で `HERDR_PANE_ID=wD:p1` を入れて fixture を読ませ、印の表は pane の中で alias を定義しないこと、印のない表は定義するか Non-goals の形であることを確かめた。mutant 23 個がすべて落ちる。機械的セルフチェックは 5 項目とも clean か fixed
  - AC3 の「`elif` の来た if の中」は、elif より後ろの alias と読む。`-z` の if の then の側で elif より前にある alias は、pane の中では走らないので印を付ける(`TestScanShellAliasFile_ElifStopsMarkingFromThatPointOn`)
  - S2(1af2fcee、implementer): 確認コマンドが exit 0(orchestrator も `-count=1` で打ち直して ok)。ずれは 1 点で、値を読み切れていない def は印が付いていても warn に残す(黙る向きを避ける、`TestCheckShellAliases_IncompletePaneGuardedAlias_StillWarns`)。案内の 3 つ目の手は codex の文と、claude の `--permission-mode` の文に付けた(`shellAliasRemedyClause`)。mutant 8 個がすべて落ちる。機械的セルフチェックは 5 項目とも clean か fixed。AC1 の fixture の Detail は status info で、2 つの alias の file:line と HERDR_PANE_ID の説明を含み、値を含まない
  - 残した穴(Known gaps に書く): 判定は引用符を外した語で見るので、引用符付きの `"[["` もガードと読む。前からある誤検知(閉じていない引用符の中の行や、行末のバックスラッシュの次の行の alias も def として報告する)は触っていない
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
