# Verify evidence: org-state-dir-local

Raw output of /verify for `docs/plans/active/2026-10-30-org-state-dir-local.md` at HEAD e0ffa2aa (merge-base c03647f9 = origin/main). The full log, with every line of run-static-verify, is `docs/evidence/verify-2026-10-30-org-state-dir-local.log` (gitignored). In the probes, `<W>` is a scratch directory outside the repository, and every command ran with `env -i PATH=/usr/bin:/bin HOME=<W>/home`.

## run-static-verify.sh

```
2:## run-static-verify.sh start 2026-10-30 12:01:23 JST load { 26.72 46.28 47.53 }
361:==> scripts/check-sync.sh
389:  IDENTICAL:      172
390:  DRIFTED:        0
391:  ROOT_ONLY:      0
395:PASS: all files in sync.
397:==> scripts/check-pipeline-sync.sh
406:==> scripts/check-skill-sync.sh
407:[ok] check-skill-sync: 14 skill(s) in lock-step
409:==> scripts/check-template-purity.sh
412:PASS: no meta-repo-specific references found in templates.
416:==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
417:==> Language packs selected: golang
418:==> Running golang verifier
419:==> Go project root: .
420:gofmt: ok
421:0 issues.
422:==> Running branch secret scan
423:secret-scan-branch: scanned c03647f9..e0ffa2aa against origin/main: clean
425:==> All verifiers passed.
428:## run-static-verify.sh rc=0 elapsed=175s end 2026-10-30 12:04:18 JST load { 53.04 49.04 48.25 }
```

## Plan digest, whitespace, U+FFFD, env greps

```
## plan digest
1ea27c42968f
4:- Approved: 2026-10-30 sha256:1ea27c42968f
## git diff --check origin/main...HEAD
rc=0
## U+FFFD in changed files
rc(grep)=1

## go build -o <scratch>/ralph ./cmd/ralph start 12:11:19 load { 16.74 53.67 57.76 }
## go build rc=0 1s; go version: go version go1.27.3 darwin/arm64; go.mod: go 1.27.3
## plan S4 grep (expect no match)
rc=1
## EnvOrgStateDir anywhere (git grep -c)
docs/plans/active/2026-10-30-org-state-dir-local.md:4
## Getenv of the retired names in production Go
internal/cli/doctor.go:901:		value := getenv(v.name)
internal/cli/org_legacy_ledger.go:107:	value := strings.TrimSpace(os.Getenv(retiredOrgStateDirEnv))
## RALPH_ORG_STATE_DIR in scripts/ tests/ .codex/ cmd/ AGENTS.md CLAUDE.md templates/base (non-skill, non-recipe)
rc=1

```

## Ledger resolution call sites (non-test Go)

```
internal/cli/doctor_merge_guarantee.go:41:	stateDir, _, err = org.ResolveOrgStateDir(explicitStateDir, explicitSet)
internal/cli/insights.go:50:				ledger, _, err := resolveOrgLedger(cmd, stateDir, orgLedgerReadOnly)
internal/cli/org.go:134:	resolvedStateDir, _, err := resolveOrgLedger(cmd, stateDir, access)
internal/cli/org.go:814:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org.go:1013:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org.go:1114:	resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, p.stateDir, orgLedgerMutating)
internal/cli/org.go:1205:			resolvedStateDir, _, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org_cleanup.go:141:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org_director.go:389:	resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, stateDir, access)
internal/cli/org_inbox.go:98:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org_inbox.go:330:	resolvedStateDir, _, err := resolveOrgLedger(cmd, stateDir, access)
internal/cli/org_legacy_ledger.go:74:func resolveOrgLedger(cmd *cobra.Command, explicit string, access orgLedgerAccess) (dir, source string, err error) {
internal/cli/org_legacy_ledger.go:75:	dir, source, err = org.ResolveOrgStateDir(explicit, cmd.Flags().Changed("state-dir"))
internal/cli/org_merge.go:228:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/org_merge_order.go:141:			resolvedStateDir, _, err := resolveOrgLedger(cmd, *stateDir, orgLedgerReadOnly)
internal/cli/org_wait_director.go:54:	resolved, _, err := resolveOrgLedger(cmd, stateDir, orgLedgerReadOnly)
internal/cli/org_watch.go:112:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
internal/cli/status.go:133:			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, stateDir, orgLedgerReadOnly)
internal/org/statedir.go:72:func ResolveOrgStateDir(explicit string, explicitSet bool) (dir, source string, err error) {
```

