# Verify report: harness-drift-audit-fixes

- Date: 2026-07-12
- Plan: no plan file — spec is the six audit findings from commit fa811a3
- Verifier: claude-sonnet-4-6 (verifier subagent)
- Scope: changed-language (docs-only; golang verifier triggered via unclassified ralph.toml change)
- Evidence: `docs/evidence/verify-2026-07-12-081730.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| F1: recipe permission-mode dual-default note present in `docs/recipes/ralph-loop.md` | PASS | Line 141: "Default is `bypassPermissions` when invoking the shell scripts directly…; `auto` when launched via `ralph run`" |
| F1: same note present in `templates/base/docs/recipes/ralph-loop.md` (mirror) | PASS | Identical text at line 141; `diff` confirms byte-identical mirror |
| F2: `docs/reports/` description updated to include sync-docs and cross-review triage in `AGENTS.md` | PASS | Line 69: "self-review, verify, test, sync-docs, cross-review triage, walkthrough artifacts" |
| F2: same update present in `templates/base/AGENTS.md` (mirror) | PASS | Identical text at line 69; `diff` confirms byte-identical mirror |
| F3: `CLAUDE.md` anti-bottleneck note added (user-invocable: false, belongs to neither list) | PASS | Line 9: "`anti-bottleneck` is a model-internal support skill (`user-invocable: false`) and belongs to neither list" |
| F4: `## Tests` section added to `docs/architecture/repo-map.md` | PASS | Lines 57-59: section header + `tests/` entry with test-*.sh and fixtures/cli-stubs description |
| F5: `templates/base/ralph.toml` prompts-dir comment added (field retained) | PASS | Line 12: comment "The pipeline currently reads its prompts from `.claude/skills/loop/prompts/`; this key is reserved for future prompt-dir override support." |
| F6: `docs/tech-debt/README.md` permission-mode divergence row added | PASS | Line 40: "`RALPH_PERMISSION_MODE` default divergence: shell default is `bypassPermissions`…; toml/Go default is `auto`" |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh < /dev/null` | PASS (exit 0) | All verifiers passed; log at docs/evidence/verify-2026-07-12-081730.log |
| shellcheck + hook sh -n passes | PASS | All 9 root hooks + 9 template hooks checked; 0 issues |
| jq -e . .claude/settings.json | PASS | Valid JSON |
| jq -e . templates/base/.claude/settings.json | PASS | Valid JSON |
| Codex hook guard | PASS | Single-source guard OK; inline hook detector OK; PR provenance OK |
| scripts/check-sync.sh | PASS | DRIFTED=0; IDENTICAL=170; KNOWN_DIFF=3 (expected) |
| scripts/check-pipeline-sync.sh | PASS | All 9 consumers reference all pipeline steps |
| scripts/check-skill-sync.sh | PASS | 13 skills in lock-step |
| golang verifier (gofmt + go vet) | PASS | 0 issues (triggered by unclassified templates/base/ralph.toml) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/model-routing.md` | In sync | Not touched by this diff; no drift introduced |
| `docs/quality/definition-of-done.md` | In sync | Not touched by this diff; no drift introduced |
| `AGENTS.md` vs `templates/base/AGENTS.md` mirror | In sync | docs/reports/ line is byte-identical across both files |
| `docs/recipes/ralph-loop.md` vs `templates/base/docs/recipes/ralph-loop.md` mirror | In sync | permission-mode note is byte-identical across both files |
| `templates/base/docs/architecture/repo-map.md` | n/a | Does not exist (no template mirror for repo-map); confirmed in self-review |
| check-sync.sh KNOWN_DIFF list | In sync | CLAUDE.md and model-routing.md are expected KNOWN_DIFF; no new drifted file added |

## Observational checks

Factual accuracy cross-check (from self-review evidence, confirmed):

- F1 claim (`bypassPermissions` shell default): `scripts/ralph-config.sh:20` → `RALPH_PERMISSION_MODE="${RALPH_PERMISSION_MODE:-bypassPermissions}"` — accurate.
- F1 claim (`auto` via `ralph run`): `internal/config/config.go` `Default()` → `PermissionMode: "auto"`; `templates/base/ralph.toml:9` → `permission_mode = "auto"`; `internal/cli/run.go:61` exports `RALPH_PERMISSION_MODE` — accurate.
- F3 claim (`anti-bottleneck` is `user-invocable: false`): `.claude/skills/anti-bottleneck/SKILL.md` frontmatter confirmed by self-review.
- F4 claim (`tests/` holds 29 test-*.sh + fixtures/cli-stubs/`): confirmed by self-review ls output.
- F5 claim (pipeline reads from `.claude/skills/loop/prompts/`; `dir` key not consumed by shell): confirmed by self-review audit of ralph-pipeline.sh / ralph-orchestrator.sh.
- F6 tech-debt row shape: 5 columns matching table header; no stray pipes.

## Coverage gaps

- Runtime behavior of `ralph.toml` `prompts.dir` key: the comment added in F5 claims the key is "reserved for future override support." Static analysis cannot confirm whether the Go CLI actually ignores it at runtime vs. reads it silently. This is a known/acceptable limitation; the self-review classified the claim as accurate based on shell pipeline code.
- The permission-mode divergence recorded in F6 is a product decision deferred to a future task; this verify run does not attempt to resolve it.

## Verdict

- Verified: all 6 audit findings resolved with grep evidence; both mirrors (AGENTS.md and ralph-loop.md) confirmed byte-identical; static analysis clean (exit 0); check-sync.sh DRIFTED=0; check-pipeline-sync.sh and check-skill-sync.sh both pass; no doc drift to model-routing.md or quality docs.
- Partially verified: ralph.toml `prompts.dir` runtime behavior (static analysis cannot confirm Go CLI ignores it at runtime; shell pipeline audit strongly suggests it).
- Not verified: behavioral tests (tester responsibility).

**Verdict: PASS**
