# gate で走ったツールの版を残し、doctor が repo の求める Node の版と座席の shell の版を比べる(#311)

- Status: Approved
- Approved: 2026-11-13 sha256:df776faef33e
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
- D7 測ることで何も変えない(計画助言の HIGH): gate の probe と doctor の shell には、ほかの変数の上に `MISE_AUTO_INSTALL=0`、`MISE_NOT_FOUND_AUTO_INSTALL=0`、`COREPACK_ENABLE_AUTO_PIN=0`、`COREPACK_ENABLE_NETWORK=0`、`COREPACK_ENABLE_DOWNLOAD_PROMPT=0`、`NPM_CONFIG_UPDATE_NOTIFIER=false`、`GOTOOLCHAIN=local` を置く(呼ぶ側の値より後ろに置き、上書きする)。`GOTOOLCHAIN=local` は、`go version` が go.mod の toolchain を取りに行かないため(S1 の実装で分かった。取りに行く先は worktree の外の module cache で、前後の比べでは見えない)。入っていない版やダウンロードの要る版は測れず、`unknown` になる(go.mod が手元より新しい go を求めれば、そのことが理由に出る)。scripts に渡す環境は変えない。gate は probe の前後で `git status --porcelain --untracked-files=all` を比べ、変わっていれば log と stderr にその行を書く(そのあとの scripts の結果は今までどおり `tree_changed` で記録される)。
- D8 共有の置き場所: 測る場所の一覧、スクリプトと環境、出力の読み取り、npm の範囲は、gate(internal/org)と doctor(internal/cli)が使うので、どちらにも属さない新しいパッケージ `internal/toolchain` に置く。gate と doctor が同じ一覧を使うことを、パッケージのテストで固定する。

## Acceptance criteria

- [ ] AC1 gate(と local の regate)の log の見出しに、scripts の前の `# tool <名前> (<場所>): <版> (<解決先>)` の行が、測った場所とツールごとに出る(PATH に無ければ `missing`、読めなければ `unknown (<理由>)`)。見出しに「scripts の子は shim で別の版を見うる」の 1 行があり、解決先が shim なら `(mise shim)` か `(asdf shim)` が付き、mise が PATH にあれば `# mise current:` の行が出る(issue の受け入れ条件 1)
- [ ] AC2 `gate_passed` と `gate_failed` の Details に、測ったツールの `tool_<名前>=<版>` と `tool_dirs=<n>` が `log=` の前に入り、`gateRecordFromDetails` が読める。空白を含む出力や任意の文は値にならず(`unknown`)、場所によって版が違えば `mixed` になり、記録が読めなくなることはない。`tool_*` の無い前の形の Details も読める
- [ ] AC3 測る場所の一覧: root だけの repo、root に印が無く `apps/a` が pnpm・`apps/b` が yarn の monorepo、サブディレクトリだけに版のファイルがある repo、pnpm の workspace(lockfile は root、package.json はサブ)で、場所とパッケージマネージャを正しく見分ける。上限を超えた分は log に数が出る。python3 はいつも、go は `go.mod` があれば root で測る。`git ls-files` が失敗すれば python3 だけを測り、log にそう書く
- [ ] AC4 probe は 1 回の sh で、全部で 15 秒を超えない。時間切れと失敗では gate の verdict は変わらず、測れなかったツールが `unknown` になる。shim の子が stdout を握っても 15 秒と猶予のうちに戻る。ctx の終わりでは probe も止まり、何も記録しない
- [ ] AC5 `ralph org gate` と `ralph org regate` の出力に `TOOLS:` の行が出る(scripts が走ったとき)。reviewer の役割 prompt、`/org` skill、director skill(4 つの写し)が `TOOLS:` を扱い、それを固定するテストが通る
- [ ] AC6 `ralph doctor` の Check「Node toolchain」: JS の場所が無い repo では shell を打たずに info。場所ごとの `.node-version`・`.nvmrc`・`engines.node`・`packageManager` と、`$SHELL -lc` で解決した node とパッケージマネージャの版が合わなければ warn(issue の受け入れ条件 2)。合えば pass。別名や読めない範囲は info。shell の失敗・時間切れは warn で、exit code は変わらない。`$SHELL` が空なら `/bin/sh` を使い、そう書く
- [ ] AC7 npm の範囲の読み取りが、形ごとの表のテスト(完全な版、部分、`^`、`~`、比較演算子、AND、`||`、ハイフン、`x`、読めない形)で確かめられる
- [ ] AC8 手順書の「Shell aliases」の次に mise・asdf の節があり、shim が子の PATH を書き換えること、`MISE_<TOOL>_VERSION` での固定、`mise current` と gate の log の `# tool` の行での確かめ方、#308 の見込みの原因(未確認)を書く
- [ ] AC9 `ralph org gate --help`、`ralph org regate --help`、`ralph doctor --help`、AGENTS.md の Repo map、tech-debt がこの挙動に合う。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [ ] AC10 測ることで何も変えない: gate の probe と doctor の shell の環境に D7 の 7 つの変数が入り、呼ぶ側の `COREPACK_ENABLE_AUTO_PIN=1` などを上書きする(テストで確かめる)。偽の shim(呼ばれると worktree にファイルを書き、install の印を残す)を PATH に置いた試験で、変数が効けば何も書かれず、効かない偽の shim では gate が probe の前後の違いを log と stderr に出す。scripts に渡す環境は変わらない

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
  - 環境: D7 の 7 つの変数が呼ぶ側の値を上書きすること
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
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
