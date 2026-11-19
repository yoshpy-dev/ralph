# sync-docs report: org-ssh-alias-host

- Date: 2026-11-20(branch の作業日は 2026-11-19。tech-debt の注の日付は 2026-11-19 にそろえた)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)。承認の digest `f4a9803ca80e` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない(Progress だけ変えた)
- Branch: `fix/org-ssh-alias-host`。run 開始時の HEAD は `744a66c7`。base は merge-base `e6d5d0ed`。origin/main は `32274500` まで進んでいる(#348、#349)が、取り込んでいない
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は `a77538ab`、verify(2 回目、pass)は `b01d04c1`、test(pass)は `63e1bc72`。引いた commit は各 report を含む最後の commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。docs の commit `5f2f0a0e` のあと、この report を足す前): `git diff --shortstat origin/main...HEAD` は 35 ファイル、+2,343/-194。Go は 22 ファイル、+1,698/-174
- 制約: Go のコードとテストは変えていない。`./scripts/run-test.sh` と Go のテストは回していない(下の「確認」)

## Summary

verify が渡した D1、D2、D5 を直した。D4(計画の Progress)は `6a17dfad` で済んでいた。

- D1: README の doctor の行の括弧を、`mergeSSHHostName` の今の規則に合わせた
- D2: Trigger に当たった tech-debt の開いた行 8 つに carry-over の注を足し、`merge_test.go` の大きさの行を 1 つ足した。8 つのうち 250 は verify の一覧になく、この走査で見つけた
- D5: evidence の「確かめていないこと」にテストの新しいファイル名を足し、binary が `7798b966` のものであることと、そのあとの 2 回の打ち直しを足した
- 計画の Progress: test と sync-docs の行を足し、artifact の 3 行に `[x]` を付けた。`PR created` は `/pr` が付ける

## Changes made

| File | Change |
|------|--------|
| `README.md` | doctor の行(137 行目)の括弧 1 か所 |
| `docs/evidence/ssh-alias-host-livefire-2026-11-19.md` | 冒頭に追記の段落、「確かめていないこと」にテストのファイル名 |
| `docs/tech-debt/README.md` | 250、251、258、310、394、409、419、501 行目に注(Trigger のセルの末尾と、Related のセルの末尾)。532 行目に新しい行。`git diff` は 9/8 の行の差。ほかの行は変わっていない |
| `docs/plans/active/2026-11-19-org-ssh-alias-host.md` | Progress(Test、Sync-docs、artifact の 3 行) |
| `docs/reports/sync-docs-2026-11-19-org-ssh-alias-host.md` | この report |
| `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` | sync_docs の 1 行 |

## D1: README の doctor の行

変えた括弧は次のとおり。

- 前: `(it fails, runs out, or prints no hostname that `gh` can take)`
- 後: `(it is not run when origin's host is not a plain host name, or it fails, runs out, or prints no hostname or one that is not a plain host name; a plain host name is made only of ASCII letters, digits, `.`, `_`, `:` and `-`, and does not start with `-`)`

コードと突き合わせた。`resolveMergeAPIHost`(`internal/org/mergeorigin.go`)は、origin の host が `mergeSSHHostName` を通らないとき `ssh -G` を打たずに理由を返す。打ったあとも、hostname の行がない、空、`mergeSSHHostName` を通らない、のどれかで origin の host のまま進む。gh は文字を確かめず、断るのは ralph である。行のほかの部分は変えていない。同じ文は README の 296 行目の merge の段落にはない(そこは「ssh -G で解く」とだけ言い、今も合う)。`git grep` で、この句は repo の中に README の 1 か所だけだった(report と計画は除く)。

## D2: tech-debt の Trigger の走査

