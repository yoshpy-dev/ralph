# Self-review report: guard-deny-only

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md
- Branch: fix/guard-deny-only(base 1c4cea5a、HEAD 291292db)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけを見た(安全性、コメントの正確さ、読みやすさ、例外処理、性能)。対象は `git diff origin/main...HEAD`(14 ファイル、+3582/-431)。`templates/base/` の 4 ファイルは root と同じ内容であることを `diff` で確かめたので、root 側を読んだ。テスト、静的解析、仕様との照合は行っていない

## Evidence reviewed

- `.claude/hooks/pre_bash_guard.sh`(991 行)と `.claude/hooks/lib_json.sh` を全行読んだ。旧版は `tests/fixtures/guard-1c4cea5a/` を読んだ。
- 旧版と新版の比較: 両方の guard を scratchpad に写し、payload を `jq -nc --arg` で組んで、新版の jq あり・jq なし、旧版の jq あり・jq なしの 4 通りに同じ例を渡した。jq なしの PATH は、テストと同じく jq を除いた道具の symlink だけにした。比較に使った例はテストの例の集まりの外から取った 49 件。新版の jq あり・なしの判定は全件で一致した(PATHDIFF 0 件)。
- 実行されることの確認: 下の H-1 と M-1 の形のうち 6 つは、危険な部分を `echo` に置き換えて実際に走らせ、文字列がコマンドとして実行されることを確かめた(`env -S'…'`、`csh -c`、zsh の `=echo`、zsh の `source <(…)`、`find -exec sh -c`、`builtin eval`)。
- Bash ツールのシェル: このマシンの Claude Code は `/bin/zsh` でコマンドを走らせる(親プロセスは `claude`、`$options[equals]` は `on`、`$options[interactivecomments]` は `on`)。
- 性能: macOS の awk(BWK 20200816)で、200 KB の長い入力 6 種(`$'…'`、バッククォートの中のバックスラッシュ、ダブルクォートの中の `\$`、プレーンな文字列、短い語の並び、ヒアドキュメント)を新旧の guard に渡して時間を測った。
- `lib_json.sh`: jq がないときの取り出しで、`\" \\ \/ \n \t \r \b \f` と `\u0000`〜`\u007F` を戻し、`\u0080` 以上はそのまま残すことを読んで確かめた。awk も jq もないときの sed だけの経路(13b36abd)も読んだ。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| HIGH | security | 文字列をまるごとコマンドとして走らせる形を、旧版は deny にし、新版は通す。比較した 49 件のうち 44 件が「旧版 deny、新版 none」(M-1 の 3 件を含む)で、jq あり・なしのどちらでも同じ結果だった。たとえば、テストの外から取った例 `watch 'git push --force'` は両方の経路で none になる。plan の Assumptions は「deny が旧版より減るのは誤検知の形だけ」と決めていて、AC7 にもチェックが入っている。しかし、そのテスト(G 節)はテスト自身の例の集まりしか比べていない。ヘッダーの 38〜39 行目「This covers find -exec, watch, flock, chroot, and other runners」も、`watch` がふつう受け取るのは引用符で囲んだ 1 つの文字列なので、事実より強い。失敗の筋書き: エージェントが `find . -exec sh -c '<deny される形>' \;` を書くと、旧版は止めたが、新版は確認も deny もなしに走らせる。 | 原因は 5 か所ある。(1) `scan_words`(`.claude/hooks/pre_bash_guard.sh:682-694`)は、値がちょうど `sudo` か `git` の語しか見ないので、`'sudo ls'` のような 1 語の文字列は素通りする(`:686-687`)。(2) `-c` を読み直すシェルが `sh bash zsh dash ksh` の 5 つだけ(`:671`)。macOS に標準で入っている `csh`・`tcsh` などは対象外。(3) `builtin`・`source`・`.` を知らず、`sh <(…)` のプロセス置換は script の引数として扱う(`shell_rules`、`:734`)。(4) `env -S` は値が別の語のときだけ読み、`-S'…'` のように値がくっついていると `:622` で読み飛ばす。(5) `git` が NOEXEC にあり(`:914`)、`git_rules`(`:749-772`)は push・reset・commit・merge・rebase・am しか見ない。そのため、コマンドを走らせる git の形(`rebase -x`、`submodule foreach`、`bisect run`、`-c alias.x='!…'`、`-c core.pager=…`)が素通りする。`man -P` も NOEXEC の中にある。計測の一覧は下の「H-1 の計測」。 | 実行器の一覧は作らないという plan の方針のまま直せる。NOEXEC にないコマンドについて、後ろの語のうち空白か改行を含むものをそれぞれ `queue("c", …, CD[ctx] + 1)` で読み直す。git も同じように扱う: 未知のサブコマンドの後ろの語と、`-c <key>=<value>` の値を読み直す。`cmd_pos` には `builtin` を、shell_rules には `source`・`.` とプロセス置換の引数(stdin として読む)を足す。`skip_env` は `-S` の値がくっついた形を読む。キューの上限(`8 * N + 65536`)は、語を 1 回ずつ足すだけなので超えない。この種類の形(上の 5 つの原因から 1 つずつ)を AC7 の例の集まりと B 節に足し、旧版との比較がこの種類も覆うようにする。38〜39 行目のコメントは、直したあとの挙動に合わせて書き直す |
| MEDIUM | security | zsh の `=コマンド名` の展開で、名前の比較を外れる。このマシンの Bash ツールは zsh で動いていて、`EQUALS` は on になっている。`=sudo ls` は zsh が `/usr/bin/sudo ls` に展開する。旧版は `sudo ` の部分一致で deny にしていたが、新版は `base("=sudo")` が `=sudo` のままなので none になる。`=git push --force` と `=git reset --hard` も同じで、旧版 deny、新版 none(jq あり・なしとも)。失敗の筋書き: zsh を使う開発者の環境で、`=` の付いた形が止まらない。 | `base()`(`.claude/hooks/pre_bash_guard.sh:521-525`)はディレクトリを外すだけ。`scan_words`(`:686`)と `judge`(`:668`)が同じ `base()` を使う。zsh で `zsh -fc '=echo x'` が `echo` を走らせることを確かめた | 元の文字(`WR`)が引用符なしの `=` で始まる語は、比べる前に先頭の `=` を 1 つ外す。zsh は `"=ls"` と `\=ls` を展開しないことを確かめた(`zsh -fc 'echo "=ls" =ls \=ls'` は `=ls /bin/ls =ls`)。`=sudo ls` と `=git push --force` を B 節と AC7 の例に足す。ヘッダー(`:29-34`)に、zsh の `=` を扱うことを 1 行書く |
| LOW | maintainability | ヘッダーの「Not covered」(`:72-77`)が、新旧どちらの guard も見ない静的な展開を挙げていない。挙げているのは実行時にしか分からないもの(変数、エイリアスなど)と、case・算術・配列だけ。ブレース展開(`{a,b}`)、パス名の展開(`?`・`[…]`)、`$'…'` の 16 進・8 進のエスケープ(`lex_ansi` は `\n \t \r` しか戻さない、`:351-367`)は、コマンドの文字列から読めるのに対象外で、そのことがどこにも書いていない。旧版も同じく通すので、後退ではない | `:72-77`、`:348-350` のコメント、`docs/tech-debt/README.md` の guard の限界の行(b) | 「Not covered」の一覧と tech-debt の行(b)に、この 3 つを 1 行ずつ足す。直すかどうかは別に決めてよい |
| LOW | maintainability | `unbq()`(`:384-393`)は、バックスラッシュが現れるたびに `t = substr(t, k + 2)` と `out = out …` でコピーするので、入力の長さの 2 乗の時間がかかる。ほかの字句解析の関数は、512 文字の窓とバッファを使ってこれを避けている。バッククォートの中にバックスラッシュが並ぶ入力を渡すと、50 KB で 0.30 秒、100 KB で 0.78 秒、200 KB で 2.56 秒、400 KB で 9.8 秒かかる(旧版はそれぞれ 0.07・0.09・0.14・0.28 秒)。AC8 の上限の 5 秒は 200 KB では守れているが、伸び方が 2 乗 | python で payload を組んで計測した(`json.dumps`、macOS の awk)。ほかの 5 種は 200 KB で 0.15〜1.46 秒 | `unbq` も `lex_word` と同じように、512 文字ごとにまとめて連結するか、`find()` の窓を使う形に直す。直さずに残すなら、tech-debt に 1 行足す |

