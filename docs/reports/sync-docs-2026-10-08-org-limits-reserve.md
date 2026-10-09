# sync-docs report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: `docs/plans/active/2026-10-08-org-limits-reserve.md`
- Pipeline cycle: 3(ユーザーが上限を 3 に上げた回。`cycle-count.json` は 2 のままなので、この report と insight event に cycle 3 と書く)。差分は merge base `51855166` から、sync-docs 開始時の branch HEAD `104be58d`(feat/org-limits-reserve)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-08-org-limits-reserve.md`(`f19df087`。Merge 可、C3-1〜C3-4)、
  `docs/reports/verify-2026-10-08-org-limits-reserve.md`(`8b83abeb`。pass、V3-1〜V3-4)、
  `docs/reports/test-2026-10-08-org-limits-reserve.md`(`104be58d`。pass、T3-1〜T3-7。`RulesDiffer` と `ReservationAppendFails_NamesIt` のテストは `066bf282`)
- cycle 2 のこの report は `27fefc47` にある。この回が上書きした。この report の節を指す箇所はほかにない(`git grep` で 0 件)

## Summary

文書で直せるものは直した。コードの変更は a94c914f だけで、cross-review が今のコードを読むので、Go のコード、コメント、help の文字列には触れていない。

- V3-1: plan の Progress checklist に 1 行足した(159 行目)。3 つの補償の規則が「新しい記録」を同じ基準で決めていないこと、`disband` だけの打ち直しと `--reserve` なしの立て直しで何が起きるか、`RulesDiffer` のテストが現状を固定していること、打ち直した `disband` で解けること
- V3-2、C3-3 のうち skill の分: `/org` skill の `stop` の行、`disband` の行、「例外が 1 つある」の段落に、台帳に新しい記録があるとき、台帳を読み書きできないときは戻さず、エラーが手で閉じる herdr のコマンドを示すことを書いた。`disband` の行には予約も戻すことを足した。4 面は同じ内容(sha256 `64efef31…`)
- V3-4 と C3-1〜C3-4: tech-debt の 166、170、171、172、176、177 行目を更新し、179 行目を足した。直さなかった C3-1〜C3-4 と test の穴(T3-4〜T3-7)は 179 行目に 1 行でまとめた

Go の help と doc の「いつも戻す」文(C3-3 の `stop` / `disband` の help、`closeDeferredSelf` と `Disband` の doc)、`compensateUnderLock` の doc(C3-1)、`selfCompensation`・`add`・`errorFor` の doc(C3-2)、skip のエラー文(C3-4 の (b))は、直さずに 179 行目に送った。

commit は 1 つで、この report と同じ commit に入れた。

## Changes made

| File | Change |
|------|--------|
| `docs/plans/active/2026-10-08-org-limits-reserve.md`(159 行目) | V3-1: Progress checklist に 1 行足した。座席は最新の状態イベント、workspace はその id の最新の workspace イベント、予約は「今の予約がなく、最後の `disbanded` の前の予約が写しと同じか」で決める。`disband` だけが打ち直されると座席と workspace は戻り予約は戻らない。`--reserve` なしで立て直されると座席は戻らず古い予約が書き戻される。エラーは手で閉じる herdr のコマンドで終わり、打ち直した `disband` で解ける。AC15 の本文は digest の中なので変えていない。digest は `1a165903b5df` のまま |
| `.claude/skills/org/SKILL.md`(168、169、190〜202 行目)と、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | V3-2、C3-3(skill の分)。`stop` の行と `disband` の行の括弧に 1 文ずつ足した(「台帳に新しい記録があるとき、台帳を読み書きできないときは戻さず、エラーが手で閉じる herdr のコマンドを示す」)。`disband` の行は「予約があればそれも戻して」を足した(予約を戻すのは workspace の経路だけなので「あれば」を付けた)。「例外が 1 つある」の段落は、予約も戻すこと(workspace の close が失敗したとき)を 1 文に足し、段落の下に 5 行の段落を足した(台帳のロックの下で読み直してから書く、新しい記録があれば残して戻さない、ロックを取れないときと読み書きできないときも戻さない、どちらもエラーが手で閉じる herdr のコマンドで終わる)。詳細は表のセルでなく段落に置き、セルの伸びを抑えた(F-8)。`scripts/sync-skills.sh` で `.agents` を作り、`templates/base` の 2 面は `cp` した |
| `docs/tech-debt/README.md` | 下の表 |
| `docs/insights/events/2026-10-09-org-limits-reserve.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 3)。UTC の日付が 10-09 に変わったあとで、`/test` の cycle 3 の event と同じファイルに入る。この slug の event は 2026-10-08 と 2026-10-09 の 2 ファイルに分かれる |
| `docs/reports/sync-docs-2026-10-08-org-limits-reserve.md` | この report(cycle 2 の版を上書き) |

