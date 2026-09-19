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

## Secret scan

`./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"`
(commit `04b6df5`, the doc-edits-plus-report commit above) failed: two
"generic secret assignment" false positives, both in this branch's own
`docs/reports/{test,verify}-2026-09-19-codex-agmsg-writable-root.md`
hygiene sentences ("no fixture ... contains any `` `api_key=` ``/
`` `password=` ``/`` `access_token=` ``-shaped string") — the scanner's
generic-assignment pattern doesn't exclude backtick as a value character,
so naming the pattern in prose (to assert it is *absent*) trips the same
rule the prose is describing. Fixed by extending `.gitallowed` with a
narrow ERE requiring the closing backtick immediately after `=` (so a real
secret pasted into a backtick span, which would have a value between `=`
and the closing backtick, still matches) — same shape as the two existing
entries for `internal/cli/doctor_shell_alias_test.go`'s fixture values
(commits `5a91cff`/`00eb183` on `main`). The first spelling of my own
allowlist comment (commit `90dc4be`) quoted the flagged sentences verbatim
without breaking the pattern names, which self-matched the very rule it
was adding (the same pitfall `00eb183` fixed for the shell-alias entry);
corrected in a follow-up commit (`ee70868`) rather than amending, per this
repo's commit-strategy rule. Final state: `secret-scan.sh --range
"$(git merge-base HEAD main)..HEAD"` exits 0 (re-run after `ee70868`,
covering all commits through the `.gitallowed` fix itself).

## Progress checklist (cycle 1)

- Plan's `PR created` checkbox left unchecked, per the handoff — `/pr` has
  not run yet.

---

## Cycle 2

- Date: 2026-09-19
- Pipeline cycle: 2 (of cap 2)
- Branch: `feat/codex-agmsg-writable-root`, HEAD `c83ce78` (base `main`);
  `git status --porcelain` empty before starting.
- Prior reports for this cycle: `docs/reports/self-review-2026-09-19-codex-agmsg-writable-root.md`
  (Cycle 2 section, merge), `docs/reports/verify-2026-09-19-codex-agmsg-writable-root.md`
  (Cycle 2 section, PASS), `docs/reports/test-2026-09-19-codex-agmsg-writable-root.md`
  (Cycle 2 section, PASS), `docs/reports/cross-review-triage-codex-agmsg-writable-root.md`.
- Range since cycle 1 (`04b6df5`/`b6e85d0`): cross-review cycle-1 fixes and
  their docs (`ab09dbc`…`4c891ec`), self-review cycle-2 fixes and docs
  (`4c48ae3`…`235abb0`), the fresh AC-3/AC-7 evidence and `/org` skill
  temp-root note (`a16e3f6`), and two test-only commits closing coverage
  gaps (`5e939d1`, `c83ce78`).

### Files changed

- `docs/tech-debt/README.md` — extended the `checkCodexAgmsgWritableRoot`
  row's "Three further limits" clause to a fourth: `$TMPDIR` and
  `AGMSG_STORAGE_PATH` are read from doctor's own process environment
  (`codexSandboxEnvFromOS`), not the herdr seat's pane, and the Detail
  names this only in the branches where the value actually drove the
  result (`codexWritableRootSuffix`'s override note,
  `codexImplicitRootDetail`'s `$TMPDIR` note) — not when doctor's own
  environment simply lacks a value the pane's has. This limit is already
  recorded as an Assumption in the plan ("座席が見る環境変数は herdr の
  pane のもので、doctor のプロセスと一致するとは限らない") but had no
  tech-debt row entry; it was missing from the row's "Three further
  limits" enumeration even though the other three are all documented
  there. Left Impact/Trigger unchanged in substance (only extended their
  wording enough to cover the fourth case) — the existing "false positive
  (misleading warn), not a missed real gap" framing still holds: doctor's
  own environment lacking a value the pane's pane has can only make the
  check consider fewer implicit roots or the wrong (default) store
  directory than the pane's own send.sh would use, which biases toward an
  unnecessary `warn`, not a silent `pass` over a real gap. Extended the
  Evidence column with the two newly-cited functions
  (`codexSandboxEnvFromOS`, `codexImplicitRootDetail`). Kept it one row,
  per the handoff.

### Files checked (no changes needed)

- **`internal/cli/doctor.go`'s Check 11b comment** (item 1 of the handoff)
  — read against the full doc comment on `checkCodexAgmsgWritableRoot`
  (`internal/cli/doctor_codex_writable_root.go`). The comment is shorter
  than the full doc comment it points readers to (it doesn't spell out the
  `model_pool` gate, the protected-directory rule, or the implicit
  temp-root rule), but it does not claim anything the code contradicts:
  "warns when a role that could actually run under workspace-write exists
  but no writable root covers the agmsg store" is still an accurate
  summary of the warn condition — reason (b) (a guarded seat inheriting
  `sandbox_mode = "workspace-write"` from the user's own config) still
  describes a seat that actually runs under `workspace-write`, even though
  ralph itself passes no sandbox flag for it; "no writable root covers"
  does not distinguish explicit vs. implicit roots or the protected-dir
  exclusion, but nothing said is false about either. Verdict: **shorter,
  not inaccurate** — no edit (this is code, not sync-docs' to touch
  regardless).
- **Recipe bullet vs. skill bullet vs. Go `Detail`/doc-comment text** (item
  2) — re-read `docs/recipes/codex-seat-permissions.md`'s "Add the agmsg
  database…" bullet and `.claude/skills/org/SKILL.md`'s writable-root
  sentence (both already updated this cycle by commits `a49763a`/`4c891ec`/
  `a16e3f6`, before this pass) against `codexSandboxReasons`,
  `codexSeatModesPossible`, `codexModelPermittedForRole`,
  `codexImplicitWritableRoots`, and `codexWritableRootSuffix`. All four
  claim categories the handoff named — (a) when a root is needed
  (`codex_verified` + edits/autonomous role, or `sandbox_mode =
  "workspace-write"` + guarded role, both gated to "a role that may use a
  codex model" and to `driver_pool`/`model_pool`), (b) what counts as
  covering (explicit root; `/tmp`/`$TMPDIR` by default unless excluded;
  never through `.git`/`.agents`/`.codex`), (c) what is read (user-level
  `config.toml` only; no profile/project/`-c`), (d) the store-location rule
  (`AGMSG_STORAGE_PATH` override, else the default `db` dir) — match the
  code exactly in both the English recipe and the Japanese skill sentence.
  No sentence in either claims more than the code does. No edit.
- **`docs/evidence/codex-seat-permissions-2026-09-18.md` P3** (item 5) —
  unchanged since `b6e85d0` (`git diff b6e85d0..HEAD` is empty for this
  file). Re-read the cycle-1 follow-up bullet: it says only that a static
  doctor check now detects the missing writable root and that no live seat
  re-verification was done for issue #164, with Run A/A2 remaining the
  evidence for the underlying failure/fix — still accurate, does not
  overclaim. No edit.
- **`README.md`, `AGENTS.md`, `docs/specs/2026-08-01-org-runtime.md`,
  `docs/recipes/codex-setup.md`, `docs/quality/definition-of-done.md`,
  `docs/quality/quality-gates.md`** (item 4) — `git diff b6e85d0..HEAD`
  empty for all six; nothing changed since cycle 1's confirmation that none
  of them enumerate individual doctor Checks. No edit.
- `docs/recipes/codex-seat-permissions.md` + `templates/base/` copy,
  `.claude/skills/org/SKILL.md` + 3 mirrors — already updated in this cycle
  by prior commits (`a16e3f6` for the temp-root note); `diff`/`cmp` across
  all mirror pairs still byte-identical (see Gate results below). No
  further edit.
- `.gitallowed` — not touched, per the handoff; the cycle-2 narrowing
  (`e2c64c3` on `main`, inherited via this branch's earlier rebase-free
  history) is the orchestrator's, not sync-docs'.

### Decided not to change (and why)

- Did not touch the plan's AC-3/AC-7 text or Deviation notes — the
  handoff's item 3 scope is the tech-debt row; the plan's own drift-closing
  edit already landed in `a16e3f6` (recorded in the plan's own Deviation
  notes for verify cycle 2), and re-reading it against the current
  `checkCodexAgmsgWritableRoot` outcome chain confirms it now enumerates
  all 9 outcomes (`model_pool` pass, implicit temp-root pass,
  blocked-ancestor-named warn included).
- Did not add a `docs/insights/events/` entry for this cycle-2 pass — same
  gap noted in cycle 1 (`.claude/skills/sync-docs/SKILL.md` still has no
  `insights-append.sh` step; tracked in `docs/tech-debt/README.md`'s
  "insight event の cycle スタンプ機構が脆弱" row).
- Did not touch `internal/cli/doctor_codex_writable_root.go` or its tests —
  documenting a known limit is not a code fix, and extending the check to
  read the seat's own pane environment (rather than doctor's process
  environment) would need doctor to inspect a running seat's process,
  which the plan's Non-goals rule out (see the tech-debt row's own Why
  deferred column, extended this cycle to say so).

### Gate results (cycle 2)

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

`docs/tech-debt/README.md` is a `ROOT_ONLY_EXCLUSIONS` entry in
`scripts/check-sync.sh` (confirmed by name in the script's exclusion list),
so the one file this cycle changed has no `templates/base/` copy to keep in
step; `check-sync.sh`'s pass above carries over unaffected by this cycle's
edit. No skill files were touched this cycle either, so
`check-skill-sync.sh`'s pass also carries over unaffected.

### Secret scan (cycle 2)

`./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"`
(merge-base `2c511a4`, covering all commits on this branch through
`d18fc36`, the tech-debt-row-plus-report commit above) → exit 0. No
`.gitallowed` change was needed this cycle.

### Progress checklist (cycle 2)

- Plan's `PR created` checkbox left unchecked, per the handoff — `/pr` has
  not run yet.
