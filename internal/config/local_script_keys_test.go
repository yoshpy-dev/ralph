package config

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// Tests of the local-only keys the scripts read: [plan_visual].browser and
// opener, [standard].base_branch, and [verify].base and
// skip_branch_secret_scan (plan 2026-10-30-script-local-config, D2, D3, and
// AC1). Only the personal file may set them, so every test that loads a
// value first sets RALPH_LOCAL_CONFIG to "", as local_test.go does.

// scriptLocalOnlyKeys are the five keys, each with a TOML value of its type
// that passes validation.
var scriptLocalOnlyKeys = []struct{ table, key, value string }{
	{"plan_visual", "browser", `"/usr/bin/chromium"`},
	{"plan_visual", "opener", `"xdg-open"`},
	{"standard", "base_branch", `"main"`},
	{"verify", "base", `"HEAD~1"`},
	{"verify", "skip_branch_secret_scan", "true"},
}

// TestLocalOnlyKeys_HoldsTheScriptKeys pins the whole list: org.state_dir and
// the five keys the scripts read, in that order. Removing one of the five
// fails here, besides TestLoad_ScriptLocalOnlyKeysInRalphToml.
func TestLocalOnlyKeys_HoldsTheScriptKeys(t *testing.T) {
	want := []string{"org.state_dir"}
	for _, k := range scriptLocalOnlyKeys {
		want = append(want, k.table+"."+k.key)
	}
	if got := LocalOnlyKeys(); !slices.Equal(got, want) {
		t.Errorf("LocalOnlyKeys() = %q, want %q", got, want)
	}
}

// TestLoad_ScriptLocalOnlyKeysInRalphToml covers AC1: a ralph.toml that sets
// one of the five keys is an error naming the file and the key, in the
// sentence of org.state_dir's, whether the personal file is read or off and
// although the personal file sets the key too. So is one that spells the
// table or the key in another letter case, in the dotted and inline-table
// forms too: toml.Unmarshal reads each of them into the key, which the case
// checks first, so none of them is vacuous.
func TestLoad_ScriptLocalOnlyKeysInRalphToml(t *testing.T) {
	capitalize := func(s string) string { return strings.ToUpper(s[:1]) + s[1:] }
	for _, k := range scriptLocalOnlyKeys {
		upTable, upKey := strings.ToUpper(k.table), strings.ToUpper(k.key)
		lower := "[" + k.table + "]\n" + k.key + " = " + k.value + "\n"
		for _, tc := range []struct{ name, doc string }{
			{"lower case", lower},
			{"an upper-case table and key", "[" + upTable + "]\n" + upKey + " = " + k.value + "\n"},
			{"a capitalized table and key", "[" + capitalize(k.table) + "]\n" + capitalize(k.key) + " = " + k.value + "\n"},
			{"an upper-case dotted key", upTable + "." + upKey + " = " + k.value + "\n"},
			{"an upper-case inline table", upTable + " = { " + upKey + " = " + k.value + " }\n"},
		} {
			var decoded Config
			if err := toml.Unmarshal([]byte(tc.doc), &decoded); err != nil {
				t.Fatalf("toml.Unmarshal(%q): %v", tc.doc, err)
			}
			if decoded.PlanVisual == (PlanVisualConfig{}) && decoded.Standard == (StandardConfig{}) && decoded.Verify == (VerifyConfig{}) {
				t.Fatalf("toml.Unmarshal(%q) set no field, so the case would not test %s.%s", tc.doc, k.table, k.key)
			}
			for _, env := range []string{"", "off"} {
				t.Run(k.table+"."+k.key+" "+tc.name+" with "+LocalConfigEnv+"="+env, func(t *testing.T) {
					t.Setenv(LocalConfigEnv, env)
					configPath, _ := localTestFiles(t, tc.doc, lower)
					_, err := Load(configPath)
					want := configPath + ": [" + k.table + "]." + k.key + " can be set only in ralph.local.toml, the git-ignored personal file, since its value belongs to one machine; remove it from this file and set it in ralph.local.toml instead"
					if err == nil || err.Error() != want {
						t.Errorf("Load: err = %v, want exactly %q", err, want)
					}
				})
			}
		}
	}
}

