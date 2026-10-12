# Verify report: org-plan-approval

- Date: 2026-10-13(cycle 1)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 951cf09bd7bc は `- Approved:` 行の digest と一致する)
- Verifier: verifier subagent (Claude)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)。self-review の 3 回目(48d7282f、CRITICAL 0・HIGH 0、Merge: yes)と、その LOW を tech-debt の行にした 78b3e873 のあとに走らせた
- Scope: 仕様適合、静的解析、文書の整合。対象はブランチ全体の `git diff 225dff27...HEAD`(HEAD 78b3e873、43 ファイル、+6832/-265)。テストは走らせていない(/test の担当)。HEAD のビルド(`go build -o bin/ralph ./cmd/ralph`、`bin/` は gitignore の対象)で、`ralph org approve` を使い捨ての台帳に対して動かした
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-plan-approval-static.txt`: `./scripts/run-static-verify.sh` の出力
  - `docs/evidence/verify-2026-10-13-org-plan-approval-sync.txt`: `check-sync.sh`、`check-skill-sync.sh`、skill の 4 面の sha256、写しの `cmp`、digest
  - `docs/evidence/verify-2026-10-13-org-plan-approval-approve-probe.txt`: approve の実機確認のスクリプトと出力
  - `docs/evidence/*.log` は gitignore の対象なので、コミットする evidence は `.txt` にした。wrapper が自分で書いた `docs/evidence/verify-2026-10-12-210514.log` はコミットしない
  - insight event: `docs/insights/events/2026-10-12-org-plan-approval.jsonl` に verify の 1 行(`--cycle auto` で cycle 1、verdict pass、low 3 は D1〜D3)を足し、この報告と同じコミットに入れた

## Spec compliance

行番号は HEAD 78b3e873 のもの。テストの関数は名前と位置だけを示し、通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `[org.approval].mode` の既定・検証・3 面 | 満たす | 定数と既定は `internal/config/config.go:166` と `:228`、`Load()` の検証は `:432`(空文字も拒否し、補わない)。3 面は `templates/base/ralph.toml:130-131`、`scripts/ralph-config.sh:103`(template の写しと `cmp` で一致し、export しない)で、`internal/config/defaults_sync_test.go:177` が固定する。テストは `TestDefault_OrgApproval`(`config_test.go:904`)と `TestLoad_OrgApproval`(`:917`。節なし、キーなし、per-plan、split、空、未知、大文字、前後の空白)。設定の値を読むのは `featureApprovalMode`(`internal/org/feature.go:345`)だけで、`withMainWorktreeOrgLimits`(`internal/cli/org.go:176`)が main worktree の値を写す。ほかに `Approval.Mode` を読む非テストのコードはない(grep) |
| AC2 approve の記録・resolve・出力 | 満たす | `Org.Approve`(`internal/org/approve.go:96`)。照合と追記は同じロックの下(`recordUnderLock`、`:146`、`recordApprovals`、`:306`)、resolve は `resolveApproved`(`:386`)、DECISION の文は `ApprovalDecisionText`(`:353`)、CLI の出力は `printApprovedPlans`(`internal/cli/org_approve.go:102`)、Details は `planApproval.details`(`internal/org/approval.go:311`)。テストは `TestOrgApprove_RecordsResolvesAndLeavesThePlan`(`approve_test.go:43`)、CLI の `TestOrgApprove_RecordsResolvesAndPrintsTheDecision`(`org_approve_test.go:155`)と `TestOrgApprove_TheDecisionCommandRunsInAPOSIXShell`(`:215`)。実機でも、stdout の 2 行と DECISION のコマンド、manifest の 1 行、件の resolve(note `approved <path> sha256:<digest>`)、計画のファイルの sha256 が変わらないことを確かめた |
| AC3 approve の拒否と all-or-nothing | 満たす | 受信箱だけで決まる検査は `approveCandidates`(`approve.go:237`)、台帳とファイルの検査は `checkApproveCandidate`(`:276`)、パスは `normalizePlanPath` と `resolvePlanFile`(`approval.go:144`、`:168`)、まとめての拒否は `approveRefusals`(`approve.go:169`)。`TestOrgApprove_Refusals`(`approve_test.go:161`)は 17 ケース(知らない ID、resolved、BLOCKED、PLAN なし・2 行・形違い、絶対パス、`..`、結びつきなし、plain な予約、disband 済み、不完全な結びつき、外を指す symlink、ないファイル、ディレクトリ、QUESTION のあとの変更)で、manifest と受信箱のバイト列が変わらないことまで見る。ほかに `TestOrgApprove_AllOrNothing`(`:259`)、`TestOrgApprove_IDList`(`:293`)、CLI の `TestOrgApprove_OneBadItemRecordsNothing`(`org_approve_test.go:293`)。実機で `e1 e2 e9` は 2 件の理由を並べて rc 1 になり、台帳の hash は変わらなかった |
| AC4 座席の pane からの approve | 満たす | `approveTripwire`(`approve.go:203`)は受信箱を読む前に走る(呼び出しは `:108`、受信箱の読み取りは `:111`)。`TestOrgApprove_Tripwire`(`approve_test.go:320`)は、受信箱を読めなくしても先に断ること、未設定・別の pane・止まった座席・dry-run の座席・disband した org の座席の pane は通ることを見る。CLI は `TestOrgApprove_RefusedFromAnActiveSeatsPane`(`org_approve_test.go:452`)。実機で leader の pane の値では rc 1 で台帳が変わらず、別の値では通って `herdr_pane=set` を記録した |
| AC5 per-plan の implementer への TASK | 満たす | 検査は見張りの検査のあと、dry-run の枝の前(`internal/org/verbs.go:326`)。`sendApprovalErr`(`:491`)、`checkTaskPlanApproved`(`approval.go:470`)、`checkPlanApproved`(`:529`)、`checkApprovedPlanFile`(`:558`)、`checkApprovedPlanHeader`(`:588`)。`TestOrgSend_ApprovalCheck_Refusals`(`approval_enforce_test.go:168`)は 9 つの理由を typed・raw・dry-run・raw の dry-run の 4 通りで送り、manifest・receipt・driver の呼び出し回数が変わらないことを見る。承認し直したあとの新しい digest は `TestOrgSend_ApprovalCheck_PassesAndApprovedAgain`(`:242`)、古い digest と打ち間違いは `TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest`(`:305`) |
| AC6 承認の要らない TASK | 満たす | `sendApprovalErr`(`verbs.go:491`)は、結びつきのない org、`RequiresApproval` が false の org、role が implementer でない座席を通す。`TestOrgSend_ApprovalCheck_NotApplied`(`approval_enforce_test.go:434`)が、reviewer への TASK、implementer への QUESTION と DECISION、split の org、トークンのない記録の org、結びつきのない org を、typed と dry-run で送る。仕様の受け入れ条件(今の 110 行)に当たる |
| AC7 implementer の spawn | 満たす | `spawnFeatureSeatErr`(`internal/org/spawn.go:1282`)を、dry-run(`:546`)、ロックの下の `checkCapacityAndStart`(`:1227`)、`spawnPrecheckErr`(`:1192`)の 3 か所で `spawnCapacityErr` のあとに呼び、拒否は `reject()` を通る。`TestOrgSpawn_ApprovalCheck_Implementer`(`approval_enforce_test.go:629`)は、dry-run の `rejected` と receipt が本番と同じ文であること、承認があれば spawned、reviewer は承認の前でも spawned、split は承認なしで spawned を見る。並びは `TestSpawnPrecheckErr_MatchesSpawn`(`feature_test.go:1246`)にも足された |
| AC8 leader のいない org の spawn | 満たす | `spawnFeatureSeatErr`(`spawn.go:1282`)は、seat id と role がどちらも leader の spawn だけを通し、ほかは active な leader がいなければ断る。`TestOrgSpawn_FeatureOrgNeedsItsLeader`(`approval_enforce_test.go:720`)は両方の方式で、seat id か role の片方だけが leader の座席も断ること、leader を立て直せば通ること、結びつきのない org は leader なしでも通ることを見る |
| AC9 前の命の承認 | 満たす | `currentPlanApprovals`(`approval.go:365`)は、最後の本物の `disbanded` より後の記録だけを読む。テストは `TestCheckPlanApproved` の `a record from the previous life (AC9)`(`approval_test.go:582`)、send(`approval_enforce_test.go:201`)、spawn(`:676`、本物の `Disband` のあと) |
| AC10 leader のタスクと雛形 | 満たす(逸脱 2 の読みで) | タスクの行は `featureLeaderTask`(`feature.go:390`)。雛形は `internal/org/prompts/leader.md:76-82`(方式ごとに飛ばす手順)と手順 1〜7(`:86`〜`:159`)。`TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`(`prompts_test.go:578`)が、語句、手順の順、例の QUESTION を escalate と `ParsePlanHeader` が受け付けること、`ApprovalDecisionText` を手順 4 が承認と読めること、拒否の文に目印と `ralph org approve` があり `--raw` は TASK の拒否にだけあることを固定する。AC10 の「BLOCKED を escalate する」は、逸脱 2 で理由ごとの次の手に替わった(下の「計画からのずれ」)。既存の固定の語句のうち 2 つ(手順の番号)は書き換えた(AC11) |
| AC11 既存のテスト | 一部(書き換えの記録は合う。通るかは /test) | 各スライスのコミットで、消えた行(コメントを除く)を持つ既存のテスト関数を数え直した。S1 f779cc56 は 11 本(feature_test.go 4、reserve_test.go 3、spawn_feature_test.go 2、verbs_test.go 1、prompts_test.go 1)で、Progress の「11 本と共有の helper」と合う。S3 e3d34062 で行を消したのは `TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused` だけで、feature_test.go の 2 本(`OrgIDAtTheLengthLimit`、`SlugAtTheLengthLimit`)は承認の記録を足しただけだった。S4 5596561a は既存のテストのファイルに触れていない。S5 36b9d3a3 は `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の語句 2 つ。9fb3fafa(L11)が同じ intro の語句をもう一度書き換えたが、base から見て変わった語句は 2 つのままである。`TestSpawnPrecheckErr_MatchesSpawn` には新しいケースが足され、既存のケースは変わっていない |
| AC12 文書と同期の検査 | 一部(D1 を /sync-docs で直す) | `check-sync.sh` は rc 0(DRIFTED 0)、`check-skill-sync.sh` は rc 0。skill の 4 面は同じ sha256(34e90d19…)、agent-messaging の 2 面は `cmp` で一致する。ずれは下の「Documentation drift」の D1〜D3 |
| AC13 start が記録する方式 | 満たす | 記録は `startFeatureLeaderParams`(`feature.go:361`)、トークンは branch の後ろ(`scopeReservedDetails`、`internal/org/reserve.go:281`)。読み戻しでは、2 つあるか空のトークンを `approvalModeUnreadable` にし(`:372`)、`RequiresApproval`(`:119`)はトークンなしだけを split と読む。`validateFeatureBinding`(`:157`)は 2 つの値のほかを断り、`sameFeature`(`:144`)は方式の違う同じ機能を別の結びつきとする。テストは `TestOrgStartPlan_ApprovalModeFromMainWorktree`(`internal/cli/org_feature_test.go:436`、本物の start で main の値を記録し、サブディレクトリの値に負けず、節がなければ per-plan、`--config` ならその値)、`TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart`(`org_approve_test.go:583`、feature worktree・サブディレクトリ・`--config` の 3 か所から send と spawn を dry-run)、`TestOrgSend_ApprovalCheck_ModeFromTheRecord`(`approval_enforce_test.go:573`)、`TestFeatureBinding_ApprovalModeRead`(`reserve_test.go:453`、11 ケース)、`TestStartFeature_RecordsApprovalMode`(`feature_test.go:315`、方式を変えた打ち直しの拒否) |
| AC14 digest が読まない場所の変更 | 満たす | `PlanContentHash`(`approval.go:280`)と、`checkApprovedPlanFile` のその照合(`:574-579`)。テストは `TestPlanContentHash`(`approval_test.go:368`)、`TestCheckPlanApproved` の AC14 の 3 ケース(`:596` Progress checklist、`:600` Branch、`:604` 見出しの後ろの Status)、send(`approval_enforce_test.go:196`、`:199`)、spawn(`:686`、checkbox の印だけなら spawned)。実機で、Progress checklist に足した 1 行は digest を変えず、approve が記録した `content=` は、その行を含む全文から先頭の 2 行を除いた内容の SHA-256 と一致した |

### 仕様 FR-7 との対応

`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7(`:65-71`)、注「7 段目の前半で決めたこと」(`:72`)、受け入れ条件、Security considerations と照らした。

- 1 項目(`[org.approval].mode` で選び、既定は per-plan): AC1 と AC13 で満たす。
- 2 項目(split では分割計画だけを人が承認する): split の org では spawn も TASK も承認を読まない(AC6、AC7)。分割計画の承認は 4 段目のままである。満たす。
- 3 項目(per-plan の機能の計画の承認、まとめての承認、implementer の spawn と実装の TASK の拒否、`PLAN:` の照合、再投入と改訂): approve(AC2、AC3)、受信箱の件の ID でのまとめての承認(A4)、拒否(AC5、AC7)、照合(AC5)は満たす。再投入は TASK ごとに照合され、改訂は digest か全文のハッシュで断られる(AC5、AC14)。要件は「TASK の本文の `PLAN: <path> <sha256>`」と書くが、実装はヘッダの塊の `PLAN: <path> sha256:<12 桁>` だけを読む(D3)。
- 4 項目(`/plan` と同じ図解ページと digest、director は代行しない): 人に見せる digest は `PlanDigest` で、`/plan` の digest と同じ値になる(`TestPlanDigest_MatchesScript`、`internal/org/split_test.go:777`)。図解ページは leader が雛形で書き、ralph は検査しない(計画の Non-goals、tech-debt 216 行)。director の扱いは 8a 段目に送ったと注に書いてある。7a の範囲では満たす。
- 5・6 項目と FR-8 の「merge してよい順」: 7b(次の PR)に送った。注の末尾にそう書いてあり、計画の Non-goals(`:50`)と合う。FR-8 の `auto_merge` は 8a 段目(仕様の段の表 `:215`)である。
- 注(`:72`): 書かれた決定をコードと照らし、すべて合った。方式の読み方と記録、トークンのない記録は split、読めない記録は承認を求める、検査は台帳とファイルの両方を見る、approve が書くもの、全文のハッシュ、検査が効く範囲、`PLAN:` が 2 行なら断る、今の命、TASK の拒否は記録しない、spawn の拒否は `rejected` と receipt、leader の検査、Security の読み、の順に確かめた。注が挙げるテスト 3 本も実在する。
- 受け入れ条件: `:109`(per-plan で未承認の TASK は拒否)は `TestOrgSend_ApprovalCheck_Refusals` と AC13 の CLI のテストで、`:110`(split は承認なしで送れる)は `TestOrgSend_ApprovalCheck_NotApplied` と同じ CLI のテストの split の場合で満たす。計画はこの 2 行を「107・108 行」と書くが、base でも 108・109 行で、注が 1 行増えた今は 109・110 行である(D4)。
- Security considerations(`:222`「director も leader もこの検証を迂回できない」): 承認については満たさない。計画の A2 と注がそう読み、tech-debt 213 行が記録している。`approveTripwire` は `HERDR_PANE_ID` の値と active な座席の pane を比べるだけなので、変数を外せば通る。leader は manifest に `plan_approved` を直接書け、`currentPlanApprovals` は Details だけを読んで書き手を区別しない。記録の `by=`・`herdr_pane=`・`cwd=` は `ralph org report` の event timeline に出る(`BuildOrgReport` はすべての event を表にする、`internal/org/report.go:164-180`)。「境界ではなく tripwire」という書き方は、skill、README、quality-gates、approve の help、コードのコメントで揃っている。Security の節そのものには注への参照がない(任意)。

### 計画からのずれ(Progress checklist の記録)の判断

- 逸脱 1(`approve.go` と `approve_test.go` に分けた): AGENTS.md の repo map が両方のファイルを挙げ、`seat.go` のコメントも `approve.go` を指す(8ad54167)。妥当。
- 逸脱 2(拒否の次の手を理由ごとに 1 つにした): `approvalNextStep`(`approval.go:663`)の 5 つの kind(TASK のヘッダは、ないときと digest が古いときで 2 通り)は、Progress、雛形の手順 7、skill、quality-gates の書き方と一致する。AC10・Scope 45 行・Risks 145 行の「BLOCKED」は、記録がない場合の後半と「それ以外」の場合に残る。妥当。
- 逸脱 3(`- Approved:` の行の形を厳しくした): `approvedLineDigest`(`:249`)。Scope 38 行の「`/implement` の 4 手目と同じ読み方」より厳しいが、コードのコメント(`:191-200`)と skill の照合の項に書いてある。Progress の「docs/plans の 22 本のうち 21 本がこの形」は数え直して合った(`- Approved:` の行を見出しの前に持つ計画が 22 本で、残る 1 本はテンプレートの `TBD`)。
- 逸脱 4(拒否の文から手順の番号を外した): `approvalNextStep` の文に手順の番号はない。妥当。
- S1 の `gofmt -w`: `internal/cli/org_feature_test.go` の base からの差分に消えた行はなく、中身は変わっていない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `scripts/ralph-config.sh` の変更で full fallback になり、golang の pack を選んだ。`gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(225dff27..78b3e873、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `shasum -a 256` で org の skill 4 面 | 4 面とも 34e90d19d554… | |
| `cmp` で agent-messaging と `ralph-config.sh` の 2 面 | どちらも一致 | quality-gates の 2 面は base と同じ KNOWN_DIFF だけが違い、新しい行(`:81`)は同じ |
| `./scripts/plan-visual.sh digest <plan>` | 951cf09bd7bc | `- Approved:` 行と一致 |
| `./scripts/secret-scan.sh --file` で evidence の 3 ファイル | rc 0 | コミットの前に確かめた |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `/org` skill の `approve` の行(`.claude/skills/org/SKILL.md:180`、4 面) | いいえ | 括弧の「approve が受信箱を読んだあとにほかの誰かが件を resolve していたときは、打ち方は示さず、承認は記録してあると言う」は 9fb3fafa(L3)の時点の文である。4b6d0d55(N3)のあと、エラーは「その人が直しを求めたかもしれないので、まだ leader に承認の DECISION を送らず、先に `ralph org inbox show <ID>` で note を読む」とも言い(`internal/org/approve.go:405-408`)、stdout には DECISION のコマンドがそのまま出る。括弧に「leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読むよう言う」を足す。4 面とも同じに直す |
| D2 `ralph org spawn --help` と `ralph org send --help`(`internal/cli/org.go`)(任意) | 一部 | どちらも見張りの拒否は書くが、per-plan の implementer の spawn と TASK の拒否、leader のいない org の spawn の拒否は書かない。計画の文書の一覧(Scope 46 行、AC12)に help は入っていないので、AC の不足ではない。help は `internal/` の文字列なので、/sync-docs で直さないなら tech-debt の行にするか 7b に回す |
| D3 仕様 FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md:72`)(任意) | ほぼ合う | 要件(`:68`)は「TASK の本文の `PLAN: <path> <sha256>`」と書く。実装はヘッダの塊の `PLAN: <path> sha256:<12 桁>` だけを読み、空行のあとの本文の行は数えない(`protocol.HeaderValues`、雛形の手順 5)。注は「TASK の `PLAN:` のヘッダ」と書くが、要件の「本文」をヘッダと読んだことは書いていない。たとえば「上の要件の「TASK の本文の `PLAN:`」は、ヘッダの塊の 1 行 `PLAN: <path> sha256:<digest>` と読んだ」を足す |
| D4 計画が挙げる仕様の受け入れ条件の行番号(情報) | 直さない | 計画の A5・AC6・Verify plan は「107・108 行」「108 行」と書くが、base(225dff27)でも 108・109 行で、今は 109・110 行。計画の本文は承認の digest の対象なので直さない。この報告は今の行番号を使った |
| D5 計画の Progress checklist と AC のチェックボックス(情報) | 遅れている | `- [ ] Review artifact created` は self-review の報告があるのに付いていない。AC1〜AC14 のチェックボックスも付いていない。`## Progress checklist` は digest の外なので、/sync-docs か /pr で付けてよい。AC は検証のあとに付ける段取りと読んだ(これまでの計画と同じ) |
| `/org` skill のほかの箇所(4 面) | 合う | verb 表の `spawn`・`send`・`start`(`:168`、`:169`、`:177`)、「機能の計画の承認」節(`:611`〜)、「start が拒否するもの」の方式の違う同じ機能(`:526`)、上限の節の方式の読み方(`:231`)、受信箱の節(`:307`)、leader のサイクル(`:852`〜)をコードと照らした。補償で予約を戻したときに前の承認が数えられないこと(tech-debt 215 行)も書いてある |
| `.claude/rules/ralph/agent-messaging.md:102-108` と template の写し | 合う | `cmp` で一致。`--raw` の TASK も同じヘッダの規則で読むこと、ほかの座席への TASK とほかの TYPE は承認が要らないことがコードと合う |
| README の org runtime(`:124` の verb 表、`:258` の段落) | 合う | `approve` が verb 表にあり、段落はまとめての all-or-nothing、herdr の pane の外で打つこと、境界ではないこと、start が記録する方式、古い版の org、leader のいない org の拒否を書く |
| AGENTS.md の repo map(`:90` の `internal/org/`) | 合う | `approve.go` と `approval.go`、`start --plan` が記録する方式を挙げる |
| `docs/quality/quality-gates.md`(2 面) | 合う | `:81` の行は 2 面で同じ。理由ごとの次の手の列挙はコードと合う |
| `templates/base/ralph.toml:120-131` | 合う | 既定 per-plan、ほかの値と `""` はエラー、start が org とともに記録すること、節がなければ per-plan であることが `Load()` と合う |
| `internal/org/prompts/leader.md` | 合う | `:76-82` と手順 1〜7。目印 `refused by the approval check`(`approval.go:645`)と、spawn の拒否が stdout の `rejected:` の行に出ること(CLI のテストが stdout の先頭を見る)が合う |
| `ralph org approve --help`、`ralph org --help`、`ralph org start --help` | 合う | ビルドで読んだ。approve の help は確かめることと all-or-nothing、記録の中身、tripwire、`--org-id` の拒否、split の org の記録を書く。`--org-id` の説明に approve が入った |
| tech-debt 213〜221 行 | 合う | 9 行とも 78b3e873 までに入った。挙げた関数とテスト(`approveTripwire`、`checkPlanApproved`、`sendApprovalErr`、`seatRoleFromEvents`、`spawnFeatureSeatErr`、`orgFeatureJSON`、`printStatusTable`、`reserveAgain`、`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`currentPlanApprovals`、`approvalModeUnreadable`、`checkTaskPlanApproved`、`currentPlanRef`、`resolveApproved`、`approvalNextStep`、`TestCheckTaskPlanApproved_StaleOrMistypedDigest`、`checkFixture`、`enforceFixture`)は実在し、書かれた挙動がコードと合う。213 行の tripwire の記述は `approveTripwire`(`approve.go:203`)の比較と合い、214 行のとおり `orgFeatureJSON`(`internal/cli/org.go:1400`)に方式の欄はない。217 行の (d) は 9fb3fafa で閉じて行から外した。同じブランチで足して外したので、取り消し線は要らない |

D1 は /sync-docs で直す。D2 と D3 は任意で、D4 と D5 は記録だけ。

## Observational checks

HEAD 78b3e873 のビルドで、タスクの worktree の gitignore された `.harness/state/verify-probe/` に台帳と機能の worktree を作り、`ralph org approve` を打った。approve は herdr も agmsg も呼ばない。受信箱の件は手で書いたので、escalate と通知は走っていない。スクリプトと出力は evidence の probe のファイルにある。

| 場合 | 結果 | 台帳 |
| --- | --- | --- |
| `--org-id a` を付ける | rc 1、`--org-id does not apply` | 変わらない |
| `HERDR_PANE_ID` が leader の pane | rc 1、tripwire の文(座席と org を名指し、pane の外の端末で打つよう言う) | 変わらない |
| `e1 e2 e9`(e2 は BLOCKED、e9 はない) | rc 1、「2 of the 3」と 2 件の理由 | 変わらない |
| Objective を 1 行変えたあとに `e1` | rc 1、digest が今は 7b9b4ffdf28d で QUESTION の 03e15b9b55e1 と違う | 変わらない |
| 計画を戻して、別の pane の値で `e1` | rc 0、approved の行、DECISION のコマンド、記録の注 | `plan_approved` 1 行(`by=verifier herdr_pane=set cwd=…`)、e1 は resolved |
| もう一度 `e1` | rc 1、already resolved | 変わらない |
| Progress checklist に 1 行足して、同じ QUESTION の `e3` | rc 0。digest は 03e15b9b55e1 のまま | 記録の `content=` は、足した行を含む全文から先頭の 2 行を除いた内容の SHA-256 と一致 |

- 計画のファイルの sha256 は approve の前後で同じだった(AC2)
- 最初の試行では、計画を変える場所を Progress checklist にしてしまい、approve は断らずに記録した。digest がその節を読まないためで、skill の「QUESTION が上がってから approve を打つまでの間に、digest が読まない場所が書き換えられていれば、approve はそれも含めて記録する」のとおりの動きである。変える場所を Objective にして取り直したのが上の表で、Progress checklist の場合は最後の行に分けて残した

## Coverage gaps

- テストは走らせていない(/test の担当)。表の関数が通ること、検査の条件を外すと落ちることは読んだだけである
- spawn と TASK の拒否は、ビルドしたバイナリでは確かめていない。見張りの検査(長く動く `ralph org watch --all`)が先に断るので、使い捨ての台帳で見張りを動かす必要があり、herdr と agmsg に触れずに済む形が組めなかった。CLI の層では `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart` が stub の PATH で本物の start から通す。計画の Verify plan が求める「一時の台帳での実機確認」のうち、approve の側(成功、計画の変更の拒否、座席の pane からの拒否)はこの報告の probe で取った
- leader が図解ページを作って撮れるか、headless の leader が承認の手順を最後まで進めるかは、実機で走らせていない(tech-debt 216 行、9 段目の試走)
- `ralph org approve` の、追記が途中で失敗した場合と resolve が失敗した場合は、テストを読んだだけである(`TestOrgApprove_AppendFailureRecordsNothing`、`TestOrgApprove_ResolveFailureKeepsTheRecord`)
- Linux と、別の herdr サーバーの pane の id が重なる場合は試していない
- release のノート(既定の per-plan で `start --plan` の org の implementer が承認を待つこと、`ralph.toml` は seed のファイル(`ownerForScaffoldPath`、`internal/cli/init.go:294`)で、おそらく `ralph upgrade` は既存のものに `[org.approval]` を足さないので、節のない利用者も既定の per-plan になること。upgrade の動きは確かめていない)と PR の本文の A2 の読みは、/pr と release の段で確かめる

## Verdict

pass

- Verified: AC1〜AC10、AC13、AC14 はコードとテストの対応を読んで満たすと判断した。AC2〜AC4 と AC14 の一部はビルドした approve でも確かめた。FR-7 の 1〜4 項目は 7a の範囲で満たし、5・6 項目と FR-8 の「merge してよい順」が 7b に送られたことは仕様の注に書いてある。受け入れ条件の 109・110 行はテストに対応がある。Security considerations の「迂回できない」は承認については満たさないと読み、仕様の注、計画の A2、tech-debt 213 行がそう書いていることを確かめた。逸脱 1〜4 は妥当と判断した。静的解析(changed scope、full fallback)、`check-sync.sh`、`check-skill-sync.sh` は rc 0
- Partially verified: AC11(書き換えた本数と理由は記録と合う。通るかは /test)、AC12(D1 を /sync-docs で直す。D2 と D3 は任意)
- Not verified: テストの実行、spawn と TASK の拒否のバイナリでの確認、leader の図解ページと承認の手順の実機、Linux、release のノートと PR の本文
