# org-recheck

- Status: Approved
- Approved: 2026-10-24 sha256:1e218f3d7907
- Owner: Claude Code
- Date: 2026-10-24
- Related request: release の前に、試行の所見の直し(#244〜#254)をすべて入れた main で、director から merge までを通しで確かめる(ユーザーの依頼「release の前に通しで確かめておきたい」、2026-10-24)
- Related issue: N/A
- Type: docs
- Branch: docs/org-recheck

## Objective

前回の試行(2026-10-20、f0668004、`docs/evidence/org-trial-2026-10-20.md`)は、所見の直しの前の binary で回した。直しは 1 つずつ分離した herdr で確かめたが、全部を入れた binary で director から merge までを通した実行はない。手順書 `docs/recipes/isolated-herdr-server.md` も、書いたあとに頭から通していない。

この計画では、main(ffb3e271)から build した binary で、分離した herdr server の上に機能の org を 2 つ同時に立てる。director はこのセッションを `/director` で昇格させたもの、人はユーザーとする。2 つの PR を `ralph org merge` で main に merge するまで回し、各所見の直しが効いているか、開いたまま残る行がどれだけ出るか、手順書を書いてあるとおりに通せるかを記録する。その結果で、release を切ってよいかをユーザーが判断できるようにする。

## Scope

- 環境(S1): 手順書のとおり `/tmp/ralph-iso` を作り、ssh で clone(`/tmp/ralph-iso/ralph`)、wrapper `/tmp/ralph-iso/run`、`claude-env`、osascript の shim、agmsg の私用の写し `/tmp/ralph-iso/agmsg`、ヘッドレスの herdr server、手順書の「Checks before the first start」をすべて打つ。binary は clone の HEAD から `go build -o /tmp/ralph-iso/bin/ralph ./cmd/ralph`
- 実行(S2・S3): 分割計画 1 つ、機能 2 つ(D4)、機能ごとの org 2 つ。承認はユーザー、merge は D7
- 確かめる項目(AC3): F3、F4、F5、F6・F13、F7、F11、F8・F9・F10・G1 の効き、開いたまま残る行の回数、gate の所要時間、前回との比較
- 記録(S4〜S6): evidence と台帳の写し、tech-debt、手順書(root と `templates/base` の 2 つの写し)、仕様の Status と根拠の表。implementer の座席が scratchpad のデータから書く
- 片付け(S7): 手順書の Clean up のとおり

## Non-goals

- 停止と再開(前回の D6)。前回は org の停止と再開も見張りの停止と再開も期待どおりだった。その後の直しで変わったのは spawn の失敗の座席の stop だけで、`docs/evidence/seat-spawn-close-2026-10-22.md` の実機で確かめた。介入と時間を減らすために外し、evidence の Not verified に書く。仕様の open question「停止と再開」は開いたままにする。実行の途中で org を止めることになったら、そのとき観察して記録する
- 開いたまま残る行(見張りから leader への ALERT の時間切れ、watcher の時間切れ、Codex の leader、FR-10、N4)の直し。回数と状況を記録するだけにする
- codex の座席、codex の director、headless の director(`ralph org director start`)
- 既定値(`max_orgs`、`max_total_seats`、`ack_minutes`、`resolve_minutes`)の見直し。前回の FR-12 の注で決めた。今回の数値が見直しの理由になりそうなら、所見として書くだけにする
- release(ユーザーが切る)
- 題材の 2 つの直しの中身。leader と座席が書き、gate と CI と reviewer の座席が確かめる。director は実装しない

## Assumptions

- 手元の版は herdr 0.7.5、agmsg 1.5.1、Claude Code 2.1.291、gh 2.102.0(yoshpy-dev が active)、git 2.49.0、go 1.26.0。前回の試行と直しの実機確認と同じ版
- main の ruleset(MAIN PROTECTION)は required check の `check` と `verify` を strict で持つ。`ralph org merge` は保護の条件を満たす(2026-10-24 に `gh api` で読んだ)
- 動いている herdr server はない。`/tmp/ralph-iso` はない。`ralph org` のプロセスも動いていない(2026-10-24 に確かめた)
- 環境変数 `RALPH_ORG_AGMSG_HOME` は `[org].agmsg_home` より先に読まれる(`ResolveAgmsgHome`、`internal/org/driver/agmsg.go:150`)
- ssh の push は、env -i の pane からも keychain の鍵で通る(前回の試行で通った)
- claude の bypass の初回承諾は、このマシンでは済んでいる(前回の試行の座席が同じ `HOME` で動いた)

## Affected areas

- 新規: `docs/evidence/org-recheck-2026-10-24.md`、`docs/evidence/org-recheck-2026-10-24/`(伏せ字にした台帳の写し)
- 更新: `docs/recipes/isolated-herdr-server.md` と `templates/base/docs/recipes/isolated-herdr-server.md`(同じバイト列)、`docs/tech-debt/README.md`(145 行と 126 行を RESOLVED、新しい所見の行、開いたまま残る行の数値)、`docs/specs/2026-10-07-org-multi-org-director.md`(Status)、`docs/evidence/org-spec-evidence-2026-10-21.md`(通しの確かめの結果)
- この計画の外で変わるもの: 機能の org が開く 2 本の PR(`internal/insights/backfill.go` と `backfill_test.go`、`internal/cli/doctor.go` と `doctor_hooks_test.go`)。それぞれの org の PR として merge する
- repo の外: `/tmp/ralph-iso`(S7 で消す)、`~/.claude.json` の trust の記録と `~/.claude/projects/` の session の記録(残る。手順書の Clean up のとおり)

## Visual review

None(リポジトリの変更は記録と手順書だけで、実行は今ある仕組みを動かすだけで何も足さない。前回の試行の計画と同じ扱い。分割計画の図解のページは S2 で director が作ってユーザーに見せる)

## Design decisions

Critical forks: None asked. ユーザーの 3 つの決定(org は 2 つ、`ralph org merge` で merge、承認はユーザー)が規模と外に出る操作を決めた。残りは下のとおり自分で選んだ。

- D1(場所と環境): 手順書のとおり `/tmp/ralph-iso`。clone は `git clone ssh://git@github.com/yoshpy-dev/ralph.git /tmp/ralph-iso/ralph`。wrapper の `env -i` の一覧は手順書の表のとおりで、`SHELL=/bin/bash`(利用者の zsh には claude にフラグを足す alias がある)、`PATH` は `/tmp/ralph-iso/bin`、claude・node・python の実体の置き場所、`~/.local/bin`、Homebrew、system の順。F1 の 2 行、`CLAUDE_ENV_FILE=/tmp/ralph-iso/claude-env`、`GOFLAGS=-timeout=30m`(前回と同じ。gate の `go test` に `-timeout` がない)を入れる
- D2(agmsg の写し): 手順書の任意の項のとおり写しを作る。ただし渡し方は、clone の `ralph.toml` の `[org].agmsg_home` ではなく、wrapper の一覧に `RALPH_ORG_AGMSG_HOME=/tmp/ralph-iso/agmsg` を足す。clone の追跡されない `ralph.toml` は `git worktree add` で機能の worktree に入らず、leader が機能の worktree で打つ spawn は cwd の `ralph.toml` を読むので、`ralph.toml` に書くと leader の座席だけが利用者の agmsg を使い、team が分かれる(consult の指摘。コードからの推論で、実機では確かめていない)。env なら server、pane、見張り、director のすべてに同じ値が届く。手順書との違いは deviation として記録し、手順書の穴を S5 で直す
- D3(通知): 手順書の shim `/tmp/ralph-iso/bin/osascript`。本物の osascript は打たない。人への経路で送られたものは台帳の `escalations.jsonl` と shim のログで数え、director がチャットで伝える
- D4(題材): 開いた tech-debt の行の直しを 2 つ。subagent の候補 6 件から選んだ。別の package で、触るファイルが重ならず、どちらも 50 行ほどで、テストは文字列を読む純関数のもの(git、chmod、プロセス、待ち時間に依存しない)。どちらも `docs/tech-debt/README.md` を書かない。行の RESOLVED はこの計画の PR が書く
  - `backfill-na`(Type `fix`、予約 `internal/insights/backfill.go`、`internal/insights/backfill_test.go`): tech-debt 145 行。`parseCrossReviewAllCycles` は triage の `Reviewer status:` を読まず、reviewer が終わらなかった report でも 0/0/0 なら `crossReviewVerdict` が `pass` を返す。`incomplete` で始まる status の cycle は `n/a` にし、テストで確かめる
  - `hooks-matcher`(Type `fix`、予約 `internal/cli/doctor.go`、`internal/cli/doctor_hooks_test.go`): tech-debt 126 行の (1) と (2)。`validateCodexHooksJSON` は matcher の型を見ない(文字列でなければ finding にする)。config.toml の併存の警告は `hooks` のキーがあるだけで出る(`[hooks]` の子が `state` だけなら出さない)。それぞれ負のテストを足す
  - slug は 11 字と 13 字。`<org_id>_implementer` は 25 字で、herdr の 32 字に収まる
- D5(models): leader は `--model opus`。分割計画の前文に「implementer は `--model opus`、reviewer は `--model sonnet`」と書く(`.claude/rules/ralph/model-routing.md` の既定)。前回は前文が leader に届かず(F5)、implementer が sonnet で動いた。opus と書けば、`spawn_started` の model で前文が届いたかを読める(consult の指摘)。座席はどれも既定の permission(`autonomous`)
- D6(フォルダ信頼): 人が答える(#254 で `/org` skill と `/director` skill に書いた決まり)。人が自分の端末で `cd /tmp/ralph-iso/ralph && claude` を打ち、「Yes, I trust this folder」を選んで `/exit` する(介入 I0)。手順書は「捨ての pane で自分で答える」と書く。trust は `~/.claude.json` にパスで記録されるので効き目は同じ、という推論で進め、deviation として記録する。機能の worktree(`.claude/worktrees/` の下)で画面が出ないことは、S3 で座席の pane を `ralph org read` で読んで確かめる
- D7(merge): 前回の D8 と同じ形で、regate の道を必ず通すための待ちを足す(Codex の計画助言 1)。1 本目の merge は、両方の機能の org が同じ base の SHA で `gate_passed` を記録し、両方とも push と disband を終え、`ralph org merge-order` が両方を `ready` と出すまで待つ。遅い leader が 1 本目の merge のあとで main を取り込んで gate を通すと、2 本目が直接 `ready` になり、regate を通らないため。そろったら、ユーザーの指示で一方を `ralph org merge` で merge する。もう 1 本は `merge-order` で `rebase` と出るはずなので、それを確かめてから、ユーザーの了解のあと clone の `ralph.toml`(clone の `.git/info/exclude` で除く)に `[org.director]` の `auto_merge = true` を書き、director が `/director` skill のとおりに regate して merge する。書く前に、`auto_merge = true` だけの scratch の `ralph.toml` が読めて `ralph org director status --config <file>` が `auto_merge: true` を出すことを確かめる。実行のあいだ、この 2 本のほかに main へ merge しない(この計画の PR は実行のあと)。`ralph org merge` か `ralph org regate` が断ったら、その理由を記録してユーザーに聞く
- D8(確かめの pane の probe): キーを送らないように、捨ての pane の中で `claude -p --model haiku --allowedTools Bash '<依頼>'` を打つ。依頼は `command -v git gh ralph claude osascript`、`git config user.name`、`gh auth status`、`git -C /tmp/ralph-iso/ralph push --dry-run origin HEAD:refs/heads/org-recheck-probe`(push の認証の確かめ。dry-run は ref を作らない)。`-p` のセッションが `CLAUDE_ENV_FILE` を読むかは確かめていないので、shim が出なければ、人に pane で対話の claude を立てて同じ依頼を打ってもらう(介入として数える)。手順書との違いは deviation として記録する
- D9(数え方): 前回の D5 と同じ。介入(聞いた時刻と答えが届いた時刻。AskUserQuestion の呼び出しと結果の時刻を transcript から読む)、機能の計画ごとの承認待ち(受信箱の件の `created` から `plan_approved`)、受信箱の件ごとの処理(`created` から `acked`、`resolved`、`notified` とその理由)、費用(座席の session の記録の token を model ごとに足し、その日の定価で見積もる。director 自身の費用は分けられないので外す)。データは scratchpad の `recheck-notes.md` と `interventions.log` に置き、`/tmp/ralph-iso` には置かない
- D10(所見の確かめ方): AC3 の各項目を、台帳(`manifest.jsonl`、`inbox.jsonl`、`escalations.jsonl`、`watch.log`)、`git log`、`herdr agent list`、座席の pane(`ralph org read`)、機能の計画のファイルのどれで読むかを、evidence の各行に書く。台帳から読める数は、前回と同じく evidence の `jq` のコマンドで写しから出し直せるようにする
- D11(記録を書く人): evidence と記録の slice は implementer の座席が、director が scratchpad に残したデータから書く。director は実行のあいだ repo のファイルを書かない
- D12(台帳の写し): 写しは 2 回取る。1 回目は disband のあと、detach の前(片付けの途中でやめても記録が残るように)。最後の写しは、director の detach、見張りの停止、分離した server の停止のあと、`/tmp/ralph-iso` を消す前に取る。detach の `detached` の行、見張りの停止の印、止まるまでに見張りが送った期限の通知も写しに入れるため(Codex の計画助言 3)。写すのは台帳の記録(`manifest.jsonl`、`inbox.jsonl`、`director.jsonl`、`escalations.jsonl`、`watch.log`、`watch-alive.json`、`watch-status-<org>.json`、`director-wait.json`、`model-receipts.jsonl`、`merge-guarantee.json`、分割計画、機能の計画、gate の記録)と shim のログで、置き場所は scratchpad。最後の写しで、受信箱の件ごとの通知と shim のログの行数、`ralph status` の停止の行を照らし合わせ、合ってから消す。S4 は伏せ字にした最後の写しを `docs/evidence/org-recheck-2026-10-24/` に置く(pane と workspace の id、session の id、ランダムな hex は `pane-1` のような安定したラベルに、`$HOME` は `~` に)

## Acceptance criteria

- [x] AC1(環境): evidence に、binary を build したコミット、道具の版、wrapper の変数の名前(値は token を含まないものだけ)、shim、agmsg の写し、models を書く。手順書の各節を書いてあるとおりに打てたか、打てなかった所(deviation)とその理由を節ごとに書く。`$HOME` のパス、token、pane id、session id、ランダムな hex を含まない
- [x] AC2(2 つの org): manifest に、承認された 1 つの分割計画から立った 2 つの機能の org があり、動いていた時間が重なる。それぞれが PR を 1 本開き、evidence が 2 本の PR を名指す
- [x] AC3(所見の直しの効き): evidence の表に、次の各項目について、期待、見たこと、読んだ場所(ファイルとコマンド)、判定(効いた、効かなかった、起きなかったので確かめられない)を 1 行ずつ書く
  - F3: `agent_not_found` で失敗した send がない。実行の終わりの前に `herdr agent list` が出す座席の agent の名前が `<org>_<seat>` のまま
  - F4: spawn の失敗があれば、その pane が閉じている(`spawn_failed` か `spawn_close_failed`)。なければ「起きなかった」
  - F5: 両方の機能の計画に `## 全機能の座席に適用する決まり` があり、分割計画の前文が写っている。`spawn_started` の model が、implementer は opus、reviewer は sonnet
  - F6・F13: 見張りの escalation と leader の escalate の件を 1 件ずつ、受信箱(`inbox.jsonl`)、`escalations.jsonl`、shim のログで照らし合わせる(Codex の計画助言 2)。見張りの escalation は受信箱の TYPE ALERT の件になっている。director が ack の期限のうちに処理した件には、人への通知がなく、`escalations.jsonl` にも shim のログにも行がない(これが正しい)。通知の条件を満たした件(ack の期限、resolve の期限、見張りがない、director の `inbox notify`)には、`notified` の理由と通知の結果が記録され、shim のログにその件の行がある。escalation が 1 件もなければ「起きなかった」
  - F7: director の leader への send が時間切れで失敗しない(manifest の `send_failed` と director の notes)。leader が Monitor で待つ(pane の読み)
  - F11: 両方の機能のブランチの `git log origin/main..<branch>` に `wip:` のコミットがない
  - F8・F9・F10・G1: gate と regate の失敗のうち、機能の変更と関係のない repo の都合によるものがない(gate の記録とログの抜粋)
  - 開いたまま残る行: `watch.log` の、見張りから leader への ALERT の時間切れの回数と、watcher の時間切れの回数。その前後の load average(director が 30 分ごとに `uptime` を notes に書く)
  - gate と regate の所要時間(gate の記録の開始と終わり)
  - フォルダ信頼: 機能の worktree の座席で trust の画面が出なかった(座席の pane の読み)
- [x] AC4(merge): 1 本目の merge の前に、両方の org の `gate_passed` が同じ base の SHA を持ち、両方が disband していた。2 本の PR が `ralph org merge` で merge された(1 本目はユーザーの指示、2 本目は `auto_merge` で)。2 本目は 1 本目の merge のあと `ralph org merge-order` で `rebase` と出て、`ralph org regate` が `via=regate` の `gate_passed` を記録してから merge された。または、どこかで止まったなら、その断りか失敗を動詞自身の理由とともに記録した
- [x] AC5(数値と前回との比較): evidence に、介入、承認待ち、受信箱の処理、費用の表を前回と同じ形で書き、前回の値と並べる。どの数もファイルかコマンドを名指し、台帳の数は写しから evidence の `jq` で出し直せる
- [x] AC6(記録): tech-debt の 145 行と 126 行が PR の番号付きで RESOLVED。今回の所見のうち直さないものには tech-debt の行がある。開いたまま残る 2 行(ALERT の時間切れ、watcher の時間切れ)に今回の回数を書き足す。仕様の Status と根拠の表に、通しの確かめの結果と evidence への参照がある
- [x] AC7(手順書): 手順書の「The steps were not run end to end again」と「An end-to-end run of these steps as written」の文を、今回の実行に合わせて直す。agmsg の写しを機能の org で使うときの渡し方(D2)を書く。今回の deviation のうち手順書の穴にあたるものを直す。root と `templates/base` の写しが同じバイト列(`scripts/check-sync.sh`)
- [x] AC8(片付け): 途中でやめたときも、次の順で済ませる。機能の org を disband する、D12 の 1 回目の写しを取る、director の登録を detach する、見張りを `ralph org watch --all --stop` で止めて `ralph status` が `org watch: not running` を出し `pgrep -fl /tmp/ralph-iso` が何も出さない、clone の機能の worktree を片付ける、分離した herdr server を止める、D12 の最後の写しを取って通知の数と停止の行を照らし合わせる、そのあとで `/tmp/ralph-iso` を消す。利用者の herdr server には触れていない。shim のログに、実行で打たれた osascript がすべてある
- [x] AC9(写し): `docs/evidence/org-recheck-2026-10-24/` に D12 の伏せ字の写しがある。evidence の `jq` を写しに打つと evidence の表になる。grep で `/Users/` のパス、pane id、session id、token が見つからない

## Implementation outline

1. S1 環境(director になる前のこのセッション、inline。repo には書かない): 手順書の頭から順に、`/tmp/ralph-iso`、clone、build、wrapper、claude-env、shim、agmsg の写し、`herdr --help` の確かめ、server の起動、`lsof` と `ps eww` の確かめ。ユーザーに I0(フォルダ信頼)を頼む。D8 の probe。clone の main のチェックアウトと ralph.toml のないディレクトリで `/tmp/ralph-iso/run ralph doctor` と `ralph status`(agmsg の行が写しを指す、`org watch: not running`、Merge guarantee が `ruleset` の保証を出す)。D7 の `auto_merge` の scratch の確かめ。手順書の各節で、書いてあるとおりに打てたかを notes に書く
2. S2 director と分割計画(director、inline): このセッションを `/director` で昇格させ、wrapper を通して attach する。分割計画 `/tmp/ralph-iso/ralph/.harness/state/org/splits/recheck.md` に、前文(D5 の models、予約の外に書かない、`docs/tech-debt/README.md` を書かない)と 2 つの機能(D4)を書く。図解のページを見せてユーザーに承認を聞き(介入)、承認の 2 行を書く
3. S3 実行(director): 2 つの org を `ralph org start --plan` で立てる。受信箱を wait で待ち、機能の計画をユーザーに見せて承認を聞き、`ralph org approve` を打つ。30 分ごとに `uptime` と token の合計を notes に書く。PR を報告し、両方の gate と disband がそろうのを待ってから(D7)merge する。止まった org はユーザーに伝え、director は直さない
4. S4 写しと evidence(S7 の最後の写しのあと。implementer、director の scratchpad のデータから): 伏せ字の写しと `docs/evidence/org-recheck-2026-10-24.md`
5. S5 手順書(implementer): AC7
6. S6 記録(implementer、sonnet): tech-debt、仕様の Status、根拠の表(AC6)
7. S7 片付け(director、inline、AC8 の順): 途中でやめたときも同じ順。D12 の写しは disband のあとと server の停止のあとの 2 回取り、最後の写しで照らし合わせてから `/tmp/ralph-iso` を消す。S4 は最後の写しのあとに走らせる
8. そのあと標準のパイプライン(self-review → verify → test → sync-docs → cross-review → pr)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`(手順書の 2 つの写し)、`./scripts/check-skill-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC9 を evidence、写し、手順書、tech-debt、仕様の Status に照らす
- Documentation drift to check: 手順書の 2 つの写しが同じバイト列。手順書と `/org` skill の前提の参照(`isolated-herdr-server.md`)がまだ合うこと。tech-debt の RESOLVED の行が PR の番号を名指すこと
- Evidence to capture: evidence のファイル、写しの grep(`/Users/`、pane id の形、token)の結果

## Test plan

- Unit tests: この計画の PR は記録と手順書だけなので、新しいテストはない。`./scripts/run-test.sh`(Go のテストは `-timeout 30m`、`GOTOOLCHAIN=go1.25.8` でも)
- Integration tests: 通しの実行そのもの。結果はテストのファイルではなく evidence
- Regression tests: `./scripts/run-test.sh` を最後のブランチで回す。2 つの機能の PR は、それぞれの org の gate と CI が確かめる
- Edge cases: spawn の失敗(F4)、見張りの escalation(F6・F13)、`ralph org merge` の断り(D7)は起きたときだけ確かめられる。起きなければ evidence に「起きなかった」と書く
- Evidence to capture: test の report の run-test の結果

## Risks and mitigations

- R1 画面で座席が止まる(フォルダ信頼、bypass の承諾、ほかのダイアログ): director は `ralph org read` で pane を読み、ユーザーに伝える。キーは送らない(`/director` skill の「してはいけないこと」)。ユーザーが分離した server の pane に入るには `/tmp/ralph-iso/run herdr` を自分の端末で打つ必要があるので、その打ち方を伝える
- R2 agmsg の team が分かれる(D2 の推論が外れる): S1 で `ralph doctor` の agmsg の行を ralph.toml のないディレクトリから確かめる。S3 で最初の座席の HELLO が leader に届くか(manifest と `ralph org read`)を見て、届かなければ org を止めてユーザーに伝える
- R3 費用と時間がかさむ(座席が回り続ける、gate が繰り返す): director は 30 分ごとに token の合計を見る。最初の start から 5 時間を過ぎたとき、または座席が同じ失敗を 3 回繰り返したときは、ユーザーに伝え、ユーザーの言葉で org を止める
- R4 機能の org が失敗するか止まる(BLOCKED、leader が終わる): director は伝えて所見として記録する。直しは実装しない
- R5 本物の変更が main に入る: 2 つの直しは小さく、gate と CI と reviewer の座席を通る。1 本目の merge と `auto_merge` の切り替えはユーザーが決める
- R6 本物の osascript が動く: どの start の前にも、probe で `command -v osascript` が shim を指すこと。見張りは wrapper から起動する
- R7 2 つの gate と全テストが同時に走ってマシンが重い: leader は gate を background で回す(雛形)。director は待つ。load average を notes に書く。時間依存のテストが落ちたら、その名前と load average を所見に書く
- R8 実行のあいだにこのセッションの会話が要約される: director は登録を読み直し(`/director` の昇格 5)、scratchpad の `interventions.log` と `recheck-notes.md` を読み直す。notes は段ごとに書き足しておく
- R9 利用者の herdr server や agmsg に触れる: すべての `ralph` と `herdr` を wrapper から打つ。素の `ralph org` を打たない。S1 で、利用者の agmsg の `db` の更新時刻を記録し、S7 で変わっていないことを確かめる

## Rollout or rollback notes

repo の変更は記録と手順書だけ。戻すときはこの計画の PR を revert する。2 本の機能の PR は別の PR なので、それぞれで revert できる。clone、server、台帳は `/tmp/ralph-iso` の下にあり、S7 で消す。

## Open questions

- `claude -p` のセッションが `CLAUDE_ENV_FILE` を読むか(D8)。S1 の probe で分かる。shim が出なければ対話の claude で確かめ直す
- D2 の推論(`ralph.toml` に書くと leader の座席だけが利用者の agmsg を使う)は、今回は env で避けるので確かめない。手順書に書くときは推論として書く

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(2026-10-24。ユーザーの決まり「作成したプランは自動承認します」による自動承認。Codex の計画助言 3 件(regate の待ち、通知の照らし合わせ、最後の写し)は承認の前に直した)
- [x] Branch created
- [x] Implementation started
- [x] S1〜S3、S7(director、inline): 2026-10-24 09:40Z〜2026-10-25 14:24Z。機能の PR #255 と #256 を `ralph org merge` で main に merge(536271a2、98d38391)。`/tmp/ralph-iso` は消した。記録は scratchpad の notes から S4 が evidence に写した
- [x] S4(implementer、opus): bcb9db16(evidence と伏せ字の写し)、77badb04(agmsg の未読の列。director が consult の指摘で probe の文を弱めた分も同じコミット)。f7ad239d(director、inline): 利用者の global gitignore の `*.log` で漏れた写しの 3 本を `git add -f` で足した(前回の試行の写しと同じ扱い)
- [x] S5(implementer、opus): 29b54de0(手順書の 8 か所)。59779049(director、inline): 専用の clone の機能の worktree の片付けを Clean up に足した(S5 の報告の範囲外の指摘)
- [x] S6(implementer): 59964c5c。計画は sonnet としたが、仕様の印の判断が入るので opus にした。Affected areas にない `docs/evidence/org-trial-2026-10-20.md` に Correction の段落を足した(consult の指摘。RC11 が前回の通知の出どころの記述を覆すため)。印の規則に照らし、RC2 で FR-4 と受け入れ条件 5 を `[ ]` に戻した(consult も同じ判定)
- 所見の番号は RC1〜RC12(既存の F・G・N などと重ならないもの)。tech-debt の行は RC1〜RC6、RC9〜RC11。RC7 は手順書で直し、RC8 と RC12 は repo の欠陥ではないので行を作らない(evidence の Findings に理由)
- [x] Review artifact created(`docs/reports/self-review-2026-10-26-org-recheck.md`、2b399912。Merge yes、LOW 6・INFO 4。implementer が a688fcf0 で 10 件とも直し、7f004c11 で報告に追記)
- [x] Verification artifact created(`docs/reports/verify-2026-10-26-org-recheck.md`、0c0a6a6e。pass、LOW 1・INFO 4。LOW-1(根拠の表の RC11 の行の「1 回ごとに通知」)は director が inline で直した。INFO-1・INFO-2・INFO-4 は直さない。INFO-2(/org skill のフォルダ信頼の項)は skill の直しで、この記録のブランチの範囲外)
- AC の印(verify のあと): AC1、AC2、AC4〜AC7、AC9 は verify が repo で確かめた。AC3 は表と写しから出る数まで確かめ、load average、終了コード、pane の読みは director の notes による。AC8 は順の記録と写しの痕(`detached`、`stopped`、`end=watch`、`ralph status` の停止の行)が合う。見張りを止めた直後の確かめは、notes に「~ から pgrep で ralph の見張りがない」とだけあって pattern を書いておらず、`pgrep -fl /tmp/ralph-iso` そのものは `rm -rf` のあと(evidence の Cleanup の 8 段目、終了コード 1)にしか記録がない
- [x] Test artifact created(`docs/reports/test-2026-10-26-org-recheck.md`、124ac68e。pass。HEAD f5bc474b で run-test.sh rc 0(shell 44 本、scope docs_only)、go test が go1.26.0 と go1.25.8 でどちらも rc 0。変わった文書を読む Go のテストはない)
- sync-docs(`docs/reports/sync-docs-2026-10-26-org-recheck.md`、0341b929): ブランチの外の文書のずれなし。verify の INFO-4(self-review の報告の event のファイル名)はここで直した。cross-review(`docs/reports/cross-review-triage-org-recheck.md`、109fb3f9): codex、cycle 1/2、指摘 0。walkthrough は 73f0eaf2
- [x] PR created(#257)
