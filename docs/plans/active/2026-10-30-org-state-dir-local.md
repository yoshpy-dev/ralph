# org-state-dir-local

- Status: Approved
- Approved: 2026-10-30 sha256:1ea27c42968f
- Owner: Claude Code
- Date: 2026-10-30
- Related request: #276 の 2 本目(A2)。`RALPH_ORG_STATE_DIR` を `--state-dir` の flag と個人用のファイルに移す
- Related issue: 276
- Type: feat
- Branch: feat/org-state-dir-local

## Objective

#276 は、人が決める値を `ralph.toml` と git が無視する個人用のファイル `ralph.local.toml` だけで決め、環境変数での上書きをやめる issue です。3 本に分けました(docs/plans/archive/2026-10-29-local-config-org.md の D1)。1 本目の A1(PR #287)は個人用のファイルの仕組みを足し、`RALPH_ORG_AGMSG_HOME` を撤去しました。この計画は 2 本目で、issue の表の「`RALPH_ORG_STATE_DIR` → `--state-dir` の flag と個人用のファイル」を受け持ちます。

- org の台帳(`.harness/state/org`、ledger)の場所を、`--state-dir` と個人用のファイルの `[org].state_dir` で決める。ralph は `RALPH_ORG_STATE_DIR` を読まない。
- `ralph insights` と `ralph doctor` に `--state-dir` を足す。いまは env でしか台帳を変えられない。
- `[org].state_dir` は個人用のファイルだけに書けるキーにする(`config.LocalOnlyKeys`)。

PR は `Refs #276` です。3 本目(B)が標準フローの値とスクリプトを受け持ち、`Closes #276` にします。

## Scope

- **config**(`internal/config/`)
  - `OrgConfig.StateDir`(`[org].state_dir`)を足す。
  - `LocalOnlyKeys` を足す。`ralph.toml` がそのキーを持てば Load はエラーにし、ファイルのパスを名指して「`ralph.local.toml` に書く」と言う。いまは `org.state_dir` だけ。`ralph.toml` の文書だけを probe して見る(`checkLocalAgmsgHome` と同じ形)。
  - `LocalOrgStateDir(dir)` を足す。`dir` の repository の個人用のファイル(`LocalPath`)だけを読み、`[org].state_dir` を返す。`~/` は展開し、空なら空を返す。絶対パスか `~/` で始まる形でなければ、ファイルのパスを名指すエラーにする(A1 の D5 と同じ形)。`ralph.toml` は読まない。`RALPH_LOCAL_CONFIG=off` なら読まない。
  - Load は `state_dir` の値を検査しない(D2)。相対の値を断るのは台帳を決める側だけなので、`--state-dir` を渡せば値の誤りを避けて動ける。
  - `ralph config get org.state_dir` は個人用のファイルの値を、書いたとおりに返す。
- **台帳の決め方**(`internal/org/statedir.go`)
  - `ResolveOrgStateDir` を `(dir, source string, err error)` を返す形に変える。
  - 決め方は flag → 個人用のファイル(source `config`)→ git の主 worktree → git の toplevel → cwd の順。env の段は外す。
  - 個人用のファイルは、ralph を打った場所(cwd)の repository の主 worktree の root のもの。`--config` には関係しない(D1)。
  - source `config` は、これまでの `env` と同じに扱う。対象は `LedgerMainWorktreeRoot`、`FeatureRepoRoot`、エラーの中で次に打つコマンドの `--state-dir` の hint(flag のときだけ付ける)。
