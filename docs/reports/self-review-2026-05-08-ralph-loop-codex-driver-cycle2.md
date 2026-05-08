# Self-review report: ralph-loop-codex-driver — cycle 2 (Codex ACTION_REQUIRED fix)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Reviewer: Claude Code (`reviewer` subagent)
- Scope: cycle-2 delta only — commits `91232dc` (P1/P2 fix) and `4e964c0`
  (sync-docs). Cycle-1 review (HEAD~5 shape) is in
  `docs/reports/self-review-2026-05-08-ralph-loop-codex-driver.md`. Diff
  quality only — no spec compliance, no test execution.

## Evidence reviewed

Delta confirmed via `git log --oneline main..HEAD | head -20`:

```
91232dc fix: address Codex cross-review ACTION_REQUIRED findings
4e964c0 docs: sync subagent-policy + tech-debt for Phase 2 Loop driver
3351df2 fix: surface Loop driver in ralph status + AGENTS map
085bad7 fix: address MEDIUM self-review findings before /verify
…
```

Files inspected for cycle-2 delta:

- `scripts/ralph-cli-driver.sh` — new `count_triage_findings` helper (lines
  25–54).
- `scripts/ralph-pipeline.sh:800-813` — Outer Loop replaces inline
  `grep -c '<CATEGORY>'` with the helper; counters initialised to 0 at
  L769–771; consumed at L827 (`-gt 0`), L833, L838.
- `tests/test-ralph-cli-driver.sh:216-313` — new Test 7 (9 assertions:
  3a-i…iii clean, 3b-i…iii populated, 2c fallback, 1d missing-file).
- `docs/recipes/ralph-loop.md` (and `templates/base/` mirror) — TOML/shell
  asymmetry callout (lines 176–215 of new file).
- `.claude/skills/loop/SKILL.md` (+ `.agents/` + both templates) — same
  callout in Japanese.
- `.claude/rules/subagent-policy.md` (+ template) — `RALPH_LOOP_DRIVER`
  paragraph (cycle-1 sync-docs).
- `docs/tech-debt/README.md` — RESOLVED row + 2 new declined-MEDIUM rows
  (cycle-1 sync-docs).
- `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md` — single
  checklist tick for AC-6 follow-up.

Mirror integrity (`scripts/check-sync.sh`): `IDENTICAL: 145 / DRIFTED: 0`,
verified separately with `cmp` on the six dual-tracked files in this delta.

Probe of `count_triage_findings` behavior (sourced from
`scripts/ralph-cli-driver.sh`, four hand-crafted fixtures in `/tmp`):

| Fixture                                              | AR  | WC  | DI  | Verdict |
|------------------------------------------------------|-----|-----|-----|---------|
| Empty body, summary `…ACTION_REQUIRED=0…`            | 0   | 0   | 0   | OK      |
| Real findings, summary `…ACTION_REQUIRED=2,WC=1,DI=2`| 2   | 1   | 2   | OK      |
| Same, key order `WC=3, AR=1, DI=2`                   | 1   | 3   | 2   | OK      |
| Body text "ACTION_REQUIRED=99", **no summary line**  | **99** | 0 | 0 | **WRONG** (1 expected) |
| Body text "ACTION_REQUIRED=99", summary present and =2| 2  | 1   | 0   | OK      |
| Partial summary `…ACTION_REQUIRED=0` (WC/DI omitted) | 0   | **0** | **0** | **WRONG** (WC=2, DI=0 expected) |
| Missing file                                         | 0   | 0   | 0   | OK      |

The two **WRONG** rows above are the reachable-but-unlikely failure modes
called out in MEDIUM #1.

