# Test report: guard-deny-only

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md(Test plan の節、AC1〜AC9)
- Tester: tester subagent (Claude)、cycle 1
- Scope: `git diff origin/main...HEAD`(merge-base f423f230、HEAD adb7eda3)と、この /test で `tests/test-pre-bash-guard.sh` に足したテスト。変更された言語は shell だけだが、`lib_json.sh` を分類できないので `run-test.sh` は full にフォールバックし、Go のテストも流れた
- Evidence: `docs/evidence/test-2026-10-07-guard-deny-only.log`(gitignore の対象なので commit しない。2 回の `run-test.sh`、`run-verify.sh`、mutation の全結果と置換の一覧、probe、Docker の出力を入れた)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 1 回目(テストを足す前) | shell 40 本 + Go 8 パッケージ | 39 本 + Go 8 | 1 本(`tests/test-secret-scan.sh` の AC-15 の 1 件) | 0 | 365 s |
| `bash tests/test-secret-scan.sh` を単独で再実行 | 1 本 | 1 本(FAIL 0) | 0 | 0 | 数秒 |
| `bash tests/test-pre-bash-guard.sh`(テストを足す前、adb7eda3) | 1508 | 1508 | 0 | 0 | 115 s |
| `bash tests/test-pre-bash-guard.sh`(テストを足したあと) | 1576 | 1576 | 0 | 0 | 110 s |
| `bash tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | 8 s |
| ubuntu:24.04、mawk 1.3.4、dash、GNU sed 4.9・grep 3.11、jq なし: guard / lib_json | 1565 / 126 | 785 / 87 | 0 / 0 | 780 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard / lib_json | 1576 / 126 | 1576 / 126 | 0 / 0 | 0 / 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1576 / 126 | 1576 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-test.sh` 2 回目(テストを足したあと) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 373 s |
| `./scripts/run-verify.sh`(mode all、scope full) | 静的な検査 + shell 71 本 + Go | すべて | 0 | 0 | 409 s |

`run-verify.sh` の中で、shellcheck(hook と verify のスクリプト)、全 hook の `sh -n`、`settings.json` の `jq -e`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、gofmt、golangci-lint(0 issues)、branch の secret scan(f423f230..adb7eda3、clean)が通り、最後に `All verifiers passed.` が出た。AC6 の 5 つの検査はこれで確かめた。

## Coverage

- Statement: 計測していない(shell に計測の道具がない)
- Branch: 計測していない
- Function: 計測していない
- Notes: 代わりに mutation で測った。guard の 39 個と `lib_json.sh` の 5 個、計 44 個の置換を、テストのコピーと hook のコピーで流した(worktree のファイルは変えていない)。テストを足す前は 36 個が赤になり、8 個が全件緑のまま残った。8 個と、深さの上限の例でしか赤にならなかった 3 個をもとにテストを足し、11 個を流し直すと 11 個とも赤になった。44 個すべてが赤になる

### Mutation の結果

数字は赤になったテストの件数(jq あり・なしの 2 経路を別に数える)。「足したあと」は流し直した 11 個だけ。ほかの 33 個は、足す前から赤にしていたテストを消していないので結果は変わらない。

