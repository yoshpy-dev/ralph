# Verify report: org-merge-guarantee

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-guarantee.md(承認済み、digest d2923a8bf359。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7、7 段目の 7c-2)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1。insight は skill のとおり `--cycle auto`)
- Scope: `git diff a16d7c4f...HEAD`(HEAD 1e71cadb、11 コミット、25 ファイル、+3992/-89)。仕様適合(AC1〜AC10 と FR-7 の 2 文)、文書のずれ、静的解析。テストは走らせていない(/test の受け持ち)。テストはコードを読み、あるかどうかと何を比べるかだけを確かめた
- Evidence: `docs/evidence/verify-2026-10-14-org-merge-guarantee.log`(gitignore の対象。run-static-verify の全出力、計画の Verify plan にある追加の静的検査、gh の読み取りの probe、`cmp` と `git diff --check`)

## Spec compliance

行番号は HEAD 1e71cadb のもの。実機の証跡 `docs/evidence/merge-guarantee-livefire-2026-10-14.md` は c56c1f11 の binary で取られている。`git diff --name-only c56c1f11 HEAD` は文書の 4 ファイルだけを返すので、HEAD の Go のコードは証跡を取ったときと同じ。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 ruleset の strict と必須の検査、または ruleset の merge queue と必須の検査で `pass` / `github_protection` | Met | `decide`(`internal/org/mergeguarantee.go:213`)の `rules.strictChecks > 0`(228)と `rules.mergeQueue && rules.checks > 0`(231)。strict の規則は、その規則自身の検査の数だけを数える(`readMergeRules` 346)。`mergeGuaranteeResult`(`internal/cli/doctor_merge_guarantee.go:65`)は `github_protection` を `pass` にする(69)。テスト: `TestMergeGuarantee_RulesetsDecideTheKind` の 3 件(strict と検査 2 件、検査の多い規則を数える、merge queue と ruleset の検査)、`TestCheckMergeGuarantee_StatusFollowsTheKindAndTheResultIsCached`、`TestRunDoctorFull_PrintsTheMergeGuarantee` の pass。実機: github/docs の main で `✓ Merge guarantee: pass — github_protection ... 37 required status checks` |
| AC2 classic の strict true は `github_protection`、admin でなく strict を読めない(404 `Not Found`、403)ときは `unknown` / `info` | Met | `readMergeClassicStrict`(435)。strict true は 247-249 で `github_protection`。404 `Not Found`、body の status 403、stderr の `(HTTP 403)` は 460 で `unknown`。`mergeGuaranteeResult` は `unknown` を `info` にする(73)。テスト: `TestMergeGuarantee_ClassicProtectionDecidesTheKind` の strict true、404 Not Found、403 の 2 形(body と stderr)、ruleset の検査と admin の要る strict の組み合わせ。実機: cli/cli の trunk で `ℹ ... info — unknown: ... can be read only with admin access`。classic の strict true は admin の権限が要るので、実機では見ていない(偽の seam だけ) |
| AC3 設定がない、または必須の検査 0 件で `warn` / `pr_checks_only`、何を設定すればよいかを言う。0 件の strict は「最新にはなるが検査は走らない」 | Met(計画からのずれの注を通して読む) | `checksOnly`(480)が理由を作り、末尾に `mergeGuaranteeFix`(102)を付ける。0 件の strict や merge queue は「the branch becomes up to date but no check runs on the updated head」(491)。勧めは「ruleset か保護ブランチで、必須の検査を 1 件以上付けて Require branches ... を有効にするか、ruleset に merge queue を足して必須の検査を付ける」。計画の本文の「または merge queue」との違いは Progress checklist の「計画からのずれ」(89b981d8、c56c1f11)にあり、判定が merge queue を ruleset の `merge_queue` の規則からしか読まない(373)ことと合う。テスト: RulesetsDecideTheKind の `pr_checks_only` 7 件(うち 6 件が末尾の勧め `mgFix` までを比べる)、ClassicProtection の not strict・off・not enabled・検査なし・`Branch not protected`・`Required status checks not enabled`。実機: この repo の main で `⚠ ... warn — pr_checks_only` |
| AC4 origin・URL・gh なし・gh の失敗(ssh の別名)・期限切れ・JSON の失敗は `info` / `unknown`、終了コードは変わらない、どの gh にも `--hostname` | Met | 失敗の経路は `readOrigin`(256)、LookPath(217、gh を打たない)、`readRepository`(288、github.com 以外の host なら ssh の別名の文を足す)、`readMergeRules`(346)、`ghJSON`(560)、`stopped`(577)で、どれも `unknown`。doctor の終了コードは `countFailed`(`internal/cli/doctor.go:222`)が数える `fail` だけで決まり、この検査は `fail` を返さない。`--hostname` は `ghArgs`(539)の 1 か所で付き、gh の呼び出しはすべて `gh`(549)を通る。`ssh.github.com` は `mergeGuaranteeAPIHost`(277)が `github.com` にする。テスト: `TestMergeGuarantee_FailuresAreUnknownWithAReason`(15 件。gh が PATH にない件は gh の呼び出しが 0)、`DeadlineExceededIsUnknown`、`OneDeadlineCoversEveryCall`、`CallerContextDoneIsUnknown`、`HostComesFromOriginsURL`。`assertGHCalls` が全 gh の引数・dir・env を比べる。実機: `github.com.emu` の origin で `info` |
| AC5 台帳のディレクトリがあるときだけ `unknown` も書く。ないときは作らない。書けなければ古いキャッシュを消し、消せなければ `warn` | Met | `saveMergeGuarantee`(640): stat が ENOENT か非ディレクトリなら `NoLedger` で何も作らず、書けなければ `os.Remove`、消せなければ `RemoveErr`。`mergeGuaranteeResult` は `RemoveErr` のときだけ種類によらず `warn`(82)。テスト: `SaveWritesAnUnknownToo`、`SaveWithoutALedgerDirCreatesNothing`、`WriteFailureRemovesTheOldCache`、`RemovalFailureIsReported`、doctor 側の `NoLedgerDirIsNotCreated`、`FailedWriteRemovesTheOldCache`、`StaleCacheThatCannotBeRemovedWarns`、`EverySaveOutcome`。実機: 台帳がないとき `not cached: there is no org ledger yet at <scratch>/none`、ディレクトリは作られない |
| AC6 `ralph status` の `merge guarantee:` の行と `merge_guarantee`、`unknown` の理由、gh なし、ネットワークなし | Met | `internal/cli/status.go:157` で `ReadMergeGuarantee` を `Dir: "."` で 1 回読み、空の roster(369)と org のある表(571)の両方で `org watch:` の行の直後に出す。JSON は `statusJSON.MergeGuarantee`(436、いつも object)。理由は `ReadMergeGuarantee`(712)の no_ledger・not_checked・unreadable・stale・origin_unreadable・origin_changed で、行の文は計画の Scope の形と同じ。status に gh を打つコードはなく、git は `getMergeOriginURL`(591)の `git remote get-url origin` だけで、キャッシュが新しいときだけ打つ。テスト: `internal/cli/status_merge_guarantee_test.go` の 8 件(台帳なし・キャッシュなし・壊れた・25 時間前、同じ origin を大文字小文字を変えて、origin の向け直し、repo の外、行の位置、JSON の 2 経路、`RunsNoGH`、制御文字)。実機: status の 3 行(この repo、origin の違う場所、repo の外) |
| AC7 `ralph org merge-order` の保証の行と `merge_guarantee`、既定のブランチが違えば `unknown` | Met | `MergeOrder`(`internal/org/mergeorder.go`)は対象がなくても `ReadMergeGuarantee` を読み(254)、対象があれば ls-remote のあとに `ForDefaultBranch`(266、`mergeguarantee.go:810`)を掛ける。`printMergeOrder`(`internal/cli/org_merge_order.go:179`)は既定のブランチの行のあと、その行がなければ先頭に出す。`MergeOrderResult.MergeGuarantee`(160)。テスト: `TestMergeOrder_FillsTheMergeGuaranteeFromTheLedgersCache`(default_branch_changed、origin_changed、`o.Now` での stale)、`TestMergeOrder_NoTargetsReadsTheGuaranteeWithoutTheLsRemote`、`TestOrgMergeOrder_` の 3 件(行の位置と JSON、分岐なしで先頭、制御文字)。実機: 分岐のない merge-order で保証の行が先頭 |
| AC10 origin の向け直し、書き込みの失敗、24 時間 | Met | 向け直し: `TestStatusCmd_MergeGuaranteeIsUnknownOnceOriginPointsToAnotherRepository` と `TestOrgMergeOrder_MergeGuaranteeLineFollowsTheDefaultBranchLine` の「origin pointed to another repository」(どちらも既定のブランチは同じ main)。書き込みの失敗で古いキャッシュが消える: `TestMergeGuarantee_WriteFailureRemovesTheOldCache`、`TestCheckMergeGuarantee_FailedWriteRemovesTheOldCache`。24 時間: `TestMergeGuarantee_ReadMakesAnOldCacheStale`(偽の時計で、24 時間の 1 秒前・ちょうど・1 秒後・72 時間・未来側、`checked_at` が読めない)と merge-order の `o.Now` の stale。実機: origin の違う場所の status で `unknown (doctor checked github.com/yoshpy-dev/ralph; origin is now github.com/github/docs)` |
| AC8 テストは本物の gh を打たず、repo の台帳に書かない | Met | `internal/cli/main_test.go:73` の TestMain が `doctorMergeGuaranteeEnv` を、失敗する Exec・失敗する LookPath・存在しない state dir に固める。`TestRunDoctor_TestMainPinRunsNoGitOrGh` が stub の gh と git の呼び出しの記録で、固定が効くことと state dir が作られないことを確かめる。org 層のテストは偽の `mgExec` で呼ばれ方を記録する。キャッシュを書くテストはどれも `t.TempDir()` の下に書く。`internal/cli` の外で doctor を走らせる Go のテストと shell の test はない(`tests/`、`cmd/`、`.github/workflows/` を grep) |
| AC9 README、skill の 4 面、仕様の注、AGENTS.md、tech-debt。skill の 4 面は同じ中身 | Met(tech-debt の 226 行目のずれは下の表) | README の doctor と status の行、merge-order の段落、skill の動詞の表と「merge してよい順」の節、仕様 FR-7 の 7c-2 の注、AGENTS.md の `internal/org/` の項(`mergeguarantee.go`)、tech-debt の 6 行が判定と表示を説明する。skill の 4 面は `cmp` で同一 |
| FR-7「doctor は保護ブランチに最新でないと merge できない設定がなければ warn」 | Met | AC1〜AC4。設定がない、または検査 0 件のときは `warn`。読めないときは `info`(C3)にした。仕様の 7c-2 の注が両方の読み方と理由を書いている |
| FR-7「status は merge の時点の保証の種類を表示」 | Met | AC6。種類は `github_protection`、`pr_checks_only`、`unknown` の 3 つ。仕様の 7c-2 の注が `unknown` を足した理由を書いている。FR-7 の checkbox は、rebase し直す役が決まっていないので付けない(注に理由がある) |
| 7c-1 の C1(doctor が台帳の下にキャッシュを書き、status と merge-order が読む)との整合 | Met | `docs/plans/archive/2026-10-14-org-merge-order.md:74` の C1 が 7c-2 に回した 4 つ(doctor の warn、台帳の下のキャッシュ、status の行、merge-order の保証の行)がすべて入っている |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack が選ばれた) | pass(rc=0) | shellcheck、hook の `sh -n` と awk、settings の `jq -e .`、Codex の guard 3 つ、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の plan の参照、`gofmt: ok`、`go vet`(出力なし)、golangci-lint 2.9.0 の `0 issues.`、staticcheck 2026.1(出力なし。`command -v` で存在を確認)、branch の secret scan(a16d7c4f..1e71cadb、clean) |
| `gofmt -l .` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の go の行は 1.25.8、手元の go は 1.26.0 |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | vet はテストのコードもコンパイルするので、1.26 だけの API はテストにもない |
| `./scripts/check-skill-sync.sh` | pass | `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | pass | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5 |
| `cmp` で `/org` skill の 4 面を比べる | pass | `.agents/`、`templates/base/.claude/`、`templates/base/.agents/` の 3 つが root と同一 |
| `git diff --check a16d7c4f...HEAD` | pass | 出力なし |
| 変更したファイルの U+FFFD の grep | pass | なし |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の `ralph doctor` の行 | Yes | 検査の名前、gh の 4 回、git と合わせて 20 秒の期限、`--hostname`、fork の origin、pass・warn・info、`fail` にしないこと、キャッシュの書き方(台帳があるときだけ、`unknown` も書く、書けなければ消す、消せなければ warn)がコードと合う |
| README の `ralph status` の行 | Yes | 行の位置、`merge_guarantee`、`unknown` の理由、gh なし、get-url はキャッシュが新しいときだけ |
| README の merge-order の段落 | Yes | 保証の行の位置、分岐なしで先頭、`pr_checks_only` の勧めは merge queue を ruleset に足すと言う |
| `ralph status` と `ralph org merge-order` の help | Yes | README と同じ条件を言う |
| `/org` skill の 4 面 | Yes | 動詞の表の `merge-order` の行と「merge してよい順」の節。4 面は同一 |
| AGENTS.md の repo map | Yes | `internal/org/` の項に `mergeguarantee.go` |
| 仕様 FR-7 の 7c-2 の注 | Yes | C1〜C7 の決定と tech-debt に送ったものがコードと合う。`ssh.github.com` と `github.com` の書き方を変えると `origin_changed` になることも、`ReadMergeGuarantee` が書かれたままの host を比べる実装と合う |
| tech-debt の新しい 6 行 | Yes | Related の列は `docs/plans/active/2026-10-14-org-merge-guarantee.md` を名指す。`/pr` の step 8 の `scripts/archive-plan.sh` が tech-debt の README の `docs/plans/active/` の参照を書き換えるので、archive のあとも切れない |
| tech-debt の 226 行目(gate を通さない PR の行) | No | self-review の M1 と同じ。Trigger の列が「Stage 7c-2 (`ralph doctor`'s protected-branch warning, `ralph status`'s guarantee display), or ...」のままで、この PR がその 7c-2。HEAD でも直っていない。/sync-docs で直す予定(self-review の Follow-ups) |
| 計画の AC の checkbox と Progress checklist | 遅れ(情報) | AC1〜AC10 は `[ ]` のまま。Progress の「Review artifact created」も、self-review の報告(debb12bd)があるのに `[ ]`。merge を止めるものではない |
| 計画の Affected areas | 小さな差(情報) | 新しいテストのファイル `internal/cli/status_merge_guarantee_test.go` と、変えた `internal/org/mergeorder_test.go` が一覧にない |
| Go のコメントの計画のパス | 情報(この PR のずれではない) | `docs/plans/active/2026-10-14-org-merge-guarantee.md` を Go のコメント(テストのファイルを含む)が 7 か所で名指し、`/pr` の archive のあとは切れる。`archive-plan.sh` が書き換えるのは tech-debt の README だけ。前からの書き方で、Go のコメントが名指す `docs/plans/active/` のパス 19 種のうち 18 種はもうない |

## Observational checks

- 実機の証跡(`docs/evidence/merge-guarantee-livefire-2026-10-14.md`)を読んだ。この repo は `warn`、github/docs は `pass`、cli/cli と ssh の別名は `unknown`(`info`)、台帳がなければ書かない。status と merge-order の行は、キャッシュの種類、origin の違い、repo の外を示している。打ち直してはいない
- self-review の Known gaps にある「既定のブランチ名の `/` を `%2F` で渡したとき、各 endpoint が受け付けるか」を、本物の gh 2.102.0 で読み取りだけの GET を打って確かめた(microsoft/vscode の `release/1.85`。どれも公開の repository の読み取りで、何も書いていない)
  - `branches/release%2F1.85` は 200 で `"name":"release/1.85"`、`protection.enabled: true`、`strict` は返らない(計画の Assumptions のとおり)。ないブランチ `no-such%2Fbranch-xyz` は 404 `Branch not found`
  - `rules/branches/release%2F1.85` と、escape しない `rules/branches/release/1.85` は同じ規則(ruleset 7988222 の `deletion` と `copilot_code_review`)を返し、`main` の規則(5 件)とも、ないブランチの `[]` とも違う。`%2F` は名前として読まれている
  - `branches/release%2F1.85/protection/required_status_checks` は 404 `Not Found`。admin でない人への答えと同じで、`%2F` が読まれたかはこの答えからは分からない

## Coverage gaps

- テストは走らせていない。計画の Regression のとおり、/test が `go test ./internal/org/... ./internal/cli/...` を go1.26 と `GOTOOLCHAIN=go1.25.8` の両方で走らせると、手元の go と go.mod の版の差を拾える
- 実機で見ていない分岐: classic の strict true(admin の権限が要る)、merge queue で決まる `github_protection`(github/docs は strict で先に決まる)。どちらも偽の seam のテストだけ
- `protection/required_status_checks` が `%2F` を受け付けるかは未確認(上の probe の 3 つ目)
- 計画の Test plan の Edge cases にある「404 の本文の message が想定外」は、strict の endpoint では 500 の件(`strict fails another way`)だけがテストにある。想定外の message の 404 も同じ最後の `return` を通るので挙動は同じだが、その入力そのものの件はない。足すなら `ClassicProtectionDecidesTheKind` の表に `mgAPIErrorReply("Something else", "404")` で `unknown` の 1 行を加えるのが一番小さい
- doctor の Detail に gh の stderr の制御文字が入るか(self-review の L3)は、gh が実際にそれを出すかを含めて未確認

## Verdict

- Verdict: pass(tech-debt の 226 行目のずれは /sync-docs で直す前提。merge を止めるものではない)
- Verified: AC1〜AC10 と FR-7 の 2 文がコードとテストのコードに対応している(テストは読んだだけ)。静的解析はすべて通った(run-static-verify、gofmt、go vet、go1.25.8 の build と vet、skill と template の同期)。文書は tech-debt の 226 行目を除いてコードと合う。`%2F` の branch 名は `branches/` と `rules/branches/` で通る
- Partially verified: AC2 の classic の strict true と、merge queue で決まる経路(テストのコードだけで、実機では見ていない)
- Not verified: テストの実行結果(/test)、`protection/required_status_checks` での `%2F`、gh の stderr の制御文字

## Cycle 2

- Date: 2026-10-14
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。self-review の cycle 2(432e5912、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 4、Merge: yes)のあとに走らせた
- Scope: `git diff fb954403 HEAD`(HEAD 95e97635、4 コミット、6 ファイル、+182/-24)。8d6499c4(計画の Progress に cycle 2 の決定の 1 行)、59abeaec(`mergeGuaranteeErrText` のエスケープ、`readMergeClassicStrict` の 403 と 404 の分け方、そのテスト、`docs/tech-debt/README.md` の 244 行目)、432e5912(self-review の cycle 2)、95e97635(その insight event)。AC の確かめ直しに要るところは cycle 1 の差分も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD 95e97635 のもの。上の cycle 1 の節の行番号は 1e71cadb のままで、`readMergeClassicStrict` の分岐は 460 行から 463〜471 行に、`mergeGuaranteeErrText` は 600 行台から 616 行に動いた
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は d2923a8bf359 で、`- Approved:` の digest と同じ。cycle 2 の決定は `## Progress checklist` の 168 行に書かれている
- Evidence: `docs/evidence/verify-2026-10-14-org-merge-guarantee-c2.md`。本物の gh で cli/cli を打った doctor と status の行、偽の gh(ネットワークに出ない)を PATH に置いた HEAD の binary の P1〜P5 と、fix の前の fb954403 の binary の対照、その probe の script、静的解析の結果を入れた。コードブロックは scratch のファイルと突き合わせて同じことを確かめた(U+FFFD と生の ESC は 0 件)。`./scripts/run-static-verify.sh` が自分で書いた生の出力は `docs/evidence/verify-2026-10-14-124031.log`(gitignore の対象、名前の時刻は UTC)
- insight event: この回の verify の 1 行を、別のコミットで `docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` に足す(`--cycle auto` は 2 を読む。`--low 2` は下の D1 と D2)

