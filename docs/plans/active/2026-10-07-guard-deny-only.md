# guard-deny-only

- Status: Approved
- Approved: 2026-10-07 sha256:87e47b5feb8e
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
- deny が旧版より減るのは、誤検知の形だけにする。誤検知とは、危険なコマンドの文字が、実行されない場所(シングルクォートの中、コマンドの引数、データとして読むヒアドキュメントの本文)にだけある形

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
- **HEREDOC の形は、形がそろったときだけ通す**。区切りの直後が改行で、本文のあとに区切りだけの行と `)"` が続くこと。少しでも違えば(区切りに引用符がない、`<<'EOF'; id` のように続きがある、`)"` のあとに続きがある)今までどおり deny にする(consult の指摘)
- Critical forks: None。字句解析の細部(前置きのコマンドの一覧、値を取る短いフラグの一覧、読み直しの深さ)は 1 slice 以内でやり直せる

## Acceptance criteria

- [ ] AC1: `pre_bash_guard.sh`(root と template、同じ内容)は、どのモード(`permission_mode` なし、`default`、`auto`、`bypassPermissions`)でも ask を返さない。PR #206 のテストで ask を期待していた行(`.git`・`.env` への書き込み、`rm -rf`、`gh pr create`)はすべて none になる
- [ ] AC2: 次がどのモードでも deny になる。
  - sudo: `sudo ls`、`sudo` + タブ + `ls`、`/usr/bin/sudo ls`、`env FOO=1 sudo ls`、`nohup sudo ls`、`xargs sudo ls`、`sh -c 'sudo ls'`、`if sudo ls; then :; fi`、`{ sudo ls; }`、`! sudo ls`、`2>/dev/null sudo ls`
  - force push: `</dev/null git push --force`、`git push --force`、`git push origin --force`、`git push -f`、`git push origin -uf main`、`git push --force-with-lease`、`git push origin +main`、`git -C dir push --force`、`git push "--force" origin main`、`bash -lc "git push --force"`
  - hard reset: `git reset --hard`、`git -C dir reset --hard HEAD~1`、`git reset -q --hard`、`eval "git reset --hard"`、`echo "$(git reset --hard)"`、``echo "`git reset --hard`"``、`echo "git reset --hard" | sh`、`sh` に流すヒアドキュメントの本文の `git push --force`
  - コミットメッセージの置換: `git commit -m "$(id)"`、``git commit -m "`id`"``、`git commit -am "$(id)"`、`git commit -m"$(id)"`、`git commit --message "$(id)"`、`git commit --message="$(id)"`、`git commit -m 'x' -m "$(id)"`、`rm -rf x && git commit -m "$(id)"`、`git commit -m "$(cat <<'EOF'; id` で始まる形、区切りに引用符のないヒアドキュメントを `git commit -F -` に流して本文に `$(id)` がある形
  - `--no-verify`: `git commit --no-verify -m x`、`git commit -n -m x`、`git commit -nm x`、`git push --no-verify`、`git merge --no-verify x`、`git -c core.hooksPath=/dev/null commit -m x`、`git -c Core.HooksPath=/dev/null commit -m x`
- [ ] AC3: 次がどのモードでも none になる。`echo 'never use sudo here'`、`visudo -c`、`man sudo`、`git push origin main`、`git push -u origin main`、`git reset --soft HEAD~1`、`git commit -m 'remove -n flag'`、`git commit -m 'drop --force and git reset --hard from docs'`、`git commit -mn`(メッセージ `n`)、`git commit -uno -m x`、`git commit -m 'fix: x' && grep -n foo file`、`git commit -F msg.txt; sed -n 1,5p file`、`git commit -m "$(cat <<'EOF'` で始まる複数行の推奨の形、区切りに引用符のあるヒアドキュメントを `git commit -F -` に流して本文に `git push --force` や `$(id)` と書いた形、`cat > notes.md <<EOF` の本文に `git push --force` と書いた形(本文はデータ)、`git log --no-verify-signatures`、`printf 'a\ngit push --force'`(シングルクォートの中の文字の `\n`)、`ls .git/ 2>&1`、`echo x > .env`、`rm -rf build/`、`gh pr create --title t`
- [ ] AC4: jq がない環境でも AC1〜AC3 が同じ結果になる。`lib_json.sh` は jq がなくても `\n`・`\t`・`\"`・`\\`・`\/`・ASCII の範囲の `\uXXXX` を戻し(`\u0080` 以上とサロゲートペアはそのまま残す)、本物の改行と文字の `\n` を区別する(`tests/test-lib-json.sh`)。`tests/test-pre-bash-guard.sh` が jq あり・なしの両方で、期待値を none / deny の 2 値で厳密に比べる
- [ ] AC5: `docs/tech-debt/README.md` の 124・127・158・160 行目が、この PR に合わせて解消済みか書き直されている。guard の挙動を説明する文書が ask に触れていない
- [ ] AC6: `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-pipeline-sync.sh`、`bash scripts/check-template-purity.sh`、`./scripts/run-verify.sh` が通る
- [ ] AC7: 旧版(origin/main の guard)が deny にする形は、新版でも deny になる。例外は Assumptions の最後の項目で定めた誤検知の形だけで、テストに入れる例外はすべて AC3 に挙げる。テストが旧版と新版に同じ例の集まり(AC2 と AC3 の全部と、PR #206 のテストの deny の行)を渡して確かめる
- [ ] AC8: 200 KB のコマンドで、guard が macOS の awk でも 5 秒以内に終わる(PR #206 の記録では、旧版の jq の経路で 28 秒)

## Implementation outline

1. S1: `lib_json.sh` の JSON の戻しとテスト(AC4 の前半)
2. S2: guard の書き直し(ask の廃止、字句解析、deny の規則)とテスト(AC1〜AC4、AC7、AC8)
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
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
