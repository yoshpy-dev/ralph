# guard-debt-cleanup

- Status: Approved
- Approved: 2026-10-09 sha256:f37da93972d7
- Owner: Claude Code
- Date: 2026-10-09
- Related request: ユーザーの依頼「2,3,4を進めて」(2026-10-09)。直前の「残タスクは?」への返答で挙げた guard の系列の 2(tech-debt (e) のコードの 2 項目)、3(awk のコメントの `SQ` の綴り)、4(`.awk` の検査の抜け)
- Related issue: N/A
- Type: refactor
- Branch: refactor/guard-debt-cleanup

## Objective

PR #214(a0094fe5)のあとに `docs/tech-debt/README.md` に残った、guard まわりの 3 つを片づける。判定は 1 つも変えない。

1. `.awk` の構文を確かめる検査がない: `scripts/verify.local.sh` の shellcheck と `sh -n` は `.claude/hooks/*.sh` しか見ないので、3 つの `.awk` の構文エラーを捕まえるのは `tests/test-pre-bash-guard.sh` の F 節だけになっている。static の検査に、guard と同じ順で 3 つを読む段を足す。あわせて tech-debt の「`.awk` を言語に分類できず static verify が全範囲で回る」の行を見直す。`scripts/detect-changed-languages.sh` には shell の言語がなく、`.sh` の変更も同じく全範囲で回る作りなので、`.awk` だけの問題ではない
2. tech-debt (e) のコードの 2 項目
   - `read_body`(`pre_bash_guard_lex.awk`)が、行末のバックスラッシュを数えるループを 2 回書いている。1 つの関数にまとめる
   - 判定を変えない 2 つの規則を消す。`end_cmd`(`pre_bash_guard_commands.awk`)の、1 語目が予約語(`RESW`)なら `NODATA` にする規則と、`exec` のあとにリダイレクトがあれば `NODATA` にする規則(`cmd_pos` が立てる `EXEC_SEEN`)。どちらも、同じ `end_cmd` の許可リスト(`data_first_ok`)が同じ場面で必ず `NODATA` にする
3. awk のコメントの `SQ` の綴り: PR #214 の前は awk のプログラムを単一引用符で渡していたので、コメントでも単一引用符を `SQ` と書いていた。今はその制約がないので、本物の引用符に戻す(コードの変数 `SQ` はそのまま)

## Scope

- S1(Objective 1): `scripts/verify.local.sh` の `run_static_checks` に、`.claude/hooks/` と `templates/base/.claude/hooks/` のそれぞれで `LC_ALL=C awk -f pre_bash_guard_lex.awk -f pre_bash_guard_commands.awk -f pre_bash_guard_rules.awk </dev/null` を回し、exit 0 で出力がないことを確かめる段を足す(`sh -n` の段の隣)。`verify.local.sh` は repo 専用で、scaffold には配らない
- S2(Objective 3): 3 つの `.awk` のコメントで、単一引用符を表す `SQ` と、それと対で書いた `DQ` を本物の引用符に戻す(約 15 行)。コメントの行だけを変える
- S3(Objective 2)
  - `pre_bash_guard_lex.awk`: 行末のバックスラッシュの数を返す関数を 1 つ作り、`read_body` の 2 つのループをその呼び出しにする
  - `pre_bash_guard_commands.awk`: `end_cmd` の予約語の規則と `exec` の規則を消す。`cmd_pos` は `exec` を飛ばす処理を残し、`EXEC_SEEN` を立てる処理を消す
  - `pre_bash_guard_rules.awk`: `RESW` の一覧(`resw_list`)と `EXEC_SEEN` の初期化を消す
  - `tests/test-pre-bash-guard.sh`: 2 つを足す。1 つは不変条件の検査で、`pre_bash_guard_rules.awk` の `datacmd_list` の文字列と、`pre_bash_guard_commands.awk` の `cmd_pos` が飛ばす包みの名前(`nm == "…"` の名前)をコードから読み、DATACMD に包みも予約語(テストに書く一覧: `if then elif else fi for while until do done case esac select function { } ! [[ ]] time coproc`)も入っていないことを確かめる。もう 1 つは B 節の行 `exec >run.sh; echo 'sudo ls'`(後ろに `sh` を置かない形。旧版も deny)
  - コメント: `end_cmd`、`cmd_pos`、`in_data`、`data_first_ok` の上と、`pre_bash_guard.sh` のヘッダー(item 2 の予約語の説明、item 6 の `NODATA` の理由の一覧、awk の呼び出しの上の `RESW`)、`rules.awk` の先頭のコメント。複合コマンドと `exec` のリダイレクトは、1 語目が DATACMD でも `git` でもないので許可リストがデータ区間を落とす、と書く。DATACMD の一覧の上に、予約語と `cmd_pos` が飛ばす包み(env、command、exec など)をこの一覧に入れないこと、入れると許可リストが落とさなくなることを書く
