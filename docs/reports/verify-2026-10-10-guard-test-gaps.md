# Verify report: guard-test-gaps

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-guard-test-gaps.md(`- Approved:` の digest 94d656c12780 は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff 49ac046c...HEAD`(base 49ac046c、HEAD c5e7c90e、11 ファイル、+329/-82)。コードの変更は 468fc73c(`WRAPPER`、`cmd_pos` の 1 行、テストの不変条件の検査と D 節の 11 行)。73486db9 は tech-debt だけ、4f3a4414 は guard のコメントだけ。テストスイートと `run-verify.sh` の test mode は実行していない(AC5 のテスト全件と PR #215 の比較、AC3 の 18 通りの mutation でテストが赤になることは /test が見る)。probe にはテストの配列にある行だけを使った
- Evidence: `docs/evidence/verify-2026-10-10-guard-test-gaps.log`(gitignore の対象なのでコミットしない。スクリプトは scratchpad の `vtg/` の `static.sh`・`locate.sh`・`mut.sh`・`cmp.sh`・`drift.sh`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `rules.awk` の BEGIN に `WRAPPER`、`cmd_pos` は包みの分岐の前に `WRAPPER` にない名前の位置を返す | Met | `pre_bash_guard_rules.awk:407-408` が `split("env command exec nohup time nice stdbuf timeout xargs", wrapper_list, " ")` で `WRAPPER` を作る。`cmd_pos`(`pre_bash_guard_commands.awk:122`)の追加は `:131` の `if (!(nm in WRAPPER)) return i` の 1 行だけで、予約語・`function`・代入を飛ばす 3 行の後、`nm == "env"` の分岐の前にある。分岐の中身(`skip_env` など)は base と同じ。`WRAPPER` の 9 語と `cmd_pos` の `nm == "…"` の 9 語は同じ集合(`locate.sh` で `diff` が一致)。`WRAPPER`・`wrapper_list` を使うのはこの 3 行と template の写しだけ |
| AC2: 不変条件の検査が実行時の DATACMD と `WRAPPER` を読み、`nm == "…"` を読まない。空なら FAIL。`exec`・`if` を DATACMD に足すと FAIL。分岐を `cmd_pos` の外に移しても名前は検査に残る | Met | 検査は `tests/test-pre-bash-guard.sh:1537-1572`。`:1540` の dump の BEGIN が DATACMD を `D `、`WRAPPER` を `W ` の接頭辞で出し、`:1541` で guard の 3 つの `.awk` の後ろに `-f` で足して読む。テストに `nm ==` と `commands_awk` は残っていない(どちらも 0 件)。失敗の文言は `:1552`(awk の失敗。stderr の先頭 300 文字を付ける)、`:1554`(DATACMD が空)、`:1556`(`WRAPPER` が空)、`:1561`(`a name in WRAPPER`)、`:1564`(`a reserved word`)。検査のコードをテストから切り出し、`.awk` の写しに当てた: HEAD は PASS(20 語、9 包み、21 予約語)。DATACMD に `exec` を足すと FAIL(`exec (a name in WRAPPER)`)、`if` を足すと FAIL(`if (a reserved word)`)、`WRAPPER` を空にすると FAIL(`no WRAPPER name read`)、DATACMD を空にすると FAIL(`no DATACMD name read`)、`rules.awk` に構文の誤りを足すと FAIL(`awk exited 2 …` と awk のメッセージ)。`timeout` の分岐を `cmd_pos` の外の関数 `is_timeout` に移した写しでは、新しい検査は 9 包みを読む。そこへ DATACMD に `timeout` を足すと FAIL(`timeout (a name in WRAPPER)`)になる。同じ写しに base の検査を当てると 8 包みしか読まず、`timeout` を足しても PASS だった。(h) の穴が塞がったことを示す |
| AC3: 包みの 9 行が jq あり・なしで deny。`WRAPPER` の要素と `cmd_pos` の分岐を 1 つずつ消す 18 通りでテストが赤 | Met(静的と抜き取り。18 通りのテストの実行は /test) | 9 行は `tests/test-pre-bash-guard.sh:900-908`、`edge_deny` の中にあり、`:1025` の `check_modes D deny absent` が読む。plan の Scope の 9 形と一字ずつ同じ。worktree の guard と template の写しで 9 行とも deny/deny(jq/no-jq)、base 49ac046c でも deny/deny、旧版(`tests/fixtures/guard-1c4cea5a/`)は none で、plan の「旧版は none」と合う。抜き取りの mutation 2 通り: `WRAPPER` から `timeout` を消した写しでは `timeout 5 …` の行だけが none、`nice` の分岐を `nice_x` に変えた写しでは `nice -n 5 …` の行だけが none になり、ほかの 8 行は deny のまま。残り 16 通りは実装者の記録(plan の Progress、S1)と self-review の 18 組の probe が赤と書いている。/verify では回していない |
| AC4: (i) の 2 行が jq あり・なしで 2 個は deny、3 個は none。下限を `q > a` にした写しで 2 行とも赤 | Met(静的と抜き取り) | 行は `:963`(`edge_deny`)と `:1076`(`edge_none`、`:1207` の `check_modes D none absent` が読む)。テストの行をそのまま評価して `od -c` で見ると、本文の行は `\` が 2 個と 3 個。worktree と template で 2 個の行は deny/deny、3 個の行は none/none(base も同じ)。`trailing_backslashes`(`pre_bash_guard_lex.awk:560-565`)の `q >= a` を `q > a` にした写しでは、2 個の行が none/none、3 個の行が deny/deny に入れ替わる |
| AC5: 判定が変わらない(テスト全件 jq あり・なし、base との比較 0 件) | Partially verified | テスト全件と PR #215 の比較の入力は回していない(/test)。抜き取り: テストの A〜D 節を stub の `check_modes` で読み、600 行(A 76、B 188、C 29、D 307)を base と HEAD の 3 つの `.awk` に直接渡して、出力と終了コードを比べた。違いは 0 件。陽性対照として HEAD の lex を下限の mutant に替えると、違いは (i) の 2 行だけになる。guard のコードの差は `cmd_pos` の 1 行と `WRAPPER` の 2 行だけで、`pre_bash_guard.sh` の差はコメントの行だけ(コメントでない変更行は 0 行)、`lex.awk` は変更なし |
| AC6: root と template が同じ、`shellcheck -S warning`、`sh -n`、static の `verify.local.sh` が通る、4 ファイルが 800 行未満 | Met | `./scripts/check-sync.sh` rc 0(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0)。guard の 4 ファイルと `lib_json.sh` は root と template で `cmp` rc 0。`shellcheck -S warning`(0.11.0、guard の `.sh` 2 本とテスト)は 0 件。`sh -n` は guard の `.sh` 2 本で rc 0、テストは `bash -n` で rc 0。`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` は rc 0、FAIL 0 で、`awk parse of the guard in .claude/hooks` と `… in templates/base/.claude/hooks` の段を含む。行数は `.sh` 234、`lex` 621、`commands` 339、`rules` 480(template も同じ) |
| AC7: `run-verify.sh` rc 0、tech-debt のテストの穴の行の (h)(i) が解消済み | Partially verified | `./scripts/run-static-verify.sh`(`run-verify.sh` の static mode)は rc 0。test mode を含む `run-verify.sh` は回していない(/test)。`docs/tech-debt/README.md:163` の (h)(i) は Debt・Impact・Why deferred・Trigger の 4 列すべてで閉じたと書いてあり、Related にこの plan がある。閉じの文の中身(`WRAPPER` を実行時に読む、分岐を外へ移しても検査に残る、18 通りが赤、m1 で 2 行とも赤)は AC2〜AC4 の抜き取りと合う。ただし (i) の書き方がそろっていない(V-2、V-3。self-review LOW 3 と同じ)。別の行 `:160` に、この PR で事実と違うようになった文が残る(V-1) |

Plan の Assumptions との照合:

- `WRAPPER` にない名前を先に返しても判定は変わらない: 9 語の集合が分岐の集合と同じで、分岐にない名前は base でも `else return i`(`commands.awk:141`)で同じ位置を返す。600 行の比較で違い 0 件。Met
- `WRAPPER` も DATACMD と同じく BEGIN の順で読める: dump のファイルは `-f` の最後なので guard の BEGIN の後に動き、HEAD で 9 包みを読んだ。Met
- 包みの 9 行は包みを飛ばしたときだけ `no_verify` で止まる: 抜き取りの 2 通りで、その包みの行だけが none になった。旧版は 9 行とも none。Met
- (i) の 2 行は下限で逆になる: `q > a` で 2 行とも逆になった。Met

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)`。shellcheck、`sh -n`(hooks 20 本)、guard の awk の parse 2 段、settings の `jq -e`、Codex hook の 3 段、check-sync、check-pipeline-sync、check-skill-sync(13)、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint(0 issues)、branch secret scan(49ac046c..c5e7c90e clean)が通った |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0、FAIL 0 | awk の parse の段は root と template の 2 つ |
| `shellcheck -S warning` | 0 件 | `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |
| `sh -n` / `bash -n` | rc 0 | guard の `.sh` 2 本は `sh -n`、テストは `bash -n` |
| `./scripts/check-sync.sh` | rc 0 | DRIFTED 0、`PASS: all files in sync.` |
| `./scripts/check-template.sh` | rc 0 | `Template structure looks good.` |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |
| `cmp`(guard の 4 ファイルと `lib_json.sh`、root と template) | 5 件とも rc 0 | |
| 3 つの `.awk` の parse(`LC_ALL=C awk -f lex -f commands -f rules </dev/null`) | rc 0、出力なし | macOS の /usr/bin/awk だけ。gawk と mawk は手元の PATH にない |
| `wc -l` | 234 / 621 / 339 / 480 | すべて 800 未満。template も同じ |
| `git diff --check 49ac046c...HEAD` | rc 0 | |
| `./scripts/plan-visual.sh digest` | 94d656c12780 | plan の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| guard のヘッダー item 3(`pre_bash_guard.sh:30-35`) | Yes | 「the wrappers (WRAPPER in pre_bash_guard_rules.awk) env …, and stdbuf」。並べた 9 語は `WRAPPER` と同じ集合(並びは違う)。`:35` で詰め直しが止まっている(I-4) |
| awk の呼び出しの上(`pre_bash_guard.sh:183`)と `rules.awk:3` | Yes | どちらも「NOEXEC, WRAPPER and DATACMD」で、BEGIN の中の順と同じ。2 つの一覧だけを挙げる文は repo のどこにも残っていない |
| `rules.awk` の BEGIN のコメント(`:399-406`、`:415-423`) | Yes | 包みの説明(timeout は時間の後)、名前と分岐が対であること、D 節の行と F 節がこの一覧を読むことを書き、コードと合う。DATACMD の上の文は「no name in WRAPPER」になった |
| `cmd_pos` のコメント(`commands.awk:119-121`) | Yes | 「the wrappers, which are the names in WRAPPER … each with its own branch below; the first other name is the command name」。コードと合う |
| ヘッダー item 6(`pre_bash_guard.sh:126`)の「a wrapper such as env, command, sh, nice, builtin or exec」 | Yes | base からある文で、1 語目の許可リストの話。`sh`・`builtin` は `WRAPPER` にないが、広い意味の wrapper として読める(self-review の再 review と同じ判断) |
| テストのヘッダー D・F 項目(`tests/test-pre-bash-guard.sh:53-78`、`:82-97`)と F 節のコメント(`:1518-1536`)、9 行と 2 行のコメント | Yes | 実行時に 2 つの一覧を読むこと、awk の失敗と空の一覧で FAIL、包みの行が赤になる理由、バックスラッシュの数を書き、コードと抜き取りの結果に合う |
| `docs/tech-debt/README.md:163`(テストの穴) | 一部ずれ | V-2、V-3、I-1 |
| `docs/tech-debt/README.md:160`(guard の限界) | No | V-1 |
| ほかの docs(`docs/quality/`、`.claude/rules/`、`internal/`、README) | Yes | `git grep 'DATACMD\|NOEXEC\|cmd_pos'` は guard・テスト・tech-debt・この plan にしか当たらない。`stdbuf` も同じ |
| plan の Progress checklist | 一部ずれ | I-2、I-3 |

### 指摘(/sync-docs 向け。docs は /verify では直していない)

- V-1(LOW、新規): `docs/tech-debt/README.md:160` の Debt 列、refactor/guard-debt-cleanup の閉じの文「The premise the two rules guarded is now a test: section F … (… the wrapper names are read from the `nm == "…"` comparisons in `cmd_pos`)」は、今の仕組みの説明として書かれている。この PR から検査は `WRAPPER` を実行時に読み、`nm ==` は読まない。plan の Scope は `:163` の行だけなので plan の範囲外だが、この PR が変えた事実なので同じ PR で直すのがよい。直し方は、括弧の終わりを「the wrapper names were read from the `nm == "…"` comparisons in `cmd_pos`; since test/guard-test-gaps they are read from `WRAPPER` at run time」のようにする
- V-2(LOW、self-review LOW 3 と同じ): `:163` の Debt 列で、(h) は本文を `~~…~~` で消してから「Closed in …」を続けるが、(i) は本文を消さずに後ろへ 1 文を足しただけ。そのため「No row kills a mutant …」「The test suite stays at 2068/0 under that mutant」が現在形で残る。(i) の本文も消してから「Closed in …」を続ける
- V-3(LOW、self-review LOW 3 と同じ): `:163` の Trigger 列の「a new wrapper goes into `WRAPPER` and gets its own row of `edge_deny`」は決まりごとで、「;」で並ぶきっかけの列挙に混ざる。外すか、「(h) and (i) done in test/guard-test-gaps (a new wrapper goes into `WRAPPER` with its own row of `edge_deny`)」のように括弧に入れる
- I-1(情報): `:163` の Related は `docs/plans/active/2026-10-10-guard-test-gaps.md` を指す。今はファイルがあるので `check_tech_debt_plan_refs` は通る。`/pr` の `scripts/archive-plan.sh` が plan を動かす前に archive の path へ書き換える(`:54`)ので、手で直す必要はない
- I-2(情報): plan の Progress checklist の「Review artifact created」が未チェックのまま。self-review report は 8f237091 と c5e7c90e でコミット済み。AC5 の未チェックは /test 待ちなので正しい
- I-3(情報、self-review の再 review の細かい点と同じ): plan の Progress の self-review の行は、残した文を「`cmd_pos` の上の」「wrapper の走査の話」と書く。実際の場所は `pre_bash_guard.sh:126`(ヘッダー item 6)で、1 語目の許可リストの話
- I-4(情報、self-review の再 review の細かい点と同じ): `pre_bash_guard.sh:35` の「its flags), and stdbuf. Names are」で段落の詰め直しが止まっている。体裁だけで、中身は正しい

## Observational checks

- 包みの 9 行と (i) の 2 行(テストの行をそのまま切り出したもの)を、worktree の guard、template の写し、base 49ac046c、旧版、mutant 3 つに通した。結果は Spec compliance の AC3・AC4 のとおりで、表は evidence の log の `mut.sh` の節にある
- F 節の検査のコード(`:1537-1572` をそのまま切り出したもの)を、`.awk` の写し 8 組(HEAD、DATACMD に `exec`、`if`、`WRAPPER` が空、DATACMD が空、構文の誤り、`timeout` の分岐を外へ移したもの、それに DATACMD の `timeout` を足したもの)に当てた。比較のため、base の検査も外へ移した 2 組に当てた
- A〜D 節の 600 行を base と HEAD の awk に直接渡した。違いは 0 件で、陽性対照(下限の mutant)では違いが (i) の 2 行だけに出る

## Coverage gaps

- テストスイートの実行、PR #215 の test の比較の入力(3,913 件)、`run-verify.sh` の test mode は /test の担当で、回していない
- AC3 の 18 通りのうち /verify が抜き取ったのは 2 通り(`WRAPPER` の `timeout`、`nice` の分岐)。抜き取りは判定の段で見た。テストが赤になることは、行の期待値(deny)から推した
- awk は macOS の /usr/bin/awk(20200816)だけ。gawk と mawk では parse も判定も見ていない(plan の Progress には、実装者が gawk と mawk でテストを回した記録がある)
- 600 行の比較は awk に直接渡したもので、hook の経路(JSON の取り出し、jq なしの経路)は通していない。hook の経路で見たのは 11 行だけ
- `WRAPPER` の名前と `cmd_pos` の分岐が対であることを機械的に確かめる検査はない。今の 9 組は D 節の 9 行が押さえる。10 個目を足すときに行も足すことは、`rules.awk:399-406` のコメントと tech-debt の Trigger 列が求めているだけ(self-review の Coverage gaps と同じ)
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない

## Verdict

- Verdict: pass
- Verified: AC1、AC2、AC4、AC6。AC3 は静的と 2 通りの抜き取りで確かめた。plan の Assumptions はコードと抜き取りに合う。静的解析は全部通り、plan の digest も一致した
- Partially verified: AC5(600 行の awk の比較で違い 0 件。テスト全件と PR #215 の比較は /test)、AC7(static mode は rc 0、tech-debt の (h)(i) は 4 列で閉じた。test mode は /test。書き方の LOW が残る)
- Not verified: テストの実行、AC3 の残り 16 通り、gawk と mawk、マージ後の効き目
- pass にした理由: コードの AC はすべて満たし、残りは /test の担当の項目と docs の書き方だけ。指摘は LOW 3 件(V-1 は新規、V-2・V-3 は self-review LOW 3 と同じ)と情報 4 件で、どれも /sync-docs で直せる
