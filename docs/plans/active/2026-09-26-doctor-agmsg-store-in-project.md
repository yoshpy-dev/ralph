# doctor-agmsg-store-in-project

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-26
- Related request: #164(`ralph doctor` の「Codex sandbox (agmsg writable root)」Check、PR #171)の cross-review cycle 2 で残った WORTH_CONSIDERING 1 件(`docs/reports/cross-review-triage-codex-agmsg-writable-root.md` の WC-3)。agmsg の保存先(`<agmsg home>/db` または `AGMSG_STORAGE_PATH`)が座席の作業ディレクトリの中にあると、codex の workspace-write は追加の writable root なしで書けるのに、Check は warn を出して不要な writable root を勧める(誤 warn。誤 pass ではない)。Codex は codex-cli 0.154.0 で、warn の状態のままプロジェクト内の保存先への sandbox 下の SQLite 書き込みが成功することを再現した
- Related issue: 170
- Type: fix
- Branch: fix/doctor-agmsg-store-in-project

## Objective

agmsg の保存先がプロジェクトディレクトリ(doctor の対象ディレクトリ)の中にあり、codex の保護ディレクトリ(`.git` / `.agents` / `.codex`)をまたがないとき、warn の Detail を、「送れない」と「writable root を足す」がその保存先を作業ディレクトリに含まない座席(task worktree など)についての話だと分かる文にする(作業ディレクトリが保存先を含む座席はすでに書けると書く)。status は warn のまま。

## Scope

1. `checkCodexAgmsgWritableRoot` がプロジェクトディレクトリを受け取る(`runDoctorFull` の `targetDir` を絶対パスにして渡す)。テストでは任意のディレクトリを渡せる
2. warn の場合だけ、保存先がプロジェクトディレクトリの配下にあり保護ディレクトリをまたがないかを判定し(既存の `codexRootCoverage` と同じく、symlink を解決した組み合わせも見る)、該当すれば warn の文そのものを条件付きにする: 「RESULT を送れない」の断定と「writable root を足す」の指示を、作業ディレクトリが保存先を含まない座席に限定し、含む座席はすでに書けると書く。config がある場合とない場合の両方の文に適用する。末尾に補足を足すだけにはしない(同じ診断の中で案内が矛盾するため)。pass と info の Detail は変えない
3. 文書: `docs/recipes/codex-seat-permissions.md`(2 コピー)と `/org` skill(4 面)に同じ補足を 1 文ずつ
4. `docs/tech-debt/README.md` の `checkCodexAgmsgWritableRoot` の行に「座席の cwd を知らない(補足で案内するだけ、status は warn のまま)」を反映

## Non-goals

- 座席の実際の `--cwd` を知ること(`ralph org spawn` が座席ごとに決める。doctor からは見えない)
- プロジェクトディレクトリを writable root とみなして pass にすること(task worktree で動く座席では誤 pass になる)
- status の変更、codex の設定の読み方の変更、ほかの Check

## Assumptions

- `runDoctorFull` の `targetDir` はプロジェクト root(`ralph.toml` のあるディレクトリ)。相対パスのことがあるので `filepath.Abs` で絶対パスにする(失敗したら補足を出さない)
- 保護ディレクトリの判定は既存の `pathCrossesCodexProtectedDir` / `codexRootCoverage` をそのまま使う(新しい規則を作らない)

## Affected areas

