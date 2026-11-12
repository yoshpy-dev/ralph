# sync-docs report: gate-lock-accept

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-gate-lock-accept.md`(承認の digest `1d8736effdd9`)。issue #309
- Branch: `feat/gate-lock-accept`。run 開始時の HEAD は `7bebb360`(test report の commit)。origin/main `d2e5c6e9` は取り込み済み
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `73e95a07`(Merge: yes、LOW 8 件 L1〜L8)、verify `d5ef9a82`(pass、LOW 4 件 V1〜V4、INFO 4 件 I1〜I4)、test `7bebb360`(pass、生き残った LOW の変異 C1・C2・C8・D1)
- 見た範囲: この task の変更 `git diff d2e5c6e9...HEAD`(75 ファイル)と計画
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `1d8736effdd9`
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit の前の数): `git diff origin/main...HEAD` は 75 ファイル、+6,818/-518(計画、report、insights を含む)。docs とテストを除くと 38 ファイル、+2,735/-394。walkthrough は書いていない

## Summary

verify の V1〜V4、self-review の L3〜L5、test が挙げた 4 つの穴を直した。V1 は tech-debt の Trigger に当たった 24 行へ持ち越しの note を足した(verify が挙げた 20 行すべてと、掃引で見つけた 4 行)。V2 は README の `ralph org gate`、merge-order、merge の段落と動詞の表。V3 は `templates/base/ralph.toml` の auto_merge の注釈。V4 は `quality-gates.md` の 2 つの写し。L3 は skill の `--log` の例の quote。L5 は director skill と `gate accept --help` に `ci_failed` の 1 文。L4 と test の穴は tech-debt の行(468 行目の広げと、476 行目の新しい行)。

コードに触れたのは 3 か所で、どれも文だけである。`gate accept --help` の 1 段落、`OrgDirectorConfig` の comment、固定のテスト 2 本(help と director skill の例)。動きは変えていない。

## Changes made

| File | Change |
|------|--------|
| `README.md` | 動詞の表の `gate` に、直列(`gate.lock`)、負荷、`gate accept`、`accepted` を足した(131 行目)。`ralph org gate` の段落(286 行目)に、gate.lock の待ち、`max_load`、負荷の鍵、`gate accept` の記録と断りを足した。merge-order の段落(288 行目)に `accepted` の定義、class の順と並び、`failed` の注、受け入れが消える条件、JSON の `log` と `accepted`、local の regate の待ちを足した。merge の段落(290 行目)に `pr_merged` の `gate=accepted` と `GATE: accepted` の行を足した。下の「V2」 |
| `templates/base/ralph.toml` | auto_merge の注釈の「lists as ready」を「ready or accepted」に直した(3 行に折り返した)。下の「V3」 |
| `internal/config/config.go` | `OrgDirectorConfig` の comment の「merges ready PRs」を「ready or accepted PRs」に直した(comment だけ。行数は同じ)。下の「I1」 |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | Quality pipeline gate の行に、gate.lock と `max_load`、負荷の記録、`ralph org gate accept` と `accepted` の merge を同じ文で足した。2 つの写しの差は元から KNOWN_DIFF の 1 語だけで、足した文は同じ。下の「V4」 |
| `.claude/skills/director/SKILL.md` と 3 つの写し | `ralph org gate accept` の例の `--log` を `'<LOG: のパス>'` にし、手順 3 の文に「空白を含みうるので `'…'` で囲む」を足した。「gate の失敗の受け入れ」の冒頭に `ci_failed` の 1 文を足した。下の「L3」「L5」 |
| `.claude/skills/org/SKILL.md` と 3 つの写し | `gate accept` の行の例の `--log` を `'<LOG のパス>'` にし、`--log` の説明に「パスは空白を含みうるので `'…'` で囲む」を足した。下の「L3」 |
| `internal/org/director_skill_sync_test.go` | director skill の例の固定を新しい形にし、`ci_failed` の 1 文の固定を 1 つ足した(順は skill の順) |
| `internal/cli/org_gate_accept.go`、`internal/cli/org_gate_accept_test.go` | `gate accept --help` に `ci_failed` の段落を足し、`TestOrgGateAccept_Help` にその固定を足した。下の「L5」 |
| `docs/tech-debt/README.md` | 既存の 25 行を変えた(88、118、159、180、238、248、249、269、293、294、306、312、313、367、372、390、398、399、405、415、418、453、457、464 の 24 行は Trigger の末尾への note と Related への計画の path、468 は gate.lock の行の広げ)。新しい 1 行を末尾に足した(476 行目)。すべて挿入だけの編集。下の「tech-debt」 |
| `docs/plans/active/2026-11-12-gate-lock-accept.md` | `## Progress checklist` の artifact の 3 つの箱を `[x]` にし、sync-docs の 1 行を足した。「PR created」と AC の箱は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-12-gate-lock-accept.md` | この report |
| `docs/insights/events/2026-11-12-gate-lock-accept.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 |

skill の 4 つの写しは、`.claude/skills/` を直して `./scripts/sync-skills.sh`、templates へ `cp`、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` の順で作った。`md5 -q` は director skill の 4 つが `bc54b0e4…`、org skill の 4 つが `2b55d79c…` でそろった。

