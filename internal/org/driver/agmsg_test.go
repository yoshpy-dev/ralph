package driver

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestAgmsg_Send(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f}

	if err := a.Send(context.Background(), "ralph-org-1", "lead", "worker-1", "start now"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"--team", "ralph-org-1", "--as", "lead", "send", "worker-1", "start now"}
	if c := f.lastCall(); c.name != "agmsg" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=agmsg args=%v", c.name, c.args, want)
	}
}

func TestAgmsg_History(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  []string
	}{
		{
			name:  "no limit omits --limit",
			limit: 0,
			want:  []string{"--team", "ralph-org-1", "--as", "lead", "history"},
		},
		{
			name:  "positive limit includes --limit",
			limit: 20,
			want:  []string{"--team", "ralph-org-1", "--as", "lead", "history", "--limit", "20"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{"history output"}}
			a := Agmsg{R: f}
			got, err := a.History(context.Background(), "ralph-org-1", "lead", tt.limit)
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

func TestAgmsg_TeamMembers(t *testing.T) {
	f := &fakeRunner{outputs: []string{"lead, worker-1"}}
	a := Agmsg{R: f}

	got, err := a.TeamMembers(context.Background(), "ralph-org-1", "lead")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "lead, worker-1" {
		t.Fatalf("want %q, got %q", "lead, worker-1", got)
	}
	want := []string{"--team", "ralph-org-1", "--as", "lead", "team"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestAgmsg_Mode(t *testing.T) {
	for _, mode := range []string{"monitor", "turn", "both", "off"} {
		t.Run(mode, func(t *testing.T) {
			f := &fakeRunner{}
			a := Agmsg{R: f}
			if err := a.Mode(context.Background(), "ralph-org-1", "lead", mode); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := []string{"--team", "ralph-org-1", "--as", "lead", "mode", mode}
			if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
			}
		})
	}
}

func TestAgmsg_Mode_InvalidRejectedBeforeRun(t *testing.T) {
	f := &fakeRunner{}
	a := Agmsg{R: f}

	err := a.Mode(context.Background(), "ralph-org-1", "lead", "bogus-mode")
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
