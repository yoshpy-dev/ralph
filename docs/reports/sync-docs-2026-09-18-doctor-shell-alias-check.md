# sync-docs report: doctor-shell-alias-check

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-shell-alias-check.md` (issue #162)
- Prior reports: `docs/reports/self-review-2026-09-18-doctor-shell-alias-check.md`,
  `docs/reports/verify-2026-09-18-doctor-shell-alias-check.md` (PASS),
  `docs/reports/test-2026-09-18-doctor-shell-alias-check.md` (PASS)
- Pipeline cycle: 1 (of cap 2)

## Summary

The plan's own Slice B (docs) had already updated the four `/org` skill
mirrors and the `docs/recipes/codex-seat-permissions.md` + template pair with
the "`ralph doctor`'s Shell aliases (codex/claude) check reports such
aliases" sentence, and self-review's Revalidation pass (N1) had already
corrected that sentence's wording to match `permissionArgsForDriver`'s
guarded-mode behavior. This pass audited every remaining doc surface listed
in the sync-docs task brief against the current diff (`git diff main...HEAD`)
and found one gap: the config-aware-grading follow-up that self-review N1
named had not been recorded in `docs/tech-debt/README.md`. That is the only
change made in this pass, plus the plan's own Progress checklist.

## Files changed

- `docs/tech-debt/README.md` — added one row for the follow-up self-review
  N1 named but did not implement: `checkShellAliases` grades a finding by
  flag class only, without reading `ralph.toml`'s `[org.permissions]` to know
  which seats are actually `guarded`, so its Detail sentences and the
  mirrored doc prose stay conditional instead of stating what happens on the
  reader's own project. The row also records the two permanent, by-design
  parsing limits: rc files pulled in via `source` are not followed
  (`shellAliasRcCandidates`'s static candidate list), and aliases defined
  dynamically (shell functions, `eval`) are invisible to the line-based
  static scanner (the plan's Non-goals rule out launching an interactive
  shell to evaluate them). `docs/tech-debt/README.md` is a `ROOT_ONLY_EXCLUSIONS`
  entry in `scripts/check-sync.sh`, so no `templates/base/` copy applies.
- `docs/plans/active/2026-09-18-doctor-shell-alias-check.md` — Progress
  checklist unchanged (`PR created` intentionally left unchecked per the
  handoff); no other plan edits needed.

## Files checked (no changes needed — already correct)

- `README.md` — `ralph doctor [--strict]` table row (line ~127) is a
  general description ("Check Claude Code, Codex, hooks, manifest drift,
  language packs") and does not enumerate individual Check names or numbers
  anywhere in the file (confirmed via `grep -n doctor README.md`); the new
  "Shell aliases (codex/claude)" Check needs no row edit, matching the
  plan's own Scope note that README's doctor description is generic.
- `AGENTS.md` — repo map's `internal/cli/` line already lists `doctor`
  generically alongside the other subcommands; it does not enumerate
  individual Checks, so no edit is needed. Kept short per the file's own
  Hard rules.
- `docs/specs/2026-08-01-org-runtime.md` — searched every `doctor` mention
  (11 hits: FR-2 model-pool probing, the dependency line for herdr/agmsg,
  AC line, and the 2026-09-18 codex-slug-drift operational note). None of
  them is a single enumerated "doctor checks" list (herdr / agmsg / org
  envelope / codex slugs / probes) — each mention is scoped to its own FR or
  operational note. There is no list for the new shell-alias Check to join.
- `docs/recipes/codex-setup.md` — describes `ralph doctor`'s hooks-trust and
  wiring checks only (`hooks=true`, `hooks.json` dispatcher wiring); it does
  not claim to enumerate every Check, and shell aliases are outside its
  hooks-trust scope, so it is unaffected by this change.
- `docs/quality/` — `definition-of-done.md` and `quality-gates.md` do not
  enumerate `ralph doctor` Checks (grepped for `doctor`; no hits in either
  file), so neither needed an edit.
- `docs/recipes/codex-seat-permissions.md` + `templates/base/docs/recipes/codex-seat-permissions.md`
  — already updated by the plan's Slice B and by self-review's wording fix;
  `diff` against the template copy confirms the two are still byte-identical.
- `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`,
  `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md` — all four already carry the
  same doctor-check sentence from Slice B; `diff` across all four mirror
  pairs confirms exact byte match.

## Decided not to change (and why)

- Did not touch the skill/recipe prose that already states the herdr/info
  vs. warn distinction, since it is already correct per the plan's Design
  decisions and self-review's revalidation, and the handoff asked me not to
  edit paragraphs already updated unless factually wrong.
- Did not add a config-aware-grading fix to `internal/cli/doctor_shell_alias.go`
  itself — that is Go implementation work outside `/sync-docs` scope; it is
  the subject of the new tech-debt row instead.
- Did not append a `docs/insights/events/` entry for this sync-docs run:
  `docs/insights/README.md`'s "Appending events" section lists only
  `/self-review`, `/verify`, `/test`, `/cross-review` as live callers of
  `scripts/insights-append.sh`, and `.claude/skills/sync-docs/SKILL.md` has
  no `insights-append.sh` step. This gap is already tracked in
  `docs/tech-debt/README.md` (row: "insight event の cycle スタンプ機構が脆弱",
  evidence `docs/reports/self-review-2026-08-24-codex-hooks-multi-event.md`
  C3-1); adding an ad hoc event here would not follow a documented
  procedure, so none was written.

## Gate results

Run after the `docs/tech-debt/README.md` edit above, in this worktree:

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step

$ ./scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
```

All three PASS. No skill files were touched in this pass (Slice B already
landed the `/org` skill edits), so `check-skill-sync.sh`'s pass carries over
unchanged.