| Mutation | 何を壊したか | 足す前 | 足したあと |
| --- | --- | --- | --- |
| G01 | 見張り(`sentinel()` の呼び出し)を外す | 220 | — |
| G02 | パイプの後段を見ずにデータ区間にする | 14 | — |
| G03 | 読み直しの深さの上限(MAXD 4 → 99) | 4 | — |
| G04 | 入れ子の上限(RMAX 24 → 100000) | 6 | — |
| G05 | 読み直しのテキストの上限(QMAX) | 2 | — |
| G06 | awk の終了コードを捨てる(`\|\| true`) | 8 | — |
| G07 | 推奨の HEREDOC の形の末尾 `)"` を確かめない | 2 | — |
| G08 | 見張りの語の境界から `.` と `-` を外す | 4 | — |
| G09 | ファイルへのリダイレクトを安全とみなす | 28 | — |
| G10 | 入れ子の中もデータ区間にする | 28 | — |
| G11 | 置換のある引用符なしのヒアドキュメントもデータにする | 2 | — |
| G12 | 読むだけでないコマンドのヒアドキュメントもデータにする | 10 | — |
| G13 | バッククォートの中を読み直さない | **0** | 6 |
| G14 | 深さ 1 以上の単純コマンドを判定しない | 4(深さと上限の例だけ) | 40 |
| G15 | `sh -c` の文字列を読み直さない | **0** | 6 |
| G16 | `eval` の引数を読み直さない | 2(深さの例だけ) | 4 |
| G17 | パイプで shell に流す前段の引数を読み直さない | 2(上限の例だけ) | 4 |
| G18 | `core.hooksPath` のキーの大文字小文字を区別する | 6 | — |
| G19 | commit の短いフラグの束の `n` を止めない | 16 | — |
| G20 | 長いオプションの省略を 3 文字から受ける | 2 | — |
| G21 | `+` で始まる refspec を止めない | 6 | — |
| G22 | zsh の先頭の `=` を落とさない | 2 | — |
| G23 | 前置きのコマンドの後ろの語を見ない(`scan_words`) | **0** | 6 |
| G24 | 読むだけのコマンドの一覧に `sort` を足す | 2 | — |
| G25 | 引数の `>(...)` でデータ区間を切らない(POUT) | **0** | 2 |
| G26 | `rg --pre` を読むだけのコマンドとして扱う | 8 | — |
| G27 | awk がないときの代替規則から sudo を外す | 3 | — |
| G28 | `git commit -F -` へのヒアストリングの置換を止めない | **0** | 4 |
| G29 | `git commit -F -` へのヒアドキュメントの置換を止めない | 12 | — |
| G30 | `git tag` の規則を外す | 2 | — |
| G31 | 前置きの後ろの `git commit -m` もデータにする | 2 | — |
| G32 | データ区間の索引を最初の 512 字の区画だけにする | **0** | 4 |
| G33 | `find_str` が 512 字の窓の境目をまたぐ一致を見落とす | **0** | 4 |
| G34 | ダブルクォートの中の置換の印を立てない | 87 | — |
| G35 | push の短いフラグの束の `f` を止めない | 8 | — |
| G36 | reset の規則を外す | 20 | — |
| G37 | コメントをデータにしない | 10 | — |
| G38 | 先頭の予約語を読み飛ばさない | 2 | — |
| G39 | `env -S` の文字列を読み直さない | **0** | 2 |
| L01 | `lib_json.sh`: `\n` を戻さない | guard 34 / lib 24 | — |
| L02 | `lib_json.sh`: `\uXXXX` を戻さない | guard 2 / lib 36 | — |
| L03 | `lib_json.sh`: 16 進の大文字を受けない | lib 4 | — |
| L04 | `lib_json.sh`: `\\` を戻さない | guard 9 / lib 32 | — |
| L06 | `lib_json.sh`: jq も awk もないときの sed の戻しを外す | lib 1 | — |

残っていた 8 個のうち 6 個(G13、G14、G15、G16、G17、G23、G39 の読み直しの系統)には共通の原因があった。テストの例はどれも旧版の文字列(`git push --force`、`sudo `、`git reset --hard`)を含むので、字句解析の読み直しを止めても見張りが同じ例を deny にする。字句解析だけが止める形(`git push origin --force`、`git -C x reset --hard`、`--no-verify`)を読み直しの場所に置いた例がなかった。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `tests/test-secret-scan.sh`(1 回目の `run-test.sh`) | `AC-15: unresolved range names the unscanned range (missing in stderr: could not scan does-not-exist..feature)`。stderr のファイルは空だった | このテストは出力を固定のパス `/tmp/ralph-secret-scan-test.out` と `.err`(21〜41 行目と 116 行目)に書き、終わるときに消す。同じテストが同時に 2 つ動くと、互いのファイルを上書きし、消し合う。2 つを同時に流すと、どちらも rc 1 で、2 件ずつ別の assertion が落ちた(`AC-7`、`AC-9`、範囲の終わりの件)。単独では FAIL 0。scanner は exit 3 のときに必ず `could not scan` を出す(`scripts/secret-scan.sh:284`、`:344`)ので、空のファイルは scanner の不具合ではない。1 回目の実行と同時に、このセッションの外で同じテストが動いていたと考えられる。未確認です。この branch は `scripts/secret-scan.sh` にも、このテストにも触れていない | このテストの一時ファイルを `$workdir` の下に移す(`mktemp -d` はすでにある)。この PR の範囲の外なので直していない。2 回目の `run-test.sh` と `run-verify.sh` は他の実行と重ならないように流し、どちらも通った |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1: どのモードでも ask を返さない | 通る | A 節。旧 ask 行 44 件を 4 モードと 2 経路で none。ask や想定外の出力は 1576 件のどこにもない(`decide()` が ask を `got=ask` として落とす) |
| AC2: 列挙した形がどのモードでも deny | 通る | B 節(モードなしと `bypassPermissions`、2 経路) |
| AC3: 列挙した形がどのモードでも none | 通る | C 節 |
| AC4: jq がなくても同じ判定。`lib_json.sh` がエスケープを戻す | 通る | guard の全件が jq あり・なしの両方で期待値と一致。`tests/test-lib-json.sh` 126/0。mutation L01〜L06 はどれも赤になる |
| AC6: 5 つの検査 | 通る | `run-verify.sh` rc 0(上の Test execution) |
| AC7: 旧版の deny は新版でも deny、例外は AC3 に挙げたものだけ | 通る | G 節が 2 経路とも通り、例外は `intentional_fixes` の 13 件と一致する。verify の V-1 を受けて、語の境界の差(`my-sudo ls`、`x.sudo ls`)を G 節で別に固定した(下の「足したテスト」) |
| AC8: 200 KB のコマンドが macOS の awk で 5 秒以内 | 通る | H 節、BWK awk 20200816、jq なし: 0.28・1.00・0.61 秒(2 回目の `run-test.sh`)。単独の実行では最大 1.63 秒、mutation を 3 本並べて流した負荷の下でも最大 1.29 秒。ubuntu の mawk では 0.07・0.38・0.19 秒 |
| AC9: 止める側に倒す場合 | 通る | F 節。mutation G03〜G06 と G27 が赤になる |
| PR #206 の deny の行 | 通る | A 節の `pr206_deny` 10 件を 4 モードで deny |
| `post_edit_verify.sh`(`lib_json.sh` のもう 1 つの利用者) | 通る | `tests/test-post-edit-verify.sh` FAIL 0(2 回とも) |
| awk の方言 | 通る | ubuntu の mawk と gawk で guard 1576/0、lib_json 126/0。mawk・dash・jq なしでも 785/0 |

