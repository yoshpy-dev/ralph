package org

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// `ralph org gate accept` at the org layer (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, AC7): GateAccept on
// the manifest fixtures of mergeorder_test.go, and the Details of a
// gate_accepted written and read back.

// The logs of the fixtures' two gate failures of feat/a, each run writing a
// log of its own.
const (
	gaLog1 = "/state/gates/org-a-20261014T070000Z.log"
	gaLog2 = "/state/gates/org-a-20261014T073000Z.log"
)

// gaFixture is a moFixture whose Org reads USER and HERDR_PANE_ID from env
// (USER=alice, nothing else unless a test sets more).
type gaFixture struct {
	*moFixture
	env map[string]string
}

func newGAFixture(t *testing.T) *gaFixture {
	t.Helper()
	fx := &gaFixture{moFixture: newMOFixture(t), env: map[string]string{"USER": "alice"}}
	fx.o.Getenv = func(key string) string { return fx.env[key] }
	return fx
}

// accept is GateAccept with p, failing the test on an error.
func (fx *gaFixture) accept(t *testing.T, p GateAcceptParams) GateAcceptResult {
	t.Helper()
	res, err := fx.o.GateAccept(p)
	if err != nil {
		t.Fatalf("GateAccept(%+v): %v", p, err)
	}
	return res
}

// gaGate is orgID's gate record of feat/a on moHeadA against
// refs/remotes/origin/main at moMain, a gate_failed for reason (a pass when
// it is empty), with log, at ts.
func gaGate(orgID, reason, log, ts string) ManifestEvent {
	r := gateRecord{Reason: reason, Head: moHeadA, Base: moMain, BaseRef: "refs/remotes/origin/main", Branch: "feat/a", By: "bob",
		Log: log}
	return ManifestEvent{TS: ts, OrgID: orgID, Event: r.event(), Details: r.details()}
}

// gaParams accepts feat/a's failure with log, with a reason.
func gaParams(log string) GateAcceptParams {
	return GateAcceptParams{Branch: "feat/a", Log: log, Reason: "the failing test is outside the change and passes alone"}
}

// gaAcceptedEvents returns the manifest's gate_accepted events.
func gaAcceptedEvents(t *testing.T, o *Org) []ManifestEvent {
	t.Helper()
	var out []ManifestEvent
	for _, ev := range mustReadEvents(t, o) {
		if ev.Event == EventGateAccepted {
			out = append(out, ev)
		}
	}
	return out
}

