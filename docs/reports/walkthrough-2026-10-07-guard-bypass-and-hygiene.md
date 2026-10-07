# Walkthrough: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/archive/2026-10-07-guard-bypass-and-hygiene.md
- Branch: fix/guard-bypass-and-hygiene
- Diff: 34 files, +2625 / -145(大半はテストとパイプラインの記録)

差分を読む順に並べる。各節の最後に、確かめた方法を書く。

## 1. Bash の確認を bypass で出さない(`pre_bash_guard.sh`)

bypass permissions のモードでも、ralph の `pre_bash_guard.sh` が `ask` を返すと Claude Code は確認を出す。guard はいま、次の順で判定する。

1. deny の規則を先に見る。`sudo `、force push、`git reset --hard`、ダブルクォートの `-m` の中のコマンド置換の 4 つで、どのモードでも deny を返す。以前はコミットメッセージの deny が ask の後ろにあり、`rm -rf x && git commit -m "$(id)"` は ask で止まっていた
2. payload の `permission_mode` が `bypassPermissions` なら、何も返さずに終わる
3. ask の規則を見る。`.git` と `.env` は、リダイレクト(`>`、`>>`、`>|`)の直後の語と `tee` の引数だけを書き込み先として調べる。`rm -rf ` と `gh pr create` は今までどおり

書き込み先の判定は `grep -E` で書いた。`case` の glob では `ls .git/ 2>&1` と `echo x > .git/x` を見分けられないためで、POSIX のクラスだけを使う。BSD と GNU の両方で同じ結果になることを、macOS と ubuntu:24.04 で確かめた。

`tee` の引数の読み取りには、cross-review と self-review を 3 回通るなかで次の規則が加わった。

- `tee` の前に来てよい文字は、行頭、空白、`; & | ( /`、バッククォート、`\`(`\tee`)、それと sed の経路で残る 2 文字の `\n` と `\t`
- 引数の読み取りは `; & | ) < #`、バッククォート、`\` で止まる。`<` の後ろは `tee` が読む入力なので数えない。`>` では止まらない。`tee out.txt 2>/dev/null .env` は `.env` に書くため

確かめ方: `tests/test-pre-bash-guard.sh` は 324 件。none / ask / deny の 3 値で比べ、jq あり・なしの両方の経路で走らせる。

## 2. jq がないときの取り出し(`lib_json.sh`)

jq がないと、guard は sed で payload からコマンドを取り出す。以前の sed は `[^"]*` で値を読んでいたので、エスケープされた `\"` で切れていた。そのため jq がない環境では、`git commit -m "$(id)"` の deny が効いていなかった(Codex の plan advisory が指摘し、実測で確かめた)。

いまは `sed -E` の `"(([^"\\]|\\.)*)"` でエスケープを読み飛ばし、1 回の置換で `\"` と `\\` を戻す。`\n` と `\t` は戻さない。guard の判定に使う文字はどれも ASCII の固定文字列なので、戻さなくても結果は変わらない。もう 1 つの利用者の `post_edit_verify.sh` は `tool_input.file_path` だけを読むので、`\"` を含まない値の結果は変わらない。

確かめ方: 上のテストの jq なしの経路と、`lib_json.sh` を直接読み込む 2 件。

## 3. verify の手入れ(`verify.local.sh`、`run-test.sh`)

- shellcheck の対象を、手で並べた一覧から `scripts/*.sh` のグロブにした。一覧は 36 本中 9 本しか含まず、`insights-append.sh` も漏れていた。グロブにすると warning が出たのは `run-test.sh` の SC2209 だけで、引用符で囲んで直した
- `run_hook_tests` は、実行権限のない `tests/test-*.sh` と、git の index で 100644 のものを FAIL として数える。以前は黙って飛ばしていた。直し方の案内は、git が追跡しているファイルなら `git update-index --chmod=+x`、未追跡なら `git add --chmod=+x` を出す
- static のモードに、`docs/tech-debt/README.md` の plan の参照がそのパスに実在するかの検査を足した(5 節)

