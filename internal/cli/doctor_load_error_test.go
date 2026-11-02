package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestRunDoctorOpts_LoadErrorUsesDefaults covers AC3 of plan
// 2026-11-01-org-ledger-gaps (tech-debt 396(b)) through runDoctorOpts on an
// initialized project with claude, codex, and go stubbed. On these two error
// paths config.Load returns the Config with ralph.toml's values in it, not
// Default(): a value validation refuses in ralph.toml, and a personal
// ralph.local.toml that does not parse after ralph.toml was read. doctor
// prints the warn line that ends in "using defaults", and the checks after
// it read Default()'s values: the agmsg check looks at the default agmsg
// home under $HOME, not at ralph.toml's [org].agmsg_home, and the envelope
// line shows Default()'s max_seats, not ralph.toml's.
func TestRunDoctorOpts_LoadErrorUsesDefaults(t *testing.T) {
	def := config.Default().Org
	cases := []struct {
		name string
		// maxSeats is the [org].max_seats ralph.toml sets.
		maxSeats int
		// local is the ralph.local.toml written next to ralph.toml, or ""
		// for none (the personal file is then not read).
		local string
		// wantErr returns the start of the error the warn line names, given
		// the project dir.
		wantErr func(dir string) string
	}{
		{
			name:     "ralph.toml value refused",
			maxSeats: 0,
			wantErr:  func(string) string { return "[org].max_seats must be >= 1, got 0" },
		},
		{
			name:     "ralph.local.toml does not parse",
			maxSeats: def.MaxSeats + 2,
			local:    "[org\n",
			wantErr: func(dir string) string {
				return filepath.Join(dir, config.LocalConfigFileName) + ": "
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupTestEmbedFS(t)
			Version = "0.1.0-test"
			dir := t.TempDir()
			if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
				t.Fatalf("init: %v", err)
			}
			tomlPath := filepath.Join(dir, "ralph.toml")
			data, err := os.ReadFile(tomlPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "[org]") {
				t.Fatalf("the test scaffold's ralph.toml has an [org] table already:\n%s", data)
			}
			tomlHome := filepath.Join(dir, "no-such-agmsg-home")
			orgTable := fmt.Sprintf("\n[org]\nagmsg_home = %s\nmax_seats = %d\n", strconv.Quote(tomlHome), tc.maxSeats)
			if err := os.WriteFile(tomlPath, append(data, []byte(orgTable)...), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.local != "" {
				if err := os.WriteFile(filepath.Join(dir, config.LocalConfigFileName), []byte(tc.local), 0o644); err != nil {
					t.Fatal(err)
				}
				// TestMain turns the personal file off for the package.
				t.Setenv(config.LocalConfigEnv, "")
			}

			binDir := filepath.Join(dir, "bin")
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, bin := range []string{"claude", "codex", "go"} {
				writeStubBin(t, binDir, bin, "")
			}
			t.Setenv("PATH", binDir)
			home := t.TempDir()
			t.Setenv("HOME", home)
			setAgmsgHomeOverride(t, "")

			var out string
			if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
				t.Fatalf("a ralph.toml warning must not fail doctor: %v\noutput:\n%s", err, out)
			}
			warnPrefix := "  ⚠ ralph.toml: warn — parse error: "
			var warn string
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, warnPrefix) {
					warn = line
					break
				}
			}
			if !strings.HasPrefix(warn, warnPrefix+tc.wantErr(dir)) || !strings.HasSuffix(warn, " — using defaults") {
				t.Errorf("doctor output has no line %q ... — using defaults:\n%s", warnPrefix+tc.wantErr(dir), out)
			}
			if want := "agmsg: warn — agmsg not installed at " + filepath.Join(home, ".agents", "skills", "agmsg"); !strings.Contains(out, want) {
				t.Errorf("doctor output does not contain %q (the agmsg check must use the default [org].agmsg_home):\n%s", want, out)
			}
			if strings.Contains(out, "not installed at "+tomlHome) {
				t.Errorf("doctor's agmsg check used ralph.toml's [org].agmsg_home %s after saying it uses the defaults:\n%s", tomlHome, out)
			}
			if want := fmt.Sprintf("Org envelope: info — model_pool: %d entries, max_seats: %d\n", len(def.ModelPool), def.MaxSeats); !strings.Contains(out, want) {
				t.Errorf("doctor output does not contain %q (the envelope line must show the default [org].max_seats, not %d):\n%s", want, tc.maxSeats, out)
			}
		})
	}
}
