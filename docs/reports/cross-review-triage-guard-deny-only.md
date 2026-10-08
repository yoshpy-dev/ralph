# Cross-review triage report: guard-deny-only

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md
- Base branch: main (51855166)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 5
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-07-guard-deny-only.md(AC7: 旧版が deny にする形は新版でも deny。例外は見張りの一致がすべてデータ区間に収まる場合だけ)
- Self-review report: docs/reports/self-review-2026-10-07-guard-deny-only.md(2 周目とそのやり直しは merge。LOW はコメントのずれで、c61ab2bf・4ffe74fe で直した)
- Verify report: docs/reports/verify-2026-10-07-guard-deny-only.md(2 周目とそのやり直しは pass)
- Implementation context summary:
  - 1 周目の指摘 3 件は、consult が足した 3 件と合わせて 46806dc9 で直した。グループ・複合コマンド・リダイレクトのある `exec` ではデータ区間を与えない、ヒアドキュメントの読み始めを文脈ごとに持つ、fd の複製を 0〜2 に限る、`printf -v` を読むだけのコマンドから外す、の 4 つ
  - 2 周目の /test が見つけた F2-1(ダブルクォートの中で `$` と `(` を行継続で分ける形)は 4e829e34 で直した。行継続のあるコマンドにはデータ区間を与えない
  - この回の review は HEAD e1dfb422 に対して、read-only の sandbox で動いた。テストスイートは回していない(本人の申告)
- 再現(2026-10-08、`scratchpad/xr3/c2p*.txt` の probe、jq あり・なしの両方)
  - 指摘 1〜3: 新版 none、旧版 deny
  - 指摘 4・5: 新版 deny、旧版 none
  - 指摘 1 の区切り `$'\x45'` は、bash でも zsh でも `E` になる。`E` の行のあとのコマンドが実行される(`sem6.sh`)
- 同じ型の穴が出るのは 3 回目になる。1 周目のレビュー、2 周目の /test(F2-1)、この回。どれも「データ区間と判定したテキストが、実際には実行されるか、実行される場所に書かれる」形で、見つかるたびに条件を 1 つずつ足してきた

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] ヒアドキュメントの区切りの `$'\x45'` を字句解析が `x45` と読む。bash は `E` と読むので、`E` の行のあとの `sudo ls` を guard はヒアドキュメントの本文(データ)として扱い、旧版の deny を消す | 再現した(bash・zsh)。字句解析は `$'…'` の中の `\n`・`\t`・`\r` しか戻さない(ヘッダーの Not covered に記載)。区切りの語に `$` があれば(`$'…'` と `$"…"`)データ区間を与えない、とすれば閉じる。旧版より弱くなる形なので AC7 に反する | `.claude/hooks/pre_bash_guard.sh`(lex_redir、ANSI-C の戻し)、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |
| 2 | [P1] `env -S` の分割した文字列と、その後ろの引数を別々に読む。`env -S 'sh -c "eval \$2" --' echo 'sudo ls'` は 2 番目の引数を実行するが、guard は後ろを独立した `echo` と見てデータ区間を与える | 再現した。前置きのコマンド(`env` など)を読み飛ばしたあとのコマンドが読むだけのコマンドでも、前置きがあればデータ区間を与えない(トップレベルの単純コマンドの元の 1 語目で判定する)とすれば、`env -S` も含めて型ごと閉じる。旧版より弱くなる形 | 同上(cmd_pos、stage_note) |
| 3 | [P1] zsh の `builtin exec >run.sh` は shell の標準出力を付け替えるが、`cmd_pos` は `builtin` で止まるので `EXEC_SEEN` が立たない。`builtin exec >run.sh; echo 'sudo ls'; sh run.sh` が通る | 再現した。2 と同じ直し方(元の 1 語目が読むだけのコマンドでなければデータ区間を与えない)で閉じる。旧版より弱くなる形 | 同上(cmd_pos、end_cmd) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 4 | [P2] `git push origin -ofoo` を force push として止める。git は `-o` の値(`--push-option=foo`)と読む。別の語で渡した push option の値も、オプションとして調べてしまう | 再現した。旧版は通していた形で、新版の規則が増やした誤検知。push option を使う人は少ないが、止められた agent は言い直しを強いられる。値を取るオプション(`-o`、`--push-option`、`--repo`、`--receive-pack` など)の値を先に読み飛ばせば直る | `.claude/hooks/pre_bash_guard.sh`(git_rules の push)、template、テスト |
| 5 | [P2] `git reset HEAD -- --hard` を hard reset として止める。`--` のあとは pathspec で、`--hard` という名前のファイルを unstage するだけ | 再現した。旧版は通していた形で、新版の規則が増やした誤検知。`--hard` という名前のファイルはまれ。commit と `--no-verify` の走査と同じように `--` で止めれば直る | 同上(git_rules の reset) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

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
