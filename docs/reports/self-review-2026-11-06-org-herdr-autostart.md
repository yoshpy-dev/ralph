# Self-review report: org-herdr-autostart

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(spec `docs/specs/2026-11-06-org-herdr-autostart-panes.md`、S1〜S5、D1〜D5、AC1〜AC9)
- Reviewer: reviewer サブエージェント(`/self-review`)。Run 1 はパイプラインの cycle 1、Run 2 は cycle 2(上限 2 の最後の run)
- ID: Run 1 の所見は M1 と L1〜L5。`docs/tech-debt/README.md` の 433・434 行が L1〜L4 を引くので、番号は振り直さず、Run 1 の所見の本文は「Run 1 の所見の本文」にそのまま残した。Run 2 の新しい所見は C2-1。
- Scope(Run 1): `git diff c206fdce...HEAD`(base c206fdce、HEAD 5adc55e1、25 ファイル、+2546 / -44)。コードは S1 b351bb47、S2 b8580f0a、S3 f3cca9e0。文書は S4 11c64fb1、実機の記録 S5 39e5a17c、plan の記録 5adc55e1、plan 4a99d9fc、spec a26bc8ab。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- Scope(Run 2): `git diff c206fdce...HEAD`(同じ base、HEAD f0edc152、39 ファイル、+3169 / -71)。Run 1 のあとに増えたのは、verify・test・sync-docs・cross-review の記録(d77ee3c3 から 944790fd)と、修正の f0edc152(7 ファイル、+120 / -7)。f0edc152 は全部読んだ。sync-docs の d6951690 は文書の変更(skill の写し、README、AGENTS.md、quality-gates、recipe、org runtime の spec)で、Run 1 の範囲の外だったので、その diff も読んだ。diff の質だけを見た。ビルド、テスト、静的解析、仕様適合は見ていない。
- 現在の状態(Run 2): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(L1〜L4 と C2-1)。M1 と L5 は f0edc152 で解消した。L1〜L4 は直さずに `docs/tech-debt/README.md` の 433・434 行に入れてある(defer)。C2-1 は Run 2 の新しい LOW。cycle 2 は上限の最後の run なので、直さない C2-1 は defer になる(「Tech debt identified」に行の案を置いた)。
- Run 1 の状態: CRITICAL 0、HIGH 0、MEDIUM 1(M1)、LOW 5(L1〜L5)。

## Evidence reviewed

### Run 1(cycle 1)

- `internal/org/driver/herdr.go` の `ServerStatus` と `HerdrServerStatus`、`ErrHerdrStatusJSONUnsupported`、`herdrUnknownJSONOption` を全部読んだ。出力が読めない場合(runner のエラー、壊れた JSON、`running` が無いか null)はどれも error で、推測で `running:false` を返す経路はない。`unknown option: --json` の正規表現は `--jsonl` に当たらない。エラーは `%w` で包まれ、`errors.Is(err, exec.ErrNotFound)` が `driver.ExecRunner` の折りたたみ後も効く(実機の記録 D の文面で確認できる)。
- `internal/org/herdrgate.go`、`spawn.go`(`Org.HerdrServerGate` と `Spawn` の呼び出し、step 0 と step 1 の doc)、`feature.go`(`StartFeature`)、`director_start.go`(`checkDirectorStart` の順と doc)の diff を全部読んだ。`checkDirectorStart` では、`HERDR_PANE_ID` の断りと driver・model・wake の入力検査がゲートより前にあり、ゲートは `checkCurrentDirector` と `checkWatchGate` の前にある。`launch_cmd` の形は `!c.cmdLaunch` で飛ばす。
- `internal/cli/org_herdr_server.go` を全部読んだ。`ensureOrgHerdrServer`、`await`、`afterExit`、`announce`、`orgHerdrServerEnv`、`orgHerdrServerDropsEnv`、`orgHerdrServerLaunchRequest`。`orgHerdrServerEnv` は `make([]string, 0, len+1)` で作るので、`environ` が空でも nil にならない。nil だと `startDetachedProcess` の `child.Env = req.env` が親の環境をそのまま渡し、落とす変数が全部残る。`startDetachedProcess`(`org_watch.go:477`)は `Setsid`、標準入力 `/dev/null`、出力を `herdr-launch.log`(0644)に追記する。見張りの起動と同じ形である。
- `internal/cli/org.go`、`org_director_headless.go`、`main_test.go` の diff を読んだ。`wireOrgHerdrServerGate` は start(両形)、spawn、director start の 4 か所で呼ばれる。TestMain は、ゲートを通す差し替えと、起動したら panic する差し替えの両方を置く。
- テストは `internal/cli/org_herdr_server_test.go`、`internal/org/herdrgate_test.go`、`internal/org/driver/herdr_test.go` の diff を関数名と主な本体まで読んだ。seam の復元は `t.Cleanup`、待ちの長さは `setOrgWatchDuration` で縮めてある。
- 文書は `.claude/skills/{org,director}/SKILL.md`、`docs/recipes/isolated-herdr-server.md`、`docs/evidence/org-herdr-autostart-2026-11-06.md` の diff を読み、skill が書く stderr の 1 行を `announce` の書式に照らした。証拠のファイルに秘密の値はなく、ホームは `~` に置き換えてある。
- 既存の `docs/tech-debt/README.md` の行を、herdr のサーバーの環境と起動に関わるものについて照らした(195 行目の PATH の版の食い違い、273 行目の環境が pane に届かない件)。この diff が開く行も、閉じるべき行も見つからなかった。

