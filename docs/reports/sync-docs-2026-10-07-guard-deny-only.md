# sync-docs report: guard-deny-only

## Cycle 1

- Date: 2026-10-08(probe の実行は 2026-10-07 の深夜)
- Plan: `docs/plans/active/2026-10-07-guard-deny-only.md`
- Pipeline cycle: 1(`cycle-count.json` が 1)。差分は merge base `f423f230` から branch HEAD `095af1d7`(fix/guard-deny-only)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-07-guard-deny-only.md`(S2c の回で Merge 可。LOW 2 件)、
  `docs/reports/verify-2026-10-07-guard-deny-only.md`(`1346740f`。pass、LOW 2 件 V-1、V-2、既知の D-1)、
  `docs/reports/test-2026-10-07-guard-deny-only.md`(`095af1d7`。pass、1576/0、mutation 44 個がすべて赤)

## Summary

この差分で古くなっていた文書は、tech-debt の guard の限界の行と、テストの穴の行だった。どちらも S2c より前の内容で、guard が 991 行から 1286 行になったことで、行番号のすべてが外れていた。2 行を書き直し、/test が見つけた `tests/test-secret-scan.sh` の取り違えを新しい行にした。plan は進捗の節だけを直した。

guard が ask を返すと書いた文書は残っていなかった(下の「Surfaces checked」)。root と `templates/base/` の同期は、変更前から変更後まで通っている。

依頼のうち、2 点は事実が依頼の文面と違ったので、row には実測のほうを書いた。

- `ssh host 'sudo ls'` は blind spot ではない。新しい guard も旧い guard も deny にする(見張りが `sudo ` を拾う)。blind spot になるのは、旧版の 4 つの文字列を含まない形で、`ssh host 'git push origin --force'` は両方とも none だった。
- `git grep sudo file` は単独では通る、という旧 row の記述が誤りになっていた。いまは deny(`git` は `DATACMD` にないので見張りが拾う。旧版も deny)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行、160 行目) | 下の「guard の限界の行」。1 行の中身を書き直し、5 列のまま |
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 下の「テストの穴の行」 |
| `docs/tech-debt/README.md`(末尾、166 行目) | 新しい行 1 件(`tests/test-secret-scan.sh` の固定パス) |
| `docs/plans/active/2026-10-07-guard-deny-only.md` | `Progress checklist` の中だけ。`Test artifact created` にチェックを付け、/test の進捗の行を 1 件足した。digest は変更後も `7efd47f36781`(Approved 行と一致) |
| `docs/insights/events/2026-10-07-guard-deny-only.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 1) |
| `docs/reports/sync-docs-2026-10-07-guard-deny-only.md` | この report |

### guard の限界の行

- 行番号をすべて外した。guard は関数名と変数名(`scan_words`、`judge`、`NOEXEC`、`noexec_list`、`DATACMD`、`datacmd_list`、`sentinel`、`git_rules`、`stage_note`、`pipe_decide`、`cmd_pos`)で指す。S2c の 1 回で全部の番号が外れたので、番号を貼り直すと次にまた外れる。残した数字は 1286 行(`wc -l`)、awk のプログラムが 1114 行(128 行目の `awk '` から 1243 行目の閉じ引用符まで)、`internal/cli/migrate.go` の 77 行目(確認済み)と、いつの guard かを示す `095af1d7`。tests の行番号(`:382`、`:426-428`)は「section B の `ac2` 配列」に置き換えた
- (a)「wrapper scan の誤検知」を「2 つの検査の誤検知」に広げた。wrapper scan の 4 形(`apt-get remove sudo -y`、`apt-get install sudo vim`、`bash -c 'x' sudo ls`、`flock l git grep sudo file`)と、見張りが引き継いだ形(`cp sudo dest`、`apt-get install sudo 2>/dev/null`、`git grep sudo file`、`git push --force-if-includes origin main`、読むだけのコマンドの出力を `sed` や `sort` に流す形)。データ区間の条件(`DATACMD` のコマンドだけで、ファイルに書かない)と、同じ pipeline の `cat` や `grep` 止まりなら通ることも書いた
- (a) に V-1 を短く足した。`my-sudo ls` と `x.sudo ls` は旧版 deny、新版 none。語の境界が文字、数字、`_`、`.`、`-` を語の一部とみなすため。別のコマンドなので弱くはなっていない
- (b) の blind spot を直した。`ssh host 'sudo ls'` を外し、`ssh host 'git push origin --force'`(none)に置き換えた。`git commit -F -` は、heredoc だけでなく here-string も読むと直した。文字列を取る runner の穴(`watch 'git push origin --force'`、`su -c`、`flock -c`、`script -c`、`parallel`、`tmux new-session`、`find -exec sh -c`)を足した。旧版も none で、旧版より弱くなってはいない。guard のヘッダーの「is still denied, as the previous guard did」が当てはまるのは旧版の 4 文字列だけである、とも書いた
- (c) の「awk がないとき旧版の 4 規則が決める」は、コードと一致していた。awk を PATH から外した場合と、終了コード 2 の awk の場合で、19 件の probe がどちらも旧版の判定と一致した。旧版の見逃しに `git push origin -f`、`git -C x reset --hard`、`sudo` + タブを足した。`my-sudo ls` が deny になることも足した
- (d) の行数を 991 から 1286 にした
- 「Why deferred」に、`noexec_list` に足すだけでは誤検知が消えない、と書いた。scratch の guard で `apt-get`、`flock`、`bash` を `noexec_list` に足しても 4 形とも deny のままで、`apt-get` を `datacmd_list` にも足すと `apt-get` の 2 形が none になった。`bash -c` と `flock` の 2 形は `datacmd_list` に足していないので、確かめていない。「Trigger」の直し方もそれに合わせた
- Related に、この PR の self-review(S2c の findings)、verify(D-1、V-1)、test(Test gaps)を足した

