package org

import (
	"os"
	"path/filepath"
	"slices"
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
			if !strings.Contains(err.Error(), "requires [org.permissions].codex_verified=true") || !strings.Contains(err.Error(), "guarded") {
				t.Errorf("expected fail-closed error to require codex_verified and mention guarded, got %v", err)
			}
		})
	}

	// TestPermissionArgsForDriver_Codex_UnknownMode_DistinctError pins the
	// self-review LOW fix: an unrecognized mode string must be reported as
	// "unknown permission mode" (the same wording the claude branch's own
	// default case uses), not folded into the "requires [org.permissions].codex_verified=true"
	// fail-closed message a genuinely known-but-unverified mode gets.
	t.Run("an unknown mode is reported distinctly from the fail-closed case", func(t *testing.T) {
		_, err := permissionArgsForDriver(config.OrgConfig{}, "codex", "not-a-real-mode")
		if err == nil {
			t.Fatalf("permissionArgsForDriver(codex, not-a-real-mode): expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown permission mode") {
			t.Errorf("expected an 'unknown permission mode' error, got %v", err)
		}
		if strings.Contains(err.Error(), "requires [org.permissions].codex_verified=true") {
			t.Errorf("did not expect the fail-closed wording for a genuinely unknown mode, got %v", err)
		}
	})
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

// TestCodexWritableRootArgs pins AC8 (plan 2026-10-07-org-state-dir-common):
// a workspace-write codex leader seat (autonomous or edits) gets --add-dir
// <state dir> only when the state dir is outside its cwd, compared with
// symlinks resolved; implementer, reviewer, and unknown roles, guarded codex
// seats, and claude seats never get it.
func TestCodexWritableRootArgs(t *testing.T) {
	root := t.TempDir()
	mainWT := filepath.Join(root, "main")
	linkedWT := filepath.Join(mainWT, ".claude", "worktrees", "slug")
	stateDir := filepath.Join(mainWT, ".harness", "state", "org")
	// siblingStateDir's path string starts with mainWT's, but it is not
	// inside mainWT.
	siblingStateDir := filepath.Join(root, "main-other", ".harness", "state", "org")
	for _, d := range []string{linkedWT, stateDir, siblingStateDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	// mainLink is a second spelling of mainWT through a symlink, the
	// portable analogue of macOS's /var -> /private/var.
	mainLink := filepath.Join(root, "main-link")
	if err := os.Symlink(mainWT, mainLink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolvedMain, err := filepath.EvalSymlinks(mainWT)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	addDir := []string{"--add-dir", stateDir}
	leader := LeaderIdentity

	cases := []struct {
		name     string
		role     string
		driver   string
		mode     string
		cwd      string
		stateDir string
		want     []string
	}{
		{"leader codex edits, cwd is a linked worktree outside the state dir", leader, "codex", PermissionModeEdits, linkedWT, stateDir, addDir},
		{"leader codex autonomous, cwd is a linked worktree outside the state dir", leader, "codex", PermissionModeAutonomous, linkedWT, stateDir, addDir},
		{"leader codex edits, a sibling dir sharing cwd's name prefix is outside", leader, "codex", PermissionModeEdits, mainWT, siblingStateDir, []string{"--add-dir", siblingStateDir}},
		{"leader codex edits, state dir under cwd", leader, "codex", PermissionModeEdits, mainWT, stateDir, nil},
		{"leader codex autonomous, state dir under cwd", leader, "codex", PermissionModeAutonomous, mainWT, stateDir, nil},
		{"leader codex autonomous, state dir is cwd itself", leader, "codex", PermissionModeAutonomous, stateDir, stateDir, nil},
		{"leader codex edits, cwd spelled through a symlink", leader, "codex", PermissionModeEdits, mainLink, stateDir, nil},
		{"leader codex edits, state dir spelled through a symlink", leader, "codex", PermissionModeEdits, mainWT, filepath.Join(mainLink, ".harness", "state", "org"), nil},
		{"leader codex edits, cwd symlink-resolved and state dir not", leader, "codex", PermissionModeEdits, resolvedMain, stateDir, nil},
		{"leader codex guarded, state dir outside cwd", leader, "codex", PermissionModeGuarded, linkedWT, stateDir, nil},
		{"leader claude autonomous, state dir outside cwd", leader, "claude", PermissionModeAutonomous, linkedWT, stateDir, nil},
		{"leader claude edits, state dir outside cwd", leader, "claude", PermissionModeEdits, linkedWT, stateDir, nil},
		{"implementer codex autonomous, state dir outside cwd", "implementer", "codex", PermissionModeAutonomous, linkedWT, stateDir, nil},
		{"implementer codex edits, state dir outside cwd", "implementer", "codex", PermissionModeEdits, linkedWT, stateDir, nil},
		{"reviewer codex autonomous, state dir outside cwd", "reviewer", "codex", PermissionModeAutonomous, linkedWT, stateDir, nil},
		{"reviewer codex edits, state dir outside cwd", "reviewer", "codex", PermissionModeEdits, linkedWT, stateDir, nil},
		{"unknown role codex autonomous, state dir outside cwd", "worker", "codex", PermissionModeAutonomous, linkedWT, stateDir, nil},
		{"empty role codex autonomous, state dir outside cwd", "", "codex", PermissionModeAutonomous, linkedWT, stateDir, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := codexWritableRootArgs(tc.role, tc.driver, tc.mode, tc.cwd, tc.stateDir)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("codexWritableRootArgs(%q, %q, %q, %q, %q) = %v, want %v", tc.role, tc.driver, tc.mode, tc.cwd, tc.stateDir, got, tc.want)
			}
		})
	}
}