### tech-debt の変更

| 行 | 変更 |
|----|------|
| 166 行目(skill の表のセルが長い、F-8) | `stop` のセルは 919 字から 976 字(2,008 バイト)、`disband` のセルは 1,075 字から 1,142 字になった。数を書き換え、この PR で 1 節ずつ足したことを Impact に書いた。Related に verify の V3-2 を足した |
| 170 行目(S8 の補償の 4 件) | (a) C2-1 の「写しで決める」半分を取り消し線にし、a94c914f(`compensateUnderLock`)で直ったこと、`seat spawned again in another pane` のケース、mutation S1 で落ちることを書いた。残りは「`reactivateSeat` が 3 つの理由で `""` を返し、`errorFor` は 3 つとも『写しは何も要らなかった』として手で閉じるコマンドなしで元のエラーを返す」こと。(c) C2-5 の `verbs.go` を 1,951 行にし、ブロックの一覧に `compensateUnderLock` と `reserveAgain` を足した。Trigger は C2-1 の「台帳を読み直す」を取り消し線にして(a94c914f で完了)残りの直し方を書き、「次の編集で移す」が a94c914f で来たのに移さなかったことを書いた。Impact の (a) を残りに合わせた |
| 171 行目(S8 の補償の test の穴) | (b) 座席を立て直す test は a94c914f で入った(`seat spawned again in another pane`)。C2-2 の側(同じ pane id を持つ古い `stopped` の座席)は残る。mutation O1(T3-5)も同じ欠けであることを書いた。Impact と Trigger を C2-2 だけに合わせ、Related に test の T3-5 を足した |
| 172 行目(補償は上限と重なりを見ない) | (a) F-5 を取り消し線にし、a94c914f と、固定する 3 ケース、mutation G4 と G5 を書いた。見出しの「three gaps」は「two gaps left, (b) and (d)」にした。Risk の「(a) can swap a live org's reservation for an older one」、Why の「All four」(「(b) to (d)」にした)、Trigger の (a) を外した |
| 176 行目(test の穴) | (a) X18 を取り消し線にし、同じ判定が `ActiveReservation(now, orgID) != nil` になったこと、mutation G4 が `org started again with another reservation` と `a spawn holds the manifest lock` で落ちることを書いた。Impact、Why、Trigger の (a) も取り消し線にした。(b) はカバレッジの値を外して関数名だけにし、`reserveAgain` の追記の失敗は `TestOrgCloseDeferredSelfWorkspace_ReservationAppendFails_NamesIt` で通ることを書いた。(c) に、補償の `a spawn holds the manifest lock` も同じプロセスの goroutine であること(T3-7)を足した |
| 177 行目(cycle 2 の文言の LOW) | Trigger の「a third pipeline run on this plan」を取り消し線にし、来たこと(cycle 3)、a94c914f は `idempotentRespawn`、`Spawn` の doc、`orgWideLimitsHelp` を変えていないこと、この回の sync-docs は文書だけを変えたので (a)〜(d) は開いたままで、ほかの Trigger も残ることを書いた。Why の「cycle 2 was the last」に、上限を 3 に上げたことと cycle 3 が 3 回目であることを足した。Related に verify の V3-4 を足した |
| 179 行目(新しい行) | cycle 3 の LOW と test の穴。(a) C3-1: `compensateUnderLock` の doc の「appends only what is still missing」が、テストの固定する規則と合わないこと、ロックの外の書き込み(`spawned`、`spawn_step`、`spawn_failed`、`org_workspace_created`、`Stop` の `stopped` / `stop_failed`、`Disband` の `org_workspace_closed` / `disbanded`)が読み直しと append の間に入りうること。(b) C3-2: ロックと 2 回目の read の失敗も `failed` に積まれるのに、3 つの doc が append に限ること。(c) C3-3: 無条件に戻すと読める文(`stop` / `disband` の help、`closeDeferredSelf` と `Disband` の doc、AC15)。(d) C3-4 と V3-1: 規則の不揃い(`disband` だけの打ち直し、`--reserve` なしの立て直し)と、skip のエラー文の「while the close waited」。(e) T3-4〜T3-7。Trigger に、直す文言と、(d) は規則を先に決めること(mutation Q1、Q2 の形)を書いた |

