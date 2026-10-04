# Test report: org-drop-qa-seat

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-org-drop-qa-seat.md
- Tester: tester subagent (Claude)、cycle 1
- Scope: worktree `.claude/worktrees/org-drop-qa-seat`(branch refactor/org-drop-qa-seat、HEAD 6d9c1a12、base origin/main 4ee080f5)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`、`go test ./... -count=1`、AC-9 の smoke、AC-2 / AC-3 / AC-4 / AC-13 の mutation、計画の edge case のうち専用のテストがない 2 つの確認
- Evidence: `docs/evidence/test-2026-10-04-org-drop-qa-seat.log`(gitignore 済みでローカルのみ。wrapper 自身のログは `docs/evidence/verify-2026-10-04-115427.log`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(rc=0) | shell 33 本 + golang verifier | shell の PASS 行 1533 | 0 | 0 | 4:01.7 |
| `go test ./... -count=1 -cover`(rc=0、キャッシュなし) | 8 パッケージ | 全パッケージ `ok` | 0 | 2(既存) | 1:04.3 |
| `go test ./... -count=1 -json`(rc=0、件数の集計用にもう 1 回) | トップレベル 1021 | 1019 | 0 | 2 | — |
| AC-9 smoke(`claude -p --model sonnet`) | 1 | 1(run 2) | 0 | — | 9 秒 |
| mutation(使い捨ての harness) | 34 mutant | 31 caught | 3 survived | — | 約 9 分 |

wrapper の golang verifier では `internal/org` 以外がキャッシュから返ったので、`go test ./... -count=1` を別に実行した。shell の件数は、suite ごとに書式が違うので `PASS` で始まる行を awk で数えたもの。FAIL の行は 0 で、要約の `FAIL: 0` の行は数えていない。

Go のパッケージごとの結果(`-json` の 2 回目):

| Package | Top-level pass / fail / skip | Subtests pass / fail | Coverage |
| --- | --- | --- | --- |
| `internal/cli` | 459 / 0 / 0 | 255 / 0 | 84.7% |
| `internal/config` | 30 / 0 / 0 | 11 / 0 | 92.3% |
| `internal/insights` | 35 / 0 / 0 | 0 / 0 | 86.1% |
| `internal/org` | 296 / 0 / 0 | 121 / 0 | 90.8% |
| `internal/org/driver` | 42 / 0 / 0 | 29 / 0 | 92.0% |
| `internal/org/protocol` | 13 / 0 / 0 | 44 / 0 | 97.9% |
| `internal/scaffold` | 30 / 0 / 2 | 0 / 0 | 75.7% |
| `internal/upgrade` | 114 / 0 / 0 | 13 / 0 | 91.2% |

skip の 2 件(`TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`)は、`cmd/ralph` から build しないと embed.FS が空になるための既存の skip で、この branch は `internal/scaffold` を変えていない。

## Coverage

- Statement: 合計 87.0%(`go tool cover -func`)。前回の記録(2026-09-20)の `internal/cli` 84.3%、`internal/org` 90.5% から、それぞれ 84.7%、90.8% になった
- Branch: Go の標準ツールでは測れない
- Function: この branch で足した関数のうち、`RetiredRoleInputErr`、`retiredRoleConfigErr`、`RetiredRoleConfigKeys`、`resolveLeaderDriver`、`checkOrgRetiredRoleKeys`、`RenderRolePrompt`、`filterLeaderHistoryLines` が 100%。`ensureLeaderJoined` は 90.0%
- Notes: 行の coverage は、雛形の文言をテストが検査しているかどうかを示さない。下の mutation で補った

## Failure analysis

テストの失敗はない。AC-9 の smoke の run 1 は harness の設定が原因で成立しなかった。

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| AC-9 smoke run 1 | 出力が `GATE: unrunnable`(合格の条件は `GATE: fail`) | harness の権限。モデルは `sh -c '…'; echo "exit=$?"` を実行しようとし、後ろの `echo` が `Bash(sh:*)` に入らず `dontAsk` で拒否された(session の transcript で確認)。雛形の挙動の問題ではない | `Bash(echo:*)` を足し、書き込みを scratch のディレクトリに閉じた harness で 1 回やり直した(run 2) |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 撤去・改名した役割以外の未知の役割は拒否されない | PASS | `TestOrgSpawn_UnknownRole_NoTemplateApplied`(`internal/cli`) |
| 役割が空の spawn は通る | PASS | `TestOrgSpawn_RetiredLeaderName_UnrelatedConfigKeysAndRolesStillSpawn/role=`、`TestOrgSpawn_RemovedRole_OnlyTheRoleIsGuarded/seat_id_with_an_empty_role_and_no_prompt` |
| 過去の receipts(`role=lead`)の insights の集計 | PASS | `internal/insights` 35/35(`testdata/receipts.jsonl` を読むテストを含む)、`internal/cli/insights_test.go` の `SEAT lead` の行 |
| reviewer 雛形の後ろに `--prompt` が連結される | PASS | `TestOrgSpawn_RoleTemplate_PromptFlagAppendedAfterTemplate` |
| `--role QA` / `--role Lead` は未知の役割として通る | PASS | `TestOrgSpawn_RemovedRole_OnlyTheRoleIsGuarded/upper-case_spelling_is_an_unknown_role`、`TestOrgSpawn_RetiredLeaderName_CaseSensitive` |
| 既知の flaky(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`) | PASS | キャッシュなしの 2 回とも通った |

