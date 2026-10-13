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
- Verdict: pass(ブランチ全体、cycle 3 の HEAD a538c899 まで。上の 3 行は cycle 1 の HEAD c84d8e19 での判断で、cycle 2 と cycle 3 の判断は下の「Cycle 2」節の「判断(cycle 2)」と「Cycle 3」節の「判断(cycle 3)」)

## Cycle 2

- Date: 2026-10-13
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。cross-review cycle 1 の ACTION_REQUIRED #1(`docs/reports/cross-review-triage-org-gate.md:27`)を直した b04289c1、計画の Notes に 1 行足した 423a9f47、self-review cycle 2(bedc0ebc、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 6、Merge: yes)、その insight の行(e1bd6b81)のあとに走らせた
- Scope: cycle 1 の verify(48093964)からの差分 `git diff 48093964..HEAD`(HEAD e1bd6b81)。コードの変更は b04289c1 の 3 ファイル(`internal/org/gate.go` +26/-8、`internal/org/lockfile.go` +24/-3、`internal/org/gate_test.go` +208/-0)だけで、残りは /test・/sync-docs・cross-review・self-review の報告と evidence、計画の Notes、sync-docs の 6716a028 による AGENTS.md・README.md・quality-gates の 2 面の直しである。AC はすべて HEAD で見直し、AC5 を詳しく見た。テストは走らせていない(/test の担当)。この節の行番号はすべて HEAD e1bd6b81 のもの
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-gate-static.txt`: 上書きした。中身は HEAD e1bd6b81 での `./scripts/run-static-verify.sh`(changed scope)の出力だけ。出力に `$HOME` と scratchpad のパスは 0 件で、伏せる箇所はなかった。cycle 1 の Evidence の 1 つ目が言う changed と full の 2 回の出力は `git show 48093964:docs/evidence/verify-2026-10-13-org-gate-static.txt` で読める
  - `docs/evidence/verify-2026-10-13-org-gate-probe.txt`: 末尾に「Cycle 2: interrupt probe」を足した(script、HEAD の binary の出力、修正前の binary の出力)。HEAD の版は新しい版の先頭とバイト単位で一致するので、前の部分は変えていない。scratchpad のパスは前の部分と同じく `$SP` に置き換えた。script と 2 つの出力は、scratchpad の元のファイルと `cmp` で一致する
  - wrapper が書いた `docs/evidence/verify-2026-10-13-112202.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に verify の 1 行(`--cycle auto` で cycle 2、verdict pass、low 2 は C2-D1 と C2-D2)を足し、この報告と同じコミットに入れた。行は `./scripts/insights-append.sh --events-dir <scratchpad>` で作り、Edit で足した

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | ブランチの差分に `scripts/run-verify.sh` があるので、cycle 1 と同じく full fallback(`shared:scripts/run-verify.sh`)で golang の pack を選んだ。`gofmt: ok`、golangci-lint は `0 issues.`。shellcheck(hook、`scripts/*.sh`、`tests/test-*.sh`)、hook の `sh -n`、guard の awk、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の plan 参照、branch の secret scan(1bddc871..e1bd6b81、clean)も通った |
| `go build ./...`、`go vet ./internal/org/ ./internal/cli/` | 出力なし(rc 0) | |
| `gofmt -l` で b04289c1 の 3 ファイル | 出力なし | |
| `git diff --check 48093964..HEAD` と `git diff --check 1bddc871...HEAD` | rc 2。`docs/evidence` の外は出力なし | 指摘の 30 行は、どれも /test が残した生の出力の行末の空白(`docs/evidence/test-2026-10-13-org-gate-run-test.txt` の `has command codex ` などの 28 行と、`docs/evidence/test-2026-10-13-org-gate-mutation.txt:396`・`:399`)。スクリプトの出力をそのまま残したもので、直さない。CI と verifier の script は `git diff --check` を走らせない(`git grep 'diff --check' -- .github scripts tests` は 0 件)。cycle 1 の表の「出力なし」は、これらのファイルが入る前の c84d8e19 での結果 |
| `cmp` で org の SKILL.md の 4 面 | 同一 | b04289c1 は skill に触れていない |
| `./scripts/plan-visual.sh digest <plan>` | 69f088275e54 | `- Approved:` 行(4 行目)と一致。423a9f47 は `## Progress checklist` の中の Notes だけを変えたので、digest は変わらない |

### Spec compliance(cycle 2)

#### AC5 と b04289c1

AC5 の文は「スクリプトを走らせている間、manifest のロックは取られていない(スクリプトの中から台帳に書く動詞が待たずに通る)。中断されたら子のスクリプトを止め、何も記録しない」。cycle 1 で確かめたのはスクリプトを走らせている間の中断だけで、cross-review cycle 1 #1 は、スクリプトが終わってから記録までの間に届いた中断が記録を残しうると指摘した。b04289c1 のあとは、中断が届く時点ごとに次のように動く。

