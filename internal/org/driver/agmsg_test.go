package driver

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// bashScript returns the expected `bash <home>/scripts/<script>` argv
// prefix, so every test below builds its "want" slice the same way the
// adapter does.
func bashScript(home, script string) []string {
	return []string{filepath.Join(home, "scripts", script)}
}

func TestAgmsg_Send(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f, Home: "/home/agmsg"}

	if err := a.Send(context.Background(), "ralph-org-1", "lead", "worker-1", "start now"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := append(bashScript("/home/agmsg", "send.sh"), "ralph-org-1", "lead", "worker-1", "start now")
	if c := f.lastCall(); c.name != "bash" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=bash args=%v", c.name, c.args, want)
	}
}

func TestAgmsg_Join(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f, Home: "/home/agmsg"}

	if err := a.Join(context.Background(), "ralph-org-1", "worker-1", "claude-code", "/repo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := append(bashScript("/home/agmsg", "join.sh"), "ralph-org-1", "worker-1", "claude-code", "/repo")
	if c := f.lastCall(); c.name != "bash" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=bash args=%v", c.name, c.args, want)
	}
}

func TestAgmsg_TeamMembers(t *testing.T) {
	f := &fakeRunner{outputs: []string{"lead, worker-1"}}
	a := Agmsg{R: f, Home: "/home/agmsg"}

	got, err := a.TeamMembers(context.Background(), "ralph-org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "lead, worker-1" {
		t.Fatalf("want %q, got %q", "lead, worker-1", got)
	}
	want := append(bashScript("/home/agmsg", "team.sh"), "ralph-org-1")
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestAgmsg_History(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		limit   int
		want    []string
	}{
		{
			name:    "neither agent nor limit",
			agentID: "",
			limit:   0,
			want:    append(bashScript("/home/agmsg", "history.sh"), "ralph-org-1"),
		},
		{
			name:    "agent only",
			agentID: "lead",
			limit:   0,
			want:    append(bashScript("/home/agmsg", "history.sh"), "ralph-org-1", "lead"),
		},
		{
			name:    "agent and limit",
			agentID: "lead",
			limit:   20,
			want:    append(bashScript("/home/agmsg", "history.sh"), "ralph-org-1", "lead", "20"),
		},
		{
			name:    "limit without agent is dropped (positional CLI cannot express it)",
			agentID: "",
			limit:   20,
			want:    append(bashScript("/home/agmsg", "history.sh"), "ralph-org-1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{"history output"}}
			a := Agmsg{R: f, Home: "/home/agmsg"}
			got, err := a.History(context.Background(), "ralph-org-1", tt.agentID, tt.limit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "history output" {
				t.Fatalf("want %q, got %q", "history output", got)
			}
			if c := f.lastCall(); !reflect.DeepEqual(c.args, tt.want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, tt.want)
			}
		})
	}
}

func TestAgmsg_Leave(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f, Home: "/home/agmsg"}

	if err := a.Leave(context.Background(), "ralph-org-1", "worker-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := append(bashScript("/home/agmsg", "leave.sh"), "ralph-org-1", "worker-1")
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestAgmsg_Whoami(t *testing.T) {
	tests := []struct {
		name      string
		agmsgType string
		want      []string
	}{
		{
			name:      "type omitted when empty",
			agmsgType: "",
			want:      append(bashScript("/home/agmsg", "whoami.sh"), "/repo"),
		},
		{
			name:      "type included when set",
			agmsgType: "claude-code",
			want:      append(bashScript("/home/agmsg", "whoami.sh"), "/repo", "claude-code"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{"whoami output"}}
			a := Agmsg{R: f, Home: "/home/agmsg"}
			got, err := a.Whoami(context.Background(), "/repo", tt.agmsgType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "whoami output" {
				t.Fatalf("want %q, got %q", "whoami output", got)
			}
			if c := f.lastCall(); !reflect.DeepEqual(c.args, tt.want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, tt.want)
			}
		})
	}
}

func TestAgmsg_DeliverySet(t *testing.T) {
	for _, mode := range []string{"monitor", "turn", "both", "off"} {
		t.Run(mode, func(t *testing.T) {
			f := &fakeRunner{}
			a := Agmsg{R: f, Home: "/home/agmsg"}
			if err := a.DeliverySet(context.Background(), mode, "claude-code", "/repo"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := append(bashScript("/home/agmsg", "delivery.sh"), "set", mode, "claude-code", "/repo")
			if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
			}
		})
	}
}

func TestAgmsg_DeliverySet_InvalidModeRejectedBeforeRun(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f, Home: "/home/agmsg"}

	err := a.DeliverySet(context.Background(), "bogus-mode", "claude-code", "/repo")
	if err == nil {
		t.Fatal("expected invalid mode to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "bogus-mode") {
		t.Fatalf("expected error to mention the invalid mode, got: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("expected no Run call for invalid mode, got %d calls: %v", len(f.calls), f.calls)
	}
}

func TestResolveAgmsgHome_Precedence(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	t.Run("env takes precedence over config and default", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "/env/agmsg-home")
		if got := ResolveAgmsgHome("/config/agmsg-home"); got != "/env/agmsg-home" {
			t.Fatalf("got %q, want %q", got, "/env/agmsg-home")
		}
	})

	t.Run("config takes precedence over default when env unset", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "")
		if got := ResolveAgmsgHome("/config/agmsg-home"); got != "/config/agmsg-home" {
			t.Fatalf("got %q, want %q", got, "/config/agmsg-home")
		}
	})

	t.Run("default used when env and config both unset", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "")
		want := filepath.Join(homeDir, ".agents", "skills", "agmsg")
		if got := ResolveAgmsgHome(""); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("tilde in env value is expanded", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "~/custom/agmsg")
		want := filepath.Join(homeDir, "custom", "agmsg")
		if got := ResolveAgmsgHome(""); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("tilde in config value is expanded", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "")
		want := filepath.Join(homeDir, "config-agmsg")
		if got := ResolveAgmsgHome("~/config-agmsg"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("non-tilde absolute path is unchanged", func(t *testing.T) {
		t.Setenv("RALPH_ORG_AGMSG_HOME", "")
		if got := ResolveAgmsgHome("/abs/agmsg-home"); got != "/abs/agmsg-home" {
			t.Fatalf("got %q, want %q", got, "/abs/agmsg-home")
		}
	})
}

func TestAgmsgVersion_RoundTrip(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "VERSION"), []byte("1.1.13\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := AgmsgVersion(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.1.13" {
		t.Fatalf("got %q, want %q", got, "1.1.13")
	}
}

func TestAgmsgVersion_MissingFile(t *testing.T) {
	home := t.TempDir() // no VERSION file.

	_, err := AgmsgVersion(home)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "VERSION") {
		t.Fatalf("expected error to mention VERSION, got: %v", err)
	}
}
