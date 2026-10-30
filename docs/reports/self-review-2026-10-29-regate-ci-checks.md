# Self-review report: regate-ci-checks

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-29-regate-ci-checks.md`(issue #264、D1〜D7、AC1〜AC10、スライス S1・S0・S2・S3・S4a・S4b と 13d19f3d・S5・S6・S7)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD 5112a656、49 ファイル、+5482/-327)。中心は `internal/org/regate_ci.go`(新規)、`regate.go`・`regate_merge.go`・`regate_git.go` の差分、`prchecks.go`(新規)と `merge.go` の差分、`gate_record.go`・`seat.go`・`mergeorder.go`・`watch.go`・`lockfile.go` の差分、`internal/cli/org_regate.go`・`org_gate.go`・`exitcode.go`、`internal/config/config.go`、2 つの `ralph.toml`。skill と README は、コードの挙動を言い切っている箇所だけ突き合わせた。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L8(LOW)。

## Evidence reviewed

- 計画の D1〜D7、Critical forks、Progress checklist(S1〜S7 と、S3 の「計画とのずれ」、S4a・S4b・S5 の引き継ぎ)。
- `regate_ci.go` の全体(679 行)。事前の確かめ(`regateCIPreCheck`)、merge の道の push と読み返し(`regateCIMerged`・`regateCIPush`・`regateCILanded`)、判定(`regateChecks`)、待ち(`regateChecksWithWait`・`regateCIRecheck`・`regateCIWait`)を、`Regate` の道の選び方(`regate.go:497-540`)と突き合わせた。
- push の失敗の 3 つの分かれ(D3): 外すのは `unread == "" && tip == res.PreMergeHead` のときだけ(`regate_ci.go:286`)。読み返した先頭が merge commit なら `regateCILanded`、それ以外(子孫、別のコミット、ブランチなし、読めない)は HEAD を残して断る。push か読み返しの最中の ctx の終わりは「may or may not」の error で HEAD を残す。push の前の ctx の終わりと、追跡 ref が読めないときだけ、push せずに undo する。
- 待ち(D5): lock を外すのは `w.sleep` の直前だけ(`regate_ci.go:550`)。取り直しは `acquireHeldFileLock(..., regateLockWait)`、marker を空にしてから `regateFirstCheck`(結びつき、走っている org)、`regateBranchChanged`、worktree の HEAD、`git ls-remote` で読んだ origin の先頭、の順。`now.passedOn(t)` なら UpToDate。sleep の失敗と ctx の終わりは何も記録せずに終わる。`regateHeldLock.release` は nil を見るので、眠りのあいだの割り込みで `Regate` の defer が二度外すことはない。
- 読み手: `git grep` で `EventGatePassed`・`EventGateFailed`・`EventGateCI*` の `_test.go` 以外の出現を見た。gate の記録を event 名で選ぶ箇所は `collectMergeTargets`(`mergeorder.go:487`)の 1 つで、merge-order、merge、regate(`readRegateBranch`)が通る。もう 1 つの `watch.go:1405` は、数えないことを注釈で決めている。`gateRecord.event()` が via と event 名を組にするので、CI の Details を `gate_passed` で書いた行も、その逆も unreadable になる。
- S3 の共用化: `readChecks` と `findPR` の前後で、case の順(ctx → 見つからない・時間切れ → no-checks の文 → JSON でない)と `mergeRefusalKind` が同じであること。`ghFailed` は ctx を先に見るので、`originPRInterrupted` で `f.err` が nil でも中断の error になる。
- gh の argv: 台帳から来る値は `--head <branch>` と `-R <repo>` の flag の値と、数値の PR 番号だけで、positional はない。
- gh と git の文のエスケープ: `refuse`・`pending`・`FailedChecks`・`w.line`・`regateCIRecheck` の `found` は `PrintableInboxText` を通る。git の stderr(`regateStderrTail`)は通らないが、local の `regatePush` と同じ扱い。
- CLI: `printRegateResult` の outcome ごとの行、知らない outcome の扱い、`ErrRegateChecksPending` と `ExitCode` の 6、`printGateFailDetail` の `ci_failed`。
- テスト: `regate_ci_test.go` の fixture(`rcFixture`、`mxExec` の呼び出しごとの `-R`・env・deadline を見る `assertKeys`)、AC7 の表(`pre-push` hook で git を kill する case を含む)、割り込み、待ちの確かめ直しの表、`assertLockedAtTheRecord`。`regate_merge_test.go` の AC10(`TestRegate_ACIPassIsNotPushedAgainOnceOriginMovedOff`)。`org_regate_test.go` の ci の CLI テスト 7 本。
- 4 つの `/director` skill の写しは md5 が一致する。
- probe: `git check-ref-format --branch` で `feat/$(id)`、`feat/a;b`、バッククォート入り、`'` 入りのブランチ名がすべて valid であることを確かめた(L8)。
- `wc -l` で、触ったファイルの origin/main と HEAD の行数を比べた(M1)。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1 この PR で 800 行を越えたファイルが 3 つあり、台帳に行がない。`internal/org/regate.go` は 763 → 973 行。台帳の 289 行は、このファイルの 800 行越えを 2026-10-16(org-regate)に RESOLVED として閉じている。新しい `internal/org/regate_ci_test.go` は 1795 行で生まれ、`internal/cli/org_regate_test.go` は 760 → 1256 行で越えた。290 行の Trigger(「`regate_test.go` か `regate_merge_test.go` の次の変更」)は、この PR が 2 つとも触った(+51、+85)ので引かれたが、払ってもいないし、前の PR のような注記もない。`regate_ci.go`(679 行)と `prchecks.go`(223 行)を別のファイルにした判断は良い。 | `wc -l`(origin/main → HEAD): `regate.go` 763→973、`org_regate_test.go` 760→1256、`regate_ci_test.go` 0→1795、`regate_test.go` 1545→1596、`regate_merge_test.go` 1991→2076。`docs/tech-debt/README.md:289-290` | `regate_ci_test.go` はすでに節で分かれている(事前の確かめと gate だけの道 `:154-535`、merge の道と push の読み返し `:537-1197`、待ち `:1199-`)ので、移すだけで 3 ファイルにできる(`rcFixture` などの共有部は `regate_ci_fake_test.go` へ)。`org_regate_test.go` の ci の部分は `org_regate_ci_test.go` へ移す。`regate.go` の増えた分は大半が `RegateParams`・`RegateResult` の ci の doc なので、分けないなら 289 行を開き直すか新しい行を足し、290 行に「regate-ci-checks も触った」と書く。 |
| LOW | exception-handling | L1 `via=regate_ci` の pass の HEAD より origin の先頭が進んでいる(誰かが別の clone から上に push した)ときも、regate は `regateCIPassMovedText` の「someone moved origin's branch since (with a force push, for one), and pushing HEAD again would undo that」で断る。この形では push は non-fast-forward で拒まれるので何も戻らない。要るのは worktree の早送りだが、文はその手を出さない。merge-order はこの形を `regate`(「pushed after the gate, or the gate's head was not pushed」)に分けて regate を打つよう言うので、director は regate を打ってこの断りに当たり、exit 1 で止めて人にこの文を渡す。local の pass なら同じ形で `regateCheckOriginHead` の「origin's tip … has commits that HEAD … does not …: bring the worktree up to date (git -C … merge --ff-only …)」になる。AC10 のテストは、親に戻した形と別のコミットの形の 2 つだけで、子孫の形を持たない。 | `internal/org/regate_merge.go:83-86`、`:118-124`、`internal/org/regate_git.go:139-145`、`internal/org/mergeorder.go:1160-1162`、`internal/org/regate_merge_test.go:1494`(`TestRegate_ACIPassIsNotPushedAgainOnceOriginMovedOff`) | ci の pass で止めるのは push だけの打ち直しに限る。origin の先頭が HEAD を含む(`regateIsAncestor(HEAD, originHead)`)ときは `regateCheckOriginHead` の文を返し、HEAD が先にある形と分かれた形にだけ今の ci の文を残す。AC10 のテストに子孫の case を 1 つ足す。 |
| LOW | readability | L2 push が届かなかったときの断り(読み返した先頭が merge の前の先頭)は、`regateStderrTail` の「; git said:」と字下げした git の行のあとに「; the merge commit … was undone (git reset --keep …), so only the reflog has it now」を続ける。ralph の文が git の最後の行と同じ字下げの行に付くので、undo したことが git の出力の一部に読める。同じ関数の「HEAD を残す」断りと `regatePush` の `PushFailed` は、git の出力を最後に置いている。 | `internal/org/regate_ci.go:288-290`(`"…fix it and %s%s; %s", …, w.again, said, regateUndoneText(res)`)、`:302-303`、`internal/org/regate_merge.go:605-611`。`TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack` の suffix がこの順を固定している | `regateUndoneText(res)` を `said` の前に移し、`…: fix it and <again>; <undone>; git said: …` の順にする。テストの prefix と suffix も合わせて直す。 |
| LOW | maintainability | L3 待ちは 30 秒ごとに `regate.lock` を取り直し、そのたびに `regateLockWait`(1 秒)で取れなければ、残りの時間があっても断って終わる(exit 1)。60 分の待ちなら取り直しは 120 回ある。merge の待ちが取り直すのは終わりの 1 回だけ。D5 は「待つあいだにほかの merge と regate を止めない」ための設計だが、ほかの 1 つが取り直しの瞬間に 1 秒より長く lock を持っていれば(local の regate のスクリプト、merge の `gh pr merge` と読み返し、cleanup、ほかの ci の regate の確かめ直しと gh の 2 回と ls-remote)、こちらの待ちが終わる。同じ台帳で 2 つの ci の待ちが重なると、どちらも 30 秒ごとに数秒ずつ持つので、1 時間のうちにほぼ確実に片方が断られる。`/director` skill は exit 1 で止めて人に伝えるので、待てば済んだ枝が人の手に渡る。skill は「1 本ずつ打つ」と書くので director だけなら重ならないが、人が並べて打つと当たる。 | `internal/org/regate_ci.go:550-557`、`:596-600`、`internal/org/regate.go:158`(`regateLockWait = time.Second`)、`internal/org/merge_wait.go:28-33`。`TestRegateCI_WaitChecksChecksAgainAfterASleep` の「another regate holds regate.lock」の case がこの断りを固定している | 取り直しで `errFileLockHeld` のときは、断らずに lock を外したまま次の眠りへ進む(残りの時間の中で)。時間が尽きた時点でまだ取れなければ、今の文で断る。今の動きを残すなら、help の `--wait-checks` の段落と skill に「取り直しの時に lock が取れなければ、待ちは断りで終わる」と書く。 |
| LOW | null-safety | L4 PR の head(`gh pr view --json headRefOid`)と checks(`gh pr checks`)を別々の 2 回で読み、そのあいだの push を見ない。`gh pr checks` の出力には commit がないので、1 回目のあとに branch が動くと、新しい head の checks を `t.head` の記録として書く。新しい head の checks はふつう pending か 0 件なので記録にならない。偽の pass になるのは、branch がすでに checks の通ったコミットへ戻されたとき(force push)に限られ、その記録は merge-order の head の比べ(記録の head が origin の先頭でない → `regate`)と merge の読み直しで止まる。もう 1 つ、head が HEAD と違うときの pending の文は理由を「GitHub has not updated the PR yet」と決めているが、gate だけの道では fetch のあとに origin が動いた場合も同じ形になる(待ちの確かめ直しでは origin の先頭の変化として断られる)。 | `internal/org/regate_ci.go:372-384`、`:379-381`、`internal/org/prchecks.go:185-188`(`--json bucket,name,state,workflow`) | checks を読んだあとに `readPRHead` をもう 1 回打ち、変わっていれば pending(「PR #n's head moved while its checks were read」)にする。pending の文は「GitHub has not updated the PR yet, or origin's branch moved after the fetch」にする。 |
| LOW | maintainability | L5 記録は、gh が今 list する checks がすべて終わったかで決まり、まだ作られていない check は数えない。この repo には PR で走る workflow が 2 つある(`verify.yml` は 10 分を越え、`check-template.yml` は短い)。push のあと verify の run の作成が遅れ(Actions の遅延)、check-template が先に通れば、`gate_ci_passed` が pass 1 件で書かれ、merge-order は `ready` を出す。判定は merge と同じ(D4。`--required` は Non-goals)で、merge の直前の読み直しと保護ブランチの必須 checks がそのあとを守るが、記録は「CI が通った」と言い過ぎる。 | `internal/org/regate_ci.go:396-409`、`.github/workflows/verify.yml:2-5`、`.github/workflows/check-template.yml:3-7`、計画の Non-goals の 3 つ目 | この PR では直さず、台帳に行を足す(下の Tech debt の 2 行目)。払うときは、`[org.gate]` に期待する check の名前を持たせるか、保護ブランチの必須 checks を読んで、それが list にないうちは pending にする。 |
| LOW | readability | L6 `ralph org regate --help` の最後の段落は「While it runs, ralph org spawn (not a --dry-run) and ralph org start (either form) of the org it works on are refused」のままで、ci の待ちの眠りのあいだは lock を外すので正しくない。同じ help の `--wait-checks` の段落も「another regate, merge, or cleanup can run meanwhile」とだけ書き、spawn と start に触れない。`Regate` の doc は「while it sleeps, another regate, merge, cleanup, or spawn of the org may run」と書き、merge の help は「While it runs, except while it waits」と書いている。待ちのあいだに spawn した org は、確かめ直しで「running again」として断られる。 | `internal/cli/org_regate.go:160-164`、`:180-182`、`internal/org/regate.go:423`、`internal/cli/org_merge.go:201` | 最後の段落を「While it runs, except while a ci regate sleeps in its wait for the checks, …」にし、`--wait-checks` の段落に「a spawn or a start of the org meanwhile makes the next check refuse」を足す。`/director` skill の手順 1 の「regate は台帳ごとに 1 つしか走らない」も同じずれなので、sync-docs に渡す。 |
| LOW | maintainability | L7 `templates/base/ralph.toml` の `[org.gate]` のコメントの最後が「The ralph repository's own ralph.toml sets "ci".」と、この meta-repo の値を書く。このファイルは `ralph init` で downstream の repo に入るので、そこで読む人には関係がなく、この repo の値が変わると古くなる。ほかの表(`[standard]`、`[org.review]`)のコメントは、ルートの `ralph.toml` が on にしていることを書いていない。 | `templates/base/ralph.toml:277-278`、`ralph.toml:1-8`(ルートのヘッダが同じことを書いている) | 最後の 1 文を消す。 |
| LOW | security | L8 新しく足した次の手のコマンドが、worktree のパスとブランチ名を quote せずに埋める(`git -C %s fetch origin, then git -C %s log --oneline --graph HEAD refs/remotes/origin/%s`、`gh pr create --head %s --base %s -R %s`、`git -C %s remote add origin <URL>` など)。git は `feat/$(id)`、`feat/a;b`、バッククォート入りのブランチ名を受け付け(probe で確かめた)、親ディレクトリに空白がある worktree のパスは、そのまま貼ると `-C` の引数が割れる。同じ package には #263 で入った `cleanupShellWord` があり、cleanup の Manual のコマンドはそれを通す。前からある regate の文(`regateCheckOriginHead` など)も quote していないので、この PR だけの形ではない。 | `internal/org/regate_ci.go:86-87`、`:109-112`、`:117-123`、`:320-323`、`internal/org/regate_merge.go:118-124`、`internal/org/cleanup.go:1079-1086`、`docs/tech-debt/README.md:388` | 新しい文のパスとブランチを `cleanupShellWord` で包む。前からある regate の文まで広げないなら、388 行に「regate の文は quote しない」と範囲を書き足す。 |

## Positive notes

- push を記録の前に置き、merge commit を外すのは「origin に届いていない」と読み返しで確かめられたときだけにしている。読み返せない、子孫、別のコミット、ブランチなしの 4 つは、どれも HEAD を残して何も書かない。外したときも merge commit は reflog に残る。force push はどこにもない。
- CI の記録を新しい event 名で書き、`gateRecord.event()` で via と event 名を組にした。古い binary は `gate_ci_*` を読み飛ばすので、revert のあとに古い regate が CI の pass を自分の pass と読み、PushOnly で押し直すことはない(D4、計画の plan advisory の指摘 1)。読み手は `collectMergeTargets` の 1 か所に足せば済み、grep で取りこぼしはなかった。
- 待ちの確かめ直しが、記録が頼るものを一通り読み直す(結びつき、走っている org、worktree の HEAD、origin から直接読んだ先頭、ほかの regate の pass)。テストは眠りのたびに lock が空いていること(`lockTaken`)と、記録の直前に lock を持ち marker を書き直していること(`assertLockedAtTheRecord`)を見ている。
- 事前の確かめを fetch の直後、すべての道の前に 1 回だけ置き、merge と同じ `findOriginPR` で PR を読む。merge-order と merge が読む PR と食い違わない。
- S3 の共用化は merge の挙動を変えていない。case の順、`mergeRefusalKind`、文言が前と同じで、merge の既存テストの期待は 1 つも変わっていない。
- CLI は知らない outcome に `GATE:` の行を出さず、exit 6 を `errors.Is` で sentinel に結んでいる。`ci_failed` の失敗で、偽の「nothing was pushed」を出さないよう分けている。
- テストの偽物(`mxExec`)は、呼び出しごとに `-R`、env、deadline、dir を見る。push の失敗は、origin が拒む形と `pre-push` hook が git を kill する形の両方を本物の git で起こし、読み返しの分かれをすべて fixture にしている。
- local のまま: `ci == nil` なら `regateGate` と `regateMerged` の元の道を通り、`TestRegateCI_LocalChecksRunTheScriptsAndCallNoExec` が Exec を一度も呼ばないことを見る。既存テストで変わったのは、S7 で意図して直した UP_TO_DATE の文の 2 か所だけ。

## Notes(所見にしないもの)

- push を時間切れ(120 秒)で止めたあと、server が ref を更新しきる窓がある。読み返しが merge の前の先頭を見て undo したあとで、origin に merge commit が載ることがありうる。それでも merge commit は reflog と origin に残り、次の regate は「origin's tip … has commits that HEAD … does not: … merge --ff-only」と言うので、コミットは失われず、記録も書かれない。
- `remote.origin.pushurl` が url と別の repository を指していると、push は pushurl へ行き、読み返し(`git ls-remote origin`)と `git remote get-url origin` は url を読む。local の regate も同じ前提に立っており、珍しい設定なので所見にしない。
- 待ちのあいだ、PR の state と base は確かめ直さない(D5 の一覧のとおり)。閉じた PR の checks で pass を書いても、merge-order は PR の state を先に見て `closed` にするので、`ready` にはならない。
- 関数の長さ: `regateChecks`(約 90 行)、`regateCIPush`(約 70 行)、`regateCIPreCheck`(約 67 行)は 50 行を越えるが、大半は断りの文の組み立てで、この package の既存の関数と同じ書き方。

## Tech debt identified

この run は report と insight event だけを書くので、台帳には書いていない。直さずに進める所見があれば、次の行を台帳に足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| regate-ci-checks で 800 行を越えたファイル: `internal/org/regate.go`(763 → 973 行、289 行で RESOLVED にしていたもの)、`internal/org/regate_ci_test.go`(新規 1795 行)、`internal/cli/org_regate_test.go`(760 → 1256 行)。290 行の 2 ファイルもこの PR で増えた(`regate_test.go` +51、`regate_merge_test.go` +85) | 読むのとレビューに時間がかかる。挙動は変わらない | (M1 を直さない場合)移すだけの分割は、レビュー中の diff を増やす | `regate.go` か、上のテストファイルの次の変更: `regate_ci_test.go` を事前の確かめと gate だけの道、merge の道と読み返し、待ちの 3 つに分け、`org_regate_test.go` の ci の部分を `org_regate_ci_test.go` に移す | `docs/plans/active/2026-10-29-regate-ci-checks.md`、`docs/reports/self-review-2026-10-29-regate-ci-checks.md`(M1) |
| ci の regate は、gh がその時点で list する checks がすべて終わったかで `gate_ci_passed` を書き、まだ作られていない check を数えない(`regateChecks`、`internal/org/regate_ci.go`)。この repo には PR で走る workflow が 2 つあり(`verify.yml`、`check-template.yml`)、verify の run の作成が遅れると、check-template の pass だけで記録になる | 記録が「CI が通った」と言い過ぎ、merge-order が `ready` を出す。merge は直前に checks を読み直し、保護ブランチの必須 checks が merge を止めるので、記録だけで merge されることはない | merge と同じ判定にする計画の方針(D4)と、`--required` の読み方を外した Non-goals による | 記録のあとに merge が必須 checks で止まった事例が出たとき、または `[org.gate]` に期待する check の名前を足すとき | `docs/plans/active/2026-10-29-regate-ci-checks.md`(D4、Non-goals)、`docs/reports/self-review-2026-10-29-regate-ci-checks.md`(L5) |

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件(M1)、LOW 8 件(L1〜L8))
- Follow-ups: L1、L2、L6、L7 は数行の直しで、この PR の中で直すのがよい(L1 は AC10 のテストに子孫の case を足す)。L3 と L4 も直しは小さい(取り直しで lock が取れないときに眠りへ戻す、checks のあとに head を読み直す)。M1 は分割するか、上の 1 行目を台帳に足す。L5 は台帳の 2 行目に送る。L8 は新しい文を `cleanupShellWord` で包むか、388 行に範囲を書く。`/director` skill の「regate は台帳ごとに 1 つしか走らない」は sync-docs に渡す。

