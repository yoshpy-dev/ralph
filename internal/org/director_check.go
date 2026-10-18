package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// The liveness check of a headless director (stage 8b-2a, plan
// docs/plans/active/2026-10-18-org-director-liveness.md, D3): CheckDirector
// reads whether the current registration that `ralph org director start`
// wrote is still alive, has ended, or cannot be told, and how it found that
// out. The cross-org watch checks every cycle and writes the result to
// director-alive.json (D4; the mark, at the end of this file, and
// watchAllRun.checkDirector in watch.go), and `ralph org director start`
// checks the current registration on the spot and cleans it up when it has
// ended (D6; checkCurrentDirector in director_start_cleanup.go). `ralph org director
// attach` refused by a starting registration reads only its starter row
// (CheckDirectorStarter, D7). The check writes nothing: no director.jsonl
// line, no herdr close, no key, no signal.
// It reads the registration without the lock, runs ps or herdr, and then
// reads it again under director.lock, and keeps its result only when the
// registration did not change in between (directorVersion), so a start that
// finished while another one checked cannot be read as one that ended.

// The states of a check (DirectorCheck.State).
const (
	// DirectorCheckAlive: the director, or its start, is still running.
	DirectorCheckAlive = "alive"
	// DirectorCheckEnded: it has ended: its agent or process is gone, or its
	// start ended before it recorded the launch, so nothing will record it
	// running. What it leaves behind (a launch herdr's pane and workspace,
	// where an agent started just before its start ended may still run, or
	// a launch cmd's terminal) is for a stop to close.
	DirectorCheckEnded = "ended"
	// DirectorCheckUnknown: the check could not tell (DirectorCheck.Detail
	// says why), and nothing may be done to the registration on its strength.
	DirectorCheckUnknown = "unknown"
)

// How a check found its state (DirectorCheck.How), one for each row of D3.
// Every check has one, the unknown ones included, so a printed check always
// names the row that decided it.
const (
	// A starting registration (launch herdr or cmd) is read by its starter,
	// the start process that recorded it (StarterPID and StarterStartTime).

	// DirectorCheckHowNoStarter: unknown, no starter is recorded (an attached
	// written before stage 8b-2a, or a start that could not read its own
	// start time).
	DirectorCheckHowNoStarter = "no_starter"
	// DirectorCheckHowStartEnded: ended, ps finds no process with the
	// starter's pid, or a zombie.
	DirectorCheckHowStartEnded = "start_ended"
	// DirectorCheckHowStarterPIDReused: unknown, the pid has a process with
	// another start time (a later process got the pid, or the clock moved).
	DirectorCheckHowStarterPIDReused = "starter_pid_reused"
	// DirectorCheckHowStartRunning: alive, the pid has the starter's start
	// time.
	DirectorCheckHowStartRunning = "start_running"
	// DirectorCheckHowStarterUnreadable: unknown, ps could not read the pid.
	DirectorCheckHowStarterUnreadable = "starter_unreadable"

	// A running launch herdr is read by `herdr agent get director_<token>`
	// and, when herdr has no such agent, by its recorded pane.

	// DirectorCheckHowAgentFound: alive, herdr has the agent.
	DirectorCheckHowAgentFound = "agent_found"
	// DirectorCheckHowAgentEnded: ended, herdr has no such agent, and the
	// recorded pane is still in a tab and a workspace labelled for this
	// launch (the agent ended and its pane went back to a shell).
	DirectorCheckHowAgentEnded = "agent_ended"
	// DirectorCheckHowPaneClosed: ended, herdr has no such agent and no
	// longer has the recorded pane, or no pane is recorded.
	DirectorCheckHowPaneClosed = "pane_closed"
	// DirectorCheckHowPaneUnconfirmed: unknown, herdr has no such agent, but
	// the recorded pane is not confirmed as this launch's (a label differs,
	// or a get failed or timed out).
	DirectorCheckHowPaneUnconfirmed = "pane_unconfirmed"
	// DirectorCheckHowAgentGetFailed: unknown, the agent get failed otherwise
	// (it timed out, no herdr server answered, or another error).
	DirectorCheckHowAgentGetFailed = "agent_get_failed"
	// DirectorCheckHowNoAgentGet: unknown, the herdr client cannot run
	// `herdr agent get` (none is configured, or it has no AgentGet).
	DirectorCheckHowNoAgentGet = "no_agent_get"

	// A running launch cmd is read by the pid its launch script checked in
	// with (PID and StartTime).

	// DirectorCheckHowProcessEnded: ended, ps finds no process with the pid,
	// or a zombie.
	DirectorCheckHowProcessEnded = "process_ended"
	// DirectorCheckHowPIDReused: unknown, the pid has a process with another
	// start time. directorProcessEnded, which a stop uses, reads that as
	// ended, since a stop must never signal it; the check does not, since
	// the clock may have moved under a director that is still running.
	DirectorCheckHowPIDReused = "pid_reused"
	// DirectorCheckHowProcessFound: alive, the pid has the recorded start
	// time.
	DirectorCheckHowProcessFound = "process_found"
	// DirectorCheckHowProcessUnreadable: unknown, ps could not read the pid.
	DirectorCheckHowProcessUnreadable = "process_unreadable"

	// DirectorCheckHowChanged: unknown, the registration changed while it was
	// checked, on both tries (CheckDirector).
	DirectorCheckHowChanged = "changed"
)

