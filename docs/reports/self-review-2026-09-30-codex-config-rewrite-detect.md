# Self-review report: codex-config-rewrite-detect

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-30-codex-config-rewrite-detect.md
- Branch: fix/codex-config-rewrite-detect(HEAD 92d1e42、コードのコミットは cd1aadb)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、堅牢さ、コメントと挙動の一致、保守性)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いたもの

## Evidence reviewed

- `git diff main...HEAD --stat`: 7 ファイル、+840 / -4(plan を除くと 6 ファイル)
- `cmp scripts/ralph-worktree.sh templates/base/scripts/ralph-worktree.sh` と `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md`: どちらも一致
- `scripts/ralph-worktree.sh` の shebang は `#!/usr/bin/env bash` と `set -euo pipefail`(1〜2 行)。`local` は既存の 10 関数以上で使われており、新しい関数の `local` は既存の書き方と揃っている
- 追跡している `.codex/config.toml` と template に `"""` / `'''` は 0 件、値の行の後ろのインラインコメントも 0 件。複数行文字列のガードが、実リポジトリで検知を止めてしまうことはない
- 呼び出し元の文言依存: `has uncommitted changes` を grep しているスキルやスクリプトはない(該当はこの diff の中と `internal/cli/migrate.go` の別の文言だけ)。専用の文言にもこの部分文字列は残っている
- fixture の repo を使った probe(scratchpad、macOS の `/bin/sh` とシステムの git)。結果は下の表のとおり。temp ファイル `ralph-worktree-codex-config.*` の残りは 0 件
- `~/.claude/plugins/cache/everything-claude-code/.../scripts/sync-ecc-to-codex.sh` と `scripts/codex/merge-codex-config.js` の書き込み先を grep で確認(recipe が否定済みとする候補の確認のため。`~/.codex` は読んでいない)
- `docs/tech-debt/README.md`: #185 や `validate_clean_base` に関わる行はない。この diff が閉じる行も、無効にする行もない

### 判定の probe

HEAD の fixture はコメント 2 行を含む短い config。作業ツリーの版だけを変えて `validate-clean-base main` を実行した。

