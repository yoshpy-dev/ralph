# Verify report: reseat-leader-progress

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)。`- Approved:` の digest は `sha256:de0cca0b6f78`(今回は計算し直していない。承認の検査は `/implement` が済ませている)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は 1)
- Scope: `git diff main...HEAD`(HEAD `9748a702`、base `48a8a661`)。11 ファイル、2240 行追加、6 行削除。Go の本体は `reseat_progress.go`(新、412 行)、`feature.go`(+26)、`reseat.go`(+16)。`leader.md` は +89 行。テストは新規 3 ファイル(`reseat_progress_test.go` 589 行、`reseat_progress_start_test.go` 411 行、`prompts_leader_reseat_test.go` 438 行)。文書は仕様 1 ファイル、self-review の報告書、計画、insight event。`origin/main` との試し merge(`git merge-tree --write-tree`)は衝突なし
- Evidence: `docs/evidence/verify-2026-11-15-reseat-leader-progress.log`(gitignore の対象で commit に入らない)

## Spec compliance

仕様 D7・FR-9・FR-10・AC-13・AC-14 と計画の AC1〜AC9 を、本体とテストの file:line で読んだ。テストは実行していない(`/test` の担当)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 reseat で立て直した leader のタスクに節が入り、承認・gate・受信箱・commit・今回の立て直しの id と起動予定の座席が載る(仕様 AC-13) | Met | `reseat.go:1252` が `StartFeatureReseat{ID, Seats}` を渡し、`reseatSeatsAfterLeader`(`reseat.go:1290`)が `plan.Start` から leader を除く。`feature.go:293` が leader の spawn の前に節を足す。本体は `reseat_progress.go:127`(読み)、`:203`(節)、`:348`(`startProgressSection`)。テスト: `TestReseatProgress_ReseatNamesItselfAndTheSeatsAfterTheLeader`(4 行の完全一致)、`TestReseatProgress_StartFeatureTellsARestartedLeaderWhatTheOrgHolds` |
| AC2 `start --plan` の立て直しでも同じ節。経過の無い org では節なし。disband のあとの start は commit だけの節 | Met(結合は部分) | `StartFeature` 1 本の道なので reseat と `start --plan` で同じ(D1)。空の org: `TestReseatProgress_AnOrgWithNoProgressHasNoSection`、`..._StartFeatureAddsNothingForAnOrgWithNoProgress`(`featureLeaderTask` の完全一致)。disband のあと: `readReseatProgress` が `lastDisbandedIndexes` の後ろだけを読む(`reseat_progress.go:127-`)ことと、commit だけの節(`..._ASectionHoldsEachPartAlone`)は別々のテストで見える。`StartFeature` を通して disband→start→commit だけの節を確かめるテストはない(V7) |
| AC3 今の命の外(前の命の承認と gate、別の org の件、resolved、disband より前の件)は載らない | Met | `reseat_progress.go:127-` の `events[start:]` と `OrgID`・`State` の絞り、`time.Parse(RFC3339)` の比較。テスト: `..._OnlyTheCurrentLifeCounts`、`..._InboxItemsOfTheCurrentLifeThatAreNotSettled`(`+09:00` と `Z`、同じ秒、読めない時刻、逆向きの文字列順) |
| AC4 commit は新しい順の先頭 20 件と総数。git が読めないときは理由が 1 行載り start は続く | Met | `reseat_progress.go:393` が `git log --no-show-signature --max-count=20` と `rev-list --count` を 1 つの 10 秒の context(`:339`)で打つ。テスト: `..._MoreThanTwentyItemsAndCommitsAreCounted`、`..._ACommitReadFailureIsOneLineAndHidesTheCommits`、`..._StartFeatureCarriesOnWhenGitFails`、`..._StartFeatureStopsAGitThatDoesNotAnswer`、`..._CommitsIgnoreLogShowSignature`。読み取りの失敗の 1 行は、節が別の理由で在るときだけ入る(S2 の記録のとおり。V8) |
| AC5 受信箱の 1 行目と commit の件名は制御文字を置き換えて 120 字で切る。本文は fence の中で `## ` が節の境目にならない | Met | `reseatProgressText`(`reseat_progress.go:326`)が `PrintableInboxText` と U+2028・U+2029 を置き換え、120 字(`reseatProgressLineRunes`)で切る。全行が `- ` かラベルで始まり、`fenceFor(body)` で囲む(`:203-`)。テスト: `..._ValuesFromPeopleAreOneCleanLine`、`..._AHeadingOrBackticksInATextDoNotLeaveTheFence` |
| AC6 `leader.md` に 4 項目の規則があり、役割の prompt のテストが文を固定する(仕様 AC-14) | Met | `leader.md:150-237`。導入の段落と 5 項目(座席の確認 `:161`、計画 `:192`、gate `:210`、受信箱 `:225`、座席 `:229`)で、規則は `次の順で進めます。`(`:239`)の前にある。テスト: `prompts_leader_reseat_test.go` の 9 テスト(位置、data の扱い、節の行の頭、手順の番号、座席の確認、計画、gate、受信箱、座席) |
| AC9 gate が通ったあと座席を止める前に全座席を reseat した場合の手順が文で決まる | Met | gate の項目(`leader.md:210-224`): TASK を送り直さず、今回の立て直しの座席が `(active)` になるのを待ち(10 分で BLOCKED)、手順 13 で止め、HEAD と `git status --porcelain` を確かめ、`gh pr list --head <ブランチ> --state all` で PR の有無を見て、なければ手順 14、あれば手順 15。テスト: `TestRenderRolePrompt_Leader_ReseatRuleGateItem` |
| AC7 既存のテスト(`TestFeatureLeaderTask` の完全一致、前文の確かめ、reseat のテスト)が通る。経過の無い場合のタスクの文は今と同じ | 未確認(実行は `/test`) | 既存のテストファイルは 1 つも変わっていない(`git diff main...HEAD --stat` に既存の `_test.go` はない)。`featureLeaderTask` も変えていない(`feature.go:293` の `s.leader.Task += ...` は節が空でないときだけ) |
| AC8 仕様の FR-9・FR-10・AC-13・AC-14 に印、AC-12 は `[ ]` に戻り実機の限りの文が残る。AGENTS.md と skill の reseat の説明が節に合う | 一部 | 仕様の印と AC-12 は済み(`docs/specs/2026-11-12-org-reseat.md:80-81,106,110-111`)。AGENTS.md と skill の説明は計画どおり `/sync-docs` の担当で、今は合っていない(下の Documentation drift の V2〜V6) |
| 仕様 FR-9 の 5 項目(承認、gate、受信箱、commit、立て直し)をすべて持つ | Met | 承認 `計画の承認:`、gate `最後の gate:`(結果、head、base、branch、via、log)、受信箱 `片付いていない受信箱の件:`(ID、TYPE、TASK_ID、state、本文の 1 行目)、commit `branch の commit:`、立て直し `今回の立て直し:` と `前の立て直し:` |
| 仕様 FR-10 計画が承認済みなら書き直さない、gate が今の HEAD と同じで pass なら PR の手順へ、受信箱は答えを待つ、立て直された座席の TASK を送り直す | Met | `leader.md` の計画・gate・受信箱・座席の 4 項目と対応する。手順番号(2・3・4・5・8・9・10・12・13・14・15、「cross-review」の小節、「agmsg の受信箱の運用」の節)は `leader.md` の本文と合い、`ralph org status` の `(active)`(`internal/cli/org.go:2158`)、`ralph org read --seat`、`history.sh {{TEAM}} leader 50`(`leader.md:1372` と同じ 1 行)も実在する。追加の `{{...}}` は `ORG_ID`、`TEAM`、`AGMSG_SCRIPTS` の 3 つで、`prompts.go:122-133` が置き換える |

