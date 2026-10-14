package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgDirectorCmd wires `ralph org director attach | detach | status`
// (FR-9 of docs/specs/2026-10-07-org-multi-org-director.md, start (A); plan
// docs/plans/active/2026-10-14-org-director-attach.md, D3, D4, D6 and D8)
// on the registration store of internal/org/director.go. The registration
// belongs to the whole ledger, so every verb refuses --org-id
// (newOrgDirectorRuntime). attach and detach change the ledger; status only
// reads it.
func newOrgDirectorCmd(stateDir, configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "director",
		Short: "Register the session the human talks to as the director of the ledger",
		Long: "ralph org director keeps the director registration of the ledger in\n" +
			"<state-dir>/director.jsonl. A director is an interactive session the\n" +
			"human talks to, promoted with the /director skill: it drafts split\n" +
			"plans, starts feature orgs, and handles the org inbox. One director is\n" +
			"registered at a time for every org in the ledger, so these verbs take\n" +
			"no --org-id. attach records the session and prints the registration's\n" +
			"id (d1, d2, ...); it is refused while another director is attached,\n" +
			"unless --replace is given. detach ends the registration and needs that\n" +
			"id, so a session that was replaced is refused and need not detach;\n" +
			"--force ends the current registration whatever its id, for a human\n" +
			"cleaning up. status shows the registration and [org.director].enabled.\n" +
			"\n" +
			"The registration does not change what `ralph org escalate` does:\n" +
			"[org.director].enabled and the cross-org watch decide that (see\n" +
			"ralph org escalate --help). Nor does it mean the session is still\n" +
			"running: an item the director does not ack reaches the human at the\n" +
			"ack deadline of `ralph org watch --all`. A director that ralph starts\n" +
			"by itself (ralph org director start) is not available yet.",
		Args: cobra.NoArgs,
		// Runnable, so a mistyped verb is an unknown command (exit 1), as
		// for `ralph org inbox`, instead of this help.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newOrgDirectorAttachCmd(stateDir, configPath),
		newOrgDirectorDetachCmd(stateDir, configPath),
		newOrgDirectorStatusCmd(stateDir, configPath),
	)

	return cmd
}

