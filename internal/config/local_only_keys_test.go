package config

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// Tests of [org].state_dir, the key only the personal file may set (plan
// 2026-10-30-org-state-dir-local, D1 to D3, AC2 to AC4 and AC9).

// TestLoad_LocalOnlyKeyInRalphToml covers AC4 (D3): a ralph.toml that sets
// [org].state_dir, also to "" and in the dotted and inline-table forms, is an
// error naming the file and saying to set the key in ralph.local.toml,
// whether the personal file is read or off. So is one that spells the table
// or the key in other letter cases, which toml.Unmarshal reads into the key
// all the same (cross-review cycle 1 of plan 2026-10-30-org-state-dir-local);
// the error names the key in lower case. A --config file of another name is
// refused the same way.
func TestLoad_LocalOnlyKeyInRalphToml(t *testing.T) {
	for _, tc := range []struct{ name, ralphToml string }{
		{"a value", "[org]\nmax_seats = 2\nstate_dir = \"/ledger\"\n"},
		{"an empty value", "[org]\nstate_dir = \"\"\n"},
		{"a dotted key", "org.state_dir = \"/ledger\"\n"},
		{"an inline table", "org = { state_dir = \"/ledger\" }\n"},
		{"an upper-case table and key", "[ORG]\nSTATE_DIR = \"/ledger\"\n"},
		{"a capitalized table and key", "[Org]\nState_Dir = \"/ledger\"\n"},
		{"an upper-case key", "[org]\nSTATE_DIR = \"/ledger\"\n"},
		{"an upper-case table", "[ORG]\nstate_dir = \"/ledger\"\n"},
		{"an upper-case dotted key", "ORG.STATE_DIR = \"/ledger\"\n"},
		{"an upper-case inline table", "ORG = { STATE_DIR = \"/ledger\" }\n"},
		{"the second of two tables", "[org]\nmax_seats = 2\n\n[ORG]\nSTATE_DIR = \"/ledger\"\n"},
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

// TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder pins that
// checkLocalOnlyKeys refuses a document exactly when toml.Unmarshal reads it
// into a Config with [org].state_dir set (cross-review cycle 1 of plan
// 2026-10-30-org-state-dir-local). go-toml v2 matches a key to a field by its
// name, else by strings.ToLower of both, not by strings.EqualFold: a key with
// U+0130 (capital I with dot above), which lowercases to "i" but does not
// fold to it, is read and refused, and one with U+017F (long s), which folds
// to "s" but lowercases to itself, is neither. Two keys, or two tables, that
// differ in letter case only are both read; a key of another name or table is
// not. The error names the key in lower case.
func TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder(t *testing.T) {
	for _, tc := range []struct {
		doc     string
		decoded bool
	}{
		{"[ORG]\nSTATE_DIR = \"/ledger\"\n", true},
		{"[oRg]\nsTaTe_DiR = \"/ledger\"\n", true},
		{"[org]\n\"STATE_D\U00000130R\" = \"/ledger\"\n", true},
		{"[org]\nstate_dir = \"/a\"\nSTATE_DIR = \"/b\"\n", true},
		{"[ORG]\nmax_seats = 2\n\n[org]\nstate_dir = \"/ledger\"\n", true},
		{"[org]\nmax_seats = 2\n\n[ORG]\nSTATE_DIR = \"/ledger\"\n", true},
		{"[org]\n\"\U0000017Ftate_dir\" = \"/ledger\"\n", false},
		{"[ORG]\nSTATE_DIRS = \"/ledger\"\n", false},
		{"[org]\nstate_dirs = \"/ledger\"\n", false},
		{"[ORGANIZATION]\nSTATE_DIR = \"/ledger\"\n", false},
		{"[organization]\nstate_dir = \"/ledger\"\n", false},
		{"[ORG.DIRECTOR]\nSTATE_DIR = \"/ledger\"\n", false},
	} {
		var cfg Config
		if err := toml.Unmarshal([]byte(tc.doc), &cfg); err != nil {
			t.Fatalf("toml.Unmarshal(%q): %v", tc.doc, err)
		}
		if decoded := cfg.Org.StateDir != ""; decoded != tc.decoded {
			t.Fatalf("toml.Unmarshal(%q) set [org].state_dir to %q, but the case says decoded=%v", tc.doc, cfg.Org.StateDir, tc.decoded)
		}
		err := checkLocalOnlyKeys("/p/ralph.toml", []byte(tc.doc))
		if refused := err != nil; refused != tc.decoded {
			t.Errorf("checkLocalOnlyKeys(%q) = %v, want refused=%v, since toml.Unmarshal sets [org].state_dir to %q", tc.doc, err, tc.decoded, cfg.Org.StateDir)
		}
		if want := "/p/ralph.toml: [org].state_dir can be set only in ralph.local.toml"; err != nil && !strings.Contains(err.Error(), want) {
			t.Errorf("checkLocalOnlyKeys(%q): err = %v, want it to contain %q", tc.doc, err, want)
		}
	}
}
