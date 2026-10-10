# Self-review report: org-feature-worktree-ownership

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-org-feature-worktree-ownership.md(承認済み、digest feadcc05c21f)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: diff の品質だけ。`git diff be050681...HEAD`(4 コミット 91bc512f・33271a68・c8863608・c8fe902d、8 ファイル、+777/-144)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。ただし依頼された「skill の文言がコードのしていることと合っているか」は、文言の根拠をコードで確かめた

## Evidence reviewed

- `internal/org/feature.go` の差分と、`StartFeature`(160〜237 行)、`checkFeatureWorktreeReuse`、`checkEnsuredFeatureWorktree`、`featureSlugChange`、`featureWorktreeMismatch` の全文を読んだ。規則は `featureWorktreeMismatch` の 1 か所にあり、呼び出しは `checkFeatureWorktreeReuse`(410 行)と `checkEnsuredFeatureWorktree`(449 行)の 2 つだけ(`grep -rn featureWorktreeMismatch internal cmd`)。使い回しの検査の側は、規則の中身を持たず、エラー文の出し分けだけをしている
- `featureCanonicalRef` の呼び出しは `StartFeature` の 1 か所(207 行)。引数は `plan.Path` で、`ResolveSplitPlanPath`(split.go:171〜200)が `filepath.EvalSymlinks` で解決した絶対パスを返す。`LoadSplitPlan` が同じ値を持つことは、計画の前提どおり
- `scripts/ralph-worktree.sh` の `--canonical-ref` の扱い(230 行の `jq --arg`、272 行の引数読み、325 行)を読んだ。値は argv と `jq --arg` を通るので、空白や引用符を含むパスでも JSON は壊れない
- `internal/org/feature_test.go` の差分 627 行を全部読んだ。`fakeWorktrees.Ensure` が本物のスクリプトと同じく `canonical_ref` を見ずに既存の記録を返すこと、`ensureDrops` と `ensureHook` の使い方、`TestStartFeature_RealWorktreeScript_OtherLedger` が `worktreeScriptRepo`(`isolateGitEnv` を呼ぶ)の上で動くこと(`GIT_DIR` などを外し、`GIT_CONFIG_GLOBAL` を `/dev/null` にする)を確かめた。`featureWorktreeState` の `exec.Command("git", ...)` もその環境の中で動く
- `/org` skill の差分を、4 面が `cmp` で同一であること、手順の番号の繰り下げ(5 を足して 6 にした)に合わせた参照(「手順の 5」「6 のロック」「手順の 2」「手順の 3」)が残らず合っていること、拒否の一覧の文言がコードのエラー文と合っていることを確かめた
- `docs/tech-debt/README.md` の差分は同時の start の行 1 行。区切りの `|` の数は変更の前後とも 6 で、`\|` はない。取り消し線と RESOLVED の書き方は、ほかの閉じた行と同じ形
- 旧い形の参照の掃き出し: `grep -rn 'canonical_ref\|split:<' ` を docs/specs・docs/evidence・docs/recipes・README・`internal/cli`・`internal/org/prompts` に打ち、旧い `split:<id>#<slug>` の形を前提にした記述が残っていないことを確かめた(`ralph org start --help` は `canonical_ref` に触れない)
- 機械的な確認: `git diff --check` は空。追加行に U+FFFD、`fmt.Print` 系、TODO・FIXME、行末の空白はない
- 小さな probe: `filepath.EvalSymlinks` が大文字小文字を正規化しないことを、この macOS(APFS)で `Foo/a.md`・`foo/a.md`・`FOO/A.MD` を渡して確かめた(F-3)

## 依頼された点の結論

