package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"charm.land/huh/v2"
)

// runAccessibleForm runs form in huh's accessible mode (what TERM=dumb turns
// on) with answers as its input, one answer per line, and returns what it
// printed with the terminal styling taken out.
//
// The input gives one byte per Read: accessible mode reads each answer with a
// bufio.Scanner of its own, which would take the later answers into its
// buffer from a reader that returns them all at once.
func runAccessibleForm(t *testing.T, form *huh.Form, answers string) string {
	t.Helper()
	var out bytes.Buffer
	err := form.WithAccessible(true).
		WithInput(iotest.OneByteReader(strings.NewReader(answers))).
		WithOutput(&out).
		Run()
	if err != nil {
		t.Fatalf("form.Run in accessible mode: %v\noutput:\n%s", err, out.String())
	}
	return stripTerminalStyling(out.String())
}

// terminalStyling matches an ANSI escape sequence (CSI or OSC), which the
// question styles put around their text.
var terminalStyling = regexp.MustCompile(`\x1b(\[[0-9;:?<=>]*[ -/]*[@-~]|\][^\x07\x1b]*(\x07|\x1b\\))`)

func stripTerminalStyling(s string) string {
	return terminalStyling.ReplaceAllString(s, "")
}

// settingsQuestionTitles returns the titles of initSettingsQuestions; with
// bothOnly it returns those of the questions asked only with both CLIs, and
// without it the others.
func settingsQuestionTitles(bothOnly bool) []string {
	var titles []string
	for _, q := range initSettingsQuestions {
		if q.bothOnly == bothOnly {
			titles = append(titles, q.title)
		}
	}
	return titles
}

func TestSettingsForm_AsksTheQuestionsOfTheCLIAndKeepsTheAnswers(t *testing.T) {
	tests := []struct {
		name    string
		cli     string
		answers string
		want    initSettings
	}{
		{
			name: "both, yes no yes no", cli: initCLIBoth, answers: "y\nn\ny\nn\n",
			want: initSettings{CLI: initCLIBoth, PlanReview: true, CI: true},
		},
		{
			name: "both, no yes no yes", cli: initCLIBoth, answers: "n\ny\nn\ny\n",
			want: initSettings{CLI: initCLIBoth, ChangeReview: true, AutoMerge: true},
		},
		{
			name: "both, all yes", cli: initCLIBoth, answers: "y\ny\ny\ny\n",
			want: initSettings{CLI: initCLIBoth, PlanReview: true, ChangeReview: true, CI: true, AutoMerge: true},
		},
		{
			name: "both, Enter keeps the initial answers", cli: initCLIBoth, answers: "\n\n\n\n",
			want: initSettings{CLI: initCLIBoth, AutoMerge: true},
		},
		{
			name: "claude, yes no", cli: initCLIClaude, answers: "y\nn\n",
			want: initSettings{CLI: initCLIClaude, CI: true},
		},
		{
			name: "codex, no yes", cli: initCLICodex, answers: "n\ny\n",
			want: initSettings{CLI: initCLICodex, AutoMerge: true},
		},
		{
			name: "codex, yes no", cli: initCLICodex, answers: "yes\nno\n",
			want: initSettings{CLI: initCLICodex, CI: true},
		},
		{
			// Four answers: a form that asked the two questions of both CLIs
			// would take the first two for them and leave CI and auto merge
			// at yes and yes.
			name: "claude, four answers given", cli: initCLIClaude, answers: "n\nn\ny\ny\n",
			want: initSettings{CLI: initCLIClaude},
		},
		{
			name: "claude, Enter keeps the initial answers", cli: initCLIClaude, answers: "\n\n",
			want: initSettings{CLI: initCLIClaude, AutoMerge: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newInitSettings(tt.cli)
			out := runAccessibleForm(t, newSettingsForm(&s), tt.answers)
			if s != tt.want {
				t.Errorf("answers %q gave %+v, want %+v\noutput:\n%s", tt.answers, s, tt.want, out)
			}

			asked := settingsQuestionTitles(false)
			notAsked := settingsQuestionTitles(true)
			if tt.cli == initCLIBoth {
				asked = append(settingsQuestionTitles(true), asked...)
				notAsked = nil
			}
			at := 0
			for _, title := range asked {
				i := strings.Index(out[at:], title)
				if i < 0 {
					t.Errorf("output should ask %q after the questions before it; got:\n%s", title, out)
					continue
				}
				at += i + len(title)
			}
			for _, title := range notAsked {
				if strings.Contains(out, title) {
					t.Errorf("output should not ask %q with one CLI; got:\n%s", title, out)
				}
			}
			if prompts := strings.Count(out, "[y/N]") + strings.Count(out, "[Y/n]"); prompts != len(asked) {
				t.Errorf("output holds %d yes/no prompts, want %d; got:\n%s", prompts, len(asked), out)
			}
		})
	}
}

