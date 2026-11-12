package org

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// Tests for LiveSeatStates, the reads of `ralph org status --live` (plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, D6 and AC6; issue
// #308). herdr is fakeResumeHerdr (resume_test.go), which scripts each
// agent's status and each pane's screen; the screens are the real ones of
// testdata/resume/.

// liveTestSeat is an active seat of org-a with a pane, as Status lists one.
func liveTestSeat(seatID, driver, pane string) SeatStatus {
	return SeatStatus{
		OrgID: "org-a", SeatID: seatID, Role: seatID, Driver: driver, PaneID: pane,
		HerdrAgentName: herdrAgentName("org-a", seatID), Event: EventSpawned, Active: true,
	}
}

// TestLiveSeatStates_HerdrStateAgainstTheScreen pins one seat per case: the
// herdr state, the screen's kind, reason, and line, and the mismatch, which
// only herdr's working next to an idle screen sets.
func TestLiveSeatStates_HerdrStateAgainstTheScreen(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name       string
		driver     string
		status     string // herdr agent_status; "" makes AgentGet fail with boom
		screen     string // fixture under testdata/resume; "" makes PaneRead fail with boom
		wantHerdr  string
		wantErr    string
		wantScreen string
		wantReason string
		wantLine   string
		wantReads  int
		mismatch   bool
	}{
		{name: "working and an idle claude screen", driver: "claude", status: "working", screen: "claude-idle-after-reply.txt",
			wantHerdr: "working", wantScreen: "idle", wantReason: "the turn ended", wantLine: "✻ Cooked for 3s · done 6:09 PM", wantReads: 1, mismatch: true},
		{name: "working and an idle codex screen", driver: "codex", status: "working", screen: "codex-idle-after-reply.txt",
			wantHerdr: "working", wantScreen: "idle", wantReason: "the turn ended", wantLine: "Worked for 4s • 18:17", wantReads: 1, mismatch: true},
		{name: "done and an idle screen", driver: "claude", status: "done", screen: "claude-idle-after-reply.txt",
			wantHerdr: "done", wantScreen: "idle", wantReason: "the turn ended", wantLine: "✻ Cooked for 3s · done 6:09 PM", wantReads: 1},
		{name: "working and a busy screen", driver: "claude", status: "working", screen: "claude-working-tool.txt",
			wantHerdr: "working", wantScreen: "busy", wantReason: "a busy line is on the screen", wantLine: "✶ Deliberating… (6s · ↓ 100 tokens)", wantReads: 1},
		{name: "blocked and a choice dialog", driver: "codex", status: "blocked", screen: "codex-hooks-review.txt",
			wantHerdr: "blocked", wantScreen: "choice", wantReason: "choice dialog", wantLine: "› 1. Review hooks", wantReads: 1},
		{name: "agent get fails", driver: "claude", screen: "claude-idle-after-reply.txt",
			wantHerdr: "unknown", wantErr: "herdr agent get org-a_seat failed: boom",
			wantScreen: "idle", wantReason: "the turn ended", wantLine: "✻ Cooked for 3s · done 6:09 PM", wantReads: 1},
		{name: "pane read fails", driver: "claude", status: "working",
			wantHerdr: "working", wantScreen: "unreadable", wantReason: "herdr pane read pane-s failed: boom", wantReads: 1},
		{name: "a driver with no screen rules", driver: "gemini", status: "working", screen: "claude-idle-after-reply.txt",
			wantHerdr: "working", wantScreen: "unknown", wantReason: "unknown driver"},
		{name: "no driver recorded", driver: "", status: "working", screen: "claude-idle-after-reply.txt",
			wantHerdr: "working", wantScreen: "unknown", wantReason: "unknown driver"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			if tc.status == "" {
				h.getErr["org-a_seat"] = boom
			} else {
				h.setStatus("org-a_seat", tc.status)
			}
			if tc.screen == "" {
				h.readErr["pane-s"] = boom
			} else {
				h.setScreens("pane-s", readResumeScreen(t, tc.screen))
			}

			got := o.LiveSeatStates([]SeatStatus{liveTestSeat("seat", tc.driver, "pane-s")})
			want := LiveSeatState{
				Read: true, Agent: "org-a_seat", HerdrState: tc.wantHerdr, HerdrErr: tc.wantErr,
				Screen: tc.wantScreen, ScreenReason: tc.wantReason, ScreenLine: tc.wantLine, Mismatch: tc.mismatch,
			}
			if len(got) != 1 || got[0] != want {
				t.Fatalf("LiveSeatStates =\n%+v\nwant\n[%+v]", got, want)
			}
			if n := h.readCount(); n != tc.wantReads {
				t.Errorf("pane reads = %d, want %d (calls %v)", n, tc.wantReads, h.calls())
			}
		})
	}
}