| probe | 作業ツリーの変更 | 結果 |
| --- | --- | --- |
| A | コメントの文言を書き換えた(削除ではない) | 専用の文言 |
| B | コメントを 1 行足した | 専用の文言 |
| C | 値の行のインデントと行末の空白だけ変えた | 専用の文言 |
| D | 改行を CRLF にしただけ | 専用の文言 |
| E | `[shell_environment_policy]` の後に `[features2] # c` と `foo = true` を足した | 専用の文言 |
| E2 | E から見出しの `# c` を外した(対照) | 一般の文言 |
| F | HEAD の版が 0 バイトで、作業ツリーに `[shell_environment_policy]` の table だけを書いた | 一般の文言 |
| F2 | F の HEAD をコメント 1 行にした(対照) | 専用の文言 |
| H1 | `[[shell_environment_policy]]` を足した | 一般の文言 |
| H2 | `[shell_environment_policy.set.nested]` を足した | 一般の文言 |
| H3 | `[ shell_environment_policy . set ]` を足した | 専用の文言 |
| H4 | 最後の table に key を 1 つ足し、その後に table を足した | 一般の文言 |
| J | repo のサブディレクトリから実行した | 専用の文言 |
| K | 値の行にインラインコメントを足した | 一般の文言 |
| 空白 | root のパスに空白を含む repo で、印字された checkout の行を貼り付けて実行した | `fatal: cannot change to '.../with'`、rc 128 |

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| 1 | MEDIUM | コメントと挙動の一致 | 検知は、差分がコメント行・空行・行頭と行末の空白・改行コードの中だけに収まる変更をすべて「既知の外部書き換え」と判定する。コメントの書き換え、コメントの追加、インデントだけの変更、CRLF 化だけの変更も専用の文言になり、`git checkout --` の案内が出る(probe A〜D)。一方で関数のコメント、エラー文言、recipe はどれも「コメントの削除」と説明している。手で書き換えたコメントが、書き換えと呼ばれて checkout の対象として案内される。「Review the diff」と「If that is the only change」があるので、消える前に気づける可能性は高い。plan の Risk 欄は判定をコメント・空行の削除に絞ることを緩和策にしているので、説明と実装が食い違っている | `scripts/ralph-worktree.sh:90-92`(comment and blank lines removed)、`:113-123` と `:130-139`(両方の版から skippable な行を捨て、空白と `\r` を削る)、`:172`(comments removed)、`docs/recipes/codex-setup.md:131`(lost all of its comments) | 判定を狭めるなら、HEAD の行を順に読み、作業ツリーの版が HEAD の skippable な行を落とすことだけを許す(作業ツリーの側に HEAD にない行を出さない)部分列の照合にする。広いままにするなら、3 か所の説明を「コメント・空行・空白・改行コードだけの差分」に揃える |
| 2 | LOW | 堅牢さ | 追加された範囲では、行末にコメントが付いた見出し `[features2] # c` が見出しとして扱われない。直前が `shell_environment_policy` の table だと key 行として吸収され、専用の文言になる(probe E、対照の E2 は一般の文言)。`is_any_header` が `\]$` を要求しているため。codex の再シリアライズがこの形を出すとは考えにくく、手で作った入力に限られる | `scripts/ralph-worktree.sh:127-128`、`:143-150` | 追加された範囲では `t ~ /^\[/` で見出しかどうかを判定する。区切りの見出しでなければ NOMATCH にする。複数行配列の `[1, 2]` 行も NOMATCH 側に倒れ、安全側になる |
| 3 | LOW | 堅牢さ | `FNR == NR` の書き方は、1 つ目のファイル(HEAD の版)が 0 バイトのとき作業ツリーの行も `old[]` に入れてしまう。そのため、空の config に table を足しただけの差分が一般の文言になる(probe F、対照の F2 は専用の文言)。見逃しの方向なので安全側で、追跡している空の config は現実には少ない | `scripts/ralph-worktree.sh:130-134` | `FILENAME == ARGV[1]` で分岐するか、HEAD の版を 1 つ目の入力に固定しない形(下の 5 の stdin 案)にする |
| 4 | LOW | 読みやすさ・操作性 | 案内のコマンドは `${root}` を引用符なしで埋め込んでいる。root のパスに空白があると、貼り付けたコマンドは `git -C` で失敗する(rc 128、何も変更しない)。実害はコマンドが通らないことだけ | `scripts/ralph-worktree.sh:174`、`:176`、`:178`。空白パスの probe | スクリプトは bash なので `printf -v qroot '%q' "$root"` で引用した値を埋め込む。テストの `_cx_assert_specific` の期待値も同じ形に揃える |
| 5 | LOW | 保守性(任意) | temp ファイルの後始末は 3 つの return の経路すべてで `rm -f` しており、probe でも残りは 0 件だった。ただし trap はないので、`mktemp` から `rm` の間にシグナルを受けると HEAD の版のコピーが `$TMPDIR` に残る(中身は追跡ファイルで、秘密ではない)。このスクリプトで `mktemp` を使うのはここだけ | `scripts/ralph-worktree.sh:103-107`、`:108-111`、`:156` | `git -C "$root" show HEAD:.codex/config.toml \| awk '...' - "$work_file"` のように HEAD の版を stdin で渡し、`"""` / `'''` の判定も awk の `index()` に移せば、temp ファイルと 3 か所の後始末が不要になる |
| 6 | LOW | 読みやすさ | `_cx_new_repo` の説明コメントと関数定義の間に、`_cx_fixture` の代入と 16 行の heredoc が挟まっている。コメントが別のもの(fixture の定義)の説明に見える | `tests/test-ralph-worktree.sh:183-186`(コメント)、`:187-203`(fixture)、`:205`(関数) | fixture の定義をコメントの上に移し、コメントを関数の直前に置く |
| 7 | LOW | 保守性 | テストの fixture の組み立てが重複している。書き換えの形の heredoc がケース 4 / 5 / 8 / 9 / 11 にほぼ同じ内容で 5 回ある。ケース 10 と 12 は `_cx_new_repo` の repo 初期化 8 行を複製している。fixture を 1 か所変えるとき、写しの直し漏れが起きやすい | `tests/test-ralph-worktree.sh:318`、`:337`、`:384`、`:404`、`:458`(書き換えの形の heredoc)、`:426-445`、`:477-487`(repo 初期化の複製) | 書き換えの形を `$_tmp` の fixture ファイル 1 つにし、`_cx_new_repo` が HEAD の fixture のパスを引数で受け取れるようにする |
| 8 | LOW | 記述の確度 | 調査記録は、`codex exec` を 15 回以上実行して再現しなかったことから「`codex exec` の起動・終了そのものは原因ではない」と結論している。示せたのは毎回起きるわけではないことまでで、関与を否定する根拠にはならない。同じ記録と recipe は、1 つのサブコマンド(`codex features enable`)の probe を「codex の通常の設定変更(サブコマンド群)」に広げている。probe の手順には、偽の project を trust 済みにしたかどうかの記載もない。codex が未 trust の project の config レイヤを読まないなら、この probe は project レイヤを通っていない(未確認) | `docs/evidence/codex-config-rewrite-2026-09-30.md:57-58`、`:60-73`、`:75-79`、`docs/recipes/codex-setup.md:156-158` | 「15 回以上の実行では再現しなかった」「`codex features enable` では再現しなかった」と、観測した範囲の言い方にする。probe の project が trust 済みだったかを手順に書く |
| 9 | LOW | 記述の正確さ | recipe は 2 つのプラグインについて「どちらも `.codex/config.toml` に書かない」と書く。実際には everything-claude-code の `sync-ecc-to-codex.sh` が `merge-codex-config.js` 経由で `$CODEX_HOME/config.toml`(既定は `~/.codex/config.toml`)に書き込む。追記だけなので、コメントは剥がさない。project 側のファイルに書かないという意味なら正しいが、読み手はユーザーレベルのファイルも含むと読みうる | `docs/recipes/codex-setup.md:153-155`。ECC 1.9.0 の `sync-ecc-to-codex.sh:24-26`、`merge-codex-config.js:313` | 「the project's `.codex/config.toml`」と書き、ECC はユーザーレベルの config に追記するだけだと添える |
| 10 | LOW | 手順の再現性 | 次に起きたときの手順にある `stat -f '%Sm %N'` は BSD と macOS 専用で、時刻をタイムゾーンのオフセットなしのローカル時刻で出す。次の手順では codex のログの時刻と照らし合わせるが、ログ側のタイムゾーンは書かれていない。9 時間ずれて照合する余地がある | `docs/evidence/codex-config-rewrite-2026-09-30.md:100-101` | `stat -f '%Sm %N' -t '%Y-%m-%dT%H:%M:%S%z'` のようにオフセット付きで出し、macOS 用のコマンドだと注記する |

