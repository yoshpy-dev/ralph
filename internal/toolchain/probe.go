package toolchain

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The probe: one shell script that, place by place, prints which file each
// tool resolves to and the first lines of its version command, the
// environment it runs in, and the reading of what it printed (plan
// 2026-11-12-gate-tool-versions, D1, D3, and D7).

// probeEnvOverrides are the variables ProbeEnv sets last (D7): mise does not
// install a missing version, and corepack neither pins a package manager in
// package.json, nor downloads one, nor asks to; npm does not check for its
// own update; and `go version` runs the go on PATH instead of downloading the
// toolchain a go.mod names (into the module cache, outside the worktree).
var probeEnvOverrides = []string{
	"MISE_AUTO_INSTALL=0",
	"MISE_NOT_FOUND_AUTO_INSTALL=0",
	"COREPACK_ENABLE_AUTO_PIN=0",
	"COREPACK_ENABLE_NETWORK=0",
	"COREPACK_ENABLE_DOWNLOAD_PROMPT=0",
	"NPM_CONFIG_UPDATE_NOTIFIER=false",
	"GOTOOLCHAIN=local",
}

// ProbeEnv returns the probe's environment: env without any entry named like
// one of the seven variables that keep the probe from changing anything
// (MISE_AUTO_INSTALL, MISE_NOT_FOUND_AUTO_INSTALL, COREPACK_ENABLE_AUTO_PIN,
// COREPACK_ENABLE_NETWORK, COREPACK_ENABLE_DOWNLOAD_PROMPT,
// NPM_CONFIG_UPDATE_NOTIFIER, GOTOOLCHAIN), then those seven with the values
// that do, last, so a caller's COREPACK_ENABLE_AUTO_PIN=1 or GOTOOLCHAIN=auto
// does not survive. env itself is not changed.
func ProbeEnv(env []string) []string {
	names := map[string]bool{}
	for _, kv := range probeEnvOverrides {
		name, _, _ := strings.Cut(kv, "=")
		names[name] = true
	}
	out := make([]string, 0, len(env)+len(probeEnvOverrides))
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); !names[name] {
			out = append(out, kv)
		}
	}
	return append(out, probeEnvOverrides...)
}

// versionArgs are the arguments of each tool's version command; a tool not
// listed gets --version.
var versionArgs = map[string][]string{
	ToolNode:    {"-v"},
	ToolPnpm:    {"-v"},
	ToolYarn:    {"-v"},
	ToolNpm:     {"-v"},
	ToolPython3: {"--version"},
	ToolGo:      {"version"},
}

// The probe's records: every line the script prints about a tool is
// recordPrefix, then four tab-separated fields: the kind, the place's Dir,
// the label (the tool's name, or miseLabel), and a value. The output a
// tool's version command prints goes between its begin and exit records as
// it is.
const (
	recordPrefix = "ralph-toolchain"
	miseLabel    = "mise-current"

	recordMissing = "missing" // the tool is not on PATH
	recordPath    = "path"    // value: what `command -v` printed
	recordBegin   = "begin"   // the version command's output follows
	recordExit    = "exit"    // value: the version command's exit status
	recordNoCD    = "nocd"    // the script could not cd into the place
	recordEnd     = "end"     // the script reached its end
)

// The most lines of output the reading keeps for a tool, and for
// `mise current`.
const (
	maxToolLines = 3
	maxMiseLines = 5
)

