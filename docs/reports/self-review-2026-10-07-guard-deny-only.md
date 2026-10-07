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

## Recommendation

- Merge: no-merge(HIGH 1 件、MEDIUM 1 件。CRITICAL はない)
- Follow-ups:
  - H-1: 後ろの語の読み直しを、値がちょうど `sudo`・`git` の語から「空白か改行を含む語」に広げる。git・`builtin`・`source`・`.`・プロセス置換・`env -S` に値がくっついた形の扱いを足し、この種類の形を AC7 の例の集まりに入れる
  - M-1: `base()` で zsh の先頭の `=` を外す
  - L-1: 「Not covered」の一覧と tech-debt の行(b)に、静的な展開の 3 つを足す
  - L-2: `unbq()` の連結をまとめる。残すなら tech-debt に書く
  - 直したら、tech-debt の guard の限界の行の(a)(誤検知)を見直す。読み直しを広げると、NOEXEC にないコマンドの引数の文字列が新しく deny になりうるため
