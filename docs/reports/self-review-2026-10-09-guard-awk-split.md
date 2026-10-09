# Self-review report: guard-awk-split

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-awk-split.md
- Branch: refactor/guard-awk-split(base 0abfede5、HEAD 1e8dc7b4)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(コメントと記録の正確さ、読みやすさ、保守性、安全性)。対象は `git diff 0abfede5..HEAD`(14 ファイル、+3219/-2849)。テスト、静的解析、仕様との照合は行っていない。probe は `scratchpad/zg/run.sh` に 1 行 1 ファイルで渡した 6 形と、bash 3.2.57・zsh 5.9・macOS の /bin/dash で区切りと先頭の語を読ませた小さなスクリプト 8 本だけを使った

## Evidence reviewed

- awk のプログラムが変わっていないこと: base の `pre_bash_guard.sh` の 175〜1552 行目と、HEAD の `lex`・`commands`・`rules` を `cat` でつないだものから、コメントの行と空行(`^[[:space:]]*(#|$)`)を除いて `cmp` で比べた。結果は一致(コードの行はどちらも 982 行)。`diff` で見ても、違うのはファイル先頭のコメント 3 つ、`new_ctx` の前・`lex_redir` の区切り・`data_first_ok`・`cmd_pos` のコメント、消した単一引用符の注記だけだった。
- `.sh` のコメント以外の差分: `git diff -U0 0abfede5 -- .claude/hooks/pre_bash_guard.sh` のうち `#` で始まらない追加の行は、`awk -f` 3 つで呼ぶ `rule=` の 1 行だけ。stdin に命令を渡すこと、`2>/dev/null`、`|| awk_status=$?`、その下の fallback は base と同じ。
- root と template の写し: guard の 4 ファイル、`lib_json.sh`、`check-template.sh` はどれも `cmp` で一致。git の mode は `.sh` が 100755、`.awk` が 100644 で、両側で同じ。
- 行数: `.sh` 230、`lex` 618、`commands` 336、`rules` 468。
- S3 のコメントを、それが説明するコードと照らした。`new_ctx` の配列は `lex_redir:504`・`read_heredocs:513-515`・`end_cmd:83-87`・`pipe_close:95`・`heredoc_done:579`・`pipe_decide:326-335`。`cmd_pos` の `EXEC_SEEN` は `cmd_pos:125,134`、`end_cmd:76`、`BEGIN`(rules:437)。`cmd_pos` を呼ぶのは `judge`(rules:21)の 1 か所だけ。`data_first_ok` は `commands:109-114`、`lex_ansi` は `lex:393-409`、`lex_dollar` の `$"…"` の分岐は `lex:349-356`。
- 区切りの読み方(`scratchpad/rv/t1〜t7.sh`): `$'\x45'` は bash と zsh で `E` になり、dash は解かない。`$"EOF"` は bash で `EOF`、zsh と dash で `$EOF`。`${x}` は 3 つとも文字のまま。バッククォートの区切りでは、`` `x` `` は 3 つとも文字のまま読み、`` `echo x` `` だけ dash が「Syntax error: EOF in backquote substitution」で止まる。
- probe(新版 jq あり/なし、旧版): `$'\echo' 'sudo ls'` は none/none と deny、`$'\x65cho' 'sudo ls'` は deny/deny と deny、`git commit -m "costs $ 5; never sudo ls"` は none/none と deny、`git commit -m $"never sudo ls"` は none/none と deny、`echo $=arr['$(sudo ls)']` は deny/deny と deny、`$"echo" 'sudo ls'` は none/none と deny。bash と zsh は `$'\echo'` を `033 c h o` と読む(`od -c`)。`$"echo" hi` は bash だと echo が動き、zsh と dash では `$echo` が見つからないと言って止まる(`t8.sh`)。
- テストの新しいコメントにある出典をたどった。triage(`docs/reports/cross-review-triage-guard-deny-only.md` の付録 cycle 2 の 4 と 5)、mutation M20 と M13 の系統(`docs/reports/test-2026-10-09-guard-msg-param-flag.md:93-105`)、`a9ef82b1`(allowlist のコミット)、group 1 の最後の 2 行、`edge_none` の「"5$" row above」(`:1145`)。どれも書いてあるとおりだった。
- `tests/test-pre-bash-guard.sh` は `set -u` だけなので、F 節の `out="$(…)"; rc=$?` は非 0 でも止まらず、`record_fail` まで届く。
- 新しい tech-debt の行(`:189`)と guard の行(`:160`): 区切りでない `|` を数えると、どちらも 7 フィールド(5 列)。新しい行の `docs/plans/active/…` へのリンクは、`scripts/archive-plan.sh:54-56` が `/pr` のときに書き換えるので、指摘にしていない。行が名前を挙げる `classify_language`、`tests/test-detect-changed-languages.sh`、`Language scope: full fallback` の文言(`run-verify.sh:98`。`run-static-verify.sh` は `run-verify.sh` を exec する)は実在する。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | (1。8c61c6cb で修正済み)`data_first_ok` のコメントは、「字句解析の値はシェルが見る値と同じ」の例外に ANSI-C の引用だけを挙げている。実際は `$"…"` も例外で、字句解析と bash は `$"echo"` を `echo` と読み、zsh と dash は `$echo` と読む。同じコミット(f593bbfc)で直した `lex_redir` のコメント(`lex.awk:490-491`)はこの違いを書いているので、2 つのコメントが食い違う。tech-debt の Debt (b) に足した「One escape goes the other way」の文も `$'\echo'` しか挙げていない | `commands.awk:98-102`。probe では `$"echo" 'sudo ls'` が新版で none/none、旧版で deny。`t8.sh` で zsh は「command not found: $echo」、dash は「$echo: not found」。`docs/tech-debt/README.md:160` の Debt (b) | コメントの例外に「`$"…"`, which zsh and dash read as a `$` before the quoted text (`$"echo"` is `$echo` to them)」を足す。Debt (b) の文にも同じ形を足す(`$'\echo'` と同じく、`$echo` という名前のコマンドがなければ何も動かない) |
| LOW | readability | (2。8c61c6cb で修正済み)`new_ctx` の前のコメントは「PLID[ctx] the id of the pipeline (CPL[cid] of each command in it)」と書く。`CPL` が付くのは `\|` で終わるコマンドだけで、パイプラインの最後のコマンドには付かない | `lex.awk:92-93`。`CPL` に書き込むのは `end_cmd` の `if (sep == "\|") { …; CPL[cid] = PLID[ctx] }`(`commands.awk:84`)だけ | 「(CPL[cid] of each of its commands that end with \|)」にする |
| LOW | readability | (3。8c61c6cb で修正済み)`lex_redir` の区切りのコメントの「(dash reports a syntax error)」が、バッククォートの区切り全部に当てはまるように読める。dash が止まるのは中に空白のある形だけ | `lex.awk:487`。macOS の /bin/dash で、`` cat <<`x` `` は bash・zsh と同じく文字のまま読む(`t3.sh`)。`` cat <<`echo x` `` は「Syntax error: EOF in backquote substitution」(`t5.sh`) | 「(dash reports a syntax error for some, such as `` `echo x` ``)」にする |
| LOW | readability | (4。8c61c6cb で修正済み)`.sh` のヘッダーに 2 点ある。(1) 「The awk program decides in two ways, and either one denies.」のあとのコロンがピリオドになり、その後にファイルの説明が入ったため、番号付きの一覧 1〜6 が導入の文から離れた。(2) Not covered に足した「printf reads its words as written」は、printf というコマンドの振る舞いの説明に読める。printf は書式の `\x..` や `%` を解釈する。語を書かれたとおりに読むのは guard の判定のほう(item 6(a)) | `pre_bash_guard.sh:12-17`、`:152-153` | (1) ファイルの説明を一覧の後ろか別の段落に移し、「two ways」の文をコロンで一覧につなぐ。(2) 「the guard reads a printf's words as written (item 6(a)), so a printf with a `$'…'` word … is not a data command」にする |
| LOW | maintainability | (5。8c61c6cb で修正済み)3 つの `.awk` の先頭のコメントにある「The shell no longer passes the program in single quotes」は、変更の経緯を書いた文で、`templates/base/` から下流に配られる。下流の読み手は、単一引用符で渡していた形を見たことがない。`commands` と `rules` では「so a single quote may appear in it」がなく、何のための文か分からない。読む順を書いた 5 行ほどの段落は、3 つの `.awk` と `.sh` の 2 か所の計 5 か所にあり、順やファイルを変えるときは 5 か所とも直す必要がある | `lex.awk:1-7`、`commands.awk:1-8`、`rules.awk:1-8`、`pre_bash_guard.sh:12-16`、`:178-186` | 単一引用符の文は今の状態で書く(例: 「Comments spell a single quote as SQ; the files themselves may hold one.」)。`commands` と `rules` からは消す。順の説明は `.sh` の呼び出しの上を正にして、`.awk` の側は「read by pre_bash_guard.sh, see the comment above its awk call」程度にする |
| LOW | maintainability | (6。8c61c6cb で修正済み)`check-template.sh` とそのテストのコメントが、追加した項目に追いついていない。(1) `required_files` の上のコメントは、hooks の中で guard の 5 ファイルだけを挙げる理由(欠けると黙って旧版の 4 規則になる)を書いていない。(2) `tests/test-check-template.sh` の「the 8 non-script entries」は、`pre_bash_guard.sh` と `lib_json.sh` というシェルスクリプト 2 つを含む。(3) `build_fixture` の説明は「every scripts/*.sh is an executable one-line stub」のままで、S2b が足した `.claude/hooks/*.sh` の分岐を書いていない | `scripts/check-template.sh:26-38`、`tests/test-check-template.sh:92`、`:143-144`、`:156` | (1) 「The guard's files are listed because a missing one makes the guard fall back to the previous guard's rules without any message」を 1 行足す。(2) 「non-`scripts/` entries」にする。(3) 「every scripts/*.sh and .claude/hooks/*.sh」にする |
| LOW | maintainability | (7。未修正。/sync-docs に回した)tech-debt の guard の行で、(d) と (e) の閉じ方が列によって揃っていない。(1) Debt と Trigger の (d) は取り消し線で閉じたが、Impact の (d)「The file is harder to read and change than the guideline intends.」はそのまま残り、まだ開いているように読める。PR #213 は、閉じた項目を Impact の列でも取り消し線にしていた(Impact (b) の rg の変数の語)。(2) Debt (e) は「only the code items stay open」と書いたすぐ後に、コードではない `SQ` の綴りの項目を挙げている。Trigger (e) は `read_body` と `end_cmd` の規則しか挙げないので、`SQ` の項目を片づけるきっかけがない | `docs/tech-debt/README.md:160` の Impact (d)、Debt (e) の「Status after refactor/guard-awk-split」、Trigger (e) | (1) Impact (d) を取り消し線にして「closed in refactor/guard-awk-split」を添える。(2) Debt (e) を「only the code items and the `SQ` spelling stay open」にし、Trigger (e) に「or the next change to an awk comment that spells a single quote」を足すか、`SQ` は残してよいと決めて Debt から外す |

CRITICAL、HIGH、MEDIUM はない。表の Evidence の行番号は、最初に review した 1e8dc7b4 のもの。各行の先頭の括弧は、下の「8c61c6cb の再 review」の結果。

細かい点(直さなくてよい): `lex.awk` と `commands.awk` は末尾が空行で終わる。節の境目でバイト単位に切った結果で、`.sh` の中にあったときと同じ。`rule=` の行は約 190 桁ある。

### 直す範囲の線引きの外の指摘

なし。`awk -f` に渡すパスは `HOOK_DIR`(`cd "$(dirname "$0")" && pwd` の結果で、`/` で始まる)を引用符で囲んでいる。`-` や `name=value` に化けることはない。`.awk` を書き換えられる人は今までも `.sh` を書き換えられたので、分けたことで新しく攻撃できる面は増えていない。`.awk` が欠けたときに黙って fallback に落ちるのは plan の Design decisions が選んだ振る舞いで、`check-template.sh` と F 節の 2 つの検査が見えるようにしている。

## 8c61c6cb の再 review(S5)

対象は 8c61c6cb(LOW 1〜6 の修正、コメントだけ)。HEAD の f5cc1d1d は plan の Progress に 2 行足しただけ(`git diff --stat 8c61c6cb f5cc1d1d`)。

確かめたこと:

- コメント以外の行: `git diff -U0 1e8dc7b4 8c61c6cb` からこの report と `docs/insights/` を除き、追加・削除の行のうち `^[-+][[:space:]]*#` でないものを数えた。0 行。
- awk のコード: 8c61c6cb の 3 つの `.awk` をつないでコメントと空行を除いたものは、base(0abfede5)の 175〜1552 行目から同じく除いたものと `cmp` で一致。
- 写し: guard の 4 ファイルと `check-template.sh` は、8c61c6cb で root と `templates/base/` が一致。
- dash の区切り(`scratchpad/rv/t9〜t11.sh`): `` `echo<タブ>x` `` と `` `x y` `` は「Syntax error: EOF in backquote substitution」、`` a`b`c `` は文字のまま読む。bash と zsh は 3 つとも文字のまま。

LOW ごとの結果:

| LOW | 結果 | 根拠 |
| --- | --- | --- |
| 1 `data_first_ok` | 修正済み | 例外を 2 つに分け、2 つ目に `$"…"` を書いた。「字句解析と bash は中身の文字、zsh と dash は `$` + 中身」は `lex_redir` のコメントと揃い、`t2.sh`・`t8.sh` の結果とも合う。「`$"echo" SQsudo lsSQ gets a data region where the previous guard denies it`」は、最初の review の probe(新版 none/none、旧版 deny)と合う。コードは変わっていないので判定も同じ |
| 2 `new_ctx` | 修正済み | 「(CPL[cid] of each of its commands that end with \|)」。`end_cmd` の 1 か所の書き込み(`commands.awk`、`if (sep == "\|")` の分岐)と合う |
| 3 `lex_redir` | 修正済み | 「dash does too for `x`, but reports a syntax error when a space or a tab is inside, as in `echo x`」。`t3`・`t5`・`t9`・`t10`・`t11` の結果と合う |
| 4 `.sh` のヘッダー | 修正済み | (1) 「The awk program (three .awk files …; see the comment above the awk call below) decides in two ways, and either one denies:」で、コロンが一覧 1〜6 につながった。(2) 「The guard reads the words of a printf as written (item 6(a))」で、主語が guard になった |
| 5 `.awk` の先頭のコメント | 修正済み | 「no longer」の文は 3 つとも消え、読み方は `.sh` の呼び出しの上のコメントを指すだけになった。順と各ファイルの中身は、そのコメント 1 か所に集めた。`rules.awk` の「the action that collects the input」は `rules.awk` の `{ IN = (NR == 1) ? $0 : IN "\001" $0 }` と合う。単一引用符を書いてよいことは書かなくなったが、`SQ` の綴りの件は tech-debt の (e) に残っているので、指摘にしない |
| 6 `check-template.sh` とテスト | 修正済み | (1) `required_files` の上に、`.awk` が 1 つ欠けると何も言わずに旧版の 4 規則へ落ちるので 5 ファイルを挙げる、という理由を足した(root と template の両方)。(2) 「the 8 entries outside scripts/ (two of them … are shell scripts)」。(3) `build_fixture` の説明に `.claude/hooks/*.sh` を足した。`:156` の case の分岐と合う |
| 7 tech-debt | 未修正 | coordinator の判断で `/sync-docs` に回した。Debt (b) に `$"echo" 'sudo ls'` の形を足すことも含む。LOW のまま表に残す |

新しい指摘: なし。細かい点(直さなくてよい)が 2 つある。`.sh` の呼び出しの上のコメントは `rules.awk` の中身を「BEGIN …, and END」と書き、`rules.awk` の先頭のコメントにある入力を集める action を挙げていない。`check-template.sh` に足した理由は `.awk` が欠けた場合だけを書いている。`lib_json.sh` や `pre_bash_guard.sh` が欠けたときはシェルのエラーで止まり、黙って落ちることはないので、書いてあることに誤りはない。

insight のイベントは足していない。skill が求めるのは review の 1 回の実行につき 1 行で、この再 review は同じ cycle 1 の report に追記するものだから。

## Positive notes

- S1 を移すだけにし、コメントを S2 と S3 に分けたので、「コードの行は 1 行も変わっていない」をコメントを除いた `cmp` 1 回で確かめられる。この review でも取り直して一致した。
- F 節の 2 つの検査で、分割で新しくできた壊れ方を 2 つとも押さえている。1 つは `.awk` が欠けたとき(旧版の 4 規則が決めたことを、awk の経路では none の `echo 'never use sudo here'` が deny になることで示す)、もう 1 つは 3 つをつないだ構文。後者は awk の stderr を捨てないので、構文エラーが失敗のメッセージに出る。guard 本体の `2>/dev/null` で構文エラーが隠れる穴を、テストの側でふさいでいる。
- S3 で直したテストのコメントは、出典を cycle の番号ではなく triage のファイル名・番号やコミットで書くようになり、`git log` と `grep` でたどれる。
- 足したテストの行は、どれも理由と旧版の判定をコメントに書いている(「The old guard denies all four (the sudo substring)」など)。G 節の比較と合わせて読める。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しく先送りにするものはない。LOW 1〜6 は 8c61c6cb で直った。LOW 7 はこの PR の `/sync-docs` で直す記録の書き方なので、先送りにはしない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 1〜6 は 8c61c6cb で修正済みで、コメント以外の行は変わっていない。残る LOW 7 は tech-debt の書き方で、`/sync-docs` に回した)
- Follow-ups:
  - `/sync-docs` 向け: LOW 7(`docs/tech-debt/README.md:160` の Impact (d)、Debt (e) の「only the code items stay open」、Trigger (e) と `SQ` の項目)。あわせて Debt (b) の「One escape goes the other way」に `$"echo" 'sudo ls'` を足す。
  - probe の注意: 見張りの語を含む形は Bash のコマンド文字列に書かず、Write でファイルに書いてから `run.sh` に渡す。`run.sh` は 1 ファイルを 1 つのコマンドとして読むので、1 行に 1 形のファイルをそのまま渡すと、全行が改行でつながった 1 つのコマンドになる。
