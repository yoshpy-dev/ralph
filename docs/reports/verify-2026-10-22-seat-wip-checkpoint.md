# Verify report: seat-wip-checkpoint

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(Status: Approved。`./scripts/plan-visual.sh digest` は `b98f2806ab31` で、`- Approved:` 行と一致)
- Verifier: verifier サブエージェント(Claude Code、パイプライン cycle 1)
- Scope: `git diff c90d01b4...HEAD`(HEAD a79b0f82、41 ファイル、+1449/-283)。S1 88706897、S2 416993cf、S3 3dedaef9、S4 b5b12ae9、S5 80cb817a・075b2160、self-review の報告 9cf4ed46 と insight e5836fe3、その LOW 7 件の直し a79b0f82 を含む。判定は HEAD の tree に対して行った
- Evidence: `docs/evidence/verify-2026-10-22-seat-wip-checkpoint.log`(run-static-verify.sh の出力をそのまま書き、末尾に rc と所要時間を足した。スクリプト自身も `docs/evidence/verify-2026-10-22-102125.log` を書いた。どちらも `docs/evidence/*.log` の gitignore によりコミットしない)

## Spec compliance

テストの実行は `/test` の担当なので、AC1〜AC5b はコードとテストの本文を読んで確かめた。AC6 は evidence の文書を読み、hook が S1 のあと変わっていないことを確かめた。AC9 の run-verify は 9d29ee0d での orchestrator の結果を引く。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 2 つの hook(root と写しは byte 同じ)が、feature ブランチの汚れた木で `RALPH_WIP_CHECKPOINT=off` のときコミットせず skipped の行を書く。値なし・`on`・`OFF`・空はコミットする。main はコミットしない | 満たす。実行は `/test` | `.claude/hooks/session_end_summary.sh:36-51`、`precompact_checkpoint.sh:39-54`。判定は `[ "${RALPH_WIP_CHECKPOINT:-}" = "off" ]` の完全一致で、`then` は md に 1 行足すだけ、`else` に元の `git add -A` と `git commit` がある。`case` の `main\|master\|unknown` と、木がきれいなときの枝は変わっていない。`cmp` で root と `templates/base/` の 2 本ずつが一致。`tests/test-wip-checkpoint-hooks.sh:236-267` は hook 2 本 × 写し 2 本ごとに、unset・`on`・`OFF`・空でコミットあり、`off` で HEAD と index が不動、main の汚れた木と feature のきれいな木で unset・`off` ともにコミットなし、を打つ。`TestWIPCheckpointOff_IsWhatTheHooksRead`(`internal/org/driver/paneenv_test.go`)が hook 4 本の判定の文字と `WIPCheckpointOff` を突き合わせる |
| AC2: 本番の `Herdr.PaneEnv` が `AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off` の順で、座席と headless の director の `tab create` に `--env` が 2 つ付き、確かめは 1 回 | 満たす。実行は `/test` | `internal/org/driver/paneenv.go:25-27`(`SeatPaneEnv` は呼ぶたびに新しい slice)。本番の組み立て 2 か所(`internal/cli/org.go:361`、`internal/cli/org_director_headless.go:789`)が `driver.NewHerdr(..., driver.SeatPaneEnv()...)`。`TabCreate`(`herdr.go:244-250`)は前から要素ごとに `--env` を足す。テスト: `TestSeatPaneEnv_ListsBothEntriesInOrder`、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`(help 1 回のあと tab create 2 回とも `--env` 2 つ)、`TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`、`...ConcurrentCallsCheckOnce`、spawn の happy path(`org_test.go:424-429`)、`TestOrgDirectorStart_LaunchesRegistersAndStops`(`org_director_headless_test.go:208`) |
| AC3: launch_cmd の起動スクリプトが check-in と exec の前に一覧のすべてを export | 満たす。実行は `/test` | `internal/org/director_cmd.go:327-330` が `driver.SeatPaneEnv()` を順に `NAME='value'` と `export NAME` で書き、そのあと check-in の行を書く。`TestDirectorLaunchScript_TextQuotesEveryValue` が全文と、agmsg home なしでも 2 つが check-in の直前に並ぶことを固める。`TestDirectorLaunchScript_ChecksInThenExecsTheAgent` と `TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` は、端末の環境を `on` にしたうえで check-in と agent が両方 `off` を見ることを確かめる |
| AC4: `--env` のない herdr で値を付けずに pane を作り、spawn と start は成功し、stderr に 2 つの値と帰結を書く warning が 1 行。説明が「1 つの値」と書かない | 満たす。実行は `/test` | `herdr.go:320-343` の `warnNoPaneEnv` は PaneEnv の要素ごとに `paneEnvConsequence`(`paneenv.go:32-42`)の節を `; ` でつなぎ、助言を `. Use ...` の別の文にする。`TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence` が 6 通りの PaneEnv で全文を、`TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce` が help なし・失敗・時間切れを、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` が CLI の 1 行と 2 つの帰結の語を固める。`herdr.go`・`paneenv.go`・`agmsg.go` を `one entry`・`one value`・`single entry` で grep して当たりなし |
| AC5: 運用規律に spawn の失敗の 1 項((a) `seat left active` で stop と status を確かめてから打ち直し、だめなら BLOCKED。(b) 打ち直し。(c) 2 回続けば BLOCKED) | 満たす。(b) は `org: spawn step` で始まる誤りに絞り、それ以外を BLOCKED にした(計画の Progress に記録あり) | `internal/org/prompts/leader.md:419-444`。引く語はどれもコードにある: `spawn failed:`(`internal/cli/org.go:846`)、`rejected:`(`:838`)、`org: spawn step` と `seat left active`(`internal/org/spawn.go:2474`、`:2490`)、`refuseStaleRespawn` の `seat left active`(`:2607`)、`retry with a larger --timeout-ms`(`:2498`)、`ralph org status` の `(active)`(`internal/cli/org.go:1782`)、stop の `--force`(`:1485`)。spawn の失敗は `return result.Err` で終了コード 1、`main.go:20` が stderr にも出す。`TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError` が文と入れ子と語の出どころを固める。手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は diff に入っていない |
| AC5b: 手順 14 が gate の打ち直しと push の前に `git log --format='%h %s' origin/<既定のブランチ>..HEAD` を打ち、`wip:` があれば BLOCKED。`14. HEAD(` と「違えば push せず、gate を打ち直す(手順 12)」が残る | 満たす。実行は `/test` | `leader.md:244-254`。`<既定のブランチ>` を決める手順 9 は `leader.md:180-186` にある。`TestRenderRolePrompt_Leader_PushRefusesWipCommits` が 5 つの句と、HEAD の比較 → `git log` → `wip:` の判定 → 打ち直し → push の順と、hook 2 本の件名が `wip: ` で始まることを固める |
| AC6: 分離した herdr で (a)〜(d) の実機の確かめ | 満たす(evidence の文書による。再現はしていない) | `docs/evidence/seat-wip-checkpoint-2026-10-22.md`。(a) off の pane で 2 つとも `off`、対照は `RALPH_WIP_CHECKPOINT` 未設定。(d) `/compact` で off はコミットなし・skipped の行あり・index 不変、対照は `wip: checkpoint before context compaction` が 1 つ。(b) C-c と `herdr pane close` で off はコミットなし・skipped の行あり。(c) 対照は `wip: checkpoint before session end` が 1 つ。確かめに使った hook は S1 88706897 のもので、`git diff --stat 88706897 HEAD -- .claude/hooks/ templates/base/.claude/hooks/ .claude/settings.json` は空なので、HEAD の hook と同じ |
| AC7: `git-commit-strategy.md`(と写し)、`/org`・`/director` の skill、`director-launch-cmd.md`(と写し)に 2 つ目の値 | 満たす | `git-commit-strategy.md:30-34`(Org Runtime Commits の段落)と `:44-58`(End-of-Session の 2 項)。`cmp` で写しと一致。`/org` skill の前提(`.claude/skills/org/SKILL.md:24-38`)と手順 6(`:891-897`)、`/director` skill の起動スクリプト(`:223-231`)と herdr の pane(`:280-298`)、recipe の手順 2(`:131-133`)と環境の節(`:249-274`、「these four variables」)、README の launch_cmd の文(258 行)。check-skill-sync と check-sync が写しの一致を通した |
| AC8: 記録(tech-debt の 191・326 行の RESOLVED、326 行の言い直し、試行の evidence の訂正の注、根拠の表と仕様の F11、32 本、`fix/seat-spawn-close` → `#249`) | 満たす。ただし LOW 2 件(V1、V2) | tech-debt 191(コメント)・192(行)、327(コメント)・328(行)が取り消し線と RESOLVED の注をそろえ、効く範囲(新しい binary、`--env` のある herdr、新しい hook)と、効かないとき手順 14 が止めることと、前の手順 9 と 14 では止まらなかったことを書く。328 行の 1 欄目は「a leader's session ends ... after the gate, the push, and the PR」を、止められた座席のセッションという読みに改め、記録がセッションを名指さないことも書く。時刻 05:56:46Z・05:56:52Z・05:57:55Z は `docs/evidence/org-trial-2026-10-20/manifest.jsonl` の `stopped`、05:57:43Z は `inbox.jsonl` の e5 の `escalated` と一致し、e5 の本文は `pull/241` を指す。試行の evidence は F11 の本文を変えず、訂正の段落(816-827 行)を足した。仕様の Status と根拠の表の「段ごとの PR」から抜いた `#` の集合は 32 個で、`cmp` で一致。`gh pr view 249 -R yoshpy-dev/ralph --json state` は `MERGED`。仕様・根拠の表・tech-debt に `fix/seat-spawn-close` は残っていない |
| AC9: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0、Go のテストが go1.25.8 でも通る | 満たす(orchestrator と implementer の結果による) | run-verify は 9d29ee0d で rc 0(914 秒)。a79b0f82 の LOW の直しのあと、implementer が gofmt、vet、`go test ./internal/org/... ./internal/cli/` を go1.26 と go1.25.8 で、check-sync と check-skill-sync を走らせ、いずれも rc 0。この verify では HEAD で run-static-verify(rc 0)、go1.25.8 の build と vet(rc 0)を打った。テストの再実行は `/test` |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` | rc 0(28 秒) | 範囲は changed の要求だが、`.claude/hooks/precompact_checkpoint.sh` が言語に振り分けられないので full に落ちた。shellcheck(hook、verify のスクリプト、`tests/test-*.sh`)、hook 22 本の `sh -n`、guard の awk、settings.json 2 本の `jq -e`、Codex の hook の検査 3 つ、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の計画の参照、golang(gofmt ok、golangci-lint `0 issues.`)、branch の secret scan(`c90d01b4..a79b0f82` clean)。secret scan の範囲の終わりは HEAD と同じ |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./internal/org/... ./internal/cli/...` | rc 0 | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `go vet ./internal/org/... ./internal/cli/...`(go1.26.0) | rc 0 | |
| `gofmt -l internal/ cmd/` | 出力なし | |
| `sh -n` / `bash -n` / `shellcheck tests/test-wip-checkpoint-hooks.sh` | rc 0 | `scripts/verify.local.sh:192` が `tests/test-*.sh` を shellcheck し、`:292` が `/test` で全部を打つので、新しいテストは一覧に足さなくても拾われる |
| `git diff --check c90d01b4...HEAD` | 出力なし | |
| 変えた 41 ファイルの U+FFFD | 0 件 | |
| 計画の digest(`./scripts/plan-visual.sh digest`) | `b98f2806ab31` | `- Approved:` 行と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGMSG_SELF_NAME` を pane や起動スクリプトの環境として書いた文書 | はい | skill 4 写し × 2、recipe 2 写し、README の 1 か所は 2 つ目の値を書く。仕様の FR-2 と FR-9 の印の注(87 行、125 行)は F3 の直し(#248)の記録で、2 つ目の値を書かなくても誤りではない。warning の文を引く文書は先頭の `warning: herdr tab create has no --env flag` だけを引くので、L7 の直し(`; use` → `. Use`)の影響はない |
| `git-commit-strategy.md` の Org Runtime Commits と End-of-Session | はい | 両節とも「ralph が作る herdr の pane すべて(leader を含む座席と headless の director)と launch_cmd の起動スクリプト」で揃う。Org Runtime Commits は無条件に「turns off」と書き、`--env` のない herdr の例外は End-of-Session の節にだけある。「(see there)」で下の節に送っているので食い違いとは見ない(INFO V3) |
| 仕様の印の数 | はい | `[x]` 26 / `[ ]` 6 は base と HEAD で同じ。根拠の表の判定は 31 行で `[x]` 25 / `[ ]` 6、base と同じ(FR-12 は表に入れない)。表の本文の「`[x]` が 25、`[ ]` が 6」とも合う |
| 仕様と根拠の表の PR の一覧 | はい | どちらも 32 個(#241・#242 を含む)で、集合が一致。F11 の直しはブランチ名で書き、32 本に数えない |
| 仕様・根拠の表・tech-debt が引くテスト名 | 1 件ずれ(V1) | 3 つの記録と `git-commit-strategy.md`、新しい evidence の `` `Test...` `` を抜き、`git grep -F 'func <name>('` で探した。base でも見つからない名前(probe や消えたテスト)が tech-debt に 11 個あり、この PR で増えたのは `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` の 1 個だけ |
| 根拠の表の F11 の行の tech-debt の欄 | ずれ(V2) | 下の V2 |
| tech-debt の行の引用 | はい | 新しいコメントが引く 4 つの行("Codex `SessionEnd`/`PreCompact` hooks not wired"、"`ralph org report` writes `docs/reports/org-manifest-<org_id>-<date>.md`"、"The session-end WIP hook commits in a feature org's worktree"、"`session_end_summary.sh` (the `SessionEnd` hook")は、`~~` を除くとそれぞれ 1 行だけに当たる。計画の Non-goals の行(今の 193 行)は開いたまま |
| `.codex/hooks.json` に `SessionEnd`・`PreCompact` がないという記述 | はい | `jq -r '.hooks \| keys[]'` は `PostToolUse`、`PreToolUse`、`SessionStart`、`UserPromptSubmit` |
| 計画の Progress checklist | 遅れ(INFO V4) | self-review の報告があるのに「Review artifact created」が `[ ]`(133 行) |

### Findings

- V1(LOW、記録): `docs/tech-debt/README.md:318`(F3 の RESOLVED のコメント)が、この PR が `TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv` に改名したテストを、前の名前 `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` で引いている。同じ PR は仕様(125 行)と根拠の表の同じ引用を新しい名前に直したので、tech-debt だけが残った。直すなら 318 行の 1 語を新しい名前にする。
- V2(LOW、記録): `docs/evidence/org-spec-evidence-2026-10-21.md:30`(試行の所見の表の F11 の行)の tech-debt の欄が、この PR で取り消し線と RESOLVED を付けた行(`docs/tech-debt/README.md:328`)を、開いた行の形 "The session-end WIP hook commits in a feature org's worktree" のまま引いている。表のほかの直した所見(F3、F4、F8、F9、F10、F12、F14)はどれも `"~~...~~"(RESOLVED の行)` と書き、#249 も merge の前に F4 の欄をその形にしていた。直すなら F11 の欄を `"~~The session-end WIP hook commits in a feature org's worktree~~"(RESOLVED の行)` にする。同じ行の「直す PR」の欄は「座席の環境 4c」の作業名を落としてブランチ名だけになったが、前文(16 行)の決まりに合うので指摘しない。
- V3(INFO): `.claude/rules/ralph/git-commit-strategy.md:30-34` は、`--env` のない herdr では値が付かないことを書かず、「(see there)」で下の節に送る。読み手が段落だけで止まると、どの pane でも止まると読める。直さなくてよい。
- V4(INFO): 計画の Progress checklist の「Review artifact created」が `[ ]` のまま。orchestrator が付ける項目。

## Observational checks

- AC6 の実機の確かめは evidence の文書を読んだだけで、再現していない(herdr 0.7.5 の分離した server と本物の claude が要る)。文書は 4 つの場合を別々に書き、対照の pane では 2 回とも `wip:` のコミットができたので、off の pane でコミットがないのは hook が走らなかったからではないと読める(どちらの pane でも md が書かれている)。
- 試行の F11 の時刻を記録と突き合わせた: implementer の `stopped` 05:56:46Z、reviewer の `stopped` 05:56:52Z、leader の `stopped` 05:57:55Z、e5 の `escalated` 05:57:43Z。コミットの時刻 05:56:51Z は試行の evidence の本文にだけあり、記録ではない。新しい文はどれもそう書いている。

## Coverage gaps

- Go とシェルのテストの実行は `/test` の担当で、この verify では打っていない(a79b0f82 のあとの結果は implementer の報告による)。
- `ralph org spawn` を通した本物の pane で 2 つの `--env` が付くことは Go のテスト(stub の herdr)だけが示す。実機は `herdr tab create --env` を直接打った。
- `--env` のない古い herdr、Codex の座席、機能の org を最後まで通す実機は試していない(記録にもそう書いてある)。
- 手順 14 の `wip:` の確かめは雛形の指示で、ralph の検査ではない。leader が従うかは確かめられない。足せる最小の検査は、`ralph org gate` が `origin/<default>..HEAD` に件名 `wip:` のコミットを見つけたら断ることで、雛形に頼らずに済む(提案。この PR の範囲の外)。

## Verdict

- Verdict: pass(CRITICAL、HIGH、MEDIUM はない。LOW 2 件(V1、V2)はどちらも記録の文のずれで、`/sync-docs` で直せる。INFO 2 件(V3、V4)は任意)
- Verified: AC1〜AC5b のコードとテストの本文の対応(hook の判定と写しの一致、`SeatPaneEnv` の中身と順と本番の 2 か所の組み立て、起動スクリプトの export の位置、warning の全文と帰結、運用規律と手順 14 の文がコードの誤りの文字と一致すること)、AC6 の evidence と hook が S1 から変わっていないこと、AC7 の文書と写し、AC8 の記録(取り消し線と RESOLVED の注、時刻の出どころ、32 本と集合の一致、#249 の `MERGED`、`fix/seat-spawn-close` が残っていないこと、印 26/6 と 25/6)、静的解析(run-static-verify rc 0、go1.25.8 と go1.26 の build と vet、gofmt、shellcheck、`git diff --check`、U+FFFD)
- Partially verified: AC8 の記録の引用のうち、改名したテストの名前(V1)と F11 の行の tech-debt の欄(V2)
- Not verified: Go とシェルのテストの実行(`/test`)、AC6 の実機の再現、`ralph org spawn` を通した本物の pane、古い herdr と Codex の座席