### H-1 の計測(旧版 deny、新版 none。jq あり・jq なしとも)

原因ごとに件数だけを挙げる。どれも旧版の `*"sudo "*`・`*"git push --force"*`・`*"git reset --hard"*` の部分一致で deny になっていた。

- 文字列を受け取る実行器(`watch`、`find -exec sh -c`、`flock -c`、`su -c`、`script -c`、`parallel`、`tmux new-session`): 13 件
- 5 つ以外のシェルの `-c`(`csh`、`tcsh`、`fish`、`ash`、`busybox sh`、`mksh`): 6 件
- `builtin eval`、`source`・`.`・`sh`・`bash` に渡すプロセス置換、`. /dev/stdin`、`bash /dev/stdin` へのヒアドキュメント: 8 件
- コマンドを走らせる git の形(`rebase -x`・`--exec`、`submodule foreach`、`bisect run`、`filter-branch --tree-filter`、`-c alias.*`、`-c core.pager`): 8 件
- NOEXEC の中で引数を走らせるもの(`man -P`、`rg --pre`)と、`LESSOPEN=` の前置き: 3 件
- `env -S` に値がくっついた形と、`xargs … sh -c`: 3 件
- zsh の `=コマンド名`(M-1): 3 件

旧版も新版も通した 5 件は、ブレース展開、パス名の展開、`$'…'` の 16 進のエスケープ(L-1)。

