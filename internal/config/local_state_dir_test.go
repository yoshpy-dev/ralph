package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests of [org].state_dir, the key only the personal file may set (plan
// 2026-10-30-org-state-dir-local, D1 to D3, AC2 to AC4 and AC9).

// TestLoad_LocalStateDir covers D2 and the edge of AC4: the personal file may
// set [org].state_dir, and Load keeps it as written without checking it, a
// relative value or one starting with ~/ included, while ralph.toml's other
// [org] keys still apply. A ralph.toml without the key and no personal file
// leave the default "".
func TestLoad_LocalStateDir(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	if got := Default().Org.StateDir; got != "" {
		t.Errorf("Default() [org].state_dir = %q, want \"\"", got)
	}
	for _, value := range []string{"/abs/../ledger", "~/ledger", "rel/ledger", "../ledger", "~"} {
		t.Run(value, func(t *testing.T) {
			configPath, _ := localTestFiles(t, "[org]\nmax_seats = 3\n\n[org.director]\nack_minutes = 20\n", "[org]\nstate_dir = \""+value+"\"\n")
			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Org.StateDir != value || cfg.Org.MaxSeats != 3 || cfg.Org.Director.AckMinutes != 20 {
				t.Errorf("Load: state_dir %q, max_seats %d, ack_minutes %d; want the personal file's %q and ralph.toml's 3 and 20",
					cfg.Org.StateDir, cfg.Org.MaxSeats, cfg.Org.Director.AckMinutes, value)
			}
		})
	}
	t.Run("absent", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\nmax_seats = 3\n", "")
		cfg, err := Load(configPath)
		if err != nil || cfg.Org.StateDir != "" {
			t.Errorf("Load = (state_dir %q, err %v), want (\"\", nil)", cfg.Org.StateDir, err)
		}
	})
}

// TestLocalOrgStateDir_Values covers AC3's config half (D2): an absolute
// value is returned cleaned and one starting with ~/ with the home directory
// in place of ~; an empty or absent state_dir, an empty personal file, and no
// personal file give "". path is the personal file's, also when it does not
// exist. [ORG] STATE_DIR is read too, by the letter-case rule Load's check of
// ralph.toml follows (TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder).
// ralph.toml is never read: one that sets the key, or that is not valid TOML,
// changes nothing.
func TestLocalOrgStateDir_Values(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	abs := filepath.Join(t.TempDir(), "ledger")

	for _, tc := range []struct {
		name, ralphToml, local, want string
		writeEmptyLocal              bool
	}{
		{name: "absolute", local: "[org]\nstate_dir = \"" + abs + "\"\n", want: abs},
		{name: "absolute, cleaned", local: "[org]\nstate_dir = \"" + abs + "/../ledger/./\"\n", want: abs},
		{name: "upper-case table and key", local: "[ORG]\nSTATE_DIR = \"" + abs + "\"\n", want: abs},
		{name: "under home", local: "[org]\nstate_dir = \"~/ledgers/a\"\n", want: filepath.Join(home, "ledgers", "a")},
		{name: "home itself", local: "[org]\nstate_dir = \"~/\"\n", want: home},
		{name: "empty", local: "[org]\nstate_dir = \"\"\n"},
		{name: "absent", local: "[org]\nmax_seats = 2\n"},
		{name: "an empty file", writeEmptyLocal: true},
		{name: "no file"},
		{name: "ralph.toml sets it", ralphToml: "[org]\nstate_dir = \"" + abs + "\"\n"},
		{name: "ralph.toml is not valid TOML", ralphToml: "[org\n", local: "[org]\nstate_dir = \"" + abs + "\"\n", want: abs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, tc.ralphToml, tc.local)
			if tc.writeEmptyLocal {
				writeLocalTestFile(t, localPath, "")
			}
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			if err != nil || got != tc.want || path != localPath {
				t.Errorf("LocalOrgStateDir = (%q, %q, %v), want (%q, %q, nil)", got, path, err, tc.want, localPath)
			}
		})
	}
}

// TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute covers AC3 (D2): a
// value that is neither absolute nor starts with ~/ is an error naming the
// personal file's path and the value, in the shape of the agmsg_home one.
func TestLocalOrgStateDir_RefusesValuesThatAreNotAbsolute(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	for _, value := range []string{"ledger", "./ledger", "../ledger", "~", "~user/ledger", " /ledger"} {
		t.Run(value, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, "", "[org]\nstate_dir = \""+value+"\"\n")
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			want := localPath + `: [org].state_dir "` + value + `" must be an absolute path or start with ~/: ralph.local.toml is read from the repository's main worktree root`
			if err == nil || !strings.Contains(err.Error(), want) || got != "" || path != localPath {
				t.Errorf("LocalOrgStateDir = (%q, %q, %v), want (\"\", %q, an error containing %q)", got, path, err, localPath, want)
			}
		})
	}
}

