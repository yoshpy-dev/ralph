package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgGateCmd wires `ralph org gate --org-id <id>` (plan
// docs/plans/active/2026-10-13-org-gate.md, AC1 to AC3 and AC5):
// (*org.Org).Gate checks the feature org and its worktree, runs the two
// scripts, and records gate_passed or gate_failed; this command passes it the
// current directory (Dir left empty), the process environment, its stderr
// for the lines of a wait for gate.lock and for the load average and of a
// version probe that changed the worktree, and the loaded config's
// [org.gate].max_load, cancels its
// context on SIGINT, SIGTERM, or SIGHUP, and prints the result
// (printGateResult). Its one subcommand, `ralph org gate accept`
// (newOrgGateAcceptCmd), records a human's acceptance of a branch's latest
// gate failure.
func newOrgGateCmd(orgID, stateDir, configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Run the verification scripts on a feature org's head and record gate_passed or gate_failed",
		Long: "ralph org gate records which head of a feature org passed the\n" +
			"verification scripts on which base. In an org started with\n" +
			"`ralph org start --plan`, the reviewer seat runs it at the leader's\n" +
			"TASK, in the org's feature worktree (or a directory under it), on the\n" +
			"head the leader pushes: after the leader rebases the feature branch on\n" +
			"the default branch and commits everything, and before it pushes and\n" +
			"opens the PR. A person may run it too.\n" +
			"\n" +
			"First, gate checks that the org holds a complete feature binding and\n" +
			"is running. It checks that the current directory is in the binding's\n" +
			"worktree, on the binding's branch (a detached HEAD is refused). It\n" +
			"checks that the worktree is clean: git status --porcelain\n" +
			"--untracked-files=all prints nothing (ignored files do not count). It\n" +
			"finds the default branch (the one refs/remotes/origin/HEAD names, else\n" +
			"main, else master) and its base ref (refs/remotes/origin/<branch>, else\n" +
			"refs/heads/<branch>), and checks that HEAD contains the base ref's tip.\n" +
			"gate never fetches: run git fetch and git rebase before it. Last, it\n" +
			"checks that ./scripts/run-static-verify.sh and ./scripts/run-test.sh\n" +
			"are executable files in the worktree's root. When a check fails, gate\n" +
			"runs nothing and records nothing, says why and what to do next on\n" +
			"stderr, and exits 1.\n" +
			"\n" +
			"A ledger runs one gate's scripts at a time. Once its checks pass, gate\n" +
			"takes gate.lock in the state dir. While another ralph org gate, or a\n" +
			"ralph org regate that runs the scripts, holds it, gate waits for it\n" +
			"with no time limit and runs nothing: stderr gets a line that starts\n" +
			"org: gate: waiting for <state dir>/gate.lock and names the holder (its\n" +
			"verb, its org_id or branch, when it took the lock, and its pid) and how\n" +
			"long gate has waited, when the wait starts, when the holder changes,\n" +
			"and every 5 minutes, and one more line once gate took the lock.\n" +
			"\n" +
			"With [org.gate].max_load above 0 (read from the ralph.toml --config\n" +
			"names, else from ./ralph.toml in the directory gate runs in; 0, the\n" +
			"default, turns it off), gate then waits, with the lock held, while the\n" +
			"machine's 1-minute load average is above it: it says so on stderr,\n" +
			"reads the load again every 30 s, and starts once it is not above it,\n" +
			"or after 30 minutes anyway, saying so on stderr and in the log. A load\n" +
			"average that cannot be read starts the gate at once. An interrupt\n" +
			"during either wait records nothing and writes no log.\n" +
			"\n" +
			"After the waits, with the lock held, gate reads the ledger and makes\n" +
			"the checks above again, and refuses the same way when one fails now:\n" +
			"an org disbanded during the wait runs no script, and the scripts run\n" +
			"on the HEAD those checks find, so a commit made during the wait is\n" +
			"what the gate checks.\n" +
			"\n" +
			"Then gate runs ./scripts/run-static-verify.sh and, when it passes,\n" +
			"./scripts/run-test.sh, in the worktree's root. The scripts get the\n" +
			"caller's environment without the RALPH_VERIFY_* and HARNESS_VERIFY_*\n" +
			"variables and with RALPH_VERIFY_SCOPE=full, so no diff narrows what\n" +
			"they check, and with RALPH_LOCAL_CONFIG=off whatever the caller set,\n" +
			"so no ralph.local.toml changes it either. They get gate.lock's open\n" +
			"file too, so a script that runs on after gate was killed keeps the\n" +
			"next gate waiting until it ends. Their output goes to a log under\n" +
			"<state dir>/gates/, not to the worktree. The log's header says how long\n" +
			"gate waited for gate.lock and who held it, how the max_load wait went,\n" +
			"and the load average (1, 5, and 15 minutes) and the CPU count before\n" +
			"the first script; its last line, the load average after the last\n" +
			"script and the highest 1-minute load average read every 30 s while\n" +
			"they ran. They can take minutes, and longer while gate waits for\n" +
			"another: give the command a long timeout, or run it in the background\n" +
			"and wait for it.\n" +
			"\n" +
			"Right after the load average and before the first script, gate\n" +
			"measures the tools the scripts are likely to run, as /bin/sh resolves\n" +
			"them with the scripts' environment: python3, and go when a go.mod is\n" +
			"tracked, at the root; and, when a package.json, a lockfile,\n" +
			".node-version, or .nvmrc is tracked, node and the package manager\n" +
			"(pnpm, yarn, or npm) at the root and in each directory with a lockfile\n" +
			"or a version file (at most 10 places; the log counts the rest). The\n" +
			"log's header gets one line per tool and place,\n" +
			"# tool <tool> (<place>): <version> (<path>), with missing for a tool\n" +
			"not on PATH, unknown (<why>) for a version that cannot be read, and\n" +
			"(mise shim) or (asdf shim) when the path is one, then the\n" +
			"# mise current: lines when mise is on PATH. A child a script starts\n" +
			"through such a shim can see other versions, which gate does not\n" +
			"measure. The probe runs with the variables that turn off mise's\n" +
			"auto-install, corepack's auto-pin and downloads, npm's update check,\n" +
			"and Go's toolchain switching (GOTOOLCHAIN=local), so a version that is\n" +
			"not installed shows as unknown; the scripts get their environment\n" +
			"unchanged. A tool those variables do not reach can still write, so\n" +
			"gate reads git status of the worktree before and after the probe. The\n" +
			"probe's shell has 15 seconds: a tool it did not reach in time is\n" +
			"unknown. Listing the tracked files before it (at most 10 seconds) and\n" +
			"those two reads come on top of that, all while gate.lock is held. A\n" +
			"probe that times out or fails does not change the verdict. When git\n" +
			"status of the worktree differs after the probe, the log lists the\n" +
			"lines, and stderr gets a line that starts org: gate: the version probe\n" +
			"changed the worktree; the scripts run as before, and a pass is\n" +
			"recorded as tree_changed while the change is there.\n" +
			"\n" +
			"When both scripts pass and HEAD and the worktree are as before, gate\n" +
			"records gate_passed in the manifest. Otherwise it records gate_failed\n" +
			"with a reason: static_failed, test_failed, head_moved (HEAD moved while\n" +
			"the scripts ran), or tree_changed (the scripts left files in the\n" +
			"worktree). Both records hold the head, the base, base_ref, the branch,\n" +
			"who ran it (by=<USER>, and herdr_pane=set or unset for whether\n" +
			"HERDR_PANE_ID was set), the load average around the scripts (load_start,\n" +
			"load_max, load_end, and ncpu, each unknown when it could not be read),\n" +
			"max_load when it was above 0, the tools measured (tool_dirs, how many\n" +
			"places, and tool_<tool> per tool, such as tool_node=v24.22.1: its\n" +
			"version, missing, unknown, or mixed when the places differ; each\n" +
			"place's is in the log only), and the log. When the org was disbanded\n" +
			"or started again while the scripts ran, gate records nothing and is\n" +
			"refused.\n" +
			"\n" +
			"On a pass, gate prints GATE: pass, HEAD: <sha>, BASE: <sha>,\n" +
			"LOG: <path>, LOAD: start <1m>, highest <1m>, end <1m> (<n> CPUs), with\n" +
			"how long it waited for max_load when it did, and TOOLS: <tool>\n" +
			"<version>, ... (such as TOOLS: node v24.22.1, pnpm 10.10.0, python3\n" +
			"3.12.1; a value is missing, unknown, or mixed as in the record, and\n" +
			"(see LOG) ends the line when one is mixed) on stdout and exits 0. The\n" +
			"reviewer writes the output to the file the leader's TASK names\n" +
			"(.harness/state/gate-<org_id>.out in the worktree) and copies the HEAD:\n" +
			"and BASE: lines into its RESULT, and the leader checks them against\n" +
			"that file and HEAD before it pushes. The TASK's command sends stderr\n" +
			"to that file too, so it also holds the lines of a wait for gate.lock or\n" +
			"for max_load, and the line of a probe that changed the worktree.\n" +
			"\n" +
			"On a failure, gate prints GATE: fail, REASON: <reason>, HEAD:, BASE:,\n" +
			"LOG:, LOAD: and TOOLS:, then the failing script's last 20 lines of\n" +
			"output, or the files the scripts left, and exits 1. When a script\n" +
			"failed outside the branch's change (a test that timed out on a loaded\n" +
			"machine, say), a human may accept that failure with ralph org gate\n" +
			"accept (see its --help), and ralph org merge-order then lists the\n" +
			"branch as accepted where a pass would be ready. A failure on tool\n" +
			"versions the repository does not ask for (pnpm's\n" +
			"ERR_PNPM_UNSUPPORTED_ENGINE naming a node version, say) is the seat's\n" +
			"environment to fix, not one to accept: ralph doctor's \"Node\n" +
			"toolchain\" check compares the two, and\n" +
			"docs/recipes/isolated-herdr-server.md (Version managers) says how to\n" +
			"pin the versions.\n" +
			"\n" +
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and\n" +
			"records nothing. Processes a script leaves running when it exits are\n" +
			"stopped too, and the log says so. gate starts no autonomous work, so it\n" +
			"needs no running watch and no approved plan.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			// The first SIGINT, SIGTERM, or SIGHUP cancels ctx with a cause
			// that names it ("<signal> signal received", signal.NotifyContext),
			// which Gate's interrupt error prints, and Gate stops the running
			// script's process group (SIGTERM, then SIGKILL after its grace):
			// the group is not ralph's, so a signal to ralph's group or a
			// closed terminal does not reach it. Later ones are caught too
			// until stop runs, so a second Ctrl-C cannot end ralph before the
			// group is stopped.
			ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			// Dir is left empty: Gate reads os.Getwd().
			res, err := rt.Gate(ctx, org.GateParams{OrgID: *orgID, Env: os.Environ(), Stderr: cmd.ErrOrStderr(),
				MaxLoad: rt.Config.Gate.MaxLoad})
			if err != nil {
				return err
			}
			return printGateResult(cmd.OutOrStdout(), res)
		},
	}
	cmd.AddCommand(newOrgGateAcceptCmd(stateDir, configPath))
	return cmd
}

