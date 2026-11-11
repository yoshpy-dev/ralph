# sync-docs report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(承認の digest `e5a18ef9477a`)。issue #307 の前半
- Branch: `fix/watch-alert-dedupe`。base は origin/main `028a8f6d`、run 開始時の HEAD は `3506f99a`(test の report のコミット)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` は cycle 1)
- 先行 report: self-review 3 回(最後の 3 回目は Merge: yes、LOW 1 件 N5)、verify `b8058948`(pass。LOW 8 件 V1〜V8、INFO 5 件 I1〜I5)、test `3506f99a`(pass。変異 61 個のうち 4 個が生き残った。M30・M50 は LOW、M48・M56 は INFO)
- 制約: 計画の本文は変えていない。変えたのは AC の 15 個の箱(`[ ]` を `[x]` に)と `## Progress checklist` だけ。`./scripts/plan-visual.sh digest` は編集の前後とも `e5a18ef9477a` で、`- Approved:` の行と同じ(digest は箱の状態を数えず、Progress の節を読まない)。箱だけを編集した時点の `git diff --numstat` は 15/15 で、Progress の編集はそのあとに足した
- コードは変えていない。変えたのは Markdown だけで、`/org` skill の本文を含む

## Summary

verify の V1〜V8、self-review の N5、test の M30・M50 を処理した。`/org` skill(4 つの写し)、README の 2 か所、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその template の写しに、今回の挙動(続いて立つ ALERT は件の `recurred`、待っている leader の stall は立てない、予約のある org の scope_change は予約の外だけ)を足した。tech-debt は、今回の行を 4 つから 6 つの穴に増やし、この変更が Trigger に当たった 9 行に引き継ぎの注か数を足した。self-review の報告の U+2028 は 3 か所で、すべて 6 文字のテキストに直した。

依頼の外で足したものが 2 つある。(1) 既知の穴の行の (6) に M48(INFO、`scopeDefaultBranchRef` の `refs/remotes/origin/` の検査が固定されていない)を入れた。依頼は M30 と M50 だが、test report の Test gaps 3 で同じ種類の穴なので、記録から落とさない。M56(等価な変異)は入れていない。(2) `docs/quality/quality-gates.md` の template の写しに触れたので、tech-debt の 118 行(「`templates/base/docs/quality/quality-gates.md` への次のタッチ」)に注を足した。verify の V6 には無い。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | (V1) `watch` の行に 1 文(続くあいだの再発は件の `recurred` と回数で、leader には ALERT を送り直さない。待っている leader の stall は立てない。予約のある org の scope_change は予約の外だけ)。`inbox` の行の `--json` の欄に `recurrences` と `last_recurred_at`、`show` の `recurred:` の行と履歴の `recurred`。「見張りの ALERT の件」の節に 3 つの項(続いて立つ ALERT、leader の stall、予約のある org の scope_change) | `45e84b3e` |
| `README.md` | (V2) 131 行のコマンドの表の `watch --all` の括弧と、278 行の見張りの段落に、同じ 3 つの規則 | `45e84b3e` |
| `AGENTS.md` | (V5) `internal/org/` の行の、見張りの escalation の括弧に 1 句。続いて立つ alert は同じ件の `recurred` で `watch_recurrence.go`、予約のある org の scope_change は予約の外だけで `watch_scope.go` | `45e84b3e` |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | (V3) Watchdog pulse layer の行に 1 文。2 つの写しは同じ文 | `45e84b3e` |
| `docs/recipes/worktrees.md`、`templates/base/docs/recipes/worktrees.md` | (V4) 「recorded worktree と比べる」を「recorded worktree の `git status --porcelain` を前のサイクルのものと比べる」に直し、予約のある org は予約の外のパス(作業中の変更と前のサイクルからの commit。org 自身の新しい計画と報告は除く)だけを読むことを足した。2 つの写しは同じ | `45e84b3e` |
| `docs/tech-debt/README.md` | 下の「tech-debt」。10 行の変更(462・357・303・211・353・355・258・210・118・180 行) | `984761f8` |
| `docs/reports/self-review-2026-11-12-watch-alert-dedupe.md` | (V7) 33 行の N1 の行の U+2028 を 6 文字のテキストに | `0c727338` |
| `docs/plans/active/2026-11-12-watch-alert-dedupe.md` | (V8) AC1〜AC15 の箱を `[x]` に。Progress の Verification と Test の 2 行を `[x]` にし、sync-docs の項と Rollout の注を足した | この report のコミット |
| `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` | この report | この report のコミット |
| `docs/insights/events/2026-11-11-watch-alert-dedupe.jsonl` | `sync_docs` の行(cycle 1、pass)を `insights-append.sh --slug watch-alert-dedupe --cycle auto` で足した。5 行から 6 行。ファイル名の日付は UTC なので 11-11 のまま(UTC は 23:40)。`git status --short` は既存の jsonl の `M` だけで、迷子のファイルはない | この report のコミット |

