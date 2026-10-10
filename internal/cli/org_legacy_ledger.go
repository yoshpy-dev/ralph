package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// orgLedgerAccess says whether a verb changes the org ledger, which decides
// what guardLegacyOrgStateDir does about a legacy worktree ledger.
type orgLedgerAccess int

const (
	// orgLedgerReadOnly verbs (`ralph status`, `ralph org status` / `read`
	// / `wait` (with or without --inbox) / `report` / `inbox` /
	// `inbox show`, `ralph insights`) only print a note about a legacy
	// ledger.
	orgLedgerReadOnly orgLedgerAccess = iota
	// orgLedgerMutating verbs (`ralph org spawn` / `start` / `send` /
	// `stop` / `disband` / `watch` / `escalate` / `inbox ack` /
	// `inbox resolve` / `inbox notify`) are refused while a legacy ledger
	// still has active seats.
	orgLedgerMutating
)

// guardLegacyOrgStateDir handles the per-worktree org ledger a linked
// worktree kept before org.ResolveOrgStateDir resolved every worktree to the
// main worktree's ledger (plan 2026-10-07-org-state-dir-common; see
// org.LegacyWorktreeStateDir, which finds it). resolvedDir and source are
// what org.ResolveOrgStateDir returned for this invocation. A flag or env
// choice (any source other than "git-main-worktree") has no legacy ledger,
// so this does nothing for it.
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
	// `ralph insights` has no --state-dir flag; the env var is how it picks
	// a state dir.
	useLegacy := "set " + org.EnvOrgStateDir + "=" + shellQuoteIfNeeded(legacyDir)
	if cmd.Flags().Lookup("state-dir") != nil {
		useLegacy = "pass --state-dir " + shellQuoteIfNeeded(legacyDir)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
		"note: this linked worktree has an older org ledger at %s (%d active seat(s)); ralph now uses the ledger shared by every worktree at %s. To use the older one, %s.\n",
		legacyDir, activeSeats, resolvedDir, useLegacy)
	return nil
}
