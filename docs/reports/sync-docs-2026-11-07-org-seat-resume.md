# sync-docs report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(承認の digest `0893ac30252e`)。spec は `docs/specs/2026-11-07-org-seat-resume.md`
- Branch: `docs/spec-org-seat-resume`。run 開始時の HEAD は `3148561b`(test の report のコミット)、base は main `11239c8d`
- Pipeline cycle: 1
- 先行 report: self-review `65eea701`(LOW 10 件 L1〜L10)、verify `39db8f25`(pass。LOW 6 件 V1〜V6、INFO 5 件 I1〜I5)、test `3148561b`(pass。変異 58 個のうち 7 個が生き残った)
- 制約: 計画の本文(`## Progress checklist` と Acceptance criteria の `[ ]` の外)は変えていない。`- Approved:` の digest は前後とも `0893ac30252e`
- Go を変えた(依頼に含まれる): help の文字列 2 か所とその固定のテスト、コメント 2 か所。振る舞いを変えるコードは無い。コミットは下の表の `a42b9845` と `a4f60ff9`。パイプラインを回し直すかは lead が決める

## Summary

verify が /sync-docs に回した項目を、V1 から V6 と I5 まで処理した。`/director` skill の `waiting` の項に `ralph org read` の打ち方を足し(V1)、README に `resume` と `[org].api_wait_minutes` を書き(V2)、`ralph org resume --help` と `ralph org send --help` の抜けを埋めてテストで固定した(V3)。tech-debt は、この PR が満たした Trigger の 16 行に注を足し、445 行と 448 行の状態を直し、新しい 3 行(`Send` のロック、生き残った変異、AC-10 の残り)を足した。仕様の FR・AC に印を付け、AC-10 だけは印を付けずに注を足した。

