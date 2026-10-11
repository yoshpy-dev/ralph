# Verify report: guard-wrapper-args

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-guard-wrapper-args.md(`- Approved:` の digest 5ebd931f0fcc は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff 577ebeaf...HEAD`(base 577ebeaf、HEAD 3c5b0066、4 ファイル、+580/-43)。コードの変更は `tests/test-pre-bash-guard.sh` だけ(78616500、888ec6f8、160eef84、20e282be、db5134f1)。docs は `docs/tech-debt/README.md` の 163 行(76f2f826、d2cb7794、b515e29a、db5134f1)、plan、self-review report。テストスイートと `run-verify.sh` の test mode は実行していない(テスト全件と 3 つの awk は /test が見る)。probe は worktree の guard と、scratch に置いた写し(`git archive HEAD .claude/hooks tests`)に対して行い、コマンドを hook の経路で直接渡した(macOS の /usr/bin/awk、jq あり・なし)。mutation の抜き取り 2 通りだけ、写しの木でテストのファイルを動かした
- Evidence: `docs/evidence/verify-2026-10-11-guard-wrapper-args.log`(static、shellcheck、check-sync、AC1 の probe、mutation、V-1 の probe の出力と、probe のスクリプト。gitignore の対象なのでコミットしない)。個別の出力は同じ名前に `-static`、`-shellcheck`、`-check-sync`、`-ac1`、`-mutation`、`-v1` を付けたファイル

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 手書きの 9 行が `wrapper_deny` にあり、D 節で同じラベルの deny の行として流れる。Scope の 25 行もあり、jq あり・なしとも deny。新しい検査が、`WRAPPER` の 9 名と 34 行の 1 語目の一致、34 行の末尾、34 行の対照が none であることで PASS。`WRAPPER` が読めないときは FAIL | Met(コードの読みと guard の probe。テストの中の PASS は /test) | `wrapper_deny` は `tests/test-pre-bash-guard.sh:1107-1144` の 34 行で、`:1145` で `check_modes D deny absent` で流れる(`edge_deny` の `:1042` と同じ引数)。最初の 9 行は base の `edge_deny` の 9 行(base の `:926-934`)と `cmp` で一致。ラベルは `check`(`:312`)が節、mode、コマンドだけで作るので、9 行のラベルは変わらない。D 節は `collect_corpus=no`(`:828`)のあとで、以後 `yes` に戻らないので、G 節の corpus にも入らない。25 行は Scope に並ぶ 25 の形とちょうど同じ(包みごとに env 5、command 2、exec 1、nohup 1、time 4、nice 2、stdbuf 2、timeout 3、xargs 5。888ec6f8 で 16 行、20e282be で 9 行が増え、20e282be の -2 は並べ替え)。3 つの `.awk` から `WRAPPER` を読むと 9 名で、34 行の 1 語目の集合と両方向で一致し、名前ごとの行数は 2〜6(exec と nohup が 2)。34 行とも末尾が ` sh -c 'git commit -n -m x'`。worktree の guard に直接渡すと、34 行は jq あり・なしとも deny/0、末尾を ` sh -c 'git commit -m x'` に替えた 34 の対照は jq あり・なしとも none/0。新しい検査のコードは `:1774-1836` で、集合の検査は `WRAPPER` が空なら `:1778` で FAIL、行のない名前・一覧にない 1 語目・末尾の不一致を 1 回の FAIL にまとめる。対照は行ごと・経路ごとに 1 件で、末尾の形でない行は jq の SKIP より先に FAIL |
| AC2: 写しで (a)〜(e) をすると新しい検査がその名前か行を出して FAIL。(f) `time` と `exec` の旗を空にした写しで足した行が FAIL。(g) 再確認の MEDIUM 7 の写しと、`skip_xargs` の `=` の道・`skip_opts` の `-` と数字の道を壊した写しで、足した 9 行のどれかが FAIL | Met(抜き取り 2 通りを確認。ほかは Progress checklist と self-review 再確認 2 の記録) | 写しの木でテストのファイルを動かした。(a) `WRAPPER` に `chrt` を足し、`cmd_pos` に `skip_opts(ctx, i + 1, "", "")` の分岐を足した写し(`diff` は 2 行): FAIL は 1 件だけで、新しい集合の検査が「chrt is in WRAPPER but has no row: add a row for chrt to wrapper_deny; …」を出した。PR #219 の検査の `chrt` の 2 件(jq あり・なし)は PASS で、AC2 (a) の「PR #219 の検査は通る」と合う(2,228 PASS / 1 FAIL)。(g) の (a)、`timeout` の値を取る長いオプションの一覧を空にした写し(`diff` は 1 行): FAIL は `D mode=<absent>: timeout --signal KILL 5s sh -c 'git commit -n -m x' -> deny` の jq あり・なしの 2 件だけ(got none, exit 0)で、その行の対照は PASS のまま(2,225 PASS / 2 FAIL)。(b) は承認済みの AC の字面が「1 つの名前の行を全部消す」になっていて、Progress の「AC2 (b) の読み方」と合う(どの名前にも 2 行以上あることは AC1 の probe で確かめた)。(b)〜(f) と (g) の残りは実行していない。Progress の S1・S3・S5 の項と self-review 再確認 2 の表が、写しごとに落ちた行を記録している |
| AC3: テスト全件が通る(macOS の awk、mawk、gawk)。件数は 2,108 + D 50 + F 1 + F 68。A〜C と E は変わらない。guard のファイルは変わっていない | Partially verified | `git diff 577ebeaf -- .claude templates` は空(0 バイト)。`git diff --name-only 577ebeaf HEAD` は plan、self-review report、tech-debt、テストのファイルの 4 つだけ。テストのファイルの写しは `templates/base/` にない。件数の式は 2,108 + 50 + 1 + 68 = 2,227 で、Progress の S5 の 2,227/0 と合う。抜き取りの mutation の 2 つの写しでも PASS と FAIL の合計は 2,229(`chrt` の 2 件が増えた写し)と 2,227 で、式と矛盾しない。テスト全件と 3 つの awk は /test の担当で、回していない |
| AC4: `./scripts/run-verify.sh` が rc 0。V-1 の文が、sentinel に掛かる行は分岐の読み違えで落ちないと書く。「`edge_deny` の 9 行」を前提にしたコメントが残らない。tech-debt の (h) の閉じの文が、行の存在はテストが強制し、行が引数を押さえているかは人の判断に残ると書き、足した行が押さえる書き方を書く。self-review の LOW 1〜6 が直っている | Partially verified | `./scripts/run-static-verify.sh`(`run-verify.sh` の static 側)は rc 0。test mode を含む `run-verify.sh` は /test。V-1 の文は `:1083-1096` にあり、「each holds a match of a sentinel rule (sudo, a force push, a hard reset), so it stays denied even when cmd_pos misreads the wrapper」と書く。HEAD の guard から `xargs` の分岐を消した写しで、コメントが挙げる `xargs` の 4 行と `xargs -n 1 sudo ls` は jq あり・なしとも deny/0 のまま、`xargs sh -c 'git commit -n -m x'` と `xargs -I{} sh -c …` は none/0 になった。`nice` の値を取る短い旗を空にした写しでは、`nice -n 5 sudo ls` が deny のまま、`nice -n 5 sh -c …` が none になった。`grep -n -i 'nine\|edge_deny'` で残るのは `edge_deny` の定義(`:831`)、流す行(`:1042`)、例の置き場所(`:1076`、`:1086`)、別の行の話(`:1194`)と、「among the first nine」(`:1053`、引数を置く行の説明)だけで、9 行を前提にした文はない。`wrapper_row_tail` と `wrapper_row_control` の旧名もない。tech-debt の (h) の閉じの文は「Whether a row puts its wrapper's own arguments before `sh -c` stays the author's call」と書き、分岐ごとの基準と足した 25 行の書き方、基準の外のもの(旗に付いた値、`env -S`、行が区別できない道、分岐が載せていないツールの旗)を書く。LOW 1〜6 は self-review の再確認 1 の表がすべて修正済みとし、LOW 5(旧名なし)は上の `grep` で、LOW 6(`why` を deny のときだけ足す)は `:1831-1833` を読んで確かめた。docs の書き方の点が 2 つ残る(V-1、V-2) |

plan の Scope、Design decisions との照合:

- Scope 1(9 行を `wrapper_deny` に移し、同じ `check_modes` で流す): Met。上の AC1
- Scope 2(集合の両方向の一致、末尾の形、対照、案内文、PASS の文言、`WRAPPER` が読めないとき FAIL): Met。行のない名前の案内は「add a row for W to wrapper_deny; if W takes arguments of its own, add rows that put them before sh -c by the per-branch criterion …」(`:1786`)、一覧にない 1 語目は「the name was removed from WRAPPER, or the row names a wrapper the guard does not have」(`:1793`)、PASS は「this checks only that the rows exist and their shape, not whether a row pins its wrapper's own arguments」(`:1800`)。Risk register の「FAIL の文言に行と対照の判定を出し、deny が中のコマンドから来ないと案内する」も `:1831-1833` が満たす
- Scope 3(コメントの直し): Met。書き方の点が 1 つ残る(V-1)
- Scope 4(25 行、man にある形だけ): Met。上の AC1。`nice -5` が macOS の man で古い形として受け付けられることは、self-review 再確認 2 が `man nice` と実行で確かめている
- Scope 5(LOW 1・3・4・5・6 の直し): Met(self-review の再確認 1 と上の `grep`)
- Scope 6(tech-debt の閉じの文): Met。書き方の点は V-2
- Design decisions の「分岐ごとの基準」: `wrapper_deny` の上のコメント(`:1052-1072`)の 3 つの箇条は、plan の基準と同じ区分で、例はどれも `wrapper_deny` の行にある。`cmd_pos`(`pre_bash_guard_commands.awk:132-140`)と照らすと、`nohup` は短い旗も長いオプションも持たず、`exec` は短い旗だけを持つ。`skip_opts` が分岐をまたいで共有する道は `--`(`:167`)、`=` の 1 語(`:169` の一覧にない形)、`-` と数字(`:173`)の 3 つ。`skip_env`、`skip_command`、`skip_xargs`(`:178-220`)の道もコメントの列挙と合う。`command -v`・`-V` の道はコマンドが走らないので deny の行にできず、`edge_none` の `command -v sudo`(`:1164`)が押さえるという括弧も事実に合う

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `Language scope: full fallback (unclassified:tests/test-pre-bash-guard.sh)`。shellcheck(hook と verify のスクリプト)、`sh -n`(hooks、root と template)、check-pipeline-sync、check-skill-sync(13)、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint(0 issues)、branch secret scan(577ebeaf..3c5b0066 clean)が通った |
| `shellcheck -S warning tests/test-pre-bash-guard.sh` | rc 0、0 件 | ShellCheck 0.11.0。既定の重さで見ると note が 3 種(SC2016 67 件、SC2329 1 件、SC1003 4 件)で、base 577ebeaf と種類も件数も同じ。この diff が足した指摘はない |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5、`PASS: all files in sync.` |
| `git diff --check 577ebeaf...HEAD` | rc 0 | 足したコメント行に 79 桁以上のものはない |
| `git diff 577ebeaf -- .claude templates` | 空 | guard のファイルは変わっていない |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-11-guard-wrapper-args.md` | 5ebd931f0fcc | plan の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| テストのヘッダー D 項目(`:59-63`) | Yes | `wrapper_deny` が手書きの行を持ち、F が名前ごとに 1 行以上を求めること、基準が `wrapper_deny` の上のコメントにあることを書く |
| テストのヘッダー F 項目(`:101-118`) | Yes | PR #219 の 2 形の検査と、新しい検査(両方向の一致、末尾、経路ごとの対照、行の存在と形だけを見ること、空の一覧で両方の検査が FAIL)を書き、コードと合う |
| `edge_deny` の中の案内(`:925`) | Yes | 「Wrappers before sh -c: wrapper_deny, after this list.」 |
| `wrapper_deny` の上のコメント(`:1044-1106`) | ほぼ Yes | 基準、行が区別できない道、`env -S` の扱いは事実に合う。例の 1 つが合わない(V-1) |
| F 節の PR #219 の検査の上のコメント(`:1694-1716`) | Yes | 9 行を前提にした文はなく、`wrapper_deny` の行と基準を指す。`timeout` の `+ 1` についての文は self-review の再確認 1 が写しで確かめた |
| F 節の新しい検査の上のコメント(`:1754-1773`) | Yes | コード(`:1774-1836`)と合う |
| `docs/tech-debt/README.md:163` | 一部ずれ | V-2、I-1 |
| `.claude/hooks/pre_bash_guard_rules.awk:399-406`(と `templates/base/` の写し) | Yes | 「a D row of tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads this list」は今も正しい(`wrapper_deny` の行も D 節で流れる)。I-2 |
| plan の Progress checklist と AC の欄 | 一部ずれ | I-3 |

### 指摘(docs は /verify では直していない)

- V-1(LOW、新規、判定に関係しない): `wrapper_deny` の上のコメント `:1083-1089` は「Other rows also run a wrapper with arguments of its own (…; echo 'sudo ls' | xargs -I{} sh -c {} and echo x | xargs sh -c 'git reset --hard' in B)」と書く。挙げた 5 行のうち、B 節の `echo x | xargs sh -c 'git reset --hard'`(`:585`)は `xargs` に自分の引数を付けていない。このファイルの「the wrapper's own arguments」は `sh -c` の前に置く包みの旗と値を指すので、この行は「引数を付けた行」の例にならない。sentinel に掛かる行は分岐の読み違えで落ちない、という結論は、この行を含めて probe のとおり成り立つ(`xargs` の分岐を消した写しで deny のまま)。直すなら、この行を括弧から外して「a hard reset」も外すか、前置きを「Other rows also run a wrapper before a command, some with arguments of its own」にする。コメント 1 文なので、次にこのファイルを触る機会でよい。plan の Scope の例(`nice -n 5 sudo ls`、`echo 'sudo ls' | xargs -I{} sh -c {}`)にはこの行は入っていなかった。S1 で足した例
- V-2(LOW、新規、/sync-docs で直せるもの): `docs/tech-debt/README.md:163` の 2 点。(1) (h) の先頭の、guard-test-gaps の時点の文「Nine rows of `edge_deny`, one per wrapper (`env sh -c 'git commit -n -m x'` and the like), turn red when …」が、今は行を持たない配列を指す。移った先の「(since test/guard-wrapper-args these rows are the `wrapper_deny` array, run in section D)」は 1,500 字ほどあとの、guard-wrapper-rows の閉じの文の中にある。`edge_deny` を `grep` して 9 行を探す人は見つけられない。最初の言及に「(now `wrapper_deny`)」を足すのがよい。(2) 閉じの文の「The criterion leaves out three things.」のあとに 4 つの文が続き、4 つ目(「An option of the real tool that a branch does not list is pinned by no row」)が 4 つ目の項目に読める。テストのコメント(`:1073-1083`)が挙げるのは前の 3 つだけなので、4 つ目を別の段にするか「three things」の数を外す
- I-1(情報、self-review の Follow-ups と同じ): 閉じの文の「(mutations (a) to (e) of the plan, all caught; test report of this PR)」の test report はまだない。/test の report ができたら、Related に `docs/reports/test-2026-10-11-guard-wrapper-args.md` を足す。Related の `docs/plans/active/2026-10-11-guard-wrapper-args.md` は、`/pr` が archive の path へ書き換えるので手で直す必要はない。表は 5 列のまま(区切りの `|` は base と同じ 6 個)、`~~` は 16 個で偶数
- I-2(情報、self-review の Follow-ups と同じ): `pre_bash_guard_rules.awk:404-406` は、F 節が一覧と `wrapper_deny` の行を突き合わせることを書いていない。偽ではない。guard のファイルは plan の Non-goals なので、次に guard を直す機会に足せばよい
- I-3(情報): plan の AC1〜AC4 の欄と、Progress の「Review artifact created」が未チェックのまま。self-review report は a57b6758 でコミットされ、bf6b12b8、8d3624a5、3c5b0066 で再確認が足されている。S2、S4、S6、S7 の行は「この commit」と書き、hash(76f2f826、d2cb7794、b515e29a、db5134f1)が入っていない。このブランチの insight event のファイル(`docs/insights/events/2026-10-11-guard-wrapper-args.jsonl`)は、この verify の前にはなかった(self-review の event がない)

## Observational checks

- AC1 の probe: 34 行と 34 の対照を、worktree の guard に jq あり・なしで渡した(136 回)。行はすべて deny/0、対照はすべて none/0。同じ実行の中で deny と none の両方が出ているので、probe の判定の読み取りは両方向に働いている
- mutation の写しは 2 組で、どちらも worktree との `diff` が 1〜2 行であることを確かめてから動かした。写しのテストのファイルは HEAD と同じ内容
- V-1 の probe: HEAD の guard、`xargs` の分岐を消した写し、`nice` の値を取る短い旗を空にした写しの 3 つに、コメントと tech-debt が挙げる行 10 本を jq あり・なしで渡した。tech-debt の「a copy of the guard without the `xargs` branch confirmed this」の根拠は `xargs` の例についてで、`nice -n 5 sudo ls` の例は `nice` の写しが裏付ける(self-review の初回も同じ点を書いている)。どちらの事実も成り立つ

## Coverage gaps

- テストスイートの実行(2,227 件、macOS の awk・mawk・gawk)と `run-verify.sh` の test mode は /test の担当で、回していない
- AC2 は (a) と (g) の (a) の 2 通りだけを、テストのファイルの中で FAIL になることまで確かめた。(b)〜(f) と (g) の残りは Progress checklist と self-review 再確認 2 の記録に頼る
- probe と mutation は macOS の /usr/bin/awk だけ。`for (k in WRAPPER)` の並びは awk ごとに違うが、集合の比較と行ごとの対照は並びに依らない
- 行が包みの引数を押さえているかはテストが判断しない(plan の Non-goals)。1 つの名前に 2 行以上あるので、1 行だけを消しても集合の検査は FAIL しない(Progress の「AC2 (b) の読み方」)。道ごとに 1 行しかない道(`env --unset HOME`、`xargs --max-args=1`、`nice -5` など)の行を消すと、その道は黙って外れる。この PR が決めた範囲
- マージ後に main のチェックアウトから動く hook の効き目は、マージ前には確かめられない(この PR は guard を変えない)

## Verdict

- Verdict: pass
- Verified: AC1(34 行、1 語目の集合、末尾、対照、D 節での流し方とラベル、`WRAPPER` が空のときの FAIL の分岐)。AC2 の (a) と (g) の (a)。guard のファイルは変わっていない。静的解析はすべて通り、plan の digest も一致した。V-1 の直しの主張(sentinel に掛かる行は分岐の読み違えで落ちず、`git commit -n` の行は落ちる)は HEAD の guard の写しで成り立った
- Partially verified: AC2 の残り(記録に頼る)、AC3(guard のファイルが変わっていないことは確認済み。テスト全件と件数は /test)、AC4(static 側は rc 0、コメントと tech-debt の文は基準に合い、言い過ぎはない。test mode は /test。書き方の点が V-1、V-2 に残る)
- Not verified: テスト全件の実行、mawk と gawk、(b)〜(f) の mutation でテストが FAIL を記録すること、マージ後の効き目
- pass にした理由: コードの AC は満たしている。残りは /test の担当の項目と docs の書き方だけで、指摘は LOW 2 件(V-1 はコメントの例 1 つ、V-2 は tech-debt の 2 点)と情報 3 件。どれも判定とテストの結果に関係しない

## 再 verify(V-1 の修正 10d73d88 のあと)

- Date: 2026-10-11
- Scope: `git diff 356535bb..10d73d88`(テストのファイルのコメント 4 行の差と plan の Progress)。テストスイートは実行していない
- Evidence: `docs/evidence/verify-2026-10-11-guard-wrapper-args-recheck-shellcheck.log`、`docs/evidence/verify-2026-10-11-guard-wrapper-args-recheck-static.log`(gitignore の対象)

### V-1 の状態

| 指摘 | 状況 | 確かめた点 |
| --- | --- | --- |
| V-1(`wrapper_deny` の上のコメントの例) | 修正済み | `tests/test-pre-bash-guard.sh:1083-1089` は「Other rows also run a wrapper, with arguments of its own (nice -n 5 sudo ls, xargs -I{} sudo ls {} and xargs -I{} flock /tmp/l git push -f in edge_deny; echo 'sudo ls' \| xargs -I{} sh -c {} in B) or bare (echo x \| xargs sh -c 'git reset --hard' in B)」。引数つきに挙げた 4 行は、どれも `sh -c` やコマンドの前に包みの旗を置く(`-n 5`、`-I{}`)。旗のない `xargs sh -c 'git reset --hard'` だけが「bare」に分かれた。括弧の sentinel(sudo、force push、hard reset)は 5 行すべてを覆ったまま。続く結論の文(`xargs` の分岐を消した写しの結果、`git commit -n` の行だけが落ちること)は変わっていない。この結果は最初の verify の V-1 の probe と同じ |

- コメントだけの変更であることを確かめた。356535bb と HEAD で、`#` で始まる行を除いたテストのファイルは `cmp` で一致する。足したコメント行に 79 桁以上のものはなく、`git diff --check 356535bb..HEAD` は空
- 段落を通して読み、宙に浮いた句はない(「or bare (…)」は「with arguments of its own (…)」と並ぶ)
- plan の Progress は、S2・S4・S6・S7 の「この commit」が hash(76f2f826、d2cb7794、b515e29a、db5134f1)になり、最初の verify の I-3 の後半が直った。S8 の項は 10d73d88 の中身(V-1 の直し、V-2 は sync-docs に回す)と合う。AC の欄と「Review artifact created」は未チェックのまま(I-3 の前半、情報)。plan の digest は 5ebd931f0fcc のままで、承認は有効

### 静的解析(再 verify)

| Command | Result | Notes |
| --- | --- | --- |
| `shellcheck -S warning tests/test-pre-bash-guard.sh` | rc 0、0 件 | 単独のコマンドで実行 |
| `./scripts/run-static-verify.sh` | rc 0 | 単独のコマンドで実行。`Language scope: full fallback (unclassified:tests/test-pre-bash-guard.sh)`、gofmt ok、golangci-lint 0 issues、branch secret scan(577ebeaf..10d73d88 clean)、`All verifiers passed.` |
| `git diff 577ebeaf -- .claude templates` | 空 | guard のファイルは変わっていない |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-11-guard-wrapper-args.md` | 5ebd931f0fcc | plan の `- Approved:` と一致 |

### 残り

- V-2(tech-debt の 2 点)と I-1(Related に test report の path)は /sync-docs で直す予定で、この再 verify では見ていない
- テスト全件(2,227 件、3 つの awk)は /test の担当。10d73d88 はコメントだけの変更なので、件数と判定は変わらないとみる。テストを流しての確認はしていない

## Verdict(再 verify)

- Verdict: pass
- Verified: V-1 の修正(例の分け方が事実に合い、結論の文は変わらない)、コメントだけの変更であること、`shellcheck -S warning` と `run-static-verify.sh` の rc 0、guard のファイルが変わっていないこと、plan の digest
- Partially verified: 最初の verify と同じ(AC2 の残り、AC3、AC4 の test 側)
- Not verified: テスト全件の実行、V-2 と I-1 の直し(/sync-docs の段)
- pass にした理由: V-1 は直り、新しい指摘はない。残る LOW は V-2 の 1 件で、/sync-docs で直す