## Positive notes

- `lib_json.sh` の jq なしの経路は、ASCII の範囲で jq と同じ値を返すようになった。49 件の比較で、新版の jq あり・なしの判定は全件で一致した。awk も jq もないときに sed だけの戻しに落とす分岐(13b36abd)のおかげで、guard が「何も止めない」状態にならない。
- 止める側に倒す作りに抜けがない。awk の終了コードを `|| awk_status=$?` で別の変数に取っていて、`set -e` の下でも捨てていない。awk が使えないときは旧版の 4 規則に戻る。深さ・キュー・入れ子の上限を超えると deny になり、理由の文言も分かれている。
- `safe_heredoc_msg`(`:878-898`)は、区切りの直後の改行、区切りだけの行、最後の `)"` までを完全に一致で比べる。少しでも違う形は deny になる。
- 長い入力への手当て(512 文字の窓、`buf` での連結、`base()` の 256 文字の打ち切り、`scan_words` を次の `git` で区切ること)にはそれぞれ理由のコメントがあり、200 KB の 6 種のうち 5 種は 1.5 秒以内に終わった。
- deny の理由の文言は定数で、payload の値を含まない。`emit_deny` のエスケープの不足は起きない。
- 秘密情報、デバッグ用の出力、コメントアウトしたコードは差分にない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(このレビューでは行を足していない。H-1 と M-1 は merge 前に直すことを勧めている。L-1 と L-2 を直さずに残すなら、`docs/tech-debt/README.md` の guard の限界の行に足す。)_

## Recommendation(cycle 1)

- Merge: no-merge(HIGH 1 件、MEDIUM 1 件。CRITICAL はない)。**下の cycle 2(S2c)で上書き。**
- Follow-ups:
  - H-1: 後ろの語の読み直しを、値がちょうど `sudo`・`git` の語から「空白か改行を含む語」に広げる。git・`builtin`・`source`・`.`・プロセス置換・`env -S` に値がくっついた形の扱いを足し、この種類の形を AC7 の例の集まりに入れる
  - M-1: `base()` で zsh の先頭の `=` を外す
  - L-1: 「Not covered」の一覧と tech-debt の行(b)に、静的な展開の 3 つを足す
  - L-2: `unbq()` の連結をまとめる。残すなら tech-debt に書く
  - 直したら、tech-debt の guard の限界の行の(a)(誤検知)を見直す。読み直しを広げると、NOEXEC にないコマンドの引数の文字列が新しく deny になりうるため

---

## S2c cycle(cycle 2)

- Date: 2026-10-07
- Reviewer: reviewer subagent (Claude)、cycle 2
- 対象: `git diff origin/main...HEAD`(HEAD 9eca7573 は main を取り込んだマージ。branch 由来の差分は df0a50d5 までと同じ)。S2c のコミットは df0a50d5。S2c が触ったのは `pre_bash_guard.sh`(root と template、同一)、`docs/tech-debt/README.md`、`tests/test-pre-bash-guard.sh` だけ。`lib_json.sh`・`post_edit_verify.sh`・`.codex/README.md` は cycle 1 のまま(内容は root と template で一致を確認)。

### S2c が何を変えたか

字句解析の規則に加えて、旧版の 4 つの部分一致規則(`sudo` の語境界つき `(^|[^A-Za-z0-9_.-])sudo[[:space:]]`、`git push --force`、`git push -f`、`git reset --hard`、`git commit` 後の最初の `-m "` に続く `$(`・バッククォート)を生のコマンド全体に当てる見張り(`sentinel`/`sentinel_str`、`.claude/hooks/pre_bash_guard.sh:1143-1174`)を足した。一致はデータ区間の中にあるときだけ無視する。データ区間は深さ 0 のトップレベルだけで作り(`DCTX`、`new_ctx:209`)、`$(...)`・バッククォート・読み直しのテキストの中は常に非データ。区間は (a) データを読むだけのコマンド(`DATACMD`、`:1196`)の引数で出力がファイルに落ちないもの(`stage_note`/`pipe_decide`/`redir_safe`、`:796-843`)、(b) `git commit`・`git tag` の置換なしの `-m` 値、(c) データとして読むヒアドキュメントの本文(`heredoc_done`/`read_body`)、(d) 深さ 0 のコメント。区間の索引は 512 字ブロック(`add_data`/`in_data`、`:760-780`)。あわせて `cname()` が引用符なしの先頭 `=` を落とし(`:609-613`)、`lex_bq()` が 512 字単位でまとめて連結する(`:439-459`)。

