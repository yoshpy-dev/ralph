# Review report: doctor-agmsg-store-in-project (self-review, cycle 1)

- Date: 2026-09-27
- Plan: docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD` at 5fb0b4e (code commit 24e455f slice A, docs commit c71a93b slice B, plan commits 8ee7762, 6fcab50, 5fb0b4e), plus a full read of `internal/cli/doctor_codex_writable_root.go`. No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- `git diff main...HEAD`: 12 files, +471/-66. At the start of the review the worktree HEAD was 5fb0b4e and `git status --porcelain` was empty.
- In the check file, the diff only adds two `if inProject { ... }` branches in front of the existing `return`s in `codexWritableRootDetail` (`internal/cli/doctor_codex_writable_root.go:819-825`, `:831-837`). The two unconditional format strings are unchanged, so a Detail for a store outside the project is byte-identical to main.
- Test call sites: 51 removed call lines in `doctor_codex_writable_root_test.go` plus 2 in `doctor_codex_writable_root_unix_test.go`, all rewritten to pass `""` for `projectDir`. Every call site that passes a real `projectDir` is in the new block at `:1911-2131`. Apart from the call-site edits, the only test-file change is that new block.
- `cmp`: the four `/org` SKILL.md mirrors are byte-identical, and both copies of `codex-seat-permissions.md` are byte-identical.
- Product run. I built the branch with `go build -o <scratchpad>/ralph ./cmd/ralph` and ran it against a scratchpad project (`ralph.toml` with codex in both pools and `codex_verified = true`), with a fake `HOME`, `RALPH_ORG_AGMSG_HOME`, and `~/.codex/config.toml` set to `exclude_slash_tmp = true`:
  - store `<proj>/agmsg/db`: warn. The in-project wording appears, and neither `seat under workspace-write cannot send` nor `; add <store>` appears.
  - `AGMSG_STORAGE_PATH=<proj>/store2/` (a directory that does not exist, with a trailing slash): warn with the in-project wording. The store is trimmed to `<proj>/store2` and the override suffix is kept at the end.
  - store `<proj>/.agents/agmsg/db`: warn with the unchanged unconditional wording (AC-2).
  - doctor run from a symlink to the project (`cd <scratch>/projlink`): warn with the in-project wording. The project is displayed as `.../projlink` and the store as `.../proj/agmsg/db`. The decision is correct; see the Observations below.
  - Without `exclude_slash_tmp`, every scratchpad fixture passes through the implicit `/tmp` root, because the scratchpad is under `/private/tmp`. So the config-absent in-project form could not be exercised through the binary here. I read it from the format string at `:820-824`.
- Where seats get their cwd: `internal/cli/org.go:324`, `:436` (`--cwd` is required), `.claude/skills/org/SKILL.md:120`, `:129`, `:182` (every example passes `--cwd .`), `internal/org/prompts/*.md` (no mention of worktrees), `docs/specs/2026-08-01-org-runtime.md:167` (worktree assignment granularity is still an open question), and `scripts/ralph-worktree.sh` (the caller chooses `--path`; this repo's task worktrees are at `<project>/.claude/worktrees/<slug>`).

## Findings

| # | Severity | Location | Finding |
|---|----------|----------|---------|
| L1 | LOW | `internal/cli/doctor_codex_writable_root.go:821-822`, `:833`; `docs/recipes/codex-seat-permissions.md:69-70` (+ template); `.claude/skills/org/SKILL.md:245-246` (+ 3 mirrors) | The in-project Detail contrasts the two kinds of seat by project membership instead of by what the seat's cwd contains. It says "the agmsg store is inside this project (P), so a codex seat whose working directory contains it can already write it; a codex seat running elsewhere (for example a task worktree) cannot…". Task worktrees in this repo live at `<project>/.claude/worktrees/<slug>`, which is inside P. So "elsewhere", read right after "inside this project", suggests "outside P", and only the parenthetical example corrects that reading. The recipe ("a seat running elsewhere") and the skill ("別の作業ディレクトリ") copy the same contrast. **Fix:** "a codex seat whose working directory does not contain it (for example a task worktree)", and 「作業ディレクトリが保存先を含まない座席(task worktree など)」. Optional: in the in-project forms, "needed because <reason>" now comes right after the recipe pointer. Moving it to just after the "no writable root covers" clause would make it clear which thing is "needed". |
| L2 | LOW | `internal/cli/doctor_codex_writable_root.go:810-813`, callers `:986`, `:1008` | `codexWritableRootDetail` takes the pair `inProject bool, projectDir string`. This allows an invalid state: `(true, "")` renders "inside this project ()". The doc comment then presents a caller convention as if it were an invariant ("inProject is always false when root != "" (a pass) or when projectDir is "" -- both cases leave this parameter unused"), and "this parameter" could mean either argument. The signature is 163 characters, the longest `func` line in the file. **Fix:** replace the pair with one `containingProject string` parameter, where `""` means the store is not in the project. The pass call then passes `""` and the sentence is no longer needed. |
| L3 | LOW | `internal/cli/doctor_codex_writable_root.go:905-918`, `:888-891` | Four problems in the new doc paragraph on `checkCodexAgmsgWritableRoot`. (a) It justifies the `""` case with "since codexRootCoverage("", ...) -- root is not absolute -- always reports covers=false". That call never happens: `:1002` guards with `if projectDir != ""`, so the comment names the wrong mechanism. (b) "(or one that does not cover the store)" leaves out the AC-2 case, a projectDir that covers the store only by crossing a protected directory. (c) "reproduces the prior unconditional wording" describes history, but that wording is still current code; "the unconditional wording" says the same thing. (d) Outcome 9 in the numbered list (`:888-891`) is the file's list of Detail variants and does not mention the narrowing. **Fix:** "a "" projectDir, or one that does not cover the store without crossing a protected directory, leaves the unconditional wording unchanged", and add half a line to outcome 9. |
| L4 | LOW | `internal/cli/doctor_codex_writable_root.go:909-910`; `docs/tech-debt/README.md:131` | The comment says "org seats often run in task worktrees under the project". Nothing in the repo supports "often": every `/org` example passes `--cwd .` (SKILL.md:120, :129, :182), the role prompts never mention worktrees, and the spec lists worktree assignment as an open question (:167). The tech-debt row repeats the claim and also reverses the containment: "task worktrees under the project, which a store at the project root would not cover". What matters is whether the worktree contains the store, not whether the store covers the worktree. The store also does not have to be at the project root. The code comment has the direction right. **Fix:** "org seats may run in a task worktree (e.g. `.claude/worktrees/<slug>`), whose working directory does not contain a store elsewhere in the project". Optionally, also cite the #170 plan in the row's evidence column. |
| L5 | LOW | `internal/cli/doctor_codex_writable_root_test.go:1911-1928` | There is no blank line between the section header paragraph (`// --- issue #170: ... ---`, ending at `:1918`) and `// codexWorkspaceWriteUnconditionalPhrase is ...` (`:1919`), so go doc attaches the section header to the const as part of its doc comment. The other section headers in the package leave a blank line (`doctor_org_test.go:871-873`, `org_test.go:1015-1017`). **Fix:** insert a blank line after `:1918`. |
| L6 | LOW | `docs/recipes/codex-seat-permissions.md:67-71` (+ template); `.claude/skills/org/SKILL.md:244-246` (+ 3 mirrors) | The new doc sentence ("When the agmsg store itself lives inside this project … the warn says so" / 「保存先がこのプロジェクトの中にある場合 … warn はその旨を示す」) does not hold for a store under `<project>/.agents`, `.git`, or `.codex`. The binary run with `RALPH_ORG_AGMSG_HOME=<proj>/.agents/agmsg` printed the unconditional wording. The sentence just before it explains the protection, so a careful reader can work this out, but the new sentence says the opposite. **Fix:** add "(and not under its `.git`, `.agents`, or `.codex` directory)" / 「(`.git`・`.agents`・`.codex` の下を除く)」. Nit: 「root がなお必要」 is understandable; 「writable root が引き続き必要」 reads more plainly. |

No CRITICAL, HIGH, or MEDIUM findings.

## Positive notes

- The decision reuses `codexRootCoverage` as is (`:1003`), so projectDir goes through the same rule an explicit `writable_roots` entry does: coverage is checked on the resolved pair, and protected directories are checked on all four spelling pairings. No new path rule was introduced.
- `inProject` is computed only on the warn path. The pass and info Details cannot change, and the not-in-project Detail is byte-identical to main by construction.
- A `filepath.Abs` failure falls back to `""`, which keeps the unconditional wording. That is the safe direction.
- Following the Codex advisory, the unconditional sentence is replaced, not appended to. The binary output confirms the Detail keeps no leftover "cannot send" or "add" sentence.
- The AC-2 tests compare the whole Detail against the `projectDir=""` form byte for byte, which is stronger than checking for absent substrings. `ProjectDirSymlinked_SameAsResolved` fails with `Fatalf` if the direct case does not reach the in-project wording, so it cannot pass vacuously.
- The new tests are hermetic. The seam env has no `Home`, `TmpDir`, or `SlashTmpDir`, and every path comes from `t.TempDir()`, so the assertions on the raw project path do not depend on where `$TMPDIR` is (cycle-log #24 in the reviewer memory).

## Observations (not findings)

- When doctor runs through a symlinked cwd, the project is displayed with the logical path and the store with the real path, so the text "S … is inside this project (P)" does not show a shared prefix. The decision itself is correct. This is display only.
- `filepath.Abs` already returns `""` on error. The explicit reset at `internal/cli/doctor.go:151-153` is redundant, but it states the intent.
- `resolveNearestExisting(store)` now runs three times on the warn path. The cost is negligible.

## Coverage gaps (for /test)

- The wiring is not pinned by any test. If `projectDir` is replaced by `""` at `internal/cli/doctor.go:154`, every test stays green: no test outside `doctor_codex_writable_root_test.go` contains "inside this project", and no `runDoctor*` test asserts this check's Detail. The only evidence for the wiring is the manual binary run.
- AC-1's "no unconditional add instruction" half is not asserted. `codexWorkspaceWriteUnconditionalPhrase` covers only the "cannot send RESULT" half. An implementation that kept `; add <store> to …` next to the new conditional sentence would still pass. A negative `strings.Contains(r.Detail, "; add "+store)` would close this.
- The plan lists a projectDir that does not exist, or that is relative, as an edge case. No new test passes one.
- Mutation reasoning (read from the code, not executed; the plan's Deviation notes say the implementer ran the always-false and always-true variants). With `inProject = false`, StoreInsideProject (both subtests), ProjectDirSymlinked, StorageOverrideInsideProject, and StoreEqualsProjectDir go red. Dropping `&& !blocked` turns all three StoreUnderProjectProtectedDir subtests and ProjectDotAgentsSymlinked red. With `inProject = true`, StoreOutsideProject and the protected-directory tests go red. Reverting the doctor.go wiring turns nothing red (see the first bullet).

## Answers to the requested checks

1. **The inProject decision.**
   - The rule is the one explicit roots already use (`codexRootCoverage`).
   - A symlinked project resolves correctly (AC-5 test and the binary run).
   - A store that does not exist yet is handled by `resolveNearestExisting` walking up to an existing parent (the override test and the binary run with `store2/`).
   - `store == projectDir` counts as covered (`rel == "."`, with no protected element).
   - A store symlinked out of the project gives false, because the resolved pair is not covered.
   - A path that reaches a protected directory inside P through a symlink is blocked on the resolved pair.
   - A projectDir below a protected directory (e.g. `~/.agents/skills/foo`) gives true. That matches the rule `RootIsDotAgentsItself_Pass` and `RootIsAgmsgHomeItself_Pass` already pin for explicit roots, and it is just as unverified against a live codex.
   - A projectDir under a covering writable root, or under `/tmp`, never reaches the warn (pass comes first).
   - A projectDir under a root that is blocked above P shows both the blocked-ancestor clause and the in-project wording. Under the check's rule the two do not contradict each other.
   - I found no input where inProject is true while the check's own rule would say a seat with cwd P cannot write the store. The underlying assumption is the recipe's (`:40`), not a new one: codex's workspace root is the seat's cwd.
2. **The four wordings.** The grammar is fine, and the in-project forms no longer contain the unconditional assertion or instruction. The blocked-ancestor clause still follows the store path, and the suffix is still last. "needed because" now comes after the recipe pointer, which is readable but loose (L1, optional). The Sprintf lines are about 140 characters, like the existing ones; the signature is 163 (L2). An operator whose seats run in task worktrees is unlikely to read the Detail as "no action needed": the status stays warn and the Detail names task worktrees. The remaining risk is the "elsewhere" contrast (L1).
3. **The explicit parameter.** This is the right choice. `codexSandboxEnv` is documented as "the part of the process environment", and projectDir is doctor's input, not the environment. Calling `os.Getwd` inside the check would break hermeticity and ignore `targetDir`. All 53 rewritten call sites pass `""`, which reproduces the old Detail by construction, and no existing test could have meant a real project because the concept did not exist before this change. The one improvement is L2.
4. **Comments.** They are accurate except for L3 and L4. The same rationale (cwd is writable, task worktrees, replace rather than append) appears in four places: `doctor.go:147-149`, `:799-812`, `:905-918`, and test `:1911-1918`. The file's style is verbose anyway, but one of the two long paragraphs in the check file could own the rationale and the other could point to it. "(issue #170)" fits the file's existing citations ("(#164)", "(cross-review AR-1)"); the "issue #170" versus "#164" spelling difference is a nit.
5. **Docs.** The English recipe sentence is accurate apart from L6 and the L1 contrast. The Japanese is plain and natural, with no AI-style phrasing; 「なお必要」 is slightly stiff (L6 nit). The four skill mirrors and the two recipe copies are identical. In the tech-debt row, see L4.
6. **Tests.**
   - The new tests are hermetic.
   - The names follow the file's `_Warn` convention, but `StoreInsideProject_Warn` names the status rather than the wording it protects; something like `_WarnNarrowsToSeatsWithoutStore` would say what it pins (nit).
   - Reverting the check logic turns the new assertions red (see Coverage gaps).
   - Reverting the `doctor.go` wiring does not.

## Recommendation

Merge is OK from a diff-quality view: 0 CRITICAL, 0 HIGH, 0 MEDIUM, 6 LOW. L1, L4, L5, and L6 are wording or comment edits of one or two lines each. L2 is a small signature change. L3 is a comment fix. I recommend fixing them in one batch in this cycle; if they are deferred, record them in a single tech-debt row. Pass the two coverage gaps (the unpinned doctor.go wiring and the missing negative assertion for the "add" half) to /test.
