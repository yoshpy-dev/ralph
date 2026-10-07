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
