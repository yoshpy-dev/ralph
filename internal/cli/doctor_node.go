package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/toolchain"
)

// Check 14, "Node toolchain" (plan
// docs/plans/active/2026-11-12-gate-tool-versions.md, D6 and AC6, issue
// #311): the Node and package-manager versions a repository asks for
// (.node-version, .nvmrc, package.json's engines.node and packageManager)
// against the versions the user's non-interactive login shell ($SHELL -lc)
// resolves, in each place the gate measures (toolchain.FindLayout). A
// repository with no Node marker runs no shell. No line of the check is
// fail: mismatches, a probe that did not finish, and a tool that is not there
// are warn, and what it cannot compare is info. A probe interrupted by a
// signal is a warn line too, and then doctor, once it has printed every
// line, returns an error (runDoctorStateDir), so it exits 1.

var (
	// doctorNodeProbe is how checkNodeToolchain measures the tools:
	// toolchain.Probe. internal/cli's TestMain pins a fake that runs no
	// shell; a test of the check sets its own (setDoctorNodeProbe).
	doctorNodeProbe = toolchain.Probe
	// doctorNodeGetenv reads $SHELL for checkNodeToolchain. Tests replace it.
	doctorNodeGetenv = os.Getenv
	// doctorNodeTimeout is the probe's one budget, every place included.
	doctorNodeTimeout = 20 * time.Second
	// doctorNodeSignalContext returns the context the probe runs under, from
	// parent, and the function that releases it: one that ends at the first
	// SIGINT, SIGTERM, or SIGHUP (signal.NotifyContext). toolchain.Run starts
	// the probe's shell in a process group of its own, which a terminal's
	// Ctrl-C does not reach, so the probe stops (its group gets SIGTERM, then
	// SIGKILL) only when this context ends. Tests replace it.
	doctorNodeSignalContext = func(parent context.Context) (context.Context, context.CancelFunc) {
		return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	}
)

// maxNodeErrBytes caps the error of a failed listing of the tracked files
// that a detail quotes (nodeErrText).
const maxNodeErrBytes = 240

const nodeToolchainCheckName = "Node toolchain"

// doctorNodeFallbackShell runs the probe when $SHELL is empty or names a
// shell that cannot run its sh script.
const doctorNodeFallbackShell = "/bin/sh"

// doctorNodeShells are the base names of the shells that run the probe's
// script (sh syntax); any other $SHELL, such as fish, is replaced by
// doctorNodeFallbackShell.
var doctorNodeShells = []string{"sh", "bash", "zsh", "dash", "ksh", "mksh", "yash"}

// maxNodeRequirementFileBytes caps a .node-version, .nvmrc, or package.json
// the check reads; a bigger one is reported as unreadable.
const maxNodeRequirementFileBytes = 1 << 20

// maxNodeShownBytes caps a value from a file that a detail quotes.
const maxNodeShownBytes = 80

// The fields a place's requirements come from.
const (
	nodeFieldNodeVersion    = ".node-version"
	nodeFieldNvmrc          = ".nvmrc"
	nodeFieldEnginesNode    = "engines.node"
	nodeFieldPackageManager = "packageManager"
)

// nodeRequirement is one version a place asks for.
type nodeRequirement struct {
	field string // one of the nodeField* constants
	file  string // the file it was read from, slash-separated, relative to the root
	value string // the spec, the range, or packageManager's value
}

// String names the requirement in a detail: ".nvmrc (22)",
// `engines.node ">=22" in package.json`, "packageManager pnpm@10.10.0 in
// package.json".
func (r nodeRequirement) String() string {
	switch r.field {
	case nodeFieldEnginesNode:
		return fmt.Sprintf("engines.node %s in %s", nodeShow(r.value), r.file)
	case nodeFieldPackageManager:
		return fmt.Sprintf("packageManager %s in %s", r.value, r.file)
	}
	return fmt.Sprintf("%s (%s)", r.file, nodePlain(r.value))
}

