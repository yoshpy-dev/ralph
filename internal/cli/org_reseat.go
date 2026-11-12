package cli

import (
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// `ralph org reseat` (spec docs/specs/2026-11-12-org-reseat.md, plan
// docs/plans/active/2026-11-12-org-reseat.md, S3): the CLI of
// (*org.Org).Reseat, which checks, stops and starts the seats of a feature
// org and records the reseat. This file builds the runtime as `ralph org
// start --plan` does (runOrgStartPlan), wires the gates (or, for --dry-run,
// what reads in their place), and prints the result.

// orgReseatHelp is the Long of `ralph org reseat`.
const orgReseatHelp = "ralph org reseat stops the seats of an org and starts them again in the\n" +
	"current environment, without a disband. The org keeps its plan approvals,\n" +
	"its reservation, and its binding to the feature. Only an org started with\n" +
	"`ralph org start --plan` is accepted; any other org is refused, with the\n" +
	"`ralph org stop` and `ralph org spawn` commands that do the same seat by\n" +
	"seat.\n" +
	"\n" +
	"With no --seat, reseat stops and starts every active seat of the org.\n" +
	"--seat names one seat, and can be repeated: a seat spawned since the\n" +
	"org's last disband, whether it is active, stopped, or its start failed,\n" +
	"so the command a failed reseat prints can be run as it is. With no\n" +
	"active seat and no --seat, reseat is refused and names the stopped seats.\n" +
	"Each seat starts with the role, driver, and model the ledger recorded for\n" +
	"it; a model of \"\" stays the CLI's own default. --driver and --model go\n" +
	"together and only with exactly one --seat: they start that seat on another\n" +
	"driver or model.\n" +
	"\n" +
	"reseat first checks everything a later stop or start would refuse: the\n" +
	"split plan and its approval, the feature worktree and its\n" +
	"ralph-worktree.sh record (reseat makes no worktree), the limits, the\n" +
	"reservation, a regate on the org, and the plan approval an implementer\n" +
	"needs. When a check fails, reseat stops nothing, records nothing, and\n" +
	"exits 1. Then it stops the seats other than the leader, then the leader.\n" +
	"It starts the leader first, as `ralph org start --plan` does from\n" +
	"<state dir>/splits/<split>.md, and then the other seats in the order the\n" +
	"org first spawned them. A stop that fails ends the reseat before any\n" +
	"start, and a start that fails ends it there. The output then names the\n" +
	"seats stopped, started, and left, and the reseat command that starts the\n" +
	"rest, and reseat exits 1. When reseat started only seats other than the\n" +
	"leader, the output ends with the `ralph org send` command of a DECISION\n" +
	"that asks the leader to send those seats their TASKs again: they start\n" +
	"with no TASK. reseat never sends it; check the leader's pane before you\n" +
	"run it.\n" +
	"\n" +
	"Two reseats of one ledger never run at once: each holds\n" +
	"<state dir>/reseat.lock from its checks until it has written its record,\n" +
	"and a second one is refused at once. Right before it stops a seat,\n" +
	"reseat reads the ledger again. When another command stopped or started\n" +
	"that seat meanwhile, reseat leaves it as it is and ends there. Every stop\n" +
	"and start a reseat makes records reseat=<id>, and the reseat ends with one\n" +
	"reseated event of the org, also when it ended early.\n" +
	"\n" +
	"--dry-run runs the checks and prints the seats reseat would stop and\n" +
	"start, in that order, with each seat's driver and model. It stops,\n" +
	"starts, and writes nothing. It only reads whether the cross-org watch and\n" +
	"the herdr server run, and a note says when a real reseat would start\n" +
	"them.\n" +
	"\n" +
	"reseat reads ralph.toml as `ralph org start --plan` does. A leader it\n" +
	"starts gets the current values of the four [reviewer] keys in its task,\n" +
	"and an [org.approval].mode other than the one the org was started with\n" +
	"refuses the reseat.\n" +
	"\n" +
	"Like `ralph org start`, a real reseat launches a long-running\n" +
	"`ralph org watch --all` when none watches the ledger, and makes sure the\n" +
	"herdr server runs, before it stops any seat (see ralph org start --help).\n" +
	orgHerdrServerHelp + "\n" +
	"\n" +
	"The panes of the seats take the herdr server's environment, not this\n" +
	"command's. To move seats to another account (another CLAUDE_CONFIG_DIR,\n" +
	"say), stop the herdr server, which ends the panes of every org and of a\n" +
	"headless director in herdr. Then run ralph org reseat from a shell with\n" +
	"the new environment: its herdr server check starts the server with that\n" +
	"environment.\n" +
	"\n" +
	"Run reseat from outside the org's seats, in the director's session or a\n" +
	"shell of your own. It is refused in the herdr pane (HERDR_PANE_ID) of an\n" +
	"active seat of the org, which it would close.\n" +
	"\n" +
	orgScaffoldVersionNoticeHelp

// newOrgReseatCmd wires `ralph org reseat`. --org-id is required
// (requireOrgID); --seat, --driver and --model go to org.ReseatParams as
// they are, and (*org.Org).PlanReseat refuses a bad combination or value.
func newOrgReseatCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seats             []string
		driverName, model string
		timeoutMS         int
		dryRun            bool
	)

	cmd := &cobra.Command{
		Use:   "reseat",
		Short: "Stop a feature org's seats and start them again in the current environment, without a disband",
		Long:  orgReseatHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			return runOrgReseat(cmd, orgReseatParams{
				orgID: *orgID, stateDir: *stateDir, configPath: *configPath,
				seats: seats, driverName: driverName, model: model, timeoutMS: timeoutMS, dryRun: dryRun,
			})
		},
	}

	cmd.Flags().StringArrayVar(&seats, "seat", nil, "seat id to stop and start again; repeat it for more seats "+
		"(default: every active seat of the org)")
	cmd.Flags().StringVar(&driverName, "driver", "", "driver CLI (claude|codex) the one --seat starts on instead of the one "+
		"the ledger recorded; needs --model and exactly one --seat")
	cmd.Flags().StringVar(&model, "model", "", "model the one --seat starts with instead of the one the ledger recorded; "+
		"needs --driver and exactly one --seat")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "run the checks and print the seats to stop and start, in order, "+
		"with each seat's driver and model; stops, starts, and writes nothing, and starts neither the watch nor the herdr server")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, orgSpawnTimeoutFlagUsage("workspace create or pane split, labels, agent start, agmsg join")+
		"; each seat's start gets its own")

	return cmd
}

