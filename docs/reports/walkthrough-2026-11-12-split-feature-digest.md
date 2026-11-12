# Walkthrough: 分割計画の digest を機能ごとにも持ち、ほかの機能の節の変更で start を断らない(split-feature-digest)

- 計画: `docs/plans/archive/2026-11-12-split-feature-digest.md`(PR の作成時に archive へ移す)
- issue: #312 の受け入れ条件 3。1(`usage_limited`)は #312a、2(座席の立て直し)は #312c で満たす。この PR では #312 を閉じない
- 差分: `git diff d4bafaa7...HEAD`(25 ファイル、約 +1,800 / −220。テストを除く Go は約 +250 / −90、Go のテストは約 +950 / −110、残りは記録と文書)

テストと記録が差分の大半を占める。次の順に読むと、仕組みを追いやすい。

## 1. 機能ごとの digest を作る(split.go)

- `featureDigest` は、前文、slug、Type(省略時の `feat` を入れたもの)、Reserve(正規化したもの)、Depends on、本文を SHA-256 にかけ、先頭 12 桁を返す。欄ごとに名前と長さを付け、決まった順に並べる。これは `featureLeaderTask` が分割計画から座席に渡す欄と同じで、両方の注釈がもう一方を指している。
- `finishSplitPlan` は、Type の既定値を入れて本文を整えたあとに `SplitFeature.Digest` を計算する。パースの挙動、全体の digest(`PlanDigest`)、承認の確かめ(`CheckApproved`)は変えていない。
- テストは `split_test.go` の `TestFeatureDigest_*` 3 本。
  - 22 行の表で、ほかの機能の節、ヘッダ、`## Features` の前置き、後ろの節、空行、CRLF を変えても digest が変わらないことを確かめる。前文、本文、Type、Reserve、Depends on(並びを含む)を変えると変わることも確かめる。
  - ハッシュにかけるバイト列を固定したテストもある。台帳に残る値なので、作り方を変えると、それより前に立てた org の次の start がすべて断られるためである。

## 2. 記録と比べ方(reserve.go、feature.go)

- `FeatureBinding.FeatureDigest` を足した。`scope_reserved` の Details には、`approval=` の後ろの最後の token `feature_digest=<12 桁>` として書く。古い binary は知らない鍵で読むのをやめるので、ほかの 5 欄と承認方式を今までどおり読める。
- `featureBindingFromTokens` は、token が 2 つあるときや値が空のときに欄を空として読む。`Complete()` は 5 欄のまま変えていない(D4)。token の無い古い記録も完全な結びつきとして読むので、gate、merge-order、status の扱いは変わらない。
- `sameFeature` は、構造体の `==` をやめて欄ごとに比べるようにした。
  - 先に split、feature、branch、worktree、承認方式を比べる。
  - 両方が機能ごとの digest を持つときは、その digest を比べる。どちらかが持たないときは、全体の digest で比べる。
  - `a` と `b` の両方に Complete を求め、比較を対称にした。
  - 全体の digest が同じなら機能ごとの digest も同じになる(分割計画は、全体の digest が飛ばす行を、ヘッダの 2 行を除いて拒否する)。このため、全体の digest に戻る道は断る側にしか倒れない。
- `validateFeatureBinding` は、機能ごとの digest が 12 桁の小文字の 16 進数であることを求める。`startFeatureLeaderParams` は `f.Digest` を入れる。
- `featureDigestRefusal` は、同じ機能で digest だけが違う拒否に 1 句を足す。機能の節か前文が変わったときは、前と今の digest を言う。古い記録のときは、どの変更でも断ること、disband して立て直せば記録されることを言う。
- 通るときは、今の「同じ start の打ち直し」と同じく何も記録しない(D2)。台帳の全体の digest は最初の start のものが残る。新しい記録を足さないので、merge-order と regate が使う記録の位置も動かない。
- テスト
  - `reserve_test.go`: token の往復、古い binary の読み方、`sameFeature` の表、検査、拒否の文、D4 の回帰(`TestFeatureBinding_OlderRecordStillBound`)。
  - `feature_test.go`: `StartFeature` での立て直しを確かめる。ほかの機能の節の変更では通り、依存先の節の変更でも通る(D7)。承認の記録はそのまま効く。自分の節か前文の変更と、古い記録では断る。
  - 期待値だけを直したもの: `spawn_feature_test.go`(承認し直した binding の機能ごとの digest も変え、断る確かめを保つ)、`verbs_test.go`(補償は token ごと書き戻す)、`internal/cli` の 2 本(`approval=` を最後の token と決め打ちしていた)。

## 3. CLI と文書

- `internal/cli/org.go`: `ralph org status --json` の `feature` に `feature_digest` を足した(古い記録では出さない)。start の help には立て直しの規則を、承認方式も同じことを含めて書いた。status の help にもキーを足した。`TestOrgStartAndStatusHelp_FeatureDigest` が文を固定する。
- org skill の 4 つの写し: 予約の記録の欄、分割計画の節、承認の節、start の手順 6、打ち直しの段落、拒否の箇条、status の行を直した。機能をまたぐ約束は前文に書くこと(D7)も足した。
- `docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 に決定の注を足した。`internal/org/seat.go` の `EventScopeReserved` の注釈は token を全部並べるようにした。`docs/tech-debt/README.md` には、満たした trigger の持ち越しと、この PR の既知の穴を 3 行で足した。

## 4. 記録

- 計画、self-review、verify、test、sync-docs、cross-review triage の報告と insight event。
- branch は 0f4c64e4 で origin/main(#313、PR #316)を取り込んだ。衝突はなかった。取り込んだあとに、`run-static-verify.sh` と、`internal/cli`・`internal/upgrade`・`internal/org` の `go test -count=1` を打ち直し、すべて通った。shell を含む `run-test.sh` 全体は CI で回る。
