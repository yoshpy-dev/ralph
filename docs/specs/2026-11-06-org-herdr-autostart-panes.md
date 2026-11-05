# org の herdr: サーバーの自動起動と 1 タブのペイン配置

## Status

2026-11-06 に人が承認した。実装は 2 本の PR に分ける。A(herdr の自動起動)を先に出し、B(1 タブのペイン配置)をそのあとに出す。

## Summary

herdr のサーバーが止まっていても、ralph が座席やワークスペースを作る前に `herdr server` を起動する。org のワークスペースはタブを 1 つだけ持ち、座席はそのタブの中のペインになる。各ペインには座席 id のラベルを付け、leader を左半分、2 席目を右上、3 席目を右下に置く。

## Background and problem

2026-11-05 に org runtime を使っていて、次の 2 件が起きた。

1. director が `ralph org start --plan` で機能 infra-deploy-base の org を立てようとし、leader の座席を作る段で herdr が `Connection refused` を返して失敗した。worktree `.claude/worktrees/org-infra-deploy-base` とブランチ `ci/infra-deploy-base` は作られたまま残った。director は人に「別の端末で herdr を起動してほしい」と頼み、人が別のタブで `herdr` を打ってから、director が同じ start を打ち直した。
2. herdr の画面で見ると、どの org のワークスペースにも「1」「leader」「implementer」「reviewer」の 4 つのタブがあった。タブ「1」では何も動いていない。

人は 1 について、同じことが起きたら herdr を自動で起動してほしいと求めた。2 については、ワークスペースごとにタブを 1 つだけにし、その中に leader・implementer・reviewer のペインを置くこと、何も動いていないタブをなくすこと、各ペインに座席の名前を付けることを求めた。配置は leader が左半分、implementer が右上 1/4、reviewer が右下 1/4 を望むが、座席がそろわない org もあるので厳密には問わない。

### Current state

- ralph にも director の手順書(`.claude/skills/director/SKILL.md`、`internal/org/prompts/director.md`)にも、herdr のサーバーを起動する処理や手順はない。herdr が止まっていると、`ralph org start` は Spawn の `workspace_create` の段で herdr のエラーをそのまま返す。`--plan` の形では、この段に来る前に worktree とブランチを作っている(`internal/org/feature.go` の `StartFeature`)。
- org のワークスペースは `resolveWorkspace`(`internal/org/spawn.go`)が `herdr workspace create --cwd <cwd> --label <org_id>` で作る。座席は Spawn が `herdr tab create --workspace <id> --cwd <cwd> --label <seat_id> --env ...` でタブごとに作る(`spawn.go` の `o.Herdr.TabCreate` の呼び出し)。`workspace create` が必ず作るタブ「1」とそのシェルのペインは、どこからも使われない。
- `ralph org director start` の herdr の形も同じ作り方で、`WorkspaceCreate` のあとに `TabCreate(DirectorTabLabel(token))` を呼ぶ(`internal/org/director_start.go`)。director のワークスペースにも空のタブ「1」が残る。
- 座席を止める・解散する・補償で閉じる前の確認 `confirmSeatPane`(`internal/org/verbs.go`)は、herdr にペインのタブとワークスペースを聞き、タブのラベルが座席 id で、ワークスペースのラベルが org_id であることを確かめる。herdr がセッションの状態を失うと id を w1 から振り直すので、記録した id が別のペインを指すことがある。この確認はそのときに別のペインを閉じないためにある。director の停止(`confirmDirectorResources`、`internal/org/director_stop.go`)も同じ関数を `DirectorTabLabel` で使う。
- 空のタブ「1」は、座席のタブがすべて閉じたあとも org のワークスペースを残している。解散(`closeOrgWorkspace`、`verbs.go`)は herdr に無いワークスペースを「すでに閉じた」として扱う。

### Desired state