`.agents/skills/org/SKILL.md` は `scripts/sync-skills.sh` で作り直し、`templates/base/` の 2 つは `cp` で合わせた。`org` の 4 つの `SKILL.md` は `cmp` で同じ。足した行の表示幅(East Asian wide を 2)は最大 76 で、同じ節の既存の行の最大 79 に収まる。

## 依頼の項目

| 項 | 結果 |
|----|------|
| V1 `/org` skill | 3 か所に足した(上の表)。`/director` skill は変えていない(計画の Non-goals、#307b。verify の I3)。Rollout の文は、計画の本文を変えない指示なので、Progress に「skill の文面を足したので、文面については `ralph upgrade` が要る。挙動は CLI の更新だけで効く」と書いた |
| V2 README | 2 か所に足した。英語の文で、131 行は括弧の中に 1 句、278 行は同じ括弧に 3 句 |
| V3 quality-gates | 2 つの写しとも、deadline の文のあと、`With [org.director].enabled ...` の前に 1 文。`check-sync.sh` は DRIFTED 0 |
| V4 worktrees の recipe | 2 つの写しを同じ文に直した。verify が言う「比べる相手は前の周期の記録」も直した |
| V5 AGENTS.md | 1 句だけ。map のままで、新しい文は足していない。template の `AGENTS.md` には repo map が無い |
| V6 tech-debt | 下の「tech-debt」 |
| V7 self-review の報告 | U+2028 は verify が言う 1 か所でなく 3 か所(33 行の中の 3 つ。fix の欄の「文字を …」、`watch_scope.go:138` の引用、`strconv.QuoteRune` の出力の引用)。すべて 6 文字のテキストにした。直し方は下の「確かめたこと」 |
| V8 計画 | AC の 15 個の箱を `[x]` にした。AC13 は文書の同期が済んだので付けた。digest は `e5a18ef9477a` のまま |
| N5 | 既知の穴の行の (5) に入れた |
| M30・M50 | 既知の穴の行の (6) に入れた(M48 を含む)。テストは足していない(コードの変更になり、パイプラインの cycle を使うため。test report に probe の形がある) |
| 7. そのほか | 下の「確かめたこと」 |

## tech-debt

行は先頭の語で見つけた(行番号は 3506f99a のもの)。5 つの列、各列のバッククォートと `~~` が偶数であること、`git show HEAD:` との比較で変わった行が意図した 10 行だけであること、新しい `file:line` が無いことを、スクリプトで確かめた(`docs/tech-debt/README.md` の plan 参照の検査も通った)。

### 既知の穴の行(462 行)

- 先頭を「Four gaps」から「Six gaps」に直した。Trigger の「`internal/org/watch.go` or `internal/org/watch_scope.go`」に `internal/org/watch_recurrence.go` を足した(V6 (a))。Related に `watch_recurrence.go`、self-review の N5、test の Test gaps 1〜3、verify の V6 を足した
- (5) N5: `evaluateReservedScope` の doc は、予約の外の commit がいつも続きの回数になるように読めるが、ALERT を立てた直後の周期の commit は `raiseOrClear` の Active で抑えられ、新しい ALERT にも回数にもならない。挙動は変えず、doc の言い過ぎとして足した
- (6) 固定されていない 3 つの枝: M30(merge-base が読めないのに origin の既定のブランチがある。何も除かない枝)、M50(HEAD が読めない周期に前の HEAD を据え置く `head != ""`)、M48(INFO。`symbolic-ref` の答えを `refs/remotes/origin/` の下に限る検査)

### Trigger に当たり、引き継ぎの注を足した行

| 行(先頭の語) | 当たったもの | 判断 |
|----|----|----|
| 357(`watch.go` の 1848 行と `watch_test.go` の 3007 行) | `escalateToInbox` の変更と `watch_test.go` の 13 本の新しいテスト | 払わない。`watch.go` は 1865 → 2046 行、`watch_test.go` は 3051 → 3975 行(028a8f6d と 3506f99a で `wc -l`)。新しいコードは `watch_recurrence.go`(147 行)と `watch_scope.go`(449 行)に置いた |
| 303(`newTestWatchRun`) | `watch_test.go` の変更 | 払わない。呼び出しは 28 から 29 になった(`grep` で数えた)。`os.MkdirTemp` のまま |
| 211 (a)(`TestPrintableInboxText`) | `inbox_test.go` の 5 本の新しいテスト(1231 → 1517 行) | 払わない。`TestPrintableInboxText` は `escalate_test.go` に残る。テストの移動は範囲外 |
| 353(`sendAlert` の 10 秒の時間切れ) | `sendAlert` の 1 行(S2 `e9c49d0c`: `alert_id` を作る行が `watchAlertID` に) | 払わない。送り方も時間切れも変えていない。leader への ALERT の数は減ったが、時間切れの原因は分からないまま、と書いた |
| 355(ALERT の件が resolve の期限で 2 回人に届く)、258(期限が尽きた件の承認) | `InboxDeadlineDue` の doc だけの変更(`deadline.go` に 4 行) | 規則は変えていないので払うものがない、と 1 文。コメントだけの変更を当たったとみなす前例(179 行)に合わせた |
| 118(template の `quality-gates.md` への次のタッチ) | この sync-docs が Watchdog pulse layer の行に 1 文 | 払わない。「Must pass in CI」の節も template の verify chain も変えていない(`verify.yml` と `run-verify.sh` に `check-coverage` と `check-pipeline-sync` は 0 件)。verify の V6 には無い、自分の編集で当たった行 |
| 180 (a)(`/org` skill の表の 1 セル) | この sync-docs が `watch` と `inbox` の行を伸ばした | 払わない。表の行(全体)の長さは `watch` が 2,848 → 3,112 字、`inbox` が 1,189 → 1,303 字(`git show HEAD:` と作業の写しで測った)。`stop` と `disband` は 1,208 と 1,219 字のまま。詳細は表の下の節に 3 項で入れた。これも verify の V6 には無い |

### 数を足した行

- 210 行(止める頼みが周期の終わりまで待つ): Debt の列の末尾に、予約のある org の座席は 1 周期に git を最大 9 回打つこと(status、HEAD、origin の既定のブランチの探索は最大 3 回、commit の一覧、前の HEAD、merge-base、ls-tree)と、どれも `DriverCallTimeout` で切られるので、git が止まると 1 座席で最大およそ 90 秒、止める待ちが延びることを足した。Trigger(「`evaluateCycle` の呼び出しの打ち切り方の変更」)には当たらない。`realGitStatus` が `runGitBounded` を使うようになったが、打ち切りの仕組みは同じなので、V6 (e) のとおり Debt の数え方だけに足した

### 退けたもの(自分の掃き出しで読んで外した)

Trigger の欄を python で切り出し(取り消し線だけの行は除く)、変えたファイル、変えた関数の名前、自分が編集した文書(README、AGENTS.md、quality-gates、`/org` skill、worktrees の recipe、agent-messaging)で照合した。約 55 行が候補になり、Trigger を読んで上の表の行を残した(自分の 462 行を除く)。退けた主なもの:

- 117(quality-gates と recipe の「次の restructure」): 1 文を足しただけで、構成は変えていない
- 179(`driver` API に触れたとき): 触れていない
- 262(inbox の deadline の道、`RunWatchAll`): `RunWatchAll` は `GitRun` の配線を足しただけで、`SweepInboxDeadlines` と期限の道は変えていない
- 356・358(`sendInboxItemToHuman` とその呼び出し): 呼び出しの行を変えていない
- 365(`inbox_test.go`): 名前は `prompts_leader_inbox_test.go` の部分一致で、5 つのファイルに入っていない
- 371(5 つの recipe): `worktrees.md` は 5 つに入らず、`docs/evidence/` の引用もない
- 401(`[org].state_dir` の段落): README の編集は見張りの段落で、その段落ではない
- 211 (c)(`pulseOrg` の読めない status ファイル): 変えていない。AC12 のテストは古いファイルが読めることの確認で、読めないときの扱いではない
- 376(RC3): すでに RESOLVED で、Trigger は取り消し線のみ

## 確かめたこと

- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 本が lock-step、`./scripts/check-template-purity.sh` は PASS。コミットの前に 3 つとも打った
- `/org` skill を読むテストを回した。`go test ./internal/org/ -run 'Skill|SendDefaults|ReportNames|CodeBlockAfter|ReadRepoFile' -count=1` は ok、`go test ./internal/cli/ -run 'RetiredStateDir|Docs' -count=1` は ok、`go test ./internal/scaffold/ ./internal/upgrade/ -count=1` は ok
- natural-japanese の lint を `/org` skill に回した。足した 617〜650 行と、変えた 367・370 行は info だけで、warn 以上はない
- 変えたファイル 13 個に U+2028・U+2029・U+FFFD は 0 件。self-review の報告は、バイトでも `LC_ALL=C grep -c -F` で U+2028(E2 80 A8)・U+2029(E2 80 A9)・U+FFFD(EF BF BD)が 0 件になった。V7 は Edit tool ではなく python で直した。Edit の `old_string` に U+2028 の文字を打つ必要があり、取り違えると 6 文字のテキストに戻らないため。1 行 1 追加 1 削除(`git diff --numstat` が 1/1)で、verdict と severity と `Merge:` の行は変わっていない
- tech-debt の plan 参照(`docs/plans/active|archive/...`)は、`verify.local.sh` の検査と同じ式で、無い参照が 0 件
- `/org` skill の文面(`templates/base/` の 2 つの写しを含む)を変えたので、`ralph upgrade` が配る文面が変わる。挙動は CLI が持つので、CLI の更新だけで効く。下流の skill の文面は `ralph upgrade` でしか新しくならない

## 残っていること(lead へ)

- 払っていないもの: 引き継ぎの注を足した行(357・303・211 (a)・353)の項目、M30・M50・M48 のテスト、N5 の doc の直し。どれもコードの変更になる。コードを変えるなら、パイプラインを回し直すかは lead が決める
- `/director` skill の 967 行の ALERT の件の項は、件に回数が付くことを書かない。#307b で直す(計画の Non-goals、verify の I3)
- 計画の Rollout の文(「skill は変えないので `ralph upgrade` は要らない」)は、本文を変えない指示のため直していない。Progress の項に、文面の更新には `ralph upgrade` が要ることを書いた。`/pr` の本文に release のノートの項として入れるかは lead が決める
- `docs/specs/2026-11-07-org-seat-resume.md` の 32 行と 166 行は、stall の条件を「出力の文字列が変わらないこと」だけで書く。待っている leader の stall を立てない規則は足されたが、その spec は座席の API エラーの話で、完了した記録なので変えていない
- 実機での件の数え直しは、verify と test のとおり未実施のまま

## Insight event

```
./scripts/insights-append.sh --slug watch-alert-dedupe --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-11-watch-alert-dedupe.jsonl` が 5 行から 6 行になった。
