# Self-review report: gate-lock-accept

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-gate-lock-accept.md`(issue #309)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `95b87a62`、base は `origin/main`。merge コミット `ae372953` で入った main 側の変更は含まない)。71 ファイル、6474 行追加、518 行削除(計画ファイルを含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

読んだもの。

- 新しいファイルの全部: `internal/org/gate_lock.go`、`gate_load.go`、`gate_accept.go`、`internal/cli/org_gate_accept.go`
- 差分の全部と、その前後: `internal/org/lockfile.go`(`waitHeldFileLock`、`openLockFile`)、`gate.go`(`Gate`、`gateCheck`)、`gate_run.go`(`runGateScripts`、`runGateScript`)、`gate_record.go`、`regate.go`(`Regate` の 491〜570 行、`regateGate`、`writeLockMarker`)、`regate_merge.go`、`approve.go`(`seatPaneTripwire`、`approverName`)、`seat.go`、`watch.go`、`mergeorder.go`(`collectMergeTargets` の全部、`classifyMergeTarget`、`classifyAccepted`、`classifyGatePassed`、`mergeOrderListNotes`)、`merge.go`(`checkMergeBranch`、step 9 と origin の読み直しの文)、`merge_run.go`(`mergeLocked`、`recordMerge`、`otherHeadMerged`)、`merge_wait.go`(`orgWaitEnd`)
- CLI の差分の全部: `org.go`、`org_gate.go`、`org_merge.go`、`org_merge_order.go`、`org_regate.go`、`status.go`、`org_director.go`、`org_director_report.go`
- `internal/config/config.go` と `templates/base/ralph.toml` の `max_load`、`internal/org/main_test.go` の固定値
- `internal/org/prompts/reviewer.md` の差分と、leader の gate の TASK のコマンド(`leader.md` 509 行の `> .harness/state/gate-{{ORG_ID}}.out 2>&1`)
- director skill と org skill の追加行、4 つの写しの `cmp`(どちらも 4 つとも同じ)
- テストのうち、D8 の実際の process のテスト(`gate_lock_test.go` 370〜494 行)と、新しいテストの一覧(`gate_lock_gate_test.go`、`gate_load_test.go`、`gate_accept_test.go`)。`t.Parallel()` は新しいテストに無い(パッケージ変数の差し替えと競合しない)
- `docs/tech-debt/README.md` の追加 5 行を差分と両方向で突き合わせた。5 行とも 5 列(`awk -F'|'` で 7)
- `git diff --check origin/main...HEAD` は rc 0。追加行の不可視 Unicode は 1 か所だけ(下の L8)

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- ロックの順: Gate は gate.lock → manifest.lock、local の Regate は gate.lock → regate.lock → manifest.lock、merge と cleanup は regate.lock → manifest.lock、GateAccept は manifest.lock だけ。regate.lock か manifest.lock を持ったまま gate.lock を待つ道は無いので循環しない。Regate の defer は gate.lock を先に登録するので、外す順は regate.lock → gate.lock
- Gate の検査の分け方: ロックの前の検査(`gateOrgLife` と `gateCheckWorktree`)を、ロックのあとに `gateCheck` で 2 つとも打ち直し、記録に使う `life` は 2 回目のもの。ロックの前の検査だけで終わる確かめは無い
- D8 の fd の受け渡し: `cmd.ExtraFiles` に渡した `*os.File` は os/exec が閉じない(呼び手のもの)。ralph が開く file は CLOEXEC なので、負荷を読む `sysctl` の子に gate.lock は渡らない。正常の終わりは `gateLock.release` の LOCK_UN でロックが外れ、`TestGateLock_ReleaseFreesTheLockAChildStillHasOpen` がそれを見る。中断のときも defer で外れる
- 負荷の goroutine: `runGateScripts` は start の直後に `defer load.sampler.stop()` を登録し、`finish` も `stop` を呼ぶ(`sync.Once`)。`highest` と `known` は `<-s.exited` のあとにだけ読むので競合しない。ctx が切れても goroutine は終わる。`stop` を待たせるのは進行中の 1 回の読み取りだけ(darwin で最長 2 秒)
- `max_load` の待ち: `timer.Stop()` は ctx の分岐で呼ばれ、上限の最後の 1 回は `min(poll, cap-経過)` で寝る。読めない負荷は待たずに始める
- `gate_accepted` の書き手と読み手: 値に空白を含みうる `failed_log` と `note` は `strconv.Quote` で最後に置き、読み手は `strconv.QuotedPrefix` で 2 つを読み、note の後ろに何かあれば読めない扱いにする。読み手は reason を `gateAcceptableReason` で、note を `gateAcceptNoteProblem` で確かめ直すので、手で書いた `head_moved` の受け入れや改行入りの note は効かない
- accept の check-then-append: 台帳の読み取り、座席の pane の確かめ、`gateAcceptFailure`、追記が 1 回の `withManifestLock` の中にある。`recordGate` も manifest lock の中で追記するので、承認待ちの間の gate の打ち直しは `--log` の照合で断られる
- merge-order の結び: `accepts` は org_id、head、base、base_ref、branch、event、reason、ts、log の全部を比べる。readable な gate の記録が枝に付くたび、読めない記録が org の枝に付くたびに `accepted` を消し、壊れた行が gate の後ろにあれば消す。どちらも「知らせる側」(failed か unreadable)に倒れる。`classifyMergeTarget` は unreadable と no_gate を先に判定し、受け入れた失敗も `classifyGatePassed` を通すので、rebase・regate・no_remote・wrong_base を飛ばさない
- merge が `accepted` を通す道: `checkMergeBranch` は class だけを広げ、org が走っていないこと、PR の head と base、checks、保護、origin の読み直しは同じ。`mergeLocked` はロックを取るたびに `checkMerge` をやり直す。`r.accepted` は class が accepted のときだけ設定し、そのとき `t.accepted` は nil にならない(`classifyAccepted` だけが accepted を返す)
- 知らせを黙らせる変更: `leaderActivityEventCount` は `gate_accepted` を数えない(数えると基準線が動くだけで、ALERT を消す方向には効かない)。`EventGateFailed` の読み手は merge-order と regate の CLI だけで、ほかの知らせの道は変わらない。`pr_merged` の末尾の ` gate=accepted` は `followup.go` の最初の `branch=` と `head=` の読み取りを変えない
- 表示の escape: accept の断りは `PrintableInboxText`、`ralph org gate accept` の出力は `inboxField`、merge-order の Detail は `inboxField`、merge の `GATE:` の行は `PrintableInboxText` を通る。保持者の印は `Verb` と `StartedAt` の制御文字を断り、org_id と branch は `%q`。`kill(pid, 0)` に渡す pid は 0 以下を断るので、プロセスグループには届かない
- `ralph org gate` は `cobra.NoArgs` なので、`ralph org gate acept` のような打ち間違いは gate を走らせずに断られる
- Go のコメントが `docs/plans/active/` の計画を指す件(追加 26 か所)は、`internal/` に既に 260 か所あるこの repo の慣例なので挙げない

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `accepted` の枝の merge でも、断りと異常の文が「the head the gate passed」と言う。受け入れた枝の head は gate が落ちた head なので、事実と違う。`otherHeadMerged` の文(PR が別の head で merge されたとき)は「a head the gate did not pass was merged」とも言い、受け入れた失敗の merge の後始末で人が読む文になる | `internal/org/merge.go:580`(step 9 の head の不一致)、`merge.go:741`(origin の読み直し)、`merge_run.go:481` と `:484`。`merge.go:220` の doc は既に「the gate's head」と書いている | 4 つの文を「the gate's head」か「the head of the gate record」に揃える。受け入れた枝だけ文を分ける必要はない |
| LOW | maintainability | `--log` は記録の log と文字列で比べるだけ。同じファイルを相対パス、`~`、macOS の `/tmp` と `/private/tmp` のように別の書き方で渡すと断られ、その断りは「a newer gate may have replaced the failure ... ask the human again」と言うので、director は同じ失敗を人に聞き直しに行く | `internal/org/gate_accept.go:331-334` | 断りの文に「--log は記録の書き方のまま(`<r.Log>`)渡す」を足すか、`os.SameFile` で同じファイルなら別の文で断る。受け入れる側には広げない(D5 の取り違え防止を保つ) |
| LOW | security | skill の例が `--log <LOG: のパス>` を quote していない。`--reason` は `'…'` で囲み `'` を含めないと書いているのに、log のパスは「空白を含みうる」ので記録では quote している。台帳のパスに空白があると引数が割れる。`cobra.NoArgs` で断られるので誤って記録はしないが、1 回目の accept が失敗する | `.claude/skills/director/SKILL.md:1278`、`.claude/skills/org/SKILL.md:376`(と 3 つずつの写し)。`internal/org/gate_record.go` の鍵の説明(log は `strconv.Quote` で空白を含みうる) | 例を `--log '<LOG: のパス>'` にする |
| LOW | maintainability | local の Regate は `regateFirstCheck` より前に gate.lock を待ち(上限なし)、続けて `max_load` を最大 30 分待つ。Gate は検査を先に打つので、断られる gate は待たない。Regate は、ledger に枝が無い、org がまだ走っている、といった 1 秒で決まる断りも、ほかの gate の終わりを待ってから出す。tech-debt の行 (c) は「スクリプトを走らせない regate(`up_to_date`、`push_only`)も待つ」だけを書き、断られる regate には触れていない | `internal/org/regate.go:518-533`(gate.lock と max_load)と `:552`(最初の検査)。`internal/org/gate.go` の `gateCheck` を 2 回打つ形との比較。`docs/tech-debt/README.md` の gate.lock の行 | ロックなしで台帳だけを読む前の検査(枝の結びつきがある、org が走っていない)を gate.lock の前に足すか、tech-debt の行 (c) に「すぐ断られる regate も待つ」を足す |
| LOW | readability | `ci_failed` も受け入れられるが、受け入れても `ralph org merge` は PR の checks がすべて pass か skipping であることを確かめ直すので、落ちた check を GitHub で打ち直して通さない限り merge は断られる。help と skill はこの帰結を書いていない。merge が止まる側に倒れるので害は小さいが、人に聞いて受け入れたあとで断られる | `internal/cli/org_gate_accept.go:31-35`(`ci_failed` を受け入れられると書く)、`internal/org/gate_accept.go:186-191`、`ralph org merge --help` の checks の確かめの文、director skill 1597〜1605 行 | accept の help と director skill の受け入れの節に「`ci_failed` を受け入れても、merge は PR の checks が通るまで断る。落ちた check を打ち直す」を 1 文足す |
| LOW | naming | 受け入れの理由の文に 5 つの名前がある: フラグ `--reason`、フィールド `GateAcceptance.Note`、Details の `note=`、CLI の `NOTE:`、JSON の `gate.accepted.reason`。同じ JSON の `gate.reason` は gate の失敗の理由なので、`reason` が 2 つの意味を持つ。ほかに、`readyRegated` が accepted の枝も数える、`gateLockDuration` が max_load の待ちの時間も整える、Regate の `run` は `gateRunOptions` で Gate の `run` は `gateRun` | `internal/org/gate_accept.go:79`、`:107`、`internal/org/mergeorder.go` の `MergeOrderAcceptance.Reason`、`mergeorder.go:1352`、`internal/org/gate_load.go:327-396`、`internal/org/regate.go:518` | JSON の鍵を `note` にするか、フィールドを `Reason` に揃える(JSON は出したあとで変えにくいので、変えるなら release の前)。`readyRegated` は `mergeableRegated` などに、`gateLockDuration` は `gateWaitDuration` などに |
| LOW | maintainability | 数値が文にそのまま書かれ、値の定義と結ばれていない。500 は `GateAcceptReasonMaxRunes` だが help の 2 か所は literal。30 秒、30 分、5 分は `gateMaxLoadPoll`、`gateMaxLoadCap`、`gateLoadSampleInterval`、`gateLockRemind` だが、gate と regate の help、2 つの skill、reviewer の prompt、テンプレート、config のコメントに literal で書かれ、CLI の help のテストは文だけを固定する | `internal/cli/org_gate_accept.go:46`、`:114`、`internal/cli/org_gate_test.go:936-949`、`internal/cli/org_regate_test.go:863`、`internal/org/gate_load.go:56-66`、`internal/org/gate_lock.go:35-42` | help の数値を exported な定数から組み立てるか、`internal/org` のテストで定数と help の数値を突き合わせる。値を変えるときに直す場所が 7 か所あることを、少なくとも定数のコメントに書く |
| LOW | readability | 小さな重複と見えない文字。`writeGateLockContent` は `writeLockMarker` の truncate と `WriteAt` を写している。CLI の `printGateLoad` は CPU 数の書き方を自前で持ち、log の `gateCPUs` と違う(1 CPU のとき CLI は「1 CPUs」、不明のとき「CPU count unknown」と「an unknown number of CPUs」)。テストの 1 行に U+2028 がそのまま入っていて、エディタでは見えない(ほかのテストは ` ` と書く) | `internal/org/gate_lock.go:163`、`internal/org/regate.go:789-804`、`internal/cli/org_gate.go:224`、`internal/org/gate_load.go:157`、`internal/org/gate_accept_test.go:174`(`watch_scope_test.go:143` は ` `) | 中身を書き換える関数を 1 つにする。CPU 数の文を 1 つの関数から出す。テストの文字列を `"one two"` にする |