AC ごとのテストは、どれもキャッシュなしの 2 回で通った。AC-1 `TestRenderRolePrompt_QA_NoTemplate`、AC-2 `TestRenderRolePrompt_Reviewer_MissionRunsGateFirstAndBlocksWithoutReviewing`、AC-3 `TestRenderRolePrompt_Leader_MissionRoutesGateBlocked`、AC-4 `TestOrgSpawn_RemovedRole_WithoutPrompt_RejectedBeforeAnyManifestWrite` と `…_WithPrompt_StartsWithPromptOnly`、AC-11 `TestLeaderIdentity_ConstantValue`、`TestOrgStart_HappyPath_SpawnsLeaderSeat_SingleAgmsgJoin_NoHello`、`TestNewWatchdogHooks_AbnormalVerdict_SendsAlertToLeader`、AC-13 `TestOrgSpawn_RetiredLeaderName_RejectedBeforeAnyManifestWrite`、`TestOrgSpawn_RetiredLeaderRoleAndID_RejectedThroughCLI`、`TestOrgSpawn_RetiredRoleOrID_RefusedBeforeModelFallback`、`TestOrgCleanupVerbs_RetiredLeaderConfigKey_StillWork`、`TestRunDoctorOpts_RetiredRoleKey_WarnsWithoutFailing`、`TestOrgSpawn_DeprecatedDriverFlagAlias`。

## Mutation(AC-2 / AC-3 / AC-4 / AC-13)

計画の記録(Slice A の 9 通り)とは別に、私が mutant を 1 つずつ入れて確かめた。対象のテストを `-run` で絞って実行し、落ちなければ `internal/org` と `internal/cli` を全部実行し直した。毎回 `git checkout` で戻し、最後に `git status --porcelain` が空であることを確かめた。

