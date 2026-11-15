# sync-docs report: reseat-leader-progress

- Date: 2026-11-16
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(承認の digest `de0cca0b6f78`)。issue #312 の #312c-2
- Branch: `feat/reseat-leader-progress`。base は origin/main `48a8a661`(#335 の PR。`e3914c93` で取り込み済み)。run 開始時の HEAD は `fc8f236a`(test report の commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は 3 回の run(`64c7f828`、`c9ce0f77`、`f57e37f4`。最後は Merge yes)、verify は `12a76fc7` が fail(V1)で、`e483b52a` で直して `1a1d8981` で打ち直し、partial-pass、test は `fc8f236a` が pass。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD` と、3 つの report の V2〜V10、Documentation drift、register への引き継ぎ。文書の各文は HEAD のコード(`reseat_progress.go`、`feature.go`、`reseat.go`、`leader.md`、`org_reseat.go`)と突き合わせた
- 制約: `go test ./...` と `run-test.sh` は回していない(handoff の指示)。Go のファイルは `ralph org reseat --help` の 1 文とその pin の 2 か所だけ変えたので、`go test ./internal/cli/ -run 'Reseat' -count=1` を回した(`ok`、26.1 秒)。ほかに `go test ./internal/org/ -run 'Skill'`(`ok`)と `go test ./internal/config/`(`ok`)
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `77a45022` を含み、あとの report の commit は docs だけなので、docs の外の数は変わらない): `git diff --shortstat origin/main...HEAD` は 27 ファイル、+2,558/-54(計画、report、insights を含む)。docs の外は 19 ファイル、+2,092/-40。docs の外でテストを除くと 15 ファイル、+651/-40(Go の本体 4 ファイルが +460/-2: `reseat_progress.go` 412、`feature.go` +26/-1、`reseat.go` +16、`org_reseat.go` +6/-1。`leader.md` が +89。残りは skill の 8 つの写しと AGENTS.md、README)。Go のテストは 4 ファイルで +1,441。walkthrough は書いていない

## Summary

verify の V2〜V6 と V9 は直した。V10(register)は、Trigger に当たった 7 行に持ち越しの注を足し、新しい 3 行を足した。V7 と V8 は新しい行(テストの穴)に、self-review の L5 と L3 の残りと S3 の implementer の 3 点は新しい行に入れた。`ralph org reseat --help` は、すでに leader のタスクが持つ物(`[reviewer]` の 4 つの値)を書いていたので、節の 1 文を足して pin した。`ralph org start --help` は触っていない(下の「help の扱い」)。

verify が `/org` skill の `start` の行と「機能ごとの org」の leader のタスクの説明にも 1 文足してよいとしていた件は足していない(下の「直さなかったこと」)。

## Changes made

| File | Change |
|------|--------|
| `README.md` | `ralph org reseat` の段落の最後の 2 文(「reseat does not tell the leader …」)を、立て直した leader が節を受け取り、`leader.md` の規則で続きから進むことに置き換えた。下の「V2」 |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「座席の立て直し」節の最後の段落(「reseat はまだ、それまでの経過を leader に渡さない」)。4 つの写しは同じ文。下の「V3」 |
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | 「座席を立て直す」節の最後の段落(「leader も立て直したときは、reseat はまだ経過を渡さないので…」)。4 つの写しは同じ文。下の「V4」 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-12 改訂 (d) の末尾に改訂の印、「2026-11-15 改訂(feat/reseat-leader-progress)」の項を足した。下の「V5」 |
| `AGENTS.md` | `internal/org/` の行に `reseat_progress.go` の一節。下の「V6」 |
| `docs/specs/2026-11-12-org-reseat.md` | Impact の表の 3 行目。下の「V9」 |
| `internal/cli/org_reseat.go`、`internal/cli/org_reseat_test.go` | `reseat --help` の `ralph.toml` の段落に節の 1 文(5 行)、`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` にその pin(1 項目)。下の「help の扱い」 |
| `docs/tech-debt/README.md` | 持ち越しの注 7 行(514、201、365、426、396、513、515)と、新しい 3 行(519〜521)。既存の字は消していない。下の「V10」 |
| `docs/plans/active/2026-11-15-reseat-leader-progress.md` | Progress の sub-bullet 3 つ(verify、test、sync-docs)と、3 つの箱(`Review artifact created`、`Verification artifact created`、`Test artifact created`)を `[x]` に。計画の本文と AC の箱は変えていない |
| `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md` | この report |
| `docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` | `sync_docs` の行(cycle 1、pass)。ts は 2026-11-15T15:57:19Z(UTC の日付なので、既存の file に足された) |

