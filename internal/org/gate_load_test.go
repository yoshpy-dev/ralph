package org

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The load average around the gate scripts and [org.gate].max_load (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, AC5 and AC6, D2 and D4):
// the two parsers, the sampler, the wait, the record's keys, and Gate and
// Regate with them, on the fixtures of gate_test.go and regate_test.go.

// loadSeq is a gateLoadReadFunc that returns its items in order, the last one
// again once they run out, and counts its calls. onCall, when set, runs with
// the number of each call (1 for the first) before it returns.
type loadSeq struct {
	mu     sync.Mutex
	items  []loadSeqItem
	calls  int
	onCall func(n int)
}

// loadSeqItem is one answer of a loadSeq: a load average, or err.
type loadSeqItem struct {
	l   gateLoadAvg
	err error
}

// loadOne is a loadSeqItem whose 1-minute value is one, and the 5- and
// 15-minute values one-half and one-quarter of it.
func loadOne(one float64) loadSeqItem {
	return loadSeqItem{l: gateLoadAvg{one: one, five: one / 2, fifteen: one / 4}}
}

func (s *loadSeq) read(context.Context) (gateLoadAvg, error) {
	s.mu.Lock()
	i := min(s.calls, len(s.items)-1)
	s.calls++
	n, onCall := s.calls, s.onCall
	s.mu.Unlock()
	if onCall != nil {
		onCall(n)
	}
	return s.items[i].l, s.items[i].err
}

func (s *loadSeq) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// gateLoadWaitedPattern matches how long a max_load wait took in its lines
// (gateLockDuration: 0s, 1s, 1m0s), which a slow machine can move.
var gateLoadWaitedPattern = regexp.MustCompile(`\b(waited|waiting) [0-9][0-9hms]*`)

// normalizeLoadWaits replaces each duration gateLoadWaitedPattern matches in
// s with <d>.
func normalizeLoadWaits(s string) string {
	return gateLoadWaitedPattern.ReplaceAllString(s, "$1 <d>")
}

// setGateLoadTimes sets the sampler's interval and the max_load wait's poll
// and cap until the test ends.
func setGateLoadTimes(t *testing.T, sample, poll, limit time.Duration) {
	t.Helper()
	oldSample, oldPoll, oldCap := gateLoadSampleInterval, gateMaxLoadPoll, gateMaxLoadCap
	t.Cleanup(func() { gateLoadSampleInterval, gateMaxLoadPoll, gateMaxLoadCap = oldSample, oldPoll, oldCap })
	gateLoadSampleInterval, gateMaxLoadPoll, gateMaxLoadCap = sample, poll, limit
}