- **cli**(`internal/cli/`)
  - 16 か所の呼び出しを、1 つの helper(`resolveOrgLedger(cmd, explicit, access)`)に寄せる。helper は台帳を決め、続けて今ある `guardLegacyOrgStateDir` を同じ `access`(`orgLedgerMutating` か `orgLedgerReadOnly`)で呼ぶ。
  - `RALPH_ORG_STATE_DIR` が空でない値で設定されているときの helper の扱い(D5):
    - 台帳が `--state-dir` か `[org].state_dir` で決まったとき、または既定の段で決まった台帳が env の値と同じ場所のとき: stderr に 1 行の注意だけを出す。注意は「もう読まない。`[org].state_dir` に書くか `--state-dir` を渡し、env を消す」。
    - 台帳が既定の段で決まり、env の値と違う場所のとき: 台帳を変えるコマンド(`orgLedgerMutating`)は、何も書かず、見張りも起こさず、driver も呼ばずに終了コード 1 で断る。エラーは env の値と既定の台帳の両方を名指す。読むだけのコマンドは注意を出して続ける。
  - `ralph insights --state-dir` を足す。`--receipts` があればそちらが勝つ。
  - `ralph doctor --state-dir` を足し、merge guarantee のキャッシュの置き場所に使う。
  - doctor は、台帳の場所が決められなければ(個人用のファイルが読めない、正しくない)、fail の検査 1 行で報告して残りの検査を続ける。merge guarantee のキャッシュの書き込みだけを飛ばす。
  - doctor の warn は、A1 の `RALPH_ORG_AGMSG_HOME` の 1 つから、撤去した変数の表(`RALPH_ORG_AGMSG_HOME`、`RALPH_ORG_STATE_DIR`)に広げる。B が変数を足す。
  - `org_legacy_ledger.go` の古い台帳の案内は、insights にも `--state-dir` ができるので、どのコマンドにも `--state-dir` を言う。
- **テスト**
  - `t.Setenv(org.EnvOrgStateDir, …)` の約 25 か所(cli のテスト 14 ファイル、`statedir_test.go`)を書き換える。台帳を指すだけのものは `--state-dir` に、env の段の振る舞いを確かめていたものは、個人用のファイルを置いて `RALPH_LOCAL_CONFIG` を空にするテストに替える。
  - env を設定しても台帳が変わらないことと、注意が出ることを確かめる。
- **文書と help**
  - `README.md`、`/org` skill(4 つの写し)、`docs/recipes/worktrees.md` と `docs/recipes/codex-seat-permissions.md`(2 つの写し)、`templates/base/ralph.toml` の `max_orgs` の注釈。
  - leader の雛形 `internal/org/prompts/leader.md:126` と `internal/org/feature.go:391` の注釈。
  - help 文: `internal/cli/org.go`(57・707〜710 行付近)、`status.go`(121・140)、`org_inbox.go`(64)、`doctor.go`(75)、`insights.go`(30・59)、ほかに grep で出るもの。
  - help や雛形の文を固定するテストがあれば合わせる。

## Non-goals

- 標準フローの値、スクリプトが読む個人用の値、`scripts/ralph-config.sh` の削除、`.claude/rules/ralph/model-routing.md` の書き直し(B)。
- `[org].agmsg_home` の扱い(A1 で決めた。`ralph.toml` にも書ける)。
- `--state-dir` の相対パスの意味(cwd から解く)と、git の段の決め方を変えること。
- 台帳の中身や形式を変えること。古い linked worktree の台帳を移すこと。
- `docs/specs/` の過去の仕様の書き換え(新しい箇条を足すのは /sync-docs の判断)。

## Assumptions

