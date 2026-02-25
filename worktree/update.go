package worktree

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Init implements tea.Model. It kicks off loading all worktrees.
func (m Model) Init() tea.Cmd {
	return loadWorktrees(m.svc, m.repos)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When blurred, only process non-key messages (async results, etc.).
	if !m.focused {
		if _, isKey := msg.(tea.KeyMsg); isKey {
			return m, nil
		}
	}

	// Global messages handled in any state.
	switch msg := msg.(type) {
	case worktreesLoadedMsg:
		m.worktrees = msg.worktrees
		items := buildListItems(m.repos, m.worktrees)
		cmd := m.list.SetItems(items)
		m.state = stateList
		return m, cmd

	case errMsg:
		m.state = stateError
		m.err = msg.err
		return m, nil

	// Async operation results — these arrive from goroutines in any state.

	case worktreeCreatedMsg:
		m.statusMsg = "Created worktree: " + msg.wt.Branch
		// Reload all worktrees to refresh the list, then emit the event
		// with the fully populated Worktree struct.
		wt := msg.wt
		return m, tea.Batch(
			loadWorktrees(m.svc, m.repos),
			func() tea.Msg {
				return WorktreeCreatedEvent{Worktree: wt}
			},
		)

	case worktreeDeletedMsg:
		m.statusMsg = "Deleted worktree"
		return m, tea.Batch(
			loadWorktrees(m.svc, m.repos),
			func() tea.Msg { return WorktreeDeletedEvent{Path: msg.path} },
		)

	case pruneDryRunMsg:
		if len(msg.pruned) == 0 {
			m.statusMsg = "Nothing to prune"
			m.state = stateList
			return m, nil
		}
		m.pruneDryResults = msg.pruned
		m.pruneRepo = msg.repo
		m.state = statePruneDryRun
		return m, nil

	case worktreesPrunedMsg:
		m.statusMsg = "Pruned stale worktrees"
		return m, tea.Batch(
			loadWorktrees(m.svc, m.repos),
			func() tea.Msg { return WorktreesPrunedEvent{Pruned: msg.pruned} },
		)
	}

	// State-specific update handling.
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateCreatePickRepo:
		return m.updateCreatePickRepo(msg)
	case stateCreateBranch:
		return m.updateCreateBranch(msg)
	case stateCreateConfirm:
		return m.updateCreateConfirm(msg)
	case stateDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case statePrunePickRepo:
		return m.updatePrunePickRepo(msg)
	case statePruneDryRun:
		return m.updatePruneDryRun(msg)
	}

	return m, nil
}

// updateList handles input in the main list view.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys when the list is filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keyMap.Switch):
			return m.handleSwitch()
		case key.Matches(msg, m.keyMap.Create):
			return m.handleCreateStart()
		case key.Matches(msg, m.keyMap.Delete):
			return m.handleDeleteStart()
		case key.Matches(msg, m.keyMap.Prune):
			return m.handlePruneStart()
		}
	}

	// Delegate everything else to the inner list.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// --- Switch flow ---

func (m Model) handleSwitch() (tea.Model, tea.Cmd) {
	wt, ok := m.SelectedWorktree()
	if !ok {
		return m, nil
	}
	// Emit the event. The parent model (or standalone root) decides what to do.
	return m, func() tea.Msg {
		return WorktreeSwitchedEvent{Worktree: wt}
	}
}

// --- Create flow ---

func (m Model) handleCreateStart() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}
	// If there's only one repo, skip the picker.
	if len(m.repos) == 1 {
		m.createRepoIdx = 0
		m.createNewBranch = true
		m.createBranch.SetValue("")
		m.createBranch.Focus()
		m.state = stateCreateBranch
		return m, m.createBranch.Focus()
	}
	m.createRepoIdx = 0
	m.state = stateCreatePickRepo
	return m, nil
}

func (m Model) updateCreatePickRepo(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "up", "k":
			if m.createRepoIdx > 0 {
				m.createRepoIdx--
			}
			return m, nil
		case "down", "j":
			if m.createRepoIdx < len(m.repos)-1 {
				m.createRepoIdx++
			}
			return m, nil
		case "enter":
			m.createNewBranch = true
			m.createBranch.SetValue("")
			m.state = stateCreateBranch
			return m, m.createBranch.Focus()
		}
	}
	return m, nil
}

func (m Model) updateCreateBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.createBranch.Blur()
			m.state = stateList
			return m, nil
		case "enter":
			if m.createBranch.Value() == "" {
				return m, nil
			}
			m.createBranch.Blur()
			m.state = stateCreateConfirm
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.createBranch, cmd = m.createBranch.Update(msg)
	return m, cmd
}

func (m Model) updateCreateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n":
			m.state = stateList
			return m, nil
		case "tab":
			m.createNewBranch = !m.createNewBranch
			return m, nil
		case "y", "enter":
			repo := m.repos[m.createRepoIdx]
			branch := m.createBranch.Value()
			m.state = stateCreating
			m.statusMsg = "Creating worktree..."
			return m, createWorktree(m.svc, repo, branch, m.createNewBranch)
		}
	}
	return m, nil
}

// --- Delete flow ---

func (m Model) handleDeleteStart() (tea.Model, tea.Cmd) {
	wt, ok := m.SelectedWorktree()
	if !ok {
		return m, nil
	}
	// Don't allow deleting the main worktree or bare repo.
	if wt.IsMain || wt.IsBare {
		m.statusMsg = "Cannot delete the main worktree"
		return m, nil
	}
	m.deleteTarget = wt
	m.deleteForce = false
	m.state = stateDeleteConfirm
	return m, nil
}

func (m Model) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n":
			m.state = stateList
			return m, nil
		case "f":
			// Toggle force mode (useful when worktree is dirty).
			m.deleteForce = !m.deleteForce
			return m, nil
		case "y", "enter":
			repo, ok := m.repoForWorktree(m.deleteTarget)
			if !ok {
				m.state = stateList
				return m, nil
			}
			m.state = stateDeleting
			m.statusMsg = "Deleting worktree..."
			return m, deleteWorktree(m.svc, repo, m.deleteTarget, m.deleteForce)
		}
	}
	return m, nil
}

// --- Prune flow ---

func (m Model) handlePruneStart() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}
	// If there's only one repo, skip the picker and go straight to dry-run.
	if len(m.repos) == 1 {
		m.pruneRepoIdx = 0
		m.pruneRepo = m.repos[0]
		m.state = statePruning
		m.statusMsg = "Checking for stale worktrees..."
		return m, pruneWorktreesDryRun(m.svc, m.repos[0])
	}
	m.pruneRepoIdx = 0
	m.state = statePrunePickRepo
	return m, nil
}

func (m Model) updatePrunePickRepo(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "up", "k":
			if m.pruneRepoIdx > 0 {
				m.pruneRepoIdx--
			}
			return m, nil
		case "down", "j":
			if m.pruneRepoIdx < len(m.repos)-1 {
				m.pruneRepoIdx++
			}
			return m, nil
		case "enter":
			m.pruneRepo = m.repos[m.pruneRepoIdx]
			m.state = statePruning
			m.statusMsg = "Checking for stale worktrees..."
			return m, pruneWorktreesDryRun(m.svc, m.pruneRepo)
		}
	}
	return m, nil
}

func (m Model) updatePruneDryRun(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n":
			m.state = stateList
			m.statusMsg = ""
			return m, nil
		case "y", "enter":
			m.state = statePruning
			m.statusMsg = "Pruning..."
			return m, pruneWorktrees(m.svc, m.pruneRepo)
		}
	}
	return m, nil
}
