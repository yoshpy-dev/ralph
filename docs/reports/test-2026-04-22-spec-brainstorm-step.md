# Test report: spec brainstorm step (docs-only)

- Date: 2026-04-22
- Plan: docs/spec-brainstorm-step (branch)
- Tester: tester subagent (`/test`)
- Scope: behavioral tests across the full repo test surface (Go unit tests + shell hook tests + project `run-test.sh` wrapper)
- Evidence: `docs/evidence/test-2026-04-22-spec-brainstorm-step.log`

## Change summary

This branch is a **docs-only change** to the `/spec` skill. `git diff --stat` against `main`:

```
 .claude/skills/spec/SKILL.md                | 50 ++++++++++++++++++-----------
 AGENTS.md                                   |  2 +-
 CLAUDE.md                                   |  2 +-
 README.md                                   |  2 +-
 templates/base/.claude/skills/spec/SKILL.md | 50 ++++++++++++++++++-----------
 templates/base/AGENTS.md                    |  2 +-
 templates/base/CLAUDE.md                    |  2 +-
 7 files changed, 69 insertions(+), 41 deletions(-)
```

All 7 modified files are Markdown. No `.go`, `.sh`, `.toml`, or other executable/config files changed.

**No new behavioral tests required — docs-only skill update.** The change modifies a Claude Code skill's instruction text (adding a Brainstorm step to `/spec`). Skill instructions are not executed by the Go CLI or hook scripts; they are read by Claude Code at runtime. There is no behavior in-repo that a unit test could meaningfully assert against.

The existing behavioral test surfaces (Go tests, mojibake hook test) were still run to confirm no regressions were introduced incidentally (e.g., via AGENTS.md/CLAUDE.md/README.md template drift that could break a template distribution test).

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test -count=1 ./...` (9 packages with tests) | 196 | 194 | 0 | 2 | ~25s wall |
| `./tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| `./scripts/run-test.sh` (local verifier + golang verifier, test mode) | — | all | 0 | — | ~2s (cached) |

Go packages covered (9 with tests, 3 with no test files):

```
ok   internal/action       4.229s
ok   internal/cli          1.313s
ok   internal/config       2.550s
ok   internal/scaffold     1.148s
ok   internal/state        2.224s
ok   internal/ui           1.862s
ok   internal/ui/panes     3.645s
ok   internal/upgrade      2.908s
ok   internal/watcher      4.853s
?    (root), cmd/ralph, cmd/ralph-tui   [no test files]
```

The 2 skipped Go tests are the pre-existing platform/environment-gated skips (documented in `.claude/agent-memory/tester/go_test_packages.md`), unrelated to this diff.

## Coverage

- Statement: n/a (docs-only change — no executable code modified)
- Branch: n/a
- Function: n/a
- Notes: Coverage is irrelevant for this diff. Markdown changes to skill instruction files cannot be exercised by Go or shell tests. The test run above confirms the rest of the repo remains green, which is the meaningful signal for a docs-only branch.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| mojibake hook exit codes across Edit/Write/MultiEdit payload shapes | Still green (11/11) | `docs/evidence/test-2026-04-22-spec-brainstorm-step.log` |
| Go `internal/scaffold` + `internal/upgrade` (template-distribution + local-edit detection) | Still green | same log |
| Full `./scripts/run-test.sh` pipeline | Still green | same log |

## Test gaps

Acknowledged and accepted as out of scope for this diff:

- **Skill-instruction changes are not verified by automated tests.** There is no project-level framework that lints skill markdown for structural correctness (e.g., "does `/spec` still declare `disable-model-invocation: true`?"). Such a linter could be added if skill-text regressions ever appear in production, but is not proportional for a single-step addition.
- **README.md / AGENTS.md / CLAUDE.md template sync** is currently not covered by a dedicated drift test. `scripts/check-sync.sh` (invoked from `scripts/verify.local.sh`) covers template file parity; the `/verify` report on this branch already confirms sync is intact.

Neither gap is introduced by this change; they pre-exist. No new tests are proposed here.

## Verdict

- Pass: yes
- Fail: no
- Blocked: no

Tests pass. Proceeding to `/sync-docs` → `/codex-review` → `/pr` is unblocked from the tester gate.