// nodeFindings is what checkNodeToolchain found, before it is printed.
type nodeFindings struct {
	warns  []string // one doctor line each
	passes []string // what was compared and met
	infos  []string // what was not compared, without duplicates
}

func (f *nodeFindings) info(s string) {
	if !slices.Contains(f.infos, s) {
		f.infos = append(f.infos, s)
	}
}

// checkNodeToolchain is Check 14. It lists the places of targetDir
// (toolchain.FindLayout; when git cannot list the tracked files, the places
// of targetDir's own top-level files, and the lines say why git could not).
// With no JS place it runs nothing and returns one info line. Otherwise it
// runs the probe once with [$SHELL, -lc] (doctorNodeShell),
// toolchain.ProbeEnv of this process's environment, targetDir as the working
// directory, and doctorNodeTimeout, under doctorNodeSignalContext, so a
// SIGINT, SIGTERM, or SIGHUP while it runs stops the probe's process group
// and the check ends with one warn line that says the probe was interrupted,
// and returns the probe's error, the cause of the interrupt, as interrupted.
// Otherwise it compares each place's requirements (nodeRequirementsAt) with
// what it measured there (nodeToolchainResults), and interrupted is nil.
func checkNodeToolchain(targetDir string) (results []checkResult, interrupted error) {
	root := targetDir
	if abs, err := filepath.Abs(targetDir); err == nil {
		root = abs
	}
	ctx := context.Background()
	layout, findErr, err := doctorNodeLayout(ctx, root)
	if err != nil {
		return []checkResult{{Name: nodeToolchainCheckName, Status: "info",
			Detail: fmt.Sprintf("could not list the files of the directory: %v: skipped", err)}}, nil
	}
	targets := layout.NodeTargets()
	if len(targets) == 0 {
		where := "the tracked files"
		if findErr != nil {
			where = "the top-level files (the tracked files could not be listed: " + nodeErrText(findErr) + ")"
		}
		return []checkResult{{Name: nodeToolchainCheckName, Status: "info",
			Detail: "no Node markers (package.json, a lockfile, .node-version, .nvmrc) in " + where + ": skipped"}}, nil
	}
	shell, shellDesc := doctorNodeShell(doctorNodeGetenv("SHELL"))
	rep, err := probeNodeToolchain(ctx, toolchain.ProbeOptions{
		Argv:    []string{shell, "-lc"},
		Dir:     root,
		Env:     toolchain.ProbeEnv(os.Environ()),
		Timeout: doctorNodeTimeout,
		Targets: targets,
	})
	if err != nil {
		return []checkResult{{Name: nodeToolchainCheckName, Status: "warn",
			Detail: fmt.Sprintf("%s was interrupted (%v) before it finished measuring: nothing was compared", shellDesc, err)}}, err
	}
	var notes []string
	if findErr != nil {
		notes = append(notes, "the tracked files could not be listed ("+nodeErrText(findErr)+"), so only the top-level files were read")
	}
	if layout.Truncated > 0 {
		notes = append(notes, fmt.Sprintf("not measured: %d more place(s) beyond the first %d", layout.Truncated, toolchain.MaxPlaces))
	}
	return nodeToolchainResults(root, layout, rep, shellDesc, notes), nil
}

// probeNodeToolchain runs doctorNodeProbe with opts under
// doctorNodeSignalContext(ctx), which it releases when the probe returns, so
// the signals are caught only while the probe runs.
func probeNodeToolchain(ctx context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
	probeCtx, stop := doctorNodeSignalContext(ctx)
	defer stop()
	return doctorNodeProbe(probeCtx, opts)
}

