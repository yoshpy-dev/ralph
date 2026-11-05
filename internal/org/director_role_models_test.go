package org

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// A headless director with a model of "" (plan 2026-11-05-org-role-models,
// D5 and AC6): the director of the built-in [org.role_models] runs its
// driver CLI's own default model, so its registration records no model and
// its agent gets no --model, in herdr's agent start and in a launch cmd's
// launch script alike.

// TestDirectorAttachHeadless_EmptyModelIsRecorded: AttachHeadless accepts an
// empty model and records it as it is, and the registration reads back with
// no model.
func TestDirectorAttachHeadless_EmptyModelIsRecorded(t *testing.T) {
	s := NewDirectorStore(t.TempDir())
	s.token = func() (string, error) { return directorTestToken, nil }
	h := testHeadlessAttach()
	h.Model = ""

	reg, err := s.AttachHeadless(h)
	if err != nil {
		t.Fatalf("AttachHeadless with an empty model: %v", err)
	}
	if reg.ID != "d1" || reg.Model != "" || reg.Driver != h.Driver {
		t.Fatalf("Registration = %+v, want d1 on %s with no model", reg, h.Driver)
	}
	cur := mustReadDirector(t, s).Current
	if cur == nil || cur.ID != "d1" || cur.Model != "" || cur.State != DirectorStateStarting {
		t.Fatalf("current registration = %+v, want d1 starting with no model", cur)
	}
	var attached map[string]any
	if err := json.Unmarshal([]byte(directorFileLines(t, s)[0]), &attached); err != nil {
		t.Fatalf("attached line: %v", err)
	}
	if _, ok := attached["model"]; ok {
		t.Fatalf("attached = %v, want no model key for an empty model", attached)
	}
}

// TestDirectorAttachHeadless_RefusesAModelThatIsNotOneWord: AttachHeadless
// checks the model with config.CheckOrgModelName, as Spawn checks a seat's,
// since the CLI's ResolveRoleModel does not check it (self-review L2 of plan
// 2026-11-05-org-role-models): a model with a space,
// one that starts with "-", and one with a shell character are refused
// before anything is read or written, and "" and one plain word pass.
func TestDirectorAttachHeadless_RefusesAModelThatIsNotOneWord(t *testing.T) {
	for _, model := range []string{"not a model", "-c", "$(id)", "opus\n"} {
		t.Run(model, func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "absent", "org")
			s := NewDirectorStore(state)
			s.token = func() (string, error) { return directorTestToken, nil }
			h := testHeadlessAttach()
			h.Model = model

			reg, err := s.AttachHeadless(h)
			want := "org: director: start: model " + strconv.Quote(model) + ` must be ""`
			if err == nil || !strings.HasPrefix(err.Error(), want) || reg.ID != "" {
				t.Fatalf("AttachHeadless(model %q) = %+v, %v; want an error starting with %q", model, reg, err, want)
			}
			if _, err := os.Stat(state); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("a refused attach made the state dir (stat err %v)", err)
			}
		})
	}
	for _, model := range []string{"", "gpt-6.1-sol", "claude-opus-4@20260101"} {
		s := NewDirectorStore(t.TempDir())
		s.token = func() (string, error) { return directorTestToken, nil }
		h := testHeadlessAttach()
		h.Model = model
		if reg, err := s.AttachHeadless(h); err != nil || reg.Model != model {
			t.Fatalf("AttachHeadless(model %q) = %+v, %v; want it recorded", model, reg, err)
		}
	}
}