## Probes 1 (AC1 to AC7)

```
## AC1/AC2: the personal file names the ledger from four places
### status --json (in main)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/ledger",
  stdout:   "state_dir_source": "config",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

### status --json (in main/pkg/sub)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/ledger",
  stdout:   "state_dir_source": "config",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

### status --json (in linked)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/ledger",
  stdout:   "state_dir_source": "config",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

### status --json (in linked/pkg/sub)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/ledger",
  stdout:   "state_dir_source": "config",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

  parsed: state_dir=<W>/ledger source=config
  parsed: state_dir=<W>/ledger source=config
### --state-dir beats the file (in linked)
$ ralph status --json --state-dir <W>/flag-ledger
  stdout: {
  stdout:   "state_dir": "<W>/flag-ledger",
  stdout:   "state_dir_source": "flag",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

  parsed: state_dir=<W>/flag-ledger source=flag
### RALPH_LOCAL_CONFIG=off skips the file (in linked, env RALPH_LOCAL_CONFIG=off)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/main/.harness/state/org",
  stdout:   "state_dir_source": "git-main-worktree",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
exit=0

  parsed: state_dir=<W>/main/.harness/state/org source=git-main-worktree
## AC3: ~/ is expanded; config get prints the value as written
  parsed: state_dir=<W>/home/ledger-home source=config
### config get org.state_dir (in linked)
$ ralph config get org.state_dir
  stdout: ~/ledger-home
exit=0

## AC3/AC5: a relative state_dir
### status (in linked)
$ ralph status
  stderr: ralph status: cannot resolve the org ledger: <W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory
exit=1

### org status (in linked)
$ ralph org status --org-id a
  stderr: ralph org status: cannot resolve the org ledger: <W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory
exit=1

### insights (in linked)
$ ralph insights --events-dir <W>/no-events
  stderr: ralph insights: cannot resolve the org ledger: <W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory
exit=1

### org stop --all (mutating, no flag) (in linked)
$ ralph org stop --all
  stderr: ralph org stop: cannot resolve the org ledger: <W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory
exit=1

### org stop --all --state-dir (Load runs, value not checked) (in linked)
$ ralph org stop --all --state-dir <W>/flag-ledger
  stdout: no active seats
exit=0

### doctor (fail line, goes on) (in main)
$ ralph doctor
  stdout: ralph doctor
  stdout: 
  stdout:   ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  stdout:   ✗ Codex: fail — codex unusable: codex not found in PATH: exec: "codex": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  stdout:   ⚠ Codex effective config: warn — .codex/config.toml not found
  stdout:   ⚠ Hooks integrity: warn — settings.json not found
  stdout:   ⚠ Manifest version: warn — no manifest found
  stdout:   ✓ Pack: dart: pass
  stdout:   ✓ Pack: golang: pass
  stdout:   ✓ Pack: python: pass
  stdout:   ✓ Pack: rust: pass
  stdout:   ✓ Pack: terraform: pass
  stderr: doctor: 3 check(s) failed
exit=1

  grep: 22:  ✗ Org ledger: fail — cannot resolve the org ledger: <W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory
  grep: 23:  ℹ Merge guarantee: info — unknown: git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is unknown; not cached: the org ledger could not be resolved
## AC5: a personal file that is not TOML
### org status (in linked)
$ ralph org status --org-id a
  stderr: ralph org status: cannot resolve the org ledger: <W>/main/ralph.local.toml: toml: expected character ]
exit=1

### org status --state-dir (Load still fails) (in linked)
$ ralph org status --org-id a --state-dir <W>/flag-ledger
  stderr: org: load config: <W>/main/ralph.local.toml: toml: expected character ]
exit=1

### insights --state-dir (no Load) (in linked)
$ ralph insights --events-dir <W>/no-events --state-dir <W>/flag-ledger
  stdout: No insight data yet.
  stdout:   Expected events:   <W>/no-events
  stdout:   Expected receipts: <W>/flag-ledger/model-receipts.jsonl
exit=0

## AC4: ralph.toml sets [org].state_dir
### config get org.max_seats (in main)
$ ralph config get org.max_seats
  stderr: config get: ralph.toml: cannot be read or is invalid: ralph.toml: [org].state_dir can be set only in ralph.local.toml, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in ralph.local.toml instead
exit=3

### config get org.max_seats, RALPH_LOCAL_CONFIG=off (in main, env RALPH_LOCAL_CONFIG=off)
$ ralph config get org.max_seats
  stderr: config get: ralph.toml: cannot be read or is invalid: ralph.toml: [org].state_dir can be set only in ralph.local.toml, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in ralph.local.toml instead
exit=3

### config get org.max_seats, empty value (in main)
$ ralph config get org.max_seats
  stderr: config get: ralph.toml: cannot be read or is invalid: ralph.toml: [org].state_dir can be set only in ralph.local.toml, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in ralph.local.toml instead
exit=3

## AC6: RALPH_ORG_STATE_DIR is not read
### status --json, env names another dir (in linked, env RALPH_ORG_STATE_DIR=<W>/env-ledger)
$ ralph status --json
  stdout: {
  stdout:   "state_dir": "<W>/main/.harness/state/org",
  stdout:   "state_dir_source": "git-main-worktree",
  stdout:   "orgs": [],
  stdout:   "org_watch": {
  stdout:     "state": "not_running"
  stdout:   },
  stdout:   "director": {
  stdout:     "enabled": true,
  stdout:     "auto_merge": true,
  stdout:     "state": "not_attached",
  stdout:     "registration": null,
  stderr: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; showing the org ledger at <W>/main/.harness/state/org instead, and the commands that change the ledger are refused while the variable names another one. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
exit=0

  parsed: state_dir=<W>/main/.harness/state/org source=git-main-worktree
### org stop --all, env names another dir (refused) (in linked, env RALPH_ORG_STATE_DIR=<W>/env-ledger)
$ ralph org stop --all
  stderr: ralph org stop: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names <W>/env-ledger, which ralph no longer reads, and the ledger ralph finds by default is <W>/main/.harness/state/org, so nothing was changed; to keep using <W>/env-ledger, write [org].state_dir = "<W>/env-ledger" in <W>/main/ralph.local.toml or pass --state-dir <W>/env-ledger, and unset RALPH_ORG_STATE_DIR; to use <W>/main/.harness/state/org, unset RALPH_ORG_STATE_DIR
exit=1

  after refusal: env-ledger exists? no; default ledger exists? no
### org stop --all --state-dir, env set (note only) (in linked, env RALPH_ORG_STATE_DIR=<W>/env-ledger)
$ ralph org stop --all --state-dir <W>/flag-ledger
  stdout: no active seats
  stderr: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; using the org ledger at <W>/flag-ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
exit=0

### org stop --all, env names the default ledger (note only) (in linked, env RALPH_ORG_STATE_DIR=<W>/main/.harness/state/org)
$ ralph org stop --all
  stdout: no active seats
  stderr: note: RALPH_ORG_STATE_DIR is set (<W>/main/.harness/state/org), but ralph no longer reads it; using the org ledger at <W>/main/.harness/state/org. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
exit=0

### org stop --all, personal file names the ledger, env another (note only) (in linked, env RALPH_ORG_STATE_DIR=<W>/env-ledger)
$ ralph org stop --all
  stdout: no active seats
  stderr: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; using the org ledger at <W>/ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
exit=0

### doctor warn line, env set (in main, env RALPH_ORG_STATE_DIR=<W>/env-ledger)
$ ralph doctor
  stdout: ralph doctor
  stdout: 
  stdout:   ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  stdout:   ✗ Codex: fail — codex unusable: codex not found in PATH: exec: "codex": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  stdout:   ⚠ Codex effective config: warn — .codex/config.toml not found
  stdout:   ⚠ Hooks integrity: warn — settings.json not found
  stdout:   ⚠ Manifest version: warn — no manifest found
  stdout:   ✓ Pack: dart: pass
  stdout:   ✓ Pack: golang: pass
  stdout:   ✓ Pack: python: pass
  stdout:   ✓ Pack: rust: pass
  stdout:   ✓ Pack: terraform: pass
  stderr: doctor: 2 check(s) failed
exit=1

  grep: 18:  ⚠ RALPH_ORG_STATE_DIR: warn — set to "<W>/env-ledger", but ralph no longer reads it, and while it names another directory than the ledger ralph finds by default, the org verbs that change the ledger are refused; write the value in [org].state_dir of ralph.local.toml (next to ralph.toml in the main worktree root, ignored by git), as an absolute path or one that starts with ~/, or pass --state-dir, and unset RALPH_ORG_STATE_DIR
  grep: 23:  ℹ Merge guarantee: info — unknown: git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is unknown; cached in <W>/ledger/merge-guarantee.json
## AC7: insights --state-dir
### insights --state-dir (in linked)
$ ralph insights --events-dir <W>/no-events --state-dir <W>/ins-ledger
  stdout: 
  stdout: === Events (0 total, 0 lines skipped) ===
  stdout: 
  stdout:   (no events)
  stdout: 
  stdout: === Routing (honored-rate per phase from events) ===
  stdout: 
  stdout:   (no events with routing fields)
  stdout: 
  stdout: === Escalation (slugs with cycle >= 2) ===
  stdout: 
  stdout:   none observed
exit=0

  grep: 14:=== Receipts (org runtime model-receipts: <W>/ins-ledger/model-receipts.jsonl) ===
  grep: 16:  ORG probe  SEAT lead  commanded=opus  honored: true=0 false=0 unknown=1  rate=n/a (unknown 1 excluded)
### insights --state-dir --receipts (receipts wins) (in linked)
$ ralph insights --events-dir <W>/no-events --state-dir <W>/ins-ledger --receipts <W>/receipts.jsonl
  stdout: 
  stdout: === Events (0 total, 0 lines skipped) ===
  stdout: 
  stdout:   (no events)
  stdout: 
  stdout: === Routing (honored-rate per phase from events) ===
  stdout: 
  stdout:   (no events with routing fields)
  stdout: 
  stdout: === Escalation (slugs with cycle >= 2) ===
  stdout: 
  stdout:   none observed
exit=0

  grep: 14:=== Receipts (org runtime model-receipts: <W>/receipts.jsonl) ===
  grep: 16:  ORG other  SEAT rev  commanded=opus  honored: true=0 false=0 unknown=1  rate=n/a (unknown 1 excluded)
```

