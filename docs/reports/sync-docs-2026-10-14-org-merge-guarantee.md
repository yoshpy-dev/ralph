# sync-docs report: org-merge-guarantee

- Date: 2026-10-14(JST)
- Plan: `docs/plans/active/2026-10-14-org-merge-guarantee.md`(承認済み。`./scripts/plan-visual.sh digest` は `d2923a8bf359` で、`- Approved:` の行と同じ。この run が変えたのは checkbox の印と Progress checklist の 3 行の注記だけ)
- Pipeline cycle: 1(`--cycle auto`。insight event は cycle 1)。差分の base は `a16d7c4f`、branch は `feat/org-merge-guarantee`。run 開始時の HEAD は `49cc80cb`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-14-org-merge-guarantee.md`(Merge: yes、CRITICAL・HIGH 0、MEDIUM 1、LOW 6)、`docs/reports/verify-2026-10-14-org-merge-guarantee.md`(pass。文書のずれは tech-debt の 226 行目だけ)、`docs/reports/test-2026-10-14-org-merge-guarantee.md`(pass、失敗 0)
- 項目の番号: M1 と L1 は self-review の Findings の ID。verify の「Documentation drift」の表の「tech-debt の 226 行目」は M1 と同じ指摘
- 内容の commit: この report、tech-debt、`internal/org/mergeguarantee.go` のコメント、計画の checkbox を含む 1 つの commit(`docs: add cycle-1 sync-docs report for org-merge-guarantee`)。insight event の jsonl は次の 1 commit に分けた(`docs: record the cycle-1 sync-docs insight for org-merge-guarantee`)
- 中断: 1 回目の実行は利用上限で止まった。再開したとき、作業ツリーの未コミットの変更(計画、tech-debt、`mergeguarantee.go` の 3 ファイル)を `git diff --numstat` で数え直した(計画 13 / 13、tech-debt 1 / 1、`mergeguarantee.go` 2 / 2)。計画の checkbox の Edit は 13 行すべてが入っていて、やり直した手順はない

## Summary

self-review の M1(verify の drift と同じ指摘)で、tech-debt の 226 行目(feature org の leader が `ralph org gate` を通さずに PR を出せる、という行)を書き換えた。行は閉じていない。この PR の 7c-2 が入っても、gate を通していない PR を止める仕組みは ralph にない。Trigger の列から「Stage 7c-2」を外し、Impact の列に、doctor、status、merge-order が出すのは既定のブランチの保護で、どれも gate なしの PR を止めないという文を足した。

L1(`mergeOriginName` の doc コメントの文)をコメントだけ直した。コードは変えていない。

計画は、verify が Met とした AC1 から AC10 の 10 個の checkbox と、Progress checklist の Review・Verification・Test の artifact 3 行を `- [x]` にした(3 行には報告のパスを添えた)。`PR created` は付けていない。

そのほかに、この差分(doctor の Check 11c、`merge-guarantee.json`、status と merge-order の `merge guarantee:` の行、JSON の `merge_guarantee`)が触れる文書を掃いた。README、AGENTS.md の repo map、`/org` skill の 4 面、仕様 FR-7 の 7c-2 の注、tech-debt の追加行、`docs/architecture/repo-map.md`、`docs/quality/`、`.claude/rules/ralph/` は、コードと合っていたので、上の 3 件のほかは変えていない。

## Changes made

| File | 項目 | Change |
|------|------|--------|
| `docs/tech-debt/README.md`(「Nothing stops a feature org's leader from pushing and opening the PR without `ralph org gate` ...」の行、226 行目)の Trigger の列 | M1 | 「Stage 7c-2 (`ralph doctor`'s protected-branch warning, `ralph status`'s guarantee display), or a report of a feature PR merged without a gate」を「A report of a feature PR merged without a gate」にした。元の Trigger に 8a 段目の director の条件はなかったので、足していない |
| 同じ行の Impact の列 | M1 | 「A PR can carry a head that no gate passed, and a human who merges it sees no warning today」の後ろに 1 文足した: `ralph doctor` は origin の既定のブランチに保証がない(`pr_checks_only`)とき warn し、`ralph status` と `ralph org merge-order` は cache した種類を出すが、どれも保護を述べるもので、ralph の gate を通していない PR を止めない。元の文は変えていない(元の文は gate の有無の警告がないことを言い、足した文は既定のブランチの保護の警告を言うので、食い違わない) |
| `internal/org/mergeguarantee.go`(`mergeOriginName` の doc コメント) | L1 | 「mergeOriginName is host/owner/repo, or "no readable origin" for a cache whose check could not read origin (host and repo both empty).」に直した(2 行のまま)。関数の本体は変えていない。ファイルは 834 行のまま(tech-debt の行(a)の「about 830 lines」と合う) |
| `docs/plans/active/2026-10-14-org-merge-guarantee.md`(`## Acceptance criteria`) | - | AC1 から AC10 の 10 個を `- [x]` にした。verify の「Spec compliance」の表は 10 個とも Met。AC3 は「Met(計画からのずれの注を通して読む)」で、計画の Scope の「または merge queue」と実装(merge queue は ruleset に足すよう言う)の違いは Progress checklist の「計画からのずれ」にある。AC9 は「tech-debt の 226 行目のずれ」が M1 の直しで消えた |
| 同じ計画(`## Progress checklist`) | - | `Review artifact created`、`Verification artifact created`、`Test artifact created` を `- [x]` にし、報告のパスと判定を添えた。`PR created` は付けていない |
| `docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-merge-guarantee`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記(`git status --short` で `M`。新しい未追跡のファイルはない)。別の commit に入れる |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`(`ralph status` の行、`ralph doctor` の行、`ralph org merge-order` の段落) | 変更なし。verify が読んだとおり、行の位置、`merge_guarantee`、`unknown` の理由、gh を打たないこと、get-url は cache が新しいときだけ、`pr_checks_only` の勧めが merge queue を ruleset に足すと言うこと、doctor が `fail` にしないこと、が実装と合う。merge-order の段落の「the gate records are the only evidence it reads」は `ready` の判定の話で、保証の行は判定に使わないので、今も合う |
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。動詞の表の `merge-order` の行と「merge してよい順」の節の保証の段落が、README と同じ条件(24 時間、origin の host と owner/repo、既定のブランチの違い、台帳がなければ書かない)を言う。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る。`/org` skill は `ralph status` の `org watch:` の行を 1 か所(見張りが動いているかの確かめ方)で挙げるだけで、`merge guarantee:` の行は挙げる必要がない |
| `AGENTS.md`(repo map の `internal/org/` の行) | 変更なし。`mergeguarantee.go` の句が入っている。`internal/cli/` の行は subcommand の名前だけで、新しいファイル(`doctor_merge_guarantee.go`)を挙げない既存の書き方と同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の「7c-2 で決めたこと」の注) | 変更なし。gh の 4 回の呼び出しの順、`--hostname` と `GH_HOST`、3 つの種類の決め方、doctor の状態、cache の書き方と読み方、tech-debt に送ったものが、verify の確認どおりコードと合う。FR-7 と FR-8 の checkbox は、rebase し直す役と 8a が残るので未完のまま。7c-1 の注(76 行目)の「7c-2(次の PR)に残した」は、7c-1 を書いた時点の記録で、7c-2 の注が 74 行目にあるので、そのままにした |
| `docs/tech-debt/README.md` の追加 6 行(240 行目から 245 行目)と 233 行目 | 変更なし。挙げた名前(`mergeGuaranteeAPIHost`、`decide`、`ReadMergeGuarantee`、`doctorMergeGuaranteeEnv` など)は関数と変数の名前で、`file:line` の参照は 226 行目にも新しい 6 行にもない。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き換える。233 行目は「since stage 7c-2」の書き方に直っている |
| `docs/tech-debt/README.md` の 226 行目の `docs/plans/archive/2026-10-13-org-gate.md` | 合う。archive 済みの計画で、パスは今もある |
| `docs/quality/quality-gates.md` と template の写し | 変更なし。「Cross-org watch gate」の行の「work with no watch」の一覧は、7c-1 で `merge-order` を足した後のまま合う(doctor と status は org の動詞ではない)。保証の判定は gate や拒否ではなく、新しい検査も足していない |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md`、`docs/architecture/design-principles.md` | 変更なし。repo-map は `internal/org/` の個別のファイルも doctor の検査も挙げない |
| `.claude/rules/ralph/`(`agent-messaging.md`、`model-routing.md` ほか)と template の写し | 変更なし。doctor の新しい検査と status・merge-order の行は agmsg のメッセージも受信箱も触らない |
| `docs/recipes/`(`codex-setup.md`、`codex-seat-permissions.md` ほか)、`.codex/README.md` | 変更なし。`ralph doctor` の検査を挙げる箇所は codex、herdr、agmsg、hooks の検査の説明で、検査の一覧を数えていない |
| `internal/cli/doctor.go` の「subprocess を増やすのは `--probe-models` だけ」の注釈と Check 11c のコメント、`ralph status` と `ralph org merge-order` の `--help` | 合う。注釈は git の `rev-parse` と `remote get-url origin`、`gh api` 最大 4 回、1 つの 20 秒の期限、gh が PATH になければ gh を打たないことを書く。verify の文書の表と同じ結論 |
| Go のコメントとテストが名指す `docs/plans/active/2026-10-14-org-merge-guarantee.md`(7 か所) | 変えていない。`/pr` の archive のあとに切れるが、`archive-plan.sh` が書き換えるのは tech-debt の README だけで、前からの書き方(Go のコメントが名指す `docs/plans/active/` のパス 19 種のうち 18 種はもうない)。verify の情報の項目と同じ判断 |
| 証跡の `$HOME` の置き換え | `docs/evidence/merge-guarantee-livefire-2026-10-14.md` と `docs/evidence/test-2026-10-14-org-merge-guarantee-*.txt` の 4 本、この PR の 3 つの報告で、`grep -c -E '/Users/[a-z]'` はどれも 0 件(`docs/quality/definition-of-done.md` が求める `~` への置き換えは済んでいる) |