依頼の外で直したものが 3 つある。(1) `internal/org/resume.go` の `ResumeOutcomeFailed` と `Resume` の doc(self-review L2。verify の最後の行が /sync-docs に回している)。(2) `docs/quality/quality-gates.md` の「見張りなしで使える」一覧(`resume` が入っていない)。(3) 2 つの recipe が列挙している pane の環境変数(claude の再試行の 3 つが入っていない)。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | `## 止まった座席を続ける` の 3 の `waiting` の項に、座席の画面を読む `ralph org read --org-id <org> --seat <座席>` を添える指示を足した(V1。`skipped` の項と同じ)。`8ab92612` |
| `README.md` | コマンド表の動詞の列挙に `resume`、同じ行の説明に 1 文。Org runtime の節の `cleanup` の段落のあとに、`resume` の段落と `[org].api_wait_minutes` の段落(V2)。`8ab92612` |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | 「Cross-org watch gate」の行の「見張りなしで使える」動詞の一覧に `resume` を足した。`Resume` は `checkWatchGate` を呼ばない(`internal/org/resume.go` と `internal/cli/org_resume.go` に 0 件)。verify の drift 表には無かった。`8ab92612` |
| `internal/cli/org_resume.go`、`internal/cli/org_resume_test.go` | `resume --help` の終了コードの文に「`resumed` の記録に失敗したとき」を足し、help を固定するテストの文を同じ文に直した(V3 の 1)。`a42b9845` |
| `internal/cli/org.go`、`internal/cli/org_send_help_test.go` | `send --help` に、pane のロック(`ralph org resume` と同じもの)を取ること、待つのは `--timeout-ms` の残りまでであること、別の send か resume が持ったままなら何も打たず記録せず終了コード 1 になることを足し、`TestOrgSend_HelpSaysItTakesThePaneLock` で固定した(V3 の 2)。`a42b9845` |
| `internal/org/verbs.go` | `Send` の本文のコメント 1 か所(「the fail-closed budget check right after that wait」)。あいだに TASK の 2 回目の承認の確認と pane のロックの待ちが入ったので、「after that wait (which ... come before)」に直した。コメントだけ(V5 の 226 行 (a))。`a42b9845` |
| `internal/org/resume.go` | `ResumeOutcomeFailed` の doc と `Resume` の doc が「`resumed` の記録の失敗で failed」と書いていたのを、コードとテスト(`TestOrgResume_ResumedEventNotRecorded_ReportedAsAnError`)どおり「Outcome は変わらず、Reason と Errs に入る」に直した。コメントだけ(self-review L2)。`a4f60ff9` |
| `docs/recipes/isolated-herdr-server.md`、`docs/recipes/director-launch-cmd.md` と、`templates/base/docs/recipes/` の写し 2 つ | seat の pane に付く `--env` の一覧と、`launch_cmd` の起動スクリプトが export する変数の一覧に、claude の再試行の 3 つ(`[org].api_wait_minutes` が 0 より大きいときの claude の座席と claude の director)を足した。`cab45835` |
| `docs/tech-debt/README.md` | 下の「tech-debt」。`261ea84a`、`cab45835` |
| `docs/specs/2026-11-07-org-seat-resume.md` | FR-1〜FR-11、非機能要件 5 つ、AC-1〜AC-9 と AC-11〜AC-15 に `[x]`。AC-10 は `[ ]` のまま、直下に何を確かめ何を確かめていないかの注を 1 項目足した。Open questions の 1 つ目を、claude の上限の画面を S10 で見たことに合わせて直した。`177ea2cd` |
| `docs/plans/active/2026-11-07-org-seat-resume.md` | `## Acceptance criteria` の AC-1〜AC-9 と AC-11〜AC-15 に `[x]`。`## Progress checklist` の Review・Verification・Test の artifact に `[x]` と report の場所、Implementation started の下に post-impl の 2 項目を足した。`177ea2cd` |
| `docs/reports/sync-docs-2026-11-07-org-seat-resume.md` | この report |
| `docs/insights/events/2026-11-07-org-seat-resume.jsonl` | `sync_docs` の行(cycle 1、pass)を `insights-append.sh --slug org-seat-resume --cycle auto` で足した。`git status --short` は既存の jsonl を `M` と出しただけで、新しいファイルはできていない |

`.agents/skills/director/SKILL.md` は `scripts/sync-skills.sh` で作り直し、`templates/base/` の写し 2 つは `cp` で合わせた(`sync-skills.sh` は作らない)。skill の 4 つのファイルは `md5` が同じ(`f931fedce7955d454724b9e6b07eb7e5`)。recipe の写しは `cp` で合わせて `cmp` の確認を `check-sync.sh` に任せた。`/org` skill は今回の直しで変えていない。

## 依頼の項目

| 項 | 結果 |
|----|------|
| V1(`waiting` にも `ralph org read`) | 直した。4 つの写しが同一。`internal/org/skill_resume_test.go` の断片 `` `ralph org read --org-id <org> --seat <座席>` を添える `` は `skipped` の項で 1 回見つかれば通るので、`waiting` の項に同じ文があることはテストでは固定していない |
| V2(README) | 書いた。動詞の列挙、表の説明、段落 2 つ。段落は 1 つ目が長い(cleanup の段落ほどではない) |
| V3(help) | 2 つとも直した。`go test ./internal/cli/ -run 'Resume\|Send\|Help' -count=1` は ok(27.171 秒)。既存の 1 本(`TestOrgResume_HelpNamesTheSentenceAndDryRun`)は固定する文を新しい文に替え、新しい 1 本(`TestOrgSend_HelpSaysItTakesThePaneLock`)を足した。弱めた assert は無い |
| V4(448 行 = 上限の画面は実機で見ていない)、V5(Trigger を満たした行)、V6(L1、L10) | 下の「tech-debt」 |
| I5(仕様の印、Open questions) | 付けた。AC-10 は未チェックのまま注を足した |
| `verbs.go` の古いコメント | コメントだけだったので直した。`gofmt -l` は空、`go vet` は通った |
| テスト(test report の変異) | 1 行にまとめた。下の「tech-debt」 |
| AC-10 の残り | 1 行にした(仕様の Open questions には足さず、仕様の AC-10 の注と tech-debt の行で受けた) |
| `.claude/rules/ralph/` の pane の環境 | 変えていない。`git-commit-strategy.md` が pane の環境として挙げるのは `RALPH_WIP_CHECKPOINT=off` だけで、WIP コミットの話の中で出る。pane の環境の一覧ではないので、claude の再試行の変数は足さなかった。ほかの rules(`model-routing.md`、`agent-messaging.md`)にも一覧は無い。`agent-messaging.md` の続行の文の段落は verify が一致を確かめている |