// doctorNodeLayout returns the places of root: toolchain.FindLayout, or,
// when that fails (not a git repository, git failed, or it ran past
// toolchain.LsFilesTimeout), the layout of the regular files directly in
// root, with FindLayout's error as findErr. The error err is non-nil only
// when root itself cannot be listed.
func doctorNodeLayout(ctx context.Context, root string) (layout toolchain.Layout, findErr, err error) {
	l, findErr := toolchain.FindLayout(ctx, root)
	if findErr == nil {
		return l, nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return toolchain.Layout{}, findErr, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return toolchain.LayoutFromFiles(files, func(rel string) ([]byte, error) {
		data, _, err := readNodeRequirementFile(filepath.Join(root, filepath.FromSlash(rel)))
		return data, err
	}), findErr, nil
}

// nodeErrText is err's text for a detail, on one line: each control
// character and each byte of invalid UTF-8 replaced by "?", cut to at most
// maxNodeErrBytes bytes (on a character boundary, ending in "..." when cut).
func nodeErrText(err error) string {
	var b strings.Builder
	s := err.Error()
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if (r == utf8.RuneError && size <= 1) || r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			r = '?'
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) <= maxNodeErrBytes {
		return out
	}
	cut := maxNodeErrBytes - len("...")
	for cut > 0 && !utf8.RuneStart(out[cut]) {
		cut--
	}
	return out[:cut] + "..."
}

// doctorNodeShell returns the shell the probe runs with and how a detail
// names it: shellVar, $SHELL, when its base name is one of doctorNodeShells,
// and doctorNodeFallbackShell, with the reason, when it is empty or names
// another shell (fish's syntax is not sh's).
func doctorNodeShell(shellVar string) (shell, desc string) {
	switch {
	case shellVar == "":
		return doctorNodeFallbackShell, doctorNodeFallbackShell + " -lc ($SHELL is empty)"
	case !slices.Contains(doctorNodeShells, filepath.Base(shellVar)):
		return doctorNodeFallbackShell, fmt.Sprintf("%s -lc ($SHELL %s is not an sh-compatible shell: %s)",
			doctorNodeFallbackShell, shellVar, strings.Join(doctorNodeShells, ", "))
	}
	return shellVar, shellVar + " -lc"
}

// nodeToolchainResults compares the requirements of each place of layout
// with rep's measurements and returns the doctor lines: one warn line for a
// probe that did not finish, one per mismatch or tool not resolved where a
// requirement asks for it, and then, with no warn, one line that lists what
// was compared (pass) or says nothing was (info); with warn lines, what was
// not compared follows as one info line. notes, about the layout, go after
// what was not compared.
func nodeToolchainResults(root string, layout toolchain.Layout, rep toolchain.Report, shellDesc string, notes []string) []checkResult {
	f := &nodeFindings{}
	measured := map[toolchain.Target]toolchain.Measurement{}
	for _, m := range rep.Measurements {
		measured[toolchain.Target{Dir: m.Dir, Tool: m.Tool}] = m
	}
	measure := func(dir, tool string) toolchain.Measurement {
		if m, ok := measured[toolchain.Target{Dir: dir, Tool: tool}]; ok {
			return m
		}
		return toolchain.Measurement{Dir: dir, Tool: tool, Status: toolchain.StatusUnknown, Reason: "not measured"}
	}
	c := nodeComparer{shellDesc: shellDesc, probeFailed: rep.Failure != "", f: f}
	for _, p := range layout.Places {
		if !p.JS {
			continue
		}
		nodeReqs, pmReq := nodeRequirementsAt(root, p.Dir, f)
		c.checkNode(p, nodeReqs, measure(p.Dir, toolchain.ToolNode))
		if pmReq != nil {
			c.checkPackageManager(p, *pmReq, measure(p.Dir, p.PackageManager))
		}
	}
	for _, n := range notes {
		f.info(n)
	}

	var out []checkResult
	if rep.Failure != "" {
		out = append(out, checkResult{Name: nodeToolchainCheckName, Status: "warn",
			Detail: fmt.Sprintf("%s did not finish measuring: %s; the tools it did not reach are not compared", shellDesc, rep.Failure)})
	}
	for _, w := range f.warns {
		out = append(out, checkResult{Name: nodeToolchainCheckName, Status: "warn", Detail: w})
	}
	if len(out) > 0 {
		if len(f.infos) > 0 {
			out = append(out, checkResult{Name: nodeToolchainCheckName, Status: "info", Detail: strings.Join(f.infos, "; ")})
		}
		return out
	}
	if len(f.passes) > 0 {
		detail := shellDesc + ": " + strings.Join(append(slices.Clone(f.passes), f.infos...), "; ")
		return []checkResult{{Name: nodeToolchainCheckName, Status: "pass", Detail: detail}}
	}
	parts := append([]string{"nothing compared"}, f.infos...)
	parts = append(parts, "resolved "+nodeMeasuredList(layout, measure))
	return []checkResult{{Name: nodeToolchainCheckName, Status: "info", Detail: shellDesc + ": " + strings.Join(parts, "; ")}}
}