## cycle 1 run 2(HEAD f103777d)

- Date: 2026-10-30
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の所見の直しの review)
- Scope: `git diff 03f57f7f..f103777d`(7 コミット、22 ファイル、+2287/-1401)。F1 e40b3466(L3・L4)、F1b f32fe553 と 02219639(待つあいだの `git ls-remote`)、F2 69f80d8f(L1・L2・L6・L7・L8)、F3 aff0dcb0(M1 のうち新しいテストの分割)、5c850e50(台帳の 3 行と 290 行への追記)、f103777d(計画の Progress)。読んだのはこの差分と、それが触るコード: `regate_ci.go` の全体、`regate.go` の doc、`regate_merge.go` の `regateHeadNotOnOrigin` から `regateUndoneText` まで、`regate_git.go` の `regateReadOriginTip`・`regateCheckOriginHead`、`merge_wait.go` の冒頭、`cleanup.go` の `cleanupGitCmd`・`cleanupShellWord`、台帳の 289・290・388・395〜397 行。
- ID: この節の所見は `R2-<n>`(LOW 5 件、R2-1〜R2-5)。上の M1・L1〜L8 とは別の番号。上の `Merge:` の行は cycle 1 の記録として残し、いまのブランチ(HEAD f103777d)の verdict はこの節の `Merge:` の行。

