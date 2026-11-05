package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestEnvelopeSummary_ListsRoleModelsMaxSeatsAndPermissionDefault pins the
// summary's exact text (plan 2026-11-05-org-role-models, Scope "doctor" and
// FR-7's {{ENVELOPE}}): each role of [org.role_models] as
// <role>=<driver>/<model>, sorted by role name, a model of "" as
// "(CLI default)", then max_seats and the permission default. The roles come
// from a map, so the same config must render byte-identically every time,
// whatever order the map yields them in.
func TestEnvelopeSummary_ListsRoleModelsMaxSeatsAndPermissionDefault(t *testing.T) {
	cfg := config.OrgConfig{
		RoleModels: map[string]config.OrgRoleModel{
			"reviewer":    {Driver: "codex", Model: "gpt-6.1-sol"},
			"leader":      {Driver: "claude", Model: "opus"},
			"director":    {Driver: "claude", Model: ""},
			"implementer": {Driver: "claude", Model: "sonnet"},
		},
		MaxSeats:    5,
		Permissions: config.OrgPermissionsConfig{Default: "autonomous"},
	}
	const want = "role_models: director=claude/(CLI default), implementer=claude/sonnet, leader=claude/opus, reviewer=codex/gpt-6.1-sol" +
		" | max_seats: 5 | permission default: autonomous"
	for i := range 20 {
		if got := EnvelopeSummary(cfg); got != want {
			t.Fatalf("EnvelopeSummary() call %d = %q, want %q", i, got, want)
		}
	}
}

// TestEnvelopeSummary_DefaultRoleModels pins the summary of config.Default's
// [org.role_models] (the claude set), which `ralph org start` renders into
// the leader's prompt when ralph.toml sets no entry.
func TestEnvelopeSummary_DefaultRoleModels(t *testing.T) {
	got := EnvelopeSummary(config.Default().Org)
	const wantModels = "role_models: director=claude/(CLI default), implementer=claude/opus, leader=claude/opus, reviewer=claude/opus |"
	if !strings.HasPrefix(got, wantModels) {
		t.Fatalf("EnvelopeSummary(config.Default().Org) = %q, want it to start with %q", got, wantModels)
	}
}

func TestEnvelopeSummary_EmptyRoleModels_NoneConfiguredMarker(t *testing.T) {
	cfg := config.OrgConfig{MaxSeats: 1, Permissions: config.OrgPermissionsConfig{Default: "guarded"}}
	got := EnvelopeSummary(cfg)
	if want := "role_models: (none configured) | max_seats: 1 | permission default: guarded"; got != want {
		t.Errorf("EnvelopeSummary() = %q, want %q for an empty [org.role_models]", got, want)
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