func TestSettingsForm_StartsOnTheTemplateValues(t *testing.T) {
	for _, cli := range []string{initCLIClaude, initCLICodex, initCLIBoth} {
		got := newInitSettings(cli)
		want := initSettings{CLI: cli, PlanReview: false, ChangeReview: false, CI: false, AutoMerge: true}
		if got != want {
			t.Errorf("newInitSettings(%q) = %+v, want %+v", cli, got, want)
		}
	}
}

// stubLookPath returns a lookPath that finds the commands in found and no
// other, and records each name it is asked for in *asked.
func stubLookPath(found []string, asked *[]string) func(string) (string, error) {
	return func(file string) (string, error) {
		*asked = append(*asked, file)
		if slices.Contains(found, file) {
			return "/stub/bin/" + file, nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
}

func TestCLIQuestion_StartsOnTheCLIFoundAndRecommendsIt(t *testing.T) {
	claude := huh.NewOption("Claude Code", initCLIClaude)
	codex := huh.NewOption("Codex", initCLICodex)
	both := huh.NewOption("Both", initCLIBoth)
	tests := []struct {
		name            string
		found           []string
		wantInitial     string
		wantOptions     []huh.Option[string]
		wantRecommended int
	}{
		{name: "neither found", wantInitial: initCLIBoth, wantOptions: []huh.Option[string]{claude, codex, both}},
		{
			name: "claude only", found: []string{"claude"}, wantInitial: initCLIClaude,
			wantOptions:     []huh.Option[string]{huh.NewOption("Claude Code (Recommended)", initCLIClaude), codex, both},
			wantRecommended: 1,
		},
		{
			name: "codex only", found: []string{"codex"}, wantInitial: initCLICodex,
			wantOptions:     []huh.Option[string]{claude, huh.NewOption("Codex (Recommended)", initCLICodex), both},
			wantRecommended: 1,
		},
		{
			name: "both found", found: []string{"claude", "codex"}, wantInitial: initCLIBoth,
			wantOptions:     []huh.Option[string]{claude, codex, huh.NewOption("Both (Recommended)", initCLIBoth)},
			wantRecommended: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var asked []string
			lookPath := stubLookPath(tt.found, &asked)
			if got := cliOptions(recommendedCLI(lookPath)); !slices.Equal(got, tt.wantOptions) {
				t.Errorf("cliOptions =\n  %+v\nwant\n  %+v", got, tt.wantOptions)
			}
			slices.Sort(asked)
			if want := []string{"claude", "codex"}; !slices.Equal(asked, want) {
				t.Errorf("recommendedCLI asked lookPath for %q, want %q", asked, want)
			}

			cli := "unset"
			view := stripTerminalStyling(newCLIQuestion(lookPath, &cli).View())
			if cli != tt.wantInitial {
				t.Errorf("newCLIQuestion set the answer to %q, want %q", cli, tt.wantInitial)
			}
			description := cliQuestionDescription(recommendedCLI(lookPath))
			wants := append([]string{initCLIQuestionTitle}, strings.Split(description, "\n")...)
			for _, o := range tt.wantOptions {
				wants = append(wants, o.Key)
			}
			for _, want := range wants {
				if !strings.Contains(view, want) {
					t.Errorf("question view should show %q; got:\n%s", want, view)
				}
			}
			// The description explains the label when an option carries it,
			// so its own mention is not an option's.
			if n := strings.Count(view, "(Recommended)") - strings.Count(description, "(Recommended)"); n != tt.wantRecommended {
				t.Errorf("question view labels %d options (Recommended), want %d; got:\n%s", n, tt.wantRecommended, view)
			}

			// Enter keeps the answer the question starts on.
			cli = "unset"
			out := runAccessibleForm(t, huh.NewForm(huh.NewGroup(newCLIQuestion(lookPath, &cli))), "\n")
			if cli != tt.wantInitial {
				t.Errorf("Enter answered %q, want the initial %q\noutput:\n%s", cli, tt.wantInitial, out)
			}
		})
	}
}

func TestCLIQuestion_ExplainsRecommendedOnlyWhenAnOptionCarriesIt(t *testing.T) {
	tests := []struct {
		name      string
		found     []string
		wantLabel bool
	}{
		{name: "neither found"},
		{name: "claude only", found: []string{"claude"}, wantLabel: true},
		{name: "codex only", found: []string{"codex"}, wantLabel: true},
		{name: "both found", found: []string{"claude", "codex"}, wantLabel: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var asked []string
			var cli string
			question := func() *huh.Select[string] { return newCLIQuestion(stubLookPath(tt.found, &asked), &cli) }

			// The form's own view shows the description: the label on the one
			// option that carries it, and the note that explains the label.
			view := stripTerminalStyling(question().View())
			wantLabels := 0
			if tt.wantLabel {
				wantLabels = 1
			}
			if n := strings.Count(view, "(Recommended)"); n != 2*wantLabels {
				t.Errorf("view mentions (Recommended) %d times, want %d (the label and the note); got:\n%s", n, 2*wantLabels, view)
			}
			if has := strings.Contains(view, initCLIRecommendedNote); has != tt.wantLabel {
				t.Errorf("view shows the note %q = %v, want %v; got:\n%s", initCLIRecommendedNote, has, tt.wantLabel, view)
			}

			// Accessible mode (TERM=dumb) shows no description, so the label
			// is all it mentions.
			accessible := runAccessibleForm(t, huh.NewForm(huh.NewGroup(question())), "\n")
			if n := strings.Count(accessible, "(Recommended)"); n != wantLabels {
				t.Errorf("accessible mode mentions (Recommended) %d times, want %d; got:\n%s", n, wantLabels, accessible)
			}
		})
	}
}

