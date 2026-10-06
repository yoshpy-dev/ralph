# Test report: guard-bypass-and-hygiene

- Date: 2026-10-07(JST。実行の記録は UTC の 2026-10-06 23:19〜23:42)
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch fix/guard-bypass-and-hygiene の HEAD 98a43eb1 と base origin/main 2a22ba78 の差分(28 ファイル、+1604/-129、Go のファイルは 0)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC7 のうちテストで確かめる部分を見た。guard のテストと新しい 2 本は、Docker の ubuntu:24.04(dash、GNU grep 3.11、GNU sed 4.9、bash 5.2.21)でも実行した
- Evidence: `docs/evidence/test-2026-10-07-guard-bypass-and-hygiene.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-06-231941.log`、full の test モードのログは `docs/evidence/verify-2026-10-06-232447.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(既定の changed) | shell 39 ファイル(1,943 件)、Go 8 パッケージ | すべて | 0 | 1(下の注) | 287 s、rc 0 |
| `RALPH_VERIFY_SCOPE=full HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh` | 同上 | すべて | 0 | 1 | 283 s、rc 0 |
| `go test ./... -count=1 -cover` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 51 s、rc 0 |
| 指定の 5 本と `test-post-edit-verify.sh` を単独で 3 回ずつ | 220 / 5 / 22 / 20 / 51 / 24 | 3 回ともすべて | 0 | 0 | guard は 13〜14 s、ほかは 0〜3 s |
| ubuntu:24.04、jq を入れる前の `test-pre-bash-guard.sh` | 220 | 110(jq なしの経路) | 0 | 110(jq の経路) | rc 0 |
| ubuntu:24.04、jq と git を入れたあとの guard、verify-local、archive-plan、skill-insight-cycle | 220 / 5 / 22 / 20 | すべて | 0 | root で走らせた archive-plan の Case 4 だけ(一般ユーザーで流し直して 22 / 22) | rc 0 |
| mutation(私が作った 39 件、39 件とも適用できた) | 39 | red 34 件、残り 5 件(テストの失敗ではない) | - | - | 331 s |

- `run-test.sh` は `Requested scope: changed` で走り、`Language scope: full fallback (unclassified:.claude/hooks/lib_json.sh)` になった。既定の実行でも scope は full と同じになり、2 回目の `RALPH_VERIFY_SCOPE=full` の実行と suite ごとの件数が `diff` で一致した。
- shell の 39 ファイルは `tests/test-*.sh` の全件で、ディスク上の一覧と実行した一覧を `comm` で比べて差はなかった。
- `verify.local.sh` の実行権限の検査(AC4)で FAIL になったテストは 0 本。ログに `not executable in the working tree (` と `git index mode is 100644 (` の行はない。手元の `git ls-files -s` では 39 本とも 100755 で、working tree でも 39 本とも実行権限がある。ubuntu で `git archive` の写しから作り直した repo でも 39 本とも 100755 だった。
- shell の skip 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」のケース。手元の git が 2.49.0 なので飛ばされる。この変更とは関係がない。
- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(上の 3 行目)。
- shell の件数の数え方: 各 suite の区間で `PASS` / `FAIL` / `SKIP` で始まる行(先頭の空白は許す)を数えた。`PASS: 220` や `PASS: 47 / 47` のような集計行と、`verify.local.sh` の `run` が出す `    OK` / `    FAIL` の行は除いた。指定の suite の件数は、各 suite 自身の集計行(`archive-plan tests: 22 passed` など)と一致した。
- 実行の前後で worktree の `git status --porcelain` は 0 行だった。

### shell の suite ごとの件数

| Suite | Passed / Total |
| --- | --- |
| `test-agent-models.sh` | 47 / 47 |
| `test-agent-phase-boundaries.sh` | 44 / 44 |
| `test-archive-plan.sh`(新規) | 22 / 22 |
| `test-branch-name.sh` | 26 / 26 |
| `test-check-mojibake.sh` | 15 / 15 |
| `test-check-skill-sync.sh` | 13 / 13 |
| `test-check-template.sh` | 55 / 55 |
| `test-codex-exec-invocation.sh` | 144 / 144 |
| `test-detect-changed-languages.sh` | 83 / 83 |
| `test-detect-languages-terraform.sh` | 8 / 8 |
| `test-ensure-pr-ready.sh` | 7 / 7 |
| `test-ensure-pr-title-prefix.sh` | 13 / 13 |
| `test-gc-artifacts.sh` | 11 / 11 |
| `test-hook-wiring.sh` | 68 / 68 |
| `test-insights-append.sh` | 51 / 51 |
| `test-language-pack-monorepo-roots.sh` | 29 / 29 |
| `test-new-feature-plan.sh` | 21 / 21 |
| `test-no-loop-references.sh` | 1 / 1 |
| `test-plan-visual.sh` | 103 / 103 |
| `test-post-edit-verify.sh` | 24 / 24 |
| `test-pr-owner-lookup.sh` | 20 / 20 |
| `test-pre-bash-guard.sh` | 220 / 220(4 から 216 件増) |
| `test-ralph-config.sh` | 19 / 19 |
| `test-ralph-dispatch.sh` | 33 / 33 |
| `test-ralph-worktree.sh` | 143 / 143 |
| `test-run-verify-branch-secret-scan.sh` | 32 / 32 |
| `test-run-verify-scope.sh` | 19 / 19 |
| `test-secret-scan-branch.sh` | 246 / 246(skip 1) |
| `test-secret-scan.sh` | 123 / 123 |
| `test-self-review-scope.sh` | 64 / 64 |
| `test-skill-insight-cycle.sh` | 20 / 20(16 から 4 件増) |
| `test-sync-skills.sh` | 22 / 22 |
| `test-template-purity.sh` | 10 / 10 |
| `test-terraform-gitignore.sh` | 47 / 47 |
| `test-terraform-pack-verify.sh` | 36 / 36 |
| `test-terraform-rule-frontmatter.sh` | 11 / 11 |
| `test-verify-local-hook-tests.sh`(新規) | 5 / 5 |
| `test-verify-mode-split.sh` | 59 / 59 |
| `test-xreview-helpers.sh` | 29 / 29 |

合計 1,943 件。#204 の test レポートの 1,694 件に、そのあと e23c8047 で `test-pr-owner-lookup.sh` と `test-insights-append.sh` に 1 件ずつ足した分を加えると 1,696 件になる。そこからの増分は guard の 216 件、skill-insight-cycle の 4 件、新しい 2 本の 22 件と 5 件で、ほかの 35 本の件数は #204 のレポートと同じだった。

## 指定の suite と AC の対応

| Suite | 中身 | AC | 結果 |
| --- | --- | --- | --- |
| `test-pre-bash-guard.sh` | guard の判定 108 件(A 1、B 1、C 12 形 × 2 モード、D 19 形 × 2、E 6 × 2、F 3 × 2、G 16、H 6、I 4)を jq あり・なしの両方で流して 216 件、`lib_json.sh` を直接 source する J の 2 件を両方の経路で 4 件 | AC1、AC2 | 220 / 220 |
| `test-verify-local-hook-tests.sh` | 実行権限なし、index が 100644、両方そろう、untracked で実行権限あり、untracked で実行権限なし | AC4 | 5 / 5 |
| `test-archive-plan.sh` | archive-plan.sh の Case 1〜5(17 件)と verify.local.sh の参照の検査の Case 6〜9(5 件) | AC6 | 22 / 22 |
| `test-skill-insight-cycle.sh` | 5 skill × 4 面の `--cycle auto` | AC5 | 20 / 20 |
| `test-insights-append.sh` | 既存の 51 件。`--phase sync_docs` を使うケースはない。そこで手で確かめた: `--phase sync_docs --verdict pass --cycle auto` は rc 0 で 1 行書き、`--phase sync-docs` は `Invalid value for --phase` で rc 1 になった | AC5 の前提 | 51 / 51 |
| `tests/test-*.sh` の 39 本(`run_hook_tests` 経由) | 実行権限の検査を通って全部が走った | AC4、AC7 | 上の表 |

AC1 の「deny が ask より先」は mutation で確かめた。ask の規則の写しを deny の前に差し込む(G2)と、F のモードなしの 6 件が ask になって FAIL した。bypass の終了を deny の前に動かす(G1)と、E・F・H の bypass の側の 20 件が none になって FAIL した。

## Plan の edge case と、それを見ているテスト

| Edge case | 見ているテスト | 結果 |
| --- | --- | --- |
| jq がないときの `permission_mode` の取り出し | guard の全 check の `[no-jq]` 側。mode を読まない mutation(G4)で、bypass の D と H の 40 件が FAIL | PASS |
| `permission_mode` が JSON にないとき | A〜F、H、I のモードなしのケース。`default` と `auto` は G | PASS |
| JSON エスケープされた `"` と `\` | I の 4 件、E の `git commit -m "$(id)"`、J の 2 件。sed fallback を旧版の `[^"]*` に戻す(L1)、戻しの置換を外す(L2)、`\\` を戻さない(L3)の 3 件とも red | PASS |
| jq なしの `tool_input.file_path`(`lib_json.sh` を直接 source) | J の 2 件。別に `post_edit_verify.sh` を origin/main と HEAD で比べた(下の節) | PASS |
| 複数行のコマンド | H の 6 件。tee の引数の終わりを `\` で切らない mutation(G13)は jq なしの側で、コマンドを 1 行にまとめてから grep する mutation(G15)は jq の側で red | PASS |
| `2>&1` と `2>/dev/null` | C の 4 形(`ls .git/ 2>&1` など) | PASS |
| fd つきの書き込み(`2> .git/x`) | D の `2> .git/x`、`&> .git/x` | PASS |
| 空白なしのリダイレクト | D の `>.git/x`。ただし heredoc まで空白のない `cat >.env<<EOF` はテストにない(Test gaps の 1) | 一部 |
| 絶対パス | D の `/abs/repo/.git/config`、`/abs/worktree/.git` | PASS |
| `.git` がファイルのとき | D の `echo x > .git`、`echo x > /abs/worktree/.git`。`.git` の後ろに `/` を必須にする mutation(G5)で red | PASS |
| 似た名前の plan | archive-plan の Case 1(`foo-bar.md`、`fooXmd`)と Case 2(`<slug>-bar.md`、`<slug>.md`)。境界の判定を外す(A1)、`.` を常に区切りにする(A2)、文末の `.` の例外を外す(A3)、`index` を正規表現の `match` にする(A8)の 4 件とも red | PASS |
| README がないとき | Case 3 の 2 件(`.gitkeep` だけのとき、`docs/tech-debt/` がないとき) | PASS |
| 移動が失敗したあとのやり直し | Case 4 の 6 件。macOS(一般ユーザー)と ubuntu の一般ユーザーで PASS、ubuntu の root では SKIP の行が出た。移動を書き換えの前に戻す mutation(A4)で red | PASS |

## Mutation

`git archive HEAD` で scratchpad に写した木の中で、1 か所ずつ文字列を置き換えて該当の suite を流し、pristine の写しから戻して `cmp` で戻ったことを確かめた。worktree の追跡ファイルには触れていない。FAIL の件数は suite の集計行を除いた数。

| ID | 書き換え | 結果 |
| --- | --- | --- |
| G1 | bypass の終了を deny の規則より前に動かす | red(E・F・H の bypass 側 20 件) |
| G2 | ask の規則の写しを deny の前に差し込む(旧版の順) | red(F 6 件) |
| G3 | 比べる値を `bypass` にする | red(40 件) |
| G4 | `permission_mode` を読まない | red(40 件) |
| G5 | `git_target` で `.git` の後ろに `/` を必須にする | red(`.git` がファイルの 2 形 × 2 経路) |
| G6 | `tee` の検査を外す | red(tee の 3 形 × 2 経路) |
| G7 | `>\|` を受け付けない(`>[>\|]?` を `>>?` に) | red(2 件) |
| G8 | `word_end` から `<` と `>` を外す | 残った |
| G9 | `tee` の前の `/` を外す(`/usr/bin/tee` を見ない) | 残った |
| G10 | `.env` の後ろの文字を許さない | red(`.env.local`、`.envrc`、8 件) |
| G11 | `.env` の後ろに `/` を許す | red(C の `echo x > .env/notes.txt`、2 件) |
| G12 | コミットメッセージの deny から `$(` を外す | red(16 件) |
| G13 | tee の引数の区切りから `\` を外す | red(H、jq なし 1 件) |
| G14 | `word_char` に `<` と `>` を入れる | 残った |
| G15 | コマンドを 1 行にまとめてから grep する | red(H、jq 1 件) |
| L1 | sed fallback を旧版の `"([^"]*)"` に戻す | red(16 件) |
| L2 | エスケープを戻す置換を外す | red(12 件) |
| L3 | `\"` だけを戻し、`\\` を戻さない | red(J 1 件) |
| V1 | 実行権限のないテストを黙って飛ばす(旧版) | red(Case 1、5) |
| V2 | index の mode の比較を 100645 にする | red(Case 2) |
| V3 | `status=1` を外す | red(Case 1、2、5) |
| V4 | untracked でも `update-index` の行を出す | red(Case 5) |
| V5 | 文末の `.` を落とさない | red(Case 6、9 の 3 件) |
| V6 | `-e` を `-f` にする(ディレクトリの plan を見落とす) | red(Case 6) |
| V7 | 参照切れでも 0 を返す | red(Case 7) |
| V8 | git の work tree の判定を常に false にする | red(Case 2、5) |
| V9 | 参照の検査を呼ばない | red(Case 6〜9 の 5 件) |
| A1 | 名前の境界を見ずに常に書き換える | red(Case 2) |
| A2 | `.` を常に区切りとして扱う | red(Case 2) |
| A3 | 文末の `.` の例外を外す | red(Case 1、9) |
| A4 | 移動を書き換えの前に戻す(旧版の順) | red(Case 4 の 3 件) |
| A5 | `cp -p` を `cp` にする | red(README の mode が 600 になる) |
| A6 | 書き換えが 0 件のとき一時ファイルを消さない | red(Case 4 のやり直し) |
| A7 | 移動先の衝突の検査を書き換えのあとに動かす | red(Case 5 の 2 件) |
| A8 | `index` を `match` にする(`.` が任意の 1 文字になる) | red(Case 1 の `fooXmd`) |
| A9 | 0 件でも README を置き換えて `Updated 0` を出す | red(Case 4 のやり直し) |
| S1 | `.agents/skills/sync-docs` の `--cycle auto` を消す | red |
| S2 | `.claude/skills/sync-docs` の `--phase sync_docs` を `sync-docs` にする | 残った |
| S3 | `.claude/skills/sync-docs` の `\|\| true` を消す | 残った |