// orgReseatParams is what runOrgReseat takes from the flags of `ralph org
// reseat`.
type orgReseatParams struct {
	orgID, stateDir, configPath string
	seats                       []string
	driverName, model           string
	timeoutMS                   int
	dryRun                      bool
}

// runOrgReseat is `ralph org reseat`. It reads the ledger and the config as
// runOrgStartPlan does: the state dir with the legacy-ledger guard
// (resolveOrgLedger), the config once (resolveOrgStartConfig, whose
// [reviewer] goes to the leader's start), the model_pool and scaffold notes,
// the runtime from that config (newOrgRuntimeFromConfig) with
// InboxHintStateDir, so the commands the result prints repeat --state-dir
// when it was given, and the main worktree root (org.FeatureRepoRoot). A
// real reseat gets start's watch gate (wireOrgStartWatchGate, which launches
// the watch when none runs) and the herdr server gate
// (wireOrgHerdrServerGate, which starts the server); PlanReseat runs both
// before it stops anything. A dry run gets what only reads
// (wireOrgReseatDryRunGates). The output is printOrgReseatDryRun's or
// printOrgReseatResult's.
func runOrgReseat(cmd *cobra.Command, p orgReseatParams) error {
	resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, p.stateDir, orgLedgerMutating)
	if err != nil {
		return err
	}
	orgCfg, reviewer, err := resolveOrgStartConfig(p.configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return err
	}
	printLegacyModelPoolNotice(orgCfg, cmd.ErrOrStderr())
	printScaffoldVersionNotice(cmd.ErrOrStderr(), "org reseat", orgScaffoldRoot(resolvedStateDir, stateDirSource))
	rt := newOrgRuntimeFromConfig(resolvedStateDir, orgCfg)
	rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)
	params := org.ReseatParams{
		OrgID: p.orgID, Seats: p.seats, Driver: p.driverName, Model: p.model,
		StateDir: resolvedStateDir, Reviewer: reviewer, TimeoutMS: p.timeoutMS, DryRun: p.dryRun,
	}
	if p.dryRun {
		wireOrgReseatDryRunGates(cmd, rt, resolvedStateDir, &params)
	} else {
		wireOrgStartWatchGate(cmd, rt, resolvedStateDir, stateDirSource, p.configPath, "reseat", "seat")
		wireOrgHerdrServerGate(cmd, rt, resolvedStateDir, "reseat")
	}
	if params.RepoRoot, err = org.FeatureRepoRoot(resolvedStateDir, stateDirSource); err != nil {
		return err
	}

	res := rt.Reseat(params)
	if p.dryRun {
		if res.Err != nil {
			return res.Err
		}
		printOrgReseatDryRun(cmd.OutOrStdout(), res.Plan)
		return nil
	}
	return printOrgReseatResult(cmd, res, rt.InboxHintStateDir)
}

