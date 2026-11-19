package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The doctor check "Merge guarantee" under a signal (plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md, D2 and AC5b, and
// ACTION_REQUIRED #1 of docs/reports/cross-review-triage-org-ssh-alias-host.md):
// ssh -G runs in a process group of its own, which a terminal's Ctrl-C does
// not reach, so only the end of the check's context stops it. A SIGINT,
// SIGTERM, or SIGHUP while the check runs ends that context, and doctor then
// runs no later check and returns an error.

// doctorMGSSHExec is a fake org.MergeOrderExec for acme/app on github.com
// with an ssh origin: `git remote get-url origin` answers
// git@github.com:acme/app.git, `ssh -G github.com` answers
// "hostname github.com" or, with block set, calls interrupt and then waits
// until its ctx is done, as an ssh -G waiting on a `Match exec` command does
// until its process group is killed, and gh is e's.
type doctorMGSSHExec struct {
	e         *doctorMGExec
	block     bool
	interrupt func()

	mu       sync.Mutex
	names    []string
	sshCause error // the cause of the ssh's ctx end, once a blocked ssh returns
}

func (s *doctorMGSSHExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	s.mu.Lock()
	s.names = append(s.names, name)
	s.mu.Unlock()
	switch {
	case name == "git" && slices.Equal(args, []string{"remote", "get-url", "origin"}):
		return "git@github.com:acme/app.git\n", nil
	case name == "ssh" && slices.Equal(args, []string{"-G", "github.com"}) && !s.block:
		return "user git\nhostname github.com\nport 22\n", nil
	case name == "ssh" && slices.Equal(args, []string{"-G", "github.com"}):
		s.interrupt()
		<-ctx.Done()
		s.mu.Lock()
		s.sshCause = context.Cause(ctx)
		s.mu.Unlock()
		return "", errors.New("signal: killed")
	}
	return s.e.run(ctx, dir, env, name, args...)
}

// ran returns the commands run, in order, and the cause a blocked ssh saw.
func (s *doctorMGSSHExec) ran() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.names), s.sshCause
}

// setDoctorMGSignalContext makes doctorMergeGuaranteeSignalContext f until
// the test ends.
func setDoctorMGSignalContext(t *testing.T, f func(context.Context) (context.Context, context.CancelFunc)) {
	t.Helper()
	old := doctorMergeGuaranteeSignalContext
	t.Cleanup(func() { doctorMergeGuaranteeSignalContext = old })
	doctorMergeGuaranteeSignalContext = f
}