参考(指摘にはしない)。

- 待つ側の log の見出しの保持者は、最後の poll で読んだ印になる。保持者の `release` は印を空にしてから LOCK_UN するので、その間(マイクロ秒)に最後の poll が当たると、見出しは「a gate whose marker is not written yet」になる。印が空のときは前に読めた保持者を残すと、この窓も消える
- scripts は gate.lock を書き込みのできる fd 3 として受け取る。fd 3 に書く script(用意していない fd 3 に出力する harness など)は印を壊すが、壊れるのは待つ側の表示だけで、ロックには効かない。scripts は元から repo のコードとして同じ権限で走るので、権限の広がりは無い

## Positive notes

- Gate はロックの前の検査をロックのあとに 2 つとも打ち直している(`gateCheck`)。ロックを分けたときに片方の検査だけが残る形になっていない
- accept は読み取りから追記までを 1 回の manifest lock で済ませ、`--log` の照合で承認待ちの間の打ち直しを断る。2 つの accept が同時に走っても 1 件になる
- 受け入れの効き方が狭い。org_id から log までの 9 つが一致したときだけ効き、新しい gate の記録、読めない記録、後ろの壊れた行のどれでも消える。迷う場面はすべて failed か unreadable に倒れる
- D8 を実際の process を SIGKILL するテストで確かめている。テストの後始末は script のプロセスグループも止める
- 負荷の goroutine の終わり方(done、exited、`sync.Once`、ctx)が型のコメントに書かれ、コードと合っている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (候補)local の `ralph org regate` は最初の検査より前に gate.lock と `max_load` を待つので、すぐ断られる regate も待つ | ほかの gate が長いと、director の regate が断りを知るのも遅れる | この self-review では決めない。直すか、既存の gate.lock の行 (c) に足すかは実装側が決める | 断られる regate が gate.lock を長く待った報告 | 本 report の LOW(Regate の待ちの順) |

_この run では `docs/tech-debt/README.md` に行を足していない(指示により、この run のコミットは本 report だけ)。上の候補は、直さない場合に既存の行 (c) へ足す。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8)
- Follow-ups:
  - 安く直せる順に、merge の文の「the head the gate passed」(LOW 1)、skill の例の `--log` の quote(LOW 3)、`--log` の断りの文(LOW 2)、`ci_failed` の受け入れの 1 文(LOW 5)
  - JSON の `gate.accepted.reason` の名前(LOW 6)は release の前に決める
  - Regate の待ちの順(LOW 4)は直すか tech-debt に足す
  - 計画の申し送りどおり、README の merge-order の class の一覧と `templates/base/ralph.toml` の auto_merge のコメントは `/sync-docs` で扱う(本 review の範囲外)
