package org

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The machine's load average around the gate scripts (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, D2, D4 and D7, issue
// #309). runGateScripts (gate_run.go) reads it right before the first script
// and right after the last one that ran, and a sampler reads the 1-minute
// value every gateLoadSampleInterval while they run and keeps the highest
// (gateLoadSampler); the log and the gate record (gate_record.go) hold the
// values. A gate with [org.gate].max_load above 0 waits after it took
// gate.lock while the 1-minute value is above it, for at most
// gateMaxLoadCap (waitGateMaxLoad). A load that cannot be read is
// "unknown" and never makes a gate wait.

// gateLoadUnknown is the value the log, the record, and the CLI give a load
// average or a CPU count that could not be read.
const gateLoadUnknown = "unknown"

// gateLoadReadTimeout bounds the sysctl that reads the load average on
// darwin (readGateLoad).
const gateLoadReadTimeout = 2 * time.Second

// gateSysctlPath is the sysctl that reads the load average on darwin, named
// by its path: a caller's PATH (a herdr pane's, a test's) may not hold
// /usr/sbin.
const gateSysctlPath = "/usr/sbin/sysctl"

// gateLoadAvg is the system's load average: the 1-, 5-, and 15-minute values.
type gateLoadAvg struct {
	one, five, fifteen float64
}

// gateLoadReadFunc reads the system's load average.
type gateLoadReadFunc func(ctx context.Context) (gateLoadAvg, error)

var (
	// gateLoadReader is how Gate and Regate read the load average:
	// readGateLoad. Tests replace it (TestMain pins a fixed value).
	gateLoadReader gateLoadReadFunc = readGateLoad
	// gateNumCPU is the CPU count the log and the record give with the load
	// average: runtime.NumCPU. Tests replace it.
	gateNumCPU = runtime.NumCPU
	// gateLoadSampleInterval is how often the sampler reads the 1-minute
	// load while the scripts run (D4). Tests shorten it.
	gateLoadSampleInterval = 30 * time.Second
	// gateMaxLoadPoll is how often a gate waiting for the load to come down
	// to [org.gate].max_load reads it again (D2). Tests shorten it.
	gateMaxLoadPoll = 30 * time.Second
	// gateMaxLoadCap is how long in all a gate waits for the load to come
	// down to [org.gate].max_load before it starts anyway (D2). Tests shorten
	// it.
	gateMaxLoadCap = 30 * time.Minute
)

// readGateLoad reads the system's load average: on darwin from `sysctl -n
// vm.loadavg` (parseSysctlLoadavg), run with LC_ALL=C and bounded by
// gateLoadReadTimeout; on linux from /proc/loadavg (parseProcLoadavg). Any
// other OS is an error.
func readGateLoad(ctx context.Context) (gateLoadAvg, error) {
	switch runtime.GOOS {
	case "darwin":
		ctx, cancel := context.WithTimeout(ctx, gateLoadReadTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, gateSysctlPath, "-n", "vm.loadavg")
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		out, err := cmd.Output()
		if err != nil {
			return gateLoadAvg{}, fmt.Errorf("%s -n vm.loadavg: %w", gateSysctlPath, err)
		}
		return parseSysctlLoadavg(string(out))
	case "linux":
		data, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return gateLoadAvg{}, err
		}
		return parseProcLoadavg(string(data))
	}
	return gateLoadAvg{}, fmt.Errorf("ralph reads the load average on darwin and linux only, not on %s", runtime.GOOS)
}

// gateLoadValuePattern is the shape of one load average value the parsers
// take: digits, and a fraction after a point.
var gateLoadValuePattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// parseSysctlLoadavg reads the output of `sysctl -n vm.loadavg` on darwin,
// such as `{ 2.45 2.30 2.18 }`: the three values between braces.
func parseSysctlLoadavg(out string) (gateLoadAvg, error) {
	inner, ok := strings.CutPrefix(strings.TrimSpace(out), "{")
	if ok {
		inner, ok = strings.CutSuffix(inner, "}")
	}
	if !ok {
		return gateLoadAvg{}, fmt.Errorf("vm.loadavg %q is not { <1m> <5m> <15m> }", out)
	}
	l, err := parseGateLoadFields(strings.Fields(inner))
	if err != nil {
		return gateLoadAvg{}, fmt.Errorf("vm.loadavg %q: %w", out, err)
	}
	return l, nil
}

