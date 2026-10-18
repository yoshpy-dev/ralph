package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgDirectorCmd wires `ralph org director attach | detach | start | stop
// | status` (FR-9 of docs/specs/2026-10-07-org-multi-org-director.md; plan
// docs/plans/archive/2026-10-14-org-director-attach.md, D3, D4, D6 and D8,
// for start (A), and plan docs/plans/active/2026-10-15-org-director-start.md
// for the headless director of start (B)) on the registration store of
// internal/org/director.go and the org-layer verbs StartDirector and
// StopDirector. The registration belongs to the whole ledger, so every verb
// refuses --org-id (newOrgDirectorRuntime). attach, detach, start, and stop
// change the ledger; status only reads it.
func newOrgDirectorCmd(stateDir, configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "director",
		Short: "Register the session the human talks to as the director of the ledger",
		Long: "ralph org director keeps the director registration of the ledger in\n" +
			"<state-dir>/director.jsonl. A director is the session the human talks\n" +
			"to: it drafts split plans, starts feature orgs, and handles the org\n" +
			"inbox. There are two ways to have one. An interactive session is\n" +
			"promoted with the /director skill, which runs attach. Or\n" +
			"`ralph org director start` launches a headless director in a herdr\n" +
			"workspace of its own and registers it, and\n" +
			"`ralph org director stop --id <id>` closes it again. With\n" +
			"[org.director].launch_cmd set, start launches it with that command,\n" +
			"which opens a terminal, and stop ends its process. One director is\n" +
			"registered at a time for every org in the ledger, so these verbs take\n" +
			"no --org-id. attach records the session and prints the registration's\n" +
			"id (d1, d2, ...); it is refused while another director is attached,\n" +
			"unless --replace is given. detach ends the registration and needs that\n" +
			"id, so a session that was replaced is refused and need not detach;\n" +
			"--force ends the current registration whatever its id, for a human\n" +
			"cleaning up. status shows the registration, the headless directors that\n" +
			"are still open, [org.director].enabled, and [org.director].auto_merge.\n" +
			"\n" +
			"The registration does not change what `ralph org escalate` does:\n" +
			"[org.director].enabled and the cross-org watch decide that (see\n" +
			"ralph org escalate --help). Nor does it mean a promoted session is\n" +
			"still running; the cross-org watch checks a headless director each\n" +
			"cycle (see status). An item the director does not ack reaches the\n" +
			"human at the ack deadline of `ralph org watch --all`.",
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
		newOrgDirectorStartCmd(stateDir, configPath),
		newOrgDirectorStopCmd(stateDir, configPath),
		newOrgDirectorStatusCmd(stateDir, configPath),
		// Hidden: only the launch script of a launch_cmd start runs it.
		newOrgDirectorLaunchedCmd(stateDir, configPath),
	)

	return cmd
}