- 4 つの guard のファイルは、root と `templates/base/` の写しをバイト単位で同じに保つ
- S4(docs だけ): `docs/tech-debt/README.md` の guard の限界の行の (e) の残り(コードの 2 項目と `SQ`)を解消済みにし、`.awk` の分類の行を見直す(分類は作りどおりとして閉じ、構文の確認は S1 で解消)

## Non-goals

- 判定の変更。テストの期待値は 1 つも変えない
- `scripts/detect-changed-languages.sh` に shell の言語を足すこと(`.sh` と `.awk` の変更は今のまま全範囲で回る。遅くなるだけで、検査は弱くならない)
- `$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'` の形(tech-debt の Debt (b)。ユーザーが今回の対象から外した 1)
- テストの穴(F 節が自動で見るのは `rules.awk` がない写しだけ、判定の比較は macOS の awk だけ)

## Assumptions

- DATACMD は `echo printf cat head tail wc cut tr grep egrep fgrep zgrep rg ls diff cd true false which type`、RESW は `if then elif else fi for while until do done case esac select function { }` で、重ならない。`git` は RESW にない(`pre_bash_guard_rules.awk` の BEGIN、a0094fe5)
- 予約語の規則は語の生の文字 `WR[ctx, 1]` を見る。予約語は引用符を含まないので、そのとき引用符を外した値 `WV[ctx, 1]` も同じ予約語で、`data_first_ok` は 0 を返し、許可リストが `NODATA` にする
- `cmd_pos` が `exec` を飛ばして `EXEC_SEEN` を立てるのは、それより前の語を飛ばしたあと(予約語、`!`、`{`、代入、包み)か 1 語目が `exec` のときだけ。どちらも 1 語目は DATACMD でも `git` でもないので、許可リストが `NODATA` にする。`cmd_pos` が飛ばす包み(env、command、exec、nohup、time、nice、stdbuf、timeout、xargs)は DATACMD にない
- `NODATA` を読むのは `in_data` だけで、`judge` は読まない。だから `end_cmd` の中で規則を消しても、`judge` の判定の順は変わらない
- tech-debt の (e) によると、guard-deny-only の cycle 3 の test で、この 2 つの規則を消す mutation(N03、N04)は 507 行の probe の判定を 1 つも変えなかった
- `scripts/verify.local.sh` は `scripts/run-verify.sh` から毎回呼ばれ、static mode では `run_static_checks` を回す。scaffold には配らない(check-sync の ROOT_ONLY_EXCLUSIONS)

## Affected areas

- `scripts/verify.local.sh`
- `.claude/hooks/pre_bash_guard.sh`、`pre_bash_guard_lex.awk`、`pre_bash_guard_commands.awk`、`pre_bash_guard_rules.awk`、それぞれの `templates/base/.claude/hooks/` の写し
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (検査の段を 1 つ足し、guard の中の判定を変えないコードとコメントを整えるだけで、部品どうしの呼び出し・データの形・境界を変えない)

## Design decisions