// parseProcLoadavg reads /proc/loadavg on linux, such as `0.52 0.58 0.59
// 1/1234 5678`: five fields, the first three the load average.
func parseProcLoadavg(data string) (gateLoadAvg, error) {
	fields := strings.Fields(data)
	if len(fields) != 5 {
		return gateLoadAvg{}, fmt.Errorf("/proc/loadavg %q has %d fields, not 5", data, len(fields))
	}
	l, err := parseGateLoadFields(fields[:3])
	if err != nil {
		return gateLoadAvg{}, fmt.Errorf("/proc/loadavg %q: %w", data, err)
	}
	return l, nil
}

// parseGateLoadFields reads the 1-, 5-, and 15-minute values from exactly
// three fields, each of gateLoadValuePattern's shape.
func parseGateLoadFields(fields []string) (gateLoadAvg, error) {
	if len(fields) != 3 {
		return gateLoadAvg{}, fmt.Errorf("%d values, not 3", len(fields))
	}
	var v [3]float64
	for i, f := range fields {
		if !gateLoadValuePattern.MatchString(f) {
			return gateLoadAvg{}, fmt.Errorf("%q is not a load average", f)
		}
		n, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return gateLoadAvg{}, fmt.Errorf("%q is not a load average: %w", f, err)
		}
		v[i] = n
	}
	return gateLoadAvg{one: v[0], five: v[1], fifteen: v[2]}, nil
}

// formatGateLoad is a load average value as the log, the record, and the CLI
// give it: two decimals, such as 16.24.
func formatGateLoad(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// gateCPUs says how many CPUs n is, for the log: "8 CPUs", or that the count
// is unknown when n is 0 or less.
func gateCPUs(n int) string {
	switch {
	case n <= 0:
		return "an unknown number of CPUs"
	case n == 1:
		return "1 CPU"
	}
	return strconv.Itoa(n) + " CPUs"
}

// GateLoad is the machine's load average around the gate scripts that one
// Gate or Regate ran (plan 2026-11-12-gate-lock-accept, AC5 and AC6), as the
// CLI prints it (`LOAD:`) and the gate record holds it.
type GateLoad struct {
	// Start, Highest, and End are 1-minute load averages with two decimals,
	// or "unknown" when none could be read: right before the first script,
	// the highest read while the scripts ran (Start and End included), and
	// right after the last script that ran.
	Start, Highest, End string
	// NCPU is the machine's CPU count; 0 when it is not known.
	NCPU int
	// MaxLoad is the [org.gate].max_load the gate ran with; 0 when it is off.
	MaxLoad int
	// Waited is how long the gate waited, after it took gate.lock, for the
	// 1-minute load to come down to MaxLoad: 0 when it did not wait.
	Waited time.Duration
}

// gateLoadSampler reads the 1-minute load every interval while the gate
// scripts run and keeps the highest (D4).
//
// Concurrency: runGateScripts starts it (startGateRunLoad, with
// startGateLoadSampler) once it has read the load before the first script,
// which the sampler starts from, and stops it (gateRunLoad.finish) after the
// last script that ran; it defers a stop too, so the sampler is stopped on
// every return, a script interrupted by ctx and a log that cannot be written
// included. The sampler's one goroutine is the
// only one that touches highest and known until it exits; stop closes done
// (once), waits for the goroutine to exit (exited), and only then reads them,
// so no lock is needed. The goroutine also exits when ctx is done. stop may be
// called again, or after ctx is done; it returns the same values. A read the
// goroutine has started when stop is called runs to its end first (at most
// gateLoadReadTimeout with sysctl).
type gateLoadSampler struct {
	done     chan struct{}
	exited   chan struct{}
	stopOnce sync.Once
	highest  float64
	known    bool
}

// startGateLoadSampler starts a sampler that reads the load with read every
// interval (more than 0) until it is stopped or ctx is done, starting from
// start when startOK.
func startGateLoadSampler(ctx context.Context, read gateLoadReadFunc, interval time.Duration, start gateLoadAvg, startOK bool) *gateLoadSampler {
	s := &gateLoadSampler{done: make(chan struct{}), exited: make(chan struct{}), highest: start.one, known: startOK}
	go func() {
		defer close(s.exited)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			if l, err := read(ctx); err == nil && (!s.known || l.one > s.highest) {
				s.highest, s.known = l.one, true
			}
		}
	}()
	return s
}