### 計画の記録 3 項目の確認

- 3 回目の self-review の推奨(N4: gate の例外を頭に 1 文、N5: 見つけた RESULT を TASK_ID で分ける)は、`291ac942` の変更が満たす。`leader.md:164-166` の頭の文は、gate の項目に当たるときは確認の残りをせず TASK を送らずに止める、と言い、`(active)` の分岐(`:167-`)より前にある。入力待ちの分岐(`:183-186`)は `gate-` なら手順 12、`xreview-` なら cross-review の小節、それ以外は手順 8。`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckItem` が頭の文の順と、TASK_ID の分岐の文を固定する。推奨は「項目の頭」だったが、実装は項目の 3 文目(状態コマンドと「leader 以外」の後)で、`(active)` の話より前という意味では同じ
- 計画の「最初の 20 件」(D4、AC4)と節の「新しい順に <n> 件」は矛盾しない。`git log` は新しい順に出すので、先頭の 20 行は新しい 20 件で、節の見出しはその向きを言うだけ。実装は `--max-count=20`(1 commit 1 行の `%h %s`)で、20 行と同じ。計画の語は「最初の」が向きを言わない点で曖昧だが、`71ddcb30` の記録が実装と合うことを書いていて、仕様の FR-9 は向きを書かない。直す文書はない
- 計画 D7 の `gh pr list --head <ブランチ>` に対し、規則と固定するテストは `--state all` を付けている。merge 済みの PR を見落とさない向きの変更で、S3 の記録に出ている。計画の本文との差は V8 に書く

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、Go の pack) | **FAIL**(rc=1) | gofmt ok、`go vet ./...` は出力なし(通過)。golangci-lint の staticcheck が 1 件: `internal/org/prompts_leader_reseat_test.go:94:5: QF1001: could apply De Morgan's law`(`if !(approval < rule && rule < steps)`)。V1。このほかの項目は通った(下の行) |
| 同スクリプト内の hook の shellcheck と `sh -n`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、branch の secret scan | PASS | `check-sync`: all files in sync。`check-skill-sync`: 14 skill が lock-step。secret scan: `48a8a661..9748a702` clean |
| `golangci-lint run ./internal/org/...`(単独の再実行) | 同じ 1 件 | 同じ行、同じ規則。再現する |
| `git diff --check main...HEAD` | PASS(rc=0) | |
| 変更ファイル 11 件の U+FFFD | 0 件 | |
| `git merge-tree --write-tree origin/main HEAD` | 衝突なし | `origin/main` は `48a8a661`(HEAD に入っている)から動いていない |