### cycle 2 の直しの確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| L3: gh と git の error text の制御文字を Detail に生で入れない。空白をまとめ、1 文字ずつ `PrintableInboxText` に通し、そのあとで 300 バイトに切る | 満たす(binary でも確認) | `mergeGuaranteeErrText`(`internal/org/mergeguarantee.go:616`、doc は `:608-615`、定数の注は `:96-97`)。呼ぶ非 test の行は 12 か所で、gh と git の error text を Detail に入れる箇所はすべてこの関数を通る(grep で確かめた)。テスト: `TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped`(`mergeguarantee_test.go:560`、cache の往復と `PrintableInboxText(Text()) == Text()`)、`TestMergeGuarantee_ErrorIsCutAfterEscaping`(`:588`、切る位置と 300 バイトちょうど)。binary(偽の gh の `esc`、P5): stderr の `ESC[31m…ESC[0m BEL` は、doctor の行でも status の行でも `\x1b[31m…\x1b[0m\a` と出て、生の ESC と BEL は 0 バイト。同じ偽の gh で fb954403 の binary は doctor の行に生の ESC と BEL を 3 バイト出した(対照)。cycle 1 の L3 の勧め(`mergeGuaranteeErrText` の中で `PrintableInboxText` を通す)のとおり。doctor が Detail を生で出す行(`internal/cli/doctor.go:208`)は変わらないので、Detail の error text 以外の出どころ(`mergeGuaranteeResult` の `s.WriteErr`・`s.RemoveErr`・`s.Path`、`doctor_merge_guarantee.go:78-88`、origin の URL から来る owner/repo)は今も生のまま。self-review の C2-1 のとおりで、binary では打っていない(読んで確かめた) |
| L4: strict の endpoint の 404 `Not Found` は admin の文のまま、403 は「admin がない、または rate limit や SSO など別の原因」と言う。種類はどちらも `unknown` | 満たす(binary でも確認) | `readMergeClassicStrict` の `switch`(`mergeguarantee.go:460-472`)。`Not Found`(`:463`)と 403(`:467`、本文の `status` か stderr の `(HTTP 403)`)は別の case になった。テスト: `ClassicProtectionDecidesTheKind`(`mergeguarantee_test.go:328`)の `strict answers 403:` の 3 行(`:368`、`:371`、`:374`)。binary: P2(`f403`)は `info — unknown: ... is unknown: GitHub answered 403, which can come from missing admin access to the repository or from another cause, such as a rate limit or an SSO authorization the token lacks (...)`、P3(`f404nf`)は fix の前と同じ admin の文。同じ `f403` を fb954403 の binary に打つと admin の文が出た(対照)。cycle 1 の L4 の勧めのとおり |
| test 報告の Gap 1: strict の endpoint の、想定外の message の 404 | 満たす(binary でも確認) | `strict answers a 404 with another message`(`mergeguarantee_test.go:386`、`Branch not found`)。最後の `return`(`mergeguarantee.go:473-474`)の一般の文で `unknown`。binary の P4(`f404bnf`)も同じ文で `info` |
| 登録簿 244 行目に (d)〜(g) を足し、(a) を約 850 行に | 満たす(中身は下の D2 を除いて HEAD と合う) | 5 列(code span の外の `|` は 6 個)。(a) は HEAD の 848 行と合う。(d) の `getMergeOriginURL` と `readMergeOriginRepo`、`defaultMergeGuaranteeGitTimeout`(`mergeguarantee.go:94`、`:783` で使う)、(e) の `MergeGuarantee.Text`(`:836`)と `subject`、(f) の 4 関数(`readRepository`、`readMergeRules`、`readMergeClassicStrict`、`ghJSON` はどれも `g.gh(`・`g.command(`・`g.stopped(` を 1 回ずつ持つ)は HEAD にある。計画の Progress 168 行は「L2・L5・L6 と test の Gap 4 を Small leftovers の行に」と書き、行はそれに Gap 6 と Gap 8 の前半(`%2F`)も足した。足した分は決定を超えるが、逆向きのずれではない |

