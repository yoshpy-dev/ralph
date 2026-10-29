# local-config-org

- Status: Approved
- Approved: 2026-10-29 sha256:1609f175543b
- Owner: Claude Code
- Date: 2026-10-29
- Related request: #276 の 1 本目(A1): 個人用の設定ファイル `ralph.local.toml` を足し、`RALPH_ORG_AGMSG_HOME` での上書きをやめる
- Related issue: #276
- Type: feat
- Branch: feat/local-config-org

## Objective

#276 は、人が決める値を `ralph.toml` と git が無視する個人用のファイルだけで決め、環境変数での上書きをやめる issue。3 本に分け(D1)、この計画は 1 本目を受け持つ。

- 個人用の設定ファイル `ralph.local.toml` を足す。`ralph.toml` と同じ形で、キーごとに上書きする。git が無視し、linked worktree(標準フローの task worktree、機能ごとの org の worktree)からは主 worktree の root のものを読む。
- `RALPH_ORG_AGMSG_HOME` を読まなくする。agmsg の場所は `[org].agmsg_home`(`ralph.toml`、個人用のファイルで上書き)だけで決まる。
- テストと `ralph org gate` が開発機の個人用のファイルと本物の agmsg に触れないようにする。

PR は `Refs #276`。2 本目(A2)は `RALPH_ORG_STATE_DIR`、3 本目(B)は標準フローの値とスクリプトを受け持ち、B が `Closes #276` にする。

## Scope

- 新しい package `internal/gitroot`(S1)
  - `internal/org/statedir.go` の `gitMainWorktree`・`parseMainWorktreeRecord`・`gitToplevel`・`samePath` を移し、作業ディレクトリを引数で取れる形にする(`git -C <dir>`)。
  - org はこれを呼ぶ。org の動き(ledger の決まり方)は変えない。既存の `statedir_test.go` が守り。
- 個人用のファイルの読み込み(S2、`internal/config`)
  - `config.LocalPath(configPath)`: `filepath.Dir(configPath)` が git の作業ツリーの中なら、その repo の主 worktree の root の `ralph.local.toml`。主 worktree を名指せない(bare の linked worktree)ときはその作業ツリーの toplevel のもの。git がない、または git の外なら `filepath.Dir(configPath)` の `ralph.local.toml`(D2)。repo の root、linked worktree、サブディレクトリのどこから打っても同じファイルになる。
  - `config.Load(path)` は `ralph.toml` を読んだあとに個人用のファイルを同じ `Config` に重ねる(D3)。`ralph.toml` がなくても個人用のファイルは読む。個人用のファイルがなければ今と同じ。読めない・正しくない個人用のファイルはエラーにし、文にそのファイルのパスを入れる。
  - `RALPH_LOCAL_CONFIG` がちょうど `off` なら個人用のファイルを読まない(D4)。ほかの値と未設定は読む。
  - 個人用のファイルの `[org].agmsg_home` は、絶対パスか `~/` 始まりでなければ Load が断る(D5)。
  - `[org].driver_pool` と `[org].model_pool` の有無の判定(`orgPoolKeysPresent`)は、2 つのファイルのどちらかにあれば「ある」とする。
  - `ralph config get` は重ねた値を返す。help に個人用のファイルの 1 段落を足す。
- 無視と隔離(S3)
  - `.gitignore` の管理ブロック(ルートと `templates/base/.gitignore`、同じ中身)に `ralph.local.toml` を足す。先頭に `/` を付けず、どの階層に置いても無視する(D10)。`ralph upgrade` で配布先にも入る。
  - `ralph org gate` のスクリプトの環境(`internal/org/gate_run.go` の `gateScriptEnv`)は、受け継いだ `RALPH_LOCAL_CONFIG` を落として `RALPH_LOCAL_CONFIG=off` を足す。gate の `go test` と、B 以降でスクリプトが読む個人用の値が、開発機のファイルで変わらないようにする。
  - `internal/cli`、`internal/org`、`internal/config` の TestMain で `RALPH_LOCAL_CONFIG=off` を固定する(`internal/config` には TestMain を新しく作る)。cwd が repo の中のテストや、repo のファイルを読むテスト(`root_ralph_toml_test.go`、テンプレートを読むテスト)は、D2 の探し方では開発機の主 worktree の root のファイルに届くため。個人用のファイルを確かめるテストは `t.Setenv("RALPH_LOCAL_CONFIG", "")` で外し、一時の repo の中で確かめる。
  - built binary を走らせるシェルテスト(`tests/*.sh`)には、今のところ設定を読むものがない。B でスクリプトが `ralph config get` を呼ぶようになったら、B でそのテストに `RALPH_LOCAL_CONFIG=off` を渡す。