164〜179 行目のうち触った 166、170〜172、176、177、179 行目に `[A-Za-z_./]*:\d+(?:-\d+)?` を当てて 0 件、列は 5 つ(HEAD の 5 列と同じ)、各列のバッククォートと `~~` は偶数である。新しい 179 行目と書き換えた箇所は、コードを関数名とテスト名で指し、`file:line` を使っていない。plan は `docs/plans/active/...` で指した(`/pr` の `archive-plan.sh` が書き換える)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `/org` skill 4 面 | 上の表のとおり直した。補償を書く箇所は `stop` の行、`disband` の行、「例外が 1 つある」の段落の 3 つだけ(`.claude/skills` の全 SKILL.md に `補償`、`戻` を grep した範囲の確認。当たるのは `org` skill だけ)。「全 org の上限と予約」節の最初の段落の「台帳のロックの下で判定する」は、spawn の判定の説明で、補償ではない |
| `README.md`(124 行目、247 行目) | `stop`、`disband`、`--reserve`、上限の説明で、補償を書いていない。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md` の FR-3 | 「org を disband したら予約を解く」と「3 段目で決めたこと」は、補償を書いていない。FR-2 の `stop` / `disband` の記述にも補償の語はない(`補償`、`戻す` を `git grep`)。変更なし |
| `docs/quality/quality-gates.md`(77 行目)、`templates/base/docs/quality/quality-gates.md`(76 行目) | a94c914f は補償だけを変えたので、「Spawn rejected, recorded in manifest」の扱いは cycle 2 のとおり(177 行目の (b))。変更なし |
| `AGENTS.md`(90 行目) | `internal/org/` の説明は `envelope.go` と `reserve.go` の分担で、a94c914f が変えた `verbs.go` の補償を書いていない。変更なし |
| `.claude/rules/`、`docs/recipes/`、`docs/architecture/`、`.ralph/`、`templates/base/docs/` | 補償の語が当たらない(`git grep` の 0 件)。変更なし |
| `templates/base/ralph.toml`、`internal/cli/org.go` の help | a94c914f が触れていない。help の「戻す」文は 179 行目の (c) に送った |
| plan の Progress checklist と承認の digest | 159 行目だけを足した。`./scripts/plan-visual.sh digest` は `1a165903b5df` で、`- Approved:` の行(4 行目)と一致した |

## Found but left

- `compensateUnderLock`、`selfCompensation`、`add`、`errorFor`、`closeDeferredSelf`、`Disband` の doc、`stop` / `disband` の help、`reserveAgain` の doc(C3-1〜C3-4)は、Go のコメントと文字列なので 179 行目の Trigger に直す文言を書いて送った
- plan の AC15 の本文(103 行目)は「補償で戻った org は予約も持ち直す」とだけ書き、skip の場合を持たない。digest の中なので変えず、159 行目で補った
- 規則を揃えるか(V3-1、C3-4)は判断が要る。179 行目の (d) に、mutation Q1、Q2 の形と、そのまま残す場合の doc の文言を書いた
- skill は `--reserve` なしの立て直しで古い予約が戻ることを書いていない。意図が決まっていない挙動なので、skill に足さなかった(179 行目の (d))
- 177 行目の (e)(Related の cycle 1 の ID に `appendix, cycle 1:` を付ける)は、cycle 2 から同じ。ID は今も指す先が解ける

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | `PASS: all files in sync.`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/run-static-verify.sh`(plan、skill、tech-debt の編集のあと、report と insight の前) | rc 0。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` が通り、tech-debt README plan references OK、`gofmt: ok`、`0 issues.`、branch secret scan clean(`51855166..104be58d`)。対象は full fallback(`scripts/ralph-config.sh` が差分にあるため) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-08-org-limits-reserve.md` | `1a165903b5df`(`- Approved:` の行と一致) |
| 4 面の `shasum -a 256` | 4 面とも `64efef313d72…` |
| `git diff --check`(作業ツリー、commit 前) | 出力なし |
| tech-debt の行の形(スクリプト) | 166、170〜179 行目: 列 5、バッククォートと `~~` が偶数、`file:line` 0 件 |

commit のあとに走らせる `git diff --check 51855166...HEAD` と `./scripts/secret-scan-branch.sh --strict` の結果は、呼び出し元への報告に書く。この report の commit が入った HEAD でしか確かめられないためである。

## Not verified

- skill の新しい文は、`errorFor`、`compensateUnderLock`、`reactivateSeat`、`reopenWorkspace`、`reserveAgain` を読んで書いた。実際の herdr で補償を起こして、エラー文が書いたとおりに出ることは確かめていない(verify と test の cycle 3 も、fake と stub による)
- 「予約を戻すのは workspace の経路だけ」は、`CloseDeferredSelfPane` が `reactivateSeat` だけを呼ぶことを読んだ結論で、172 行目の (b) と同じ。実行はしていない
- 179 行目の「直す文言」は、self-review と test の提案を写したもので、実装して試してはいない。T3-4 の probe は test report が使い捨てで作ったもので、この回は再現していない(`docs/evidence/` の log は gitignore の対象で手元にだけある)
- `ralph org stop --help` と `ralph org disband --help` は出していない
