# Verify report: guard-awk-split

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-awk-split.md(`- Approved:` の digest cfb34e566022 は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff 0abfede5..HEAD`(base 0abfede5、HEAD 6dab8a5d、16 ファイル、+3320/-2853)。コードの変更は S1 の da3b55d2 だけで、7347fa6a・f593bbfc・c3a323aa・8c61c6cb は guard のコメント、dfb8785a は `check-template.sh` とそのテスト、2196a341 は tech-debt。テストスイートは実行していない(AC3 と AC5 の実行・mutation は /test が見る)。probe には plan とテストの配列にある形だけを使った
- Evidence: `docs/evidence/verify-2026-10-09-guard-awk-split.log`(gitignore の対象なのでコミットしない。スクリプトは `scratchpad/vrf/`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 4 ファイルとも 800 行未満、`.sh` に awk のプログラムが残らない | Met | `wc -l`: `pre_bash_guard.sh` 231、`_lex.awk` 616、`_commands.awk` 336、`_rules.awk` 465(template の写しも同じ)。`grep -c "LC_ALL=C awk '"` は root と template とも 0。`.awk` のコードの行に単一引用符は 0 個 |
| AC2: S1 で 3 つを連結したものが base の 175〜1552 行目と一致 | Met | `git show da3b55d2:` の `lex`・`commands`・`rules` を連結し、`git show 0abfede5:.claude/hooks/pre_bash_guard.sh \| sed -n '175,1552p'` と `cmp`。root と `templates/base/` とも rc 0(S1 の行数は 599・320・459)。S1 の `.sh` は、base の 1〜174 行目と 1553 行目以降に比べて、174〜175 行目(`awk '` と `' 2>/dev/null)" \|\| awk_status=$?`)が `awk -f` の 1 行になっただけ。HEAD でも、3 つの `.awk` からコメントと空行を除いたものは base のプログラムから同じく除いたものと `cmp` で一致(982 行)。行末のバックスラッシュで続く行は 0 なので、コメントの行を足し引きしても読み方は変わらない。`.sh` のコメント以外の差分(base → HEAD)も `rule=` の 1 行だけ |
| AC3: テストが全部通る(jq あり・なし)、S1 でも 2,032 件 | Not verified(/test) | テストは実行していない。静的に見たこと: A〜D 節の `check_modes` の行は、base 580 行と S1 580 行がバイト単位で同じ(S1 はテストを変えていない)。HEAD は 588 行で、消えた行は 0、足した行は 8。`shellcheck -S warning` と `bash -n` は通る。2,032 件と 2,063 件は implementer と orchestrator の報告の値 |
| AC4: F 節の 2 つ(`.awk` が 1 つない写し、3 つの parse) | Met(静的と観察。実行は /test) | 行は `tests/test-pre-bash-guard.sh:1432-1476` にある。parse: `LC_ALL=C /usr/bin/awk -f lex -f commands -f rules </dev/null` は BWK awk 20200816 で root と template とも rc 0・出力なし。ubuntu:24.04 の mawk 1.3.4 と dash(jq なし)でも rc 0・出力なしで、`sudo ls` deny、`ls` none、`--no-verify` の commit は deny。`commands.awk` を抜いて読ませると rc 2(`calling undefined function end_cmd`)。`rules.awk` を消した写しの guard では `sudo ls` deny、`ls` none、`git commit --no-verify -m x` none(旧版の 4 規則が決めた) |
| AC5: B 節 group 11 に 3 行(deny)、D 節 `edge_none` に 4 行(none)、M20 と M13 系統が赤 | Partially verified | 3 行は `guard_deny_only_forms` の group 11(`:740-742`)、4 行は `edge_none`(`:1156-1159`)にある。f593bbfc の `git commit -m $"never sudo ls"` も `edge_none`(`:1164`)。8 行を HEAD の root と template(jq あり・なし)に通すと、B の 3 行は deny/deny、D の 5 行は none/none。base の guard(1 ファイル)も同じ判定で、判定を固定する行であって振る舞いの変更ではない。旧版の fixture は 8 行とも deny。mutation M20・M13 系統が赤になることは /test が見る |
| AC6: 写しが一致、`check-template.sh` が通り、`.awk` がないと `Missing required file` | Met | `./scripts/check-sync.sh` rc 0(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0)。guard の 4 ファイルと `check-template.sh` は root と template で `cmp` rc 0、git の mode は `.sh` 100755・`.awk` 100644 で両側同じ。`./scripts/check-template.sh` rc 0。`git archive HEAD` の木から `pre_bash_guard_lex.awk` を消すと `FAIL: Missing required file: .claude/hooks/pre_bash_guard_lex.awk` で rc 1。`ralph init` した木から `rules.awk` を消しても同じく rc 1 |
| AC7: init と upgrade の両方で 3 つの `.awk` があり、guard が deny/none/deny | Met(HEAD のビルドで取り直した) | `git archive` の HEAD(6dab8a5d)と base(0abfede5)から ralph をビルドした。HEAD の `ralph init` の木と、base の `ralph init` のあとに HEAD の `ralph upgrade` をかけた木(created 3 = 3 つの `.awk`、updated 2 = `.sh` と `check-template.sh`)の両方で、4 つのファイルが manifest に `owner = 'core'` で載り、`.awk` は 0644、内容は HEAD の template とバイト単位で同じ。guard は `sudo ls` deny、`ls` none、`git commit --no-verify -m x` deny。最後の形は `PreToolUse.d/10-pre-bash-guard.sh` を通しても deny。両方の木で `check-template.sh` rc 0 |
| AC8: S3 のコメントが直り、S2・S3 でコメント以外の行が変わらない | Met | S3 の項目を HEAD のコードと照らした: `cmd_pos` のコメントが `EXEC_SEEN` と `end_cmd` を書く(`commands.awk:118-121`、書き込みは `:125`・`:134`、読むのは `:72`)、`new_ctx` の前の一覧に `HPQ`・`HPN`・`PLC`・`PLN`・`STC`・`STN`・`PLID`・`CPL`、`lex_redir` の区切りのコメントが「シェルはほとんど書かれたとおりに読む。違うのは `\n` `\t` `\r` 以外のエスケープと `$"…"`」、`data_first_ok` のコメントが 2 つの例外を書く、ヘッダーの Not covered に printf、テストのヘッダー item B が group 11 まで、group 1 のコメント、`edge_*` の出典(triage のファイル名と a9ef82b1)。古い文言(「were false none before」「Cycle 2 (P2-4)」「change A, cycle 2」)は 0 件。da3b55d2 の後の 13 コミットそれぞれで、guard の 8 ファイルの追加・削除の行のうち `^[-+][[:space:]]*(#\|$)` でないものは 0 行 |
| AC9: `run-verify.sh` rc 0、tech-debt の (d) と直した (e) が解消済みでコードの項目だけ残る | Partially verified | `./scripts/run-static-verify.sh`(`run-verify.sh` の static mode)は rc 0。テストを含む `run-verify.sh` は /verify では回していない。tech-debt `:160` は Debt と Trigger の (d) を取り消し線で閉じ、(e) の S2・S3 の項目を解消済みにした。ただ、Impact (d) と Why deferred (d) はまだ開いて読め(V-1、V-2)、Debt (e) にはコードでない `SQ` の綴りの項目が残る(V-2) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)`。shellcheck、`sh -n`(hooks 20 本)、settings の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync(13)、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint(0 issues)、branch secret scan(0abfede5..6dab8a5d clean)が通った |
| `shellcheck -S warning` | 0 件 | guard の `.sh`(root と template)、`check-template.sh`(両方)、`tests/test-pre-bash-guard.sh`、`tests/test-check-template.sh` |
| `sh -n` | rc 0 | guard の `.sh` の両方。テスト 2 本は `bash -n` で rc 0 |
| `./scripts/check-sync.sh` | rc 0 | DRIFTED 0、ROOT_ONLY 0 |
| `./scripts/check-template.sh` | rc 0 | `Template structure looks good.` |
| 3 つの `.awk` の parse(guard と同じ順と引数) | rc 0・出力なし | BWK awk 20200816(root と template)、mawk 1.3.4(Docker)。gawk は手元にない |
| `wc -l` | 231 / 616 / 336 / 465 | すべて 800 未満 |
| `./scripts/plan-visual.sh digest` | cfb34e566022 | plan の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| guard のヘッダー(`pre_bash_guard.sh:12-13`、`:174-186`) | Yes | awk のプログラムが 3 つの `.awk` にあること、読む順(lex、commands、rules)、5 ファイルを一緒に置くこと、`.awk` が欠けると fallback になることを書く。item 3 と 6(a) の「NOEXEC/DATACMD in pre_bash_guard_rules.awk」は実際の場所(`rules.awk` の BEGIN、BEGIN はこのファイルにだけある)と合う。item 6(b) に `$"…"`(`:104-105`)。単一引用符の注記(base の 485 行目)は消えた |
| 3 つの `.awk` の先頭のコメント | Yes | どれも自分の節と、読み方の説明は `.sh` の呼び出しの上にあることだけを書く。節の見出しは lex(Text access、Lexing)、commands(Simple-command assembly、Data regions)、rules(Rule judgement、The sentinel、Main)で plan の Scope と同じ |
| `AGENTS.md:103`(`.claude/hooks/`) | Yes | guard のファイルの構成には触れていない。直すところはない |
| `.claude/rules/ralph/git-commit-strategy.md:69,71` と template | Yes | `pre_bash_guard.sh` を入口として挙げるだけで、今も正しい。root と template は `cmp` で一致 |
| guard に触れるほかの文書 | Yes | repo 全体の `git grep -i 'pre_bash_guard\|bash guard'`(reports・plans・insights・tests・tech-debt・hook 本体を除く)の各行を見た。`internal/org/prompts/implementer.md:27`、`.codex/README.md:116` は判定の説明で、判定は変わっていない。`internal/cli/migrate.go:77` は settings.json から直接呼んでいた hook のコマンドの一覧で、`.awk` は入らない |
| `scripts/check-template.sh:30-42` と template | Yes | 5 ファイルを挙げる理由のコメントと `required_files` が合う |
| `docs/tech-debt/README.md:189`(`.awk` を分類できない件) | Yes | 行が引く文言 `Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)` は今回の static verify の出力と同じ。`classify_language`(`scripts/detect-changed-languages.sh:276`)と `tests/test-detect-changed-languages.sh` は実在する |
| `docs/tech-debt/README.md:160` の Why deferred (d) | No(V-1) | 「Moving the awk program into its own file adds a second core file … whether that needs code changes in the manifest or in `internal/cli/migrate.go` (it names the guard at line 77) was not checked」が閉じないまま残る。AC7 の取り直しで、init と upgrade はコードを変えずに 4 ファイルを core として届けた。`migrate.go:77` は `legacyRalphHookCommands`(settings.json の直接のコマンド)で、`.awk` は関係しない。(d) を閉じたことと、この確認の結果を書くとよい |
| `docs/tech-debt/README.md:160` の Impact (d)、Debt (e)、Trigger (e)、Debt (b) | No(V-2、self-review LOW 7 と同じ) | Impact (d)「The file is harder to read and change than the guideline intends.」が閉じていない。Debt (e) は「only the code items stay open」の直後に、コードでない `SQ` の綴りの項目を挙げ、Trigger (e) にその項目のきっかけがない。Debt (b) の「One escape goes the other way」は `$'\echo' 'sudo ls'` だけで、`$"echo" 'sudo ls'`(新版 none/none、旧版 deny。`commands.awk:100-103` のコメントは書いている)がない。/sync-docs に回すと self-review が決めている |
| `docs/tech-debt/README.md:160` の Debt (d) の行数と (e) のコミット | 情報(V-3) | 「at c3a323aa the script is 230 lines and the three files 618, 336, and 468 lines」は c3a323aa では正しい(確かめた)が、8c61c6cb のあと HEAD は 231・616・336・465。(e) の「every comment-only item below was fixed there (7347fa6a, f593bbfc, c3a323aa …)」は、コメントを直した 8c61c6cb を挙げていない。どちらも誤りではないので、直すかどうかは /sync-docs の判断 |
| plan の Progress | 情報 | 「Verification artifact created」は未チェック。この report のあとで更新される。AC のチェックは実装と合っている |

## Observational checks

- AC7 の init と upgrade(`scratchpad/vrf/ac7/run.sh`): orchestrator の `ac7/` の手順を、HEAD 6dab8a5d からビルドした ralph で取り直した。結果は上の表のとおりで、orchestrator が 644f5c59 で取った結果と同じだった(created 3、updated 2)。
- 判定の比較の対照(`ac7/control.sh`): `rules.awk` を消した写しでは `--no-verify` の commit の形が none になる。AC7 の deny は awk の経路が決めたものと分かる。
- 足した 8 行の判定(`scratchpad/vrf/probe.sh`): HEAD の root と template、jq あり・なしの 4 通り、base の guard、旧版の fixture。`sudo` を含む形は Bash のコマンド行に書かず、テストの配列から取り出したファイルを guard に渡した。
- Docker(ubuntu:24.04、mawk 1.3.4、dash、jq なし)で、template の guard 5 ファイルを使って parse と 3 つの判定を確かめた。

## Coverage gaps

- テストスイートの実行(AC3 の件数、AC4・AC5 の実行、mutation M20 と M13 系統)は /test が見る。base と分割後の guard を全コーパスで比べることもしていない。コードの行が base と一致するので判定は変わらないはずだが、実行での確認は /test に任せる。
- gawk での parse と判定は確かめていない(手元にない)。plan の前提では CI の ubuntu で mawk と gawk の両方を回す。
- 静的解析の側に `.awk` の検査がない(V-4)。`scripts/verify.local.sh:176`・`:191` の shellcheck と `sh -n` は `.claude/hooks/*.sh` だけを見る。3 つの `.awk` の構文エラーを捕まえるのはテストの F 節だけで、guard 本体は awk の stderr を捨てて黙って fallback に落ちる。いちばん小さい追加は、`verify.local.sh` に root と template の両方で `LC_ALL=C awk -f …_lex.awk -f …_commands.awk -f …_rules.awk </dev/null` を回し、rc 0 で出力が空のときだけ通す 1 段を足すこと。`.awk` を言語に分類する件(tech-debt `:189`)と一緒に決めるとよい。
- legacy layout(v1)から `ralph upgrade` で移行する経路で `.awk` が届くかは確かめていない。AC7 の upgrade は base の v2 の木からだけ。
- merge のあとに main のチェックアウトの guard として Claude Code で効くことは、merge 前には確かめられない。

## Verdict

- Verdict: partial-pass
- Verified: AC1、AC2、AC6、AC7、AC8。S1 の 3 つの `.awk` の連結は base の 175〜1552 行目とバイト単位で一致し、HEAD でもコメントと空行を除いたコードは base と同じ(982 行)。`.sh` のコードの変更は `rule=` の 1 行だけで、S1 のあとの 13 コミットはどれも guard のコメント以外の行を変えていない。init と upgrade の両方で 4 ファイルが core として届き、awk の経路で `sudo ls` deny、`ls` none、`--no-verify` の commit は deny(`.awk` がないと none)。静的解析は全部通り、plan の digest も一致した
- Partially verified: AC4 と AC5(行の位置と判定は確かめた。テストの実行と mutation は /test)、AC9(static mode の `run-verify.sh` は rc 0。tech-debt の書き方に V-1・V-2 が残る)
- Not verified: AC3(テストの実行)、gawk、legacy layout からの upgrade、merge 後の実際の効き目
- partial にしたのは AC9 の tech-debt の残りがあるため。コードの AC は満たしている。指摘は LOW 2 件(V-1、V-2。V-2 は self-review LOW 7 と同じ)と情報 2 件(V-3、V-4)で、どれも記録の書き方か検査の追加で、マージを止めるものではない