| 中断が届く時点 | HEAD の動き | 固定するテストと実機の確認 |
| --- | --- | --- |
| スクリプトの実行中 | cycle 1 と同じ。`runGateScript`(`internal/org/gate_run.go:180`)がスクリプトのプロセスグループを止め、`Gate` は `gateInterrupted`(`internal/org/gate.go:311-320`)を返す(`gate.go:287-289`) | `TestGate_InterruptStopsTheScriptsAndRecordsNothing`(`internal/org/gate_test.go:811`)、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(`internal/cli/org_gate_test.go:655`)。probe の Q4(SIGINT、スクリプトの子の `sleep` も消えた) |
| スクリプトのあと、worktree の読み直し(`gateCheckAfter`、`gate.go:569-584`) | git が ctx で切れれば `gate.go:292-294` で `gateInterrupted`。git が先に終われば、次の行の検査で止まる | この時点だけを狙うテストはない。`TestGate_InterruptAfterTheScriptsRecordsNothing` の 300 ms の待ち(`gate_test.go:909`)が足りないときにここを通り、文言は同じ(self-review cycle 2 の「Positive notes が頼っていた点」) |
| ロックを取る前 | `recordGate` の先頭の `ctx.Err()`(`gate.go:720-722`)が `context.Cause(ctx)` を返す | `TestRecordGate_InterruptRecordsNothing`(`gate_test.go:942`)の「ctx done before the lock」。時計を読まない(ロックの中に入らない)ことまで見る |
| ロックの待ち | `withManifestLockContext`(`internal/org/lockfile.go:65-67`)から `withFileLockContext`(`lockfile.go:83-102`)。待ちの timeout を呼び手の ctx から作り(`:90`)、ctx が終わっていれば「the wait was stopped」と言う(`:94-96`)。`Gate` はそれを `gateInterrupted` の文に替える(`gate.go:301-303`) | `TestGate_InterruptAfterTheScriptsRecordsNothing`(`gate_test.go:892`、3 秒以内に返り、manifest のバイト列が変わらない)と、`TestRecordGate_InterruptRecordsNothing` の「ctx done while waiting for the lock」。probe の Q1(SIGTERM)と Q2(SIGHUP)は信号から 0.02 秒で rc 1 を返し、ロックが空いたあとも manifest は 2 行のままだった |
| ロックの中、追記の直前 | `ctx.Err()`(`gate.go:747-749`) | `TestRecordGate_InterruptRecordsNothing` の「ctx done under the lock, right before the append」 |
| 追記のあと | 記録は残り、結果どおりに終わる(pass なら rc 0) | 中断なしの対照は「ctx not done records」と probe の Q3(ロックを 2 秒持たれても、待ってから `gate_passed` を記録し rc 0、2.02 秒) |

- 修正前の binary(789c8864 から作った。b04289c1 の親で、`internal/`・`cmd/`・`scripts/` は c84d8e19 と同じ)に同じ probe を当てると、Q1 と Q2 はロックが空くまで待ってから `GATE: pass` を出して rc 0 で終わり、`gate_passed` を 1 行ずつ書いた。cross-review の指摘は実機で再現し、HEAD では起きない。Q4 は修正の前後で同じ動き(rc 1、何も記録しない)だった
- AC5 の前半(スクリプトの間はロックを持たない)は変わらない。ロックを取るのは今も `recordGate` の中(`gate.go:724`)だけで、`TestGate_ManifestLockNotHeldWhileScriptsRun`(`gate_test.go:772`)が固定する
- 残る窓: 追記の直前の検査(`gate.go:747`)から `appendEvent`(`:750`)が終わるまでに届いた中断は、記録を消さない。`Gate` の doc(`gate.go:242-244`)は「up to the moment the event is appended」と書き、この境界を明かしている。中断が届いた時点で、gate の仕事は終わっている
- 判定: 満たす(実機でも確認)。cycle 1 の「満たす」は、スクリプトのあとから記録までの中断について誤っていた。cycle 1 の表は書き換えず、ここに記録する

#### ほかの AC

b04289c1 が変えたのは `Gate` の doc と `recordGate` の呼び出し(`gate.go:239-244`、`:301-303`)、`recordGate`(`gate.go:707-763`)、`lockfile.go` のロックの関数 4 つ、`gate_test.go` の追加だけである。

- AC1・AC3: pass と fail の記録は、ctx が終わっていなければ前と同じ道で追記する(「ctx not done records」が Details まで比べる。probe の Q3)。記録のキーの順(`gate.go:146-162`)は変わらない
- AC2・AC4・AC11: `Gate` の本体のうち `gate.go:245-300`(拒否の検査からスクリプトの実行と `gateCheckAfter` まで)と、拒否の検査と base の決め方の関数(`gate.go:333-624`)、環境の固定(`gate_run.go`)は変わっていない
- AC6・AC8・AC10・AC12・AC13: 該当するファイル(`watch.go`、`seat.go`、`spawn.go`、雛形、skill、`run-verify.sh` と写し、`tests/test-run-verify-concurrency.sh`、`approval_enforce_test.go`)は c84d8e19 から変わっていない(`git diff --stat c84d8e19..HEAD` が空)
- AC7: `gate.go`、`gate_run.go`、`org_gate.go` に `WatchGate` と承認の関数の呼び出しは 0 件のまま
- AC9: b04289c1 は `gate_test.go`(このブランチで新しく作ったファイル)への追加だけで、消した行は 0。base からあるテストのファイル(`prompts_test.go`、`approval_enforce_test.go`)は cycle 1 から変わっていない
- `withManifestLock` と `withFileLock` の呼び元(`approve.go:147`、`spawn.go:621`、`:796`、`verbs.go:1361`、`inbox.go:327`)は、`context.Background()` を渡す形になった。Background は終わらないので `lockfile.go:94` の枝に入らず、「timed out after 5s」の文と待ちの長さは前と同じ

#### cycle 1 の表の行番号の引き直し

cycle 1 の表の行番号は、見出しのとおり HEAD c84d8e19 のもの。b04289c1 で動いたのは次の参照で、ほかの参照(`gate.go:27`・`:146-162`・`:164`、`gate_run.go`、`org_gate.go`、`org_gate_test.go`、`watch.go`、`seat.go`、`spawn.go`、雛形、`prompts_test.go`、`approval_enforce_test.go`、`run-verify.sh`、`tests/`、skill、仕様、tech-debt)は HEAD でも同じ行を指す。sync-docs で直った文書の行(`AGENTS.md:90`、`README.md:252`・`:254`・`:256`・`:258`、quality-gates の `:82` と `:81`)も、同じ段落か行を指したまま。