### AC の確かめ直し(HEAD 95e97635)

cycle 2 の Go の差分は `readMergeClassicStrict`(doc と `switch`)、`mergeGuaranteeErrText`(doc と本体)、定数の注、`unicode/utf8` の import の削除だけで(`git show 59abeaec -- internal/org/mergeguarantee.go`)、`decide`、`readMergeRules`、`readRepository`、`ghJSON`、キャッシュの読み書き、`mergeGuaranteeResult`、status、merge-order は変わっていない。`mergeGuaranteeErrText` の変更は Detail と `origin_unreadable` の理由(`:794`)の文だけを変え、種類と doctor の状態を変えない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(cycle 1 の実機確認のまま) | 判定の経路は変わっていない。cycle 1 の github/docs の `pass` のまま |
| AC2 | 満たす(binary でも確認。本文の言い過ぎは下の注) | classic の strict true: P1(`strict`)で `pass — github_protection ... the branch protection requires branches to be up to date before merging`。cycle 1 では偽の seam のテストだけだったが、今回は binary を偽の gh で通した(本物の GitHub ではない)。strict を読めないとき: P3(404 `Not Found`)と P2(403)はどちらも `info` で `unknown`。本物の gh の cli/cli(404 `Not Found`)も `info` で、行は `docs/evidence/merge-guarantee-livefire-2026-10-14.md` の 63 行目と `cmp` で同じ |
| AC3 | 満たす(cycle 1 の実機確認のまま) | `checksOnly` と `mergeGuaranteeFix` は変わっていない。`mergeClassicNotStrict` の case(`:461-462`)は `switch` の 1 つめのまま |
| AC4 | 満たす(binary でも確認) | gh の失敗: P5(`esc`、HTTP 401)は `info` で `unknown`。doctor の終了コード: 本物の gh の cli/cli は rc 0。偽の gh の P1〜P5 の rc 1 は、絞った PATH に claude がないための `✗ Claude Code CLI: fail` だけによる(doctor の出力で確かめた)。`--hostname`: P1〜P4 の gh の 4 回と P5 の 1 回は、どれも `gh api --hostname github.com` で始まる |
| AC5 | 満たす(binary でも確認) | `SaveMergeGuarantee` は変わっていない。P2〜P5 の `unknown` も `cached in <scratch>/.../merge-guarantee.json` と出て、cli/cli の台帳には `merge-guarantee.json` だけができた |
| AC6 | 満たす(binary でも確認) | P1〜P5 の status の行は doctor の Detail と同じ文で、P5 は `\x1b` の形のまま生の ESC が 0 バイト。`inboxField` の二度目の `PrintableInboxText` が文を変えないことは、`ControlCharactersInAnErrorAreEscaped` の最後の assertion が固める |
| AC7 | 満たす(コードとテストで確認。binary では打っていない) | merge-order は status と同じ `ReadMergeGuarantee` と `inboxField` を通す。cycle 2 で merge-order の経路は変わっていない |
| AC10 | 満たす(cycle 1 の確認のまま) | 向け直し、書き込みの失敗、24 時間の経路は変わっていない |
| AC8 | 満たす | 新しいテストの行と 2 本は偽の `mgExec` を使い、キャッシュは `mgStateDir`(`mergeguarantee_test.go:839`、`t.TempDir()` の下)に書く |
| AC9 | 一部(D1、LOW) | skill の 4 面、README、AGENTS.md、help は cycle 2 で変わっておらず、403 を admin と結ぶ文はない(下の grep)。仕様 FR-7 の 7c-2 の注の 403 の書き方が D1 |

