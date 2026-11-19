# sync-docs report: org-ssh-alias-host

- Date: 2026-11-20 JST(UTC では 2026-11-19)。cycle 2
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)。承認の digest `f4a9803ca80e` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない(Progress に 5 行足しただけ)
- Branch: `fix/org-ssh-alias-host`。run 開始時の HEAD は `e6a98fad`(origin/main `32d6dc51`(#348、#349、#350)を取り込んだ merge)。base は `32d6dc51`
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2。`cap` の記録はなく、`[standard].max_pipeline_cycles` の既定 2 なので、この run が最後)
- 先行 report: self-review は `e07bdb0b`(Merge: yes、LOW 6)、verify は `fd76b7c0`(pass)、test は `530bf97c`(pass)、cross-review の triage は `79eb08dd`(ACTION_REQUIRED #1 は `240324fe` で直済み)。引いた commit は各 report を含む最後の commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。文書の commit `a42eb2d0` のあと、この report を足す前): `git diff --shortstat origin/main...HEAD` は 40 ファイル、+2,900/-216。Go は 25 ファイル、+2,058/-196
- 制約: Go のコードとテストは変えていない。`./scripts/run-test.sh` と Go のテストは回していない(下の「Cycle 2 の確認」)

## Cycle 2 の概要

verify が渡した D6、D7、D8 を直した(D1〜D5 は cycle 1)。

- D6: README の doctor の行の末尾に、Merge guarantee と Node toolchain の検査が signal で止まるときの動きを 1 文足した
- D7: tech-debt の「Five files went past the 800-line guide」の行(今は 420 行目)の、この branch の注を直した。`doctor.go` は 1651 → 1683 行で、変えた commit は `3c369ff4`(コメント)と `240324fe`(コード)の 2 つ。#341 の注はそのまま
- D8: 最後の run で残した LOW(self-review の L4、L6、L7、L9、L11、L12)と、test report の M12 を 1 行にして register の末尾(535 行目)に足した
- 計画の Progress: cycle 2 の self-review、verify、test、main の取り込み、この sync-docs の 5 行。`PR created` は `/pr` が付ける

## Cycle 2 で変えたもの

| File | Change |
|------|--------|
| `README.md` | doctor の行(137 行目)の末尾に 1 文 |
| `docs/tech-debt/README.md` | 420 行目の Trigger のセルの末尾(この branch の注)と Related のセルの末尾。535 行目に新しい行。変わったのはこの 2 行だけ(`git diff --numstat` は 2/1) |
| `docs/plans/active/2026-11-19-org-ssh-alias-host.md` | Progress に 5 行(`git diff --numstat` は 5/0) |
| `docs/reports/sync-docs-2026-11-19-org-ssh-alias-host.md` | この report。cycle 1 の report はこの下にそのまま残した |
| `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` | sync_docs の 1 行(cycle 2) |

文書の変更は 1 つの commit `a42eb2d0` に入れた。この report と event は次の commit。

## D6: README の doctor の行

足した文は次のとおり。

> A SIGINT, SIGTERM, or SIGHUP while the `Merge guarantee` check or the probe of the `Node toolchain` check runs stops that check, and the process group of its `ssh -G` or probe with it: doctor prints the lines so far and `Interrupted: the <check> check did not finish. Run ralph doctor again.`, runs no check after it, and exits 1 even when no line is `fail`; an interrupted `Merge guarantee` check is cached as `unknown` like any other `unknown`.

置いた場所は行の末尾(Merge guarantee の段落の最後の「nothing is cached.」の次)。doctor の行は Node toolchain の検査を説明していないので、この文が README で名前を出す最初の場所になる。書く前に、文の各部分をコードと突き合わせた(関数名で引く)。

| 文の部分 | コード |
|---|---|
| Merge guarantee の検査が signal で止まる | `runMergeGuaranteeCheck`(`doctor_merge_guarantee.go`)が `doctorMergeGuaranteeSignalContext`(`signal.NotifyContext` に SIGINT、SIGTERM、SIGHUP)の ctx で `org.CheckMergeGuarantee` を走らせ、`defer stop()` で検査が返るときに解く |
| Node toolchain は「probe」の間だけ | `probeNodeToolchain`(`doctor_node.go`)が `doctorNodeSignalContext` の下で probe を走らせる。`git ls-files` の層(`doctorNodeLayout`)は `context.Background()` なので signal を受けない。そのため「検査」でなく「検査の probe」と書いた |
| process group ごと止まる | ssh: `realMergeDetachedExec` が ctx の終了で group に SIGKILL を送る。Node の probe: `toolchain.Run` が group に SIGTERM、`KillGrace`(2 s)後に SIGKILL |
| そこまでの行と `Interrupted:` の行 | `printDoctorResults`(`doctor.go`)。Merge guarantee の割り込みでは、割り込みの unknown の行を含む。Node では warn の 1 行を含む |
| 後の検査を打たない | Merge guarantee の割り込みは `runDoctorStateDir` が Check 12 から 14 の前に `printDoctorResults` へ返す。Node は最後の検査なので、後に何もない。依頼の文は「runs no later check」だったが、Node では言うことがない文になるので「runs no check after it」にした |
| fail の行がなくても exit 1 | `printDoctorResults` は interrupt があれば、fail の行がなくても error を返す(README の「a `warn` never changes the exit code」に対する例外)。verify の probe が rc 1 を見た |
| 割り込みの Merge guarantee は unknown としてキャッシュされる | `checkMergeGuarantee` は interrupted のときも `SaveMergeGuarantee` を呼ぶ。verify の probe で `merge-guarantee.json` が割り込みの unknown になった。台帳の dir がなければ書かないのは、ほかの unknown と同じなので「like any other `unknown`」とした |

spec の 7c-2(`docs/specs/2026-10-07-org-multi-org-director.md`)の「`unknown` は `info` で doctor の終了コードを変えない」は、種類と状態の対応の文で、割り込みは種類でないので偽にならない。変えていない。

## D7: tech-debt の `doctor.go` の行(420 行目)

この branch の注だけを書き直した(Trigger のセルの末尾と、Related のセルの `docs/plans/active/...` の括弧)。#341 の注は 1 文字も変えていない。

数えた値は次のとおり(`git show <commit>:internal/cli/doctor.go` の行数)。

| 時点 | 行数 |
|---|---|
| origin/main `e6d5d0ed`(`32d6dc51` も同じ。2 つの間で `doctor.go` を変えた commit はない) | 1651 |
| `5f2f0a0e`(cycle 1 の sync-docs。cycle 1 の注が数えた値) | 1653 |
| `240324fe`(branch で `doctor.go` を最後に変えた commit)と HEAD | 1683 |

変えた commit は 2 つ。`3c369ff4` は +2 行(numstat 4/2)で、`runDoctorOpts` の上のコメント。`240324fe` は +30 行(numstat 39/9)で、`runDoctorStateDir` の早い return、`printDoctorResults` と型 `doctorInterrupt` への切り出し、`runDoctorOpts` の上と Check 11c のコメントの signal の 1 句。branch 全体では 1651 → 1683(numstat 42/10)。注には、origin/main の値と、「branch で `doctor.go` を最後に変えた commit」の値を、ほかの行の注と同じ形で書いた(merge-base という語は、main を取り込んで動いたので使わない)。

払わない理由も直した。前の注は「the edit is a comment」と言っていたが、`240324fe` はコードの変更で、計画の Progress が「計画からのずれ」として記録している(計画の Scope は `doctor.go` を名指ししない)。`240324fe` が変えたほかの 3 ファイルは 161、266、550 行で、800 行の目安の内側にあるので、注に書いた。

## D8: 残した LOW の行(535 行目)

cycle 2 の self-review report の「Tech debt identified」にある英語の行を使い、`docs/reports/` の report と HEAD のコードと突き合わせてから足した。変えた点は次のとおり。

| 項目 | 突き合わせの結果 | 行への反映 |
|---|---|---|
| (a) L4 | cycle 2 の report の L4 の表は、引数 7 つと `originURL, host string` の並び、`sshRun == nil` の既定が 2 か所、「was interrupted」の枝を本番の呼び手が読まないこと、`mergeSSHResolveRuns` が同じ形を足していること、の 4 点。行の元の文は 4 点目のかわりに「排除法で理由の文を決める」を書いていた。この点は cycle 1 の report(`a77538ab`)の L4 にあり、cycle 2 の report にはない。コードは、`ReadMergeOrigin` も doctor の `readOrigin` も ctx を先に見るので「was interrupted」の枝の文は捨てられ、`resolveMergeAPIHost` の `mergeSSHResolveRuns` の偽の枝は「host が普通の名前でない」と決めうつ形である | 4 点とも書き、排除法の点は cycle 1 の版にあると断った |
| (b) L6 | 元の文は「pgid の three-way test」。テストのコードは `pgid != pid` または `pgid == test` の 2 条件で、3 通りではない。`ps -o pgid=` は 3 つのファイルに 1 回ずつある | 「group が ssh 自身のもので、テストのでない」と書き直した |
| (c) L7 | 表の欄は 7 つ(name、origin、ssh、host、apiHost、sshRuns、detail)。`map[bool]int{true: 1, false: 0}[tc.sshRuns]` は `mergeguarantee_alias_test.go` に 2 か所 | そのまま |
| (d) L9 | `mergeSSHAliasHint` は `apiHost`、`originHost`、`why` を `%s` で出し、`why` の中の host は `%q` | そのまま |
| (e) L12 | `runMergeGuaranteeCheck` が検査の返りで ctx を見るので、最後の命令のあとの signal でも「did not finish」が出る | そのまま |
| (f) L11 | signal を送る手順が `doctor_merge_guarantee_signal_test.go` に 2 回。`signalSelf` は `org_gate_test.go` にある | そのまま |
| (g) M12(新しい) | test report の Test gaps 2 と変異 M12。`TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` は取った数と解いた数が doctor の返りまでにそろうことを見るので、`stop` を `printDoctorResults` まで持ち越す形は通る。その形では `--probe-models` の Check 12 などで受けた Ctrl-C が飲み込まれる | (g) として足し、Trigger に `runMergeGuaranteeCheck` と signal のテストの次の変更を足した。足す行は test report の案(包んだ `stop` が呼ばれた時点で、後の検査がまだ走っていないこと)にならった |

Impact の列には、(g) が「あとの編集がそう直したときだけ」効くことと、(e) は数命令の窓の話であることを書いた。Why deferred の列には、(e) を直すと Node の検査が共有する文と、そのテストの期待文が変わることを足した。Related は計画、cycle 2 の self-review report の L 番号、cycle 2 の test report の Test gaps 2 と M12。5 つのセル、バッククォートは列ごとに偶数、`file:line` の参照なし。

## Cycle 2 の Trigger の走査

verify は「当たったのは 419 行(今は 420 行)だけ」と渡した。自分でも走査した。

- 1 本目: 全行の Trigger のセルから `~~` で消した部分を除き、`doctor`、`signal`、`interrupt`、`NotifyContext`、`Ctrl`、`SIGINT`、`process group`、`Node toolchain`、`MergeGuarantee`、`doctor_node` などの語を探した。`next change to` を含めたので当たりが多い(多くは別の話)。当たった行から、doctor か signal の話になるものを読んだ
- 2 本目: `doctor.go`、`runDoctorStateDir`、`runDoctorFull`、`runDoctorOpts`、`printDoctor*`、`checkMergeGuarantee`、`doctor_merge_guarantee`、`doctorNodeSignalContext`、`checkNodeToolchain` の名前だけを探した

結果は verify と同じで、当たったのは 420 行だけだった。ほかに読んだ行と、当たらなかった理由は次のとおり。

| 行 | 理由 |
|---|---|
| 135、136、146、150 | Trigger は `doctor_codex_writable_root.go`、`doctor_shell_alias.go` の次の変更か、`runDoctorFull` の対象 dir の決め方。`240324fe` はどちらも変えていない |
| 241、243 | gate、merge、regate の signal の扱い(`signal.NotifyContext` から離れる、など)。doctor の seam は別で、その 3 つの command の扱いは変えていない |
| 256、257、259、311 | `mergeguarantee.go` の次の編集か、kind の読み方。`240324fe` は `internal/org` を変えていない |
| 260、471 | `checkCodexModelSlugs`、`checkManifestVersion`。`240324fe` はどちらも変えていない |
| 497、499、501、504 | Node toolchain の検査の Trigger(`doctor_node_test.go` の次の変更、probe の records など)。`240324fe` は `doctor_node.go` と `doctor_node_test.go` を変えておらず、`checkNodeToolchain` の呼び出し側(`runDoctorStateDir`)と印字だけが変わった |

cycle 1 の注を足した行(250、251、258、310、394、409、501、532 行目)は、`240324fe` が触ったファイル(`doctor.go`、`doctor_merge_guarantee*.go`)を数えていないので、注の数は変わらない。それらの行番号は main の取り込みで動いた(下の対応)。

## Cycle 2 で他に確かめた文書

| 文書 | 結果 |
|---|---|
| `docs/specs/2026-10-07-org-multi-org-director.md` の 7c-2 | 変えない(D6 の最後) |
| `AGENTS.md`、`CLAUDE.md`、`docs/quality/`、`docs/architecture/repo-map.md`、`.claude/rules/ralph/` | 変えない。doctor の signal の動きと `Interrupted:` を言う文書はない(`git grep`。`Interrupted: the` が当たるのは README、tech-debt、Go のコード、計画と report だけ) |
| org skill の 4 写し | 変えない。`check-skill-sync.sh` は ok、`check-sync.sh` は PASS |
| `internal/upgrade/notes/` | 変えない。release ノートは `/release` で書く。計画の Rollout の候補の文は ssh の別名のことだけで、doctor の signal は入っていない |

## Cycle 2 で直さなかったこと

- self-review の L11 と L12 は Go のコードかテストの直しになる。最後の run なので、直せば full pipeline の再走になる。D8 の行に入れた
- 計画の Scope の 21〜22 行目の古い文(cycle 1 と同じ。digest に入る)
- 行 248 の Trigger の古い言い回し(cycle 1 と同じ)
- cycle 1 の部分(この下)の文のうち、`doctor.go` を「1651 → 1653 行、変えた commit は `3c369ff4` だけ」と言う箇所は、D7 で上書きされた。cycle 1 の記録なので書き換えていない

## Cycle 2 の確認

- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0。`tech-debt README plan references` は OK、`check-template-purity.sh` は PASS
- 計画の digest: `f4a9803ca80e`(Progress を足したあとも同じ)
- tech-debt の編集: HEAD(編集前)と突き合わせた(scratch の `tdcheck.py`)。変わった行は 420 行目と、新しい 535 行目だけ。どちらも 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし。420 行目の Trigger のセルは、HEAD の文と 8,628 字目まで同じで(その先がこの branch の注)、Related のセルは 1,486 字目まで同じ
- natural-japanese の lint(この report、`--json`): 警告 2 種。和欧文間の半角空白は、cycle 1 の部分や repo のほかの report と同じ書き方なので、そろえて残した。文末コロンは 4 行のうち、表の前置きの 2 行(cycle 2 の部分)を句点の文に直した。残りの 2 行は `コマンド: 結果` の箇条書き(cycle 2 と cycle 1 に 1 行ずつ)で、report の形なので残した。`new.endswith(HEAD の report)` は True で、cycle 1 の report は 1 文字も変わっていない
- テスト(`run-test.sh`、Go のテスト)は回していない。この step が変えたのは文書だけで、Go のコードは変えていない

### 行番号の対応

cycle 1 の部分が引く tech-debt の行番号は、origin/main の取り込みで動いた。`5f2f0a0e` の行を行頭の文字で探した値は次のとおり。

| cycle 1 の行 | 今の行 | 行の頭 |
|---|---|---|
| 250 | 251 | `ralph org merge-order --json` cannot tell(RESOLVED) |
| 251 | 252 | Small leftovers of the org-merge-order PR |
| 258 | 259 | Small leftovers of the org-merge-guarantee PR |
| 310 | 311 | `internal/org/mergeguarantee.go` (877 lines |
| 394 | 395 | Four test files went past the 800-line guide |
| 409 | 410 | Three files went past the 800-line guide |
| 419 | 420 | Five files went past the 800-line guide |
| 501 | 502 | Test gap of feat/gate-tool-versions (#311) |
| 532 | 534 | `internal/org/merge_test.go` is 834 lines |

## Cycle 2 の insight event

`./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` に足す(UTC の日付が同じなので、既存のファイルに追記され、新しいファイルはできない)。結果は commit の前に `git status --short` で確かめる。

---

## Cycle 1 の記録

この節から下は cycle 1 の report で、本文は書き換えていない。D1〜D5、`5f2f0a0e` と `b4521a4c` の記録で、cycle 1 の verify と plan の Progress がここを引く。tech-debt の行番号は上の「行番号の対応」で今の番号に読み替える。

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
