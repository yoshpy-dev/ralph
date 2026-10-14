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
