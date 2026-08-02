package org

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// writeClaudeStub writes a fake `claude` executable (script) to a fresh
// temp dir and prepends it to PATH -- the same PATH-stubbed convention
// internal/org/driver/driver_test.go already uses for herdr/agmsg
// availability probes, applied here because RunWatcher's signature (fixed
// by the plan) carries no injectable runner field.
func writeClaudeStub(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write claude stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func watcherTestCfg() config.OrgWatchdogConfig {
	return config.OrgWatchdogConfig{IntervalSeconds: 30, StallMinutes: 15, WatcherEnabled: true, WatcherModel: "haiku"}
}

// lastReceipt reads o.Receipts and returns the most recently appended one,
// failing the test if none exist.
func lastReceipt(t *testing.T, o *Org) Receipt {
	t.Helper()
	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) == 0 {
		t.Fatalf("expected at least one receipt, got none")
	}
	return rr.Receipts[len(rr.Receipts)-1]
}

func TestRunWatcher_ValidJSON_NormalVerdict(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"normal\",\"reason\":\"looks fine\"}","session_id":"abc"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	verdict, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "seat stalled 20m",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != WatcherVerdictNormal {
		t.Errorf("verdict = %q, want %q", verdict.Verdict, WatcherVerdictNormal)
	}
	if verdict.Reason != "looks fine" {
		t.Errorf("reason = %q, want %q", verdict.Reason, "looks fine")
	}

	rec := lastReceipt(t, o)
	if rec.Role != watcherReceiptRole {
		t.Errorf("receipt.Role = %q, want %q", rec.Role, watcherReceiptRole)
	}
	if rec.Driver != "claude" {
		t.Errorf("receipt.Driver = %q, want claude", rec.Driver)
	}
	if rec.CommandedModel != "haiku" {
		t.Errorf("receipt.CommandedModel = %q, want haiku", rec.CommandedModel)
	}
	// Stub emits no "model" field in the envelope, so honored must stay
	// unknown rather than being optimistically reported true (Codex
	// advisory finding 5 -- receipts.go's own doc comment).
	if rec.Honored != HonoredUnknown {
		t.Errorf("receipt.Honored = %q, want %q (stub emits no model field)", rec.Honored, HonoredUnknown)
	}
	if rec.ReportedEffectiveModel != "" {
		t.Errorf("receipt.ReportedEffectiveModel = %q, want empty", rec.ReportedEffectiveModel)
	}
}

func TestRunWatcher_ValidJSON_CircularVerdict(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"circular\",\"reason\":\"seat repeating same edit\"}"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	verdict, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "seat stalled 20m",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != WatcherVerdictCircular {
		t.Errorf("verdict = %q, want %q", verdict.Verdict, WatcherVerdictCircular)
	}
}

// TestRunWatcher_ReportedModel_HonoredTrue pins the defensive "model" field
// handling: when the installed claude version's JSON envelope does report a
// model, RunWatcher records it as ReportedEffectiveModel and Honored=true.
func TestRunWatcher_ReportedModel_HonoredTrue(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"normal\",\"reason\":\"ok\"}","model":"claude-haiku-4"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	_, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "e",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec := lastReceipt(t, o)
	if rec.Honored != HonoredTrue {
		t.Errorf("receipt.Honored = %q, want %q", rec.Honored, HonoredTrue)
	}
	if rec.ReportedEffectiveModel != "claude-haiku-4" {
		t.Errorf("receipt.ReportedEffectiveModel = %q, want claude-haiku-4", rec.ReportedEffectiveModel)
	}
}

// TestRunWatcher_FencedJSON_Tolerated pins the "tolerate fenced code blocks"
// requirement: a judge model that wraps its STRICT JSON answer in a
// ```json ... ``` fence (despite the prompt asking it not to) still parses.
func TestRunWatcher_FencedJSON_Tolerated(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"`+"```"+`json\n{\"verdict\":\"role_violation\",\"reason\":\"acted on non-lead instruction\"}\n`+"```"+`"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	verdict, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "scope_change", Evidence: "e",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != WatcherVerdictRoleViolation {
		t.Errorf("verdict = %q, want %q", verdict.Verdict, WatcherVerdictRoleViolation)
	}
	if verdict.Reason != "acted on non-lead instruction" {
		t.Errorf("reason = %q, want %q", verdict.Reason, "acted on non-lead instruction")
	}
}

