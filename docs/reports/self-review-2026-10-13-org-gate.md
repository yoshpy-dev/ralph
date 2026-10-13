# Self-review report: org-gate

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-13-org-gate.md(承認済み、digest 69f088275e54)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: diff の品質だけ。`git diff 1bddc871...HEAD`(HEAD fbed9f05、6 コミット、26 ファイル、+4700/-206)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の Design decisions B1〜B11 と Progress checklist の Notes にある決定(`log=` を最後に置き `strconv.Quote` で囲む、`origin/HEAD` を完全な ref 名で読む、記録の前に org の命を確かめ直す、終了コードが 0〜255 でなければ失敗にする、など)は、欠陥として数えていない

## Evidence reviewed

- 読んだ差分: `internal/org/gate.go`(931 行、全体)、`internal/cli/org_gate.go`(全体)、`seat.go`・`watch.go`・`spawn.go`・`cli/org.go` の hunk、`scripts/run-verify.sh` の全 hunk と全文、`templates/base/scripts/run-verify.sh`(root と `cmp` で同一)、`internal/org/prompts/leader.md` の全文と `reviewer.md` の hunk、`.claude/skills/org/SKILL.md` の hunk(`.agents/` 側と `templates/base/` 側の 2 組は `cmp` で root と同一)、`docs/quality/quality-gates.md` の行、AGENTS.md、仕様の注、`docs/tech-debt/README.md` の追加 7 行(221〜227 行目)
- 読んだテスト: `internal/org/gate_test.go`(fixture、拒否の表、記録、base の決め方、ロック、中断、環境、Details の往復、座席の扱い、見張りの数え方)、`gate_runverify_test.go`、`internal/cli/org_gate_test.go`(fixture、往復、中断、record 時の拒否、help)、`prompts_test.go` の追加分、`approval_enforce_test.go` の追加分、`tests/test-run-verify-concurrency.sh` の全文
- 確かめたこと(読むだけでは決められなかったものは、使い捨ての repo で流した)
  - `git diff --check` は空。追加行に TODO・FIXME・デバッグ出力・秘密情報はない。`gate.go` が使う `absPath`、`approverName`、`pathErrorCause`、`sameFeature`、`lastDisbandedIndexes`、`herdrPaneIDEnv` は既存で、定義を確かめた
  - `syscall.Setpgid`・`syscall.Kill` に build tag はないが、`internal/org/lockfile.go` が `syscall.Flock` を同じく tag なしで使っており、`.goreleaser.yml` の `goos` は darwin と linux だけなので、新しい制約ではない
  - probe 1(M3): base(1bddc871)と HEAD の `run-verify.sh` を、8 秒かかる言語 verifier の stub を持つ repo に置き、`/bin/sh`(bash 3.2.57)と `/bin/dash` で TERM を送った。pid だけに送ると、base は 0.0 秒、HEAD は 8.1 秒(verifier が終わるまで)で、どちらも終了コード 143。プロセスグループ全体に送ると、base は 0.0 秒、HEAD は 0.1 秒で、終了コード 143。`TMPDIR` を空のディレクトリにして HEAD を 4 通り(2 つの sh、pid とグループ)走らせ、終わったあとに `ralph-verify-*` も `.harness/state/verify-scope.*.tmp` も残らなかった
  - probe 2(L2): 衝突した `git rebase` の途中で `git branch --show-current` は空を返し、`git switch feat` は `fatal: cannot switch branch while rebasing` で失敗した
  - `git grep` で `gateRecordFromDetails`・`isGateReason` の呼び出しを数えた。非テストの呼び出しは自身と互いだけ