// nodeMeasuredList names what was measured at each JS place, for the line
// that says nothing was compared: "repo root: node v24.22.1, pnpm missing".
func nodeMeasuredList(layout toolchain.Layout, measure func(dir, tool string) toolchain.Measurement) string {
	var places []string
	for _, p := range layout.Places {
		if !p.JS {
			continue
		}
		var tools []string
		for _, tool := range []string{toolchain.ToolNode, p.PackageManager} {
			m := measure(p.Dir, tool)
			switch m.Status {
			case toolchain.StatusFound:
				tools = append(tools, tool+" "+m.Version)
			case toolchain.StatusMissing:
				tools = append(tools, tool+" missing")
			default:
				tools = append(tools, tool+" unknown")
			}
		}
		places = append(places, nodePlaceLabel(p.Dir)+": "+strings.Join(tools, ", "))
	}
	return strings.Join(places, "; ")
}

// nodePlaceLabel names a place in a detail.
func nodePlaceLabel(dir string) string {
	if dir == toolchain.RootDir {
		return "repo root"
	}
	return dir
}

// nodeComparer compares one place's requirements with its measurements and
// adds what it finds to f.
type nodeComparer struct {
	shellDesc   string
	probeFailed bool
	f           *nodeFindings
}

// coveredByFailure reports whether m is a tool the probe did not reach
// because it did not finish, which the probe's own warn line reports.
func (c nodeComparer) coveredByFailure(m toolchain.Measurement) bool {
	return c.probeFailed && m.Status == toolchain.StatusUnknown && strings.HasPrefix(m.Reason, "not measured")
}

// notResolved is the warn line for a tool the requirements ask for that the
// shell does not resolve, or whose version it could not read.
func (c nodeComparer) notResolved(p toolchain.Place, reqs []nodeRequirement, m toolchain.Measurement) string {
	names := make([]string, len(reqs))
	for i, r := range reqs {
		names[i] = r.String()
	}
	asked := fmt.Sprintf("%s: %s asks for %s", nodePlaceLabel(p.Dir), strings.Join(names, " and "), m.Tool)
	if m.Status == toolchain.StatusMissing {
		return fmt.Sprintf("%s, but %s finds no %s on PATH; install it, or put it first on PATH for a non-interactive login shell",
			asked, c.shellDesc, m.Tool)
	}
	where := ""
	if m.Path != "" {
		where = " (" + m.Path + ")"
	}
	return fmt.Sprintf("%s, but %s could not read the version of %s%s: %s", asked, c.shellDesc, m.Tool, where, m.Reason)
}

// checkNode compares the node requirements of place p with m, the node the
// shell resolved there. A spec or range it cannot read, and an alias, are
// info; when reqs is not empty and node is not resolved, that is one warn.
func (c nodeComparer) checkNode(p toolchain.Place, reqs []nodeRequirement, m toolchain.Measurement) {
	var readable []nodeRequirement
	for _, r := range reqs {
		if note := nodeRequirementNote(r); note != "" {
			c.f.info(note)
			continue
		}
		readable = append(readable, r)
	}
	if len(reqs) == 0 {
		return
	}
	if m.Status != toolchain.StatusFound {
		if !c.coveredByFailure(m) {
			c.f.warns = append(c.f.warns, c.notResolved(p, reqs, m))
		}
		return
	}
	for _, r := range readable {
		c.compareNode(p, r, m)
	}
}