## Left open

- self-review の L2(`readMergeOriginRepo` と `getMergeOriginURL` の読み取りが 2 つある。`ReadMergeGuarantee` の get-url の 10 秒が `MergeOrderParams` に出ない)、L3(doctor の Detail に gh の stderr の制御文字が残りうる)、L4(strict の 403 の文が原因を言い切る)、L5(status と merge-order の行が branch と repository を 2 回言う。直すと README と skill の例の行が変わる)、L6(gh の呼び出しの 3 行が 4 か所にある)は、Go のコードに触れる直しなので、この run では直していない。直すと `/self-review` から回し直しになる。self-review は、直さずに進めるなら L2 と L6 を 1 行にまとめて tech-debt の `mergeguarantee.go` の行(a)の近くに足すよう勧めている。足すかどうかは lead の判断に任せた
- verify の Coverage gaps と test の Test gaps(strict の endpoint の 404 で message が想定外の行がない、classic の strict true と merge queue と classic の検査の組み合わせは偽の seam だけ、`protection/required_status_checks` が `%2F` を受け付けるかが未確認、gh の stderr の制御文字が未確認)は、文書では直せない。どれも merge を止めるものではないと両方の報告が言う
- 7c-1 の注(仕様 76 行目)の「7c-2(次の PR)」の「次の PR」は、7c-1 を書いた時点の言い方のまま。7c-2 の注が 74 行目に入ったので、読み違えは起きにくいと判断して直していない
- release のノート(計画の Rollout が言う、doctor が gh で保護ブランチを読むこと(ネットワーク、最大 20 秒)、`merge-guarantee.json`、status と merge-order の `merge guarantee:` の行)と PR の本文は、この run の範囲に含めていない(`/pr`)。walkthrough は書いていない(`/pr` が決める)。差分は `git diff a16d7c4f...HEAD --stat` で 31 ファイル、+5,251 / -89(HEAD `49cc80cb` の時点。テストと証跡と報告を含む)
- 計画の `PR created` は `/pr` が付ける

