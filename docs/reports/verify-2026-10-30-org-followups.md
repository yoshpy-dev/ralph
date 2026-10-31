# Verify report: org-followups

- Date: 2026-11-01(cycle 1 と cycle 2)
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(issue #270、AC1〜AC8、D1〜D5)
- Verifier: verifier サブエージェント(`/verify`)。cycle 2 の節はパイプラインの cycle 2(`cycle-count.json` は 2)、cycle 1 の節は cycle 1
- 構成: 先頭が cycle 2 の節、次がこのファイルの判定の `## Verdict`(cycle 2 のもの)、最後が cycle 1 の節。cycle 1 の節は中身を変えず、見出しを 1 段下げ、判定の見出しを「Verdict(cycle 1)」にし、冒頭の Scope と Evidence の行をその節に移した。ファイルの `## Verdict` は cycle 2 の 1 つだけ
- Evidence: `docs/evidence/verify-2026-10-30-org-followups.md`(コミットする。先頭が cycle 2 の要約、続いて cycle 1)。全出力は `docs/evidence/verify-2026-10-30-org-followups.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。cycle 2 の分は末尾に足した)。run-verify 自身のログは、cycle 2 が `docs/evidence/verify-2026-10-31-180917.log`(同じく対象外)

## Cycle 2(HEAD 12d28dae)

### Scope(cycle 2)

- ブランチ全体: `git diff origin/main...HEAD`(merge-base = origin/main 1cc84ffa、HEAD 12d28dae、45 ファイル、+6897/-134)。HEAD は origin/feat/org-followups と同じ。
- 重点: cycle 1(d2890dbf、HEAD 9a49f306)のあとの変更 `git diff 9a49f306..HEAD`(23 ファイル、+1115/-106)。コードは F4 a4cab358(merge-order の follow-up の表の SOURCE の列)、F5 25c9fa77(follow-up だけの経路の `gh_available`)、F6 12d28dae(コメントと help の折り返し)。文書は sync-docs 9717f0e8(tech-debt の注と行、見張りのいらない動詞に `followup`)、計画の再承認 a5372aaf と Progress の 32c73433、triage 6c9281aa、self-review cycle 2 の c18ffcfd。
- spec の準拠と静的解析と文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。

### Spec compliance(cycle 2)

| Acceptance criterion | Status | cycle 1 からの変化と根拠 |
| --- | --- | --- |
| AC1 `followup add`・`list`・`done`、記録が残る | 満たす | 変化なし。`internal/org/followup.go` と `internal/cli/org_followup.go` は 9a49f306 から変わっていない。probe: 見張りのない台帳で add・list・`list --json`・done・`list --all` がどれも rc 0(evidence の cycle 2 の B 節) |
| AC2 lock、壊れた行、ID | 満たす | 変化なし。store のコードは変わっておらず、cycle 1 の根拠と probe がそのまま使える |
| AC3 merge-order の follow-up の節と `--json`(判定の道 `gh`)、終了コード | 満たす | F4: 表の見出しは `ID ISSUE AFTER_BRANCH STATE SOURCE PR DETAIL`(`internal/cli/org_merge_order.go:280`)、各行の SOURCE は `inboxField(f.Source)`(`:287`)、値は `FollowupMergeSourceGH`(`internal/org/mergeorder_followup.go:286`)。F5: follow-up の検索が gh を見つけたかを返し(`:173`、`:199-207`)、`GHAvailable` に OR で足す(`:134-136`)。probe(偽の gh、機能のブランチのない台帳): 7 件の行はどれも 7 列で SOURCE は `gh`、状態は ready 1・waiting 3・closed 1・unknown 2 で、DETAIL は cycle 1 と同じ。`--json` の `gh_available` は、gh が答えると true、全ブランチで gh が失敗しても true、PATH に gh がなければ false(全件 `unknown`)、open の follow-up がない台帳(空、done だけ)では false で gh の呼び出しは 0。機能のブランチがある台帳では、follow-up なしで true / gh なしで false(前と同じ)、follow-up ありで true(gh 8 回)、全失敗で true、gh なしで false。13 通りとも rc 0(evidence の cycle 2 の C 節) |
| AC4 status の follow-up、`recorded`、gh なし | 満たす(既存の gh なしのテストが通ることは /test) | `status_followup.go` の変更はコメントだけ(`:170-178`)。probe: status の text と `--json` のあとも偽の gh の呼び出しは 0、SOURCE は `recorded` |
| AC8 同じ名前のブランチの前の merge を数えない | 満たす | 変化なし。probe: feat/old(2020 に merged の PR #3)と feat/open(open の #12 と 2020 に merged の #4)は merge-order で `waiting`、DETAIL は「does not count」 |
| AC5 status の予約、JSON の形 | 満たす | 変化なし(`status.go` は 9a49f306 から変わっていない) |
| AC6 skill・README・AGENTS.md・仕様の注・help、写しと固定のテスト | 満たす(テストの実行は /test) | F4 の列は help(`org_merge_order.go:103-104`)、`/org` skill の merge-order の行(`.claude/skills/org/SKILL.md:259`)と「merge を待つ issue」の節(`:1648`)、固定のテスト(`internal/org/skill_followup_test.go:125`・`:199`、`internal/cli/org_merge_order_followup_test.go:30`)にある。4 つの写しは `cmp` で同じ。見張りのいらない動詞の一覧(quality-gates の Cross-org watch gate の行 `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79`、`/org` skill の見張りの節 `:1963-1966` の 4 つの写し)に `followup` の動詞が入った |
| AC7 出力だけで状態と理由がわかる | 満たす | merge-order の表に SOURCE の列が入り、テキストだけで判定の道(`gh` / `recorded`)がわかる(cycle 1 の I2 は解決) |

設計の決め(D1〜D5)との照合:

- D1・D3・D4: 変化なし。
- D2: 満たす。cycle 1 の I2 は F4 で解決し、表と JSON の両方に判定の道が出る。
- D5: 満たす。cycle 1 の I1 は解決した。計画の D5(`:56`)は「ready と waiting で割れたら merge-order、merge-order が `unknown` なら理由を添えて status」の形に直り、digest 6ef9fe7cb76c で承認し直してある(Progress の `:114`)。help・skill・README・AGENTS.md・仕様の注・tech-debt 409 行の文と同じ意味。

見張りのいらない `followup` の動詞:

- コード: 3 つの動詞は `newOrgFollowupStore`(`internal/cli/org_followup.go:217-227`)で台帳を解決して store を作るだけで、見張りの gate(`wireOrgWatchGate`)を呼ばない。gate を呼ぶのは spawn(`internal/cli/org.go:869`)、send(`:1273`)と、`wireOrgStartWatchGate`(`:496`)を通る start(`:1075`、`:1178`)と director start(`internal/cli/org_director_headless.go:335`)だけ。
- probe: 見張りのない台帳(`watch.lock` と `watch-alive.json` がない)で、`followup add`・`list`・`list --json`・`done`・`list --all` がどれも rc 0 で、台帳には `followups.jsonl` と `followups.lock` だけができた。同じ台帳で `ralph org spawn --dry-run` は見張りの拒否(「no ralph org watch --all is running ...」)で rc 1 になり、台帳に何も足さなかった。

F5 の `gh_available` の文書:

- 欄の説明(`internal/org/mergeorder.go:192-197`)は、機能のブランチの検索と open の follow-up の after-branch の検索のどちらかが gh を見つければ true、gh がないときと、聞く相手(機能のブランチも open の follow-up も)がないときは false、と書く。probe の 13 通りと合う。`MergeOrder` の説明(`:289-310`)、`mergeOrderFollowups`(`mergeorder_followup.go:108-114`)、`lookupFollowupMerges`(`:170-172`)、package の注(`:24-26`)も同じことを書く。
- `gh_available` は help・README・`/org` skill・仕様には書かれていない(main のときから)。`git grep` で当たるのはコード、テスト、計画の記録、tech-debt の閉じた 245 行だけで、外の文書とのずれはない。

self-review cycle 2 の所見の扱い: C2-2(コメント 2 か所と help の 74 桁の行)は F6 で直った。package の注は Followups・GHAvailable・記録が読めないときの注記を挙げ(`mergeorder_followup.go:24-26`)、merge-order の help の行は最長 72 桁。C2-1 は下の C2-V1 で確かめた。

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、39 秒) | 範囲は `Language scope: changed (changed_languages)`、`golang`。gofmt `ok`、go vet と staticcheck は無出力(`Skipping` の行 0)、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の構文、`jq -e`、Codex の hook の検査 3 つ)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync(7 行とも `[ok]`)、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照もすべて OK。secret scan は `1cc84ffa..12d28dae` で clean。timeout には当たらず、打ち直しもしていない |
| `./scripts/check-template-purity.sh` | pass(rc=0) | 単独でも打った |
| `./scripts/check-skill-sync.sh` | pass(rc=0、14 本) | 単独でも打った |
| `./scripts/check-sync.sh` | pass(rc=0、DRIFTED 0、ROOT_ONLY 0) | quality-gates.md は KNOWN_DIFF。Cross-org watch gate の行は 2 つの写しで同じ文で、ほかの違い(root の 54・57-58・82 行)は既存のもの |
| `git diff --check origin/main...HEAD` | pass(無出力) | |
| 変えたファイルの U+FFFD の grep | 0 件 | |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-followups.md` | `6ef9fe7cb76c` | `- Approved:` の digest と同じ |
| `go build -o <scratch>/ralph ./cmd/ralph` | pass(rc=0、go1.27.3) | probe 用。前後で `git status --porcelain` は空 |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org merge-order --help` | はい | 節の列に SOURCE(`org_merge_order.go:103-104`)。行幅は 72 桁まで |
| `printMergeOrderFollowups`・`printStatusFollowups` の説明 | はい | `org_merge_order.go:265-274`、`status_followup.go:170-178` |
| `MergeOrderResult.GHAvailable` と F5 のコメント | はい | 上の「F5 の `gh_available` の文書」 |
| `/org` skill(4 つの写し) | はい | SOURCE の列(`:259`、`:1648`)、見張りのいらない動詞(`:1965`) |
| `/director` skill(4 つの写し) | はい | `:1107-1108` の SOURCE の文は、status と merge-order の両方の表に合う |
| quality-gates(root と template) | はい | Cross-org watch gate の行に `followup` の動詞 |
| 計画の D5 | はい | cycle 1 の I1 は解決 |
| 計画の Progress | 遅れ(C2-I2) | 下の Findings |
| tech-debt 177・235・246・251・303・359 の注(cycle 1 の V1)、410 の行数(cycle 1 の V2) | はい | sync-docs 9717f0e8 が注を足した。410 の 963 行と 189 行は今と同じ |
| tech-debt 303・411・412 | いいえ(C2-V1、self-review の C2-1) | 下の Findings |
| tech-debt 118 | いいえ(C2-V2) | 下の Findings |
| tech-debt 246・251 の注の列挙 | はい(C2-I1) | 下の Findings |

### Findings(cycle 2)

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-V1 | LOW | self-review の C2-1 を確かめた。台帳の 3 行が F5・F6 のあとで古い。(a) 303 行の注は `mergeorder.go` を「1291 to 1322 lines」と書くが、今は 1328 行。(b) 411 行は「to 1322」「(360 lines)」と書くが、今は 1328 行と 371 行(self-review の 370 は F6 の前の数で、F6 が package の注に 1 行足した)。411 行の Trigger「The next change to `internal/org/mergeorder.go`」に F5・F6(どちらもコメントだけ)が当たったが、注はない。(c) 412 行の Trigger の (4)「the next change to `checkFollowupPRs` or `lookupFollowupMerges`」に F5(`lookupFollowupMerges` の戻り値と `ghFound`)が当たったが、払っておらず、注もない | `docs/tech-debt/README.md:303`・`:411`・`:412`、`wc -l`(evidence の cycle 2 の D 節) | cycle 2 の /sync-docs で 303 と 411 の数を 1328 と 371 に直し、411 と 412 に 9717f0e8 と同じ形の持ち越しの注を足す。412 の (4) はテストを 1 つ足して払ってもよい |
| C2-V2 | LOW | C2-1 の外にもう 1 行ある。118 行の Trigger は「`templates/base/docs/quality/quality-gates.md` …への次のタッチ」で、sync-docs 9717f0e8 がこのファイル(79 行の Cross-org watch gate の行に `followup` の動詞)を変えたので当たった。払っておらず(「Must pass in CI」の節が挙げる `check-coverage.sh` と `check-pipeline-sync.sh` は、今も `templates/base/.github/workflows/verify.yml` が呼ばない)、持ち越しの注もない。同じ行には、同じ形のタッチに feat/reviewer-runs-gate が足した 2026-10-26 の注がある。9717f0e8 は cycle 1 の verify のあとなので、cycle 1 では見られなかった | `docs/tech-debt/README.md:118`、`templates/base/docs/quality/quality-gates.md:58-59`・`:79`、`templates/base/.github/workflows/verify.yml`(2 つのスクリプト名の grep は 0 件)、9717f0e8 | cycle 2 の /sync-docs で、118 行に 2026-10-26 の注と同じ形の注を足す(feat/org-followups が Cross-org watch gate の行を変えたが払っていない、Carried over) |
| C2-I1 | INFO | 246 行と 251 行の org-followups の注は、sync-docs のとき(HEAD 523b9091)までの編集を挙げる。F5・F6 も同じ 2 つの Trigger(`mergeorder.go`・`org_merge_order.go` の次の編集、`MergeOrder` の次の編集)に当たるが、変えたのは `GHAvailable` の決め方と説明、`MergeOrder` の説明、help の折り返しで、どちらの行の文字の項目にも触れない。注の「Carried over」は今も正しい | `docs/tech-debt/README.md:246`・`:251`、F5 25c9fa77、F6 12d28dae | 任意。C2-V1 の直しのときに、246 の注の列挙に F5 の `GHAvailable` を一言足してもよい |
| C2-I2 | INFO | 計画の Progress には、cycle 2 の self-review(c18ffcfd、LOW 2)と F6(12d28dae、C2-2 の直し)の記録がまだない。最後の行は 127 行の cross-review と F5。Progress は digest に入らない(Progress に 127 行を足した 32c73433 のあとも digest は 6ef9fe7cb76c) | `docs/plans/active/2026-10-30-org-followups.md:127` | cycle 2 の /sync-docs で記録を足す |

### Observational checks(cycle 2)

HEAD から build した binary を、scratchpad の一時 repo で打った。組み方は cycle 1 と同じで、bare origin と main を作り、`env -i` で HOME と git の設定を一時のものにし、PATH には `remote get-url origin` にだけ `https://github.com/acme/app.git` と答える git の包みと、`--head` ごとに決めた JSON を返す偽の gh だけを置いた(`PROBE_GH_MODE=fail` のときはどの呼び出しも exit 1)。ネットワーク、本物の gh、herdr、agmsg は使っていない。出力は evidence の cycle 2 の B 節と C 節にある。

- 見張りのいらない動詞と `gh_available` の 13 通りは、上の「見張りのいらない `followup` の動詞」と AC3 の行のとおり。
- status は follow-up のある台帳で gh を呼ばず、SOURCE は `recorded` だった。
- `internal/cli/org.go:452-456`(`wireOrgWatchGate` の説明)と `internal/org/spawn.go:297-299` の「ほかの動詞は見張りなしで動く」という例は `followup` を挙げない。ただし `approve`・`gate`・`merge-order` なども main のときから挙げていない例示なので、この branch のずれとはしない。

### Coverage gaps(cycle 2)

- Go のテストの結果は /test の範囲。F5 の新しいテスト `TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch`(`internal/org/mergeorder_followup_test.go:661`)、`TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable`(`:725`)、`TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable`(`internal/cli/org_merge_order_followup_test.go:282`)と、F4 の固定のテストは、存在と中身を読んだところまで。
- gh が一度答えたあとに見つからなくなる場合(`ghFound = false`、`mergeorder_followup.go:202`)と、検索の途中の中断は CLI から作れないので probe していない。コードとテストの存在で確かめた。
- 本物の gh と GitHub は、cycle 1 と同じく使っていない。
- tech-debt の Trigger の掃き出しは、cycle 2 で変わったものの名前(`mergeorder`、`merge-order`、`MergeOrder`、`GHAvailable`、`lookupFollowupMerges`、`checkFollowupPRs`、`printMergeOrderFollowups`、`status_followup`、`quality-gates`、`/org` skill、watch gate ほか)の部分一致で引いた。語で書かれた Trigger(出来事)は、引けた行だけを読んだ。

## Verdict

pass

- Verified: AC1〜AC8 と D1〜D5 は HEAD 12d28dae で満たす。cycle 1 から変わった AC3・AC6・AC7(SOURCE の列、`gh_available`、見張りのいらない動詞の一覧)は、HEAD の binary の probe とコードで確かめた。変わっていない AC1・AC2・AC4・AC5・AC8 は、コードが 9a49f306 から変わっていないことと probe の一部で確かめた。静的解析はすべて通り、計画の digest は承認時の 6ef9fe7cb76c と同じ。cycle 1 の I1・I2・V1・V2 と、self-review の C2-2 は解決している。
- Partially verified: AC4 と AC6 のテストが通ること、gh が途中で消える場合と中断。
- Not verified: Go のテストの結果(/test)、本物の gh。
- 所見: LOW 2 件(C2-V1 は self-review の C2-1 と同じ台帳の 3 行で、正しい数は 1328 と 371。C2-V2 は C2-1 の外の 118 行)、INFO 2 件(C2-I1 は 246・251 の注の列挙、C2-I2 は Progress の遅れ)。どれも merge を止めない。C2-V1・C2-V2・C2-I2 は cycle 2 の /sync-docs に回す。
- Insight event: `docs/insights/events/2026-10-31-org-followups.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 2`、cycle は 2)。ファイル名は UTC の日付。