- 子のプロセスには台帳を明示で渡している。`ralph org watch --detach` の子には `--state-dir`(`internal/cli/org_watch.go:358`)、director には prompt と check-in に `--state-dir`(`internal/org/director_cmd.go:323`)、機能の org の leader にはタスクの `- 台帳:` の行(`internal/org/feature.go:417`)。だから、個人用のファイルの読み違いは子に伝わらない。
- cli と org と config の TestMain は `RALPH_LOCAL_CONFIG=off` に固定している(A1)。そのため、`--state-dir` を渡さないテストも開発機の個人用のファイルを読まない。
- 新しいキーなので、既存の `ralph.toml` に `[org].state_dir` はない。LocalOnlyKeys のエラーで既存の project が落ちることはない。
- `ralph status` の一部の経路は、読めない `ralph.toml` でも動く(#274)。台帳の場所は個人用のファイルだけで決めるので、`ralph.toml` が壊れていても台帳は決まる。

## Affected areas

- `internal/config/`(`config.go`、`local.go` とテスト)
- `internal/org/statedir.go`、`internal/org/statedir_test.go`、`internal/org/feature.go`(注釈)、`internal/org/prompts/leader.md`
- `internal/cli/`(`org.go`、`org_watch.go`、`org_inbox.go`、`org_merge_order.go`、`org_merge.go`、`org_director.go`、`org_wait_director.go`、`status.go`、`insights.go`、`doctor.go`、`doctor_merge_guarantee.go`、`org_legacy_ledger.go` とテスト)
- `README.md`、`.claude/skills/org/SKILL.md`(と 3 つの写し)、`docs/recipes/worktrees.md`、`docs/recipes/codex-seat-permissions.md`(と `templates/base/` の写し)、`templates/base/ralph.toml`

## Visual review

- ページ: `.harness/state/plan-visual/org-state-dir-local.html`。かんたん版の図は `overview` と `easy-broken` の 2 つ。折りたたみの中は `detail-overview`(ファイルの地図)と `detail-order`(台帳の決め方の順、今とこれから)。
- lint: `./scripts/plan-visual.sh lint` が exit 0。
- self-check: 全体と 4 つの図を `shot` で撮り、PNG を読んだ。`easy-broken` の ✓ が箱の上に浮いていたので、線の上に移して撮り直した。ほかは崩れなし。計画助言を反映したあと、`overview` の ✕ の文、「承認の前に」、折りたたみの要約、`detail-overview` の helper の名前を直し、lint(exit 0)と `overview`・`detail-overview`・全体の撮り直しを済ませた。

## Design decisions

分かれ目は consult(consult-plan-statedir276a2)に聞き、推奨案を選んだ(ユーザーの指示)。

- D1 どの個人用のファイルの `state_dir` を読むか: ralph を打った場所(cwd)の repository の主 worktree の root のもの。`--config` には関係しない。env の代わりなので「この機械のこの repository の値」であり、git の主 worktree の段(`gitroot.MainWorktree`)と同じ repository を見る。台帳が決まってから主 worktree の `ralph.toml` を読む処理(`withMainWorktreeOrgLimits` など)があるので、`--config` に依存させると順が循環しうる。退けた案: `--config` があればその個人用のファイル。
- D2 個人用のファイルの相対の `state_dir`: 断る(絶対パスか `~/`)。A1 の D5 と同じ。退けた案: 主 worktree の root から解く、cwd から解く(`--state-dir` と同じ)。どちらも相対パスの意味が 3 つ目になる。断るのは台帳を決める側(`LocalOrgStateDir`)だけで、Load は値を検査しない。Codex の計画助言 2(MEDIUM)を受けた形で、`org stop` など Load を通るコマンドも、`--state-dir` を渡せば値の誤りを避けて動く。TOML として読めない個人用のファイルは A1 から Load がエラーにしており、直す道はファイルを直すこと(エラーがパスを名指す)。
- D3 `ralph.toml` の `[org].state_dir`: `config.LocalOnlyKeys` で Load がエラーにする。commit された値は全員の台帳を 1 か所に向けてしまい、黙って無視すると誤りが隠れる。`templates/base/ralph.toml` には注釈でだけ書く(有効な行を書くと init したすべての project が落ちる)。退けた案: 無視して warn、許す。
- D4 個人用のファイルが読めない・正しくないとき: `ResolveOrgStateDir` が error を返し、`--state-dir` のないコマンドは終了コード 1 にする。黙って次の段に落ちると、別の台帳を使ったまま気づけない。読むのは個人用のファイルだけで、`ralph.toml` は読まない。doctor だけは fail の検査 1 行にして続ける(壊れを診る道具が先に止まらないように。consult の指摘)。
- D5 `RALPH_ORG_STATE_DIR` がまだ設定されているとき: 値としては読まない。台帳が `--state-dir` か `[org].state_dir` で決まったとき、または既定の台帳が env の値と同じ場所のときは、stderr に 1 行の注意だけを出す。既定の台帳が env の値と違う場所のときは、台帳を変えるコマンドを断り(終了コード 1、何も書かない)、読むだけのコマンドは注意を出す。doctor も warn を出す。
  - 初めは「注意だけ」を選んでいた(consult)。Codex の計画助言 1(HIGH)を受けて、台帳を変えるコマンドを断る形に変えた。start と spawn は見張りを自動で起こす(`wireOrgStartWatchGate`、`internal/cli/org.go:479`)ので、watch gate では止まらず、更新した人の新しい作業が既定の台帳で始まる。元の台帳の org はそのまま動くので、上限と予約が 2 つの台帳で別々に数えられてしまう。
  - 退けた案: doctor だけ(A1 の D8 と同じ)、すべてのコマンドを断る(`ralph status` で状況を見ることもできなくなる)。
- D6 呼び出しの寄せ方: cli の 16 か所を `resolveOrgLedger(cmd, explicit, access)` の 1 つに寄せる。台帳を決めること、`cmd.Flags().Changed("state-dir")`、古い台帳の検査(`guardLegacyOrgStateDir`)、env の扱い(D5)を 1 か所で扱う。どの呼び出しも、台帳を決めた直後に同じ `access` で `guardLegacyOrgStateDir` を呼んでいるので、その組を helper にする。
- D7 スライスの切り方: 返り値の変更と env の撤去を 2 つのスライスに分ける。S2 で error と config の段を足し、env は残す(既存のテストはそのまま通る)。S4 で env を外してテストを書き換える。どちらのコミットもビルドとテストが通る。

## Acceptance criteria

- [ ] AC1 `--state-dir` を渡さないとき、個人用のファイルの `[org].state_dir`(絶対パスか `~/`)が台帳の場所になる。対象は `ralph org` の動詞、`ralph status`、`ralph insights`、`ralph doctor` で、source は `config`。`--state-dir` を渡せばそちらが勝つ。
- [ ] AC2 個人用のファイルは、cwd の repository の主 worktree の root のもの。linked worktree からもサブディレクトリからも同じ値になり、`--config` に別のファイルを渡しても変わらない。
- [ ] AC3 個人用のファイルの相対の `state_dir` は、台帳を決めるときにエラーになり、パスを名指す。`~/` は home に展開する。Load はこの値を検査しないので、`--state-dir` を渡したコマンド(Load を通る `ralph org stop` など)は値の誤りに妨げられずに動く(CLI の経路のテストで確かめる)。
- [ ] AC4 `ralph.toml` に `[org].state_dir` があると Load がエラーにし、ファイルのパスと `ralph.local.toml` に書くことを言う。`templates/base/ralph.toml` は注釈でだけ触れ、init した project の Load は通る。
- [ ] AC5 個人用のファイルが読めない・正しくないとき、`--state-dir` のない org の動詞、status、insights は終了コード 1 でパスを名指す。`ralph.toml` が壊れていても台帳の場所は決まる。doctor は fail の検査 1 行を出して残りを続け、merge guarantee のキャッシュは書かない。TOML として読めない個人用のファイルは、`--state-dir` を渡しても Load がエラーにする(A1 から。エラーがパスを名指す)。
- [ ] AC6 本番のコードは `RALPH_ORG_STATE_DIR` を値として読まない。env を設定しても台帳は変わらない。空でない値で設定されているときは次のとおり(D5)。
  - 台帳が `--state-dir` か `[org].state_dir` で決まったとき、または既定の台帳が env の値と同じ場所のときは、stderr に 1 行の注意を出す。
  - 既定の台帳が env の値と違う場所のときは、台帳を変えるコマンドは終了コード 1 で断り、台帳に何も書かず、見張りを起こさず、driver を呼ばない。読むだけのコマンドは注意を出して続ける。
  - doctor は warn を出す。
- [ ] AC7 `ralph insights --state-dir <dir>` と `ralph doctor --state-dir <dir>` がその台帳を使う。insights は `--receipts` があればそちらが勝つ。古い台帳の案内(`org_legacy_ledger.go`)は、どのコマンドにも `--state-dir` を言う。
- [ ] AC8 source `config` は、これまでの `env` と同じに扱う。対象は `LedgerMainWorktreeRoot`、`FeatureRepoRoot`、`loadMainWorktreeOrgConfig` の判定、エラーの hint(`--state-dir` を付けない)。watch の子、director、leader には、これまでどおり `--state-dir` を明示で渡す。
- [ ] AC9 `RALPH_LOCAL_CONFIG=off` のとき、個人用のファイルの段は飛ばす(gate、regate、テスト)。
- [ ] AC10 文書と help: README、`/org` skill の 4 つの写し、recipe 2 本の 2 つの写し、`templates/base/ralph.toml`、leader の雛形、help 文が、台帳の場所を `--state-dir` と `[org].state_dir` で説明する。`RALPH_ORG_STATE_DIR` に触れるのは「もう読まない」の説明だけ。
- [ ] AC11 `go build ./...`、gofmt、`go vet`、golangci-lint、`check-sync.sh`、`check-skill-sync.sh`、`./scripts/run-verify.sh` が通る。

## Implementation outline

1. S1 config: `OrgConfig.StateDir`、`LocalOnlyKeys` と `ralph.toml` の probe、`LocalOrgStateDir(dir)`(相対の `state_dir` の拒否はここだけ)、テスト(`config get org.state_dir` を含む) — verify: `test -z "$(gofmt -l internal/config internal/cli)" && go vet ./internal/config/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ConfigGet|TemplateRalphToml|LoadTemplate'`
2. S2 台帳の決め方: `ResolveOrgStateDir` が error を返し、flag → env(S4 まで残す)→ config → git の順にする。source `config` の扱い(AC8)、cli の helper `resolveOrgLedger`(台帳を決めて `guardLegacyOrgStateDir` を呼ぶ)と 16 か所の呼び出し、doctor の fail の検査(AC5)、`--state-dir` を渡したコマンドが値の誤りに妨げられないことの CLI のテスト(AC3)、org と cli のテスト — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/`
3. S3 insights と doctor の `--state-dir`、古い台帳の案内、テスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Insights|Doctor|Legacy|MergeGuarantee'`
4. S4 env の撤去: `ResolveOrgStateDir` から env の段を外す。helper の env の扱い(D5: 注意、または台帳を変えるコマンドの拒否)と、拒否で台帳に何も書かず、見張りを起こさず、driver を呼ばないことのテスト(start、spawn、send)。doctor の撤去した変数の表、テストの env の書き換え(約 25 か所) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ && ! grep -rn 'os.Getenv(EnvOrgStateDir)\|Getenv("RALPH_ORG_STATE_DIR")' internal --include='*.go'`
5. S5 文書と help: README、`/org` skill(`.claude` と `templates/base/.claude` を直して sync-skills で写しを作る)、recipe 2 本(2 つの写し)、`templates/base/ralph.toml`、leader の雛形と `feature.go` の注釈、help 文と、それを固定するテスト — verify: `./scripts/sync-skills.sh && CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && test -z "$(gofmt -l internal/org internal/cli)" && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Leader|Prompt|Render|SkillSync' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Help|Status|Inbox|Insights|Doctor|TemplateRalphToml'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、secret scan)。
- Spec compliance criteria to confirm: AC1〜AC11。特に、本番のコードに `RALPH_ORG_STATE_DIR` を値として読む箇所がないこと(grep)、source `config` が `env` と同じ扱いを受けること(`statedir.go` の switch)、16 か所の呼び出しがすべて helper を通ること。
- Documentation drift to check: `RALPH_ORG_STATE_DIR` の記述が、README、skill、recipe、テンプレート、leader の雛形、help 文に設定の手段として残っていないこと(「もう読まない」の説明だけ)。`docs/specs/` の過去の仕様は書き換えない。
- Evidence to capture: grep の結果、`run-static-verify.sh` の出力、計画の digest。

