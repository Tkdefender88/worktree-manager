package worktree

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// loadWorktrees returns a tea.Cmd that loads worktrees from all configured repos.
// Each repo's worktrees are fetched via the Service interface. The results are
// collected into a map keyed by repo name and sent back as a worktreesLoadedMsg.
func loadWorktrees(svc Service, repos []Repo) tea.Cmd {
	return func() tea.Msg {
		result := make(map[string][]Worktree)

		for _, repo := range repos {
			wts, err := svc.List(repo.Path)
			if err != nil {
				return errMsg{err: err}
			}
			// Tag each worktree with the repo name.
			for i := range wts {
				wts[i].Repo = repo.Name
			}
			result[repo.Name] = wts
		}

		return worktreesLoadedMsg{worktrees: result}
	}
}

// createWorktree returns a tea.Cmd that creates a new worktree.
// The worktree is created at <repo.WorktreePath>/<branch> and the result is
// sent back as a worktreeCreatedMsg or errMsg.
func createWorktree(svc Service, repo Repo, branch string, newBranch bool) tea.Cmd {
	return func() tea.Msg {
		wtPath := filepath.Join(repo.WorktreePath, branch)
		if err := svc.Create(repo.Path, wtPath, branch, newBranch); err != nil {
			return errMsg{err: err}
		}
		return worktreeCreatedMsg{
			repo:   repo,
			branch: branch,
			path:   wtPath,
		}
	}
}

// deleteWorktree returns a tea.Cmd that removes a worktree.
func deleteWorktree(svc Service, repo Repo, wt Worktree, force bool) tea.Cmd {
	return func() tea.Msg {
		if err := svc.Delete(repo.Path, wt.Path, force); err != nil {
			return errMsg{err: err}
		}
		return worktreeDeletedMsg{path: wt.Path}
	}
}

// pruneWorktreesDryRun returns a tea.Cmd that runs a dry-run prune and
// reports what would be removed.
func pruneWorktreesDryRun(svc Service, repo Repo) tea.Cmd {
	return func() tea.Msg {
		pruned, err := svc.Prune(repo.Path, true)
		if err != nil {
			return errMsg{err: err}
		}
		return pruneDryRunMsg{repo: repo, pruned: pruned}
	}
}

// pruneWorktrees returns a tea.Cmd that actually prunes stale worktrees.
func pruneWorktrees(svc Service, repo Repo) tea.Cmd {
	return func() tea.Msg {
		pruned, err := svc.Prune(repo.Path, false)
		if err != nil {
			return errMsg{err: err}
		}
		return worktreesPrunedMsg{pruned: pruned}
	}
}
