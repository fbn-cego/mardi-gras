package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InTmux returns true if the current process is running inside a tmux session.
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// TmuxAvailable returns true if the tmux binary is on PATH.
func TmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// WindowName returns the tmux window name for a given issue ID.
func WindowName(issueID string) string {
	return "mg-" + issueID
}

// tmuxSocketPath extracts the socket path from $TMUX (format: "socket,pid,idx").
// In nested tmux, the default `tmux` command connects to the default server which
// is typically the outer session. Using -S with the socket from $TMUX ensures we
// target the server that owns our current pane.
func tmuxSocketPath() string {
	tmux := os.Getenv("TMUX")
	if tmux == "" {
		return ""
	}
	parts := strings.SplitN(tmux, ",", 2)
	return parts[0]
}

// tmuxCmd builds an exec.Cmd for tmux, injecting -S <socket> when running nested
// so that the command targets the correct server.
func tmuxCmd(args ...string) *exec.Cmd {
	if sock := tmuxSocketPath(); sock != "" {
		args = append([]string{"-S", sock}, args...)
	}
	return exec.Command("tmux", args...)
}

// LaunchInTmux opens a new tmux window (tab) running claude for the given issue.
func LaunchInTmux(prompt, projectDir, issueID string) (string, error) {
	winName := WindowName(issueID)
	// Build agent command based on detected runtime
	var agentArgs []string
	switch DetectRuntime() {
	case RuntimeCursor:
		agentArgs = []string{"cursor-agent", "-f", "-p", prompt}
	default: // Claude Code
		agentArgs = []string{"claude", "--teammate-mode", "tmux", prompt}
	}

	tmuxArgs := []string{"new-window",
		"-n", winName, // name the window for easy identification
		"-d",          // don't switch focus
		"-c", projectDir,
		"-P", "-F", "#{window_id}", // print the new window ID
		"--",
	}
	tmuxArgs = append(tmuxArgs, agentArgs...)
	cmd := tmuxCmd(tmuxArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux new-window: %w", err)
	}
	windowID := strings.TrimSpace(string(out))

	// Tag the window so we can discover it later via list-windows.
	_ = tmuxCmd("set-option", "-w", "-t", windowID,
		"@mg_agent", winName).Run()

	return windowID, nil
}

// ListAgentWindows returns a map of issueID -> windowID for all tmux windows
// tagged with the @mg_agent option.
func ListAgentWindows() (map[string]string, error) {
	out, err := tmuxCmd("list-windows", "-a",
		"-F", "#{@mg_agent}\t#{window_id}").Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-windows: %w", err)
	}
	return parseAgentWindows(string(out)), nil
}

// parseAgentWindows extracts agent windows from tmux list-windows output.
// Each line is "mg-<issueID>\t@<winNum>" for tagged windows, or "\t@<winNum>" for untagged.
func parseAgentWindows(output string) map[string]string {
	agents := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		tag := strings.TrimSpace(parts[0])
		windowID := strings.TrimSpace(parts[1])
		if strings.HasPrefix(tag, "mg-") && windowID != "" {
			issueID := strings.TrimPrefix(tag, "mg-")
			agents[issueID] = windowID
		}
	}
	return agents
}

// KillAgentWindow closes the tmux window for the given issue.
func KillAgentWindow(issueID string) error {
	agents, err := ListAgentWindows()
	if err != nil {
		return err
	}
	windowID, ok := agents[issueID]
	if !ok {
		return fmt.Errorf("no agent window for %s", issueID)
	}
	return tmuxCmd("kill-window", "-t", windowID).Run()
}

// SelectAgentWindow switches focus to the tmux window for the given issue.
func SelectAgentWindow(issueID string) error {
	agents, err := ListAgentWindows()
	if err != nil {
		return err
	}
	windowID, ok := agents[issueID]
	if !ok {
		return fmt.Errorf("no agent window for %s", issueID)
	}
	return tmuxCmd("select-window", "-t", windowID).Run()
}
