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
// git does not run, or when the file does not exist (readLocalFile).
func readLocalConfig(configPath string) (*localConfig, error) {
	if os.Getenv(LocalConfigEnv) == "off" {
		return nil, nil
	}
	return readLocalFile(LocalPath(configPath))
}

// readLocalFile returns the personal file at path, or nil when it does not
// exist. Only fs.ErrNotExist counts as no file: any other read error (the
// path is a directory, it may not be read) is returned, and names the file's
// path.
func readLocalFile(path string) (*localConfig, error) {
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

// LocalOnlyKeys returns the dotted keys that only the personal file may set:
// values of one machine, such as where the org ledger is, which a committed
// ralph.toml would impose on every clone (plan 2026-10-30-org-state-dir-local,
// D3). Load refuses a ralph.toml that sets one (checkLocalOnlyKeys). Each
// call returns a new slice.
func LocalOnlyKeys() []string {
	return []string{"org.state_dir"}
}

// checkLocalOnlyKeys refuses data, the ralph.toml Load read from path, when
// it sets a key of LocalOnlyKeys, with any value, an empty one included, in
// any letter case toml.Unmarshal reads into the key (tomlDocHasKey). It
// reads that document alone, the way checkLocalAgmsgHome reads the personal
// one: once the personal file is applied on top, the merged config no longer
// tells which document set a key. The error names the key as LocalOnlyKeys
// spells it, not as the document does.
func checkLocalOnlyKeys(path string, data []byte) error {
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return err
	}
	for _, key := range LocalOnlyKeys() {
		if !tomlDocHasKey(doc, key) {
			continue
		}
		dot := strings.LastIndex(key, ".")
		return fmt.Errorf("%s: [%s].%s can be set only in %s, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in %s instead",
			path, key[:dot], key[dot+1:], LocalConfigFileName, LocalConfigFileName)
	}
	return nil
}

// tomlDocHasKey reports whether doc, a TOML document read into a map, sets
// key, a dotted name such as "org.state_dir", the way toml.Unmarshal reads
// the document into a struct. go-toml v2 matches a key to a field by the
// field's name, else by strings.ToLower of both (structFieldPath), so
// [ORG] STATE_DIR sets org.state_dir too. TOML keeps keys that differ in
// letter case only apart, and the decoder applies each of them to the one
// field, so every such key of a table is followed: [org] and [ORG] are two
// tables of one document, and state_dir and STATE_DIR two keys of one table.
func tomlDocHasKey(doc map[string]any, key string) bool {
	segment, rest, nested := strings.Cut(key, ".")
	// strings.ToLower as the decoder has it, not strings.EqualFold, which
	// differs for a few letters: U+0130 lowercases to "i" but does not fold
	// to it, and U+017F folds to "s" but lowercases to itself.
	lower := strings.ToLower(segment)
	for name, v := range doc {
		if strings.ToLower(name) != lower {
			continue
		}
		if !nested {
			return true
		}
		if table, ok := v.(map[string]any); ok && tomlDocHasKey(table, rest) {
			return true
		}
	}
	return false
}

// LocalOrgStateDir returns the [org].state_dir of the personal file of the
// repository dir is in ("" is the working directory): the file
// LocalPath(dir/ralph.toml) names, so a linked worktree and a subdirectory
// read the one at the main worktree's root. It reads that file alone, never
// ralph.toml, which may not set the key (LocalOnlyKeys), and never the file
// --config names (plan 2026-10-30-org-state-dir-local, D1). path is the
// file's path, also when it does not exist.
//
// value is "" when LocalConfigEnv is "off" (then git does not run, and path
// is "" too), when the file does not exist, and when it sets no state_dir or
// an empty one. A value that starts with "~/" is returned with the home
// directory (os.UserHomeDir) in place of "~", and an absolute value is
// returned cleaned. Any other value (a relative path, "~" alone, "~user/...")
// is an error naming the file's path and the value, for the reason
// checkLocalAgmsgHome gives (D2). Load does not check the value, so a
// command given --state-dir runs past a wrong one. A file that cannot be
// read or parsed is an error naming its path.
func LocalOrgStateDir(dir string) (value, path string, err error) {
	if os.Getenv(LocalConfigEnv) == "off" {
		return "", "", nil
	}
	path = LocalPath(filepath.Join(dir, "ralph.toml"))
	local, err := readLocalFile(path)
	if err != nil || local == nil {
		return "", path, err
	}
	var probe struct {
		Org struct {
			StateDir string `toml:"state_dir"`
		} `toml:"org"`
	}
	if err := toml.Unmarshal(local.data, &probe); err != nil {
		return "", path, fmt.Errorf("%s: %w", path, err)
	}
	stateDir := probe.Org.StateDir
	switch {
	case stateDir == "":
		return "", path, nil
	case strings.HasPrefix(stateDir, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", path, fmt.Errorf("%s: [org].state_dir %q: %w", path, stateDir, err)
		}
		return filepath.Join(home, strings.TrimPrefix(stateDir, "~/")), path, nil
	case filepath.IsAbs(stateDir):
		return filepath.Clean(stateDir), path, nil
	}
	return "", path, fmt.Errorf("%s: [org].state_dir %q must be an absolute path or start with ~/: %s is read from the repository's main worktree root, while ralph may run in a linked worktree or a subdirectory",
		path, stateDir, LocalConfigFileName)
}
