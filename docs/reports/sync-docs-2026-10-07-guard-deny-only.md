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

---

## Cycle 3 (cap raised to 3)

- Date: 2026-10-08
- Plan: `docs/plans/active/2026-10-07-guard-deny-only.md`
- Pipeline cycle: `cycle-count.json` は 2 のまま。cross-review の 2 周目のあと、ユーザーが `RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 3 に上げた 3 回目の run。差分は merge base `f423f230` から branch HEAD `e5c9e6be` まで。cycle 2 の sync-docs は `d5e197de`
- cycle 2 の sync-docs のあとの変更: `e1dfb422`(main の取り込み)、`a9ef82b1`(許可リスト)、`849f5411`(reset の `--pathspec-from-file` の値の読み飛ばし、理由の文とヘッダー)、`8e94e76b`(テストの追加)。guard の判定とコメントを最後に変えたのは `849f5411` で、`git diff 849f5411 HEAD` は `.claude/hooks/` と `templates/base/.claude/hooks/` に差分を出さない。root と template の guard は `cmp` で同一
- 先行 report の cycle 3 の節:
  `docs/reports/self-review-2026-10-07-guard-deny-only.md`(`48628bd2`。merge、MEDIUM 1・LOW 4、C3-1〜C3-5)、
  `docs/reports/verify-2026-10-07-guard-deny-only.md`(`92dc5df8`。pass、LOW 3、V4-1〜V4-3、/sync-docs に渡す一覧 1〜15)、
  `docs/reports/test-2026-10-07-guard-deny-only.md`(`e5c9e6be`。pass、1822/0、mutation 27 個のうち 24 個が赤)、
  `docs/reports/cross-review-triage-guard-deny-only.md`(cycle 2 が先頭、`a9af4e05`)

### Summary

a9ef82b1 の許可リストと 849f5411 で、推奨の HEREDOC のコミット形を説明する 4 か所が実際の判定とずれていた。`internal/org/prompts/implementer.md`、tech-debt の 124・125 行目、guard の限界の行は、通らない場合として行末の `\` だけを挙げていた。実際には、同じ Bash の呼び出しに読むだけのコマンドでも `git` でもないコマンドがあると、本文に見張りの語がなくても deny になる。この 4 か所と `git-commit-strategy.md`(root と template)を直した。

guard の限界の行は、測った時点、行数、データ区間が 1 つもなくなる場合(3 つから 5 つへ)、新しい誤検知、Why deferred の経緯を直した。テストの穴の行は、件数を 1822/0 に、等価・防御的な変異を AL02・DL02・DL05・N02〜N04 に直した。verify の一覧(1〜15)は、書く前にコードとテストに当て直した。一覧と違った点は 4 つある。

- (e) の既存の 6 点は、6 点ともまだ開いていた。`cmd_pos` のコメント、`'` を書けない注意の置き場所、`read_body` の同じループ 2 つ、`new_ctx` の前の配列の一覧、テストのヘッダーの B、`guard_deny_only_forms` の「false none before」を、コードとテストを読んで確かめた
- C3-5 の `tests/test-pre-bash-guard.sh` の変数の行(許可リストが `$c` も閉じると書く)は、8e94e76b が直後に足したコメントが説明しているので載せていない。残るのは `P2-4`・`P2-5` と「change A」の 3 か所だけ
- 一覧 8 の `exec echo 'sudo ls'` は、いまの行が「passes」と書いていた。probe で新版 deny、facd295b の guard は none、旧版 deny だった(下の「Probe results」)
- plan の Progress の 183 行目が「LOW 3」で終わっていたので、「件(V4-1〜V4-3 …)」までを補った

guard が ask を返すと書いた文書は、cycle 2 に続いて残っていない。

### Changes made

| File | Change |
|------|--------|
| `internal/org/prompts/implementer.md`(27〜33 行目) | HEREDOC の形が通る条件と、実際の運用を直した。下の「implementer.md と git-commit-strategy.md」 |
| `.claude/rules/ralph/git-commit-strategy.md`、`templates/base/.claude/rules/ralph/git-commit-strategy.md`(69 行目) | Rules に 1 行足した。root と template は `cmp` で同一 |
| `docs/tech-debt/README.md`(124 行目、125 行目) | 解消済みの記録に、つないだ呼び出しの deny を足した(124 は英語、125 は日本語) |
| `docs/tech-debt/README.md`(guard の限界の行、160 行目) | 下の「guard の限界の行」。5 列のまま |
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 下の「テストの穴の行」。5 列のまま |
| `docs/plans/active/2026-10-07-guard-deny-only.md` | `Progress checklist` の中だけ。/test と /sync-docs の 2 行を足し、183 行目の途切れを補った。digest は `7efd47f36781` のまま |
| `docs/insights/events/2026-10-08-guard-deny-only.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`。cycle は 2) |
| `docs/reports/sync-docs-2026-10-07-guard-deny-only.md` | この節 |

### implementer.md と git-commit-strategy.md

`internal/org/prompts/implementer.md` の guard の説明は、cycle 2 の「どの行もバックスラッシュで終わらなければ通る」から次の内容に変えた。

- HEREDOC の形は、同じ Bash 呼び出しのどのコマンドも、読むだけのコマンド(`echo` や `grep` など)か `git` で始まり、どの行もバックスラッシュで終わらないときだけ通る
- 実際には `git commit` を単独のコマンドで実行する
- 検証のコマンドにつなぐときと、行末にバックスラッシュがある行を書き換えられないときは、`git commit -F <ファイル>` を使う

折り返しの幅は、表示幅 72 以内(この段落の元の行は 76 以内)。go:embed で読まれる prompt だが、この文を固定するテストはない(`git grep` で古い文の一部と `pre_bash_guard` を探した。`internal/cli/migrate*.go` が hook のパスを持つだけ)。`go test ./internal/org/...` は 3 パッケージとも ok。