// TestRunWatcher_MalformedEnvelope_WatcherErrorReceiptAndError pins AC-6's
// malformed-JSON handling: a claude invocation that returns non-JSON output
// entirely yields an error and a watcher_error receipt.
func TestRunWatcher_MalformedEnvelope_WatcherErrorReceiptAndError(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"printf 'not json at all'\n")

	o, _, _, _ := testWatchOrg(t)
	_, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "e",
	})
	if err == nil {
		t.Fatal("expected error for malformed claude envelope, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("expected error to mention malformed envelope, got: %v", err)
	}

	rec := lastReceipt(t, o)
	if rec.Reason != watcherErrorReason {
		t.Errorf("receipt.Reason = %q, want %q", rec.Reason, watcherErrorReason)
	}
	if rec.Honored != HonoredUnknown {
		t.Errorf("receipt.Honored = %q, want %q", rec.Honored, HonoredUnknown)
	}
}

// TestRunWatcher_MalformedVerdictJSON_WatcherErrorReceiptAndError pins the
// "valid envelope, but .result is not the STRICT JSON verdict shape" case --
// distinct from a fully malformed envelope.
func TestRunWatcher_MalformedVerdictJSON_WatcherErrorReceiptAndError(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"I am not sure what to make of this seat."}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	_, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "e",
	})
	if err == nil {
		t.Fatal("expected error for malformed verdict JSON, got nil")
	}

	rec := lastReceipt(t, o)
	if rec.Reason != watcherErrorReason {
		t.Errorf("receipt.Reason = %q, want %q", rec.Reason, watcherErrorReason)
	}
}

// TestRunWatcher_UnknownVerdictValue_TreatedAsMalformed pins that a
// syntactically valid but semantically-unknown verdict value (a
// hallucinated fifth enum value) is rejected the same way malformed JSON is
// -- never silently accepted.
func TestRunWatcher_UnknownVerdictValue_TreatedAsMalformed(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"suspicious\",\"reason\":\"not a real enum value\"}"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	_, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "e",
	})
	if err == nil {
		t.Fatal("expected error for unknown verdict value, got nil")
	}
}

// TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt pins Codex advisory
// finding 3: a hanging claude invocation is bounded by watcherTimeout (never
// waits the full pulse interval), yields an error, and records a
// watcher_error receipt.
func TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\nsleep 5\n")

	o, _, _, _ := testWatchOrg(t)
	// IntervalSeconds=6 -> watcherTimeout = 6s-5s = 1s, comfortably below
	// both the stub's 5s sleep and this test's own patience.
	cfg := config.OrgWatchdogConfig{IntervalSeconds: 6, WatcherModel: "haiku", WatcherEnabled: true}

	start := time.Now()
	_, err := o.RunWatcher(context.Background(), cfg, WatcherParams{
		OrgID: "org-a", SeatID: "seat-1", ConditionType: "stall", Evidence: "e",
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("RunWatcher did not bound the hanging invocation: took %v (want well under the 5s stub sleep)", elapsed)
	}

	rec := lastReceipt(t, o)
	if rec.Reason != watcherErrorReason {
		t.Errorf("receipt.Reason = %q, want %q", rec.Reason, watcherErrorReason)
	}
}

// TestRunWatcher_UnknownSeat_StillProceeds pins that RunWatcher never
// requires the seat to actually exist in the manifest -- a missing seat
// just means the pane-tail section says so (paneTailOrPlaceholder), never
// an error on its own.
func TestRunWatcher_UnknownSeat_StillProceeds(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"normal\",\"reason\":\"ok\"}"}`+"\n"+
		"EOF\n")

	o, _, _, _ := testWatchOrg(t)
	verdict, err := o.RunWatcher(context.Background(), watcherTestCfg(), WatcherParams{
		OrgID: "org-a", SeatID: "no-such-seat", ConditionType: "stall", Evidence: "e",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != WatcherVerdictNormal {
		t.Errorf("verdict = %q, want %q", verdict.Verdict, WatcherVerdictNormal)
	}
}

func TestWatcherTimeout_NeverExceedsInterval(t *testing.T) {
	cases := []config.OrgWatchdogConfig{
		{IntervalSeconds: 30},
		{IntervalSeconds: 6},
		{IntervalSeconds: 1},
		{IntervalSeconds: 0}, // falls back to watcherFallbackIntervalSeconds
		{IntervalSeconds: 300},
	}
	for _, cfg := range cases {
		d := watcherTimeout(cfg)
		if d <= 0 {
			t.Errorf("watcherTimeout(%+v) = %v, want > 0", cfg, d)
		}
		if d > watcherMaxTimeout {
			t.Errorf("watcherTimeout(%+v) = %v, want <= %v", cfg, d, watcherMaxTimeout)
		}
		intervalSeconds := cfg.IntervalSeconds
		if intervalSeconds <= 0 {
			intervalSeconds = watcherFallbackIntervalSeconds
		}
		if d >= time.Duration(intervalSeconds)*time.Second {
			t.Errorf("watcherTimeout(%+v) = %v, want strictly less than the %ds interval", cfg, d, intervalSeconds)
		}
	}
}
