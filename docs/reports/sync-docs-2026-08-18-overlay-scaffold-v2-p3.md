# Sync-docs report: overlay-scaffold-v2 Phase 3 (residual pass)

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Doc maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Scope: `git diff main...HEAD` on `feat/overlay-scaffold-v2-p3` (base `main`, HEAD `62631aa`). Slice 5 (commit `895ba8d`) already performed the main doc sweep (spec exit-code resolution, `AGENTS.md` repo map, README upgrade section, `docs/recipes/adding-a-language-pack.md`, `docs/tech-debt/README.md`). This pass checks the four fix-slice commits landed after slice 5 (`3d8461d`, `6b617dc`, `c785300`, `62631aa`) for doc drift they may have introduced or left unreconciled.

## Verdict

**No drift found.** Slice 5's doc sweep and the fix slices' own `docs/tech-debt/README.md` row updates (commit `6b617dc`) already account for every surface this residual pass was asked to check. One plan-hygiene gap was found and fixed (see below).

## What was checked

1. **`AGENTS.md` repo map after M4/M5 API deletions** — the repo map entry for `internal/upgrade/` (line ~89) names concepts (replace planner, block engine, settings 3-way merge, advisory diff, report writer), not the specific Go symbols (`SetFileUnmanaged`, `WithTemplateHash`, `FileStatePartial`, `BaselineStatusAvailable`) that verify's MEDIUM 5 finding had flagged as unreachable and that the fix slice (`3d8461d`) deleted. Confirmed: no doc references any of those four symbol names. Repo map still accurate.
2. **README's upgrade section vs. final flag set** — `## \`ralph upgrade\`` (README.md:126-141) documents exactly `--dry-run`, `--diff` (implying `--dry-run`), and `--pager auto|always|never`; states plainly "There is no `--force` flag." Cross-checked against `internal/cli/upgrade.go`/`upgrade_v2.go` flag registration: matches. No stale `--force` mentions found anywhere in README, `docs/recipes/`, `docs/quality/`, `.claude/rules/`, or `.claude/skills/` (the only `--force` hits are the unrelated `ralph-worktree.sh cleanup --force-branch` flag).
3. **Manifest API doc references** — grepped `SetFileUnmanaged`, `WithTemplateHash`, `SetFileWithBaseline`, `SetFileResolvedWithBaseline`, `addBaselineOnly` across `docs/`, `.claude/rules/`, `.claude/skills/`, `templates/`. All remaining hits are in archived plans and dated self-review/verify reports (`docs/plans/archive/2026-04-22-upgrade-detect-local-edits.md`, `docs/reports/self-review-2026-08-1{7,8}-*.md`, `docs/reports/verify-2026-08-1{7,8}-*.md`) — historical artifacts, correctly left untouched. No living doc (README, AGENTS.md, rules, skills, recipes, quality docs) references any of these deleted/dead symbols.
4. **`docs/quality/definition-of-done.md`** — no claims about upgrade interactivity, conflict resolution, or baselines; only references the post-implementation pipeline and generic evidence-redaction rules. Nothing to reconcile.
5. **`docs/recipes/codex-setup.md`** — re-confirmed the "Recovery from a half-applied upgrade" section's "represents ... as an `add` plus a `remove`" wording (lines 87-100). This describes the *net effect* of a template path rename under the current engine (`internal/upgrade/replaceplan.go`'s `OpCreate` + `OpDelete`, confirmed present; `diff.go`/`merge.go`/`baseline.go` confirmed deleted), not the deleted legacy engine's per-file `[a]pply/[k]eep/[e]dit` conflict-resolution vocabulary. Still accurate; no edit needed.
6. **Stale `baseline` references in living docs** — grepped `README.md`, `AGENTS.md`, `docs/recipes/`, `docs/quality/`, `.claude/rules/`, `.claude/skills/` (and their `templates/base/`/`.agents/skills/` twins) for `baseline`. Only hit: `.claude/skills/work/SKILL.md`'s "so the implementer starts from an unambiguous baseline" — generic English usage (starting point), unrelated to the deleted `.ralph/baseline/` scaffold mechanism. No change needed.
7. **`docs/tech-debt/README.md` fix-slice reconciliation** — the fix slices (`3d8461d`, `6b617dc`, `c785300`) already rewrote every tech-debt row that referenced the deleted legacy engine (the pack-rule-rename-detection row now describes `PlanCoreReplaceDesired`'s successor behavior with an explicit `<!-- UPDATED 2026-08-18 in feat/overlay-scaffold-v2-p3 -->` marker; the Codex-hooks-parity and `pre_bash_guard.sh` rows retargeted their "next touch" trigger from "Phase 3" to "next config change" since Phase 3 landed without touching either surface; a new row records the Phase-3 coverage loss from deleting the legacy test suite). No further edits needed.

## Fixed in this pass

- **Plan progress checklist** (`docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`) — "Review artifact created" and "Verification artifact created" were still unchecked even though `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md` and `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md` both exist and are committed (`618f55a`, `c785300`). Checked both. Added and checked a "Sync-docs artifact created" line (the shared `docs/plans/templates/feature-plan.md` doesn't carry this line either, matching the same gap noted and closed in the Phase 2 plan's own sync-docs pass).

## Verification after doc edits

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

## Files changed

- `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md` (checklist only)
- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p3.md` (this report)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` (appended `sync_docs` event)

## Known gaps

None. This was a residual/confirmatory pass; slice 5's doc sweep already covered the substantive drift from Phase 3's behavior changes.

---

## Cycle 2 (final cycle, 2/2)

- Date: 2026-08-18
- Scope: delta since cycle-1 (`7602317..HEAD`, HEAD `e21b1b1`). Commits reviewed: `df1a529` (cross-review triage), `fc8e9a9` (containment fixes: `ValidateRealParentChain` + exception-face/report write guards), `cee54b2` (cycle-2 self-review), `78d9039`/`4b53880` (cycle-2 LOW fixes, incl. `writeFileV2` doc-comment reword and a tech-debt row addition), `aa7fb9c` (verify cycle-2, ticked AC-1..AC-12), `e21b1b1` (test cycle-2 artifact).
- Verdict: **No drift found.**

### What was checked

1. **Write-containment model depth vs. living docs** — `fc8e9a9` adds `ValidateRealParentChain` (symlink-parent-chain validation) to `ApplyOps`, `writeFileV2`, and `WriteUpgradeReport`, extending the containment guard beyond the leaf-only `Lstat` check that already existed pre-cycle-2. Grepped `containment`, `symlink`, `ApplyOps`, `Lstat`, `ValidateRealParentChain` across `README.md`, `AGENTS.md`, `docs/specs/2026-08-17-overlay-scaffold-v2.md`, `docs/quality/`, `docs/recipes/`, `.claude/rules/`. No living doc describes the write-containment mechanism at implementation depth (leaf checks, parent-chain checks, or `ApplyOps` internals) — the one hit is the spec's `.claude/skills/` symlink-handling design note (line 179), which is about a different, unrelated symlink concern (Claude Code's own directory-discovery symlink support), not this write path. The spec's "Security considerations" section (line 172) states the path-traversal defense at design level ("shared validator `scaffold.CleanLocalRelPath`") without enumerating individual containment mechanisms — consistent with how it already omitted the pre-existing leaf-`Lstat`/`ApplyOps`-preflight checks from Phase 1/3 cycle 1. No living doc needs an update to reflect `ValidateRealParentChain`; the implementation-level explanation lives correctly in the code's own doc comments (`replaceplan.go`, `upgrade_v2.go`, `report.go`) and in the self-review/verify reports.
2. **`docs/tech-debt/README.md` row consistency after C2-1/C2-4** — verified both edits landed and cross-reference correctly: the rename/move-detection row's evidence column now reads `(M4, row reconciliation)` (`78d9039`, fixing the C2-1 finding that it pointed at the wrong ID), and the 5-LOW register row gained a 6th item naming the unused `in io.Reader` parameter with its file and ~25 call sites (`4b53880`, resolving C2-4's untracked-deferral gap). No other row was touched in this delta; the six rows fc8e9a9/cee54b2/78d9039/4b53880 did not need to touch (the git-hook-chaining/pager/`apErr` coverage-loss row from `6b617dc`, and the other rows edited in `3d8461d` at cycle 1) remain as they were and are unaffected by this delta.
3. **Plan Deviations/checklist coherence** — `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`: all twelve AC checkboxes (`AC-1`..`AC-12`) are ticked, with `aa7fb9c` adding inline cycle-2 strengthening notes on AC-4b and AC-9 (both now cite the `fc8e9a9` containment fixes) and a qualification note on AC-4c. The Progress checklist's "Review/Verification/Test/Sync-docs artifact created" rows are all ticked and match the existing artifacts under `docs/reports/`; "Plan reviewed" and "PR created" remain unticked, correctly outside this delta's scope. The Deviations section (slice-3/slice-4 git-hook-reinstall finding, slice-3 code-movement notes) is unchanged in this delta and still accurately describes the only mid-plan deviations — nothing in cycle 2 introduced a new deviation from the plan's scope.