## Cycle 1(HEAD 9a49f306)

- Scope(cycle 1): `git diff origin/main...HEAD`(merge-base = origin/main 1cc84ffa、HEAD 9a49f306、37 ファイル、+5851/-97)。spec の準拠と静的解析と文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない
- Evidence(cycle 1): `docs/evidence/verify-2026-10-30-org-followups.md`(コミットする。静的解析の要約、probe の出力、help、規則の文の grep、行数、tech-debt の掃き出し)。全出力は `docs/evidence/verify-2026-10-30-org-followups.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-31-160435.log`(同じく対象外)

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `followup add` が記録して ID を出し、`list` が open を(`--all` で done も)表と `--json` で出し、`done <id>` が完了にする。記録は `followups.jsonl` に残り、プロセスや director が替わっても読める | 満たす | `internal/cli/org_followup.go:50-117`(add、`added <id>: issue #<N> after <B>` は `:105`)、`:119-166`(list、既定は open だけ `:146-151`、`--json` は `{items, corrupt_lines, ignored_events}` `:232-236`)、`:168-209`(done)。store は `internal/org/followup.go:147-172`(Add)と `:181-208`(Done。2 度目は changed=false で 0)。probe: 別々のプロセスで add 11 回、list、`list --json`、done、2 度目の done(`f12 is already done; nothing changed`、rc 0)、知らない ID(rc 1)、`list --all` の `DONE_AT`・`DONE_NOTE` の列 |
| AC2 自分の lock の中で読み直してから追記、壊れた行と知らない event を数えて警告、ID は壊れた行を含めて重ならない | 満たす | `followup.go:152-167`(`followups.lock` の中で `readRaw` → `nextID` → `appendLocked`)、`:288-290`(`withFileLock(..., "followups.lock", ...)`、ほかの lock と別のファイル)、`:57`(壊れた行からも `"id":"f<N>"` を拾う)、`:319-321`(最後の行が切れていれば改行を足す)、`:380-414`(知らない event などを ignored に数える)。CLI の警告は `org_followup.go:241-249`。probe: `"id":"f50"` で切れた最後の行のあとの add は `f51` になり、切れた行は別の行のまま残った。list は 1 行の壊れと 1 件の無視を stderr に出した。空の台帳に 20 の add を同時に打つと、ID は 20 個とも異なり記録は 20 行 |
| AC3 merge-order が open の follow-up の PR を gh で引き、記録より後に merged は `ready`、open は `waiting`、merge されずに closed は `closed`(理由)、gh がないか引けなければ `unknown`(理由)。表のあとの節と `--json`(判定の道 `gh`)。終了コードは今と同じ | 満たす | `internal/org/mergeorder_followup.go:165-222`(after-branch ごとに 1 回の gh。gh がなければ全件 `unknown` `:192-194`、失敗はそのブランチだけ `:195-197`)、`:274-327`(状態と DETAIL)、`:56`(`source` は `gh`)。`internal/org/mergeorder.go:326`・`:344`(対象のある道とない道の両方で節を作り、エラーにしない)。CLI は `internal/cli/org_merge_order.go:191`(`Followups: true`)、`:273-287`(節)。probe(偽の gh): ready・waiting・closed・unknown(gh の失敗、`mergedAt` が空)が DETAIL つきで出た。gh のない PATH では 12 件とも `unknown`、`gh was not found ...`。どの場合も rc 0。同じブランチの follow-up 2 件で gh は 1 回 |
| AC4 status が open の follow-up を出し、記録より後の `pr_merged` か `worktree_cleaned` で「着手できる」。表と JSON に判定の道(`recorded`)。gh は走らせない | 満たす(既存の gh なしのテストが通ることは /test) | `internal/cli/status_followup.go:126-168`(manifest の event と `followups.jsonl` だけを読む)、`:26`(`recorded`)、`internal/org/followup.go:470-504`(`FollowupRecordedMerge`、`!ts.Before(added)`)。表の SOURCE の列は `status_followup.go:184`、JSON の `source` は `:49`。probe: status を text・`--json`・`--org-id` の 3 回打って、偽の gh の呼び出しの記録は空のままだった。`status.go:153` の「status runs no gh」の文は残る。テスト `TestStatusCmd_FollowupsRunNoGH`(`status_followup_test.go:668`)は PATH に呼び出しを記録する gh だけを置く |
| AC8 同じ名前のブランチの前の merge(記録より前の `pr_merged`、前に merged の PR)で「着手できる」にならない(D5) | 満たす | status: `followup.go:486-495`(記録より前の event は数えない)。merge-order: `mergeorder_followup.go:333-351`(`mergedAt` が記録より前は `before`)と `:319-323`。probe: 2020 の `pr_merged` しかない feat/old は status で `waiting`、2020 に merged の PR #3 しかない feat/old は merge-order で `waiting`(「does not count」と理由)。open の PR #12 と 2020 に merged の #4 を持つ feat/open も `waiting` で、#4 を数えないと DETAIL に書く |
| AC5 status が走っている org の予約のパスを表と `--json` に出す(予約だけの org も)。空と中身ありの JSON の形は同じ | 満たす | `status_followup.go:74-99`(`RunningOrgs` と `ActiveReservation`、予約だけの org に組を足す)、`:105-109`(`  reserved: a, b`)、`:113-115`(空の配列)。`status.go:582`(`reservation`)、`:614`(`followups`)、`:772`(座席のない組では見出しを出さない)。probe: 予約だけの org-r は `org_id: org-r (active 0/0)` と `reserved:` の行、JSON は `reservation` 2 件・`seats` 0。予約のない org-t は `"reservation":[]`。空の台帳と中身のある台帳で top level の key は同じ 7 つ、`followups` は同じ 4 つの key |
| AC6 `/org`・`/director` skill(と写し)、README、AGENTS.md、仕様の注、`ralph org` の help が動詞と使い方を書き、写しの検査と文を固定するテストが通る | 満たす(テストの実行は /test) | `/org` skill の表 `.claude/skills/org/SKILL.md:254-256`(3 つの動詞)、`:246`・`:259`(status と merge-order の行)、`:359`、節「merge を待つ issue」`:1605-1658`。`/director` skill `:853`、`:933`、節 `:1090-1119`。4 つの写しは `cmp` で同じ、check-skill-sync と check-sync が通る。README `:131`(動詞の一覧と説明)、`:132`(status)、`:284`(merge-order)。AGENTS.md `:91`。仕様 FR-9 の注 `docs/specs/2026-10-07-org-multi-org-director.md:153`。help: `internal/cli/org.go:55-58`、`:61`、`ralph org --help` の Available Commands に `followup`。文を固定するテストは `internal/org/skill_followup_test.go`(4 本)、`TestOrgFollowup_Help`、`TestOrgMergeOrder_HelpDescribesTheFollowups`、`TestStatusCmd_HelpDescribesReservationsAndFollowups` |
| AC7 `followup list`・status・merge-order の出力だけで、どの issue がどのブランチ待ちで、着手できるか・できないか・わからないか(理由)がわかる | 満たす | list は ISSUE・AFTER_BRANCH・STATE、status と merge-order の節は ID・ISSUE・AFTER_BRANCH・STATE と、理由と次の手を 1 文で言う DETAIL(merge-order は PR も)。probe の出力(evidence の 2 節)で、ready は「ask the human about issue #100 now, then run ralph org followup done f1」、waiting は「PR #12 is open」「has no PR yet」「merged before the follow-up was recorded ... does not count」、closed は「reopen PR #9 or open a new PR ... or ask the human」、unknown は gh の失敗と `mergedAt` が読めないことを書いた。status の waiting は「ralph org merge-order asks GitHub」と次の手を言う |