### Run 2(cycle 2)

- f0edc152 の diff を 7 ファイルとも読んだ。コードは `spawn.go`、`herdrgate.go`、`org_herdr_server.go`。テストは `herdrgate_test.go` と `org_herdr_server_test.go`。文書は evidence と tech-debt。
- M1: `spawn.go` の step 0 の doc、step 1 の doc(538 行以降)、ゲートの呼び出し(648 行)、PR① のコメント(659-663 行)、`herdrgate.go` の先頭(20-25 行)を読む順に並べ、同じ事実を言っていることを確かめた。事実とは、ゲートは冪等の return より前にあり、active な座席の打ち直しも通ること、サーバーが動いていれば読み取りは 1 回であること、失敗は `SpawnOutcomeFailed` で何も書かないことである。plan の D1 は「台帳の読み書きと herdr の呼び出しより前」と言い、打ち直しへの影響は書いていない(Notes)。
- M1 のテスト `TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo`(`herdrgate_test.go:95`): ゲートを step 1 の後ろへ移すと、前半(active な座席 + 失敗するゲートが `SpawnOutcomeFailed`)が赤くなる。後半は、通るゲートが 1 回だけ呼ばれ、冪等の答えが元と同じ pane で、台帳・receipt・driver 呼び出しの数が変わらないことを見る。コードを読んで導いたもので、動かしていない。
- C21: `TestOrgStartPlan_HerdrServerGate_StartsAStoppedServerOnceInStartsWords`(`org_herdr_server_test.go:731`)。`herdrGateFake`(`setOrgHerdrServerSeams`)は答えを順に出し、出し切ると最後を繰り返す。答えは「止まっている、動いている」なので、StartFeature のゲートが 2 回、Spawn のゲートが 1 回の計 3 回読み、起動は 1 回になる。StartFeature のゲートを外すと、Spawn のゲートが起動して行の動詞が `spawn` になり、期待の `ralph org start:` で赤くなる。Spawn のゲートを外すと読み取りが 2 回になって赤くなる。名前の接頭辞は同じファイルの `TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree` に揃う。
- L5 と I3: `org_herdr_server.go:142` と `:186-187` のコメントを、`ensureOrgHerdrServer` の switch と `await` のループの実際の分岐に照らした。help の文字列(:48)とそれを固定するテスト(`org_herdr_server_test.go:772`)は合っている。`refused with nothing made` と `command is refused` を repo 全体で grep し、残るのは verify report の引用だけだった。
- L1: `afterExit` の doc(:208-213)が書く 2 つの経路を、`await` の `orgHerdrChildExited` の分岐と `afterExit` の 1 回の読み取りに照らし、書かれたとおりに起きることを読んで確かめた。2 つの start を並べた実機の確かめはない(doc も「見ていない」と書く)。
- register: 318 行の末尾の注(verify V1)、431 行(M1 の挙動)、433 行(L1)、434 行(L2〜L4)を読んだ。433・434 行の主張を HEAD に照らした。`HerdrServerStatus.Compatible` を読む本番のコードは無い(本番の hit は `driver/herdr.go:731` の代入だけ)。`orgHerdrServerDropsEnv` は `CLAUDECODE` と `CLAUDE_CODE_` の接頭辞だけを落とし、`CODEX_` は残す(:300-326)。10 秒は help(:43)、`/org` の skill(:28)、`/director` の skill(:624)、`orgHerdrServerWaitDeadline`(:82)にある。318・433・434 行は ` | ` で数えて 5 列である。
- 実機の記録(I1): `docs/evidence/org-herdr-autostart-2026-11-06.md` の AC2 の表(55 行)とその下の段落(61 行)を、verify I1 の推奨文と plan の AC2 に照らした。
- sync-docs の d6951690: `.claude/skills/{org,director}/SKILL.md` と `.agents/skills/` の写し、`templates/base/` の写し、2 つの recipe の template の写しを `cmp` で確かめた(同一)。`quality-gates.md` は root と template が元から違う(`scripts/check-sync.sh` が既知の差として扱う)が、この PR の Herdr server gate の行は両方で同一である。AGENTS.md の変更は、`internal/org/` の項に `herdrgate.go` の説明を 1 つ足しただけだった。

