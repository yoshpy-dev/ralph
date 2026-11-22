# Test report: issue-depends-wait

## Run 2 (cycle 2)

- Date: 2026-11-22(実行は 13:24:07〜13:48:16 JST。パイプラインの cycle 2、`cycle-count.json` は `cycle: 2`)
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `60862f1e`(worktree は clean、push 済み。base は origin/main `2a069b50`)。cycle 1 のあとの変更は、cross-review の 2 件の直し `ff59a0e1`(頼る先の merge の確かめに (c) を足し、「merge を待つ issue」の 2 で頼る先のブランチの `ready` を確かめ直す)と、cycle 2 の self-review の LOW の直し `792ffd97`、それに報告だけ。skill の文とテストだけで、Go の本番コードは変えていない。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `ps -axo pid,command` で `run-test.sh` と `run-verify.sh` がないことを見た(別の tester の `go test -overlay … ./internal/org/` の変異の run が 2 つ走っていたが、run-test.sh ではない)。起動は cycle 1 と同じで、perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。そのあと、`-v` の run、`(cached)` だった 8 パッケージの `-count=1` の打ち直し、変異 20 個、cycle 1 のテストを HEAD の skill に当てる run を足した
- Evidence: `docs/evidence/test-2026-11-22-issue-depends-wait-c2.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない。cycle 1 の log はそのまま残した)。run-test.sh の出力の後ろに、1 分ごとの load と空き容量、`-v` の run の要約、打ち直しの出力、変異の表と落ちたテスト、`~/.codex/config.toml` と herdr のプロセスの前後を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-042407.log`(同じく追跡外)

### Test execution (cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 13:24:07〜13:47:29 JST、1,402 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | cycle 1 と同じ 2 つ(`test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」) | 約 11 分(13:24:07〜13:35 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` が 8(次の行で打ち直した) | 約 12 分(13:35 ごろ〜13:47:29)、下の表 |
| `(cached)` の 8 パッケージを `go test -count=1` で打ち直し | 8 | 8 | 0 | 0 | 13:47:59〜13:48:16、rc 0 |
| `go test -count=1 -v -run 'TestDirectorSkill\|TestOrgSkill\|IssueDepends\|SplitQuestions' ./internal/org/` | 116(`=== RUN` の行) | 116 | 0 | 0 | 1.147 s、rc 0 |

| パッケージ | run-test.sh | `-count=1` の打ち直し |
| --- | --- | --- |
| `cmd/ralph` | ok (cached) | ok 3.894 s |
| `internal/cli` | ok 741.066 s | — |
| `internal/config` | ok 11.048 s | — |
| `internal/gitroot` | ok (cached) | ok 3.797 s |
| `internal/insights` | ok (cached) | ok 0.442 s |
| `internal/org` | ok 734.152 s | — |
| `internal/org/driver` | ok (cached) | ok 1.703 s |
| `internal/org/protocol` | ok (cached) | ok 1.394 s |
| `internal/scaffold` | ok (cached) | ok 2.807 s |
| `internal/toolchain` | ok (cached) | ok 16.257 s |
| `internal/upgrade` | ok (cached) | ok 2.579 s |

golang pack の verifier は `go test ./...` を `-count=1` なしで回し、GOCACHE は worktree の `.harness/cache/go-build` を使う。`(cached)` は、テスト binary と、テストが読んだファイルと環境変数が、同じ cache の前の run と同じだったという意味になる。`cmd/ralph` は templates を embed するルートのパッケージに頼るので、cycle 2 の skill の templates で回った run の結果のはずだが、どの run かは確かめていない。cycle 1 の run(`(cached)` 0)とそろえるため、8 つとも `-count=1` で打ち直し、すべて通った。`internal/cli` と `internal/org` は、golang pack の 1 パッケージの上限 30 分のおよそ 41% で終わった。

shell の段の `PASS` の行は 5,390 で、cycle 1 と同じ数だった。数字と一時ファイルの名前を伏せて並べた `PASS` の行を cycle 1 の log と比べると、違いは cycle 1 の log の末尾に足した要約の 1 行だけで、通ったテストの組は同じだった。

