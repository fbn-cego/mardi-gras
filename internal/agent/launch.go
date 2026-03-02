package agent

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/matt-wright86/mardi-gras/internal/data"
)

// Available returns true if the claude CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// BuildPrompt composes the initial prompt for a Claude Code session
// given a selected issue and its evaluated dependencies.
func BuildPrompt(issue data.Issue, deps data.DepEval, issueMap map[string]*data.Issue) string {
	var b strings.Builder

	b.WriteString("Work on this Beads issue:\n\n")
	fmt.Fprintf(&b, "## %s: %s\n\n", issue.ID, issue.Title)

	fmt.Fprintf(&b, "Status: %s | Type: %s | Priority: %s\n",
		issue.Status, issue.IssueType, data.PriorityLabel(issue.Priority))
	if issue.Owner != "" {
		fmt.Fprintf(&b, "Owner: %s\n", issue.Owner)
	}
	if issue.Assignee != "" {
		fmt.Fprintf(&b, "Assignee: %s\n", issue.Assignee)
	}

	if issue.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", issue.Description)
	}

	if issue.Notes != "" {
		fmt.Fprintf(&b, "\n### Notes\n%s\n", issue.Notes)
	}

	if issue.AcceptanceCriteria != "" {
		fmt.Fprintf(&b, "\n### Acceptance Criteria\n%s\n", issue.AcceptanceCriteria)
	}

	if len(deps.Edges) > 0 {
		b.WriteString("\n### Dependencies\n")
		for _, edge := range deps.Edges {
			switch edge.Status {
			case data.DepBlocking:
				if dep, ok := issueMap[edge.DependsOnID]; ok {
					fmt.Fprintf(&b, "- Blocked by: %s (%s) -- %s\n",
						edge.DependsOnID, dep.Title, dep.Status)
				}
			case data.DepMissing:
				fmt.Fprintf(&b, "- Missing: %s (not found)\n", edge.DependsOnID)
			case data.DepResolved:
				if dep, ok := issueMap[edge.DependsOnID]; ok {
					fmt.Fprintf(&b, "- Resolved: %s (%s) -- closed\n",
						edge.DependsOnID, dep.Title)
				}
			case data.DepNonBlocking:
				if dep, ok := issueMap[edge.DependsOnID]; ok {
					fmt.Fprintf(&b, "- Related: %s (%s) -- %s\n",
						edge.DependsOnID, dep.Title, edge.Type)
				}
			}
		}
	}

	fmt.Fprintf(&b, "\n---\nWhen you begin work, run: bd update %s --status=in_progress\n", issue.ID)
	fmt.Fprintf(&b, "When finished, run: bd close %s\n", issue.ID)
	b.WriteString("\nIf this task is complex enough to benefit from parallel work, consider using agent teams to spawn teammates for independent subtasks.")

	return b.String()
}

// Command returns an *exec.Cmd that launches claude interactively
// with the given prompt, working directory set to projectDir.
func Command(prompt, projectDir string) *exec.Cmd {
	c := exec.Command("claude", prompt)
	c.Dir = projectDir
	return c
}