// TestLoad_LocalScriptKeys covers AC1: the five keys default to "" and false,
// and stay so with no file, with a ralph.toml and a personal file that set
// other keys, and with their tables present but empty. The personal file
// sets each of them, an absolute browser path with spaces included, and its
// values load as written while ralph.toml's other keys still apply.
func TestLoad_LocalScriptKeys(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	def := Default()
	if def.PlanVisual != (PlanVisualConfig{}) || def.Verify != (VerifyConfig{}) || def.Standard.BaseBranch != "" {
		t.Errorf("Default(): plan_visual %+v, verify %+v, standard.base_branch %q; want all empty and false", def.PlanVisual, def.Verify, def.Standard.BaseBranch)
	}
	for _, tc := range []struct{ name, ralphToml, local string }{
		{"no file", "", ""},
		{"other keys", "[standard]\ncross_review = true\n", "[org]\nmax_seats = 3\n"},
		{"empty tables", "[plan_visual]\n\n[verify]\n", "[plan_visual]\n\n[verify]\n\n[standard]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, _ := localTestFiles(t, tc.ralphToml, tc.local)
			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.PlanVisual != def.PlanVisual || cfg.Verify != def.Verify || cfg.Standard.BaseBranch != def.Standard.BaseBranch {
				t.Errorf("Load: plan_visual %+v, verify %+v, standard.base_branch %q; want the defaults", cfg.PlanVisual, cfg.Verify, cfg.Standard.BaseBranch)
			}
		})
	}

	const chrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	configPath, _ := localTestFiles(t, "[standard]\nmax_pipeline_cycles = 3\n",
		"[standard]\nbase_branch = \"release/1.2\"\n\n"+
			"[plan_visual]\nbrowser = \""+chrome+"\"\nopener = \"none\"\n\n"+
			"[verify]\nbase = \"HEAD^\"\nskip_branch_secret_scan = true\n")
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantPlanVisual := PlanVisualConfig{Browser: chrome, Opener: "none"}
	wantVerify := VerifyConfig{Base: "HEAD^", SkipBranchSecretScan: true}
	if cfg.PlanVisual != wantPlanVisual || cfg.Verify != wantVerify || cfg.Standard.BaseBranch != "release/1.2" {
		t.Errorf("Load: plan_visual %+v, verify %+v, standard.base_branch %q; want %+v, %+v, %q",
			cfg.PlanVisual, cfg.Verify, cfg.Standard.BaseBranch, wantPlanVisual, wantVerify, "release/1.2")
	}
	if cfg.Standard.MaxPipelineCycles != 3 {
		t.Errorf("Load: standard.max_pipeline_cycles = %d, want ralph.toml's 3", cfg.Standard.MaxPipelineCycles)
	}
}

// scriptValueErrors holds, by dotted key, the format of the error
// validateScriptValues returns for a value of that key; %q is the value.
var scriptValueErrors = map[string]string{
	"standard.base_branch": `[standard].base_branch %q must be "" or a branch name of A-Z, a-z, 0-9, '.', '_', '/', and '-' that does not start with '-' and has no ".."`,
	"plan_visual.browser":  `[plan_visual].browser %q must be "", "none", or an absolute path with no control character (a "~/" is not expanded)`,
	"plan_visual.opener":   `[plan_visual].opener %q must be "", "none", or one command word with no white space or control character that does not start with '-'`,
	"verify.base":          `[verify].base %q must be "" or a git revision with no white space or control character that does not start with '-'`,
}

// TestLoad_LocalScriptValueValidation covers D3 and AC1: the values the
// personal file gives the four string keys pass or fail Load as D3 says, and
// each error names the key, the value, and the personal file. "" passes each
// key, and so does "none" the browser and the opener. base_branch takes the
// characters of a branch name only, so a revision such as HEAD~1 fails it,
// while verify.base takes one. A browser must be an absolute path, so a
// relative one, "~/" or "~", and one that starts with a space fail. A
// leading '-', white space, and a line break fail the other three.
func TestLoad_LocalScriptValueValidation(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	for _, tc := range []struct {
		key      string
		accepted []string
		rejected []string
	}{
		{
			key:      "standard.base_branch",
			accepted: []string{"", "main", "release/1.2", "feature_x-y.z", "origin/main"},
			rejected: []string{"-main", "--all", "a..b", "..", "a b", "main\n", "\tmain", "HEAD~1", "HEAD^", "a:b", "~/x", "ä", "a*b"},
		},
		{
			key: "plan_visual.browser",
			accepted: []string{
				"", "none", "/usr/bin/chromium",
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			},
			rejected: []string{"chromium", "./chromium", "bin/chromium", "~/chromium", "~", "None", "-x", " /usr/bin/chromium", "/usr/bin/chromium\n", "/usr/\tbin"},
		},
		{
			key:      "plan_visual.opener",
			accepted: []string{"", "none", "open", "xdg-open", "/usr/bin/open"},
			rejected: []string{"-n", "--new", "open -a Safari", "open\n", "\topen", " open", "open ", "a\tb"},
		},
		{
			key:      "verify.base",
			accepted: []string{"", "HEAD~1", "HEAD^", "origin/main", "abc1234", "main"},
			rejected: []string{"-x", "--all", "HEAD 1", "HEAD~1\n", "\tHEAD", " HEAD", "HEAD ", "origin/main\t"},
		},
	} {
		dot := strings.LastIndex(tc.key, ".")
		table, name := tc.key[:dot], tc.key[dot+1:]
		load := func(t *testing.T, value string) (Config, string, error) {
			t.Helper()
			configPath, localPath := localTestFiles(t, "", fmt.Sprintf("[%s]\n%s = %q\n", table, name, value))
			cfg, err := Load(configPath)
			return cfg, localPath, err
		}
		for _, value := range tc.accepted {
			t.Run(fmt.Sprintf("%s %q accepted", tc.key, value), func(t *testing.T) {
				cfg, _, err := load(t, value)
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				got := map[string]string{
					"standard.base_branch": cfg.Standard.BaseBranch,
					"plan_visual.browser":  cfg.PlanVisual.Browser,
					"plan_visual.opener":   cfg.PlanVisual.Opener,
					"verify.base":          cfg.Verify.Base,
				}[tc.key]
				if got != value {
					t.Errorf("%s = %q, want %q", tc.key, got, value)
				}
			})
		}
		for _, value := range tc.rejected {
			t.Run(fmt.Sprintf("%s %q rejected", tc.key, value), func(t *testing.T) {
				_, localPath, err := load(t, value)
				want := fmt.Sprintf(scriptValueErrors[tc.key], value) + " (with " + localPath + " applied)"
				if err == nil || err.Error() != want {
					t.Errorf("Load: err = %v, want exactly %q", err, want)
				}
			})
		}
	}
}
