# sync-docs report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: `docs/plans/active/2026-10-08-org-limits-reserve.md`
- Pipeline cycle: 4(ユーザーが上限を 4 に上げた回。`cycle-count.json` は 2 のままなので、この report と insight event に cycle 4 と書く)。差分は merge base `51855166` から、sync-docs 開始時の branch HEAD `1c8f18cd`(feat/org-limits-reserve)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-08-org-limits-reserve.md`(`83362823`。Merge 可、C4-1〜C4-4)、
  `docs/reports/verify-2026-10-08-org-limits-reserve.md`(`642e7291`。pass、V4-1〜V4-3)、
  `docs/reports/test-2026-10-08-org-limits-reserve.md`(`1c8f18cd`。pass、T4-1〜T4-9。足したテストは `69cb7d19`)
- cycle 3 のこの report は `736ce19d` にある。この回が上書きした。この report の節を指す箇所はほかにない(`git grep` で 0 件)

## Summary

文書で直せるものは直した。Go のコード、コメント、help の文字列には触れていない(この回の変更は文書の 6 ファイルだけ)。C4-1 / V4-1 は、ユーザーが 2026-10-09 に今の挙動を残す(「戻す」)と決めたので、その記録を plan と tech-debt に書いた。

- plan: 進捗の 161 行目に、(a)「pane だけを後回しにした disband の補償は予約を戻さない」が 83aec44e で解けたことを足し、162 行目を足した(C4-1 の決定と、窓がオペレーター次第の長さになること)。digest は `1a165903b5df` のまま
- `/org` skill(V4-2): 「例外が 1 つある」の段落(190〜198 行目)を、`disband` の最後の close が失敗したときは pane の経路でも workspace の経路でも予約を一緒に戻し、動いている座席の通常の `stop` は予約を解いていないので戻さない、に直した。4 面は同じ内容(sha256 `c96ae543…`)
- tech-debt(V4-3 と C4-1〜C4-4): 170、172、176、177、179 行目を HEAD に合わせ、180 行目を足した。180 行目は cycle 4 の LOW 4 件と、残す価値のある test の穴(T4-7、T4-8)を 1 行にまとめた

## Changes made

| File | Change |
|------|--------|
| `docs/plans/active/2026-10-08-org-limits-reserve.md`(161、162 行目) | 161 行目の (a) に「83aec44e で解決。`CloseDeferredSelfPane` が `reserveAgain` を呼ぶ」を足した。162 行目を足した。`disband --force` で自分の pane の close が失敗したあと、止まっている座席への `stop` の打ち直しも失敗すると座席と予約が戻ること、ユーザーが残すと決めたこと、`TestOrgCloseDeferredSelfPane_StopRetriedAfterForcedDisband_ReservationRestored` が固定すること、窓はリスクの節の数十秒ではなくオペレーターがかけた時間になること。AC15 の本文とリスクの節は digest の中なので変えていない |
| `.claude/skills/org/SKILL.md`(190〜198 行目)と、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | V4-2。段落を組み直した。座席と workspace を「動いている」に戻す文と、予約を戻す文を分け、予約は「`disband` の最後の close が失敗したとき、pane の経路でも workspace の経路でも」戻すとした。動いている座席の通常の `stop` は予約を解いていないので戻さない、と書いた。上限と重なりを見ない文以降は元のまま。`stop` の行(168)と `disband` の行(169)は変えていない。`disband` の行の「予約があればそれも戻して」は 83aec44e のあとの 2 つの経路に合っている。`scripts/sync-skills.sh` で `.agents` を作り、`templates/base` の 2 面は `cp` した |
| `docs/tech-debt/README.md` | 下の表 |
| `docs/insights/events/2026-10-09-org-limits-reserve.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 4) |
| `docs/reports/sync-docs-2026-10-08-org-limits-reserve.md` | この report(cycle 3 の版を上書き) |

### tech-debt の変更