## Probes 2 (AC6, more verbs)

The first send block used a mistyped flag (`--seat`) and is kept as it ran; the last block is the corrected send.

```
### ralph org send --org-id a --seat implementer --text TYPE:_TASK (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=1
  err: unknown flag: --seat
### ralph org disband --all (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=1
  err: ralph org disband: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names <W>/env-ledger, which ralph no longer reads, and the ledger ralph finds by default is <W>/main/.harness/state/org, so nothing was changed; to keep using <W>/env-ledger, write [org].state_dir = "<W>/env-ledger" in <W>/main/ralph.local.toml or pass --state-dir <W>/env-ledger, and unset RALPH_ORG_STATE_DIR; to use <W>/main/.harness/state/org, unset RALPH_ORG_STATE_DIR
### ralph org inbox (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=0
  out: no open or acked inbox items
  err: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; showing the org ledger at <W>/main/.harness/state/org instead, and the commands that change the ledger are refused while the variable names another one. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
### ralph org merge-order (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=0
  out: merge guarantee: unknown (no org ledger yet)
  out: note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it or ralph org gate records a gate on it
  err: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; showing the org ledger at <W>/main/.harness/state/org instead, and the commands that change the ledger are refused while the variable names another one. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
### ralph org director status (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=0
  out: director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work; once attached, a director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)
  out: enabled: true
  out: auto_merge: true
  err: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; showing the org ledger at <W>/main/.harness/state/org instead, and the commands that change the ledger are refused while the variable names another one. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
### ralph org send --org-id a --to implementer --text <TASK> (env RALPH_ORG_STATE_DIR=<W>/env-ledger)
exit=1
  err: ralph org send: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names <W>/env-ledger, which ralph no longer reads, and the ledger ralph finds by default is <W>/main/.harness/state/org, so nothing was changed; to keep using <W>/env-ledger, write [org].state_dir = "<W>/env-ledger" in <W>/main/ralph.local.toml or pass --state-dir <W>/env-ledger, and unset RALPH_ORG_STATE_DIR; to use <W>/main/.harness/state/org, unset RALPH_ORG_STATE_DIR
  env-ledger exists? no; default ledger exists? no
```