## Documentation drift

`/sync-docs` に渡す一覧。重い順。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:303` の最後の 2 文 | No(V2、MEDIUM) | 「reseat does not tell the leader what the org had done so far (the approved plan, finished slices, the gate result); the director or the human does, in a DECISION.」は、承認した計画と gate の結果については今は逆。立て直した leader のタスクの終わりに、ralph が台帳から作った節が入る。済んだスライスは節の commit の行から読む形になった。DECISION を打つのは、節が条件を満たさないとき(計画の digest が違う、gate の head が違う、など)の補いに変わる |
| `/org` skill の 4 つの写し(`.claude`、`.agents`、`templates/base/.claude`、`templates/base/.agents` の `skills/org/SKILL.md`、どれも `:1835-1838`) | No(V3、MEDIUM) | 「reseat はまだ、それまでの経過(承認した計画、済んだスライス、gate の結果)を leader に渡さない。立て直した leader は、雛形の『機能ごとの org』の手順 1(計画を書く)から始め直すことがあるので、director か人が、済んだ手順を DECISION で leader に伝える。」。4 つは `cmp` で同じなので、1 つ直して `scripts/sync-skills.sh` と `check-skill-sync.sh` に通す。`start` の行(`:370`)と「機能ごとの org」の leader のタスクの説明(`:796-820`)にも、節が入ることを 1 文足してよい(必須ではない) |
| `/director` skill の 4 つの写し(同じ 4 パスの `skills/director/SKILL.md`、どれも `:1564-1569`) | No(V4、MEDIUM) | 「leader も立て直したときは、reseat はまだ経過(…)を leader に渡さないので、leader は雛形の手順 1 から始め直すことがある。leader が済んだ手順をやり直し始めたら…人に伝え…」。前提が逆。director の手順は、leader が節を読んでも手順をやり直したときに限る形に直す |
| `docs/specs/2026-08-01-org-runtime.md:84`(2026-11-12 改訂 (d)) | No(V5、LOW) | 末尾の「reseat はまだ、それまでの経過(…)を leader に渡さない(仕様の 2 本目)」。この仕様は「本文の記述は書き換えない」(`:85`)で、改訂の節を日付ごとに足す作り。2026-11-15 の改訂の節を足して、(d) の末尾に改訂の印を付ける |
| `AGENTS.md:92` の `internal/org/` の行 | No(V6、LOW) | reseat の記述(`reseat.go`、`reseat.lock`、`reseated`)に、leader のタスクに足す経過の節(`reseat_progress.go`: 台帳・受信箱・branch から作り、`StartFeature` が leader の起動の前に足す)がない。計画の Verify plan が名指す項目 |
| `ralph org reseat --help`(`internal/cli/org_reseat.go:101-103`)と `ralph org start --help`(`internal/cli/org.go:1115-1124`) | Yes(INFO) | 「A leader it starts gets the current values of the four [reviewer] keys in its task」は、間違いではなく不足。節に触れるかは任意(FR-11 は要求していない)。触るなら `org_reseat_test.go` の help の pin も合わせる(tech-debt 行 513(f)、514) |
| `ralph org reseat` の終わりの文(`internal/cli/org_reseat.go:454`、`reseat sends the leader nothing, and the seats it started have no TASK`) | Yes | leader が対象に無いときの DECISION の注で、その場合は今も正しい。leader も立て直したときは DECISION を出さない(仕様 D5)ので、今回の変更と矛盾しない |
| `docs/specs/2026-11-12-org-reseat.md`(仕様自身) | ほぼ Yes(V9、LOW) | 印と AC-12 と Status の 2026-11-15 の段落は合う。(1) `:141` の影響の表が、2 本目の変更先を `feature.go` の `featureLeaderTask` と書くが、実装は `featureLeaderTask` を変えず `StartFeature` に足した(計画 D1)。`StartFeature` に直す。(2) `:7` の「2 本目は未着手」と `:36` の「経過を持たない」は日付の付いた過去の記述で、直さない。(3) `:15` が引く `docs/plans/archive/2026-11-15-reseat-leader-progress.md` は `/pr` が計画を移したあとにだけ在る(`reseat_progress.go` と `feature.go` と 2 つのテストのコメントも同じ名前を指す)。`/pr` の archive の名前が同じことを確かめる |
| `docs/tech-debt/README.md` | No(V10、LOW) | 下に分けて書く |

### tech-debt の Trigger(`/sync-docs` が「carried over again」を足す行)

表の 5 列目の区切りは ` | `(`\|` は除く)で分け、4 列目の Trigger を読んだ。

- 行 514(`reseat.go`・`reseat_test.go`・`startFeatureInput` の 800 行の目安): Trigger「The next change to `reseat.go`…」に当たる。`reseat.go` は 1,537 → 1,553 行(+16、`wc -l` を `main` と HEAD で比べた)、`feature.go` は 848 → 873 行(+25)。新しい 4 ファイルは 800 行未満(412、589、411、438)。払わない理由は前の回と同じ(move-only の分割は `/self-review` からやり直しで、計画の Scope に無い)
- 行 201 の (e)・(f)(`runOrgStartPlan`・`StartFeature`・`scriptFeatureWorktrees` の次の変更): `StartFeature` に当たる
- 行 365(`prompts_test.go` などの大きさ): Trigger「The next change to `Send` or to the leader template tests」に、新しい leader の役割 prompt のテスト `prompts_leader_reseat_test.go`(438 行、新規)が当たる。5 つのファイル(`verbs.go`、`verbs_test.go`、`prompts_test.go`、`spawn.go`、`spawn_test.go`)は変わっていない。`leader.md` は 1,391 → 1,480 行
- 行 426(role prompt の大きさ): Trigger「The next change that adds a section to `leader.md`」。この PR は `##` や `###` の見出しを足さず、「機能ごとの org」の導入に規則の段落と 5 項目を足した。見出しの字義では当たらないが、行 426 の直し方(任意の手順を別の埋め込みファイルに移す)の対象になる種類の追加で、`leader.md` は +89 行。carried over を足すのが安全
- 行 396(`resolvedOrClean` の呼び手): 2026-11-05 の前例(`StartFeatureParams` の新しい field `DriverFromRoleEntry` を hit と数えた)に従うと、新しい field `StartFeatureParams.Reseat` で当たる
- 行 515 の (h) と行 513 の (f): `/sync-docs` が `/org` skill の reseat の節、`/director` skill の「座席を立て直す」、`ralph org reseat --help` を直すと、その作業自身が当たる
- 行 516、377、384、367: 当たらない(`Disband` と `takeReseatLock`、手順 4 とスライスの手順、手順 14、gate の `wip:` の確認のどれも触っていない)。`leader.md` の機能ごとの org の手順は変えていない(規則を手順の前に置いただけ)