## Findings

### Run 1 の所見の状態(Run 2 時点)

| ID | Severity | 状態 | 根拠 |
| --- | --- | --- | --- |
| M1 | MEDIUM | 解消(f0edc152) | `spawn.go` の step 1 の doc と PR① のコメントが、ゲートは冪等の return より前で、打ち直しも通ると言う。`herdrgate.go` の先頭も同じ。テスト `TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo` が「active な座席 + 失敗するゲート = `SpawnOutcomeFailed`、台帳・receipt・driver は動かない」を固定する。副作用そのもの(止まっていたサーバーが起動したまま残る)は register の 431 行にある。 |
| L1 | LOW | defer(register 433 行) | `afterExit` を締め切りまで読み続ける形には直さず、doc(:208-213)に 2 つの結果と「打ち直せば進む」を書いた。433 行に Impact、直さない理由(失敗が最大 10 秒遅れる)、Trigger(2 つの start を実機で並べる)がある。 |
| L2 | LOW | defer(register 434 行 (a)) | 仕様の FR-A3 を変える話なので残した。434 行 (a) が Trigger(`CODEX_HOME` だけに絞る)を持つ。 |
| L3 | LOW | defer(register 434 行 (b)) | 434 行 (b) は「読まれないフィールド」の半分を書く。もう半分、型の doc が「ほかのキーは保持しない」と言う点(`driver/herdr.go` の `HerdrServerStatus` の doc の "they read nothing else here, so the other keys ... are not kept")は書いていないが、(b) の Trigger(`Compatible` を読むか消す)で一緒に解ける。 |
| L4 | LOW | defer(register 434 行 (c)) | 434 行 (c) が 3 か所と Trigger(既定値の定数を別に置いて固定する)を書く。 |
| L5 | LOW | 解消(f0edc152) | `ensureOrgHerdrServer` の switch のあと(:142)と `await` の `left` の取り直し(:186-187)にコメントが付いた。どちらも実際の分岐と合う。 |

Run 1 の外から回ってきた項目:

