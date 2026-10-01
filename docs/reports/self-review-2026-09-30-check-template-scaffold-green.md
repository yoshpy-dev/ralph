# Self-review report: check-template-scaffold-green

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-30-check-template-scaffold-green.md(issue #189)
- Branch: fix/check-template-scaffold-green(HEAD d3dca29、コードのコミットは d66be51)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、堅牢さ、コメントと挙動の一致、保守性)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いた 5 ファイル

## Evidence reviewed

- `git diff main...HEAD --stat`(plan を除く): 5 ファイル、+257 / -31
- `cmp scripts/check-template.sh templates/base/scripts/check-template.sh`: 一致
- ブランチのバイナリ(`go build -o <scratchpad>/ralph-189 ./cmd/ralph`)で scratchpad に fresh scaffold を作り、`CI=true sh scripts/check-template.sh` を実行した。exit 0 で FAIL 行はない。scaffold には `find` の探索先 5 つ(`.claude/hooks`、`packs`、`scripts`、`.claude/skills`、`.claude/agents`)がすべてある
- 同じ scaffold を macOS の `/bin/sh`(bash 3.2)、`/bin/dash`、busybox ash、ubuntu:24.04 の dash で走らせた。docker は `--network none` で、scaffold は tar で渡した。ubuntu:24.04 の `/bin/sh` は dash
- `git show main:scripts/check-template.sh` を同じ fixture で走らせ、新旧の出力を比べた
- `ralph init` の副作用: `internal/cli/init.go:235-252` が対象の中で `git init` と git hook の導入をする。`internal/cli/git_hooks.go:190-205` は project の外を指す `core.hooksPath` を拒むので、case G は HOME に書かない
- `go run ./cmd/ralph init --yes <scratchpad>` の所要時間は、ビルドキャッシュがある状態で 0.92 秒(real)
- 外した 3 項目の残り: docs、README、AGENTS.md、rules、skills、template を grep した。必須と読める記述はない。`scripts/check-sync.sh:62-63` はこの 2 つの docs を root 専用として扱っており、今回の判断と合う。`scripts/bootstrap.sh:52-60` は検査を呼んで結果を出すだけで、項目の名前は出さない
- `templates/base` の中で `issue #NNN` を含むのは `scripts/check-template.sh:25` だけ。配られる `scripts/` と `.claude/hooks/` で "meta-repo" を含むのもこのファイルだけ
- `docs/tech-debt/README.md`: 削除した行が挙げていたのは 3 点(引数込みの文字列をパスとして検査する、subshell で fail-open になる、meta-repo 専用の 3 項目)で、この diff はその 3 点をすべて直している。ほかに閉じられる行や無効になる行はない
- `scripts/verify.local.sh:149`: shellcheck の対象に 1 語足しただけ。template に `verify.local.sh` はないので、同期する先もない

### probe の結果

