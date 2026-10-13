# Verify report: org-gate

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-13-org-gate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 69f088275e54 は `- Approved:` 行の digest と一致する)
- Verifier: verifier subagent (Claude)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)。self-review(c7ba86ef、CRITICAL 0・HIGH 0・MEDIUM 3・LOW 9、Merge: yes)と、その所見を直した e0adb359・aeba07cb、報告への追記 c84d8e19 のあとに走らせた
- Scope: `git diff 1bddc871...HEAD`(HEAD c84d8e19、11 コミット、29 ファイル、+5262/-207)。仕様適合、静的解析、文書のずれ。テストは走らせていない(/test の担当)。行番号はすべて HEAD c84d8e19 のもの
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-gate-static.txt`: `./scripts/run-static-verify.sh` の changed と `RALPH_VERIFY_SCOPE=full` の 2 回の出力
  - `docs/evidence/verify-2026-10-13-org-gate-sync.txt`: `check-sync.sh`、`check-skill-sync.sh`、`shellcheck`(0.11.0)、`sh -n` と `dash -n`、写しの `cmp`、`git diff --check`、`go build ./...`、`go vet ./internal/org/ ./internal/cli/`
  - `docs/evidence/verify-2026-10-13-org-gate-help.txt`: `go run ./cmd/ralph org gate --help`、`--org-id` なしの実行、`ralph org --help` の gate の行
  - `docs/evidence/verify-2026-10-13-org-gate-probe.txt`: HEAD から作った binary を一時の台帳と一時の repo で打った確認(下の Observational checks)と、その script
  - wrapper が書いた `docs/evidence/verify-2026-10-13-072200.log` と `verify-2026-10-13-072224.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に verify の 1 行(`--cycle auto` で cycle 1、verdict pass、low 4 は D1〜D4)を足し、この報告と同じコミットに入れた
  - コミットの前に、この報告、4 つの evidence、insight の行を `./scripts/secret-scan.sh --file` で確かめた(どれも rc 0)

## Spec compliance