| 項目 | 状態 | 根拠 |
| --- | --- | --- |
| test C21(start --plan を本物のゲートで通すテストが無い) | 解消(f0edc152) | 上の C21 の項。起動 1 回、読み取り 3 回、start の言葉の 1 行を固定する。 |
| verify I1(実機の記録の `HERDR_TAB_ID`) | 解消(f0edc152) | 表は「herdr がどのペインにも付ける値。ralph の値を落としたかは、この方法では確かめられない」になり、段落が、落とした根拠は `CODEX_PROBE` が届き `CLAUDECODE` と `CLAUDE_CODE_PROBE` が無いこと、`HERDR_*` はユニットテストが固定すること、と分けて書く。 |
| verify I3(help の「refused」) | help は解消。skill に同じ語が残る | help は「fails」になり、テストも合っている。skill の同じ事象の語が直っていないのが C2-1。 |
| verify V1(別のソケットを指す端末の director start) | defer(register 318 行の注) | 318 行の末尾の注が、入口が増えたこと、コードを読んでの推論で確かめていないこと、直すには登録に herdr のサーバーを記録する必要があることを書く。 |

### Run 2 の新しい所見

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | readability | verify I3 は、herdr のゲートの失敗を help が「refused」と書くが、実際は `spawn failed:`(`rejected:` ではない)になる、という指摘だった。f0edc152 は help の文字列とその固定テストを「fails」にしたが、skill の同じ事象の語は直っていない。`/org` skill の前提は「何も作らずに断られ」、`/director` skill の director start の手順は「何も書かずに断られ」と書く。`/org` skill 自身は、`断られる` を `rejected:` の結果(見張りのゲートの断り)に使い、herdr のゲートは `spawn failed:` で `rejected` ではないと同じ行で区別している。`/director` skill の start --plan の手順は同じ失敗を「失敗し」と書くので、2 つの skill の中でも語が揃わない。leader の雛形は `spawn failed:` を「それ以外のエラー」として BLOCKED に回すので、動きは変わらない。語の問題だけである。I3 は verify の所見だが、Run 1 もこの skill の語を見ていなかった。 | `.claude/skills/org/SKILL.md:35`(「何も作らずに断られ」)、`:335`(`spawn` の行。見張りのゲートの断りは `rejected: ...`、herdr のゲートは「`spawn failed:` で終わる(入力の拒否の `rejected` ではない)」)。`.claude/skills/director/SKILL.md:162`(「何も書かずに断られ」)、`:624`(「worktree もブランチも作らずに失敗し」)。`.agents/skills/` と `templates/base/` の写し(4 通り × 2 skill)は `cmp` で同一なので、8 ファイルに同じ語がある。 | 直すなら 35 行と 162 行を「失敗し」に揃え、`scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の 4 ファイルをそろえる。skill が変わるので全パイプラインを回し直すことになり、cycle 2 は上限の最後の run なので、上限を上げない限り直せない。直さないなら、register の 434 行に (d) として足す(下の「Tech debt identified」に文案)。 |

### Run 1 の所見の本文(Run 1 のまま)

下の表の `file:line` は Run 1 の HEAD(5adc55e1)での行番号である。f0edc152 でコメントが増えたので、`spawn.go` と `org_herdr_server.go` の行は今の位置とずれる(`afterExit` は 204 行から 214 行へ)。今の位置は、上の Run 2 の Evidence と状態の表にある。

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
- Run 2: M1 の直しは、矛盾が読まれる場所に入っている。step 1 の doc(「config 依存の検証を一切しない」と言う段落)の末尾と、PR① のコメント(冪等の early return より前に置け、と言う側)の両方に、ゲートは例外であることを書き、「on purpose」の理由(manifest の読みが座席を見つけるので、ゲートはその前)まで添えた。`herdrgate.go` の先頭も同じ事実を言い、3 か所が食い違わない。
- Run 2: M1 のテストは、失敗側と通る側を 1 つのテストに入れ、どちらも台帳の event 数、receipt 数、driver の呼び出し数、agmsg の呼び出し数の snapshot で「何も変わらない」を見る。返り値だけを見るテストではない。
- Run 2: I1 の直しは、確かめられないことを「確かめられない」と書くだけでなく、代わりの根拠(対照の `CODEX_PROBE` と、`HERDR_*` を固定するユニットテストの名前)を指す。
- Run 2: L1 の doc は、実機で見ていないと先に書き、起きうる 2 つの結果を書く。register の 433 行も同じ姿勢で、Trigger が「先に 2 つの start を実機で並べる」になっている。
- Run 2: 433・434 行は Impact、直さない理由、Trigger を別の列に書き、Trigger が次の具体的な手になっている。列の数は 5 で崩れていない。

## Notes (finding にしなかったもの)

- `ServerStatus` のエラー文は herdr の標準出力を `%q` で丸ごと含む。出力が長い herdr だとエラーが長くなるが、読めない出力のときだけである。
- `orgHerdrServerDropsEnv` は見張りの `orgWatchChildDroppedEnv` を共有する。見張りの側で名前を足し引きすると herdr のサーバーの環境も変わるが、`TestOrgHerdrServerEnv_...` が 4 つの名前をリテラルで固定するので、変えるとテストが落ちる。
- ゲートを 2 回呼ぶ経路: `StartFeature` の中の Spawn がゲートをもう一度呼ぶ(plan D1 に書いてある)。worktree を作ったあとにサーバーが止まると、その Spawn が失敗して worktree が残る。この diff が作る新しい形ではなく、Spawn の失敗全般と同じ扱いである。
- skill の `/director` の文(「立てる」の 5)は、ゲートが登録の確認より前にあるため、昇格したセッションの登録で断られた start でも、止まっていたサーバーが起動したまま残ると書く。M1 と同じ種類の副作用を、director start の側ではすでに文書にしている。
- Run 2: M1 の「on purpose」の注は、ゲートの呼び出し(648 行)の直前ではなく、`resolvedPermMode` のコメント(659-663 行)の中にある。ゲートの直前のコメントは打ち直しに触れない。矛盾が読まれる場所と step 1 の doc には入っていて足りるので、所見にしなかった。
- Run 2: plan の D1 は位置(台帳の読み書きと herdr の呼び出しより前)を決めるだけで、no-op の打ち直しが通ることは書かない。コードの doc は「D1 が manifest の読みより前に置くので」と導いていて、それは D1 から出る。判断を記録したのは cycle 1 の triage だが、cycle 2 の `/cross-review` が同じファイルを書き直すので、残るのは doc と register の 431 行になる。両方に入っているので足りる。
- Run 2: `internal/cli/org_herdr_server_test.go` は 794 行で、800 行の 6 行下にある。434 行 (c) の Trigger が勧める「既定値を固定するテスト」を同じファイルに足すと 800 を超える。そのときはファイルを分ける。
- Run 2: 疑って外れたもの。新しいテストの名前の接頭辞(`TestOrgStartPlan_HerdrServerGate_`)が、`TestOrgHerdrServerGate_` で絞る実行から漏れるかと見た。同じファイルの verb ごとのテスト(`TestOrgSpawn_HerdrServerGate`、`TestOrgStart_HerdrServerGate_…`、`TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree`、`TestOrgDirectorStart_HerdrServerGate_…`)が同じ型の接頭辞を使い、`HerdrServerGate` で grep すれば全部出るので、所見にしなかった。
- Run 2 の引き継ぎ(食い違いではない): register が指す先のうち、triage(433・434 行の Related)と verify report(318 行の V1、433 行の AC5)は、cycle 2 の `/cross-review` と `/verify` が同じファイルを書き直しうる。V1 と AC5 の節、cycle 1 の triage の判断の文が残るようにするか、行の側を commit(944790fd など)で指す。この report は Run 1 の本文を残したので、L1〜L4 の指し先は消えない。
- Run 2 の引き継ぎ: plan の AC1〜AC9 と Progress の Review・Verification・Test の artifact は `[ ]` のまま(verify I2 の h が拾っている)。この run は report と insight event だけをコミットするので付けていない。cycle 1 の sync-docs の report(97 行)は I1 と I3 を「この run の範囲外」と書くので、cycle 2 の `/sync-docs` が書き換える。

## Tech debt identified

Run 1 の時点の記載。cycle 1 は最後の run ではなかったので、defer は Run 2 で register に入った(下の Run 2)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | cycle 1 で、M1 と L1〜L5 は直せる範囲にある。直さないものは、cycle の最後の run で defer として `docs/tech-debt/README.md` に足す | - | - |

_(行を足していないので、`docs/tech-debt/` への追記もしていない。)_

Run 2: L1〜L4 は f0edc152 で register に入った(433・434 行)。318 行には verify V1 の注が付いた。C2-1 はまだ register に無い。この run は report と insight event だけをコミットするので、行の案をここに置く。`/sync-docs` が 434 行の (d) か独立した行として足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1: the `/org` skill (prerequisites, `.claude/skills/org/SKILL.md:35`) and the `/director` skill (step 5 of the start, `.claude/skills/director/SKILL.md:162`) say that a herdr server gate that cannot start a server 'is refused' (`断られ`), the word the `/org` skill uses for the `rejected:` outcome of the watch gate, while its `spawn` row says the gate ends with `spawn failed:`, not `rejected`; the start --plan step of the `/director` skill (`:624`) says it 'fails' (`失敗し`), as the `--help` text does since f0edc152 | A reader of a skill can take the gate's failure for a `rejected` outcome; the leader template routes `spawn failed:` as BLOCKED, so no behavior changes | LOW, wording only. Fixing it changes 8 skill files (`.claude/skills` and `.agents/skills`, and the two `templates/base` copies of each of the two skills) and needs the full pipeline again, and cycle 2 was the last run | The next change to the herdr paragraph of either skill: write `失敗し` as the `--help` text and the start --plan step of the `/director` skill do | `docs/reports/self-review-2026-11-06-org-herdr-autostart.md` (C2-1) |

_(`docs/tech-debt/` には足していない。)_

## Recommendation

- Merge: yes。CRITICAL と HIGH と MEDIUM はない。M1 と L5 は f0edc152 で解消し、f0edc152 は MEDIUM 以上の新しい所見を作っていない。残る LOW は 5 件で、L1〜L4 は register に入り、C2-1 は語だけの指摘で挙動を変えない。
- Follow-ups(Run 2、cycle 2 = 上限の最後の run):
  - C2-1: 直さなければ defer になる。直すなら skill 8 ファイルの変更で、全パイプラインを回し直すので、上限を上げる判断が要る。語だけなので、上の「Tech debt identified」の行を `/sync-docs` が register に足して先へ進むのが軽い。
  - M1 と L5 は解消。L1〜L4 は register の 433・434 行にあり、主張は HEAD のコードと合う。433・434 行の Trigger は、いずれも次にそのコードを触るときの具体的な手になっている。
  - f0edc152 の 3 つの追加(テスト 2 本、doc、register の行)に、コードの指摘はない。
  - 引き継ぎは Notes の最後の 2 項(register が指す verify と triage の節を cycle 2 の再実行で消さない、plan の `[ ]` と古い sync-docs の記述は cycle 2 の `/verify` と `/sync-docs` が扱う)。
  - この run は diff の質だけを見た。ビルドとテストは動かしていないので、新しい 2 本のテストが通ることは `/test` が確かめる。
- Run 1 の判定: yes。CRITICAL と HIGH はない。M1 は挙動を壊さず、doc とテストの追加で閉じられる。
- Follow-ups(Run 1):
  - M1: Spawn の step 1 の doc と `herdrgate.go` に、ゲートが冪等の return より前にあることと打ち直しへの影響を書き、active な座席 + 失敗するゲートのテストを足す。どちらも org 層だけの小さな変更。
  - L1: 実機で 2 つの `ralph org spawn` を同時に打ち、負けた側の挙動を一度見ておくと、`afterExit` を変えるか文書にするかを決められる。見るまでは未確認である。
  - L2: 仕様の FR-A3 を変える話になるので、残すか絞るかは仕様の判断を仰ぐ。絞るなら FR-A3 と AC-A2、`kept` のテスト、plan の AC2 が揃う。
  - L3〜L5 は小さい。M1 か L1 を直す run のついでに直せる。
  - 直す場合は、cycle 2 で `/self-review` から全パイプラインを回す。
