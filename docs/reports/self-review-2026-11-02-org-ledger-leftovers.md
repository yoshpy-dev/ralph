# Self-review report: org-ledger-leftovers

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-org-ledger-leftovers.md`(PR #296 の穴と tech-debt の 388・390・391・395・420 行、S1〜S5、D1〜D7、AC1〜AC8)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1、Run 1)
- Scope: `git diff 31812a3b...HEAD`(base 31812a3b、HEAD 837f4a4f)。コードは S1 3b54f1b1(移すだけ)、S2 47a4fda1 と f972c80b(`org.ShellWord`)、S3 a326e050(`--state-dir` を他の動詞の案内へ)、S4 94c07d76(help・`recordGone`・コメント)、S5 cf69cab5(テストだけ)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- ID: 所見は L1〜L11。CRITICAL、HIGH、MEDIUM はなく、LOW が 11 件。

## Evidence reviewed

- S1 は移すだけかを、3b54f1b1 で消えた行と足された行を `sort` して `comm` で突き合わせて確かめた(対象は `cleanup.go`、`cleanup_text.go`、`org_cleanup_test.go`、`org_cleanup_steps_test.go`)。旧にだけある行は 0。新にだけある行は `package`、`import`、括弧、`import` の語、新ファイル冒頭のコメント 2 行だけ。S1 の移した行は、あとのコミットが変えた所(S2 以降の diff)だけ読んだ。行数は `cleanup.go` 792、`cleanup_text.go` 359、`org_cleanup_test.go` 729、`org_cleanup_steps_test.go` 598 で、4 つとも 800 以下。
- S2 から S4 の本番コードは `git diff 3b54f1b1..HEAD` で全部読んだ(`cleanup.go`、`cleanup_text.go`、`director_stop.go`、`merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go`、`regate.go`、`regate_ci.go`、CLI の `org.go`、`org_cleanup.go`、`org_merge.go`、`org_legacy_ledger.go`)。
- `VerbCommand` / `verbCommand` / `orgVerbHint` の本番の呼び出しを grep し、org の層 15 文と CLI の 3 文が全部、`MergeParams` か `CleanupParams` の `HintStateDir` から値を受けることを追った。`merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go`、`cleanup.go`、`cleanup_text.go`、`org_merge.go`、`org_cleanup.go` の文字列に、ヘルパーを通らない `ralph org merge-order / inbox / status / disband` は残っていない。`prchecks.go`、`mergeorigin.go`、`mergeorder.go`、`mergeguarantee.go`、`regate_git.go` にも、その動詞を名指す文字列はない。
- escape は 15 文それぞれの外側を追った。`cleanup.go:191`(`checkCleanupParams` の拒否)、`merge.go:238-239`(自前で包む)、`merge.go:250,252`(`checkMerge` が `refused` 全体を包む。`mergeenforce.go` の Detail もここを通る)、`merge_run.go:525`(`failed`)と `:570-571`(自前)、`merge_wait.go:144`。
- `registrationAtRisk` の新しいコメントの「git は HEAD を読めない registration の HEAD を 0 で出す」を、git 2.49.0 で打って確かめた。worktree の `HEAD` ファイルを、無い branch への ref にした場合は `HEAD 000…`、`branch refs/heads/does-not-exist`。壊れた文字列にした場合は `HEAD 000…` だけで、`branch` も `detached` も出ない。`regBranch == ""` と `!regDetached` の組が 0 の HEAD になる、という説明と合う。
- 新しいテスト `merge_hint_state_dir_test.go`、`shell_word_test.go`、`org_shell_word_test.go`、`org_hint_state_dir_test.go`、`cleanup_run_test.go`、`abs_path_getwd_test.go`、`org_retired_state_dir_env_abs_test.go`(追記分)、`org_retired_state_dir_env_local_path_test.go` を全部読んだ。既存のテストの書き換え(`org_merge_test.go`、`org_cleanup_test.go`、`org_cleanup_steps_test.go`、`cleanup_test.go`、`cleanup_retry_state_dir_test.go` ほか)は diff で読んだ。
- diff 全体に U+FFFD は 0 件。TODO、debug の出力もない。
- `docs/tech-debt/README.md` の 388・390・391・395・420 行は、plan が /sync-docs の担当と記録している(AC8)。388 の行は今も `cleanupShellWord` と `shellQuoteIfNeeded` を名指す。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: CLI の層に `--state-dir` を足す規則が 2 つある。`cleanupRetryCommand` は `ralph org cleanup` と `--branch`、`--state-dir`(`PrintableInboxText` のあと `ShellWord`)を手で組み、10 行下の新しい `orgVerbHint` は同じ規則を `org.VerbCommand` の上に組む。org の層は `orgRetryCommand` を `VerbCommand` の上に組み直したが、CLI の層は `cleanupRetryCommand` を `orgVerbHint` の上に組み直していない。`orgVerbHint` の doc が「`cleanupRetryCommand` の規則」と言うとおり、2 つは同じ規則の写し。層をまたぐ重複は無い(`orgVerbHint` は `VerbCommand` に escape を足しただけ)。 | `internal/cli/org_cleanup.go:217-227`(`cleanupRetryCommand`)、`:234-236`(`orgVerbHint`)、`internal/org/cleanup_text.go:39-41`(`orgRetryCommand` は `VerbCommand` 経由)。`--state-dir` を手で足す CLI の箇所は `internal/cli/org.go:621-623`(`orgReadCommandHint`、escape なし)にもある。 | `cleanupRetryCommand` を `orgVerbHint("cleanup"+branchArg, hintStateDir)` にする。`branchArg` は `" --branch "+org.ShellWord(org.PrintableInboxText(branch, false))`、branch が空なら空。出力は変わらず、既存の `cleanupRetryCommand` のテストがそのまま効く。 |
| LOW | maintainability | L2: 2 つの共有ヘルパーを置いた場所と、`ShellWord` の doc が合っていない。(a) `ShellWord` と `VerbCommand` は cleanup 専用の `cleanup_text.go`(先頭のコメントは「cleanup の拒否の文と、それが名指すコマンド」)にあるのに、使うのは merge 系 4 ファイル、`regate.go`、`regate_ci.go`、`director_stop.go`、CLI。いつも quote する `shellQuote` は `feature.go` にあり、3 つが別々の場所にある。(b) `ShellWord` の doc は「internal/org と internal/cli が、印字するコマンドの中で値を 1 語にするのに使う 1 つの関数」と言うが、`shellQuote` を直接呼ぶ所が残る。`InboxCommand` は `ralph org inbox show e1 --state-dir '/state/org'` と必ず quote し、merge の新しい案内は `ralph org inbox --state-dir /state/org` と quote しない。同じ台帳が動詞によって 2 通りに出る。plan は `shellQuote` を 388 の範囲の外としているが、doc がそれを言っていない。 | (a) `internal/org/cleanup_text.go:15,53,354`、`feature.go:521`。(b) `cleanup_text.go:349-353` の doc、`escalate.go:123`、`approve.go:388,415`、`watchgate.go:82`。 | (a) `shellQuote` の隣(例: `shellword.go`)に 3 つをまとめる。(b) doc を「要るときだけ quote する関数のうち、internal/org と internal/cli が使う 1 つ。いつも quote する `shellQuote` は escalate・approve・watchgate の文が直接使う」に直す。388 の行を閉じるときも、`shellQuote` が残ることを書く。 |
| LOW | security | L3: `VerbCommand` と `mergeCheck.verbCommand` は escape しない。安全は、15 の呼び出しの外側にある `PrintableInboxText` に頼っていて、`VerbCommand` の doc はそれを言わない。CLI の `orgVerbHint` は自分で escape する(CLI の文は包まれていないため)ので、同じ系統の関数が 2 つの規則を持つ。今回追った 15 か所はどれも包まれているが、次に足す呼び出しが包まれた文の外に出ても、型も doc も止めない。あわせて、`merge_wait.go` は `VerbCommand("status --org-id "+r.orgID, …)` と org_id を `ShellWord` に通さず `verbAndArgs` に連結する。org_id は識別子の検証を通る値で、裸のまま出る文字だけなので今は同じ出力だが、doc の「dir は 1 語」が言うのは dir だけ。 | `internal/org/cleanup_text.go:43-58`(`VerbCommand` の doc に escape の記述なし)、`internal/cli/org_cleanup.go:234-236`、`internal/org/merge_wait.go:144-146`。外側の `PrintableInboxText` は Evidence reviewed に列挙した。 | `VerbCommand` の doc に「escape しない。呼び出し側が文全体を `PrintableInboxText` に通す(CLI は `orgVerbHint`)」を 1 文足す。org_id は `ShellWord(r.orgID)` にすれば、検証を通る値の出力は変わらず、doc の約束と合う。 |
| LOW | maintainability | L4: `--branch` が無いときの案内を作る `org_merge.go` の 8 行が、`orgInboxHintStateDir` の規則(`--state-dir` を渡したときだけ値を出す)を自分で繰り返す。理由は正しく、台帳を解決する前に断る(テスト `TestOrgMerge_FlagRefusalsReadNothing` が読まないことを固定する)ので、`orgInboxHintStateDir(cmd, resolvedStateDir)` は使えない。ただし規則が `cmd.Flags().Changed("state-dir")` の 1 行として 2 か所に分かれ、`hintStateDir, _, _ = org.ResolveOrgStateDir(*stateDir, true)` の捨てた error と source が安全な理由(`explicitSet` の戻りでは error は常に nil)はコメントに無い。 | `internal/cli/org_merge.go:226-233`、`internal/cli/org_inbox.go:353-358`(`orgInboxHintStateDir`)、`internal/org/statedir.go:73-77`(「err は tier 2 だけ」)。 | `Changed` の判定と `ResolveOrgStateDir(…, true)` を `orgFlagStateDirHint(cmd, stateDir)` のような 1 つの関数にして、`orgInboxHintStateDir` と `org_merge.go` が呼ぶ。作らないなら、`org_merge.go` のコメントに「explicitSet のとき error は nil」と 1 句足す。 |
| LOW | readability | L5: doc の言い回しが 4 か所で読みにくい。(a) `printMergeResult` の doc は、「exit 0 は記録された merge だけ」の文のあとに `; it names ralph org merge-order with hintStateDir` を `;` でつなぎ、`it` が結果か error か決まらない。(b) `mergeCheck.hintStateDir` の「which again and verbCommand repeat」は、動詞が欠けた文に読める(`again()` と `verbCommand()` がこの値を繰り返す、の意味)。(c) `registrationAtRisk` の新しいコメントは「one of the last kind」「For both of the last two」と、位置で種類を指す。文を 1 つ足すと指す先が変わる。(d) `newOrgMergeCmd` の doc の「the commands its refusals and the cleanup's name, to run again or of another verb such as ralph org merge-order」は、1 文に主語が 3 つ入る。 | (a) `internal/cli/org_merge.go` の `printMergeResult` の doc(`:333-337`)。(b) `internal/org/merge.go:166-167`。(c) `internal/org/cleanup_text.go:298-309`。(d) `internal/cli/org_merge.go:36-42`。 | (a) 文を分け、「その error は `ralph org merge-order` を `hintStateDir` 付きで名指す」と主語を書く。(b) 「`again` と `verbCommand` がこの値を各コマンドに付ける」。(c) 「HEAD が commit を指さない場合(0 の HEAD)」のように種類を名前で言う。(d) 「やり直すコマンドと、merge-order のような別の動詞のコマンドが、渡された `--state-dir` を持つ」と 2 つに分ける。 |
| LOW | maintainability | L6: 390 の分割後の `cleanup.go` は 792 行で、余裕が 8 行しかない。plan 自身が「S3 以降は文の組み立てを `cleanup_text.go` に置く」と記録しているが、次に cleanup の文を 1 つ足すと 800 行を超え、390 の trigger(「次に cleanup.go を変えるとき」)がまた立つ。`checkCleanupParams` のように文を作る関数が `cleanup.go` に残っている。 | `wc -l internal/org/cleanup.go` = 792。plan Progress の S3(「cleanup.go は 792 行」)。`internal/org/cleanup.go:258` に `checkCleanupParams`。 | 390 の行を閉じるとき、余裕が 8 行であることと、新しい文は `cleanup_text.go` に置く約束を書く。もう 1 つ文を作る関数を移して余裕を増やす手もある。 |
| LOW | readability | L7: テストの `hintFlag(hint)` は引数の中身を見ない。空でなければ必ず `" --state-dir '/my ledgers/org'"` を返す。この定数が `cleanupRetryHint`(`cleanup_retry_state_dir_test.go:24`)の値と同じなのは偶然に近い結びつきで、`hintStateDirRuns` に 3 つ目の値を足すと期待が黙って間違う。名前は「hint を flag にする」と読める。 | `internal/org/merge_hint_state_dir_test.go:20-25`(`hintFlag`)、`:54`(`hintStateDirRuns`)。 | `" --state-dir " + ShellWord(hint)` にする(`ShellWord` は `shell_word_test.go` の表で別に固定している)。定数のままにするなら `hintStateDirRuns` の隣に置いて結びつきを見せる。 |
| LOW | naming | L8: `internal/org/cleanup_test.go` の `TestCleanupShellWord` は、消えた名前(`cleanupShellWord`)のまま `ShellWord` を試す。新しい `TestShellWord_QuotesOnlyWhatAShellReads`(`shell_word_test.go`)も同じ関数を試し、2 つの表は行が重ならない(`=main`、`feat/a~1`、`/My Repos/ralph` は古い方、`+ : @ ,` と空文字列は新しい方)。`rg 'TestShellWord'` では新しい 1 本しか出ず、古い方は `rg cleanupShellWord` で探しても出ない。 | `internal/org/cleanup_test.go:1800-1817`、`internal/org/shell_word_test.go:11-28`。 | 古い方の関数名を 1 行だけ `TestShellWord_QuotesWhatAShellReadsInAPath` のように直す(D6 が許す 1 行の編集)。行を新しい表に寄せる直しは、386 の 4 ファイルから行を消すことになるので、しない。 |
| LOW | readability | L9: `TestOrgMerge_NotMergedExitsOneWithNothingOnStdout` は、表の期待文字列を、ループの中で `strings.ReplaceAll(want, "ralph org merge-order", "ralph org merge-order --state-dir "+…)` と書き換える。表の文は出力と一致しなくなり、失敗したときに表の文字列で出力を grep できない。置換は `merge-order --all` の行があると `--all` の前に flag を入れる。今の 3 行は `merge-order again` だけなので壊れていない。同じ PR の他のテストは、完全な文字列か `%STATEWORD%` の置き場所で書く。 | `internal/cli/org_merge_test.go:559-561`。同じファイルの `:786` が `strings.NewReplacer("%STATE%", …, "%STATEWORD%", …)` の置き場所を使う。 | 期待の文字列に `%STATEWORD%` を書き、ループで `NewReplacer` を使う。 |
| LOW | naming | L10: `checkCleanupHeadNotHere` の `work := org.ShellWord(fx.work)` は、quote 済みの語なのに `work` と名づけられ、同じ `want` の式の中で生の `fx.work` と並ぶ。「which git does not have in " + fx.work」は生の path、「git -C " + work」は quote 済みの語で、どちらかを取り違えると空白のないケースでは通り、空白のあるケースでだけ落ちる。 | `internal/cli/org_cleanup_test.go:437`、`:439-445` の `want`。 | `workWord` に改名する。 |
| LOW | maintainability | L11: `TestOrgCleanup_HintStateDirMergeOrderReplaysOnTheLedgerNamedFirst` の骨組みは、`TestOrgCleanup_RetryCommandReplaysOnTheLedgerNamedFirst` の写し。`exec.LookPath("sh")`、fixture、`moveLedgerAway`、個人用ファイルの subtest(`t.Setenv(config.LocalConfigEnv, "")`、`writeTestFile`、`defer os.Remove`)、既定の台帳が変わらない確かめが同じ形で 2 回出る。3 本目の replay テストが 4 回目の写しになる。 | `internal/cli/org_hint_state_dir_test.go:122-181`、`internal/cli/org_retry_state_dir_test.go:200-254`。 | 個人用ファイルを書いて消す部分を `withPersonalLedgerFile(t, fx, func())` のような helper に出す。 |

### 依頼された点の判断

1. org の層と CLI の層の重複: org の層は `orgRetryCommand` が `VerbCommand` の上にあり、規則は 1 か所。CLI の層は `orgVerbHint` が `VerbCommand` の上にあるのに、`cleanupRetryCommand` だけが上に乗っていない(L1)。層をまたぐ重複はない。`orgVerbHint` は `VerbCommand` と escape の組で、薄い包みとして妥当。
2. `hintStateDir` の受け渡し: 引数で渡すのは `recordMerge`(`merge_run.go:430` が `run.m.hintStateDir` を渡す)、`printMergeResult`、`cleanupNotInLedger`(呼び出し 2 か所とも手元の値)、`enforce` と `readEnforcementRepository`。フィールドで持つのは `mergeCheck`、`mergeWait`、`MergeEnforcementParams`で、設定する場所は `newMergeCheck(p)`、`newMergeWait(p)`、`merge.go:330` の 3 つ。`enforce` が引数なのは、`mergeGuaranteeRun` を doctor の `CheckMergeGuarantee` と共有していて、フィールドを足すと doctor 側で使われない値が載るため(`mergeguarantee.go:199`)。妥当。値は 15 の org の文すべてに届いていて、取りこぼしは見つからなかった。コピーが 4 つ(`MergeParams`、`mergeCheck`、`mergeWait`、`MergeEnforcementParams`)に増えたのは、層を分けた作りの代価で、直すほどではない。
3. 新しいコマンドの語の escape: org の層は文の外側で包む(L3 に条件を書いた)。CLI の層は `orgVerbHint` が dir を escape してから `ShellWord` に渡し、`cleanupRetryCommand` と同じ順(escape して quote)。制御文字を含む dir は印字と実際の path が違うが、これは `cleanupRetryCommand` が前から持つ性質で、今回の変更ではない。`merge.go:238` の「branch が無い」拒否は、前は escape されなかった文が `PrintableInboxText` で包まれた(改善)。
4. help の読みやすさ: 理由のコマンドが 2 通りあることを、branch の側の理由、PR や origin を読めなかった理由、`--state-dir` の 3 文で順に書いていて、前の 1 文より正確で読める。`--branch\n<branch>` が 70 桁の折り返しで割れるのは、他の行も同じ。直しは要らない。
5. 新しいテストが言うことを打っているか: 打っている。`assertHintStateDir` は `--state-dir` の出現数まで数えるので、足りない文も余計な文も落ちる。各文の組を hint あり・なしの 2 回で打つ。2 台帳の replay は、既定の台帳に `feat/a` が無いことを先に確かめるので、hint が `--state-dir` を落とすと落ちる形。HeadNotHere の空白のケースは、`fx.work` と `fx.wt` に空白があることを `Fatalf` で確かめてから進むので、空振りしない。Linux だけのテスト 2 本は、`absPath` と `retiredOrgStateDirEnvAbs` の元の値を package の初期化で退避し、他のテストの差し替えが残っても影響を受けない。darwin では `t.Skip` になるので、この review では動作を見ていない(CI の ubuntu-latest が証跡)。

## Positive notes

- S1 は純粋な移動。旧にだけある行は 0、新にだけある行は `package`、`import`、括弧、冒頭コメントだけ。
- 要るときだけ quote する関数が 1 つに減った。CLI の `shellQuoteIfNeeded` と `shellSafeUnquoted`、`org_merge_test.go` の `orgLayerShellWord` と `orgPlainWord`、plan が範囲外で見つけた `directorShellWord` が消えた。`directorShellWord` の置き換えには、記録された id が `director.jsonl` の手編集で何にでもなりうる、という理由のコメントを残してある。
- org の層の 15 文は、どれも外側で escape される。`checkMerge` が `refused` の全体を包み、`mergeenforce.go` の Detail もその経路で escape される。
- 案内の文が 1 か所の関数を通る形になり、15 + 3 文に対して「`--state-dir` を足す」判断は `VerbCommand` の 1 行(`if hintStateDir != ""`)で決まる。
- `registrationAtRisk` の新しいコメントの根拠(git が壊れた HEAD の registration を 0 の HEAD で出す)は、git 2.49.0 で再現できた。
- `merge_hint_state_dir_test.go` は、文ごとに「期待の文」と「`--state-dir` の出現数」を結び、hint の有無で落ちる変異を捕まえる。
- 秘密、debug の残り、範囲外のファイル、U+FFFD は見つからなかった。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(L1〜L11 は PR の中で直せる LOW) | - | - | - | - |

_(`docs/tech-debt/README.md` の 388・390・391・395・420 行の書き換えは、plan が /sync-docs の担当と記録済み。388 の行を閉じるときは L2 の `shellQuote` の残りを、390 の行を閉じるときは L6 の余裕 8 行を書く。)_

## Coverage gaps

- Go のテスト、`go vet`、静的解析は打っていない(/verify と /test の担当)。S2 で消した `regexp` の import が他で使われていないことは、各スライスの verify が rc 0 だったという plan の記録を信頼した。
- Linux だけのテスト 2 本は darwin で SKIP のため、この review では実行結果を見ていない。plan の S5 には linux/arm64 の docker での PASS が記録されている。CI の amd64 は PR の CI で確かめる。
- `registrationAtRisk` のコメントを確かめた git は 2.49.0 だけ。
- `TestOrgMerge_NotMergedExitsOneWithNothingOnStdout` など、L9 の置換が壊す条件(`merge-order --all` の行)は、今の表にその行が無いので実行して確かめていない。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 11。どれも merge を止めない)
- Follow-ups:
  - L1、L3、L4、L5 は `cleanup_text.go`、`org_cleanup.go`、`org_merge.go`、`merge.go` の小さな直し。出力の文は変わらない。直すなら /verify の前に 1 コミットで直せる。
  - L2 の (a) は移すだけ、(b) は doc の 1 文。L6 は 390 の行の書き方の話で、/sync-docs で足せる。
  - L7〜L11 はテストの書き方で任意。L8 は `cleanup_test.go`(386 行のファイル)の 1 行の改名だけにする(D6)。