| 行 | 変更 |
|----|------|
| 170 行目(S8 の補償の 4 件) | (c) C2-5 の `verbs.go` を 2,011 行(`wc -l`、cycle 4)にし、ブロックの一覧に `releasedReservation` と `startsOrg` を足した。Trigger の「次の編集で移す」が 83aec44e でも使われなかったことを書いた(a94c914f と 83aec44e の両方)。(a) の RESOLVED の括弧は、`reactivateSeat` と `reopenWorkspace` が「最新の記録が写しと同じとき」に書くのに対し、`reserveAgain` は「同じ `disbanded` が同じ添字にあり、その後ろに org の立ち上げがないとき」に書く、と分けた |
| 172 行目(補償は上限と重なりを見ない) | 見出しの「two gaps left, (b) and (d)」を「one gap left, (d)」にし、(b) は 83aec44e で直ったと書いた。(a) の RESOLVED の括弧を HEAD の条件に書き直した(`releasedReservation` が d と解いた予約を返す、`now[d] == before[d]`、`now[d+1:]` に `startsOrg` の立ち上げがない、`ActiveReservation(now, orgID) != nil` は走査に含まれる、固定するテストと mutation K4、K4b、K5、K5b、K1K2)。(b) は取り消し線にして RESOLVED を書いた(`disband deferred the pane`、K6、通常の `stop` の 2 ケース、K3)。(d) に pane の経路の窓のテストがないこと(T4-8)を足した。Risk の (b) の文、Why の「(b) to (d)」(「(c) and (d)」にした)、Trigger の (b) を直した。Related に cycle 4 の C4-2、T4-8、mutation を足した |
| 176 行目(test の穴) | (a) の RESOLVED の括弧を書き直した。a94c914f の `ActiveReservation(now, orgID) != nil` は 83aec44e が足した走査に含まれ、単独では結果を変えない(K1、等価)。今この窓を固定するのは走査(K4)で、`a spawn holds the manifest lock` は式と `startsOrg` の `scope_reserved` の枝の両方を外したときだけ落ちる(K1K2)。Related に cycle 4 の mutation を足した |
| 177 行目(cycle 2 の文言の LOW) | Why の「上限を 3 に上げた」に、4 に上げたことと cycle 4 が 4 回目であることを足した。Trigger の「a third pipeline run」の取り消しの注に、cycle 4(83aec44e)も `idempotentRespawn`、`Spawn` の doc、`orgWideLimitsHelp` を変えていないこと、cycle 3 と cycle 4 の sync-docs が文書だけを変えたことを足した |
| 179 行目(cycle 3 の LOW) | 見出しを「C3-1 to C3-3、verify V3-3、C3-4 と V3-1 は 83aec44e で直った」にした。(a) の規則の文に予約の場合を足した(同じ `disbanded` が同じ添字にあり、その後ろに立ち上げがない)。(c) に、`disband --help`、`closeDeferredSelf` の doc、`Disband` の doc が予約を挙げないことを足し(pane の経路も戻すようになった)、skill の段落は cycle 4 の sync-docs で直したことを書いた。(d) は取り消し線にして RESOLVED を書き(`releasedReservation`、`startsOrg`、改名したテスト)、残る分(skip のエラー文の「written to the manifest while the close waited」。台帳が消えた、または置き換えられた場合にも出る)を書いた。Impact の (d) も同じく取り消し線と残りにした。Why の「(d) is a design choice」を「what remains of (d) is error text」にし、上限を 4 に上げたことを足した。Trigger の (d) は、規則を決める節(Q1、Q2)を外し、skip のエラー文を直す提案だけを残した。Related の「Q1, Q2」を外した |
| 180 行目(新しい行) | cycle 4 の LOW と test の穴。(a) C4-1 / V4-1: pane の経路の doc の言い切りが、止まっている座席への `stop` の打ち直しで成り立たないこと。`disband --force` のあと、その `stop` の close が失敗すると座席と forced disband が解いた予約が戻る。補償を書けなかった `disband` のあとも同じ入力になる(読み)。ユーザーが残すと決めたこと、テスト名、仮の修正 Q4 で落ちること、窓がオペレーター次第の長さであること。(b) C4-2: `ActiveReservation(now, orgID) != nil` が走査に含まれ、K1 は等価。(c) C4-3: `reservationBeforeLastDisband` に本番の呼び出しがなく、そのテストが `releasedReservation` と逆の答えを固定すること。(d) C4-4: 再整形の漏れ、`that` の指す先、`compensateUnderLock` の「同じ呼び出しの append は `now` に入らない」前提、`startedAfter` への切り出し。(e) T4-7(古い ralph が解散した org を新しい ralph が workspace の中から disband して close が失敗すると、古い `disbanded` が解いた予約が戻る。probe だけ)と T4-8(pane の経路の窓のテストがない)。Trigger に、直す文言と、C4-1 の選択を変える場合の形を書いた |

166 行目(F-8)は変えていない。この回は skill の表のセル(`stop` の行、`disband` の行)に触れていないので、セルの長さ(`stop` 976 字、`disband` 1,142 字)は cycle 3 のままである(`git diff` が段落だけを示す)。

