package data

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBranchName(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		expected string
	}{
		{
			name:     "Bug issue",
			issue:    Issue{ID: "bd-a1b2", Title: "Fix login token expiry", IssueType: TypeBug},
			expected: "fix/bd-a1b2-fix-login-token-expiry",
		},
		{
			name:     "Feature issue",
			issue:    Issue{ID: "bd-c3d4", Title: "Add search feature", IssueType: TypeFeature},
			expected: "feat/bd-c3d4-add-search-feature",
		},
		{
			name:     "Task issue",
			issue:    Issue{ID: "bd-e5f6", Title: "Update documentation", IssueType: TypeTask},
			expected: "task/bd-e5f6-update-documentation",
		},
		{
			name:     "Chore issue",
			issue:    Issue{ID: "bd-g7h8", Title: "Clean up CI config", IssueType: TypeChore},
			expected: "chore/bd-g7h8-clean-up-ci-config",
		},
		{
			name:     "Special characters stripped",
			issue:    Issue{ID: "bd-i9j0", Title: "Handle @mentions & #tags (v2)", IssueType: TypeFeature},
			expected: "feat/bd-i9j0-handle-mentions-tags-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchName(tt.issue)
			if got != tt.expected {
				t.Errorf("BranchName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Fix login/auth bug", "fix-login-auth-bug"},
		{"UPPER CASE", "upper-case"},
		{"   spaces   ", "spaces"},
		{"no-change", "no-change"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCreateWorktreePathComputation(t *testing.T) {
	issue := Issue{ID: "bd-a1b2", Title: "Fix login", IssueType: TypeBug}
	branch := BranchName(issue)
	projectDir := "/home/user/work/my-project"

	expectedBase := "/home/user/work/my-project-worktrees"
	expectedPath := expectedBase + "/" + branch

	got := filepath.Join(filepath.Dir(projectDir), filepath.Base(projectDir)+"-worktrees", branch)
	if got != expectedPath {
		t.Errorf("worktree path = %q, want %q", got, expectedPath)
	}
}

func TestPruneStaleWorktreesNoneStale(t *testing.T) {
	issues := []Issue{
		{ID: "bd-1", Metadata: nil},
		{ID: "bd-2", Metadata: map[string]interface{}{}},
	}
	count, err := PruneStaleWorktrees(issues, "/tmp/fake-project")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pruned, got %d", count)
	}
}

func TestRemoveWorktreeNoMetadata(t *testing.T) {
	issue := Issue{ID: "bd-test", Metadata: nil}
	err := RemoveWorktree(issue, "/tmp/fake-project")
	if err == nil {
		t.Error("expected error for issue with no worktree metadata")
	}
}

func TestDiscoverReposIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	// Make dir itself a git repo
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	repos, err := DiscoverRepos(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 || repos[0] != dir {
		t.Errorf("expected [%s], got %v", dir, repos)
	}
}

func TestDiscoverReposChildren(t *testing.T) {
	dir := t.TempDir()
	// Create child repos
	for _, name := range []string{"alpha", "beta"} {
		child := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Join(child, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Create a non-repo child
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	repos, err := DiscoverRepos(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(repos)
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d: %v", len(repos), repos)
	}
	if filepath.Base(repos[0]) != "alpha" || filepath.Base(repos[1]) != "beta" {
		t.Errorf("unexpected repos: %v", repos)
	}
}

func TestDiscoverReposNone(t *testing.T) {
	dir := t.TempDir()
	repos, err := DiscoverRepos(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %v", repos)
	}
}

func TestDiscoverReposDeepNesting(t *testing.T) {
	dir := t.TempDir()
	// Depth 2: org/repo
	// Depth 3: org/group/repo
	// Depth 4: org/group/subgroup/repo
	for _, path := range []string{
		"org1/repo-a",
		"org1/repo-b",
		"org2/services/svc-x",
		"org2/services/providers/svc-y",
	} {
		if err := os.MkdirAll(filepath.Join(dir, path, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Hidden dirs should be skipped (e.g., .gitlab-ci-local)
	if err := os.MkdirAll(filepath.Join(dir, "org1", "repo-a", ".gitlab-ci-local", "builds", ".docker", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Non-repo dir should be ignored
	if err := os.MkdirAll(filepath.Join(dir, "org1", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	repos, err := DiscoverRepos(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(repos)
	if len(repos) != 4 {
		t.Fatalf("expected 4 repos, got %d: %v", len(repos), repos)
	}
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = filepath.Base(r)
	}
	sort.Strings(names)
	expected := []string{"repo-a", "repo-b", "svc-x", "svc-y"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("repos[%d] = %q, want %q (all: %v)", i, names[i], want, names)
		}
	}
}

func TestResolveGitRepo(t *testing.T) {
	issue := Issue{
		ID:       "bd-test",
		Metadata: map[string]interface{}{"git_repo": "/tmp/nonexistent-repo"},
	}
	// Falls back to projectDir when metadata path doesn't exist
	got := ResolveGitRepo(issue, "/fallback")
	if got != "/fallback" {
		t.Errorf("expected /fallback, got %s", got)
	}

	// Uses metadata when path exists
	dir := t.TempDir()
	issue.Metadata["git_repo"] = dir
	got = ResolveGitRepo(issue, "/fallback")
	if got != dir {
		t.Errorf("expected %s, got %s", dir, got)
	}
}

func TestGitRepoPath(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		expected string
	}{
		{"nil metadata", Issue{}, ""},
		{"empty metadata", Issue{Metadata: map[string]interface{}{}}, ""},
		{"has git_repo", Issue{Metadata: map[string]interface{}{"git_repo": "/path/to/repo"}}, "/path/to/repo"},
		{"non-string git_repo", Issue{Metadata: map[string]interface{}{"git_repo": 42}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GitRepoPath(tt.issue); got != tt.expected {
				t.Errorf("GitRepoPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWorktreePath(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		expected string
	}{
		{
			name:     "metadata has worktree string",
			issue:    Issue{Metadata: map[string]interface{}{"worktree": "/tmp/worktrees/feat/bd-a1b2"}},
			expected: "/tmp/worktrees/feat/bd-a1b2",
		},
		{
			name:     "metadata nil",
			issue:    Issue{},
			expected: "",
		},
		{
			name:     "metadata empty map",
			issue:    Issue{Metadata: map[string]interface{}{}},
			expected: "",
		},
		{
			name:     "worktree key is not a string",
			issue:    Issue{Metadata: map[string]interface{}{"worktree": 42}},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreePath(tt.issue)
			if got != tt.expected {
				t.Errorf("WorktreePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}