## Verification

コミットの前に、worktree のルートで打った。

- `gofmt -l internal/`: 出力なし。`go vet ./internal/org/...`: 出力なし(rc 0)
- `./scripts/run-verify.sh`: `==> All verifiers passed.`、rc 0(出力 4,925 行、`FAIL` を含む行は 0。ブランチの secret scan `a16d7c4f..49cc80cb against origin/main` は clean)。出力は scratchpad に保存した。名前の付いた証跡 `docs/evidence/verify-2026-10-14-113230.log` は gitignore の対象で、コミットしない(手元のコピーだけ)
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 13 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-guarantee.md`: `d2923a8bf359`(承認時の digest と同じ)
- tech-debt の 226 行目の形の検査(scratchpad の `tdcheck.py`。`git show HEAD:` との比較): 変わった行は 226 行目だけ、`|` で分けた断片は 7 つ(= 5 列、`HEAD` と同じ)、列ごとの backtick は偶数、`file:line` の参照は 0、「Stage 7c-2」は残っていない
- 編集前に、書き換える文字列が各ファイルに 1 回だけあることを `anchors.py` で数えた(17 個すべて 1)。編集は Edit だけで行った(`sed -i` や `gofmt -w` は使っていない)
- insight の追記のあと `git status --short` に新しい未追跡のファイルはなく、既存の jsonl が `M` になっただけ

## Insight event

`./scripts/insights-append.sh --slug org-merge-guarantee --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で、`docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` に 1 行足した(cycle 1)。この jsonl だけの commit にした。