- agmsg のテスト用の差し替え口(S4a、D6)
  - `internal/org/driver` に package 変数 `AgmsgHomeOverride`(空なら無効)を置き、`ResolveAgmsgHome` が最初に見る。production のコードは設定しない。
  - `internal/cli` と `internal/org` の TestMain が存在しないパスに固定する。`setupOrgStubPATH`(`internal/cli/org_test.go`)はスタブに向け、`t.Cleanup` で戻す。
  - 隔離のために `RALPH_ORG_AGMSG_HOME` を `t.Setenv` しているテストを、この差し替え口に移す。env の効き方そのものを確かめるテストは S4b で書き直す。このスライスの間は env をまだ読むので、テストは通ったまま。
- `RALPH_ORG_AGMSG_HOME` の撤去(S4b)
  - `driver.ResolveAgmsgHome` は env を読まない(差し替え口、`[org].agmsg_home`、既定の順)。`driver.AgmsgHomeEnv` を消す。
  - detach した watch の子(`internal/cli/org_watch.go` の `orgWatchChild`)は env を作り直さない。`orgWatchChildAgmsgHome` の絶対化は消し、`--config` の `[org].agmsg_home` が相対なら、watch を起動する前に断る(`ralph org watch --all --detach` と `ralph org start`、D7)。director start の D13 と同じ規則・同じ文にする。
  - `ralph org director start`(`internal/cli/org_director_headless.go`)は env を読まず、herdr のときの警告も消す。launch_cmd の起動スクリプト(`internal/org/director_cmd.go`、`director_start.go`)は `RALPH_ORG_AGMSG_HOME` を export しない。`DirectorStartParams` と起動スクリプトの `AgmsgHome` を消す。
  - `ralph doctor` は、環境に `RALPH_ORG_AGMSG_HOME` があれば warn で「もう読まない。`ralph.local.toml` の `[org].agmsg_home` に書く」と出す(移行の手がかり、D8)。
  - `scripts/ralph-config.sh`(ルートと `templates/base/`)の `RALPH_ORG_AGMSG_HOME` の既定の行と説明、それを突き合わせる `internal/config/defaults_sync_test.go` の検査、`tests/test-ralph-config.sh` の記述を消す。
  - env の効き方を確かめていたテスト(`driver/agmsg_test.go`、`director_cmd_test.go`、`org_director_headless_test.go`、`org_watch_test.go`、`org_runtime_env_test.go`、`prompts_leader_inbox_test.go`、`spawn_test.go` など)を、新しい決まり方に合わせて書き直す。
  - コードの注釈(`config.go`、`prompts.go`、`spawn.go`、help の文)から env の記述を消す。
- 文書(S5)
  - README の設定の節: `ralph.local.toml` の置き場所、上書きの決まり、git が無視すること、`RALPH_ORG_AGMSG_HOME` を読まなくなったこと。
  - `templates/base/ralph.toml` の先頭と `agmsg_home` の注釈。
  - `/org` と `/director` skill(`.claude/skills/` と `templates/base/.claude/skills/` を直し、`./scripts/sync-skills.sh` で `.agents/` の写し)の agmsg の記述。
  - recipe(`docs/recipes/` と `templates/base/docs/recipes/`)の `director-launch-cmd.md` と `isolated-herdr-server.md` の agmsg の記述。
  - `AGENTS.md` の地図: `internal/gitroot/` の行、`internal/config/` の行に個人用のファイル。