// doctorCheckLines returns the check lines of doctor's output.
func doctorCheckLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  ") {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor: a signal while
// ssh -G waits ends the context ssh -G runs under, with the signal's cause,
// so the check returns at once, with no gh call. The check's line is unknown
// and says the check was interrupted, and it is cached as any unknown is (or
// not cached, when the ledger cannot be resolved); doctor runs no later
// check (no Node toolchain line), prints the lines before as it does without
// the signal, prints the Interrupted line, and returns an error that says
// the Merge guarantee check was interrupted, with the count of failed checks
// when one failed. Through the seam, the context ends; with SIGINT to the
// process, the default context (signal.NotifyContext) does. Each context the
// check takes is released by the time doctor returns. Without the signal,
// doctor runs every check.
func TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor(t *testing.T) {
	cases := []struct {
		name      string
		cause     string
		ledgerErr bool
		// setup sets the seam (or leaves the default) and returns the
		// interrupt.
		setup func(t *testing.T) (interrupt func())
	}{
		{"the context ends", "test stopped", false, setupDoctorMGStoppableContext},
		{"the context ends after a failed check", "test stopped", true, setupDoctorMGStoppableContext},
		{"SIGINT to the process", "interrupt signal received", false, func(t *testing.T) func() {
			// A channel of the test's own is registered first, so a SIGINT
			// the check does not catch cannot end the test binary.
			caught := make(chan os.Signal, 1)
			signal.Notify(caught, syscall.SIGINT)
			t.Cleanup(func() { signal.Stop(caught) })
			return func() {
				if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
					t.Errorf("send SIGINT to the test process: %v", err)
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, _ := doctorMGProject(t)
			interrupt := tc.setup(t)
			var taken, released int
			inner := doctorMergeGuaranteeSignalContext
			setDoctorMGSignalContext(t, func(parent context.Context) (context.Context, context.CancelFunc) {
				taken++
				ctx, stop := inner(parent)
				return ctx, func() { released++; stop() }
			})
			stateDir := t.TempDir()
			s := &doctorMGSSHExec{e: newDoctorMGExec(t, org.MergeGuaranteeGitHubProtection), interrupt: interrupt}
			env := doctorMGEnv(s.e, true, stateDir)
			useDoctorMGEnv(t, func(targetDir, _ string, _ bool) (org.MergeGuaranteeParams, string, error) {
				params, ledger, _ := env(targetDir)
				params.Exec = s.run
				if tc.ledgerErr {
					return params, "", errors.New("ralph.local.toml: [org].state_dir is not absolute")
				}
				return params, ledger, nil
			})

			var calmOut string
			calmErr := captureStdoutText(t, &calmOut, func() error { return runDoctorFull(dir, false, false) })
			wantCalmErr := ""
			if tc.ledgerErr {
				wantCalmErr = "doctor: 1 check(s) failed"
			}
			if (calmErr == nil) != (wantCalmErr == "") || (calmErr != nil && calmErr.Error() != wantCalmErr) {
				t.Fatalf("without a signal, doctor returned %v, want %q\noutput:\n%s", calmErr, wantCalmErr, calmOut)
			}
			for _, want := range []string{"  ✓ Merge guarantee: pass — github_protection: ", " Node toolchain: "} {
				if !strings.Contains(calmOut, want) || strings.Contains(calmOut, "\nInterrupted: ") {
					t.Fatalf("without a signal, doctor output lacks %q or says it was interrupted:\n%s", want, calmOut)
				}
			}
			if calmRan, _ := s.ran(); !slices.Equal(calmRan, []string{"git", "ssh", "gh", "gh"}) {
				t.Fatalf("without a signal, the commands run = %q, want git, ssh, and two gh", calmRan)
			}

			s.mu.Lock()
			s.names, s.block = nil, true
			s.mu.Unlock()
			var out string
			err := captureStdoutText(t, &out, func() error { return runDoctorFull(dir, false, false) })

			// The signal's cause, not ssh -G's 5 s bound, ended ssh -G's ctx.
			ran, sshCause := s.ran()
			if !slices.Equal(ran, []string{"git", "ssh"}) {
				t.Errorf("commands run = %q, want git and ssh only: the check went on after the signal", ran)
			}
			if sshCause == nil || sshCause.Error() != tc.cause {
				t.Errorf("ssh -G's ctx ended with %v, want the signal's cause %q", sshCause, tc.cause)
			}
			wantErr := "doctor: interrupted (" + tc.cause + ") while the Merge guarantee check read origin's repository"
			if tc.ledgerErr {
				wantErr += "; 1 check(s) failed"
			}
			if err == nil || err.Error() != wantErr {
				t.Errorf("doctor returned %v, want %q", err, wantErr)
			}
			saved := "; cached in " + org.MergeGuaranteePathIn(stateDir)
			if tc.ledgerErr {
				saved = "; not cached: the org ledger could not be resolved"
			}
			mgLine := "  ℹ Merge guarantee: info — unknown: the check was interrupted (" + tc.cause + ") while ssh -G github.com " +
				"ran, so the protection of origin's repository's default branch is unknown" + saved
			lines, calmLines := doctorCheckLines(out), doctorCheckLines(calmOut)
			mgAt := slices.IndexFunc(calmLines, func(l string) bool { return strings.Contains(l, " Merge guarantee: ") })
			if want := append(slices.Clone(calmLines[:max(mgAt, 0)]), mgLine); mgAt < 0 || !slices.Equal(lines, want) {
				t.Errorf("check lines after the signal are\n%s\nwant the lines before the Merge guarantee as without one, "+
					"then its interrupted line and no later check:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
			}
			if want := "\nInterrupted: the Merge guarantee check did not finish. Run ralph doctor again.\n"; !strings.Contains(out, want) {
				t.Errorf("doctor output lacks %q:\n%s", want, out)
			}
			if strings.Contains(out, "All checks passed.") || strings.Contains(out, "Some checks failed.") != tc.ledgerErr {
				t.Errorf("doctor's closing lines say all checks passed, or do not say some failed when one did (%v):\n%s",
					tc.ledgerErr, out)
			}
			if !tc.ledgerErr {
				if c := readDoctorMGCache(t, stateDir); c.Kind != org.MergeGuaranteeUnknown || !strings.Contains(c.Detail, "the check was interrupted (") {
					t.Errorf("cache = %+v, want the interrupted unknown", c)
				}
			}
			if taken != 2 || released != 2 {
				t.Errorf("the check took %d signal contexts and released %d, want 2 and 2 (one per doctor run)", taken, released)
			}
		})
	}
}

// setupDoctorMGStoppableContext sets doctorMergeGuaranteeSignalContext to
// one that ends with the cause "test stopped" when the returned interrupt is
// called.
func setupDoctorMGStoppableContext(t *testing.T) func() {
	stopped := make(chan struct{})
	setDoctorMGSignalContext(t, func(parent context.Context) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancelCause(parent)
		go func() {
			select {
			case <-stopped:
				cancel(errors.New("test stopped"))
			case <-ctx.Done():
			}
		}()
		return ctx, func() { cancel(nil) }
	})
	return func() { close(stopped) }
}

// TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP: the
// default context ends at a SIGINT, a SIGTERM, or a SIGHUP to the process,
// with the signal as its cause, and not before.
func TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			// A channel of the test's own is registered first, so a signal
			// the context does not catch cannot end the test binary.
			caught := make(chan os.Signal, 1)
			signal.Notify(caught, sig)
			defer signal.Stop(caught)
			ctx, stop := doctorMergeGuaranteeSignalContext(context.Background())
			defer stop()
			if ctx.Err() != nil {
				t.Fatalf("the context ended before the signal: %v", context.Cause(ctx))
			}
			if err := syscall.Kill(os.Getpid(), sig); err != nil {
				t.Fatalf("send %s to the test process: %v", sig, err)
			}
			select {
			case <-ctx.Done():
			case <-time.After(10 * time.Second):
				t.Fatalf("the context did not end within 10s of %s", sig)
			}
			if got, want := context.Cause(ctx), sig.String()+" signal received"; got == nil || got.Error() != want {
				t.Errorf("cause = %v, want %q", got, want)
			}
		})
	}
}
