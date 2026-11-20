# org-reseat-restart-herdr

- Status: Approved
- Approved: 2026-11-20 sha256:f5d55e0b6ca2
- Owner: Claude Code
- Date: 2026-11-20
- Related request: issue #354。座席を別の Claude のアカウントへ移すとき、herdr のサーバーを新しいアカウントの環境で起動し直す手順を人に頼むほかない。仕様は `docs/specs/2026-11-20-org-reseat-restart-herdr.md`(PR #355 で merge、consult の指摘 3 点を入れて自律に承認)。系列の指示で計画は推す案で自動承認
- Related issue: #354
- Type: feat
- Branch: feat/org-reseat-restart-herdr

## Objective

`ralph org reseat` に `--restart-herdr` と `--claude-config-dir` を足し、座席を別のアカウントへ移すときに、人が herdr のサーバーを止めなくても、director か人が 1 回の reseat で済ませられるようにする。herdr のサーバーにその org の座席のほかの pane が 1 つでもあれば断り、ほかの pane を黙って消さない。reseat は、動いているサーバーの `CLAUDE_CONFIG_DIR` を読めたら出す。

## Scope

仕様の FR-1〜FR-8 をすべて入れる。

- herdr のドライバ(`internal/org/driver/herdr.go`): 全部の pane の一覧(`herdr pane list`、workspace を絞らない)と、`herdr server stop` の 2 つの呼び出しを足す
- サーバーの環境(`internal/cli/org_herdr_server.go` と新しいファイル): `--claude-config-dir` の値でゲートが起動するサーバーの `CLAUDE_CONFIG_DIR` を置き換える、`''` で外す。動いているサーバーの `CLAUDE_CONFIG_DIR` を `herdr status server --json` の socket → `lsof -t` → `ps eww`(Linux は `/proc/<pid>/environ`)で読む。呼び出しはどれも期限つきで、テストでは差し替えられる形にする
- reseat(`internal/org/reseat.go`): `ReseatParams` に立て直しの指定と、全部の pane の一覧・サーバーを止める・止まったかを読む関数を足す(今の `WatchProbe`、`HerdrServerProbe` と同じく関数の欄)。順は、今の確かめ(`PlanReseat`)→ 止める前の確かめ(呼んだ側が pane の中でない、生きた pane がその org の座席の pane だけ)→ 座席を止める(`stopAll`)→ もう一度 pane の確かめ → `herdr server stop` → running でなくなるまで待つ(10 秒)→ ゲートで起動(`checkHerdrServerGate`)→ 座席を立てる(`startAll`)。`--restart-herdr` は、その org の active な座席の全部を対象にするときだけ受ける(`--seat` で一部だけを選んでいれば断る。サーバーを止めると選ばれなかった座席も死ぬため。Codex の計画助言の指摘 1)。どれかで断れば、その時点までの状態を今の reseat の失敗の扱いで残し、打ち直しの命令(`reseatRetryCommand`)を、どこまで進んだかで変える: サーバーを止める前に断ったなら `--restart-herdr` と `--claude-config-dir` を写し、サーバーを起動し直したあとに断った(leader が立ち、ほかの座席が立たなかった、など)なら `--restart-herdr` を外して `--claude-config-dir` だけを写す(立ち直った座席をもう一度殺さないため)。最後の pane の確かめと `herdr server stop` のあいだには、ほかの呼び出しを置かない
- CLI(`internal/cli/org_reseat.go`): 2 つの flag、`--dry-run` での確かめの表示、サーバーの `CLAUDE_CONFIG_DIR` の 1 行、移したい値と違うときの知らせ、TUI も閉じることの 1 行、断る文(ほかの pane の id・label・cwd と、記録だけが戻った pane の見分け方)、help
- 文書: `ralph org reseat --help`、`/director` skill の「座席を立て直す」3 と「しないこと」(人に頼まれたときに `--restart-herdr` を打ってよい例外)、`/org` skill の同じ手順(どちらも 4 写し)、README の `ralph org reseat` の行、`docs/specs/2026-11-12-org-reseat.md` に追記、AGENTS.md の reseat の説明に 1 句

## Non-goals

- 複数の org をまとめてほかのアカウントへ移すこと
- `ralph status` と `ralph doctor` での見える化
- keychain の扱いの確かめ、herdr の `live_handoff`
- 見張りの ALERT を立て直しのあいだ止めること(座席を止めてからサーバーを止めるので、active な座席の liveness は上がりにくい)

