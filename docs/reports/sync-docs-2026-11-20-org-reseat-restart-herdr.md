# sync-docs report: org-reseat-restart-herdr

- Date: 2026-11-20(パイプラインの cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`(承認の digest `f5d55e0b6ca2`。この run の前後とも同じ)。issue #354
- Branch: `feat/org-reseat-restart-herdr`。run 開始時の HEAD は `b7c41aa3`(origin/main の `29dd00fd`、#358 を取り込んだ merge)
- 先行 report: self-review(run 1〜3、すべて Merge: yes)、verify `b7cd2637`(pass、LOW V1〜V7、INFO I1〜I7)、test `445452d2`(pass、LOW の穴 1 はテストだけの `261df0c9` で塞いだ)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: verify のあとのコードの commit は 3 つ。`562e1ab5` は立て直したあとにサーバーの `CLAUDE_CONFIG_DIR` をもう 1 度読んで `herdr server:` の行を出し、違えば `note:` を出す変更で、サーバーの知らせの helper を `internal/cli/org_reseat_herdr_server.go` に移し、V4・I2・I5 のコメントを直した。`b1fd40e8` はその `note:` の文の直し。`261df0c9` はテストだけ。文書の側は、verify の V1〜V3・V5〜V7、I1・I3・I4・I7 を扱った
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go のファイルを変えたのは `5799b25c` だけで、`ralph org reseat --help` の 1 文とその pin。計画の本文(digest に入る節)は変えず、AC の印と Progress だけを直した
- 差分の大きさ(`/pr` が walkthrough を決める材料。この run の report と insights の commit の前の数): `git diff 29dd00fd...HEAD` は 39 ファイル、+5,149/-167(計画、report、insights を含む)。Go のテストでないコードは +1,198/-62、Go のテストは +2,675/-4。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify の文書の指摘(V1〜V3、V5〜V7)と INFO の I1・I3・I4・I7 を直した。内容は、仕様の FR-3 の注と印、org runtime の仕様と multi-org director の仕様の注、`/org` skill のコマンド表の `reseat` の行と前提の文、2 つの skill の窓の文と立て直したあとの `herdr server:` の行、README の reseat の段落、quality-gates の 2 つの写し、`ralph org reseat --help` の 1 文、tech-debt の持ち越しの注 12 行と窓の行の直し、計画の AC の印と Progress である。

直さなかったものは 2 つある。計画の本文にある古い関数名(I3)と「1 秒未満」(I1 の計画の側)は、本文が承認の digest に入るので Progress に書いた。AGENTS.md は変えていない(下の項)。

自分の掃引で、依頼の一覧にない点を 2 つ扱った。(1) tech-debt の 118 行(template の quality-gates に触れるたびに当たる)と 182 行の (a)(コマンド表の次の編集)は、この pass 自身の編集で当たったので注を足した。(2) 470 行(director skill の次の編集)は、行が RESOLVED で Trigger だけが残っているので、注を足さなかった。窓の文の「`reseat.lock` を取るのは reseat だけ」は、`takeReseatLock` の呼び出しが `Reseat` の 1 か所だけであることを `git grep` で確かめてから書いた。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `internal/cli/org_reseat.go`、`internal/cli/org_reseat_restart_test.go` | help に、`--restart-herdr` がサーバーを起動し直した本当の reseat が、読み直した `CLAUDE_CONFIG_DIR` の行をもう 1 度出し、違えば `note:` が続くことを 1 文足し、`TestOrgReseatHelp_SaysHowToRestartTheHerdrServer` に 1 つ pin を足した(`fix:`) | `5799b25c` |
| `README.md` | reseat の段落: 2 回目の pane の一覧で断ったとき座席が止まったままであること、10 秒の待ち、窓、立て直したあとの `herdr server:` の行と `note:` | `257a30ff` |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | 「Herdr server gate」の行(2 つの写しは同じ文): `--claude-config-dir` の例外、`--restart-herdr` が 2 度目に打つゲート、その失敗は座席を止めたまま残して打ち直しを出すこと | `257a30ff` |
| `docs/specs/2026-11-20-org-reseat-restart-herdr.md` | FR-3 に日付つきの注(2 回目の確かめで断っても `reseated` は記録する)、`- 実装:` の行、FR・NFR・AC の 20 項目に `[x]`(`git diff --numstat` は 22/20) | `257a30ff` |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-20 改訂(feat/org-reseat-restart-herdr)の (a)〜(d) と本文の印の注、2026-11-12 改訂 (d) の末尾に印 | `257a30ff` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-9 に 2026-11-20 の項(director が人に頼まれたときだけ `--restart-herdr` つきの reseat を打ってよい) | `257a30ff` |
| `.claude/skills/org/SKILL.md` とその 3 つの写し | 前提の環境の文に `--claude-config-dir` の例外、コマンド表の `reseat` の行に 2 つの flag、窓の文に同じ repository の `start`・`spawn`、立て直したあとの行と `note:` | `f4b032bb` |
| `.claude/skills/director/SKILL.md` とその 3 つの写し | 3 の窓の文(同じ repository の別の shell の `start`・`spawn`、`reseat.lock`)、立て直したあとの行と `note:` を人に伝え、違えば座席の画面で確かめてもらうこと | `f4b032bb` |
| `docs/tech-debt/README.md` | 持ち越しの注 12 行と Related(計画の active のパス)、窓の行(545)の 2 か所 | `34a84334` |
| `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md` | AC の 10 項目に `[x]`、`Test artifact created` に `[x]` と 3 つの sub-bullet。`git diff --numstat` は 14/11 で、digest は `f5d55e0b6ca2` のまま | この report の commit |
| `docs/reports/sync-docs-2026-11-20-org-reseat-restart-herdr.md` | この report | この report の commit |
| `docs/insights/events/2026-11-20-org-reseat-restart-herdr.jsonl` | `sync_docs` の行(cycle 1、pass)。同じファイルに self_review(3 行)、verify、test の行が既にあり、欠けた phase はない | この report の commit |

skill の 4 つの写しは、`.claude` を編集し、`./scripts/sync-skills.sh` で `.agents/skills` を作り、`templates/base/.claude/skills` と `templates/base/.agents/skills` へ `cp` した(`sync-skills.sh` は templates を作らない)。6 組すべてを `cmp` で確かめた。

## 依頼の項目ごと

| 項目 | したこと |
|------|----------|
| V1 | FR-3 に、`finish`(`internal/org/reseat.go`)が `reseated` を記録すること、前の仕様の FR-8 と AC4 のテストに合うこと、括弧書きはこの注のとおりに読むことを足した。括弧書き自体は変えていない(FR-1 の順の注と同じ形) |
| V2 | コマンド表の行に `--restart-herdr` と `--claude-config-dir` の文を足した。表の行は 2,858 → 3,111 文字(`stop` と `disband` は 1,208 と 1,219 のまま)。節の stderr の行の項と director skill の 3 に、立て直したあとにもう 1 度行を出すことと、違うときの `note:`、人に伝えること、違うときは座席の画面で確かめてもらうことを足した。director skill の 2 と 4 の pin された文は触っていない |
| V3 | quality-gates の 2 つの写し(行は同じ文のまま)と、`/org` skill の前提の環境の文を直した。ゲートの失敗の列は、`--restart-herdr` が止めたあとのゲートの失敗では座席が止まったままになることを例外として書いた |
| V5 | README の「nothing stopped」を、1 回目の確かめで断るときだけの文にし、2 回目の一覧は別の文にした(座席は止まったまま、サーバーは動いたまま、`reseated` が 1 件、打ち直しを出す)。help の「Before it stops anything」の段落は立て直したあとの行を書いていなかったので、help に 1 文足した(`5799b25c`)。pin は 1 つ(行が入った help を `strings.Fields` でつないで比べる) |
| V6 | org runtime の仕様に 2026-11-20 改訂を足し、2026-11-12 改訂 (d) の末尾に印を付けた(2026-11-15 の印と同じ形)。multi-org director の仕様には、FR-9 の最後の日付つきの項の次に 2026-11-20 の項を足した。director の権限は仕様の本文に「しないこと」の節がないので、FR-9 の項にした |
| V7 | 下の「register の Trigger の走査」 |
| I1 | 窓の行(545)の 1 つめのセルに、同じ repository の別の shell の `ralph org start` と `spawn`(`reseat.lock` を取るのは reseat だけ)と、計画が書く「1 秒未満」は測っておらず、この行の書き方を取ること、2 つめのセルに「最後の一覧と停止のあいだに別の呼び出しは走らない」を足した。skill の 2 か所(4 写しずつ)にも同じ repository を足した。計画の本文は変えていない(下の「残したこと」) |
| I3 | 計画の本文は変えず、Progress に訂正の sub-bullet を足した(Scope と S2 のずれ (5) は S2 のときの形で、`reseatRetryCommand` は 53c12663 が消し、今は `reseatRetryCommandWithHerdr` だけ) |
| I4 | 計画の AC1〜AC8(AC1b、AC4b を含む)に `[x]`、仕様の FR・NFR・AC に `[x]`。仕様の Open questions は箇条書きでチェックボックスではないので、そのまま。`./scripts/plan-visual.sh digest` は `f5d55e0b6ca2` |
| I7 | `org_reseat.go` の大きさを書いた register の文は、520 行の Trigger の注だけ(下の 520 行)。その注に 554 → 793(`02402849`、verify が数えた)→ 695(`5799b25c`)と、helper の移動は `PlanReseat` を `reseat.go` から出す分割ではないことを書いた。「800 行に近い」と書いた register の文は、ほかになかった |
| Progress | `Test artifact created` に test の commit と、verify のあとのコード 3 つ(562e1ab5、b1fd40e8、261df0c9)、この pass の commit、訂正 2 点を書いた |
| AGENTS.md | 変えていない。`internal/cli/` の行は subcommand の名前だけを挙げ、reseat のファイルは挙げない。`internal/org/` の行の reseat の句は `--restart-herdr` と `--claude-config-dir` を既に書く。`org_reseat_herdr_server.go` を足す場所がなかった |

## register の Trigger の走査

`docs/tech-debt/README.md` の 5 列の表を ` | ` で分け、Trigger の列から取り消し線の文を除いて、branch が変えたファイルと関数の名前、この pass が編集するファイル(README、skill、quality-gates、help)で照らした。verify の行番号は main の merge で 518〜520 が 519〜521 に、539 が 545 に動いたので、行の最初の語で探した。

注を足した行(行番号は `34a84334` の値):

| 行(先頭の語) | 当たった Trigger | 注に書いたこと |
|------|------|------|
| 519(Fourteen small findings of docs/spec-org-reseat) | (a)(d)(f)(j)(l)(m)(n) | 払わなかった理由は、計画の Scope が立て直しの確かめと flag であること。(a) は status を読む関数が 3 つ目と、知らせの中の 4 つ目の読みで増えたこと。help の (f)(j)(m) は変わらず、新しい文は 5799b25c で足した |
| 520(`reseat.go` (1,307 lines) and ...) | `reseat.go`、`reseat_test.go` | `reseat.go` 1,574 → 1,933、`reseat_test.go` 2,004(2 行)、`org_reseat.go` 554 → 793 → 695、新しいファイルの大きさ。helper の移動は Trigger の分割ではない。数は `5799b25c`(最後の Go の commit)で数えた |
| 521(Test gaps of docs/spec-org-reseat) | (c)(h)(i)(j)(m)(n) | (h) はコマンド表の行を直した(V2)が、skill と help を比べるテストはない。ほかは払わなかった |
| 441(`herdr.go` の大きさ) | pane 操作 | `herdr.go` 971 → 1,009、`herdr_test.go` 2,397 → 2,545 |
| 181(org-stop-all の F-7) | ドライバの API | `PaneListAll`・`ServerStop`・`Cwd`。`NewHerdrError` は今もテストのために export され、新しい fake のメソッドは呼ばない |
| 88(2)(batchable LOW) | `spawn_test.go`、`watch_test.go` | 6,408 → 6,448、4,463 → 4,472。`filepath.Join(dir, "receipts.jsonl")` の fixture はそのまま |
| 308(`newTestWatchRun`) | `watch_test.go` | `newTestWatchRun(` を含む行は 30 のまま(base と HEAD を `grep -c` で数えた) |
| 440(a)(herdr autostart) | `orgHerdrServerDropsEnv`、ゲートの環境 | 起動するサーバーの環境に `--claude-config-dir` の上書きを足した。`CODEX_` の名前はそのまま |
| 185(a)(org-stop-all の test gaps) | herdr の `get` の parser | `herdrPaneJSON` に `Cwd`、`PaneList` の読みを `herdrPaneListResult` に移した。壊れた envelope の case は新しい `PaneListAll` の分だけ |
| 323(director の生存の確かめ) | herdr のサーバーへの届き方 | ゲートの包み、`ServerStop`、socket への `lsof`。director の登録がサーバーを記録しないことは変わらない |
| 118(template の quality-gates) | 「次のタッチ」 | この pass が template の quality-gates の 1 行を直したので、日付つきの注。「Must pass in CI」と verify chain は触れていない |
| 182(a)(org skill のコマンド表) | 「次の編集」 | 表の行 2,858 → 3,111 文字。`stop` と `disband` は変わらない |
| 545(窓の行) | — | 直した(I1)。Trigger の注ではない |

当たったが注を足さなかった行: 470 行(`/director` skill の次の編集)は、1 つめのセルが取り消し線で「RESOLVED 2026-11-13 in docs/director-notice-handling」になっており、director skill は雛形の版の知らせを今は書いている(`git grep -c -i` が 7 件)。Trigger のセルに取り消し線がない行なので、足すなら Trigger に `Done in` を付けて閉じる方が合うが、この branch の範囲ではないので触れていない。

当たらなかった行(読んで外した): 193、202(`HerdrClient` の型のコメントではなく interface の中のメソッドの doc、2026-11-06 の前例と同じ)、203、267、281、324、362、366、369、370(`spawn.go` 3,122 行と `spawn_test.go` 6,448 行は、前の注の数に 300 を足した値の下で、失敗の経路には触れていない)、375、387、396、402、431、447(`TabCreate` は変わっていない)、448、467、486、522(`takeReseatLock` は変わっていない)、531、533、117(「restructure」ではなく 1 行の編集)、235、252、259、376(recipe は編集していない)、406、473。どれも、名前が出るのは過去の注の中の履歴(`reseat.go`、`herdr_test.go` などの数)か、Trigger が名指す関数がこの branch で変わっていない行だった。

`reseatRetryCommand` の名前は、519 行と 521 行の 2026-11-13 の注の中に、そのときの変更の記録として残る。今の関数を指す文ではないので、そのままにした。

## 検証

| 確かめたこと | 結果 |
|------|------|
| `gofmt -l internal/cli`、`go vet ./internal/cli/` | 出力なし |
| `go test -count=1 ./internal/cli/ -run 'Reseat\|Help'`(help の変更の後) | ok、102 s |
| `go test -count=1 ./internal/org/ -run 'Prompt\|Skill\|DocsMatch\|OrgSkill\|Director'`(skill の変更の後) | ok、125.7 s |
| `go test -count=1 ./internal/cli/ -run 'Agents\|Readme\|Docs\|Skill\|Director'` | ok、60.4 s |
| skill の 4 つの写し | org と director の 6 組すべて `cmp` で同じ |
| `./scripts/check-skill-sync.sh` | 14 skill が同じ |
| `./scripts/check-sync.sh` | DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6、PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照を含む) |
| `./scripts/plan-visual.sh digest` | `f5d55e0b6ca2`(計画の `- Approved:` の行と同じ) |
| tech-debt の編集 | HEAD と比べて、変わったのは 13 行(88、118、181、182、185、308、323、440、441、519、520、521、545)だけ。列は 5 のまま、バッククォートは列ごとに偶数、`file:line` の参照は足していない。545 行以外は追記だけ(削除と置換 0) |
| quality-gates の 2 つの行 | 2 つの写しで同じ文(`cmp` で行を比べた) |

回していないもの: `./scripts/run-test.sh`(依頼の指示)。help の変更(`5799b25c`)は Go のコードなので、パイプラインを回し直すかどうかは lead が決める(変えたのは help の定数の 1 文と pin の 1 項目で、ロジックは触っていない)。仕様に足した文のうち、立て直したあとの行の挙動(`HerdrRestarted` のときだけ出す、`note:` の条件)は、`org_reseat_herdr_server.go` と test report の変異 5a・5b で確かめた形に合わせた。座席のある org での通しと 2 つのアカウントでの切り替えは、実機では確かめていない(仕様の注と Known gaps に書いてある)。

## 残したこと

- 計画の本文: Scope の `reseatRetryCommand`(I3)と、Design decisions 3 と Risks の「1 秒未満」(I1)。本文は digest に入るので、Progress に書いた
- 470 行: 上のとおり、RESOLVED の行の Trigger に取り消し線がない
- tech-debt の 519 行などの help の項目((f)(j)(m))は、この branch では払わなかった
- `/pr` へ: PR の本文の Known gaps に、窓(長さは測っていない、同じ repository の `start`・`spawn` も入りうる)、Codex の計画助言の指摘 2 を文字どおりには入れなかったこと、座席のある org での通しと 2 つのアカウントでの切り替えを確かめていないことを書く

## Insight event

`./scripts/insights-append.sh --slug org-reseat-restart-herdr --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行を足した(exit 0。cycle は `cycle-count.json` から 1)。この report と同じ commit に入れる。`git status --short` は、既存の jsonl の `M` と計画の `M` だけを示した。