### 検証(新旧 guard を同じ payload で比較。jq ありの経路)

- 見張りが止める側: 旧版の deny を実行の位置で保つか。H-1 と M-1 の全区分を例の外から当て、すべて `new=deny` を確認した。`find . -exec sh -c '…'`、`watch '…'`、`csh -c`/`tcsh -c`、`source <(…)`/`. <(…)`、`builtin eval`、`git rebase -x`/`git submodule foreach`、`flock`/`su -c`/`script -c`/`parallel`/`tmux new-session`、`man -P`、`fish`/`ash`/`busybox sh`/`mksh` の `-c`、`env -S'…'`、`chroot`、`=sudo ls`/`=git push --force`/`=git reset --hard`。
- データ区間が広すぎないか(実行の位置の文字が外れないか)。危険な実行形を当て、すべて `new=deny`: 読むだけのコマンドの出力を `sh`・`bash`・`eval`・`xargs`・`sed`・`sort`・`cat|sh` に流す形、`echo … > >(sh)`、ファイルへの `>`・`>>`・`tee x.sh`、`echo "$(sudo ls)"`(置換は区間から除かれ、字句解析でも読み直す)、`sh <<EOF`/`bash -s <<EOF` のヒアドキュメント、`git commit -F -` に引用符なしの区切りで置換つき本文を流す形、`git commit -m "$(id)"`/`-am "$(id)"`/`-m"$(id)"`(後ろ 2 つは旧版 none、新版 deny で強くなった)、`git -c core.hooksPath=… commit`(同じく強くなった)。
- データ区間の例外が効くか(旧版の誤検知を外すか)。すべて `new=none`、かつ旧版は `deny`: `echo 'never use sudo here'`、`echo hi # sudo ls`、`grep -n 'git push --force' docs.md`、`echo 'sudo ls' > /dev/null`、`git commit -m 'drop --force and git reset --hard from docs'`、推奨の HEREDOC 形、引用符つき区切りのヒアドキュメントを `git commit -F -` に流して本文に `git push --force` を書いた形。
- 性能: macOS の BWK awk(20200816)で 200 KB の入力を測った。echo のデータ区間に一致を約 12,000 個詰めた 204 KB で 0.15〜0.23 秒、一致なしの 204 KB で 0.15 秒。見張りの全文走査と `in_data` のブロック索引に 2 乗の伸びはない。
- L-2 の確認: バッククォートの中にバックスラッシュを並べた入力で、50 KB 0.16 秒 / 100 KB 0.25 秒 / 200 KB 0.42 秒(cycle 1 は 0.30 / 0.78 / 2.56 秒)。線形になった。
- `DATACMD` の中身は実引数を実行しないコマンドだけ(`rg` は `--pre` を `stage_note:800` で除外)。デバッグ出力・TODO・コメントアウトは差分にない。

### cycle 1 の指摘の解消

| cycle 1 | 状態 | 根拠 |
| --- | --- | --- |
| HIGH(例の外の 44/49 が旧版 deny→新版 none) | 解消 | H-1 の全区分が見張りで `new=deny`。実行の位置の文字はデータ区間に入らない |
| MEDIUM(zsh の `=` 展開) | 解消 | `cname()` が先頭 `=` を落とし、見張りの `NONB` 境界で `=sudo` も当たる。`=sudo ls` ほか `new=deny` |
| LOW L-1(「Not covered」の記載漏れ) | 解消 | ヘッダー `:100-105` にブレース展開・パス名展開・`$'…'` の 16/8 進エスケープを明記 |
| LOW L-2(`unbq()` の 2 乗時間) | 解消 | `unbq` を削除、`lex_bq()` が 512 字単位で連結。上の計測で線形 |

