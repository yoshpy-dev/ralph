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

---

## pipeline cycle 2 の再実行(F2-1 の修正 4e829e34 のあと)

- Date: 2026-10-08
- Reviewer: reviewer subagent (Claude)。pipeline の 2 周目の中で、/test の F2-1 を直したあとに回し直した self-review。cycle は 2 のまま。ID は前の節と混ざらないよう `R2-` で始めた
- 対象: `git diff 8f852e4f..HEAD`(HEAD 5f044db2)。guard を変えたのは 0ef6fc4f・63b6743a(ヘッダーのコメントだけ)と 4e829e34(END の 1 行とコメント)。テストは 3e9afad6(tester が足した 6 形)と 4e829e34(B 節の 8 の 3 形)。残りは記録(verify と test の報告、insight、plan の Progress)。root と template の guard は `cmp` で同一
- していないこと: テストスイートと静的解析は流していない。下の判定は、hook に case ファイルを渡した結果(scratchpad の `sr4/` と `xr1/run.sh`。新版と旧版を、jq あり・なしの両方で比べた)と、`echo` だけの形を shell に渡した結果による。新しい回避の形は探していない

### 4e829e34 が変えたこと

END の `set_text(IN)` の直後(`.claude/hooks/pre_bash_guard.sh:1305`)で、生のコマンドに `\` と改行の並びがあれば `NODATA = 1` にする。`NODATA` を読むのは `in_data()`(`:842`)だけなので、そのコマンドのデータ区間はすべてなくなり、見張りの 4 規則が旧版どおりに決める。ヘッダーの方針の段落(`:98-109`)と `in_data()` のコメント(`:837-840`)にこの理由を書き、`guard_deny_only_forms` の 8 に F2-1 の 3 形を足した。

### 確認したこと

1. `NODATA` の行の位置と範囲
   - awk は `RS = "\001"`(`:1256`)で読むので、改行はレコードの中に残る。`IN` はレコードを `"\001"` でつなぎ直す(`:1298`)ので、コマンドと同じ文字列になる。違うのは末尾の `\001` が落ちることだけで、行継続には関係しない
   - 立てる場所は END の最初で、字句解析より前にある。`NODATA` を 0 にするのは BEGIN(`:1295`)だけで、`in_data()` を呼ぶ `sentinel()` は END の最後(`:1325`)に走る。そのため、行継続がコマンドのどこにあっても、全体のデータ区間に効く。働きは区間を消す向きだけで、字句解析の deny は弱めない
   - `IN` に出ない行継続があるかを 3 つ確かめた。(a) CRLF: `\`・CR・LF の並びは `index` に当たらない。bash 3.2、bash 5.2(ubuntu:24.04)、zsh 5.9、dash に `echo "$\<CR><LF>(echo SUBST-RAN)"` を渡すと、4 つとも置換を実行せず、文字のまま表示した(`sr4/s2-dq-bscrlf.sh`)。CR があると行継続にならないので、guard が見る必要はない。hook は同じ形(`sr4/a2-dq-bscrlf.txt`)を新版 none/none、旧版 deny/deny とする。shell が実行しない形なので、データ区間の扱いとして正しい。(b) jq のない経路: `lib_json.sh` の awk は左から 1 つずつ戻すので、JSON の `\\\n` は `\` と改行になる。JSON の `\\u000a`、`\\n`、`\\\u000a`、大文字の `\\u000A` を payload に直接書いて渡した(`sr4/u1.json`〜`u4.json`)。jq あり・なしとも `\` と改行に戻り、新版・旧版とも deny/deny だった。(c) `command="$(...)"` は末尾の改行を落とすので、コマンドの最後にある行継続は `IN` に出ない。その後ろに文字がないので、shell がつなぐ相手もない
2. AC3 と、意図した誤検知の修正
   - `ac3`(`tests/test-pre-bash-guard.sh:601-635`)と `intentional_fixes`(`:1109-1123`)の形には、`\` と改行の並びが 1 つもない(`printf 'a\ngit push --force'` の `\n` は 2 文字のまま)。そのため、この 2 つの判定は 4e829e34 で変わらない。`edge_none` にも行継続を含む形はない
   - ただし、同じコマンドのどこかに行継続があると、意図した修正が効かなくなる(R2-1)
3. 3e9afad6 のテストと新しいコメント: R2-2 と R2-3
4. hook で確かめた形(新版 jq あり/なし、旧版 jq あり/なし)
   - F2-1 の形: `sr4/a1`(`echo "$\<改行>(...)"`)、`a3`(`printf '%s'`)、`a4`(`git tag -a v1 -m`)、`a5`(`"x$\<改行>\<改行>(...)"`)、orchestrator の `xr3/f1.txt`〜`f7.txt` は、新版・旧版とも deny/deny
   - 字句解析が止めるべき形: `sr4/d1`(`git commit -m "$\<改行>(id)"`)と `d2`(中身が `git push origin --force`)は、新版・旧版とも none/none。旧版より弱くはない。ただ、bash と dash はこの置換を実行する(`sr4/s1-dq-bsnl.sh`: bash 3.2・bash 5.2・dash は `SUBST-RAN` を出した。zsh は行継続を取ったうえで `$(echo SUBST-RAN)` を文字のまま出した)。/test の Test gaps に挙がっている点なので、下の Tech debt に入れた
   - triage の case ファイル: `xr1/` の 54 件のうち 53 件は deny/deny で、`v-ml4.txt` だけ none/none(前の節と同じ)。`xr3/q1.txt`(P2-5 の形)は none/none
   - 止めすぎる側: R2-1 の 7 形は、8f852e4f の guard で none、HEAD で deny、旧版で deny
   - 確かめた形の中に、旧版より弱くなったものはない

### 再実行の findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R2-1 | LOW | maintainability | ヘッダーの方針の段落(`.claude/hooks/pre_bash_guard.sh:103-106`)は、行継続でデータ区間を消す代価として、単一引用符の中の行継続だけを挙げている。plan の Progress の F2-1 の行も同じ書き方をしている。実際には `index()` が文字の並びしか見ないので、次の場合もコマンドのデータ区間がすべて消える。shell が取り除き、字句解析も同じに読む引用符の外の行継続(`lex_cmds` の `:253`)。引用符つき区切りのヒアドキュメントの本文の行末の `\`。コメントの行末の `\`。行継続ではない `\\` と改行の並び。AC3 の中心にある推奨のコミット形もこれに当たる。`git add a \<改行>  b && git commit -m "$(cat <<'EOF' …)"` は deny になり、その理由の文は「代わりにシングルクォートまたは HEREDOC (<<'EOF') を使用してください」と、すでに使っている形を勧める。旧版も deny にしていたので、AC7 には反しない。また END のコメント(`:1301-1302`)の「`"$\ newline (cmd)"` is a substitution」は bash と dash にしか当てはまらない。zsh 5.9 は行継続を取るが、置換にはしない(テストの 8 のコメントは「in bash and dash」と書き分けている) | `sr4/` の 7 形。`b1`(力ずくの push の文字列を `grep -n` で探し、行継続のあとにファイル名)、`b2`(行継続のあとの推奨のコミット形。本文に sudo の語)、`b3` と `c3`(推奨のコミット形の本文の行末に `\`)、`b4`(コメントの行末に `\`)、`b5`(`echo foo\\<改行>…`)、`c2`(行継続のあとの、見張りの語を含まない推奨のコミット形)。どれも 8f852e4f の guard では none、HEAD では jq あり・なしとも deny、旧版も deny。行継続のない `b6` と `c1` は、新版で none。shell では、`s3`(`\\` と改行)と `s4`(コメントの行末の `\`)の次の行が、bash・zsh・dash で別のコマンドとして走った。`s5`(推奨の形の本文の行末の `\`)は、bash 5.2・zsh・dash では文字のまま残った | 括弧の中を次の意味に書き直す。「`\` と改行の並びがあればデータ区間を消す。shell がそれを文字として残す場所(単一引用符、引用符つき区切りのヒアドキュメント、コメント、`\\` の後ろ)でも、引用符の外のふつうの行継続でも消すので、行継続で書いた複数行のコマンドは旧版の規則で決まる」。END のコメントは「bash and dash」と書く。plan の Progress も合わせる。/sync-docs では、tech-debt の guard の限界の行の (a) に誤検知の型として足し、回避策(行継続を使わない、コマンドを分ける)を書く。この止めすぎを意図どおりとテストで固定するなら、`c2` の形を `edge_sentinel_deny` に 1 つ足し、規則を狭めたときに `edge_none` へ移すと書く |
| R2-2 | LOW | maintainability | `read_body` の本文ごとの規則(`:587` の `!joined && !HBSNL[h]`)は、4e829e34 のあとでは判定を変えない。`HDZ` を立てるのはトップレベルの本文だけで、その本文は `IN` の一部にあたる。つないだ本文(`joined`)にも、行継続を含む区切りの語(`HBSNL`)にも `\` と改行の並びがあるので、`NODATA` が先にすべての区間を消す。その結果、3 つのずれが出た。(1) ヘッダーの「(only that body loses its region)」(`:107-109`)は、結果の上では成り立たない。この括弧書きは P2-1 の修正(0ef6fc4f)で入った。4e829e34 のあとは、P2-1 の前の書き方(コマンドのデータ区間をすべて消す)の方が結果に合う。(2) 3e9afad6 が J02 と L03 を赤にするために足した 2 形は、もうその規則を固定していない。`guard_deny_only_forms` の 3 にある、`cat <<EOF` の本文で `$\` と `(…)` をつないだ形(`tests/test-pre-bash-guard.sh:570`)と、`edge_sentinel_deny` にある、区切りの語を `EO\<改行>F` と書いた形(`:909`)の 2 つ。`:905-908` のコメントの「a change that gives such a body its data region moves this to edge_none」も成り立たない。(3) `:107` の行「the command. A heredoc」だけが短く、段落の折り返しが途中で止まっている | 写しの hook で `:587` の条件を `if (DCTX[HX[h]])` に変えた変異体(`sr4/m-hdz/`)では、2 形とも deny のまま。F2-1 の行を消した変異体(`sr4/m-nodata/`)では、8 の 3 形が none になり、この 2 形は deny のまま残った(本文ごとの規則が止める)。2 つの仕組みが重なっているので、片方を外しても 2 形は赤にならない | ヘッダーの括弧書きを「行継続の規則がある間、この本文の規則は判定を変えない。行継続の規則を狭めたときのために残す」という意味に直し、`:107` から折り返し直す。テストの 2 か所のコメントには、いまは行継続の規則でも deny になること、J02 と L03 の変異はこの 2 形では赤にならないことを書く。/test の cycle 2 の mutation の表(J02 と L03 の「足したあと」の 6 件と 2 件)は 4e829e34 の前の値なので、/test の再実行で測り直す |
| R2-3 | LOW | readability | `guard_deny_only_forms` の前のコメント(`tests/test-pre-bash-guard.sh:537-544`)は、配列にある形の種類を並べているが、4e829e34 が足した 8 の種類(ダブルクォートの中で `$` と `(` を行継続で分けた形)を挙げていない。出どころも「cross-review と follow-up probes」だけで、F2-1 が /test の報告から来たことが分からない | `:537-544`、`:591-596` | 種類の一覧に「a backslash-newline between $ and ( inside double quotes (/test F2-1)」を足す |

### 前の節の指摘の状態

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| P2-1 | 0ef6fc4f で直したが、4e829e34 のあとは括弧書きが結果に合わない | R2-2 |
| P2-2 | 解消 | ヘッダーの (6)(a)(`:79-86`)に、fd 0〜2 への複製、`>&-`、`&>` と `&>>` を除くこと、`printf without -v` が入った(0ef6fc4f・63b6743a)。`redir_safe`(`:857-863`)と合う |
| P2-3 | 直さずに記録へ回すと plan の Progress にある。tech-debt にはまだない | `docs/tech-debt/README.md` は 8f852e4f から変わっていない |
| P2-4 | 未。/sync-docs の担当 | 同上 |
| P2-5 | orchestrator が bash・zsh・dash で確かめ、穴ではないとした(plan の Progress)。shell 側は確かめ直していない | hook は `xr3/q1.txt` を新版・旧版とも none/none |
| P2-6 | 未。この周の commit は (1)〜(4) のどれも変えていない | `git diff 8f852e4f..HEAD -- .claude/hooks/pre_bash_guard.sh` はヘッダーと `in_data()` のコメント、END の 1 行だけ |

### Tech debt identified

この周が上限 2 の最後の周なので、直さない LOW はそのまま持ち越しになる。この commit では `docs/tech-debt/README.md` を変えていない。/sync-docs で、前の節の 3 行と合わせて、guard の限界の行にまとめてほしい。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| guard のヘッダーとテストのコメントが 4e829e34 の結果とずれている(R2-1 の書き方、R2-2、R2-3) | コメントが実際の判定と違うので、次に行継続の規則を狭める人が、どのテストが本文ごとの規則を守っているかを読み違える | コメントだけの修正でも guard の変更になり、self-review → verify → test をもう一度回すことになる | 次に guard を変える PR | この節の R2-1、R2-2、R2-3 |
| 行継続を含むコマンドの誤検知(R2-1) | 推奨のコミット形を含め、行継続で書いた複数行のコマンドは、データの中の語でも止まる。理由の文が使っている形を勧める | 旧版も止めていた形で、AC7 には反しない。狭めるには、下の字句解析の修正が先に要る | 誤検知の報告、または下の行を直すとき | この節の R2-1 |
| 字句解析が `$` の直後の行継続を読み飛ばさない(`lex_dollar` の `:388`) | `git commit -m "$\<改行>(id)"` のような形を、字句解析のコミットの規則が見ない。bash と dash は置換を実行する。旧版も none | 4e829e34 は見張りに任せる方を選んだ(consult と合意)。字句解析の修正(/test の Proposed fix の (1))は `lex_dq`・`lex_brace`・`lex_hd`・`lex_word` からの経路を変えるので、この周では広すぎる | 次に guard を変える PR。直したら、行継続の規則を狭められるかも見直す | /test の cycle 2 の F2-1 と Test gaps、この節の確認の 4 |

### Recommendation(pipeline cycle 2 の再実行、現時点)

- Merge: merge(CRITICAL・HIGH・MEDIUM はない。F2-1 はデータ区間の側で閉じ、確かめた形の中に旧版より弱くなったものはない。LOW の 3 件は、コメントと記録のずれと、旧版と同じ止めすぎ)
- 前の節までの Merge 行はそれぞれの時点の判定で、現時点の判定はこの行
- Follow-ups:
  - /test: cycle 2 の mutation の表を測り直す。J02 と L03 は、4e829e34 のあとは赤にならないはず(R2-2)。F2-1 の行を消す変異が 8 の 3 形で赤になることも見る(この review の `sr4/m-nodata/` では none になった)
  - /sync-docs: 上の Tech debt の 3 行と、前の節の 3 行を guard の行にまとめる。P2-4 の行番号は 4e829e34 でまた動いた(guard は 1370 行)
  - R2-1 から R2-3 のコメントは、直すなら次の PR で

### 未確認の点

- Windows: Cygwin の bash の `igncr` のように CR を捨てる設定では、`\<CR><LF>` が行継続になりうる。Git Bash などでの既定値は確かめていない。ralph のリリースは darwin と linux だけを作る
- 4e829e34 のあとの guard を、mawk・gawk・busybox の awk で流してはいない。`index()` と文字列の連結だけの 1 行で、方言の差はおそらくない。未確認です
- R2-1 の止めすぎがどれくらいの頻度で起きるかは測っていない

## cycle 3 (cap raised to 3)

- Date: 2026-10-08
- Reviewer: reviewer subagent (Claude)。cross-review の 2 周目のあと、ユーザーが上限を 3 に上げてから回した self-review。cross-review の手順どおり `cycle-count.json` は 2 のまま。ID は前の節と混ざらないよう `C3-` で始めた
- 対象: `git diff 373fa29d..HEAD -- .claude tests templates`(HEAD 57bf08f8)。guard とテストの変更は a9ef82b1。ほかに c61ab2bf・4ffe74fe(前の節の R2-1〜R2-3 のコメント修正)と 180c7389(tester が足した 6 形)がある。`.claude/skills/org/`、`templates/base/` の org の skill と `codex-seat-permissions.md` の差分は、e1dfb422 で取り込んだ main の変更で、`git diff origin/main HEAD` で差がない。root と template の guard は `cmp` で同一
- していないこと: テストスイートと静的解析は流していない。判定は、hook に case ファイルを渡した結果(scratchpad の `sr5/` と `xr1/run.sh`)と、無害な形を shell に渡した結果による。新しい回避の形は作っていない。使ったのは、triage と test ファイルにある形と、見張りの語をデータとして含む形だけ

### a9ef82b1 が変えたこと

- `end_cmd` が `data_first_ok`(`.claude/hooks/pre_bash_guard.sh:742-756`)を呼ぶ。トップレベルの単純コマンドの 1 語目の値が DATACMD の名前でも `git` でもなければ、`NODATA` を立てる(`:727`)。値に `/` があるときと、語がない(リダイレクトだけの)ときも同じ
- `lex_redir` は、ヒアドキュメントの区切りの生の語に `$` かバッククォートがあれば `NODATA` を立てる(`:540`)
- `push_rules`(`:1113-1130`)は、値を取る長いオプション(`push_val_opt`、`:1133-1135`)と短い `-o` の値を読み飛ばす。束ねた短いオプションは位置で読む(`:1120-1127`)。`reset_rules` は `--` で止まる(`:1138-1143`)
- テスト: `guard_deny_only_forms` の 9 に cross-review の 3 形と許可リストの 8 形、`ac3` に push と reset の誤検知の 6 形、`edge_deny` に push の 3 形を足した。`=echo sudo ls` は `edge_none` から `edge_sentinel_deny` に移った

### 確認したこと

1. triage の P1 と、同じ型
   - triage の case ファイル `xr3/c2p1a`〜`c2p1c` は、新版が deny/deny(jq あり/なし)、旧版も deny/deny。修正の前は新版が none だった(triage の再現)
   - どの規則が閉じたかを変異体で分けた。規則を 1 つずつ外した写し(`sr5/m_*`)に、テストの 6 つの配列の 239 行(deny が 115 行、none が 124 行。jq ありの経路)を渡した。区切りの規則を外すと、P1-1 の形(区切りが `$'\x45'`)だけが none になる。この形の 1 語目は `cat` なので、許可リストでは閉じない。許可リストを外すと、P1-2・P1-3、許可リストの 6 形、`=echo sudo ls` の 9 行が none になる
   - 型について(コードを読んだ結果): トップレベルの DATACMD の引数がデータになるのは、出力の行き先が端末、DATACMD の次の段、`/dev/null`、`/dev/stderr`、fd 0〜2 の複製、閉じた fd だけのとき(`stage_note` と `pipe_decide`、`:911-961`)。置換の範囲はデータから外す(`xnote`)。その出力を読めるトップレベルのコマンド(shell、`eval`、前置き、道のある名前)は、許可リストでなくなった。このテキストを shell が実行する道や、実行される場所に書く道は、コードの上では見つからなかった。残るのは 3 つで、どれも hook では確かめていない。名前が実際に指すもの(下の未確認の点)、DATACMD の各コマンドが自分のオプションでする処理(一覧はこの diff で変わっていない)、git が保存したメッセージ(データ区間の (b))を同じコマンドの中の置換が実行時に読み返す場合
   - triage の case ファイル `xr1/` の 52 件のうち 51 件は、新版・旧版とも deny/deny。`v-ml4.txt`(複数行のダブルクォートの引数の中の語)だけが新版 none、旧版 deny で、前の節と同じ。shell はこの語を実行しない。`xr3/q1.txt`(P2-5 の形)も前の節と同じ none/none
2. `data_first_ok` の正しさ(`sr5/bn/b01`〜`b32`。新版と旧版、jq あり/なし)
   - 引用符を外した値で比べる。`"echo"`、`\echo`、`e\cho`、`ec""ho` の後ろの見張りの語は none(`b01`〜`b04`)。shell もこれらを `echo` と読む
   - `$` のある値: `$c` と `${x:-echo}` は、字句解析の値に `$` がそのまま残るので一致しない(`lex_dollar`、`:398-418`)。`$c` の形は deny。`$'echo'` は none(`b05`)で、これは C3-4
   - リダイレクトだけのコマンドは `WN < 1` で 0 を返す。`>out.txt` の次の行の `echo` は deny(`b17`)。zsh はリダイレクトだけのコマンドで `NULLCMD`(既定は `cat`)や `READNULLCMD`(既定は `more`)を走らせる(zshparam)ので、データ区間を与えない方が正しい
   - `|`、`&&`、`||`、`;`、`&`、改行の後ろの 1 語目: どれも `end_cmd` を通り、語の数が 0 に戻るので、段ごと・要素ごとに判定される。全部が DATACMD か git なら none(`b08`〜`b12`、`b21`、`b27`)。1 つでも外れると deny(`b18` の `:`、`b19` の `!`、`b20` の代入、C3-1 の 4 形)。`:` は DATACMD にないので `true` と扱いが違うが、旧版も deny にしていて害はない
   - ヒアドキュメントだけのコマンド: `cat <<'EOF'`、`<<'EOF' cat`、`cat <<"EOF"`、`git commit -F - <<'EOF'`、推奨のコミット形、`git add … &&` の後ろの推奨のコミット形は none(`b22`〜`b28`)。区切りに `$` があると deny(`b29`)
3. push と reset
   - test ファイルの `ac2`・`ac3`・`edge_deny`・`edge_none` から push と reset の 118 行を取り出して新版に渡した(`sr5/rows/`)。jq あり/なしとも期待どおりで、`ac2` の 34 行と `edge_deny` の 52 行は deny、`ac3` の 14 行と `edge_none` の 18 行は none
   - 束ねた短いオプション: 最初の `f` が最初の `o` より前にあれば deny。`o` が最後の文字なら次の語を値として読む。`-fo x` と `-uf` は deny、`-of` と `-ofoo` は none。git 2.49 の `git push -h` で値を取る短いオプションは `-o` だけなので、`-of` は push option の `f` で、force ではない。`--repo` などが次の語を値に取るのも、git の parse-options が次の語を `-` で始まっていても値に取るのと合う。この読み方の中に force の取りこぼしは見つからなかった
   - push の `--`: 特に扱わない。`--` の後ろの `+ref` は deny のままで、`--force` という名前の refspec も deny になる。止める側なので、このままでよい
   - reset の `--`: C3-2
4. AC3 と意図した修正
   - a9ef82b1 は `ac3` から行を消しておらず、`intentional_fixes` も変えていない。`edge_none` から消えたのは `=echo sudo ls` の 1 行で、`edge_sentinel_deny` に移った(旧版も deny)
   - `ac3` の 35 行、`edge_none` の 76 行、`intentional_fixes` の 13 行は、新版の jq ありの経路で none。1800/0 の件数は、スイートを流していないので確かめていない
5. コードの質
   - awk の本文(`:153-1394`)に単一引用符はない。root と template の guard は同一
   - コメントのずれは C3-3〜C3-5

### cycle 3 の findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-1 | MEDIUM | maintainability | 推奨のコミット形 `git commit -m "$(cat <<'EOF' … EOF)"` は、同じ Bash のコマンドに DATACMD でも git でもないコマンドがあると deny になる。本文に見張りの語がなくても止まる。この形はいつも見張りの 4 番目の規則(`-m "` の後ろの `$(`)に当たり、それを外していたのはデータ区間 (b) だけだったため。373fa29d では none、旧版は deny なので、AC7 には反しない。許可リストとしては筋の通った挙動だが、次の 3 点が合っていない。(1) 理由の文は「代わりにシングルクォートまたは HEREDOC (<<'EOF') を使用してください」で、使っている形を勧める。(2) ヘッダーの `:117-119` は、推奨の形が通る条件を「その git が 1 語目で、どの行も `\` で終わらない」とだけ書き、同じコマンドのほかの単純コマンドの条件を書いていない。(3) 配る文書が古くなった。`internal/org/prompts/implementer.md:28-30` は「どの行もバックスラッシュで終わらなければ通る」と書き、その少し前に、検証を走らせてからステージしてコミットする手順を書いている。この手順を 1 つのコマンドにつなぐと止まる。`docs/tech-debt/README.md:125` の解消済みの行は「推奨の HEREDOC の形は通る」とし、残る場合として行末の `\` だけを挙げる。plan の Progress の a9ef82b1 の行の「止める側への逸脱」も、この代価を挙げていない | `sr5/ch/b01`〜`b07`。`git add a.txt && …` の後ろの推奨の形は新版 none。`make test &&`、`./scripts/run-verify.sh &&`、`GIT_EDITOR=true`、後ろに `&& ./scripts/secret-scan-branch.sh --strict` の 4 形は、新版が deny(commit_message)、373fa29d の guard(`sr5/prev/`)が none、旧版が deny。`-F -` の形と単一引用符の `-m` は、`make test &&` の後ろでも 3 つの guard とも none | 挙動は変えない。同じコマンドの前の方にある定義が、メッセージの置換で走るものを変えうるので、この形だけを許可リストから外すことは勧めない。記録を合わせる。/sync-docs で、`implementer.md:28-30` に「コミットは単独のコマンドで実行する(`git add` とはつないでよい)」の意味を足し、tech-debt の 125 行目の「残るのは」と guard の限界の行にこの場合を足す。次に guard を変えるとき、ヘッダーの `:117-119` に「同じコマンドのトップレベルのコマンドがすべて読むだけのコマンドか git のとき」を足し、理由の文に「コミットを単独のコマンドで実行する」を足す |
| C3-2 | LOW | security | `reset_rules`(`:1138-1143`)は `--` で走査を止めるが、値を取る `--pathspec-from-file` の値を先に読み飛ばさない。git はその直後の `--` をこのオプションの値として読み、続く `--hard` をオプションとして読む。作業ディレクトリに中身が空の `--` という名前のファイルがあると、`git reset --pathspec-from-file -- --hard` は hard reset を行う。新版は none、373fa29d の guard は deny、旧版は none。AC7 には反しないが、a9ef82b1 が足した取りこぼし | `sr5/reset-dd.sh`(scratchpad の使い捨てのリポジトリ、git 2.49)。空の `--` では rc 0 で `HEAD is now at …` と出て、作業ツリーの変更が消えた。パスを書いた `--` では `fatal: Cannot do hard reset with paths.`(rc 128)。hook は `sr5/rs/r1` を新版 none、373fa29d deny、旧版 none とした。`=` で値をつけた `r2` は新版も deny | `push_val_opt` と同じく、`--` を見る前に `opt_is(a, "--pathspec-from-file")` の値を読み飛ばす(`=` がなければ次の語)。`ac3` の `git reset -- --hard` の 2 形は none のまま。`commit_rules`(`:1155`)と `no_verify_rules`(`:1144-1149`)も `--` で止まる。こちらの値を取るオプションは調べていない(a9ef82b1 より前からある形) |
| C3-3 | LOW | maintainability | 予約語の規則(`:716`)と exec の規則(`:721`)は、許可リスト(`:727`)が入ったあとでは判定を変えない。`:716` が当たるのは 1 語目の生の語が `if` や `{` のときで、その値は DATACMD でも git でもない。`EXEC_SEEN` を立てるのは、`cmd_pos` が前置きをたどって `exec` に着いたとき(`:772`)で、1 語目が DATACMD か git なら `cmd_pos` は 1 を返し、前置きをたどらない。ヘッダー(`:107-110`)と `in_data` のコメント(`:877-882`)は、この 2 つを別の理由として並べている | 変異体 `sr5/m_resw`、`m_exec`、`m_resw_exec` で、239 行のうち判定が変わる行は 0。ほかの規則のうち単独で判定を決めているのは、`(` の規則(`:285`、`(echo sudo ls)` の 1 行)と区切りの規則(`:540`、P1-1 の 1 行) | 2 行と `EXEC_SEEN`(`:763`、`:772`、`:1362`)を消すか、「許可リストに含まれる。許可リストを狭めたときに cycle 1 の形を開け直さないために残す」とコメントする。/test には、この 2 つの変異が等価になったと伝える |
| C3-4 | LOW | readability | 区切りの規則のコメント(`:535-539`)は、字句解析が「`$"..."`、`${...}`、置換を展開しない」ことを理由に挙げ、shell は展開するように読める。bash・zsh・dash は区切りの中の `$x`、`${x}`、バッククォートを展開しない。字句解析と違うのは、`$'...'` のエスケープと `$"..."` だけ。規則は止める側に倒れているので、ずれはコメントだけにある。`data_first_ok` も同じ字句解析の値を使い、そのコメント(`:743-744`)は「shell が見るとおりの値」と書くが、`$'...'` では成り立たない(`lex_ansi`、`:444-460`)。`$'echo'` はデータになる(`b05`)。コードを読む限り害はない。2 つの値が違うとき、shell の側の名前には制御文字、`\`、先頭の `$` のどれかが入り、その名前のコマンドがすでにある場合にしか動かない(ヘッダーの Not covered の状態) | `sr5/delim-sem.sh`: `${x}`、`$x`、バッククォートの区切りで、3 つの shell とも本文は文字どおりの区切りの行まで続いた(dash はバッククォートの形を構文エラーにした)。`sr5/dq-sem.sh`: `$"abc"` は bash が `abc`、zsh と dash が `$abc`。未知のエスケープ `\l` は、bash が `\` を残し、zsh は落とした(字句解析も落とす) | 区切りのコメントを「`$'...'` と `$"..."` は shell と同じには読まない。ほかの `$` とバッククォートは文字どおりだが、規則を簡単にするため一緒に止める」の意味に直す。`data_first_ok` のコメントに `$'...'` の例外を書くか、区切りの規則とそろえて、`WR[ctx, 1]` に `$` があれば 0 を返す(代価は `$'echo'` と `$"echo"` だけ) |
| C3-5 | LOW | readability | テストのコメントのずれ。(1) `tests/test-pre-bash-guard.sh:666` の「Cycle 2 (P2-4, P2-5)」と `:829` の「Cycle 2 (P2-4)」は、triage の書き方(1〜5 の番号と、重さの [P1]・[P2])と合わない。この報告では P2-4 と P2-5 は別の指摘(tech-debt の行の行番号と、未読のヒアドキュメント)を指すので、grep すると別のものに着く。`:999` の「change A」は repo のどこにも定義がない。(2) `:616-620` は、許可リストが「a variable as the command name」も閉じると書くが、`$c 'sudo ls'`(`:627`)は許可リストを外しても deny になる。`$c` は読むだけのコマンドではないので、もともとデータ区間がない。この行は許可リストを固定していない | `P2-4`・`P2-5`・`change A` を grep した結果(この報告の P2 の表、`docs/reports/cross-review-triage-guard-deny-only.md` の表)。変異体 `sr5/m_allow` で、`$c` の行は deny のまま、ほかの許可リストの 6 行は none | 「cross-review cycle 2, findings 4 and 5」のように書き、「change A」は「the allowlist (a9ef82b1)」にする。変数の行は前からの deny だと書くか、変数を別のコマンドにして、後ろに読むだけのコマンドを置く形に変える |

### 前の節の指摘の状態

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| R2-1 | 解消 | c61ab2bf がヘッダーの方針の段落を書き直し、END のコメントを「in bash and dash」にした。plan の Progress の F2-1 の行も直った。誤検知は d5e197de が tech-debt の guard の限界の行に記録した |
| R2-2 | 解消 | c61ab2bf が「(only that body loses its region)」を消した。4ffe74fe が、テストの 2 か所のコメントに、行継続の規則で deny になることを書いた |
| R2-3 | 解消 | c61ab2bf が `guard_deny_only_forms` の前のコメントに 8(/test の F2-1)を足した |

### Tech debt identified

この周は、上げた上限 3 の最後の周にあたる。直さない指摘はそのまま持ち越しになる。この commit では `docs/tech-debt/README.md` を変えていない。/sync-docs で次の 1 行にまとめてほしい。C3-1 の記録(`implementer.md`、125 行目、guard の限界の行)は、この周の /sync-docs で直せる。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| a9ef82b1 の guard の残り(C3-1 の理由の文とヘッダー、C3-2 の reset の値、C3-3 の重なった 2 規則、C3-4 と C3-5 のコメント) | C3-1: つないだコマンドの中の推奨のコミット形が止まり、理由の文が使っている形を勧める。C3-2: 中身が空の `--` という名前のファイルがあるときの hard reset を通す(旧版も通す)。C3-3〜C3-5: 次に guard を変える人が、コメントと変異の結果を読み違える | guard を 1 行でも変えると self-review から /cross-review までをもう一度回すことになり、上限 3 を超える | 次に guard を変える PR | この節の C3-1〜C3-5 |

### Recommendation(cycle 3、現時点)

- Merge: merge(CRITICAL・HIGH はない。cross-review の 2 周目の P1 3 件は閉じた。旧版が deny にした形のうち、確かめた中で新版が none にしたのは、shell が実行しないデータの形だけ。MEDIUM の 1 件は旧版と同じ deny で、理由の文と記録が挙動に合っていない問題。LOW の 4 件は、旧版も通す reset の 1 形と、重なった規則とコメントのずれ)
- 前の節までの Merge 行はそれぞれの時点の判定で、現時点の判定はこの行
- Follow-ups:
  - /sync-docs: C3-1 の 3 か所(`internal/org/prompts/implementer.md:28-30`、`docs/tech-debt/README.md:125`、guard の限界の行)と、上の Tech debt の 1 行。guard は 1437 行になった
  - /test: C3-3 の 2 つの変異は等価になった。許可リストを外す変異と区切りの規則を外す変異は、この review では 9 行と 1 行で赤になった
  - C3-2 は旧版も通す形なので、直すなら次の PR で

### 未確認の点

- 名前が指すもの: Claude Code の Bash ツールは、利用者の shell のスナップショットを読み込む。ここでは `cat` が `bat`、`ls` が `eza -aal --icons` のエイリアスで、`grep` は Claude Code の関数になっている(`type cat grep ls`)。許可リストが見るのは名前で、ヘッダーの Not covered はエイリアスと関数を挙げている。これらのプログラムが引数をどう扱うかは確かめていない
- git が保存したメッセージを、同じコマンドの中の置換や git の下位コマンドが実行時に読み返す場合(確認したことの 1)。hook では確かめていない。ヘッダーの「実行時にしか分からないもの」に入る
- テストスイート(1800/0)、mawk・gawk・busybox の awk、ubuntu での実行はしていない。239 行の変異の比較は jq ありの経路だけで回した

## cycle 4 (cap raised to 4)

- Date: 2026-10-08
- Reviewer: reviewer subagent (Claude)。cross-review の 3 周目のあと、ユーザーが上限を 4 に上げてから回した self-review。`cycle-count.json` は 2 のまま。ユーザーはこれを最後の run とした。ID は `C4-` で始めた
- 対象: `git diff 48628bd2..HEAD -- .claude tests templates internal`(HEAD fbf3c584)。guard の変更は 849f5411(C3-1 の理由の文とヘッダー、C3-2 の reset)と b3c3fdaa(printf、DATACMD、`no_verify_rules`、`commit_rules`、ヘッダーの Not covered)。テストの変更は 8e94e76b(tester)と b3c3fdaa。`internal/org/prompts/implementer.md` と `git-commit-strategy.md` の差分は /sync-docs(24876df7)の文書の変更。root と template の guard と rule は `cmp` で同一
- していないこと: テストスイートと静的解析は流していない(1856/0 は orchestrator の申告)。判定の根拠は、hook に行ごとのファイルを渡した結果(scratchpad の `sr6/`)、計測用に写した hook の出力、shell に無害な形を渡した結果。見張りの語を含む形で hook に渡したのは、triage の case ファイルと test ファイルの配列の行だけで、新しい形は作っていない

### 849f5411 と b3c3fdaa が変えたこと

- `reset_rules`(`.claude/hooks/pre_bash_guard.sh:1145-1152`)は、`--` を見る前に `--pathspec-from-file` の値を読み飛ばす(`=` がなければ次の語)
- `stage_note`(`:923`)は、printf の語のどれかが `-v` で始まるか、`%`・`$`・バッククォートを含めば、引数をデータにしない。判定は語の値(`WV`)で読む
- DATACMD の一覧(`:1352`)から `test` と `[` を外した
- `no_verify_rules`(`:1157-1159`)は `--` で止まらない
- `commit_rules`(`:1184`)は、値を取る長いオプション 10 個を `opt_is` で読み、`=` があれば 1 語、なければ 2 語進む
- 理由の文(commit_message、`:1437`)とヘッダー(`:117-120`)に「git commit を単独のコマンドで打つ」を足した。ヘッダーの Not covered に、起動ファイルで立てた shell のオプションを足した(`:130-131`)

### 確認したこと

1. triage cycle 3 の 2 件
   - `xr3/c3p1a`(printf `%n`)と `c3p1b`(`test -v`)は、新版が deny/deny(jq あり/なし)、旧版も deny/deny。`[ -v` は test ファイルの B 節の行で、下の 4 で deny/deny
   - `c3p2a`(`git merge -m --` の形)、`c3p2b`(`git commit --trail --` の形)、`c3p2c`(メッセージが `--no-verify` の merge)は、新版が deny/deny、旧版が none/none
   - triage が挙げた形は 2 件とも閉じた。ただし #1 の型は、printf の判定が語の値を読むため、`$'...'` で書くと残る(C4-1)。#2 の型は 1 文字の略記で残る(C4-2)
2. 正しさ
   - `i += (index(a, "=") ? 1 : 2)`: `=` で値をつけた形は 1 語、離した形は 2 語進む。`opt_is` は `=` から後ろを外して比べるので、`--date=now` の次の語は読まれる。mawk 1.3.4(ubuntu:24.04、jq なしの経路)でも、commit・printf・merge・rebase・reset を含む 136 行の判定が配列の期待どおりだった
   - `opt_is` の略記の衝突: 値を取る 10 個のオプションと `--message`・`--file` について、長さ 1 から全長までのすべての接頭辞を git 2.49.0 に渡した(`sr6/abbrev.sh`、使い捨てのリポジトリ)。4 文字以上の接頭辞は、そのオプションに決まるか、git が ambiguous で止める(`--re` は reuse-message と reset-author、`--fi` は file と fixup、`--pathspec-f` までは pathspec-file-nul と重なる)。guard が次の語を値として飛ばし、git がその語をオプションとして読む組み合わせはなかった。`--da` は date だけに決まり、`--dry-run` と重なるのは `--d` から。3 文字の接頭辞の扱いは C4-2
   - `no_verify_rules` が `--` で止まらないこと: AC2 の `git merge --no-verify x` は deny のまま。AC3 に merge・rebase・am の形はない。止めすぎが増えるのは、`--` の後ろの語が `--no` から `--no-verify` までの接頭辞のとき(`git merge --no` は git も unknown option で止める)と、メッセージの値がちょうど `--no-verify` のとき(test ファイルが意図した止めすぎとして固定している)
   - printf の規則と AC3: `printf 'a\ngit push --force'` の語の値は `a\ngit push --force` で、`%`・`$`・バッククォートを含まない。計測用の写しで SRO=1、判定は none のまま
3. DATACMD のほかの名前。置換の中身は `echo HIT-<名前> >&9` だけにした(`sr6/builtins.zsh`、`sr6/builtins.bash`)
   - zsh 5.9(`zsh -f`): echo、true、false、cd(引数 1 つと 2 つ)、type(`-m`、`-w` も)、which(`-m`、`-p` も)、`/bin/ls`、`/bin/cat` は置換を実行しなかった。cdablevars を立てた cd、extendedglob を立てた `type -m`・`which -m` に glob qualifier を渡した形も実行しなかった。printf は、変換のない書式、`%s`、`%c`、`%b`、書式の中の `\x25n` では実行せず、`%d`、`%x`、`%f`、`%n`、`%1$n` で実行した。数値の変換も引数を算術式として評価する。b3c3fdaa はどの `%` でも外すので、挙動としては閉じている(コメントの話は C4-3)。`test -v` は実行し、`test … -eq` と `[ … -eq ]` は実行しなかった
   - bash 3.2: どれも実行しなかった(`test -v` も printf の `%n` もない)
   - bash 5.2(ubuntu:24.04): `printf -v`、`test -v`、`$'\x2dv'` と書いた `printf -v` を実行した
   - 外部コマンド: shell は引数を評価しない。プログラムが引数を実行に使う道は、rg の `--pre`(`:917` で外している)のほかに見つからなかった。macOS の zgrep、egrep、fgrep、diff、which は Mach-O のバイナリで、diff は Apple diff(FreeBSD diff 由来)。`diff --help` に別のプログラムを走らせるオプションはなかった。GNU 側(zgrep はシェルスクリプト)は調べていない
   - まとめ: 引数を変数名や添字として評価したのは、zsh の printf の数値変換と `%n`、bash の `printf -v`、zsh と bash の `test -v`(と `[ -v`)だけで、b3c3fdaa はどれもデータから外した。ただし外す判定が語の値を読むので、`$'...'` で書いた書式と引数は外れない(C4-1)
4. AC7: test ファイルの配列の 489 行を、新旧の guard の jq あり/なしに渡した(`sr6/dump.sh`、`sr6/runall.sh`)。内訳は A の `pr206_deny` 10 行と `former_none` 22 行、B の `ac2`・`self_review_forms`・`guard_deny_only_forms` 156 行、C の `ac3` 29 行、D の 3 配列 259 行、`intentional_fixes` 13 行
   - 新版の判定は、どの行も配列の期待どおり
   - コーパス(A の deny、B、C)で旧版 deny・新版 none になるのは C の 13 行で、`intentional_fixes` の 13 行と内容のハッシュで一致した(jq あり/なしとも)
   - 849f5411 が C から D に移した 6 行は旧版も none なので、この集合は変わらない。D の `edge_none` には旧版が deny にする行が 40 行あるが、D はコーパスに入らない(AC7 の文との照合は /verify の範囲)
5. コードの質: awk の本文(`:156-1406`)に単一引用符はない。`bash -n` は通る。root と template は同一。`commit_rules` の `:1184` は 326 文字の 1 行の条件で、前の版も 1 行の条件だった。ヘッダーの `:131` が 124 桁(C4-3)

### cycle 4 の findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C4-1 | HIGH | security | printf の判定(`:923`)は語の値 `WV` を読むが、`$'...'` の値は字句解析と shell で違う。`lex_ansi`(`:443-462`)は `\n`・`\t`・`\r` しか戻さず、ほかのエスケープでは `\` を落とすだけなので、`$'\x25n'` を `x25n`、`$'\x24('` を `x24(` と読む。zsh と bash はそれぞれ `%n`、`$(` と読む。このため `$'...'` で書いた書式と引数は `%` も `$` も含まないことになり、printf の引数がデータ区間になる。zsh は、その引数の添字の中の置換を `%n` で実行する。データ区間の中の見張りの一致は無視されるので、旧版が deny にする形を、shell が実行する位置で新版が通すことになる。triage cycle 3 #1 と同じ型で、b3c3fdaa の閉じ方の外にある。同じ読み違いは、46806dc9 からある `-v` の判定(bash 5.2 の `printf $'\x2dv'`)と、rg の `--pre` の判定(`:917`)にもある。ヘッダーの Not covered(`:125-127`)は、`$'...'` の 16 進と 8 進のエスケープを「字句解析は見落とし、見張りは書かれたとおりの文字を見る」と書く。ここでは見落とした値が見張りの一致を外す側に働くので、この文はこの場合に成り立たない | (1) zsh 5.9: `printf $'\x25n' $'a[\x24(echo HIT >&9; echo 1)]'` と、8 進の `$'\045n'`・`$'\044('` の形で HIT が出た(`sr6/builtins.zsh`)。bash 5.2.21: `printf $'\x2dv' $'a[\x24(echo HIT >&9; echo 1)]' x` で HIT が出た(`sr6/builtins.bash`)。(2) 計測用の写し(`sr6/instr.py`。`stage_note` の `SRO[cid] = ro` の後で `WR`・`WV`・`SRO`・`SSAFE` をファイルに書く): この 2 形の語の値は `x25n`、`a[x24(echo X >&2; echo 1)]`、`x2dv` で、SRO=1、SSAFE=1。`'%n'` の形は SRO=0。(3) `lex_dollar`(`:417`)は `$'` を `lex_ansi` に渡し、`xnote` を呼ばないので、その語の範囲はデータ区間から外れない。見張りの語を入れたこの形は hook に渡していない(probing の約束)。新版が none を返すことは、コードを読んだ結果で、hook では確かめていない | `:923` の `$` の判定を生の語で読む(`index(WR[ctx, j], "$")`)。`$'...'` と `$"..."` の語は生の語に必ず `$` があるので外れ、`\x25` で `%` を作る形も `$'` が要るので一緒に閉じる。この 1 語の変更を入れた写し(`sr6/fx/`)で 489 行を流すと、判定は 1 行も変わらず、AC3 の `printf 'a\ngit push --force'` は SRO=1 のまま、上の `$'...'` の 3 形は SRO=0 になった。`:917` の rg も、`WR` に `$` があれば外す。ヘッダーの `:125-127` に、printf と rg は `$` を含む語があればデータにしないことを書く。B 節に `$'...'` で書いた printf の行を足す(見張りの語は既存の行と同じ `sudo id` を使う) |
| C4-2 | MEDIUM | security | `opt_is`(`:1105-1111`)は 4 文字以上の略記しか読まない(`:1110` の `k >= 4`)が、git 2.49 は `--` の後ろが 1 文字の略記も、一意なら受け付ける。`git reset --h` は `--hard` に決まり、作業ツリーの変更を捨てる。guard はこれを通し、test ファイルは `edge_none`(`tests/test-pre-bash-guard.sh:893-896`)でこの行を「略記に見えるだけの長いオプション」として none に固定している。この前提は git 2.49 では成り立たない。同じ下限のため、git commit の `--m`(`--message`)と `--c`(`--cleanup`)も読まれない。`--m` は次の `--` を値に取るので、triage cycle 3 #2 の形を `--trail` ではなく `--m` で書くと、`commit_rules` は `--m` を 1 語として飛ばし、`--`(`:1165`)で走査を止める。b3c3fdaa のコメント「also abbreviated」(`:1182`)は 2 文字目からしか成り立たない。旧版はどちらも通す(`git reset --hard` という並びがない)ので、AC7 には反しない。`git reset --h` の行は 5b0b20d5 からある | `sr6/abbrev.sh`(git 2.49.0): `git commit --m` は "option `message' requires a value"、`--c` は "option `cleanup' requires a value"、`git reset --h -- a` は "Cannot do hard reset with paths"。commit のほかの 1 文字の接頭辞(`--a`、`--d`、`--f`、`--p`、`--r`、`--s`、`--t`)は ambiguous。`sr6/reset-h.sh`: 変更した `f` が `git reset --h` で元に戻り、rc 0。`--c` は、値が整理のモードでなければ git が止めるので、`--no-verify` の抜け道にはならない(コードを読んだ結果) | `:1110` を `k >= 3` にする。git 2.49 では、1 文字の接頭辞は一意ならそのオプションに決まり、一意でなければ ambiguous で止まる(push の `--e` は `--exec` に決まり、`--r`・`--p`・`--f`・`--n` は ambiguous。`sr6/abbrev-push.sh`)。guard が値として飛ばした語を git がオプションとして読む組み合わせは増えない。この変更を入れた写し(`sr6/m3/`)で 489 行を流すと、変わるのは `git reset --h` の 1 行(none から deny)だけで、deny の行は減らなかった。その行は `edge_deny` に移し、コメントを直す。ヘッダーの `:59-62` の「at least 4 characters」も直す |
| C4-3 | LOW | readability | b3c3fdaa のコメントが理由を狭く書いている。(1) `stage_note` のコメント(`:918-922`)は、`%` を外す理由を `%n` だけで説明する。zsh は `%d`・`%x`・`%f` でも引数を算術式として評価し、添字の中の置換を実行する。規則はどの `%` でも外すので挙動は正しいが、次に読む人が `%n` に狭めると開く。(2) DATACMD のコメント(`:1350-1351`)は、`test` と `[` を外す理由を zsh の `-v` とするが、bash 5.2 の `test -v` も同じく実行する。(3) ヘッダーの `:131` は 124 桁で、前後の行は 78 桁前後。足した 1 文が折り返されていない | `sr6/builtins.zsh` の `HIT-printf-d`・`HIT-printf-x`・`HIT-printf-f`。ubuntu:24.04 の bash 5.2.21 で `sr6/builtins.bash` を流した `HIT-test-v`。`awk '{print length}'` で `:131` は 124 | (1) 数値の変換と `%n` は引数を算術式として評価する、の意味に直す。(2) 「in zsh and bash 4.2+」にする。(3) `:130-132` を折り返す。次に guard を変えるときでよい |
| C4-4 | LOW | security | `--no-verify` の規則は、`git_rules` の振り分け(`:1092`)で merge・rebase・am にだけ当たる。git 2.49 の `git pull` も `--[no-]verify`(pre-merge-commit と commit-msg のフック)を持つが、guard は見ない。ヘッダーの `:53` は commit、push、merge、rebase、am と書いていて、コードとは合っている。旧版には `--no-verify` の規則がないので、後退ではない | `git pull -h` の出力に `--[no-]verify  control use of pre-merge-commit and commit-msg hooks`。`git pull --no-v=x` は no-verify と no-verify-signatures の ambiguous(`sr6/abbrev.out`) | 振り分けに `pull` を足すか、tech-debt の guard の行に、`git pull --no-verify` は見ないと書く |

### 前の節の指摘の状態

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| C3-1 | 解消 | 849f5411 が理由の文(`:1437`)に「HEREDOC の形は git commit を単独のコマンドで打ったときだけ通る」を足し、ヘッダーの `:117-120` を直した。`implementer.md`、`git-commit-strategy.md`(root と template)、tech-debt の 124・125 行目は 24876df7 が直した |
| C3-2 | 解消 | `reset_rules` が `--pathspec-from-file` の値を先に読み飛ばす(`:1148`)。`edge_deny` の 3 行(`--pathspec-from-file --`、`--pathspec-from-file f`、`--pathspec-fr --`)は新版 deny/deny、`edge_none` の `--pathspec-from-file=f --` は none/none |
| C3-3 | 未修正、記録済み | 予約語と exec の規則(`:718`、`:723`)はそのまま。tech-debt の guard の限界の行が、等価な変異として記録している |
| C3-4 | 未修正、記録済み | 区切りの規則のコメント(`:537-541`)と `data_first_ok` のコメントはそのまま。同じ行に記録されている。C4-1 は同じ値の違いが printf の引数で効く場合 |
| C3-5 | 一部解消 | (2) は 8e94e76b が直した(`$c` の行のコメントと、許可リストを固定する 3 行)。(1) は残る: `tests/test-pre-bash-guard.sh:836` の「Cycle 2 (P2-4)」、`:957` の「Cross-review cycle 2 (P2-4, P2-5)」(849f5411 が `ac3` から移したときもこの書き方を残した)、`:1061` の「change A」 |

### Tech debt identified

ユーザーはこの run を最後とした。直さない指摘はそのまま持ち越しになる。この commit では `docs/tech-debt/README.md` を変えていない。直さないなら、/sync-docs で guard の限界の行に次をまとめて足してほしい。行の中の「1442 lines at e5c9e6be」「line 154 to its closing quote on line 1399」も、今は 1450 行、155 行目から 1407 行目になっている。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| b3c3fdaa の閉じ方の残り(C4-1 printf と rg の判定が `$'...'` の語の値を読む、C4-2 `opt_is` の 4 文字の下限と `git reset --h` の行、C4-3 コメント、C4-4 `git pull --no-verify`) | C4-1: 旧版が deny にした形を、zsh(この Bash ツールの shell)と bash 5.2 が実行する位置で通す。C4-2: 1 文字の略記の hard reset と、`--m` を使った `--no-verify` を通す(旧版も通す)。C4-3・C4-4: 次に読む人が規則を狭めたり、規則の届く範囲を読み違えたりする | 上限 4 の最後の run で、guard を 1 行でも変えると self-review から /cross-review までをもう一度回すことになる | 次に guard を変える PR。C4-1 と C4-2 は、どちらも 1 語の変更で、489 行の判定は C4-2 の 1 行のほかに変わらない | この節の C4-1〜C4-4 |

### Recommendation(cycle 4、現時点)

- Merge: no-merge(HIGH 1 件。C4-1 は、旧版が deny にした形を shell が実行する位置で通す AC7 の型で、b3c3fdaa が閉じた triage cycle 3 #1 と同じ。直し方は `:923` の 1 語で、489 行の判定は変わらない。直さずに PR にするなら、既知の穴として tech-debt と PR 本文に書く。CRITICAL はない。MEDIUM の C4-2 は旧版も通す形)
- 前の節までの Merge 行はそれぞれの時点の判定で、現時点の判定はこの行
- Follow-ups:
  - C4-1 を直すなら、guard の変更になるので、パイプラインの規則どおり self-review から回し直す。直さないなら、上の Tech debt の 1 行と PR 本文の既知の穴に書く
  - /test: C4-1 の型の行(`$'...'` で書いた printf)は test ファイルにない。C4-2 の `git reset --h` の行は、git 2.49 では本物の hard reset を none に固定している
  - /sync-docs: guard の行数と awk の範囲(1450 行、155〜1407 行目)

### 未確認の点

- C4-1 で新版が none を返すことは、コードと計測用の写しの SRO・SSAFE から読んだ。見張りの語を入れた形は hook に渡していない
- GNU の zgrep(シェルスクリプト)と、Linux の diff・stat などが引数を実行に使うかは調べていない
- 489 行の新旧比較は macOS の awk で、mawk では commit・printf・merge・rebase・reset を含む 136 行だけを jq なしの経路で流した。gawk と busybox の awk では流していない