### テストの穴の行

- (a) を「どれも pin されていない」から「一部が pin されている」に書き直した。39ed2759 で pin されたもの(wrapper scan の 3 形、見張りの 4 形、`apt-get install sudo` の none と `bash script.sh` の none、再読み込みの場所ごとの deny の例、512 字の窓と区画の端)と、まだ pin されていないもの(blind spot、runner の穴、`apt-get install sudo vim`、`| sed` の形)を、それぞれ `tests/test-pre-bash-guard.sh` に当たって確かめて書いた。`ssh host 'sudo ls'` は blind spot ではなく deny で、テストにもない、と書いた
- (c) に、tester が ubuntu:24.04 で mawk 1.3.4 と gawk 5.2.1 を手で流した結果(1576/0)と、busybox awk は流していないことを足した
- (f) を新しく足した。V-2: AC8 は 5 秒、section H のテストは 10 秒を超えたときだけ失敗する。実測は 0.28〜1.63 秒。10 秒にした理由は test にも plan にも書かれていない
- 前書き、Impact、Why deferred、Trigger、Related をそれに合わせて直した

### 新しい行: `tests/test-secret-scan.sh` の固定パス

- `expect_exit` が出力を `/tmp/ralph-secret-scan-test.out` と `.err` に書く。ファイルを読む側(失敗時の dump、`expect_stderr_contains`、`expect_stderr_not_contains`)と `trap` も同じ 2 つのパスを使う(8 行)。同時に 2 つ動かすと互いに上書きし、先に終わったほうが消す
- 単独では 123 passed、0 failed。2 つを同時に起動すると、どちらも rc 1 で 2 件ずつ落ちた。落ちる検査は実行ごとに違った(`AC-15` と `AC-9`、`AC-7` と `a failed driver listing names git's exit code`)。この sync-docs の中で再現した
- 直し方: 2 つのファイルを `$workdir` の下に移す。`expect_exit` の最初の呼び出し(50 行目)は `workdir` の設定(40 行目)より後なので、順序は問題ない。`trap` は `$workdir` をすでに消す
- 今直さない理由: コードの変更になり、post-implementation pipeline が `/self-review` からのやり直しになる

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| guard を ask と書く文書 | `docs/`(reports、plans、evidence、insights を除く)、`.claude/rules/`、`.claude/skills/`、`.agents/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`.codex/`、`templates/` に対し、`ask`、`permissionDecision`、`permission_mode`、`bypassPermissions`、`confirmation` などで `git grep` した。guard が確認を出すと書く箇所はなかった。残った一致は、guard と関係ない用法(`ralph.toml` の `guarded` は driver CLI の既定のモード、`/org` skill の `bypassPermissions` は Claude Code の初回の承諾ダイアログ、`ralph upgrade` の移行の確認、`.codex/config.toml` の `permission_mode` のコメント)と、RESOLVED の tech-debt 行 |
| `.codex/README.md`(root と template、115〜119 行目) | 「`deny` is the only decision the guard returns: it never asks for confirmation and does not read `permission_mode`」。コードと一致。2 面は `check-sync.sh` で一致 |
| `.claude/rules/ralph/git-commit-strategy.md`(root と template、70 行目) | 「`pre_bash_guard.sh` blocks dangerous patterns at command time」。ask に触れていない。変更なし |
| `README.md`、`.codex/hooks/README.md` | 「bash guard」「Bash guardrails」と呼ぶだけで、挙動を書いていない。変更なし |
| `AGENTS.md` の Repo map、`docs/architecture/repo-map.md`、`docs/quality/` | `pre_bash_guard.sh`、`lib_json.sh`、`tests/test-pre-bash-guard.sh`、`tests/test-lib-json.sh` に触れる箇所がない。新しい `tests/test-lib-json.sh` を載せる一覧もない。変更なし |
| guard のヘッダー(1〜105 行目) | 今回の probe と一致。runner の文字列の穴は、ヘッダーの「is still denied, as the previous guard did」が旧版の 4 文字列にしか当たらない点だけがずれる。コードの変更になるので触っていない。tech-debt の (b) に書いた |
| root と `templates/base/` | `pre_bash_guard.sh` は root と template で同じ判定(この report の probe 107 件を両方に流し、全件の判定が一致)。`check-sync.sh` は PASS |

## Found but left

- guard のヘッダーのコメントの直し(runner の文字列の穴)は、guard の変更になるので、この sync-docs では触れていない。
- `$'\tsudo ls'` は旧版 deny、新版 none だった(probe)。見張りの語の境界で、`$'…'` の `\t` の `t` が `sudo` に接している。シェルでは `<tab>sudo ls` という 1 語の名前のコマンドになり、`sudo` は動かない。V-1 と同じ種類の差で、tech-debt の行には入れていない。
- `ssh host 'sudo ls'` を pin するテストは足していない(テストの変更になる)。テストの穴の行の (a) に、deny で、テストにない、と書いた。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-guard-deny-only.md` | `7efd47f36781`(編集の前後とも。plan の Approved 行と一致) |
| tech-debt の 3 行の列の数 | 5 列、行末が `\|`(160、163、166 行目) |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill が一致) |
| `./scripts/run-static-verify.sh` | PASS(rc 0。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan 参照、gofmt、golangci-lint 0 issues、branch secret scan `f423f230..095af1d7` clean。言語の範囲は `lib_json.sh` を分類できず full にフォールバック) |
| probe 一式(新旧の guard、jq あり) | 下の「Probe results」 |

