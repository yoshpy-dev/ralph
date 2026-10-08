# guard-deny-only

- Status: Approved
- Approved: 2026-10-07 sha256:7efd47f36781
- Owner: Claude Code
- Date: 2026-10-07
- Related request: 「guard の残りの穴を塞いでください。そのほかのhooksも含めて、一切確認されないようにしたいです。」(2026-10-07)。ユーザーは「guard の ask を全モードで廃止」「停止する hook は誤検知の分だけ直す」「ECC plugin の block-no-verify はそのまま」を選び、Codex plan advisory の指摘には「Update plan」を選んだ
- Related issue: N/A
- Type: fix
- Branch: fix/guard-deny-only

## Objective

`pre_bash_guard.sh` を、確認(ask)を一切返さず、止めるべきコマンドだけを deny する guard にする。

1. ask の規則(`.git`・`.env` への書き込み、`rm -rf `、`gh pr create`)をすべてのモードで廃止する。PR #206 は bypass でだけ ask を飛ばしていたが、bypass 以外(たとえば cross-review の `claude -p --permission-mode auto`)では確認が出る。ask の規則の取りこぼしと誤検知(tech-debt の guard の行)は、規則ごとなくなる
2. deny の取りこぼしを塞ぐ。いまの guard は文字列の部分一致で判定していて、`git commit -am "$(id)"`、`git push origin --force`、`git -C dir reset --hard` など 18 形を通す(2026-10-07 実測)
3. deny の誤検知を直す。repo が推奨するコミットの形 `git commit -m "$(cat <<'EOF' ... EOF\n)"` を止めている(tech-debt の 124 行目)。コミットメッセージやほかのコマンドの引数に書いた文字(`git commit -m 'drop --force and git reset --hard from docs'`、`echo 'never use sudo here'`)や、`visudo -c` を deny と取り違える
4. `--no-verify` を正確に止める規則を足す。ECC plugin の block-no-verify は残すが、ECC がない環境(下流のプロジェクト)でも止まるようにし、`git commit` と同じコマンドの `grep -n` を取り違えない
5. jq がない環境でも jq のある環境と同じ判定にする。いまの `lib_json.sh` の sed は `\n` や `\t` を戻さないので、本物の改行と、文字としてのバックスラッシュと `n` が同じ出力になり、区別できない(Codex plan advisory が実測で確かめた)

## Scope

