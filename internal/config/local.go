package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// LocalConfigFileName is the personal config file Load applies on top of
// ralph.toml: a git-ignored file of the same shape, whose keys override
// ralph.toml's key by key. LocalPath says where it is read from.
const LocalConfigFileName = "ralph.local.toml"

// LocalConfigEnv names the variable that makes Load skip the personal file
// when it is exactly "off" (any other value, empty included, or no value
// reads it). It is how `ralph org gate` and the tests keep a developer's
// ralph.local.toml out of what they run; it does not override any setting.
const LocalConfigEnv = "RALPH_LOCAL_CONFIG"

// LocalPath returns the path of the personal file for the config file at
// configPath (which need not exist). When configPath's directory is inside
// a git working tree, it is the ralph.local.toml at the root of that
// repository's main worktree, whichever worktree or subdirectory the
// directory is: a git-ignored file does not exist in a linked worktree (a
// standard-flow task worktree, a feature org's worktree), and a command run
// from a subdirectory without its own ./ralph.toml must still find the file
// (plan 2026-10-29-local-config-org, D2). When git names no main worktree
// (a linked worktree of a bare repository), it is the one at the toplevel of
// the working tree. Outside git, or when git fails, it is the one in
// configPath's directory.
//
// The directory is made absolute before git runs in it, since `git -C` with
// a relative directory would be read against the process's working
// directory, not configPath's.
func LocalPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if root, ok := gitroot.MainWorktree(dir); ok {
		return filepath.Join(root, LocalConfigFileName)
	}
	if top, err := gitroot.Toplevel(dir); err == nil && top != "" {
		return filepath.Join(top, LocalConfigFileName)
	}
	return filepath.Join(dir, LocalConfigFileName)
}

// localConfig is the personal file Load applies: its path and its content.
type localConfig struct {
	path string
	data []byte
}

// readLocalConfig returns the personal file for the config file at
// configPath (LocalPath), or nil when LocalConfigEnv is "off", in which case
// git does not run, or when the file does not exist. Only fs.ErrNotExist
// counts as no file: any other read error (the path is a directory, it may
// not be read) is returned, and names the file's path.
func readLocalConfig(configPath string) (*localConfig, error) {
	if os.Getenv(LocalConfigEnv) == "off" {
		return nil, nil
	}
	path := LocalPath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return &localConfig{path: path, data: data}, nil
}

// checkLocalAgmsgHome refuses a relative [org].agmsg_home in the personal
// file: the file is read from the main worktree's root, while the process
// that resolves the value may run in a linked worktree or a subdirectory,
// so a relative value would name a different agmsg depending on where ralph
// runs (plan 2026-10-29-local-config-org, D5). It reads the personal
// document alone, not the merged config, so a relative value in ralph.toml
// keeps its meaning (resolved from the working directory). An absolute
// value, one that starts with "~/", and an empty one (which Load backfills
// with the default) pass.
func checkLocalAgmsgHome(local *localConfig) error {
	var probe struct {
		Org struct {
			AgmsgHome string `toml:"agmsg_home"`
		} `toml:"org"`
	}
	if err := toml.Unmarshal(local.data, &probe); err != nil {
		return fmt.Errorf("%s: %w", local.path, err)
	}
	home := probe.Org.AgmsgHome
	if home == "" || filepath.IsAbs(home) || strings.HasPrefix(home, "~/") {
		return nil
	}
	return fmt.Errorf("%s: [org].agmsg_home %q must be an absolute path or start with ~/: %s is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory",
		local.path, home, LocalConfigFileName)
}