1 分 load は 5.83〜26.72(13:24:07 から 1 分ごとに 24 回測った)。26.72 は Go の段の 13:38 の値で、別の tester の変異の run と重なったとき。私の side run(`-v` の run、テスト binary の build、変異 20 個、cycle 1 のテストを当てる run)はどれも shell の段(13:24〜13:31 ごろ)に済ませた。打ち直しは run-test.sh が終わったあとに回した。空き容量は 48 GiB から 47 GiB だった。

run の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じだった。読まれたかどうかは測っていない。herdr のプロセス(`herdr server` の pid 59855、11 月 20 日 13:39:32 起動)は前後で同じで、触れていない。worktree の `git status --porcelain` は run のあとも空だった。

#### cycle 2 で足したテストと変えたテスト

`-v` の run で、足した 1 関数と固定を変えた 5 関数が PASS し、SKIP は 0 だった。`=== RUN` の行は cycle 1 の 115 から 1 増えた。

- 足した関数: `TestDirectorSkill_StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck`(「merge を待つ issue」の 2 と引き継ぎ。頼る先のブランチの `ready` を確かめ直す文、done にする note、その note が `ValidateFollowupNote` に通り `'` を含まないこと)
- 固定を変えた関数(`internal/org/skill_issue_depends_test.go`): `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`((c) の文、gh のコマンド、時刻の形、読めないときの扱い。`Org.now()` が JST の時計でも `2026-11-22T03:04:05Z` の形を返すこと。JSON の名前に `pr`・`number`・`ts`・`mergedAt` を足した)、`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling`(skill 全体で 1 回だけ現れる文に、(c)、gh のコマンド、読めないときの 2 文を足した)、`TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding`(引き継ぎの問いの条件に、2 で止まったブランチの例外を足した)、`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`(org skill は確かめを名前で呼ぶ文に替え、節に `(a)`・`(b)`・`(c)` がないことを見る)
- 固定を変えた関数(`internal/org/skill_split_questions_test.go`): `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`(fragment 1 つ)

#### 直しとテストの対応

| 直し | テスト | 結果 |
| --- | --- | --- |
| cross-review #1: (c)(PR の merge が結びつきの記録より前でない) | `FilingWaits…`(step 1 の (c) の文と gh のコマンド、`Org.now()`、`checkMergeOrderJSONNames`)、`DependencyMergeQuestionIsAskedOnceAtFiling` | PASS。C1・C1b・C4・C5・C9・C12 と G1〜G4 で落ちる |
| cross-review #2: 2 で頼る先のブランチの `ready` を確かめ直す | `StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck` | PASS。C2・C2b・C2c・C6・C7 で落ちる |
| C2-L1: org skill は確かめを名前で呼び、条件を並べない | `TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge` | PASS。C3 で落ちる。C3c は落ちない(Test gaps の 2) |
| C2-L2: 読めない時刻は数えず、2 が報告のたびに読み直し、人には 1 回だけ伝える | `FilingWaits…`(読めないときの 4 文)、`StartCheck…`(2 で done にしない文) | PASS。C7・C8・C11 で落ちる |
| C2-L3: 引き継ぎの例外を、それが直す規則の条件に入れる | `TakeoverAndStartQuestion…`、`StartCheck…` の引き継ぎ | PASS。C10 で落ちる |

#### 変異 (cycle 2)

cycle 1 と同じく、HEAD の `git archive` の写しで `internal/org` のテスト binary を 1 つ作り、写しの skill を替えて `-test.run 'TestDirectorSkill|TestOrgSkill'` で回した。cycle 1 と違い、どの変異も 4 写しを同じに替えた。写しの食い違いは `check-sync.sh` が見つけるので、テストだけが見張るのは 4 写しをそろえた書き換えになる。置き換える文字列は、空白と改行を無視して各写しに 1 回だけ現れることを確かめてから替えた。対照(変異なし)は rc 0。毎回元に戻し、最後に 8 つの写しが worktree と同じことを `cmp` で確かめた(違いは 0)。Go の 4 つは、写しの本番コードを `-overlay` で替えて回した。

依頼された 3 つは、C1((c) を消す)、C2(`ready` の確かめ直しを消す)、C3(org skill に (a)・(b) を並べ直す)にあたる。

