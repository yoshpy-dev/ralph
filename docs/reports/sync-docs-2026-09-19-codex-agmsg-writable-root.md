# sync-docs report: codex-agmsg-writable-root

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` (issue #164)
- Prior reports: `docs/reports/self-review-2026-09-19-codex-agmsg-writable-root.md`
  (all findings fixed, verdict merge), `docs/reports/verify-2026-09-19-codex-agmsg-writable-root.md`
  (PASS, no drift), `docs/reports/test-2026-09-19-codex-agmsg-writable-root.md` (PASS)
- Pipeline cycle: 1 (of cap 2)

## Summary

The plan's own Slice B (docs) had already updated the four `/org` skill
mirrors and the `docs/recipes/codex-seat-permissions.md` + template pair with
the writable-root check description, and self-review's later cycles kept that
wording aligned with the shipped `codexSeatModesPossible`/`codexSandboxReasons`
logic (role- and `driver_pool`-based, not just `codex_verified`). This pass
audited every remaining doc surface named in the sync-docs task brief against
the current diff (`git diff main...HEAD`) and the shipped
`internal/cli/doctor_codex_writable_root.go`, and made two changes: a new
`docs/tech-debt/README.md` row for the check's own known limits, and a
one-line dated follow-up note on `docs/evidence/codex-seat-permissions-2026-09-18.md`
P3.

## Files changed

- `docs/tech-debt/README.md` — added one row (following the existing
  `checkShellAliases` row's exact five-column format) recording the limits
  `checkCodexAgmsgWritableRoot` documents but does not address: (1) it reads
  only the user-level `config.toml` — codex profiles, a project-level
  `.codex/config.toml`, and `-c` overrides are never read, and a configured
  `profile` only earns a "not evaluated" note in the Detail
  (`codexWritableRootSuffix`), not a value the check actually consults; (2)
  whether codex expands `~` or a relative path in `writable_roots` is
  unconfirmed, so `pathCovers` treats any non-absolute root as never
  covering — a deliberate false-warn bias, not a confirmed true negative;
  (3) the claim that a guarded codex seat inherits the user config's own
  `sandbox_mode` is inferred from `permissionArgsForDriver` returning no
  sandbox flag for guarded seats (`codexSandboxReasons`'s reason (b)), not
  live-verified. Framed the impact as a false-positive risk (a project using
  one of these unread layers gets a misleading `warn`, not a missed real
  gap) and cited the plan's Non-goals/Assumptions as why each was deferred.
  `docs/tech-debt/README.md` is a `ROOT_ONLY_EXCLUSIONS` entry in
  `scripts/check-sync.sh`, so no `templates/base/` copy applies.
- `docs/evidence/codex-seat-permissions-2026-09-18.md` — appended one
  dated follow-up bullet under P3 ("workspace-write 下の codex 座席は
  agmsg に書けない"), in the same style as P1's existing 追記 bullet:
  notes that `ralph doctor`'s new "Codex sandbox (agmsg writable root)"
  check now warns on this same config gap via a static `config.toml` read,
  and that no live codex seat re-verification was done for this issue (Run
  A / A2 remain the evidence for the underlying failure/fix). Left the rest
  of P3 (and every other P-section) unchanged, per the handoff's "do NOT
  rewrite it" instruction.
- `docs/reports/sync-docs-2026-09-19-codex-agmsg-writable-root.md` — this
  report.
- `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` — no edit;
  `PR created` left unchecked per the handoff.

## Files checked (no changes needed — already correct)

- `README.md`, `AGENTS.md` — `grep -n doctor` on both files shows only the
  generic `ralph doctor [--strict]` description (README) and the
  `internal/cli/` repo-map line naming `doctor` alongside other subcommands
  (AGENTS.md); neither enumerates individual Check names, matching the
  precedent already confirmed for #162. No edit needed.
- `docs/specs/2026-08-01-org-runtime.md` — searched every `writable`,
  `agmsg`, `codex_verified`, and `doctor` hit (grep). The only Check with
  its own dedicated operational note is "Org codex model slugs" (#159's
  drift-observation procedure, lines 17–22), which exists because that
  Check tracks a *stale-default* concern the spec's own FR-2/model_pool
  section already owns. There is no enumerated "doctor checks" list, and
  neither the shell-alias check (#162) nor this writable-root check (#164)
  was added to this spec when they shipped — same precedent holds here. No
  edit.
- `docs/recipes/codex-setup.md` — scoped to `ralph doctor`'s hooks-trust
  and dispatcher-wiring checks only (`hooks=true`, `.codex/hooks.json`
  routing); does not claim to enumerate every Check and has no org/seat
  permission content, so the writable-root check is out of its scope. No
  edit (same conclusion as #162's sync-docs pass).
- `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md` —
  `grep -n doctor` returns no hits in either file. No edit.
- `docs/recipes/codex-seat-permissions.md` + `templates/base/docs/recipes/codex-seat-permissions.md`
  — already updated by the plan's Slice B; `diff` confirms the two copies
  are still byte-identical after this pass's other edits.
- `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`,
  `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md` — already updated by Slice
  B with the same writable-root-check sentence; `diff` across all four
  confirms exact byte match. The sentence's conditions (`codex_verified =
  true` + edits/autonomous role, or `sandbox_mode = "workspace-write"` +
  guarded role, `driver_pool` gate) match `codexSandboxReasons` /
  `codexSeatModesPossible` in the shipped code.

## Decided not to change (and why)

- Did not edit the recipe or skill prose the handoff said was already
  updated — read both against the shipped Go code (`codexSandboxReasons`,
  `codexNotNeededWhyA`/`codexNotNeededWhyB`, `codexWritableRootSuffix`) and
  found them accurate: the role/`driver_pool` gating, the
  `AGMSG_STORAGE_PATH` override note, and the "profile is not evaluated"
  caveat all match.
- Did not rewrite any other part of `docs/evidence/codex-seat-permissions-2026-09-18.md`
  P3 beyond the one appended bullet — it is a dated evidence record of Run
  A/A2, and those runs remain the only live-fire evidence for the
  underlying root-cause/fix; this check's own doctor-output evidence lives
  in the plan's Deviation notes (AC-7) instead, not in this file.
- Did not add an `docs/insights/events/` entry for this sync-docs pass:
  `.claude/skills/sync-docs/SKILL.md` still has no `insights-append.sh`
  step (same gap #162's sync-docs pass found, already tracked in
  `docs/tech-debt/README.md`'s "insight event の cycle スタンプ機構が脆弱"
  row).
- Did not touch `internal/cli/doctor_codex_writable_root.go` or its tests —
  the tech-debt row records known limits as documentation, not as a code
  fix; extending the check to read project-level config/profiles/`-c`
  overrides or to live-verify `~`/relative expansion is out of `/sync-docs`
  scope and is exactly what the plan's own Non-goals deferred.

## Gate results

Run after the two doc edits above, in this worktree:

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