register に足す新しい項(いまは計画の Progress と self-review にしかない):

1. 1 回目の self-review の L5(`reseatProgressSection` の長さ、`inboxSummary` との二重、`NewReplacer` を毎回作ること)と L3 の残り(gate の行の `branch <名前>(<時刻>)`、`via <名前>(<時刻>)` の括弧)。Trigger は「次に `reseat_progress.go` か `leader.md` の規則を触るとき」
2. S3 の implementer が範囲外として挙げた 3 点: `gh pr list --state all` は merge されずに閉じた PR も「ある」と読む。PR を作ったあと RESULT を上げる前に止まった leader は、gate の分岐で手順 15 に進むので RESULT が上がらない。承認の無いコミット済みの計画から手順 2 に進むとき、計画助言が済んでいたか分からない
3. 実機の限り: 経過の節を読んだ leader を分離した herdr の実機で動かしていない(仕様 AC-12 を `[ ]` に戻した理由と同じ)。行 515 の (g)(実機の初回の通し)の Trigger に重ねて書ける
4. テストの穴: `StartFeature` を通した disband→start→commit だけの節(V7)

## Observational checks

- `leader.md` の規則が指す手順を、番号のついた手順の本文と突き合わせた。手順 3(reviewer を承認の前に立ててよい)、手順 4(計画の先頭 2 行の書き換え)、手順 10(`archive-plan.sh` で計画を archive に移す)、手順 12(gate、`HEAD:`・`BASE:` の控え)、手順 13(座席を止める)、手順 14(PR)、手順 15(後始末)は規則の言い方と合う
- gate の項目が手順 14 に渡す `HEAD:` と `BASE:` は「この行の head と base」と書かれ、手順 12 の控えの代わりになる。規則が `SYNC_DOCS:` と cross-review の値の控えを branch の report から読み、読めなければ理由を書くとしている点は、手順 14 の RESULT の書き方と矛盾しない
- 節の `今回の立て直し:` の行は reseat の id と、leader のあとに起動する座席だけを書く。leader は reseat が最初に立てるので、規則の座席の項目と gate の項目は「`(active)` になるまで待ち、10 分で BLOCKED」を共有する。`Seat` の spawn を leader がしない点は R6 と合う

