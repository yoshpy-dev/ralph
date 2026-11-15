# Self-review report: gate-tool-versions

- Date: 2026-11-15(Run 3)。Run 1 と Run 2 は 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 3 回目。`cycle-count.json` は cycle 2、cap 3(cap を上げても cycle は増えない)
- Scope: Run 2 の review(`a1a396f6`)から HEAD `24b25bae` までの差。中心は cross-review の ACTION_REQUIRED の直し `dddd83bb`(`rtc_tool` が名前ではなく `command -v` の出した path を打つ)。ほかに docs と comment の `f697bf0d`、そのあとの reports と plan の記録(`git log --oneline a1a396f6..HEAD`)。origin/main の取り込み(`34936129`)で入った #327・#328 の差は見ていない。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけで、仕様への適合、テストの網羅、文書の食い違いは `/verify`、`/test`、`/sync-docs` の担当。テストと静的解析は走らせていない
- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 4 件)

## Run 3 (cycle-count 2, cap 3)

### Findings

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。ID は Run 2 の `LOW 1`〜`LOW 3`(register の行が指している)と衝突しないよう `R3-n` にした。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R3-1 | LOW | readability | 直しで「alias は打たない」と言い切る文が、ksh93 では偽になった。`TestProbe_Ksh93AliasOfOneAbsolutePath` の 1 つ目の case は、値が 1 つの絶対パスの alias を `StatusFound` で測る(打つ)。手元の `/bin/ksh` で確かめた: `alias node=<node22 の path>` のとき `command -v node` は値だけを出し、path で打つと v22、名前を変数で打つと v24。それなのに次の文は alias を一律に「打たない」と言う: `doc.go:19-20`(「an alias, a function, or a builtin is recorded as unknown and not run」)、`probe.go:186-187`(`BuildScript` の doc の「an alias or a function a startup file defines gets a noexec record and is not run」)、手順書 `docs/recipes/isolated-herdr-server.md` の「Version managers」の節(2 つの写し。「resolves to an alias or a function ... is not run」)。例外を書いたのは `notFileReason` の comment(`:319-324`)だけ。(`recordNoExec` の行末 comment と doctor の hint は、打たなかった tool にだけ出る文なので偽ではない。)doc.go のこの文は、形の面でも `dddd83bb` が「and runs that path」を足したため「it runs only ..., and runs the version command only when ..., and runs that path (...)」と 3 つの節がつながり、かっこの中身が「runs that path」にかかって読める。同じ hunk で `MISE_NOT_FOUND_AUTO_INSTALL=0),` が 1 行(31 桁)に孤立していて、再 wrap が 3 行で止まっている。`probeFunc` の comment(`:136-146`)は `"$rtc_p"` を打つ理由を 2 つ挙げ、1 つ目は「alias は alias の行を記録しながら PATH のファイルを打つ」だが、これは `7c0bf678` の `case $rtc_p in /*` がすでに止めている(alias は noexec になり何も打たれない)。`"$rtc_p"` が守るのは、`command -v` が絶対パスを答える場合(ksh93 の alias)だけ | `internal/toolchain/doc.go:17-24`、`internal/toolchain/probe.go:136-146`、`:186-187`、`:319-324`、`docs/recipes/isolated-herdr-server.md`(`templates/base/` に写しあり)、`internal/toolchain/probe_test.go:890-914` | 一律の言い方を「`command -v` が絶対パス以外を出したとき(sh、dash、bash、zsh の alias、関数、builtin)」に直し、ksh93 の例外は `notFileReason` を指す(手順書は `/sync-docs` の範囲)。doc.go の文は 2 文に切って wrap をやり直す。`probeFunc` の comment は「打つのは `command -v` が出した path だけ。alias と関数は `/*` の検査で打たれない。ksh93 は値が 1 つの絶対パスの alias をその path で出し、それを打てば ksh が名前で打つものと同じになる」まで短くする |
| R3-2 | LOW | readability | どの shell を対象にするかを 2 か所が違う数で言う。`BuildScript` の doc は「written for sh, dash, bash, and zsh」(`probe.go:188-191`)、`notFileReason` の comment は「ksh is outside the four shells the script is written for」(`:324`)。一方 `doctorNodeShells` は sh、bash、zsh、dash、ksh、mksh、yash の 7 つを「the shells that run the probe's script」とし(`doctor_node.go:66-69`)、`$SHELL` がそのどれかならそこで打つ。この直しと `TestProbe_Ksh93AliasOfOneAbsolutePath` で ksh93 は測られる側になったのに、`notFileReason` の読者は「ksh は対象外」と読む。mksh と yash は手元になく(`/bin/mksh`、`/usr/bin/yash` なし)、alias の `command -v` の形は未確認で、plan の Progress だけがそう書く。ただし `"$rtc_p"` にしたので、形を知らない shell でも記録と実行は同じ path になる。起きうるのは unknown と誤った Reason までで、偽の一致は起きない | `internal/toolchain/probe.go:188-191`、`:319-324`、`internal/cli/doctor_node.go:66-69`、`docs/plans/active/2026-11-12-gate-tool-versions.md`(Progress の 2026-11-15 の行) | 1 か所(`doctorNodeShells` か `BuildScript`)に「テスト済み: sh、dash、bash、zsh、ksh93。受けるが未確認: mksh、yash(記録と実行は同じ path なので、起きうるのは unknown と Reason の誤りまで)」と書き、`notFileReason` の「outside the four shells」は消す |
| R3-3 | LOW | maintainability | 新しい ksh93 のテストが、既知の誤りを正解として固定し、表の形は登録済みの LOW 3 と同じ。(a) 2 つ目の case「an argument」(`probe_test.go:909-912`)は、alias に引数があるとき `Reason: reasonNotFile` つまり「not a file (a shell function or builtin)」を期待する。alias なのに「関数か builtin」と言う文で、同じ tool の doctor の hint は「an alias or function」と言うので、1 行に 2 つの原因が並ぶ。plan の Progress は「残した点」と書くだけで、register の行 (a)(`docs/tech-debt/README.md:498`)は相対パスの入り口しか挙げない。plan は `/pr` で archive されるので、このまま進むと記録が残らない。(b) 新しい表(`:896-914`)は `value`、`printed`、`want` の 3 つが `func(node22 string)` で、2 つの case の違いは `" --x"` の接尾辞とその引用だけ。登録済みの LOW 3(行 (c))と同じ形で、行 (c) は旧い表しか挙げない | `internal/toolchain/probe_test.go:896-914`、`internal/toolchain/probe.go:313-314`、`:319-324`、`docs/tech-debt/README.md:498`。手元の確認: `alias node='<path> --x'` のとき `/bin/ksh` の `command -v node` は `'<path> --x'` を出す | 行 (a) に「ksh93 の引数つき alias は引用符つきで出て、`/` で始まらず Reason が誤る」を足し(直す場所は同じ `notFileReason`。先頭が `'` なら alias、`/` を含むなら相対パス、で分けられる)、行 (c) に新しい表を足す。テストには、case の `Reason` の上に「既知の誤り。行 (a) で追う」と 1 行書き、直す人が意図して変えるようにする |
| R3-4 | LOW | maintainability | register の行 (a)〜(c) の「Why deferred」が成り立たなくなり、Trigger は文字どおり満たされた。「Why deferred」は「`7c0bf678` は既定の cap(2)の 2 回目で、直しは code の変更でパイプラインを `/self-review` からやり直す」と言う。今は cap が 3(`.harness/state/standard-pipeline/cycle-count.json`: cycle 2、cap 3)で、`dddd83bb` のためにパイプラインはすでに `/self-review` から回り直している。やり直しの費用を理由にした先送りは、この run では理由にならない。Trigger は「probe の records(`rtc_tool`、noexec の record、`notFileReason`、`ParseProbe`)の次の変更」で、`dddd83bb` は `rtc_tool` の打つ行を変え、`notFileReason` の comment を書き換えた。record の形は変えていないので「満たされていない」とも読めるが、行はどちらとも言っていない | `docs/tech-debt/README.md:498`(Why deferred と Trigger の 2 つのセル)、`.harness/state/standard-pipeline/cycle-count.json`、`git show --stat dddd83bb` | 判断はオーケストレーターに任せる。直すなら (a)〜(c) と R3-3 は `rtc_tool`、`notFileReason`、`ParseProbe`、`Measurement`、`notResolved` の同じ数か所で、`Measurement` に `NotRun` の欄を 1 つ足せば (b) が閉じ、`notFileReason` に相対パスと引用符つき alias の分岐を足せば (a) と R3-3 の (a) が閉じる。直さないなら、2 つのセルを「cap は ksh の直しのためだけに 3 にした。ここは範囲の外」と書き直す。`cross-review-triage-gate-tool-versions.md` の「Cycle: 2/2 (cap reached」は `/cross-review` が書き換えるので、行から指さない |

### Evidence reviewed

`dddd83bb` の 3 ファイル(`probe.go`、`doc.go`、`probe_test.go`)の差を全部読み、`rtc_tool`(`probe.go:147-171`)と `BuildScript`、`notFileReason`、`ShimOf`、`doctor_node.go` の `notResolved` と `doctorNodeShells` を現在の HEAD で読み直した。`f697bf0d` の差(comment、doc.go、手順書 2 つの写し、register)、plan の Progress の追加分(`24b25bae`)、`docs/tech-debt/README.md:498`、cross-review の triage(`ca14df41`)と insight event も読んだ。

名前を打つのをやめて path を打つことで挙動が変わるかは、次の順に確かめた。

- argv[0]: 手元で `#!/bin/sh` の `arg0`(`echo $0`)を PATH に置き、名前で打つ場合と `"$(command -v arg0)"` で打つ場合を比べた。どちらも `$0` は PATH で見つかったフルパスで、同じ。コンパイル済みの binary は名前で打つと argv[0] が裸の名前、path で打つとフルパスになる。この probe が打つ tool(node、pnpm、yarn、npm、python3、go、mise)で、argv[0] の basename ではなく裸の名前に頼るものは見つからなかった。shim(mise、asdf、pyenv、volta)は basename で振り分けるので、basename は変わらない。plan の Progress にある手元の確認(node、pnpm、python3、go、npm、mise の shim の node と python3 が、名前と path で同じ版を返した)と合う
- 安全性: `"$rtc_p"` は `command -v` の出力(PATH の環境が決める)を引用符つきで 1 語として打つ。`case $rtc_p in /*)` を通ったものだけが届き、`eval` も glob も word splitting もない。先頭が `/` なので `-` 始まりの引数にもならない。stdin は `</dev/null`、stderr は stdout へ、record の書式は変更前と同じなので、`ParseProbe`、doctor、gate の log は変わらない
- ksh93: `/bin/ksh` で 3 通りを打った。値が 1 つの絶対パスの alias は `command -v` が値だけを出し、`type` は「node is an alias for <path>」、名前を変数で打つと v24(PATH のファイル)、path で打つと v22(alias の指す先)。引数つきの alias は `'<path> --x'` と引用符つきで出て、`/` で始まらないので noexec になる。テストの期待と comment の説明はどちらもこれと合う
- 新しいテスト 2 本は、読みで次のように判断した。`TestProbe_RunsTheFileTheRecordNames` は bash の `BASH_ENV` で関数 `command` を定義して `command -v node` に node22 の path を答えさせる。変更前のコードは名前で PATH の `node` を打つので、version は v24.22.1 になり marks に `path-node` が残る。期待は node22 の path、v22.1.0、marks が node22 だけなので、3 つとも落ちる。bash があれば Linux の CI でも走る。`TestProbe_Ksh93AliasOfOneAbsolutePath` は `/bin/ksh` がなければ skip し、対照の実行が ksh93 の形を出さないときも skip するので、形の違う ksh で黙って通ることはない。Linux の CI では走らず、回帰は前者が持つ
- script の text の固定: `"$rtc_p" "$@" </dev/null 2>&1` が 1 つ、`"$rtc_t" "$@"` が 0 個。後者は旧い綴りの再導入だけを防ぐ変更検知で、挙動は 2 本のテストが持つ

見ていないもの: mksh と yash の `command -v`(手元になし)、テストの実行。

### Positive notes

- 直しは `rtc_tool` の 1 語(`"$rtc_t"` → `"$rtc_p"`)で、record の形を変えていないので、`ParseProbe`、doctor、gate の log に波及しない。記録した path と打つ path が文字列として同じになり、shell が `command -v` で何を出しても食い違えないので、手元で形を確かめられない mksh と yash にも効く
- bash のテストは、ksh93 の実物がない環境でも「記録と実行が同じ」という性質を見られる作りで、ksh93 のテストは実物の振る舞いを対照の実行で確かめてから本番を見る
- `writeRecordRunFakes` で 2 本のテストが fixture を共有し、戻り値に名前を付けて `dir`、`bin`、`node22`、`marks` の役目を読めるようにしている。marks のファイル名(`path-node`、`node22`)が、どの node が打たれたかをそのまま言う

### Tech debt identified

新しい行は要らない。R3-3 と R3-4 は既存の行(`docs/tech-debt/README.md:498`)の直しで、本文と 2 つのセル(Why deferred、Trigger)を同じ向きに揃える。行の「Related plan/report」セルが指す `self-review-...md`(cycle 2、LOW 2 と LOW 3)は、下の「Run 2」の表に同じ番号で残してある。

### Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 4 件)
- Follow-ups:
  - R3-1 と R3-2 は comment だけの直しで、この run の `/sync-docs` がまだ走っていないので、cycle 2 の LOW 1 と同じようにそこで直せる
  - R3-3 と R3-4 は register の行の直しで、同じく `/sync-docs` で済む。テストの `Reason` の上に 1 行書く程度なら comment なので同じ扱い。表の形を直す(R3-3 の (b))なら code の変更になり、record-only の例外は使えず、パイプラインが `/self-review` からやり直しになる
  - 4 件とも直さず `/pr` へ進む場合は、R3-4 の行の 2 つのセルだけは書き直す(cap の理由が偽のまま archive されるため)
  - この Run の insight event の cycle は 2 のまま(cap を上げても cycle は増えない)で、Run 2 の `self_review` の event と cycle の値が同じになる。見分けは `ts` でつく

## Run 2(cycle-count 2、`a1a396f6`)

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3、Merge: yes。対象は cross-review の cycle 1 の直し `7c0bf678`(script の頭で 9 つの変数を `unset` と `export` し直す、`command -v` が絶対パスを出さない tool を打たず noexec にする)と merge `41f722d1`。ID は register の行 `docs/tech-debt/README.md:498` が指している番号のまま。

| ID | Severity | Finding(要約) | 今の状態(HEAD `24b25bae`) |
| --- | --- | --- | --- |
| LOW 1 | LOW | alias の comment が実際と違う。`"$rtc_t"` は alias を展開しないので、alias で起きるのは「記録は alias を指すのに PATH のファイルを打つ」ずれで、alias の中身は動かない。「どの shell も `alias x=` と出す」も偽で、ksh93 は値だけを出す | 直った。`f697bf0d` が `probeFunc`、`notFileReason`、テストの comment を直し、`dddd83bb` が推奨どおり `"$rtc_p"` を打つようにした。直した文に出た残りは R3-1 と R3-2 |
| LOW 2 | LOW | 「ファイルではないので打たなかった」を 4 か所(script の `case /*`、`ParseProbe`、`ShimOf`、`notResolved`)が `Path` の形から組み直す。`Measurement` に欄がない。warn の行が同じことを 3 回言う | 残り。行 (b) に登録済み(`docs/tech-debt/README.md:498`)。R3-4 を参照 |
| LOW 3 | LOW | `TestProbe_AliasOrFunctionInStartupFileIsNotRun` の表が読みにくい(`startup` と `wantPath` が closure、`%s` を `strings.Replace` で埋める)。定数 `aliasRec` と `aliasZsh` の名前が揃わない | 残り。行 (c) に登録済み。新しい ksh93 のテストの表が同じ形で、R3-3 を参照 |

## Run 1(cycle 1、`245b85f9` と `5b84df9a`)

1 回目は LOW 7、2 回目は LOW 5 で、判定はどちらも yes だった。1 回目の LOW のうち 6 件は `b0b3bfe2` で直った。2 回目の LOW 1(doctor が中断を飲み込み exit 0)と LOW 4(root の nocd が測れた値を上書き)は `9c19e5ba` で直り、LOW 3(`doc.go` の言い切り)も同じ commit で直った。LOW 2(文字の濾しが 3 か所)は tech-debt に書き、2 回目の残した LOW(doctor で `ProbeEnv` を 2 回当てる)は冪等なので残した。
