package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

func testOrgConfig() config.OrgConfig {
	return config.OrgConfig{
		DriverPool: []string{"claude", "codex"},
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "claude", Model: "opus"},
			{Driver: "claude", Model: "sonnet"},
			{Driver: "claude", Model: "haiku"},
			{Driver: "codex", Model: "gpt-5-codex"},
		},
		Roles:    map[string][]string{},
		MaxSeats: 3,
		// The org-wide limits at config.Default's values, so they never bind
		// in a test that does not set them lower itself.
		MaxOrgs:       10,
		MaxTotalSeats: 30,
	}
}

// TestValidateOrgWideCapacity pins the two org-wide limits at their
// boundaries: max_orgs refuses only an org_id that is not running yet once
// the running count reaches the limit, max_total_seats refuses any new seat
// once the active total reaches it, both errors point at disband, and a
// limit of 0 (a hand-built config; config.Load rejects it) is no limit.
func TestValidateOrgWideCapacity(t *testing.T) {
	req := func(orgID string) SpawnRequest {
		return SpawnRequest{OrgID: orgID, SeatID: "seat-1", Driver: "claude", Model: "sonnet"}
	}
	tests := []struct {
		name      string
		maxOrgs   int
		maxTotal  int
		orgID     string
		running   []string
		total     int
		wantError string
	}{
		{"new org below max_orgs", 2, 30, "org-c", []string{"org-a"}, 1, ""},
		{"new org at max_orgs", 2, 30, "org-c", []string{"org-a", "org-b"}, 2, `max_orgs 2 reached: org_id "org-c" is not running and 2 orgs are (org-a, org-b)`},
		{"running org at max_orgs", 2, 30, "org-b", []string{"org-a", "org-b"}, 2, ""},
		{"running org over max_orgs", 1, 30, "org-a", []string{"org-a", "org-b"}, 2, ""},
		{"seat below max_total_seats", 10, 3, "org-a", []string{"org-a"}, 2, ""},
		{"seat at max_total_seats", 10, 3, "org-a", []string{"org-a"}, 3, "max_total_seats 3 reached: 3 seats are active across all orgs"},
		{"new org at max_total_seats", 10, 3, "org-c", []string{"org-a"}, 3, "max_total_seats 3 reached"},
		{"both reached reports max_orgs first", 1, 1, "org-b", []string{"org-a"}, 1, "max_orgs 1 reached"},
		{"zero limits are not set", 0, 0, "org-z", []string{"org-a", "org-b"}, 99, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testOrgConfig()
			cfg.MaxOrgs, cfg.MaxTotalSeats = tt.maxOrgs, tt.maxTotal
			err := ValidateOrgWideCapacity(cfg, req(tt.orgID), tt.running, tt.total)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected an error containing %q, got %v", tt.wantError, err)
			}
			for _, hint := range []string{"ralph org disband --org-id <id>", "ralph org disband --all"} {
				if !strings.Contains(err.Error(), hint) {
					t.Errorf("expected the error to point at %q, got %v", hint, err)
				}
			}
		})
	}
}

