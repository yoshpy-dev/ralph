# Walkthrough: org-feature-worktree(4 段目)

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-09-org-feature-worktree.md(PR 作成時に active から移す)
- Branch: feat/org-feature-worktree(base 765da6bd、29 commits、39 files、+7975/-446。うち `internal/` は 20 files、+5647/-256 で、その半分ほどがテスト)

差分は大きいが、新しいファイル 2 つ(`split.go` と `feature.go`)とその周りの直しでできている。次の順に読むと、依存の向きに沿って追える。

## 1. 分割計画(`internal/org/split.go`)

`ralph org start --plan` が読むファイルの形式と検査。

- `ResolveSplitPlanPath` は、`--plan` が台帳の下の `splits/<id>.md` を指すことを、symlink を解決してから確かめる
- `LoadSplitPlan` → `parseSplitPlan` は、先頭の `- Status:` / `- Approved:`、`## Features` の下の `### <slug>` の節、各節の `- Type:` / `- Reserve:` / `- Depends on:` を読む。残りの行は本文で、leader の task になる
- `PlanDigest` は `scripts/plan-visual.sh digest` と同じ値を返す(`TestPlanDigest_MatchesScript` が本物のスクリプトと突き合わせる)
- `rejectDigestSkippedLine` は、digest が読まない書き方(先頭以外の `- Status:` / `- Approved:`、`- Branch:`、`## Progress checklist`、`- [x]`)を拒否する。承認のあとに書き換えられる本文が、必ず digest に入るようにするため(計画の Codex plan advisory 2)
- `splitFence` はコードフェンスを数え、中の行を見出しやフィールドとして読まない(cross-review cycle 1 の #2、d49bbc34)
- slug は 20 文字まで(`maxFeatureOrgIDLen`、self-review の M1、8aae7ce2)

## 2. 予約の記録への結びつき(`internal/org/reserve.go`)

3 段目の予約の記録 `scope_reserved` に、どの分割計画のどの機能の org かを足した(9d7a85ae)。

- `FeatureBinding` と `Reservation`。Details は `paths=… split=… feature=… digest=… branch=…` で、worktree はイベントの `Worktree` フィールドに入る。古いバイナリは最初の空白で切るので、パスだけを読む
- `reservationFromEvent` / `featureBindingFromTokens`。一部だけ壊れた記録は、どれとも一致しない結びつきとして読む(`Complete`)
- `reservationDecision` は、結びつきを持つ start を、同じ結びつきのときだけ通す。予約のないまま走っている org(昇格したセッションの leader、`start <task>` の org)には立てない

## 3. spawn の入口と先読み(`internal/org/spawn.go`)

- `checkSpawnInput` に Spawn の入力検査をまとめ、`SpawnParams.Feature` の検査を足した。相対の `--cwd` は、打った場所から絶対パスにしてから herdr・agmsg・台帳に渡す(af138af6、実機の Run 2 で見つけた)
- `spawnPrecheckErr` と `idempotentRespawnDecision` は、ロックの下の判定と同じ部品を、書き込みなしで呼ぶ。`TestSpawnPrecheckErr_MatchesSpawn` が、同じ台帳で両方のエラーが一致することを固定する

## 4. start --plan の手順(`internal/org/feature.go`)

`(*Org).StartFeature` の順は計画の図 2 と同じ。

1. `readStartFeature`: 分割計画の検査、承認、機能、org_id(既定は slug、20 文字まで)
2. `spawnPrecheckErr` による先読み。通らなければ worktree を作らない
3. `checkFeatureWorktreeReuse`: 残っている worktree の記録(`canonical_ref`・パス・ブランチ・kind)と、実際のチェックアウトを照合する(計画の Codex plan advisory 1)
4. `scriptFeatureWorktrees.Ensure` が `scripts/ralph-worktree.sh ensure` を main worktree のルートで打つ。失敗は `ensureFailureErr` が原因ごとの案内にする(self-review の L1)
5. `o.Spawn` で leader を立てる。task(`featureLeaderTask`)には、台帳の絶対パスと、`ralph org` のコマンドに付ける `--state-dir` が入る(cross-review cycle 1 の #1、d49bbc34)

## 5. 台帳と repo のルート(`internal/org/statedir.go`)

- `FeatureRepoRoot` は、worktree を作る main worktree のルートを決める
- `LedgerMainWorktreeRoot` は、`--state-dir` / `RALPH_ORG_STATE_DIR` が main worktree の既定の台帳を指すとき、全体の上限を main の `ralph.toml` から読む(07d38e6d)。3 段目は「既定の解決のときだけ」だったので、ここは 3 段目の挙動の変更にあたる。leader が `--state-dir` を付けても、feature branch の `ralph.toml` で上限を変えられないようにするため

## 6. CLI(`internal/cli/org.go`)

- `newOrgStartCmd`: `--plan` / `--feature` と、併用できないフラグの検査(`checkOrgStartPlanInput`)。`--plan` なしの形は今のまま
- `runOrgStartPlan`: 台帳の解決は 1 回(`newOrgSpawnRuntimeAt`)
- `ralph org status`: `feature:` の行と JSON の `feature`(壊れた記録は `(incomplete record)` と `"incomplete": true`)
- `withMainWorktreeOrgLimits` が `LedgerMainWorktreeRoot` を使う。`orgWideLimitsHelp` もそれに合わせた(9c1d447f)

## 7. 補償(`internal/org/verbs.go`)

`releasedReservation` は `Reservation` ごと返し、`reserveAgain` は自分の close の失敗のあと、結びつきと worktree も一緒に書き戻す。それ以外の補償の条件は 3 段目のまま。

## 8. 雛形と文書

- `internal/org/prompts/leader.md`: 既定の編成(implementer 1 席・reviewer 1 席)と「機能ごとの org」の手順。締めの順は、座席の stop → report → 計画の archive → コミット → secret scan → push → `gh pr create` → disband(report を PR に入れ、worktree に未追跡のファイルを残さないため)
- `/org` skill の 4 面: 「編成パターン」の節を「機能ごとの org」に置き換えた。分割計画の形式、承認、拒否の一覧、後始末を書いている
- README、AGENTS.md、仕様 FR-4 の「4 段目で決めたこと」、`docs/quality/quality-gates.md`(2 面)、`templates/base/ralph.toml` のコメント、`docs/tech-debt/README.md`

## パイプラインの往復

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review cycle 1 | M1 slug の長さと herdr の agent 名の上限、M2 出荷する skill に未実装の director の説明、L1・L3・L4 | 8aae7ce2、2c1bb13b |
| cross-review cycle 1 | #1 leader に台帳の場所が渡らない、#2 コードフェンスで本文が切れる | d49bbc34、07d38e6d、9c1d447f |
| cross-review cycle 2 | 別の台帳の同じ名前の分割計画が worktree を引き継げる、別の分割計画からの同時の start | merge の直後の別 PR で直す |

実機の確認は `docs/evidence/org-feature-worktree-live-2026-10-09.md`(Run 1〜4)にある。