`git-commit-strategy.md` の Rules には、Single-line messages の次に 1 行を足した。`pre_bash_guard.sh` は、同じ Bash 呼び出しが `git` でも読むだけのコマンド(`echo` や `grep`)でもないコマンドを実行すると HEREDOC の形を deny にするので、そのあとは `git commit -F <file>` を使う、という内容。「同じ呼び出しに別のコマンドがあると deny」と書かず、`git add a.txt && git commit …` の形が通ることに合わせて条件を書いた(verify V4-1 の指示)。

### 124・125 行目

- 124 行目(RESOLVED の記録、英語): 「Two remainders」を「Three remainders」にして、3 つ目に、a9ef82b1 以降は同じ Bash 呼び出しの全トップレベルコマンドが読むだけのコマンドか `git` で始まらないと HEREDOC の形が deny になること(`make test &&` と `./scripts/run-verify.sh && git add a.txt &&` の後ろの形)を足した。単独の形と `git add a.txt &&` の後ろの形は通り、`git commit -F <file>` と引用符つきの heredoc を `-F -` に流す形はどのコマンドの後ろでも通る
- 125 行目(RESOLVED の記録、日本語): 括弧の中の「残るのは」に 3 つ目を足した(awk がない環境の代替規則、行末の `\`、a9ef82b1 以降のつないだ呼び出し)。回避は「`git commit` を単独で実行するか `-F <ファイル>`」

### guard の限界の行(160 行目)

- 測った時点を 2026-10-08 の `e5c9e6be` にし、判定とコメントの最後の変更は `849f5411` と書いた。行数を 1442 行にし、途中の変更に a9ef82b1 と 849f5411 を足した
- (a) に許可リストを足した。データ区間は、トップレベルのどの単純コマンドも、1 語目(引用符を外した値。代入と前置きを読み飛ばす前の語)が `/` を含まない `DATACMD` の名前か `git` のときだけ与える(`data_first_ok`、`end_cmd` から呼ぶ)。データ区間が 1 つもなくなる場合は 3 つから 5 つにした。新しい 2 つは、許可リストの外れ(代入、`env` などの前置き、道のある名前、変数、置換、リダイレクトだけのコマンド。`end_cmd` が `NODATA` を立てる)と、ヒアドキュメントの区切りの `$` かバッククォート(`lex_redir` が `NODATA` を立てる)。理由の文に、`env -S`・前置き・道のある名前、区切りの `$'\x45'` を足した
- (a) の誤検知に、`exec echo 'sudo ls'` が deny になったこと(facd295b では通った)を書き、許可リストの誤検知の段落を足した: 前置きつき(`env echo`、`command echo`、`nice grep`、`x=1 echo`、zsh の `=echo`)、道のある名前(`./echo`、`/tmp/x/cat`、`/bin/echo`)、非データコマンドと同じ呼び出しにある読むだけのコマンド(`make test && echo …`、`python3 x.py; grep …`。この 2 つは一覧になく、追加の probe で確かめた)、同じ呼び出しにある推奨のコミット形。対照として、`git status && echo …` と `cd docs && grep … && ls`、単独と `git add a.txt &&` の後ろの推奨の形、`-F <file>`、`-F - <<'EOF'`、単一引用符の `-m` が通ることを書いた
- 同じ段落に、self-review C3-1 が許可リストから推奨のコミット形だけを外さなかった理由(同じ呼び出しの前の方の定義が、メッセージの置換で走るものを変えうる)と、理由の文が実際の規則より狭いこと(V4-1。単独でなくても `git add a.txt &&` の後ろは通り、`-F -` の heredoc はどのコマンドの後ろでも通る)を書いた。回避は「`git commit` を単独の Bash コマンドで実行する、または検証につなぐなら `-F <file>`」
- (a) の「The same form with no such line passes」に、`git commit` が単独のときだけ、と足した
- (d) の行数を測り直した。guard は 1442 行、awk のプログラムは 154 行目の `awk '` から 1399 行目の閉じる引用符までの 1246 行
- (e) の書き出しを直し、既存の 6 点は e5c9e6be で確かめ直したと書いた。足した項目:
  - C3-3: 予約語の規則と `exec` の規則は、許可リストのあと判定を変えない(1 語目が DATACMD でも `git` でもないため。`cmd_pos` は 1 語目が DATACMD か `git` なら前置きをたどらず、`EXEC_SEEN` は 0 のまま)。N03・N04 は、507 行の probe で判定を 1 つも変えず、J02・L03・N02 と同じ等価な変異になる。`(` の規則と区切りの規則は、まだ単独で判定を決める。ヘッダーと `in_data` のコメントは 2 つを別の理由として並べている。cycle 2 の項目 1(`cmd_pos` の副作用)は、この 2 つが消えるまで意味を失う
  - C3-4: 区切りの規則のコメントが「字句解析は `$"..."`、`${...}`、置換を展開しない」とだけ書き、shell が展開するように読める。実際は `${x}` と `$x` の区切りを bash・zsh・dash が文字どおりに読む(この周で確かめた)。バッククォートの区切りは bash・zsh が文字どおり、dash が構文エラー(self-review)。違うのは `$'...'` の `\n`・`\t`・`\r` 以外のエスケープと `$"..."`。`data_first_ok` のコメントも、`$'...'` では「shell が見るとおりの値」と書けない
  - C3-5 の残り 3 か所: `edge_deny` の「Cycle 2 (P2-4)」、`edge_none` の「Cross-review cycle 2 (P2-4, P2-5)」(この report の self-review では P2-4・P2-5 は別の指摘で、triage では 4・5 番)、`edge_sentinel_deny` の「(change A, cycle 2)」(どの文書にも定義がない。許可リストのこと)。変数の行の説明は、8e94e76b のコメントが説明しているので外した
  - 既存の 5 番目(テストのヘッダーの B)に、`guard_deny_only_forms` の前の段落が 8 番で終わり、9 番(cross-review の 2 周目の形と許可リストの形)のコメントが配列の中にしかないことを足した
- Impact、Why deferred、Trigger、Related をそれに合わせた。Why deferred の (a) には、上限 3 の run で許可リストが条件の積み足しに置き換わったこと(許可リストが `env -S` と `builtin exec` を、区切りの規則が `$'\x45'` を閉じたこと)を書いた。(e) の理由に、上限 3 の run でも pipeline の回数が尽きたことを足した。Related に self-review・verify・test の「cycle 3 (cap raised to 3)」の節と triage の cycle 2 を足した
- 載せなかったもの: C3-1 の guard の側(理由の文とヘッダー)と C3-2(reset の値)は 849f5411 で直った(V4-2)

### テストの穴の行(163 行目)

- 前書きに「pipeline cycles 2 and 3」を足した
- (a): 849f5411・a9ef82b1・8e94e76b で pin されたもの(許可リスト、区切りの `$`、push の値を取るオプション、reset の `--` と `--pathspec-from-file`)を足した。27 個の mutation のうち 24 個が赤になり、3 個が生き残る(AL02 は等価、DL02・DL05 は防御的)。足す前は 11 個が緑で、8e94e76b の 11 行が閉じた。N03・N04 は N02 と同じ等価な変異になったので、等価な変異は J02・L03・N02・N03・N04・AL02。J02・L03・N02 の文の「all 1730 tests green」は「(the suite at facd295b)」を足して残した
- (a): 字句解析だけの穴(`$` と行継続と `(`)は、いまの guard、facd295b の guard、旧版のどれも none なので、テストはこの穴を none としてしか固定できない、と足した
- (c): 件数を直した。ubuntu:24.04 は mawk 1.3.4(jq あり)と gawk 5.2.1(jq あり)で 1822/0、mawk(jq なし)で 908/0(jq の経路の 903 件は skip)。ubuntu の数は test report の cycle 3 の節によるもので、この sync では流していない。macOS の BWK awk は、この sync の中で `bash tests/test-pre-bash-guard.sh` を流して 1822 passed、0 failed、0 skipped を確かめた(e5c9e6be)
- Related に test report の「cycle 3 (cap raised to 3)」の節(Mutation、Test gaps)を足した

### verify の一覧との対応

| 一覧 | 状態 |
|------|------|
| 1 implementer.md | 反映 |
| 2 tech-debt 124 行目 | 反映 |
| 3 tech-debt 125 行目 | 反映 |
| 4 git-commit-strategy.md | 反映(1 行、root と template は同一) |
| 5 測った時点 | 反映(`e5c9e6be`。判定とコメントは `849f5411`) |
| 6 行数 | 反映(1442 行、awk は 154〜1399 行目の 1246 行。`wc -l` と `grep` で測り直した) |
| 7 許可リスト | 反映((a)。`NODATA` を立てる場所に `lex_redir` も足した) |
| 8 新しい誤検知 | 反映。`exec echo` の文を直した。一覧にない `make test && echo …` と `python3 x.py; grep …` を probe して足した |
| 9 Why deferred の (a) | 反映 |
| 10 (e) の C3-3・C3-4 | 反映 |
| 11 (e) の C3-5 | 残りの 3 か所だけ反映(変数の行は 8e94e76b が説明している) |
| 12 C3-1 の guard の側と C3-2 | 載せていない。V4-1 は (a) に 1 文 |
| 13 Related | 反映 |
| 14 テストの数 | 反映(1822/0) |
| 15 C3-3 の等価な変異 | 反映 |

### Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| guard の挙動を書く文書(`docs/`(reports、plans、evidence、insights、tech-debt を除く)、`.claude/rules/`、`.claude/skills/`、`.claude/agents/`、`.agents/`、`.codex/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`templates/base/`、`internal/`、`packs/`) | `pre_bash_guard`、HEREDOC、`cat <<'EOF'`、`-F`、「単独のコマンド」、`deny-only`、「never asks for confirmation」で `git grep` した。間違った記述は `implementer.md` と `git-commit-strategy.md` の 2 か所で、直した。ほかの `cat <<'EOF'` は `internal/cli/org_test.go` と `internal/org/watcher_test.go` の shell の stub で、guard の説明ではない。`docs/specs/2026-04-16-ralph-cli-tool.md` の 1 行は hook のパスの一覧 |
| `.codex/README.md`(root と template、114〜119 行目) | 「`deny` is the only decision the guard returns: it never asks for confirmation」。コードと一致。変更なし |
| guard のヘッダー(1〜120 行目) | 現在のコードと一致。許可リストの段落(98〜120 行目)が 5 つの場合と、推奨の形が通る条件(同じ呼び出しのすべてのコマンドが読むだけのコマンドか `git` で始まり、どの行も `\` で終わらない)を書いている。コメントのずれは (e) に載せた |
| deny の理由の文(`commit_message`、`emit_deny`) | 単独のコマンドで打ったときだけ通る、と書く。規則より狭い(V4-1)ことを (a) に載せた |
| tech-debt の guard を書く行 | 124・125・160・163 を直した。101・102・128・129・161・162 は RESOLVED の記録で、いまの挙動を書いていない |
| root と `templates/base/` | guard、`lib_json.sh`、`git-commit-strategy.md` は `cmp` で同一。`check-sync.sh` は PASS |

### Found but left

- guard とテストのコメントのずれ((e) の C3-3〜C3-5 と既存の 6 点)は、guard やテストのコメントを変えると pipeline が最初からやり直しになる(上限 3 は尽きた)ので、触っていない
- `no_verify_rules` が、`--` で止まる前に値を取るオプションを読み飛ばさない点。verify の Coverage gaps は、コードを読んだだけで形を渡していないと書いている。この sync でも確かめていないので、tech-debt の行には載せていない(旧版に `--no-verify` の規則がなく、AC7 に関わらない)
- tech-debt の 163 行目 (b) の「jq も awk もない経路」は、2026-10-07 の手での確かめのまま。この周では流していない

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-guard-deny-only.md` | `7efd47f36781`(Progress checklist の編集の後。plan の Approved 行と一致) |
| tech-debt の 125、160、163 行目の列の数と、124 行目の `-->` の位置 | 5 列、行頭と行末が `\|`。124 行目は `-->` で終わり、足した文に `--` はない(元からある `--message` 1 か所だけ)。4 行とも `file:line` の参照はない(正規表現が拾う 163 行目の `ubuntu:24` は OS のイメージ名) |
| `./scripts/check-sync.sh` | rc 0、PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-skill-sync.sh` | rc 0、PASS(13 skill) |
| `./scripts/run-static-verify.sh` | rc 0(shellcheck、全 hook の `sh -n`、`settings.json` の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint 0 issues、branch secret scan は `51855166..e5c9e6be` で clean。この commit は含まない) |
| `go test ./internal/org/...` | `internal/org`、`internal/org/driver`、`internal/org/protocol` が ok |
| `bash tests/test-pre-bash-guard.sh`(macOS、BWK awk) | PASS 1822、FAIL 0、SKIP 0 |

### Probe results

新しい guard は worktree の `.claude/hooks/pre_bash_guard.sh`(e5c9e6be)、「前」は `git show facd295b:` で取り出した cycle 2 の sync-docs 時点の guard、「旧」は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`(`origin/main` の guard とバイト単位で一致することを `cmp` で確かめた)。「新」は jq あり/jq なしの順で、どの行も 2 つの経路で同じだった。決定は none か deny で、ask と想定外の出力は 1 件もなかった。79 件を流し、下の表は同じ結果の行をまとめた。

| コマンド | 新 | 前 | 旧 |
|---------|----|----|----|
| `env echo 'sudo ls'`、`command echo 'sudo ls'`、`nice grep 'sudo ' f`、`x=1 echo 'sudo ls'`、`=echo sudo ls` | deny | none | deny |
| `./echo 'sudo ls'`、`/tmp/x/cat 'sudo ls'`、`/bin/echo 'sudo ls'`、`exec echo 'sudo ls'` | deny | none | deny |
| `make test && echo 'sudo ls'`、`python3 x.py; grep -n 'sudo ' f` | deny | none | deny |
| `echo 'sudo ls'`、`grep -n 'git push --force' docs.md`、`echo hi && cd docs && grep -rn 'sudo ' .`、`git status && echo 'sudo ls'`、`cd docs && grep -n 'sudo ' f && ls`(対照) | none | none | deny |
| 推奨のコミット形(単独。本文に `sudo ls` と `git push --force` の語)、`git add a.txt &&` の後ろ | none | none | deny |
| 推奨のコミット形の `make test &&` の後ろ、`./scripts/run-verify.sh && git add a.txt &&` の後ろ、`GIT_EDITOR=true git commit`、後ろに `&& ./scripts/secret-scan-branch.sh --strict` | deny | none | deny |
| `make test && git commit -F - <<'EOF'`(引用符つき)、`make test && git commit -F msg.txt`、`make test && git commit -m 'feat: add a thing'` | none | none | none |
| 推奨の形で本文に行末が `\` の行、`git commit -m "feat: add $(date) stamp"` | deny | deny | deny |
| ヒアドキュメントの区切りが `"$X"`、バッククォート、`$'\x45'`(本文に `sudo ls`) | deny | none | deny |
| 区切りが `'EOF'`(対照。本文に `sudo ls`) | none | none | deny |
| `x=1; echo 'sudo ls'`、`echo hi; env FOO=1 true; echo 'sudo ls'`、`>out.txt; echo 'sudo ls'`、`$c; echo 'sudo ls'` | deny | none | deny |
| `$'echo' 'sudo ls'`、`$"echo" 'sudo ls'`、`"echo" 'sudo ls'` | none | none | deny |
| `$c 'sudo ls'` | deny | deny | deny |
| `env -S 'sh -c "eval \$2" --' echo 'sudo ls'`、`builtin exec >run.sh; echo 'sudo ls'; sh run.sh` | deny | none | deny |
| `git commit -m "$\<改行>(id)"`、`echo "$\<改行>(git push origin --force)"`(字句解析だけの穴) | none | none | none |
| `git push origin -ofoo`、`git reset HEAD -- --hard`、`git reset --pathspec-from-file=f -- --hard` | none | deny | none |
| `git push -ofoo --force origin`、`git reset --pathspec-from-file -- --hard` | deny | deny | none |
| tech-debt の 160 行目 (a) が deny と書く形(`apt-get remove sudo -y`、`apt-get install sudo vim`、`bash -c 'x' sudo ls`、`flock l git grep sudo file`、`cp sudo dest`、`git grep sudo file`、`git push --force-if-includes origin main`、`echo 'sudo ls' \| sed 's/x/y/'`、`grep -r 'sudo ' . \| sort`、`(echo sudo ls)`、`if grep …`、`for … grep …`、`(cd docs && grep …)`、`exec >run.log; echo 'sudo ls'`、`echo 'sudo ls' &>/dev/null`、`echo 'sudo ls \<改行>x'`、`gh pr create --body` の heredoc、`ssh host 'sudo ls'`) | deny | deny | deny |
| 同じ行が通ると書く形(`git push origin --force-if-includes`、`apt-get install sudo`、`echo 'sudo ls' \| cat`、`grep -r 'sudo ' . \| grep x`、`cd docs && grep -rn 'sudo ' .`、`echo 'sudo ls' >/dev/null 2>&1`、`git commit -F msg.txt`、`ssh host 'git push origin --force'`) | none | none | 下の注 |
| `my-sudo ls`、`x.sudo ls`、`visudo -c` | none | none | deny |

「同じ行が通ると書く形」の旧の列は、`git push origin --force-if-includes`、`apt-get install sudo`、`git commit -F msg.txt`、`ssh host 'git push origin --force'` が none、残りの 4 件(`echo … \| cat`、`grep … \| grep x`、`cd docs && grep …`、`>/dev/null 2>&1`)が deny だった。

shell での確かめ(`${x}` と `$x` の区切り、`$'E\x4e\x44'` の区切りを使い、本文に `END` の行と実行すると「command not found」になる行を入れた安全な形): `${x}` と `$x` の区切りは、bash・zsh・dash とも文字どおりに読み、本文は区切りの行まで続いた。`$'E\x4e\x44'` は bash と zsh が `END` と読み、本文を `END` の行で終えて、次の `END` の行をコマンドとして実行した(command not found)。dash は `$'…'` を戻さず、本文がファイルの終わりまで続いた。

probe の入力は scratchpad の `sd4/`(`cases-a.txt`〜`cases-d.txt`、`delim.sh`、`run3.sh`)に置いた。worktree の guard は変えていない。

## Cycle 4 (cap raised to 4)

- Date: 2026-10-08
- Plan: `docs/plans/active/2026-10-07-guard-deny-only.md`
- Pipeline cycle: `cycle-count.json` は 2 のまま。cross-review の 3 周目のあと、ユーザーが `RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 4 に上げた 4 回目の run で、ユーザーはこれを最後の run とした。差分は cycle 3 の sync-docs `24876df7` から branch HEAD `9e32db94` まで
- cycle 3 の sync-docs のあとの変更: `af7d5987`(triage の cycle 3)、`b3c3fdaa`(printf、`test`・`[`、`--no-verify` の走査、commit の値を取るオプション)、`777f923d`(self-review)、`12e9a9ad`(printf と rg を書かれたままの語で見る、`opt_is` の下限、`pull`)、`0333603d`(verify)、`6b1f7acc`(ヘッダーのコメント)、`389436de`(テスト 7 行)、`9e32db94`(test)。guard の判定を最後に変えたのは `12e9a9ad`。`git diff 12e9a9ad 6b1f7acc -- .claude/hooks/pre_bash_guard.sh` で `#` で始まらない行は 0 本。root と template の guard は `cmp` で同一
- 先行 report の cycle 4 の節:
  `docs/reports/self-review-2026-10-07-guard-deny-only.md`(`777f923d`。no-merge、HIGH 1・MEDIUM 1・LOW 2、C4-1〜C4-4。4 つとも `12e9a9ad` で直った)、
  `docs/reports/verify-2026-10-07-guard-deny-only.md`(`0333603d`。pass、LOW 3、V5-1〜V5-3、/sync-docs に渡す一覧 1〜15)、
  `docs/reports/test-2026-10-07-guard-deny-only.md`(`9e32db94`。pass、1886/0、mutation 20 個すべてが赤)、
  `docs/reports/cross-review-triage-guard-deny-only.md`(cycle 3 が先頭、cycle 2 と cycle 1 は付録)

### Summary

b3c3fdaa と 12e9a9ad で guard の判定が変わった。tech-debt の guard の限界の行(160 行目)は、printf を `-v` だけで見て、`test` と `[` をデータコマンドに数え、`--no-verify` の規則に触れていなかった。この行とテストの穴の行(163 行目)を直した。ほかの文書(`.claude/rules/`、`.claude/skills/`、`AGENTS.md`、`README.md`、`.codex/README.md`、`templates/base/`、`internal/`)に、この 2 つの変更で間違いになった記述はなかった(下の「Surfaces checked for drift」)。

verify の一覧(1〜15)は、書く前にコード、テスト、probe に当て直した。一覧と違った点が 5 つある。

- 行数: 一覧は 1458 行、awk は 156〜1415 行目(12e9a9ad の値)。6b1f7acc がヘッダーのコメントを 3 行足したので、いまの guard は 1461 行、awk は 159〜1418 行目(どちらも 1260 行)。行には両方を書いた
- V5-1(一覧 7): 6b1f7acc が 3 点とも直していた。ヘッダーの `pull`(53 行目)、略記の下限(「at least one more character」、`--h` の例)、(a) の printf・rg・`test`・`[` の条件、`no_verify_rules` のコメント、テストの略記の行の前のコメントを、guard とテストを読んで確かめた。残るのは、ヘッダーの Not covered が printf と rg は書かれたままの語を見ることに触れていない点だけで、(e) に 1 文で足した
- 一覧 12(V5-2 を 163 行目に足す): 389436de が固定したので足していない。行には元から V5-2 の記述がなく、消す対象もなかった。代わりに、固定されていない rg の変数の語(`rg $x sudo pat .`)を足した
- rg の変数の語を、160 行目の (b) に足し、Impact と Trigger にも書いた。一覧にはない。verify の cycle 4 の節は、rg の検査の 3 項目(「b3c3fdaa と 12e9a9ad の確かめ」の 3)で、`$x` の差を plan の Non-goals の範囲の設計の選び方として述べている。probe で形を確かめた(下の「Probe results」)
- (e) に、テストの `guard_deny_only_forms` の前の段落が 8 番で終わっているのに加え、配列の中に 9 番が 2 つ(cross-review の 2 周目の形と 3 周目の形)あり、番号のないコメントが 2 つ(self-review C4-1、cycle 4 の /test)あることを足した。コードを読んで見つけた

guard が ask を返すと書いた文書は、cycle 3 に続いて残っていない。

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行、160 行目) | 下の「guard の限界の行」。5 列のまま、バッククォートの数は各列で偶数 |
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 下の「テストの穴の行」。5 列のまま |
| `docs/plans/active/2026-10-07-guard-deny-only.md` | `Progress checklist` の中だけ。/test の 4 回目と /sync-docs の 4 回目の 2 行を足した。digest は `7efd47f36781` のまま |
| `docs/insights/events/2026-10-08-guard-deny-only.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`。cycle は 2) |
| `docs/reports/sync-docs-2026-10-07-guard-deny-only.md` | この節 |

