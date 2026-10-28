# review-switches-skills

- Status: Approved
- Approved: 2026-10-29 sha256:3cf8dfa26cbf
- Owner: Claude Code
- Date: 2026-10-29
- Related request: 標準フロー: 計画助言を相手のモデルにし、計画助言と cross-review を設定で切り替える
- Related issue: #266
- Type: feat
- Branch: feat/review-switches-skills

## Objective

標準フローの `/plan` の計画助言を、cross-review と同じ判別で相手のモデルにする(driver が Claude Code なら codex、Codex なら claude)。`/plan` と `/cross-review` が #265 の設定(`[standard].plan_advisory`、`[standard].cross_review`)を `ralph config get` で読み、off なら飛ばす。飛ばしたことは計画、cross-review の triage の report、PR の本文に残す。「optional」の意味を、設定と相手の CLI の有無に合わせて文書で書き直す。

あわせて、#265(PR #284)から持ち越した 3 点を片付ける: skill に `ralph config get` の読み方の約束を書く(#265 の D4)、repo の root から打つ前提、PATH に `ralph` がないとき(exit 127)の扱い、ルートの `ralph.toml` のキーを読む常設のテスト。

## Scope

- `/plan` skill の手順 11(`.claude/skills/plan/SKILL.md` と `templates/base/.claude/skills/plan/SKILL.md`、`./scripts/sync-skills.sh` で `.agents/` の写し)
  - 名前を「Codex plan advisory」から「相手のモデルの計画助言(other-model plan advisory)」に改める。
  - 最初に設定を読む(下の「読み方の約束」)。off なら飛ばす。
  - driver と相手を決める(D5): driver は skill を動かしている CLI(Claude Code なら claude、Codex なら codex)で、`RALPH_PRIMARY_CLI` があればそれが優先する。相手は `scripts/xreview-helpers.sh` の `pick_reviewer` で決める。`which` は driver の判別には使わず、選んだ相手の CLI があるかの確かめにだけ使う。
  - 相手が codex のとき: 今の呼び出し(`codex exec --sandbox read-only --ignore-rules`、20 分の watchdog)のまま。
  - 相手が claude のとき(D2): `claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"` を、書き込みをさせず、project の hook も走らせない形で呼ぶ。`--restricted`(user・project・local の settings を読まないので hook が載らない)、`--strict-mcp-config` と `--disallowedTools "mcp__*"`(MCP の道具を止める)、`--tools Read,Grep,Glob`(読む道具だけ)、`--permission-mode dontAsk`(許可を求める操作は断る)、環境に `RALPH_WIP_CHECKPOINT=off`(万一 hook が走っても WIP コミットをしない)。flag は実装で claude 2.1.304 の help と小さな実行で確かめ、計画の Progress に書く。同じ watchdog の形、`</dev/null`、出力のファイル、完了の契約(rc 0 かつ出力が空でない)を codex とそろえる。
  - 呼ぶ前と後で `git rev-parse HEAD`、`git status --porcelain`(未追跡を含む)、`git diff --cached` を比べ、変わっていれば止まって人に報告する(Codex の計画助言 1)。相手が codex のときも同じ確かめをする。
  - 相手の CLI がないとき: 今までどおり飛ばして記録する。
  - 結果(実施、off で飛ばした、古い ralph で飛ばした、ralph がなくて飛ばした、相手の CLI がなくて飛ばした、未完了)を、計画の `## Progress checklist` に書く(digest の外)。
- `/plan` skill の手順 12 と冒頭の説明の「Codex findings」「Codex advisory」も、相手のモデルの計画助言の言い方に改める。
- `/cross-review` skill(4 つの写し)
  - 手順 1 の前に設定を読む。off なら reviewer を呼ばず、triage の report を `Reviewer status: skipped (<理由>)` と `After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0` の形で書き、insight event を `--verdict n/a` で足して `/pr` に進む。
  - 相手の CLI がないときも同じ形で report を書く(今は「skip to /pr」だけで report を書かない)。冒頭の説明と CLI の表の「silently skips」も書き直す。
  - 手順 2 の driver の判別を `/plan` と同じにする(D5)。
  - 相手が claude のとき(driver が Codex)の `claude -p --permission-mode auto` の呼び出しにも、`RALPH_WIP_CHECKPOINT=off` と、前後の HEAD・index・作業ツリーの確かめを足す(Codex の計画助言 1 と同じ穴。reviewer は差分を読むので道具の制限は今のまま)。