## tech-debt

`docs/tech-debt/README.md` の変更は、`git show HEAD:` と行ごとに比べた(HEAD は `3148561b` の版)。変わった行は 18 行の編集(下の 2 つの表)と、末尾に足した 3 行。どれも 5 欄(raw では 7 片)で、欄ごとの backtick と `~~` は偶数、足した行の `file:line` の引用は 0。存在しない plan の参照は 0 件(`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` の「tech-debt README plan references」が OK)。Related に足した plan は `docs/plans/active/2026-11-07-org-seat-resume.md`(`/pr` の `archive-plan.sh` が `archive/` に書き換える)。行は先頭の言葉で示す。行番号は `3148561b` の版のもの。

### この PR が Trigger を満たした行(注を足した)

注の形は 2026-11-05 の org-role-models と同じ(何が当たったか、払わなかった理由、「Carried over to the next change that meets this trigger」)。

| 行(行番号) | 当たったもの | 払わなかった理由 |
|----|----|----|
| 6 batchable LOW findings from org-debt-batch(88。(2)) | `spawn_test.go` の doc 2 行(6406 → 6408) | その fixture の receipts のパスは変えていない。Scope は pane の環境 |
| What a leader is told about its reservation(188。(a)) | `leader.md`(1324 → 1329)。(b) は当たっていない | `RolePromptVars` と `{{SCOPE}}` は変えていない |
| Small findings of org-feature-worktree(200。(g)) | `leader.md` の agmsg の受信箱の節。(d) の `HerdrClient` の型の comment は変えていない(interface の中のメソッドの doc だけ) | (g) は f1b0352e で、(f) と (j) は b0fc4c59 で直済み |
| Small findings of org-plan-approval(226。(a)(b)) | `Send` | (a) は払った(`a42b9845`)。(a) の Debt・Impact・Trigger に `~~` を付けて閉じた。(b) の「The watch gate is」の短い行は残っている |
| Three test gaps of the org-director-attach PR(262) | `/director` skill の `### 件の処理` の ALERT の項と `### 答える順` | `director_skill_sync_test.go` は変えていない。(1)〜(3) はそのまま |
| Two premises of the org-director-start PR's D12 and D13(276) | `createWorkspaceAndPane` が `WorkspaceCreate` に再試行の変数を渡す | codex の director を実機で動かしていない。(a) は open |
| `director_start.go` is 1203 lines(319) | 1307 → 1311 行、`director_cmd.go` 733 → 746 | move-only の分割は Scope の外。行が挙げるテストファイルは変えていない |
| `verbs.go` is 2277 lines ほか(365) | `verbs.go` 2352 → 2442、`spawn.go` 3059 → 3093、`spawn_test.go` 6406 → 6408。`verbs_test.go` と `prompts_test.go` は変わらず | move-only の分割は Scope の外 |
| RC1(372) | `implementer.md` 181 → 186、`reviewer.md` 565 → 570 | agmsg の path は Scope の外。足した bullet は pane の行の扱い |
| RC6(379) | `/director` skill の `### 答える順` の send の文と、`ralph org read` を指す 2 か所 | 他の「届いた印」は書いていない。`ralph org read` は変えていない |
| Five files went past the 800-line guide(414) | `internal/config/config.go` 1121 → 1138 | move-only の分割は Scope の外。ほかの 4 つは変えていない |
| `internal/org/driver/herdr.go` grew(435) | `herdr.go` 968 → 971(`PaneEnv` の doc と `warnNoPaneEnv` の doc) | コメントだけ。分割は Scope の外 |
| The first seat of a new org has a window(436) | `placeSeatPane` の `seatPaneExtraEnv` | `pane_created` の append と `TabRename` の順は変えていない |
| `TabCreate` fails a reply that names the pane(441) | `placeSeatPane` の `tab_create` の枝 | `TabCreate` の戻りは変えていない |
| `placeSeatPane` is 89 lines(442) | `placeSeatPane` の本文は 89 行のまま。`seatPaneExtraEnv` と `claudeRetryPaneEnv` を隣に足した | move-only の分割は Scope の外 |
| The recipes under `templates/base/docs/recipes/` cite `docs/evidence/`(371) | 2 つの recipe の編集(`cab45835`) | 引用は足していない。数は `isolated-herdr-server.md` が 13 件・6 ファイル、`director-launch-cmd.md` が 1 件・1 ファイルのまま(`grep -o` で 2026-11-07 に数えた) |

