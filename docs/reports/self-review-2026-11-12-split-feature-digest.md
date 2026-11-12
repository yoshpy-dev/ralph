# Self-review report: split-feature-digest

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-split-feature-digest.md`(issue #312 の受け入れ条件 3)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff 9ac72c50...HEAD`(HEAD `172602cc`、7 コミット、16 ファイル)。`reserve.go`、`split.go`、`feature.go`、`cli/org.go` は diff 全部と周辺のコードを読んだ。テストの diff 5 本(`split_test.go`、`reserve_test.go`、`feature_test.go`、`spawn_feature_test.go`、`verbs_test.go`)と CLI のテスト 2 本も読んだ。org skill の 4 つの写しは `shasum` が同じことだけ見て、本文は 1 つ読んだ。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: `L1` から `L5`(LOW)

## Evidence reviewed

lead に頼まれた「start --plan を黙って通す穴」を、次の 6 つの道で調べた。コードを読んで追ったもので、実行はしていない。

1. 座席に渡る中身と digest の入力が一対一か。`featureLeaderTask`(`feature.go:486-516`)が分割計画から取るのは、plan.ID、plan.Path、plan.Digest、f.Slug、ブランチ(Type と slug から作る)、f.Reserve、f.DependsOn、承認方式、台帳、reviewer の値、`plan.Preamble`、`f.Body`。`featureDigest`(`split.go:654`)の入力は preamble、slug、Type、Reserve、DependsOn、Body。残りのうち、id・ブランチ・worktree・承認方式は binding の欄として `sameFeature` が比べる。path は id と台帳から決まる。台帳と reviewer の値は、全体の digest でも比べていなかった。落ちている欄は無い。
2. 機能の節に、digest にも Body にも入らない行が無いか。欄の行の鍵は `Type`・`Reserve`・`Depends on` の 3 つだけ(`split.go:94`)で、ほかの行は `bodyLines` に入る(`split.go:539-545`)。節の外で捨てられるのは、ヘッダ、`## Features` の最初の `### ` より前、`## Features` の後の節。いずれも座席に渡らない。
3. 全体の digest への戻り(片方が機能ごとの digest を持たない組)が本当に「より緩くない」か。`PlanDigest` が飛ばす行は `- Status:`・`- Approved:`・`- Branch:`、`## Progress checklist` の節、`- [x]` の行。`parseSplitPlan` は `- Branch:`・見出し・チェック済みの箱を fence の中でも拒否し(`rejectDigestSkippedLine`、`split.go:458`)、`- Status:`・`- Approved:` はヘッダの 1 行ずつしか許さない(`split.go:358`、`473-487`)。ヘッダは前文に入らないので、全体の digest が同じなら前文も機能の節も同じになる。`sameFeature` の注釈の主張は成り立つ。
4. binding の全体の digest を、今の分割計画と比べる読み手が無いか。非テストコードで `.Digest` を読む所を全部 grep した。binding の欄を読むのは `reserve.go`、`feature.go`(書く側)、`cli/org.go`(status の JSON)だけ。`approval.go` と `approve.go` の `Digest` は機能の計画の `PlanRef` で、別物。`gate.go:688` と `regate.go:978` は、台帳から読んだ 2 つの binding を比べるだけで、今の分割計画は見ない。
5. 比較の分岐。`sameFeature`(`reserve.go:176-188`)は、`Complete`、split、feature、branch、worktree、承認方式を先に比べ、そのあとで digest を選ぶ。承認方式が違えば機能ごとの digest が同じでも偽になる。片方だけが持つ組と両方が持たない組は全体の digest で比べる。`a` と `b` の両方に `Complete` を求めるので、前の `*a == *b` と同じく対称。
6. token の読み。`featureBindingFromTokens`(`reserve.go:398-430`)は、`feature_digest=` が 2 つ、値が空のとき欄を空にする(全体の digest の比較に戻る、断る側)。メモの後ろに書いた token は読まない。`validateFeatureBinding` は 12 桁の小文字の 16 進を必須にするので、start が入れ忘れた binding は Spawn が断る。

