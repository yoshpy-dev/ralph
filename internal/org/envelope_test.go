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
