# guard-awk-split

- Status: Approved
- Approved: 2026-10-09 sha256:cfb34e566022
- Owner: Claude Code
- Date: 2026-10-09
- Related request: ユーザーの依頼「残っているguard の後続の件を修正できますか？それと、前回の一覧の4~5の着手してください。」(2026-10-09)の 5(guard のファイルが 800 行を超えるので、awk のプログラムを別のファイルに分ける)。4 と guard の後続は PR #213 で済んだ。PR #213 が tech-debt に持ち越した 2 件(guard のヘッダーの `$"…"`、テストの行)と、guard-deny-only から持ち越してきたコメントの項目もここで扱う
- Related issue: N/A
- Type: refactor
- Branch: refactor/guard-awk-split

## Objective

`.claude/hooks/pre_bash_guard.sh` は 1596 行で、そのうち 175〜1552 行目の 1378 行が、単一引用符で渡す awk のプログラムになっている。ユーザーのルール(`~/.claude/rules/code-review.md` の「Files are cohesive (<800 lines)」)を満たすため、awk のプログラムを 3 つの `.awk` ファイルに移し、`awk -f` で読む。判定は 1 つも変えない。

あわせて、`docs/tech-debt/README.md` の guard の限界の行の (e) にあるコメントの項目を片づける。(e) の Trigger は「次に guard を変える change でコメントを直す」で、3 回持ち越されてきた。判定を変えないこの PR が、直すのにいちばん安全な場所になる。

1. PR #213 が足した 2 件: ヘッダーの item 6(b) が、展開しない `$` に二重引用符の外の `$"…"` を挙げていない(振る舞いは変わらない)。テストの穴として、zsh の修飾 `=`・`^`・`+` の添字を固定する行がない(mutation M20 が残る)。展開しない `$` の条件(後ろが空白・タブ・改行・文字列の終わり)を固定する行もない(M13 の系統が残る)
2. guard-deny-only から持ち越したコメントだけの項目(Scope の S3 に並べる)

## Scope

- S1(移すだけ): 新しい 3 つのファイルに awk のプログラムを移す。root と `templates/base/` の写しはバイト単位で同じに保つ
  - `.claude/hooks/pre_bash_guard_lex.awk`: 今のプログラムの 175〜773 行目(「Text access」「Lexing」の節、599 行)
  - `.claude/hooks/pre_bash_guard_commands.awk`: 774〜1093 行目(「Simple-command assembly」「Data regions」の節、320 行)
  - `.claude/hooks/pre_bash_guard_rules.awk`: 1094〜1552 行目(「Rule judgement」「The sentinel」「Main」の節、459 行)
  - `.claude/hooks/pre_bash_guard.sh` と template の写し: awk のプログラムを消し、`LC_ALL=C awk -f "$HOOK_DIR/pre_bash_guard_lex.awk" -f "$HOOK_DIR/pre_bash_guard_commands.awk" -f "$HOOK_DIR/pre_bash_guard_rules.awk"` で呼ぶ。標準入力に命令を渡すこと、stderr を捨てること、`awk_status` と旧版の 4 規則への fallback は今のまま
  - この slice では `.awk` にファイル先頭のコメントを置かず、中身を 1 文字も変えない(AC2)