- herdr が止まっていると、`ralph org start`・`ralph org spawn`・`ralph org director start` が自分で `herdr server` を起動し、そのことを stderr の 1 行で知らせる。director は人に herdr の起動を頼まない。人に頼むのは、最初の座席の承諾画面に答えるために `herdr` で接続することだけになる。
- org のワークスペースにはタブが 1 つだけあり、そのラベルは org_id である。leader → implementer → reviewer の順に座席を作ると、leader が左半分、implementer が右上、reviewer が右下のペインになり、それぞれに座席 id のラベルが付く。director の headless のワークスペースも、タブ 1 つ・ペイン 1 つになる。

## Requirements

### Functional requirements

#### A. herdr のサーバーの自動起動(1 本目の PR)

- [ ] FR-A1: `ralph org start`(`--plan` の形とそうでない形)、`ralph org spawn`、`ralph org director start`(herdr で立てる形)は、herdr にワークスペースやペインを作らせる前に `herdr status server --json` を読む。`running` が false なら `herdr server` を新しいセッション(setsid)で切り離して起動し、`running` が true になるまで最大 10 秒待つ。`start --plan` は、worktree とブランチを作る前にこれを済ませる。`--dry-run` では起動しない。`director start` の `[org.director].launch_cmd` の形は herdr を使わないので対象にしない。
- [ ] FR-A2: `herdr server` の起動が失敗しても、もう一度 `herdr status server --json` を読み、`running` が true なら先に進む。別の ralph が先に起動した場合で、herdr は 2 つ目のサーバーを `error: herdr server is already running` と終了コード 1 で止める。
- [ ] FR-A3: 起動するサーバーの環境は、ralph の環境から次を除いたものにする。`HERDR_PANE_ID`、`HERDR_TAB_ID`、`HERDR_WORKSPACE_ID`、`HERDR_ENV`、`CLAUDECODE`、`CLAUDE_CODE_` で始まる変数。サーバーの環境はすべてのペインに入るので、director の Claude Code のセッションの変数(`CLAUDE_CODE_MESSAGING_TOKEN` など)を座席に渡さないためである。`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` は残し、分離した herdr の手順(`docs/recipes/isolated-herdr-server.md`)と両立させる。`CODEX_` で始まる変数は残す(codex の座席の観測が `CODEX_HOME` を読む)。作業ディレクトリは `HOME`、標準入力は `/dev/null`、標準出力と標準エラーは台帳の state dir の `herdr-launch.log` に追記する。
- [ ] FR-A4: サーバーを起動したときは、stderr に 1 行を書く。中身は、起動したこと、pid、ソケットのパス、`herdr` で接続すると座席のペインと最初の座席の承諾画面が見えることである。
- [ ] FR-A5: 起動できないとき(`herdr` が PATH に無い、10 秒たっても `running` にならない)は、worktree・ブランチ・ワークスペースのどれも作らずに失敗する。エラーには `herdr-launch.log` のパスと、手で起動する方法(別の端末で `herdr` を打つ)を書く。`compatible` が false のサーバー(クライアントと版が違う)は止めも置き換えもせず、今までどおり herdr の失敗として返す。
- [ ] FR-A6: director の手順書(`.claude/skills/director/SKILL.md` と `.agents/skills/director/SKILL.md`、`internal/org/prompts/director.md`)と `/org` の skill に、herdr が止まっていても ralph が起動すること、人には起動を頼まず、承諾画面のために `herdr` で接続することだけを頼むことを書く。

#### B. 1 タブのペイン配置(2 本目の PR)

