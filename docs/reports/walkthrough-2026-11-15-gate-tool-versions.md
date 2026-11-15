# Walkthrough: gate で走ったツールの版を残し、doctor が Node の版を比べる(gate-tool-versions)

- 計画: `docs/plans/archive/2026-11-12-gate-tool-versions.md`(PR の作成のあとに archive へ移す)
- issue: #311。PR #332
- 差分: `git diff origin/main...HEAD`(62 ファイル、約 +8,420 / −320。`internal/` の Go の本体は約 +3,170、Go のテストは約 +3,950、残りは skill・手順書・文書・計画・報告)

## 1. 共有のパッケージ `internal/toolchain`(読む価値が一番高い)

gate(internal/org)と doctor(internal/cli)の両方が使うので、どちらにも属さない新しいパッケージに置いた(計画の D8)。

- `projects.go`: 測る場所の一覧。`FindLayout` が worktree の `git ls-files` を `LsFilesTimeout`(10 秒)の中で読み、`LayoutFromFiles` が場所を決める。場所は、lockfile(`pnpm-lock.yaml`、`yarn.lock`、`package-lock.json`)か版のファイル(`.node-version`、`.nvmrc`、`.tool-versions`、`mise.toml`、`.mise.toml`)のあるディレクトリと、JS の印が 1 つでもあれば root。上限は `MaxPlaces`(10)で、超えた分は数だけ残す。場所ごとのパッケージマネージャは `placeManager` が、近い lockfile、その場所の `packageManager`、root の `packageManager`、npm の順に決める。`Targets` は gate 用(node、パッケージマネージャ、python3、`go.mod` があれば go)、`NodeTargets` は doctor 用(node とパッケージマネージャ)。
- `probe.go`: 測るスクリプトと読み取り。`BuildScript` は 1 回の shell で場所ごとに subshell を作って絶対パスで `cd` し、`rtc_tool` が `command -v` と版の命令を打って、`ralph-toolchain` で始まる記録の行を出す。`ParseProbe` がその記録を `Measurement`(found、missing、unknown と理由)に読む。`Probe` は `Run` で 1 回だけ打つ。
- `probe.go` の環境: `ProbeEnv` は呼ぶ側の環境から 9 つの変数を大文字小文字を問わず消し、止める値で置き直す(mise の自動インストール 2 つ、corepack の自動 pin・取得・問い合わせ、npm の更新の確認、`GOTOOLCHAIN=local`、pnpm の版の切り替え 2 つ)。cross-review の cycle 1 の直し(7c0bf678)で、スクリプトの頭でも `probeEnvLines` が同じ 9 つを unset と export で置き直すようになった。login shell の起動ファイルが値を戻しても効く。
- `probe.go` の打たない場合と打つもの: 同じ直しで、`command -v` が絶対パスを出さないとき(alias、関数、builtin)は `noexec` の記録だけを出して何も打たない。`ParseProbe` は unknown にして、理由で alias とそれ以外を分ける。cross-review の cycle 2 の直し(dddd83bb)で、打つのは名前ではなく `command -v` が出したパス(`"$rtc_p"`)になった。macOS の ksh93 は、値が 1 つの絶対パスの alias をそのパスとして返すので、名前で打つと記録と違うファイルが打たれていた。今はどの shell でも、記録した解決先と打つものが同じ文字列になる。
- `run.go`: `Run` は shell を自分の process group で走らせ、時間切れと ctx の終わりで group ごと止める。shim の子が stdout を握っても `KillGrace` のうちに戻る。出力は `MaxOutputBytes`(1 MiB)で切る。
- `version.go`: `VersionToken` が出力から版のトークンを取り出す(`Python 3.12.1` は `3.12.1`、`go version go1.27.3 darwin/arm64` は `go1.27.3`)。`ShimOf` は解決先が mise か asdf の shims の下かを見る。
- `npmrange.go`: doctor が `engines.node` を比べるための npm の範囲の部分集合(比較演算子、`^`、`~`、`x`、AND、`||`、ハイフン)。`MatchVersionSpec` は `.node-version` と `.nvmrc` の接頭辞の比べ方、`ParsePackageManager` は `packageManager` の読み取り。

## 2. gate と regate

