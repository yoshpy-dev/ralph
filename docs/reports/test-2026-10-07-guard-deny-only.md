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

---

## pipeline cycle 2(cross-review の指摘の修正のあと)

- Date: 2026-10-08
- Tester: tester subagent (Claude)。pipeline の 2 周目で、上限 2 の最後の周
- Scope: `git diff origin/main...HEAD`(merge-base f423f230、HEAD 63b6743a)と、この周で `tests/test-pre-bash-guard.sh` に足したテスト。この周のコードの変更は 46806dc9(guard とテスト)と、0ef6fc4f・63b6743a(guard のヘッダーのコメントだけ)
- 上の cycle 1 の節と、その `## Verdict` は、その時点(HEAD adb7eda3)の判定として残した。いまの判定は、この節の最後の `## Verdict(pipeline cycle 2)` にある
- Evidence: `docs/evidence/test-2026-10-07-guard-deny-only.log` の末尾(「pipeline cycle 2」の見出しから)に足した。gitignore の対象なので commit しない。2 回の `run-test.sh`、`run-verify.sh`、mutation の置換・結果・落ちた assertion、probe、shell での確かめ、Docker の出力が入っている

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 1 回目(テストを足す前、63b6743a) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 396 s |
| 1 回目の中の `tests/test-pre-bash-guard.sh` | 1688 | 1688 | 0 | 0 | — |
| ubuntu:24.04、mawk 1.3.4、dash、jq なし: guard / lib_json(テストを足したあと) | 1693 / 126 | 849 / 87 | 0 / 0 | 844 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard / lib_json | 1704 / 126 | 1704 / 126 | 0 / 0 | 0 / 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1704 / 126 | 1704 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-test.sh` 2 回目(テストを足したあと) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 424 s |
| 2 回目の中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1704 / 126 | 1704 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh`(mode all、scope full、テストを足したあと) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 434 s |

- `run-test.sh` は 2 回とも、`lib_json.sh` を分類できないので full にフォールバックした(`Language scope: full fallback (unclassified:.claude/hooks/lib_json.sh)`)。Go のテストは `internal/org` だけが実際に走り、ほかはキャッシュから
- `tests/test-secret-scan.sh` は 2 回の `run-test.sh` と `run-verify.sh` のどれでも通った。cycle 1 の固定パスによる取り違えは、この周では起きなかった。3 回とも、ほかの実行と重ならないように流した
- `run-verify.sh` は `All verifiers passed.` で終わった。中で shellcheck、全 hook の `sh -n`、`settings.json` の `jq -e`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、gofmt、golangci-lint(0 issues)、branch の secret scan(f423f230..63b6743a、clean)が通った
- AC8(H 節、macOS の BWK awk、jq なし): 1 回目 0.285・1.026・0.637 秒、2 回目 0.317・1.063・0.609 秒。ubuntu の mawk では 0.071・0.369・0.182 秒

### Mutation の結果(cycle 2)

46806dc9 が足した仕組みを 1 つずつ止める置換を 19 個作り、hook のコピーで流した(worktree のファイルは変えていない)。置換が当たったことは `diff` の hunk の数で、awk の文法は取り出したプログラムを awk に読ませて確かめた。数字は落ちた assertion の件数で、jq あり・なしの 2 経路を別に数える。「足す前」は 63b6743a のテスト(1688 件)、「足したあと」はこの周で足したテスト(1704 件)で流し直した 6 個だけ。

| Mutation | 依頼の項目 | 何を壊したか | 足す前 | 足したあと |
| --- | --- | --- | --- | --- |
| N01 | `(` の NODATA | トップレベルの `(` で NODATA を立てない | 14 | — |
| N02 | `(` の NODATA(閉じ側) | 対応のない `)` で NODATA を立てない | **0** | 0(等価。下の注) |
| N03 | RESW の NODATA | 先頭の予約語と `{` `}` で NODATA を立てない | 26 | — |
| N04 | exec の NODATA | リダイレクトのある `exec` で NODATA を立てない | 6 | — |
| N06 | 3 つまとめて | `in_data()` が NODATA を見ない | 50 | — |
| H01 | 文脈ごとのヒアドキュメント | 待ちのヒアドキュメントを 1 つの列にまとめ、どの文脈の改行でも本文を読む(46806dc9 の前の読み方) | 6 | — |
| J01 | 行継ぎのつなぎ | 本文の行末の `\<改行>` をつながない | 10 | — |
| J02 | 行継ぎのつなぎ | つないだ本文にもデータ区間を与える(`joined` を見ない) | **0** | 6 |
| J03 | 行継ぎのつなぎ | つないだ行を `<<-` の区切りと比べるときにタブを落とさない(`lstrip_tabs`) | **0** | 2 |
| J04 | 行継ぎのつなぎ | 3 行以上をつながない(内側のループを止める) | **0** | 2 |
| L01 | LW_BSNL | 区切りの語の `\<改行>` を記録しない | 6 | — |
| L02 | LW_BSNL | `LW_BSNL` を区切りの引用符の判定(`HQ`)に使わない | **0** | 2 |
| L03 | LW_BSNL | 区切りに `\<改行>` がある本文にもデータ区間を与える(`HBSNL` を見ない) | **0** | 2 |
| R01 | redir_safe の fd の範囲 | `>&` の複製を、数字なら何でも安全とみなす(46806dc9 の前の範囲) | 6 | — |
| R05 | redir_safe の fd の範囲 | `>&` の行き先をすべて安全とみなす | 12 | — |
| R02 | `&>` の読み分け | `&>` を `>` として読む(46806dc9 の前の読み方) | 6 | — |
| R03 | `&>` の読み分け | `&>` と `&>>` の `/dev/null` を安全とみなす | 6 | — |
| P01 | printf -v | `printf -v` を読むだけのコマンドから外さない | 10 | — |
| P02 | printf -v | 離した `-v c` だけを見て、つけた `-vc` を見ない | 6 | — |

依頼の 8 項目は、どれも主な置換(N01、N03、N04、H01、J01、L01、R01、P01)が足す前のテストで赤になった。どの置換も、落ちたのはその仕組みのための B 節の例と、G 節の AC7 の比較だった(evidence の 15)。仕組みの一部だけを止める 5 個(J02、J03、J04、L02、L03)は緑のまま残ったので、テストを足して赤にした。

N02 は等価の置換で、テストでは赤にできない。字句解析は `(` と `)` で語を切り、トップレベルの `(` はすべて深さを上げて NODATA を立てる。深さ 0 で対応のない `)` が出るのは case のパターンだけで、case のコマンドには `case` と `esac` がコマンドの位置にあり、そこで RESW が NODATA を立てる。RESW を止めた N03 では `case x in x) echo 'sudo ls';; esac | sh` が deny のままで、`)` と RESW を両方止めた N06 では赤になった。2 つは case の形で重なっていて、`)` だけが判定を決める正しいシェルの入力はない。対応のない `)` だけの入力は構文エラーになり、何も実行されない。

### 足したテスト(cycle 2、`tests/test-pre-bash-guard.sh`、6 形・16 件)

guard と `lib_json.sh` は変えていない。どの例も今の guard の挙動を固定する。shell での動きは、危ない部分を `echo LINE-RAN` や `echo SUBST-RAN >&2` に置き換えて、macOS の bash 3.2、zsh 5.9、dash で確かめた(evidence の 17)。