func newOrgDirectorAttachCmd(stateDir, configPath *string) *cobra.Command {
	var (
		driver  string
		replace bool
	)

	cmd := &cobra.Command{
		Use:   "attach",
		Short: "Register this session as the director of the ledger",
		Long: "ralph org director attach registers the session that runs it as the\n" +
			"director of the ledger (an attached event in <state-dir>/director.jsonl)\n" +
			"and prints the registration's id: attached d1 (driver=claude). Keep the\n" +
			"id: `ralph org director detach --id <id>` needs it, and\n" +
			"`ralph org director status` shows the current one, so a session that\n" +
			"finds another id there was replaced. --driver is the session's agent,\n" +
			"claude or codex. The event also records who ran attach: USER, the\n" +
			"current directory, and whether HERDR_PANE_ID was set (not its value).\n" +
			"\n" +
			"One director is attached at a time: while one is, attach is refused\n" +
			"unless --replace is given, which ends the current registration and\n" +
			"records the new one in one write, under one lock. The replaced\n" +
			"session's detach is then refused, and it need not detach.\n" +
			"\n" +
			"attach is refused, and writes nothing, when [org.director].enabled is\n" +
			"false: escalate then sends every item to the human at once. Without\n" +
			"--config, enabled is read from the main worktree's ralph.toml when the\n" +
			"ledger is the main worktree's, as `ralph org escalate` reads it.\n" +
			"\n" +
			"When no `ralph org watch --all` accepts work on the ledger, attach still\n" +
			"records the registration and warns on stderr: until one does, escalate\n" +
			"sends items to the human at once instead of leaving them to the\n" +
			"director. Start one with `ralph org watch --all --detach`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch driver {
			case org.DirectorDriverClaude, org.DirectorDriverCodex:
			case "":
				return fmt.Errorf("org: director attach: --driver is required: %s or %s", org.DirectorDriverClaude, org.DirectorDriverCodex)
			default:
				return fmt.Errorf("org: director attach: --driver %q is not %s or %s", driver, org.DirectorDriverClaude, org.DirectorDriverCodex)
			}
			rt, err := newOrgDirectorRuntime(cmd, "director attach", *stateDir, *configPath, orgLedgerMutating, true)
			if err != nil {
				return err
			}
			if !rt.cfg.Director.Enabled {
				return fmt.Errorf("org: director attach: [org.director].enabled is false in %s, so ralph org escalate already sends every item to the human at once "+
					"and leaves nothing to a director; to use one, set [org.director].enabled = true there",
					orgDirectorEnabledFile(*configPath, rt.stateDir, rt.stateDirSource))
			}
			who := orgDirectorCaller()
			reg, replaced, err := rt.store.Attach(org.DirectorAttach{
				Driver: driver, Replace: replace, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane,
			})
			if err != nil {
				if errors.Is(err, org.ErrDirectorAttached) {
					return rt.attachedError(err, driver)
				}
				return err
			}
			line := fmt.Sprintf("attached %s (driver=%s)", reg.ID, reg.Driver)
			if replaced != nil {
				line += fmt.Sprintf(", replacing %s (driver=%s, attached at %s)",
					inboxField(replaced.ID), inboxField(replaced.Driver), inboxField(replaced.AttachedAt))
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, line)
			_, _ = fmt.Fprintf(out, "detach it with: %s\n", rt.command("ralph org director detach --id "+reg.ID))
			probe := orgEscalateWatchProbe(rt.stateDir, rt.cfg.DeadmanMinutes)()
			if !org.WatchAcceptsWork(probe) {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: director attach: %s is recorded, but no ralph org watch --all accepts work on the ledger %s (%s), "+
					"so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with %s\n",
					reg.ID, rt.stateDir, orgDirectorWatchGap(probe), rt.command("ralph org watch --all --detach"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&driver, "driver", "", "the session's agent: claude or codex (required)")
	cmd.Flags().BoolVar(&replace, "replace", false, "end the current registration and record this one instead of refusing")

	return cmd
}

func newOrgDirectorDetachCmd(stateDir, configPath *string) *cobra.Command {
	var (
		id    string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "detach",
		Short: "End the director registration of the ledger",
		Long: "ralph org director detach ends the current director registration of the\n" +
			"ledger (a detached event in <state-dir>/director.jsonl). Give --id, the\n" +
			"id attach printed. When it is not the current registration's, detach is\n" +
			"refused and writes nothing: that session was replaced with\n" +
			"`ralph org director attach --replace` (or its registration has already\n" +
			"ended), and it need not detach. --force ends the current registration\n" +
			"whatever its id, for a human cleaning up after a session that is gone,\n" +
			"and is recorded as forced; it cannot be combined with --id. With no\n" +
			"registration, detach exits 1. detach reads no ralph.toml (nor the main\n" +
			"worktree's), so one that does not load does not block it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "" && !force:
				return errors.New("org: director detach: give --id <id>, the id attach printed, or --force")
			case id != "" && force:
				return errors.New("org: director detach: --id and --force cannot be used together")
			}
			rt, err := newOrgDirectorRuntime(cmd, "director detach", *stateDir, *configPath, orgLedgerMutating, false)
			if err != nil {
				return err
			}
			who := orgDirectorCaller()
			ended, err := rt.store.Detach(org.DirectorDetach{
				ID: id, Force: force, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane,
			})
			if err != nil {
				return err
			}
			if force {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "detached %s (forced; driver=%s, attached at %s)\n",
					inboxField(ended.ID), inboxField(ended.Driver), inboxField(ended.AttachedAt))
				return nil
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "detached %s\n", ended.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "the id of the registration to end (d1, d2, ...), as attach printed it")
	cmd.Flags().BoolVar(&force, "force", false, "end the current registration whatever its id (a human's cleanup)")

	return cmd
}

func newOrgDirectorStatusCmd(stateDir, configPath *string) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the director registration of the ledger and [org.director].enabled",
		Long: "ralph org director status shows [org.director].enabled, read as\n" +
			"`ralph org escalate` reads it, and the current director registration of\n" +
			"the ledger: the director: line of `ralph status` (attached, not\n" +
			"attached, disabled, or unreadable), then enabled and, when a director is\n" +
			"attached, the registration's fields, or all of it as JSON with --json.\n" +
			"With enabled false the state is disabled whether or not a registration is\n" +
			"recorded; one that is still recorded is named and its fields are shown.\n" +
			"The registration does not tell whether its session is still running.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := newOrgDirectorRuntime(cmd, "director status", *stateDir, *configPath, orgLedgerReadOnly, true)
			if err != nil {
				return err
			}
			report := readOrgDirectorReport(rt.stateDir, rt.cfg.Director.Enabled)
			warnDirectorDamage(cmd.ErrOrStderr(), rt.store.Path(), report)
			if jsonOut {
				return writeIndentedJSON(cmd.OutOrStdout(), report)
			}
			printOrgDirectorReport(cmd.OutOrStdout(), report)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

	return cmd
}

// orgDirectorRuntime is what the director verbs work on: the registration
// store of the ledger, the state dir org.ResolveOrgStateDir resolved with its
// source, the --state-dir the printed commands repeat
// (orgInboxHintStateDir), and the [org] config with Director.Enabled read by
// withMainWorktreeDirectorEnabled (the zero value for detach, which reads
// none).
type orgDirectorRuntime struct {
	store          *org.DirectorStore
	stateDir       string
	stateDirSource string
	hintStateDir   string
	cfg            config.OrgConfig
}