func TestValidateSpawn(t *testing.T) {
	t.Run("in-pool ok", func(t *testing.T) {
		cfg := testOrgConfig()
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Driver: "claude", Model: "sonnet"}
		if err := ValidateSpawn(cfg, req, 0); err != nil {
			t.Fatalf("expected in-pool spawn to be allowed, got error: %v", err)
		}
	})

	t.Run("out-of-pool model rejected", func(t *testing.T) {
		cfg := testOrgConfig()
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Driver: "claude", Model: "not-a-real-model"}
		err := ValidateSpawn(cfg, req, 0)
		if err == nil {
			t.Fatal("expected out-of-pool model to be rejected")
		}
		if got := err.Error(); !strings.Contains(got, "not in [org].model_pool") {
			t.Errorf("expected model_pool rejection message, got: %s", got)
		}
	})

	t.Run("driver not in driver_pool rejected", func(t *testing.T) {
		cfg := testOrgConfig()
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Driver: "gemini", Model: "sonnet"}
		err := ValidateSpawn(cfg, req, 0)
		if err == nil {
			t.Fatal("expected driver not in driver_pool to be rejected")
		}
		if got := err.Error(); !strings.Contains(got, "not in [org].driver_pool") {
			t.Errorf("expected driver_pool rejection message, got: %s", got)
		}
	})

	t.Run("role constraint violation rejected", func(t *testing.T) {
		cfg := testOrgConfig()
		cfg.Roles = map[string][]string{"reviewer": {"opus"}}
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Role: "reviewer", Driver: "claude", Model: "sonnet"}
		err := ValidateSpawn(cfg, req, 0)
		if err == nil {
			t.Fatal("expected role constraint violation to be rejected")
		}
		if got := err.Error(); !strings.Contains(got, "not permitted for role") {
			t.Errorf("expected role rejection message, got: %s", got)
		}
	})

	t.Run("role constraint satisfied ok", func(t *testing.T) {
		cfg := testOrgConfig()
		cfg.Roles = map[string][]string{"reviewer": {"opus"}}
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Role: "reviewer", Driver: "claude", Model: "opus"}
		if err := ValidateSpawn(cfg, req, 0); err != nil {
			t.Fatalf("expected role-satisfying spawn to be allowed, got error: %v", err)
		}
	})

	t.Run("empty role falls back to pool-wide ok even with roles configured", func(t *testing.T) {
		cfg := testOrgConfig()
		cfg.Roles = map[string][]string{"reviewer": {"opus"}}
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-1", Role: "", Driver: "claude", Model: "haiku"}
		if err := ValidateSpawn(cfg, req, 0); err != nil {
			t.Fatalf("expected unrestricted role to allow full pool, got error: %v", err)
		}
	})

	t.Run("max_seats at limit rejected", func(t *testing.T) {
		cfg := testOrgConfig()
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-4", Driver: "claude", Model: "sonnet"}
		err := ValidateSpawn(cfg, req, cfg.MaxSeats)
		if err == nil {
			t.Fatal("expected spawn at max_seats limit to be rejected")
		}
		if got := err.Error(); !strings.Contains(got, "max_seats") {
			t.Errorf("expected max_seats rejection message, got: %s", got)
		}
	})

	t.Run("max_seats below limit ok", func(t *testing.T) {
		cfg := testOrgConfig()
		req := SpawnRequest{OrgID: "org-a", SeatID: "seat-3", Driver: "claude", Model: "sonnet"}
		if err := ValidateSpawn(cfg, req, cfg.MaxSeats-1); err != nil {
			t.Fatalf("expected spawn below max_seats to be allowed, got error: %v", err)
		}
	})

	t.Run("seats in a different org_id are not counted", func(t *testing.T) {
		// Build real manifest events for two org_id namespaces: "org-a" at
		// its max_seats cap, "org-b" empty. ActiveSeatCount must scope by
		// org_id so org-b's spawn is evaluated against 0, not 3 (AC-2).
		cfg := testOrgConfig()
		events := []ManifestEvent{
			{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
			{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-2", Event: EventSpawned},
			{TS: "2026-08-01T00:00:02Z", OrgID: "org-a", SeatID: "seat-3", Event: EventSpawned},
		}

		orgAActive := ActiveSeatCount(events, "org-a", RosterOptions{})
		if orgAActive != cfg.MaxSeats {
			t.Fatalf("expected org-a active seat count %d, got %d", cfg.MaxSeats, orgAActive)
		}
		orgBActive := ActiveSeatCount(events, "org-b", RosterOptions{})
		if orgBActive != 0 {
			t.Fatalf("expected org-b active seat count 0 (unaffected by org-a), got %d", orgBActive)
		}

		reqOrgA := SpawnRequest{OrgID: "org-a", SeatID: "seat-4", Driver: "claude", Model: "sonnet"}
		if err := ValidateSpawn(cfg, reqOrgA, orgAActive); err == nil {
			t.Fatal("expected org-a spawn to be rejected at max_seats")
		}

		reqOrgB := SpawnRequest{OrgID: "org-b", SeatID: "seat-1", Driver: "claude", Model: "sonnet"}
		if err := ValidateSpawn(cfg, reqOrgB, orgBActive); err != nil {
			t.Fatalf("expected org-b spawn to be unaffected by org-a's seat count, got error: %v", err)
		}
	})
}