// directorCheckAttempts is how many times CheckDirector checks before it
// gives up on a registration that keeps changing (DirectorCheckHowChanged).
const directorCheckAttempts = 2

// DirectorCheck is the result of one check of a headless registration: its
// ID, State (DirectorCheck*), How (DirectorCheckHow*), CheckedAt (the time the
// caller gave, UTC RFC3339), and, for unknown, Detail, a short reason
// escaped for a terminal (PrintableInboxText); Detail is empty for alive and
// ended. It is the result in memory: CheckDirector and CheckDirectorStarter
// return it, and DirectorAlive.Check gives it back from the mark. Nothing
// writes it as JSON. The mark, director-alive.json, is DirectorAlive (six
// keys, attached_at among them), and the director.check of status's --json
// (D8) is the CLI's orgDirectorReportCheck (four keys, no id). Its JSON tags
// are kept for TestDirectorCheck_JSON only, so changing them changes neither
// file nor output.
type DirectorCheck struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	How       string `json:"how"`
	CheckedAt string `json:"checked_at"`
	Detail    string `json:"detail"`
}

// DirectorCheckResult is what CheckDirector returns. Checked is false when
// there was nothing to check: no registration is current, the current one is
// a promoted session's (`ralph org director attach`), or it is stopped. Check
// and Registration are then zero. Otherwise Check is the result, and
// Registration is the registration it was computed from, as read at the start
// of the try that produced it (version A, plan D3): for a try whose
// registration did not change, the registration as it still is; for
// DirectorCheckHowChanged, the second try's read. version identifies that read,
// so a caller that acts on the result (start's cleanup, D6) can record what it
// did only while director.jsonl still holds the same version
// (directorVersionOf under director.lock).
type DirectorCheckResult struct {
	Checked      bool
	Check        DirectorCheck
	Registration DirectorRegistration
	version      directorVersion
}

// directorVersion is the version of registration id as one read of
// director.jsonl found it (plan D3): currentID, the id of the current
// registration then ("" for none); events, the number of lines that parse as
// an event and carry id, its attached included, whether the fold applied
// them or not; and folded, the registration as the fold read it (empty when
// no attached made one). Two versions are equal (==) exactly when no event for
// id was recorded in between and the current registration is the same one:
// director.jsonl is append-only (a failed write is cut back, and one that
// could not be cut back leaves a line that counts as soon as it parses), so a
// new event for id raises events, and the detached after which id is no
// longer current, or a new attached after that, changes currentID. folded is
// a second guard for a file rewritten by hand. Events for another
// registration change nothing here, unless they change which one is current.
type directorVersion struct {
	id        string
	currentID string
	events    int
	folded    string
}