### Verification after this pass

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

### Files changed (cycle 2)

- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p3.md` (this Cycle 2 section)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` (appended `sync_docs` cycle-2 event)

### Known gaps (cycle 2)

None. No living-doc edits were needed; the delta's doc-facing changes (tech-debt row updates, plan AC/checklist ticks, code doc-comment reword) were already made by the fix and verify commits themselves and are confirmed consistent above.

---

## Cycle 3 (final cycle, 3/3, after cap raise)

- Date: 2026-08-18
- Scope: delta since cycle-2 sync-docs (`b80b5f9..HEAD`, HEAD `f45463a`). Commits reviewed: `1bb8367` (cycle-2 cross-review triage rewrite), `41ad745` (dry-run exception-face preview + no-op short-circuit + spec NFR-1 amendment), `a0b9363` (cycle-3 self-review, one HIGH: C3-1), `6533227` (fix: veto no-op short-circuit on seed advisories and version change — closes C3-1/C3-2/C3-3/C3-4), `739c9c1` (docs: qualify cycle-1 AR references in the plan — closes C3-5), `4df2ce9` (cycle-3 verify, one LOW carried over from before `739c9c1` landed), `f45463a` (cycle-3 test artifact).
- Verdict: **Drift found and fixed.** README's `ralph upgrade` section had gone stale against `41ad745`'s behavior change (see below). Everything else in the assignment's residual checklist was already consistent.