// probeFunc is the shell function the script defines: rtc_tool <label>
// <dir> <tool> <args...> prints the missing record, or the path record,
// the begin record, the tool's output with stdin from /dev/null and stderr
// with it, and the exit record. Each record starts on a line of its own
// even after output with no final newline. It uses only POSIX shell (sh,
// bash, zsh) and names no variable zsh reserves (path, status).
const probeFunc = `rtc_m='` + recordPrefix + `'
rtc_tool() {
	rtc_l=$1
	rtc_d=$2
	rtc_t=$3
	shift 3
	rtc_p=$(command -v "$rtc_t" 2>/dev/null)
	if [ "$?" -ne 0 ] || [ -z "$rtc_p" ]; then
		printf '\n%s\t` + recordMissing + `\t%s\t%s\t\n' "$rtc_m" "$rtc_d" "$rtc_l"
		return 0
	fi
	printf '\n%s\t` + recordPath + `\t%s\t%s\t%s\n' "$rtc_m" "$rtc_d" "$rtc_l" "$rtc_p"
	printf '%s\t` + recordBegin + `\t%s\t%s\t\n' "$rtc_m" "$rtc_d" "$rtc_l"
	"$rtc_t" "$@" </dev/null 2>&1
	rtc_rc=$?
	printf '\n%s\t` + recordExit + `\t%s\t%s\t%s\n' "$rtc_m" "$rtc_d" "$rtc_l" "$rtc_rc"
	return 0
}
`

// BuildScript returns the probe script for targets in the repository root
// root, an absolute path: for each place, in the order its first target
// comes, a subshell that cds into it by its absolute path (`cd '<root>'`
// for the root, `cd '<root>'/'<dir>'` for the others), so a login shell's
// startup files that cd elsewhere change nothing, and measures its tools in
// order; a place it cannot enter prints a nocd record instead. With
// miseCurrent, it then runs `mise current` the same way at the root when
// mise is on PATH. It ends with an end record. The script runs each tool's
// version command only (`node -v`, `python3 --version`, `go version`,
// ...), never installs or writes anything itself, and is written for sh,
// dash, bash, and zsh (/bin/sh -c for the gate, a login shell's -lc for
// doctor); a shell with another syntax, such as fish, fails it (no end
// record).
func BuildScript(root string, targets []Target, miseCurrent bool) string {
	var b strings.Builder
	b.WriteString(probeFunc)
	var dirs []string
	byDir := map[string][]string{}
	for _, t := range targets {
		if _, seen := byDir[t.Dir]; !seen {
			dirs = append(dirs, t.Dir)
		}
		byDir[t.Dir] = append(byDir[t.Dir], t.Tool)
	}
	for _, dir := range dirs {
		b.WriteString("(\n")
		writeCD(&b, root, dir)
		for _, tool := range byDir[dir] {
			fmt.Fprintf(&b, "\trtc_tool %s %s %s", shellQuote(tool), shellQuote(dir), shellQuote(tool))
			args, ok := versionArgs[tool]
			if !ok {
				args = []string{"--version"}
			}
			for _, a := range args {
				b.WriteString(" " + shellQuote(a))
			}
			b.WriteString("\n")
		}
		b.WriteString(")\n")
	}
	if miseCurrent {
		b.WriteString("(\n")
		writeCD(&b, root, RootDir)
		fmt.Fprintf(&b, "\trtc_tool %s %s mise current\n", shellQuote(miseLabel), shellQuote(RootDir))
		b.WriteString(")\n")
	}
	fmt.Fprintf(&b, "printf '\\n%%s\\t%s\\t.\\t\\t\\n' \"$rtc_m\"\n", recordEnd)
	return b.String()
}

// writeCD writes the line of a place's subshell that cds into dir, a
// place's Dir, by its absolute path under root (root itself for RootDir),
// each part single-quoted, or prints a nocd record for dir and leaves the
// subshell.
func writeCD(b *strings.Builder, root, dir string) {
	target := shellQuote(root)
	if dir != RootDir {
		target += "/" + shellQuote(dir)
	}
	fmt.Fprintf(b, "\tcd %s >/dev/null 2>&1 || { printf '\\n%%s\\t%s\\t%%s\\t\\t\\n' \"$rtc_m\" %s; exit 0; }\n",
		target, recordNoCD, shellQuote(dir))
}

// shellQuote returns s as one single-quoted shell word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Status is how a tool's measurement ended.
type Status string

const (
	// StatusFound: the tool is on PATH and its version command printed a
	// version token and exited 0.
	StatusFound Status = "found"
	// StatusMissing: `command -v` did not find the tool.
	StatusMissing Status = "missing"
	// StatusUnknown: the tool is on PATH but its version could not be read
	// (see Measurement.Reason), or the probe did not reach it.
	StatusUnknown Status = "unknown"
)