// nodeRequirementNote returns why the node requirement r is not compared (an
// alias, a spec or range this check cannot read), or "" when it is.
func nodeRequirementNote(r nodeRequirement) string {
	if r.field == nodeFieldEnginesNode {
		if _, err := toolchain.ParseRange(r.value); err != nil {
			return fmt.Sprintf("engines.node %s in %s is not a range this check reads; not compared", nodeShow(r.value), r.file)
		}
		return ""
	}
	res, err := toolchain.MatchVersionSpec(r.value, "0.0.0")
	switch {
	case err != nil:
		return fmt.Sprintf("%s holds %s, which is not a version or a known alias; not compared", r.file, nodeShow(r.value))
	case res == toolchain.SpecAlias:
		return fmt.Sprintf("%s names the alias %s, which this check does not compare", r.file, nodeShow(r.value))
	}
	return ""
}

// compareNode compares the readable node requirement r with node m (found).
func (c nodeComparer) compareNode(p toolchain.Place, r nodeRequirement, m toolchain.Measurement) {
	var ok bool
	var err error
	if r.field == nodeFieldEnginesNode {
		var rng toolchain.Range
		if rng, err = toolchain.ParseRange(r.value); err == nil {
			ok, err = rng.Allows(m.Version)
		}
	} else {
		var res toolchain.SpecResult
		res, err = toolchain.MatchVersionSpec(r.value, m.Version)
		ok = res == toolchain.SpecMatch
	}
	label := nodePlaceLabel(p.Dir)
	switch {
	case err != nil:
		c.f.warns = append(c.f.warns, fmt.Sprintf("%s: %s asks for node, but the version %s resolves, %s, is not a full version: %v",
			label, r, c.shellDesc, nodeResolved(m), err))
	case ok:
		c.f.passes = append(c.f.passes, fmt.Sprintf("%s: node %s meets %s", label, m.Version, r))
	case r.field == nodeFieldEnginesNode:
		c.f.warns = append(c.f.warns, fmt.Sprintf("%s: %s does not allow node %s that %s resolves; "+
			"pin a version it allows with MISE_NODE_VERSION=<version>, or put such a node first on PATH",
			label, r, nodeResolved(m), c.shellDesc))
	default:
		want := strings.TrimPrefix(strings.TrimPrefix(r.value, "v"), "V")
		c.f.warns = append(c.f.warns, fmt.Sprintf("%s: %s asks for node %s, but %s resolves node %s; "+
			"pin it with MISE_NODE_VERSION=%s, or put node %s first on PATH",
			label, r.file, r.value, c.shellDesc, nodeResolved(m), want, want))
	}
}

// checkPackageManager compares the packageManager requirement r of place p
// with m, the package manager the shell resolved there (the place's,
// toolchain.Place.PackageManager): name and version must be the same, the
// "+<hash>" dropped.
func (c nodeComparer) checkPackageManager(p toolchain.Place, r nodeRequirement, m toolchain.Measurement) {
	name, version, err := toolchain.ParsePackageManager(r.value)
	if err != nil {
		c.f.info(fmt.Sprintf("packageManager %s in %s is not <name>@<full version>; not compared", nodeShow(r.value), r.file))
		return
	}
	if !slices.Contains([]string{toolchain.ToolPnpm, toolchain.ToolYarn, toolchain.ToolNpm}, name) {
		c.f.info(fmt.Sprintf("packageManager %s@%s in %s names a package manager this check does not measure", name, version, r.file))
		return
	}
	r.value = name + "@" + version
	label := nodePlaceLabel(p.Dir)
	if name != p.PackageManager {
		c.f.warns = append(c.f.warns, fmt.Sprintf("%s: %s names %s, but the place's package manager is %s (from %s); "+
			"make packageManager and the lockfile agree", label, r, name, p.PackageManager, p.PackageManagerFrom))
		return
	}
	if m.Status != toolchain.StatusFound {
		if !c.coveredByFailure(m) {
			c.f.warns = append(c.f.warns, c.notResolved(p, []nodeRequirement{r}, m))
		}
		return
	}
	if strings.TrimPrefix(m.Version, "v") == version {
		c.f.passes = append(c.f.passes, fmt.Sprintf("%s: %s %s meets %s", label, name, m.Version, r))
		return
	}
	enable := "corepack enable"
	if name == toolchain.ToolNpm {
		enable = "corepack enable npm"
	}
	c.f.warns = append(c.f.warns, fmt.Sprintf("%s: %s, but %s resolves %s %s; "+
		"run %s so %s follows packageManager, or pin it with MISE_%s_VERSION=%s",
		label, r, c.shellDesc, name, nodeResolved(m), enable, name, strings.ToUpper(name), version))
}

