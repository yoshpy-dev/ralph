package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// checkCodexHookTrust is `ralph doctor`'s "Codex hook trust" check (plan
// 2026-11-12-codex-hook-trust-check, D3; issue #310), right after Check 3
// ("Codex effective config"). Check 3 confirms the project's hook wiring;
// this check confirms codex approved those hooks, since codex skips a hook
// it has no approval for, and a codex seat then runs without it. It reads
// every hooks.json codex reads for targetDir (the checkout root's and each
// nested layer's down to targetDir; plan D7 (a), AC10) and [hooks.state] of
// codex's user config (codexHookTrustFor), compares keys only, and never
// prints the config's content.
//
// Outcomes, in order:
//  1. codex is not in [org].driver_pool (codexSeatPossible) -> pass "not
//     needed"; no file is read.
//  2. resolveEnv fails -> info, approvals not checked.
//  3. the root layer's hooks.json is missing, or a layer's hooks.json is
//     unreadable or not of codex's shape (HooksProblems) -> info when the
//     only such file is the one Check 3 checks
//     (<targetDir>/.codex/hooks.json; Check 3 reports a problem with it, or,
//     when .codex/config.toml is missing, warns about that instead); warn
//     otherwise, naming the other files, since then nothing else says that
//     their hooks do not run in a codex seat (a linked worktree whose main
//     checkout has no valid hooks.json; plan advisory finding 2).
//  4. no hooks.json has a handler codex asks approval for -> pass.
//  5. codex's user config cannot be read or decoded -> warn, with the read
//     reason or only the line and column (codexConfigDecodeError).
//  6. codex's user config does not exist -> warn: no hook is approved.
//  7. a hook is not approved or is turned off (enabled = false) -> warn,
//     with the counts over every hooks.json, up to three hooks, and how to
//     approve them; with two root spellings (a main checkout opened through
//     a symlink), under both (AC9).
//  8. every hook is approved -> pass, saying that only the keys were
//     checked, not the hashes.
//
// Like Check 11b it never returns "fail": a hook codex skips is a real gap,
// but not one that should flip doctor's exit code.
func checkCodexHookTrust(orgCfg config.OrgConfig, targetDir string, resolveEnv func() (codexSandboxEnv, error)) checkResult {
	r := checkResult{Name: "Codex hook trust"}
	if !codexSeatPossible(orgCfg) {
		r.Status = "pass"
		r.Detail = "not needed: codex is not in [org].driver_pool, so no codex seat can be spawned"
		return r
	}
	env, err := resolveEnv()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve the codex config path: %v — hook approvals not checked", err)
		return r
	}
	report := codexHookTrustFor(targetDir, env)
	if problems := report.HooksProblems(); len(problems) > 0 {
		return codexHookTrustHooksProblem(r, report, problems, targetDir)
	}
	m := report.SlotCount()
	if m == 0 {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("no hook in %s needs approval", codexHookTrustPathList(report.ReadPaths(), "or"))
		return r
	}
	cfgDisplay := codexConfigDisplayPath(env.Home, env.ConfigPath)
	r.Status = "warn"
	switch {
	case report.ConfigState == codexHookSourceUnreadable || report.ConfigState == codexHookSourceInvalid:
		r.Detail = codexHookTrustConfigProblemDetail(report, cfgDisplay)
	case report.ConfigState == codexHookSourceMissing:
		r.Detail = fmt.Sprintf("no codex user config at %s, so none of the %s in %s is approved — %s; %s",
			cfgDisplay, codexHookNoun(m), codexHookTrustPathList(report.SlotPaths(), "and"),
			codexHookSkipClause(m), codexHookTrustApprovalStep(report.ApprovalDirs()))
	case len(report.NotApproved) > 0 || len(report.Off) > 0:
		r.Detail = codexHookTrustGapDetail(report, cfgDisplay)
	default:
		r.Status = "pass"
		r.Detail = fmt.Sprintf("%s in %s approved in %s (keys only; the hash is not checked — a hook changed since its approval makes codex show Hooks need review in the seat's pane)",
			codexHookNoun(m), codexHookTrustPathList(report.SlotPaths(), "and"), cfgDisplay)
	}
	return r
}