### S2c の findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security(非後退) | 見張りの `sudo` 語境界 `NONB` に `-` と `.` が入るため、`my-sudo ls`・`x.sudo ls` が旧版 deny→新版 none になる。データ区間の例外ではないので、AC7 が例外として挙げる「一致がすべてデータ区間の中」から外れる。ただし `my-sudo`・`x.sudo` は別コマンドで、本物の `sudo`(先頭、空白・`/` 区切り、`/usr/bin/sudo`、`./sudo`)は deny のままなので、安全上の後退ではなく誤検知の削減。ヘッダー `:40`(sentinel の sudo の説明)と一致 | `.claude/hooks/pre_bash_guard.sh:1203`(`NONB`)、`:1146`。probe: `my-sudo ls`/`x.sudo ls` は `new=none`、`./sudo ls`/`/usr/bin/sudo ls` は `new=deny` | /verify の AC7 比較コーパスに `-`/`.` 前置きの非 sudo を 1 つ入れて、この差を意図どおりと固定する。コード変更は不要 |
| LOW | maintainability(既知、/sync-docs) | `docs/tech-debt/README.md` の guard の限界の行が S2c 後に古い。「991 行」と書くが今は 1286 行。参照する行番号はすべて S2c 前のもの(`scan_words` :677-694→:879-891、`judge` :674→:853、`noexec_list` :914→:1191、`git_rules` :749-772→:946、フォールバック :950-965→:1249-1260、awk 本体 :100-948→:128-1242)。見張りが引き継ぐ誤検知(`cp sudo dest`、`apt-get install sudo vim`、`git push --force-if-includes`、読むだけのコマンドの出力を sed・sort に流す形)を (a) に挙げていない。4 つの誤検知は実測で確認(いずれも `old=deny`/`new=deny`) | `docs/tech-debt/README.md`(guard-limits 行);`.claude/hooks/pre_bash_guard.sh`(行数と行番号) | /sync-docs で行番号を貼り直し、「991 行」を 1286 行に直し、見張りの 4 誤検知を (a) に足す。タスクで既知として渡された項目であり、ここでは確認のみ |

cycle 1 の Positive notes はそのまま有効。S2c はそれを弱めていない。

## Recommendation(現時点)

- Merge: merge(CRITICAL・HIGH なし。cycle 1 の HIGH/MEDIUM は S2c が解消。残るは LOW 2 件で、1 件は非後退の観測、もう 1 件は /sync-docs で閉じる既知項目)
- Follow-ups:
  - /verify: AC2・AC3・AC7 の box が未チェック(仕様適合は /verify の領分)。AC7 の比較コーパスに `-`/`.` 前置きの非 sudo を 1 つ入れて、上の LOW を意図どおりと固定する
  - /sync-docs: tech-debt の guard の限界の行を貼り直す(行数 991→1286、行番号、見張りの 4 誤検知)

---

## pipeline cycle 2(cross-review の指摘の修正 46806dc9)

- Date: 2026-10-08
- Reviewer: reviewer subagent (Claude)。pipeline の 2 周目で、上限 2 の最後の周。上の「S2c cycle(cycle 2)」は 1 周目の中で 2 回目に回した self-review なので、この節とは別もの。ID は前の節と混ざらないよう `P2-` で始めた
- 対象: `git diff origin/main...HEAD`(merge-base f423f230、HEAD 0f46fb22)。コードの変更は 46806dc9 だけで、`.claude/hooks/pre_bash_guard.sh`(root と template は `cmp` で同一)と `tests/test-pre-bash-guard.sh` を変えている。0f46fb22 は plan の Progress に 2 行を足しただけ
- していないこと: テストと静的解析は流していない。cross-review の指摘と同じ型の形を、テストの外で新しく作って探すこと(依頼の 5 番)もしていない。下の確認は、コードを読んだ結果と、triage の case ファイル、AC3 の形、危険な部分を持たない `echo` の形で行った

### 46806dc9 が変えたこと

- `NODATA`: 次のどれかがあると `in_data()`(`:835`)が常に 0 を返し、そのコマンドにはデータ区間が 1 つもなくなる。トップレベルの `(` と `)`(`lex_cmds` の `:267`・`:272`)、1 語目が予約語の単純コマンド(`RESW`、`end_cmd` の `:692`)、リダイレクトのついた `exec`(`EXEC_SEEN`、`:697`)
- 未読のヒアドキュメントを文脈ごとに持つ(`HPQ`/`HPN`。`lex_redir` の `:525`、`read_heredocs(ctx)` の `:534-537`)
- 区切りに引用符のない本文では、行継続をつないでから区切りと比べる(`read_body` の `:555-576`)。区切りの語の中の行継続は `LW_BSNL` で見分け、引用符として数えない(`:520-521`)。どちらの場合も、そのヒアドキュメントの本文はデータにしない(`:581`)
- `redir_safe()`(`:850-856`): fd の複製で安全とみなすのは `>&0`・`>&1`・`>&2`・`>&-` だけ。`&>` と `&>>`(`lex_redir` の `:490-492` で別の演算子に分けた)、`>|`、`<>` は安全とみなさない
- `printf -v` を読むだけのコマンドから外す(`stage_note` の `:869`)

