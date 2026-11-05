# org-role-models

- Status: Approved
- Approved: 2026-11-05 sha256:82b7b97155f4
- Owner: Claude Code
- Date: 2026-11-05
- Related request: 2026-11-05 のユーザーの依頼「ralphの設定ファイル(ralph.toml)でorg runtimeの各役割にモデルを割り当てるキーを新設したいです。ただし、席を立てるときの --model で上書き可能とします。これが実現すると、そもそもmodel_poolは不要になるでしょうか？」。/spec で決めた仕様は `docs/specs/2026-11-05-org-role-models.md`
- Related issue: N/A
- Type: feat
- Branch: docs/spec-org-role-models

## Objective

org runtime の役割ごとに driver とモデルの組を `ralph.toml` の `[org.role_models]` で決められるようにし、`ralph org spawn`・`ralph org start`・`ralph org director start` が `--driver` と `--model` を省いたときはその値で立てる。`[org].model_pool` と `[org.roles]` は廃止する。要件は spec の FR-1〜FR-9 と受け入れ条件のとおりで、この計画はその作り方と順を決める。

## Scope

- **設定(spec FR-1・FR-2・FR-5)**: `internal/config` に `[org.role_models]`(役割の名前 → `{ driver, model }`)を足す。組み込みの既定は claude の組と codex の組の 2 つで、実際に使われる `driver_pool` に `claude` があれば claude の組を使う。役割のエントリは、`ralph.toml` と `ralph.local.toml` のそれぞれで `driver` と `model` の両方のキーを要する。`driver` は `claude` か `codex`、`model` は `[reviewer].claude_model` と同じ文字の規則(`""` は CLI の既定)。エントリの `driver` が `driver_pool` にあるかは `config.Load` では見ず、どの設定の値を使うかを決めたあとで、解決の関数と doctor が見る(D9)。
- **解決(spec FR-3)**: `internal/org` に役割と 2 つのフラグから driver とモデルを決める関数を置き、spawn・start(`<task>` と `--plan` の両方)・director start が使う。spawn と director start の `--driver` は必須でなくなり、start の `--driver` の既定は leader の役割の driver になる。`--model` を省いたときの警告はなくす。
- **モデルが `""` のとき**: 座席と headless の director を `--model` なしで起動する(herdr の pane と `launch_cmd` の起動スクリプトの両方)。director の登録は空のモデルを受け付け、`director status` は CLI の既定と分かる表示にする。codex の座席の receipt は、指示したモデルがないので honored を `unknown` にし、観測したモデルは記録する。spawn の 8 秒の観測で見つからなかった座席も、`ralph org stop` のときの観測(`internal/org/verbs.go` の `observeStopModelReceipt`)で実効モデルを記録する。
- **読み先(spec FR-4)**: `applyMainWorktreeOrgLimits` と `withMainWorktreeDirectorStartConfig` が、main worktree の `ralph.toml` から `RoleModels` と `DriverPool` も取る。
- **廃止(spec FR-6)**: `model_pool` と `[org.roles]` を `OrgConfig` から外す。`config.Load` は、どのファイルがどちらのキーを書いているかを記録するだけで止まらない。`model_pool` が残っていれば spawn・start・director start が stderr に 1 行出す。`[org.roles]` が残っていれば 3 つの動詞は何も記録せずに断る。`[org.role_models]` の改名前の役割名(`lead`)は、今の `[org.roles]` と同じく spawn が断る。
- **doctor(spec FR-7)**: 「Org codex model slugs」と `--probe-models` を role_models に、codex writable root の検査を `driver_pool` に、envelope の行を role_models に変え、残ったキーの warn を足す。
- **init(spec FR-8)**: 片方の CLI だけの init から `model_pool` の処理を消す。
- **テンプレート**: `templates/base/ralph.toml` から `model_pool` の配列と `[org.roles]` のコメント例を消し、`[org.role_models]` の既定をコメントで示す。`defaults_sync_test.go` がコメントと `Default()` の一致を検査する。
- **雛形と文書(spec FR-9)**: leader と director の雛形、`/org` と `/director` skill、`model-routing.md`、README、recipes、`docs/tech-debt/README.md`、org-runtime spec の改訂注記。skill と rule は `.claude/`・`.agents/`・`templates/base/` の写しをそろえる。

