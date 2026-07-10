# Verify report: model-routing (upstream ralph 移植)

- Date: 2026-07-10
- Plan: `docs/plans/active/2026-07-10-model-routing.md`
- Branch: `chore/model-routing` (base `main`, merge-base `57757a0`)
- Commits: `f2a4904` (port) + `1e5f3d0` (Go-layer fix)
- Verifier scope: spec compliance + static analysis only. `run-test.sh` NOT run (tester's job).

## Verdict: PASS

All 7 acceptance criteria met (excluding the `run-test.sh` half of AC6, which is the tester's responsibility). Static analysis clean, sync checks pass, no documentation drift on default values.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|-----------|--------|----------|
| AC1 | root + templates/base: verifier/tester/doc-maintainer=`sonnet`, reviewer=`opus` | PASS | `grep '^model:'` — root: reviewer=opus, verifier/tester/doc-maintainer=sonnet; templates/base identical |
| AC2 | model-routing.md exists in both locations, no frontmatter | PASS | Both files present; both start with `# Model routing` (no `---` block). `diff` shows only the intentional 4-line Go-layer bullet in the root copy (KNOWN_DIFF, plan scope item 10) |
| AC3 | ralph-config.sh effective values `opus`/`high`/`opus`, both copies md5-identical | PASS | `sh -c '. scripts/ralph-config.sh; echo ...'` → `MODEL=opus EFFORT=high REVIEWER=opus`; both copies md5 `49c32ba2c7ac28276af6b49b1f77bbed`; env override (`RALPH_MODEL=sonnet`) still returns `sonnet` |
| AC4 | templates/base/ralph.toml `opus`/`high`/`opus` | PASS | `model = "opus"` (L4), `effort = "high"` (L5), `claude_reviewer_model = "opus"` (L23) |
| AC4b | Go `Default()` returns `opus`/`high`/`opus` | PASS | `internal/config/config.go`: `Model: "opus"` (L60), `Effort: "high"` (L61), `ClaudeReviewerModel: "opus"` (L74), all inside `Default()` |
| AC5 | no `claude-opus-4-7`/`xhigh` config values outside docs/specs, docs/plans, docs/reports | PASS | Whole-tree grep of `.sh/.toml/.md/.go` excluding those dirs → only hits are the intentional `xhigh`/`max` diminishing-returns **prose** in model-routing.md (root L42, template L42). No config-value hits |
| AC6a | `./scripts/run-verify.sh` succeeds | PASS | Exit 0. gofmt ok, 0 lint issues, all Go packages ok, all shell test suites pass |
| AC6b | `./scripts/run-test.sh` succeeds | DEFERRED | Tester's responsibility — not run here |
| AC7 | `./scripts/check-skill-sync.sh` succeeds | PASS | `[ok] check-skill-sync: 13 skill(s) in lock-step`, exit 0 |
| — | subagent-policy.md references model-routing.md (both copies) | PASS | Both copies L3: "Model tier assignment per seat is defined in `model-routing.md`." |
| — | cross-review SKILL.md 4 mirrors use `--model opus` | PASS | `.claude`/`.agents` + templates/base ×2, lines 55 & 154 all `--model opus` |
| — | test/Go fixtures synced (scope items 8, 9, 10) | PASS | `tests/test-ralph-config.sh` asserts opus/high (L54,57); `config_test.go` asserts opus/high (L11,14,83-84,106); `embed_test.go` pins `claude_reviewer_model = "opus"` (L162); `doctor_loop_test.go` uses `opus` (L42,101,130) |

## Commands run

```sh
# AC1
for a in reviewer verifier tester doc-maintainer; do grep -m1 '^model:' .claude/agents/$a.md; done
for a in reviewer verifier tester doc-maintainer; do grep -m1 '^model:' templates/base/.claude/agents/$a.md; done
# AC2
head -1 .claude/rules/model-routing.md; head -1 templates/base/.claude/rules/model-routing.md
diff .claude/rules/model-routing.md templates/base/.claude/rules/model-routing.md
# AC3
sh -c '. scripts/ralph-config.sh; echo $RALPH_MODEL $RALPH_EFFORT $RALPH_CLAUDE_REVIEWER_MODEL'
RALPH_MODEL=sonnet sh -c '. scripts/ralph-config.sh; echo $RALPH_MODEL'
md5 scripts/ralph-config.sh templates/base/scripts/ralph-config.sh
# AC4/4b
grep -nE 'model|effort|reviewer' templates/base/ralph.toml
grep -nE 'Model:|Effort:|ClaudeReviewerModel:' internal/config/config.go
# AC5
grep -rnE 'claude-opus-4-7|xhigh' . --include='*.sh' --include='*.toml' --include='*.md' --include='*.go' \
  | grep -vE '/docs/(specs|plans|reports)/|/\.git/|/\.ralph/baseline/'
# AC6a / AC7 / sync
RALPH_VERIFY_BASE=main ./scripts/run-verify.sh          # exit 0
./scripts/check-skill-sync.sh                            # exit 0
./scripts/check-sync.sh                                  # exit 0, model-routing.md = KNOWN_DIFF
# doc drift
grep -nE 'claude-opus-4-7|xhigh|claude-sonnet-4-20250514' README.md docs/recipes/ralph-loop.md templates/base/docs/recipes/ralph-loop.md
```

## Documentation drift

No drift. `README.md`, `docs/recipes/ralph-loop.md`, and `templates/base/docs/recipes/ralph-loop.md` contain **no** stale default values (`claude-opus-4-7`/`xhigh`/`claude-sonnet-4-20250514`). Both recipe tables show `RALPH_MODEL=opus`, `RALPH_EFFORT=high`, `RALPH_CLAUDE_REVIEWER_MODEL=opus`. The `RALPH_MODEL=sonnet` on ralph-loop.md L161 is an override demonstration, not a default claim — consistent.

`./scripts/check-sync.sh` reports `DRIFTED: 0`, `KNOWN_DIFF: 3` (model-routing.md, verify.yml, CLAUDE.md), and `PASS: all files in sync.` model-routing.md correctly surfaces as **KNOWN_DIFF**, not DRIFTED — matching the documented intentional Go-layer bullet divergence.

## Notes / deviations

- **Verify ran full-scope, not changed-scope.** `run-verify.sh` reported `Language scope: full`. Unlike koalive, the upstream ralph detector has no `classify_harness_file` harness classification (plan Non-goals item 1 documents this as intentionally not ported — ralph's detector structure differs and the same misclassification does not occur). Full scope passed cleanly (Go + shell suites), so this is stronger coverage, not a gap.
- `.ralph/baseline/` divergence after a live-config change is expected and was not flagged.
- AC5 residual `xhigh`/`max` hits in model-routing.md (both copies) are intentional policy prose about effort diminishing returns, verified by reading the lines — not config values.

## Verified vs. unverified

- **Verified:** all frontmatter tiers, model-routing.md presence/no-frontmatter/KNOWN_DIFF, effective shell config + env-override survival + md5 identity, ralph.toml + Go `Default()` values, stale-ID absence, subagent-policy references, cross-review 4-mirror `--model`, test/Go fixture sync, static analysis (run-verify exit 0), skill-sync, check-sync, doc-drift absence.
- **Deferred to tester:** `./scripts/run-test.sh` (behavioral test suite, incl. `tests/test-ralph-config.sh` runtime execution).
- **Not verifiable statically (tester/runtime):** whether the Anthropic API / installed `claude` CLI accept the alias `opus` and effort `high` at runtime. Alias-over-full-ID was chosen specifically to avoid stale-ID breakage, but live acceptance is a `/test` concern.