| # | 替えたもの | 変異 | 落ちたテスト |
| --- | --- | --- | --- |
| C1 | director の 4 写し | (c) を消す。規則の文を「(a) か (b) に当たるときだけ」に戻し、(c) の項と「(c) に当たらないときも聞かずに待つ」の文を消す | `FilingWaits…`、`DependencyMergeQuestionIsAskedOnceAtFiling` |
| C1b | 同じ | 規則の文だけを (a)・(b) に戻す((c) の項は残す) | 同じ 2 つ |
| C2 | 同じ | 2 の、頼る先のブランチの `ready` を確かめ直す文のかたまりを消す | `StartCheck…` |
| C2b | 同じ | 頼る先のブランチの `ready` を「確かめずに待ち終えたと数える」に替える | `StartCheck…` |
| C2c | 同じ | 2 の `--note` の文言だけを「頼る先のものでない」に替える(引き継ぎの文言は残す) | `StartCheck…` |
| C3 | org の 4 写し | 節の最後に「確かめは (a) `split` と `feature`、(b) `feature` と `issue` で名指す結びつきによる。」を足す(挿入だけ) | `TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`(`(a)`・`(b)` がないことの確かめだけで落ちる) |
| C3c | 同じ | 同じ内容を `(a)`・`(b)` の印なしで足す(挿入だけ。穴を測るための変異) | 落ちない |
| C4 | director の 4 写し | (c) の向きを「より後でない」に替える | `FilingWaits…`、`DependencyMergeQuestion…` |
| C5 | 同じ | merge の時刻を、(a) か (b) に当たる件だけでなく全部の件で読む | `FilingWaits…` |
| C6 | 同じ | 引き継ぎの、`DONE_NOTE` の done の件を数えない文を消す | `StartCheck…` |
| C7 | 同じ | 2 で、時刻を読めないときも done にして記録し直す | `StartCheck…` |
| C8 | 同じ | 1 の「(c) に当たらないと決まったのではないので、2 の done と記録し直しは打たない」を消す | `FilingWaits…` |
| C9 | 同じ | 読めない時刻を「数える」に替える | `FilingWaits…`、`DependencyMergeQuestion…` |
| C10 | 同じ | 引き継ぎの問いの条件から、2 で止まったブランチの例外を消す | `StartCheck…`、`TakeoverAndStartQuestion…` |
| C11 | 同じ | 読めなかったことを「確かめるたびに」伝えるに替える | `FilingWaits…`、`DependencyMergeQuestion…` |
| C12 | 同じ | (c) の文を 2 にもう 1 つ書く(挿入だけ) | `DependencyMergeQuestion…`(1 回だけの数) |
| G1 | `spawn.go:416`(overlay) | `Org.now()` の `.UTC()` を外す | `FilingWaits…`(JST の時計で `+09:00` の形になる) |
| G2 | `mergeorder.go` の `MergeOrderBinding.TS`(overlay) | JSON の名前を `bound_at` に替える | `FilingWaits…` |
| G3 | `mergeorder.go:750` の `ghPullRequest.MergedAt`(overlay) | JSON の名前を `merged_at` に替える | `FilingWaits…` |
| G4 | `mergeorder.go:263` の `MergeOrderEntry.PR`(overlay) | JSON の名前を `pull_request` に替える | `FilingWaits…` |

20 個のうち 19 個が落ちた。落ちなかった C3c は穴を測るために足した変異で、Test gaps の 2 に書いた。

#### 古いテストの固定の変え方 (cycle 2)

cycle 1 の HEAD `95d755b2` の `skill_issue_depends_test.go` と `skill_split_questions_test.go` を overlay で HEAD に当て、`-test.run 'TestDirectorSkill|TestOrgSkill' -v` で回した。110 が PASS、5 が FAIL だった。落ちたのは cycle 2 で書き換えた 4 つの fragment だけで、4 写しとも同じだった。規則の文(`FilingWaits…` と `DependencyMergeQuestion…` の 2 か所で読む)、引き継ぎの「ただし、follow-up のないそのブランチが…」の文、org skill の依存の文、`skill_split_questions_test.go` の「(done の件も含めて)なく」の文の 4 つになる。どれも条件を足す書き換え((c)、2 で止まったブランチの例外、`DONE_NOTE` の done を数えないこと)か、org skill の条件を director の 1 に移す書き換え(条件は director 側のテストが固定している)で、テストの意図を弱めたものではない。

### Coverage (cycle 2)

