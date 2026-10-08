# Walkthrough: guard-deny-only

- Date: 2026-10-08
- Plan: docs/plans/archive/2026-10-07-guard-deny-only.md(この PR の最後のコミットで archive に移す)
- Branch: fix/guard-deny-only(base main 51855166。途中で main を 2 回取り込んだ)
- Diff(HEAD 544fa098、この walkthrough を除く): 24 files、+7,308 / -434。報告・insight・plan・evidence・`templates/` の写しを除くと 11 files、+3,319 / -305。そのうちテストが +1,751

## 何を変えたか

`.claude/hooks/pre_bash_guard.sh` が確認(ask)を返さないようにした。止めるべきコマンドだけを deny にする。

- ask の規則(`.git`・`.env` への書き込み、`rm -rf `、`gh pr create`)は、モードによらずすべてなくした。guard は `permission_mode` を読まない
- deny の判定は 2 つの方法を重ね、どちらかが deny なら deny にする
  - 字句解析: awk で書いた shell に近い字句解析が、語と単純コマンドを切り出す。`$(...)`・バッククォート・`sh -c`・`eval`・shell に流すヒアドキュメントやパイプの中身は、コマンドとして読み直す。コマンドの位置にある `sudo`、force push、hard reset、コミットメッセージの置換、`--no-verify`、`core.hooksPath` を止める
  - 見張り: 旧版の 4 つの文字列の規則(sudo、force push、hard reset、コミットメッセージの置換)を、生のコマンド全体に当てる。一致がすべて「データ区間」の中にあるときだけ無視する。データ区間は、読むだけのコマンドの引数、`git commit`・`git tag` のメッセージ、データとして読むヒアドキュメントの本文、コメントの 4 つ
- データ区間を与えるのは、テキストの行き先を guard が見分けられるときだけにした。トップレベルの単純コマンドの 1 語目がすべて読むだけのコマンドか `git` で、値に `/` がないときに限る。グループ・複合コマンド、リダイレクトのある `exec`、行継続、区切りに `$` かバッククォートがあるヒアドキュメントでも与えない。与えないときは、見張りが旧版どおりに決める
- 読むだけのコマンドの一覧から、引数を評価しうるものを外した。`test` と `[` は入れない(zsh と bash 5 の `-v` は添字の中の置換を実行する)。printf は、どの語にも書かれたままの形で `%`・`$`・バッククォートがなく、`-v` もないときだけ読むだけにする(zsh の `%n` と数値の変換は引数を評価する)。rg は `--pre` も、`$'…'`・`$"…"` の語もないときだけ読むだけにする
- 判定しきれない入力の扱い: 読み直しの深さ・キューの大きさ・入れ子の上限を超えた入力は deny にする。awk がない(または失敗した)ときは、旧版の 4 規則で決める
- `.claude/hooks/lib_json.sh` は、jq がなくても JSON の文字列のエスケープ(`\n`・`\t`・`\"`・`\\`・`\/`・ASCII の `\uXXXX`)を戻す。jq も awk もないときは、前と同じ sed の取り出しに戻る

結果として、旧版が通していた force push や `--no-verify` の書き方の多くを止める。旧版が止めていた誤検知のうち 13 形(コミットメッセージや `grep` の引数に書いた `sudo `・`git push --force`、`visudo -c`、推奨の HEREDOC のコミット形など)を通す。

## 読む順

1. plan の Design decisions と Progress checklist(どう決め、どう直してきたか)
2. guard のヘッダー(`.claude/hooks/pre_bash_guard.sh` の 1〜138 行目)。2 つの判定、データ区間の定義、データ区間を与えない場合、見ないもの(Not covered)
3. awk のプログラム(159〜1418 行目)。節の見出しの順に読む
   - Text access(`set_text`・`at`・`find`): 512 文字の窓で読む。macOS の BWK awk は `substr()` のたびに文字列全体をなめるため
   - Lexing(`lex_cmds`・`lex_word`・`lex_dq`・`lex_dollar`・`lex_bq`・`lex_redir`・`read_heredocs`・`read_body`): 文脈(`new_ctx`)ごとに語を切り出す。ヒアドキュメントは宣言した文脈の改行で読み始める
   - Simple-command assembly(`end_cmd`・`data_first_ok`・`cmd_pos`・`skip_*`): コマンドの位置を決め、データ区間を与えてよいかを決める
   - Data regions(`add_data`・`in_data`・`redir_safe`・`stage_note`・`pipe_decide`): パイプの後段とリダイレクトの行き先を見て、読むだけのコマンドの引数をデータ区間にする
   - 規則(`judge`・`scan_words`・`shell_rules`・`git_rules`・`push_rules`・`reset_rules`・`no_verify_rules`・`commit_rules`・`tag_rules`)
   - 見張り(`sentinel`・`sentinel_str`)と END(読み直しのキューを回し、最後に見張りを当てる)
4. awk が失敗したときの分岐(1419 行目以降)と、deny の理由の文
5. `.claude/hooks/lib_json.sh` の awk の戻し
6. テスト(`tests/test-pre-bash-guard.sh` の A〜I 節、`tests/test-lib-json.sh`)。G 節は `tests/fixtures/guard-1c4cea5a/` の旧版と新版に同じ例を渡し、旧版 deny から新版 none になる形が `intentional_fixes` の 13 件だけであることを確かめる

## 実装と直しの経緯