commit は 1 つ(`docs: sync docs for gate-lock-accept (cycle 1)`)。

## verify が挙げた drift

### V1: 条件に当たった tech-debt の行

verify が挙げた 20 行は、すべて Trigger の本文を読んで、当たりと判断した。453 は「`[org]` の検証の次の変更」で、`[org.gate].max_load` の検査が `validateLoaded` の同じ場所に入るので、当たりの端にあたる(Trigger の言う `[org]` が子の表を含むかは読み方による)。note は Trigger セルの末尾に足し(`. 2026-11-12: feat/gate-lock-accept hit this trigger with ..., but did not pay it: ... Carried over to the next change that meets this trigger`)、Related の末尾に `docs/plans/active/2026-11-12-gate-lock-accept.md` を足した。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない。

verify の一覧と食い違った所はない。ただし verify の一覧の番号は、行の頭の語で引き直した(下の表)。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 88「6 batchable LOW findings」(3)(4) | `watch.go`(`leaderActivityEventCount` の doc)、`status.go`(help の 1 段落) | (3) の `SEAT: ` の header、(4) の `printStateDirLine` と `omitempty` は変えていない。持ち越し |
| 118「`templates/base/docs/quality/quality-gates.md` の「Must pass in CI」」 | **この sync-docs 自身の編集**(V4)。`templates/base/docs/quality/quality-gates.md` の 1 行に文を足した | 「Must pass in CI」の節と verify の鎖は変えていない。verify はこの行を挙げていない(掃引で見つけた)。持ち越し |
| 159「org runtime FR-7 quality gate is prompt-instructed」 | 品質の道そのもの(gate の直列、負荷の記録、受け入れた失敗の merge) | 規則は文のまま(D3)。director は人に聞いてから打つ、は skill の文。verb が断るのは座席の pane だけ。reviewer が待つ間にすることは reviewer の雛形の文。持ち越し |
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の表の `gate`、`regate`、`merge-order`、`merge` の行と、新しい `gate accept` の行 | 表の行の長さ(2,163 → 3,268、5,220 → 5,919、4,091 → 4,705、5,045 → 5,285 字、新しい行 1,686 字)。`stop` と `disband` のセル(1,208 と 1,219 字)は変わらない。表の作り直しは計画の Scope の外。持ち越し |
| 238「Small findings of org-gate cycle 2」 | `gate.go`(`Gate`)、`gate_run.go`(`runGateScripts`、`runGateScript`)、`lockfile.go`(`waitHeldFileLock`、`heldFileLock` の doc) | (a)〜(d)、(f)〜(h) が名指す comment と関数、(i) の割り込みのテストは触っていない。持ち越し |
| 248「`ralph org merge-order --json` cannot tell ...」の Still open(外のコマンドの runner) | **掃引で見つけた**。`readGateLoad` が darwin で `/usr/sbin/sysctl -n vm.loadavg` を `exec.CommandContext` で走らせる | 7 つ目の runner になる(`WaitDelay` なし、stderr を error に足さない、定数は `gateLoadReadTimeout`)。runner をまとめるのは Scope の外。持ち越し |
| 249「Small leftovers of the org-merge-order PR」 | `mergeorder.go`(`collectMergeTargets`、`classifyMergeTarget`、`classifyAccepted`、`mergeGateFailed`、`mergeRegateFailedDetail`、`mergeOrderListNotes`)と `org_merge_order.go` の help | 文字の項目は Scope の外。(o) を広げる: 複数 ready の note は `ready` と `accepted` の数になったが、`--org-id` なしの「run ralph org gate again」のまま。持ち越し |
| 269「Two mutants of the org-director-start PR」(b) | `director_skill_sync_test.go` に足したテスト | (b) が言う `- 設定:` の行と始める step 3 は変えていない(1223 → 1307 行)。持ち越し |
| 293「`regate_test.go` and `regate_merge_test.go` are over the 800-line file size」 | `regate_test.go` 1596 → 1771 行、`regate_merge_test.go` 2105 → 2112 行 | gate.lock のテスト 4 本を足し、時計の読み取りの閾値を 2 から 3 に。移動だけの分割は Scope の外。持ち越し |
| 294「`ralph org merge-order` reads a `gate_failed` that `ralph org regate` wrote」 | gate の記録の鍵(`load_*`、`max_load` を `log=` の前に足した)、`mergeRegateFailedDetail` の 1 行 | `merged=` の鍵は足していない。受け入れは regate の merge commit の head にしか効かない(D5)ので、この行の読み方に頼っている。持ち越し |
| 306「`mergeguarantee.go` (877 lines, ...)」 | `mergeorder.go` 1328 → 1438 行 | runner を別のファイルへ移す件は Scope の外。持ち越し |
| 312「On a branch with a gate record and no feature binding」 | merge-order の分類(`accepted` を足した) | `orgRunsOn` は変えていない(binding のある枝だけ)。持ち越し |
| 313「`ralph org merge-order` classifies a PR that gh reads as MERGED」 | 新しい class `accepted` | 他の base への merge は今も `merged`。`TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged` は変えていない。持ち越し |
| 367「Only the leader's role prompt keeps a `wip:` commit out」 | `gate.go`、`gate_run.go` | 変えたのはスクリプトを始める時と記録の鍵で、gate の検査の中身(`wip:` の検査はここに入る)は変えていない。持ち越し |
| 372「RC1 (org recheck 2026-10-24)」 | `reviewer.md`(gate の TASK の節。570 → 594 行) | 座席の agmsg の path は足していない。持ち越し |
| 390「Four test files went past the 800-line guide」 | `org_merge_test.go` 1195 → 1243、`merge_run_test.go` 1111 → 1158、`director_skill_sync_test.go` 1223 → 1307 | `cleanup_test.go` は変えていない。持ち越し |
| 398「Two test gaps that the `/test` of local-config-org left」(a) | `Regate`、`runGateScripts` | 新しい Regate のテスト 4 本は、ロックと印と待つ順を固定し、scripts の環境は読まない。持ち越し |
| 399「`ralph org cleanup` and `ralph org merge` print the command to run again with no `--state-dir`」の残り | **掃引で見つけた**。merge-order の Detail(`failed` に log、新しい `accepted`)と merge の class の確かめ | `mergeGateCommand` と `mergeRegateCommand` は変えていない。新しい `accepted` の Detail は command を名指さず、`regate` の Detail は元の文に文を足しただけ。持ち越し |
| 405「Three files went past the 800-line guide with the ci regate」 | `regate.go` 995 → 1054 行、`org_regate_test.go` 1273 → 1305 行 | `regate_ci.go` は触っていない(901 行)。分割は Scope の外。持ち越し |
| 415「Five files went past the 800-line guide」 | `config.go` 1173 → 1190 行(`max_load` の欄、既定値、検査) | 分割は Scope の外。ほかの 4 ファイルは触っていない。持ち越し |
| 418「`mergeorder.go` grew from 1291 lines」 | `mergeorder.go` 1328 → 1438 行 | ranking と PR の lookup の移動は Scope の外。持ち越し |
| 453「`[org].api_wait_minutes` has no upper bound」 | `validateLoaded` に `max_load` の検査を足した | `api_wait_minutes` は今も上限がない。持ち越し |
| 457「Seven mutants of `ralph org resume` survive」(境界) | `leaderActivityEventCount` の doc と、`TestLeaderActivityEventCount_GateEvents` の 1 行 | 関数のコードは変えていない。(a)〜(g) は触っていない。持ち越し |
| 464「The `/director` skill ... do not say what a director does when it sees the scaffold version note」 | **掃引で見つけた**。director skill の 4 つの写し(受け入れの節、merge の節、してはいけないこと) | skill は今も `ralph upgrade`、`manifest.toml`、scaffold を名指さない(`git grep -c -i` は 0 件)。`prompts/director.md` は触っていない。持ち越し |

