# sync-docs report: org-ledger-leftovers

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-org-ledger-leftovers.md`(PR #296 の穴と tech-debt 388・390・391・395・420 行、AC1〜AC8、D1〜D7、S1〜S5、S2 の補い、F1)
- Branch: `fix/org-ledger-leftovers`。run 開始時の HEAD は `deb8c6c0`、base は origin/main `c8c8c0f1`(`e208a2c6` で取り込み済み)
- Pipeline cycle: 1
- 先行 report: self-review(Merge: yes、LOW 11。10 件は `bb0afab1` で直し、L6 だけ残る)、verify(pass、LOW 4 と INFO 3)、test(pass、`run-test.sh` rc 0。Linux だけの 2 本は darwin で SKIP)
- 制約: Go のファイル、README、`/org` と `/director` の skill は変えていない。plan は AC1〜AC6 と AC8 の箱、/sync-docs の行だけを変えた。digest は前後とも `78557bfbc265`

## Summary

tech-debt の 388・390・391・395・420 行を、直した形に書き換えた。391 は全部払ったので全体を取り消し線にし、388・390・395・420 は払った部分に取り消し線と「Closed in fix/org-ledger-leftovers (<sha>)」を付け、残る部分に Impact と Trigger を書いた。386 には繰り越しの注を足した。verify V1 の 7 行(170・197・290・313・392・401・403)は、名前と場所を直し、当たったが払わなかった Trigger に日付つきの注を足した。Trigger の sweep で、verify も依頼も挙げていない 389 行が当たっていたので、同じ形で足した。AGENTS.md の `internal/org/` の行には `cleanup_text.go` と `shellword.go` を足した(V3)。

表の末尾に新しい行は足していない。残る穴はすべて既存の行に入った(下の「表の末尾に足さなかった理由」)。

コード、help、skill の文は変えていないので、pipeline の再実行は要らない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 14 行を書き換えた(170、197、290、313、386、388、389、390、391、392、395、401、403、420)。ほかの行は同一。内容のコミットは `b3d3d06b` |
| `AGENTS.md` | repo map の `internal/org/` の行に `cleanup_text.go`(cleanup の拒否の文と、それが名指すコマンド)と `shellword.go`(`ShellWord`、`VerbCommand`)を 1 句ずつ足した。同じコミット `b3d3d06b` |
| `docs/plans/active/2026-11-02-org-ledger-leftovers.md` | AC1〜AC6 と AC8 の箱を `[x]` にし、Progress checklist に「Docs synced」の行を足した。AC7 の箱は付けていない(下の「lead へ」) |
| `docs/reports/sync-docs-2026-11-02-org-ledger-leftovers.md` | この report |
| `docs/insights/events/2026-11-02-org-ledger-leftovers.jsonl` | `sync_docs` の行(cycle 1、pass、findings 0)を `insights-append.sh --slug org-ledger-leftovers --cycle auto` で足した。既存のファイルに入り、迷子のファイルはできなかった |

## 5 行の扱い

| 行 | 扱い |
|----|------|
| 388(quote の関数が 2 つ) | 「2 つの関数」の部分を閉じた。S2(`47a4fda1`、`f972c80b`)と F1(`bb0afab1`)で、要るときだけ quote する関数は `ShellWord`(`internal/org/shellword.go`)の 1 つになった。`cleanupShellWord`、`shellQuoteIfNeeded` と `shellSafeUnquoted`、計画の外で見つかった `directorShellWord` が消えた。表のテストは org の層と CLI の層に 1 本ずつ(6 つの値)。残りは、いつも quote する `shellQuote` を escalate・approve・watchgate・director_cmd・prompts が直接使い続けるため、`InboxCommand` と `CommandWithStateDir` が書く `--state-dir '/state/org'` と、merge と cleanup が足す `--state-dir /state/org`(`VerbCommand`)で同じ台帳が 2 通りに出ること(self-review L2、verify I3)。どちらも打てば同じ動きをする。Why deferred に、inbox の文が `shellQuote` を選んだのは feat/org-cross-watch の決定で、この PR の Scope(D3)は要るときだけ quote する関数だったと書いた。新しい Trigger は `InboxCommand`・`CommandWithStateDir`・`approve` の `--state-dir` の文の次の変更 |
| 390(長すぎるファイル) | 払った。S1(`3b54f1b1`、移すだけ)と F1(`bb0afab1`)。`cleanup.go` 1119 → 792 行、`cleanup_text.go` 325、`shellword.go` 58、`org_cleanup_test.go` 1280 → 729、`org_cleanup_steps_test.go` 598(`6c400e87` で数えた)。移した宣言は org 18 本、cli 10 本で、宣言ごとに byte 単位で同じ。残りは `cleanup.go` の余裕が 8 行しかないこと(self-review L6)。新しい文や拒否のコマンドは `cleanup_text.go` に置く約束と、「`cleanup.go` が 800 行を超えたら `checkCleanupParams` ほか残った組み立てを移す」を Trigger に書いた。Why deferred に、候補 15 個だけでは 903 行で届かず `checkRecord`・`checkRegistration`・`pathWithin` も移したが、拒否の文を作る `checkCleanupParams` は残った、と書いた |
| 391(cleanup の文とコメント) | 全部払った(RESOLVED)。S4(`94c07d76`)。(a) `recordGone` は「remove what is left by hand」で終わり(今は `cleanup_text.go`)、(b) `registrationAtRisk` のコメントは detached HEAD だけを名指す。Debt・Impact・Trigger を取り消し線にした |
| 395(他の動詞の案内に `--state-dir`) | 「Left open」を閉じた。S3(`a326e050`)と F1。18 文(org の層 15、CLI 3)が `VerbCommand` / `orgVerbHint` を通り、`--state-dir` を渡したときだけ ` --state-dir <dir>` を付ける。2 つの台帳で出力の merge-order を打ち直すテストがあり、build した binary も CLI の文でそれを確かめた(org の層の 15 文はテストだけで、binary は出していない)。残りは verify V4: merge-order の Detail が名指す `ralph org regate --branch <branch>`(`mergeRegateCommand`、branch は `ShellWord` を通らない)と `ralph org gate --org-id <org>`(`mergeGateCommand`)に `--state-dir` が付かない。merge はその Detail を「ready でない」拒否にそのまま写す。gate・regate の行(`gateWording.again`、403)には入れなかった。403 は regate 自身の拒否の文の quote で、V4 の直し先は merge-order の Detail の作り方(`mergeorder.go`)と、その出力のテストだから。Trigger は `mergeRegateCommand`・`mergeGateCommand`・merge-order の Detail・merge の「ready でない」拒否の次の変更。`ledgerMoveStep` が古い値を繰り返さないのは意図した例外で、債務ではないと書いた |
| 420(ledger-gaps の小さな残り) | (a) を閉じた(S4、`94c07d76`: help は理由のコマンドが 2 通りあると書き、`TestOrgCleanup_Help` が固定する。`--branch` なしの 2 つの場合は既存の `TestOrgCleanup_RefusalsAreSkippedLines` と `TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing` が見る)。(b) を閉じた(388 と同じ)。(c) は `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` について閉じた(S5、`cf69cab5`: 期待を `org.ShellWord` で作り、空白を含む一時ディレクトリの subtest がある)。残りは、test report の Failure analysis の 10 本: 一時ディレクトリのパスに空白があると、`internal/org/cleanup_test.go` の 9 本(サブテスト 16)と `internal/cli/org_cleanup_test.go` の `TestOrgCleanup_LeavesOutCleanedAndReboundBranches` が落ちる。base `c8c8c0f1` でも同じ 10 本が落ちる。期待を quote なしのパスで組むためで、コードの出力は `ShellWord` で正しい。テスト名を全部書いた。(d) は notDir の拒否、`git worktree remove` の失敗、`ok == false`、Linux の `os.Getwd` の 4 項を閉じた(S5)。残りは「本物の binary は main checkout の root からだけ打った」で、この branch の実物の確かめも scratch repository の root から打ったため閉じていない。Linux のテストは darwin で SKIP、linux/arm64 の docker で PASS、linux/amd64 は PR の CI を証跡にする、と書いた。Impact・Why deferred・Trigger・Related も項ごとに直した |

### 表の末尾に足さなかった理由

計画の Risks は末尾に足さないと決めていて、足す必要もなかった。空白で落ちる 10 本は 420(c)、V4 は 395、`shellQuote` の残りは 388、`cleanup.go` の余裕は 390、実物の確かめの場所は 420(d) に、それぞれ元の行の続きとして入った。#269 の側の行(422〜424)とは行が重ならない。

## 386 の繰り越しの注

Trigger の末尾に 2026-11-02 の注を足した。この branch は 4 つのうち 3 つを編集し、どれにもテストを足していない(`origin/main` `c8c8c0f1` と `6c400e87` で数えた)。

- `internal/cli/org_merge_test.go`: 1207 → 1195 行(`--state-dir` の案内の期待と、`orgLayerShellWord`・`orgPlainWord` の削除)
- `internal/org/merge_run_test.go`: 1111 行のまま(3 行。`cleanupShellWord` が `ShellWord` になった)
- `internal/org/cleanup_test.go`: 1817 行のまま(17 行。同じ改名、`TestCleanupShellWord` の改名、「remove what is left by hand」の期待 4 行、コメントの語)
- `internal/org/director_skill_sync_test.go`: 変えていない(1223 行)

払わなかった理由は、移すだけの分割は Scope の外で、計画の D6 が 4 つとも分けないと決めたこと(全部分けると移すだけの diff が 5,000 行を超える)。この branch のテストは新しいファイルに入れた。

## verify V1 の 7 行と、sweep で足した 389 行

| 行 | 古くなった名前・場所 | Trigger の注 |
|----|----------------------|--------------|
| 170(`LegacyWorktreeStateDir`) | 名前は古くない | `guardLegacyOrgStateDir` の 3 つの文が、消えた `shellQuoteIfNeeded` の代わりに `org.ShellWord` を使う。`: @ + ,` を含むパスは quote なしで出る。当たったが、`os.Stat` と `SamePath` の順は変えておらず、Scope は quote の関数なので払わなかった。V-2 の注は変わらない |
| 197(l) | (l) の前半「`shellQuote` は `shellQuoteIfNeeded` の quote の枝の写し」を閉じた(`shellQuoteIfNeeded` が消え、`shellQuote` は `shellword.go` に移り、CLI は `org.ShellWord` を呼ぶ)。`mustAbs` の半分は残した | (l) の Trigger を、quote の部分は取り消し線で閉じ、`StartFeature` の `p.StateDir` の扱いの次の編集に書き換えた。Impact は quote の部分を閉じ、`mustAbs` の部分を残した |
| 290(`regate_test.go`) | 名前は古くない | コメント 1 行(`cleanupShellWord` が `ShellWord`)で当たった。1596 行のまま。分割は Scope の外で払わなかった。`regate_merge_test.go` は触っていない |
| 313(`wake_cmd` の `{target}`) | 「`shellQuote` in `internal/org/feature.go`」を `shellword.go` に直した | Trigger には当たっていないので注は足していない |
| 389(`git log` の前に fetch がない拒否) | 拒否の場所を `cleanup.go` から `cleanup_text.go` に直した | sweep で見つけた。`recordGone`(S4 で語を変えた)が「those refusals」に当たり、S1 で `checkRecord`・`checkRegistration`・`recordGone` を移した。`fetchHeadCmd` を名指す直しは S4 の 1 句の範囲の外で払わなかった |
| 392(`resolvedOrClean` の呼び出し元) | 呼び出し元の `pathWithin` が `cleanup_text.go` に移ったと書いた | 呼び出し元の `cleanup.go` と `feature.go` を編集して当たった。呼び出しの改名は Scope の外で払わなかった。Related に計画と verify を足した |
| 401(`regate.go`・`regate_ci.go`) | 名前は古くない | `regate.go` 1 行、`regate_ci.go` 3 行(改名と、コメントの語)で当たった。995 行と 901 行のまま。`--wait-checks` の待ちの分割は Scope の外で払わなかった |
| 403(`gateWording.again`) | Trigger の「`cleanupShellWord` で」を `ShellWord` に直し、元の名前が `cleanupShellWord` だったと括弧で残した | `regateCheckWorktree` と `regateCIPreCheck` の 2 つの拒否の改名で当たった。もとから branch を quote していた文なので、共通の末尾 `gateWording.again` と R2-2 の local の文は変えておらず払わなかった |

### sweep で退けたもの

全 open 行の Trigger(取り消し線の外)と Debt を、この diff が変えたファイル名・関数名・help の語で引いた。「any of」「these files」型の Trigger も別に引いた。当たりに見えて退けたものは次のとおり。

| 行 | 退けた理由 |
|----|------------|
| 150、134、394(`org_test.go`) | 変えたのは別のテストのコメント 1 行。Trigger が指すテストではない |
| 171(`pathWithinDir`) | cleanup の `pathWithin` とは別の関数 |
| 176、200(`internal/cli/org.go`、`feature.go`) | Trigger はそれぞれ `Stop` 系の関数、`TestStartFeature_*` と `otherRecordFix` で、この diff が変えた所ではない |
| 237(`newOrgMergeCmd` の signal) | signal の扱いは変えていない |
| 248、249、304(`CheckMergeEnforcement`、`mergeenforce.go`) | 変えたのは案内の文の `--state-dir` だけで、保護の読み方と各 Trigger の条件は変えていない |
| 266(`absPath`) | Trigger は `checkDirectorStart` の path 解決。新しい Linux のテストは本物の `absPath` を呼ぶだけ |
| 274(`DirectorCleanupError`) | 変えたのは `newDirectorCleanupError` の quote で、Trigger の `DirectorCleanupError.Error`・`directorRecordStopError`・`restoreLocked` ではない |
| 301(`mergeGHSaid`、`mergeLsRemoteRefusal`) | 変えていない(`mergeAndRecord` の文の中だけ) |
| 397(`checkRetiredOrgStateDirEnv`) | Trigger は拒否の条件の変更と、README・skill の段落の変更。この diff は quote の関数を替え、`ok == false` のテストを足しただけで、条件は変えていない。依頼どおり、397 は次の計画の分として触っていない |
| 414(`mergeorder.go`) | 変えていない。V4 を直すときに触ることになる |

## ほかの文書の食い違い

| 対象 | 結果 |
|------|------|
| README の merge の段落(`ralph org merge`)と cleanup の段落 | `ralph org cleanup --help` の「理由のコマンドは 2 通り」と、merge-order・inbox・status・disband の `--state-dir` を読んだ。cleanup の段落の「most often … run `ralph org cleanup --branch <branch>` again」は help の書き分けと矛盾しない。merge の段落の「after any other skip, fix the reason and run `ralph org cleanup --branch <branch>`」は、merge が cleanup に必ず branch を渡すので正しい。どちらの段落も、案内のコマンドが `--state-dir` を繰り返すことを書いておらず、細部は `--help` に任せている。変えるところはない(verify I1 と同じ判断) |
| `/org` skill の 4 つの写し | `cleanup` と `merge` の行は、片付けが断られたあとは `--branch` で打ち直すと言うだけで、理由がどのコマンドを名指すかも、案内のコマンドが `--state-dir` を持つかも書いていない。変えていない。`./scripts/check-skill-sync.sh` は 14 skill が揃っている。台帳の場所と `RALPH_ORG_STATE_DIR` の条件の項は 397 の分なので触っていない |
| `/director` skill | `CLEANUP MANUAL:` のない断りは `ralph org cleanup --branch <ブランチ>` を打ち直す、という文は今も正しい。変えていない |
| `docs/specs/`、`.claude/rules/`、`docs/quality/`、`docs/architecture/` | 変えた関数名も、出力の文も名指す所はない(grep) |
| verify V2(`cleanup.go` にあると書くコメント 2 つ) | `6c400e87` で直し済み。`followup.go` は `cleanupDetails in cleanup_text.go`、`cleanup_test.go` は `registrationAtRisk of cleanup_text.go` |
| 消えた名前の文書 | `shellQuoteIfNeeded`・`cleanupShellWord`・`directorShellWord` を、取り消し線の外で名指す tech-debt の行は残っていない。203 行は取り消し線の中の履歴なので直していない。計画と古い evidence は記録なので直していない |

## lead へ

- AC7 の箱は付けていない。verify は Partially verified、test は Linux だけの 2 本が darwin で SKIP で、AC の条件は「CI(ubuntu-latest)で通る」。PR の CI が通ったら、lead か /pr が付ける。tech-debt の 420(d) にも、Linux の結果は PR の CI を証跡にすると書いた。
- Progress checklist の「Verification artifact created」と「Test artifact created」の箱は、依頼が「plan のほかを変えない」としたので付けていない。verify と test の report はどちらもコミット済み(`e1f2184a`、`deb8c6c0`)。
- 空白で落ちる 10 本は直していない。Go のファイルを変えない依頼だったため。9 本は 386 の 4 ファイルの 1 つ(`internal/org/cleanup_test.go`)にあり、計画の D6 はこのファイルにテストを足さない。直し方は test report の Failure analysis に書いてある(期待と、`sh -c` で打つコマンドを `ShellWord` で組む)。
- tech-debt の行は計画を `docs/plans/active/2026-11-02-org-ledger-leftovers.md` で書いた。`scripts/archive-plan.sh` が /pr で `archive/` に書き換える。`verify.local.sh` の計画参照の検査は、いまの path で通る。
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff origin/main...HEAD` は 50 ファイル、+2,868/−1,171。Go のコードは 42 ファイルで +2,055/−1,156(本番コード 16 ファイルで +535/−467)。walkthrough はここでは書いていない。

