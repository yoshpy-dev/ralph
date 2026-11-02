# Verify report: org-ledger-leftovers

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-org-ledger-leftovers.md`(PR #296 の穴と tech-debt 388・390・391・395・420 行、AC1〜AC8、D1〜D7、Progress の S1〜S5、S2 の補い f972c80b、self-review の直し F1 bb0afab1)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main c8c8c0f1、HEAD 92a01e23、44 ファイル、+2303/−1154)。e208a2c6 で origin/main を取り込んでいるので、branch の変更は三点の diff で見た。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。打ったのは静的なものだけで、run-static-verify、check-sync、linux/amd64 の `go vet`、S1 の宣言ごとの比較、HEAD の binary の help と `ralph org merge` の拒否の probe の 5 つ。
- Evidence: `docs/evidence/verify-2026-11-02-org-ledger-leftovers.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。run-static-verify と check-sync の全出力、linux の vet、S1 の比較、grep の結果、binary の probe、2 つの help、tech-debt の Trigger の照合)。run-verify 自身のログは `docs/evidence/verify-2026-11-02-040351.log`(同じく対象外)。
- 計画の digest: `./scripts/plan-visual.sh digest` は `78557bfbc265` で、`- Approved:` の行と同じ。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(390): `cleanup.go` は 800 行以下で文の関数は `cleanup_text.go` に、`org_cleanup_test.go` は 2 つに分かれてどちらも 800 行以下。S1 は移すだけで、テストの名前は前後で同じ | Met | HEAD の `wc -l` は `internal/org/cleanup.go` 792、`cleanup_text.go` 325、`shellword.go` 58(F1 で `ShellWord`・`shellPlainWord`・`VerbCommand`・`shellQuote` を移した)、`internal/cli/org_cleanup_test.go` 729、`org_cleanup_steps_test.go` 598。S1(3b54f1b1)は移すだけ: Go の構文木で top-level の宣言を doc コメントごと切り出し、3b54f1b1^ と 3b54f1b1 で byte 単位に比べた(scratchpad に置いた比較用の小さなプログラム)。internal/org は 43 の宣言がすべて同じで、そのうち 18 が `cleanup_text.go` へ移った。internal/cli は 32 の宣言がすべて同じで、10 が `org_cleanup_steps_test.go` へ移った。どちらも、違う宣言、消えた宣言、増えた宣言、重なった宣言は 0。テストの名前は `func Test`・`Benchmark`・`Fuzz`・`Example` を git grep で数え(サブパッケージ込み)、org 1515、cli 980 で前後同じ。`go test -list` は /test の担当なので打っていない(Progress の記録は org 1424、cli 979 で前後同じ)。 |
| AC2(388、420(b)): 語を quote する関数は `org.ShellWord` の 1 つで、`shellQuoteIfNeeded` と `shellSafeUnquoted` はない。6 つの値で CLI の層と org の層が同じ形を書くことを表のテストが確かめる | Met(コードとテストの存在で確認。テストの結果は /test) | 消えた名前(`shellQuoteIfNeeded`、`shellSafeUnquoted`、`cleanupShellWord`、`directorShellWord`、`cleanupPlainWord`、`orgLayerShellWord`、`orgPlainWord`)は、Go のコード、README、AGENTS.md、CLAUDE.md、docs/specs、.claude/rules、.claude/skills、templates に 0 件。要るときだけ quote する関数は `org.ShellWord`(`internal/org/shellword.go:31`)だけ。ほかに single quote の escape を書くのは、いつも quote する `shellQuote`(`shellword.go:19`)だけで、計画の Progress(S2 の補い)と `ShellWord` の doc(`shellword.go:28-30`)が残すと書いている。`prompts.go` の `agmsgScriptsWord` は `shellQuote` を呼ぶだけで、`config.go` の `reviewerWordPattern` は設定値の検証なので quote しない。表のテストは `internal/org/shell_word_test.go`(`TestShellWord_QuotesOnlyWhatAShellReads` の 10 行と、`orgRetryCommand` を cleanup と merge で打つ `TestShellWord_OrgRetryCommandWritesTheBranchAndTheStateDirWithIt`)と `internal/cli/org_shell_word_test.go`(`cleanupRetryCommand` と `orgReadCommandHint`)。どちらも feat/a+b、a:b、a@b、a,b、空白、single quote の 6 つの値に同じ期待の語を書くので、2 つの層が同じ形を出すことを固定する。 |
| AC3(395): `--state-dir` を付けて打った merge と cleanup が出す、merge-order・inbox・status・disband を案内する文は、どれも `--state-dir <dir>` を含む。付けないときは、個人用のファイルで台帳を決めたときも含めて含まない。文の組ごとにテストがあり、2 つの台帳で打ち直すテストがある | Met(コードとテストの存在、HEAD の binary の probe で確認。テストの結果は /test) | 数え直した 18 文は Progress の数と一致する。org の層 15 文: `cleanup.go:265`、`merge.go` の 239・277・289・528(inbox)・564・568・733、`mergeenforce.go:186`、`merge_run.go` の 419・439・442・519・571、`merge_wait.go:145-146`(status と disband)。CLI の層 3 文: `org_merge.go:238`、`:356`、`org_cleanup.go:357`。どれも `VerbCommand`(`shellword.go:52`)を通り、`merge.go:359` の `verbCommand` と CLI の `orgVerbHint`(`org_cleanup.go:231`)もその上に乗る。値の出どころは `MergeParams.HintStateDir` と `CleanupParams.HintStateDir` で、CLI は `orgInboxHintStateDir`(`org_inbox.go:353-358`、`--state-dir` を渡したときだけ値を返す)から設定する(`org_merge.go:265`、`org_cleanup.go:159`)。`--branch` がないときの merge の拒否は台帳を解く前に出るので、同じ規則を `cmd.Flags().Changed("state-dir")` で繰り返す(`org_merge.go:226-236`)。個人用のファイルで決めた台帳では値が空になり、`--state-dir` は付かない。文の組ごとのテストは、`internal/org/merge_hint_state_dir_test.go` の 5 本(`TestCleanupFeatureWorktree_HintStateDirReachesMergeOrderWithNoBranch`、`TestCheckMerge_HintStateDirReachesTheOtherVerbs` の 7 ケース、`TestCheckMergeEnforcement_HintStateDirReachesMergeOrder`、`TestMerge_HintStateDirReachesMergeOrderOnceGHPRMergeRan` の 5 ケース、`TestMerge_HintStateDirReachesStatusAndDisbandOnceTheWaitIsUsedUp`)と、`internal/cli/org_hint_state_dir_test.go` の 3 本で、18 文すべてに当たる。打ち直すテストは `TestOrgCleanup_HintStateDirMergeOrderReplaysOnTheLedgerNamedFirst`(`org_hint_state_dir_test.go:120`)。出力から取り出したコマンドを sh で語に分けて打ち、最初に選んだ台帳の一覧と同じになること、既定の台帳には feat/a がないこと、個人用のファイルの subtest では素のコマンドが同じ台帳を読むことを見る。HEAD の binary で `--branch` なしの `ralph org merge` を打つと、空白を含む `--state-dir` は quote され、`+ : @` だけの値は裸のまま、相対パスは絶対パスになり、`--state-dir` なしでは付かない(4 つとも rc 1、台帳は作られない)。対象の外に残した言及は下の「AC3 の対象の外」にまとめた。 |
| AC4(420(a)): cleanup の help は、SKIPPED の理由のコマンドが 2 通りあることを言い、help を固定したテストがその文を見る。`--branch` なしの 2 つの場合のテストがある | Met | help(`org_cleanup.go:111-117`)は、branch の側の理由は `ralph org cleanup --branch <branch>`(`--branch` なしで打っても)、PR や origin を読めなかった理由は打ったとおりの形、どちらも `--state-dir` は渡したときだけ、と書く。HEAD の binary の `ralph org cleanup --help` にもその文が出る。コードと合う: branch の側の理由は `cleanupAgain(p)`(`cleanup_text.go:17-19`)が作り、`p.Branch` は step ごとの branch(`org_cleanup.go:407`)。PR や origin の理由は `cleanupRetryCommand(branch, …)` が打った `--branch` で作る。固定するテストは `TestOrgCleanup_Help`(`org_cleanup_steps_test.go:497-500`)。`--branch` なしの 2 つの場合は既存のテストが受け持つ。`TestOrgCleanup_RefusalsAreSkippedLines`(`org_cleanup_test.go:379` で flag なしに打ち、`:353`・`:360`・`:369` で `cleanup --branch feat/a --state-dir …` を期待)と、`TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`(`:248-251`、`--branch` なしの `ralph org cleanup --state-dir … again`)と `TestOrgCleanup_WithoutGHRemovesNothing`(`:186-189`)。 |
| AC5(391): 記録がないときの拒否は「remove what is left by hand」を含み、`registrationAtRisk` のコメントは detached HEAD だけを名指す | Met | `recordGone` は `cleanup_text.go:150` で `remove what is left by hand`。コメント(`cleanup_text.go:266-286`)は「Only a detached HEAD is kept on a new branch (keptBranch)」(`:275`)で、コードと合う。0 の HEAD は reflog の案内(`:308`)、別の branch の HEAD も reflog の案内(`:311`)、残る detached HEAD だけが `keptBranch`(`:315`)に行く。`cleanup.go:534` の「remove it by hand」は `st.notDir` の拒否の文で、指す物は worktree の場所にある物なので AC5 の対象ではない。 |
| AC6(420(c)): HeadNotHere の期待はコードと同じ quote で作られ、空白を含む作業ディレクトリのケースがある | Met(テストの存在で確認。結果は /test) | `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead`(`org_cleanup_test.go:406-425`)は 2 つの subtest に分かれる。空白のケースは GOTMPDIR と TMPDIR を空白入りのディレクトリに向け、`fx.work` と `fx.wt` に空白がなければ `Fatalf` で止まる。期待の `git -C` の語は `org.ShellWord` で作る(`:437-445`)。 |
| AC7(420(d)): `st.notDir`、`git worktree remove` の失敗、`LedgerLocalConfigPath` の `ok == false` のテストがある。Linux だけのテストが CI で通る | Partially verified(テストの存在と linux 向けのコンパイルまで。CI の結果は未確認) | `internal/org/cleanup_run_test.go:27`(ディレクトリでない物)、`:83`(`git worktree lock` で remove を失敗させる)、`internal/cli/org_retired_state_dir_env_local_path_test.go:27`(`:41-42` で `ok == false` の前提を確かめてから打つ)。Linux だけのテストは `internal/org/abs_path_getwd_test.go:24` と `internal/cli/org_retired_state_dir_env_abs_test.go:105` で、`runtime.GOOS != "linux"` なら skip し、本物の関数を package の初期化で退避して呼ぶ。`GOOS=linux GOARCH=amd64 go vet ./internal/org/ ./internal/cli/` は rc 0 なので、CI の環境でコンパイルは通る。ubuntu-latest での結果は PR の CI で見る(Progress には docker の linux/arm64 で 2 本とも PASS と記録)。tech-debt 420(d) の「本物の binary は main checkout の root からだけ打った」の項は、計画の Scope の 420(d) に入っていない(下の /sync-docs への申し送り)。 |
| AC8: tech-debt の 388・390・391・395・420 行の書き換え、`run-verify.sh` と `run-test.sh` が通る | 未着手(書き換えは次の段の /sync-docs、テストは /test) | `docs/tech-debt/README.md` はこの diff で変わっていない。静的な側(`run-static-verify.sh`)はこのステップで通った。 |

