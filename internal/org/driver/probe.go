package driver

import (
	"context"
	"fmt"
	"slices"
)

// ProbeModel launches a minimal, side-effect-free invocation of drv with
// model to confirm the CLI accepts that model id. A model of "" launches drv
// without --model, so the probe checks the CLI's own default model, the way
// a seat whose [org.role_models] entry has model "" is launched (plan
// 2026-11-05-org-role-models). A non-nil error (with stderr detail folded in
// by the Runner, see ExecRunner.Run) means the model id was rejected or the
// CLI itself failed to run.
//
// codex `--model` support on `codex exec` is best-effort (known upstream
// gap: some codex builds ignore or reject --model on the exec subcommand),
// so a ProbeModel failure for drv=="codex" is advisory, not fatal -- the
// caller (checkOrgModelProbes in internal/cli/doctor.go, wired into
// `ralph doctor --probe-models`) decides how to surface that severity.
func ProbeModel(ctx context.Context, r Runner, drv, model string) error {
	var modelArgs []string
	what := "default model"
	if model != "" {
		modelArgs = []string{"--model", model}
		what = fmt.Sprintf("model %q", model)
	}
	switch drv {
	case "claude":
		args := slices.Concat(modelArgs, []string{"-p", "ping", "--output-format", "text"})
		if _, err := r.Run(ctx, "claude", args...); err != nil {
			return fmt.Errorf("probe claude %s: %w", what, err)
		}
		return nil
	case "codex":
		args := slices.Concat([]string{"exec"}, modelArgs, []string{"--skip-git-repo-check", "ping"})
		if _, err := r.Run(ctx, "codex", args...); err != nil {
			return fmt.Errorf("probe codex %s: %w", what, err)
		}
		return nil
	default:
		return fmt.Errorf("probe: unknown driver %q (want claude|codex)", drv)
	}
}
