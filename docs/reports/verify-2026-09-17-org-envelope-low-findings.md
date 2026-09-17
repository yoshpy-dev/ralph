# Verify report: org-envelope-low-findings

- Date: 2026-09-17
- Plan: `docs/plans/active/2026-09-17-org-envelope-low-findings.md` (issue #154)
- Verifier: `verifier` subagent (Claude Code, standard flow)
- Diff scope: `git diff 4905c34...HEAD` on `chore/org-envelope-low-findings` (HEAD `057bfd7`), 18 files, +464/-54.
- Scope note: spec compliance (AC-1..AC-11 grep/doc gates) and static analysis only. No behavioral test suite was run — that is `/test`'s job. `git status --porcelain` confirmed a clean working tree before and after this pass; nothing here reflects uncommitted changes.

## Overall verdict: Pass

All 11 acceptance criteria confirmed with direct evidence. Static analysis is clean. Both skill-mirror sync gates pass. One pre-existing, self-documented doc-drift note is flagged below as informational (not a fail).

## Static analysis

`HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh` (via `./scripts/run-static-verify.sh`, changed-language scope = golang):

- `gofmt -l .` → `gofmt: ok`
- `go vet ./...` → silent (pass)
- `golangci-lint run ./...` → `0 issues.`
- `staticcheck ./...` → silent (pass; `command -v staticcheck` confirms it ran)
- `scripts/check-sync.sh` → `PASS: all files in sync.` (`DRIFTED: 0`)
- `scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`
- `scripts/check-pipeline-sync.sh`, `scripts/check-template-purity.sh` → both OK
- Overall: `==> All verifiers passed.`
- Evidence: `docs/evidence/verify-2026-09-17-101642.log`

## Acceptance criteria

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 | Pass | `grep -n 'TrimSpace' internal/cli/doctor_codex_models.go` empty. `codexModelsCachePath` (`internal/cli/doctor_codex_models.go:24-38`) joins a non-empty `CODEX_HOME` literally and returns `(string, error)` when `os.UserHomeDir` fails. `checkCodexModelSlugs` (`:89-94`) catches that error and sets `Status="info"`, `Detail` naming the failure. Both required tests present and read: `TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace` (`doctor_org_test.go:516-542`, sibling-directory technique proving the literal, untrimmed path is read) and `TestCheckCodexModelSlugs_HomeUnresolvable_Info` (`:583-604`, `HOME=""` + `CODEX_HOME=""` → `info`). A third test, `TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally` (`:549-571`), covers the whitespace-only edge case named in the plan's Edge cases — beyond the AC's literal (a)/(b) pair, additive. |
| AC-2 | Pass | No change required, and the "no change" claim is itself verified: `grep -rn 'DefaultModelForDriver(' internal --include='*.go' \| grep -v _test` returns only the definition (`internal/org/envelope_summary.go:51`), whose doc (`:49-50`) states "An error is returned when no model_pool entry matches driver" with no stale caller claim. Cross-checked the plan's citation of commit `463e943` ("docs: fix stale DefaultModelForDriver comments and spec revision markers", 2026-09-16) via `git show` — it is a real ancestor commit and its diff matches the described fix (`internal/org/envelope_summary.go`, `internal/cli/org.go`, `internal/cli/org_test.go`). Recorded in Deviation notes. |
| AC-3 | Pass | `grep -n 'PR③' internal/org/prompts.go` empty. `PlanPath`'s doc (`internal/org/prompts.go:35-41`) now enumerates all four templates ("lead.md, implementer.md, reviewer.md, qa.md") as non-consumers, and the replacer comment (`:80-82`) is neutralized ("stays so a template that re-adds {{PLAN_PATH}} can never ship the literal placeholder"). `grep -rn 'PLAN_PATH' internal/org/prompts/` empty. |
| AC-4 | Pass | Leaded row (`.claude/skills/org/SKILL.md:83`) now reads "実装は完了済みか既存フロー(`/work`)で進める前提で、Lead 自身は実装しない" — consistent with `lead.md`'s implementer-first delegation, no residual cross-reference to compare against the Solo row (self-review L6 fix). `grep -rn 'Lead 自身か既存フロー' .claude .agents templates/base` empty. All four mirrors are byte-identical (`md5` = `9ce9b00a952b1ff1120fa270e6082204` for all of `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md`). `./scripts/check-skill-sync.sh` → `[ok]`; `./scripts/check-sync.sh` → `PASS`, `DRIFTED: 0`. |
| AC-5 | Pass | (a) `internal/cli/org_test.go:672-675`: `if n := strings.Count(out, wantWarn); n != 1` — no longer a bare `strings.Contains`. (b) `internal/org/prompts_test.go` adds a section-scoped helper `markdownSection` (`:17-38`) anchored to whole lines (`"\n"+header+"\n"`), and `TestRolePrompts_SeatTemplatesContainFanOutSection` (`:196-220`) asserts `max_seats` / `lead` / `送ることは絶対に` against `section` (the sliced substring), not the full rendered `text`. `grep -n 'strings.Contains' internal/org/prompts_test.go` confirms lines 212/215 operate on `section`. |
| AC-6 | Pass | `internal/org/prompts/lead.md` mission paragraph (lines 7-10) ends every sentence in polite form: 「…座席です。」「…振る舞ってください。」「…委譲してください。」「…限定します。」 — no plain-form (常体) sentence remains. `TestRenderRolePrompt_Lead_DelegatesToImplementer` (`prompts_test.go:222-238`) asserts `implementer` is present and `budget` is absent; test source confirmed present and matches the AC. |
| AC-7 | Pass | `grep -n 'later slice' internal/config/config.go` empty. `OrgModelPoolEntry`'s doc (`internal/config/config.go:97-103`) now names `checkCodexModelSlugs` (`internal/cli/doctor_codex_models.go`) as the consumer that warns on a stale slug, replacing the "added in a later slice" placeholder. |
| AC-8 | Pass | `pruneRetiredConditions`'s `PendingAlerts` loop (`internal/org/watch.go:562-566`) deletes only from `status.PendingAlerts`; the redundant `delete(status.Escalated, alertID)` inside that loop is gone. The subsequent, distinct loop (`:568-572`) still owns pruning `status.Escalated` under the identical predicate. `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` present at `internal/org/watch_test.go:1689`. |
| AC-9 | Pass | `grep -n 'claude/opus, claude/sonnet, claude/haiku' internal/org/prompts_test.go` empty. `internal/org/prompts_test.go:151`: `vars.Envelope = EnvelopeSummary(config.Default().Org)` — the lead fixture now derives from the live default pool rather than a hardcoded 3-entry string. |
| AC-10 | Pass | `grep -n 'func TestLoad_DriverPoolOnlyOverride_Codex(' internal/config/config_test.go` empty (old name gone). New name `TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder` (`config_test.go:730`) carries a doc comment (`:725-729`) stating what it asserts (exact codex-only subset, declared order). `TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors` (`:812-834`) asserts `want := `[org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool`` — both the `not present in [org].model_pool` phrase and the literal `gpt-5.5` are present in the same assertion (`:831-833`), matching the error format at `internal/config/config.go:299` byte-for-byte (`%q` quoting included). |
| AC-11 | Pass (verify-scope portion) | `docs/tech-debt/README.md:130`: the row is struck (`~~...~~`) and the debt-item cell carries `(RESOLVED 2026-09-17 in chore/org-envelope-low-findings)` immediately after the closing `~~`, following the closure-marker convention used by the other 43 struck rows (self-review L3 fix). Issue #154 and a per-item (1)-(10) disposition are recorded in the Trigger cell's "Closed 2026-09-17 … (issue #154): …" text. `./scripts/run-verify.sh` (static mode) is green — see Static analysis above. The AC's `go test ./internal/...` clause is explicitly out of scope for `/verify` per this task's instructions (that is `/test`'s job) and is not evaluated here. |

## Documentation drift

- **No unresolved drift found.** The three items the task flagged for extra scrutiny (AC-1's untrimmed-`CODEX_HOME` design rationale, AC-4's four-mirror parity, AC-11's tech-debt closure convention) all check out against the plan's Design decisions and the self-review's fix-and-revalidate rounds (L1–L8, N1–N3).
- **Informational, not a fail — forward-referencing citation in the tech-debt row.** `docs/tech-debt/README.md:130`'s Related plan/report cell cites `docs/plans/archive/2026-09-17-org-envelope-low-findings.md (closing plan; archived by \`/pr\`)`. That path does not exist yet — the plan is still at `docs/plans/active/2026-09-17-org-envelope-low-findings.md` at this pipeline stage, confirmed by `ls`. This is a deliberate, self-labeled forward reference (the self-review's "Residue noted, not raised as a finding" section calls out the identical pattern for a *different*, pre-existing dangling citation and confirms this new one is self-describing rather than silently broken). It will resolve automatically once `/pr` archives the plan. No action needed before merge.
- **Plan Progress checklist accurately reflects pipeline stage.** "Verification artifact created" and "Test artifact created" are unchecked, which is correct — this report is the verification artifact being created now, and `/test` has not yet run.
- Swept for the same defect class the plan's L1 finding named (stale template enumeration missing `implementer.md`): `grep -rn 'reviewer\.md' --include='*.go' internal/ | grep -v _test` returns only the two already-fixed sites (`prompts.go:36`, `spawn.go:608`), both listing all four templates. No other site found.

## Known gaps / out of scope for this pass

- Behavioral tests (`go test ./internal/...`) were not run — `/test` owns this, including the AC-11 test clause.
- Codex's `find_codex_home` source (`codex-rs/utils/home-dir/src/lib.rs`) is external to this repo and was not independently re-fetched; the plan's citation (rust-v0.149.1, rust-v0.154.0, checked 2026-09-17) is taken as given, consistent with the self-review's own stated limitation on this point.