// nodeResolved names a found tool's version and the file it resolved to:
// "v24.22.1 (/opt/homebrew/bin/node)", with ", a mise shim" when it is one.
func nodeResolved(m toolchain.Measurement) string {
	if m.Path == "" {
		return m.Version
	}
	shim := ""
	if m.Shim != "" {
		shim = ", a " + m.Shim + " shim"
	}
	return fmt.Sprintf("%s (%s%s)", m.Version, m.Path, shim)
}

// nodeRequirementsAt returns the requirements of the place dir (relative to
// root): for node, the .node-version and .nvmrc in the nearest directory at
// or above dir, up to root, that holds either, and engines.node; and
// packageManager, nil when no package.json sets it. engines.node and
// packageManager each come from dir's own package.json when it sets the
// field, else from root's. What cannot be read is added to f as info.
func nodeRequirementsAt(root, dir string, f *nodeFindings) (node []nodeRequirement, pm *nodeRequirement) {
	node = nearestVersionFiles(root, dir, f)
	own := readNodePackageJSON(root, path.Join(dir, "package.json"))
	top := own
	if dir != toolchain.RootDir {
		top = readNodePackageJSON(root, "package.json")
	}
	for _, pj := range []nodePackageJSON{own, top} {
		if pj.problem != "" {
			f.info(fmt.Sprintf("%s %s; its engines.node and packageManager are not compared", pj.rel, pj.problem))
		}
	}
	if r, ok := pickNodeField(own, top, nodeFieldEnginesNode, f); ok {
		node = append(node, r)
	}
	if r, ok := pickNodeField(own, top, nodeFieldPackageManager, f); ok {
		pm = &r
	}
	return node, pm
}

// nearestVersionFiles returns the .node-version and .nvmrc of the nearest
// directory at or above dir, up to root, that holds either, both when it
// holds both. A file that cannot be read, or holds no version, stops the
// search and is added to f as info.
func nearestVersionFiles(root, dir string, f *nodeFindings) []nodeRequirement {
	for d := dir; ; d = path.Dir(d) {
		var reqs []nodeRequirement
		found := false
		for _, name := range []string{nodeFieldNodeVersion, nodeFieldNvmrc} {
			rel := path.Join(d, name)
			data, exists, err := readNodeRequirementFile(filepath.Join(root, filepath.FromSlash(rel)))
			switch {
			case !exists:
				continue
			case err != nil:
				f.info(fmt.Sprintf("could not read %s: %v; not compared", rel, err))
			default:
				if spec := toolchain.VersionFileSpec(data); spec != "" {
					reqs = append(reqs, nodeRequirement{field: name, file: rel, value: spec})
				} else {
					f.info(rel + " holds no version; not compared")
				}
			}
			found = true
		}
		if found || d == toolchain.RootDir {
			return reqs
		}
	}
}

// nodeJSONField is one field of a package.json.
type nodeJSONField struct {
	present bool   // the field is set (null counts as not set)
	value   string // its value, when it is a string
	problem string // why it cannot be used ("is not a string"), "" when it can
}

