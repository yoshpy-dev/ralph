# Self-review report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(cycle 2)。cycle 1 の 2 回の review(`245b85f9`、`5b84df9a`)は末尾の「Cycle 1 の経過」に短くまとめた
- Scope: `git diff origin/main...HEAD`(HEAD `e45f7fe3`、origin/main は `e902705c`)。cycle 2 で足された commit は `7c0bf678`(cross-review の ACTION_REQUIRED 2 件の修正)と、計画に記録だけを足した `e45f7fe3`。merge `41f722d1`(#312a を取り込んだ)の解決も見た。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テストと静的解析は走らせていない
- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 3 件)

## Findings

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3(すべて cycle 2 で新しい)。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | alias についての説明が実際の動きと合わない。`probeFunc` の doc は「alias を打つと alias の中身が動く」と書き、`notFileReason` の doc は「どの shell も alias を `alias <name>=...` と出す」と書く。実際には、`"$rtc_t"` は変数なので alias を展開しない。手元で alias `nodex='echo ALIAS'` と PATH 上のファイル `nodex` を置いて `f() { t=$1; shift; "$t" "$@"; }; f nodex -v` を打つと、bash と dash は PATH のファイル(`FILE`)を打ち、`ALIAS` は出ない(関数なら `FUNC` が出る)。つまり alias で起きていたのは「記録は `alias ...` なのに、打ったのは PATH のファイル」というずれで、alias の中身は動かない。計画の cross-review の項(「`"$rtc_t"` は alias を展開せずに PATH のファイルを打つ」)は正しく書いているので、コードの comment と commit message(「an alias ... ran instead」)だけが違う。「どの shell も」も合わない。macOS の `/bin/ksh`(ksh93)は `alias node=/x/node22` に `command -v node` で alias の値 `/x/node22` だけを出し、`/*` を通って `"$rtc_t"` が PATH のファイルを打つ。ksh は対応する 4 つの shell(sh、dash、bash、zsh)の外だが、doc は範囲を言っていない。テストの comment(770 行目「whatever the function or alias would have run」)も同じ言い方 | `internal/toolchain/probe.go:136-138`、`:311-313`。`internal/toolchain/probe_test.go:765-770` | alias は「打っても alias は動かないが、記録の path と打つものがずれる」と書き直す。「どの shell も」は「sh、dash、bash、zsh は」にする。きっちり閉じたいなら、`"$rtc_t"` の代わりに検査を通った `"$rtc_p"` を打てば、記録と実行が同じ文字列になる(ksh93 の alias の値も、記録した path をそのまま打つので合う)。ただし argv[0] がフルパスになるので、shim の挙動を見てから決める |
| LOW | maintainability | 「ファイルではないので打たなかった」という事実を、doctor が Path の形から組み直している。判定が 4 か所にある(script の `case $rtc_p in /*`、`ParseProbe` の `noexec`、`ShimOf` の `HasPrefix(p, "/")`、doctor の `!strings.HasPrefix(m.Path, "/")`)うえ、`Measurement` に「打たなかった」を表す欄がない。その結果、(a) doctor の hint が `Path` の形に頼り、`./node` のような相対パスにも「an alias or function there is not measured」と言う。相対パスは cycle 1 から既知の件で、hint の文が間違った助言を足したぶんだけ少し悪くなった。(b) 同じ事実が 1 行に 3 回出る。例: `... could not read the version of node (alias node='/x'): an alias in the shell, not run (the probe runs files only); make node resolve to a file in a non-interactive login shell (an alias or function there is not measured)`。Reason の「not run」と「the probe runs files only」、hint の「is not measured」は同じ内容で、gate の log では `unknown (an alias in the shell, not run (the probe runs files only)) (alias node='/x')` と括弧が 3 重になる | `internal/cli/doctor_node.go:373-375`、`internal/toolchain/probe.go:153-159`、`:315`、`:410-411`、`internal/toolchain/version.go:64`。テスト `internal/cli/doctor_node_test.go` の `TestDoctorNodeToolchain_NotAFileWarnsWithAHint`(hint の文字列を全文で固定している) | `Measurement` に `NotRun bool`(または `NotFile`)を足し、`ParseProbe` が `noexec` で立て、doctor はそれを見る。hint は Reason が言わないことだけにする(「; make node a file on PATH of a non-interactive login shell」程度)。相対パスの Reason を直すなら、`strings.Contains(path, "/")` で「a relative path」と分けると、hint の誤りも一緒に消える |
| LOW | readability | `TestProbe_AliasOrFunctionInStartupFileIsNotRun` の表が読みにくい。`startup` と `wantPath` が `func(fake string) string` で、関数の行は `fake` を使わない(`func(string) string { return "node" }`)。fake の node の中身は `fmt.Sprintf` ではなく `strings.Replace(script, "%s", ...)` で埋めていて、`script` の `%s` は printf の書式に見える。別のテストの定数 `aliasRec` と `aliasZsh` は名前の付け方が揃っていない(前者の「Rec」は何の略か読めない) | `internal/toolchain/probe_test.go:243-244`、`:775-800` | 表の欄は文字列(`startupFmt` と `wantPath` に `%s` を 1 つ)にして `fmt.Sprintf` で埋める。定数は `aliasPosix` と `aliasZsh` のように shell の系統で揃える |

