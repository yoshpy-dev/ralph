# sync-docs report: org-regate

- Date: 2026-10-17(JST)
- Plan: `docs/plans/active/2026-10-16-org-regate.md`(承認済み。この run で計画は触っていない)
- Pipeline cycle: 2。base は `670ec219`、branch は `feat/org-regate`。run 開始時の HEAD は `f630e216`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-16-org-regate.md`、`docs/reports/verify-2026-10-16-org-regate.md`(cycle 2 の 2 回目は pass。LOW 4 件のうち V2-2〜V2-4 を下の変更 1〜3 で直す。V2-5 は変更 5 の tech-debt の行)、`docs/reports/test-2026-10-16-org-regate.md`(cycle 2 は pass)
- cycle 1 の sync-docs(0257bd93)が直した `docs/quality/quality-gates.md` の 2 面と `/org` skill の `reset --keep` の条件は、この run で触っていない。cycle 1 の表は 0257bd93 の版の report にある
- コードとテストは変えていない。変えたのは Markdown の 5 ファイル(`README.md`、仕様、tech-debt、`/org` skill の `.claude` 側 2 面)、`scripts/sync-skills.sh` で再生成した skill の写し 2 面、insight event の 1 行

## Summary

cycle 2 のコード側の変更は 3 つ。merge の前に ignored のファイルを断る確かめ(D3、85558f2c・ca840a45・c621ccc9)、読めない gate の記録の付け先を org が持ったすべてのブランチにする変更(D2、f793924d)、help の語(a59220f5・413ab307・5e12dfe0)。verify が挙げた LOW の文書のずれ 3 件(V2-2〜V2-4)を直し、残る端の入力と確かめの穴を tech-debt の 7 行にした。`ralph org regate --help`、`ralph org merge-order --help`、`spawn` と `start` の help は、コードに当てて読み、直す箇所を見つけなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の 8a-3b の注) | D3 と D2 を足した。D3: merge の前に、merge が書くパスに HEAD が追跡していないファイル・追跡していないものを含むディレクトリ・そのパスの上の場所にあるファイルがないかを確かめ、あれば merge も記録もせずに退かすパスを 10 件まで出して断る。理由(git merge は ignored のファイルを黙って上書きし、`--no-overwrite-ignore` は fast-forward にしか効かず、きれいさの確かめは ignored を見ない)、パスは `git merge-tree --write-tree` の木から取ること、2.38 より前の git の落とし先(directory rename の移った先を見ない)、大文字小文字を区別しない FS ではディスク上の綴りで判じること。D2: 読めない gate の記録の付け先を「打った org のその時点の結びつきのブランチ」から、その org がそれまでに完全な結びつきか読める記録を持ったすべてのブランチに変えた理由(regate が立て直した org の古い結びつきのブランチにも書く)と、付け先が 2 つ以上のときの理由の欄の文。この 2 つを固定するテストの名前 |
| 同(7c-1 の注) | 「読めない記録は、打った org のその時点の結びつきのブランチに付ける」の履歴は残し、「8a-3b で、その org が持ったすべてのブランチに付けるように変えた」を添えた(V2-2) |
| `.claude/skills/org/SKILL.md`(`regate` の行) | 作業ツリーの確かめ、identity の確かめのあとに、D3 の断りの 1 文を足した(merge が書くパスに ignored のファイルか追跡していないものを含むディレクトリがあれば、merge も記録もせずに退かすパスを 10 件まで出して終了コード 1)(V2-3) |
| `templates/base/.claude/skills/org/SKILL.md` | 同じ文(root と同一バイト。Edit で同じ置換をした) |
| `.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `scripts/sync-skills.sh` で再生成した(root は引数なし、template は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)。差分は上の 1 文だけで、4 面は同一バイト |
| `README.md`(`ralph org regate` の段落) | 「A conflict, a hook that stops the merge, or a missing git identity makes no commit and records nothing」の文に、ignored のファイルかそれを含むディレクトリが merge の書く場所にあるときも同じで、worktree の外へ退かすパスを 10 件まで出す、を足した(V2-3) |
| `docs/tech-debt/README.md`(7c-1 の残りの行) | f793924d で変わった 3 か所を今の規則と名前に直した(V2-4)。(s) の付け先の説明(「org の最新の読める記録のブランチ」を、org-regate PR からはその org が読める記録を持ったすべてのブランチに)、テスト名(`...GatedBranch` を `...GatedBranches` に)、(t) の `gatedBranch`(今は順序つきの集合 `orgBranches`。行は、unreadable な記録の `branch=` を足す変異と、org のほかのブランチが古い pass の `ready` に戻る変異を言う形にした) |
| 同(表の末尾に 7 行) | (a) 2.38 より前の git の落とし先が directory rename の移った先を見ない。(b) `branch.<name>.mergeOptions` を `git merge` は読み `git merge-tree` は読まない。(c) `isolateGitEnv` が `XDG_CONFIG_HOME` を触らない。(d) 大文字小文字の 6 case が CI の ubuntu で飛ぶ。(e) D3 が断りすぎる 2 つの場合(V2-5 の `readme.md` と `README.md`、入れ子の repository)。(f) 綴りだけ違うディレクトリでファイルごとに `git ls-files` を打つ(I6)。(g) merge が commit を作らなかったときの断りが、関係のない履歴でも hook を挙げる |
| `docs/insights/events/2026-10-16-org-regate.jsonl` | `sync_docs`、cycle 2、pass、findings 0 の 1 行を `insights-append.sh` で足した |

