# sync-docs report: seat-agmsg-self-name

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-agmsg-self-name.md`(承認の digest `dc40c452f092` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。計画には触っていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `1e47c309`、branch は `fix/seat-agmsg-self-name`。run 開始時の HEAD は `c7d4d85b`(この report を含む commit の SHA ではない)。文書の直しは `ab5c6c96`
- 先行 report: `docs/reports/self-review-2026-10-22-seat-agmsg-self-name.md`(Merge 可、LOW 4。L1・L2・L4 は 4991744e で直った)、`docs/reports/verify-2026-10-22-seat-agmsg-self-name.md`(pass。LOW V1・V3、INFO V2・V4・V5)、`docs/reports/test-2026-10-22-seat-agmsg-self-name.md`(pass。G1・G2・G5 と V4 は 9c88b759 で直った。最後の節に警告の文が変わったことがある)
- この run が変えたのは、14 ファイル(下の「Changes made」。写しを含む)、この report、insight event。コード、先行 3 report、計画、evidence の文書(`docs/evidence/org-spec-evidence-2026-10-21.md` の 1 文を除く)には触っていない

## Summary

`1e47c309...c7d4d85b` の差分は、ralph が打つ agmsg のすべての呼び出しに `AGMSG_SELF_NAME=off` を足し(`AgmsgSelfNameEnv`、`AgmsgSelfNameOff`)、座席と headless の director の herdr の pane を `herdr tab create --env AGMSG_SELF_NAME=off` で作り(help に `--env` があるときだけ。ないときは `--env` なしで作って stderr に警告を 1 行出す)、`launch_cmd` の起動スクリプトに同じ export を足す。コードを読んで、この 3 つが文書の記述と合うかを確かめた。

直したのは、verify が挙げたずれ V1(起動スクリプトの export の説明と「these two variables」)、V3(警告を書く文書がない)、V2(herdr の pane の環境の文)と、test report の最後の節が指す警告の文の変更である。警告は、文書では先頭の `warning: herdr tab create has no --env flag` だけを引き、全文は写していない。

## Changes made

| File(写しを含む) | Change |
|------|--------|
| `docs/recipes/director-launch-cmd.md` と `templates/base/docs/recipes/director-launch-cmd.md`(`cmp` で同一) | 手順 2 に `AGMSG_SELF_NAME=off` を足した(スクリプトの順: token、agmsg home(あれば)、`AGMSG_SELF_NAME`)。Environment の節に `AGMSG_SELF_NAME` の項を足し、agmsg 1.5.1 が herdr の agent と pane の名前を変えること、agmsg home の有無によらず export すること、herdr の director の pane には `herdr tab create --env` で付くこと(0.7.5 は持つ。古い herdr では付かず、`warning: herdr tab create has no --env flag` で始まる行が stderr に出ること)を書いた。`RALPH_ORG_AGMSG_HOME` の項の「herdr の pane の環境は herdr server のもの」に「`AGMSG_SELF_NAME` を除く」の旨を足し、最後の文の「these two variables」を「these three variables」にした(V1、V2) |
| `.claude/skills/director/SKILL.md` と 3 つの写し(`.agents/skills/director/`、`templates/base/.claude/skills/director/`、`templates/base/.agents/skills/director/`。4 つとも同一) | 起動スクリプトの説明に `AGMSG_SELF_NAME=off` の export を足した(agmsg home の有無によらない)。herdr の pane の環境の項に「ralph が足す `AGMSG_SELF_NAME=off` だけが加わる」を足し、herdr で立てる director の pane を `--env AGMSG_SELF_NAME=off` で作ること、help に `--env` がなければ付けずに警告が 1 行出て、その herdr では director が自分で打つ agmsg が pane の名前を変えうることの項を足した(V1、V2、V3) |
| `.claude/skills/org/SKILL.md` と 3 つの写し(4 つとも同一) | 「前提」の herdr / agmsg の項に、座席の pane を `herdr tab create --env AGMSG_SELF_NAME=off` で作ること(0.7.5 の `--env`)、`--env` のない herdr でも動くが警告が stderr に 1 行出て座席の agmsg が名前を変えると ralph が座席に届かなくなりうること、ralph 自身が打つ agmsg には herdr の版によらず off が付くことを 1 か所に書いた(V3) |
| `README.md`(258 行) | 「`RALPH_ORG_AGMSG_HOME` reaches this director through the launch script」の文に、起動スクリプトが `AGMSG_SELF_NAME=off` も export することを足した |
| `docs/tech-debt/README.md`(317 行、F3 の RESOLVED コメントだけ。変わった行はこの 1 行で、行数は 337 のまま) | 新しい 3 つのテスト(`TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`)を、同じファイルの既存の名前の並びに足した。コメントの commit の欄に `9c88b759`(警告のテスト)を足した。「`--env` のない herdr では、座席が自分で打つ agmsg が座席の agent の名前を変えうる」を、headless の director も含む言い方(「a seat or a headless director runs itself」「the herdr agent of its pane」)にした |
| `docs/specs/2026-10-07-org-multi-org-director.md`(122 行、FR-9 の印の注)と `docs/evidence/org-spec-evidence-2026-10-21.md`(55 行、FR-9 の根拠の行) | どちらも警告のテストを 1 つ挙げていたので、`internal/org/driver/herdr_test.go` の `TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce` と `TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr`、`internal/cli/org_runtime_env_test.go` の `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` に直した(ファイル名を隣に書いた)。`[x]` と `[ ]` の印、判定の数は変えていない |
| `docs/reports/sync-docs-2026-10-22-seat-agmsg-self-name.md` | 新規 |
| `docs/insights/events/2026-10-22-seat-agmsg-self-name.jsonl` | sync_docs の event を 1 行足した |

足した文に `file:line` の引きはなく、テストは名前で引いた。3 つのテスト名はどれも `func <名前>(` が隣に書いたファイルにある(`herdr_test.go` の 342 行と 428 行、`org_runtime_env_test.go` の 129 行)。足した行に U+FFFD、ホームのパス、pane の id、token はない。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `internal/org/director_cmd.go` の起動スクリプト(`directorLaunchScript.text`) | 文書と合う | `RALPH_ORG_AGMSG_HOME` の export は `AgmsgHome` があるときだけ、`AGMSG_SELF_NAME='off'` の export は常に、check-in の行の前。文書の「agmsg home の有無によらない」と合う |
| `internal/org/driver/herdr.go` の `TabCreate` と `paneEnvSupported` | 文書と合う | `--env` は help に `--env` があるときだけ付く。help の失敗と時間切れも付けない側に倒れる。警告は `Warn`(nil なら `os.Stderr`)に 1 行。形は 2 つあり、help に `--env` がないときは `warning: herdr tab create has no --env flag, ...`、help を打てなかったときは `warning: could not check whether herdr tab create has --env (...), ...`。文書は前者の先頭だけを引き、後者は書いていない(下の Decisions) |
| `internal/org/driver/agmsg.go` の `Agmsg.run` | 文書と合う | 7 つの呼び出しが `EnvRunner` を通って off を足す。`/org` skill の「ralph 自身が打つ agmsg には、herdr の版によらず off が付く」と合う |
| `.claude/skills/org/SKILL.md` の「leader の pane は herdr サーバーの環境で動き、…ralph は start に渡した `--state-dir` も `RALPH_ORG_STATE_DIR` も pane に渡さない」 | 変更なし | この文は台帳の場所の話で、今も正しい。ralph が pane に足すのは `AGMSG_SELF_NAME` だけで、state dir は足さない |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | pane を作る flag や agmsg の環境を書いた文はない。この仕様は「本文の履歴の記述は書き換えない」とし、後継の仕様(2026-10-07)が現在の記述を持つ。後継の FR-9 の注が `--env` と警告を書いている。Dependencies の herdr の行は版を書かず、この PR の Non-goals も herdr の最低の版を決めないので、足さなかった |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | message の型と形の規約だけで、環境を書かない |
| README | 258 行だけ | herdr の要件や pane の作り方を並べた文はなく(herdr の語は 124、248、258 行ほか)、警告を書く場所がない |
| `AGENTS.md` の repo map | 変更なし | `internal/org/` の行の「herdr/agmsg driver adapters」は今も合う。AGENTS.md は短く保つので、`EnvRunner` や `PaneEnv` は足さなかった |
| `docs/architecture/repo-map.md`、`docs/quality/`(`quality-gates.md` を含む)、`docs/recipes/director-wake.md`、`docs/recipes/codex-seat-permissions.md`、`internal/org/prompts/director.md` | 変更なし | agmsg の環境や pane の作り方を書いた文はない。`director.md` の `RALPH_DIRECTOR_TOKEN` の行は token の話だけ |
| tech-debt の F4 の行と、「leader への ALERT」の行 | 変更なし | F4 の行の「the agmsg self-rename of the row above」は、計画の Progress が 4b に渡した所。この run の範囲の外(依頼の「他の行に触らない」) |
| `CLAUDE_ENV_FILE` の回り道(D-F3)を手順として書く文書 | なし | verify の確かめと同じ。記録の外で出るのは仕様の FR-9 の注と tech-debt の 2 行だけで、どれも手順ではない |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の V1(LOW)起動スクリプトの export の文書 | 直した(recipe の手順 2 と Environment の節、`/director` skill、README) |
| verify の V2(INFO)herdr の pane の環境の文 | 直した(recipe の `RALPH_ORG_AGMSG_HOME` の項、`/director` skill の同じ項) |
| verify の V3(LOW)警告を書く文書がない | 直した(`/org` skill の「前提」、`/director` skill、recipe の Environment の節。各文書で 1 か所)。先頭の `warning: herdr tab create has no --env flag` だけを引いた。help を打てなかったときの警告(`warning: could not check whether herdr tab create has --env`)は書かなかった。利用者が出会うのは前者がほとんどで、後者の原因は herdr の呼び出しの失敗や時間切れなので、警告の文そのものが原因を書くため |
| verify の V4(INFO)警告の接頭辞と出口 | 9c88b759 で直っている(`warning: ` で始め、出口は `Warn`、nil なら `os.Stderr`)。文書は「stderr に 1 行」と書き、`cmd.ErrOrStderr()` とは書いていない |
| verify の V5(INFO)計画の Progress の「Review artifact created」が `[ ]`、Assumptions の行番号 | 計画は `[x]` に直っている。Assumptions の行番号(787 → 788)は、計画を書いた時点の値で、計画は digest の範囲なので直さない。Progress に sync-docs の項はまだない(下の Left open) |
| test の最後の節(警告の文が 9c88b759 で変わった) | 文書は先頭の `warning: herdr tab create has no --env flag` を引くので、新しい文とも合う。全文は写していない |
| test の G3、G4(INFO) | 文書の話ではない。tech-debt の行は立てていない(test report が直さないと決めている) |

## Plan drift notes

- 計画の Progress の「sync-docs に渡すもの」の 4 項目(recipe の 131 行と 248 行、`/director` skill、`AgmsgSelfNameOff` の doc コメントに起動スクリプト、未確認のこと)のうち、recipe と skill はこの run で直した。doc コメントは 4991744e で済んでいる(`agmsg.go` の `AgmsgSelfNameOff`)。未確認のこと(0.7.5 より古い herdr が `tab create --help` を無視すると tab を 1 つ作るか)は今も未確認で、文書は「`--env` を挙げない herdr」の動きだけを書き、help を無視する herdr のことは書いていない
- 計画の AC と Scope は、この run の変更と食い違わない

## Left open

- 計画の Progress に sync-docs の項がない。計画は依頼どおり触らなかった。`/pr` か orchestrator が足す(digest の範囲の外)
- 0.7.5 より古い herdr が `tab create --help` を無視して tab を作るかどうか: 未確認(計画、verify、test が同じことを書いている)。文書には書いていない
- 本物の herdr と agmsg での `ralph org spawn` や headless の director の実機の確かめ: 文書の記述は、コードとテストと evidence の (e) に基づく。新しい binary での実機はない
- release のノートに書くこと(計画の Rollout): ralph が打つ agmsg は名前を書き換えない、`--env` のある herdr では座席の pane にも付く、ない herdr では警告が出る。PR の本文の仕事

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/check-sync.sh`(編集と commit のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。`./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つの写しは `cp` した。`/org` と `/director` の 4 つの写しは、それぞれ `md5` が 1 種類 |
| `cmp docs/recipes/director-launch-cmd.md templates/base/docs/recipes/director-launch-cmd.md` | 同一 |
| `git diff --check` | rc 0 |
| `GOFLAGS=-timeout=30m go test -count=1 -timeout 30m ./internal/scaffold/... ./internal/config/...` | rc 0(`internal/scaffold` ok、`internal/config` ok) |
| `go test -count=1 ./internal/org -run <director_skill_sync_test.go、report_names_sync_test.go、send_defaults_sync_test.go の全テスト>` | ok。`/director` skill の節の同期テストは、足した文に反応しない |
| `./scripts/plan-visual.sh digest`(計画) | `dc40c452f092`(承認の digest と同じ) |
| 足した行の走査(ホームのパス、`hiroki`、U+FFFD)と tech-debt の 317 行の構造 | 漏れなし。317 行は `<!-- RESOLVED` で始まり `-->` で終わり、バッククォートは偶数。変わった行は 317 だけ(`c7d4d85b` と比べた) |

`run-verify.sh`、`run-static-verify.sh`、`run-test.sh` は走らせていない(依頼どおり)。コードは変えていない。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 新しい binary を本物の herdr と agmsg で動かしたときの警告の見え方、0.7.5 より古い herdr の挙動、Go のテストの全体(`/test` が済ませた)

## Insight event

`./scripts/insights-append.sh --slug seat-agmsg-self-name --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-22-seat-agmsg-self-name.jsonl` に 1 行足した(cycle 1)。この run の append も UTC の 2026-10-22 に走ったので、verify と test の event と同じファイルに入った(self-review の event だけが `2026-10-21-seat-agmsg-self-name.jsonl` にある。verify の報告が書いた日付またぎの食い違い)。