### cycle 1 の所見の扱い

| ID | 扱い | 根拠 |
| --- | --- | --- |
| M1 | 一部解消 | `regate_ci_test.go` は 4 つに分かれ、どれも 800 行未満(767・571・726・300)。`regate.go` と `org_regate_test.go` は台帳の 395 行。直しのあいだに 800 行を越えた `regate_ci.go` が 395 行にない(R2-1) |
| L1 | 解消 | `regateHeadNotOnOrigin` が `regateIsAncestor(HEAD, originHead)` で子孫の形を分け(`internal/org/regate_merge.go:87-94`)、`regateCIPassBehindText`(`:143`)が force push とも undo とも言わずに早送りを出す。AC10 のテストに子孫の case がある |
| L2 | 大半を解消 | push が届かなかった断りは `…; <undone>; git said: …` の順になった(`internal/org/regate_ci.go:296-297`)。undo が git reset で失敗した形には同じずれが残る(R2-5) |
| L3 | 解消 | 待つあいだは lock なしで読むだけ、取り直しは終わりの 1 回(`regate_ci.go:667-695`)。merge の待ちと同じ形 |
| L4 | 解消 | checks のあとに head を読み直し、動いていれば moved の pending(`regate_ci.go:446-457`)。pending の文も「or origin's … moved after regate read it」になった(`:420-422`) |
| L5 | 台帳へ | `docs/tech-debt/README.md:396` |
| L6 | 解消 | help の最後の段落(`internal/cli/org_regate.go:185-189`)と、`/director`・`/org` skill の 8 つの写し。help のテストが新しい文を固定している |
| L7 | 解消 | `templates/base/ralph.toml` の最後の 1 文が消えた |
| L8 | 大半を解消 | 新しい文のパスとブランチは `cleanupGitCmd`・`cleanupShellWord` を通り、`sh -c` で走らせるテストがある。共通の末尾「run ralph org regate --branch X again」は台帳の 397 行。前からある regate の文は 397 行の範囲に入っていない(R2-2) |