掃引で見つけた 4 行(118、248、399、464)は、verify も handoff も挙げていない。

### V2: README

verify が挙げた 2 段落に加えて、動詞の表の行と merge の段落も直した。help(`internal/cli/org_gate.go`、`org_gate_accept.go`、`org_merge_order.go`、`org_merge.go`、`org_regate.go`)の文と 1 つずつ照らした。

- 動詞の表の `gate`: 記録に負荷が入ること、台帳で 1 本ずつ走ること、`gate accept` を足した。`ralph org` の動詞の一覧(`gate/merge-order/...`)は `gate accept` を 1 語として足さなかった。`director`、`followup`、`inbox` の部分動詞も、一覧では親の名前だけである
- `ralph org gate` の段落: gate.lock、待つ間の stderr の行、待ったあとの読み直し、scripts への fd の継承、`max_load`(既定 0、負は設定エラー、30 秒ごとに読み、30 分で始める)、負荷の鍵 4 つと `max_load`、`gate accept` の記録と断り(help の断りの一覧と照らし、「最新の記録が `--log` と違う」を主語を取り違えない形で書いた)、`ci_failed` の受け入れは merge を通さないこと
- merge-order の段落: `accepted` の定義、class の順(`failed` の注、`ready` の前の `accepted`)、並び(`ready`、`accepted`、`rebase`、…)、「`ready` と `accepted` を 1 つ merge すると、ほかの `ready` と `accepted` は `rebase` になる」、受け入れが消える条件と regate の merge commit の失敗、JSON の `log` と `accepted`、local の regate の待ち
- merge の段落: `pr_merged` の末尾の `gate=accepted` と `GATE: accepted (...)` の行