## Probes 3 (AC5, AC7, doctor warn)

```
### AC5: broken ralph.toml, valid personal file: status --json (ledger still resolved)
exit=0
  out: {
  out:   "state_dir": "<W>/ledger",
  out:   "state_dir_source": "config",
  out:   "orgs": [],
  out:   "org_watch": {
  out:     "state": "not_running"
### AC5: ralph.local.toml is a directory: org status
exit=1
  err: ralph org status: cannot resolve the org ledger: read <W>/main/ralph.local.toml: is a directory
### AC7: doctor --state-dir
exit=1
  out: ralph doctor
  out: 
  out:   ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  out:   ✗ Codex: fail — codex unusable: codex not found in PATH: exec: "codex": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  out:   ⚠ Codex effective config: warn — .codex/config.toml not found
  out:   ⚠ Hooks integrity: warn — settings.json not found
  err: doctor: 2 check(s) failed
  grep: 22:  ℹ Merge guarantee: info — unknown: git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is unknown; cached in <W>/doc-ledger/merge-guarantee.json
### doctor warn while the personal file names the ledger and the env names another
exit=1
  out: ralph doctor
  out: 
  out:   ✗ Claude Code CLI: fail — claude unusable: claude not found in PATH: exec: "claude": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  out:   ✗ Codex: fail — codex unusable: codex not found in PATH: exec: "codex": executable file not found in $PATH — neither claude nor codex is usable, and ralph needs at least one of them
  out:   ⚠ Codex effective config: warn — .codex/config.toml not found
  out:   ⚠ Hooks integrity: warn — settings.json not found
  err: doctor: 2 check(s) failed
  grep: 18:  ⚠ RALPH_ORG_STATE_DIR: warn — set to "<W>/env-ledger", but ralph no longer reads it, and while it names another directory than the ledger ralph finds by default, the org verbs that change the ledger are refused; write the value in [org].state_dir of ralph.local.toml (next to ralph.toml in the main worktree root, ignored by git), as an absolute path or one that starts with ~/, or pass --state-dir, and unset RALPH_ORG_STATE_DIR
### same setting: org stop --all goes on
exit=0
  out: no active seats
  err: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; using the org ledger at <W>/ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
```

