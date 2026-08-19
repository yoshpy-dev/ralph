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

## Cycle 2

- HEAD at start: `5312a55` (docs: add phase-5 test cycle-2 section (pass))
- Scope: residual drift from the cycle-2 behavior changes — `e01939e`
  (doctor `--strict` now also fails on ownership-planning errors and a
  deleted settings snapshot), `e39d436` (allowlist parallel arrays; `--strict`
  help + README meta-failure clause added in the same commit), `1f50407`
  (stale allowlist name fix in the purity-guard failure message).

### Verdict: no residual drift

### 1. `--strict` described as exactly-five-checks or warn-only for planning errors elsewhere

No drift. `grep -rn -- "--strict" docs/ .claude/rules/ templates/base/` was
audited in full. README's `ralph doctor [--strict]` row already carries the
C2-M1 fix (`e39d436`): it names the meta-failure clause ("and on the
meta-failures that make those checks impossible ... unparseable manifest,
unreadable tracked file"). No other doc claims an exact five-check
enumeration or describes planning errors as warn-only under `--strict`:
`docs/specs/2026-08-17-overlay-scaffold-v2.md` and the archived
`docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md` describe FR-9 (a)-(e)
in the abstract (no "exactly five" closure claim); `docs/reports/*.md` hits
are historical narrative (self-review/verify/cross-review-triage findings
and their fix confirmations), not living spec text. `templates/base/` has
zero `--strict` mentions (doctor.go is Go source, not shipped template
content — see point 3).

### 2. Tech-debt register table — column count and contradiction check

No drift. `awk -F'|' '/^\|/ {print NR": "NF}' docs/tech-debt/README.md`
confirms every row from the cycle-2-adjacent additions (line 117, `74d28ef`'s
quality-gates.md wiring-mismatch row; line 118, `e39d436`'s batched C2-L4/L5
deferral row) renders at 7 awk fields (5 markdown columns, matching the
6-column header incl. leading/trailing pipe) — consistent with every other
row added this series. Pre-existing rows at irregular field counts (8/9/11/19,
e.g. lines 22/24/32/46/57/79/90) predate this cycle and come from literal `|`
characters inside code spans (shell `||`, table examples), not something
`e01939e`/`e39d436` introduced. Content check: the C2-L4/L5 row's claim that
cycle-1 triage/verify report line-number pointers into `doctor.go` "shifted
downward" after `e01939e`'s diff is accurate (confirmed the diff inserts code
above the pointed-at lines) and does not contradict the newly-fixed AR#1/AR#2
rows recorded in the plan's Deviations (below) — it is a separate, narrower
claim about historical-report accuracy, not scaffold-integrity behavior.

### 3. `templates/base/` — cycle-2 template-side counterpart check

No drift, none needed. `check-template-purity.sh` is registered in
`scripts/check-sync.sh`'s `ROOT_ONLY_EXCLUSIONS` (line 37) — it is
intentionally not shipped to `templates/base/`. `internal/cli/doctor.go` is
Go binary source consumed by `go build`, not template content rendered by
`ralph init`; `find templates/base -iname "*doctor*" -o -iname "*purity*"`
returns no hits, confirming there is no template-side file that could drift
from the cycle-2 Go-side behavior changes.

### 4. Plan Deviations — cycle-2 note

Real gap, fixed. The plan's `## Deviations` section had three Slice-4/5
entries but no cycle-2 pipeline note. Added one bullet summarizing: cross-review
cycle-1 ACTION_REQUIRED (AR#1 doctor `--strict` planning-error fail-open,
AR#2 settings-snapshot deletion escaping FR-9(e)) fixed in `e01939e`;
self-review cycle-2's 5 new findings (C2-M1, C2-M2, C2-L1..L3) fixed in
`e39d436`; remaining C2-L4/C2-L5 deferred to the batched tech-debt row at cap
2/2.

## Files changed (Cycle 2)

- `docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md` (Deviations bullet)
- `docs/reports/sync-docs-2026-08-19-overlay-scaffold-v2-p5.md` (this section)

## Cycle 2 known gaps

None identified.