## Non-goals

- `RALPH_ORG_STATE_DIR` と `[org].state_dir`、`ralph insights` と `ralph doctor` の `--state-dir`、`config.LocalOnlyKeys`(A2)。
- 標準フローの値(`RALPH_STANDARD_MAX_PIPELINE_CYCLES`、reviewer のモデルと effort)、スクリプトが読む個人用の値(plan-visual のブラウザ、base、secret scan の skip)、`scripts/ralph-config.sh` の削除、`.claude/rules/ralph/model-routing.md` の書き直し(B)。
- `ralph.toml` の相対の `agmsg_home` の意味(cwd から解く)を変えること。
- `ralph doctor` が個人用のファイルの中身や、git に載ってしまった `ralph.local.toml` を調べること。
- `RALPH_TERRAFORM_INIT_VALIDATE` など、#276 の表にない変数(B で扱う)。

## Assumptions

- go-toml/v2 の `Unmarshal` は、文書にあるキーだけを上書きする。配列と配列の表(`[[org.model_pool]]`)は、文書の最初の出現で作り直す(Default() の model_pool に重ねても重複しないことから推測。S2 のテストで確かめる)。
- git が無視するファイルは linked worktree には現れない。標準フローの skill は task worktree で、機能ごとの org の leader は機能の worktree で動く。
- `internal/cli/main_test.go` の TestMain は `orgWatchLaunch` を panic する関数に固定しているので、テストから本物の ralph の子は起動しない。差し替え口が子に届かないことは問題にならない。
- `internal/org` の TestMain は `statedir_test.go` にある。

## Affected areas

- 新規: `internal/gitroot/`(コードとテスト)
- `internal/org/`: `statedir.go`、`gate_run.go`、`driver/agmsg.go`、`director_cmd.go`、`director_start.go`、`prompts.go`、`spawn.go`(注釈)、テスト
- `internal/config/`: `config.go`(と新しい `local.go`)、`defaults_sync_test.go`、`root_ralph_toml_test.go`、テスト
- `internal/cli/`: `config.go`、`org_watch.go`、`org_director_headless.go`、`doctor.go`、`main_test.go`、`org_test.go` ほかテスト
- `.gitignore`、`templates/base/.gitignore`
- `scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`、`tests/test-ralph-config.sh`
- `README.md`、`AGENTS.md`、`templates/base/ralph.toml`、`/org` と `/director` skill の 4 つの写し、recipe 2 本の 2 つの写し

## Visual review

- ページ: `.harness/state/plan-visual/local-config-org.html`(かんたん版の図 2 つ: `overview`、`easy-isolation`。折りたたみの中に `detail-overview`、`detail-lookup`、`detail-agmsg`)
- lint: `./scripts/plan-visual.sh lint` が exit 0。
- self-check: 全体と 5 つの図を `shot` で撮り、PNG を読んだ。ファイルの地図で箱からはみ出していた文字を直して撮り直し、すべて崩れなし。計画助言を反映したあと、`detail-lookup` と `detail-overview` の文を D2・D10 に合わせて直し、lint(exit 0)と 2 つの撮り直しを済ませた。

## Design decisions

Critical forks: 人には聞かず、ユーザーの指示どおり consult(consult-plan-settingstoml、2 回)に聞いて推奨案を選んだ。