func newOrgDirectorAttachCmd(stateDir, configPath *string) *cobra.Command {
	var (
		driver     string
		replace    bool
		wakeFlag   string
		wakeTarget string
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
			"session's detach is then refused, and it need not detach. A headless\n" +
			"director that `ralph org director start` launched counts too, starting\n" +
			"or running: the refusal then names `ralph org director stop --id`,\n" +
			"which closes its herdr pane, and --replace leaves that pane open. For\n" +
			"one launched with [org.director].launch_cmd, that stop ends its\n" +
			"process, run from a terminal other than the director's, and --replace\n" +
			"leaves the process running. The refusal then also says what\n" +
			"`ralph org watch --all` found when it last checked that director, and,\n" +
			"for one still starting, what ps reads now of the start process that\n" +
			"recorded it; when either says it has ended, the refusal names\n" +
			"`ralph org director start`, which cleans it up before it starts a new\n" +
			"headless director. attach itself runs no herdr and cleans up nothing.\n" +
			"\n" +
			"attach is refused, and writes nothing, when [org.director].enabled is\n" +
			"false: escalate then sends every item to the human at once. Without\n" +
			"--config, enabled is read from the main worktree's ralph.toml when the\n" +
			"ledger is the main worktree's, as `ralph org escalate` reads it.\n" +
			"\n" +
			"--wake is how the director learns of a new open inbox item: pull (it\n" +
			"waits on the inbox with `ralph org wait --inbox`), push, or both\n" +
			"(pull,push). With push, `ralph org watch --all` wakes it by running\n" +
			"[org.director].wake_cmd with {target} replaced by --wake-target and\n" +
			"{text} by one fixed line that names the new items. Without --wake,\n" +
			"attach records [org.director].wake. attach is refused, and writes\n" +
			"nothing (with --replace, the current registration stays), when --wake\n" +
			"has a value other than pull and push or one twice, when the wake\n" +
			"has push and wake_cmd is empty, when wake_cmd has {target} and no\n" +
			"--wake-target is given, when --wake-target is given and the wake has no\n" +
			"push, and when --wake-target is empty, has a control character, is\n" +
			"longer than 256 characters, or starts with \"-\". Without --config,\n" +
			"wake and wake_cmd are read from the main worktree's ralph.toml by the\n" +
			"rule of enabled. The success line adds wake= (and wake_target=) when the\n" +
			"wake has push: attached d1 (driver=claude, wake=push, wake_target=...).\n" +
			"\n" +
			"When no `ralph org watch --all` accepts work on the ledger, attach still\n" +
			"records the registration and warns on stderr: until one does, escalate\n" +
			"sends items to the human at once instead of leaving them to the\n" +
			"director. Start one with `ralph org watch --all --detach`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOrgDirectorDriver("director attach", driver); err != nil {
				return err
			}
			flagWake, err := checkOrgDirectorAttachWakeFlags(cmd, wakeFlag, wakeTarget)
			if err != nil {
				return err
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
			wake, err := rt.attachWake(*configPath, flagWake, wakeTarget)
			if err != nil {
				return err
			}
			who := orgDirectorCaller()
			reg, replaced, err := rt.store.Attach(org.DirectorAttach{
				Driver: driver, Replace: replace, Wake: wake, WakeTarget: wakeTarget, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane,
			})
			if err != nil {
				if errors.Is(err, org.ErrDirectorAttached) {
					return rt.attachedError(err, driver)
				}
				return rt.notRestoredError(err)
			}
			line := fmt.Sprintf("attached %s (driver=%s%s)", reg.ID, reg.Driver, orgDirectorAttachWakeDetails(reg))
			if replaced != nil {
				line += fmt.Sprintf(", replacing %s (driver=%s, attached at %s%s)",
					inboxField(replaced.ID), inboxField(replaced.Driver), inboxField(replaced.AttachedAt), orgDirectorAttachWakeDetails(*replaced))
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, line)
			_, _ = fmt.Fprintf(out, "detach it with: %s\n", rt.command("ralph org director detach --id "+reg.ID))
			if replaced != nil {
				rt.printHeadlessStillOpen(out, *replaced)
			}
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
	cmd.Flags().StringVar(&wakeFlag, "wake", "", "how the director learns of a new inbox item: pull, push, or pull,push (default: [org.director].wake); "+
		"push has ralph org watch --all wake it by running [org.director].wake_cmd with {target} = --wake-target")
	cmd.Flags().StringVar(&wakeTarget, "wake-target", "", "where [org.director].wake_cmd sends the wake line, its {target} (such as a terminal's handle); "+
		"only with a wake that has push")

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
			"worktree's), so one that does not load does not block it. detach only\n" +
			"records: when it ends a headless director's registration\n" +
			"(`ralph org director start`), the director's herdr pane and workspace\n" +
			"stay open, and detach prints the `ralph org director stop --id <id>`\n" +
			"that closes them. For one launched with [org.director].launch_cmd, its\n" +
			"process keeps running, and the stop detach prints ends it.",
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
				return rt.notRestoredError(err)
			}
			out := cmd.OutOrStdout()
			if force {
				_, _ = fmt.Fprintf(out, "detached %s (forced; driver=%s, attached at %s)\n",
					inboxField(ended.ID), inboxField(ended.Driver), inboxField(ended.AttachedAt))
			} else {
				_, _ = fmt.Fprintf(out, "detached %s\n", ended.ID)
			}
			rt.printHeadlessStillOpen(out, ended)
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
		Short: "Show the director registration of the ledger, [org.director].enabled, and auto_merge",
		Long: "ralph org director status shows [org.director].enabled, read as\n" +
			"`ralph org escalate` reads it, [org.director].auto_merge, read from the\n" +
			"same ralph.toml, and the current director registration of the ledger:\n" +
			"the director: line of `ralph status` (attached, not attached, disabled,\n" +
			"or unreadable), then enabled, auto_merge and, when a director is\n" +
			"attached, the registration's fields, or all of it as JSON with --json.\n" +
			"For a headless director (ralph org director start) the fields add its\n" +
			"state (starting or running), model, permission mode, token, workspace,\n" +
			"and pane (for one launched with [org.director].launch_cmd, launch cmd,\n" +
			"the pid and start_time its launch script checked in with, and\n" +
			"terminal_name, the name director_<token> launch_cmd gave its terminal,\n" +
			"in place of the workspace and pane), and the headless directors that are\n" +
			"no longer attached but still open are listed (open_headless), to be\n" +
			"closed with `ralph org director stop --id <id>`.\n" +
			"The fields show the registration's wake, and wake_target when attach was\n" +
			"given --wake-target. For a registration whose wake has push, the\n" +
			"director: line adds wake= and where `ralph org watch --all` sends the\n" +
			"wake line: the --wake-target, the herdr pane, or the terminal name.\n" +
			"With enabled false the state is disabled whether or not a registration is\n" +
			"recorded; one that is still recorded is named and its fields are shown.\n" +
			"The director: line ends with what auto_merge means: with false, merges\n" +
			"are left to the human; with true, the director (once one is attached)\n" +
			"merges the PRs `ralph org merge-order` lists as ready with\n" +
			"`ralph org merge`, without asking. With enabled false, auto_merge = true\n" +
			"has no effect, and the line says so.\n" +
			"The registration does not tell whether its session is still running\n" +
			"for a promoted session (the /director skill,\n" +
			"`ralph org director attach`). For a headless director the director:\n" +
			"line says instead what `ralph org watch --all`, which checks it each\n" +
			"cycle, last found: checked by the watch at <time>: alive (<how>), ended\n" +
			"(<how>), or unknown (<how>: <detail>); or not checked yet, before the\n" +
			"watch has checked this registration (with why, when the watch's mark\n" +
			"director-alive.json cannot be read). An ended one is cleaned up by the\n" +
			"next `ralph org director start`, which checks it again first (with\n" +
			"enabled false start is refused, and the line names only the stop), or\n" +
			"stopped with `ralph org director stop --id <id>`. --json carries what\n" +
			"the watch found as check (state, how, checked_at, detail;\n" +
			"director.check in `ralph status --json`), left out until the watch has\n" +
			"checked the current registration, and the registration of a headless\n" +
			"director carries starter_pid and starter_start_time: the pid of the\n" +
			"`ralph org director start` process that recorded it and that process's\n" +
			"start time, when start could read them.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := newOrgDirectorRuntime(cmd, "director status", *stateDir, *configPath, orgLedgerReadOnly, true)
			if err != nil {
				return err
			}
			report := readOrgDirectorReport(rt.stateDir, rt.cfg.Director)
			warnDirectorDamage(cmd.ErrOrStderr(), rt.store.Path(), report)
			if jsonOut {
				return writeIndentedJSON(cmd.OutOrStdout(), report)
			}
			printOrgDirectorReport(cmd.OutOrStdout(), report, rt.hintStateDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

	return cmd
}

// checkOrgDirectorDriver refuses, for verb (director attach or director
// start), a --driver other than claude and codex, or none, before anything
// is read.
func checkOrgDirectorDriver(verb, driverName string) error {
	switch driverName {
	case org.DirectorDriverClaude, org.DirectorDriverCodex:
		return nil
	case "":
		return fmt.Errorf("org: %s: --driver is required: %s or %s", verb, org.DirectorDriverClaude, org.DirectorDriverCodex)
	}
	return fmt.Errorf("org: %s: --driver %q is not %s or %s", verb, driverName, org.DirectorDriverClaude, org.DirectorDriverCodex)
}

// orgDirectorRuntime is what the director verbs work on: the registration
// store of the ledger, the state dir org.ResolveOrgStateDir resolved with its
// source, the --state-dir the printed commands repeat
// (orgInboxHintStateDir), and the [org] config with Director.Enabled and
// Director.AutoMerge read by withMainWorktreeDirectorEnabled (the zero value
// for detach and stop, which read none, and for start, which loads its own:
// startDirector). attach reads Director.Wake and Director.WakeCmd on top of
// it, by withMainWorktreeDirectorWakeConfig (attachWake), so status, which
// shows the recorded wake, reads neither.
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
// set for attach and status, which read [org.director].enabled (status
// auto_merge too): for them a ralph.toml that does not load, or a main
// worktree ralph.toml that does not, is an error. detach and stop pass false
// and read neither, so a config that does not load does not block ending a
// registration (self-review L4 of org-director-attach); start passes false
// and loads the config itself, by withMainWorktreeDirectorStartConfig's
// rule.
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

// printHeadlessStillOpen writes to w, after detach or attach --replace ended
// reg, one line when reg is a headless director (`ralph org director
// start`) that is not stopped: ending its registration only records it
// (plan 2026-10-15-org-director-start, D4), so its herdr pane and workspace
// stay open, and the line names the stop that closes them, with this
// invocation's --state-dir. For a launch cmd (plan
// 2026-10-16-org-director-launch-cmd, D7) the line says that the process its
// launch script checked in with keeps running, and names the stop that ends
// it; with no check-in recorded yet, that a check-in is refused from now on,
// as the registration is not current, so the launch script starts no agent,
// and names the stop that records it stopped.
func (rt *orgDirectorRuntime) printHeadlessStillOpen(w io.Writer, reg org.DirectorRegistration) {
	if !reg.Headless() || reg.State == org.DirectorStateStopped {
		return
	}
	stop := rt.command("ralph org director stop --id " + inboxField(reg.ID))
	switch {
	case reg.CmdLaunch() && reg.PID > 0:
		_, _ = fmt.Fprintf(w, "the process of the headless director %s (pid %d) keeps running; stop it with: %s\n", inboxField(reg.ID), reg.PID, stop)
	case reg.CmdLaunch():
		_, _ = fmt.Fprintf(w, "the headless director %s has no pid yet: a check-in of its launch script is refused now, so it starts no agent; "+
			"record its stop with: %s\n", inboxField(reg.ID), stop)
	default:
		_, _ = fmt.Fprintf(w, "the herdr pane and workspace of the headless director %s stay open; close them with: %s\n", inboxField(reg.ID), stop)
	}
}

// notRestoredError is a store error err of attach, detach, start, or stop
// (startError, stopError) followed, when it wraps
// org.ErrDirectorWriteNotRestored, by the command that reads the
// registration again: director.jsonl may hold part of the failed write, so
// the store does not say that nothing was recorded. The store does not know
// the --state-dir of this invocation, so the command is said here, as
// attachedError says its two (self-review C2-L1 of org-director-attach).
// Any other err is returned as it is.
func (rt *orgDirectorRuntime) notRestoredError(err error) error {
	if !errors.Is(err, org.ErrDirectorWriteNotRestored) {
		return err
	}
	return fmt.Errorf("%w: read the registration with %s", err, rt.command("ralph org director status"))
}

// orgDirectorEnabledFile names the file [org.director].enabled was read from
// for the director verbs, by the rule of withMainWorktreeDirectorEnabled:
// the --config file, else the main worktree's ralph.toml when the ledger is
// the main worktree's, else ./ralph.toml (resolveOrgConfig). director start
// reads [org.roles] from the same file (withMainWorktreeDirectorStartConfig),
// and attach and start read [org.director].wake and wake_cmd from it
// (withMainWorktreeDirectorWakeConfig and that function; plan
// 2026-10-18-org-director-wake, D9).
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

// orgDirectorCaller is who runs a director verb, as its events record it:
// USER (empty when unset), the current directory ("" when os.Getwd fails;
// it is for an audit only), and whether HERDR_PANE_ID is set, never its
// value.
func orgDirectorCaller() org.DirectorWho {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return org.DirectorWho{User: os.Getenv("USER"), Cwd: cwd, HerdrPane: os.Getenv("HERDR_PANE_ID") != ""}
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
// director status` report them, and their --json wire shape. AutoMerge is
// [org.director].auto_merge, read from the same ralph.toml as Enabled and
// always present in the JSON, false included; it does not change State.
// State is disabled when Enabled is false, whatever the registration;
// otherwise unreadable when director.jsonl could not be read, attached when
// a registration is current, and not_attached. Registration is the current
// registration (null when there is none or the file could not be read); a
// headless one carries its launch, state, token, model, permission mode,
// workspace, and pane (org.DirectorRegistration), and one with launch cmd
// its pid, start time, and terminal name instead of the workspace and the
// pane (orgDirectorReportRegistration). OpenHeadless lists the other open
// headless registrations, detached but with their pane or process not yet
// closed, which `ralph org director stop --id` closes (omitted when none).
// Error is the read error, set whenever the read failed (with disabled too),
// and CorruptLines and IgnoredEvents the counts of org.DirectorReadResult.
// Check is what the watch's check last found for the current registration
// when it is headless and not stopped (plan 2026-10-18-org-director-liveness,
// D8: watchChecks), read from the watch's mark when the mark is that
// registration's (readOrgDirectorCheck); it is omitted when the watch has not
// checked it yet, when the mark is another registration's or cannot be read
// (markErr, which only the text form shows), and for a promoted session.
type orgDirectorReport struct {
	Enabled       bool                            `json:"enabled"`
	AutoMerge     bool                            `json:"auto_merge"`
	State         string                          `json:"state"`
	Registration  *orgDirectorReportRegistration  `json:"registration"`
	Check         *orgDirectorReportCheck         `json:"check,omitempty"`
	OpenHeadless  []orgDirectorReportRegistration `json:"open_headless,omitempty"`
	Error         string                          `json:"error,omitempty"`
	CorruptLines  int                             `json:"corrupt_lines"`
	IgnoredEvents int                             `json:"ignored_events"`
	watchChecks   bool
	markErr       string
}

// orgDirectorReportRegistration is a registration as status reports it: the
// org.DirectorRegistration the fold made, its fields inline in the JSON, and
// for a launch cmd TerminalName, the name launch_cmd gave the terminal
// (org.DirectorAgentName of the token; plan
// 2026-10-16-org-director-launch-cmd, AC8). The name follows from the token
// and is not recorded, so it is added here, not to the fold's registration,
// and a herdr or promoted registration's JSON is what it was before.
type orgDirectorReportRegistration struct {
	org.DirectorRegistration
	TerminalName string `json:"terminal_name,omitempty"`
}

// newOrgDirectorReportRegistration is reg as status reports it.
func newOrgDirectorReportRegistration(reg org.DirectorRegistration) orgDirectorReportRegistration {
	r := orgDirectorReportRegistration{DirectorRegistration: reg}
	if reg.CmdLaunch() {
		r.TerminalName = org.DirectorAgentName(reg.Token)
	}
	return r
}

// readOrgDirectorReport reads the registration of the ledger at stateDir
// (org.DirectorStore.Read, no lock) into an orgDirectorReport with
// director's Enabled as the switch and its AutoMerge, and the watch's check
// of the current registration (readOrgDirectorCheck).
func readOrgDirectorReport(stateDir string, director config.OrgDirectorConfig) orgDirectorReport {
	report := orgDirectorReport{Enabled: director.Enabled, AutoMerge: director.AutoMerge}
	res, err := org.NewDirectorStore(stateDir).Read()
	switch {
	case err != nil:
		report.Error, report.State = err.Error(), orgDirectorStateUnreadable
	case res.Current != nil:
		cur := newOrgDirectorReportRegistration(*res.Current)
		report.Registration, report.State = &cur, orgDirectorStateAttached
		report.watchChecks, report.Check, report.markErr = readOrgDirectorCheck(stateDir, *res.Current)
	default:
		report.State = orgDirectorStateNotAttached
	}
	for _, reg := range res.OpenHeadless {
		if res.Current == nil || reg.ID != res.Current.ID {
			report.OpenHeadless = append(report.OpenHeadless, newOrgDirectorReportRegistration(reg))
		}
	}
	report.CorruptLines, report.IgnoredEvents = res.CorruptLines, res.IgnoredEvents
	if !director.Enabled {
		report.State = orgDirectorStateDisabled
	}
	return report
}

// orgDirectorLine is the one director: line that `ralph status` and `ralph
// org director status` print for r. A headless registration adds its state,
// model, pane, and workspace (for a launch cmd, the launch, the terminal
// name, and the pid: orgDirectorLaunchDetails). A registration whose wake
// has push ends its details with the wake and where the watch sends
// (orgDirectorPushDetails); a pull-only one's line is that of stage 8a.
// After the details, a registration the watch checks (headless and not
// stopped) says what the watch's check found (orgDirectorWatchCheckClause),
// and every other one (a promoted session's, or a headless one recorded
// stopped while still current) that the registration does not tell whether
// its session is still running; disabled adds the check after the stop
// command. Attached and not attached
// end with what [org.director].auto_merge means (orgDirectorAutoMergeClause),
// and disabled says that auto_merge = true has no effect; unreadable says
// nothing of it. The open headless registrations that are not current are
// named at the end (orgDirectorOpenHeadlessNote). The values
// from director.jsonl, which a human can edit, have their control
// characters escaped (inboxField), so the line stays one line. The stop
// commands it names repeat hintStateDir as --state-dir
// (org.CommandWithStateDir; the caller passes orgInboxHintStateDir's value,
// the state dir only when --state-dir was given), as the other director
// commands do.
func orgDirectorLine(r orgDirectorReport, hintStateDir string) string {
	note := orgDirectorOpenHeadlessNote(r.OpenHeadless, hintStateDir)
	switch r.State {
	case orgDirectorStateAttached:
		reg := r.Registration
		details := "driver=" + inboxField(reg.Driver)
		if reg.Headless() {
			details += ", headless " + inboxField(reg.State) + ", model=" + inboxField(reg.Model) + orgDirectorLaunchDetails(reg.DirectorRegistration)
		}
		details += ", attached at " + inboxField(reg.AttachedAt)
		if reg.User != "" {
			details += ", by " + inboxField(reg.User)
		}
		if !reg.Headless() && reg.PaneID != "" {
			details += ", pane " + inboxField(reg.PaneID)
		}
		details += orgDirectorPushDetails(reg.DirectorRegistration)
		alive := "the registration does not tell whether its session is still running"
		if r.watchChecks {
			alive = orgDirectorWatchCheckClause(r, hintStateDir, true)
		}
		return "director: attached " + inboxField(reg.ID) + " (" + details + ") -- " + alive + "; " +
			"inbox items it does not ack reach the human at the ack deadline" + orgDirectorAutoMergeClause(r.AutoMerge, true) + note
	case orgDirectorStateDisabled:
		state := "[org.director].enabled = false"
		if r.AutoMerge {
			state += ", so [org.director].auto_merge = true has no effect"
		}
		switch {
		case r.Error != "":
			state += "; director.jsonl cannot be read: " + inboxField(r.Error)
		case r.Registration != nil && r.Registration.Headless():
			reg := r.Registration
			state += "; registration " + inboxField(reg.ID) + " is still recorded, headless " + inboxField(reg.State) + orgDirectorLaunchDetails(reg.DirectorRegistration) +
				": stop it with " + org.CommandWithStateDir("ralph org director stop --id "+inboxField(reg.ID), hintStateDir)
			if r.watchChecks {
				// start refuses while the switch is off, so the check names no
				// cleanup by start: the stop above is the way out.
				state += "; " + orgDirectorWatchCheckClause(r, hintStateDir, false)
			}
		case r.Registration != nil:
			state += "; registration " + inboxField(r.Registration.ID) + " is still recorded"
		}
		return "director: disabled (" + state + ") -- ralph org escalate sends every item to the human at once" + note
	case orgDirectorStateUnreadable:
		return "director: unreadable (" + inboxField(r.Error) + ")"
	}
	return "director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work" +
		orgDirectorAutoMergeClause(r.AutoMerge, false) + note
}

// orgDirectorAutoMergeClause is the part of the director: line that says,
// while the switch is on, what [org.director].auto_merge means: with false,
// merges are left to the human; with true, the director merges the PRs
// `ralph org merge-order` lists as ready with `ralph org merge` without
// asking, once one is attached when attached is false (plan
// 2026-10-17-org-auto-merge, AC2).
func orgDirectorAutoMergeClause(autoMerge, attached bool) string {
	switch {
	case !autoMerge:
		return "; merges are left to the human ([org.director].auto_merge = false)"
	case attached:
		return "; the director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)"
	}
	return "; once attached, a director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)"
}

// orgDirectorHerdrIDs is ", pane <id>" and ", workspace <id>" for the herdr
// ids recorded for a headless registration, each only when recorded.
func orgDirectorHerdrIDs(reg org.DirectorRegistration) string {
	ids := ""
	if reg.PaneID != "" {
		ids += ", pane " + inboxField(reg.PaneID)
	}
	if reg.WorkspaceID != "" {
		ids += ", workspace " + inboxField(reg.WorkspaceID)
	}
	return ids
}

// orgDirectorLaunchDetails is what the director: line and the open_headless
// lines say of where a headless registration runs: for a launch herdr its
// herdr ids (orgDirectorHerdrIDs), and for a launch cmd ", launch cmd", the
// terminal name launch_cmd gave the terminal (", name director_<token>"),
// and ", pid <pid>" once the check-in recorded one (plan
// 2026-10-16-org-director-launch-cmd, AC8).
func orgDirectorLaunchDetails(reg org.DirectorRegistration) string {
	if !reg.CmdLaunch() {
		return orgDirectorHerdrIDs(reg)
	}
	details := ", launch " + inboxField(reg.Launch) + ", name " + inboxField(org.DirectorAgentName(reg.Token))
	if reg.PID > 0 {
		details += ", pid " + strconv.Itoa(reg.PID)
	}
	return details
}

// orgDirectorOpenHeadlessNote is the end of the director: line that names
// the open headless registrations that are not current (detached, or
// replaced, with their herdr pane or launch_cmd process not stopped yet) and
// how to stop them, or "" when there are none (plan
// 2026-10-15-org-director-start, AC10). The stop command repeats
// hintStateDir as --state-dir, as orgDirectorLine's does.
func orgDirectorOpenHeadlessNote(open []orgDirectorReportRegistration, hintStateDir string) string {
	if len(open) == 0 {
		return ""
	}
	names := make([]string, 0, len(open))
	for _, reg := range open {
		names = append(names, inboxField(reg.ID)+" ("+inboxField(reg.State)+orgDirectorLaunchDetails(reg.DirectorRegistration)+")")
	}
	if len(open) == 1 {
		return "; the headless director " + names[0] + " is still open but not attached: stop it with " +
			org.CommandWithStateDir("ralph org director stop --id "+inboxField(open[0].ID), hintStateDir)
	}
	return "; the headless directors " + strings.Join(names, ", ") + " are still open but not attached: stop each with " +
		org.CommandWithStateDir("ralph org director stop --id <id>", hintStateDir)
}

// warnDirectorDamage writes to w (stderr, so --json output stays clean) the
// lines of director.jsonl at path that did not parse and the events the fold
// ignored, as warnInboxDamage does for the inbox.
func warnDirectorDamage(w io.Writer, path string, r orgDirectorReport) {
	if r.CorruptLines > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d unreadable director line(s) skipped in %s\n", r.CorruptLines, path)
	}
	if r.IgnoredEvents > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d director event(s) ignored in %s (unknown kind or id, an attached with an unknown driver or a malformed token, a reused id, "+
			"or while another registration is current, a detached of an id that is not the current one, a workspace_created or pane_created "+
			"that names no open headless registration or repeats one, a launched of an id that is not the current one starting, "+
			"or a stopped that names no open headless registration)\n", r.IgnoredEvents, path)
	}
}