// TestGateAccept_RecordsTheLatestFailure covers AC7's record: one
// gate_accepted, appended after the failure, under the org of the failure's
// gate record (org-a, though org-b holds the branch's latest binding), at
// the clock's time, SeatID empty, with Details the failure's head, base,
// base_ref, branch, event, reason, ts and log, by=USER, herdr_pane, and the
// trimmed reason; the result holds the same. A gate_ci_failed is accepted
// the same way, by=unknown without USER, and herdr_pane=set for a pane no
// active seat has.
func TestGateAccept_RecordsTheLatestFailure(t *testing.T) {
	ciFail := moCIGate("org-a", "feat/a", moHeadA, moMain, GateReasonCIFailed)
	ciFail.TS = "2026-10-14T07:00:00Z"
	for _, tc := range []struct {
		name    string
		failure ManifestEvent
		env     map[string]string
		log     string
		details string
	}{
		{"a gate_failed", gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z"), nil, gaLog1,
			"head=" + moHeadA + " base=" + moMain + " base_ref=refs/remotes/origin/main branch=feat/a failed_event=gate_failed " +
				"failed_reason=test_failed failed_ts=2026-10-14T07:00:00Z by=alice herdr_pane=unset failed_log=\"" + gaLog1 + "\" " +
				"note=\"the failing test is outside the change and passes alone\""},
		{"a gate_ci_failed, no USER, a pane of no seat", ciFail, map[string]string{"USER": "", herdrPaneIDEnv: "p-9"}, moRegateCILog,
			"head=" + moHeadA + " base=" + moMain + " base_ref=refs/remotes/origin/main branch=feat/a failed_event=gate_ci_failed " +
				"failed_reason=ci_failed failed_ts=2026-10-14T07:00:00Z by=unknown herdr_pane=set failed_log=\"" + moRegateCILog +
				"\" note=\"the failing test is outside the change and passes alone\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGAFixture(t)
			for k, v := range tc.env {
				fx.env[k] = v
			}
			fx.add(t, moBind("org-a", "a"), tc.failure, moDisbanded("org-a"), moBindTo("org-b", "split-2", "a2", "feat/a"))
			before := fileOrAbsent(t, fx.path)
			p := gaParams(tc.log)
			p.Reason = "  " + p.Reason + "\t"
			res := fx.accept(t, p)
			if res.Refused != "" || res.Accepted == nil {
				t.Fatalf("GateAccept = %+v; want a recorded acceptance", res)
			}
			after := fileOrAbsent(t, fx.path)
			if !strings.HasPrefix(after, before) || strings.Count(after[len(before):], "\n") != 1 {
				t.Fatalf("the manifest got %q; want one line appended", after[len(before):])
			}
			events := mustReadEvents(t, fx.o)
			got := events[len(events)-1]
			want := ManifestEvent{TS: mergeOrderTestListedAt, OrgID: "org-a", Event: EventGateAccepted, Details: tc.details}
			if got != want {
				t.Fatalf("appended %+v\nwant     %+v", got, want)
			}
			a, ok := gateAcceptanceFromDetails(got.Details)
			if !ok {
				t.Fatalf("gateAcceptanceFromDetails(%q) does not read", got.Details)
			}
			a.TS, a.OrgID = got.TS, got.OrgID
			if *res.Accepted != a {
				t.Errorf("Accepted = %+v\nwant       %+v", *res.Accepted, a)
			}
		})
	}
}

