# Test report: issue-depends-wait

- Date: 2026-11-22(実行は 11:48:20〜12:08:53 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `95d755b2`(worktree は clean。origin/main `2a069b50` を取り込み済み)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `ps -axo pid,command` で `run-test.sh` と `run-verify.sh` がないことを見た(別の worktree `org-result-missing` の `go test ./internal/org/ -run ...` は走っていたが、run-test.sh ではない)。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。そのあと、足したテストの `-v` の run、skill の文の変異 8 個、merge-order の overlay の変異 2 個、base の古いテストを HEAD の skill に当てる run を足した
- Evidence: `docs/evidence/test-2026-11-22-issue-depends-wait.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力の後ろに、1 分ごとの load と空き容量、`-v` の run の要約、変異の表と落ちたテスト、`~/.codex/config.toml` と herdr のプロセスの前後を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-024820.log`(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 11:48:20〜12:08:53 JST、1,233 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0。stub の古い git の行は通る) | 約 10 分(11:48:20〜11:58 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 11 分(11:58 ごろ〜12:08:53)、下の表 |
| `go test -count=1 -v -run 'TestDirectorSkill\|TestOrgSkill\|IssueDepends\|SplitQuestions' ./internal/org/` | 115(`=== RUN` の行) | 115 | 0 | 0 | 0.98 s、rc 0 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.686 s |
| `internal/cli` | ok | 642.586 s |
| `internal/config` | ok | 11.077 s |
| `internal/gitroot` | ok | 2.405 s |
| `internal/insights` | ok | 1.863 s |
| `internal/org` | ok | 646.487 s |
| `internal/org/driver` | ok | 3.535 s |
| `internal/org/protocol` | ok | 3.597 s |
| `internal/scaffold` | ok | 2.676 s |
| `internal/toolchain` | ok | 20.208 s |
| `internal/upgrade` | ok | 3.785 s |

`internal/cli` と `internal/org` は、golang pack の 1 パッケージの上限 30 分のおよそ 36% で終わった。`RALPH_VERIFY_SCOPE=full` で回したので、`(cached)` のパッケージはなく、`-count=1` の打ち直しは要らなかった。shell の段の `PASS` の行は 5,390 で、org-start-issue の run(`docs/reports/test-2026-11-22-org-start-issue.md`)と同じ数だった。この branch は shell のテストを足していない。

1 分 load は run の全体で 5.09〜18.52(11:48:20 から 1 分ごとに 21 回測った)。Go の段(11:59 以降)は 5.09〜8.44 だった。私の side run(`-v` の run、テスト binary の build、変異 10 個)はどれも shell の段(11:49〜11:52)に済ませ、Go の段には何も回していない。base のテストを当てる run は run-test.sh が終わったあとに回した。空き容量は 56 GiB から 53 GiB で、ENOSPC の心配はなかった。

run の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じだった。書かれていない。読まれたかどうかは測っていない(この machine では読みで atime が変わらない)。足したテストは skill の 4 写しと `t.TempDir()` の状態ディレクトリだけを読み書きする。herdr のプロセス(`herdr server` の pid 59855)は前後で同じで、触れていない。

### 足したテストと変えたテスト

`-v` の run で、足した 7 関数と、固定を変えた 4 関数が PASS し、SKIP は 0 だった。skill を読むテストは写しがないと `t.Skipf` するが、ここでは skip していない。

- 足した関数(`internal/org/skill_issue_depends_test.go`): `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`、`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling`、`TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`、`TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding`、`TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues`、`TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`、`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`
- 固定を変えた関数(`internal/org/skill_split_questions_test.go`): `TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes`、`TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`、`TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`、`TestDirectorSkill_AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn` の step 1 | PASS。変異 D1・D4・D7 で落ちる |
| AC2 | 同じテストの step 4、`TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`(状態ディレクトリへの `FollowupAdd` と merge-order の fixture)、`TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding` の引き継ぎ | PASS。overlay の変異 G1・G2 で落ちる |
| AC3 | `TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues` | PASS |
| AC4 | `TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`(例の印の行が印の行の形に当たり、ブランチと note が `ValidateFollowupAdd` に通る) | PASS。変異 D5 で落ちる |
| AC5 | 4 写しが同じことは `cmp` で見た(director と org の各 3 組、どれも同じ)。`check-sync.sh` と `check-skill-sync.sh` は test mode では回らないので、結果は verify の報告(rc 0)による | 写しは同じ |
| AC6 | `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`((a)・(b)・問い・merge-order の JSON の名前を reflect で照合)、`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling`、`TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding`、`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`。JSON の `binding.issue` は既存の `TestMergeOrder_BindingIssue` と `TestMergeOrder_ResultJSONShape`(run-test.sh の `internal/org` で通った) | PASS。変異 D1・D2・D3・D6・O1 で落ちる |
| AC7 | `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn` の step 2、`TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues` | PASS |

計画の Edge cases との対応は次のとおり。頼る先がすでに merge 済みの場合は step 1 の確かめの規則と問い(D1)、頼る先が走っている org の場合は step 1 の `feature:` と `issue:` の行の照合と slug だけが合う org の扱い(D7)、頼る先が起票した機能と頼る先が 2 つの場合は例(`api` と `jobs-close`、D5)、重なりと依存が同じブランチを指す場合は step 1 の「同じものは 1 つにする」(D4)が受け持つ。

### 変異

skill は実行時にディスクから読むので、HEAD の `git archive` の写しで `internal/org` のテスト binary を 1 つ作り、写しの skill を 1 か所ずつ替えて回した。置き換える文字列は、空白と改行を無視して 1 回だけ現れることを確かめてから替えた(D5 だけは空行を消す変異なので、文字どおりに照合した)。毎回元に戻し、最後に 8 つの写しが worktree と同じことを `cmp` で確かめた。対照(変異なし)は、足した 7 関数の絞り込みと `-test.run 'TestDirectorSkill|TestOrgSkill'` のどちらも rc 0。1 つの写しだけを替えても、4 写しのどれを替えたかによらず内容のテストが落ちた。

| # | 写し | 変異 | 落ちたテスト |
| --- | --- | --- | --- |
| D1 | `.claude/skills/director` | (b) から「`feature` がその機能の slug」の条件を消す | `FilingWaits…`、`DependencyMergeQuestionIsAskedOnceAtFiling`(1 回だけの数) |
| D2 | `.agents/skills/director` | 「merge を待つ issue」の 2 で、follow-up のあるブランチも `feature` だけが合えば聞くと書き換える | `TakeoverAndStartQuestion…`、`DependencyMergeQuestionIsAskedOnceAtFiling`(follow-up のないブランチ以外への問い) |
| D3 | `templates/base/.claude/skills/director` | 「起票と記録」の 4 に「`binding` の `feature` だけが合うときは、記録の前に人に聞く。」を足す(挿入だけ) | `DependencyMergeQuestionIsAskedOnceAtFiling` だけ(step 4 は聞かない) |
| D4 | `templates/base/.agents/skills/director` | 「重なる org のブランチと頼る先のブランチで同じものは 1 つにする。」を消す | `FilingWaits…` |
| D5 | `.claude/skills/director` | 例の `api` のラベルとコードブロックのあいだの空行を消す | `IssueDependsExampleFollowsAFiledDependency` |
| D6 | `.claude/skills/director` | 問いの行を「merge を待つ issue」の 2 にもう 1 つ書く | `DependencyMergeQuestionIsAskedOnceAtFiling`(2 回)、`TakeoverAndStartQuestion…`(広い絞り込みでは `AsksWhetherToStart…` も) |
| D7 | `.agents/skills/director` | 「`feature:` の行の slug だけが合う org は頼る先の機能の org ではない」の文を消す | `FilingWaits…` |
| O1 | `.claude/skills/org` | 「`feature` だけが合うときは、director が起票のときに 1 回だけ人に聞く。」を消す | `TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge` |
| G1 | `mergeorder_followup.go:315`(overlay) | PR のないブランチの DETAIL を `has no PR yet:` から `has no open PR:` に替える | `DependencyBranchIsRecordedBeforeItExistsAndWaits` |
| G2 | 同じ行(overlay) | PR のないブランチの状態を `waiting` から `unknown` に替える | `DependencyBranchIsRecordedBeforeItExistsAndWaits` |

10 個とも落ちた。実装者の報告にある 25 個の変異は打ち直していない。上の 10 個はそれとは別に選んだ。

### 古いテストの固定の変え方

既存のファイルで変えたのは `skill_split_questions_test.go` だけなので、origin/main の版をそのまま overlay で HEAD に当て、`-test.run 'TestDirectorSkill|TestOrgSkill'` で回した。落ちたのは、diff で書き換えた 6 つの fragment(「人への聞き方」の 1、「起票と記録」の 1 の 2 つと 2 の 1、「merge を待つ issue」の引き継ぎの 1 と 2 の 1)だけで、4 写しとも同じだった。ほかの古い fragment は HEAD の skill でも通る。書き換えた 6 つは、計画どおり依存の規則に合わせた変更で、テストの意図を弱めたものではない(「重ならない機能は待たない」は「重ならず頼る先もない機能は待たない」になり、頼る先のある機能は待つと固定するようになった)。

## Coverage

- Statement: 測っていない。この変更は skill の文とテストだけで、Go の本番コードを変えていない
- Branch: 該当なし
- Function: 該当なし
- Notes: 足したテストのうち、本番コードを通るのは `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`(`FollowupStore.Add` と merge-order の follow-up の判定)と、merge-order の JSON の名前を reflect で見る `checkMergeOrderJSONNames` だけ。ほかは skill の文の固定で、効き目は上の変異で確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

flaky と知られているテストも、この run では落ちなかった。打ち直しはしていない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 予約の重なりだけで決まる今の待ちの文(重なる org のブランチを全部待つ、`feature:` の行のない org には待つブランチがない、待つブランチがなければ着手を自動で聞かない) | 残っている | `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` の変えていない fragment が PASS。base の版を当てても、書き換えた fragment 以外は通る |
| PR のないブランチの follow-up を merge-order が `waiting` と出す | 変わっていない | `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` と既存の merge-order のテスト(run-test.sh の `internal/org`)が PASS。G1・G2 で落ちる |
| merge-order の JSON の `binding` に `split`・`feature`・`issue` がある(#365) | 変わっていない | `TestMergeOrder_BindingIssue`、`TestMergeOrder_ResultJSONShape` が PASS。`checkMergeOrderJSONNames` が struct の tag を照合 |

## Test gaps

1. どのテストも skill の文を固定するだけで、director が手順どおりに動くことは確かめていない。起票のときに 1 回だけ聞くこと、まだないブランチの follow-up を記録して待つこと、頼る先の PR の merge で待つ側の follow-up が `ready` になって着手を聞くことを、実際の director のセッションで追った run はない。fixture で追えるのは、`FollowupAdd` が存在しないブランチを受け付けることと、merge-order がそれを `waiting` と出すことまで。
2. 「人に聞く文がない」ことの確かめ(`directorDependsAskingSentences`)は、`` `feature`だけが合 `` と `聞く` を両方含む文だけを数える。「slug だけが同じなら人に尋ねる」のような別の言い方の問いを step 4、引き継ぎ、「merge を待つ issue」の 2 に足しても落ちない。D3 のように同じ言い方で足せば落ちる。
3. fragment の照合は空白を無視した順序つきの部分一致なので、固定した文のあいだに矛盾する文を足しても、それが固定した文を壊さないかぎり落ちない。skill の文を固定するテストに共通の限界で、この branch で増えたものではない。
4. 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 2026-11-22 の項目を固定するテストはない。verify の V1・V2(`03ced266` より前の文のまま)は `/sync-docs` に回っている。
5. AC5 の `check-sync.sh` と `check-skill-sync.sh` は test mode の run-test.sh では回らない。この報告では 4 写しの `cmp` だけを見た。スクリプトの rc は verify の報告による。

## Verdict

- Pass: 11 Go パッケージと 46 本の shell のテストが通った(run-test.sh rc 0、1,233 s、`(cached)` 0)。足した 7 関数と固定を変えた 4 関数は `-v` で PASS し、SKIP は 0。変異 10 個はすべて落ちた
- Fail: 0
- Blocked: 0。PR に進んでよい