### guard の限界の行(160 行目)

- 測った時点: 書き出しを「e5c9e6be で測った。ただし cap-4 の形(printf、rg、`test`、`[`、`--no-verify`)は 6b1f7acc で測り、(a)・(b) に判定が書いてある 29 形は 6b1f7acc で測り直した」にした。6b1f7acc の判定は 12e9a9ad と同じ。行数を 1461 行(6b1f7acc。12e9a9ad は 1458 行)にし、途中の変更に b3c3fdaa、12e9a9ad、6b1f7acc を足した
- (a) のデータ区間の説明: `DATACMD` の括弧に「`test` と `[` は入らない。`printf` と `rg` は条件つき」を足した。許可リストの文のあとに、3 つの条件の段落を足した。printf は、どの語も `-v` で始まらず、書かれたままの語に `%`・`$`・バッククォートがないときだけ(zsh の `%n` と `%d` などの数値の変換は引数を算術式として評価し、bash と zsh の `printf -v` は変数に書く)。rg は、`--pre` で始まる語がなく、`$'…'`・`$"…"` の語がないときだけ。`test` と `[` は `DATACMD` に入らない(zsh と bash 5 の `-v` が添字を評価する)。zsh と bash での実行は cross-review の 3 周目と self-review C4-1 の確かめで、この sync では繰り返していないと書いた。printf と rg が書かれたままの語を見る理由(字句解析は `$'…'` の `\n`・`\t`・`\r` しか戻さない。`$'\x25n'` は zsh には `%n`、字句解析には `x25n`)も同じ段落に書いた
- (a) の誤検知: cap-4 の条件の誤検知の段落を足した。`printf '%s\n' 'sudo ls'`、`printf` の後ろにバッククォートや `$x`、`[ -n 'sudo ls' ]`、`test -n 'sudo ls'`、`test -n 'git push --force'`、`rg $"sudo ls" .`。旧版も deny、e5c9e6be は none。回避は `echo` や `cat` で表示する、printf の書式から `%` を外す。`printf -v c 'sudo ls'` は e5c9e6be でも deny
- (a) の `--no-verify` の規則: 旧版にない規則なので新しい deny であり、後退ではないと書いた。読むサブコマンドは commit、push、merge、rebase、am、pull。merge・rebase・am・pull の走査は `--` で止まらない(`git merge -m -- --no-verify feature` と `git rebase --onto -- --no-verify main` は deny。前者は e5c9e6be で none)ので、メッセージが `--no-verify` という文字の `git merge -m --no-verify feature` も deny。`commit_rules` は `--` で止まるが、値を取る長いオプション(10 個)を略記でも値ごと読む(`git commit --trail -- --no-verify -m fix` は deny。e5c9e6be は none)。`opt_is` は `--` のあと 1 文字から前置きとして読むので、git が曖昧として断る略記(`git push --f origin main`、`git commit --n -m x`、`git merge --n feature`、`git pull --n origin main`)も deny。git 2.49.0 は 4 つとも rc 129 の `ambiguous option` で止める。`git reset --h` は本物の deny(git 2.49.0 は hard reset をした)。通る形は `git merge --no-ff -- feature`、`git pull --no-rebase origin main`、`git push --follow-tags origin main`。git が 2.49.0 でない場合は確かめていない
- (b): hex や octal のエスケープの文のあとに、printf と rg は 12e9a9ad から書かれたままの語を見る、と足した。`rg $'\x2d-pre' sh 'sudo ls'`、`printf $'\x25n' $'arr[\x24(sudo id; echo 1)]'`、`printf $'\x2dv' c 'sudo ls'` は deny(e5c9e6be は none、旧版は deny)。残る穴として、rg の変数の語を書いた。`rg $x sudo pat .` は none(旧版は deny)。rg は `$'…'`・`$"…"` の語では読むだけから外れるが、`$x` では外れない。`x` が呼び出しの外で `--pre` になっていれば、rg は次の語を各ファイルへのプログラムとして走らせる(標準のプログラムの代わりに引数を記録する stand-in で、`x=--pre; rg $x ./prog.sh HIT file` が走ることを確かめた)。同じ呼び出しで代入すると(`x=--pre; rg $x sudo pat .`、`export x=--pre && rg $x sudo pat .`)、1 語目が代入や `export` なので deny になる。self-review C4-1 が勧めた「`$` を含む語はすべて外す」を 12e9a9ad が `$'` と `$"` に狭めたこと、判断は plan の Non-goals の範囲として残ったことも書いた。printf には同じ穴がない(`$` を含む語があれば外れる)
- (d): 行数を 1461 行(6b1f7acc。12e9a9ad は 1458、e5c9e6be は 1442)に、awk のプログラムを 1260 行(6b1f7acc は 159〜1418 行目、12e9a9ad は 156〜1415 行目、e5c9e6be は 1246 行)に直した
- (e): 書き出しを 6b1f7acc に直し、既存の項目は 6b1f7acc で確かめ直したと書いた(等価な変異の件だけは cycle 3 の test report による)。`guard_deny_only_forms` の前の段落が 8 番で止まる件に、9 番が 2 つあることと、番号のないコメントが 2 つあることを足した。V5-1 は 6b1f7acc で直ったこと、残る 1 点(Not covered の文)、plan の Scope の 3 行(承認 digest の範囲なので触らず、逸脱は Progress)を書いた
- Impact: (a) に printf・`test`・`[`・rg の新しい誤検知と `--no-verify` の規則の止めすぎ、(b) に rg の変数の語を足した
- Why deferred: 「The cap-4 run went the same way」を足した。cross-review の 3 周目の 2 件、上限を 4 に上げたこと、b3c3fdaa、self-review の C4-1・C4-2 と 12e9a9ad、止める側にだけ動いた根拠(self-review と verify の比較。旧版が deny にして新版が通す行は `intentional_fixes` の 13 行のまま、12e9a9ad は deny から none に変えた行がない)、mutation 20/20(8 個は足す前に緑)、ユーザーが最後の run としたこと。(e) の理由に、cap-4 の run でも pipeline の回数が尽きたことと、self-review の cycle 4 の Tech debt の行は載せないこと(C4-1〜C4-4 は 12e9a9ad、コメントのずれは 6b1f7acc で直った)を足した
- Trigger: (a) に、printf・rg・`test`・`[` の条件を緩めるのは設計の判断で、回帰の守りは section B の行と `edge_sentinel_deny` の 6 行であること、`--no-verify` の規則の止めすぎを狭める方向(merge・rebase・am・pull の値を取るオプションを `commit_rules` と同じに読む)を足した。(b) に rg の変数の語の直し方(`stage_note` の次の変更で `$` を含む語は rg を外す。self-review C4-1 の勧め。rg についての影響は report に測った記録がない)を足した
- Related: self-review・verify・test の「cycle 4 (cap raised to 4)」の節(self-review: C4-1〜C4-4、Tech debt identified。verify: V5-1〜V5-3、/sync-docs に渡す一覧。test: Mutation、Test gaps)と triage の cycle 3 を足した。cycle 2 の分類は、cycle 3 の追加で先頭から付録(「付録: cycle 2」)に移っていたので、書き方を直した

