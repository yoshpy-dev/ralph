package org

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/toolchain"
)

// The tool versions a gate runs with (plan
// docs/plans/active/2026-11-12-gate-tool-versions.md, D1 to D5 and D7, issue
// #311). runGateScripts (gate_run.go) measures them right after the log's
// `# load at start:` line and before the first script (measureGateTools):
// which node, package manager, python3, and go /bin/sh resolves with the
// scripts' environment in each place toolchain.FindLayout lists, at which
// version. The log gets one line per tool and place (gateToolsLogLines), and
// the gate record and the CLI's TOOLS: line one value per tool
// (gateToolsSummary). Measuring changes nothing the scripts see: the probe
// gets toolchain.ProbeEnv of their environment, and the worktree's status is
// compared before and after it (D7).

var (
	// gateToolsProbe is how Gate and Regate measure the tools:
	// toolchain.Probe. Tests replace it (TestMain pins a fixed report).
	gateToolsProbe = toolchain.Probe
	// gateToolsTimeout is the probe's one budget, all places and tools
	// included (D3). Tests shorten it.
	gateToolsTimeout = 15 * time.Second
)

// gateToolsShell is the shell and flag the probe script follows (D1): the
// shell the scripts' shebangs most often name, not the caller's.
var gateToolsShell = []string{"/bin/sh", "-c"}

// gateToolNames are the tools a gate record and the TOOLS: line name, in
// their order.
var gateToolNames = [...]string{
	toolchain.ToolNode, toolchain.ToolPnpm, toolchain.ToolYarn, toolchain.ToolNpm, toolchain.ToolPython3, toolchain.ToolGo,
}

// The values of a tool in the record and the TOOLS: line other than a version
// token (D4): not on PATH in any place, not readable in any place, and not
// the same in every place (its version or its status differs between the
// places; the log has each place's).
const (
	gateToolMissing = "missing"
	gateToolUnknown = "unknown"
	gateToolMixed   = "mixed"
)

// gateToolValuePattern is the shape of a value a gate record holds for a
// tool: one token (toolchain.VersionToken's shape), so the record stays
// readable (gateRecordFromDetails).
var gateToolValuePattern = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// GateTool is one tool of GateTools.
type GateTool struct {
	// Name is the tool: node, pnpm, yarn, npm, python3, or go.
	Name string
	// Value is its version token (v24.22.1, 3.12.1, go1.27.3) when every
	// place it was measured in resolved the same one, missing or unknown when
	// it was so in every place, and mixed otherwise.
	Value string
}

// GateTools is what one Gate or Regate measured of the tools right before
// its scripts (plan 2026-11-12-gate-tool-versions, AC2 and AC5), as the CLI
// prints it (`TOOLS:`) and the gate record holds it (tool_dirs and tool_*).
type GateTools struct {
	// Dirs is how many places the tools were measured in.
	Dirs int
	// Tools has one entry per tool measured, in the order node, pnpm, yarn,
	// npm, python3, go.
	Tools []GateTool
}

// Mixed reports whether any tool's value is mixed: the places differ, and
// only the log says how.
func (g *GateTools) Mixed() bool {
	if g == nil {
		return false
	}
	return slices.ContainsFunc(g.Tools, func(t GateTool) bool { return t.Value == gateToolMixed })
}

// gateToolValue is m's value in the summary: its version token when it was
// found (unknown for one that is not a token), missing, or unknown.
func gateToolValue(m toolchain.Measurement) string {
	switch m.Status {
	case toolchain.StatusFound:
		if gateToolValuePattern.MatchString(m.Version) {
			return m.Version
		}
	case toolchain.StatusMissing:
		return gateToolMissing
	}
	return gateToolUnknown
}

// gateToolsSummary is the summary of measurements taken in dirs places: per
// tool of gateToolNames that was measured, the one value every place gave it
// (gateToolValue), or mixed when the places differ.
func gateToolsSummary(dirs int, measurements []toolchain.Measurement) *GateTools {
	values := map[string][]string{}
	for _, m := range measurements {
		values[m.Tool] = append(values[m.Tool], gateToolValue(m))
	}
	out := &GateTools{Dirs: dirs}
	for _, name := range gateToolNames {
		vs := values[name]
		if len(vs) == 0 {
			continue
		}
		v := vs[0]
		if slices.ContainsFunc(vs[1:], func(o string) bool { return o != v }) {
			v = gateToolMixed
		}
		out.Tools = append(out.Tools, GateTool{Name: name, Value: v})
	}
	return out
}