// TestLiveSeatStates_SkipsSeatsWithNothingToRead: a stopped seat, a dry-run
// seat, and a seat with no pane get a LiveSeatState with Read false in
// their place, and herdr is asked about none of them. The seats it reads
// are read one after another in roster order: the agent, then the pane, 60
// lines as send's screen check reads it.
func TestLiveSeatStates_SkipsSeatsWithNothingToRead(t *testing.T) {
	o, h := resumeTestOrg(t)
	h.setStatus("org-a_implementer", "working")
	h.setScreens("pane-i", readResumeScreen(t, "claude-idle-after-reply.txt"))
	h.setStatus("org-a_leader", "done")
	h.setScreens("pane-l", readResumeScreen(t, "codex-idle-after-reply.txt"))

	stopped := liveTestSeat("reviewer", "claude", "pane-r")
	stopped.Active, stopped.Event = false, EventStopped
	dryRun := liveTestSeat("planner", "claude", "pane-d")
	dryRun.DryRun = true
	noPane := liveTestSeat("helper", "claude", "")
	seats := []SeatStatus{
		liveTestSeat("implementer", "claude", "pane-i"), stopped, dryRun, noPane, liveTestSeat("leader", "codex", "pane-l"),
	}

	got := o.LiveSeatStates(seats)
	if len(got) != len(seats) {
		t.Fatalf("len(LiveSeatStates) = %d, want %d", len(got), len(seats))
	}
	for i, s := range seats {
		if want := s.Active && !s.DryRun && s.PaneID != ""; got[i].Read != want {
			t.Errorf("seat %s: Read = %v, want %v", s.SeatID, got[i].Read, want)
		}
	}
	for _, i := range []int{1, 2, 3} {
		if got[i] != (LiveSeatState{}) {
			t.Errorf("seat %s: got %+v, want the zero LiveSeatState", seats[i].SeatID, got[i])
		}
	}
	if !got[0].Mismatch || got[4].Mismatch {
		t.Errorf("Mismatch: implementer %v (want true), leader %v (want false)", got[0].Mismatch, got[4].Mismatch)
	}
	wantCalls := []string{"get org-a_implementer", "read pane-i 60", "get org-a_leader", "read pane-l 60"}
	if calls := h.calls(); !slices.Equal(calls, wantCalls) {
		t.Errorf("herdr calls = %v, want %v", calls, wantCalls)
	}
}

// TestLiveSeatStates_ChangesNothing: reading the live states appends nothing
// to the manifest and types nothing into a pane.
func TestLiveSeatStates_ChangesNothing(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "implementer", "implementer", "claude", "pane-i")
	h.setStatus(name, "working")
	h.setScreens("pane-i", readResumeScreen(t, "claude-idle-after-reply.txt"))
	before, err := os.ReadFile(o.Manifest.Path())
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	result, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	got := o.LiveSeatStates(result.Seats)
	if len(got) != 1 || !got[0].Mismatch || got[0].Agent != name {
		t.Fatalf("LiveSeatStates = %+v, want one mismatched seat with agent %s", got, name)
	}
	after, err := os.ReadFile(o.Manifest.Path())
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("the manifest changed:\nbefore %s\nafter  %s", before, after)
	}
	if sends := resumeSends(h.calls()); len(sends) != 0 {
		t.Errorf("sends = %v, want none", sends)
	}
}

// TestLiveSeatStates_AHerdrClientWithoutAgentGet: a herdr client that cannot
// run `herdr agent get` gives the state unknown and says so; the pane is
// still read.
func TestLiveSeatStates_AHerdrClientWithoutAgentGet(t *testing.T) {
	o, h, _ := testOrg(t)
	got := o.LiveSeatStates([]SeatStatus{liveTestSeat("seat", "claude", "pane-s")})
	if len(got) != 1 || got[0].HerdrState != "unknown" || got[0].HerdrErr != resumeReasonNoAgentGet {
		t.Fatalf("LiveSeatStates = %+v, want herdr state unknown with %q", got, resumeReasonNoAgentGet)
	}
	if got[0].Screen != "unknown" || got[0].Mismatch {
		t.Errorf("screen %q, mismatch %v; want unknown (the fake's \"pane output\" is no screen) and no mismatch", got[0].Screen, got[0].Mismatch)
	}
	if !slices.Contains(h.calls, "pane_read") {
		t.Errorf("herdr calls = %v, want a pane_read", h.calls)
	}
}

