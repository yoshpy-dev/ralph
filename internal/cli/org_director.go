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
	"github.com/yoshpy-dev/ralph/internal/org/driver"
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
			"`ralph org director stop --id <id>` closes it again. One director is\n" +
			"registered at a time for every org in the ledger, so these verbs take\n" +
			"no --org-id. attach records the session and prints the registration's\n" +
			"id (d1, d2, ...); it is refused while another director is attached,\n" +
			"unless --replace is given. detach ends the registration and needs that\n" +
			"id, so a session that was replaced is refused and need not detach;\n" +
			"--force ends the current registration whatever its id, for a human\n" +
			"cleaning up. status shows the registration, the headless directors that\n" +
			"are still open, and [org.director].enabled.\n" +
			"\n" +
			"The registration does not change what `ralph org escalate` does:\n" +
			"[org.director].enabled and the cross-org watch decide that (see\n" +
			"ralph org escalate --help). Nor does it mean the session is still\n" +
			"running: an item the director does not ack reaches the human at the\n" +
			"ack deadline of `ralph org watch --all`.",
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
			"session's detach is then refused, and it need not detach. A headless\n" +
			"director that `ralph org director start` launched counts too, starting\n" +
			"or running: the refusal then names `ralph org director stop --id`,\n" +
			"which closes its herdr pane, and --replace leaves that pane open.\n" +
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
			if err := checkOrgDirectorDriver("director attach", driver); err != nil {
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
			who := orgDirectorCaller()
			reg, replaced, err := rt.store.Attach(org.DirectorAttach{
				Driver: driver, Replace: replace, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane,
			})
			if err != nil {
				if errors.Is(err, org.ErrDirectorAttached) {
					return rt.attachedError(err, driver)
				}
				return rt.notRestoredError(err)
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
				return rt.notRestoredError(err)
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

func newOrgDirectorStartCmd(stateDir, configPath *string) *cobra.Command {
	var (
		driverName, model string
		timeoutMS         int
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a headless director in a herdr workspace of its own and register it",
		Long: "ralph org director start launches a headless director and registers it\n" +
			"as the director of the ledger: --driver (claude or codex) runs with the\n" +
			"role prompt internal/org/prompts/director.md in a herdr workspace of its\n" +
			"own, labelled ralph_director_<token>, in one tab labelled\n" +
			"director_<token>, where <token> is 12 hex digits new for every start. It\n" +
			"prints the registration's id, the pane, and the stop command:\n" +
			"started d1 (driver=codex, model=..., permission=..., pane=...,\n" +
			"workspace=...). Open herdr and talk to the director in that pane.\n" +
			"\n" +
			"It writes <state-dir>/director.jsonl and the prompt file\n" +
			"<state-dir>/prompts/director_<token>.md, and nothing to the manifest:\n" +
			"the director is not an org seat, so max_orgs and max_total_seats do not\n" +
			"count it and `ralph org stop --all` does not stop it. The attached event\n" +
			"(with launch herdr, the token, the model, and the permission mode) is\n" +
			"written before any herdr call, so the registration is decided first;\n" +
			"workspace_created and pane_created follow as herdr hands out the ids,\n" +
			"and launched once the agent has started.\n" +
			"\n" +
			"The director runs with the permission mode of\n" +
			"[org.permissions.roles].director, else [org.permissions].default (for\n" +
			"codex, autonomous and edits need [org.permissions].codex_verified), and\n" +
			"with --model, which [org.roles].director must allow; without --model,\n" +
			"with the first model_pool entry for --driver that it allows, and a\n" +
			"warning. Without --config, those settings and [org.director].enabled\n" +
			"are read from the main worktree's ralph.toml when the ledger is the main\n" +
			"worktree's, so a feature worktree's ralph.toml does not choose them. The\n" +
			"director runs in the ledger's main worktree root, else in the directory\n" +
			"start runs in.\n" +
			"\n" +
			"start is refused, and writes nothing and runs no herdr, when\n" +
			"[org.director].enabled is false, while a director is attached (one\n" +
			"still starting included: stop a headless one with\n" +
			"`ralph org director stop --id <id>`, detach a promoted session), and\n" +
			"inside a herdr pane (HERDR_PANE_ID is set): run start from a terminal\n" +
			"outside herdr. start has no --replace. Like `ralph org start`, it makes\n" +
			"sure a long-running `ralph org watch --all` watches the ledger first: it\n" +
			"launches one when none runs, and is refused when a watch holds the lock\n" +
			"but is stale, stopping, or a --once run. When a step fails after the\n" +
			"registration, start closes the pane and workspace it made, records the\n" +
			"director stopped, and exits 1; what it could not close is printed with\n" +
			"the herdr commands and the stop command that finish the cleanup.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOrgDirectorDriver("director start", driverName); err != nil {
				return err
			}
			rt, err := newOrgDirectorRuntime(cmd, "director start", *stateDir, *configPath, orgLedgerMutating, false)
			if err != nil {
				return err
			}
			return rt.startDirector(cmd, *configPath, driverName, model, timeoutMS)
		},
	}

	cmd.Flags().StringVar(&driverName, "driver", "", "the director's agent: claude or codex (required)")
	cmd.Flags().StringVar(&model, "model", "", "model name or alias (default: first [org].model_pool entry permitted for role director on --driver, with a warning)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "how long herdr agent start may take, in milliseconds")

	return cmd
}

func newOrgDirectorStopCmd(stateDir, configPath *string) *cobra.Command {
	var (
		id    string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Close the herdr pane and workspace of a headless director and record it stopped",
		Long: "ralph org director stop closes the herdr pane and workspace of a\n" +
			"headless director that `ralph org director start` launched, and records\n" +
			"it stopped: a stopped event in <state-dir>/director.jsonl and, when it is\n" +
			"the current registration, a detached in the same write, so no director\n" +
			"is attached afterwards. Nothing is written to the manifest. Give --id,\n" +
			"the id start printed. stop works on every headless registration that is\n" +
			"still open: running, still starting (it closes what was recorded), or\n" +
			"detached (a headless director that ran detach, or was replaced, keeps\n" +
			"its pane until stop).\n" +
			"\n" +
			"Before it sends C-c to the pane and closes the pane and the workspace,\n" +
			"stop checks that herdr has the pane in a tab labelled director_<token>\n" +
			"inside a workspace labelled ralph_director_<token>, the names start gave\n" +
			"them; a pane or workspace that fails the check (a label differs, or\n" +
			"herdr cannot be asked) is not sent C-c and is not closed, because its\n" +
			"recorded id may now name another launch's. A pane or workspace herdr no\n" +
			"longer knows counts as closed, so a director whose pane a human closed\n" +
			"in herdr is cleaned up with stop too.\n" +
			"\n" +
			"stop is refused, and writes nothing, for an id with no open headless\n" +
			"registration (a promoted session's registration is ended with\n" +
			"`ralph org director detach --id`), one already stopped, and from inside\n" +
			"the director's pane or workspace: run it from a terminal outside the\n" +
			"director's workspace. When the labels do not match or cannot be\n" +
			"checked, or a close fails, stop exits 1 with the reason and the herdr\n" +
			"commands that close it by hand. --force, a human's cleanup, then\n" +
			"records the director stopped anyway, closing nothing more, and prints\n" +
			"what was left open and how to find it (herdr workspace list, the\n" +
			"workspace labelled ralph_director_<token>); when the labels match,\n" +
			"--force closes as usual. stop reads no ralph.toml (nor the main\n" +
			"worktree's), so one that does not load does not block it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(id) == "" {
				return errors.New("org: director stop: give --id <id>, the id start printed (ralph org director status lists the open headless directors)")
			}
			rt, err := newOrgDirectorRuntime(cmd, "director stop", *stateDir, *configPath, orgLedgerMutating, false)
			if err != nil {
				return err
			}
			return rt.stopDirector(cmd, id, force)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "the id of the headless director to stop (d1, d2, ...), as start printed it (required)")
	cmd.Flags().BoolVar(&force, "force", false, "record the stop even when the pane or workspace cannot be confirmed or closed, closing nothing more (a human's cleanup)")

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
			"For a headless director (ralph org director start) the fields add its\n" +
			"state (starting or running), model, permission mode, token, workspace,\n" +
			"and pane, and the headless directors that are no longer attached but\n" +
			"still open are listed (open_headless), to be closed with\n" +
			"`ralph org director stop --id <id>`.\n" +
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
// (orgInboxHintStateDir), and the [org] config with Director.Enabled read by
// withMainWorktreeDirectorEnabled (the zero value for detach and stop, which
// read none, and for start, which loads its own: startDirector).
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
// not, is an error. detach and stop pass false and read neither, so a config
// that does not load does not block ending a registration (self-review L4 of
// org-director-attach); start passes false and loads the config itself, by
// withMainWorktreeDirectorStartConfig's rule.
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
// said once, here (self-review L7 of org-director-attach). The current
// registration is read again here, without the lock, so it can be newer
// than the one the store refused on. A headless one (`ralph org director
// start`, starting or running) is ended with `ralph org director stop --id`,
// which closes its herdr pane, not with detach (plan
// 2026-10-15-org-director-start, AC2).
func (rt *orgDirectorRuntime) attachedError(err error, driverName string) error {
	replace := rt.command("ralph org director attach --driver " + driverName + " --replace")
	var cur *org.DirectorRegistration
	if res, readErr := rt.store.Read(); readErr == nil {
		cur = res.Current
	}
	if cur != nil && cur.Headless() {
		return fmt.Errorf("%w: stop it with %s from a terminal outside herdr, which closes its herdr pane, "+
			"or replace it with %s, which leaves that pane open until such a stop", err, rt.command("ralph org director stop --id "+inboxField(cur.ID)), replace)
	}
	detach := "ralph org director detach --id <its id>"
	if cur != nil {
		detach = "ralph org director detach --id " + inboxField(cur.ID)
	}
	return fmt.Errorf("%w: end it with %s from the session that attached it (or with --force in place of --id, when that session is gone), "+
		"or replace it with %s", err, rt.command(detach), replace)
}

// notRestoredError is a store error err of attach or detach followed, when
// it wraps org.ErrDirectorWriteNotRestored, by the command that reads the
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

// startDirector is `ralph org director start` on rt (plan
// 2026-10-15-org-director-start, D5 and D6): it loads the config by
// withMainWorktreeDirectorStartConfig's rule, refuses a switched-off director
// and a model [org.roles].director does not allow (resolveModelOrWarn's
// default is one it allows), wires start's watch gate, and runs
// (*org.Org).StartDirector in the ledger's main worktree root, else in the
// current directory. A refusal or failure is returned (exit 1) in the words
// of startError; on success it prints the registration, the pane to talk to
// the director in, and the stop command.
func (rt *orgDirectorRuntime) startDirector(cmd *cobra.Command, configPath, driverName, model string, timeoutMS int) error {
	cfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return fmt.Errorf("org: load config: %w", err)
	}
	if cfg, err = withMainWorktreeDirectorStartConfig(cfg, configPath, rt.stateDir, rt.stateDirSource); err != nil {
		return err
	}
	if !cfg.Director.Enabled {
		return fmt.Errorf("org: director start: [org.director].enabled is false in %s, so ralph org escalate already sends every item to the human at once "+
			"and leaves nothing to a director, and no director is started; to use one, set [org.director].enabled = true there",
			orgDirectorEnabledFile(configPath, rt.stateDir, rt.stateDirSource))
	}
	resolvedModel, err := resolveModelOrWarn(cfg, driverName, org.DirectorRole, model, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if !org.ModelAllowedForRole(cfg, org.DirectorRole, resolvedModel) {
		return fmt.Errorf("org: director start: model %q is not permitted for role %s by [org.roles].%s in %s (it allows %s)",
			resolvedModel, org.DirectorRole, org.DirectorRole, orgDirectorEnabledFile(configPath, rt.stateDir, rt.stateDirSource),
			strings.Join(cfg.Roles[org.DirectorRole], ", "))
	}
	cwd, ok := org.LedgerMainWorktreeRoot(rt.stateDir, rt.stateDirSource)
	if !ok {
		if cwd, err = os.Getwd(); err != nil {
			return fmt.Errorf("org: director start: read the current directory: %w", err)
		}
	}
	o := newOrgDirectorHerdrRuntime(cfg)
	wireOrgStartWatchGate(cmd, o, rt.stateDir, rt.stateDirSource, configPath, "director start", "director")
	res, err := o.StartDirector(org.DirectorStartParams{
		Store: rt.store, Driver: driverName, Model: resolvedModel, Cwd: cwd, TimeoutMS: timeoutMS, Who: orgDirectorCaller(),
	})
	if err != nil {
		return rt.startError(err)
	}
	reg := res.Registration
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "started %s (driver=%s, model=%s, permission=%s, pane=%s, workspace=%s)\n", inboxField(reg.ID), inboxField(reg.Driver),
		inboxField(reg.Model), inboxField(reg.PermissionMode), inboxField(reg.PaneID), inboxField(reg.WorkspaceID))
	_, _ = fmt.Fprintf(out, "talk to the director in herdr: pane %s, in the tab %s of the workspace labelled %s\n",
		inboxField(reg.PaneID), res.TabLabel, res.WorkspaceLabel)
	_, _ = fmt.Fprintf(out, "stop it with: %s\n", rt.command("ralph org director stop --id "+inboxField(reg.ID)))
	return nil
}

