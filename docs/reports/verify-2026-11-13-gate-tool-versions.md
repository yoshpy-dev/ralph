# Verify report: gate-tool-versions

- Date: 2026-11-15(Run 3)。Run 1 と Run 2 は 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311、承認の digest `2774af0aab07`。HEAD で計算し直しても同じ)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 3 回目。`cycle-count.json` は cycle 2、cap 3(cap を上げても cycle は増えない)。insight event の cycle は `--cycle auto` で 2 になり、Run 2 の verify の event と同じ値になる。見分けは `ts`
- Scope: ブランチ全体、`git diff e902705c...HEAD`(HEAD `afb8b78c`)。Run 2 の verify(`d1ca8cd6`)からの差は、merge `34936129`(origin/main の #328、bracketed paste。この機能とは別の変更)、cycle 2 の test と sync-docs の記録、cross-review の 1 件(ACTION_REQUIRED)の直し `dddd83bb`(`rtc_tool` が名前ではなく `command -v` の出した path を打つ)、cycle 2 の sync-docs `f697bf0d`、3 回目の self-review `afb8b78c`。重点は `dddd83bb`(`internal/toolchain/probe.go`、`doc.go`、`probe_test.go`)。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-13-gate-tool-versions.log`(gitignore 済み。Run 2 の内容の後ろに Run 3 を足した。run-static-verify の全出力、tech-debt の Trigger の走査、merge の token 照合、観察 P8〜P12 の出力)

## Run 3 (cycle-count 2, cap 3)

### 結論

verdict は pass。CRITICAL・HIGH・MEDIUM はなく、新しい指摘は LOW 1 件(V11)だけで、tech-debt の 1 行の数と note の話。AC1〜AC10 は `dddd83bb` の後も満たす。

`dddd83bb` の効きを、実機の ksh93(`/bin/ksh`、Version AJM 93u+ 2012-08-01)で、直す前の binary(`dddd83bb^`)と HEAD の binary を並べて確かめた。`.profile` が `alias node=<node22 の path>` を置く login ksh で、`command -v node` は値の path だけを出す。直す前の doctor は「node v24.22.1 (node22 の path)」と warn し(記録は node22、打たれたのは PATH の node)、HEAD は「node v22.1.0 meets .node-version (22)」と pass し、打たれたのは node22 だけだった。引数つきの alias(`'<path> --x'`)は直す前も HEAD も打たれず warn になる。

`dddd83bb` のコードの差は 1 行(`"$rtc_t" "$@"` が `"$rtc_p" "$@"`)で、残りは注とテスト。probe の sh の数(1)、予算(15 秒)、process group、環境の 9 つの変数、record の形は変わらない。

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V11 | LOW | tech-debt 497 行目(`probe_test.go` のサイズの行)の Trigger「The next change to `internal/toolchain/probe_test.go`」を `dddd83bb` が引いたのに、持ち越しの note が無く、数と分け方の見積もりも古い。行は「911 lines」「`shellOnPath` and the four tests after it, to the end of the file, about 200 lines」と言う。HEAD は 1032 行で、`shellOnPath`(710 行目)から末尾までは 323 行、テスト 6 本とヘルパー 1 つ(`writeRecordRunFakes`)。直しで足したのは `TestProbe_RunsTheFileTheRecordNames` と `TestProbe_Ksh93AliasOfOneAbsolutePath`、ヘルパー | `wc -l`(HEAD)1032。`git show f697bf0d:internal/toolchain/probe_test.go \| wc -l` は 911、`dddd83bb` は 1032(+121)。`grep -n '^func '` の 710 行目以降は 8 つ。Trigger の走査(cell 4 を python で取り出し、`probe.go`・`probe_test`・`doc.go`・`rtc_tool`・`notFileReason`・`ParseProbe`・`noexec` に当たる行)は 497 と 498 だけ | /sync-docs で 497 行目の数(1032、`f697bf0d` の時点で 911)と見積もり(6 本とヘルパー、約 320 行)を直し、495 行目と同じ型の持ち越しの note を足す(「2026-11-15: `dddd83bb` が Trigger を引き、テスト 2 本とヘルパーを足したが、分けなかった。分けるのは move-only のコード変更で、パイプラインを `/self-review` からやり直す」)。R3-3 の「既知の誤りを示す 1 行」を test に書くなら、その数も合わせる |

INFO(merge を止めない)

- I12: origin の `feat/gate-tool-versions` はすでに `afb8b78c`(HEAD)にある(`git ls-remote origin`)。指示どおりこの verify の commit は push しないので、ローカルが 1 つ進む。origin/main は `bbe2807d`(#328)で、`git rev-list --count HEAD..origin/main` は 0、`git merge-tree --write-tree --name-only origin/main HEAD` は衝突なし
- I13: merge `34936129`(AGENTS.md と tech-debt の衝突の解決)を、base・ours・theirs・merged の token の数で照合した。`ours + (theirs - base) - (base - theirs)` と merged の差は両ファイルとも 0(失った token も余った token も無い)
- I14: 計画の受け入れ条件の checkbox(AC1〜AC10)は `[ ]` のまま(/pr が付ける)。計画の D1・D6・D7 と AC10 の本文は、`dddd83bb` の「打つのは path」も書かない。digest が覆う D の本文が意図して後れる扱いは Run 2 の I8 と同じで、`"$rtc_p"` は D1 の「`command -v` と版を順に出させる」に反しない
- I15: Run 2 の I10(doctor の中断が exit 1)、Run 1 の I1・I2・I5・I6 は変わらない。`cross-review-triage-gate-tool-versions.md` の cycle の表記(「2/2 (cap reached」)は cap を 3 にしたあと `/cross-review` が書き換える

## Spec compliance

ブランチ全体を AC ごとに確かめた。`dddd83bb` が触れない AC は、Run 2 の根拠を引き、HEAD のコードで読み直して食い違いが無いことを確かめた(`internal/org/gate_tools.go`、`gate_run.go`、`gate_record.go`、`internal/cli/doctor_node.go`、`internal/toolchain/projects.go`、`version.go` は `d1ca8cd6` から無変更)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 log の `# tool` の行、shim の印、`# mise current:`、見出しの 1 行(issue の受け入れ条件 1) | 満たす | 変わらない。打つ語が path になっても、record の形(path、begin、exit)は同じで、`ParseProbe` と `gateToolsLogLines` に波及しない。shim の印は `ShimOf(Path)` で、`command -v` の出す path のまま。マルチコール型の shim(symlink、`$0` の basename で振り分ける script)を path で打っても、版を返す(P11) |
| AC2 記録の `tool_*` と `tool_dirs`、読み戻し | 満たす | `gate_record.go` は無変更 |
| AC3 測る場所の一覧 | 満たす | `projects.go` は無変更 |
| AC4 1 回の sh、15 秒、時間切れで verdict が変わらない、stdout を握る子、ctx の終わり | 満たす(直しの後も) | 直しはツールを打つ語だけを変えた。sh の数、`gateToolsTimeout`、`Run` の process group と `WaitDelay` に触れていない。時間切れと ctx のテストは `probe_test.go` に残っている(走らせていない) |
| AC5 `TOOLS:` の行、reviewer の prompt、`/org` と director の skill(4 写し) | 満たす | `TOOLS:` は `org` の skill 4 写しに各 5 件、director の skill 4 写しに各 1 件、reviewer の prompt に 4 件、gate と regate の help にある。`/org` skill の 3 写しと手順書の 2 写しは `cmp` で同じ。merge で入った `/org` skill の差(#328 の `send` の行)は `TOOLS:` の行に触れていない。`check-skill-sync.sh` が 14 skill で通った |
| AC6 doctor の「Node toolchain」(issue の受け入れ条件 2) | 満たす(ksh93 でも) | login ksh93 で、alias が単一の絶対パスなら path の版で比べ(P8)、引数つきの alias は打たずに warn する。zsh の alias と関数は warn で何も動かず(P9)、dash、sh、bash、ksh、zsh の PATH の node は、記録した path が打たれて pass(P10)。`$SHELL` が空・fish の扱いは `doctor_node.go` が無変更(Run 2 の P6) |
| AC7 npm の範囲の表のテスト | 満たす | `npmrange.go` は無変更 |
| AC8 手順書の mise・asdf の節 | 満たす | 2 写しは `cmp` で同じ。ksh93 の例外を言わない文は R3-1 |
| AC9 help、AGENTS.md の Repo map、tech-debt、2 つの同期の検査 | 満たす | `check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)と `check-skill-sync.sh`(14 skill)は run-static-verify の中で通った。gate・regate・doctor の help は `dddd83bb` に合わなくなる文が無い。tech-debt の数と note は V11 と R3-4 |
| AC10 測ることで何も変えない | 満たす(直しの後も) | 9 つの変数を置くスクリプトの頭(`probeEnvLines`)は `"$rtc_p"` の前に走る。login ksh93 の `.profile` が 5 つの変数を戻しても、打たれた node は `0\|local\|0\|false\|unset` を見た(P12。素の login ksh の対照は `AUTO_PIN=1 GOTOOLCHAIN=auto`)。`ProbeEnv` と scripts に渡す環境は無変更 |

### 計画の決定(D1、D6、D7)と直しの関係

- D1(1 回の sh、scripts と同じ環境と cwd、`command -v` と版): 満たす。解決先を測る D1 は、解決した path を打つ形と食い違わない。直す前は、`command -v` が答えた path と、打った名前が引く PATH のファイルが別になりえた
- D6(doctor、`$SHELL -lc`、warn): 満たす。`$SHELL` が ksh のとき、記録と版が同じファイルを指す。ksh93 の alias(単一の絶対パス)は、ksh が名前で打つものと同じファイルの版になる
- D7(測ることで何も変えない): 満たす。打つ先が名前から path に変わっただけで、ツールが見る環境は同じ

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | 通過(rc=0) | shellcheck、hook の `sh -n` と awk、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(gofmt ok、`0 issues.`)、branch の secret scan(`bbe2807d..afb8b78c`、clean)。raw は evidence に |
| `git diff --check origin/main...HEAD` | rc=0 | |
| `./scripts/plan-visual.sh digest <plan>` | `2774af0aab07` | 計画の `- Approved:` の行と同じ |
| `cmp` で `/org` skill の 2 写し(root と `templates/base/`)、手順書の 2 写し | 同じ | |
| `git ls-remote origin` | main `bbe2807d`、`feat/gate-tool-versions` `afb8b78c` | I12 |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | HEAD は origin/main を含む |

## Documentation drift

/sync-docs へ渡す一覧。3 回目の self-review の R3-1〜R3-4 を、HEAD で読み直して確かめ、`dddd83bb` が作った新しいずれ(V11)を足した。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `internal/toolchain/doc.go` 17〜24 行 | ずれた(R3-1) | 「an alias, a function, or a builtin is recorded as unknown and not run」は、ksh93 の単一の絶対パスの alias(P8)に偽。形の面では「runs only ..., and runs the version command only when ..., and runs that path (...)」と節がつながり、かっこの中身が「runs that path」にかかって読める。`MISE_NOT_FOUND_AUTO_INSTALL=0),` が 22 行目に 1 行で孤立している。2 文に切って折り返しをやり直す |
| `internal/toolchain/probe.go` の注(`probeFunc` 131〜146 行、`BuildScript` 174〜190 行、`notFileReason` 317〜324 行) | ずれた(R3-1、R3-2) | `probeFunc` の「alias は alias の行を記録しながら PATH のファイルを打つ」は、`case $rtc_p in /*` がすでに止めている(alias は noexec)。`"$rtc_p"` が守るのは、`command -v` が絶対パスを答える場合(ksh93 の alias)。`BuildScript` の「written for sh, dash, bash, and zsh」と `notFileReason` の「ksh is outside the four shells」は、`doctorNodeShells`(sh、bash、zsh、dash、ksh、mksh、yash の 7 つ、`doctor_node.go:66-69`)と数が違う。ksh93 は `dddd83bb` のテストで測る側に入った |
| `docs/recipes/isolated-herdr-server.md` と `templates/base/` の写し(360〜363 行) | ずれた(R3-1) | 「a tool that the login shell resolves to an alias or a function (or to a relative `PATH` entry) is not run」は ksh93 の単一の絶対パスの alias に偽。363 行目は 101 字で、`f697bf0d` の挿入が折り返されていない(ほかの行は 80 字未満)。R3-1 を直すとき、同じ段落をまとめて折り返す |
| `docs/tech-debt/README.md` 497 行目 | ずれた(V11) | 数(911 → 1032)、分け方の見積もり(4 本・約 200 行 → 6 本とヘルパー・約 320 行)、持ち越しの note |
| `docs/tech-debt/README.md` 498 行目 | ずれた(R3-3、R3-4) | 本文の (a) は ksh93 の引数つき alias(引用符つきで出て、`/` で始まらず、Reason が「関数か builtin」になる)を挙げない。(c) は新しい表(`TestProbe_Ksh93AliasOfOneAbsolutePath`)を挙げない。「Why deferred」の「the default pipeline cap (2)」は cap 3 のいま成り立たない。Trigger の語(`rtc_tool`、`notFileReason`)は `dddd83bb` が変えているので、文字どおりに満たされている。直さないなら 2 つのセルを書き直す(cap は ksh の直しのために 3 にした。ここは範囲の外) |
| `ralph org gate --help`、`ralph org regate --help`、`ralph doctor --help`、`doctor.go` の注 | 合う | 打つ語に触れる文が無い |
| README.md、`docs/quality/quality-gates.md`、仕様 FR-7 の 2026-11-13 の注、AGENTS.md の Repo map | 合う | 変わらない |
| reviewer の役割 prompt、`/org`・director の skill(4 写し) | 合う | `dddd83bb` で触れていない。merge で入った差は #328 の分 |
| 計画の Progress | /sync-docs が足す | `dddd83bb` と 3 回目の self-review の行は既にある(175 行目、`24b25bae`)。sync-docs の記録は /sync-docs が足す |

`/sync-docs` の注意: R3-1 と R3-2 の直しがコードの注だけでも、497 行目(`probe_test.go` に触れるとき)と 498 行目(`rtc_tool`、`notFileReason` に触れるとき)の Trigger は、注の直しでも引きうる。触れたら、数と持ち越しの note を同じ commit で揃える。

## Observational checks

HEAD から作った binary(`ralph-head`)と、`dddd83bb^` を `git archive` で展開して作った binary(`ralph-base`)を、scratch の HOME と git repo(`.node-version` が `22`、`package.json`、`pnpm-lock.yaml`)で `env -i ... ralph doctor --state-dir <存在しないパス>` として打った。node は PATH の先頭の偽の実行ファイル(v24.22.1)と、別の場所の `node22`(v22.1.0)で、呼ばれると自分の名前と `$0` を印のファイルに書く。raw は evidence の P8〜P12。scratch は `$TMPDIR/v3-gtv`(repo の外)。実機の HOME と `~/.local/share`、pnpm の store には何も書いていない。

| 場面 | 結果 |
| --- | --- |
| P8 login ksh93、`.profile` が `alias node=<node22>`(直す前 / HEAD) | control の `command -v node` は node22 の path。直す前: warn「resolves node v24.22.1 (node22 の path)」、打たれたのは PATH の node。HEAD: pass「node v22.1.0 meets .node-version (22)」、打たれたのは node22 だけ |
| P8 同、`alias node='<node22> --x'`(両方) | `command -v` は引用符つき `'<path> --x'`。warn「not a file (a shell function or builtin), not run」で何も打たれない。alias なのに「関数か builtin」と言う点は R3-3 のまま |
| P9 login zsh、alias と関数(HEAD) | alias は warn「an alias in the shell, not run」、関数は warn「not a file (a shell function or builtin)」。どちらも何も打たれない(`func-ran` の印なし) |
| P10 node22 を PATH の先頭に置いた login dash、sh、bash、ksh、zsh(HEAD) | どれも pass(v22.1.0)で、打たれたのは PATH の先頭のファイルだけ |
| P11 login zsh、`shims/node` が `shims/multi` への symlink(`$0` の basename で振り分ける script) | pass(v22.9.9)。path で打っても basename で振り分ける shim は動く |
| P12 login ksh93、`.profile` が `COREPACK_ENABLE_AUTO_PIN=1`、`GOTOOLCHAIN=auto`、`MISE_AUTO_INSTALL=1`、`PNPM_CONFIG_...=true`、`pnpm_config_...=true` を export | 対照(素の login ksh)は `AUTO_PIN=1 GOTOOLCHAIN=auto`。doctor が打った node は `0\|local\|0\|false\|unset` を見た |

scratch の repo は全部の実行のあとも `git status --porcelain` が空。

## Coverage gaps

- テストの実行結果(`/test`)。上の AC の根拠は、テストのソースとコードを読んだものと、上の観察。`TestProbe_RunsTheFileTheRecordNames`(bash の `BASH_ENV`)と `TestProbe_Ksh93AliasOfOneAbsolutePath`(`/bin/ksh` が無いか、ksh93 でなければ skip)は読んだだけで、走らせていない
- mksh と yash の `command -v` が alias で何を出すか(手元に無い)。`"$rtc_p"` にしたので、形を知らない shell でも記録と実行は同じ path になり、起きうるのは unknown と誤った Reason まで
- gate の `/bin/sh -c` での置き直しの直接観察と、実機の座席(codex の shell)での gate(tech-debt 491 行目のまま)。doctor 経由で、同じスクリプトを dash、sh、bash、ksh93、zsh に打った
- 実機の mise の shim は、この run では打っていない(Run 2 の P7 と計画の Progress の手元の確かめが証拠。P11 の symlink の shim が代わり)
- mise と corepack の変数が実際に取得と pin を止めること(計画の Assumptions は文書による)

足すと確かさがいちばん上がる検査: `TestProbe_Ksh93AliasOfOneAbsolutePath` と `TestProbe_RunsTheFileTheRecordNames` を、`/test` が走らせた結果で確かめる。

## Verdict

Run 3 の判定。

- Verified: AC1〜AC10 は、ブランチ全体で、コードとテストのソース、`run-static-verify.sh` の通過(rc=0)、実際の login shell(ksh93、zsh、dash、sh、bash)の観察から満たす。cross-review の 1 件の直しは、ksh93 の alias で、記録した path と打たれたファイルが同じになった(直す前の binary は v24 を記録の node22 の横に出した)。issue #311 の受け入れ条件 1・2 は AC1・AC6 に、計画助言の 2 件は D2・D7 と AC3・AC10 に入っている
- Partially verified: gate の `/bin/sh -c`(doctor 経由の同じスクリプトで観察)。mksh と yash
- Not verified: テストの実行結果。実機の座席での gate。実機の mise の shim(この run では未実行)
- Drift(/sync-docs へ): V11 tech-debt 497 行目の数と note、R3-1(doc.go、probe.go の注、手順書の 2 写し)、R3-2(shell の数)、R3-3 と R3-4(498 行目の本文と 2 つのセル)
- Verdict: pass(CRITICAL・HIGH・MEDIUM なし、LOW 1 件。R3-1〜R3-4 は self-review の指摘で、このレポートでも HEAD で確かめた)

## Run 2 の経過(cycle-count 2、HEAD `a1a396f6`)

pass で、LOW 4(V7〜V10)だった。cycle 2 の観察 P1〜P7(`7c0bf678` の 2 つの直しを実際の login shell で確かめた)の写しは evidence の cycle 2 の節にある。HEAD での状態:

| ID | Run 2 の指摘 | 状態 |
| --- | --- | --- |
| V7 | `doctor_node_test.go` が 996 → 1016 行になり、tech-debt 495 行目の Trigger を引いた。415 行目の「691 lines」も古い | 直った。495 行目は 1016 行と持ち越しの note、415 行目は 699 行(`12a9d1e3`) |
| V8 | `probe_test.go` が 909 行で、サイズの行が無い | 497 行目が足された。`dddd83bb` で数が古くなり、Trigger を引いた(V11) |
| V9 | doc.go、手順書、tech-debt 492 行目が直しのあとの probe を書き切っていない | 直った(doc.go、手順書の 2 写し、492 行目の混在した綴りの note)。直した文に出た残りは R3-1 |
| V10 | PATH の相対の項目で node が unknown になり、Reason と hint が事実と違う。self-review の LOW 1〜3 は tech-debt に行が無い | 498 行目に (a)〜(c) として書かれた。`dddd83bb` のあとの残りは R3-3、R3-4 |

Run 2 の INFO のうち、I7(origin/main が進んだ)は 3 回目の merge `34936129` で取り込まれた。I8、I9、I10、I11 は変わらない(I11 の `pnpm@1.0.0` は手元の store に残っていて、この run は何も足していない)。

## Run 1 の経過(cycle 1、HEAD `a6657d48`)

pass で、MEDIUM 1(V1)、LOW 5(V2〜V6)だった。

| ID | Run 1 の指摘 | 状態 |
| --- | --- | --- |
| V1(MEDIUM) | pnpm 自身の版の切り替えが 7 つの変数で止まらず、worktree の外に pnpm を入れる | 直った(S7 `e7274fec`)。`pnpm_config_manage_package_manager_versions=false` と `npm_config_...` を足し、9 つにした。`ProbeEnv` は大文字の綴りも消す。実機の doctor が取れない版名の repo で pnpm 12.9.1 を返し、store に新しいファイルが出ないことを Run 2 で確かめた |
| V2 | README.md の gate の段落に `TOOLS:` と `tool_*` が無い | 直った(README.md:291) |
| V3 | 仕様 FR-7 に 2026-11-13 の注が無い | 直った |
| V4 | `runDoctorOpts` の注が Check 14 の subprocess を書かない | 直った(`doctor.go:76-84`) |
| V5 | tech-debt の Trigger を引いた 13 行に持ち越しの note が無い | 直った |
| V6 | `doctor_node_test.go` のサイズの行が無い | 行は足された(495 行目)。以後は V7 |

Run 1 の INFO のうち、I3(手順書の箇条と `/org` skill の前提が Node toolchain を挙げない)は直り、I4(文字の濾し処理が 3 か所)は tech-debt の 494 行目にある。I1、I2、I5、I6 は変わらない。
