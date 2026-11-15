# gate で走ったツールの版を残し、doctor が repo の求める Node の版と座席の shell の版を比べる(#311)

- Status: Approved
- Approved: 2026-11-13 sha256:2774af0aab07
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/311(codex の座席の gate が、mise の shim のせいで repo の求めと違う Node で走り、`pnpm install` が 2 回落ちた。gate の log には版が残らず、doctor も気づけなかった)
- Related issue: 311
- Type: feat
- Branch: feat/gate-tool-versions

## Objective

`ralph org gate` と、scripts を走らせる `ralph org regate` が、scripts の前に主なツール(node、pnpm か yarn か npm、python3、go)の版と解決先を、測った場所ごとに log と記録と CLI の出力に残すようにする。`ralph doctor` に、repo が求める Node とパッケージマネージャの版(`.node-version`、`.nvmrc`、`package.json` の `engines.node` と `packageManager`)と、座席と同じ非対話の shell(`$SHELL -lc`)で解決される版を、同じ場所の一覧で比べる Check を足す。測ることで環境や作業ツリーを変えない。手順書に、mise や asdf の shim が子のプロセスの PATH を書き換える件と、`MISE_<TOOL>_VERSION` での固定を書く。issue #311 の受け入れ条件 1・2 をこの PR で満たす。

## Scope

- 新しいパッケージ `internal/toolchain`: 測る場所の一覧(git ls-files の印)、場所ごとのパッケージマネージャ、1 回の sh で測るスクリプトと環境、出力の版のトークンの取り出し、shim の印、npm の範囲の部分集合の読み取り。gate と doctor が共有する
- gate のツールの版(`internal/org/gate_tools.go`、`runGateScripts` の見出しの段から呼ぶ)。log の行、記録の鍵、CLI の `TOOLS:` の行、probe の前後の作業ツリーの比べ
- doctor の Node の Check(`internal/cli/doctor_node.go`、Check 13 のあと、並びの最後)
- `docs/recipes/isolated-herdr-server.md` の「Shell aliases」の次の節
- reviewer の役割 prompt、`/org` skill と director skill(4 つの写し)の `TOOLS:` の行の扱いと、それを固定するテスト
- `ralph org gate --help`、`ralph org regate --help`、`ralph doctor --help` の該当箇所、AGENTS.md の Repo map(`internal/toolchain/` の行と internal/org の行)、`docs/tech-debt/README.md`

## Non-goals