// gateFailedScripts names, for the gate_failed reasons that come from a
// script's exit code, the script whose tail printGateResult shows, as the log
// names it.
var gateFailedScripts = map[string]string{
	org.GateReasonStaticFailed: "./" + org.GateStaticScript,
	org.GateReasonTestFailed:   "./" + org.GateTestScript,
}

// printGateResult prints res and returns the command's error.
//
// A refusal prints nothing on stdout and is the error (exit 1); a refusal at
// the record, after the scripts ran, also names their log.
//
// A pass prints six lines on stdout and returns nil (exit 0):
//
//	GATE: pass
//	HEAD: <sha>
//	BASE: <sha>
//	LOG: <path>
//	LOAD: start <1m>, highest <1m>, end <1m> (<n> CPUs)
//	TOOLS: <tool> <version>, ...
//
// LOAD: (printGateLoad) is printed when the scripts ran (res.Load set), and
// TOOLS: (printGateTools) when the tools were measured before them (res.Tools
// set). A recorded failure prints GATE: fail, REASON: <reason>, HEAD:, BASE:,
// LOG:, LOAD: and TOOLS:, the same way, then a blank line and the detail of
// the reason: the failing script's tail (static_failed, test_failed), the
// files the scripts left with a hint (tree_changed), or one line
// (head_moved). The quoted lines are indented by two spaces, so no line of a
// script's output starts a line like GATE:. The error (exit 1) is one line
// naming the reason.
func printGateResult(w io.Writer, res org.GateResult) error {
	if res.Refused != "" {
		if res.LogPath != "" {
			return fmt.Errorf("%s; the scripts' output is in %s", res.Refused, res.LogPath)
		}
		return errors.New(res.Refused)
	}
	if res.Passed {
		_, _ = fmt.Fprintf(w, "GATE: pass\nHEAD: %s\nBASE: %s\nLOG: %s\n", res.Head, res.Base, res.LogPath)
		printGateLoad(w, res.Load)
		printGateTools(w, res.Tools)
		return nil
	}

	_, _ = fmt.Fprintf(w, "GATE: fail\nREASON: %s\nHEAD: %s\nBASE: %s\nLOG: %s\n", res.Reason, res.Head, res.Base, res.LogPath)
	printGateLoad(w, res.Load)
	printGateTools(w, res.Tools)
	printGateFailDetail(w, res.Reason, res.Tail, res.Dirty, gateCIFailure{}, gateCLIDetailWords)
	return fmt.Errorf("org: gate: recorded gate_failed (reason=%s); nothing to push until the gate passes", res.Reason)
}

