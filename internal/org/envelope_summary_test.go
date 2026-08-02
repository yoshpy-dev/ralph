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
