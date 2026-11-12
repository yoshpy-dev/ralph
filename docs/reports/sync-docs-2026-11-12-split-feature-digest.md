# sync-docs report: split-feature-digest

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-split-feature-digest.md`(承認の digest `5a95d18e25ff`)。issue #312 の受け入れ条件 3
- Branch: `fix/split-feature-digest`。base は `9ac72c50`、origin/main `d4bafaa7`(#313 の merge)を `0f4c64e4` で取り込み済み。run 開始時の HEAD は `0f4c64e4`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `890ac5c2`(Merge: OK、LOW 5 件 L1〜L5)、verify `e0105040`(pass。LOW 4 件 V1〜V4、INFO 1 件 I1)、test `7925461b`(pass。Test gaps 1〜5)
- 見た範囲: この task の変更 `git diff 9ac72c50..34d74a8d` と計画。#313 の変更(merge で入った)は見ていない
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `5a95d18e25ff`。コードの動きは変えていない(Go の変更は comment 1 か所)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit の前の数): `git diff origin/main...HEAD` は 20 ファイル、+1,611/-211(計画、report、insights を含む)。docs 以外は +1,205/-211、そのうちテストを除くと +345/-117(org skill の 4 つの写しを含む)。walkthrough は書いていない

## Summary

verify の V1〜V4 を直した。V1 は spec の FR-4 に 2026-11-12 の決定の注記を足した。V2 は tech-debt の Trigger を引いた 6 行に note を足し、掃引でさらに 2 行(425、427)が当たったので同じ形で足した。V3 は 220 行目の項目の文に `feature_digest` を足した。V4 は `internal/org/seat.go` の `EventScopeReserved` の comment を直した。I1(`AGENTS.md`)は触らなかった。新しい tech-debt の行を 3 つ足した(残した穴 2 つ、テストの穴、self-review の L1〜L5)。ほかの文書(README、quality-gates、repo-map、skill、help)は実装と合っていた。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-4 の注に 2026-11-12 の決定の注記を 1 項目足した(95 行目、FR-5 の前)。FR-4 の本文と受け入れ条件は変えていない。下の「V1」 |
| `docs/tech-debt/README.md` | 既存の 9 行を変えた(180、192、200、201、220、223、396、425、427)。変えたのは Trigger の末尾への note と Related への計画の path、220 行目の項目の文への括弧 1 つだけで、挿入だけの編集。新しい 3 行を末尾に足した(468〜470 行)。下の「tech-debt」 |
| `internal/org/seat.go` | `EventScopeReserved` の comment が書く binding の token を、`approval=` と `feature_digest=` まで書く形に直した(comment だけ)。下の「V4」 |
| `docs/plans/active/2026-11-12-split-feature-digest.md` | `## Progress checklist` の artifact の 3 つの箱を `[x]` に、後段(self-review、verify、test の commit と結果)と sync-docs の 2 行を「Implementation started」の下に足した。「PR created」と AC の箱は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-12-split-feature-digest.md` | この report |
| `docs/insights/events/2026-11-12-split-feature-digest.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 |

commit は 1 つ(`docs: sync docs for split-feature-digest`)。

## verify が挙げた drift

### V1: FR-4 の決定の注記

`docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 の最後の注(2026-10-30 の leader の sync-docs)の次に、「2026-11-12 に決めたこと(issue #312 の受け入れ条件 3)」を足した。ほかの FR の 2026-11-12 の注(#307)と同じ形(`  - <日付> ...で決めたこと: ...。詳細は計画(...)に書いた`)で、内容は次のとおり。

- 起きたこと(ほかの機能の節の直しで全体の digest が変わり、止めただけの org を立て直せなかった)
- 機能ごとの digest の材料(前文と、その機能の slug・Type・Reserve・Depends on・本文)と、入らないもの
- start が `feature_digest=` を `approval=` の後ろに書くこと、`sameFeature` が両方に token があるときだけそれで比べること
- 立て直せる条件、台帳に何も足さないので全体の digest は最初のまま残ること、`plan_approved` が効いたままになること
- 承認の digest と `CheckApproved` は変えていないので、FR-4 の「分割計画の digest が承認時の値と一致しなければ start を拒否する」はそのままであること
- 前文かその機能の節を直した start と、token の無い古い記録の org は、今までどおり断ること
- 残る穴 2 つ(依存先の節にだけ書いた約束、古い binary の補償)

FR-4 の `[ ]` の印と「印(...)」の項目は変えていない(FR-4 は RC2 のために `[ ]` のままで、この注は印を動かさない)。立て直しを実機の herdr で回していないことは、469 行目の tech-debt の行 (c) に書いた。

`/natural-japanese` の lint(`/private/tmp` に書いた下書きへの hook)は「和欧文間の半角空白」を 1 件指した。この spec は全体が `全体の digest が` の形で書かれていて(同じ日の #307 の注も同じ)、1 つだけ詰めると不揃いになるので、そのまま残した。

### V2: tech-debt の Trigger(6 行 + 掃引で 2 行)

verify が挙げた 6 行は、行を開いて Trigger を読んだ。掃引は後の節に書く。note は Trigger セルの末尾に足し(`. 2026-11-12: fix/split-feature-digest hit ...`)、Related の末尾に `docs/plans/active/2026-11-12-split-feature-digest.md` を足した。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の表の `status` の行(4 つの写し) | 623 → 689 字(`git show` から数えた)。`stop` と `disband` のセル(1,208 と 1,219 字)は変わらず、`status` の行はそのセルより短い。表の作り直しは計画の Scope の外。持ち越し |
| 192「Test gaps left by the org-limits-reserve cycle 2 fix」(c) | `ralph org status` の `Long` | **一部を払った**。新しい文は `TestOrgStartAndStatusHelp_FeatureDigest` が cobra の出力から確かめる。`reserved:` の行、`feature:` の行、不完全な記録の注と `--config` の説明は読んでいない(`ralph org status --help` を読むテストは今もこれだけ。`status_test.go` と `org_director_wait_line_test.go` が読むのは `ralph status` の `--help`)ので、(c) は残りについて開いたまま |
| 200「Small findings of org-feature-worktree」(k) | `split.go` の先頭の comment に機能ごとの digest の段落を足した | (k) が言う段落(コードフェンスの段落)と `parseSplitPlan` の comment は変わっていない。持ち越し |
| 201「Test gaps left by org-feature-worktree」(e)(f) | `StartFeature` の doc | 注釈だけの変更。2026-11-06 と 2026-11-12 の note も doc の変更を数えているので、同じ形で数えた。走らない枝は今も走らない。持ち越し |
| 223「Small leftovers of org-plan-approval」(c) | `featureBindingFromTokens` の鍵の表と comment | (c) が求めるテスト(読めない承認方式の binding を補償で書き戻し、`RequiresApproval` が保たれることを見る)は足していない。`TestOrgCloseDeferredSelf_CloseFails_BindingRestored` は期待の Details に token を足しただけ。持ち越し |
| 396「`resolvedOrClean` in `internal/org/statedir.go`」 | 呼び出し側の `feature.go`、`split.go`、`feature_test.go` | どの `resolvedOrClean` の呼び出しも編集していない。前の note と同じ形。持ち越し |

掃引で見つけた 2 行(verify も handoff も挙げていない):

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 425「Small leftovers of standard-config-gaps」(c) の 4 つのテスト | `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart`(`org_approve_test.go`)と `TestOrgStartPlan_ApprovalModeFromMainWorktree`(`org_feature_test.go`)の assert と comment | Trigger は「4 つのテストのどれかの次の変更」で、`commitRalphTomls` を呼ぶこと。2 つとも今も 2 つの `ralph.toml` を自分で書いている。持ち越し |
| 427「Seven test gaps of org-cross-review」(6) | `ralph org status` の JSON(`orgFeatureJSON` の `feature_digest`) | Trigger は「`ralph org status` の JSON の次の変更」。座席の `role` と `driver` は変わっていないが、字義どおりに当たる。実 JSON で leader の driver コマンドを回すテストは今もない。持ち越し |

### V3: 220 行目の項目の文

`--json` の `feature` が運ぶ欄の列挙に「(the `--json` object also its `feature_digest`, left out for a record an older ralph wrote)」を足した。`feature:` の行は digest を出さない(split、slug、branch、worktree だけ)ので、括弧で `--json` の object だけと限った。

### V4: `internal/org/seat.go` の comment

`EventScopeReserved` の comment は binding を `split= feature= digest= branch=` の 4 つと書いていて、`approval=` は前から抜けていた。今回 `feature_digest=` も抜けたので、`split=<id> feature=<slug> digest=<hex> branch=<branch> approval=<mode> feature_digest=<hex>`(最後の 2 つは binding が持つときだけ)に直し、順と理由は `reserve.go` の `scopeReservedPathsKey` を指すとした。`gofmt -l` は出力なし、`go build ./internal/org/` は通った。comment だけの変更なので、テストは回し直していない。

### I1: `AGENTS.md`

触らなかった。`internal/org/` の行は 1 行が非常に長く、org のほとんどの変更が触る場所で、別のセッションとの衝突が起きやすい。機能ごとの digest は split plan の中の仕組みで、`split.go` の先頭の comment、spec の FR-4 の注記、skill が書いている。`sameFeature` と `feature_digest` で grep すれば見つかるので、map の行に足す理由が弱いと判断した。足すなら「split plans and their approval digest」の後ろに「and each feature's own digest」を足せば足りる。

## 新しい tech-debt の行(468〜470 行)

| 行(先頭の語) | 内容 |
|----|----|
| 「Two gaps of the feature digest of a restarted feature org」 | (a) 依存先の機能の節にだけ書いた約束は依存する側の leader に渡らず、直しても依存する側の立て直しは断られない(計画 D7。skill が約束を前文に書くよう言うのが緩和)。(b) 古い binary の自己終了の補償が binding を書き戻すと `feature_digest=` が落ち、全体の digest の比較に戻る(断る側に倒れる)。どちらもコードから読んだもので、古い binary は動かしていない。Trigger は(a)が報告か `Depends on` に slug 以上を渡す変更、(b)が `reserveAgain` か `featureBindingFromTokens` の鍵の次の変更か、古い binary を動かせるとき |
| 「Test gaps of fix/split-feature-digest」 | (a) test report の Test gap 1: `featureDigestRefusal` の guard(R12b〜R12f、R12i の 6 変異が package 全体でも生き残る)。`TestReservationDecision_Feature` の 4 行が `want` の `FeatureDigest` を held と同じにしているため。R12c は `StartFeature` から届く。(b) Test gap 3: `featureLeaderTask` に欄を足したら `featureDigest` にも足す約束は comment だけ。(c) Test gap 2: 実機の herdr と本物の `ralph-worktree.sh` で立て直していない。(d) Test gap 5: skill の立て直しの文を固定するテストが無い。Trigger に test report が作った throwaway の形を写した |
| 「Five small findings of fix/split-feature-digest」 | self-review の L1〜L5。L1 `featureDigestRefusal` が `sameFeature` の 5 つの比較を写している、L2 `approvedDigestPattern` の流用、L3 局所変数 `featureDigest` が関数を隠す、L4 壊れた token の記録にも「an older ralph」と言う、L5 comment が長い 1 文。判定は変えない。Trigger は `sameFeature` と `featureDigestRefusal` の次の変更、`validateFeatureBinding` と `String()` の次の変更、拒否の文言の次の変更 |

行の中の名前はすべて関数、テスト、変異の名前で、新しい `file:line` は無い(touched した行で `.go:数字`、`.md:数字` の形は 0 件)。参照した名前は `git grep` で実在を確かめた(`featureDigestRefusal`、`sameFeature`、`featureBindingFromTokens`、`validateFeatureBinding`、`approvedDigestPattern`、`featureDigestPattern`、`featureLeaderTask`、`reserveAgain`、`CloseDeferredSelfPane`、`TestReservationDecision_Feature`、`TestFeatureDigest_ChangesOnlyWithTheFeatureAndPreamble`、`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` ほか)。

test report の Test gap 4(古い binary の補償を本物で動かしていない)は 1 つ目の行の (b) に入れた。

### 掃引

Trigger の欄を python で切り出し(`" | "` で分けた 4 つ目のセル、バッククォートの中の `" | "` は区切りにしない、`~~...~~` を外し、最初の日付つきの note より前だけ)、diff の足した行と消した行と hunk の関数名から、Trigger が挙げる識別子(CamelCase か `_` を含むもの、ファイル名)を探した。当たった行は Trigger を 1 つずつ読んだ。`status --json`、`ralph org status`、`FeatureBinding`、`scope_reserved`、`sameFeature`、`split.go`、`start --help`、`skill` の語でも Trigger を探した。

当たらない行と理由:

| 行(先頭の語) | 判断 |
|----|----|
| 189(F-2 の `scopeReservedEvent`)、194(`TestReservationBeforeLastDisband`)、191、192 の (a) | diff に名前が出るのは comment の言及か hunk の文脈の関数名だけで、Trigger が挙げる関数の本体は変わっていない |
| 195「A `ralph` of another version on the pane's `PATH`」 | Trigger の `newOrgStartCmd` の `RunE` は変えていない(help の文だけ)。「the next release that changes ... the event shape」は、`scope_reserved` の Details に token が増える release を指すとも読めるが、release はまだなので note は足していない。release を切るときに lead が決める(下の「残っていること」) |
| 200 の (i)、(l)、201 の (j)、(k)、391 の (b)、368、377 | `maxFeatureOrgIDLen`、`StartFeature` の `p.StateDir`、`splitFence` と fence の扱い、`StartFeature` の検査、`LoadSplitPlan` の警告、skill の分割計画の例は、どれも変えていない。377 の Trigger は skill の「split-plan example」で、diff が足したのはその外の箇条書き 1 つと承認の節の 1 段落 |
| 137、274、288、316、358、432、249 | 挙げる関数とテストは変わっていない(`fakeHerdr` は feature_test.go の文脈に出るだけ) |

## 確かめたこと(文書が実装と合っていた所)

- org skill の 4 つの写し: `shasum` が同じ(`43d03de5…`)。`./scripts/check-skill-sync.sh` は 14 skill で ok(verify)。status の行、予約の記録の欄、分割計画の節、立て直しと拒否の節が新しい規則を言う
- `ralph org start --help` と `ralph org status --help` は verify が HEAD から build した binary で読んだ。`TestOrgStartAndStatusHelp_FeatureDigest` が文を固定する
- `README.md` の org の段落(274 行目)と `docs/quality/quality-gates.md` の Split plan approval の行(82 行目)は、「承認後に変わった計画は拒否する」と書き、binding の比べ方は書いていない。今も正しい。書き足していない
- `docs/architecture/repo-map.md`、director skill、leader の雛形、`templates/base/docs/`: 比べ方を書いておらず、食い違う文は無い(verify が確かめた)
- 計画の D1〜D7 と AC1〜AC11 は verify が照合済み。この pass で本文は変えていない

## 残りの項目の扱い

| 項 | 扱い |
|----|----|
| L1〜L5 | 1 つの行にした(470 行目)。コードを変えるのでパイプラインを回し直す理由にならない。cross-review の前に直すなら 1 commit にまとめられる、と self-review が書いている |
| Test gaps 1・3・2・5 | 469 行目 |
| Test gap 4 | 468 行目の (b) |
| 実機の herdr での立て直し | 469 行目の (c) |

## 残っていること(lead へ)

- コードの動きとテストは変えていない(Go は comment 1 か所)。パイプラインの再走は要らない
- 払っていないもの: 持ち越しの note を足した 8 行の Trigger と、新しい 3 行の中身(どれもコードかテストの変更)
- 195 行目(ralph の版のずれ)の Trigger「the next release that changes ... the event shape」に、この変更(`scope_reserved` の Details の新しい token)が当たるかは判断が分かれる。release を切るとき、古い binary を pane の `PATH` に置く運用の人に向けて、release ノートの項目(計画の Rollout: 「`start --plan` は、その機能の節と前文が変わっていなければ、分割計画のほかの節の変更では断らない(この版より前に立てた org は対象外)」)に加えて、古い binary の補償が token を落とすことを一言書くかを決めてほしい。note は足していない
- `AGENTS.md` は触っていない(上の I1)。足すなら数語で、衝突しやすい行である
- 同じファイル(`docs/tech-debt/README.md`)を別のセッションも編集している。この commit は 9 行への挿入と末尾の 3 行の追加で、既存の行は消していない。merge の衝突は末尾の追加行と、180・201 行目の Trigger の末尾で起きうる
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 180、192、200、201、220、223、396、425、427(どれも挿入だけの編集、`difflib` の操作が `insert` のみ)と、末尾の新しい 3 行(468〜470)。変わった行も新しい行も列は 5、列ごとのバッククォートと `~~` は偶数。新しい `file:line` は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-split-feature-digest.md` は編集の前後とも `5a95d18e25ff`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`git diff --check` は rc 0。変えた 3 ファイル(spec、tech-debt、`seat.go`)の U+FFFD は 0 件。`gofmt -l internal/org/seat.go` は出力なし、`go build ./internal/org/` は通った
- 実行していないもの: テスト、`run-verify.sh` の全体(コードの動きを変えていない)

## Insight event

```
./scripts/insights-append.sh --slug split-feature-digest --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-split-feature-digest.jsonl` が 3 行から 4 行になり、足した行は cycle 1、pass、ts は 2026-11-12T11:40:55Z。この時点の `git status --short` は spec、計画、tech-debt、`seat.go`、既存の jsonl の `M` だけで、迷子のファイルは無かった。