// gateToolsExplainLine is the log's first line about the tools: how they were
// measured, and what that does not cover (D1).
const gateToolsExplainLine = "# tools: measured before the scripts with their environment in /bin/sh, with auto-install and " +
	"auto-pin off; a child a script starts through a shim such as mise or asdf can see other versions"

// gateToolsLogLines are the log's lines about the tools layout lists and rep
// measured: gateToolsExplainLine; why layout is the root alone when the
// tracked files could not be listed (layoutErr); one `# tool <tool>
// (<place>): <value>` line per measurement, with the path the shell resolved
// and the shim it is, when any; the lines `mise current` printed; how many
// places beyond toolchain.MaxPlaces and tracked paths were left out; and why
// the probe did not finish, when it did not.
func gateToolsLogLines(layout toolchain.Layout, layoutErr error, rep toolchain.Report) []string {
	lines := []string{gateToolsExplainLine}
	if layoutErr != nil {
		lines = append(lines, "# tools: the tracked files could not be listed ("+gateToolsOneLine(layoutErr.Error())+
			"): python3 is measured at the root only")
	}
	for _, m := range rep.Measurements {
		var value string
		switch m.Status {
		case toolchain.StatusFound:
			value = m.Version
		case toolchain.StatusMissing:
			value = gateToolMissing
		default:
			value = gateToolUnknown + " (" + m.Reason + ")"
		}
		if m.Path != "" {
			value += " (" + m.Path + ")"
		}
		if m.Shim != "" {
			value += " (" + m.Shim + " shim)"
		}
		lines = append(lines, "# tool "+m.Tool+" ("+m.Dir+"): "+value)
	}
	if mc := rep.Mise; mc != nil {
		for _, line := range mc.Lines {
			lines = append(lines, "# mise current: "+line)
		}
		switch {
		case mc.ExitCode < 0:
			lines = append(lines, "# mise current: did not finish")
		case mc.ExitCode > 0:
			lines = append(lines, "# mise current: exited "+strconv.Itoa(mc.ExitCode))
		case len(mc.Lines) == 0:
			lines = append(lines, "# mise current: printed nothing")
		}
	}
	var left []string
	if layout.Truncated > 0 {
		left = append(left, fmt.Sprintf("%d more place(s) with a lockfile or a version file were not measured (the limit is %d)",
			layout.Truncated, toolchain.MaxPlaces))
	}
	if layout.SkippedPaths > 0 {
		left = append(left, fmt.Sprintf("%d tracked path(s) were skipped (a control character or invalid UTF-8 in the name)",
			layout.SkippedPaths))
	}
	if len(left) > 0 {
		lines = append(lines, "# tools: "+strings.Join(left, "; "))
	}
	if rep.Failure != "" {
		lines = append(lines, "# tools: the probe did not finish ("+rep.Failure+"): the tools it did not reach are unknown")
	}
	return lines
}

// gateToolsUnmeasured is the report of a probe that failed with failure before
// it measured anything: every target unknown, not measured.
func gateToolsUnmeasured(targets []toolchain.Target, failure string) toolchain.Report {
	rep := toolchain.Report{Failure: failure}
	for _, t := range targets {
		rep.Measurements = append(rep.Measurements, toolchain.Measurement{Dir: t.Dir, Tool: t.Tool, Status: toolchain.StatusUnknown,
			Reason: "not measured: " + failure})
	}
	return rep
}

// gateToolsOneLine is s with each control character replaced by "?", so it
// stays on one log line.
func gateToolsOneLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '?'
		}
		return r
	}, s)
}