- D1 分け方: #276 を変数の群で 3 本に分ける。A1(この計画: 個人用のファイルの仕組みと agmsg_home)、A2(state_dir)、B(標準フロー)。A1 と A2 は `Refs #276`、B が `Closes #276`。前例は #216/#218 と 8a の 3 分割。A を 1 本にすると差分が 1,000 行前後になり、cross-review の直しが重い。A2 の `ResolveOrgStateDir` の返り値の変更(呼び出し約 15、テスト約 60)は agmsg と独立している(consult 2 回目)。
- D2 個人用のファイルの探し方: config のパスの dir から始める(cwd からは始めない、consult 1 回目)。その dir が git の作業ツリーの中なら、toplevel かどうかにかかわらず主 worktree の root のファイルにする。主 worktree に寄せるのは、git が無視するファイルが linked worktree にないため。サブディレクトリでも主 root に寄せるのは、Codex の計画助言 1(HIGH)の指摘による。`--config` なしでサブディレクトリから打つと `./ralph.toml` の dir はサブディレクトリになり、toplevel だけを見る規則では個人用のファイルを読まない。そうなると、今まで env がどこからでも効いていた agmsg の場所が既定に戻る。主 root に寄せると、台帳(git-main-worktree)と同じ規則になる。テストの cwd が repo の中のときは開発機のファイルに届くので、TestMain の `RALPH_LOCAL_CONFIG=off` で防ぐ(S3)。
- D3 重ね方: `ralph.toml` と同じ形の文書を、同じ `Config` にもう一度 `Unmarshal` する。表はキーごとに重なり、配列は置き換わる。検査は重ねたあとに 1 回。`ralph config get` も同じ値を返す。別の形(個人用のキーだけの struct)は、`ralph config get` で読めず B と形がそろわないので採らない。
- D4 `RALPH_LOCAL_CONFIG=off`: gate とテストの隔離に使う内部の受け渡し。`RALPH_WIP_CHECKPOINT=off` と同じ「ちょうど `off` だけ」の形。人の設定を上書きする値ではなく、個人用のファイルを読まないだけ。
- D5 個人用のファイルのパスの値は絶対か `~/`: 個人用のファイルは主 worktree の root にあるが、読むプロセスの cwd は linked worktree のこともある。相対を許すと、どこから解くかで別の agmsg を指す。今の env の相対値の絶対化(`org_watch.go` の `orgWatchChildAgmsgHome`)は env と一緒に消える(consult 1 回目)。
- D6 テストの agmsg の差し替え口は `driver` に置く: cli だけに置くと、`internal/org/prompts.go` の `ResolveAgmsgHome("")` と org のテストが外れる(consult 2 回目)。`testing.Testing()` のときだけ env を読む案は、production のコードが env を読む形が残るので採らない。
- D7 watch の起動での相対の `--config` の `agmsg_home`: 断る。env で絶対パスを子に渡す道がなくなるため。director start の D13(どこで動いても断る)と 1 つの規則にする。hidden flag で子に渡す案は、人も使えてしまうので採らない。
- D8 `ralph doctor` の移行の warn: env を読むのは「もう効かない」と知らせるためだけで、値には使わない。黙って効かなくなると、`RALPH_ORG_AGMSG_HOME` を使っていた人の座席が別の agmsg に向く。
- D9 `internal/gitroot`: config も「主 worktree の root はどこか」を求める。org の実装を写すと約 25 行が重複するので、package に移して両方が使う(consult 2 回目、案のまま)。
- D10 無視の規則は先頭の `/` なしの `ralph.local.toml`: D2 で読むのは repo の root のファイルだけだが、サブディレクトリや `--config` の隣に置いてしまったファイルもコミットされないようにする(Codex の計画助言 2、MEDIUM)。読まれない場所のファイルを無視しても害はない。

## Acceptance criteria