| cycle 1 の参照(c84d8e19) | HEAD e1bd6b81 | 中身 |
| --- | --- | --- |
| `gate.go:243-304` | `gate.go:245-306` | `Gate`(AC1、AC7) |
| `gate.go:331-361` | `gate.go:333-363` | `gateOrgLife` |
| `gate.go:382-439` | `gate.go:384-441` | `gateCheckWorktree` |
| `gate.go:445-468` | `gate.go:447-470` | `gateCheckBranch` |
| `gate.go:497-510` | `gate.go:499-512` | `gateNoBaseRefusal` |
| `gate.go:514-530` | `gate.go:516-532` | `gateCheckAncestor` |
| `gate.go:534-550` | `gate.go:536-552` | `gateCheckScript` |
| `gate.go:567-582` | `gate.go:569-584` | `gateCheckAfter` |
| `gate.go:592-622` | `gate.go:594-624` | `resolveGateBase` |
| `gate.go:711-745` | `gate.go:713-763`(doc は 707 行から) | `recordGate` |
| `gate_test.go:319` | `:320` | `TestGate_PassRecordsGatePassed` |
| `gate_test.go:372` | `:373` | `TestGate_PassesFromASubdirectoryAndWithIgnoredFiles` |
| `gate_test.go:392` | `:393` | `TestGate_RefusedBeforeTheScripts` |
| `gate_test.go:614` | `:615` | `TestGate_ScriptFailuresAndChangesRecordGateFailed` |
| `gate_test.go:694` | `:695` | `TestGate_BaseResolution` |
| `gate_test.go:771` | `:772` | `TestGate_ManifestLockNotHeldWhileScriptsRun` |
| `gate_test.go:810` | `:811` | `TestGate_InterruptStopsTheScriptsAndRecordsNothing` |
| `gate_test.go:868` | `:869` | `TestGate_InterruptedBeforeTheScripts` |
| `gate_test.go:1061` | `:1269` | `TestGate_ScriptEnvironmentIsFixed` |
| `gate_test.go:1090` | `:1298` | `TestGateScriptEnv` |
| `gate_test.go:1141` | `:1349` | `TestGate_NoWatchGateNoApprovalCheck` |
| `gate_test.go:1219` | `:1427` | `TestGateEvents_AreNotASeat` |
| `gate_test.go:1260` | `:1468` | `TestLeaderActivityEventCount_GateEvents` |

`gate.go` は 239 行目より後ろが 2 行ずれ、`gate_test.go` は import の 1 行で 1 行、884 行目に足した 207 行でその後ろがさらにずれた。新しいテストは `TestGate_InterruptAfterTheScriptsRecordsNothing`(`:892`)と `TestRecordGate_InterruptRecordsNothing`(`:942`)。

### Documentation drift(cycle 2)

cycle 1 の D1〜D3 は sync-docs の 6716a028 で直った。`AGENTS.md:90` は「`gate.go` with the script running in `gate_run.go`」と書き、README には `ralph org gate` の段落(`README.md:260`)と動詞の表の `gate`(`:124`)が入り、quality-gates の 2 面の行は拒否の条件を全部挙げる。

