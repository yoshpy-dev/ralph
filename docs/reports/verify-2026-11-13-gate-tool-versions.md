# Verify report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311、承認の digest `2774af0aab07`。HEAD で計算し直しても同じ)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン cycle 2。cycle 1 の記録は末尾の「Cycle 1 の経過」に短くまとめた(V1〜V6 の ID は変えていない)
- Scope: ブランチ全体、`git diff e902705c...HEAD`(HEAD `a1a396f6`。`e902705c` は `41f722d1` が取り込んだ origin/main の先頭で #327)。cycle 1 の verify(`a6657d48`)からの差は、S7 `e7274fec`(V1 の直し)、`748ef386`、cycle 1 の test・sync-docs・cross-review の記録、merge 2 回(`349f5763`、`41f722d1`)、cross-review の直し `7c0bf678`、計画の記録 `e45f7fe3`、cycle 2 の self-review `a1a396f6`。重点は `7c0bf678`(`internal/toolchain/probe.go`、`internal/cli/doctor_node.go` とそのテスト)。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-13-gate-tool-versions.log`(gitignore 済み。cycle 1 の内容の後ろに cycle 2 を足した。run-static-verify の全出力、scratch の probe 6 本(P1〜P6)の出力、実機の login shell の観察 P7 の写し)

## Cycle 2 の結論

verdict は pass。MEDIUM 以上はなく、LOW 4 件(V7〜V10)は文書と記録の話で、挙動は直さなくても通る。

`7c0bf678` の 2 つの直しは、実際の login shell(zsh と bash 3.2)で意図どおりに効いた。

- 起動ファイルが 9 つの変数を戻しても、スクリプトの頭の `unset` と `export` が 0 と `local` と `false` に戻す。`.zprofile` が `COREPACK_ENABLE_AUTO_PIN=1`、`GOTOOLCHAIN=auto`、`pnpm_config_manage_package_manager_versions=true`、`PNPM_CONFIG_MANAGE_PACKAGE_MANAGER_VERSIONS=true`、`MISE_AUTO_INSTALL=1` を export する zsh で、素の login shell は `AUTO_PIN=1 GOTOOLCHAIN=auto` を見る(対照)。doctor が打つ偽の node は `0|local|false||0`(大文字の綴りは unset)、偽の pnpm は `0|local|false|false` を見た。bash 3.2 の `.bash_profile` でも同じ
- `command -v` が絶対パスを出さない node(alias、関数、`relbin/node` の相対 PATH)は打たれず、doctor は「could not read the version of node (...)」の warn に直し方の一言を付ける。alias の中身も関数の中身(`func-ran` の印)も動かなかった

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V7 | LOW | cycle 2 の直し `7c0bf678` が `internal/cli/doctor_node_test.go` を +20 行(996 → 1016)にして、tech-debt の「次の変更で分ける」Trigger を引いた。持ち越しの note が無く、数も古い。`internal/cli/doctor_node.go` は 691 → 699 行で、415 行目の note(「691 lines」)も古い | 495 行目の Trigger 欄は「The next change to `internal/cli/doctor_node_test.go`」で、本文は「996 lines ... last changed by `748ef386`」「`doctor_node.go` is 691 lines」。`wc -l`(HEAD): 1016 と 699。直しの diff に `internal/cli/doctor_node_test.go` がある。持ち越しの note が 415 行目にあるのは doctor.go の 1648 行の件 | /sync-docs で 495 行目の数と「last changed by」を直し、「2026-11-13: `7c0bf678` hit this trigger ... did not pay it (a move-only split restarts the pipeline from /self-review, and the cap is 2)」の note を足す。415 行目の「691 lines」を 699 に |
| V8 | LOW | `internal/toolchain/probe_test.go` はこのブランチの新しいファイルで、909 行になり 800 行の目安を超えた。cycle 1 の verify の時は 648 行、S7 の後で 719 行、直しで +190 行。サイズの行が無い | `wc -l`(HEAD 909)、`git show a6657d48:internal/toolchain/probe_test.go \| wc -l`(648)、`git show 7c0bf678^:...`(719)。`grep -n probe_test docs/tech-debt/README.md` は空 | /sync-docs でサイズの行を足す(495 行目と同じ型。新しいテスト 4 つとシェルの helper を `probe_startup_test.go` に分ける案)。分けるなら pipeline の再走が要るので、行を足すだけが無理のない形 |
| V9 | LOW | 直しのあと、probe の説明がいくつか足りない。(a) `internal/toolchain/doc.go` 15-21 行は「The probe script itself installs nothing and writes no file: it runs only `command -v` and each tool's version command」「in an environment whose last entries (ProbeEnv)」。今は、version command は `command -v` が絶対パスを出したときだけ打ち、9 つの変数はスクリプトの頭でもう 1 回置く(`probe.go:116`、`:150-159`)。(b) 手順書の「Version managers (mise, asdf)」(2 写し、`:335-:360`)と tech-debt の 492 行目は、変数が起動ファイルの後でも効くことと、alias・関数・相対 PATH の node が `unknown` で doctor が warn することを書かない。492 行目は、スクリプトの頭が混在した綴り(`Pnpm_Config_...`)を unset しない限りも書かない(`probe.go:112` の注釈にはある) | doc.go の該当行。`grep -n alias docs/recipes/isolated-herdr-server.md` は「Shell aliases」の節(codex の alias)のみ | /sync-docs で doc.go の 2 文を直し、手順書(2 写し)の doctor の箇条か変数の段落に、「login shell の起動ファイルが後から変数を戻しても、スクリプトの頭が戻す」「`$SHELL -lc` が node や pnpm を alias・関数・builtin(と相対 PATH の項目)で解決したら、打たずに版が読めない warn にする。PATH 上のファイルにする」の 2 文を足す。492 行目に混在した綴りを足す。仕様 FR-7 の 2026-11-13 の注(`docs/specs/2026-10-07-org-multi-org-director.md:141`)は誤りではないので任意 |
| V10 | LOW | PATH に相対の項目があると、`command -v` は `relbin/node` を出し、probe は打たずに `unknown` にする。Reason は「not a file (a shell function or builtin)」、hint は「(an alias or function there is not measured)」で、どちらも相対パスの場合は事実と違う。直す前は、そのファイルを打って版を測っていた(挙動の変更)。この件と、cycle 2 の self-review の LOW 1(alias の説明が実際と違う。`probe.go` の `probeFunc` の注と `notFileReason` の注)・LOW 2(「打たなかった」を `Path` の形から組み直す判定が 4 か所。1 行に同じことが 3 回出る)・LOW 3(テストの表の読みにくさ)は、どこにも tech-debt の行が無い。計画の Progress に書いてあるだけで、計画は /pr で archive される | 実際の login zsh で P4(`relbin` が PATH の先頭、`relbin/node` は repo 内のファイル)を打つと「could not read the version of node (relbin/node): not a file (a shell function or builtin), not run ...; make node resolve to a file ... (an alias or function there is not measured)」。印は `node-env` が無く、打たれていない。self-review の cycle 2 の Findings と「Tech debt identified」 | 直すなら、self-review の LOW 2 の案(`Measurement` に「打たなかった」の欄を足し、相対パスは `strings.Contains(path, "/")` で分ける)で 3 件がまとまって閉じる。コードに触れるので cycle の上限(2)に当たる。直さず進めるなら、/sync-docs で tech-debt に 1 行(この 4 点、Trigger はその次の `probe.go` か `doctor_node.go` の変更)。現実には、login shell の PATH に相対の項目を置く設定は少なく、gate の `/bin/sh -c` は起動ファイルを読まないので、害は小さい |

INFO(merge を止めない)

- I7: origin/main が `bbe2807d`(#328、send-claude-paste)まで進んだ。`git merge-tree --write-tree --name-only origin/main HEAD` は `AGENTS.md` と `docs/tech-debt/README.md` でぶつかる(前の 2 回の merge と同じ 2 つ。skill の写し 4 つは自動で解ける)。/pr の前に 3 回目の取り込みが要る(計画の R6)
- I8: 計画の D1・D6・D7 と AC10 の本文は、スクリプトの頭で変数を置き直すことと、打たないこと(noexec)を書かない。Progress の 166 行目と 7c0bf678 のコミット文にある。digest が覆う D の本文は意図して後れる扱い(cycle 1 の V1 のときと同じ)。AC10 の条件(9 つの変数が呼ぶ側の値を上書き)は、直しで強くなった向きで満たす
- I9: 計画の受け入れ条件の checkbox(AC1〜AC10)は `[ ]` のまま。/pr が付ける
- I10: cycle 1 の I1(doctor の中断が exit 1。D6・AC6 の「exit code は変わらない」は warn の話)は変わっていない。Progress の S6 の行に理由がある
- I11: この verify の観察は scratch の HOME と偽の node・pnpm で行った。実機の login shell(`/bin/zsh`、mise の shim)の P7 は、取りに行けない名前(`pnpm@0.0.1-doesnotexist`)の repo で打ち、pnpm の store の側に新しいファイルは出なかった。cycle 1 で入れた `pnpm@1.0.0` は手元の store に残っている(戻していない)

## Spec compliance

ブランチ全体を AC ごとに確かめた。cycle 1 から変わらない AC は、cycle 1 の根拠を引き、HEAD のコードで読み直して食い違いが無いことを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 log の `# tool` の行、shim の印、`# mise current:`、見出しの 1 行(issue の受け入れ条件 1) | 満たす | `gateToolsLogLines`(`internal/org/gate_tools.go`)が説明の 1 行、場所とツールごとの `# tool <tool> (<dir>): <版> (<path>)`、`missing`、`unknown (<理由>)`、`(mise shim)`・`(asdf shim)`、`# mise current:` を書く。`runGateScripts` が `# load at start:` の後、最初のスクリプトの前に呼ぶ。直しの noexec は `unknown (an alias in the shell, not run (...)) (alias node=...)` の形でこの枠に入る(括弧が重なる点は V10)。issue の「走った版」との差は cycle 1 の I6 のまま |
| AC2 記録の `tool_*` と `tool_dirs`、読み戻し、`unknown`・`mixed`、前の形 | 満たす | `gateRecord.toolDetails` は `log=` の前に書き、1 トークンでない値を `unknown` にする。noexec の測定は `StatusUnknown` なので `tool_<tool>=unknown`。`TestGateRecordDetails_Tools` は cycle 1 のとおり。この cycle の diff は記録に触れていない |
| AC3 測る場所の一覧 | 満たす | `internal/toolchain/projects.go` は cycle 1 から変わっていない(`git diff a6657d48 HEAD -- internal/toolchain/projects.go` は空) |
| AC4 1 回の sh、15 秒、時間切れで verdict が変わらない、stdout を握る子、ctx の終わり | 満たす(直しの後も) | 直しはスクリプトの頭に `unset` と `export` の 2 行を足し、`rtc_tool` に `case` を 1 つ足しただけで、sh の数(1 回)と予算(`gateToolsTimeout` 15 s)、`Run` の process group と `WaitDelay` には触れていない。noexec の時はツールを打たないので、待つ時間は増えない。時間切れ・ctx のテストは `probe_test.go` に残っている(走らせていない) |
| AC5 `TOOLS:` の行、reviewer の prompt、`/org` と director の skill(4 写し) | 満たす | `7c0bf678` はどれにも触れていない。cycle 1 の verify 以後に変わったのは、S7 の pnpm の文(`/org` skill の gate の行、gate・regate の help、手順書)と、main から取り込んだ #312a の文(`usage_limited`)。どちらも読み直して食い違いは無い。reviewer の prompt は `a6657d48` と同じ。skill の org の 3 写しと手順書の 2 写しは `cmp` で同じ。`check-skill-sync.sh` が 14 skill で通った |
| AC6 doctor の「Node toolchain」(issue の受け入れ条件 2) | 満たす(直しの後も) | `notResolved` は、`m.Path` が空でなく `/` で始まらないとき、warn の末尾に直し方を足す(`doctor_node.go:373`)。実際の login shell で、alias は warn、関数は warn、相対 PATH は warn(V10)、合う node と pnpm は pass(zsh と bash 3.2)、`$SHELL` が sh の仲間でないと `/bin/sh -lc` で測ってそう書く(P6)を観察した。`coveredByFailure` の「not measured」の接頭辞は新しい Reason と重ならない(定数 `reasonAlias`、`reasonNotFile`)。実機の `/bin/zsh -lc`(mise の shim の pnpm 12.9.1)は P7 のとおり。warn は exit code を変えない(cycle 1 の観察。中断だけ exit 1 は I10) |
| AC7 npm の範囲の表のテスト | 満たす | `npmrange.go` は cycle 1 から変わっていない |
| AC8 手順書の mise・asdf の節 | 満たす | `### Version managers (mise, asdf)` が `### Shell aliases` の直後。2 写しは `cmp` で同じ。直しで足りなくなった説明は V9 |
| AC9 help、AGENTS.md の Repo map、tech-debt、2 つの同期の検査 | 満たす | gate と regate の help は変数、15 秒、`TOOLS:`、`tool_*`、stderr の行をコードと合わせて書いている。直しに合わなくなる文は無い(help は「unknown (<why>)」と書くだけで、gate の `/bin/sh -c` は起動ファイルを読まない)。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh` は run-static-verify の中で通った。tech-debt の更新漏れは V7・V8・V10 |
| AC10 測ることで何も変えない | 満たす(直しの後は、起動ファイルの上書きにも) | 9 つの変数は `probeEnvOverrides` 1 つにあり、環境(`ProbeEnv`)とスクリプトの頭(`probeEnvLines`)の 2 経路が同じ一覧から出る。`ProbeEnv` は呼ぶ側の変数を大文字小文字を問わず消してから 9 つを置く。スクリプトの頭は 9 つの 3 つの綴りを `unset` して 9 つを `export` する(混在した綴りは unset しない、注釈にある)。実機で、起動ファイルが 5 つの変数を戻しても probe の中の node と pnpm は戻された値を見ないことを確かめた(P1、P5)。scripts に渡す環境は `gateScriptEnv` のままで、probe の `sh` だけが `unset`・`export` する。偽の shim の 2 つの試験は cycle 1 のとおり。V1 の直し(S7)で、`packageManager` が取れない版を指す repo の実機の doctor は pnpm 12.9.1 を返し、pnpm の store に新しいファイルは出なかった(P7。cycle 1 は同じ操作で `pnpm@1.0.0` が入った) |

### 計画の決定(D1、D6、D7)と直しの関係

- D1(1 回の sh、scripts と同じ環境と cwd、`command -v` と版): 満たす。直しは `command -v` が絶対パスを出したときだけ版を打つ。D1 は解決先を測ることを言っていて、alias を打つことは言っていない
- D6(doctor、`$SHELL -lc`、warn): 満たす。alias・関数・builtin は「版が読めない」warn の枠に入り、exit code は変わらない(P2、P3)
- D7(測ることで何も変えない): 満たす。スクリプトの頭で置き直すので、login shell の起動ファイルが後から戻しても効く。D7 の本文に書かれていない追加の層は I8

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | 通過(rc=0) | shellcheck、hook の `sh -n` と awk、`jq -e`、Codex の 3 つの guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(gofmt ok、`0 issues.`)、branch の secret scan(`e902705c..a1a396f6`、clean)。raw は evidence に |
| `git diff --check e902705c...HEAD` | rc=0 | |
| `gofmt -l internal/toolchain internal/org internal/cli` | 空 | |
| `./scripts/plan-visual.sh digest <plan>` | `2774af0aab07` | 計画の `- Approved:` の行と同じ |
| `cmp` で skill の org の 3 写し、手順書の 2 写し | 同じ | |
| `git ls-remote origin` | main `bbe2807d`、ブランチ `a1a396f6` | main は取り込み後に進んだ(I7) |
| `git merge-tree --write-tree --name-only origin/main HEAD` | `AGENTS.md`、`docs/tech-debt/README.md` で衝突 | I7 |

## Documentation drift

/sync-docs へ渡す一覧。直しが作ったずれ(V7〜V9)と、cycle 1 の指摘の閉じ具合を分けた。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `internal/toolchain/doc.go` 15-21 行 | ずれた(V9) | 「it runs only `command -v` and each tool's version command」は「絶対パスの時だけ」を足す。「last entries (ProbeEnv)」はスクリプトの頭で置き直すことを足す。実装者が申し送った件 |
| `internal/toolchain/probe.go` の注(`probeFunc`、`notFileReason`) | ずれた(V10) | alias の中身は動かない(`"$rtc_t"` は alias を展開しない)。「every shell prints an alias as `alias <name>=...`」は ksh93 を除く。self-review の LOW 1。コードの注なので扱いは lead が決める |
| `docs/recipes/isolated-herdr-server.md` と templates/base の写し | 足りない(V9) | 変数が起動ファイルの後でも効くこと。alias・関数・builtin・相対 PATH の node は打たれず doctor が warn すること |
| `ralph org gate --help`、`ralph org regate --help` | 合う | 変数と `unknown (<why>)` の文は直しと食い違わない。gate の `/bin/sh -c` は起動ファイルを読まないので、書き足しは任意 |
| `ralph doctor --help`、`doctor.go` の注 | 合う | Check の一覧が無く、`runDoctorOpts` の注は Check 14 の subprocess を書いている |
| README.md、`docs/quality/quality-gates.md`、仕様 FR-7 の 2026-11-13 の注 | 合う | 「9 つの変数を置き」は今も正しい。置き直しと noexec の追記は任意 |
| reviewer の役割 prompt、`/org`・director の skill(4 写し) | 合う | この cycle で触れていない |
| AGENTS.md の Repo map | 合う | |
| `docs/tech-debt/README.md` | ずれた(V7、V8、V10) | 495 行目の数と Trigger の note、`probe_test.go` のサイズの行、492 行目の混在した綴り、V10 の 1 行。cycle 1 の V5 の 13 行は持ち越しの note がある(下の表) |
| doctor の Check の並び | 合う | Check 3、3b … 13、14。この cycle で変わっていない |

## Observational checks

HEAD から `go build ./cmd/ralph` で作った binary を、scratch の HOME と git repo(`.node-version` の 99、`engines.node` の `>=99`、`packageManager: pnpm@99.0.0`、`pnpm-lock.yaml`)で `ralph doctor --state-dir <存在しないパス>` として打った。node と pnpm は PATH の先頭の偽の実行ファイルで、呼ばれると自分の環境を印のファイルに書く。`/etc/zshenv` が `ZDOTDIR=$HOME/.config/zsh` を置くので、zsh の起動ファイルは `$HOME/.config/zsh/.zprofile` に置いた。raw は evidence の P1〜P6。

| 場面 | 結果 |
| --- | --- |
| P1 zsh、`.zprofile` が 5 つの変数を戻す | 対照(素の login zsh)は `AUTO_PIN=1 GOTOOLCHAIN=auto`。doctor は pass。偽の node は `0\|local\|false\|\|0`、偽の pnpm は `0\|local\|false\|false` を見た |
| P2 zsh、`alias node=<偽の node>` | warn(「alias node=... : an alias in the shell, not run」+ 直し方)。偽の node は打たれなかった(`node-env` の印なし) |
| P3 zsh、`node() { ... }` の関数 | warn(「could not read the version of node (node): not a file (a shell function or builtin), not run ...」+ 直し方)。関数は動かなかった(`func-ran` なし) |
| P4 zsh、PATH の先頭が相対の `relbin` | warn(「relbin/node: not a file (a shell function or builtin)」)。打たれなかった。V10 |
| P5 bash 3.2、`.bash_profile` が 5 つの変数を戻す | 対照は `AUTO_PIN=1 GOTOOLCHAIN=auto`。doctor は pass。偽の node は `0\|local\|false\|\|0`、偽の pnpm は `0\|local\|false\|false` |
| P6 `$SHELL` が `/opt/none/fish` | `/bin/sh -lc` に切り替え、Detail に「$SHELL ... is not an sh-compatible shell」と書く。warn は 2 件(node と pnpm が PATH に無い) |
| P7 実機の login zsh、mise の shim の pnpm 12.9.1、`packageManager: pnpm@0.0.1-doesnotexist` | warn「resolves pnpm 12.9.1 (.../mise/shims/pnpm, a mise shim); run corepack enable ... MISE_PNPM_VERSION=0.0.1-doesnotexist」。pnpm の store(`~/.local/share/pnpm`、`~/Library/pnpm`、`~/.cache/pnpm`)に、打つ前の印より新しいファイルは無い。`~/.local/share/mise` はディレクトリの mtime だけが動いた(新しい項目はなく、ほかのセッションも mise を使っている) |

## Coverage gaps

- テストの実行結果(`/test`)。上の AC の根拠は、テストのソースとコードを読んだものと、上の観察
- gate 側の `/bin/sh -c` で、起動ファイルの上書きを直接観察していない。同じスクリプトを doctor 経由で zsh と bash 3.2 に打った。gate の sh は起動ファイルを読まないので、頭の置き直しが効くかどうかは gate では実質試される場面が少ない
- ksh93 と dash で `command -v` が alias を出す形(self-review が ksh93 の違いを見つけた)。doctor が対応する shell は sh、dash、bash、zsh、ksh、mksh、yash だが、観察は zsh と bash 3.2 だけ
- 実機の座席(codex の shell)での gate。tech-debt 491 行目のまま

足すと確かさがいちばん上がる検査: `TestProbe_AliasOrFunctionInStartupFileIsNotRun` の表に、PATH の相対の項目の行を足す。V10 を直すか tech-debt に書くかを決める材料になる。

## Verdict

- Verified: AC1〜AC10 は、ブランチ全体で、コードとテストのソース、`run-static-verify.sh` の通過(rc=0)、実際の login shell(zsh と bash 3.2)の観察から満たす。cross-review の 2 件の直しは意図どおりに効いた(起動ファイルが戻した 5 つの変数が probe の中で 0、`local`、`false` になる。alias・関数・相対 PATH の node は打たれず、doctor が warn する)。V1(pnpm 自身の版の切り替え)は S7 で直った(P7)。issue #311 の受け入れ条件 1・2 は AC1・AC6 に、計画助言の 2 件は D2・D7 と AC3・AC10 に入っている
- Partially verified: gate の `/bin/sh -c` での置き直し(doctor 経由の同じスクリプトで観察)。ksh93 などの shell
- Not verified: テストの実行結果。実機の座席での gate。mise と corepack の変数が実際に取得と pin を止めること(計画の Assumptions は文書による。pnpm は P7 と cycle 1 の観察)
- Drift(/sync-docs へ): V7 tech-debt 495・415 行目の数と Trigger の note、V8 `probe_test.go` のサイズの行、V9 doc.go の 2 文と手順書(2 写し)と tech-debt 492 行目、V10 の tech-debt の 1 行(直さない場合)。任意: gate・regate の help、仕様 FR-7 の注
- Verdict: pass(CRITICAL・HIGH・MEDIUM なし、LOW 4 件)

## Cycle 1 の経過

cycle 1 の verify(HEAD `a6657d48`)は pass で、MEDIUM 1(V1)、LOW 5(V2〜V6)だった。cycle 2 の時点の状態:

| ID | cycle 1 の指摘 | 状態 |
| --- | --- | --- |
| V1(MEDIUM) | pnpm 自身の版の切り替えが 7 つの変数で止まらず、worktree の外に pnpm を入れる | 直った(S7 `e7274fec`)。`pnpm_config_manage_package_manager_versions=false` と `npm_config_...` を足し、9 つにした。ProbeEnv は大文字の綴りも消す。実機の doctor が取れない版名の repo で pnpm 12.9.1 を返し、store に新しいファイルが出ないことを確かめた(P7) |
| V2 | README.md の gate の段落に `TOOLS:` と `tool_*` が無い | 直った(README.md:291) |
| V3 | 仕様 FR-7 に 2026-11-13 の注が無い | 直った(`docs/specs/2026-10-07-org-multi-org-director.md:141`) |
| V4 | `runDoctorOpts` の注が Check 14 の subprocess を書かない | 直った(`doctor.go:76-84`) |
| V5 | tech-debt の Trigger を引いた 13 行に持ち越しの note が無い | 直った。13 行(159、180、238、293、294、367、371、372、390、398、405、415、464)に「feat/gate-tool-versions」の note がある |
| V6 | `doctor_node_test.go` のサイズの行が無い | 行は足された(495 行目)。cycle 2 の直しで数が古くなり、Trigger を引いた(V7) |

cycle 1 の INFO のうち、I3(手順書の箇条と `/org` skill の前提が Node toolchain を挙げない)は、手順書(`:335`)と `/org` skill(`:1666`)、director skill(`:1317`)が挙げるようになった。I4(文字の濾し処理が 3 か所)は tech-debt の 494 行目にある。I1、I2、I5、I6 は変わらない。