AC2 の本文と fix の関係: AC2 は「classic の保護に必須の検査があり、admin でなく strict を読めない(404 `Not Found` か 403)とき `unknown`(`info`)」と書き、403 を admin のない人の答えとして並べる。fix は 403 の Detail の文を「admin がない、または別の原因(rate limit、SSO)」に変えたが、AC2 が求める結果(種類が `unknown`、状態が `info`)は 404 `Not Found` でも 403 でも変わらず、AC2 は Detail の原因の書き方を求めていない。計画の Scope の 30 行は「404 で message が `Not Found`(admin の権限がない)、403 なら strict は分からない」と書き、fix はこちらの書き方と同じになった。Assumptions の 64 行が観測したのは admin でない人への 404 `Not Found`(cli/cli)だけで、403 を返す実例は repo にない。そのため、これは計画とのずれではなく、AC2 の本文の括弧が 403 を admin のない人の答えと言い過ぎている、と判定した。動きは計画の Scope・C3・C4 と合うので、Progress checklist への記録は必須ではない。記録するなら、候補は次の 1 文: 「cycle 2 の 59abeaec で、strict の 403 は種類を `unknown` のまま、Detail は admin のないことを原因の 1 つとだけ言う形にした(Scope 30 行の書き方。AC2 の括弧の 403 は admin のない人の答えとしては観測していない)」。