### 確認したこと

1. triage の 6 つの穴: `xr1/` の case ファイル 54 件を `xr1/run.sh` で新旧の guard に、jq あり・なしで渡した。53 件は新版も旧版も deny/deny だった。残る `v-ml4.txt` だけは新版 none/none、旧版 deny/deny になる。これは一致がダブルクォートで囲んだ `cat` の引数の中にある形で、引数は表示されるだけなので、設計どおりのデータ区間にあたる。triage の 3 件と consult の 3 件は、どれも閉じている
2. `NODATA` の範囲: 立つのは `DCTX[ctx]` が 1 のとき、つまり主コマンドを深さ 0 で読んでいる間だけ(`MAIN` が 1 なのは `:1297-1299` の間だけ)。読み直しの文脈では立たない。`NODATA` を読むのは `in_data()` だけで、`in_data()` を呼ぶのは `sentinel()` と `sentinel_str()`(`:1215`・`:1232`・`:1240`)だけ。`sentinel()` はすべての字句解析と読み直しのあと(`:1313`)に走るので、どこで立ってもコマンド全体に効く。働きはデータ区間を消す向きだけなので、deny を弱めることはない。guard は呼び出しごとに awk を 1 回起こすので、別のコマンドに持ち越されることもない
3. 予約語: 見るのは単純コマンドの 1 語目の元の文字(`WR[ctx, 1]`)だけ。引数の位置の `if` も、引用符やバックスラッシュのついた `"if"`・`\if` も数えない。shell でもこれらは予約語にならない。リダイレクトの後ろの `if` は数えてしまうが、止める側に倒れるだけ。複合コマンドでは、閉じる語(`fi`・`done`・`esac`・`}`)か `)` が必ずコマンドの位置に来る。そのため、始まりを読み落としても閉じる側で `NODATA` が立つ
4. `<<-` の行継続: `lstrip_tabs(cmp)`(`:571`)は、つないだ 1 行の先頭のタブだけを外す。`echo` だけを書いた無害な 2 形を bash と zsh に渡した。続きの行の先頭にタブがある形では本文が続き、ない形ではそこで本文が終わった。どちらの shell も、つないだ行の先頭だけを外して区切りと比べている。guard の判定と同じ
5. AC3 の形: `redir_safe` と `printf -v` の変更が触れうる AC3 の形 9 件は、新版で none のまま。`/dev/null` へのリダイレクト、`2>&1`、`grep` へのパイプ、`printf` の引数の文字の `\n`、推奨の HEREDOC の形、コメント、`grep -n`、`&& grep`、引用符のない `echo` の 9 つ
6. awk の中の `'`: awk のコメントに `'` を 1 つ入れた写しを作ると、`sh -n` が構文エラー(rc 2)を出した。この写しの hook は何も出力せずに exit 2 で終わる。テストは全件に exit 0 を求める(`tests/test-pre-bash-guard.sh:1142`)ので、このように壊れると全件が赤になる。awk だけが壊れる形でも、awk がないときの旧版の規則が働き、AC3 の none は deny に、字句解析だけが止める形は none に変わるので、やはり赤になる
7. root と template の guard は `cmp` で同一。デバッグ出力、秘密情報、コメントアウトしたコードは差分にない

