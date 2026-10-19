# Test report: org-director-wait-bound

- Date: 2026-10-19(JST 22:37〜23:05)
- Plan: docs/plans/active/2026-10-19-org-director-wait-bound.md(8 段目の 8b-2c。Claude Code の director の受信箱の wait を 25 分で区切る)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 5c7fa9df(base 3a803a87、15 コミット、13 ファイル、+1,094/-67)。Go の変更は `internal/org/director_skill_sync_test.go` だけで、本番のコードは変わっていない。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)。実行中の load average は 8〜13
- Evidence: `docs/evidence/test-2026-10-19-org-director-wait-bound.md`(各コマンドの結果、shell の 41 本、worktree の状態)と `docs/evidence/test-2026-10-19-org-director-wait-bound-mutation.txt`(mutant 22 回の表、落ちたテストと問題の行)。run-test.sh の全ログは `docs/evidence/verify-2026-10-19-134151.log`(gitignore の対象)
- 追跡しているファイルは変えていない。mutant は scratchpad に `git archive HEAD` で展開した写しで打った。`git status --porcelain` は実行の前もあとも空だった。Go の実行は 1 つずつ順に打った

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。golang を選んだ。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 996 秒、rc 0 |
| `go test ./internal/org/ -run 'TestDirectorSkill' -count=1 -v -timeout 30m`(go1.26.0) | top-level 4、subtest 21 | 25 | 0 | 0 | `ok` の行で 0.553 秒、rc 0 |
| 同じコマンドを `GOTOOLCHAIN=go1.25.8` で | top-level 4、subtest 21 | 25 | 0 | 0 | `ok` の行で 0.356 秒、rc 0 |
| mutant(`git archive` の写し。go1.26.0) | 22 回 | 赤 16(狙いどおり) | - | - | 1 回 1〜6 秒 |
| 同じ 22 回(go1.25.8) | 22 回 | 判定と問題の行は go1.26.0 と同じ | - | - | 1 回 1〜6 秒 |

- run-test.sh の Go は cli(333.0 s)、config、org(428.7 s)を流し直した。insights、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。変わった Go のファイルは org の `_test.go` 1 本で、ほかのパッケージがコンパイルするものに入らないので、cached の結果はこの差分で古くならない
- shell は 41 本すべてが `OK` で終わり、集計の行に 0 でない失敗の数はなかった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- 絞った回の 4 本は `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`、`TestDirectorSkill_PinsTheAutoMergeSteps`、足した `TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap`(4 つの写しを読む。以下「写しのテスト」)、`TestDirectorSkillWaitBoundProblems`(表の 21 件。以下「表のテスト」)。go1.25.8 の出力は go1.26.0 と最後の `ok` の行の秒数だけが違う
- 既存のテストで変わったのは 109 行のコメントの計画の道(`active` を `archive` に)だけで、判定は変えていない。消したテストはない
- 記録にある揺れ(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、doctor の 2 本)は、この回では出なかった

## Coverage

- Statement: 本番のコードは変わっていないので、測る対象がない。足した関数(`directorSkillWaitBoundProblems` と helper の 5 つ)はテストのファイルにあり、Go の coverage はテストのファイルを数えない
- Branch: 測っていない。代わりに表の 21 件と mutant で判定の枝を見た。表は `N <= 0`、`N >= M`、`M` が上限を超える、0 件と 2 件の一致(N と M のそれぞれ)、分で割り切れない N、小節の外の `timeout_ms` と「分」、`--timeout-ms 0` の 3 つの形(途中、文末、折り返し)、箇条がない場合、ほかの小節にだけ箇条がある場合を通る
- Function: 同上
- Notes: skill の 4 つの写しは HEAD で byte 単位で同じ(sha256 `2956f50d…`)。3a803a87 の 4 つの写しも互いに同じ(`b5985650…`)

## Mutation

mutant は `git archive HEAD` の写しで打った。写しのテストは自分のソースの場所(`runtime.Caller`)から skill を読むので、写しの中では写しの skill を読む。HEAD のテストのファイルのまま、変えた skill を読ませられる。1 回ごとに 5 つのファイルを元に戻し、最後に `cmp` で元と同じことを確かめた。