func TestParseSysctlLoadavg(t *testing.T) {
	for _, tc := range []struct {
		name, out string
		want      gateLoadAvg
		wantErr   string // "" means it must read
	}{
		{"darwin's output", "{ 2.45 2.30 2.18 }\n", gateLoadAvg{2.45, 2.30, 2.18}, ""},
		{"no space inside the braces", "{2.45 2.30 2.18}", gateLoadAvg{2.45, 2.30, 2.18}, ""},
		{"whole numbers", "{ 16 0 3 }", gateLoadAvg{16, 0, 3}, ""},
		{"empty", "", gateLoadAvg{}, "is not { <1m> <5m> <15m> }"},
		{"no braces", "2.45 2.30 2.18", gateLoadAvg{}, "is not { <1m> <5m> <15m> }"},
		{"no closing brace", "{ 2.45 2.30 2.18", gateLoadAvg{}, "is not { <1m> <5m> <15m> }"},
		{"two values", "{ 2.45 2.30 }", gateLoadAvg{}, "2 values, not 3"},
		{"four values", "{ 2.45 2.30 2.18 1.00 }", gateLoadAvg{}, "4 values, not 3"},
		{"a word", "{ high 2.30 2.18 }", gateLoadAvg{}, `"high" is not a load average`},
		{"a negative value", "{ -1.00 2.30 2.18 }", gateLoadAvg{}, `"-1.00" is not a load average`},
		{"a decimal comma", "{ 2,45 2,30 2,18 }", gateLoadAvg{}, `"2,45" is not a load average`},
		{"NaN", "{ NaN 2.30 2.18 }", gateLoadAvg{}, `"NaN" is not a load average`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlLoadavg(tc.out)
			if tc.wantErr == "" {
				if err != nil || got != tc.want {
					t.Fatalf("parseSysctlLoadavg(%q) = %+v, %v; want %+v", tc.out, got, err, tc.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || got != (gateLoadAvg{}) {
				t.Fatalf("parseSysctlLoadavg(%q) = %+v, %v; want an error containing %q", tc.out, got, err, tc.wantErr)
			}
		})
	}
}

func TestParseProcLoadavg(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		want       gateLoadAvg
		wantErr    string // "" means it must read
	}{
		{"linux's file", "0.52 0.58 0.59 1/1234 5678\n", gateLoadAvg{0.52, 0.58, 0.59}, ""},
		{"a busy machine", "157.03 80.10 40.00 90/2000 1", gateLoadAvg{157.03, 80.10, 40}, ""},
		{"empty", "", gateLoadAvg{}, "has 0 fields, not 5"},
		{"three fields", "0.52 0.58 0.59", gateLoadAvg{}, "has 3 fields, not 5"},
		{"four fields", "0.52 0.58 0.59 1/1234", gateLoadAvg{}, "has 4 fields, not 5"},
		{"six fields", "0.52 0.58 0.59 1/1234 5678 9", gateLoadAvg{}, "has 6 fields, not 5"},
		{"a word", "x 0.58 0.59 1/1234 5678", gateLoadAvg{}, `"x" is not a load average`},
		{"an exponent", "1e3 0.58 0.59 1/1234 5678", gateLoadAvg{}, `"1e3" is not a load average`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseProcLoadavg(tc.data)
			if tc.wantErr == "" {
				if err != nil || got != tc.want {
					t.Fatalf("parseProcLoadavg(%q) = %+v, %v; want %+v", tc.data, got, err, tc.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || got != (gateLoadAvg{}) {
				t.Fatalf("parseProcLoadavg(%q) = %+v, %v; want an error containing %q", tc.data, got, err, tc.wantErr)
			}
		})
	}
}

// TestReadGateLoad_ReadsThisMachine runs the real reader (TestMain pins
// gateLoadReader, not readGateLoad): on darwin and linux it reads three
// values of zero or more; elsewhere it is an error that names the OS.
func TestReadGateLoad_ReadsThisMachine(t *testing.T) {
	l, err := readGateLoad(context.Background())
	switch runtime.GOOS {
	case "darwin", "linux":
		if err != nil || l.one < 0 || l.five < 0 || l.fifteen < 0 {
			t.Fatalf("readGateLoad = %+v, %v; want three values of zero or more", l, err)
		}
	default:
		if err == nil || !strings.Contains(err.Error(), runtime.GOOS) {
			t.Fatalf("readGateLoad on %s = %+v, %v; want an error naming the OS", runtime.GOOS, l, err)
		}
	}
}

func TestFormatGateLoad(t *testing.T) {
	for v, want := range map[float64]string{0: "0.00", 2.5: "2.50", 16.24: "16.24", 157: "157.00", 0.005: "0.01"} {
		if got := formatGateLoad(v); got != want {
			t.Errorf("formatGateLoad(%v) = %q; want %q", v, got, want)
		}
	}
}

// waitForCalls waits until s has had at least n calls.
func waitForCalls(t *testing.T, s *loadSeq, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for s.count() < n {
		if time.Now().After(deadline) {
			t.Fatalf("the reader had %d calls in 10s; want %d", s.count(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestGateLoadSampler_KeepsTheHighestAndStops covers D4's sampler: it keeps
// the highest 1-minute value it read, the start's included, skips a read that
// failed, and reads no more once stop returned; a second stop returns the
// same values.
func TestGateLoadSampler_KeepsTheHighestAndStops(t *testing.T) {
	for _, tc := range []struct {
		name     string
		start    loadSeqItem
		reads    []loadSeqItem
		want     float64
		wantKnow bool
	}{
		{"a read above the start", loadOne(1), []loadSeqItem{loadOne(2), loadOne(5), {err: errors.New("boom")}, loadOne(3)}, 5, true},
		{"the start above every read", loadOne(9), []loadSeqItem{loadOne(2), loadOne(3)}, 9, true},
		{"no start, one read", loadSeqItem{err: errors.New("boom")}, []loadSeqItem{{err: errors.New("boom")}, loadOne(4)}, 4, true},
		{"nothing read", loadSeqItem{err: errors.New("boom")}, []loadSeqItem{{err: errors.New("boom")}}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seq := &loadSeq{items: tc.reads}
			s := startGateLoadSampler(context.Background(), seq.read, time.Millisecond, tc.start.l, tc.start.err == nil)
			waitForCalls(t, seq, len(tc.reads)+2)
			got, known := s.stop()
			if got != tc.want || known != tc.wantKnow {
				t.Fatalf("stop = %v, %v; want %v, %v", got, known, tc.want, tc.wantKnow)
			}
			calls := seq.count()
			select {
			case <-s.exited:
			default:
				t.Fatal("stop returned with the sampler's goroutine still running")
			}
			time.Sleep(20 * time.Millisecond)
			if again, knownAgain := s.stop(); again != got || knownAgain != known || seq.count() != calls {
				t.Fatalf("after stop: stop = %v, %v and %d calls; want %v, %v and %d calls", again, knownAgain, seq.count(), got, known, calls)
			}
		})
	}
}

// TestGateLoadSampler_EndsWithCtx: the goroutine ends when ctx is done, and a
// stop after that returns at once with what it had read.
func TestGateLoadSampler_EndsWithCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seq := &loadSeq{items: []loadSeqItem{loadOne(7)}}
	s := startGateLoadSampler(ctx, seq.read, time.Millisecond, gateLoadAvg{one: 1}, true)
	waitForCalls(t, seq, 1)
	cancel()
	select {
	case <-s.exited:
	case <-time.After(10 * time.Second):
		t.Fatal("the sampler's goroutine did not end within 10s of ctx done")
	}
	if got, known := s.stop(); got != 7 || !known {
		t.Fatalf("stop after ctx done = %v, %v; want 7, true", got, known)
	}
}

// TestWaitGateMaxLoad covers AC6's wait on its own: 0 or less reads nothing
// and does not wait; a load not above max_load does not wait; above, then not
// above, waits and starts; above for the whole (shortened) cap starts anyway;
// a load that cannot be read, at first or during the wait, starts at once.
// Each says so on stderr, and logLines gives the log's line.
func TestWaitGateMaxLoad(t *testing.T) {
	boom := loadSeqItem{err: errors.New("sysctl: boom")}
	for _, tc := range []struct {
		name       string
		maxLoad    int
		reads      []loadSeqItem // nil: the reader must not be called
		wantCalls  int
		wantWait   bool
		wantAbove  bool
		wantStderr []string // each in this order, and no other line
		wantLog    string   // the one log line; "" for none
	}{
		{name: "zero", maxLoad: 0},
		{name: "below zero", maxLoad: -3},
		{name: "below", maxLoad: 4, reads: []loadSeqItem{loadOne(3.99)}, wantCalls: 1,
			wantLog: "# max_load: 4; did not wait: the 1-minute load average was not above it"},
		{name: "equal", maxLoad: 4, reads: []loadSeqItem{loadOne(4)}, wantCalls: 1,
			wantLog: "# max_load: 4; did not wait: the 1-minute load average was not above it"},
		{name: "unreadable", maxLoad: 4, reads: []loadSeqItem{boom}, wantCalls: 1,
			wantStderr: []string{"org: gate: [org.gate].max_load is 4, but the load average cannot be read (sysctl: boom): starting without waiting"},
			wantLog:    "# max_load: 4; did not wait: the load average could not be read (sysctl: boom)"},
		{name: "above, then not above", maxLoad: 4, reads: []loadSeqItem{loadOne(6), loadOne(5), loadOne(3)}, wantCalls: 3, wantWait: true,
			wantStderr: []string{
				"org: gate: the 1-minute load average is 6.00, above [org.gate].max_load 4: waiting for it to come down, reading it again every 5ms for up to 1m0s",
				"org: gate: the 1-minute load average is 3.00, not above [org.gate].max_load 4, after waiting <d>: starting",
			},
			wantLog: "# max_load: 4; waited <d> for the 1-minute load average to come down to it"},
		{name: "above, then unreadable", maxLoad: 4, reads: []loadSeqItem{loadOne(6), boom}, wantCalls: 2, wantWait: true,
			wantStderr: []string{
				"org: gate: the 1-minute load average is 6.00, above [org.gate].max_load 4: waiting for it to come down, reading it again every 5ms for up to 1m0s",
				"org: gate: the load average cannot be read now (sysctl: boom), after waiting <d> for it to come down to [org.gate].max_load 4: starting",
			},
			wantLog: "# max_load: 4; waited <d>; started when the load average could not be read (sysctl: boom)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setGateLoadTimes(t, time.Hour, 5*time.Millisecond, time.Minute)
			seq := &loadSeq{items: tc.reads}
			read := seq.read
			if tc.reads == nil {
				read = func(context.Context) (gateLoadAvg, error) {
					t.Error("the load was read with max_load at or below 0")
					return gateLoadAvg{}, nil
				}
			}
			var stderr strings.Builder
			w, err := waitGateMaxLoad(context.Background(), read, tc.maxLoad, &stderr, "org: gate: ")
			if err != nil {
				t.Fatalf("waitGateMaxLoad: %v", err)
			}
			if seq.count() != tc.wantCalls || w.didWait != tc.wantWait || w.stillAbove != tc.wantAbove {
				t.Errorf("waitGateMaxLoad = %+v after %d reads; want didWait %t, stillAbove %t after %d reads", w, seq.count(),
					tc.wantWait, tc.wantAbove, tc.wantCalls)
			}
			if got := strings.Split(normalizeLoadWaits(strings.TrimSuffix(stderr.String(), "\n")), "\n"); stderr.Len() > 0 || len(tc.wantStderr) > 0 {
				if !slices.Equal(got, tc.wantStderr) {
					t.Errorf("stderr lines = %q\nwant %q", got, tc.wantStderr)
				}
			}
			var wantLog []string
			if tc.wantLog != "" {
				wantLog = []string{tc.wantLog}
			}
			got := w.logLines()
			for i := range got {
				got[i] = normalizeLoadWaits(got[i])
			}
			if !slices.Equal(got, wantLog) {
				t.Errorf("logLines = %q; want %q", got, wantLog)
			}
		})
	}
}

// TestWaitGateMaxLoad_StartsAnywayAfterTheCap covers D2's cap: with the
// 1-minute load above max_load at every read, the wait reads it again every
// poll and starts after the (shortened) cap, saying so on stderr and in the
// log's line.
func TestWaitGateMaxLoad_StartsAnywayAfterTheCap(t *testing.T) {
	setGateLoadTimes(t, time.Hour, 5*time.Millisecond, 60*time.Millisecond)
	seq := &loadSeq{items: []loadSeqItem{loadOne(30.126)}}
	var stderr strings.Builder
	begin := time.Now()
	w, err := waitGateMaxLoad(context.Background(), seq.read, 24, &stderr, "org: regate: ")
	if err != nil {
		t.Fatalf("waitGateMaxLoad: %v", err)
	}
	if took := time.Since(begin); !w.didWait || !w.stillAbove || w.waited < gateMaxLoadCap || took < gateMaxLoadCap {
		t.Fatalf("waitGateMaxLoad = %+v after %s; want didWait and stillAbove after at least %s", w, took, gateMaxLoadCap)
	}
	if seq.count() < 3 {
		t.Errorf("the load was read %d times in the cap of %s with a poll of %s; want it read again on each poll", seq.count(),
			gateMaxLoadCap, gateMaxLoadPoll)
	}
	lines := strings.Split(normalizeLoadWaits(strings.TrimSuffix(stderr.String(), "\n")), "\n")
	wantFirst := "org: regate: the 1-minute load average is 30.13, above [org.gate].max_load 24: waiting for it to come down, " +
		"reading it again every 5ms for up to 60ms"
	if len(lines) != 2 || lines[0] != wantFirst ||
		lines[1] != "org: regate: the 1-minute load average is still 30.13, above [org.gate].max_load 24, after waiting <d>: starting anyway" {
		t.Fatalf("stderr lines = %q; want the start of the wait and that it starts anyway", lines)
	}
	if got := w.logLines(); len(got) != 1 ||
		normalizeLoadWaits(got[0]) != "# max_load: 24; waited <d>; started with the 1-minute load average still above it" {
		t.Fatalf("logLines = %q", got)
	}
}

// TestWaitGateMaxLoad_CtxEndsTheWait: ctx done while the wait sleeps ends it
// with an error that wraps ctx's cause and names max_load.
func TestWaitGateMaxLoad_CtxEndsTheWait(t *testing.T) {
	setGateLoadTimes(t, time.Hour, time.Hour, 2*time.Hour)
	cause := errors.New("interrupt signal received")
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(nil) })
	seq := &loadSeq{items: []loadSeqItem{loadOne(9)}}
	var stderr lockedBuffer
	done := make(chan error, 1)
	go func() {
		_, err := waitGateMaxLoad(ctx, seq.read, 2, &stderr, "org: gate: ")
		done <- err
	}()
	waitForStderr(t, &stderr, "waiting for it to come down")
	cancel(cause)
	select {
	case err := <-done:
		if !errors.Is(err, cause) || !strings.HasPrefix(err.Error(), "org: gate: wait for the 1-minute load average to come down to [org.gate].max_load 2: ") {
			t.Fatalf("waitGateMaxLoad = %v; want an error naming max_load that wraps the cause", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("waitGateMaxLoad did not return within 10s of ctx done")
	}
	if seq.count() != 1 {
		t.Errorf("the load was read %d times; want once, before the sleep", seq.count())
	}
}

// TestGateRecordDetails_Load covers AC5's keys in the Details: written after
// herdr_pane= and via= and before log=, max_load= only above 0, an empty
// value written as unknown; read back as written; and a Details of the shape
// before them (no load keys) still reads, with the load fields empty.
func TestGateRecordDetails_Load(t *testing.T) {
	sha1, sha1b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	base := gateRecord{Head: sha1, Base: sha1b, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", By: "alice",
		Log: "/state/gates/auth 1.log"}
	const fields = " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset"
	log := " log=" + strconv.Quote(base.Log)

	r := base
	r.LoadStart, r.LoadMax, r.LoadEnd, r.NCPU = "16.24", "157.03", "80.10", "12"
	if got, want := r.details(), "head="+sha1+" base="+sha1b+fields+" load_start=16.24 load_max=157.03 load_end=80.10 ncpu=12"+log; got != want {
		t.Errorf("details = %q\nwant      %q", got, want)
	}
	withMax := r
	withMax.Via, withMax.Reason, withMax.MaxLoad = gateViaRegate, GateReasonTestFailed, 24
	if got, want := withMax.details(), "reason=test_failed head="+sha1+" base="+sha1b+fields+
		" via=regate load_start=16.24 load_max=157.03 load_end=80.10 ncpu=12 max_load=24"+log; got != want {
		t.Errorf("details = %q\nwant      %q", got, want)
	}
	unknown := base
	unknown.LoadStart, unknown.LoadEnd = gateLoadUnknown, "1.00"
	wantUnknown := unknown
	wantUnknown.LoadMax, wantUnknown.NCPU = gateLoadUnknown, gateLoadUnknown
	if got, want := unknown.details(), "head="+sha1+" base="+sha1b+fields+" load_start=unknown load_max=unknown load_end=1.00 ncpu=unknown"+log; got != want {
		t.Errorf("details = %q\nwant      %q", got, want)
	}

	for _, tc := range []struct {
		name string
		in   gateRecord
		want gateRecord
	}{
		{"load keys", r, r},
		{"load keys, via and max_load", withMax, withMax},
		{"unknown values", unknown, wantUnknown},
		{"the shape before the load keys", base, base},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := gateRecordFromDetails(tc.in.details())
			if !ok || got != tc.want {
				t.Fatalf("details %q read back as %+v (%v); want %+v", tc.in.details(), got, ok, tc.want)
			}
		})
	}

	// max_load reads best-effort: a value that is not a number above 0
	// reads as 0, and the record still reads.
	for _, bad := range []string{"0", "-4", "x", "", "1.5"} {
		d := strings.Replace(withMax.details(), " max_load=24 ", " max_load="+bad+" ", 1)
		got, ok := gateRecordFromDetails(d)
		if want := withMax; !ok || got.MaxLoad != 0 || got.LoadStart != want.LoadStart {
			t.Errorf("details %q read back as %+v (%v); want max_load 0 and the rest", d, got, ok)
		}
	}
	// A load key given twice makes the record unreadable, as any key before
	// log= does.
	twice := strings.Replace(r.details(), " ncpu=12", " ncpu=12 ncpu=12", 1)
	if got, ok := gateRecordFromDetails(twice); ok {
		t.Errorf("details %q with ncpu twice read back as %+v; want it unreadable", twice, got)
	}
}

// gateLoadReaderForGate sets gateLoadReader, until the test ends, to a reader
// whose first call (the start) is 2.00/1.00/0.50, whose second (the
// sampler's first tick) is 9.00 and writes out/sampled, and whose later
// calls (more ticks, and the end) are 3.00/1.50/0.75, with 4 CPUs.
func gateLoadReaderForGate(t *testing.T, out string) *loadSeq {
	t.Helper()
	seq := &loadSeq{items: []loadSeqItem{loadOne(2), loadOne(9), loadOne(3)}}
	seq.onCall = func(n int) {
		if n == 2 {
			// The stubs wait for this file (gateLoadHook), so the scripts run
			// until the sampler has read the load once.
			writeGateFile(t, out, "sampled", "", 0o644)
		}
	}
	setGateLoadReader(t, seq.read, 4)
	return seq
}

// gateLoadHook makes a stub wait until the sampler read the load once
// (gateLoadReaderForGate).
const gateLoadHook = `while [ ! -f "$GATE_TEST_OUT/sampled" ]; do sleep 0.01; done
`

// TestGate_RecordsTheLoadAroundTheScripts covers AC5 on Gate: the log's
// header has the start's three values and the CPU count, the log's end the
// end's three values and the highest 1-minute value the sampler read while
// the scripts ran; GateResult.Load and the gate_passed's Details hold them.
func TestGate_RecordsTheLoadAroundTheScripts(t *testing.T) {
	setGateLoadTimes(t, 5*time.Millisecond, time.Hour, time.Hour)
	fx := newGateFixture(t)
	gateLoadReaderForGate(t, fx.out)
	fx.hook(t, "static", gateLoadHook)
	res := fx.gate(t)

	want := &GateLoad{Start: "2.00", Highest: "9.00", End: "3.00", NCPU: 4}
	if !res.Passed || res.Load == nil || *res.Load != *want {
		t.Fatalf("gate = %+v (Load %+v); want a pass with Load %+v", res, res.Load, want)
	}
	events := fx.gateEvents(t)
	if len(events) != 1 || !strings.Contains(events[0].Details, " herdr_pane=unset load_start=2.00 load_max=9.00 load_end=3.00 ncpu=4 log=") {
		t.Fatalf("gate events = %+v; want one gate_passed with the load keys before log=", events)
	}
	if r, ok := gateRecordFromDetails(events[0].Details); !ok || r.LoadStart != "2.00" || r.LoadMax != "9.00" || r.LoadEnd != "3.00" ||
		r.NCPU != "4" || r.MaxLoad != 0 {
		t.Fatalf("record %q read as %+v (%v)", events[0].Details, r, ok)
	}
	log := fileOrAbsent(t, res.LogPath)
	head, base := fx.head(t)
	for _, w := range []string{
		"# head: " + head + "\n# base: refs/remotes/origin/main " + base + "\n# load at start: 2.00 1.00 0.50 (4 CPUs)\n\n==> ralph org gate: running ",
		"==> ralph org gate: ./scripts/run-test.sh exited 0\n# load at end: 3.00 1.50 0.75; highest 1-minute load while the scripts ran: 9.00\n",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log does not have %q:\n%s", w, log)
		}
	}
	if strings.Contains(log, "# max_load:") {
		t.Errorf("log has a max_load line with max_load 0:\n%s", log)
	}
}

// TestGate_RecordsAnUnknownLoad covers AC5's unknown: a load that cannot be
// read is unknown in the log, the record, and GateResult.Load, with the CPU
// count; the gate runs and records as before.
func TestGate_RecordsAnUnknownLoad(t *testing.T) {
	fx := newGateFixture(t)
	setGateLoadReader(t, (&loadSeq{items: []loadSeqItem{{err: errors.New("sysctl: boom")}}}).read, 1)
	fx.setExit(t, "static", 2)
	res := fx.gate(t)

	want := &GateLoad{Start: gateLoadUnknown, Highest: gateLoadUnknown, End: gateLoadUnknown, NCPU: 1}
	if res.Reason != GateReasonStaticFailed || res.Load == nil || *res.Load != *want {
		t.Fatalf("gate = %+v (Load %+v); want static_failed with Load %+v", res, res.Load, want)
	}
	if events := fx.gateEvents(t); len(events) != 1 ||
		!strings.Contains(events[0].Details, " herdr_pane=unset load_start=unknown load_max=unknown load_end=unknown ncpu=1 log=") {
		t.Fatalf("gate events = %+v; want one gate_failed with unknown load keys", events)
	}
	log := fileOrAbsent(t, res.LogPath)
	for _, w := range []string{
		"\n# load at start: unknown (sysctl: boom); 1 CPU\n",
		"==> ralph org gate: ./scripts/run-static-verify.sh exited 2\n# load at end: unknown (sysctl: boom); highest 1-minute load while the scripts ran: unknown\n",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log does not have %q:\n%s", w, log)
		}
	}
	if strings.Contains(log, "running ./scripts/run-test.sh") {
		t.Errorf("the test script ran after static_failed:\n%s", log)
	}
}

// TestGate_MaxLoadStartsAnywayAfterTheCap covers AC6 on Gate: with the
// 1-minute load above max_load for the whole (shortened) cap, the gate says
// on stderr that it waits and that it starts anyway, runs the scripts, writes
// the wait in its log's header, and records max_load= with the load keys.
func TestGate_MaxLoadStartsAnywayAfterTheCap(t *testing.T) {
	setGateLoadTimes(t, time.Hour, 5*time.Millisecond, 50*time.Millisecond)
	fx := newGateFixture(t)
	setGateLoadReader(t, (&loadSeq{items: []loadSeqItem{loadOne(5)}}).read, 2)
	var stderr lockedBuffer
	res, err := fx.o.Gate(context.Background(), GateParams{OrgID: "org-a", Dir: fx.worktree, Env: fx.env, Stderr: &stderr, MaxLoad: 2})
	if err != nil || !res.Passed || res.Load == nil || res.Load.MaxLoad != 2 || res.Load.Waited < 50*time.Millisecond ||
		res.Load.Start != "5.00" {
		t.Fatalf("gate = %+v (Load %+v), %v; want a pass with max_load 2 and a wait of the cap", res, res.Load, err)
	}
	for _, w := range []string{
		"org: gate: the 1-minute load average is 5.00, above [org.gate].max_load 2: waiting for it to come down",
		"org: gate: the 1-minute load average is still 5.00, above [org.gate].max_load 2, after waiting <d>: starting anyway\n",
	} {
		if !strings.Contains(normalizeLoadWaits(stderr.String()), w) {
			t.Errorf("stderr does not have %q:\n%s", w, stderr.String())
		}
	}
	if events := fx.gateEvents(t); len(events) != 1 ||
		!strings.Contains(events[0].Details, " load_start=5.00 load_max=5.00 load_end=5.00 ncpu=2 max_load=2 log=") {
		t.Fatalf("gate events = %+v; want one gate_passed with max_load=2 after the load keys", events)
	}
	if log := fileOrAbsent(t, res.LogPath); !strings.Contains(normalizeLoadWaits(log),
		"\n# max_load: 2; waited <d>; started with the 1-minute load average still above it\n# load at start: 5.00 2.50 1.25 (2 CPUs)\n") {
		t.Errorf("log does not have the max_load line before the load at start:\n%s", log)
	}
}

// TestGate_MaxLoadNotAboveDoesNotWait: with the 1-minute load not above
// max_load, the gate does not wait, says nothing on stderr, and records
// max_load= all the same.
func TestGate_MaxLoadNotAboveDoesNotWait(t *testing.T) {
	setGateLoadTimes(t, time.Hour, time.Hour, time.Hour)
	fx := newGateFixture(t)
	var stderr lockedBuffer
	res, err := fx.o.Gate(context.Background(), GateParams{OrgID: "org-a", Dir: fx.worktree, Env: fx.env, Stderr: &stderr, MaxLoad: 2})
	want := testGateLoad()
	want.MaxLoad = 2
	if err != nil || !res.Passed || res.Load == nil || *res.Load != *want {
		t.Fatalf("gate = %+v (Load %+v), %v; want a pass with Load %+v", res, res.Load, err, want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr = %q; want nothing", stderr.String())
	}
	if events := fx.gateEvents(t); len(events) != 1 || !strings.Contains(events[0].Details, testGateLoadDetails+" max_load=2 log=") {
		t.Fatalf("gate events = %+v; want one gate_passed with max_load=2", events)
	}
	if log := fileOrAbsent(t, res.LogPath); !strings.Contains(log, "\n# max_load: 2; did not wait: the 1-minute load average was not above it\n") {
		t.Errorf("log does not have the max_load line:\n%s", log)
	}
}

// TestGate_MaxLoadWaitInterrupted covers AC6's ctx: an interrupt while the
// gate waits for the load ends it with nothing recorded, no log, no script
// run, and gate.lock released.
func TestGate_MaxLoadWaitInterrupted(t *testing.T) {
	setGateLoadTimes(t, time.Hour, time.Hour, 2*time.Hour)
	fx := newGateFixture(t)
	setGateLoadReader(t, (&loadSeq{items: []loadSeqItem{loadOne(5)}}).read, 2)
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(nil) })
	var stderr lockedBuffer
	done := make(chan gateOutcome, 1)
	go func() {
		res, err := fx.o.Gate(ctx, GateParams{OrgID: "org-a", Dir: fx.worktree, Env: fx.env, Stderr: &stderr, MaxLoad: 2})
		done <- gateOutcome{res, err}
	}()
	waitForStderr(t, &stderr, "above [org.gate].max_load 2: waiting for it to come down")
	held, heldErr := fileLockHeld(gateLockPathIn(fx.stateDir))
	cancel(errors.New("interrupt signal received"))
	got := waitGate(t, done, 10*time.Second)
	if heldErr != nil || !held {
		t.Errorf("gate.lock while the gate waited for the load: held %t, %v; want held", held, heldErr)
	}
	if got.err == nil || !strings.Contains(got.err.Error(), "org: gate: interrupted (interrupt signal received) before the scripts ran: nothing was recorded") {
		t.Fatalf("gate = %+v, %v; want the interrupt before the scripts", got.res, got.err)
	}
	if got.res.Load != nil || got.res.LogPath != "" {
		t.Errorf("gate = %+v; want no load and no log", got.res)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Errorf("the interrupted gate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the interrupted gate made %s (stat: %v); want no log", gateLogDir, err)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("the interrupted gate ran %v; want no script run", calls)
	}
	if held, err := fileLockHeld(gateLockPathIn(fx.stateDir)); err != nil || held {
		t.Errorf("gate.lock after the interrupted gate: held %t, %v; want free", held, err)
	}
}