- **2 つの規則は残さずに消す**。残して「許可リストの後ろの予備」と書く案もある。消す理由は 3 つ。1 つ目に、Assumptions のとおり、どちらも許可リストが必ず先回りするので、判定を変える道がない。2 つ目に、判定を変えないコードは読む人に「この規則が効く場面がある」と思わせ、tech-debt (e) が指摘したコメントのずれを生んできた。3 つ目に、規則が守っていた前提(DATACMD に予約語も `cmd_pos` が飛ばす包みも入らない)を、テストの不変条件の検査が直接確かめる。今あるテストの行だけでは足りない: `exec` を DATACMD に 1 つ足すと `exec >run.sh; echo 'sudo ls'` が通るようになるが、今の `exec` の行は後ろの `sh run.sh` などがほかの理由で止めるので赤にならない(Codex の plan advisory の MEDIUM)。だから不変条件の検査と、後ろに `sh` を置かない `exec` の行を足す。DATACMD の一覧の上にも、入れてはいけない名前を書いておく(`(echo sudo ls)` は `(` の規則だけでも止まるので、許可リストの確かめには使えない)
- **`verify.local.sh` の段は guard の呼び出しと同じ形にする**。3 つを別々に読むと、ファイルをまたぐ関数の呼び出しを確かめられない。guard と同じ順と引数で、空の入力を渡す
- **言語の分類は足さない**。shell の言語を足すと、`run-static-verify.sh` に言語ごとの段を足すことになり、この PR の範囲を超える。`.sh` も全範囲で回る今の作りを、tech-debt の行に書いて閉じる
- **S1 を先に入れる**。同じ PR の S2・S3 で `.awk` を変えるとき、新しい段がその構文を確かめる
- Critical forks: None(規則を消すかどうかは、判定を変えないことがコードから言えるので、読みやすさで決めた。戻すのは 2 行と一覧を足し直すだけ)

## Acceptance criteria