### V3: `templates/base/ralph.toml`

290 行目の「each PR that `ralph org merge-order` lists as ready」を「ready or accepted」に直し、折り返しを 4 行にした。`go test ./internal/config/` は ok(13.3 秒)。

### V4: `quality-gates.md`

Quality pipeline gate の行の、`ralph org gate` が何を確かめ、何を記録するかの文の次に、gate.lock(gate と local の regate が先に取り、時間の上限なしで待つ)、`max_load`(上限 30 分)、負荷の記録、`ralph org gate accept`(人か、人に言われた director が打つ。`static_failed`・`test_failed`・`ci_failed` だけ)と `accepted` の merge(PR の checks と保護の live の確かめは同じ。`ci_failed` の受け入れは、PR の check が落ちている間は merge を通さない)、gate の記録が増えると受け入れが消えること、を足した。root と templates の写しに同じ文を入れた。check-sync は `PASS`(DRIFTED 0)。

### I1〜I4

- I1: `OrgDirectorConfig` の comment を直した(上の表)
- I2: 計画の Progress checklist の 3 つの箱を `[x]` にした。AC の箱は計画の `## Acceptance criteria` にあり、digest の中なので触っていない。`/pr` の前に付ける
- I3: 仕様の FR-7 の注が名指す `docs/plans/archive/2026-11-12-gate-lock-accept.md` は、`/pr` が計画を archive に移すと正しくなる。直していない
- I4: 条件に当たっていない(`reviewer.md` の文は timeout にその時間を入れるよう言う)。直していない

## self-review と test の指摘

### L3: skill の例の `--log` の quote

director skill の例を `--log '<LOG: のパス>'` に、org skill の行の例を `--log '<LOG のパス>'` にした。director skill の手順 3 は「空白を含みうるので `'…'` で囲む」を言い、org skill の行の `--log` の説明にも同じ文を足した。`--reason` と同じ形で、パスに `'` が入る場合は書いていない。director skill の例は `director_skill_sync_test.go` が固定していたので、固定の文字列を新しい形に直した。

### L5: `ci_failed` の受け入れは merge を通さない