// gateToolsTreeLines are the log's lines about the worktree's status before
// and after the probe (D7): none when both are the same; the lines in only
// one of them when they differ; and why they were not compared when one could
// not be read (beforeErr, afterErr). changed is how many lines differ.
func gateToolsTreeLines(before, after []string, beforeErr, afterErr error) (lines []string, changed int) {
	switch {
	case beforeErr != nil:
		return []string{"# tools: the worktree was not compared before and after the probe: " + gateToolsOneLine(beforeErr.Error())}, 0
	case afterErr != nil:
		return []string{"# tools: the worktree was not compared before and after the probe: " + gateToolsOneLine(afterErr.Error())}, 0
	}
	var diff []string
	for _, l := range after {
		if !slices.Contains(before, l) {
			diff = append(diff, l)
		}
	}
	for _, l := range before {
		if !slices.Contains(after, l) {
			diff = append(diff, l)
		}
	}
	if len(diff) == 0 {
		return nil, 0
	}
	lines = []string{"# tools: the probe changed the worktree; git status --porcelain --untracked-files=all differs in:"}
	for _, l := range diff {
		lines = append(lines, "#   "+l)
	}
	return lines, len(diff)
}

// measureGateTools measures the tools of the worktree root before the gate
// scripts (D1 to D3, D7): it lists the places with toolchain.FindLayout (the
// root alone, python3 only, when that fails), reads `git status --porcelain
// --untracked-files=all` in root, runs the probe once with gateToolsProbe
// (gateToolsShell, root, env, gateToolsTimeout, `mise current` included),
// reads the status again, and writes gateToolsLogLines and gateToolsTreeLines
// to log. env is the scripts' environment (gateScriptEnv); the probe gets
// toolchain.ProbeEnv of it. When the status changed, stderr gets one line that
// starts with w.prefix and says so; the scripts run as before. A probe that
// timed out or failed leaves the tools it did not reach unknown and is no
// error. The error is ctx's cause when ctx is done (the log then says the
// probe was interrupted), or a log write that failed, which starts with
// w.prefix.
func measureGateTools(ctx context.Context, root string, env []string, log, stderr io.Writer, w gateWording) (*GateTools, error) {
	write := func(lines ...string) error {
		for _, line := range lines {
			if _, err := fmt.Fprintf(log, "%s\n", line); err != nil {
				return fmt.Errorf("%swrite the log: %w", w.prefix, err)
			}
		}
		return nil
	}
	interrupted := func() (*GateTools, error) {
		_ = write("# tools: interrupted")
		return nil, context.Cause(ctx)
	}

	layout, layoutErr := toolchain.FindLayout(ctx, root)
	if ctx.Err() != nil {
		return interrupted()
	}
	if layoutErr != nil {
		layout = toolchain.RootLayout()
	}
	before, beforeErr := gateDirtyLines(ctx, w.prefix, root)
	if ctx.Err() != nil {
		return interrupted()
	}
	targets := layout.Targets()
	rep, err := gateToolsProbe(ctx, toolchain.ProbeOptions{
		Argv: gateToolsShell, Dir: root, Env: env, Timeout: gateToolsTimeout, Targets: targets, MiseCurrent: true,
	})
	switch {
	case ctx.Err() != nil:
		return interrupted()
	case err != nil:
		// toolchain.Probe returns an error only when ctx is done; any other
		// is a probe that failed, which leaves every tool unknown.
		rep = gateToolsUnmeasured(targets, gateToolsOneLine(err.Error()))
	}
	after, afterErr := gateDirtyLines(ctx, w.prefix, root)
	if ctx.Err() != nil {
		return interrupted()
	}

	treeLines, changed := gateToolsTreeLines(before, after, beforeErr, afterErr)
	if err := write(slices.Concat(gateToolsLogLines(layout, layoutErr, rep), treeLines)...); err != nil {
		return nil, err
	}
	if changed > 0 {
		if stderr == nil {
			stderr = io.Discard
		}
		_, _ = fmt.Fprintf(stderr, "%sthe version probe changed the worktree (%d line(s) of git status, listed in the gate log): the "+
			"scripts run as before, and a pass is recorded as tree_changed while the change is there\n", w.prefix, changed)
	}
	return gateToolsSummary(len(layout.Places), rep.Measurements), nil
}
