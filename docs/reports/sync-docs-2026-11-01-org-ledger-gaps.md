# sync-docs report: org-ledger-gaps

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-ledger-gaps.md`(PR #290 の Known gaps と tech-debt 395〜400 行)
- Branch: `fix/org-ledger-gaps`。run 開始時の HEAD は `ea6adfdb`(main の #295 の取り込み)、base は origin/main `f6c844d1`
- Pipeline cycle: 1
- 先行 report: self-review(Merge: yes、LOW 10)、verify(pass、LOW 3 と INFO 2)、test(pass、`run-test.sh` rc 0)
- 制約: Go のファイル、README、`/org` と `/director` の skill、`internal/config/config.go` は変えていない。plan は AC11 の箱だけを `[x]` にした。digest は前後とも `338876ea047b`

## Summary

tech-debt の 395〜400 行を直した形に書き換えた(398・399・400 は全部払った行、395・396・397 は一部だけ払った行)。verify の V2 は AGENTS.md の 1 語、V3 は 5 行の Trigger の繰り越しとして書いた。自分で開いた行を掃いて、依頼の 5 行のほかに 3 行(148、187、411)の Trigger が当たっていたので、同じ形の注を足した。388 行には、この PR が穴を広げたことを 1 文足した。表の末尾に新しい行(420 行)を足した。

コード、help、skill の文は変えていないので、pipeline の再実行は要らない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 395〜400 行の書き換え、Trigger の繰り越しの注(135、148、187、386、390、391、392、411 行)、388 行の相互参照、末尾の新しい行(420 行)。下の表のとおり |
| `AGENTS.md` | Repo map の `internal/config/` の行。`LocalOrgStateDir` を `LocalOrgStateDirAt` に直した(1 語) |
| `docs/plans/active/2026-11-01-org-ledger-gaps.md` | AC11 の箱を `[x]`。Progress は lead の未コミットの変更(test の行と main 取り込みの 1 行)がそのまま同じコミットに入る |
| `docs/reports/sync-docs-2026-11-01-org-ledger-gaps.md` | この report |
| `docs/insights/events/2026-11-01-org-ledger-gaps.jsonl` | `sync_docs` の行(cycle 1、pass、findings 0)。`insights-append.sh --cycle auto` で足した。UTC の日付がまだ 2026-11-01 なので既存のファイルに入った |

## 6 行の扱い

| 行 | 扱い |
|----|------|
| 395(再実行のコマンド) | 一部払った。元の Debt・Impact・Trigger を `~~` で消し、S7(`e9b94433`)で払った中身を書いた(`orgRetryCommand`、`cleanupRetryCommand`、`HintStateDir`、`--branch` の有無を保つこと、`ledgerMoveStep` の例外)。残りは、cleanup と merge の文のうち別の verb を名指すもの。`ralph org merge-order` を名指す文が約 12(`internal/org/merge.go`・`merge_run.go`・`mergeenforce.go`・`cleanup.go`、`internal/cli/org_merge.go`・`org_cleanup.go`。2 つは `--all` つき)、`ralph org inbox` が `merge.go` に 1、`merge_wait.go` の `ralph org status --org-id` と `ralph org disband --org-id`。場所は grep で数えた。regate の文は `gateWording.again` の行の領分と書いた。新しい Trigger は、これらの文の次の変更 |
| 396(個人用のファイル) | (a) を S2(`d8e70e0f`)、(b) を S3(`f35b70a9`)で払ったとして `~~` で消した。(c) を足した。self-review の L3(`Load` の doc の理由句が S3 で偽になった)と、verify が挙げた `StateDir` の doc(`LocalOrgStateDir` を名指す)。Trigger は #269 の merge のあと。#269 が `config.go` を変えているので触らなかったと Why deferred に書いた |
| 397(断る条件の文) | 一部払った。Go の側は S6(`de78896f`)で払った(定数 `retiredOrgStateDirRefusalCondition`、doctor の warn と escalate の help、`TestRetiredOrgStateDirRefusalCondition`)。残りは README の `[org].state_dir` の段落と `/org` skill の 4 つの写しで、定数と結ぶテストがない。#269 が変えている最中なので触らなかった(plan の D4)。Trigger は「#269 の merge のあと、定数の語で書くか、違いを認めて行を消す」 |
| 398(`--strict`) | 全部払った。Debt・Impact・Trigger を `~~` で消し、`(RESOLVED 2026-11-01 in fix/org-ledger-gaps, S4, commit dddc8716 ...)` と書いた。テストは `TestDoctorHelp_StrictFlagHasNoValueName` |
| 399(2 つの leftover) | 全部払った(行の中にほかの項はない)。(a) は S9(`9756e03e`)、(b) は S5(`bc2e191e`)。(a) の頭が guard の「org:」でなく、コマンドのパスに guard と同じ語を続ける形(`resolveOrgLedger` の既存の形)であることを書いた。(b) は `gitroot.MainWorktree` が 1 回の呼び出しで git を 2 回起動するので、既定の層は 4 回から 2 回になった、と書いて言い過ぎを避けた |
| 400(`local_test.go` の 800 行) | 全部払った。S1(`93397d1f`)。1011 から 600 行。移した先の 2 つのファイルは `ea6adfdb` で 169 行と 314 行(S1 の時点では 262 行だったので、あとのスライスがテストを足した) |

行番号は 395〜400 のまま動いていない(上に行を足していない)。

## Trigger の繰り越し

依頼の 5 行(verify V3)。どれも `2026-11-01: fix/org-ledger-gaps hit this trigger with ... but did not pay it: ... Carried over to the next change that meets this trigger` の形で Trigger の末尾に足し、Related に plan を足した。行数は `wc -l` を HEAD で打った値。

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 135 | `doctor_codex_writable_root.go` の `checkCodexAgmsgWritableRoot` の doc(`3f15c3ff`)。1024 → 1025 行 | check と codex の config は変えていない。Scope は doctor の既定値の使い方(S3) |
| 386 | `org_merge_test.go` 1160 → 1207、`merge_run_test.go` 1107 → 1111、`cleanup_test.go` 1816 → 1817(`director_skill_sync_test.go` は 1223 のまま)。新しいテストは新しい 2 つのファイルへ | move-only の分割で、Non-goals(#290・#291 より前から 800 行を超えていたファイルの分割)に当たる |
| 390 | `cleanup.go` 1089 → 1119(`orgRetryCommand`・`cleanupCommand` を足し、`ledgerMoveStep`・`branchMovedSteps`・`checkWorktree`・`remove` を編集)、`org_cleanup_test.go` 1266 → 1280 | 同上 |
| 391 | `cleanup.go` の拒否の文(再実行のコマンドと `ledgerMoveStep`・`branchMovedSteps`・`checkWorktree`) | `recordGone` の文 (a) と `registrationAtRisk` のコメント (b) は変えていない |
| 392 | `statedir.go` 225 → 269(`defaultOrgStateDirBase`、`ledgerBaseDir`、`LedgerLocalConfigPath` を足し、`mustAbs` が `absPath` を呼ぶ)。`resolvedOrClean` の包みは残る | 呼び出しの改名は Scope の外 |

自分で掃いて見つけた行(`docs/tech-debt/README.md` の open な行の Trigger 欄と、Debt 欄にだけ変えたファイル名がある行を python で走査し、当たった行の Trigger を読んだ)。

| 行 | 当たったもの | 扱い |
|----|--------------|------|
| 148 | Trigger が「`ResolveOrgStateDir` の変更」「`MainWorktreeRoot`」を名指す。この PR は `ResolveOrgStateDir` の探索を 1 回にし、`MainWorktreeRoot` の中身を `ledgerBaseDir` へ移した | 注を足した。開いている部分(設定の探索)には触れていない。これまでの注と同じ「met that part and left the open part」の形 |
| 187 (d) | 「`withMainWorktreeOrgLimits` or `MainWorktreeRoot`」。`MainWorktreeRoot` の中身を移した | 注を足した。bare repository の linked worktree の CLI テストは足していない。足したのは `FeatureRepoRoot` を打つ別のテスト |
| 411 | 「The next change to any of the five」の 5 つのうち `internal/cli/doctor.go` が 1514 → 1519 行 | 注を足した。move-only の分割は Non-goals |
| 388 | `cleanupShellWord` も `shellQuoteIfNeeded` も変えていないので Trigger には当たらない。ただし L5 のとおり、この PR が `--state-dir` の値にも広げた | Trigger 欄に 1 文足し、新しい行の (b) を指した |

見て却下した候補は次のとおり。

- 142(`--strict` は secret-scan のもの)
- 144(Trigger は `doctor_codex_writable_root_test.go` と target directory の決め方。どちらも変えていない)
- 170(`guardLegacyOrgStateDir` と `LegacyWorktreeStateDir` は変えていない。変えたのは `errOrgLedgerUnresolved` のコメントだけ)
- 186(`spawn.go` は `absPath` のコメントだけ)
- 197 (l)(`shellQuoteIfNeeded` の本体は変えていない)
- 198 (j)(`LedgerMainWorktreeRoot` は変えていない)
- 203(g)(h) は消してある
- 208(`newOrgRuntimeAt` を変えていない)
- 246(`org_merge_order.go` を変えていない)
- 266(cli の `absPath` を変えていない)
- 272(`checkStartAgmsgHome` を変えていない。`agmsgHomeIsRelative` の中身だけ)
- 294(`isolateGitEnv` は `internal/org` のもの)
- 361・362・378(`spawn.go` は `absPath` のコメントの 2 行だけで、`Send`・失敗の道・`reject` は変えていない)
- 388(上のとおり注のみ)
- 403(`cleanupShellWord` は使っただけ)

## 新しい行(420 行)

「Small leftovers of org-ledger-gaps」。(a) verify V1(`ralph org cleanup --help` が「the reason names that command」と言うが、`--branch` なしの run の理由は `--branch` なしを名指す。どちらを打っても片付けは進む)。(b) self-review L5(`cleanupShellWord` と `shellQuoteIfNeeded` の綴りの差。この PR で `--state-dir` の値にも広がった。388 行と同じ穴なので、388 行の Trigger に注を足し、この行の Trigger は 388 行のものを指す)。(c) `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` が `fx.work` のパスに空白があると落ちる。(d) test gaps: `cleanupRun.run` の `st.notDir` の拒否と `cleanupRun.remove` の `git worktree remove` の失敗、Linux の `os.Getwd` の失敗(差し替えで確かめた)。test report の Test gaps 2(`LedgerLocalConfigPath` が false のときの守り)と 4(実物の binary は main checkout の root からだけ)も、同じ test gaps の項なので (d) に入れた(依頼は 1 と 3 だけ)。Impact は LOW。Related は plan と 3 つの報告。関数名は書き、行番号は書いていない。

## AGENTS.md の変更

前: `and `LocalOrgStateDir` (the org ledger's place for `ResolveOrgStateDir`)`
後: `and `LocalOrgStateDirAt` (the org ledger's place for `ResolveOrgStateDir`)`

ほかの語は変えていない。`ResolveOrgStateDir` が呼ぶのは `LocalOrgStateDirAt`(`internal/org/statedir.go`)で、`LocalOrgStateDir` を呼ぶ本番のコードはもうない。`homepath.go` の 2 つの関数は足していない(verify が任意とした)。

## 触らなかったもの

| 対象 | 理由と、見つけた食い違い |
|------|--------------------------|
| `README.md` | #269 が変えている最中。verify の I1(L288・L290 の cleanup と merge の段落が、再実行のコマンドが `--state-dir` を含むことに触れない)は、`--help` に任せる要約なので偽ではない。S4 が回した exit 3 の段落は、L135 がもとから local-only のキーを名指していて直す所がない。`[org].state_dir` の段落(L148)と定数の言い方の差は 397 行に残した |
| `/org`・`/director` の skill | 変えると 4 つの写しと `sync-skills.sh` が要るので止める条件に当たる。変える必要も見つからなかった(verify が `.claude/skills/org/SKILL.md` と `.claude/skills/director/SKILL.md` の打ち直しの文を読んで合うとした。この pass では grep で `cleanup --branch` と `--state-dir` の出現を確かめただけで、段落は読み直していない)。`/org` の `RALPH_ORG_STATE_DIR` の段落は 397 行に残した |
| `internal/config/config.go` | #269 が同じファイルを変えている。`Load` の doc の理由句と `StateDir` の doc の `LocalOrgStateDir` は 396 行の (c) に書いた |
| `docs/specs/2026-10-07-org-multi-org-director.md`(L137、#263 で決めたことの段落) | 台帳を動かしたあとの打ち直しで「`--state-dir` か `RALPH_ORG_STATE_DIR` で新しい場所を指す」と書いたままで、変数は #290 で撤去された。この diff の前からの食い違いで、`ledgerMoveStep` の文は #290 で `--state-dir` か `[org].state_dir` になっている。決定の記録の段落なので触らず、lead に回す(1 句の直し) |

grep で見て、この diff の振る舞いと食い違わなかったもの: `docs/specs/2026-08-01-org-runtime.md`(`org cleanup`・`again`・`RALPH_ORG_STATE_DIR`・`ralph.local.toml`・`state_dir` の出現なし)、`.claude/rules/ralph/`(`model-routing.md` の「org state-dir precedence」の 1 文だけで、今も合う)、`docs/recipes/worktrees.md`(L57 は `--state-dir` か `[org].state_dir` と書いていて合う)、`docs/recipes/director-launch-cmd.md`(stop のコマンドが `--state-dir` を持つ話で、無関係)、`docs/quality/`(`exits 3` は secret-scan の話)、`docs/architecture/repo-map.md`(`internal/config` の項目がない)。

## 確かめた結果

- `./scripts/run-static-verify.sh`: rc 0。変更した言語は golang で、gofmt `ok`、`0 issues.`、shellcheck と `sh -n`、check-sync(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本一致)、check-template-purity、tech-debt の計画参照の検査はすべて OK。branch secret scan は `f6c844d1..ea6adfdb` で clean。ログは `docs/evidence/verify-2026-11-01-154940.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)
- tech-debt の構造: `git show HEAD:docs/tech-debt/README.md` と行ごとに比べた。変わった行は 135、148、187、386、388、390、391、392、395〜400、411、420(新しい行)だけ。どの行も 5 欄に割れ、欄ごとの backtick と `~~` が偶数、`|` の数が変わらず、`file:line` の参照はない
- plan の digest: `./scripts/plan-visual.sh digest` は `338876ea047b`(承認時と同じ)
- `git status --short`(コミット前): `AGENTS.md`、`docs/insights/events/2026-11-01-org-ledger-gaps.jsonl`、`docs/plans/active/2026-11-01-org-ledger-gaps.md`、`docs/tech-debt/README.md` が `M`。迷子のファイルはない(slug は日付なしの `org-ledger-gaps`)

## lead に確認したいこと

- plan の Risks は「tech-debt の表は中ほどの行を書き換えるだけで、末尾には足さない」と書いているが、依頼は末尾に 1 行足すことだったので、依頼に従った。#269 や #277 の branch も末尾に足すなら、その部分が衝突する
- 上の `docs/specs/2026-10-07-org-multi-org-director.md` の 1 句を直すか
- 395 行の残り(別の verb を名指す約 12 の文)は、cross-review で再実行になる直しが出たら、一緒に直すこともできる(文の数が少ない)。出なければ Known gaps に書く

## lead の判断

- tech-debt の表の末尾の 1 行は、私の依頼どおりで残す。計画の Risks の文と食い違うが、衝突したら両方の行を残して直す(ユーザーの指示どおり)。
- `docs/specs/2026-10-07-org-multi-org-director.md` は直した(別のコミット)。137 行は、この PR が変えた台帳を動かす案内(`ledgerMoveStep`)と同じ中身なので、`--state-dir` か `ralph.local.toml` の `[org].state_dir` で新しい場所を指し、元の `--state-dir` は付けない形にした。88 行(2 か所)と 146 行は 4 段目と 8a-1 の決定の記録で、決めた当時の形を残し、`RALPH_ORG_STATE_DIR` のあとに「(PR #290 で撤去)」とだけ足した。
- 395 行の残りは、cross-review の結果を見て決める。
