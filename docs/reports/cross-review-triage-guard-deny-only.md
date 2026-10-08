# Cross-review triage report: guard-deny-only

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md
- Base branch: main (51855166)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/3 (cap raised to 3 by the user; cycle-count.json stays at 2, as the cross-review skill prescribes for a cap raise)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-07-guard-deny-only.md(AC7: 旧版が deny にする形は新版でも deny。例外は見張りの一致がすべてデータ区間に収まる場合だけ)
- Self-review report: docs/reports/self-review-2026-10-07-guard-deny-only.md(cycle 3 は merge。MEDIUM の C3-1 と LOW の C3-2 は 849f5411 で直した)
- Verify report: docs/reports/verify-2026-10-07-guard-deny-only.md(cycle 3 は pass)
- Implementation context summary:
  - cross-review の 2 周目のあと、ユーザーが上限を 3 に上げた。a9ef82b1 で、データ区間を与える条件を許可リストにした。トップレベルの単純コマンドの 1 語目がすべて読むだけのコマンドか `git` で、値に `/` がないときだけ与える
  - この回の review は HEAD 24876df7 に対して、read-only の sandbox で動いた。テストスイートは回していない(本人の申告)
- 再現(2026-10-08、`scratchpad/xr3/c3p*.txt` の probe、jq あり・なしの両方)
  - 指摘 1: `printf '%n' 'arr[$(sudo id; echo 1)]'` は新版 none、旧版 deny
  - 同じ型(Codex の指摘にはない): `test -v 'arr[$(sudo id; echo 1)]'` も新版 none、旧版 deny
  - zsh 5.9(`zsh -f`)に、`echo … >&2` だけを置換に入れた無害な形を流した。`printf '%n'`、`test -v`、`[ -v` の 3 つとも置換を実行した。`printf '\x25n'` は `%n` を文字として出しただけで、実行しなかった(`sem7.zsh`)。macOS の bash 3.2 には `test -v` がなく、`printf` は `%n` を受け付けない
  - 指摘 2: `git merge -m -- --no-verify feature` と `git commit --trail -- --no-verify -m fix` は、新版・旧版とも none。`git merge -m --no-verify feature`(メッセージが `--no-verify` という文字)は、新版 deny、旧版 none
- 同じ型の穴(データ区間と判定したテキストが実行される)が出るのは 4 回目になる。今回の形は、読むだけのコマンドの一覧に入れた builtin が、zsh では引数を変数の名前や添字として評価する、というもの

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] zsh の `printf '%n'` は、引数を代入先の名前として読み、配列の添字の中の置換を実行する。`-v` だけを読むだけのコマンドから外していて、`%n` は外していない。引数がデータ区間になり、見張りの `sudo` の一致が消える | 再現した(zsh)。同じ型として、zsh の `test -v` と `[ -v` も添字の中の置換を実行し、guard は通す。旧版はどちらも deny なので AC7 に反する。直し方の候補は、`printf` を読むだけのコマンドとして扱うのを書式に `%` がない場合に限る、`test` と `[` を読むだけのコマンドの一覧から外す、の 2 つ。AC3 の `printf 'a\ngit push --force'` は `%` を含まないので、変わらない | `.claude/hooks/pre_bash_guard.sh`(stage_note、DATACMD の一覧)、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P1] `--no-verify` の走査(merge・rebase・am など)と commit の走査が、値を取るオプションの値を先に読み飛ばさずに `--` を区切りとみなす。`git merge -m -- --no-verify feature` と、`--trailer` を略した `git commit --trail -- --no-verify -m fix` が通る。逆に `git merge -m --no-verify feature` を止める | 再現した。旧版には `--no-verify` の規則がなく、どちらの形も旧版も通すので、旧版からの後退ではない。新しく足した規則に抜け道がある、という問題。`-m` の値に `--` をわざわざ書く形なので、偶然には起きにくい。直すなら、`--no-verify` の走査は `--` で止めない(止める側に倒す)、commit の値を取る長いオプションを `opt_is` の略記で読み飛ばす | 同上(no_verify_rules、commit_rules) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 付録: cycle 2(2026-10-08、HEAD e1dfb422 に対する review)

- Cycle: 2/2(上限に達した)。指摘 5 件、分類は ACTION_REQUIRED 3・WORTH_CONSIDERING 2・DISMISSED 0。ユーザーは「上限を 3 に上げて直す」を選び、a9ef82b1 と 849f5411 で直した
- 再現(`scratchpad/xr3/c2p*.txt` の probe、jq あり・なしの両方)。指摘 1〜3 は新版 none・旧版 deny、指摘 4・5 は新版 deny・旧版 none。指摘 1 の区切り `$'\x45'` は、bash でも zsh でも `E` になり、`E` の行のあとのコマンドが実行される(`sem6.sh`)

