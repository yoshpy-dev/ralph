# Test evidence: org-state-dir-local

Excerpts of the /test run for `docs/plans/active/2026-10-30-org-state-dir-local.md` at HEAD e5d5a5a4 (merge-base c03647f9 = origin/main). The full log is `docs/evidence/test-2026-10-30-org-state-dir-local.log` (gitignored, 5,194 lines of run-test.sh output, then the sections below). run-test.sh was detached with perl fork + setsid (signals reset to default, std fds to /dev/null) and waited on with a foreground `caffeinate -w`.

## run-test.sh

```
## run-test.sh start 2026-10-30 12:40:21 JST load { 9.32 21.81 47.67 } HEAD e5d5a5a4
# Verification run
- Timestamp: 2026-10-30T03:40:21Z
- Mode: test
- Requested scope: changed

5175:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
5176:==> Language packs selected: golang
5177:==> Running golang verifier
5178:==> Go project root: .
5191:==> All verifiers passed.
?   	github.com/yoshpy-dev/ralph	[no test files]
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	0.893s
ok  	github.com/yoshpy-dev/ralph/internal/cli	505.448s
ok  	github.com/yoshpy-dev/ralph/internal/config	19.649s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	3.863s
ok  	github.com/yoshpy-dev/ralph/internal/insights	1.245s
ok  	github.com/yoshpy-dev/ralph/internal/org	721.614s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	5.520s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.564s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.259s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	3.473s

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-30-034021.log
## run-test.sh rc=0 elapsed=1375s end 2026-10-30 13:03:15 JST load { 9.05 9.23 17.00 }
```

## Shell suites (file, lines containing PASS, result)

```
tests/test-agent-models.sh                   48    OK
tests/test-agent-phase-boundaries.sh         45    OK
tests/test-archive-plan.sh                   22    OK
tests/test-branch-name.sh                    26    OK
tests/test-check-mojibake.sh                 16    OK
tests/test-check-skill-sync.sh               14    OK
tests/test-check-sync.sh                     17    OK
tests/test-check-template.sh                 60    OK
tests/test-codex-exec-invocation.sh          144   OK
tests/test-detect-changed-languages.sh       84    OK
tests/test-detect-languages-terraform.sh     9     OK
tests/test-ensure-pr-ready.sh                7     OK
tests/test-ensure-pr-title-prefix.sh         13    OK
tests/test-gc-artifacts.sh                   11    OK
tests/test-golang-pack-test-timeout.sh       35    OK
tests/test-hook-wiring.sh                    69    OK
tests/test-insights-append.sh                51    OK
tests/test-language-pack-monorepo-roots.sh   30    OK
tests/test-lib-json.sh                       127   OK
tests/test-new-feature-plan.sh               21    OK
tests/test-no-loop-references.sh             1     OK
tests/test-plan-visual.sh                    243   OK
tests/test-post-edit-verify.sh               25    OK
tests/test-pr-owner-lookup.sh                20    OK
tests/test-pre-bash-guard.sh                 2228  OK
tests/test-ralph-config.sh                   19    OK
tests/test-ralph-dispatch.sh                 34    OK
tests/test-ralph-worktree.sh                 143   OK
tests/test-run-verify-branch-secret-scan.sh  33    OK
tests/test-run-verify-concurrency.sh         38    OK
tests/test-run-verify-scope.sh               20    OK
tests/test-secret-scan-branch.sh             247   OK
tests/test-secret-scan.sh                    124   OK
tests/test-self-review-scope.sh              207   OK
tests/test-skill-insight-cycle.sh            20    OK
tests/test-slice-verification.sh             163   OK
tests/test-sync-skills.sh                    23    OK
tests/test-template-purity.sh                11    OK
tests/test-terraform-gitignore.sh            48    OK
tests/test-terraform-pack-verify.sh          37    OK
tests/test-terraform-rule-frontmatter.sh     12    OK
tests/test-verify-local-hook-tests.sh        5     OK
tests/test-verify-mode-split.sh              60    OK
tests/test-wip-checkpoint-hooks.sh           156   OK
tests/test-xreview-helpers.sh                30    OK
4129:  SKIP: the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)
```

## go test -list, coverage, mutants

