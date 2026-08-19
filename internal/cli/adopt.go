package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

// adoptOptions collects `ralph adopt`'s inputs, independent of cobra flag
// parsing, so runAdoptIO stays directly testable.
type adoptOptions struct {
	// Path is the single target to adopt. Ignored when All is set.
	Path string
	// All adopts every fork and unresolved-drift path in one run.
	All bool
	// Yes skips the y/N confirmation prompt.
	Yes bool
}

// newAdoptCmd wires `ralph adopt <path>|--all` (spec FR-3): resets a fork or
// unresolved-drift owner=core path back to owner=core, discarding its
// current disk content in favor of the active template. This is the
// destructive counterpart to `ralph eject` (FR-4's other resolution route
// for unresolved drift, and the only way back from a fork).
func newAdoptCmd() *cobra.Command {
	var all, yes bool

	cmd := &cobra.Command{
		Use:   "adopt <path>|--all",
		Short: "Reset a fork or drifted core path back to owner=core using the current template",
		Long: `Overwrites <path>'s disk content with the current embedded template and
records owner=core in .ralph/manifest.toml, discarding whatever content was
there before (a fork's user edits, or an unresolved-drift core path's
divergence). --all adopts every fork and every unresolved-drift path in one
run.

This is destructive: it discards content that git is the only way to
recover. adopt therefore requires a clean git work tree before it writes
anything (the same precondition ` + "`ralph upgrade`" + `'s legacy migration uses), shows
the full target list, and asks for y/N confirmation (skip with --yes). Every
target is preflight-checked (path containment, no symlinked parents, a
regular-file-or-absent leaf) before any target is written — one failing
target aborts the whole batch with zero writes.

A fork whose path the current templates no longer ship (retired) is
rejected: adopt has nothing to restore it to.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return fmt.Errorf("adopt --all takes no <path> argument")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("adopt requires exactly one <path> argument, or --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			return runAdoptIO(".", adoptOptions{Path: path, All: all, Yes: yes}, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "adopt every fork and unresolved-drift path")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the y/N confirmation prompt")

	return cmd
}

// adoptTarget names one path adopt will reset to owner=core, plus a
// human-readable description of the state it is being adopted from (used
// only in the confirmation listing).
type adoptTarget struct {
	Path      string
	FromState string
}

// runAdoptIO is the testable core of `ralph adopt`.
func runAdoptIO(targetDir string, opts adoptOptions, in io.Reader, out, errOut io.Writer) error {
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	manifest, manifestPath, err := requireV2ManifestForOwnership(absDir)
	if err != nil {
		return err
	}

	desired, plan, err := resolveOwnershipPlan(absDir, manifest, errOut)
	if err != nil {
		return err
	}

	var targets []adoptTarget
	if opts.All {
		var retired []string
		targets, retired = resolveAdoptAllTargets(manifest, desired, plan)
		for _, p := range retired {
			writef(out, "  skipped (retired — not adoptable): %s\n", p)
		}
		if len(targets) == 0 {
			writef(out, "Nothing to adopt: no fork or unresolved-drift paths found.\n")
			return nil
		}
	} else {
		if opts.Path == "" {
			return fmt.Errorf("adopt requires a <path> argument, or --all")
		}
		target, terr := resolveAdoptSingleTarget(manifest, desired, driftPathSet(plan), opts.Path)
		if terr != nil {
			return terr
		}
		targets = []adoptTarget{target}
	}

	// Git-clean precondition BEFORE confirmation: dirty tree aborts with
	// zero writes (git is adopt's only rollback mechanism — see plan's
	// Codex finding 2 and checkGitCleanForDestructiveOp, migrate.go).
	if err := checkGitCleanForDestructiveOp(absDir, "adopt"); err != nil {
		return err
	}

	// Preflight every target before any write: one failing target aborts
	// the whole batch with zero writes (AC-7's partial-failure guarantee).
	for _, t := range targets {
		if err := validateMigrationWriteTarget(absDir, t.Path); err != nil {
			return fmt.Errorf("adopt preflight failed for %s: %w; no files were changed", t.Path, err)
		}
	}

	writef(out, "The following path(s) will be adopted (content reset to the current template, owner -> core):\n")
	for _, t := range targets {
		writef(out, "  %s (%s -> core)\n", t.Path, t.FromState)
	}

	confirmed, cerr := confirmDestructiveOp(in, out, opts.Yes, "adopt")
	if cerr != nil {
		return fmt.Errorf("reading adopt confirmation: %w", cerr)
	}
	if !confirmed {
		writef(out, "Adopt aborted; no files were changed.\n")
		return nil
	}

	for _, t := range targets {
		content, ok := desired[t.Path]
		if !ok {
			// Unreachable: retired targets are filtered out of both
			// resolveAdoptSingleTarget and resolveAdoptAllTargets before
			// reaching this loop. Kept as a defensive guard, matching the
			// surrounding fail-closed style.
			return fmt.Errorf("%s: no current template content available; aborting mid-batch — the work tree was clean before this run (git status), so `git status`/`git diff` show exactly what adopt wrote so far; fix the underlying issue and re-run", t.Path)
		}
		if err := writeFileV2(absDir, t.Path, content); err != nil {
			return fmt.Errorf("writing %s: %w; the work tree was clean before this run (git status), so `git status`/`git diff` show exactly what adopt wrote so far — fix the underlying issue and re-run", t.Path, err)
		}
		templateHash := scaffold.HashBytes(content)
		if err := manifest.SetFileOwned(t.Path, scaffold.OwnerCore, templateHash, templateHash); err != nil {
			return fmt.Errorf("recording %s as owner=core: %w", t.Path, err)
		}
	}

	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	writef(out, "Adopted %d path(s): owner -> core, disk content reset to the current template.\n", len(targets))
	return nil
}

// resolveOwnershipPlan runs the same read-only classification runUpgradeV2
// uses (buildDesiredStateV2 + PlanCoreReplaceDesired with identical
// ReplaceOptions) so eject/adopt's notion of "fork" / "unresolved drift" /
// "retired" can never drift out of sync with what `ralph upgrade` itself
// would report. It only reads disk (via PlanCoreReplaceDesired) — zero
// writes.
func resolveOwnershipPlan(absDir string, manifest *scaffold.Manifest, errOut io.Writer) (desired map[string][]byte, plan upgrade.ReplacePlan, err error) {
	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return nil, upgrade.ReplacePlan{}, fmt.Errorf("loading templates: %w", err)
	}
	desired, preservePrefixes, _, _, err := buildDesiredStateV2(baseFS, manifest, errOut)
	if err != nil {
		return nil, upgrade.ReplacePlan{}, fmt.Errorf("building desired state: %w", err)
	}
	planOpts := upgrade.ReplaceOptions{
		SkipPaths:        v2SkipPaths(),
		PreservePrefixes: preservePrefixes,
		OwnerForPath:     ownerForScaffoldPath,
	}
	plan, err = upgrade.PlanCoreReplaceDesired(manifest, absDir, desired, planOpts)
	if err != nil {
		return nil, upgrade.ReplacePlan{}, fmt.Errorf("planning: %w", err)
	}
	return desired, plan, nil
}

// driftPathSet indexes plan.Drift by path for O(1) single-target lookups.
// plan.Drift is only ever populated for owner=core paths (classifyCore,
// internal/upgrade/replaceplan.go) or genuinely untracked paths
// (classifyUntracked) — resolveAdoptSingleTarget/resolveAdoptAllTargets
// additionally require a tracked owner=core manifest entry before treating
// a drift hit as adoptable, so an untracked drifted path is never adopted
// (AC-3: untracked paths are rejected).
func driftPathSet(plan upgrade.ReplacePlan) map[string]bool {
	set := make(map[string]bool, len(plan.Drift))
	for _, d := range plan.Drift {
		set[d.Path] = true
	}
	return set
}

// resolveAdoptSingleTarget normalizes rawPath and classifies it against the
// v2 ownership rules adopt understands: owner=fork, or owner=core with an
// unresolved-drift hit in plan.Drift. Everything else (untracked, v2
// exception face, owner=seed/block, already-converged owner=core, or a
// legacy/unattributed entry) is rejected with zero writes.
func resolveAdoptSingleTarget(manifest *scaffold.Manifest, desired map[string][]byte, drift map[string]bool, rawPath string) (adoptTarget, error) {
	path, err := cleanMigrationPath(rawPath)
	if err != nil {
		return adoptTarget{}, fmt.Errorf("%s: %w", rawPath, err)
	}
	if v2SkipPaths()[path] {
		return adoptTarget{}, v2ExceptionFaceError(path)
	}

	entry, tracked := manifest.Files[path]
	if !tracked {
		return adoptTarget{}, fmt.Errorf("%s: not tracked in the manifest; nothing to adopt", path)
	}

	var fromState string
	switch {
	case entry.Owner == scaffold.OwnerFork:
		fromState = "fork"
	case entry.Owner == scaffold.OwnerCore && drift[path]:
		fromState = "drifted core"
	case entry.Owner == scaffold.OwnerSeed:
		return adoptTarget{}, fmt.Errorf("%s: owner=seed is user-owned; adopt only applies to owner=fork paths or unresolved drift on owner=core paths", path)
	case entry.Owner == scaffold.OwnerBlock:
		return adoptTarget{}, fmt.Errorf("%s: owner=block is managed by the block engine, not adopt; customize content outside the ralph-managed block markers, or use .ralph/local/ for downstream extensions", path)
	case entry.Owner == scaffold.OwnerCore:
		return adoptTarget{}, fmt.Errorf("%s: already owner=core with no unresolved drift; nothing to adopt", path)
	default:
		return adoptTarget{}, fmt.Errorf("%s: no recorded ownership (legacy manifest entry); adopt only applies to owner=fork paths or unresolved drift on owner=core paths — run `ralph upgrade` to migrate a legacy manifest first", path)
	}

	if _, ok := desired[path]; !ok {
		return adoptTarget{}, fmt.Errorf("%s: retired — the current templates no longer ship this path; adopt has nothing to restore it to (existing content is left as-is; delete it manually if it is no longer needed)", path)
	}
	return adoptTarget{Path: path, FromState: fromState}, nil
}

// resolveAdoptAllTargets enumerates every fork (plan.Advisories with
// Owner=fork) and every unresolved-drift owner=core path (plan.Drift,
// filtered to manifest-tracked owner=core entries — see driftPathSet's doc)
// as --all's adopt targets, sorted by path for deterministic output. A
// candidate whose path the current templates no longer ship is moved to
// retired instead of targets, matching the single-path rejection.
func resolveAdoptAllTargets(manifest *scaffold.Manifest, desired map[string][]byte, plan upgrade.ReplacePlan) (targets []adoptTarget, retired []string) {
	var candidates []adoptTarget

	for _, adv := range plan.Advisories {
		if adv.Owner != scaffold.OwnerFork {
			continue
		}
		candidates = append(candidates, adoptTarget{Path: adv.Path, FromState: "fork"})
	}

	for _, d := range plan.Drift {
		entry, tracked := manifest.Files[d.Path]
		if !tracked || entry.Owner != scaffold.OwnerCore {
			// Outside eject/adopt's target boundary (AC-3): only
			// manifest-tracked owner=core paths are adoptable via drift.
			continue
		}
		candidates = append(candidates, adoptTarget{Path: d.Path, FromState: "drifted core"})
	}

	for _, c := range candidates {
		if _, ok := desired[c.Path]; !ok {
			retired = append(retired, c.Path)
			continue
		}
		targets = append(targets, c)
	}

	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	sort.Strings(retired)
	return targets, retired
}
