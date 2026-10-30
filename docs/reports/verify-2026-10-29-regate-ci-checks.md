# Verify report: regate-ci-checks

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-29-regate-ci-checks.md`(AC1〜AC10、D1〜D7、Verify plan、Progress checklist の S0〜S7・S4a・S4b・F1〜F4)。issue #264
- Verifier: verifier サブエージェント(`/verify`、パイプライン cycle 1)
- Scope: `git diff origin/main...HEAD`(base c03647f9、HEAD 23dfc9c7、56 ファイル、+6709/-349)。テストは打っていない(`/test` の範囲)。打ったのは `run-static-verify.sh` と同期の検査のほかに、`go build ./...`、help を読むための binary の build(scratchpad)、その binary での `ralph config get` と `--wait-checks` の断りの probe だけで、`go test` は打っていない
- Evidence: `docs/evidence/verify-2026-10-29-regate-ci-checks.log`(gitignore。static の全出力、同期の検査、`git diff --check`、写しの `cmp`、U+FFFD、計画の digest、AC1 の削除行、古い読み手の grep、probe、help 2 つの本文)。wrapper 自身のログは `docs/evidence/verify-2026-10-30-033536.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 既定(`local`)では今の regate と同じ動きをする(今の regate のテストが変えずに通る) | Partly(動きは同じ。文言 1 つが変わった) | `p.Checks` が空か `local` なら `ci` は nil のままで(`internal/org/regate.go:529-535`)、gate だけの道は前と同じ `regateGate`(`:550-553`)、merge の道は `if ci != nil`(`internal/org/regate_merge.go:277`)の外で前のまま。local は Exec を一度も呼ばない(`TestRegateCI_LocalChecksRunTheScriptsAndCallNoExec`、`internal/org/regate_ci_test.go:746`)。org 層の今のテスト(`regate_test.go`、`regate_merge_test.go`)は削除行が 0。CLI の local の失敗の文は前と同じ組み立てになる(`internal/cli/org_regate.go:316-337`)。ただし local の `UP_TO_DATE:` の行が「a gate_passed of HEAD」から「a pass of HEAD」に変わり(`internal/cli/org_regate.go:358`)、`internal/cli/org_regate_test.go` の期待 2 か所(`:160`、`:412`)を書き換えた。help を固定する 1 行も延ばした。括弧の「変えずに通る」は、この文言について字義どおりには満たさない。Progress の S7 に記録がある(所見 I1) |
| AC2 `ci` ではスクリプトを走らせず、merge と push のあと PR の checks で記録する(`gate_ci_passed`、`via=regate_ci`、head は push した SHA、base は取り込んだ先頭)。古い event 名だけを選ぶ読み手は読まない | Met(静的) | merge の道は `regateCIMerged`(`internal/org/regate_ci.go:183-207`)がスクリプトを呼ばずに push(`regateCIPush`、`:245-314`)してから `regateChecksWithWait` に進む。記録は `regateChecks`(`:519-568`)で、Head は `t.head`(merge の道では `regate_merge.go:276` で merge commit)、Base は `t.base.sha`(`regate.go:527` の既定のブランチの先頭)、Via は `gateViaRegateCI`。event 名は `gateRecord.event()`(`internal/org/gate_record.go:73-84`)が via で分ける。origin/main の読み手は `case EventGatePassed, EventGateFailed` だけを選ぶ(`mergeorder.go:481`、`watch.go:1401`。計画の `:458` は計画の base 5eba4013 での行)ので、名前の違う CI の記録は読まない。テストは `TestGateRecordDetails_CIRecord`(`internal/org/gate_test.go`)、`TestRegateCI_MergePathRecordsTheChecksOfThePushedMergeCommit`(`internal/org/regate_ci_merge_test.go:124`)、`TestRegateCI_GateOnlyRecordsFromThePRsChecks`(`regate_ci_test.go:392`) |
| AC3 `ci` の passed で merge-order が `ready`、`ralph org merge` が merge できる | Met(静的) | merge-order の収集が 4 つの event を選び(`internal/org/mergeorder.go:487`)、`classifyGatePassed`(`:1153`)は via を見ない。テストは `TestMergeOrder_ReadsTheCIRecordsOfRegate`(`internal/org/mergeorder_test.go:894`、ready・rebase・regate の行)と `TestMerge_MergesABranchWhoseLatestRecordIsAGateCIPassed`(`internal/org/merge_test.go:468`) |
| AC4 CI が落ちたら `gate_ci_failed`(`ci_failed`、`via=regate_ci`)、merge-order は `ready` を出さず次の手を出す | Met(静的) | 落ちた check があり pending がなければ reason `ci_failed`(`regate_ci.go:478-479`)。`isGateReason` が `ci_failed` を含む(`gate_record.go:157`)。merge-order の文は `mergeRegateFailedDetail` の via=regate_ci の分かれ(`mergeorder.go:1111-1114`)。CLI は `GATE: fail`・`REASON: ci_failed`・`CHECKS: failed` と落ちた check(`internal/cli/org_gate.go:211-221`)。テストは上の merge-order の failed の行と `TestOrgRegate_CIFailureAfterAMergeNamesTheChecksAndKeepsThePush`(`internal/cli/org_regate_test.go:1070`) |
| AC5 終わっていないときは記録せず `CHECKS: pending` と終了コード 6、読めないときは exit 1、打ち直すと D6 の道、`--wait-checks` は lock を外して待つ | Met(静的) | pending の 3 つ(PR の head が違う、pending の check がある、0 件)は `readRegateChecks`(`regate_ci.go:436-462`)、記録しない終わりは `regateChecksPending`(`:491-495`)。CLI は `ErrRegateChecksPending` を包み(`internal/cli/org_regate.go:303-304`)、`ExitCode` が 6 にする(`internal/cli/exitcode.go:43-44`)。gh の失敗と出力が JSON でないときは refused(`regate_ci.go:446-453`、exit 1)。D6 は `regate.go:550-551`。待ちは lock を外して読むだけで、終わったら 1 回だけ取り直す(`regateChecksWithWait` `:677-735`、`poll` `:756-787`、`regateCIRecheck` `:807-863`)。テストは `TestRegateCI_GateOnlyRecordsNothingWhileTheChecksAreNotDone`(`regate_ci_test.go:486`)、`TestOrgRegate_CIPendingExits6AndTheNextRunReadsTheChecks`(`internal/cli/org_regate_test.go:1120`)、`TestRegateCI_WaitChecksGoesOnWhileAnotherHoldsTheLock`(`regate_ci_wait_lock_test.go:25`)ほか待ちの 11 本。D5 との違い(lock の下でまた pending なら待たずに 6 で終える)は Progress の F1 に記録がある |
| AC10 `ci` の passed があり HEAD が origin の先頭でないとき、push だけの打ち直しをせずに断る | Met(静的) | `regateHeadNotOnOrigin` が via=regate_ci の pass を先に分け、origin の先頭が HEAD の子孫なら `regateCIPassBehindText`、それ以外は `regateCIPassMovedText` で断る(`internal/org/regate_merge.go:86-95`、`:129`、`:143`)。この分かれは local の regate でも効く。`regateResetHint` は via=regate だけを見る(`:165-167`)。テストは `TestRegate_ACIPassIsNotPushedAgainOnceOriginMovedOff`(`internal/org/regate_merge_test.go:1498`)と `TestRegate_ACIRecordNamesNoResetKeep`(`:1569`) |
| AC6 `ci` で origin が GitHub でない、gh がない、open な PR がないときは merge の前に断る | Met(静的) | `regateCIPreCheck`(`regate_ci.go:66-136`)が `ReadMergeOrigin`、`LookPath("gh")`、`findOriginPR`(`internal/org/prchecks.go:89-109`。Foreign・NotOpen・WrongBase を含む)の順に見る。呼ぶのは fetch の直後で、HEAD と origin の確かめと merge の前(`regate.go:529-535`)。計画の D2 は fetch の前に置いていたが、fetch のあとにしたことは Progress の S4a に記録がある。fetch が変えるのは `refs/remotes/origin/*` だけ。テストは `TestRegateCI_RefusedBeforeAnythingChanges`(`regate_ci_test.go:166`)、`TestRegateCI_ChecksOriginGHAndThePRBeforeEveryPath`(`:253`)、`TestOrgRegate_CIWithNoPRIsRefusedBeforeTheMerge`(`internal/cli/org_regate_test.go:1202`) |
| AC7 push の失敗は読み返した先頭で 3 つに分ける | Met(静的) | `regateCIPush`(`regate_ci.go:286-313`)が `regateReadOriginTip`(`internal/org/regate_git.go:74`、`git ls-remote origin refs/heads/<b>`)の結果で分ける。元の先頭なら merge commit を戻して refused、merge commit なら `regateCILanded` で checks に進み、子孫・別のコミット・ブランチなし・読めないは HEAD を残して refused。テストは `TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack`(`internal/org/regate_ci_merge_test.go:248`、7 つの場面) |
| AC8 ルートの `ralph.toml` が `ci`、template が `local` とコメント、`ralph config get` が値を返し、ほかの値は `config.Load` が断り、ずれの検査がキーを見る | Met(静的と probe) | `OrgGateConfig`(`internal/config/config.go:581-602`)、既定(`:660-662`)、検証(`:944-946`)、`ralph.toml:18-19`、`templates/base/ralph.toml:269-279`、`orgTemplateDrift` の行(`internal/config/defaults_sync_test.go`)、`TestRootRalphToml_KnownKeysAndReviewSwitchesOn`(`internal/config/root_ralph_toml_test.go:27`)。build した binary で、worktree のルートが `ci`、ralph.toml なしと template の写しが `local`(どれも rc 0)、`"remote"` が rc 3 と検証の文になった |
| AC9 `/director`・`/org` の skill、README、仕様 FR-7 の注、AGENTS.md、`ralph org regate --help` が `ci` を書き、写しの検査と固定するテストが通る | Partly(文書のずれ 5 つ) | 書いた箇所: `.claude/skills/director/SKILL.md:871`、`:883`、`:962-1027`、`.claude/skills/org/SKILL.md:242`、`:1237-1246`、`:1336-1339`、`:1352-1418`、`README.md:280`、`docs/specs/2026-10-07-org-multi-org-director.md:132`(FR-7 の下)、`AGENTS.md:91`、help(`internal/cli/org_regate.go` の Long)。写しは `cmp` で 6 つとも一致、`check-skill-sync.sh` と `check-sync.sh` が通った。固定するテストは `TestDirectorSkill_PinsTheCIRegate`(`internal/org/director_skill_sync_test.go:1139`)と `TestOrgSkill_PinsTheCIRegate`(`internal/org/org_skill_send_sync_test.go:358`)があり、実行は `/test`。下の Documentation drift の V1〜V5 が残る |

