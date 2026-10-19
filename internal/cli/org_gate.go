package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgGateCmd wires `ralph org gate --org-id <id>` (plan
// docs/plans/active/2026-10-13-org-gate.md, AC1 to AC3 and AC5):
// (*org.Org).Gate checks the feature org and its worktree, runs the two
// scripts, and records gate_passed or gate_failed; this command passes it the
// current directory (Dir left empty) and the process environment, cancels its
// context on SIGINT, SIGTERM, or SIGHUP, and prints the result
// (printGateResult).
func newOrgGateCmd(orgID, stateDir, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "gate",
		Short: "Run the verification scripts on a feature org's head and record gate_passed or gate_failed",
		Long: "ralph org gate records which head of a feature org passed the\n" +
			"verification scripts on which base. The leader of an org started with\n" +
			"`ralph org start --plan` runs it in the org's feature worktree (or a\n" +
			"directory under it), on the head it pushes: after it rebases the\n" +
			"feature branch on the default branch and commits everything, and\n" +
			"before it pushes and opens the PR.\n" +
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
			"Then gate runs ./scripts/run-static-verify.sh and, when it passes,\n" +
			"./scripts/run-test.sh, in the worktree's root. The scripts get the\n" +
			"caller's environment without the RALPH_VERIFY_* and HARNESS_VERIFY_*\n" +
			"variables and with RALPH_VERIFY_SCOPE=full, so no diff narrows what\n" +
			"they check. Their output goes to a log under <state dir>/gates/, not\n" +
			"to the worktree. They can take minutes: give the command a long\n" +
			"timeout, or run it in the background and wait for it.\n" +
			"\n" +
			"When both scripts pass and HEAD and the worktree are as before, gate\n" +
			"records gate_passed in the manifest. Otherwise it records gate_failed\n" +
			"with a reason: static_failed, test_failed, head_moved (HEAD moved while\n" +
			"the scripts ran), or tree_changed (the scripts left files in the\n" +
			"worktree). Both records hold the head, the base, base_ref, the branch,\n" +
			"who ran it (by=<USER>, and herdr_pane=set or unset for whether\n" +
			"HERDR_PANE_ID was set), and the log. When the org was disbanded or\n" +
			"started again while the scripts ran, gate records nothing and is\n" +
			"refused.\n" +
			"\n" +
			"On a pass, gate prints GATE: pass, HEAD: <sha>, BASE: <sha> and\n" +
			"LOG: <path> on stdout and exits 0. The leader copies the HEAD: and\n" +
			"BASE: lines into its RESULT. On a failure, it prints GATE: fail,\n" +
			"REASON: <reason>, HEAD:, BASE: and LOG:, then the failing script's last\n" +
			"20 lines of output, or the files the scripts left, and exits 1.\n" +
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
			// that names it (gateSignalContext), and Gate stops the running
			// script's process group (SIGTERM, then SIGKILL after its grace):
			// the group is not ralph's, so a signal to ralph's group or a
			// closed terminal does not reach it. Later ones are caught too
			// until stop runs, so a second Ctrl-C cannot end ralph before the
			// group is stopped.
			ctx, stop := gateSignalContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			// Dir is left empty: Gate reads os.Getwd().
			res, err := rt.Gate(ctx, org.GateParams{OrgID: *orgID, Env: os.Environ()})
			if err != nil {
				return err
			}
			return printGateResult(cmd.OutOrStdout(), res)
		},
	}
}

// gateSignalContext returns a copy of parent that the first of sigs cancels
// with the cause "<signal> signal received" ("interrupt signal received" for
// SIGINT, "terminated signal received" for SIGTERM, "hangup signal received"
// for SIGHUP), and a stop that cancels it (cause context.Canceled) and
// unregisters sigs. When parent ends first, the cause stays parent's
// ("context canceled" for the command's context). Gate's interrupt error
// prints the cause, so it names the signal.
//
// It does what signal.NotifyContext does from Go 1.26. NotifyContext itself
// is not used: it sets that cause only from Go 1.26, and go.mod allows Go
// 1.25.8, which the release build uses; there the cause is only "context
// canceled" (PR #225's CI).
//
// The goroutine ends at the first signal or when ctx ends, but sigs stay
// registered with signal.Notify until stop runs, so a later signal is caught
// too (and dropped once the channel's one slot is full) and does not end
// ralph.
//
// It is signalCauseContext without its second result, the signal that
// arrived, which only `ralph org wait --inbox --director` reads.
func gateSignalContext(parent context.Context, sigs ...os.Signal) (context.Context, func()) {
	ctx, _, stop := signalCauseContext(parent, sigs...)
	return ctx, stop
}