// newOrgDirectorRuntime builds the orgDirectorRuntime of a director verb. The
// registration belongs to the whole ledger, so an --org-id is refused rather
// than ignored, as the inbox verbs refuse it (newOrgInboxRuntime). access
// is the verb's orgLedgerAccess for guardLegacyOrgStateDir. loadConfig is
// set for attach and status, which read [org.director].enabled: for them a
// ralph.toml that does not load, or a main worktree ralph.toml that does
// not, is an error. detach passes false and reads neither, so a config that
// does not load does not block ending a registration (self-review L4 of
// org-director-attach).
func newOrgDirectorRuntime(cmd *cobra.Command, verb, stateDir, configPath string, access orgLedgerAccess, loadConfig bool) (*orgDirectorRuntime, error) {
	if cmd.Flags().Changed("org-id") {
		return nil, fmt.Errorf("org: %s: --org-id does not apply: the director registration belongs to the whole ledger, not to one org", verb)
	}
	resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
	if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, access); err != nil {
		return nil, err
	}
	var cfg config.OrgConfig
	if loadConfig {
		var err error
		if cfg, err = resolveOrgConfig(configPath); err != nil {
			return nil, fmt.Errorf("org: load config: %w", err)
		}
		if cfg, err = withMainWorktreeDirectorEnabled(cfg, configPath, resolvedStateDir, stateDirSource); err != nil {
			return nil, err
		}
	}
	return &orgDirectorRuntime{
		store:          org.NewDirectorStore(resolvedStateDir),
		stateDir:       resolvedStateDir,
		stateDirSource: stateDirSource,
		hintStateDir:   orgInboxHintStateDir(cmd, resolvedStateDir),
		cfg:            cfg,
	}, nil
}

// command is command followed by the --state-dir of this invocation, when
// one was given (org.CommandWithStateDir).
func (rt *orgDirectorRuntime) command(command string) string {
	return org.CommandWithStateDir(command, rt.hintStateDir)
}

// attachedError is the store's ErrDirectorAttached refusal err followed by
// the commands that end or replace the current registration. The store's
// message names the registration and ends there, so the two ways out are
// said once, here (self-review L7 of org-director-attach). The id for the
// detach command is read again here, without the lock, so it can be newer
// than the one the store refused on.
func (rt *orgDirectorRuntime) attachedError(err error, driver string) error {
	detach := "ralph org director detach --id <its id>"
	if res, readErr := rt.store.Read(); readErr == nil && res.Current != nil {
		detach = "ralph org director detach --id " + inboxField(res.Current.ID)
	}
	return fmt.Errorf("%w: end it with %s from the session that attached it (or with --force in place of --id, when that session is gone), "+
		"or replace it with %s", err, rt.command(detach), rt.command("ralph org director attach --driver "+driver+" --replace"))
}

// orgDirectorEnabledFile names the file [org.director].enabled was read from
// for the director verbs, by the rule of withMainWorktreeDirectorEnabled:
// the --config file, else the main worktree's ralph.toml when the ledger is
// the main worktree's, else ./ralph.toml (resolveOrgConfig).
func orgDirectorEnabledFile(configPath, resolvedStateDir, stateDirSource string) string {
	if configPath != "" {
		return "the --config file " + configPath
	}
	if root, ok := org.LedgerMainWorktreeRoot(resolvedStateDir, stateDirSource); ok {
		return "the main worktree's " + filepath.Join(root, "ralph.toml")
	}
	if abs, err := filepath.Abs("ralph.toml"); err == nil {
		return abs
	}
	return "./ralph.toml"
}

// orgDirectorWho is who runs attach or detach, as the director event records
// it: USER (empty when unset), the current directory ("" when os.Getwd
// fails; it is for an audit only), and whether HERDR_PANE_ID is set, never
// its value.
type orgDirectorWho struct {
	User      string
	Cwd       string
	HerdrPane bool
}

func orgDirectorCaller() orgDirectorWho {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return orgDirectorWho{User: os.Getenv("USER"), Cwd: cwd, HerdrPane: os.Getenv("HERDR_PANE_ID") != ""}
}

// orgDirectorWatchGap says why the watch probe found accepts no work, in the
// words of escalate's help.
func orgDirectorWatchGap(probe org.WatchLiveness) string {
	switch probe.State {
	case org.WatchLivenessStale:
		return "its liveness mark is stale"
	case org.WatchLivenessStopping:
		return "it is stopping"
	case org.WatchLivenessRunning:
		return "it is a --once run"
	}
	return "none runs"
}

// The states of orgDirectorReport.State.
const (
	orgDirectorStateAttached    = "attached"
	orgDirectorStateNotAttached = "not_attached"
	orgDirectorStateDisabled    = "disabled"
	orgDirectorStateUnreadable  = "unreadable"
)