## Test plan

- Unit tests
  - config: `[org].state_dir` の読み込み(絶対、`~/`、空)、相対の拒否、`ralph.toml` の LocalOnlyKeys のエラー、`LocalOrgStateDir` と `RALPH_LOCAL_CONFIG=off`。
  - org: `ResolveOrgStateDir` の各段と順、error、`LedgerMainWorktreeRoot`・`FeatureRepoRoot` の source `config`。
- Integration tests(cli)
  - 個人用のファイルの `state_dir` で `ralph org status`、`ralph status`、`ralph insights` が同じ台帳を読む。
  - linked worktree とサブディレクトリから打っても同じになる。`--config` に別のファイルを渡しても変わらない。
  - `ralph doctor` の fail の検査と続行、`--state-dir` の flag、env の注意と doctor の warn。
  - 逃げ道(AC3、AC5): 相対の `state_dir` と、TOML として読めない個人用のファイルの 2 つの場合で、`--state-dir` を付けた `ralph org stop` と `ralph org status` を CLI の経路で打つ。前者は動き、後者はパスを名指すエラーになる。
  - env の拒否(AC6): env が既定の台帳と違う場所を指すとき、`ralph org start`、`ralph org spawn`、`ralph org send`(TASK)は終了コード 1 になる。台帳の manifest が増えないこと、見張りの起動の差し替え口が呼ばれないこと、driver の stub が呼ばれないことを確かめる。`ralph status` と `ralph org status` は注意を出して続ける。`--state-dir` を渡すか、env が既定の台帳と同じ場所なら、注意だけで動く。
