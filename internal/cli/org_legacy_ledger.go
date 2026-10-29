package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// orgLedgerAccess says whether a verb changes the org ledger, which decides
// what guardLegacyOrgStateDir does about a legacy worktree ledger.
type orgLedgerAccess int

const (
	// orgLedgerReadOnly verbs (`ralph status`, `ralph org status` / `read`
	// / `wait` (with or without --inbox, and --inbox --director, which
	// writes only its own mark director-wait.json) / `report` / `inbox` /
	// `inbox show` / `merge-order` / `director status`, `ralph insights`)
	// only print a note about a legacy ledger.
	orgLedgerReadOnly orgLedgerAccess = iota
	// orgLedgerMutating verbs (`ralph org spawn` / `start` / `send` /
	// `stop` / `disband` / `watch` / `escalate` / `approve` / `gate` /
	// `regate` / `merge` / `inbox ack` / `inbox resolve` / `inbox notify` /
	// `director attach` / `director detach` / `director start` /
	// `director stop` / `director launched`) are refused while a legacy
	// ledger still has active seats.
	orgLedgerMutating
)

// errOrgLedgerUnresolved marks resolveOrgLedger's error when
// org.ResolveOrgStateDir could not resolve the ledger, so a caller can tell
// it from guardLegacyOrgStateDir's refusal with errors.Is: `ralph org
// escalate` still tells the human then, writing to no ledger
// (newOrgEscalateCmd). Its text is the middle of the error, between the
// command and org.ResolveOrgStateDir's error.
var errOrgLedgerUnresolved = errors.New("cannot resolve the org ledger")

// resolveOrgLedger resolves the org ledger of cmd's invocation with
// org.ResolveOrgStateDir and then runs guardLegacyOrgStateDir on it with
// access. explicit is the --state-dir value; whether it was passed is
// cmd.Flags().Changed("state-dir"). Every verb that resolves the ledger
// from --state-dir and its defaults goes through this function once per
// invocation (plan 2026-10-30-org-state-dir-local, D6).
//
// A ledger that cannot be resolved (a personal ralph.local.toml that cannot
// be read or parsed, or whose [org].state_dir is neither absolute nor starts
// with "~/") is an error that names the command and, through
// org.ResolveOrgStateDir's error, the file, so the verb exits 1 instead of
// falling back to another ledger (D4). That error is marked with
// errOrgLedgerUnresolved. guardLegacyOrgStateDir's error is returned as it
// is, unmarked.
func resolveOrgLedger(cmd *cobra.Command, explicit string, access orgLedgerAccess) (dir, source string, err error) {
	dir, source, err = org.ResolveOrgStateDir(explicit, cmd.Flags().Changed("state-dir"))
	if err != nil {
		return "", "", fmt.Errorf("%s: %w: %w", cmd.CommandPath(), errOrgLedgerUnresolved, err)
	}
	if err := guardLegacyOrgStateDir(cmd, dir, source, access); err != nil {
		return "", "", err
	}
	return dir, source, nil
}

// guardLegacyOrgStateDir handles the per-worktree org ledger a linked
// worktree kept before org.ResolveOrgStateDir resolved every worktree to the
// main worktree's ledger (plan 2026-10-07-org-state-dir-common; see
// org.LegacyWorktreeStateDir, which finds it). resolvedDir and source are
// what org.ResolveOrgStateDir returned for this invocation. A flag, env, or
// personal-file choice (any source other than "git-main-worktree") has no
// legacy ledger, so this does nothing for it.
//
// A mutating verb gets an error while the legacy ledger still records an
// active seat, or cannot be read: acting on the shared ledger then would,
// for example, let `disband` succeed on an empty roster while the real seats
// keep running. The caller returns that error before any manifest write,
// herdr call, or agmsg call. In every other case where a legacy ledger
// exists, one note goes to stderr (stdout, including --json output, is left
// alone) and the verb continues. Each verb calls this once per invocation,
// so the note prints once.
func guardLegacyOrgStateDir(cmd *cobra.Command, resolvedDir, source string, access orgLedgerAccess) error {
	legacyDir, activeSeats, err := org.LegacyWorktreeStateDir(resolvedDir, source)
	if err != nil {
		if access == orgLedgerMutating {
			return fmt.Errorf("org: refusing to change the org ledger: cannot tell whether this linked worktree's older ledger still has active seats (%w); "+
				"pass --state-dir to choose a ledger, e.g. --state-dir %s for the ledger shared by every worktree",
				err, shellQuoteIfNeeded(resolvedDir))
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"note: cannot read this linked worktree's older org ledger (%v); showing the ledger shared by every worktree at %s.\n",
			err, resolvedDir)
		return nil
	}
	if legacyDir == "" {
		return nil
	}
	if access == orgLedgerMutating && activeSeats > 0 {
		return fmt.Errorf("org: refusing to change the org ledger: this linked worktree still has an older ledger at %s with %d active seat(s), "+
			"and ralph now uses the ledger shared by every worktree at %s. "+
			"Pass --state-dir %s to manage the seats in the older ledger, or --state-dir %s to use the shared one",
			legacyDir, activeSeats, resolvedDir, shellQuoteIfNeeded(legacyDir), shellQuoteIfNeeded(resolvedDir))
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
		"note: this linked worktree has an older org ledger at %s (%d active seat(s)); ralph now uses the ledger shared by every worktree at %s. To use the older one, pass --state-dir %s.\n",
		legacyDir, activeSeats, resolvedDir, shellQuoteIfNeeded(legacyDir))
	return nil
}