- 確かめていないもの: テスト、`gofmt`、`golangci-lint`、`shellcheck`、`-race` は走らせていない。run-verify.sh のコメントにある「shellcheck 0.11 が SC2329 を見落とす」は確かめていない。leader のツール(Claude Code の Bash、Codex)が timeout のときに何を送るかは確かめていない

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 3、LOW 9。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | per-plan の org で、archive のあとに gate が `GATE: fail` ではなく「断り」を返し、直す手が衝突する rebase のとき、leader は implementer に TASK を頼めない。revert の手は `GATE: fail` の枝にしか書かれていない | 下の M1 | 「断られたとき」の枝に、rebase が衝突したら `git rebase --abort` のあと archive のコミットを revert してから implementer に頼む、を足す。SKILL の同じ手順(`.claude/skills/org/SKILL.md:780-788`)と、`prompts_test.go` の固定する語句も合わせる |
| M2 | MEDIUM | exception-handling | スクリプトを `Setpgid` で別のプロセスグループに入れたので、ralph が SIGINT と SIGTERM 以外で終わると(SIGKILL、SIGHUP)スクリプトの群れが残る。計画と雛形は「切られたら打ち直す」と書くが、残った群れと打ち直した gate が同じ worktree で重なる | 下の M2 | SIGHUP も `NotifyContext` に足す。SIGKILL は捕まえられないので、雛形の「打ち直す」の前に前の gate の群れが残っていないか見る手(`pgrep`、または log の更新)を書くか、gate の開始時に前の gate の log の使用中を検出する |
| M3 | MEDIUM | readability | `run-verify.sh` のコメントは「HUP、INT、TERM は 128 + 番号で終わり、EXIT trap を走らせる」と言うが、前景のパイプラインが終わるまで trap は走らない。pid だけに TERM を送ると、base は即座に終わり、HEAD は verifier が終わるまで(probe では 8.1 秒)終わらない。Case 6 は kill の直後に stub を解放するので、この違いを区別できない | 下の M3 | コメントを実際の動きに直す(グループ全体への信号なら即座、pid だけなら前景のパイプラインのあと)。Case 6 に「解放しないまま kill して、グループ全体に送れば 1 秒以内、pid だけなら verifier のあと」の 2 本を足す。pid だけでも止めたいなら、パイプラインを `&` と `wait` で走らせて trap で子を止める |
| L1 | LOW | maintainability | 既定のブランチの決め方が 3 か所で別の言い方になっている。gate は `origin/HEAD`、`main`、`master`(完全な ref 名で読む)。leader の手順 9 は「`ralph org gate` と同じ決め方」と書きながら `git symbolic-ref --quiet --short` を使う。reviewer の雛形は `master` を落とす | `gate.go:561-572`、`leader.md:164-168`、`reviewer.md:32-33`、計画 Notes(2026-10-13 の「実装で決めた細部」) | reviewer の雛形に `master` を足す。leader の文は「`--short` の出力が `remotes/origin/<名前>` のときは `remotes/` を除く」と書くか、`--short` を外して `refs/remotes/origin/` を除く形にそろえる |
| L2 | LOW | exception-handling | 断りの「次の手」が成り立たない場合が 2 つある。(a) 衝突した rebase の途中は HEAD が detached になり、断りは「`git switch <ブランチ>` を打て」と言うが、probe では `git switch` が失敗する。(b) origin がない repo で既定のブランチが見つからないとき、断りは `git fetch origin` を勧める | (a)`gate.go:440-442`、(b)`gate.go:464-466`(テスト `gate_test.go:477-484` は origin を外した状態で、この文を期待している) | (a)detached のときに `git rev-parse --git-path rebase-merge` などで進行中の rebase を見分け、`git rebase --continue` か `--abort` を言う。(b)origin がなければ fetch を言わず、既定のブランチを作る手を言う |
| L3 | LOW | naming | `gateGitMust` は `Must` で始まるが、panic せずエラーを返す。`internal/` の非テストコードで `Must` を含む関数の定義はこれ 1 つ(`regexp.MustCompile` のような panic の慣習と逆) | `gate.go:662` | `gateGitOut` のように、返すものを名前にする |
| L4 | LOW | maintainability | CLI が org 層の値を別の定数で持つ。`gateFailedScripts` はスクリプトのパスを、`gateDirtyListLines` は 20 を、それぞれ書き写している | `org_gate.go:108-115` と `gate.go:40-41`、`gate.go:58`。コメントは「org 層と同じ」と書いてあるだけで、ずれたときに落ちる検査がない | org 層から `GateStaticScript`・`GateTestScript` と上限を export する。または CLI の表示用の行を `GateResult` に持たせる |
| L5 | LOW | exception-handling | スクリプトが普通に終わったあと、プロセスグループは掃除されない。背景に残したプロセスは gate の記録のあとも動き、pipe を握ったままだと 5 秒待って `ErrWaitDelay` を log に 1 行書くだけで通る。中断のときだけ `stopGateGroup` が群れを止める | `gate.go:802-820`(中断の分岐だけが `stopGateGroup` を呼ぶ)、`gate.go:817-819` | 終了後に `kill(-pgid, 0)` で群れが残っていれば、`tree_changed` の仲間として扱うか、SIGTERM と SIGKILL で止めて log に書く。しないなら `runGateScript` の doc に「普通の終了では群れを止めない」と書く |
| L6 | LOW | maintainability | 見張りのコメント(c)の前提「座席は台帳に書けない」は、codex の leader 以外の座席が `--add-dir` を持たないことだけが根拠で、claude の座席には当てはまらない。`permissions.go` は同じ箇所で「claude has no such sandbox」と書いている | `watch.go:1208-1211`、`permissions.go:150-157`。B3 も同じ前提 | コメントを「leader が打つ。codex の座席は台帳に書けないが、claude の座席は書ける(permissions.go)ので、座席が gate を打てば leader の動きとして数えられる」に直す |
| L7 | LOW | maintainability | 追加した tech-debt の行(226 行目)は、既存の行(155 行目)を「narrows」と言うが、155 行目の本文の「nothing in `internal/org` ... runs or checks the gates」は、`gate.go` が両スクリプトを走らせるようになったので偽になった。155 行目は今回の PR が触れておらず、行の Trigger の「verb か hook か leader 側の検査」のうち verb が一部入ったことも書かれていない | `docs/tech-debt/README.md:155`、`:226` | 155 行目の本文に「feature org の PR 前の gate は `ralph org gate` が記録する(226 行目)。reviewer の座席が走らせる検査は雛形頼みのまま」と 1 文足す |
| L8 | LOW | readability | `gate.go` は 931 行、`gateCheckWorktree` は 68 行で、検査、base の決め方、git の補助、スクリプトの実行、tail の buffer、記録が 1 ファイルに入っている。`gateRecordFromDetails` と `isGateReason` は 7c が使うまで非テストの呼び出し元がない(コメントは 7c のためと書く) | `gate.go:203`、`:234`、`:411-478` | 今すぐは要らない。7c で読む側を足すとき、実行(`runGateScript`、`gateOutput`)と記録(`gateRecord*`)を別ファイルに分ける |
| L9 | LOW | maintainability | 雛形の手順 12 は gate を「バックグラウンドで走らせて終わりを待つ」ことを勧めるが、背景のときに stdout の `HEAD:`・`BASE:` をどう受け取るかを書いていない。手順 14 はその 2 行を RESULT に写せと言う | `leader.md:192-195`、`:196-198`、`:222-223` | 「背景なら出力をファイル(`.harness/state/` の下、ignore 済み)に書き、終わってから読む」と 1 文足す |