b04289c1 が変えた動きを、文書の文と突き合わせた結果は次のとおり。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org gate --help`(`internal/cli/org_gate.go:73-74`) | はい | 「An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and records nothing」。スクリプトのあとの中断も記録しなくなったので、b04289c1 で文と動きが合った(cycle 1 の時点では、ロックを待つ間の中断で文が破れていた)。スクリプトが走っていなければ「stops the running script」は空振りするだけで、誤りではない |
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md:786` の「ツールに切られた gate は何も記録しない」、`:849` の「中断された gate は何も記録しない」)と leader の雛形(`internal/org/prompts/leader.md:205`) | はい | 上と同じ理由で合う。4 面は `cmp` で同一。SIGKILL で切られたときに残るスクリプトの扱い(`:786-788`)は b04289c1 と関係しない |
| quality-gates の 2 面、README、AGENTS.md、仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:73`) | はい | どれも gate の中断と manifest のロックの待ちに触れていない(`git grep -i -E` で `interrupt`・`中断`・`manifest lock`・`manifest.lock`・`lockfile` を探して 0 件)。b04289c1 が変えたものを記述していない |
| 中断の文言 | 情報 | probe の Q1 と Q2 で、2 つのスクリプトが通ったあと(log に `exited 0`)の中断にも「the scripts were stopped and nothing was recorded; the output so far is in ...」と出た。self-review cycle 2 の C2-2 が実機でも出た。文書はこの文を引いていない(`git grep 'scripts were stopped'` はコードとテストと self-review の報告だけ)ので文書のずれではない。直すなら C2-2 の推奨どおり |
| `lockfile.go` の doc | 情報 | `withFileLock` の doc(`lockfile.go:69`)は自分を「the flock mechanism behind withManifestLock」と呼ぶが、今の `withManifestLock` は `withManifestLockContext` から `withFileLockContext` を通り、`withFileLock` を通らない。`:28-29`・`:34` の「withFileLock serves both locks」も同じ。動きは変わらず、self-review の C2-3(`acquireFlock` の doc)と同じ編集で直せる |
| C2-D1 `docs/tech-debt/README.md` | いいえ(LOW) | self-review cycle 2 の「Tech debt identified」は C2-1〜C2-6 を 1 行にまとめて足すよう勧め、「orchestrator が足す」と書く。HEAD の README にその行はない(`org-gate` の行は 11 件あるが、C2-1〜C2-6 と `recordGate` を挙げる行は 0 件)。上限の最後の回で直さないと決めた所見なので、/pr の前に行を足す。前例は org-limits-reserve(`:185`)と org-plan-approval(`:220`)の cycle 2 の行 |
| C2-D2 計画の Progress checklist(cycle 1 の D4 の残り) | 遅れている(LOW) | 成果物の 3 つの checkbox は eb5ffdac(review と verification)と bf80094a(test)で付き、self-review のあとの直し(`:179`)と cross-review の直し(`:181`)も Notes に入った。残りは AC1〜AC13 の checkbox で、まだすべて未チェック。checkbox を付けても digest は変わらない(`plan-visual.sh` の digest は先頭の `- [x]` を `- [ ]` として読む)。Affected areas(`:66-74`)に `gate_run.go`、`gate_runverify_test.go`、`lockfile.go` はないが、この節は digest の中にあるので、書き足すと承認が崩れる。Notes の 2 行が代わりに記録しているので、書き足さなくてよい |

### Observational checks(cycle 2)

`docs/evidence/verify-2026-10-13-org-gate-probe.txt` の「Cycle 2: interrupt probe」に script と出力を残した。HEAD の worktree から `go build` した binary と、`git archive 789c8864` を scratchpad に展開して作った修正前の binary を、cycle 1 と同じ形の一時の repo と台帳(結びつきのある `scope_reserved` と leader の `spawned` の 2 行)で打った。manifest のロックは perl の `flock(LOCK_EX)` で `<state dir>/manifest.lock` を持って塞いだ。スクリプトは stub で、`go test` と `run-test.sh` は走らせていない。herdr、`~/.config/herdr`、agmsg、osascript、ネットワークは使っていない。

| # | 入力 | HEAD | 修正前(789c8864) |
| --- | --- | --- | --- |
| Q1 | 2 つのスクリプトが通り、ロックを 3 秒持たれている間に SIGTERM | rc 1、0.02 秒で返る。stderr は「interrupted (terminated signal received): ... nothing was recorded」。ロックが空いたあとも manifest は 2 行 | rc 0、信号の 0.77 秒後(ロックが空いたとき)に `GATE: pass`。`gate_passed` を 1 行書いた |
| Q2 | Q1 と同じで SIGHUP | rc 1、0.02 秒。「hangup signal received」。manifest は変わらない | rc 0、2.40 秒後に `GATE: pass`。`gate_passed` を 1 行書いた |
| Q3 | 中断なしで、ロックを 2 秒持たれる | 待ってから `gate_passed` を記録し rc 0(2.02 秒) | 同じ |
| Q4 | `run-test.sh` の子が `sleep 30` している間に SIGINT | rc 1、0.02 秒。「interrupt signal received」。子の `sleep` は消え、manifest は変わらない | 同じ |

### Coverage gaps(cycle 2)

- テストは走らせていない(/test の担当)。新しい 2 本(subtest は合わせて 4 つ)が HEAD で通るかと、self-review cycle 2 が変異を当てて読んだ結果が本当に赤になるかは、/test が確かめる
- 中断が `gateCheckAfter` の git の最中に届く時点は、テストも probe も狙って打っていない。読んだ限りでは `gate.go:292-294` で同じ文言になる
- 追記の直前の検査から `appendEvent` が終わるまでの窓(上の「残る窓」)は塞いでいない。`Gate` の doc に書いてあるので、契約の範囲内と判断した
- probe は macOS(darwin/arm64、go1.26.0)の一時の台帳で打った。Linux と、herdr の pane の中の leader が gate をバックグラウンドで打って信号を送る形は未確認(cycle 1 の Coverage gaps と同じ)
- cycle 1 の Coverage gaps のうち「中断(信号)とロックの経路は probe では打っていない」は、この回の Q1〜Q4 で埋まった。`head_moved`、`tree_changed`、`recordGate` の中で org が変わっていたときの拒否は、今も probe では打っていない

### 判断(cycle 2)

- Verified: b04289c1 のあと、スクリプトのあとの中断(ロックを取る前、ロックの待ち、ロックの中の追記の前)は何も記録せず rc 1 で終わる。コードの 3 か所の検査とそれを固定するテストを読んで対応をつけ、HEAD の binary で SIGTERM と SIGHUP がロックの待ちを 0.02 秒で切り、manifest を変えないことを確かめた。同じ probe で修正前の binary は記録して rc 0 を返したので、probe は直した欠陥を見分ける。help、skill の 4 面、雛形の「中断された gate は何も記録しない」は HEAD の動きと合う。静的解析(changed scope、full fallback)、`go build`、`go vet`、`gofmt -l`、`check-sync.sh`、`check-skill-sync.sh` は rc 0 で、計画の digest は 69f088275e54 のまま一致する。ほかの AC は b04289c1 で変わっていない
- Partially verified: AC10 の記録の 2 件(C2-D1 tech-debt の行がない、C2-D2 計画の AC の checkbox)。どちらも動きの誤りではなく、/sync-docs か /pr の前に記録を足せば閉じる
- Not verified: テストの実行(新しい 2 本を含む)、`gateCheckAfter` の最中の中断、Linux、本物の leader の座席での中断
- 判断: pass(cycle 2)

## Cycle 3

- Date: 2026-10-13
- Verifier: verifier subagent (Claude)。パイプラインの 3 回目(cycle 3)。PR #225 の CI(go.mod の `go 1.25.8`)で `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` が落ちたので、上限を 3 に上げて回した。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま(self-review cycle 3 と同じ)。その直しの 8b681dc3 と、self-review cycle 3(a538c899、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 2、Merge: yes)のあとに走らせた
- Plan: `docs/plans/archive/2026-10-13-org-gate.md`(f001bd4f で archive に移った。この報告の 4 行目の active のパスは cycle 1 の時点のもの)。`./scripts/plan-visual.sh digest` は 69f088275e54 で、`- Approved:` 行(4 行目)と一致する
- Scope: cycle 2 の verify(0bc87603)からの差分 `git diff 0bc87603..HEAD`(HEAD a538c899)。コードの変更は 8b681dc3 の 2 ファイル(`internal/cli/org_gate.go` +45/-7、`internal/cli/org_gate_test.go` +77/-0)だけで、残りは test cycle 2、sync-docs cycle 2(a1590e44)、cross-review cycle 2、walkthrough(2d6567b9)、archive(f001bd4f)、self-review cycle 3 の記録である。AC5 と中断を述べる文書を詳しく見て、ほかの AC が変わっていないことを確かめた。テストは走らせていない(/test の担当)。この節の行番号はすべて HEAD a538c899 のもの
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-gate-static.txt`: 上書きした。中身は HEAD a538c899 での `./scripts/run-static-verify.sh`(changed scope)の出力と、その下の見出し「CI/release toolchain」に続く go1.25.8 の `go version`・`go build ./...`・`go vet ./internal/...`・`gofmt -l`。出力に `$HOME` と scratchpad のパスは 0 件で、伏せる箇所はなかった。cycle 2 の中身は `git show 0bc87603:docs/evidence/verify-2026-10-13-org-gate-static.txt` で読める
  - `docs/evidence/verify-2026-10-13-org-gate-probe.txt`: 末尾に「Cycle 3: interrupt probe on the CI/release toolchain」を足した(script と、3 つの binary の出力)。HEAD の版は新しい版の先頭とバイト単位で一致するので、前の部分は変えていない。scratchpad のパスは前の部分と同じく `$SP` に置き換えた
  - wrapper が書いた `docs/evidence/verify-2026-10-13-134531.log` は gitignore の対象なので、コミットしない
  - CI のログは `gh run view` で読んだだけで、コミットしない。run 38401704870(84bbf4fb)と 38401983445(f001bd4f)が失敗、38407441217(a538c899)が成功。下で引く行番号は `--log-failed` と `--log` の出力の行
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に verify の 1 行(`--cycle 3`、verdict pass、low 4 は C3-D1〜C3-D4)を足し、この報告と同じコミットに入れた