## Non-goals

- `[org.watchdog].watcher_model` と `[reviewer]` の 4 キー。
- `ralph config get` で `[org.role_models]` を読むこと(表は今も exit 2)。
- `ralph insights` の集計と receipt の項目の追加。
- spawn の `--leader-driver` の既定(leader の識別子の agmsg の種類)。
- 座席のモデルにコストの上限をかける仕組み。
- 下流の `ralph.toml` を書き換える移行の道具(`ralph upgrade` は seed の advisory diff を出すだけのまま)。

## Assumptions

- go-toml/v2 v2.3.0 は、既存の map への 2 回目の `Unmarshal` を役割ごとにマージし、インライン表のエントリは丸ごと置き換え、表見出しのエントリはキーごとにマージする(2026-11-05 に一時モジュールで確かめた。spec の Research findings)。どのファイルのエントリにも両方のキーを要すれば、どちらの書き方でも結果は同じになる。
- `gpt-6.1-sol` は codex の `models_cache.json` にある(2026-11-05 に手元で確かめた)。
- `claude` と `codex` は、`--model` を渡さなければ自分の設定の既定のモデルで動く。
- 機能の org の leader は `--state-dir` で main worktree の台帳を指して spawn するので、`loadMainWorktreeOrgConfig` の条件に当たり、main worktree の role_models が使われる(`withMainWorktreeOrgLimits` のコメントにある今の挙動)。
- 各スライスのコミットはリポジトリ全体がビルドでき、そのスライスの検査が通る状態にする。古い型と関数は、使う側を移し終えたスライス(S5)でまとめて外す。

## Affected areas

- `internal/config/config.go`(`OrgConfig`、`Default()`、`Load`、`orgPoolKeysPresent`、`filterModelPoolByDrivers`)、`internal/config/local.go`、新しい `internal/config/role_models.go`(型、既定の 2 組、ファイルごとの検査、文字の規則、残ったキーの記録)
- `internal/config/config_test.go`、`local_test.go`、`defaults_sync_test.go`、`defaults_sync_settings_test.go`、新しい `internal/config/role_models_test.go`
- `internal/org/envelope.go`(`ValidateSpawnEnvelope`、`ModelAllowedForRole`、`modelInPool`)、`envelope_summary.go`(`EnvelopeSummary`、`DefaultModelForDriver`、`DefaultModelForDriverAndRole`)、新しい `internal/org/role_models.go`(解決の関数、`[org.roles]` の拒否)
- `internal/org/spawn.go`(argv の `--model`、receipt、`spawn_started` の event に CLI の既定の印、`retiredRoleConfigErr` の近くの拒否)、`verbs.go`(`observeStopModelReceipt` と `codexSpawnCorrelation`)、`prompts.go`(`RetiredRoleConfigKeys`)、`permissions.go`(コメント)
- `internal/org/director_start.go`、`director_cmd.go`(空のモデルの起動と登録の検査)、`director.go` か `org_director_report.go` の表示
- `internal/org` のテスト: `envelope_test.go`、`envelope_summary_test.go`、`spawn_test.go`、`prompts_test.go`、`receipts_test.go`、`feature_test.go`、`watch_test.go`、director のテスト
- `internal/cli/org.go`(spawn・start・start --plan のフラグと解決、`resolveModelOrWarn`、`applyMainWorktreeOrgLimits`、`withMainWorktreeDirectorStartConfig`)、`org_director_headless.go`、`org_director.go`
- `internal/cli/doctor.go`(`checkOrgEnvelope`、`checkOrgModelProbes`)、`doctor_codex_models.go`、`doctor_codex_writable_root.go`、`internal/org/driver/probe.go`(空のモデルの probe)
- `internal/cli/init_settings.go`
- `internal/cli` のテスト: `org_test.go`、`org_director_headless_test.go`、`org_director_test.go`、`doctor_org_test.go`、`doctor_codex_writable_root_test.go`、`doctor_load_error_test.go`、`init_settings_test.go`、`init_settings_write_test.go`、`config_test.go`
- `internal/org/prompts/leader.md`、`internal/org/prompts/director.md`
- `templates/base/ralph.toml`
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と `.agents/skills/`・`templates/base/.claude/skills/`・`templates/base/.agents/skills/` の写し
- `.claude/rules/ralph/model-routing.md` と `templates/base/.claude/rules/ralph/model-routing.md`
- `README.md`、`docs/recipes/codex-seat-permissions.md`、`docs/recipes/director-launch-cmd.md` と `templates/base/docs/recipes/` の写し、`docs/tech-debt/README.md`、`docs/specs/2026-08-01-org-runtime.md`(改訂注記)

