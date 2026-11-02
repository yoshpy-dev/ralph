# config-path-and-ledger-docs

- Status: Approved
- Approved: 2026-11-02 sha256:917011181315
- Owner: Claude Code
- Date: 2026-11-02
- Related request: 2026-11-02 のユーザーの依頼「残りの穴も直せますか？」のうち、PR #297(#291 の穴)が残した穴(docs/tech-debt/README.md の 421 行)と、#269 待ちで残した PR #296 の穴(396 行の (c)、397 行の残り)
- Related issue: N/A
- Type: fix
- Branch: fix/config-path-and-ledger-docs

## Objective

前の 2 本(PR #296・#297)が残した穴のうち、PR #299 の後に残ったものを直す。

- org の動詞で、壊れた `ralph.toml` のエラーが、読んだファイルを名指さない。main worktree の root から打ったときと、`--config` で名指したファイルのとき。
- `ralph org start --plan` の周りのテストに穴と重複がある。
- leader が `[reviewer]` の値で codex と claude を起こす通しと、既定以外の codex の model は、実物で確かめていない。
- `internal/config/config.go` の 2 つの doc が古い。
- 古い変数 `RALPH_ORG_STATE_DIR` で断る条件を、README と `/org` skill がテストに結ばれない言葉で書いている。

1 本目(PR #290 の系列の残り)は PR #299 で merge した。#269 は PR #298 で merge されたので、#269 待ちだった文書もこの PR で直す(PR #299 の計画の D1)。

## Scope

行番号は main 50f2ffc0 の docs/tech-debt/README.md のもの。

- **421 (a)(b): 壊れた設定のエラーにパスを付ける**
  - `resolveConfig`(`internal/cli/org.go`)が `loadConfigFile` のエラーを `<path>: <err>` で包む(D1)。`<path>` は読んだパスそのもので、`--config` の値か `ralph.toml`(相対のまま)。
  - これで `org: load config: ` で始まる 4 つの文(`newOrgRuntimeAt`、`resolveOrgStartConfig`、`internal/cli/org_director.go:396`、`internal/cli/org_director_headless.go:305`)と、`resolveOrgConfig` を通るほかの呼び出し元の文が、読んだファイルを名指す。接頭辞 `org: load config: ` は変えない。
  - `config.Load`、`ralph config get`、`ralph doctor` の文は変えない(D1)。
  - `TestOrgStartPlan_UnreadableConfigStartsNothing` に、main worktree の root から打つ場合を足す(421 の Trigger のとおり)。
- **421 (c): テストの重複**: `internal/cli/org_start_plan_config_test.go` で main worktree に `ralph.toml` を commit する 2 行を helper にする。
- **421 (d): テストの穴**
  - `newOrgSpawnRuntimeAt` が `newOrgRuntimeAt` のエラーを返す道、`newOrgRuntimeAt` の `org: load config:` の道、`runOrgStartPlan` が `resolveOrgLedger` のエラーを返す道のテスト。`runOrgStartPlan` が `resolveModelOrWarn` のエラーを返す道は、org-feature-worktree の test gaps の行が追うので入れない。
  - `TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml` の対照に `max_total_seats` を足す。`--config` のときと、台帳が main worktree の外のときは、main の `max_total_seats` を使わないことを確かめる。
  - 台帳が main worktree の外のとき、main の `ralph.toml` の `[org.approval].mode` を使わないことを、記録された承認方式で直接確かめる。
- **421 (e): 実物の確かめ**(/test の証跡)
  - codex の model: 一時の repository の `ralph.toml` に、既定(`gpt-6-astra`)と違う、手元の codex が知っている slug を書く。skill と同じ形で codex の reviewer を起こし、codex の session の記録の `turn_context.model` がその slug であることを見る。これは必ず取る(D3)。
  - leader の通し: 一時の repository の `ralph.toml` で `[reviewer]` の 4 つを既定以外にし、`[org.review].plan_advisory = true`(既定は false)にして、`ralph org start --plan` で feature org の leader を起こす。leader の plan advisory は、leader と違う側の CLI を 1 つだけ起こす(`pick_reviewer`)ので、2 つの場合を分けて打つ(plan advisory 1)。
    - claude の leader → codex の reviewer: codex の引数 `-m` と `model_reasoning_effort` が、タスクの `reviewer の codex:` の行の値であること。
    - codex の leader → claude の reviewer: claude の引数 `--model` と `--effort` が、タスクの `reviewer の claude:` の行の値であること。
    - 前提として、先に leader の pane で `which ralph; ralph version` と、leader の worktree での `ralph config get org.review.plan_advisory`(`true`)を確かめる。approval が per-plan なら leader は advisory のあと QUESTION で止まるので、引数を見たら disband する。
    - 環境で止まったら、止まった段と出力を証跡に残す(D3)。打てなかった場合は「確かめていない」として tech-debt の行に残す。前提(pane の新しい ralph、advisory の on)が整わないまま打った結果を、通ったことにも環境の問題にもしない。
- **396 (c): `internal/config/config.go` の doc**
  - `Load` の doc から、「`ralph doctor` がそのまま続けるから `Default()` を返す」の理由句を外す(doctor は PR #296 から、Load のエラーのとき `config.Default()` で続ける)。`Default()` を返すという主張の文は残す。
  - `StateDir` の doc が、台帳の場所を `LocalOrgStateDir` で読むと書いているのを、`ResolveOrgStateDir` が `LocalOrgStateDirAt` で読む形にする。
- **397 の残り: 断る条件の言葉を 1 つにする**(D2)
  - README の `[org].state_dir` の段落(148 行)の条件の文を、定数 `retiredOrgStateDirRefusalCondition`(`internal/cli/org_legacy_ledger.go`)の言葉にする(今の「the variable names」を `RALPH_ORG_STATE_DIR` names に)。
  - `/org` skill の 4 つの写しの、台帳の場所の項の日本語の文は残し、条件の文の後に、定数の文を英語のまま code span で足す。
  - README と skill の写しが定数の文を含むことを確かめるテストを `internal/cli` に足す。README は backtick を外して比べ、skill は code span の中を比べる。読むのは `README.md`、`.claude/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`(`.agents` の写しは `scripts/check-skill-sync.sh` が、templates の写しどうしは `scripts/check-sync.sh` が一致を見る)。
- **tech-debt の表**: 396・397・421 行を、直した形に書き換える(/sync-docs)。

## Non-goals

- `config.Load` のエラーの形を変えること(D1)。`ralph config get` と `ralph doctor` の文はそのまま。
- エラーの文のパスを絶対パスにすること。読んだパスをそのまま出す(相対の `ralph.toml` を含む)。
- leader が PATH の `ralph` を使うこと(tech-debt 192 行)。通しでは pane の PATH を整えるだけで、コードは変えない。
- `runOrgStartPlan` が `resolveModelOrWarn` のエラーを返す道のテスト(別の行が追う)。
- PR #299 が残した穴(tech-debt 386・388・390・395・420 行の残り)。
- 古い変数を `retiredEnvs` から外すこと。

## Assumptions

- `org: load config: ` の接頭辞を保てば、`org_wait_director_test.go:274` の `HasPrefix` は通る。変わる固定文は `org_start_plan_config_test.go` の 139〜160 行あたり(`toml: ` で始まることを見る所)。実装の前に、`org: load config` と `toml: ` を固定するテストを grep で数え直す。
- `ReadFile` のエラー(`open ralph.toml: permission denied`)はすでにパスを含むので、包むとパスが 2 回出る。`ralph config get` も今日同じ形で出しているので受け入れる(D1)。
- leader の plan advisory の switch は `command ralph config get org.review.plan_advisory`(`internal/org/prompts/leader.md`)で読む。PATH の ralph 6.0.0 には `config get` がないので、pane の PATH が古い ralph を指すと advisory は飛ぶ。
- セッション C(#277、feat/init-config-questions)は README の 124 行あたりを変えている。この計画が変える 148 行とは重ならない。

## Affected areas

- `internal/cli/`: `org.go`(`resolveConfig`)、`org_start_plan_config_test.go`、テストの新しいファイル(条件の文のテスト、必要なら 421 (d) のテスト)
- `internal/config/config.go`(doc だけ)
- `README.md`(148 行の 1 文)
- `/org` skill の 4 つの写し(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`)
- `docs/tech-debt/README.md`(396・397・421 行)
- `docs/evidence/`(/test の実物の確かめ)

## Visual review

- Page: `.harness/state/plan-visual/config-path-and-ledger-docs.html`(かんたんな図 2 つ、詳しい図 3 つ)
- Lint: `./scripts/plan-visual.sh lint` は exit 0(最初は overview の補足の文に設定ファイルの名前があって L3 で落ちたので、言葉に直した)。
- Self-check: 全体、overview、easy-condition、detail-overview、detail-error、detail-live を `shot` で撮って読んだ。どれもきれい。

## Design decisions

- **D1: パスは `resolveConfig` で付け、`config.Load` は変えない。** `Load` の doc の規則(個人用のファイルの誤りと、それを重ねたあとの検証エラーだけ、どちらのファイルか分からないときにパスを名指す)では、呼び出し元が渡したパスは繰り返さない。`ralph config get` はその規則どおり自分でパスを付けている。`Load` で付けると、config get、doctor、`resolveOrgStartConfig` の main の文の 3 か所でパスが二重になり、3 か所のテストが連れて変わる。`resolveConfig` なら 1 か所で、`org: load config: ` の 4 つの文に同時に効く。tech-debt 421 の Trigger も「resolveConfig (or Load)」と書く。consult-plan-leftovers は前に Load を推したが、この根拠で撤回した。
- **D2: 397 は README と skill を定数の言葉に結び、テストで確かめる。** 別案は「README と skill は今も同じ条件を正しく書いているので、食い違いを認めて行を閉じる」(行の Trigger が許す)。README はほぼ定数と同じ文で、直すのは 1 語。skill は日本語の文を残して code span を足すだけなので安く、5 か所が 1 つの定数に結ばれる。
- **D3: 実物の確かめは /test で 1 回ずつ。codex の model の確かめは必ず取り、leader の通しは止まったら記録する。** codex の model は org なしで確かめられる。leader の通しは herdr・agmsg・見張りと、pane の PATH の新しい ralph が要り、環境で止まりうる。止まった段と出力を証跡にして、PR の Known gaps に書く。leader の advisory は違う側の CLI を 1 つだけ起こすので、claude の leader と codex の leader の 2 つを打ち、advisory を on にすることを前提に含める(plan advisory 1)。
- Critical forks: D1〜D3 は consult-plan-leftovers に聞いたうえで選んだ(ユーザーの指示「計画の分かれ目は consult に聞いて推奨案を自分で選ぶ」)。

## Acceptance criteria

- [ ] AC1(421 (a)(b)): main worktree の root で、壊れた `ralph.toml` のまま `ralph org start --plan` を打つと、エラーは `org: load config: ralph.toml: ` で始まり、何も作らない。`--config <file>` の壊れたファイルでは `org: load config: <file>: ` で始まる。spawn と `start <task>` と `director start` の同じ道も同じ形。`TestOrgStartPlan_UnreadableConfigStartsNothing` に root のケースがあり、元のコードでは落ちる。`config.Load`、`ralph config get`、`ralph doctor` の文は変わらない。
- [ ] AC2(421 (c)): main worktree に `ralph.toml` を commit する手順は helper 1 つで、テストの中に同じ 2 行が繰り返されない。
- [ ] AC3(421 (d)): 3 つのエラーの道を打つテストがある。上限のテストに `max_total_seats` の対照があり、`--config` のときと台帳が外のときは main の値を使わない。台帳が外のとき main の `[org.approval].mode` を使わないことを、記録された承認方式で確かめるテストがある。
- [ ] AC4(396 (c)): `Load` の doc に doctor の理由句がなく、`StateDir` の doc は `LocalOrgStateDirAt` を名指す。
- [ ] AC5(397): README の段落と `/org` skill の 4 つの写しが、定数 `retiredOrgStateDirRefusalCondition` の文を含む(README は backtick を外して、skill は code span で)。それを確かめるテストがあり、README か skill の文を変えると落ちる。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。
- [ ] AC6(421 (e)): 証跡に、既定と違う codex の slug で起こした reviewer の session の `turn_context.model` がその slug であることがある。leader の通しは、2 つの場合(claude の leader → codex、codex の leader → claude)それぞれについて、前提(pane の `ralph version` が新しい binary、leader の worktree で `org.review.plan_advisory` が `true`)と、起こした reviewer の引数がタスクの行の値と一致したことか、止まった段とその出力がある。
- [ ] AC7: tech-debt の 396・397・421 行が、直した形に書き換わっている。AC6 で打てなかった場合や止まった場合は、その場合を「確かめていない」として 421 行に残す。`./scripts/run-verify.sh` と `./scripts/run-test.sh` が通る。

## Implementation outline

各スライスの verify の `<go test>` は `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test` を指す。`internal/cli` のテストは全部で 10 分を超えるので、`-run` で絞る。全体は /verify と /test で打つ。

1. S1(AC1): `resolveConfig` でパスを付ける。root のケースと `--config` のケース、変わる固定文 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'OrgStartPlan|OrgSpawn|OrgStart|OrgDirector|OrgWait|LoadConfig|ResolveConfig' ./internal/cli/`
2. S2(AC2、AC3): テストの helper と穴(テストだけ) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'OrgStartPlan|OrgSpawn|NewOrgRuntime' ./internal/cli/`
3. S3(AC4): `config.go` の doc — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && <go test> ./internal/config/`
4. S4(AC5): README、skill の 4 つの写し、条件の文のテスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'RefusalCondition|RetiredStateDirEnv|RetiredEnv' ./internal/cli/ && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`

tech-debt の表の書き換え(AC7)は /sync-docs で行う。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh`。
- Spec compliance criteria to confirm: AC1〜AC7。AC1 は `org: load config` の文をすべて数え、どれも読んだパスを名指すことを見る。
- Documentation drift to check: README の `[org].state_dir` の段落と org の動詞のエラーの説明、`/org` skill の 4 つの写し、docs/tech-debt/README.md の 396・397・421 行。
- Evidence to capture: docs/reports/verify-2026-11-02-config-path-and-ledger-docs.md。

## Test plan

- Unit tests: 各スライスで足すテスト(上の AC)。
- Integration tests: `internal/cli` の org のテスト(`OrgStartPlan`、`OrgSpawn`、`OrgDirector`、`RetiredStateDirEnv`)を通しで打つ。
- Regression tests: `./scripts/run-test.sh` 全体。既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` と 30 分の timeout)で落ちたら、1 回だけ打ち直して記録する。
- Edge cases: main の root と subdirectory と linked worktree、`--config` の壊れたファイル、読めないファイル(権限)、個人用のファイルの誤り(パスが 2 つ出る形)。
- Evidence to capture: docs/reports/test-2026-11-02-config-path-and-ledger-docs.md。実物の確かめは docs/evidence/ に残す(D3)。作ったディレクトリ、herdr の server、見張り、org は終わったら片付ける。

## Risks and mitigations

- **パスの二重**: `ReadFile` のエラーと個人用のファイルの誤りで、パスが 2 つ並ぶ。`ralph config get` と同じ形なので受け入れ、テストの 1 つで形を固定する。
- **固定文の取りこぼし**: S1 の前に `org: load config` と `toml: ` を固定するテストを数え直す。
- **leader の通しが重い・止まる**: D3 のとおり、止まった段を記録して進む。codex の model の確かめは org なしで取る。HOME の下に作るものは一意の名前にし、終わったら消す。main の `.codex/config.toml` は前後で変わらないことを見る。
- **他のセッションとの衝突**: README はセッション C も変えている(124 行あたり)。skill の 4 つの写しと tech-debt の表は衝突しやすいファイル。衝突したら両方の変更を残して直す。tech-debt の表は中ほどの行を書き換えるだけにし、末尾には足さない。

## Rollout or rollback notes

- org の動詞が設定ファイルを読めないときのエラーの文が、読んだファイルの名前を含むようになる。終了コードと何も作らない振る舞いは変えない。
- 戻すときは PR を revert する。設定ファイルや台帳の形は変えない。

## Open questions

- なし(分かれ目は consult-plan-leftovers で決めた)。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh。`ralph_setting` は main から build した ralph で読んだ)。1 回目は codex が「Selected model is at capacity」で rc 1 になり、`-o` のファイルがなかった(incomplete)。一時的なエラーなので 1 分おいて 1 回だけ打ち直し、2 回目は rc 0 で完了した。git の状態は前後で同じ。所見 1 件は、ユーザーの指示「HIGH と MEDIUM は聞かずに計画に反映する」に従って反映した。
  - [MEDIUM] 1: AC6 の leader の通しは、advisory が leader と違う側の CLI を 1 つだけ起こすこと(`pick_reviewer`)と、`[org.review].plan_advisory` が既定で off であることを考えておらず、advisory が動かないか片方しか確かめられない。止まった段の記録でも AC6 を満たせるので、試験の設定の不足を環境の問題として扱う恐れがある。→ Scope の 421 (e)、D3、AC6、AC7 に、advisory を on にする前提と leader の worktree での確かめ、claude の leader → codex と codex の leader → claude の 2 つの場合、打てなかった場合を「確かめていない」として tech-debt に残す条件を書いた。図の detail-live も直し、lint と撮り直しで確かめた。
- [x] Plan approved — 自動承認(ユーザーの指示)。2026-11-02、digest 917011181315。分かれ目(D1〜D3)は consult-plan-leftovers の判定(D1 は前の Load の推しを撤回して resolveConfig、D2 は README と skill を定数に結ぶ、D3 は codex の model を必ず取り leader の通しは先に pane の ralph を確かめる)に沿って選んだ。
- [x] Branch created — fix/config-path-and-ledger-docs(base 50f2ffc0)
- [x] Implementation started
  - S1(AC1): 40cedc9e。`resolveConfig` がエラーを `<path>: <err>` で包む。`org: load config: ` の 4 か所はそのまま。`TestOrgStartPlan_UnreadableConfigStartsNothing` に root のケースを足し、`--config` のケースは `org: load config: <path>: ` を見る。新しいテスト `TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad`(spawn・start <task>・director start × `./ralph.toml` と `--config`)。古いコードの写しで root のケースと新しいテストが落ちた。固定文を変えたのは `org_start_plan_config_test.go` だけで、`HasPrefix`・`Contains`・`errors.Is` の 5 つのテストはそのまま通った。
  - S1 の補い(手で直した 2 ファイル): a6be3f5e。implementer が範囲の外で見つけた `ralph status` の warning(`internal/cli/status.go` の `statusOrgConfig`)は自分で `ralph.toml` を名指していたので、S1 のあと `load ralph.toml: ralph.toml: toml: …` と二重になった。計画の Assumptions が見落としていた呼び出し元。warning は名前をエラーに任せる形にし、出力は S1 の前と同じに戻った。`status_test.go`(1611 行)は既存の条件に「`ralph.toml: ralph.toml` を含まない」を足しただけで、`status.go` を戻すと落ちる(8 件)ことを確かめた。機械的な確かめ: どれも clean。
  - S2(AC2、AC3): f185f22d。テストだけ。helper `commitRalphTomls`。3 つのエラーの道のうち 2 つは S1 のテストがすでに通るので重ねず、`resolveOrgLedger` の道は `TestOrgStartPlan_RefusedLedgerStartsNothing`(git の外で `RALPH_ORG_STATE_DIR` を別の場所に)。上限のテストに `max_total_seats` の対照 2 つ、`TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead`(4 ケース、binding の `Approval` を直接読む)。5 つの変異はどれも新しいテストで落ちた。`org_start_plan_config_test.go` は 547 行。範囲の外で見つけたもの: 同じ add と commit の 2 行がほかのテストファイル 5 つにある(AC2 の対象はこのファイルだけ)。
  - S3(AC4、手で直した 1 ファイル): 3311cf16。`Load` の doc の理由句を外し、`StateDir` の doc は `org.ResolveOrgStateDir` が `LocalOrgStateDirAt` で読むと書いた。`go test ./internal/config/` ok。機械的な確かめ: どれも clean。
  - S4(AC5): d257824e。README の 1 語(「the variable names」→ `RALPH_ORG_STATE_DIR` names)、skill の 4 写しに定数の code span、`TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs`(README は backtick を外し、`.claude` と `templates/base/.claude` の skill は code span を見る。3 つの変異で落ちた)。templates の `.agents` は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で作った。implementer は sonnet。
  - S4 の補い(手で直した skill の 4 写し): b16f4bd0。implementer の指摘で、skill の「条件は Go の定数と同じ」は scaffold した project では指す先がない(Go の定数はない)ので、「`ralph doctor` の warn と `ralph org escalate --help` は、この条件を英語でこう書く」にした。code span は変えていないのでテストはそのまま通る。check-skill-sync と check-sync は通った。
  - 実装の終わり: S1〜S4 と補いのコミットはすべて push 済み(40cedc9e … b16f4bd0)。各スライスの verify はすべて rc 0。
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