## 確かめた結果

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。plan と report を書いたあとにもう一度打って、同じ
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていないので、4 つの写しは前後で同一
- `./scripts/run-static-verify.sh`(内容のコミット `b3d3d06b` の上で、plan と report を書く前の tree): rc 0、`All verifiers passed.`。gofmt `ok`、`0 issues.`、check-pipeline-sync、check-template-purity、tech-debt の計画参照の検査はすべて OK。branch secret scan は `c8c8c0f1..b3d3d06b` で clean。ログは `docs/evidence/verify-2026-11-02-054541.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。`run-test.sh` は打っていない(Go のファイルを変えていない。直前の test report は rc 0)
- tech-debt の構造: `git show HEAD:docs/tech-debt/README.md` と行ごとに比べた(`HEAD` は内容のコミットの前)。変わった行は 170、197、290、313、386、388、389、390、391、392、395、401、403、420 の 14 行だけ。14 行ともに 5 欄に割れ、欄ごとの backtick と `~~` が偶数で、`file:line` の参照はない。引用した `docs/plans`・`docs/reports`・`docs/evidence` の path は全部ある。`verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep でも欠けはない
- plan の digest: `./scripts/plan-visual.sh digest` は `78557bfbc265`(承認時と同じ)。plan の diff は +8/−7 行で、箱 7 つと Progress の 1 行
- `git status --porcelain`(コミット前): `AGENTS.md`、`docs/tech-debt/README.md`(内容のコミットに入れた)、plan、`docs/insights/events/2026-11-02-org-ledger-leftovers.jsonl` が `M`、この report が `??`。迷子のファイルはない