### pipeline cycle 2 の findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| P2-1 | LOW | maintainability | ヘッダーの方針の段落(`:97-103`)は、行継続でつないだヒアドキュメントの本文と、行継続を含む区切りの語も「コマンドのデータ区間をすべて消す」と書いている。実際に消えるのは、そのヒアドキュメントの本文だけ。`read_body` の `:581` は `HDZ` を立てないだけで、`NODATA` は立てないので、同じコマンドのほかのデータ区間は残る。`in_data()` のコメント(`:831-833`)は正しく、`NODATA` の理由に、グループと複合コマンド、`exec` のリダイレクトだけを挙げている。triage の 3 番も「つないだ本文にはデータ区間を与えない」と書いていて、コードと合う | 区切りに引用符のないヒアドキュメントで本文の行を行継続でつないだ形(`sr3/j1.txt`)と、区切りの語に行継続を入れた形(`sr3/j2.txt`)のあとに、`echo` の 1 行を置いた。新版は jq あり・なしとも none。`(true)` のあとに同じ 1 行を置いた形(`sr3/j3.txt`)は deny | 段落を「そのヒアドキュメントの本文にデータ区間を与えない」に直す。確認の 4 のとおり、guard の区切りの判定は bash・zsh と同じなので、本文だけを外す今のコードで足りる |
| P2-2 | LOW | maintainability | ヘッダーの (6)(a)(`:79-85`)が 46806dc9 の前のまま。出力の行き先に「or another fd」と書くが、いま安全とみなすのは fd 0〜2 への複製と、閉じる `>&-` だけ(`redir_safe` の `:853`)。`rg without --pre` は書いてあるのに、`printf -v` の除外(`:869`)は書いていない。`&>`・`&>>`・`>|`・`<>` を安全とみなさないことも書いていない | `:84`、`:853`、`:869` | (a) の行き先を「/dev/null、/dev/stderr、fd 0〜2 への複製、`>&-`」に直し、`printf without -v` を足す |
| P2-3 | LOW | unnecessary-change | `&>` と `&>>` を別の演算子に分けた(`lex_redir` の `:488-492`)ことで増えたのは、`/dev/null` と `/dev/stderr` に向けた形の deny だけだった。095af1d7 の guard も、`&>` の後ろを `>` として読んで行き先を比べていたので、ファイルに向けた `&>` と `&>>` はすでに deny にしていた。いまは `echo … &>/dev/null` が deny で、同じ意味の `echo … >/dev/null 2>&1` は none。コメントの理由(標準出力と標準エラー出力をファイルに送る)も、`/dev/null` には当たらない。テストは `&>/dev/null` を「安全でないリダイレクト」として deny に固定している(`tests/test-pre-bash-guard.sh:571`・`:576`) | `echo` に同じ引数を与え、リダイレクトだけを変えて比べた(jq あり)。`&>out.txt` と `&>>log` は 095af1d7 でも新版でも deny。`&>/dev/null` と `&>/dev/stderr` は 095af1d7 で none、新版で deny。`>/dev/null 2>&1` はどちらも none。旧版(1c4cea5a)はどれも deny なので、AC7 には反しない | `redir_safe` で `&>`・`&>>` を `>`・`>>` と同じに扱う(行き先が `/dev/null` か `/dev/stderr` のときだけ安全)。テストの `:576` は edge_none に移す。この周で直さないなら、P2-4 の行に誤検知として書く |
| P2-4 | LOW | maintainability | tech-debt の guard の限界の行(`docs/tech-debt/README.md:160`)が 46806dc9 で古くなった。(a) の誤検知の説明は、DATACMD の引数などのデータ区間の説明で終わっていて、データ区間が 1 つもなくなる場合を挙げていない。トップレベルのサブシェル、グループ、複合コマンド、リダイレクトのついた `exec`、P2-3 の `&>/dev/null` がそれにあたる。この誤検知は、エージェントがふつうに書く形(ファイルを回す `for` の中の `grep` など)にも当たる。行は「095af1d7 で測った」と書いている。(d) は 1286 行で awk の本体が 1114 行(128〜1243 行目)と書くが、いまは 1358 行で、awk の本体は 135〜1315 行目。テストのコメント(`:546`)は、deny に移した 2 件を「false none before」と書く。2 件は表示するだけの形なので、誤った none ではない。今回の方針の代価として deny になった | 無害な形で 095af1d7 と比べた(jq あり)。ファイルを回す `for` の中の `grep -n`、`if grep -q`、`(cd docs && grep -rn …)`、`echo … &>/dev/null` は、095af1d7 で none、新版で deny。サブシェルのない `cd docs && grep -rn …` と、`2>/dev/null \| head` のついた `grep` は、どちらも none。旧版はどれも deny | /sync-docs で、(a) にこの型を足し、計測したコミットと (d) の行数を直す。行番号ではなく関数名で指す今の方針は残す。テストの `:546` は、旧版も deny にしていたこと、データ区間を与えない規則の代価で deny に移したことが分かる書き方にする |
| P2-5 | LOW | security(未確認) | 未読のヒアドキュメントは、宣言した文脈の改行でしか読まれない。`$(...)` の中で宣言され、その文脈が改行の前に閉じると、その分は読まれないまま残る(`lex_cmds` の `:271` は `HPN[ctx]` を見ずに戻る)。続く行は外側の文脈がコマンドとして読み、トップレベルならデータ区間が付きうる。その行を shell が本文として読むのか、コマンドとして読むのかは確かめていないので、穴かどうかは分からない。ただ、行き先が分からないときはデータ区間を与えないという今回の方針から見ると、未読の分を黙って捨てる経路が 1 つ残っている | `:271`、`:283`、`:534-537`。新しい形での probe はしていない | 主コマンドを読んでいる間(`MAIN`)に、未読の分を残したまま文脈が閉じたら `NODATA` を立てる(`:271` の `return` の前と `:283` で `if (MAIN && HPN[ctx] > 0) NODATA = 1`)。止める側に倒れるだけなので、AC3 には触れない |
| P2-6 | LOW | readability | 読みやすさの小さな点が 4 つある。(1) `EXEC_SEEN` は `cmd_pos()` が副作用で立て(`:717`・`:726`)、`end_cmd()` が `judge()` のあとに読む(`:697`)。`cmd_pos()` のコメント(`:712-713`)にこの副作用がない。(2) awk の中に `'` を書けないという注意が、`lex_dollar` のコメントの中(`:378-379`)にしかない。この周の実装で一度踏んでいる。(3) `read_body` は行末のバックスラッシュを数える同じ 3 行を 2 回書いている(`:556-558`・`:565-567`)。(4) 文脈ごとの配列の一覧(`:200-211`)に、新しい `HPQ`/`HPN` がない。テストのヘッダーの B 節(`:27-36`)も、46806dc9 が足した形の種類を挙げていない | 各行番号 | (1) は `cmd_pos()` のコメントに 1 行足す。(2) は awk の始まり(`:136`)に 1 行書く。(3) は小さな関数にまとめる。(4) は一覧とヘッダーに 1 行ずつ足す |

