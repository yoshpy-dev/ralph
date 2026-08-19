# Sync-docs report — overlay-scaffold-v2-p5

- Plan: `docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md`
- Date: 2026-08-19
- Cycle: 1
- HEAD at start: 38b7c63 (docs: add phase-5 test report (pass))
- Scope: residual drift check only. Slice 5 (a164b2c) already ran the primary
  docs-alignment pass (spec FR/NFR checkboxes, README command table,
  AGENTS.md repo map, CLAUDE.md, repo-map.md, tech-debt rows), refined by
  7cf4f47/74d28ef.

## Verdict: drift found and fixed

One real drift item, fixed. All other checked points: no drift (already
current, or out of scope by design).

## Per-point findings

### 1. `docs/recipes/` (root) — drift/adopt/eject mentions

No drift.

- `docs/recipes/adding-a-language-pack.md` already tells the reader to run
  `ralph eject .claude/rules/ralph/<language>.md` before diverging from a
  pack's shipped rule content (line ~44) — up to date with FR-2.
- `docs/recipes/worktrees.md`'s only match is `--force-branch` (a
  `ralph-worktree.sh` cleanup flag, unrelated to scaffold ownership/drift).
- `docs/recipes/codex-setup.md`'s "Recovery from a half-applied upgrade"
  section describes a specific one-time rename recovery (`codex-review` →
  `cross-review`) via `git restore`, not general scaffold-ownership drift.
  Root and `templates/base/` copies are byte-identical (`diff` exit 0); no
  eject/adopt mention warranted for this narrow scenario.

### 2. `docs/quality/definition-of-done.md` — doctor/--strict/purity mention

Judged out of scope. `grep -n "doctor\|--strict\|purity"` returns no hits, and
this is consistent with the doc's existing scope: it lists the standard
`/work` pipeline definition-of-done, not a CLI command/gate inventory — it
doesn't mention `ralph status`, `ralph upgrade`, or any other command either.
`templates/base/docs/quality/definition-of-done.md` and
`templates/base/docs/quality/quality-gates.md` show the same pattern (zero
command mentions). Adding a doctor/--strict/purity reference here would be
scope creep beyond residual drift, not a fix of stale content.

### 3. `.claude/rules/ralph/*.md` — stale hooks-wiring/drift/fork descriptions

No drift. `grep -rn "calls hook scripts directly"` across `.claude/rules/ralph/*.md`
returns nothing (that phrasing was only ever in `AGENTS.md`'s repo map, which
Slice 5 already corrected — confirmed current worktree `AGENTS.md` reads
"`hooks/<event>.d/` wiring runs under both Claude Code and Codex — both route
through `ralph-dispatch.sh`"). The only `drift`/`fork` hits in
`.claude/rules/ralph/*.md` are unrelated: `planning.md` and
`post-implementation-pipeline.md`/`subagent-policy.md` use "fork" only in the
sense of `/plan`'s "critical-fork resolution" (a decision branch), not
scaffold ownership.

### 4. `templates/base/` shipped docs — do downstream users learn about eject/adopt/doctor --strict?

No drift found; confirmed this is a pre-existing pattern, not something
Phase 5 regressed. `templates/base/AGENTS.md` and
`templates/base/docs/quality/{definition-of-done,quality-gates}.md` mention
zero `ralph` subcommands (not `upgrade`, not `doctor`, not `status` either) —
command documentation was never shipped there; `README.md` (which carries the
full command table) is a root-only, non-shipped file
(`templates/base/` has no `README.md`; confirmed via directory listing).
The one place `templates/base/` does document CLI usage,
`templates/base/docs/recipes/adding-a-language-pack.md`, already references
`ralph eject`, `ralph upgrade`, and `ralph pack add` and is current (see
point 1). No sibling gap to close.

### 5. `ralph upgrade` drift-report guidance text — real drift, fixed

FR-4 states eject/adopt are now **the** resolution path for unresolved
drift, but the three user-facing "Unresolved drift" messages in
`internal/cli/upgrade_v2.go` (real-run error path, no-op tail, and
`--dry-run` preview) and the `## Unresolved drift` section rendered by
`internal/upgrade/report.go` said only "left untouched" / "would be left
untouched" with no resolution pointer — stale relative to the now-shipped
FR-2/FR-3 commands.

Fixed by appending one guidance line at each site:

```
Resolve with `ralph eject <path>` (keep the local change, tracked as a
fork) or `ralph adopt <path>` (discard the local change, revert to
template).
```

- `internal/cli/upgrade_v2.go:225-230` — real-run error path (`runUpgradeV2`)
- `internal/cli/upgrade_v2.go:304-309` — no-op tail (`finishNoOpUpgradeV2`)
- `internal/cli/upgrade_v2.go:784-789` — `--dry-run` preview
- `internal/upgrade/report.go:123-131` — `## Unresolved drift` markdown
  section (`renderDriftSection`)

Verified no existing test asserts an exact/whole message string for these
sites — all use `strings.Contains` on substrings ("Unresolved drift",
"## Unresolved drift") that remain unchanged, so the appended line is
additive and safe (`internal/cli/migrate_test.go:2239`,
`internal/cli/upgrade_v2_test.go:1733`, `internal/upgrade/report_test.go:41,95`).

### 6. `docs/insights/README.md` schema mention

Skipped per scope note — `grep -n "eject\|adopt\|owner\|fork"` returns no
hits, confirming no change is implicated.

## Files changed

- `internal/cli/upgrade_v2.go` (3 message sites)
- `internal/upgrade/report.go` (1 section header)

## Verification evidence

```
$ gofmt -l internal/cli/upgrade_v2.go internal/upgrade/report.go
(no output — clean)

$ go build ./...
(no output — clean)

$ go vet ./...
(no output — clean)

$ go test ./internal/upgrade/... ./internal/cli/...
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	1.233s
ok  	github.com/yoshpy-dev/ralph/internal/cli	32.424s
```

## Known gaps

None identified for this residual-drift pass. Full-repo `go test ./...` and
`./scripts/run-verify.sh` are the tester/verifier subagents' responsibility
for this cycle and are not re-run here (doc-maintainer scope is limited to
the two Go files above, both covered by the package tests run).