## Assumptions

- `herdr pane list`(workspace なし)は、サーバーの全部の pane を返す(herdr 0.7.5。2026-11-20 にこのマシンで、別の repo の org の 3 つと shell の 1 つを返した)
- `herdr server stop` はすべての pane を閉じ、プロセスを殺す(`docs/evidence/org-reseat-live-2026-11-12.md:240-268`)。起動し直すと記録だけの pane が `pane list` に戻る(同 263〜270 行)
- サーバーの環境は `lsof -t <socket>` と `ps eww` で読める(このマシン、macOS 26.5.2、herdr 0.7.5 で、pid 1 つと環境 96 項目を手で確かめた)
- ゲート(`ensureOrgHerdrServer`)が起動するサーバーの環境は、呼んだ側の環境から pane の変数と `CLAUDECODE`・`CLAUDE_CODE_*` を除いたもの(`orgHerdrServerEnv`)

## Affected areas

- `internal/org/driver/herdr.go` とテスト
- `internal/org/reseat.go`、`internal/org/reseat_test.go`(と、立て直しの新しいテストのファイル)
- `internal/cli/org_herdr_server.go`、新しい `internal/cli/org_herdr_server_env.go` とテスト、`internal/cli/org_reseat.go`、`internal/cli/org_reseat_test.go`
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md`(templates の写し、`.agents` の写しは `scripts/sync-skills.sh`)、`README.md`、`docs/specs/2026-11-12-org-reseat.md`、`AGENTS.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-reseat-restart-herdr.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-order`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 3 枚の図を `shot` で撮って読み、重なりも切れもなかった

## Design decisions

Critical forks: None。仕様の D1〜D5 に従う。計画の段で足した判断は次の 2 つ(系列の指示で自律に選んだ)。

1. 止める前の確かめは 2 回(座席を止める前と、止めたあとサーバーを止める直前)。1 回目で断れば何も変えない。2 回目で断れば座席は止まったまま残るので、今の reseat の失敗の文(打ち直しの命令つき)で終える
2. サーバーが止まらない(10 秒で running のまま)、または起動し直せないときは、座席を立てずに断り、打ち直しの命令を出す。サーバーがどちらの状態かは `herdr status server --json` の結果をそのまま出す
3. 最後の pane の確かめから `herdr server stop` までの窓(1 秒未満)は閉じずに残し、Known gaps と tech-debt に書く。Codex の計画助言の指摘 2 は「サーバー側で pane の作成を締め出すか、条件つきの停止を前提にし、できなければ自動の立て直しを断れ」だが、herdr 0.7.5 にはどちらもない(`api schema` の `server.stop` は `EmptyParams` だけ)。文字どおりに入れると機能がなくなる。窓で消えうるのは、ゲートを通った直後の生まれかけの座席(サーバーが落ちればその spawn は失敗し pane を閉じる、打ち直しで戻る)と、人が herdr を直接触って作った pane。ralph の pane を作る動詞に共有のロックを取らせる案(A)は、人の直接の操作を防げず、同じプロセスの中のロックの持ち替えなどの穴があり、4 つの動詞にまたがるので、設計メモごと tech-debt に回す(consult の判定)。`--restart-herdr` は人が頼んだときだけ打つ、と skill に書く
4. `--restart-herdr` は active な座席の全部を対象にするときだけ受け、打ち直しの命令はサーバーを起動し直したかどうかで `--restart-herdr` を写すか外すかを変える(Codex の計画助言の指摘 1)

## Acceptance criteria

仕様の AC1〜AC8 に従う。

- [ ] AC1: その org の座席の pane だけがあるサーバー(fake の herdr)で `reseat --restart-herdr` が、座席を止め、`herdr server stop` を打ち、running でなくなったあとゲートで起動し、座席を立て、`reseated` を記録する(呼ぶ順をテストで固定)
- [ ] AC1b: `--seat` で active な座席の一部だけを選んで `--restart-herdr` を付けると、何もせずに断る
- [ ] AC2: ほかの pane が 1 つあれば、何も止めずに断り、その pane の id・label・cwd を出す(終了コード 1)。記録だけが戻った pane の見分け方も出す
- [ ] AC3: 呼んだ側に `HERDR_PANE_ID` があれば断る
- [ ] AC4: 座席を止めたあとにほかの pane ができていれば、サーバーを止めずに終わり、打ち直しの命令を出す
- [ ] AC4b: サーバーを起動し直したあと leader が立ち、ほかの座席が立たなかったとき、出した打ち直しの命令は `--restart-herdr` を持たず `--claude-config-dir` を持つ。その命令を打つと、サーバーを止めずに残りの座席だけを立てる(テストで命令を実際に打つ)。サーバーを止める前に断ったときの打ち直しは `--restart-herdr` を持つ
- [ ] AC5: `--claude-config-dir /x` でゲートが起動するサーバーの環境の `CLAUDE_CONFIG_DIR` が `/x`、`''` ならない。存在しないディレクトリは断る。打ち直しの命令にも値が写る
- [ ] AC6: サーバーの環境が読める(fake の lsof と ps)とき、reseat が `CLAUDE_CONFIG_DIR` を出し、移したい値と違えば知らせる。読めなければ理由を出し、断らない。pid が 2 つ以上なら herdr のものに絞る
- [ ] AC7: `--restart-herdr` なしの今の reseat のテストが変わらずに通る(サーバーの環境の 1 行の出力で変わる箇所は、意図した変更として報告する)
- [ ] AC8: 文書(help、director と org の skill の 4 写し、README、reseat の仕様、AGENTS.md)が新しい flag と確かめを書く。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S1 ドライバとサーバーの環境: 全部の pane の一覧と `server stop` の呼び出し、ゲートの環境の置き換え(`--claude-config-dir`)、動いているサーバーの `CLAUDE_CONFIG_DIR` の読み(AC5 の環境の部分、AC6 の読みの部分) — verify: `test -z "$(gofmt -l internal/org/driver internal/cli)" && go vet ./internal/org/driver/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'HerdrServer'`
2. S2 reseat の立て直し: `ReseatParams` の欄、2 回の確かめ、止めて起動する順、失敗の扱い、打ち直しの命令、CLI の flag と出力(AC1〜AC7) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Reseat' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Reseat|HerdrServer'`
3. S3 文書(AC8) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: 仕様の FR-1〜FR-8、NFR-1〜NFR-3、AC1〜AC8
- Documentation drift to check: help、skill、README、reseat の仕様の、別のアカウントへ移す手順の文が新しい flag と一致すること
- Evidence to capture: 実機の確かめ(下の Test plan)の記録を `docs/evidence/` に置く

