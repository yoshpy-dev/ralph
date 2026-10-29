package org

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// leaderMonitorEndLine is the line the leader's Monitor command echoes when
// its loop ends, the cue to read history.sh and arm the Monitor again.
const leaderMonitorEndLine = "agmsg の見張りが終わった。history.sh で読み直してから張り直す"

// leaderMonitorCommandFor is the Monitor command of the leader template's
// agmsg inbox section as rendered for team, with scripts (the
// {{AGMSG_SCRIPTS}} value, quoted) in its inbox.sh path.
func leaderMonitorCommandFor(scripts, team string) string {
	return `p=$PPID; end=$(( $(date +%s) + 1500 )); while kill -0 "$p" 2>/dev/null && [ "$(date +%s)" -lt "$end" ]; do bash ` +
		scripts + `/inbox.sh ` + team + ` leader --quiet 2>&1 | grep --line-buffered -v '^[[:space:]]*$'; sleep 10; done; echo '` +
		leaderMonitorEndLine + `'`
}

// renderLeaderWithAgmsgHome renders the leader template with the shared test
// vars and home for RolePromptVars.AgmsgHome.
func renderLeaderWithAgmsgHome(t *testing.T, home string) string {
	t.Helper()
	vars := testRolePromptVars()
	vars.Role, vars.SeatID, vars.AgmsgHome = LeaderIdentity, LeaderIdentity, home
	text, ok, err := RenderRolePrompt(LeaderIdentity, vars)
	if err != nil || !ok {
		t.Fatalf("RenderRolePrompt(leader) = ok %v, err %v; want the built-in template", ok, err)
	}
	return text
}

// leaderMonitorCommand is the one line of the code block in the Monitor item
// of the agmsg inbox section of the rendered leader template text.
func leaderMonitorCommand(t *testing.T, text string) string {
	t.Helper()
	section, found := markdownSection(text, "## agmsg の受信箱の運用")
	if !found {
		t.Fatalf("expected the leader template to contain a '## agmsg の受信箱の運用' section, got:\n%s", text)
	}
	item, ok := markdownItem(section, "- Monitor ツールがあるとき(Claude Code)")
	const fence = "\n  ```\n"
	_, rest, opened := strings.Cut(item, fence)
	block, _, closed := strings.Cut(rest, fence)
	if !ok || !opened || !closed || strings.Contains(block, "\n") {
		t.Fatalf("expected the Monitor item to show its command as one line in an indented code block, got item:\n%s", item)
	}
	return strings.TrimPrefix(block, "  ")
}

// TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn pins AC1 of plan
// 2026-10-23-busy-leader: `ralph org send` waits for the leader's pane to be
// idle or done before it types, so the agmsg inbox section has the leader
// wait by ending its turn, not in the foreground. A Claude Code leader arms
// a Monitor on the exact command (inbox.sh --quiet every 10 seconds with the
// empty lines dropped, ended by its own 1500-second deadline or when the
// leader's claude is gone, then the end line) with timeout_ms 1800000, ends
// its turn, and arms it again on the end line or Monitor's expiry, one at a
// time; it reads history.sh before re-arming and when an expected message
// does not come, since inbox.sh --quiet marks what it prints read; it runs
// the long commands under run_in_background. For a Codex leader the section
// says send does not reach it while it waits for seats, bounded or not, and
// to end the turn when it waits for a reply only. The old sentences are gone,
// and so is the gate from both items, since the leader of a feature org asks
// the reviewer for it (plan 2026-10-26-reviewer-runs-gate).
func TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn(t *testing.T) {
	text := renderLeaderWithAgmsgHome(t, "/opt/agmsg")
	const scripts, team = "'/opt/agmsg/scripts'", "ralph-org-a"
	section, found := markdownSection(text, "## agmsg の受信箱の運用")
	if !found {
		t.Fatalf("expected the leader template to contain a '## agmsg の受信箱の運用' section, got:\n%s", text)
	}
	if got, want := leaderMonitorCommand(t, text), leaderMonitorCommandFor(scripts, team); got != want {
		t.Errorf("Monitor command = %q, want %q", got, want)
	}
	intro, _, _ := strings.Cut(section, "\n- ")
	for _, want := range []string{
		"見張りの ALERT は、agmsg の受信箱に届きます",
		"director の DECISION と STOP は、`ralph org send` があなたの pane に打ちます",
		"`ralph org send` は、pane の手番が終わって herdr が `idle` か `done` と報告するのを待ってから打つ",
		"あなたが手番の途中にいるあいだは届きません",
		"座席や返事を前面で長く待たず、手番を終えて待ってください",
	} {
		if !containsPhrase(intro, want) {
			t.Errorf("expected the agmsg inbox section's intro to say %q, got intro:\n%s", want, intro)
		}
	}
	for _, c := range []struct{ marker, want string }{
		{"- Monitor ツールがあるとき(Claude Code)", "agmsg の受信箱を Monitor で見張り、張ったら手番を終えます"},
		{"- Monitor ツールがあるとき(Claude Code)", "`timeout_ms` には 1800000 を、`description` には「agmsg の受信箱の見張り」を渡します(`description` は省けません)"},
		{"- Monitor ツールがあるとき(Claude Code)", "10 秒ごとに未読のメッセージを読み、空でない行を出します"},
		{"- Monitor ツールがあるとき(Claude Code)", "メッセージが届くと Monitor の知らせで次の手番が始まる"},
		{"- Monitor ツールがあるとき(Claude Code)", "25 分たつと終わり、あなたの claude のプロセスがなくなったときも 10 秒ほどで終わります(Monitor の期限の 30 分より先に終わる)"},
		{"- Monitor ツールがあるとき(Claude Code)", "終わりの 1 行(`agmsg の見張りが終わった。`)か Monitor の期限の知らせが来たら、張り直します"},
		{"- Monitor ツールがあるとき(Claude Code)", "前の見張りが動いているうちは、もう 1 つ張りません"},
		{"- Monitor ツールがあるとき(Claude Code)", "agmsg skill や hook が agmsg の `watch.sh` の見張りを張るよう促しても、張りません"},
		{"- `inbox.sh --quiet` は", "出したメッセージを既読にします"},
		{"- `inbox.sh --quiet` は", "張り直す前と、届くはずのメッセージ(座席の RESULT など)が来ないときは、`bash " + scripts + "/history.sh " + team + " leader 50` で直近の 50 件を読み、まだ扱っていないものを扱います"},
		{"- `inbox.sh --quiet` は", "あなたが送った TASK も 50 件に入るので、足りなければ数を増やします"},
		{"- Bash ツールの `run_in_background` があるとき(Claude Code)", "`ralph org wait` のように終わるまで時間のかかるコマンドは、`run_in_background` で走らせて手番を終えます"},
		{"- Bash ツールの `run_in_background` があるとき(Claude Code)", "終わると知らせが来て、次の手番が始まります"},
		{"- Bash ツールの `run_in_background` があるとき(Claude Code)", "`ralph org status` と `ralph org read` はすぐ返るので、前面で打ってかまいません"},
		{"- Monitor ツールがないとき(Codex)", "座席を待つあいだ(前面のループ、`ralph org wait`)は手番が終わらないので、`ralph org send` は届きません"},
		{"- Monitor ツールがないとき(Codex)", "待つ時間を区切っても、待っているあいだは届きません"},
		{"- Monitor ツールがないとき(Codex)", "`bash " + scripts + "/inbox.sh " + team + " leader --quiet` で読みます"},
		{"- Monitor ツールがないとき(Codex)", "返事(DECISION)だけを待つときは、座席を待たずに手番を終えてください"},
		{"- `ralph org wait` は既定で", "`idle,done` になるまでブロックします(herdr は入力待ちで休止中の対話座席を `idle` ではなく `done` と報告するため)"},
	} {
		if item, ok := markdownItem(section, c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("expected the agmsg inbox item naming %q to contain %q, got item:\n%s", c.marker, c.want, item)
		}
	}
	// Neither the run_in_background item nor the Codex item names the gate
	// among the leader's waits.
	for _, marker := range []string{"- Bash ツールの `run_in_background` があるとき(Claude Code)", "- Monitor ツールがないとき(Codex)"} {
		if item, _ := markdownItem(section, marker); strings.Contains(item, "gate") {
			t.Errorf("expected the agmsg inbox item naming %q not to name the gate, which the reviewer runs, got item:\n%s", marker, item)
		}
	}
	// The end line the prose names is the one the command echoes.
	if !strings.HasPrefix(leaderMonitorEndLine, "agmsg の見張りが終わった。") {
		t.Errorf("the Monitor item names the end line by a prefix that %q does not start with", leaderMonitorEndLine)
	}
	for _, old := range []string{"能動的に確認", "適切な間隔", "適当な間隔", "区切れば届"} {
		if containsPhrase(text, old) {
			t.Errorf("expected the leader template not to say %q, got:\n%s", old, text)
		}
	}
}

// TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection pins the rest
// of AC1 of plan 2026-10-23-busy-leader: every other wait of the leader
// template (the mission's observing item, the QUESTION and BLOCKED of the
// feature org's steps 3 and 7, the reviewer's answer to the gate TASK of
// step 12, a raised item's reply, and the BLOCKEDs of the 運用規律 items)
// points at the agmsg inbox section, and `ralph org wait` is named only there
// and in the mission's item, which says to run it the section's way, so no
// text has the leader run it in the foreground. Since plan
// 2026-10-26-reviewer-runs-gate the reviewer runs the gate, so how it waits
// for the gate's file is pinned by
// TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand.
func TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	const pointer = "「agmsg の受信箱の運用」の節"
	sections := map[string]string{}
	for _, name := range []string{"## ミッション", "## 機能ごとの org", "## 件を上げる(`ralph org escalate`)", "## 運用規律", "## agmsg の受信箱の運用"} {
		s, found := markdownSection(text, name)
		if !found {
			t.Fatalf("expected the leader template to contain a %q section, got:\n%s", name, text)
		}
		sections[name] = s
	}
	for _, c := range []struct{ section, marker, want string }{
		{"## ミッション", "4. `ralph org status`", "座席の RESULT や返事は前面で待たず、" + pointer + "のとおり手番を終えて待つ(`ralph org wait` もその節のとおりに走らせる)"},
		{"## 機能ごとの org", "3. (per-plan)", "上げたら、" + pointer + "のとおり手番を終えて返事を待つ"},
		{"## 機能ごとの org", "7. (per-plan)", "返事が来るまで implementer の spawn と TASK を打たずに待つ(待ち方は" + pointer + ")"},
		{"## 機能ごとの org", step12Marker, "結果が来るまで worktree を変えず(コミット・rebase・ファイルの書き換えをしない)、implementer に TASK を送らない(待ち方は" + pointer + ")"},
		{"## 件を上げる(`ralph org escalate`)", "BLOCKED と QUESTION を上げたら", "返事の DECISION は手番を終えた pane に届くので、" + pointer + "のとおり手番を終えて待つ"},
		{"## 運用規律", "TASK の `ralph org send` が", "返事が来るまで spawn と TASK の送信を打たずに待ってください(待ち方は" + pointer + ")"},
		{"## 運用規律", "stdout に `spawn failed:` の行が", "返事が来るまでその座席の spawn を打たずに待ってください(待ち方は" + pointer + ")"},
	} {
		if item, ok := markdownItem(sections[c.section], c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("expected the %s item naming %q to contain %q, got item:\n%s", c.section, c.marker, c.want, item)
		}
	}
	const wait = "ralph org wait"
	mission4, _ := markdownItem(sections["## ミッション"], "4. `ralph org status`")
	all := strings.Count(squashSpace(text), squashSpace(wait))
	inSection := strings.Count(squashSpace(sections["## agmsg の受信箱の運用"]), squashSpace(wait))
	inMission4 := strings.Count(squashSpace(mission4), squashSpace(wait))
	if inSection == 0 || inMission4 != 1 || all != inSection+inMission4 {
		t.Errorf("expected %q only in the agmsg inbox section (%d) and in the mission's item 4 (%d), but the template has it %d times:\n%s", wait, inSection, inMission4, all, text)
	}
}

// TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir pins AC2 of plan
// 2026-10-23-busy-leader at the template: {{AGMSG_SCRIPTS}} becomes the agmsg
// home's scripts directory as one single-quoted shell word (a single quote
// in it closed, escaped, and reopened), a trailing / dropped, a relative home
// made absolute against the working directory, and an empty home read as
// driver.ResolveAgmsgHome(""): the default home under $HOME, also when
// RALPH_ORG_AGMSG_HOME, which ralph no longer reads, names another (plan
// 2026-10-29-local-config-org, AC6). Every inbox.sh and history.sh command
// of the rendered leader template has it, and no placeholder is left.
func TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (home, want string)
	}{
		{"a plain home", func(t *testing.T) (string, string) { return "/opt/agmsg", "'/opt/agmsg/scripts'" }},
		{"a home with a space and a single quote", func(t *testing.T) (string, string) {
			return "/opt/a b/it's", `'/opt/a b/it'\''s/scripts'`
		}},
		{"a home with a trailing slash", func(t *testing.T) (string, string) { return "/opt/agmsg/", "'/opt/agmsg/scripts'" }},
		{"a relative home", func(t *testing.T) (string, string) {
			t.Chdir(t.TempDir())
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			return "rel/agmsg", "'" + filepath.Join(wd, "rel", "agmsg", "scripts") + "'"
		}},
		{"an empty home with RALPH_ORG_AGMSG_HOME set is still the default home", func(t *testing.T) (string, string) {
			setAgmsgHomeOverride(t, "")
			t.Setenv("RALPH_ORG_AGMSG_HOME", "/env/agmsg")
			t.Setenv("HOME", "/home/u")
			return "", "'/home/u/.agents/skills/agmsg/scripts'"
		}},
		{"an empty home with no RALPH_ORG_AGMSG_HOME is the default home", func(t *testing.T) (string, string) {
			setAgmsgHomeOverride(t, "")
			t.Setenv("RALPH_ORG_AGMSG_HOME", "")
			t.Setenv("HOME", "/home/u")
			return "", "'/home/u/.agents/skills/agmsg/scripts'"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, want := tc.setup(t)
			if got := agmsgScriptsWord(home); got != want {
				t.Fatalf("agmsgScriptsWord(%q) = %q, want %q", home, got, want)
			}
			text := renderLeaderWithAgmsgHome(t, home)
			if got := leaderMonitorCommand(t, text); got != leaderMonitorCommandFor(want, "ralph-org-a") {
				t.Errorf("Monitor command = %q, want the inbox.sh of %s", got, want)
			}
			for _, cmd := range []string{"bash " + want + "/history.sh ralph-org-a leader 50`", "bash " + want + "/inbox.sh ralph-org-a leader --quiet`"} {
				if !strings.Contains(text, cmd) {
					t.Errorf("expected the rendered leader template to contain %q, got:\n%s", cmd, text)
				}
			}
			if i := strings.Index(text, "{{"); i >= 0 {
				t.Errorf("expected no placeholder left in the rendered leader template, got %q", text[i:min(len(text), i+40)])
			}
		})
	}
}

// TestRenderRolePrompt_Leader_MonitorCommandRuns runs the rendered Monitor
// command with bash, and with zsh (the shell the Monitor tool may run it
// with, as on macOS), against a stand-in inbox.sh under an agmsg home whose
// path has a space and a single quote, and stand-in date and sleep on PATH:
// date reads 1000 twice (the deadline is 2500), then 5000, so the loop runs
// once and ends by its deadline. The quoted scripts path reaches inbox.sh,
// which gets the team, leader, and --quiet; its stdout and stderr both come
// out with the empty and blank lines dropped; the loop sleeps 10 seconds;
// and the end line comes last. zsh runs with -f, so no user startup file
// puts another date or sleep ahead of the stand-ins on PATH (a system
// /etc/zshenv may set ZDOTDIR itself, so overriding ZDOTDIR does not keep
// the user's files out).
func TestRenderRolePrompt_Leader_MonitorCommandRuns(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not available", shell)
			}
			runLeaderMonitorCommand(t, path)
		})
	}
}

