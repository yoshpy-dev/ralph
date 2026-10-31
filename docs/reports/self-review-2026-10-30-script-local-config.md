# Self-review report: script-local-config

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-script-local-config.md`(#276 の B2、D1〜D7、AC1〜AC7、Progress の S1〜S7)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD 2cb62e4b、48 ファイル、+3557/-937)。2cb62e4b は origin/main(#292)の取り込みで、そのマージで入った変更は対象にしていない。中心は S1 8bb68fe2、S1b 67d9d80b、S2 e4794cd7、S3 674b193c、S4 7803a530、S5 5f201546、S6 a338dc66、S7 7fa65455。計画ファイルと Progress だけのコミットは読んだが所見の対象にしていない。見た観点は命名、読みやすさ、null 安全、例外処理、セキュリティ(値をコマンドに渡す形)、保守性。仕様の適合、テストの実行、静的解析、文書のずれは /verify・/test・/sync-docs の仕事なので見ていない。
- ID: 所見は L1〜L5(すべて LOW)。

## Evidence reviewed

- `scripts/ralph-config.sh` の全体(160 行)。`_ralph_setting_get` の分け方(exit 0 の空は既定値が空のキーだけ値、127 と `unknown command "config"` は黙って既定値、それ以外は `_rs_why` に理由)、`ralph_setting` と `ralph_setting_or_fail` の stdout・stderr・戻り値、`mktemp -d` の後始末(`rm -rf` は通常の道で必ず走る。trap は置かず、呼び出し側の trap を壊さない)。`${1-}` を使い `set -u` で落ちないこと。POSIX 以外の構文はない。
- HEAD から build した binary(scratchpad)で、一時 repo の `ralph.local.toml` に相対パスの `[plan_visual].browser` を書いた状態を作り、4 つの読み手を打った。`ralph config get verify.base` は exit 3。`detect-changed-languages.sh` は `scope=full` と `reason=config_error:verify.base` を出して rc 0。`detect_base_branch` は stdout 空で rc 1。`plan-visual.sh open` は path を stdout に出し、`plan_visual.opener exited 3 … using the empty default` を stderr に出して rc 0。D7 の振る舞いは本物の binary でも設計どおり。
- `detect_base_branch` を bash、`zsh -f`、dash で、偽の ralph(値あり、exit 3、空)を PATH の先頭に置いて打った。9 通りとも同じ結果(`feat`、失敗 rc 1 と 2 行の stderr、`main`)。`zsh -c` は rc ファイルが PATH を組み替えて本物の `/opt/homebrew/bin/ralph` が先に当たるので、probe には `-f` が要る(テストの `run_script` は `-f` を付けている)。
- `plan-visual.sh` を `ralph-config.sh` のないディレクトリに写して `open` を打った(L1)。`ralph-config.sh` を、関数のない古い版に差し替えて `detect_base_branch` を打った(L2)。`ralph.local.toml` の `skip_branch_secret_scan` に `0`、`"1"`、`1`、`true` を書いて `ralph config get` を打ち、`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=0 ralph doctor` の warn の文を読んだ(L3)。
- 値をコマンドに渡す形。`detect-changed-languages.sh` は `git merge-base HEAD "$base_ref"`、`xreview-helpers.sh` は `printf '%s\n' "$_dbb_setting"`、skill は `--base "$BASE"`、`plan-visual.sh` は `"$_opener" "$_html_abs"`(後ろの引数は絶対パス)と、どれも二重引用符の中。`-` で始まる値は Load が `baseBranchPattern` と `isScriptWord` で断るので、option には読まれない。ブラウザは絶対パスだけを通すので `-` で始まらない。`ralph config get` は Load を通った値だけを出す。
- 写し。`ralph-config.sh`、`plan-visual.sh`、`xreview-helpers.sh`、`detect-changed-languages.sh`、`run-verify.sh`、`secret-scan-branch.sh` の 6 つは root と `templates/base/scripts/` が `cmp` で一致。`quality-gates.md` は KNOWN_DIFF で、root と写しの差分の +/- 行が同じ。`cross-review/SKILL.md` は `.claude/` と `templates/base/.claude/` が `cmp` で一致(`.agents/` は設計上の差がある)。`verify.local.sh` は root だけの script。
- `detect_base_branch` の呼び出し元を grep した。skill の 4 つの写し(`BASE=$(detect_base_branch) || exit 1`)と `scripts/secret-scan-branch.sh` だけ。5 つの変数名を、tests と過去の記録を除いて grep した。残りは `doctor.go` の表、README の「もう読まない」の説明、`ralph-config.sh` の同じ注釈だけ。`gateLocalConfigEnv` は `internal/org/gate_run.go:53` に実在する。
- Go。`validateScriptValues`、`isScriptWord`、`baseBranchPattern`(`$` は end-of-text なので末尾の改行も断る)、`LocalOnlyKeys` の 6 つ、`retiredScriptEnv`、`config get --help` の追記、`gate_run.go` の注釈。`local_script_keys_test.go` は大文字小文字の 5 つの綴りを、本物の `toml.Unmarshal` が値を読むことを確かめてから使っている(自分の写しとの突き合わせではない)。
- テスト。`tests/lib/fake-ralph.sh` の全体、`tests/test-xreview-helpers.sh` の Test 3(3a〜3k)、`test-run-verify-branch-secret-scan.sh` の `run_verify` と `test_skip_setting`、`test-run-verify-scope.sh` の D7 の節、`gate_runverify_test.go` の stub、`test-ralph-config.sh` の頭(PATH は `/usr/bin:/bin` と stub だけで、手元の ralph に届かない)。「ralph なし」の case は、`test-plan-visual.sh` と `test-detect-changed-languages.sh` が必要な道具だけの PATH を組むので、手元の ralph に届かない。
- 800 行の目安。変更したファイルのうち 800 行を超えるものを、base との行数で比べた(L4)。
- 確かめていない: テスト、静的解析、変異の実行(skill の規則どおり走らせていない)。CI の Linux で `test-plan-visual.sh` のブラウザ自動検出 2 件が SKIP でなく通るか。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | L1 `plan_visual_setting` だけが `ralph-config.sh` の有無を確かめずに source する。ほかの 3 つの読み手は、ファイルがなければ設定を読まずに進む(ralph を打たず、既定の振る舞い)。`plan-visual.sh` は、`ralph-config.sh` が `$0` の隣にないとき(部分的に写した repo、`scripts/plan-visual.sh` への別の場所からの symlink)、`open` が path を stdout に出したあとで落ち、`shot` も落ちる。ファイル頭は「opener がないか失敗しても exit 0」と約束している。同じ「root を探して source し、読む」の形が 4 か所にあり、root の探し方も 3 通り(script の隣、cwd の git toplevel、cwd の `./`)で、守りの有無がそのずれの 1 つになっている。 | `scripts/plan-visual.sh:159` `. "$_scripts_dir/ralph-config.sh"`(守りなし)。守りあり: `scripts/xreview-helpers.sh:58`、`scripts/detect-changed-languages.sh:185`、`scripts/run-verify.sh:258`。probe(`ralph-config.sh` なし、`open p.html`): path を出したあと、dash は rc 2(`.: cannot open`)、`/bin/sh`(bash の POSIX モード)は rc 1、bash は rc 127(`ralph_setting: command not found`)。 | `plan_visual_setting` の先頭に `[ -f "$_scripts_dir/ralph-config.sh" ] \|\| return 0`(2 つのキーの既定値は空なので、空を返せば足りる)を足し、`test-plan-visual.sh` に「隣に `ralph-config.sh` がないコピーで `open` が rc 0」の case を足す。写しは `templates/base/scripts/plan-visual.sh` も同じ。 |
| LOW | readability | L2 設定を読めなかったときの 3 つの文が、原因を `ralph.local.toml` に決めている。実際は ralph.toml の誤り、別のキーの誤り(検証は全体で 1 つなので、どのキーの誤りでも全キーが exit 3 になる)、`mktemp` の失敗、関数のない古い・手で直した `ralph-config.sh` でも同じ文が出る。探すべき場所を違うところに案内する。直前の行(`ralph-config.sh` の `ralph config get … exited 3 (…)`)に本当の理由があるので、情報は落ちていない。 | `scripts/xreview-helpers.sh:62`、`scripts/detect-changed-languages.sh:189`(`[verify].base could not be read from ralph.local.toml`)、`scripts/secret-scan-branch.sh:161`。probe: `ralph.local.toml` に誤った `[plan_visual].browser` だけを書くと、`detect-changed-languages.sh` は「`[verify].base` could not be read from ralph.local.toml」と言う(書いていないキー)。関数のない `ralph-config.sh` では `ralph_setting_or_fail: command not found` のあとに同じ文が出る。 | 文を「ralph config get が失敗した(理由は上の行。1 つのキーの誤りが全キーを失敗させる)」の形にして、キー名は読もうとしたキーとして残す。3 つの文は `test-xreview-helpers.sh` の 3h と `test-secret-scan-branch.sh` が句で固定しているので、テストの句も同時に直す。 |
| LOW | exception-handling | L3 `ralph doctor` の `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` の行は、`1` のときだけ意味があった変数なのに、どんな空でない値にも「値を `[verify].skip_branch_secret_scan` に書く」と案内する。`0` や `yes` を持つ人がそのとおりに `= 0` と書くと、bool のキーに整数なので Load が断り、D7 によって全キーが失敗する(verify は full scope、`/cross-review` は止まり、secret scan は cannot scan)。つまり、無害だった変数の移行案内が、設定全体を壊す書き方を勧める。テストの値は `1` だけ。 | `internal/cli/doctor.go:909-910`(`more` は「as true if it is 1, the only value that skipped the scan」)。`internal/cli/doctor_retired_script_env_test.go:26` の値は `"1"` だけ。probe: `skip_branch_secret_scan = 0`、`"1"`、`1` はどれも `ralph config get` が exit 3、`true` は `true`。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=0 ralph doctor` の warn は「write the value in … , as true if it is 1, …」。 | `more` を「1 のときだけ `true` と書く。ほかの値は何も書かない(その値ではスキップしていなかった)」の形にし、テストに `0` の値の case を足す。 |
| LOW | maintainability | L4 800 行の目安を超えるファイルが、この PR で伸びたか、超えた。`defaults_sync_test.go` だけは S1b で分けたが、ほかの 5 つには register の行がない。B1 の PR は同じ状況を「have no row of their own for this branch」と書いて行にした。 | base から HEAD の行数: `tests/test-detect-changed-languages.sh` 669 → 848(この PR で超えた)、`tests/test-plan-visual.sh` 857 → 998、`tests/test-secret-scan-branch.sh` 2116 → 2217、`internal/config/config.go` 1065 → 1180(+115。新しい 2 つの型と `validateScriptValues`)、`internal/cli/doctor.go` 1483 → 1510。`docs/tech-debt/README.md` を `grep` しても、ファイルの大きさを理由にこの 5 つを名指す行はない(この PR が足した行を含む)。 | 1 行にまとめて register に足す(下の Tech debt identified)。分けるなら `config.go` の `PlanVisualConfig`・`VerifyConfig` と `validateScriptValues` を `internal/config/script_values.go` へ、が移すだけで済む単位。 |
| LOW | readability | L5 `scripts/ralph-config.sh` の頭の終了コードの一覧で、4 つ目の項目が読みにくい。条件が「any other (…), and a temporary directory that cannot be made」と 4 行にわたり、結果の「a failure:」が次の行から始まるので、「be made」の行が単独で残り、どこまでが条件かが追えない。ほかの項目(`0`、`127`、`1 with …`)は 1〜2 行で結果が同じ行にある。 | `scripts/ralph-config.sh:38-49`(`templates/base/scripts/ralph-config.sh` も同じ)。 | 条件を「any other exit code (2 for …, 3 for …, any other 1, and the rest), or no temporary directory」と 1 文にして、結果を「a failure:」から新しい段落にする。コメントだけの変更なので、写しも同じに直す。 |