- [ ] FR-B1: org のワークスペースを作る座席は、`herdr workspace create --cwd <cwd> --label <org_id> --env <PaneEnv の各項> --env RALPH_PRIMARY_CLI=<driver>` が作るルートペインに入る。続けてタブのラベルを org_id にする(`herdr tab rename`)。org のワークスペースに空のタブは残らない。
- [ ] FR-B2: 開いたワークスペースがある org の座席は、`herdr pane split <起点> --direction <向き> --ratio 0.5 --cwd <cwd> --env ...` で作ったペインに入る。台帳で生きている座席のペインが 1 つだけなら、そのペインを右に分ける。2 つ以上なら、leader 以外で最後に spawn した座席のペインを下に分ける。leader → implementer → reviewer の順なら、左半分・右上・右下になる。reviewer を先に作った org では reviewer が右上になり、それでよい。
- [ ] FR-B3: 座席のペインには `herdr pane rename <pane> <seat_id>` で座席 id のラベルを付ける。
- [ ] FR-B4: 台帳が開いていると記録したワークスペースを使う前に、herdr がそれを持ち、ラベルが org_id であることを確かめる。herdr に無いか、ラベルが違えば、閉じたと記録してから新しいワークスペースを作る。1 つのタブのペインをすべて閉じるとワークスペースごと消えるので、座席をすべて止めたあとの spawn はこの道を通る。
- [ ] FR-B5: 同じ org の spawn が同時に走ったときに同じペインを 2 席が分けないよう、分割の起点を決めてからラベルを付け終えるまで、org 単位のロックをとる。
- [ ] FR-B6: 止める・解散する・補償で閉じる前の確認は、ペインのラベルが座席 id で、ワークスペースのラベルが org_id であることを見る。この版より前に作った座席(タブのラベルが座席 id で、ペインのラベルが空)も、今までどおり受け入れる。分けたあとラベルを付ける前に止まった spawn のペイン(その spawn が記録したペイン id で、ラベルが空、ワークスペースのラベルが org_id)は、補償で閉じてよい。
- [ ] FR-B7: `ralph org director start` の herdr の形も同じ作り方にする。director はルートペインに入り、ペインのラベルは `DirectorTabLabel(token)`、タブのラベルは `DirectorWorkspaceLabel(token)` にする。`director stop` の前の確認もペインのラベルを見て、この版より前に立てた director(タブのラベル)も受け入れる。
- [ ] FR-B8: herdr に `--env` があるかの確認(今は `herdr tab create --help` だけを見る)を、使うサブコマンド `workspace create` と `pane split` に広げる。`--env` が無い herdr では、今と同じ警告を出して `--env` なしで作る。

### Non-functional requirements

- [ ] herdr 0.7.5 で動くことを確かめた操作だけを使う。`status server --json`、`server`、`workspace create --env`、`pane split --direction right|down --ratio --cwd --env`、`pane rename`、`tab rename` である(Research findings)。
- [ ] 1 回の spawn で増える herdr の呼び出しは、状態の確認 1 回とラベル付け 1〜2 回に収める。herdr が動いているときは `herdr server` を呼ばない。
- [ ] herdr を呼ぶのは今までどおり `internal/org/driver/herdr.go` の `Herdr` だけにし、`internal/org` は `HerdrClient` を通して呼ぶ。サーバーの切り離し起動は、見張りの起動(`internal/cli/org_watch.go` の `startDetachedProcess`)と同じやり方にする。
- [ ] 振る舞いが変わる文書(org runtime の仕様、分離した herdr の手順、`/org` と `/director` の skill)を、それぞれの PR で直す。

## Acceptance criteria

### A(1 本目の PR)

- [ ] AC-A1: Given herdr のサーバーが止まっている(ソケットのファイルが残っている場合と無い場合の両方)、when `ralph org start --plan` を打つ、then サーバーが起動して leader の座席ができ、stderr に起動の 1 行(pid とソケットのパスを含む)が出る。
- [ ] AC-A2: Given `CLAUDECODE` と `CLAUDE_CODE_*` と `HERDR_PANE_ID` がある環境、when ralph がサーバーを起動する、then 起動した `herdr server` の環境にそれらの変数が無く、`XDG_*` と `HERDR_SOCKET_PATH` は残っている。
- [ ] AC-A3: Given herdr のサーバーが動いている、when `ralph org start`・`spawn`・`director start` を打つ、then `herdr server` は呼ばれない。
- [ ] AC-A4: Given `herdr server` を起動しても 10 秒以内に `running` にならない、when `ralph org start --plan` を打つ、then worktree もブランチも作らずに失敗し、エラーにログのパスと手で起動する方法がある。
- [ ] AC-A5: Given 2 つの `ralph org start` がほぼ同時にサーバーの起動を試みる、when 片方の `herdr server` が `already running` で終わる、then どちらの start も先に進み、動いているサーバーは 1 つである。
- [ ] AC-A6: director と `/org` の手順書に、herdr の起動を人に頼まないことと、承諾画面のために `herdr` で接続するよう頼むことが書いてある。