設計の決め(D1〜D5)との照合:

- D1: 満たす。`added` と `done` の 2 つの event(`followup.go:33-36`)、自分の lock、lock の中での読み直し、ID の採番、切れた行の前の改行、純粋な畳み込み(`foldFollowups`、`:352-370`)、open → done、2 度目の done は 0 で何も変えない。
- D2: 満たす(I2 あり)。status は gh を打たず `recorded`、merge-order は gh で `gh`。status の waiting の DETAIL が merge-order を名指す(`status_followup.go:31`)。merge-order の表には SOURCE の列がない(I2)。
- D3: 満たす。動詞と引数の検査(`followup.go:219-286`)、`list` の既定と `--all`・`--json`、`--org-id` を断る(`org_followup.go:219-221`)。note の規則は `ValidateInboxNote`(`internal/org/inbox.go:401-418`)と同じ順と判定で、文言だけが違う。
- D4: 満たす。`reservation` は常に配列、`followups` は空の道と中身のある道で同じ object(S4b で配列から object に変えたことは Progress に記録がある)。
- D5: 満たす(I1 あり)。記録より後の merge だけを数え、記録より前の `pr_merged` と head が同じ `worktree_cleaned` を数えない(M1、`followup.go:475-503`)。probe: 2020 の `pr_merged` と同じ head の `worktree_cleaned` の feat/clean-before は status で `waiting`、別の head の feat/clean-new は `ready`。`pr_merged` のない feat/clean-gh は status で `ready`、merge-order で `waiting` になり、文書にある例外のとおり割れた。割れたときの規則(ready と waiting なら merge-order、merge-order が unknown なら理由を添えて status)は、help 4 か所(followup、followup add、merge-order、status)、`/org`・`/director` skill、README、AGENTS.md、仕様の注、tech-debt の 409 行に同じ意味の文がある(evidence の 4 節。狭める前の言い方は 0 件)。add の help(`org_followup.go:73-75`)と `/director` skill の手順 1 は「merge 済みなら follow-up にせず、すぐ人に聞く」と書く。片付けの時刻の問題は tech-debt 409 行に送ってある。