// printGateLoad prints the LOAD: line of a gate whose scripts ran, from load
// (nil prints nothing): the 1-minute load average before the first script,
// the highest while they ran, and after the last, each with two decimals or
// "unknown", the CPU count, and, when the gate waited for [org.gate].max_load,
// how long and for which value:
//
//	LOAD: start 16.24, highest 18.03, end 15.90 (12 CPUs), waited 3m0s for max_load 24
func printGateLoad(w io.Writer, load *org.GateLoad) {
	if load == nil {
		return
	}
	cpus := "CPU count unknown"
	if load.NCPU > 0 {
		cpus = fmt.Sprintf("%d CPUs", load.NCPU)
	}
	line := fmt.Sprintf("LOAD: start %s, highest %s, end %s (%s)", load.Start, load.Highest, load.End, cpus)
	if load.Waited > 0 {
		line += fmt.Sprintf(", waited %s for max_load %d", load.Waited.Round(time.Second), load.MaxLoad)
	}
	_, _ = fmt.Fprintln(w, line)
}

// printGateTools prints the TOOLS: line of a gate whose scripts ran, from
// tools (nil prints nothing): each tool measured before them with its value,
// a version, missing, unknown, or mixed when the places it was measured in
// differ, and, when any is mixed, "(see LOG)", whose `# tool` lines have each
// place's (plan 2026-11-12-gate-tool-versions, D5):
//
//	TOOLS: node v24.22.1, pnpm 10.10.0, python3 3.12.1
//	TOOLS: node mixed, pnpm 10.10.0, python3 3.12.1 (see LOG)
func printGateTools(w io.Writer, tools *org.GateTools) {
	if tools == nil {
		return
	}
	parts := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		parts = append(parts, tool.Name+" "+tool.Value)
	}
	line := "TOOLS: " + strings.Join(parts, ", ")
	if len(parts) == 0 {
		line = "TOOLS: none measured"
	}
	if tools.Mixed() {
		line += " (see LOG)"
	}
	_, _ = fmt.Fprintln(w, line)
}

