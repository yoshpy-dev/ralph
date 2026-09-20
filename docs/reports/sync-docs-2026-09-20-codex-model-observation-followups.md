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

## Cycle 2 (2026-09-20)

- Scope: `git diff main...HEAD` on `fix/codex-model-observation-followups`,
  HEAD `97a4429` (pipeline cycle 2 of cap 2, the final cycle). Prior
  reports (this cycle's sections): `docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`
  Cycle 2 (Merge once C2-H1 fixed; C2-H1/C2-L1/C2-L2/C2-L3 all fixed
  in-cycle, confirmed below), `docs/reports/verify-2026-09-20-codex-model-observation-followups.md`
  Cycle 2 (Pass, no new drift found — explicitly re-confirms cycle 1's
  clause), `docs/reports/test-2026-09-20-codex-model-observation-followups.md`
  Cycle 2 (Pass; names one still-open, non-blocking test-coverage item).
- Delta since cycle 1 (`7c95d3f`): test budgets only (`5e239b0`,
  `711cd9d`, `a1afc0c`) plus a comment-only change to two doc-comment
  enumerations in `internal/org/verbs.go` (`711cd9d`). No production
  behavior changed. Also landed: a cycle-1 `/cross-review` round
  (`e5c0aed`/`0f379da`, one ACTION_REQUIRED, test-only, fixed by
  `5e239b0`) and a cycle-2 self-review round (`bb1e79b`→`84b3387`, HIGH 1
  + LOW 3, all fixed in-cycle) — both already reconciled by this cycle's
  `/verify` and `/test` passes, re-read here rather than re-litigated.

### The eight doc files re-verified against HEAD — no change

Re-read all eight copies in full against the code at `97a4429`
(`internal/org/verbs.go`'s `observeStopModelReceipt`,
`codexSpawnCorrelation`; `internal/org/spawn.go`'s
`observeCodexSpawnReceipt`; `internal/org/codex_session.go`'s
`ObserveCodexEffectiveModel`), not just diffed against cycle 1's edit:
`.claude/rules/ralph/model-routing.md` + `templates/base/` copy,
`docs/recipes/codex-seat-permissions.md` + `templates/base/` copy,
`.claude/skills/org/SKILL.md` + `.agents/skills/org/SKILL.md` +
`templates/base/` × 2. The clause added in cycle 1 ("that single look is
itself bounded by the same up-to-8s budget as spawn's poll" / the
recipe's and skill's matching wording) is still accurate: `defaultCodexModelObserveTimeout`
is unchanged (`spawn.go:33`, `8 * time.Second`), `observeStopModelReceipt`
still derives its `obsCtx` from the same `o.codexModelObserveTimeout()`
helper Spawn's poll uses (`verbs.go:960-961`), and the cycle-2 production
change (`verbs.go`, comment-only) touches neither function's behavior.
This matches both `/verify` Cycle 2 (independently confirmed byte-identity
across all 8 copies via `cmp`, and the clause's accuracy via the same
code sites) and `/self-review` Cycle 2 (judgment point 6: "the doc clause
is accurate ... elides [nothing worth stating]"). Independently
re-confirmed here: `cmp`/`grep -c` across all 8 copies (see Gate results)
— no edit made to any of them.

### `docs/tech-debt/README.md` — two candidates considered, neither added

The hand-off named two items from this cycle's `/test` and `/self-review`
reports and asked whether either belongs in the existing codex-observer
row (line 134) as one clause. Decision: **no change**, for both.

1. **`observeStopModelReceipt`'s `o.Receipts.Read()` failure branch
   (`verbs.go:935-938`) has no dedicated test** (`/test` Cycle 2, "Test
   gaps": "Low risk (same silent-degrade contract, narrow race/corruption
   window)... not blocking"). This is a different risk from the one the
   existing row names: the row is about *codex's own* undocumented
   session-record format changing out from under the parser (an external,
   unversioned dependency that could silently break every codex-seat
   receipt repo-wide); this branch is a local I/O-failure path reading
   ralph's own `model-receipts.jsonl` on `stop`, pre-dating this branch
   (from PR #174/#165, not introduced by #173). Both this plan's cycle-1
   and cycle-2 `/test` reports independently characterize this branch as
   one instance of a generic, repo-wide pattern ("the same
   'untested filesystem-fault-injection branch' pattern repeated at every
   read call in this file (`findSeat`, `Status`, `Disband...`), not
   specific to this plan's changes" — cycle-1 report; "same shape as the
   append-failure case" — cycle-2 report). Folding a generic,
   already-acknowledged, low-probability gap into a row scoped to a
   specific external-format risk would blur what that row is tracking,
   and per this task's own instruction not to add a second row, there is
   no other place to put it that fits. The gap is already durably
   recorded once, in `/test`'s own report — adding it here would
   duplicate that record rather than rescue information that would
   otherwise disappear (the report is the primary record for this class
   of gap already; see also this plan's cycle-1 `/test` report making the
   identical judgment about four other pre-existing untested branches).
2. **`TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` can pass
   without reading the record's age if its 50ms budget is cut short**
   (self-review C2-L2). Not open: `711cd9d` already added the one-sentence
   fix self-review itself recommended, naming
   `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked`
   (`codex_session_test.go`, runs on `context.Background()`, cannot be
   cut short) as the deterministic pin for the age-exclusion rule —
   confirmed present in the comment at `spawn_test.go:2679-2686` at HEAD.
   `/test` Cycle 2 independently re-confirmed this under a 60ms
   per-scan-delay injection and explicitly calls it "not a new finding,
   just reconfirming the self-review's own characterization holds at
   HEAD". A resolved, already-documented-in-code redundancy is not tech
   debt.

### Testing-guidance doc for the three budget categories — none exists, none created

Checked whether any doc already discusses test timing or fixtures, per
the hand-off's own conditional ("if, and only if"): `.claude/rules/ralph/testing.md`
(general test-quality rules — coverage, edge cases, naming; no mention of
timing, budgets, fixtures, or flakiness), `.claude/rules/ralph/golang.md`
(general Go style — package layout, error handling, `go vet`/`go test`;
same, no mention), `docs/quality/definition-of-done.md`,
`docs/quality/quality-gates.md` (checklists and gate commands; same).
`grep -n -i 'timing\|budget\|fixture\|flak\|timeout'` across all four
returns nothing. Per the hand-off's own default, no such place exists, so
none was created; `testCodexObserveGenerousBudget`'s own doc comment and
the three-category classification rule at `spawn_test.go:220-241` remain
the documentation for this pattern, as they already were.

### Leftover reference sweep (#173 / AR-3 / AR-4)

```
grep -rl '#173\|AR-3\|AR-4' . --exclude-dir=.git | grep -v '^docs/reports/\|^docs/plans/\|^internal/org/'
```

Empty. The only remaining matches are inside `docs/reports/`,
`docs/plans/` (dated artifacts, expected) and `internal/org/*.go`
provenance comments (e.g. "issue #173 handoff item 9g", "Stop compares
against the spawn that actually launched (issue #173)") — all of these
cite which issue a fix or test came from, none describe either fixed
behaviour as still open or pending. Self-review Cycle 2's own Positive
notes already blessed this citation style ("`issue #173` at seven code
sites reads well ... strictly better than the bare `AR-3`/`AR-4` it
replaced").

### Files changed in this pass

- `docs/plans/active/2026-09-20-codex-model-observation-followups.md` —
  added one Deviation notes bullet ("2026-09-20 sync-docs cycle 2")
  summarizing this pass: the eight doc files re-verified with no change,
  the two tech-debt candidates considered and declined with reasons, the
  testing-guidance-doc check (none exists, none created), and the
  leftover-reference sweep result. Progress checklist and Readiness
  checklist were already accurate (Verification/Test artifacts ticked in
  cycle 1, correctly still true; `PR created` correctly still unticked).
  AC-1 through AC-6 were already ticked by the team lead's own cycle-2
  `/verify` adjudication (`bca703c`, before this pass started); AC-7
  correctly stays unticked (its `Closes #173` PR-body half is `/pr`'s
  job). Scope/AC untouched by this pass.

## Gate results (Cycle 2)

| Check | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skill(s) in lock-step |
| `./scripts/check-sync.sh` | PASS — `DRIFTED: 0` (158 IDENTICAL, 11 TEMPLATE_ONLY, 5 KNOWN_DIFF, all pre-existing, same as cycle 1) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references found in templates |
| `go test ./internal/org/ -run TestSendDefaults -count=1 -v` | PASS — `TestSendDefaults_DocsMatchConstant` |
| Eight-copy clause re-check | `cmp` confirms `.claude/skills/org/SKILL.md` byte-identical to its 3 mirrors and `docs/recipes/codex-seat-permissions.md` byte-identical to its template copy; `grep -c` confirms the clause text present in both `model-routing.md` copies (wrapped differently, same wording) and all 4 skill mirrors |
| `750ms`/`--enter-delay-ms` same-line constraint | Unaffected — no edit to the skill or recipe this cycle; `grep -n 'enter-delay-ms'` still shows both on one line in each file |
| Credential-shaped string scan | `git diff -- docs/plans/active/2026-09-20-codex-model-observation-followups.md \| grep -niE "key\s*=\|token\s*=\|secret\s*=\|password\s*="` — no match (this cycle's only changed file). `.gitallowed` was not edited. |

## Known gaps (Cycle 2)

- Same scope limits as cycle 1: did not re-run the full Go test suite,
  `-race`, `TMPDIR=/tmp`, or the range secret scan — all belong to
  `/test` (already Pass, cited above) and `/pr` respectively.
- This is the final pipeline cycle (cap 2/2) — no further sync-docs pass
  is expected after this one for the current cross-review round.