## Positive notes

- `ralph_setting` の約束(どの場合も 0、`set -e` の呼び出し側が止まらない)を変えずに、`_ralph_setting_get` に分けて `ralph_setting_or_fail` を足した。分け方が 1 か所で、2 つの関数が違うのは理由の扱いだけ。`_rs_` の接頭辞、`mktemp` 失敗の分岐、「exit 0 の空は既定値が空のキーだけ値」の条件が、ヘッダと関数の間で合っている。
- D7 が 3 つの層で閉じている。helper(`_or_fail`)、読み手(`fallback_full`、`return 1`)、呼び出し側(skill の `|| exit 1`、`secret-scan-branch.sh` の `cannot_scan`)。本物の binary でも同じ結果になった。`--strict` の push の守りは `/pr` と leader が使っているので、`cannot_scan` の exit 3 が実際に push を止める。
- `detect_base_branch` と `detect-changed-languages.sh` が `ralph-config.sh` を subshell の中で source する。呼び出し側の shell に関数と `_rs_` の変数が残らず、`3k` のテストが固定している。`CDPATH=''` と `--` を付けた `cd`、二重引用符、`git rev-parse --show-toplevel` で root を引く形は、worktree の中でも通る(`3e`)。
- 値の検証が、使われ方(`"$opener" "$page"` の 1 語、`[ -f "$browser" ]` の絶対パス、ブランチ名の文字)に合わせて 4 つのキーで別々になっている。ブラウザは空白を許し(`/Applications/Google Chrome.app/…`)、opener と `verify.base` は許さない。エラーの文がキーと値を名指す。テストは受け入れる値と断る値を表で持つ。
- `fake-ralph.sh` が env を読まず、答えを `BIN/ralph.fake/` に置く。env を消して起動する script でも答えが届く。`_old` の stderr が本物の v6.0.0 と同じ 1 行で、未知のキーは本物と同じ文で exit 2、期待しない呼び出しは exit 99。それぞれのテストが「env を設定しても効かない」行と、呼び出し記録(`pwd -P|config get <key>`)による「どこで何回読んだか」の行を持つ。
- gate のテストが、stub に `RALPH_LOCAL_CONFIG` の値で答えを変えさせ、`off` のときは既定値、そうでないときは skip と `HEAD` を答えさせる。コントロールの実行(gate を通さない)が skip を実際に効かせ、gate の実行が scan を走らせることを同じ fixture で見ている。
- `defaults_sync_settings_test.go` への移動が、`go test -list` の本数(104)と `cmp` で、中身を変えない移動だと確かめられている(S1b)。