- B 節の `guard_deny_only_forms` の 2 に、verify の V2-4 の形(`cat <<'OUT'` の行末にバッククォートを開き、次の行に `sudo ls`、その次の行でバッククォートを閉じ、`x`、`OUT` と続く複数行のコマンド。probe の `v-ml2.txt` とバイト単位で同じ)。3 つの shell とも、バッククォートの中を実行し、本文は `x` の行だけになる。46806dc9 の前の guard(a3103e91 の時点)も deny で、19 個の置換のどれでも deny のままだった。バッククォートの中は待ちの列に入れてあとで読むので、その中の改行は `read_heredocs()` に届かない。バッククォートの読み方を変えたとき(たとえば `$(...)` と同じようにその場で読むとき)に、この例が効く。旧版も deny なので、AC7 の比較の例の集まりに入る
- B 節の `guard_deny_only_forms` の 3 に `cat <<EOF`、`$\`、`(sudo ls)`、`EOF` の 4 行(J02 を赤にする)。引用符のない本文では `\<改行>` を取ってから展開するので、`$` と `(` が 1 つのコマンド置換になり、3 つの shell とも実行した。46806dc9 の前の guard は none だった。`joined` の印がこの形を止めているのに、テストに例がなかった。旧版も deny
- D 節の `edge_deny` に、字句解析だけが止める形を 3 つ: `<<-EOF` の本文でタブ・`EO\`・`F` とつないだ終わりの行のあとの `git push origin --force`(J03)、`E\`・`O\`・`F` の 3 行でつないだ終わりの行のあとの `git push origin --force`(J04)、区切りの語を `EO\<改行>F` と書いた本文の `$(git push origin --force)`(L02)。J03 と J04 の形は、bash と zsh がつないだ行を区切りと比べるので最後の行を実行する。dash はつないだ行を区切りとみなさず、最後の行は本文の文字になる(guard は bash と zsh に合わせているので、dash に対しては止めすぎの向き)。L02 の形は 3 つの shell とも置換を実行した。旧版はどれも none
- D 節の `edge_sentinel_deny` に、区切りの語を `EO\<改行>F` と書いた本文の `sudo ls`(L03)。区切りの語に `\<改行>` があると、本文はデータ区間を持たない(guard のヘッダーの (6))。3 つの shell はこの本文を表示するだけなので、この deny は誤検知で、ヘッダーに書いてあるとおりの止めすぎにあたる。本文にデータ区間を与えるように変えたら `edge_none` に移す、とコメントに書いた
- ファイルの先頭の D 節の説明に「heredoc terminators and delimiters joined by a backslash-newline」を足した

### Failure analysis(cycle 2)

テストスイートの中に落ちたものはない。テストファイルの外の形で、AC7 に反する穴を 1 つ見つけた。guard は変えないよう依頼されているので、ここに記録する。

| ID | Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- | --- |
| F2-1(HIGH、AC7 違反) | ダブルクォートの中で `$` と `(` のあいだに `\<改行>` を挟んだ形。新版が none、旧版(fixture)が deny になったもの(jq あり・なし、root と template のどれでも同じ): `echo "$\<改行>(sudo ls)"`、`echo "x$\<改行>(sudo ls)"`、`\<改行>` を 2 つ挟んだ形、`printf '%s' "$\<改行>(sudo ls)"`、`grep "$\<改行>(sudo ls)" file`、`echo ... > /dev/null`、`echo ... \| grep x`、中身が `git push --force` と `git reset --hard` の形、`git commit -m "$\<改行>(sudo ls)"`、`git tag -a v1 -m "$\<改行>(sudo ls)"` | guard は none を返す。`bash -c` と `dash -c` は `\<改行>` を取ってから読むので、`"$\<改行>(echo SUBST-RAN)"` で置換を実行した(macOS の bash 3.2、dash)。zsh 5.9 は実行せず、そのまま表示した。bash で Bash の呼び出しを動かす環境では `sudo ls` が走る。Claude Code と Codex がどの shell で動かすかは、ここでは確かめていない | `lex_dollar()`(`.claude/hooks/pre_bash_guard.sh:382-403`)は `$` の次の 1 文字(`:384`)だけで `(` と `{` を判定する。`lex_dq()` は `$` を先に `lex_dollar()` に渡し(`:362`)、`\<改行>` はそのあとで捨てる(`:358`)。`$` はただの文字になり、続く `(sudo ls)` は引数の文字として読まれる。引数に置換がないので (a) のデータ区間になり、見張りの一致がその中に入る。46806dc9 の前の guard(a3103e91)も同じ結果なので、データ区間を入れた S2c(df0a50d5)からある穴で、cycle 1 の /test と /verify は見落とした。引用符のない語は `(` で語が切れてトップレベルの `(` が NODATA を立てるので deny になり、引用符のないヒアドキュメントの本文は `joined` が止める(上で足したテスト) | guard の変更が要る(この /test ではしていない)。(1) `lex_dollar()` の先頭で、`$` の直後の `\<改行>` を読み飛ばしてから `(` と `{` を見る。`lex_dq()`・`lex_brace()`・`lex_hd()`・`lex_word()` はどれもここを通る。(2) 止める側に倒し、ダブルクォートの中の `\<改行>` を置換ありとみなす。直すときは、B 節の `guard_deny_only_forms` に `$'echo "$\\\n(sudo ls)"'`、`$'grep "$\\\n(sudo ls)" file'`、`$'git commit -m "$\\\n(sudo ls)"'` を同じ commit で足す(いまは赤になるので、この周では commit していない)。(1) で直すなら、D 節の `edge_deny` に字句解析だけが止める `$'echo "$\\\n(git push origin --force)"'`、`$'git commit -m "$\\\n(id)"'`、`$'cat <<EOF\n$\\\n(git push origin --force)\nEOF'` も足せる(この 3 つは旧版も none) |

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1: どのモードでも ask を返さない | 通る | A 節。1704 件のどこにも ask や想定外の出力はない |
| AC2: 列挙した形がどのモードでも deny | 通る | B 節。この周で足した 2 形も、2 モードと 2 経路で deny |
| AC3: 列挙した形がどのモードでも none | 通る | C 節の 29 件 |
| AC4: jq がなくても同じ判定 | 通る | 1704 件で jq あり・なしが期待値と一致。`tests/test-lib-json.sh` 126/0。ubuntu の jq なしでも guard 849/0 |
| AC6: 5 つの検査 | 通る | `run-verify.sh` rc 0 |
| AC7: 旧版の deny は新版でも deny、例外は AC3 に挙げたものだけ | テストの例の集まりでは通る。テストの外に反例がある | G 節は 2 経路とも、旧版 deny から新版 none に変わる形が `intentional_fixes` の 13 件と一致した(足した 2 形は旧版・新版とも deny)。テストにない F2-1 の形で、旧版 deny から新版 none に変わる |
| AC8: 200 KB のコマンドが 5 秒以内 | 通る | H 節の最大 1.063 秒(BWK awk、jq なし) |
| AC9: 止める側に倒す場合 | 通る | F 節 |
| cross-review と consult が挙げた穴の型(46806dc9) | 通る | B 節の `guard_deny_only_forms` 28 形が 2 モードと 2 経路で deny。8 つの仕組みはどれも置換で赤になる |
| `post_edit_verify.sh`(`lib_json.sh` のもう 1 つの利用者) | 通る | `tests/test-post-edit-verify.sh` FAIL 0(2 回の `run-test.sh` と `run-verify.sh`) |
| awk の方言 | 通る | ubuntu の mawk と gawk で guard 1704/0、lib_json 126/0 |

### Test gaps(cycle 2)

- F2-1 の形はテストにない。赤になるので、guard を直す commit と一緒に入れる(上の Proposed fix)
- `\<改行>` で `$` と `(` を分けた形のうち、旧版の 4 つの文字列を含まないもの(ダブルクォート、引用符のないヒアドキュメントの本文、`git commit -m "..."`、`git commit -F -` へのヒアドキュメント)は、新版も旧版も none。旧版より弱くはないので AC7 の対象ではないが、字句解析は追えていない
- N02 は等価の置換で、テストで赤にできない(上の注)
- dash との違い: つないだ行を区切りとみなす読み方は bash と zsh に合わせていて、dash ではその行のあとも本文になる。guard は dash に対して止めすぎる向きなので、穴にはならない
- 止めすぎの形: L03 の形(テストで固定した)と、本文の途中の行をつないだ形(`cat <<EOF` の本文の `x\` のあとの `EOF` は 3 つの shell とも本文の文字なのに、その次の行の `sudo ls` が deny)。後者はテストに入れていない
- busybox の awk(alpine)では流していない。Claude Code が実際に呼ぶ hook での効き目は、merge までは確かめられない(このセッションで効いているのは main のチェックアウトの旧版)。計測したカバレッジはなく、19 個の置換は手で選んだもの
- `tests/test-secret-scan.sh` の固定パスの問題は、この PR の外の既存の問題として残る(この周では起きなかった)

## Verdict(pipeline cycle 2)

- Pass: no。テストスイートは通る(`./scripts/run-test.sh` 2 回、`./scripts/run-verify.sh`、`tests/test-pre-bash-guard.sh` 1704/0、`tests/test-lib-json.sh` 126/0、ubuntu の mawk・gawk)。46806dc9 の 8 つの仕組みは、どれも置換で赤になる(19 個のうち 18 個が赤、残る 1 個は等価)。それでも F2-1 は AC7 の反例で、bash と dash では `sudo ls` が走る形なので、pass にはしない
- Fail: F2-1(HIGH)。guard の変更が要る。pipeline は上限 2 の 2 周目なので、上限を上げて直すか、既知の穴として記録して /pr に進むかは、orchestrator と人が決める。直す場合は、Proposed fix のテストを同じ commit で入れる
- Blocked: なし

---

## pipeline cycle 2 のやり直し(F2-1 の修正のあと)

- Date: 2026-10-08
- Tester: tester subagent (Claude)
- 経緯: 上の cycle 2 の /test(6cca4ce4)は F2-1 で fail だった。guard を 4e829e34 で直し、self-review(373fa29d)と verify(3950ffdd)をやり直して、どちらも通った。この節はそのあとにやり直した /test の結果で、cycle 2 の中のやり直しなので cycle の数は 2 のまま。上の 2 つの Verdict は、それぞれの時点の判定として残した。いまの判定は、この節の最後の `## Verdict(pipeline cycle 2 のやり直し)` にある
- Scope: `git diff 6cca4ce4..4ffe74fe` と、この /test で足したテスト。コードの変更は 4e829e34 だけで、END でコマンドに `\<改行>` があれば NODATA を立てる 1 行と、B 節の 8 の 3 形を足した。ほかの commit はコメントと記録
- Evidence: `docs/evidence/test-2026-10-07-guard-deny-only.log` の末尾(21 節から)。2 回の `run-test.sh`、2 回の `run-verify.sh`、probe、mutation の置換・結果・落ちた assertion、shell での確かめ、Docker の出力

### Test execution(cycle 2 のやり直し)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 1 回目(4ffe74fe、テストを足す前) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 427 s |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1716 / 126 | 1716 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 1 回目(mode all、scope full、テストを足す前) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 641 s |
| ubuntu:24.04、mawk 1.3.4、dash、jq なし: guard / lib_json(テストを足したあと) | 1719 / 126 | 862 / 87 | 0 / 0 | 857 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard / lib_json | 1730 / 126 | 1730 / 126 | 0 / 0 | 0 / 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1730 / 126 | 1730 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-test.sh` 2 回目(テストを足したあと) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 406 s |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1730 / 126 | 1730 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 2 回目(テストを足したあと) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 434 s |

- `run-test.sh` は 2 回とも `lib_json.sh` を分類できず full にフォールバックした。Go は `internal/org` だけが実際に走り、ほかはキャッシュから
- `run-verify.sh` は 2 回とも `All verifiers passed.` で終わった。中で shellcheck(`tests/test-*.sh` を含む)、全 hook の `sh -n`、`settings.json` の `jq -e`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、gofmt、golangci-lint(0 issues)、branch の secret scan(2 回とも f423f230..4ffe74fe で clean。足したテストはその時点で commit していないので、push の前の `./scripts/secret-scan-branch.sh --strict` で見る)が通った
- `tests/test-secret-scan.sh` は 4 回とも通った。ほかの実行と重ならないように流した
- AC8(H 節、macOS の BWK awk、jq なし)の最大は 1.165 秒(1 回目の `run-verify.sh`)。ubuntu の mawk では 0.074・0.373・0.189 秒

### F2-1 の形の確かめ(cycle 2 のやり直し)

39 個の probe ファイルを、新版(root と template)と旧版(root の fixture と、1c4cea5a の template)に、それぞれ jq あり・なしで読ませた(1 ファイルにつき 8 通り)。39 個は、上の F2-1 の表の形、依頼元の `xr3/f1.txt`〜`f7.txt`、対照の形、字句解析だけが止める形で、どれも期待どおりだった(evidence の 23)。

- F2-1 の 11 形(`echo`・`grep`・`printf`・`git commit -m`・`git tag -a -m`、`x$` の前置き、`\<改行>` 2 つ、`> /dev/null`、`| grep x`、中身が `git push --force` と `git reset --hard`)と `xr3` の 7 個: 新版・旧版とも 8 通りすべて deny。6cca4ce4 の guard では、このうち F2-1 の 11 形と `f1`・`f3` が none だった(`f2`・`f4`〜`f7` は 6cca4ce4 でも deny)
- 文字だけの `\<改行>` の 3 形(単一引用符の中の `$\<改行>(...)`、`$` のないダブルクォート、`$\<改行>((1+1))` のあとの引数): 6cca4ce4 では none、いまは旧版と同じ deny。shell は何も実行しない
- 字句解析だけが止める 4 形(`echo "$\<改行>(git push origin --force)"`、`git commit -m "$\<改行>(id)"`、`git commit -F -` へのヒアドキュメントの `$\<改行>(id)`、`cat <<EOF` の本文の `$\<改行>(git push origin --force)`。probe のファイルは 5 個で、`git commit -m` の形が 2 個ある): 新版も旧版も 8 通りすべて none。上の cycle 2 の Test gaps に書いた既知の穴のまま
- `\` と改行を JSON の `\`・`\u000a` で書いた F2-1 の形(5 通り)も、jq あり・なしとも deny
- CRLF の `echo "$\<CR><LF>(sudo ls)"` は新版 none・旧版 deny。bash・zsh・dash とも置換を実行せず文字として表示する(evidence の 23)ので、データ区間の中の言及にあたり、穴ではない

### Mutation の結果(cycle 2 のやり直し)

4e829e34 の 1 行に対する置換を 3 個足し、cycle 2 の 19 個と合わせて 22 個を、4ffe74fe のテスト(1716 件)で流した。数字は落ちた assertion の件数。「cycle 2」は上の cycle 2 の表の最後の値(足したあとの値がない置換は足す前の値)。

| Mutation | 何を壊したか | cycle 2 | 4ffe74fe のテスト | 足したあと(1730 件) |
| --- | --- | --- | --- | --- |
| F01 | END の NODATA の行を消す | — | 14 | 28 |
| F02 | `$` の直後の `\<改行>` だけを見る | — | **0** | 10 |
| F03 | `$`・`\<改行>`・`(` の並びだけを見る | — | **0** | 16 |
| N06 | `in_data()` が NODATA を見ない | 50 | 62 | — |
| J01 | 本文の行末の `\<改行>` をつながない | 10 | 4 | — |
| J02 | つないだ本文にもデータ区間を与える | 6 | **0**(等価) | — |
| J03 / J04 | `lstrip_tabs` を外す / 3 行以上をつながない | 2 / 2 | 2 / 2 | — |
| L01 | 区切りの語の `\<改行>` を記録しない | 6 | 2 | — |
| L02 | `HQ` が `LW_BSNL` を見ない | 2 | 2 | — |
| L03 | `HBSNL` を見ない | 2 | **0**(等価) | — |
| N01・N02・N03・N04・H01・R01・R05・R02・R03・P01・P02 | cycle 2 と同じ | 14・0・26・6・6・6・12・6・6・10・6 | 同じ | — |

- F01 で落ちたのは、B 節の 8 の 3 形(2 モード × 2 経路で 12 件)と、G 節の AC7 の比較 2 件だった。新しい 3 形は NODATA の行を消すと赤になる
- F02 と F03 は、4e829e34 の判定を狭めた置換で、4ffe74fe のテストでは緑のままだった。F03 は `echo "$\<改行>\<改行>(sudo ls)"` を通す。bash 3.2 と dash はこれを置換として実行する(zsh は実行しない)ので、F2-1 と同じ穴が開く。F02 が通すのは文字だけの `\<改行>` で、shell が実行する形はない。ただし guard のヘッダーは、これらも deny になると書いている。下のテストを足し、F02 は 10 件、F03 は 16 件で赤になった
- J02 と L03 は、4e829e34 のあとは等価になった。`joined` と `HBSNL` が変えるのは `HDZ` だけで、`HDZ` は END の `add_data()` にしか使われない。条件の `DCTX` は MAIN かつ深さ 0 の文脈、つまりコマンドそのものの文字を読んでいるときだけ真になる。そのため、`joined` か `HBSNL` が立つときは、コマンドの文字に `\<改行>` があり、END で NODATA が立って `in_data()` は 0 を返す。cycle 2 で J02 と L03 のために足した例は、いまは NODATA の規則で deny になる。2 つの条件はどの判定も変えない(guard は変えていない。4ffe74fe のテストのコメントにも、本文だけの規則はもう判定を変えないと書いてある)
- J01 は 10 件から 4 件、L01 は 6 件から 2 件に減った。以前落ちた例のうち B 節の 3 の形は NODATA で deny になり、残るのは字句解析だけが止める D 節の例(J01 は `<<-EOF` のタブと 3 行つなぎの 2 形、L01 は `EO\<改行>F` の本文の `$(git push origin --force)`)。どちらも赤のまま
- N06 の 12 件の増加は、B 節の 8 の 3 形の分
- 22 個の置換を小さな probe(83 形)でも流した。テストにある形で判定が変わった置換は 17 個で、全件で赤になった 17 個と一致した。F02 と F03 で判定が変わったのは、テストになかった形だけだった(evidence の 27)

### 足したテスト(cycle 2 のやり直し、`tests/test-pre-bash-guard.sh`、6 形・14 件)

commit は 180c7389。guard と `lib_json.sh` は変えていない。どの例もいまの guard と旧版で deny になる。shell での動きは、危ない部分を `echo SUBST-RAN >&2` や `echo LINE-RAN` に置き換えて、macOS の bash 3.2、zsh 5.9、dash で確かめた(evidence の 23)。

- B 節の `guard_deny_only_forms` の 8 に `echo "$\<改行>\<改行>(sudo ls)"`(F03 を赤にする)。bash と dash は置換を実行する
- D 節の `edge_sentinel_deny` に、文字だけの `\<改行>` の 5 形(F02 を赤にする): `$` のないダブルクォートの中、単一引用符の中、推奨のコミットの形(引用符つきの区切りのヒアドキュメントを `git commit -F -` に渡す)の本文の行末、コメントの中、`\\` のあと。3 つの shell はどれも文字として読むので、この deny は誤検知で、guard のヘッダーに書いてある止めすぎにあたる。旧版も deny。データ区間を与えるように変えたら `edge_none` に移す、とコメントに書いた
- ファイルの先頭の D 節の説明に「a backslash-newline anywhere, also one that is only text」を足した
- 6 形は、テストのファイルの文字列を bash で読み、probe のファイルとバイト単位で一致することを確かめた

### Failure analysis(cycle 2 のやり直し)

テストの失敗はない。F2-1 は解消した。probe で調べた形の中に、旧版が deny で新版が none になる、shell が実行する形は残っていない。

### Regression checks(cycle 2 のやり直し)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F2-1(ダブルクォートの中で `$` と `(` を `\<改行>` で分けた形) | 直った | 11 形と `xr3` の 7 個が、新版・旧版、root・template、jq あり・なしのすべてで deny。B 節の 8 の 4 形が 2 モードと 2 経路で deny |
| AC1・AC2・AC3・AC4・AC9 | 通る | A・B・C・F 節。1730 件で ask はなく、jq あり・なしが期待値と一致。`tests/test-lib-json.sh` 126/0 |
| AC6: 5 つの検査 | 通る | `run-verify.sh` 2 回とも rc 0 |
| AC7: 旧版の deny は新版でも deny、例外は AC3 に挙げたものだけ | テストの例の集まりで通る。probe の 39 形でも反例はない | G 節は 2 経路とも、旧版 deny から新版 none に変わる形が `intentional_fixes` の 13 件と一致した |
| AC8: 200 KB のコマンドが 5 秒以内 | 通る | H 節の最大 1.165 秒 |
| `post_edit_verify.sh`(`lib_json.sh` のもう 1 つの利用者) | 通る | `tests/test-post-edit-verify.sh` 24/0 |
| awk の方言 | 通る | ubuntu の mawk と gawk で guard 1730/0、lib_json 126/0 |

### Test gaps(cycle 2 のやり直し)

- 字句解析だけが止める 4 形は、新版も旧版も none のまま。旧版より弱くはないので AC7 の対象ではない。`lex_dollar()` が `$` の直後の `\<改行>` を読み飛ばせば止められる(cycle 2 の Proposed fix の (1))
- J02・L03・N02 は等価の置換で、テストで赤にできない。J02 と L03 の条件(`joined`、`HBSNL`)は、4e829e34 のあとはどの判定も変えない
- 推奨のコミットの形は、本文のどこかの行が `\` で終わると deny になる(旧版も deny)。この止めすぎは D 節で固定した
- busybox の awk(alpine)では流していない。Claude Code が実際に呼ぶ hook での効き目は、merge までは確かめられない。計測したカバレッジはなく、22 個の置換は手で選んだもの
- `tests/test-secret-scan.sh` の固定パスの問題は、この PR の外の既存の問題として残る(この周では起きなかった)

## Verdict(pipeline cycle 2 のやり直し)

- Pass: yes。cycle 2 の /test は F2-1 で fail だったが、guard を 4e829e34 で直し、cycle 2 の中でこの /test をやり直した。`./scripts/run-test.sh` 2 回、`./scripts/run-verify.sh` 2 回、`tests/test-pre-bash-guard.sh` 1730/0、`tests/test-lib-json.sh` 126/0、ubuntu の mawk・gawk が通る。F2-1 の形は新版・旧版とも deny になり、NODATA の行を消す置換は新しい 3 形を赤にする。22 個の置換のうち 19 個が赤、残る 3 個(J02、L03、N02)は等価。/sync-docs に進んでよい
- Fail: なし
- Blocked: なし

## cycle 3 (cap raised to 3)

- Date: 2026-10-08
- Tester: tester subagent (Claude)。cross-review の 2 周目のあと、ユーザーが上限を 3 に上げてから回した /test。cross-review の手順どおり `cycle-count.json` は 2 のまま。上の 3 つの Verdict は、それぞれの時点の判定として残した。いまの判定は、この節の最後の `## Verdict(cycle 3)` にある
- Scope: `git diff facd295b..HEAD -- .claude tests templates`(HEAD 14b3fd37)と、この /test で足したテスト。guard の判定を変えたのは a9ef82b1(許可リスト `data_first_ok`、ヒアドキュメントの区切りの `$`・バッククォート規則、git push の値を取るオプション、git reset の `--` 停止)と 849f5411(reset の `--pathspec-from-file` の値の読み飛ばし)。root と template の guard は `cmp` で同一(`run-verify.sh` の `check-sync.sh` で確認)
- Evidence: `docs/evidence/test-2026-10-07-guard-deny-only.log` の末尾(「cycle 3 (cap raised to 3)」の見出し、31〜45 節)。gitignore の対象なので commit しない
- Guard・lib_json は変更していない(self-review・verify が報告したバグはそのまま持ち越し。下の Test gaps)。足したのは `tests/test-pre-bash-guard.sh` の 11 行だけ

### Test execution(cycle 3)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 1 回目(テストを足す前、14b3fd37) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 723 s(docker・probe と並走した高負荷下) |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1794 / 126 | 1794 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 1 回目(mode all、scope full、テストを足す前) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 1556 s(同上) |
| ubuntu:24.04、mawk 1.3.4、dash、GNU sed 4.9・grep 3.11、jq なし: guard / lib_json(テストを足したあと) | 1811 / 126 | 908 / 87 | 0 / 0 | 903 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard / lib_json | 1822 / 126 | 1822 / 126 | 0 / 0 | 0 / 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1822 / 126 | 1822 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-test.sh` 2 回目(テストを足したあと) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | — |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1822 / 126 | 1822 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 2 回目(テストを足したあと) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | — |

- `run-test.sh` は 2 回とも `lib_json.sh` を分類できず full にフォールバックした。Go は `internal/org` だけが実際に走り(19〜22 s)、ほかはキャッシュから。shell は 40 本すべて緑で、assertion の `FAIL` は 0、0 でない `FAIL:` の要約も 0
- `run-verify.sh` は 2 回とも `All verifiers passed.` で終わった。中で shellcheck(`tests/test-*.sh` を含む)、全 hook の `sh -n`、`settings.json` の `jq -e`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、gofmt、golangci-lint(0 issues)、branch の secret scan が通った
- `tests/test-secret-scan.sh`(固定 `/tmp` パスの既知の flaky)は 2 回とも通った。ほかの実行と重ならないように流した
- 件数の増え方: 14b3fd37 の 1794 から、足した 11 行で 1822。`guard_deny_only_forms`(B 節、`absent` と `bypassPermissions` の 2 モード × jq あり・なしの 2 経路 = 1 行 4 件)に 3 行で +12、`edge_deny`・`edge_none`(D 節、1 モード × 2 経路 = 1 行 2 件)にそれぞれ 3 行・5 行で +6・+10。計 +28。macOS の BWK awk で AC8(jq なし)の最大は 1.049 秒(2 回目)
- `tests/test-lib-json.sh` は変更なし、126/0

### 足したテスト(cycle 3、`tests/test-pre-bash-guard.sh`、11 行)

commit は後述の `test:` コミット。guard と `lib_json.sh` は変えていない。どの行も新版と旧版の両方で期待どおり(新版: 11 行すべて、旧版: deny 6 行は deny、none 5 行のうち push/reset の 4 行は none、`"echo"` の行は旧版 deny→新版 none のデータ区間の例)。各行の形はバイト単位で probe ファイルと一致させた(evidence 42)。

- `guard_deny_only_forms`(B、deny)に 3 行。許可リストを固定する: `$c; echo 'sudo ls'`(変数のコマンドの後ろのデータコマンド)、`>out.txt; echo 'sudo ls'`(リダイレクトだけのコマンドの後ろ)、`echo hi; >out.txt; echo 'sudo ls'`(データコマンドの後ろのリダイレクトだけのコマンド)。self-review C3-5 は、既存の `$c 'sudo ls'` が許可リストを外しても deny(`$c` はデータコマンドでなく、自分の引数にデータ区間がない)なので許可リストを固定しないと指摘した。この 3 行は、後ろのデータコマンドのデータ区間が許可リストで消えることを突く。許可リストを外す変異(AL01)で 3 行とも none になる
- `edge_deny`(D、deny)に 3 行。値を読み飛ばしても後ろの force・hard reset を見逃さないこと: `git push -ofoo --force origin`、`git push --push-option=x --force origin`、`git reset --pathspec-fr -- --hard`。git 2.49 の実測で、前 2 つは forced update、3 つ目は hard reset(evidence 33)。旧版はこの 3 つを通す
- `edge_none`(D、none)に 5 行。値を取るオプションがフラグに見える値を読むこと: `git push -o --force origin main`、`git push --push-option --force origin main`、`git push --push-opt --force origin main`(省略形)、`git push -uof origin main`(git 2.49 はこの 4 つを non-fast-forward で拒否、force ではない。evidence 33)。許可リストが引用符を外した値で 1 語目を比べること: `"echo" 'sudo ls'`(旧版はこれを deny にする、データ区間の例)

### Mutation の結果(cycle 3)

a9ef82b1・849f5411 が変えた判定への変異 27 個(許可リスト 8、区切り 5、push 値オプション 4、push の束ねた `-o` 5、reset 5)に、C3-3 で「許可リストのあとでは判定を変えない」とされた予約語・exec の NODATA 規則と、以前の周の規則を加えた。各変異を使い捨ての写しに当て、system awk で構文を確かめ(evidence 37、構文エラーなし)、テストの A〜D 節から取り出した行(足す前 496 行、足したあと 507 行)に新旧の guard を渡して、期待と違う行の数を数えた(jq の経路。evidence 38〜40)。

| 変異 | 壊したもの | 足す前(14b3fd37) | 足したあと(507 行) | 判定 |
| --- | --- | --- | --- | --- |
| AL01 | 許可リストの NODATA を外す | 12 赤 | 12 赤 | 殺 |
| AL02 | 1 語目の `/` を許す | 0 | 0 | **等価**(`/` を含む値は `git` でも DATACMD の名前でもないので、この検査は冗長) |
| AL03 | リダイレクトだけのコマンドに data を許す | **0** | 2 赤 | 殺(足した redir-only 行) |
| AL04 | 1 語目を生の語で比べる | **0** | 1 赤 | 殺(足した `"echo"` 行) |
| AL05 | `git` を許可リストから外す | 12 赤 | 12 赤 | 殺 |
| AL06 | 許可リストの `DCTX` 条件を外す | 1 赤 | 1 赤 | 殺(既存 `echo "sudo $(date) ls"`) |
| AL07 | 代入を読み飛ばさない | 1 赤 | 1 赤 | 殺(既存 `x=1 echo …`) |
| AL08 | `WN<1` の検査を外す | **0** | 1 赤 | 殺(足した `echo hi; >out.txt; echo …`、前のコマンドの古い語を読む) |
| DL01 | 区切りの規則を丸ごと外す | 1 赤 | 1 赤 | 殺(`cat <<$'\x45'`) |
| DL02 | `$` だけ見る(バッククォートを落とす) | 0 | 0 | **防御的**(バッククォートの区切りで字句解析と食い違う shell がない。下記) |
| DL03 | バッククォートだけ見る(`$` を落とす) | 1 赤 | 1 赤 | 殺(`cat <<$'\x45'`) |
| DL04 | 生の語でなく値を見る | 1 赤 | 1 赤 | 殺(`cat <<$'\x45'`) |
| DL05 | 区切りの規則に `DCTX` 条件を足す | 0 | 0 | **防御的**(入れ子の区切りのテストがない。下記) |
| PU01 | 値オプションの読み飛ばしの行を外す | **0** | 2 赤 | 殺(足した長い値オプション + `--force`) |
| PU02 | 値を読み飛ばさない | **0** | 2 赤 | 殺(同上) |
| PU03 | `=` つきでも次の語も読み飛ばす | **0** | 1 赤 | 殺(足した `--push-option=x --force`) |
| PU04 | 値オプションを省略形で受けない | **0** | 1 赤 | 殺(足した `--push-opt --force`) |
| OC01 | `-o` が末尾のとき次の語を読み飛ばさない | **0** | 1 赤 | 殺(足した `-o --force`) |
| OC02 | `-o` があれば常に次を読み飛ばす | **0** | 1 赤 | 殺(既存 `-o x --force`) |
| OC03 | `f` がどこにあっても force | 1 赤 | 2 赤 | 殺(`-ofoo`、`-uof`) |
| OC04 | `f` が `o` の後ろでも force | 1 赤 | 2 赤 | 殺 |
| OC05 | `o` は先頭のときだけ | **0** | 1 赤 | 殺(足した `-uof`) |
| RS01 | reset の `--` 停止を外す | 3 赤 | 3 赤 | 殺 |
| RS02 | `--pathspec-from-file` の値の読み飛ばしを外す | 1 赤 | 2 赤 | 殺 |
| RS03 | `--pathspec-from-file` を省略形で受けない | **0** | 1 赤 | 殺(足した `--pathspec-fr -- --hard`) |
| RS04 | `=` つきでも次の語を読み飛ばす | 1 赤 | 1 赤 | 殺 |
| RS05 | `--pathspec-from-file` の値を読み飛ばさない | 2 赤 | 2 赤 | 殺 |

- 27 個のうち 24 個が殺、3 個(AL02・DL02・DL05)が生き残る。足す前は 11 個(AL03・AL04・AL08・PU01〜PU04・OC01・OC02・OC05・RS03)が緑のまま(14b3fd37 の時点で穴だった)で、足した 11 行で閉じた
- C3-3 の再測: 予約語の NODATA(N03)と exec の NODATA(N04)を外す変異は、足したあとの 507 行でも判定を変えない(0 赤)。self-review C3-3 のとおり等価。閉じかっこの NODATA(N02、cycle 2 で等価)も 0。開きかっこ(N01、1 赤)、`in_data` の NODATA ゲート丸ごと(N06、34 赤)は殺せる
- 以前の周の規則も再測した: 見張りの force の束ね `f`(C02、6 赤)、force の長いオプション(C03、31 赤)、reset `--hard`(C04、11 赤)、`+refspec`(C05、2 赤)、コミットメッセージのバッククォート(C06、1 赤)はすべて殺せる。字句解析なしの代替規則の `sudo`(C01)は、この行が awk が無いときだけ効くため、awk ありの行の probe では判定を変えない(0)。awk 無しの経路(AC9、F 節)はテストスイートに入っていて `run-test.sh` で緑

### 生き残った 3 個の分析

1. **AL02(等価)**: 許可リストは `index(v, "/")` で `/` を含む 1 語目を弾くが、この検査を外しても判定は変わらない。`v == "git"` も `v in DATACMD` も、`/` を含まない値にしか真にならない(DATACMD の名前に `/` はなく、`git` にもない)ので、`/` を含む値はこの検査がなくても 0 を返す。防御のための冗長な行で、deny-only の入力で区別できない
2. **DL02(防御的、バッククォートの半分)**: 区切りの規則は `LW_RAW` に `$` かバッククォートがあれば NODATA を立てる。`$` の検出は DL01・DL03・DL04 が殺せるので固定されている。バッククォートの検出は、字句解析が区切りの語の生のバッククォート文字列を終端にする一方、shell が違う語を終端にして本文を早く終える、という食い違いを想定した防御。実機で確かめると(evidence 34)、bash 3.2 はバッククォートを含む区切りの語(`cat <<\`echo END\``)を字句どおりに読み、字句解析の終端(生の `` `echo END` ``)と食い違わず、dash は構文エラーにする。zsh もコマンド置換を終端に使わない。どの主要な shell でも「字句解析が本文(データ)と見た行を shell が実行する」食い違いが起きないので、バッククォートの検出を落としても判定が変わる deny-only の形を作れない。既知の gap として記録(下記)
3. **DL05(防御的)**: 区切りの規則に `DCTX`(トップレベル)条件を足す変異。入れ子の文脈(`$(...)` の中など)のヒアドキュメントの区切りに `$`・バッククォートがある形のテストが無いので判定を変えない。NODATA はコマンド全体に効くグローバルなので、入れ子の区切りが NODATA を立てる現状と、立てない変異で判定が変わるのは、入れ子の区切りと同じコマンドのトップレベルにデータ区間があり、かつ shell がその文字を実行する、という極端な形に限られる。テストにも、この周で作った形にも無い。既知の gap として記録(下記)

### Failure analysis(cycle 3)

テストの失敗はない。cross-review の 2 周目の P1 の 3 形(区切りの `$'\x45'`、`env -S`、`builtin exec`)と、許可リストが閉じた 7 形は、B 節の `guard_deny_only_forms` にあり新旧とも deny(前の周と verify cycle 3 で確認済み)。この周で新たに殺した 11 個の変異は、足した 11 行が閉じた。

### Regression checks(cycle 3)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1・AC2・AC3・AC4・AC9 | 通る | `run-test.sh` 2 回。1822 件で ask はなく、jq あり・なしが一致。`tests/test-lib-json.sh` 126/0 |
| AC6: 静的な検査 | 通る | `run-verify.sh` 2 回とも `All verifiers passed.`。root と template の guard は同一 |
| AC7: 旧版の deny は新版でも deny | 通る | 足した deny 6 行は新旧とも deny、`"echo"` 行は旧版 deny・新版 none のデータ区間の例(C 節相当の扱い)。スイートの AC7 corpus(F 節)は `run-test.sh` で緑 |
| AC8: 200 KB のコマンドが 5 秒以内 | 通る | macOS BWK awk で最大 1.049 秒、ubuntu の mawk で 0.071〜0.365 秒 |
| 許可リスト(a9ef82b1)・reset の値(849f5411)の判定 | 固定した | 上の mutation 表。足す前は 11 個の変異が緑だった |
| awk の方言 | 通る | ubuntu の mawk と gawk で guard 1822/0、lib_json 126/0 |

### Test gaps(cycle 3)

- AL02・DL02・DL05 は等価または防御的な変異で、deny-only の形では殺せない(上の分析)。DL02・DL05 は guard のバッククォート・入れ子の区切りの防御で、実機の shell では食い違いが起きないか極端な形に限られる
- self-review・verify が報告した guard の持ち越し(C3-3 の重なった規則、C3-4 と C3-5 のコメントのずれ、`no_verify_rules` の値を取るオプション)は、guard を変えると上限 3 を超えるため直さない。この /test は guard を変えていない
- 字句解析だけが止める形(`$` と `(` を行継続で分けた中身が `(id)` などの形)は前の周からの既知の穴で、この周の変異とは別。新旧とも none
- このセッションで効いている Bash の guard は main のチェックアウトの旧版なので、新版を Claude Code の実際の呼び出しで確かめるのは merge 後になる。計測したカバレッジはなく、変異は手で選んだ。入れ子のヒアドキュメントを使う形は、作っても shell ごとの差が大きく脆いので作っていない
- `tests/test-secret-scan.sh` の固定パスの問題は、この PR の外の既存の問題として残る(この周では起きなかった)

## Verdict(cycle 3)

- Pass: yes。`./scripts/run-test.sh` 2 回(rc 0、`tests/test-pre-bash-guard.sh` 1822/0、`tests/test-lib-json.sh` 126/0、Go 8/8、assertion の FAIL 0)、`./scripts/run-verify.sh` 2 回(rc 0、`All verifiers passed.`)、ubuntu:24.04 の mawk(jq あり・なし)と gawk で guard 1822/0・lib_json 126/0。27 個の mutation のうち 24 個が赤、残る 3 個(AL02 は等価、DL02・DL05 は実機の shell で食い違いが起きない防御的な規則)。足した 11 行で、14b3fd37 の時点で緑だった 11 個の変異(許可リストのデータ区間、push の値オプション、束ねた `-o`、reset の省略形)を閉じ、self-review C3-5(`$c 'sudo ls'` が許可リストを固定しない)に答えた。C3-3 のとおり予約語・exec の NODATA は許可リストのあと等価。guard と `lib_json.sh` は変えていない
- Fail: なし
- Blocked: なし
- Known gaps: AL02(等価)・DL02・DL05(防御的)。guard の持ち越し(self-review C3-1〜C3-5、verify V4-1〜V4-3)は次に guard を変える PR で

## cycle 4 (cap raised to 4)

- Date: 2026-10-08
- Tester: tester subagent (Claude)。cross-review の 3 周目のあと、ユーザーが上限を 4 に上げてから回した /test で、上限 4 の最後の run。`cycle-count.json` は 2 のまま。上の 5 つの Verdict は、それぞれの時点の判定として残した。いまの判定は、この節の最後の `## Verdict(cycle 4)` にある
- Scope: `git diff e5c9e6be..HEAD -- .claude tests templates`(HEAD 6b1f7acc)と、この /test で足したテスト。guard の判定を変えたのは 2 つのコミット。b3c3fdaa は、printf の `%`・`$`・バッククォートの検査、DATACMD から `test`・`[` を外すこと、merge・rebase・am の `--no-verify` の走査を `--` で止めないこと、commit の値を取る長いオプションを `opt_is` で読むことを入れた。12e9a9ad は、printf を書かれたままの語(`WR`)で見ること、rg の `$'`・`$"` の検査、`opt_is` の下限を 3 文字にすること、`pull` を入れた。6b1f7acc はコメントだけ。root と template の guard は `diff` で同一(`run-verify.sh` の `check-sync.sh` も通る)
- Evidence: `docs/evidence/test-2026-10-07-guard-deny-only.log` の末尾(「cycle 4 (cap raised to 4)」の見出し、46〜59 節)。gitignore の対象なので commit しない
- Guard・lib_json は変えていない。足したのは `tests/test-pre-bash-guard.sh` の 7 行だけ(389436de)

### Test execution(cycle 4)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 1 回目(テストを足す前、6b1f7acc) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 847 s(probe と並走) |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1868 / 126 | 1868 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 1 回目(mode all、scope full、テストを足す前) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 490 s |
| ubuntu:24.04、mawk 1.3.4、dash、GNU sed 4.9・grep 3.11、jq なし: guard / lib_json(テストを足したあと) | 1875 / 126 | 940 / 87 | 0 / 0 | 935 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard / lib_json | 1886 / 126 | 1886 / 126 | 0 / 0 | 0 / 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1886 / 126 | 1886 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-test.sh` 2 回目(テストを足したあと) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 635 s |
| その中の `tests/test-pre-bash-guard.sh` / `tests/test-lib-json.sh` | 1886 / 126 | 1886 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh` 2 回目(テストを足したあと) | 静的な検査 + shell 40 本 + Go | すべて | 0 | 0 | 421 s |

- `run-test.sh` は 2 回とも `lib_json.sh` を分類できず full にフォールバックした。Go は 1 回目に `internal/cli`(117.6 s)と `internal/org`(16.5 s)、2 回目に `internal/org`(12.0 s)が走り、ほかはキャッシュから。shell は 40 本すべて緑で、assertion の `FAIL` は 0、0 でない `FAIL:` の要約も 0
- `run-verify.sh` は 2 回とも `All verifiers passed.` で終わった。shellcheck(足した行を含む `tests/test-*.sh`)、golangci-lint(0 issues)、`check-sync.sh`、branch の secret scan(`51855166..6b1f7acc`、clean)が中で通った
- `tests/test-secret-scan.sh`(固定 `/tmp` パスの既知の flaky)は 4 回とも通った。ほかの実行と重ならないように流した
- 件数の増え方: 1868 から 1886。`guard_deny_only_forms`(B 節、1 行 4 件)に 2 行で +8、`edge_deny`・`edge_none`・`edge_sentinel_deny`(D 節、1 行 2 件)に 1・1・3 行で +10。macOS の BWK awk で AC8(jq なし)の最大は 1.049 秒(1 回目)と 1.033 秒(2 回目)

### 足したテスト(cycle 4、`tests/test-pre-bash-guard.sh`、7 行)

どの行も新版で期待どおり(jq あり・なし)。旧版は、見張りの語を持つ 5 行を deny、git の 2 行を none にする(`--no-verify` の規則が旧版にないこと、`--` だけの merge に見張りの語がないことによる)。各行の形は、probe に使った候補のファイルとバイト単位で一致させた(evidence 53)。

- `guard_deny_only_forms`(B、deny)に 2 行。verify V5-2 に答える
  - `printf $'\x25n' $'arr[\x24(sudo id; echo 1)]'`: 添字の置換も `$'...'` で書いたので、字句解析のあとのどの語の値にも `$` が残らない。書かれたままの語の `$` だけが printf をデータコマンドから外す。zsh 5.9 は、見張りの語を無害な `echo` に替えた同じ形で添字の置換を実行した(evidence 54)。既存の `printf $'\x25n' 'arr[$(…)]'` は、2 つ目の引数の値に `$` があるので、`WV` に戻しても deny のまま
  - `rg $'\x2d-pre' sh 'sudo ls'`: `$'\x2d-pre'` は shell(zsh・bash)には `--pre`、字句解析には `\x2d-pre` なので、`--pre` の検査では外れない。`$'` の検査だけが rg をデータコマンドから外す(evidence 54)
- `edge_deny`(D、deny)に 1 行: `git commit --trailer=x --no-verify -m fix`。`=` で付けた値は 1 語なので、次の語はまたフラグとして読む。git 2.49 で、失敗する pre-commit hook がこの形では走らないことを確かめた(evidence 55)
- `edge_none`(D、none)に 1 行: `git merge --no-ff -- feature`。`--` だけでは `--no-verify` の略記にならない(`opt_is` は `--` の後ろに 1 文字以上を求める)。git 2.49 は `git merge --no-ff -m m -- feature` で merge する(evidence 55)
- `edge_sentinel_deny`(D、deny)に 3 行。どれも 1 つの条件だけを固定する
  - `rg $"sudo ls" .`: rg の `$"` の検査
  - ``printf `echo x` 'sudo ls'``: printf のバッククォートの検査
  - `printf -v c 'sudo ls'`: printf の `-v` の検査。`guard_deny_only_forms` の 7 節の `-v` の 2 行は後ろで `$c` を実行するので、`-v` の検査がなくても許可リストが deny にする

### Mutation の結果(cycle 4)

b3c3fdaa と 12e9a9ad が変えた判定に、依頼の 9 個と、その副条件の 11 個を当てた(printf 6、rg 4、DATACMD 3、`no_verify_rules` 1、commit 3、`opt_is` 2、`pull` 1)。手順は cycle 3 と同じで、使い捨ての写しを作り、system awk で構文を確かめ(evidence 49、構文エラーなし)、A〜D 節の行に渡して期待と違う行を数えた(evidence 50〜52)。

| 変異 | 壊したもの | 足す前(524 行) | 足したあと(531 行、jq / jq なし) | 判定 |
| --- | --- | --- | --- | --- |
| P01 | printf の `%`・`$`・バッククォートを `WV` で見る(依頼) | 1 赤 | 2 / 2 赤 | 殺(`$'\x2dv'` の行と、足した `$'\x25n'` の行) |
| P02 | printf の `%` を落とす(依頼) | 1 赤 | 1 / 1 赤 | 殺(既存 `printf '%s\n' 'sudo ls'`) |
| P03 | printf の `$` を落とす(依頼) | 3 赤 | 4 / 4 赤 | 殺 |
| P04 | printf のバッククォートを落とす | **0** | 1 / 1 赤 | 殺(足した ``printf `echo x` …``) |
| P05 | printf の書式の語を見ない(`j = i + 2`) | 2 赤 | 4 / 4 赤 | 殺 |
| P06 | printf の `-v` を落とす | **0** | 1 / 1 赤 | 殺(足した `printf -v c 'sudo ls'`) |
| R01 | rg の `$'`・`$"` を落とす(依頼) | **0** | 2 / 2 赤 | 殺(足した rg の 2 行) |
| R02 | rg の `$'` だけ落とす | **0** | 1 / 1 赤 | 殺(足した `rg $'\x2d-pre' …`) |
| R03 | rg の `$"` だけ落とす | **0** | 1 / 1 赤 | 殺(足した `rg $"sudo ls" .`) |
| R04 | rg の `$'`・`$"` を `WV` で見る | **0** | 2 / 2 赤 | 殺(足した rg の 2 行) |
| D01 | DATACMD に `test`・`[` を戻す(依頼) | 4 赤 | 4 / 4 赤 | 殺 |
| D02 | `test` だけ戻す | 2 赤 | 2 / 2 赤 | 殺 |
| D03 | `[` だけ戻す | 2 赤 | 2 / 2 赤 | 殺 |
| N01 | `no_verify_rules` を `--` で止める(依頼) | 2 赤 | 2 / 2 赤 | 殺(既存 `git merge -m -- --no-verify feature` など) |
| CM01 | commit の値オプションを完全一致で比べる(依頼) | 1 赤 | 1 / 1 赤 | 殺(既存 `git commit --trail -- --no-verify -m fix`) |
| CM02 | `=` つきでも 2 語進む | **0** | 1 / 1 赤 | 殺(足した `--trailer=x --no-verify`) |
| CM03 | 常に 1 語だけ進む | 2 赤 | 2 / 2 赤 | 殺 |
| O01 | `opt_is` の下限を `k >= 4` に戻す(依頼) | 2 赤 | 2 / 2 赤 | 殺(既存 `git reset --h`、`git commit --m -- --no-verify`) |
| O02 | `opt_is` の下限を `k >= 2` にする | **0** | 1 / 1 赤 | 殺(足した `git merge --no-ff -- feature`) |
| PL01 | `pull` を落とす(依頼) | 1 赤 | 1 / 1 赤 | 殺(既存 `git pull --no-verify origin main`) |

- 20 個すべてが殺。等価な変異はこの周にはない。足す前は 8 個(P04・P06・R01〜R04・CM02・O02)が緑のままで、足した 7 行で閉じた。HEAD は足したあとの 531 行で、jq あり・なしとも期待と違う行が 0
- 依頼の 9 個のうち、足す前に緑だったのは R01 だけ(V5-2 のとおり、rg と `$'...'` の語を組んだ行がなかった)
- P06 は前の周からの穴で、この周の変更とは関係しない。`-v` の 2 行は後ろの `$c` が 1 語目なので、a9ef82b1 の許可リストで deny になる。cycle 3 からは、`-v` の検査を外しても判定が変わらなかった(cycle 3 では `-v` の変異を当てていない)。cycle 2 の Lesson と同じで、広い規則が入ると、前からある規則を固定していた行が別の理由で deny になる
- R03 について: 字句解析は `$"` の `$` を落としてダブルクォートの文字列として読む(`lex_dollar`)。このため `$"--pre"` は、`$"` の検査より前に `--pre` の検査で外れる。`$"` の検査が shell の上で効くのは、bash が翻訳カタログで文字列を置き換えるときだけと思われる(未確認)。guard の判定は `rg $"sudo ls" .` で区別できるので、行を足して固定した
- 全体を流しての確認(evidence 56): R02 と P04 の写しで `tests/test-pre-bash-guard.sh` 全体を流した。足す前のテストファイルでは 2 つとも 1868/0 で緑、足したあとのファイルでは R02 が 6 件(B の 4 件と、G の AC7 の比較の 2 件)、P04 が 2 件の FAIL。行の probe の予測と一致した

### Failure analysis(cycle 4)

テストの失敗はない。self-review C4-1〜C4-4 の形(`$'...'` で書いた printf の書式、`git reset --h`、`git commit --m -- --no-verify`、`git pull --no-verify`)は、12e9a9ad の行と足した行で新版が deny にする。

### Regression checks(cycle 4)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1・AC2・AC3・AC4・AC9 | 通る | `run-test.sh` 2 回。1886 件で ask はなく、jq あり・なしが一致。`tests/test-lib-json.sh` 126/0 |
| AC6: 静的な検査 | 通る | `run-verify.sh` 2 回とも `All verifiers passed.`。root と template の guard は同一 |
| AC7: 旧版の deny は新版でも deny | 通る | 足した 7 行のうち見張りの語を持つ 5 行は新旧とも deny。git の 2 行は旧版が none。スイートの G 節は `run-test.sh` で緑 |
| AC8: 200 KB のコマンドが 5 秒以内 | 通る | macOS BWK awk で最大 1.049 秒、ubuntu の mawk で 0.071〜0.388 秒 |
| cross-review cycle 3 と self-review cycle 4 の修正(b3c3fdaa、12e9a9ad) | 固定した | 上の mutation の表。足す前は 8 個の変異が緑だった |
| awk の方言 | 通る | ubuntu の mawk と gawk で guard 1886/0、lib_json 126/0 |

### Test gaps(cycle 4)

- 変異は手で選んだ 20 個で、計測したカバレッジはない。この周は、変異の対象の条件を固定する行だけを足した
- rg は `$'`・`$"` の語で外れるが、`$x` のような変数の語では外れない。self-review C4-1 は「`WR` に `$` があれば外す」を勧め、verify の cycle 4 の節(rg の検査を述べた 3 項)はこの差を plan の Non-goals の範囲の設計の選び方とした。この /test は今の条件を固定しただけで、その判断は変えていない
- self-review・verify の持ち越し(C3-3〜C3-5、V4-1、V5-1 のコメントのずれ、V5-3 の記録)は、guard を変えると上限 4 を超えるので直さない
- このセッションで効いている Bash の guard は main のチェックアウトの旧版なので、新版を Claude Code の実際の呼び出しで確かめるのは merge 後になる
- `tests/test-secret-scan.sh` の固定パスの問題は、この PR の外の既存の問題として残る(この周では起きなかった)

## Verdict(cycle 4)

- Pass: yes。`./scripts/run-test.sh` 2 回(rc 0、`tests/test-pre-bash-guard.sh` 1868/0 と 1886/0、`tests/test-lib-json.sh` 126/0、Go 8/8、assertion の FAIL 0)、`./scripts/run-verify.sh` 2 回(rc 0、`All verifiers passed.`)、ubuntu:24.04 の mawk(jq あり・なし)と gawk で guard 1886/0(jq なしは 940/0、jq の経路の 935 件は skip)、lib_json 126/0。20 個の mutation はすべて赤。足す前に緑だった 8 個(printf のバッククォートと `-v`、rg の `$'`・`$"`、commit の `=` つきの値、`opt_is` の `--`)を、足した 7 行で閉じた。V5-2 の 2 点(rg の `$'` の行、`WR` だけに頼る printf の行)にも答えた。guard と `lib_json.sh` は変えていない
- Fail: なし
- Blocked: なし
- Known gaps: rg の変数の語(verify の判断のまま)。guard の持ち越し(self-review C3-3〜C3-5、verify V4-1・V5-1・V5-3)は次に guard を変える PR で
