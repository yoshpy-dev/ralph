package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// codexSandboxTestEnv returns a resolveEnv closure fixed to the given
// config path and AGMSG_STORAGE_PATH override (Home left ""), for tests
// that don't want to mutate process environment variables at all.
func codexSandboxTestEnv(configPath, storageOverride string) func() (codexSandboxEnv, error) {
	return func() (codexSandboxEnv, error) {
		return codexSandboxEnv{ConfigPath: configPath, AgmsgStoragePath: storageOverride}, nil
	}
}

// writeCodexConfig writes content to a config.toml under dir, creating
// parent directories as needed, and returns its path.
func writeCodexConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// codexOrgConfig builds a minimal config.OrgConfig fixture for
// checkCodexAgmsgWritableRoot's tests. DriverPool always includes "codex"
// (so a codex seat can be spawned at all and the check proceeds past
// outcome 2, the driver_pool check) -- a test that specifically wants to
// exercise outcome 2 itself builds a config.OrgConfig by hand instead.
// defaultMode is
// [org.permissions].default ("" leaves it unset, which
// org.ResolvePermissionMode then falls back to "autonomous" for, matching
// config.Default()); roles is [org.permissions].roles, may be nil.
func codexOrgConfig(codexVerified bool, defaultMode string, roles map[string]string) config.OrgConfig {
	return config.OrgConfig{
		DriverPool: []string{"claude", "codex"},
		Permissions: config.OrgPermissionsConfig{
			CodexVerified: codexVerified,
			Default:       defaultMode,
			Roles:         roles,
		},
	}
}

func TestCheckCodexAgmsgWritableRoot_NotNeeded_CodexVerifiedFalse_NoGuardedRole(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml") // absent -- irrelevant here, guardedRole fails first.

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(false, "", nil), agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	want := "not needed: [org.permissions].codex_verified is false and no role resolves to guarded"
	if r.Detail != want {
		t.Errorf("Detail = %q, want %q", r.Detail, want)
	}
}

// TestCheckCodexAgmsgWritableRoot_NotNeeded_GuardedDefaultConfigAbsent covers
// "codex_verified = true + default guarded + no roles -> reason (a) absent"
// plus why-b's "config does not exist" branch in the same fixture.
func TestCheckCodexAgmsgWritableRoot_NotNeeded_GuardedDefaultConfigAbsent(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "guarded", nil), agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	want := fmt.Sprintf("not needed: no role resolves to edits or autonomous and %s does not exist", noSuchCfg)
	if r.Detail != want {
		t.Errorf("Detail = %q, want %q", r.Detail, want)
	}
}

// TestCheckCodexAgmsgWritableRoot_NotNeeded_GuardedDefaultConfigExists covers
// why-b's third branch: a guarded-capable role exists and the config
// exists, but it doesn't set sandbox_mode = "workspace-write".
func TestCheckCodexAgmsgWritableRoot_NotNeeded_GuardedDefaultConfigExists(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "profile = \"x\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "guarded", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	want := fmt.Sprintf("not needed: no role resolves to edits or autonomous and %s does not set sandbox_mode = \"workspace-write\"", cfgPath)
	if r.Detail != want {
		t.Errorf("Detail = %q, want %q", r.Detail, want)
	}
}