// printOrgDirectorReport prints `ralph org director status`'s text form: the
// director: line (orgDirectorLine), enabled, auto_merge, and, when a
// registration is current, one line per field of it, with control
// characters escaped. The fields of a headless registration (launch to
// workspace_id) and pane_id are left out when not set, so a promoted
// session's lines are those of stage 8a-1 (with auto_merge after enabled),
// and so are pid, start_time, and terminal_name, which only a launch cmd
// has (plan 2026-10-16-org-director-launch-cmd, AC8). wake_target follows
// wake only when a promoted session was attached with --wake-target (plan
// 2026-10-18-org-director-wake, AC2): a headless registration's target is
// on its pane_id or terminal_name line. Then one open_headless line per
// open headless registration that is not current. hintStateDir is
// orgDirectorLine's.
func printOrgDirectorReport(w io.Writer, r orgDirectorReport, hintStateDir string) {
	_, _ = fmt.Fprintln(w, orgDirectorLine(r, hintStateDir))
	_, _ = fmt.Fprintf(w, "enabled: %t\n", r.Enabled)
	_, _ = fmt.Fprintf(w, "auto_merge: %t\n", r.AutoMerge)
	if reg := r.Registration; reg != nil {
		for _, f := range []struct {
			name, value string
			omitEmpty   bool
		}{
			{"id", reg.ID, false}, {"driver", reg.Driver, false}, {"wake", strings.Join(reg.Wake, ","), false},
			{"wake_target", reg.WakeTarget, true},
			{"attached_at", reg.AttachedAt, false}, {"user", reg.User, false}, {"cwd", reg.Cwd, false},
			{"herdr_pane", yesNo(reg.HerdrPane), false},
		} {
			if f.omitEmpty && f.value == "" {
				continue
			}
			_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
		}
		pid := ""
		if reg.PID > 0 {
			pid = strconv.Itoa(reg.PID)
		}
		for _, f := range []struct{ name, value string }{
			{"launch", reg.Launch}, {"state", reg.State}, {"model", reg.Model}, {"permission_mode", reg.PermissionMode},
			{"token", reg.Token}, {"workspace_id", reg.WorkspaceID}, {"pane_id", reg.PaneID},
			{"pid", pid}, {"start_time", reg.StartTime}, {"terminal_name", reg.TerminalName},
		} {
			if f.value != "" {
				_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
			}
		}
	}
	for _, reg := range r.OpenHeadless {
		_, _ = fmt.Fprintf(w, "open_headless: %s (%s%s)\n", inboxField(reg.ID), inboxField(reg.State), orgDirectorLaunchDetails(reg.DirectorRegistration))
	}
}