設計判断との照合:

- D1: `[org.approval].mode` と同じ検証の形で、空文字も断る(`config.go:940-946`)。regate は `newOrgRuntime` の設定を読む(`internal/cli/org_regate.go:221`)
- D3: 上の AC7 のとおり。push か読み返しの最中の中断も HEAD を残す(`regate_ci.go:264-273`、`:289-292`)
- D4: すべて skipping は refused(`regate_ci.go:480-482`)で `ci_failed` にしない。落ちた check と pending が両方あれば pending(`:458-460`)。merge の判定はこの場面で NotPassed になるので、D4 の括弧の「merge と同じく」は正しくない。Progress の S3 にずれとして記録がある。log は `gh pr checks` の JSON(`writeGateChecksLog`、`:573-586`)
- D5: `regateWaitChecksRefusal`(`regate_ci.go:594-606`)が範囲(0〜6h)と local での 0 より大きい値を、lock と台帳より前に断る(下の probe)
- D7: via を見る 3 か所は `mergeorder.go:1111`、`regate_merge.go:86`、`:167`。watch の数え方は `gate_ci_*` を足さず(`internal/org/watch.go:1405`)、注釈がその理由を書く(`:1380-1384`)

計画の digest: `./scripts/plan-visual.sh digest` が `994880bfa86d` を返し、`- Approved: 2026-10-30 sha256:994880bfa86d`(`:4`)と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | 言語の範囲は changed で、`ralph.toml` が分類されないので full に落ち(`unclassified:ralph.toml`)、選ばれた pack は golang だけ。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`(root と template の 10 本ずつ)、guard の awk、settings の `jq`、Codex の hook の検査 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照がすべて通った。secret scan は c03647f9..23dfc9c7 で clean |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6(`ralph.toml` を含む) |
| `git diff --check origin/main...HEAD` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | |
| `cmp`(`/director` と `/org` の skill の 3 つの写し) | 6 つとも rc 0 | |
| `go build ./...`、`go build -o <scratchpad>/ralph-264 ./cmd/ralph` | rc 0 | go1.27.3(go.mod と同じ) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org regate --help`(`internal/cli/org_regate.go` の Long) | No(V1、既知) | `:70-72` は、local の `UP_TO_DATE:` を「a gate_passed of HEAD on that tip」のときと書くが、`passedOn`(`internal/org/regate.go:662`)は local でも `gate_ci_passed` を UpToDate と読む。`:113-114` の PUSH_ONLY の条件「a gate_passed」は、`regate_ci` の pass が `regate_merge.go:86` で断られるので字義どおりには合う。どちらも /sync-docs への引き継ぎにすでにある。ci の段落(help の出力の 82〜137 行)、終了コード、`--wait-checks` の範囲と断りは、コードと probe に合う |
| `docs/quality/quality-gates.md:82`、`templates/base/docs/quality/quality-gates.md:81` | No(V2、既知) | regate は「runs the same two scripts, records the same events with `via=regate`」と書くだけで、`ci` ではスクリプトを走らせず `gate_ci_*` を書くことがない |
| `printRegateResult` の doc comment(`internal/cli/org_regate.go:246`) | No(V3、既知) | 「recorded a gate_passed」と書くが、`gate_ci_failed` の記録と ci の push のあとも同じ行を先に出す |
| `/org` skill の `regate` の行(`.claude/skills/org/SKILL.md:242` と写し 3 つ) | No(V4、新規) | (a) 「その HEAD とその先頭の `gate_passed` なら … `UP_TO_DATE:`」は V1 と同じく `gate_ci_passed` を落としている。(b) 「台帳ごとに 1 つ」と「regate が走っている間は、その org の `spawn` … と `start` … が断られる」に、`ci` の待ちの例外がない。同じ skill の手順の節(`:1355`、`:1412-1414`)と help の出力の 149〜152 行には例外がある。同じ行の `--wait-checks` の文は lock を外すと書く |
| `README.md:280` | No(V5、新規) | (a) 「One regate, merge, or cleanup runs per ledger at a time … and while it runs `ralph org spawn` … are refused」に `ci` の待ちの例外がない(merge の `--wait-checks` の例外も前から書いていない)。(b) ci の regate は「before anything else」に origin・gh・PR を確かめると書くが、確かめるのは lock・結びつき・worktree の確かめと fetch のあと(`internal/org/regate.go:529-535`。help は「Right after the fetches」) |
| `/director` skill(「merge のあとの取り直し」`:962-1027`、`:871`、`:883`) | Yes | 値の読み方(`ralph config get`)、local と ci の違い、事前の確かめと終了コード 1、`--wait-checks 60m`、`ci_failed` で merge commit が origin に残ること、終了コード 6 を `GATE:` でなく終了コードと `CHECKS:`・`NEXT:` で見分けること、2 回続けば人に伝えることが、コードと help に合う |
| `/org` skill の記録の形(`:1237-1246`)と手順(`:1336-1339`、`:1352-1418`) | Yes | event を分けた理由、`ci_failed` の読み方、待ちの間の spawn と start の例外が D4・D5・D7 に合う |
| 仕様 FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md:132`) | Yes | D1〜D7 と Critical forks に合う。名指したテスト(`TestOrgRegate_CIPendingExits6AndTheNextRunReadsTheChecks`)はある。本物の CI では打っていないので FR-7 の印を変えないと書き、印は `[ ]` のまま |
| `AGENTS.md:91` の地図 | Yes | `regate_ci.go`、`gate_ci_passed`/`gate_ci_failed`、`via=regate_ci`、merge と共用する `prchecks.go` |
| `ralph org merge-order --help` | Yes | ready の条件と failed の行が `gate_ci_*` を書く |
| `templates/base/ralph.toml:269-279`、`ralph.toml`、`scripts/check-sync.sh` の KNOWN_DIFFS のコメント、`.claude/rules/ralph/model-routing.md:151` | Yes | template に `[org.gate]` とコメント、ルートに `ci`。template の model-routing.md には該当の文がない |
| PR の前の gate(`ralph org gate`) | Yes(変わっていない) | `internal/org/gate.go`、`gate_run.go`、`internal/org/prompts/` に差分がない。`internal/cli/org_gate.go` の差分は `printGateFailDetail` の引数と `ci_failed` の表示だけで、help の Long は同じ |
| `docs/tech-debt/README.md:290`、`:395-397` | Yes | 数を読み直した。`regate.go` 763→995、`regate_ci.go` 901、`org_regate_test.go` 760→1271、`regate_test.go` +51、`regate_merge_test.go` +114。分けた 4 つは 767・630・726・377 行で、どれも 800 未満 |

Info(ずれではない):

- I1: AC1 の文言の変化(上の表)。`is a gate_passed of HEAD` を読むものは、help の 2 行と PUSH_ONLY の行とそのテストだけで、skill はこの文を読まない
- I2: 計画の D4・D7 が挙げる `mergeorder.go:458` と `watch.go:1401` は、計画の base 5eba4013 での行。HEAD では `:487` と `:1405`
- I3: `ci` の regate も、走らせない 2 つのスクリプトが実行できるファイルとしてあることを確かめる(`regateCheckWorktree` の `gateCheckScripts`、`internal/org/regate.go:844`)。help の最初の段落は両方の値に同じく書くので、help とコードは合う。計画にはこの点の記述がない
- I4: `docs/specs/2026-08-01-org-runtime.md:37` の (c)(2026-10-27 の注)は「`ralph org regate` は変えておらず、今までどおり 2 つのスクリプトを走らせる」と書く。日付のある注で、その仕様は履歴を書き換えない約束なので、ずれとして数えない

## Observational checks

- build した binary(`<scratchpad>/ralph-264`)の probe。worktree のルートで `ralph config get org.gate.regate_checks` が `ci`、ralph.toml のない dir と template の写しの dir で `local`(どれも rc 0)、`regate_checks = "remote"` で rc 3 と `[org.gate].regate_checks "remote" must be "local" or "ci"`。同じ dir の `ralph org regate --branch feat/x --wait-checks 1m` は `org: load config: …` で rc 1
- `--wait-checks` の断り: local で `1m`・`-1s`・`7h`、ci で `7h`・`-1s` がどれも rc 1 で、`RALPH_ORG_STATE_DIR` に向けた dir は作られなかった(台帳を読む前に断る、help の「before the ledger is read」に合う)
- `.github/workflows/verify.yml` は `pull_request` で `./scripts/run-verify.sh` を走らせる(計画の前提)

## Coverage gaps

- テストの実行(`/test`)。上の表のテストは読んで、主張と照らしただけ
- 本物の GitHub の PR と CI での `ci` の regate(仕様の注も打っていないと書く)。PR の head の更新の遅れ(計画の Open questions)は未確認
- 偽の gh を使った ci の regate の probe はしていない。台帳と bare origin と偽の gh を組むと `/test` の fixture と同じになるため
- 古い binary(origin/main)が `gate_ci_*` を含む台帳を読む実測はしていない。古い読み手が event 名で選ぶことはソースで確かめた
- 小さく足せる検査: V1・V4 の「`gate_passed` なら UP_TO_DATE」の文が残らないように、help と `/org` skill の regate の行に `gate_ci_passed` の語があることを `TestOrgSkill_PinsTheCIRegate` と help の固定テストに 1 行ずつ足す

## Verdict

- Verdict: pass
- Verified: AC2〜AC8 と AC10(静的。AC8 は probe も)、D1〜D7 との照合、PR の前の gate が変わっていないこと、`run-static-verify.sh`・`check-skill-sync.sh`・`check-sync.sh` の通過、`git diff --check`、U+FFFD、写しの一致、計画の digest、tech-debt の数
- Partially verified: AC1(動きは同じ。local の `UP_TO_DATE:` の文言が変わり、CLI のテストの期待 2 か所を書き換えた。Progress の S7 に記録あり)、AC9(LOW の文書のずれ V1〜V5。V1〜V3 は /sync-docs に引き継ぎ済み、V4 と V5 は新規で同じく /sync-docs に渡す)
- Not verified: テストの実行、本物の GitHub の CI、古い binary での実測
- Result: pass。動きが受け入れ条件と食い違う所見はない。所見は LOW 5 つ(V1〜V5、どれも文書)と Info 4 つ(I1〜I4)