判定の言葉は次のとおり。「満たす」はコードと文書を読み、AC の各項目に対応する経路と、それを固定するテストの名前を確かめたもの。テストが通ることは /test が確かめる。「満たす(実機でも確認)」は、加えて HEAD の binary を一時の台帳で打って確かめたもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 pass のとき終了コード 0、stdout に `GATE: pass`・`HEAD:`・`BASE:`・ログのパス、`gate_passed` を 1 行、ログは `<state-dir>/gates/` の下で worktree に書かない | 満たす(実機でも確認) | `Gate`(`internal/org/gate.go:243-304`)、記録の Details(`gate.go:146-162`、キーの順は `head base base_ref branch by herdr_pane log`)、ログ(`internal/org/gate_run.go:91-115`)、出力(`internal/cli/org_gate.go:141-143` は `LOG:` を足した 4 行。計画の「3 行とログのパス」と同じ中身)。テスト: `TestGate_PassRecordsGatePassed`(`internal/org/gate_test.go:319`、`git status --ignored` まで空を確かめる)、`TestOrgGate_PassPrintsGateHeadBaseAndLog`(`internal/cli/org_gate_test.go:260`)、`TestOrgGate_StartPlanThenGateInTheFeatureWorktree`(`:541`)。probe の P1 と P9 |
| AC2 走らせる前の拒否(9 種)は、スクリプトを走らせず、何も書かず、終了コード 1 で理由と次の手 | 満たす(一部は実機でも確認) | `gateOrgLife`(`gate.go:331-361`)、`gateCheckWorktree`(`:382-439`)、`gateCheckBranch`(`:445-468`、rebase の途中を見分ける)、`gateNoBaseRefusal`(`:497-510`)、`gateCheckAncestor`(`:514-530`)、`gateCheckScript`(`:534-550`)。テスト: `TestGate_RefusedBeforeTheScripts`(`gate_test.go:392`、AC2 の 9 種すべてに subtest があり、manifest、`gates/`、スクリプトの呼び出しを調べる)、`TestGate_PassesFromASubdirectoryAndWithIgnoredFiles`(`:372`)、`TestOrgGate_RefusalPrintsNothingAndRecordsNothing`(`org_gate_test.go:421`)。probe の P2(未追跡)、P3(rebase していない head)、P6(detached HEAD)、P7(main のチェックアウト)、P8(知らない org)で、どれも manifest の行数が変わらなかった |
| AC3 `static_failed`・`test_failed`・`head_moved`・`tree_changed` を記録し、`GATE: fail`・理由・ログのパス・最後の 20 行、終了コード 1 | 満たす(static と test は実機でも確認) | `runGateScripts`(`gate_run.go:129-169`、static が落ちたら test を走らせない)、`gateCheckAfter`(`gate.go:567-582`)、`printGateResult`(`org_gate.go:134-171`、5 行のヘッダと空行と 2 字下げの詳細。計画 Notes の「実装で決めた細部」に記録がある)。テスト: `TestGate_ScriptFailuresAndChangesRecordGateFailed`(`gate_test.go:614`、4 つの理由)、`TestOrgGate_FailurePrintsTheReasonAndExits1`(`org_gate_test.go:316`)、`TestOrgGate_PrintGateResultEdges`(`:378`)。probe の P4(test_failed、最後の 20 行が `test line 11`〜`30`)、P5(static_failed、ログに `running ./scripts/run-test.sh` がない) |
| AC4 既定のブランチと base ref の決め方 | 満たす | `resolveGateBase`(`gate.go:592-622`、`origin/HEAD` は完全な ref 名で読む。計画 Notes に記録がある)。テスト: `TestGate_BaseResolution`(`gate_test.go:694`、origin/HEAD が trunk、trunk と同名のローカルブランチ、存在しない先、origin の main と master、master だけ、ローカルの master だけ、ローカルの main だけ、origin なしの 8 通り)。「どれもない」は AC2 の表の 2 つの subtest |
| AC5 走らせている間はロックを持たない。中断で子を止め、何も記録しない | 満たす | ロックは `recordGate`(`gate.go:711-745`)の中だけで取る。中断は `runGateScript`(`gate_run.go:180-213`、`Setpgid` とグループへの SIGTERM、猶予のあと SIGKILL)と、CLI の `signal.NotifyContext`(`org_gate.go:96`、SIGINT・SIGTERM に加えて SIGHUP。self-review の M2 の直し)。テスト: `TestGate_ManifestLockNotHeldWhileScriptsRun`(`gate_test.go:771`)、`TestGate_InterruptStopsTheScriptsAndRecordsNothing`(`:810`)、`TestGate_InterruptedBeforeTheScripts`(`:868`)、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(`org_gate_test.go:655`、3 つの信号と ctx の取り消し) |
| AC6 `leaderActivityEventCount` が数え、座席の状態を変えない | 満たす | `internal/org/watch.go:1243`(case (c) と、`:1208-1222` のコメント)。event の定数は `internal/org/seat.go:81-82` で、`stateEvents` には入れていない。`openOrgWorkspaces`(`internal/org/spawn.go:2095`)は PaneID のない event を飛ばすので影響しない。テスト: `TestLeaderActivityEventCount_GateEvents`(`gate_test.go:1260`)、`TestGateEvents_AreNotASeat`(`:1219`、Roster、`RunningOrgs`、`orgsToDisband`、report の Roster、`Status`) |
| AC7 見張りの関門と承認の検査をかけない | 満たす(実機でも確認) | `Gate` は `WatchGate` も承認の関数も呼ばない(`gate.go:243-304` を読んで確かめた)。テスト: `TestGate_NoWatchGateNoApprovalCheck`(`gate_test.go:1141`、断る見張りと承認のない per-plan の org で pass)。probe は見張りのない台帳と承認のない per-plan の結びつきで pass した |
| AC8 leader と reviewer の雛形の手順と語句、`prompts_test.go` の固定 | 満たす | leader の手順 9〜16(`internal/org/prompts/leader.md:163-255`)。fetch と rebase、衝突のときの TASK、report と reviewer のレポートのコミット、別の archive のコミット、secret scan、gate、stop、`HEAD:` の確認と push、RESULT への写し、gate のあとのコミットの禁止、時間がかかること、revert の規則(手順 10 の終わり、`:186-195`)がある。reviewer の RESULT の `HEAD:`・`BASE:`(`internal/org/prompts/reviewer.md:29-37`、`:56-57`、`:65-66`)。テスト: `TestRenderRolePrompt_Leader_FeatureOrgGate`(`internal/org/prompts_test.go:755`、語句と順序)、`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase`(`:935`) |
| AC9 既存のテストは手順の番号の語句のほかは書き換えない | 満たす(通ることは /test) | base からある test のファイルで消えた行は、`prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:474`)の assertion 3 本と、その doc コメントだけ(`git diff 1bddc871...HEAD -- internal/org/prompts_test.go internal/org/approval_enforce_test.go` の `-` 行)。計画 Notes の「3 本」と一致する。`approval_enforce_test.go` は追加だけ。`tests/` の既存の shell のテストは変わっていない |
| AC10 文書が新しい動きに合い、`check-sync.sh` と `check-skill-sync.sh` が通る | 満たす(LOW のずれは下の D1〜D4) | 2 つの script は rc 0(sync の evidence)。skill の 4 面は `cmp` で同一。quality-gates の 2 面の追加文は同じで、残る差は base からの KNOWN_DIFF。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:73`)、AGENTS.md(`:90`)、tech-debt(`docs/tech-debt/README.md:155` の追記と `:221-230` の 10 行)は動きに合う。AGENTS.md は `gate_run.go` を挙げていない(D1) |
| AC11 打った側の `RALPH_VERIFY_*` をスクリプトに渡さず `RALPH_VERIFY_SCOPE=full` を渡す。本物の `run-verify.sh` で言語の検証が走り、落ちる stub なら `gate_failed` | 満たす(環境の固定は実機でも確認) | `gateScriptEnv`(`gate_run.go:59-72`)。テスト: `TestGate_ScriptEnvironmentIsFixed`(`gate_test.go:1061`)、`TestGateScriptEnv`(`:1090`)、`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`(`internal/org/gate_runverify_test.go:72`、対照の実行で言語の pack が選ばれないことを先に確かめる)。probe の P1 で、`RALPH_VERIFY_BASE=HEAD RALPH_VERIFY_SCOPE=changed HARNESS_VERIFY_X=1` を与えた gate のスクリプトは `RALPH_VERIFY_SCOPE=full RALPH_VERIFY_BASE=<unset> HARNESS_VERIFY_X=<unset>` を見た |
| AC12 重なった `run-verify.sh` の終了コードが混ざらない。ファイルがなければ 0 にしない。`verify-scope` が残る。2 つの写しが同じ | 満たす | 実行ごとの `mktemp`(`scripts/run-verify.sh:59-61`)、EXIT trap(`:55`)、終了コードの読み方(`:325-335`、0〜255 の整数のほかは失敗)、`verify-scope` の公開(`:310-319`、rename で置く)。`cmp scripts/run-verify.sh templates/base/scripts/run-verify.sh` は同一。テスト: `tests/test-run-verify-concurrency.sh` の Case 1(`:325`)、Case 2(`:335`)、Case 3・4(`:374`)、Case 5(`:394`)、Case 6(`:412`)。`scripts/verify.local.sh:292` の `tests/test-*.sh` の glob が拾う |
| AC13 archive した計画の `PLAN:` の TASK は断られ、元のパスに戻すと通る | 満たす | テスト: `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored`(`internal/org/approval_enforce_test.go:1017`) |

### 仕様 FR-7 との対応

- 「後から入る PR は、main に rebase してゲートを取り直してから出す」: AC1〜AC5 と AC8(leader の手順 9〜14)で満たす。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:73`)は「push する head で最後のゲートを取る」と読んだと書く
- 「reviewer の結果には、検証した head と base の SHA を残す」: AC8 の reviewer の雛形(`reviewer.md:29-37`)で満たす。ralph はこの 2 行を読まない(B2)。tech-debt の `:227` がその限界を記録する
- 7c に送ったもの: `ralph doctor` の保護ブランチの warn、`ralph status` の保証の表示、FR-8 の merge してよい順。仕様の注の最後の文がそう書き、FR-7(`:65`)と FR-8(`:74`)の checkbox は未完のまま。差分の中のコード、help、skill、雛形は、この 3 つを実装したとは書いていない。`gate.go:27` と `:164` のコメント、skill の「gate の記録」(`.claude/skills/org/SKILL.md:851`)は「7c が読む」「今後入る」と先のこととして書く

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | 差分に `scripts/run-verify.sh` があるので full fallback(`shared:scripts/run-verify.sh`)で golang の pack を選んだ。`gofmt: ok`、golangci-lint は `0 issues.`。shellcheck(hook、`scripts/*.sh`、`tests/test-*.sh`)、hook の `sh -n`、guard の awk、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照、branch の secret scan(1bddc871..c84d8e19、clean)も通った |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | rc 0 | 上と同じ結果。`run-verify.sh` 自身が変わったので full でも 1 回走らせた |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `shellcheck scripts/run-verify.sh tests/test-run-verify-concurrency.sh templates/base/scripts/run-verify.sh` | rc 0 | ShellCheck 0.11.0。`-s sh` でも rc 0 |
| `shellcheck` に `# shellcheck disable=SC2329` の行を除いた `run-verify.sh` の写し | rc 1(SC2329 info) | コメントの「shellcheck 0.11 が EXIT trap の呼び出しを見落とす」は正しく、disable は要る(self-review が未確認として残した点) |
| `sh -n`、`dash -n`(2 つの script) | rc 0 | |
| `cmp`(`run-verify.sh` の 2 面、org の SKILL.md の 4 面) | 同一 | |
| `git diff --check 1bddc871...HEAD` | 出力なし | |
| `go build ./...`、`go vet ./internal/org/ ./internal/cli/` | 出力なし(rc 0) | |
| `./scripts/plan-visual.sh digest <plan>` | 69f088275e54 | `- Approved:` 行(4 行目)と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | はい | `cmp` で同一。動詞の表の `gate` の行(`.claude/skills/org/SKILL.md:181`)、古い台帳の検査の動詞(`:57`)、`--state-dir` を付けるコマンドの一覧(`:500-501`)、「org の終わり方」の手順 1〜7(`:758` から)、「gate の記録」(`:833` から)、締めの順とチェックリストが gate を含む。行の中身は `gate.go`・`gate_run.go`・`org_gate.go` の動きと合う(拒否の条件、記録のキー、4 行と 5 行の出力、3 つの信号、残ったプロセスを止めること、見張りと承認が要らないこと) |
| leader の雛形と skill の手順 | はい | 雛形の手順 9〜16 と skill の手順 1〜7 は、同じ順(rebase、report、archive、secret scan、gate、stop、push と PR、disband)で、同じ revert の規則を持つ。雛形の中の「手順 N」「この節の N」の参照は新しい番号に合う(`leader.md:41`、`:63`、`:173`、`:184-186`、`:211-229`)。skill の「手順 9〜16 と同じ」(`:761`)も合う |
| `ralph org gate --help` | はい | help の evidence。検査の順、fetch しないこと、環境の固定、ログの置き場所、記録のキー、立て直しのときの拒否、4 行と 5 行の出力、SIGHUP を含む中断、残ったプロセス、見張りと承認が要らないことは、コードと合う |
| `docs/quality/quality-gates.md:82` と `templates/base/docs/quality/quality-gates.md:81` | はい(D3 は情報) | 追加の文は 2 面で同じ。2 面の差は base からの KNOWN_DIFF のまま |
| 仕様の注(FR-7 の 7b) | はい | 上の「仕様 FR-7 との対応」 |
| tech-debt | はい | `:155` に feature org の gate の 1 文を足し、`:221-230` に 10 行を足した。参照する計画のパスは `docs/plans/active/` で、`scripts/archive-plan.sh` が archive のときに書き換える(`tech-debt README plan references` の検査も通った)。`:155` の Trigger「the next change to the org quality pipeline」はこの PR で起きたが、行は閉じずに範囲を狭めた形で残した。reviewer の座席の検査が雛形頼みのままなので、それで合う |
| D1 AGENTS.md の repo map(`AGENTS.md:90`) | いいえ(LOW) | gate の項目は「`gate.go`: `ralph org gate` runs `run-static-verify.sh` and `run-test.sh` ...」と書くが、スクリプトを走らせるのは e0adb359 で分けた `gate_run.go` で、AGENTS.md に `gate_run.go` は 0 件。パッケージの doc(`seat.go`)は aeba07cb で直したが、AGENTS.md は直していない。`gate.go`/`gate_run.go` と並べれば足りる |
| D2 `README.md:252` の機能ごとの org の段落 | 遅れている(LOW) | 「The leader opens the PR with `gh pr create` and disbands the org」のままで、PR の前の rebase と `ralph org gate` を書いていない。README は escalate、watch、approve の段(`:254`、`:256`、`:258`)をそれぞれの段で足してきたが、gate の段はない(`ralph org gate` は README に 0 件)。計画の文書の範囲(Scope の最後の項目)は README を挙げていないので、計画の外のずれとして /sync-docs に渡す |
| D3 quality-gates の行の拒否の列挙 | 情報(LOW) | 「it refuses before running ... on a dirty tree, a head that does not contain the default branch's tip, or a directory outside the feature worktree or off its branch」は、結びつきがない・走っていない、既定のブランチがない、スクリプトがない・実行できない、rebase の途中、の拒否を挙げていない。誤りではないが、列挙が全部に見える。直すなら「among other checks」を足すか、skill の動詞の表を指す |
| D4 計画の Progress checklist と Notes | 遅れている(LOW) | `Review artifact created` が未チェック(self-review の報告は c7ba86ef にある)。AC1〜AC13 の checkbox もすべて未チェック。self-review のあとの直し(`gate_run.go` への分割、SIGHUP、スクリプトが残したプロセスを止めること、rebase の途中の拒否、origin のない repo の次の手、export した定数)は self-review の報告の追記にだけあり、計画の Notes と Affected areas(`gate_run.go` がない)にはない。digest の外の節なので承認は崩れない。/verify は計画を書き換えないので、/sync-docs か /pr の前に記録する |

