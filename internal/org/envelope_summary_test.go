package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

func TestEnvelopeSummary_ListsModelPoolMaxSeatsAndPermissionDefault(t *testing.T) {
	cfg := config.OrgConfig{
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "claude", Model: "opus"},
			{Driver: "claude", Model: "sonnet"},
			{Driver: "codex", Model: "gpt-5-codex"},
		},
		MaxSeats:    5,
		Permissions: config.OrgPermissionsConfig{Default: "autonomous"},
	}
	got := EnvelopeSummary(cfg)
	for _, want := range []string{"claude/opus", "claude/sonnet", "codex/gpt-5-codex", "max_seats: 5", "permission default: autonomous"} {
		if !strings.Contains(got, want) {
			t.Errorf("EnvelopeSummary() = %q, missing %q", got, want)
		}
	}
	// Declared model_pool order must be preserved, not re-sorted.
	if strings.Index(got, "claude/opus") > strings.Index(got, "claude/sonnet") {
		t.Errorf("EnvelopeSummary() = %q, expected model_pool order preserved (opus before sonnet)", got)
	}
}

func TestEnvelopeSummary_EmptyModelPool_NoneConfiguredMarker(t *testing.T) {
	cfg := config.OrgConfig{MaxSeats: 1, Permissions: config.OrgPermissionsConfig{Default: "guarded"}}
	got := EnvelopeSummary(cfg)
	if !strings.Contains(got, "(none configured)") {
		t.Errorf("EnvelopeSummary() = %q, want a %q marker for an empty model_pool", got, "(none configured)")
	}
}

func TestEnvelopeSummary_BlankPermissionDefault_FallsBackToPackageDefault(t *testing.T) {
	cfg := config.OrgConfig{MaxSeats: 2}
	got := EnvelopeSummary(cfg)
	if !strings.Contains(got, "permission default: "+defaultPermissionMode) {
		t.Errorf("EnvelopeSummary() = %q, want fallback %q", got, defaultPermissionMode)
	}
}

func TestDefaultModelForDriver_ReturnsFirstMatchingPoolEntry(t *testing.T) {
	cfg := config.OrgConfig{
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "codex", Model: "gpt-5-codex"},
			{Driver: "claude", Model: "opus"},
			{Driver: "claude", Model: "sonnet"},
		},
	}
	got, err := DefaultModelForDriver(cfg, "claude")
	if err != nil {
		t.Fatalf("DefaultModelForDriver: unexpected error: %v", err)
	}
	if got != "opus" {
		t.Fatalf("DefaultModelForDriver(claude) = %q, want %q (first matching entry)", got, "opus")
	}
}

func TestDefaultModelForDriver_NoMatchingDriver_ReturnsError(t *testing.T) {
	cfg := config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{{Driver: "codex", Model: "gpt-5-codex"}}}
	_, err := DefaultModelForDriver(cfg, "claude")
	if err == nil {
		t.Fatal("expected an error when no model_pool entry matches driver")
	}
	if !strings.Contains(err.Error(), `driver "claude"`) {
		t.Errorf("expected error to name the unmatched driver, got: %v", err)
	}
}

func TestDefaultModelForDriver_EmptyModelPool_ReturnsError(t *testing.T) {
	if _, err := DefaultModelForDriver(config.OrgConfig{}, "claude"); err == nil {
		t.Fatal("expected an error for an empty model_pool")
	}
}

// TestDefaultModelForDriver_DefaultPoolHeads verifies the shipped
// config.Default().Org.ModelPool's per-driver head entries: claude's first
// entry is the "fable" alias and codex's first entry is the "gpt-6-astra"
// slug, per the model_pool refresh (AC-3).
func TestDefaultModelForDriver_DefaultPoolHeads(t *testing.T) {
	cfg := config.Default().Org
	if got, err := DefaultModelForDriver(cfg, "claude"); err != nil {
		t.Fatalf("DefaultModelForDriver(claude): unexpected error: %v", err)
	} else if got != "fable" {
		t.Errorf("DefaultModelForDriver(claude) = %q, want %q", got, "fable")
	}
	if got, err := DefaultModelForDriver(cfg, "codex"); err != nil {
		t.Fatalf("DefaultModelForDriver(codex): unexpected error: %v", err)
	} else if got != "gpt-6-astra" {
		t.Errorf("DefaultModelForDriver(codex) = %q, want %q", got, "gpt-6-astra")
	}
}

// TestDefaultModelForDriverAndRole_UnrestrictedRole_ReturnsFirstMatchingPoolEntry
// covers a role absent from [org.roles] (or config with no [org.roles] at
// all): the full model_pool is allowed for that role, so the result matches
// plain DefaultModelForDriver.
func TestDefaultModelForDriverAndRole_UnrestrictedRole_ReturnsFirstMatchingPoolEntry(t *testing.T) {
	cfg := config.OrgConfig{
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "claude", Model: "fable"},
			{Driver: "claude", Model: "opus"},
			{Driver: "claude", Model: "sonnet"},
		},
	}
	got, err := DefaultModelForDriverAndRole(cfg, "claude", "implementer")
	if err != nil {
		t.Fatalf("DefaultModelForDriverAndRole: unexpected error: %v", err)
	}
	if got != "fable" {
		t.Fatalf("DefaultModelForDriverAndRole(claude, implementer) = %q, want %q (first matching entry, no role restriction)", got, "fable")
	}
}

// TestDefaultModelForDriverAndRole_RoleRestricted_SkipsImpermissiblePoolHead
// covers the self-review MEDIUM-2 scenario: [org.roles] restricts
// "implementer" to "sonnet" while the claude pool head is "fable" -- the
// fallback must skip "fable" and land on the first permitted entry
// ("sonnet"), not warn-then-reject.
func TestDefaultModelForDriverAndRole_RoleRestricted_SkipsImpermissiblePoolHead(t *testing.T) {
	cfg := config.OrgConfig{
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "claude", Model: "fable"},
			{Driver: "claude", Model: "opus"},
			{Driver: "claude", Model: "sonnet"},
		},
		Roles: map[string][]string{"implementer": {"sonnet"}},
	}
	got, err := DefaultModelForDriverAndRole(cfg, "claude", "implementer")
	if err != nil {
		t.Fatalf("DefaultModelForDriverAndRole: unexpected error: %v", err)
	}
	if got != "sonnet" {
		t.Fatalf("DefaultModelForDriverAndRole(claude, implementer) = %q, want %q (first role-permitted entry)", got, "sonnet")
	}
}

// TestDefaultModelForDriverAndRole_NoPermittedEntryForRole_ReturnsError
// covers a role restricted to models that exist only for a different
// driver: no claude entry is permitted for "reviewer", so the fallback must
// error rather than silently pick an impermissible model.
func TestDefaultModelForDriverAndRole_NoPermittedEntryForRole_ReturnsError(t *testing.T) {
	cfg := config.OrgConfig{
		ModelPool: []config.OrgModelPoolEntry{
			{Driver: "claude", Model: "fable"},
			{Driver: "claude", Model: "opus"},
			{Driver: "codex", Model: "gpt-6-astra"},
		},
		Roles: map[string][]string{"reviewer": {"gpt-6-astra"}},
	}
	_, err := DefaultModelForDriverAndRole(cfg, "claude", "reviewer")
	if err == nil {
		t.Fatal("expected an error when no model_pool entry for driver is permitted for role")
	}
	if !strings.Contains(err.Error(), `driver "claude"`) || !strings.Contains(err.Error(), `role "reviewer"`) {
		t.Errorf("expected error to name both the driver and role, got: %v", err)
	}
}