commit は 2 つ。内容の commit `77a45022`(上の表の README から計画の Progress まで。16 ファイル)と、この report と insights の commit。

## verify が挙げた drift

### V2: README

reseat の段落の終わりを、次の内容に直した。節の名前(立て直しの経過)、中身(最新の計画の承認、最後の gate の記録、open か acked の受信箱の件、branch の新しい順 20 件の commit と総数、今回の reseat の id と leader のあとに起動する座席、前の reseat)、`start --plan` の立て直しも同じ節を足すこと、経過のない org と、受信箱か git が読めなかっただけの場合は節がないこと、leader は `leader.md` の規則で手順 1 からではなく、座席の確認、計画、gate、受信箱、座席の順に進むこと(承認済みの計画は書き直さない、HEAD と同じ gate の pass は PR の手順へ)、DECISION は節が言えないことと、leader が動き続けているあいだに reseat が立てた座席のためのものであること。「see above」は、段落の前のほうにある DECISION の説明(leader が対象でないとき)を指す。

### V3: `/org` skill

「leader も立て直したときは(`start --plan` の立て直しも同じ)」で始め、節の足し場所(`StartFeature` が leader の起動の前)と中身、節がない場合、leader の規則(まず leader 以外の座席の状態、次に計画、gate、受信箱、座席)、director か人が経過を DECISION に書き写す必要はないこと、節の条件に当たらないとき(計画の digest が承認のときと違う、gate の head が今の HEAD と違う、など)は雛形の手順のとおりその手順から進むこと、を書いた。1 行の幅は 78 桁以内(隣の行の最大と同じ)。編集は `.claude/skills/org/SKILL.md` にだけ手でして、`./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`templates/base/` の 2 つは `cp` した。`check-skill-sync.sh` は 14 skill で ok。

### V4: `/director` skill

前提を直した: leader のタスクには節が入り、leader は `leader.md` の規則で続きから進むので、director が経過を DECISION に書き写す必要はない。leader 以外の座席だけを立て直したときに reseat が出す DECISION は、これまでどおり director が送る(同じ節の 6)。leader がそれでも済んだ手順をやり直し始めたら、人に伝え、DECISION で伝えるかを決めてもらう手順は残した。disband のあとの `start --plan` は承認が消えているので、承認を求め直すのは誤りではない、と括弧に足した(`leader.md` の計画の項目が、disband のあとの per-plan は手順 2・3 から進むと書く)。「DECISION を「答える順」の 1 と 2 のとおりに送る」の文は変えていない。

### V5: `docs/specs/2026-08-01-org-runtime.md`

(d) の末尾に `**(2026-11-15 改訂で変更: …)**` を足した(この仕様の前の印と同じ形)。新しい項は、前の最後の改訂の項(`2026-11-15 改訂(fix/send-codex-paste)`)の次、`### 運用ノート` の前。(a) は節の中身と足す場所、`reseat` と `start --plan` の両方、節がない場合、`leader.md` の 5 項目の規則。もう 1 項は「本文は書き換えない」で、(d) の印と、leader 以外の座席だけを立て直したときの DECISION がこの改訂の外であることを書いた。項の日付は 2026-11-15 とした(姉妹の仕様 `2026-11-12-org-reseat.md` の Status の段落と同じ日で、実装の日)。この run は 2026-11-16 の日本時間。

### V6: `AGENTS.md`

`internal/org/` の行の reseat の項(`reseat.go` … `one reseated event)`)のあと、`and the pane lock` の前に、`the progress section of a restarted leader's task (reseat_progress.go: …)` を足した。中身の 6 つ(最新の計画の承認、最後の gate の記録、open と acked の受信箱の件、branch の commit、今回の reseat の座席、前の reseat)、`StartFeature` が leader を起動する前に足すこと、reseat と `start --plan` の立て直しで同じこと、経過のない org では足さないこと。行のほかの字は変えていない。`check-sync.sh` は PASS(`internal/org/` の行は root だけの行で、`templates/base/AGENTS.md` には無い)。

### V9: `docs/specs/2026-11-12-org-reseat.md`

Impact の表の `feature.go` の `featureLeaderTask`(2 本目)を、`internal/org/reseat_progress.go` と `feature.go` の `StartFeature`(`featureLeaderTask` は変えず、`StartFeature` が節を足す)に直した。verify が V9 の (2) と (3) とした 3 点(`:7` と `:36` の日付つきの過去の記述、`:15` が引く `docs/plans/archive/...` は `/pr` のあとにだけ在る)は直していない。`:15` は 2026-11-15 の段落で、`/pr` が計画を archive へ移す名前は同じ(`archive-plan.sh` が `docs/plans/active/<name>` を `docs/plans/archive/<name>` に移す)。