CRITICAL と HIGH はない。

## Positive notes

- porcelain の完全一致(` M .codex/config.toml`)は、porcelain v1 のパスが repo root 基準なので、サブディレクトリから実行しても成り立つ(probe J)。ステージ済み、部分ステージ、削除、未追跡、他のファイルとの同時変更は、関数に入る前に一般の文言へ落ちる
- 想定外の形はどれも一般の文言に倒れる(H1、H2、H4、K、`git show` の失敗、awk の失敗)。関数は `&&` の条件の中で呼ばれるので、中で失敗しても `set -e` で落ちず、戻り値 1 で一般の文言になる
- 止まる / 通る の判定と exit code は変わらず、変わるのは文言だけ。専用の文言にも `has uncommitted changes` が残っている
- `die` は複数行の文言をそのまま出す。`ralph-worktree:` の接頭辞は 1 行目にだけ付き、案内のコマンドの行は貼り付けやすい
- 案内は diff を見る手順が checkout より先にあり、自動では戻さない
- テストの fixture repo はすべて `$_tmp` の下に作られ、既存の `trap cleanup EXIT HUP INT TERM` で消える。`assert_contains` は `grep -qF --` を使うので、`-` で始まる文字列でも安全
- script と recipe は template と byte 一致

## Coverage gaps

