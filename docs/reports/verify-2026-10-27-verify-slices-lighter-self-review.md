# Verify report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(AC1〜AC7、D4〜D6、Verify plan、Progress checklist の記録)。issue #261 の受け入れ条件 5 つ
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 44c14b97...HEAD`(HEAD 804d05f9、41 ファイル、+947/-118、12 コミット)。Go のコードはなく、文書、skill、rules、agent の定義と shell のテスト 2 本。self-review の直しの 3 コミット(cb094ae8、bc56cc4f、804d05f9)も、計画の D4・D6 と 2 回目の self-review の LOW 3 件に照らして読んだ。テストは打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-27-verify-slices-lighter-self-review.log`(gitignore。`run-static-verify.sh` の全出力、同期の検査、shellcheck、digest の計算、gofmt の probe、drift の grep)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 雛形 2 つとその写しの `## Implementation outline` に、検証コマンドの書き方と `RALPH_VERIFY_SCOPE=changed` の注意 | Met | `docs/plans/templates/feature-plan.md:38-42` に 3 行のコメント(1 項目 1 スライス、末尾に `— verify: <command>`、Go の例、`RALPH_VERIFY_SCOPE=changed` で絞れない理由)と `1. <slice> — verify: <command>` がある。`.claude/skills/plan/template.md`、`.agents/skills/plan/template.md`、`templates/base/` の 3 つは `cmp` で同一。注意の中身は実物と合う: scope は言語パックを選ぶだけ(`scripts/run-verify.sh:11-14`)、Go の pack は `go test ./...`(`packs/languages/golang/verify.sh:115`)。`gofmt -l` は未整形のファイルを並べても rc 0 で、`test -z` で包むと rc 1 になることを scratch のディレクトリで確かめた |
| AC2 `/plan` の手順 7 がスライスごとの検証コマンドを求める | Met | `.claude/skills/plan/SKILL.md:41-42`(「one item per slice, each ending with that slice's verification command, the narrowest check …」と、雛形のコメントへの参照)。写し 3 つは同一 |
| AC3 handoff と inline のスライスが計画のコマンドを使い、手順 12 と Completion gate にフルの `run-verify.sh` がない。D4 と D6 がある | Met | handoff は `.claude/skills/implement/SKILL.md:40`(「the slice's command from the plan's `## Implementation outline`」)。D4 は同じ行で、コマンドのない計画では最も狭いコマンドを選んで `## Progress checklist` に書いてから渡し、再承認は要らない(digest がその節を読まないのは `scripts/plan-visual.sh:44,78,608` で確認)。inline は `:43` で計画のコマンドを打ち、stage のあと `git diff --cached` で 5 項目を確かめ、結果を Progress checklist に書く(D6)。手順 12 は `:49` で各スライスの検証の結果を確かめ、フルの `run-verify.sh` を打たないと書く。Completion gate は `:84-85,90` で、`run-verify.sh` の行はなく `/verify` と `/test` を求める |
| AC4 Validation Gate、model-routing の handoff と報告の契約、subagent-policy | Met | `.claude/rules/ralph/git-commit-strategy.md:10-13`(手順 2 が計画のスライスのコマンド)。`.claude/rules/ralph/model-routing.md:17-18`(報告に self-check の結果)、`:37`(handoff の行)、`:40-51`(報告の契約、5 項目の正本、found and not fixed と最後のスライス、秘密情報で止めたときの orchestrator の動き)。`.claude/rules/ralph/subagent-policy.md:55-59`。model-routing の root と `templates/base/` の差は KNOWN_DIFF の org receipts の 3 か所だけで、基準の 44c14b97 と同じ形(行番号が 10 ずれただけ) |
| AC5 implementer(Claude Code と Codex)に機械的な確かめと報告の欄 | Met | `.claude/agents/implementer.md:33-40`(stage のあとに 5 項目、範囲内で直す、直せない項目は found and not fixed、消せない秘密情報はコミットせずに止める)、`:48`(報告の欄)。`.codex/agents/implementer.toml:35-53,61-63` が同じ。`templates/base/` の 2 つは `cmp` で同一。TOML 4 つは `tomllib` で読める |
| AC6 `/self-review` が 6 項目で、外した 4 項目は implementer の確かめにある。reviewer の定義と Area も合う | Met | `.claude/skills/self-review/SKILL.md:3`(description)、`:16-21`(6 項目)、`:23-28`(実装の側が見る 5 項目と、`/verify` の branch secret scan と CI の secret scan)。秘密情報の 2 か所は実在する(`scripts/run-verify.sh:250-252`、`.github/workflows/verify.yml:16-19`)。reviewer は `.claude/agents/reviewer.md:3,12-13`、`.codex/agents/reviewer.toml:2,7-16`。Area は `.claude/skills/self-review/template.md:12-13`。D5: self-review の skill 2 つ、`reviewer.md`、`reviewer.toml` に `run-test.sh`・`run-static-verify.sh`・`run-verify.sh` は 0 件 |
| AC7 同期の検査とテスト 6 つが通る | Partially(静的の半分) | `check-skill-sync.sh` と `check-sync.sh` は通った(下の表)。`tests/test-agent-models.sh`、`test-self-review-scope.sh`、`test-new-feature-plan.sh`、`test-slice-verification.sh` の実行は `/test` に残す。2 本の新しいテストは shellcheck と `sh -n` が通り、`scripts/verify.local.sh:292` の `tests/test-*.sh` の glob で拾われる |

issue #261 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 雛形のスライスに検証コマンドの欄と `RALPH_VERIFY_SCOPE=changed` の注意 | Met | AC1 と同じ |
| handoff が計画の検証コマンドを写し、手順 12 にフルの `run-verify.sh` がない | Met | AC3 と同じ |
| implementer の報告の契約に機械的な確かめの結果(Claude Code と Codex) | Met | AC5 と同じ。正本は model-routing の報告の契約 |
| `/self-review` が 6 項目で、外した 4 項目は implementer の確かめにある | Met | AC6 と同じ。issue の「null safety と例外処理」は D3 で 2 行に分け、10 − 4 = 6 になる |
| 既存の同期の検査が通る | Met(静的の半分) | `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh` が通った。`test-agent-models.sh` は `/test` |

self-review の直しの 3 コミット:

- cb094ae8: 5 項目の正本を model-routing に置き、各所を同じ語にした。found and not fixed は次のスライスの前に直す。inline の順序は stage → 確かめ → 打ち直し → 記録 → commit になった。D4 の文に「(for an inline slice, before implementing it)」が足され、D6 の inline の確かめは stage したあとの差分を読む形になった。どちらも計画の意図より狭めていない
- bc56cc4f: 範囲内で消せない秘密情報はコミットせずに止める。implementer の 2 形式に入り、`tests/test-self-review-scope.sh:264-265` が固める。計画の変更は `## Progress checklist` だけ
- 804d05f9: 2 回目の self-review の LOW 3 件に対応した。(1) 最後のスライスは手順 13 の前、秘密情報で止まったときは orchestrator が消すか範囲を狭めて出し直す(`implement/SKILL.md:42-43`、`model-routing.md:48-51`)。(2) Progress checklist の重複行を消し、「このコミット」を bc56cc4f にした。(3) scope テストの `assert_joined_contains_exact` にファイル欠けの分岐を足した(`tests/test-self-review-scope.sh:78-79`)。3 件とも直っている
- 5 項目の並び「unnecessary changes, typos and copy-paste errors, debug leftovers, secrets, and files outside the slice」は、空白を詰めると model-routing、`implement/SKILL.md`、`self-review/SKILL.md`、`implementer.md`、`reviewer.md`、`implementer.toml`、`reviewer.toml` の 7 か所すべてに 1 回ずつ、同じ語で出る
- 計画の digest: 87b8e41c、f6e6e536、bc56cc4f、804d05f9、HEAD の 5 時点とも `bd5c0d0ce1bb` で、`- Approved: 2026-10-27 sha256:bd5c0d0ce1bb` と一致する

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | changed scope を頼んだが、`.codex/agents/implementer.toml` を分類できず full にした(`unclassified`)。golang の pack は gofmt ok、golangci-lint `0 issues.`。local verifier の shellcheck(`tests/test-*.sh` を含む、`scripts/verify.local.sh:192`)、hook の `sh -n`、settings の `jq`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照はすべて OK。secret scan は 44c14b97..804d05f9 で clean |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致。`self-review/SKILL.md` の `.agents` 側は `allowed-tools` の 1 行がないだけで、基準の時点から同じ |
| `./scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | rc 0 | 6 ファイルともパイプラインの全段に触れる |
| `shellcheck tests/test-slice-verification.sh tests/test-self-review-scope.sh` | rc 0 | 出力なし |
| `sh -n` で 2 本のテスト | rc 0 | |
| `git diff --check 44c14b97...HEAD` | rc 0 | 出力なし |
| `tomllib` で `.codex/agents/{implementer,reviewer}.toml` と `templates/base/` の 2 つ | OK | 4 つとも `name`、`description`、`developer_instructions` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 「`run-verify.sh` (or equivalent)」をスライスの検証とする文 | Yes | 履歴(`docs/reports`、`docs/plans/archive`、`docs/evidence`、`docs/specs`)を除くと、計画の Verify plan と、旧い文の復活を禁じる `tests/test-slice-verification.sh:210,231` だけ |
| `/self-review` の旧い 4 項目(`Debug code`、`unnecessary-change`、`typo`、`Secrets and credentials`) | Yes | 残るのは、実装の側の 5 項目として書いた文と、復活を禁じる `tests/test-self-review-scope.sh:182-184`。README.md、AGENTS.md、`docs/quality/*.md`、`.claude/rules/ralph/*.md`、`.codex/*.md` に self-review の観点を並べた文はない(`README.md:220` は「diff quality artifact」、`docs/quality/definition-of-done.md:11` は「diff quality only」) |
| タスク全体の「done の前に `run-verify.sh`」 | Yes(情報) | `.claude/rules/ralph/ralph-workflow.md:49`、`README.md:99-103`、`.codex/AGENTS.override.md:68`、hook の文(`.claude/hooks/post_edit_verify.sh:129`、`.claude/hooks/session_start_context.sh:20`、`templates/base/` の同名)。どれもタスクの完了の話で、`/verify` と `/test` が包みのスクリプトを通して打つので食い違わない。計画は `ralph-workflow.md` を同じ理由で残したと Progress に書き、`.codex/AGENTS.override.md` は Non-goals に挙げた。hook の 2 つは計画に書かれていない。`post_edit_verify.sh` はコードのファイルを編集するたびに「Run ./scripts/run-verify.sh before claiming done」を足すので、implementer がスライスの報告の前にフルの検査を打つよう読む余地がある。implementer の定義の「run the handoff's verification commands exactly as given」(`.claude/agents/implementer.md:25`)が優先するので、矛盾ではなく後続の候補 |
| org 側の検証の文 | 範囲外(#262) | `internal/org/prompts/implementer.md:20` は「指示がなければ `./scripts/run-verify.sh`」のまま。`git-commit-strategy.md:27` の「Org runtime seats commit the same way as the standard flow above」は、手順 2 が計画のスライスのコマンドを指すようになったので、org の座席にはまだ当てはまらない。どちらも計画の Non-goals(#262)にある |
| `/plan` の手順 7 の `RALPH_VERIFY_SCOPE` の注意 | Yes(情報) | issue の「やること」は雛形と `/plan` の記入の手順の両方に注意を書くと言う。手順 7 は注意を繰り返さずに雛形のコメントを指す(`plan/SKILL.md:42`)。issue の受け入れ条件と計画の AC2 は雛形の注意だけを求めるので満たしている |
| 計画の Scope のテストの置き場 | 逸れ(情報、記録なし) | Scope は「`tests/test-slice-verification.sh` が implementer の定義の機械的な確かめの文を固める」と言うが、その検査は S4(d501efad)から `tests/test-self-review-scope.sh:251-272` にあり、slice のテストには一度も入っていない(d501efad、c77b12c9、cb094ae8 の時点で確認)。中身は固められている。Progress checklist には書かれていない |
| 計画の Acceptance criteria と Progress checklist の印 | 遅れ(情報) | AC1〜AC7 はすべて `[ ]`。「Verification artifact created」も `[ ]` で、この report の commit のあとに付ける。Progress の「scope 198/198、agent-models 47/47」は bc56cc4f の時点の数で、804d05f9 が slice のテストに検査を 3 つ足した。数は `/test` で数え直す |

## Observational checks

- `/implement` の手順 6 は「Use the full `./scripts/run-verify.sh` as a slice command only when nothing narrower covers the slice's files」と、フルの検査を最後の手段として残す。計画には書かれていないが、「最も狭いコマンド」の原則の延長で、AC3 の「手順 12 と Completion gate にない」には触れない
- 雛形の Go の例では、`gofmt -l internal/foo` が下位のディレクトリまで見て、`go vet ./internal/foo/` と `go test ./internal/foo/` はそのパッケージだけを見る。gofmt のほうが広いので、取りこぼしはない(scratch の probe で `foo/sub/b.go` も並ぶことを確認)
- `/implement` の手順 7 の inline は「If a secret cannot be removed within scope, do not commit: unstage it and stop. Then commit with conventional format」と続く。「Then commit」は秘密情報がない通常の場合の続きとして読める。文の読みやすさの話なので `/self-review` の範囲で、この report の判定には入れない

## Coverage gaps

- AC7 のテスト 4 つ(`test-agent-models.sh`、`test-self-review-scope.sh`、`test-new-feature-plan.sh`、`test-slice-verification.sh`)と `./scripts/run-test.sh` の全体は `/test` に残す
- 新しい文を実際の `/implement` の実行で使った結果(implementer が 5 項目の結果を返すか、コマンドのない計画で orchestrator が Progress に書くか)は、この PR では確かめられない。今回の 5 スライスと直しのスライスの記録(Progress checklist の bc56cc4f の行)が 1 件の実例になる
- Codex 側の implementer と reviewer の定義が Codex の custom agent として読まれることは、TOML として読めることまでを確かめた。Codex の実行での振る舞いは確かめていない
- 文言の検査は 2 本のテストが固めるが、検査が変異で落ちることは計画の記録(S5 の「変異 5 種」)による。この verify では打ち直していない

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC5、AC6。issue の受け入れ条件 1〜4。self-review の直しの 3 コミットは計画の D4・D6 を狭めず、2 回目の self-review の LOW 3 件をすべて直している。計画の digest は承認の値のまま。静的解析はすべて通った(`run-static-verify.sh` rc 0、`check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、shellcheck、`sh -n`、`git diff --check`、TOML の読み込み)
- Partially verified: AC7 と issue の受け入れ条件 5(同期の検査は通った。テスト 4 つは `/test`)
- Not verified: テストの実行、新しい文での実際の `/implement` の振る舞い、Codex の実行での agent の定義の読まれ方
- Findings: 情報 5 件(hook の「done の前に run-verify.sh」が implementer に読まれる余地、org 側の文は #262、`/plan` の手順 7 は注意を雛形に任せる、テストの置き場の逸れが記録されていない、計画の印と Progress の数の遅れ)。CRITICAL・HIGH・MEDIUM・LOW はない
- 判定: pass(HEAD 804d05f9)