### テストの穴の行(163 行目)

- 前書きを「pipeline cycles 2 to 4」にした
- (a): b3c3fdaa・12e9a9ad・389436de で固定されたもの(printf の `%`・`$`・バッククォート・`-v` と `$'\x25n'`・`$'\x2dv'` の行、rg の `--pre`・`$'…'`・`$"…"`、`test -v` と `[ -v`、`test`・`[`・`%` の誤検知、merge・rebase の `--` を越える `--no-verify` と commit の値を取る長いオプション、1 文字の略記と `pull`)を足した。cycle 4 の /test の mutation は 20 個で、内訳は printf 6、rg 4、`DATACMD` 3、`no_verify_rules` 1、commit 3、`opt_is` 2、`pull` 1。足す前は 8 個が緑で、7 行が閉じ、いまは 20 個すべてが赤、等価な変異はない
- (a): 固定されていないものに、`rg $x sudo pat .` を足した(test report、cycle 4、Test gaps)
- (c): 件数を test report の cycle 4 の節に直した。ubuntu:24.04 は mawk 1.3.4(jq あり)と gawk 5.2.1(jq あり)で 1886/0、mawk(jq なし)で 940/0(jq の経路の 935 件は skip)。macOS の BWK awk は、この sync の中で `bash tests/test-pre-bash-guard.sh` を流して 1886 passed、0 failed、0 skipped を確かめた(9e32db94)
- Why deferred: 固定されていない限界に、rg の変数の語が cap-4 の run で加わり、最後の /test もこれを固定していないと足した
- Related: test report の「cycle 4 (cap raised to 4)」の節(Mutation、Test gaps)と verify の V5-2(389436de が固定)を足した

