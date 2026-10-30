package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// Tests of the personal file ralph.local.toml (plan
// 2026-10-29-local-config-org, D2 to D5, AC1 to AC4 and AC7). Every test
// that reads a personal file first sets RALPH_LOCAL_CONFIG to "", so a
// TestMain that pins it to off for the rest of the package does not hide the
// file. The git fixtures run git without the developer's git config
// (isolateLocalGitEnv).

// isolateLocalGitEnv keeps the git fixtures below independent of the
// developer's git config and of a GIT_DIR-style variable inherited from a git
// hook, which would otherwise point `git worktree add` at the real
// repository.
func isolateLocalGitEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}
}

// runLocalGit runs git with args and fails the test on a non-zero exit.
func runLocalGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writeLocalTestFile writes content to path, creating its parent
// directories.
func writeLocalTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// localGitFixture is a temporary repository with one commit at main, a
// linked worktree beside it (linked), a linked worktree of a bare repository
// stored as <tmp>/project/.git (bareLinked), and a directory outside git
// (outside), each with a pkg/sub subdirectory except outside. The paths are
// the raw t.TempDir() ones, not symlink-resolved.
type localGitFixture struct {
	main, linked, bareLinked, outside string
}

func newLocalGitFixture(t *testing.T) localGitFixture {
	t.Helper()
	isolateLocalGitEnv(t)
	tmp := t.TempDir()
	f := localGitFixture{
		main:       filepath.Join(tmp, "main"),
		linked:     filepath.Join(tmp, "linked"),
		bareLinked: filepath.Join(tmp, "project", "wt"),
		outside:    filepath.Join(tmp, "outside"),
	}
	bare := filepath.Join(tmp, "project", ".git")
	runLocalGit(t, "init", "--quiet", f.main)
	runLocalGit(t, "-C", f.main, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", "init")
	runLocalGit(t, "-C", f.main, "worktree", "add", "--quiet", f.linked, "-b", "linked")
	runLocalGit(t, "clone", "--quiet", "--bare", f.main, bare)
	runLocalGit(t, "-C", bare, "worktree", "add", "--quiet", f.bareLinked, "-b", "feat")
	for _, dir := range []string{
		f.outside,
		filepath.Join(f.main, "pkg", "sub"),
		filepath.Join(f.linked, "pkg", "sub"),
		filepath.Join(f.bareLinked, "pkg", "sub"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return f
}

// checkLocalPathIn fails the test unless got is the personal file in
// wantDir. The directories are compared with gitroot.SamePath, since git
// prints symlink-resolved paths (macOS's /var is /private/var) and the file
// itself does not exist.
func checkLocalPathIn(t *testing.T, configPath, got, wantDir string) {
	t.Helper()
	if filepath.Base(got) != LocalConfigFileName || !gitroot.SamePath(filepath.Dir(got), wantDir) {
		t.Errorf("LocalPath(%q) = %q, want %s in %q", configPath, got, LocalConfigFileName, wantDir)
	}
}

// TestLocalPath_InGitIsTheMainWorktreeRoot covers D2: from the main
// worktree's root or a subdirectory, and from a linked worktree's root or a
// subdirectory, the personal file is the one at the main worktree's root.
// The working directory is outside git, so the root can only come from
// configPath.
func TestLocalPath_InGitIsTheMainWorktreeRoot(t *testing.T) {
	f := newLocalGitFixture(t)
	t.Chdir(f.outside)

	for _, dir := range []string{
		f.main,
		filepath.Join(f.main, "pkg", "sub"),
		f.linked,
		filepath.Join(f.linked, "pkg", "sub"),
	} {
		configPath := filepath.Join(dir, "ralph.toml")
		checkLocalPathIn(t, configPath, LocalPath(configPath), f.main)
	}
}

// TestLocalPath_BareRepositoryLinkedWorktreeIsItsToplevel covers a linked
// worktree of a bare repository, where git names no main worktree: the
// personal file is the one at that worktree's toplevel.
func TestLocalPath_BareRepositoryLinkedWorktreeIsItsToplevel(t *testing.T) {
	f := newLocalGitFixture(t)
	t.Chdir(f.main)

	for _, dir := range []string{f.bareLinked, filepath.Join(f.bareLinked, "pkg", "sub")} {
		configPath := filepath.Join(dir, "ralph.toml")
		checkLocalPathIn(t, configPath, LocalPath(configPath), f.bareLinked)
	}
}

// TestLocalPath_OutsideGitIsTheConfigFilesDirectory covers a directory
// outside git and one that does not exist: the personal file is beside the
// config file. The working directory is the main worktree, so a git run in
// it instead of in configPath's directory would name the main root.
func TestLocalPath_OutsideGitIsTheConfigFilesDirectory(t *testing.T) {
	f := newLocalGitFixture(t)
	t.Chdir(f.main)

	for _, dir := range []string{f.outside, filepath.Join(f.outside, "missing")} {
		configPath := filepath.Join(dir, "ralph.toml")
		checkLocalPathIn(t, configPath, LocalPath(configPath), dir)
	}
}

// TestLocalPath_RelativeConfigPath covers configPath "ralph.toml", the
// default of the ralph verbs: its directory is the working directory, made
// absolute, so a linked worktree's subdirectory still names the main root
// and outside git the result is an absolute path.
func TestLocalPath_RelativeConfigPath(t *testing.T) {
	f := newLocalGitFixture(t)

	t.Run("linked worktree subdirectory", func(t *testing.T) {
		t.Chdir(filepath.Join(f.linked, "pkg", "sub"))
		checkLocalPathIn(t, "ralph.toml", LocalPath("ralph.toml"), f.main)
	})
	t.Run("outside git", func(t *testing.T) {
		t.Chdir(f.outside)
		got := LocalPath("ralph.toml")
		if !filepath.IsAbs(got) {
			t.Errorf("LocalPath(%q) = %q, want an absolute path", "ralph.toml", got)
		}
		checkLocalPathIn(t, "ralph.toml", got, f.outside)
	})
}

// localTestFiles writes the ralph.toml (when ralphToml is not empty) and the
// ralph.local.toml (when local is not empty) of a fresh directory outside
// git, where LocalPath names the file beside ralph.toml. It returns the
// ralph.toml path and the personal file's path.
func localTestFiles(t *testing.T, ralphToml, local string) (configPath, localPath string) {
	t.Helper()
	dir := t.TempDir()
	configPath = filepath.Join(dir, "ralph.toml")
	localPath = filepath.Join(dir, LocalConfigFileName)
	if ralphToml != "" {
		writeLocalTestFile(t, configPath, ralphToml)
	}
	if local != "" {
		writeLocalTestFile(t, localPath, local)
	}
	return configPath, localPath
}

// TestLoad_LocalFileOverridesKeyByKey covers AC1: a key the personal file
// sets overrides ralph.toml's, and the other keys of the same table, map
// entries included, keep ralph.toml's values.
func TestLoad_LocalFileOverridesKeyByKey(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	configPath, _ := localTestFiles(t,
		"[standard]\nplan_advisory = true\n\n[org]\nmax_seats = 3\nmax_orgs = 4\n\n[org.roles]\nleader = [\"opus\"]\n\n[org.director]\nack_minutes = 20\nresolve_minutes = 90\n",
		"[org]\nmax_seats = 7\n\n[org.roles]\nreviewer = [\"sonnet\"]\n\n[org.director]\nack_minutes = 5\n")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, tc := range []struct {
		key       string
		got, want int
	}{
		{"[org].max_seats", cfg.Org.MaxSeats, 7},
		{"[org].max_orgs", cfg.Org.MaxOrgs, 4},
		{"[org.director].ack_minutes", cfg.Org.Director.AckMinutes, 5},
		{"[org.director].resolve_minutes", cfg.Org.Director.ResolveMinutes, 90},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.key, tc.got, tc.want)
		}
	}
	if !cfg.Standard.PlanAdvisory {
		t.Error("[standard].plan_advisory = false, want ralph.toml's true")
	}
	wantRoles := map[string][]string{"leader": {"opus"}, "reviewer": {"sonnet"}}
	if !reflect.DeepEqual(cfg.Org.Roles, wantRoles) {
		t.Errorf("[org.roles] = %v, want %v", cfg.Org.Roles, wantRoles)
	}
}

// TestLoad_LocalFileReplacesArrays covers AC1 and the plan's assumption
// about go-toml/v2: an array, an array of tables ([[org.model_pool]])
// included, that the personal file sets replaces ralph.toml's, even when it
// is shorter, instead of being appended to it or overwriting it element by
// element.
func TestLoad_LocalFileReplacesArrays(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	configPath, _ := localTestFiles(t,
		"[org]\ndriver_pool = [\"claude\", \"codex\"]\n\n"+
			"[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"opus\"\n\n"+
			"[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"sonnet\"\n\n"+
			"[[org.model_pool]]\ndriver = \"codex\"\nmodel = \"gpt-6-astra\"\n\n"+
			"[org.director]\nwake = [\"pull\", \"push\"]\n",
		"[org]\ndriver_pool = [\"claude\"]\n\n"+
			"[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"haiku\"\n\n"+
			"[org.director]\nwake = [\"push\"]\n")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := []string{"claude"}; !slices.Equal(cfg.Org.DriverPool, want) {
		t.Errorf("[org].driver_pool = %v, want %v", cfg.Org.DriverPool, want)
	}
	if want := []OrgModelPoolEntry{{Driver: "claude", Model: "haiku"}}; !slices.Equal(cfg.Org.ModelPool, want) {
		t.Errorf("[org].model_pool = %v, want %v", cfg.Org.ModelPool, want)
	}
	if want := []string{DirectorWakePush}; !slices.Equal(cfg.Org.Director.Wake, want) {
		t.Errorf("[org.director].wake = %v, want %v", cfg.Org.Director.Wake, want)
	}
}

// TestLoad_LocalFileWithoutRalphToml covers AC1: the personal file is read
// even when ralph.toml does not exist, and an empty one leaves Default().
func TestLoad_LocalFileWithoutRalphToml(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")

	t.Run("a key", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "", "[org]\nmax_seats = 8\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		want := Default()
		want.Org.MaxSeats = 8
		if !reflect.DeepEqual(cfg, want) {
			t.Errorf("Load = %+v, want Default() with [org].max_seats 8: %+v", cfg, want)
		}
	})
	t.Run("an empty file", func(t *testing.T) {
		configPath, localPath := localTestFiles(t, "", "")
		writeLocalTestFile(t, localPath, "")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if !reflect.DeepEqual(cfg, Default()) {
			t.Errorf("Load = %+v, want Default()", cfg)
		}
	})
}