- [ ] AC1 `ralph.local.toml` が config のファイルの隣にあれば、`config.Load` はその値で `ralph.toml` の値をキーごとに上書きする(表はキーごと、配列は置き換え)。`ralph.toml` がなくても読む。`ralph config get` は上書きした値を返す。
- [ ] AC2 config のファイルの dir が git の作業ツリーの中なら、主 worktree の root の `ralph.local.toml` を読む。linked worktree やサブディレクトリに同名のファイルがあっても、それは読まない。差し替え口を外した状態で、主 worktree の root、linked worktree の root、サブディレクトリから打った `ralph config get org.agmsg_home` が同じ値を返す。
- [ ] AC3 読めない、または正しくない `ralph.local.toml` は Load のエラーになり、文にそのファイルのパスが入る。`ralph config get` は exit 3。
- [ ] AC4 `RALPH_LOCAL_CONFIG=off` のときは個人用のファイルを読まない。`ralph org gate` のスクリプトは、呼んだ側の値にかかわらず `RALPH_LOCAL_CONFIG=off` で走る。
- [ ] AC5 ルートと `templates/base/` の `.gitignore` の管理ブロックに `ralph.local.toml`(先頭の `/` なし)があり、2 つの管理ブロックは同じ。`git check-ignore` が `ralph.local.toml` と `config/ralph.local.toml` の両方で通る。
- [ ] AC6 `RALPH_ORG_AGMSG_HOME` を設定しても、`ralph org` の各 verb と `ralph doctor` が使う agmsg の場所は変わらない。agmsg の場所は `[org].agmsg_home`(個人用のファイルで上書き)、なければ既定。`internal/` と `cmd/` の production のコードで、この名前を読むのは doctor の warn だけ。
- [ ] AC7 個人用のファイルの相対の `[org].agmsg_home` は Load が断る。`--config` の相対の `agmsg_home` は、watch の起動(`--detach` と `ralph org start`)でも director start と同じ文で断る。
- [ ] AC8 launch_cmd の起動スクリプトは `RALPH_ORG_AGMSG_HOME` を export しない。director start は env についての警告を出さない。
- [ ] AC9 `ralph doctor` は、環境に `RALPH_ORG_AGMSG_HOME` があれば warn を出す。
- [ ] AC10 テストは本物の agmsg と開発機の個人用のファイルに触れない。`internal/cli` と `internal/org` の TestMain が `driver.AgmsgHomeOverride` を存在しないパスに固定する。`internal/cli`、`internal/org`、`internal/config` の TestMain が `RALPH_LOCAL_CONFIG` を off に固定する。
- [ ] AC11 org の ledger の決まり方は変わらない(`internal/gitroot` に移したあとも `statedir_test.go` が通る)。config と org は同じ helper を使う。
- [ ] AC12 README、`templates/base/ralph.toml`、`/org` と `/director` skill、recipe 2 本、`AGENTS.md` が新しい決まりを書き、`RALPH_ORG_AGMSG_HOME` を設定の手段として書いていない。
- [ ] AC13 `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、Go のテスト、`./scripts/run-verify.sh` が通る。

## Implementation outline

1. S1 `internal/gitroot` を作り、org の git の helper を移す(AC11) — verify: `test -z "$(gofmt -l internal/gitroot internal/org)" && go vet ./internal/gitroot/ ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/gitroot/ ./internal/org/`
2. S2 `config.LocalPath` と Load の重ね合わせ、`RALPH_LOCAL_CONFIG=off`、個人用のファイルの相対の `agmsg_home` の拒否、`ralph config get` の help(AC1〜AC4 の Load 側、AC7 の前半) — verify: `test -z "$(gofmt -l internal/config internal/cli)" && go vet ./internal/config/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Config'`
3. S3 `.gitignore` の管理ブロック、gate の env、cli・org・config の TestMain の `RALPH_LOCAL_CONFIG=off`(AC4 の gate 側、AC5、AC10 の半分) — verify: `./scripts/check-sync.sh && git check-ignore -q ralph.local.toml && git check-ignore -q config/ralph.local.toml && test -z "$(gofmt -l internal/org internal/cli internal/config)" && go vet ./internal/org/ ./internal/cli/ ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Gate' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Gate|Config'`
4. S4a `driver.AgmsgHomeOverride` と TestMain の固定、`setupOrgStubPATH` と隔離のテストの移行(AC10) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/... ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/... ./internal/cli/`
5. S4b `RALPH_ORG_AGMSG_HOME` の撤去(driver、watch の子、director start、起動スクリプト、doctor の warn、`ralph-config.sh` とその検査、テストの書き直し)(AC6〜AC9) — verify: `test -z "$(gofmt -l internal)" && go vet ./internal/... && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ ./internal/org/... ./internal/cli/ && bash tests/test-ralph-config.sh && ./scripts/check-sync.sh`
6. S5 文書(README、テンプレートの ralph.toml、skill 2 種、recipe 2 本、AGENTS.md)(AC12) — verify: `./scripts/sync-skills.sh && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && ! grep -rn 'RALPH_ORG_AGMSG_HOME' README.md templates/base/ralph.toml .claude/skills .agents/skills templates/base/.claude/skills templates/base/.agents/skills docs/recipes templates/base/docs/recipes`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck)。
- Spec compliance criteria to confirm: AC1〜AC13。特に AC6 は `grep -rn 'RALPH_ORG_AGMSG_HOME\|AgmsgHomeEnv' internal cmd` の production のコードの結果が doctor の warn だけであること。AC2 は linked worktree で実際に `ralph config get` を打った出力。
- Documentation drift to check: `RALPH_ORG_AGMSG_HOME` の記述が、README、skill、recipe、テンプレート、`AGENTS.md` に設定の手段として残っていないこと。`RALPH_ORG_STATE_DIR` の記述は A2 で直すので残ってよい。`docs/specs/` の過去の仕様は書き換えない。
- Evidence to capture: grep の結果、linked worktree での `ralph config get` の出力、`git check-ignore` の結果、`check-sync` と `check-skill-sync` の出力。

## Test plan

- Unit tests:
  - `internal/gitroot`: 主 worktree、linked worktree、git の外、bare の linked worktree(主 worktree なし)、symlink を挟んだパスの同一判定。
  - `config.LocalPath`: 主 worktree の root、主 worktree のサブディレクトリ、linked worktree の toplevel とそのサブディレクトリ(どれも主 root を返す)、bare の linked worktree(その toplevel)、git の外(dir のまま)。
  - `config.Load`: 個人用のファイルがキーを上書きする、表の別のキーは残る、配列と `[[org.model_pool]]` は置き換わる、`ralph.toml` がなくても読む、個人用のファイルが読めない(ディレクトリ)・TOML が壊れている・検査に落ちる値のときのエラーにパスが入る、`RALPH_LOCAL_CONFIG=off` で読まない、`OFF` や空では読む、個人用のファイルの相対の `agmsg_home` を断り `~/` と絶対は通す、`driver_pool` が個人用のファイルだけにあるときの pool の扱い。
  - `driver.ResolveAgmsgHome`: 差し替え口 > cfgValue > 既定、env は効かない、`~/` の展開。
  - gate の env: 受け継いだ `RALPH_LOCAL_CONFIG=<何か>` を落とし `off` を足す。
- Integration tests:
  - `ralph config get` を個人用のファイルのある一時 repo で打ち、上書きした値が出る。差し替え口を外し、主 worktree の root、linked worktree の root、サブディレクトリ(`./ralph.toml` がない)から打った `org.agmsg_home` が同じ値になる(Codex の計画助言 1 の回帰テスト)。
  - watch の起動(`--detach`、`org start`)で `--config` の相対の `agmsg_home` を断る。絶対と `~/` は通る。
  - launch_cmd の起動スクリプトに `RALPH_ORG_AGMSG_HOME` がない。
  - `ralph doctor` が env があるときだけ warn を出す。
- Regression tests: `internal/org`・`internal/cli`・`internal/config` の既存のテストすべて、`tests/test-ralph-config.sh`、`./scripts/run-test.sh`。
- Edge cases: 個人用のファイルが空、個人用のファイルが symlink、config のパスが相対(`ralph.toml`)、`--config` が別の dir を指す、linked worktree の toplevel を symlink 経由で指す。
- Evidence to capture: go test の出力(既知の不安定なテストの打ち直しは 1 回だけで、結果を記録する)、shell テストの出力。

## Risks and mitigations

- テストが本物の agmsg を叩く: env を読まなくすると、隔離していたテストが既定の `~/.agents/skills/agmsg` に向く。S4a で差し替え口に移し終えてから、S4b で env を消す。TestMain は存在しないパスに固定する。
- 開発機の `ralph.local.toml` がテストや gate の結果を変える: TestMain と gate で `RALPH_LOCAL_CONFIG=off`。探し方を cwd でなく config のパスから始める(D2)。
- `RALPH_ORG_AGMSG_HOME` を使っていた人の設定が黙って効かなくなる: doctor の warn(D8)と release のノートで知らせる。
- go-toml の重ね方が想定と違う(配列が足される等): S2 のテストで確かめ、違えば Load で個人用のファイルの配列を明示的に置き換える。
- Load が git を呼ぶようになる: `git -C <dir>` の失敗は「git の外」と扱い、dir のまま読む。`RALPH_LOCAL_CONFIG=off` のときは git を呼ばない。
- `ralph.toml` は cwd の `./ralph.toml`、個人用のファイルは主 root のものになり、サブディレクトリから打つと 2 つの出どころが違う: `ralph.toml` の読み方は今のまま(Non-goals)。個人用のファイルの値は今の env と同じく、repo のどこから打っても効く。README にこの違いを書く。
- ほかのセッションとの衝突: `.gitignore`、README、`AGENTS.md`、`/org` と `/director` skill を A(#268・#269・#270)と C(#267・#271)も触る。merge 前に `git merge origin/main` で取り込み、両方の変更を残す。
- 差分が大きい S4a: テストの機械的な置き換えが多い。置き換えの前後で、テストの数(`go test -list`)が変わらないことを確かめる。

## Rollout or rollback notes

- release のノート: `RALPH_ORG_AGMSG_HOME` は読まれなくなった。値は `ralph.local.toml`(git が無視する、`ralph.toml` の隣)の `[org].agmsg_home` に、絶対パスか `~/` 始まりで書く。`ralph upgrade` は `.gitignore` の管理ブロックに `/ralph.local.toml` を足す。`ralph doctor` は env が残っていれば warn を出す。`--config` の相対の `agmsg_home` は watch の起動でも断る。
- 戻すときは PR を revert する。個人用のファイルがなければ、動きは revert 前の `ralph.toml` だけのときと同じ。

## Open questions

- なし。A2 と B の範囲は D1 のとおり。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(`scratchpad/bin/ralph`(main 7522835c から build)で `standard.plan_advisory` = true を読んだ。PATH の ralph は v6.0.0 で `config get` がない。前後の git の状態は同じ)。指摘 2 件をどちらも反映した。
  1. [HIGH] `--config` なしでサブディレクトリから打つと、toplevel だけを見る規則では個人用のファイルを読まず、agmsg が既定に戻る → D2 を「dir が git の作業ツリーの中なら主 worktree の root」に変え、AC2 と回帰テストを足した。config の TestMain にも off を足した。
  2. [MEDIUM] `/ralph.local.toml` はルートしか無視しない → 先頭の `/` なしにした(D10)。AC5 に `config/ralph.local.toml` の `git check-ignore` を足した。
  - consult(consult-plan-settingstoml)の最後の要点「シェルテストにも off を渡す」は、A1 には設定を読むシェルテストがないので B で扱うと Scope に書いた。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest sha256:1609f175543b。分かれ目は consult(consult-plan-settingstoml、2 回)に聞いて推奨案を選び、理由は Design decisions に書いた。
- [x] Branch created — feat/local-config-org(worktree `.claude/worktrees/local-config-org`、state id `plan-local-config-org`)
- [x] Implementation started
  - S1 ab012b7d(impl-276a1-s1): `internal/gitroot`(MainWorktree・Toplevel・SamePath・ResolvedOrClean)。検証 rc=0(gitroot 4s、org 655s)。self-check は 5 項目とも clean。`resolvedOrClean` は範囲外の permissions.go・split.go・feature_test.go・feature.go も使うので、statedir.go に gitroot を呼ぶ 1 行の wrapper を残した(実装は gitroot の 1 つだけ)。
  - S5 で直す古い参照(S1 の報告): `internal/cli/status_merge_guarantee_test.go:336` の注釈の `gitMainWorktree`、`internal/org/seat.go:7` の package doc の「git (statedir.go, ...)」、`docs/tech-debt/README.md` の 170 行と 198 行の `samePath`。
  - S2 の handoff に入れる(consult が S1 の implementer に返した点): `-C` に渡す dir は `filepath.Abs` してから渡す。gitroot の package doc(config が ralph.local.toml を探すのに使う)と食い違わない形にする。
  - S2 9ed3793d と 08d1d949(impl-276a1-s2): `internal/config/local.go`(LocalPath、LocalConfigEnv、個人用のファイルの読み込みと相対の agmsg_home の拒否)、Load の重ね合わせ(検査は `validateLoaded` に抜き出し)、`ralph config get` の help。go-toml v2.3.0 は配列と `[[org.model_pool]]` を置き換え、`[org.roles]` はキーごとに重なる(テストで固定)。検証 rc=0、変異 4 つをテストが落とした。self-check は 5 項目とも clean。
  - 計画からのずれ: S2 だけでは cli の 2 本(`TestStatusCmd_MergeGuaranteeRunsNoGH`、`TestOrgMergeOrder_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit`)が、Load が個人用のファイルを探す `git -C <dir> rev-parse` を想定外の git として落ちた。cli の TestMain の `RALPH_LOCAL_CONFIG=off` を S3 から S2 に前倒しした(08d1d949)。S3 に残るのは org と config の TestMain、gate の env、`.gitignore`。
  - S5 の文書に足す点: 本番の `ralph status` と `ralph org merge-order` は個人用のファイルを探すために git を 2 回多く呼ぶ。linked worktree やサブディレクトリに置いた `ralph.local.toml` は読まれない(何も言わずに無視される)。`internal/cli/org.go` の 165・315・608 行付近の注釈「ファイルがなければ既定」に個人用のファイルを足す。`statedir.go` の `resolvedOrClean` の wrapper(呼び出し 7 か所を `gitroot.ResolvedOrClean` に寄せれば消せる)を tech-debt に 1 行。
  - S3 08926b11(impl-276a1-s3): ルートと `templates/base/` の `.gitignore` の管理ブロックに `ralph.local.toml`(先頭の `/` なし)、gate の env は受け継いだ `RALPH_LOCAL_CONFIG` を落として `=off` を足す(`gateLocalConfigEnv`)、org と config の TestMain に off。検証 rc=0、self-check は 5 項目とも clean。
  - S5 の文書に足す点(S3 から): `internal/org/gate.go` の GateParams.Env の doc と `internal/cli/org_gate.go` の help に `RALPH_LOCAL_CONFIG=off` がない(help の文は `internal/cli/org_gate_test.go:806` が固定している)。
  - S4a 35e612f9(impl-276a1-s4a が利用上限で止まり、impl-276a1-s4a-r が残った変更を引き継いで再開): `driver.AgmsgHomeOverride`(上書き → env → cfgValue → 既定)、cli と org の TestMain が存在しない home に固定、`setupOrgStubPATH` は `setAgmsgHomeOverride` でスタブに向ける。env の振る舞いを確かめるテストは上書きを空にして残した(S4b で書き直す)。`func Test` の数は 2173 で HEAD 08926b11 と同じ、増えたのは driver のサブテスト 2 つ。self-check は 5 項目とも clean。
  - S4a の検証: handoff のコマンドは rc=0(テストはキャッシュから)。`-count=1` で 2 回回した(load 11〜15、ほかのセッションと並走)。1 回目は org が 30 分の timeout と既知の `TestStopDirectorProcess_…NotEndedInTime`、cli が `TestOrgStart_AStoppingOrOnceWatch_IsRefusedAndNothingLaunched/--plan_stopping`(1 秒以内の条件に 1.43 秒)。2 回目は cli と driver と protocol が ok、org が既知の SIGTERM のテストと `TestRegate_Interrupted/while_a_script_runs`(スクリプトの開始の印が 10 秒以内に出ない)。どれも agmsg の解決を通らない。オーケストレーターが単独で回し直し、`TestRegate_Interrupted` は ok(55s、load 16)、`TestOrgStart_AStoppingOrOnceWatch_…` は ok(5s)。ログは scratchpad の `impl-s4a/`。
  - S4b に回すもの: env を直接読む 3 か所(`internal/cli/org_watch.go:443`、`internal/cli/org_director_headless.go:349` と `:409`)と、env の振る舞いを確かめるテスト 10 ファイル(S4a の報告の分類表)。
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