// stop ends the sampler's goroutine, waits for it, and returns the highest
// 1-minute load it saw, the start included; known is false when no reading
// succeeded.
func (s *gateLoadSampler) stop() (highest float64, known bool) {
	s.stopOnce.Do(func() { close(s.done) })
	<-s.exited
	return s.highest, s.known
}

// gateRunLoad is the load average around one runGateScripts: the reading
// before the first script, the CPU count, and the sampler that runs while the
// scripts do (startGateRunLoad, finish).
type gateRunLoad struct {
	start    gateLoadAvg
	startErr error
	ncpu     int
	sampler  *gateLoadSampler
}

// startGateRunLoad reads the load with gateLoadReader and the CPU count with
// gateNumCPU, starts a sampler from that reading (every
// gateLoadSampleInterval), and returns them with the log's `# load at start:`
// line: the three values and the CPU count, or unknown and why. The caller
// stops the sampler (finish, and a deferred sampler.stop).
func startGateRunLoad(ctx context.Context) (*gateRunLoad, string) {
	l := &gateRunLoad{ncpu: gateNumCPU()}
	l.start, l.startErr = gateLoadReader(ctx)
	l.sampler = startGateLoadSampler(ctx, gateLoadReader, gateLoadSampleInterval, l.start, l.startErr == nil)
	if l.startErr != nil {
		return l, fmt.Sprintf("# load at start: %s (%v); %s", gateLoadUnknown, l.startErr, gateCPUs(l.ncpu))
	}
	return l, fmt.Sprintf("# load at start: %s %s %s (%s)", formatGateLoad(l.start.one), formatGateLoad(l.start.five),
		formatGateLoad(l.start.fifteen), gateCPUs(l.ncpu))
}

// finish stops l's sampler, reads the load once more, and returns the
// GateLoad of the run, with w's limit and how long the gate waited for it,
// and the log's `# load at end:` line: the three values then (or unknown and
// why) and the highest 1-minute value read, the start and the end included.
func (l *gateRunLoad) finish(ctx context.Context, w gateMaxLoadWait) (*GateLoad, string) {
	highest, known := l.sampler.stop()
	end, endErr := gateLoadReader(ctx)
	if endErr == nil && (!known || end.one > highest) {
		highest, known = end.one, true
	}
	load := &GateLoad{Start: gateLoadUnknown, Highest: gateLoadUnknown, End: gateLoadUnknown, NCPU: max(l.ncpu, 0),
		MaxLoad: max(w.maxLoad, 0)}
	if w.didWait {
		load.Waited = w.waited
	}
	if l.startErr == nil {
		load.Start = formatGateLoad(l.start.one)
	}
	if known {
		load.Highest = formatGateLoad(highest)
	}
	endValues := fmt.Sprintf("%s (%v)", gateLoadUnknown, endErr)
	if endErr == nil {
		load.End = formatGateLoad(end.one)
		endValues = formatGateLoad(end.one) + " " + formatGateLoad(end.five) + " " + formatGateLoad(end.fifteen)
	}
	return load, fmt.Sprintf("# load at end: %s; highest 1-minute load while the scripts ran: %s", endValues, load.Highest)
}

// gateMaxLoadWait is what a gate's wait for [org.gate].max_load was
// (waitGateMaxLoad).
type gateMaxLoadWait struct {
	// maxLoad is [org.gate].max_load; 0 or less when the wait is off.
	maxLoad int
	// didWait is set when the 1-minute load was above maxLoad, so the gate
	// waited; waited is how long it waited.
	didWait bool
	waited  time.Duration
	// stillAbove: after gateMaxLoadCap the 1-minute load was still above
	// maxLoad, so the gate started anyway.
	stillAbove bool
	// unreadable is why the load could not be read, when that kept the gate
	// from waiting or ended its wait.
	unreadable string
}

