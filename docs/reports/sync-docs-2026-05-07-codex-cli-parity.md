# Sync-docs report: Codex CLI standard-flow parity

- Date: 2026-05-07
- Plan: `docs/plans/active/2026-05-07-codex-cli-parity.md`
- Spec: `docs/specs/2026-05-07-codex-cli-parity.md`
- Branch: `feat/codex-cli-parity`
- Maintainer: `doc-maintainer` subagent (Claude Opus 4.7, 1M context)
- Scope: documentation drift after the Codex parity implementation (Slices 1-7 + self-review HIGH fix `82679f1`). Loop driver work is OUT OF SCOPE — tracked in yoshpy-dev/ralph#44.

## Method

Walked the documentation surface that the parent flow flagged plus everything else that could plausibly mention the renamed skill, the new Codex side trees, or the new drift gate. For each surface, compared the on-disk doc against the actual implementation (skill bodies, scripts, templates) and updated only the lines that had drifted. No new content was invented.

Surfaces inspected:

- Root: `AGENTS.md`, `CLAUDE.md`, `README.md`
- Architecture: `docs/architecture/repo-map.md`
- Quality: `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md`
- Rules: `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`
- Skills: `.claude/skills/cross-review/SKILL.md`, `.claude/skills/audit-harness/SKILL.md`
- Recipes: `docs/recipes/codex-setup.md`
- Tech debt: `docs/tech-debt/README.md`
- Templates: `templates/base/AGENTS.md`, `templates/base/CLAUDE.md`, `templates/base/.claude/rules/post-implementation-pipeline.md`, `templates/base/.claude/rules/subagent-policy.md`, `templates/base/.codex/AGENTS.override.md`, `templates/base/.codex/README.md`

Cross-checked against the parent prompt's seven drift candidates plus a `grep -rln "codex-review\|codex-triage"` sweep over root docs.

## Findings

### Updated in this pass

| File | What drifted | Fix applied |
| --- | --- | --- |
| `AGENTS.md` (root) | Repo map listed `.claude/{rules,skills,agents,hooks}/` but said nothing about `.agents/skills/`, `templates/base/.codex/`, `docs/recipes/`, or the new `scripts/check-skill-sync.sh` drift gate. The `.claude/rules/` line did not flag that those rules are read by both CLIs. The `scripts/` line listed `ralph-config.sh` but not the Codex availability probe or the drift gate. | Added entries for `.agents/skills/`, `templates/base/.codex/`, `docs/recipes/`. Annotated `.claude/rules/` as read by both CLIs and `.claude/agents/` as Claude-only. Extended `scripts/` line with `check-skill-sync.sh` and `codex-check.sh`. |
| `CLAUDE.md` (root) | "Claude-specific directories" section was silent on the Codex equivalents that landed in this PR (`.agents/skills/` mirror, `.codex/config.toml` `[hooks]`). A reader would not see how the Claude-side surfaces relate to the new Codex side. | Annotated each of the four `.claude/*` entries with its Codex counterpart (or "no equivalent" for `.claude/agents/`) and named the drift gate. |
| `docs/architecture/repo-map.md` | The "Skills" line for `cross-review/` still said "via Codex" rather than the bidirectional pairing. There was no "Codex control plane" section despite `.agents/skills/` and `templates/base/.codex/` being shipped this PR. The "Process artifacts" list did not include `docs/recipes/`. The "Extensions / scripts" line did not include `check-skill-sync.sh`. | Switched cross-review wording to "via the other CLI (Claude → Codex; Codex → Claude)". Added a "Codex control plane" subsection naming `.agents/skills/`, `templates/base/.codex/`, and `internal/state/PipelineCheckpoint.CrossReviewTriage`. Added `docs/recipes/` to Process artifacts and `check-skill-sync.sh` to the scripts list. |

### Confirmed already in sync