// TestLoad_LocalFileThroughASymlink covers a personal file that is a
// symlink: Load reads the file it points at.
func TestLoad_LocalFileThroughASymlink(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	configPath, localPath := localTestFiles(t, "", "")
	target := filepath.Join(t.TempDir(), "personal.toml")
	writeLocalTestFile(t, target, "[org]\nmax_orgs = 9\n")
	if err := os.Symlink(target, localPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.MaxOrgs != 9 {
		t.Errorf("[org].max_orgs = %d, want the symlinked file's 9", cfg.Org.MaxOrgs)
	}
}

// TestLoad_LocalFileErrorsNameItsPath covers AC3: a personal file that is a
// directory, cannot be read, is not valid TOML, or makes the merged config
// fail validation is an error naming the file's path, with and without a
// ralph.toml beside it.
func TestLoad_LocalFileErrorsNameItsPath(t *testing.T) {
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
			writeLocalTestFile(t, localPath, "[org]\nmax_seats = 2\n")
			if err := os.Chmod(localPath, 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(localPath, 0o644) })
		}, "permission denied"},
		{"not valid TOML", func(t *testing.T, localPath string) {
			writeLocalTestFile(t, localPath, "[org\nmax_seats = 2\n")
		}, ""},
		{"fails validation", func(t *testing.T, localPath string) {
			writeLocalTestFile(t, localPath, "[org]\nmax_seats = 0\n")
		}, "[org].max_seats must be >= 1, got 0 (with " + "%LOCAL%" + " applied)"},
	} {
		for _, withRalphToml := range []bool{true, false} {
			name := tc.name + " without ralph.toml"
			if withRalphToml {
				name = tc.name + " with ralph.toml"
			}
			t.Run(name, func(t *testing.T) {
				ralphToml := ""
				if withRalphToml {
					ralphToml = "[org]\nmax_orgs = 4\n"
				}
				configPath, localPath := localTestFiles(t, ralphToml, "")
				tc.setup(t, localPath)
				_, err := Load(configPath)
				if err == nil {
					t.Fatal("Load: nil error, want one naming the personal file")
				}
				if !strings.Contains(err.Error(), localPath) {
					t.Errorf("Load: err = %v, want it to name %s", err, localPath)
				}
				if want := strings.ReplaceAll(tc.want, "%LOCAL%", localPath); !strings.Contains(err.Error(), want) {
					t.Errorf("Load: err = %v, want it to contain %q", err, want)
				}
			})
		}
	}
}

