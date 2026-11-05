# Self-review report: org-herdr-autostart

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(spec `docs/specs/2026-11-06-org-herdr-autostart-panes.md`、S1〜S5、D1〜D5、AC1〜AC9)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff c206fdce...HEAD`(base c206fdce、HEAD 5adc55e1、25 ファイル、+2546 / -44)。コードは S1 b351bb47、S2 b8580f0a、S3 f3cca9e0。文書は S4 11c64fb1、実機の記録 S5 39e5a17c、plan の記録 5adc55e1、plan 4a99d9fc、spec a26bc8ab。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- 現在の状態: CRITICAL 0、HIGH 0、MEDIUM 1(M1)、LOW 5(L1〜L5)。

## Evidence reviewed

- `internal/org/driver/herdr.go` の `ServerStatus` と `HerdrServerStatus`、`ErrHerdrStatusJSONUnsupported`、`herdrUnknownJSONOption` を全部読んだ。出力が読めない場合(runner のエラー、壊れた JSON、`running` が無いか null)はどれも error で、推測で `running:false` を返す経路はない。`unknown option: --json` の正規表現は `--jsonl` に当たらない。エラーは `%w` で包まれ、`errors.Is(err, exec.ErrNotFound)` が `driver.ExecRunner` の折りたたみ後も効く(実機の記録 D の文面で確認できる)。
- `internal/org/herdrgate.go`、`spawn.go`(`Org.HerdrServerGate` と `Spawn` の呼び出し、step 0 と step 1 の doc)、`feature.go`(`StartFeature`)、`director_start.go`(`checkDirectorStart` の順と doc)の diff を全部読んだ。`checkDirectorStart` では、`HERDR_PANE_ID` の断りと driver・model・wake の入力検査がゲートより前にあり、ゲートは `checkCurrentDirector` と `checkWatchGate` の前にある。`launch_cmd` の形は `!c.cmdLaunch` で飛ばす。
- `internal/cli/org_herdr_server.go` を全部読んだ。`ensureOrgHerdrServer`、`await`、`afterExit`、`announce`、`orgHerdrServerEnv`、`orgHerdrServerDropsEnv`、`orgHerdrServerLaunchRequest`。`orgHerdrServerEnv` は `make([]string, 0, len+1)` で作るので、`environ` が空でも nil にならない。nil だと `startDetachedProcess` の `child.Env = req.env` が親の環境をそのまま渡し、落とす変数が全部残る。`startDetachedProcess`(`org_watch.go:477`)は `Setsid`、標準入力 `/dev/null`、出力を `herdr-launch.log`(0644)に追記する。見張りの起動と同じ形である。
- `internal/cli/org.go`、`org_director_headless.go`、`main_test.go` の diff を読んだ。`wireOrgHerdrServerGate` は start(両形)、spawn、director start の 4 か所で呼ばれる。TestMain は、ゲートを通す差し替えと、起動したら panic する差し替えの両方を置く。
- テストは `internal/cli/org_herdr_server_test.go`、`internal/org/herdrgate_test.go`、`internal/org/driver/herdr_test.go` の diff を関数名と主な本体まで読んだ。seam の復元は `t.Cleanup`、待ちの長さは `setOrgWatchDuration` で縮めてある。
- 文書は `.claude/skills/{org,director}/SKILL.md`、`docs/recipes/isolated-herdr-server.md`、`docs/evidence/org-herdr-autostart-2026-11-06.md` の diff を読み、skill が書く stderr の 1 行を `announce` の書式に照らした。証拠のファイルに秘密の値はなく、ホームは `~` に置き換えてある。
- 既存の `docs/tech-debt/README.md` の行を、herdr のサーバーの環境と起動に関わるものについて照らした(195 行目の PATH の版の食い違い、273 行目の環境が pane に届かない件)。この diff が開く行も、閉じるべき行も見つからなかった。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | Spawn の herdr ゲートが、冪等の early return(step 1)より前にある。既に spawn 済みの座席への打ち直し(何も作らない no-op)も、herdr の状態を読み、止まっていれば `herdr server` を起動し、起動できなければ `failed` で返る。Spawn の doc の step 1 は「config 依存の検証を一切しない」と言い、ゲートの 10 行下のコメントは「冪等の early return は、no-op の打ち直しが要らない検証より前に置く」(PR① の前例)と言う。ゲートはその不変条件に当たる検証だが、doc は step 0 の順だけを書き、打ち直しへの影響を書いていない。 | `internal/org/spawn.go:641`(ゲート)、`:529`(step 1 の "no config-dependent validation attempted at all")、`:651`(PR① precedent)。`internal/org/herdrgate.go` の doc も順だけを書く。`herdrgate_test.go` に、active な座席の打ち直しとゲートの組み合わせを置くテストはない。 | 台帳のロックの外で herdr の有無を見たいので、ゲートを step 1 の後ろへ移すのは難しい(plan D1 の「台帳の読み書きより前」と合わない)。現実的なのは、意図した選択として step 1 の doc と `herdrgate.go` に「ゲートは冪等の return より前にあり、active な座席の打ち直しも通る。サーバーが動いていれば状態の読み取り 1 回だけ」と書き、active な座席 + 失敗するゲート = `failed`・台帳に何も書かない、をテストで固定すること。 |
| L1 | LOW | exception-handling | 2 つの ralph が同時にサーバーを起動するとき、負けた側が誤った結果を返しうる。`afterExit` は、起動した子が終わったあとに状態を 1 回だけ読む。勝った側のサーバーが応答する前(plan の Assumptions では約 0.5 秒かかる)に負けた側の子が終わると、その 1 回は `running:false` になり、負けた側の動詞が「exited before it answered」で失敗する。逆に、勝った側が応答した時点で負けた側の子がまだ終わっていないと、`orgHerdrChildExited` が false で、負けた側が自分の pid で "started the herdr server" の 1 行を出す(動いているのは別の ralph のサーバー)。herdr が 2 つ目をどの時点で終わらせるかは、実機で確かめていない(記録は A〜D の逐次のみ)。 | `internal/cli/org_herdr_server.go:204`(`afterExit`)、`:176-:190`(`await` の `orgHerdrChildExited` による announce の分岐)。`TestOrgHerdrServerGate_AServerThatExitsFirst` の読み取りは `[not, not, running]` に固定され、勝った側が起動途中の読み取りはない。 | `afterExit` を、`await` の残りの時間のあいだ読み続ける形にする(子が終わったあとも deadline まで読み、running になれば無言で通す)。直さないなら、doc とエラー文に「同時の起動では片方が失敗しうる。打ち直す」と書く。 |
| L2 | LOW | security | 起動したサーバーの環境の残し方が非対称である。`CLAUDE_CODE_*` は接頭辞で落とすが、`CODEX_*` は丸ごと残す。理由は `CODEX_HOME` を座席の model 観測が読むことだけである。テスト `TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly` は `CODEX_SANDBOX=seatbelt` を「残る」側に固定している。Codex の director のセッションから打つと、そのセッション由来の変数(`CODEX_SANDBOX*` など。実際にどの名前があるかは未確認)が、サーバーを通して claude の座席を含む全 pane に入る。`CLAUDE_CODE_*` を落とす理由(コメントの「サーバーの環境はすべてのペインに入る」)が、同じ形でここにも当てはまる。plan の R1 は `CLAUDE_CODE_` で始まらない新しい名前しか書いていない。 | `internal/cli/org_herdr_server.go:290`(コメント)、`:315`(`orgHerdrServerDropsEnv`)。`internal/cli/org_herdr_server_test.go:489` の `kept` に `CODEX_SANDBOX=seatbelt`。spec FR-A3 が「`CODEX_` で始まる変数は残す」と決めている。 | 残す `CODEX_*` を `CODEX_HOME` に絞る(model 観測の理由を満たす最小)。仕様の判断どおり丸ごと残すなら、残るリスクとして plan の Risks か `orgHerdrServerEnv` の doc に 1 行足し、テストの `CODEX_SANDBOX=seatbelt` の理由を書く。 |
| L3 | LOW | maintainability | `HerdrServerStatus.Compatible` は `ServerStatus` が埋めるだけで、本番のコードは読まない。型の doc は「org の動詞は Running 以外を読まないので、ほかのキーは保持しない」と言いながら、読まれない Compatible を持つ。ゲートの doc は「`compatible` が何を言っても通る」と書き、テストがそれを固定するためにだけ使う。 | `internal/org/driver/herdr.go:666`(型)、`:731`(代入)。`grep '\.Compatible'`(テスト以外)が出すのはこの代入だけ。 | 残すなら型の doc に「ゲートは読まない。版の違うサーバーの対処(この plan の非目標)に備えて持つ」と書く。消すなら null の扱いの説明も要らなくなる。 |
| L4 | LOW | maintainability | 待ちの 10 秒が、help の文、skill の文、`orgHerdrServerWaitDeadline` の既定値の 3 か所にあり、突き合わせるものがない。help のテストは help の文面の "waits up to 10s" を固定するが、変数の既定値は見ない。変数だけ変えても、テストは通ったまま help と skill が古い値を言う。 | `internal/cli/org_herdr_server.go:43`(help の "waits up to 10s" と "within the 10s")、`:82`(`orgHerdrServerWaitDeadline = 10 * time.Second`)、`org_herdr_server_test.go:728` の `TheHelpOfEachVerbSaysItStartsTheServer`。 | 既定値が `10 * time.Second` であることを 1 つのテストで固定する(テストは変数を縮めるので、既定値の定数を別に置くか、縮める前の値を読む)。help の文字列を変数から組み立てる案もあるが、grep しにくくなるので勧めない。 |
| L5 | LOW | readability | `await` のループが暗黙である。`left` を `for` の後処理で更新したうえで、本体でも `if left = time.Until(deadline); left <= 0 { break }` で上書きする。`ensureOrgHerdrServer` の `switch` は、`err == nil && !st.Running` のとき、どの case にも入らず落ちて起動に進む。どちらも読めば分かるが、前者は「sleep の前に deadline を見直す」意図が、後者は「ここに来るのは動いていないと読めたとき」が、コードに出ていない。 | `internal/cli/org_herdr_server.go:176`、`:185`(`await`)。`ensureOrgHerdrServer` の `switch`(`case err == nil && st.Running` から `case err != nil` まで)。 | 本体の上書きに 1 行コメントを足し、`switch` の後ろに「err == nil で running でない。起動する」と 1 行足す。 |

## Positive notes

- 状態を読めない場合は推測せず失敗にし、通すのは `unknown option: --json` の古い herdr だけにしている。その判定も `--jsonl` を含めない正規表現で狭めてあり、plan の D3 が言う「通すと worktree が残る」事案を避ける形になっている。
- `Org.HerdrServerGate` は nil なら何もしない。`WatchGate` と同じ形で、org 層は CLI の起動の方法を知らない。ゲートの位置の理由(`checkCurrentDirector` が herdr に聞く、dry-run は聞かない)は doc とコメントに書かれ、`launch_cmd` の形で呼ばない点も揃っている。
- `TestMain` の差し替えは、ゲートを通す 1 つと、本物の起動を panic にする 1 つの 2 本立てで、panic の文面が直し方(`set orgHerdrServerLaunch to a fake`)を言う。それを確かめるテスト(`TestMain MakesARealLaunchPanic`)もある。
- 環境の整理は純関数(`orgHerdrServerEnv`)で、環境の入力と期待を名前で固定している。`PWD` の置き換えと、`=` を含まない行、`CLAUDE_CODE`・`CLAUDECODE_X`・`CLAUDE_CODE_` のような境界の名前までテストにある。
- 実機の記録は、確かめていないこと(`start --plan`・`director start` の実機、古い herdr、座席の agent の起動)を節にして書いている。環境の値や token は写さず、ホームは `~` にしてある。
- 大きいファイルの行数: `spawn.go`(2767)、`director_start.go`(1260)、`org.go`(2182)は base から 800 行を超えており、この diff の追加は 27 / 19 / 12 行。新しいファイルは `org_herdr_server.go` が 317 行、テストが 756 行で、800 行の内に収まる。

## Notes (finding にしなかったもの)

- `ServerStatus` のエラー文は herdr の標準出力を `%q` で丸ごと含む。出力が長い herdr だとエラーが長くなるが、読めない出力のときだけである。
- `orgHerdrServerDropsEnv` は見張りの `orgWatchChildDroppedEnv` を共有する。見張りの側で名前を足し引きすると herdr のサーバーの環境も変わるが、`TestOrgHerdrServerEnv_...` が 4 つの名前をリテラルで固定するので、変えるとテストが落ちる。
- ゲートを 2 回呼ぶ経路: `StartFeature` の中の Spawn がゲートをもう一度呼ぶ(plan D1 に書いてある)。worktree を作ったあとにサーバーが止まると、その Spawn が失敗して worktree が残る。この diff が作る新しい形ではなく、Spawn の失敗全般と同じ扱いである。
- skill の `/director` の文(「立てる」の 5)は、ゲートが登録の確認より前にあるため、昇格したセッションの登録で断られた start でも、止まっていたサーバーが起動したまま残ると書く。M1 と同じ種類の副作用を、director start の側ではすでに文書にしている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | cycle 1 で、M1 と L1〜L5 は直せる範囲にある。直さないものは、cycle の最後の run で defer として `docs/tech-debt/README.md` に足す | - | - |

_(行を足していないので、`docs/tech-debt/` への追記もしていない。)_

## Recommendation

- Merge: yes。CRITICAL と HIGH はない。M1 は挙動を壊さず、doc とテストの追加で閉じられる。
- Follow-ups:
  - M1: Spawn の step 1 の doc と `herdrgate.go` に、ゲートが冪等の return より前にあることと打ち直しへの影響を書き、active な座席 + 失敗するゲートのテストを足す。どちらも org 層だけの小さな変更。
  - L1: 実機で 2 つの `ralph org spawn` を同時に打ち、負けた側の挙動を一度見ておくと、`afterExit` を変えるか文書にするかを決められる。見るまでは未確認である。
  - L2: 仕様の FR-A3 を変える話になるので、残すか絞るかは仕様の判断を仰ぐ。絞るなら FR-A3 と AC-A2、`kept` のテスト、plan の AC2 が揃う。
  - L3〜L5 は小さい。M1 か L1 を直す run のついでに直せる。
  - 直す場合は、cycle 2 で `/self-review` から全パイプラインを回す。