### 前の節の指摘の状態

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| cycle 1 の HIGH・MEDIUM・L-1・L-2 | 解消のまま | S2c の節の表のとおり。46806dc9 はこれらの経路を変えていない |
| S2c の LOW(`my-sudo ls` の差) | 解消 | /verify の V-1 で意図どおりと固定し、tech-debt の行 (a) に書いてある |
| S2c の LOW(tech-debt の行が古い) | 一度解消し、また古くなった | 27581ff5 で /sync-docs が直した。46806dc9 のあとの状態は P2-4 |

### Tech debt identified

最後の周なので、ここで直さない LOW はそのまま持ち越しになる。この commit では `docs/tech-debt/README.md` を変えていない。/sync-docs で、次の 3 行を `docs/tech-debt/README.md:160` の guard の行にまとめてほしい。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| guard のヘッダーとテストのコメントが 46806dc9 の挙動とずれている(P2-1、P2-2、P2-6、P2-4 のテストのコメント) | コメントに合わせてコードを広げると、閉じた穴がまた開く | コメントだけの修正でもコードの変更になり、上限 2 の pipeline をもう 1 周回すことになる | 次に guard を変える PR | この節の P2-1、P2-2、P2-4、P2-6 |
| `&>`・`&>>` を `/dev/null` に向けた形と、トップレベルの複合コマンドの誤検知(P2-3、P2-4) | 無害なコマンドが止まり、言い換えが要る | 旧版も止めていた形で、AC7 には反しない。直すと最後の周を越える | 誤検知の報告 | この節の P2-3、P2-4 |
| 文脈が改行の前に閉じたときに残る未読のヒアドキュメント(P2-5) | 未確認。shell が続く行を本文として読むなら、データ区間の判定が shell とずれる | shell の振る舞いを確かめていない | 次に guard を変えるとき、またはこの形の shell の振る舞いを確かめたとき | この節の P2-5 |

### Recommendation(pipeline cycle 2、現時点)

- Merge: merge(CRITICAL・HIGH・MEDIUM はない。triage の 3 件と consult の 3 件は閉じている。LOW の 6 件は、旧版より弱くならない誤検知か、コメントと記録のずれか、未確認の 1 点)
- 上の 2 つの節にある Merge 行は、それぞれの時点の判定で、この行が現時点の判定
- Follow-ups:
  - /sync-docs: `docs/tech-debt/README.md:160` を P2-4 のとおりに直し、上の Tech debt の 3 行をその行にまとめる
  - /test: 46806dc9 が足した仕組みごとに、その仕組みだけを外した写しでテストが赤になるかを見る。仕組みは、`NODATA` の 3 つの立て方、文脈ごとのヒアドキュメント、本文の行継続のつなぎ、`LW_BSNL`、`redir_safe` の fd、`printf -v`
  - P2-3 と P2-5 は、直すなら次の PR で

### 未確認の点

- 依頼の 5 番(同じ型の残り): テストの外で新しい形を作って探すことはしていない。コードを読んで見つけたのは P2-5 だけ
- `RESW`(`:1269`)は閉じた一覧で、bash の `coproc` など、一覧にない予約語がある。それらが出力の行き先を変えるかどうかは確かめていない
- P2-5 の形を shell がどう読むか