// directorVersionOf is the directorVersion of id in raw, director.jsonl as
// one read found it, and fold, foldDirector(raw).
func directorVersionOf(raw []byte, fold *directorFold, id string) directorVersion {
	v := directorVersion{id: id}
	if fold.current != nil {
		v.currentID = fold.current.ID
	}
	if reg := fold.regs[id]; reg != nil {
		v.folded = fmt.Sprintf("%#v", *reg)
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev DirectorEvent
		if json.Unmarshal(line, &ev) == nil && ev.ID == id {
			v.events++
		}
	}
	return v
}

// readDirectorVersion reads the directorVersion of id under director.lock,
// which it holds only for the read and under which it writes nothing, so a
// writer that took the lock before it has finished its write (plan D3 and
// D4). A file that cannot be read, or a lock that cannot be taken, is an
// error.
func (s *DirectorStore) readDirectorVersion(id string) (directorVersion, error) {
	var v directorVersion
	err := s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		v = directorVersionOf(raw, foldDirector(raw), id)
		return nil
	})
	return v, err
}

// CheckDirector checks the current registration of store when it is
// headless and not stopped (plan 2026-10-18-org-director-liveness, D3), and
// records the check as made at now. It writes nothing anywhere: no
// director.jsonl line, and no herdr call but `agent get`, `pane get`, `tab
// get`, and `workspace get`. It returns a result with Checked false, and a
// nil error, when there is nothing to check (DirectorCheckResult). The error
// is set only when director.jsonl cannot be read: at the read without the
// lock, or at the read under it, which also fails when director.lock cannot
// be taken.
//
// Each try reads the registration without the lock (version A), runs
// o.directorCheckAfterRead (a test-only seam), checks it (checkDirectorOnce),
// and reads its version again under director.lock (readDirectorVersion).
// When the two versions are equal, the result is kept and returned with the
// registration of version A. When they differ (any event was recorded for it,
// such as launched, workspace_created, pane_created, stopped, or detached, or
// another registration became current), the result is thrown away and the
// next try starts from a fresh read, which may find nothing to check. After
// directorCheckAttempts tries that all changed, the result is unknown
// DirectorCheckHowChanged, with the registration the last try read. So a
// start whose launched landed after the read is never read as start_ended on
// the strength of its start process having ended since: the check reads it
// again and checks it as running.
//
// The rows of D3 (checkDirectorOnce):
//   - starting, launch herdr or cmd: the starter is read with
//     readDirectorProcess. No starter recorded is unknown no_starter; gone or
//     a zombie is ended start_ended; alive with another start time is unknown
//     starter_pid_reused; alive with the recorded one is alive start_running;
//     a read that fails, or a state the reader never gives, is unknown
//     starter_unreadable.
//   - running, launch herdr: `herdr agent get DirectorAgentName(token)`
//     through o.Herdr (watchHerdrProbe; without one, unknown no_agent_get),
//     under callWithTimeout. Success is alive agent_found. herdr's
//     agent_not_found (isAgentNotFound) checks the recorded pane as stop
//     checks it (confirmSeatPane with DirectorWorkspaceLabel and
//     DirectorTabLabel, directorLabelWords): a pane in the labelled tab and
//     workspace is ended agent_ended, a pane herdr no longer has is ended
//     pane_closed, and a label that differs or a get that fails is unknown
//     pane_unconfirmed. No pane recorded reads as pane_closed, since a stop of
//     the same registration (confirmDirectorResources) finds no pane to check
//     or close either. Any other failure of the agent get, a timeout or no
//     server included, is unknown agent_get_failed.
//   - running, launch cmd: the pid its launch script checked in with is read
//     with readDirectorProcess. Gone or a zombie is ended process_ended; alive
//     with another start time is unknown pid_reused; alive with the recorded
//     one is alive process_found; a read that fails, or a state the reader
//     never gives, is unknown process_unreadable.
func (o *Org) CheckDirector(store *DirectorStore, now time.Time) (DirectorCheckResult, error) {
	if store == nil {
		return DirectorCheckResult{}, errors.New("org: director: check: no director store was given")
	}
	checkedAt := now.UTC().Format(time.RFC3339)
	var res DirectorCheckResult
	for attempt := 1; attempt <= directorCheckAttempts; attempt++ {
		raw, err := store.readRaw()
		if err != nil {
			return DirectorCheckResult{}, err
		}
		fold := foldDirector(raw)
		cur := fold.current
		if cur == nil || !cur.Headless() || cur.State == DirectorStateStopped {
			return DirectorCheckResult{}, nil
		}
		reg := *cur
		version := directorVersionOf(raw, fold, reg.ID)
		if o.directorCheckAfterRead != nil {
			o.directorCheckAfterRead(attempt)
		}
		state, how, detail := o.checkDirectorOnce(reg)
		res = DirectorCheckResult{
			Checked:      true,
			Check:        DirectorCheck{ID: reg.ID, State: state, How: how, CheckedAt: checkedAt, Detail: detail},
			Registration: reg,
			version:      version,
		}
		again, err := store.readDirectorVersion(reg.ID)
		if err != nil {
			return DirectorCheckResult{}, err
		}
		if again == version {
			return res, nil
		}
	}
	res.Check.State, res.Check.How = DirectorCheckUnknown, DirectorCheckHowChanged
	res.Check.Detail = PrintableInboxText(fmt.Sprintf("director.jsonl recorded a change to %s (an event for it, or another current registration) while it was checked, on each of %d tries",
		res.Registration.ID, directorCheckAttempts), false)
	return res, nil
}