// startError is a StartDirector error in the words of the CLI. A refusal
// while a registration is current says how to end it (startAttachedError).
// A cleanup that did not finish (*org.DirectorCleanupError) already says the
// failure, what is still open, and the herdr commands; its stop command gets
// this invocation's --state-dir (withStateDirInStopCommand). A write that
// may hold part of itself names the status command (notRestoredError). Every
// other error (HERDR_PANE_ID set, a replace during start with what start
// closed, the permission mode, the watch gate) already says what happened
// and is returned as it is.
func (rt *orgDirectorRuntime) startError(err error) error {
	var attached *org.DirectorAttachedError
	if errors.As(err, &attached) {
		return rt.startAttachedError(attached)
	}
	return rt.notRestoredError(rt.withStateDirInStopCommand(err))
}

// startAttachedError is start's refusal e while a registration is current,
// followed by the way to end it: `ralph org director stop --id` from outside
// herdr for a headless one, else attach's detach commands for a promoted
// session. start has no --replace (plan 2026-10-15-org-director-start,
// Non-goals). e.Current was read again after the refusal; when it is nil
// (none is current by then, or the read failed), status reads it.
func (rt *orgDirectorRuntime) startAttachedError(e *org.DirectorAttachedError) error {
	cur := e.Current
	switch {
	case cur == nil:
		return fmt.Errorf("%w: read the current registration with %s, and run start again once none is attached (start has no --replace)",
			e, rt.command("ralph org director status"))
	case cur.Headless():
		return fmt.Errorf("%w: stop it with %s from a terminal outside herdr, then run start again (start has no --replace)",
			e, rt.command("ralph org director stop --id "+inboxField(cur.ID)))
	}
	return fmt.Errorf("%w: end it with %s from the session that attached it (or with --force in place of --id, when that session is gone), "+
		"then run start again (start has no --replace)", e, rt.command("ralph org director detach --id "+inboxField(cur.ID)))
}

