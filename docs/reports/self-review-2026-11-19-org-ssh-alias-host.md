# Self-review report: org-ssh-alias-host

- Date: 2026-11-20(4 回目の `/self-review`。パイプラインの 3 回目の run。cross-review の 2 回目の ACTION_REQUIRED #1 を直した `664d4771` と `04161b43` のあと。`cycle-count.json` は `cycle: 2`、`cap: 3`)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`
- Branch: `fix/org-ssh-alias-host`、HEAD `8226d125`(計画だけの commit)、コードの最後は `04161b43`、merge base は `main` の `32d6dc51`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff main...HEAD` の全体(41 ファイル、+3234 −217)の diff の質だけ。重点は `664d4771`(`mergeOriginFromURLAsWritten`、`mergeSSHHost`)と `04161b43`(doctor の中断の文)。テストと静的解析は走らせていない
- 番号: 前回までの L1〜L12 の番号は変えない(Go の test のコメントがこの report の `L1`・`L3`・`L10` などを引く)。新しい指摘は L13 から振る

## Evidence reviewed

- `664d4771`(6 ファイル、+200 −72)と `04161b43`(2 ファイル、+5 −1)の全 diff。変えた関数は HEAD の全文で読んだ: `ReadMergeOrigin`、`resolveMergeAPIHost`、`mergeSSHResolveRuns`、`mergeSSHHost`、`mergeSSHHostName`、`mergeSSHAliasHint`、`mergeOriginFromURL`、`mergeOriginFromURLAsWritten`、`mergeGuaranteeRun.readOrigin`、`aliasNote`
- 前提の確かめ(scratch の `-F` 設定、手元の OpenSSH_10.2p1): `Host Work-GH`(`HostName github.com`)のとき `ssh -G Work-GH` は `hostname github.com`、`ssh -G work-gh` と `ssh -G WORK-GH` は `hostname work-gh`。`Host` の照合は大文字小文字を区別し、解けない host の hostname は小文字で出る。triage と `mergeSSHHost` の doc の前提は合っている
- 小文字化の確かめ(scratch の `go run`、全 rune を走査): `strings.ToLower(s) != s` で `strings.EqualFold(s, ToLower(s))` が false になる rune は U+0130(İ)の 1 つだけ。U+212A(Kelvin 記号)は EqualFold が true。`url.Parse` は `ssh://İ.example/o/r` を通し、`Hostname()` は `İ.example`、`ssh://GitHub.COM:22/` は `GitHub.COM`(大文字のまま、port なし)、`ssh://[FE80::1]/` は `FE80::1`
- `ssh` を打つ道の全数: 非 test の `"ssh"` の実行は `mergeorigin.go:196` の 1 か所。その前に `mergeSSHResolveRuns(originURL, host)` を通り、`sshHost` と述語が見る値は同じ純関数 `mergeSSHHost` の結果
- `mergeOriginFromURL` の呼び手(`mergeorder.go:932`、`mergeorigin.go:139`、`mergeguarantee.go:305` と `:885`)は変更前と同じ値を受ける: 失敗の返りは `strings.ToLower("")` で `""`、成功は同じ parse の host を小文字にしたもの
- `mergeSSHHost(line, host)` の呼び手は `resolveMergeAPIHost`(`mergeorigin.go:183`)、`mergeSSHResolveRuns`(`:229`)、doctor の `readOrigin`(`mergeguarantee.go:313`)。`resolveMergeAPIHost` の `cmd`(`:193`)と doctor の `stopped` の文は同じ host で作られる
- register: `git diff main...HEAD -- docs/tech-debt/README.md` を `664d4771`・`04161b43` と突き合わせた(両方向)。直した 2 つの commit は register の行を変えない。`Small leftovers of fix/org-ssh-alias-host` の行(`:535`)が「cycle 2 of 2」「cap 2」と書く点は下の Tech debt に書く
- 掃き出し: `grep -n '"ssh"'`、`ssh -G` を含む文字列(非 test)、`originHost` と `mergeSSHAliasHint` の呼び手、`ACTION_REQUIRED` を引く Go のコメント、`mergeguarantee.go` の `<host>` を名指す doc

## 前回の指摘の扱い

| 前回 | 結果 | 根拠 |
| --- | --- | --- |
| L1(入力の host の検査が出力より緩い) | 解決(前回から変わらない。入力が as-written の host になっても同じ述語) | `mergeSSHHostName` を `mergeSSHHost` の結果に掛ける(`mergeorigin.go:229`)。`ssh` の引数 `sshHost`(`:196`)は述語が見たものと同じ値 |
| L2(merge と cleanup の `--help` が古い) | 解決(前回から変わらない) | `internal/cli` はこの 2 commit で変わらない |
| L3(doctor が「ssh -G が走ったか」を条件から写す) | 解決(前回から変わらない) | 3 か所が同じ `(line, host)` から `mergeSSHHost` を引く。`04161b43` が doctor の文を `cmd` と同じ host にした。test の表に大文字の別名の行がある(`mergeguarantee_alias_test.go:370`)。この行は `mergeSSHHost` を `host` に戻すと落ちる |
| L5(最後の `switch` の 3 つ目の条件が重い) | 解決(前回から変わらない) | `mergeorigin.go:216` |
| L8(L2 の直しが regate の help と doctor のコメントに届いていない) | 解決(前回から変わらない) | 同上 |
| L10(`which gh cannot take as a host` が ralph の規則を gh のものと言う) | 解決(前回から変わらない) | `mergeSSHHostNameWhy` を as-written の host に使う(`mergeorigin.go:185`)。Kelvin 記号の host の理由文は「only ASCII letters...」になる |
| L4(`resolveMergeAPIHost` の形) | 残した。許容できる。ただし数え直しと、組が 1 つ増えた | 下の L4 の行 |
| L6(偽の ssh のプロセスグループの確かめが 3 か所) | 残した。許容できる(変わらない) | `mergeorigin_test.go:510`、`mergeguarantee_alias_test.go:428`、`regate_ci_ssh_alias_test.go:68` |
| L7(test の表の読みにくさ) | 残した。許容できる。ただし表が 1 欄増えた | 下の L7 の行 |
| L9(hint が host を `%s` で出す) | 残した(変わらない)。隣に L14 が立つ | 下の L9 と L14 |
| L11(signal を打つ手順の手書き 2 回) | 残した。許容できる(変わらない) | `doctor_merge_guarantee_signal_test.go:116,246`。この 2 commit は `internal/cli` を触らない |
| L12(割り込みの文が最後の命令のあとの signal にも出る) | 残した。許容できる(変わらない) | `doctor_merge_guarantee.go`、`doctor.go` はこの 2 commit で変わらない |

## Findings

CRITICAL、HIGH、MEDIUM はない。LOW は 9 件。前回から残した 6 件(L4、L6、L7、L9、L11、L12)と、新しい 3 件(L13、L14、L15)。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | null-safety | L13: `mergeSSHHost` の防御の枝 `!ok \|\| !strings.EqualFold(written, host)` は、doc が「host を originURL から読んだ呼び手は見ない」と言うが、U+0130(İ)で届く。`strings.ToLower("İ")` は ASCII の `i` で、`EqualFold("İ", "i")` は false。origin が `git@İ.example.com:o/r.git` なら `host` は `i.example.com`、`written` は `İ.example.com` で、枝に入って `i.example.com` を返す。ASCII なので `mergeSSHHostName` を通り、`ssh -G i.example.com` が走る。cycle 2 の直しが除いた「origin に書かれていない host で ssh の設定を引く」ずれが、この 1 文字で戻る。Kelvin 記号(U+212A)は EqualFold が true なので as-written を返し、`mergeSSHHostName` が断る(2 つの test の行が固定する)。同じ「小文字にすると ASCII になる非 ASCII」で、2 つの文字の扱いが逆になる。退行ではない(`55cdb7c3` から `664d4771` の前まで、小文字の host を渡していた)。枝を通る test はなく、`mergeSSHHost` を直接呼ぶ test もない | `mergeorigin.go:232-247`(doc の「which a caller that read host from originURL never sees」は `:239-240`)。`mergeorder.go:945`(`strings.ToLower`)。確かめは上の Evidence の rune の走査。test の Kelvin の行は `mergeorigin_test.go:173` と `mergeguarantee_alias_test.go:374` | `EqualFold` の条件を外して `if !ok { return host }; return written` にする。`host` を `mergeOriginFromURL` から取った呼び手には `ok` が常に true で、枝は元の「見ない」道だけになり、İ の as-written(非 ASCII)は述語が断つ。`mergeorigin_test.go:173` の隣に `git@İ.example.com:o/r.git`(ssh を打たず、fallback は `i.example.com`、理由は `notRun("İ.example.com")`)を足す。もう 1 つの形は `host` の引数ごと外して `mergeSSHHost(originURL)` にすること(`resolveMergeAPIHost`・`mergeSSHResolveRuns`・`readOrigin` が同じ `originURL, host` の組を 3 度渡す形が 1 つ減り、L4 にも効く) |
| LOW | readability | L14: ssh -G は as-written の host を受けるようになったが、同じ host を指す 5 か所が小文字の host のまま。(1) `mergeSSHAliasHint` は、origin が `git@Work-GH:o/r.git` で ssh -G が失敗した行で「`ssh -G Work-GH failed (...)`, so gh read the host work-gh as origin names it: if work-gh is an ssh alias (a Host in ~/.ssh/config)」と、同じ文の前半と後半で別の綴りを出す。`Host` の照合は大文字小文字を区別するので、利用者が `~/.ssh/config` を `work-gh` で探しても当たらない。(2) `aliasNote` は大文字の別名の origin に「origin's host github.com.emu is an ssh alias of github.com」と小文字で出し、test がそれを固定する。(3) `MergeGuaranteeCheck.APIHost` の doc は「the hostname `ssh -G <Host>` gave」で、`Host` は `:127` が小文字と書く。(4) `CheckMergeGuarantee` の step 1 の doc は「read by mergeOriginFromURL (host and owner/repo). For an ssh origin, `ssh -G <host>`」で、読むと小文字の host を渡すと取れる。(5) `merge.go` の `originHost` の doc は「origin's host as its URL names it」で、小文字とも書かない。`664d4771` は `mergeorigin.go` の doc だけを書き直し、`mergeguarantee.go` を触らない(`04161b43` の 1 行だけ)。キャッシュの `host` と `originHost` を小文字のままにするのは commit message の意図で、そこは問題ない | `mergeorigin.go:302-309`(hint)、`mergeguarantee.go:326-331`(`aliasNote`)、`:127`、`:134-137`、`:169-173`、`merge.go:131-135`。test: `mergeguarantee_alias_test.go:35`(`mgAliasNote`)と `:61,64`(大文字の別名の 2 行が `mgAliasNote` を期待)、`mergeorigin_test.go:304-322`(hint の 5 行は小文字の host だけ)、`:382-391`(大文字の別名の ssh -G 失敗は `why` と `Arg` だけを見て hint は見ない) | 小文字のままにするなら、(1) と (2) の文に「(lowercased)」を足し、(3) (4) (5) の `<host>` を「origin's host as its URL writes it, for ssh -G」に直す。as-written にするなら、`mergeSSHAliasHint` の `originHost` と `aliasNote` の host に `mergeSSHHost(line, host)` の値を渡す(キャッシュの `host` は小文字のまま)。どちらでも、大文字の別名と hint の組を固定する行を `TestMergeSSHAliasHint` に 1 つ足す |
| LOW | maintainability | L15: `doctor_merge_guarantee_signal_test.go:20` は「ACTION_REQUIRED #1 of docs/reports/cross-review-triage-org-ssh-alias-host.md」と引く。cycle 2 の triage はその file の先頭を cycle 2 の ACTION_REQUIRED(大文字小文字の指摘)に書き直し、cycle 1 を `## Cycle 1(履歴)` の下へ移したので、file に ACTION_REQUIRED の #1 が 2 つある。doctor の Ctrl-C を指すこの comment は、先に当たるほう(大文字小文字の指摘)に解決する。同じ file を引く新しい comment は「cycle 2 of ...」で曖昧さを避けている(`mergeorigin_test.go:77,527`、`mergeguarantee_alias_test.go:45`)が、`mergeguarantee_alias_test.go:369` の「(the cross-review of cycle 2)」は path がない | `doctor_merge_guarantee_signal_test.go:19-21`。`docs/reports/cross-review-triage-org-ssh-alias-host.md` の `## ACTION_REQUIRED`(`:22`)と `### ACTION_REQUIRED`(`:64`) | `:20` を「ACTION_REQUIRED #1 of cycle 1 of ...」にし、`:369` に triage の path を足す。triage は cycle ごとに書き直される file なので、`cycle N` を落とさない形が repo の先例(`ACTION_REQUIRED #1` を cycle つきで引く `mergeorder_followup_test.go:655` ほか)に合う |
| LOW | maintainability | L4(残した。数え直し): `resolveMergeAPIHost` の引数は 6 つ(`ctx, dir, sshRun, timeout, originURL, host`)で、前の report と register の行が書く「7 つ」は数え違い。`664d4771` は `originURL, host` の組を取る関数を 1 つ増やした(`mergeSSHHost`)。この組は `resolveMergeAPIHost`、`mergeSSHResolveRuns`、`mergeSSHHost` の 3 つが取り、`readOrigin` は 3 つとも呼ぶ。`sshRun == nil` の既定は `ReadMergeOrigin`(`:122`)と中(`:187`)の 2 か所のまま | `mergeorigin.go:118,122,178,187,228,241`、`mergeguarantee.go:311-313` | 動きは変わらない。L13 の「`host` の引数ごと外す」で組が 1 つ減る。直さないなら register の「seven」を「six」に |
| LOW | maintainability | L6(残した): 偽の ssh を PATH に置いてプロセスグループを確かめる手順が 3 つの test ファイルに複製 | `mergeorigin_test.go:510`、`mergeguarantee_alias_test.go:428`、`regate_ci_ssh_alias_test.go:68` | `soFakeSSH` の隣に共通の確認を置く |
| LOW | readability | L7(残した。表が 1 欄増えた): `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` の表は、`664d4771` で `sshHost` が入り位置指定 8 欄になった。文字列の欄が `origin, sshHost, ssh` と 3 つ続き、行は `mgAliasOrigin, soEmu, mgAliasSSH` と並ぶ。入れ替えても型は通る。`map[bool]int{true: 1, false: 0}[tc.sshRuns]` は 2 か所のまま | `mergeguarantee_alias_test.go:54-70`(表)、`:186`、`:386` | 欄名つきの struct literal と、`want := 0; if tc.sshRuns { want = 1 }` |
| LOW | security | L9(残した): `mergeSSHAliasHint` が host を `%s` で出す(同じ文の `why` は `%q`)。host は利用者の git config の値で、制御文字を持てるのは scp 形の origin だけ | `mergeorigin.go:302-309` | `apiHost` と `originHost` も `%q` にし、期待文を直す。または生で出すと決めて、`%q` と混ぜない。L14 の直しと同じ commit でやると文の期待が 1 度で済む |
| LOW | maintainability | L11(残した): 新しい test が「自分のプロセスに signal を送る」手順を 2 回、手で書いた。`signalSelf(sig)` が同じ手順を持つ | `doctor_merge_guarantee_signal_test.go:116,246`。`org_gate_test.go:817` | 表の `setup` と 2 つ目の test の `Notify` から `Kill` までを `signalSelf(sig)(t, nil)` に |
| LOW | exception-handling | L12(残した): 検査が返った時点で ctx が終わっていれば、最後の命令のあとに来た signal でも `Interrupted: the Merge guarantee check did not finish.` が出る(Node の検査は probe が割り込まれたときだけ `interrupted` を返す) | `doctor_merge_guarantee.go:111-119`、`doctor.go:244-249`、`:320` | 残してよい。直すなら banner を「did not finish」に頼らない形に(Node の test の期待文も変わる) |

## 依頼された点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| as-written の host の経路は正しいか(scp 形、`ssh://`、`user@`、port、IPv6) | 正しい | scp 形は `strings.Cut(s, ":")` の前半から最後の `@` の後ろを取る(大文字は残る)。`ssh://` は `u.Hostname()` が user と port を除き、大文字を残す(`GitHub.COM`、`FE80::1` を確かめた)。IPv6 の `[...]` は外れ、`:` は許可リストに入っているので通る。scheme の大文字(`SSH://`)は `url.Parse` が小文字にし、`mergeOriginIsSSH` と `AsWritten` の scheme 表で同じに扱う。test の表に scp 形、`ssh://`、port つき、別名、`;` を含む大文字の行がある |
| 呼び手に与えた host が as-written と大文字小文字を除いて違うときの防御の枝は | 枝に穴が 1 つ: U+0130 | L13。Kelvin 記号は枝に入らず、述語が断つ(正しい) |
| Kelvin 記号の場合 | 正しい | `EqualFold` が true で as-written(非 ASCII)を返し、`mergeSSHHostName` が false。ssh は走らず、理由は `ssh -G was not run for the host "K.example.com", which is not a plain host name (...)`。test は `mergeorigin_test.go:173` と `mergeguarantee_alias_test.go:374`(doctor 側は何も名指さない) |
| `mergeOriginFromURL` の呼び手は変わらないか | 変わらない | 4 つの呼び手が同じ値を受ける(Evidence)。named result の代入は成功と失敗の両方で元と同じ。`mergeOriginFromURL` の doc の「as written, lowercased」は今も合う |
| doctor の `readOrigin`、解く関数、`mergeSSHResolveRuns` は同じ host を読むか | 読む | 3 か所とも `mergeSSHHost(line, host)` を引く。`04161b43` が `stopped` の文を `cmd` と同じにし、test の表が大文字の別名の行で固定する |
| 小文字のまま残した値(解いた hostname、fallback、キャッシュの `host`、`originHost`)は一貫するか | 動きは一貫する。文と doc が揃っていない | L14 |

## 確かめたが挙げなかったもの

- `mergeOriginFromURLAsWritten` の名前と、`mergeOriginFromURL` の doc の「host as written, lowercased」: どちらも自分が何を返すか書いていて、読み間違える形ではない
- `mergeSSHHost` が `originURL` を再 parse する(1 回の doctor の検査で `url.Parse` が 8 回ほど): 費用は無視できる。形の問題は L4 と L13 の直しに含まれる
- triage の「`mergeorigin.go`(`resolveMergeAPIHost`、183〜186 行)」の pointer: HEAD でも同じ区画(`sshHost :=` から not-run の `}` まで)に当たる
- ssh が出す hostname の小文字化と `mergeGuaranteeAPIHost` の順序: `strings.ToLower(resolved)` のあとに `ssh.github.com` を読み替える(`:219`)。変わらない
- Go の comment が `docs/reports/cross-review-triage-<slug>.md` を引くこと自体: repo の慣習の範囲(前の run と同じ判断)。L15 は引き方の曖昧さだけを挙げた

## Positive notes

- 前提を原因の側で直した。小文字にするのをやめて、`mergeOriginFromURL` を薄い wrapper にし、parse は 1 か所に残した(`mergeOriginFromURLAsWritten`)。呼び手 4 つの値は変わらない
- 小文字のままにする値と as-written にする値を commit message と doc の両方に書いた(解いた hostname、fallback、キャッシュの `host`、`originHost` は小文字、`ssh -G` の引数、述語、`why` の文は as-written)
- `mergeSSHResolveRuns` に `mergeSSHHost` を通したので、「ssh -G が走ったか」の述語は解く関数と doctor で同じ値を見たまま。`04161b43` が doctor 側の文の host も揃え、test の行を足した
- test が大文字の経路を本物の `realMergeDetachedExec` まで通す(`TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten`: 偽の ssh が `Work-GH` にだけ github.com を答えるので、小文字の host ならこの test は落ちる)。表に別名の大文字、fallback の小文字、`;` を含む大文字、Kelvin 記号の行がある

## Tech debt identified

この run はパイプラインの 3 回目で、cap は 3(`cycle-count.json` の `cap`)なので最後の run になる。直さずに進める LOW は、そのまま延期になる。register の `Small leftovers of fix/org-ssh-alias-host` の行(`docs/tech-debt/README.md:535`)は (a)〜(g) を持つ。この run の指摘のうち L13・L14・L15 は行にない。`/sync-docs` が次の変更を行にまとめる(直した文字は消す)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Additions to the row "Small leftovers of fix/org-ssh-alias-host (#340)": (h) `mergeSSHHost` in `internal/org/mergeorigin.go` falls back to the lowercased host when `strings.EqualFold(written, host)` is false, which its doc says a caller that read `host` from the URL never sees; U+0130 ("İ") does reach it (`strings.ToLower` gives the ASCII "i", which `EqualFold` does not fold back), so `ssh -G i.example.com` runs for the origin `git@İ.example.com:o/r.git`, while U+212A (the Kelvin sign) is refused as a host that is not plain; no test reaches the branch (self-review L13). (i) The `mergeSSHAliasHint` clause ("gh read the host work-gh as origin names it: if work-gh is an ssh alias"), `aliasNote` ("origin's host work-gh is an ssh alias of github.com"), and the docs of `MergeGuaranteeCheck.APIHost`, step 1 of `CheckMergeGuarantee`, and `originHost` in `internal/org/merge.go` name the lowercased host where `ssh -G` got the host as origin's URL writes it, next to a `why` that says `ssh -G Work-GH` (L14). (j) `internal/cli/doctor_merge_guarantee_signal_test.go` cites "ACTION_REQUIRED #1" of the triage file without a cycle, and the file now holds a #1 for cycle 2 above the one for cycle 1; `mergeguarantee_alias_test.go` has one comment citing "the cross-review of cycle 2" with no path (L15). Corrections to the row as it stands: `resolveMergeAPIHost` takes six arguments, not seven (item (a)); the table of `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` has eight positional fields since `664d4771`, not seven (item (c)); "left in place at the last pipeline run (cycle 2 of 2)" and "the pipeline was at its last run (cap 2)" are false since the cap was raised to 3 for this task, so say "left in place at the last pipeline run (run 3, cap 3)" | LOW. (h) needs an origin host that holds U+0130 and an ssh config with a `Host` that the lowercased name matches; (i) is wording and docs on the case of a host, no behavior; (j) is a pointer in a comment | The pipeline is at its last run (cap 3), where a fix of code starts it again; the branch fixed the findings that changed what a user reads or what ran, and the cross-review's two findings. (h) is no regression: the code before `664d4771` passed the lowercased host for every origin | The next change to `mergeSSHHost` or `resolveMergeAPIHost` for (h) (drop the `EqualFold` condition, or the `host` argument); the next change to `mergeSSHAliasHint`, `aliasNote`, or the `Host` and `APIHost` docs of the guarantee check for (i); the next signal or alias test of these files for (j) | `docs/plans/active/2026-11-19-org-ssh-alias-host.md`; self-review of org-ssh-alias-host (L13, L14, L15), `docs/reports/self-review-2026-11-19-org-ssh-alias-host.md`; `docs/reports/cross-review-triage-org-ssh-alias-host.md` (cycle 2, ACTION_REQUIRED #1) |

この run は、依頼された範囲(report だけの commit)に合わせて `docs/tech-debt/README.md` を変えていない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 9 件。前回の 6 件はすべて残り、新しい 3 件を足した。`664d4771` と `04161b43` は意図した直しで、新しい指摘はどれも動きの退行ではない)
- Follow-ups:
  - L13 は 1 行の直しで、doc の「見ない」を本当にする。ただし code の直しで、cap が 3/3 なので full pipeline の再走になる。直さずに register の (h) へ入れるほうが小さい。cap をもう 1 つ上げて直すなら、L14 と L9 を同じ commit にまとめると文の期待が 1 度で済む
  - L15 は comment だけの直しだが、`internal/` の Go を変えるので「記録だけの修正」の例外に入らない
  - `/sync-docs` へ(数えない。文書のずれ): (1) register の `Small leftovers of fix/org-ssh-alias-host` の行の「cycle 2 of 2」「cap 2」は cap の引き上げで偽になった。「seven arguments」は「six」、表の「seven positional fields」は 1 つの表で 8 欄。上の Tech debt に直し方を書いた。(2) `cycle-count.json` の `cycle` は cap の引き上げで 2 のまま(`cycle: auto` の event は cycle 2 を付ける)。前の run の self-review の event と同じ値になる
  - 計画の Progress は `664d4771`・`04161b43` を記録している。計画に「as-written の host」の AC はなく、cross-review の直しとして記録されている。digest で固定された計画を触らないやり方は前の run と同じ
- Known gaps:
  - 本物の `ssh` で `GitHub.COM` や `İ` の origin を打った確かめはしていない。`Host` の大文字小文字の前提だけを scratch の `-F` 設定で確かめた(`ssh -G` は接続しない)
  - L13 の `i.example.com` は、枝に入って述語を通るところまでをコードと rune の走査で確かめた。`ssh` の実行は走らせていない
  - テスト、静的解析、spec の合致、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の範囲)