- `.claude/hooks/lib_json.sh`(root と template、同じ内容): jq がないときの取り出しで、JSON の文字列のエスケープを正しく戻す。値の切り出しは今の `sed -E` のまま、戻しを `LC_ALL=C` の awk で 1 文字ずつ行う(`\"`、`\\`、`\/`、`\n`、`\t`、`\r`、`\b`、`\f`、`\u0000`〜`\u007F`)。`\u0080` 以上はそのまま残す。guard の判定に要るのは ASCII だけで、UTF-8 への変換は awk の方言で結果が変わる(gawk は UTF-8 のロケールで `%c` を符号化する。consult の指摘)。Claude Code は ASCII 以外の文字を生の UTF-8 で送るので、`\uXXXX` はまれ(推測)。`post_edit_verify.sh` も同じ関数を使う
- `.claude/hooks/pre_bash_guard.sh`(root と template、同じ内容):
  - ask の規則、`permission_mode` の読み取り、bypass の分岐、書き込み先の正規表現を消す
  - deny の判定を、awk で書いた shell に近い字句解析にする。コマンドは stdin で渡す(`awk -v` はエスケープを処理するので使わない)
    - 語は、引用符(`'`、`"`)とバックスラッシュを shell と同じように扱い、引用符を外した値で持つ。`git push "--force"` の語は `--force`、`git commit -m 'remove -n flag'` の `-m` の後ろは 1 語の `remove -n flag`
    - 引用符の外の `;`、`&&`、`||`、`|`、`&`、改行、`(`、`)` で単純コマンドに区切る
    - 中身が実行される場所は、中身をコマンドとして読み直す: 引用符の外とダブルクォートの中の `$(...)` とバッククォート、`sh`・`bash`・`zsh`・`dash`・`ksh` に `-c`(`-lc` などの束を含む)で渡す文字列、`eval` の引数、shell に流し込むヒアドキュメントの本文、パイプで引数なしの shell(`| sh`、`| bash -s`)に流す前段のコマンドの引数。読み直しは深さ 4 まで
    - ヒアドキュメント(`<<WORD`、`<<-WORD`、`<<'WORD'`、`<<"WORD"`、`<<\WORD`)は、本文をデータとして読み飛ばす。区切りに引用符がなければ、本文の中の `$(...)` とバッククォートは展開されるので読み直す
    - 単純コマンドの「コマンドの位置」は、先頭の予約語(`if`、`then`、`else`、`elif`、`do`、`while`、`until`、`!`、`{`、`}`)、リダイレクト(`2>/dev/null`、`>file`、`<file`、`2>&1`、`&>file` など、数字の fd つきも)、`VAR=value` と前置きのコマンド(`env`(とその `-i`・`-u NAME`・`VAR=value`)、`command`、`exec`、`nohup`、`time`、`nice`(`-n N`)、`timeout`(時間)、`xargs`(とそのフラグ)、`stdbuf`)を読み飛ばした最初の語。コマンド名はパスを外した名前(`/usr/bin/sudo` は `sudo`)で比べる
  - deny の規則(どれもモードによらない):
    - `sudo`: コマンドの位置の語が `sudo`
    - force push: コマンドが `git` で、`-C <dir>`、`-c <k=v>`、`--git-dir=…` などの前置きを読み飛ばしたサブコマンドが `push`。引数に `--force`、`--force-with-lease`(`=` つきも)、`-f` を含む短いフラグの束、`+` で始まる refspec のどれかがある
    - hard reset: サブコマンドが `reset` で、引数に `--hard` がある
    - コミットメッセージのコマンド置換: サブコマンドが `commit` で、`-m` / `--message` / `--message=` / 短いフラグの束の `m` の値(くっついた `-m"..."` も)のもとの文字列が、ダブルクォートの中の `$(` かバッククォート、または引用符の外の `$(` かバッククォートを含む。ただし値が、`"$(cat <<'DELIM'`(`<<"DELIM"`、`<<\DELIM` も)の直後に改行、本文、`DELIM` だけの行、`)"` の順に並ぶ形だけは通す。引用符つきの区切りのヒアドキュメントは中身を展開しないため。`git commit -F -`(`--file=-`)に区切りに引用符のないヒアドキュメントを流し、本文に `$(` かバッククォートがある形も止める
    - `--no-verify`: サブコマンドが `commit`、`push`、`merge`、`rebase`、`am` のどれかで、引数に `--no-verify` がある。`commit` では `-n` と、`n` を含む短いフラグの束も止める。束は左から読み、値を取るフラグ(`m`、`F`、`C`、`c`、`t`、`u`、`S`)が出たらそこで止める(`-nm` は止め、`-mn` はメッセージ `n`、`-uno` は `--untracked-files=no` なので止めない)。`git -c core.hooksPath=…`(キーは大文字小文字を区別しない)の後ろのサブコマンドも止める
  - 閉じない引用符などの壊れた入力でも、エラーにならず exit 0 で終わる
  - 前置きのコマンドの一覧に頼らない補い(S2 のあとで、`find -exec sudo`、`watch sudo`、`flock … git push --force` などを旧版より弱く通すと分かったため): コマンドの位置の語が「引数を実行しないと分かっているコマンド」(`echo`、`printf`、`man`、`info`、`whatis`、`apropos`、`which`、`type`、`grep` の仲間、`rg`、`ag`、`cat`、`less`、`more`、`head`、`tail`、`wc`、`sort`、`cut`、`jq`、`ls`、`test`、`[`、`cd`、`true`、`false`、`cp`、`mv`、`rm`、`mkdir`、`touch`、`ln`、`chmod`、`stat`、`file`、`diff`、`git`(自分の規則で見る))でなければ、残りの語を調べる。値のパスを外した名前がちょうど `sudo` で、後ろに語が 1 つ以上ある語があれば deny、ちょうど `git` の語があれば、その語から git の規則を当てる
  - 長いオプションの省略: git は一意な前置きを受け付ける。引数が `--` で始まり、`=` 以降を落として 4 文字以上で、`--force`、`--force-with-lease`、`--hard`、`--no-verify`、`--message`、`--file` のどれかの前置きなら、そのオプションとみなす(git が曖昧として断る形も deny になるが、その形はどうせ失敗する)
  - 止める側に倒す場合: 読み直しの深さ、キューの上限、入れ子の上限のどれかを超えたら deny にし、理由に「入れ子が深すぎる」と書く
  - awk が使えない、または 0 以外で終わったときは、旧版の `case` の glob の deny 4 規則で判定する。awk の終了コードは捨てずに別の変数に取る
  - 見張りとデータ区間(S3 のあとの self-review で、旧版が止めていた形の多くを通すと分かったため。S2c): 字句解析の規則に加えて、旧版の 4 規則と同じ文字列の一致を、生のコマンド全体に当てる(sudo は語の境界つき `(^|[^A-Za-z0-9_.-])sudo[[:space:]]`。`visudo` は外れ、zsh の `=sudo` は当たる)。一致したら deny。ただし一致した場所がすべて「データ区間」の中なら通す。データ区間は深さ 0 のトップレベルだけで決め、入れ子の中は常にデータではない:
    - (a) データを読むだけのコマンド(`echo`、`printf`、`cat`、`head`、`tail`、`wc`、`cut`、`tr`、`grep` の仲間、`--pre` のない `rg`、`ls`、`stat`、`diff`、`test`、`[`、`cd`、`true`、`false`、`which`、`type`)の、最初の引数から単純コマンドの終わりまで。リダイレクトの行き先と置換の部分は除く。そのコマンドの出力を受ける後段がすべて同じ一覧のコマンドで、リダイレクトの行き先が `/dev/null`・`/dev/stderr`・fd の複製だけ(ファイルへの `>`・`>>` と `>(…)` がない)ときだけ成り立つ。`sed`、`awk`、`man`、`less`、`more`、`sort`、`tee`、`jq` はコマンドを実行できるか、ファイルに書くので入れない
    - (b) `git commit` と `git tag` の `-m` / `--message` の値で、置換を含まないもの。推奨の HEREDOC の形(`"$(cat <<'D'` … `D` の行 … `)"` で中身がこれだけのもの)は、引用符つきの 1 語と同じに扱う
    - (c) 区切りに引用符のあるヒアドキュメント(または区切りに引用符がなく本文に置換がないもの)の本文で、(a) の一覧のコマンドか `git commit -F -` に流し、(a) と同じ出力の条件を満たすもの
    - (d) 深さ 0 のコメント(`#` から行末まで)
  - zsh の `=` 展開: 引用符のない `=` で始まる語は、`=` を落として比べる(`=sudo ls` は `sudo ls`)