計画の Progress には、Risks の「既存の検索結果を使い回す」を採らずブランチごとに gh を引くこと(S3)、JSON の `followups` を object にしたこと(S4b)が記録されている。AC の checkbox は `[ ]` のままで、/pr までに付ける運用どおり。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、68 秒) | 範囲は `Language scope: changed (changed_languages)`、`golang`。gofmt `ok`、go vet と staticcheck は無出力(`Skipping` の行なし)、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の構文、`jq -e`、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照もすべて OK。secret scan は `1cc84ffa..9a49f306` で clean。timeout には当たらず、打ち直しもしていない |
| `./scripts/check-template-purity.sh` | pass(rc=0) | 単独でも打った |
| `./scripts/check-skill-sync.sh` | pass(rc=0、14 本) | 単独でも打った |
| `./scripts/check-sync.sh` | pass(rc=0、DRIFTED 0、ROOT_ONLY 0) | 単独でも打った |
| `git diff --check origin/main...HEAD` | pass(無出力) | |
| 変えたファイルの U+FFFD の grep | 0 件 | |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-followups.md` | `3d00daf5cf87` | `- Approved:` の digest と同じ |
| `go build -o <scratch>/ralph ./cmd/ralph` | pass(rc=0、go1.27.3) | probe 用。前後で `git status --porcelain` は空 |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org --help`、`ralph org followup --help` と 3 つの動詞の help | はい | `--org-id` の help の動詞の一覧に followup が入った。help を固定する既存のテスト 3 本(approve、director、regate)も同じ文に合わせてある |
| `ralph status --help` | はい | `reserved:` の行、follow-up の節、`recorded`、gh を使わないこと、`merged_at` が片付けの時刻になりうること、`followups` の object、割れたときの規則 |
| `ralph org merge-order --help` | はい | 節の列、`followups` の key、状態ごとの条件、`unknown` の理由の列挙(コードの `FollowupMergeUnknown` の説明の 8 つから、CLI では起きない中断を除いた 7 つ。Progress の F2 の L6 のとおり)、終了コードを変えないこと |
| `/org` skill(4 つの写し) | はい | 動詞の表、古い台帳の節の動詞の分け方(add・done は変える側、list は読むだけ。`org_legacy_ledger.go:24-39` とコードの呼び分け `org_followup.go:97`・`:137`・`:188` に合う)、merge-order と status の行、「merge を待つ issue」の節 |
| `/director` skill(4 つの写し) | はい | 進捗の報告、merge の報告、「merge を待つ issue」の 3 手順 |
| README、AGENTS.md | はい | |
| 仕様 FR-9 の注(2026-10-30) | はい | `docs/specs/2026-10-07-org-multi-org-director.md:153` |
| 計画の D5 の本文 | いいえ(I1) | 下の Findings |
| `docs/tech-debt/README.md` 409・411 行 | はい | 409 の規則の文と 411 の行数(1322、360)は今と同じ |
| `docs/tech-debt/README.md` 410 行 | いいえ(V2) | 下の Findings |
| `docs/tech-debt/README.md` の Trigger に当たった行 | いいえ(V1) | 下の Findings |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt の Trigger にこの branch が当たったのに、`org-followups` の持ち越しの注がない行が 6 ある。177 (a)(`/org` skill の command table の次の編集。status と merge-order の行を伸ばし、3 行を足した)、235(`internal/org/lockfile.go` の次の変更。コメントだけ)、246(`mergeorder.go`・`org_merge_order.go`・`mergeorder_test.go` の次の編集)、251(`MergeOrder` の次の編集)、303(`mergeorder.go` の次の変更。この branch が足した 411 行と同じファイルの大きさの話だが、303 には触れていない)、359(`ralph status` の座席の表の次の変更。座席のない組で見出しを出さなくした `status.go:772`) | evidence の 6 節、各行の Trigger の欄 | /sync-docs で各行に「carried over by feat/org-followups」の注を足す。303 と 411 は重なるので、片方からもう片方を指すとよい |
| V2 | LOW | tech-debt 410 行の行数が古い。`status.go` は「959 lines」、`status_followup.go` は「187 lines」とあるが、今は 963 と 189。c9cb588f のときの数で、aae31317(R2-2 の help と説明)で増えた | `docs/tech-debt/README.md:410`、evidence の 5 節 | /sync-docs で 963 と 189 に直す |
| I1 | INFO | 計画の D5 の本文は「2 つの道が食い違ったら merge-order(gh)の答えを使う」のまま。実装と文書は R2-1 で「ready と waiting で割れたら merge-order、merge-order が unknown なら理由を添えて status」に狭めた。狭めたことは Progress の F3 の記録にしかない。D5 は承認の digest に入るので、直すと承認し直しになる | `docs/plans/active/2026-10-30-org-followups.md:56`・`:124` | 記録済みの逸脱として扱い、PR の本文で触れる。D5 を直すなら承認し直す |
| I2 | INFO | D2 は「表と JSON にどの道で判定したか(status は `recorded`、merge-order は `gh`)を出す」と書くが、merge-order の表には SOURCE の列がない(列は ID・ISSUE・AFTER_BRANCH・STATE・PR・DETAIL)。JSON の `source` は `gh` で、AC3 が求めるのは `--json` の判定の道なので AC は満たす。help と skill は今の列のとおりに書いてあり、Progress には逸脱としての記録がない | `internal/cli/org_merge_order.go:273-287`、計画 D2(`:55`) | 任意。Progress に「merge-order の表は PR の列で道を示す」と記録するか、列を足す |