### Evidence reviewed

- 記録がいつも lock の下の読みから作られること: `regateChecks` を呼ぶのは `regateChecksWithWait` の最初(lock を持ったまま、`regate_ci.go:659`)と、`regateCIRecheck` が lock を取り直したあと(`:695`)の 2 か所だけ。poll の読み(`:743`)の結果は `last.pending` と `last.moved` にしか使われない。`assertLockedAtTheRecord` が記録の直前に lock を持っていることを見る。
- poll が確かめ直しを飛ばす道はない。poll が error も ranOut も返さなければ、必ず `regateCIRecheck`(`:683`)を通る。ranOut と ctx の終わりは lock を取り直さずに終わり、何も記録しない。`regateHeldLock.release` は nil を見るので、`Regate` の defer が二度外すことはない。
- 終わらない道もない。poll は回るたびに締め切りを見る(`:736-738`)。1 回の読みは gh が 3 回と ls-remote が 1 回で、それぞれ `ci.timeout` と `regateGitTimeout` で区切られ、締め切りを越えるのはその分だけ(merge の待ちと同じ)。
- ls-remote の分かれ: `regateReadOriginTip` の err は ctx の終わりだけ(`regate_git.go:79-80`)なので、poll の `interrupted()`(`regate_ci.go:752-753`)は正しい。tip が HEAD なら眠り続ける(GitHub の遅れ)。tip が別のコミット、ブランチがない、読めないなら poll を止め(`:754-755`)、`regateCIRecheck` が lock の下で読み直して断る。読めなかった tip が lock の下で HEAD なら、`regateChecks` の結果で終わる(pending なら exit 6、待ちは再開しない)。merge の待ちも「the read fails」で終わる(`merge_wait.go:29-30`)ので、扱いはそろっている。
- merge の道で PR の head が merge の前の先頭のままのときは ls-remote を打たない(`readRegateChecks` の `head == preMerge`、`regate_ci.go:417-419`)。origin が読めない状態でも待ちが続くことを subtest「the head the merge commit was made on, with origin not readable」が見る。
- CI が確かめていない head の記録: lock の下の `readRegateChecks` は head、checks、head の順に読み、2 回目の head が HEAD でなければ記録しない。その前に `regateCIRecheck` が worktree の HEAD と origin の先頭(ls-remote)を確かめる。残る窓は 2 回目の head から記録の append までと、head が A→B→A と動く形で、cycle 1 の L4 と同じく merge-order の head の比べが止める。
- L1 の次の手: 子孫の形では push するものがなく、相手のコミットも戻らないので、`--ff-only` の早送りが正しい。local の pass の同じ形(`regateCheckOriginHead`、`regate_git.go:142-145`)も同じ手を出す。テストは早送りのあとの regate が origin の先頭を gate して pass になるところまで走らせる。
- L8: `TestRegateCI_CommandsQuoteTheWorktreeAndTheBranch`(`internal/org/regate_ci_test.go:301`)は、空白・`$`・`;` を含むパスと `$`・`;` を含むブランチで、名指したコマンドを `sh -c` で走らせる。`cleanupGitCmd` は `-C` のパスとすべての引数を `cleanupShellWord` に通す(`internal/org/cleanup.go:374-380`)。
- F3 が移動だけであること: `go/parser` で 69f80d8f の `regate_ci_test.go` と aff0dcb0 の 4 ファイルのトップレベルの宣言を doc comment ごと切り出し、並べ替えて比べた。61 件がバイト単位で一致した。増えたのは 2 つの新しいファイルの冒頭の節の注釈(`regate_ci_merge_test.go:17-20`、`regate_ci_wait_lock_test.go:14-15`)と、import の分け方だけ。
- 4 つの `/director` skill と 4 つの `/org` skill の写しは、それぞれ md5 が一致する。
- `wc -l`(merge-base c03647f9 → HEAD): `regate_ci.go` 0→874(cycle 1 の時点で 679、e40b3466 で 813、f32fe553 で 846、02219639 で 862、69f80d8f で 874)、`regate.go` 763→994、`org_regate_test.go` 760→1267、`regate_merge_test.go` 1991→2105、`regate_test.go` 1545→1596。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。R2-5 は format string と `regateMergeLeft` から読んだだけで、git reset を失敗させる probe はしていない。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | R2-1 M1 の残りの台帳の行(395 行)が、直しのあいだに 800 行を越えた `internal/org/regate_ci.go` を名指さない。cycle 1 の 679 行が、F1 で 813、F1b で 846 と 862、F2 で 874 になった。395 行の払い方「`Regate` の ci の部分を `regate_ci.go` の helpers の後ろへ移す」は、すでに越えたこのファイルをさらに大きくする。同じ行の「the row above marked RESOLVED」は 106 行上の 289 行を指す位置の言い方で、すぐ上の 394 行は別の件。290 行に足した文の「`regate_merge_test.go` +85」は cycle 1 の数で、5c850e50 の時点ですでに +114 だった(F2 の L1 のテストで +29) | 上の `wc -l`、`docs/tech-debt/README.md:395`、`:290`、`:289` | 395 行に `internal/org/regate_ci.go`(0 → 874)を足し、払い方を「待ち(`regateCIWait`、`poll`、`regateCIRecheck`)を `internal/org/regate_ci_wait.go` に分け、`Regate` の ci の部分はそちらの helpers へ」のように、越えたファイルを小さくする形にする。「the row above」は「the `internal/org/regate.go` row marked RESOLVED 2026-10-16」と名前で指す。290 行の +85 を +114 にする |
| LOW | security | R2-2 L8 の後半(前からある regate の文まで広げないなら範囲を台帳に書く)が届いていない。新しい 397 行は共通の末尾と `--wait-checks` の断りだけを挙げ、前からある regate の文が `git -C <worktree>` とブランチを quote せずに出すことに触れない: `regateCheckOriginHead` の push と `merge --ff-only`、`regateResetHint` の `reset --keep refs/remotes/origin/<branch>`、`regateMergeTip` の `git -C %s merge %s`、`regateMergeLeft` の `status`・`log`・`reset --keep`。L1 の直しのあとは、同じ `merge --ff-only refs/remotes/origin/<branch>` を、ci の pass の断り(`regateCIPassBehindText`)は quote して、local の同じ形(`regateCheckOriginHead`)は quote せずに出す | `internal/org/regate_git.go:139-145`、`internal/org/regate_merge.go:143-149`、`:175`、`:424`、`:576-579`、`docs/tech-debt/README.md:397` | `regateCheckOriginHead` の 2 つのコマンドを `cleanupGitCmd` で包む(L1 の文の双子なので数行。テストのパスは `cleanupPlainWord` の文字だけなら期待は変わらない)。残りを包まないなら、397 行の説明と払い方に `regateResetHint`・`regateMergeTip`・`regateMergeLeft` の文も quote しないことを書き足す |
| LOW | readability | R2-3 `ralph org regate --help` の `--wait-checks` の段落は、F1b が足した待ちの 3 つ目の終わり方を書かない。段落は e40b3466 で書かれ、「every 30 s it only reads the PR's head and checks again. Once they are done (or gh fails), it takes the lock again, once」のまま。F1b(f32fe553・02219639、どちらもコミットの本文が空)のあとは、PR の head が別のコミットなら lock なしで `git ls-remote origin` も打ち、origin の先頭が HEAD でない・読めないときにも止まって取り直し、たいていは断る(exit 1)。help のテストはこの不足した文を固定している。中の doc も、`regateCIWait` の「takes again once, when it stops」とファイル冒頭の「and then takes the lock again, once」が、時間切れと割り込みでは取り直さないこと(`regateChecksWithWait` の doc と help には書いてある)を落としている | `internal/cli/org_regate.go:163-175`、`internal/cli/org_regate_test.go:825`、`internal/org/regate_ci.go:28-33`、`:594-595`、`:749-756` | help の文を「Once they are done (or gh fails), or the PR's head is another commit and origin's tip, read then with git ls-remote, is not HEAD or cannot be read, it takes the lock again, once, …」にし、テストの固定も合わせる。`regateCIWait` の doc は「takes again once, when the wait stops short of its limit」にする |
| LOW | readability | R2-4 取り直したあとの lock の下の読みがまた pending のとき、ChecksPending と stderr の行は「they were not done again when it read them with regate.lock taken again」と言う。poll が gh の失敗(`read.refused`)で止まった形でもこの文になるが、そのとき checks は一度も done になっていないので「again」が合わない。help は「Once they are done (or gh fails)」と 2 つを分けて書いている | `internal/org/regate_ci.go:698-699`、`:705-706`、`:747-748` | `last.refused != ""` のときは「the read after the sleep failed, and the checks were not done when it read them with regate.lock taken again」のように分ける |
| LOW | readability | R2-5 L2 の直しは、undo が git reset --keep で失敗した形(signal、時間切れ、exit が 0 以外)を考えていない。そのとき `res.NotUndone` はすでに reset の `regateStderrTail` で終わるので、断りは「…; the merge commit … was not undone, since git reset --keep … failed (exit N): … ; git said:\n  <reset の行>; git said:\n  <push の行>」になり、push の「; git said:」が reset の字下げした最後の行に付く。どちらの git の出力かは文から分からない。cycle 1 の L2 の Recommendation(undone を said の前へ)がこの形を見落としていた(reviewer の見落とし) | `internal/org/regate_ci.go:291`、`:296-297`、`internal/org/regate_merge.go:555-566`、`:576-579` | push の tail に command を名指す(「; git push said:」。`regateStderrTail` に見出しの引数を足すか、この分かれだけ別の helper にする)。あるいは `res.NotUndone != ""` のときだけ `said` を `failed` の直後に置く |