director skill の「gate の失敗の受け入れ」の冒頭に、`ci_failed`(ci の regate が PR の checks から記録した失敗)を受け入れても、`ralph org merge` は PR の checks を読み直してすべてが pass か skipping になるまで断るので、落ちた check を GitHub で打ち直すのが先になる、を 1 文足した(`TestDirectorSkill_PinsTheAcceptedGateFailureMerge` に固定を足した)。`gate accept --help` に同じ内容の段落を足した(`TestOrgGateAccept_Help` に固定を足した)。org skill の行には足していない(依頼は director skill と help だけ)。README と `quality-gates.md` には同じ内容を書いた。

### L4 と test の穴

- 468 行目(`gate.lock` has no wait limit ...): 第 1 列の末尾に「local の regate が最初の検査より前に gate.lock と max_load を待つので、すぐ断られる regate も先に待つ」を足し、(c) を第 2〜4 列の 3 か所で「断られる regate」を含む形に広げた。Trigger の (c) に、台帳だけを読む検査を gate.lock の前に置く案を足した
- 476 行目(新しい行): C1(`waitGateMaxLoad` の待ったあとの上限と同じ値)、C2(最後の sleep を上限の残りに切り詰めること)、C8(`gateRunLoad.finish` が終わりの値を最大に入れること)、D1(CLI の regate が config の `max_load` を渡すこと)。それぞれの変異が何を変えるか、probe が HEAD で通り変異で落ちること(probe の source は test report の evidence の log にあり、commit していない)、Trigger に足すテストの形を書いた。名前は関数とテストの名前で、`file:line` は無い

## 掃引

Trigger の欄を python で切り出し(`" | "` で分けた 4 つ目のセル、`~~...~~` を外し、最初の日付つきの note より前だけ)、変更ファイルの名前と、diff の関数名(`func` の追加と削除、hunk の関数名)を当てた。名前だけでは拾えない Trigger のために、キーワード(gate、merge-order、`/director`、skill、accept、approve、reviewer、config、template、help、800、「any of」「the five」)でもう 2 回引き、当たった行は Trigger を 1 つずつ読んだ。hunk の関数名は 1 つ前の関数を指すことがあるので、`watch.go` と `mergeorder.go` と `lockfile.go` は hunk の中身を読んでから判断した。当たった行は上の表のとおり。

当たらない行と理由は次の表のとおり。どれも Trigger の本文を読んで外した。

| 行(先頭の語) | 判断 |
|----|----|
| 148(config の解決の規則)、151(`DefaultRoleModels`) | `resolveOrgConfig` と `DefaultRoleModels` は変えていない。148 は `ralph.toml` という名前に当たっただけ |
| 245、246、250、254(merge-order の director、ssh の origin、merge guarantee) | Trigger の中身は director の段の判断と `mergeguarantee.go` で、日付つきの note が段ごとに説明している。`MergeOrder` の関数は変えていない(hunk は `mergeTarget`、`collectMergeTargets` 以降) |
| 240(`signal.NotifyContext` の 3 つの command) | 差分に出る `signal.NotifyContext` の行は context で、signal の扱いは編集していない |
| 261、262、275、311、362、365、380、381、407、417、421、426、427、462、463 | 挙げる関数、テスト、節は変えていない。362 と 417 は `status.go` の help の文だけ(seat の表と section は変えていない)。365 は `prompts_test.go` の 2,700 → 2,722 行(行の数え始めの 2,670 に 300 を足した閾値の内側で、足したのは reviewer の雛形のテスト)。380 は skill の「答える順」を直していない(新しい節の手順 5 が参照するだけ)。407 は `gateWording` と regate の refusal の文を変えていない。426 は `reviewer.md` に節を足していない(既存の節の文を直しただけ)。462 は `watch.go` の comment の 1 か所で、`scope_change`・`stall`・recurrence のコードではない |
| 219(`approveTripwire`) | tripwire の動きは変えていない(`seatPaneTripwire` に取り出して accept と共有した)。Trigger は 8a と「本物の境界が要る時」 |

## 確かめたこと(文書が実装と合っていた所)