### M1 の根拠

- `leader.md:176-185`(手順 10)は archive を別のコミットにし、その SHA を控える。`leader.md:199-202`(手順 12 の「断られたとき」)は「stderr に書かれた次の手に従う(作業ツリーが汚れている、rebase していない、など)」とだけ言う。revert は `leader.md:203-209` の `GATE: fail` の枝にしかない
- gate が `refs/remotes/origin/<既定のブランチ>` を HEAD が含まないと断るのは、手順 9 の fetch のあとに main が進んだときで、report の作成、secret scan、手順 12 までの間に起こる(`gate.go:482-498`)。断りの次の手は `git fetch` と `git rebase`。この rebase が衝突すると、leader は手順 9 の文(`leader.md:170-173`)で implementer に TASK を頼むが、per-plan では計画が `docs/plans/archive/` にあるので TASK は承認の検査で断られる(`approval_enforce_test.go` の `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored` が、この断りを「no such file or directory」で固定している)
- 手順 12 の `GATE: fail` の枝は revert を書いている(B11)ので、同じ問題の `GATE: fail` 側は塞がれている。「断り」側は B11 の対象から漏れた。SKILL の締めの手順 4(`.claude/skills/org/SKILL.md:780-788`)も同じ

### M2 の根拠

- `gate.go:792` が `Setpgid: true` を付け、`gate.go:796-800` の `Cancel` と `stopGateGroup` が SIGTERM から SIGKILL へ上げる。この経路は ctx が終わったときだけで、ctx を終わらせるのは `org_gate.go:92` の `signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)` だけ。テスト(`gate_test.go:748-801`、`org_gate_test.go:654-719`)は ctx の取り消し、SIGINT、SIGTERM を調べていて、ralph が SIGKILL か SIGHUP で終わる場合は調べていない
- スクリプトの群れは ralph のグループの外にあるので、ralph のグループに送る killpg や端末の SIGHUP は届かない。ralph が SIGKILL で終わると、群れは init の子として動き続ける(OS の動きで、probe はしていない)
- 計画の Risks(`docs/plans/active/2026-10-13-org-gate.md:144`)と雛形(`leader.md:192-195`)は、leader のツールの timeout(Claude Code の Bash は既定 2 分、最大 10 分)で gate が切られることを前提にして、「何も記録されないので打ち直せばよい」と書く。ツールがどの信号で切るかは未確認。SIGTERM なら ralph が群れを止める。SIGKILL なら群れが残り、打ち直した gate と同じ worktree で `go test` などが重なる
- 重なっても結果が混ざらないのは B10 の範囲(終了コード、範囲、docs だけの印)で、証拠 log(`docs/evidence/verify-<秒>.log`)、テストが使う port や temp ファイル、worktree への書き込みは対象外。重なりで起きるのは、前の群れがあとから worktree を汚して `tree_changed` で落ちる、通るはずの gate が落ちる、という偽の失敗が主で、偽の通過は `gateCheckAfter` が塞いでいる
- gate 同士の排他は計画で退けた(B10)。ここで求めるのは排他ではなく、SIGHUP の追加と、残った群れへの手

### M3 の根拠

