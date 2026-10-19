package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// The wake of a director in the director verbs (plan
// 2026-10-18-org-director-wake, D3, D4, D9, and AC2): attach's check of
// --wake and --wake-target, the wake attach records and its check against
// [org.director].wake_cmd, and what attach's success line and the director:
// line say of a wake with push. newOrgDirectorAttachCmd (org_director.go)
// registers the two flags and calls the first three; orgDirectorLine
// (org_director_report.go) calls orgDirectorPushDetails.

// checkOrgDirectorAttachWakeFlags is attach's check of --wake and
// --wake-target before anything is read (plan 2026-10-18-org-director-wake,
// D3 and AC2), as checkOrgDirectorDriver checks --driver. It returns the
// --wake value split on "," (nothing trimmed), or nil when --wake was not
// given, and refuses one that config.ValidateDirectorWakeNamed refuses (a
// value other than pull and push, an empty part, or one twice), naming
// --wake. A --wake-target that was given, an empty one included, must pass
// org.ValidateDirectorWakeTarget. Whether the wake can be sent needs the
// config, so attachWake checks that.
func checkOrgDirectorAttachWakeFlags(cmd *cobra.Command, wakeFlag, wakeTarget string) ([]string, error) {
	var wake []string
	if cmd.Flags().Changed("wake") {
		wake = strings.Split(wakeFlag, ",")
		if err := config.ValidateDirectorWakeNamed("--wake", wake); err != nil {
			return nil, fmt.Errorf("org: director attach: %w", err)
		}
	}
	if cmd.Flags().Changed("wake-target") {
		if err := org.ValidateDirectorWakeTarget(wakeTarget); err != nil {
			return nil, fmt.Errorf("org: director attach: --wake-target: %w", err)
		}
	}
	return wake, nil
}

// attachWake resolves the wake attach records and checks that it can be
// sent (plan 2026-10-18-org-director-wake, D4 and D9): flagWake, the --wake
// that checkOrgDirectorAttachWakeFlags returned, else [org.director].wake,
// and [org.director].wake_cmd, both read by
// withMainWorktreeDirectorWakeConfig's rule (from the main worktree's
// ralph.toml when --config is not given and the ledger is the main
// worktree's), so a feature worktree's ralph.toml chooses neither. It runs
// org.CheckDirectorAttachWake with target, the --wake-target; a refusal
// names the file the settings were read from (orgDirectorEnabledFile's
// rule) and is returned before anything is written. A main worktree
// ralph.toml that does not load is an error.
func (rt *orgDirectorRuntime) attachWake(configPath string, flagWake []string, target string) ([]string, error) {
	cfg, err := withMainWorktreeDirectorWakeConfig(rt.cfg, configPath, rt.stateDir, rt.stateDirSource)
	if err != nil {
		return nil, err
	}
	wake, read := flagWake, "[org.director].wake_cmd is read from "
	if wake == nil {
		wake, read = cfg.Director.Wake, "[org.director].wake and wake_cmd are read from "
	}
	if err := org.CheckDirectorAttachWake(wake, target, cfg.Director.WakeCmd); err != nil {
		return nil, fmt.Errorf("org: director attach: %w; %s%s", err, read, orgDirectorEnabledFile(configPath, rt.stateDir, rt.stateDirSource))
	}
	return wake, nil
}

// orgDirectorAttachWakeDetails is what attach's success line says of reg's
// wake, inside its parentheses after the driver (plan
// 2026-10-18-org-director-wake, AC2): nothing for a wake without push, so a
// pull-only line is the line of stage 8a; else ", wake=" and the wake, and
// ", wake_target=" and the recorded target when there is one. A replaced
// registration was read from director.jsonl, which a human can edit, so the
// values are escaped (inboxField).
func orgDirectorAttachWakeDetails(reg org.DirectorRegistration) string {
	if !reg.WakesByPush() {
		return ""
	}
	details := ", wake=" + inboxField(strings.Join(reg.Wake, ","))
	if reg.WakeTarget != "" {
		details += ", wake_target=" + inboxField(reg.WakeTarget)
	}
	return details
}

// orgDirectorPushDetails is the end of the details of the director: line
// for a registration whose wake has push (plan
// 2026-10-18-org-director-wake, AC2): ", wake=" and the wake, then " to "
// and the target the watch sends to (ResolvedWakeTarget: a promoted
// session's --wake-target, a herdr pane, or a launch cmd's terminal name)
// when there is one. For a wake without push it is "". The values come from
// director.jsonl, so they are escaped (inboxField).
func orgDirectorPushDetails(reg org.DirectorRegistration) string {
	if !reg.WakesByPush() {
		return ""
	}
	details := ", wake=" + inboxField(strings.Join(reg.Wake, ","))
	if target := reg.ResolvedWakeTarget(); target != "" {
		details += " to " + inboxField(target)
	}
	return details
}