// wireOrgReseatDryRunGates sets what a dry run reads in place of the two
// gates, neither of which it may run: they may start the cross-org watch or
// `herdr server` (plan P8, AC-19). PlanReseat with DryRun never calls the
// Org's gates; it reads the watch (ProbeWatch on the ledger, as p.WatchProbe
// is left nil) when the Org has a watch gate, and p.HerdrServerProbe when
// the Org has a herdr server gate. So the gates set here are ones that only
// read: wireOrgWatchGate's check of the watch, which launches none, and a
// herdr server gate on the same probe, which starts nothing.
func wireOrgReseatDryRunGates(cmd *cobra.Command, rt *org.Org, resolvedStateDir string, p *org.ReseatParams) {
	wireOrgWatchGate(cmd, rt, resolvedStateDir)
	probe := orgReseatHerdrServerProbe(cmd.ErrOrStderr())
	p.HerdrServerProbe = probe
	rt.HerdrServerGate = func() error {
		running, err := probe()
		if err == nil && !running {
			err = errors.New("org: reseat --dry-run: the herdr server is not running, and a dry run does not start it")
		}
		return err
	}
}

// orgReseatHerdrServerProbe is the herdr server probe of a reseat's dry run:
// one read of `herdr status server --json` (readOrgHerdrServerStatusWithin,
// bounded by orgHerdrStatusTimeout), which starts nothing. It reads the
// status as the gate (ensureOrgHerdrServer) does: a herdr that has no
// `status server --json` counts as running, with the gate's warning line on
// stderr, and a status that cannot be read for another reason is the gate's
// refusal, in the gate's words.
func orgReseatHerdrServerProbe(stderr io.Writer) func() (bool, error) {
	g := orgHerdrServerGate{verb: "reseat", stderr: stderr}
	return func() (bool, error) {
		st, err := readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout)
		switch {
		case err == nil:
			return st.Running, nil
		case errors.Is(err, driver.ErrHerdrStatusJSONUnsupported):
			_, _ = fmt.Fprintf(stderr, "warning: this herdr has no 'herdr status server --json' (herdr 0.7.5 has it), "+
				"so ralph org %s cannot check or start the herdr server and goes on as before\n", g.verb)
			return true, nil
		case errors.Is(err, exec.ErrNotFound):
			return false, g.notOnPATH(err)
		}
		return false, fmt.Errorf("org: %s: could not read the herdr server's status (%w), so nothing was made; %s", g.verb, err, g.next())
	}
}

// The notes a dry run ends with when its plan says a real reseat would start
// the watch or the server (ReseatPlan.StartsWatch, StartsHerdrServer).
const (
	orgReseatStartsWatchNote  = "note: no cross-org watch runs; a real reseat starts it"
	orgReseatStartsServerNote = "note: the herdr server is not running; a real reseat starts it"
)

// printOrgReseatDryRun prints a dry run's plan on w: a `stop:` line per seat
// it would stop, in stop order, with the role, driver and model the seat
// runs with now (orgReseatRecordedSeat); a `start:` line per seat it would
// start, in start order, with those it would start with
// (orgReseatStartSeat); then the notes.
func printOrgReseatDryRun(w io.Writer, plan org.ReseatPlan) {
	for _, s := range plan.Stop {
		_, _ = fmt.Fprintf(w, "stop: %s\n", orgReseatRecordedSeat(s))
	}
	for _, s := range plan.Start {
		_, _ = fmt.Fprintf(w, "start: %s\n", orgReseatStartSeat(s, ""))
	}
	if plan.StartsWatch {
		_, _ = fmt.Fprintln(w, orgReseatStartsWatchNote)
	}
	if plan.StartsHerdrServer {
		_, _ = fmt.Fprintln(w, orgReseatStartsServerNote)
	}
}