計画の C1〜C7: C3(`unknown` は `info`)は P2〜P5 で、C4(strict を読めなければ推測で `github_protection` にしない)は P2・P3・P4 で、C2 と C5(台帳があれば `unknown` も書く)は P2〜P5 の `cached in` で、C7(どの gh にも `--hostname`)は gh の呼び出しの記録で、HEAD の binary でも保たれている。C1(gh が PATH になければ打たない、全体の期限)と C6 は、差分が触れていない。

### 静的解析(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | pass(rc 0) | local verifier の全項目が OK、`check-sync.sh` PASS(IDENTICAL 167、DRIFTED 0)、`check-pipeline-sync.sh` ok、`check-skill-sync: 13 skill(s) in lock-step`、`check-template-purity.sh` PASS、tech-debt の plan の参照 OK、`gofmt: ok`、`go vet` と staticcheck は出力なし、golangci-lint `0 issues.`、`secret-scan-branch: scanned a16d7c4f..95e97635 against origin/main: clean` |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `go vet ./internal/org/... ./internal/cli/...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go vet ./...`、`GOTOOLCHAIN=go1.25.8 go build ./...` | pass | 出力なし。go.mod は `go 1.25.8`。fix が `unicode/utf8` を外し `unicode` をテストで使い始めたが、どちらも 1.25.8 で通る |
| `./scripts/check-skill-sync.sh` | pass | `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | pass | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check fb954403 HEAD` | pass | 出力なし |
| 変更したファイルの U+FFFD の grep | pass | 0 件 |