- ケース 12 のコメントは「HEAD に `.codex/config.toml` がない」と書くが、porcelain が `?? .codex/` なので porcelain の照合で止まる。関数の `git show` 失敗の分岐(`scripts/ralph-worktree.sh:104-107`)は通らない。` M` は HEAD にそのパスがあることを含むので、呼び出しの約束の下ではこの分岐に到達しない。防御として残すのは妥当で、テストのコメントを「porcelain の照合で止まる」に直せば誤読がなくなる
- codex が実際に出す書き換えの形は、この diff の中では再現していない。判定が実物に一致するかは、記録された 2 回の観測の記述に依存している
- テストの一式は実行していない(このフェーズの範囲外)。上の probe は macOS の `/bin/sh`、bash、システムの git で行った

## Recommendation

- Merge: 可(CRITICAL / HIGH なし)。ただし 1 の MEDIUM はこの cycle での修正を勧める。狭めるか説明を揃えるかは実装側の判断
- Follow-ups: 2〜4 はスクリプトの小さな修正で、1 と同じコミットで直せる。8〜10 は調査記録と recipe の言い方の修正。5〜7 は任意

## Cycle 2

- Date: 2026-09-30
- Reviewed HEAD: a777e82(コードのコミットは Slice B 0bb6aae、Slice C d84cdab、Slice D ee397e9)
- Scope: `git diff b164556...HEAD -- scripts/ tests/ docs/recipes/ docs/evidence/ templates/ docs/tech-debt/`。diff の品質だけで、範囲は cycle 1 と同じ
- Cycle: 2/2(上限の cycle)

### Evidence reviewed

- `cmp` で script と recipe が template と一致することを確認
- 新しい判定(HEAD の生の行を順に照合する)を fixture の repo で probe した。macOS の `/bin/sh`、`/bin/bash` 3.2.57、awk 20200816 を使った。結果は下の表のとおり
- `FILENAME == "-"` の移植性: macOS の awk、ubuntu:24.04 の mawk(docker、`--network none`)、BusyBox の awk で、どれも `-` が返った。gawk は手元になく未確認
- `%q` の出力: bash 3.2 は UTF-8 のロケール(このマシンは `en_US.UTF-8`)で、日本語の先頭バイトを生のまま、続くバイトの一部を 8 進のエスケープで出す。端末には `$'/tmp/�\227��\234��\236/repo'` と表示された。C ロケールでは全バイトがエスケープされる
- 空白を含む TMPDIR で `sh tests/test-ralph-worktree.sh` を実行し、120 / 0 だった。ただし macOS の `mktemp -d` はこの TMPDIR を使わず、`/var/folders/...` に作っていた(probe で確認)
- ECC 1.9.0 の `scripts/codex/merge-codex-config.js` で、merge する table の一覧(27〜34 行の `TABLE_PATHS`)と書き込み(295、300、313 行)を確認した
- `docs/insights/events/2026-09-30-codex-config-rewrite-detect.jsonl`: cycle 1 の self_review / verify / test / sync_docs / cross_review がそろっている
- tech-debt に追加された行: セルの区切りは 6 本で、見出しの行と同じ

