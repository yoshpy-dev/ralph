# guard-test-gaps

- Status: Approved
- Approved: 2026-10-10 sha256:94d656c12780
- Owner: Claude Code
- Date: 2026-10-10
- Related request: ユーザーの依頼「テストの穴が 2 つを修正して」(2026-10-10)。PR #215(49ac046c)で `docs/tech-debt/README.md` のテストの穴の行に (h)(i) として残した 2 つ
- Related issue: N/A
- Type: test
- Branch: test/guard-test-gaps

## Objective

PR #215 で残した guard のテストの穴を 2 つ塞ぐ。判定は 1 つも変えない。

1. (h) DATACMD の不変条件の検査は、`cmd_pos` が飛ばす包みの名前を、`pre_bash_guard_commands.awk` の `function cmd_pos` の中の `nm == "…"` から読んでいる。包みの分岐を別の関数に移すと、その名前は検査から黙って外れる。あわせて、包みの分岐を消したり一覧とずらしたりしても気づけない: 包みを飛ばさないと後ろの `sh -c` の中を読まなくなり、`timeout 5 sh -c 'git commit -n -m x'` のような形が通るが、今のテストの行の判定は 1 つも変わらない(Codex の plan advisory が確かめた)
2. (i) `trailing_backslashes`(`pre_bash_guard_lex.awk`)の数え始めの下限を変える mutation を、テストが殺せない。ヒアドキュメントの本文の、バックスラッシュだけでできた行の扱いを固定する行がない(base の `read_body` も同じで、PR #215 の test が見つけた)

## Scope

- (h): 包みの名前を guard の BEGIN の一覧にし、`cmd_pos` とテストが同じ一覧を使う
  - `pre_bash_guard_rules.awk` の BEGIN: `WRAPPER`(`env command exec nohup time nice stdbuf timeout xargs`)の一覧を足す。DATACMD の一覧の上のコメントで包みの名前を並べている文を、この一覧を指す形に直す
  - `pre_bash_guard_commands.awk` の `cmd_pos`: 包みの分岐の前に、`nm` が `WRAPPER` になければその位置を返す判定を足す。包みごとの飛ばし方の分岐(`skip_env` など)、先頭の予約語・`function`・代入を飛ばす処理は今のまま
  - `tests/test-pre-bash-guard.sh` の不変条件の検査: 包みの名前を、ソースの `nm == "…"` ではなく、実行時の `WRAPPER` から読む(DATACMD と同じく、3 つの `.awk` の後ろに `BEGIN` を足して出力させる)。空なら FAIL。DATACMD が `WRAPPER` とテストの予約語の一覧のどちらとも重ならないことを確かめる。包みの名前を並べている F 節のコメントも直す
  - `tests/test-pre-bash-guard.sh` の D 節 `edge_deny`: 包みごとに、後ろの `sh -c` の中の `git commit -n` を止める 9 行を足す(`env sh -c 'git commit -n -m x'`、`command …`、`exec …`、`nohup …`、`time …`、`nice -n 5 …`、`stdbuf -o0 …`、`timeout 5 …`、`xargs …`)。49ac046c の guard で 9 行とも deny(jq あり・なし)。旧版は `--no-verify` の規則がないので none で、B 節には置けない
- (i): `tests/test-pre-bash-guard.sh` の D 節に 2 行を足す(test の報告の提案)
  - `edge_deny`: `cat <<EOF`、本文がバックスラッシュ 2 個だけの行、`EOF`、`git commit -n -m x`、`EOF`(2 個は継がないので `EOF` で本文が終わり、`git commit -n` が命令になる)
  - `edge_none`: 同じ形でバックスラッシュ 3 個(3 個目で次の行と継ぐので `EOF` で終わらず、`git commit -n -m x` は本文になる)
- 4 つの guard のファイルは、root と `templates/base/` の写しをバイト単位で同じに保つ
- `docs/tech-debt/README.md`: テストの穴の行の (h)(i) を解消済みにする

## Non-goals

- 判定の変更。テストの今の期待値は 1 つも変えない
- `cmd_pos` の包みごとの飛ばし方(オプションの読み方)を変えること
- `cmd_pos` が先頭で飛ばす予約語(`if then else elif do while until ! { }`)を一覧にすること。テストの予約語の一覧がこの 10 語をすべて含むので、検査の網は広がらない(consult の指摘)
- `$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'` の形(tech-debt の Debt (b))
- tech-debt のテストの穴の行のほかの項目

## Assumptions

- `cmd_pos`(49ac046c)は、先頭の予約語・`function`・代入を飛ばしたあと、`cname` の値 `nm` が 9 つの包みのどれかなら、それぞれの関数で飛ばし、どれでもなければその位置を返す(`else return i`)。包みの分岐の前に「`WRAPPER` になければ返す」を置いても、9 つの名前の分岐は同じに動き、それ以外の名前は今と同じに返る
- DATACMD を実行時に読む今の検査(`BEGIN { for (k in DATACMD) print k; exit }` を 3 つの `.awk` の後ろに足す)は、BEGIN の順に頼っている。`WRAPPER` も同じ BEGIN で作るので、同じ読み方が使える
- 包みの 9 行は、包みを飛ばしたあとの `sh -c` の中身を読み直して `no_verify` で止まる。包みを飛ばさないと、コマンドの位置は包みの名前のままで `sh -c` を読まず、見張りの 4 規則にも `--no-verify` はないので通る
- (i) の 2 行は、`text(a, b)` の右端が排他なので、2 個の行は今 2 個と数えて継がず(deny)、下限を `q > a` に変えると 1 個と数えて継ぐ(none)。3 個の行は今 3 個で継ぎ(none)、変えると 2 個で継がない(deny)。`no_verify` は `commit_rules` の命令の規則で見張りの語ではないので、本文のどこかにバックスラッシュと改行があってデータ区間がなくなっても、判定は本文か命令かで決まる(consult が確かめた)

## Affected areas

- `.claude/hooks/pre_bash_guard_commands.awk`、`.claude/hooks/pre_bash_guard_rules.awk`、それぞれの `templates/base/.claude/hooks/` の写し
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (guard の中の包みの判定を一覧に置き換え、テストの読み方と行を足すだけで、部品どうしの呼び出し・データの形・境界を変えない)

## Design decisions

- **包みを guard の一覧にし、テストもそれを読む**。テストがソースの書き方を読む今の形は、分岐の場所や書き方を変えると黙って外れる。一覧を `cmd_pos` の入口の判定に使えば、`cmd_pos` が飛ばす名前は一覧にある名前だけになるので、テストが一覧を読めば漏れがない
- **一覧と分岐のずれは、包みごとの行で捕まえる**。一覧から名前を消すか分岐を消すと、その包みは飛ばされず、後ろの `sh -c` の中の `git commit -n` を見逃す。9 行がそれぞれの包みでこれを止めることを確かめ、一覧の要素と分岐を 1 つずつ消す mutation がどれもテストを赤にすることを AC にする(Codex の plan advisory の MEDIUM)
- **振る舞いだけで不変条件を確かめる案はとらない**。DATACMD の各名前 D について `D sudo ls` が none になることを確かめる案は、`timeout` のように次の語も飛ばす包みや、オプションを読む包みで見落としが出る
- **(i) は test の報告が示した 2 行にする**。本文に見張りの語を置かず、継ぎの数え方だけで `git commit -n`(no_verify)が命令か本文かが決まる形
- Critical forks: None(どれも判定を変えない。一覧にするかどうかは、検査の漏れをなくす方法として決めた)

## Acceptance criteria

- [x] AC1: `pre_bash_guard_rules.awk` の BEGIN に `WRAPPER` があり、`cmd_pos` は包みの分岐の前に、`WRAPPER` にない名前の位置を返す
- [x] AC2: 不変条件の検査が、実行時の DATACMD と `WRAPPER` を読み、ソースの `nm == "…"` を読まない。`WRAPPER` が空なら FAIL。写しで `exec` か `if` を DATACMD に足すと FAIL。写しで包みの分岐を `cmd_pos` の外の関数に移しても、その名前は検査に残る
- [x] AC3: D 節の包みの 9 行が jq あり・なしで deny。写しで `WRAPPER` の 9 つの要素を 1 つずつ消すと(9 通り)、どれもテストが赤になる。写しで `cmd_pos` の 9 つの分岐を 1 つずつ消しても(9 通り)、どれもテストが赤になる
- [x] AC4: D 節の (i) の 2 行が、jq あり・なしで期待どおり(2 個の行は deny、3 個の行は none)。`trailing_backslashes` の下限を `q > a` に変えた写しで、2 行とも赤になる
- [x] AC5: 判定が変わらない。`bash tests/test-pre-bash-guard.sh` が、今の 2,068 件と足した行を含めて全部通る(jq あり・なし)。base(49ac046c)の guard とこの PR の guard に、テストの配列の行と PR #215 の test の比較の入力を渡し、判定の違いが 0 件
- [x] AC6: root と `templates/base/` の写しがバイト単位で同じ(`./scripts/check-sync.sh`)。`shellcheck -S warning`、`sh -n`、`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`(guard の awk の構文の段を含む)が通る。4 つの guard のファイルは 800 行未満
- [x] AC7: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` のテストの穴の行の (h)(i) が解消済みになる

## Implementation outline

1. S1(implementer): `WRAPPER` の一覧、`cmd_pos` の判定、コメント、不変条件の検査の読み方、D 節の 11 行、template の写し、mutation の確かめ。AC1〜AC6
2. S2(inline、docs だけ): tech-debt の (h)(i)。AC7

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`、`./scripts/check-sync.sh`、`./scripts/check-template.sh`
- Spec compliance criteria to confirm: AC1〜AC7
- Documentation drift to check: DATACMD の一覧の上のコメント、`cmd_pos` のコメント、テストの F 節の説明、tech-debt のテストの穴の行
- Evidence to capture: mutation の結果(AC2〜AC4)、判定の比較の件数

## Test plan

- Unit tests: D 節の 11 行。不変条件の検査の読み方を変える
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: base(49ac046c)との判定の比較(PR #215 の test が使った入力の集まり)。mawk と gawk でもテストを回す
- Edge cases: 包みのあとに包みが続く形(`env nohup sh -c …`)、代入のあとの包み、`nice` と `timeout` の引数の形
- Evidence to capture: AC2〜AC4 の mutation で赤になる行と検査

## Risks and mitigations

- `cmd_pos` の入口の判定を変えて、包みでない名前の扱いがずれると、コマンドの位置が変わり判定が変わる。分岐の中身は変えず、「一覧にない名前を返す」を足すだけにする。base との判定の比較で確かめる
- 一覧と分岐がずれる(一覧から消す、分岐を消す、一覧に足して分岐を足さない)と、その包みは飛ばされず、後ろの `sh -c` の中を読まないので、そこに書いた禁止の形を見逃す(安全側ではない)。包みごとの 9 行がこれを止め、AC3 の mutation で 18 通りすべてが赤になることを確かめる
- (i) と包みの行は `git commit -n` を含むので、手元の Bash の行に書くと別の hook に止められる。テストのファイルに書き、Bash の行には書かない

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く(guard の 2 つの `.awk` の書き換え)
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-10: ユーザーが承認ゲートで Approve。承認の前に、consult(consult-plan-testgaps)の指摘(`CMDSKIP` は検査の網を広げないので外す、包みを散文で並べたコメント 2 か所を S1 で直す、mutation を S1 の確認に入れる)と、Codex の plan advisory の MEDIUM(一覧と分岐がずれると `sh -c` の中の `--no-verify` を見逃すが今のテストでは気づけない。「安全側」は誤り)を反映し、包みごとの 9 行と 18 通りの mutation を AC3 にした。9 行は 49ac046c の guard で deny(jq あり・なし)を確かめた。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [x] Implementation started
  - S1 完了(468fc73c、implementer/opus): `rules.awk` の BEGIN に `WRAPPER`、`cmd_pos` に「`WRAPPER` にない名前はその位置を返す」、不変条件の検査は実行時の DATACMD と `WRAPPER` を 1 つの dump で読む(ソースの `nm ==` は読まない)、D 節に包みの 9 行と (i) の 2 行。テストは 2,090/0(macOS、gawk、mawk)。AC2: `exec`・`if` を DATACMD に足すと FAIL、`timeout` の分岐を `cmd_pos` の外に移しても検査に残る(前の読み方では抜けた)。AC3: 18 通りすべて赤。AC4: 下限を `q > a` にすると 2 行とも赤。base との判定の比較は 1,938 回で違い 0。handoff の外で、テストのヘッダーの D 項目に 2 文を足した。plan の AC5 の「PR #215 の test の入力(3,913 件)」での比較は S1 では回していない(/test で回す)
  - self-review(8f237091、reviewer/opus): Merge yes、LOW 3 件。`WRAPPER` の判定は判定を変えない(base と HEAD に 60,480 形を流して違い 0)、18 組の写しでそれぞれ消した包みの行だけが none になる、ヒアドキュメントの 2 行のバックスラッシュの数はコメントどおり。LOW 1(BEGIN の一覧を説明するコメントに `WRAPPER` がない)と LOW 2(ヘッダーの包みの散文が `WRAPPER` を指していない)は S3 で直し、LOW 3(tech-debt の閉じ方)は sync-docs に回す。LOW 2 のうち `pre_bash_guard.sh` のヘッダーの item 6 の「a wrapper such as env, command, sh, nice, builtin or exec」の文は、データ区間の許可リストの 1 語目の話で、wrapper を「別のコマンドを動かすコマンド」という広い意味で使っている。base からあり、`sh` や `builtin` を `WRAPPER` に入れる話ではないので残す
  - S3(inline、コメントだけ): `pre_bash_guard.sh` のヘッダーの item 3 に `(WRAPPER in pre_bash_guard_rules.awk)` を足し、awk の呼び出しの上と `rules.awk` の先頭のコメントの一覧に `WRAPPER` を足した。template の写しは同じ。テストは 2,090/0、static は FAIL なし
  - S2 完了(73486db9、inline、docs だけ): tech-debt のテストの穴の行の (h)(i) を、Debt・Impact・Why deferred・Trigger の 4 列で解消済みにし、Related にこの plan を足した
  - 再 review(c5e7c90e、reviewer): Merge yes、新しい指摘なし。LOW 1・LOW 2 が 4f3a4414 で直ったことを確かめた(変わった行はコメントだけ。item 6 の文を残したことにも同意)。LOW 3(tech-debt の (i) の閉じ方と Trigger の列の決まりごと)は sync-docs に回した
  - verify(a1b57346、verifier): pass。AC1・AC2・AC4・AC6 は Met、AC3 は静的と 2 通りの抜き取り、AC5・AC7 は一部(テスト全件と test mode は /test)。指摘は LOW 3 件(V-1: guard の限界の行に `nm == "…"` の文が残る、V-2・V-3: self-review の LOW 3 と同じ)と情報 4 件
  - test(0104799d、tester): pass。`./scripts/run-test.sh` rc 0(shell 40 本 3,944 件、Go 8 packages)、guard のテストは 2,090/0(macOS の BWK awk、mawk、gawk のどれでも)。AC3 の 18 個と AC4 の 1 個の mutation(計 19 個)は 3 つの awk のすべてで赤。base と HEAD の判定の比較は、固有の payload 4,347 を jq あり・なしで 3 つの awk に渡し、8,694 組(17,388 回の実行)で違い 0。包みの端の入力 153 件を含み、mutant 3 個を同じ比較に通すと 50・50・60 組が変わる
  - sync-docs(この commit): tech-debt のテストの穴の行で、Debt の列の (i) を (h) と同じく `~~` で消して「Closed in …」を続け、Trigger の列の決まりごと(新しい包みは `WRAPPER` に入れて自分の行を持つ)を (h) の閉じの文へ移した(`cmd_pos` の分岐が要ること、名前だけで分岐がない場合は `cmd_pos` が名前の位置を返すこと、10 個目の包みのずれは自分の行がなければ捕まらないことも書いた)。(h)(i) の閉じに test の結果(mutation 19 個が 3 つの awk で赤、4,347 payload の比較で違い 0)を足し、(c) に 2090/0 の回を足し、Related に test の報告を足した。guard の限界の行は、不変条件の検査が `WRAPPER` を実行時に読むことに文を合わせた(V-1)。この Progress の self-review の行は、残した文の場所を直した。ほかの docs に drift はなく、`./scripts/check-sync.sh` は DRIFTED 0。直していない情報の指摘: `pre_bash_guard.sh` のヘッダー item 3 の段落の詰め直しが途中で止まっている(体裁だけ。guard のファイルは触らない取り決め)
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