### Static analysis(cycle 3)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | ブランチの差分に `scripts/run-verify.sh` があるので、cycle 1・2 と同じく full fallback(`shared:scripts/run-verify.sh`)で golang の pack を選んだ。`gofmt: ok`、golangci-lint(2.9.0、go1.26.0 で build)は `0 issues.`。shellcheck、hook の `sh -n`、guard の awk、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の plan 参照、branch の secret scan(1bddc871..a538c899、clean)も通った |
| `GOTOOLCHAIN=go1.25.8 go version` | `go version go1.25.8 darwin/arm64` | toolchain は module cache の `~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.8.darwin-arm64` にあった。手元の既定は go1.26.0 |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | rc 0、出力なし | go.mod の版で全体が build できる |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/...` | rc 0、出力なし | |
| go1.25.8 と go1.26.0 の `gofmt -l internal/cli/org_gate.go internal/cli/org_gate_test.go` | 出力なし | |
| `git diff --check 0bc87603..HEAD` | 出力なし(rc 0) | cycle 2 で指摘した `docs/evidence/test-*.txt` の行末の空白は、test cycle 2 の書き直しでなくなった |
| `cmp` で org の SKILL.md の 4 面 | 同一 | 8b681dc3 は skill に触れていない |
| `./scripts/plan-visual.sh digest docs/plans/archive/2026-10-13-org-gate.md` | 69f088275e54 | `- Approved:` 行と一致。a1590e44 が付けた AC の checkbox は digest が `- [ ]` として読み、f001bd4f の `PR created` は digest が飛ばす `## Progress checklist` の中にある(`scripts/plan-visual.sh:25-31`) |

### Spec compliance(cycle 3)

#### AC5 と 8b681dc3

AC5 の文は「中断されたら子のスクリプトを止め、何も記録しない」で、中断のエラーが信号の名前を言うことまでは求めていない。CI が落ちたのは、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(`internal/cli/org_gate_test.go:655`)が cause の文(`:661-663`)まで比べているためである。run 38401983445 の失敗文(5970〜5976 行、3 つの subtest)は「gate = org: gate: interrupted (context canceled): the scripts were stopped and nothing was recorded; ...」で、止めることと記録しないことは go1.25.8 でも成り立ち、信号の名前だけが抜けていた。