// reasonNotMeasured is the Reason of a target the script did not finish.
const reasonNotMeasured = "not measured"

// Measurement is one tool measured in one place.
type Measurement struct {
	Dir    string // the place's Dir
	Tool   string
	Status Status
	// Path is what `command -v` printed (a path, or a function or alias
	// name), with control characters replaced by "?"; "" when missing or
	// not reached.
	Path string
	// Shim is ShimOf(Path): ShimMise, ShimAsdf, or "".
	Shim string
	// Version is the version token (VersionToken) when Status is
	// StatusFound.
	Version string
	// Reason says why Status is StatusUnknown, on one line with no control
	// character, at most 200 bytes.
	Reason string
}

// MiseCurrent is what `mise current` printed at the root.
type MiseCurrent struct {
	Path     string   // what `command -v mise` printed
	Lines    []string // its first non-blank output lines, at most 5, control characters replaced
	ExitCode int      // -1 when the probe stopped before it ended
}

// Report is the reading of a probe's output.
type Report struct {
	// Measurements has one entry per target, in the targets' order.
	Measurements []Measurement
	// Mise is nil when `mise current` was not run: not asked for, mise not
	// on PATH, the root could not be entered, or the probe stopped before
	// it.
	Mise *MiseCurrent
	// Complete is whether the script printed its end record.
	Complete bool
	// TimedOut is whether Probe stopped the script at its timeout.
	TimedOut bool
	// Failure says, on one line, why the script did not complete or end in
	// time (a timeout, the shell's exit, a shell that could not start); ""
	// when it completed and ended on its own.
	Failure string
}

// probeState is what the records said about one label in one place.
type probeState struct {
	missing bool
	path    string
	begun   bool
	lines   []string
	exited  bool
	code    int
}

// ParseProbe reads out, what BuildScript(targets, ...) printed (other lines,
// such as a login shell's greeting, are ignored), into one Measurement per
// target: StatusMissing when the tool is not on PATH; StatusFound with the
// version token of the first of its first three non-blank output lines that
// holds one (VersionToken) when its version command exited 0; and
// StatusUnknown otherwise, with the reason: the exit status and the first
// output line, no output, no version in the output, the place could not be
// entered, or "not measured" when the output stops before the tool's exit
// record. A nocd record for a place does not change a tool there whose exit
// record came: the root can get a second nocd record, from the subshell of
// `mise current` (BuildScript), after its own subshell measured its tools.
// It also reads `mise current`'s lines and whether the end record came.
// Failure and TimedOut are left for the caller (Probe sets them).
func ParseProbe(out []byte, targets []Target) Report {
	type key struct{ dir, label string }
	states := map[key]*probeState{}
	noCD := map[string]bool{}
	var rep Report
	var open *probeState
	openMax := 0
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		fields := strings.SplitN(line, "\t", 5)
		if len(fields) == 5 && fields[0] == recordPrefix {
			open = nil
			kind, dir, label, value := fields[1], fields[2], fields[3], fields[4]
			switch kind {
			case recordNoCD:
				noCD[dir] = true
				continue
			case recordEnd:
				rep.Complete = true
				continue
			}
			k := key{dir, label}
			st := states[k]
			if st == nil {
				st = &probeState{code: -1}
				states[k] = st
			}
			switch kind {
			case recordMissing:
				st.missing = true
			case recordPath:
				st.path = value
			case recordBegin:
				st.begun = true
				open, openMax = st, maxToolLines
				if label == miseLabel {
					openMax = maxMiseLines
				}
			case recordExit:
				st.exited = true
				if code, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
					st.code = code
				}
			}
			continue
		}
		if open != nil && strings.TrimSpace(line) != "" && len(open.lines) < openMax {
			open.lines = append(open.lines, line)
		}
	}

	for _, t := range targets {
		m := Measurement{Dir: t.Dir, Tool: t.Tool, Status: StatusUnknown}
		st := states[key{t.Dir, t.Tool}]
		if st != nil && st.path != "" {
			m.Path = cleanText(st.path, maxPathBytes)
			m.Shim = ShimOf(m.Path)
		}
		switch {
		case st != nil && st.missing:
			m.Status = StatusMissing
		case noCD[t.Dir] && (st == nil || !st.exited):
			m.Reason = "cannot cd into the directory"
		case st == nil || !st.exited:
			m.Reason = reasonNotMeasured
		case st.code != 0:
			m.Reason = "exit " + strconv.Itoa(st.code)
			if len(st.lines) > 0 {
				m.Reason += ": " + strings.TrimSpace(st.lines[0])
			}
		case len(st.lines) == 0:
			m.Reason = "no output"
		default:
			for _, line := range st.lines {
				if v, ok := VersionToken(t.Tool, line); ok {
					m.Status, m.Version = StatusFound, v
					break
				}
			}
			if m.Status != StatusFound {
				m.Reason = "no version in the output: " + strings.TrimSpace(st.lines[0])
			}
		}
		m.Reason = cleanText(m.Reason, maxReasonBytes)
		rep.Measurements = append(rep.Measurements, m)
	}

	if st := states[key{RootDir, miseLabel}]; st != nil && !st.missing && st.begun {
		mc := &MiseCurrent{Path: cleanText(st.path, maxPathBytes), ExitCode: st.code}
		for _, line := range st.lines {
			mc.Lines = append(mc.Lines, cleanText(strings.TrimSpace(line), maxReasonBytes))
		}
		rep.Mise = mc
	}
	return rep
}