- Statement: 測っていない。cycle 2 も skill の文とテストだけで、Go の本番コードを変えていない
- Branch: 該当なし
- Function: 該当なし
- Notes: cycle 2 で足した確かめのうち本番コードを通るのは、`Org.now()` の形の確かめ(`spawn.go`)と、`checkMergeOrderJSONNames` が reflect で読む struct の tag(`MergeOrderEntry.PR`、`MergeOrderPR.Number`、`MergeOrderBinding.TS`、`ghPullRequest.MergedAt`)だけ。効き目は G1〜G4 で確かめた

### Failure analysis (cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

flaky と知られているテストも、この run では落ちなかった。打ち直したのは `(cached)` の 8 パッケージだけで、失敗の打ち直しはしていない。

### Regression checks (cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 で固定した依存の待ちの規則((a)・(b)、起票で 1 回だけの問い、まだないブランチの記録) | 残っている | cycle 1 のテストを当てる run で、落ちたのは cycle 2 で書き換えた 4 つの fragment だけ |
| shell のテストの組 | 変わっていない | `PASS` の行の組が cycle 1 と同じ |
| merge-order の JSON の `binding`(`split`・`feature`・`issue`)と、PR のないブランチの follow-up の `waiting` | 変わっていない | `internal/org` が PASS。`checkMergeOrderJSONNames` と G2・G4 |

### Test gaps (cycle 2)

1. cycle 1 の 1 と同じく、director の実際の動きを追う run はない。(c) について固定したのは、台帳の `binding.ts` と gh の `mergedAt` を時刻として比べるという文と、台帳側の形(`Org.now()`)と、JSON の名前まで。director が文字列ではなく時刻として比べること、gh が実際に `Z` の形で返すことは確かめていない。
2. 新しい穴: org skill に条件を並べないことの確かめは、節に `(a)`・`(b)`・`(c)` の文字があるかだけを見る。印なしで条件を並べ直す文(C3c)は落ちない。cycle 1 の 2(問いの別の言い方)と同じく、言い回しに頼る確かめの限界になる。
3. cycle 1 の 2(`directorDependsAskingSentences` が別の言い方の問いを数えない)は残っている。台帳の 562 行の (2) にあたる。同じ行の Trigger (1)(`skill_issue_depends_test.go` に次のテストを足したとき)は、cycle 2 の `StartCheck…` で当たったが払っていない(verify の V8)。
4. AC5 の `check-sync.sh` と `check-skill-sync.sh` は test mode では回らない。4 写しが同じことは `cmp` で見た(director と org の各 3 組、どれも同じ)。スクリプトの rc は verify の報告による。

## Verdict

- Pass: 11 Go パッケージと 46 本の shell のテストが通った(run-test.sh rc 0、1,402 s)。`(cached)` の 8 パッケージは `-count=1` で打ち直して通った。足した 1 関数と固定を変えた 5 関数は `-v` で PASS し(116 件)、SKIP は 0。変異 20 個のうち 19 個が落ち、落ちなかった 1 個(C3c)は穴を測る変異で、Test gaps の 2 に書いた
- Fail: 0
- Blocked: 0。`/sync-docs` に進んでよい

## Run 1 (cycle 1、履歴)

