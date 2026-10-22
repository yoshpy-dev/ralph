package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The headless director of `ralph org director start` and `stop` (plan
// docs/plans/active/2026-10-15-org-director-start.md): the two verbs, the
// orgDirectorRuntime methods only they use, and the *org.Org they run on.
// The hidden verb `ralph org director launched`, the check-in of the launch
// script of a start with [org.director].launch_cmd (plan
// 2026-10-16-org-director-launch-cmd, D4), is here too. newOrgDirectorCmd,
// which wires them, and the runtime and helpers that the director verbs
// share are in org_director.go.

func newOrgDirectorStartCmd(stateDir, configPath *string) *cobra.Command {
	var (
		driverName, model string
		timeoutMS         int
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a headless director in a herdr workspace of its own, or with [org.director].launch_cmd, and register it",
		Long: "ralph org director start launches a headless director and registers it\n" +
			"as the director of the ledger: --driver (claude or codex) runs with the\n" +
			"role prompt internal/org/prompts/director.md in a herdr workspace of its\n" +
			"own, labelled ralph_director_<token>, in one tab labelled\n" +
			"director_<token>, where <token> is 12 hex digits new for every start. It\n" +
			"prints the registration's id, the pane, and the stop command:\n" +
			"started d1 (driver=codex, model=..., permission=..., pane=...,\n" +
			"workspace=...). Open herdr and talk to the director in that pane.\n" +
			"\n" +
			"With [org.director].launch_cmd set, start runs it in place of herdr.\n" +
			"start writes the launch script <state-dir>/launch/director_<token>.sh\n" +
			"(mode 0700), which goes to the directory the director runs in, exports\n" +
			"RALPH_DIRECTOR_TOKEN (and RALPH_ORG_AGMSG_HOME when it is set where\n" +
			"start runs), checks in, and then execs the agent that start found on\n" +
			"its PATH with the arguments herdr would give it. start runs launch_cmd's\n" +
			"argv, with {script}, {script_sh}, {name} (director_<token>) and {cwd}\n" +
			"replaced and no shell between, for up to 10 s, and then waits for the\n" +
			"check-in, which records the script's pid and that pid's start time, for\n" +
			"up to --timeout-ms. It prints started d1 (driver=..., model=...,\n" +
			"permission=..., launch=cmd, name=director_<token>, pid=...). Talk to\n" +
			"the director in the terminal launch_cmd opened; ralph never closes that\n" +
			"terminal, so it is left for you to close after stop.\n" +
			"\n" +
			"It writes <state-dir>/director.jsonl and the prompt file\n" +
			"<state-dir>/prompts/director_<token>.md, and nothing to the manifest:\n" +
			"the director is not an org seat, so max_orgs and max_total_seats do not\n" +
			"count it and `ralph org stop --all` does not stop it. The attached event\n" +
			"(with launch herdr, the token, the model, and the permission mode) is\n" +
			"written before any herdr call, so the registration is decided first;\n" +
			"workspace_created and pane_created follow as herdr hands out the ids,\n" +
			"and launched once the agent has started. With launch_cmd, attached\n" +
			"(with launch cmd) is written before launch_cmd runs, and launched (with\n" +
			"the pid and its start time) when the check-in comes.\n" +
			"\n" +
			"The director runs with the permission mode of\n" +
			"[org.permissions.roles].director, else [org.permissions].default, and\n" +
			"with --model, which [org.roles].director must allow; without --model,\n" +
			"with the first model_pool entry for --driver that it allows, and a\n" +
			"warning. Without --config, those settings and [org.director].enabled\n" +
			"are read from the main worktree's ralph.toml when the ledger is the main\n" +
			"worktree's, so a feature worktree's ralph.toml does not choose them.\n" +
			"[org.director].launch_cmd is read the same way. So are\n" +
			"[org.director].wake, which start records as the director's wake (see\n" +
			"ralph org director attach --help), and wake_cmd. The director runs in the\n" +
			"ledger's main worktree root, else in the directory start runs in. With\n" +
			"--config, the ralph org commands the director's prompt has it run carry\n" +
			"--config with that file's absolute path, so the director reads the\n" +
			"settings start read (the prompt's stop command, for the human, carries\n" +
			"none: stop reads no ralph.toml). With herdr, RALPH_ORG_AGMSG_HOME\n" +
			"set where start runs does not reach the director, whose herdr pane has\n" +
			"the herdr server's environment: start warns when it is set, and\n" +
			"[org].agmsg_home in ralph.toml is what reaches the director. With\n" +
			"launch_cmd it does reach the director, as the launch script exports it\n" +
			"(a relative value made absolute from where start runs, as for the\n" +
			"watch start launches), and start does not warn.\n" +
			"\n" +
			"A codex director runs only in guarded: start refuses autonomous and\n" +
			"edits whatever [org.permissions].codex_verified says, because they run\n" +
			"codex under --sandbox workspace-write, which keeps .git read-only, so\n" +
			"the director could not start a feature org (ralph org start --plan runs\n" +
			"git worktree add). Set [org.permissions.roles].director = \"guarded\" and\n" +
			"let your codex config (~/.codex/config.toml) write .git and reach the\n" +
			"network, or use --driver claude; a guarded codex director follows that\n" +
			"config, so under sandbox_mode = \"workspace-write\" it meets the same\n" +
			"wall.\n" +
			"\n" +
			"start is refused, and writes nothing and runs no herdr, when\n" +
			"[org.director].enabled is false, for a codex director in autonomous or\n" +
			"edits, when the --config file's [org].agmsg_home is a relative path\n" +
			"(neither absolute nor starting with ~/: the director runs in the\n" +
			"ledger's main worktree root when the ledger has one and the watch\n" +
			"resolves it from where start runs, so the two may resolve it from\n" +
			"different directories, and start refuses it wherever it runs; give an\n" +
			"absolute path), while a promoted session is attached (detach it; for a\n" +
			"headless director, see below), and inside a herdr pane (HERDR_PANE_ID\n" +
			"is set): run start from a terminal outside herdr. start has no\n" +
			"--replace. With launch_cmd, HERDR_PANE_ID does not refuse start, which\n" +
			"runs no herdr then, and start is also refused when the --driver agent is\n" +
			"not on its PATH. It is refused too when [org.director].wake has push and\n" +
			"wake_cmd is empty: the terminal launch_cmd opens is one ralph cannot\n" +
			"type into, so set wake_cmd (its {target} is director_<token>) or use\n" +
			"wake = [\"pull\"]. In herdr, push needs no wake_cmd: the watch types the\n" +
			"wake line into the director's pane. Like `ralph org start`, it makes\n" +
			"sure a long-running `ralph org watch --all` watches the ledger first:\n" +
			"it launches one when none runs, and is refused when a watch holds the\n" +
			"lock but is stale, stopping, or a --once run. When a step fails after\n" +
			"the registration, start closes the pane and workspace it made, records\n" +
			"the director stopped, and exits 1; what it could not close is printed\n" +
			"with the herdr commands and the stop command that finish the cleanup.\n" +
			"With launch_cmd, when launch_cmd fails or runs out its 10 s, no check-in\n" +
			"comes within --timeout-ms, or the registration is replaced while start\n" +
			"waits, start stops what checked in by its pid and start time, as stop\n" +
			"does, and records the director stopped, exits 1, and says that the\n" +
			"terminal launch_cmd may have opened is left for you to close; a launch\n" +
			"script that checks in from it afterwards is refused and starts no agent.\n" +
			"When start cannot confirm that the process ended, it records the\n" +
			"director detached only and prints the stop command that finishes the\n" +
			"cleanup.\n" +
			"\n" +
			"While a headless director that start launched is the attached one,\n" +
			"starting or running, start checks it on the spot, before the watch, as\n" +
			"`ralph org watch --all` checks it every cycle: one still starting by the\n" +
			"start process that recorded it, one running in herdr with herdr agent\n" +
			"get and its pane, and one launched with launch_cmd by its pid and that\n" +
			"pid's start time. When the check finds it alive, or cannot tell, start\n" +
			"is refused, writes nothing, and says what the check found: stop it with\n" +
			"`ralph org director stop --id <id>` from a terminal outside herdr (for\n" +
			"launch_cmd, from a terminal other than the director's). When the check\n" +
			"finds it ended, start cleans it up as that stop would, but only what the\n" +
			"registration recorded when the check read it: it closes the pane and\n" +
			"workspace that herdr's labels confirm as that launch's, or, for\n" +
			"launch_cmd, sends nothing to a process that is gone, and records the\n" +
			"director stopped. It then prints one line, cleaned up d1, which had\n" +
			"ended (...), and goes on, so the line is printed even when start is\n" +
			"refused afterwards. When the cleanup cannot finish (a label that is not\n" +
			"that launch's, a close that fails, or a registration that changed after\n" +
			"the check read it), start records nothing, starts no director, and\n" +
			"exits 1 with the stop command that finishes the cleanup.",
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
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "how long herdr agent start may take, or with [org.director].launch_cmd how long start waits for the launch script's check-in, in milliseconds")

	return cmd
}

func newOrgDirectorStopCmd(stateDir, configPath *string) *cobra.Command {
	var (
		id    string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a headless director (close its herdr pane and workspace, or end its launch_cmd process) and record it stopped",
		Long: "ralph org director stop closes the herdr pane and workspace of a\n" +
			"headless director that `ralph org director start` launched, or ends the\n" +
			"process of one it launched with [org.director].launch_cmd, and records\n" +
			"it stopped: a stopped event in <state-dir>/director.jsonl and, when it is\n" +
			"the current registration, a detached in the same write, so no director\n" +
			"is attached afterwards. Nothing is written to the manifest. Give --id,\n" +
			"the id start printed. stop works on every headless registration that is\n" +
			"still open: running, still starting (it closes what was recorded), or\n" +
			"detached (a headless director that ran detach, or was replaced, keeps\n" +
			"its pane, or its process, until stop).\n" +
			"\n" +
			"For a director in herdr, before it sends C-c to the pane and closes the\n" +
			"pane and the workspace, stop checks that herdr has the pane in a tab\n" +
			"labelled director_<token> inside a workspace labelled\n" +
			"ralph_director_<token>, the names start gave them; a pane or workspace\n" +
			"that fails the check (a label differs, or herdr cannot be asked) is not\n" +
			"sent C-c and is not closed, because its recorded id may now name another\n" +
			"launch's. A pane or workspace herdr no longer knows counts as closed, so\n" +
			"a director whose pane a human closed in herdr is cleaned up with stop\n" +
			"too.\n" +
			"\n" +
			"For a director launched with launch_cmd (launch cmd), stop runs no\n" +
			"herdr. It reads with ps the pid the launch script checked in with, and\n" +
			"sends SIGTERM to that pid alone, only when its start time is the one the\n" +
			"check-in recorded, so the launch script before its exec and the agent\n" +
			"after it are stopped alike; it then waits up to 10 s for the process to\n" +
			"end before it records the stop. A pid that is gone, is a zombie, or has\n" +
			"another start time (another process has it now) counts as ended and is\n" +
			"sent nothing, and so is a registration still starting with no check-in\n" +
			"yet, whose check-in is refused afterwards. stop never sends SIGKILL and\n" +
			"does not close the terminal launch_cmd opened (name director_<token>):\n" +
			"close it yourself. With or without --force, stop is refused, and writes\n" +
			"nothing, when RALPH_DIRECTOR_TOKEN is the registration's token: it then\n" +
			"runs inside the director, whose process would end before stop records\n" +
			"it, so run it from a terminal other than the director's. When the\n" +
			"process does not end within the 10 s, stop records nothing and exits 1\n" +
			"with the pid to look at (ps -o pid,lstart,command -p <pid>): run it\n" +
			"again once that process has ended. When ps cannot be read (before the\n" +
			"SIGTERM, stop sends nothing; while it waits after the SIGTERM, the\n" +
			"SIGTERM has already been sent), stop records nothing and exits 1 with\n" +
			"the pid too: run it again once ps can read it. Either way, --force\n" +
			"records the stop instead: it takes the same steps and, when the end is\n" +
			"still not confirmed, records the stop anyway, sends nothing more, and\n" +
			"leaves the process as it is. When the launch script checks in after\n" +
			"stop read the registration, stop has sent that process nothing, so it\n" +
			"records nothing and exits 1: run it again to stop the process that\n" +
			"checked in.\n" +
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
			"--force closes as usual. When a start that is still running records a\n" +
			"workspace or pane after stop read the registration, stop has not closed\n" +
			"it, so it records nothing and exits 1 with what it closed: run it again\n" +
			"to close what start recorded. When start's cleanup (after a step of\n" +
			"start failed) or another stop records the director stopped while stop\n" +
			"closes it, stop records nothing more and exits 0 with what it closed and\n" +
			"that the director was already recorded stopped. stop reads no ralph.toml\n" +
			"(nor the main worktree's), so one that does not load does not block it.",
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
	cmd.Flags().BoolVar(&force, "force", false, "record the stop even when the pane or workspace cannot be confirmed or closed, closing nothing more, "+
		"or with launch_cmd when the process's end cannot be confirmed, sending nothing more (a human's cleanup)")

	return cmd
}

// orgDirectorAgentLookPath and orgDirectorRalphExecutable are the
// (*org.Org).DirectorAgentLookPath and DirectorRalphExecutable of director
// start with [org.director].launch_cmd: nil means exec.LookPath and
// os.Executable. They are variables so the tests can give a fake agent and
// a check-in that runs the test binary; TestMain points both at a failure,
// since PATH may hold a real claude or codex and os.Executable is the test
// binary.
var (
	orgDirectorAgentLookPath   func(file string) (string, error)
	orgDirectorRalphExecutable func() (string, error)
)

// orgDirectorStopWait is the (*org.Org).DirectorStopWait of director start
// and stop: how long they wait, after the SIGTERM to the process of a launch
// cmd, for it to end. 0 means the org layer's 10 s; it is a variable so the
// tests can wait less for a process that ignores its SIGTERM.
var orgDirectorStopWait time.Duration

// startDirector is `ralph org director start` on rt (plan
// 2026-10-15-org-director-start, D5 and D6): it loads the config by
// withMainWorktreeDirectorStartConfig's rule, refuses a switched-off director,
// a relative [org].agmsg_home of a --config (checkStartAgmsgHome, which also
// warns when RALPH_ORG_AGMSG_HOME is set and start uses herdr), and a model
// [org.roles].director does not allow (resolveModelOrWarn's default is one
// it allows), wires start's watch gate, and runs (*org.Org).StartDirector in
// the ledger's main worktree root, else in the current directory, with
// configPath, which StartDirector makes absolute and gives as --config to the
// ralph org commands the director's prompt has it run (not to the prompt's
// stop command, which the human runs). With [org.director].launch_cmd set
// (plan 2026-10-16-org-director-launch-cmd), it also gives StartDirector
// RALPH_ORG_AGMSG_HOME as set here, which the launch script exports after
// it cds to the directory the director runs in, so a relative value is made
// absolute from this directory first (agmsgHomeIsRelative), as the watch
// start launches gets it (orgWatchChildAgmsgHome). A refusal or failure is
// returned (exit 1) in the words of startError; on success it prints the
// registration, where to talk to the director (the herdr pane, or the
// terminal launch_cmd opened), and the stop command, which, like the
// prompt's, has no --config (stop reads no ralph.toml). When start cleaned up
// a current headless registration its check found ended (plan
// 2026-10-18-org-director-liveness, D6), the line that says so comes first on
// stdout (DirectorStartParams.Notice), also when start fails afterwards.
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
	cmdLaunch := len(cfg.Director.LaunchCmd) > 0
	if err := rt.checkStartAgmsgHome(cmd, configPath, cfg.AgmsgHome, cmdLaunch); err != nil {
		return err
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
	o.DirectorAgentLookPath, o.DirectorRalphExecutable = orgDirectorAgentLookPath, orgDirectorRalphExecutable
	wireOrgStartWatchGate(cmd, o, rt.stateDir, rt.stateDirSource, configPath, "director start", "director")
	agmsgHome := ""
	if cmdLaunch {
		agmsgHome = os.Getenv(orgAgmsgHomeEnv)
		if agmsgHomeIsRelative(agmsgHome) {
			abs, err := filepath.Abs(agmsgHome)
			if err != nil {
				return fmt.Errorf("org: director start: make the agmsg home %s of %s absolute: %w", agmsgHome, orgAgmsgHomeEnv, err)
			}
			agmsgHome = abs
		}
	}
	out := cmd.OutOrStdout()
	res, err := o.StartDirector(org.DirectorStartParams{
		Store: rt.store, Driver: driverName, Model: resolvedModel, Cwd: cwd, ConfigPath: configPath, TimeoutMS: timeoutMS,
		AgmsgHome: agmsgHome, Who: orgDirectorCaller(), Notice: out,
	})
	if err != nil {
		return rt.startError(err)
	}
	reg := res.Registration
	if reg.CmdLaunch() {
		_, _ = fmt.Fprintf(out, "started %s (driver=%s, model=%s, permission=%s, launch=%s, name=%s, pid=%d)\n", inboxField(reg.ID), inboxField(reg.Driver),
			inboxField(reg.Model), inboxField(reg.PermissionMode), org.DirectorLaunchCmd, res.TerminalName, reg.PID)
		_, _ = fmt.Fprintf(out, "talk to the director in the terminal that [org.director].launch_cmd opened (name %s)\n", res.TerminalName)
	} else {
		_, _ = fmt.Fprintf(out, "started %s (driver=%s, model=%s, permission=%s, pane=%s, workspace=%s)\n", inboxField(reg.ID), inboxField(reg.Driver),
			inboxField(reg.Model), inboxField(reg.PermissionMode), inboxField(reg.PaneID), inboxField(reg.WorkspaceID))
		_, _ = fmt.Fprintf(out, "talk to the director in herdr: pane %s, in the tab %s of the workspace labelled %s\n",
			inboxField(reg.PaneID), res.TabLabel, res.WorkspaceLabel)
	}
	_, _ = fmt.Fprintf(out, "stop it with: %s\n", rt.command("ralph org director stop --id "+inboxField(reg.ID)))
	return nil
}

// checkStartAgmsgHome is start's check of the agmsg home the director uses
// (plan 2026-10-15-org-director-start, D13), before the --model fallback and
// StartDirector, so a refusal writes nothing, runs no herdr, and launches no
// watch. The director runs in the ledger's main worktree root (when the
// ledger has one) with the herdr server's environment, while the watch that
// start launches gets the agmsg home resolved from this directory
// (orgWatchChildAgmsgHome), so the two may resolve a relative
// [org].agmsg_home from different directories and use different agmsg
// installs. A relative value of a --config that was given
// (agmsgHomeIsRelative, the rule orgWatchChildAgmsgHome uses) is therefore
// refused wherever start runs: the rule does not compare the two
// directories, so it also refuses in the main worktree root and on a ledger
// with no main worktree, where they are the same (plan D13 chose the simple
// rule). Without --config both read the main worktree root's ralph.toml and
// resolve it from that root, so nothing is refused. The refusal holds with
// [org.director].launch_cmd too (cmdLaunch; plan
// 2026-10-16-org-director-launch-cmd, D10: one rule for both). When
// RALPH_ORG_AGMSG_HOME is set here and start uses herdr, it writes one
// warning line on stderr and goes on: the director's pane does not get the
// variable. With launch_cmd it warns about nothing, since the launch script
// exports the variable as set here (D3), made absolute by startDirector.
func (rt *orgDirectorRuntime) checkStartAgmsgHome(cmd *cobra.Command, configPath, cfgAgmsgHome string, cmdLaunch bool) error {
	if cmd.Flags().Changed("config") && configPath != "" && agmsgHomeIsRelative(cfgAgmsgHome) {
		return fmt.Errorf("org: director start: [org].agmsg_home %q in %s is a relative path: the director and the watch may resolve it from different directories "+
			"and so use different agmsg installs (the director runs in the ledger's main worktree root when the ledger has one, while the watch resolves it "+
			"from this directory), so start refuses it wherever it runs; give an absolute path (or one that starts with ~/)",
			cfgAgmsgHome, orgDirectorEnabledFile(configPath, rt.stateDir, rt.stateDirSource))
	}
	if !cmdLaunch && os.Getenv(orgAgmsgHomeEnv) != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: director start: %s is set here, but the director's herdr pane has the herdr server's environment, "+
			"so the director does not get it from here; [org].agmsg_home in the ralph.toml the director reads (the --config file, when given) reaches it\n", orgAgmsgHomeEnv)
	}
	return nil
}