- Regression tests
  - env を設定しても台帳が変わらない。
  - `--state-dir` が個人用のファイルに勝つ。
  - hint の `--state-dir`、古い台帳の案内。
  - 既存の org と cli のテスト。
- Edge cases
  - 個人用のファイルがない、空の `state_dir`、TOML として読めない個人用のファイル、`~` だけ、`~user/…`。
  - bare repository の linked worktree(主 worktree がない)、git の外。
- Evidence to capture: `./scripts/run-test.sh` の出力。負荷で落ちたテストの名前と load average と打ち直しの結果も残す。

## Risks and mitigations

- `RALPH_ORG_STATE_DIR` に頼っていた人は、更新のあと別の台帳(既定)を見る。元の org が見えなくなり、新しい作業が既定の台帳で始まると上限と予約が 2 つの台帳で別々に数えられる(start と spawn は見張りを自動で起こすので、watch gate では止まらない)。→ D5: 台帳を変えるコマンドは断り、読むだけのコマンドは注意を出す。doctor の warn、release のノート。
- 個人用のファイルの壊れで、すべての org の動詞が止まる。→ エラーはパスを名指す。doctor は止まらずに報告する。`state_dir` の値の誤り(相対など)は `--state-dir` を渡せば避けられる(Load は値を検査しない)。TOML として読めないファイルは、直すしかない(A1 からの振る舞い)。
- テストの書き換え(約 25 か所)で、env の段を確かめていた意図が消える。→ 個人用のファイルの段のテストに置き換え、消したものは Progress に記録する。
- 呼び出し 16 か所の返り値の変更で、error を落とす箇所が出る。→ helper に寄せ、`errcheck`(golangci-lint)と S2 のテストで確かめる。
- 3 セッションの並行で、`README.md`、`/org` skill の写し、`docs/tech-debt/README.md` が衝突しうる。→ 取り込みは `git merge origin/main`、両方の変更を残す。