| probe | シェル | 結果 |
| --- | --- | --- |
| fresh scaffold | macOS sh、dash、busybox ash、ubuntu dash | exit 0、`Template structure looks good.` |
| fresh scaffold から `ralph-dispatch.sh` を消し、`run-test.sh` の実行属性を外す | macOS sh、busybox ash、ubuntu dash | exit 1。実行属性の FAIL が 1 行、同じ文面の hook の FAIL が 7 行 |
| 実行後に残った `check-template.*` の一時ディレクトリ | busybox ash、ubuntu dash | 0 件 |
| 引数のない hook コマンド(`./.claude/hooks/ralph-dispatch.sh`) | dash | FAIL なし |
| JSON の `\t` で区切った hook コマンド | dash | FAIL。計画の Non-goals にある形で、閉じる側に倒れる |
| `scripts/locked/`(`chmod 000`)の中に実行属性のないスクリプト | macOS sh | 新: exit 0 で `looks good`。main: stderr に `find: scripts/locked: Permission denied` |
| `.claude/settings.json` を `chmod 000` | macOS sh | exit 0 で `looks good`(main も同じ) |
| `trap cleanup EXIT` だけを持つ同じ形の script に SIGTERM | dash / bash | dash は一時ディレクトリが残る(rc 143)。bash は消す |
| `mktemp -d` の失敗(親ディレクトリがない) | dash、bash | exit 1。mktemp のエラーだけが出て FAIL 行はない。trap を張る前なので消すものもない |
| `set -e` の下で EXIT trap の最後のコマンドが失敗する | dash、bash、sh | exit 0 が 1 に変わる。cleanup は自前の一時ディレクトリへの `rm -rf` だけで、失敗する経路は見当たらない |
| 実行可能な `scripts/with space/exec.sh` を main の版で検査 | macOS sh | exit 1。FAIL は `scripts/with` と `space/exec.sh` の 2 行で、`with space` という文字列は出ない |

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| 1 | MEDIUM | 配布物のコメント | `required_files` の上のコメントが "meta-repo-only" と "(issue #189)" を書いている。このファイルは byte 一致のまま scaffold に配られる。scaffold された project では #189 はその project 自身の issue を指し、"meta-repo" も説明のない言葉になる。purity guard は固定の文字列しか見ないので、これを止めない | `templates/base/scripts/check-template.sh:21-25`(root の同じ行と一致)。`templates/base` で `issue #NNN` を含むファイルはこれ 1 つ | 両方のコピーで通じる言い方にする。例:「`ralph init` が配るファイルだけを並べる。ralph のソースリポジトリにしかない README.md と docs のノートは入れない」。issue 番号は commit と plan に残す |
| 2 | LOW | 堅牢さ | `find` に新しく付いた `2>/dev/null` が、探索先がない場合とサブツリーが読めない場合を区別せずに黙らせる。読めないディレクトリの中のスクリプトは検査されないまま、`Template structure looks good.` で exit 0 になる。main は exit を無視しつつ、find のエラーを stderr に出していた。計画の逸脱メモにある「以前の `$(find ...)` と同じ許容」は exit の扱いには当てはまるが、stderr には当てはまらない。root にも fresh scaffold にも探索先は全部あるので、この抑制が必要になる場面はテストにも出てこない。build_fixture の「探索先がないと警告が出ないように」というコメントも、この抑制で意味を失った | `scripts/check-template.sh:62-63, 71, 79`、`tests/test-check-template.sh:100-101`。probe の表の `scripts/locked/` の行 | `2>/dev/null` を外し、`\|\| true` は残す。探索先がない project では main と同じく stderr に 1 行出る。存在する探索先だけを `find` に渡す形でもよい |
| 3 | LOW | 堅牢さ | hook の一覧を作るパイプラインの終了コードは `tr` のもので、grep の終了コードは見えない。末尾の `\|\| true` は grep の「一致なし」を受けているように読めるが、実際には何も受けていない。grep の `2>/dev/null` と合わせて、読めない settings.json では hook を 1 つも検査せずに exit 0 になる。main から続く挙動だが、この PR は hook の検査を閉じる側に直したので、閉じない経路はここだけが残る | `scripts/check-template.sh:90-91`。probe の表の `chmod 000` の行 | grep の終了コードを別に受け、2 以上なら `fail` する(下の修正案) |
| 4 | LOW | 出力の読みやすさ | 引数を落としたので、7 つの event が同じ dispatcher を参照する設定では、dispatcher がないときに同じ文面の FAIL 行が 7 行出る。以前は event 名で行を見分けられた | `scripts/check-template.sh:92-98`。probe の表の 2 行目 | パスだけにしてから `sort -u` した一覧を読む。AC-2 は引数を出さないことを求めているので、event 名を戻すより重複を消すほうが合う |
| 5 | LOW | コメント | 一時ディレクトリの上のコメントは「find で作る一覧のループはすべて」と書き、理由の 1 つに `while` へのパイプ(subshell)を挙げる。`while` にパイプしていたのは grep で作る hook のループで、find の 3 つのループの問題は `$(find ...)` の単語分割だった。subshell の問題が実際にあったループが、コメントの対象から外れている。テストの見出しの「required_files list and its find-driven checks」と「fail-open-adjacent」も、hook の検査と fresh scaffold を言い表していない | `scripts/check-template.sh:11-14`、`tests/test-check-template.sh:3, 8` | 「下の一覧を読むループ(find の 3 つと settings の hook の 1 つ)」のように対象を書き分ける。テストの見出しには hook の検査と fresh scaffold を足す |
| 6 | LOW | コメント(diff の外) | Go のテストのコメントが、非スクリプト項目の例として README.md と docs を挙げたまま残っている。今の非スクリプト項目は AGENTS.md、CLAUDE.md、`.claude/settings.json` の 3 つ。計画は「Go 側は変わらない」として、このファイルに触れていない | `internal/scaffold/embed_test.go:171` | 例を `AGENTS.md, CLAUDE.md, .claude/settings.json` に直す |
| 7 | LOW | テストの出力 | `go` がないときの case G は `pass` を呼ぶので、`PASS: G. fresh scaffold check skipped` と出て、pass の件数にも入る。ほかの suite は `SKIP:` を出し、件数に数えない。計画も「SKIP と明示」と書いている | `tests/test-check-template.sh:343-344`。既存の書き方は `tests/test-secret-scan-branch.sh:264` と `tests/test-secret-scan.sh:211` | `printf '  SKIP: %s\n' ...` で出し、件数に数えない |
| 8 | LOW | テストの判定 | F4 の `! grep -qF 'with space'` は、F4 が防ぐ回帰(単語分割)の出力と一致しない。main の版の出力は `scripts/with` と `space/exec.sh` の 2 行で、`with space` を含まない。この回帰を捕まえているのは rc の判定だけ | `tests/test-check-template.sh:327`。probe の表の最後の行 | D、E、G と同じ `! grep -q '^FAIL:'` にする |
| 9 | LOW | 堅牢さ | trap が EXIT だけなので、dash では SIGTERM で一時ディレクトリが残る。Ubuntu の `/bin/sh` は dash なので、`#!/usr/bin/env sh` は Linux の CI と開発機で dash を選ぶ。中身はパスの一覧だけで、実行は 1 秒かからない。macOS の `/bin/sh`(bash)は消す | `scripts/check-template.sh:15-19`。probe の表の SIGTERM の行(同じ trap の形の script で確認) | `trap 'cleanup; exit 130' INT` と `trap 'cleanup; exit 143' TERM` を足す。直さない場合は known gap として残す |