// TestRegate_MaxLoadWaitsBeforeRegateLock covers AC6 with D1 on a local
// regate: it waits for the load holding gate.lock but not regate.lock, so a
// merge or a cleanup is not refused meanwhile; once the load comes down it
// runs the scripts and records max_load= and the load keys after via=regate.
func TestRegate_MaxLoadWaitsBeforeRegateLock(t *testing.T) {
	setGateLoadTimes(t, time.Hour, 10*time.Millisecond, time.Hour)
	fx := newRegateFixture(t)
	release := make(chan struct{})
	seq := &loadSeq{items: []loadSeqItem{loadOne(9), loadOne(1)}}
	seq.onCall = func(n int) {
		if n == 2 {
			<-release
		}
	}
	setGateLoadReader(t, seq.read, 2)
	var stderr lockedBuffer
	done := make(chan error, 1)
	var res RegateResult
	go func() {
		var err error
		res, err = fx.o.Regate(context.Background(), RegateParams{Branch: "feat/auth", Env: fx.env, Stderr: &stderr, MaxLoad: 3})
		done <- err
	}()
	waitForStderr(t, &stderr, "org: regate: the 1-minute load average is 9.00, above [org.gate].max_load 3: waiting for it to come down")
	gateHeld, heldErr := fileLockHeld(gateLockPathIn(fx.stateDir))
	regateHeld, heldErr2 := fileLockHeld(regateLockPathIn(fx.stateDir))
	close(release)
	// The regate is waited for first, so it never outlives the test's
	// temporary directories.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("regate: %v", err)
		}
	case <-time.After(time.Minute):
		t.Fatal("regate did not return within a minute of the load coming down")
	}
	if heldErr != nil || heldErr2 != nil || !gateHeld || regateHeld {
		t.Fatalf("while the regate waited for the load: gate.lock held %t (%v), regate.lock held %t (%v); want only gate.lock held",
			gateHeld, heldErr, regateHeld, heldErr2)
	}
	if res.Load == nil {
		t.Fatalf("regate = %+v; want a Load", res)
	}
	want := &GateLoad{Start: "1.00", Highest: "1.00", End: "1.00", NCPU: 2, MaxLoad: 3, Waited: res.Load.Waited}
	if res.Outcome != RegateOutcomeGated || !res.Passed || *res.Load != *want || res.Load.Waited <= 0 {
		t.Fatalf("regate = %+v (Load %+v); want a gated pass with Load %+v and a wait", res, res.Load, want)
	}
	if !strings.Contains(stderr.String(), "org: regate: the 1-minute load average is 1.00, not above [org.gate].max_load 3, after waiting ") {
		t.Errorf("stderr does not say the load came down:\n%s", stderr.String())
	}
	if last := lastEvent(t, fx.o); last.Event != EventGatePassed ||
		!strings.Contains(last.Details, " via=regate load_start=1.00 load_max=1.00 load_end=1.00 ncpu=2 max_load=3 log=") {
		t.Fatalf("last event = %+v; want a gate_passed via regate with the load keys and max_load=3", last)
	}
}

// TestRegateCI_NoLoadAndNoMaxLoadWait: a ci regate runs no script, so it
// neither reads the load nor waits for max_load, and its gate_ci_passed has
// no load key.
func TestRegateCI_NoLoadAndNoMaxLoadWait(t *testing.T) {
	fx := newRCFixture(t)
	setGateLoadReader(t, func(context.Context) (gateLoadAvg, error) {
		t.Error("a ci regate read the load")
		return gateLoadAvg{one: 99}, nil
	}, 2)
	p := fx.params()
	p.MaxLoad = 1
	res, err := fx.o.Regate(context.Background(), p)
	if err != nil || res.Outcome != RegateOutcomeGated || !res.Passed || res.Load != nil {
		t.Fatalf("regate = %+v, %v; want a gated pass with no Load", res, err)
	}
	last := lastEvent(t, fx.o)
	if last.Event != EventGateCIPassed || strings.Contains(last.Details, "load_") || strings.Contains(last.Details, "ncpu=") ||
		strings.Contains(last.Details, "max_load=") {
		t.Fatalf("last event = %+v; want a gate_ci_passed with no load key", last)
	}
}
