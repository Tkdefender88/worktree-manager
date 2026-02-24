package worktree

import tea "github.com/charmbracelet/bubbletea"

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
