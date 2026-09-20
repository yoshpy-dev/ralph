# sync-docs report: codex-model-observation-followups

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-model-observation-followups.md` (issue #173)
- Prior reports: `docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`
  (Merge, 2 MEDIUM + 5 LOW, all fixed in `6564680`; point 6 flags the
  drift this pass closes), `docs/reports/verify-2026-09-20-codex-model-observation-followups.md`
  (Pass; names the same drift and its insertion point),
  `docs/reports/test-2026-09-20-codex-model-observation-followups.md` (Pass)
- Scope: `git diff main...HEAD` on `fix/codex-model-observation-followups`,
  HEAD `4300307`. Code changes are confined to `internal/org/`.

## Summary

Read `internal/org/verbs.go` (`Stop`, `seatFromEvents`, `codexSpawnCorrelation`,
`codexSpawnInfo`, `observeStopModelReceipt`) and `internal/org/codex_session.go`
(`ObserveCodexEffectiveModel`, `codexRolloutCandidates`, `scanRolloutRecord`,
`isCtxDoneErr`) at HEAD against every doc passage naming spawn's or stop's
model-observation timing. One known drift (named by both self-review and
verify): nothing operator-facing stated that `ralph org stop`'s own single
look for a codex seat's session record is itself bounded by the same
up-to-8s budget as spawn's poll — before this branch stop's observation was
effectively unbounded-but-fast; this branch adds the same
`codexModelObserveTimeout`-derived `context.WithTimeout` to
`observeStopModelReceipt`, so the bound is now a new, real, statable fact.
Added one clause to each of the three surfaces the hand-off named. All other
named surfaces were re-read and confirmed still accurate; nothing else was
edited.

## Files changed in this pass

- `.claude/rules/ralph/model-routing.md` — one clause added to the existing
  "Org runtime model receipts" sentence describing stop's re-observation:
  `... appends one (that single look is itself bounded by the same
  up-to-8s budget as spawn's poll), so a seat can have an \`unknown\`
  receipt from spawn ...`. Placed at the exact point verify's report named.
- `templates/base/.claude/rules/ralph/model-routing.md` — the same clause,
  mirrored into the template copy's slightly different wording (it omits
  the meta-repo-only `internal/org/codex_session.go` path reference two
  sentences earlier; that reference was left untouched, only the new
  clause was added).
- `docs/recipes/codex-seat-permissions.md` — one clause added inside the
  autonomous-mode "model-retirement dialog" bullet, at the sentence that
  already says stop "looks once more": `... \`ralph org stop\` looks once
  more (bounded by the same up-to-8-s budget as spawn's poll) and, when it
  finds the seat's session record, appends the observed one ...`.
- `templates/base/docs/recipes/codex-seat-permissions.md` — the identical
  clause (the two recipe copies were byte-identical before and after this
  edit).
- `.claude/skills/org/SKILL.md` — one clause added to the "codex 座席の実効
  モデルは receipts に記録される" bullet's stop sentence: `... 取れなかった
  座席は \`stop\` 時に spawn と同じ最大 8 秒の上限でもう一度だけ探し、記録が
  見つかれば receipt を1件追記する ...`. Reflowed the surrounding four lines
  to stay within the file's existing ~65-78 display-column wrap (measured
  with `unicodedata.east_asian_width`, East Asian wide characters counted
  as 2; new lines measure 70/67/69/41).
- `.agents/skills/org/SKILL.md` — regenerated via `./scripts/sync-skills.sh`
  (mirrors `.claude/skills/` → `.agents/skills/`; confirmed byte-identical
  to the root copy after the run).
- `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md` — copied by hand from the
  root/`.agents/` copies (`sync-skills.sh` does not touch `templates/base/`,
  per agent-memory `feedback_sync_skills_leaves_templates_stale.md`);
  confirmed byte-identical to their root counterparts via `diff` before
  and after.
- `docs/plans/active/2026-09-20-codex-model-observation-followups.md` —
  checked off "Verification artifact created" and "Test artifact created"
  in the Progress checklist (both reports already exist and passed; the
  checklist was simply stale, not a Scope/AC edit). Added one Deviation
  notes bullet ("2026-09-20 sync-docs cycle 1") summarizing this pass's
  edits and the four judgment points decided against a doc change.

## Judgment points from the hand-off

1. **The known drift.** Closed — see Files changed above. Kept to one
   clause per surface, at the exact sentence that already makes the claim
   being qualified; did not restate the mechanism (poll interval, reason
   taxonomy, ctx cooperative-cancellation caveat) anywhere.