### Positive notes

- L3 の直しは merge の待ちと同じ形(読むだけ、終わりに 1 回取り直す)で、`TestRegateCI_WaitChecksGoesOnWhileAnotherHoldsTheLock` が、眠りのたびにほかの 1 つが lock を持ち、読みのあいだも持ち続ける状態で、待ちが続いて記録まで進むことを見る。前の形なら最初の取り直しで断られる。
- F1b の 2 つ目のコミットは、1 つ目の「別の head なら止まる」が GitHub の遅れでも待ちを終える穴を、lock なしの ls-remote で分けて塞いだ。読めない origin を lock の下で読み直す subtest まである。
- L1 のテストは文を比べるだけでなく、早送りのあとの regate が pass になるところまで走らせるので、名指した次の手が通ることを見ている。
- L8 のテストは、quote の形ではなく、名指したコマンドがそのまま貼って動くことを `sh -c` で見る。
- F3 は宣言 61 件がバイト単位で一致する移動だけで、分けた 2 つのファイルの冒頭に、fixture がどこにあるかの注釈を足している。

### Notes(所見にしないもの)

- poll の 1 回の読みの失敗(gh の一時的な失敗、ls-remote が読めない)で、60 分の待ちが早く終わることがある。lock の下で読み直して pending なら exit 6 で、`/director` skill は打ち直す(6 が 2 回続けば人に伝える)。merge の待ちも読みの失敗で終わるので、そろえた扱いとして所見にしない。
- テストと注釈は cycle 1 の所見を「self-review L3」「L4 of docs/reports/self-review-2026-10-29-regate-ci-checks.md」と引く。この report は書き換えずに後ろへ足しており、この節の ID は `R2-<n>` なので、引き先はぶれない。