// withStateDirInStopCommand adds this invocation's --state-dir to the
// StopCommand of the *org.DirectorCleanupError err is or wraps, so the
// error's message names the stop that records the cleanup on this ledger
// (the org layer does not know the flag). It returns err.
func (rt *orgDirectorRuntime) withStateDirInStopCommand(err error) error {
	var cleanup *org.DirectorCleanupError
	if errors.As(err, &cleanup) && cleanup.StopCommand != "" {
		cleanup.StopCommand = rt.command(cleanup.StopCommand)
	}
	return err
}

// stopDirector is `ralph org director stop --id <id> [--force]` on rt (plan
// 2026-10-15-org-director-start, D4 and D11): (*org.Org).StopDirector with
// only the herdr driver and no config, as detach reads none, then the
// result in the CLI's words (printOrgDirectorStop), or the refusal
// (stopError).
func (rt *orgDirectorRuntime) stopDirector(cmd *cobra.Command, id string, force bool) error {
	o := newOrgDirectorHerdrRuntime(config.OrgConfig{})
	res, err := o.StopDirector(org.DirectorStopParams{Store: rt.store, ID: id, Force: force, Who: orgDirectorCaller()})
	if err != nil {
		return rt.stopError(cmd.ErrOrStderr(), id, force, res, err)
	}
	printOrgDirectorStop(cmd, res)
	return nil
}