## Rollout or rollback notes

- release のノートに次を書く:
  - ralph は `RALPH_ORG_STATE_DIR` を読まない。台帳を変えていた人は、`ralph.local.toml` の `[org].state_dir`(絶対パスか `~/`)に書くか、`--state-dir` を渡し、env を消す。env が残っていて既定の台帳と違う場所を指していると、台帳を変えるコマンドは断られる。
  - `ralph insights` と `ralph doctor` に `--state-dir` ができた。
  - `ralph.toml` に `[org].state_dir` を書くとエラーになる。
- 戻すときは、この PR を revert する。台帳の中身と形式は変えない。ただし revert した ralph は `[org].state_dir` を読まない(A1 の Load は知らないキーを無視する。推測、未確認)。移行した人は、env を設定し直すか `--state-dir` を渡さないと、既定の台帳に戻る。

## Open questions

- なし。B の範囲は A1 の D1 のとおり。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(`scratchpad/bin/ralph`(main 7522835c から build)で `standard.plan_advisory` = true を読んだ。PATH の ralph は v6.0.0 で `config get` がない。gpt-6-astra、xhigh、read-only。前後の git の状態は同じ)。指摘 2 件を、ユーザーの指示(HIGH と MEDIUM は聞かずに反映)どおり反映した。
  1. [HIGH] env を撤去すると台帳が 2 つに割れうる。start と spawn は見張りを自動で起こすので、watch gate では止まらない → D5 を「注意だけ」から「既定の台帳が env と違えば、台帳を変えるコマンドを断る」に変えた。AC6、S4、Test plan(拒否で manifest に書かず、見張りを起こさず、driver を呼ばない)、Risks、Rollout(revert した ralph は `[org].state_dir` を読まない)を直した。
  2. [MEDIUM] `--state-dir` を渡しても、`newOrgRuntimeAt` の `config.Load` が同じ個人用のファイルを検査するので逃げ道にならない → 相対の `state_dir` を断るのは `LocalOrgStateDir`(台帳を決める側)だけにし、Load は値を検査しない(D2)。AC3・AC5 と Test plan に、CLI の経路で逃げ道を確かめるテスト(相対の値と、TOML として読めないファイル)を足した。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest sha256:1ea27c42968f。分かれ目は consult(consult-plan-statedir276a2)に聞いて推奨案を選び、理由は Design decisions に書いた。
