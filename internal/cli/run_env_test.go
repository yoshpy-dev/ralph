package cli

import (
	"strings"
	"testing"
)

// TestAppendEnvIfMissing_TomlFillsWhenEnvAbsent confirms that the loop
// driver value from ralph.toml reaches the orchestrator when the user has
// not already set the corresponding env var. This is the AC-4 path: TOML
// alone must be runtime-effective.
func TestAppendEnvIfMissing_TomlFillsWhenEnvAbsent(t *testing.T) {
	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	if !containsKV(got, "RALPH_LOOP_DRIVER", "codex") {
		t.Errorf("expected RALPH_LOOP_DRIVER=codex in %v", got)
	}
}

// TestAppendEnvIfMissing_EnvWinsOverToml confirms the documented priority:
// an explicit env var must NOT be overwritten by the TOML value.
func TestAppendEnvIfMissing_EnvWinsOverToml(t *testing.T) {
	env := []string{"PATH=/usr/bin", "RALPH_LOOP_DRIVER=claude"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	count := 0
	var lastVal string
	for _, e := range got {
		if strings.HasPrefix(e, "RALPH_LOOP_DRIVER=") {
			count++
			lastVal = strings.TrimPrefix(e, "RALPH_LOOP_DRIVER=")
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 RALPH_LOOP_DRIVER entry, got %d in %v", count, got)
	}
	if lastVal != "claude" {
		t.Errorf("env should win: got %q, want claude", lastVal)
	}
}

// TestAppendEnvIfMissing_DoesNotMatchPrefix guards against accidental matches
// when a variable's name is a prefix of another (e.g. RALPH_LOOP and
// RALPH_LOOP_DRIVER).
func TestAppendEnvIfMissing_DoesNotMatchPrefix(t *testing.T) {
	env := []string{"RALPH_LOOP_DRIVER_EXTRA=ignored"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	if !containsKV(got, "RALPH_LOOP_DRIVER", "codex") {
		t.Errorf("RALPH_LOOP_DRIVER not added; prefix match leaked: %v", got)
	}
}

func containsKV(env []string, key, value string) bool {
	want := key + "=" + value
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