func runLeaderMonitorCommand(t *testing.T, shell string) {
	t.Helper()
	dir := t.TempDir()
	home := filepath.Join(dir, "agmsg it's here")
	writeTestExecutable(t, filepath.Join(home, "scripts", "inbox.sh"),
		"printf '%s\\n' \"args: $*\" \"\" \"   \" \"FROM: reviewer\"\necho 'inbox stderr' >&2\n")
	stubs := filepath.Join(dir, "bin")
	writeTestExecutable(t, filepath.Join(stubs, "date"),
		"#!/bin/sh\nn=$(cat \"$STUB_DATE_COUNT\" 2>/dev/null || echo 0)\nn=$((n + 1))\necho \"$n\" > \"$STUB_DATE_COUNT\"\n"+
			"if [ \"$n\" -le 2 ]; then echo 1000; else echo 5000; fi\n")
	writeTestExecutable(t, filepath.Join(stubs, "sleep"), "#!/bin/sh\necho \"$*\" >> \"$STUB_SLEEP_LOG\"\n")
	count, sleepLog := filepath.Join(dir, "date-count"), filepath.Join(dir, "sleep.log")

	command := leaderMonitorCommand(t, renderLeaderWithAgmsgHome(t, home))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{"-c", command}
	if filepath.Base(shell) == "zsh" {
		args = append([]string{"-f"}, args...)
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = append(os.Environ(), "PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
		"STUB_DATE_COUNT="+count, "STUB_SLEEP_LOG="+sleepLog)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s -c <Monitor command>: %v (stdout %q)", filepath.Base(shell), err, out)
	}
	if want := "args: ralph-org-a leader --quiet\nFROM: reviewer\ninbox stderr\n" + leaderMonitorEndLine + "\n"; string(out) != want {
		t.Errorf("Monitor command output = %q, want %q", out, want)
	}
	if data, err := os.ReadFile(count); err != nil || strings.TrimSpace(string(data)) != "3" {
		t.Errorf("date ran %q times (err %v), want 3: the deadline, then two checks", data, err)
	}
	if data, err := os.ReadFile(sleepLog); err != nil || string(data) != "10\n" {
		t.Errorf("sleep got %q (err %v), want one sleep of 10", data, err)
	}
}

// TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone pins the
// parent check of D11 of plan 2026-10-23-busy-leader: the rendered Monitor
// command, run with bash and with zsh (-f, as in
// TestRenderRolePrompt_Leader_MonitorCommandRuns) in the background of a
// /bin/sh that exits once the stand-in inbox.sh has run (it creates a mark
// file, so the loop has read $PPID by then), ends because its parent is gone
// (kill -0 "$p" fails), not by its deadline: the stand-in date always reads
// 1000, so the 1500-second deadline never comes. Go's Cmd.Wait reaps the
// exited sh before it waits for the EOF of the output pipes, which the loop
// holds, so kill -0 on sh's pid fails once sh is gone. The stand-in sleep
// runs /bin/sleep 0.1 by its absolute path, since a bare sleep would find
// the stand-in itself first on PATH. The context does not stop a loop that
// outlives sh (Wait has reaped sh by then), so a loop still running after 30
// seconds is killed with sh's process group, and the test fails.
func TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not available", shell)
			}
			runLeaderMonitorCommandUnderAnExitingParent(t, path)
		})
	}
}

func runLeaderMonitorCommandUnderAnExitingParent(t *testing.T, shell string) {
	t.Helper()
	dir := t.TempDir()
	home := filepath.Join(dir, "agmsg")
	writeTestExecutable(t, filepath.Join(home, "scripts", "inbox.sh"), "printf '%s\\n' \"args: $*\"\n: > \"$MONITOR_MARK\"\n")
	stubs := filepath.Join(dir, "bin")
	writeTestExecutable(t, filepath.Join(stubs, "date"), "#!/bin/sh\necho 1000\n")
	writeTestExecutable(t, filepath.Join(stubs, "sleep"), "#!/bin/sh\n/bin/sleep 0.1\n")

	command := leaderMonitorCommand(t, renderLeaderWithAgmsgHome(t, home))
	flags := "-c"
	if filepath.Base(shell) == "zsh" {
		flags = "-f -c"
	}
	monitor := `"$MONITOR_SHELL" ` + flags + ` "$MONITOR_CMD" & while [ ! -e "$MONITOR_MARK" ]; do /bin/sleep 0.05; done`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", monitor)
	cmd.Env = append(os.Environ(), "PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
		"MONITOR_SHELL="+shell, "MONITOR_CMD="+command, "MONITOR_MARK="+filepath.Join(dir, "inbox-ran"))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("/bin/sh -c '%s': %v (stdout %q, stderr %q)", monitor, err, stdout.String(), stderr.String())
		}
	case <-ctx.Done():
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		err := <-done
		t.Fatalf("the %s Monitor command still ran 30 seconds after it started, so it did not end when its parent sh was gone (wait after SIGKILL: %v, stdout %q)",
			filepath.Base(shell), err, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "args: ralph-org-a leader --quiet\n") {
		t.Errorf("Monitor command output = %q, want it to contain the inbox.sh line %q", out, "args: ralph-org-a leader --quiet")
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if last := lines[len(lines)-1]; last != leaderMonitorEndLine {
		t.Errorf("Monitor command output ends with %q, want the end line %q (output %q)", last, leaderMonitorEndLine, out)
	}
}