| File | Why it is in sync (no edit needed) |
| --- | --- |
| `AGENTS.md` Primary loop section | Already lists step 8 as "Cross-review (auto, optional — cross-model second opinion via the other CLI: Claude → Codex; Codex → Claude)". |
| `CLAUDE.md` Default behavior bullets | Already references `/cross-review` everywhere; codex-review string is gone. |
| `README.md` Quick start, Operating loop, Features, Portability | Already shows the bidirectional pipeline diagram, both `/cross-review` and `$cross-review` invocation forms, the `.codex/` and `.agents/skills/` scaffold tree, and the "Known differences between Claude Code and Codex" table. |
| `templates/base/AGENTS.md` | Already restructured to be the dual-CLI source of truth (skill-invocation table, Codex setup checklist, repo map listing `.codex/` and `.agents/skills/`). |
| `templates/base/CLAUDE.md` | Already slim and Claude-only, with explicit pointers to `.codex/AGENTS.override.md` and `.codex/README.md`. |
| `templates/base/.codex/AGENTS.override.md` and `templates/base/.codex/README.md` | Already explain `codex trust .`, `[features] codex_hooks = true`, and the Claude→Codex permission mapping. |
| `templates/base/.codex/config.toml` | Already ships `model = "gpt-5.5"`, `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"`, `[features] codex_hooks = true`, and profile placeholders. |
| `docs/quality/definition-of-done.md` | Already references `cross-review` and adds the two parity-related checklist items (skill drift check green, both CLIs exercised when shared surfaces change). |
| `docs/quality/quality-gates.md` | Does not reference the renamed skill at all; nothing to update. |
| `.claude/rules/post-implementation-pipeline.md` (root + template) | Already includes the "CLI execution mode" table and a `cross-review` canonical order. The "Where this order is referenced" list points at `.claude/skills/cross-review/SKILL.md`. |
| `.claude/rules/subagent-policy.md` (root + template) | Already adds the "Post-implementation pipeline under Codex — sequential inline" section. The Codex triage section references `cross-review-triage-<slug>.md`. |
| `.claude/skills/cross-review/SKILL.md` | Already documents the two-step driver detection (`RALPH_PRIMARY_CLI` then auto-detect), both reviewer paths (`codex exec review` vs `claude -p` adversarial prompt), and the triage report header `Driver: <…>  Reviewer: <…>`. |
| `.claude/skills/audit-harness/SKILL.md` | Already inspects `.codex/`, `.agents/skills/`, and `scripts/check-skill-sync.sh`. The 32 KiB AGENTS.md size budget block is in place. |
| `docs/recipes/codex-setup.md` | Already covers `codex trust .`, `ralph doctor`, `$skill` mention syntax, the bidirectional cross-review pairing, the drift gate, and the half-applied upgrade recovery flow. |
| `docs/tech-debt/README.md` | Already records the two debt entries the self-review identified (Loop pipeline `_codex_*` residue + missing `codex_review_triage` → `cross_review_triage` migration). |
| Active plan progress checklist | Slices 1-7 boxes are checked. Self-review/Verify/Test boxes are still unchecked. This is normal in-flight pipeline checklist drift; `/pr` will tick the remaining boxes when the PR ships. Not edited here. |

## Cross-references verified

- `grep -rln "codex-review\|codex-triage"` over `AGENTS.md`, `CLAUDE.md`, `README.md`, `docs/architecture/repo-map.md`, and `docs/quality/`: zero hits. The remaining hits in the repo are all inside the AC-9 allowlist (the plan body, the spec body, immutable historical reports, the rename-history note in `docs/recipes/codex-setup.md`, and the explicit allowlist literal inside `scripts/check-sync.sh`).
- `AGENTS.md` size after edits: 5821 bytes (root) and 4973 bytes (template). Both well under the 32 KiB Codex cap; `audit-harness` is silent. The root grew by ~535 bytes for the four new repo-map entries, which is intentional.
- Skill inventory: `.claude/skills/` has 13 directories (anti-bottleneck, audit-harness, cross-review, loop, plan, pr, release, self-review, spec, sync-docs, test, verify, work). `.agents/skills/` has the same 13. `templates/base/.claude/skills/` and `templates/base/.agents/skills/` each have 12 (no `release/`, repo-maintainer-only). Matches the verify report.
- The tech-debt entries for this branch are present and consistent with the self-review and verify reports.

## Coverage gaps

None for this sync-docs pass. The four documentation gaps already identified by `/verify` (live Codex smoke test, `tests/upgrade_downgrade_test.go`, permission semantic equivalence, JSON checkpoint migration) are all behavioural or test gaps, not documentation gaps. They remain recorded in the walkthrough and tech-debt files.

## Verdict

Documentation surface is now consistent with the implementation. All updates were narrow corrections of stale or missing references; no documentation was rewritten or generated from scratch. The drift candidates listed in the parent prompt are all resolved or were already aligned.

Recommendation: proceed to `/cross-review`. No documentation-driven follow-up work is required for the merge.

---