残った 5 件の扱い。G8、G9、G14 は scratchpad で元の guard と mutation の guard に同じ payload を渡して比べた(jq あり・なしとも同じ結果)。

| ID | 差が出るコマンド | 元の guard | mutation |
| --- | --- | --- | --- |
| G8 | `cat >.env<<EOF`(複数行の heredoc) | ask | none |
| G9 | `/usr/bin/tee .git/x`、`echo x \| /usr/bin/tee -a .env` | ask | none |
| G14 | `cat >/tmp/out.txt</repo/.git/HEAD`、`sort >/tmp/s.txt<./.env.example` | none | ask |

- G8 と G9 は、plan の Progress checklist が書く S1 の handoff からの差分(書き込み先の語の終わりに `<` と `>` を足した、`tee` の直前に `/` を許した)そのもので、コードにはあるがテストがない。G14 は、書き込み先の前に `<` で読み込みが続く形の誤検知を防いでいる部分で、これもテストがない。どれも D か C の一覧に 1 行足せば見分けられる(Test gaps の 1〜3)
- S2 と S3 は、`test-skill-insight-cycle.sh` が `--cycle auto` しか見ないことを示す。S2 の値(`sync-docs`)は `insights-append.sh` が rc 1 で断るが、`|| true` があるので `/sync-docs` は何も言わずに event を書かずに終わる。AC5 の `--phase sync_docs`、`--verdict pass`、`|| true` があることは verify が grep で確かめた(Test gaps の 4)