## Help of 35 commands

```
RALPH_ORG_STATE_DIR lines:
799:RALPH_ORG_STATE_DIR, which ralph no longer reads, names another directory
distinct --state-dir usages:
   1       --state-dir string       org state directory the merge guarantee check caches its result in (default: the ledger ralph resolves -- [org].state_dir in ralph.local.toml, else the main worktree's .harness/state/org)
   1       --state-dir string    org state directory whose model-receipts.jsonl is read (default: the ledger ralph resolves -- [org].state_dir in ralph.local.toml, else the main worktree's .harness/state/org); --receipts overrides it
   1       --state-dir string   org manifest/receipts state directory (default: resolved by org.ResolveOrgStateDir -- [org].state_dir in the ralph.local.toml at the main worktree's root, else the main worktree's .harness/state/org (shared by its linked worktrees), else the enclosing git toplevel's .harness/state/org, else cwd's .harness/state/org)
  29       --state-dir string   org manifest/receipts/inbox state directory (default: resolved by org.ResolveOrgStateDir -- [org].state_dir in the ralph.local.toml at the main worktree's root, else the main worktree's .harness/state/org (shared by its linked worktrees), else the enclosing git toplevel's .harness/state/org, else cwd's .harness/state/org)
```

## tech-debt Trigger sweep

Open rows of `docs/tech-debt/README.md` whose Trigger column (`re.split(r"(?<!\) \| ", row)[3]`) names a changed file, a function of a changed hunk, or a skill, template, or env name this branch touched (plain lowercase words dropped). The report reads each hit.

