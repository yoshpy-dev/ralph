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
