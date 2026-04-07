package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// StatusInput is the JSON structure Claude Code sends on stdin.
type StatusInput struct {
	CWD     string `json:"cwd"`
	Model   *Model `json:"model"`
	Version string `json:"version"`

	Workspace     *Workspace     `json:"workspace"`
	ContextWindow *ContextWindow `json:"context_window"`
	Cost          *Cost          `json:"cost"`
	RateLimits    *RateLimits    `json:"rate_limits"`
	Vim           *Vim           `json:"vim"`
	Agent         *AgentInfo     `json:"agent"`
	Worktree      *Worktree      `json:"worktree"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir string   `json:"current_dir"`
	ProjectDir string   `json:"project_dir"`
	AddedDirs  []string `json:"added_dirs"`
}

type ContextWindow struct {
	TotalInputTokens    int      `json:"total_input_tokens"`
	TotalOutputTokens   int      `json:"total_output_tokens"`
	ContextWindowSize   int      `json:"context_window_size"`
	UsedPercentage      *float64 `json:"used_percentage"`
	RemainingPercentage *float64 `json:"remaining_percentage"`
}

type Cost struct {
	TotalCostUSD       float64 `json:"total_cost_usd"`
	TotalDurationMS    int64   `json:"total_duration_ms"`
	TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

type RateLimits struct {
	FiveHour *RateLimit `json:"five_hour"`
	SevenDay *RateLimit `json:"seven_day"`
}

type RateLimit struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

type Vim struct {
	Mode string `json:"mode"`
}

type AgentInfo struct {
	Name string `json:"name"`
}

type Worktree struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	OriginalCWD    string `json:"original_cwd"`
	OriginalBranch string `json:"original_branch"`
}

// ANSI color helpers
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dimmed    = "\033[2m"
	fgBlack   = "\033[30m"
	fgRed     = "\033[31m"
	fgGreen   = "\033[32m"
	fgYellow  = "\033[33m"
	fgBlue    = "\033[34m"
	fgMagenta = "\033[35m"
	fgCyan    = "\033[36m"
	fgWhite   = "\033[37m"
	bgRed     = "\033[41m"
	bgGreen   = "\033[42m"
	bgYellow  = "\033[43m"
	bgMagenta = "\033[45m"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to read stdin")
		os.Exit(1)
	}

	var input StatusInput
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprint(os.Stderr, "failed to parse JSON")
		os.Exit(1)
	}

	var segments []string

	// Context window battery
	if seg := contextSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	// Rate limit indicator (compact)
	if seg := rateLimitSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	// Model name
	if input.Model != nil && input.Model.DisplayName != "" {
		segments = append(segments, fgWhite+input.Model.DisplayName+reset)
	}

	// Directory
	if seg := dirSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	// Git branch + status
	if seg := gitSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	// Worktree
	if seg := worktreeSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	// Cost
	if seg := costSegment(&input); seg != "" {
		segments = append(segments, seg)
	}

	sep := fgWhite + "│" + reset
	fmt.Print(strings.Join(segments, " "+sep+" "))
}

func contextSegment(input *StatusInput) string {
	if input.ContextWindow == nil || input.ContextWindow.UsedPercentage == nil {
		return ""
	}

	pct := int(*input.ContextWindow.UsedPercentage)
	total := input.ContextWindow.ContextWindowSize
	usedTokens := pct * total / 100

	// Battery bar: 5 cells
	filled := (pct + 10) / 20
	if filled > 5 {
		filled = 5
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 5-filled)

	// Pick color based on usage
	// 132k tokens is where Opus becomes less likely to search outside its context
	color := fgGreen
	if pct >= 80 {
		color = bold + fgRed
	} else if usedTokens >= 132000 {
		color = fgYellow
	}

	// Biohazard warning when context exceeds 132k tokens
	prefix := ""
	if usedTokens >= 132000 {
		prefix = bold + fgYellow + "☣ " + reset
	}

	text := bar
	if total > 0 {
		usedK := usedTokens / 1000
		totalK := total / 1000
		text = fmt.Sprintf("%s %dk/%dk (%d%%)", bar, usedK, totalK, pct)
	} else {
		text = fmt.Sprintf("%s %d%%", bar, pct)
	}

	return prefix + color + text + reset
}

func rateLimitSegment(input *StatusInput) string {
	if input.RateLimits == nil {
		return ""
	}

	var parts []string
	if rl := input.RateLimits.FiveHour; rl != nil && rl.UsedPercentage >= 50 {
		color := fgYellow
		if rl.UsedPercentage >= 80 {
			color = bold + fgRed
		}
		parts = append(parts, fmt.Sprintf("%s5h:%.0f%%%s", color, rl.UsedPercentage, reset))
	}
	if rl := input.RateLimits.SevenDay; rl != nil && rl.UsedPercentage >= 50 {
		color := fgYellow
		if rl.UsedPercentage >= 80 {
			color = bold + fgRed
		}
		parts = append(parts, fmt.Sprintf("%s7d:%.0f%%%s", color, rl.UsedPercentage, reset))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func dirSegment(input *StatusInput) string {
	dir := ""
	if input.Workspace != nil {
		dir = input.Workspace.CurrentDir
	}
	if dir == "" {
		dir = input.CWD
	}
	if dir == "" {
		return ""
	}

	// Shorten home directory
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(dir, home) {
			dir = "~" + dir[len(home):]
		}
	}

	// Truncate to last 3 path components
	parts := strings.Split(dir, "/")
	if len(parts) > 4 {
		dir = "…/" + strings.Join(parts[len(parts)-3:], "/")
	}

	return bold + fgCyan + dir + reset
}

func gitSegment(input *StatusInput) string {
	dir := ""
	if input.Workspace != nil {
		dir = input.Workspace.CurrentDir
	}
	if dir == "" {
		dir = input.CWD
	}
	if dir == "" {
		return ""
	}

	branch := gitBranch(dir)
	if branch == "" {
		return ""
	}

	seg := bold + fgMagenta + " " + branch + reset

	if dirty := gitDirty(dir); dirty != "" {
		seg += " " + bold + fgRed + dirty + reset
	}

	return seg
}

func gitBranch(dir string) string {
	// Fast path: read .git/HEAD directly
	gitDir := findGitDir(dir)
	if gitDir == "" {
		return ""
	}

	headBytes, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}

	head := strings.TrimSpace(string(headBytes))
	if strings.HasPrefix(head, "ref: refs/heads/") {
		return head[len("ref: refs/heads/"):]
	}
	// Detached HEAD — show short hash
	if len(head) >= 7 {
		return head[:7]
	}
	return head
}

func findGitDir(dir string) string {
	for {
		candidate := filepath.Join(dir, ".git")
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return candidate
			}
			// .git file (worktree) — read the gitdir pointer
			data, err := os.ReadFile(candidate)
			if err == nil {
				line := strings.TrimSpace(string(data))
				if strings.HasPrefix(line, "gitdir: ") {
					gitdir := line[len("gitdir: "):]
					if !filepath.IsAbs(gitdir) {
						gitdir = filepath.Join(dir, gitdir)
					}
					return gitdir
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func gitDirty(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = dir
	// Create a new process group so we can kill the entire child tree on timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the process group, not just the process.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return "✎"
	}
	return ""
}

func worktreeSegment(input *StatusInput) string {
	if input.Worktree == nil || input.Worktree.Name == "" {
		return ""
	}
	return bold + fgYellow + "⌥ " + input.Worktree.Name + reset
}

func costSegment(input *StatusInput) string {
	if input.Cost == nil || input.Cost.TotalCostUSD < 0.01 {
		return ""
	}
	return fgWhite + fmt.Sprintf("$%.2f", input.Cost.TotalCostUSD) + reset
}