- skill の 8 つの写し: `check-skill-sync.sh` は 14 skill で ok。`check-sync.sh` は `PASS: all files in sync.`(DRIFTED 0)
- `README.md` の class の順と並びは `ralph org merge-order --help` の順(`ready`、`accepted`、`rebase`、`wrong_base`、`regate`、`unreadable`、`failed`、`no_remote`、`no_gate`)と同じ
- `AGENTS.md` の `internal/org/` の行は S5 が `gate_lock.go`、`gate_load.go`、`gate_accept.go` を足してあり、今の実装と合っている。この pass では触っていない
- `.claude/rules/ralph/agent-messaging.md` は触る必要がない(accept は agmsg の message ではない。計画の Verify plan のとおり)
- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 の 2026-11-12 の追記は `accepted` の merge と `gate=accepted` を言っている。FR-8 の 8a-3a の段落(「`ready` と分類したブランチ」)は日付つきの記録なので直していない
- `docs/recipes/codex-seat-permissions.md` の gate の文(reviewer が `ralph org gate` を打つ)は今も正しい。gate.lock と負荷の話は足していない(座席の権限の recipe で、gate の動きの説明ではない)

## 残っていること(lead へ)

- Go は 3 か所を直した(`gate accept --help` の段落、`OrgDirectorConfig` の comment、固定のテスト 2 本)。動きは変えていない。範囲を絞ったテストを走らせたので、パイプラインの全体の再走は要らないと判断した。`go test ./internal/cli/ -run 'GateAccept|Help'` と `go test ./internal/config/` と `go test ./internal/org/ -run 'DirectorSkill|OrgSkill|SkillSend|TestRenderRolePrompt_Reviewer'` は ok。ただし test report の数(1,116 本など)は、この変更の前の数である
- 払っていないもの: 持ち越しの note を足した 24 行の Trigger、新しい 476 行目の 4 つの穴(どれもコードかテストの変更)
- self-review の L1(merge の文の「the head the gate passed」)、L2(`--log` の断りの文)、L6(名前)、L7(定数と help の数)、L8(小さな重複)は、コードの変更なので触っていない。cross-review の前に直すなら 1 commit にまとめられる
- 行の中の `wc -l` の数(`regate.go` 1054、`mergeorder.go` 1438、`config.go` 1190、`director_skill_sync_test.go` 1307、`reviewer.md` 594 など)は、この commit のあとの HEAD で正しい(この pass が最後の Go の変更)。このあとコードを足すなら数え直す
- 同じファイル(`docs/tech-debt/README.md`)を別のセッションも編集しうる。この commit は 25 行への挿入と末尾の 1 行の追加で、既存の行は消していない。merge の衝突は Trigger の末尾と末尾の追加行で起きうる
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える
- AC の箱(計画の `## Acceptance criteria`)は `[ ]` のまま。verify が AC1〜AC14 を満たすと判定したので、`/pr` の前に付ける

## Verification

- `git show HEAD:docs/tech-debt/README.md` との行ごとの比較(`difflib`)で、変わった行は 25 行(88、118、159、180、238、248、249、269、293、294、306、312、313、367、372、390、398、399、405、415、418、453、457、464、468)と、末尾の新しい 1 行(476)。変わった行はすべて `insert` だけの編集で、列は 5、`~~` は偶数。新しい `file:line` は 0
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0(pipeline の参照、skill の同期、template の純度、tech-debt の計画の参照を含む)
- `./scripts/check-skill-sync.sh`: 14 skill で ok。`./scripts/check-sync.sh`: `PASS`
- `go test -timeout=30m ./internal/config/`: ok。`go test ./internal/cli/ -run 'GateAccept|Help'`: ok。`go test ./internal/org/ -run 'DirectorSkill|OrgSkill|SkillSend|TestRenderRolePrompt_Reviewer'`: ok(`TestDirectorSkill_PinsTheAcceptedGateFailureMerge` と `TestOrgSkill_PinsTheGateLockAndTheAcceptedMerge` は `-v` で PASS)。`gofmt -l`(編集した 4 つの Go ファイル)は出力なし、`go vet`(config、cli、org)は通った
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-gate-lock-accept.md` は編集の前後とも `1d8736effdd9`
- `git diff --check` は rc 0。変えた 19 ファイルとこの report の U+FFFD は 0 件
- 実行していないもの: パッケージ全体のテスト(`./scripts/run-test.sh`)、`run-verify.sh` の全体

## Insight event

```
./scripts/insights-append.sh --slug gate-lock-accept --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-gate-lock-accept.jsonl` が 3 行から 4 行になり、足した行は cycle 1、pass、ts は 2026-11-12T14:03:07Z。この時点の `git status --short` は既存のファイルの `M` だけで、迷子のファイルは無かった。
