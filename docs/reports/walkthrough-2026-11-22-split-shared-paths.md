# Walkthrough: split-shared-paths

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-split-shared-paths.md
- Issue: #363
- Base: main bd079dbb

差分は 44 ファイル、3,899 行。そのうち約 1,000 行は報告、計画、insight で、Go は 21 ファイル(約 2,100 行、うちテストが約 1,700 行)。残りは雛形、skill の 4 写し、仕様、README、AGENTS.md、tech-debt。下の順に読むと早い。

## 1. 何が足りなかったか

分割計画の機能はそれぞれ `- Reserve:` でパスを予約し、並んで走る org の予約は重ならないように分ける。ところが lockfile や root の `tsconfig.json` のように、どの機能も変えざるをえないファイルを宣言する道がなかった。機能の org がそれを変えるたびに、見張りが「予約の外の変更」(scope_change)を知らせ、director が 1 件ずつ確かめて閉じていた。

## 2. 分割計画の `- Shared:` の行

- ヘッダ(最初の `## ` の見出しより前)に `- Shared: <パス>, <パス>` を 1 行まで書ける(`SplitPlan.Shared`、`internal/org/split.go:126`。読みは `parseSplitShared`、`split.go:536`)。パスは `- Reserve:` と同じ規則で読む
- 空の値、空の項目、`.`(repo 全体)、2 行目、どれかの機能の `- Reserve:` と重なるパスは読み込みで断る。ヘッダより後のフェンスの外で行頭に書いた `- Shared:` も断る(そこに書いても何も共有しないため)
- 機能ごとの digest(`featureDigest`、`split.go:747`)は、`- Shared:` の行があるときだけ共有のパスを入れる。行のない分割計画の digest はこの変更の前と同じ

## 3. 記録、受け付け、見張り

- `ralph org start --plan` は共有のパスを結びつきに入れ、予約の記録の Details の最後に `shared=<パス>,...` を書く(`FeatureBinding.Shared` と `SharedPaths`、`internal/org/reserve.go:126`。読みは `sharedFromToken`、`reserve.go:507`)。古い binary は知らない鍵から先を note と読むので、予約のパスとほかの結びつきは今までどおり読める。壊れた `shared=` は「共有のパスなし」と読む
- 受け付けは、共有のパスと別の org の予約のパスの重なりを両方向で断る(`reservationOverlapErr`、`reserve.go:743`)。共有のパスどうしの重なりは断らない
- 見張りの scope_change は、予約のパスと共有のパスを合わせた内を「内」と数える(`internal/org/watch_scope.go`)
- leader のタスクに `- 共有のパス:` の行が入り、座席の `--scope` は予約のパスと共有のパスを合わせたものになる。`ralph org reseat` が立て直す座席も同じ(`internal/org/feature.go`、`internal/org/reseat.go`)

## 4. ぶつかったときの直し方(skill)

並んで走る org が同じ共有のパスを変えると、あとから merge する側が既定のブランチを取り込むときにそこで衝突する。ralph は直さず、org skill の「共有のパスの直し方」に手順を書いた。生成されるファイル(lockfile、`go.sum`)は手で merge せず、元のファイルの衝突を先に直してから、既定のブランチの版に戻して作り直す。cross-review の指摘で、この機能が lockfile の中だけで上げた版を書き留めて作り直しのあとに確かめ、戻っていれば入れ直さずに `BLOCKED` で止まる手順を足した(入れ直し方はパッケージマネージャと依存の種類で違う。pnpm は間接の依存の版つき update を断る)。director skill の「共有のパス」は `- Shared:` の行の書き方を言う。

## 5. 先に読むテスト

- `TestLoadSplitPlan_Shared` と `TestLoadSplitPlan_SharedRejects`(`internal/org/split_shared_test.go:29`、`:144`)、`TestFeatureDigest_WithoutSharedIsUnchanged`(`:222`)
- `TestScopeReservedDetails_SharedTokenLast` と `…_OlderBinaryReadsSixTokens`(`internal/org/reserve_shared_test.go:76`、`:103`)、`TestReservationDecision_Shared`(`:225`)
- `TestStartFeature_SharedChangedRefused` と `TestStartFeature_SharedAdmission`(`internal/org/feature_shared_test.go:104`、`:146`)
- `TestWatch_SharedScope_ChangesToTheSharedPathsRaiseNothing`(`internal/org/watch_scope_shared_test.go:64`)
- `TestFeatureLeaderTask_SharedPaths`(`internal/org/feature_shared_task_test.go:32`)、`TestReseat_StartsTheSeatsWithTheLeadersTaskScope`(`internal/org/reseat_shared_test.go:101`)
- `internal/org/skill_shared_paths_test.go`: skill の 4 写しの文と並び

## 6. 確かめたこと

- `./scripts/run-test.sh` rc 0 が 2 回(1 周目 HEAD 77a6961a、2 周目 HEAD c17f1b9e。2 周目は #362 を取り込んだあと)。変異 11 個がすべて落ちた
- `./scripts/run-static-verify.sh` pass(2 回)。#366 を取り込んだあと(3fba16b4)は、org と cli の絞ったテストと static verify が ok。通しのテストは CI に任せる
- cross-review は 2 周。1 周目の ACTION_REQUIRED 1 件は直し、2 周目は指摘なし
- skill の手順 1 の pnpm の動き(`pnpm update` が `package.json` の範囲を書き換える、間接の依存の版つき update を断る、`--no-save` で上げた版が作り直しで戻る)は、self-review Run 2 が pnpm 12.9.1 で確かめた

## 7. 残したもの

- tech-debt の行: `ralph status` と `ralph org status` は共有のパスを出さない、`reserve.go` が 800 行を超えた、test の穴 2〜4 と self-review の LOW 6、ヘッダに `- Shared:` があるときの拒否の文の遠回りと `BLOCKED` で止まるときの rebase・merge の扱い
- npm と yarn の lockfile での手順 1 の動きは確かめていない