## Cycle 2 (2026-05-07)

Cycle 1 cross-review (`docs/reports/cross-review-triage-2026-05-07-codex-cli-parity.md`) returned ACTION_REQUIRED, triggering a fix-and-revalidate pass. Cycle 2 commits since the cycle-1 sync-docs:

- `3abd1d7` — cross-review ACTION_REQUIRED fix: `probeBinary` now actually runs `<bin> --version` (catches stale shims), `templates/base/.codex/config.toml` ships two default-on `PostToolUse` mojibake hooks, `docs/reports/templates/cross-review-triage-report.md` becomes bidirectional (Driver/Reviewer/Triager placeholders, "Reviewer finding" column captions). Mirrored into `templates/base/`.
- `79d7a73` — cycle-2 self-review CRITICAL fix: corrected the Codex hook script paths from the bogus `./scripts/check_mojibake.sh` and `./scripts/commit-msg-guard.sh` (the latter would exit 1 on every commit because it expects a git commit-msg `$1`) to the real `./.claude/hooks/check_mojibake.sh` path; reverted the commit-msg hook entry to a comment-only stanza pointing at the correct `.git/hooks/` install path. `probeBinary` hardened with a 5s timeout, first-non-empty-line output, and a symmetric `LookPath` / `--version` error wrap. Tech-debt row for the missing-timeout case marked RESOLVED 79d7a73 with traceability.

### Cycle-2 method

Walked the seven surfaces flagged by the parent prompt. For each, compared the on-disk doc against the actual implementation (current `.codex/config.toml` template, current `probeBinary` body, current bidirectional triage template). No documentation was rewritten or generated from scratch.

### Updated in cycle 2

| File | What drifted | Fix applied |
| --- | --- | --- |
| `templates/base/.codex/README.md` (Hooks section) | Said "Add new entries to both surfaces" with no acknowledgement that the template now ships two default-on `PostToolUse` mojibake hooks. A reader following the README to a fresh `ralph init` would not know the hooks already exist, and would not learn the `commit-msg-guard.sh` wiring trap that 79d7a73 had to fix. | Rewrote the section to lead with "ships default-on with two `PostToolUse` hooks pointing at `./.claude/hooks/check_mojibake.sh`", explain that they satisfy `ralph doctor`'s hook-entry check on first init, and call out explicitly that `scripts/commit-msg-guard.sh` is a git `commit-msg` hook (consumes `$1`) — not a Codex `PostToolUse` hook — so attaching it to `^git commit` would exit 1 on every commit. |

### Confirmed already-aligned in cycle 2

| File | Why it is in sync (no edit needed) |
| --- | --- |
| `README.md` | The "Features" / "Operating loop" / "Portability" / "Known differences" sections do not enumerate individual `[hooks]` entries, so the cycle-2 default-on change does not regress any sentence here. The pipeline-order references and bidirectional cross-review wording landed in cycle-1 and remain valid. |
| `AGENTS.md` (root) | No mention of specific Codex hook entries, only `templates/base/.codex/` as the surface. The `.codex/config.toml` cycle-2 contents are an internal template detail, not an AGENTS-level contract. |
| `templates/base/AGENTS.md` | Same as above — references `.codex/config.toml` and `[hooks]` as a surface, not its specific entries. The "Codex setup checklist" still describes `codex trust .` + `ralph doctor` correctly. |
| `CLAUDE.md` (root) | Lists `.claude/hooks/` and points at the Codex equivalent at the directory level. The cycle-2 hook-script-path fix is invisible at this level of abstraction. |
| `templates/base/CLAUDE.md` | Same as root CLAUDE.md — directory-level reference only. |
| `.codex/README.md` (root) | Repo does not ship a root `.codex/` directory (verified with `ls`). The active `.codex/README.md` is the template body, which is what cycle 2 edited. |
| `docs/recipes/codex-setup.md` (root + template) | "[hooks] entries are visible" warning text remains correct: doctor still warns if a user strips the default-on hooks. The recipe does not describe the specific default entries, so no drift. The two files are byte-identical (`diff -u` empty). |
| `docs/quality/definition-of-done.md` | No mention of specific hook entries; the parity checklist and pipeline-order rules from cycle-1 stand as written. |
| `docs/architecture/repo-map.md` | The `templates/base/.codex/` line names `config.toml`, `hooks/`, `AGENTS.override.md`, `README.md` as the surface. No drift from cycle-2 — only `config.toml` body changed, which is internal to the template. |
| `internal/cli/doctor.go::probeBinary` (referenced from plan AC-1b / AC-6) | The plan's AC-1b/AC-6 describe doctor's contract at the CLI-detection + version + effective-config level. The cycle-2 hardening (5s timeout, first-line truncation, symmetric error wrap) is internal to `probeBinary` and does not change the contract observed by `ralph doctor` callers. No plan or spec edit needed. |
| `docs/reports/templates/cross-review-triage-report.md` (root + template) | Already bidirectional after `3abd1d7`. `diff -u` between root and template is empty. Greps over `docs/`, `.claude/`, `.agents/`, `templates/base/` for `Codex finding` returned only the pipeline-cycle prose (`When fixing Codex findings, ...`) which describes the act of fixing reviewer findings when Codex is the reviewer — that wording remains accurate and is unrelated to template column captions. |
| `docs/tech-debt/README.md` | Already records the cycle-2 entries (`checkHooks` not extended to `.codex/`, plus the now-RESOLVED `probeBinary` no-timeout row preserved with the strikethrough + RESOLVED comment for traceability). No further edits in this pass. |

