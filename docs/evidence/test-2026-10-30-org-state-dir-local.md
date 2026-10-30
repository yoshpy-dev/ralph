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

## cycle 2 (HEAD fcbd5dbf)

Excerpts of the cycle 2 /test run (S9 6415aabd: local-only keys in ralph.toml refused in any letter case the go-toml decoder reads). The full log is `docs/evidence/test-2026-10-30-org-state-dir-local-cycle2.log` (gitignored, 5,194 lines of run-test.sh output, then the sections below). Launched the same way as cycle 1.

### run-test.sh (cycle 2)

```
## run-test.sh start 2026-10-30 14:30:49 JST load { 15.09 14.08 10.88 } HEAD fcbd5dbf
# Verification run
- Timestamp: 2026-10-30T05:30:49Z
- Mode: test
- Requested scope: changed

5175:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
5176:==> Language packs selected: golang
5177:==> Running golang verifier
5178:==> Go project root: .
5191:==> All verifiers passed.
?   	github.com/yoshpy-dev/ralph	[no test files]
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/cli	760.324s
ok  	github.com/yoshpy-dev/ralph/internal/config	14.853s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	915.284s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-30-053049.log
## run-test.sh rc=0 elapsed=1766s end 2026-10-30 15:00:16 JST load { 20.75 34.35 43.08 }
```

### Shell suites (cycle 2): files, OK, lines containing PASS

```
45 files, 45 OK, 4796 PASS lines
4129:  SKIP: the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)
```

### Uncached rerun, -count=5, -count=20, m6 (cycle 2)

```
## uncached rerun of the packages run-test.sh served from the cache (go test -count=1)
ok  	cmd/ralph	1.312s
ok  	internal/gitroot	11.050s
ok  	internal/insights	3.045s
ok  	internal/org/driver	3.580s
ok  	internal/org/protocol	3.177s
ok  	internal/scaffold	2.454s
ok  	internal/upgrade	2.291s

## go test ./internal/config/ -run 'LocalOnly|LetterCase|LocalOrgStateDir' -count=5 -v (rc 0, 32 s)
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_--config_file_of_another_name
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_capitalized_table_and_key_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_capitalized_table_and_key_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_dotted_key_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_dotted_key_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_value_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/a_value_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_empty_value_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_empty_value_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_inline_table_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_inline_table_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_dotted_key_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_dotted_key_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_inline_table_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_inline_table_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_key_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_key_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_table_and_key_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_table_and_key_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_table_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/an_upper-case_table_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/the_second_of_two_tables_with_RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphToml/the_second_of_two_tables_with_RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults/RALPH_LOCAL_CONFIG=
   5     --- PASS: TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults/RALPH_LOCAL_CONFIG=off
   5     --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath/a_directory
   5     --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath/no_home_directory
   5     --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath/not_a_string
   5     --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath/not_valid_TOML
   5     --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath/unreadable
   5     --- PASS: TestLocalOrgStateDir_LocalConfigEnv/empty_reads_the_file
   5     --- PASS: TestLocalOrgStateDir_LocalConfigEnv/OFF_reads_the_file
   5     --- PASS: TestLocalOrgStateDir_LocalConfigEnv/off_skips_the_file_and_git
   5     --- PASS: TestLocalOrgStateDir_LocalConfigEnv/on_reads_the_file
   5     --- PASS: TestLocalOrgStateDir_LocalConfigEnv/unset_reads_the_file
   5     --- PASS: TestLocalOrgStateDir_ReadsTheMainWorktreesFile/a_bare_repository's_linked_worktree
   5     --- PASS: TestLocalOrgStateDir_ReadsTheMainWorktreesFile/the_working_directory
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/_/ledger
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/../ledger
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/./ledger
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/~
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/~user/ledger
   5     --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute/ledger
   5     --- PASS: TestLocalOrgStateDir_Values/absent
   5     --- PASS: TestLocalOrgStateDir_Values/absolute
   5     --- PASS: TestLocalOrgStateDir_Values/absolute,_cleaned
   5     --- PASS: TestLocalOrgStateDir_Values/an_empty_file
   5     --- PASS: TestLocalOrgStateDir_Values/empty
   5     --- PASS: TestLocalOrgStateDir_Values/home_itself
   5     --- PASS: TestLocalOrgStateDir_Values/no_file
   5     --- PASS: TestLocalOrgStateDir_Values/ralph.toml_is_not_valid_TOML
   5     --- PASS: TestLocalOrgStateDir_Values/ralph.toml_sets_it
   5     --- PASS: TestLocalOrgStateDir_Values/under_home
   5     --- PASS: TestLocalOrgStateDir_Values/upper-case_table_and_key
   5 --- PASS: TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder
   5 --- PASS: TestLoad_LocalOnlyKeyInRalphToml
   5 --- PASS: TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults
   5 --- PASS: TestLocalOnlyKeys
   5 --- PASS: TestLocalOrgStateDir_FileErrorsNameItsPath
   5 --- PASS: TestLocalOrgStateDir_LocalConfigEnv
   5 --- PASS: TestLocalOrgStateDir_ReadsTheMainWorktreesFile
   5 --- PASS: TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute
   5 --- PASS: TestLocalOrgStateDir_Values
   5 --- PASS: TestTemplateDriftProbesHoldNoLocalOnlyKey
ok  	internal/config	31.898s

## go test ./internal/config/ -run 'LetterCase|LocalOnlyKeyInRalphToml$' -count=20 -v (rc 0, 36 s)
  20 --- PASS: TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder
  20 --- PASS: TestLoad_LocalOnlyKeyInRalphToml
ok  	internal/config	35.066s

## m6: order-dependent tomlDocHasKey (return on the first lower-case match), same -run, -count=20, -overlay (rc 1, 46 s)
  166,168c166,167
  < 		if table, ok := v.(map[string]any); ok && tomlDocHasKey(table, rest) {
  < 			return true
  < 		}
  ---
  > 		table, ok := v.(map[string]any)
  > 		return ok && tomlDocHasKey(table, rest)
  19     --- FAIL: TestLoad_LocalOnlyKeyInRalphToml/the_second_of_two_tables_with_RALPH_LOCAL_CONFIG=
  16     --- FAIL: TestLoad_LocalOnlyKeyInRalphToml/the_second_of_two_tables_with_RALPH_LOCAL_CONFIG=off
  20 --- FAIL: TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder
  20 --- FAIL: TestLoad_LocalOnlyKeyInRalphToml
FAIL
```