- `tests/test-pre-bash-guard.sh`: ask を期待していた行はすべて none にする(モードを変えても none)。AC2・AC3 の形を、jq あり・なしの両方で、none / deny の 2 値で厳密に比べる。旧版の guard との比較(下の AC7)も入れる
- `tests/test-lib-json.sh`(新規、名前は既存のテストに合わせて決める): jq を外した PATH で `lib_json.sh` を読み、エスケープの戻しを確かめる
- `docs/tech-debt/README.md`: 124 行目(HEREDOC の誤検知)、127 行目(Codex の ask)、158 行目(ask の取りこぼしと誤検知)、160 行目(テストの穴)を、この PR に合わせて解消済みにするか書き直す
- 文書(`/sync-docs` で確かめる): `.codex/README.md` の 116 行目付近、`.claude/rules/ralph/git-commit-strategy.md` の Enforcement、guard に触れているほかの文書

## Non-goals

- ECC plugin の hook(block-no-verify、commit-quality、config-protection)と security-guidance の hook。ユーザーの選択で残す。plugin の hook を 1 つだけ外す公式の手段はない
- Claude Code 本体が出す確認(重要なパスへの `rm`、`AskUserQuestion`、セッションをまたぐメッセージの保留)。hook でも設定でも止められない(Claude Code の文書)
- 実行前に中身が分からないもの: 変数(`git commit -m "$msg"`、`$cmd`)、エイリアス、関数、ファイルから読むスクリプト(`bash script.sh`、`cat file | sh`)、`git` の alias、リモートで動くコマンド(`ssh host '...'`)
- `.git` や `.env` への書き込みを deny にすること。止めずに通すのがユーザーの選択
- 完全な shell の文法(算術展開、`case` 文、プロセス置換の細部など)。上で読み直すと決めた場所のほかは追わない

## Assumptions

- hook が ask を返さなければ、bypass では確認が出ない。bypass 以外のモードでも、guard 由来の確認はなくなる。ほかの hook はどれも ask を返さない(2026-10-07 に project・グローバル・plugin の hook を洗って確認。plugin の hook は exit 2 や `block` で止めるだけ)
- awk は POSIX の機能だけを使えば、macOS の awk(BWK)、Linux の mawk・gawk で同じに動く。PR #206 の `archive-plan.sh` でも awk を使い、ubuntu の mawk で通った。1 文字ずつの `substr` は BWK awk で長い入力に遅くなりうる(consult の指摘)ので、性能を BWK awk で測る
- 2026-10-07 に、いまの guard(origin/main、1c4cea5a)へ AC2・AC3 の例を渡して測った。AC2 の取りこぼしの例のうち 18 形は通る。AC3 のうち `git commit -m 'drop --force and git reset --hard from docs'`、`echo 'never use sudo here'`、`visudo -c`、推奨の HEREDOC の形は deny になる。jq あり・なしで結果は同じ
- 旧 guard は文字列の部分一致なので、`sh -c 'sudo ls'`、`bash -lc "git push --force"`、`eval "git reset --hard"`、`echo "$(git reset --hard)"`、`xargs sudo ls`、`/usr/bin/sudo ls` も deny にしている(consult と Codex の指摘)。新しい guard は、中身が実行される場所を読み直すことでこれらを deny のまま保つ
- deny が旧版より減るのは、誤検知の形だけにする。誤検知とは、旧版の文字列の一致がすべて、Scope で定めたデータ区間(読むだけのコマンドの引数、`git commit`・`git tag` のメッセージ、データとして読むヒアドキュメントの本文、コメント)の中にある形。`gh pr create --body "$(cat <<'EOF' …)"` のように、データ区間に入らないコマンドの引数に危険な文字を書いた形は、旧版と同じく deny のまま(本文は `--body-file` で渡す。`/pr` の skill はそうしている)

## Affected areas