### Cycle-2 cross-references verified

- `grep -rn "Codex finding\|codex finding"` over `docs/recipes/`, `docs/quality/`, `.claude/rules/`, `.claude/skills/`, `.agents/skills/`, `templates/base/`, `AGENTS.md`, `CLAUDE.md`, `README.md`: only hits are the pipeline-cycle prose in `post-implementation-pipeline.md` (root + template) and the orchestrator script string. None reference the triage template column captions.
- `diff -u docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md`: empty — root/template still in lock-step.
- `diff -u docs/reports/templates/cross-review-triage-report.md templates/base/docs/reports/templates/cross-review-triage-report.md`: empty — bidirectional template mirrored.
- Tech-debt RESOLVED comment for `probeBinary` timeout matches the actual code in `internal/cli/doctor.go:121-140` (5s `context.WithTimeout`, first-non-empty-line return, symmetric `LookPath`/`--version` error wrap).
- `templates/base/.codex/config.toml` `[[hooks.PostToolUse]]` entries point at `./.claude/hooks/check_mojibake.sh`, which exists at both `templates/base/.claude/hooks/check_mojibake.sh` and the root `.claude/hooks/check_mojibake.sh`.

### Cycle-2 verdict

The only documentation surface that drifted in cycle 2 was the `templates/base/.codex/README.md` Hooks section (it was still describing the pre-cycle-1 "no default entries — add your own" model). All other surfaces flagged by the parent prompt were already aligned. The cycle-2 implementation changes (`probeBinary` hardening, hook-path repair) are either internal-helper details not exposed in any documented contract, or were correctly anticipated by cycle-1 phrasing.

Recommendation: proceed to `/cross-review` (cycle 2/2 — cap reached after this cycle per `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`). No further documentation-driven follow-up work is required for the merge.

---

## Cycle 3 / Post-PR (2026-05-07)

PR #45 is already created. Cycle-2 cross-review triage (`docs/reports/cross-review-triage-2026-05-07-codex-cli-parity.md`) marked the cap reached and recommended proceeding to `/pr`. After the cap, two follow-up commits landed on the branch:

- `dbaeb30` — `feat: vendor .codex/ at the meta-repo root for dogfood parity`. Added `.codex/{config.toml, AGENTS.override.md, README.md, hooks/.gitkeep}` byte-identical to `templates/base/.codex/`. Updated the root `AGENTS.md` repo map to list both `.codex/` (root) and `templates/base/.codex/` (template) and to call out the parity check explicitly. Drops the `check-sync.sh` TEMPLATE_ONLY count from 14 → 10.
- `4bb0b87` — `chore: archive Codex CLI parity plan after PR #45 hand-off`. Moved `docs/plans/active/2026-05-07-codex-cli-parity.md` → `docs/plans/archive/`.

Plus a user-edited surface in this session: `templates/base/.codex/README.md` "Hooks" section was rewritten to lead with the default-on mojibake hooks and to warn off the `commit-msg-guard.sh` `^git commit` PostToolUse trap. The same edit must hold for the root `.codex/README.md` (since the two are required byte-identical by `check-sync.sh`).

### Cycle-3 method

