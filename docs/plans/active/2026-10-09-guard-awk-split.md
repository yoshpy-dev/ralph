# guard-awk-split

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-09
- Related request: ユーザーの依頼「残っているguard の後続の件を修正できますか？それと、前回の一覧の4~5の着手してください。」(2026-10-09)の 5(guard のファイルが 800 行を超えるので、awk のプログラムを別のファイルに分ける)。4 と guard の後続は PR #213 で済んだ。PR #213 が tech-debt に持ち越した 2 件(guard のヘッダーの `$"…"`、テストの行)もここで扱う
- Related issue: N/A
- Type: refactor
- Branch: refactor/guard-awk-split

## Objective

`.claude/hooks/pre_bash_guard.sh` は 1596 行で、そのうち 174〜1553 行目の 1378 行が単一引用符で渡す awk のプログラムになっている。ユーザーのルール(`~/.claude/rules/code-review.md` の「Files are cohesive (<800 lines)」)を満たすよう、awk のプログラムを 3 つの `.awk` ファイルに移し、`awk -f` で読む。判定は 1 つも変えない。

あわせて、PR #213 が持ち越した 2 件を片づける(`docs/tech-debt/README.md` の guard の限界の行の (e) の (1)(2))。

1. ヘッダーの item 6(b) が、展開しない `$` に二重引用符の外の `$"…"` を挙げていない(コードは展開しないものとして扱う。振る舞いは変わらない)
2. テストの穴: zsh の修飾 `=`・`^`・`+` の添字を固定する行がない(mutation M20 が残る)。展開しない `$` の条件(後ろが空白・タブ・改行・文字列の終わり)を固定する行がない(M13 の系統が残る)

## Scope

- 新しいファイル(root と `templates/base/` の写しをバイト単位で同じに保つ)
  - `.claude/hooks/pre_bash_guard_lex.awk`: 今の awk のプログラムの「Text access」と「Lexing」の節(約 600 行)
  - `.claude/hooks/pre_bash_guard_commands.awk`: 「Simple-command assembly」と「Data regions」の節(約 320 行)
  - `.claude/hooks/pre_bash_guard_rules.awk`: 「Rule judgement」「The sentinel」「Main」の節(約 460 行)
  - 各ファイルの先頭に、そのファイルが何の節を持ち、どう呼ばれるか(3 つを `lex`、`commands`、`rules` の順に `-f` で渡す)を書いた短いコメントを置く
- `.claude/hooks/pre_bash_guard.sh` と template の写し: awk のプログラムを消し、`LC_ALL=C awk -f "$HOOK_DIR/pre_bash_guard_lex.awk" -f "$HOOK_DIR/pre_bash_guard_commands.awk" -f "$HOOK_DIR/pre_bash_guard_rules.awk"` で呼ぶ。標準入力に命令を渡すこと、stderr を捨てること、`awk_status` と旧版の 4 規則への fallback は今のまま。ヘッダーの説明のうち、awk のプログラムの位置や単一引用符に触れる文を直す
  - 移したあとの `lex_dollar` のコメントにある「このプログラムに単一引用符を書けない」の注記を消す(シェルの単一引用符で渡さなくなるので、制約がなくなる)
  - ヘッダーの item 6(b) に、二重引用符の外の `$"…"` を展開しない `$` として足す(上の 1)
- `tests/test-pre-bash-guard.sh`
  - 上の 2 の行を足す: B 節 group 11 に `echo $=arr['$(sudo ls)']`、`echo $^arr['$(sudo ls)']`、`echo $+arr['$(sudo ls)']`(どれも旧版も deny)、D 節 `edge_none` に `git commit -m "costs $ 5; never sudo ls"`、`$` のあとをタブにした行、改行にした行、`rg 'sudo ' foo$`
  - F 節(awk がないときの fallback)に、3 つの `.awk` のうち 1 つがない写しの guard を足す。旧版の 4 規則が決める(`sudo ls` は deny、`ls` は none)
  - 3 つの `.awk` を guard と同じ順と引数で読み、空の入力で exit 0、出力なしになることを確かめる行を足す(どの awk でも構文が通ることの確認。BSD awk、mawk、gawk は CI と手元で回る awk に任せる)
- `docs/tech-debt/README.md`: guard の限界の行の (d)(ファイルの大きさ)を解消済みにし、(e) の (1)(2) を解消済みにする。単一引用符の注記の項目も、分割で意味がなくなるので解消済みにする

## Non-goals