// nodePackageJSON is what checkNodeToolchain reads of one package.json.
type nodePackageJSON struct {
	rel         string // its path, slash-separated, relative to the root
	problem     string // why it could not be read, "" when it was or does not exist
	enginesNode nodeJSONField
	packageMgr  nodeJSONField
}

// readNodePackageJSON reads engines.node and packageManager from the
// package.json at rel. A file that does not exist sets nothing; one that
// cannot be read, or is not a JSON object, sets problem.
func readNodePackageJSON(root, rel string) nodePackageJSON {
	pj := nodePackageJSON{rel: rel}
	data, exists, err := readNodeRequirementFile(filepath.Join(root, filepath.FromSlash(rel)))
	if !exists {
		return pj
	}
	if err != nil {
		pj.problem = fmt.Sprintf("could not be read (%v)", err)
		return pj
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || fields == nil {
		pj.problem = "is not a JSON object"
		return pj
	}
	pj.packageMgr = nodeJSONString(fields[nodeFieldPackageManager])
	if raw, ok := fields["engines"]; ok && !nodeJSONNull(raw) {
		var engines map[string]json.RawMessage
		if json.Unmarshal(raw, &engines) != nil || engines == nil {
			pj.enginesNode = nodeJSONField{present: true, problem: "cannot be read: engines is not an object"}
		} else {
			pj.enginesNode = nodeJSONString(engines["node"])
		}
	}
	return pj
}

// nodeJSONString reads raw, a field's value, as a string field.
func nodeJSONString(raw json.RawMessage) nodeJSONField {
	if raw == nil || nodeJSONNull(raw) {
		return nodeJSONField{}
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nodeJSONField{present: true, problem: "is not a string"}
	}
	return nodeJSONField{present: true, value: s}
}

func nodeJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// pickNodeField returns field (engines.node or packageManager) as a
// requirement from own when own sets it, else from top. A value that is not
// a string is added to f as info, and gives no requirement.
func pickNodeField(own, top nodePackageJSON, field string, f *nodeFindings) (nodeRequirement, bool) {
	get := func(pj nodePackageJSON) nodeJSONField {
		if field == nodeFieldEnginesNode {
			return pj.enginesNode
		}
		return pj.packageMgr
	}
	for _, pj := range []nodePackageJSON{own, top} {
		v := get(pj)
		if !v.present {
			continue
		}
		if v.problem != "" {
			f.info(fmt.Sprintf("%s in %s %s; not compared", field, pj.rel, v.problem))
			return nodeRequirement{}, false
		}
		return nodeRequirement{field: field, file: pj.rel, value: v.value}, true
	}
	return nodeRequirement{}, false
}

// readNodeRequirementFile reads the regular file p, up to
// maxNodeRequirementFileBytes. exists is false, with no error, when p does
// not exist; a FIFO or a device is not opened, so the read cannot block.
func readNodeRequirementFile(p string) (data []byte, exists bool, err error) {
	fi, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if !fi.Mode().IsRegular() {
		return nil, true, errors.New("not a regular file")
	}
	file, err := os.Open(p)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = file.Close() }()
	data, err = io.ReadAll(io.LimitReader(file, maxNodeRequirementFileBytes+1))
	if err != nil {
		return nil, true, err
	}
	if len(data) > maxNodeRequirementFileBytes {
		return nil, true, fmt.Errorf("bigger than %d bytes", maxNodeRequirementFileBytes)
	}
	return data, true, nil
}

// nodeShow quotes s, a value read from a file, for a detail, cut to
// maxNodeShownBytes.
func nodeShow(s string) string {
	if len(s) > maxNodeShownBytes {
		return fmt.Sprintf("%q...", s[:maxNodeShownBytes])
	}
	return fmt.Sprintf("%q", s)
}

// nodePlain returns s as it is when it is at most maxNodeShownBytes of
// printable ASCII with no double quote ("24", "lts/*"), and nodeShow(s)
// otherwise.
func nodePlain(s string) string {
	if len(s) > maxNodeShownBytes || strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r > 0x7e || r == '"' }) {
		return nodeShow(s)
	}
	return s
}
