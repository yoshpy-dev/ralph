# Self-review report: org-feature-worktree-ownership

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-org-feature-worktree-ownership.md(承認済み、digest feadcc05c21f)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)。初回は 7bd95ebd の時点、再実行は 52800bae・fc50ad10 のあと(cross-review の前の直しなので cycle は増えない)。現況は末尾の「再実行」の節と Recommendation
- Scope: diff の品質だけ。初回は `git diff be050681...HEAD`(4 コミット 91bc512f・33271a68・c8863608・c8fe902d、8 ファイル、+777/-144)、再実行は `git diff 7bd95ebd HEAD`(2 コミット、7 ファイル、+345/-92)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。ただし依頼された「skill の文言がコードのしていることと合っているか」は、文言の根拠をコードで確かめた
- 番号の付け方: 初回の finding は F-1〜F-5(付け替えない)、再実行で足した finding は N1〜N5

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

## Findings(初回、7bd95ebd の時点。現況は「再実行」の節)

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
- エラー文が先の org のブランチに触れない道を先に書く方針は、使い回しの拒否・`ensure` のあとの拒否・ブランチ衝突の 3 か所で揃っている(初回の時点では F-1 の例外があり、再実行で直った)。再実行では、`--org-id` を渡した start の枝まで、コード・skill・fake のテスト・本物のスクリプトのテスト(`TestStartFeature_RealWorktreeScript_OrgIDOtherPlan`、別の `--org-id` だけではブランチ衝突で止まり、別の `--org-id` と slug の変更で通ることまで)が同じ 3 分岐で揃っている

## 再実行(52800bae・fc50ad10 のあと)

読んだもの: `git diff 7bd95ebd HEAD` の全部(`feature.go`、`feature_test.go` 309 行、skill 4 面の差分、計画の進捗の 1 行)。skill の 4 面は `cmp` で同一。`scripts/ralph-worktree.sh` の `ensure_worktree`(284〜304 行)と `update_last_seen`(251〜259 行)を読み、既存の記録を返すときに変わるのは `last_seen_at` だけであることを確かめた。`scripts/archive-plan.sh` を読み、archive が書き換えるのは `docs/tech-debt/README.md` の `docs/plans/active/<name>` の参照だけで、コードとテストのコメントは書き換えないことを確かめた。

### 初回の指摘の現況

| ID | 現況 | 根拠 |
| --- | --- | --- |
| F-1 (MEDIUM) | 直った | `otherRecordFix(orgID, slug, rec, want)`(`feature.go:481`)が、`orgID == slug` なら slug の変更、そうでなければ「別の `--org-id` か、`--org-id` を外す」を案内し、記録がこの機能のブランチにもあるとき(`rec.Branch == want.Branch`)に slug の変更を足す。2 つの呼び出し(`feature.go:421` の使い回しの拒否と、`feature.go:464` の `ensure` のあとの拒否)がどちらも通る。`ensure` のあとの拒否では、`ensure` が返した記録はパス・ブランチ・種類が同じなので、常に slug の変更が付く(テストの「made during ensure」と本物のスクリプトのテストがその形)。skill の 4 面も `--org-id` を渡した場合の枝を足し、コードの 3 分岐(slug のとき、`--org-id` を渡したとき、記録が同じブランチにもあるとき)と合っている。残りは N3 |
| F-2 (LOW) | 直った | 待つ相手と契機が「the owner of the other org (org_id %s) removes that worktree once the org's PR is merged」になり、`cleanup` の語は出さない(`feature.go:461-463`、テストが `cleanup` の不在を確かめる)。skill も同じ形(`SKILL.md` の `ensure` のあとの拒否の段)。待ったあとの打ち直しが使い回しの拒否に戻る点は、skill の書き方で見えるようになった |
| F-3 (LOW) | 残る(tech-debt へ送る、とのこと) | `featureCanonicalRef` の doc(`feature.go:144-149`)と skill の「どう書いても同じ計画なら同じ」は直っていない。`docs/tech-debt/README.md` は `7bd95ebd..HEAD` の差分にないので、行はまだない。計画の進捗の記録(fc50ad10)は「F-3 は tech-debt に送る」と書く。この PR の中で行が入ることを、`/sync-docs` か `/pr` の前に確かめてほしい |
| F-4 (LOW) | 直った | 1 つ目の `switch` の `default` で `otherPlan = rec.CanonicalRef != want.CanonicalRef` を作り、2 つ目の `switch` は `case otherPlan:` になった(`feature.go:447-460`)。冗長な `err == nil && ok` は消えた |
| F-5 (LOW) | 一部直った | (a) 先頭のコメントが「別の start の記録のときはその start に、ほかのときは(この start が作ったかもしれない)次の start に」と、実際の 2 通りを言うようになった(`feature.go:41-45`)。(b) `featureWorktreeMismatch` の doc が両方の文頭を挙げるようになった。(c) `feature.go` の 2 か所(16 行と 430 行)とテストのコメントを `archive/` の参照に直した。ただし同じ stage-4 計画の `active/` 参照が 6 か所残り、ほかの変更と揃っていない(N4) |