// TestLoad_RalphTomlValidationErrorDoesNotNameAnAbsentLocalFile pins the
// other side of AC3: with no personal file, a validation error of ralph.toml
// stays as it was, with no ralph.local.toml in it.
func TestLoad_RalphTomlValidationErrorDoesNotNameAnAbsentLocalFile(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	configPath, _ := localTestFiles(t, "[org]\nmax_seats = 0\n", "")
	_, err := Load(configPath)
	if err == nil || err.Error() != "[org].max_seats must be >= 1, got 0" {
		t.Errorf("Load: err = %v, want exactly %q", err, "[org].max_seats must be >= 1, got 0")
	}
}

// TestLoad_LocalConfigEnv covers AC4 on the Load side: RALPH_LOCAL_CONFIG
// set to exactly "off" skips the personal file, here one that is not even
// valid TOML, and runs no git; any other value, empty included, and no value
// read it. git is a stub on PATH that logs its calls and fails, so Load
// treats the directory as outside git either way.
func TestLoad_LocalConfigEnv(t *testing.T) {
	bin := t.TempDir()
	gitLog := filepath.Join(t.TempDir(), "git.log")
	writeLocalTestFile(t, filepath.Join(bin, "git"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$LOCAL_TEST_GIT_LOG\"\nexit 1\n")
	if err := os.Chmod(filepath.Join(bin, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("LOCAL_TEST_GIT_LOG", gitLog)

	t.Run("off skips the file and git", func(t *testing.T) {
		t.Setenv(LocalConfigEnv, "off")
		configPath, _ := localTestFiles(t, "[org]\nmax_seats = 3\n", "[org\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.MaxSeats != 3 {
			t.Errorf("[org].max_seats = %d, want ralph.toml's 3", cfg.Org.MaxSeats)
		}
		if _, err := os.Stat(gitLog); !os.IsNotExist(err) {
			t.Errorf("Load ran git with %s=off (stat %s: %v)", LocalConfigEnv, gitLog, err)
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
			configPath, _ := localTestFiles(t, "[org]\nmax_seats = 3\n", "[org]\nmax_seats = 6\n")
			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Org.MaxSeats != 6 {
				t.Errorf("[org].max_seats = %d, want the personal file's 6", cfg.Org.MaxSeats)
			}
			if _, err := os.Stat(gitLog); err != nil {
				t.Errorf("Load ran no git to find the personal file: %v", err)
			}
		})
	}
}

// TestLoad_LocalAgmsgHome covers AC7's first half (D5): the personal file's
// [org].agmsg_home must be absolute or start with ~/, and an empty one keeps
// the backfill to the default. A relative value in ralph.toml keeps its
// meaning and still loads.
func TestLoad_LocalAgmsgHome(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	abs := filepath.Join(t.TempDir(), "agmsg")

	for _, value := range []string{"agmsg", "./agmsg", "../agmsg", "~", "~user/agmsg"} {
		t.Run("refuses "+value, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, "", "[org]\nagmsg_home = \""+value+"\"\n")
			_, err := Load(configPath)
			if err == nil {
				t.Fatalf("Load: nil error, want a refusal of agmsg_home %q", value)
			}
			want := localPath + `: [org].agmsg_home "` + value + `" must be an absolute path or start with ~/`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("Load: err = %v, want it to contain %q", err, want)
			}
		})
	}
	for _, tc := range []struct{ value, want string }{
		{abs, abs},
		{"~/agmsg", "~/agmsg"},
		{"", Default().Org.AgmsgHome},
	} {
		t.Run("accepts "+tc.value, func(t *testing.T) {
			configPath, _ := localTestFiles(t, "[org]\nagmsg_home = \"/from/ralph-toml\"\n", "[org]\nagmsg_home = \""+tc.value+"\"\n")
			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Org.AgmsgHome != tc.want {
				t.Errorf("[org].agmsg_home = %q, want %q", cfg.Org.AgmsgHome, tc.want)
			}
		})
	}
	t.Run("a relative value in ralph.toml loads", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\nagmsg_home = \"rel/agmsg\"\n", "[org]\nmax_seats = 2\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.AgmsgHome != "rel/agmsg" {
			t.Errorf("[org].agmsg_home = %q, want ralph.toml's %q", cfg.Org.AgmsgHome, "rel/agmsg")
		}
	})
}