- [ ] AC1: `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` が、guard の 3 つの `.awk` を root と template のそれぞれで読む段を回して OK になる。`git archive HEAD` で展開した木で 1 つの `.awk` に構文エラーを入れると、その段が FAIL になり、`verify.local.sh` は 0 以外で終わる
- [ ] AC2: 3 つの `.awk` のコメントの行に `SQ` も `DQ` も残らない(`grep -n '^[[:space:]]*#.*\(SQ\|DQ\)'` が何も出さない)。S2 の差分はコメントの行だけ
- [ ] AC3: `read_body` で行末のバックスラッシュを数える処理は 1 つの関数の呼び出しで、`while (q >= … && at(q) == BS)` の形のループは guard に 1 つだけ残る
- [ ] AC4: `RESW`、`resw_list`、`EXEC_SEEN` が guard の 4 つのファイル(コメントを含む)に残らない。`in_data` の上のコメントとヘッダーの `NODATA` の理由の一覧が、複合コマンドと `exec` のリダイレクトを許可リストの場合として書く。DATACMD の一覧の上に、予約語と包みを入れないことが書いてある
- [ ] AC5: 判定が変わらない。`bash tests/test-pre-bash-guard.sh` が、今の 2,063 件と足した行・検査を含めて全部通る(jq あり・なし)。base(a0094fe5)の guard とこの PR の guard に、テストの配列の行と PR #213・#214 の probe の行を jq あり・なしで渡し、判定の違いが 0 件
- [ ] AC5b: 消した規則の前提がテストで守られる。写しの `pre_bash_guard_rules.awk` の DATACMD に `exec` だけを足すと、足した `exec >run.sh; echo 'sudo ls'` の行と不変条件の検査が赤になる。`for` だけを足すと不変条件の検査が赤になる。`if grep -q 'sudo ' file; then echo ok; fi` の行は、許可リストの `NODATA` の行を消した写しで赤になる(これらの行を許可リストだけが止めていることの確認)
- [ ] AC6: root と `templates/base/` の 4 つの guard のファイルがバイト単位で同じ(`./scripts/check-sync.sh`)。`shellcheck -S warning` と `sh -n` が通り、4 つとも 800 行未満
- [ ] AC7: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` の guard の限界の行の (e) が、コードの 2 項目と `SQ` を含めて解消済みになり、`.awk` の分類の行は、分類は作りどおり、構文の確認は S1 で解消、として閉じる

## Implementation outline

1. S1(implementer): `verify.local.sh` の段。AC1
2. S2(implementer): コメントの `SQ`・`DQ`。AC2、AC6
3. S3(implementer): `read_body` の関数、2 つの規則と一覧と `EXEC_SEEN` を消す、コメント、テストの不変条件の検査と `exec` の行。AC3、AC4、AC5、AC5b、AC6
4. S4(inline、docs だけ): tech-debt。AC7

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(S1 の段を含む)、`shellcheck -S warning`、`./scripts/check-sync.sh`、`./scripts/check-template.sh`
- Spec compliance criteria to confirm: AC1〜AC7。AC4 は grep で、AC2 と S2 の差分がコメントだけであることは diff で
- Documentation drift to check: guard のヘッダーと各関数のコメントが、消した規則に触れていないこと。tech-debt の guard の行と `.awk` の行
- Evidence to capture: AC1 の FAIL の出力、AC5 の比較の件数

## Test plan

- Unit tests: 今のテストの期待値は変えない。不変条件の検査と、B 節の `exec >run.sh; echo 'sudo ls'` の行を足す(AC5b)
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: base と分割後の判定の比較(PR #214 の test で使った入力の集まり、991 件の payload)を、base の a0094fe5 とこの PR の guard で取り直す。mawk と gawk でもテストを回す
- Edge cases: 予約語で始まるコマンド(`if`、`for`、`{ … }`、`function`)、`exec` のリダイレクト(`exec >log`、`exec 3>f`)、`! exec`、`env exec`、行末のバックスラッシュが偶数・奇数の行が続くヒアドキュメントの本文(`read_body` の 2 つの道)
- Evidence to capture: 比較の件数と違い 0、許可リストの行を消した mutation で赤になる行

## Risks and mitigations

- 規則を消したあとで、誰かが DATACMD に予約語や包みを足すと、その場面のデータ区間が残り、見張りの一致が無視される。テストの不変条件の検査がその変更で赤になり、DATACMD の一覧の上のコメントにも書く(AC5b)。不変条件の検査が読む文字列の形(`split("…", datacmd_list, " ")`)が変わると検査が読めなくなるので、読めないときは検査を FAIL にする
- `read_body` の関数化で、境界(本文の最後の行に改行がない、`le > N`)の扱いがずれると、ヒアドキュメントの終わりの判定が変わる。ループの中身は変えず、範囲の始まりと終わりだけを引数にする。テストと判定の比較で確かめる
- コメントの `SQ` を戻すとき、コードの行の `SQ`(変数)を誤って変えると判定が変わる。S2 の差分がコメントの行だけであることを確かめる

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く(4 つのファイルの書き換えで、ファイルは増えも減りもしない)
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-09: ユーザーが承認ゲートで Approve。承認の前に、consult(consult-plan-guarddebt)の指摘(`(echo sudo ls)` は `(` の規則だけでも止まるので許可リストの確かめに使えない)と、Codex の plan advisory の MEDIUM(DATACMD に `exec` を 1 つ足すと穴が開くが、今のテストの行では気づけない)を反映し、テストに不変条件の検査と `exec >run.sh; echo 'sudo ls'` の行を足すことにした(AC5b)。返答で勧めた「4 と 2・3 を分ける」から、1 つの PR にまとめる形に変えたことはユーザーに伝えた。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [x] Implementation started
  - S1 完了(f09d8209、inline。1 ファイルに段を 1 つ足すだけで、引き継ぎより安い): `verify.local.sh` に `check_guard_awk` を足し、`run_static_checks` の `sh -n` の段の隣で root と template の両方を読む。static は rc 0(2 つの段とも OK)。scratch に写した木で `pre_bash_guard_commands.awk` に構文エラーを足すと、その段が awk のエラーを出して FAIL、`verify.local.sh` は rc 1(ほかの FAIL は、root だけを壊したので写しと食い違った check-sync の 1 つ)。戻すと rc 0
  - S2 完了(b312dc80、implementer/sonnet): コメントの 15 行(lex 6、commands 6、rules 3)の `SQ`・`DQ` を本物の引用符に戻した。コードの行は変わっていない(コメントと空行を除くと前後で一致)。テストは 2,063/0、static の `verify.local.sh` は rc 0。AC2 の grep の書き方の誤り: plan の grep は部分一致なので、変数名を指すコメントの `DQ_SUBST`・`DQ_EXP`・`DQ_SUB`(lex の 4 行)にも当たる。引用符の綴りとしての `SQ`・`DQ` は、語の境界を付けた `grep -nE '^[[:space:]]*#.*(^|[^A-Za-z0-9_])(SQ|DQ)([^A-Za-z0-9_]|$)'` で 0 件。AC2 はこの grep で確かめる(AC の意図は変えない)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