// TestStartDirector_EmptyModel_HerdrAgentStartHasNoModelFlag: a herdr start
// with an empty model starts the agent with the permission flags and the
// prompt pointer only, and records the running registration with no model.
func TestStartDirector_EmptyModel_HerdrAgentStartHasNoModelFlag(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	res := f.mustStart(t, DirectorDriverClaude, "")

	wantPrompt := filepath.Join(f.dir, "prompts", "director_"+directorTestToken+".md")
	want := []string{"--permission-mode", "bypassPermissions", PromptFilePointer(wantPrompt)}
	if len(f.h.agentStartArgs) != 1 || !slices.Equal(f.h.agentStartArgs[0], want) {
		t.Fatalf("agent args = %q, want %q", f.h.agentStartArgs, want)
	}
	if res.Registration.Model != "" || res.Registration.State != DirectorStateRunning {
		t.Fatalf("Registration = %+v, want running with no model", res.Registration)
	}
	if cur := mustReadDirector(t, f.store).Current; cur == nil || cur.Model != "" || cur.Driver != DirectorDriverClaude {
		t.Fatalf("current registration = %+v, want claude with no model", cur)
	}
}

// TestDirectorLaunchScript_EmptyModel_ExecsWithoutModelFlag: the launch
// script of an empty model execs the agent with the permission flags and the
// prompt pointer only.
func TestDirectorLaunchScript_EmptyModel_ExecsWithoutModelFlag(t *testing.T) {
	s := testLaunchScript("/b")
	s.Model = ""
	text, err := s.text()
	if err != nil {
		t.Fatalf("text with an empty model: %v", err)
	}
	wantExec := "\nexec " + shellQuote(s.AgentPath) + " '--permission-mode' 'bypassPermissions' " + shellQuote(s.PromptPointer) + "\n"
	if !strings.HasSuffix(text, wantExec) {
		t.Fatalf("launch script =\n%s\nwant it to end with%s", text, wantExec)
	}
	if strings.Contains(text, "--model") {
		t.Fatalf("launch script passes --model for an empty model:\n%s", text)
	}
}

// TestStartDirector_EmptyModel_LaunchCmdAgentGetsNoModel runs a launch cmd
// start with an empty model end to end: the agent the launch script execs
// gets the permission flags and the prompt pointer, and no --model.
func TestStartDirector_EmptyModel_LaunchCmdAgentGetsNoModel(t *testing.T) {
	f := newCmdStartFixture(t, launchCmdInBackground, cmdStartOptions{})
	p := f.params(DirectorDriverClaude, "")
	p.TimeoutMS = 30000
	res, err := f.o.StartDirector(p)
	if err != nil {
		t.Fatalf("StartDirector: %v", err)
	}
	if res.Registration.Model != "" || !res.Registration.CmdLaunch() {
		t.Fatalf("Registration = %+v, want a launch cmd registration with no model", res.Registration)
	}
	wantPrompt := filepath.Join(f.dir, "prompts", "director_"+directorTestToken+".md")
	var args []string
	for _, line := range f.agentMarkerLines(t) {
		if arg, ok := strings.CutPrefix(line, "arg="); ok {
			args = append(args, arg)
		}
	}
	if want := []string{"--permission-mode", "bypassPermissions", PromptFilePointer(wantPrompt)}; !slices.Equal(args, want) {
		t.Fatalf("the agent got %q, want %q", args, want)
	}
}

// TestDirectorAgentArgs pins the one order both launches use: the
// permission flags, --model and the model when there is one, then the
// prompt pointer.
func TestDirectorAgentArgs(t *testing.T) {
	perm := []string{"--permission-mode", "bypassPermissions"}
	if got, want := directorAgentArgs(perm, "opus", "p"), []string{"--permission-mode", "bypassPermissions", "--model", "opus", "p"}; !slices.Equal(got, want) {
		t.Fatalf("directorAgentArgs(opus) = %q, want %q", got, want)
	}
	if got, want := directorAgentArgs(perm, "", "p"), []string{"--permission-mode", "bypassPermissions", "p"}; !slices.Equal(got, want) {
		t.Fatalf("directorAgentArgs(\"\") = %q, want %q", got, want)
	}
	if got, want := directorAgentArgs(nil, "", "p"), []string{"p"}; !slices.Equal(got, want) {
		t.Fatalf("directorAgentArgs(nil, \"\") = %q, want %q", got, want)
	}
	directorAgentArgs(perm, "opus", "p")
	if !slices.Equal(perm, []string{"--permission-mode", "bypassPermissions"}) {
		t.Fatalf("directorAgentArgs changed its permArgs to %q", perm)
	}
}