数え直したもの。ファイルの行数は `wc -l` を base(main `11239c8d`)と、そのファイルを最後に変えたコミット `a42b9845` で取った(注には「the last commit of the branch that changed a counted file」と書いた。`a4f60ff9` は `resume.go` だけを変え、注が数えるファイルを変えていない)。

### 状態を直した行

| 行 | 変更 |
|----|------|
| The stopped-screen match of `ClassifyResumeScreen`(445) | Trigger の後半「上限の画面を初めて実機で見たとき」は S10 が満たした。S11(`8e9504f6`)がまとまりの区切りを直し、prompt のまとまりを照合から外した。「per entry に締める」は払っていない(普通の返事が一覧の文を引く場合が残る)。Impact に self-review L10(Enter だけの文も同じ部分一致で、3 か所を見て、空の入力欄に Enter を 1 回押して `kind=enter_only` の `resumed` を記録しうる)を足した。L10 は新しい行にせず、self-review が言う「同じ行の Impact」に足した |
| The usage-limit screens of claude and codex(448) | claude の `You've hit your weekly limit` は S10 が実機で見て、S11 が出どころを `seen:` にした。この分は済み。行は、claude の session・Opus・Sonnet の 3 つ(`docs:`)と codex の 2 つ(`binary strings:`)のために open のまま。Debt、Trigger、Related に足した |

### 新しい 3 行(末尾)

| 行 | 中身 |
|----|------|
| `Send` takes the pane lock after its idle/done wait(self-review L1) | `Send` のロックは idle/done の待ちのあとなので、その間に resume が打つと、`Send` は手番を始めた座席に打ち、`confirmSubmitted` が相手の手番を見て確かめた扱いにする。`Send` と `pane_lock.go` の doc にこの帰結は無く、`TestOrgResume_SendWaitsForAResumeHoldingThePaneLock` はその成功を期待値にしている。Trigger はロックまたは待ちの次の変更。直し方は、ロックの中で `AgentGet` をもう一度打つか、doc に書くか |
| Seven mutants of `ralph org resume` survive(test report の Test gaps 1〜6) | (a) R6 リーダーを座席 id で見分ける節、(b) R21 `LockWait` の既定 30 秒、(c) L1・L5 resume と `resume --dry-run` の台帳の分類(`TestOrgLegacyLedger_*` の表に 1 行ずつ足せば死ぬと overlay で確かめ済み)、(d) C9 画面の判定の順、(e) C10 `ResumeInputHolds` の先頭 12 文字、(f) V2 deadman が `resumed` を leader の活動に数えないこと、(g) `pane_lock_test.go` の 2 つのテストの `<-typing` に期限が無いこと(C3、C11 で 30 分止まる)。Trigger は項目ごとの足すテスト。(g) は `<-h.entered` の行と同じ判断を引く |
| AC-10 is met only in part | 週の利用上限(日本時間 2026-11-07 23 時にリセット)で、座席が作業を最後まで進めるところは見ていない。役割 prompt の 1 行も、実機の座席が `worker` だったので効きを見ていない。やり直しは live の記録の 1 の手順を、役割 `implementer` の座席で、リセットのあとに |

