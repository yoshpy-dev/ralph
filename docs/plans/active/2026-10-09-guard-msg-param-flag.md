# guard-msg-param-flag

- Status: Draft
- Approved: TBD
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
3. 穴(旧版は deny、この plan を書く途中で見つけた): zsh は配列の添字を評価する。引用符の外の `$arr['$(cmd)']`(連想配列の `$h['$(cmd)']` も)は単一引用符の中の置換を実行するが、字句解析は `$` と `arr[` をただの文字、`'$(cmd)'` を引用とみなすので、読むだけのコマンドの引数ならデータ区間になる。2026-10-09 に `zsh -f` 5.9 で、置換に無害な `echo … >&2` を入れて実行を確かめた。二重引用符の中の形(`"$arr['$(cmd)']"`)と `${arr['$(cmd)']}` は、いまの guard もすでに deny にする

## Scope

- `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh`(バイト単位で同じに保つ)
  - `lex_dollar`: 読んだ `$` が展開かどうかの印を立てる。`${…}` と、`$` に名前(英字・`_`)、数字、特別な記号(`@*#?$!-`)が続く形を「展開」とし、`$'…'` と `$"…"` を「ANSI-C か locale の引用」とする。単独の `$`(`$` のあとが引用符の終わりや空白など)はどちらでもない。名前のあとに `[` が続くとき(zsh の添字)は、対応する `]` まで読み(引用符とバックスラッシュを飛ばし、入れ子の `[` `]` を数える)、`$name[…]` の範囲を `xnote` でデータ区間から外す
  - `lex_word` と `lex_dq`: 語(と二重引用符の中)で、上の 2 つの印を集める。`add_word` が語ごとに持つ(`WEXP`、`WANSI` のような名前)
  - `msg_check`: 生の語の `${` の検査を、語の「展開」の印に置き換える。推奨の HEREDOC の形(`safe_heredoc_msg`)の例外と、検査の順序は今のまま
  - `stage_note` の rg: 生の語の `$'`・`$"` の検査を、語の「展開」か「ANSI-C か locale の引用」の印に置き換える。`--pre` の検査は今のまま
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
- zsh 5.9 で、`$arr['$(…)']`、`"$arr['$(…)']"`、`${arr['$(…)']}`、`$h['$(…)']` は置換を実行し、`x='$(…)'; echo $x` と `$arr[$'\x24(…)']` は実行しなかった

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (guard の 1 ファイルとその template の写し、テスト、tech-debt の記録だけの変更で、部品どうしの呼び出し・状態・データの形・境界を変えない)

## Design decisions

- **印は `lex_dollar` で立て、語ごとに持つ**。生の文字を後から調べ直すと、引用符の中かどうかを別にたどることになり、PR #211 の止めすぎがそこから出た。`lex_dollar` は引用符の外か二重引用符の中の `$` でしか呼ばれないので、ここで立てた印は「実際に展開される `$`」を表す
- **`$name[…]` は範囲ごとデータから外す**。`${…}` と同じ扱いにそろえる。bash では `$arr` のあとの `[…]` はただの文字だが、外しても止めすぎになるのは添字の中に見張りの語がある形だけで、旧版も deny にする
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
  - `rg $x sudo pat .`、`rg "$x" 'sudo ' .`、`rg $'\x2d-pre' sh 'sudo ls'`
  - `echo $arr['$(sudo ls)']`、`echo $h['$(sudo ls)']`
  - `git commit -m ${(e):-'$(sudo ls)'}`、`git commit --message''=${(e):-'$(sudo ls)'}`、`git commit -"m"${(e):-'$(sudo ls)'}`、`git commit -m $arr['$(sudo ls)']`
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
- Edge cases: 単独の `$`(`"foo$"`、`'foo$'`)、`$1` などの数字、`$@` などの記号、入れ子の添字(`$a[$b[1]]`)、閉じない添字(`$a[`)、二重引用符の中の `$arr[…]`(いまも deny)
- Evidence to capture: mutation(印を立てない、`msg_check` で印を見ない、rg で印を見ない、`$name[…]` の `xnote` を外す)でテストが赤になること

## Risks and mitigations

- `$name[…]` を読む処理を新しく書くので、閉じない `[` や長い添字で読みすぎるおそれがある。行の終わりか語の終わりで止め、テストに閉じない形を入れる。上限を超えた入力は既存の deny(`too_deep` など)の仕組みに乗せる
- 印の置き換えで、PR #211 が止めた形(`${(e)…}` のメッセージ、引用符を挟んだ形)が通るようになると後退になる。AC2 にその形を入れ、B 節で固定する
- `rg $x` を止めると、変数を使う無害な rg(`rg "$pattern" 'sudo ' src/`)も見張りの語があれば deny になる。旧版も deny にする

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
