package data

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SetStatus runs `bd update <id> --status=<status>` to change an issue's status.
func SetStatus(issueID string, status Status) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, "--status="+string(status))
}

// ClaimIssue runs `bd update <id> --claim` to atomically set assignee and status to in_progress.
// Fails if the issue is already claimed by another agent, preventing races in multi-agent workflows.
func ClaimIssue(issueID string) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, "--claim")
}

// CloseIssue runs `bd close <id>` to close an issue.
func CloseIssue(issueID string) error {
	return execWithTimeout(timeoutShort, "bd", "close", issueID)
}

// SetPriority runs `bd update <id> --priority=<n>` to change priority.
func SetPriority(issueID string, priority Priority) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID, fmt.Sprintf("--priority=%d", priority))
}

// CreateIssue runs `bd create` with the given parameters and returns the new issue ID.
func CreateIssue(title string, issueType IssueType, priority Priority) (string, error) {
	args := []string{
		"create",
		"--title=" + title,
		"--type=" + string(issueType),
		fmt.Sprintf("--priority=%d", priority),
	}
	out, err := runWithTimeout(timeoutShort, "bd", args...)
	if err != nil {
		return "", wrapExitError("bd create", err)
	}
	// bd create prints the new issue ID
	return strings.TrimSpace(string(out)), nil
}

// BranchName generates a git branch name from an issue.
func BranchName(issue Issue) string {
	prefix := "feat"
	switch issue.IssueType {
	case TypeBug:
		prefix = "fix"
	case TypeChore:
		prefix = "chore"
	case TypeTask:
		prefix = "task"
	}
	slug := slugify(issue.Title)
	return fmt.Sprintf("%s/%s-%s", prefix, issue.ID, slug)
}

// slugify converts a title to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ', r == '-', r == '_', r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := b.String()
	result = strings.TrimRight(result, "-")
	if len(result) > 50 {
		result = result[:50]
		result = strings.TrimRight(result, "-")
	}
	return result
}

// WorktreePath returns the worktree path stored in an issue's metadata.
// Returns "" if not set or not a string.
func WorktreePath(issue Issue) string {
	if issue.Metadata == nil {
		return ""
	}
	v, ok := issue.Metadata["worktree"]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// CreateWorktree creates a git worktree for the given issue and stores
// the worktree path in the issue's metadata. Returns the absolute worktree path.
func CreateWorktree(issue Issue, projectDir string) (string, error) {
	// Check if worktree already tracked in metadata
	if wt := WorktreePath(issue); wt != "" {
		return "", fmt.Errorf("worktree already exists: %s", wt)
	}

	branch := BranchName(issue)
	baseDir := filepath.Join(filepath.Dir(projectDir), filepath.Base(projectDir)+"-worktrees")
	wtPath := filepath.Join(baseDir, branch)
	absPath, err := filepath.Abs(wtPath)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}

	// Partial failure recovery: dir exists but metadata wasn't set
	if info, statErr := os.Stat(absPath); statErr == nil && info.IsDir() {
		if metaErr := setWorktreeMetadata(issue.ID, absPath); metaErr != nil {
			return "", fmt.Errorf("set worktree metadata: %w", metaErr)
		}
		return absPath, nil
	}

	// Ensure parent directories exist (branch names contain slashes like feat/...)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("create worktree directory: %w", err)
	}

	// Try creating with new branch first
	ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", absPath, "-b", branch)
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		// Branch may already exist — retry without -b
		ctx2, cancel2 := context.WithTimeout(context.Background(), timeoutShort)
		defer cancel2()
		cmd2 := exec.CommandContext(ctx2, "git", "worktree", "add", absPath, branch)
		cmd2.Dir = projectDir
		if err2 := cmd2.Run(); err2 != nil {
			return "", fmt.Errorf("git worktree add: %w", err2)
		}
	}

	// Store worktree path on the bead
	if metaErr := setWorktreeMetadata(issue.ID, absPath); metaErr != nil {
		return "", fmt.Errorf("worktree created at %s but metadata update failed: %w", absPath, metaErr)
	}

	return absPath, nil
}

func setWorktreeMetadata(issueID, absPath string) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID,
		"--set-metadata", "worktree="+absPath)
}

// unsetWorktreeMetadata removes the worktree key from an issue's metadata.
func unsetWorktreeMetadata(issueID string) error {
	return execWithTimeout(timeoutShort, "bd", "update", issueID,
		"--unset-metadata", "worktree")
}

// RemoveWorktree removes the git worktree for an issue and clears its metadata.
// If the worktree directory is already gone, it skips git removal and just cleans metadata.
func RemoveWorktree(issue Issue, projectDir string) error {
	wtPath := WorktreePath(issue)
	if wtPath == "" {
		return fmt.Errorf("no worktree set for %s", issue.ID)
	}

	// If directory still exists, remove via git
	if _, err := os.Stat(wtPath); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutShort)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "worktree", "remove", wtPath, "--force")
		cmd.Dir = projectDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git worktree remove: %w", err)
		}
	}

	// Prune stale worktree entries
	_ = execWithTimeout(timeoutShort, "git", "-C", projectDir, "worktree", "prune")

	// Clear metadata on the bead
	if err := unsetWorktreeMetadata(issue.ID); err != nil {
		return fmt.Errorf("unset worktree metadata: %w", err)
	}

	return nil
}

// PruneStaleWorktrees finds issues with worktree metadata pointing to missing directories
// and clears their metadata. Runs git worktree prune once at the end. Returns the count
// of successfully pruned entries.
func PruneStaleWorktrees(issues []Issue, projectDir string) (int, error) {
	var pruned int
	var firstErr error

	for i := range issues {
		wtPath := WorktreePath(issues[i])
		if wtPath == "" {
			continue
		}
		// Only prune if directory is missing
		if _, err := os.Stat(wtPath); err == nil {
			continue // directory exists, not stale
		}
		// Directory gone — clear metadata
		if err := unsetWorktreeMetadata(issues[i].ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		pruned++
	}

	// Single prune pass at the end
	if pruned > 0 {
		_ = execWithTimeout(timeoutShort, "git", "-C", projectDir, "worktree", "prune")
	}

	return pruned, firstErr
}