// startError is a StartDirector error in the words of the CLI. A refusal
// while a registration is current says how to end it (startAttachedError).
// A cleanup that did not finish (*org.DirectorCleanupError) already says the
// failure, what is still open, and the herdr commands; its stop command gets
// this invocation's --state-dir (withStateDirInStopCommand). A write that
// may hold part of itself names the status command (notRestoredError). Every
// other error (HERDR_PANE_ID set, a replace during start with what start
// closed, the permission mode, a codex director in autonomous or edits, the
// watch gate) already says what happened
// and is returned as it is.
//
// With [org.director].launch_cmd (plan 2026-10-16-org-director-launch-cmd,
// D6) ralph cannot close the terminal launch_cmd may have opened, so a
// cleanup that recorded the stop (*org.DirectorLaunchCmdStoppedError) also
// says that the terminal is left for the human to close and that its launch
// script, whose check-in is now refused, ends without the agent; and a
// cleanup that did not finish (a *org.DirectorCleanupError of a launch cmd,
// whose text names no herdr pane or command) also says to close the
// terminal once the stop it names is recorded.
//
// start's cleanup of a current headless registration its check found ended
// (plan 2026-10-18-org-director-liveness, D6) fails as such a
// *org.DirectorCleanupError too, its stop command with this invocation's
// --state-dir. When it found that the registration changed after the check
// read it (org.ErrDirectorChangedSinceCheck), it also names the status
// command that reads the registration as it is now.
func (rt *orgDirectorRuntime) startError(err error) error {
	var attached *org.DirectorAttachedError
	if errors.As(err, &attached) {
		return rt.startAttachedError(attached)
	}
	var stopped *org.DirectorLaunchCmdStoppedError
	if errors.As(err, &stopped) {
		return fmt.Errorf("%w; the terminal that [org.director].launch_cmd may have opened (name %s) is left open for you to close, "+
			"and a launch script that checks in from it now is refused and ends without starting the agent", err, stopped.TerminalName)
	}
	err = rt.withStateDirInStopCommand(err)
	var cleanup *org.DirectorCleanupError
	if errors.As(err, &cleanup) && cleanup.Launch == org.DirectorLaunchCmd {
		err = fmt.Errorf("%w; the terminal that [org.director].launch_cmd may have opened (name %s) is left open: close it once the stop is recorded",
			err, cleanup.TerminalName)
	}
	if errors.Is(err, org.ErrDirectorChangedSinceCheck) {
		err = fmt.Errorf("%w; read the registration as it is now with %s", err, rt.command("ralph org director status"))
	}
	return rt.notRestoredError(err)
}