### 文書のずれ(cycle 2)

`403` を含む非 test の行を README.md、AGENTS.md、仕様、tech-debt、skill の 4 面、`internal/cli`、`internal/org` で grep すると、`mergeguarantee.go` の 4 行(433、434、467、469)のほかは、仕様の 74 行目と tech-debt の 244 行目だけだった。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様 FR-7 の 7c-2 の注(`docs/specs/2026-10-07-org-multi-org-director.md:74`) | 一部ずれ(D1、LOW) | 「classic の保護に必須の検査があり、その strict を読めない(admin でない人への 404 `Not Found`、403)ときは、推測で `github_protection` にせず `unknown` にした」。結果の `unknown` はコードと合う。括弧は「admin でない人への」が 403 にも掛かるように読め、そう読むと、403 では admin のないことを原因の 1 つとしか言わない Detail と食い違う。self-review の C2-3 と同じ判定で、文書のずれと見る。この回の /sync-docs が 1 句を「admin でない人への 404 `Not Found`、または 403(admin のないことは原因の 1 つ)」の形に直すのが一番小さい。直らなければ PR の known gaps に挙げる |
| `docs/tech-debt/README.md:244` の Related の `docs/reports/test-2026-10-14-org-merge-guarantee.md (Test gaps 4 and 6)` | 一部ずれ(D2、LOW) | 今は正しく指す: test 報告の Test gaps の 4 は空の context、6 は `os.Stat` のエラー。前例の org-merge-order の test 報告は cycle 2 以降を `## Cycle N` の節と `### Test gaps(cycle N)` で書き足し、cycle 1 の番号を変えていないので、self-review の C2-4 (b) の「番号が動く見込みが高い」とは違い、番号はおそらく動かない(未確認)。ただし (g) の 3 つめの `%2F` は Gap 8 の前半で、「4 and 6」に入っていない。cycle 2 の /test が節を足すと Test gaps の一覧が 2 つになり、どちらの番号かも曖昧になる。中身(empty context、`os.Stat`、`%2F`)で指すか、「4, 6 and 8」にする |
| `docs/tech-debt/README.md:244` の Why deferred の「cycle 2 of the pipeline, capped at 2」 | yes(情報) | HEAD では正しい(`cycle-count.json` は 2、上限は既定の 2)。上限を上げて 3 回目を回し、(d)〜(f) を直すと偽になる(self-review の C2-4 (a)。同じ形の文が上限の引き上げで偽になった前例は 119 行目)。今のずれではない |
| `readMergeClassicStrict` の doc(`mergeguarantee.go:430-437`) | yes(情報) | 404 `Not Found` は admin の文、403 は原因の 1 つ、という説明はコードと合う。「GitHub answers 403 without admin access」という GitHub の動きの主張は観測されていない(self-review の C2-2)。コードとの食い違いではないので文書のずれには数えず、self-review が勧めた登録簿の (i) に回す |
| `mergeGuaranteeErrText` の doc と定数の注(`:96-97`、`:608-615`) | yes | 空白をまとめる、1 文字ずつエスケープ、そのあとで切る、doctor は Detail をそのまま出す、status と merge-order の二度目のエスケープは文を変えない、の 5 点が P5 と fb954403 の対照の出力に合う。doc は「Detail 全体をエスケープする」とは言っていない。fix の commit message の「so no raw control character reaches the terminal」は error text の部分についての文で、Detail のほかの出どころ(C2-1)を含めると広すぎるが、commit message は文書ではないので情報にとどめる |
| README の doctor の行(`README.md:127`) | yes | 「its `strict`, which needs admin access」は endpoint の要件で、403 の原因を言っていない。cycle 2 で変わっていない |
| skill の 4 面、AGENTS.md、`ralph status` と `ralph org merge-order` の help | yes | cycle 2 で変わっていない。403 と admin を結ぶ文はない(grep)。`check-skill-sync.sh` と `check-sync.sh` が通る |
| 実機の証跡 `docs/evidence/merge-guarantee-livefire-2026-10-14.md` の cli/cli の行(63 行目) | yes | HEAD の binary を本物の gh で打った doctor の行は、scratch の path を `<scratch>` にすると 63 行目と `cmp` で同じ。404 `Not Found` の文は fix で変わっていない。証跡は c56c1f11 の binary のものだが、この行に限っては HEAD でも正しい |
| self-review の cycle 2 が勧めた登録簿の (h)(i) と C2-4 の 2 点の直し | 未追加(情報) | 計画の Progress 168 行は「cycle 2 で新しく出た LOW は直さず同じ行に足す」と決めている。244 行目にはまだない。/sync-docs が足す前提 |
| 計画の AC の checkbox と Progress | yes | AC1〜AC10 は `[x]`、cycle 2 の決定は 168 行にあり、`[ ]` は「PR created」だけ。cycle 1 の節の「checkbox が遅れている」の情報は解消した。Affected areas に新しいテストのファイルがない件は本文なので直さない(cycle 1 のまま) |