// TestGateAccept_RefusalsRecordNothing covers AC7's refusals: each one
// returns Refused, no error and no acceptance, and leaves the manifest as it
// was. The flags first: an empty or blank --branch or --log, and a --reason
// that is empty once trimmed, spans lines, holds a control character, is not
// UTF-8, or has more than 500 characters; then the ledger: no gate record
// names the branch, the latest cannot be read or a corrupt line follows it,
// it is a pass, its reason is head_moved or tree_changed, its log cannot be
// read or is not --log, or the failure is accepted already. A branch with a
// control character is escaped in the refusal.
func TestGateAccept_RefusalsRecordNothing(t *testing.T) {
	bind := moBind("org-a", "a")
	fail1 := gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z")
	unquotedLog := fail1
	unquotedLog.Details = strings.Replace(fail1.Details, `log="`+gaLog1+`"`, "log="+gaLog1, 1)
	accepted := moAcceptOf(t, fail1, "carol", "accepted once")
	accepted.TS = "2026-10-14T07:10:00Z"
	withReason := func(reason string) GateAcceptParams {
		p := gaParams(gaLog1)
		p.Reason = reason
		return p
	}
	latest := "the latest gate record of the branch feat/a (gate_failed at 2026-10-14T07:00:00Z)"
	const again = "follow the branch's Detail in ralph org merge-order to run the gate again"
	for _, tc := range []struct {
		name  string
		lines []any
		p     GateAcceptParams
		want  string
	}{
		{"no --branch", []any{bind, fail1}, GateAcceptParams{Log: gaLog1, Reason: "r"}, "org: gate accept: --branch is required: " +
			"name the feature branch whose latest gate failure the human accepts (ralph org merge-order lists the ledger's branches)"},
		{"a blank --branch", []any{bind, fail1}, GateAcceptParams{Branch: " ", Log: gaLog1, Reason: "r"}, "--branch is required"},
		{"no --log", []any{bind, fail1}, GateAcceptParams{Branch: "feat/a", Reason: "r"}, "org: gate accept: --log is required: " +
			"name the gate log of the failure the human read and accepts (the LOG: line of ralph org gate, or the gate's log in " +
			"ralph org merge-order --json)"},
		{"no --reason", []any{bind, fail1}, withReason(""), "org: gate accept: --reason is required: say in one line why the " +
			"human accepts the failure"},
		{"a blank --reason", []any{bind, fail1}, withReason(" \t\n "), "--reason is required"},
		{"a --reason of two lines", []any{bind, fail1}, withReason("one\ntwo"), "org: gate accept: --reason must be a single line"},
		{"a --reason with U+2028", []any{bind, fail1}, withReason("one two"), "--reason must be a single line"},
		{"a --reason with a control character", []any{bind, fail1}, withReason("red \x1b[31m"),
			"org: gate accept: --reason contains a control character (U+001B)"},
		{"a --reason that is not UTF-8", []any{bind, fail1}, withReason("a\xffb"), "org: gate accept: --reason is not valid UTF-8"},
		{"a --reason of 501 characters", []any{bind, fail1}, withReason(strings.Repeat("é", 501)),
			"org: gate accept: --reason is 501 characters; at most 500"},
		{"no gate record names the branch", []any{bind}, gaParams(gaLog1), "org: gate accept: no gate record in the ledger names " +
			"the branch feat/a, so there is no gate failure to accept and nothing was recorded: ralph org merge-order lists the " +
			"ledger's branches and their gate records"},
		{"an empty ledger", nil, gaParams(gaLog1), "no gate record in the ledger names the branch feat/a"},
		{"another branch", []any{bind, fail1}, GateAcceptParams{Branch: "feat/b", Log: gaLog1, Reason: "r"},
			"no gate record in the ledger names the branch feat/b"},
		{"a branch with a control character", []any{bind, fail1}, GateAcceptParams{Branch: "feat/\x1b", Log: gaLog1, Reason: "r"},
			`no gate record in the ledger names the branch feat/\x1b,`},
		{"the latest record cannot be read", []any{bind, fail1, eventAt(moGarbled("org-a", EventGateFailed), "2026-10-14T07:20:00Z")},
			gaParams(gaLog1), "org: gate accept: the latest gate record of the branch feat/a (gate_failed at 2026-10-14T07:20:00Z) " +
				"cannot be read, so which failure it holds is unknown and nothing was recorded: " + again},
		{"a corrupt line follows the latest record", []any{bind, fail1, moCorruptLine}, gaParams(gaLog1),
			"org: gate accept: a corrupt manifest line follows " + latest + " and may be a newer one, so nothing was recorded: " + again},
		{"the latest record is a pass", []any{bind, fail1, gaGate("org-a", "", gaLog2, "2026-10-14T07:30:00Z")}, gaParams(gaLog2),
			"org: gate accept: the latest gate record of the branch feat/a (gate_passed at 2026-10-14T07:30:00Z) is a pass, not a " +
				"failure, so there is nothing to accept and nothing was recorded: ralph org merge-order lists the branch by that pass"},
		{"head_moved", []any{bind, gaGate("org-a", GateReasonHeadMoved, gaLog1, "2026-10-14T07:00:00Z")}, gaParams(gaLog1),
			"org: gate accept: " + latest + " failed with reason head_moved: the scripts passed, so there is nothing to accept and " +
				"nothing was recorded: " + again},
		{"tree_changed", []any{bind, gaGate("org-a", GateReasonTreeChanged, gaLog1, "2026-10-14T07:00:00Z")}, gaParams(gaLog1),
			latest + " failed with reason tree_changed: the scripts passed"},
		{"the record's log cannot be read", []any{bind, unquotedLog}, gaParams(gaLog1), "org: gate accept: " + latest + " failed " +
			"with reason test_failed, but its log cannot be read from the record, so the failure --log names cannot be checked " +
			"against it and nothing was recorded: " + again},
		{"another log", []any{bind, fail1}, gaParams(gaLog2), "org: gate accept: " + latest + " failed with reason test_failed and " +
			"its log is " + gaLog1 + ", not " + gaLog2 + ", so nothing was recorded: a newer gate may have replaced the failure that " +
			"was read; read " + gaLog1 + " and ask the human again before accepting that failure"},
		{"accepted already", []any{bind, fail1, accepted}, gaParams(gaLog1), "org: gate accept: " + latest + " failed with reason " +
			"test_failed, and carol accepted that failure at 2026-10-14T07:10:00Z already, so nothing more was recorded: ralph org " +
			"merge-order lists the branch by that acceptance"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGAFixture(t)
			fx.add(t, tc.lines...)
			before := fileOrAbsent(t, fx.path)
			res := fx.accept(t, tc.p)
			if res.Accepted != nil || !strings.Contains(res.Refused, tc.want) {
				t.Fatalf("GateAccept = %+v\nwant a refusal saying %q", res, tc.want)
			}
			if strings.ContainsFunc(res.Refused, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
				t.Errorf("the refusal holds a control character: %q", res.Refused)
			}
			if after := fileOrAbsent(t, fx.path); after != before {
				t.Errorf("the refusal wrote to the manifest:\nbefore %q\nafter  %q", before, after)
			}
		})
	}
}