走査は 3 本の python(scratch)で行った。tech-debt の全行から、Trigger のセルが全部 `~~` の行を除き、(1) この branch が変えたファイルの名前と、変えた Go の関数の名前、(2) 変えた行にある 8 文字以上の識別子のうち Trigger に書かれたもの、(3) 「any of」「the five」「next touch」などの集合の語がある Trigger のうち、Debt のセルが変えたファイルの名前を持つ行、を拾った。そのあと、拾った行の Trigger を 1 行ずつ読み、変えた関数の hunk を読んだ。

### 当たった行(注を足した)

| 行 | 当たったもの | 注の中身 |
|---|---|---|
| 250 | 「the next PR that adds an external command to `internal/org`」。この PR は `ssh -G` を足した(**verify の一覧にない**) | 新しい外部コマンドを足したが、runner は足していない。`realMergeDetachedExec` を通し(cleanup は nil で、同じになる)、呼び出しの期限 `mergeSSHResolveTimeout`(5 s)だけ持つ。runner の統合は払っていない(計画の D2) |
| 251 | 「or their tests」。`internal/org/mergeorder_test.go` 3281 → 3291 行(`moExec` の fake に `ssh` の case) | 字面どおりに当たった。lettered item はどれも Scope の外。`mergeOriginRepo.check` は host を比べないので、計画の Non-goals どおり別名は届かない |
| 258 | 「The next edit to `internal/org/mergeguarantee.go`」。877 → 937 行 | (a) の分割は払っていない。(b)、(d) はそのまま(get-url の読み手は増えていない)。(g) を 1 か所広げた: `resolveMergeAPIHost` が ssh の error 文を `mergeGuaranteeErrText` に通すが、制御文字を持つ error を渡すテストがない(`mergeorigin_test.go` の制御文字は host と hostname で、丸ごと断られる)。mutant を入れていない |
| 310 | `mergeguarantee.go` とそのテストの部分。877 → 937、`mergeguarantee_test.go` 1334 → 1396 | 払っていない。新しい doctor のテストは新しいファイル `mergeguarantee_alias_test.go`(418 行)に入れた。`mergeorder.go` は触っていないので、もう半分は当たっていない。行の頭の「877」「1334」は元の計画の時点の数で、前の branch の注も直していない |
| 394 | 4 ファイルのうち 1 つ。`internal/cli/org_merge_test.go` 1243 → 1248 行(`TestOrgMerge_Help` の 1 断片) | 払っていない。残り 3 つは触っていない(1158、1318、1817 行のまま) |
| 409 | 3 ファイルのすべて。`regate.go` 1058 → 1061(`RegateParams.Exec` の doc)、`regate_ci.go` 901 → 913(`sshRun`、`newRegateCI`)、`org_regate_test.go` 1335 → 1339(help の pin) | 払っていない。ci の新しいテストは `regate_ci_ssh_alias_test.go`(82 行)に入れたので、`regate_ci_test.go` は 767 行のまま |
| 419 | 「the next change to any of the five」。`internal/cli/doctor.go` 1651 → 1653(`runDoctorOpts` の doc comment。コミット `3c369ff4`、2 回目の self-review の L8) | 払っていない。comment だけで、計画の Scope は `doctor.go` を名指ししていない。ほかの 4 ファイルは触っていない |
| 501 | 「a gate, regate, or merge test of `internal/cli`」の字面。`TestOrgMerge_Help`、`TestOrgRegate_Help` の pin | 判断: 当たったとして注を足した。help のテストは gate も probe も走らせないので、この行が言う費用は増えていない。`gateToolsProbe` の pin は Scope の外 |

各行の Related の列の末尾に `docs/plans/active/2026-11-19-org-ssh-alias-host.md` を足した。`archive/` は `/pr` の `archive-plan.sh` が書き換えるので先には書かない。行数は merge-base `e6d5d0ed` と HEAD の `wc -l`(この pass で数えた)で、verify の表と全部一致した。

### 新しい行