| ID | 変更 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| T1 / T2 | reviewer: 項目 1 から `./scripts/run-static-verify.sh` / `./scripts/run-test.sh` を消す | caught | Reviewer_MissionRunsGateFirst… |
| T3 / T4 | reviewer: 項目 2 / 3 から `GATE: fail` / `GATE: unrunnable` を消す | caught | 同上 |
| T5 / T6 | reviewer: 項目 2 / 3 から「差分レビューに進まない」を消す | caught | 同上 |
| T7 | reviewer: 項目 4 から `GATE: pass` を消す | caught | 同上 |
| T8 | reviewer: 項目 1 の「最初に」を消す | survived | なし |
| T9 | reviewer: 「スクリプトの出力を正とし、推測で結果を上書きしない」を「スクリプトの出力を参考にする」にする | survived | なし |
| L1〜L5 | leader: 「implementer 座席に差し戻」「implementer には戻さず」「`GATE: unrunnable`」「人に上げる」「reviewer 座席へ委譲」を 1 つずつ消す | caught | Leader_MissionRoutesGateBlocked |
| L6 | leader: 「直ったら reviewer にもう一度ゲートから実行させる」を「直ったら完了とする」にする | survived | なし |
| G1 | spawn: qa の拒否から `prompt == ""` の条件を外す | caught | RemovedRole_WithPrompt_StartsWithPromptOnly |
| G2 | spawn: qa の拒否を無効にする | caught | RemovedRole_WithoutPrompt_Rejected… |
| G3 / G4 | spawn: `--id lead` / `--role lead` の拒否を無効にする | caught | RetiredLeaderName_Rejected… |
| G5 | spawn: Spawn の先頭の旧名チェックの呼び出しを無効にする | caught | 同上 |
| G6 / G7 | spawn: ralph.toml の旧キーの検査を real / dry-run で無効にする | caught | 同上 |
| G8 | spawn: 旧名の照合を大文字小文字を区別しないようにする | caught | RetiredLeaderName_CaseSensitive |
| P1 | 表から `qa` を消す | caught | RetiredRoles_RemovedRoleNamesTheReviewerAsSuccessor |
| P2 / P3 / P4 | 旧キーの走査: `[org.permissions.roles]` / `[org.roles]` を見ない、撤去した役割のキーも数える | caught | RetiredRoleConfigKeys |
| C1 | CLI: model の fallback より前の旧名チェックを無効にする | caught | RetiredRoleOrID_RefusedBeforeModelFallback |
| C2〜C5 | CLI: `--lead-driver` の矛盾の検査を外す、旧フラグの値を捨てる、`MarkDeprecated` を外す、`Changed()` の代わりに値で判定する | caught | DeprecatedDriverFlagAlias |
| K1 / K2 | doctor: warn を pass にする、検査を結果に入れない | caught | CheckOrgRetiredRoleKeys、RunDoctorOpts_RetiredRoleKey_WarnsWithoutFailing |

計画が AC-2 / AC-3 で求めた文言(2 本のスクリプト名、`GATE: fail`、`GATE: unrunnable`、2 か所の「差分レビューに進まない」、差し戻し、`GATE: unrunnable` の扱い)は、1 つずつ消すとすべてテストが落ちた。生き残った 3 つは Test gaps に書く。

## 計画の edge case の確認(使い捨てのテスト)

専用のテストがない 2 つを、`internal/org/zz_tester_scratch_test.go` を一時的に置いて確かめた。実行の後に削除し、`git status --porcelain` が空であることを確かめた。

- org_id が 25 文字なら leader 座席を立てられ(herdr の名前は 32 文字)、26 文字なら `combined org_id+seat_id length 33 exceeds herdr's 32-character agent-name limit` で拒否される。拒否のときは herdr と agmsg の呼び出しが 0 回。計算は既存の `TestOrgSpawn_CombinedIdentifierLength_*` と同じなので、leader 専用のテストは足さなくてよいと判断した
- `--role qa --prompt " "`(空白だけ)は拒否されずに起動する。計画の「実装中の逸脱」(Slice B)に書かれた動作と一致する

## AC-9 smoke(LLM の振る舞いの 1 回の観察。Known gap)

結果: run 2 で合格の条件を満たした。`TYPE: BLOCKED` と `GATE: fail` があり、`SEVERITY:` はない。LLM の振る舞いを 1 回見ただけなので保証ではない。

### 手順

