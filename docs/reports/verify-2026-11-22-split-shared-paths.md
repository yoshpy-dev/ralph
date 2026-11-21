# Verify report: split-shared-paths

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 2 回目(`cycle-count.json` は cycle 2)。今の判定は、この節のあとの `## Verdict`。Run 1 は見出しを 1 段下げて、`---` の下にそのまま残す。所見の番号は Run 1 の続き(V9 から)
- Scope: `git diff origin/main...HEAD`(merge-base は origin/main 10e9322d、確かめた HEAD は e5dbabe8)。中心に見たのは Run 1 のあと(`77a6961a..e5dbabe8`)の変更: sync-docs の 5cdc7150(org skill のコマンド表の `spawn` の行と分割計画の例、tech-debt の注、Go のコメントの計画の指し方)、main の取り込み 809a887e(PR #368、issue #362)、cross-review の ACTION_REQUIRED の直し 2c0c28e6、self-review Run 2 の R2-1 と R2-2 の直し e5dbabe8。Go の変更はコメントと、skill を読むテストの文字列だけで、コードの動きは変わっていない。テストは打っていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-split-shared-paths.log` の「===== Run 2 (cycle 2)」の行から後(gitignore の対象。Run 1 の内容はその前に 1 バイトも変えずに残した。`head -c` と `cmp` で確かめた)。wrapper 自身の log は `docs/evidence/verify-2026-11-21-203122.log` と `docs/evidence/verify-2026-11-21-203742.log`

### Spec compliance (cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1〜AC5、AC7、AC8 | met(Run 1 のまま) | Run 1 のあとにこの branch が変えた Go のファイルは、テストを除くと 5cdc7150 の `internal/org/reserve.go` のコメント 1 か所(計画を path でなく slug で指す)だけ。main から来た Go の変更(`internal/org/permissions.go`、`internal/config/config.go`、doctor のファイルなど)は、下の「main の取り込み(cycle 2)」のとおり、この branch の関数に触れていない。merge のあとの木で go vet、golangci-lint、staticcheck が通った |
| AC6: org skill が共有のパスの衝突の直し方を書く(4 写し)。check-sync と check-skill-sync が通る | met | `### 共有のパスの直し方` の 1 は、生成されるファイルを手で merge せず、既定のブランチの版から作り直す手順のまま。2c0c28e6 と e5dbabe8 で、作り直す前にこの機能が lockfile の中だけで上げた版を書き留め、作り直したあとに 1 つずつ確かめ、戻った版があれば入れ直さずに `BLOCKED` で止まる小項目が加わった。計画の Scope(あとから merge する側が既定のブランチを取り込み、lockfile を作り直す。生成物は手で merge しない)の中にあり、Non-goals(ralph が自動で直さない)にも触れない。4 写しは `cmp` で同じ。`check-sync.sh` は `DRIFTED: 0`、`check-skill-sync.sh` は 14 skill が lock-step。`skill_shared_paths_test.go` がこの節に固定する 16 の断片を Python に移して当てた(下の Observational checks) |
| issue #363 の受け入れ条件 1〜3 | met | Run 1 のまま。3(直し方を skill に書く)は AC6 |
| cross-review の ACTION_REQUIRED 1(lockfile の中だけで上げた版が作り直しで黙って戻る) | 対応済み。直し方は triage の案と違う | triage の案は「同じ更新のコマンドで当て直し、残ったことを確かめる」。2c0c28e6 はそのとおりに書いたが、self-review Run 2 の R2-1 が pnpm 12.9.1 で、当て直しのコマンドが `package.json` の範囲も書き換えること、間接の依存では版つきの update が断られることを示した。e5dbabe8 は当て直しをやめ、書き留めた版を確かめて、戻った版があれば一覧を付けて `BLOCKED` で止まる形にした。版が黙って戻ることは確かめの小項目で防げる。入れ直すかどうかは escalate の先の人が決める。PR 本文にこの違いを書くとよい |
| self-review Run 2 の R2-1(MEDIUM)、R2-2(LOW) | 直った(e5dbabe8) | R2-1: 例は「間接の依存の版や、元のファイルの範囲を変えずに lockfile だけで上げた版」になり、当て直しの命令は消えた。止まる理由として、pnpm が間接の依存の版つきの `pnpm update` を断ることと、入れ直しが元のファイルを変えうることを書いた。R2-2: 1 は 6 つの小項目に分かれ、「控える」は「書き留める」に、`<先頭>` は `<取り込む前の先頭>` になった。古い語(`控えた版`、`版を控える`、`<先頭>`、`同じ更新のコマンド`)は、`docs/reports` と `docs/evidence` を除く repo に残っていない(`git grep`)。R2-3(LOW)は直していない。依頼のとおり `/sync-docs` が台帳に足す |

### main の取り込み(cycle 2)

- 両側が変えたファイルは 7 つ: org skill の 4 写し、`README.md`、`internal/cli/org.go`、`docs/tech-debt/README.md`
- 取り込みが branch に入れた変更(`c1d3628b..809a887e`)の `+`・`-` の行は、34 ファイルのすべてで main の変更(`9cd63d60..10e9322d`)と同じ並びだった(ファイルごとに変更行を hash して比べた)。この branch の変更(`9cd63d60..c1d3628b`)も、取り込みのあと(`10e9322d..809a887e`)で 43 ファイルのすべてが同じ。どちらの側の行も、落ちても書き換わってもいない。衝突の印(`<<<<<<<`、`=======`、`>>>>>>>`)は 4 ファイルとも 0
- 中身がぶつかるところもない。#362 が org skill に足したのは codex の座席の通信の項(HEAD の 2692 行あたり、`## permission 作法` の節)で、この branch が触った節(コマンド表、分割計画、承認、担当範囲の予約、org の終わり方、共有のパスの直し方、座席の立て直し)と重ならない。`internal/cli/org.go` は `withMainWorktreeDirectorStartConfig` の doc の 2 行、`README.md` は Codex の表の 1 行。tech-debt は末尾で、この branch の 3 行(547〜549 行)のあとに #362 の行(550 行)が並ぶ
- 手順 1 の「生成するコマンドが動かない(ネットワークのない sandbox など)」は、#362 で codex の座席が既定で外へ通信できるようになったあとも成り立つ。`[org.permissions].codex_network_access = false` の座席と、人の codex config に従う guarded の codex の座席は、通信なしでありうる
- #362 が足した台帳の行の Trigger を、この branch の変更と突き合わせた。当たるものはない(`[org.permissions]` の読み方、doctor のファイル、codex の recipe を、この branch は触っていない)

### Static analysis (cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope、HEAD e5dbabe8) | pass(`rc=0`、`All verifiers passed.`) | `Language packs selected: golang`。local verifier(shellcheck、`sh -n`、jq、check-sync は `IDENTICAL: 172`・`DRIFTED: 0`、check-pipeline-sync、check-skill-sync は 14 skill、check-template-purity、tech-debt README plan references)はすべて OK。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし |
| branch secret scan(上の中) | clean | `scanned 10e9322d..e5dbabe8 against origin/main` |
| `./scripts/run-static-verify.sh`(この report、計画の Progress、insight の行を working tree に入れて 2 回目) | pass(`rc=0`) | 同じ内容。tech-debt README plan references も OK。wrapper の log は `docs/evidence/verify-2026-11-21-203742.log` |
| `./scripts/plan-visual.sh digest <plan>` | `9046d5c8297e` | 承認の digest と同じ。下の V9 で Progress を直したあとも同じ(Progress は digest の外) |
| `cmp`(org skill と director skill、それぞれ 4 写し) | 同じ | |

### Documentation drift (cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` FR-4 の 2026-11-22 の項目(97 行) | yes | 直し方を「あとから merge する側が既定のブランチを取り込み、lockfile は手で merge せずに作り直す」と要約し、`/org` skill の「共有のパスの直し方」を指す。2c0c28e6 と e5dbabe8 のあとも、この要約は偽にならない。書き留め、確かめ、`BLOCKED` は要約にない(下の INFO) |
| 同じ仕様の FR-3 の 2026-11-22 の項目(83 行) | yes | 直し方には触れない |
| director skill の `### 共有のパス` の「衝突」の項 | yes | 「(lockfile は手で merge せずに作り直す)」と指し先だけを書く。4 写しは同じ |
| `internal/org/prompts/leader.md` の共有のパスの段落 | yes | implementer に衝突の解消を頼むとき、「`/org` skill に書かれた共有のパスの直し方に従うよう TASK に書きます」とだけ書く。座席の `BLOCKED` を `ralph org escalate` で上げる leader の決まりは、手順 1 の「leader はそれを escalate する」と合う |
| `README.md` の分割計画の段落 | yes | 指し先(the `/org` skill's 共有のパスの直し方)だけ |
| V6: org skill のコマンド表の `spawn` の行 | yes(5cdc7150 で直った) | `--reserve` の括弧に「走っている機能の org の共有のパスと重なっても拒否」が入った。4 写しとも |
| V7: org skill の分割計画の例 | yes(5cdc7150 で直った) | 座席の決まりが「予約と共有のパスの外のファイルは編集しない」になった。4 写しとも |
| V8: tech-debt の Trigger の引き継ぎの注 | yes(5cdc7150 で直った) | Run 1 の Coverage gaps が挙げた 13 行(1〜12 と、判断の要る 13)は、どれも 2026-11-22 の注を持つ。新しい 3 行(547〜549 行)は HEAD で成り立つ。548 行と 549 行がこの report の「Coverage gaps、INFO」と「V3」を引くので、Run 1 の節と番号は残した |
| cycle 2 の直し(2c0c28e6、e5dbabe8)が当たる Trigger | なし | 台帳の各行の Trigger のセル(表の 4 つ目)を、`共有のパスの直し方`、`skill_shared_paths`、`/org` skill、`org の終わり方`、`merge してよい順`、`lockfile`、`BLOCKED` で探した。当たった行の Trigger は、コマンド表、分割計画の例、leader.md の手順、または支払いの手順の中の `skill_shared_paths_test.go` で、この 2 コミットが触った手順 1 とその固定の断片には当たらない |
| V9(LOW、このコミットで直した): 計画の Progress | 直した | 「Review artifact created」の行は self-review Run 2 の R2-1(MEDIUM)を挙げたまま、e5dbabe8 で直ったことを書いていなかった。cross-review の直し 2c0c28e6 の記録も計画になかった。「/pr の前」の項は main の取り込みを先のこととして書いていた(809a887e で済んだ)。この 3 つを「Verification artifact created」の行に書いた |

### Observational checks (cycle 2)

- `TestOrgSkill_SharedPathsFixSaysTheLaterMergeRegeneratesGeneratedFiles` の最初の場所(`### 共有のパスの直し方`)の判定を Python に移した(scratchpad の `frag_port.py`。`directorSkillSubsection` と `skillOrderedFragmentProblems` の写しで、空白と改行を除いて順に探す)。テストの実行ではない。結果は 3 つ: (1) HEAD のテストの 16 断片は、HEAD の 4 写しのすべてにある。(2) HEAD のテストを e5dbabe8 の前(32de14d4)の skill に当てると、e5dbabe8 で足した 7 断片が欠ける。(3) 32de14d4 のテストの 14 断片は 32de14d4 の skill にある。移植が Go の判定と同じ向きに動くことは (3) で、新しい固定が直しを見分けることは (2) で確かめた
- 手順 1 は番号の項目 1 の中に `- ` の小項目を 3 文字下げで置き、続きの行は 5 文字下げ。CommonMark の入れ子のリストとして読める形だが、描画はしていない

### Coverage gaps (cycle 2)

- テストの実行(`/test` の担当)。Go のテストは、上の移植のほかは走らせていない
- e5dbabe8 は self-review Run 2(HEAD 2c0c28e6)のあとのコミットで、diff の質のレビューを受けていない。この verify は仕様への適合、文書のずれ、静的解析だけを見た
- 手順 1 の「pnpm は間接の依存に版を指定した `pnpm update` を断る」は、self-review Run 2 が pnpm 12.9.1 で確かめたもの。文は版を言わない。npm と yarn の lockfile は誰も確かめていない。「`go.sum` は版を `go.mod` が決めるので、書き留めるものはない」も読みだけで、確かめていない
- INFO: 手順 1 の止まり方が名指すのは、走っている org の座席(implementer が leader に `BLOCKED`、leader が escalate)だけ。disband 済みの org のブランチを直すのは人か、立て直した機能の org(director skill の regate の手順の 3。director 自身は直さない)なので、人が直すときは止まって自分で決めればよい。食い違いには数えない
- INFO: `BLOCKED` で止まるとき、途中の rebase か merge を残すか `--abort` するかを、手順は書かない。Run 1 の文(生成するコマンドが動かないときの `BLOCKED`)から変わっておらず、escalate を受けた人が決める
- INFO: 仕様 FR-4 と director skill の要約は、書き留め、確かめ、`BLOCKED` を言わない。要約としては偽でないので、ずれには数えない
- INFO: self-review Run 2 の「Tech debt identified (Run 2)」の台帳の行の案は、R2-1 と R2-2 が直ったので要らない。`/sync-docs` が足すのは R2-3 の行だけ

### Run 1 の所見の行き先

| ID | 内容 | 行き先 | HEAD で |
| --- | --- | --- | --- |
| V1〜V5 | コメント、doc、`--reserve` の usage、recipe、計画の記録 | 9535c7a7 | 成り立つ。9535c7a7 のあと `seat.go` と recipe は変わらず、`reserve.go` は 5cdc7150 のコメント、`internal/cli/org.go` は main の doc の 2 行だけ |
| V6 | コマンド表の `spawn` の行 | 5cdc7150 | 成り立つ(上の表) |
| V7 | 分割計画の例の座席の決まり | 5cdc7150 | 成り立つ(上の表) |
| V8 | tech-debt の Trigger の注 | 5cdc7150 | 成り立つ(上の表) |
| Run 1 の INFO 4 件 | `reserve.go` の行数、release ノート、skill の「行を変える」の言い方、status の表示 | 行数は台帳の 548 行、release ノートは計画の Progress の 129 行、status は台帳の 547 行。「行を変える」はずれに数えないまま | 成り立つ |

## Verdict

- Verdict: pass
- Verified: AC1〜AC8 と issue #363 の受け入れ条件 3 つ(Run 1 の静的な確かめに、cycle 2 の差分の読みを足した)。cross-review の ACTION_REQUIRED 1 と self-review Run 2 の R2-1・R2-2 が skill の 4 写しとテストの固定で直ったこと。手順 1 が計画の Scope と Non-goals の中にあること。仕様 FR-4、director skill、leader の prompt、README の要約が今の手順 1 と食い違わないこと。main の取り込み(809a887e)で両側の変更行がそのまま残り、#362 の変更とこの branch の変更がぶつからないこと。`./scripts/run-static-verify.sh` が 2 回とも rc=0(gofmt、go vet、golangci-lint 0 issues、staticcheck、check-sync `DRIFTED: 0`、check-skill-sync、secret scan clean)。計画の digest は `9046d5c8297e` のまま
- Partially verified: なし
- Not verified: テストの実行(`/test`)、e5dbabe8 の diff の質、手順 1 の pnpm 12.9.1 以外の版と npm・yarn の動き
- Findings (cycle 2): LOW 1 件(V9、計画の Progress の古い記録、このコミットで直した)、INFO 4 件。Run 1 の V6〜V8 は 5cdc7150 で直った
- Insight event: `./scripts/insights-append.sh --slug split-shared-paths --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行(cycle 2)を足し、このレポートと同じコミットに入れた

---

## Run 1 (cycle 1、履歴)

Run 1(HEAD 6168f606 と 9535c7a7)の記録。見出しを 1 段下げたほかは書いたときのまま。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff 9cd63d60...HEAD`(merge-base 9cd63d60 = main、確かめた HEAD は 6168f606)。コードの変更は S1 35458c51、S2 2bb50cd4、S3 89429da9、S3b c9e99c63、文書は S4 eb43b1b7、self-review の LOW の直しは 143deb8f と 6168f606。このあと /verify が見つけた文書のずれを 9535c7a7 で直した(コメント、help の文、recipe、計画の記録。動きは変えていない)。テストは打っていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-split-shared-paths.log`(gitignore の対象。wrapper 自身の log は `docs/evidence/verify-2026-11-21-174514.log` と `docs/evidence/verify-2026-11-21-175556.log`)

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: ヘッダの `- Shared: pnpm-lock.yaml, tsconfig.json` が `SplitPlan.Shared` に正規化して入る。2 行、`.`、絶対パス、`..`、機能の `- Reserve:` との重なりは理由を言って断る | met(静的) | `internal/org/split.go:505` の `splitHeader.read` がフェンスの外の `- Shared:` を 1 行だけ読み、2 行目を断る。`parseSplitShared`(`:536`)は空の値、空の項目、`NormalizeReservePaths` の断り(絶対パス、`..`、空白・カンマ・制御文字)、`.` を断る。機能の `- Reserve:` との重なりは `finishSplitPlan`(`:705`)が機能、両方の行、重なった組を名指して断る。ヘッダより後のフェンスの外の `- Shared:` は `:389` で断る(self-review の L3)。`TestLoadSplitPlan_Shared` と `TestLoadSplitPlan_SharedRejects`(`split_shared_test.go`)が AC1 の形を 1 つずつ持つ |
| AC2: `- Shared:` のない計画の、計画の digest と機能の digest が今と同じ | met(静的) | `featureDigest`(`split.go:747`)は `len(shared) > 0` のときだけ `shared` の欄を最後に足す。base の関数(9cd63d60 の `split.go:654-667`)は同じ 6 欄を同じ `%s %d\n%s\n` の形で hash する。base の形で書いた入力を `shasum -a 256` に通すと `1f1742b5c65c` で、`TestFeatureDigest_WithoutSharedIsUnchanged` が固定する値と同じ。計画の digest は `scripts/plan-visual.sh` と `PlanDigest` が変わっていないので同じ |
| AC3: 予約の Details の最後に `shared=` が入り、読み戻せる。`shared=` のない古い記録は空 | met(静的、計画からのずれは記録済み) | `scopeReservedDetails`(`reserve.go:382`)が binding の最後の token(note の前)に `shared=` を書き、`featureBindingFromTokens`(`:466`)が鍵の表に足して読む。値は `sharedFromToken`(`:507`)で正規化し、読めない値と `shared=` のない記録は `""`。共有のパスは計画の書く `Reservation.Shared []string` ではなく `Reservation.Feature.Shared`(文字列)と `SharedPaths()` に入る。理由は計画の Progress の S2 に書かれている(`SpawnParams.reservation()` が組み立てる、`FeatureBinding` を `==` で比べられるままにする)。AC3 の「`Shared` に読み戻す」はこの欄で満たす。テストは `TestScopeReservedDetails_SharedRoundTrip`、`_SharedTokenLast`、`_OlderBinaryReadsSixTokens`、`TestFeatureBinding_SharedRead`、`TestStartFeature_RecordsSharedPaths` |
| AC4: 共有のパスだけの変更に scope_change の ALERT が立たない。共有でも予約の内でもない変更は今のとおり | met(静的) | `watch.go:1355` が予約のパスと `r.Feature.SharedPaths()` を `evaluateReservedScope` に渡し、`watch_scope.go:445` が両方を合わせて「外」を決める。`scopePathInside` の規則と ALERT の文は変わっていない。テストは `TestWatch_SharedScope_ChangesToTheSharedPathsRaiseNothing`、`_AnAlertNamesOnlyThePathsOutsideTheReservationAndTheSharedPaths`、`_AnOrgWithoutSharedPathsAlertsOnThemAsBefore`。変異で落ちることは `/test` が確かめる |
| AC5: 共有のパスどうしでは断らず、予約どうしの重なりは今のとおり断る | met(静的) | `reservationOverlapErr`(`reserve.go:743`)の最初の判定は予約と予約で、エラーの文は base と同じ。共有と共有は比べない。`TestReservationDecision_Shared` の「the same shared paths」「a shared directory holding another org's shared directory」が通り、「paths that overlap a reservation」が今の文で断られる。`TestStartFeature_SharedAdmission` の「another split plan that shares the same paths starts alongside」も同じ |
| AC7: 共有のパスと走っている org の予約の重なり、予約と走っている org の共有のパスの重なりを、start の順の両方で断る(ファイルとディレクトリ) | met(静的) | `reservationOverlapErr` の 2 つ目(新しい側の共有と相手の予約)と 3 つ目(新しい側の予約と相手の共有)。呼び出しは manifest の lock の下の `reservationDecision`。`TestReservationDecision_Shared` がファイルとファイル、ファイルとディレクトリ、ディレクトリとディレクトリ、同じ prefix の別ディレクトリを両方の順で持ち、`TestStartFeature_SharedAdmission` が `StartFeature` と素の `Spawn --reserve` で両方の順を確かめる |
| AC8: leader のタスクの `- 共有のパス:` の行と、予約と共有を合わせた `--scope`。leader の prompt。reseat と `start --plan` の立て直しも同じ | met(静的) | `feature.go:513`(binding に `Shared`)、`:590-601`(`--scope` は予約のあとに共有、`- 共有のパス:` の行は共有があるときだけ)。`prompts/leader.md:130-136` が共有のパスも変えてよいと書く。`reseat.go:476` が非 leader の座席の `--scope` を同じ並びで作る。テストは `TestFeatureLeaderTask_SharedPaths`、`_SharedScopeIsOneShellWord`、`TestStartFeature_SharedPathsLineAtStartAndRestart`、`TestReseat_SharedPathsLineInTheLeadersTask`、`reseat_shared_test.go` の 3 つ、`TestRenderRolePrompt_Leader_FeatureOrgSharedPaths`。implementer と reviewer の prompt は `{{SCOPE}}` だけで範囲を区切るので、`--scope` に入った共有のパスがそのまま効く |
| AC6: director skill の `- Shared:` の書き方、org skill の衝突の直し方(各 4 写し)。check-sync と check-skill-sync が通る | met | director skill の `### 共有のパス`、org skill の `### 共有のパスの直し方` と `### 分割計画`・`### 担当範囲の予約`・`### org の終わり方` の 1・`### 座席の立て直し` の 4 の追記。4 写しは `cmp` で同じ。`check-sync.sh` は `DRIFTED: 0`、`check-skill-sync.sh` は 14 skill が lock-step。文は `skill_shared_paths_test.go` が固定する |
| issue #363 の受け入れ条件 1〜3 | met | 1 は AC4、3 は AC6。2(予約の重なりの判定は変わらない)は AC5 で満たす。計画は issue にない断り(AC7: 共有のパスと別の org の予約の重なり)を足した。codex の計画助言の 2 を受けた決定で、計画の Design decisions と Progress に書かれている |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope、HEAD 6168f606) | pass(`rc=0`、`All verifiers passed.`) | `Language packs selected: golang`。local verifier(shellcheck、`sh -n`、jq、check-sync `DRIFTED: 0`、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README plan references)はすべて OK |
| golang verifier(上の中) | pass | `gofmt: ok`、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし |
| branch secret scan(上の中) | clean | `scanned 9cd63d60..6168f606` |
| `./scripts/run-static-verify.sh`(9535c7a7 の直しを working tree に入れて 2 回目) | pass(`rc=0`) | 同じ内容。check-sync は `IDENTICAL: 172`、`DRIFTED: 0` |
| `./scripts/secret-scan-branch.sh`(9535c7a7 のコミットのあと) | clean | `scanned 9cd63d60..9535c7a7` |
| `./scripts/plan-visual.sh digest <plan>`(Progress を直したあと) | `9046d5c8297e` | 承認の digest と同じ。Progress の節は digest の外 |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と各写し | yes(V6、V7 を除く) | 4 写しは同じ。書かれたことを `split.go`、`reserve.go`、`feature.go`、`reseat.go`、`watch.go` と突き合わせ、偽の主張は見つからなかった。参照先の節(「分割計画の承認」の 4、「機能ごとの出力先」の 1 と 3、「org の終わり方」の 1、「merge してよい順」)はある |
| README.md | yes | 分割計画の段落に `- Shared:` の形、断る形、見張り、`--scope`、受け付けの両方の順、衝突の直し方の指し先。見張りの段落にも共有のパス |
| AGENTS.md | yes | `reserve.go`(共有のパスつきの予約)、`split.go`(`- Shared:`)、`watch_scope.go`(予約と共有のパスの外だけ)の 3 か所 |
| `docs/specs/2026-10-07-org-multi-org-director.md` | yes | 2026-11-22 の項目は FR-3(予約と見張りの扱い、83 行)と FR-4(分割計画の形、97 行)にあり、互いを指す。依頼の文と計画の記録は「FR-2」と書いていたが、正しくは FR-3 と FR-4(計画の記録は V5 で直した) |
| `docs/specs/2026-11-12-org-reseat.md` FR-6 | yes | 90 行の FR-6 に日付つきの追記。`reseat.go:476` と合う |
| `internal/org/prompts/leader.md` | yes | 「機能ごとの org」の段落。`featureSharedLead` の語と合う |
| `ralph org start --help`(`internal/cli/org.go`) | yes | 立て直しの条件と機能の digest の説明に共有のパス(L4 の直し) |
| `split.go` の形の doc | yes | ヘッダの `- Shared:`、ヘッダより後の行の断り、フェンス、digest |
| V1: `internal/org/seat.go` の `EventScopeReserved` のコメント | 直した(9535c7a7) | binding の token を `feature_digest=<hex>` までしか書いていなかった。`shared=<paths>` を足し「the last three」にした |
| V2: `FeatureBinding.FeatureDigest` の doc(`reserve.go:107`)と `orgFeatureJSON` の doc(`internal/cli/org.go:2069`) | 直した(9535c7a7) | 機能の digest を「前文と機能の節」と書いていた。`- Shared:` の行があれば共有のパスも入る、を足した |
| V3: `--reserve` の usage(`orgReserveFlagUsage`、`internal/cli/org.go:846`、spawn と start の共通) | 直した(9535c7a7) | 断る場合を「走っている別の org の予約と重なるとき」だけ書いていた。走っている機能の org の分割計画の共有のパスと重なるときも断られる、を足した。この文を固定するテストはない |
| V4: `docs/recipes/worktrees.md` と `templates/base/docs/recipes/worktrees.md` | 直した(9535c7a7) | 予約のある org の見張りは「予約の外のパスだけを読む」と書いていた。`start --plan` の org では共有のパスも内と数える、を足した。2 つは今も同じ |
| V5: 計画の Progress の self-review の直しの記録 | 直した(9535c7a7) | 「仕様の FR-2 の同じ日の項目」は FR-4(6168f606 が直したのは 97 行)。Progress は digest の外 |
| V6(LOW、直していない): org skill のコマンド表の `spawn` の行の `--reserve` の括弧 | no | 「走っている他の org の予約と重なれば拒否」のまま。正本の節 `### 担当範囲の予約` は直っている。1 行が長い表の行で、ほかのレーンも同じ行を触りうるので、`/sync-docs` に任せる。直すなら「、走っている機能の org の共有のパスと重なっても拒否」を足して 4 写しに配る |
| V7(LOW、直していない): org skill の分割計画の例 | no | 例のヘッダに `- Shared: go.mod, go.sum` が入ったが、同じ例の前文は「予約の外のファイルは編集しない」のまま。字面どおり読むと、宣言した共有のパスを座席に編集させない。tech-debt の RC4 の行(「RC4 (org recheck 2026-10-24)」で始まる行)が前から指摘している食い違いが、共有のパスにも広がった。直すなら前文の行を「予約と共有のパスの外のファイルは編集しない」にするか、RC4 の直し(「leader の雛形が名指すファイルだけ」)とまとめる。例を読むテスト `TestSkills_SharedLineExamplesParse` はこの行を見ない |
| V8(LOW): `docs/tech-debt/README.md` の Trigger | no | この PR が引いた Trigger に、引き継ぎの注がない(下の「Coverage gaps」の一覧)。self-review は register に足していない理由を書いているが、それは新しい行の話で、既存の行の引き継ぎの注には触れていない。`/sync-docs` か `/pr` で注を足す |

### Observational checks

- `scripts/plan-visual.sh digest` は Progress を直したあとも `9046d5c8297e` で、承認は保たれる
- AC2 の固定値 `1f1742b5c65c` を、base の符号化の形から独立に計算して一致を確かめた(上の表)
- 4 写しの skill と、root と template の recipe は `cmp` で同じ

### Coverage gaps

- テストの実行と変異の実行はしていない(`/test` の担当)。各 AC は、コードの読みとテストがあることまでを確かめた
- この PR が引いた tech-debt の Trigger(行は最初の語で示す。行番号は動く)。どれも払っていない:
  1. 「Two documentation findings of org-stop-all」(a): コマンド表の `watch` の行を直した(表の行全体で 3,378 → 3,410 文字)。`stop` と `disband` の行は変わらない(1,208 と 1,219)
  2. 「What a leader is told about its reservation and the org-wide limits」(a): leader の prompt を変えた(`leader.md` 1,626 → 1,634 行)
  3. 「Test gaps left by org-feature-worktree」(e)、(f): `StartFeature` の doc を変えた(`feature.go` 874 → 898 行)
  4. 「Small leftovers of org-plan-approval」(c): `featureBindingFromTokens` の鍵の表を変えた。読めない承認方式の binding を補償で書き戻すテストは足していない
  5. 「`internal/org/verbs.go` is 2277 lines」: leader の雛形の新しいテストのファイル `prompts_leader_shared_test.go`(64 行)。#312c-2 の前例と同じ数え方
  6. 「RC4 (org recheck 2026-10-24)」: org skill の分割計画の例と、leader.md の「機能ごとの org」の両方を変えた。V7 のとおり食い違いが広がった
  7. 「`resolvedOrClean` in `internal/org/statedir.go`」: 呼び手の `feature.go` と `split.go` を変えた
  8. 「Eleven gaps that fix/watch-alert-dedupe leaves」: `watch_scope.go` の `evaluateReservedScope` と `watch.go` の scope_change の呼び出しを変えた
  9. 「Two gaps of the feature digest of a restarted feature org」(b): `featureBindingFromTokens` に鍵 `shared` を足した。古い binary の自己終了の補償が binding を書き戻すと、`feature_digest=` と同じく `shared=` も落ち、その org の共有のパスは「なし」と読まれる(見張りが共有のパスの変更を知らせ、受け付けがそのパスを守らなくなる)。この行の (b) の確かめに `shared=` も足すとよい
  10. 「Test gaps of fix/split-feature-digest」(a) `featureDigestRefusal` を変えた、(b) `featureLeaderTask` のタスクを変えた、(d) org skill の「start が拒否するもの」の立て直しの段落を変えた。(d) の段落を 4 写しで固定するテストは足していない
  11. 「Five small findings of fix/split-feature-digest」(a)、(e) `sameFeature` の doc、`featureDigestRefusal`、`reservationDecision` を変えた。(b)、(c) `validateFeatureBinding` と `FeatureBinding.String()` を変えた。`featureDigestPattern` は今も `split_test.go:790` にある
  12. 「`internal/org/reseat.go` (1,307 lines)」: `reseat.go` 1,933 → 1,938 行、`reseat_test.go` 2,004 → 2,005 行
  13. 判断が要るもの: 「The role prompts grew with org-cross-review」は「節を足す変更」が Trigger。この PR は `leader.md` に段落を 1 つ足しただけだが、2026-11-16 の注は段落の追加も数えている
- 9535c7a7 の直しが新たに引いた Trigger はない(`orgFeatureJSON`、`seat.go`、`worktrees.md`、`orgReserveFlagUsage` を Trigger の列で探した)
- INFO: `internal/org/reserve.go` は 6168f606 で 821 行(base 686 行)になり、コードレビューの 800 行の目安を超えた。register にこのファイルの大きさの行はない。`split.go` は 777 行
- INFO: 既存の分割計画のうち、ヘッダより後のフェンスの外に `- Shared:` で始まる行を持つもの、またはヘッダの自由な文として `- Shared:` で始まる行を持つものは、この版から読み込みが断られる(前は本文か自由な文)。鍵は #363 で決まったので影響は小さいと読むが、計画の Rollout は revert の向きだけを書いている。release ノートに 1 行あるとよい
- INFO: director skill と org skill は「`- Shared:` の行を変える(足す、消すも)と…すべての機能の digest も変わる」と書く。同じパスを書き方だけ変えたとき(並びや `./` の有無)は、機能の digest は変わらず全体の digest だけが変わる(`TestFeatureDigest_Shared` の「the same shared paths written another way」)。「行を変える」を「共有のパスを変える」と読めば正しいので、ずれには数えない
- INFO: `ralph status` と `ralph org status` は共有のパスを出さない。self-review が tech-debt の行の案を書き、`/pr` で足すとしている

### Verdict

- Verified: AC1〜AC8 と issue #363 の受け入れ条件 3 つ(静的、コードとテストの読み)。AC2 の固定値を base の符号化から独立に計算して一致を確かめた。`./scripts/run-static-verify.sh` が 2 回とも rc=0(gofmt、go vet、golangci-lint 0 issues、staticcheck、check-sync `DRIFTED: 0`、check-skill-sync、secret scan clean)。skill の 4 写し、README、AGENTS.md、2 つの仕様、leader の prompt、`ralph org start --help` は実装と合う
- Partially verified: なし
- Not verified: テストの実行と変異の実行(`/test`)
- Findings: 直した LOW 5 件(V1〜V5、9535c7a7)、直していない LOW 3 件(V6、V7、V8)、INFO 4 件
- Verdict: pass
- Insight event: `./scripts/insights-append.sh --slug split-shared-paths --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行を足し、このレポートと同じコミットに入れた