// TestLoad_LocalPoolKeys covers orgPoolKeysPresent across the two files: a
// pool key counts as set when either document sets it. driver_pool set only
// in the personal file still filters the default model_pool to its drivers;
// a model_pool set in either file turns the filtering off.
func TestLoad_LocalPoolKeys(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	var claudeDefaults []OrgModelPoolEntry
	for _, entry := range Default().Org.ModelPool {
		if entry.Driver == "claude" {
			claudeDefaults = append(claudeDefaults, entry)
		}
	}

	t.Run("driver_pool only in the personal file", func(t *testing.T) {
		for _, ralphToml := range []string{"", "[org]\nmax_seats = 2\n"} {
			configPath, _ := localTestFiles(t, ralphToml, "[org]\ndriver_pool = [\"claude\"]\n")
			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load (ralph.toml %q): %v", ralphToml, err)
			}
			if !slices.Equal(cfg.Org.ModelPool, claudeDefaults) {
				t.Errorf("ralph.toml %q: [org].model_pool = %v, want the claude defaults %v", ralphToml, cfg.Org.ModelPool, claudeDefaults)
			}
		}
	})
	t.Run("model_pool in the personal file, driver_pool in ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\ndriver_pool = [\"claude\"]\n",
			"[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"opus\"\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if want := []OrgModelPoolEntry{{Driver: "claude", Model: "opus"}}; !slices.Equal(cfg.Org.ModelPool, want) {
			t.Errorf("[org].model_pool = %v, want %v", cfg.Org.ModelPool, want)
		}
	})
	t.Run("model_pool in ralph.toml, driver_pool in the personal file", func(t *testing.T) {
		configPath, localPath := localTestFiles(t,
			"[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"opus\"\n\n[[org.model_pool]]\ndriver = \"codex\"\nmodel = \"gpt-6-astra\"\n",
			"[org]\ndriver_pool = [\"claude\"]\n")
		_, err := Load(configPath)
		want := `[org].model_pool entry driver "codex" not present in [org].driver_pool [claude] (with ` + localPath + " applied)"
		if err == nil || err.Error() != want {
			t.Errorf("Load: err = %v, want %q: ralph.toml's explicit model_pool is not filtered", err, want)
		}
	})
}

