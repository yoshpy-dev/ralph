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