### V10: tech-debt

Trigger に当たった行(行番号は commit `77a45022` のもの。verify の時点から動いていない)。どの行も、Trigger のセルの末尾に日付つきの note、Related のセルの末尾に `docs/plans/active/2026-11-15-reseat-leader-progress.md`(`/pr` の `archive-plan.sh` が書き換える。`archive/` とは書いていない)を足した。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 514「`internal/org/reseat.go` (1,307 lines)」 | `reseat.go` 1,537 → 1,553(+16)、`feature.go` 848 → 873(+25)。この run の help の pin で `org_reseat_test.go` 923 → 926 | 持ち越し。`startFeatureInput` とその 2 つの隣は編集していない。新しい 4 ファイル(412、589、411、438)は 800 行未満。`reseat_test.go` は 1,868 のまま。数は `e483b52a` の `wc -l`、`internal/cli` の 2 ファイル(526、926)は help の文のあと |
| 201「Test gaps left by org-feature-worktree」(e)、(f) | `StartFeature`(節の呼び出しと `StartFeatureParams.Reseat`) | 持ち越し。(e) の通らない枝と (f) の防御の枝は測り直していない。新しい呼び出しは 4 つの `TestReseatProgress_StartFeature*` が通す |
| 365「`internal/org/verbs.go` is 2277 lines」 | leader の役割の prompt の新しいテスト `prompts_leader_reseat_test.go`(438 行) | 持ち越し。ほかの 6 ファイル(`verbs.go` 2639、`verbs_test.go` 5893、`prompts_test.go` 2727、`spawn.go` 3109、`spawn_test.go` 6408、`director_skill_sync_test.go` 1318)は触れていない |
| 426「The role prompts grew with org-cross-review」 | `leader.md` 1,391 → 1,480(+89)。見出しは足していないが、「機能ごとの org」の導入に規則(段落と 5 項目)を足した | 持ち越し。字義どおりの「section」ではないことと、この行の直し方の対象に当たることを書いた |
| 396「`resolvedOrClean` in `internal/org/statedir.go`」 | `StartFeatureParams.Reseat` の新しい field と節の呼び出し(2026-11-05 の `DriverFromRoleEntry` の前例に従った) | 持ち越し。`resolvedOrClean` の呼び出しは編集していない |
| 513「Fourteen small findings of docs/spec-org-reseat」(f)、(j)、(m)、(n) | この run の help の 1 文と pin(`orgReseatHelp`、`TestOrgReseatHelp_…`) | 持ち越し。4 項目はどれも払っていない。新しい文は `ralph.toml` の段落にあり、4 項目が名指す所ではない |
| 515「Test gaps of docs/spec-org-reseat」(h)、(n) | (h) は `/org` skill の reseat の節と `/director` skill の「座席を立て直す」の編集。(n) は help の 1 文 | 持ち越し。(n) の文は 1 語ずつ pin したが、前の文は 1 つの断片ずつのまま。変異は回していない。command table の reseat の行は編集していない |

新しい行(519〜521。行頭の語で探す)。