// TestLoad_LinkedWorktreeReadsTheMainWorktreesLocalFile covers AC2 at the
// Load level: from a linked worktree's root and a subdirectory of either
// worktree, Load reads the main worktree root's personal file, not the
// same-named files beside the config file.
func TestLoad_LinkedWorktreeReadsTheMainWorktreesLocalFile(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	f := newLocalGitFixture(t)
	t.Chdir(f.outside)
	writeLocalTestFile(t, filepath.Join(f.main, LocalConfigFileName), "[org]\nmax_seats = 7\n")

	for i, dir := range []string{
		f.linked,
		filepath.Join(f.linked, "pkg", "sub"),
		filepath.Join(f.main, "pkg", "sub"),
	} {
		writeLocalTestFile(t, filepath.Join(dir, LocalConfigFileName), fmt.Sprintf("[org]\nmax_seats = %d\n", 2+i))
		configPath := filepath.Join(dir, "ralph.toml")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q): %v", configPath, err)
		}
		if cfg.Org.MaxSeats != 7 {
			t.Errorf("Load(%q): [org].max_seats = %d, want the main worktree root's 7", configPath, cfg.Org.MaxSeats)
		}
	}
}

// Tests of [org].state_dir, the key only the personal file may set (plan
// 2026-10-30-org-state-dir-local, D1 to D3, AC2 to AC4 and AC9).

// TestLoad_LocalOnlyKeyInRalphToml covers AC4 (D3): a ralph.toml that sets
// [org].state_dir, also to "" and in the dotted and inline-table forms, is an
// error naming the file and saying to set the key in ralph.local.toml,
// whether the personal file is read or off. A --config file of another name
// is refused the same way.
func TestLoad_LocalOnlyKeyInRalphToml(t *testing.T) {
	for _, tc := range []struct{ name, ralphToml string }{
		{"a value", "[org]\nmax_seats = 2\nstate_dir = \"/ledger\"\n"},
		{"an empty value", "[org]\nstate_dir = \"\"\n"},
		{"a dotted key", "org.state_dir = \"/ledger\"\n"},
		{"an inline table", "org = { state_dir = \"/ledger\" }\n"},
	} {
		for _, env := range []string{"", "off"} {
			t.Run(tc.name+" with "+LocalConfigEnv+"="+env, func(t *testing.T) {
				t.Setenv(LocalConfigEnv, env)
				configPath, _ := localTestFiles(t, tc.ralphToml, "[org]\nstate_dir = \"/personal\"\n")
				_, err := Load(configPath)
				want := configPath + ": [org].state_dir can be set only in ralph.local.toml, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in ralph.local.toml instead"
				if err == nil || err.Error() != want {
					t.Errorf("Load: err = %v, want exactly %q", err, want)
				}
			})
		}
	}
	t.Run("a --config file of another name", func(t *testing.T) {
		t.Setenv(LocalConfigEnv, "")
		other := filepath.Join(t.TempDir(), "other.toml")
		writeLocalTestFile(t, other, "[org]\nstate_dir = \"/ledger\"\n")
		_, err := Load(other)
		if want := other + ": [org].state_dir can be set only in ralph.local.toml"; err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Load: err = %v, want it to contain %q", err, want)
		}
	})
}

// TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults covers the config Load
// returns with the error above: Default(), with none of ralph.toml's other
// values and none of the personal file's, so `ralph doctor`, which goes on
// with it after saying "using defaults", does use the defaults (self-review
// L4 of plan 2026-10-30-org-state-dir-local).
func TestLoad_LocalOnlyKeyInRalphTomlReturnsDefaults(t *testing.T) {
	for _, env := range []string{"", "off"} {
		t.Run(LocalConfigEnv+"="+env, func(t *testing.T) {
			t.Setenv(LocalConfigEnv, env)
			configPath, _ := localTestFiles(t,
				"[org]\nmax_seats = 2\nstate_dir = \"/ledger\"\n\n[org.director]\nack_minutes = 20\n",
				"[org]\nmax_orgs = 9\n")
			cfg, err := Load(configPath)
			if want := configPath + ": [org].state_dir can be set only in ralph.local.toml"; err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("Load: err = %v, want it to contain %q", err, want)
			}
			if !reflect.DeepEqual(cfg, Default()) {
				t.Errorf("Load returned max_seats %d, max_orgs %d, ack_minutes %d with the error; want Default() as it is\ngot  %+v\nwant %+v",
					cfg.Org.MaxSeats, cfg.Org.MaxOrgs, cfg.Org.Director.AckMinutes, cfg, Default())
			}
		})
	}
}

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

// TestLocalOnlyKeys pins the list: each key is a key of Config, so a
// misspelled entry fails here instead of never matching, and
// checkLocalOnlyKeys refuses a document that sets it, with a value of any
// type, naming the key. A key of another name or in another table passes,
// and so does a document where the key's table is not a table. Each call
// returns a new slice.
func TestLocalOnlyKeys(t *testing.T) {
	keys := LocalOnlyKeys()
	if !slices.Contains(keys, "org.state_dir") {
		t.Errorf("LocalOnlyKeys() = %q, want org.state_dir in it", keys)
	}
	configKeys := tomlKeys(reflect.TypeFor[Config](), "")
	for _, key := range keys {
		if !slices.Contains(configKeys, key) {
			t.Errorf("LocalOnlyKeys(): %q is not a key of Config (%q)", key, configKeys)
			continue
		}
		dot := strings.LastIndex(key, ".")
		table, name := key[:dot], key[dot+1:]
		err := checkLocalOnlyKeys("/p/ralph.toml", []byte("["+table+"]\n"+name+" = 1\n"))
		if want := "/p/ralph.toml: [" + table + "]." + name + " can be set only in ralph.local.toml"; err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("checkLocalOnlyKeys(%s): err = %v, want it to contain %q", key, err, want)
		}
	}
	for _, doc := range []string{
		"[org]\nstate_dirs = \"/x\"\nmax_seats = 2\n",
		"[standard]\nstate_dir = \"/x\"\n",
		"[org.director]\nstate_dir = \"/x\"\n",
		"org = \"x\"\n",
		"",
	} {
		if err := checkLocalOnlyKeys("/p/ralph.toml", []byte(doc)); err != nil {
			t.Errorf("checkLocalOnlyKeys(%q) = %v, want nil", doc, err)
		}
	}
	keys[0] = "changed"
	if LocalOnlyKeys()[0] == "changed" {
		t.Error("LocalOnlyKeys() returned a slice a caller can change")
	}
}

// TestLocalOrgStateDir_Values covers AC3's config half (D2): an absolute
// value is returned cleaned and one starting with ~/ with the home directory
// in place of ~; an empty or absent state_dir, an empty personal file, and no
// personal file give "". path is the personal file's, also when it does not
// exist. ralph.toml is never read: one that sets the key, or that is not
// valid TOML, changes nothing.
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