確かめ方: `tests/test-verify-local-hook-tests.sh` の 5 通り(偽のテストを置いた一時ディレクトリで `verify.local.sh` を走らせる)。

## 4. `/sync-docs` の insight event

`/sync-docs` の SKILL.md(4 面)に、`--phase sync_docs --verdict pass --cycle auto` の event を書く節を足した。過去の 50 行は担当エージェントが任意に書いたもので、#203 と #204 には 1 行もない。

確かめ方: `tests/test-skill-insight-cycle.sh` が 5 skill × 4 面を見る(20 件)。

## 5. tech-debt の plan のパス(`archive-plan.sh`)

`docs/tech-debt/README.md` は、archive 済みの plan 16 本を `docs/plans/active/` のパスで指していた(34 か所)。これを archive のパスに直した。参照がずれたのは、plan を移すときに参照を書き換える処理がなかったためなので、`archive-plan.sh` に次の手順を足した。

1. 移動先が空いていることを確かめる
2. README の `docs/plans/active/<name>` を `docs/plans/archive/<name>` に書き換え、件数を出す。名前は awk で文字列として比べ、直後に名前の文字が続く参照(似た名前の別の plan)は書き換えない。文末の `.` は区切りとして扱う
3. plan を移す

書き換えを移動の前に置いたのは、途中で止まってもやり直せるようにするため(Codex の plan advisory の指摘)。移動で失敗したあとにやり直すと、移動だけが走り、書き換えは 0 件になる。`/pr` の step 8 には、書き換えた README も同じコミットに入れると書いた。

確かめ方: `tests/test-archive-plan.sh` の 22 件(書き換え、似た名前、README がないとき、移動が失敗したあとのやり直し)。

## 6. パイプラインの経過

| 段階 | 結果 |
|---|---|
| cycle 1 | self-review(Merge 可、MEDIUM 1)、verify と test は pass。cross-review は ACTION_REQUIRED 2(改行の後の `tee`、`tee` の引数が `<` を越える) |
| cycle 2 | 0db1a97e で直した。self-review の C2-M1(`\tee` とバッククォートの中の `tee`)を ab3ee31c で直した。cross-review は ACTION_REQUIRED 1(閉じるバッククォートで `.git` の語が終わらない)と WORTH_CONSIDERING 1 |
| 追加の 1 周(cap を 3 に) | 3c0ba22a、206d8335、b94a9106 で直した。self-review は Merge 可(LOW 3)、verify と test は pass。cross-review は WORTH_CONSIDERING 1(プロセス置換の中の `.env` を書き込み先と取り違える誤検知)で、tech-debt に記録して PR に進んだ |

途中の判断は、ユーザーの事前の指示(寝ている間の確認はすべて承認扱い)に従って推奨の選択肢で進めた。各段の記録は `docs/reports/` の 5 本のレポートにある。

## 7. 残した穴

- 旧 guard が後ろの `>` に偶然当たって確認を出していた形は、捕まえなくなった。例は `cp hook .git/hooks/pre-commit 2>&1` と `tee out 2>&1 .env > /dev/null`。`cp`・`mv`・`sed -i` で書く形の検出は、もともと別の作業として tech-debt に置いている
- 誤検知が 3 つ残る。`tee "build .env.log"`(引用符の中の空白)、`printf "a\\ntee .env"`、`printf x | tee >(diff - .env)`
- jq がない環境では、`>` の後ろと `tee` の後ろのタブが見えない。旧 guard も見えていなかった
- Codex が `ask` をどう扱うかは確かめていない。Codex の payload にも `permission_mode` があるので、bypass では Codex でも ask を返さなくなる

どれも `docs/tech-debt/README.md` の guard の行と、テストの穴の行に書いた。