// startAttachedError is start's refusal e while a registration is current,
// followed by the way to end it: `ralph org director stop --id` for a
// headless one, run from outside herdr for a launch herdr and from a
// terminal other than the director's for a launch cmd (plan
// 2026-10-16-org-director-launch-cmd), else attach's detach commands for a
// promoted session. start has no --replace (plan
// 2026-10-15-org-director-start, Non-goals). e.Current was read again after
// the refusal; when it is nil (none is current by then, or the read failed),
// status reads it. When start checked a headless one on the spot and found it
// alive or could not tell (e.Check; plan 2026-10-18-org-director-liveness,
// D6), the refusal ends with what that check found.
func (rt *orgDirectorRuntime) startAttachedError(e *org.DirectorAttachedError) error {
	cur := e.Current
	checked := ""
	if e.Check != nil && cur != nil {
		checked = "; start checked " + inboxField(cur.ID) + " just now: " + orgDirectorCheckText(*e.Check)
	}
	switch {
	case cur == nil:
		return fmt.Errorf("%w: read the current registration with %s, and run start again once none is attached (start has no --replace)",
			e, rt.command("ralph org director status"))
	case cur.CmdLaunch():
		return fmt.Errorf("%w: stop it with %s from a terminal other than the director's, then run start again (start has no --replace)%s",
			e, rt.command("ralph org director stop --id "+inboxField(cur.ID)), checked)
	case cur.Headless():
		return fmt.Errorf("%w: stop it with %s from a terminal outside herdr, then run start again (start has no --replace)%s",
			e, rt.command("ralph org director stop --id "+inboxField(cur.ID)), checked)
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
// result in the CLI's words (printOrgDirectorStop, exit 0, also when start's
// cleanup or another stop recorded the stop first), or the refusal
// (stopError). A launch cmd's registration is stopped by its pid and start
// time with no herdr call (plan 2026-10-16-org-director-launch-cmd, D7).
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
// (an unconfirmed label's stop command carries --force already). A stop
// that found a workspace or pane a start recorded after stop read the
// registration (org.ErrDirectorChangedDuringStop, plan D14) comes as a
// *org.DirectorCleanupError too and is written the same way: it left nothing
// open of what it read, so it names no --force, and its stop command is the
// stop to run again. A promoted session's registration says how to detach
// it (notHeadlessError). An id with no open headless registration names
// status, which lists them. A write that may hold part of itself names
// status (notRestoredError).
//
// A launch cmd's stop (plan 2026-10-16-org-director-launch-cmd, D7) comes as
// a *org.DirectorCleanupError too, and w gets what was done to the process
// (its ProcessNote). When the end of the process was not confirmed
// (org.ErrDirectorProcessNotEnded: ps could not be read, or it outlived its
// SIGTERM), the error adds that the pid may still be running and, unless ps
// could not be read (orgDirectorPSLook), the ps that shows it, and without
// --force names stop --force, which records the stop and leaves the process
// as it is. A check-in that came after stop read the registration
// (org.ErrDirectorCheckedInDuringStop) left nothing it read running, so it
// names no --force.
func (rt *orgDirectorRuntime) stopError(w io.Writer, id string, force bool, res org.DirectorStopResult, err error) error {
	var cleanup *org.DirectorCleanupError
	switch {
	case errors.As(err, &cleanup):
		_, _ = fmt.Fprintf(w, "stop %s: %s\n", inboxField(cleanup.ID), orgDirectorStopNotes(res))
		err = rt.withStateDirInStopCommand(err)
		forceCommand := rt.command("ralph org director stop --id " + inboxField(cleanup.ID) + " --force")
		if errors.Is(err, org.ErrDirectorProcessNotEnded) && cleanup.PID > 0 {
			err = fmt.Errorf("%w; pid %d may still be running%s", err, cleanup.PID, orgDirectorPSLook(": ", err, cleanup.PID))
			if !force {
				err = fmt.Errorf("%w; or record the stop and leave the process as it is with %s", err, forceCommand)
			}
		}
		if !force && !cleanup.Unconfirmed && len(cleanup.HerdrCommands) > 0 {
			err = fmt.Errorf("%w; or record the stop and leave them open with %s", err, forceCommand)
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
// commands that find this launch's workspace by its label. When start's
// cleanup or another stop recorded the stop first (res.AlreadyStopped, plan
// AC8), the first line says the director was already recorded stopped, with
// the notes, and the next says who may have recorded it and that this stop
// recorded nothing more. It names no stop to run, since one would be refused
// as already stopped.
//
// For a launch cmd (plan 2026-10-16-org-director-launch-cmd, D7) the notes
// are what was done to the process, and the lines after them say that the
// terminal launch_cmd opened, named by res.TerminalName, is left for the
// human to close (with no check-in recorded, that it may have opened one,
// whose check-in is refused from now on); a forced stop's warning names the
// pid that may still be running and, unless ps could not be read, the ps
// that shows it.
func printOrgDirectorStop(cmd *cobra.Command, res org.DirectorStopResult) {
	id := inboxField(res.Registration.ID)
	out := cmd.OutOrStdout()
	forced := ""
	if res.Forced {
		forced = "forced; "
	}
	if res.AlreadyStopped {
		_, _ = fmt.Fprintf(out, "%s was already recorded stopped (%s)\n", id, orgDirectorStopNotes(res))
		_, _ = fmt.Fprintf(out, "start's cleanup or another stop recorded %s stopped after this stop read it, so this stop recorded nothing more\n", id)
	} else {
		_, _ = fmt.Fprintf(out, "stopped %s (%s%s)\n", id, forced, orgDirectorStopNotes(res))
	}
	if res.WasCurrent {
		_, _ = fmt.Fprintln(out, "the director is no longer attached")
	}
	if res.Registration.CmdLaunch() {
		printOrgDirectorCmdStop(cmd, res)
		return
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

// printOrgDirectorCmdStop is the part of printOrgDirectorStop that is a
// launch cmd's: the terminal left open and, for a forced stop, the warning
// on stderr.
func printOrgDirectorCmdStop(cmd *cobra.Command, res org.DirectorStopResult) {
	id, pid, name := inboxField(res.Registration.ID), res.Registration.PID, inboxField(res.TerminalName)
	out := cmd.OutOrStdout()
	if pid > 0 {
		_, _ = fmt.Fprintf(out, "the terminal that [org.director].launch_cmd opened (name %s) is left open for you to close\n", name)
	} else {
		_, _ = fmt.Fprintf(out, "the terminal that [org.director].launch_cmd may have opened (name %s) is left open for you to close, "+
			"and a launch script that checks in from it now is refused and ends without starting the agent\n", name)
	}
	if res.Forced {
		why := "-"
		if res.CloseErr != nil {
			why = inboxField(res.CloseErr.Error())
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: director stop --force: %s is recorded stopped, but pid %d, which its launch script checked in with, "+
			"may still be running (%s)%s\n", id, pid, why, orgDirectorPSLook("; ", res.CloseErr, pid))
	}
}

// orgDirectorPSLook is what the error or the warning of a launch cmd's stop
// whose process's end was not confirmed for why says after "pid <pid> may
// still be running": sep and the ps that shows pid, or nothing when why is
// that ps could not be read (org.ErrDirectorProcessUnreadable), since ps
// cannot show it then (the stop error names the stop to run again once ps
// can read it).
func orgDirectorPSLook(sep string, why error, pid int) string {
	if errors.Is(why, org.ErrDirectorProcessUnreadable) {
		return ""
	}
	return fmt.Sprintf("%slook with ps -o pid,lstart,command -p %d", sep, pid)
}

// orgDirectorStopNotes is what StopDirector did to the pane and the
// workspace: its three notes, escaped (a close failure's note holds herdr's
// error text); for a launch cmd, what it did to the process (ProcessNote).
func orgDirectorStopNotes(res org.DirectorStopResult) string {
	if res.Registration.CmdLaunch() {
		return inboxField(res.ProcessNote)
	}
	return inboxField(res.CtrlCNote) + ", " + inboxField(res.PaneNote) + ", " + inboxField(res.WorkspaceNote)
}

// orgDirectorCheckInParentPID and orgDirectorCheckInReadProcess are how
// `ralph org director launched` reads the pid of the process that ran it and
// that pid's start time: os.Getppid and org.ReadDirectorProcess. The check-in
// reads the start process that recorded the registration with the same
// orgDirectorCheckInReadProcess (MarkCmdLaunched's ReadStarter; plan
// 2026-10-18-org-director-liveness, D2). They are variables so the tests can
// give another parent or a ps that cannot be read.
var (
	orgDirectorCheckInParentPID   = os.Getppid
	orgDirectorCheckInReadProcess = org.ReadDirectorProcess
)

// orgDirectorCheckInPSTimeout bounds the ps that `ralph org director
// launched` runs, as defaultDriverCallTimeout bounds each driver call.
const orgDirectorCheckInPSTimeout = 10 * time.Second

func newOrgDirectorLaunchedCmd(stateDir, configPath *string) *cobra.Command {
	var (
		id, token string
		pid       int
	)

	cmd := &cobra.Command{
		Use:   "launched",
		Short: "Check in the launch script of a headless director started with launch_cmd (run by that script only)",
		Long: "ralph org director launched is the check-in of the launch script that\n" +
			"`ralph org director start` writes when [org.director].launch_cmd is set.\n" +
			"The script runs it with its own pid (--pid \"$$\") before it execs the\n" +
			"agent, and execs the agent only when it exits 0. It records launched,\n" +
			"with the pid and the pid's start time as ps reads it, in\n" +
			"<state-dir>/director.jsonl, and prints nothing.\n" +
			"\n" +
			"It is refused, writes nothing, and exits 1 with the reason on stderr,\n" +
			"so the agent does not start, when --pid is not the pid of the process\n" +
			"that ran it, when ps cannot read that pid's start time, or when the\n" +
			"registration --id is not the current one, was not started with\n" +
			"launch_cmd, is no longer starting (stopped, or already checked in), or\n" +
			"has another launch token than --token. It is refused the same way when\n" +
			"the `ralph org director start` that recorded the registration has\n" +
			"ended (ps finds no process with its pid, a zombie, or another process\n" +
			"in its place), since no start waits for the check-in then; a ps that\n" +
			"cannot read that pid does not refuse it. It reads no ralph.toml.",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" || token == "" || !cmd.Flags().Changed("pid") {
				return errors.New("org: director launched: give --id, --token, and --pid; the launch script that ralph org director start writes runs this check-in itself")
			}
			rt, err := newOrgDirectorRuntime(cmd, "director launched", *stateDir, *configPath, orgLedgerMutating, false)
			if err != nil {
				return err
			}
			return rt.directorLaunched(id, token, pid)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "the id of the registration start wrote (d1, d2, ...)")
	cmd.Flags().StringVar(&token, "token", "", "the launch token of that registration")
	cmd.Flags().IntVar(&pid, "pid", 0, "the pid of the launch script, which the agent keeps across the exec")

	return cmd
}

// directorLaunched is `ralph org director launched` on rt (plan
// 2026-10-16-org-director-launch-cmd, D4): it refuses a pid that is not this
// process's parent, the launch script that ran it, and a pid whose start
// time ps does not read (org.ReadDirectorProcess, alive), and then records
// the check-in with the store's MarkCmdLaunched, which refuses the rest
// under the lock, a start that recorded the registration and has ended
// included (read with orgDirectorCheckInReadProcess too; plan
// 2026-10-18-org-director-liveness, D2). Every refusal writes nothing and is
// returned (one line on stderr, exit 1), and the script then exits without
// the agent.
func (rt *orgDirectorRuntime) directorLaunched(id, token string, pid int) error {
	verb := "org: director launched " + inboxField(id)
	if parent := orgDirectorCheckInParentPID(); pid != parent {
		return fmt.Errorf("%s: %w: --pid %d is not the pid of the process that ran this check-in (%d); the launch script passes its own (\"$$\")",
			verb, org.ErrDirectorCheckInRefused, pid, parent)
	}
	ctx, cancel := context.WithTimeout(context.Background(), orgDirectorCheckInPSTimeout)
	defer cancel()
	proc, err := orgDirectorCheckInReadProcess(ctx, pid)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %w: the start time of pid %d cannot be read: %s", verb, org.ErrDirectorCheckInRefused, pid, inboxField(err.Error()))
	case proc.State != org.DirectorProcessAlive:
		return fmt.Errorf("%s: %w: pid %d is %s, not a running process", verb, org.ErrDirectorCheckInRefused, pid, proc.State)
	}
	return rt.notRestoredError(rt.store.MarkCmdLaunched(org.DirectorCmdCheckIn{
		ID: id, Token: token, PID: pid, StartTime: proc.StartTime, Who: orgDirectorCaller(),
		ReadStarter: orgDirectorCheckInReadProcess,
	}))
}

// newOrgDirectorHerdrRuntime is the *org.Org of director start and stop:
// cfg and the real herdr driver (driver.ExecRunner, herdr on PATH), with
// none of the manifest, receipts, agmsg, or inbox newOrgRuntimeAt wires,
// since StartDirector and StopDirector use none of them (the director is not
// a seat, plan 2026-10-15-org-director-start, D2). Getenv is nil, so they
// read HERDR_PANE_ID, HERDR_WORKSPACE_ID, and RALPH_DIRECTOR_TOKEN from the
// environment, and so is DirectorProcessReader, so a launch cmd's process
// is read with ps; DirectorStopWait is orgDirectorStopWait. The director's
// pane carries AGMSG_SELF_NAME=off, as a seat's does (newOrgRuntimeAt).
func newOrgDirectorHerdrRuntime(cfg config.OrgConfig) *org.Org {
	return &org.Org{Config: cfg, Herdr: driver.NewHerdr(driver.ExecRunner{}, driver.AgmsgSelfNameOff), DirectorStopWait: orgDirectorStopWait}
}