tech-debt の新しい行は、コードに当てて確かめてから書いた。(b) は git 2.49.0 の scratch repository で、`branch.feat.mergeOptions` を `-s ours` にすると `merge-tree` は tip の足したファイルを挙げ、`git merge` は何も書かないことを見た(regate は通していない)。逆向き(merge が書くのに merge-tree が挙げないパス)は見ていないので、行にそう書いた。(c) は同じ scratch で、`GIT_CONFIG_GLOBAL=/dev/null` と `GIT_CONFIG_NOSYSTEM=1` のもとでも `$XDG_CONFIG_HOME/git/ignore` が効き、`XDG_CONFIG_HOME` を空のディレクトリにすると効かなくなることを見た。(e) の 2 つ目(入れ子の repository)は、コードが打つ git のコマンドを scratch で打ち、gitlink を足す tip と ignored の入れ子の repository で、`ls-files --others --directory` が `vendor/x/` を untracked と出す(regate は断る)一方、素の `git merge --no-ff` は成功することを見た。regate の binary は通していないので、行にそう書いた。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `internal/cli/org_regate.go` の help | 合っている | 「it does when an ignored file or directory is on disk where the merge would write」の段落が D3 の断りを言い、退かすパスを出すことも言う。10 件の上限と merge-tree の道は書かないが、help の水準としてずれではない |
| `internal/cli/org_merge_order.go` の help | 合っている | D2 の付け先は help に書かれていない(cycle 1 も同じ)。付け先が 2 つ以上のときの文は、実行時の Detail(`classifyMergeTarget` の「may be a record of another branch of the same org, <org>, which has had N branches」)にある。`unreadable` の分類の行はコードと同じ |
| `ralph org spawn --help`、`ralph org start --help`(`internal/cli/org.go`) | 合っている | 「regate が走っている間は断られる」の段落が cycle 2 で入った。cycle 1 の「Left open」の 1 点目のうち help の側は解消した |
| `/org` skill の `spawn` と `start` の行 | 直した(この run のあと、下の Left open の 1 点目) | この run の時点では、regate が走っている間の断りは書かれていなかった。「機能ごとの org」節の「merge してよい順」(その org が走っている間の regate と、regate が走っている間の `spawn` と `start --plan` の断り)が言う。行に足すかは決めずに残す。INFO |
| `/org` skill の「merge してよい順」の手順 4 の「断り(worktree がない、作業ツリーが汚れている、HEAD が origin の先頭と違う、など)」 | 変更なし | 「など」で ignored のファイルの断りも覆われ、次の手は「stderr の次の手に従う」。4 面とも同じ |
| `/org` skill と README の merge-order の記述 | 変更なし | `unreadable` を「最新の gate の記録が読めない、またはそのあとに manifest の壊れた行がある」と言う。付け先の規則は skill にも README にもなく(D2 の前も同じ)、仕様と tech-debt が持つ |
| `/director` skill の「merge のあとの取り直し」(4 面) | 変更なし | 断りの個々の場合を並べず、「regate が出した文(衝突したファイル、`REASON:` と `LOG:`、push の失敗の理由、断りの理由と次の手)をそのまま人に伝える」と言う。ignored のファイルの断りもこれに入る。director は変更なし |
| `AGENTS.md`(`internal/org/` の行) | 合っている | `regate.go`・`regate_merge.go`・`regate_git.go` の役割分担と `regate.lock` の句。cycle 2 で足したファイルはない(変わったのは既存の `.go` とテスト) |
| `docs/quality/quality-gates.md` と template の写し | 変更なし | regate の文は断りの個々の場合を並べない |
| `docs/architecture/repo-map.md`、`docs/recipes/`、`docs/quality/definition-of-done.md` | 変更不要 | regate にも merge-order の付け先にも触れていない |
| `internal/org/prompts/`(`leader.md`、`reviewer.md`、`director.md`) | 変更不要 | 計画の Non-goals |
| `.claude/rules/ralph/` | 変更不要 | regate は座席にも message にも触れない |
| 絶対パスの `$HOME` | なし | 今回の差分に `/Users/` も `hiroki.yoshioka` もない。U+FFFD もない(変更した 8 ファイルを数えた) |
| 計画の Progress | 変更不要 | 計画は触らない依頼。cycle 2 の記録は 2 周目の self-review・verify・test まで入っている。`PR created` は /pr が付ける |