注: tech-debt と plan の編集は `Edit` で行った。probe 用の guard の複製(`noexec_list` と `datacmd_list` に足した版)は scratch に置き、worktree の guard は変えていない。

## Probe results

新しい guard は root の `.claude/hooks/pre_bash_guard.sh`、旧い guard は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`。どちらも jq ありの PATH で、各コマンドを `tool_input.command` に入れた payload で流した。決定は none か deny(ask は 1 件も出なかった)。

| コマンド | 新 | 旧 |
|---------|----|----|
| `cp sudo dest` | deny | deny |
| `apt-get install sudo vim` | deny | deny |
| `apt-get install sudo` | none | none |
| `apt-get install sudo 2>/dev/null` | deny | deny |
| `git push --force-if-includes origin main` | deny | deny |
| `git push origin --force-if-includes` | none | none |
| `echo 'sudo ls' \| sed 's/x/y/'` | deny | deny |
| `grep -r 'sudo ' . \| sed s/a/b/` | deny | deny |
| `grep -r 'sudo ' . \| sort` | deny | deny |
| `echo 'sudo ls' \| cat`、`echo 'sudo ls' \| grep x` | none | deny |
| `apt-get remove sudo -y`、`bash -c 'x' sudo ls`、`flock l git grep sudo file` | deny | deny |
| `git grep sudo file` | deny | deny |
| `my-sudo ls`、`x.sudo ls`、`visudo -c` | none | deny |
| `./sudo ls`、`/usr/bin/sudo ls`、`sudo ls`、`=sudo ls` | deny | deny |
| `watch 'git push origin --force'` | none | none |
| `watch 'git -C x reset --hard'`、`watch 'git push origin -f'`、`watch 'git commit --no-verify -m x'` | none | none |
| `watch 'git push --force'`、`watch 'git reset --hard'`、`watch 'sudo ls'` | deny | deny |
| `watch git push origin --force`(引用符なし) | deny | none |
| `su -c '…'`、`flock /tmp/l -c '…'`、`script -c '…' /dev/null`、`parallel '…' ::: a`、`tmux new-session '…'`、`find . -exec sh -c '…' \;`(中身は `git push origin --force`) | none | none |
| `sh -c 'git push origin --force'`、`xargs -I{} sh -c 'git push origin --force'`、`find . -exec git push origin --force \;` | deny | none |
| `ssh host 'sudo ls'`、`ssh host sudo ls` | deny | deny |
| `ssh host 'git push --force'` | deny | deny |
| `ssh host 'git push origin --force'` | none | none |
| `git commit -m "$msg"`、`cat file \| sh`、`bash script.sh`、`git config core.hooksPath /dev/null`、`echo "$(id)" \| git commit -F -` | none | none |
| `git commit -F - <<< "$(id)"`、`git commit -F - <<EOF`(本文に `$(id)`) | deny | none |
| `{su,}do ls`、`?udo ls`、`[s]udo ls`、`$'\x73udo' ls`、`$'\163udo' ls` | none | none |
| `gh pr create --body "$(cat <<'EOF'` ... `git push --force` ... `EOF` `)"` | deny | deny |

awk なしの経路(awk を PATH から外した場合と、終了コード 2 の awk の場合)。19 件の判定が旧版と同じだった。

- deny: `echo 'never use sudo here'`、`visudo -c`、`my-sudo ls`、`git commit -m 'drop --force and git reset --hard from docs'`、推奨の HEREDOC の commit、`git push --force`、`git push -f`、`git push --force-with-lease`、`git reset --hard`、`git commit -m "$(id)"`、`sudo ls`
- none: `git push origin --force`、`git push origin -f`、`git -C x reset --hard`、`sudo` + タブ + `ls`、`sudo`(単独)、`git commit -m 'ok'`、`ls`、`echo hi`

scratch の guard の複製での確かめ:

- `apt-get`、`flock`、`bash` を `noexec_list` に足した版: `apt-get remove sudo -y`、`apt-get install sudo vim`、`bash -c 'x' sudo ls`、`flock l git grep sudo file` が 4 件とも deny
- `apt-get` を `noexec_list` と `datacmd_list` の両方に足した版: `apt-get` の 2 件が none、`bash -c` と `flock` の 2 件は deny(この 2 つは足していない)

`tests/test-secret-scan.sh`:

- 単独で 123 passed、0 failed、約 25 秒
- 同時に 2 つ起動: 2 つとも rc 1、2 件ずつ失敗。落ちた検査は一方が `AC-15` と `AC-9`、他方が `AC-7` と `a failed driver listing names git's exit code`