### B(2 本目の PR)

- [ ] AC-B1: Given 新しい機能の org、when leader → implementer → reviewer の順に spawn する、then そのワークスペースのタブは 1 つでラベルが org_id であり、leader のペインが左半分、implementer が右上、reviewer が右下にあり、各ペインのラベルが座席 id である。
- [ ] AC-B2: Given reviewer を implementer より先に spawn した org、when 3 席がそろう、then 3 つのペインが 1 つのタブに収まり、空のタブは無い。
- [ ] AC-B3: Given 3 席がある org、when implementer を `ralph org stop` で止める、then implementer のペインだけが閉じ、ほかのペインはそのまま動いている。
- [ ] AC-B4: Given 座席をすべて止めてワークスペースが消えた org、when 次の座席を spawn する、then 台帳に古いワークスペースを閉じたと記録し、新しいワークスペースのルートペインに座席が入る。
- [ ] AC-B5: Given herdr が id を振り直し、記録したペイン id が別の org のペインを指している、when その座席を止める、then ralph はそのペインに C-c も close も送らない。
- [ ] AC-B6: Given この版より前に作った org(座席ごとのタブ)、when その座席を止め、org を解散する、then 今までどおり閉じられる。
- [ ] AC-B7: Given split のあと rename の前に止まった spawn、when その座席の補償が走る、then ラベルの無いそのペインが閉じられる。
- [ ] AC-B8: Given `ralph org director start`(herdr の形)、when director が立つ、then そのワークスペースはタブ 1 つ・ペイン 1 つで、ペインにラベルが付き、`director stop` がそれを閉じる。
- [ ] AC-B9: 分離した herdr(`docs/recipes/isolated-herdr-server.md`)の上で、機能の org を 1 つ立てて 3 席をそろえ、AC-B1 と AC-B3 を実機で確かめた記録を `docs/evidence/` に残す。

## User stories

1. As director を使う人, I want herdr が止まっていても org が立つこと, so that 別の端末で herdr を起動してから start を打ち直す手間がなくなる.
2. As director を使う人, I want 1 つの org の座席を 1 つの画面で並べて見ること, so that タブを行き来せずに leader と座席の様子を同時に追える.
3. As 座席を止める ralph, I want 記録した id が別のペインを指していたら閉じないこと, so that herdr が id を振り直しても、ほかの org の作業を壊さない.

## Constraints

### In scope

- `ralph org start`・`spawn`・`director start`(herdr の形)での herdr のサーバーの自動起動と、その環境の整理
- org のワークスペースと director の headless のワークスペースを、タブ 1 つの中のペインで作ること
- ペインのラベルによる停止前の確認と、この版より前に作った座席の受け入れ
- 関係する文書と skill の更新

### Out of scope