- 原因: go1.25.8 の `os/signal/signal.go:278-297` の `NotifyContext` は `context.WithCancel` で ctx を作り、信号で `c.cancel()` を呼ぶので、cause は `context.Canceled` のままになる。go1.26.0 の同じ関数は `context.WithCancelCause`(`signal.go:282`)と `signalError(s.String() + " signal received")`(`:294`)を使う。self-review cycle 3 が「1.25 のソースは読んでいない」と残した点は、module cache の go1.25.8 の GOROOT で読んで確かめた
- 直し: `gateSignalContext`(`internal/cli/org_gate.go:109-145`、関数は 126 行から)が `context.WithCancelCause` と `signal.Notify` を使い、最初の信号で `"<信号> signal received"` を cause にする。RunE の呼び出しは `org_gate.go:97`。gate の経路で cause を文にするのは `internal/org/gate.go:316`・`:319`・`:684`・`:721`・`:748`、`internal/org/gate_run.go:203`、`internal/org/lockfile.go:95` で、どれも CLI から渡る ctx を読むだけ(`git grep 'context.Cause'`)。`signal.NotifyContext` が残る呼び出しは `internal/cli/org_watch.go:153` で、cause を読まない
- 実機: 下の「Observational checks(cycle 3)」。go1.25.8 で build した HEAD の binary は 7 通りすべての中断で信号の名前を出し、修正前(f001bd4f)の binary は「context canceled」を出した。どちらも rc 1 で、manifest は変わらず、スクリプトの子は消えた。2 つ目の SIGINT は、どちらの binary も終わらせなかった
- CI: a538c899 の run 38407441217 は成功し、`ok github.com/yoshpy-dev/ralph/internal/cli 34.511s`(5126 行)。CI の Go の版はログに出ない。`.github/workflows/verify.yml` は `setup-go` を使わず runner の Go に任せる。f001bd4f の失敗文が「context canceled」なので 1.26 より前だとはわかるが、1.25.8 そのものかは確かめていない。`release.yml:19-21` は `go-version-file: go.mod` なので、release の build は `go.mod:3` の 1.25.8 を使う
- テスト: 新しい `TestOrgGate_SignalContextParentCauseAndStop`(`org_gate_test.go:729`)と `TestOrgGate_SignalContextKeepsLaterSignalsCaught`(`:763`、子プロセスで SIGHUP の cause と 2 つ目の信号での生存を見る)。通るかと、go1.25.8 で赤から緑に変わるかは /test が確かめる
- 判定: 満たす(go1.25.8 と go1.26.0 の実機でも確認)。中断のエラーが信号の名前を言うことは、HEAD では go1.25.8 でも成り立つ

#### 中断を述べる文書

| Doc | 文 | 判定 |
| --- | --- | --- |
| `ralph org gate --help`(`internal/cli/org_gate.go:73-76`) | 「An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and records nothing」 | 合う。エラーの文面は約束していない |
| `/org` skill の動詞の表(`.claude/skills/org/SKILL.md:181`) | 「SIGINT・SIGTERM・SIGHUP では走っているスクリプトを止め、何も記録しない」 | 合う。4 面は `cmp` で同一 |
| skill の `:786`・`:849`、leader の雛形(`internal/org/prompts/leader.md:205`) | 「ツールに切られた gate は何も記録しない」「中断された gate は何も記録しない」 | 合う |
| RunE のコメント(`org_gate.go:90-96`)と helper の doc(`:109-125`) | 「Later ones are caught too until stop runs, so a second Ctrl-C cannot end ralph before the group is stopped」「go.mod allows Go 1.25.8, which the release build uses」 | 合う。Q7 で、2 つ目の SIGINT のあとも ralph は動き続けた。go.mod と release.yml は上の CI の項のとおり |
| README、AGENTS.md、quality-gates の 2 面、仕様の注 | 中断の cause の文を書いていない | 該当なし。`git grep 'signal received'` は、報告のほかでは `org_gate.go` と `org_gate_test.go` だけ |

8b681dc3 のあとで、動きより多くを主張している文書はない。中断の文面を固定しているのはテスト(`org_gate_test.go:661-663`、`:795-796`)だけである。

#### ほかの AC

8b681dc3 が変えたのは `internal/cli/org_gate.go`(RunE のコメントと呼び出し、`gateSignalContext` の追加)と `internal/cli/org_gate_test.go`(723 行から 2 つのテストの追加)だけ。`internal/org/`、`cmd/`、`templates/`、`scripts/`、`tests/`、`.claude/`、`.agents/`、`.codex/`、README、AGENTS.md、quality-gates、仕様は 0bc87603 から変わっていない(`git diff --stat 0bc87603..HEAD -- <それらのパス>` が空)。

- AC1・AC3: `printGateResult` は中身が変わらず、38 行下がっただけ(下の引き直しの表)。probe の Q3 で pass の 4 行と `gate_passed` の記録を見た
- AC9: 8b681dc3 は `org_gate_test.go`(このブランチで作ったファイル)に足しただけで、消した行は 0(numstat 77/0)
- AC10: cycle 2 の C2-D1(tech-debt の行がない)と C2-D2(AC の checkbox)は a1590e44 で閉じた。`docs/tech-debt/README.md:231` に「Small findings of org-gate cycle 2」の行があり、計画の AC1〜AC13 は `- [x]` になった。cycle 3 の記録のずれは下の C3-D1〜C3-D4
- AC2・AC4・AC6・AC7・AC8・AC11・AC12・AC13: 該当するファイルは変わっていない