### 足したテスト(`tests/test-pre-bash-guard.sh`、68 件)

guard と `lib_json.sh` は変えていない。足したのはテストだけで、どれも今の guard の挙動を固定する。

- D 節の `edge_deny` に、字句解析だけが止める形を読み直しの場所ごとに 21 形: `$(...)`、バッククォート(ダブルクォートの中も)、代入の中の `$(...)`、`<(...)`、`${x:-$(...)}`、`sh -c`、`bash -lc`、`eval`、`env -S`、`env --split-string=`、`bash <<<`、`| sh`、`sh <<'EOF'`、`cat <<'EOF' | sh`、引用符なしのヒアドキュメントの `$(...)` とバッククォート、`xargs sh -c`、`find -exec git`、`flock … git`、`watch git`。旧版はどれも none で、新版で増えた検出にあたる(plan の Design decisions は字句解析の新しい検出を残すと決めている)
- `edge_deny` に `git commit -F - <<< "$(id)"` とバッククォートの形、`edge_none` に `git commit -F - <<< '$(id)'`(G28)
- `edge_sentinel_deny` に `echo sudo ls >(sh)`(G25。shell で動かしても無害だが、旧版も deny にしていた形)と、tech-debt の guard の行の (a) に挙がっている誤検知 3 形(`apt-get remove sudo -y`、`bash -c 'x' sudo ls`、`flock l git grep sudo file`)。3 形は旧版も deny にしていたので AC7 に反しない。直したら `edge_none` に移すとコメントに書いた
- `check_named`(長い例に短いラベルを付けるだけの補助)で 4 件: 600 字を超えるデータ区間の後ろの一致は none(`echo` と `git commit -m`、G32)、512 字の窓の境目をまたぐ見張りの一致は deny(505 字目からの `git push --force`、511 字目からの `sudo`、G33)
- G 節に、旧版が `my-sudo ls` と `x.sudo ls` を deny にすることを 2 経路で固定した(新版の none は D 節が固定している)。AC3 の配列は plan の AC3 と同じに保つため変えていない。verify の V-1 の勧め (a) を、AC3 に手を入れない形にしたもの

## Test gaps

- 文字列を実行するコマンド(`watch`、`find -exec sh -c`、`su -c` など)に、旧版の文字列を含まない形を渡すと、旧版も新版も通す。`watch 'git push origin --force'` は jq あり・なしとも none(evidence の probe)。旧版より弱くはないので AC7 の対象ではない。header のコメントは「字句解析が追えない文字列は旧版と同じく止める」と書いていて、止めるのは旧版の 4 つの文字列に限られる。tech-debt の guard の行 (b) にこの種類を書き足すとよい(/sync-docs の担当)
- AC8 の上限: テストの H 節は 10 秒で、AC8 の 5 秒より緩い(verify の V-2)。実測は最大 1.63 秒。テストは変えていない
- busybox の awk(alpine)では流していない
- Claude Code が実際に呼ぶ hook での効き目は、merge までは確かめられない。このセッションで効いているのは main のチェックアウトの旧版
- 計測したカバレッジはない。mutation の 44 個は手で選んだもので、すべての分岐を網羅するものではない
- `tests/test-secret-scan.sh` の固定パスによる取り違え(上の Failure analysis)は、この PR の外の既存の問題として残る

## Verdict

- Pass: yes。`./scripts/run-test.sh`(2 回目、shell 40 本と Go 8 パッケージ)、`./scripts/run-verify.sh`、`tests/test-pre-bash-guard.sh` 1576/0、`tests/test-lib-json.sh` 126/0、ubuntu の mawk・gawk。mutation 44 個がすべて赤になる。AC1〜AC4、AC6〜AC9 の振る舞いの側を確かめた。/sync-docs に進んでよい
- Fail: なし。1 回目の `run-test.sh` の `tests/test-secret-scan.sh` の 1 件は、テストの固定パスに同時実行が重なったもので、この diff とは関係しない。単独と 2 回目の実行では通った
- Blocked: なし
