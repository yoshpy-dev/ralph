# Verify report: guard-msg-param-flag

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-msg-param-flag.md(`- Approved:` の digest ebf5a9ac3a15 は `./scripts/plan-visual.sh digest` の値と一致。承認の後の plan の変更は Progress だけ)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff origin/main...HEAD`(base c3a9242e、HEAD 899fff16、7 ファイル、+593/-90)。コードは 1e032dea(S1)と 43e73568(S1b)、tech-debt は b4646f85(S2)、self-review の LOW 3 のコメントの直しは 899fff16。テストスイートは実行していない(AC5 は /test が見る)。probe には plan・テストの配列・tech-debt の行・`scratchpad/mp/` にある形だけを使い、guard を避ける新しい形は作っていない
- Evidence: `docs/evidence/verify-2026-10-09-guard-msg-param-flag.log`(gitignore の対象なのでコミットしない。scripts は `scratchpad/mpv/`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 10 形がどのモードでも none(jq あり・なし) | Met | `mpv/probe.sh` の 01〜09 と 40(推奨の HEREDOC の形)。HEAD の root と template で none/none、permission_mode を absent・default・auto・bypassPermissions・acceptEdits・plan の 6 通りにしても jq あり・なしとも none。base(c3a9242e)では単一引用符・`\${`・`$'…'` の 4 つのメッセージと `rg 'foo$' 'sudo ' .` が deny/deny で、この PR で none になった。10 形はすべて D 節の `edge_none` にある(配列を source して取り出し、文字列で完全一致) |
| AC2: 12 形がどのモードでも deny、旧版も deny | Met | probe の 10〜21。HEAD の root と template で deny/deny、6 モードとも deny、旧版(`tests/fixtures/guard-1c4cea5a/`)も 12 形とも deny/deny。base で none だったのは 8 形(`rg $x`、`rg "$x"`、`rg $(echo --pre)`、添字の 4 形、`git commit -m $arr['$(sudo ls)']`)。12 形はすべて B 節の `guard_deny_only_forms` にある(group 9 の `$'\x2d-pre'` と group 10 の `${(e)…}` の 3 形は base からあり、残りの 8 形は group 11 で足された) |
| AC2b: 添字の中の置換を読み直す 2 形がどのモードでも deny、旧版は通す | Met | probe の 22・23。HEAD deny/deny(6 モードとも)、base も deny/deny、旧版 none/none。2 形とも D 節の `edge_deny` にある。`lex_word` は添字を読み飛ばさず(`.claude/hooks/pre_bash_guard.sh:340-399`)、語の終わりで `xnote` するだけ(`:391`) |
| AC3: `ac3` の 29 形は none のまま、G 節の例外は `intentional_fixes` の 13 件のまま | Met(テストの実行は /test) | `ac3`(29 行)と `intentional_fixes`(13 行)は base と HEAD で同じ(`mpv/arrays.py`、removed 0・added 0)。HEAD は 29 形とも none/none。G 節の比較を静的にたどり直した(`mpv/analyze.py`): 比較の例は `collect_corpus=yes` の間に集める `pr206_deny`・`ac2`・`self_review_forms`・`guard_deny_only_forms`・`ac3` の 223 行で、そのうち「旧版 deny かつ HEAD none」は jq あり・なしとも 13 行、`intentional_fixes` と完全に一致した。C 節の旧版 deny も 13 行で一致。group 11 の 13 行は HEAD も旧版も deny なので、この一覧に入らない |
| AC4: root と template の guard がバイト単位で同じ | Met | `cmp` rc 0(`lib_json.sh` も rc 0)。`./scripts/check-sync.sh` は PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0)。全 618 行の判定でも template と root の違いは 0 行 |
| AC5: guard のテスト、lib_json のテスト、`run-verify.sh` が通る | Not verified here | /test の担当。`run-static-verify.sh` の静的な部分は rc 0。implementer の記録ではテストは 2032 件 |
| AC6: tech-debt の、PR #211 の止めすぎの行の (a) と、guard の限界の行の rg の変数の語の記述が解消済みになる | Partially met | `docs/tech-debt/README.md:166` の (a) は取り消し線と「Closed in fix/guard-msg-param-flag」の注記(b4646f85)になり、Trigger も (a) を済みにした。`:160` は Debt 列の rg の段落に「Closed in …」の注記が入り、`:163` の「Not pinned: `rg $x sudo pat .`」も取り消し線になった。注記の中の判定(`rg $x sudo pat .`、`rg "$x" 'sudo ' .`、`rg $(echo --pre) sh 'sudo ls'` が deny、`rg 'foo$' 'sudo ' .` が none)は probe と一致する。ただし `:160` のほかの列と Debt 列の別の 2 か所には、rg の変数の語を未解決として書いた記述が残る(V-1)。self-review の LOW 1 のとおり、この cycle の /sync-docs で直す予定 |

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `.claude/hooks/pre_bash_guard.sh` が言語に分類されず full に切り替わった。shellcheck(hook と verify の scripts)OK、root と template の hook すべての `sh -n` OK、check-sync PASS、check-pipeline-sync OK、check-skill-sync 13 件 OK、template-purity PASS、tech-debt の plan 参照 OK、golang(gofmt ok、0 issues)、branch secret scan(c3a9242e..899fff16)clean |
| `shellcheck -S warning`(guard 2 つ、テスト) | rc 0 | shellcheck 0.11.0。指摘 0 件 |
| `cmp` guard の root と template | rc 0 | `lib_json.sh` も rc 0 |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `sh -n` guard、`bash -n` テスト | rc 0 | |
| awk のプログラムの単一引用符 | 他に 0 個 | `LC_ALL=C awk '`(`:174`、ファイルに 1 か所だけ)のあとの最初の単一引用符は `:1553` の行頭で、続きは ` 2>/dev/null)" \|\| awk_status=$?`。つまり意図した閉じ引用符で、間の 1378 行に単一引用符はない(`mpv/awkq.py`)。base のプログラムは `:165-1444` |
| plan の digest | 一致 | ebf5a9ac3a15 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| guard のヘッダーのデータ区間 (a)〜(d)(`:80-109`)と「Not covered」(`:138-146`) | おおむね Yes | (a) の rg の条件は `stage_note` の判定(`:1040`)と、添字の範囲は `lex_word` の `xnote`(`:391`)と合う。(b) の「展開する `$`」は `LD_EXP`(`:521`)と合うが、二重引用符の外の `$"…"` を展開しないものに挙げていない(V-3) |
| `lex_word`・`lex_dq`・`lex_dollar`・`add_word`・`stage_note`・`msg_check` のコメント | Yes | self-review LOW 3 の 3 点(`lex_word` の「決してデータにならない」、`stage_note` の並べ替えの例、テストの group 10 の「whole source word」)は 899fff16 で直った |
| テストのヘッダーの B と D、group 11 と D 節の新しいコメント | Yes | コメントに書かれた PR #211 と旧版の判定は、全 618 行の判定(下)と合う |
| `.claude/rules/ralph/git-commit-strategy.md:69` と template の写し | Yes | HEREDOC の形と、読むだけのコマンドを「`echo` や `grep` など」と例で挙げるだけ。この PR で変わる振る舞いは書いていない。写しは check-sync で一致 |
| `internal/org/prompts/implementer.md:24-33`、`.codex/README.md:114-118` | Yes | ダブルクォートの `-m` の `$(` とバッククォート、HEREDOC の形、deny だけで permission_mode を読まないこと。どれも変わっていない |
| tech-debt の guard の限界の行(`:160`) | No | V-1 |
| tech-debt のテストの穴の行(`:163`) | ほぼ Yes | rg の変数の語の「Not pinned」は取り消し線になった。Why deferred の「the rg variable word joined them in the cap-4 run, whose last /test did not pin it」は当時の記述として読める。前の PR が書いた「Pinned since …」の段落に当たるものはこの PR にはない(任意) |
| tech-debt の `:166`・`:167` | 一部 | V-2 |
| plan | 一部 | V-4(Scope の「`edge_sentinel_deny` などから `edge_none` に移す」に当たる行はなかった)。AC のチェックボックスが空なのはいつもの遅れで、判定には使わない |

### Findings

- V-1(LOW、/sync-docs 向け): `docs/tech-debt/README.md:160` は、Debt 列の rg の段落(`rg $x sudo pat .` の前後)にだけ「Closed in fix/guard-msg-param-flag」の注記を足した。self-review LOW 1 が挙げた 5 点は HEAD でもそのまま残っている。Impact (b) の「a variable word that rg reads as `--pre`」、Why deferred の「the rg variable word of (b) stay open」、Trigger (b) の「add `rg $x sudo pat .` to `edge_sentinel_deny`」(実際の固定先は B 節 group 11)、Impact (a) の rg の止めすぎを `$'…'` と `$"…"` の語だけとする記述、それに発火した Trigger (d) と (e) の記録がないこと。(d) の大きさは、guard が 1487 行から 1596 行に(+109 行。self-review の時点の +106 に、899fff16 のコメントの 3 行が加わった)、awk のプログラムが 1280 行(`:165-1444`)から 1380 行(`:174-1553`)に増えた。self-review が挙げていない古い記述も Debt 列に 2 か所ある
  - データ区間の定義: 「A `${…}` span is no data region either, since 139d6627 (…; `xnote` in `lex_dollar`, the `${` check in `msg_check`, and `HSUB` in `lex_hd`)」。`msg_check` が見るのは、もう `${` ではなく語の展開の印 `WEXP` で(`:1401`)、メッセージは `${` に限らず展開する `$` があればデータにならない。zsh の添字の範囲(`$` から語の終わりまで)がデータにならないことも、この定義に書かれていない
  - 「Three commands carry a condition of their own」の段落: rg を「no word as written is in `$'…'` or `$"…"` form」の場合だけ data command とする。いまは、展開する `$` か置換がある語でも data command から外れる(`:1040`)
- V-2(LOW、/sync-docs 向け): self-review LOW 2 の 3 点は HEAD でもそのまま残っている。(1) `:167` の冒頭は「commit 1e032dea adds」と書くが、特別なパラメータの添字の判定(`SPECIAL_PARAMS`)は 43e73568 で入った(`git show 43e73568` に 2 か所あり、1e032dea には 0 か所)。(2) `:167` の (a) は止めすぎの範囲を「any `$` that expands (a variable, `$1`, `$@`, as well as `${…}`)」と書くが、`LD_EXP` は後ろが空白・終わり・閉じる `"` でない `$` すべてに立つ。`git commit -m "costs 5$, never sudo ls"` は base none/none、HEAD deny/deny、旧版 deny で、self-review の probe `fp01` と同じ結果だった。(3) `:166` の Trigger の末尾に句点がない。ほかに、`:167` の (c) の例に長さの添字 `$#x[` がない(任意)。`:166` の (b)〜(d) の例は HEAD でも書かれたとおり deny で(`cat <<EOF` の本文 `${HOME} sudo ls`、`echo ${x:-'sudo ls'}`、`stat f; echo 'sudo ls'`、`echo 'sudo ls'; stat f`)、旧版も deny。対照の `git commit -m 'use ${HOME}'` と `git commit -m "${msg}"` は none
- V-3(LOW、guard のコメント、持ち越し): ヘッダーの (b)(`:99-105`)は、展開しない `$` として「単一引用符と ANSI-C 文字列の中、エスケープされたもの、後ろが空白・タブ・改行・終わり・閉じる二重引用符のもの」を挙げるが、二重引用符の外の `$"…"`(locale の引用)を挙げていない。`lex_dollar` はこの形に `LD_ANSI` を立てて `LD_EXP` を 0 にする(`:510-518`)ので、ヘッダーのとおりに読むとメッセージがデータにならないはずの `git commit -m $"never sudo ls"` は、HEAD でも base でも none/none になる(旧版は deny)。振る舞いはこの PR で変わっておらず、`lex_dollar` のコメント(`:459-461`)は正しく書いている。guard のファイルを直すと pipeline の再実行になるので、`:160` の (e) に足して持ち越すか、次に guard を直すときにまとめる
- V-4(情報): plan の Scope は「PR #211 が D 節の `edge_sentinel_deny` などに置いた、この plan で none になる形は `edge_none` に移す」と書くが、移す行はなかった。base の 589 行はすべて HEAD に同じ順で残っていて(削除 0)、base の行で判定が変わったものもない。HEAD で判定が変わった 23 行はすべてこの PR で足した行で、`edge_none` の 9 行のうち 6 行が base deny から none になった。plan の Objective 3 と Test plan に名前のある `echo $=arr['$(sudo ls)']`、`echo "$arr['$(sudo ls)']"`、`echo ${arr['$(sudo ls)']}` はテストの配列にない。probe では 3 形とも HEAD deny/deny で、旧版も deny、後の 2 形は base でも deny、`$=arr` は base none。足すかどうかは /test に任せる

## Observational checks

probe(`mpv/probe.sh`、`mpv/judge.sh`、`mpv/analyze.py`、`mpv/docker/bundle.sh`)。判定は jq あり/なし。「旧版」は `tests/fixtures/guard-1c4cea5a/`、base は c3a9242e。guard の写しは `git show` で取り出し、`lib_json.sh` を同じディレクトリに置いた。

| 形 | base | HEAD(root と template、6 モード) | 旧版 |
| --- | --- | --- | --- |
| AC1 の単一引用符・`\${`・`$'…'` の 4 メッセージ、`rg 'foo$' 'sudo ' .` | deny/deny | none/none | deny/deny |
| AC1 の `rg -n 'sudo ' .`、`echo "${HOME}" 'sudo ls'`、推奨の HEREDOC の形 | none/none | none/none | deny/deny |
| AC1 の `git commit -m "${msg}"`、`git commit -m 'use ${HOME}'` | none/none | none/none | none/none |
| AC2 の `rg $x`、`rg "$x"`、`rg $(echo --pre)`、添字の 4 形、`-m $arr['$(sudo ls)']` | none/none | deny/deny | deny/deny |
| AC2 の `rg $'\x2d-pre'`、`${(e)…}` の 3 メッセージ | deny/deny | deny/deny | deny/deny |
| AC2b の 2 形 | deny/deny | deny/deny | none/none |
| S1b の `$@[`、`$#x[`、`$*[`、二重引用符の中で開く添字。S1 のバッククォートの rg | none/none | deny/deny | deny/deny |
| Test plan の端: `"foo$"` のメッセージと rg | `rg "foo$"` だけ deny | none/none | deny/deny |
| Test plan の端: `rg $1`、`rg "$@"`、`-m "use $1 …"`、`$a[$b[1]]'sudo ls'` | none/none | deny/deny | deny/deny |
| Test plan の端: `echo $a[ ; git push origin --force` | deny/deny | deny/deny | none/none |

40 形と推奨の HEREDOC の形の全部で、HEAD の root と template、jq あり・なし、6 モードの判定が期待どおりだった(食い違い 0)。

テストの配列 12 個(`former_ask` から `broken` と `intentional_fixes`)の 618 行を、配列の節を stub つきで source して取り出し、HEAD の root と template、base、旧版に jq あり・なしで渡した(約 4,900 回、8 並列で 4 分 30 秒)。

- 配列の期待値と HEAD の判定が違う行: 0(`broken` は none か deny、`intentional_fixes` は none とした)
- template と root が違う行: 0。HEAD の jq あり・なしが違う行: 0
- base から判定が変わった行: 23。none から deny が 17 行(B 節 group 11 の 13 行、`edge_sentinel_deny` に足した 4 行)、deny から none が 6 行(`edge_none` に足した 4 つのメッセージと `rg 'foo$'`・`rg "foo$"`)。23 行とも旧版は deny。self-review の数と同じ
- ubuntu:24.04(dash を sh に、mawk 1.3.4、jq なし)でも、40 形と推奨の HEREDOC の形、`broken` を除く 593 行が期待どおりだった(食い違い 0)

## Coverage gaps

- テストスイート(`tests/test-pre-bash-guard.sh`、`tests/test-lib-json.sh`、`run-verify.sh`)は実行していない。AC5、G 節の比較が実際に通るか、Test plan の mutation は /test が見る
- zsh と bash の振る舞い(添字の評価、`$[…]`、二重引用符の中で開く添字が引用符の後ろまで続くこと)は、/verify ではもう一度確かめていない。plan の Assumptions(`zsh -f` 5.9、bash 3.2)と self-review の `dqsub.zsh` による
- テストと plan と tech-debt にない形での旧版との比較はしていない(新しい回避の形は作らない)。self-review が未確認とした、引用符のない区切りのヒアドキュメントの本文の中の添字も調べていない
- gawk と BSD awk の判定は見ていない(macOS の `/usr/bin/awk` と ubuntu の mawk だけ。implementer の `mp/impl/probe-s1b-bwk.txt` が bwk の記録を持つ)
- この session で効いている guard は main のチェックアウトのものなので、merge 後に Claude Code で実際に効くかは確かめられない

## Verdict

- Verdict: partial-pass
- Verified: AC1〜AC4。AC1 の 10 形はどのモードでも none、AC2 の 12 形と AC2b の 2 形はどのモードでも deny(jq あり・なし、root と template)。AC2 は旧版も deny、AC2b は旧版が通す。どの形も plan が名前を挙げた配列にある。`ac3` の 29 形と `intentional_fixes` の 13 件は base から変わらず、G 節の比較を静的にたどり直すと、旧版 deny で HEAD none の例は `intentional_fixes` と一致した。テストの配列の 618 行は HEAD で期待どおりで、base から判定が変わった 23 行はこの PR で足した行だけだった。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とバイト単位で一致、awk のプログラムに単一引用符はなく、plan の digest も一致した。self-review の LOW 3 は 899fff16 で直っている
- Partially verified: AC6。`:160` の Debt 列、`:163`、`:166` の (a) は解消済みになった。`:160` の残りの記述(V-1)と `:166`・`:167` の書き方(V-2)は、この cycle の /sync-docs で直す。V-3 は guard のコメントなので持ち越す
- Not verified: テストスイートの実行(AC5)、zsh と bash での再確認、テストと記録にない形での旧版との比較、gawk・BSD awk での判定、merge 後の Claude Code での実際の効き目
- partial にしたのは AC6 の残りがあるため。コードの AC は満たしていて、指摘は LOW 3 件と情報 1 件で、マージを止めるものではない