### cycle 2 の分類(ACTION_REQUIRED)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] ヒアドキュメントの区切りの `$'\x45'` を字句解析が `x45` と読む。bash は `E` と読むので、`E` の行のあとの `sudo ls` を guard はヒアドキュメントの本文(データ)として扱い、旧版の deny を消す | 再現した(bash・zsh)。区切りの語に `$` があればデータ区間を与えない、とすれば閉じる | `.claude/hooks/pre_bash_guard.sh`(lex_redir)、template、テスト |
| 2 | [P1] `env -S` の分割した文字列と、その後ろの引数を別々に読む。`env -S 'sh -c "eval \$2" --' echo 'sudo ls'` は 2 番目の引数を実行するが、guard は後ろを独立した `echo` と見てデータ区間を与える | 再現した。トップレベルの単純コマンドの元の 1 語目で判定する許可リストにすれば閉じる | 同上(cmd_pos、stage_note) |
| 3 | [P1] zsh の `builtin exec >run.sh` は shell の標準出力を付け替えるが、`cmd_pos` は `builtin` で止まるので `EXEC_SEEN` が立たない | 再現した。2 と同じ直し方で閉じる | 同上(cmd_pos、end_cmd) |

### cycle 2 の分類(WORTH_CONSIDERING)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 4 | [P2] `git push origin -ofoo` を force push として止める。git は `-o` の値と読む | 再現した。旧版は通していた形で、新版の規則が増やした誤検知。値を取るオプションの値を先に読み飛ばせば直る | 同上(git_rules の push) |
| 5 | [P2] `git reset HEAD -- --hard` を hard reset として止める。`--` のあとは pathspec | 再現した。旧版は通していた形で、新版の規則が増やした誤検知。`--` で止めれば直る | 同上(git_rules の reset) |

## 付録: cycle 1(2026-10-08、HEAD 27581ff5 に対する review)

- Base branch: main (f423f230)。Cycle: 1/2。指摘 3 件、分類は ACTION_REQUIRED 3・WORTH_CONSIDERING 0・DISMISSED 0。ユーザーは「直してやり直す」を選び、46806dc9 で直した
- 再現(`scratchpad/xr1/` の probe、jq あり・なしの両方): 新版 none、旧版 deny
  - 指摘 1: `(echo 'git push --force') | sh`、`{ echo 'sudo ls'; } | sh`、`{ echo 'git reset --hard'; } > run.sh`。同じ型として `if`・`for`・`case`・関数の出力を `sh` に流す形と、`echo 'sudo ls' | if true; then sh; fi`
  - 指摘 2: `cat <<'OUT' "$(` の改行のあとに `git push --force`、`)"`、`x`、`OUT` が続く形。bash・zsh・dash・sh のどれでも `$(...)` の中が実行される
  - 指摘 3: `cat <<EOF` のあとに `EO\` と `F` の 2 行、続けて `git push --force`。bash・zsh・sh は 2 行をつないで区切りと見なし、続く行を実行する(dash はつながない)。`<<-EOF` でも同じ

### cycle 1 の分類(ACTION_REQUIRED)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] サブシェル・ブレースの中の単純コマンドに、外側のパイプとリダイレクトが分かる前にデータ区間を与える。`(echo 'git push --force') \| sh` を新版は止めない | 再現した。`if`・`for`・`case`・関数・パイプの先の `if` でも同じ。グループや複合コマンドが 1 つでもあるコマンドではデータ区間を与えない(見張りが旧版どおりに決める)形にすれば、型ごと閉じる | `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |
| 2 | [P1] `$(...)` の中の改行で、外側のコマンドの未読のヒアドキュメントを読み始める。実行される置換の中身がヒアドキュメントの本文として扱われ、見張りから外れる | 再現した。実際の shell は置換の中を実行する。未読のヒアドキュメントは読み始めた文脈のものとして持ち、入れ子の文脈の改行では読まない。持ち方が崩れる場合はデータ区間を与えない | 同上 |
| 3 | [P1] 区切りに引用符のないヒアドキュメントで、バックスラッシュと改行の組を外さずに区切りの行と比べる。`EO\` と `F` の 2 行で終わったあとのコマンドが本文として扱われる | 再現した(bash・zsh・sh)。引用符のない区切りでは、行末のバックスラッシュと改行をつないでから区切りと比べる。つないだ本文にはデータ区間を与えない | 同上 |
