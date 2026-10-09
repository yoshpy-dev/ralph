# guard-msg-param-flag

- Status: Approved
- Approved: 2026-10-09 sha256:ebf5a9ac3a15
- Owner: Claude Code
- Date: 2026-10-09
- Related request: PR #211 の後続。ユーザーの依頼「残っているguard の後続の件を修正できますか?それと、前回の一覧の4~5の着手してください。」(2026-10-09)。この plan は guard の判定の 2 件(止めすぎと `rg $x`)を扱う。一覧の 5(guard のファイルの分割)は別の plan と PR で扱う
- Related issue: N/A
- Type: fix
- Branch: fix/guard-msg-param-flag

## Objective

`.claude/hooks/pre_bash_guard.sh` の判定を 3 点直す。どれも「引用符の外で実際に展開される `$` かどうか」を語ごとに見分ければ直る。

1. 止めすぎ(PR #211 が増やした): コミットとタグのメッセージの語に `${` を文字として書き(単一引用符の中、`\${`、`$'…'`)、見張りの語も含む形が deny になる(`git commit -m 'mention ${HOME}; never sudo ls'`。PR #211 の前は none、旧版は deny)。`msg_check` が語の生の文字に `${` があるかだけを見ているため
2. 穴(旧版は deny): `rg $x sudo pat .` や `rg "$x" 'sudo ' .` のように変数の語を持つ rg を、読むだけのコマンドとして扱う。変数の値が `--pre=…` ならプログラムを実行するが、データ区間になって見張りの一致が無視される。同じく、rg の判定は生の語に `$'` か `$"` があるかを見るので、単一引用符の中の正規表現の行末(`rg 'foo$' 'sudo ' .`)まで止める(止めすぎ、PR #210 の self-review の C4R-1)
3. 穴(旧版は deny、この plan を書く途中で見つけた): zsh は配列の添字を評価する。引用符の外の `$arr['$(cmd)']`(連想配列の `$h['$(cmd)']`、修飾つきの `$~arr['$(cmd)']`・`$=arr['$(cmd)']` も)は単一引用符の中の置換を実行するが、字句解析は `$` と `arr[` をただの文字、`'$(cmd)'` を引用とみなすので、読むだけのコマンドの引数ならデータ区間になる。古い算術の書き方 `$['$(cmd)']` も、bash と zsh の両方で置換を実行する。2026-10-09 に `zsh -f` 5.9 と bash 3.2 で、置換に無害な `echo … >&2` を入れて実行を確かめた。二重引用符の中の形(`"$arr['$(cmd)']"`)と `${arr['$(cmd)']}` は、いまの guard もすでに deny にする。添字は語の境目で終わる(zsh は `$a[1 + 1]` を invalid subscript とする)

## Scope

- `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh`(バイト単位で同じに保つ)
  - `lex_dollar`: 読んだ `$` が展開かどうかの印を立てる。`$'…'` と `$"…"`(二重引用符の外)を「ANSI-C か locale の引用」とする。それ以外で、`$` のあとが文字列の終わり・空白・(二重引用符の中で)閉じる `"` のどれでもない形を「展開」とする(`${…}`、`$(…)`、`$名前`、`$1`、`$@`、zsh の修飾つきの `$~x`・`$=x`・`$^x`・`$+x` などを含む。余分に立つのは止める側)。印は、入れ子(`lex_brace`、`lex_cmds`)から戻ったあとに立てる(`LD_SUBST` と同じ順)
  - 添字: `$` のあとに zsh の修飾の記号(`~`、`=`、`^`、`+`)が 0 個以上、名前が 0 文字以上続き、その次が `[` のとき(`$arr[`、`$~arr[`、算術の `$[`)、その `$` の位置を覚えておき、`lex_word` が語の終わりで、その位置から語の終わりまでを `xnote` でデータ区間から外す。`[…]` の中は読み飛ばさず、`lex_word` が今までどおり字句解析する。だから添字の中の `$(…)` やバッククォートは、今までどおり置換として読み直され、置換ありの印(`LW_SUBST`)も立つ(`git commit -m $arr[$(date)]` は deny のまま)。語の境目は `lex_word` の既存の規則を使うので、閉じない `[` や `;` の後ろまで読むことはない。二重引用符の中の `$name[` も同じく、`lex_dq` が位置を返し、`lex_word` がその位置から語の終わりまでを外す
  - `lex_word` と `lex_dq`: 語(と二重引用符の中)で、上の印を集める(`lex_dq` は二重引用符の中の展開の印を返す)。`add_word` が語ごとに持つ(`WEXP`、`WANSI` のような名前)。`add_word` の呼び出しは `lex_cmds` とリダイレクトの 2 か所
  - `msg_check`: 生の語の `${` の検査を、語の「展開」の印に置き換える。推奨の HEREDOC の形(`safe_heredoc_msg`)の例外と、検査の順序は今のまま
  - `stage_note` の rg: 生の語の `$'`・`$"` の検査を、語の「展開」か「ANSI-C か locale の引用」か「置換あり」(`WS`)の印に置き換える(`rg $(echo --pre) sh 'sudo ls'` も読むだけにしない)。`--pre` の検査は今のまま
  - ヘッダーのデータ区間の説明と、関数のコメント
- `tests/test-pre-bash-guard.sh`: 下の AC の形を足す。旧版も deny にする形は B 節の `guard_deny_only_forms` に入れ、AC7 の比較の例に含める。PR #211 が D 節の `edge_sentinel_deny` などに置いた、この plan で none になる形は、`edge_none` に移す
- `docs/tech-debt/README.md`: PR #211 が足した止めすぎの行の (a) と、guard の限界の行の rg の変数の語の穴を、解消済みにする

## Non-goals

- guard のファイルの分割(別の plan と PR で扱う)
- printf の検査(生の語の `%`・`$`・バッククォート)を印に置き換えること。printf は `%` を生の語で見る必要があり、今のままでも穴はない
- `$name[…]` 以外の仕組みで zsh や bash が評価する形。見つかれば tech-debt に記録する
- 変数の値そのもの(実行時にしか分からない)。`echo $x 'sudo ls'` のような、読むだけのコマンドの変数の引数は、rg を除いて今のままデータとして扱う(echo・cat・grep などは、変数の値がオプションでも実行しない)

## Assumptions

- 新版の guard は PR #211(c3a9242e)のもの。旧版は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`
- 2026-10-09 の probe(`scratchpad/mp/`、jq あり・なし): 単一引用符・`\${`・`$'…'` で `${` を書いたメッセージ 4 形は新版 deny、`git commit -m "${msg}"` と `git commit -m 'use ${HOME}'` は none、`rg $x sudo pat .` と `rg "$x" 'sudo ' .` は新版 none・旧版 deny、`rg 'foo$' 'sudo ' .` は deny、`x=--pre; rg $x sudo pat .` は deny(許可リストの外の代入)
- zsh 5.9 で、`$arr['$(…)']`、`"$arr['$(…)']"`、`${arr['$(…)']}`、`$h['$(…)']`、`$~arr['$(…)']`、`$=arr['$(…)']` は置換を実行し、`x='$(…)'; echo $x` と `$arr[$'\x24(…)']` は実行しなかった。`$['$(…)']` は zsh と bash 3.2 の両方で実行した。`$arr[$(…)]`(引用符なし)はどちらも実行するが、いまの guard は置換として読み直して deny にする

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (guard の 1 ファイルとその template の写し、テスト、tech-debt の記録だけの変更で、部品どうしの呼び出し・状態・データの形・境界を変えない)

## Design decisions

- **印は `lex_dollar` で立て、語ごとに持つ**。生の文字を後から調べ直すと、引用符の中かどうかを別にたどることになり、PR #211 の止めすぎがそこから出た。`lex_dollar` は引用符の外か二重引用符の中の `$` でしか呼ばれないので、ここで立てた印は「実際に展開される `$`」を表す
- **`$name[` から語の終わりまでをデータから外し、中は読み飛ばさない**。対応する `]` まで読む案は、語の境目を越えて読むと後ろのコマンドを飲み込み(`echo $a[ ; git push origin --force`)、添字の中の置換の読み直しも別に書くことになる(Codex の plan advisory と consult の指摘)。語の終わりまでを外す案は、`lex_word` の既存の境目と字句解析をそのまま使い、外れても止める方向にしかずれない。bash では `$arr` のあとの `[…]` はただの文字だが、外しても止めすぎになるのは同じ語の後ろに見張りの語がある形だけで、旧版も deny にする
- **「展開」の印は広めに立てる**。`$` のあとが終わり・空白・閉じる `"` のときだけ「展開でない」とする。zsh の修飾(`$~x` など)の取りこぼしを避けるため。余分に立っても、影響はメッセージと rg がデータ区間を失うこと(見張りが旧版どおりに決める)だけ
- **printf は変えない**。範囲を guard の判定 2 件と、その途中で見つけた `$name[…]` に絞る
- **直す範囲の線引き**: 直すのは上の 3 点だけ。self-review や cross-review が別の仕組みの形を見つけたら、tech-debt の guard の行に記録する。記録だけで済む指摘には、post-implementation-pipeline.md の「記録だけの修正」の例外を使う
- Critical forks: None

## Acceptance criteria

- [ ] AC1: 次がどのモードでも none になる(jq あり・なし)
  - `git commit -m 'mention ${HOME}; never sudo ls'`
  - `git commit -m "mention \${HOME}; never sudo ls"`
  - `git commit -m $'mention ${HOME}; never sudo ls'`
  - `git tag -a v1 -m 'mention ${HOME}; never sudo ls'`
  - `rg 'foo$' 'sudo ' .`
  - `rg -n 'sudo ' .`
  - `git commit -m "${msg}"`、`git commit -m 'use ${HOME}'`
  - `echo "${HOME}" 'sudo ls'`、推奨の HEREDOC の形で本文に `${HOME}` と `sudo ls` を書いたコミット
- [ ] AC2: 次がどのモードでも deny になる(jq あり・なし)。どれも旧版も deny にする
  - `rg $x sudo pat .`、`rg "$x" 'sudo ' .`、`rg $'\x2d-pre' sh 'sudo ls'`、`rg $(echo --pre) sh 'sudo ls'`
  - `echo $arr['$(sudo ls)']`、`echo $h['$(sudo ls)']`、`echo $~arr['$(sudo ls)']`、`echo $['$(sudo ls)']`
  - `git commit -m ${(e):-'$(sudo ls)'}`、`git commit --message''=${(e):-'$(sudo ls)'}`、`git commit -"m"${(e):-'$(sudo ls)'}`、`git commit -m $arr['$(sudo ls)']`
- [ ] AC2b: 添字の中の置換は今までどおり字句解析が読み直す。`git commit -m $arr[$(date)]` と、置換の中でコマンド名を引用符で分けた `echo $arr[$(s''udo ls)]` が、どのモードでも deny になる(jq あり・なし。旧版は前者・後者とも通すので、D 節の `edge_deny` に置く)
- [ ] AC3: plan 2026-10-07-guard-deny-only の AC3 の 29 形(テストの C 節の `ac3`)は none のまま。G 節で、旧版 deny から新版 none になる形が `intentional_fixes` の 13 件のまま
- [ ] AC4: `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh` がバイト単位で同じ(`./scripts/check-sync.sh`)
- [ ] AC5: `bash tests/test-pre-bash-guard.sh` と `bash tests/test-lib-json.sh` が通り、`./scripts/run-verify.sh` が rc 0
- [ ] AC6: `docs/tech-debt/README.md` の、PR #211 の止めすぎの行の (a) と、guard の限界の行の rg の変数の語の記述が、解消済みになる

## Implementation outline

1. S1(1 slice、implementer): `lex_dollar` の印と `$name[…]` の `xnote`、`lex_word`・`lex_dq`・`add_word` で印を語ごとに持つ、`msg_check` と rg の検査を印に置き換える、コメント、template の写し、テスト(AC1〜AC3)
2. S2(inline、docs だけ): tech-debt の 2 か所(AC6)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`、`./scripts/check-sync.sh`、awk のプログラムに単一引用符がないこと
- Spec compliance criteria to confirm: AC1〜AC6。AC1 と AC2 は probe(`scratchpad/mp/`)でも新旧を比べる
- Documentation drift to check: ヘッダーのデータ区間の説明、tech-debt の 2 か所
- Evidence to capture: probe の新旧比較、テストの件数

## Test plan

- Unit tests: AC2 の形を B 節に、AC1 の形を D 節の `edge_none` に足す(PR #211 が deny として置いた形は移す)
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: 既存の 1,948 件、`tests/test-lib-json.sh` の 126 件
- Edge cases: 単独の `$`(`"foo$"`、`'foo$'`)、`$1` などの数字、`$@` などの記号、入れ子の添字(`$a[$b[1]]`)、閉じない添字(`echo $a[ ; git push origin --force` が deny のまま)、二重引用符の中の `$arr[…]`(いまも deny)
- Evidence to capture: mutation(印を立てない、`msg_check` で印を見ない、rg で印を見ない、rg で `WS` を見ない、`$name[` の `xnote` を外す)でテストが赤になること

## Risks and mitigations

- `$name[` の扱いで、添字の中の置換を読み落とすと、今は止まる形(`git commit -m $arr[$(date)]`)が通る後退になる。添字を読み飛ばさず `lex_word` の字句解析に任せ、AC2b で固定する(Codex の plan advisory の HIGH)
- 印の置き換えで、PR #211 が止めた形(`${(e)…}` のメッセージ、引用符を挟んだ形)が通るようになると後退になる。AC2 にその形を入れ、B 節で固定する
- `rg $x` を止めると、変数を使う無害な rg(`rg "$pattern" 'sudo ' src/`)も見張りの語があれば deny になる。旧版も deny にする

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-09: ユーザーが承認ゲートで Approve。Codex plan advisory の HIGH(添字を読み飛ばすと中の置換を読み落とす)と、consult(consult-plan-msgflag)の指摘(`]` まで読むと語の境目を越える、rg に `WS` を足す、`$[…]` を確かめる)を、承認の前に plan に反映した。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [x] Implementation started
  - S1 完了(1e032dea、implementer/opus): `lex_dollar` が「展開」(`LD_EXP`)、「ANSI-C か locale の引用」(`LD_ANSI`)、添字の始まりの位置(`LD_SUB`)を立て、`lex_word`・`lex_dq`・`add_word` が語ごとに持つ(`WEXP`、`WANSI`)。`msg_check` と rg の判定を印に置き換え、添字は `$` から語の終わりまでを `xnote` した。テスト 2016 件。既存のテストで判定が変わったものはない。印の名前の文字に数字も含めた(zsh は `$0[1]` を添字とする)。rg の `WS` を固定するため、バッククォートの形 `` rg `echo --pre` sh 'sudo ls' `` を B 節に足した
  - S1b 完了(43e73568、implementer/opus): S1 の implementer が見つけた、特別なパラメータの添字(`$@[…]`、`$*[…]`、`$#x[…]`。zsh で置換の実行を確かめた)を添字の判定に足した。plan の Objective 3(zsh の添字の評価)の範囲で、Scope の書き方(名前 0 文字以上)を広げたもの(止める側)。二重引用符の中で開く添字(`"$arr["'$(…)'"]"`)をテストで固定した。テスト 2032 件
  - S2 完了(inline、docs だけ): tech-debt の 3 か所を直した。guard の限界の行の rg の変数の語、テストの穴の行の rg、PR #211 の止めすぎの行の (a) を解消済みにし、この PR で増える止めすぎ(`$` の展開のあるメッセージと rg、添字のあとの同じ語)を新しい行に書いた
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
