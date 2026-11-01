package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/gitroot"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// orgLedgerAccess says whether a verb changes the org ledger, which decides
// what guardLegacyOrgStateDir does about a legacy worktree ledger and what
// checkRetiredOrgStateDirEnv does while the retired RALPH_ORG_STATE_DIR
// names another directory than the ledger the default tiers found.
type orgLedgerAccess int

const (
	// orgLedgerReadOnly verbs (`ralph status`, `ralph org status` / `read`
	// / `wait` (with or without --inbox, and --inbox --director, which
	// writes only its own mark director-wait.json) / `report` / `inbox` /
	// `inbox show` / `merge-order` / `director status` / `followup list`,
	// `ralph insights`) only print a note about a legacy ledger, and about
	// RALPH_ORG_STATE_DIR naming another directory than the default ledger.
	orgLedgerReadOnly orgLedgerAccess = iota
	// orgLedgerMutating verbs (`ralph org spawn` / `start` / `send` /
	// `stop` / `disband` / `watch` / `escalate` / `approve` / `gate` /
	// `regate` / `merge` / `cleanup` / `inbox ack` / `inbox resolve` /
	// `inbox notify` / `director attach` / `director detach` /
	// `director start` / `director stop` / `director launched` /
	// `followup add` / `followup done`) are refused while a legacy ledger
	// still has active seats, and while RALPH_ORG_STATE_DIR names another
	// directory than the default ledger (checkRetiredOrgStateDirEnv).
	orgLedgerMutating
)

// errOrgLedgerUnresolved marks resolveOrgLedger's error when
// org.ResolveOrgStateDir could not resolve the ledger, or when
// checkRetiredOrgStateDirEnv refused a mutating verb, so a caller can tell
// it from guardLegacyOrgStateDir's refusal with errors.Is: `ralph org
// escalate` still tells the human then, writing to no ledger
// (newOrgEscalateCmd). Its text is the middle of the error, between the
// command and the cause.
var errOrgLedgerUnresolved = errors.New("cannot resolve the org ledger")

// retiredOrgStateDirEnv is the variable that named the org ledger before
// the personal file's [org].state_dir and --state-dir took its place (plan
// 2026-10-30-org-state-dir-local, D5). org.ResolveOrgStateDir no longer
// reads it. checkRetiredOrgStateDirEnv reads it only to say so, or to refuse
// a verb that would change another ledger than the one it names, and `ralph
// doctor` warns while it is set (retiredEnvs).
const retiredOrgStateDirEnv = "RALPH_ORG_STATE_DIR"

// retiredOrgStateDirRefusalCondition is the condition checkRetiredOrgStateDirEnv
// applies when it refuses a verb that changes the ledger: neither --state-dir
// nor [org].state_dir names the ledger (the source is not "flag" or
// "config"), and the variable names another directory than the ledger the
// default tiers found (not gitroot.SamePath). `ralph doctor`'s warn for the
// variable (retiredEnvs) and the help of `ralph org escalate`
// (newOrgEscalateCmd) say it in these words, and
// TestRetiredOrgStateDirRefusalCondition runs the cases through the judgment
// and checks both sentences against it.
const retiredOrgStateDirRefusalCondition = "neither --state-dir nor [org].state_dir names the ledger and " +
	retiredOrgStateDirEnv + " names another directory than the ledger ralph finds by default"

// resolveOrgLedger resolves the org ledger of cmd's invocation with
// org.ResolveOrgStateDir, then runs checkRetiredOrgStateDirEnv and
// guardLegacyOrgStateDir on it with access. explicit is the --state-dir
// value; whether it was passed is cmd.Flags().Changed("state-dir"). Every
// verb that resolves the ledger from --state-dir and its defaults goes
// through this function once per invocation (plan
// 2026-10-30-org-state-dir-local, D6), so each note prints once.
//
// A ledger that cannot be resolved (a personal ralph.local.toml that cannot
// be read or parsed, or whose [org].state_dir is neither absolute nor starts
// with "~/") is an error that names the command and, through
// org.ResolveOrgStateDir's error, the file, so the verb exits 1 instead of
// falling back to another ledger (D4). So is checkRetiredOrgStateDirEnv's
// refusal. Both errors are marked with errOrgLedgerUnresolved.
// guardLegacyOrgStateDir's error is returned as it is, unmarked.
func resolveOrgLedger(cmd *cobra.Command, explicit string, access orgLedgerAccess) (dir, source string, err error) {
	dir, source, err = org.ResolveOrgStateDir(explicit, cmd.Flags().Changed("state-dir"))
	if err != nil {
		return "", "", fmt.Errorf("%s: %w: %w", cmd.CommandPath(), errOrgLedgerUnresolved, err)
	}
	if err := checkRetiredOrgStateDirEnv(cmd, dir, source, access); err != nil {
		return "", "", fmt.Errorf("%s: %w: %w", cmd.CommandPath(), errOrgLedgerUnresolved, err)
	}
	if err := guardLegacyOrgStateDir(cmd, dir, source, access); err != nil {
		return "", "", err
	}
	return dir, source, nil
}