1. scratch のディレクトリに `internal/org/prompts/reviewer.md` を写し、5 つのプレースホルダを sed で置き換えた(`{{ORG_ID}}`→smoke、`{{SEAT_ID}}`→reviewer、`{{TEAM}}`→ralph-smoke、`{{ROLE}}`→reviewer、`{{SCOPE}}`→fixture only)。`{{` は 0 個。依頼のとおりの TASK と注記を末尾に足した(`prompt.md`、119 行、sha256 `0812f3bc…`。run 1 と run 2 で同じファイル)
2. cwd を scratch のディレクトリにして、shell の alias を通さない `command claude` を実行した。claude は 2.1.285、transcript のモデルは `claude-sonnet-5-5`

### 実行したコマンドと、依頼のコマンドからの変更

依頼のコマンドは `command claude -p --model sonnet --allowedTools 'Bash(sh:*)' --output-format text`。`--allowedTools` の書き方はこの版でも通った。ただ、このままでは「sh のコマンドだけを許可する」状態にならない。ユーザーの `~/.claude/settings.json` に `permissions.defaultMode: "auto"` と、Bash の allow の規則(`git commit`、`curl`、`cp`、`echo` など)が入っているためで、何もしなければこれらも効く。そこで次のフラグを足した。

- run 1: `command claude -p --model sonnet --setting-sources project --strict-mcp-config --permission-mode dontAsk --allowedTools 'Bash(sh:*)' --output-format text < prompt.md > out.txt 2> err.txt`
- run 2: `command claude -p --model sonnet --setting-sources project --strict-mcp-config --permission-mode acceptEdits --permission-prompts none --allowedTools 'Bash(sh:*)' 'Bash(echo:*)' --output-format text < prompt.md > out.txt 2> err.txt`

どちらの前にも haiku で権限を確かめた。run 1 の設定では `sh -c 'echo …'` が動き、`ls -la /` が拒否された。run 2 の設定では、`sh -c '…; exit 1'; echo "exit=$?"` が動き、cwd の中の `docs/reports/x.md` への Write が通った。cwd の外への Write と `ls /` は拒否された。

run 2 で `Bash(echo:*)` を足したのは、run 1 でモデルが終了コードを見るために `; echo "exit=$?"` を付け、そこが拒否されたためである。書き込みを許したのは、雛形が reviewer に `docs/reports/` へのレポートを求めているからで、書ける範囲は scratch のディレクトリの中に限った。smoke の実行はこの 2 回だけで、run 1 が成立しなかった理由は transcript の拒否の文で確かめた。

### run 1(harness が原因で不成立)

rc=0、2026-10-04T11:55:56Z〜11:56:10Z、err.txt は空。モデルが呼んだツールは、Bash `sh -c 'echo "--- FAIL: TestFixture (fixture_test.go:12)"; exit 1'; echo "exit=$?"`(拒否)と、`docs/reports/cross-review-t-1.md` への Write(拒否)。out.txt の全文:

````text
ゲートを実行できなかったので、leader への BLOCKED メッセージを標準出力に出します(smoke のため送信はしません)。

```
TYPE: BLOCKED
TASK_ID: t-1
GATE: unrunnable

EVIDENCE: なし(レポートは書き込めなかった。理由は SUMMARY を参照)
SUMMARY: ゲートのコマンド `sh -c 'echo "--- FAIL: TestFixture (fixture_test.go:12)"; exit 1'` を
  実行できなかった。Bash ツールが "Permission to use Bash has been denied because
  Claude Code is running in don't ask mode" で拒否された。docs/reports/cross-review-t-1.md
  への Write も同じ理由で拒否されたため、レポートは残せていない。ゲートの結果は不明で、
  推測では埋めていない。差分レビューも行っていない。Bash と docs/reports/ への
  Write を許可して再実行すれば進められる。
```
````

合格の条件(`GATE: fail`)は満たしていない。ただし、ゲートを実行できなかったときの雛形の項目 3(`GATE: unrunnable` と理由を付けて BLOCKED、差分レビューに進まない)には沿っていて、結果を推測で埋めてもいない。これも 1 回の観察である。