## Test plan

- Unit tests: ドライバの 2 つの呼び出し(fake の runner)、環境の置き換えと外し、環境の読み(lsof と ps の出力の形、pid が複数、読めない、値に空白)
- Integration tests: reseat の立て直しの順と断る形(fake の herdr と fake のゲート)
- Regression tests: 今の reseat、ゲート、ドライバのテスト
- Edge cases: 2 回目の確かめで断る、止まらない、起動できない、`--claude-config-dir` だけ(`--restart-herdr` なし)で動いているサーバーに効かないことの表示、`--dry-run` で何も止めない
- 実機の確かめ: このマシンの herdr のサーバーには別の repo の org の pane があるので、本物のサーバーは止めない。分離した herdr のサーバー(`docs/recipes/isolated-herdr-server.md` の XDG の 3 変数)で、shell の pane が 1 つあるときに断ることと、止めて起動し直せることを確かめる(座席は立てない)。できなければ理由を書いて fake のテストに留める
- Evidence to capture: `go test -run 'Reseat|HerdrServer'` の要約、`./scripts/run-test.sh` の結果、実機の確かめの記録

## Risks and mitigations

- ほかの pane を消す: 生きた pane の一覧で 2 回確かめ、確かめられなければ止めない。最後の確かめと停止のあいだの窓(1 秒未満)は残る(Design decisions 3)。そこで消えうるのは、生まれかけの座席と人が直接作った pane。PR の Known gaps と tech-debt に、窓の長さ、消えうるもの、ralph の動詞に共有のロックを取らせる案の設計メモ(同じプロセスの中の持ち替え、ソケットの取れない herdr、書けないロックのディレクトリの穴)を書く
- 一部の座席だけの立て直しで残りの座席を殺す: active な座席の全部を求める(AC1b)。打ち直しで立ち直った座席を殺す: 打ち直しの命令を進み具合で変える(AC4b)
- 止めたあと起動できない: 座席を立てずに断り、打ち直しの命令を出す。打ち直しにも `--claude-config-dir` を写す
- `lsof` や `ps` が使えない環境: 読めないと出して断らない(見える化だけが欠ける)
- 新しいアカウントの資格で座席が動くことは、2 つのアカウントを持つ人の手でしか確かめられない: 仕様の Open questions に残し、PR の Known gaps に書く