// stopError is a StopDirector error in the words of the CLI. For a pane or
// workspace stop could not confirm or close (*org.DirectorCleanupError) it
// first writes what was done to them on w (stderr), and its stop command
// gets this invocation's --state-dir; without --force, a close that failed
// also names stop --force, which records the stop with what is still open
// (an unconfirmed label's stop command carries --force already). A promoted
// session's registration says how to detach it (notHeadlessError). An id
// with no open headless registration names status, which lists them. A
// write that may hold part of itself names status (notRestoredError).
func (rt *orgDirectorRuntime) stopError(w io.Writer, id string, force bool, res org.DirectorStopResult, err error) error {
	var cleanup *org.DirectorCleanupError
	switch {
	case errors.As(err, &cleanup):
		_, _ = fmt.Fprintf(w, "stop %s: %s\n", inboxField(cleanup.ID), orgDirectorStopNotes(res))
		err = rt.withStateDirInStopCommand(err)
		if !force && !cleanup.Unconfirmed && len(cleanup.HerdrCommands) > 0 {
			err = fmt.Errorf("%w; or record the stop and leave them open with %s", err, rt.command("ralph org director stop --id "+inboxField(cleanup.ID)+" --force"))
		}
	case errors.Is(err, org.ErrDirectorNotHeadless):
		return rt.notHeadlessError(id, err)
	case errors.Is(err, org.ErrDirectorNoOpenHeadless) && !errors.Is(err, org.ErrDirectorStopped):
		err = fmt.Errorf("%w; %s lists the open headless directors", err, rt.command("ralph org director status"))
	}
	return rt.notRestoredError(err)
}