```
135	O	doctor_codex_writable_root, doctor_codex_writable_root.go
136	O	doctor_shell_alias, doctor_shell_alias.go
144	O	doctor.go, doctor_codex_writable_root
148	O	MainWorktreeRoot, RALPH_ORG_STATE_DIR, ResolveOrgStateDir, ralph.toml
149	O	defaults_sync_test, defaults_sync_test.go
150	O	org_test, org_test.go
157	O	/org` skill
170	O	guardLegacyOrgStateDir
174	O	/director` skill
177	O	/org` skill, newOrgStartCmd, ralph.local.toml
183	O	/org` skill
186	O	ResolveOrgStateDir, newOrgRuntime
188	O	RALPH_ORG_STATE_DIR, ralph.toml
192	O	/director` skill, RALPH_ORG_STATE_DIR, leader template, newOrgStartCmd
196	O	/director` skill
197	O	leader.md
198	O	LedgerMainWorktreeRoot, MainWorktreeRoot
203	O	Escalate, newOrgEscalateCmd, newOrgEscalateFallbackRuntime, newOrgRuntime, org.go, org_inbox, org_inbox.go
208	O	newOrgRuntime, readOrgWatchHeartbeat
215	O	escalate_test, escalate_test.go
236	O	leader template, org_wait_director, org_wait_director.go
241	O	/director` skill
242	O	/director` skill
243	O	/director` skill
246	O	/director` skill, org_merge, org_merge_order, org_merge_order.go
247	O	/director` skill
248	O	/director` skill
249	O	/director` skill
251	O	mergeGuaranteeResult
258	O	org_director, org_director.go
259	O	/director` skill, defaults_sync_test, defaults_sync_test.go, org_wait_director
266	O	director_skill_sync_test, director_skill_sync_test.go
268	O	codex-seat-permissions, codex-seat-permissions.md
272	O	ralph.toml
273	O	codex-seat-permissions, codex-seat-permissions.md
294	O	local_test, local_test.go
313	O	feature.go
352	O	/org` skill
353	O	Escalate
354	O	escalate.go, escalate_test, escalate_test.go
359	O	status.go
362	O	director skill, director_skill_sync_test, director_skill_sync_test.go, leader template, leader.md, prompts_test, prompts_test.go
364	O	leader template
368	O	README.md, codex-seat-permissions, codex-seat-permissions.md, ralph.local.toml
369	O	ralph.local.toml
372	O	leader template
373	O	/org` skill, leader template, leader.md
376	O	/director` skill
377	O	/director` skill
380	O	leader.md
386	O	cleanup_test, cleanup_test.go, org_merge
390	O	cleanup.go, ledgerMoveStep
391	O	cleanup.go
392	O	statedir.go
```

## S8 check (cb33a636, HEAD b55e877b)

```
## S8 check: HEAD b55e877b, start 12:35:53 load { 12.95 38.15 61.17 }
$ go build ./...
rc=0 1s
$ go build -o <scratch>/ralph-s8 ./cmd/ralph
rc=0 1s
$ go vet ./internal/cli/
rc=0 1s
load { 12.95 37.73 60.89 }
$ golangci-lint run ./internal/cli/
0 issues.
rc=0 2s
$ gofmt -l (S8 files)
rc=0
end 12:35:59 load { 12.95 37.73 60.89 }
```

```
### doctor, personal file names <W>/ledger, RALPH_ORG_STATE_DIR=<W>/env-ledger (HEAD b55e877b binary)
exit=1
  18:  ⚠ RALPH_ORG_STATE_DIR: warn — set to "<W>/env-ledger", but ralph no longer reads it, and while neither --state-dir nor [org].state_dir names the ledger and the variable names another directory than the ledger ralph finds by default, the org verbs that change the ledger are refused; write the value in [org].state_dir of ralph.local.toml (next to ralph.toml in the main worktree root, ignored by git), as an absolute path or one that starts with ~/, or pass --state-dir, and unset RALPH_ORG_STATE_DIR
  23:  ℹ Merge guarantee: info — unknown: git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is unknown; cached in <W>/ledger/merge-guarantee.json
### same setting: org stop --all from the linked worktree
exit=0
  out: no active seats
  err: note: RALPH_ORG_STATE_DIR is set (<W>/env-ledger), but ralph no longer reads it; using the org ledger at <W>/ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
### no personal file, same env: org stop --all (refused)
exit=1
  err: ralph org stop: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names <W>/env-ledger, which ralph no longer reads, and the ledger ralph finds by default is <W>/main/.harness/state/org, so no
### ralph org escalate --help (last sentence)
43-cannot be resolved because the personal ralph.local.toml cannot be read
44-or its [org].state_dir is neither absolute nor starts with ~/, and when
45-neither --state-dir nor [org].state_dir names the ledger and
46-RALPH_ORG_STATE_DIR, which ralph no longer reads, names another directory
47:than the ledger found by default.
```
