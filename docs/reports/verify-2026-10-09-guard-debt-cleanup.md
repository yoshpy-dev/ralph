# Verify report: guard-debt-cleanup

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-debt-cleanup.md(`- Approved:` の digest f37da93972d7 は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff a0094fe5..HEAD`(base a0094fe5、HEAD a709e0bd、14 ファイル、+479/-124)。コードの変更は f09d8209 と 566d8a8d(`scripts/verify.local.sh`)、d7dac506(guard の 3 つの `.awk` とテスト)、c7274f3b(テストと guard のコメント)。b312dc80 は `.awk` のコメントだけ、f435dd5a は tech-debt だけ。テストスイートと `run-verify.sh` の test mode は実行していない(AC5 の実行と比較、AC5b でテストが赤になることは /test が見る)。probe にはテストの配列にある形だけを使った
- Evidence: `docs/evidence/verify-2026-10-09-guard-debt-cleanup.log`(gitignore の対象なのでコミットしない。スクリプトは `scratchpad/vgd/` の `ac1.sh`・`acs.sh`・`ac5.sh`・`probe.sh`・`mkcases.py`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: static の `verify.local.sh` が root と template の `.awk` を読む段を回して OK、壊した `.awk` で FAIL と 0 以外の終了 | Met | HEAD の `./scripts/run-static-verify.sh` に `==> awk parse of the guard in .claude/hooks` と `... in templates/base/.claude/hooks` が出て、どちらも OK。`git archive HEAD` を展開した scratch の木で `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` を回した。手を加えない木は rc 0・FAIL 0。root の `pre_bash_guard_commands.awk` の末尾に `function broken_syntax( {` を足すと rc 1 で、root の段が awk の `syntax error at source line 232 ... pre_bash_guard_commands.awk` を出して FAIL(もう 1 つの FAIL は写しと食い違った check-sync)。同じ行を両方の写しに足すと、FAIL は 2 つの awk の段だけ。両方の `pre_bash_guard.sh` を消した木は rc 0 で、awk の段の行は 0 行(飛ばされる)。`.sh` を残して両方の `rules.awk` を消すと、2 つの段が `can't open file` で FAIL、rc 1(S1b の「`.sh` があって `.awk` が欠ければ FAIL」と合う) |
| AC2: `.awk` のコメントに `SQ`・`DQ` が残らない、S2 はコメントの行だけ | Met | Progress に記録した語の境界つきの `grep -nE '^[[:space:]]*#.*(^\|[^A-Za-z0-9_])(SQ\|DQ)([^A-Za-z0-9_]\|$)'` は root と template の 6 ファイルで 0 件(rc 1)。plan の部分一致の grep は lex の 4 行(240〜242、316)に当たるが、どれも変数名 `DQ_SUBST`・`DQ_EXP`・`DQ_SUB`。b312dc80 の追加・削除の行のうちコメントでないものは 0 行、コメントの行は 15 行を書き換え(-15/+15)。3 つの `.awk` からコメントと空行を除くと、b312dc80^ と b312dc80 で `cmp` が一致 |
| AC3: 行末のバックスラッシュを数える処理が 1 つの関数、ループは 1 つだけ | Met | `while (q >= … && at(q) == BS)` の形は `pre_bash_guard_lex.awk:563`(`trailing_backslashes` の中)と template の同じ行だけ。base は 2 つ。`read_body` の 2 か所は `bs = trailing_backslashes(ls, le)`(`:533`)と `bs = trailing_backslashes(p2, le)`(`:540`)。base と HEAD のコードの行(コメントと空行を除く)の差は、この 2 つの置き換え、`read_body` の局所変数から `q` を外したこと、関数の追加だけ。ループの中身と下限(`ls`、`p2`)は base と同じ |
| AC4: `RESW`・`resw_list`・`EXEC_SEEN` が 4 ファイルに残らず、コメントが許可リストの場合として書く | Met | `grep -n 'RESW\|resw_list\|EXEC_SEEN'` は guard の 4 ファイルと template の写しで 0 件(rc 1)、`-i` でも 0 件。base では `.sh` 1・`commands` 6・`rules` 4 行。`in_data` の上(`commands.awk:235-243`)は「the allowlist in end_cmd, which also covers a compound command such as if, for, case or { and an exec with a redirection」。ヘッダー item 6(`pre_bash_guard.sh:119-143`)は「This also covers a compound command at the top level ... and an exec with a redirection, since their first word is neither a data command nor git」。`end_cmd` の許可リストの上(`commands.awk:67-76`)も同じ。DATACMD の一覧の上(`rules.awk:405-414`)に「No reserved word ... and no wrapper that cmd_pos steps past ... may be listed here」 |
| AC5: 判定が変わらない(テスト全件、base との比較 0 件) | Partially verified | テストと比較は実行していない(/test)。静的に見たこと: テストの差分で消えた行は header の 1 行(item F の書き換え)だけで、期待値の行は消えていない。B 節に `exec >run.sh; echo 'sudo ls'`(`tests/test-pre-bash-guard.sh:620`)を足し、`guard_deny_only_forms` は `check_modes B deny`(`:757`)で読まれる。guard のコードの差は AC3・AC4 の行と、消した 2 つの規則だけ(`pre_bash_guard.sh` のコードの行は base と `cmp` で一致、c7274f3b の guard の差分はコメントだけ)。観察: テストの 6 行を base・HEAD の root・template に jq あり・なしで通すと判定は全部同じ(下の Observational checks)。1,912 回で違い 0 は Progress の S3 の記録で、取り直していない |
| AC5b: 消した規則の前提がテストで守られる | Met(静的と観察。テストが赤になることは /test) | 不変条件の検査は `tests/test-pre-bash-guard.sh:1490-1556`。DATACMD は guard の 3 つの `.awk` と `BEGIN { for (k in DATACMD) print k; exit }` のファイルを 1 つの awk で読み(`:1512-1515`)、包みの名前は `cmd_pos` の `nm == "…"` から読み(`:1516-1526`)、予約語は `:1510` の一覧(`for` を含む)。awk の失敗と空の一覧は FAIL(`:1535-1540`)。同じ読み方を scratch の写しに当てた: HEAD は 20 語、DATACMD に `exec` を足した写しは `exec` を含む 21 語、`for` を足した写しは `for` を含む 21 語。包みの名前は `env command exec nohup time nice stdbuf timeout xargs` なので、どちらの写しも検査の重なりに入る。判定: `exec` を足した写しでは新しい行が none/none になり、後ろに `sh run.sh` を置いた既存の行は deny のまま(plan の Design decisions のとおり)。許可リストの `NODATA` の行を消した写しでは `if grep -q 'sudo ' file; then echo ok; fi`(`:594`、B 節)が none/none になる |
| AC6: 4 ファイルが写しと同じ、`shellcheck -S warning` と `sh -n` が通る、800 行未満 | Met | `./scripts/check-sync.sh` rc 0(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0)。guard の 4 ファイルと `lib_json.sh` は root と template で `cmp` rc 0。`shellcheck -S warning`(`scripts/verify.local.sh`、guard の `.sh` 2 本、`tests/test-pre-bash-guard.sh`)は 0 件。`sh -n` は guard の `.sh` 2 本と `verify.local.sh` で rc 0、テストは `bash -n` で rc 0。行数は `.sh` 233、`lex` 621、`commands` 335、`rules` 471(写しも同じ) |
| AC7: `run-verify.sh` rc 0、tech-debt の (e) と `.awk` の行を閉じる | Partially verified | `./scripts/run-static-verify.sh`(`run-verify.sh` の static mode)は rc 0。test mode を含む `run-verify.sh` は回していない(/test)。tech-debt `:160` の (e) は Debt・Trigger を取り消し線にして「Closed in refactor/guard-debt-cleanup」を付け、`:189` は Impact・Trigger を取り消し線にして閉じた。ただし `:160` の Debt 列の別の段落が消した `EXEC_SEEN` を今の仕組みとして書いており(V-1)、self-review LOW 3 の 4 項目も残る(V-2〜V-5)。`:163` の Trigger は発火したが持ち越しの記録がない(V-6) |

Plan の Assumptions との照合:

- DATACMD と RESW は重ならない: a0094fe5 の `rules.awk` の 2 つの `split` の文字列を比べ、DATACMD と `git` のどれも RESW にない。Met
- `cmd_pos` の包みの名前は DATACMD にない: HEAD の `cmd_pos` から読んだ 9 語と、実行時の DATACMD の 20 語に重なりはない。`cmd_pos` が `exec` まで進むのは、それより前の語が予約語(`commands.awk:124` の一覧)、`function` とその名前、代入、包みのどれかだったときか、1 語目が `exec` のときだけで、どの 1 語目も `data_first_ok` が 0 を返す。Met
- `NODATA` を読むのは `in_data` だけ: 読むのは `commands.awk:245` の `if (NODATA) return 0` だけ。`in_data` を呼ぶのは `sentinel` と `sentinel_str`(`rules.awk:352`、`:369`、`:377`)だけで、`sentinel` を呼ぶのは END(`rules.awk:470`)だけ。`judge` は読まない。Met
- `verify.local.sh` は scaffold に配らない: `scripts/check-sync.sh:39` の ROOT_ONLY_EXCLUSIONS にあり、`templates/base/scripts/verify.local.sh` はない。`run-verify.sh:48-51` が呼ぶ。Met
- N03・N04 が 507 行の probe で同値だった件は tech-debt と cycle 3 の test report の記録で、取り直していない

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0(16 s) | `Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)`。shellcheck、`sh -n`(hooks 20 本)、新しい awk の段 2 つ、settings の `jq -e`、Codex hook の 3 段、check-sync、check-pipeline-sync、check-skill-sync(13)、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint(0 issues)、branch secret scan(a0094fe5..a709e0bd clean)が通った |
| `shellcheck -S warning` | 0 件 | `scripts/verify.local.sh`、`.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |
| `sh -n` | rc 0 | guard の `.sh` 2 本と `verify.local.sh`。テストは `bash -n` で rc 0 |
| `./scripts/check-sync.sh` | rc 0 | DRIFTED 0、ROOT_ONLY 0、`PASS: all files in sync.` |
| `./scripts/check-template.sh` | rc 0 | `Template structure looks good.` |
| 3 つの `.awk` の parse(`LC_ALL=C awk -f lex -f commands -f rules </dev/null`) | rc 0・出力なし | BWK awk 20200816(root と template)、mawk 1.3.4(ubuntu:24.04 の Docker、ネットワークなし。DATACMD は 20 語)。gawk は手元にない |
| `wc -l` | 233 / 621 / 335 / 471 | すべて 800 未満。template も同じ |
| `./scripts/plan-visual.sh digest` | f37da93972d7 | plan の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| guard のヘッダー(`pre_bash_guard.sh`) | Yes | item 6(`:119-143`)は複合コマンドと `exec` のリダイレクトを許可リストの場合として書き、残る `NODATA` は `(`・`)`、区切りの語、バックスラッシュと改行。awk の呼び出しの上(`:176-188`)は「the command lists NOEXEC and DATACMD」。item 3(`:30-31`)の予約語は `cmd_pos` が飛ばす語の説明で、今のコード(`commands.awk:124-125`)と合う。item 6 の `(`・`)` の例は subshell だけで、`in_data` のコメントは `case` の節の `)` と関数定義の `()` も挙げる。誤りではないので指摘にしない |
| `end_cmd`・`data_first_ok`・`cmd_pos`・`in_data` のコメント | Yes | `cmd_pos`(`commands.awk:117-118`)は「or 0 ... as command -v or exec >log. judge calls it.」だけになり、`EXEC_SEEN` の説明は消えた。`data_first_ok`(`:92-109`)は 0 を返す語に予約語を足した。`in_data`(`:235-243`)と `lex_cmds` の分岐(`lex.awk:146`、`:151`)は合う |
| `rules.awk` の先頭と BEGIN のコメント | Yes | 先頭は「the command lists NOEXEC and DATACMD」。BEGIN(`:405-414`)の説明は `lex_cmds` の `(`・`)` の規則と許可リストの役割に合う。`(x)` と書く POSIX の形の `case` の節は閉じの `)` が深さ 1 で読まれるので `(` の規則に掛かる(self-review の再 review の細かい点と同じ)。判定には関係しない |
| `tests/test-pre-bash-guard.sh` の header item F | Yes | `:89-96` は、実行時に DATACMD を読むこと、予約語の一覧はテストに書くこと、包みの名前は `cmd_pos` の `nm == "…"` から読むこと、awk の失敗と空の一覧は FAIL を書き、`:1490-1556` のコードと合う |
| `scripts/verify.local.sh` の先頭のコメント | Yes | `:7-8` の static の一覧に「the awk parse of the guard」がある |
| `docs/tech-debt/README.md:160`(guard の限界) | No | V-1、V-2、V-3 |
| `docs/tech-debt/README.md:163`(テストの穴) | No | V-4、V-6 |
| `docs/tech-debt/README.md:189`(`.awk` の分類) | No | V-5 |
| ほかの docs(`docs/quality/`、`docs/architecture/repo-map.md`、`.claude/rules/`、`internal/`) | Yes | `git grep -i 'reserved.word\|exec with a redirection\|exec rule\|NODATA'` は guard とテストと plan・report を除くと tech-debt にしか当たらない。`docs/quality/quality-gates.md` は `verify.local.sh` の段の一覧を持たない |
| plan の Progress checklist | 一部ずれ | I-1 |

### 指摘(/sync-docs 向け。docs は /verify では直していない)

- V-1(LOW、新規): `docs/tech-debt/README.md:160` の Debt 列のデータ区間の定義(「Five cases have no data region at all」から始まる段落)が、消した仕組みを今のものとして書く。「a group or compound command at the top level (a subshell `(...)`, a brace group, or a reserved word such as `if`, `for`, or `case` in command position; `NODATA`, set in `lex_cmds` and `end_cmd`); an `exec` with a redirection (`EXEC_SEEN`, set in `cmd_pos`)」。d7dac506 のあと、`EXEC_SEEN` はなく、予約語と `{` で始まるコマンドと `exec` のリダイレクトは 1 つ目の場合(許可リストの外れ、`end_cmd`)に入る。`lex_cmds` が立てるのは `(`・`)` の場合だけ。直し方の案: 1 つ目の場合に「which also covers a compound command whose first word is a reserved word or `{`, and an `exec` with a redirection」を足し、3 つ目を「a `(` or `)` at the top level (`NODATA`, set in `lex_cmds`)」にし、`exec` の場合を消して「Four cases」にする。同じ段落の後ろの「`exec` may have changed a file descriptor」は理由の説明として残してよい。self-review は `git grep` の当たりを「plan と tech-debt の経緯の文だけ」としたが、この段落は経緯でなく今の定義
- V-2(LOW、self-review LOW 3 の (4)): `:160` の Debt (e) の閉じの文「section F ... checks that `datacmd_list` holds no reserved word」。c7274f3b から検査は実行時の DATACMD を読む(`datacmd_list` 以外の書き方で足した名前も捕まえる)。Trigger (e) の「a change that adds a name to `datacmd_list` keeps the invariant check of section F passing」も DATACMD の言い方にする
- V-3(LOW、self-review LOW 3 の (1)): `:160` の Why deferred に「the comment items of (e) were closed in refactor/guard-awk-split except the `SQ` spelling, and its code items are still open」が現在形で残る。閉じたことは同じ列の最後の 1 文(「refactor/guard-debt-cleanup settled them with that evidence.」)だけ
- V-4(LOW、self-review LOW 3 の (2)): `:163` の「the equivalent mutants are now J02, L03, N02, N03, N04, and AL02 (... so N03 and N04 no longer exist)」は 1 つの文の中で一覧と括弧が食い違う
- V-5(LOW、self-review LOW 3 の (3)): `:189` の Debt item の列に取り消し線も `(RESOLVED …)` もなく、「a syntax error in one of the three files is caught only by the parse check in section F」が現在形で残る。Why deferred の「Its parse check in section F covers the syntax of the three files for now.」も同じ
- V-6(LOW、新規): `:163` の Trigger「the next change ... to section F of `tests/test-pre-bash-guard.sh` (add the lex and commands copies next to the rules copy, and an unreadable copy that is skipped for root)」が発火した。この PR は F 節に不変条件の検査(`tests/test-pre-bash-guard.sh:1490-1556`)と header item F の文を足したが、lex と commands の写しは足していない。plan の Non-goals が「F 節が自動で見るのは `rules.awk` がない写しだけ」を外したので作業の持ち越しは正しいが、行に「carried over by refactor/guard-debt-cleanup」の記録がない。Related の列にもこの plan がない(N03・N04 の注記はこの PR で足した)
- I-1(情報): plan の Progress checklist の「Review artifact created」が未チェックのまま。self-review report は 2583435f と a709e0bd でコミット済み
- I-2(情報): `:160` と `:189` の Related は `docs/plans/active/2026-10-09-guard-debt-cleanup.md` を指す。`scripts/archive-plan.sh:54-79` が plan を動かす前に tech-debt の `docs/plans/active/<name>` を archive の path に書き換えるので、`/pr` のあとも `check_tech_debt_plan_refs` は通るはず(未実行)。ほかの項目にある「(`/pr` rewrites this path when it archives the plan)」の注記はこの 2 つにない

## Observational checks

テストの配列にある 5 行と、ヘッダーの例の 1 行(データとして通る形)を、1 ファイル 1 形で guard に渡した(`scratchpad/vgd/probe.sh`、macOS の BWK awk。表は jq あり/jq なし)。m_exec は HEAD の写しの `datacmd_list` に `exec` を、m_for は `for` を足したもの、m_allow は HEAD の写しから `end_cmd` の許可リストの `NODATA` の行を消したもの。

| 形(テストの行) | base a0094fe5 | HEAD root | HEAD template | m_exec | m_for | m_allow |
| --- | --- | --- | --- | --- | --- | --- |
| `exec >run.sh; echo 'sudo ls'`(`:620`、新しい行) | deny/deny | deny/deny | deny/deny | none/none | deny/deny | none/none |
| `exec >run.sh; echo 'sudo ls'; sh run.sh`(`:613`) | deny/deny | deny/deny | deny/deny | deny/deny | deny/deny | none/none |
| `exec 3>run.sh; echo 'sudo ls' >&3; sh run.sh`(`:614`) | deny/deny | deny/deny | deny/deny | deny/deny | deny/deny | deny/deny |
| `if grep -q 'sudo ' file; then echo ok; fi`(`:594`) | deny/deny | deny/deny | deny/deny | deny/deny | deny/deny | none/none |
| `builtin exec >run.sh; echo 'sudo ls'; sh run.sh`(`:654`) | deny/deny | deny/deny | deny/deny | deny/deny | deny/deny | none/none |
| `echo 'never use sudo here'`(ヘッダーの例) | none/none | none/none | none/none | none/none | none/none | none/none |

- base・HEAD・template の判定は 6 形とも同じ
- m_exec では新しい行だけが none になる。後ろに `sh run.sh` を置いた既存の行は deny のままなので、既存の行だけでは DATACMD への `exec` の追加に気づけない(plan の Design decisions と S3 の記録のとおり)
- m_for は判定を変えない。`for` の追加を捕まえるのは不変条件の検査だけで、検査の入力(実行時の DATACMD)に `for` が入ることは確かめた
- m_allow では `exec 3>…` の行だけが deny のまま残る(`>&3` の出力先の規則)。ほかの 4 形は許可リストだけが止めている
- 新しい `check_guard_awk` は awk の stderr を出力に混ぜて「出力なし」を求める。テストの F 節の parse の検査(`tests/test-pre-bash-guard.sh:1480`)も同じく `2>&1` で出力なしを求めるので、CI の awk で警告が出れば両方が落ちる。mawk 1.3.4 では出力なし

## Coverage gaps

- テストスイート(2,068 件、jq あり・なし、mawk・gawk)、base との 1,912 回の比較、AC5b の写しでテストが赤になることは実行していない(/test)。上の表は 6 形だけ
- `run-verify.sh` の test mode(AC7 の rc 0 の残り半分)は回していない(/test)
- gawk での parse は手元に gawk がなく未確認。CI(ubuntu-latest)の awk でも通るはずだが、確かめていない
- guard は main のチェックアウトから動くので、この branch の guard が session で効くことはマージ前には確かめられない
- テストにない形での guard の頑健さは probe していない(テストの行だけを使った)
- self-review LOW 2 の (2)(包みの名前はソースの `nm == "…"` から読むので、一部を別の関数に移すと黙って減る)は残る。足すと効く最小の検査は、読んだ包みの名前に `exec` と `env` が入っていることを確かめる 1 行

## Verdict

- Verdict: partial-pass
- Verified: AC1、AC2、AC3、AC4、AC6。AC5b は静的と観察で確かめた(検査の行と読み方、写しで検査の入力が変わること、新しい行と `if grep -q` の行がそれぞれの写しで none になること)。plan の Assumptions はコードと合う。消した 2 つの規則のほかに guard のコードの行は `read_body` の関数化しか変わっていない。静的解析は全部通り、plan の digest も一致した
- Partially verified: AC5(期待値の行は消えておらず、6 形の判定は base と同じ。テスト全件と 1,912 回の比較は /test)、AC7(static mode の `run-verify.sh` は rc 0。tech-debt に V-1〜V-6 が残る)
- Not verified: テストの実行、gawk、マージ後の実際の効き目
- partial にしたのは AC7 の tech-debt の書き方が残るため。コードの AC は満たしている。指摘は LOW 6 件(V-1 と V-6 は新規、V-2〜V-5 は self-review LOW 3 と同じ)と情報 2 件で、どれも tech-debt と plan の記録の書き方。マージを止めるものではなく、/sync-docs で直せる
