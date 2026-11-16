# sync-docs report: reseat-notices

- Date: 2026-11-16(JST では 11-17 の 0 時台)
- Plan: `docs/plans/active/2026-11-16-reseat-notices.md`(承認の digest `2aa9b9bde56c`。`./scripts/plan-visual.sh digest` で計算し直した値と今も一致する)
- Branch: `fix/reseat-notices`。base は origin/main `30e122a1`、run 開始時の HEAD は `f2e54adc`(verify と test の記録の commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は `56992f8e`(初回)、`edca8ac7`(S5 の見直し)、`8c6b60e0`(記録。Merge yes)。verify は `61689558` で partial pass。test は `3884a8c3` で pass、あとの S7 は `abb27824`。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD` と、verify の V1 から V4 と INFO、test の G1 から G6。tech-debt の全行の Trigger を自分で走査した(下の「Trigger の走査」)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit が足す前の HEAD `f2e54adc` の値): `git diff --shortstat origin/main...HEAD` は 22 ファイル、+1,108/-105。内訳は docs が +487/-2、テストの Go が +453/-34、残りの 11 ファイル(Go の本体、skill、`director.md`、`.codex/README.md` など)が +168/-69。この commit は docs だけを足す
- 制約: Go のコードは触っていない(依頼どおり)。テストは回していない。`scripts/check-sync.sh` と `scripts/check-skill-sync.sh` は回した(下の「確認」)

## Summary

verify の V1 から V4 を直した。V1 は `README.md` の hook の行の動詞に reseat を足した。V2 は tech-debt 513 行目の (i) を払ったと書いた。V3 は 513、514、515、480 の 4 行に持ち越しの note を足した。V4 は 479、480、515 の動詞の一覧と今の状態を言う文を合わせた。INFO は 468 行目の「three」を 1 語直し、ファイル先頭の comment は直さなかった。

走査で、V3 の 4 行のほかに、515 行目の (n) の Trigger も満たしていた(同じ行の note に足した)。ほかの候補は満たしていない。

test report の G1 から G3 は S7 で閉じているので、行は足していない。G4 は 515 行目の (e) に 1 文足した。G5 と G6 は既存の項目(515 の (i)、480 の (c))が持っていて、新しい行は足していない。

## Changes made

| File | Change |
|------|--------|
| `README.md` | `ralph org` の行の hook の行の文。動詞の一覧に `reseat` を足し、「the agent they launch」を「an agent they launch」にし、末尾の括弧に「`reseat` は立てる codex の座席ごとに 1 行を、どの座席も止める前に、`--dry-run` でも出す」を足した。下の「V1」 |
| `docs/tech-debt/README.md` | 468、479、480、513、514、515 の 6 行。ほかの行は変わっていない。下の「tech-debt」 |
| `docs/plans/active/2026-11-16-reseat-notices.md` | Progress に sync-docs の 1 行。計画の本文と AC の箱は変えていない(digest は同じ) |
| `docs/reports/sync-docs-2026-11-16-reseat-notices.md` | この report |

## verify が挙げた drift

### V1(MEDIUM): `README.md` の hook の行

直した。reseat は `start` と同じく「codex の agent を立てる前に出す」動詞で、立てる座席が複数なので 1 行ずつ出る。同じ行の雛形の版の文は `reseat` を挙げていたので、片方だけ古かった。文面は `.codex/README.md` の 2 写しの文(`ralph org reseat prints one for each codex seat it starts`)と同じ向きにした。

### V2(LOW): tech-debt 513 行目の (i)

(i) は L14「reseat は hook の行を出さない」。この PR が `b21c75c7`(`ReseatParams.OnPlan`)、`72cc7d72`(CLI の呼び出し)、`0d7dfe1f`(`.codex/README.md` ほか)で払った。ほかの行の閉じ方に合わせた。

- Debt: `~~(i) …~~ Closed in fix/reseat-notices (commits …)`。実際の直し方が行の提案と違う(提案は `printOrgReseatDryRun` と `printOrgReseatResult` に 1 行ずつ出させる。実際は `OnPlan` の callback)ので、その違いと、行が出る時(本物は印の書き込みのあと、最初の止めの前。dry run は計画のすぐあと)を書いた。確かめていないこと(印の書き込みの失敗、本物の herdr と codex)も、それを持つ行を名指して書いた
- Impact: `~~(i): …~~ (closed)`
- Why deferred: 末尾に「(i) was later done inside fix/reseat-notices, whose plan is about that note」
- Trigger: `~~(i) …~~ Done in fix/reseat-notices (commits …)`。Trigger が求めていた `.codex/README.md` の 2 写しへの reseat の追加も済んでいると書いた
- Related: 計画を `docs/plans/active/2026-11-16-reseat-notices.md` で足した(`archive-plan.sh` が `/pr` で `archive/` に書き換える。先には書かない)

513 行目の過去の日付つき note(2026-11-15 の「(i): ffd32109 and b568ca8c changed `runOrgReseat` … neither prints the codex hook-trust note」など)は、その日の記録なので変えていない。

### V3(LOW): 満たした Trigger の持ち越し

4 行とも、Trigger のセルの末尾に日付つきの note、Related のセルの末尾に計画を足した。数は `wc -l` で測り直した(origin/main `30e122a1` と、Go を最後に変えた commit `abb27824`。HEAD も同じ数)。

| 行(先頭の語) | 当たった所 | 書いたこと |
|---|---|---|
| 513「Fourteen small findings of docs/spec-org-reseat」(f)、(j)、(m)、(n) | `orgReseatHelp` の dry run の段落の文と、末尾の hook の段落。`TestOrgReseatHelp_…` の pin | どれも払っていない。help は (f) の `ralph org status --org-id <id>` を今も言わず、(j) の `Spawn` の error の `ralph org stop` に `--config` が付かないことも、(m) の `reseated` を書かない reseat のことも、(n) の ragged line もそのまま。新しい文は 4 項目が名指す段落にない。(i) の Trigger も同じ commit で満たし、払ったと書いた |
| 514「`internal/org/reseat.go` (1,307 lines)」 | `reseat.go` 1,553 → 1,574、`reseat_test.go` 1,868 → 2,004、`internal/cli/org_reseat_test.go` 927 → 954 | 持ち越し。move-only の分割は `/self-review` から回し直すことになり、計画の Scope にもない。`internal/cli/org_reseat.go` は 527 → 554 でガイドの下。`feature.go`(874)と `startFeatureInput` の周りは触っていない |
| 515「Test gaps of docs/spec-org-reseat」(h)、(i)、(j)、(n) | (h) は S4(`f18738bd`)が `/director` の「座席を立て直す」を編集。(i)(j) は `runOrgReseat` と `scaffold_version_notice_cli_test.go`。(n) は `orgReseatHelp` | (h) と (j) と (n) は払っていない。(i) は help の半分だけ払った(`TestScaffoldVersionHelp_EachVerbDescribesTheNote` と `TestCodexHookTrustHelp_EachVerbDescribesTheNote` が reseat を持つ)が、reseat を走らせて雛形の版の行と `model_pool` の行を確かめるテストはない。(n) は handoff の 3 項目(h)(i)(j)にない。`orgReseatHelp` を変えたので当たっていた |
| 480「Test gaps of feat/codex-hook-trust-check」 | `codex_hook_trust_notice.go`(`printCodexHookTrustNotice` と `codexHookTrustAgent` の doc、`orgCodexHookTrustNoticeHelp`)と、`codex_hook_trust_notice_test.go` | 持ち越し。行の文面と読む物は変えていない。(a) と (b) の fixture はこの PR の Scope(reseat の行)にない。Related に計画を足した |

### V4(LOW): 今の状態を言う文

| 行 | 直した所 |
|---|---|
| 515 (i) | 「lists the other four verbs, and the help paragraph of `reseat` is pinned only by `TestOrgReseatHelp_…`」を、reseat が `0d7dfe1f` から一覧に入り、help の段落は 2 つのテストが pin すること、残るのは reseat を走らせるテストだけであることに書き換えた |
| 479 (a) | 「in the launch note of `ralph org spawn`, `start`, and `director start`」に `reseat` を足した。reseat の行も同じ関数が出すので、同じく出続ける |
| 480 (c) | 「The launch note of `org spawn`, `start`, and `director start` was not run against a real herdr」に `reseat` を足した。reseat の行のテストも stub の herdr で走る |

### INFO

- I1(468 行目の「three `--help` texts」): 「four」に直した。`orgScaffoldVersionNoticeHelp` は start、director attach、director start、reseat の 4 つの help が持つ(`org.go`、`org_director.go`、`org_director_headless.go`、`org_reseat.go` で確かめた)。この行の (b) の Trigger を満たしたかは、verify の読みに従って満たしていないとした。定数の値は変わらず、変わったのは doc comment の動詞の一覧だけで、(b) が求める「fallback を言う句」には触れていない。comment だけの変更も hit と数える読みなら、1 文の note を足せる(落としてよい)
- I2(`scaffold_version_notice_cli_test.go` の先頭の comment): 直していない。comment は、このファイルのテストが走らせる動詞(org start の 2 つの形、director attach、director start、status)を挙げている。reseat を走らせるテストはこのファイルにないので、reseat を足すと偽になる。足すなら「reseat は help だけ」という文が要り、1 語では済まない。help の側は、同じファイルの `TestScaffoldVersionHelp_EachVerbDescribesTheNote` の doc comment が reseat を挙げている
- I3: 記録だけ。何もしていない

## test report の穴

| 穴 | 扱い |
|---|---|
| G1、G2、G3(LOW) | S7(`abb27824`、`TestCodexHookTrustNotice_ReseatNamesEachCodexSeatAtTheWorktree`)で閉じている。tech-debt の行は足していない |
| G4(INFO、印の書き込みの失敗で `OnPlan` を呼ばない経路) | 515 行目の (e)(I/O の失敗の return のテストがない)が、同じ箇所(`reseat.lock` の印の書き込み)を挙げている。新しい行も文字も足さず、(e) に 1 文足した(「since fix/reseat-notices the marker's write is also where `Reseat` does not call `OnPlan` …」)。Related に test report の G4 を足した |
| G5(INFO、reseat を走らせて雛形の版の行を確かめる CLI のテストがない) | 515 の (i) の残り半分と同じ。V4 の直しでその半分が (i) の今の状態になった。新しい行は足していない |
| G6(INFO、本物の herdr と codex で走らせていない) | 480 の (c)(V4 で reseat を足した)と、515 の (g)(live の確認)が持っている。新しい行は足していない |

## Trigger の走査

handoff は 4 行(513、514、515、480)と 479、480、515 の文、468 の INFO を挙げた。tech-debt の全行から、Trigger が全部 `~~` の行を除き、Trigger のセルを、この PR が変えた物(Go の 9 ファイル、関数と定数の名前、`/director` skill、`director.md`、`.codex/README.md`、`README.md`、`docs/tech-debt/README.md`)の語で探し、候補の Trigger を読んだ。handoff にない新しい hit は、515 の (n) だけだった(V3 の表)。ほかの候補は下の表のとおり当たっていない。

| 行(先頭の語) | 判断 | 理由 |
|---|---|---|
| 135「`checkCodexAgmsgWritableRoot`」、153「codex ignores `[profiles.*]`」 | 当たっていない | Trigger は `doctor_codex_writable_root.go` と `.codex/config.toml` の変更。どちらも触っていない。「hook trust」の語は、前の branch の note にある |
| 195「A `ralph` of another version」、201「Test gaps left by org-feature-worktree」 | 当たっていない | Trigger は `newOrgStartCmd` の RunE と `runOrgStartPlan`。どちらも編集していない。`printCodexHookTrustNotice` は前の branch の note に出る |
| 262「Three test gaps of the org-director-attach PR」 | 当たっていない | Trigger は `/director` の受信箱の節(`受信箱の処理`)。この PR の編集は「座席を立て直す」と、「進捗の報告」の「人に向けた知らせの行」で、どちらも別の節(見出しは `1469`、`1583`、`1612` 行目) |
| 365「`internal/org/verbs.go` is 2277 lines」 | 当たっていない | Trigger は `Send` と leader の雛形のテスト、6 ファイルの 300 行の増加。`prompts_notice_test.go` の変更は director の prompt と skill のテストで、547 → 593 行。6 ファイルは触っていない |
| 380、381、421、422 | 当たっていない | send の確かめの手順、wait の終了コード、`skill_split_questions_test.go`、`--limit`。どれにも触っていない |
| 465「The `/director` skill … do not say what a director does when it sees the scaffold version note」 | 当たっていないとした | Debt が `~~ RESOLVED 2026-11-13` で、払う物が残っていない |
| 468「Two small findings of feat/scaffold-version-notice」 | 上の I1 | (b) は定数の doc comment だけで、値は同じ |
| 482「Test-structure findings of the codex-hook-trust-check self-review」 | 当たっていない | Trigger は notice の行の文面と `codex_hook_trust_test.go` への次のテスト。行の文面は変えず、触ったのは `codex_hook_trust_notice_test.go` |
| 504、508「Test gaps of docs/director-notice-handling」 | 当たっていない | 504 は note の行そのものの末尾の変更か `####` の見出し。508 は `HOOK_SEAT:` の DECISION の文。この PR が変えたのは動詞の一覧と、行のあとに続く文だけ |
| 361、516、520、524、526、457 | 当たっていない | `Stop`、`Disband`、`takeReseatLock`、leader の規則の節、`resume` の関数。どれも編集していない |
| 118、180、371 | 当たっていない | `quality-gates.md`(template 側)、`/org` skill の command table、5 つの recipe。どれも編集していない |

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md`(`internal/org/` と `internal/cli/` の行) | 変えない。`internal/cli/` の行は subcommand の一覧、`internal/org/` の行は機構の層(reseat の項は `reseat.go` と `reseat_progress.go`)で、どの動詞が stderr に何を出すかを書いていない。`hook trust`、`scaffold version` の語は 0 件。計画の Verify plan の問いへの答えは「書かない」 |
| `CLAUDE.md` | 変えない。どちらの行も出てこない |
| `docs/specs/2026-11-12-org-reseat.md` | 変えない。stderr に出す人向けの行を 1 つも書いていない(`hook` の語は 1 項だけで、leader が上げる hook の QUESTION の扱い。reseat 自身の行の話ではない)。reseat が出す雛形の版の行(#312c 4 回目から)も無いので、hook の行だけを足すと片方だけになる |
| `/org` skill の reseat の行と「座席の立て直し」節 | 変えない。2 つの行を挙げていない(spawn と start の行も同じ)。dry run の段落は watch と herdr の `note:` だけを言い、「ほかに出さない」とは言っていない |
| `docs/quality/quality-gates.md`(2 写し) | 変えない。「Cross-org watch gate」の行は reseat を挙げている。この PR は gate を変えない |
| `.codex/README.md` と `templates/base/.codex/README.md` | `0d7dfe1f` で reseat が入っていて、`check-sync.sh` で一致 |
| `/director` skill の 4 写しと `director.md` | verify が確認済み。S4(`f18738bd`)で手順 2、4 と小節の reseat の項に文が入っている |
| `docs/architecture/repo-map.md` | 変えない。新しいファイルも script もない |
| `internal/upgrade/notes/8.2.0.md` | verify が確認済み。次の版の注記は計画の Non-goals |

## 直さなかったこと

- `scaffold_version_notice_cli_test.go` の先頭の comment(上の I2)
- 468 行目の Trigger (b) への note(上の I1。足すなら 1 文)
- 513 の (f)、(j)、(m)、(n) と、515 の (h)、(j)、(n)、480 の (a)、(b) は払っていない。この PR の Scope にない
- 計画の Acceptance criteria の箱は `[ ]` のまま。AC の記録は `/pr` の前の手順に任せる
- 過去の日付つきの note、report の中の古い数字

## 確認

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- tech-debt の計画の参照: `docs/plans/(active|archive)/…` の参照はすべてその場所にある(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索を手で回した)
- tech-debt の編集: HEAD と突き合わせ、変わった行は 468、479、480、513、514、515 だけ。どの行も 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし
- 計画の digest: `2aa9b9bde56c`(変わっていない)

## Insight event

`./scripts/insights-append.sh --slug reseat-notices --flow standard --phase sync_docs --verdict pass --cycle auto --source skill --events-dir <scratch>` で作業用のディレクトリに書いた。この run の前の phase(self-review、verify、test)も `docs/insights/events/` に足していないので、同じ扱いにした。行は次のとおり(`/pr` の前に、ほかの phase の行と順に足す)。

```
{"schema":1,"ts":"2026-11-16T15:40:22Z","slug":"reseat-notices","flow":"standard","phase":"sync_docs","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```
