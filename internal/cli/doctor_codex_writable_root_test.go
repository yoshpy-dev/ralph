package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// codexSandboxTestEnv returns a resolveEnv closure fixed to the given
// config path and AGMSG_STORAGE_PATH override, for tests that don't want to
// mutate process environment variables at all.
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

func TestCheckCodexAgmsgWritableRoot_NotNeeded_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml") // absent -- no reason either way.

	r := checkCodexAgmsgWritableRoot(false, agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "not needed") {
		t.Errorf("expected detail to mention not needed, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "")

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

func TestCheckCodexAgmsgWritableRoot_RootEqualsStore_Pass(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	store := filepath.Join(agmsgHome, "db")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+store+"\"]\n")

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

		r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "pass" {
		t.Fatalf("expected pass (symlinked root resolves to the store), got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_GuardedSeatSandboxModeNoRoots_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	r := checkCodexAgmsgWritableRoot(false, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "guarded codex seats inherit it") {
		t.Errorf("expected detail to mention guarded seats, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_MissingConfigCodexVerified_Warn(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	noSuchCfg := filepath.Join(cfgDir, "config.toml")

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(noSuchCfg, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"not valid TOML", "line", "column"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
	if strings.Contains(r.Detail, "CANARYLEAK123") {
		t.Errorf("config content must never leak into Detail, got: %s", r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_WritableRootsWrongType_Info(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{"string instead of array", "[sandbox_workspace_write]\nwritable_roots = \"x\"\n", ""},
		{"array of ints", "[sandbox_workspace_write]\nwritable_roots = [1, 2]\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agmsgHome := t.TempDir()
			writeAgmsgHome(t, agmsgHome, "")
			cfgDir := t.TempDir()
			cfgPath := writeCodexConfig(t, cfgDir, tc.toml)

			r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
			if r.Status != "info" {
				t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, "writable_roots is not an array of strings") {
				t.Errorf("expected detail to name the type problem, got: %s", r.Detail)
			}
		})
	}
}

func TestCheckCodexAgmsgWritableRoot_AgmsgMissing_Info(t *testing.T) {
	agmsgHome := filepath.Join(t.TempDir(), "no-such-agmsg-home")
	cfgDir := t.TempDir()
	cfgPath := writeCodexConfig(t, cfgDir, "sandbox_mode = \"workspace-write\"\n")

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

		r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, override))
		if r.Status != "warn" {
			t.Fatalf("expected warn (root covers the default store, not the override), got %s (%s)", r.Status, r.Detail)
		}
	})

	t.Run("root covering the override passes with suffix", func(t *testing.T) {
		cfgDir := t.TempDir()
		cfgPath := writeCodexConfig(t, cfgDir, "[sandbox_workspace_write]\nwritable_roots = [\""+override+"\"]\n")

		r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, override))
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

		r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, override+"/"))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "not a regular file") {
		t.Errorf("expected detail to mention not a regular file, got: %s", r.Detail)
	}
}

// TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes proves a
// FIFO config path is rejected WITHOUT ever being opened -- opening it would
// block forever, hanging the whole doctor run.
func TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no FIFOs on windows")
	}
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := syscall.Mkfifo(cfgPath, 0o644); err != nil {
		t.Fatal(err)
	}

	type result struct {
		r checkResult
	}
	done := make(chan result, 1)
	go func() {
		done <- result{checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))}
	}()

	select {
	case got := <-done:
		if got.r.Status != "info" {
			t.Fatalf("expected info, got %s (%s)", got.r.Status, got.r.Detail)
		}
		if !strings.Contains(got.r.Detail, "not a regular file") {
			t.Errorf("expected detail to mention not a regular file, got: %s", got.r.Detail)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("checkCodexAgmsgWritableRoot did not return within 5s -- likely blocked opening the FIFO")
	}
}

func TestCheckCodexAgmsgWritableRoot_ConfigTooLarge_Info(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	oversized := strings.Repeat("a", codexConfigMaxBytes+1)
	if err := os.WriteFile(cfgPath, []byte("# "+oversized+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
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

	r := checkCodexAgmsgWritableRoot(true, agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "warn" {
		t.Fatalf("expected warn (no configured root actually covers the store), got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckCodexAgmsgWritableRoot_ResolveEnvError_Info(t *testing.T) {
	wantErr := errors.New("boom")
	// resolveEnv only runs after agmsg is confirmed installed.
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	r := checkCodexAgmsgWritableRoot(true, agmsgHome, func() (codexSandboxEnv, error) { return codexSandboxEnv{}, wantErr })
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