// liveHangingHerdr is a fakeResumeHerdr whose agent get and pane read never
// answer: each returns only when its ctx ends.
type liveHangingHerdr struct{ *fakeResumeHerdr }

func (liveHangingHerdr) AgentGet(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (liveHangingHerdr) PaneRead(ctx context.Context, _ string, _ int) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// TestLiveSeatStates_EachCallEndsAtTheDriverCallTimeout: a herdr that never
// answers holds each call for Org.DriverCallTimeout only, and the seat says
// what failed.
func TestLiveSeatStates_EachCallEndsAtTheDriverCallTimeout(t *testing.T) {
	o, _ := resumeTestOrg(t)
	o.Herdr = liveHangingHerdr{newFakeResumeHerdr()}
	o.DriverCallTimeout = 50 * time.Millisecond

	start := time.Now()
	got := o.LiveSeatStates([]SeatStatus{liveTestSeat("seat", "claude", "pane-s")})
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond || elapsed > 5*time.Second {
		t.Errorf("LiveSeatStates took %v, want two calls of 50ms each (under 5s)", elapsed)
	}
	want := LiveSeatState{
		Read: true, Agent: "org-a_seat", HerdrState: "unknown",
		HerdrErr: "herdr agent get org-a_seat failed: context deadline exceeded", Screen: "unreadable",
		ScreenReason: "herdr pane read pane-s failed: context deadline exceeded",
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("LiveSeatStates =\n%+v\nwant\n[%+v]", got, want)
	}
}

// TestLiveSeatStates_ErrorsAreOneShortLine: a herdr failure of several lines
// is kept on one line and cut to liveErrMaxRunes runes.
func TestLiveSeatStates_ErrorsAreOneShortLine(t *testing.T) {
	o, h := resumeTestOrg(t)
	h.getErr["org-a_seat"] = errors.New("exit status 1\nstderr: connect: connection refused\n" + strings.Repeat("x", 200))
	h.setScreens("pane-s", readResumeScreen(t, "claude-idle-after-reply.txt"))

	got := o.LiveSeatStates([]SeatStatus{liveTestSeat("seat", "claude", "pane-s")})
	if len(got) != 1 {
		t.Fatalf("LiveSeatStates = %+v, want one state", got)
	}
	msg := got[0].HerdrErr
	if strings.Contains(msg, "\n") || !strings.HasPrefix(msg, "herdr agent get org-a_seat failed: exit status 1 stderr: connect:") {
		t.Errorf("HerdrErr = %q, want one line starting with the call and its first words", msg)
	}
	if n := len([]rune(msg)); n != liveErrMaxRunes+1 || !strings.HasSuffix(msg, "…") {
		t.Errorf("HerdrErr has %d runes (%q), want %d ending with …", n, msg, liveErrMaxRunes+1)
	}
}

// TestLiveSeatState_TextOfTheColumns pins the two columns `ralph org status
// --live` prints for a seat: the herdr state with its failure, and the
// screen as send's errors write it, its line cut to 60 runes.
func TestLiveSeatState_TextOfTheColumns(t *testing.T) {
	long := strings.Repeat("あ", 70)
	cases := []struct {
		name       string
		state      LiveSeatState
		herdr      string
		screenText string
	}{
		{"idle with its line", LiveSeatState{HerdrState: "working", Screen: "idle", ScreenReason: "the turn ended", ScreenLine: "✻ Cooked for 3s · done 6:09 PM"},
			"working", `idle (the turn ended: "✻ Cooked for 3s · done 6:09 PM")`},
		{"no line", LiveSeatState{HerdrState: "done", Screen: "unknown", ScreenReason: "unknown driver"},
			"done", "unknown (unknown driver)"},
		{"failures", LiveSeatState{HerdrState: "unknown", HerdrErr: "herdr agent get x failed: boom", Screen: "unreadable", ScreenReason: "herdr pane read p failed: boom"},
			"unknown (herdr agent get x failed: boom)", "unreadable (herdr pane read p failed: boom)"},
		{"a long line", LiveSeatState{HerdrState: "working", Screen: "busy", ScreenReason: "a busy line is on the screen", ScreenLine: long},
			"working", `busy (a busy line is on the screen: "` + strings.Repeat("あ", 60) + `…")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.HerdrText(); got != tc.herdr {
				t.Errorf("HerdrText = %q, want %q", got, tc.herdr)
			}
			if got := tc.state.ScreenText(); got != tc.screenText {
				t.Errorf("ScreenText = %q, want %q", got, tc.screenText)
			}
		})
	}
}