// orgDirectorReport is the director switch and registration of a ledger as
// `ralph status` (its director: line and director key) and `ralph org
// director status` report them, and their --json wire shape. State is
// disabled when Enabled is false, whatever the registration; otherwise
// unreadable when director.jsonl could not be read, attached when a
// registration is current, and not_attached. Registration is the current
// registration (null when there is none or the file could not be read),
// Error the read error, set whenever the read failed (with disabled too),
// and CorruptLines and IgnoredEvents the counts of org.DirectorReadResult.
type orgDirectorReport struct {
	Enabled       bool                      `json:"enabled"`
	State         string                    `json:"state"`
	Registration  *org.DirectorRegistration `json:"registration"`
	Error         string                    `json:"error,omitempty"`
	CorruptLines  int                       `json:"corrupt_lines"`
	IgnoredEvents int                       `json:"ignored_events"`
}

// readOrgDirectorReport reads the registration of the ledger at stateDir
// (org.DirectorStore.Read, no lock) into an orgDirectorReport with enabled
// as the switch.
func readOrgDirectorReport(stateDir string, enabled bool) orgDirectorReport {
	report := orgDirectorReport{Enabled: enabled}
	res, err := org.NewDirectorStore(stateDir).Read()
	switch {
	case err != nil:
		report.Error, report.State = err.Error(), orgDirectorStateUnreadable
	case res.Current != nil:
		report.Registration, report.State = res.Current, orgDirectorStateAttached
	default:
		report.State = orgDirectorStateNotAttached
	}
	report.CorruptLines, report.IgnoredEvents = res.CorruptLines, res.IgnoredEvents
	if !enabled {
		report.State = orgDirectorStateDisabled
	}
	return report
}

// orgDirectorLine is the one director: line that `ralph status` and `ralph
// org director status` print for r. The values from director.jsonl, which a
// human can edit, have their control characters escaped (inboxField), so
// the line stays one line.
func orgDirectorLine(r orgDirectorReport) string {
	switch r.State {
	case orgDirectorStateAttached:
		reg := r.Registration
		details := "driver=" + inboxField(reg.Driver) + ", attached at " + inboxField(reg.AttachedAt)
		if reg.User != "" {
			details += ", by " + inboxField(reg.User)
		}
		if reg.PaneID != "" {
			details += ", pane " + inboxField(reg.PaneID)
		}
		return "director: attached " + inboxField(reg.ID) + " (" + details + ") -- the registration does not tell whether its session is still running; " +
			"inbox items it does not ack reach the human at the ack deadline"
	case orgDirectorStateDisabled:
		note := "[org.director].enabled = false"
		switch {
		case r.Error != "":
			note += "; director.jsonl cannot be read: " + inboxField(r.Error)
		case r.Registration != nil:
			note += "; registration " + inboxField(r.Registration.ID) + " is still recorded"
		}
		return "director: disabled (" + note + ") -- ralph org escalate sends every item to the human at once"
	case orgDirectorStateUnreadable:
		return "director: unreadable (" + inboxField(r.Error) + ")"
	}
	return "director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work"
}

// warnDirectorDamage writes to w (stderr, so --json output stays clean) the
// lines of director.jsonl at path that did not parse and the events the fold
// ignored, as warnInboxDamage does for the inbox.
func warnDirectorDamage(w io.Writer, path string, r orgDirectorReport) {
	if r.CorruptLines > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d unreadable director line(s) skipped in %s\n", r.CorruptLines, path)
	}
	if r.IgnoredEvents > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d director event(s) ignored in %s (unknown kind or id, an attached with an unknown driver, a reused id, "+
			"or while another registration is current, or a detached of an id that is not the current one)\n", r.IgnoredEvents, path)
	}
}

// printOrgDirectorReport prints `ralph org director status`'s text form: the
// director: line (orgDirectorLine), enabled, and, when a registration is
// current, one line per field of it, with control characters escaped.
// pane_id is left out when not set.
func printOrgDirectorReport(w io.Writer, r orgDirectorReport) {
	_, _ = fmt.Fprintln(w, orgDirectorLine(r))
	_, _ = fmt.Fprintf(w, "enabled: %t\n", r.Enabled)
	reg := r.Registration
	if reg == nil {
		return
	}
	for _, f := range []struct{ name, value string }{
		{"id", reg.ID}, {"driver", reg.Driver}, {"wake", strings.Join(reg.Wake, ",")},
		{"attached_at", reg.AttachedAt}, {"user", reg.User}, {"cwd", reg.Cwd},
		{"herdr_pane", yesNo(reg.HerdrPane)},
	} {
		_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
	}
	if reg.PaneID != "" {
		_, _ = fmt.Fprintf(w, "pane_id: %s\n", inboxField(reg.PaneID))
	}
}
