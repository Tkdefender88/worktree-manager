// Package git implements the worktree.Service interface by shelling out to git.
package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Tkdefender88/worktree-manager/worktree"
)

// WorktreeService implements worktree.Service using real git commands.
type WorktreeService struct{}

// NewWorktreeService returns a new WorktreeService.
func NewWorktreeService() *WorktreeService {
	return &WorktreeService{}
}

// List returns all worktrees for the repository at repoPath by parsing
// the output of `git worktree list --porcelain`.
//
// Porcelain format (one block per worktree, separated by blank lines):
//
//	worktree /path/to/worktree
//	HEAD abc123...
//	branch refs/heads/main
//
// Bare repos show "bare" instead of branch. Detached HEAD shows "detached".
func (s *WorktreeService) List(repoPath string) ([]worktree.Worktree, error) {
	out, err := runGit(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	worktrees, err := parsePorcelain(out)
	if err != nil {
		return nil, fmt.Errorf("parsing worktree list: %w", err)
	}

	// Check dirty status for each non-bare worktree.
	for i := range worktrees {
		if worktrees[i].IsBare {
			continue
		}
		dirty, err := isDirty(worktrees[i].Path)
		if err != nil {
			// Non-fatal: mark as unknown rather than failing the whole list.
			continue
		}
		worktrees[i].IsDirty = dirty
	}

	return worktrees, nil
}

// Create creates a new worktree at worktreePath. If newBranch is true, a new
// branch is created with `-b`. Otherwise the existing branch is checked out.
func (s *WorktreeService) Create(repoPath string, worktreePath string, branch string, newBranch bool) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, worktreePath)
	} else {
		args = append(args, worktreePath, branch)
	}

	if _, err := runGit(repoPath, args...); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	return nil
}

// Delete removes the worktree at worktreePath. If force is true, the worktree
// is removed even with uncommitted changes.
func (s *WorktreeService) Delete(repoPath string, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	if _, err := runGit(repoPath, args...); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}
	return nil
}

// Prune cleans up stale worktree references. If dryRun is true, it reports
// what would be pruned without removing anything.
//
// Note: git worktree prune --verbose writes its progress to stderr, not stdout.
func (s *WorktreeService) Prune(repoPath string, dryRun bool) ([]string, error) {
	args := []string{"worktree", "prune", "--verbose"}
	if dryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	// git worktree prune --verbose writes output to stderr.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pruning worktrees: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var pruned []string
	scanner := bufio.NewScanner(&stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			pruned = append(pruned, line)
		}
	}
	return pruned, scanner.Err()
}

// parsePorcelain parses the porcelain output of `git worktree list --porcelain`.
func parsePorcelain(data []byte) ([]worktree.Worktree, error) {
	var worktrees []worktree.Worktree
	var current *worktree.Worktree

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// Blank line separates worktree entries.
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		key, value, _ := strings.Cut(line, " ")

		switch key {
		case "worktree":
			current = &worktree.Worktree{
				Path: value,
			}
			// The first worktree listed is always the main worktree.
			if len(worktrees) == 0 {
				current.IsMain = true
			}
		case "HEAD":
			if current != nil {
				current.HEAD = value
			}
		case "branch":
			if current != nil {
				// Strip refs/heads/ prefix to get the short branch name.
				current.Branch = strings.TrimPrefix(value, "refs/heads/")
			}
		case "bare":
			if current != nil {
				current.IsBare = true
			}
		case "detached":
			if current != nil {
				current.IsDetached = true
			}
		}
	}

	// Handle last entry if file doesn't end with a blank line.
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, scanner.Err()
}

// isDirty checks if a worktree has uncommitted changes using `git status --porcelain`.
// Returns true if there are any modified, added, or untracked files.
func isDirty(worktreePath string) (bool, error) {
	out, err := runGit(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// runGit executes a git command in the given directory and returns stdout.
// If the command fails, the error includes stderr.
func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