// codexHookTrustHooksProblem words outcome 3 of checkCodexHookTrust for
// problems, the layers whose hooks.json could not be read as codex reads
// one. It is info when the only such file is the one Check 3 reads, and
// warn naming every other one otherwise. The warning says no hook runs when
// no other hooks.json holds one, and that the named files' hooks do not run
// when another does.
func codexHookTrustHooksProblem(r checkResult, report codexHookTrustReport, problems []codexHookLayer, targetDir string) checkResult {
	var own string
	var others []string
	for _, layer := range problems {
		path := layer.HooksPath()
		if codexHookTrustSameDir(filepath.Dir(filepath.Dir(path)), targetDir) {
			own = path + " " + codexHookTrustLayerProblem(layer)
			continue
		}
		others = append(others, path+", which "+codexHookTrustLayerProblem(layer))
	}
	if len(others) == 0 {
		r.Status = "info"
		r.Detail = fmt.Sprintf("hook approvals not checked: %s (the same file Codex effective config checks)", own)
		return r
	}
	consequence := "so no hook runs in a codex seat"
	if report.SlotCount() > 0 {
		consequence = "so no hook from it runs in a codex seat"
		if len(others) > 1 {
			consequence = "so no hook from them runs in a codex seat"
		}
	}
	r.Status = "warn"
	r.Detail = fmt.Sprintf("codex seats read hooks from %s, %s", strings.Join(others, ", and from "), consequence)
	return r
}

// codexHookTrustLayerProblem says what is wrong with layer's hooks.json:
// "does not exist", "could not be read (<reason>)", or "is not valid
// (<reason>)".
func codexHookTrustLayerProblem(layer codexHookLayer) string {
	switch layer.State {
	case codexHookSourceMissing:
		return "does not exist"
	case codexHookSourceUnreadable:
		return fmt.Sprintf("could not be read (%s)", codexConfigReadReason(layer.Err))
	default:
		return fmt.Sprintf("is not valid (%v)", layer.Err)
	}
}

// codexHookTrustSameDir reports whether a and b are the same directory: the
// same file on disk when both exist, else the same absolute path. It tells
// whether the hooks.json the check reads is the one Check 3 reads.
func codexHookTrustSameDir(a, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	if aErr == nil && bErr == nil {
		return os.SameFile(aInfo, bInfo)
	}
	aAbs, aAbsErr := filepath.Abs(a)
	bAbs, bAbsErr := filepath.Abs(b)
	if aAbsErr != nil || bAbsErr != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return aAbs == bAbs
}

// codexHookTrustConfigProblemDetail words outcome 5 of checkCodexHookTrust:
// codex's user config exists but could not be read or decoded. A decode
// failure names only its line and column, never the config's text.
func codexHookTrustConfigProblemDetail(report codexHookTrustReport, cfgDisplay string) string {
	notChecked := fmt.Sprintf("the approvals of the %s in %s were not checked",
		codexHookNoun(report.SlotCount()), codexHookTrustPathList(report.SlotPaths(), "and"))
	var decodeErr *codexConfigDecodeError
	if !errors.As(report.ConfigErr, &decodeErr) {
		return fmt.Sprintf("could not read %s (%s) — %s", cfgDisplay, codexConfigReadReason(report.ConfigErr), notChecked)
	}
	if decodeErr.HasPosition {
		return fmt.Sprintf("%s could not be decoded as a codex config (line %d, column %d) — %s",
			cfgDisplay, decodeErr.Line, decodeErr.Column, notChecked)
	}
	return fmt.Sprintf("%s could not be decoded as a codex config — %s", cfgDisplay, notChecked)
}

