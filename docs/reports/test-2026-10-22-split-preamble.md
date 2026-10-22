# Test report: split-preamble

- Date: 2026-10-22(JST 23:17〜10/23 00:37、UTC 14:17〜15:37)
- Plan: `docs/plans/active/2026-10-22-split-preamble.md`(試行の所見を直す系列の 5 本目、F5。AC1〜AC6 と AC9 の挙動、Test plan の境界の場合、回帰)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 8303bd52(base a7bdc884)。verify のあとの 3b21b51b は Go のコメントと根拠の表だけ、8303bd52 は計画の印と insight event だけを変えている。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`、toolchain はキャッシュ済み)
- Evidence: `docs/evidence/test-2026-10-22-split-preamble.log`(各実行の出力、時刻と load average、mutant の定義と結果、canary の記録、関数ごとの coverage。ホームは `~`、session の scratchpad は `<scratch>`、`$TMPDIR` は `<tmp>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-22-141758.log`(1 回目)、`docs/evidence/verify-2026-10-22-145150.log`(2 回目)。どれも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告のほかは変えていない。mutant は `go test -overlay` で scratchpad のファイルに差し替えて打ち、worktree のファイルには書いていない(batch の前後で `git status --porcelain` は 0 行)。本物の herdr・agmsg・osascript・Orca・tmux・gh や network を打つものは実行していない(6 の canary)。insight event は追記していない(末尾の「Insight event」)

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(1 回目) | shell 44 本、Go 8 パッケージ | shell 44 本、Go 7 パッケージ | Go 1 件(internal/cli の `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`) | shell 1 件(下) | 1,898 秒、rc 1。2・7 と重ねて打ち、load average(1 分)は最大 56 |
| 1' | 同じコマンド(2 回目、ほかに何も走らせずに) | shell 44 本、Go 8 パッケージ | shell 44 本、Go 8 パッケージ | 0 | shell 1 件(下) | 1,447 秒、rc 0。cli 519.6 秒、org 626.1 秒。開始時 load 13.7、終了時 6.4 |
| 2 | `GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./internal/org/ ./internal/cli/` | 2 パッケージ | 2 | 0 | 0 | 891 秒、rc 0。org 885.0 秒、cli 668.1 秒。1 と 7 に重なった |
| 3 | 新しいテストと、同じ関数を見る前からのテスト(org 10 本、cli 2 本)を `-count=3 -v` | 12 × 3 = 36(subtest を含む RUN 321) | 321 | 0 | 0(`--- SKIP` は 0 行) | org 2 秒、cli 4 秒 |
| 4 | 3 と同じに `-race`(`-count=1`) | 12 | 12 | 0 | 0 | org 2.8 秒、cli 3.1 秒。`DATA RACE` は 0 行 |
| 5 | 1 で落ちたテストだけを `-count=20 -v` | 20 | 20 | 0 | 0 | 8 秒(load 13) |
| 6 | 3 の 12 本を canary の PATH と偽の HOME で(下) | 12(RUN 107) | 107 | 0 | 0 | canary の呼び出しは 0 回 |
| 7 | mutant 41 本(下の表) | 41 | - | 38 本が赤 | - | 生き残りは G1・G2・G3 の 3 本で、どれも org パッケージ全体でも生き残った |
| 8 | org パッケージ全体を `-coverprofile` で | 1 パッケージ | 1 | 0 | 0 | 618.9 秒、statement 94.8% |

- run-test.sh は 2 回とも `==> Language scope: changed (changed_languages)`、pack は golang。2 回目の Go 8 パッケージは、打つ前に worktree の GOCACHE の testcache を消したので、どれも `(cached)` ではない。shell の 44 本はどれも最後の行が `OK` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばしている
- 1 回目の失敗は、この diff が触っていないテストの競合で、2 回目は通った(「Failure analysis」)。AC9 の判定には 2 回目を使う
- 3 の 12 本は、org の `TestLoadSplitPlan_TrialPreamble`、`TestLoadSplitPlan_Preamble`、`TestLoadSplitPlan_ParsesFeatures`、`TestLoadSplitPlan_Rejects`、`TestFeatureLeaderTask`、`TestFeatureLeaderTask_Preamble`、`TestFenceFor`、`TestStartFeature_LeaderPromptHoldsPreamble`、`TestRenderRolePrompt_Leader_FeatureOrgSeatRules`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(変えていない回帰の確かめ)と、cli の `TestOrgStartHelp_TaskSource`、`TestOrgStartPlan_FlagCombinationsRefused` である
- 6 の canary: `herdr`・`agmsg`・`osascript`・`open`・`tmux`・`gh`・`orca`・`claude`・`codex`・`curl`・`wget`・`nc`・`ssh` を、呼ばれたら記録して exit 97 する script にした dir を PATH の先頭に置き、`HOME` を空の scratch の dir にした(`GOCACHE`・`GOMODCACHE` は元の値、`GOPROXY=off`、`GOFLAGS` は空、`GOTOOLCHAIN=local`)。終わったあと canary の `herdr` を直に打つと exit 97 で 1 行記録したので、canary は呼ばれれば記録する。org のテストは `fakeHerdr` と `fakeAgmsg` を使い、cli の 2 本は help の表示と、herdr を打つ前の拒否だけを見る

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestLoadSplitPlan_TrialPreamble`(`internal/org/split_test.go:520`) | 追跡している試行の分割計画 `docs/evidence/org-trial-2026-10-20/splits/trial.md` を読む(ファイルがなければ `t.Fatalf` で落ち、skip はしない)。前文は `## 目的\n` で始まり `\n- 予約の外のファイルは編集しない` で終わり、「座席の決まり(両方の機能に共通):」を含み、題・`- Status:`・`- Approved:`・`## Features`・2 つの `### <slug>` を含まない。2 機能の slug・Type・Reserve と本文の先頭と末尾が今と同じ。ファイルの 6〜15 行が前文になることを目でも突き合わせた。mutant S1・S4〜S7 が赤 |
| AC2 | `TestLoadSplitPlan_Preamble`(`split_test.go:563`、11 件)、`TestLoadSplitPlan_Rejects` の 3 行(前文の `- Status:`、前文の `- [x]`、前文の閉じていないフェンス)、`TestLoadSplitPlan_ParsesFeatures` の前文の行 | フェンスに入った `## Features`(3 連と、3 連を含む 4 連)は前文に残る。チルダのフェンスも丸ごと残る。前文の中の `### ` と `- Reserve:` の形の行はただの文。`## Features` より後の節は前文にも本文にも入らない。ヘッダの自由文と前後の空行は入らない。ヘッダのフェンスの中の `## ` は前文を始めない。ヘッダのあとすぐ `## Features` なら空。CRLF でも前文の行に `\r` が残らない。`## Features \t` も見出しとして読む。mutant S2・S3・S8a・S8b・S9・S10 が赤 |
| AC3 | `TestFeatureLeaderTask_Preamble`(`internal/org/feature_test.go:580`、7 件と前文なしの確かめ)、`TestFenceFor`(`:631`、7 行) | 前文があれば、今の行、空行、前置きの行、フェンス、前文、フェンス、空行、本文の順。本文が空なら末尾の空行もない。空白だけの前文は何も足さない。inline code とチルダのフェンスでは 3 連、3 連のフェンスを含めば 4 連、4 連を含めば 5 連。`fenceFor` は並びの位置(先頭、途中、末尾、行をまたぐ)によらず最長 + 1、最低 3。mutant F1〜F15 が赤 |
| AC4 | `TestStartFeature_LeaderPromptHoldsPreamble`(`feature_test.go:657`、2 件) | `fakeHerdr` で `StartFeature` を回し、leader の role prompt のファイルの `## タスク` の節が、今の行、前置きの行、フェンスで囲んだ前文、本文の順で、`featureLeaderTask` の出力と同じ文字列になる。前文の `{{ORG_ID}}` は置き換えられずに残る。雛形の `## ` の見出しはどれも 1 回ずつ見つかり、前文の `## 目的` は開きのフェンスの直後に 1 回だけある。2,000 字を超える前文でも start は spawned で終わり、前文は削られずに入る。mutant P1(`{{TASK}}` を先に置き換える)が赤になったのはこのテストだけ |
| AC5 | `TestRenderRolePrompt_Leader_FeatureOrgSeatRules`(`internal/org/prompts_test.go:1049`)、変えていない `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:475`) | 手順 1・3・5 の項に、それぞれの文が今の文のあとに順に並ぶ。前置きの行が「全機能の座席に適用する決まり(」で始まることと、`protocol.DefaultMaxBodyChars` が 2,000 であることも見る。手順の位置を固めるテストは変えずに通る。mutant L1〜L6(文を消す、「囲みの行も含めて」を消す、「原文のまま」を「要約して」に、手順 3 の文を手順 4 へ移す、「写さず」を「写し」に、2,000 を 4,000 に)がすべて赤 |
| AC6 | `TestOrgStartHelp_TaskSource`(`internal/cli/org_feature_test.go:524`)、`TestOrgStartPlan_FlagCombinationsRefused` の「a task argument」の行 | help の文(空白を詰めて比べる)とエラーの文が、タスクは機能の本文と分割計画の前文から作られると書く。mutant C1・C2 が赤 |
| AC7・AC8 | なし(文書と記録) | /verify の範囲 |
| AC9 | 1'、2 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh` が rc 0(2 回目)。変えた org と cli のテストが go1.25.8 でも通る。静的な部分(`run-static-verify.sh` rc 0、go1.25.8 の vet と build)は verify の報告にある。`run-verify.sh` を 1 本のコマンドとしては打っていない |

## 計画の Test plan の項目

| 項目 | テスト |
| --- | --- |
| 試行の分割計画を読む(AC1) | `TestLoadSplitPlan_TrialPreamble` |
| 前文の中のフェンスの `## Features`、`## Features` より後の節、前後の空行、空の前文、CRLF、前文の中の `### ` | `TestLoadSplitPlan_Preamble` の各行 |
| `featureLeaderTask` の前文あり・なし・4 連のバッククォート・前文の中のフェンス | `TestFeatureLeaderTask_Preamble`、`TestFenceFor` |
| `StartFeature` で leader の role prompt を読む(2,000 字を超える前文も) | `TestStartFeature_LeaderPromptHoldsPreamble` |
| 手順 1・3・5 の文 | `TestRenderRolePrompt_Leader_FeatureOrgSeatRules` |
| start の help とエラーの文 | `TestOrgStartHelp_TaskSource`、`TestOrgStartPlan_FlagCombinationsRefused` |
| 前文のない分割計画でタスクが今とバイトで同じ | 変えていない `TestFeatureLeaderTask` が、前文のない計画のタスクを全文の文字列で比べる。`TestFeatureLeaderTask_Preamble` の最後の確かめも同じ形 |
| 今の split・feature・prompts・start のテストが、期待の文を直すほかは変えずに通る | `git diff a7bdc884 HEAD` の 4 つのテストファイルで消えた行は 3 行だけ: cli の「a task argument」の行(期待を足した。前の期待の部分文字列は今のエラーにも含まれる)と、`TestLoadSplitPlan_ParsesFeatures` の doc コメント 2 行。ほかは足した行だけで、そのうえで 1' と 2 が通った |
| 前文が空行だけ | 読み込みの側では、前文は `## ` の見出しから始まるので空行だけにはならない(ヘッダのあとの空行だけなら「no section ...」の行で空)。タスクの側は「a preamble of blank lines」の 2 行 |
| 前文の中の `{{ORG_ID}}` | `TestStartFeature_LeaderPromptHoldsPreamble`。P1 が赤 |
| 前文の中の 3 連と 4 連、チルダのフェンス | 読み込みとタスクの両方に行がある(上の AC2・AC3) |
| `## Features` の見出しの末尾の空白 | `TestLoadSplitPlan_Preamble` の「Features heading with trailing blanks」 |

## Mutant

`go test -overlay` で、org は `-run '^(TestLoadSplitPlan|TestFeatureLeaderTask|TestFenceFor|TestStartFeature_LeaderPromptHoldsPreamble|TestRenderRolePrompt_Leader)'`、cli は `-run '^(TestOrgStartPlan|TestOrgStartHelp)'` で打った。置き換えは文字どおりで、1 か所だけに当たることを確かめた。build に失敗した mutant はない。生き残った 3 本は org パッケージ全体(`-count=1`)でも打った。HEAD の同じ `-run` は org・cli とも通る(対照)。

| ID | 置き換え | 結果 |
| --- | --- | --- |
| S1 | 見出しの行を前文に足さない | 赤 |
| S2 | `## Features` より後の見出しも前文に足す | 赤 |
| S3 | `## Features` より後の本文の行も前文に足す | 赤 |
| S4 | 本文の行を前文に足さない | 赤 |
| S5 | 前後の空行を詰めない | 赤 |
| S6 | `Preamble` をいつも空に | 赤 |
| S7 | ヘッダの行も前文に足す | 赤 |
| S8a・S8b | 見出しの行・本文の行に `\r` を残す | 赤(CRLF の行) |
| S9 | 最初の節だけを前文にする | 赤 |
| S10 | フェンスの中の行を前文に足さない | 赤 |
| F1 | 空白だけの前文でもブロックを出す | 赤 |
| F2 | ブロックを出さない | 赤 |
| F3 | ブロックの前の空行をなくす | 赤 |
| F4 | 前置きの行を出さない | 赤 |
| F5 | 閉じのフェンスを出さない | 赤 |
| F6 | ブロックを本文のあとに置く | 赤 |
| F7 | フェンスを最長と同じ長さに | 赤 |
| F8 | 最低 3 をなくす | 赤 |
| F9a・F9b | 最低を 2・4 に | 赤 |
| F10 | 並びの数え直しをしない(バッククォートの総数) | 赤 |
| F11 | 末尾の並びを数えない | 赤(`TestFenceFor` だけ) |
| F12 | フェンスをチルダに | 赤 |
| F13・F14 | 前置きの行の先頭・末尾の文字を変える | 赤 |
| F15 | 前文の 1 行目だけでフェンスの長さを決める | 赤 |
| P1 | `{{TASK}}` を replacer の前に置き換える(前文の `{{ORG_ID}}` が置き換わる) | 赤 |
| L1 | 手順 1 の文を消す | 赤 |
| L1b | 「囲みの行も含めて」を消す | 赤 |
| L1c | 「原文のまま写す(言い換えず、削らない。」を「要約して写す(」に | 赤 |
| L2 | 手順 3 の文を消す | 赤 |
| L3 | 手順 5 の driver と model の文を消す | 赤 |
| L4 | 「TASK の本文に写さず」を「写し」に | 赤 |
| L5 | 「2,000 字」を「4,000 字」に | 赤 |
| L6 | 手順 3 の文を手順 4 の先頭へ移す | 赤 |
| C1 | help を前の文に戻す | 赤 |
| C2 | エラーを前の文に戻す | 赤 |
| G1 | `- Depends on:` のない機能にだけブロックを出す | 生き残り(org 全体 628 秒でも。G1) |
| G2 | 読み込みで前文全体に `strings.TrimSpace` | 生き残り(org 全体 583 秒でも。G2) |
| G3 | タスクで前文に `strings.TrimSpace` | 生き残り(org 全体 583 秒でも。G2) |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`(`internal/cli/org_watch_test.go:1325`、1 回目の run-test.sh だけ) | `org_watch_test.go:1353: a child that ignores SIGTERM ended after 2.159ms; want SIGKILL after the 200ms grace` | この diff の前からあるテストの競合で、この PR の変更とは関係しない(diff は `internal/cli/org_watch*.go` に触れていない)。子の script は `stdin: eof` を log に書いてから `trap '' TERM` を打つが、テストは `stdin: eof` を見た時点で stop を送るので、load が高いと trap より先に SIGTERM が届く。log の行と trap の間に `sleep 0.3` を入れた写し(overlay)では 3/3 が同じ文で落ち、trap を先に打ってから log を書く写しは、同じ `sleep 0.3` があっても 10/10 通った。HEAD のままでは単独 `-count=20` で 20/20、2 回目の run-test.sh でも通った。1 回目は go1.25.8 の実行と mutant の batch に重なり、load average は最大 56 だった | この PR では直さない。`docs/tech-debt/README.md` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行(281 行あたり)と同じ形の競合なので、行を 1 つ足すか、script の `trap` を log の行より前に移す(上の 10/10 の形)別の PR を勧める |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 前文のない分割計画の leader のタスク | 変わらない | `TestFeatureLeaderTask`(変えていない、全文の比較)が 1'・2・3 で通る |
| 手順の位置(機能ごとの org の手順 1〜) | 変わらない | `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(変えていない)が通る |
| 分割計画の拒否(digest が飛ばす行、閉じていないフェンス) | 変わらない | `TestLoadSplitPlan_Rejects` の前からの行がすべて通る。前文の中でも同じ拒否になることを 3 行足して確かめている |
| `## Features` より後の節を拒否しない(D4) | 変わらない | `splitPlanFixture` の `## Notes` を含む既存のテストと、「a section after Features ...」の行が通る |

## Coverage

- Statement: internal/org 94.8%(8。前の記録の 94.7% とほぼ同じ)
- Function: `parseSplitPlan`・`finishSplitPlan`・`trimBlankLines`・`featureLeaderTask`・`fenceFor`・`RenderRolePrompt` が 100%、`StartFeature` が 95.5%
- Branch: 計測していない(Go の標準ツールにない)。分岐は mutant で見た
- Notes: internal/cli の coverage は取っていない。cli で変えたのは help の文字列とエラーの文字列の 2 か所で、どちらも C1・C2 で赤になることを確かめた

## Test gaps

- G1(LOW): 依存のある機能(2 本目以降)の leader のタスクに前文が入ることを確かめるテストがない。前文を確かめるテストの機能は、どれも `- Depends on:` がない(`TestFeatureLeaderTask_Preamble` の `auth-core`、`TestStartFeature_LeaderPromptHoldsPreamble` の `auth-core`)。`featureLeaderTask` は機能によって分かれる分岐を持たないので今は正しいが、計画の目的「どの機能の leader にも届く」を固める行がない。mutant G1(依存のない機能にだけブロックを出す)は org パッケージ全体でも生き残った。直すなら、`TestFeatureLeaderTask_Preamble` に `DependsOn` のある機能と前文の行を 1 つ足し、期待を文字で組み立てる(`featureLeaderTask` 自身の出力と比べると、G1 でも両辺が同じく変わるので赤にならない)
- G2(INFO): 前文の最後の行の末尾の空白を残すことを確かめるテストがない。G2・G3(前文に `strings.TrimSpace`)は org 全体でも生き残った。前文は `## ` の見出しで始まるので先頭は削られず、変わるのは最後の行の末尾の空白だけで、markdown の意味はほぼ変わらない。対応は要らないと考える
- 実機(本物の herdr と leader の座席)で、leader が前文を計画に写し、指定の driver と model で座席を立てるかは見ていない(計画の Non-goals)。ralph はそれを検査しない(tech-debt の open の行)
- `run-verify.sh` を 1 本のコマンドとしては打っていない。AC9 は、verify の `run-static-verify.sh`(rc 0)と、この報告の `run-test.sh`(2 回目 rc 0)を合わせて見ている

## Insight event

依頼がこの報告のほかのファイルを変えないとしたので追記していない。`--events-dir` を scratch にして作った行は次のとおりで、追記は orchestrator に任せる(G1 を LOW 1 件として数えた)。

```
./scripts/insights-append.sh --slug split-preamble --flow standard --phase test --verdict pass --cycle auto --low 1 --source skill
{"schema":1,"ts":"2026-10-22T15:37:35Z","slug":"split-preamble","flow":"standard","phase":"test","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":1},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```

## Verdict

- Verdict: pass
- Pass: run-test.sh(2 回目)rc 0、shell 44 本と Go 8 パッケージ。go1.25.8 の org と cli も rc 0。新しいテスト 7 本と、行や確かめを足した既存のテスト 3 本(`TestLoadSplitPlan_ParsesFeatures`、`TestLoadSplitPlan_Rejects`、`TestOrgStartPlan_FlagCombinationsRefused`)は、3 回の実行、`-race`、canary の下で通った。mutant 41 本のうち 38 本が赤
- Fail: なし。1 回目の run-test.sh の 1 件は、この diff と関係しない前からの競合で、原因を確かめた(Failure analysis)
- Blocked: なし
- Gaps: G1(LOW、依存のある機能の行がない)、G2(INFO)。どちらも PR を止めない

## この報告のあとに埋めた穴

この報告のあとの 7a955d2c(`test:` の 1 コミット)が G1 と G2 を埋めた。変えたのは `internal/org/feature_test.go` と `internal/org/split_test.go` だけで、コードと雛形は変えていない。上の表と Verdict は /test の時点のまま残し、ここに足すだけにする。

- G1: `TestFeatureLeaderTask_Preamble` に、`auth-core` に依存する機能 `auth-ui` の行を足した。期待は、前文のない計画で作ったその機能の頭の部分に、前置きの 1 行・囲み・前文・本文を文字で足して組み立てる
- G2: `TestLoadSplitPlan_Preamble` と `TestFeatureLeaderTask_Preamble` に、前文の最後の行の末尾の空白 2 つが残る行を 1 つずつ足した
- mutant: 生き残りだった 3 本を 1 本ずつ入れて打ち直した。G1 は依存する機能の行が、G2 は読み込みの行が、G3 はタスクの行が、それぞれ 1 行だけ赤になった。ほかの 38 本は打ち直していない
- 打ち直したテスト: `gofmt -l internal/org/` は出力なし、`go vet ./internal/org/` は rc 0。前文に関わるテストは、手元の Go と `GOTOOLCHAIN=go1.25.8` の両方で ok。`go test -count=1 -timeout 30m ./internal/org/` は ok(383 秒)。`./scripts/run-test.sh` 全体と internal/cli は 7a955d2c の tree で打ち直していない