- [x] Branch created — feat/org-state-dir-local(worktree `.claude/worktrees/org-state-dir-local`、state id `plan-org-state-dir-local`、base e3a3fbf7)
- [x] Implementation started
  - S1 e2e7b7f1(impl-276a2-s1): `OrgConfig.StateDir`、`config.LocalOnlyKeys()`(関数。ralph.toml の文書だけを map で読み、ドット区切りのキーをたどる。値が空でもキーがあればエラー、`RALPH_LOCAL_CONFIG=off` でもエラー)、`LocalOrgStateDir(dir) (value, path string, err error)`(`dir` が "" なら cwd、off なら ("", "", nil) で git なし、`~/` は `os.UserHomeDir()` で展開し失敗はエラー、絶対は Clean)。Load は値を検査しない(D2)。`config get org.state_dir` は reflect で引けるので配線なし、cli のテスト 2 本で固定。drift の検査は手書きの probe なので除外は要らず、`TestTemplateDriftProbesHoldNoLocalOnlyKey` を足した。検証 rc=0(18 秒、load 5〜7)、golangci-lint 0 件、変異 7 つはすべてテストが落とした。self-check は 5 項目とも clean。
  - S5 か /sync-docs に回すもの(S1 の報告): `.claude/rules/ralph/model-routing.md` の「Where the values live」の drift の説明に local-only のキーを書くか。`ralph config get --help` の exit 3 の説明は local-only のキーを名指していない(help を固定するテストあり)。
  - S2 daf8e512(impl-276a2-s2): `ResolveOrgStateDir` は `(dir, source, err)`、順は flag → env(S4 まで)→ config → git。config の段のエラーで下の段に進まない。helper `resolveOrgLedger(cmd, explicit, access)` は `org_legacy_ledger.go`、エラーは `<CommandPath>: cannot resolve the org ledger: <err>`。本番の呼び出しは 15 か所(計画の「16」は doc コメント 1 行を数えていた)、helper を通すのは 14、doctor は直接呼ぶ(guard なし、`cmd` なし。S3 で見直す)。spawn・start・start `--plan` の guard は `newOrgSpawnRuntimeAt` から helper に移した(1 回だけ走る)。doctor は台帳が決まらないとき `Org ledger` の fail 行を出し、Merge guarantee は走らせてキャッシュを書かない(`; not cached: …`)。新しいテスト: org 4 本(と既存のテストに config の行)、cli は `org_ledger_config_test.go` の 5 本(AC1・AC2 の 4 か所と `--config`、AC3 の回復は `org stop --all` と `org status` を `--state-dir` 付きで、AC5)。検証: gofmt・vet・build rc=0、org(1245 本)と cli(870 本)をそれぞれ 4 分割して前景で rc=0(load 92→6)、golangci-lint 0 件、変異 8 つはすべてテストが落とした。self-check は 5 項目とも clean。気づいた点: 個人用のファイルが有効で値がないとき、主 worktree の特定の git が 2 回走る。
  - S2 で escalate の振る舞いが後退した(implementer の報告): S2 より前は、個人用のファイルが TOML として読めないと `newOrgRuntimeAt` の Load で失敗して `EscalateNotRecorded` に流れ、banner・通知・`escalations.jsonl` が人に届いた。いまは `resolveOrgLedger` が先に失敗して終了コード 1 だけになる(相対の `state_dir` も同じ)。consult(consult-plan-escalate276a2)に聞き、案 B(直す)を選んだ。leader の雛形(`internal/org/prompts/leader.md:713-714`)は「`--state-dir` は機能ごとの org のもの。それ以外の org では省く」と書いており、`ralph org spawn` で作った普通の org の leader は escalate を `--state-dir` なしで打つ。直し方: `resolveOrgLedger` のエラーのうち、台帳を決める段(`ResolveOrgStateDir`)のものに `errors.Is` で判別できる印を付け、escalate だけはそのエラーで `EscalateNotRecorded` に流す。Inbox がないときは banner に台帳のパスの代わりにエラーを書き、jsonl は書かない。古い台帳の拒否は今までどおり素のエラー。D4(黙って別の台帳を使わない)とは矛盾しない: どの台帳にも書かず、人に知らせるだけ。スライス S2b として足す。S4 の D5 の拒否を escalate でどう扱うかは S4 で決める。
  - S4 に渡す env のテストの一覧(S2 の報告、28 か所): scratchpad の `s4-env-test-sites.md`(handoff に写す)。
  - S2b c78f6903(impl-276a2-s2b): `resolveOrgLedger` は台帳を決める段のエラーを sentinel `errOrgLedgerUnresolved` で包む(エラー文は変わらない)。古い台帳の拒否には付けない。escalate はその印のとき、台帳のない fallback(DesktopNotify だけ)で `EscalateNotRecorded` に流す。banner は `ORG ESCALATION (NOT RECORDED): org=<id> type=<T> -- the item is not in any inbox: <cause>`、どの台帳にも書かない。`validateEscalation` が先(断られた message は通知しない)。escalate の help に 1 文を足し、テストで固定。検証: `-run 'Escalat|Inbox'`(org)と `-run 'Escalat|Inbox|Ledger|Legacy'`(cli)が rc=0、go build・golangci-lint 0 件、変異 5 つはすべてテストが落とした。self-check: 誤字が fixed(doc コメント 2 か所)、ほかは clean。
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