1. `featureWorktreeMismatch` の共有: 規則は 1 か所にある。2 つの呼び出しは、差分の出し方(使い回しの検査は `refuse`、`ensure` のあとの検査は専用の文)だけを分け、`canonical_ref`・パス・ブランチ・kind・チェックアウトの比べ方は同じ関数を通る。`checkEnsuredFeatureWorktree` は `Lookup` の失敗・記録なし・不一致を `what` にまとめ、`canonical_ref` だけが違う場合を別の文にする
2. 新しい `canonical_ref` と `plan.Path` の出どころ: 出どころは symlink を解決した絶対パスで、`--plan` の書き方(相対パス、symlink 経由)によらず同じ文字列になる。大文字小文字の違いには効かない(F-3)
3. エラー文: `cleanup` と `--org-id` を案内しない側(`ensure` のあとの拒否)は、テストが両方の語の不在を確かめている。ただし、案内の「slug を変える」は `--org-id` を明示した場合に効かない(F-1)
4. テストの隔離: 本物のスクリプトを使うテストは、一時 repo と `isolateGitEnv` の上で動く。ledger B のパスの空白(`ledger b`)も通す。隔離に問題は見つからなかった
5. skill とコード: 拒否の一覧、手順の番号、エラー文の案内は合っている。skill の `canonical_ref` の分岐も F-1 と同じ前提(slug が org_id の既定)に立つ

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| F-1 | MEDIUM | exception-handling | 「別の記録に当たったら、この機能の slug を変えて承認し直す」という案内が、`--org-id` を明示した start では効かない。記録の id は `org-<org_id>`(`featureWorktreeID`、134〜136 行)で、slug を変えて変わるのは既定の org_id とブランチだけ。`--org-id X` を渡している start は、slug を変えても org_id が X のままなので、同じ記録 `org-X` に当たり、同じ拒否が返る。`featureSlugChange(branch)` は branch しか受け取らず(470 行)、org_id が slug の既定かどうかを見ない。文は「(the slug is the default org_id and part of the branch ...)」と条件を括弧で言うが、拒否の本文の先頭に置いた案内としては、明示した org_id の側には直せない道を勧めている | `feature.go:466-472`(`featureSlugChange`)、呼び出しは 418 行(使い回しの拒否)と 458 行(`ensure` のあとの拒否)。同じ文が skill の `canonical_ref` の分岐(`.claude/skills/org/SKILL.md:439-446`)にある。記録の不一致を見る拒否のテストはどれも既定の org_id で、`--org-id` を渡した start が別の記録に当たるケースがない(`feature_test.go` で `OrgID` を入れるのは 311・366・513・515・646 行で、いずれも org_id の上限・不正値・結びつきの衝突を見るテスト)。別の `--org-id` を案内しないのは、ブランチ `<type>/<slug>` が変わらず先の org のブランチに当たるため(計画の Design decisions)。明示した org_id の場合は、`--org-id` を外すか変えることと slug を変えることの両方が要る | `featureSlugChange` に org_id と slug を渡し、`orgID != feature.Slug` のときは「`--org-id` を外すか別の値にし、ブランチも変えるなら slug も変える」を案内に足す(ブランチが先の org のものと同じだと `ensure` がブランチ衝突で止まり、そのときは `ensureFailureErr` の slug 先頭の案内に着く)。テストは「`--org-id` を明示して別の分割計画の記録に当たる」を 1 ケース足す。パイプラインは cycle 1 なので直しは 1 回の再実行で入る。直さないなら tech-debt の同時の start の行の隣に 1 行足し、skill の分岐に「`--org-id` を渡したときは org_id も変える」を書く |
| F-2 | LOW | maintainability | 「先の org が終わって worktree が消えるのを待つ」という案内に、消す主体がない。機能の worktree は `--cleanup-policy manual` で、disband と PR のあとも残り、消せるのは `ralph-worktree.sh cleanup` だけ(`feature.go:50-53` の定数のコメントが自分でそう言う)。待ったあとに打ち直すと、記録が残っているので使い回しの検査(`checkFeatureWorktreeReuse`)の `canonical_ref` が違う側の拒否に戻り、「要らなければ cleanup」の案内に着く。待つ道は、先の org の持ち主が cleanup するまで閉じない | `feature.go:456-458`(「wait until the other org (org_id %s) is done and its worktree is removed」)、skill の同じ段(466〜470 行)。このエラー文は `cleanup` の語を出さないことをテストが縛る(`TestStartFeature_RecordChangedDuringEnsureRefused` の `banned`)ので、書き方は「先の org の持ち主が merge のあとに後始末して worktree を消す」のように、コマンドを出さず主体を言う形になる | 文を「the other org's operator removes its worktree after the merge」のように、待つ相手と消える契機を言う形に直す。skill も同じ形にする。直さなくても、打ち直しの拒否が cleanup の案内に着くので行き止まりにはならない |
| F-3 | LOW | readability | `featureCanonicalRef` の doc が「`--plan` の書き方によらず同じ canonical_ref になる」と言う(144〜149 行)が、効くのは symlink と相対パスだけ。`filepath.EvalSymlinks` は大文字小文字を正規化しない(この macOS で `Foo/a.md`・`foo/a.md`・`FOO/A.MD` を渡すと、渡した綴りのまま返る)。大文字小文字を区別しない FS(APFS の既定)で、`--state-dir` の綴りが前回と違うと、同じ分割計画が別の `canonical_ref` になり、別の計画のものとして拒否される(拒否の側に倒れるので害は出ない) | `feature.go:144-152`、`split.go:171-200`(`dir` も `resolved` も渡された綴りのまま)。skill の「`--plan` をどう書いても同じ計画なら同じになり」(`.claude/skills/org/SKILL.md:434-437`)も同じ言い方 | doc と skill の「どう書いても」を「symlink と相対パスは解決するので、どう書いても」に直し、大文字小文字は別の綴りとして扱うと 1 句足す。コードは直さない(fail closed で、台帳の置き場所は repo から決まる) |
| F-4 | LOW | readability | `checkEnsuredFeatureWorktree` の分岐が、直前の `switch` が知っていることを組み立て直している。2 つ目の `switch` の `case err == nil && ok && rec.CanonicalRef != want.CanonicalRef` は、`Lookup` の契約(「記録がなければ ok は false、読めなければ error」、`FeatureWorktrees` の doc)から `err == nil` が冗長で、`ok` は「記録がない」を除くために要る。同じ関数の doc は 14 行あり、`what` を作る `switch` と文を選ぶ `switch` の 2 段に分かれている | `feature.go:439-462` | 1 つ目の `switch` の `default` で、`otherPlan := rec.CanonicalRef != want.CanonicalRef` を使って分けるか、`what` と一緒に「別の分割計画の記録か」の真偽を返す。冗長な `err == nil` を外す |
| F-5 | LOW | readability | 先頭の手順の説明と関数の doc に、実際と少しずれる言い方がある。(a) 41〜43 行の「one refused in 5 leaves the worktree to the start that made it」は、別の start が記録を作った場合の話だけ。`ensure` のあとの拒否は、記録が消えた・読めない・ほかの点が合わない場合にも出て、そのとき worktree はこの start 自身が `ensure` で作ったものでありうる(拒否のあとも残り、打ち直しは使い回す)。(b) `featureWorktreeMismatch` の doc は返り値を「"the worktree record <id>" で始まる文の続き」と言うが、`checkEnsuredFeatureWorktree` はそれを「its record <id> ...」の続きに使う。(c) コードのコメントとテストのコメントが `docs/plans/active/2026-10-10-org-feature-worktree-ownership.md` を指す(`feature.go:427` の 1 か所と、`feature_test.go` の 8 か所)。`/pr` が計画を archive へ動かすので、この参照は merge の時点で古くなる。同じ形の古い参照が既にある(`feature.go:16` は archive 済みの `active/2026-10-09-org-feature-worktree.md` を指す) | `feature.go:41-43`、`feature.go:475-481`、`feature.go:427`、`grep -c 'org-feature-worktree-ownership.md' internal/org/feature_test.go internal/org/feature.go`(8 と 1) | (a) は「leaves a worktree that another start made to that start」と、自分で作った場合を別に言う。(b) は「the rest of a sentence about the record」程度に緩める。(c) は計画を slug(`org-feature-worktree-ownership`)で指すか、既存の流儀に合わせて残す |