## Coverage gaps

- テストの実行結果(`/test`)。この報告書はテストの本体と固定する文を読んだだけで、通ることは確かめていない
- 経過の節を受け取った leader が規則のとおり動くかは、役割の prompt の文とそのテストまでしか確かめられない。実機での通し(reseat した leader が節を読んで承認済みの計画を書き直さない、など)は無く、AC-12 の限りと同じ
- `StartFeature` を通した「disband のあとの start で commit だけの節」の結合テストは無い。部品(今の命の絞り、commit だけの節)は別々のテストが見ている
- 利用者の環境差: 偽の gpg で `log.showSignature=true` を確かめるテストは `isolateGitEnv` で利用者の設定を切っている。実際の署名つき commit の branch での読みは見ていない

## Findings

| ID | Severity | Area | Finding | Needs |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | static | `prompts_leader_reseat_test.go:94` の `if !(approval < rule && rule < steps)` が staticcheck QF1001(De Morgan)に当たり、`run-static-verify.sh` が rc=1。S3 の slice の verify コマンド(`gofmt`・`go vet`・`go test -run`)には `golangci-lint` がなく、`/verify` で初めて出た。`.github/workflows/verify.yml` には golangci-lint を入れる step がなく、runner に無ければ CI では skip される(未確認) | `if approval >= rule || rule >= steps {` に直す。挙動は同じ(整数の比較)。直したら `./scripts/run-static-verify.sh` を 1 回打ち直して rc=0 を確かめる |
| V2 | MEDIUM | doc | `README.md:303` の末尾が逆(上の表) | `/sync-docs` |
| V3 | MEDIUM | doc | `/org` skill 4 写しの `:1835-1838` が逆 | `/sync-docs` |
| V4 | MEDIUM | doc | `/director` skill 4 写しの `:1564-1569` が逆 | `/sync-docs` |
| V5 | LOW | doc | `docs/specs/2026-08-01-org-runtime.md:84` (d) の末尾 | `/sync-docs`(2026-11-15 改訂の節を足す) |
| V6 | LOW | doc | `AGENTS.md:92` に `reseat_progress.go` がない | `/sync-docs` |
| V7 | LOW | coverage | `StartFeature` を通した disband→start→commit だけの節のテストが無い | `/test` で変異を当てるか、tech-debt の Test gaps に書く |
| V8 | LOW | spec | 計画 D7 の `gh pr list --head <ブランチ>` に `--state all` を足している(規則とテストが固定)。AC4 の「理由が 1 行載る」は節が別の理由で在るときだけ(S2 の記録は D6・AC7 を根拠にする)。計画本文との差で、記録はある | 計画の本文は変えない。`/pr` の Known gaps か tech-debt に書く |
| V9 | LOW | doc | 仕様 `:141` の影響の表が `featureLeaderTask` を 2 本目の変更先と書く | `/sync-docs` |
| V10 | LOW | doc | tech-debt の carried-over(行 514、201、365、426、396)と新しい項 4 つ | `/sync-docs` |