- `run-verify.sh:35-37` のコメント: 「HUP, INT, and TERM exit with 128 + the signal number, which runs the EXIT trap」。trap は `run-verify.sh:50-53`。ブロックは `run-verify.sh:71-285` の `{ ... } 2>&1 | tee` で、前景のパイプライン
- POSIX の sh は、前景のコマンドを待っている間に受けた信号の trap を、そのコマンドが終わってから走らせる。probe 1 で bash 3.2 と dash の両方がそうだった。pid だけに TERM を送ると HEAD の run-verify.sh は 8.1 秒かかって 143 で終わる(base は trap がなく 0.0 秒で死ぬが、ブロックは孤児になって 8 秒動く)
- gate の中断はグループ全体に TERM を送るので 0.1 秒で終わる(probe 1)。M3 は gate の動きではなく、コメントの主張と、単独の `kill <pid>`・`timeout` の `--foreground` などの使い方に関わる
- `tests/test-run-verify-concurrency.sh:370-372` は `kill -TERM "$term_pid"` の直後に `: > "$term_release"` で stub を解放してから、終了コード 143 と一時ファイルの掃除を調べる。信号がすぐ効いても、verifier が終わってから効いても同じ結果になるので、コメントの主張を固定していない

## Positive notes

- gate の中断: `exec.CommandContext` の `Cancel` を差し替えて、グループに SIGTERM を送り、猶予のあとに SIGKILL を送る。TERM を無視するスクリプトと、その子(`sleep 300 &`)を使うテストが、両方の経路と「`run-test.sh` が走らない」「manifest が変わらない」「log に interrupted が残る」を固定している(`gate_test.go:748-801`)
- ロック: スクリプトの間は manifest のロックを持たず、記録の直前に同じロックの中で manifest を読み直し、binding と命(最後の `disbanded` の位置)が同じ場合だけ書く(`gate.go:897-931`)。disband、再 start、別の機能への束ね直しの 3 通りを、log を残したままの断りとして試験している(`gate_test.go:870-919`)
- 断りの文は、理由のあとに 1 つの次の手が付いている(ただし L2 の 2 件を除く)。dirty の断りは 20 行で打ち切り、残りの数を言う。拒否は何も記録せず log も作らない(`assertGateRefused` が `GateResult` の他の欄、manifest、`gates/`、スクリプトの呼び出しを全部調べる)
- Details は `log=` を最後に置いて `strconv.Quote` で囲み、読む側は最初の ` log=` で切る。`/my state/a "quoted" log=x.log` を含む往復と、重複キー、短い SHA、大文字、`refs/` の外の `base_ref`、未知の reason を断る表がある(`gate_test.go:1045-1095`)
- 環境: 名前の接頭辞だけで一致を決めるので、`X=RALPH_VERIFY_BASE=1` や `KEEP_RALPH_VERIFY_BASE` は残る。出荷する本物の `run-verify.sh` を使うテストが、`RALPH_VERIFY_BASE=HEAD` でも言語の検証が走ることを対照つきで固定している(`gate_runverify_test.go:72-116`)
- `run-verify.sh`: 一時ファイルは EXIT trap の前に変数を空で用意してから `mktemp` を呼び、trap の本体は空の変数を飛ばし、`|| true` で終わるので、`set -e` 下で終了コードを変えない。終了コードのファイルが空、非数、256 以上なら失敗にする。probe 1 で、TERM のあとに一時ファイルが残らず、bash 3.2 と dash で 143 になることを確かめた
- テストの作り: 実 git の一時 repo、固定の clock、固定の log 名、`isolateGitEnv` で個人の git 設定から切り離し、herdr と agmsg と osascript は呼ばない(CLI の往復は `setupOrgStubPATH` の stub)。待ちは 10 秒か 15 秒の上限つきの poll で、`time.Sleep` を結果の判定に使っていない。`signalSelf` は自分の channel を先に登録してから自分に信号を送るので、gate の handler が外れていてもテストの binary は死なない
- 写し: `scripts/run-verify.sh` と `templates/base/scripts/run-verify.sh` は同一。`/org` skill の 4 面、quality-gates の 2 面の追加分は同じ文。`templates/base/` に `internal/`・`cmd/`・計画のパスの参照は入っていない

## Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。この PR が足した 7 行(`README.md:221-227`)は、計画の B2・B4・B6・B9・B10 と Non-goals に沿っている。MEDIUM 3 件は修正で直せる大きさなので、行にはしない。LOW は、修正に回さなかった分を次の回の終わりに 1 行にまとめる(下の Follow-ups)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この回では足さない) | — | — | — | — |

## Known gaps

- gate は、検証される worktree の中の `scripts/run-static-verify.sh` と `scripts/run-test.sh` を走らせる。implementer がその 2 つを書き換えれば、通る記録が取れる。reviewer が同じスクリプトを同じ worktree で走らせる今の作りと同じで、計画は扱っていない。quality-gates の行は「recorded mechanically」と書くが、記録するのは「その worktree のスクリプトが通った」ことまで
- 環境は `RALPH_VERIFY_` と `HARNESS_VERIFY_` の接頭辞の名前だけを除く。ほかの変数は渡る(たとえば `scripts/secret-scan-branch.sh:145` の `GITHUB_BASE_REF`)。leader の環境に入ることはまずないが、B9 の文は「範囲を狭める変数はない」と読める
- codex の leader の sandbox でスクリプトと fetch が動くかは未確認(tech-debt に行がある)。M2 の「ツールが timeout で何を送るか」も未確認