// checkDirectorOnce is one try of CheckDirector on reg, a headless
// registration that is starting or running (the fold has no other state for
// an open headless one): it returns the state, the how, and, for unknown, the
// detail, escaped. CheckDirector's doc lists the rows.
func (o *Org) checkDirectorOnce(reg DirectorRegistration) (state, how, detail string) {
	switch {
	case reg.State == DirectorStateStarting:
		state, how, detail = o.checkDirectorStarter(reg)
	case reg.HerdrLaunch():
		state, how, detail = o.checkDirectorAgent(reg)
	default:
		state, how, detail = o.checkDirectorProcess(reg)
	}
	return state, how, PrintableInboxText(detail, false)
}

// checkDirectorStarter is checkDirectorOnce for a starting registration: it
// reads the start process that recorded reg (D1 and D3).
func (o *Org) checkDirectorStarter(reg DirectorRegistration) (state, how, detail string) {
	if reg.StarterPID <= 0 || reg.StarterStartTime == "" {
		return DirectorCheckUnknown, DirectorCheckHowNoStarter,
			"no ralph org director start process is recorded for it, so whether its start still runs cannot be told"
	}
	pid := strconv.Itoa(reg.StarterPID)
	proc, err := o.readDirectorProcess(reg.StarterPID)
	switch {
	case err != nil:
		return DirectorCheckUnknown, DirectorCheckHowStarterUnreadable, "ps could not read the start process, pid " + pid + ": " + err.Error()
	case proc.State == DirectorProcessGone, proc.State == DirectorProcessZombie:
		return DirectorCheckEnded, DirectorCheckHowStartEnded, ""
	case proc.State != DirectorProcessAlive:
		return DirectorCheckUnknown, DirectorCheckHowStarterUnreadable, fmt.Sprintf("ps read the start process, pid %s, as %q", pid, proc.State)
	case proc.StartTime != reg.StarterStartTime:
		return DirectorCheckUnknown, DirectorCheckHowStarterPIDReused, fmt.Sprintf("pid %s, the start process, started %q, not %q as recorded: another process got the pid, or the clock moved",
			pid, proc.StartTime, reg.StarterStartTime)
	}
	return DirectorCheckAlive, DirectorCheckHowStartRunning, ""
}