### verify の一覧との対応

| 一覧 | 状態 |
|------|------|
| 1 測った時点 | 反映(12e9a9ad が最後の判定。e5c9e6be で測った形と 6b1f7acc で測り直した形を書き分けた) |
| 2 行数 | 反映(1461 行、awk は 159〜1418 行目の 1260 行。12e9a9ad の 1458 行、156〜1415 行目も併記。`wc -l` と `grep` で測り直した) |
| 3 データ区間の条件 | 反映(printf、rg、`test`、`[`) |
| 4 誤検知 | 反映(一覧の 3 形に、`rg $"…"`、printf のバッククォートと `$x` を足した) |
| 5 `--no-verify` の止めすぎ | 反映(ほかに、git が 2.49.0 で rc 129 にする略記 4 形と `git reset --h` を probe して書いた) |
| 6 (b) の書かれたままの語 | 反映(printf と rg。変数の語の穴も) |
| 7 (e) の V5-1 | V5-1 は 6b1f7acc で直っていた。残る 1 点だけ反映。9 番が 2 つあることも足した |
| 8 self-review の Tech debt の行 | 載せていない(C4-1〜C4-4 は直った) |
| 9 Why deferred | 反映 |
| 10 Related | 反映(triage の cycle 2 の場所も直した) |
| 11 163 行目の固定したもの | 反映 |
| 12 163 行目の V5-2 | 足していない(389436de が固定。行に記述がなかった)。rg の変数の語を足した |
| 13 件数 | 反映(1886/0、940/0、mutation 20/20。plan の Progress の 188 行目の 1868 件は 12e9a9ad の時点の件数なので残した) |
| 14 ほかの文書 | 直すところなし(下の「Surfaces checked for drift」で `git grep` した) |
| 15 PR 本文の既知の穴 | この sync の範囲外(/pr)。V5-1 の残り 1 点、V5-2(固定済み)、C3-3〜C3-5 の材料は 160・163 行目に載せた |

### Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| guard の挙動を書く文書(`docs/`(reports、plans、evidence、insights、tech-debt を除く)、`.claude/rules/`、`.claude/skills/`、`.claude/agents/`、`.agents/`、`.codex/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`templates/base/`、`internal/`、`packs/`) | `pre_bash_guard`、`pre-bash-guard`、`DATACMD`、`datacmd_list`、「data command」、「at least 4」「4 characters」、`test`・`[` を並べた書き方、「commit, push, merge, rebase」「merge, rebase and am」、`printf without -v`、`rg without --pre` で `git grep`。間違った記述は見つからなかった。`git-commit-strategy.md`(root と template の 69 行目)と `implementer.md` は「`git` でも読むだけのコマンド(`echo` や `grep`)でもないコマンド」と書き、`echo` と `grep` は `DATACMD` にいまも入っているので変更なし。`.codex/README.md`(root と template、114〜119 行目)の「`deny` is the only decision the guard returns」はコードと一致。`internal/cli/migrate*.go` は hook のパスを持つだけ |
| guard のヘッダー(1〜136 行目) | 6b1f7acc のあと、コードと一致(`--no-verify` のサブコマンドに `pull`、略記の下限、(a) の条件)。残るコメントのずれは (e) |
| `tests/test-pre-bash-guard.sh` のコメント | 6b1f7acc が略記の行の前のコメントを直した。`edge_deny` の `--no-verify` の行の前のコメントは「merge, rebase and am」と書くが、cycle 3 の cross-review の形についての説明で、次のコメントが `pull` に触れている。9 番が 2 つある点は (e) |
| plan の Scope(36〜46 行目) | `--no-verify` のサブコマンド(`pull` なし)、略記(4 文字以上)、データコマンド(`test`・`[` あり、printf に条件なし)は変更前のまま。承認 digest の範囲なので触らず、逸脱は Progress に書いてある |
| tech-debt の guard を書く行 | 160・163 行目を直した。124・125 行目は RESOLVED の記録で、つないだ呼び出しの deny は変わっていない。101・102・128・129・161・162 は RESOLVED の記録で、いまの挙動を書いていない |
| root と `templates/base/` | guard、`lib_json.sh`、`git-commit-strategy.md` は `cmp` で同一。`check-sync.sh` は PASS |