// notHeadlessError is stop's refusal err of registration id, which a session
// promoted with the /director skill attached: it has no herdr pane of
// ralph's to close. While it is current it is ended with detach (read again
// here, without the lock); otherwise it has already ended.
func (rt *orgDirectorRuntime) notHeadlessError(id string, err error) error {
	if res, readErr := rt.store.Read(); readErr == nil && res.Current != nil && res.Current.ID == id {
		return fmt.Errorf("%w: end it with %s from the session that attached it (or with --force in place of --id, when that session is gone)",
			err, rt.command("ralph org director detach --id "+inboxField(id)))
	}
	return fmt.Errorf("%w: it is not the current registration, so it has already ended and there is nothing to stop", err)
}

// printOrgDirectorStop prints what StopDirector did: on stdout the stopped
// line with the notes on the C-c, the pane, and the workspace (forced when
// --force recorded it with something left open), whether the director is
// no longer attached, and the label to look for when no workspace id was
// recorded; for a forced stop, a warning on stderr with why and the herdr
// commands that find this launch's workspace by its label.
func printOrgDirectorStop(cmd *cobra.Command, res org.DirectorStopResult) {
	id := inboxField(res.Registration.ID)
	out := cmd.OutOrStdout()
	forced := ""
	if res.Forced {
		forced = "forced; "
	}
	_, _ = fmt.Fprintf(out, "stopped %s (%s%s)\n", id, forced, orgDirectorStopNotes(res))
	if res.WasCurrent {
		_, _ = fmt.Fprintln(out, "the director is no longer attached")
	}
	if label := res.UnrecordedWorkspaceLabel; label != "" {
		_, _ = fmt.Fprintf(out, "no workspace id was recorded for %s: herdr may still have a workspace labelled %s that start made before it could record it; "+
			"look for it with herdr workspace list\n", id, label)
	}
	if res.Forced {
		why := "-"
		if res.CloseErr != nil {
			why = inboxField(res.CloseErr.Error())
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: director stop --force: %s is recorded stopped, but its herdr pane or workspace may still be open (%s); "+
			"find and close them by hand: %s\n", id, why, strings.Join(res.HerdrCommands, ", then "))
	}
}