## Verdict

打ち直し(2026-11-16、orchestrator の inline): V1 は e483b52a で直した(`if approval >= rule || rule >= steps {`、テストの 1 行)。`./scripts/run-static-verify.sh` を打ち直して rc=0、`==> All verifiers passed.`(log は `docs/evidence/verify-2026-11-16-reseat-leader-progress-rerun.log`、git の追跡外)。`go test ./internal/org/ -run 'RenderRolePrompt_Leader_Reseat'` も通る。残りは `/sync-docs` の文書の直し(V2〜V6、V9、V10)と、`/test` で見る AC7、tech-debt か Known gaps に書く V7・V8 なので、判定は partial-pass にする。下の行は打ち直しの前の判定で、そのまま残す。

- Verdict(打ち直しのあと): **partial-pass**

打ち直しの前:

- Verified: 計画の AC1・AC3・AC5・AC6・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 の本体とテストの対応、`leader.md` の規則が指す手順番号・コマンド・placeholder の実在、3 回目の self-review の推奨(N4・N5)を満たす直し、「最初の 20 件」と「新しい順に <n> 件」が矛盾しないこと、`check-sync`・`check-skill-sync`・`check-pipeline-sync`・`check-template-purity`・secret scan・`git diff --check`・gofmt・`go vet`、`origin/main` との試し merge
- Partially verified: AC2(disband のあとの start の結合、V7)、AC4(読み取りの失敗の行は節が別の理由で在るときだけ、V8)、AC8(仕様の印は済み、AGENTS.md と skill の説明は `/sync-docs` の担当で今は合っていない、V2〜V6)
- Not verified: AC7(既存のテストが通ること、`/test`)、節を読んだ leader の実機での動き、golangci-lint が CI で動くか
- Verdict: **fail**。静的検査が 1 件で落ちる(V1、staticcheck QF1001、テストの 1 行)。仕様の AC は読んだ範囲ですべて満たす。V1 を直して `run-static-verify.sh` が rc=0 になれば、残りは `/sync-docs` の文書の直し(V2〜V6、V9、V10)で、判定は partial-pass になる。insight event の `verdict` は `fail`