// gateDetailWords is how printGateFailDetail words its next steps for the
// verb that ran the gate: while is what nothing may change the worktree
// during, again the step that runs the verb again.
type gateDetailWords struct {
	while, again string
}

// gateCLIDetailWords is ralph org gate's wording.
var gateCLIDetailWords = gateDetailWords{while: "the gate runs", again: "run the gate again"}

// gateCIFailure is what printGateFailDetail prints for ci_failed, the reason
// of a gate_ci_failed, which only ralph org regate records, from the checks
// of the branch's PR (plan docs/plans/active/2026-10-29-regate-ci-checks.md,
// D4): the PR, the checks that failed (org.RegateResult.FailedChecks, escaped
// by the org layer), and the log that holds the checks' JSON. ralph org gate
// passes the zero value.
type gateCIFailure struct {
	pr     int
	failed []string
	log    string
}

// printGateFailDetail prints the detail of a recorded gate_failed with
// reason, after a blank line: the failing script's tail (static_failed,
// test_failed), the files the scripts left with a hint (tree_changed), or one
// line (head_moved), worded with words; for the ci_failed of a gate_ci_failed,
// ci's failed checks, its log, and a hint. The quoted lines are indented by
// two spaces (printIndented).
func printGateFailDetail(w io.Writer, reason string, tail, dirty []string, ci gateCIFailure, words gateDetailWords) {
	switch reason {
	case org.GateReasonStaticFailed, org.GateReasonTestFailed:
		script := gateFailedScripts[reason]
		switch n := len(tail); n {
		case 0:
			_, _ = fmt.Fprintf(w, "\n%s printed nothing\n", script)
		case 1:
			_, _ = fmt.Fprintf(w, "\nlast line of %s:\n", script)
		default:
			_, _ = fmt.Fprintf(w, "\nlast %d lines of %s:\n", n, script)
		}
		printIndented(w, tail)
	case org.GateReasonTreeChanged:
		shown := dirty[:min(len(dirty), org.GateDirtyListLines)]
		_, _ = fmt.Fprintf(w, "\nthe scripts left %d uncommitted change(s) or untracked file(s) in the feature worktree:\n", len(dirty))
		printIndented(w, shown)
		if more := len(dirty) - len(shown); more > 0 {
			_, _ = fmt.Fprintf(w, "  ... and %d more\n", more)
		}
		_, _ = fmt.Fprintln(w, "hint: when the scripts write these files, add their paths to .gitignore and commit that; then "+words.again)
	case org.GateReasonHeadMoved:
		_, _ = fmt.Fprintln(w, "\nHEAD moved while the scripts ran: commit, reset, or switch nothing in the feature worktree while "+words.while+
			", and run it again")
	case org.GateReasonCIFailed:
		if len(ci.failed) == 0 {
			_, _ = fmt.Fprintf(w, "\nthe checks of PR #%d did not pass\n", ci.pr)
		} else {
			_, _ = fmt.Fprintf(w, "\n%d check(s) of PR #%d ended and are neither pass nor skipping:\n", len(ci.failed), ci.pr)
			printIndented(w, ci.failed)
		}
		if ci.log != "" {
			_, _ = fmt.Fprintf(w, "the JSON of every check gh pr checks listed is in %s\n", ci.log)
		}
		_, _ = fmt.Fprintln(w, "hint: fix the branch in the feature worktree, push the fix to origin, and then "+words.again)
	}
}

// printIndented prints each of lines (none holds a line break) on its own
// line, indented by two spaces.
func printIndented(w io.Writer, lines []string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, "  "+line)
	}
}