### 修正案(3 と 4)

grep の終了コードを別に受け、パスだけにしてから重複を消す案です。`cut` でパスを取り出すので、ループの中の `${hook_cmd%% *}` はなくても動きます。ただし AC-5 の mutation(パス全体を使う)の対象が `cut` に移るので、残すか外すかは実装側で決めてください。

```sh
rc=0
grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json > "$tmpdir/hooks.raw" || rc=$?
if [ "$rc" -gt 1 ]; then
  fail "Cannot read .claude/settings.json (grep exit $rc)"
fi
tr -d '"' < "$tmpdir/hooks.raw" | cut -d ' ' -f 1 | sort -u > "$tmpdir/hooks.list"
```

## Positive notes

- 4 つのシェル(macOS sh、dash、busybox ash、ubuntu dash)で、ループの中の `fail` が exit 1 に伝わった。一時ディレクトリも残らなかった
- case G は `go run` の失敗を FAIL として出し、skip には倒さない(`tests/test-check-template.sh:353-359`)
- `GOLDEN_ENTRIES` の件数のコメント(非スクリプト 3 + スクリプト 22)と、4 箇所を同時に変えるというメンテのコメントは、実際の一覧と合っている
- tech-debt の行の削除に過不足はない

## Coverage gaps

- テストは走らせていない。確かめたのは上の probe だけ
- GitHub の ubuntu runner の image そのものは確かめていない。ubuntu:24.04 の container で `/bin/sh` が dash であることを見ただけ
- SIGINT の probe は background の job に送ったため、SIGINT が無視されて無効だった。SIGTERM の結果だけを根拠にしている
- settings の hook コマンドが `./` で始まらない形(`bash .claude/hooks/x.sh` など)は grep に一致せず、検査されない。main から続く範囲で、ralph が生成する形でもないので、指摘にはしていない

## Recommendation

- Merge: 可(CRITICAL / HIGH なし)。1 の MEDIUM は配布物に出る文言なので、この cycle で直すことを勧める
- Follow-ups: 2〜4 と 9 は `check-template.sh` の小さな修正で、1 と同じコミットで直せる(byte 一致の template も同時に)。5〜8 はコメントとテストの書き方の修正で、任意

## Cycle 2

- Date: 2026-10-01
- Reviewed HEAD: bc8a7e4(コードのコミットは Slice B 8fd5113 と Slice C 39c2629)
- Scope: `git diff 375d6fa...HEAD -- scripts/ templates/ tests/ internal/`。diff の品質だけで、範囲の考え方は cycle 1 と同じ
- Cycle: 2/2(上限の cycle)

