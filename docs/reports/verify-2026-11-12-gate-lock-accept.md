# Verify report: gate-lock-accept

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-gate-lock-accept.md`(issue #309。`- Approved:` の digest `1d8736effdd9` は `./scripts/plan-visual.sh digest` で出し直した値と一致した)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff d2e5c6e9...HEAD`(HEAD `f9e94788`。origin/main `d2e5c6e9` を 2 回目に取り込んだ merge のあと。73 ファイル、6554 行追加、518 行削除)。静的解析は `./scripts/run-static-verify.sh` の既定の scope で、`templates/base/ralph.toml` が分類されないため全体に落ちた(golang)。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-gate-lock-accept.log`(git が無視する場所。run-static-verify.sh の全出力、HEAD から build した binary の `ralph org gate`・`gate accept`・`merge-order`・`regate` の `--help`、`ralph config get org.gate.max_load`、skill の写しの md5、origin の先頭)
- Insight event: `docs/insights/events/2026-11-12-gate-lock-accept.jsonl` に verify の 1 行(cycle 1、pass、low 4)を足し、この report と同じコミットに入れた
- ID の付け方: `V1` から `V4`(LOW、/sync-docs に渡す drift)、`I1` から `I4`(INFO)

## Spec compliance

受け入れ条件ごとに、コードと、それを固定するテストのソースを読んで判定した。テストを走らせて通ることは `/test` が確かめる。「満たす(静的)」は、コードがそう動き、その期待値を持つテストがあるという意味である。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 同じ台帳の 2 つの gate は直列。待つ間の stderr に前の gate の動詞・org・開始の時刻、後の gate の log の見出しに待った時間と前の gate(issue の受け入れ条件 1) | 満たす(静的) | `Gate`(`internal/org/gate.go:164`)は 188 行目の検査のあと 191 行目で `acquireGateLock` を取り、defer で最後まで持つ(記録の `recordGate` もその中)。`acquireGateLock`(`gate_lock.go:96`)は待ちの始め、保持者が変わったとき、`gateLockRemind`(5 分)ごとに行を出し、`gateLockHolder`(`:180`)が印の verb・org_id か branch・`started`・pid を言う。見出しの行は `gateLockWait.logLines`(`:74`、`# waited <d> for gate.lock: <holder>`)。テストは `TestGate_TwoGatesRunTheirScriptsOneAtATime`、`TestAcquireGateLock_WaitsForTheHolderAndSaysWhoItIs`、`TestOrgGate_WaitsForGateLockOnTheCommandsStderr` |
| AC2 gate.lock の待ちを ctx で切ると何も記録せず、log も作らない | 満たす(静的) | 待ちの `waitHeldFileLock`(`lockfile.go:231`)は ctx で file を閉じて戻り、`Gate` はその誤りで `createGateLog` の前に返る。max_load の待ちの中断も同じ位置で返る。`TestGate_InterruptedWhileWaitingForGateLockRecordsNothing`、`TestAcquireGateLock_ACancelledWaitHoldsAndWritesNothing` |
| AC3 待ったあと org と worktree を読み直す。HEAD が動けば新しい HEAD、disband されれば走らせずに断る | 満たす(静的) | `gate.go:203` が `gateCheck`(`:244`、`gateOrgLife` と `gateCheckWorktree`)をロックのあとにもう一度呼び、記録に使う `life` と `target` はこの 2 回目のもの。`TestGate_WaitsForGateLockAndGatesTheHeadItFindsAfterTheWait`、`TestGate_DisbandedWhileWaitingForGateLockRunsNothing` |
| AC4 local の regate は gate.lock を regate.lock の前に取る。ci の regate は取らない | 満たす(静的) | `Regate` の `regate.go:523`(gate.lock、`p.Checks != RegateChecksCI` のときだけ)が `:536`(regate.lock)と `:552`(最初の検査)より前にある。`TestRegate_WaitsForGateLockWithRegateLockFree`、`TestRegate_ACIRegateTakesNoGateLock` |
| AC5 log の見出しに始めの 1・5・15 分値と CPU 数、結びに終わりの値と最大。Details の `load_start`・`load_max`・`load_end`・`ncpu` が `log=` の前。読めなければ `unknown`。前の形の Details も読める(issue の受け入れ条件 2) | 満たす(静的) | `startGateRunLoad`(`gate_load.go:257`)と `finish`(`:272`)。`gateRecord.details` は via のあと、log の前に 4 つの鍵を書き、`gateRecordFromDetails` は知らない鍵と欠けた鍵を飛ばす(`gate_record.go`)。`TestGateRecordDetails_Load`、`TestGate_RecordsTheLoadAroundTheScripts`、`TestGate_RecordsAnUnknownLoad`、`TestOrgGate_PrintsTheLoadLine` |
| AC6 `max_load` は既定 0、負は設定エラー。超えていれば 30 秒ごとに読み直し、30 分で始めて stderr と log に書き、記録に `max_load=` | 満たす(静的) | `waitGateMaxLoad`(`gate_load.go:346`)。検証は `internal/config/config.go:1108`。`max_load=` は値が 0 より大きければ待ったかどうかによらず記録に入る。`TestWaitGateMaxLoad*`、`TestGate_MaxLoadStartsAnywayAfterTheCap`、`TestGate_MaxLoadWaitInterrupted`、`TestLoad_OrgGateMaxLoad` |
| AC7 `ralph org gate accept` の記録と断り。同時の 2 つは 1 件 | 満たす(静的)+ 観察 | `GateAccept`(`gate_accept.go:254`)は読み取り、`seatPaneTripwire`、`gateAcceptFailure`(`:301`)、追記を 1 回の `withManifestLock` の中で行う。断りは計画の一覧と同じ 10 種(記録が無い、成功、読めない、後ろの壊れた行、head_moved と tree_changed、log の不一致、受け入れ済み、空の旗、`--org-id`、座席の pane)。`TestGateAccept_RefusalsRecordNothing`、`TestGateAccept_ALogOfAFailureANewerOneReplacedIsRefused`、`TestGateAccept_TwoAtOnceRecordOne`。観察: HEAD の binary を空の state dir に向けると、`--org-id` 付き、空の `--reason`、記録の無い枝の 3 つがどれも exit 1 で断られ、書かれたのは `manifest.lock` だけだった |
| AC14 gate.lock は scripts に継がれる。親だけを SIGKILL しても script が終わるまで持たれ、正常の終わりでは親の LOCK_UN で外れる | 満たす(静的) | `runGateScript` が `cmd.ExtraFiles = []*os.File{lock}` を設定する(`gate_run.go:262`)。`gateLock.release`(`gate_lock.go:153`)は印を空にしてから LOCK_UN する。実際の process のテストは `TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled` と `TestGateLock_ReleaseFreesTheLockAChildStillHasOpen`。通るかは `/test` |
| AC8 merge-order の `accepted`(ready の次)、detail、JSON の `gate.log` と `gate.accepted`。後ろの gate 記録で無効。regate の merge commit の失敗は `regate` のまま(issue の受け入れ条件 3) | 満たす(静的)+ help | `collectMergeTargets` は gate の記録が付くたび(`mergeorder.go:552`、`:566`)と、後ろに壊れた行があるとき(`:584`)に `accepted` を消し、`gate_accepted` は `:568` で今の gate に `accepts` が合うときだけ付く。`classifyAccepted`(`:1192`)は `classifyGatePassed` の `ready` だけを `accepted` に替え、`regate` には「その head にしか効かない」を足す。`TestMergeOrder_AGateRecordAfterTheAcceptanceVoidsIt`、`TestMergeOrder_AcceptanceOfRegatesMergeCommitAppliesOnlyToThatHead`、`TestMergeOrder_AcceptedFailureIsListedRightAfterReady`。`merge-order --help` の class の表と並びにも `accepted` がある |
| AC9 merge は `accepted` を `ready` と同じ確かめで merge し、出力に受け入れの行、`pr_merged` の末尾に `gate=accepted` | 満たす(静的) | `checkMergeBranch` は class の判定だけを広げ(`merge.go:297`)、org が走っていないこと、PR の head と base、checks、保護、origin の読み直しは同じ。`recordMerge` が ` gate=accepted` を末尾に足す(`merge_run.go:572`)。CLI は `GATE: accepted (<reason>) by <by> at <ts>: <text>`。`TestCheckMerge_PassesForAnAcceptedBranch`、`TestMerge_MergesAnAcceptedBranchAndRecordsGateAccepted`、`TestOrgMerge_MergesAnAcceptedBranch` |
| AC10 `leaderActivityEventCount` は `gate_accepted` を数えない | 満たす(静的) | 関数(`watch.go:1566`)の switch に `EventGateAccepted` は無く、変わったのは注釈だけ。`gate_test.go` の表の行「gate_accepted in the watched org does not count」 |
| AC11 `Default()`、テンプレートの注釈、`defaults_sync_test.go` が一致 | 満たす | `Default()` の `MaxLoad: 0`、`templates/base/ralph.toml:393-404`、`orgTemplateDrift` と `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` に `org.gate.max_load` の行。`ralph config get org.gate.max_load` は `0` を出した |
| AC12 director skill と org skill の 4 写し、reviewer の役割 prompt。check-skill-sync と check-sync が通る | 満たす | director skill の写し 4 つは md5 `37fa055e…`、org skill の 4 つは `cdf84dda…` でそろう。director skill の「gate の失敗の受け入れ」節(人か、人がはっきり答えたときの director が打つ、打つ前に log・落ちた job・差分・負荷を確かめる)、「merge と auto_merge」の `accepted` の merge、gate.lock と max_load の待ち。org skill の動詞表の `gate accept` の行と「gate の記録」「merge してよい順」。reviewer の prompt は他の org の gate を待つこと、Codex の timeout にその時間を入れること、BLOCKED に `LOAD:` を写すことを言う。両方の check は静的解析の中で通った |
| AC13 `gate --help` と `gate accept --help`、AGENTS.md、仕様の FR-7 の注記、tech-debt の行 | 満たす | help は待ち、負荷、max_load、accept を書く(evidence の log)。AGENTS.md の internal/org の行に `gate_lock.go`・`gate_load.go`・`gate_accept.go`。仕様の FR-7 の下に 2026-11-12 の追記。tech-debt に 5 行(どれも 5 列で、計画の active のパスを指す) |