Walked the surfaces flagged by the parent prompt:
1. AGENTS.md (root + template) repo map for new dual `.codex/` entries
2. `docs/architecture/repo-map.md` Codex control plane section
3. `README.md` "What `ralph init` scaffolds" tree
4. `docs/recipes/codex-setup.md` (root + template) hook reality
5. `docs/quality/{definition-of-done,quality-gates}.md` for `check-skill-sync.sh` as CI gate
6. `.claude/rules/{post-implementation-pipeline,subagent-policy}.md` for `/cross-review` consistency
7. Sweep for `codex-review` outside the AC-9 allowlist
8. Confirm root `.codex/README.md` matches the new template wording

For each, compared the on-disk doc against the actual repo state (newly-vendored `.codex/`, archived plan, current `check-sync.sh` summary, current Hooks section). No documentation was rewritten or invented.

### Updated in cycle 3

| File | What drifted | Fix applied |
| --- | --- | --- |
| `docs/architecture/repo-map.md` Codex control plane | Said `templates/base/.codex/` is shipped to scaffolded projects "the meta-repo itself does not vendor a `.codex/` directory". Outdated as of `dbaeb30`. | Replaced with two lines: `.codex/` for the meta-repo (same shape as the template, dogfood) + `templates/base/.codex/` as the `ralph init` source (kept byte-identical, validated by `scripts/check-sync.sh`). Mirrors the wording already in `AGENTS.md`. |
| `CLAUDE.md` (root) Claude-specific directories | Said "Codex equivalents live under `templates/base/.codex/config.toml` `[hooks]` and ship to scaffolded projects". The meta-repo now also vendors the surface at root, and the equivalents exist at both places. | Changed to "Codex equivalents live under `.codex/config.toml` `[hooks]` for the meta-repo and `templates/base/.codex/config.toml` for scaffolded projects; the two are kept byte-identical". |

### Confirmed already-aligned in cycle 3

| File | Why it is in sync (no edit needed) |
| --- | --- |
| `AGENTS.md` (root) repo map | `dbaeb30` already added the dual-entry block at lines 68–69: root `.codex/` ("dogfoods the parity it ships") + `templates/base/.codex/` ("kept identical via `scripts/check-sync.sh`"). No contradictions. |
| `templates/base/AGENTS.md` repo map | Lists `.codex/` and `.agents/skills/` from the perspective of a scaffolded project (no meta-repo references — correctly). The template repo map is project-flat by design. |
| `README.md` "What `ralph init` scaffolds" tree (lines 132–165) | Tree shows `.codex/{config.toml, hooks/, AGENTS.override.md, README.md}` and `.agents/skills/`. This is what `ralph init` produces; root `.codex/` in the meta-repo is dogfood and does not change scaffolded output. Tree is still accurate. |
| `README.md` Portability section (lines 240–262) | Lists `.codex/config.toml`, `.codex/hooks/`, `.codex/AGENTS.override.md`, `.codex/README.md` as Codex-native surfaces. True for both the meta-repo and scaffolded projects after `dbaeb30`. |
| `README.md` source-tree details (lines 287–311) | Shows `templates/`, `cmd/`, `internal/`, `scripts/`, etc. Does not enumerate root `.codex/` — and that is fine because the source tree section is high-level and the dual-`.codex/` parity is already covered by `AGENTS.md` and `repo-map.md`. |
| `docs/recipes/codex-setup.md` (root + template) | `diff -u root template` empty. Hook wording: "no `[hooks]` entries are visible" warning text remains correct (doctor still warns if a user strips the default-on hooks). The recipe does not enumerate the specific default entries, so the cycle-2 hook script-path repair is invisible at recipe level. No drift. |
| `templates/base/.codex/README.md` and root `.codex/README.md` | `diff -r .codex/ templates/base/.codex/` empty (verified). The user-edited Hooks section is consistent at both places: leads with default-on `PostToolUse` mojibake hooks pointing at `./.claude/hooks/check_mojibake.sh`, and explicitly warns that `scripts/commit-msg-guard.sh` is a git `commit-msg` hook (consumes `$1`) — not a Codex `PostToolUse` hook. |
| `templates/base/.codex/AGENTS.override.md` and root `.codex/AGENTS.override.md` | Byte-identical (verified). Permission/sandbox table, skill-invocation rules, "What Codex must not do" list — all unchanged from cycle-2 and still correct. |
| `templates/base/.codex/config.toml` and root `.codex/config.toml` | Byte-identical (verified). The two `[[hooks.PostToolUse]]` mojibake entries point at `./.claude/hooks/check_mojibake.sh`, which exists at both `templates/base/.claude/hooks/check_mojibake.sh` and the root `.claude/hooks/check_mojibake.sh`. |
| `docs/quality/definition-of-done.md` | Already includes `check-skill-sync.sh` as a checklist item ("Skill drift check (`./scripts/check-skill-sync.sh`) is green so `.claude/skills/` and `.agents/skills/` stay in lock-step") and the parity-related "both CLIs were exercised" item. No drift. |
| `docs/quality/quality-gates.md` | Already lists `check-skill-sync.sh` under "Must pass in CI before merge". `check-sync.sh` (templates/root parity) is also listed. No drift. |
| `.claude/rules/post-implementation-pipeline.md` (root + template) | `/cross-review` referenced consistently in the canonical order, CLI execution mode table, step-responsibilities table, re-run prose, and the "Where this order is referenced" list. Identical pattern in template. |
| `.claude/rules/subagent-policy.md` (root + template) | `/cross-review` referenced as inline at step 5 ("After step 4, `/cross-review` runs inline (optional), then `/pr`"). Codex triage section produces `cross-review-triage-<slug>.md`. Identical pattern in template. |
| Active plans directory | Empty after `4bb0b87` (only `.gitkeep` remains). Plan correctly archived to `docs/plans/archive/2026-05-07-codex-cli-parity.md`. |
| `docs/architecture/repo-map.md` Process artifacts list | Already lists `docs/recipes/`, `docs/quality/`, `docs/reports/`, `docs/tech-debt/`, `docs/plans/{active,archive}/`. No drift from `4bb0b87` (archive move is automatically reflected — the path is the same). |