## Evidence reviewed

cycle 2 の差分 `7c0bf678`(4 ファイル、319 行追加、28 行削除)を全部と、`internal/toolchain/probe.go` の全文(509 行)を読んだ。memory の cycle 2 の手順に従い、cycle 1 の指摘の行を 1 つずつ確かめた(下の「Cycle 1 の経過」)。

shell の行の可搬性は、`/bin/sh`(macOS では bash 3.2 の POSIX mode)、`/bin/bash`、`/bin/zsh`、`/bin/dash` で手元で打って確かめた。

- スクリプトの頭の `unset` と `export`: 9 つの変数の大文字と小文字の綴りを `unset` し、9 つを `export` する行を、`PNPM_CONFIG_MANAGE_PACKAGE_MANAGER_VERSIONS=true GOTOOLCHAIN=auto` を渡した 4 つの shell で打つと、どれも大文字の綴りは unset、`GOTOOLCHAIN` は `local`、`pnpm_config_manage_package_manager_versions` は `false`、rc は 0 だった。zsh が lower-case の配列に結ぶ名前(`path`、`fpath` など)は 9 つの名前と重ならない
- `rtc_tool` の `case $rtc_p in /*) ;; *) ...; return 0 ;; esac`: ソースの関数を書き写して 4 つの shell で打った。PATH 上のファイルは path、begin、exit の記録、関数と builtin(`cd`)は path と noexec の記録で止まり、PATH に無いものは missing。`printf` の書式の `%s` の数と引数の数は 3 つの record すべてで合い、noexec の記録は末尾が空の 5 欄になって `ParseProbe` の `len(fields) == 5` を通る。zsh は `-f` なしだと `~/.zshenv` が PATH を変えるため手元の fake が見つからなかったが、これは起動ファイルの影響そのもので、`-f` を付けると他の shell と同じ結果だった
- `command -v` の alias の出し方: sh、bash(`expand_aliases` あり)、dash、zsh は `alias ...` を出す。zsh は引用符を付けない(`alias node=/x/node22`)ことをテストが押さえている。ksh93 だけ値を出す(finding 1)

merge `41f722d1` の解決は、親ごとの語単位の diff で確かめた。`AGENTS.md`: main 側(`e902705c`)との差は branch 側の足した語(`internal/toolchain/` の項、`gate_tools.go` の句)だけ、branch 側(`0abfb274`)との差は main 側の足した語(`usage_limited` の句)だけで、どちらの側の語も落ちていない。`docs/tech-debt/README.md`: main 側から消えた語はなく、branch 側から消えたのは `below)` と `dialog)` の 2 語で、どちらも main 側が同じ場所に `below); ...` と `dialog); ...` を足したもの。表の列数も、触れた 37 行は他の行と同じ(5 列)。行内の `docs/plans/active/2026-11-12-gate-tool-versions.md` への参照 22 か所は、`/pr` の `archive-plan.sh` が書き換える(main の行は archive の path)。

ほかに見たもの: `internal/cli/doctor_node.go` の `notResolved` と 2 つの呼び出し(`:396`、`:480`)、`internal/org/gate_tools.go` の log の行と `Mise` の使い手(`MiseCurrent: true` を渡すのは gate だけで、gate の shell は `/bin/sh -c` で起動ファイルを読まないので、`mise` が関数になる場合は gate では起きない)、`9c19e5ba`(cycle 1 の S6)の本体の変更。

## Positive notes