| ID | 変えたこと | 結果 | 落ちたテストと問題の行 |
| --- | --- | --- | --- |
| M1a〜M1d | (1) 写しの 1 つで wait の `--timeout-ms 1500000` を `1800000` にする。4 つの写しで 1 回ずつ | 4 回とも赤 | 写しのテスト。その写しの道を付けて `N=1800000 must be less than the Monitor's timeout_ms M=1800000` と、`says 25 分, want N/60000=30` が 3 件 |
| M2 | (2) 箇条の Monitor の `1800000` を `1800001` にする | 赤 | 写しのテスト。`M=1800001 is over the Monitor tool's maximum 1800000` と、例外の段落の `timeout_ms` に `1800000` が M と違うこと |
| M3a | (3) 4 つの写しの Claude Code の箇条を 3a803a87 の箇条(`--timeout-ms 0`)に戻す。小節のほかは HEAD のまま | 赤 | 写しのテスト。写しごとに 4 件: M の一致が 0 件、`N=0 must be greater than 0`、`says 25 分, want N/60000=0`、`contains --timeout-ms 0` |
| M3b | (3) 4 つの写しを 3a803a87 の skill に丸ごと戻す(AC4 の「前の形に戻すと落ちる」) | 赤 | 写しのテスト。写しごとに 3 件: M の一致が 0 件、N=0、`--timeout-ms 0` |
| M4 | (4) 例外の段落の `1800000` を `600000` にする | 赤 | 写しのテスト。`has timeout_ms に 600000, but the Claude Code bullet's M=1800000` |
| M5a〜M5c | (5) 「25 分」の 3 か所(490 行の箇条、499 行の Bash の文、542 行の `note:` の箇条)を、1 か所ずつ別の写しで「20 分」にする | 3 回とも赤 | 写しのテスト。`says 20 分, want N/60000=25` |
| M6 | (6) helper の `n >= m` の確かめを消す。skill は HEAD のまま | 赤 | 表のテストの `N_equal_to_M`(問題が 0 件)と `N_greater_than_M`(1 件)。写しのテストは緑 |
| M6x | (6) の対照。4 つの写しで N を 1800000 に、「25 分」を 3 か所とも「30 分」にする(N == M で、分の数は合う)。テストは HEAD | 赤 | 写しのテスト。写しごとに `N=1800000 must be less than … M=1800000` の 1 件だけ |
| M6y | M6x の skill と M6 のテスト | 赤(表のテストだけ) | 写しのテストは緑 |
| M7 | 追加。箇条から Monitor の `timeout_ms` に `1800000` を消す(M がない) | 赤 | 写しのテスト。M の一致が 0 件 |
| M8 | 追加。Codex の箇条の `--timeout-ms 300000` を `--timeout-ms 0` にする | 赤 | 写しのテスト。`contains --timeout-ms 0` |
| G1 | 穴の確かめ(AC3)。4 つの写しの Claude Code の箇条から、時間切れで打ち直す 2 行を消す | 緑 | - |
| G1b | G1 に加え、終了コードの `1:` の箇条を 3a803a87 の文(Codex の時間切れだけを打ち直す)に戻す | 緑 | - |
| G2 | 穴の確かめ(AC3、D3)。4 つの写しから Monitor の期限切れの例外の段落を消す | 緑 | - |
| G2b | G2 に加え、失敗にしない終わり方の列挙を 3a803a87 の文に戻す(Monitor の期限切れの例外がなくなる) | 緑 | - |
| G3 | 穴の確かめ(AC2)。4 つの写しから Bash の `run_in_background` の 5 行を消す | 緑 | - |