### 退けたもの

Trigger 欄を python で読み(取り消し線の外)、この diff が変えたファイル・関数・skill・recipe と突き合わせた。名指された関数の hunk を読んで決めた。

| 行 | 退けた理由 |
|----|------------|
| 137(`Send` の seat lookup、`DefaultSendEnterDelayMS`、`fakeHerdr` の遅延差し込み) | `Send` の seat lookup も `DefaultSendEnterDelayMS` も変えていない。verify が挙げた `fakeResumeHerdr` の `onSendText` は、`Send` のテストの `fakeHerdr` ではなく、遅延でもない |
| 337(`Send` の idle の待ち) | idle/done の待ち(`AgentWait`)は変えていない。ロックはその後ろ |
| 363・364(spawn の失敗の道) | 失敗の道(`fail(...)` の呼び出し)は変えていない。変えたのは pane を作る呼び出しの引数 |
| 183・184・189・191(`Stop`、`CloseDeferredSelfPane`、`spawnCapacityErr`、`Spawn` の doc) | 名指された関数は変えていない。変わった `spawn.go` の doc は `placeSeatPane` の doc(hunk の見出しは `dryRunSpawn` だが、変えた doc は次の関数のもの) |
| 148(`resolveOrgConfig`、`withMainWorktreeOrgLimits`) | 変えていない |
| 376・383・425・426(`leader.md` の feature-org の手順、節の追加) | 足したのは agmsg の受信箱の節の 1 文で、手順にも節にも当たらない |
| 387・388(implementer の self-check の項) | `implementer.md` の差分は star topology の節の bullet で、self-check の項ではない |
| 396(`director_cmd_test.go`、`director_wake_test.go`) | どちらも変えていない |
| 367・338・355・322 | Trigger が名指す `gate.go`、codex の reviewer、`InboxDeadlineDue`、`director_wait.go` は変えていない |

### 数えたあとで見つけた違い

- `verbs.go` の古いコメントは、依頼の言うとおり 403〜405 行にあった(`enterDelay` を作る直前)。226 行の (a) が引く文と同じなので、226 行の (a) を閉じた
- verify の V5 は 188(a) を挙げたが、(b) は当たっていない(`EnvelopeSummary` は変わらない)。注はそう書いた
- 200 の (g) は verify の一覧に無く、sweep で見つけた。371 は、recipe を直したので足した。どちらも前の PR が同じ形の注を足している

## Go の変更の確かめ