2. **Spawn's existing "looks for up to 8 s" wording.** Left unchanged. It
   was already accurate before this branch (spawn's poll was already
   ctx-bounded in shape, just not cooperatively-cancelled mid-pass) and
   remains accurate after (the plan's Non-goals: "挙動の約束(最大 8 秒、
   stop は 1 回)は変わらない" — behavior promises did not change, only the
   internal cooperativeness of the existing bound did). No qualifier
   needed; the sentence was already the accurate one, not a case of a
   promise that changed.
3. **A doc sentence for the rejected-retry fix (AR-3).** No doc change
   made, per the hand-off's stated default. Checked
   `docs/specs/2026-08-01-org-runtime.md` FR-9 (`grep -n 'FR-9'`): it
   states the three-value receipt shape and that `honored=true` requires a
   driver observation — it makes no claim about *which* spawn event
   supplies the compared values, so the pre-fix bug was never described
   there and nothing needs correcting now. Checked both recipe copies in
   full: neither states or implies that the commanded-model comparison
   follows the roster's latest state; the closest passage
   (`reported_effective_model`/`honored` under the autonomous-mode bullet)
   only describes the *outcome* shape, not the *source* of the compared
   value. No operator-facing sentence ever described the old bug, so there
   is nothing for this fix to correct in prose.
4. **`docs/tech-debt/README.md`.** No change. `grep -n '#173'
   docs/tech-debt/README.md` — no match anywhere in the file. The existing
   `ObserveCodexEffectiveModel` row (line 134, added by PR #174 for #165)
   describes the observer's dependency on codex's own undocumented
   session-record shape and ralph's `PromptFilePointer` text; nothing in
   this branch's diff touches that dependency (the ctx-cooperativeness and
   roster-vs-correlated-spawn fixes are orthogonal to the record-shape
   parsing risk the row names), so the row needed no edit — confirmed by
   re-reading it in full, not just grepping for keywords.
5. **Other surfaces.** Re-checked, no change:
   - `README.md` — no match for any codex-model-observation term
     (`grep -n` for the mechanism's vocabulary returns nothing); the file
     never described this mechanism to begin with.
   - `AGENTS.md` repo map, `internal/org/` line — already a short,
     feature-level description ("receipts, permission-mode envelopes,
     ..."); no individual source file or timing detail is named at that
     level for any other package concern either, so nothing to add.
   - `docs/insights/README.md` — its `honored` field row is the historical
     `source:pipeline` field (a different, `/work`-subagent-tier mechanism
     documented separately in `model-routing.md`'s "Standard flow
     delegation" section, not the org-runtime codex receipts this plan
     touches); confirmed by re-reading the row and its "see `driver`"
     pointer, not by keyword match alone.
   - `internal/org/prompts/lead.md` — no mention of spawn/stop timing at
     all; it defers all verb detail to the `/org` skill, which this pass
     already updated. Nothing to change.

## Gate results

| Check | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skill(s) in lock-step |
| `./scripts/check-sync.sh` | PASS — `DRIFTED: 0` (158 IDENTICAL, 11 TEMPLATE_ONLY, 5 KNOWN_DIFF, all pre-existing; `.claude/rules/ralph/model-routing.md`'s root-vs-template divergence is an existing, documented `KNOWN_DIFF`, unaffected by this pass's edit shape) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references found in templates |
| `go test ./internal/org/ -run TestSendDefaults -count=1 -v` | PASS — `TestSendDefaults_DocsMatchConstant` |
| `750ms`/`--enter-delay-ms` same-line constraint | Unaffected — this pass's edits are in a different section of both the skill and the recipe; `grep -n 'enter-delay-ms' .claude/skills/org/SKILL.md docs/recipes/codex-seat-permissions.md` shows the `750 ms` sentence untouched on its original single line in both files |
| Credential-shaped string scan | `git diff -- . ':!docs/plans/active/2026-09-20-codex-model-observation-followups.md' \| grep -niE "key\s*=\|token\s*=\|secret\s*=\|password\s*="` — no match. `.gitallowed` was not edited. |

## Known gaps

- Did not re-run `go test ./internal/org/... ` at large, `-race`, or
  `TMPDIR=/tmp` — those are `/test`'s completed job (Pass, cited above);
  this pass ran only the gates this task's constraints listed.
- Did not re-run the range secret scan (`./scripts/secret-scan.sh --range
  ...`) — that is a `/pr`-time pre-check; manually spot-checked the full
  diff for credential-shaped `key=value` strings (none, see Gate results
  above) and confirmed no `.gitallowed` edit was made.

## Insight event

Appended via `./scripts/insights-append.sh --slug
codex-model-observation-followups --flow standard --phase sync_docs
--verdict complete --source skill --driver claude --cycle 1`.
