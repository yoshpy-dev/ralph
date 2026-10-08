# guard-zsh-data-gaps

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-09
- Related request: PR #210 の後続。`docs/tech-debt/README.md` の「Findings of the last `/cross-review` run of fix/guard-deny-only」の行の (a)〜(c) を直す。ユーザーは残タスクの一覧を見て「はいお願いします」と PR の作成を頼んだ(2026-10-09)
- Related issue: N/A
- Type: fix
- Branch: fix/guard-zsh-data-gaps

## Objective

`.claude/hooks/pre_bash_guard.sh` に残した 3 つの穴を塞ぐ。2 つは、旧版(PR #210 の前)が deny にしていた形を新版が通す穴で、1 つは新版が増やした誤検知。

1. zsh の `${(e):-'$(cmd)'}`。zsh は `(e)` フラグで値を評価し直し、単一引用符の中の置換を実行する。字句解析は、引用符の外の `${…}` の中の単一引用符を引用とみなして中の `$(` を置換として見ないので、この語がデータ区間になり、見張りの一致が無視される。読むだけのコマンドの引数(`echo ${(e):-'$(sudo ls)'}`)と、引用符の外の `git commit -m` / `git tag -m` の値で起きる(2026-10-09 に新版 none・旧版 deny を確かめた)。`$(` を単一引用符でなくバックスラッシュで隠す形(`${(e):-\$(cmd)}`)も同じで、こちらは区切りに引用符のないヒアドキュメントの本文でも起きる。`lex_hd` は `${` を見ても置換ありの印(HSUB)を立てない(そのコメントは立てると書いている)。consult と Codex の plan advisory の指摘を受けて足した。`--message''=${…}` や `-"m"${…}` のように引用符を挟んでつなげたメッセージも同じ型
2. zsh の `stat -A NAME`。zsh/stat モジュールを読み込んでいると、NAME を配列の名前として読み、添字の中の置換を実行する。`stat` は読むだけのコマンドの一覧にあるので、`stat -A 'arr[$(sudo id; echo 1)]' /dev/null` の引数がデータ区間になる
3. `tr "sudo" "abcd"`。`tr` は読むだけのコマンドの一覧(DATACMD)にあるが、引数を実行しないコマンドの一覧(NOEXEC)にないので、`scan_words` が 1 つ目の集合をコマンドの名前と読み、deny にする(旧版は none)

## Scope

- `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh`(バイト単位で同じに保つ)
  - `lex_dollar` の `${` の分岐: `${…}` の範囲を、`$(…)` と同じく、データ区間から外す範囲として登録する(`xnote`)。中に置換があるかどうかの印(`LD_SUBST`)は今のまま変えない。そうすると、`echo "${HOME}" 'sudo ls'` の `'sudo ls'` のように `${…}` の外にある文字は、これまでどおりデータとして読める
  - `msg_check`: コミットとタグのメッセージの語の、元の語全体の生の文字(`WR[ctx, j]`。`msg_attached` が渡す切り出しではない)が `${` を含むときは、データ区間にしない。ただし推奨の HEREDOC の形(`safe_heredoc_msg`)は、区切りに引用符があり中身を展開しないので、今までどおりデータにする。deny にはしない(`git commit -m "${msg}"` を止めないため)。データ区間でなくなった語は、見張りが旧版どおりに決める
  - `lex_hd`: 区切りに引用符のないヒアドキュメントの本文で `${` を見たら、置換ありの印(HSUB)を立てる。本文はデータ区間 (c) でなくなり、見張りが旧版どおりに決める。`git commit -F -` の本文の deny(`heredoc_done` の `$(` とバッククォートの部分文字列)は変えない
  - DATACMD の一覧から `stat` を外す
  - NOEXEC の一覧に `tr` を足す
  - ヘッダーのデータ区間の説明 (a)・(b) と、`lex_dollar` のコメントに、上の 3 点を書く
- `tests/test-pre-bash-guard.sh`: 下の AC の形を足す。旧版も deny にする形は B 節の `guard_deny_only_forms` に入れ、AC7 の比較の例に含める
- `docs/tech-debt/README.md`: 「Findings of the last `/cross-review` run」の行を解消済みにし、guard の限界の行とテストの穴の行で、この 3 点に触れているところを直す

## Non-goals

- `${…}` の中を字句解析で正確に読むこと(zsh のフラグを 1 つずつ解釈すること)。範囲ごとデータから外せば足りる
- `${…}` 以外の仕組みで zsh や bash が評価する形(`$[…]` の算術、ほかのモジュールの builtin、起動ファイルの設定に依存する形など)。見つかれば tech-debt の guard の行に記録する
- 変数の語で `--pre` を渡す `rg $x …` など、実行時にしか分からない穴(tech-debt の guard の行に残す)
- guard のファイルを分けること(800 行の目安を超えている件)

## Assumptions

- 新版の guard は PR #210(0931f791)のもの。旧版は `tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh`
- zsh の振る舞いは `zsh -f`(5.9)で、無害な `echo … >&2` を置換に入れて確かめた。`${(e):-'…'}` は引用符の外でも二重引用符の中でも置換を実行し、`(e)` がなければ bash も zsh も文字のまま扱う。二重引用符の中の形、引用符の外のヒアドキュメントの本文の形は、いまの guard もすでに deny にする

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (guard の 1 ファイルとその template の写し、テスト、tech-debt の記録だけの変更で、部品どうしの呼び出し・状態・データの形・境界を変えない)

## Design decisions

- **`${…}` は範囲ごとデータから外す**。zsh のフラグ(`(e)`、`(P)` など)だけを見分ける案もあるが、フラグの書き方は多く(`${(e)x}`、`${(%)x}`、`${(ej:x:)…}`)、取りこぼすと同じ型の穴がまた残る。範囲ごと外しても、`${…}` の外の文字はデータのまま読めるので、止めすぎは `${…}` の中に見張りの語がある形だけになる。どれも旧版も deny にする
- **メッセージの `${` は deny にせず、データ区間から外すだけにする**。コミットメッセージの置換の規則(`commit_message`)に含めると、`git commit -m "${msg}"` を新しく止めてしまう(旧版は通す)。データ区間から外せば、見張りが旧版どおりに決める。`msg_check` の中では、今の順序(置換ありの語で推奨の HEREDOC の形かどうかを先に見る)を保ち、`${` の検査はそのあとに置く。先に置くと、本文に `${` を書いた推奨の形がデータから落ちる
- **直す範囲の線引き**: 直すのは「`${…}` の型 × データ区間の (a) 引数・(b) メッセージ・(c) ヒアドキュメントの本文」と、`stat`・`tr` の一覧の変更だけ。self-review や cross-review が別の仕組みの形を見つけたら、tech-debt の guard の行に記録する。記録だけで済む指摘には、post-implementation-pipeline.md の「記録だけの修正」の例外を使う
- **`stat` は一覧から外す**。`-A` があるときだけ外す案もあるが、`-A` は値を取るオプションの束(`-LA`)にもなりうる。`stat` の引数に見張りの語を書くことはまれなので、外しても困らない
- Critical forks: None

## Acceptance criteria

- [ ] AC1: 次がどのモードでも deny になる(jq あり・なし)。どれも旧版も deny にする
  - `echo ${(e):-'$(sudo ls)'}`
  - `echo x${(e):-'$(sudo ls)'}y`
  - `echo ${(e):-\$(sudo ls)}`、`echo "${(e):-\$(sudo ls)}"`
  - `cat <<EOF` の本文が `${(e):-\$(sudo ls)}` の形
  - `git commit -m ${(e):-'$(sudo ls)'}`
  - `git tag -a v1 -m ${(e):-'$(sudo ls)'}`
  - `git commit --message''=${(e):-'$(sudo ls)'}`、`git commit -"m"${(e):-'$(sudo ls)'}`、`git tag -a v1 --message''=${(e):-'$(sudo ls)'}`
  - `stat -A 'arr[$(sudo id; echo 1)]' /dev/null`
- [ ] AC2: 次がどのモードでも none になる(jq あり・なし)
  - `tr "sudo" "abcd"`、`echo x | tr "sudo" "abcd"`
  - `git commit -m "${msg}"`、`git commit -m "$msg"`
  - `echo "${HOME}" 'sudo ls'`(`${…}` の外の文字はデータのまま)
  - 推奨の HEREDOC の形で、本文に `${HOME}` と `sudo ls` を書いたコミット
  - `git commit -F - <<EOF` の本文が `use ${HOME} here` の形(本文に見張りの語がない)
  - `stat -f %z file`
- [ ] AC3: plan 2026-10-07-guard-deny-only の AC3 の 29 形(テストの C 節の `ac3`)は none のまま。G 節で、旧版 deny から新版 none になる形が `intentional_fixes` の 13 件のままであることを、テストが確かめる
- [ ] AC4: `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh` がバイト単位で同じ(`./scripts/check-sync.sh`)
- [ ] AC5: `bash tests/test-pre-bash-guard.sh` と `bash tests/test-lib-json.sh` が通り、`./scripts/run-verify.sh` が rc 0
- [ ] AC6: `docs/tech-debt/README.md` の「Findings of the last `/cross-review` run of fix/guard-deny-only」の行が解消済みになり、guard の限界の行とテストの穴の行から (a)〜(c) の未解決の記述がなくなる

## Implementation outline

1. S1(1 slice、implementer): guard の 5 点(`lex_dollar` の `xnote`、`msg_check`、`lex_hd` の HSUB、DATACMD から `stat`、NOEXEC に `tr`)とコメント、template の写し、テスト(AC1〜AC3 の形)
2. S2(inline か doc-maintainer): tech-debt の 3 行を直す(AC6)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning`、`./scripts/check-sync.sh`、awk のプログラムに単一引用符がないこと
- Spec compliance criteria to confirm: AC1〜AC6。AC1 と AC2 は probe(`scratchpad/zg/`)でも新旧を比べる
- Documentation drift to check: ヘッダーのデータ区間の説明、tech-debt の 3 行
- Evidence to capture: probe の新旧比較、テストの件数

## Test plan

- Unit tests: `tests/test-pre-bash-guard.sh` に AC1 の 5 形(B 節、AC7 の比較に入る)と AC2 の形(D 節の `edge_none`)を足す
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: 既存の 1,886 件、`tests/test-lib-json.sh` の 126 件
- Edge cases: 二重引用符の中の `${…}`(いまも deny)、`${…}` の中に `$(` がある形(いまも deny)、`${…}` の外の見張りの語(none のまま)、推奨の HEREDOC の形の本文の `${HOME}`(none のまま)
- Evidence to capture: mutation(`xnote` を外す、`msg_check` の条件を外す、`msg_check` で切り出しの文字を見る、`lex_hd` の HSUB を外す、`stat` を戻す、`tr` を外す)でテストが赤になること

## Risks and mitigations

- `${…}` の範囲を外すことで、`${…}` の中に見張りの語がある無害な形(`echo ${x:-'sudo ls'}`)も deny になる。旧版も deny にするので、AC7 には反しない
- `msg_check` の条件で、`${` を含む引用符の外のメッセージが見張りに戻る。`git commit -m "${msg}"` は見張りの規則に当たらないので none のまま(AC2)
- 同じ型の穴(zsh が評価するのに字句解析がデータとみなす形)は、ほかにも残りうる。前の PR では cross-review を 4 周回した。この PR では Design decisions の線引きに沿って、`${…}` の型の形だけを直し、別の仕組みの形は tech-debt に記録する
- `lex_hd` の変更で、区切りに引用符のないヒアドキュメントの本文に `${…}` があると、本文に見張りの語がある形が deny になる(`cat <<EOF` の本文が `${HOME} sudo ls`)。旧版も deny にする
- `stat` を外すと、`stat` が先頭のコマンドの呼び出し全体がデータ区間を持たなくなる(`stat f; echo 'sudo ls'` が deny)。旧版も deny にする

## Rollout or rollback notes

- マージすると、main のチェックアウトから動く session にすぐ効く。下流には `ralph upgrade` で core として届く
- ロールバックはこの PR の revert

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