- S1〜S3(`181e1065`・`e48797a2`・`5b0b20d5`・`13b36abd`・`c133bde5`): `lib_json.sh` の戻し、字句解析の guard、前置きの一覧に頼らない補いと、上限を超えた入力や awk の失敗を deny にする処理、tech-debt と文書
- 1 回目の self-review(`511f6382`)は no-merge。例の外の 49 形のうち 44 形を旧版より弱く通した。consult と相談し、旧版の一致を見張りに残してデータ区間だけを通す作り(S2c、`df0a50d5`)に変えた。ユーザーが plan を承認し直した(digest 7efd47f36781)
- cross-review の 1 周目(`f29d4132`): グループ・複合コマンド、`$(...)` の中の改行、ヒアドキュメントの行継続で、データ区間の判定が外れる 3 件。consult が同じ型をあと 3 つ挙げた。`46806dc9` で直した
- 2 周目の /test(`6cca4ce4`)は fail。ダブルクォートの中で `$` と `(` を行継続で分ける形。`4e829e34` で直し、2 周目の中で self-review・verify・test をやり直した
- cross-review の 2 周目(`a9af4e05`): 同じ型の 3 件(区切りの `$'\x45'`、`env -S`、zsh の `builtin exec`)と、新しい誤検知 2 件(`git push -ofoo`、`git reset -- --hard`)。上限に達し、ユーザーが上限を 3 に上げた。条件を足すのをやめ、許可リストにした(`a9ef82b1`)。3 回目の self-review を受けて `849f5411` で `git reset --pathspec-from-file -- --hard` と理由の文を直した
- cross-review の 3 周目(`af7d5987`): zsh の `printf '%n'` と、同じ型の `test -v`・`[ -v`(引数の添字の中の置換を実行する)、`--no-verify` の規則の抜け道(`-m` の値に `--` を渡す形、`--trailer` の略記)。ユーザーが上限を 4 に上げた。`b3c3fdaa` で直し、4 回目の self-review が見つけた `$'\x25n'` で書いた書式と git の 1 文字の略記(`git reset --h`)を `12e9a9ad` で直した
- テストの追加は tester が mutation の結果から足した(`39ed2759`・`3e9afad6`・`180c7389`・`8e94e76b`・`389436de`)。最後の mutation は 20 個すべてが赤

## 計画からの逸脱

- S2c の作り(見張りとデータ区間)は、1 回目の self-review のあとに plan を書き直し、ユーザーが承認し直した
- 止める側への逸脱(plan の本文は変えず、Progress checklist に記録した): fd の複製で安全とみなすのを 0〜2 に限った。グループ・複合コマンド・`exec`・行継続・前置きつきの読むだけのコマンド・道のある名前ではデータ区間を与えない。`test`・`[` を読むだけのコマンドから外し、printf と rg に条件を足した。どれも旧版と同じ deny になる。plan の Scope の 38・41・45 行目は変更前の規則のまま残っている(承認 digest の範囲なので本文は変えていない)
- 推奨の HEREDOC のコミット形は、同じ Bash の呼び出しに読むだけのコマンドでも git でもないコマンド(`make test &&` など)があると止まる。旧版も止めていた。許可リストからこの形だけを外すと、同じ呼び出しの関数定義が置換の中身を変えられるので外さず、理由の文、`git-commit-strategy.md`、`internal/org/prompts/implementer.md` に「単独のコマンドで打つ」と書いた

## 変えていないもの

- ECC plugin の block-no-verify(`git commit` と同じコマンドに `-n` の付くフラグがあると止める)。plugin の hook を 1 つだけ外す公式の手段がない
- Claude Code 本体の確認(重要なパスへの `rm` など)。hook でも設定でも止められない
- guard が見ないもの: 変数、alias、関数、スクリプトのファイル、`ssh` の先、前の呼び出しで仕込んだ shell の状態、`$'…'` の 16 進・8 進のエスケープ、ブレース展開、パス名展開。旧版も見ない

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
| --- | --- | --- |
| core | `.claude/hooks/pre_bash_guard.sh`、`.claude/hooks/lib_json.sh`、`.claude/hooks/post_edit_verify.sh`(コメントだけ) | 置き換え |
| core | `.claude/rules/ralph/git-commit-strategy.md`、`.codex/README.md` | 置き換え |

`templates/base/` の写しは root とバイト単位で同じ(`check-sync.sh`)。

## 残る穴

`docs/tech-debt/README.md` の guard の限界の行とテストの穴の行に書いた。主なものは次のとおり。

- 字句解析だけが止められる形の一部(`git commit -m "$\` 改行 `(id)"` など)は、新版も旧版も通す
- 実行時にしか分からないもの: 変数の語で `--pre` を渡す `rg $x …`(新版は通し、旧版は止める。同じコマンドで変数を設定すれば止まる)、前の呼び出しで仕込んだ shell の状態、起動ファイルの shell のオプション(zsh の `cdablevars`)
- 止めすぎ: 前置きつきの読むだけのコマンド、道のある名前、行継続のあるコマンド、`&>/dev/null`、つないだ呼び出しの HEREDOC のコミット形、`%` のある printf、`test`・`[`。どれも旧版も deny。新しい `--no-verify` の規則は `git merge -m --no-verify`(メッセージがその文字)も止める
- guard のファイルは 1,461 行で、ファイルの大きさの目安(800 行)を超える
- cross-review の 4 周目の結果は、PR 本文と `docs/reports/cross-review-triage-guard-deny-only.md` に書く