// printOrgReseatResult prints a real reseat's result and returns its error.
// A reseat refused before anything was stopped (res.Seats nil) prints
// nothing on stdout. Otherwise stdout gets, in the order things happened:
// a `stopped:` or `stop failed:` line per seat whose stop was tried, in stop
// order; a `started:` or `start failed:` line per seat whose start was tried,
// in start order; a line per target seat the reseat left before its start
// (orgReseatLeftLine); on success, the `reseated` line with the reseat's id;
// the reseat that starts the seats left (res.Retry), when there are any; and
// the DECISION for the leader (res.Decision), when only seats other than the
// leader started, after a line that says reseat sent nothing and names the
// read of the leader's pane. The codex model-mismatch warning of each stop
// and start goes to stderr, as stop and spawn print it. The commands repeat
// --state-dir hintStateDir when it is set (org.Org.InboxHintStateDir).
func printOrgReseatResult(cmd *cobra.Command, res org.ReseatResult, hintStateDir string) error {
	if res.Seats == nil {
		return res.Err
	}
	out := cmd.OutOrStdout()
	plan := res.Plan
	for _, seat := range plan.Stop {
		r, ok := orgReseatSeatResult(res, seat.SeatID)
		if !ok || r.Stop == nil {
			continue
		}
		printCodexModelMismatchWarning(cmd, seat.SeatID, r.Stop.ModelReceipt)
		verb := "stopped"
		if r.Outcome == org.ReseatSeatStopFailed {
			verb = "stop failed"
		}
		_, _ = fmt.Fprintf(out, "%s: %s\n", verb, orgReseatRecordedSeat(seat))
	}
	for i, seat := range plan.Start {
		r := res.Seats[i]
		if r.Spawn == nil {
			continue
		}
		printCodexModelMismatchWarning(cmd, seat.SeatID, r.Spawn.ModelReceipt)
		if r.Outcome == org.ReseatSeatStarted {
			_, _ = fmt.Fprintf(out, "started: %s\n", orgReseatStartSeat(seat, r.Spawn.Seat.PaneID))
			continue
		}
		_, _ = fmt.Fprintf(out, "start failed: %s\n", orgReseatStartSeat(seat, ""))
	}
	for i, seat := range plan.Start {
		if line := orgReseatLeftLine(seat, res.Seats[i].Outcome, res.Changed); line != "" {
			_, _ = fmt.Fprintln(out, line)
		}
	}
	if res.Err == nil {
		_, _ = fmt.Fprintf(out, "reseated org_id %s (reseat=%s)\n", plan.OrgID, plan.ID)
	}
	if res.Retry != "" {
		_, _ = fmt.Fprintf(out, "to start the seats left, run:\n%s\n", res.Retry)
	}
	if res.Decision != "" {
		_, _ = fmt.Fprintf(out, "reseat sends the leader nothing, and the seats it started have no TASK: check the leader's pane "+
			"with %s, then tell the leader with:\n%s\n",
			orgReadCommandHint(plan.OrgID, org.LeaderIdentity, hintStateDir, hintStateDir != ""), res.Decision)
	}
	return res.Err
}

// orgReseatSeatResult returns the result of seatID in res.
func orgReseatSeatResult(res org.ReseatResult, seatID string) (org.ReseatSeatResult, bool) {
	for _, r := range res.Seats {
		if r.SeatID == seatID {
			return r, true
		}
	}
	return org.ReseatSeatResult{}, false
}

// orgReseatLeftLine is the line of a target seat the reseat left before its
// start, by its outcome: one it stopped, one it did not stop (changed is the
// seat whose record changed meanwhile, ReseatResult.Changed), and one that
// was not active. A seat whose stop or start was tried has its line already,
// and gets "".
func orgReseatLeftLine(seat org.ReseatSeat, outcome org.ReseatSeatOutcome, changed string) string {
	switch {
	case outcome == org.ReseatSeatStopped:
		return "left stopped: " + seat.SeatID + " (stopped by this reseat, not started again)"
	case outcome == org.ReseatSeatNotStopped && seat.SeatID == changed:
		return "left as it is: " + seat.SeatID + " (another command stopped or started it meanwhile)"
	case outcome == org.ReseatSeatNotStopped:
		return "left running: " + seat.SeatID + " (not stopped)"
	case outcome == org.ReseatSeatNotStarted:
		return "not started: " + seat.SeatID
	}
	return ""
}

// orgReseatRecordedSeat is `<seat> (<role>, <driver> <model>)` with the
// driver and model the ledger recorded for the seat, the ones it runs with
// before the reseat (orgReseatModel).
func orgReseatRecordedSeat(s org.ReseatSeat) string {
	return fmt.Sprintf("%s (%s, %s %s)", s.SeatID, s.Role, s.RecordedDriver, orgReseatModel(s.RecordedModel))
}

// orgReseatStartSeat is `<seat> (<role>, <driver> <model>)` with the driver
// and model the seat starts with, followed, inside the parentheses, by
// `; the ledger had <driver> <model>` when --driver and --model start it on
// another driver or model, and by `, pane <pane>` when pane is not "".
func orgReseatStartSeat(s org.ReseatSeat, pane string) string {
	line := fmt.Sprintf("%s (%s, %s %s", s.SeatID, s.Role, s.Driver, orgReseatModel(s.Model))
	if s.Overridden && (s.Driver != s.RecordedDriver || s.Model != s.RecordedModel) {
		line += fmt.Sprintf("; the ledger had %s %s", s.RecordedDriver, orgReseatModel(s.RecordedModel))
	}
	if pane != "" {
		line += ", pane " + pane
	}
	return line + ")"
}

// orgReseatModel is model as a reseat's lines print it: "CLI default" for
// "", which runs the driver CLI's own default model.
func orgReseatModel(model string) string {
	if model == "" {
		return "CLI default"
	}
	return model
}