### Observational checks

HEAD から build した binary を、scratchpad の一時 repo(bare origin と main)で打った。`env -i` で HOME と git の設定を一時のものにし、PATH は `remote get-url origin` にだけ `https://github.com/acme/app.git` と答える git の包みと、`--head` ごとに決めた JSON を返す偽の gh だけにした。ネットワーク、本物の gh、herdr、agmsg は使っていない。出力は evidence の 2 節にある。

- AC1〜AC8 の probe は上の表のとおり。
- `--note ''` は note なしとして受け付けられた(`followup.go:226-228` のとおり)。`--note ' '` は「the note is blank: give some text or leave it out」で断られるので、空白だけの note と空の note の扱いが違う。help は「--note is optional」と書き、空の値には触れない。動きとして困ることはないので所見にはしない。
- 記録がディレクトリのとき、`followup list` は rc 1 で読めない理由を出し、`ralph status` は stderr に警告を出して rc 0 で続けた(help のとおり)。
- 機能のブランチを結びつけた台帳でも、merge-order は表と注記のあとに同じ follow-up の節を出した。
- `ralph org cleanup` は `MergeOrderParams` に `Followups` を渡さない(`internal/cli/org_cleanup.go:167`)。gh の呼び出しを `ghBranchPRs` にまとめたあとも、cleanup と merge-order の対象のブランチの `gh pr list` の引数は前と同じ(`mergeorder.go:970` 以下、fields は `mergeOrderGHFields`)。