- 「Small findings of feat/reseat-leader-progress (#312c-2), the first run of its self-review: L5 and the remainder of L3」: (a) `reseatProgressSection` が 86 行、(b) `reseatProgressFirstBodyLine` と `inboxSummary` の二重、(c) `reseatProgressText` が呼ぶたびに `NewReplacer` を作ること、(d) gate の行の括弧が値に付くこと。(d) は handoff の `branch <名前>(<時刻>)` と `via <名前>(<時刻>)` に、コードを読んで見つけた `pass(gate_passed)head <sha>` を足した(`reseat_progress.go:220`)。規則が比べるのは `head` と `base` だけなので判断は変わらない
- 「Known gaps of feat/reseat-leader-progress (#312c-2), about the rule that `internal/org/prompts/leader.md` gives a re-seated leader」: (a) `--state all` は merge されずに閉じた PR も「ある」と読む、(b) PR を作ったあと RESULT を上げる前に止まった leader は手順 15 に進み RESULT が上がらない、(c) 「承認なし、コミット済みの計画」から手順 2 に進むとき計画助言が済んだか分からない(計画助言の結果の残る場所を読んで、Trigger に書いた: `.harness/state/plan-advisory-out.md`、split の計画の `## Design decisions`、per-plan の QUESTION の `ADVISORY:` の行)、(d) 節を読んだ leader の実機での動きは見ていない(仕様 AC-12 と同じ限り)
- 「Test gaps of feat/reseat-leader-progress (#312c-2), found by its verify report (V7, V8) and its test report (Test gaps 1 and 2)」: (a) `StartFeature` を通した「disband のあとの start で commit だけの節」のテストがない(部品は別々のテストが見る)、(b) 読み取りの失敗の行は節が別の理由で在るときだけ載る(計画の D6・AC7 のとおりの設計で、バグではない)と明記した

どの新しい行も 5 つのセルで、バッククォートと `~~` は列ごとに偶数。計画の参照は `docs/plans/active/2026-11-15-reseat-leader-progress.md` だけ。

## help の扱い

- `ralph org reseat --help`: 「A leader it starts gets the current values of the four [reviewer] keys in its task」と、すでに leader のタスクの中身を書いていたので、handoff の条件(すでに leader が受け取る物を書いているとき)に当たる。その段落に 1 文足した: 「A leader it starts also gets, at the end of its task, what the org holds for its current life, when it holds any: the latest plan approval, the last gate record, the open and acked inbox items, the branch's commits, the seats this reseat starts after the leader, and the previous reseat. Its role prompt tells it to go on from there.」。既存の 2 つの pin の断片(`[reviewer]` の文と、`The reseat, stop, spawn, …` の文)は、間に入れても壊れない。新しい文を `TestOrgReseatHelp_…` の項目に足した。`go run ./cmd/ralph org reseat --help` で段落の見た目を確かめた
- `ralph org start --help`(`internal/cli/org.go`): 触っていない。verify は reseat の help と同じ扱いで「間違いではなく不足」とし、handoff が reseat の help だけを指したので、start の help は handoff の範囲の外とした。start の help も leader のタスクの `[reviewer]` の値を書く(段落の終わりの「written in the leader's task as two lines」)ので、足すなら同じ種類の 1 文で足りる。start の help の pin の場所は、この run では調べていない

## 直さなかったこと

- `/org` skill の command table の `reseat` の行と `start` の行: 節への 1 文を足す案(verify の V3 の備考)は、いったん `reseat` の行に足したが、tech-debt の行 180 の (a)(「次に command table を編集するときは、各 cell を 1〜2 文に切る」)に当たり、cell を伸ばす向きなので戻した。節の説明は「座席の立て直し」節の最後の段落にあり、table の行は「座席の立て直し」節を指す
- `docs/specs/2026-11-12-org-reseat.md` の `:7` の「2 本目は未着手」と `:36` の「経過を持たない」: 日付の付いた過去の記述で、verify も直さないとしている
- Go のコメントが指す `docs/plans/archive/2026-11-15-reseat-leader-progress.md`: `/pr` が計画を移したあとにだけ在る名前。`archive-plan.sh` の archive 先の名前と同じことを確かめた(`docs/plans/active/` の名前をそのまま `archive/` に移す)
- 計画の本文と AC の箱: 変えていない。`./scripts/plan-visual.sh digest` は編集の前後とも `de0cca0b6f78`

## 掃引

Trigger 列を python で切り出し(`(?<!\\) \| ` で分けた 4 つ目のセル、全体が取り消し線の行は外す。日付つきの note より前の部分を読む)、この branch が変えた file、関数、skill、doc の語 36 個で候補を出した(51 行)。1 回目の候補から、原文の Trigger を読んで判断した。

当たった行は上の表のとおり。当たらないとした行。

| 行(先頭の語) | 判断 |
|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の command table の編集に当たる。最終的に table は編集していない(上の「直さなかったこと」)ので、持ち越しの注は足していない |
| 177、244〜252、262、355、380、381、472 | `/director` skill の別の節(`stop --all` の説明、progress の報告、inbox の節、send の確かめ、wait の終了コード、accept)の編集を待つ。この branch が編集したのは「座席を立て直す」の最後の段落だけ |
| 200(g) | 取り消し線の項目(`f1b0352e` で閉じた)。`leader.md` の機能ごとの org の節を編集したが、(g) は終わっている |
| 377、384、367、372、427、508 | `leader.md` の機能ごとの org の手順 1・4・14、「運用規律」の hook の項、cross-review の小節、reviewer の手順を名指す。この branch は規則を「次の順で進めます。」の前に置いただけで、手順は変えていない |
| 195、203、204、199、391、431、432、425、475、476 | `StartFeature` や `featureLeaderTask` の別の部分(version skew、worktree の再利用、digest、herdr のゲートの順序、設定の読み)の Trigger。`featureLeaderTask` は変えていない |
| 516、517 | `Disband` と `takeReseatLock`、`ApprovalDecisionCommand` と `orgReadCommandHint`。どちらも変えていない |
| 371、401、468 | README を挙げるが、Trigger は recipe の引用、`[org].state_dir` の段落、scaffold のノートで、reseat の段落ではない |

## 確かめたこと(文書が実装と合っていた所)

- `reseat_progress.go`: 「新しい順に <n> 件」(`reseatProgressMaxCommits` は 20)、「古い順に <n> 件」と「ほか <n> 件(新しいもの)」の受信箱、読み取りの失敗は別の理由で節があるときだけ(`reseatProgressSection` の doc と `progress` の変数)、`今回の立て直し:` と `前の立て直し:` の 2 行。skill と README の言い方と合う
- `leader.md:150-237`: 導入と 5 項目(座席の確認、計画、gate、受信箱、座席)。skill の「まず leader 以外の座席の状態、次に計画、gate、受信箱、座席」は、この順
- `reseat.go` の `startAll` が `StartFeatureReseat{ID, Seats}` を渡し、`StartFeature`(`feature.go`)が leader の spawn の前に `startProgressSection` を呼ぶ。AGENTS.md の「before it starts the leader」と合う
- `ralph org reseat` の終わりの文(`internal/cli/org_reseat.go` の「reseat sends the leader nothing, and the seats it started have no TASK」)は leader が対象でないときの DECISION の注で、今も正しい(verify の「Yes」の行)
- `docs/quality/`、`docs/architecture/repo-map.md`、`docs/recipes/`、`.claude/rules/ralph/` に、reseat が leader に経過を渡さないと書いた所は無い(`git grep` で確かめた)

## 残っていること(lead へ)

- 決めてほしい点: この run は Go を 2 か所変えた(help の 1 文と、その pin の 1 項目。`77a45022` の `internal/cli/org_reseat.go` と `org_reseat_test.go`)。`/test` の report(`fc8f236a`)のあとの Go の変更で、フルの `run-test.sh` はこの 2 か所を通していない。`go test ./internal/cli/ -run 'Reseat'` は通った。パイプラインの規則では、Go の変更は `/self-review` からの回し直しになりうる。help の文を入れずに進めるなら、`77a45022` の 2 つの Go のファイルだけ戻せばよい(pin も一緒に消える。tech-debt の行 513 と 515 の note は help の文を前提に書いたので、戻すときは note の「added one sentence to `orgReseatHelp`」の段落も外す)
- 払っていないもの: 持ち越しの注を足した 7 行は、どれも「Carried over to the next change that meets this trigger」。新しい 3 行の項目もすべて開いている
- 節を読んだ leader の実機での動きは見ていない(新しい行の (d)、仕様 AC-12)
- `docs/tech-debt/README.md`、`AGENTS.md`、4 つの skill の写しは、ほかの session も編集しうる。この run の tech-debt の変更は新しい 3 行と 7 行への追記で、既存の字は消していない。merge の衝突は末尾(新しい 3 行)と、365・514 の Trigger の末尾で起きやすい

## Verification

- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-reseat-leader-progress.md` は編集の前後とも `de0cca0b6f78`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`./scripts/check-skill-sync.sh` は 14 skill で ok
- `go build ./...` は成功、`go vet ./internal/cli/ ./internal/org/` は出力なし、`gofmt -l internal/` は出力なし、`git diff --check` は rc 0
- tech-debt: `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で、無い計画の参照は 0 件。触った 10 行はどれも 5 つのセルで、バッククォートと `~~` は列ごとに偶数。`git diff --word-diff=porcelain` に出る消えた語は 5 つで、どれも語の後ろに句読点(`.`、`;`)を足した所
- `go test ./internal/cli/ -run 'Reseat' -count=1`: `ok  github.com/yoshpy-dev/ralph/internal/cli 26.120s`。`go test ./internal/org/ -run 'Skill' -count=1`: `ok`。`go test ./internal/config/ -count=1`: `ok`
- 変えたファイルの U+FFFD は 0 件
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`(handoff の指示)、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)、`/natural-japanese` の lint(hook は書き込みごとに回り、指摘は出なかった)

## Insight event

```
./scripts/insights-append.sh --slug reseat-leader-progress --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-reseat-leader-progress.jsonl` に足した行は cycle 1、pass、ts は 2026-11-15T15:57:19Z(UTC の日付が 2026-11-15 なので、既存の file に足された)。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。