## lib_json.sh のもう 1 つの caller(post_edit_verify.sh)

`lib_json.sh` を使う hook は `pre_bash_guard.sh` と `post_edit_verify.sh` の 2 本。`test-post-edit-verify.sh` は jq を必須にしていて、jq なしの経路を通らない。そこで origin/main と HEAD の `post_edit_verify.sh` と `lib_json.sh` を scratchpad に置き、同じ 5 つの payload を jq あり・なしで流して、stdout、stderr、終了コード、`.harness/state/` に書いたファイルを比べた。jq なしの PATH は `/bin` と `/usr/bin` から jq だけを抜いた symlink の集まり(macOS は `/usr/bin/jq` を持つため)。

| Payload | jq | jq なし |
| --- | --- | --- |
| Write、`/repo/scripts/x.sh` | 同じ | 同じ |
| Write、`/repo/docs/a b.md`、content に `\"` | 同じ | 同じ |
| Edit、`/repo/src/main.go`、old_string に `\"`、new_string に `\\` | 同じ | 同じ |
| Write、`/repo/we\"ird.md`(file_path に `\"`) | 同じ | 違う(下) |
| apply_patch(file_path なし) | 同じ | 同じ |

違いの出た 1 件は、旧版が file_path を `/repo/we\` で切って code の扱いにし、`Code file edited...` を出していたもの。新版は `/repo/we"ird.md` を読み、何も出さない。jq の経路と同じ結果で、plan の Risks の「`\"` を含まない値の結果は変わらない」とも合う。