- (1)〜(5) は写しのテストが落とす。(6) は写しのテストでは赤にならない。HEAD の skill は N < M なので、消した確かめが出す問題はもともとない。(6) を落とすのは表のテストの 2 件で、M6x と M6y から、写しが N == M に変わったときに落とすのはこの確かめだけだと分かる(ほかの確かめは、分の数を合わせた変更を通す)
- 計画の Edge cases との対応: 写しの 1 つだけ数字が違う(M1a〜M1d、M2、M4、M5a〜M5c)、N が M 以上(M1、M6x、表の 2 件)、M が 1800000 を超える(M2)、Monitor の数字がない(M3a、M3b、M7)、`--timeout-ms 0` が残る(M3a、M3b、M8)。どれも赤になった

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | run-test.sh と絞った回で、失敗したテストはない。mutant の赤は狙ったもの | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Monitor で見張らせた `--timeout-ms 0` の wait が Monitor の期限で SIGTERM を受け、director が失敗として止まる(この計画が直す不具合) | 固定された。skill を前の形に戻すと写しのテストが落ちる | mutant M3a、M3b |
| skill が wait の時間切れの文 `no open inbox item arrived within` を名指す | pass(go1.26.0、go1.25.8) | `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`。evidence の 1・2 節 |
| skill の merge の手順(auto_merge) | pass(go1.26.0、go1.25.8) | `TestDirectorSkill_PinsTheAutoMergeSteps`。evidence の 1・2 節 |
| 全体(Go 8 パッケージ、shell 41 本) | pass | run-test.sh。evidence の 3 節 |

## Test gaps

- T1(LOW): 足したテストが固定するのは数と `--timeout-ms 0` だけで、AC2 と AC3 の文は固定しない。G1〜G3、G1b、G2b のように 4 つの写しを揃えて消すか前の文に戻すと、TestDirectorSkill の 4 本はどれも緑のままだった。`check-skill-sync.sh` は 4 つの写しが同じことしか見ないので、揃えた変更には気づかない。G1b は Claude Code の director が時間切れの wait を打ち直すと名指す 2 か所を、G2b は Monitor の期限切れの例外を、skill から消す。どちらも、この計画が Claude Code の director の振る舞いとして足した規則である(G1b のあとも、失敗の段落の「時間を区切った wait の時間切れは失敗にしない」という一般の文は、3a803a87 と同じく残る)。`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は時間切れの文が skill のどこかにあれば通り、Codex の箇条にも同じ文があるので、Claude Code の箇条の文は固定しない。AC4 が求めるのは数と `--timeout-ms 0` だけなので、受け入れ条件は満たしている。直すなら、`TestDirectorSkill_PinsTheAutoMergeSteps` と同じ書き方(`withoutSpaces` の断片を順に探す)で、`### wait で待つ` の小節の中の次の断片を固定する: Claude Code の箇条の「Codex と同じくもう一度打つ」、`1:` の箇条の「Claude Code でも Codex でも上のとおり打ち直す」、例外の段落の「その wait を見張っていた Monitor の期限切れの知らせ」と「失敗とした終わりは Monitor の期限切れだったと人に伝える」。G1b と G2b がそのテストで赤になることを確かめる
- T2(INFO): M6 の確かめ(`n >= m`)を、写しのテストは HEAD の skill では使わない。N < M の skill では出す問題がないため。表のテストの 2 件が固定しており、穴ではない
- 確かめていないこと: skill の文に従う director の振る舞い(実機の確認は self-review の直しの前の 5aea657f の文で打たれ、dc98d5ec と de2b05d1 の文では打たれていない。これは livefire の evidence と verify report に書かれている)。本物の 25 分と 30 分の長さ、Monitor の最大と既定の値(verify の V2)。どれも behavioral test の範囲の外で、/test では打っていない

## Verdict

- Verdict: pass
- Pass: run-test.sh(shell 41 本と Go 8 パッケージ、996 秒、rc 0)、絞った回(go1.26.0 と go1.25.8 で 25 本ずつ PASS)、計画の Regression の 2 本、頼まれた mutant (1)〜(6) の 12 回すべてが赤((6) は表のテストで赤。写しのテストは N < M の HEAD の skill では緑で、それは作りどおり)、計画の Edge cases の 5 つすべてが赤
- Fail: なし
- Blocked: なし
- 残る穴: T1(LOW)。AC2 と AC3 の文を固定するテストがない。HEAD の skill の文は正しく、テストが固定していないだけなので、マージを止めるものではない