### Evidence reviewed

- `git diff 375d6fa...HEAD --stat`(対象の 4 ファイル): +282 / -51
- `cmp scripts/check-template.sh templates/base/scripts/check-template.sh`: 一致。HEAD のバイナリで作った fresh scaffold の `scripts/check-template.sh` も HEAD と一致
- template の版に `issue #` と `meta-repo` は 0 件
- root と template の `.gitignore` で、探索先の下にある ignore 対象は `.claude/hooks/local/` だけ。packs と scripts に同じ扱いは要らない
- `[ -d "$root" ] && set -- ...` のループを、最後の探索先がない状態にして `set -eu` の下で走らせた。macOS sh、dash、bash、busybox ash、ubuntu dash のどれも止まらず、残りの探索先を渡した
- top level の `set --` が上書きする位置引数を読むのは、このブロック(72〜84 行)だけ。`fail` の `$1` は関数の引数
- 下の 12 ケースを、macOS の BSD find、ubuntu:24.04 の GNU find、busybox find で、uid 1000 として走らせた(docker は `--network none`)。3 つの結果は一致した
- 同じ trap の形の script に TERM と HUP を送った。dash、bash、macOS sh、busybox ash のどれも exit 143 / 129 で終わり、一時ディレクトリは消えた
- 読めないファイル(`chmod 000` の `.sh`)は、BSD find でも GNU find でも rc 0 で列挙される
- mutation: `git archive HEAD` を scratchpad に展開し、`check-template.sh` を 1 箇所ずつ変えて `tests/test-check-template.sh` を走らせた。go を PATH から外したので case G は SKIP。対照は 42 / 0 / 1

### probe の結果(HEAD、uid 1000、3 種類の find で同じ)

| ケース | 結果 |
| --- | --- |
| fresh scaffold | exit 0 |
| `.claude/hooks/local/disabled` を 000 | exit 0 |
| `.claude/hooks/local` 自体を 000 | exit 0 |
| `.claude/hooks/local/a.sh` に実行属性がない | exit 0 |
| `scripts/locked` を 000 | exit 1。`could not list scripts ...` と find の `Permission denied` |
| `.claude/agents/sub` を 000 | exit 1。`could not list agent files ...` |
| `.claude/skills/zz` を 000 | exit 1。`Skill missing SKILL.md`(以前からの判定) |
| `.claude/settings.json` を 000 | exit 1。`could not read hook commands ...` と grep の `Permission denied` |
| settings.json が `{}`(hook なし) | exit 0 |
| dispatcher を削除 | exit 1。FAIL は 1 行 |
| `packs/` を削除 | exit 0。何も出ない |
| `.claude/hooks/Stop.d/local/x` を 000(実在の規約ではない比較用) | exit 1。prune は `.claude/hooks/local` だけに効く |

### mutation の結果

| mutation | 落ちたケース |
| --- | --- |
| prune を `-not -path '.claude/hooks/local/*'` に戻す(AR-1 の退行) | F7 |
| settings の grep の rc 2 を無視する | E(読めない settings.json) |
| scripts の find の rc による `fail` を外す | F5 |
| `-print` を外す | なし |
| 探索先を `[ -d ]` で絞らずに全部渡す | なし |
| `.claude/skills` と `.claude/agents` の `[ -d ]` を外す(2 件) | なし |
| skills と agents の find の rc による `fail` を外す(2 件) | なし |
| INT / TERM / HUP の trap を外す | なし(計画の known gap に記載あり) |

### cycle 1 の指摘の確認

| cycle 1 | 状態 | HEAD での根拠 |
| --- | --- | --- |
| #1 MEDIUM 配布されるコメント | 修正済み | `scripts/check-template.sh:27-29`。issue 番号も "meta-repo" もなく、scaffold の読み手にも通じる |
| #2 find の `2>/dev/null` | 修正済み | 72-84、92-100、108-116 行。探索先を絞り、rc が 0 でなければ FAIL。find の stderr も出る |
| #3 grep の rc | 修正済み | 127-131 行。rc 1 は通し、2 以上は FAIL |
| #4 同じ FAIL が 7 行 | 修正済み | 136 行の `cut` と `sort -u`。probe でも 1 行 |
| #5 コメントとテストの見出し | 修正済み | 11-16 行、`tests/test-check-template.sh:2-12` |
| #6 `embed_test.go` の例 | 修正済み | `internal/scaffold/embed_test.go:171` |
| #7 G の skip | 修正済み | `tests/test-check-template.sh:71-74` の `skip` と 491 行。集計の行にも skipped が出る |
| #8 F4 の判定 | 修正済み | `tests/test-check-template.sh:397` が `^FAIL:` で判定する |
| #9 signal の trap | 修正済み | 22-24 行。probe で exit 143 / 129、一時ディレクトリは残らない |