// logLines are the lines the gate log's header gets for w (gateRunOptions):
// one line when maxLoad is above 0, none otherwise.
func (w gateMaxLoadWait) logLines() []string {
	if w.maxLoad <= 0 {
		return nil
	}
	var what string
	switch {
	case !w.didWait && w.unreadable != "":
		what = "did not wait: the load average could not be read (" + w.unreadable + ")"
	case !w.didWait:
		what = "did not wait: the 1-minute load average was not above it"
	case w.stillAbove:
		what = "waited " + gateLockDuration(w.waited) + "; started with the 1-minute load average still above it"
	case w.unreadable != "":
		what = "waited " + gateLockDuration(w.waited) + "; started when the load average could not be read (" + w.unreadable + ")"
	default:
		what = "waited " + gateLockDuration(w.waited) + " for the 1-minute load average to come down to it"
	}
	return []string{fmt.Sprintf("# max_load: %d; %s", w.maxLoad, what)}
}

// waitGateMaxLoad waits, with gate.lock held, while the 1-minute load average
// read with read is above maxLoad ([org.gate].max_load, D2). With maxLoad 0 or
// less it returns at once and reads nothing. A load that cannot be read
// starts the gate at once, with one line on stderr. While the 1-minute load
// is above maxLoad, it says so on stderr once, sleeps gateMaxLoadPoll, and
// reads it again; it returns once the load is not above maxLoad, or it cannot
// be read, saying so on stderr, or, after gateMaxLoadCap with the load still
// above maxLoad, with a line that the gate starts anyway. Each line starts
// with prefix ("org: gate: "). When ctx is done, it returns an error that
// wraps ctx's cause.
func waitGateMaxLoad(ctx context.Context, read gateLoadReadFunc, maxLoad int, stderr io.Writer, prefix string) (gateMaxLoadWait, error) {
	w := gateMaxLoadWait{maxLoad: maxLoad}
	if maxLoad <= 0 {
		return w, nil
	}
	interrupted := func() (gateMaxLoadWait, error) {
		return gateMaxLoadWait{}, fmt.Errorf("%swait for the 1-minute load average to come down to [org.gate].max_load %d: %w",
			prefix, maxLoad, context.Cause(ctx))
	}
	l, err := read(ctx)
	switch {
	case ctx.Err() != nil:
		return interrupted()
	case err != nil:
		w.unreadable = err.Error()
		_, _ = fmt.Fprintf(stderr, "%s[org.gate].max_load is %d, but the load average cannot be read (%s): starting without "+
			"waiting\n", prefix, maxLoad, w.unreadable)
		return w, nil
	case l.one <= float64(maxLoad):
		return w, nil
	}
	_, _ = fmt.Fprintf(stderr, "%sthe 1-minute load average is %s, above [org.gate].max_load %d: waiting for it to come down, "+
		"reading it again every %s for up to %s\n", prefix, formatGateLoad(l.one), maxLoad, gateMaxLoadPoll, gateMaxLoadCap)
	w.didWait = true
	begin := time.Now()
	for {
		timer := time.NewTimer(min(gateMaxLoadPoll, max(gateMaxLoadCap-time.Since(begin), 0)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return interrupted()
		case <-timer.C:
		}
		l, err = read(ctx)
		w.waited = time.Since(begin)
		switch {
		case ctx.Err() != nil:
			return interrupted()
		case err != nil:
			w.unreadable = err.Error()
			_, _ = fmt.Fprintf(stderr, "%sthe load average cannot be read now (%s), after waiting %s for it to come down to "+
				"[org.gate].max_load %d: starting\n", prefix, w.unreadable, gateLockDuration(w.waited), maxLoad)
			return w, nil
		case l.one <= float64(maxLoad):
			_, _ = fmt.Fprintf(stderr, "%sthe 1-minute load average is %s, not above [org.gate].max_load %d, after waiting %s: "+
				"starting\n", prefix, formatGateLoad(l.one), maxLoad, gateLockDuration(w.waited))
			return w, nil
		case w.waited >= gateMaxLoadCap:
			w.stillAbove = true
			_, _ = fmt.Fprintf(stderr, "%sthe 1-minute load average is still %s, above [org.gate].max_load %d, after waiting %s: "+
				"starting anyway\n", prefix, formatGateLoad(l.one), maxLoad, gateLockDuration(w.waited))
			return w, nil
		}
	}
}