計画が名指した確認も行った。

- issue の受け入れ条件 1〜3 は AC1、AC5、AC8 がそれぞれ覆う(上の表)。
- ユーザーの決定 2 点。merge が `accepted` を受け付け、PR の checks・保護・head の確かめは変わらないこと(AC9、`merge.go:297` で広がるのは class の判定だけ)。regate で head が変われば受け入れが消えること(D5。regate が書く gate の記録はどれも、`collectMergeTargets` の 552・566 行目で `accepted` を消す。head が同じ打ち直しでも消える)。
- 2 回の origin/main の取り込み。2 回目の merge `f9e94788` の衝突は `docs/tech-debt/README.md` だけで、この枝の 5 行と #320 の 3 行の両方が残っている。#320 が変えた `sameFeature`(`internal/org/reserve.go:176`)を、この枝は `recordGate` の検査(`gate.go:752`)で呼ぶ。その `life` はロックのあとの 2 回目の `gateCheck` のもので、比べ方が機能ごとの digest に緩んでも、disband の検出(`lastDisband` の比較)は変わらない。`internal/cli/org.go` の `--org-id` の説明の一覧は #320 の分とこの枝の `gate accept` の両方を持つ。org skill の 4 写しは同じで、check-skill-sync も通る。origin の main は今も `d2e5c6e9` で、`git merge-tree --write-tree` は衝突なしで通る。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0、1 分 42 秒) | 始めの `uptime` の load average は 19.17。時間切れは無かった。分類されないパス(`templates/base/ralph.toml`)があるので全体の scope に落ちた |
| ・ shellcheck、hook の `sh -n` 20 本、guard の awk、settings.json 2 本の `jq -e`、Codex の hook の 3 つの guard | ok | |
| ・ `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6 |
| ・ `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照 | ok | |
| ・ golang: gofmt、`go vet ./...`、golangci-lint、staticcheck | ok | gofmt ok、golangci-lint 0 issues、go vet と staticcheck は出力なし |
| ・ branch の secret scan | clean | `d2e5c6e9..f9e94788` |
| `./scripts/plan-visual.sh digest <plan>` | `1d8736effdd9` | 計画の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org gate --help`、`gate accept --help`、`regate --help`、`merge --help`、`merge-order --help` | yes | 待ち、負荷、`max_load`、accept、`accepted` の class と並び、`GATE: accepted` の行、`--org-id` の一覧 |
| director skill と org skill(4 写しずつ) | yes | AC12 の行 |
| `internal/org/prompts/reviewer.md` | yes | AC12 の行 |
| AGENTS.md の internal/org の行 | yes | |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 の注記 | yes | 注記は計画を archive のパスで名指す(I3) |
| `templates/base/ralph.toml` の `[org.gate]` | yes | `max_load = 0` と注釈 |
| `templates/base/ralph.toml` の `[org.director]` の auto_merge の注釈 | no(V3) | |
| README.md | no(V2) | 「ready or accepted」は 3 か所直っている |
| `docs/quality/quality-gates.md`(root と templates/base) | no(V4) | |
| `docs/tech-debt/README.md` | 一部(V1) | 新しい 5 行は正しい。条件に当たった既存の行に記録が無い |
| `.claude/rules/ralph/agent-messaging.md` | 触る必要なし | accept は agmsg の message ではない(計画の Verify plan のとおり) |
| `internal/config/config.go` の `OrgDirectorConfig` の注釈 | 軽微(I1) | |

### V1 (LOW) 条件に当たった tech-debt の行に、持ち越しの記録が無い

この枝が `docs/tech-debt/README.md` に入れたのは新しい 5 行だけで、既存の行は 1 文字も変えていない。Trigger の列(4 列目)を、diff が触ったファイルと関数の名前で引くと、次の行の条件に当たる。どれも解消はしておらず、この repo の慣例では「`2026-11-12: feat/gate-lock-accept hit this trigger …, but did not pay it …`」の記録を足す。どの行の Impact も security の主張ではないので LOW とした。

- 238: `internal/org/gate.go`、`gate_run.go`、`lockfile.go` の次の変更
- 367: `gate.go` か `gate_run.go` の次の変更(`wip:` の検査)
- 398 (a): `Regate` か `runGateScripts` の次の変更(Regate の scripts の環境のテスト)
- 405: `internal/org/regate.go` の次の変更(ファイルの分割)
- 293: `regate_test.go` か `regate_merge_test.go` の次の変更
- 294: gate の記録の鍵の次の変更(この枝は `load_start` などを `log` の前に足した。行は `merged=` を同じ場所に足す案を言う)
- 249: `mergeorder.go` の次の変更、とくに `collectMergeTargets` と `mergeOrderListNotes`(この枝はどちらも変えた)
- 306: `mergeorder.go` の次の変更(runner の移動)
- 418: `mergeorder.go` の次の変更(分類の移動)
- 312: merge-order の分類の次の変更
- 313: merge-order の class の次の変更(`accepted` を足した)
- 159: org の品質の道の次の変更(失敗した gate を人の受け入れで merge できるようになった)
- 180 (a): `/org` skill の動詞表の次の編集(gate・regate・merge-order・merge の行と、新しい `gate accept` の行)
- 372: reviewer の役割 prompt の次の変更
- 269 (b): `director_skill_sync_test.go` に足す次のテスト(`TestDirectorSkill_PinsTheAcceptedGateFailureMerge`)
- 390: 4 つのテストファイルの次の変更(`org_merge_test.go`、`merge_run_test.go`、`director_skill_sync_test.go` の 3 つを変えた)
- 415: 5 つのファイルの次の変更(`internal/config/config.go`、1190 行)
- 453: `config.go` の `[org]` の検証の次の変更(`[org.gate].max_load` の負の値の検査を足した。行の払い方は `api_wait_minutes` の上限で、小さい)
- 88 (3)・(4): 対応するファイルに触れた次の機会(`watch.go` と `internal/cli/status.go`)
- 457(境界): `leaderActivityEventCount` の次の変更。この枝が変えたのは注釈だけだが、計画の Affected areas は watch.go のこの関数の説明を挙げている

このほか、self-review の LOW 4(local の regate は最初の検査より前に gate.lock と max_load を待つので、すぐ断られる regate も待つ)は、`ralph org regate --help` が「before it … reads the ledger」と正しく書いている。新しい行 468 の (c) は `up_to_date` と `push_only` しか挙げていないので、/sync-docs で (c) を広げてもよい。

### V2 (LOW) README の gate と merge-order の段落

- 288 行目の merge-order の段落: 分類の順(`merged` … `ready`)と並び(`ready`, `rebase`, …)に `accepted` が無い。「Merging one `ready` PR moves the default branch, so the other `ready` ones become `rebase`」も `accepted` を数えない。help(`merge-order --help`)と skill は直っている
- 286 行目の `ralph org gate` の段落: gate.lock の直列、待つ間の stderr、記録の負荷の鍵、`max_load`、`ralph org gate accept` のどれも無い
- 計画の S5 の申し送りが /sync-docs に渡した分と同じ

### V3 (LOW) `templates/base/ralph.toml` の auto_merge の注釈

290 行目の「each PR that `ralph org merge-order` lists as ready」は、`accepted` を含まない。同じ意味の `ralph status` の行(`internal/cli/status.go`)と `ralph org director status --help` は、この枝で「ready or accepted」に直した。計画の S5 の申し送りのとおり /sync-docs の分。

### V4 (LOW) `docs/quality/quality-gates.md` の Quality pipeline gate の行

87 行目は `ralph org gate` が何を確かめ、何を記録するかを書くが、台帳の gate が gate.lock で 1 本ずつ走ること、失敗の記録を人が `ralph org gate accept` で受け入れると merge-order と merge が `ready` と同じに扱うこと、が無い。後者は品質の関門の例外の道なので、この文書の読者に要る。root と `templates/base/` の写しは KNOWN_DIFF なので、両方を直す。

### INFO

- I1: `internal/config/config.go:346` の `OrgDirectorConfig` の注釈「AutoMerge, whether the director merges ready PRs without asking」。364 行目の欄の注釈は「ready or accepted」に直っている
- I2: 計画の受け入れ条件の印はすべて `[ ]` のままで、Progress checklist の「Review artifact created」も self-review の report があるのに `[ ]`。いつもの遅れで、/pr までに付ける
- I3: 仕様の注記は計画を `docs/plans/archive/2026-11-12-gate-lock-accept.md` で名指す。今は無いパスだが、/pr が計画を archive に移したあとに正しくなる(archive-plan.sh が書き換えるのは tech-debt だけなので、仕様は先に archive のパスを書く必要がある)
- I4: tech-debt の 338 行目(Codex の reviewer は gate の間 `ralph org send` が届かない)の Impact は、gate が gate.lock と max_load を待つ時間のぶん長くなる。条件には当たっていない。reviewer の prompt は timeout にその時間を入れるよう言う

## Observational checks

- HEAD から build した binary の help 5 つを読み、上の drift の表と照らした(evidence の log)
- `ralph org gate accept` の断り 3 つを、HEAD の binary と空の scratch の state dir で打った。どれも exit 1 で、`--org-id` の文、`--reason is required` の文、「no gate record in the ledger names the branch b」の文が出た。state dir に残ったのは `manifest.lock` だけで、manifest は作られなかった
- `ralph config get org.gate.max_load` は `0` を出した
- `seat.go` の `EventGateAccepted` の注釈の主張(RunningOrgs と orgsToDisband に数えない)を、`RunningOrgs`(`reserve.go:519`、Roster と workspace と予約だけを見る)と `orgsToDisband`(`verbs_all.go:337`、org-level では disbanded・workspace・scope_reserved だけを見る)で確かめた

## Coverage gaps

- テストの実行結果(AC14 の実際の process を SIGKILL するテスト、2 つの Gate を同時に走らせるテストを含む)は `/test` が確かめる
- herdr の実機で、reviewer 座席の gate が他の org の gate を待つこと、Codex の timeout の長さが足りること。この枝では誰も確かめていない
- darwin の `sysctl -n vm.loadavg` の読み取りは `TestReadGateLoad_ReadsThisMachine` が、linux の `/proc/loadavg` は解析のテストだけが見る。linux の実機は見ていない
- 待つ側の log の見出しの保持者は最後の poll で読んだ印で、前の gate が印を空にしてから LOCK_UN するまでの短い間に当たると「a gate whose marker is not written yet」になる(self-review の参考の 1 つ目)。AC1 の「前の gate が出る」が外れる窓はこの間だけで、テストでは再現していない

決定論の確かめで足りないものとして、tech-debt の Trigger の列を diff の関数とファイルで引く確かめ(V1 で手で行ったもの)がある。いちばん小さい形は、`git diff --name-only <base>...HEAD` のファイル名を各行の 4 列目と照らし、当たった行のうち `<branch>` の名前を含まないものを出すだけの script で、`run-static-verify.sh` の tech-debt の段に足せる。

## Verdict

- Verified: AC1〜AC14 はコードとテストのソースから満たす。AC7 と AC11 と AC13 は build した binary でも観察した。issue の受け入れ条件 1〜3 とユーザーの決定 2 点は、それぞれ AC1・AC5・AC8 と、AC8・AC9・D5 にある。2 回の origin/main の取り込みのあとも、この枝の動きと文書は #320 の変更と食い違っていない。静的解析はすべて通った
- Partially verified: AC14 と AC1 の、実際の process と同時実行での振る舞いは、テストのソースまで。通るかは `/test`
- Not verified: テストの実行結果。herdr の実機での gate.lock の待ちと Codex の timeout。linux での負荷の読み取り
- Drift(/sync-docs へ): V1 条件に当たった tech-debt の約 20 行に持ち越しの記録が無い。V2 README の gate と merge-order の段落。V3 テンプレートの auto_merge の注釈。V4 quality-gates.md の 2 写し。I1〜I4 は任意
- Verdict: pass
