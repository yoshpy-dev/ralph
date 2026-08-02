package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

func TestResolvePermissionMode(t *testing.T) {
	t.Run("default applies when no role override", func(t *testing.T) {
		cfg := config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "edits"}}
		if got := ResolvePermissionMode(cfg, "worker"); got != "edits" {
			t.Fatalf("ResolvePermissionMode = %q, want edits", got)
		}
	})

	t.Run("role override wins over default", func(t *testing.T) {
		cfg := config.OrgConfig{Permissions: config.OrgPermissionsConfig{
			Default: "autonomous",
			Roles:   map[string]string{"reviewer": "guarded"},
		}}
		if got := ResolvePermissionMode(cfg, "reviewer"); got != "guarded" {
			t.Fatalf("ResolvePermissionMode = %q, want guarded", got)
		}
	})

	t.Run("unknown role falls back to default", func(t *testing.T) {
		cfg := config.OrgConfig{Permissions: config.OrgPermissionsConfig{
			Default: "edits",
			Roles:   map[string]string{"reviewer": "guarded"},
		}}
		if got := ResolvePermissionMode(cfg, "not-a-known-role"); got != "edits" {
			t.Fatalf("ResolvePermissionMode = %q, want edits (default)", got)
		}
	})

	t.Run("zero-value config falls back to autonomous", func(t *testing.T) {
		// A hand-built config.OrgConfig{} (never went through config.Load(),
		// which backfills Permissions.Default to "autonomous") must still
		// resolve to the documented default -- this is what testOrgConfig()
		// relies on implicitly across the rest of this package's tests.
		var cfg config.OrgConfig
		if got := ResolvePermissionMode(cfg, "worker"); got != "autonomous" {
			t.Fatalf("ResolvePermissionMode = %q, want autonomous", got)
		}
	})
}

func TestPermissionArgsForDriver_Claude(t *testing.T) {
	cases := []struct {
		mode string
		want []string
	}{
		{"autonomous", []string{"--permission-mode", "bypassPermissions"}},
		{"edits", []string{"--permission-mode", "acceptEdits"}},
		{"guarded", nil},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			got, err := permissionArgsForDriver(config.OrgConfig{}, "claude", tc.mode)
			if err != nil {
				t.Fatalf("permissionArgsForDriver(claude, %q): unexpected error %v", tc.mode, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("permissionArgsForDriver(claude, %q) = %v, want %v", tc.mode, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("permissionArgsForDriver(claude, %q) = %v, want %v", tc.mode, got, tc.want)
				}
			}
		})
	}
}

func TestPermissionArgsForDriver_Codex_FailClosed(t *testing.T) {
	t.Run("guarded is allowed with no flags", func(t *testing.T) {
		got, err := permissionArgsForDriver(config.OrgConfig{}, "codex", "guarded")
		if err != nil {
			t.Fatalf("permissionArgsForDriver(codex, guarded): unexpected error %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("permissionArgsForDriver(codex, guarded) = %v, want no flags", got)
		}
	})

	for _, mode := range []string{"autonomous", "edits"} {
		t.Run(mode+" is rejected fail-closed when codex_verified is false", func(t *testing.T) {
			_, err := permissionArgsForDriver(config.OrgConfig{}, "codex", mode)
			if err == nil {
				t.Fatalf("permissionArgsForDriver(codex, %q): expected fail-closed error, got nil", mode)
			}
			if !strings.Contains(err.Error(), "not yet live-verified") || !strings.Contains(err.Error(), "guarded") {
				t.Errorf("expected fail-closed error to mention live-verification and guarded, got %v", err)
			}
		})
	}
}

// TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping pins AC-8: once
// [org.permissions].codex_verified is true, codex's autonomous/edits modes
// resolve to the live-verified CLI flags instead of erroring; guarded is
// unaffected either way.
func TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping(t *testing.T) {
	verifiedCfg := config.OrgConfig{Permissions: config.OrgPermissionsConfig{CodexVerified: true}}

	t.Run("autonomous", func(t *testing.T) {
		got, err := permissionArgsForDriver(verifiedCfg, "codex", "autonomous")
		if err != nil {
			t.Fatalf("permissionArgsForDriver(codex, autonomous) with codex_verified=true: unexpected error %v", err)
		}
		want := []string{"--sandbox", "workspace-write", "--ask-for-approval", "never"}
		if len(got) != len(want) {
			t.Fatalf("permissionArgsForDriver(codex, autonomous) = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("permissionArgsForDriver(codex, autonomous) = %v, want %v", got, want)
			}
		}
	})

	t.Run("edits", func(t *testing.T) {
		got, err := permissionArgsForDriver(verifiedCfg, "codex", "edits")
		if err != nil {
			t.Fatalf("permissionArgsForDriver(codex, edits) with codex_verified=true: unexpected error %v", err)
		}
		want := []string{"--sandbox", "workspace-write"}
		if len(got) != len(want) {
			t.Fatalf("permissionArgsForDriver(codex, edits) = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("permissionArgsForDriver(codex, edits) = %v, want %v", got, want)
			}
		}
	})

	t.Run("guarded still needs no flags", func(t *testing.T) {
		got, err := permissionArgsForDriver(verifiedCfg, "codex", "guarded")
		if err != nil {
			t.Fatalf("permissionArgsForDriver(codex, guarded) with codex_verified=true: unexpected error %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("permissionArgsForDriver(codex, guarded) = %v, want no flags", got)
		}
	})
}