### Tech debt identified

この run は report と insight event だけを書くので、台帳には書いていない。R2-1 と R2-2 は台帳の行の直し(R2-2 は `regateCheckOriginHead` を包めば数行のコード)、R2-3〜R2-5 は文の直しで、どれもこの PR の中で直せる。直さずに進めるなら、次の 1 行を台帳に足す(R2-1 は 395 行と 290 行の直しなので、ここには入れない)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| ci の regate の文の小さなずれ(self-review cycle 1 run 2、R2-2〜R2-5)。(a) R2-2: `regateCheckOriginHead`(`internal/org/regate_git.go`)、`regateResetHint`・`regateMergeTip`・`regateMergeLeft`(`internal/org/regate_merge.go`)の次の手のコマンドは worktree とブランチを quote しない。同じ `merge --ff-only` を `regateCIPassBehindText` は quote して出す。(b) R2-3: `ralph org regate --help` の `--wait-checks` の段落は、PR の head が別のコミットで origin の先頭が HEAD でない・読めないときにも待ちが止まることと、そのときの `git ls-remote` を書かない。(c) R2-4: poll が gh の失敗で止まったあとの pending も「they were not done again」と言う(`regateChecksWithWait`)。(d) R2-5: push が届かず undo の git reset も失敗した断りは、2 つの「git said:」を名前なしで続ける(`regateCIPush`) | LOW。挙動は正しい。(a) 特殊な文字を含むパスやブランチでは、貼ったコマンドが割れる。(b)〜(d) 読む人が待ちの終わり方や git の出力を読み違える | 文の直しは小さいが、固定したテストの期待も動く | `regateCheckOriginHead`・`regateCIPush`・`regateChecksWithWait` か、`ralph org regate --help` の次の変更 | `docs/plans/active/2026-10-29-regate-ci-checks.md`(Progress checklist)、`docs/reports/self-review-2026-10-29-regate-ci-checks.md`(cycle 1 run 2、R2-2〜R2-5) |

### Recommendation

- Merge: 可(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。HEAD f103777d について)。cycle 1 の L1・L3・L4・L6・L7 は解消、L5 は台帳へ送った。M1・L2・L8 は大半を解消し、残りが R2-1・R2-5・R2-2。
- Follow-ups: R2-1 は 395 行に `regate_ci.go` を足して払い方を直し、290 行の数を直す(台帳だけ)。R2-2 は `regateCheckOriginHead` を `cleanupGitCmd` で包むか、397 行の範囲を広げる。R2-3〜R2-5 は文の直しで、R2-3 は help のテストの固定も合わせる。直さずに進めるなら、上の 1 行を台帳に足す。