## ubuntu:24.04 での実行

`git archive HEAD` の写しを tar で標準入力からコンテナに渡した。

- 環境: aarch64、`/bin/sh` は dash、GNU grep 3.11、GNU sed 4.9、bash 5.2.21。jq と git は最初は入っておらず、途中で `apt-get install jq git` で jq 1.7 と git 2.43.0 を入れた
- jq を入れる前の `bash tests/test-pre-bash-guard.sh`: 110 件 PASS(jq なしの経路)、110 件 SKIP、rc 0。`sh tests/test-skill-insight-cycle.sh` 20 / 20
- jq を入れたあと: guard 220 / 220、`sh tests/test-verify-local-hook-tests.sh` 5 / 5、`sh tests/test-archive-plan.sh` は root で 16 件 PASS と Case 4 の SKIP の行、一般ユーザー(`useradd` で作った)で 22 / 22。verify-local は一般ユーザーでも 5 / 5
- verify が「GNU の環境では動かしていない」と残した guard の判定は、テストの 220 件で GNU grep / sed でも同じ結果になった

`TMPDIR` を git の work tree の中に向けると、`test-verify-local-hook-tests.sh` の Case 1 が FAIL した(ubuntu。GNU の `mktemp -d` は引数なしでも `TMPDIR` に従う)。fixture が外側の repo の中に入り、`verify.local.sh` が正しく `git add --chmod=+x` の行を出すため、Case 1 の「repo の外なので git の直し方は出ない」という期待と食い違う。スクリプトの不具合ではなく、テストが置き場所を前提にしているためで、verify の Coverage gaps が書いた懸念がそのまま起きた。テストの写しの `TMP_ROOT="$(mktemp -d)"` の次の行に `export GIT_CEILING_DIRECTORIES="$TMP_ROOT"` を足すと、同じ条件で 5 / 5 になり、`TMPDIR` を外しても 5 / 5 のままだった(Test gaps の 5)。`test-archive-plan.sh` は同じ条件で 16 / 16(root)だった。macOS の `mktemp -d` は引数なしだと `TMPDIR` を無視するので、手元ではこの状況は起きない。