| 確かめ | 結果 |
|----|------|
| `gofmt -l internal/cli internal/org` | 空 |
| `go vet ./internal/cli/ ./internal/org/` | 通った |
| `go test ./internal/cli/ -run 'Resume\|Send\|Help' -count=1` | ok(27.171 秒)。`a42b9845` の内容で |
| `go test ./internal/org/ -run 'Skill' -count=1` | ok。skill の編集と `cp` のあとで打った |
| `go test ./internal/org/ -run 'SendDefaults\|Skill\|Recipe' -count=1` | ok。recipe の編集のあとで打った |
| `go test ./internal/org/ -run 'TestOrgResume_ResumedEventNotRecorded\|TestOrgSend_Budget' -count=1` | ok。`resume.go` のコメントを直したあとで打った |
| `./scripts/run-static-verify.sh` | exit 0、`All verifiers passed.`(gofmt ok、`0 issues.`、branch secret scan は `11239c8d..177ea2cd` が clean)。証跡は `docs/evidence/verify-2026-11-07-004103.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしていない。手元のコピーのみ) |

打っていないもの: `./scripts/run-test.sh`、`internal/org` と `internal/cli` の全体のテスト。変えた Go はコメント 2 か所、help の文字列 2 か所、help のテストだけで、判定のコードは変えていない。

## 確かめたこと

- `./scripts/check-sync.sh`: PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。skill と recipe の編集のあとで打った
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- skill の写し: `director` の 4 つの `SKILL.md` の `md5` が一致
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0(template の一致、template purity、tech-debt の plan の参照を含む)
- plan の digest: `./scripts/plan-visual.sh digest` は編集の前後とも `0893ac30252e`。plan の diff は `- [ ]` から `- [x]` への替えと `## Progress checklist` の中だけ
- 仕様の箱: `[ ]` は AC-10 の 1 つだけ(`grep` で数えた)
- `git diff --check` は空。迷子のファイルは無い。scratch は `/tmp/docmaint-seat-resume/` に置いた

## 残っていること(lead へ)

- **self-review の L3〜L9 は、直しも register の行も無い**。L2 は今回直した。L1 と L10 は今回 register に入れた。残りの L3(見直しで herdr が失敗したときの理由が `the screen changed` になる)、L4(`failed` の座席の一部に pane の読み方の案内が出ない)、L5(role prompt が続行の文で「始まる」行を指示とする)、L6(`heldFileLock` の doc の一覧に pane のロックが無い)、L7(読む行数 60 が CLI と org の 2 か所)、L8(起動スクリプトの `RetryEnv` の名前を検査しない)、L9(`PrintableInboxText` が書式の文字を通す)は、self-review が「この PR の中で直せる」と言っている。依頼は L1 と L10 だけを挙げたので、行は足していない。直すか、行にするかを PR の前に決める
- パイプライン: Go の `a42b9845` と `a4f60ff9` は help の文字列、テスト、コメントだけを変える。判定のコードは変えていない。`post-implementation-pipeline.md` は「cycle の上限で `.go` の編集は全体の回し直し」と書く。cycle 1 の途中の編集なので数えない扱いにしたが、回し直すかは lead が決める
- `/director` skill の `waiting` の項に `ralph org read` が入っていることは、テストで固定していない。断片のテストは `skipped` の項の同じ文で通る。固定するなら `resumeOutcomeBullet(ResumeOutcomeWaiting)` から `skipped` の項の手前までを切り出す検査を `skill_resume_test.go` に足す
- 計画の本文の Open questions の 1 つ目(「S10 でも当てられない」)は、S10 で claude の上限の画面を見たので古い。digest の範囲なので触っていない。仕様の Open questions は直した
- PR の本文の既知の穴に載せるもの: AC-10 の最後の段階(座席が作業を最後まで進めること)と役割 prompt の効き、実機で見ていない道(Enter だけ、codex、headless の director、複数の org、応答が返らない通信断)、live の確認は S11 の前の binary、続行の文が「始まる」行を指示とする L5
- release のノートに載せるもの(計画の Rollout): 動いている org の座席には、spawn し直すまで再試行の変数が入らない。`api_wait_minutes` は 0 で止められる。README にも書いた
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff main...HEAD` は 82 ファイル、+7,134/−147(HEAD `177ea2cd` まで。この report とその jsonl は含まない)。Go のテストでないコードは +1,639/−63、Go のテストは +3,267/−15。walkthrough はここでは書いていない

## Insight event

`./scripts/insights-append.sh --slug org-seat-resume --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を打った。cycle は `cycle-count.json` と同じ 1。既存の `docs/insights/events/2026-11-07-org-seat-resume.jsonl`(test の行がある)に足され、`2026-11-06-org-seat-resume.jsonl` の self_review と verify の行と合わせて 4 つの phase がそろった。