- `docs/reports/templates/cross-review-triage-report.md`(と写しがあれば写し): `Reviewer status:` の値に `skipped (<reason>)` を足す(consult の指摘)。
- `.codex/AGENTS.override.md` と `templates/base/.codex/AGENTS.override.md`: 「post-implementation pipeline の前に `RALPH_PRIMARY_CLI=codex`」を、`/plan` の計画助言と cross-review の前にも渡す形に直す(Codex の計画助言 2)。
- `/pr` skill(4 つの写し)と PR の雛形: 計画助言と cross-review を飛ばしたときは、本文の Verification にその理由を書く。
- `/implement` skill の手順 13 の e(4 つの写し): 「optional」の説明を設定に合わせる。
- 読み方の約束(skill に書く、#265 の D4 と D6)
  - repo の root(task worktree の root)で `ralph config get <key>` を打つ。cwd の `./ralph.toml` を読むため。
  - 終了コード 0: 出力が `true` なら実施、`false` なら off として飛ばす。
  - 終了コード 1 で stderr が `unknown command "config"` を含む(動詞のない古い ralph): off として飛ばし、「古い ralph」と記録する。
  - 終了コード 127(`ralph` が PATH にない): off として飛ばし、「ralph がない」と記録する。
  - それ以外(2、3、ほかの 1): 止まって人に報告する。黙って off にしない。
- `internal/insights/backfill.go`: triage の report の `Reviewer status:` が `skipped` で始まるときも、`incomplete` と同じく verdict を `n/a` にする。テストを足す。
- `internal/config` のテスト: ルートの `ralph.toml` を、知らないキーを拒む読み方(go-toml/v2 の `DisallowUnknownFields`)で読み、知らないキーがないことと、4 つの実施有無が true であることを確かめる(#284 の Known gaps)。
- `internal/config/defaults_sync_test.go`: `/plan` skill が新しく読む `RALPH_CLAUDE_REVIEWER_MODEL` の fallback(`opus`)を、`scripts/ralph-config.sh` の既定と照合する(cross-review の照合と同じ形)。
- 文書の「optional」の書き直し: `.claude/rules/ralph/post-implementation-pipeline.md`、`.claude/rules/ralph/ralph-workflow.md`、`.claude/rules/ralph/subagent-policy.md`(どれも `templates/base/` の写しを含む)、`.claude/rules/ralph/model-routing.md`(計画助言も `RALPH_CLAUDE_REVIEWER_MODEL` を読む)、`README.md`、`AGENTS.md` と `.ralph/core/AGENTS.core.md`、`docs/quality/definition-of-done.md`(と写し)。どれも「設定が on で、相手の CLI があるときに走る。off か、相手の CLI がなければ飛ばして記録する」の意味にする。
- PR の本文の release のノート: 既定の off で、scaffold した project の計画助言と cross-review が飛ばされるようになる(続けたい project は `[standard]` の 2 つを true にする)。新しい binary(`ralph config get` のある ralph)が要る。driver が Codex のとき、計画助言は claude を呼ぶ。

## Non-goals

- org の leader の計画助言(#267)と reviewer の cross-review(#269)。
- 標準フローのほかの設定を `ralph.toml` に移すこと(#276)。`RALPH_CLAUDE_REVIEWER_MODEL` などはまだ環境変数で読む。
- claude の reviewer の effort(#276)。
- `ralph config get` の動詞そのものの変更(#265 で入った。終了コード 3 と 2 の順序のテストは #284 の Known gaps のまま)。
- この repo の PATH の `ralph` を新しい binary に入れ替えること(利用者の環境)。

## Assumptions

- #265(PR #284)で `ralph config get` と `[standard]` の 2 つ、ルートの `ralph.toml`(4 つが true)が main に入っている。
- 2026-10-27 のユーザーの方針(issue #266)は蒸し返さない。
- この repo の PATH の `ralph` は brew の v6.0.0 で `config get` を持たない(2026-10-29 に確認)。新しい binary に入れ替えるまで、この repo の skill は「古い ralph」として計画助言と cross-review を飛ばして記録する。

## Affected areas

- `.claude/skills/{plan,cross-review,pr,implement}/SKILL.md` と、それぞれの 3 つの写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)、`.claude/skills/pr/template.md` と写し
- `internal/insights/backfill.go` とテスト、`internal/config/defaults_sync_test.go`、`internal/config` の新しいテスト
- `.claude/rules/ralph/` の 4 つ(と写し)、`README.md`、`AGENTS.md`、`.ralph/core/AGENTS.core.md`、`docs/quality/definition-of-done.md`(と写し)

## Visual review

None (skill の手順と文書の書き換えが中心で、新しい仕組みの部品は `ralph config get` を読む分岐だけ。図にすると文の言い換えになる)

## Design decisions

- D1 読み方の約束は skill に書く(#265 の D4 の持ち越し)。exit 127 も「古い ralph」と同じく off として飛ばし、記録で区別する。ralph がない環境で計画助言や cross-review を止めると、標準フローそのものが進まなくなるため。設定の誤り(2、3)だけを止める。
- D2 相手が claude のときの計画助言は、読む道具だけを許し、許可を求める操作はすべて断り、MCP の道具と project の hook も止める形で呼ぶ(issue の「書き込みをさせない権限」)。`--tools` は MCP の道具を制限しない(consult が CLI reference で確認)ので `--strict-mcp-config` と `--disallowedTools "mcp__*"` を足す。hook はモデルの道具の制限と別に user の権限で走り、この repo の SessionEnd の hook は feature branch の未コミットの変更を `git add -A` で WIP コミットする(Codex の計画助言 1、HIGH)ので、`--restricted` で project の settings を読まない形にし、`RALPH_WIP_CHECKPOINT=off` も渡し、前後の git の状態を比べる。正確な flag は実装で claude 2.1.304 の help と小さな実行で確かめて決め、計画の Progress に書く。
- D5 driver は skill を動かしている CLI 自身(Claude Code か Codex か)とし、`RALPH_PRIMARY_CLI` を明示の上書きにする。`which` で driver を推すと、両方の CLI が入っていて `RALPH_PRIMARY_CLI` がないとき、Codex の driver でも claude とみなして codex を相手に選ぶ(Codex の計画助言 2、MEDIUM。読み取りで再現した)。skill を読んでいる agent は自分がどちらの CLI かを知っているので、それを使う。`.codex/AGENTS.override.md` も、`/plan` の前から `RALPH_PRIMARY_CLI=codex` を渡す形にする。`pick_reviewer` の対応(driver の反対)は変えない。
- D3 飛ばした cross-review も triage の report を書く(`Reviewer status: skipped (<理由>)`、数は 0)。`/pr` と insights が report の有無に頼らずに済み、飛ばした理由が report に残る。insights の backfill は `skipped` を `incomplete` と同じく verdict n/a として読む。
- D4 ルートの `ralph.toml` のテストは、知らないキーを拒む読み方で読む。テンプレートでキーを改名したとき、ルートの古い名前を黙って無視しないため。`config.Load` は知らない表を意図して無視する(`TestLoad_IgnoresRetiredOrgBudgetTable`)ので、Load ではなく `toml.NewDecoder(...).DisallowUnknownFields()` で別に読む(consult)。
- Critical forks: None(方針は issue で決まっており、残る選択はどれも 1 スライスの範囲で戻せる)。consult に聞いて確かめる。

## Acceptance criteria

- [x] AC1: `/plan` の手順 11 が、driver が Claude Code なら codex、Codex なら claude を呼ぶ形で書かれている。driver は skill を動かしている CLI(`RALPH_PRIMARY_CLI` が上書き)で、`which` で推さない。claude の呼び出しは読む道具だけで、許可を求める操作を断り、MCP の道具と project の hook を止め、`RALPH_WIP_CHECKPOINT=off` を渡す。モデルは `RALPH_CLAUDE_REVIEWER_MODEL`(fallback `opus`)。名前は相手のモデルの計画助言。S2 の小さな実行で、claude の呼び出しが計画を読めて、`mcp__` の道具が見えず、呼ぶ前後で HEAD・index・作業ツリー(未追跡を含む)が変わらないことを確かめる(未コミットの変更がある feature branch で)。
- [x] AC9: `.codex/AGENTS.override.md` と写しが、`/plan` の計画助言と cross-review の前にも `RALPH_PRIMARY_CLI=codex` を渡すと書いている。`/cross-review` の手順 2 の driver の判別も AC1 と同じ。`/cross-review` が claude を呼ぶ道にも `RALPH_WIP_CHECKPOINT=off` と前後の git の状態の確かめがある。triage の雛形に `skipped (<reason>)` がある。
- [x] AC2: `/plan` と `/cross-review` が、最初に repo の root で `ralph config get standard.plan_advisory` / `standard.cross_review` を打ち、読み方の約束(0 の true/false、1 と `unknown command "config"`、127、それ以外は止まる)に従う。off、古い ralph、ralph がない、相手の CLI がないときは飛ばし、計画の Progress checklist(`/plan`)と triage の report の `Reviewer status: skipped (<理由>)`(`/cross-review`)に理由を残す。`/pr` が PR の本文にその理由を書く。
- [x] AC3: 設定が on で相手の CLI がないときは、今までどおり飛ばして記録する(`/cross-review` は report を書く形に変わる)。
- [x] AC4: `internal/insights` の backfill が `Reviewer status: skipped …` の cycle を verdict n/a として読む(テスト)。
- [x] AC5: ルートの `ralph.toml` を知らないキーを拒む読み方で読むテストがあり、4 つの実施有無が true であることを確かめる。キーを 1 つ改名した写しで落ちる。
- [x] AC6: `RALPH_CLAUDE_REVIEWER_MODEL` の plan skill の fallback と `scripts/ralph-config.sh` の既定の照合が `TestDefaultsLockStep` にある。
- [x] AC7: 「optional」の文が、Scope に挙げた文書で「設定が on で相手の CLI があるときに走る。そうでなければ飛ばして記録する」の意味になっている(`git grep` で確かめる)。
- [x] AC8: `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`go test ./internal/config/ ./internal/insights/`、skill の文を固定するテスト(`tests/test-check-skill-sync.sh`、`tests/test-slice-verification.sh` など、実装で洗い出す)が通る。

## Implementation outline

1. S1 Go の土台: `internal/insights/backfill.go` の `skipped` の扱いとテスト、`internal/config` のルートの `ralph.toml` のテスト。 — verify: `test -z "$(gofmt -l internal/insights internal/config)" && go vet ./internal/insights/ ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/insights/ ./internal/config/`
2. S2 `/plan` の手順 11 と 12: 読み方の約束、driver の判別(D5)、claude の呼び出し(D2)、前後の git の状態の確かめ、結果の記録、名前の改め。4 つの写し。`defaults_sync_test.go` に plan skill の `RALPH_CLAUDE_REVIEWER_MODEL` の fallback の照合。claude の呼び出しを小さな実行で確かめる(AC1)。 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ -run 'LockStep'`
3. S3 `/cross-review`、`/pr`、`/implement`、Codex の上書き: 設定の読み、`skipped` の report と triage の雛形、driver の判別(D5)、claude の reviewer の `RALPH_WIP_CHECKPOINT=off` と前後の確かめ、PR の本文への記録、手順 13 の e、`.codex/AGENTS.override.md` と写し。4 つの写しと PR の雛形。 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && bash tests/test-check-skill-sync.sh && bash tests/test-pr-owner-lookup.sh && bash tests/test-slice-verification.sh`
4. S4 文書の「optional」: rules の 4 つ(と写し)、`README.md`、`AGENTS.md` と `.ralph/core/AGENTS.core.md`、`docs/quality/definition-of-done.md`(と写し)。 — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && ./scripts/check-template-purity.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Init'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`。
- Spec compliance criteria to confirm: AC1〜AC8。issue #266 の受け入れ条件 4 つ。
- Documentation drift to check: AC7 の `git grep`(「optional」「Codex plan advisory」「Codex not available」)。skill の 4 つの写しの一致。
- Evidence to capture: `run-static-verify.sh` の出力、claude の計画助言の呼び出しを小さな計画で 1 回走らせた結果(S2)。

## Test plan

- Unit tests: backfill の `skipped`、ルートの `ralph.toml` のテスト、fallback の照合。
- Integration tests: skill の文を固定する shell のテスト。
- Regression tests: backfill の `incomplete` と `complete` の扱いが変わらないこと。
- Edge cases: ルートの `ralph.toml` に知らないキーがある写し(落ちる)、`Reviewer status: skipped` の report の cycle 2。
- Evidence to capture: `./scripts/run-test.sh` の出力。

## Risks and mitigations

- R1: 既定の off で、scaffold した project の計画助言と cross-review が飛ばされるようになる。軽減: issue の方針。release のノートに、続けたい project は `[standard]` の 2 つを true にすると書く。
- R2: この repo の PATH の `ralph` は古いので、merge のあとこの repo の skill は計画助言と cross-review を「古い ralph」として飛ばす。軽減: 記録が残るので黙っては外れない。新しい binary(`go install ./cmd/ralph` で PATH の順を brew より前にするか、次の release)で戻る。merge の報告でユーザーに伝える。`go run ./cmd/ralph` は子の終了コード 2・3 を 1 に変えるので、代わりにはならない(consult)。
- R6: claude の計画助言の flag(`--restricted` など)の組み合わせで、計画の調べ物に要る読み取りまで止まる、または hook が別の経路で走る。軽減: S2 の小さな実行で、計画を読めることと git の状態が変わらないことを確かめる。前後の git の状態の比較は、flag が効かなかった場合の最後の守り。
- R3: claude の `-p` を読む道具だけで動かす flag が、claude の版で変わりうる。軽減: S2 で小さな実行で確かめ、完了の契約(rc 0 かつ出力あり)を codex と同じにして、だめなら「未完了」として記録して進む。
- R4: Codex の sandbox の中から `claude -p` が通るかは分からない。軽減: 通らなければ相手の CLI の呼び出しの失敗として「未完了」を記録して進む(今の codex の未完了と同じ扱い)。
- R5: skill の 4 つの写しと README・AGENTS.md は、ほかのセッションの PR と衝突しやすい。軽減: 衝突したら両方を残し、`sync-skills.sh` で写しを作り直す。

## Rollout or rollback notes

- release のノート(PR の本文): 標準フローの計画助言と cross-review は `[standard].plan_advisory` と `[standard].cross_review` で決まり、既定は off。今までどおり走らせたい project は 2 つを true にする。新しい binary が要る(古い ralph では off として飛ばす)。driver が Codex のとき、計画助言は claude を呼ぶ。飛ばした cross-review も triage の report を書く。
- 戻すときはこの PR を revert する。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — consult-plan-switchskills: 直してから進める(D2 に MCP の拒否、Scope に triage の雛形・`/plan` の手順 12 の名前・`/cross-review` の「silently skips」、D4 は Load でなく Decoder で読む、R2 に `go run` は代わりにならないこと)。Codex plan advisory: HIGH 1(claude の呼び出しで project の SessionEnd の hook が `git add -A` で WIP コミットしうる → D2 に `--restricted`・`RALPH_WIP_CHECKPOINT=off`・前後の git の状態の比較、`/cross-review` の claude の道にも)、MEDIUM 1(両方の CLI があると Codex の driver でも codex を相手に選ぶ → D5 で driver を skill を動かす CLI 自身にし、`.codex/AGENTS.override.md` を直す)。どちらも反映(AC1 と AC9)。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest は Approved の行。Critical forks: None
- [x] Branch created (feat/review-switches-skills, worktree .claude/worktrees/review-switches-skills)
- [x] Implementation started
  - S1 07cf95f8(impl-266-s1): backfill の field と変数を `incomplete` から `noReview` に改め、`incomplete` か `skipped` で始まる status を verdict n/a にした。テスト 2 つ(skipped の report、cycle 1 は complete で cycle 2 は Skipped)。`internal/config/root_ralph_toml_test.go` の `TestRootRalphToml_KnownKeysAndReviewSwitchesOn` は、ゼロ値の `Config` に `DisallowUnknownFields` で読み(ファイル自身が 4 つを true に書いていることを確かめるため)、ファイルがないときは Fatal(追跡しているファイルで、消すと 4 つが黙って off に戻るため)。写しで、キーの改名・false・削除がそれぞれ落ち、backfill の `skipped` の判定を消すと新しい 2 つのテストが落ちた。検証コマンド rc 0(orchestrator も `-count=1` で打ち直した)。
  - S2 93dec2cc(impl-266-s2): `/plan` の手順 11 を相手のモデルの計画助言に(a 設定の読み、b driver と reviewer、c 起動、d 完了の契約と後の確かめ、e〜h は reviewer の所見)。codex の起動は 11.c に残した(`/cross-review` が 11.c を codex の理由と watchdog の参照先にしているため)。claude の呼び出しは計画の形のまま、flag の変更なし(`RALPH_WIP_CHECKPOINT=off command claude -p --model … --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "<prompt>"`、prompt は値を複数取る flag のあとに置く)。git の状態の確かめは、計画の 3 つに `git diff` と未追跡のファイルの中身のハッシュを足した(porcelain だけでは、変更済みや未追跡のファイルへの編集が見えないため)。exit 0 で true でも false でもない出力も「止まる」側。結果は Progress の `Plan reviewed` の項目に `plan advisory: <outcome>` で書く。`defaults_sync_test.go` に plan skill の `RALPH_CLAUDE_REVIEWER_MODEL` の fallback の照合。小さな実行(haiku): `--restricted` なしでは `-p` でも project の hook が走って wip コミットが入り、skill の形では hook も git の変化もなく、init event は道具が Glob・Grep・Read だけ、MCP server 0。`--disallowedTools "mcp__*"` は glob が効く(274 → 0)。検証コマンド rc 0、`tests/test-codex-exec-invocation.sh`・`test-slice-verification.sh`・`test-check-skill-sync.sh` も通った。fallback を変えると `shell_and_skills` が落ちる。範囲外の気づき: `scripts/ralph-config.sh` の見出しのコメント(16〜17、31〜33、37〜38 行)と `tests/test-codex-exec-invocation.sh:9` のコメントが「Codex plan advisory」の言い方のまま。S4 の範囲に足す(コメントだけ)。`model-routing.md` の sync note は S4 のとおり。
  - S3 9e233825(impl-266-s3): `/cross-review` に手順 0(設定の読み、`/plan` 11.a と同じ文)。手順の番号は変えていない(insights の README、`insights-append.sh`、`paneenv.go`、`/plan` 11.c、`post-implementation-pipeline.md` が番号を名指すため)。飛ばすときは手順 1・2 だけ走らせて見出しの cycle・driver・reviewer を決め、手順 6 で skipped の report(数 0、Triage context に理由)を書き、件数を確かめ、insight を `--verdict n/a` で足して `/pr` へ(cycle-count は増やさない)。相手の CLI がないときも同じ。手順 2 は `/plan` 11.b と同じ判別(D5)。両方の reviewer の道に `/plan` 11.c と同じ git の状態の確かめ、claude の道に `RALPH_WIP_CHECKPOINT=off`(道具は今のまま、PR の Known gaps に `/plan` の形との差を書く)。triage の雛形の `skipped (<reason>)`。`/pr` の手順 5 と雛形の Verification に、計画助言と cross-review の行(ran / incomplete / skipped と理由、記録がなければ `not recorded`)。`/implement` の 13.e。`.codex/AGENTS.override.md` と写しで、`/plan` と `/cross-review` の前にも `RALPH_PRIMARY_CLI=codex`。検証コマンド rc 0、`test-codex-exec-invocation.sh`・`test-skill-insight-cycle.sh`・`test-sync-skills.sh` も通り、skipped の report は `count_triage_findings` で 0/0/0。範囲外の気づき(S4 に足す、コメントと文書だけ): `scripts/xreview-helpers.sh:62-64` と写し(「the /cross-review auto-detect default」)、`internal/org/driver/paneenv.go:34-40`(「/plan's advisory calls codex directly」は S2 で事実と違う)、`docs/recipes/codex-setup.md:67` と写し(「auto-detects when only codex is on $PATH」)。
  - S4 f88561b8(impl-266-s4): rules の 4 つ(と写し)、`AGENTS.md`・`.ralph/core/AGENTS.core.md`(と写し)、`README.md`、`docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md`(sweep で見つかった)、`docs/recipes/codex-setup.md`(と写し)、`scripts/ralph-config.sh`・`scripts/xreview-helpers.sh`(と写し)・`internal/org/driver/paneenv.go` のコメント、`tests/test-codex-exec-invocation.sh:9` のコメント、`.codex/config.toml`(と写し)のコメント。「optional」は「switch で決まり、飛ばしたら記録する」に。26 ファイル。検証コマンド rc 0。`paneenv.go` の warning の文字列はテストが固定しているので変えていない(RALPH_PRIMARY_CLI のない座席には今も正しい)。/sync-docs に渡すこと: `templates/base/ralph.toml:28-29` の「The skills start reading these switches in a later release.」は、この branch で事実と違う(`[org.review]` 側の同じ文は #267・#269 まで正しいので残す)。`docs/tech-debt/README.md:31`(pick_reviewer の唯一の呼び手が /cross-review の auto-detect と書く。trigger「2 つ目の呼び手」が /plan 11.b で満たされた)と `:154` の (c)(claude の呼び出しの形)を直すか PR の Known gaps にする。
- [x] Review artifact created — self-review 1031e5f4(2 回目、Merge: OK、1 回目の L1 (a)(b)・L2・L3・L4・L6 は閉じた。新しい LOW 2: N1 は orchestrator がインラインで直した、N2 は PR の Known gaps)
  - N1(orchestrator のインライン、`/cross-review` の 4 つの写し、コミットは下の記録の直前): claude の道の prompt の雛形が、前の呼び出しの `$BASE`(表示も導出し直しもしない)を使っていた。F1 が直した型と同じ。空の差分の確かめの行を `echo "base=$BASE rc=$?"` にして base を記録させ、雛形は `<base>`(表示された値)を使う形にした。bash と zsh で `base=main rc=1`。`check-skill-sync`、`check-sync`、`test-codex-exec-invocation.sh`(144/144)、`test-check-skill-sync.sh` が通った。機械的な確かめ: 5 項目とも clean。文の直しなので self-review は回し直さず、/verify・/test・cross-review で見る。
  - N2 は PR の Known gaps: driver の上書きの処理(小文字にして claude/codex だけ採る)が skill の本文の 8 か所に同じ形で入った。テストはなく、`check-skill-sync` は写しの一致しか見ない。`xreview-helpers.sh` に helper とテストを足す変更は、この PR の範囲を超える。
  - self-review baa2f400(1 回目): Merge: OK、LOW 6。L2(`$DRIVER` を同じ Bash の呼び出しの中で定めていない。空だと `pick_reviewer` は codex を返し、D5 が直そうとした誤りと同じ形になる)と L3(`/cross-review` の手順 4 の空の差分だけが report を書かずに進む)は skill の動きに効くので、修正スライス F1 で直す。L4(`/pr` の雛形の語彙と記録の語彙の対応、`Plan reviewed` の box を付ける手順)、L6(コメントの折り返し)、L1 の (a)(`config.go:31` の「start to in issue #266」)と (b)(テンプレートの `[standard]` の「only that」が exit 127 の扱いより狭い)も F1 で直し、S4 の持ち越しの `templates/base/ralph.toml:28-29` の古い 1 文も F1 に入れる。L1 の (c)(`paneenv.go` の warning)は consult の判定どおり据え置き。L5(`/cross-review` の claude の道の形)は PR の Known gaps。tech-debt の 2 行は /sync-docs。F1 のあと self-review をもう 1 回回す(cycle-count は増やさない)。
  - F1 08b2ceba(impl-266-f1): L2 は driver と reviewer を 1 回の Bash で決める snippet(`DRIVER=<claude|codex>`、`RALPH_PRIMARY_CLI` を小文字にして claude/codex のときだけ上書き、`pick_reviewer`、値を表示)と「後の手順は表示された値を使う」。両 skill で同じ。L3 は空の差分も `skipped (empty diff)` の report を書く。空の差分の確かめも前の呼び出しの `$BASE` に頼っていて、`BASE=` のとき rc 0 で「差分なし」と誤るので、`detect_base_branch` と同じ 1 回の呼び出しにまとめた(L3 の範囲に入れたずれ)。L4 は `/pr` の記録と Verification の行の対応(`ran (<reviewer>)`、incomplete、skipped、`not recorded`)、`/plan` 11 は記録するとき `Plan reviewed` の box も付ける。L6 はコメントの折り直し。L1 (a) `config.go` のコメント、(b) テンプレートの `[standard]` のコメント(later release の文を消し、exit 127 も off として記録する、と足した)。snippet を bash と zsh の新しい shell で 8 通り走らせ、期待どおりの driver と reviewer。検証コマンド rc 0。
- [x] Verification artifact created — verify f0290dc4(cycle 1、pass。`run-static-verify.sh` rc 0、golangci-lint 0 件。AC1〜AC3・AC9 を満たす、AC4〜AC6 は読んで満たす(実行は /test)、AC7 は満たすが V1〜V4 が残る。PATH の ralph 6.0.0 は 2 つのキーとも rc 1 と `unknown command "config"` で off の分岐、build した binary は root で `true`、知らないキー 2、型の誤り 3、ralph なし 127 を実測)。LOW の V1(README:135 が off にする失敗を古い ralph だけと書く)、V2(tech-debt:31)、V3(tech-debt:154)、V4(飛ばす条件に空の差分を挙げない十分条件の文 3 か所: implement 13.e、ralph-workflow.md:41-42、codex-setup.md:65-66、と写し)は /sync-docs で直す。issue の「判別の順に which」を D5 で外したことは PR の本文に書く。
- [x] Test artifact created — test cd97750e(cycle 1、pass。`run-test.sh` rc 0、shell 45 本、Go 9 パッケージ、cli 368 s・org 476 s。名指しのテストと shell の 6 本が通った。変異 7 個中 6 個が落ちた。生き残った a3(`After triage:` のあとの `pendingNoReview` のリセットを消す)は base でも生き残る既存の穴で、PR の Known gaps に書く。driver の snippet は 8 つの写しで同じ 1 行で、bash と zsh で 28 通りが期待どおり。base の行は差分ありで rc=1、差分なしで rc=0、ref の誤りで rc=128(この場合は reviewer を呼んで失敗し incomplete になる)。skipped の report 5 通りは `count_triage_findings` で 0/0/0、HEAD の binary の backfill は n/a と読み、base の binary は pass と読んだ)
- [x] Sync-docs report created — `docs/reports/sync-docs-2026-10-29-review-switches-skills.md`(cycle 1。verify の V1〜V4 を直した: README の `ralph config get` の行に exit 127 を足した、tech-debt の `pick_reviewer` の行と `/cross-review` の claude の道の行を今の呼び手と呼び出しに合わせた、空の差分を言わない十分条件の文を `/implement` 13.e・ralph-workflow.md・codex-setup.md・definition-of-done.md(と写し)で直した。sweep で codex-setup.md の `origin/$BASE` も local の base に直した。release ノートの本文は report の「Release notes」)
- [x] Cross-review — 5634ccca(codex、cycle 1/2、指摘 0、Case C。PATH の ralph が v6.0.0 なので、main の版の skill で回した)。AC7 は sync-docs 67410c6f で V1〜V4 を直したので `[x]` にした。consult-plan-switchskills の完了判定: 進めてよい(release のノートに claude の `--restricted` の版の下限を足す)。
- [ ] PR created
