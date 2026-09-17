# org-envelope-low-findings

- Status: Done (PR #157)
- Owner: Claude Code
- Date: 2026-09-17
- Related request: PR #152 の self-review(cycle 1 / cycle 2)で LOW と判定し、パイプライン cap 到達時に先送りした所見 10 件を 1 PR で一括修正する(issue #154)
- Related issue: 154
- Type: chore
- Branch: chore/org-envelope-low-findings

## Objective

`docs/tech-debt/README.md` の「org-implementer-seat-envelope: deferred LOW findings batch (cosmetic)」行に列挙された 10 件を 1 PR で処理し、行をクローズする。すべてコスメティック(stale doc comment、テストアサートの精度、文体、冗長コード、best-effort 経路のエラー捨て)で、org runtime の実行時挙動は `ralph doctor` の info 経路の文言以外変えない。

## Scope

| # | 所見 | ファイル | 対応 |
|---|------|---------|------|
| 1 | `codexModelsCachePath` が `os.UserHomeDir` のエラーを捨て、`CODEX_HOME` を trim せずに join | `internal/cli/doctor_codex_models.go`, `internal/cli/doctor_org_test.go` | `CODEX_HOME` は **trim しない**。codex 本体の `find_codex_home`(`codex-rs/utils/home-dir/src/lib.rs`、rust-v0.149.1 と rust-v0.154.0 で同一を 2026-09-17 に確認)は `is_empty()` でのみ濾し、空でなければ値をそのまま使うので、doctor も `home == ""` の厳密判定に揃える(現状の `TrimSpace(home) == ""` は空白のみを未設定扱いにしており、codex とずれる)。home 解決失敗時は `(string, error)` で返し、呼び出し元 `checkCodexModelSlugs` は status `info` +「could not resolve … — skipping」で報告する。テスト 2 件追加(末尾空白付きディレクトリを literal に読む回帰 / HOME 空でのエラー経路) |
| 2 | `DefaultModelForDriver` の doc comment の呼び出し元記述 | `internal/org/envelope_summary.go` | 現状確認のみ。463e943 で「production caller なし、CLI は role-aware 版を `resolveModelOrWarn` 経由で使う」に更新済み。`grep -rn 'DefaultModelForDriver(' internal --include='*.go' \| grep -v _test` は定義行のみで一致しており修正不要。Deviation notes に記録 |
| 3 | `RolePromptVars.PlanPath` の doc が非消費者として `implementer.md` を挙げていない | `internal/org/prompts.go` | 「4 雛形(lead/implementer/reviewer/qa)のいずれも `{{PLAN_PATH}}` を参照しない」に書き換え。同じコメントにある「PR③」前方参照(struct doc と replacer コメントの 2 箇所)も「将来の `--plan` フラグ配線のため保持」に中立化する |
| 4 | `/org` skill の Leaded 行が `lead.md` の implementer 優先委譲と矛盾 | `.claude/skills/org/SKILL.md`(+ `.agents/`, `templates/base/.claude/`, `templates/base/.agents/` の 3 ミラー) | 「実装は Lead 自身か既存フローに任せつつ」を「実装は完了済みか既存フロー(`/work`)で進める前提で Lead 自身は実装せず」に改める。`scripts/sync-skills.sh` で `.agents/` を再生成し、`templates/base/` の 2 面へ cp |
| 5 | テストのアサートが主張より弱い 2 箇所 | `internal/cli/org_test.go`(fallback warning)、`internal/org/prompts_test.go`(fan-out) | (a) `strings.Contains` → `strings.Count(out, wantWarn) == 1`。(b) `## 座席内 fan-out` から次の `## ` 見出しまでを切り出すヘルパーを足し、`max_seats` / `lead` / `送ることは絶対に` をその節内で検査する |
| 6 | `lead.md` の委譲文で常体・敬体が混在 | `internal/org/prompts/lead.md` | 「…qa 座席へ委譲する。」→「…qa 座席へ委譲してください。」に揃える(段落は敬体) |
| 7 | `OrgModelPoolEntry` doc の「added in a later slice」 | `internal/config/config.go` | 「`ralph doctor` の codex slug check(`checkCodexModelSlugs`)が warn する」に書き換え |
| 8 | `pruneRetiredConditions` の `PendingAlerts` ループが `Escalated` を冗長に削除 | `internal/org/watch.go` | `PendingAlerts` ループ内の `delete(status.Escalated, alertID)` を削除。直後の `Escalated` ループが同条件で削除するため挙動不変 |
| 9 | `prompts_test.go` の fixture が旧 3 エントリの model_pool 文字列をハードコード | `internal/org/prompts_test.go` | `vars.Envelope = EnvelopeSummary(config.Default().Org)` に置き換え(既定プールの変更に自動追従) |
| 10 | `driver_pool` フィルタ回帰テスト 2 件の命名・アサート精度 | `internal/config/config_test.go` | `TestLoad_DriverPoolOnlyOverride_Codex` を `…_Codex_KeepsOnlyCodexDefaultEntriesInOrder` に改名し doc comment を付す。`…_RolesReferencingFilteredModelErrors` に `not present in [org].model_pool` と `"gpt-5.5"` の両方を含むアサートを追加 |

加えて `docs/tech-debt/README.md` の該当行を `~~` でクローズし、各項目の処置(2 は確認済み・修正不要)と issue #154 を記す。

## Non-goals

- org runtime の挙動変更(#153 `StopParams.Reason`、#155 codex permission、#156 slug 観測は別 issue)
- `DefaultModelForDriver` の削除や API 変更(production caller 不在は既知だが、本 issue の範囲外)
- 4 雛形(`internal/org/prompts/*.md`)の内容見直し(6 の一文のみ)
- `docs/tech-debt/README.md` の他行の整理

## Assumptions

- 4 つの skill ミラーは `scripts/sync-skills.sh`(`.claude/` → `.agents/`)と手動 cp(`templates/base/` 2 面)で揃え、`check-skill-sync.sh` / `check-sync.sh` が一致を保証する
- codex の `CODEX_HOME` 判定は `std::env::var("CODEX_HOME").ok().filter(|val| !val.is_empty())`(rust-v0.149.1 と rust-v0.154.0 で同一を 2026-09-17 に確認。タグのみを根拠にし、動く `main` は引かない)。doctor はこの判定に合わせ、値を trim も正規化もしない
- `os.UserHomeDir` は unix で `$HOME` 空のときエラーを返すので、`t.Setenv("HOME", "")` + `t.Setenv("CODEX_HOME", "")` で 1 のエラー経路を決定的に再現できる
- `internal/org` パッケージ内テストから `internal/config` を import しても循環しない(`envelope_summary.go` が既に import 済み)

## Affected areas

- `internal/cli/doctor_codex_models.go`, `internal/cli/doctor_org_test.go`, `internal/cli/org_test.go`
- `internal/org/prompts.go`, `internal/org/prompts_test.go`, `internal/org/prompts/lead.md`, `internal/org/watch.go`, `internal/org/spawn.go`(Slice D で追加: 雛形一覧コメント 1 行)
- `internal/config/config.go`, `internal/config/config_test.go`
- `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md`
- `docs/tech-debt/README.md`

## Design decisions

Critical forks: None(全項目が数行の可逆な編集で、既定で解ける)。

既定として採った選択:

- 1 は `bool` ではなく `error` を返す。呼び出し元が info 文言にエラー内容を含められ、テストで原因を検査できる
- 1 で `CODEX_HOME` を trim しない。tech-debt 行は「untrimmed で join している」を欠点として挙げるが、codex 自身が literal に使う以上、trim すると doctor が codex と別のディレクトリを見て false pass / false warn になる。空白のみ判定も codex と同じ厳密な空文字判定に揃える(Codex plan advisory の MEDIUM-1 を codex ソースで裏取りして採用)
- 9 はリテラルを 9 エントリに更新するのではなく `EnvelopeSummary(config.Default().Org)` を使う。`defaults_sync_test.go` が既定を 3 面ロックしているので、テストが既定の変更に自動で追従する方が陳腐化しない
- 3 は issue の指摘(implementer.md 欠落)に加え、同じコメントの「PR③」前方参照も中立化する。同一箇所の同種の陳腐化で、別 PR に分ける理由がない

## Acceptance criteria

- [x] AC-1: `codexModelsCachePath` は空でない `CODEX_HOME` を literal に使い(`grep -n 'TrimSpace' internal/cli/doctor_codex_models.go` が空)、`CODEX_HOME` 空かつ home 解決失敗時にエラーを返す。`checkCodexModelSlugs` はその場合 status `info` で失敗理由を Detail に含める。テスト: (a) 末尾空白付き名のディレクトリ(`<tmp>/codex `)に全 slug 入り cache、trim 後名(`<tmp>/codex`)に slug 欠落 cache を置き、`CODEX_HOME=<tmp>/codex ` で pass になる(literal 側を読んだ証拠)、(b) `HOME=""` + `CODEX_HOME=""` で info になる、の 2 件が追加され pass
- [x] AC-2: `DefaultModelForDriver` の doc が現状(production caller なし、CLI は `resolveModelOrWarn` 経由で role-aware 版)と一致することを確認し、修正不要を Deviation notes に記録
- [x] AC-3: `grep -n 'PR③' internal/org/prompts.go` が空。`PlanPath` doc が 4 雛形すべてを非消費者として挙げる。`grep -rn 'PLAN_PATH' internal/org/prompts/` は引き続き空
- [x] AC-4: `/org` SKILL.md の Leaded 行が「Lead 自身は実装しない」旨で `lead.md` と整合し、`grep -rn 'Lead 自身か既存フロー' .claude .agents templates/base` が空。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が pass
- [x] AC-5: `org_test.go` の fallback warning 検査が `strings.Count(...) != 1` で失敗する形になっている。`prompts_test.go` の fan-out 検査が節スコープのヘルパー経由で、全文検査(`strings.Contains(text, "max_seats")` 等)を残さない
- [x] AC-6: `lead.md` ミッション段落の文末がすべて敬体。`TestRenderRolePrompt_Lead_DelegatesToImplementer` pass
- [x] AC-7: `grep -n 'later slice' internal/config/config.go` が空で、doc が `checkCodexModelSlugs` を名指しする
- [x] AC-8: `pruneRetiredConditions` の `PendingAlerts` ループに `Escalated` への delete がなく、`TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` pass
- [x] AC-9: `grep -n 'claude/opus, claude/sonnet, claude/haiku' internal/org/prompts_test.go` が空で、lead fixture が `EnvelopeSummary(config.Default().Org)` を使う
- [x] AC-10: `grep -n 'func TestLoad_DriverPoolOnlyOverride_Codex(' internal/config/config_test.go` が空(改名済み)で新名に doc comment がある。roles 参照テストが `not present in [org].model_pool` と `gpt-5.5` の両方を検査する
- [x] AC-11: `docs/tech-debt/README.md` の該当行が `~~` でクローズされ、issue #154 と項目別処置を記す。`./scripts/run-verify.sh` と `go test ./internal/...` が green

## Implementation outline

1. Slice A(Go ソースのコメント・コード): 項目 1, 3, 7, 8 と項目 1 のテスト 2 件。`go test ./internal/cli/... ./internal/org/... ./internal/config/...` → commit `fix: harden codexModelsCachePath and sweep stale org doc comments`
2. Slice B(テスト精度): 項目 5a, 5b, 9, 10。同テスト実行 → commit `test: tighten org/config assertions and refresh lead envelope fixture`
3. Slice C(文書・ミラー・tech-debt): 項目 4(4 面)、項目 6、tech-debt 行クローズ。`./scripts/sync-skills.sh` → `templates/base/` 2 面へ cp → `./scripts/check-skill-sync.sh` / `./scripts/check-sync.sh` → `go test ./internal/org/...`(lead.md 変更の確認)→ commit `docs: align /org Leaded row and lead.md with implementer-first delegation; close #154 tech-debt row`
4. 最後に `./scripts/run-verify.sh` を通し、post-implementation pipeline へ

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet / lint)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC-1〜AC-11 の grep ゲートをそのまま実行して空/非空を確認する
- Documentation drift to check: `.claude/skills/org/SKILL.md` の 4 ミラー一致、`docs/tech-debt/README.md` の行クローズ、`internal/org/prompts/lead.md` を参照するテストの整合
- Evidence to capture: grep ゲートの出力、check-sync 系スクリプトの exit code

## Test plan

- Unit tests: `internal/cli`(`TestCheckCodexModelSlugs_*` 既存 5 件 + 新規 2 件)、`internal/org`(prompts / watch)、`internal/config`(driver_pool 系 5 件)
- Integration tests: なし(CLI 経路は `TestOrgSpawn_ModelFlagOmitted_*` の既存 dry-run で足りる)
- Regression tests: `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation`(項目 8)、`TestRenderRolePrompt_Lead_*`(項目 6, 9)、`TestRolePrompts_SeatTemplatesContainFanOutSection`(項目 5b)
- Edge cases: `CODEX_HOME="<dir> "`(末尾空白)は codex と同じくその literal パスを読む(AC-1(a))。`CODEX_HOME="  "`(空白のみ)は codex では「存在しないパス」で fatal になる値なので、doctor も未設定扱いにせず literal に join し「cache not found at …」の info で終わる(既存 `CacheMissing_InfoNamesPath` の経路)。`HOME=""` かつ `CODEX_HOME=""` は info で終わり panic しない。末尾空白ディレクトリのテストは Windows では作れないため `runtime.GOOS == "windows"` で skip
- Evidence to capture: `go test ./internal/... -count=1` の出力(`docs/reports/test-*.md`)

## Risks and mitigations

- 項目 9 でテストが `config.Default()` に依存する → 既定プール変更時にテストが追従するのは意図した結合。`defaults_sync_test.go` が別途 3 面一致を守る
- 項目 4 の 4 面ミラーで 1 面だけ取り残す → `check-skill-sync.sh`(`.claude` vs `.agents`)と `check-sync.sh`(root vs `templates/base`)の両方を CI 前に回す
- 項目 1 の signature 変更 → 呼び出し元は `checkCodexModelSlugs` の 1 箇所のみ(テストは `checkCodexModelSlugs` 経由)。コンパイルで検出される
- 項目 1 の空白のみ `CODEX_HOME` の扱い変更(未設定扱い → literal)→ そのような値は codex 側で fatal になるため実運用上は存在しない。doctor は info で終わり fail しない
- 項目 8 の削除で Escalated が残る → 直後のループが同条件で削除。既存回帰テストが 3 マップすべての prune を検査済み

## Rollout or rollback notes

単一 PR。実行時挙動の変化は `ralph doctor` の (a) home 解決失敗時の info 文言、(b) 空白のみ `CODEX_HOME` を未設定扱いしなくなる点、の 2 つで、いずれも best-effort な info 経路。revert は PR 単位で安全。下流プロジェクトへは `/org` SKILL.md の文言変更が `ralph upgrade` 経由で配布される(managed core ファイル)。

## Open questions

なし。

## Deviation notes

- 2026-09-17 plan: Codex plan advisory(codex-cli 0.154.0)が MEDIUM 1 件(`CODEX_HOME` trim が codex リゾルバと乖離)を報告。codex ソースのタグ 2 ref(+ 当日の main)で裏取りし、項目 1 の方針を「trim しない・厳密空判定」に変更。Design decisions / AC-1 / Test plan / Risks / Rollout を更新済み
- 項目 2(`DefaultModelForDriver` doc)は 463e943 で修正済みを確認。本 PR では触らない
- 2026-09-17 work: `./scripts/branch-name.sh from-plan` は issue 番号付きの `chore/154/org-envelope-low-findings` を返すが、`/plan` が作成した worktree state(`plan-org-envelope-low-findings`)は `chore/org-envelope-low-findings` で登録済み。`/work` 手順 2d(既存 state を resume)に従い state 側のブランチを維持
- 2026-09-17 work: Slice A = 314b89f(項目 1・3・7・8 + テスト 2 件)、Slice B = 746c70d(項目 5・9・10)。いずれも implementer 委譲、逸脱なし。Slice B の implementer が `docs/reports/self-review-2026-09-16-org-implementer-seat-envelope.md` に旧テスト名 `TestLoad_DriverPoolOnlyOverride_Codex` が残ると報告 — 過去レポートは当時の名称を記録した履歴として据え置く(tech-debt 行は Slice C でクローズ)
- 2026-09-17 work: Slice C = 3a9362c(項目 4・6、tech-debt 行クローズ、4 面ミラー同期)。implementer が tech-debt 行編集中に句「since every file is already open.」を一度消し、自己検出して復元。orchestrator 側でパイプ数(6→6)と他 3 セルのハッシュ一致を確認済み。全 11 AC 達成、`./scripts/run-verify.sh` green(evidence: `docs/evidence/verify-2026-09-17-092738.log`、gitignored)
- 2026-09-17 self-review(cycle 1): Merge 判定、LOW 8 件(L1〜L8)、CRITICAL/HIGH/MEDIUM なし。本 PR の趣旨(先送り LOW の一括解消)に照らし、8 件すべてを同 cycle 内で修正する(Slice D)。うち L1 は `internal/org/spawn.go` の雛形一覧コメントに `implementer.md` が欠ける同種欠陥で、plan の Affected areas 外だがスコープを 1 コメント行分だけ広げる。L4 は plan の Edge cases にあった空白のみ `CODEX_HOME` を実際にアサートするテストを追加。L7 に従い、codex ソースの根拠から動く `main` ref を外す(本 plan の記述も同時に修正)
- 2026-09-17 work: Slice D = ff30ee2(L1〜L8、9 ファイル)。implementer 委譲、逸脱なし(L1 のコメント再折返しが段落 6 行に及んだのみ)。orchestrator 側で tech-debt 行のパイプ数 6 と Impact/Why deferred セルのハッシュ一致、cli/org テスト pass を確認。`./scripts/run-verify.sh` green(evidence: `docs/evidence/verify-2026-09-17-095011.log`)
- 2026-09-17 self-review 再検証: L1〜L8 全件解消。修正コミット ff30ee2 から新規 LOW 2 件(N1: `markdownSection` が空セクションで次セクション本文を返す(現雛形では到達不能)、N2: 空白のみ `CODEX_HOME` テストに Windows skip なし)。2 ファイル・数行のため inline 例外(`.claude/rules/ralph/subagent-policy.md` の trivial edit)として orchestrator が修正、Slice E = b677a95。N1 には normal / empty / EOF / `###` 降格 / 行中言及の 5 ケースを固定する `TestMarkdownSection_AnchorsHeaderAndBoundsBody` を追加。`gofmt -w` 後 `./scripts/run-verify.sh` green(evidence: `docs/evidence/verify-2026-09-17-100515.log`)
- 2026-09-17 self-review N1/N2 再検証: 両件解消。Slice E から新規 LOW 1 件(N3: テーブルテスト第 5 ケース「行中言及」がフィクスチャに行中言及を含まず、どの実装でも通る)。フィクスチャに `see ## D for details` を入れ `## D` を not-found 期待にして判別可能化(unanchored 実装では found=true になることを確認)。inline、`./scripts/run-verify.sh` green(evidence: `docs/evidence/verify-2026-09-17-101028.log`)。これで self-review の fix-and-revalidate は打ち切り、以降の LOW は Follow-ups 記載に留める

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#157)