// CheckDirectorStarter is the starter row of the check on its own (plan
// 2026-10-18-org-director-liveness, D3 and D7), for `ralph org director
// attach` refused by a starting headless registration: for reg, headless,
// starting, and with a starter recorded, it reads the starter with read
// (nil: ReadDirectorProcess, which runs ps) under one driver call timeout,
// as CheckDirector's row does (checkDirectorStarter), and returns the check
// made at now, its Detail escaped, and true. For any other registration it
// reads nothing and returns false. It makes no herdr call, writes nothing,
// and reads director.jsonl not at all, so it binds the result to no version:
// the result is for a refusal's text, never for acting on the registration.
func CheckDirectorStarter(reg DirectorRegistration, now time.Time, read func(context.Context, int) (DirectorProcess, error)) (DirectorCheck, bool) {
	if !reg.Headless() || reg.State != DirectorStateStarting || reg.StarterPID <= 0 || reg.StarterStartTime == "" {
		return DirectorCheck{}, false
	}
	o := &Org{DirectorProcessReader: read}
	state, how, detail := o.checkDirectorStarter(reg)
	return DirectorCheck{ID: reg.ID, State: state, How: how, CheckedAt: now.UTC().Format(time.RFC3339), Detail: PrintableInboxText(detail, false)}, true
}

// checkDirectorAgent is checkDirectorOnce for a running launch herdr: it asks
// herdr for the agent and, when herdr has none, looks at the recorded pane
// (D3).
func (o *Org) checkDirectorAgent(reg DirectorRegistration) (state, how, detail string) {
	probe, ok := o.Herdr.(watchHerdrProbe)
	if !ok {
		return DirectorCheckUnknown, DirectorCheckHowNoAgentGet, "the herdr client configured cannot run herdr agent get"
	}
	name := DirectorAgentName(reg.Token)
	err := o.callWithTimeout(func(ctx context.Context) error {
		_, err := probe.AgentGet(ctx, name)
		return err
	})
	switch {
	case err == nil:
		return DirectorCheckAlive, DirectorCheckHowAgentFound, ""
	case !isAgentNotFound(err):
		return DirectorCheckUnknown, DirectorCheckHowAgentGetFailed, "herdr agent get " + name + " failed: " + err.Error()
	case reg.PaneID == "":
		return DirectorCheckEnded, DirectorCheckHowPaneClosed, ""
	}
	gone, err := o.confirmSeatPane(DirectorWorkspaceLabel(reg.Token), DirectorTabLabel(reg.Token), reg.PaneID, directorLabelWords)
	switch {
	case err != nil:
		return DirectorCheckUnknown, DirectorCheckHowPaneUnconfirmed, "herdr has no agent " + name + ", but its pane is not confirmed: " + err.Error()
	case gone:
		return DirectorCheckEnded, DirectorCheckHowPaneClosed, ""
	}
	return DirectorCheckEnded, DirectorCheckHowAgentEnded, ""
}

// checkDirectorProcess is checkDirectorOnce for a running launch cmd: it reads
// the pid its launch script checked in with (D3). Unlike directorProcessEnded,
// another start time is unknown, not ended.
func (o *Org) checkDirectorProcess(reg DirectorRegistration) (state, how, detail string) {
	pid := strconv.Itoa(reg.PID)
	proc, err := o.readDirectorProcess(reg.PID)
	switch {
	case err != nil:
		return DirectorCheckUnknown, DirectorCheckHowProcessUnreadable, "ps could not read pid " + pid + ", which its launch script checked in with: " + err.Error()
	case proc.State == DirectorProcessGone, proc.State == DirectorProcessZombie:
		return DirectorCheckEnded, DirectorCheckHowProcessEnded, ""
	case proc.State != DirectorProcessAlive:
		return DirectorCheckUnknown, DirectorCheckHowProcessUnreadable, fmt.Sprintf("ps read pid %s, which its launch script checked in with, as %q", pid, proc.State)
	case proc.StartTime != reg.StartTime:
		return DirectorCheckUnknown, DirectorCheckHowPIDReused, fmt.Sprintf("pid %s started %q, not %q as its check-in recorded: another process got the pid, or the clock moved",
			pid, proc.StartTime, reg.StartTime)
	}
	return DirectorCheckAlive, DirectorCheckHowProcessFound, ""
}