## guard の追加の probe

テストにない入力を、HEAD の guard に jq あり・なしで渡した(`jq -nc --arg` で payload を組んだ)。

| コマンド | モード | jq | jq なし | 備考 |
| --- | --- | --- | --- | --- |
| `echo x > .git/x` | なし、`default`、`plan`、`acceptEdits`、`auto`、`dontAsk` | ask | ask | テストは `default` と `auto` だけを持つ |
| `echo x > .git/x` | `bypassPermissions` | none | none | |
| `echo x > .git/x` | `BypassPermissions` | ask | ask | 大文字小文字は区別する |
| `git commit -m "$(id)"` | 上の 8 通りすべて | deny | deny | |
| `echo x 2>&1 > .git/x` | なし | ask | ask | |
| `echo x > /dev/null 2>&1; ls .git/` | なし | none | none | |
| `git log > .git-log.txt`、`echo x > .gitmodules` | なし | none | none | `.git` で始まる別の名前 |
| `echo x > .environment` | なし | ask | ask | 最後の要素が `.env` で始まるので、設計どおり |
| `echo "> .git/x"`、`echo '> .env'` | なし | ask | ask | 文字列の中の `>` も当たる。plan の Risks に書いてある挙動 |
| `cp hook .git/hooks/pre-commit` | なし | none | none | plan の Non-goals(self-review M-1) |
| `echo a` の次の行に `  > .git/x` | なし | ask | ask | |
| `echo a<TAB>><TAB>.git/x` | なし | ask | none | 下の注 |
| `git commit -m "fix: \"quoted\" word"` | なし | none | none | |
| `git commit -F - <<'EOF'` ... | なし | none | none | |

