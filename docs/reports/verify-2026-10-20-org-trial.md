# Verify report: org-trial

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-org-trial.md`(仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 9 段目、FR-12。Status: Approved、`plan-visual.sh digest` は `5b82ee1e30a1` で `- Approved:` 行と一致)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD 8e1d5c40、27 ファイル)。記録だけの変更で、Go のコードと script は変わっていない。証拠 `docs/evidence/org-trial-2026-10-20.md` と snapshot `docs/evidence/org-trial-2026-10-20/`(21 ファイル)、仕様の FR-12 の印と注と Open questions、`docs/tech-debt/README.md` の 20 行(表の行 19 と RESOLVED のコメント 1)、計画、self-review の報告、insight event。AC1〜AC10、計画の Verify plan、FR-12 の本文
- Evidence: `docs/evidence/verify-2026-10-20-org-trial.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-20-091143.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)

## Spec compliance

証拠の 17 個の `sh` block は、worktree の root で 1 つずつ `sh -c` で走らせ、出力を直上の表の本文か直下の `text` block と 1 行ずつ比べた(scratchpad の `recompute.py`、`python3 -I`)。17 個とも rc 0 で、すべて一致した。比べ方の対照として、期待値を 2 か所(e2 の待ち 423 を 424、`resolved 6` を 7)変えた写しでは、その 2 block だけが不一致になった。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(setup): binary の commit、道具の版、分離(D2)、shim(D3)、model(D4)。`$HOME` の道、認証情報、pane id、session id、乱数の hex を含まない | 満たす | 証拠 109〜186 行。binary は f0668004(`git log` で PR #240 の merge commit)、herdr 0.7.5・Claude Code 2.1.291・agmsg 1.5.1、`env -i` の変数の一覧、shim の動き、model の表(block 2 で再計算)。漏えいの grep は 0 件(Observational checks)。長い hex は 15 種で、9 つは repo にある commit、5 つは plan の digest と `content=` の sha256、残る 1 つ(36d3491f)は regate 1 で戻された merge commit で、証拠の Merges が「only the reflog has it now」と書く。7〜11 桁の hex で commit でないものは 36d3491f と e8a0d3e3(F11 の push されなかった WIP commit)の 2 つで、どちらも証拠が名指す commit |
| AC2(two orgs): 承認した 1 つの分割計画から 2 つの org、走った期間が重なる、各 1 本の PR、証拠が両方を名指す | 満たす | block 3 が `scope_reserved` の `split=trial digest=0ffd32661606` を両 org に出し、重なりは 9370 s(2 h 36 min 10 s、手計算で一致)。block 4 で分割計画と 2 つの機能計画の `- Approved:` 行。`gh pr view`(読むだけ)で #241 は `fix/report-escape`、#242 は `fix/doctor-escape`、どちらも MERGED。変更したファイルの一覧も証拠 259〜267 行と一致 |
| AC3(merges): 2 本とも `ralph org merge`(1 本目は人に頼まれて、2 本目は `ralph org regate` のあと `auto_merge` で)、または断りの理由の記録 | 満たす | block 5 の `pr_merged`: #241 は `auto_merge=false`、merge commit a9e8ea04、#242 は `auto_merge=true`、a1ce47d8。gh の `mergeCommit` も同じ 2 つ。block 14 の regate 2 件(1 件目 `test_failed`、2 件目 `gate_passed`、どちらも `via=regate`)。順序は I6(06:09:00Z)→ #241 の merge(06:10:59Z)、I7(06:20:18Z)→ regate 2 の `gate_passed`(07:09:51Z)→ #242(07:14:16Z) |
| AC4(numbers): 介入、承認待ち、director の扱い、費用、停止と再開の表。各数字の出どころ。台帳の数字は snapshot から `jq` で再計算できる | 満たす | 介入は block 6、承認待ちは block 7、受信箱の扱いは block 8・9、費用は block 13、停止と再開は block 15〜17。本文の数字も手で確かめた: manifest 91 行・escalation 30 行・shim 49 行、1302 s = 21 min 42 s、opus の 1h の cache write 491,330 と sonnet 382,876(`seat-usage.tsv` の和)、e1 の 49 s と e2 の 391 s と 888 s、43 s、23 件と 7 件の内訳、DECISION の 48 min と 97 min と 49 min、R2 の期限 07:21Z。単価は `watcher-usage.txt` と awk の表と本文の 3 か所で同じ。表の外の数字(load average、終了コード)は「notes」と出どころを書いている |
| AC5(stop and restart): org の止め直し(D6 の 1 つ目)と見張りの止め直し(2 つ目)の各項目と、期待どおりか | 満たす(1 つ目は形を変えて実施) | 証拠 580〜676 行。2 つ目: 停止 05:48:38Z から次の wait まで、manifest・escalation・受信箱・shim の 4 つとも 0 件(block 15)、`watch.log` に開始の行が 2 つ、`watch-alive.json` の pid 33773 は 05:49:08Z 開始。1 つ目: stop から disband までの manifest(block 16)、doctor-escape の受信箱は e6 が最後で `plan_approved` は org ごとに 1 件(block 17)。止めた時機が計画の D6 と違うことは計画の Progress の注、証拠の Part 1 の冒頭、仕様の新しい Open question の 3 か所に書かれている。終了コードは director のメモにだけある(Coverage gaps) |
| AC6(defaults): 4 つの既定ごとに、決めたことと試行の数字。変えたなら 3 か所とテスト | 満たす | 仕様 101 行の注: `max_orgs` 10 と `max_total_seats` 30 は 2 org・各 3 席、`ack_minutes` 10 は ack までの最長 213 s で期限切れ 0、`resolve_minutes` 60 は resolve までの最長 997 s で期限切れ 0、4 つとも変えない。数字は block 8 の表と一致。`internal/config/config.go` 550〜571 行の `Default()` は 10・30・10・60 のままで、diff に Go と `ralph.toml` と `ralph-config.sh` はない。manifest の `seat_id` は leader・implementer・reviewer の 3 つだけ |
| AC7(spec): FR-12 の印、Open questions の 2 項目に答えるか注を指す | 満たす(V2) | 仕様 100 行が `- [x]`。246・247 行の 2 項目は答えと「(FR-12 の注)」を足し、248 行に止める時機の問いを足した。FR-1〜FR-11 の印の決まりとの関係は V2 |
| AC8(tech-debt): ruleset の行は `ralph org merge` が merge したなら解消、2 本の PR が直した行は PR 番号つきで解消、直していない所見ごとに行 | 満たす | ruleset の行(今の 298 行のコメントと 299 行)は register の書き方(`<!-- RESOLVED … Row preserved for traceability. -->` と打ち消し)で解消。217 行 (a) は PR #241、246 行 (h) は PR #242 を merge commit つきで挙げ、`TestBuildOrgReport_EscapesPipesInEventDetails`(`internal/org/report_test.go:95`)と `TestMergeGuaranteeResult_EscapesControlCharacters`(`internal/cli/doctor_merge_guarantee_test.go:338`)は HEAD にあり、`doctor_merge_guarantee.go:90` は Detail 全体を `org.PrintableInboxText` に通す。所見は F1・F2 が 313 行、F3〜F8 が 314〜319 行、F9 が 304 行、F10 が 309 行、F11〜F14 が 320〜323 行、R1 が 324 行で、14 件と R1 のどれにも行がある |
| AC9(cleanup): 決まった順で片付け、利用者の herdr server に触れない、shim の log がすべての `osascript` を持つ | 記録はある(実行は確かめられない) | 証拠 868〜902 行の 7 段は AC9 の順と同じ。snapshot はその順と矛盾しない(`director.jsonl` に `detached` がなく、`watch-alive.json` は 07:16:24Z に `running`、つまり detach と watch の停止の前に写した)。各段の終了コードと `pgrep`・`lsof` の結果は director のメモにだけある。AC9 の `not_running` は、`ralph status` の文の形(`internal/cli/status.go:335` の `org watch: not running`)で証拠に引かれている |
| AC10(snapshot): D12 の記録、`jq` が表を再現、`/Users/`・pane id・session id・認証情報がない | 満たす | 17 block が一致。gate の記録は manifest の `gate_passed`・`gate_failed` で、`gate_record.go` の言うとおり別のファイルはない。`pane_id` は `pane-1`〜`pane-10` と `ws-1`・`ws-2` だけ、watch-status の `agent_session` は `session-1`〜`session-6`、`tab_id`・`terminal_id`・`workspace_id` も label だけ、`by=` は `user-1`、gate の `herdr_pane=` は `set` か `unset` |

FR-12 の本文(「2 つの org を同時に走らせ、人の介入回数・介入時間・承認待ちの時間・総費用・停止と再開の成否を記録する。その結果で上限と期限の既定を見直す」)の 7 項目は、すべて証拠の表と仕様の注にある。総費用は座席と watcher の $12.81 で、director の分は分けられないので外したと 3 か所(証拠 548〜550 行、仕様の注、証拠の Not verified)に書いてある。仕様の注の数字(介入 7 回と 1302 秒、承認待ち 109 秒と 423 秒、$12.81、終了コード 4、43 秒、213 秒、997 秒、F1〜F14)は証拠の数字と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh`(HEAD 8e1d5c40) | exit 0、`All verifiers passed.` | 言語の範囲は `changed (docs_only)` で language pack は選ばれず、Go は走っていない。shellcheck、hook の `sh -n`、guard の awk の構文、settings.json の jq、check-sync(DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照が OK。branch の secret scan は `a1ce47d8..8e1d5c40` で clean |
| register の欄の数の awk(依頼の 1 行) | `22: 7`、`32: 5`、`278: 10` の 3 行だけ | 3 行とも base にあった行。変えた表の 19 行はどれも 6 本の区切り(5 欄) |
| `git diff --check origin/main...HEAD` | exit 2 | `gate-excerpts.md` の 4 行の行末の空白(V3) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-20-org-trial.md` | `5b82ee1e30a1`、rc 0 | `- Approved:` 行と一致。承認のあとの変更は Progress の節だけ |
| 証拠の 17 block の再計算(`python3 -I recompute.py`) | rc 0、17/17 一致 | 期待値を変えた写しで 2 block が不一致になることも確かめた |

計画の Verify plan の `go vet` と `gofmt -l` は「既定を変えたとき」の条件つきで、既定は変わっていないので走らせていない。

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 証拠の所見が引くコード | 合う | `internal/org/watch.go:404`(`exec.CommandContext(ctx, "osascript", …)`)、`watch.go:1557`(`_ = w.escalateFn(…)`)、`escalateAlert` は director の設定を読まない(1536〜1562 行)、`internal/org/verbs.go:36` の `sendDetailsMaxLen = 200` と 654〜658 行の `text[:sendDetailsMaxLen]`、`internal/org/report.go:180` は `\|` だけを置き換える、`featureLeaderTask` は `f.Body` を足す(`internal/org/feature.go:399`)、`compensatePaneCtx` は C-c を送るだけで pane を閉じない(`internal/org/spawn.go:2348`)、`scripts/check-sync.sh` 68〜76 行の除外に `review-` と `org-manifest-` がない、`.claude/hooks/session_end_summary.sh:38` の WIP commit、`packs/languages/golang/verify.sh:109` の `go test ./...` に `-timeout` がない、`internal/org/gate_run.go:50` が除くのは `RALPH_VERIFY_` と `HARNESS_VERIFY_` だけ |
| 証拠の gate の行の参照 | 合う | `gate-excerpts.md` に、元の log の 389〜406 行(ROOT_ONLY の 2 件)、5083〜5088 行(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`)、601.405 s と 602.724 s の timeout と、そのとき走っていたテスト名がある |
| 証拠の Snapshot の表の `reports/` の行(40 行)と F14(832〜842 行) | ほぼ合う(V1) | doctor-escape の報告は 04:44:51Z までの event しか持たない |
| 仕様 FR-12 の印と FR-1〜FR-11 の印の決まり | 印は計画どおり(V2) | 74・75・78・97 行は「ほかの FR と同じく付けない」 |
| tech-debt の変えた行が指す道と節 | 合う | 引く道はすべてある(`docs/plans/archive/2026-10-20-report-escape.md`・`2026-10-20-doctor-escape.md`、`internal/org/driver/agmsg.go`、`internal/org/prompts/leader.md` の「agmsg の受信箱の運用」の節(379 行)ほか)。証拠の節の名前(Setup、Findings、Merges、Numbers、Gates、Trial deviations、Cleanup、Snapshot、Not verified)は証拠の見出しにある。行の中の相互参照(「the `check-sync.sh` row above」「the WIP commit row below」「the `truncateForDetails` row above (F12)」)は、313〜324 行の並びで向きが合う。行番号で行を指す形は 0 件 |
| tech-debt の解消の行 | 合う | 217 行 (a) と 246 行 (h) は Debt・Impact・Trigger・Related の各欄で打ち消しか「done in PR #…」になり、246 行の 1 文目の総覧から (h) が外れている。ruleset の行は 298 行のコメントと 299 行の 2 欄の打ち消しで、register の既存の書き方と同じ |
| 計画が新しい行で `docs/plans/active/2026-10-20-org-trial.md` を引くこと | `/pr` で直る | `scripts/archive-plan.sh` が `/pr` で register の参照を archive に書き換える(54〜115 行)。証拠 3 行の参照は書き換えの対象外(V5) |
| `/director` と `/org` の skill | 変わっていない | 計画の Verify plan のとおり、diff に skill はない |

## Observational checks

- 漏えいの grep(証拠と snapshot の全ファイル): `/Users/` 0 件、`/home/`・ログイン名 0 件、UUID の形 0 件、`/var/folders`・`/private/var` 0 件、認証情報の形(`gh[pousr]_`・`github_pat_`・`sk-`・`Bearer`・`AKIA`・`xox`)0 件、herdr の pane id の形(`w<N>:p<M>`、`p_`・`w_`)0 件。メールの形は clone の URL の `git@github.com` だけ。`ORCA_TERMINAL_HANDLE` は名前だけで値はない
- branch の diff 全体の `/Users/` は 4 行で、計画 326・376 行と self-review の報告 13・65 行が grep の対象として書いた文字列だけ
- watch-status の `agent_get` の中の id の形の欄(`*_id`、`agent_session`)を `jq` で全部出した。値はすべて label で、元の id は残っていない
- 証拠の時刻と台帳の対応: I2 の答え 03:24:50Z と `plan_approved` 03:25:02Z、I3 の 03:30:54Z と 03:31:22Z、I6 の 06:09:00Z と #241 の `pr_merged` 06:10:59Z。GitHub の `mergedAt`(06:10:57Z、07:14:15Z)は `pr_merged` の 1〜2 秒前で、read-back のあとに記録する順と合う
- F13 の対応づけ: doctor-escape の `leader_is_anomaly_subject` の escalation は 12 件、shim の通知は 11 件。時刻順に並べると、03:48:26Z の 1 件の後の通知は 03:53:45Z で、03:53:30Z の escalation の 15 秒後になる。証拠の説明と一致する
- F12: U+FFFD の JSON の escape を持つ `sent` は manifest の 4 行(04:09:23Z、04:44:51Z、04:58:07Z、05:37:13Z)。報告の U+FFFD は doctor-escape の報告に 2 行、report-escape の報告に 1 行
- self-review の再 review の新しい LOW(N1、216 行の Trigger と Related)は 8e1d5c40 で直っている。Trigger は「the first feature org run with a codex leader, or a run whose leader's PNG reading can be checked」になり、Related に証拠の節が入った

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 証拠の Snapshot の表は org の報告を「written by its leader before the disband」と書く。doctor-escape の報告の event は 04:44:51Z で終わり、Roster は reviewer を active のまま示す。書かれたのは 04:44:51Z から gate 1 の開始(04:58:53Z、check-sync がこのファイルを ROOT_ONLY と出した)までの間で、disband(06:13:20Z)の約 1.5 時間前になる。そのため doctor-escape の 6 件の `sent` のうち 2 件(05:00:57Z、05:37:13Z)、gate、PR、D6 の 1 つ目、disband がこの報告にない。F14 は「All 11 `sent` events」と「5 in report-escape's, 4 in doctor-escape's」を並べるが、足して 11 にならない理由を書いていない。数字と判定は変わらない | 証拠 40 行と 832〜842 行。`reports/org-manifest-doctor-escape-2026-10-20.md` の Event timeline の最後の行。`gate-excerpts.md` 21 行。生ログの `## org reports` | Snapshot の表か F14 に「doctor-escape の報告は 04:44:51Z までの event を持つ(gate 1 の前に leader が書いたもの)」と 1 文足す。証拠は `docs/reports/` の記録ではないので、直すなら record-only の例外の外になる。直さないなら known gap として残してよい |
| V2 | LOW | 仕様の FR-12 だけに印が付いた。FR-7・FR-8・FR-10 の注は「FR-1〜FR-6 と同じく付けない」「ほかの FR と同じく付けない」と、FR の印を付けない決まりを書いている。FR-12 の注は、FR-12 だけ印を付けた理由を書いていない。読み手は FR-1〜FR-11 がまだ終わっていないと読みうる。AC7 は印を求めているので、計画との食い違いではない | 仕様 74・75・78・97・100・101 行 | FR-12 の注に「FR-12 は試行の記録で満たされるので印を付けた。FR-1〜FR-11 は今までどおり付けない」のような 1 文を足す。または今のままにする |
| V3 | INFO | `git diff --check` が `gate-excerpts.md` の 4 行(23・25・32・36 行)の行末の空白を出す。元の log の空行に `392: ` の形の行番号を付けた行で、内容は元の log のとおり。CI と run-static-verify は `git diff --check` を走らせない。追跡済みの証拠の md にも、行末に空白を持つものが 3 つある | 生ログの `## git diff --check` | 対応は要らない。消すなら行番号の後ろの空白だけを消す |
| V4 | INFO | 計画の Progress checklist の「Review artifact created」(508 行)が未チェックのまま。self-review の報告は a056c5df でコミット済み | 計画 508〜511 行 | `/sync-docs` か `/pr` で印を付ける |
| V5 | INFO | 証拠 3 行は `docs/plans/active/2026-10-20-org-trial.md` を引く。`scripts/archive-plan.sh` は register だけを書き換えるので、`/pr` のあとこの参照は古い道になる。`docs/evidence/` の md の 24 件が同じ形で、この PR が始めたものではない | 証拠 3 行、`scripts/archive-plan.sh` 54〜115 行 | 対応は要らない |
| V6 | INFO | register 304 行の Trigger(「The test fails again … then find the cause instead of only re-running」)は、試行の F9 で条件を満たした。Debt 欄は失敗と、原因を調べなかったこと(計画の Non-goals)を書いている。self-review の I3 と同じ点で、lead の判断で今のまま | register 304 行 | 対応は要らない |

## Coverage gaps

- director のメモ(コミットされていない)だけにあるもの: 各 verb の終了コード(wait の 4、send の 1、stop・merge・regate の 0)、cleanup の各段の結果と `pgrep`・`lsof` の出力、load average、Timeline で出どころが「notes」の時刻。snapshot に跡がある所(停止の窓の 0 件、`watch.log` の開始 2 回、`watch-alive.json` の pid と開始時刻、`director.jsonl` に `detached` がないこと、`director-wait.json`)とは矛盾しない
- `seat-usage.tsv` と `watcher-usage.txt` の和は、`~/.claude/projects/` の session 記録から取り直していない。単価は 2026-10-20 の価格表から読んだという記述を信じ、ページは開き直していない。費用の再計算は、この 2 つのファイルからの計算だけを確かめた
- `interventions.log` の時刻は director の transcript から読んだ値で、transcript は読んでいない
- session の label と session 記録の対応(証拠 91〜99 行)は確かめていない
- 2 本の PR の CI とテストは `/test` の範囲で、ここでは走らせていない。Go のコードは diff にないので、Go の静的解析も走らせていない
- 決定的な verifier は足りている。証拠の `sh` block と表の一致を CI で見る check は足せるが、この 1 つの証拠のための check になるので、今回は提案に留める

## Verdict

- Verdict: pass(CRITICAL・HIGH・MEDIUM はない。LOW 2 件は、証拠の doctor-escape の報告の範囲の書き方(V1)と、仕様の FR の印の決まりと FR-12 の印の関係(V2)で、どちらも数字と AC の可否を変えない。INFO 4 件(V3〜V6)は対応不要)
- Verified: AC1〜AC8 と AC10 を証拠、snapshot、仕様、register、コードと突き合わせた。証拠の 17 block の再計算(17/17 一致と、比べ方の対照)。本文の数字の手計算。所見が引くコードの行。PR #241・#242 の状態と merge commit と変更ファイル(gh で読むだけ)。register の欄の数と解消の書き方と道。漏えいの grep と hex の分類。静的解析(run-static-verify exit 0、計画の digest)
- Partially verified: AC5 と AC9 の終了コードと cleanup の結果(証拠の記述と snapshot の跡が合うことだけ)、費用(2 つの記録ファイルからの計算だけ)
- Not verified: director のメモと transcript、session 記録の和、価格表、PR の CI とテスト