| probe | 作業ツリーの変更 | 結果 |
| --- | --- | --- |
| M1 | 内容は同じで、モードだけ変えた(`chmod +x`) | 専用の文言 |
| M2 | 末尾の改行を外しただけ | 専用の文言 |
| M3 | 末尾の改行を足しただけ(HEAD にはなかった) | 専用の文言 |
| M4 | 末尾に空行を足しただけ | 専用の文言 |
| M5 | 末尾に空白だけの行を足しただけ | 専用の文言 |
| G1 | コメント行と空行を消し、policy の table を足した(対照) | 専用の文言 |
| G2 | policy の table の後に、インデントした `[[features]]` を置いた | 一般の文言 |
| G3 | policy の table に `exclude = [1, 2]` と inline table の key を置いた | 専用の文言(key 行として扱う) |
| G4 | policy の table に、入れ子の配列の続きの行 `  [1, 2],` を置いた | 一般の文言(安全側) |
| G5 | policy の table に dotted key の `features.hooks = false` を置いた | 専用の文言(policy の名前空間の中) |
| G6 | 末尾に CRLF の policy の table を足した | 一般の文言 |
| G7 | HEAD の見出しの行 `[features]` を消した | 一般の文言 |
| G8 | HEAD の末尾がすでに policy の table で、そこに key を足した | 一般の文言(安全側の見逃し) |

照合の順序について。貪欲な照合で位置がずれる心配はない。前から照合して一致した作業ツリー側の行は、HEAD の行を順に拾ったものになる。拾われなかった HEAD の行はすべて空行かコメント行になる。つまり一致の結論は「HEAD から空行とコメント行だけを消したもの」を意味し、どの順で照合しても変わらない。HEAD のコメント行が後ろの作業ツリーの行と同じ文字列でも、照合できるのは同じ位置の行とだけなので、途中への挿入や移動は NOMATCH になる。HEAD の空行と追加部分の空行が揃う場合も、結果は変わらない。追加部分では空行が許されているからである。追加部分で policy 以外の table が始まるのは `[` で始まる行だけで、それは G2 と Slice D で塞がっている。`key = [1, 2]` のように `=` の後で `[` が来る行は、key 行のまま扱われる(G3)。

### cycle 1 の所見の確認

| # | cycle 1 の所見 | HEAD での状態 |
| --- | --- | --- |
| 1 | MEDIUM: 判定が広すぎる | 直っている。照合は生の行で行い、コメントの書き換え、コメントの追加、インデント、CRLF は一般の文言になる(ケース 14、15、16、11)。文言も「comment or blank lines deleted」に変わった。ただし、何も変わっていない形が残っている(下の C2-1) |
| 2 | LOW: 行末コメント付きの見出し | 直っている。追加部分は `/^[ \t]*\[/` で見出しを判定し、受け入れるのは policy の見出しの正確な形だけ(ケース 17、22〜24) |
| 3 | LOW: `FNR == NR` と空の HEAD | 直っている。`FILENAME == "-"` で区別する(ケース 18)。移植性は上のとおり |
| 4 | LOW: root が引用されていない | 空白は直っている(ケース 19)。ただし `%q` が非 ASCII の root で新しい問題を作った(下の C2-2) |
| 5 | LOW: temp ファイル | 直っている。HEAD の版を stdin で渡すので、temp ファイルはない |
| 6 | LOW: 説明コメントの位置 | 直っている。`tests/test-ralph-worktree.sh:201-209` のコメントが `:210` の関数の直前にある |
| 7 | LOW: fixture の重複 | 直っている。`_cx_write_rewrite_shape` と `_cx_new_repo` の引数にまとめた |
| 8 | LOW: 調査記録の結論の強さ | 直っている。`codex exec` の結論を弱め、`codex features enable` の範囲を限定し、trust 済みでの再 probe を足した |
| 9 | LOW: plugin の記述の範囲 | 「project's」の限定は入った。ただし ECC が書く中身の説明が recipe と調査記録で食い違い、どちらも実物と合わない(下の C2-4) |
| 10 | LOW: `stat` の書式 | 直っている。`-t '%Y-%m-%dT%H:%M:%S%z'` に変わり、GNU 版の書き方も添えてある |

### Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | コメントと挙動の一致 | 判定には「既知の変更が 1 つでもあった」という条件がない。そのため、awk の行の比較では違いが見えない変更が、どれも専用の文言になる。該当するのは、モードだけの変更、末尾の改行の有無、末尾に空行や空白だけの行を足す変更(probe M1〜M5)。文言は「comment or blank lines deleted and/or a [shell_environment_policy] table appended」だが、実際にはどちらも起きていない。関数のコメントは、行末の変更は一般の文言に落ちると書き(94〜96 行)、` M` なので差分は必ずあり、差分なしの場合を特別扱いしないとも書いている(104〜106 行)。後者の理由付けが、そのままこの漏れを作っている。recipe と調査記録にも同じ言い方がある。checkout で消えるのはモードや末尾の改行だけなので、実害は文言が不正確になることにとどまる | `scripts/ralph-worktree.sh:94-96`、`:104-106`、`:139-162`(照合と追加部分のループ)、`docs/recipes/codex-setup.md:136`、`docs/evidence/codex-config-rewrite-2026-09-30.md:151` | awk で「HEAD の行を 1 つ以上落とした」か「policy の見出しが 1 つ以上ある」ことを記録し、どちらもなければ NOMATCH にする。モードだけの変更と末尾の改行だけの変更のテストを足す |
| C2-2 | LOW | 操作性(修正による退行) | cycle 1 の 4 に対する修正の `printf -v qroot '%q'` は、`#!/usr/bin/env bash` が選ぶ macOS の bash 3.2 と UTF-8 のロケールで、非 ASCII の root を生のバイトとエスケープの混ざった形で出す。端末には `$'/tmp/�\227��\234��\236/repo'` と表示され、端末からコピーしたパスは壊れる。修正前は、空白のない非 ASCII のパスならそのまま貼り付けられた。生のバイト列を eval すれば元に戻る(zsh でも bash でも rc 0)ので、ケース 19 の `bash -c` による往復ではこの問題を捕まえられない | `scripts/ralph-worktree.sh:182`。bash 3.2.57 で C、`en_US.UTF-8`、`ja_JP.UTF-8` の各ロケールで probe | 単一引用符の形にする。`sq="'\\''"; qroot="'${root//\'/$sq}'"` なら、bash 3.2 で日本語、空白、`'`、`$` とバッククォートの 4 種のパスが zsh で元に戻ることを確認した(置き換える文字列を直接書く `${root//\'/\'\\\'\'}` は、bash 3.2 では `'` を含むパスで壊れる)。非 ASCII の root のテストを 1 つ足す |
| C2-3 | LOW | テストの堅牢さ | `_cx_assert_specific` は、引用しない root で `git -C $2 diff ...` を期待している。文言が出すのは `%q` の形なので、このテストが通るのは fixture の root に引用が要らないからにすぎない。macOS の `mktemp -d` は TMPDIR を使わないので問題は出ない。Linux の `mktemp -d` は TMPDIR に従うので、空白か非 ASCII を含む TMPDIR では、ケース 1、2、3、18 の文言の assertion が落ちるはずである(コードからの推論で、Linux では実行していない)。また、ケース 19 の `bash -c "$_cx_checkout_line"` は `set -e` の下で守られていない。印字された行が壊れると、FAIL の行も集計も出さずにテスト全体が止まる(plan の Slice B の記録に「%q を外すとテストが git のエラーで中断する」とある) | `tests/test-ralph-worktree.sh:286-294`(`:289` ほか)、`:568` | 期待値を、スクリプトと同じ引用の形で helper の中で組み立てる(C2-2 の単一引用符の形にするなら、sh の中で同じ置き換えを書ける)。ケース 19 の実行を `set +e` で囲み、終了コードを `assert_eq` で確かめる |
| C2-4 | LOW | 記述の正確さ | ECC がユーザーレベルの config に書く中身の説明が、recipe と調査記録で食い違い、どちらも実物と合わない。recipe は「`[features]` の項目」、調査記録は「ECC 自身のフック定義」と書いている。`merge-codex-config.js` が足すのは MCP 以外の基本設定で、中身は `features`、`profiles.strict`、`profiles.yolo`、`agents.*` の table と、ルートの key である。調査記録の「追記だけ」も正確ではない。ルートの key は最初の table の前に差し込み、既存の table には key を足す。ただし全体を再シリアライズしない文字列の編集なので、コメントを剥がさないという結論は変わらない。「フック定義」の記述は cycle 1 からあり、私も見落としていた | `docs/recipes/codex-setup.md:160-162`、`docs/evidence/codex-config-rewrite-2026-09-30.md:38`、`:41`。ECC 1.9.0 の `merge-codex-config.js:27-34`、`:295`、`:300`、`:313` | 両方を「MCP 以外の基本設定(`[features]`、`[profiles.*]`、`[agents.*]`、ルートの key)を、既存の文字列に差し込む形で足す」に揃える |
| C2-5 | LOW | 読みやすさ | 関数のコメントに、書き直しの跡が 2 つ残っている。103 行は `is` で途切れる短い行で、104 行に続く(折り返しの直し残し)。107 行は、Slice B で正規化をやめて生の行の比較にした後も「line-based normalization」と書いている | `scripts/ralph-worktree.sh:102-104`、`:107` | 102〜104 行を折り返し直し、107 行を「line-based comparison」にする |