タブを `>` の後ろに置いた形だけ、jq の経路と sed の経路で判定が分かれた。sed の経路ではタブが 2 文字の `\t` のまま残り、`\` が語を切るため。self-review の L-1 と verify の V-7 が指摘した既知の差で、テストの D にある `echo x<TAB>> .git/x`(タブが `>` の前)は両方で ask になる。

## 実データでの archive の確かめ

`git archive HEAD` の写しで、この plan を `sh scripts/archive-plan.sh docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` で archive した。rc 0、`Archived:` の行だけが出て `Updated` の行は出ず、`docs/tech-debt/README.md` は byte 単位で変わらなかった。README に残る `docs/plans/active/` の参照は 0 件で、40 種類の plan の参照はどれも実在した。verify の予想(この plan への参照はないので `Updated` の行は出ない)と合う。

## Coverage

- Statement: shell には計測の道具がない。Go は `go test ./... -count=1 -cover` で `internal/cli` 84.7%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org` 90.8%、`internal/org/driver` 92.0%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。2026-10-04 からの数値と同じで、このブランチは Go のファイルを変えていない
- Branch: 計測していない
- Function: 計測していない
- Notes: shell の範囲は、上の edge case の表と mutation の結果で示した。新しいコードのうち、mutation で見分けられなかったのは guard の 3 か所(G8、G9、G14)と skill の 2 か所(S2、S3)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

