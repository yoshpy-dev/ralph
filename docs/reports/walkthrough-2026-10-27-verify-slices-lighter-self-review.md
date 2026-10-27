# Walkthrough: verify-slices-lighter-self-review(スライスごとの検証を計画のコマンドにし、/self-review を 6 観点に絞る)

- Date: 2026-10-27(UTC)
- Plan: docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md(/pr で archive に移す)
- Issue: #261
- Branch: refactor/verify-slices-lighter-self-review(この walkthrough の前の HEAD be63c8a5。29 commits、48 files、+1303/-122)
- Base: 44c14b97(origin/main、「Merge pull request #260」)
- 読む範囲: Go のコードは変えていない。中身は skill・rule・agent の定義の文と、shell のテスト 2 本。
  - 同じ文の写しが 4 か所にある。`.claude/`、`.agents/`(`sync-skills.sh` で生成)、`templates/base/.claude/`、`templates/base/.agents/`。読むのは `.claude/` の側だけでよい。
  - agent の定義は Claude の `.md` と Codex の `.toml` に、それぞれ root と template の 2 本がある。
  - テストは `tests/test-slice-verification.sh`(新規、+304)と `tests/test-self-review-scope.sh`(+207)。
  - 記録は計画(+148)、後工程の report 5 本(+334)、insight event(+13)、tech-debt(+3)。
- 後工程は 2 回まわした(cycle 2/2)。1 回目の cross-review が ACTION_REQUIRED を 1 件出し、直したあと全体を回し直した。2 回目の cross-review の所見は 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → 計画の雛形(S1) → /implement の手順 6・7・12(S2) → implementer の確かめ(S3) → /self-review の 6 観点(S4) → テスト(S5) → レビューで足した再開の道 → 確かめていないこと
```

## 1. この PR が何か

- これまで: `/implement` は、スライスごとと手順 12 で `./scripts/run-verify.sh`(静的検査とテストのフル実行)を打っていた。このリポジトリでは 1 回 10〜18 分かかる。後工程の `/verify` と `/test` も同じ検査をもう一度走らせていた。`/self-review` は 10 項目を見ていて、その中には機械的に確かめられる 4 項目(不要な変更、誤字とコピペの誤り、デバッグの残り、秘密情報)もあった。
- この PR から:
  - 計画の `## Implementation outline` に、スライスごとの検証コマンドを書く。触ったところだけを確かめる、いちばん狭いコマンドにする。
  - `/implement` はそのコマンドを implementer の handoff に写す。手順 12 ではフルの検査を打たない。フルの静的検査とテストは、`/verify` と `/test` が 1 回ずつ走らせる。
  - implementer は、コミットの前に staged diff で 5 項目を確かめる。不要な変更、誤字とコピペの誤り、デバッグの残り、秘密情報、スライスの外のファイルの 5 つで、結果を項目ごとに報告する。
  - `/self-review` は残りの 6 観点だけを見る。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 つ。
- レビューで足した道: implementer は、秘密情報を範囲内で消せないときや、検証の失敗を範囲内で直せないとき、コミットせずに止まる。そのスライスは「再開」として出し直す(下の 6)。
- release のノートに書くこと: この branch のコミットはすべて `docs:` か `test:` なので、`.goreleaser.yml` が changelog から外す。ノートに手で 1 行足す必要があり、その文の案は sync-docs report の hand-off 節にある。
  - 雛形の `docs/plans/templates/feature-plan.md` は seed のファイルで、`ralph upgrade` は上書きせず advisory を出すだけ。既存のプロジェクトでは、ユーザーが自分で雛形を更新する。
- 戻し方: この PR を revert する。

## 2. 計画の雛形(S1、168c3950)

- `docs/plans/templates/feature-plan.md:36-42` と `.claude/skills/plan/template.md` の `## Implementation outline` に、HTML のコメントが 3 つと例の行 `1. <slice> — verify: \`<command>\`` が入った
- コメントが言うこと:
  - コマンドは、そのスライスが触るファイルを覆ういちばん狭い検査にする。
  - Go なら `test -z "$(gofmt -l internal/foo)" && go vet ./internal/foo/ && go test ./internal/foo/` を使う。`gofmt -l` は未整形のファイルを一覧しても終了コード 0 を返すので、`test -z` で包む。
  - `RALPH_VERIFY_SCOPE=changed` では絞れない。これは言語パックを選ぶだけで、Go の中では `go test ./...` を打つ。
- `/plan` の手順 7(`.claude/skills/plan/SKILL.md`)の「implementation outline」に、各項目の最後にそのスライスの検証コマンドを書くことを足した
- 見ること: 雛形は seed なので、既存のプロジェクトには `ralph upgrade` で入らない(上の release のノート)

## 3. /implement の手順 6・7・12(S2、dd7d4070、cb094ae8、804d05f9)

- 手順 6(`.claude/skills/implement/SKILL.md:40`): handoff の検証コマンドは、計画の項目の「— verify: `<command>`」から取る。計画にコマンドがない場合は、いちばん狭いコマンドを選んで計画の `## Progress checklist` に書いてから出す。古い雛形で承認された計画がこれに当たる。digest はこの節を読まないので、承認をやり直さなくてよい(D4)。フルの `run-verify.sh` を使うのは、ほかに覆うものがないときだけ。
- 手順 7(`:42-43`):
  - 委譲したスライスの判定に「5 項目それぞれに結果があり、found and not fixed がない」を足した。
  - found and not fixed の項目は、次のスライスの前に別のスライスとして直させる。最後のスライスなら手順 13 の前に直させる。
  - inline のスライスでも、orchestrator が同じ確かめをする(D6)。
- 手順 12(`:49`): フルの `run-verify.sh` をやめ、各スライスの検証が通ったことを報告と Progress checklist で確かめるだけにした
- Completion gate(`:80-`): 「Every slice's verification command passed」と「`/verify` と `/test`(手順 13)が通った」の 2 行
- `.claude/rules/ralph/git-commit-strategy.md:10` の Validation Gate の手順 2 も、計画のスライスのコマンドを打つ形にした
- `.claude/rules/ralph/model-routing.md:29-56`:
  - handoff の表の「Exact verification commands」の行が、計画の `## Implementation outline` を指す。
  - report contract に機械的な確かめの結果を足し、5 項目の正本の並びを置いた。ほかの言及は、すべてこの並びを一字一句くり返す。
  - `.claude/rules/ralph/subagent-policy.md` も同じ向きに直した。
- 見ること: model-routing は root と template で中身が違う(KNOWN_DIFF)ので、2 本を別々に直してある

## 4. implementer の確かめ(S3、f1c68ebd、cb094ae8、bc56cc4f、d33bb358)

- `.claude/agents/implementer.md:33-42` の「Mechanical self-check before commit」:
  - stage したあと `git diff --cached` を読み、5 項目を確かめる。各項目に何を探すかの 1 行がある。
  - 範囲内で直せない項目は残して found and not fixed として報告し、理由も書く。
  - 秘密情報だけは扱いが違う。範囲内で消せなければコミットせず、unstage して止まる。報告にはファイルと行を書き、値は書かない。
- report contract(`:44-50`)は項目ごとに clean、fixed、found and not fixed のどれかを返す
- Codex の `.codex/agents/implementer.toml` も同じ文。root と template はどちらも `cmp` で一致する
- 見ること: 文の最後の「These are the checks `/self-review` leaves to the implementing side, so do not skip them.」は、変更の経緯ではなく理由を書く文にしてある(テストが「no longer」「first four」を禁じる)

## 5. /self-review の 6 観点(S4、d501efad)

- `.claude/skills/self-review/SKILL.md:16-21` の番号付きの観点が 6 つになった。`:23-` の段落は、外した 4 項目を実装の側が確かめると言う。秘密情報は `/verify` の branch の secret scan と CI の secret scan でも拾う。スクリプトの名前は書かない(D5、テストが禁じる)
- `template.md` の Area の値も 6 つ(naming、readability、null-safety、exception-handling、security、maintainability)
- `.claude/agents/reviewer.md` と `.codex/agents/reviewer.toml` の focus の行も 6 観点
- 見ること: frontmatter の `description` も 6 観点に直してある(sync-docs で確かめた)

## 6. レビューで足した再開の道(dba1c20f、d33bb358、6741f9d6)

- cross-review の 1 回目が指摘した点(P2): 秘密情報で止まった implementer は、範囲内の未コミットの変更を作業ツリーに残す。新しい implementer を出すと、「Before any edit」の確かめ(`implementer.md:21`)が既存の変更を見つけて止まり、立て直せない。
- 直し方(consult の判定で、秘密情報に限らず「コミットせずに止まったスライス」全般にした):
  - `implementer.md:21` の段落に例外を 1 文足した。handoff が止まったスライスの再開だと述べ、範囲内の未コミットの変更をそのスライスのものと名指しするときは、それを引き継ぐ。止まらない。
  - 手順 7(`.claude/skills/implement/SKILL.md:42`)は次の順にした。
    1. 秘密情報を作業ツリーから出す。セッション終了と圧縮前の `wip:` チェックポイントが `git add -A` で全部 stage するため。
    2. 必要ならスライスを絞るか、計画を直す。
    3. 再開として出し直す。
    4. 絞って外したファイルは、追跡中なら `git restore --staged --worktree -- <file>` で戻し、新規なら消す。別のスライスに回してもよい。
  - model-routing の handoff の表に「Resume」の行を足した。手順 6 にも、再開が引き継ぐ未コミットの変更を例外として足した。
- 見ること: Resume は implementer の「Required handoff fields」には入れていない。欠けると止まる 6 項目の一覧なので、入れるとふつうの dispatch のたびに止まる(cycle 2 の self-review で確かめた)。

## 7. テスト(S5、c77b12c9 ほか)

- `tests/test-slice-verification.sh`(150 件)が固める文:
  - 雛形、`/plan` の手順 7、`/implement` の手順 6・7・12 と Completion gate
  - Validation Gate、model-routing、subagent-policy
  - Codex の README の 5 項目の並び
- `tests/test-self-review-scope.sh`(206 件)が固めるもの:
  - 外した 4 項目が `/self-review` の観点に戻っていないこと。大文字小文字を問わずに見る。
  - reviewer の定義と template の Area の値。
  - implementer の 5 項目の正本の語、秘密情報で止まる文、再開の文。
- どちらも、改行をつないだ全文で照合する。文を折り返し直しても通る。
- 変異で効き方を確かめた。cycle 1 では、手順 6 の「narrowest」を「broadest」に変えてもテストが通った。その穴は dba1c20f でふさいだ。cycle 2 では、「as a resume」を消す、再開の文を消す、README の「and」を消す、の 3 つがそれぞれ落ちる。

## 8. 確かめていないこと

- 実際の `/implement` の実行で、新しい文がどう効くか。止まったスライスの再開も含む。テストは文を固めるだけで、振る舞いは見ていない。
- Codex の custom agent として読ませたときの実際の振る舞い
- hook の文「Run ./scripts/run-verify.sh before claiming done」が残っている(`.claude/hooks/post_edit_verify.sh`、`session_start_context.sh` とその template)。implementer がスライスごとにフルの検査を打つ合図と読むおそれがある。計画の外なので `docs/tech-debt/README.md` に記録した。
- org の座席の検証の文(`internal/org/prompts/implementer.md` など)は #262 で扱う(計画の Non-goals)
