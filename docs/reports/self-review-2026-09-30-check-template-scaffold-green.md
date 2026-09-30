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