### AC3 の対象の外

計画の Scope と Progress の S3 が対象の外とした言及を、grep で確かめた。

- help と flag の説明: `org_merge.go` の 49〜70・167・186・276 行、`org_cleanup.go` の 57・61・191 行。打つコマンドの案内ではない。
- `org_cleanup.go:180-181`: merge-order の読み方の説明(「as ralph org merge-order --all reads them」)。打つコマンドではない。
- `merge.go:420-428` の `mergeLsRemoteRefusal`: merge-order のエラー文の接頭辞を merge のものに換え、「ralph org merge reads origin」と言い換える。コマンドは名指さない。
- `ledgerMoveStep`(`cleanup_text.go:75`): 台帳の新しい場所を `--state-dir <新しい場所>` で名指す。古い値を繰り返さないのが正しい(PR #296 からの例外)。
- merge の「ready でない」拒否(`merge.go:288-289`)が付ける merge-order の Detail: `ralph org regate --branch <branch>`(`mergeRegateCommand`、`mergeorder.go:1113-1115`)と `ralph org gate --org-id <org>`(`mergeGateCommand`、`:1171-1179`)。gate と regate は計画の 4 つの動詞の外。V4 に書いた。
- コメント。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | Requested scope は changed、Language scope は changed(changed_languages)で、golang pack を選んだ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さない(通ったときの形)。`Skipping` の行はない。 |
| shellcheck(hook と verify の script)、`sh -n` の hook、Codex の inline hook の検出、PR provenance の guard | pass | |
| `scripts/check-sync.sh`(run-static-verify の中と単独の 2 回) | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。 |
| `check-pipeline-sync.sh` / `check-skill-sync.sh`(14 skill)/ `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | c8c8c0f1..92a01e23 は clean。 |
| `GOOS=linux GOARCH=amd64 go vet ./internal/org/ ./internal/cli/`(追加) | pass(rc 0) | Linux だけで走る 2 本のテストが CI の環境でコンパイルされることを見た。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org cleanup --help` | Yes | AC4 のとおり。HEAD の binary で確かめた。 |
| `ralph org merge --help` | Yes | 「run ralph org cleanup --branch <branch> again; the reason names that command with the --state-dir merge was given, if any」は、merge が cleanup に必ず branch を渡すので正しい。ほかの動詞のコマンドが `--state-dir` を持つことは書いていない(I1)。 |
| `README.md:288`(merge)と `:290`(cleanup) | Yes(要約の粒度) | 食い違う文はない。cleanup の段落の「most often … run `ralph org cleanup --branch <branch>` again」は help の 2 通りの書き分けと矛盾しない。どちらの段落も、出すコマンドが `--state-dir` を繰り返すことは書かず、細部は `--help` に任せている(I1)。README はこの diff で変わっていない。 |
| `AGENTS.md` の地図の `internal/org/` の行 | No(V3) | `cleanup_text.go` と `shellword.go` を名指していない。 |
| `docs/specs/`、`.claude/rules/`、`.claude/skills/` | Yes | 変えた関数も出力の文も名指していない(grep)。 |
| コードのコメント | No(V2) | 移した関数を `cleanup.go` にあると書くコメントが 2 つ残る。 |
| 消えた名前を書く文書 | 一部(V1) | 計画、古い evidence(`docs/evidence/test-2026-10-15-org-director-start.md`)、tech-debt の 197・388・390・403・420 行だけ。388・390・420 は AC8、197 と 403 は V1。 |
| `docs/tech-debt/README.md` 388・390・391・395・420 行 | 未着手 | AC8 で /sync-docs が書き換える。行ごとの中身は Verdict の申し送りに書いた。 |
| `docs/tech-debt/README.md` 386 行 | 未着手 | D6 のとおり、繰り越しの注を /sync-docs が書く。この branch の変更は `internal/cli/org_merge_test.go`(1207 → 1195 行)、`internal/org/merge_run_test.go`(1111 行のまま、6 行を書き換え)、`internal/org/cleanup_test.go`(1817 行のまま、32 行を書き換え)で、`director_skill_sync_test.go` は変わっていない。数えたのは origin/main と HEAD。 |
| `docs/tech-debt/README.md` 170・197・290・313・392・401・403 行 | No(V1) | 下の V1。Trigger の列を python で読み、取り消し線の中を除いて照合した。 |
| 計画の Acceptance criteria の checkbox | 遅れ(INFO、I2) | AC1〜AC8 はすべて `[ ]` のまま。 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 計画が挙げる 388・390・391・395・420(と D6 の 386)のほかに、この diff が Trigger に当たる、または名前や場所が古くなった tech-debt の行が 7 つある。170: Trigger「`guardLegacyOrgStateDir` の次の変更」に当たる(3 つの文の quote の関数を `org.ShellWord` に替えた。`: @ + ,` を含むパスの出力の形が変わる)。197(l): Trigger「`shellQuote` か `shellQuoteIfNeeded` の次の変更」に当たる。`shellQuoteIfNeeded` が消え、`ShellWord` が `shellQuote` を呼ぶので、「`shellQuote` が `shellQuoteIfNeeded` の quote の枝の写し」という所見は解消した。`mustAbs` の半分は残る。本文の「`shellQuote` in `feature.go`」は `shellword.go` に変わった。290: Trigger「`regate_test.go` の次の変更」に当たる(コメントの名前を 1 行)。313: 本文の「`shellQuote` in `internal/org/feature.go`」が古い(今は `shellword.go`)。392: Trigger「呼び出し元か `statedir.go` の次の編集」に当たる(`resolvedOrClean` の呼び出し元の `cleanup.go`、`feature.go` を編集し、`pathWithin` を `cleanup_text.go` に移した)。401: Trigger「`regate.go`、`regate_ci.go` の次の変更」に当たる(名前の置き換え 1 行と 3 行)。403: Trigger「`gateWording` か regate の拒否の文の次の変更」に、`regate_ci.go` の拒否の文の中の呼び出しの改名で当たる。直し方の「`cleanupShellWord` で」は `ShellWord` に変わった。 | evidence の log の「tech-debt Trigger sweep」と diff の stat。self-review は 388・390・391・395・420 だけを挙げた。 | /sync-docs で、行ごとに日付つきで「当たったが払わなかった」とその理由を書く(この表のほかの行と同じ形)。197(l) は前半を閉じ、313 と 403 は名前と場所を直す。 |
| V2 | LOW | S1 で関数を `cleanup_text.go` に移したのに、`cleanup.go` にあると書くコメントが 2 つ残る。`internal/org/followup.go:530` の「cleanupDetails in cleanup.go」と、`internal/org/cleanup_test.go:247` の「registrationAtRisk of cleanup.go」。どちらも `cleanup_text.go`(`:323`、`:287`)にある。 | evidence の log の「stale file references」。 | 1 語ずつ `cleanup_text.go` に直す。`cleanup_test.go` は 386 行のファイルなので、D6 の「1 行だけ」の範囲で直す。コメントだけの変更なので、/sync-docs のコミットに入れてもよい。 |
| V3 | LOW | `AGENTS.md` の地図の `internal/org/` の行は、新しい `cleanup_text.go`(cleanup の拒否の文と、それが名指すコマンド)と `shellword.go`(印字するコマンドの語: `ShellWord`、`shellQuote`、`VerbCommand`)を名指さない。これまでの移すだけの分割は地図に足してきた(80dd2982 が `director_start_cleanup.go` を、fb4d169f が `gate_record.go` と `regate_inway.go` を足した)。 | `AGENTS.md:91`。evidence の log の「AGENTS.md internal/org files not named」。名指さないファイルはほかに 15 あり、地図はすべてを書く形ではない。 | /sync-docs で、cleanup の項(`cleanup.go`)に `cleanup_text.go` を足す。`shellword.go` は internal/org と internal/cli の両方が使うので、足すなら cleanup と merge の項の近くに 1 句。 |
| V4 | LOW | merge の「ready でない」拒否(`merge.go:288-289`)は merge-order の Detail をそのまま付け、Detail は `ralph org regate --branch <branch>`(`mergeRegateCommand`、`mergeorder.go:1113-1115`、branch は `ShellWord` を通らない)や `ralph org gate --org-id <org>`(`mergeGateCommand`)を名指す。`--state-dir` を付けて打っても、この 2 つは付かない。merge-order 自身の出力も同じ。計画の Scope は 4 つの動詞だけで、Progress の S3 も「regate と gate を名指す Detail」を変えなかったと書くので、計画どおり。ただ、395 の行の「Left open」も tech-debt のほかの行も、この残りを書いていない(395 が regate について書くのは regate 自身の拒否の文、`gateWording.again` の行)。 | `merge.go:289`、`mergeorder.go:1113-1179`。tech-debt を `mergeRegateCommand` と `mergeGateCommand` で引くと 0 件。 | /sync-docs で 395 を閉じるとき、残りとして書くか、新しい行を足す。直すなら merge-order の Detail の作り方が変わるので、merge-order の出力とテストも一緒に変わる。 |
| I1 | INFO | merge と cleanup の help と README は、merge-order・inbox・status・disband のコマンドが `--state-dir` を繰り返すことを書いていない。cleanup の help は打ち直しのコマンドについてだけ書く。README は要約で、merge の段落の wait の拒否は `ralph org status --org-id <org>` だけを挙げ、disband のコマンドは書かない(以前から同じ)。食い違いではない。 | `README.md:288,290`、HEAD の binary の 2 つの help(evidence の log)。 | 直さなくてよい。書くなら、merge の help の「the reason names that command with the --state-dir merge was given」の隣に 1 文。 |
| I2 | INFO | 計画の Acceptance criteria の checkbox が全部 `[ ]` のまま。 | 計画の 96-103 行。 | /pr の前に lead が付ける。AC8 は /sync-docs と /test のあと。 |
| I3 | INFO | 同じ台帳が、動詞によって 2 通りの形で出る。merge と cleanup の新しい案内は `VerbCommand`(要るときだけ quote)を、inbox・watch・approve の案内は `InboxCommand` と `CommandWithStateDir`(いつも `shellQuote`)を使う。計画どおりで、self-review L2(b) も挙げている。`VerbCommand` と `CommandWithStateDir` は quote の仕方だけが違う同じ形の関数。 | `shellword.go:52-58`、`watchgate.go:80-85`、`escalate.go:120-126`。 | 388 を閉じるとき、`shellQuote` の残りと一緒に書く(self-review の申し送りと同じ)。 |

## Observational checks

- HEAD の binary を `go build -o <S>/ralph ./cmd/ralph` で作った(`<S>` は scratchpad のディレクトリ)。`ralph org cleanup --help` と `ralph org merge --help` の全文を evidence の log に写した。
- scratchpad に `git init` した空の repo で、`--branch` なしの `ralph org merge` を 4 通りに打った。`--state-dir '<S>/my ledger'` は `ralph org merge-order --state-dir '<S>/my ledger' lists the ledger's branches`、`--state-dir <S>/a+b:c@d` は quote なし、`--state-dir rel/led` は repo からの絶対パス、`--state-dir` なしは `ralph org merge-order lists …`。4 つとも rc 1 で、台帳のディレクトリは作られなかった。
- `ralph org cleanup --org-id x` は台帳を読む前に断る(rc 1)。
- 計画の digest は承認時と同じ `78557bfbc265`。

## Coverage gaps

- Go のテストの結果は見ていない(/test の担当)。表のテスト、18 文のテスト、2 つの台帳の打ち直し、HeadNotHere の空白のケース、AC7 の 3 本が通ることは /test で見る。
- Linux だけで走る 2 本は darwin では skip で、このステップではコンパイルまでしか見ていない。CI(ubuntu-latest、amd64)の結果は PR の CI で見る。
- S1 が移すだけであることは、宣言の byte 比較と名前の数で見た。`go test -list` の比較は Progress の記録による。
- `cleanup` の 18 文のうち org の層の文を binary で出すには、origin が GitHub を名指し gh が PR を読む repo が要るので、打っていない。/test の計画が、build した binary で cleanup の merge-order の案内を写して打つことを evidence にする。
- AC8(tech-debt の書き換え)は /sync-docs、テストは /test の担当。

## Verdict

pass

- Verified: AC1(行数、S1 の宣言ごとの byte 比較、テストの名前の数)、AC2(消えた名前が 0 件、要るときだけ quote する関数が 1 つ、表のテストの中身)、AC3(18 文の数え直し、値の出どころ、文の組ごとのテストと打ち直しのテストの中身、binary の probe)、AC4(help の文とコードの一致、固定するテスト、`--branch` なしの 2 つの場合のテスト)、AC5(文とコメント、コメントとコードの一致)、AC6(テストの中身)。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan、linux/amd64 の go vet)はすべて通った。計画の digest は承認時と同じ。
- Partially verified: AC7(テストの存在とコンパイルまで。CI の結果は未確認)。
- Not verified: Go のテストの結果(/test)、AC8(/sync-docs と /test)。
- 所見: LOW 4 件(V1〜V4)、INFO 3 件(I1〜I3)。どれも merge を止めない。
- /sync-docs に回すもの:
  - AC8: 388(`org.ShellWord`、`internal/org/shellword.go` の 1 つに寄せて閉じる。いつも quote する `shellQuote` は escalate・approve・watchgate・director_cmd・prompts・feature の文が使い続け、同じ台帳が動詞によって 2 通りに出る。I3 と self-review L2)。390(`cleanup.go` 792 行で余裕 8 行、新しい文は `cleanup_text.go` に置く約束。self-review L6。分けたファイルは `cleanup_text.go` 325 行、`shellword.go` 58 行、`org_cleanup_test.go` 729 行、`org_cleanup_steps_test.go` 598 行)。391((a) と (b) を閉じる)。395(18 文で閉じる。残りは V4 の gate と regate のコマンドと、`ledgerMoveStep` の例外)。420((a)〜(c) を閉じる。(d) は 4 項を閉じ、Linux の結果は PR の CI を証跡にする。「本物の binary は main checkout の root からだけ」の項は計画の Scope の外なので、/test が覆わなければ残す)。
  - 386: D6 の繰り越しの注(Documentation drift の表の行数)。
  - V1: 170・197・290・313・392・401・403 行。
  - V2: 2 つのコメントの `cleanup.go` を `cleanup_text.go` に。
  - V3: `AGENTS.md` の地図に `cleanup_text.go`(と `shellword.go`)。
  - V4: 395 の残りか、新しい行。
