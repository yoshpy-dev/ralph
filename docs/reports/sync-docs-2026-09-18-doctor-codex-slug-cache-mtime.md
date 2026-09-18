# sync-docs report: doctor-codex-slug-cache-mtime

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-codex-slug-cache-mtime.md` (issue #159)
- Prior reports: `docs/reports/self-review-2026-09-18-doctor-codex-slug-cache-mtime.md`
  (Merge, no open findings), `docs/reports/verify-2026-09-18-doctor-codex-slug-cache-mtime.md`
  (Pass), `docs/reports/test-2026-09-18-doctor-codex-slug-cache-mtime.md` (Pass)

## Summary

`checkCodexModelSlugs` (`internal/cli/doctor_codex_models.go`) now appends a
freshness clause to its warn/pass Detail: `(cache written <UTC RFC3339>, <age>
ago)`, a `; cache may be stale — launch codex once to refresh, then re-run`
note past 24h, or `(cache changed while reading; freshness unknown —
re-run)`. Status values are unchanged; no CLI flag changed. The Scope items 4
and 5 doc updates (`.claude/skills/org/SKILL.md` + 3 mirrors, and
`docs/specs/2026-08-01-org-runtime.md` operating-note (d)) were already made
inline during `/work` (commit `fd1be42`, per the plan's Deviation notes) and
verified in sync by `/self-review` and `/verify`. This pass's job was the
wider sweep the task handoff named explicitly. **No files were changed in
this pass** — every candidate surface was already consistent with the new
Detail format, or correctly out of scope.

## Surfaces checked (no changes needed)

- **`README.md`** — the `ralph doctor [--strict]` row in the Commands table
  (line 127) describes the check generically (core-hash/block/settings-key/
  manifest drift, plus `--strict` exit semantics); it does not enumerate
  individual doctor sub-checks or their Detail text. `grep -n -i 'codex
  model\|model slug\|model_pool' README.md` finds nothing. No per-check
  freshness detail to go stale.
- **`docs/recipes/codex-setup.md`** (+ `templates/base/docs/recipes/codex-setup.md`,
  confirmed byte-identical by `diff`) — mentions `ralph doctor` three times
  (presence check, trust/untrusted warning, "resolve every doctor warning")
  but never the codex model slug check specifically; `grep -n -i
  'models_cache\|slug\|doctor'` finds no on-topic hit beyond those generic
  mentions. Nothing here describes Detail text for any specific check.
- **`.claude/rules/ralph/model-routing.md`** ("Org runtime model receipts",
  lines 92-95) — "`ralph doctor` warns when a slug is missing from codex's
  local model cache" is still accurate: the Status contract (warn on a
  missing slug) is unchanged by this PR, only the Detail string gained a
  trailing freshness clause. No update needed.
- **`AGENTS.md`** repo map — the `internal/cli/` bullet is a one-line
  subcommand list (`init, upgrade, eject, adopt, status, doctor, pack,
  insights, org, version`); it does not enumerate individual doctor
  sub-checks or Detail formats for any of them, consistent with "AGENTS.md is
  a map, not an encyclopedia." No update needed.
- **`templates/base/ralph.toml`** comments (lines 23-28) — say codex model
  entries "go stale when codex retires a model" and point at
  `~/.codex/models_cache.json`; this is a *reason* callout (why slugs can go
  stale), not a claim about the doctor check's cache-freshness *output*, so
  it does not contradict the new Detail clause. Left as is.
- **`docs/tech-debt/README.md`** — searched for any row referencing the
  doctor slug check's freshness gap or issue #159
  (`grep -n 'checkCodexModelSlugs\|mtime\|freshness\|#159'`). One row exists
  (line 130) but it is already struck and marked `RESOLVED 2026-09-17` for an
  unrelated batch (home-resolution error surfacing, stale doc comments in
  `internal/org`); it does not reference cache freshness. No open row to
  close, and this PR does not open one — matches the plan's Non-goals
  ("`docs/tech-debt/README.md` の変更(該当行なし)").
- **`docs/reports/walkthrough-2026-09-18-org-codex-slug-update-procedure.md`**
  and **`docs/plans/archive/2026-09-18-org-codex-slug-update-procedure.md`**
  (issue #156's closing artifacts) — historical pipeline artifacts describing
  the mtime-observation *procedure* as it stood before this PR (e.g. "`stat`
  で cache の mtime が観測時刻に更新されたことを確認"). Left unedited per the
  task handoff: they are a record of what was true when written, not living
  docs. The *current* procedure text lives in
  `docs/specs/2026-08-01-org-runtime.md` (d), which was already updated
  in-branch to point at the new `cache written` Detail field.
- **Plan's own Status/Progress checklist**
  (`docs/plans/active/2026-09-18-doctor-codex-slug-cache-mtime.md:115-123`) —
  already accurate: everything through "Test artifact created" is checked,
  "PR created" is correctly unchecked (no PR exists yet). No edit needed.

## Mirror/sync verification

No skill-mirror or cross-file content was changed in this pass, so
`sync-skills.sh` was not re-run. Re-confirmed the state left by `/work` and
`/self-review` directly in this worktree rather than trusting prior reports'
word:

```
$ diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md                 # exit 0, no output
$ diff .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md  # exit 0, no output
$ diff .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md  # exit 0, no output
$ ./scripts/check-skill-sync.sh   # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-sync.sh         # PASS: all files in sync. IDENTICAL: 157, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5
```

All four skill copies remain byte-identical; both sync gates pass.

## Files changed in this pass

None. Every documentation surface the change could plausibly affect was
already consistent, or is correctly excluded (generic doctor descriptions
with no per-check Detail text, a reason-only `ralph.toml` comment, a
resolved-and-unrelated tech-debt row, and #156's historical artifacts left
untouched per the task handoff).

## Known gaps

- Did not re-run `./scripts/run-verify.sh` or `go test ./internal/cli/...`;
  that is `/verify`'s and `/test`'s completed job (both cited above, Pass).
- AC-7 (`go test ./internal/cli/... -count=1` and `./scripts/run-verify.sh`
  green, PR body `Closes #159`) is `/pr`-step work per the plan's
  Implementation outline, not `/sync-docs` scope.
