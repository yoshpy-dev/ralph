# trial-docs-wrapup

- Status: Approved
- Approved: 2026-10-23 sha256:932bafc59d5c
- Owner: Claude Code
- Date: 2026-10-23
- Related request: 試行の所見を直す系列の 8 本目(最後)「文書のまとめ」(ユーザーの依頼「2. 修正可能なものは修正してください」、2026-10-21)
- Related issue: N/A
- Type: docs
- Branch: docs/trial-docs-wrapup

## Objective

stage 9 の試行の所見のうち、文書で直せるものを直して系列を終える。分離したヘッドレスの herdr server で org runtime を動かす手順を recipe にし(F1・F2)、新しい repo の folder trust を誰がいつ済ませるかを `/org` skill と `/director` skill に書き(folder trust)、`/org` skill の RESULT の例を reviewer の雛形に合わせ、gate と regate が「数分」で終わるという文を直す。記録では、「fix/busy-leader」を「#253」に直し、試行の記録にあって所見の表にも tech-debt にもない watcher の時間切れ 6 回を行にし、仕様と根拠の表に系列が終わったことと開いたまま残る行を書く。

## Scope

- recipe `docs/recipes/isolated-herdr-server.md`(英語。`templates/base/docs/recipes/isolated-herdr-server.md` に同じ中身を置く)
  - 分離する理由: ユーザー自身の herdr server が動いていることがある。herdr は `HOME` では設定と状態の置き場所を変えず、`HERDR_SOCKET_PATH` だけを変えた server はユーザーの `session.json` から workspace を戻し、ユーザーのログに書く。`XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` の 3 つを専用の場所に向ける。unix socket のパスの長さの上限(約 104 文字)があるので、短い `/tmp/<名前>` の下に置く
  - 最小の環境: `env -i` で空にしてから渡す変数(`HOME` は本当の HOME。macOS の claude は keychain を `$HOME/Library/Keychains` で読む)。session の中から起動すると `CLAUDE_CODE_*` や端末の変数が pane に漏れる
  - F1: `XDG_CONFIG_HOME` を移すと、`~/.config/git/config` と `~/.config/gh` に設定を置く利用者の git と gh が pane の中で設定を失う。server の環境に `GIT_CONFIG_GLOBAL` と `GH_CONFIG_DIR` を足す
  - F2: herdr の pane は login の bash を起こし、macOS の `/etc/profile` の `path_helper` が `/usr/bin` を server の `PATH` の前に出す。`BASH_ENV` は効かず、`PATH` を export するファイルを `CLAUDE_ENV_FILE` で指すと claude の Bash tool に効く
  - shell の alias: `SHELL` で pane の shell を選ぶ理由と、`ralph doctor` の「Shell aliases (codex/claude)」の Check。利用者固有の alias やインストール先(mise のパスなど)は書かず、一般の形にする
  - 任意の分離: agmsg の写しと `[org].agmsg_home`、通知の `osascript` の shim
  - 台帳も分ける: 専用の clone か scratch の git repo で動かす。ほかの org が使う repo の linked worktree では、台帳が main worktree のルートに決まって共有される。`ralph org watch --all --detach` と `ralph org start` は、その台帳で仕事を受け付けている見張りがあれば、server や環境を比べずにそれを使う(`ensureOrgWatch`、`internal/cli/org_watch.go`)。最初の start の前に、その repo の `ralph status` が見張りの動いていないこと(`org watch: not running`)を言うのを確かめ、見張りが分離した環境から起動されるようにする(試行は専用の clone、#253 の実機は scratch の repo を使った)
  - ralph、herdr、見張り(`ralph org watch --all --detach`)を同じ環境(`HERDR_SOCKET_PATH` と `PATH`)で打つこと
  - start の前の確かめ: server のプロセスが開いているファイルが専用の場所の下だけであること(`lsof`)、捨ての workspace の pane で claude を起動し、その Bash tool で `command -v`(git、gh、ralph、claude、shim を置いたなら osascript)、`git config user.name`、`gh auth status` を打つこと
  - folder trust: 専用の環境の中で、新しい repo に一度だけ答えること(`/org` skill の前提を指す)
  - 片付け: org を閉じ、その台帳の見張りを `ralph org watch --all --stop` で止め、`ralph status` が `org watch: not running` を言い、専用の場所を名前に含むプロセスが残っていないことを確かめてから `herdr server stop` を打ち、専用の場所を消す(見張りは org がなくなっても止まらない。試行の Cleanup の順)。止める頼みが 30 秒で終わらなかったときは、見張りが止まるまで専用の場所を消さない。やり直すときは専用の `session.json` を消し org の id を変える。分離の外に残るもの(`~/.claude.json` の trust の記録、`~/.claude/projects/` の session の記録、agmsg の run の印)。gh と git の設定や token を evidence に写さない
  - `/org` skill の前提から 1 行で指す
- folder trust の文書(コードの確かめは入れない)
  - `/org` skill の前提に「フォルダ信頼(repo ごとに 1 回)」の項目を足す。人が、最初の org を立てる前に repo の main のチェックアウトで claude を一度起動して「Yes, I trust this folder」を選ぶ。claude は trust したディレクトリの下を聞き直さないので、`.claude/worktrees/` の下の機能の worktree は聞かれない(試行で確かめた)。repo の外の `--cwd` は別に要る。codex の trust の画面は確かめていないので、`codex trust .`(`docs/recipes/codex-setup.md`)を指す。画面で止まった座席は herdr が blocked と報告し、`ralph org send` は届かない。`ralph org read` で画面を読み、人が pane で答える
  - 前提の「bypass 初回承諾」の項目の主語を「人が」にそろえる
  - `/director` skill の「してはいけないこと」に、trust と bypass の画面にキーを送って答えないこと、人に頼むことを足す
- `/org` skill の typed protocol の節の RESULT の例を、reviewer の雛形(`internal/org/prompts/reviewer.md` の RESULT の例とレポート契約)に合わせる。`GATE: pass`・`HEAD:`・`BASE:` のヘッダを足し、EVIDENCE を `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` の形にする。例が typed protocol の検査を通ることと、reviewer のレポートの名前の形であることを Go のテストで固定する
- gate と regate の所要時間の文
  - `/org` skill の動詞の表の `gate` の行と「merge してよい順」の regate、`/director` skill の「merge のあとの取り直し」の 1、leader の雛形 `internal/org/prompts/leader.md` の機能ごとの org の手順 12 の「数分かかる」を、「repo のテストの量とマシンの負荷によっては数十分かかる」の趣旨の一般の文にする。scaffold に配る文なので、試行の数値は書かない
  - `internal/org/prompts_test.go` の固定の文を合わせる
- 記録
  - 「fix/busy-leader」を「#253」に直す: 仕様(Status の一覧、FR-5 の印、NFR ツール非依存の印)、根拠の表(所見の節の前置き、F7 の行、FR-5 の行、NFR ツール非依存の行、段ごとの PR、「確かめていないこと」)、tech-debt(F7 の RESOLVED の注、ファイルの大きさの行)。PR の一覧を 36 本にする(`gh pr view 253` で `MERGED` を読む)
  - 根拠の表の F1・F2・folder trust の行を、この PR で直したと書き、tech-debt の列を RESOLVED の行にする。tech-debt の F1・F2 の行と folder trust の行を、ほかの RESOLVED の行と同じ形(取り消し線と `<!-- RESOLVED ... -->` の注)にする
  - 試行の `watch.log` の「watcher error … invoke claude: claude: timed out」6 回を、所見の表の新しい行と tech-debt の新しい行にする。watcher は `claude -p` を `watcherInvokeTimeout`(60 秒、`internal/org/watcher.go`)で打ち切る。6 回は高い負荷のあいだに起き、原因は確かめていない。直さない
  - この PR は系列の最後なので、ブランチ名 `docs/trial-docs-wrapup` で書き、系列の最後の PR なので番号は書き足さないと書く(番号は `gh pr list --state merged --head docs/trial-docs-wrapup` で引ける)。仕様の一覧の本数(36 本)には数えない
  - 仕様の Status の節と根拠の表の「予定の順」に、系列が終わったことと、開いたまま残る行(leader への ALERT、Codex の leader、watcher の時間切れ)を書く
  - 試行の gate と regate の所要時間(1375〜1990 秒、負荷の低い regate は 792 秒)は、根拠の表に、この PR が「数分」を直した根拠として書く

## Non-goals

- `ralph org start` が spawn の前に trust を確かめて断るコード(claude の非公開のファイル `~/.claude.json` を読むことになる。`ralph doctor` も codex の trust を Go から読んでいない(`internal/cli/doctor.go:314`))
- watcher の時間切れを直すこと(`watcherInvokeTimeout` を変える、設定にする)。原因を測ってから決める
- Codex の leader に `ralph org send` が届くようにすること(#253 が tech-debt に残した行)
- leader への ALERT の時間切れの原因を調べること(tech-debt の「The watch's ALERTs to the leaders failed」の行)
- tech-debt の古い RESOLVED の注に残るブランチ名(#240・#243・#244・#246・#247 のもの)を番号に書き換えること
- recipe の手順を通しで実機で打ち直すこと。手順は試行と直近の実機の記録(`docs/evidence/org-trial-2026-10-20.md`、`docs/evidence/busy-leader-2026-10-23.md` など)から写す
- `AGENTS.md` の repo map の `docs/recipes/` の行を書き直すこと

## Assumptions

- recipe は `docs/recipes/` と `templates/base/docs/recipes/` で同じ中身にする。`scripts/check-sync.sh` はこの 2 つを比べ、違ってよいのは `adding-a-language-pack.md` だけ
- skill の 4 つの写しは `scripts/sync-skills.sh` を 2 回(root と templates/base)打って揃え、`scripts/check-skill-sync.sh` で確かめる
- leader の雛形 `internal/org/prompts/leader.md` は go:embed の 1 つだけで、templates に写しはない
- 試行で確かめた trust の動き(trust したディレクトリの下の git worktree では画面が出ない)は、claude 2.1.291 のもの。版が変われば変わりうる

## Affected areas

- `docs/recipes/isolated-herdr-server.md`(新規)、`templates/base/docs/recipes/isolated-herdr-server.md`(新規)
- `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`
- `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`
- `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- RESULT の例を固定するテスト(`internal/org/report_names_sync_test.go` に足すか、同じパッケージの新しいテストのファイル)
- `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/tech-debt/README.md`

## Visual review

- Page: `.harness/state/plan-visual/trial-docs-wrapup.html`(かんたん版の図 3 枚 `overview`・`easy-texts`・`easy-records`、承認の前に、折りたたみの中に `detail-overview`(変えるファイル)と `detail-records`(記録の書き換え))
- Lint: `./scripts/plan-visual.sh lint` が 0 で終わった
- Self-check: ページ全体と 5 枚の図を `shot` で撮って読んだ。`detail-overview` の矢印の札がグループの縁に重なっていたので、すき間に移して撮り直した

## Design decisions

Critical forks: None。consult(consult-plan-wrapup)が 4 点とも次の案でよいと判定した。

- D1 folder trust は文書だけにする。`start` で確かめるには claude の `~/.claude.json` を読むことになり、その形は claude が決めて公開していない。tech-debt の行の trigger は、文書にするか start で確かめるかのどちらでもよいと書く
- D2 所要時間は一般の言い方にする。skill と leader の雛形は `ralph init` で downstream に配られ、テストの量は repo ごとに違う。試行の数値は記録にだけ書く
- D3 この PR はブランチ名で書き、番号を書き足さない。仕様の一覧は「すべてが merge 済みであることを `gh pr view` で確かめた」と書くので、merge の前の PR を数に入れると偽になる。PR を作ったあとに番号を書くコミットを足す案は、一覧の本数の文とぶつかる
- D4 recipe は templates にも置く。scaffold に配る `/org` skill が recipe を指すので、置かないと downstream でリンクが切れる。中身は利用者に固有のもの(alias、インストール先)を除いて一般化する
- D5 RESULT の例は reviewer の雛形と同じヘッダを持たせ、テストで固定する。#244 の sync-docs が残した食い違い(例は `self-review-foo.md`、雛形は `self-review-` で始まる名前を使わないと書く)が、また開かないようにする

## Acceptance criteria

- [x] AC1 recipe: `docs/recipes/isolated-herdr-server.md` と `templates/base/docs/recipes/isolated-herdr-server.md` があり、中身が同じ(`cmp`)。Scope の recipe の項目(分離する理由、最小の環境、F1、F2、shell の alias、任意の分離、台帳を分けることと最初の start の前に見張りが動いていないことの確かめ、同じ環境で打つこと、start の前の確かめ、folder trust、片付け)をすべて含む。片付けは、見張りを止めて止まったことを確かめる手順が `herdr server stop` と専用の場所の削除より前にあり、止まらなかったときは消さない分岐を持つ。利用者に固有のパス(`mise`、`/tmp/ot`、利用者の名前)を含まない。`/org` skill の前提から指されている
- [x] AC2 folder trust: `/org` skill の前提に folder trust の項目があり、人が答えること、main のチェックアウトで一度、`.claude/worktrees/` の下は聞かれない(試行で確かめた)、repo の外の `--cwd`、codex は確かめていないことと `codex trust .`、画面で止まった座席の見え方と答え方を書く。bypass の項目の主語が人。`/director` skill の「してはいけないこと」に、trust と bypass の画面にキーを送らないこと。4 つの写しが揃う(`scripts/check-skill-sync.sh` が 0)
- [x] AC3 RESULT の例: `/org` skill の RESULT の例が `GATE: pass`・`HEAD:`・`BASE:` のヘッダを持ち、EVIDENCE が `docs/reports/review-` で始まる。Go のテストが、例を skill から読み、typed protocol の検査(`internal/org/protocol` の `Parse` と `Validate`)を通ること、`GATE`・`HEAD`・`BASE` のヘッダがあること、EVIDENCE が reviewer のレポートの名前の形であることを確かめる。例を `self-review-foo.md` に戻すとテストが落ちる
- [x] AC4 所要時間: `grep -rn '数分' .claude/skills/org .claude/skills/director .agents/skills/org .agents/skills/director templates/base/.claude/skills/org templates/base/.claude/skills/director templates/base/.agents/skills/org templates/base/.agents/skills/director internal/org/prompts` が gate と regate について 0 件。新しい文が、テストの量とマシンの負荷で数十分かかりうることを言う。`internal/org/prompts_test.go` の固定の文が新しい文に合う
- [x] AC5 記録: `grep -rn 'fix/busy-leader' docs/specs docs/evidence docs/tech-debt` が 0 件(main の 3c1cd14b では仕様 3 件、根拠の表 6 件、tech-debt 2 件)。仕様の一覧が 36 本で、#253 を `gh pr view 253` で読んだ結果を根拠の表に書く。F1・F2・folder trust の行がこの PR で直したと書き、tech-debt の 2 行が RESOLVED。watcher の時間切れの行が所見の表と tech-debt にある。仕様の Status の節と根拠の表の「予定の順」が系列の終わりと開いたまま残る行を書く
- [x] AC6 `./scripts/run-verify.sh` が 0 で終わる。`go test ./internal/org/... -timeout 30m` が、手元の Go と `GOTOOLCHAIN=go1.25.8` の両方で通る

## Implementation outline

1. S1(AC1 の recipe、implementer): recipe を書き、templates に同じものを置く
2. S2(AC1 の前提からの 1 行、AC2〜AC4、implementer): `/org` skill の前提から recipe を指す 1 行、folder trust の項目、bypass の主語、`/director` skill の禁止、RESULT の例とそのテスト、「数分」の 4 か所と固定の文。skill は sync-skills.sh で揃える
3. S3(AC5、implementer sonnet): 記録の書き換え(#253、36 本、F1・F2・folder trust、watcher の時間切れ、系列の終わり)。`gh pr view 253 --json state,mergedAt` を読む
4. run-verify、post-implementation pipeline

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`(gofmt、go vet、golangci-lint、shellcheck、`scripts/check-sync.sh`、`scripts/check-skill-sync.sh`)
- Spec compliance criteria to confirm: AC1〜AC6。仕様の項目の印(`[x]`/`[ ]`)は変えない(この PR は FR と NFR の判定を動かさない)。根拠の表の判定の列も変えない
- Documentation drift to check: recipe の中身が試行と実機の記録(`docs/evidence/org-trial-2026-10-20.md` の Setup と F1・F2、`docs/evidence/busy-leader-2026-10-23.md` の「分離のしかた」)と食い違わないこと。folder trust の文が試行の記録(Setup の Trust)と食い違わないこと。RESULT の例が reviewer の雛形と食い違わないこと。「数分」が残っていないこと。「fix/busy-leader」が記録に残っていないこと
- Evidence to capture: verify の report に、`cmp` の結果、grep の件数(「数分」、「fix/busy-leader」、利用者に固有のパス)、`gh pr view 253` の出力の要点

## Test plan

- Unit tests: RESULT の例のテスト(新規)。`internal/org/prompts_test.go` の leader の雛形の固定の文(既存のテストの文を変える)
- Integration tests: なし(文書の系列)
- Regression tests: `go test ./internal/org/... ./internal/cli/... -timeout 30m`、`GOTOOLCHAIN=go1.25.8` でも。`./scripts/run-test.sh`
- Edge cases: RESULT の例のテストが、例の囲みを見つけられないときに落ちること(skill の見出しや囲みを動かしたら気づく)。例を `self-review-foo.md` に戻した mutant と、`GATE:` を消した mutant で落ちること
- Evidence to capture: test の report に、mutant の結果と、両方の Go の版の結果

## Risks and mitigations

- recipe の手順が実機で打ち直していないために古い。手順は直近の実機の記録(2026-10-22〜23 の 4 回)と試行から写し、確かめた版(herdr 0.7.5、Claude Code 2.1.291、agmsg 1.5.1)を recipe に書く。「確かめていないこと」に、recipe を通しで打っていないことを書く
- trust の動きが claude の版で変わる。試行で確かめた版を書き、変わりうると書く
- 記録の書き換えで、行番号で指す所がずれる。根拠の表は tech-debt の行を最初のセルの先頭の数語で指す決まりなので、取り消し線を付けた行の指し方(`~~` 付き)を他の RESOLVED の行に合わせる
- skill の 4 写しの片方だけを直す。sync-skills.sh を 2 回打ち、check-skill-sync で確かめる
- director の skill のテスト(`internal/org/director_skill_sync_test.go`)が節の構成を固めている。足すのは箇条の 1 項目で、節は足さない。test で確かめる

## Rollout or rollback notes

- 文書とテストと leader の雛形の文だけの変更で、動きは変わらない。leader の雛形の文は新しい binary の spawn から効く。戻すときは PR を revert する
- release のノート: leader の雛形の文が変わる(gate の所要時間)。挙動の変更はない。この PR の題は `docs:` なので changelog には出ない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- Codex plan advisory(2026-10-23): 2 件。[HIGH] 台帳と見張りを分離していない、[MEDIUM] 片付けで見張りを止めていない。どちらも Scope の recipe の項目と AC1 に足した(ユーザーの常設の指示により聞かずに直した)
- 承認: ユーザーの常設の指示「作成したプランは自動承認します」により自動承認した
- [x] Implementation started
- 実装(2026-10-23): S1 は e9884d9b・493e4ce8・e07bf30b(recipe と写し)、S2 は be8acda8(skill の 8 つの写し、leader の雛形、`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`)、S3 は 63e629a3(仕様・根拠の表・tech-debt)。S3 のあと、仕様の Status の FR-10 の理由を根拠の表の FR-10 の行の言い方にそろえた(orchestrator の 1 文の直し)。watcher の時間切れの行は、計画の「高い負荷のあいだに起き」とせず、負荷の読みがある時刻だけを書いた(03:38Z〜03:52Z の 4 回には読みがない)
- [x] Review artifact created
- self-review(2026-10-23、58ca93cc): Merge yes、LOW 8・INFO 3。5ac0af89 で直し、再レビュー(16cc451b)の LOW 2・INFO 1 を f15b696c で直し、最後の再レビュー(edf04ef7)は新しい LOW 以上なし。INFO の N4(template の recipe が出典にする `docs/evidence/` は scaffold に配られない)は既存の 3 つの recipe と同じ形なので、この PR では直さない
- [x] Verification artifact created
- verify(2026-10-23、c6d98b0a): pass、INFO 4。INFO-1(`ps` が環境を読めないとき)、INFO-2(RESULT の例の SUMMARY を雛形の severity 別件数の形に)、INFO-3(repo の外の `--cwd` の信頼は推論で未確認と書く)、INFO-4(この印)を、tester の前に直した
- [x] AC6 の run-verify: f310e994 で rc 0(`docs/evidence/verify-2026-10-23-050654.log`、gitignore の対象)。`GOTOOLCHAIN=go1.25.8 go test ./internal/org/...` は ok
- [x] Test artifact created
- test(2026-10-23、6053ab00): pass。run-test.sh は rc 0、`go test ./internal/org/... ./internal/cli/...` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で rc 0、新しいテストの変異 8 つはすべて落ちた。テストの穴は、skill の 3 か所の「数十分」を固定するテストがないこと(AC4 は grep で見る)
- [x] PR created
- PR(2026-10-23): ブランチ `docs/trial-docs-wrapup` の PR を作った。cross-review(454df239)は指摘 0、walkthrough は 4f484799。系列の最後の PR なので、記録に番号は書き足さない(D3)