CRITICAL、HIGH、MEDIUM はない。

### Positive notes

- 照合の順序で結論が変わらないこと、追加部分で policy 以外の table が始まる道が `[` の行しかないことを、上の probe で確かめた(G2〜G8)
- 想定外の形はどれも一般の文言に倒れる。インデントした `[[...]]`、入れ子の配列の続きの行、CRLF、見出しの削除、HEAD の末尾にある既存の policy の table への key の追加(G2、G4、G6、G7、G8)
- HEAD の版は stdin で awk に渡され、temp ファイルと 3 か所の後始末がなくなった。`git cat-file -e` で HEAD にないパスを先に落としている
- `FILENAME == "-"` は BWK awk、mawk、BusyBox の awk で同じ値になる
- Slice D の判定は、インデントした policy の見出しも一般の文言にする。受け入れる形を増やさずに穴を塞いでいる
- 新しいテストのケース 14〜24 は、それぞれ 1 つの条件だけを変えている。ケース 22〜24 は、判定を `/^\[/` に戻すと落ちる
- script と recipe は template と byte 一致。tech-debt の行はセルの数が合っている。insight のイベントもそろっている

### Coverage gaps

- gawk での `FILENAME == "-"` は確かめていない。CI は pull_request のときだけ動くので、このブランチの Linux での結果はまだない
- 空白を含む TMPDIR での実行(テスト報告にある)は、macOS では `mktemp -d` がその TMPDIR を使わないため、引用が必要な root を試せていない。ディレクトリを明示するケース 19 は例外
- C2-3 の Linux での失敗は、コードを読んだうえでの推論で、実行はしていない
- Windows の `core.autocrlf=true` のように checkout で改行を変換する環境では、`git show` の blob と作業ツリーの版がすべての行で食い違い、一般の文言になる。見逃しの方向なので安全側。対象の環境かどうかは判断していない

### Recommendation

- Merge: 可(CRITICAL、HIGH、MEDIUM なし)
- Follow-ups: この cycle は上限の 2/2 で、plan は `/pr` で archive される。直さなかった LOW は、どこにも残らないまま置き去りになる。C2-1〜C2-3 はスクリプトとテストの小さな修正で、1 コミットで直せる。直さないなら、C2-1〜C2-5 をまとめた tech-debt の行を 1 つ作ることを勧める。発火条件は「`codex_config_external_rewrite_only` への次の変更」がよい
