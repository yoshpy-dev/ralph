# Test report: reseat-notices

- Date: 2026-11-16(run の終わりは 2026-11-17 00:13 JST)。cycle 1(`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-16-reseat-notices.md`
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `61689558`(verify report の commit のあと。worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから。scope は wrapper の既定の changed で、選ばれた pack は golang。`detect-languages.sh` が返すのも golang だけなので、full と同じ範囲になる。local verifier は scope によらず `tests/*.sh` を全部回す
- Evidence: `docs/evidence/test-2026-11-16-reseat-notices.log`(5,619 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 23:51:39 〜 2026-11-17 00:13:16 JST、1,297 s、rc 0 |
| local verifier(`tests/*.sh` 45 本) | 45 本 | 45 本 | 0(FAIL の行は `FAIL: 0` の集計だけ) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`git 2.41 より古い本物の git` の場合(手元は git 2.49.0) | 約 12 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.057 s |
| `internal/cli` | ok | 656.120 s |
| `internal/config` | ok | 13.568 s |
| `internal/gitroot` | ok | 4.189 s |
| `internal/insights` | ok | 5.428 s |
| `internal/org` | ok | 744.430 s |
| `internal/org/driver` | ok | 4.465 s |
| `internal/org/protocol` | ok | 1.578 s |
| `internal/scaffold` | ok | 3.474 s |
| `internal/toolchain` | ok | 25.647 s |
| `internal/upgrade` | ok | 2.704 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg`): 始まり `16.30 20.87 20.59`、終わり `8.22 10.68 13.41`。途中の読みは 8〜19。この run の間、ほかの `go test` は動いていなかった
- `GOFLAGS` は足していない。golang の pack(`packs/languages/golang/verify.sh:115`)が `-timeout=30m` を GOFLAGS の先頭に入れる。`internal/org` の 744 s は上限の 41%、`internal/cli` の 656 s は 36%
- 起動: Bash ツールの `run_in_background` で `$TMPDIR/reseat-notices-test/runner.sh` を動かした。runner は pid、始まりと終わりの時刻、負荷、rc を meta file に書く。`nohup` や `&` は使っていない。待つのは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返した
- run-verify.sh は自分の evidence も `docs/evidence/verify-2026-11-16-145140.log` に書いた(同じく追跡外)

落ちたテストがないので、`-count=1 -v` の打ち直しはしていない。

## 計画のテストと AC の対応

全パッケージの run が通ったので、次のテストはすべて通っている。

| AC / Test plan の項 | テスト | 結果 |
| --- | --- | --- |
| AC1(本物の reseat、codex の座席に 1 行、止める前、claude の座席には出さない、計画の前の断りでは出さない) | `TestCodexHookTrustNotice_ReseatPrintsOneLineForACodexSeat`、`..._ReseatOfClaudeSeatsPrintsNothing`、`..._ReseatRefusedBeforeItsPlanPrintsNothing`(4 行)、`TestReseat_OnPlanRunsOnceBeforeTheFirstStop`(`every seat`、`one seat on codex`)、`TestReseat_OnPlanIsNotCalledOnARefusal`(5 行) | pass |
| AC2(dry run も同じ行) | `TestCodexHookTrustNotice_ReseatDryRunPrintsTheSameLine`、`TestReseat_OnPlanRunsOnceBeforeTheFirstStop/a_dry_run` | pass |
| AC3(`--help` の 2 段落) | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`、`TestCodexHookTrustHelp_EachVerbDescribesTheNote`、`TestScaffoldVersionHelp_EachVerbDescribesTheNote` | pass |
| AC4(director skill の 4 写し) | `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`、`TestDirectorSkill_ReseatPointsAtTheHumanNotes` | pass |
| AC5(`director.md`) | `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection` | pass |
| AC6(check-sync と check-skill-sync) | `tests/test-check-sync.sh`、`tests/test-check-skill-sync.sh`(local verifier の中)。2 つの script そのものは verify の担当 | pass |
| AC7(`OnPlan` が nil の reseat) | 既存の reseat のテスト。`go test -list Reseat` で数えると、名前に Reseat を含む top-level のテストは `internal/org` に 93(この PR の 2 つを含む)、`internal/cli` に 17。既存のものは `OnPlan` を渡さない `st.reseatParams` を使う | pass |
| Test plan の Integration: `--driver codex` で claude の座席を codex に替えると行が出る | `TestCodexHookTrustNotice_ReseatPrintsOneLineForACodexSeat` | pass |
| Test plan の Edge cases: codex の座席が複数なら座席ごとに 1 行 | テストがない | gap(G2) |
| Test plan の Edge cases: worktree の hooks.json が main のチェックアウトに読み替えられる | `..._ReseatPrintsOneLineForACodexSeat` が main のチェックアウトの hooks.json を名指す行を待つ。ただしテストの cwd も main のチェックアウトなので、渡す dir の取り違えは見分けられない | 一部(G1) |

## 変異の確かめ

`git archive HEAD` の写し(`$TMPDIR/reseat-notices-test/copy`)に `go test -overlay` で当てた。worktree は触っていない。対照の run(変異なし)は `-run 'CodexHookTrust|Reseat|Probe_' ./internal/cli/` が 51 s で ok、`-run Reseat ./internal/org/` が 15 s で ok。

| ID | 変異 | 落ちたテスト | 結果 |
| --- | --- | --- | --- |
| M1 | CLI が `OnPlan` を渡さない(`org_reseat.go:226` の行を消す) | cli: `..._ReseatPrintsOneLineForACodexSeat`、`..._ReseatDryRunPrintsTheSameLine` | killed |
| M2 | reseat の関数が座席の driver の代わりに `"codex"` を渡す(`org_reseat.go:275`) | cli: `..._ReseatOfClaudeSeatsPrintsNothing` | killed |
| M3 | `printCodexHookTrustNotice` の driver の判定(`codex_hook_trust_notice.go:38-40`)を消す | cli: spawn の `claude` の行、start、start --plan、reseat、director start の claude の 5 テスト | killed |
| M4 | 渡す dir を `plan.Binding.Worktree` から `"."` に替える | なし(cli の filter で 0 件) | survived(G1) |
| M10 | 座席の繰り返しを最初の 1 席で `break` する | なし | survived(G2) |
| M5 | dry run の `OnPlan` の呼び出しを消す | org: `TestReseat_OnPlanRunsOnceBeforeTheFirstStop/a_dry_run`。cli: `..._ReseatDryRunPrintsTheSameLine` | killed |
| M6 | dry run の `err == nil &&` を外す | org: `TestReseat_OnPlanIsNotCalledOnARefusal` の dry run の 2 行。cli は 0 件 | killed(org だけ) |
| M7 | 本物の reseat の `OnPlan` の呼び出しを消す | org: `every_seat`、`one_seat_on_codex`。cli: `..._ReseatPrintsOneLineForACodexSeat` | killed |
| M8 | 本物の reseat の呼び出しを `stopAll` のあとへ移す | org: 同じ 2 行(順の記録)。cli: `..._ReseatPrintsOneLineForACodexSeat`(行を書いた時点の manifest) | killed |
| M9 | 本物の reseat の呼び出しを印の書き込みの前へ移す | org: 同じ 2 行(印の確かめ) | killed(org だけを回した) |

依頼された 2 つの変異(CLI が `OnPlan` を渡さない M1、driver の判定を外す M2 と M3)はどれも落ちた。M6 が cli で残るのは、`PlanReseat` が断るときに `ReseatPlan{}` を返し(`reseat.go:401` からの `refuse`)、`printOrgReseatCodexHookTrustNotices` が `plan.Binding == nil` で何も出さないため。org のテストが押さえているので穴とはしない。

M4 と M10 を見分ける使い捨てのテストを `$TMPDIR/reseat-notices-test/mut/probe_reseat_notices_test.go` に書き、overlay で足した。`printOrgReseatCodexHookTrustNotices` を直接呼び、座席 3 つ(codex の leader、claude の implementer、codex の reviewer)の計画を、入れ子の `.codex/hooks.json` を持つ下のディレクトリから渡す。HEAD では通り(`-v` で PASS を確かめた)、M4 では行が「4 hooks in <root>/.codex/hooks.json and <root>/sub/.codex/hooks.json」になって落ち、M10 では reviewer の行が消えて落ちた。

## Coverage

- Statement: 変えた関数を、filter をかけた run で測った。`printCodexHookTrustNotice` 100.0%、`runOrgReseat` 90.0%、`printOrgReseatCodexHookTrustNotices` 75.0%(cli、`-run 'CodexHookTrust|Reseat'`)、`Reseat` 93.3%(org、`-run Reseat`)
- Branch: 測る道具がない
- Function: 上と同じ
- Notes: 通らないブロックは 3 つ。`org_reseat.go:272`(`plan.Binding == nil` の return。`PlanReseat` が binding の無い org を断るので、本物の経路では来ない)、`reseat.go:1152`(印の書き込みの失敗。計画の Progress に記録済みの穴)、`reseat.go:1134`(org_id の検査の失敗。この PR の前からある分岐で、filter の外のテストが通るかは見ていない)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| spawn、start(2 つの形)、director start の hook の行(#310) | pass | `TestCodexHookTrustNotice_*` の既存のテストが全部通った。M3 でそれらが driver の判定を押さえていることも確かめた |
| 雛形の版の行(#313)と reseat の既存の動き(#312c) | pass | `internal/cli` と `internal/org` の全テスト |
| skill の 4 写しと `.codex/README.md` の 2 写しの一致 | pass | `tests/test-check-skill-sync.sh`、`tests/test-check-sync.sh` |

## Test gaps

- G1(LOW): `printOrgReseatCodexHookTrustNotices` が機能の worktree を渡すことを押さえるテストがない。M4(`"."` を渡す)が残った。CLI のテストはどれも main のチェックアウトのルートから reseat を打ち、linked worktree の hooks は main のチェックアウトから読まれるので、cwd を渡しても同じ行になる。入れ子の `.codex/hooks.json` を持つ下のディレクトリから打つと、cwd の版は座席が読まない層まで数える
- G2(LOW): codex の座席が 2 つ以上ある reseat のテストがない。計画の Test plan の Edge cases にある項で、M10(最初の 1 席で止める)が残った。CLI の fixture は座席を claude で立て、`--driver` は `--seat` 1 つにしか付けられないので、本物の reseat で codex の座席を 2 つにするには codex で spawn した台帳が要る
- G3(LOW): codex の leader を reseat するテスト(`codex seat leader` の行)がない。使い捨てのテストでは関数の直接の呼び出しで確かめた
- G4(INFO): 印の書き込みが失敗したときに `OnPlan` を呼ばない経路のテストがない(計画の Progress に記録済み)
- G5(INFO): reseat を走らせて雛形の版の行を確かめる CLI のテストがない。この PR の前からの穴(verify report の V4 (a))
- G6(INFO): 本物の herdr と codex では走らせていない

G1 から G3 を閉じる一番小さいテストは、使い捨てのテストをそのまま `internal/cli/codex_hook_trust_notice_test.go` に入れる形になる。`printOrgReseatCodexHookTrustNotices` を、codex・claude・codex の 3 席の `org.ReseatPlan` と、入れ子の hooks.json を持つ下のディレクトリを cwd にして呼び、codex の 2 席の行だけが順に、worktree のルートの hooks.json だけを名指して出ることを確かめる。HEAD で通り、M4 と M10 で落ちることは確かめてある。

## Insight event

`./scripts/insights-append.sh --slug reseat-notices --flow standard --phase test --verdict pass --cycle auto --source skill --events-dir "$TMPDIR/reseat-notices-test/events"` で作業用のディレクトリに書いた。依頼はこの report だけの commit なので、`docs/insights/events/` には足していない(verify と同じ扱い)。

```
{"schema":1,"ts":"2026-11-16T15:23:12Z","slug":"reseat-notices","flow":"standard","phase":"test","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```

## Verdict

- Pass: 56/56(Go の 11 パッケージと shell の 45 本。`==> All verifiers passed.`、rc 0)
- Fail: 0
- Blocked: 0
- Verdict: **pass**。AC7 は既存の reseat のテストがすべて通って満たされた。AC1 から AC6 のテストも通った。G1 から G3 は LOW の穴で、PR を止めない。verify report の文書のずれ(V1 から V4)は `/sync-docs` の担当として、ここでは触れていない