---

## Cycle 2

- Date: 2026-10-08
- Plan: `docs/plans/active/2026-10-07-guard-deny-only.md`
- Pipeline cycle: 2(`cycle-count.json` が 2)。差分は merge base `f423f230` から branch HEAD `facd295b` まで。cycle 1 の sync-docs は `27581ff5`
- cycle 1 のあとの変更: `46806dc9`(cross-review の 3 件と consult の 3 件の修正)、`4e829e34`(F2-1 の修正)、テストの追加(`3e9afad6`、`180c7389`)、コメントの修正(`0ef6fc4f`、`63b6743a`、`c61ab2bf`、`4ffe74fe`)。guard の判定を最後に変えたのは `4e829e34`、コメントを最後に変えたのは `4ffe74fe`。コメントの行を除いたコードの行は、`4e829e34` と HEAD で同じ 1005 行だった(`git show` で取り出して比べた)
- 先行 report の cycle 2 の節(それぞれ「pipeline cycle 2」の節と、そのあとの再実行・やり直しの節):
  `docs/reports/self-review-2026-10-07-guard-deny-only.md`(merge。LOW)、
  `docs/reports/verify-2026-10-07-guard-deny-only.md`(pass。LOW。/sync-docs に渡す一覧 1〜13 は再実行の節にある)、
  `docs/reports/test-2026-10-07-guard-deny-only.md`(pass、1730/0、mutation 22 個のうち 19 個が赤、3 個は等価)、
  `docs/reports/cross-review-triage-guard-deny-only.md`