- `.claude/hooks/lib_json.sh`、`templates/base/.claude/hooks/lib_json.sh`(`post_edit_verify.sh` も使う)
- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- `tests/test-pre-bash-guard.sh`、`tests/test-lib-json.sh`(新規)
- `docs/tech-debt/README.md`
- 文書: `.codex/README.md`(root と template)、`.claude/rules/ralph/git-commit-strategy.md`(root と template)

## Visual review

- 図解ページ: `.harness/state/plan-visual/guard-deny-only.html`
- 自己チェック: ヘッドレス Chrome で全体・`#overview`・`#flow`・`#examples` を撮って確認。Codex plan advisory と consult を受けて設計を直したあと、図 1 に `lib_json.sh` の変更と `tests/test-lib-json.sh` を足し、図 3 に引用符で囲んだフラグ、`core.hooksPath` の大文字、置換や `sh -c` の中身を読み直す行を足して撮り直した。図 3 の「今」の列は、いまの guard に同じ payload を渡した実測

## Design decisions

- **ask は全モードで廃止(ユーザー確定、2026-10-07)**。採らなかった案: bypass 以外では ask を残して精度を上げる
- **停止する hook は誤検知の分だけ直す(ユーザー確定)**。ECC の block-no-verify はそのまま残す(ユーザー確定)。誤検知はコミットを単独のコマンドで打てば起きない
- **deny の判定は、shell に近い awk の字句解析で書く(Codex plan advisory の指摘 1 と consult を受けて、ユーザーが「Update plan」を選択)**。語は引用符を外した値で持ち、引用符でつながった部分は 1 語として扱う。中身が実行される場所(コマンド置換、`sh -c`、`eval`、shell に流すヒアドキュメントとパイプ)は読み直す。最初の案は「引用符の中は数えない」だったが、`git push "--force"` や `echo "$(git reset --hard)"` を見逃すので採らない。正規表現を足していく案も、取りこぼしと誤検知の組み合わせが増え続ける(PR #206 で 3 周かかった)ので採らない
- **`lib_json.sh` で JSON を正しく戻す(Codex plan advisory の指摘 2)**。改行やタブを字句解析の側で推し量る作りでは、本物の改行と文字の `\n` を区別できない
- **前置きの一覧に頼らず、引数を実行しないコマンドだけを例外にする(S2 のあとで決めた。consult の確認つき)**。前置きのコマンドは `find -exec`、`watch`、`flock`、`chroot`、`docker run` など切りがなく、一覧を足し続けると取りこぼしが残る。引数を実行しないと分かっているコマンドを一覧にし、それ以外の後ろにある `sudo`・`git` を見る。`sudo` は後ろに語があるときだけ止めるので、ファイル名としての `sudo`(`touch sudo`)は止めない。残る誤検知(`apt-get install sudo vim` のような形)は tech-debt に書く
- **旧版の文字列の一致を見張りに残し、データだと示せた場所だけを通す(S3 のあとの self-review を受けて決めた。consult の確認つき)**。字句解析が実行の場所を見つけて止める作りでは、実行のしかた(`find -exec sh -c '…'`、`watch '…'`、`csh -c`、`source`、`git rebase -x` など)を足し続けることになり、self-review では例の外の 49 形のうち 44 形を旧版より弱く通した。逆に、旧版の一致を出発点にし、データだと示せた場所(読むだけのコマンドの引数、コミットメッセージ、データとして読むヒアドキュメントの本文、コメント)だけを外せば、旧版より減るのは示せた誤検知だけになる。字句解析の規則(`git push origin --force` などの新しい検出)は残す
- **guard が自分で判定しきれないときは止める側に倒す**。深さやキューの上限を超えた入力は deny、awk が使えないときは旧版の規則に戻す。S2 は awk が失敗すると何も止めずに通していた
- **HEREDOC の形は、形がそろったときだけ通す**。区切りの直後が改行で、本文のあとに区切りだけの行と `)"` が続くこと。少しでも違えば(区切りに引用符がない、`<<'EOF'; id` のように続きがある、`)"` のあとに続きがある)今までどおり deny にする(consult の指摘)
- Critical forks: None。字句解析の細部(前置きのコマンドの一覧、値を取る短いフラグの一覧、読み直しの深さ)は 1 slice 以内でやり直せる

## Acceptance criteria

- [x] AC1: `pre_bash_guard.sh`(root と template、同じ内容)は、どのモード(`permission_mode` なし、`default`、`auto`、`bypassPermissions`)でも ask を返さない。PR #206 のテストで ask を期待していた行(`.git`・`.env` への書き込み、`rm -rf`、`gh pr create`)はすべて none になる
- [x] AC2: 次がどのモードでも deny になる。
  - sudo: `sudo ls`、`sudo` + タブ + `ls`、`/usr/bin/sudo ls`、`env FOO=1 sudo ls`、`nohup sudo ls`、`xargs sudo ls`、`sh -c 'sudo ls'`、`if sudo ls; then :; fi`、`{ sudo ls; }`、`! sudo ls`、`2>/dev/null sudo ls`、`find . -exec sudo rm x \;`、`watch sudo ls`、`flock /tmp/l sudo ls`、`chroot /x sudo ls`
  - force push: `</dev/null git push --force`、`git push --force`、`git push origin --force`、`git push -f`、`git push origin -uf main`、`git push --force-with-lease`、`git push origin +main`、`git -C dir push --force`、`git push "--force" origin main`、`bash -lc "git push --force"`、`find . -exec git push --force \;`、`flock /tmp/l git push --force`、`git push --force-with`、`git push --force-with=main`
  - hard reset: `git reset --ha`、`git reset --har`、`git reset --hard`、`git -C dir reset --hard HEAD~1`、`git reset -q --hard`、`eval "git reset --hard"`、`echo "$(git reset --hard)"`、``echo "`git reset --hard`"``、`echo "git reset --hard" | sh`、`sh` に流すヒアドキュメントの本文の `git push --force`
  - コミットメッセージの置換: `git commit -m "$(id)"`、``git commit -m "`id`"``、`git commit -am "$(id)"`、`git commit -m"$(id)"`、`git commit --message "$(id)"`、`git commit --message="$(id)"`、`git commit --mess "$(id)"`、`git commit -m 'x' -m "$(id)"`、`rm -rf x && git commit -m "$(id)"`、`git commit -m "$(cat <<'EOF'; id` で始まる形、区切りに引用符のないヒアドキュメントを `git commit -F -` に流して本文に `$(id)` がある形
  - 文字列ごと実行する形とファイルへの書き込み(S2c): `find . -exec sh -c 'sudo ls' \;`、`watch 'git push --force'`、`csh -c 'sudo ls'`、`tcsh -c 'git reset --hard'`、`source <(echo 'sudo ls')`、`. <(echo 'git push --force')`、`builtin eval 'git push --force'`、`git rebase -x 'git push --force' main`、`git submodule foreach 'git push --force'`、`echo 'sudo ls' | xargs -I{} sh -c {}`、`echo 'sudo ls' > >(sh)`、`echo 'sudo ls' >> ~/.zshrc`、`echo 'git push --force' | tee x.sh`、`cat <<'EOF' | sh` の本文の `sudo ls`、`cat > notes.md <<EOF` の本文の `git push --force`、`=sudo ls`、`=git push --force`
  - `--no-verify`: `git commit --no-veri -m x`、`git commit --no-verify -m x`、`git commit -n -m x`、`git commit -nm x`、`git push --no-verify`、`git merge --no-verify x`、`git -c core.hooksPath=/dev/null commit -m x`、`git -c Core.HooksPath=/dev/null commit -m x`
- [x] AC3: 次がどのモードでも none になる。`echo 'never use sudo here'`、`echo sudo ls`、`grep sudo file`、`touch sudo`、`apt-get install sudo`、`visudo -c`、`man sudo`、`git push origin main`、`git push -u origin main`、`git reset --soft HEAD~1`、`git commit -m 'remove -n flag'`、`git commit -m 'drop --force and git reset --hard from docs'`、`git commit -mn`(メッセージ `n`)、`git commit -uno -m x`、`git commit -m 'fix: x' && grep -n foo file`、`git commit -F msg.txt; sed -n 1,5p file`、`git commit -m "$(cat <<'EOF'` で始まる複数行の推奨の形、区切りに引用符のあるヒアドキュメントを `git commit -F -` に流して本文に `git push --force` や `$(id)` と書いた形、`echo never use sudo here`(引用符なし)、`echo hi # sudo ls`(コメント)、`grep -n 'git push --force' docs.md`、`echo 'sudo ls' > /dev/null`、`echo 'git push --force' | grep force`、`git log --no-verify-signatures`、`printf 'a\ngit push --force'`(シングルクォートの中の文字の `\n`)、`ls .git/ 2>&1`、`echo x > .env`、`rm -rf build/`、`gh pr create --title t`
- [x] AC4: jq がない環境でも AC1〜AC3 が同じ結果になる。`lib_json.sh` は jq がなくても `\n`・`\t`・`\"`・`\\`・`\/`・ASCII の範囲の `\uXXXX` を戻し(`\u0080` 以上とサロゲートペアはそのまま残す)、本物の改行と文字の `\n` を区別する(`tests/test-lib-json.sh`)。`tests/test-pre-bash-guard.sh` が jq あり・なしの両方で、期待値を none / deny の 2 値で厳密に比べる
- [x] AC5: `docs/tech-debt/README.md` の 124・127・158・160 行目が、この PR に合わせて解消済みか書き直されている。guard の挙動を説明する文書が ask に触れていない
- [x] AC6: `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-pipeline-sync.sh`、`bash scripts/check-template-purity.sh`、`./scripts/run-verify.sh` が通る
- [x] AC7: 旧版(origin/main の guard)が deny にする形は、新版でも deny になる。例外は、見張りの一致がすべてデータ区間の中にある形だけで、テストに入れる例外はすべて AC3 に挙げる。比較の例の集まりには、S3 のあとの self-review が挙げた実行のしかたの種類(上の AC2 の「文字列ごと実行する形」)を入れる。テストが旧版と新版に同じ例の集まり(AC2 と AC3 の全部と、PR #206 のテストの deny の行)を渡して確かめる
- [x] AC8: 200 KB のコマンドで、guard が macOS の awk でも 5 秒以内に終わる(PR #206 の記録では、旧版の jq の経路で 28 秒)
- [x] AC9: 止める側に倒す場合が働く。読み直しの深さ・キュー・入れ子の上限を超えた入力は deny になる。PATH から awk を外すと、旧版の 4 規則で判定する(`sudo ls`、`git push --force`、`git reset --hard`、`git commit -m "$(id)"` が deny になる)。テストで確かめる

## Implementation outline

1. S1: `lib_json.sh` の JSON の戻しとテスト(AC4 の前半)
2. S2: guard の書き直し(ask の廃止、字句解析、deny の規則)とテスト(AC1〜AC4、AC7、AC8)。S2b: 前置きの一覧に頼らない補い、長いオプションの省略、止める側に倒す場合とテスト(AC2、AC3、AC9)。S2c: 見張りとデータ区間、zsh の `=` 展開、self-review の LOW(`unbq()` の 2 乗の時間、guard の「見ないもの」のコメント)とテスト(AC2、AC3、AC7)
3. S3: tech-debt と文書(AC5)。`/sync-docs` と重なる分はそちらで確かめる

## Verify plan

- Static analysis checks: shellcheck(guard、`lib_json.sh`、テスト)、`sh -n`・`dash -n`、`check-sync.sh`、`check-template-purity.sh`
- Spec compliance criteria to confirm: AC1〜AC8。とくに AC7(旧版の deny が新版でも deny か)を、テストとは別の例でも確かめる
- Documentation drift to check: guard の挙動を説明する文書と tech-debt の行
- Evidence to capture: verify レポート

## Test plan

- Unit tests: `tests/test-pre-bash-guard.sh`(AC1〜AC3、AC7)、`tests/test-lib-json.sh`(AC4)
- Integration tests: `./scripts/run-verify.sh` の全体
- Regression tests: `./scripts/run-test.sh` の全体、`tests/test-post-edit-verify.sh`(`lib_json.sh` のもう 1 つの利用者)
- Edge cases: 引用符で囲んだフラグ、バックスラッシュでエスケープした引用符と空白、複数行のコマンド、`$(...)` の入れ子、ダブルクォートの中のバッククォート、`git -C` と `git -c` の前置き、短いフラグの束、ヒアドキュメントの区切りの 5 通り、閉じない引用符とかっこ、読み直しの深さの上限、`\uXXXX`(ASCII と日本語)
- Evidence to capture: test レポート。ubuntu:24.04(mawk、GNU grep・sed、dash)でも guard と `lib_json.sh` のテストを流す

## Risks and mitigations

- `.git`・`.env` への書き込みや `rm -rf` が、どのモードでも確認なしで走る → ユーザーが選んだ挙動。deny の規則と Claude Code 本体の確認は残る。PR 本文に書く
- 字句解析の誤りで、旧版が deny にしていた形を通してしまう → AC7 の比較テストで固定する。旧版の deny が新版で none になる形は、AC3 の誤検知の一覧にあるものだけ
- 字句解析が大きくなり、読みにくくなる → 1 つの awk のプログラムを、字句解析、単純コマンドの組み立て、規則の判定の 3 つの関数に分け、それぞれにコメントを書く
- awk の方言の差 → POSIX の機能だけを使い、macOS の awk と ubuntu の mawk の両方でテストを流す。`split(s, a, "")` のような POSIX にない書き方は使わない
- 長いコマンドで遅くなる → AC8 で測る。遅ければ `match()` で次の特殊文字まで飛ぶ作りにする
- `lib_json.sh` の戻しを変えると、`post_edit_verify.sh` の `file_path` の取り出しが変わる → エスケープを含まないパスの結果は変わらない。`tests/test-post-edit-verify.sh` を流す
- 壊れた入力で hook がエラーになる → exit 0 で終わることをテストで確かめる。壊れた入力の判定は deny でも none でもよい

## Rollout or rollback notes

- hook とテストと文書の変更。戻すときは PR を revert する
- 下流には次のリリースの `ralph upgrade` で届く

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-07: ユーザーが承認ゲートで Approve。Codex plan advisory の 2 件は「Update plan」で反映、consult(consult-plan-denyonly)は 2 回目で「進めてよい」
- [x] Branch created
- [x] Implementation started
  - S1 完了(181e1065、implementer/opus): `lib_json.sh` が jq なしでも JSON のエスケープを戻す(512 文字の窓で読む awk)。`tests/test-lib-json.sh`(125 件)。guard のテストの jq なしの PATH に awk を足した(範囲を広げて承認)
  - S2 完了(e48797a2、implementer/opus): guard を deny だけにし、POSIX awk の字句解析で判定。テスト 1057 件、AC7 の例外 7 件は AC3 と一致、200 KB で macOS の awk 0.28〜0.83 秒。比較用の例の外で、旧版より弱く通す形(`find -exec sudo`、`flock … git push --force`、`git push --force-with`)と、awk が使えないと何も止めないことが分かった。consult(consult-plan-denyonly)と相談して Scope に 4 点を足し、ユーザーが再承認(digest 87e47b5feb8e → bf33328dae82)
  - S2b 完了(5b0b20d5、implementer/opus): 引数を実行しないコマンド以外の後ろの sudo・git を見る補い、長いオプションの省略、止める側に倒す場合(深さ・キュー・入れ子の上限、awk がないときは旧版の 4 規則)。テスト 1203 件、AC7 の例外 9 件。入れ子の上限は mawk の eval stack に合わせて 24 にした。同じ git の後ろの読み直しを次の git の手前までにして、2 乗の時間を避けた
  - 追加の修正(13b36abd、inline: 2 ファイルの分岐とテスト 1 件): jq も awk もないと `lib_json.sh` がコマンドを返さず guard が何も止めなかったので、sed だけの戻しに落とすようにした
  - S3 完了(c133bde5、implementer/sonnet): tech-debt の 124・127・159 行目を解消済みに、158・160 行目を残る限界に書き直した。`.codex/README.md` と `post_edit_verify.sh` のコメントを直した
  - AC1〜AC9: テスト(`tests/test-pre-bash-guard.sh` 1203 件、`tests/test-lib-json.sh` 126 件)と `run-verify.sh` で確かめた(2026-10-07)
  - self-review(511f6382、reviewer/opus): no-merge。HIGH は、例の外の 49 形のうち 44 形を旧版より弱く通すこと(文字列ごと実行する形、csh・tcsh の `-c`、`source`・`.`・プロセス置換、git の `rebase -x`・`submodule foreach` など)。MEDIUM は zsh の `=sudo`。consult と相談し、旧版の一致を見張りに残してデータ区間だけを通す作り(S2c)に変えることにした。AC2・AC3・AC7 を書き直してチェックを外し、ユーザーが再承認(digest bf33328dae82 → 7efd47f36781)
  - S2c 完了(df0a50d5、implementer/opus): 旧版の 4 規則を見張りとして生の文字列に当て、一致がすべて深さ 0 のデータ区間に収まるときだけ通す。テスト 1508 件(ubuntu でも 1508/0)、AC7 の例外 13 件は AC3 と一致。main を取り込んだ(9eca7573、衝突は `docs/tech-debt/README.md` だけ)
  - self-review の 2 回目(46c96511、reviewer/opus): merge。1 回目の HIGH・MEDIUM は S2c で解消。LOW 2 件(`my-sudo ls` の差、tech-debt の guard の行が古い)
  - verify(1346740f、verifier/opus): pass。AC1〜AC9 を満たす。LOW 2 件(V-1 `my-sudo ls`・`x.sudo ls` は AC7 の比較の例に入らない、V-2 テストの上限 10 秒と AC8 の 5 秒)
  - test(095af1d7、tester/opus): pass。`tests/test-pre-bash-guard.sh` は 1576/0(39ed2759 で 68 件を足した)、`tests/test-lib-json.sh` は 126/0、`run-test.sh` と `run-verify.sh` も通る。mutation は 44 個すべてが赤になる(足す前は 8 個が緑のまま)。1 回目の `run-test.sh` で落ちた `tests/test-secret-scan.sh` の 1 件は固定の `/tmp` パスに同時実行が重なったもので、この diff とは関係しない(tech-debt に記録)
  - cross-review の 1 周目(f29d4132、Codex): ACTION_REQUIRED 3 件。グループ・複合コマンドの中の単純コマンドにデータ区間を与える、`$(...)` の中の改行で外側のヒアドキュメントを読み始める、引用符のない区切りのヒアドキュメントで行継続をつながない。どれも旧版が deny にした形を通す。ユーザーは「直してやり直す」を選んだ。consult(consult-plan-xrfix)が同じ型の穴をあと 3 つ挙げた(`exec` のリダイレクトと fd 3 以上への複製、`printf -v`、区切り語の中の行継続)。6 つとも probe で再現した
  - cross-review の指摘の修正(46806dc9、implementer/opus): テキストの行き先を guard が見分けられないコマンド(グループ・複合コマンド、`exec` のリダイレクト)では、データ区間を 1 つも与えず、見張りが旧版どおりに決める。ヒアドキュメントの読み始めを文脈ごとに持ち、引用符のない区切りでは行継続をつないでから区切りと比べる。fd の複製で安全とみなすのを 0〜2 に限り、`printf -v` を読むだけのコマンドから外した。45 行目の「fd の複製」を 0〜2 への複製に読み替えたのは、止める側への逸脱。テスト 1688 件。旧版も deny にしていた `(echo sudo ls)` と `if grep -q 'sudo ' file; …` は deny に移した(AC3 には入っていない)。最初の implementer は途中で止まり、何も変えていない(安全側の判定で応答が止められた)。本物の shell で形を探す手順を外して頼み直した
  - 2 周目: self-review(8f852e4f)は merge で LOW 6 件。P2-1・P2-2・V2-3 のコメントは 0ef6fc4f・63b6743a で直した。P2-5 は bash・zsh・dash で確かめて穴ではなかった(`$(...)` の中のヒアドキュメントは、文脈が閉じると本文なしで終わる)。P2-3(`&>/dev/null` も deny になる。旧版も deny)は直さず記録に回す。verify(7eeefee5)は pass
  - 2 周目の test(6cca4ce4、tester/opus)は fail。F2-1: ダブルクォートの中で `$` と `(` を行継続で分けた形を新版が通す(bash と dash は置換として実行する。旧版は deny)。S2c からある穴。consult(consult-plan-xrfix)と相談し、2 周目の中で直して self-review → verify → test をやり直すことにした(cycle は 2 のまま)
  - F2-1 の修正(4e829e34、inline: 1 行とコメント、テスト 3 形): コマンドのどこかに行継続があれば、データ区間を 1 つも与えない。引用符を見ないので、ただの文字の `\` と改行(単一引用符の中、引用符つきのヒアドキュメントの本文やコメントの行末、`\\` のあと)でも止めすぎるが、どれも旧版と同じ deny。テスト 1716 件
  - 2 周目の self-review のやり直し(373fa29d)は merge で LOW 3 件、verify のやり直し(3950ffdd)は pass で LOW 2 件。どれもコメントのずれで、c61ab2bf と次のコミットで直した
  - 2 周目の test のやり直し(facd295b、tester/opus): pass。`tests/test-pre-bash-guard.sh` は 1730/0(180c7389 で 14 件を足した。ubuntu の mawk と gawk でも 1730/0)、`tests/test-lib-json.sh` は 126/0、`run-test.sh` と `run-verify.sh` も通る。mutation は 22 個のうち 19 個が赤、残る 3 個(J02、L03、N02)は等価。F2-1 の形は新版・旧版とも deny になった。字句解析だけが止める形(`$` と `(` を行継続で分け、中身が `(id)` や `git push origin --force`)は新旧とも none のままで、tech-debt に記録した
  - 2 周目の sync-docs(doc-maintainer/sonnet): tech-debt の guard の限界の行とテストの穴の行を 4e829e34 の状態に直した(データ区間が 1 つもなくなる 3 つの場合、それに伴う誤検知、字句解析だけの穴、残るコメントのずれ、1730/0)。124・125 行目の「推奨の形は通る」に行末の `\` の例外を足し、166 行目に再実行の結果を足した。`internal/org/prompts/implementer.md` の guard の説明を、推奨の HEREDOC 形式は通ると直した
  - main を取り込んだ(e1dfb422、衝突は `docs/tech-debt/README.md` の末尾だけ。両側の行を残した)
  - cross-review の 2 周目(a9af4e05、Codex): ACTION_REQUIRED 3 件(区切りの `$'\x45'` を字句解析が `x45` と読む、`env -S` の後ろの引数、zsh の `builtin exec` のリダイレクト。どれも旧版が deny にした形を通す)と WORTH_CONSIDERING 2 件(`git push -ofoo` と `git reset -- --hard` を新版が止める誤検知)。同じ型の穴は 3 回目。上限に達し、ユーザーは「上限を 3 に上げて直す」を選んだ(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3`。cross-review の手順どおり cycle-count.json は 2 のまま)。consult(consult-plan-xrfix)と相談し、条件を足すのをやめて許可リストにした
  - 許可リストの修正(a9ef82b1、implementer/opus): トップレベルの単純コマンドの 1 語目(前置きと代入を読み飛ばす前の値)がすべて読むだけのコマンドか `git` で、値に `/` がないときだけデータ区間を与える。ヒアドキュメントの区切りに `$` かバッククォートがあればデータ区間を与えない。git push は値を取るオプション(`--repo`、`--receive-pack`、`--exec`、`--recurse-submodules`、`-o`・`--push-option`)の値を読み飛ばし、git reset は `--` で走査を止める。止める側への逸脱: 前置きつきの読むだけのコマンド(`env echo …`、`nice grep …`)や `/bin/echo` のような道のある 1 語目は、旧版と同じく見張りが決める。none から deny に変わったテストの例は `=echo sudo ls` の 1 件で、旧版も deny。テスト 1800 件
  - 上限 3 の 3 回目の run: self-review(48628bd2)は merge で MEDIUM 1・LOW 4。C3-1 は推奨の HEREDOC のコミット形が、同じ呼び出しに読むだけでも git でもないコマンドがあると止まること(旧版も deny)。許可リストからこの形だけを外すと、同じ呼び出しの関数定義が置換の中身を変えられるので、外さずに文書と理由の文で「単独のコマンドで打つ」と伝える。849f5411(inline)で理由の文とヘッダーを直し、C3-2(`git reset --pathspec-from-file -- --hard` を通す)を直した。implementer が `ac3` の配列に入れた git の none の 6 件は D 節に移した(`ac3` は plan の AC3 と一致させる)。テスト 1794 件。verify(92dc5df8)は pass で LOW 3
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
