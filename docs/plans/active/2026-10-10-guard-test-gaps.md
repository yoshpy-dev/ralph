# guard-test-gaps

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-10
- Related request: ユーザーの依頼「テストの穴が 2 つを修正して」(2026-10-10)。PR #215(49ac046c)で `docs/tech-debt/README.md` のテストの穴の行に (h)(i) として残した 2 つ
- Related issue: N/A
- Type: test
- Branch: test/guard-test-gaps

## Objective

PR #215 で残した guard のテストの穴を 2 つ塞ぐ。判定は 1 つも変えない。

1. (h) DATACMD の不変条件の検査は、`cmd_pos` が飛ばす包みの名前を、`pre_bash_guard_commands.awk` の `function cmd_pos` の中の `nm == "…"` から読んでいる。包みの分岐を別の関数に移すと、その名前は検査から黙って外れる。同じく、`cmd_pos` が先頭で飛ばす語(`if then else elif do while until ! { }`)はコードに直に書かれていて、テストは自前の一覧で確かめている
2. (i) `trailing_backslashes`(`pre_bash_guard_lex.awk`)の数え始めの下限を変える mutation を、テストが殺せない。ヒアドキュメントの本文の、バックスラッシュだけでできた行の扱いを固定する行がない(base の `read_body` も同じで、PR #215 の test が見つけた)

## Scope

- (h): 包みの名前と、`cmd_pos` が先頭で飛ばす語を、guard の BEGIN の一覧にし、`cmd_pos` とテストが同じ一覧を使う
  - `pre_bash_guard_rules.awk` の BEGIN: `WRAPPER`(`env command exec nohup time nice stdbuf timeout xargs`)と `CMDSKIP`(`if then else elif do while until ! { }`)の一覧を足す。DATACMD の一覧の上のコメントを、この 2 つの一覧を指す形に直す
  - `pre_bash_guard_commands.awk` の `cmd_pos`: 先頭で飛ばす語の比較を `r in CMDSKIP` にし、包みの分岐の前に `nm` が `WRAPPER` になければその位置を返す。包みごとの飛ばし方の分岐(`skip_env` など)は今のまま。`function` を 2 語飛ばす処理と代入を飛ばす処理も今のまま
  - `tests/test-pre-bash-guard.sh` の不変条件の検査: 包みの名前を、ソースの `nm == "…"` ではなく、実行時の `WRAPPER` と `CMDSKIP` から読む(DATACMD と同じく、3 つの `.awk` の後ろに `BEGIN` を足して出力させる)。どちらも空なら FAIL。DATACMD が `WRAPPER`・`CMDSKIP`・テストの予約語の一覧のどれとも重ならないことを確かめる
- (i): `tests/test-pre-bash-guard.sh` の D 節に 2 行を足す(test の報告の提案)
  - `edge_deny`: `cat <<EOF`、本文がバックスラッシュ 2 個だけの行、`EOF`、`git commit -n -m x`、`EOF`(2 個は継がないので `EOF` で本文が終わり、`git commit -n` が命令になる)
  - `edge_none`: 同じ形でバックスラッシュ 3 個(3 個目で次の行と継ぐので `EOF` で終わらず、`git commit -n -m x` は本文になる)
- 4 つの guard のファイルは、root と `templates/base/` の写しをバイト単位で同じに保つ
- `docs/tech-debt/README.md`: テストの穴の行の (h)(i) を解消済みにする

## Non-goals

- 判定の変更。テストの今の期待値は 1 つも変えない
- `cmd_pos` の包みごとの飛ばし方(オプションの読み方)を変えること
- `$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'` の形(tech-debt の Debt (b))
- tech-debt のテストの穴の行のほかの項目

## Assumptions

- `cmd_pos`(49ac046c)は、`WR` が `if then else elif do while until ! { }` のどれかなら 1 語進み、`function` なら 2 語進み、代入なら 1 語進む。そのあと `cname` の値 `nm` が 9 つの包みのどれかなら、それぞれの関数で飛ばし、どれでもなければその位置を返す。包みの分岐の前に「`WRAPPER` になければ返す」を置いても、9 つの名前の分岐は同じに動き、それ以外の名前は今の `else return i` と同じに返る
- `cmd_pos` が先頭で飛ばす語の比較は生の語 `r`(`WR`)で、包みの比較は `cname` の値 `nm` で行う。一覧にしても、どちらの値で比べるかは変えない
- DATACMD を実行時に読む今の検査(`BEGIN { for (k in DATACMD) print k; exit }` を 3 つの `.awk` の後ろに足す)は、BEGIN の順に頼っている。`WRAPPER` と `CMDSKIP` も同じ BEGIN で作るので、同じ読み方が使える
- (i) の 2 行は、PR #215 の test の J の行の 24 番と 34 番に当たり、`trailing_backslashes` の下限を `q > a` に変えた mutation(m1)で判定が変わる

## Affected areas

- `.claude/hooks/pre_bash_guard_commands.awk`、`.claude/hooks/pre_bash_guard_rules.awk`、それぞれの `templates/base/.claude/hooks/` の写し
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (guard の中の 2 つの比較を一覧に置き換え、テストの読み方と行を足すだけで、部品どうしの呼び出し・データの形・境界を変えない)

## Design decisions

- **包みと先頭の語を guard の一覧にし、テストもそれを読む**。テストがソースの書き方を読む今の形は、分岐の場所や書き方を変えると黙って外れる。一覧を `cmd_pos` の入口の判定に使えば、`cmd_pos` が飛ばす名前は一覧にある名前だけになるので、テストが一覧を読めば漏れがない。包みの分岐を足して一覧に足し忘れると、その包みは飛ばされず、包みのテストで分かる
- **振る舞いで確かめる案はとらない**。DATACMD の各名前 D について `D sudo ls` が none になることを確かめる案は、`timeout` のように次の語も飛ばす包みや、オプションを読む包みで見落としが出る
- **(i) は test の報告が示した 2 行にする**。本文に見張りの語を置かず、継ぎの数え方だけで `git commit -n`(no_verify)が命令か本文かが決まる形
- Critical forks: None(どちらも判定を変えない。一覧にするかどうかは、検査の漏れをなくす方法として決めた)

## Acceptance criteria

- [ ] AC1: `pre_bash_guard_rules.awk` の BEGIN に `WRAPPER` と `CMDSKIP` があり、`cmd_pos` は先頭の語を `CMDSKIP` で、包みを `WRAPPER` で判定する(包みの分岐の前に、`WRAPPER` にない名前を返す)
- [ ] AC2: 不変条件の検査が、実行時の DATACMD、`WRAPPER`、`CMDSKIP` を読み、ソースの `nm == "…"` を読まない。`WRAPPER` か `CMDSKIP` が空なら FAIL。写しで `exec` か `if` を DATACMD に足すと FAIL。写しで包みの分岐を `cmd_pos` の外の関数に移しても、その名前は検査に残る(`WRAPPER` から読むので)
- [ ] AC3: D 節の 2 行が、jq あり・なしで期待どおり(2 個の行は deny、3 個の行は none)。`trailing_backslashes` の下限を `q > a` に変えた写しで、少なくとも一方が赤になる
- [ ] AC4: 判定が変わらない。`bash tests/test-pre-bash-guard.sh` が、今の 2,068 件と足した行を含めて全部通る(jq あり・なし)。base(49ac046c)の guard とこの PR の guard に、テストの配列の行と PR #215 の test の比較の入力を渡し、判定の違いが 0 件
- [ ] AC5: root と `templates/base/` の写しがバイト単位で同じ(`./scripts/check-sync.sh`)。`shellcheck -S warning`、`sh -n`、`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`(guard の awk の構文の段を含む)が通る。4 つの guard のファイルは 800 行未満
- [ ] AC6: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` のテストの穴の行の (h)(i) が解消済みになる

## Implementation outline

1. S1(implementer): `WRAPPER` と `CMDSKIP` の一覧、`cmd_pos` の判定、コメント、不変条件の検査の読み方、D 節の 2 行、template の写し。AC1〜AC5
2. S2(inline、docs だけ): tech-debt の (h)(i)。AC6

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`、`./scripts/check-sync.sh`、`./scripts/check-template.sh`
- Spec compliance criteria to confirm: AC1〜AC6
- Documentation drift to check: DATACMD の一覧の上のコメント、`cmd_pos` のコメント、テストの F 節の説明、tech-debt のテストの穴の行
- Evidence to capture: mutation の結果、判定の比較の件数

## Test plan

- Unit tests: D 節の 2 行。不変条件の検査の読み方を変える
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: base(49ac046c)との判定の比較(PR #215 の test が使った入力の集まり)。mawk と gawk でもテストを回す
- Edge cases: 包みのあとに包みが続く形(`env nohup echo …`)、先頭の語が続く形(`if ! { …`)、`function` の形、代入のあとの包み
- Evidence to capture: AC2・AC3 の mutation で赤になる行と検査

## Risks and mitigations

- `cmd_pos` の入口の判定を変えて、包みでない名前の扱いがずれると、コマンドの位置が変わり判定が変わる。分岐の中身は変えず、「一覧にない名前を返す」を足すだけにする。base との判定の比較で確かめる
- 一覧と分岐がずれる(一覧に足して分岐を足さない)と、その名前は `else return i` で返り、包みとして飛ばされない。安全側(コマンドの位置を早めに決める)で、不変条件の検査はその名前を DATACMD に入れないことを求めるだけになる
- (i) の行は `git commit -n` を含むので、手元の Bash の行に書くと別の hook に止められる。テストのファイルに書き、Bash の行には書かない

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く(guard の 2 つの `.awk` の書き換え)
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
