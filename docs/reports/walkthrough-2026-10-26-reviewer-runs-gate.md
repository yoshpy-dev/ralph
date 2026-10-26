# Walkthrough: reviewer-runs-gate(機能の org の PR の前の gate を、leader に代わって reviewer 座席が leader の TASK で打つ。機能の org の codex の reviewer にも台帳への `--add-dir`)

- Date: 2026-10-26(UTC)
- Plan: docs/plans/active/2026-10-26-reviewer-runs-gate.md(/pr で archive に移す)
- PR: ブランチ `feat/reviewer-runs-gate` の PR(番号はまだない。merge のあとは `gh pr list --state merged --head feat/reviewer-runs-gate` で引ける)
- Branch: feat/reviewer-runs-gate(この walkthrough の前の HEAD 9ffad9ce。20 commits、42 files、+2493/-594)
- Base: 633cd7c8(origin/main、「Merge pull request #258」)
- 読む範囲: 動きを変えるコードは `internal/org/permissions.go`(+33/-21)、`internal/org/spawn.go`(+9/-3)、`internal/org/approval.go`(+4)の 3 files。ほかの Go の 6 files はコメントと help と断りの文だけで、`internal/org/watch.go`(+30/-17)、`internal/cli/org_gate.go`(+13/-9)、`internal/org/regate.go`(+9/-8)、`internal/org/gate.go`(+9/-7)、`internal/cli/org_regate.go`(+8/-7)、`internal/org/regate_merge.go`(+7/-7)。雛形は `internal/org/prompts/leader.md`(+120/-60)と `internal/org/prompts/reviewer.md`(+86/-10)。テストの 6 files は `internal/org/prompts_test.go`(+551/-70)、`internal/org/spawn_test.go`(+87)、`internal/org/permissions_test.go`(+69/-12)、`internal/org/prompts_leader_inbox_test.go`(+21/-24)、`internal/org/regate_test.go`(+7/-7)、`internal/cli/org_gate_test.go`(+5/-1)。skill は `/org` の 4 写しが各 +159/-66、`/director` の 4 写しが各 +10/-5(合計 +676/-284)。文書は README(+2/-2)、quality-gates の 2 写し(各 +1/-1)、recipe の 2 写し(各 +12/-6)、`git-commit-strategy.md` の 2 写し(各 +3/-3)。記録は tech-debt(+23/-21)、仕様(+6/-2)、根拠の表(+2/-2)。残りの 684 行は計画(+181)、後工程の報告 5 本(+498。cross-review は triage)、insight event(+5)で、追加 2493 行の 27% にあたる。skill の写しも 676 行(27%)で、同じ文の 4 回分
- 後工程は 1 回(cycle 1/2)で終えた。self-review は /verify の前に 3 回まわり、cross-review の指摘は 0 件。記録だけの修正の例外は使っていない(triage の commit 9ffad9ce が HEAD で、そのあとに commit はない)

```
何を変えるか → 計画 → --add-dir の判定(S1) → コメントと help と断りの文(S2) → leader と reviewer の雛形(S3、逸れ 1、直し) → 雛形のテスト → skill と文書(S5、逸れ 2) → 記録(S6、sync-docs) → 後工程 → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- ユーザーの依頼「静的検査とテストもreviewerが担当するように変更してください」(2026-10-26)に応える。対象は機能ごとの org(`ralph org start --plan` で立てた org)の、PR の前の最後の検証
- これまで: leader の雛形の「機能ごとの org」の手順 12 で、leader が `ralph org gate` を打っていた。gate は `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を走らせ、結果を台帳に `gate_passed` か `gate_failed` として記録する。reviewer はレビューの TASK の初めに同じ 2 つを素で走らせるが、記録しない
- この PR から: leader は手順 9〜11(reviewer のレポートのコミット、fetch と rebase、org の report と計画の archive のコミット、secret scan)を済ませてから、TASK_ID `gate-<n>` の TASK で reviewer に gate を頼む。reviewer は本文のコマンドが決まった 2 つ(`mkdir -p .harness/state` と、出力を `.harness/state/gate-<org_id>.out` に書く `ralph org gate`)と同じときだけ打ち、そのファイルの `GATE:` の行で RESULT か BLOCKED を返す。leader はファイルを自分で読み、`HEAD:` を RESULT と `git rev-parse HEAD` に照らしてから座席を止め、push と PR に進む。leader は `ralph org gate` も 2 つのスクリプトも打たない。テストは今までどおり review と gate で 2 回走り、2 回とも reviewer が走らせる(D2)
- 決定の変更: 2026-10-07 にユーザーが選んだ「codex の `--add-dir` は leader だけ」(`docs/plans/archive/2026-10-07-org-state-dir-common.md` の Progress。当時の self-review M-1 への答え)を変えた。機能の org の codex の reviewer にも、permission mode が autonomous か edits で、台帳が cwd の下にないとき、`--add-dir <台帳>` が付く(`codexWritableRootArgs`、`internal/org/permissions.go:198`)。結びつきのない org の reviewer、implementer、未知の役割、director には付かない
- 残る危険: その reviewer は、leader と同じく台帳のディレクトリ全体に書ける(manifest の `plan_approved` と `gate_passed` の記録、受信箱、座席の prompt ファイル)。gate は repo のテストのコードを reviewer の子として同じ sandbox の中で走らせる。sandbox はこれを止めない。gate の記録だけに留めるのは、reviewer の雛形の文(`ralph org gate` のほかの `ralph org` の動詞を打たず、台帳のファイルを直接書かない)と、本文のコマンドを 2 つと照らす決まりだけ(self-review F1、`permissions.go:173-178` のコメント)。claude の座席には sandbox がなく、前から台帳に書けた
- 見張り(`ralph org watch --all`)の数え方は変えていない。gate の記録は今までどおり leader の動きとして数える(D6)
- 実機では動かしていない。このため仕様の FR-7 を `[x]` から `[ ]` に戻し、「印(2026-10-26)」に、この変更を含む binary での次の実行の記録で付け直すと書いた。次の実機の実行で印を付け直す必要がある。仕様の `[ ]` は 5 から 6(FR-3、FR-4、FR-7、FR-10、ツール非依存、受け入れ条件 5)、根拠の表の判定は `[x]` 26・`[ ]` 5 から 25・6 になった
- release のノートに書くこと(計画の Rollout): 「機能の org の gate は reviewer が leader の TASK で打つ。codex の reviewer にも機能の org では `--add-dir` が付く」。雛形(go:embed)と spawn の引数は binary に入るので、新しい binary で spawn した座席から効く。手元に入っている ralph は 5.1.0(`ralph version`)で、この変更を使うには main か release から作った binary が要る。計画の Assumptions によれば、古い binary で立てた leader は自分で gate を打ち続け、新しい reviewer は `gate-` の TASK が来たときだけ gate を打つので、食い違わない。コードの commit の件名は `feat:` 4 本(e67152da、5e8f5b50、ce85125a、8902cbf0)、`fix:` 2 本(286f3d28、d3c8e337)、`refactor:` 1 本(64f36f3f)なので、自動の changelog に出る(`.goreleaser.yml` が除くのは `docs:`・`test:`・`chore:`)
- 戻し方: この PR を revert する

## 2. 読む順(ファイルごと)

commit は 20 本。f9b01f00 が計画。コードは e67152da(S1)、5e8f5b50(S2)、ce85125a(S3)、8902cbf0(計画からの逸れ 1)と、self-review の直しの 286f3d28・64f36f3f・d3c8e337。skill と文書は 480b8781(S5)、12e4b9d5(逸れ 2)、67f3609d(直し)。記録は 097a8c66(S6)と 7aedb84c(sync-docs)。残りは後工程の report と計画の印。計画の S4(テスト)は S1〜S3 の commit にそれぞれ入っている。行番号は HEAD 9ffad9ce のもの。

1. `internal/org/permissions.go`・`spawn.go`・`approval.go`(e67152da、64f36f3f)
   - 変更の本体は 2 行。`codexWritableRootArgs`(`permissions.go:198`)が引数 `featureOrg` を受け、`writesLedger := role == LeaderIdentity || (featureOrg && role == reviewerRole)`(`:199`)で `--add-dir` を足す座席を決める。codex であること、mode が autonomous か edits であること(`CodexModeRunsUnderWorkspaceWrite`、`:86`)、台帳が cwd の下にないことの 3 条件は変えていない
   - Spawn(`spawn.go:1050-1051`)は `featureOrg := ActiveFeature(events, p.OrgID) != nil` を渡す。`events` は manifest のロックの下で読み直したもの(`:773-777`、stale の座席を扱う道では `:954-958`)で、`spawnFeatureSeatErr`(`:1515`)が見るものと同じ。新しいロックも読み直しも足していない
   - `reviewerRole`(`approval.go:70`)は、64f36f3f で `implementerRole`(`:66`)の隣に足した定数(self-review F6)。役割は seat_id ではなく `--role` の値で比べ、大文字小文字を区別する
   - コメントは `permissions.go:156-186` と `spawn.go:1042-1048`。64f36f3f が `permissions.go:173-178` に、reviewer が台帳全体に書けること、gate がテストのコードを同じ sandbox で走らせること、止めるのは雛形だけであることを書いた(self-review F1)
   - 見ること: `--add-dir` の位置は permission の flag のすぐあと、`--model` の前で、leader と同じ。dry-run の道と `spawnPrecheckErr` は触っていない(self-review の肯定的な点)
2. S1 のテスト(e67152da)
   - `TestCodexWritableRootArgs`(`permissions_test.go:182`)は結びつきのない org の表になり、`featureOrg=false` を渡す。reviewer の 2 行の名前を「reviewer of an org with no feature binding」に替え、期待値の nil はそのまま
   - 新しい `TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir`(`:261`)は `featureOrg=true` の 15 行。機能の org の reviewer(autonomous と edits、台帳が cwd の外)と leader に `--add-dir` が付き、台帳が cwd の下、guarded、claude、大文字の `Reviewer`、implementer、未知と空の役割、director には付かない
   - 新しい `TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir`(`spawn_test.go:3831`)は、claude の leader を結びつきあり・なしで spawn してから、codex の座席を linked worktree の cwd で spawn し、2 回目の AgentStart の argv 全体を比べる。事例は 3 つ(機能の org の reviewer、機能の org の implementer、結びつきのない org の reviewer)を autonomous と edits で打つ
   - 見ること: test の mutant c3・c4(`spawn.go` の `ActiveFeature` の判定を逆にする、`&& false` を足す)は表のテストでは落ちず、argv のテストだけが落とす。spawn が `featureOrg` を渡すところを見ているのは argv のテストだけ
3. S2: コメント、help、断りの文(5e8f5b50)
   - `watch.go` のコメント (c)(`:1345-1380`)を書き直した。`leaderActivityEventCount`(`:1386`)の `case EventGatePassed, EventGateFailed`(`:1401`)は変えていない。コメントは 3 つのことを書く。gate は leader の `sent` の TASK を起点にするので、記録を leader が頼んだ仕事が進んだ印として数える。数え方を変えると、保留中の ALERT の基準値(`watchPendingAlert.ManifestLen`、`:239`)が更新と戻しをまたいでずれる。`checkDeadman` は `activity && !subjectIsLeader`(`:1587`)のときだけ消すので、止まった leader を見逃さない
   - regate の断りの文(`regate.go:481-483`)が「its reviewer runs ralph org gate --org-id X in the feature worktree at its leader's TASK」になった。字面どおりに固定するテスト `TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns`(`regate_test.go:585`、期待値は `:594-596`)も直した
   - `ralph org gate` の help(`internal/cli/org_gate.go:29-35` と `:70-76`)は、reviewer 座席が leader の TASK で打つこと、人が打ってもよいこと、reviewer が出力をファイルに書いて `HEAD:` と `BASE:` を RESULT に写し、leader がファイルと HEAD に照らすことを言う。`TestOrgGateHelp`(`internal/cli/org_gate_test.go:866`)が 2 句を足して固定する
   - `gate.go` の冒頭のコメント(`:16-27`)に「Gate does not check who runs it」(`:22`)を足した
   - 計画の影響範囲の表にないが、同じ向きに直したものがある。`ralph org regate` の help(`internal/cli/org_regate.go:37-44`)と、`regate_merge.go` の 2 つのコメント(`:70-73`、`:112-114`)(verify の I1。計画の Progress に書いた)
   - 見ること: S2 は振る舞いを変えない。変わった Go の文字列は断りの 1 文と help だけで、gate の検査と記録の形は計画の Non-goals のとおり
4. leader の雛形(`internal/org/prompts/leader.md`、ce85125a、286f3d28、d3c8e337)
   - 手順 12(`:249-322`)が変更の中心。頭(`:249-254`)で implementer と reviewer を手順 13 まで止めないと言う(self-review F2 で reviewer に足し、F10 で implementer にも広げた)。送る前に `ralph org status` で reviewer が `(active)` かを見て、なければ手順 14 と同じ flag で spawn し直す(`:255-258`)。TASK_ID は `gate-<n>`(`:258-259`)で、本文には 2 つのコマンド(`:260-261`)を書かせ、「あなた自身は `ralph org gate` も 2 つのスクリプトも打たない」(`:265`)。待つあいだ worktree を変えず、implementer に TASK を送らない(`:266-268`)。reviewer の停滞の ALERT(`SEAT: reviewer`、`CONDITION: stall`)では stop も spawn し直しもせず、`ralph org read` で見る(`:268-272`、D7)。同じ理由で gate の TASK が 2 回続けて通らなければ BLOCKED を上げる(`:273-276`。F3 で足し、F9 で数える単位を gate の TASK にした)
   - 手順 12 の結果の 4 つの枝。`GATE: pass` の RESULT(`:278-289`)は、`.out` の `GATE: pass` と `HEAD:` を RESULT と `git rev-parse HEAD` に照らし、座席を止める前に `wip:` の `git log` を打つ(D4)。`GATE: unrunnable` の BLOCKED(`:290-312`)は、作業ツリーが汚れているという断りを手順 9 と同じく 3 つに分ける(F4、N3)。ほかに `GATE: fail` の BLOCKED(`:313-317`)と、reviewer の QUESTION(`:318-322`、F3)
   - 手順 9(`:203-227`)はレビューの TASK(TASK_ID が `gate-` で始まらない)で始まり、gate の TASK の RESULT は手順 12 で扱うと書く(`:225-227`、D3)。手順 5 には `--role` を `--id` の隣に足した(`:162-164`、F5)
   - 手順 14(`:324-356`)は、手順 12 で控えた `HEAD:` と HEAD を照らし(`:324-326`)、`wip:` をもう一度確かめる。違えば、手順 5 と同じ flag(`--role reviewer` を含む)で reviewer を spawn し直して手順 12 から頼み直し、spawn が断られたら BLOCKED を上げる(`:335-341`、R8)。RESULT の `HEAD:`・`BASE:` は gate の出力から写す(`:354-355`、例は `:358-366`)
   - ほかに直した所: ミッションの 5(`:32-35`、gate の TASK への BLOCKED は手順 12)、6(`:42-45`、止める決まりの例外)、7(`:46-49`、report は reviewer に gate を頼む前)、「機能ごとの org」の前文(`:78-80`、gate のコマンドにも `--state-dir`)、受信箱の節(`:512-522`、待ちの例から gate を外した)、運用規律(`:528-529`)
   - 見ること: leader が本文に書かせるコマンドと reviewer の雛形が照らすコマンドは、展開の前の字面が同じ(`leader.md:261`、`reviewer.md:51`)。手順 14 は reviewer だけを spawn し直すので、そのあとの gate が落ちると implementer がいない(F11、下の 5)
5. reviewer の雛形(`internal/org/prompts/reviewer.md`、ce85125a、8902cbf0、286f3d28)
   - ミッションの前文(`:8-15`)の書いてよいものに、gate の TASK で `ralph org gate` が台帳に書く記録とログと `.harness/state/gate-{{ORG_ID}}.out` を足し、ほかの `ralph org` の動詞を打たず、台帳のファイルを直接書かないと書く。ミッションの 1(`:17-21`)は `gate-` の TASK を下の節に回し、4 の括弧(`:40-42`)は、PR の記録は gate の TASK で打つ `ralph org gate` が残すと言う
   - 新しい「gate の TASK」の節(`:44-109`)の中身は次のとおり。本文のコマンドが 2 つと同じときだけそのまま打つ(`:48-55`。最初の文で条件を言うのは self-review N1 の直し)。違えば打たずに QUESTION で聞く(`:57-61`。計画からの逸れ 1、8902cbf0)。Claude Code は `run_in_background` で、Codex は前面で timeout を長くして打つ(`:63-67`)。worktree を書き換えず、コミットしない(`:68-70`)。ツールに切られたら、前のスクリプトが終わるのを待って 1 回だけ打ち直す(`:71-74`)。返し方は `.out` の `GATE:` の行で 3 つに分ける(`:75-85`)。最後に例の RESULT と BLOCKED(`:87-109`)
   - スコープ規律(`:159-161`)、座席内 fan-out の例(`:183-186`、gate の TASK のコマンドは子に任せない)、レポート契約(`:198-199`、gate の TASK ではレポートを書かない)も直した
   - 見ること: Codex の reviewer は gate を前面で打つので、そのあいだ leader の `ralph org send` は届かない(`:66-67`)。tech-debt の新しい行にした(下の 9)
6. 雛形のテスト(ce85125a、8902cbf0、286f3d28、d3c8e337。`prompts_test.go` +551/-70、`prompts_leader_inbox_test.go` +21/-24)
   - `leaderGateCommand`(`prompts_test.go:1042`)は、leader が書かせるコマンドと reviewer の節が打つコマンドを 1 つの定数で突き合わせる
   - `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`(`:1070`)は手順 12 と 14 の文を順に探す。表の検査は最初に見つからない文で止まる(`:1123` の直後の `break`)
   - `TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`(`:1246`)は、雛形を文に切り(`templateSentences`、`:1179`)、`ralph org gate` を名指す文が許した 4 つの句(`leaderGateRunSentences`、`:1168`)のどれかを含み、それぞれ 1 回だけ出ることを見る。判定は `leaderRunsGateItself`(`:1214`)。verify の写しの probe では、origin/main の `leader.md` がこの判定で 8 文落ちた
   - `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`(`:1542`)は、reviewer の節の文と、例の 2 つのメッセージを `protocol.Parse`・`Validate`・`HeaderValues` に通す(`:1628-1657`)
   - 既存の pin も書き換えた。`TestRenderRolePrompt_Leader_FeatureOrgGate`(`:814`)、`TestRenderRolePrompt_Leader_PushRefusesWipCommits`(`:1303`)、`prompts_leader_inbox_test.go` の `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`(`:76`)と `TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection`(`:152`)。leader の手順 12 の `run_in_background` の順を見ていた検査は消し、同じ順の検査を reviewer のテストに移した(計画の R5)
   - mutant は、追跡しているファイルを変えずに打った。verify は文の検査 2 つを `python3 -I` に写して 4 つ、test は `go test -overlay` で雛形の文を 6 つ(m1〜m6)と spawn の引数を 4 つ(c1〜c4)。どれも落ち、変えない写しの対照 2 つは通った
   - 見ること: pin は `containsPhrase` と `squashSpace` で空白を除いて比べるので、折り返しに影響されない。テストが見るのは雛形の文だけで、モデルがそのとおりに動くかは見ない(test の Test gaps 2)
7. skill の 8 写し(480b8781、67f3609d、d3c8e337)
   - `/org` skill(`.claude/skills/org/SKILL.md`)。動詞の表の `gate` の行(`:221`)は、打つのは機能の org の reviewer 座席で、leader が TASK で頼む(人が打ってもよい)と言う。`regate` の行(`:223`)も同じ向き。`start --plan` の説明(`:663-664`)は、gate の TASK のコマンドにも `--state-dir` を付けると言う。「org の終わり方」(`:943-1076`)では、項 4(`:977-1045`)が雛形の手順 12、項 5(`:1046`)が 13、項 6(`:1047-1070`)が 14 にあたる
   - `/org` skill のほかの直し: 「gate の記録」(`:1109-1115`)に、見張りが leader の動きとして数える理由を足した。役割の reviewer(`:1309-1340`)は gate の TASK を 5 つの小箇条で書く(self-review F7 で組み直した)。leader のサイクルの 7(`:1410-1416`)、Codex の leader の待ち(`:1447-1450`)、codex の `--add-dir` の段落(`:1489-1501`)、「permission 作法」節のタスク終了の項(`:1559-1566`)、完了条件(`:1585-1589`)も直した
   - `/director` skill(`.claude/skills/director/SKILL.md`)は 3 か所。leader が gate を頼んでいる間の reviewer の停滞の件の読み方(`:666-670`)、前面で待つ leader の例から gate を外したこと(`:713`)、regate を打たないブランチの説明(`:937-939`)
   - 写し 3 つずつは `scripts/sync-skills.sh` でそろえた。`/org` の 4 つと `/director` の 4 つは、それぞれ `md5 -q` が同じ値(この walkthrough を書く前に打った)。`check-skill-sync.sh` は「14 skill(s) in lock-step」(verify と sync-docs の記録)
   - 見ること: 項 4〜6 が雛形の手順 12〜14 と同じことを言うか(self-review が 3 回とも文ごとに照らした)。項 4 は雛形より短く、escalate の本文に書くことを一部略している(self-review 2 回目は食い違いではないとした)。役割の reviewer の項は `reviewer.md` の節と内容が重なる(F7 は指しにせず小箇条にした)
8. 文書(480b8781、12e4b9d5、67f3609d)
   - README の 2 段落(`README.md:252`、`:264`)。`:264` は、reviewer が `gate-<n>` の TASK で gate を打ち、leader が出力のファイルと `git rev-parse HEAD` に照らすと言う。末尾の「It needs no running watch and no approved plan」は `ralph org gate` という動詞の話なので残した(sync-docs の見送り 2。下の 5)
   - quality-gates の Quality pipeline gate の行(`docs/quality/quality-gates.md:82`、template の写しは `:81`)。2 つの差は前からの KNOWN_DIFF の「(see docs/tech-debt)」だけ
   - recipe の「The org ledger and `--add-dir`」節(`docs/recipes/codex-seat-permissions.md:81` からの節、変更は `:89-99` と `:109-111`)は、機能の org の codex の reviewer にも付くこと、台帳全体に書けて雛形だけが止めることを書く。recipe は scaffold 先に届くので、「leader-only choice of 2026-10-07」という履歴の句は 67f3609d で消した(self-review F8)。2 つの写しは `md5 -q` が同じ
   - `.claude/rules/ralph/git-commit-strategy.md:56-58`(template の写しも同じ): leader が `wip:` を確かめる時が、「gate を打ち直す前と push の前」から「reviewer の gate が通ったあとと push の前(手順 12 と 14)」になった。計画の影響範囲になかったファイルで、計画からの逸れ 2(12e4b9d5)
   - 見ること: AGENTS.md の repo map の gate の説明は打つ座席を書かないので、変えていない(sync-docs)
9. 記録(097a8c66、67f3609d、7aedb84c)
   - 仕様(`docs/specs/2026-10-07-org-multi-org-director.md`): Status に新しい段落(`:39`)を足した。FR-7 を `[ ]` にし(`:113`)、「2026-10-26 に決めたこと」(`:124`)と「印(2026-10-26)」(`:126`)を足した。Research findings の 1 段目の `--add-dir` の行(`:258`)には今回の変更を書き足した。7b と 8a-3b の注(`:121`、`:123`)は当時の決定として残す。仕様の `- [ ]` は base 5・HEAD 6、`- [x]` は 27・26(この walkthrough を書く前に数えた)
   - 根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`): FR-7 の行(`:79`)の判定を `[ ]` にして理由を書き、判定の数の文(`:105`)を `[x]` 25、`[ ]` 6 にした。表の `[x]` の行は base 26・HEAD 25、`[ ]` の行は 5・6。仕様の `[x]` が 1 つ多いのは、FR-12 が表に入らないため
   - tech-debt(`docs/tech-debt/README.md`): 097a8c66 は、gate を打つ座席を書いた 8 行(base の 224・225・229・230・231・233・330・359 行)を reviewer の形に直した。225 行は「codex の reviewer が gate のスクリプトを sandbox の中で走らせられるか」の行になった。67f3609d は、承認の tripwire の行(`:216`)に、codex の reviewer も台帳に直接書けると 1 文足した(F1)。sync-docs(7aedb84c)は 13 行に持ち越しの注を足した(verify の V1 の 10 行と、Trigger が当たっていた 3 行)。org-gate の小さな所見の行(`:234`)の (e) は取り消し線にした。新しい行は 2 つで、Codex の reviewer が gate を前面で打つあいだ `ralph org send` が届かない行(`:331`)と、F11 の行(`:375`、表の最後)。RESOLVED にした行はない
   - 見ること: tech-debt の変わった 15 行が 5 欄のままか(sync-docs が確かめた)。F11 の行は、試行の F11(session-end の WIP の行)と取り違えないよう、本文に「not the trial's F11」と書いてある
10. 後工程の report(a183673e、d1677868、c9caf03a、ad1a5c64、d3309d33、7aedb84c、9ffad9ce)と計画の印
    - self-review(3 回、Merge OK): 1 回目(HEAD 6182cb2e)は MEDIUM 2(F1、F2)、LOW 6(F3〜F8)、INFO 2(N1、N2)。286f3d28(雛形: F2〜F5、N1)、64f36f3f(Go: F6、F1 のコメント)、67f3609d(skill、recipe、tech-debt: F1、F7、F8)で直した。2 回目(HEAD 67f3609d)は LOW 2(F9、F10)と INFO 1(N3)で、d3c8e337 で直した。3 回目(HEAD d3c8e337)は直しが足した LOW の F11 だけ。3 回とも /verify の前で、cycle 1 の中
    - verify(ad1a5c64。対象は HEAD c9caf03a。pass): AC1〜AC7 を満たし、AC5 は写しの probe と mutant 4 つで一部、AC8 は /test に回した。`GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh` rc 0、`go vet` は go1.27.3 と go1.25.8 で rc 0、`check-sync.sh` は IDENTICAL 172・DRIFTED 0・ROOT_ONLY 0。LOW の V1(tech-debt の 10 行の持ち越しの注)と情報の I1(計画の印と影響範囲)は sync-docs が受けた
    - test(d3309d33。対象は HEAD ad1a5c64。pass): Go のテストは go1.27.3 と go1.25.8 の両方で、テストのある 8 パッケージがすべて ok(`internal/org` 1,197 本、`internal/cli` 826 本)。shell の 44 本は rc 0(656 秒)。mutant 10 個がすべて落ちた。外の道具の名前の canary は、名指しの組で 0 回呼ばれた。coverage は `internal/org` 95.0%、`codexWritableRootArgs` と `leaderActivityEventCount` が 100.0%、`Spawn` が 92.5%
    - sync-docs(7aedb84c): diff の外の文書にずれは見つからなかった。tech-debt の直しは上の 9。`GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh`、`check-sync.sh`、`check-skill-sync.sh` が rc 0。見送りは 4 つで、計画の影響範囲の表の抜け(下の 3 の逸れ)、README の文と見張りのない gate の TASK、tech-debt の `:257` の文、実機(残りの 3 つは下の 5)
    - cross-review(9ffad9ce。triage、cycle 1/2、指摘 0): 7aedb84c の上で `codex exec review --base main` を read-only の sandbox と `--ignore-rules` で打ち(model gpt-6-astra、reasoning effort xhigh)、rc 0。reviewer の判定は、permission の変更、spawn の組み込み、reviewer への gate の受け渡しに、直すべき退行なし。codex はテストも実機の座席も走らせていない
    - 計画の印: 6182cb2e が Implementation started と逸れ 2 件、ad1a5c64 が AC1〜AC7 と Review・Verification の artifact、d3309d33 が AC8 と Test の artifact、7aedb84c が Progress の sub-bullet 1 つ。`git diff f9b01f00 HEAD` で計画に動いたのは AC の checkbox と Progress だけで、本文は承認のあと変わっていない。承認の digest は、HEAD で `./scripts/plan-visual.sh digest` が返した値と `- Approved:` の行で同じ
    - insight event は `docs/insights/events/2026-10-26-reviewer-runs-gate.jsonl` の 5 行(self_review、verify、test、sync_docs、cross_review)。どれも cycle 1、verdict pass で、UTC の日付がどれも 2026-10-26 なのでファイルは 1 つ

## 3. 設計の判断(計画の D1〜D8)

- D1 gate を打つのは reviewer、頼むのは leader: push する head は、reviewer のレポート、org の report、計画の archive のコミットのあとにしか決まらない。そこで leader が手順 9〜11 を済ませてから、gate だけの TASK を送る。レビューと gate を 1 つの TASK にする案は、gate のあとにレポートのコミットが入って head がずれるので採らなかった
- D2 レビューの TASK の素の実行は残す: 壊れたコードをレビューしないため。テストは review と gate で 2 回走り、2 回とも reviewer が走らせるので、依頼を満たす。1 回にする案は、gate が落ちるまで壊れたコードのレビューが続くので採らなかった
- D3 gate の TASK は TASK_ID `gate-<n>` で見分ける: レビューの RESULT と gate の RESULT は、どちらも `GATE:`・`HEAD:`・`BASE:` を持つ。leader が gate の RESULT を手順 9 の合図と読むと、コミットを足して HEAD がずれる(consult の指摘)
- D4 leader は `.out` と HEAD を照らす: reviewer の RESULT は座席の申告なので、leader は ralph の出力が入ったファイルを自分で読む。merge の根拠は今までどおり台帳の `gate_passed` で、merge-order と merge が PR の head と照らすので、座席が申告を違えても merge には進まない
- D5 codex の reviewer にも `--add-dir`: 上の 1 のとおり。範囲は機能の org の reviewer だけ。claude の reviewer はもともと台帳に書けるので、codex の reviewer をそれにそろえる形になる
- D6 見張りは gate の記録を今までどおり数える: 最初の案は gate の記録を数から外すことで、Codex の計画助言の指摘で取りやめた。保留中の ALERT は数を基準値として状態ファイルに残し、見張りの再起動をまたいで比べるので、数え方を変えると更新と戻しで基準値がずれる。副作用として、reviewer の gate の記録が leader 以外の座席についての保留中の ALERT を消しうる。subject が leader の ALERT は動きの数によらず件になるので、止まった leader は見逃さない(R7)
- D7 座席の停滞の ALERT は雛形で扱う: Codex の reviewer が前面で gate を走らせるあいだ、pane の出力が変わらず、見張りが停滞の ALERT を出しうる。停滞の判定は変えず、leader の雛形と `/director` skill に扱いを書いた
- D8 FR-7 は `[ ]`: 印の規則(`docs/evidence/org-spec-evidence-2026-10-21.md:5`)は、外の道具に頼る項目の `[x]` に実機の記録を 1 つ求める。FR-7 の実機の記録は leader が打った gate のものなので、2026-10-26 の RC2 の直しが FR-4 と受け入れ条件 5 を `[ ]` に残した前例に合わせた(consult の指摘)
- 計画からの逸れ 2 つ(計画の Progress): 1 つ目は、reviewer が本文のコマンドを 2 つと照らし、違えば QUESTION で聞くこと(8902cbf0、consult の勧め)。2 つ目は `git-commit-strategy.md` の 2 写しの 1 文(12e4b9d5)。ほかに直しで、影響範囲の表にない `approval.go`(`reviewerRole`)、`regate_merge.go`、`internal/cli/org_regate.go`、`internal/cli/org_gate_test.go` にも触れた。承認済みの本文の表は変えず、Progress に書いた
- 計画の承認は、ユーザーの決まり「作成したプランは自動承認します」による(計画の Progress)。Codex の計画助言の MEDIUM 2 件(座席を止めたあとの HEAD の不一致で reviewer を spawn し直す道、D6 の取り下げ)は承認の前に入れた

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1(S1) | e67152da、64f36f3f。`permissions.go:199` の判定と `spawn.go:1050-1051` の受け渡し。`TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir`(15 行)、`TestCodexWritableRootArgs`(結びつきのない org の行)、`TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir`(argv の位置)。test の c1〜c4 が落ちた。本物の codex は打っていない |
| AC2(S2) | 5e8f5b50。`watch.go:1401` の case は変わらず、`TestLeaderActivityEventCount_GateEvents` と `TestWatch_Deadman_*`(19 本)が通った(test)。`leaderActivityEventCount` の関数の coverage は 100.0%。断りの文(`regate.go:481-483`、テストの期待値は `regate_test.go:594-596`)、help(`TestOrgGateHelp`)、`gate.go:16-27` のコメント |
| AC3(S3、leader) | ce85125a、286f3d28、d3c8e337。手順 12(`leader.md:249-322`)、9(`:203-227`)、14(`:324-356`)。`TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`(verify の probe では origin/main の雛形が 8 文で落ちる)、`TestRenderRolePrompt_Leader_PushRefusesWipCommits`。test の m1・m3・m5 と、verify の mutant 4 つが落ちた |
| AC4(S3、reviewer) | ce85125a、8902cbf0、286f3d28。「gate の TASK」の節(`reviewer.md:44-109`)と前文(`:8-15`)。`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` が例のメッセージを `protocol.Parse`・`Validate`・`HeaderValues` に通す。test の m2・m4・m6 が落ちた |
| AC5(S4) | 新しいテスト 5 本と、既存の pin の書き換え(上の 2 の 2 と 6)。消えた pin(leader の手順 12 の `run_in_background` の順)は reviewer のテストに移した。計画は「手で 1 回ずつ戻す」としたが、追跡しているファイルを変えないよう、verify は `python3 -I` の写し、test は `go test -overlay` で mutant を打った。合わせて 14 個(verify 4、test の雛形 6 とコード 4)がすべて落ち、対照の 2 つは通った |
| AC6(S5) | 480b8781、12e4b9d5、67f3609d、d3c8e337。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が verify と sync-docs で通った。`/org` skill の「org の終わり方」1〜7 は雛形の手順 9〜14 と 16 と同じ順(verify)。verify は、leader が gate を打つと言う文と、`--add-dir` を leader だけと言う文を掃引した。残ったのは仕様の当時の注と、今回の変更の説明だけ |
| AC7(S6) | 097a8c66。仕様の FR-7 が `[ ]` で、注と印がある。Status と根拠の表の数は `[x]` 25、`[ ]` 6。tech-debt の `:225` は codex の reviewer を指す |
| AC8 | `GOTOOLCHAIN=go1.25.8 ./scripts/run-verify.sh` の通しは 6182cb2e で rc 0(直しの 4 commit の前。計画の Progress と verify の記録)。そのあとの確かめ: verify の `run-static-verify.sh` rc 0(c9caf03a)と `go vet` の 2 版、test の Go のテスト(ad1a5c64、go1.27.3 と go1.25.8)と shell の 44 本、sync-docs の `run-static-verify.sh` rc 0。test の HEAD ad1a5c64 のあとに変わったのは insight event、計画、report 3 本、tech-debt だけで、Go とスクリプトは含まない(`git diff --name-only ad1a5c64 HEAD` で、この walkthrough を書く前に確かめた) |

## 5. 確かめていないこと

- 実機: herdr の上で reviewer が gate の TASK を受けて `ralph org gate` を打つこと、codex の reviewer が `--add-dir` で台帳に書けること、gate のスクリプト(Go のビルドのキャッシュやネットワークを使うもの)を codex の sandbox の中で走らせられること、leader が結果を雛形どおりに扱うこと。計画の Non-goals、R6、D8 のとおり。FR-7 の `[ ]` は次の実行の記録で付け直す。tech-debt の `:225` は開いたまま
- F11(self-review 3 回目、LOW): 手順 14 が reviewer だけを spawn し直して手順 12 に戻ったあと、その gate が `GATE: fail` か implementer の手が要る断りになると、implementer は手順 13 で止まっている。implementer を立て直す手は雛形にも skill にもない。HEAD の不一致はまれで、誤った push や記録は起きない。直さず tech-debt の行(`:375`)にした
- N2(self-review 1 回目、INFO): `watch.go` のコメント (c)(`:1345-1380`)は 36 行の 1 つの塊のまま残した。中身は合っている。次にこのコメントに手を入れるときに段落を分ける
- gate の TASK は TASK なので、長く走る見張り(`ralph org watch --all`)がないと `ralph org send` に断られる(`.claude/rules/ralph/agent-messaging.md`)。以前は leader が gate を直接打ったので、見張りがなくても打てた。`ralph org start --plan` が見張りを起こすので普通は起きない。断られたときの枝は手順 12 になく、leader は一般の裁定(ミッションの 5)か BLOCKED で受けることになる。実機で確かめていない(sync-docs の見送り 2)
- Codex の reviewer が gate を前面で打つあいだ(数十分)、leader の `ralph org send`(STOP、DECISION)は reviewer に届かない。tech-debt の新しい行(`:331`)にした。Claude Code の reviewer は `run_in_background` で打つので当たらない
- `./scripts/run-test.sh` は通しで打っていない。Bash ツールの 600 秒の上限を越えないよう、shell の部分は `verify.local.sh` の test モードを直接打ち、Go は版ごとにパッケージとテスト名で分けて打った。wrapper が golang の pack を選ぶ道は、この実行では通っていない(test の Test gaps 4)。canary の PATH は名指しの組(org 87 本、cli 24 本)にだけ置いた
- `./scripts/run-verify.sh` の通しは 6182cb2e のもの。そのあとの直し(286f3d28、64f36f3f、67f3609d、d3c8e337)を含む HEAD では、静的検査、Go のテスト、shell のテストを別々に打った(上の AC8)
- 手元は macOS(Darwin 25.5.0、arm64)だけで、Linux では走らせていない(CI が見る)
- 雛形のテストは文があるかと順を見るだけで、モデルがその文に従うかは見ない。雛形の文は coverage にも出ず、mutant で確かめた
- verify の V1 は sync-docs が持ち越しの注で受けた。払ったのは org-gate の (e) の 1 項目だけで、注を足したほかの行は開いたまま
- sync-docs の見送り 3: tech-debt の `:257` の「each copy of the skill holds the phrase once」は、`/director` skill の各写しが同じ句を 2 回持つことと合わない。origin/main でも同じで、この PR の diff の外にある
- archive のあと、この計画の `docs/plans/active/` の path を指す参照が切れる。HEAD(この walkthrough の前)で 29 行、8 ファイルにある。`scripts/archive-plan.sh` が書き換えるのは tech-debt の 16 行だけで、残る 13 行(仕様の 2 行、根拠の表の 1 行、report 5 本の 10 行)は `active/` のまま切れる。この walkthrough の `Plan:` の行も同じになる。前の PR にも同じ前例があり、self-review も取り上げないとした
- この walkthrough を書くために打ったのは、`git` の読み取り(`diff`、`log`、`show`、`grep`)、`grep`、`sed`、`wc`、`md5 -q`、`./scripts/plan-visual.sh digest`、`ralph version`、`date`。テスト、`go`、`herdr`、`gh`、network には触れていない。テストと静的検査の結果は各 report の記録に頼り、この walkthrough では打ち直していない
