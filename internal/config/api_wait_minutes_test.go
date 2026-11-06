package config

import "testing"

// Tests of [org].api_wait_minutes (plan 2026-11-07-org-seat-resume, S1 and
// AC-9; spec FR-8): how long a claude seat and a claude headless director
// wait out an API error before they end the turn. The default is 30, 0 is
// allowed (no retry variables), and Load refuses a negative value.
// TestDefaultsLockStep checks that templates/base/ralph.toml sets the same
// default (orgTemplateDrift).

// TestDefault_OrgAPIWaitMinutesIs30 pins the built-in default.
func TestDefault_OrgAPIWaitMinutesIs30(t *testing.T) {
	if got := Default().Org.APIWaitMinutes; got != 30 {
		t.Errorf("[org].api_wait_minutes default = %d, want 30", got)
	}
}

// TestLoad_OrgAPIWaitMinutes loads a ralph.toml per case: an unset key keeps
// the default 30, 0 and a positive value load as written, and a negative
// value is refused with an error naming the key and the value.
func TestLoad_OrgAPIWaitMinutes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		toml    string
		want    int
		wantErr string
	}{
		{"no [org] section keeps the default", "[standard]\nplan_advisory = true\n", 30, ""},
		{"[org] without the key keeps the default", "[org]\nmax_seats = 3\n", 30, ""},
		{"zero is allowed", "[org]\napi_wait_minutes = 0\n", 0, ""},
		{"one is allowed", "[org]\napi_wait_minutes = 1\n", 1, ""},
		{"a larger value is allowed", "[org]\napi_wait_minutes = 90\n", 90, ""},
		{"minus one is refused", "[org]\napi_wait_minutes = -1\n", 0, "[org].api_wait_minutes must be >= 0, got -1"},
		{"a large negative value is refused", "[org]\napi_wait_minutes = -30\n", 0, "[org].api_wait_minutes must be >= 0, got -30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, _ := localTestFiles(t, tc.toml, "")
			cfg, err := Load(configPath)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("Load: err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Org.APIWaitMinutes != tc.want {
				t.Errorf("[org].api_wait_minutes = %d, want %d", cfg.Org.APIWaitMinutes, tc.want)
			}
		})
	}
}

// TestLoad_LocalFileSetsAPIWaitMinutes covers the personal file (spec FR-8:
// ralph.local.toml may set the key): its value overrides ralph.toml's, the
// merged value is the one checked, so a personal file may fix a negative
// value of ralph.toml, and a negative value of the personal file is refused
// with an error naming that file.
func TestLoad_LocalFileSetsAPIWaitMinutes(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")

	t.Run("the personal file overrides ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\napi_wait_minutes = 45\n", "[org]\napi_wait_minutes = 0\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.APIWaitMinutes != 0 {
			t.Errorf("[org].api_wait_minutes = %d, want the personal file's 0", cfg.Org.APIWaitMinutes)
		}
	})

	t.Run("the personal file fixes a negative value of ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\napi_wait_minutes = -1\n", "[org]\napi_wait_minutes = 10\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.APIWaitMinutes != 10 {
			t.Errorf("[org].api_wait_minutes = %d, want the personal file's 10", cfg.Org.APIWaitMinutes)
		}
	})

	t.Run("a negative value of the personal file is refused", func(t *testing.T) {
		configPath, localPath := localTestFiles(t, "[org]\napi_wait_minutes = 45\n", "[org]\napi_wait_minutes = -5\n")
		_, err := Load(configPath)
		want := "[org].api_wait_minutes must be >= 0, got -5 (with " + localPath + " applied)"
		if err == nil || err.Error() != want {
			t.Fatalf("Load: err = %v, want %q", err, want)
		}
	})

	t.Run("the personal file alone sets it", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "", "[org]\napi_wait_minutes = 5\n")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.APIWaitMinutes != 5 {
			t.Errorf("[org].api_wait_minutes = %d, want the personal file's 5", cfg.Org.APIWaitMinutes)
		}
	})
}