### Documentation drift(cycle 3)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| C3-D1 PR の walkthrough(`docs/reports/walkthrough-2026-10-13-org-gate.md`)が 8b681dc3 より古い | いいえ(LOW) | `:33` は「`signal.NotifyContext` で SIGINT、SIGTERM、SIGHUP を ctx の取り消しにする」のままで、今は `gateSignalContext`。節の見出し(`:31`)の commit の列に 8b681dc3 がない。`:55` は「上限は既定の 2 のまま」で、往復の表(`:57-66`)は cross-review cycle 2 で終わり、CI の失敗、8b681dc3、self-review cycle 3 の C3-1・C3-2 の行がない。walkthrough は 8b681dc3 より前の 2d6567b9 で書いた。slug の名前の記録なので、/sync-docs か /pr の前に直せる |
| C3-D2 walkthrough の AC の番号 | いいえ(LOW) | `:13` は `leaderActivityEventCount` の項目に「(AC13)」と書くが、計画では AC6(`docs/plans/archive/2026-10-13-org-gate.md:105`)。AC13 は archive した計画の TASK の検査(`:112`)である。8b681dc3 とは関係なく、2d6567b9 からある |
| C3-D3 `docs/tech-debt/README.md:228`(SIGKILL の行) | いいえ(LOW) | Related plan/report の列が `` `internal/cli/org_gate.go` (`signal.NotifyContext`) `` を指す。今の信号の処理は `gateSignalContext`。行の本文(「only a SIGINT, SIGTERM, or SIGHUP to ralph」)は動きと合っていて、古いのは関数名だけ |
| C3-D4 tech-debt に C3-1・C3-2 がない | いいえ(LOW) | self-review cycle 3 の「Tech debt identified」は (j) C3-1 と (k) C3-2 を `docs/tech-debt/README.md:231` の行に足すよう勧め、orchestrator が足すと書く。HEAD の行の項目は (a)〜(i) で、`gateSignalContext` と `C3-` は 0 件。cycle 2 の C2-D1 と同じ形で、/pr の前に足す |
| Go のコメントの計画のパス | 情報 | `git grep -n 'docs/plans/active/2026-10-13-org-gate'` は、報告のほかに Go のコメント 8 か所を返す(`internal/cli/org_gate.go:18`、`internal/cli/org_gate_test.go:23`、`internal/org/approval_enforce_test.go:1011`、`internal/org/gate.go:19`、`internal/org/gate_runverify_test.go:12`、`internal/org/gate_test.go:24`、`internal/org/prompts_test.go:741`・`:930`)。`scripts/archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` だけで(`:54-65`)、f001bd4f はそこの 12 か所を archive のパスに替え、active のパスは 0 件残る(static の「tech-debt README plan references」も OK)。archive 済みのほかの計画も Go のコメントに active のパスで残っている(`2026-10-12-org-plan-approval` は 9 か所、`2026-08-02-org-runtime-seats` は 8 か所)。この repo の慣例どおりなので、ずれに数えない |
| 報告の計画のパス | 情報 | self-review、test、sync-docs、cross-review triage、この報告の見出しの `docs/plans/active/...` は、それぞれを書いた時点の記録で、計画は当時 active にあった |
| PR #225 の本文 | 情報 | `gh pr view 225` の本文は cycle 1・2 の結果だけを挙げ、cycle 3 と 8b681dc3 を書いていない。中断の行(本文の 19 行目と 79 行目)は「止めて何も記録しない」で、動きと合う。cycle 3 を書き足すのは PR の更新のとき |
| この報告の cycle 1 の AC5 の行(`signal.NotifyContext`、`org_gate.go:96`) | 情報 | cycle 1 の表は書き換えず、下の引き直しの表に記録する |

### cycle 1・cycle 2 の表の行番号の引き直し

8b681dc3 で動いた参照は次の 3 つ。`org_gate.go` は元の 89 行目までが同じで、元の 96〜107 行目が 1 行、元の 108 行目から後ろが 38 行下がった。`org_gate_test.go` の新しいテストは 723 行目から入ったので、この報告が引く `org_gate_test.go:260`・`:316`・`:378`・`:421`・`:541`・`:655`・`:661-663` は同じ行を指したまま。cycle 2 の help の `org_gate.go:73-74` も変わらない。`internal/org/` と雛形、skill、`run-verify.sh`、`tests/` の参照は、cycle 2 の引き直しの表(HEAD e1bd6b81)のまま。計画の行の参照は、a1590e44 の AC の checkbox 13 行と f001bd4f の `PR created` の 1 行がどれも行の中の置き換えで行数を変えないので(numstat 14/14)、archive のパスでも同じ行を指す。

| 参照(c84d8e19、e1bd6b81 でも同じ) | HEAD a538c899 | 中身 |
| --- | --- | --- |
| `org_gate.go:141-143` | `org_gate.go:179-181` | pass の 4 行の出力(AC1) |
| `org_gate.go:134-171` | `org_gate.go:172-209` | `printGateResult`(AC3) |
| `org_gate.go:96`(`signal.NotifyContext`) | `org_gate.go:97`(`gateSignalContext` の呼び出し)。helper は `:109-145`(関数は 126 行から) | 中断の信号(AC5) |

### Observational checks(cycle 3)

`docs/evidence/verify-2026-10-13-org-gate-probe.txt` の「Cycle 3」に script と出力を残した。3 つの binary を用意した。HEAD の worktree から `GOTOOLCHAIN=go1.25.8` で build したもの、同じく go1.26.0 で build したもの、`git archive f001bd4f`(8b681dc3 の親。`internal/`・`cmd/`・`scripts/` の差は `org_gate.go` と `org_gate_test.go` だけ)を scratchpad に展開して go1.25.8 で build した修正前のものである。`go version <binary>` はそれぞれ go1.25.8、go1.26.0、go1.25.8 を返した。repo と台帳は cycle 2 と同じ形(結びつきのある `scope_reserved` と leader の `spawned` の 2 行)で、Q1〜Q4 は cycle 2 と同じ手順、Q5〜Q7 は新しく足した。cycle 2 との違いとして、`HOME` と `XDG_CONFIG_HOME` を scratch の下に向けた。スクリプトは stub で、`go test` と `run-test.sh` は走らせていない。herdr、`~/.config/herdr`、agmsg、osascript、ネットワークは使っていない。走らせたあと、`pgrep` で probe のプロセスが残っていないことを確かめた。

