# sync-docs report: org-ssh-alias-host

- Date: 2026-11-20 JST(UTC では 2026-11-19)。run 3(パイプラインの 3 回目の run)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)。承認の digest `f4a9803ca80e` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない(Progress に 3 行足しただけ)
- Branch: `fix/org-ssh-alias-host`。run 開始時の HEAD は `150057d0`。base は `32d6dc51`(`main` と `origin/main` も同じ)
- Pipeline run: 3 回目(`cycle-count.json` は `cycle` 2、`cap` 3。cap は cross-review の 2 回目で 2 から 3 に上げた。この run が cap 3 の最後)
- 先行 report: self-review は `ee9bae65`(LOW 9、Merge: yes。L13 と L15 は `74fef538` で直し、L14 は残した)、verify は `d70db328`(pass)、test は `db70487b`(pass)、cross-review の triage は `ebbbfd1c`(cycle 2 の ACTION_REQUIRED #1 は `664d4771` と `04161b43` で直済み)。引いた commit は各 report を含む最後の commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。内容の commit `783cd459` のあと、この report を足す前): `git diff --shortstat main...HEAD` は 41 ファイル、+3,285/-218。Go は 26 ファイル、+2,207/-197
- 制約: Go は doc comment 4 か所だけを変えた(コードの行は 0)。`./scripts/run-test.sh` と Go のテストは回していない(下の「Run 3 の確認」)
- この report は 3 つの run の記録を重ねている。run 3(この節、`---` まで)、cycle 2(`a42eb2d0`、その下)、cycle 1(`5f2f0a0e`、さらにその下)。下の 2 つは書き換えていない。verify と計画の Progress が引く D1〜D8 の記録はそちらにある

## Run 3 の概要

verify が渡した D9〜D15 を直した。D9〜D12 は tech-debt の register、D14 は Go の doc comment、D13 と D15 は INFO だったが、どちらも直した。

- D9: 「Small leftovers of fix/org-ssh-alias-host (#340)」の行(535 行目)を run 3 に合わせた。cap と cycle、引数の数、表の欄の数を直し、L14 と `mergeSSHHost` の枝の test gap を足し、この行自身の Trigger が `664d4771` と `74fef538` に当たったことの注を足した
- D10、D11: 252 行目と 311 行目の #340 の注にあった「`mergeorder.go` を触っていない」を、`664d4771` が触ったと直した。行数も数え直した
- D12: 423 行目に #340 の注を足した
- D13: 534 行目の `merge_test.go` を 835 行、足した行を 16 行に直した
- D14: `mergeguarantee.go` と `merge.go` の doc comment 4 か所(verify の 3 か所と、同じ file の header comment 1 か所)。`ssh -G` が受けるのは origin の URL に書かれたとおりの host で、キャッシュの `Host` と `originHost` はそれを小文字にしたものだと書いた
- D15: README の doctor の行と spec の 7c-2 に、キャッシュの `host` が小文字だと足した
- 計画の Progress: verify、test、この sync-docs の 3 行

## Run 3 で変えたもの

| File | Change |
|------|--------|
| `internal/org/mergeguarantee.go` | doc comment 3 か所(`MergeGuaranteeCheck.APIHost`、`CheckMergeGuarantee` の step 1、キャッシュが持つものを言う header comment)。937 → 940 行 |
| `internal/org/merge.go` | `mergeReady.originHost` のコメント。778 → 779 行 |
| `README.md` | doctor の行に「lowercased」(`git diff --numstat` は 1/1) |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 7c-2 の決定の 1 文に「小文字にしたもの」(1/1) |
| `docs/tech-debt/README.md` | 252、259、311、423、534、535 行目。変わったのはこの 6 行だけ(`git diff --numstat` は 6/6) |
| `docs/plans/active/2026-11-19-org-ssh-alias-host.md` | Progress に 3 行(3/0) |
| `docs/reports/sync-docs-2026-11-19-org-ssh-alias-host.md` | この report。下の 2 つの run の記録はそのまま残した |
| `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` | sync_docs の 1 行(run 3) |

内容の変更は 1 つの commit `783cd459` に入れた(`git show --numstat` の合計は 22 行足して 15 行消した)。header comment の直しを書いてから、まだ push していない最初の commit(`ec1f8ea9`)を `git commit --amend` で直して今の SHA になった。この report と event は次の commit。

## D14: Go の doc comment

verify は、計画の Verify plan が名指す `CheckMergeGuarantee` の doc の step 1 と、`MergeGuaranteeCheck.APIHost` の doc、`mergeReady.originHost` のコメントを挙げた。この 3 か所を直し、同じ file の先頭近くにある header comment の 1 文(「The cache keeps origin's host as its URL names it (host)」。README の D15 と同じ文型)も、同じずれなので直した。`ssh -G` に渡す host(書かれたとおり、大文字小文字を保つ。`mergeSSHHost`)と、キャッシュと `originHost` の値(小文字)を分けて書いた。

- `MergeGuaranteeCheck.APIHost`: 「the hostname `ssh -G <Host>` gave」を「the hostname `ssh -G` gave for origin's host as its URL writes it (`mergeSSHHost`; `Host` is that host lowercased)」にした
- `CheckMergeGuarantee` の step 1: 「read by `mergeOriginFromURL` (host and owner/repo)」を「host, lowercased, and owner/repo」にし、`ssh -G <host>` の `<host>` が書かれたとおりで、小文字ではないと足した。解けなかったときに残る origin の host も「(lowercased)」と書いた
- `mergeReady.originHost`: 「as its URL names it」に「lowercased (`ssh -G` got it as the URL writes it, `mergeSSHHost`)」を足した
- header comment(`mergeguarantee.go` の 40 行目付近): 「as its URL names it (host)」を「as its URL names it, lowercased (host)」にした。同じ comment の「the hostname `ssh -G <host>` gives」は、`<host>` を小文字と読む文ではないので、そのままにした

確かめたこと。

- コメントだけ: `git diff -U0 -- internal` の変更行から `//` で始まる行を除くと 0 行
- `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/` は通った(gofmt は出力なし、vet は出力なし)
- ほかの 2 か所は直さなかった。`MergeGuaranteeCheck.Host` の doc は元から「as written, lowercased」で、`MergeOrigin.originHost` の doc(`mergeorigin.go`)も「lowercased」と書いている
- 直さなかった 2 つの文(`mergeSSHAliasHint` と `aliasNote` が出す文)は、動きが変わるので register の (h) に回した(下)

## D9〜D12: tech-debt の register

行は内容で探した(行番号は verify の値と同じだったが、行頭の文字で確かめた)。535 行目は 3,845 字から 7,764 字、ほかの行は 252 行目 +439 字、259 行目 +463 字、311 行目 +443 字、423 行目 +689 字、534 行目 +153 字(run 開始時の `150057d0` との差)。5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし、生の `|` の数は元と同じ(行ごとに 6)。変わったのは上の 6 行だけで、HEAD(編集前)と突き合わせた(scratch の `tdcheck.py`)。

### 535 行目(D9)

| 項目 | 直したこと |
|---|---|
| 最初の文 | 「left in place at the last pipeline run (cycle 2 of 2)」を「run 3, at cap 3: the cross-review of cycle 2 raised the cap from 2 for this task」にした。Why deferred の「cap 2」も直した |
| (a) | 引数は 6 つで、`ctx, dir, sshRun, timeout, originURL, host` と書いた。`originURL, host` の組を取る関数は `664d4771` で 3 つ目(`mergeSSHHost`)が増えたことと、`readOrigin` が 3 つとも呼ぶことを足した。cycle 2 の report とこの行が「7 つ」と書いていた数え違いは、run 3 の self-review が数え直したと書いた |
| (c) | 表は 7 欄で、`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` のものは `664d4771` が `sshHost` を足して 8 欄(`origin, sshHost, ssh` と文字列の欄が 3 つ続く)。下の「verify の記述との差」を見る |
| (h)(新しい) | L14。`mergeSSHAliasHint` と `aliasNote` が小文字の host を出す。`git@Work-GH:o/r.git` の例と `GitHub.com.EMU` の例、`Host` の照合が大文字小文字を区別するので小文字の名前で探すと見つからないこと、表示だけの差であること、test report の Test gaps 4。3 つの doc comment はこの pass で直したと書いた |
| (i)(新しい) | `mergeSSHHost` の `!ok` か空の `written` の枝。呼び手 3 つ(`resolveMergeAPIHost`、`mergeSSHResolveRuns`、`readOrigin`)は同じ `originURL` から `host` を読み、`mergeOriginFromURLAsWritten` は空の host を断るので、今の呼び手は届かない。直接呼ぶ test もない(test report の Test gaps 3) |
| L13 と L15 | 残りには入れず、`74fef538` で直したと (i) の末尾に書いた(L13: `EqualFold` の条件を外した、L15: test のコメントが triage の cycle を名指す) |
| Impact、Why deferred | (h) と (i) の影響と、直さなかった理由を足した。Why deferred の「cross-review の 1 件」は「2 件(cycle ごとに 1 件)」にし、直した L13 と L15 を入れた |
| Trigger | (h) と (i) の払い方を足した。そのあとに、この行自身の Trigger が `664d4771` と `74fef538` に当たったことの注(下) |
| Related | self-review の L14 と L4 の数え直し、test report の Test gaps 3 と 4、計画の sync-docs の行(cycle 2 と run 3) |

Trigger の注は次の形にした。「この行を書いた PR 自身が、行のあとの commit で当たった。払っていない。2 つの commit は同じ PR の pipeline の中の finding の直しで、払うのはこの PR のあとの次の変更。(g) の test は直しの範囲の外」。verify の案のうち「同じ PR の中の変更だと注で言う」を取った。(a) から (g) の Trigger の文面は変えず、末尾を `Carried over to the next change that meets this trigger, after this PR` にした。`664d4771` が触ったのは `mergeSSHHost`、`mergeOriginFromURLAsWritten`、`mergeorigin_test.go` と `mergeguarantee_alias_test.go` の表と行。`74fef538` が触ったのは `mergeSSHHost`、`mergeorigin_test.go` の行、`doctor_merge_guarantee_signal_test.go` と `mergeguarantee_alias_test.go` のコメント(`git show 74fef538` で確かめた)。

### 252 行目、311 行目(D10、D11)

どちらの注にも、`mergeorder.go` を触っていないと書いてあった。`664d4771` は `mergeOriginFromURL` を `mergeOriginFromURLAsWritten` を包む薄い関数にして、1438 → 1447 行にした(`git diff 32d6dc51 HEAD -- internal/org/mergeorder.go` は 10 行足して 1 行消した。e6d5d0ed と 32d6dc51 で `mergeorder.go`、`mergeorder_test.go`、`mergeguarantee.go`、`mergeguarantee_test.go`、`merge_test.go` は行数が同じなので、注が引く e6d5d0ed の値のままにした)。

- 252: 「`mergeorder.go` and `internal/cli/org_merge_order.go` were not touched」を、`org_merge_order.go` は触っていない、`mergeorder.go` は `664d4771` が触った(1438 → 1447。`mergeRepoFromOriginURL` が呼ぶ `mergeOriginFromURL` が薄い関数になった。返す owner/repo は同じ)、と直した。払わなかった理由に、「`mergeorder.go` の変更は `ssh -G` に host の大文字小文字を渡すもので、merge-order が照合するものは変えない」と足した
- 311: 「`mergeorder.go` was not touched ... so the other part of this trigger did not fire」を、`664d4771` までは触っていなかった、`664d4771` で Trigger の `mergeorder.go` の半分に当たったが、runner(`realMergeOrderExec`、`realMergeDetachedExec`)は動かしていない、と直した。`mergeguarantee_test.go` は 1334 → 1398(注は 1396 だった。`664d4771` が 17 行足して 15 行消した)、`mergeguarantee_alias_test.go` は 446 行(注は 418 行)

### 259 行目と 311 行目の `mergeguarantee.go` の行数

verify は「877 → 937 行は HEAD でも合う」と渡した。doc comment を直したあとは 940 行なので、259 行目と 311 行目の「877 → 937」を「877 → 940」にし、「937 before that pass's doc-comment fixes」と書いた。259 行目には、`04161b43` と run 3 のこの pass(コメントだけ)がこの行の Trigger「The next edit to `internal/org/mergeguarantee.go`」に当たり、払っていないという 1 文も足した。後ろの半分は、この pass 自身の編集が当たった分で、verify の一覧にはない。

### 423 行目(D12)

Trigger のセルの末尾に #340 の注を足した。`664d4771` が `mergeorder.go` を 1438 → 1447 行にしたが、順位づけにも PR の照合の関数にも触らず、move-only の分割は Scope の外。計画の Non-goals は merge-order の PR の照合(`mergeOriginRepo.check`)を外している。Related には `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(Non-goals と注)を足した(`archive-plan.sh` が `/pr` で `archive/` に書き換えるので、先には書かない)。

### 534 行目(D13)

834 → 835 行、足した行 15 → 16。`git diff --numstat 32d6dc51 HEAD -- internal/org/merge_test.go` は 35 行足して 19 行消した(819 → 835)。`664d4771` は 4 行足して 3 行消した(1 行増)ので、「最後の 1 行は `664d4771` が 3 つの期待値に host の大文字小文字を入れたとき」と書き、「834 は最初の sync-docs の pass の値」と残した。

### verify の記述との差

- D9 の (3) は「`mergeorigin_test.go` の主な表は `fd76b7c0` で既に 8 欄」と言うが、その表(`name, url string`、`ssh`、`timeout`、`wantHost`、`wantWhy`、`wantSSH`)は 7 欄で、`fd76b7c0` でも同じだった(`git show fd76b7c0:internal/org/mergeorigin_test.go` で確かめた)。8 欄なのは `mergeguarantee_alias_test.go` の `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` だけで、同じ file の 135 行目の表(`name, origin, ssh, host, apiHost, sshRuns, detail`)は 7 欄のまま。(c) は「7 欄の表と、8 欄の 1 つ」と書いた
- D11 の「937 行は HEAD でも合う」は、この pass の doc comment のあとは 940 行(上)
- D12 で verify は「ranking と PR の lookup を動かしていない注を足す」と言った。実際に `mergeorder.go` へ入ったのは `mergeOriginFromURL` の薄い関数化だけで、ranking と lookup の関数は変えていない

## D15: README と spec

- README の doctor の行(137 行目): 「with origin's host as its URL names it (`host`)」を「with origin's host as its URL names it, lowercased (`host`)」にした。この句は repo の中に README の 1 か所だけだった(`git grep`。Go のコメントの 2 か所は D14 で直した)
- spec の 7c-2(`docs/specs/2026-10-07-org-multi-org-director.md`): 「キャッシュは origin に書かれた host(`host`)のまま」を「host(小文字にしたもの。`host`)のまま」にした。同じ段落の、照合が大文字小文字を無視するという文とは矛盾しない
- 計画の AC4 と Scope の 26 行目は digest に入るので触っていない(verify の D15 もそれでよいとした)

## Run 3 の Trigger の走査

verify は「当たったのは 252、259、311、423、534、535」と渡した。自分でも走査した。この pass 自身の編集(`mergeguarantee.go` と `merge.go` のコメント、README の doctor の行、spec の 7c-2)が当たる行も含めた。

- 1 本目: 全行の Trigger のセルから `~~` で消した部分を除き、触ったファイルと関数の名前(`mergeguarantee.go`、`merge.go`、`mergeorder.go`、`mergeorigin.go`、`mergeSSHHost`、`mergeOriginFromURL`、`mergeOriginFromURLAsWritten`、`resolveMergeAPIHost`、`ReadMergeOrigin`、`readOrigin`、`MergeGuaranteeCheck`、`CheckMergeGuarantee`、`APIHost`、`originHost`、`mergeSSHAliasHint`、`aliasNote`、`unresolvedWhy`、`ssh -G`、`merge guarantee`)と、`README.md`、`docs/specs` を探した
- 2 本目: 上の語から `comment` と `doc comment` を除いて、もう一度走らせた。1 本目だけに出た 76 行は、`comment` の語でしか当たらず、触ったファイルや関数の名前を Trigger に持たないので、1 行ずつは読んでいない

2 本目で当たったのは 19 行。読んだ結果は次のとおり(534 行目は Trigger に名前がなく、数を直すために別に開いた)。

| 行 | 結果 |
|---|---|
| 252、259、311、423、535 | 当たり。上の注を足した |
| 534 | Trigger(「more than an in-place fix」)には当たらない。数だけ直した |
| 248、249、256、257 | Trigger は merge-order の PR の照合と report、fork の checkout、director、保護を読む側の話。`merge.go`、`readOrigin`、「merge guarantee」は経緯の文にだけある。今回のコメントの直しは当たらない |
| 251 | 「adds an external command to `internal/org`」。run 3 は新しい外部コマンドを足していない。cycle 1 の注はそのまま |
| 254 | RESOLVED の行(1 つ目のセルだけが `~~`)。Trigger は ssh の別名の使い手と director の話で、今回の編集は当たらない。Trigger のセルを打ち消していないのは、この PR の slice 3 が書いた形のまま(251 行目も同じ形)で、触っていない |
| 317、241 | 317 の Trigger は `orgRunsOn` と merge-order の順位づけの変更で、`mergeorder.go` は 2026-11-12 の注の中にある。`664d4771` は順位づけを触らない。241 の Trigger は `gate.go`、`gate_run.go`、`lockfile.go` で、`mergeorder.go` は経緯の文にだけある |
| 410 | Trigger は `regate.go`、`regate_ci.go`、`org_regate_test.go`。run 3 は触っていない(`ReadMergeOrigin` は経緯の文) |
| 420 | Trigger は 5 ファイルの次の変更で、`doctor.go` などを名指す。run 3 は `doctor.go`、`doctor_merge_guarantee*.go` を触っていない |
| 179、376、406、518 | `docs/specs` と `README.md` は Trigger の中の語だが、179 は director の spec の「あとの段」を指す語、376 は recipe の引用、406 は `[org].state_dir` の段落、518 は別の PR の README。今回の 1 句の直しは当たらない |

## Run 3 で他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md`、`CLAUDE.md`、`docs/quality/`、`docs/architecture/repo-map.md`、`.claude/rules/ralph/` | 変えない。キャッシュの `host` の大文字小文字を言う文はない(`git grep`) |
| org skill の 4 写し | 変えない。`ssh -G <別名>` と書くだけで、小文字の host を受けるとは言わない。`check-skill-sync.sh` は ok、`check-sync.sh` は PASS |
| `docs/evidence/ssh-alias-host-livefire-2026-11-19.md` | 変えない。origin は小文字で、run 3 の直しの影響を受けない(verify の判断と同じ) |
| `internal/upgrade/notes/` | 変えない。release ノートは `/release` で書く |
| 計画の Scope の 21〜22 行目と、252 行目の Trigger の古い言い回し | cycle 1 と同じ理由で直していない(Scope は digest に入る) |

## Run 3 で直さなかったこと

- L14 のうち `mergeSSHAliasHint` と `aliasNote` が出す文は、`internal/` のコードと 2 つの test の期待を変えるので、register の (h) に回した。cap 3 の最後の run で、コードの直しは full pipeline の再走になる
- `mergeSSHHost` の枝の test gap(Test gaps 3)は register の (i) に入れた
- `.claude/skills/pr/SKILL.md` の 46 行目は、計画の Progress が PR の follow-up に回すと書いている

## Run 3 の確認

- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0。`tech-debt README plan references` は OK、`check-template-purity.sh` は PASS
- `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/`: 通った
- 計画の digest: `f4a9803ca80e`(Progress を足したあとも同じ)
- tech-debt の編集: 上の「D9〜D12」の冒頭。変わった行は 252、259、311、423、534、535 だけ
- テスト(`run-test.sh`、Go のテスト)は回していない。この step が変えた Go はコメントだけで、コードの行は変えていない(`git diff -U0 -- internal` の変更行はすべて `//` で始まる)

## Run 3 の insight event

`./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-org-ssh-alias-host.jsonl` に足した(UTC の日付が同じなので既存のファイルへの追記)。exit 0 のあとの `git status --short` は、この jsonl の `M` と、この report の `M` だけで、日付つきの新しいファイルはできていない。`--cycle auto` は `cycle-count.json` の `cycle` 2 を読むので、cycle 2 の sync-docs の event と同じ値が付く(`internal/insights` は run 2 と run 3 を同じ cycle 2 で数える。verify の INFO と同じ)。

---

## Cycle 2 の記録(パイプラインの 2 回目の run)

この節から下は cycle 2 の report で、本文は書き換えていない。D6〜D8、`a42eb2d0` と `d4a91a01` の記録で、cycle 2 の verify と計画の Progress がここを引く。tech-debt の行番号は run 3 の時点でも同じ(行は足していない)。

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