// TestGateAccept_ReasonOf500CharactersIsRecorded: the bound of --reason is
// inclusive, counted in characters, not bytes.
func TestGateAccept_ReasonOf500CharactersIsRecorded(t *testing.T) {
	fx := newGAFixture(t)
	fx.add(t, moBind("org-a", "a"), gaGate("org-a", GateReasonStaticFailed, gaLog1, "2026-10-14T07:00:00Z"))
	p := gaParams(gaLog1)
	p.Reason = strings.Repeat("é", GateAcceptReasonMaxRunes)
	if res := fx.accept(t, p); res.Accepted == nil || res.Accepted.Note != p.Reason {
		t.Fatalf("GateAccept = %+v; want the 500-character reason recorded", res)
	}
}

// TestGateAccept_ALogOfAFailureANewerOneReplacedIsRefused covers the plan
// advisory's HIGH (D5): the human read and accepted failure F1, but a gate
// run again meanwhile recorded failure F2 on the same head, so accepting
// F1's log is refused, naming F2's log to read and ask about; accepting
// F2's log then records F2. An acceptance of F1 recorded before F2 does not
// carry over to F2 either.
func TestGateAccept_ALogOfAFailureANewerOneReplacedIsRefused(t *testing.T) {
	fx := newGAFixture(t)
	fail1 := gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z")
	fail2 := gaGate("org-a", GateReasonTestFailed, gaLog2, "2026-10-14T07:30:00Z")
	fx.add(t, moBind("org-a", "a"), fail1)
	if res := fx.accept(t, gaParams(gaLog1)); res.Accepted == nil {
		t.Fatalf("GateAccept of F1 = %+v; want it recorded", res)
	}
	fx.add(t, fail2)
	before := fileOrAbsent(t, fx.path)
	res := fx.accept(t, gaParams(gaLog1))
	want := "its log is " + gaLog2 + ", not " + gaLog1 + ", so nothing was recorded: a newer gate may have replaced the failure " +
		"that was read; read " + gaLog2 + " and ask the human again"
	if res.Accepted != nil || !strings.Contains(res.Refused, want) {
		t.Fatalf("GateAccept of F1 after F2 = %+v\nwant a refusal saying %q", res, want)
	}
	if fileOrAbsent(t, fx.path) != before {
		t.Fatal("the refusal wrote to the manifest")
	}
	res = fx.accept(t, gaParams(gaLog2))
	if res.Accepted == nil || res.Accepted.FailedLog != gaLog2 || res.Accepted.FailedTS != fail2.TS {
		t.Fatalf("GateAccept of F2 = %+v; want F2 accepted", res)
	}
	if got := gaAcceptedEvents(t, fx.o); len(got) != 2 {
		t.Fatalf("gate_accepted events = %+v; want F1's and F2's", got)
	}
}