### Summary

46806dc9 と 4e829e34 で、tech-debt の guard の限界の行とテストの穴の行がまた古くなっていた。測った時点(095af1d7)、行数(1286 行)、データ区間が 1 つもなくなる場合、それに伴う誤検知、テストの件数(1576)が当たる。verify の再実行の一覧(1〜13)は、書く前に現在のコードとテストに当て直した。1〜12 は反映した。13(plan の Progress)は、一覧にあった 176 行目の範囲の書き方と 373fa29d・c61ab2bf の記録がすでに入っていたので、/test と /sync-docs の 2 行だけを足した。

一覧になかった直しが 1 件ある。tech-debt の 124・125 行目(解消済みの記録)が、推奨の HEREDOC の形は通り、残るのは awk がない環境だけ、と書いていた。4e829e34 のあとは、コマンドのどこかの行末に `\` がある場合も deny になるので、その例外を足した。

6 の「コメントのずれ」は、c61ab2bf と 4ffe74fe で直ったものを外した。V3-1(テストのコメント)、V3-2(END のコメントの折り返し)、R2-1〜R2-3、P2-1・P2-2 は現在のコードとテストを読んで、直っていることを確かめた。開いているのは 6 点で、新しい項目 (e) に書いた。

guard が ask を返すと書いた文書は、cycle 1 に続いて残っていない。

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行、160 行目) | 下の「guard の限界の行」。5 列のまま |
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 下の「テストの穴の行」 |
| `docs/tech-debt/README.md`(166 行目、`tests/test-secret-scan.sh` の固定パス) | 再実行で 4 回通ったことを 1 文足した。行は開いたまま |
| `docs/tech-debt/README.md`(124 行目、125 行目) | 解消済みの記録に、行末の `\` がある場合は deny になる、と足した(124 は英語、125 は行の言語に合わせて日本語) |
| `internal/org/prompts/implementer.md`(27 行目) | guard の説明を直した。下の「implementer.md」 |
| `docs/plans/active/2026-10-07-guard-deny-only.md` | `Progress checklist` の中だけ。/test のやり直しの行と /sync-docs の行を足した。digest は `7efd47f36781` のまま |
| `docs/insights/events/2026-10-08-guard-deny-only.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`) |
| `docs/reports/sync-docs-2026-10-07-guard-deny-only.md` | この節 |

### guard の限界の行(160 行目)

- 測った時点を 2026-10-08 の `facd295b` にし、「判定の最後の変更は `4e829e34`、コメントは `4ffe74fe`」と書いた。行番号を関数名で指す方針はそのまま
- 行数を測り直した。guard は 1372 行(`wc -l`)、awk のプログラムは 143 行目の `awk '` から 1329 行目の閉じる引用符までの 1187 行
- (a) に、データ区間が 1 つもなくなる 3 つの場合を足した: トップレベルのグループと複合コマンド(`NODATA`。`lex_cmds` と `end_cmd` が立てる)、リダイレクトのついた `exec`(`EXEC_SEEN`。`cmd_pos` が立てる)、コマンドのどこかにある `\` と改行(`END` の最初の数行。引用符を見ない)。理由は guard のヘッダーの方針の段落と合わせ、行き先を guard が見分けられないため、と書いた
- (a) に、それに伴う誤検知を足した。どれも旧版も deny にし、095af1d7 の guard は通していた(下の probe)。`(echo sudo ls)`、`if grep -q …`、`for` の中の `grep`、`(cd docs && grep -rn …)`、`exec >run.log; echo …`、`&>/dev/null`(self-review の P2-3)、行継続のあるすべての形。行継続は、ただの文字のものも含む: 単一引用符の中、ダブルクォートの中、引用符つきのヒアドキュメントの本文の行末、コメントの行末、`\\` のあと、行継続で複数行に書いたコマンド、行末が `\` の行がある推奨のコミット形。対照として、`cd docs && grep -rn …`、`exec echo …`、`>/dev/null 2>&1`、行末に `\` のない推奨の形が none になることも書いた。回避策は、行継続を使わずに書くか、コマンドを分けること。コミットメッセージは、`\` で終わる行を書き換えるか `git commit -F <file>`。deny の理由の文は HEREDOC を勧めるが、その形をすでに使っている、とも書いた
- (b) に、字句解析だけの穴を足した。`git commit -m "$\<改行>(id)"`、`echo "$\<改行>(git push origin --force)"`、区切りに引用符のないヒアドキュメントを `git commit -F -` に流す形の本文の同じ並び。`lex_dollar` が `$` の次の 1 文字だけで `(` と `{` を見るため。新版も旧版も none なので回帰ではない。shell は、このマシンの bash 3.2.57 と dash が二重引用符の形で置換を実行し、zsh 5.9 は文字のまま出した。ヒアドキュメントの形は 3 つとも実行した。見張りの文字列を中に持つ形(`(sudo ls)`、`(git push --force)`)は、行継続で区間がなくなるので deny
- (c) の probe の数を測り直した。awk を PATH から外した場合は 62 件、awk が 2 で終わる場合は 19 件、旧版と全件一致
- (d) の行数を直した
- (e) を新しく足した。開いているコメントのずれ 6 点: `cmd_pos` が立てる `EXEC_SEEN` の副作用を `cmd_pos` のコメントが書かない(P2-6 の 1)、awk に `'` を書けないという注意が `lex_dollar` のコメントにしかない(2)、`read_body` が行末のバックスラッシュを数えるループを 2 回書いている(3)、`new_ctx` の前の配列の一覧に `HPQ`・`HPN` がない(4)、テストのヘッダーの B にある形の種類が `guard_deny_only_forms` の足した形を挙げない(4)、`guard_deny_only_forms` の 1 のコメントが「false none before」と書く 2 件は表示するだけの形で、データ区間を与えない規則で deny になった(P2-4 のテストのコメント)
- Impact、Why deferred、Trigger、Related をそれに合わせた。Why deferred には、データ区間を与えない規則が AC7 に由来し、最後の周で止める側にだけ直したこと、行継続の規則を狭めるには `lex_dollar` の修正が先に要ることを書いた。Related に、self-review・verify・test の「pipeline cycle 2」の節とそのあとの再実行の節、triage の report を足した

### テストの穴の行(163 行目)

- 前書きに「pipeline cycle 2」を足した
- (c): 1576/0 を 1730/0 に直した(test report の cycle 2 の再実行。ubuntu の mawk 1.3.4 と gawk 5.2.1)。macOS の BWK awk でも、この sync-docs の中で `bash tests/test-pre-bash-guard.sh` を流して 1730 passed、0 failed、0 skipped を確かめた
- (a): Not pinned に、字句解析だけの穴(テストに例がない。`$\<改行>(id)` の形を `tests/test-pre-bash-guard.sh` で探して、なかった)を足した。46806dc9 と 4e829e34 で pin されたもの(`guard_deny_only_forms`、`edge_sentinel_deny` の文字だけの行継続、`edge_deny` の本文つなぎ)も足した。J02、L03、N02 は等価の mutation で、pin できないと書いた(tester が見つけた。`read_body` の本文ごとの規則と `HBSNL` は 4e829e34 のあと判定を変えず、対応のない `)` は `case` の中にしか出ず、`case` と `esac` が `NODATA` を立てる)。22 個のうち 19 個が赤になり、22 個は手で選んだもので、カバレッジは測っていない、とも書いた
- Related に、test report の cycle 2 の節(Mutation、Test gaps)と、verify の V3-1 を足した

### 124・125 行目と 166 行目

- 124 行目(RESOLVED の記録、英語): 「One remainder」を「Two remainders」にして、`4e829e34` 以降は行末に `\` がある行があると推奨の HEREDOC の形も deny になる、と足した
- 125 行目(RESOLVED の記録、日本語): 括弧の中の「awk がない環境の代替規則にだけ残る」を、awk がない環境の代替規則と、コマンドのどこかの行末に `\` がある場合、に直した
- 166 行目: `tests/test-secret-scan.sh` は、test report の cycle 2 の再実行が 4 回流した(`run-test.sh` 2 回、`run-verify.sh` 2 回。ほかの実行と重ならない)すべてで通った。重なりが原因だという診断に合うが、パスは直していないので行は開いたまま、と書いた

### implementer.md

`internal/org/prompts/implementer.md` の 27 行目は「このリポジトリの `pre_bash_guard.sh` フックはシェルレベルで `-m "$(` 形式を拒否するため、両エージェント共通で確実に使えるのは `-F` 形式」と書いていた。直した文は次のとおり。

- `pre_bash_guard.sh` フックは、ダブルクォートの `-m` の値に `$(` やバッククォートがあると拒否する
- ただし上の HEREDOC 形式は、コマンドのどの行もバックスラッシュで終わらなければ通る
- 行末にバックスラッシュがあると拒否されるので、その行を書き換えるか `-F <ファイル>` を使う

前後の文(HEREDOC と `-F` を両方挙げる文、「1 スライス = 1 コミット」)と、折り返しの幅(表示幅 77 以内)はそのまま。この prompt は go:embed で読まれるが、古い文を固定するテストはない(`internal/` と `tests/` で、古い文の一部と `pre_bash_guard` を `git grep` した。`internal/cli/migrate*.go` が hook のパスを持つだけ)。`go test ./internal/org/...` は 3 パッケージとも ok。

### verify の一覧との対応

| 一覧 | 状態 |
|------|------|
| 1 測った時点 | 反映(`facd295b`、判定は `4e829e34`、コメントは `4ffe74fe`) |
| 2 行数 | 反映(1372 行、awk は 143〜1329 行目の 1187 行) |
| 3 データ区間が 1 つもなくなる場合 | 反映((a)) |
| 4 誤検知の型 | 反映((a)。`&>/dev/null`、行継続の形、回避策を含む) |
| 5 字句解析だけの穴 | 反映((b)) |
| 6 コメントのずれ | 開いている 6 点だけを (e) に反映。c61ab2bf・4ffe74fe で直ったものは載せていない |
| 7 Related | 反映 |
| 8〜10 テストの穴の行 | 反映 |
| 11 implementer.md | 反映 |
| 12 ほかの文書 | 下の「Surfaces checked」。直したのは tech-debt の 124・125 行目だけ |
| 13 plan | 2 行を足した。176 行目の範囲の書き方と 373fa29d・c61ab2bf の記録は、すでに直っていた |

### Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| guard の挙動を書く文書(`docs/`(reports、plans、evidence、insights を除く)、`.claude/rules/`、`.claude/skills/`、`.claude/agents/`、`.agents/`、`.codex/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`templates/`、`internal/`) | `pre_bash_guard`、`git commit -m`・`-F`、HEREDOC、「guard が拒否・確認する」などで `git grep` した。間違った記述は `internal/org/prompts/implementer.md` の 1 か所だけで、直した。`internal/cli/migrate*.go` は hook のパスを持つだけ |
| `.claude/rules/ralph/git-commit-strategy.md`(root と template、53〜70 行目) | HEREDOC を推奨し、「`pre_bash_guard.sh` blocks dangerous patterns」とだけ書く。誤りはない。行末の `\` の例外は書いていない(下の「Found but left」) |
| `.codex/README.md`(root と template、114〜119 行目) | 「`deny` is the only decision the guard returns: it never asks for confirmation」。コードと一致。変更なし |
| guard のヘッダー(1〜120 行目) | 現在のコードと一致。方針の段落(98〜111 行目)が 3 つの場合、行継続の範囲、推奨の形が通る条件を書いている |
| tech-debt の guard を書く行 | 124・125 を直した。101・102・128・129・161・162 は RESOLVED の記録で、いまの挙動を書いていない |
| root と `templates/base/` | guard は `cmp` で同一。`check-sync.sh` は PASS |

### Found but left

- `git-commit-strategy.md`(root と template)の「Safe Quoting」に、行末が `\` の行があると推奨の HEREDOC の形も guard が deny にする、と 1 行足す案がある。記述は間違っていないので足していない。足すなら root と template を同じに保ち、`check-sync.sh` を通す。判断は orchestrator
- guard とテストのコメントの 6 点((e))は、guard のコードの変更になる(コメントだけでも pipeline が最初からやり直しになる)ので、触っていない
- tech-debt の 163 行目 (b) の「jq も awk もない経路」は 2026-10-07 の手での確かめのまま。この周では流していない

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-guard-deny-only.md` | `7efd47f36781`(編集の後。plan の Approved 行と一致) |
| tech-debt の 160、163、166 行目の列の数 | 5 列、行頭と行末が `\|` |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `./scripts/run-static-verify.sh` | rc 0(branch secret scan は `f423f230..facd295b` で clean。この commit は含まない) |
| `go test ./internal/org/...` | `internal/org`、`internal/org/driver`、`internal/org/protocol` が ok |
| `bash tests/test-pre-bash-guard.sh`(macOS、BWK awk) | PASS 1730、FAIL 0、SKIP 0(約 2 分) |

### Probe results

新しい guard は root の `.claude/hooks/pre_bash_guard.sh`、旧い guard は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`(origin/main の guard とバイト単位で一致)。比較用に、`git show` で取り出した 095af1d7 の guard(「095」)と、`4e829e34^` の guard も流した。決定は none か deny で、ask は 1 件も出なかった。「新」は jq あり/jq なしの順。

| コマンド | 新 | 旧 | 095 |
|---------|----|----|-----|
| `(echo sudo ls)` | deny/deny | deny | none |
| `if grep -q 'sudo ' file; then echo ok; fi` | deny/deny | deny | none |
| `for f in docs/*.md; do grep -n 'git push --force' "$f"; done` | deny/deny | deny | none |
| `(cd docs && grep -rn 'sudo ' .)` | deny/deny | deny | none |
| `cd docs && grep -rn 'sudo ' .`(対照) | none/none | deny | |
| `exec >run.log; echo 'sudo ls'` | deny/deny | deny | none |
| `exec echo 'sudo ls'`(対照)、`echo 'sudo ls'`(対照) | none/none | deny | |
| `echo 'sudo ls' &>/dev/null` | deny/deny | deny | none |
| `echo 'sudo ls' >/dev/null 2>&1`(対照) | none/none | deny | |
| 行継続: 単一引用符の中、二重引用符の中、引用符つきヒアドキュメントの本文の行末、コメントの行末、`\\` のあと、`echo 'sudo ls' \` の次の行に `\| cat`、`git commit -F -` の引用符つきヒアドキュメントの本文の行末 | deny/deny | deny | none |
| 推奨のコミット形(本文に `sudo ls` の語。行末が `\` の行あり)、同(`sudo` なし。行末が `\` の行あり) | deny/deny | deny | none |
| 推奨のコミット形(行末が `\` の行なし。本文に `sudo ls` の語あり、なし) | none/none | deny | |
| `git commit -F - <<'EOF'` の引用符つき本文(行末が `\` なし)(対照) | none/none | deny | |
| `git commit -F msg.txt`、`git commit -m 'feat: add $(date) stamp'` | none/none | none | |
| `git commit -m "feat: add $(date) stamp"`、同じバッククォート形 | deny/deny | deny | |
| `git commit -m "$\<改行>(id)"`、`echo "$\<改行>(git push origin --force)"`、`git commit -F - <<EOF` + `$\<改行>(id)` + `EOF` | none/none | none | none |
| `echo "$\<改行>(sudo ls)"`、`echo "$\<改行>(git push --force)"` | deny/deny | deny | none(4e829e34 の前) |
| `gh pr create --title t --body "$(cat <<'EOF'` … `git push --force` の語 … `EOF` `)"` | deny/deny | deny | |

1 行の形 62 件(`list1.txt`)は、tech-debt の行の (a)(b)(c) に挙げたコマンドの主なものを 1 件ずつ流した(cycle 1 の 107 件の全部ではない)。新しい guard の jq あり・なしの判定は 62 件で同じで、行の書き方と食い違う件はなかった。

awk なしの経路(`PATH` から awk を外し、jq は残した): 62 件すべてが旧版の判定と一致した。`sudo` + タブ + `ls` と `sudo` 単独は none、`git push origin --force` は none、`echo 'never use sudo here'`、`visudo -c`、`my-sudo ls` は deny、推奨の HEREDOC の形(`sudo` の語なし)は deny。awk を「stdin を読んで終了コード 2 で終わる」ものに差し替えた場合: 19 件すべてが旧版の判定と一致した。

shell での確かめ(`echo SUBST-RAN` と `echo HEREDOC-SUBST-RAN` に置き換えた安全な形):

- `echo "$\<改行>(echo SUBST-RAN)"`: bash 3.2.57 と `/bin/sh` と dash は `SUBST-RAN` を出し、zsh 5.9 は `$(echo SUBST-RAN)` を文字のまま出した
- `cat <<EOF` の本文の `$\<改行>(echo HEREDOC-SUBST-RAN)`: 4 つの shell とも `HEREDOC-SUBST-RAN` を出した

probe の入力は scratchpad の `sd3/`(`a01`〜`g01`、`list1.txt`、`list-c.txt`)に置いた。worktree の guard は変えていない。