// orgDirectorStopNotes is what StopDirector did to the pane and the
// workspace: its three notes, escaped (a close failure's note holds herdr's
// error text).
func orgDirectorStopNotes(res org.DirectorStopResult) string {
	return inboxField(res.CtrlCNote) + ", " + inboxField(res.PaneNote) + ", " + inboxField(res.WorkspaceNote)
}

// newOrgDirectorHerdrRuntime is the *org.Org of director start and stop:
// cfg and the real herdr driver (driver.ExecRunner, herdr on PATH), with
// none of the manifest, receipts, agmsg, or inbox newOrgRuntimeAt wires,
// since StartDirector and StopDirector use none of them (the director is not
// a seat, plan 2026-10-15-org-director-start, D2). Getenv is nil, so they
// read HERDR_PANE_ID and HERDR_WORKSPACE_ID from the environment.
func newOrgDirectorHerdrRuntime(cfg config.OrgConfig) *org.Org {
	return &org.Org{Config: cfg, Herdr: driver.Herdr{R: driver.ExecRunner{}}}
}

// orgDirectorEnabledFile names the file [org.director].enabled was read from
// for the director verbs, by the rule of withMainWorktreeDirectorEnabled:
// the --config file, else the main worktree's ralph.toml when the ledger is
// the main worktree's, else ./ralph.toml (resolveOrgConfig). director start
// reads [org.roles] from the same file (withMainWorktreeDirectorStartConfig).
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
// director status` report them, and their --json wire shape. State is
// disabled when Enabled is false, whatever the registration; otherwise
// unreadable when director.jsonl could not be read, attached when a
// registration is current, and not_attached. Registration is the current
// registration (null when there is none or the file could not be read); a
// headless one carries its launch, state, token, model, permission mode,
// workspace, and pane (org.DirectorRegistration). OpenHeadless lists the
// other open headless registrations, detached but with their pane not yet
// closed, which `ralph org director stop --id` closes (omitted when none).
// Error is the read error, set whenever the read failed (with disabled too),
// and CorruptLines and IgnoredEvents the counts of org.DirectorReadResult.
type orgDirectorReport struct {
	Enabled       bool                       `json:"enabled"`
	State         string                     `json:"state"`
	Registration  *org.DirectorRegistration  `json:"registration"`
	OpenHeadless  []org.DirectorRegistration `json:"open_headless,omitempty"`
	Error         string                     `json:"error,omitempty"`
	CorruptLines  int                        `json:"corrupt_lines"`
	IgnoredEvents int                        `json:"ignored_events"`
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
	for _, reg := range res.OpenHeadless {
		if res.Current == nil || reg.ID != res.Current.ID {
			report.OpenHeadless = append(report.OpenHeadless, reg)
		}
	}
	report.CorruptLines, report.IgnoredEvents = res.CorruptLines, res.IgnoredEvents
	if !enabled {
		report.State = orgDirectorStateDisabled
	}
	return report
}