触った 170、172、176、177、179、180 行目に `[A-Za-z_./]*:\d+(?:-\d+)?` を当てて 0 件、列は 5 つ(`|` で分けて 7 片、HEAD の行と同じ)、各列のバッククォートと `~~` は偶数である。コードは関数名とテスト名で指し、`file:line` を使っていない。plan は `docs/plans/active/...` で指した(`/pr` の `archive-plan.sh` が書き換える)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `/org` skill 4 面 | 上の表のとおり直した。補償を書く箇所は `stop` の行、`disband` の行、「例外が 1 つある」の段落、その下の段落の 4 つ。`stop` の行は通常の `stop` が予約を戻さないままなので変えない。下の段落(台帳のロックの下で読み直す、新しい記録は残す)は予約の経路を分けない書き方なので変えない |
| `README.md`(124 行目) | `stop`、`disband`、`--reserve`、上限の説明で、補償を書いていない。`補償`、`compensat`、`戻` が当たらない。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md` の FR-3 | 「org を disband したら予約を解く」と「3 段目で決めたこと」は補償を書いていない(`補償`、`compensat`、`戻す` を `git grep` して、仕様で当たるのは「手戻り」「戻し方」だけ)。変更なし |
| `docs/quality/quality-gates.md`(77 行目)、`templates/base/docs/quality/quality-gates.md`(76 行目) | 83aec44e は補償だけを変えたので、「Spawn rejected, recorded in manifest」の扱いは cycle 3 のとおり(177 行目の (b))。変更なし |
| `AGENTS.md`(`internal/org/` の行) | `envelope.go` と `reserve.go` の分担で、83aec44e が変えた `verbs.go` の補償を書いていない。変更なし |
| `.claude/rules/`、`docs/recipes/`、`docs/architecture/`、`.ralph/`、`templates/base/docs/`、`templates/base/ralph.toml` | 補償と予約の復元の語が当たらない(`git grep` の 0 件)。変更なし |
| `internal/cli/org.go` の help、`closeDeferredSelf` と `Disband` の doc | コードの文字列とコメントなので直さず、179 行目の (c) に送った(`disband --help` が予約を挙げない点を足した) |
| plan の Progress checklist と承認の digest | 161、162 行目だけを変えた。`./scripts/plan-visual.sh digest` は `1a165903b5df` で、`- Approved:` の行(4 行目)と一致した |

## Found but left

- C4-1 の doc の言い切り、C4-2 の式、C4-3 の関数とテスト、C4-4 の comment と `startedAfter` は、Go のコードとコメントなので 180 行目の Trigger に直し方を書いて送った
- verify V4-1 の 3 つめ(任意)の、`/org` skill の `stop` の行に「`disband` のあとに打ち直した `stop` の close が失敗したときは予約も戻す」を足す案は入れなかった。段落が「動いている座席の通常の `stop`」と書いて範囲を限っており、この入力は 180 行目の (a) に書いたので、skill には足していない。足すなら 4 面を同時に直し、F-8 の行(166 行目)の `stop` のセルの長さも書き換える
- 規則の違い 1 つ(立て直したうえで `disband` し直した場合、workspace だけが開き直され、予約も動いている座席も戻らない)は、verify が「AC15 に反しない」と判断した。tech-debt には足していない(`org started again and disbanded again`、`_Stay` が今の挙動を固定している)
- self-review の「その他」の (3)(待つ間に同じ org の座席に `rejected` だけが足された入力では予約が戻る。読みによる推測でテストなし)は足さなかった。172 行目の (d) の窓と同じ性質で、行を増やす理由が弱い
- 177 行目の (e)(Related の cycle 1 の ID に `appendix, cycle 1:` を付ける)は、cycle 2 から同じ。ID は今も指す先が解ける

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | `PASS: all files in sync.`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/run-static-verify.sh`(plan、skill、tech-debt の編集のあと、report と insight の前) | rc 0。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` が通り、tech-debt README plan references OK、`gofmt: ok`、`0 issues.`、branch secret scan clean(`51855166..1c8f18cd`)。対象は full fallback(`scripts/ralph-config.sh` が差分にあるため) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-08-org-limits-reserve.md` | `1a165903b5df`(`- Approved:` の行と一致) |
| 4 面の `shasum -a 256` | 4 面とも `c96ae543aac0…` |
| `git diff --check`(作業ツリー、commit 前) | 出力なし |
| tech-debt の行の形(スクリプト) | 170、172、176、177、179、180 行目: 7 片、バッククォートと `~~` が偶数、`file:line` 0 件。HEAD の行も 7 片 |

commit のあとに走らせる `git diff --check 51855166...HEAD` の結果は、呼び出し元への報告に書く。この report の commit が入った HEAD でしか確かめられないためである。

## Not verified

- skill の新しい文は、`reserveAgain`、`releasedReservation`、`CloseDeferredSelfPane`、`Stop`、`Disband` を読み、verify の V4-1 と test の T4-1 の実行結果に照らして書いた。実際の herdr で補償を起こして、エラー文が書いたとおりに出ることは確かめていない(verify と test の cycle 4 も fake と stub による)
- 180 行目の (a) の「補償を書けなかった `disband` のあとの `stop` の打ち直しも同じ入力になる」は verify の読みで、実行していない。T4-7 は test report が使い捨てで作った probe の結果で、この回は再現していない(`docs/evidence/` の log は gitignore の対象で手元にだけある)
- 180 行目の「直す文言」は、self-review と verify の提案を写したもので、実装して試してはいない
- `ralph org stop --help` と `ralph org disband --help` は出していない