## Rollout or rollback notes

- 新しい flag は付けたときだけ働く。付けない reseat は、サーバーの `CLAUDE_CONFIG_DIR` の 1 行(と、違うときの知らせ)のほかは変わらない
- 戻すときはこの PR を revert する
- release ノートの候補: 「`ralph org reseat --restart-herdr` と `--claude-config-dir` で、座席を別のアカウントへ移すときの herdr のサーバーの立て直しを 1 回で済ませる。ほかの pane があれば断る。reseat がサーバーの `CLAUDE_CONFIG_DIR` を出す」

## Open questions

- なし(仕様の Open questions を引き継ぐ)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件。[HIGH] 1: `--seat` で一部の座席だけを立て直すと残りの座席が死に、打ち直しに `--restart-herdr` を写すと立ち直った leader を殺す → 入れた(Scope、AC1b、AC4b)。[HIGH] 2: 最後の pane の確かめから停止までの窓で、別の repo の spawn や人の直接の操作が作った pane を消しうる、サーバー側の締め出しか条件つきの停止を前提にせよ → herdr 0.7.5 にどちらもないので文字どおりには入れず、窓を残して Known gaps と tech-debt に書く(Design decisions 3)。consult(consult-spec-herdr354 の続き、判定「方針を変える」: ralph の動詞に共有のロックを取らせる案ではなく、窓を残して記録する)に従った。git の状態の差は、助言の最中に orchestrator が計画の Visual review の節を書いたことによるもの
- [x] Plan approved — 2026-11-20、系列の指示(計画は推す案で自動承認)で承認した。人の目での承認はしていない。指摘 2 を文字どおりに入れなかったことは、PR の本文と最後の報告でユーザーに伝える
- [x] Branch created
- [x] Implementation started
  - S1(558227cf、implementer): 確認コマンドが exit 0(orchestrator も `-count=1` で打ち直して ok)。ドライバに `PaneListAll` と `ServerStop`(`HerdrClient` にも)、`HerdrPane` に `Cwd`。ゲートの環境の上書き `orgHerdrServerEnvOverride` と `wireOrgHerdrServerGateWithEnv`・`ensureOrgHerdrServerWithEnv`(今の関数は変えずに足した)。`checkOrgClaudeConfigDir`(絶対パスを返す、package 内)。サーバーの `CLAUDE_CONFIG_DIR` の読み `orgHerdrServerClaudeConfigDirReader`(set・unset・読めないの 3 つ。環境の項目が 1 つもなければ読めない、サーバーが動いていなければ読めない)。変異 17 個がすべて落ちる。`ServerStop` の成功時の応答は実機で取れていない(exit 0 で error envelope でなければ成功)。機械的セルフチェックは 5 項目とも clean か fixed(fixture の資格情報風の名前を改名)
  - S2(e00adaf5、implementer): 確認コマンドが exit 0(orchestrator も org を `-count=1` で打ち直して ok)。変異 15 個がすべて落ちる。ずれ: (1) サーバーの `CLAUDE_CONFIG_DIR` の行は本番では出すが、CLI のテストでは TestMain で読みを nil にして出さない(範囲外の `codex_hook_trust_notice_test.go` の 4 本が reseat の stderr 全体を固定しているため。行は個別の fake で確かめた)、(2) `HERDR_PANE_ID`、座席の一部だけの指定、サーバーが動いていない・読めないの確かめを `PlanReseat` の 9 にし、ゲートを 10 にした(ゲートが止まったサーバーを起動する前に断るため。仕様の「1〜9 のあと」とずれる)、(3) 打ち直しの `--restart-herdr` は「サーバーが止まったのを見たか」で決める(見ていなければ残す、見たら外す)、(4) `--claude-config-dir` は与えられれば打ち直しに必ず写す、(5) 既存の関数と help の文は変えずに足した(`reseatRetryCommandWithHerdr`)、(6) サーバーを止めたら座席が 1 つも止まっていなくても `reseated` を記録する。AC4 と AC4b の打ち直しの命令はテストで実際に打った。機械的セルフチェックは 5 項目とも clean か fixed
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