### run 2(採用する結果)

rc=0、2026-10-04T11:57:13Z〜11:57:22Z、err.txt は空。モデルが呼んだツールは、Bash `sh -c 'echo "--- FAIL: TestFixture (fixture_test.go:12)"; exit 1'; echo "exit=$?"`(出力は `--- FAIL: TestFixture (fixture_test.go:12)` と `exit=1`)と、scratch の中の `docs/reports/cross-review-t-1.md` への Write。out.txt の全文:

````text
ゲートが落ちました。差分レビューは行っていません。次の BLOCKED を leader 宛ての出力とします。この smoke には agmsg がないので、送信はしていません。

```
TYPE: BLOCKED
TASK_ID: t-1
GATE: fail

EVIDENCE: docs/reports/cross-review-t-1.md
SUMMARY: ゲートコマンドが exit 1 で失敗。TestFixture が fixture_test.go:12 で失敗。差分レビューは行っていない。
```
````

`grep -c` の結果は `TYPE: BLOCKED` 1、`GATE: fail` 1、`SEVERITY:` 0。BLOCKED には、項目 2 が求める root cause(テスト名・ファイル・行)とレポートのパスが入っている。

out.txt の外のことを 1 つ書いておく。scratch に書かれたレポートは、ゲートの失敗そのものを `SEVERITY: HIGH` の項目として載せ、`Diff review was not performed.` と書いている。合格の条件は out.txt だけを見るので、判定には影響しない。差分の所見ではなくゲートの結果の記録だが、雛形は fail のときのレポートの書き方を決めていないので、下流の読み手が所見と取り違えるおそれは残る。

### 後片付けの確認

smoke の後、worktree と main checkout(`/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph`、HEAD 4ee080f5)の `git status --porcelain` はどちらも空だった。モデルが書いたファイルは scratch のディレクトリの中だけにある。この report を書き、insight event を足した後の worktree の差分は、この report(未追跡)と `docs/insights/events/2026-10-04-org-drop-qa-seat.jsonl` に足した test の 1 行だけで、main checkout は空のまま。

## Test gaps

- AC-9 は LLM の振る舞いを 1 回見ただけで、保証ではない(計画どおり Known gap)。`GATE: fail` の経路は run 2、`GATE: unrunnable` の経路は run 1 で、どちらも 1 回だけ見た。codex の reviewer 座席と、`guarded` の座席は見ていない(計画の Open questions)
- 雛形のテストは文言があるかどうかだけを見るので、次の 3 つの mutant が生き残った。
  - T8: ゲートを「最初に」実行するという順序は検査していない。項目の番号で順序は残るが、項目 1 を後ろへ動かしても落ちない
  - T9: 「スクリプトの出力を正とし、推測で結果を上書きしない」は検査していない。計画の R3 は、この文言と「fail ならレビューに進まない」の両方をテストで確かめるとしていたが、テストにあるのは後者だけである。足すなら、Reviewer_MissionRunsGateFirst… の項目 1 に「推測で結果を上書きしない」を加えるのが一番小さい
  - L6: leader が `GATE: fail` の後に reviewer にゲートからやり直させる手順は検査していない
- 文言の検査は、否定の言い回しに書き換える mutant(例:「進まないとは限らない」)は止められない。テストの方式から来る限界である
- AC-5 / AC-12 / AC-14 の grep は手で実行するもので、テストにはなっていない(verify の report の指摘と同じ)

## Verdict

- Pass: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(rc=0)、`go test ./... -count=1`(8 パッケージすべて ok、トップレベル 1019 pass、0 fail、既存の skip 2)、AC-10 のテストの部分、AC-1〜AC-4・AC-11・AC-13 のテスト、mutation 34 のうち 31 caught、AC-9 smoke(run 2、1 回の観察)
- Fail: なし
- Blocked: なし
- Overall: PASS。/pr に進んでよい。生き残った mutant 3 つは LOW の Test gap で、マージを止めるものではない