```
## go test -list counts (HEAD e5d5a5a4)
internal/org 1285, internal/cli 910, internal/config 90, internal/gitroot 6, cmd/ralph 2, internal/insights 41, internal/org/driver 73, internal/org/protocol 15, internal/scaffold 32, internal/upgrade 114

## coverage runs
### config
ok  	internal/config	44.183s	coverage: 96.7% of statements
### org
ok  	internal/org	45.683s	coverage: 22.8% of statements
### cli
ok  	internal/cli	380.677s	coverage: 55.4% of statements
### per-function (changed functions)
internal/config/config.go:710:	Load				90.9%
internal/config/local.go:118:	LocalOnlyKeys			100.0%
internal/config/local.go:127:	checkLocalOnlyKeys		88.9%
internal/config/local.go:176:	LocalOrgStateDir		100.0%
internal/org/escalate.go:233:		EscalateNotRecorded			100.0%
internal/org/statedir.go:72:		ResolveOrgStateDir			100.0%
internal/org/statedir.go:98:		MainWorktreeRoot			87.5%
internal/org/statedir.go:123:		LedgerMainWorktreeRoot			100.0%
internal/org/statedir.go:146:		FeatureRepoRoot				85.7%
internal/org/statedir.go:176:		LegacyWorktreeStateDir			90.0%
internal/org/statedir.go:210:		resolvedOrClean				100.0%
internal/org/statedir.go:219:		mustAbs					75.0%
internal/cli/doctor.go:105:				runDoctorStateDir			95.9%
internal/cli/doctor.go:899:				checkRetiredEnv				100.0%
internal/cli/doctor_merge_guarantee.go:40:		mergeGuaranteeEnvFromOS			100.0%
internal/cli/doctor_merge_guarantee.go:70:		checkMergeGuarantee			100.0%
internal/cli/doctor_merge_guarantee.go:85:		mergeGuaranteeStatus			100.0%
internal/cli/doctor_merge_guarantee.go:104:		mergeGuaranteeResult			100.0%
internal/cli/insights.go:16:			newInsightsCmd				100.0%
internal/cli/insights.go:70:			runInsights				89.5%
internal/cli/org_legacy_ledger.go:74:		resolveOrgLedger			100.0%
internal/cli/org_legacy_ledger.go:106:		checkRetiredOrgStateDirEnv		92.9%
internal/cli/org_legacy_ledger.go:150:		guardLegacyOrgStateDir			100.0%

## mutants (go test -overlay)
### m1
  80c80
  < 	if local != "" {
  ---
  > 	if false && local != "" {
--- FAIL: TestResolveOrgStateDir_PersonalFileTier (10.71s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/main_root (0.74s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/main_subdirectory (0.61s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/sibling_worktree_root (0.70s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/nested_worktree_subdirectory (0.63s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/a_value_under_the_home_directory (1.29s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileTier/a_value_under_a_relative_home_directory (1.30s)
--- FAIL: TestResolveOrgStateDir_PersonalFileOrder (5.93s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileOrder/the_retired_env_does_not_beat_the_file (0.98s)
FAIL	internal/org	109.277s
### m2
  121c121
  < 	if access == orgLedgerMutating {
  ---
  > 	if false && access == orgLedgerMutating {
--- FAIL: TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger (31.52s)
--- FAIL: TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger (1.58s)
--- FAIL: TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman (0.66s)
--- FAIL: TestOrgRetiredStateDirEnv_NamingTheLegacyLedgerRefusesStop (1.06s)
FAIL	internal/cli	102.984s
### m3
  116c116
  < 	if source == "flag" || source == "config" || gitroot.SamePath(envDir, resolvedDir) {
  ---
  > 	if source == "flag" || gitroot.SamePath(envDir, resolvedDir) {
--- FAIL: TestOrgRetiredStateDirEnv_OnlyANoteWhenTheLedgerIsChosenOrTheSame (5.06s)
    --- FAIL: TestOrgRetiredStateDirEnv_OnlyANoteWhenTheLedgerIsChosenOrTheSame/[org].state_dir_naming_another_ledger (0.40s)
FAIL	internal/cli	28.456s
### m4
  127c127
  < 	case "flag", "config":
  ---
  > 	case "flag":
--- FAIL: TestLedgerMainWorktreeRoot (3.78s)
FAIL	internal/org	24.541s
### m5
  77c77
  < 	if err != nil {
  ---
  > 	if false && err != nil {
--- FAIL: TestResolveOrgStateDir_PersonalFileErrors (31.16s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileErrors/a_relative_state_dir (0.21s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileErrors/~_alone (0.18s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileErrors/~user (0.17s)
    --- FAIL: TestResolveOrgStateDir_PersonalFileErrors/not_TOML (0.17s)
FAIL	internal/org	242.537s
```
