package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The headless director of `ralph org director start` and `stop` (plan
// docs/plans/active/2026-10-15-org-director-start.md): the two verbs, the
// orgDirectorRuntime methods only they use, and the *org.Org they run on.
// newOrgDirectorCmd, which wires them, and the runtime and helpers that the
// director verbs share are in org_director.go.

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
			"start runs in. With --config, every ralph org command in the director's\n" +
			"prompt carries --config with that file's absolute path, so the director\n" +
			"reads the settings start read.\n" +
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

// startDirector is `ralph org director start` on rt (plan
// 2026-10-15-org-director-start, D5 and D6): it loads the config by
// withMainWorktreeDirectorStartConfig's rule, refuses a switched-off director
// and a model [org.roles].director does not allow (resolveModelOrWarn's
// default is one it allows), wires start's watch gate, and runs
// (*org.Org).StartDirector in the ledger's main worktree root, else in the
// current directory, with configPath, which StartDirector makes absolute and
// gives every ralph org command of the director's prompt as --config. A
// refusal or failure is returned (exit 1) in the words of startError; on
// success it prints the registration, the pane to talk to the director in,
// and the stop command, which needs no --config (stop reads no ralph.toml).
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
		Store: rt.store, Driver: driverName, Model: resolvedModel, Cwd: cwd, ConfigPath: configPath, TimeoutMS: timeoutMS, Who: orgDirectorCaller(),
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