// The mark of the check (plan 2026-10-18-org-director-liveness, D4 and D8):
// each cycle of `ralph org watch --all` whose CheckDirector checked a
// registration (Checked) replaces director-alive.json in the org state dir
// with the result and that registration's attached_at
// (watchAllRun.checkDirector, watch.go). One watch runs per ledger
// (watch.lock), so the mark has one writer. A cycle with nothing to check
// writes nothing and leaves the last mark as it was, so the mark may be of a
// registration that is no longer current: a reader goes by DirectorAliveFor,
// which keeps a mark only for the registration whose id and attached_at it
// holds (an id is given again once director.jsonl is removed, and the
// attached_at tells the two apart). The mark is never read to decide what to
// close or stop.

// DirectorAliveFileName is the file name of the mark within the org state
// dir.
const DirectorAliveFileName = "director-alive.json"

// DirectorAlive is the mark, director-alive.json, with its six keys id,
// attached_at, state, how, checked_at, and detail: the DirectorCheck of the
// registration the watch last checked, with AttachedAt, that registration's
// attached_at (DirectorRegistration.AttachedAt). A key added here is not
// added to status's director.check (the CLI's orgDirectorReportCheck), which
// copies four of them. Its values are read from a file, so a reader keeps
// them as read and escapes them where it prints them (PrintableInboxText),
// as it does the registration's.
type DirectorAlive struct {
	ID         string `json:"id"`
	AttachedAt string `json:"attached_at"`
	State      string `json:"state"`
	How        string `json:"how"`
	CheckedAt  string `json:"checked_at"`
	Detail     string `json:"detail"`
}

// directorAliveOf is the mark of res, a result with Checked set: its Check
// and the attached_at of its Registration.
func directorAliveOf(res DirectorCheckResult) DirectorAlive {
	c := res.Check
	return DirectorAlive{ID: c.ID, AttachedAt: res.Registration.AttachedAt, State: c.State, How: c.How, CheckedAt: c.CheckedAt, Detail: c.Detail}
}

// Check is the mark's DirectorCheck, without the attached_at, for a reader
// that prints it (attach's refusal). The director.check of status's --json
// (D8) does not come from it: the CLI copies the mark's state, how,
// checked_at, and detail into orgDirectorReportCheck.
func (m DirectorAlive) Check() DirectorCheck {
	return DirectorCheck{ID: m.ID, State: m.State, How: m.How, CheckedAt: m.CheckedAt, Detail: m.Detail}
}

// DirectorAlivePathIn returns the director-alive.json path within an
// already-resolved org state dir (see WatchAlivePathIn).
func DirectorAlivePathIn(stateDir string) string {
	return filepath.Join(stateDir, DirectorAliveFileName)
}

// writeDirectorAlive replaces the mark in stateDir (writeJSONFileAtomic, as
// writeWatchAlive does), so a reader sees the old mark or the new, never a
// part.
func writeDirectorAlive(stateDir string, mark DirectorAlive) error {
	return writeJSONFileAtomic(DirectorAlivePathIn(stateDir), mark)
}

// ReadDirectorAlive reads the mark in stateDir. A missing mark (or state dir)
// is nil with a nil error: no watch has checked a registration of this
// ledger yet. A mark that cannot be read or does not parse is an error. The
// mark returned may be of any registration; pass it through DirectorAliveFor.
func ReadDirectorAlive(stateDir string) (*DirectorAlive, error) {
	path := DirectorAlivePathIn(stateDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("org: director: read %s: %w", path, err)
	}
	var mark DirectorAlive
	if err := json.Unmarshal(data, &mark); err != nil {
		return nil, fmt.Errorf("org: director: parse %s: %w", path, err)
	}
	return &mark, nil
}

// DirectorAliveFor returns mark when it is the mark of reg: both are set,
// and the mark's id and attached_at equal reg's ID and AttachedAt. Otherwise
// it returns nil, and the reader says reg was not checked yet (D8). It
// compares the values only; it does not check them.
func DirectorAliveFor(mark *DirectorAlive, reg *DirectorRegistration) *DirectorAlive {
	if mark == nil || reg == nil || mark.ID != reg.ID || mark.AttachedAt != reg.AttachedAt {
		return nil
	}
	return mark
}