### What was checked

1. **README's `ralph upgrade` section vs. the no-op/dry-run behavior change in `41ad745`/`6533227`** (residual check item 1) — `41ad745` made a fully-converged rerun write **zero files, including the dated report** (`finishNoOpUpgradeV2` only writes to stdout), and made `--dry-run` additionally preview the four "exception faces" (`AGENTS.md`, `.gitignore`, `.claude/settings.json`, the settings snapshot) that the core replace plan never covers. README.md:137 still claimed unconditionally "A dated report is written to `docs/reports/upgrade-<version>-<date>.md` summarizing every action" and README.md:141's exit-3 sentence still claimed unresolved-drift paths are "listed in the report and on stderr" — both false on a converged no-op rerun with residual drift (`finishNoOpUpgradeV2` explicitly skips the report and only writes to stderr in that case; see the self-review's C3-1/C3-2 analysis and `docs/specs/2026-08-17-overlay-scaffold-v2.md` NFR-1 as amended by `41ad745`). Fixed both sentences (see below) and added one clause to the `--dry-run` flag description noting it also previews the exception faces, so the flag docs match `renderUpgradeV2ExceptionFacePreview`'s actual scope. This is the drift the team lead's hand-off specifically flagged as a residual risk.
2. **Spec NFR-1 amendment (`41ad745`) reads coherently in context** — re-read NFR-1 alongside NFR-2 through NFR-5 and the plan's Design decisions section. The amended wording ("標準出力に no-op を明記。収束済みの再実行では日付付きレポートファイル自体を書かない — レポートは新たな適用結果があった実行でのみ生成される") is self-contained, matches the plan's AC-5 wording (itself brought into line by `6533227`/`739c9c1`), and matches `finishNoOpUpgradeV2`'s actual behavior confirmed in the cycle-3 verify report's "AC-5 / AC-10 under the new no-op semantics" section. No incoherence found.
3. **Tech-debt rows** — grepped `docs/tech-debt/README.md` for any row this delta's commits should have touched. None of the seven delta commits (`1bb8367`, `41ad745`, `a0b9363`, `6533227`, `739c9c1`, `4df2ce9`, `f45463a`) modify `docs/tech-debt/README.md`, and none needed to: C3-1/C3-2 (the no-op-swallows-advisories/version bugs) were genuinely **fixed** by `6533227`, not deferred, so no new row was warranted — consistent with the self-review's cycle-3 recommendation ("if the operator instead chooses to ship as-is, C3-1 and C3-2 must go to tech-debt" — they didn't ship as-is; they were fixed). Rows already reconciled in cycle 2 (`3d8461d`/`6b617dc` fixes, confirmed in this report's cycle-1/cycle-2 sections above) remain untouched and still accurate.
4. **Plan coherence (AC-5 reworded, AR refs qualified)** — `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`: AC-5 now reads "同一バイナリでの再実行が書き込みゼロの no-op(標準出力に no-op 明記、レポートファイルは書かない)" (`6533227`), verbatim-consistent with the amended NFR-1 and with README.md's now-corrected report-writing bullet. AC-4b and AC-9 now both read "cycle-1 cross-review AR#1/AR#2" (`739c9c1`), resolving the C3-5 finding (bare `AR#1`/`AR#2` was ambiguous once `1bb8367` overwrote the triage report's rows with cycle-2 findings using the same numbering). All twelve AC checkboxes remain ticked and match the artifacts under `docs/reports/`. Progress checklist rows (Review/Verification/Test/Sync-docs artifact created) are all ticked; "Plan reviewed" and "PR created" remain correctly unticked. No further plan edits needed.
5. **Spot-checked `docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md`** (touched by `1bb8367` in this delta) for internal consistency with its own commit message ("update ... for cycle 2") — the report's own header and rows describe cycle-2 findings; the C3-4 finding (insight-event `cycle:1` mislabel on the line `1bb8367` appended) was independently caught by the self-review and fixed in `6533227`(confirmed above in "What was checked" item 3 of this report's earlier cycle-1 section — grepped `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl`, the `cross_review` line now reads `"cycle":2`). No remaining mismatch.