func TestCLIQuestion_TheNumberChosenSetsTheCLI(t *testing.T) {
	tests := []struct {
		found  []string
		answer string
		want   string
	}{
		{answer: "1\n", want: initCLIClaude},
		{answer: "2\n", want: initCLICodex},
		{answer: "3\n", want: initCLIBoth},
		{found: []string{"claude"}, answer: "2\n", want: initCLICodex},
		{found: []string{"claude", "codex"}, answer: "1\n", want: initCLIClaude},
		// A number out of range is asked again.
		{found: []string{"codex"}, answer: "4\n3\n", want: initCLIBoth},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.found, "+")+" "+strings.TrimSpace(tt.answer), func(t *testing.T) {
			var asked []string
			var cli string
			out := runAccessibleForm(t, huh.NewForm(huh.NewGroup(newCLIQuestion(stubLookPath(tt.found, &asked), &cli))), tt.answer)
			if cli != tt.want {
				t.Errorf("answer %q gave %q, want %q\noutput:\n%s", tt.answer, cli, tt.want, out)
			}
			for i, o := range initCLIOptions {
				if line := string(rune('1'+i)) + ". " + o.label; !strings.Contains(out, line) {
					t.Errorf("output should list %q; got:\n%s", line, out)
				}
			}
		})
	}
}

// technicalTerm matches, as a word and in any case, a term of ralph's that
// the init questions must not show (Design decision D3 of the
// init-config-questions plan).
var technicalTerm = regexp.MustCompile(`(?i)\b(cross[- ]?review|regate|gate|advisory|driver|pool|director|leader|seat|org|watcher|per-plan|toml)\b`)

func TestInitSettingsQuestions_TechnicalTermMatcherFindsEachTerm(t *testing.T) {
	terms := []string{
		"cross-review", "cross review", "crossreview", "regate", "gate", "advisory", "driver", "pool",
		"director", "leader", "seat", "org", "watcher", "per-plan", "toml",
	}
	for _, term := range terms {
		for _, text := range []string{"a " + term + " b", strings.ToUpper(term) + ".", "ralph." + term} {
			if !technicalTerm.MatchString(text) {
				t.Errorf("technicalTerm should find %q in %q", term, text)
			}
		}
	}
	for _, text := range []string{"navigate", "gateway", "organize", "spool", "seated"} {
		if technicalTerm.MatchString(text) {
			t.Errorf("technicalTerm should find no term in %q: it matches words, not parts of words", text)
		}
	}
}

// initQuestionTexts returns every text the CLI question and the settings form
// show from the tables their builders read: titles, descriptions, and option
// labels with and without (Recommended).
func initQuestionTexts() []string {
	texts := []string{initCLIQuestionTitle, initCLIQuestionDescription, initCLIRecommendedNote}
	for _, o := range initCLIOptions {
		texts = append(texts, o.label, o.label+" (Recommended)")
	}
	for _, q := range initSettingsQuestions {
		texts = append(texts, q.title, q.description)
	}
	return texts
}