// TestCheckCodexAgmsgWritableRoot_NotNeeded_SandboxModeSetButNoGuardedRole is
// the explicit case: sandbox_mode = "workspace-write" is set, the default
// mode is autonomous (no guarded-capable role anywhere), so reason (b) is
// absent despite the config setting the key -- proving guardedRole gates
// it, not sandbox_mode alone.
func TestCheckCodexAgmsgWritableRoot_NotNeeded_SandboxModeSetButNoGuardedRole(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(false, "autonomous", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "no role resolves to guarded") {
		t.Errorf("expected detail to name the missing guarded role, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	wantStore := filepath.Join(agmsgHome, "db")
	for _, want := range []string{wantStore, cfgPath, "docs/recipes/codex-seat-permissions.md", `"attempt to write a readonly database"`} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}

// TestCheckCodexAgmsgWritableRoot_BothReasons_WarnDetailNumbersEachReason is
// self-review NEW-1's regression test: each reason already contains its own
// " and " (the role condition), so when both fire they must be numbered
// (1)/(2) rather than joined with a third, indistinguishable " and ".
func TestCheckCodexAgmsgWritableRoot_BothReasons_WarnDetailNumbersEachReason(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")
	store := filepath.Join(agmsgHome, "db")

	orgCfg := codexOrgConfig(true, "guarded", map[string]string{"implementer": "autonomous"})
	r := checkCodexAgmsgWritableRoot(orgCfg, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	want := "needed because (1) [org.permissions].codex_verified = true and a role resolves to edits or autonomous " +
		"(ralph passes --sandbox workspace-write to those codex seats) and (2) sandbox_mode = \"workspace-write\" in " + cfgPath +
		" and a role resolves to guarded (guarded codex seats inherit it); add " + store
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_RootEqualsStore_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	store := filepath.Join(agmsgHome, "db")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+store+"\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, store) {
		t.Errorf("expected detail to contain the store %q, got: %s", store, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_RootIsAncestor_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+agmsgHome+"\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass (agmsg home is an ancestor of its db dir), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_PrefixSiblingNotCovered pins the
// path-element-boundary rule: a root that merely shares a string prefix
// with the store (…/dbx vs …/db, or /a/b vs /a/bc) must not be treated as
// covering it.
func TestCheckCodexAgmsgWritableRoot_PrefixSiblingNotCovered(t *testing.T) {
	t.Run("dbx sibling of db", func(t *testing.T) {
		agmsgHome := t.TempDir()
		writeAgmsgHome(t, agmsgHome, "")
		sibling := agmsgHome + "x" // …/db becomes …/dbx as a *root*, not a parent.
		store := filepath.Join(agmsgHome, "db")
		cfgDir := t.TempDir()
		cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+filepath.Join(sibling, "db")+"\"]\n")

		r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
		if r.Status != "warn" {
			t.Fatalf("expected warn, got %s (%s) store=%s", r.Status, r.Detail, store)
		}
	})

	t.Run("/a/b does not cover /a/bc", func(t *testing.T) {
		if !pathCovers("/a/b", "/a/b") {
			t.Errorf("expected /a/b to cover itself")
		}
		if pathCovers("/a/b", "/a/bc") {
			t.Errorf("/a/b must not cover /a/bc")
		}
	})
}

func TestCheckCodexAgmsgWritableRoot_SymlinkedRoot_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	store := filepath.Join(agmsgHome, "db")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "db-link")
	if err := os.Symlink(store, link); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+link+"\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass (symlinked root resolves to the store), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_StoreSymlinkedOutsideConfiguredRoot_Warn is
// self-review MEDIUM-3's regression test: the agmsg store directory is
// itself a symlink pointing OUTSIDE every configured root. codex's sandbox
// evaluates the real (resolved) path, so this must warn, not pass on a
// textual match with the symlink's own path.
func TestCheckCodexAgmsgWritableRoot_StoreSymlinkedOutsideConfiguredRoot_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	elsewhere := t.TempDir()
	store := filepath.Join(agmsgHome, "db")
	if err := os.Symlink(elsewhere, store); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+agmsgHome+"\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn (the store symlinks outside every configured root), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_StoreSymlinkedRealTargetAlsoListed_Pass is
// the symlinked-store test's counterpart: when the real target IS also
// listed as a writable root, the check passes and names that root (not the
// symlink path the store happens to live under).
func TestCheckCodexAgmsgWritableRoot_StoreSymlinkedRealTargetAlsoListed_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	elsewhere := t.TempDir()
	store := filepath.Join(agmsgHome, "db")
	if err := os.Symlink(elsewhere, store); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+agmsgHome+"\", \""+elsewhere+"\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, elsewhere) {
		t.Errorf("expected detail to name the real target root %q, got: %s", elsewhere, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_GuardedSeatSandboxModeNoRoots_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(false, "guarded", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "guarded codex seats inherit it") {
		t.Errorf("expected detail to mention guarded seats, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_EmptyStringRoleKeyDoesNotSuppressWarn is
// self-review NEW-2's end-to-end regression test, reproducing the exact
// finding: default = "guarded" plus a roles entry keyed by the empty
// string used to intercept ResolvePermissionMode's default lookup and
// silently disable reason (b), turning a should-warn into a pass. With the
// fix, guardedRole is still true (from the "guarded" default), so this
// still warns.
func TestCheckCodexAgmsgWritableRoot_EmptyStringRoleKeyDoesNotSuppressWarn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	orgCfg := codexOrgConfig(false, "guarded", map[string]string{"": "edits"})
	r := checkCodexAgmsgWritableRoot(orgCfg, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn (the empty-string role key must not shadow the \"guarded\" default), got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "guarded codex seats inherit it") {
		t.Errorf("expected detail to mention guarded seats, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_MissingConfigCodexVerified_WarnNamesConfigAbsent
// is self-review LOW-4's regression test: the "not needed" all-clear must
// not silently apply to a warn -- when the config is absent, the warn
// Detail must say so explicitly rather than looking identical to a config
// that exists but simply omits sandbox_mode.
func TestCheckCodexAgmsgWritableRoot_MissingConfigCodexVerified_WarnNamesConfigAbsent(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	store := filepath.Join(agmsgHome, "db")
	for _, want := range []string{
		noSuchCfg + " does not exist, so no writable root covers the agmsg store " + store,
		"writable_roots in " + noSuchCfg,
	} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
	// self-review NEW-3: the old wording attached "(<cfg> does not exist)"
	// right after the store path, reading as a claim about the store.
	if strings.Contains(r.Detail, "("+noSuchCfg+" does not exist)") {
		t.Errorf("the does-not-exist clause must not be parenthesized after the store path, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_HomeSubstitution_TildeInDetail is
// self-review MEDIUM-2's regression test: the "~" substitution must come
// from the injected seam's Home field, not a direct os.UserHomeDir() call
// -- proven here with a Home that is a temp dir, never the real developer
// home, so the substitution can only have happened through the seam.
func TestCheckCodexAgmsgWritableRoot_HomeSubstitution_TildeInDetail(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	home := t.TempDir()
	cfgPath := writeCodexConfig(t, filepath.Join(home, ".codex"), "")

	resolveEnv := func() (codexSandboxEnv, error) {
		return codexSandboxEnv{Home: home, ConfigPath: cfgPath}, nil
	}
	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, resolveEnv)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	wantTilde := "~" + string(filepath.Separator) + ".codex" + string(filepath.Separator) + "config.toml"
	if !strings.Contains(r.Detail, wantTilde) {
		t.Errorf("expected detail to contain %q (home substitution via the injected seam), got: %s", wantTilde, r.Detail)
	}
	if strings.Contains(r.Detail, home) {
		t.Errorf("the raw home-prefixed path must not appear once substituted, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_MalformedTOML_InfoWithoutContent proves the
// TOML decode-error branch reports only the line/column, never any part of
// the config's own content -- which can hold secrets.
func TestCheckCodexAgmsgWritableRoot_MalformedTOML_InfoWithoutContent(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	// Unterminated table header -- invalid TOML syntax. The canary string
	// must never leak into the Detail.
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write\nwritable_roots = [\"CANARYLEAK123\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"could not be decoded as a codex config", "line", "column"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
	if strings.Contains(r.Detail, "CANARYLEAK123") {
		t.Errorf("config content must never leak into Detail, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_DuplicateKey_InfoWithoutKeyNameOrMessage is
// self-review MEDIUM-1's regression test: go-toml v2.3.0 raises a
// duplicate-key error as a plain (non-*toml.DecodeError) error whose text
// repeats the key's own name -- exactly what must never reach the Detail.
func TestCheckCodexAgmsgWritableRoot_DuplicateKey_InfoWithoutKeyNameOrMessage(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "note = \"CANARYLEAK123\"\nnote = \"CANARYLEAK123\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not be decoded as a codex config") {
		t.Errorf("expected the decode-failure wording, got: %s", r.Detail)
	}
	for _, notWant := range []string{"CANARYLEAK123", "note", "already defined"} {
		if strings.Contains(r.Detail, notWant) {
			t.Errorf("config key/value text must never leak into Detail (found %q), got: %s", notWant, r.Detail)
		}
	}
}

// TestCheckCodexAgmsgWritableRoot_TypeMismatch_InfoWithPosition is
// self-review LOW-5's regression test: a type mismatch (sandbox_mode = 5)
// is syntactically valid TOML that go-toml still rejects as a
// *toml.DecodeError (with a position), so it must be worded as a decode
// failure, never "not valid TOML".
func TestCheckCodexAgmsgWritableRoot_TypeMismatch_InfoWithPosition(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = 5\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"could not be decoded as a codex config", "line", "column"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
	if strings.Contains(r.Detail, "not valid TOML") {
		t.Errorf("a type mismatch in otherwise-valid TOML must not be called \"not valid TOML\", got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_WritableRootsWrongType_Info(t *testing.T) {
	cases := []struct {
		name string
		toml string
	}{
		{"string instead of array", "[sandbox_workspace_write]\nwritable_roots = \"x\"\n"},
		{"array of ints", "[sandbox_workspace_write]\nwritable_roots = [1, 2]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agmsgHome := t.TempDir()
			writeAgmsgHome(t, agmsgHome, "")
			cfgDir := t.TempDir()
			cfgPath := writeCodexConfig(t, cfgDir, tc.toml)

			r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
			if r.Status != "info" {
				t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, "writable_roots is not an array of strings") {
				t.Errorf("expected detail to name the type problem, got: %s", r.Detail)
			}
		})
	}
}

// TestCheckCodexAgmsgWritableRoot_CodexNotInDriverPool_Pass is self-review
// LOW-6's outcome 2 regression test: codex is not spawnable at all when
// it's missing from [org].driver_pool, so the check passes immediately --
// even with codex_verified = true and no config to read.
func TestCheckCodexAgmsgWritableRoot_CodexNotInDriverPool_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	orgCfg := config.OrgConfig{
		DriverPool: []string{"claude"},
		Permissions: config.OrgPermissionsConfig{
			CodexVerified: true,
			Default:       "autonomous",
		},
	}
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml") // never read.

	r := checkCodexAgmsgWritableRoot(orgCfg, agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "[org].driver_pool") {
		t.Errorf("expected detail to mention driver_pool, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_AgmsgMissing_Info(t *testing.T) {
	agmsgHome := filepath.Join(t.TempDir(), "no-such-agmsg-home")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "agmsg not installed at "+agmsgHome) {
		t.Errorf("expected detail to name the resolved home, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded proves
// grading uses driver.AgmsgAvailable, not the separate "agmsg" check's
// Status (which is "info" for a differently-versioned install): with roots
// missing, a version-mismatched-but-installed agmsg home still warns.
func TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "9.9.9") // differs from agmsgTestedVersion.
	if av := checkAgmsgAvailable(agmsgHome); av.Status != "info" {
		t.Fatalf("test setup: expected the agmsg check itself to be info for a version mismatch, got %s", av.Status)
	}

	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn despite the agmsg version mismatch (install itself is fine), got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_StorageOverride(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	overrideDir := t.TempDir()
	override := filepath.Join(overrideDir, "custom-store")

	t.Run("root covering only the default db dir warns", func(t *testing.T) {
		defaultStore := filepath.Join(agmsgHome, "db")
		cfgDir := t.TempDir()
		cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+defaultStore+"\"]\n")

		r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, override))
		if r.Status != "warn" {
			t.Fatalf("expected warn (root covers the default store, not the override), got %s (%s)", r.Status, r.Detail)
		}
	})

	t.Run("root covering the override passes with suffix", func(t *testing.T) {
		cfgDir := t.TempDir()
		cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+override+"\"]\n")

		r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, override))
		if r.Status != "pass" {
			t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
		}
		if !strings.Contains(r.Detail, "store taken from AGMSG_STORAGE_PATH") {
			t.Errorf("expected detail to contain the override suffix, got: %s", r.Detail)
		}
	})

	t.Run("trailing slash on the override is stripped", func(t *testing.T) {
		cfgDir := t.TempDir()
		cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+override+"\"]\n")

		r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, override+"/"))
		if r.Status != "pass" {
			t.Fatalf("expected pass (trailing slash on override must be stripped before comparison), got %s (%s)", r.Status, r.Detail)
		}
	})
}

func TestCheckCodexAgmsgWritableRoot_ProfileSuffix(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "profile = \"work\"\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, `profile "work"`) || !strings.Contains(r.Detail, "is not evaluated") {
		t.Errorf("expected detail to name the profile suffix, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_ConfigIsDirectory_Info(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.MkdirAll(cfgPath, 0o755); err != nil {
		t.Fatal(err)
	}

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "not a regular file") {
		t.Errorf("expected detail to mention not a regular file, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes lives in
// doctor_codex_writable_root_unix_test.go (self-review LOW-7: syscall.Mkfifo
// does not exist on windows).

func TestCheckCodexAgmsgWritableRoot_ConfigTooLarge_Info(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	oversized := strings.Repeat("a", codexConfigMaxBytes+1)
	if err := os.WriteFile(cfgPath, []byte("# "+oversized+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "larger than 1 MiB") {
		t.Errorf("expected detail to mention the size limit, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_RelativeAndTildeRoots_NeverMatch(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	store := filepath.Join(agmsgHome, "db")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir,
		"[sandbox_workspace_write]\nwritable_roots = [\"~/agmsg\", \"relative/db\", \""+store+"x\"]\n")

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn (no configured root actually covers the store), got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_ResolveEnvError_Info(t *testing.T) {
	wantErr := errors.New("boom")
	// resolveEnv only runs after agmsg is confirmed installed and codex is
	// confirmed spawnable.
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, func() (codexSandboxEnv, error) { return codexSandboxEnv{}, wantErr })
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not resolve the codex config path") {
		t.Errorf("expected detail to mention the resolution failure, got: %s", r.Detail)
	}
}

func TestCodexSandboxEnvFromOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir on windows does not key off HOME")
	}

	t.Run("CODEX_HOME used literally", func(t *testing.T) {
		codexHome := filepath.Join(t.TempDir(), "custom-codex-home")
		t.Setenv("CODEX_HOME", codexHome)
		t.Setenv("AGMSG_STORAGE_PATH", "")

		env, err := codexSandboxEnvFromOS()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(codexHome, "config.toml")
		if env.ConfigPath != want {
			t.Errorf("ConfigPath = %q, want %q", env.ConfigPath, want)
		}
	})

	t.Run("falls back to HOME/.codex when CODEX_HOME is unset", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CODEX_HOME", "")
		t.Setenv("HOME", dir)

		env, err := codexSandboxEnvFromOS()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(dir, ".codex", "config.toml")
		if env.ConfigPath != want {
			t.Errorf("ConfigPath = %q, want %q", env.ConfigPath, want)
		}
		if env.Home != dir {
			t.Errorf("Home = %q, want %q", env.Home, dir)
		}
	})

	t.Run("honours AGMSG_STORAGE_PATH", func(t *testing.T) {
		t.Setenv("CODEX_HOME", t.TempDir())
		t.Setenv("AGMSG_STORAGE_PATH", "/some/store")

		env, err := codexSandboxEnvFromOS()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env.AgmsgStoragePath != "/some/store" {
			t.Errorf("AgmsgStoragePath = %q, want /some/store", env.AgmsgStoragePath)
		}
	})

	t.Run("CODEX_HOME and HOME both empty returns an error", func(t *testing.T) {
		t.Setenv("CODEX_HOME", "")
		t.Setenv("HOME", "")

		if _, err := codexSandboxEnvFromOS(); err == nil {
			t.Fatal("expected an error when CODEX_HOME and HOME are both empty")
		}
	})
}

func TestPathCovers(t *testing.T) {
	cases := []struct {
		name string
		root string
		want map[string]bool
	}{
		{"root equals target", "/a/b", map[string]bool{"/a/b": true}},
		{"root is an ancestor", "/a", map[string]bool{"/a/b": true, "/a/b/c": true}},
		{"prefix sibling, not an ancestor", "/a/b", map[string]bool{"/a/bc": false}},
		{"db vs dbx sibling", "/x/db", map[string]bool{"/x/dbx": false, "/x/db/child": true}},
		{"relative root never matches", "relative/db", map[string]bool{"relative/db": false}},
		{"tilde root never matches", "~/agmsg/db", map[string]bool{"~/agmsg/db": false}},
		{"trailing slash on root is cleaned", "/a/b/", map[string]bool{"/a/b": true, "/a/b/c": true}},
	}
	for _, tc := range cases {
		for target, want := range tc.want {
			t.Run(tc.name+"__"+target, func(t *testing.T) {
				if got := pathCovers(tc.root, target); got != want {
					t.Errorf("pathCovers(%q, %q) = %v, want %v", tc.root, target, got, want)
				}
			})
		}
	}
}

func TestAgmsgStoreDir(t *testing.T) {
	t.Run("no override uses agmsgHome/db", func(t *testing.T) {
		dir, fromOverride := agmsgStoreDir("/home/agmsg", "")
		if dir != filepath.Join("/home/agmsg", "db") || fromOverride {
			t.Errorf("agmsgStoreDir(%q, %q) = (%q, %v), want (%q, false)", "/home/agmsg", "", dir, fromOverride, filepath.Join("/home/agmsg", "db"))
		}
	})
	t.Run("override strips exactly one trailing slash", func(t *testing.T) {
		dir, fromOverride := agmsgStoreDir("/home/agmsg", "/custom/store/")
		if dir != "/custom/store" || !fromOverride {
			t.Errorf("agmsgStoreDir override = (%q, %v), want (\"/custom/store\", true)", dir, fromOverride)
		}
	})
	t.Run("override without trailing slash is unchanged", func(t *testing.T) {
		dir, fromOverride := agmsgStoreDir("/home/agmsg", "/custom/store")
		if dir != "/custom/store" || !fromOverride {
			t.Errorf("agmsgStoreDir override = (%q, %v), want (\"/custom/store\", true)", dir, fromOverride)
		}
	})
}

func TestCoveringWritableRoot_StoreNotCreatedYet_AncestorCovered(t *testing.T) {
	agmsgHome := t.TempDir() // exists; its "db" child does not.
	store := filepath.Join(agmsgHome, "db")

	root := coveringWritableRoot([]string{agmsgHome}, store)
	if root != agmsgHome {
		t.Fatalf("expected the existing ancestor to cover the not-yet-created store, got root=%q", root)
	}
}

// TestCoveringWritableRoot_StoreNotCreatedYetUnderSymlinkedParent pins that
// both sides of the comparison resolve symlinks through their nearest
// existing ancestor: the store directory does not exist yet, the root names
// it through a symlinked parent, and the two must still compare equal.
func TestCoveringWritableRoot_StoreNotCreatedYetUnderSymlinkedParent(t *testing.T) {
	base := t.TempDir()
	realHome := filepath.Join(base, "real-agmsg")
	if err := os.MkdirAll(realHome, 0o755); err != nil {
		t.Fatal(err)
	}
	linkHome := filepath.Join(base, "link-agmsg")
	if err := os.Symlink(realHome, linkHome); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	store := filepath.Join(realHome, "db")       // not created
	rootViaLink := filepath.Join(linkHome, "db") // not created either
	if got := coveringWritableRoot([]string{rootViaLink}, store); got != rootViaLink {
		t.Fatalf("coveringWritableRoot = %q, want the configured root %q", got, rootViaLink)
	}
	if got := coveringWritableRoot([]string{"relative/db", "~/db"}, store); got != "" {
		t.Fatalf("relative and ~-prefixed roots must never cover the store, got %q", got)
	}
}

func TestCodexSeatModesPossible(t *testing.T) {
	cases := []struct {
		name               string
		orgCfg             config.OrgConfig
		wantWorkspaceWrite bool
		wantGuarded        bool
	}{
		{"empty config uses the built-in default (autonomous)", config.OrgConfig{}, true, false},
		{"default guarded", config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "guarded"}}, false, true},
		{
			"default autonomous plus one guarded role",
			config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "autonomous", Roles: map[string]string{"reviewer": "guarded"}}},
			true, true,
		},
		{
			"all roles guarded, default guarded too",
			config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "guarded", Roles: map[string]string{"a": "guarded", "b": "guarded"}}},
			false, true,
		},
		{
			"an edits role under a guarded default",
			config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "guarded", Roles: map[string]string{"implementer": "edits"}}},
			true, true,
		},
		{
			// self-review NEW-2: a roles entry keyed by the empty string
			// must not shadow the real default. "" is never a real role
			// name (internal/cli/org.go's --role is required and non-blank
			// at spawn), so it is applied as just another configured mode
			// alongside the "guarded" default, not in place of it.
			"an empty-string role key does not shadow the default",
			config.OrgConfig{Permissions: config.OrgPermissionsConfig{Default: "guarded", Roles: map[string]string{"": "autonomous"}}},
			true, true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWW, gotG := codexSeatModesPossible(tc.orgCfg)
			if gotWW != tc.wantWorkspaceWrite || gotG != tc.wantGuarded {
				t.Errorf("codexSeatModesPossible(%+v) = (%v, %v), want (%v, %v)", tc.orgCfg, gotWW, gotG, tc.wantWorkspaceWrite, tc.wantGuarded)
			}
		})
	}
}

func TestCodexNotNeededWhyA(t *testing.T) {
	cases := []struct {
		name          string
		codexVerified bool
		want          string
	}{
		{"codex_verified false", false, "[org.permissions].codex_verified is false"},
		{"codex_verified true, so the role condition is what is missing", true, "no role resolves to edits or autonomous"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexNotNeededWhyA(tc.codexVerified); got != tc.want {
				t.Errorf("codexNotNeededWhyA(%v) = %q, want %q", tc.codexVerified, got, tc.want)
			}
		})
	}
}

func TestCodexNotNeededWhyB(t *testing.T) {
	cfgDisplay := "/some/config.toml"
	cases := []struct {
		name        string
		guardedRole bool
		exists      bool
		want        string
	}{
		{"no guarded role wins regardless of config", false, true, "no role resolves to guarded"},
		{"guarded role but config absent", true, false, cfgDisplay + " does not exist"},
		{"guarded role, config exists, sandbox_mode unset", true, true, cfgDisplay + ` does not set sandbox_mode = "workspace-write"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexNotNeededWhyB(tc.guardedRole, tc.exists, cfgDisplay); got != tc.want {
				t.Errorf("codexNotNeededWhyB(%v, %v, %q) = %q, want %q", tc.guardedRole, tc.exists, cfgDisplay, got, tc.want)
			}
		})
	}
}

// TestCodexSandboxReasonClause is self-review NEW-1's pure unit test: zero
// reasons is never actually rendered by checkCodexAgmsgWritableRoot (it
// takes the "not needed" branch instead), but is still defined as "" here;
// one reason is unchanged; two or more are numbered so the boundary
// between them stays visible despite each reason already containing its
// own " and ".
func TestCodexSandboxReasonClause(t *testing.T) {
	cases := []struct {
		name    string
		reasons []string
		want    string
	}{
		{"zero reasons renders as empty", nil, ""},
		{"one reason is unchanged", []string{"reason a"}, "reason a"},
		{"two reasons are numbered", []string{"reason a", "reason b"}, "(1) reason a and (2) reason b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexSandboxReasonClause(tc.reasons); got != tc.want {
				t.Errorf("codexSandboxReasonClause(%v) = %q, want %q", tc.reasons, got, tc.want)
			}
		})
	}
}

// TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault proves the TestMain
// seam (main_test.go) works end to end: without any test-local override,
// checkCodexAgmsgWritableRoot resolves through doctorCodexSandboxEnv, which
// TestMain has pinned to a non-existent config path -- so with the
// default project config (codex_verified = false, no sandbox_mode), the
// check reads pass/"not needed" regardless of the developer's real
// ~/.codex/config.toml.
func TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)

	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	t.Setenv("RALPH_ORG_AGMSG_HOME", agmsgHome)

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("runDoctorOpts returned error: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "Codex sandbox (agmsg writable root): pass") {
		t.Errorf("expected a pass-level Codex sandbox line in output:\n%s", out)
	}
	if !strings.Contains(string(out), "not needed") {
		t.Errorf("expected the not-needed detail in output:\n%s", out)
	}
}

// TestRunDoctorOpts_CodexSandboxCheck_WarnsThroughTheSeam overrides the
// doctorCodexSandboxEnv seam and sets [org.permissions].codex_verified =
// true in the project's ralph.toml, proving runDoctorOpts wires
// checkCodexAgmsgWritableRoot's result into its printed output end to end,
// and that a warn-level finding does not flip the exit code.
func TestRunDoctorOpts_CodexSandboxCheck_WarnsThroughTheSeam(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	ralphTomlPath := filepath.Join(dir, "ralph.toml")
	existing, err := os.ReadFile(ralphTomlPath)
	if err != nil {
		t.Fatal(err)
	}
	appended := string(existing) + "\n[org.permissions]\ncodex_verified = true\n"
	if err := os.WriteFile(ralphTomlPath, []byte(appended), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)

	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	t.Setenv("RALPH_ORG_AGMSG_HOME", agmsgHome)

	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml")
	orig := doctorCodexSandboxEnv
	doctorCodexSandboxEnv = codexSandboxTestEnv(noSuchCfg, "")
	t.Cleanup(func() { doctorCodexSandboxEnv = orig })

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runErr := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if runErr != nil {
		t.Fatalf("runDoctorOpts returned error with only a warn-level codex-sandbox finding: %v\noutput:\n%s", runErr, out)
	}
	if !strings.Contains(string(out), "Codex sandbox (agmsg writable root): warn") {
		t.Errorf("expected a warn-level Codex sandbox line in output:\n%s", out)
	}
}