- scripts が python3 などの shim を通して起こす子が見る版の測定(D1。log で手がかりを並べるだけにする)
- python・go の版の求め(`.python-version`、`go.mod` の `go`/`toolchain`)と座席の shell の比べ(#311 は Node と pnpm を挙げている)
- `mise exec` や `mise x` での再現
- 座席ごと(claude と codex)の shell の違いの自動の見分け(doctor は doctor を打った人の `$SHELL` で測る)
- #308 の原因の確定(手順書には未確認と書いて触れるだけ)

## Assumptions

- `exec.LookPath` は ralph 自身の PATH を見る。scripts と同じ PATH で解決するには、scripts と同じ環境(`gateScriptEnv(env)`)を渡した `/bin/sh -c` の中で `command -v` を打つ
- gate の記録の Details は、知らない鍵を古い binary も新しい binary も読み飛ばすが、空白を含むトークンや `key=value` でないトークンがあると記録ごと読めなくなる(`gateRecordFromDetails` が `ok=false` を返す)
- `go version` は `go version go1.27.3 darwin/arm64`、`python3 --version` は `Python 3.12.1`、`node -v` は `v24.22.1`、`pnpm -v` は `10.10.0` の形で出す
- `package.json` の `packageManager` は `pnpm@10.10.0` か `pnpm@10.10.0+sha512.<hash>` の形(corepack の決まり)
- mise の shim は、設定にあって入っていない版を呼ばれると既定で自動インストールする(`MISE_AUTO_INSTALL`、`MISE_NOT_FOUND_AUTO_INSTALL` で止まる)。corepack は `COREPACK_ENABLE_AUTO_PIN` が有効だと `package.json` に `packageManager` を書き、`COREPACK_ENABLE_NETWORK=0` で取得を止める(計画助言が示した mise と corepack の文書による。実機では確かめていない)
- nvm・mise・asdf の版のファイルは、そのディレクトリから上へ探される
- codex の座席はコマンドを非対話の shell で打つ。doctor の `$SHELL -lc` はそれに近いが、同じとは限らない(未確認)

## Affected areas

- `internal/toolchain/`(新しいパッケージ): `projects.go`(測る場所の一覧)、`probe.go`(スクリプト、環境、出力の読み取り)、`version.go`(版のトークン、shim の印)、`npmrange.go`(npm の範囲の部分集合)と各テスト
- `internal/org/gate_tools.go`(新)、`internal/org/gate_tools_test.go`(新)、`internal/org/gate_run.go`(見出しの段で probe を呼ぶ)、`internal/org/gate_record.go`(`tool_*` の鍵)、`internal/org/gate.go`、`internal/org/regate.go`(結果を記録と GateResult・RegateResult に渡す)
- `internal/cli/org_gate.go`、`internal/cli/org_regate.go`(`TOOLS:` の行)
- `internal/cli/doctor_node.go`(新)、`internal/cli/doctor_node_test.go`(新)、`internal/cli/doctor.go`(Check 14 の 1 行と help)、`internal/cli/main_test.go`(差し替え口の固定)
- `internal/org/prompts/reviewer.md`、`internal/org/prompts_test.go`
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と、それぞれの `.agents/skills/` と `templates/base/` の写し、`internal/org/org_skill_send_sync_test.go`、`internal/org/director_skill_sync_test.go`
- `docs/recipes/isolated-herdr-server.md`、AGENTS.md、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/gate-tool-versions.html`(やさしい図 2 枚: overview、easy-doctor。折りたたみの中に詳しい図 2 枚: detail-overview、detail-probe)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: 全体と 4 枚の図をそれぞれ `shot` で撮って読んだ。detail-probe の箱の文の詰まりを直し、計画助言の反映(共有のパッケージ、場所ごとの測定、止める変数)で詳しい図 2 枚と「承認の前に」を描き直して撮り直した。重なり・はみ出しなし

## Design decisions

critical forks は無い(下の D1〜D8 は既定の案と計画助言で決まる)。consult(consult-plan-toolversions)と codex の計画助言に聞き、推す案を自律で選んだ。

- D1 解決の仕方(自律で選んだ): scripts と同じ環境と cwd で `/bin/sh -c` を 1 回だけ打ち、測る場所ごとに `cd` して、ツールの `command -v` と版を順に出させる。scripts が python3 の shim を通して起こす子は、shim が PATH を書き換えると別の版を見うる(issue の場面)。その版は測らず、手がかりを log に並べる: 解決先のパスが mise か asdf の shims の下なら `(mise shim)` か `(asdf shim)` と付け、mise が PATH にあれば `mise current` の出力を `# mise current:` の行に書き、見出しに「scripts の子は shim で別の版を見うる」と 1 行書く。
- D2 測る場所(計画助言の MEDIUM): `internal/toolchain` が、worktree の `git ls-files` の追跡されているファイルから測る場所の一覧を作り、gate と doctor が共有する。場所は、lockfile(`pnpm-lock.yaml`、`yarn.lock`、`package-lock.json`)か版のファイル(`.node-version`、`.nvmrc`、`.tool-versions`、`mise.toml`、`.mise.toml`)があるディレクトリと、JS の印(`package.json` を含む)が 1 つでもあれば root。上限は 10 か所(パッケージ変数)で、超えた分は log に数を書く。場所ごとのパッケージマネージャは、その場所か上のディレクトリで最も近い lockfile(`pnpm-workspace.yaml` は pnpm)、無ければその場所の `package.json` の `packageManager`、無ければ root の `packageManager`、どれも無ければ npm。`go.mod` があれば go を root で、python3 はいつも root で測る(scripts の解釈系としてよく使われ、issue の原因もそれだった)。`git ls-files` が失敗すれば python3 だけを root で測り、log にそう書く。
- D3 時間の上限と止め方: probe は gate.lock を持ったまま走るので、全部を 1 回の sh にまとめて 1 つの予算(15 秒、パッケージ変数)にする。sh は自分の process group で走らせ、時間切れと ctx の終わりで group ごと止める(`runGateScript` と同じ Setpgid と WaitDelay。shim の子が stdout を握っても待ち続けない)。時間切れや失敗では、測れなかったツールを `unknown` にして gate を続ける。ctx の終わりは gate の割り込みと同じ扱いで、何も記録しない。
- D4 記録の値: `gate_passed` と `gate_failed` の Details に、測ったツールの `tool_node=v24.22.1 tool_pnpm=10.10.0 tool_python3=3.12.1 tool_go=go1.27.3` を `log=` の前に書く。値は出力から版のトークンを取り出したもの(`Python 3.12.1` は `3.12.1`、`go version go1.27.3 darwin/arm64` は `go1.27.3`)で、`^[A-Za-z0-9._+-]+$` に合わなければ `unknown`、PATH に無ければ `missing`、場所によって版が違えば `mixed`。測った場所の数を `tool_dirs=<n>` に書く。場所ごとの版と解決先のパスは、空白を含みうるので記録に入れず、log の `# tool <名前> (<場所>): <版> (<解決先>)` の行だけに書く。
- D5 CLI と座席: `ralph org gate` と `ralph org regate` は `LOAD:` の行の後ろに `TOOLS: node v24.22.1, pnpm 10.10.0, python3 3.12.1` の 1 行を出す(`mixed` はそのまま出し、場所ごとの版は log を見るよう書く)。reviewer の役割 prompt は、`LOAD:` と同じく `TOOLS:` を `GATE: fail` の BLOCKED の本文に写す。`/org` skill と director skill は、gate の失敗を読むときに `TOOLS:` と log の `# tool` の行を見ることを書く。
- D6 doctor の Node の Check: Check 13 のあと、並びの最後に「Node toolchain」として置く(別レーンの #310 が Check 3 の直後に足すので離す)。D2 の一覧に JS の場所が無ければ shell を打たず、info の 1 行(Node の印なし: skipped)にする。あれば、場所ごとに `cd` する 1 回の `$SHELL -lc` で node とパッケージマネージャの `command -v` と版を打つ(`$SHELL` が空なら `/bin/sh` で、Detail にそう書く。スクリプトは POSIX の sh の文法なので、`$SHELL` の名前が sh・bash・zsh・dash・ksh・mksh・yash でなければ(fish など)、`/bin/sh -lc` で測り、Detail にそう書く。S1 の実装で分かった)。20 秒の時間切れ(パッケージ変数)、差し替え口のパッケージ変数を internal/cli の TestMain で固定する。場所ごとの求めは、その場所か上で最も近い `.node-version`・`.nvmrc`、その場所の `package.json`(無ければ root)の `engines.node` と `packageManager`。比べ方: `.node-version`・`.nvmrc` は完全な版か部分(`24`、`24.15`)を、`v` の有無を問わずに接頭辞で比べる。`lts/*` などの別名は比べずに info。`engines.node` は npm の範囲の部分集合(比較演算子、`^`、`~`、`x`/`*` の部分版、空白区切りの AND、`||`、ハイフンの範囲)を読み、読めなければ info。`packageManager` は `+` 以降を落として完全一致。どれか合わなければ warn(場所、合わない欄、求める版、解決した版と解決先、直し方として `MISE_NODE_VERSION` などと corepack)。shell の失敗と時間切れも warn。warn は exit code を変えない(doctor の今の決まり)。
- D7 測ることで何も変えない(計画助言の HIGH): gate の probe と doctor の shell には、ほかの変数の上に `MISE_AUTO_INSTALL=0`、`MISE_NOT_FOUND_AUTO_INSTALL=0`、`COREPACK_ENABLE_AUTO_PIN=0`、`COREPACK_ENABLE_NETWORK=0`、`COREPACK_ENABLE_DOWNLOAD_PROMPT=0`、`NPM_CONFIG_UPDATE_NOTIFIER=false`、`GOTOOLCHAIN=local`、`pnpm_config_manage_package_manager_versions=false`、`npm_config_manage_package_manager_versions=false` を置く(呼ぶ側の値より後ろに置き、上書きする)。最後の 2 つは、pnpm が `packageManager` の別の版を自分で取りに行くのを止める(verify の V1 で分かった。pnpm 12.9.1 で、どちらか 1 つで取りに行かなくなることを確かめた。`--config.` の形は効かない)。`GOTOOLCHAIN=local` は、`go version` が go.mod の toolchain を取りに行かないため(S1 の実装で分かった。取りに行く先は worktree の外の module cache で、前後の比べでは見えない)。入っていない版やダウンロードの要る版は測れず、`unknown` になる(go.mod が手元より新しい go を求めれば、そのことが理由に出る)。scripts に渡す環境は変えない。gate は probe の前後で `git status --porcelain --untracked-files=all` を比べ、変わっていれば log と stderr にその行を書く(そのあとの scripts の結果は今までどおり `tree_changed` で記録される)。
- D8 共有の置き場所: 測る場所の一覧、スクリプトと環境、出力の読み取り、npm の範囲は、gate(internal/org)と doctor(internal/cli)が使うので、どちらにも属さない新しいパッケージ `internal/toolchain` に置く。gate と doctor が同じ一覧を使うことを、パッケージのテストで固定する。

## Acceptance criteria

- [x] AC1 gate(と local の regate)の log の見出しに、scripts の前の `# tool <名前> (<場所>): <版> (<解決先>)` の行が、測った場所とツールごとに出る(PATH に無ければ `missing`、読めなければ `unknown (<理由>)`)。見出しに「scripts の子は shim で別の版を見うる」の 1 行があり、解決先が shim なら `(mise shim)` か `(asdf shim)` が付き、mise が PATH にあれば `# mise current:` の行が出る(issue の受け入れ条件 1)
- [x] AC2 `gate_passed` と `gate_failed` の Details に、測ったツールの `tool_<名前>=<版>` と `tool_dirs=<n>` が `log=` の前に入り、`gateRecordFromDetails` が読める。空白を含む出力や任意の文は値にならず(`unknown`)、場所によって版が違えば `mixed` になり、記録が読めなくなることはない。`tool_*` の無い前の形の Details も読める
- [x] AC3 測る場所の一覧: root だけの repo、root に印が無く `apps/a` が pnpm・`apps/b` が yarn の monorepo、サブディレクトリだけに版のファイルがある repo、pnpm の workspace(lockfile は root、package.json はサブ)で、場所とパッケージマネージャを正しく見分ける。上限を超えた分は log に数が出る。python3 はいつも、go は `go.mod` があれば root で測る。`git ls-files` が失敗すれば python3 だけを測り、log にそう書く
- [x] AC4 probe は 1 回の sh で、全部で 15 秒を超えない。時間切れと失敗では gate の verdict は変わらず、測れなかったツールが `unknown` になる。shim の子が stdout を握っても 15 秒と猶予のうちに戻る。ctx の終わりでは probe も止まり、何も記録しない
- [x] AC5 `ralph org gate` と `ralph org regate` の出力に `TOOLS:` の行が出る(scripts が走ったとき)。reviewer の役割 prompt、`/org` skill、director skill(4 つの写し)が `TOOLS:` を扱い、それを固定するテストが通る
- [x] AC6 `ralph doctor` の Check「Node toolchain」: JS の場所が無い repo では shell を打たずに info。場所ごとの `.node-version`・`.nvmrc`・`engines.node`・`packageManager` と、`$SHELL -lc` で解決した node とパッケージマネージャの版が合わなければ warn(issue の受け入れ条件 2)。合えば pass。別名や読めない範囲は info。shell の失敗・時間切れは warn で、exit code は変わらない。`$SHELL` が空なら `/bin/sh` を使い、そう書く
- [x] AC7 npm の範囲の読み取りが、形ごとの表のテスト(完全な版、部分、`^`、`~`、比較演算子、AND、`||`、ハイフン、`x`、読めない形)で確かめられる
- [x] AC8 手順書の「Shell aliases」の次に mise・asdf の節があり、shim が子の PATH を書き換えること、`MISE_<TOOL>_VERSION` での固定、`mise current` と gate の log の `# tool` の行での確かめ方、#308 の見込みの原因(未確認)を書く
- [x] AC9 `ralph org gate --help`、`ralph org regate --help`、`ralph doctor --help`、AGENTS.md の Repo map、tech-debt がこの挙動に合う。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC10 測ることで何も変えない: gate の probe と doctor の shell の環境に D7 の 9 つの変数が入り、呼ぶ側の `COREPACK_ENABLE_AUTO_PIN=1` などを上書きする(テストで確かめる)。偽の shim(呼ばれると worktree にファイルを書き、install の印を残す)を PATH に置いた試験で、変数が効けば何も書かれず、効かない偽の shim では gate が probe の前後の違いを log と stderr に出す。scripts に渡す環境は変わらない

## Implementation outline

1. S1 `internal/toolchain`: 測る場所の一覧、場所ごとのパッケージマネージャ、1 回の sh のスクリプトと環境(D7 の変数)、出力の読み取り、版のトークン、shim の印、npm の範囲の部分集合。AC3、AC7、AC10 の環境の部分 — verify: `test -z "$(gofmt -l internal/toolchain)" && go vet ./internal/toolchain/ && go test -timeout=30m ./internal/toolchain/`
2. S2 gate のツールの版: `gate_tools.go`(toolchain を呼び、process group で走らせ、log の行と記録の値を作る、probe の前後の作業ツリーの比べ)、`gate_run.go` の見出しの段、`gate_record.go` の `tool_*` の鍵、`gate.go` と `regate.go` から記録と結果へ、CLI の `TOOLS:` の行。AC1、AC2、AC4、AC5 の CLI、AC10 の gate の部分 — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && go test -timeout=30m ./internal/org/ -run 'Gate|Regate|Tool' && go test -timeout=30m ./internal/cli/ -run 'Gate|Regate'`
3. S3 doctor の Node の Check: `doctor_node.go`(toolchain の一覧、`$SHELL -lc`、場所ごとの求めと比べ)、`doctor.go` の Check 14 の 1 行と help、TestMain の差し替え口。AC6、AC10 の doctor の部分 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && go test -timeout=30m ./internal/cli/ -run 'Doctor|Node'`
4. S4 文書と座席: 手順書の節、reviewer の役割 prompt、`/org` skill と director skill の 4 写し、sync のテスト、help の残り、AGENTS.md、tech-debt。AC5 の残り、AC8、AC9 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && go test -timeout=30m ./internal/org/ -run 'Prompt|Skill' && go test -timeout=30m ./internal/cli/ -run 'Help|Doctor'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、skill と templates の同期)
- Spec compliance criteria to confirm: AC1〜AC10。issue の受け入れ条件 1(gate の log に Node と pnpm の版)と 2(doctor が違いに warn)が AC1 と AC6 で覆われていること。計画助言の 2 件が D2・D7 と AC3・AC10 に出ていること
- Documentation drift to check: 各 help、reviewer の役割 prompt、2 つの skill の 4 写し、手順書、AGENTS.md、tech-debt
- Evidence to capture: verify の report に run-static-verify の結果と AC ごとの根拠

## Test plan

- Unit tests:
  - 測る場所の一覧: AC3 の構成ごと、上限、`git ls-files` の失敗
  - 版のトークンの取り出し: node、pnpm、yarn、npm、python3、go の実際の出力の形、空、任意の文、空白、制御文字
  - shim の印: mise と asdf のパス、shim でないパス
  - 環境: D7 の 9 つの変数が呼ぶ側の値を上書きすること
  - probe: 差し替え口の sh で、全部見つかる、一部 missing、場所で版が違う(`mixed`)、時間切れ(短くした予算)、stdout を握る子(group ごと止まる)、ctx の終わり
  - 記録の鍵の往復、前の形の読み取り、`unknown`・`missing`・`mixed`
  - doctor: JS の場所なしで shell を打たない、`.node-version`・`.nvmrc`・`engines.node`・`packageManager` のそれぞれの合う・合わない、上のディレクトリの版のファイル、別名、読めない範囲、shell の失敗・時間切れ、`$SHELL` が空、monorepo の場所ごとの warn
  - npm の範囲: 表のテスト(AC7)
- Integration tests:
  - 実際の git の worktree で、package.json と pnpm-lock.yaml を置いた gate の log と記録に `# tool` の行と `tool_*` の鍵が出る
  - 偽の shim(呼ばれると worktree にファイルを書く)を PATH に置いた gate: D7 の変数が効く偽の shim では何も書かれない。効かない偽の shim では probe の前後の違いが log と stderr に出る(AC10)
  - CLI の `ralph org gate` の出力に `TOOLS:` の行
  - `ralph doctor` の出力に「Node toolchain」の行(差し替え口の shell)
- Regression tests: 既存の gate・regate・doctor のテストがそのまま通ること(Details を全文で比べる既存のテストは、固定した probe の結果に合わせて期待値を直す)
- Edge cases: 空の `package.json`、壊れた JSON、`engines` が文字列でない、`packageManager` が無い、`.nvmrc` の末尾の改行やコメント、`node -v` が 2 行出す shim、空白を含むディレクトリの名前
- Evidence to capture: test の report に `./scripts/run-test.sh` の結果とテストの名前

## Risks and mitigations

- R1 probe が gate.lock を持ったまま走るので、台帳のほかの gate を待たせる → 1 回の sh で 15 秒の予算にまとめ、group ごと止める(D3)
- R2 直接測った版が、scripts が shim を通して起こす子の版と違いうる(issue の場面) → 手がかり(shim の印、`mise current`、見出しの 1 行)を並べ、Known gaps と tech-debt に書く(D1)
- R3 記録の値の形が崩れて記録ごと読めなくなる → 値を 1 トークンに限り、合わなければ `unknown`(D4、AC2)
- R4 doctor の `$SHELL -lc` が codex の座席の shell と違う → Detail に使った shell を書き、手順書で座席の shell を合わせる方法を書く。tech-debt に残す
- R5 npm の範囲の自前の読み取りが npm と違う結果を出す → 部分集合だけを読み、読めなければ info にして warn を出さない(D6)。表のテストで形ごとに固定する
- R6 ほかのレーンが同じファイル(doctor.go、AGENTS.md、tech-debt、skill の写し)を変える → PR を作る前と merge の前に origin/main を取り込む。doctor の Check は #310 の置き場所(Check 3 の直後)から離す
- R7 D7 の変数が効かないツール(別の版の管理ツール、自前の shim)は、測るときに環境を変えうる → gate は probe の前後の作業ツリーを比べて知らせる(D7)。worktree の外(共有のツールの置き場)の変化は見えないので、tech-debt に残す
- R8 測る場所の上限(10)を超える monorepo では、一部の場所を測らない → log に数を書き、tech-debt に残す

## Rollout or rollback notes

- 新しい鍵 `tool_*` は古い binary が読み飛ばす。rollback すると gate の log と記録から版の行が消えるだけ
- doctor の新しい Check は warn か info だけで、exit code を変えない
- release のノートに: gate の `TOOLS:` の行と log の `# tool` の行、doctor の「Node toolchain」、手順書の mise・asdf の節、probe が自動インストールと自動 pin を止めること

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件を計画に反映した。(1) [HIGH] 版の確認が環境や作業ツリーを変えうる(mise の shim の自動インストール、corepack の自動 pin)→ probe と doctor の shell に自動インストール・通信・自動 pin を止める 6 つの変数を置き、gate は probe の前後の作業ツリーを比べる(D7、AC10、R7)。(2) [MEDIUM] monorepo で実際に使う場所のツールと版を取り違える → gate と doctor が共有する測る場所の一覧(lockfile と版のファイルのある場所)を新しいパッケージ `internal/toolchain` に置き、場所ごとに測る(D2、D8、AC3、R8)。slice を 4 つにした。consult(consult-plan-toolversions)の判定「直してから進める」も反映済み(D3 の 1 回の sh と 15 秒、D4 の値の形、D5 の `TOOLS:` の写し、D6 の印なしの扱い)
- [x] Plan approved — ユーザーの 2026-10-11 と 2026-11-12 の指示(作った計画は自動承認)に従って承認を記録した
- [x] Branch created
- [x] Implementation started
  - S1 1a4d4a0a(implementer): `internal/toolchain`(測る場所の一覧、`ProbeEnv`、`BuildScript`、`ParseProbe`、`Probe`、`Run`、版のトークン、shim の印、npm の範囲、版のファイルの spec、`packageManager`)。検証コマンド exit 0、`-count=3` と `-race` も通り、golangci-lint 0 件、変異 13 個のうち 12 個が落ちた(残る 1 個は `</dev/null` で、文字列の固定で補った)。ツールの出力をコマンド置換に入れず直接流す(shim の子が pipe を握っても止まらない)。自己点検: コメント 3 か所を直した、ほかは clean
  - 計画の直し(2026-11-13): S1 の申し送り 2 件を D6・D7・AC10・Test plan に入れた。(1) `go version` が go.mod の toolchain を取りに行くので `GOTOOLCHAIN=local` を止める変数に足した(7 つ)。(2) `$SHELL` が POSIX の sh の仲間でなければ(fish など)doctor は `/bin/sh -lc` で測る。digest が変わったので、図(止める変数を「…」で省いていて変更不要)を確かめ、ユーザーの指示(自動承認)で承認を取り直した(df776faef33e)
  - S2 43ee884f(implementer): `gate_tools.go`(probe、log の行、場所をまたぐ集約、記録の値、probe の前後の `git status` の比べ)、見出しの段、`tool_dirs` と `tool_*` の鍵、GateResult・RegateResult の `Tools`、CLI の `TOOLS:` の行、`ProbeEnv` に `GOTOOLCHAIN=local`。検証コマンド exit 0 を 2 回(org 603 s と 548 s)、golangci-lint 0 件、変異 2 個が落ちた。Details と結果を全文で比べる既存のテストを、TestMain で固定した probe の結果に合わせて直した。自己点検: 長すぎるコメント 5 か所を直した、ほかは clean。申し送り: `internal/toolchain/doc.go` に `GOTOOLCHAIN=local` が無い、gate と regate の help(S4)
  - S3 ff342a39(implementer): `doctor_node.go`(Check 14「Node toolchain」、場所ごとの求め、`$SHELL` の選び方、warn は 1 件 1 行)、`doctor.go` の 1 行、TestMain の固定。検証コマンド exit 0 を 2 回、`-count=3 -race` と golangci-lint 0 件、変異 10 個がすべて落ちた。`ralph doctor --help` には Check の一覧(Long)が無いので help は変えていない(AC9 の doctor の help は、今の help と食い違わないことを確かめる形にする)。自己点検: テストの無駄な条件を 1 か所直した、ほかは clean
  - origin/main(#323、14531858)を取り込んだ(4af8276b、衝突なし、`go vet` は通る)
  - S4 d44583c8(implementer): 手順書の `### Version managers (mise, asdf)`(2 つの写し)、reviewer の役割 prompt の `TOOLS:` の写し、`/org` skill と director skill の 4 写し、固定のテスト、gate と regate の help、`internal/toolchain/doc.go`、AGENTS.md の Repo map、tech-debt の 4 行。検証コマンド exit 0 を 2 回。`ralph doctor --help` には Check の一覧が無いので変えていない。自己点検: 折り返しとコメントの文を直した、ほかは clean。申し送り: 手順書の #308 の節から新しい節への参照が無い(sync-docs で扱う)
  - 全 slice の検証コマンドが通った(S1〜S4)
  - self-review(cycle 1 の 1 回目、245b85f9): CRITICAL・HIGH・MEDIUM 0、LOW 7、Merge: yes
  - origin/main(#324、4220ea7c)を取り込んだ(a734cfb6、衝突なし。doctor の並びは Check 3b … Check 13、Check 14)
  - S5 b0b3bfe2(implementer): self-review の LOW 1・2・3・5・6・7 を直した。doctor の probe を signal で止める、どの場所も絶対パスへ cd(login shell の cd に左右されない)、FindLayout のエラーを行に出す、help・手順書・org skill の probe の上限と「何も書かない」の文を正確にする、probe の中の割り込みで「scripts を止めた」と言わない。LOW 4(ProbeEnv を 2 回当てる)は害が無いので残した。検証コマンド exit 0 を 2 回。自己点検: 不要な変数とテストの入力を直した、ほかは clean。self-review を取り直す
  - self-review(cycle 1 の 2 回目、5b84df9a): CRITICAL・HIGH・MEDIUM 0、LOW 5(S5 で入った 4 件と、残すと決めた LOW 4)、Merge: yes
  - S6 9c19e5ba(implementer): 2 回目の LOW 1(probe を Ctrl-C で止めると doctor が「All checks passed.」と exit 0 で終わる。知らせが消える向きなので直した)、LOW 3(doc.go の言い切り)、LOW 4(root の nocd の記録が測れた値を上書きしうる)。中断した doctor は、全部の結果を出したあと「Interrupted」の行を出して exit 1 で終わる(D6 の「warn は exit code を変えない」は warn の話で、中断は別。cleanup・merge・regate の中断も exit 1)。LOW 2(文字を濾す処理が 3 か所)は sync-docs で tech-debt に書く。検証コマンド rc=0 を 2 回、golangci-lint 0 件、変異 3 個がすべて落ちた。自己点検: 長すぎるコメント 1 行を直した、ほかは clean。変更が小さく狙いが決まっているので self-review の 3 回目は取らず、verify・test・cross-review で全体を見る
  - verify(cycle 1、5389757f): pass、MEDIUM 1(V1)、LOW 5(V2〜V6)。V1: pnpm は `packageManager` が別の版を指すと、7 つの変数をくぐって自分でその版を取ってきて worktree の外(pnpm の store)に書く。D7 に反するので直す。手元で、取りに行けない版名(`pnpm@0.0.1-doesnotexist`)を置いた scratch の dir で確かめた: 何も付けないと registry に取りに行って WARN を出し、`pnpm_config_manage_package_manager_versions=false` か `npm_config_manage_package_manager_versions=false` を付けると取りに行かずに手元の 12.9.1 を返す。`--config.` の形は効かない。計画の D7・AC10・Test plan を 9 つの変数に直し、図(変数を「…」で省いていて変更不要)を確かめ、ユーザーの指示(自動承認)で承認を取り直した(2774af0aab07)。verify が観察のために手元の pnpm の store に pnpm@1.0.0 を入れた件は、書く前の中身が分からないので戻していない(verify の report と evidence の log に記録がある)。V2〜V6 は sync-docs に渡す
  - S7 e7274fec(implementer): `pnpm_config_manage_package_manager_versions=false` と `npm_config_manage_package_manager_versions=false` を止める変数に足した(9 つ)。手元の pnpm 12.9.1 は大文字の綴り(`PNPM_CONFIG_…`、`NPM_CONFIG_…`)も読み、どれか 1 つが `true` なら取りに行くので、`ProbeEnv` は呼ぶ側の変数を大文字小文字を問わず消してから 9 つを置く。偽の pnpm で store に書かないことのテスト、doc.go・手順書の 2 写し・gate と regate の help・org skill の 4 写し・tech-debt の行を直した。検証コマンド rc=0、変異 4 個がすべて落ちた。自己点検 clean
  - 748ef386(inline、1 ファイルのテストの直し): doctor の環境のテストを 9 つの変数を確かめる形に直して改名した(S7 の申し送り)。`go test -run TestDoctorNodeToolchain_ProbeEnvForcesTheNineVariables` が通る。自己点検 5 項目 clean
  - test(cycle 1、432e51ad): pass。1 回目は負荷の高い手元で internal/cli と internal/org が 30 分で時間切れになり、上限を 60 分にした 2 回目で通った。internal/cli の gate・regate・merge のテストが本物の probe を通ることは tech-debt に書いた
  - sync-docs(cycle 1): README の gate の段落(`TOOLS:` の行、`tool_dirs` と `tool_<ツール>`、doctor の「Node toolchain」)、仕様 FR-7 の 2026-11-13 の注、手順書の #308 の節から新しい節への 1 文(2 つの写し)、`docs/quality/quality-gates.md` の gate の段落(2 つの写し)、`doctor.go` の `runDoctorOpts` の注を直した。tech-debt は、Trigger に当たった 15 行に持ち越しの note を足し、新しい 3 行(文字を濾す 3 か所、`doctor_node_test.go` のサイズ、internal/cli のテストが本物の probe を通ること)を足した。report は `docs/reports/sync-docs-2026-11-13-gate-tool-versions.md`
  - sync-docs で残した小さな点 2 つ(どちらも tech-debt には書かない): (1) doctor の経路では `ProbeEnv` が 2 回当たる。冪等なので害は無い。(2) 最初の Ctrl-C のあと doctor が probe の process group を止めている間、2 回目の Ctrl-C も `stop()` が呼ばれるまで受け取られる
  - cross-review(cycle 1、0abfb274): codex(gpt-6-astra、xhigh)の P2 が 2 件で、どちらも ACTION_REQUIRED。(1) login shell の起動ファイルが 9 つの変数を後から上書きしうる。(2) `command -v` が alias を報告しても、`"$rtc_t"` は alias を展開せずに PATH のファイルを打つので、記録と実行がずれる(偽の pass になりうる)。consult(consult-plan-xreviewfix)の判定「進めてよい」に従い、(2) は「打たずに知らせる」案を自律で選んだ。alias を eval で打つ案は、alias の中身(`mise exec` など)を doctor が実行し、mise の `exec_auto_install`(既定 true)で入っていない版を入れうる。そのため 10 個目の変数が要り、D7 と範囲の両方で重い。cycle を 2 にした
  - 41f722d1: origin/main(e902705c、#312a の PR #327)を merge した。AGENTS.md の internal/org の行と tech-debt の 4 か所がぶつかり、両方の変更を残して解いた
  - 7c0bf678(implementer、sonnet。opus の implementer は編集の前に止まったので止めて出し直した): スクリプトの頭で 9 つの変数の大文字と小文字の綴りを `unset` し、9 つを `export` し直す(`probeEnvLines`、`probeEnvOverrides` から作る)。`command -v` が絶対パスを出さないとき(alias、関数、builtin)は `noexec` の記録だけを出して何も打たず、`ParseProbe` は unknown にして Reason で alias とそれ以外を分ける。doctor の warn には「login shell でファイルに解決させる」の一言を足した。検証コマンド rc=0、変異 2 種でそれぞれ新しいテストが落ちた。自己点検 clean。残した点: PATH に相対パスがあると `command -v` が `./node` を出し、Reason が「関数か builtin」になる。`internal/toolchain/doc.go` の「版の命令だけを打つ」に「ファイルのときだけ」を足していない(cycle 2 の sync-docs に渡す)
  - self-review(cycle 2、a1a396f6): CRITICAL・HIGH・MEDIUM 0、LOW 3、Merge: yes。LOW 1(`probeFunc` と `notFileReason` の注が実際と違う。`"$rtc_t"` は alias を展開しないので、alias を打てば PATH のファイルが打たれて記録は alias を指す。ksh93 は alias の値だけを出す)は sync-docs でコードの注だけを直す。LOW 2(「打たなかった」を `Path` の形から組み直す判定が 4 か所)と LOW 3(`TestProbe_AliasOrFunctionInStartupFileIsNotRun` の表の読みにくさ)は tech-debt に書く
  - verify(cycle 2、d1ca8cd6): pass、LOW 4(V7〜V10)。V7 `doctor_node_test.go` が 996 → 1016 行で tech-debt 495 行目の Trigger を引いた。V8 `probe_test.go` が 909 行でサイズの行が無い。V9 doc.go、手順書、tech-debt 492 行目が直しのあとの probe を書き切っていない。V10 PATH の相対の項目の node が unknown になり、Reason と hint が事実と違う(直さず tech-debt に書く)。実際の login shell で alias・関数・相対 PATH は warn、合う node と pnpm は pass
  - 34936129: origin/main(bbe2807d、#328 の PR。#319 の bracketed paste)を 3 回目に merge した。AGENTS.md と tech-debt がぶつかり、両方の変更を残して解いた
  - test(cycle 2、12a9d1e3): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` を 17:08〜17:48 JST(負荷 14〜66)に回した。cli 1452 s、org 1559 s、toolchain 16.2 s、config 57.8 s。直しの新しいテスト関数 5 本は通り、取り込んだ #327・#328 のテストも一緒に通った
  - sync-docs(cycle 2): コードの注だけ直した(`probe.go` の `probeFunc` と `notFileReason`、`probe_test.go` の 1 か所。alias は PATH のファイルを打つことになる、ksh93 は alias の値だけを出す)。`internal/toolchain/doc.go` に「絶対パスのときだけ打つ」と「9 つの変数をスクリプトの頭でも置く」を足し、手順書(2 つの写し)の「Version managers (mise, asdf)」に 2 文を足した。tech-debt は、`doctor_node_test.go` の行(495 行目)の数と持ち越しの note、415 行目の `doctor_node.go` の数、492 行目の混在した綴り、371 行目の note を直し、新しい 2 行(`probe_test.go` のサイズ、cycle 2 の LOW 3 件)を足した。report は `docs/reports/sync-docs-2026-11-13-gate-tool-versions.md`(cycle 2 を先頭に、cycle 1 は履歴の節)。README、help、AGENTS.md の Repo map、仕様 FR-7 の注は、偽になった文が無いので変えていない
  - sync-docs(cycle 2)の report と insight event: doc-maintainer が文書の commit(f697bf0d)のあと、report を書く前に通信エラー(ENOTFOUND)で止まった。2026-11-15 に orchestrator が inline で書いた(e45aa6ab)
  - cross-review(cycle 2、ca14df41): 1 回目はセッションの再開で `-o` を書く前に止まり(git の状態は変わらず)、cycle を数えずに打ち直した。codex の P2 が 1 件で ACTION_REQUIRED。doctor は `$SHELL` として ksh を受けるが、macOS の ksh93 は値が 1 つの絶対パスの alias を `command -v` でそのパスとして返し、確かめを通る。打つのは名前(`"$rtc_t"`)なので PATH のファイルが打たれ、記録と版がずれて偽の一致になりうる。consult(consult-plan-kshalias)の判定「進めてよい(案 b)」に従い、確かめを通ったパス(`"$rtc_p"`)を打つ案を自律で選んだ。ksh を `/bin/sh` の fallback に回す案は、座席が ksh なら別の shell で測ることになり、手元に無い mksh・yash の形も確かめられないので採らなかった。上限(2)に届いたので、ユーザーの指示どおり `cycle-count.json` の `cap` を 3 にした(cycle は 2 のまま)
  - 手元の確かめ(案 b の前提): node、pnpm、python3、go、npm と、mise の shim(mise 本体への symlink)の node と python3 は、`command -v` のパスで打っても名前で打っても同じ版を返した
  - dddd83bb(implementer、sonnet): `rtc_tool` が `"$rtc_p" "$@"` を打つ。注(`probeFunc`、`BuildScript`、`notFileReason`、doc.go)を合わせた。テスト: bash の `BASH_ENV` で `command -v node` に node22 のパスを返す関数を置く場面(v22 と node22 のパス、node22 だけが打たれる)、`/bin/ksh` があるときの ksh93 の実物(値が 1 つの絶対パスの alias は v22、引数つきの alias は not-a-file の unknown で何も打たない)。検証コマンド rc=0、`"$rtc_t"` に戻す変異で新しいテストが落ちた。自己点検 clean。残した点: 引数つきの ksh の alias の Reason が「関数か builtin」になる(tech-debt の cycle 2 の LOW の行に近い件)
  - self-review(3 回目、afb8b78c): CRITICAL・HIGH・MEDIUM 0、LOW 4(R3-1〜R3-4)、Merge: yes。R3-1 と R3-2 は注と手順書の文が ksh93 の alias(値が 1 つの絶対パスなら打たれる)と shell の数(doctor は 7 つを受ける)に合わない件、R3-3 と R3-4 は tech-debt の cycle 2 の行が ksh93 の新しいテストの表と「Why deferred」の cap(2)を挙げない件
  - verify(3 回目、fc83653d): pass、LOW 1(V11、tech-debt 497 行目の数と持ち越しの note)。AC1〜AC10 は `dddd83bb` のあとも満たす。実機の `/bin/ksh`(ksh93)で、直す前の binary と直したあとの binary を並べて確かめた(alias が 1 つの絶対パスなら、直す前は PATH の node が打たれて記録とずれ、直したあとは記録した path が打たれる。引数つきの alias は両方とも打たれず warn)
  - test(3 回目、346f7553): pass。2026-11-15 20:06〜20:37 JST、cli 824 s、org 1256 s、toolchain 26.3 s
  - sync-docs(3 回目): コードの注だけ直した(`internal/toolchain/doc.go`、`probe.go` の `probeFunc`・`BuildScript`・`notFileReason`)。alias は打たないという文に「ksh93 は値が 1 つの絶対パスの alias をそのパスで出し、そのパスが打たれる」を足し、`BuildScript` と `notFileReason` の shell の数を doctor の 7 つに合わせた(sh と ksh93 までは実機で走らせ、mksh と yash は未確認)。手順書(2 つの写し)の「Version managers (mise, asdf)」に ksh93 の 1 文を足して段落を折り返し直した。tech-debt は 497 行目の数(1032)と分け方の見積もり(約 325 行)と持ち越しの note、498 行目の (a)・(c)・Why deferred・Trigger の持ち越しの note、371 行目の手順書の note を直した。report は `docs/reports/sync-docs-2026-11-13-gate-tool-versions.md` の Run 3
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created(#332)