// TestGateAccept_TwoAtOnceRecordOne covers AC7's last sentence: two accepts
// of the same failure that run at once both read and append under the
// manifest lock, so one records and the other is refused as accepted
// already.
func TestGateAccept_TwoAtOnceRecordOne(t *testing.T) {
	for range 5 {
		fx := newGAFixture(t)
		fx.add(t, moBind("org-a", "a"), gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z"))
		var wg sync.WaitGroup
		start := make(chan struct{})
		results := make([]GateAcceptResult, 2)
		errs := make([]error, 2)
		for i := range results {
			wg.Go(func() {
				<-start
				results[i], errs[i] = fx.o.GateAccept(gaParams(gaLog1))
			})
		}
		close(start)
		wg.Wait()
		accepted, refused := 0, 0
		for i, res := range results {
			switch {
			case errs[i] != nil:
				t.Fatalf("GateAccept %d: %v", i, errs[i])
			case res.Accepted != nil:
				accepted++
			case strings.Contains(res.Refused, "alice accepted that failure at "+mergeOrderTestListedAt+" already"):
				refused++
			}
		}
		if accepted != 1 || refused != 1 {
			t.Fatalf("results = %+v; want one recorded and one refused as accepted already", results)
		}
		if got := gaAcceptedEvents(t, fx.o); len(got) != 1 {
			t.Fatalf("gate_accepted events = %+v; want one", got)
		}
	}
}

// TestGateAccept_SeatPaneTripwire covers AC7's pane refusal: HERDR_PANE_ID
// equal to an active real seat's pane refuses with approve's sentence for
// gate accept, recording nothing; another pane is recorded as herdr_pane=set.
func TestGateAccept_SeatPaneTripwire(t *testing.T) {
	seat := ManifestEvent{OrgID: "org-b", SeatID: "seat-1", Event: EventSpawned, Role: "worker", PaneID: "p-7"}
	fx := newGAFixture(t)
	fx.add(t, seat, moBind("org-a", "a"), gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z"))
	fx.env[herdrPaneIDEnv] = "p-7"
	before := fileOrAbsent(t, fx.path)
	res := fx.accept(t, gaParams(gaLog1))
	const want = `org: gate accept: refused, since HERDR_PANE_ID "p-7" is the pane of the active seat "seat-1" of org_id "org-b": ` +
		"ralph org gate accept records a human's acceptance of a gate failure, so a human runs it from a terminal outside the " +
		"seats' panes (a human in a herdr pane that only shares its id with a seat of another herdr server runs it from a " +
		"terminal outside herdr)"
	if res.Accepted != nil || res.Refused != want {
		t.Fatalf("GateAccept = %+v\nwant the refusal %q", res, want)
	}
	if fileOrAbsent(t, fx.path) != before {
		t.Fatal("the pane refusal wrote to the manifest")
	}
	fx.env[herdrPaneIDEnv] = "p-8"
	if res := fx.accept(t, gaParams(gaLog1)); res.Accepted == nil || !res.Accepted.HerdrPane {
		t.Fatalf("GateAccept from another pane = %+v; want it recorded with herdr_pane=set", res)
	}
}

// TestGateAccept_ManifestThatCannotBeReadIsAnError: a manifest path that is
// a directory fails the read under the lock, with nothing recorded.
func TestGateAccept_ManifestThatCannotBeReadIsAnError(t *testing.T) {
	fx := newGAFixture(t)
	mkdirAll(t, fx.path)
	res, err := fx.o.GateAccept(gaParams(gaLog1))
	if err == nil || !strings.HasPrefix(err.Error(), "org: gate accept: read manifest: ") || res.Accepted != nil || res.Refused != "" {
		t.Fatalf("GateAccept = %+v, %v; want the read error", res, err)
	}
}

// TestGateAcceptanceDetails_RoundTrip: a gate_accepted's Details read back
// as written, with a log and a note that hold spaces, quotes, the text of
// the next key, a backslash, and characters outside ASCII, and with
// herdr_pane both ways. A key the reader does not know before failed_log is
// skipped.
func TestGateAcceptanceDetails_RoundTrip(t *testing.T) {
	for _, pane := range []bool{false, true} {
		a := GateAcceptance{
			Head: moHeadA, Base: moMain, BaseRef: "refs/heads/main", Branch: "feat/a", FailedEvent: EventGateCIFailed,
			FailedReason: GateReasonCIFailed, FailedTS: "2026-10-14T07:00:00Z", FailedLog: `/state/gates/a b" note="x.log`,
			By: "alice", HerdrPane: pane, Note: `CI's "flaky" job — ünïcode \ and note="y"`,
		}
		got, ok := gateAcceptanceFromDetails(a.details())
		if !ok || got != a {
			t.Fatalf("gateAcceptanceFromDetails(%q) = %+v, %v\nwant %+v", a.details(), got, ok, a)
		}
		withUnknown := strings.Replace(a.details(), " failed_log=", " future_key=1 failed_log=", 1)
		if got, ok := gateAcceptanceFromDetails(withUnknown); !ok || got != a {
			t.Fatalf("with an unknown key: %+v, %v; want %+v", got, ok, a)
		}
	}
}

// TestGateAcceptanceFromDetails_Malformed: anything out of shape reads as
// not ok, so merge-order skips it and the branch stays failed.
func TestGateAcceptanceFromDetails_Malformed(t *testing.T) {
	a := GateAcceptance{
		Head: moHeadA, Base: moMain, BaseRef: "refs/remotes/origin/main", Branch: "feat/a", FailedEvent: EventGateFailed,
		FailedReason: GateReasonTestFailed, FailedTS: "2026-10-14T07:00:00Z", FailedLog: gaLog1, By: "alice", Note: "ok",
	}
	good := a.details()
	if _, ok := gateAcceptanceFromDetails(good); !ok {
		t.Fatalf("the fixture's Details %q do not read", good)
	}
	tokens, _, _ := strings.Cut(good, " failed_log=")
	logPart := "failed_log=" + strconv.Quote(gaLog1)
	with := func(old, new string) string { return strings.Replace(good, old, new, 1) }
	for name, details := range map[string]string{
		"empty":                       "",
		"no failed_log":               tokens,
		"no note":                     tokens + " " + logPart,
		"text after note":             good + " x=1",
		"note before failed_log":      tokens + ` note="ok" ` + logPart,
		"a single-quoted note":        with(`note="ok"`, `note='o'`),
		"a backquoted note":           with(`note="ok"`, "note=`ok`"),
		"a backquoted failed_log":     with(strconv.Quote(gaLog1), "`"+gaLog1+"`"),
		"an unterminated note":        with(`note="ok"`, `note="ok`),
		"a key given twice":           "head=" + moHeadA + " " + good,
		"a token without =":           with(" by=", " junk by="),
		"an empty token":              with(" base=", "  base="),
		"a token with an empty key":   with(" by=", " =x by="),
		"a head that is not a SHA":    with("head="+moHeadA, "head=abc"),
		"a base that is not a SHA":    with("base="+moMain, "base=main"),
		"a base_ref not under refs/":  with("base_ref=refs/remotes/origin/main", "base_ref=origin/main"),
		"an empty branch":             with("branch=feat/a", "branch="),
		"no branch":                   with(" branch=feat/a", ""),
		"failed_event gate_passed":    with("failed_event=gate_failed", "failed_event=gate_passed"),
		"failed_reason head_moved":    with("failed_reason=test_failed", "failed_reason=head_moved"),
		"failed_reason tree_changed":  with("failed_reason=test_failed", "failed_reason=tree_changed"),
		"an empty failed_ts":          with("failed_ts=2026-10-14T07:00:00Z", "failed_ts="),
		"an empty by":                 with("by=alice", "by="),
		"herdr_pane neither way":      with("herdr_pane=unset", "herdr_pane=maybe"),
		"no herdr_pane":               with(" herdr_pane=unset", ""),
		"an empty failed_log":         with(strconv.Quote(gaLog1), `""`),
		"an empty note":               with(`note="ok"`, `note=""`),
		"a note of two lines":         with(`note="ok"`, `note="o\nk"`),
		"a note with a control":       with(`note="ok"`, `note="o\x1bk"`),
		"a note that is not UTF-8":    with(`note="ok"`, `note="o\xffk"`),
		"a note of 501 characters":    with(`note="ok"`, strconv.Quote(strings.Repeat("x", 501))),
		"a quoted note with no key":   with(` note="ok"`, ` "ok"`),
		"a failed_log with no quotes": with(strconv.Quote(gaLog1), gaLog1),
	} {
		if got, ok := gateAcceptanceFromDetails(details); ok {
			t.Errorf("%s: gateAcceptanceFromDetails(%q) = %+v, ok; want not ok", name, details, got)
		}
	}
}

// moAcceptOf is the gate_accepted of failed, a gate failure whose TS is set,
// with by as who accepted it and note as why, under failed's org, as
// GateAccept records it; its TS is left for the test.
func moAcceptOf(t *testing.T, failed ManifestEvent, by, note string) ManifestEvent {
	t.Helper()
	r, ok := gateRecordFromDetails(failed.Details)
	if !ok || r.Reason == "" || failed.TS == "" {
		t.Fatalf("moAcceptOf: %+v is not a readable gate failure with a ts", failed)
	}
	a := GateAcceptance{
		Head: r.Head, Base: r.Base, BaseRef: r.BaseRef, Branch: r.Branch, FailedEvent: failed.Event, FailedReason: r.Reason,
		FailedTS: failed.TS, FailedLog: r.Log, By: by, Note: note,
	}
	return ManifestEvent{OrgID: failed.OrgID, Event: EventGateAccepted, Details: a.details()}
}

// eventAt is ev with its TS set to ts.
func eventAt(ev ManifestEvent, ts string) ManifestEvent {
	ev.TS = ts
	return ev
}

// TestGateAccept_MergeOrderListsTheBranchAsAccepted runs GateAccept and
// MergeOrder on one ledger: the branch GateAccept records an acceptance for
// is accepted, its gate's Accepted the one GateAccept returned.
func TestGateAccept_MergeOrderListsTheBranchAsAccepted(t *testing.T) {
	fx := newGAFixture(t)
	fx.add(t, moBind("org-a", "a"), gaGate("org-a", GateReasonTestFailed, gaLog1, "2026-10-14T07:00:00Z"), moDisbanded("org-a"))
	res := fx.accept(t, gaParams(gaLog1))
	if res.Accepted == nil {
		t.Fatalf("GateAccept = %+v", res)
	}
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
	mo, err := fx.o.MergeOrder(context.Background(), fx.params(false))
	if err != nil {
		t.Fatal(err)
	}
	e := moEntry(t, mo, "feat/a")
	want := MergeOrderAcceptance{By: "alice", TS: mergeOrderTestListedAt, Reason: res.Accepted.Note}
	if e.Class != MergeClassAccepted || e.Gate == nil || e.Gate.Accepted == nil || *e.Gate.Accepted != want || e.Gate.Log != gaLog1 {
		t.Fatalf("feat/a = %+v (gate %+v); want accepted with %+v and the log %s", e, e.Gate, want, gaLog1)
	}
}