// TestLocalOrgStateDir_FileErrorsNameItsPath covers a personal file that is a
// directory, cannot be read, is not valid TOML, or sets state_dir to a value
// that is not a string, and a ~/ value with no home directory: each is an
// error naming the file's path.
func TestLocalOrgStateDir_FileErrorsNameItsPath(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, localPath string)
		want  string
	}{
		{"a directory", func(t *testing.T, localPath string) {
			if err := os.Mkdir(localPath, 0o755); err != nil {
				t.Fatal(err)
			}
		}, "is a directory"},
		{"unreadable", func(t *testing.T, localPath string) {
			if os.Geteuid() == 0 {
				t.Skip("root ignores the permission bits")
			}
			writeLocalTestFile(t, localPath, "[org]\nstate_dir = \"/ledger\"\n")
			if err := os.Chmod(localPath, 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(localPath, 0o644) })
		}, "permission denied"},
		{"not valid TOML", func(t *testing.T, localPath string) {
			writeLocalTestFile(t, localPath, "[org\nstate_dir = \"/ledger\"\n")
		}, ""},
		{"not a string", func(t *testing.T, localPath string) {
			writeLocalTestFile(t, localPath, "[org]\nstate_dir = 1\n")
		}, ""},
		{"no home directory", func(t *testing.T, localPath string) {
			t.Setenv("HOME", "")
			writeLocalTestFile(t, localPath, "[org]\nstate_dir = \"~/ledger\"\n")
		}, `[org].state_dir "~/ledger": `},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, "", "")
			tc.setup(t, localPath)
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			if err == nil || !strings.Contains(err.Error(), localPath) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("LocalOrgStateDir: err = %v, want one naming %s and containing %q", err, localPath, tc.want)
			}
			if got != "" || path != localPath {
				t.Errorf("LocalOrgStateDir = (%q, %q), want (\"\", %q)", got, path, localPath)
			}
		})
	}
}