## Cycle 2

- Date: 2026-10-14(JST)
- Plan: `docs/plans/active/2026-10-14-org-merge-guarantee.md`(承認済み。`./scripts/plan-visual.sh digest` は `d2923a8bf359` のままで、`- Approved:` の行と同じ。この run が変えたのは `## Progress checklist` の 1 行の追加だけ)
- Pipeline cycle: 2(`--cycle auto`。insight event は cycle 2。既定の上限 2 の最後の回)。差分の base は `a16d7c4f`、branch は `feat/org-merge-guarantee`。run 開始時の HEAD は `99508d81`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report(どれも `## Cycle 2` の節): self-review(`432e5912`。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4(C2-1 から C2-4)、Merge: yes)、verify(`a99faa87`。pass、LOW 2(D1、D2)、情報 5)、test(`6a91a616`。pass、失敗 0、新しい穴 9 から 13)
- cycle 2 で Go を変えたのは `59abeaec`(`internal/org/mergeguarantee.go` とそのテスト)だけ。この run は Go のコード、テスト、script、skill、rule を変えていない
- 項目の番号: C2-1 から C2-4 は self-review の Cycle 2 の所見の ID、D1 と D2 は verify の Cycle 2 の「文書のずれ」の ID。Gap 9 から 11 は test の Cycle 2 の Test gaps の番号で、Gap 2 と 3 は cycle 1 のときの番号のまま(test の Cycle 2 の表も同じ番号を使う)
- 内容の commit: この report、仕様、tech-debt、計画の Progress checklist を含む 1 つの commit(`docs: add cycle-2 sync-docs report for org-merge-guarantee`)。insight event の jsonl は次の 1 commit に分けた(`docs: record the cycle-2 sync-docs insight for org-merge-guarantee`)

### Summary

cycle 2 の self-review、verify、test が出した文書側の指摘を直した。コードが要る LOW は直さず、tech-debt の 244 行目に足した(cycle 2 は上限の最後の回で、直すと 3 回目の pipeline が要る)。

- D1(verify)と C2-3(self-review): 仕様 FR-7 の 7c-2 の注(74 行目)の括弧が、403 を admin でない人への答えとして並べていた。コードの Detail は 403 では admin のないことを原因の 1 つとしか言わないので、括弧だけを直した
- D2 と C2-4(self-review): 244 行目の Related の「Test gaps 4 and 6」を中身で指す形にし、Why deferred の「cycle 2 of the pipeline, capped at 2」から数を外した
- C2-1 と C2-2(self-review)を 244 行目の (h) と (i) に足した。test の Gap 11 は (i) に、Gap 2、3、9、10 は (g) に足した。行頭の文は (h) と (i) の分類も言うようにした
- verify の AC2 の注: 計画の Progress checklist に 1 行で記録した。digest は変わらない

ほかの面(README、AGENTS.md、docs/recipes、`/org` skill の 4 面、org runtime の仕様)に、403 を admin のないことと結ぶ文や、旧い 403 の Detail を引く文はなかったので、変えていない。

### Changes made