## Visual review

- ページ: `.harness/state/plan-visual/org-role-models.html`(かんたんな図 2 つ: `overview`、`easy-legacy`。詳細の図 3 つ: `detail-overview`、`detail-resolve`、`detail-load`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 5 つの図を `shot` で撮って読んだ。詳細 2 でスライスの印がタイトルに重なっていたので位置を直して撮り直し、すべての PNG で重なり・切れ・線の貫通がないことを確かめた

## Design decisions

Critical forks: None(spec で決めた 4 点、すなわち `model_pool` と `[org.roles]` の廃止、driver とモデルの組、既定の値、codex だけのときの既定の作り方のほかに、手戻りが 1 スライスを超える分かれ目は見つからなかった)。

以下は spec の範囲でこの計画が決めた作り方で、どれも 1 スライスの中で戻せる。

- **D1 残ったキーの記録**: `config.Load` は `ralph.toml` と `ralph.local.toml` のそれぞれを pointer の probe(今の `orgPoolKeysPresent` と同じ方法)で読み、`[org].model_pool` か `[org.roles]` を書いたファイルを `OrgConfig` の TOML でない項目(`toml:"-"`)に記録する。Load 自身はエラーにも警告にもしない。spawn・start・director start と doctor がこの記録を読む。改名前の役割名で spawn を断る前例(`RetiredRoleConfigKeys`)と同じく、読み込みを止めないので `status`・`stop`・`disband` は動く。
- **D2 既定の埋め方**: `Default()` は claude の組を持つ(既定の `driver_pool` に claude があるため)。`Load` は 2 つのファイルを読んだあと、どちらにも書かれていない役割を、実際に使われる `driver_pool` で選んだ組(`config.DefaultRoleModels(driverPool)`)で埋める。`Default()` を `Load` なしで使う呼び出し(テストの組み立てなど)も claude の組を持つ。
- **D3 ファイルごとの検査**: 役割のエントリの `driver` と `model` のキーがあるかは、上書きする前にファイルごとに pointer の probe で調べる。上書きしたあとの値では、表見出しで `model` だけを書いた上書きと区別できないため。
- **D4 解決の関数**: `internal/org` に `ResolveRoleModel(cfg config.OrgConfig, role, driverFlag, modelFlag string) (driver, model string, err error)` を置く。spawn・start・start --plan・director start の CLI はこれだけを呼ぶ。規則は spec FR-3 のとおりで、`--model` の文字の検査もここで行う(spawn の `ValidateSpawnEnvelope` も同じ検査をもう一度行い、CLI を通らない呼び出しを守る)。
- **D5 モデルが `""` のときの記録**: spawn の receipt の `commanded_model` は `""` のまま記録する。codex の観測は、指示がないので `honored` を `unknown` にし、観測したモデルを `reported_effective_model` に入れ、理由に指示がなかったことを書く。spawn の出力行と `director status` は `model=(CLI default)` と出す。`spawn_started` の event には、モデルを CLI の既定にしたことを示す印を足す。`observeStopModelReceipt` は今 `corr.Model == ""` を「モデルの項目がなかった古い event」とみなして観測を飛ばすので、この印がある event では観測を続け、同じ形の receipt(honored は `unknown`、観測したモデルを記録)を足す。印のない古い event は今どおり飛ばす(計画の助言 2)。
- **D6 読み先**: `applyMainWorktreeOrgLimits` が `RoleModels`(map を丸ごと置き換える)・`DriverPool`・D1 の記録を main worktree の設定から取る。spawn と start --plan はこの関数を共有しているので、両方に効く。`withMainWorktreeDirectorStartConfig` は今 `Roles` と `ModelPool` を取っているところを、`RoleModels`・`DriverPool`・D1 の記録に替える。
- **D7 `[org.roles]` の拒否の場所**: `org.Spawn` の `retiredRoleConfigErr` と同じ位置(本番の道は冪等の早い return のあと、dry-run は先頭)に置き、spawn と start を覆う。director start は自分の検査の中で同じ文を返す。エラーは `[org.roles]` を書いたファイルを名指し、消すか `[org.role_models]` に移すよう案内する。`model_pool` の 1 行も同じ所で stderr に出す。
- **D8 文字の規則の共有**: モデル名の正規表現は `internal/config` に 1 つ置き(`[reviewer].claude_model` の検査と同じもの)、`internal/org` はそれを使う。
- **D9 driver と `driver_pool` の突き合わせは読み込みのあと**: `config.Load` は、エントリのキーの有無・`driver` の値(`claude` か `codex`)・`model` の文字だけを見る。エントリの `driver` が `driver_pool` にあるかは、D6 でどの設定の `RoleModels` と `DriverPool` を使うかを決めたあとに、`ResolveRoleModel` が解決した driver で確かめ(役割と `driver_pool` を名指して断る)、`ralph doctor` が warn で出す。理由は 2 つ。機能の org の spawn は、feature worktree の写しを `config.Load` で読んでから main の値で上書きする。写しの `driver_pool` が古く、main の `ralph.local.toml` が役割を codex に割り当てると、正しい main の設定が使われる前に写しの読み込みで失敗してしまう(計画の助言 1)。また、読み込みの失敗は `status`・`stop`・`disband` まで止める。spec の FR-5 と受け入れ条件の 1 件を、この形に直した。

## Acceptance criteria

- [ ] AC1 設定と既定: `[org.role_models]` を書かない設定では、leader・implementer・reviewer が claude / `opus`、director が claude / `""` になる。`driver_pool = ["codex"]` だけを書くと、3 役が codex / `gpt-6.1-sol`、director が codex / `""` になる。1 つの役割だけを書けば、その役割だけが置き換わる(`internal/config` のテスト)。
- [ ] AC2 ファイルごとの検査: `ralph.toml` か `ralph.local.toml` のエントリに `driver` か `model` のキーがなければ、`config.Load` はそのファイルと役割の名前を出して失敗する。インラインと表見出しの両方で確かめる。両方のキーを書いた `ralph.local.toml` は、その役割だけを置き換える。
- [ ] AC3 値の検査: エントリの `driver` が `claude` でも `codex` でもないか、`model` が文字の規則に合わないか `-` で始まれば、`config.Load` は失敗する。`model = ""` は通る。エントリの `driver` が `driver_pool` にないときは `config.Load` は通り、その役割をフラグなしで立てると `ResolveRoleModel` が役割と `driver_pool` を名指して断る(D9)。
- [ ] AC4 解決: `ResolveRoleModel` が spec FR-3 の規則どおりに決める。両方のフラグ、`--model` だけ、役割と違う `--driver` だけ(`--model` を名指して断る)、役割の `model` が `""` で違う `--driver`(CLI の既定で通る)、エントリのない役割(役割の名前と `[org.role_models].<role>` を名指して断る)、文字の規則に合わない `--model`(断る)を表のテストで打つ。
- [ ] AC5 動詞: spawn・start `<task>`・start --plan・director start を `--driver` と `--model` なしで打つと、役割の既定で立つ(`internal/cli` のテスト。spawn は `--dry-run`、start と director start はテストの runner)。spawn の `--driver` は必須でない。start の `--driver` を省くと leader の役割の driver を使う。省略時の警告は出ない。
- [ ] AC6 モデルが `""`: 座席の argv、headless の director の herdr の argv、`launch_cmd` の起動スクリプトのどれにも `--model` が入らない。director の登録は空のモデルで書けて読め、`director status` は `(CLI default)` を出す。codex の座席の receipt は honored が `unknown` で、観測したモデルが入る。spawn の観測が時間切れになり、その後に codex の session の記録ができた座席では、`ralph org stop` が観測した receipt を足す。印のない古い `spawn_started` では、stop は今どおり何も足さない。
- [ ] AC7 読み先: feature worktree の `ralph.toml` の `[org.role_models]` と `driver_pool` が main worktree のものと違うとき、`--config` なしの spawn と start --plan は main worktree の値を使う。写しの `driver_pool = ["claude"]` と、main の `ralph.local.toml` の `reviewer = { driver = "codex", ... }`、main の `driver_pool` の codex の組み合わせでも、`--config` なしの reviewer の spawn は codex で立つ。`--config` を渡せば、そのファイルの値を使う。
- [ ] AC8 残ったキー: `model_pool` を残した設定でも `config.Load` は通り、spawn・start・director start は stderr に 1 行出して進む。`[org.roles]` を残した設定では、3 つの動詞は manifest・receipt・herdr に何も書かずに断り、`ralph org status`・`stop`・`disband` は通る。`[org.role_models].lead` は新しい名前を示して断られる。
- [ ] AC9 doctor: 「Org codex model slugs」は role_models の codex のエントリ(`model` が空でないもの)を見る。`--probe-models` は role_models の組を重複なしで起動し、`model` が `""` の組は `--model` なしで起動する。codex writable root の検査は `driver_pool` に codex があるかで決まる。envelope の行は役割ごとの driver とモデルを出す。`model_pool` か `[org.roles]` を書いたファイルがあれば warn を出し、`[org.roles]` はもう制限しないと書く。`driver` が `driver_pool` にない役割のエントリがあれば warn を出す。
- [ ] AC10 init: claude だけ・codex だけ・両方の init が作る `ralph.toml` に `model_pool` はなく、`config.Load` で読め、codex だけなら役割の既定が codex の組になる。
- [ ] AC11 ロックステップ: `Default()` の 2 組とテンプレートのコメントの値が違うと、`defaults_sync_test.go` が失敗する。
- [ ] AC12 雛形と文書: leader の雛形から「`--model` は必ず明示する」がなくなり、前文に指定があるときだけ `--driver` と `--model` を渡す手順になる。`/org` skill の「既定の model_pool」節は `[org.role_models]` の節になる。`git grep -n 'model_pool\|org\.roles'` に残るのは、履歴(`docs/reports`・`docs/plans/archive`・`docs/insights`・`docs/evidence`)と、残ったキーを扱うコード・テスト・文書だけ。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る。
- [ ] AC13 全体: `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` が通る。

## Implementation outline

1. S1 設定の追加: `internal/config/role_models.go` に `OrgRoleModel` 型、`[org.role_models]` の項目、既定の 2 組と `DefaultRoleModels`、ファイルごとのキーの検査、`driver` の値と文字の規則の検査(`driver_pool` との突き合わせはしない、D9)、D1 の残ったキーの記録を足す。`Load` に組み込み、`templates/base/ralph.toml` に `[org.role_models]` のコメントを足し、`defaults_sync_test.go` に既定のコメントの検査を足す。`ModelPool` と `Roles` はまだ外さない(AC1・AC2・AC3・AC11) — verify: `go build ./... && test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. S2 org の解決と起動: `internal/org/role_models.go` に `ResolveRoleModel` と `[org.roles]` の拒否を置く。`ValidateSpawnEnvelope` から `model_pool` と `[org.roles]` の検査を外して文字の検査に替え、`RetiredRoleConfigKeys` を role_models に向け、`EnvelopeSummary` を role_models にする。空のモデルを座席の argv・receipt・`spawn_started` の印・stop のときの観測(`verbs.go`)・director の herdr の argv・起動スクリプト・登録の検査で扱う。`ResolveRoleModel` は解決した driver が `driver_pool` にあるかを確かめる(D9)。`DefaultModelForDriverAndRole` と `ModelAllowedForRole` は CLI がまだ使うので残す(AC4・AC6・AC8 の org 側) — verify: `go build ./... && test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/`
3. S3 CLI の解決と読み先: spawn・start・start --plan・director start が `ResolveRoleModel` を使うようにし、`--driver` を任意にし、start の `--driver` の既定を役割の driver にし、`resolveModelOrWarn` を消す。`applyMainWorktreeOrgLimits` と `withMainWorktreeDirectorStartConfig` に D6 を入れる。`model_pool` の 1 行を stderr に出す。flag の help を直す(AC5・AC7・AC8 の CLI 側) — verify: `go build ./... && test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Org|Director|Start|Spawn'`
4. S4 doctor: envelope の行、`--probe-models`(`internal/org/driver/probe.go` の空のモデル)、「Org codex model slugs」、codex writable root の検査を role_models と `driver_pool` に替え、残ったキーの warn を足す(AC9) — verify: `go build ./... && test -z "$(gofmt -l internal/cli internal/org/driver)" && go vet ./internal/cli/ ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Doctor|Codex|Probe'`
5. S5 古いものの除去: `OrgConfig` から `ModelPool` と `Roles` を外し、`Load` のプールの処理(`orgPoolKeysPresent` のプールの部分、`filterModelPoolByDrivers`、プールと `[org.roles]` の検査)、org の `DefaultModelForDriver`・`DefaultModelForDriverAndRole`・`ModelAllowedForRole`・`modelInPool`、`init_settings.go` の `model_pool` の処理、テンプレートの `model_pool` の配列と `[org.roles]` のコメント例を消し、テストを直す(AC10 と、AC1〜AC9 がこの形でも通ること) — verify: `go build ./... && test -z "$(gofmt -l internal/config internal/org internal/cli)" && go vet ./internal/config/ ./internal/org/... ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ ./internal/org/... ./internal/cli/`
6. S6 雛形と文書: `internal/org/prompts/leader.md` と `director.md`、`/org` と `/director` skill(`scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` へ写す)、`model-routing.md` の 2 つ、README、recipes の 2 組、`docs/tech-debt/README.md`、org-runtime spec の改訂注記を直す(AC12) — verify: `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt|Prompt|Director' && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、check-skill-sync、check-sync、secret scan を含む)。
- Spec compliance criteria to confirm: spec `docs/specs/2026-11-05-org-role-models.md` の FR-1〜FR-9 と受け入れ条件 15 件を、この計画の AC1〜AC13 とテスト名に対応づけて確かめる。とくに FR-3 の 4 つの規則、FR-5 のファイルごとの検査と読み込みのあとの `driver_pool` の突き合わせ、FR-6 の 2 つのキーの扱いの違い(`model_pool` は 1 行、`[org.roles]` は拒否)。
- Documentation drift to check: `git grep -n 'model_pool\|org\.roles\|プール先頭\|--model は必ず'` の残り。`ralph org spawn --help`・`ralph org start --help`・`ralph org director start --help` の `--driver` と `--model` の説明。`/org` skill の `spawn`・`start`・`director` の行。AGENTS.md の repo map の `internal/config/` の行(`[org.role_models]` を足すか)。
- Evidence to capture: `./scripts/run-static-verify.sh` の結果、`--dry-run` の spawn がフラグなしで出す行(既定、`driver_pool = ["codex"]`、`model_pool` が残った設定、`[org.roles]` が残った設定)、`ralph doctor` の envelope・slug・残ったキーの行。

## Test plan

- Unit tests:
  - `internal/config`: 既定の 2 組の選び方、1 役割だけの上書き、`ralph.local.toml` のインラインと表見出し、キーが欠けたエントリ(ファイルごと)、`claude`・`codex` 以外の driver(失敗)と `driver_pool` にない driver(読み込みは通る)、文字の規則(空白、引用符、`;`、`$`、先頭の `-`、`""`)、残ったキーの記録(どちらのファイルが書いたか)、改名前の役割名。
  - `internal/org`: `ResolveRoleModel` の表のテスト(AC4 の 6 通りと、役割の `model` が `""` で `--model` を渡す場合、役割の driver が `driver_pool` にない場合)、`ValidateSpawnEnvelope` の文字の検査、空のモデルの argv と receipt、spawn の観測が時間切れになり stop の前に session の記録ができた座席の stop の receipt(印のない古い event では足さないことも)、`EnvelopeSummary` の決まった順の出力、`RetiredRoleConfigKeys` の role_models、`[org.roles]` の拒否で manifest と receipt に何も書かれないこと、director の登録と起動スクリプトの空のモデル。
- Integration tests:
  - `internal/cli`: spawn の `--dry-run` をフラグなし・`--model` だけ・`--driver` だけで打つ。start `<task>` と start --plan(テストの runner)、director start(herdr と `launch_cmd`)。main worktree と feature worktree の `ralph.toml` が違う台帳での読み先(今の max_orgs のテストと同じ組み立て)。写しの古い `driver_pool` と main の `ralph.local.toml` の codex の役割の組み合わせ(AC7)。`model_pool` が残った設定の stderr の 1 行と、`[org.roles]` が残った設定の拒否、その設定での `status`・`stop`・`disband`。doctor の 4 つの検査と warn。init の 3 通りの出力を `config.Load` で読むこと。
- Regression tests: `go test ./...` の全体(S5 のあと)。今の spawn・start・director start のテストのうち、`--driver` と `--model` を明示するものは、今と同じ結果になる。`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` のような model_pool の復旧のテストは、置き換えるか消し、消すものは self-review の report に理由を書く。
- Edge cases: 役割の名前に改名前の名前を使う、`[org.role_models]` に director だけを書く、`driver_pool = ["codex"]` で claude のエントリを明示する(失敗)、`--model ""` を明示する(省略と同じに扱う)、`--driver` と役割の driver が同じで `--model` なし、エントリのない独自の役割で `--driver` と `--model` を渡す。
- Evidence to capture: `./scripts/run-test.sh` の結果と、新しく足したテストの名前の一覧。

## Risks and mitigations

- **R1 下流の `model_pool` の 1 行**: テンプレートから写した `model_pool` は下流のほぼすべての `ralph.toml` にあるので、消すまで spawn のたびに stderr に 1 行出る。止めはしない。1 行には消せばよいことを書き、release ノートにも書く。
- **R2 `[org.roles]` を書いた下流の spawn が止まる**: 消すか `[org.role_models]` に移すまで spawn・start・director start が断られる。エラーにファイルと直し方を書き、`stop` と `disband` は止めない。release ノートに書く。
- **R3 版が混ざったときの director の登録**: 新しい ralph が空のモデルで書いた director の登録を、古い ralph(更新前から動いている見張りなど)が読むと、モデルが空だとして断る。release ノートに「更新の前に見張りと headless の director を止める」と書く(8a 系列の release と同じ注意)。
- **R4 既定のモデルが変わる**: `--model` を省いたときのモデルが、claude は `fable` から `opus` に、codex は `gpt-6-astra` から `gpt-6.1-sol` に、director は CLI の既定に変わる。release ノートに書く。leader の雛形は今まで `--model` を明示させていたので、雛形どおりに動いていた org の座席は、前文の指定がない限り role_models の値に変わる。
- **R5 綴り違いが起動まで見つからない**: プールの検査がなくなるので、文字の規則に合う綴り違いは座席が起動してから失敗する。doctor が role_models を検査し、codex の座席は receipt が実効モデルを記録する。leader の雛形は、ふだん `--model` を渡さない手順にする。
- **R6 差分が大きい(約 50 ファイル)**: 6 つのスライスに分け、各コミットでリポジトリ全体がビルドでき、そのスライスの検査が通るようにする。S5 で古いものをまとめて外す。
- **R7 leader が任意のモデルを選べる**: `--model` に許可リストはない。今の `model_pool` も、leader が書き換えられる feature worktree の写しから読んでいたので強い制限ではなかった。D6 で役割の既定は main worktree から読む。指示したモデルは receipt に残る。
- **R8 doctor の probe の費用**: `--probe-models` は opt-in のまま。対象は role_models の重複を除いた組なので、今の 8 エントリより少ない。

## Rollout or rollback notes

- release ノート(次の release を切るときにユーザーが使う): `[org].model_pool` と `[org.roles]` の廃止と移し方、`--model` を省いたときの既定の変更(R4)、`[org.roles]` が残ると spawn が断られること(R2)、更新の前に見張りと headless の director を止めること(R3)。
- 戻すときは PR を revert する。古い binary は `[org.role_models]` を知らないキーとして読み飛ばし、`model_pool` がなければ組み込みのプールを使うので、下流が `model_pool` を消していても読み込みは通る。

## Open questions

なし。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件をどちらも計画に入れた。(1) [HIGH] feature worktree の写しが main の値で上書きされる前に `config.Load` の検査で失敗しうる → D9 を足し、driver と `driver_pool` の突き合わせを読み込みのあとに移し、AC3・AC7・AC9 と spec の FR-5・受け入れ条件を直した。(2) [MEDIUM] モデルが `""` の座席で stop のときの観測が飛ばされる → D5 に `spawn_started` の印と `observeStopModelReceipt` の変更を足し、AC6 と Affected areas に `verbs.go` を入れた
- [x] Plan approved — 2026-10-11 のユーザーの指示(作った計画は自動承認、承認待ちは不要)に従って承認した。図解ページは `.harness/state/plan-visual/org-role-models.html`
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