// TestLocalOrgStateDir_LocalConfigEnv covers AC9: RALPH_LOCAL_CONFIG set to
// exactly "off" gives "" and no path, even for a personal file with a value
// or one that is not valid TOML, and runs no git; any other value, empty
// included, and no value read the file. git is a stub on PATH that logs its
// calls and fails, so the directory counts as outside git either way.
func TestLocalOrgStateDir_LocalConfigEnv(t *testing.T) {
	bin := t.TempDir()
	gitLog := filepath.Join(t.TempDir(), "git.log")
	writeLocalTestFile(t, filepath.Join(bin, "git"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$LOCAL_TEST_GIT_LOG\"\nexit 1\n")
	if err := os.Chmod(filepath.Join(bin, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("LOCAL_TEST_GIT_LOG", gitLog)
	abs := filepath.Join(t.TempDir(), "ledger")

	t.Run("off skips the file and git", func(t *testing.T) {
		t.Setenv(LocalConfigEnv, "off")
		for _, local := range []string{"[org]\nstate_dir = \"" + abs + "\"\n", "[org\n"} {
			configPath, _ := localTestFiles(t, "", local)
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			if err != nil || got != "" || path != "" {
				t.Errorf("personal file %q: LocalOrgStateDir = (%q, %q, %v), want (\"\", \"\", nil)", local, got, path, err)
			}
		}
		if _, err := os.Stat(gitLog); !os.IsNotExist(err) {
			t.Errorf("LocalOrgStateDir ran git with %s=off (stat %s: %v)", LocalConfigEnv, gitLog, err)
		}
	})

	for _, tc := range []struct {
		name  string
		value string
		unset bool
	}{
		{"empty", "", false},
		{"OFF", "OFF", false},
		{"on", "on", false},
		{"unset", "", true},
	} {
		t.Run(tc.name+" reads the file", func(t *testing.T) {
			if err := os.Remove(gitLog); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			t.Setenv(LocalConfigEnv, tc.value)
			if tc.unset {
				if err := os.Unsetenv(LocalConfigEnv); err != nil {
					t.Fatal(err)
				}
			}
			configPath, localPath := localTestFiles(t, "", "[org]\nstate_dir = \""+abs+"\"\n")
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			if err != nil || got != abs || path != localPath {
				t.Errorf("LocalOrgStateDir = (%q, %q, %v), want (%q, %q, nil)", got, path, err, abs, localPath)
			}
			if _, err := os.Stat(gitLog); err != nil {
				t.Errorf("LocalOrgStateDir ran no git to find the personal file: %v", err)
			}
		})
	}
}

// TestLocalOrgStateDir_ReadsTheMainWorktreesFile covers AC2's config half
// (D1): from the main worktree's root, a linked worktree's root, and a
// subdirectory of either, given as a directory, as "" (the working
// directory), or relative to it, LocalOrgStateDir reads the main worktree
// root's personal file, not the same-named files in those directories. In a
// linked worktree of a bare repository it reads the one at that worktree's
// toplevel.
func TestLocalOrgStateDir_ReadsTheMainWorktreesFile(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	f := newLocalGitFixture(t)
	t.Chdir(f.outside)
	want := filepath.Join(f.outside, "main-ledger")
	writeLocalTestFile(t, filepath.Join(f.main, LocalConfigFileName), "[org]\nstate_dir = \""+want+"\"\n")

	check := func(t *testing.T, dir, wantValue, wantDir string) {
		t.Helper()
		got, path, err := LocalOrgStateDir(dir)
		if err != nil || got != wantValue {
			t.Errorf("LocalOrgStateDir(%q) = (%q, %v), want (%q, nil)", dir, got, err, wantValue)
		}
		checkLocalPathIn(t, filepath.Join(dir, "ralph.toml"), path, wantDir)
	}
	for _, dir := range []string{
		f.main,
		filepath.Join(f.main, "pkg", "sub"),
		f.linked,
		filepath.Join(f.linked, "pkg", "sub"),
	} {
		if dir != f.main {
			writeLocalTestFile(t, filepath.Join(dir, LocalConfigFileName), "[org]\nstate_dir = \"/decoy-in-"+filepath.Base(dir)+"\"\n")
		}
		check(t, dir, want, f.main)
	}
	t.Run("the working directory", func(t *testing.T) {
		t.Chdir(filepath.Join(f.linked, "pkg", "sub"))
		check(t, "", want, f.main)
		t.Chdir(f.linked)
		check(t, filepath.Join("pkg", "sub"), want, f.main)
	})
	t.Run("a bare repository's linked worktree", func(t *testing.T) {
		bareWant := filepath.Join(f.outside, "bare-ledger")
		writeLocalTestFile(t, filepath.Join(f.bareLinked, LocalConfigFileName), "[org]\nstate_dir = \""+bareWant+"\"\n")
		check(t, filepath.Join(f.bareLinked, "pkg", "sub"), bareWant, f.bareLinked)
	})
}

// TestLocalOrgStateDirAt_ReadsTheGivenFileWithoutGit covers the reader
// org.ResolveOrgStateDir uses with the directory it has found (plan
// 2026-11-01-org-ledger-gaps, AC7): it reads the file at the path it is
// given, wherever that is, with LocalOrgStateDir's values and its errors
// naming that path, and runs no git; with LocalConfigEnv "off" it reads
// nothing. git is a stub on PATH that logs its calls and fails.
func TestLocalOrgStateDirAt_ReadsTheGivenFileWithoutGit(t *testing.T) {
	bin := t.TempDir()
	gitLog := filepath.Join(t.TempDir(), "git.log")
	writeLocalTestFile(t, filepath.Join(bin, "git"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$LOCAL_TEST_GIT_LOG\"\nexit 1\n")
	if err := os.Chmod(filepath.Join(bin, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("LOCAL_TEST_GIT_LOG", gitLog)
	abs := filepath.Join(t.TempDir(), "ledger")
	path := filepath.Join(t.TempDir(), "not-a-repository", LocalConfigFileName)

	for _, tc := range []struct{ name, env, body, want, wantErr string }{
		{name: "an absolute value", body: "[org]\nstate_dir = \"" + abs + "/./\"\n", want: abs},
		{name: "no state_dir", body: "[org]\nmax_seats = 2\n"},
		{name: "a relative value", body: "[org]\nstate_dir = \"rel\"\n",
			wantErr: path + `: [org].state_dir "rel" must be an absolute path or start with ~/`},
		{name: "not valid TOML", body: "[org\n", wantErr: path + ": "},
		{name: "off", env: "off", body: "[org\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(LocalConfigEnv, tc.env)
			writeLocalTestFile(t, path, tc.body)
			got, err := LocalOrgStateDirAt(path)
			if tc.wantErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tc.wantErr) || got != "" {
					t.Errorf("LocalOrgStateDirAt = (%q, %v), want (\"\", an error starting with %q)", got, err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("LocalOrgStateDirAt = (%q, %v), want (%q, nil)", got, err, tc.want)
			}
		})
	}
	t.Run("no file", func(t *testing.T) {
		t.Setenv(LocalConfigEnv, "")
		if got, err := LocalOrgStateDirAt(filepath.Join(t.TempDir(), LocalConfigFileName)); err != nil || got != "" {
			t.Errorf("LocalOrgStateDirAt = (%q, %v), want (\"\", nil)", got, err)
		}
	})
	if _, err := os.Stat(gitLog); !os.IsNotExist(err) {
		t.Errorf("LocalOrgStateDirAt ran git (stat %s: %v)", gitLog, err)
	}
}