## Observational checks

`docs/evidence/verify-2026-10-13-org-gate-probe.txt` に script と出力を残した。HEAD c84d8e19 から `go build` した binary を、scratchpad の一時の repo(bare の origin、`origin/HEAD` は main、`git worktree add` の feature worktree)と一時の台帳(結びつきのある `scope_reserved` と leader の `spawned` の 2 行、per-plan、承認なし、見張りなし)で打った。スクリプトは stub で、`go test` と `run-test.sh` は走らせていない。herdr、agmsg、ネットワークは使っていない。

| # | 入力 | 結果 |
| --- | --- | --- |
| P1 | きれいな worktree、caller の環境に `RALPH_VERIFY_BASE=HEAD` など | `GATE: pass` と 4 行、rc 0。`gate_passed` を 1 行(seat_id 空、`by=probe herdr_pane=unset`、`log=` は引用符つき)。worktree の `git status --ignored` は空。stub は `RALPH_VERIFY_SCOPE=full` を見て、`RALPH_VERIFY_BASE` と `HARNESS_VERIFY_X` を見なかった |
| P2 | 未追跡のファイル | 拒否、rc 1、`?? untracked.txt` を並べた。manifest は 3 行のまま |
| P3 | origin/main が 1 コミット進んだ | 拒否、rc 1、次の手は `git fetch origin main` と `git rebase refs/remotes/origin/main`。manifest は変わらない |
| P4 | rebase のあと、`scripts/` から打ち、test の stub が 1 で終わる | `GATE: fail`、`REASON: test_failed`、最後の 20 行(`test line 11`〜`30`)、rc 1。`gate_failed` を 1 行 |
| P5 | static の stub が 2 で終わる | `REASON: static_failed`、rc 1。そのログに `running ./scripts/run-test.sh` はない |
| P6 | detached HEAD | 拒否、rc 1、次の手は `git switch feat/probe`。manifest は変わらない |
| P7 | main のチェックアウトから | 拒否、rc 1、feature worktree へ `cd` するよう言う |
| P8 | 知らない org_id | 拒否、rc 1 |
| P9 | 落ちたあとにもう一度 | pass、rc 0。同じ秒のログは `-2` の付いた名前になった。manifest には gate_passed、gate_failed 2 行、gate_passed の順で 4 行が並ぶ |