`record=false` で通るとき、`reservationDecision` は重なりの検査を飛ばす(`reserve.go:605`)。ただし通る条件に `slices.Equal(held.Paths, want.Paths)` があり、機能の Reserve は digest にも入っているので、この機能の予約は変わらない。ほかの機能の Reserve が変わって重なった場合は、その機能自身の start が記録の道を通るので、そこで断られる。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | Maintainability | `featureDigestRefusal` が、`sameFeature` と同じ 5 欄(split・feature・branch・worktree・承認方式)の比較を書き写している。`sameFeature` に欄が増えても、こちらは気づかずに古い条件のまま「節か前文が変わった」と言うか、何も言わなくなる。判定には効かず、拒否の文だけが外れる | `reserve.go:176-182` と `reserve.go:653-655` の条件が同じ並び | 「digest 以外の欄が同じか」を返す小さい関数(例 `sameFeatureExceptDigests`)を 1 つ作り、両方から呼ぶ |
| L2 | LOW | Naming | 機能ごとの digest の形の検査に `approvedDigestPattern` を使っている。この変数は `- Approved:` の行の digest の形として定義されていて(`split.go:73-75`)、機能ごとの digest の形とたまたま同じ。片方の長さを変えると、もう片方の検査が黙って変わる。テストは `split_test.go:791` に同じ正規表現の `featureDigestPattern` を持つが、本番側には名前が無い。本番側に同じ名前を足すと、同じ package のテストの宣言と衝突する | `reserve.go:222`、`split.go:666`(`[:12]` で切る)、`split_test.go:791` | `featureDigestPattern` を `split.go` の `featureDigest` の隣に置き、`validateFeatureBinding` からそれを使う。テストの宣言は消して本番の変数を使う |
| L3 | LOW | Naming | ローカル変数 `featureDigest` が、同じ package の関数 `featureDigest`(`split.go:654`)を隠している。`FeatureBinding.String()` と、テストの 3 か所(`feature_test.go:301`、同 `1972` の引数、`reserve_test.go:729` の引数)。コンパイルは通るが、`featureDigest` の grep が 2 つの意味に当たる | `reserve.go:147-152` | `String()` の変数を `featureDigestText` などに変え、テスト側も `fd` などにする |
| L4 | LOW | Error text | 機能ごとの digest が無い記録への拒否の文が、一律に「(an older ralph started it)」と言う。`featureBindingFromTokens` は、`feature_digest=` が 2 つか値が空の壊れた記録も同じく空にする(その旨は同じ関数の注釈にある)。その記録には「古い ralph が立てた」が当たらない。status の help と skill の「この版より前の ralph が立てた org では出ない」も同じ言い方 | `reserve.go:661-663` と `reserve.go:392-397` | 文を「no feature digest in its record (an older ralph started it, or the token is damaged)」のように両方を言う形にする。help と skill は今のままでも誤りではないので、直さなくてよい |
| L5 | LOW | Readability | `sameFeature` の注釈(`reserve.go:159-175`)と `reservationDecision` の注釈(`reserve.go:576-598`)が、それぞれ 1 つの長い文になっている。括弧が 3 重で、「全体の digest への戻りが緩くない理由」の論証が文の途中に入っている。次に直す人が条件を読み取りにくい | `reserve.go:159-175`、`576-598` | 規則(両方が持つとき、どちらかが持たないとき、不完全なとき)を 3 文に分け、「緩くない理由」は別の段落にする |

## Positive notes

- 機能ごとの digest の符号化(`<名前> <長さ>\n<値>\n`)は、欄の境目をまたぐ文字の移動で衝突しない。Reserve と DependsOn は、パスと slug にコンマが入らない規則(`NormalizeReservePaths`、`ValidateIdentifier`)があるので、コンマでつないでも曖昧にならない。`TestFeatureDigest_Encoding` が入力のバイト列を固定している。
- digest を、生の行ではなく parse の結果から作っている。最後の機能の後ろに機能を足すだけで digest が変わる問題を避けていて、`TestFeatureDigest_ChangesOnlyWithTheFeatureAndPreamble` の行がそれを確かめている(変わる 9 行、変わらない 13 行、どの行も「編集が何も変えない」場合は `t.Fatal` で落ちる)。
- `Complete()` を 5 欄のままにして、token の無い古い記録の org が gate・merge-order・status から消えないようにしている(D4)。`TestFeatureBinding_OlderRecordStillBound` が `ActiveFeature`、`gateOrgLife`、`collectMergeTargets` の 3 つで確かめている。
- token を `approval=` の後ろに置き、古い binary が 5 欄を今までどおり読めることをテストで確かめている(`TestScopeReservedDetails_OlderBinaryReadsFiveTokens`)。
- 拒否の文が、何が変わったか(節か前文か、記録に digest が無いか)と、前と今の digest を言う。拒否の理由を読んだ人が次の手(disband か別の `--org-id`)を選べる。

## Coverage gaps

- テストは走らせていない(`/test` の範囲)。diff を読んだだけで、`go vet` も `gofmt` も見ていない。
- `featureLeaderTask` に欄が増えたとき `featureDigest` にも足す、という約束は、両方の注釈にある。欄の増加を機械で捕まえる検査は無い。tech-debt の行にするか、`/sync-docs` で判断する(今回の指摘ではない)。
- tech-debt は、L1 から L5 のどれも足していない。5 件とも LOW で、このブランチで直せる。

## Recommendation

- Merge: OK(CRITICAL・HIGH・MEDIUM なし、LOW 5)。start --plan が、機能自身の節、前文、承認方式の変更を黙って通す道は見つからなかった。
- Follow-ups: L1 から L5 は、このブランチに足す次のコミットで直せるが、パイプラインを回し直す理由にはならない。直すなら cross-review の前に 1 コミットにまとめる。