- S2(PR #213 の持ち越しと分割の後始末)
  - 各 `.awk` の先頭に、そのファイルが持つ節と、guard が 3 つを `lex`、`commands`、`rules` の順に `-f` で渡すことを書いた短いコメントを置く
  - `lex_dollar` のコメントにある「このプログラムに単一引用符を書けない」の注記を消す(シェルの単一引用符で渡さなくなるので、制約がなくなる)。`.sh` のヘッダーで awk のプログラムの位置に触れる文を直し、4 つのファイルを一緒に置くことを書く
  - ヘッダーの item 6(b) に、二重引用符の外の `$"…"` を展開しない `$` として足す
  - `tests/test-pre-bash-guard.sh`: B 節 group 11 に `echo $=arr['$(sudo ls)']`、`echo $^arr['$(sudo ls)']`、`echo $+arr['$(sudo ls)']`(どれも旧版も deny)。D 節 `edge_none` に `git commit -m "costs $ 5; never sudo ls"`、`$` のあとをタブにした行、改行にした行、`rg 'sudo ' foo$`。F 節に、3 つの `.awk` のうち 1 つがない写しの guard(写しには `lib_json.sh` も置く。旧版の 4 規則が決め、`sudo ls` は deny、`ls` は none)と、3 つの `.awk` を guard と同じ順と引数で読むと空の入力で exit 0・出力なしになる行
  - `scripts/check-template.sh` と template の写し: `required_files` に guard の 5 つのファイル(`.claude/hooks/pre_bash_guard.sh`、`lib_json.sh`、3 つの `.awk`)を足す。下流の検査と CI の `check-template` で、欠けたファイルが分かるようにする
- S3(guard-deny-only から持ち越したコメントだけの項目。tech-debt の (e) の文面のとおり)
  - `cmd_pos` のコメントに、`EXEC_SEEN` を立て、それを `end_cmd` が `judge` のあとで読むことを書く
  - `new_ctx` の前のコメントの、文脈ごとの配列の一覧に `HPQ` と `HPN` を足す
  - `lex_redir` の区切りの規則の上のコメントを、シェルが区切りを文字のまま読むことに合わせて直す(区切りの `$'…'` の `\n`・`\t`・`\r` 以外のエスケープと `$"…"` だけを、字句解析とシェルが違って読む)
  - `data_first_ok` の上のコメントの「シェルが見る値」を、字句解析が `$'…'` の `\n`・`\t`・`\r` だけを戻すことに合わせて直す
  - ヘッダーの「Not covered」の段落に、printf は語を書かれたとおりに読むことを書く(rg は PR #213 から `lex_dollar` の印を読む)
  - テストのコメント: ヘッダーの item B に `guard_deny_only_forms` が足してきた形の種類を書き、配列の前の段落を group 11 まで伸ばす。group 1 の前の「were false none before」を直す(`(echo sudo ls)` と `if grep -q` の形は文字を出すだけで、トップレベルのグループと複合コマンドにデータ区間がないので deny になる。旧版も deny)。`edge_deny` の「Cycle 2 (P2-4)」、`edge_none` の「Cross-review cycle 2 (P2-4, P2-5)」、`edge_sentinel_deny` の「(change A, cycle 2)」を、triage の番号と allowlist(a9ef82b1)の名前に直す
- S4(docs だけ): `docs/tech-debt/README.md` の guard の限界の行で、(d)(ファイルの大きさ)と、(e) のうち S2・S3 で直した項目を解消済みにする。残る (e) のコード側の項目(下の Non-goals)は残す

## Non-goals

- 判定の変更。S1 の 3 つの `.awk` を連結したものは、移す前の awk のプログラムとバイト単位で一致させる
- (e) のうちコードを変える項目: `read_body` の、末尾のバックスラッシュを数えるループの重複。判定を変えない予約語と `exec` の規則を消すこと(mutation N03・N04 が等価)と、それに伴うヘッダーと `in_data` の上のコメント
- `.sh` の残り(ヘッダーのコメント、jq での取り出し、fallback、deny の文言)を別のファイルに分けること。分割後の `.sh` は約 220 行になる
- awk のプログラムをさらに細かく分けること。3 ファイルとも 800 行の目安を下回る
- `ralph upgrade` の適用順(delete、create、update)や途中失敗の扱いを変えること

## Assumptions

- POSIX の awk は `-f` を何度でも受け取り、プログラムの文字列を順につなげて 1 つのプログラムとして読む。関数の定義と呼び出しはファイルをまたいでよい。BEGIN は `rules` にしかない。BSD awk(macOS)、mawk、gawk で同じに動く(PR #213 の test は ubuntu の mawk と gawk でもテストを回した)
- 3 つのうち 1 つでも読めないと awk は exit 2 で止まる。今の guard は awk が exit 0 以外で終わると旧版の 4 規則で決めるので、読めないファイルがあるときもその fallback になる
- `ralph init` と `ralph upgrade` は `templates/` を `//go:embed all:templates` で丸ごと持つ。新しいファイルは core として届き(`internal/scaffold/render.go` の `ownerForScaffoldPath` の既定)、`ralph upgrade` は今ないファイルを `OpCreate` で作る。適用の順は delete、create、update(`internal/upgrade/replaceplan.go:94`)。分割を入れる向きでは `.awk` を作ってから `.sh` を書き換えるので、途中で止まっても古い 1 ファイルの `.sh` が残り、awk の経路で動く。`.awk` は `FilePerm` で 0644 になり、`awk -f` で読むので実行権限は要らない
- `.claude/hooks/PreToolUse.d/10-pre-bash-guard.sh` は `exec "$(dirname "$0")/../pre_bash_guard.sh"` で呼ぶので、`HOOK_DIR` は `.claude/hooks` になる。Codex も `ralph-dispatch.sh` から同じ道を通る
- `scripts/verify.local.sh` の shellcheck と `sh -n` は `.claude/hooks/*.sh` だけを見る。`.awk` は今までも shellcheck の対象ではなかった(単一引用符の中の文字列だった)

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- 新規: `.claude/hooks/pre_bash_guard_lex.awk`、`pre_bash_guard_commands.awk`、`pre_bash_guard_rules.awk`、それぞれの `templates/base/.claude/hooks/` の写し
- `tests/test-pre-bash-guard.sh`
- `scripts/check-template.sh`、`templates/base/scripts/check-template.sh`
- `docs/tech-debt/README.md`

## Visual review

None (1 つのフックの中身を 3 つのファイルに移すだけで、フックの呼ばれ方・部品どうしの呼び出し・データの形を変えない。ファイルの対応は Scope の行番号の一覧で足りる)

## Design decisions

- **3 つのファイルに分け、節の境目で切る**。1 つの `.awk` に移すと 1378 行になり、800 行の目安を超える。今のプログラムは「# ===」で区切った 7 つの節を持ち、上から順に下の節が上の節の関数を使う。字句解析(Text access + Lexing)、コマンドの組み立てとデータ区間(Simple-command assembly + Data regions)、判定(Rule judgement + sentinel + Main)の 3 つに分けると、どれも 800 行を下回り、役割が名前で分かる
- **ファイル名は `pre_bash_guard_` で始める**。`.claude/hooks/` の中で `grep pre_bash_guard` が 4 つのファイルを全部拾う。下のディレクトリに置く案は、`.claude/hooks/<event>.d/` の並びと紛らわしいのでとらない
- **読めないファイルがあるときは、今の fallback に任せる**。全部 deny にする案は、部分的な `ralph upgrade` の失敗などで、すべての Bash の呼び出しを止める。fallback は旧版の 4 規則を保ち、awk が見つからないときと同じ振る舞いになる。欠けたファイルは `check-template.sh` の `required_files` で見えるようにする(consult の指摘)
- **S1 は移すだけにし、`cmp` で確かめる**。S1 のコミットでは `.awk` に何も足さず、3 つを連結したものが base の 175〜1552 行目とバイト単位で一致する。ファイル先頭のコメント、単一引用符の注記、ほかのコメントの直しは S2 と S3 に分ける(consult の指摘)
- **(e) のコメントだけの項目は直す**。「移しただけ」は S1 の `cmp` が保証するので、S2 と S3 にコメントの直しを入れても崩れない。判定を変えない PR は、コメントを直すのにいちばん安全な場所になる。コードを変える項目は Non-goals に残す(consult の指摘)
- **ロールバックは 2 段にする**。この PR をそのまま revert した版を下流に配ると、`ralph upgrade` は 3 つの `.awk` を先に消してから `.sh` を書き戻す。そこで書き込みが失敗すると、分割版の `.sh` だけが残り、旧版の 4 規則で動く(`--no-verify` や `core.hooksPath` を止められない)。戻すときは、まず `.awk` を残したまま `.sh` を 1 ファイルの形に戻して配り、`.awk` は後の変更で消す(Codex の plan advisory の HIGH)
- Critical forks: None(ファイルの大きさはユーザーのルールが、fallback は今の guard の設計が、ロールバックの手順は `ralph upgrade` の適用順が決めている)

## Acceptance criteria

- [ ] AC1: 3 つの `.awk` と `pre_bash_guard.sh` が、どれも 800 行未満になる。`pre_bash_guard.sh` に awk のプログラムが残らない(`grep -c "LC_ALL=C awk '"` が 0)
- [ ] AC2: S1 のコミットで、`cat` で 3 つの `.awk` を `lex`、`commands`、`rules` の順に連結したものが、base(0abfede5)の `pre_bash_guard.sh` の 175〜1552 行目と `cmp` で一致する
- [ ] AC3: `bash tests/test-pre-bash-guard.sh` が、今の 2,032 件と足した行を含めて全部通る(jq あり・なし)。S1 のコミットでも、足す前の 2,032 件が全部通る
- [ ] AC4: F 節に足した 2 つが通る。`.awk` が 1 つない写しの guard で、`sudo ls` が deny、`ls` が none。3 つの `.awk` を guard と同じ順で読むと、空の入力で exit 0、出力なし
- [ ] AC5: B 節 group 11 に足した 3 行が deny、D 節 `edge_none` に足した 4 行が none(jq あり・なし)。PR #213 の test report の mutation M20(`RE_NOTFLAG` から `=` を外す)と M13 の系統(展開しない `$` の条件を 1 つずつ外す)が、足した行で赤になる
- [ ] AC6: root と `templates/base/` の写し(guard の 4 つ、`check-template.sh`)がバイト単位で同じ(`./scripts/check-sync.sh`)。`./scripts/check-template.sh` が通り、`.awk` を 1 つ消した写しの木では `Missing required file` で失敗する
- [ ] AC7: `ralph init` で作った新しいプロジェクトと、base の ralph で作ってから分割後の ralph で `ralph upgrade` したプロジェクトの両方に、3 つの `.awk` があり、そこで動く guard が `sudo ls` を deny、`ls` を none、`git commit --no-verify -m x` を deny にする(最後の形は旧版の 4 規則の fallback では none なので、awk の経路が動いていることが分かる)
- [ ] AC8: S3 の項目のコメントが直っている(tech-debt の (e) の文面と照らす)。S2 と S3 のコミットで、awk のプログラムのコメント以外の行が変わっていない(`git diff` でコメントの行だけ)
- [ ] AC9: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` の guard の限界の行で、(d) と、(e) のうち S2・S3 で直した項目が解消済みになり、コードを変える項目だけが残る

## Implementation outline

1. S1(implementer): awk のプログラムを 3 つの `.awk` に移し、`.sh` を `awk -f` で呼ぶ形にする。template の写し。AC1、AC2、AC3(足す前の件数)、AC6 の写しの一致
2. S2(implementer): `.awk` の先頭のコメント、単一引用符の注記、`.sh` のヘッダー、item 6(b) の `$"…"`、テストの行(AC4、AC5)、`check-template.sh` の `required_files`(AC6)。template の写し
3. S3(implementer): (e) のコメントだけの項目(AC8)。template の写し
4. S4(inline、docs だけ): tech-debt の (d) と (e) を解消済みにする(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`(`.sh`、`check-template.sh`、テスト)、`./scripts/check-sync.sh`、`./scripts/check-template.sh`
- Spec compliance criteria to confirm: AC1〜AC9。AC2 は S1 のコミットで `cmp`、AC8 は S2 と S3 の差分でコメント以外の行が変わっていないこと
- Documentation drift to check: guard のヘッダー(awk のプログラムの位置、単一引用符、4 つのファイル)、`AGENTS.md` の `.claude/hooks/` の説明、`docs/tech-debt/README.md` の guard の行、`.claude/rules/ralph/git-commit-strategy.md`
- Evidence to capture: `wc -l` の 4 ファイル、AC2 の `cmp`、base と分割後の判定の比較の件数

## Test plan

- Unit tests: AC4 と AC5 の行を `tests/test-pre-bash-guard.sh` に足す
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`。AC7 の `ralph init` と `ralph upgrade` は、一時ディレクトリでビルドした ralph を使って手で回し、証拠に残す
- Regression tests: 既存の 2,032 件、`tests/test-lib-json.sh`、`tests/test-ralph-dispatch.sh`。base の guard と分割後の guard に同じ入力(テストの配列と PR #213 の probe の行)を渡し、判定が 1 件も変わらないこと
- Edge cases: `.awk` が 1 つない、`.awk` が読めない(権限 000)、awk がない(F 節の既存の行)、jq がない経路
- Evidence to capture: base と分割後の判定の比較、mutation M20 と M13 の系統が赤になること、AC7 の 3 つの形の判定

## Risks and mitigations

- 移すときに行を落とすか重ねると、判定が変わるか awk が構文エラーで止まり、旧版の 4 規則の fallback に黙って落ちる(guard はエラーを捨てる)。AC2 の `cmp` と AC4 の構文の確認で捕まえる。fallback に落ちると F 節以外の多くの行が落ちるので、テストでも分かる
- 下流で `.awk` が欠けると、guard は fallback で動き、`--no-verify` などの新しい規則が効かない。分割を入れる向きの `ralph upgrade` は `.awk` を先に作るので、途中で止まっても古い `.sh` が残る。欠けたファイルは `check-template.sh` で見え、AC7 は fallback と判定が違う形で awk の経路を確かめる
- この PR を revert して配ると、`.awk` を消したあとに `.sh` の書き込みが失敗する窓ができる。ロールバックは 2 段にする(Rollout を参照)。`ralph upgrade` をもう一度回すと、残りの作業を終える(`ApplyOps` のコメント)
- 手で `.sh` だけを写した下流(ralph を使わずにファイルを持ち込んだ場合)は fallback になる。`.sh` のヘッダーに、4 つのファイルを一緒に置くことを書く
- S3 のコメントの直しで、レビューの周回が増える。直す項目は tech-debt の (e) の文面に限り、新しい指摘で直すのは誤りだけにする

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く
- 配る前(release の前)に戻すなら、この PR の revert で足りる
- release のあとに戻すときは 2 段にする。1 段目は、`.sh` を 1 ファイルの形(awk のプログラムを持つ形)に戻し、3 つの `.awk` は残して配る(`.sh` は `.awk` を読まないので、残っても害はない)。2 段目で、`.awk` を消す変更を配る

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-09: ユーザーが承認ゲートで Approve。承認の前に、Codex の plan advisory の HIGH(revert を配ると `.awk` を消したあとの `.sh` の書き込み失敗で fallback に落ちる窓ができる)を 2 段のロールバックと AC7 の `--no-verify` の形で、consult(consult-plan-awksplit)の指摘 3 点((e) のコメントだけの項目を S3 で直す、`check-template.sh` の `required_files`、AC2 を S1 の素の `cmp` に)を反映した。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