## Progress checklist

- Plan's `PR created` checkbox left unchecked, per the handoff — `/pr` has
  not run yet.

## Cycle 2 (post cross-review fix + cycle-2 self-review/verify/test)

- Date: 2026-09-19
- Trigger: cross-review cycle 1 found 4 issues (AR-1, AR-2, WC-1, WC-2 —
  `docs/reports/cross-review-triage-doctor-shell-alias-check.md`); the user
  chose to fix all four and re-run the full pipeline as cycle 2/2. Cycle-2
  self-review found 6 more findings (C2-1..C2-6, all fixed in-cycle),
  `/verify` (`573e474`, PASS) and `/test` (`2756ea0`, PASS) confirmed the
  fixes.
- Reviewed: `git diff e4e98ab..HEAD` (this pass's own cycle-1 commit as the
  base), 13 commits, touching `internal/cli/doctor_shell_alias.go` (candidate
  files now include `.zshenv`/`.zprofile`/`.zshrc`/`.zlogin` under each of
  `$ZDOTDIR`/`~`/`~/.config/zsh`; codex's sentence split into three clauses
  for `--model`/`--sandbox`/`--ask-for-approval`, with the sandbox and
  approval clauses now naming the `[org.permissions].codex_verified = true`
  gate; a stat failure on a candidate directory is reported once as
  `could not read`), plus the four `/org` skill mirrors and the
  `codex-seat-permissions.md` recipe pair, already updated in this branch
  before this pass started (`f1898d8`, `4d26aed`, `812d5c4`).

### Files changed

- `docs/tech-debt/README.md` — re-verified every sentence in the cycle-1 row
  against the current code and tightened the wording rather than lengthening
  it:
  - The row already named the `[org.permissions]`-blind grading gap; added
    that the gap now also covers `codex_verified`: the codex clauses name
    `[org.permissions].codex_verified = true` in prose but the check itself
    never reads `ralph.toml` to know whether that flag is actually `true` on
    the reader's own project, so the clause's conditional phrasing is never
    resolved into a concrete statement either way.
  - Added the one new parsing limit the cycle-2 self-review documented
    (`shellAliasFileScan.Unclosed`'s doc comment, C2-1/C2-2 area): an
    unbalanced quote that a *later*, unrelated alias statement's own quote
    happens to close swallows the lines in between with no `not fully
    parsed` signal — zsh itself would report that rc as broken, and there is
    no local marker for it in a single statement's open/closed state. Kept
    the existing two limits (`source` not followed; dynamic
    functions/`eval` aliases invisible to a static scan) alongside it as one
    "three further limits" clause instead of a fourth sentence.
  - Updated the "Why deferred" and "Related plan/report" cells to cite
    cycle-2's C2-4 finding and the cross-review triage report alongside the
    original N1 reference, since C2-4 independently reconfirmed the same
    gap this cycle.
  - `docs/tech-debt/README.md` remains a `ROOT_ONLY_EXCLUSIONS` entry in
    `scripts/check-sync.sh`; no `templates/base/` copy needed.
- No other files changed in this cycle-2 pass.

### Files checked (no changes needed — already correct)

- `README.md`, `AGENTS.md`, `docs/specs/2026-08-01-org-runtime.md`,
  `docs/recipes/codex-setup.md`, `docs/quality/definition-of-done.md`,
  `docs/quality/quality-gates.md` — `git diff e4e98ab..HEAD --stat` shows
  none of these files were touched since cycle 1, and cycle 1's audit already
  confirmed none of them enumerates individual `ralph doctor` checks. Nothing
  in cycle 2's behavior changes (candidate file list, clause splitting,
  `codex_verified` naming) creates new drift in files that were already
  confirmed not to enumerate checks.
- `.claude/skills/org/SKILL.md` (+3 mirrors) and
  `docs/recipes/codex-seat-permissions.md` (+ template) — read the current
  text against the three-clause Go code
  (`shellAliasCodexSentence`/`shellAliasClaudeSentence`) line by line. All
  three surfaces (Go Detail, skill, recipe) tell the same story: codex
  `--sandbox` fails edits/autonomous and silently applies on guarded; codex
  `--ask-for-approval` fails autonomous only and silently applies on
  edits/guarded; claude `--model` never breaks the spawn (ralph's value is
  appended last); claude `--permission-mode` is only withheld from guarded,
  so a guarded claude seat runs with the alias's value. The skill explicitly
  states the `codex_verified = true` precondition for edits/autonomous codex
  seats existing at all (`.claude/skills/org/SKILL.md:44-46`), matching the
  Go clauses' parenthetical. The recipe's alias paragraph does not repeat
  the `codex_verified` precondition inline, but its own opening paragraph
  (lines 3–5, unchanged from cycle 1) already states the default-`false`
  fail-closed behavior immediately above the alias paragraph — the same
  precondition, stated once at the top of the document rather than
  re-stated per-flag. I judge this consistent (matching what `/verify`
  cycle 2 already concluded) and did not add a sentence; flagging here per
  the handoff's "report, don't edit" instruction in case the team wants the
  recipe to restate it inline for a reader who skips the opening paragraph.
  All four skill mirrors confirmed byte-identical by `diff`; both recipe
  copies confirmed byte-identical by `diff`.

### Decided not to change (and why)

- Same insight-event reasoning as cycle 1: `sync-docs/SKILL.md` still has no
  `insights-append.sh` step, so no event was appended for this pass either.
- Did not touch the Go code, the skill paragraphs, or the recipe paragraph —
  all were judged factually correct against the current implementation.

### Gate results

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step

$ ./scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
```

All three PASS. Plan's `PR created` checkbox left unchecked again.