Cycle-1 closure check: cycle-1 raised 5 MEDIUM. Three (#1
`RALPH_CLAUDE_REVIEWER_MODEL` symmetry, #3 `.stderr` surface, #5 globals
contract) closed in `085bad7`; #4 (`pick_reviewer` fallback) and the
original #1' (preflight `--version` rigor) declined and registered in
`docs/tech-debt/README.md` rows 4 and 5 by `4e964c0`. None reopened by
the cycle-2 delta.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | null-safety / robustness | `count_triage_findings` will mis-count when (a) the agent fails to emit the canonical `After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N` summary line, **and** (b) any line above the table contains the substring `ACTION_REQUIRED=<digits>` (e.g. a reviewer finding text like "old parser counts ACTION_REQUIRED=2"). The probe above shows this returns `99` for a clean report whose body merely *mentions* `ACTION_REQUIRED=99` in prose — an order-of-magnitude over-count that re-introduces the same "spurious Inner Loop regression" failure mode the P1 fix was supposed to close, just under a slightly narrower trigger condition. The summary-line format is **not codified anywhere** I could find — `.claude/skills/cross-review/SKILL.md` does not prescribe it, `prompts/adversarial-claude.md` does not prescribe it; only the helper's own header comment calls it "canonical". The current real triage fixture (`docs/reports/cross-review-triage-2026-05-08-ralph-loop-codex-driver.md:12`) emits it, but agent output is not contractually guaranteed to keep doing so. | `scripts/ralph-cli-driver.sh:42` (`grep -m1 -E 'ACTION_REQUIRED=[0-9]+'` is anchored only to the regex, not to a leading `- After triage:` marker); `.claude/skills/cross-review/SKILL.md` (no `After triage:` mention); `tests/test-ralph-cli-driver.sh:288-310` (Test 7c covers no-summary fallback but not the no-summary + body-mention combination). | Two complementary fixes: (1) tighten the regex to require the prefix, e.g. `grep -m1 -E '^- After triage: ACTION_REQUIRED=[0-9]+' "$_file"`. The leading `- ` plus the literal phrase makes accidental body-text collision essentially impossible. (2) Promote the summary line to a contract by adding it to the cross-review SKILL.md output checklist (line 96 area, alongside "All findings in their classified sections"). Together they make the helper's docstring claim of "canonical" actually true. **Not blocking for cycle 2** because the bug is reachable only if the agent omits the summary line **and** a reviewer phrases their finding with `ACTION_REQUIRED=N`, which is unlikely on this PR's clean report — but it is the same class of bug the fix targeted. |
| LOW | maintainability | Helper's awk fallback pattern `f && /^\|/ && !/^\| *# / && !/^\| *-+/` is correct for the table shapes shipped in the templates, but the negative-lookahead-by-substring is fragile. A future template author who renames the header column from `# | Reviewer finding | …` to `No. | Finding | …` will silently start counting the header row as a finding, off-by-one in every section. The exclusion is a structural property of "the row whose first cell is `#` followed by space" — encode that more directly. | `scripts/ralph-cli-driver.sh:51`; reference fixtures `tests/test-ralph-cli-driver.sh:233-244` (header is `\| # \|`). | Either (a) skip the **first** `^\|` row after each `## <CATEGORY>` heading unconditionally (it is by convention always the column header), or (b) skip both row 1 and row 2 (header + separator) and start counting from row 3. Both are O(1) state changes in the awk script and remove the substring assumption. Worth doing the next time the helper is touched, not now. |
| LOW | typo / readability | The pipeline-side comment block at `scripts/ralph-pipeline.sh:800-807` (and its template mirror) reads well but uses `≥2 matches` (Unicode `≥`, U+2265) where the surrounding scripts use ASCII `>=` everywhere else. The repo also runs a mojibake guard (`.claude/hooks/check_mojibake.sh`); U+2265 is not in the U+FFFD class so the hook will not fire, but consistency-of-style argues for `>=`. | `scripts/ralph-pipeline.sh:803` (`reports ≥2 matches`); compare with same file e.g. `scripts/ralph-pipeline.sh:827` (`-gt 0`). | Replace `≥2` with `>=2` in both root and template copies. One-character noise-level change. |
| LOW | maintainability | `count_triage_findings` returns the *raw integer string* via `printf '%s\n'`. If `${_n:-0}` ever evaluates to a non-numeric value (e.g. `cut` returns an empty string and the default fires — already handled — but a future change to the regex could let through a leading `+` or letters), `[ "$_action_required" -gt 0 ]` at `scripts/ralph-pipeline.sh:827` would error out under POSIX `sh` with `integer expression expected`, and because the `if` guard treats the non-zero exit as "ran the body of the truthy branch" on some shells (dash, ash differ from bash here) the result is nondeterministic. Today this is unreachable because the regex is `[0-9]+`, but the function would be more robust if it normalised the output via e.g. `_n=$(printf '%s' "${_n:-0}" | tr -cd '0-9'); printf '%s\n' "${_n:-0}"`. | `scripts/ralph-cli-driver.sh:44-45`; consumer at `scripts/ralph-pipeline.sh:827`. | Optional hardening for next maintenance pass; not blocking. |