532 行目に `internal/org/merge_test.go`(834 行、base で 819 行)の大きさの行を足した。verify が「行を足すかは `/sync-docs` の判断」と渡したもの。この branch は既存のテストを 15 行直しただけで、新しいテストは `merge_ssh_alias_test.go` に入れた。行の Trigger は、次の変更で新しいテストを新しいファイルに入れることと、動かすなら末尾の 2 つの `TestMergeOrigin_*`(約 95 行)を `mergeorigin_test.go` に移すこと。309 行目の RESOLVED の註(`merge_test.go` が 626 行だった時点の決まり)に合わせた。

### 当たっていない行

| 行 | 理由 |
|---|---|
| 248 | Trigger は報告か 8a の director。この PR は merge-order の `mergeOriginRepo.check` を触っていない(計画の Non-goals)。Trigger の本文の「`readOrigin` of `internal/org/merge.go`」は base から `ReadMergeOrigin` に任せる形で、host が解いた値になった点だけ古いが、結論(「this gap does not reach a merge」)は今も合う。直していない |
| 253、296、127、328 | 打ち消しの行(RESOLVED の行)。253 はこの PR が閉じたもの。296 の `regate.go` の分割は `regate_git.go` で済んでいる |
| 254 | この PR が足した director skill の行。計画のパスは active のまま(`archive-plan.sh` が `/pr` で書き換える) |
| 255、256、257 | Trigger は報告か 8a の director、`MergeGuaranteeVersion` の変更。版は上げていない。`mergeenforce.go` は comment だけ |
| 308 | Trigger の `mergeLsRemoteRefusal`、`mergeGHSaid`、`realMergeOrderExec` の error は触っていない(`merge.go` の hunk は `ghAccess`、`ghMissing`、`readOrigin`、`mergeReady`、`MergeParams`) |
| 311、315、410、245 | 報告か gh の変更が Trigger。`listOriginBranches` は触っていない |
| 403 | 前半の `cleanupOriginRefusal`、`MergeParams` は `~~` で消えている(done)。生きている後半の `mergeRegateCommand`、`mergeGateCommand`、merge-order の Detail、merge の not-ready の断りは触っていない。`cleanupOriginRefusal` の中断の文は変わった(この PR)が、前半の Trigger |
| 411 | `regateCIPreCheck` の本文は変わったが、変わったのは `ReadMergeOrigin` の引数、`newRegateCI` の取り出し、`regateCIInterrupted` に渡す step の語で、中断の error の文。Trigger の `gateWording` と「refusal texts」(`again` で終わる断り)は触っていない。`regateCIInterrupted` の文に `again` の終わりはない |
| 182 | 「`/org` skill の command table の次の編集」。skill の変更は 2000 行目の箇条書きの 1 文で、table(393〜398 行目)の外 |
| 241、381、511、242 | skill の別の節、signal の処理、`gateSignalContext` が Trigger。help の文だけ変えた `org_merge.go`、`org_regate.go` は signal を触っていない |
| 190、202、435、470、146 | `doctor.go` を Debt や note の文脈で名指すが、Trigger は別の関数(`checkManifestVersion`、`runDoctorFull` の対象の選び方など)。`runDoctorOpts` の comment は Trigger の外 |
| 88、175、301、297、386、422 | Trigger が別のファイルか、前の branch の note の中に名前があるだけ |
| 375、405、517、531 | README を名指すが、Trigger は別の段落(`[org].state_dir`、recipe の evidence の引用、`pack add`) |

## D5: evidence