### Fixed in this pass

- **README.md** (`## \`ralph upgrade\``) — three edits:
  - The "dated report" bullet now states the report is skipped on a true no-op rerun, with the exact stdout message it prints instead.
  - The `--dry-run` flag description now names the four exception faces it previews (previously described only as "the plan").
  - The exit-3 sentence now says unresolved-drift paths are listed on stderr always, but in the report only on runs that write one (previously implied the report always exists).

### Verification after doc edits

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

### Files changed (cycle 3)

- `README.md` (`ralph upgrade` section: no-op report semantics, dry-run exception-face preview, exit-3 report caveat)
- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p3.md` (this Cycle 3 section)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` (appended `sync_docs` cycle-3 event)

### Known gaps (cycle 3)

None. The README fix closes the one drift this cycle introduced; everything else in the residual checklist (spec NFR-1 coherence, tech-debt rows, plan coherence) was already consistent by the time this pass ran, because the fix commits (`6533227`, `739c9c1`) and the cycle-3 verify pass (`4df2ce9`) had already reconciled the non-README artifacts.

---

## Cycle 4 (final cycle, 4/4)

- Date: 2026-08-18
- Scope: delta since cycle-3 sync-docs (`07239cf..HEAD`, HEAD `e2aca9a`). Commits reviewed: `1f594c7` (cycle-3 cross-review triage update), `ae295f8` (triage summary-count correction), `b3c6307` (fix: classify `.codex/AGENTS.override.md` as seed per L3 overlay contract — cycle-3 cross-review AR#1), `d3a9fa6` (cycle-4 self-review artifact), `7f1c531` (fix: address phase-3 cycle-4 self-review findings — tech-debt row + doc-comment line-length fix + insight-event cycle mislabel fix + test-version-pin comment), `4aba6f3` (cycle-4 verify artifact), `e2aca9a` (cycle-4 test artifact).
- Verdict: **No drift found.**

### What was checked

1. **`.codex/AGENTS.override.md` ownership classification (`b3c6307`) vs. living docs describing `.codex/` or the L1/L3 ownership split** — `b3c6307` moves `.codex/AGENTS.override.md` from the `ownerForScaffoldPath` catch-all (core) to an explicit `OwnerSeed` case, matching the spec's classification. Confirmed against `docs/specs/2026-08-17-overlay-scaffold-v2.md` lines 41/43: the L1 core row already lists `.codex/`(`AGENTS.override.md` 除く)`(".codex/" excluding AGENTS.override.md)` and the L3 overlay row already lists `.codex/AGENTS.override.md` explicitly — the spec had this file classified as L3/seed from the start; `b3c6307` brings the code into alignment with a spec that was already correct, not the other way around. Also checked `AGENTS.md`'s repo map line 108 (`.codex/` bullet: lists `config.toml`, `agents/`, `hooks/`, `AGENTS.override.md`, `README.md` and notes root/`templates/base/.codex/` parity via `check-sync.sh`) — this line describes directory contents and the sync gate, not per-file owner classification, so `b3c6307`'s owner change doesn't touch anything this bullet claims. No doc edit needed.
2. **Tech-debt row added by `7f1c531`** (`docs/tech-debt/README.md`, new last row) — cross-checked its four claims against the repo: (a) "pre-b3c6307 commits... record `.codex/AGENTS.override.md` with owner=core" — matches `b3c6307`'s diff (the field only exists after that commit); (b) "`git tag --contains` on the first LayoutV2 commit returns no tags" — identified the manifest v3 ownership schema's introducing commit via `git log --all --oneline -S "LayoutV2" -- internal/scaffold/` (`75de545`, "feat: add manifest v3 ownership schema and shared path validator") and confirmed `git tag --contains 75de545` returns empty; the repo does carry 35 pre-existing release tags (`v1.0.0`..`v3.3.3`) for unrelated prior work, none of which contain this commit — so the tech-debt row's "no released binary ever produced a v2 manifest" claim holds; (c) the "Owner re-attribution... is the migration classifier's job (spec Phase 4)" pointer matches the plan's own Deviations entry (same commit, `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`: "レガシー(旧分類)manifest の再分類は Phase 4 の移行分類器のスコープ") and the plan's Non-goals ("旧レイアウト→v2 の移行ロジック(Phase 4)") — internally consistent, not a new or dangling forward reference; (d) evidence column paths (`docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md`, `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md`, `internal/cli/init.go`) all exist and contain the referenced content (triage cycle-3 AR#1, self-review Cycle 4, `ownerForScaffoldPath`). No correction needed.
3. **Plan Deviations section coherence** — the plan already carries its own note on the same `b3c6307` finding (added by that same commit, not this pass): "cycle-3 cross-review AR#1 の fix で発見: ...レガシー(旧分類)manifest の再分類は Phase 4 の移行分類器のスコープとし、本フェーズでは追わない". This note and the new tech-debt row (`7f1c531`) describe the same fact from two angles (plan-local deviation log vs. cross-plan tech-debt register) and do not contradict each other. Progress checklist: all rows through "Sync-docs artifact created" remain ticked; "Plan reviewed" and "PR created" remain correctly unticked.
4. **`7f1c531`'s other three changes** — (a) the `init.go` doc-comment line-wrap fix (splitting a URL-adjacent path reference across two lines) is a formatting-only change with no content delta, confirmed byte-for-byte content match aside from the line break; (b) the insight-event `cross_review` line's `cycle` field correction (`1` → `3`, self-review C4 finding) matches the actual pipeline history (this is cycle-3's cross-review, not cycle-1's — corroborated by the surrounding self_review/verify/test lines all stamped `cycle:3` in the same file); (c) the `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` comment/version-pin change is a test-only edit (`internal/cli/upgrade_v2_test.go`), out of doc-sync scope. None require a living-doc update.

### Verification after this pass

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

### Files changed (cycle 4)

- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p3.md` (this Cycle 4 section)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` (appended `sync_docs` cycle-4 event)

### Known gaps (cycle 4)

None. This was a residual/confirmatory pass on a small delta (one classification fix, one self-review fix-slice, two triage/report-only commits); no living-doc edit was needed because the spec already classified `.codex/AGENTS.override.md` as L3/seed before `b3c6307` landed, and the new tech-debt row is internally consistent with the plan's own Deviations note on the same finding.