### Cycle-3 cross-references verified

- `diff -r .codex/ templates/base/.codex/` returns empty (root and template byte-identical, all four files plus `hooks/.gitkeep`).
- `./scripts/check-sync.sh` final summary: `IDENTICAL: 143  DRIFTED: 0  ROOT_ONLY: 0  TEMPLATE_ONLY: 10  KNOWN_DIFF: 3` → `PASS: all files in sync`. The TEMPLATE_ONLY 14→10 drop matches the commit message in `dbaeb30`.
- `grep -rln "meta-repo does not vendor\|meta-repo itself does not vendor\|does not vendor.*\.codex"` over the repo: zero hits after the cycle-3 edit (was 1 hit in `docs/architecture/repo-map.md:20` before).
- `grep -rn "codex-review"` over `AGENTS.md`, `CLAUDE.md`, `README.md`, `docs/architecture/`, `docs/quality/`, `.claude/rules/`, `.claude/skills/`, `.agents/skills/`, `templates/base/`: only hits are inside the AC-9 allowlist (the rename-history note in `docs/recipes/codex-setup.md` and its template mirror at lines 83 and 88). No fresh residue.
- `grep -rn "/cross-review\|cross-review"` over `.claude/rules/post-implementation-pipeline.md` and `.claude/rules/subagent-policy.md`: 11 hits, all consistent with the post-rename canonical form.
- `ls docs/plans/active/`: only `.gitkeep` remains (the parity plan correctly moved to archive in `4bb0b87`).

### Cycle-3 verdict

**Drift found and fixed: 2 surfaces** — both were narrow textual updates to bring the documentation into alignment with `dbaeb30` (root `.codex/` now exists, byte-identical to the template).

1. `docs/architecture/repo-map.md` Codex control plane section: replaced the "the meta-repo itself does not vendor a `.codex/` directory" line with the new dual-entry block.
2. `CLAUDE.md` (root) Claude-specific directories: updated the Codex equivalents pointer to name both root `.codex/` (meta-repo) and `templates/base/.codex/` (scaffolded projects).

Everything else flagged by the parent prompt was already aligned — `dbaeb30` correctly updated `AGENTS.md`, the README scaffold tree is unaffected (root `.codex/` is meta-repo only), the recipes file is in lock-step, the quality docs already gate on `check-skill-sync.sh`, and the cross-review skill name is consistent everywhere outside the rename-history allowlist. `check-sync.sh` confirms the four root `.codex/` files are byte-identical to their template counterparts.

Recommendation: push the cycle-3 sync-docs commit to PR #45. No `/cross-review` regression is needed: the cap is already exhausted (cycle 2/2), and cycle-3 only updates documentation surfaces — no behavior, no tests, no shipped binary changed. The PR description and pre-checks remain valid.
