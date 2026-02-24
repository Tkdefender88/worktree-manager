package worktree

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"
)

// Init implements tea.Model. It kicks off loading all worktrees.
func (m Model) Init() tea.Cmd {
	return loadWorktrees(m.svc, m.repos)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.debug != nil {
		spew.Fdump(m.debug, msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Let ctrl+c always quit, regardless of state.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

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
	}

	// In list state, delegate all messages to the inner list model.
	if m.state == stateList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}