- Date: 2026-11-22(実行は 11:48:20〜12:08:53 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `95d755b2`(worktree は clean。origin/main `2a069b50` を取り込み済み)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `ps -axo pid,command` で `run-test.sh` と `run-verify.sh` がないことを見た(別の worktree `org-result-missing` の `go test ./internal/org/ -run ...` は走っていたが、run-test.sh ではない)。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。そのあと、足したテストの `-v` の run、skill の文の変異 8 個、merge-order の overlay の変異 2 個、base の古いテストを HEAD の skill に当てる run を足した
- Evidence: `docs/evidence/test-2026-11-22-issue-depends-wait.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力の後ろに、1 分ごとの load と空き容量、`-v` の run の要約、変異の表と落ちたテスト、`~/.codex/config.toml` と herdr のプロセスの前後を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-024820.log`(同じく追跡外)

### Test execution

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

#### 足したテストと変えたテスト

`-v` の run で、足した 7 関数と、固定を変えた 4 関数が PASS し、SKIP は 0 だった。skill を読むテストは写しがないと `t.Skipf` するが、ここでは skip していない。

- 足した関数(`internal/org/skill_issue_depends_test.go`): `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`、`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling`、`TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`、`TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding`、`TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues`、`TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`、`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`
- 固定を変えた関数(`internal/org/skill_split_questions_test.go`): `TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes`、`TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`、`TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`、`TestDirectorSkill_AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`

#### 計画のテストと AC の対応

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

#### 変異

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

#### 古いテストの固定の変え方

既存のファイルで変えたのは `skill_split_questions_test.go` だけなので、origin/main の版をそのまま overlay で HEAD に当て、`-test.run 'TestDirectorSkill|TestOrgSkill'` で回した。落ちたのは、diff で書き換えた 6 つの fragment(「人への聞き方」の 1、「起票と記録」の 1 の 2 つと 2 の 1、「merge を待つ issue」の引き継ぎの 1 と 2 の 1)だけで、4 写しとも同じだった。ほかの古い fragment は HEAD の skill でも通る。書き換えた 6 つは、計画どおり依存の規則に合わせた変更で、テストの意図を弱めたものではない(「重ならない機能は待たない」は「重ならず頼る先もない機能は待たない」になり、頼る先のある機能は待つと固定するようになった)。

### Coverage

- Statement: 測っていない。この変更は skill の文とテストだけで、Go の本番コードを変えていない
- Branch: 該当なし
- Function: 該当なし
- Notes: 足したテストのうち、本番コードを通るのは `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`(`FollowupStore.Add` と merge-order の follow-up の判定)と、merge-order の JSON の名前を reflect で見る `checkMergeOrderJSONNames` だけ。ほかは skill の文の固定で、効き目は上の変異で確かめた

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

flaky と知られているテストも、この run では落ちなかった。打ち直しはしていない。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 予約の重なりだけで決まる今の待ちの文(重なる org のブランチを全部待つ、`feature:` の行のない org には待つブランチがない、待つブランチがなければ着手を自動で聞かない) | 残っている | `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` の変えていない fragment が PASS。base の版を当てても、書き換えた fragment 以外は通る |
| PR のないブランチの follow-up を merge-order が `waiting` と出す | 変わっていない | `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` と既存の merge-order のテスト(run-test.sh の `internal/org`)が PASS。G1・G2 で落ちる |
| merge-order の JSON の `binding` に `split`・`feature`・`issue` がある(#365) | 変わっていない | `TestMergeOrder_BindingIssue`、`TestMergeOrder_ResultJSONShape` が PASS。`checkMergeOrderJSONNames` が struct の tag を照合 |

### Test gaps

1. どのテストも skill の文を固定するだけで、director が手順どおりに動くことは確かめていない。起票のときに 1 回だけ聞くこと、まだないブランチの follow-up を記録して待つこと、頼る先の PR の merge で待つ側の follow-up が `ready` になって着手を聞くことを、実際の director のセッションで追った run はない。fixture で追えるのは、`FollowupAdd` が存在しないブランチを受け付けることと、merge-order がそれを `waiting` と出すことまで。
2. 「人に聞く文がない」ことの確かめ(`directorDependsAskingSentences`)は、`` `feature`だけが合 `` と `聞く` を両方含む文だけを数える。「slug だけが同じなら人に尋ねる」のような別の言い方の問いを step 4、引き継ぎ、「merge を待つ issue」の 2 に足しても落ちない。D3 のように同じ言い方で足せば落ちる。
3. fragment の照合は空白を無視した順序つきの部分一致なので、固定した文のあいだに矛盾する文を足しても、それが固定した文を壊さないかぎり落ちない。skill の文を固定するテストに共通の限界で、この branch で増えたものではない。
4. 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 2026-11-22 の項目を固定するテストはない。verify の V1・V2(`03ced266` より前の文のまま)は `/sync-docs` に回っている。
5. AC5 の `check-sync.sh` と `check-skill-sync.sh` は test mode の run-test.sh では回らない。この報告では 4 写しの `cmp` だけを見た。スクリプトの rc は verify の報告による。

### Verdict(cycle 1)

- Pass: 11 Go パッケージと 46 本の shell のテストが通った(run-test.sh rc 0、1,233 s、`(cached)` 0)。足した 7 関数と固定を変えた 4 関数は `-v` で PASS し、SKIP は 0。変異 10 個はすべて落ちた
- Fail: 0
- Blocked: 0。PR に進んでよい