- スクリプトの頭の行を `probeEnvOverrides` から作ったので、環境(`ProbeEnv`)と script の 2 つの経路が同じ一覧から出る。comment に「なぜ 2 回要るか」(起動ファイルが後から上書きする)と、直せない穴(大文字小文字の混ざった綴り)が書いてある
- `TestProbe_StartupFileCannotTurnTheSafetyVariablesBackOn` は、先に対照の実行で「起動ファイルが `ProbeEnv` の下でも変数を戻す(`1 true`)」ことを確かめてから本番を見る。`export` の行を消すと `UNSET UNSET`、`unset` の行を消すと `0 true` になるはずで、どちらの行を消しても落ちる(読んで導いたもので、変異は走らせていない)
- `TestProbe_AliasOrFunctionInStartupFileIsNotRun` は「Measurement が unknown」に加えて「どのファイルも打たれていない」(marks が空)を見る。ガードを外すと、関数の場合は関数が、alias の場合は PATH 上のファイルが打たれて marks に出るはずなので、どちらの行も効く(これも読んで導いた)
- `scriptWithoutEnvLines` は、頭の行が別の場所へ動くと `TrimPrefix` が何もせず、対照の実行は変数を戻した script を打つ。すると「shim が書く」ことを期待する対照が落ちるので、黙って通る向きではない
- `noexec` の Reason が `"not measured"` で始まらないことを定数で固定し、`coveredByFailure` との結合(接頭辞)を comment とテストの両方に書いた

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この review で足す行はない。ただし 1 点。PATH に相対パスがあるときに Reason が「関数か builtin」になる件(cycle 2 の implementer が残した点)は、今は計画の Progress にしか書かれていない。計画は `/pr` で archive されるので、直さずに進めるなら `docs/tech-debt/README.md` に 1 行要る(finding 2 の直しに含めれば行は要らなくなる)。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 3 件)
- Follow-ups:
  - finding 1(comment と commit message の alias の説明)は文の直しだけで済み、`/sync-docs` の前に直すと文書側が正しい元から写せる
  - finding 2 は `Measurement` に 1 欄足す小さな直しで、相対パスの件も一緒に閉じる。hint の文言を直すなら `TestDoctorNodeToolchain_NotAFileWarnsWithAHint` の全文も変わる
  - finding 3 は読みやすさだけで、直さなくても挙動は変わらない
  - 3 件をまとめて直すなら、コードと comment に触れるので cycle の上限(2)に当たる。直すなら record-only の例外は使えない(コードを変えるため)。直さず `/pr` へ進む場合は、残した LOW を 1 行にまとめて `docs/tech-debt/README.md` に書く

## Cycle 1 の経過

cycle 1 の 1 回目(`245b85f9`)は LOW 7、2 回目(`5b84df9a`)は LOW 5 で、判定はどちらも yes だった。cycle 2 の時点の状態:

| cycle 1 の指摘 | 状態 | 確かめたこと |
| --- | --- | --- |
| 1 回目 LOW 1〜3・5〜7 | 直った(`b0b3bfe2`) | 2 回目の review で確認済み |
| 2 回目 LOW 1(doctor が中断を飲み込み exit 0) | 直った(`9c19e5ba`) | `checkNodeToolchain` が中断の cause を返し、`runDoctorStateDir` が全部の結果を出した後に `Interrupted` の行と error を返す。中断と失敗が重なると失敗の件数も error に入る |
| 2 回目 LOW 2(文字の濾しが 3 か所) | tech-debt に書いた | `docs/tech-debt/README.md` の「Three one-line text filters ...」の行(`4684ecf0`)。差分が足した行は計画の記録と合う |
| 2 回目 LOW 3(`doc.go` の言い切り) | 直った(`9c19e5ba`) | 「Measuring is set up to change nothing」「The probe script itself installs nothing」に変わった。cycle 2 で「ファイルのときだけ」を足す件は `/sync-docs` に渡してある |
| 2 回目 LOW 4(root の nocd が測れた値を上書き) | 直った(`9c19e5ba`) | `ParseProbe` の `case noCD[t.Dir] && (st == nil \|\| !st.exited)`。cycle 2 の `noexec` の case はその前にあり、同じ向きで効く |
| 2 回目の残した LOW(doctor で `ProbeEnv` を 2 回当てる) | 残した(orchestrator の判断) | 冪等で害がない。cycle 2 で script の頭の行が 3 つ目の経路になったが、同じ一覧から出るので悪くはならない |