## Recommendation

- Merge: yes(cycle 2 の時点、branch 全体。CRITICAL・HIGH・MEDIUM はない。LOW は cycle 2 の 6 件(下の「Cycle 2」の C2-1〜C2-6)で、どれも動きと記録の値を変えない。cycle 1 の MEDIUM 3 件と LOW 9 件は e0adb359・aeba07cb で直っていることを HEAD で確かめた(L8 の半分と M2 の SIGKILL は、前者が stage 7c、後者が tech-debt の行に載る)。cycle 1 の判定は「yes、MEDIUM 3、LOW 9」で、上の所見の表はその時点のまま残す)
- Follow-ups:
  1. M1: 手順 12 の「断られたとき」と SKILL の手順 4 に、衝突した rebase のあとの archive の revert を足し、`prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgGate` の語句と順序の検査を合わせる
  2. M3: `run-verify.sh` のコメントを実際の動きに合わせ、`tests/test-run-verify-concurrency.sh` の Case 6 を、解放しないまま kill する形に直す(`/bin/sh` と dash の両方)。root と `templates/base/` の 2 つを同じに保つ
  3. M2: `signal.NotifyContext` に SIGHUP を足す。ツールの timeout で残る群れへの手を雛形に書くかどうかは、実機での確認(leader のツールが何を送るか)のあとに決める
  4. L1〜L9: 直せるものは 1 と同じ修正にまとめる。次の回で残る分は、tech-debt に 1 行にして、L7 の既存の行の文もそのとき直す

## このレビューのあとの修正(orchestrator が追記)

このレビューのあと、/verify の前に、所見を e0adb359(`fix: address org-gate self-review findings`)と aeba07cb(パッケージの doc の 1 行)で直した。上の所見の表と判定は、レビューした fbed9f05 の時点のまま残す。

- M1: archive のコミットのあとに implementer へ TASK を送るときは、rebase の途中なら `git rebase --abort`、続けて archive のコミットを revert してから送る、を手順 10 の共通の規則にした(`GATE: fail` と、直すのに implementer の手が要る断りの両方)。skill の 4 面と `prompts_test.go` も合わせた
- M2・L9: CLI が SIGHUP も捕まえる。雛形は gate をバックグラウンドで走らせ、出力を `.harness/state/gate-<org_id>.out` に書いてから読む形にし、打ち直す前に前の gate のスクリプトが終わるのを待つと書いた。SIGKILL で残るスクリプトの群れには仕掛けを足さず、tech-debt の行にした
- M3: `run-verify.sh` のコメントを実際の動きに直し、Case 6 をグループへの TERM と pid だけへの TERM の 2 つに分けた(`/bin/sh` と dash)
- L1〜L8: 既定のブランチの決め方を 3 か所でそろえた(L1)。rebase の途中と origin がない repo の断りの次の手を直した(L2)。`gateGitMust` を `gateGitOutput` に改名した(L3)。スクリプトのパスと一覧の上限を org 層から export した(L4)。スクリプトが残したプロセスを止めて log に書くようにした(L5)。見張りのコメントを直した(L6)。tech-debt の既存の行に 1 文足した(L7)。スクリプトの実行を `gate_run.go` に分けた(L8)
- レビューの Known gaps のうち、worktree のスクリプトをそのまま走らせることと、除く環境変数の範囲は、tech-debt の行にした

## Cycle 2

- Date: 2026-10-13
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 回の最後)
- Scope: `git diff 789c8864..HEAD`(HEAD 423a9f47、2 コミット、4 ファイル、+259/-11)。b04289c1(`internal/org/gate.go` の `recordGate` と `Gate` の doc、`internal/org/lockfile.go` の `withManifestLockContext` と `withFileLockContext`、`internal/org/gate_test.go` の新しいテスト 2 つと補助 2 つ)と、計画の進捗に 1 行足した 423a9f47。加えて、cycle 1 の追記が頼っている e0adb359(18 ファイル、+897/-439、`gate_run.go` の新設を含む)と aeba07cb を、cycle 1 の各所見が HEAD で直っているかを確かめる範囲で読んだ。この 2 つは cycle 1 のレビュー(fbed9f05 まで)のあとに入り、/verify、/test、/cross-review(Codex、bf80094a)は通ったが、self-review は diff の品質を見ていない。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない

### 読んだものと確かめたこと