- サーバーの再起動で戻った古い org のワークスペースの検出と片付け。herdr は再起動のとき session.json からワークスペース・タブ・ペインとそのラベルを戻すが、中のエージェントは戻らない。台帳はその座席を動いていると記録したままになる。
- 最初の座席の承諾画面への自動の応答。人が `herdr` で接続して答える。
- 端末のウィンドウを開いて herdr の画面を出すこと
- 役割でペインの位置を固定すること(`herdr pane swap` で implementer を常に右上にするなど)
- 既存のワークスペースを新しい形に移すこと。この版より前に作ったワークスペースにあとから spawn した座席は、生きている座席のペインを分けて入る(leader のタブの中に入る)。
- 版の違うサーバー(`compatible` が false)への対処
- 見張り・`send`・`read`・`wait` など、何も作らない動詞での自動起動

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/org/driver/herdr.go` | サーバーの状態の読み取り、`workspace create --env` とルートペインの id、`pane split`、`pane rename`、`tab rename`、`pane get` のラベル、`--env` の確認の範囲 | High |
| `internal/org/spawn.go` | `HerdrClient`、`resolveWorkspace` の作り直しの規則、座席のペインの作り方、補償の規則、org 単位の分割ロック | High |
| `internal/org/verbs.go` | `confirmSeatPane` をペインのラベルの確認に移し、前の版の座席も受け入れる | High |
| `internal/org/director_start.go`、`director_stop.go` | director のワークスペースをルートペインで作り、停止前の確認を移す | Medium |
| `internal/org/feature.go`、`internal/cli/org.go`、`internal/cli/org_director_headless.go` | start・spawn・director start の前の自動起動と stderr の 1 行 | Medium |
| `internal/cli/org_watch.go` | 切り離し起動の関数を herdr のサーバーにも使えるようにする | Low |
| `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md`、`.agents/` の写し、`internal/org/prompts/director.md` | 人に herdr の起動を頼まない。座席がタブでなくペインであること | Medium |
| `docs/specs/2026-08-01-org-runtime.md`、`docs/recipes/isolated-herdr-server.md` | ワークスペースとペインの形、自動起動の記述 | Low |
| `internal/org` と `internal/org/driver` のテスト | herdr の偽物(`fakeHerdr`)に新しい呼び出しを足し、argv と配置と確認の規則をテストする | Medium |

## Dependencies

- herdr 0.7.5 以上。この仕様の操作はすべて 0.7.5 で確かめた。
- 見張りの切り離し起動(`startDetachedProcess`)と、start の前に見張りを確保する仕組み(`wireOrgStartWatchGate`)。A はこれと同じ位置に herdr の確認を置く。

## Research findings

### Codebase analysis

- 座席のタブは `spawn.go` の `o.Herdr.TabCreate(ctx, workspaceID, p.Cwd, p.SeatID, driver.PrimaryCLIEntry(p.Driver))` で作られ、ワークスペースは `resolveWorkspace` が台帳の `org_workspace_created` を見て使い回すか作る。`openOrgWorkspaces` は `org_workspace_closed` の無い記録を開いているとみなし、herdr には聞かない。
- `confirmSeatPane` は `PaneGet` → `TabGet` → `WorkspaceGet` の順に herdr に聞く。`compensateStale` はラベルの食い違い(`errLabelMismatch`)ならペインを残して進む。
- `closeOrgWorkspace` は、herdr に無いワークスペースを「すでに閉じた」として扱い、`org_workspace_closed` を記録する。
- `driver.Herdr.paneEnvSupported` は `herdr tab create --help` に `--env` があるかだけを見て、結果をキャッシュする。
- 見張りの切り離し起動(`orgWatchChild`、`startDetachedProcess`)は setsid、標準入力 `/dev/null`、ログへの追記で動き、`HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV` を落とす。
- leader の雛形(`internal/org/prompts/leader.md` の「機能ごとの org」の手順 3)は、計画の承認の前に reviewer を spawn してよいとしている。spawn の順番は org ごとに違いうる。

### Best practices

2026-11-06 に、分離した herdr 0.7.5 のサーバー(`XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` を専用の場所に向けたもの)で次を確かめた。ユーザーの herdr のサーバーには触れていない。

- `herdr workspace create --cwd X --label org-x` の応答は、ラベル `"1"` のタブ(`tab_id` w1:t1)と、そのルートペイン(`pane_id` w1:p1)を含む。
- `herdr pane rename w1:p1 leader` のあと、`herdr pane get w1:p1` は `"label":"leader"` を返す。`herdr agent start` で claude を起動したあともラベルは変わらない。
- `herdr pane split w1:p1 --direction right --ratio 0.5` と、その右のペインの `--direction down --ratio 0.5` で、`pane layout` が左半分(27x23)・右上(27x12)・右下(27x11)を返した。タブは 1 つのままだった。claude が動いているペインを分けても、エージェントは `idle` のままだった。
- `workspace create --env RALPH_T1=root` の値はルートペインに入った。`pane split --env RALPH_T2=split` の値は新しいペインに入り、親のペインの `RALPH_T1` は入らなかった。
- 1 つのタブのペインを順に閉じると、最後のペインを閉じたところでワークスペースも消え、`workspace get` が `workspace_not_found` を返した。
- サーバーを SIGKILL で落とすとソケットのファイルが残り、`herdr workspace create` は `Error: Os { code: 61, kind: ConnectionRefused, message: "Connection refused" }` で終わった。ソケットのファイルが無いときは `code: 2, kind: NotFound` で終わった。どちらでも `herdr status server --json` は `running` が false の状態を返す(動いているときは `{"status":"running","running":true,"compatible":true,...}`)。
- ソケットのファイルが残っていても、`herdr server` は約 0.5 秒で `running` になった。再起動したサーバーは、ワークスペース・タブ・ペインを同じ id とラベルで戻したが、エージェントは `agent_not_found` で戻らなかった。
- サーバーが動いているところでもう 1 つ `herdr server` を起動すると、`error: herdr server is already running` を出して終了コード 1 で終わった。
- `pane split` に `--label` は無い(`herdr pane split -h`)。ラベルは split のあとの `pane rename` で付ける。

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| ralph が herdr を自動で起動する(設定のキーなし) | 人が start を打つときも director が打つときも同じに動く | 起動した環境を ralph が整える必要がある | Yes |
| 設定 `[org].herdr_autostart` で切り替える | 自動起動を止められる | 雛形と既定値の同期テストも直すので変更が増え、止めたい場面が見当たらない | No |
| director の手順書だけを直す | ralph のコードは変わらない | 人が start を打つときは自動にならない | No |
| 端末のウィンドウを開いて herdr の画面まで出す | 承諾画面にすぐ答えられる | 端末のアプリに左右され、Orca などでは狙いどおりに開かないことがある | No |
| spawn の順でペインを並べる | 確かめた操作だけで済む | reviewer を先に作った org では reviewer が右上になる | Yes |
| 役割で位置を固定する(`pane swap`) | implementer が常に右上になる | swap の振る舞いは確かめておらず、呼び出しが増える | No |

## Design decisions

- 自動起動は設定のキーを足さずに常に行う(人の回答、2026-11-06)。
- 起動するのは画面のないサーバーだけにし、端末は開かない(人の回答)。
- ペインは spawn の順で並べる(人の回答)。
- director の headless のワークスペースも同じ形に直す(人の回答)。
- 2 本の PR に分け、A を先に出す。A は B に依らず、B は停止の安全確認を移すので、別々にレビューする。

## Rollout

- A と B は、それぞれ merge したあとの次の release で配る。ralph のバイナリを入れ替えれば振る舞いが変わり、`ralph.toml` を直す必要はない。
- B を入れた ralph でも、この版より前に作った org の座席は止められ、解散できる(FR-B6)。前の版のワークスペースに新しく spawn した座席は、そのワークスペースのタブの中に入る。動いている org を新しい形にしたいときは、その org を解散して立て直す。
- B を入れる前に作った director のワークスペースも、`director stop` で閉じられる(FR-B7)。

## Security considerations

- サーバーの環境はすべてのペインに入る。Claude Code のセッションから起動すると `CLAUDE_CODE_MESSAGING_TOKEN` などが座席に渡るので、FR-A3 の変数を落とす。落とす変数の名前は、テストで固定する。
- 停止前の確認を緩めない。ペインのラベル・ワークスペースのラベルのどちらかが違えば閉じない。ラベルの無いペインを閉じてよいのは、その spawn が自分で記録した id のときだけである(FR-B6)。
- `herdr-launch.log` には herdr の起動の行だけが書かれる。環境の値は書かない。

## Open questions

なし。

## References

- 2026-11-05 の事案(director が人に herdr の起動を頼んだ文面と、ワークスペースのタブの様子)は、この仕様を作った依頼の原文にある。
- `docs/specs/2026-08-01-org-runtime.md`(org runtime の仕様)
- `docs/specs/2026-10-07-org-multi-org-director.md`(機能ごとの org と director)
- `docs/recipes/isolated-herdr-server.md`(分離した herdr のサーバー)
- herdr の CLI: https://herdr.dev/docs/cli-reference