## Coverage gaps

- テストは走らせていない(/test の担当)。表の「満たす」は、コードの経路とテストの名前と assertion を読んで対応づけた結果で、テストが通るかは確かめていない
- 実機の確認は stub のスクリプトで行った。出荷する本物の `run-verify.sh` を gate が走らせる経路は `TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier` だけが見る。中断(信号)とロックの経路、`head_moved` と `tree_changed`、`recordGate` の中で org が変わっていたときの拒否は、probe では打っていない
- herdr の pane の中の leader(claude と codex)が雛形どおりに gate をバックグラウンドで打ち、出力のファイルを読めるかは未確認。codex の leader の sandbox で fetch とスクリプトが動くかも未確認(tech-debt の `:222`)。leader のツールが timeout で送る信号も未確認(`:228`)
- `tests/test-run-verify-concurrency.sh` の Case 6 は「信号から 1 秒後にまだ動いている」「2 秒以内に終わる」という時間の assertion を持つ。混んだ CI の runner で揺れるかは、/test と CI で見る。グループへの送信の部分は perl がないと飛ばす
- `run-verify.sh` は `.harness/state/verify-scope` を実行の最後に置くようになった(以前はブロックの中で書いた)。repo の中に、実行の途中でこのファイルを読むものはない(`git grep` で `run-verify.sh` 自身と計画だけ)。下流の project の `.ralph/local/verify.d/` が実行中に読んでいれば、前の実行の中身を読む。計画の Scope(36 行)はこの置き方を指定しているので、計画どおり
- `gate_passed` が保証するのは、その worktree の 2 つのスクリプトが通ったことまでで、ブランチがスクリプトを書き換えれば中身は変わる(tech-debt の `:229`)

## Verdict

- Verified: 静的解析はすべて rc 0(changed と full の `run-static-verify.sh`、`check-sync.sh`、`check-skill-sync.sh`、2 つの script の shellcheck、`go build`、`go vet`)。AC1〜AC13 は、コードと文書の経路と、それを固定するテストの名前で対応がつく。AC9 で書き換えた既存の assertion は計画の記録どおり 3 本。写しの 2 組は同一。FR-7 の 7b の 2 項目は満たし、7c の 3 項目は主張していない。HEAD の binary を一時の台帳で打った 9 通りは、AC1・AC2・AC3・AC7・AC11 の文のとおりに動いた
- Partially verified: AC10 の文書は合うが、LOW のずれが 4 件ある(D1 AGENTS.md の `gate_run.go`、D2 README の gate の段、D3 quality-gates の列挙、D4 計画の Progress と Notes)。どれも動きの誤りではなく、/sync-docs で直せる
- Not verified: テストが通ること(/test)、本物の leader の座席での手順(herdr、codex の sandbox、ツールの timeout)
- Verdict: pass