cross-review の AR-1 も直っている。78 行の `-path '.claude/hooks/local' -prune -o -type f -name '*.sh' -print` は、探索先の文字列 `.claude/hooks` から find が出すパスの形と一致し、`-print` は右側の枝にだけ付いているので、prune したディレクトリは出力されない。

### Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | テストの判定 | Slice B と C で増えた分岐のうち、外すとテストが落ちるのは 3 つだけ(上の mutation の表)。押さえられていない分岐のうち 2 つは、外すと製品の挙動が変わる。探索先の絞り込みを外すと、`packs/` のない project が `could not list scripts` で FAIL になる。`-print` を外すと、実行ビットのない `.claude/hooks/local` が `Script is not executable` で FAIL になる。どちらの mutation でもテストは緑のまま。build_fixture は探索先を必ず作るので、探索先のない project を通るケースが 1 つもない | `scripts/check-template.sh:73-75, 78, 92, 108`、`tests/test-check-template.sh:120-125`。probe: `packs/` を消した scaffold では、絞り込みを外した版が `find: packs: No such file or directory` と FAIL を出し、HEAD は exit 0。`chmod 644 .claude/hooks/local` では、`-print` を外した版だけが FAIL を出す | fixture から `packs/` と `.claude/agents/` を消して exit 0 を確かめるケースと、F8 で `local` を 644 にするケースを足す。読めない `.claude/agents` のサブツリーは F5 の隣に足せる。網羅の判断は /test に渡す |
| C2-2 | LOW | コメント | 配られるコメントに不正確な箇所が 2 つある。69-70 行の「a missing root stays silent, as before」の "as before" が正しいのは、ブランチの途中の Slice A(d66be51)に対してだけ。main は探索先がないと stderr に `find: packs: No such file or directory` を出していたので、merge 後に読むと main の挙動を取り違える。67 行の「an unreadable file or directory」の file は意味がない。find は中の項目を lstat するだけで、読めないファイルで find が失敗することはない。あわせて、新しい 4 つの FAIL の文言は小文字で始まり、既存の FAIL(`Missing required file` など)と揃っていない | `scripts/check-template.sh:65-71`、80、96、112、130 行(template も同じ)。probe: main の版と d66be51 の版を `packs/` のない scaffold で走らせて比べた。読めないファイルの probe は上の Evidence を参照 | 「a missing root is skipped without a message」「an unreadable directory anywhere inside it」のように直す。FAIL の文言の頭を揃えるかは任意 |

### Recommendation

- Merge: 可(CRITICAL / HIGH / MEDIUM なし)。cycle 1 の 9 件と AR-1 は、どれも HEAD で直っている
- Follow-ups: 今回が上限の cycle なので、C2-1 と C2-2 を直さずに /pr へ進むなら、tech-debt に 1 行まとめて残す。C2-2 はコメントと文言だけの修正、C2-1 は fixture を 2〜3 個足す修正

### Known gaps

- SIGINT は確かめていない(background の job では SIGINT が無視される)。TERM と HUP の結果だけを根拠にしている
- F5 や F7 の途中で止めると、000 にしたディレクトリが TMPDIR に残る。EXIT trap の `rm -rf` は、中身のある 000 のディレクトリを消せない(macOS で `Permission denied`、rc 1)。残る可能性があるのは check-template.sh を 1 回走らせる間だけ
- `.claude/skills` の下の読めないディレクトリは `Skill missing SKILL.md` として報告される。以前からの判定で、閉じる側に倒れる
- GitHub の ubuntu runner の image そのものは確かめていない。確認は ubuntu:24.04 の container で行った
- テストは worktree では走らせていない。走らせたのは scratchpad のコピーでの mutation だけ