### Found but left

- guard とテストのコメントのずれ((e) の C3-3〜C3-5、既存の 6 点、V5-1 の残り 1 点)は、guard やテストのコメントを変えると pipeline が最初からやり直しになる(上限 4 は尽き、ユーザーがこの run を最後とした)ので、触っていない
- rg の変数の語の穴は、tech-debt に載せただけで、guard は変えていない
- plan の Progress に verify(`0333603d`)の行がない。依頼は /test と /sync-docs の 2 行だったので足していない。V5-3 は、Progress の記録で足りるかを orchestrator が決める、としている
- git 2.49.0 以外での略記の読み方、mawk・gawk 以外の awk(busybox)での判定は、test report と同じく流していない
- tech-debt の 163 行目 (b) の「jq も awk もない経路」は、2026-10-07 の手での確かめのまま。この周でも流していない
- zsh と bash 5 での `printf %n`・`printf -v`・`test -v` の実行は、cross-review の 3 周目と self-review の確かめによる。この sync では繰り返していない

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-guard-deny-only.md` | `7efd47f36781`(Progress checklist の編集のあと。plan の Approved 行と一致) |
| tech-debt の 160・163 行目の列の数と、バッククォートの数 | 5 列、各列で偶数。`|` を含む 22・32 行目が 5 列にならないのは、この sync の前の HEAD でも同じ(検査の正規表現の限界) |
| tech-debt の `docs/plans/(active\|archive)/…` の参照 | すべて存在する(`run-static-verify.sh` の `tech-debt README plan references` も OK) |
| `./scripts/check-sync.sh` | rc 0、PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0、PASS(13 skill) |
| `./scripts/run-static-verify.sh` | rc 0(check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint 0 issues、branch secret scan は `51855166..9e32db94` で clean。この commit は含まない) |
| `bash tests/test-pre-bash-guard.sh`(macOS、BWK awk) | PASS 1886、FAIL 0、SKIP 0 |

### Probe results

新しい guard は worktree の `.claude/hooks/pre_bash_guard.sh`(6b1f7acc)、「e5c9」は `git show e5c9e6be:` で取り出した cycle 3 の sync-docs 時点の guard、「旧」は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`(`origin/main` の guard とバイト単位で一致する)。「新」は jq あり/jq なしの順で、32 形のどれも 2 つの経路で同じだった。決定は none か deny で、ask と想定外の出力は 1 件もなかった。下の表は同じ結果の形をまとめた。