// codexHookTrustMaxTails is how many key tails outcome 7 of
// checkCodexHookTrust names before it shortens the list with "…".
const codexHookTrustMaxTails = 3

// codexHookTrustGapDetail words outcome 7 of checkCodexHookTrust: the counts
// of hooks not approved and turned off over every hooks.json, the first
// hooks (not approved first, named by ItemName), and what to do about each
// kind.
func codexHookTrustGapDetail(report codexHookTrustReport, cfgDisplay string) string {
	m, n, k := report.SlotCount(), len(report.NotApproved), len(report.Off)
	paths := codexHookTrustPathList(report.SlotPaths(), "and")
	var head string
	switch {
	case n > 0 && k > 0:
		head = fmt.Sprintf("%d of %s in %s %s not approved in %s, %d turned off (enabled = false)",
			n, codexHookNoun(m), paths, codexHookIsAre(n), cfgDisplay, k)
	case n > 0:
		head = fmt.Sprintf("%d of %s in %s %s not approved in %s", n, codexHookNoun(m), paths, codexHookIsAre(n), cfgDisplay)
	default:
		head = fmt.Sprintf("%d of %s in %s %s turned off (enabled = false) in %s", k, codexHookNoun(m), paths, codexHookIsAre(k), cfgDisplay)
	}
	var names []string
	for _, item := range append(append([]codexHookTrustItem{}, report.NotApproved...), report.Off...) {
		names = append(names, report.ItemName(item))
	}
	if len(names) > codexHookTrustMaxTails {
		names = append(names[:codexHookTrustMaxTails], "…")
	}
	var steps []string
	if n > 0 {
		steps = append(steps, codexHookTrustApprovalStep(report.ApprovalDirs()))
	}
	if k > 0 {
		steps = append(steps, fmt.Sprintf("remove enabled = false from the [hooks.state] entries of the hooks turned off in %s", cfgDisplay))
	}
	return fmt.Sprintf("%s (%s) — %s; %s", head, strings.Join(names, ", "), codexHookSkipClause(n+k), strings.Join(steps, ", and "))
}

// codexHookTrustApprovalStep is how to approve the hooks: open codex where
// it reads them and answer its Hooks need review dialog. dirs is
// codexHookTrustReport.ApprovalDirs, one directory per spelling of the
// deepest layer holding a hook not approved; with two spellings (plan D1,
// AC9), codex keys an approval by the path it was opened under, so both need
// one. dirs is never empty here: some hook is not approved.
func codexHookTrustApprovalStep(dirs []string) string {
	const dialog = "(Hooks need review → Trust all and continue)"
	if len(dirs) < 2 {
		return fmt.Sprintf("open codex in %s and trust the hooks %s", dirs[0], dialog)
	}
	return fmt.Sprintf("open codex in %s and again in %s and trust the hooks each time %s: codex keys an approval by the path it was opened under, and a seat may run under either",
		dirs[0], dirs[1], dialog)
}

// codexHookTrustPathList joins paths into one phrase with conj before the
// last one: "a", "a and b", "a, b, and c".
func codexHookTrustPathList(paths []string, conj string) string {
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return paths[0]
	case 2:
		return paths[0] + " " + conj + " " + paths[1]
	default:
		return strings.Join(paths[:len(paths)-1], ", ") + ", " + conj + " " + paths[len(paths)-1]
	}
}

// codexHookNoun is "1 hook" or "<n> hooks".
func codexHookNoun(n int) string {
	if n == 1 {
		return "1 hook"
	}
	return fmt.Sprintf("%d hooks", n)
}

// codexHookIsAre is "is" for one hook and "are" for any other count.
func codexHookIsAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

// codexHookSkipClause says what codex does with n hooks it has no approval
// for, or that are turned off.
func codexHookSkipClause(n int) string {
	if n == 1 {
		return "codex skips it, so it does not run in codex seats"
	}
	return "codex skips them, so they do not run in codex seats"
}