- 判定の変更。3 つの `.awk` を連結した中身は、移す前の awk のプログラムと、ファイル先頭のコメントと消す注記を除いて一致させる
- (e) のほかのコメントの項目(`cmd_pos` の `EXEC_SEEN`、`new_ctx` の配列の一覧、`lex_redir` の区切りの説明、`data_first_ok` の説明、テストのコメントの ID など、guard-deny-only の self-review から持ち越したもの)。分割の差分を「移しただけ」と読めるように保つため。tech-debt に残し、次に guard の判定を変える PR で直す
- コードの整理(`read_body` の重複したループ、判定を変えない予約語と `exec` の規則を消すこと)
- `.sh` の残り(ヘッダーのコメント、jq での取り出し、fallback、deny の文言)を別のファイルに分けること。分割後の `.sh` は約 220 行になる
- awk のプログラムをさらに細かく分けること。3 ファイルとも 800 行の目安を下回る

## Assumptions

- POSIX の awk は `-f` を何度でも受け取り、プログラムの文字列を順につなげて 1 つのプログラムとして読む。関数の定義と呼び出しはファイルをまたいでよい。BEGIN は `rules` にしかない。BSD awk(macOS)、mawk、gawk で同じに動く(PR #213 の test は ubuntu の mawk と gawk でもテストを回した)
- 3 つのうち 1 つでも読めないと awk は exit 2 で止まる。今の guard は awk が exit 0 以外で終わると旧版の 4 規則で決めるので、読めないファイルがあるときもその fallback になる
- `ralph init` と `ralph upgrade` は `templates/` を `//go:embed all:templates` で丸ごと持つ。新しいファイルは core として届き(`internal/scaffold/render.go` の `ownerForScaffoldPath` の既定)、`ralph upgrade` は今ないファイルを `OpCreate` で作る(`internal/upgrade/replaceplan.go`)。`.awk` は `FilePerm` で 0644 になり、`awk -f` で読むので実行権限は要らない
- `.claude/hooks/PreToolUse.d/10-pre-bash-guard.sh` は `exec "$(dirname "$0")/../pre_bash_guard.sh"` で呼ぶので、`HOOK_DIR` は `.claude/hooks` になる。Codex も `ralph-dispatch.sh` から同じ道を通る
- `scripts/verify.local.sh` の shellcheck と `sh -n` は `.claude/hooks/*.sh` だけを見る。`.awk` は今までも shellcheck の対象ではなかった(単一引用符の中の文字列だった)

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- 新規: `.claude/hooks/pre_bash_guard_lex.awk`、`pre_bash_guard_commands.awk`、`pre_bash_guard_rules.awk`、それぞれの `templates/base/.claude/hooks/` の写し
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (1 つのフックの中身を 3 つのファイルに移すだけで、フックの呼ばれ方・部品どうしの呼び出し・データの形を変えない。ファイルの対応は Scope の一覧で足りる)

## Design decisions

- **3 つのファイルに分け、節の境目で切る**。1 つの `.awk` に移すと 1378 行になり、800 行の目安を超える。今のプログラムは「# ===」で区切った 7 つの節を持ち、上から順に下の節が上の節の関数を使う。字句解析(Text access + Lexing)、コマンドの組み立てとデータ区間(Simple-command assembly + Data regions)、判定(Rule judgement + sentinel + Main)の 3 つに分けると、どれも 800 行を下回り、役割が名前で分かる
- **ファイル名は `pre_bash_guard_` で始める**。`.claude/hooks/` の中で `grep pre_bash_guard` が 4 つのファイルを全部拾う。下のディレクトリに置く案は、`.claude/hooks/<event>.d/` の並びと紛らわしいのでとらない
- **読めないファイルがあるときは、今の fallback に任せる**。全部 deny にする案は、部分的な `ralph upgrade` の失敗などで、すべての Bash の呼び出しを止める。fallback は旧版の 4 規則を保ち、awk が見つからないときと同じ振る舞いになる
- **1 つ目の slice は移すだけにする**。3 つの `.awk` を連結し、各ファイル先頭のコメントを除いたものが、移す前の 175〜1552 行目と一致することを `cmp` で確かめる。コメントの直しとテストの行は 2 つ目の slice に分ける
- **(e) のほかのコメントの項目は直さない**(Non-goals)。分割の差分を移しただけと読めるようにし、レビューの周回を増やさない
- Critical forks: None(ファイルの大きさはユーザーのルールが、fallback は今の guard の設計が決めている)

## Acceptance criteria

- [ ] AC1: 3 つの `.awk` がそれぞれ 800 行未満で、`pre_bash_guard.sh` の awk のプログラムがなくなる(`grep -c "LC_ALL=C awk '"` が 0)。`.sh` も 800 行未満
- [ ] AC2: 3 つの `.awk` を `lex`、`commands`、`rules` の順に連結し、各ファイルの最初の `# ====` の行より前(ファイル先頭のコメント)を除いたものが、base(0abfede5)の `pre_bash_guard.sh` の 175〜1552 行目と一致する。違うのは、単一引用符の注記を消した行(S2)だけ
- [ ] AC3: `bash tests/test-pre-bash-guard.sh` が、今の 2,032 件を含めて全部通る(jq あり・なし)。base の guard と分割後の guard に同じ入力を渡して、判定が 1 件も変わらない(テストの配列と、PR #213 の probe の行)
- [ ] AC4: F 節に足した 2 つが通る。`.awk` が 1 つない写しの guard で `sudo ls` が deny、`ls` が none。3 つの `.awk` を guard と同じ順で読むと、空の入力で exit 0、出力なし
- [ ] AC5: B 節 group 11 に足した 3 行が deny、D 節 `edge_none` に足した 4 行が none(jq あり・なし)。PR #213 の test report の mutation M20(`RE_NOTFLAG` から `=` を外す)と M13 の系統(展開しない `$` の条件を 1 つずつ外す)が、足した行で赤になる
- [ ] AC6: root と `templates/base/` の 4 つのファイルがバイト単位で同じ(`./scripts/check-sync.sh`)。`./scripts/check-template.sh` が通る
- [ ] AC7: `ralph init` で作った新しいプロジェクトと、base の ralph で作ってから分割後の ralph で `ralph upgrade` したプロジェクトの両方に、3 つの `.awk` があり、そこで動く guard が `sudo ls` を deny、`ls` を none にする
- [ ] AC8: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` の guard の限界の行で、(d) と (e) の (1)(2)、単一引用符の注記の項目が解消済みになる

## Implementation outline

1. S1(implementer): awk のプログラムを 3 つの `.awk` に移し、`.sh` を `awk -f` で呼ぶ形にする。各 `.awk` の先頭にコメントを置く。template の写し。AC1、AC2、AC3、AC6
2. S2(implementer): 単一引用符の注記を消し、ヘッダーの item 6(b) に `$"…"` を足す。テストに AC4 と AC5 の行を足す。template の写し
3. S3(inline、docs だけ): tech-debt の (d)、(e) の (1)(2)、単一引用符の注記の項目を解消済みにする。AC8

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`(`.sh` とテスト)、`./scripts/check-sync.sh`、`./scripts/check-template.sh`
- Spec compliance criteria to confirm: AC1〜AC8。AC2 は `cmp` で、AC3 は base と分割後の guard の判定の比較で確かめる
- Documentation drift to check: guard のヘッダー(awk のプログラムの位置、単一引用符)、`AGENTS.md` の `.claude/hooks/` の説明、`docs/tech-debt/README.md` の guard の行、`.claude/rules/ralph/git-commit-strategy.md`
- Evidence to capture: `wc -l` の 4 ファイル、AC2 の `cmp`、判定の比較の件数

## Test plan

- Unit tests: AC4 と AC5 の行を `tests/test-pre-bash-guard.sh` に足す
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`。AC7 の `ralph init` と `ralph upgrade` は、一時ディレクトリで手で回して証拠に残す(ビルドした ralph を使う)
- Regression tests: 既存の 2,032 件、`tests/test-lib-json.sh`、`tests/test-ralph-dispatch.sh`
- Edge cases: `.awk` が 1 つない、`.awk` が読めない(権限 000)、awk がない(F 節の既存の行)、jq がない経路
- Evidence to capture: base と分割後の判定の比較、mutation M20 と M13 の系統が赤になること

## Risks and mitigations

- 移すときに行を落とすか重ねると、判定が変わるか awk が構文エラーで止まり、旧版の 4 規則の fallback に黙って落ちる(guard はエラーを捨てる)。AC2 の `cmp` と AC4 の構文の確認で捕まえる。fallback に落ちると F 節以外の多くの行が落ちるので、テストでも分かる
- 下流のプロジェクトで `.sh` だけが更新され、`.awk` が届かないと、guard は fallback で動き、`--no-verify` などの新しい規則が効かない。`ralph upgrade` は commit barrier で一度に書く。AC7 で `upgrade` の経路を確かめる
- 手で `.sh` だけを写した下流(ralph を使わずにファイルを持ち込んだ場合)は、同じく fallback になる。ヘッダーに、4 つのファイルを一緒に置くことを書く

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く
- ロールバックはこの PR の revert(`.sh` が awk のプログラムを持つ形に戻り、`.awk` は消える)

## Open questions

- なし

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