| File | 項目 | Change |
|------|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の「7c-2 で決めたこと」の注、74 行目) | D1、C2-3 | 括弧だけを直した: 「(admin でない人への 404 `Not Found`、403)」を「(admin でない人への 404 `Not Found`、または admin のないことのほかに rate limit や SSO の承認でも返りうる 403)」にした。行のほかの部分は変えていない(`git diff --numstat` は 1 / 1)。self-review が勧めた言い方(「admin のない人への 404 `Not Found`、または 403(admin のないことは原因の 1 つ)」)ではなく、lead の指示の言い方を使った。どちらも 403 を admin のない人への答えとは言わず、コードの Detail と合う |
| `docs/tech-debt/README.md` の 244 行目(Small leftovers of the org-merge-guarantee PR)、Debt の列 | C2-1、C2-2、Gap 2・3・9・10・11、C2-4 | 行頭の文に (h) と (i) の分類を足した((a)(b)(d)(e)(f) と (i) は動きを変えず、(g) はテストの穴、(h) は doctor の Detail の escape の穴)。(g) のあとに「Four more cases have no test」を足した: 各 call site の escape(制御文字を持つエラーで試すのは `readRepository` の 1 か所だけ。`readMergeRules` と `ghJSON` の gh の失敗、`readMergeClassicStrict` の 404 `Not Found`・403・ほかの失敗、`readMergeGuaranteeOrigin` の get-url の失敗は、生の `err.Error()` に戻しても落ちるテストがない。残り 5 か所は mutant を作っていない)、403 と 404 `Not Found` の Detail の検査の数、classic の strict true、merge queue と classic の必須の検査の組み合わせ(最後の 2 つは偽の gh の seam だけ)。(h) は C2-1(doctor は Detail をそのまま出し、escape するのは gh と git の error text だけ。cache の書き込みと削除のエラー、state dir の path、origin の host と owner/repo は生で残る。`mergeGuaranteeResult` の最後の `PrintableInboxText` 1 回で閉じる)。(i) は C2-2(`readMergeClassicStrict` の doc の「GitHub answers 403 without admin access」は観測されていない)と Gap 11(strict の endpoint の 403 と 404 `Branch not found` は実物で見ていない)。呼び出し箇所は行番号でなく関数名で書いた(登録簿の方針) |
| 同じ行の Impact の列 | 同上 | (g) に、call site の Detail が生の `err.Error()` に戻ること、検査の数を 1 に決め打ちしても気づかないこと、classic の strict true と merge queue が偽の gh でしか動いていないことを足した。(h) は origin の URL、state dir の path、cache の書き込みの OS のエラーの ESC が doctor の行に生で出ること(status と merge-order は cache から読んだ Detail を escape する)。(i) は読む人がどちらの答えか分からないことと、403 と `Branch not found` の 404 のテストの行が、実物で出たことのない答えを使っていること |
| 同じ行の Why deferred の列 | C2-4(a)、同上 | 「(cycle 2 of the pipeline, capped at 2)」を「(the last pipeline run)」にした。(g) には、call site の escape と検査の数が last run の /test で見つかったこと(/test はテストを変えない。probe の行 PC1 から PC4 は HEAD で通り、mutation の evidence にある)、classic の strict true に admin の権限が要ること、merge queue と classic の必須の検査の例になる公開の repository がないことを足した。(h) は出どころがローカルで LOW、直す 1 行は code の変更で、最後の回に re-run が残っていなかったこと。(i) は同じ理由 |
| 同じ行の Trigger の列 | 同上 | (g) に `readMergeClassicStrict`(403 と 404 `Not Found` の Detail の検査の数も)、`mergeGuaranteeErrText`、`internal/org/mergeguarantee.go` に gh と git の手順を足すとき(call site の escape)を足した。(h) は `mergeGuaranteeResult` の次の編集か、doctor の行の制御文字の報告。(i) は `readMergeClassicStrict` の次の編集か、strict の endpoint が 403 を返す実物の読み |
| 同じ行の Related の列 | D2、C2-4(b) | 「(Test gaps 4 and 6)」をやめ、中身で指した(ruleset の空の context、`os.Stat` のエラー、`%2F`、偽の gh の seam でしか見ていない classic の strict true と merge queue。Cycle 2 は Test gaps 9、10、11)。self-review の「(L2, L5, L6)」に「Cycle 2: C2-1 and C2-2」を足した。`internal/cli/doctor_merge_guarantee.go`(`mergeGuaranteeResult`)、`mergeGuaranteeErrText`、`readMergeGuaranteeOrigin`、`mergeOriginFromURL` を足した。handoff にはなかったが、(g) を閉じる probe の行 PC1 から PC4 の置き場として `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-mutation.txt` も足した |
| `docs/plans/active/2026-10-14-org-merge-guarantee.md`(`## Progress checklist`) | verify の AC2 の注 | cycle 1 の cross-review の注の次、`PR created` の前に 1 行足した: cycle 2(`59abeaec`)で strict の 403 は `unknown` のままになり、Detail は admin のないことを原因の 1 つとしか言わない(Scope 30 行の書き方)。AC2 の括弧は 403 を admin でない人の答えとして並べるが、そう答える実例は観測しておらず、AC2 が求める結果は変わらないので、計画とのずれとはしない。残った LOW(C2-1、C2-2、test の Gap 2・3・9・10・11)を 244 行目の (h)、(i)、(g) に足したことも書いた。`## Progress checklist` の外は変えていない(digest は `d2923a8bf359` のまま) |
| `docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` | - | `sync_docs` の event を 1 行足す(`--cycle auto`、cycle 2、pass)。別の commit に入れる |
| この report | - | `## Cycle 2` の節を足した(cycle 1 の節は変えていない) |

