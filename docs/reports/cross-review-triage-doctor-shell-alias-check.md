# Cross-review triage report: doctor-shell-alias-check

- Date: 2026-09-18
- Plan: docs/plans/active/2026-09-18-doctor-shell-alias-check.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 4
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=2, DISMISSED=0
- User decision (2026-09-18, AskUserQuestion): fix all four findings (AR-1, AR-2, WC-1, WC-2) and re-run the full pipeline as cycle 2/2

## Triage context

- Active plan: docs/plans/active/2026-09-18-doctor-shell-alias-check.md (pinned in `.harness/state/standard-pipeline/active-plan.json`)
- Self-review report: docs/reports/self-review-2026-09-18-doctor-shell-alias-check.md (cycle 1: 9 findings, all resolved; revalidation: 6 new findings N1-N6, all fixed in-cycle without a second reviewer pass)
- Verify report: docs/reports/verify-2026-09-18-doctor-shell-alias-check.md (PASS)
- Implementation context summary: The check statically scans shell rc files for `codex` / `claude` aliases that add a flag ralph itself passes to a seat. Severity and wording were reworked twice in this cycle: claude findings were split from codex findings after a CLI probe (claude 2.1.274 accepts a repeated flag, last value wins), and the revalidation finding N1 made the sentences conditional on the flag class (model vs permission) because `permissionArgsForDriver` passes nothing to guarded seats. The reviewer ran `codex exec review --base main` (codex-cli 0.154.0, `-m gpt-6-astra -c model_reasoning_effort=xhigh`, stdin closed). A first attempt through `command codex` (alias bypassed) failed with an API 400 (`'max' is not supported with the 'gpt-5.5' model`: project `.codex/config.toml` model + user-level effort) and produced no findings; it is not counted as a review. All four findings were checked against the code before classification.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] `$ZDOTDIR/.zprofile` is not a candidate: the list has only `$ZDOTDIR/.zshrc` and `$ZDOTDIR/.zshenv`, and `.zprofile` is looked up under HOME only. An alias there is loaded by herdr's login zsh while doctor prints `pass — no shell rc file found to scan`. | Real: evidence P1 of #155 established that the pane shell is a login zsh, which reads `.zshenv`, `.zprofile`, `.zshrc`, `.zlogin` from `$ZDOTDIR`. The candidate list is Scope item 1 of the plan, so this is in scope, and the failure mode is the false all-clear that M2/N2 were about. Cheap fix: build the zsh file set (`.zshenv`, `.zprofile`, `.zshrc`, `.zlogin`) for each of `$ZDOTDIR`, `~`, `~/.config/zsh`. | `internal/cli/doctor_shell_alias.go:59-77` |
| AR-2 | [P2] The codex permission clause says edits and autonomous spawns fail for both `--sandbox` and `--ask-for-approval`, but `codexEditsArgs` is only `--sandbox workspace-write`. With `alias codex='codex -a never'` an edits seat starts, with approvals silently disabled. The skill and recipe mirror the same wording. | Real, and the same class as revalidation finding N1, which the fix got half right: `internal/org/permissions.go:66-67` confirms `--ask-for-approval` is passed to autonomous seats only. The sentence states a falsehood about a permission-relevant case (approvals disabled without notice on edits and guarded seats). Fix: track sandbox and approval flags separately in the Detail clauses and correct the skill (4 mirrors), the recipe (2 copies), and the plan wording. | `internal/cli/doctor_shell_alias.go:575-590`, `.claude/skills/org/SKILL.md` (+3 mirrors), `docs/recipes/codex-seat-permissions.md` (+template) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| WC-1 | [P2] The alias VALUE is tokenized with `strings.Fields`, so quotes inside the value survive: `alias codex='codex "--model" gpt-5'` reports `pass`, while zsh re-parses the alias text and passes `--model`. | Real (zsh does remove the inner quotes on expansion), but the spelling is unusual: quoting a flag NAME is rare, and the common forms (`-m "gpt-5"`, `--model="gpt-5"`) are already detected because the flag token itself is unquoted. Cheap to fix by running the value through the existing statement reader and flattening its words, so worth doing if a fix cycle happens anyway. | `internal/cli/doctor_shell_alias.go:440-442` |
| WC-2 | [P2] A candidate whose `os.Stat` fails for a reason other than not-exist (e.g. an inaccessible parent directory) is dropped silently, so doctor can print `pass — no shell rc file found to scan` although an rc file exists. | Real but rare (a home subdirectory without search permission). It is the same incomplete-inspection-reads-as-pass pattern that M3 fixed for open errors. Cheap fix: drop only `fs.ErrNotExist`; keep other stat failures as candidates so the open error reaches the `could not read` clause and the status becomes `info`. | `internal/cli/doctor_shell_alias.go:81-84` |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