- `internal/org/gate_tools.go`: `measureGateTools` が scripts の前に probe を 1 回打つ。予算は `gateToolsTimeout`(15 秒)で、shell は `gateToolsShell`(`/bin/sh -c`)、環境は scripts と同じ(`gateScriptEnv`)。log の見出しに `gateToolsLogLines` の `# tool <ツール> (<場所>): <版> (<解決先>)` の行と `# mise current:` の行を書く。probe の前後で `git status --porcelain --untracked-files=all` を比べ、変わっていれば log と stderr に書く(`gateToolsTreeLines`)。時間切れや失敗で gate の verdict は変わらず、測れなかったツールが unknown になる。ctx の終わりは割り込みと同じ扱いで、何も記録しない。
- `internal/org/gate_record.go`: `gate_passed` と `gate_failed` の Details に `tool_dirs` と `tool_<ツール>` を `log=` の前に足す。値は `gateToolValue`(版、missing、unknown)と、場所で違えば `mixed`(`gateToolsSummary`)。空白を含む値は入れないので、記録が読めなくなることはない。前の形の Details もそのまま読める。
- `internal/org/gate_run.go`、`gate.go`、`regate.go`、`regate_merge.go`: 見出しの段で `measureGateTools` を呼び、結果を `GateResult` と `RegateResult` に渡す。local の regate も同じ道を通る。
- `internal/cli/org_gate.go`、`org_regate.go`: `LOAD:` の行のあとに `printGateTools` の `TOOLS: node v24.22.1, pnpm 10.10.0, python3 3.12.1` の 1 行を出す。`mixed` があれば log を見るよう書く。

## 3. doctor の Check 14「Node toolchain」

`internal/cli/doctor_node.go`。

- `checkNodeToolchain`: 同じ場所の一覧(`NodeTargets`)が空なら shell を打たずに info の 1 行。あれば `doctorNodeShell` が選んだ shell の `-lc` で probe を 1 回打つ(`doctorNodeTimeout`、20 秒)。`$SHELL` が空や fish のときは `/bin/sh` を使い、そう書く。SIGINT などで止まれば probe の process group を止め、warn の 1 行で終える。
- 比べ方: `nodeRequirementsAt` が場所ごとの求め(近い `.node-version` と `.nvmrc`、`engines.node`、`packageManager`)を集め、`compareNode` と `checkPackageManager` が測った版と比べる。合わなければ warn(場所、欄、求め、解決した版と解決先、直し方)、`lts/*` などの別名や読めない範囲は info。warn は exit code を変えない。
- cross-review の直しで、alias や関数に解決した tool の warn に「login shell でファイルに解決させる」の一言が付く(`notResolved`)。

## 4. 座席と skill、文書

- `internal/org/prompts/reviewer.md`: reviewer は `LOAD:` と同じく `TOOLS:` を `GATE: fail` の BLOCKED の本文に写す。
- `/org` と `/director` の skill(4 つの写し): gate の失敗を読むときに `TOOLS:` と log の `# tool` の行を見る。`org_skill_send_sync_test.go` と `director_skill_sync_test.go` が文を固定する。
- `docs/recipes/isolated-herdr-server.md`(2 つの写し): 「Version managers (mise, asdf)」の節。shim が子の PATH を書き換えること、`MISE_<TOOL>_VERSION` での固定、`mise current` と gate の log での確かめ方、#308 の見込みの原因(未確認)。
- README、`docs/quality/quality-gates.md`(2 つの写し)、AGENTS.md、仕様 FR-7 の 2026-11-13 の注、`docs/tech-debt/README.md`。

## 見どころ

- 測ることで何も変えないこと(D7)は、`TestProbe_ChangesNothingWithShimsThatWouldPinAndInstall` と `TestProbe_PnpmDoesNotFetchThePackageManagerVersion` で見る。対照の実行(変数を置かない)で偽の shim が書くことを先に確かめてから、probe では書かないことを確かめる。
- 起動ファイルが変数を戻しても効くことは `TestProbe_StartupFileCannotTurnTheSafetyVariablesBackOn`、alias と関数を打たないことは `TestProbe_AliasOrFunctionInStartupFileIsNotRun`(どちらも bash の `BASH_ENV` で起動ファイルをまねる)。記録したパスを打つことは `TestProbe_RunsTheFileTheRecordNames` と、`/bin/ksh` があるときの `TestProbe_Ksh93AliasOfOneAbsolutePath`。
- 本物の probe を通す統合テストは `TestGate_RecordsTheToolVersionsWithTheRealProbe`。internal/org の他のテストは TestMain で probe を固定している。