No CRITICAL or HIGH findings. Pipeline must NOT stop before `/verify`.

## Positive notes

- The fix lives in a **named, separately-testable helper** rather than
  inline in the orchestrator. That is the right shape for the bug class
  Codex flagged — the original inline `grep -c` could not be unit-tested
  without spinning up a triage report end-to-end. Test 7's 9 assertions
  catch all three of the regressions that mattered (template-only false
  positive, summary-preferred parse, missing-file safety). Total
  47 assertions overall as advertised in the commit message.
- The `if [ ! -s "$_file" ]; then printf '0\n'; return 0; fi` early
  return is the right shape: an empty / missing file is functionally
  equivalent to "no findings" and a 0 keeps the downstream `-gt 0`
  comparisons safe. Combined with the L769–771 caller-side initialisation
  to 0, the counter chain is null-safe.
- The recipe rewrite (`docs/recipes/ralph-loop.md`) is genuinely clearer
  than the cycle-1 version — the "two options" framing (Go binary OR
  exported env var) plus the "Important asymmetry" callout matches how
  operators actually onboard. The `ralph doctor reports the effective
  driver and source` sentence is a useful escape hatch. P2 is closed
  cleanly without code change, which is the cheapest possible fix.
- Mirror discipline held: every dual-tracked file (root + `templates/base`)
  is byte-identical (`cmp` verified for all six). `scripts/check-sync.sh`
  reports `DRIFTED: 0`. No pattern of "edit only one side" regressed.
- `4e964c0`'s `tech-debt` row update preserves the original line via
  strikethrough rather than deleting it — that traceability shape matches
  the pre-existing `<!-- RESOLVED 2026-05-07 in commit 79d7a73 ... -->`
  comment style elsewhere in the file. Future maintainers can still
  reconstruct *what was originally promised vs. what shipped*.
- Cycle-1 closures hold: none of the three MEDIUM fixes from `085bad7`
  are reverted, weakened, or shadowed by the cycle-2 changes.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `count_triage_findings` summary regex `grep -m1 -E 'ACTION_REQUIRED=[0-9]+'` is anchorless and the canonical `After triage:` line is not contracted in `.claude/skills/cross-review/SKILL.md`. A clean report whose body merely *mentions* `ACTION_REQUIRED=N` in prose returns `N` — the same class of failure the P1 fix targeted, just under a narrower trigger | Reachable false-positive Inner Loop regression on a clean review; cycle counter still bounds the runaway, but the 2nd cycle is wasted when no real findings exist | Not blocking for the cycle-2 hand-off; reachable only when the agent omits the summary line **and** a reviewer phrases a finding with `ACTION_REQUIRED=N`, which is rare in practice on a fresh review | First report of "clean cross-review still triggered Inner Loop regression"; or whenever the triage report shape is next edited | This report MEDIUM #1, `scripts/ralph-cli-driver.sh:42`, `.claude/skills/cross-review/SKILL.md` |

_(Single row added; appended to `docs/tech-debt/README.md` is recommended
in the `/sync-docs` step rather than here — flagged so the doc-maintainer
can pick it up.)_

## Recommendation

- Merge: **MERGE** — no CRITICAL or HIGH findings. The pipeline must NOT
  stop. Proceed to `/verify`.
- Both Codex ACTION_REQUIRED findings (P1 parser, P2 TOML/shell asymmetry)
  are addressed with appropriate evidence: P1 via a tested helper that
  passes the four explicit edge cases shipped, P2 via a documentation
  callout plus the existing `ralph doctor` source-of-truth line. Cycle-1
  MEDIUM closures are not reverted.
- Follow-ups (non-blocking):
  - Tighten `count_triage_findings`'s summary regex with a `^- After
    triage: ` prefix anchor and add the summary line to the cross-review
    SKILL.md output checklist so the helper's "canonical" claim becomes
    contractual (MEDIUM #1).
  - Replace `≥2` with `>=2` in `scripts/ralph-pipeline.sh:803` and its
    template mirror (LOW #3) — bundle with any nearby touch.
  - Consider hardening the awk fallback to skip the first 2 rows after
    each heading rather than substring-excluding `# ` and `-+` (LOW #2).
  - Hand off the new MEDIUM #1 row to `/sync-docs` for inclusion in
    `docs/tech-debt/README.md`.