`TMPDIR` を repo の中に向けたときの Case 1 の FAIL は、既定の環境(macOS、CI の ubuntu)では起きないため、この表には入れていない。Test gaps の 5 に書いた。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| bypass で guard が ask を返し、確認が出る(plan の Objective 1) | 直った | D と H の bypass のケース(none)。G3、G4 で red |
| ask に当たるコマンドでは deny まで届かない(`rm -rf x && git commit -m "$(id)"` が ask) | 直った | F の 3 形が全モードで deny。G2 で red |
| jq がないと `git commit -m "$(id)"` が何も返さない(plan の Assumptions) | 直った | E の jq なしの側。L1、L2 で red |
| `ls .git/ 2>&1`、`cat .env.example 2>/dev/null` などで確認が出る | 直った | C の 12 形が全モードで none |
| bypass 以外のモードでは、書き込み、`rm -rf`、`gh pr create` に ask を返す | 変わらず | D と G の ask のケース |
| `post_edit_verify.sh` の file_path の取り出し | `\"` を含まない値は変わらず、含む値は jq の経路と同じになった | 上の caller の節 |
| 実行権限のない `tests/test-*.sh` が黙って飛ばされる(plan の Objective 3) | 直った | `test-verify-local-hook-tests.sh` の 5 件。V1、V3 で red |
| `docs/tech-debt/README.md` が archive 済みの plan を active のパスで指す(plan の Objective 5) | 直った | 実データの確かめで active の参照 0 件。`archive-plan.sh` と `verify.local.sh` の検査は 22 件と A1〜A9、V5〜V9 |
| 既存の shell 35 本と Go 8 パッケージ | 変わらず | 件数は #204 の test レポートと e23c8047 の追加分のとおり。Go の coverage も同じ |

## Test gaps

コードは足していない。小さい順に、足すなら次の形になる。

1. `cat >.env<<EOF`(G8)。`tests/test-pre-bash-guard.sh` の `writes` に `$'cat >.env<<EOF\nA=1\nEOF'` を 1 行足せば見分けられる
2. `/usr/bin/tee`(G9)。`writes` に `'echo x | /usr/bin/tee -a .git/x'` を 1 行足す
3. `<` の読み込みが後ろに続く書き込み先の誤検知(G14)。`reads` に `'cat >/tmp/out.txt</repo/.git/HEAD'` を 1 行足す
4. `/sync-docs` の insight event の `--phase sync_docs`、`--verdict pass`、`|| true`(S2、S3)。`test-skill-insight-cycle.sh` は `--cycle auto` しか見ない。phase の値を間違えても `|| true` で黙って event が書かれなくなるので、skill ごとの `--phase <値>` を確かめる 1 行が足せる
5. `TMPDIR` が git の work tree の中にあると `test-verify-local-hook-tests.sh` の Case 1 が落ちる(ubuntu で再現)。`TMP_ROOT` を作った直後に `export GIT_CEILING_DIRECTORIES="$TMP_ROOT"` を足せば、置き場所によらず 5 / 5 になる(scratchpad の写しで確かめた)。CI と macOS の既定では起きないので、今は止める理由にはならない
6. タブを `>` の後ろに置いた書き込みは、jq がないと ask にならない(self-review L-1、verify V-7)。今回の probe で差を確かめた。直すなら hook の変更になる
7. `plan`、`acceptEdits`、`dontAsk` のモードはテストになく、probe だけで確かめた。コードの分岐は `bypassPermissions` との完全一致だけなので、実害の見込みは低い
8. Claude Code の実際の payload に `permission_mode: "bypassPermissions"` が入ること。この session で動いている guard は main のチェックアウトの旧版なので、新しい guard の live での確認はマージのあとになる。おそらく文書どおり。未確認です
9. `/pr` での実際の archive。scratchpad の写しで確かめた結果は上のとおりで、実際の `/pr` で同じになるかはその時点で分かる

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh`(shell 39 ファイル 1,943 件、Go 8 パッケージ、rc 0)、`RALPH_VERIFY_SCOPE=full HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`(同じ件数、rc 0)、`go test ./... -count=1`(8 / 8)、指定の 5 本(220、5、22、20、51 件)を単独で 3 回ずつ、ubuntu:24.04 の GNU grep / sed での guard と新しい 2 本、`post_edit_verify.sh` の新旧の比較、mutation 39 件のうち 34 件が red。実行権限の検査で落ちたテストは 0 本
- Fail: なし
- Blocked: なし

テストは通っており、/pr に進めない理由はない。Test gaps の 1〜5 はそれぞれ 1〜2 行で足せるので、/sync-docs か次の修正の機会に足すかを判断してほしい。