### 観察

- 本物の gh 2.102.0 で、origin を `https://github.com/cli/cli.git` にした空の repo に HEAD の binary の doctor を打った(読み取りの GET だけ、台帳は scratch)。`info — unknown`、rc 0、キャッシュは `kind: unknown`、`default_branch: trunk`、`api_repo: cli/cli`。status の行は `unknown (trunk of github.com/cli/cli, checked 2026-10-14T12:40:52Z): ...`
- 偽の gh の P1〜P5 は、cycle 1 の verify が「実機では見ていない」とした classic の strict true(P1)を、binary を通して初めて `pass` で見た。本物の GitHub での確認ではない
- P2 の 403 の本文の message(`Resource not accessible by personal access token`)は、偽の gh に私が書いたもので、GitHub がこの endpoint でそれを返すかは確かめていない
- doctor は herdr を `exec.LookPath` で探すだけで(`driver.checkAvailable`)、herdr のサーバーには触れない。打つ前に読んで確かめた

### Coverage gaps(cycle 2)

- テストは走らせていない(/test の担当)。新しい 4 行と 2 本、`unicode` の import を go1.26 と `GOTOOLCHAIN=go1.25.8` の両方で走らせるのは /test
- strict の endpoint が本物の GitHub で 403 を返す実例は、今回も見ていない(self-review の C2-2)。P2 は偽の gh
- `ralph org merge-order` の保証の行は、cycle 2 では binary で打っていない(経路は変わらず、テストが受け持つ)
- C2-1 の、Detail の error text 以外の出どころ(cache の OS のエラー、state dir の path、scp 形の origin の owner/repo)が doctor の行に生で出ることは、コードを読んで確かめただけ
- `protection/required_status_checks` が `%2F` を含む名前を受け付けるかは、cycle 1 のまま未確認(admin の権限が要る)
- 足すと確信がいちばん増える確認: この回の /sync-docs のあとに、仕様 74 行目の 403 の句と登録簿 244 行目((h)(i) の追記、Related の Gap の指し方)を grep し直す。D1 と D2 が閉じたかが分かる。打つのは grep 2 回で、ネットワークに出ない

### 判定(cycle 2)

- Verdict: pass
- Verified: cycle 2 の 3 つの直し(L3、L4、Gap 1)をコードとテストの名前で辿り、HEAD 95e97635 の binary で P1〜P5 として再現し、fb954403 の binary で直す前の動きと見比べた。本物の gh の cli/cli の行は実機の証跡と同じ。AC1〜AC10 は満たしたまま(AC2・AC4〜AC6 は binary で確かめ直し、AC1・AC3・AC7・AC10 は経路が変わっていないことを diff で確かめた)。計画の C1〜C7 は保たれている。静的解析はすべて通った(go1.25.8 の build と vet を含む)。計画の digest は承認時と同じ
- Partially verified: AC9(D1)。登録簿 244 行目の指し方(D2)。AC2 の本文の括弧は 403 を admin のない人の答えと言い過ぎているが、計画とのずれではない(上の注)
- Not verified: テストの実行(/test)、本物の GitHub の 403、merge-order の binary の行、C2-1 の出どころの binary での再現、`%2F`
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2(D1、D2)。情報は 5 つ(244 行目の cap の数字、`readMergeClassicStrict` の doc の未観測の主張、fix の commit message の範囲、(h)(i) の未追加、AC2 の本文の言い過ぎ)