// ProbeOptions is what Probe runs.
type ProbeOptions struct {
	// Argv is the shell and its flags the script follows: {"/bin/sh", "-c"}
	// for the gate, {shell, "-lc"} for doctor.
	Argv []string
	// Dir is the repository root, where the script starts; the targets'
	// places are relative to it. Probe makes it absolute, and the script
	// cds into each place by its absolute path (BuildScript).
	Dir string
	// Env is the caller's environment; the script gets ProbeEnv(Env).
	Env []string
	// Timeout is the one budget of the whole script (D3).
	Timeout time.Duration
	// Targets are the tools to measure (Layout.Targets or
	// Layout.NodeTargets).
	Targets []Target
	// MiseCurrent also runs `mise current` at the root when mise is on PATH.
	MiseCurrent bool
}

// Probe builds the script for opts.Targets in opts.Dir, made absolute
// (BuildScript), runs it once with Run in that directory under
// ProbeEnv(opts.Env) within opts.Timeout, and reads its output (ParseProbe).
// When the script does not complete, Report.Failure says why, and every
// target it did not reach is StatusUnknown with the reason "not measured:
// <Failure>". The error is non-nil only when ctx is done
// (context.Cause(ctx)); the caller treats that as an interrupt and uses
// nothing of the probe. A shell that cannot start is a Failure, not an
// error.
func Probe(ctx context.Context, opts ProbeOptions) (Report, error) {
	root := opts.Dir
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	res, runErr := Run(ctx, opts.Argv, BuildScript(root, opts.Targets, opts.MiseCurrent), root, ProbeEnv(opts.Env), opts.Timeout)
	if ctx.Err() != nil {
		return Report{}, context.Cause(ctx)
	}
	rep := ParseProbe(res.Stdout, opts.Targets)
	rep.TimedOut = res.TimedOut
	switch {
	case runErr != nil:
		rep.Failure = runErr.Error()
	case res.TimedOut:
		rep.Failure = "timed out after " + opts.Timeout.String()
	case !rep.Complete:
		rep.Failure = "the shell exited " + strconv.Itoa(res.ExitCode) + " before the probe's end"
		if line := firstLine(string(res.Stderr)); line != "" {
			rep.Failure += ": " + line
		}
	}
	if rep.Failure != "" {
		rep.Failure = cleanText(rep.Failure, maxReasonBytes)
		for i := range rep.Measurements {
			if rep.Measurements[i].Reason == reasonNotMeasured {
				rep.Measurements[i].Reason = cleanText(reasonNotMeasured+": "+rep.Failure, maxReasonBytes)
			}
		}
	}
	return rep, nil
}