- `internal/cli/doctor_codex_writable_root.go`(引数、warn の Detail)
- `internal/cli/doctor.go`(呼び出し)
- `internal/cli/doctor_codex_writable_root_test.go`、`internal/cli/doctor_codex_writable_root_unix_test.go`(呼び出しの更新と新しいテスト)
- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`
- `.claude/skills/org/SKILL.md` と 3 つのミラー
- `docs/tech-debt/README.md`

## Design decisions

- status は warn のまま、Detail の補足だけにする(issue の指定。座席の cwd が分からないため)
- 判定は既存の coverage の関数を再利用し、symlink の解決済みの組み合わせも含めて保護ディレクトリをまたがないことを確かめる
- Critical forks: None(issue が方式を指定している)
- **Codex plan advisory(2026-09-27、MEDIUM 1、ユーザー決定: 対応案で plan を更新)**: 補足を末尾に足すだけでは、既存の warn の「RESULT を送れない」の断定と無条件の「root を足す」の指示と矛盾し、不要な設定をなお誘発する → プロジェクト内の保存先では warn の文そのものを条件付きにする(Scope 2、AC-1)

## Acceptance criteria

- [ ] AC-1: 保存先がプロジェクトディレクトリの配下にあり保護ディレクトリをまたがない warn では、Detail が「作業ディレクトリがこの保存先を含む座席はすでに書ける」と述べ、「RESULT を送れない」と「writable root を足す」は作業ディレクトリが保存先を含まない座席(task worktree など)についてだけ述べる。無条件の「送れない」の断定と無条件の追加の指示は残らない。config がある場合とない場合の両方で確かめる。status は warn
- [ ] AC-2: 保存先がプロジェクトの `.agents`(または `.git` / `.codex`)の配下にある warn では、補足が入らない
- [ ] AC-3: 保存先がプロジェクトの外にある warn では、Detail が従来と同じ(既存のテストがそのまま pass)
- [ ] AC-4: pass(明示の root、暗黙の root)と info の Detail は変わらない
- [ ] AC-5: プロジェクトディレクトリが symlink を含むパスで渡されても(macOS の `/var` と `/private/var`)、AC-1 / AC-2 の判定が同じになる
- [ ] AC-6: `AGMSG_STORAGE_PATH` でプロジェクト内に置いた保存先でも AC-1 が成り立つ
- [ ] AC-7: recipe(2 コピー)と `/org` skill(4 面)に同じ補足があり、`check-skill-sync.sh` / `check-sync.sh` が pass。tech-debt の行が更新されている
- [ ] AC-8: `./scripts/run-verify.sh` green、`TMPDIR=/tmp go test ./internal/cli/... -count=1` ok、push 前の `./scripts/secret-scan-branch.sh --strict` が clean

## Implementation outline

1. Slice A: 引数の追加、warn の Detail の補足、呼び出しの更新、テスト(AC-1〜AC-6)
2. Slice B: recipe と skill、tech-debt(AC-7)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet)
- Spec compliance criteria to confirm: AC-1〜AC-8、Non-goals(status は warn のまま、pass にしない)
- Documentation drift to check: recipe、`/org` skill、tech-debt、doctor の Check の一覧を書いた文書
- Evidence to capture: ビルドした `ralph` で、偽の HOME とプロジェクト内の保存先(`AGMSG_STORAGE_PATH`)の fixture に対し、Check の行に補足が出ること

## Test plan

- Unit tests: AC-1〜AC-6 の各ケース(hermetic な `t.TempDir()`、seam の env)
- Regression tests: 既存の writable root のテスト一式(引数の追加による呼び出しの更新のみ)
- Edge cases: 保存先がプロジェクトディレクトリそのもの、プロジェクトディレクトリが空文字または解決できない、保存先が存在しない(nearest existing の解決)
- Evidence to capture: red/green(補足の条件を外すと AC-1 または AC-2 が落ちる)

## Risks and mitigations

- 呼び出しの更新が多い(テストで 53 か所)。機械的な変更なので、テストの helper でまとめてもよい
- 補足の文が長くなり Detail が読みにくくなる。1 文に収める

## Rollout or rollback notes

- doctor の 1 つの Check の Detail だけの変更。revert すれば従来の文言に戻る

## Open questions

- なし

## Deviation notes

- 2026-09-27 plan: Codex plan advisory の MEDIUM 1 件を反映(warn の文そのものを条件付きにする)。ユーザー決定(AskUserQuestion): 対応案で plan を更新
- 2026-09-27 work: Slice A(24e455f)と B(c71a93b)は implementer に委譲。A: `checkCodexAgmsgWritableRoot` に `projectDir` の引数を追加(`runDoctorFull` が `targetDir` を絶対パスにして渡す)。warn のときだけ既存の `codexRootCoverage` で保存先がプロジェクト内かつ保護ディレクトリをまたがないかを判定し、該当すれば config の有無の両方の文を条件付きの形にする(「送れない」と「root を足す」は別の作業ディレクトリの座席だけの話として書く)。プロジェクト外の文は従来と byte 一致。テストの呼び出し 53 か所を更新し、新しいテスト約 10 件。判定を常に false / true にする mutation で対応するテストが落ちる。実機でプロジェクト内の保存先の Detail を確認。B: recipe 2 コピー、`/org` skill 4 面、tech-debt の行に 1 文ずつ。orchestrator は HEAD 一致・porcelain 空・差分・テスト・`check-skill-sync.sh` を確認
- 2026-09-27 self-review cycle 1(`docs/reports/self-review-2026-09-27-doctor-agmsg-store-in-project.md`、f558dd3): pass、LOW 6。「elsewhere」は task worktree(プロジェクトの中)と矛盾する対比、`(inProject, projectDir)` の組が不正な状態を表せる、コメントの誤りと重複、「seats often run in task worktrees」の裏付けがない、テストの見出しの空行、recipe と skill が保護ディレクトリの例外を書いていない。/test への引き継ぎ: doctor.go の配線を固定するテストがない、AC-1 の「add」側の否定の確認がない
- 2026-09-27 work: Slice C は implementer に委譲(a2faa60、9 ファイル)。6 件を修正し、doctor.go の配線を固定するテスト(`runDoctorOpts` 経由でプロジェクト内の保存先の文が出る)と AC-1 の否定の確認を追加。配線を "" に戻すとその 1 件だけが落ちることを implementer が確認。orchestrator は HEAD 一致・porcelain 空・変更ファイル・差分・テストを確認
- 2026-09-27 verify(`docs/reports/verify-2026-09-27-doctor-agmsg-store-in-project.md`、ebb112d): pass。AC-1〜AC-7 を対応付け、AC-8 は静的な半分のみ(`go test` は /test)。プロジェクト外の文は main と byte 一致、文書の drift なし
- 2026-09-27 中断: ユーザーの依頼で、別のターミナルで続行するため /test の途中で一時停止した(tester は停止し、未コミットの変更なし)。再開の手順は下の「再開の手順」の節
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## 再開の手順(2026-09-27 中断時点)

- worktree: `.claude/worktrees/doctor-agmsg-store-in-project`、branch `fix/doctor-agmsg-store-in-project`(未 push)。この節を追加したコミットが HEAD。`git status --porcelain` は空
- pipeline の状態: `.harness/state/standard-pipeline/active-plan.json` と `cycle-count.json`(cycle 1、cap 2)は worktree に残してある。消さずにそのまま使う
- 済み: plan(Codex plan advisory の MEDIUM 1 件を反映)、work(Slice A 24e455f、B c71a93b、C a2faa60)、self-review cycle 1(LOW 6、すべて Slice C で修正)、verify cycle 1(pass)
- 次に行うこと(この順):
  1. /test cycle 1(`tester`): `./scripts/run-test.sh`、`go test ./internal/cli/ -run 'CodexAgmsgWritableRoot|Doctor' -count=1 -race`、`TMPDIR=/tmp go test ./internal/cli/... -count=1`。mutation(判定を常に true / false、`!blocked` を外す、doctor.go で "" を渡す、2 つの文の入れ替え)、ビルドしたバイナリと main のバイナリで実機の比較(プロジェクト内・`.agents` の下・プロジェクト外)。実機は必ず `env -u ZDOTDIR HOME=<偽の HOME> CODEX_HOME=<偽の codex home>` で実行し、本物の HOME / `~/.codex` / agmsg の保存先に触れない。report は `docs/reports/test-2026-09-27-doctor-agmsg-store-in-project.md`
  2. /sync-docs(`doc-maintainer`): recipe 2 コピー、`/org` skill 4 面、tech-debt の行は Slice B / C で更新済み。drift の再確認と、この plan への記録、sync-docs の report
  3. /cross-review: worktree で Codex の review(`codex -m gpt-6-astra -c 'model_reasoning_effort=xhigh' exec review --base main`、stdin は `/dev/null`)を実行し、triage を `docs/reports/cross-review-triage-doctor-agmsg-store-in-project.md` に書く(件数は `count_triage_findings` で確認)
  4. /pr: `Closes #170`。push の前に `./scripts/run-verify.sh`、`TMPDIR=/tmp go test ./internal/cli/... -count=1`、`./scripts/secret-scan-branch.sh --strict`(exit 0 のときだけ push)。差分が 500 行を超えるので walkthrough を書く。gh の書き込みは `gh auth switch --hostname github.com --user yoshpy-dev` と同じコマンドの中で行い、終わったら `hiroki-yoshioka_dena` に戻す。plan を確定(Status、PR 番号、AC のチェック)してから archive、再 scan、push。CI が pass したら merge(このリポジトリでは自作 PR の CI pass 後の merge が承認済み)、`./scripts/ralph-worktree.sh cleanup --id plan-doctor-agmsg-store-in-project --force-branch`、main を pull
- 注意: コミットは単独の Bash 呼び出しで、メッセージは heredoc から `-F -` で渡す(行番号付きの検索や表示、run-verify と同じ呼び出しに入れると PreToolUse の hook が誤って止める)。secret 風のリテラルを fixture・report・コメントに書かない

## Readiness checklist

- [x] Check の本体、Detail の組み立て、coverage の関数、呼び出し元、文書の該当箇所を確認した
- [x] critical fork なし
- [x] Codex plan advisory(1 件、対応案で plan を更新)
- [x] AC は hermetic な fixture で決定的に確認できる