## Notes(所見にしないもの)

- `secret-scan-branch.sh` の `cannot_scan` は、既定のモードでは報告して exit 0 にする(`report_and_exit`、早期警告の扱い)。`run-verify.sh` は既定のモードで打つので、設定の誤りのとき、branch の scan は走らず、`==> Running branch secret scan` と 1 行の notice のあとに `All verifiers passed` が出る。ほかの `cannot_scan` の理由(base ref がない、merge-base がない)と同じ扱いで、`--strict` を使う `/pr` と leader の push の守りが止める。計画の AC3b の「`secret-scan-branch.sh` は exit 3」は `--strict` の話として読む。/verify が既定のモードで打つと exit 0 が返る。
- `config get` を持つが B2 のキーを持たない ralph(main の B1 と B2 の間の build)では、`standard.base_branch` と `verify.base` が exit 2 になり、`_or_fail` は失敗として扱う(full scope、cannot scan)。release は A1〜B2 を同じ版に含み、v6.0.0 は `config get` を持たないので、実在する版には当たらない。その間の build を PATH に置いた開発者だけに出る。
- `tests/lib/fake-ralph.sh:84-87` の `fake_ralph_value` は、`_fake_ralph_answer` が置いた `_fr_key` を読む(関数の間の暗黙の受け渡し)。ヘッダは `_fr_` の接頭辞を書くが、この受け渡しは書かない。今は呼ぶ順が固定なので壊れない。
- `tests/test-plan-visual.sh:446` のブラウザ自動検出の 2 件は、`/Applications` に Chrome か Chromium がある Mac では SKIP で、通過にも失敗にも数えない。Linux の CI で走る前提を、計画の Progress が書いている。PR の CI のログで SKIP でなく通ったかを見る。
- 古い `ralph-config.sh` を手で写した repo のエラー文のずれは、S5 の implementer と consult が低として認めている。L2 がその文を直すと同時に解消する。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| script-local-config で 800 行を超えたか、超えたまま伸びたファイル 5 つ: `tests/test-detect-changed-languages.sh`(669 → 848)、`tests/test-plan-visual.sh`(857 → 998)、`tests/test-secret-scan-branch.sh`(2116 → 2217)、`internal/config/config.go`(1065 → 1180)、`internal/cli/doctor.go`(1483 → 1510) | LOW。読む・見直すのに時間がかかる。振る舞いは正しい | 移動だけの分割は、設定の振る舞いの見直しと混ざる(S1b は目安の行に直に当たった 1 つだけを分けた) | 5 つのどれかを次に変える PR。先に `config.go` の `PlanVisualConfig`・`VerifyConfig`・`validateScriptValues` を別ファイルへ移す | 本 report の L4、`docs/plans/active/2026-10-30-script-local-config.md` |

_register の `docs/tech-debt/README.md` には追記していない。L4 を直すか、行にして先送りするかを決めるのは、この report を受け取る側で、/sync-docs が行を足す流れが B1 にもあった。_

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない(LOW が 5 件)。
- Follow-ups: L1 と L3 は小さな直しで、振る舞いが変わる(L1 は `open` の rc、L3 は doctor の文)。L2 と L5 は文の直しで、L2 はテストの句にも触れる。L4 は register の 1 行にするか、`config.go` の分割にするかの判断が要る。L1〜L3 を直すなら、パイプラインの全体の再実行に入る(コード、テスト、doctor の文を変えるので、記録だけの修正の例外には当たらない)。