## Positive notes

- 規則の 1 か所化: `featureWorktreeMismatch` が比べ方を持ち、2 つの検査は文の出し分けだけを持つ。片方だけに足す直しが起きにくい形で、既存の `checkFeatureWorktreeReuse` の分岐がほぼそのまま移っている(分岐の順と文言は変わっていない)
- 隔離と固定: 本物のスクリプトのテストは `worktreeScriptRepo`(`isolateGitEnv` つき)の上で、2 つの台帳、空白を含む台帳のパス、先の org の ブランチ・worktree・記録が変わらないことの確認(`featureWorktreeState`)まで通す。`spyWorktrees` で「使い回しの拒否では `ensure` を呼ばない」を呼び出しの列で縛っている
- `fakeWorktrees.Ensure` が本物のスクリプトと同じく `canonical_ref` を見ないことを doc に書き、`ensureDrops` で「記録が消える」ケースを fake の側に足した。テストが本物の挙動からずれる穴を作っていない
- 差分は目的の範囲に収まっている。`internal/cli` と `internal/org` のほかのファイルには触れていない。skill の 4 面は `cmp` で同一で、tech-debt の行は取り消し線の範囲が実際に直した部分(別の分割計画の同時 start)だけ
- エラー文が先の org のブランチに触れない道を先に書く方針は、使い回しの拒否・`ensure` のあとの拒否・ブランチ衝突の 3 か所で揃っている(F-1 の例外を除く)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| F-1 を直さない場合: 「slug を変える」案内が `--org-id` を明示した start に効かない | 明示した org_id では、案内どおりにしても同じ拒否が返る。cleanup の案内が同じ文に残るので行き止まりにはならない | パイプラインの cycle 上限(2)と、直しが再実行を要すること | 直すなら今(cycle 1 の再実行)。残すなら、director(8a 段)が `--org-id` を明示して start を出す変更 | `docs/plans/active/2026-10-10-org-feature-worktree-ownership.md`、この報告の F-1 |
| F-2〜F-5(LOW): 案内の主体、doc の言い過ぎ、冗長な分岐、古くなる計画パス | 読み手の一手間。動作は変わらない | 全部 LOW | 次に `checkEnsuredFeatureWorktree`・`featureSlugChange`・`featureCanonicalRef` を触る変更 | この報告の F-2〜F-5 |

_(F-1 を直さず、F-2〜F-5 を残す場合に限り、上の 2 行を `docs/tech-debt/README.md` に 1 行にまとめて足す。今回は書き込んでいない。実装側が直すか決めてから、決まった形で足す。)_

## Recommendation

- Merge: 可(CRITICAL・HIGH なし。MEDIUM 1 件、LOW 4 件)
- Follow-ups: F-1 を直すか、tech-debt に 1 行で残すかを決める。直す場合は `featureSlugChange` と skill の `canonical_ref` の分岐と 1 ケースのテストで済み、再実行は cycle 2(最後の回)になる。F-2〜F-5 は直さなくてよい。直すなら F-1 と同じコミットに入れると再実行が 1 回で済む