| # | 入力 | HEAD(go1.25.8) | 修正前 f001bd4f(go1.25.8) | HEAD(go1.26.0) |
| --- | --- | --- | --- | --- |
| Q1 | 2 つのスクリプトが通り、ロックを 3 秒持たれている間に SIGTERM | rc 1、0.02 秒。「interrupted (terminated signal received)」。manifest は 2 行のまま | rc 1、0.03 秒。「interrupted (context canceled)」。manifest は変わらない | rc 1、0.02 秒。「terminated signal received」 |
| Q2 | Q1 と同じで SIGHUP | rc 1、0.02 秒。「hangup signal received」 | rc 1、0.02 秒。「context canceled」 | rc 1、0.02 秒。「hangup signal received」 |
| Q3 | 中断なしで、ロックを 2 秒持たれる | 待ってから `gate_passed` を記録し rc 0(2.02 秒) | 同じ(2.01 秒) | 同じ(2.03 秒) |
| Q4 | `run-test.sh` の子が `sleep 30` している間に SIGINT | rc 1、0.05 秒。「interrupt signal received」。子の `sleep` は消えた | rc 1、0.02 秒。「context canceled」。子は消えた | rc 1、0.01 秒。「interrupt signal received」 |
| Q5 | Q4 と同じで SIGTERM | rc 1、0.01 秒。「terminated signal received」 | rc 1、0.01 秒。「context canceled」 | rc 1、0.01 秒。「terminated signal received」 |
| Q6 | Q4 と同じで SIGHUP | rc 1、0.03 秒。「hangup signal received」 | rc 1、0.01 秒。「context canceled」 | rc 1、0.02 秒。「hangup signal received」 |
| Q7 | SIGINT の 0.3 秒後に 2 つ目の SIGINT。スクリプトは TERM を trap し、1 秒かけて終わる | 2 つ目を送った時点で gate は動いていた。rc 1、1.06 秒。「interrupt signal received」。log に trap の 2 行 | 同じ動きで rc 1、1.09 秒。「context canceled」 | 同じ動きで rc 1、1.07 秒。「interrupt signal received」 |

- 中断のどの場合も manifest は変わらず、最後の manifest は 3 行(`scope_reserved`、`spawned`、Q3 の `gate_passed`)だった
- Q7 で 2 つ目の SIGINT が ralph を終わらせていれば、rc は 130 になり、gate のエラーの行も出ない。rc 1 でエラーの行が出たので、2 つ目の信号は捕まえられている。返るまでの 1.06 秒は、スクリプトが TERM の trap で 1 秒眠ってから終わるのを gate が待った時間である。修正前でも同じなのは、go1.25.8 の `NotifyContext` も stop まで `Notify` を外さないため(`signal.go:286`、`:307-310`)
- 修正前の binary だけが「context canceled」を出したので、この probe は 8b681dc3 が直した欠陥を見分ける

### Coverage gaps(cycle 3)

- テストは走らせていない(/test の担当)。self-review cycle 3 の follow-up 2 のとおり、`GOTOOLCHAIN=go1.25.8 go test ./internal/cli -run 'TestOrgGate_(InterruptStopsTheScriptAndRecordsNothing|SignalContext)'` を /test で回すのが、CI の失敗を手元で再現できるいちばん安い確認になる。CI は a538c899 で `internal/cli` が ok だったが、その Go の版はログに出ない
- golangci-lint は手元の 2.9.0(go1.26.0 で build)でしか回していない。go1.25.8 で回したのは `go build`、`go vet`、`gofmt -l` だけ
- `run-static-verify.sh` の golang pack は手元の `go` を使い、go.mod の `go` 行の版では回らない。今回の欠陥は Go 1.26 の標準ライブラリの動きに頼ったもので、型は同じなので go1.25.8 の `go vet` でも捕まらない。捕まえるには go.mod の版で `go test` を回す必要があり、決定論の検査を足すなら /test の側(手元の `go env GOVERSION` が go.mod の `go` 行より新しいときに、`GOTOOLCHAIN=go<go 行>` で gate の CLI のテストを 1 回回す)になる。toolchain の download が要るので、ネットワークがないときは飛ばす形になる
- Linux での probe は打っていない。CI(ubuntu-latest)で `internal/cli` が通ったことは、信号の文字列(`interrupt`・`terminated`・`hangup`)が Linux でも同じであることの間接の証拠にとどまる
- `gateSignalContext` の「親が先に終わると親の cause を残す」と「stop の cause は `context.Canceled`」は、テスト(`org_gate_test.go:729`)の assertion を読んだだけで、probe では打っていない
- 本物の leader の座席(herdr の pane、codex の sandbox、ツールの timeout が送る信号)での中断は、cycle 1・2 と同じく未確認

### 判断(cycle 3)

- Verified: 8b681dc3 のあと、go1.25.8 で build した binary の中断のエラーは信号の名前を言う(7 通りすべて)。同じ probe で修正前の binary は「context canceled」を出したので、probe は直した欠陥を見分ける。原因(go1.25.8 の `NotifyContext` は cause を付けない)は go1.25.8 のソースで確かめた。止めることと記録しないことは修正の前後で同じで、2 つ目の SIGINT も ralph を終わらせない。help、skill の 4 面、leader の雛形の中断の文は、動きより多くを主張していない。static は既定の toolchain で rc 0、go1.25.8 では `go build ./...`・`go vet ./internal/...`・`gofmt -l` が rc 0。計画の digest は 69f088275e54 のまま一致する。CI は a538c899 で成功した
- Partially verified: 記録のずれが 4 件(C3-D1・C3-D2 walkthrough、C3-D3・C3-D4 tech-debt)。どれも LOW で、動きの誤りではない
- Not verified: テストの実行(go1.25.8 を含む)、CI の Go の正確な版、Linux での probe、本物の leader の座席での中断
- 判断: pass(cycle 3)