- 読んだ: `git diff 789c8864..HEAD` の全体、`lockfile.go` の全文、`gate.go` の `Gate`(245-306)・`gateInterrupted`(308-320)・`recordGate`(707-763)、`gate_run.go` の全文、`internal/cli/org_gate.go` の RunE、`manifest.go` の `appendJSONLine`、`gate_test.go` の fixture(`run`、`gateStub`、`stopRebaseOnAConflict`)と新しいテスト(884-1095)、e0adb359 の `leader.md`・`reviewer.md`・`org_gate.go`・`watch.go`・`run-verify.sh`・`tests/test-run-verify-concurrency.sh` の hunk、`docs/tech-debt/README.md` のこの PR の追加分(`git diff --numstat 1bddc871...HEAD` が 11 行追加・1 行削除で、新しい行 10 と書き換えた行 1)
- `git diff --check 789c8864..HEAD` は空。追加行に TODO・FIXME・デバッグ出力・秘密情報はない
- `git grep` で確かめた: `withManifestLock` の非テストの呼び出し元(`approve.go:147`、`verbs.go:1361`、`spawn.go:621`、`:796`)と `withFileLock`(`inbox.go:327`)は、`context.Background()` に委ねる形に変わっただけで挙動は同じ。`recordGate(` の呼び出しは `Gate` と新しいテストだけ。`gateGitMust`・`gateDirtyListLines`・小文字の `gateStaticScript`・`gateTestScript` は 0 件
- 新しいテストが何を赤にするかは、変異を当てて読んで辿った(走らせていない)
  - `recordGate` の先頭の `ctx.Err()` を消す: 「ctx done before the lock」の `clockReads != 0` で赤(ロックの中で `o.now()` が読まれる)
  - ロックの中の `ctx.Err()` を消す: 「ctx done under the lock」で記録が残り赤
  - `withManifestLockContext` を `withManifestLock` に戻す: 「while waiting」は 5 秒で timeout し、`took > 3s` と文言の検査で赤。`TestGate_InterruptAfterTheScriptsRecordsNothing` も `took` で赤
  - `withFileLockContext` の `ctx.Err() != nil` の枝を消す: 文言に「timed out」が出て赤
  - `recordGate` 末尾の `switch` の 1 つ目の枝を消す: 全部緑のまま(C2-4)
- 確かめていないもの: テスト、`gofmt`、`-race`、skill の 4 面と `prompts_test.go` の e0adb359 の hunk(`check-skill-sync` と /test の受け持ち)