### 再実行で足した指摘

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | maintainability | tech-debt の同時の start の行(RESOLVED の注)が、F-1・F-2 の直しで言えなくなったことを言っている。注は「エラーは slug を変えて承認し直すか、先の org が終わるのを待つことを案内し、`cleanup` も別の `--org-id` も案内しない」と言うが、`--org-id` を渡した start では別の `--org-id` を案内し、待つ相手は「先の org の持ち主が、PR の merge のあとに worktree を消す」になった。行は `7bd95ebd..HEAD` で触られていない | `docs/tech-debt/README.md:192` の「at neither the cleanup nor another `--org-id`, since the branch `<type>/<slug>` stays the same whatever the org_id」と「until the other org is done」。コード側は `feature.go:421-466` | 行の該当の 1 文を新しい案内に合わせる。F-3 の行を足すときに一緒に直すと 1 回で済む |
| N2 | LOW | readability | テストのコメント 2 か所が、直しのあとも既定の org_id のときの話だけを言う。`TestStartFeature_WorktreeRecordMismatchRefused` の doc(`feature_test.go:665-670`)は、`canonical_ref` が違う記録の拒否を「別の slug を先に案内し、別の `--org-id` は案内しない」と言い切るが、`--org-id` を渡した start では逆になった(その場合は `TestStartFeature_OtherRecordWithOrgID`)。`TestStartFeature_OtherRecordWithOrgID` の doc の末尾の行は、次の関数名で始まる行が段落の続きに付いている(`// TestStartFeature_RealWorktreeScript_OrgIDOtherPlan follows the fix.`)ので、doc の書き出しに見える | `feature_test.go:665-672`、`feature_test.go:1009-1018` | 前者に「(既定の org_id のとき。`--org-id` を渡したときは `TestStartFeature_OtherRecordWithOrgID`)」を足し、後者は空の `//` 行を挟むか、文として「`TestStartFeature_RealWorktreeScript_OrgIDOtherPlan` follows it through the real script.」に直す |
| N3 | LOW | exception-handling | `otherRecordFix` は `orgID == slug` で既定の org_id かを決める。`--org-id` に slug と同じ値を明示した start もこの枝に入り、「slug を変えて承認し直す」と案内されるが、明示した org_id は slug を変えても変わらず、同じ拒否が返る(次の拒否は `--org-id` を渡した枝の案内に着くので、2 回で収束する)。F-1 の小さな再発。skill の枝の見出しは「org_id が slug のとき(`--org-id` を渡さない既定)」で、コードの条件(値が同じ)より狭く書いてある | `feature.go:481-483`(`if orgID == slug`)。`StartFeatureParams.OrgID` は `""` が既定で(`feature.go:83`)、`readStartFeature` が slug を入れる前の値は `p.OrgID` で分かる | 実際に起きにくい使い方(slug と同じ値を `--org-id` に渡す)なので直さなくてよい。直すなら `checkFeatureWorktreeReuse` と `checkEnsuredFeatureWorktree` に、slug の代わりに「`--org-id` が空か」を渡す。直さないなら skill の見出しを「org_id が slug と同じとき」に合わせる |
| N4 | LOW | unnecessary-change | F-5 (c) の直しが 2 か所だけで、同じ参照を持つほかのコメントと食い違った。`feature.go:16` と `feature_test.go` の先頭は `archive/2026-10-09-org-feature-worktree.md` になったが、同じファイルを指す `active/` の参照が `split.go:17`、`reserve.go:27`、`internal/cli/org.go:530`、`internal/cli/org_feature_test.go:19`、`prompts_test.go:419`、`spawn_feature_test.go:11` に残る。`archive/2026-10-10-org-feature-worktree-ownership.md`(新しい計画)は、`/pr` が計画を動かすまで存在しない | `grep -rn 'plans/active/2026-10-09-org-feature-worktree' internal`(6 件)。`scripts/archive-plan.sh` が書き換えるのは tech-debt の参照だけ | 範囲を絞るなら `feature.go:16` とテスト先頭の編集を戻し、新しい計画の参照だけ `archive/` にする。揃えるなら 6 か所も同じコミットで直す(コメントだけの変更)。どちらでも動作は変わらない |
| N5 | LOW | readability | `checkFeatureWorktreeReuse` と `checkEnsuredFeatureWorktree` が `orgID, slug string` を並べて受け取る(root、orgID、slug の 3 つの string が続く)。呼び出しは `feature.Slug` を末尾の位置に渡していて合っているが、取り違えてもコンパイルが通る | `feature.go:397`、`feature.go:443`、呼び出しは 211 行と 227 行 | N3 を直すときに、2 つの string を「org_id が既定か」の bool か、`otherRecordFix` に渡す小さな構造体に置き換えると取り違えがなくなる。直さなくてよい |

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| F-3: 大文字小文字を区別しない FS で、綴りの違う `--state-dir` は別の `canonical_ref` になる。`featureCanonicalRef` の doc と skill が「どう書いても同じ」と言う | 拒否の側に倒れる(別の計画のものとして拒否され、slug の変更か cleanup の案内に着く)。害はない | LOW。コードは直さず、doc の言い方だけの問題 | 次に `featureCanonicalRef` か skill の該当の段を触る変更 | この報告の F-3。行は `7bd95ebd..HEAD` にまだない(計画の進捗は「tech-debt に送る」) |
| N1〜N5(LOW): 同時の start の行の注が古い、テストのコメントの言い切り、`--org-id` に slug と同じ値を渡した枝、`active/` 参照の食い違い、string の並び | 読み手の一手間。動作は変わらない | 全部 LOW | N1 は F-3 の行を足すとき。N2〜N5 は次に `otherRecordFix` かその呼び出しを触る変更 | この報告の N1〜N5 |

_(`docs/tech-debt/README.md` には、この報告では書き込んでいない。F-3 の行は実装側が足す。足すときは N1 の文もそこで直す。)_

## Recommendation

- Merge: 可(CRITICAL・HIGH なし。MEDIUM 0 件。LOW は F-3 が残り、新しく N1〜N5 の 5 件)。初回の MEDIUM(F-1)は直った
- Follow-ups: (1) F-3 の tech-debt の行を、この PR の中で `/pr` の前に足す。そのとき N1(同時の start の行の注)も新しい案内に直す。(2) N4 は、範囲を絞って `feature.go:16` とテスト先頭の編集を戻すか、6 か所を揃えるかを決める。(3) N2・N3・N5 は直さなくてよい。直すならコメントだけの N2 が安い。N1・N4 はコメントと tech-debt の変更だけだが、`.go` のコメントを触れば diff は変わるので、触った場合は `/self-review` から回し直す(cross-review の前なので cycle は増えない)。直さずに `/verify` へ進んでも、動作に影響する問題はない
