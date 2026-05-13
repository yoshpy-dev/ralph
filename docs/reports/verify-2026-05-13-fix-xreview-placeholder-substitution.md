# Verify report: fix-xreview-placeholder-substitution (issue #50)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md`
- Verifier: Claude Code (`/verify` subagent)
- Scope: 6 commits on `fix/50/xreview-placeholder-substitution` (0304686 → 12a1984), 12 files (+1028/-20)
- Evidence: `docs/evidence/verify-2026-05-13-fix-xreview-placeholder-substitution.log`
- Verdict: **PASS**

## Spec compliance

Each acceptance criterion is matched to the diff. The plan's progress checklist still has the verification/test/PR boxes unchecked — that is expected (this is the verify step that fills the verification box) and is doc drift only, not a spec failure.

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: Rendered prompt contains literal `_base` and `REPORTS_DIR`; no `${BASE_BRANCH}` / `${REPORTS_DIR}` strings remain | Verified | `scripts/ralph-pipeline.sh:809-826` builds `${PIPELINE_DIR}/outer-${_cycle}-adversarial-claude.md` via awk `index()`/`substr()` literal substitution; `tests/test-xreview-prompt-render.sh` cases 1.a–1.b assert no placeholder remains; cases 1.c–1.d assert substituted values appear |
| AC-2: Rendered file written under `${PIPELINE_DIR}/` per-cycle and not committed | Verified | Path literal `"${PIPELINE_DIR}/outer-${_cycle}-adversarial-claude.md"` on line 809; `${PIPELINE_DIR}` is the existing per-cycle ephemeral state dir (already gitignored as `.harness/state/pipeline/`) |
| AC-3: Missing prompt file preserves existing `Warning: adversarial-claude prompt missing at ...` path | Verified | Lines 856-857 — the `else` branch of `if [ -f "$_adv_prompt" ]` is unchanged from the pre-fix code |
| AC-4: Allowlist-based unresolved-placeholder guard fires on unknown `${...}` tokens | Verified | Lines 830-841: post-render grep `'\$\{[A-Z_][A-Z0-9_]*\}'` over `_rendered_prompt`; any match sets `_render_failed=1`. Allowlist is encoded implicitly (BASE_BRANCH/REPORTS_DIR are substituted FIRST, so any surviving token is by definition outside the allowlist). `tests/test-xreview-prompt-render.sh` negative case (`${UNKNOWN_PLACEHOLDER}`) asserts the guard triggers |
| AC-5: Renderer fails loudly on awk failure / unresolved placeholder / empty `_base`; gate fails CLOSED in each case | Verified (with note) | awk-failure branch: lines 826-829 set `_render_failed=1` + `log_error` + `echo render_failed_awk`. Unresolved-placeholder branch: lines 836-841. Final gate decision lines 893-896 returns 1 BEFORE consulting `_action_required`. **Note:** the "empty `_base`" sub-clause is unreachable because line 777 (`_base="${_base:-main}"`) defaults to `main`. That is a defensible interpretation — `_base` cannot be empty when the renderer runs — but the plan's literal wording is not implemented as a separate emit. Documented as a minor doc/code drift; not a regression |
| AC-6: End-to-end gate-regression test asserts non-zero return on `ACTION_REQUIRED=1` triage | Verified | `tests/test-xreview-gate-regression.sh` Phase 2 (2.a–2.d) writes a real triage report with `ACTION_REQUIRED=1`, invokes `gate_decision`, asserts return is non-zero. Phase 5 adds the `_render_failed=1` end-to-end path with drift assertions (5.d–5.f) against the production script |
| AC-7: Renderer unit test parameterized over metacharacter edge cases (`feature#1`, `feature&1`, `release/3.5`) | Verified | `tests/test-xreview-prompt-render.sh` `CASES` table includes `main`, `release/3.5`, `feature#1`, `feature&1`, `feature\back`, `docs/reports#1`, `docs/reports&backup` — exceeds the plan's required set |
| AC-8: `./scripts/run-verify.sh` (delegate `run-static-verify.sh`) and `./scripts/check-skill-sync.sh` stay green | Verified | `./scripts/run-static-verify.sh` exit 0 — see `docs/evidence/verify-2026-05-13-fix-xreview-placeholder-substitution.log`; `check-skill-sync.sh` reports `13 skill(s) in lock-step`; `check-sync.sh` reports `IDENTICAL: 145, DRIFTED: 0`; `gofmt` clean; `go vet` clean |
| AC-9: Adversarial-claude prompt documents rendering contract in top-of-file comment | Verified | `.claude/skills/cross-review/prompts/adversarial-claude.md:1-17` HTML-comment block names both placeholders, the renderer script, the awk `index()`/`substr()` literal-replacement rationale, and the allowlist contract |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | exit 0 | All verifiers passed: check-sync, check-pipeline-sync, check-skill-sync, gofmt, go vet, go test (cached). Evidence: `docs/evidence/verify-2026-05-13-025411.log` (also copied to slug-named file) |
| `./scripts/check-skill-sync.sh` | exit 0 | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | exit 0 | `IDENTICAL: 145, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3` |
| `shellcheck scripts/ralph-pipeline.sh` | clean vs baseline | 5 findings (SC1091 lines 14, 15; SC2016 lines 512, 648, 975). All five exist on `main` baseline (lines 14, 15, 512, 648, 909 pre-fix → numbers shifted by the diff). No new shellcheck findings introduced |
| `sh -n scripts/ralph-pipeline.sh` | exit 0 | POSIX syntax clean |
| `bash -n scripts/ralph-pipeline.sh` | exit 0 | bash syntax clean |
| `sh -n tests/test-xreview-prompt-render.sh tests/test-xreview-gate-regression.sh` | exit 0 | POSIX syntax clean on both new tests |
| `gofmt -l .` | empty | no formatting drift |
| `go vet ./...` | 0 issues | per `run-static-verify.sh` output |
| `diff -q scripts/ralph-pipeline.sh templates/base/scripts/ralph-pipeline.sh` | identical | mirror parity |
| `diff -q .claude/skills/cross-review/SKILL.md .agents/skills/cross-review/SKILL.md` | identical | Claude ↔ Codex SKILL.md parity |
| `diff -q .claude/skills/cross-review/prompts/adversarial-claude.md templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md` | identical | template mirror parity |