### 所見(cycle 2)

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6。cycle 1 の ID(M1〜L9)と取り違えないよう、ID に `C2-` を付ける。C2-5 と C2-6 は e0adb359 に由来し、b04289c1 の差分ではない。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | naming | 新しい 3 か所の comment が欠陥を「cross-review cycle 1 #1」とだけ指す。同じ字面が `internal/org` の別の PR の comment にあり、指す先の triage 報告は cycle ごとに書き換わる | `gate.go:714`、`gate_test.go:889`、`:940-941`。別の PR: `verbs.go:274` と `approval.go:477` は `of org-plan-approval` を添え、`watch.go:192` は triage のパスを添え、`internal/cli/org_watch.go:732` は添えない | `cross-review cycle 1 #1 of org-gate`(`verbs.go:274` の形)にする |
| C2-2 | LOW | exception-handling | `gateInterrupted` は、スクリプトが 0 で終わったあとの中断でも「the scripts were stopped ... the output so far is in <log>」と言う。b04289c1 でこの文を使う経路が 1 つ増え、新しいテストが文を固定している | `gate.go:311-320`(文は `:318-319`)、呼び出し `:293`(元からある)と `:302`(新しい)、固定 `gate_test.go:924` | 呼び出し側が「スクリプトが走っていたか」を渡し、終わったあとの中断は「the gate was interrupted after the scripts finished and nothing was recorded; their output is in <log>」のように分ける。`internal/cli/org_gate_test.go:700` の固定はスクリプト実行中の中断なので、そのまま残る |
| C2-3 | LOW | maintainability | `acquireFlock` の doc と文言は、ctx が自分の timeout だけだった時代のまま。呼び手の ctx が取り消されても「timed out after 5s」と言う | `lockfile.go:196-199`(「bounded strictly by ctx's own timeout」)、`:209`(`manifestLockTimeout` を決め打ち)。表に出ないのは `withFileLockContext` が `:94-96` で文を作り直すから | doc を「ctx が終わるまで」にし、`:209` から「timed out after」を外す(文は `withFileLockContext` が決める) |
| C2-4 | LOW | maintainability | `recordGate` 末尾の `switch` の 1 つ目の枝は、観測できる効果がない。唯一の呼び元 `Gate` が、ctx が終わっていれば `err` を捨てて `gateInterrupted` の文に置き換える | `gate.go:755-761` と `:302`。枝を消して「; nothing was recorded, so run the gate again」を付けても、`errors.Is(err, context.Canceled)` は `%w` で通り、新しい 2 つのテスト(subtest は 4 つ)はどれも全文を固定しない(「while waiting」は「the wait was stopped: context canceled」を含むかだけ見る) | `switch` をやめて `if err != nil` の 1 つに戻す。枝を残すなら、`recordGate` の戻り値の文言を固定するテストを足す |
| C2-5 | LOW | readability | watch の comment (c) の書き直しが途中までしか折り返されず、1 行だけ短い行が残った | `internal/org/watch.go:1219`(「pending alert whose baseline」だけで終わり、`:1220` は元の折り返し) | 1219-1221 を同じ幅に組み直す。動きには関係しない |
| C2-6 | LOW | maintainability | `stopGateGroup` が `manifestLockPollInterval` で poll する。その定数の doc は使い手を列挙していて、gate の止め方は入っていない | `gate_run.go:236`、`lockfile.go:32-36` | gate 用の名前(`gatePollInterval`)を置くか、定数の doc の列挙に足す。今のままだと、lock の poll を変えると gate がグループを止める間隔も変わる |

#### C2-1 の根拠

- 3 か所とも、コメントの本文が欠陥の中身を文で書いているので、読む人が困るのは「#1」の出典を探すときだけ。`docs/reports/cross-review-triage-org-gate.md` は cycle 2 の `/cross-review` も書き換えるので、cycle 2 のあとは「#1」が cycle 1 の指摘を指すと言えなくなる

#### C2-2 の根拠

- `gateInterrupted` は ctx が終わっていれば、`logPath` が空でなければ常に「the scripts were stopped」と言う。`gateCheckAfter` の中断(`gate.go:293`)は b04289c1 より前からあり、同じ不正確さを持っていた。b04289c1 はロック待ちの経路(`:302`)を足し、その文を `gate_test.go:924` が一字一句固定した。ユーザーが見るのは Ctrl-C のあとの 1 行で、スクリプトが通っていたのか途中で止めたのかは、この文からは分からない。「何も記録していない」と log のパスは正しく、打ち直すという次の手も変わらないので、LOW にした

#### C2-4 の根拠

- `recordGate` の `err != nil && ctx.Err() != nil` の枝のコメントは「Interrupted: Gate says so (gateInterrupted)」で、枝自身が、文を `Gate` に任せると言っている。ctx の取り消しは戻らないので、`recordGate` の `switch` と `gateInterrupted` が別の時刻に `ctx.Err()` を読んでも、ユーザーに見える文は 1 つに決まる
- ctx の状態で分類するので、本物の append 失敗(`appendJSONLine` は 1 回の `Write`。失敗したとき部分書きがありうる)と中断が重なると、中断として報告され、元のエラーが落ちる。`gateInterrupted` が cycle 1 から持つ作りと同じで、b04289c1 の新しい穴ではない

### cycle 1 の所見を HEAD で見直した結果

| ID | 状態 | 確かめた場所 |
| --- | --- | --- |
| M1 | 直った | `leader.md` の手順 10 の共通の規則と手順 12 の「断られたとき」の枝。skill の 4 面と `prompts_test.go` は読んでいない |
| M2 | 直った(SIGHUP)・残りは tracked | `internal/cli/org_gate.go` の `signal.NotifyContext` に SIGHUP、`leader.md` 手順 12 の「前の gate のスクリプトが終わるのを待つ」、tech-debt 228 行目(SIGKILL の行。Related の「(M2)」は、この報告の ID を書き換えていないので今も解決する) |
| M3 | 直った | `run-verify.sh` のコメントと、Case 6 のグループ TERM と pid だけの TERM の 2 分割。pid だけの検査は 1 秒の否定の待ちで、遅い machine では弱くなる向きに働き、赤にはならない |
| L1 | 直った | `leader.md` 手順 9 と `reviewer.md` の `master` |
| L2 | 直った | `gateCheckBranch`、`gateRebaseInProgress`、`gateNoBaseRefusal` と、その表のテスト(`stopRebaseOnAConflict` の `--apply` あり・なし) |
| L3 | 直った | `gateGitMust` は `git grep` で 0 件 |
| L4 | 直った | `org_gate.go` が `org.GateStaticScript`・`org.GateTestScript`・`org.GateDirtyListLines` を使う |
| L5 | 直った | `gate_run.go` の `stopGateStrays`、`TestGate_StopsWhatAScriptLeftRunning`(4 通り) |
| L6 | 直ったが C2-5 | `watch.go` の comment (c) |
| L7 | 直った | tech-debt の FR-7 の行に 1 文(PR の diff の `-|` と `+|` の組) |
| L8 | 半分 | `gate_run.go` への分離は済み。`gateRecordFromDetails`・`isGateReason` が 7c まで非テストの呼び元を持たない半分は、cycle 1 の推奨(「今すぐは要らない」)どおり手を入れておらず、comment が 7c を名指しする。7c が払うので、行は求めない |
| L9 | 直った | `leader.md` 手順 12 の「バックグラウンドで走らせ、ファイルに書く」 |
| Known gaps | tracked | tech-debt の「worktree のスクリプト」と「環境の接頭辞」の 2 行(PR の追加 10 行の中) |

cycle 1 の「Positive notes」が頼っていた点は、b04289c1 のあとも成り立つ。ロックは `recordGate` の中だけで取り、スクリプトの間は持たない。記録の前に manifest を読み直して binding と命を比べる。待ちは上限つきの poll。新しいテストの `time.Sleep(300ms)`(`gate_test.go:909`)と 200 ms の否定の待ち(`:995`)は、結果の判定ではなく中断を置く位置を決めるために使っている。前者が外れても(割り込みが `gateCheckAfter` の git に当たっても)文言は同じで緑のままなので、弱くなる向きに働く。ロック待ちの分岐そのものは `recordGate` の 4 つの subtest が分けて固定している。後者は goroutine が 200 ms 以内にロック待ちに入る前提で、外れると赤になる向きだが、そこまでの処理は `MkdirAll` と `OpenFile` だけで余裕は大きい。待ちはすべて 10 秒の上限があり、`holdManifestLock` の cleanup がロックを放す。

### b04289c1 で確かめて、所見にしなかったもの

- cross-review の指摘(中断が記録されうる)は閉じている。窓は 3 つあった: ロックを取る前、ロックを待つ間、ロックの中の読みから追記まで。順に、先頭の `ctx.Err()`、`withManifestLockContext`、追記直前の `ctx.Err()` が塞ぐ。残る窓は、追記の直前の確認と `Write` の間だけで、`Gate` の doc が「up to the moment the event is appended」と書いている
- `withManifestLockContext` の doc の「free なロックは ctx を見ず、fn は ctx が終わっていても走る」は `acquireFlock` の最初の試行が ctx を見る前に走ることと合う。呼び手が fn の中でも確かめる、という指示は `recordGate` が守っている
- org が変わっていた断り(`refused`)は、ctx が終わっていても先に返る。どちらも何も記録しないので問題ない
- 計画の 423a9f47 は日付つきの進捗の記録 1 行。「このあと回す」は書いた時点の話なので、所見にしない

### Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。cycle 2 は上限の最後で、C2-1〜C2-6 は直すと 3 回目の pipeline が要る。どれも動きと記録の値を変えないので、次の 1 行にまとめて足すことを勧める(orchestrator が足す)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Six small diff-quality leftovers in the org-gate PR, found in self-review cycle 2 (all LOW): (1) comments cite "cross-review cycle 1 #1" without the plan slug (`gate.go`, `gate_test.go`), a token other PRs in `internal/org` also use; (2) `gateInterrupted` says "the scripts were stopped ... the output so far" also when the scripts had finished and the interrupt hit the re-read or the lock wait; (3) `acquireFlock`'s doc and "timed out after" text assume its ctx is only the timeout; (4) the first arm of `recordGate`'s closing `switch` has no observable effect, since `Gate` replaces the message; (5) `watch.go`'s comment (c) has one short orphan line from a partial reflow; (6) `stopGateGroup` polls at `manifestLockPollInterval`, whose doc lists other users | Misleading or redundant sentences and one redundant branch; none changes behavior or a recorded value. (2) is the only user-visible one: a Ctrl-C during the lock wait reports stopped scripts that had already passed | Found at the pipeline cap (2/2); fixing needs a third run, and each item is a few lines | The next edit to `internal/org/gate.go`, `gate_run.go`, or `lockfile.go`, or a report that the interrupt message misled someone | `docs/reports/self-review-2026-10-13-org-gate.md` (Cycle 2, C2-1 to C2-6) |

### Recommendation

- Merge: 上の「Recommendation」の Merge の行のとおり(yes)。cycle 2 の差分に CRITICAL・HIGH・MEDIUM はなく、cross-review の指摘は閉じ、cycle 1 の直しは HEAD で成り立つ
- Follow-ups:
  1. 上の tech-debt の 1 行を足す(上限なので、C2-1〜C2-6 は直さない)。3 回目を回す理由が別にあるなら、C2-2 と C2-4 と C2-1 を先に直す
  2. cycle 2 の `/verify` と `/test` が書き直す報告に、b04289c1 で動いた `gate.go` の行を指す pointer が残っている(`verify-2026-10-13-org-gate.md` の AC5 の行は `gate.go:711-745`、`test-2026-10-13-org-gate.md` は `gate.go:723-724`。`recordGate` は今 713-763 で、`:723-724` は条件の枝ではなく `var refused` と lock の呼び出しに当たる)。関数名で書くか、HEAD で引き直す
  3. `/cross-review` を回すなら、triage 報告の「#1」が cycle 2 の新しい指摘を指す。C2-1 の字面を直すなら、その前がよい