244 行目は 4,911 bytes から 9,134 bytes になった(Debt の列 4,059、Impact 1,469、Why deferred 1,644、Trigger 958、Related 998)。5 列のまま。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`(127 行目の `ralph doctor` の行ほか) | 変更なし。「its `strict`, which needs admin access」は endpoint の要件を言う文で、403 の原因を言っていない。`403` の grep は 0 件 |
| `AGENTS.md`(repo map の `internal/org/` の行) | 変更なし。保証の句が 1 つあり、403 と admin の語はない |
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。`403` と `admin` の語は 0 件(`strict` は `secret-scan-branch.sh --strict` の 2 か所だけで、保証の話ではない)。`check-skill-sync.sh` と `check-sync.sh` は run-verify の中で通った |
| `docs/specs/2026-08-01-org-runtime.md`、`docs/recipes/` | 変更なし。保証、`required_status_checks`、`github_protection` の語は 0 件 |
| `docs/specs/2026-10-07-org-multi-org-director.md` の 74 行目以外 | 変更なし。旧い括弧の句(「`Not Found`、403)」)は 0 件になり、新しい句は 1 件ある |
| `docs/tech-debt/README.md` の 244 行目以外 | 変更なし。242 行目の「admin access」は `/protection` の admin の設定を読む話で、403 ではない。`git diff` で変わった行は 244 行目だけ |
| 計画の Scope 30 行と AC2(93 行) | Scope 30 行は「403 なら strict は分からない」と書き、cycle 2 の動きと合う。AC2 の括弧は 403 を admin のない人の答えとして並べたまま。AC2 は `/pr` の archive のあとの文書で、求める結果(`unknown`、`info`)は変わらないので直さず、Progress checklist に記録した |
| `internal/` の非 test の Go の `403` | `internal/org/mergeguarantee.go` の 4 行(433、434、467、469)だけで、cycle 2 の動きと同じ向き。433 行目からの doc コメントの「GitHub answers 403 without admin access」は未観測の主張で、C2-2 として 244 行目の (i) に足した。Go のコメントなので、この run は直していない |
| `docs/quality/`、`docs/architecture/`、`.claude/rules/ralph/` | 変更なし。`403` の語は 0 件。cycle 2 は `mergeguarantee.go` とそのテストだけを変えたので、これらが引く契約は動いていない |
| 証跡と報告の `$HOME` の置き換え | cycle 2 の evidence 5 本(`test-2026-10-14-org-merge-guarantee-c2-*.txt` の 4 本と `verify-2026-10-14-org-merge-guarantee-c2.md`)、この PR の self-review、verify、test の報告、この report で、`grep -c -E '/Users/[a-z]'` は 0 件(この節を足したあとの値) |

### Left open

- C2-1 の直し(`mergeGuaranteeResult` の最後で `PrintableInboxText` を 1 回通す。ESC を持つ `WriteErr` のテスト 1 本)、C2-2 の直し(`readMergeClassicStrict` の doc の 1 文)、test の Gap 9 と 10 を閉じる probe の行(PC1 から PC4)をテストに足すこと: どれも Go のコードかテストに触れる。cycle 2 は上限の最後の回なので直さず、244 行目の (h)、(i)、(g) に載せた。3 回目を回す理由が別にあるなら、self-review の Follow-ups の 2 のとおり先に直せる。その場合は (h)、(i)、(g) の call site の escape と検査の数を閉じ、244 行目の「(the last pipeline run)」と「the last pipeline run had no re-run left」の文(Why deferred の列、3 か所)も直す
- 計画の AC2 の括弧(403 を admin でない人の答えとして並べる)は、`/pr` が計画を archive するので直していない。Progress checklist に記録した
- strict の endpoint が本物の GitHub で 403 を返す実例と、404 `Branch not found` の実例は、まだない。admin でない token の読みは、ある branch もない branch も `Not Found` の 404 だった(test の evidence)。文書では直せない
- `protection/required_status_checks` が `%2F` を含む名前を受け付けるかは未確認のまま(admin の権限が要る。(g) に載っている)
- verify の情報の項目のうち、fix の commit message(`59abeaec`)の「so no raw control character reaches the terminal」が error text の部分の話であることは、commit message が文書でないので変えていない。C2-1 の (h) が閉じれば広い言い方も合う
- release のノートと PR の本文は、この run の範囲に含めていない(`/pr`)。walkthrough は書いていない(`/pr` が決める)。差分は `git diff a16d7c4f...HEAD --shortstat` で 38 ファイル、+6,922 / -90(HEAD `99508d81` の時点。うちテストを除く Go が +1,099 / -54、テストの Go が +2,533 / -26、報告と証跡が +3,001。この run の変更は含まない)
- 計画の `PR created` は `/pr` が付ける

### Verification

コミットの前に、worktree のルートで打った。`./scripts/run-verify.sh` は仕様、tech-debt、計画を編集したあとに走らせ、この節を含む report は run-verify のあとに書いた。

- `./scripts/run-verify.sh`: `==> All verifiers passed.`、rc 0(出力 4,925 行。`FAIL: [1-9]` を含む行は 0。ブランチの secret scan `a16d7c4f..99508d81 against origin/main` は clean)。出力は scratchpad に保存した。名前の付いた証跡 `docs/evidence/verify-2026-10-14-135640.log` は gitignore の対象で、コミットしない(手元のコピーだけ)。run-verify の中の `check-sync.sh` は `PASS: all files in sync.`、`check-skill-sync.sh` は `13 skill(s) in lock-step`、tech-debt の plan 参照は `OK`
- `git diff --check`(report を書いたあと): 出力なし、rc 0。変えた 5 ファイルの U+FFFD は 0 個。`git diff --numstat` は、report 78 / 0(`## Cycle 2` の節だけの追加)、計画 1 / 0、仕様 1 / 1、tech-debt 1 / 1、jsonl 1 / 0
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-guarantee.md`: `d2923a8bf359`(承認時の digest と同じ)。計画の差分は `git diff --numstat` で 1 / 0(Progress checklist の 1 行の追加だけ)
- 仕様 74 行目: `git diff --numstat` は 1 / 1。新しい括弧の句(「または admin のないことのほかに rate limit や SSO の承認でも返りうる 403)ときは」)の `grep -c -F` は 1、旧い「`Not Found`、403)」は 0
- tech-debt の 244 行目の形の検査(scratchpad の `tdcheck.py`。`git show HEAD:` との比較): 変わった行は 244 行目だけ、`|` で分けた断片は 7 つ(= 5 列。`awk -F'|'` の NF も 7)、列ごとの backtick は偶数、`file:line` の参照は 0、「capped at 2」は 0、「(Test gaps 4 and 6)」は 0。(h) と (i) は Debt の列に 2 か所ずつ(行頭の文と本体)、Impact、Why deferred、Trigger に 1 か所ずつ。挙げたパスは全部存在する
- 編集の前に、書き換える文字列が README に 1 回だけあることを `anchors.py` で数えた(1 回目は 1 つが 2 回あり(`internal/org/mergeguarantee.go` の前の `; `)、1 つ前の文字列と合わせて直し、10 個すべて 1 にした)。編集は Edit だけで行った(`sed -i` や cp は使っていない)
- insight の追記のあと `git status --short` に新しい未追跡のファイルはなく、既存の jsonl が `M` になっただけ(`<date>-<date>-<slug>.jsonl` のような取り違えたファイルは出ていない)

### Insight event

`./scripts/insights-append.sh --slug org-merge-guarantee --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で、`docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` に 1 行足した(cycle 2、pass)。この jsonl だけの commit にした。