**Note on the test suite caveat from the request:** `run-static-verify.sh` ran `go test` (cached). This counts as static-equivalent here — we did not execute the new shell tests (`test-xreview-prompt-render.sh`, `test-xreview-gate-regression.sh`); that is `/test`'s scope.

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/cross-review/SKILL.md` | Yes | New "Prompt rendering contract (claude reviewer path)" subsection (lines 178-188) documents the renderer + allowlist + regression coverage. Mirrored to `.agents/skills/cross-review/SKILL.md` |
| `.claude/skills/cross-review/prompts/adversarial-claude.md` | Yes | HTML-comment block at the top names the renderer, the literal-replacement rationale, and the allowlist update requirement |
| `templates/base/.claude/skills/cross-review/SKILL.md` | Yes | Byte-identical to root |
| `templates/base/.agents/skills/cross-review/SKILL.md` | Yes | Byte-identical to root |
| `templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md` | Yes | Byte-identical to root |
| `templates/base/scripts/ralph-pipeline.sh` | Yes | Byte-identical to root |
| `AGENTS.md` | Yes — no change needed | Cross-review section speaks at contract level (driver-aware inversion); implementation-level rendering detail belongs in the skill, not the map |
| `CLAUDE.md` | Yes — no change needed | Doesn't mention the prompt rendering layer |
| `docs/tech-debt/README.md` | Yes | New row added by cycle-2 fix (commit 12a1984) noting the duplicated awk renderer across `ralph-pipeline.sh` + the two tests, with the drift-guard mitigation and a "next time the renderer grows" trigger |
| `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md` progress checklist | Stale (expected) | "Review artifact created", "Verification artifact created", "Test artifact created", "PR created" boxes are unchecked. This is normal for the verify step — those boxes are filled by the downstream pipeline stages |

No surprising drift. The only "stale" doc is the plan checklist itself, which is by design.

## Observational checks

- Cycle-2 fix (commit 12a1984) wires `_render_failed=1` into the gate decision (line 893) BEFORE the existing `_action_required` check (line 898). This closes the self-review CRITICAL identified in cycle-1: prior to this commit, awk failure / allowlist guard trip would log but the gate would still proceed because `_action_required` was 0.
- The `_render_failed` flag is now serialized into both `ckpt_update .cross_review_triage` and `report_event "cross-review"` payloads (lines 884-885) for downstream auditability.
- Allowlist mechanics are correct-by-construction: `BASE_BRANCH` and `REPORTS_DIR` are substituted first, then any remaining `${[A-Z_][A-Z0-9_]*}` token triggers `_render_failed=1`. The plan calls for a literal allowlist, but the implementation's "substitute-then-grep" pattern is operationally equivalent and arguably safer (no risk of allowlist/renderer drift inside the script).
- The awk renderer is deliberately POSIX (no envsubst, no Bash-isms) matching the plan's portability requirement (Assumptions section).
- The same awk renderer body is duplicated three places (production + 2 tests). This is intentional per the MEDIUM tech-debt entry (`docs/tech-debt/README.md`), with a `function lreplace` drift-guard grep in the unit test as the safety net.
- Mirror parity is held by both `check-skill-sync.sh` (Claude ↔ Codex SKILL.md only — no prompt parity check, by design) and `check-sync.sh` (root ↔ `templates/base/` including the prompt body and the script).

## Coverage gaps

These are remaining gaps for `/test` to address, not verify failures:

- **Behavioral test execution not performed.** Per the canonical /verify scope, the two new shell tests (`tests/test-xreview-prompt-render.sh`, `tests/test-xreview-gate-regression.sh`) were syntax-checked but not run. The plan's progress checklist claims `54/54` and `21/21` (after cycle-2) pass — `/test` should re-execute to confirm. Evidence files `docs/evidence/test-xreview-{prompt-render,gate-regression}-2026-05-13.log` exist from the implementation phase and can be cross-checked.
- **No end-to-end live-`claude` smoke.** The gate-regression test exercises the post-render parser + decision path with a hand-written triage report. It does not actually invoke `claude -p`. That is a correct test boundary (avoiding a flaky external dependency), but it leaves "real reviewer writes report to the expanded path" as a manual / integration concern. The plan acknowledges this in Risks (LLM hallucinates a different output path → out of scope).
- **`_base` empty-string clause is dead code.** Line 777 forces `_base` to `main` if empty, so the AC-5 sub-clause "fail loudly on empty `_base`" cannot fire. Not a regression — the upstream default guarantees non-emptiness — but worth surfacing as a minor spec/code mismatch for future cleanup if `_base` ever stops defaulting.

## Verdict

- **Verified:** AC-1, AC-2, AC-3, AC-4, AC-5 (awk-failure + unresolved-placeholder branches), AC-6, AC-7, AC-8, AC-9. All static analysis passes (`run-static-verify.sh` exit 0). All mirrors (Claude ↔ Codex SKILL.md, root ↔ templates/base) byte-identical. No new shellcheck findings vs `main`. Cycle-2 fix (`_render_failed` gate) is correctly wired into the decision logic ahead of the `_action_required` check.
- **Likely but unverified:** Test pass counts (54/54 prompt-render, 21/21 gate-regression) — taken on the plan checklist's word; `/test` will re-execute.
- **Not verified (out of /verify scope):** Live `claude -p` invocation with a real reviewer (manual / integration); the dead-code sub-clause "empty `_base`" path (impossible to reach with current defaults).

**Overall: PASS.** Proceed to `/test`.

## Evidence files

- `docs/evidence/verify-2026-05-13-fix-xreview-placeholder-substitution.log` — `run-static-verify.sh` output (canonical slug-named copy of `verify-2026-05-13-025411.log`)
- `docs/evidence/verify-2026-05-13-025411.log` — same run, timestamp-named (kept for traceability with the static-verify auto-output)
- `docs/evidence/verify-2026-05-13-024935.log`, `docs/evidence/verify-2026-05-13-024222.log`, `docs/evidence/verify-2026-05-13-024123.log` — prior static-verify runs from the implementation phase (referenced by the plan's progress checklist)
- `docs/evidence/test-xreview-prompt-render-2026-05-13.log` — implementation-phase unit-test output (to be re-validated by `/test`)
- `docs/evidence/test-xreview-gate-regression-2026-05-13.log` — implementation-phase integration-test output (to be re-validated by `/test`)