// checkRetiredOrgStateDirEnv handles retiredOrgStateDirEnv for the ledger
// org.ResolveOrgStateDir returned for this invocation (resolvedDir, with its
// source), so that a user whose ledger the variable named does not go on
// with another ledger unawares (plan 2026-10-30-org-state-dir-local, D5).
// It does nothing while the variable is unset or blank. Its value is made
// absolute against the working directory, as the retired tier did.
//
// When --state-dir or the personal file named the ledger (source "flag" or
// "config"), or the default tiers found the directory the variable names
// (gitroot.SamePath), one note goes to stderr and the verb goes on. When the
// default tiers found another directory, a read-only verb gets a note that
// also says the verbs that change the ledger are refused, and a mutating
// verb gets an error, which it returns before any manifest write, watch
// launch, herdr call, or agmsg call. The error names both directories and
// how to keep using the variable's: a verb that changed the default ledger
// would start work there while the orgs of the variable's ledger keep
// running, and the org-wide limits and reservations would be counted in two
// ledgers.
func checkRetiredOrgStateDirEnv(cmd *cobra.Command, resolvedDir, source string, access orgLedgerAccess) error {
	value := strings.TrimSpace(os.Getenv(retiredOrgStateDirEnv))
	if value == "" {
		return nil
	}
	envDir, err := filepath.Abs(value)
	if err != nil {
		envDir = value
	}
	const choose = "Set [org].state_dir in " + config.LocalConfigFileName + " or pass --state-dir to choose the ledger, and unset " + retiredOrgStateDirEnv + "."
	if source == "flag" || source == "config" || gitroot.SamePath(envDir, resolvedDir) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: %s is set (%s), but ralph no longer reads it; using the org ledger at %s. %s\n",
			retiredOrgStateDirEnv, envDir, resolvedDir, choose)
		return nil
	}
	if access == orgLedgerMutating {
		// On its default tiers org.ResolveOrgStateDir read the personal file
		// in the directory it joined with .harness/state/org, so the path
		// comes from resolvedDir with no git call.
		localPath, ok := org.LedgerLocalConfigPath(resolvedDir, source)
		if !ok {
			localPath = config.LocalPath("ralph.toml")
		}
		return fmt.Errorf("%s names %s, which ralph no longer reads, and the ledger ralph finds by default is %s, so nothing was changed; "+
			"to keep using %s, write [org].state_dir = %q in %s or pass --state-dir %s, and unset %s; to use %s, unset %s",
			retiredOrgStateDirEnv, envDir, resolvedDir,
			envDir, envDir, localPath, shellQuoteIfNeeded(envDir), retiredOrgStateDirEnv,
			resolvedDir, retiredOrgStateDirEnv)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: %s is set (%s), but ralph no longer reads it; showing the org ledger at %s instead, "+
		"and the commands that change the ledger are refused while the variable names another one. %s\n",
		retiredOrgStateDirEnv, envDir, resolvedDir, choose)
	return nil
}

// guardLegacyOrgStateDir handles the per-worktree org ledger a linked
// worktree kept before org.ResolveOrgStateDir resolved every worktree to the
// main worktree's ledger (plan 2026-10-07-org-state-dir-common; see
// org.LegacyWorktreeStateDir, which finds it). resolvedDir and source are
// what org.ResolveOrgStateDir returned for this invocation. A flag or
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