### Coverage gaps

- Go のテスト(`go test`)と `./scripts/run-test.sh` は打っていない(/test の範囲)。AC4 の「今の gh なしのテストが通る」と AC6 の「文を固定するテストが通る」は、テストの存在と中身を読んだところまで。
- lock の取り合いは、20 の add を同時に打って ID が重ならないことで見た。`followups.lock` をほかのプロセスが持ったまま待つ場合の期限(`manifestLockTimeout`、5 秒)は、コードとテスト `TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites` の存在で確かめた。
- 本物の GitHub と gh の `mergedAt` の形、gh の 1000 件の上限、fork の PR、origin と別の repository の PR は probe していない。テスト(`mergeorder_followup_test.go` の 13 本)の存在で確かめた。
- tech-debt の Trigger の掃き出しは、ファイルと関数の名前の部分一致で引いた。語で書かれた Trigger(出来事)は、引けた行だけを読んだ。

### Verdict(cycle 1)

pass

- Verified: AC1〜AC8 と D1〜D5 は、コードの該当箇所と、HEAD の binary の probe か grep で確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync、check-template-purity、check-pipeline-sync、secret scan、`git diff --check`)はすべて通り、計画の digest は承認時と同じ。
- Partially verified: AC4 と AC6 のテストが通ること、lock を持つ別のプロセスとの取り合い、本物の gh の出力。
- Not verified: Go のテストの結果(/test)。
- 所見: LOW 2 件(V1 tech-debt の Trigger 6 行、V2 410 行の行数)、INFO 2 件(I1 計画の D5 の本文、I2 merge-order の表の SOURCE)。どれも merge を止めない。V1 と V2 は /sync-docs に回す。
- Insight event: `docs/insights/events/2026-10-31-org-followups.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 2`)。ファイル名は UTC の日付。