// signalCauseContext is gateSignalContext with one more result: received
// returns the first of sigs that the goroutine took, which cancelled ctx
// unless ctx had just ended, or nil when it took none (no signal came, or
// parent or stop ended ctx first). stop waits for the goroutine to end, so
// once stop has returned, received no longer changes. With no sigs it
// registers none, since signal.Notify with no signals would relay every
// signal (SIGCHLD and SIGURG too); the director's wait drops the ones that
// were ignored when ralph started (notIgnoredSignals).
func signalCauseContext(parent context.Context, sigs ...os.Signal) (ctx context.Context, received func() os.Signal, stop func()) {
	ctx, cancel := context.WithCancelCause(parent)
	ch := make(chan os.Signal, 1)
	if len(sigs) > 0 {
		signal.Notify(ch, sigs...)
	}
	var (
		mu  sync.Mutex
		got os.Signal
	)
	ended := make(chan struct{})
	if ctx.Err() == nil {
		go func() {
			defer close(ended)
			select {
			case s := <-ch:
				mu.Lock()
				got = s
				mu.Unlock()
				cancel(errors.New(s.String() + " signal received"))
			case <-ctx.Done():
			}
		}()
	} else {
		close(ended)
	}
	received = func() os.Signal {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
	stop = func() {
		cancel(nil)
		<-ended
		signal.Stop(ch)
	}
	return ctx, received, stop
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
// A pass prints four lines on stdout and returns nil (exit 0):
//
//	GATE: pass
//	HEAD: <sha>
//	BASE: <sha>
//	LOG: <path>
//
// A recorded failure prints GATE: fail, REASON: <reason>, HEAD:, BASE: and
// LOG:, the same way, then a blank line and the detail of the reason: the
// failing script's tail (static_failed, test_failed), the files the scripts
// left with a hint (tree_changed), or one line (head_moved). The quoted lines
// are indented by two spaces, so no line of a script's output starts a line
// like GATE:. The error (exit 1) is one line naming the reason.
func printGateResult(w io.Writer, res org.GateResult) error {
	if res.Refused != "" {
		if res.LogPath != "" {
			return fmt.Errorf("%s; the scripts' output is in %s", res.Refused, res.LogPath)
		}
		return errors.New(res.Refused)
	}
	if res.Passed {
		_, _ = fmt.Fprintf(w, "GATE: pass\nHEAD: %s\nBASE: %s\nLOG: %s\n", res.Head, res.Base, res.LogPath)
		return nil
	}

	_, _ = fmt.Fprintf(w, "GATE: fail\nREASON: %s\nHEAD: %s\nBASE: %s\nLOG: %s\n", res.Reason, res.Head, res.Base, res.LogPath)
	printGateFailDetail(w, res.Reason, res.Tail, res.Dirty, gateCLIDetailWords)
	return fmt.Errorf("org: gate: recorded gate_failed (reason=%s); nothing to push until the gate passes", res.Reason)
}

// gateDetailWords is how printGateFailDetail words its next steps for the
// verb that ran the gate: while is what nothing may change the worktree
// during, again the step that runs the verb again.
type gateDetailWords struct {
	while, again string
}

// gateCLIDetailWords is ralph org gate's wording.
var gateCLIDetailWords = gateDetailWords{while: "the gate runs", again: "run the gate again"}

// printGateFailDetail prints the detail of a recorded gate_failed with
// reason, after a blank line: the failing script's tail (static_failed,
// test_failed), the files the scripts left with a hint (tree_changed), or one
// line (head_moved), worded with words. The quoted lines are indented by two
// spaces (printIndented).
func printGateFailDetail(w io.Writer, reason string, tail, dirty []string, words gateDetailWords) {
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
	}
}

// printIndented prints each of lines (none holds a line break) on its own
// line, indented by two spaces.
func printIndented(w io.Writer, lines []string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, "  "+line)
	}
}