func TestInitSettingsQuestions_ShowNoTechnicalTerm(t *testing.T) {
	for _, text := range initQuestionTexts() {
		if term := technicalTerm.FindString(text); term != "" {
			t.Errorf("question text %q uses the term %q", text, term)
		}
	}

	// What the questions render, so that a text set outside the tables (a
	// Confirm's Yes and No, a label added in a builder) is checked too.
	var rendered []string
	for _, found := range [][]string{nil, {"claude"}, {"codex"}, {"claude", "codex"}} {
		var asked []string
		var cli string
		rendered = append(rendered, newCLIQuestion(stubLookPath(found, &asked), &cli).View())
		rendered = append(rendered, runAccessibleForm(t, huh.NewForm(huh.NewGroup(newCLIQuestion(stubLookPath(found, &asked), &cli))), "\n"))
	}
	for _, cli := range []string{initCLIClaude, initCLICodex, initCLIBoth} {
		s := newInitSettings(cli)
		for _, field := range newSettingsFields(&s) {
			view := field.View()
			if !strings.Contains(view, "Yes") || !strings.Contains(view, "No") {
				t.Errorf("a %s settings question should render its Yes and No buttons; got:\n%s", cli, view)
			}
			rendered = append(rendered, view)
		}
		rendered = append(rendered, runAccessibleForm(t, newSettingsForm(&s), strings.Repeat("\n", 4)))
	}
	for _, text := range rendered {
		text = stripTerminalStyling(text)
		if term := technicalTerm.FindString(text); term != "" {
			t.Errorf("a question renders the term %q; got:\n%s", term, text)
		}
	}
}

func TestInitSettingsQuestions_DescriptionLinesFitAnEightyColumnTerminal(t *testing.T) {
	descriptions := []string{initCLIQuestionDescription, initCLIRecommendedNote}
	for _, q := range initSettingsQuestions {
		descriptions = append(descriptions, q.description)
	}
	for _, description := range descriptions {
		for _, line := range strings.Split(description, "\n") {
			if len(line) > 76 {
				t.Errorf("description line %q is %d characters, want at most 76 so it does not wrap in an 80-column terminal", line, len(line))
			}
		}
	}

	// Each settings question shows its description, line by line.
	s := newInitSettings(initCLIBoth)
	fields := newSettingsFields(&s)
	if len(fields) != len(initSettingsQuestions) {
		t.Fatalf("newSettingsFields with both CLIs = %d questions, want %d", len(fields), len(initSettingsQuestions))
	}
	for i, field := range fields {
		view := stripTerminalStyling(field.View())
		for _, line := range strings.Split(initSettingsQuestions[i].description, "\n") {
			if !strings.Contains(view, line) {
				t.Errorf("question %q should show the description line %q; got:\n%s", initSettingsQuestions[i].title, line, view)
			}
		}
	}
}

func TestAskInitSettings_SkipsTheQuestionsWhenRalphTomlExists(t *testing.T) {
	const note = "ralph.toml already exists, so ralph init does not ask about its settings (edit ralph.toml to change them; --force asks, and overwrites every file init writes).\n"
	tests := []struct {
		name   string
		setup  func(t *testing.T, dir string)
		exists bool
	}{
		{name: "no ralph.toml", setup: func(*testing.T, string) {}},
		{name: "regular file", exists: true, setup: func(t *testing.T, dir string) {
			writeMarkerFile(t, filepath.Join(dir, "ralph.toml"))
		}},
		{name: "symlink to a file", exists: true, setup: func(t *testing.T, dir string) {
			file := filepath.Join(t.TempDir(), "elsewhere.toml")
			writeMarkerFile(t, file)
			symlinkMarker(t, file, filepath.Join(dir, "ralph.toml"))
		}},
		{name: "dangling symlink", exists: true, setup: func(t *testing.T, dir string) {
			symlinkMarker(t, filepath.Join(dir, "missing"), filepath.Join(dir, "ralph.toml"))
		}},
		{name: "directory", exists: true, setup: func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "ralph.toml"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		for _, force := range []bool{false, true} {
			name := tt.name
			if force {
				name += " with --force"
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				tt.setup(t, dir)
				var out bytes.Buffer
				got := askInitSettings(dir, force, &out)
				want := force || !tt.exists
				if got != want {
					t.Errorf("askInitSettings = %v, want %v", got, want)
				}
				switch {
				case want && out.Len() != 0:
					t.Errorf("askInitSettings should print nothing when it asks; got %q", out.String())
				case !want && out.String() != note:
					t.Errorf("askInitSettings printed %q, want the one line %q", out.String(), note)
				}
			})
		}
	}

	t.Run("target directory does not exist yet", func(t *testing.T) {
		if !askInitSettings(filepath.Join(t.TempDir(), "project"), false, io.Discard) {
			t.Error("askInitSettings = false, want true for a target directory that does not exist yet")
		}
	})
}