## Left open

- (この run のあとに orchestrator が直した)`/org` skill の `spawn` と `start` の行に、regate が走っている間の断りを 1 文ずつ足した(4 面)。あわせて、regate の行、「merge してよい順」の注、README、仕様の 8a-3b の注が「`spawn` と `start --plan`」と書いていたのを、コードと help に合わせて「`spawn`(`--dry-run` を除く)と `start`(どちらの形も)」にした(`start <task>` も Spawn の `checkCapacityAndStart` を通るため)。計画は `start --plan` と書くが、`start <task>` も spawn なので食い違いではなく、承認済みの本文は変えない
- test report の穴のうち、tech-debt に行がないもの: fetch と push の途中の中断(穴 1)、push・merge・`git var`・`reset --keep` の上限(穴 2)、`merge --abort` 自体の失敗(穴 3)、`regateUndoMerge` の読めない場合(穴 4)、D3 の確かめの error の道(穴 5)。計画の Progress のとおり PR の既知の穴に書く。穴 8(並べるパスの数の境目)は f630e216 のテストで埋まった(10 個と 11 個の行。`more >= 0` の mutant で 10 個の行が落ちることを確かめた)。穴 6 と穴 7 は、この run の tech-debt の (a) と (d) の行になった
- `internal/org/director_cmd_test.go` の sh の trap を 100 ms の sleep で待つテストが、負荷が高い go1.25.8 の全体の実行で 1 回落ちたこと(tech-debt に行がある)。この PR では直さない
- V2-5 の直し(同じ綴りのときも `git ls-files` を聞く)は、コードの変更なので入れていない。tech-debt の (e) の行にした

## Verification

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh`(root と template) | 14 skill と 13 skill を再生成。変わったのは `org` の 2 面だけ |
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/run-static-verify.sh` | 終了コード 0。scope は changed(golang)。tech-debt の plan の参照、gofmt、golangci-lint(`0 issues.`)、`secret-scan-branch: scanned 670ec219..f630e216 against origin/main: clean`。出力は scratch に保存(repo の外) |
| `/org` skill の 4 面 | 同一バイト |
| `docs/tech-debt/README.md` の新しい 7 行と 7c-1 の行 | 区切りの縦線はどの行も 6 つ(5 列)。参照する plan と report は実在する。行が挙げる関数とテストの名前はコードに実在する |
| この report を書く前の変更 8 ファイルの U+FFFD、差分の `/Users/` と `hiroki.yoshioka` | どちらも 0 件 |
| `git diff --check` | 出力なし、終了コード 0 |

`./scripts/verify.local.sh` は走らせていない。変えたのは文書と skill の写しだけで、`run-static-verify.sh` が同じ範囲の検査(skill の lock-step、template の一致、tech-debt の plan の参照、secret scan)を含んで通っている。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Go のテスト(コードを変えていないので走らせていない。verify と test の report が先行している)。tech-debt の (b) の逆向き(merge が書くのに merge-tree が挙げないパス)と (e) の 2 つ目(入れ子の repository)を regate の binary で打つこと、`-s ours` 以外の `mergeOptions` の効き方