// orgDirectorLine is the one director: line that `ralph status` and `ralph
// org director status` print for r. A headless registration adds its state,
// model, pane, and workspace, and the open headless registrations that are
// not current are named at the end (orgDirectorOpenHeadlessNote). The values
// from director.jsonl, which a human can edit, have their control
// characters escaped (inboxField), so the line stays one line.
func orgDirectorLine(r orgDirectorReport) string {
	note := orgDirectorOpenHeadlessNote(r.OpenHeadless)
	switch r.State {
	case orgDirectorStateAttached:
		reg := r.Registration
		details := "driver=" + inboxField(reg.Driver)
		if reg.Headless() {
			details += ", headless " + inboxField(reg.State) + ", model=" + inboxField(reg.Model) + orgDirectorHerdrIDs(*reg)
		}
		details += ", attached at " + inboxField(reg.AttachedAt)
		if reg.User != "" {
			details += ", by " + inboxField(reg.User)
		}
		if !reg.Headless() && reg.PaneID != "" {
			details += ", pane " + inboxField(reg.PaneID)
		}
		return "director: attached " + inboxField(reg.ID) + " (" + details + ") -- the registration does not tell whether its session is still running; " +
			"inbox items it does not ack reach the human at the ack deadline" + note
	case orgDirectorStateDisabled:
		state := "[org.director].enabled = false"
		switch {
		case r.Error != "":
			state += "; director.jsonl cannot be read: " + inboxField(r.Error)
		case r.Registration != nil && r.Registration.Headless():
			reg := r.Registration
			state += "; registration " + inboxField(reg.ID) + " is still recorded, headless " + inboxField(reg.State) + orgDirectorHerdrIDs(*reg) +
				": stop it with ralph org director stop --id " + inboxField(reg.ID)
		case r.Registration != nil:
			state += "; registration " + inboxField(r.Registration.ID) + " is still recorded"
		}
		return "director: disabled (" + state + ") -- ralph org escalate sends every item to the human at once" + note
	case orgDirectorStateUnreadable:
		return "director: unreadable (" + inboxField(r.Error) + ")"
	}
	return "director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work" + note
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

// orgDirectorOpenHeadlessNote is the end of the director: line that names
// the open headless registrations that are not current (detached, or
// replaced, with their herdr pane not closed yet) and how to close them, or
// "" when there are none (plan 2026-10-15-org-director-start, AC10).
func orgDirectorOpenHeadlessNote(open []org.DirectorRegistration) string {
	if len(open) == 0 {
		return ""
	}
	names := make([]string, 0, len(open))
	for _, reg := range open {
		names = append(names, inboxField(reg.ID)+" ("+inboxField(reg.State)+orgDirectorHerdrIDs(reg)+")")
	}
	if len(open) == 1 {
		return "; the headless director " + names[0] + " is still open but not attached: stop it with ralph org director stop --id " + inboxField(open[0].ID)
	}
	return "; the headless directors " + strings.Join(names, ", ") + " are still open but not attached: stop each with ralph org director stop --id <id>"
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
// director: line (orgDirectorLine), enabled, and, when a registration is
// current, one line per field of it, with control characters escaped. The
// fields of a headless registration (launch to workspace_id) and pane_id are
// left out when not set, so a promoted session's lines are those of stage
// 8a-1. Then one open_headless line per open headless registration that is
// not current.
func printOrgDirectorReport(w io.Writer, r orgDirectorReport) {
	_, _ = fmt.Fprintln(w, orgDirectorLine(r))
	_, _ = fmt.Fprintf(w, "enabled: %t\n", r.Enabled)
	if reg := r.Registration; reg != nil {
		for _, f := range []struct{ name, value string }{
			{"id", reg.ID}, {"driver", reg.Driver}, {"wake", strings.Join(reg.Wake, ",")},
			{"attached_at", reg.AttachedAt}, {"user", reg.User}, {"cwd", reg.Cwd},
			{"herdr_pane", yesNo(reg.HerdrPane)},
		} {
			_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
		}
		for _, f := range []struct{ name, value string }{
			{"launch", reg.Launch}, {"state", reg.State}, {"model", reg.Model}, {"permission_mode", reg.PermissionMode},
			{"token", reg.Token}, {"workspace_id", reg.WorkspaceID}, {"pane_id", reg.PaneID},
		} {
			if f.value != "" {
				_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
			}
		}
	}
	for _, reg := range r.OpenHeadless {
		_, _ = fmt.Fprintf(w, "open_headless: %s (%s%s)\n", inboxField(reg.ID), inboxField(reg.State), orgDirectorHerdrIDs(reg))
	}
}