| コマンド | 新 | e5c9 | 旧 |
|---------|----|------|----|
| `printf '%s\n' 'sudo ls'`、`` printf `echo x` 'sudo ls' ``、`printf "$x" 'sudo ls'`、`[ -n 'sudo ls' ]`、`test -n 'sudo ls'`、`test -n 'git push --force'` | deny | none | deny |
| `rg $"sudo ls" .` | deny | none | deny |
| `printf -v c 'sudo ls'` | deny | deny | deny |
| `rg $'\x2d-pre' sh 'sudo ls'`、`printf $'\x25n' $'arr[\x24(sudo id; echo 1)]'`、`printf $'\x2dv' c 'sudo ls'` | deny | none | deny |
| `printf 'a\ngit push --force'`、`rg 'sudo ls' .`(対照) | none | none | deny |
| `rg --pre 'sudo ls' x .` | deny | deny | deny |
| `rg $x 'sudo ls' .`、`rg $x sudo pat .` | none | none | deny |
| `x=--pre; rg $x sudo pat .`、`export x=--pre && rg $x sudo pat .` | deny | deny | deny |
| `git merge -m --no-verify feature` | deny | deny | none |
| `git merge -m -- --no-verify feature`、`git commit --trail -- --no-verify -m fix`、`git rebase --onto -- --no-verify main` | deny | none | none |
| `git push --f origin main`、`git commit --n -m x`、`git merge --n feature`、`git pull --n origin main` | deny | none | none |
| `git reset --h`、`git commit --m -- --no-verify`、`git pull --no-verify origin main` | deny | none | none |
| `git merge --no-ff -- feature`、`git pull --no-rebase origin main`、`git push --follow-tags origin main` | none | none | none |

tech-debt の 160 行目が判定を書いている 29 形(verify の `vf6/docforms.sh`)を、e5c9e6be、b3c3fdaa、HEAD、旧版に渡し直した。HEAD の判定は、29 形とも行の記述のとおりだった(通ると書いた 15 形は none、deny と書いた 12 形は deny、判定なしと書いた 2 形は none)。

git 2.49.0 の使い捨てのリポジトリ(remote はローカルの bare)で、略記を実行した。

| コマンド | 結果 |
|---------|------|
| `git push --f origin master` | rc 129。`ambiguous option: f (could be --force-if-includes or --follow-tags)` |
| `git commit --n -m x` | rc 129。`ambiguous option: n (could be --no-allow-empty or --no-allow-empty-message)` |
| `git merge --n feature` | rc 129。`ambiguous option: n (could be --no-signoff or --no-verify)` |
| `git pull --n origin master` | rc 129。`ambiguous option: n` |
| `git push --n origin master`、`git rebase --n master`、`git am --n x.patch` | rc 129。`ambiguous option: n` |
| `git reset --h`(`f` を書き換えたあと) | rc 0。`HEAD is now at …`。`f` は元の内容に戻った(hard reset) |

rg の `--pre` を変数の語で渡す確かめ(`x=--pre; rg $x ./prog.sh HIT target.txt`。`prog.sh` は引数をログに書いて `cat` する stand-in): rg は `HIT line` を出し、ログに `prog ran with: …/target.txt` が残った。つまり変数の語が `--pre` として読まれ、次の語がプログラムとして走った。

probe の入力は scratchpad の `sd5/`(`p01.txt`〜`p15.txt`、`g01.txt`〜`g13.txt`、`q01.txt`〜`q04.txt`、`gitabbrev.sh`、`rgvar.sh`、`hist.sh`)に置いた。worktree の guard は変えていない。
