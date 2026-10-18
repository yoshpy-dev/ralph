package org

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The wake of a registration (plan 2026-10-18-org-director-wake, D3, D4, and
// D8): the refusals of attach and start, the check of a wake target, and how
// the watch reads a registration's wake and target. The Wake and WakeTarget
// fields of a registration, DirectorWakePull, and DirectorWakePush are in
// director.go.

// MaxDirectorWakeTargetRunes is the longest wake target
// ValidateDirectorWakeTarget accepts, in characters (runes).
const MaxDirectorWakeTargetRunes = 256

// The refusals of CheckDirectorAttachWake and of StartDirector's wake check
// (plan 2026-10-18-org-director-wake, D4), each wrapped by an error that
// says the wake and what to do.
var (
	// ErrDirectorWakeCmdNeeded: the wake has push, but nothing could send
	// the wake line: [org.director].wake_cmd is empty, and the registration
	// has no default way to be sent to (a promoted session, or a headless
	// director in a terminal launch_cmd opened; only a headless director in
	// herdr has one, its herdr pane).
	ErrDirectorWakeCmdNeeded = errors.New("the wake has push, but [org.director].wake_cmd is empty")
	// ErrDirectorWakeTargetNeeded: the wake has push and
	// [org.director].wake_cmd has {target}, but attach was given no
	// --wake-target to put there.
	ErrDirectorWakeTargetNeeded = errors.New("[org.director].wake_cmd has {target}, but no --wake-target was given")
	// ErrDirectorWakeTargetWithoutPush: attach was given --wake-target, but
	// its wake has no push, so nothing would ever be sent there.
	ErrDirectorWakeTargetWithoutPush = errors.New("--wake-target is given, but the wake has no push")
	// ErrDirectorWakeTargetInvalid: the wake target is not of a shape the
	// watch sends to (ValidateDirectorWakeTarget).
	ErrDirectorWakeTargetInvalid = errors.New("the wake target is not usable")
)

// WakesByPush reports whether the registration's wake has push, so `ralph
// org watch --all` wakes it when a new open inbox item arrives (plan
// 2026-10-18-org-director-wake, D3 and D6). The watch goes by this recorded
// wake, not by [org.director].wake.
func (r DirectorRegistration) WakesByPush() bool {
	return slices.Contains(r.Wake, DirectorWakePush)
}

// ResolvedWakeTarget is where the wake line of the registration goes, the
// {target} of [org.director].wake_cmd (plan 2026-10-18-org-director-wake,
// D3): for a headless director in herdr its pane (PaneID, empty until its
// pane_created), for one in a terminal launch_cmd opened the terminal's name
// DirectorAgentName(token), which launch_cmd's {name} gave it, and for a
// promoted session its WakeTarget, the --wake-target of attach (empty when
// none was given). A registration of a launch this binary does not know
// counts as a promoted session's.
func (r DirectorRegistration) ResolvedWakeTarget() string {
	switch {
	case r.HerdrLaunch():
		return r.PaneID
	case r.CmdLaunch():
		return DirectorAgentName(r.Token)
	}
	return r.WakeTarget
}

// ValidateDirectorWakeTarget returns an error wrapping
// ErrDirectorWakeTargetInvalid and saying why when target is not a wake
// target the watch sends to (plan 2026-10-18-org-director-wake, D4 and D8):
// it must not be empty, must be valid UTF-8 with no control character
// (unicode.IsControl), must be at most MaxDirectorWakeTargetRunes
// characters long, and must not start with "-", which the command
// [org.director].wake_cmd runs could read as an option. attach checks
// --wake-target with it before anything is written, and the watch checks
// a recorded target with it before it sends, since director.jsonl can be
// edited by hand. The value is quoted in the error, so its characters
// cannot drive a terminal.
func ValidateDirectorWakeTarget(target string) error {
	var reason string
	switch {
	case target == "":
		reason = "it is empty"
	case !utf8.ValidString(target):
		reason = fmt.Sprintf("%q is not valid UTF-8", target)
	case strings.IndexFunc(target, unicode.IsControl) >= 0:
		reason = fmt.Sprintf("%q has a control character", target)
	case utf8.RuneCountInString(target) > MaxDirectorWakeTargetRunes:
		reason = fmt.Sprintf("it is %d characters long, more than %d", utf8.RuneCountInString(target), MaxDirectorWakeTargetRunes)
	case strings.HasPrefix(target, "-"):
		reason = fmt.Sprintf(`%q starts with "-", which the command [org.director].wake_cmd runs could read as an option`, target)
	default:
		return nil
	}
	return fmt.Errorf("%w: %s", ErrDirectorWakeTargetInvalid, reason)
}

// CheckDirectorAttachWake is `ralph org director attach`'s check that the
// wake it is about to record can be sent (plan 2026-10-18-org-director-wake,
// D4): wake is the resolved wake (the --wake value, else
// [org.director].wake), target the --wake-target ("" when none was given),
// and wakeCmd [org.director].wake_cmd, all read by the caller, so a refusal
// writes nothing. In this order, it refuses:
//
//   - a wake with push while wakeCmd is empty (ErrDirectorWakeCmdNeeded): a
//     promoted session has no default way to be sent the wake line
//   - a wake with push, a wakeCmd that has {target} in some element, and no
//     target (ErrDirectorWakeTargetNeeded)
//   - a target with a wake that has no push
//     (ErrDirectorWakeTargetWithoutPush)
//   - a target ValidateDirectorWakeTarget refuses
//     (ErrDirectorWakeTargetInvalid)
//
// The {target} test is a substring test, which finds what the placeholder
// rule of config.ValidateDirectorWakeCmd finds: no brace can come between
// the braces of "{target}". The wake itself and wakeCmd are not validated
// here: the CLI validates --wake, and config.Load the two settings. Its
// errors do not start with "org: "; the CLI adds its verb.
func CheckDirectorAttachWake(wake []string, target string, wakeCmd []string) error {
	push := slices.Contains(wake, DirectorWakePush)
	joined := strings.Join(wake, ",")
	switch {
	case push && len(wakeCmd) == 0:
		return fmt.Errorf("%w (wake %s): a session attached with ralph org director attach has no default way to be sent the wake line "+
			"(only a headless director in herdr has one, its herdr pane), so ralph org watch --all could not wake it; "+
			"set [org.director].wake_cmd, or attach with --wake pull", ErrDirectorWakeCmdNeeded, joined)
	case push && target == "" && directorWakeCmdHasTarget(wakeCmd):
		return fmt.Errorf("%w (wake %s): pass --wake-target with where [org.director].wake_cmd sends the wake line", ErrDirectorWakeTargetNeeded, joined)
	case !push && target != "":
		return fmt.Errorf("%w (wake %s): the watch sends to the wake target only to push; drop --wake-target, or attach with --wake push or --wake pull,push",
			ErrDirectorWakeTargetWithoutPush, joined)
	case target != "":
		if err := ValidateDirectorWakeTarget(target); err != nil {
			return fmt.Errorf("--wake-target: %w", err)
		}
	}
	return nil
}

// The push wake of stage 8b-1 (plan 2026-10-18-org-director-wake, D5 to D8):
// each cycle of `ralph org watch --all` (RunWatchAll), right after the inbox
// deadlines, wakes the attached director when its registration wakes by
// push and the inbox holds open items it was not woken for yet. One fixed
// line that names their ids (DirectorWakeText) is sent once per cycle, by
// [org.director].wake_cmd or, when that is empty, as input to the herdr pane
// of a headless director in herdr; then every one of those items gets a
// woken event (InboxStore.AppendWoken). A wake that fails is not tried again
// for the same registration for directorWakeRetry, in the watch's memory
// only (DirectorWakeMemory). Waking leaves the inbox deadlines alone: an item
// the director does not ack still reaches the human at its ack deadline
// (D10). The registration's wake and target are read with WakesByPush and
// ResolvedWakeTarget above; the runner wake_cmd shares with launch_cmd is
// runDirectorCmd in director_cmd.go.

const (
	// directorWakeRetry is how long WakeDirector waits after a failed wake of
	// a registration before it tries that registration again (D6).
	directorWakeRetry = 5 * time.Minute
	// directorWakeListMax is how many item ids the wake line and the failure
	// lines name; the rest are only counted (D6 and D7).
	directorWakeListMax = 20
	// directorWakeOutputLimit is how much of wake_cmd's output, its last
	// bytes, the failure line of a wake holds (D8): the output is otherwise
	// discarded.
	directorWakeOutputLimit = 512
)

// Why WakeDirector sent nothing without a failure:
// DirectorWakeResult.Skipped. The first four write no line; the last three
// are each said once per registration on the banner (DirectorWakeMemory).
const (
	// DirectorWakeSkippedNoRegistration: no registration is current.
	DirectorWakeSkippedNoRegistration = "no_registration"
	// DirectorWakeSkippedNoPush: the registration's wake has no push.
	DirectorWakeSkippedNoPush = "no_push"
	// DirectorWakeSkippedNotRunning: the registration is headless and not
	// running (starting, or stopped and not detached yet).
	DirectorWakeSkippedNotRunning = "not_running"
	// DirectorWakeSkippedNothingToWake: no item is open and not woken.
	DirectorWakeSkippedNothingToWake = "nothing_to_wake"
	// DirectorWakeSkippedWakeCmdInvalid: [org.director].wake_cmd is not one
	// config.ValidateDirectorWakeCmd accepts (an Org can be built without
	// config.Load).
	DirectorWakeSkippedWakeCmdInvalid = "wake_cmd_invalid"
	// DirectorWakeSkippedTargetInvalid: the target the line would go to
	// (wake_cmd's {target}, or the herdr pane) is not one
	// ValidateDirectorWakeTarget accepts: a hand-edited director.jsonl, or a
	// herdr pane not recorded.
	DirectorWakeSkippedTargetInvalid = "target_invalid"
	// DirectorWakeSkippedNoSender: wake_cmd is empty and the registration is
	// not a headless director in herdr, so nothing can type into it.
	DirectorWakeSkippedNoSender = "no_sender"
)

// DirectorWakeText is the one line the watch sends to wake a director for
// the items ids (D7): their ids in the order given (the inbox's ID order),
// at most directorWakeListMax of them and then how many more, and where to
// read them. It never holds a body, a TYPE, an org_id, or a target, holds
// none of the shell characters ; | & $ ` < > ( ) * ? ' ", and does not start
// with /, !, or #, so wherever the line lands it is no command. ids must be
// inbox item ids (e<N>, which the fold guarantees) and not empty.
func DirectorWakeText(ids []string) string {
	return "New org inbox items for the director: " + directorWakeIDList(ids) +
		". Read them with ralph org inbox and handle each as the director skill says."
}

// directorWakeIDList is ids joined by one space, cut to the first
// directorWakeListMax with " plus <n> more" after them: the list of
// DirectorWakeText and of the failure lines.
func directorWakeIDList(ids []string) string {
	if len(ids) <= directorWakeListMax {
		return strings.Join(ids, " ")
	}
	return strings.Join(ids[:directorWakeListMax], " ") + fmt.Sprintf(" plus %d more", len(ids)-directorWakeListMax)
}

// DirectorWakeItems returns the ids of the items a wake is for (D5): open and
// not woken, in the order of items (InboxReadResult.Items is in ID order).
// The delivery does not matter: an item escalate sent to the human at once
// is woken for as one it left to the deadline is.
func DirectorWakeItems(items []InboxItem) []string {
	var ids []string
	for _, item := range items {
		if item.State == InboxStateOpen && !item.Woken {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

// DirectorWakeMemory is what one watch process remembers between
// WakeDirector calls, in memory only, so a restarted watch starts with
// nothing remembered and tries at once. It holds the time of the last failed
// wake of each registration id, so a director that cannot be woken is not
// tried every cycle, and the registrations already warned about, per
// reason, so each warning is written once. A replaced registration has a new
// id, so neither applies to it. The zero value is ready to use. It is not
// safe for concurrent use.
type DirectorWakeMemory struct {
	failedAt map[string]time.Time
	warned   map[directorWakeWarnKey]bool
}

type directorWakeWarnKey struct {
	id, reason string
}

// holdsBack reports whether the wake of registration id failed less than
// directorWakeRetry before now.
func (m *DirectorWakeMemory) holdsBack(id string, now time.Time) bool {
	at, ok := m.failedAt[id]
	return ok && now.Sub(at) < directorWakeRetry
}

func (m *DirectorWakeMemory) recordFailure(id string, now time.Time) {
	if m.failedAt == nil {
		m.failedAt = map[string]time.Time{}
	}
	m.failedAt[id] = now
}

func (m *DirectorWakeMemory) recordSuccess(id string) {
	delete(m.failedAt, id)
}

// warnOnce reports whether the warning reason for registration id is still
// to be written, and remembers that it now is.
func (m *DirectorWakeMemory) warnOnce(id, reason string) bool {
	key := directorWakeWarnKey{id: id, reason: reason}
	if m.warned[key] {
		return false
	}
	if m.warned == nil {
		m.warned = map[directorWakeWarnKey]bool{}
	}
	m.warned[key] = true
	return true
}

// DirectorWakeResult is what one WakeDirector call did. Err is set when the
// step could not start (director.jsonl or the inbox could not be read, or
// there is no inbox store); the caller (the watch) logs it and goes on.
type DirectorWakeResult struct {
	// RegistrationID is the d<N> of the current registration, "" when none.
	RegistrationID string
	// IDs are the items to wake (DirectorWakeItems), set once the inbox was
	// read.
	IDs []string
	// Skipped is one of the DirectorWakeSkipped* reasons when nothing was
	// sent and nothing failed.
	Skipped string
	// HeldBack is set when nothing was tried because the wake of this
	// registration failed less than directorWakeRetry ago.
	HeldBack bool
	// Sent is set when the line was sent, by Via (one of the InboxWokenVia*
	// values).
	Sent bool
	Via  string
	// Recorded is set when the woken events of IDs were written.
	Recorded bool
	// Failure is why the send failed, or, with Sent, why the woken events
	// were not written; WakeDirector has written its line on the banner and
	// recorded it in the memory.
	Failure error
	Err     error
}

// WakeDirector is the wake step of one watch cycle (D6 and D8): it wakes the
// registration current in director.jsonl at stateDir for the inbox items it
// was not woken for yet. In this order:
//
//  1. It reads director.jsonl (an error is Err) and does nothing, with no
//     line, when no registration is current, when the registration's wake
//     has no push, and when the registration is headless and not running
//     (directorToWake).
//  2. It reads the inbox (an error is Err) and does nothing when no item is
//     open and not woken (directorToWake).
//  3. It does nothing when mem holds the registration back: its wake failed
//     less than directorWakeRetry before now (HeldBack).
//  4. It chooses the sender, wake_cmd or the herdr pane, and skips a
//     registration that has none, saying so once on banner
//     (sendDirectorWake).
//  5. It sends DirectorWakeText(ids) once. When that succeeds it appends a
//     woken event for every item; when it fails it writes no woken event. A
//     failed send or write is said on banner and recorded in mem
//     (sendDirectorWake).
//
// wakeCmdDir is the absolute directory wake_cmd runs in, the directory of
// the ralph.toml it was read from (D9; WatchAllParams.DirectorWakeDir); an
// empty or relative one fails every wake by wake_cmd. mem nil means nothing
// is remembered, and banner nil means os.Stderr. Every value read from a
// file is escaped on the banner (PrintableInboxText), and no line holds an
// item's body. Config.Director.Enabled is not read.
func (o *Org) WakeDirector(now time.Time, stateDir, wakeCmdDir string, mem *DirectorWakeMemory, banner io.Writer) DirectorWakeResult {
	reg, res := o.directorToWake(stateDir)
	if reg == nil {
		return res
	}
	if mem == nil {
		mem = &DirectorWakeMemory{}
	}
	banner = bannerOrStderr(banner)
	if mem.holdsBack(reg.ID, now) {
		res.HeldBack = true
		return res
	}
	return o.sendDirectorWake(*reg, res, now, wakeCmdDir, mem, banner)
}

// directorToWake is steps 1 and 2 of WakeDirector. It reads director.jsonl
// at stateDir and the inbox (Org.Inbox), and returns the current
// registration with a result that holds its RegistrationID and the IDs to
// wake (DirectorWakeItems). It returns no registration, with the result
// WakeDirector returns as it is, when director.jsonl or the inbox cannot be
// read or there is no inbox store (Err), and when there is nothing to send
// (Skipped, and no line is written): no registration is current, the
// registration's wake has no push (WakesByPush), the registration is
// headless and not running, or no item is open and not woken.
func (o *Org) directorToWake(stateDir string) (*DirectorRegistration, DirectorWakeResult) {
	dres, err := NewDirectorStore(stateDir).Read()
	if err != nil {
		return nil, DirectorWakeResult{Err: fmt.Errorf("org: director wake: %w", err)}
	}
	reg := dres.Current
	if reg == nil {
		return nil, DirectorWakeResult{Skipped: DirectorWakeSkippedNoRegistration}
	}
	res := DirectorWakeResult{RegistrationID: reg.ID}
	switch {
	case !reg.WakesByPush():
		res.Skipped = DirectorWakeSkippedNoPush
		return nil, res
	case reg.Headless() && reg.State != DirectorStateRunning:
		res.Skipped = DirectorWakeSkippedNotRunning
		return nil, res
	case o.Inbox == nil:
		res.Err = errors.New("org: director wake: no inbox store is configured")
		return nil, res
	}
	inbox, err := o.Inbox.Read()
	if err != nil {
		res.Err = fmt.Errorf("org: director wake: %w", err)
		return nil, res
	}
	if res.IDs = DirectorWakeItems(inbox.Items); len(res.IDs) == 0 {
		res.Skipped = DirectorWakeSkippedNothingToWake
		return nil, res
	}
	return reg, res
}

// sendDirectorWake is steps 4 and 5 of WakeDirector for reg, with res as
// directorToWake returned it. It chooses the sender (directorWakeSender):
// Config.Director.WakeCmd when it is not empty, run in wakeCmdDir
// (runDirectorWakeCmd); otherwise, for a headless director in herdr, its
// herdr pane (typeDirectorWakeLine); otherwise nothing, since nothing else
// can type into a promoted session or into the terminal launch_cmd opened.
// A wake_cmd that is not valid, a target that is not usable, and no sender
// are each said once per registration on banner and skipped. Then it sends
// DirectorWakeText(res.IDs) once. On success it appends one woken event per
// item, all items included (not only the ones the line names), with reg's
// id and the way it was sent; a write that fails is said on banner and
// recorded in mem as a failure, so a broken inbox is not woken every cycle.
// On failure it records the failure in mem at now, writes one line on
// banner, and writes no woken event.
func (o *Org) sendDirectorWake(reg DirectorRegistration, res DirectorWakeResult, now time.Time, wakeCmdDir string, mem *DirectorWakeMemory, banner io.Writer) DirectorWakeResult {
	sender, skipped, why := o.directorWakeSender(reg, wakeCmdDir)
	if skipped != "" {
		res.Skipped = skipped
		if mem.warnOnce(reg.ID, skipped) {
			writeDirectorWakeLine(banner, "watchdog: director %s is not woken: %s (said once)", reg.ID, why)
		}
		return res
	}
	if err := sender.send(DirectorWakeText(res.IDs)); err != nil {
		mem.recordFailure(reg.ID, now)
		res.Failure = err
		writeDirectorWakeLine(banner, "watchdog: director %s not woken for %s: %v", reg.ID, directorWakeIDList(res.IDs), err)
		return res
	}
	res.Sent, res.Via = true, sender.via
	if err := o.Inbox.AppendWoken(res.IDs, reg.ID, sender.via); err != nil {
		mem.recordFailure(reg.ID, now)
		res.Failure = err
		writeDirectorWakeLine(banner, "watchdog: director %s was woken for %s, but the woken events were not recorded, so it is not woken again for %s: %v",
			reg.ID, directorWakeIDList(res.IDs), directorWakeRetry, err)
		return res
	}
	mem.recordSuccess(reg.ID)
	res.Recorded = true
	return res
}

// writeDirectorWakeLine writes one banner line of WakeDirector, escaped as a
// whole (PrintableInboxText), since its values come from director.jsonl,
// from herdr, and from wake_cmd's output.
func writeDirectorWakeLine(banner io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintln(banner, PrintableInboxText(fmt.Sprintf(format, args...), false))
}

// directorWakeSend is how WakeDirector sends the wake line to a
// registration: send sends text, and via (one of the InboxWokenVia* values)
// is what the woken events record.
type directorWakeSend struct {
	via  string
	send func(text string) error
}

// directorWakeSender chooses how the wake line reaches reg (D8). With
// Config.Director.WakeCmd set, that runs for every registration
// (runDirectorWakeCmd): it must pass config.ValidateDirectorWakeCmd, and
// when an element has {target}, reg's target (ResolvedWakeTarget) must pass
// ValidateDirectorWakeTarget, which also catches a hand-edited
// director.jsonl. With no wake_cmd, a headless director in herdr is typed
// into at its pane (typeDirectorWakeLine), which must pass the same check;
// any other registration has no sender. When there is no sender, skipped is
// the DirectorWakeSkipped* reason and why says it for the banner.
func (o *Org) directorWakeSender(reg DirectorRegistration, wakeCmdDir string) (s directorWakeSend, skipped, why string) {
	wakeCmd := o.Config.Director.WakeCmd
	if len(wakeCmd) > 0 {
		if err := config.ValidateDirectorWakeCmd(wakeCmd); err != nil {
			return s, DirectorWakeSkippedWakeCmdInvalid, fmt.Sprintf("%v; fix it and restart ralph org watch --all", err)
		}
		target := ""
		if directorWakeCmdHasTarget(wakeCmd) {
			target = reg.ResolvedWakeTarget()
			if err := ValidateDirectorWakeTarget(target); err != nil {
				return s, DirectorWakeSkippedTargetInvalid, fmt.Sprintf("[org.director].wake_cmd has {target}, but %v", err)
			}
		}
		return directorWakeSend{via: InboxWokenViaWakeCmd, send: func(text string) error {
			return o.runDirectorWakeCmd(wakeCmd, target, text, wakeCmdDir)
		}}, "", ""
	}
	if !reg.HerdrLaunch() {
		what := "a session attached with ralph org director attach"
		if reg.CmdLaunch() {
			what = "a headless director in the terminal [org.director].launch_cmd opened"
		}
		return s, DirectorWakeSkippedNoSender, fmt.Sprintf("its wake has push, but [org.director].wake_cmd is empty, and nothing else can type into %s; "+
			"its inbox items reach the human at the ack deadline", what)
	}
	if err := ValidateDirectorWakeTarget(reg.PaneID); err != nil {
		return s, DirectorWakeSkippedTargetInvalid, fmt.Sprintf("its herdr pane is not one to type into: %v", err)
	}
	return directorWakeSend{via: InboxWokenViaHerdr, send: func(text string) error {
		return o.typeDirectorWakeLine(reg, text)
	}}, "", ""
}

// directorWakeCmdHasTarget reports whether an element of wakeCmd has
// {target}, for CheckDirectorAttachWake and directorWakeSender. A substring
// test finds what the placeholder rule of config.ValidateDirectorWakeCmd
// finds: no brace can come between the braces of "{target}".
func directorWakeCmdHasTarget(wakeCmd []string) bool {
	return slices.ContainsFunc(wakeCmd, func(arg string) bool { return strings.Contains(arg, config.DirectorWakePlaceholderTarget) })
}

// runDirectorWakeCmd runs wakeCmd, [org.director].wake_cmd, with {target}
// replaced by target and {text} by text in every element
// (config.ExpandDirectorWakeArg), without a shell, in dir (D8 and D9):
// runDirectorCmd under driverCallTimeout, with stdin empty and the last
// directorWakeOutputLimit bytes of its output kept for the error. A first
// element that contains "/" and is not absolute is resolved from dir, so
// the program is the one next to the ralph.toml wake_cmd was read from, not
// one in the watch's working directory; a name without "/" is looked up in
// PATH as exec does. dir must be absolute: an empty or relative one is an
// error, and nothing runs.
func (o *Org) runDirectorWakeCmd(wakeCmd []string, target, text, dir string) error {
	switch {
	case dir == "":
		return errors.New("wake_cmd: no directory to run it in was given (the directory of the ralph.toml it was read from)")
	case !filepath.IsAbs(dir):
		return fmt.Errorf("wake_cmd: the directory to run it in must be absolute (the directory of the ralph.toml it was read from), got %q", dir)
	}
	values := map[string]string{config.DirectorWakePlaceholderTarget: target, config.DirectorWakePlaceholderText: text}
	argv := make([]string, 0, len(wakeCmd))
	for _, arg := range wakeCmd {
		expanded, err := config.ExpandDirectorWakeArg(arg, values)
		if err != nil {
			return err
		}
		argv = append(argv, expanded)
	}
	if strings.Contains(argv[0], "/") && !filepath.IsAbs(argv[0]) {
		argv[0] = filepath.Join(dir, argv[0])
	}
	return o.callWithTimeout(func(ctx context.Context) error {
		return runDirectorCmd(ctx, "wake_cmd", argv, dir, 0, directorWakeOutputLimit)
	})
}

// typeDirectorWakeLine types text into the herdr pane of reg, a headless
// director in herdr, and presses Enter (D8). It first confirms the pane as
// stop does (confirmSeatPane with DirectorTabLabel and
// DirectorWorkspaceLabel of reg's token, directorLabelWords): a pane herdr
// no longer has, or one not confirmed, is an error, and nothing is typed.
// Then `herdr pane send-text`, a wait of sendEnterDelay (the 750 ms of
// `ralph org send`, so Enter is not read as part of the text), and `herdr
// pane send-keys Enter`, each under its own driverCallTimeout.
func (o *Org) typeDirectorWakeLine(reg DirectorRegistration, text string) error {
	if o.Herdr == nil {
		return errors.New("no herdr client is configured")
	}
	tabLabel, wsLabel := DirectorTabLabel(reg.Token), DirectorWorkspaceLabel(reg.Token)
	gone, err := o.confirmSeatPane(wsLabel, tabLabel, reg.PaneID, directorLabelWords)
	switch {
	case err != nil:
		return fmt.Errorf("pane %q is not confirmed as %s's (a tab labelled %s in a workspace labelled %s), so nothing was typed into it: %w",
			reg.PaneID, reg.ID, tabLabel, wsLabel, err)
	case gone:
		return fmt.Errorf("herdr has no pane %q, so nothing was typed into it", reg.PaneID)
	}
	if err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneSendText(ctx, reg.PaneID, text)
	}); err != nil {
		return fmt.Errorf("herdr pane send-text %q: %w", reg.PaneID, err)
	}
	time.Sleep(o.sendEnterDelay())
	if err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneSendKeys(ctx, reg.PaneID, "Enter")
	}); err != nil {
		return fmt.Errorf("herdr pane send-keys %q Enter, after the line was typed: %w", reg.PaneID, err)
	}
	return nil
}