- 「確かめていないこと」の fake のテストの一覧に `merge_ssh_alias_test.go` と `regate_ci_ssh_alias_test.go` を足し、別名の origin のテストは `da80502f` で移ったと書いた。`merge_test.go` と `regate_ci_test.go` には branch が直した既存のテストが残るので、名前は残した
- 冒頭に追記の段落を足した。「そのあとのコミットは文書だけ」は書いた時点のことで、`55cdb7c3` と `3c369ff4` がコードを変えた。verify の 1 回目が HEAD `0a71a930` の binary で、test が HEAD `6a17dfad` の binary で、同じ場面を打ち直し、この文書と同じ内容の行を見た(path と `checked_at` は違う)。指し先は `docs/reports/verify-2026-11-19-org-ssh-alias-host.md` の AC7 と Observational checks、`docs/reports/test-2026-11-19-org-ssh-alias-host.md` の「HEAD での実機の確かめ」
- 追記は、verify の 1 回目が打った場面(別名でない 2 つの origin を含む)と、test が打った場面(別名の origin の doctor、キャッシュ、`ralph status`、ssh を外した doctor)を分けて書いた。test は別名でない origin を打っていないので、「4 つの場面すべて」とは書いていない

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md`(92 行目の `internal/org/` の行) | 変えない。merge guarantee の説明(`mergeguarantee.go` が保護を読み `merge-guarantee.json` に種類を持つ)と `ReadMergeOrigin` の説明は今も合う。`api_host` はキャッシュの細部で、地図に書かない(計画の D5) |
| `README.md` の 296 行目(`ralph org merge` の段落) | 変えない。「ssh origin の host を `ssh -G` で解く」は今のコードと合う |
| org skill の 4 写し(2000〜2005 行目) | 変えない。`check-skill-sync.sh` は ok、`check-sync.sh` は PASS |
| `docs/specs/2026-10-07-org-multi-org-director.md`(7c-2、8a-3a、#263 の決定) | 変えない。7c-2 は「#340 で改めた」と書いてある。8a-3a と #263 の決定は日付つきの記録で、別名を解かないとは言っていない |
| `docs/quality/`、`docs/architecture/repo-map.md`、`.claude/rules/ralph/` | 変えない。merge guarantee、`api_host`、`ssh -G` の記述なし(`git grep`) |
| `CLAUDE.md` | 変えない |
| `.claude/skills/pr/SKILL.md` の 46 行 | 変えない。計画の Progress が PR の follow-up に回すと書いている |
| `internal/upgrade/notes/` | 変えない。release ノートは `/release` で書く。候補の文は計画の Rollout に書いてある |

## 直さなかったこと

- 計画の Scope の 21〜22 行目は、解いた値を断る規則を 1 回目の版のまま書く(空、`-` で始まる、空白・制御文字・`/` を含む)。今の規則(`A–Z a–z 0–9 . _ : -` だけ)は D6 と Risks が言う。Scope の本文は digest に入り、直すと承認を取り直すことになるので、直していない。D6 が「計画からのずれ」と書いている(verify の INFO)
- 行 248 の Trigger の本文の古い言い回し(上の表)
- テストのこと(`resolveMergeAPIHost` の `sshRun == nil` の既定、cleanup が nil を渡すこと、`mergeGuaranteeErrText` の ssh の call site の制御文字): 範囲外。test の Test gaps と tech-debt の 258 の注に書いた
- 行 310 の頭の「877 行」「1334 行」は、元の計画の時点の数で、前の branch の注も直していない。この branch の注に今の数を書いた

## 確認

- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0。`tech-debt README plan references` は OK(計画の参照は `docs/plans/active/2026-11-19-org-ssh-alias-host.md`)、`check-template-purity.sh` は PASS
- 計画の digest: `f4a9803ca80e`(Progress を変えたあとも同じ)
- tech-debt の編集: HEAD(編集前)と突き合わせ(`tdcheck.py`、scratch)。変わった行は 250、251、258、310、394、409、419、501 の 8 行と、新しい 532 行目だけ。どの行も 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、変わったセルは元の文の末尾に足しただけ(append only)、新しい `file:line` の参照なし
- evidence の lint(natural-japanese、`--json`): 警告 1 件(和欧文間の半角空白)。ファイルの元の書き方(3〜10 行目ほか)と同じなので、そろえて残した
- テスト(`run-test.sh`、Go のテスト)は回していない。この step が変えたのは文書だけで、Go のコードは変えていない

## Insight event

`./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` に足した。この branch は前の phase の行も同じファイルに commit している。
